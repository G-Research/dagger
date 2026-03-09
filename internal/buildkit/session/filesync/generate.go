package filesync

//go:generate protoc -I=. -I=../../../../ --gogoslick_out=Minternal/fsutil/types/stat.proto=github.com/G-Research/dagger/internal/fsutil/types,plugins=grpc:. filesync.proto
