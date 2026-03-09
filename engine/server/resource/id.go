package resource

import "github.com/G-Research/dagger/dagql/call"

type ID struct {
	call.ID
	Optional bool
}
