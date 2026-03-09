package schema

import (
	"github.com/G-Research/dagger/core"
	"github.com/G-Research/dagger/dagql"
)

type socketSchema struct{}

var _ SchemaResolvers = &socketSchema{}

func (s *socketSchema) Install(srv *dagql.Server) {
	dagql.Fields[*core.Socket]{}.Install(srv)
}
