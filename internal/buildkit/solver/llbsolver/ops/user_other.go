//go:build !linux && !windows
// +build !linux,!windows

package ops

import (
	"github.com/G-Research/dagger/internal/buildkit/snapshot"
	"github.com/G-Research/dagger/internal/buildkit/solver/pb"
	"github.com/G-Research/dagger/internal/buildkit/worker"
	copy "github.com/G-Research/dagger/internal/fsutil/copy"
	"github.com/pkg/errors"
)

func getReadUserFn(_ worker.Worker) func(chopt *pb.ChownOpt, mu, mg snapshot.Mountable) (*copy.User, error) {
	return readUser
}

func readUser(chopt *pb.ChownOpt, mu, mg snapshot.Mountable) (*copy.User, error) {
	if chopt == nil {
		return nil, nil
	}
	return nil, errors.New("only implemented in linux and windows")
}
