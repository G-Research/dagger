package config

import "github.com/G-Research/dagger/internal/buildkit/util/compression"

type RefConfig struct {
	Compression            compression.Config
	PreferNonDistributable bool
}
