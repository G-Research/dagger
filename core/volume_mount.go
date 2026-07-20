package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ctrdmount "github.com/containerd/containerd/v2/core/mount"
	"github.com/dagger/dagger/engine/slog"
	bkcache "github.com/dagger/dagger/engine/snapshots"
	"github.com/dagger/dagger/internal/buildkit/identity"
	"github.com/moby/sys/userns"
	"golang.org/x/sys/unix"

	"github.com/dagger/dagger/dagql"
)

func prepareExecVolumeMount(cfg *execVolumeMountConfig) (bkcache.Mountable, error) {
	if cfg == nil {
		return nil, fmt.Errorf("invalid volume mount options")
	}
	if cfg.Volume.Self() == nil {
		return nil, fmt.Errorf("volume mount missing volume")
	}
	vol := cfg.Volume.Self()
	if vol.Backend != VolumeBackendKindSSHFS || vol.SSHFS == nil {
		return nil, fmt.Errorf("unsupported volume backend %q", vol.Backend)
	}
	return &execVolumeMount{volume: cfg.Volume}, nil
}

type execVolumeMount struct {
	volume dagql.ObjectResult[*Volume]
}

func (mnt *execVolumeMount) Mount(ctx context.Context, readonly bool) (bkcache.MountableRef, error) {
	mounts, release, err := mountSSHFSVolume(ctx, readonly, mnt.volume.Self().SSHFS)
	if err != nil {
		return nil, err
	}
	return &execVolumeMountInstance{
		mounts:  mounts,
		release: release,
	}, nil
}

type execVolumeMountInstance struct {
	mounts  []ctrdmount.Mount
	release func() error
}

func (mnt *execVolumeMountInstance) Mount() ([]ctrdmount.Mount, func() error, error) {
	return mnt.mounts, mnt.release, nil
}

func mountSSHFSVolume(ctx context.Context, readonly bool, cfg *SSHFSVolumeConfig) (_ []ctrdmount.Mount, _ func() error, rerr error) {
	privateKey, err := plaintextSecret(ctx, cfg.PrivateKey, "volume private key")
	if err != nil {
		return nil, nil, err
	}

	var knownHosts []byte
	if cfg.KnownHosts.Self() != nil {
		knownHosts, err = plaintextSecret(ctx, cfg.KnownHosts, "volume known hosts")
		if err != nil {
			return nil, nil, err
		}
		if len(knownHosts) == 0 && !cfg.InsecureSkipHostKeyCheck {
			return nil, nil, fmt.Errorf("volume known hosts empty")
		}
	} else if !cfg.InsecureSkipHostKeyCheck {
		return nil, nil, fmt.Errorf("volume known hosts missing")
	}

	source, port, releaseService, err := sshfsMountSource(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	release := releaseService
	defer func() {
		if rerr != nil && release != nil {
			_ = release()
		}
	}()

	workDir, err := os.MkdirTemp("", "dagger-sshfs-")
	if err != nil {
		return nil, nil, fmt.Errorf("create sshfs workdir: %w", err)
	}
	release = joinCleanup(func() error {
		return os.RemoveAll(workDir)
	}, release)

	tmpMount := ctrdmount.Mount{
		Type:    "tmpfs",
		Source:  "tmpfs",
		Options: []string{"nodev", "nosuid", "mode=0700", fmt.Sprintf("uid=%d,gid=%d", os.Geteuid(), os.Getegid())},
	}
	if userns.RunningInUserNS() {
		tmpMount.Options = nil
	}
	if err = ctrdmount.All([]ctrdmount.Mount{tmpMount}, workDir); err != nil {
		return nil, nil, fmt.Errorf("mount sshfs workdir tmpfs: %w", err)
	}
	// Cleanup is ordered outside-in: unmount the sshfs mount, unmount the tmpfs
	// holding key material, remove the temp dir, then release any backing service.
	release = joinCleanup(unmountWithDetachFallback(workDir), release)

	mountDir := filepath.Join(workDir, "mnt")
	if err = os.Mkdir(mountDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create sshfs mountpoint: %w", err)
	}
	keyPath := filepath.Join(workDir, identity.NewID())
	if err = os.WriteFile(keyPath, privateKey, 0o600); err != nil {
		return nil, nil, fmt.Errorf("write sshfs private key: %w", err)
	}

	var knownHostsPath string
	if len(knownHosts) > 0 {
		knownHostsPath = filepath.Join(workDir, identity.NewID())
		if err = os.WriteFile(knownHostsPath, knownHosts, 0o600); err != nil {
			return nil, nil, fmt.Errorf("write sshfs known_hosts: %w", err)
		}
	}

	debug := sshfsDebugEnabled()
	args := sshfsCommandArgs(source, mountDir, sshfsCommandConfig{
		Port:                     port,
		PrivateKeyPath:           keyPath,
		KnownHostsPath:           knownHostsPath,
		HostKeyAlias:             cfg.HostKeyAlias,
		InsecureSkipHostKeyCheck: cfg.InsecureSkipHostKeyCheck,
		AllowOther:               sshfsAllowOther(),
		Readonly:                 readonly,
		ConnectTimeout:           cfg.ConnectTimeout,
		ServerAliveInterval:      cfg.ServerAliveInterval,
		ServerAliveCountMax:      cfg.ServerAliveCountMax,
		Debug:                    debug,
		Reconnect:                cfg.Reconnect,
	})
	cmd := osexec.CommandContext(ctx, "sshfs", args...)

	if debug {
		// In debug mode sshfs runs foreground and logs the entire session
		// (including any mid-transfer reconnects) to a persistent file, and we
		// poll for mount readiness rather than waiting for the process to exit.
		logPath := filepath.Join(os.TempDir(), "dagger-sshfs-debug-"+identity.NewID()+".log")
		logFile, ferr := os.Create(logPath)
		if ferr != nil {
			return nil, nil, fmt.Errorf("create sshfs debug log: %w", ferr)
		}
		release = joinCleanup(logFile.Close, release)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		slog.SpanLogger(ctx, InstrumentationLibrary).Info("sshfs debug logging enabled",
			"log_path", logPath, "mount", mountDir)

		stop, serr := startSSHFSForeground(ctx, cmd, mountDir)
		if serr != nil {
			return nil, nil, fmt.Errorf("mount sshfs volume: %w (see debug log %s)", serr, logPath)
		}
		release = joinCleanup(stop, joinCleanup(unmountWithDetachFallback(mountDir), release))
	} else {
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stop, serr := startSSHFSForeground(ctx, cmd, mountDir)
		if serr != nil {
			_ = unmountWithDetachFallback(mountDir)()
			return nil, nil, fmt.Errorf("mount sshfs volume: %w%s", serr, formatCommandStderr(stderr.Bytes()))
		}
		// The foreground process is the FUSE daemon; release tears it down
		// (killing the process group so a blocked in-flight I/O returns EIO
		// rather than wedging forever) before unmounting.
		release = joinCleanup(stop, joinCleanup(unmountWithDetachFallback(mountDir), release))
	}
	bindOptions := []string{"rbind"}
	if readonly {
		bindOptions = append(bindOptions, "ro")
	}
	return []ctrdmount.Mount{{
		Type:    "bind",
		Source:  mountDir,
		Options: bindOptions,
	}}, release, nil
}

type sshfsCommandConfig struct {
	Port                     string
	PrivateKeyPath           string
	KnownHostsPath           string
	HostKeyAlias             string
	InsecureSkipHostKeyCheck bool
	AllowOther               bool
	Readonly                 bool
	ConnectTimeout           int
	ServerAliveInterval      int
	ServerAliveCountMax      int
	Debug                    bool
	// Reconnect retries the SSHFS transport in the background if the link
	// drops. Off by default so a dropped link fails the in-flight I/O (EIO)
	// quickly instead of retrying forever.
	Reconnect bool
}

func sshfsAllowOther() bool {
	if os.Geteuid() == 0 && !userns.RunningInUserNS() {
		return true
	}
	fuseConf, err := os.ReadFile("/etc/fuse.conf")
	if err != nil {
		return false
	}
	return fuseConfAllowsOther(fuseConf)
}

func fuseConfAllowsOther(fuseConf []byte) bool {
	for _, line := range strings.Split(string(fuseConf), "\n") {
		line, _, _ = strings.Cut(line, "#")
		if strings.TrimSpace(line) == "user_allow_other" {
			return true
		}
	}
	return false
}

func sshfsCommandArgs(source, mountDir string, cfg sshfsCommandConfig) []string {
	args := []string{source, mountDir}
	if cfg.Port != "" {
		args = append(args, "-p", cfg.Port)
	}
	args = append(args,
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "IdentityFile="+cfg.PrivateKeyPath,
	)
	if cfg.AllowOther {
		args = append(args, "-o", "allow_other")
	}
	if cfg.InsecureSkipHostKeyCheck {
		args = append(args,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
		)
	} else {
		args = append(args,
			"-o", "StrictHostKeyChecking=yes",
			"-o", "UserKnownHostsFile="+cfg.KnownHostsPath,
		)
		if cfg.HostKeyAlias != "" {
			args = append(args, "-o", "HostKeyAlias="+cfg.HostKeyAlias)
		}
	}
	// Automatically re-establish the SSH connection if it drops mid-session so a
	// stalled/dropped link recovers instead of wedging the FUSE mount. Off by
	// default: a dropped link then surfaces EIO to the caller and fails the
	// exec quickly instead of retrying forever (which is what wedges large
	// transfers). Opt in via the reconnect volume option.
	if cfg.Reconnect {
		args = append(args, "-o", "reconnect")
	}
	if cfg.ServerAliveInterval > 0 {
		args = append(args, "-o", fmt.Sprintf("ServerAliveInterval=%d", cfg.ServerAliveInterval))
	}
	if cfg.ServerAliveCountMax > 0 {
		args = append(args, "-o", fmt.Sprintf("ServerAliveCountMax=%d", cfg.ServerAliveCountMax))
	}
	if cfg.ConnectTimeout > 0 {
		args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", cfg.ConnectTimeout))
	}
	if cfg.Readonly {
		args = append(args, "-o", "ro")
	}
	// Run foreground (-f) so the FUSE daemon is the managed process (not a
	// detached child). This keeps it in a process group we control, so a
	// stalled/dropped transport can be killed (returning EIO to the in-flight
	// FUSE op) instead of wedging the exec forever. The caller polls for mount
	// readiness instead of waiting for the process to daemonize.
	args = append(args, "-f")
	if cfg.Debug {
		// Keep the daemon's stderr open and log the whole session for debugging.
		args = append(args, "-o", "sshfs_debug", "-o", "loglevel=DEBUG3")
	}
	return args
}

func sshfsDebugEnabled() bool {
	v := os.Getenv("DAGGER_SSHFS_DEBUG")
	return v == "1" || strings.EqualFold(v, "true")
}

// sshfsIsMounted reports whether mountDir is a mount point in the current mount
// namespace. Used by the debug path, where sshfs runs foreground and never
// daemonizes, so process exit cannot signal mount readiness.
func sshfsIsMounted(mountDir string) bool {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(" "+mountDir+" "))
}

// startSSHFSForeground starts a foreground (-f) sshfs process, waits for the
// mount to become ready by polling, and returns a stop function that tears the
// process (and thus the mount) down. Unlike the daemonizing path, the process
// stays alive for the whole mount lifetime.
func startSSHFSForeground(ctx context.Context, cmd *osexec.Cmd, mountDir string) (func() error, error) {
	cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true, Pdeathsig: unix.SIGTERM}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start sshfs: %w", err)
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	kill := func(sig unix.Signal) { _ = unix.Kill(-cmd.Process.Pid, sig) }
	timeout := time.After(2 * time.Minute)
	for {
		if sshfsIsMounted(mountDir) {
			break
		}
		select {
		case werr := <-waitCh:
			return nil, fmt.Errorf("sshfs exited before mount ready: %w", werr)
		case <-ctx.Done():
			kill(unix.SIGKILL)
			<-waitCh
			return nil, ctx.Err()
		case <-timeout:
			kill(unix.SIGKILL)
			<-waitCh
			return nil, fmt.Errorf("sshfs mount readiness timed out")
		case <-time.After(150 * time.Millisecond):
		}
	}

	return func() error {
		kill(unix.SIGTERM)
		select {
		case <-waitCh:
		case <-time.After(10 * time.Second):
			kill(unix.SIGKILL)
			<-waitCh
		}
		return nil
	}, nil
}

func sshfsMountSource(ctx context.Context, cfg *SSHFSVolumeConfig) (string, string, func() error, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse sshfs endpoint: %w", err)
	}
	host := u.Hostname()
	port := u.Port()
	var release func() error
	if cfg.ServiceHost.Self() != nil {
		running, releaseService, err := startSSHFSServiceHost(ctx, cfg.ServiceHost)
		if err != nil {
			return "", "", nil, err
		}
		release = releaseService
		// Service-backed SSHFS connects to the running service address, while
		// HostKeyAlias continues to verify the logical endpoint host.
		host = running.Host
		port, err = sshfsServiceHostPort(running, port)
		if err != nil {
			if release != nil {
				_ = release()
			}
			return "", "", nil, err
		}
	}
	source, err := sshfsSourceForURL(u, host)
	if err != nil {
		if release != nil {
			_ = release()
		}
		return "", "", nil, err
	}
	return source, port, release, nil
}

func startSSHFSServiceHost(ctx context.Context, service dagql.ObjectResult[*Service]) (*RunningService, func() error, error) {
	query, err := CurrentQuery(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get current query for sshfs service host: %w", err)
	}
	svcs, err := query.Services(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get services for sshfs service host: %w", err)
	}
	running, release, err := svcs.StartResultWithDependencyExitPropagationSuppressed(ctx, service)
	if err != nil {
		return nil, nil, fmt.Errorf("start sshfs service host: %w", err)
	}
	return running, func() error {
		release()
		return nil
	}, nil
}

func sshfsServiceHostPort(running *RunningService, endpointPort string) (string, error) {
	if len(running.Ports) == 0 {
		return "", fmt.Errorf("sshfs service host exposes no ports")
	}
	if endpointPort != "" {
		port, err := strconv.Atoi(endpointPort)
		if err != nil {
			return "", fmt.Errorf("parse sshfs endpoint port %q: %w", endpointPort, err)
		}
		for _, exposed := range running.Ports {
			if exposed.Port == port {
				if exposed.Protocol != "" && exposed.Protocol != NetworkProtocolTCP {
					return "", fmt.Errorf("sshfs service host endpoint port %s is not TCP", endpointPort)
				}
				return endpointPort, nil
			}
		}
		return "", fmt.Errorf("sshfs service host does not expose endpoint port %s", endpointPort)
	}

	var tcpPorts []Port
	for _, exposed := range running.Ports {
		if exposed.Protocol == "" || exposed.Protocol == NetworkProtocolTCP {
			tcpPorts = append(tcpPorts, exposed)
		}
	}
	if len(tcpPorts) == 0 {
		return "", fmt.Errorf("sshfs service host exposes no TCP ports")
	}
	if len(tcpPorts) > 1 {
		return "", fmt.Errorf("sshfs service host exposes multiple TCP ports; include the SSH port in the endpoint")
	}
	return strconv.Itoa(tcpPorts[0].Port), nil
}

func sshfsSourceForURL(u *url.URL, host string) (string, error) {
	user := ""
	if u.User != nil {
		user = u.User.Username()
	}
	if user == "" {
		return "", fmt.Errorf("sshfs endpoint missing user")
	}
	if host == "" {
		return "", fmt.Errorf("sshfs endpoint missing host")
	}
	if strings.HasPrefix(host, "-") {
		return "", fmt.Errorf("sshfs endpoint host must not start with '-'")
	}
	path := u.Path
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("sshfs endpoint missing absolute path")
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s@%s:%s", user, host, path), nil
}

func plaintextSecret(ctx context.Context, secret dagql.ObjectResult[*Secret], label string) ([]byte, error) {
	if secret.Self() == nil {
		return nil, fmt.Errorf("%s missing", label)
	}
	plaintext, err := secret.Self().Plaintext(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return plaintext, nil
}

func unmountWithDetachFallback(dir string) func() error {
	return func() error {
		if err := ctrdmount.Unmount(dir, 0); err != nil {
			if detachErr := ctrdmount.Unmount(dir, unix.MNT_DETACH); detachErr != nil {
				return errors.Join(err, detachErr)
			}
		}
		return nil
	}
}

func joinCleanup(first, second func() error) func() error {
	return func() error {
		return errors.Join(callCleanup(first), callCleanup(second))
	}
}

func callCleanup(cleanup func() error) error {
	if cleanup == nil {
		return nil
	}
	return cleanup()
}

func formatCommandStderr(stderr []byte) string {
	trimmed := strings.TrimSpace(string(stderr))
	if trimmed == "" {
		return ""
	}
	return ": " + trimmed
}
