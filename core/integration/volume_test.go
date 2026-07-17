package core

// These tests cover opaque filesystem volumes backed by SSHFS. Unit tests cover
// address parsing and schema hiding; this suite exercises the live engine path.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/dagger/dagger/internal/buildkit/identity"
	"github.com/dagger/dagger/internal/testutil"
	"github.com/dagger/testctx"
	"github.com/stretchr/testify/require"
)

type VolumeSuite struct{}

func TestVolume(t *testing.T) {
	testctx.New(t, Middleware()...).RunTests(VolumeSuite{})
}

func (VolumeSuite) TestSSHFSVolumeMount(ctx context.Context, t *testctx.T) {
	c := connect(ctx, t)
	fixture := newSSHFSVolumeFixture(ctx, t, c, "hello from sshfs\n")

	out, err := c.Container().
		From(alpineImage).
		WithMountedVolume("/mnt", fixture.Volume(c), dagger.ContainerWithMountedVolumeOpts{ReadOnly: true}).
		WithExec([]string{"cat", "/mnt/hello.txt"}).
		Stdout(ctx)
	require.NoError(t, err)
	require.Equal(t, fixture.contents, out)
}

// sshfsChaosKiller drops the client's sshd connection exactly once, when the
// client creates /data/KILL, then clears the marker. It only kills
// per-connection children (classic "sshd: root@notty" or the newer split
// "sshd-session"), never the -D listener, so recovery isn't confounded by the
// whole daemon bouncing.
const sshfsChaosKiller = `(
	while true; do
		if [ -e /data/KILL ]; then
			rm -f /data/KILL
			for p in /proc/[0-9]*; do
				cmd=$(tr '\0' ' ' < "$p/cmdline" 2>/dev/null) || continue
				case "$cmd" in *-D*) continue ;; esac
				case "$cmd" in
					*"sshd: "*|*"sshd-session"*) kill "${p##*/}" 2>/dev/null || true ;;
				esac
			done
		fi
		sleep 1
	done
) &`

func (VolumeSuite) TestSSHFSVolumeReconnectsAfterDrop(ctx context.Context, t *testctx.T) {
	c := connect(ctx, t)
	fixture := newSSHFSVolumeFixtureWithChaos(ctx, t, c, "reconnect payload\n", sshfsChaosKiller)

	// Read once to establish the connection, wait past a server-side kill, then
	// retry reading. sshfs -o reconnect does NOT replay the in-flight op, so the
	// read that discovers the dropped connection may itself EIO; a subsequent
	// read must succeed once the mount recovers. Each read is bounded so a true
	// hang is caught, not only EIO. Emit PASS/FAIL rather than exiting non-zero
	// so the diagnostic below still runs.
	out, err := c.Container().
		From(alpineImage).
		WithMountedVolume("/mnt", fixture.Volume(c)).
		WithExec([]string{"sh", "-c", `
result=FAIL
timeout 5 cat /mnt/hello.txt >/dev/null 2>&1 || true
# Trigger exactly one server-side disconnect, then let the mount recover.
: > /mnt/KILL || true
sleep 5
i=0
while [ "$i" -lt 20 ]; do
	if timeout 5 cat /mnt/hello.txt >/dev/null 2>&1; then result=PASS; break; fi
	i=$((i + 1))
	sleep 2
done
echo "$result"
`}).
		Stdout(ctx)
	require.NoError(t, err)

	// Read the sshd server log through a fresh volume (a new sshfs process, so it
	// works even if the first mount stayed wedged) to see reconnection attempts.
	sshdLog, logErr := c.Container().
		From(alpineImage).
		WithEnvVariable("CACHEBUST", identity.NewID()).
		WithMountedVolume("/mnt", fixture.Volume(c), dagger.ContainerWithMountedVolumeOpts{ReadOnly: true}).
		WithExec([]string{"sh", "-c", "tail -n 60 /mnt/sshd.log 2>&1 || true"}).
		Stdout(ctx)
	if logErr != nil {
		t.Logf("could not read sshd log: %v", logErr)
	} else {
		t.Logf("sshd server log (tail):\n%s", sshdLog)
	}

	require.Equal(t, "PASS", strings.TrimSpace(out), "mount did not recover after reconnect")
}

func (VolumeSuite) TestSSHFSVolumeRejectsWrongKnownHosts(ctx context.Context, t *testctx.T) {
	c := connect(ctx, t)
	fixture := newSSHFSVolumeFixture(ctx, t, c, "hello from sshfs\n")

	// Control the fixture first, so the failure below is attributable to
	// host-key verification instead of a broken SSH service.
	out, err := c.Container().
		From(alpineImage).
		WithMountedVolume("/mnt", fixture.Volume(c), dagger.ContainerWithMountedVolumeOpts{ReadOnly: true}).
		WithExec([]string{"cat", "/mnt/hello.txt"}).
		Stdout(ctx)
	require.NoError(t, err)
	require.Equal(t, fixture.contents, out)

	_, err = c.Container().
		From(alpineImage).
		WithMountedVolume("/mnt", fixture.VolumeWithKnownHosts(c, c.SetSecret(
			"sshfs-test-wrong-known-hosts-"+identity.NewID(),
			fixture.wrongKnownHosts,
		))).
		WithExec([]string{"cat", "/mnt/hello.txt"}).
		Stdout(ctx)
	requireErrOut(t, err, "failed to mount /mnt")
}

func (VolumeSuite) TestSSHFSVolumeCachedExecDoesNotRereadRemoteContents(ctx context.Context, t *testctx.T) {
	c := connect(ctx, t)
	fixture := newSSHFSVolumeFixture(ctx, t, c, "v1\n")
	vol := fixture.Volume(c)

	// The read recipe below is intentionally identical before and after the
	// remote write. It should return the cached result, while a cache-busted read
	// proves the remote contents did change.
	readCached := func() string {
		out, err := c.Container().
			From(alpineImage).
			WithMountedVolume("/mnt", vol, dagger.ContainerWithMountedVolumeOpts{ReadOnly: true}).
			WithExec([]string{"cat", "/mnt/hello.txt"}).
			Stdout(ctx)
		require.NoError(t, err)
		return out
	}

	require.Equal(t, "v1\n", readCached())

	_, err := c.Container().
		From(alpineImage).
		WithMountedVolume("/mnt", vol).
		WithExec([]string{"sh", "-c", "printf 'v2\n' > /mnt/hello.txt"}).
		Sync(ctx)
	require.NoError(t, err)

	freshRead, err := c.Container().
		From(alpineImage).
		WithEnvVariable("CACHEBUSTER", identity.NewID()).
		WithMountedVolume("/mnt", vol, dagger.ContainerWithMountedVolumeOpts{ReadOnly: true}).
		WithExec([]string{"cat", "/mnt/hello.txt"}).
		Stdout(ctx)
	require.NoError(t, err)
	require.Equal(t, "v2\n", freshRead)

	require.Equal(t, "v1\n", readCached())
}

func (VolumeSuite) TestModuleAcceptsAndMountsVolume(ctx context.Context, t *testctx.T) {
	c := connect(ctx, t)
	fixture := newSSHFSVolumeFixture(ctx, t, c, "hello from a module\n")
	volID, err := fixture.Volume(c).ID(ctx)
	require.NoError(t, err)

	modDir := t.TempDir()
	copyTestdataFixture(ctx, t, modDir, "modules", "go", "call-volume")
	err = c.ModuleSource(modDir).AsModule().Serve(ctx)
	require.NoError(t, err)

	res, err := testutil.QueryWithClient[struct {
		Test struct {
			Read string
		}
	}](c, t, `query($volume: ID!) { test { read(volume: $volume) } }`, &testutil.QueryOptions{
		Variables: map[string]any{
			"volume": volID,
		},
	})
	require.NoError(t, err)
	require.Equal(t, fixture.contents, res.Test.Read)
}

func (VolumeSuite) TestModuleCannotConstructSSHFSVolume(ctx context.Context, t *testctx.T) {
	c := connect(ctx, t)

	_, err := moduleFixture(t, c, "go/sshfs-volume-constructor-denied").
		With(daggerCallAt(".", "fn")).
		Sync(ctx)
	requireErrOut(t, err, "SshfsVolume")
	requireErrOut(t, err, "undefined")
}

type sshfsVolumeFixture struct {
	endpoint   string
	privateKey *dagger.Secret
	knownHosts *dagger.Secret
	service    *dagger.Service
	contents   string

	wrongKnownHosts string
}

func (f sshfsVolumeFixture) Volume(c *dagger.Client) *dagger.Volume {
	return f.VolumeWithKnownHosts(c, f.knownHosts)
}

func (f sshfsVolumeFixture) VolumeWithKnownHosts(c *dagger.Client, knownHosts *dagger.Secret) *dagger.Volume {
	return c.SshfsVolume(f.endpoint, f.privateKey, dagger.SshfsVolumeOpts{
		KnownHosts:              knownHosts,
		ExperimentalServiceHost: f.service,
	})
}

func newSSHFSVolumeFixture(ctx context.Context, t *testctx.T, c *dagger.Client, contents string) sshfsVolumeFixture {
	return newSSHFSVolumeFixtureWithChaos(ctx, t, c, contents, "")
}

// newSSHFSVolumeFixtureWithChaos builds the sshd fixture, optionally injecting a
// shell snippet (chaos) into the service startup script just before sshd is
// exec'd. Existing callers pass "" and get the unmodified service.
func newSSHFSVolumeFixtureWithChaos(ctx context.Context, t *testctx.T, c *dagger.Client, contents, chaos string) sshfsVolumeFixture {
	t.Helper()

	const (
		sshPort     = 2222
		logicalHost = "example.com"
	)

	sshBase := c.Container().
		From(alpineImage).
		WithExec([]string{"apk", "add", "openssh"})

	keygen := sshBase.WithExec([]string{"sh", "-ec", `
mkdir -p /root/.ssh
ssh-keygen -t ed25519 -f /root/.ssh/host_key -N ""
ssh-keygen -t ed25519 -f /root/.ssh/id_ed25519 -N ""
`})

	hostPubKey, err := keygen.File("/root/.ssh/host_key.pub").Contents(ctx)
	require.NoError(t, err)

	userPrivateKey, err := keygen.File("/root/.ssh/id_ed25519").Contents(ctx)
	require.NoError(t, err)

	userPubKey, err := keygen.File("/root/.ssh/id_ed25519.pub").Contents(ctx)
	require.NoError(t, err)

	startScript := `#!/bin/sh
set -eu

mkdir -p /run/sshd /root/.ssh
cp /root/.ssh/id_ed25519.pub /root/.ssh/authorized_keys
chmod 700 /root/.ssh
chmod 600 /root/.ssh/authorized_keys
chmod 600 /root/.ssh/host_key
if [ ! -e /data/hello.txt ]; then
	cp /seed/hello.txt /data/hello.txt
fi
chmod 644 /data/hello.txt

cat > /etc/ssh/sshd_config <<'EOF'
Port 2222
ListenAddress 0.0.0.0
HostKey /root/.ssh/host_key
PasswordAuthentication no
PubkeyAuthentication yes
PermitRootLogin prohibit-password
AuthorizedKeysFile .ssh/authorized_keys
Subsystem sftp internal-sftp
LogLevel VERBOSE
EOF

` + chaos + `

exec "$(which sshd)" -D -E /data/sshd.log -f /etc/ssh/sshd_config
`
	setupScript := c.Directory().
		WithNewFile("start.sh", startScript, dagger.DirectoryWithNewFileOpts{Permissions: 0o755}).
		File("start.sh")

	service := keygen.
		WithNewFile("/seed/hello.txt", contents).
		// Keep remote writes across service restarts; use a unique key so
		// parallel tests and prior runs cannot inherit each other's contents.
		WithMountedCache("/data", c.CacheVolume("sshfs-test-data-"+identity.NewID())).
		WithMountedFile("/root/start.sh", setupScript).
		WithExposedPort(sshPort).
		WithDefaultArgs([]string{"sh", "/root/start.sh"}).
		AsService()

	return sshfsVolumeFixture{
		endpoint:   fmt.Sprintf("sshfs://root@%s:%d/data", logicalHost, sshPort),
		privateKey: c.SetSecret("sshfs-test-private-key-"+identity.NewID(), userPrivateKey),
		knownHosts: c.SetSecret(
			"sshfs-test-known-hosts-"+identity.NewID(),
			fmt.Sprintf("[%s]:%d %s", logicalHost, sshPort, strings.TrimSpace(hostPubKey)),
		),
		service:  service,
		contents: contents,
		wrongKnownHosts: fmt.Sprintf(
			"[%s]:%d %s",
			logicalHost,
			sshPort,
			strings.TrimSpace(userPubKey),
		),
	}
}
