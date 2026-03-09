//go:build !dfrunsecurity
// +build !dfrunsecurity

package dockerfile2llb

import (
	"github.com/G-Research/dagger/internal/buildkit/client/llb"
	"github.com/G-Research/dagger/internal/buildkit/frontend/dockerfile/instructions"
)

func dispatchRunSecurity(_ *instructions.RunCommand) (llb.RunOption, error) {
	return nil, nil
}
