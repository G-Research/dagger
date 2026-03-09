// ctrns provides utilities for containerd resources that are pre-namespaced
// (instead)
package ctrns

import (
	"github.com/containerd/containerd/v2/core/content"
	containerdsnapshotter "github.com/G-Research/dagger/internal/buildkit/snapshot/containerd"
)

type ContentStoreNamespaced = containerdsnapshotter.Store

func ContentWithNamespace(store content.Store, ns string) *ContentStoreNamespaced {
	return containerdsnapshotter.NewContentStore(store, ns)
}
