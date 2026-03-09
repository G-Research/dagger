package moby_buildkit_v1_types //nolint:revive

//go:generate protoc -I=. -I=../../../../ --gogo_out=Minternal/buildkit/solver/pb/ops.proto=github.com/G-Research/dagger/internal/buildkit/solver/pb,plugins=grpc:. worker.proto
