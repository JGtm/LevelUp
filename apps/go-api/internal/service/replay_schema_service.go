// Package service — replay_schema_service.go : le décompte des rejeux rangés par version de schéma (page
// d'administration), titre par titre. La lecture des en-têtes est derrière le port `ReplaySchemaCensus` ; ce service
// range les titres qui rangent des rejeux et date la réponse.
package service

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ReplaySchemaDeps assemble ce que le service ne sait pas construire lui-même.
type ReplaySchemaDeps struct {
	Census port.ReplaySchemaCensus
	// Titles rend les titres qui rangent des rejeux (capacité du titre, jamais un slug comparé ici).
	Titles func() []string
}

// ReplaySchemaService décompte les rejeux par schéma.
type ReplaySchemaService struct {
	deps ReplaySchemaDeps
}

// NewReplaySchemaService construit le service.
func NewReplaySchemaService(deps ReplaySchemaDeps) *ReplaySchemaService {
	return &ReplaySchemaService{deps: deps}
}

// ReplaySchema rend le décompte de chaque titre. Le premier titre dont le décompte échoue interrompt la réponse.
func (s *ReplaySchemaService) ReplaySchema(ctx context.Context) (domain.AdminReplaySchema, error) {
	out := domain.AdminReplaySchema{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		CurrentSchema: s.deps.Census.CurrentSchema(),
		Titles:        []domain.AdminReplaySchemaTitle{},
	}
	for _, slug := range s.deps.Titles() {
		count, err := s.deps.Census.CountReplaySchemas(ctx, slug)
		if err != nil {
			return domain.AdminReplaySchema{}, fmt.Errorf("décompte des rejeux du titre %s: %w", slug, err)
		}
		out.Titles = append(out.Titles, count)
	}
	return out, nil
}
