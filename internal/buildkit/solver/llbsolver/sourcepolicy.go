package llbsolver

import (
	"context"

	"github.com/G-Research/dagger/internal/buildkit/solver/pb"
)

type SourcePolicyEvaluator interface {
	Evaluate(ctx context.Context, op *pb.SourceOp) (bool, error)
}
