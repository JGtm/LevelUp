package port

// replay_schema_census.go — la lecture qui compte les rejeux rangés d'un titre par version de schéma (page
// d'administration « rejeux à l'ancien schéma »).

import (
	"context"

	"levelup/go-api/internal/domain"
)

// ReplaySchemaCensus lit les en-têtes des rejeux rangés d'un titre.
type ReplaySchemaCensus interface {
	// CurrentSchema rend le schéma que produit le serveur qui répond.
	CurrentSchema() int
	// CountReplaySchemas compte les rejeux rangés du titre : courants, à l'ancien schéma, illisibles. Un dossier
	// absent est un titre sans rejeu (tous les comptes à zéro), pas une erreur.
	CountReplaySchemas(ctx context.Context, titleSlug string) (domain.AdminReplaySchemaTitle, error)
}
