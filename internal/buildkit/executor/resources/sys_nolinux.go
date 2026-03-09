//go:build !linux

package resources

import resourcestypes "github.com/G-Research/dagger/internal/buildkit/executor/resources/types"

func newSysSampler() (*Sampler[*resourcestypes.SysSample], error) {
	return nil, nil
}
