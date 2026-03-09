package ctrns

import (
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/G-Research/dagger/internal/buildkit/util/leaseutil"
)

type LeasesManagerNamespace = leaseutil.Manager

func LeasesWithNamespace(leases leases.Manager, ns string) leases.Manager {
	return leaseutil.WithNamespace(leases, ns)
}
