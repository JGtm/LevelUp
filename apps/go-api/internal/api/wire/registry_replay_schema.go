// Package api — registry_replay_schema.go : le décompte des rejeux rangés par version de schéma (page
// d'administration), tiré du registre de services. La règle est dans service.ReplaySchemaService, la lecture des
// en-têtes dans replaybuild.SchemaCensus.
package wire

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/service"
)

// ReplaySchemaReport compte, titre par titre, les rejeux rangés courants, à l'ancien schéma et illisibles.
func (r *ServiceRegistry) ReplaySchemaReport(ctx context.Context) (domain.AdminReplaySchema, error) {
	return service.NewReplaySchemaService(service.ReplaySchemaDeps{
		Census: replaybuild.NewSchemaCensus(r.cfg.PathResolver()),
		Titles: func() []string { return titlesWithReplay(r.cfg.Roots().Resources, title.DefaultRegistry()) },
	}).ReplaySchema(ctx)
}
