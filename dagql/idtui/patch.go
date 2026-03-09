package idtui

import (
	"context"
	"fmt"

	"github.com/G-Research/dagger"
	"github.com/G-Research/dagger/util/patchpreview"
)

func PreviewPatch(ctx context.Context, changeset *dagger.Changeset) (*patchpreview.PatchPreview, error) {
	rawPatch, err := changeset.AsPatch().Contents(ctx)
	if err != nil {
		return nil, fmt.Errorf("get patch: %w", err)
	}
	return patchpreview.New(ctx, rawPatch, changeset)
}
