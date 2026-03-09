//go:build !windows
// +build !windows

package local

import (
	"context"
	"io"

	"github.com/G-Research/dagger/internal/fsutil"
)

func writeTar(ctx context.Context, fs fsutil.FS, w io.WriteCloser) error {
	return fsutil.WriteTar(ctx, fs, w)
}
