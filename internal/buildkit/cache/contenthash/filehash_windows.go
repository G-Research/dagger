//go:build windows
// +build windows

package contenthash

import (
	"os"

	fstypes "github.com/G-Research/dagger/internal/fsutil/types"
)

func setUnixOpt(_ string, _ os.FileInfo, _ *fstypes.Stat) error {
	return nil
}
