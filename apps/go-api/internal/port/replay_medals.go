package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// MedalNameLookup retrouve l'identité d'une médaille depuis son nom ANGLAIS, celui que le film
// inscrit dans le document de rejeu : le pont entre ce nom et le référentiel du jeu.
//
// C'est la méthode `LookupMedalMetaByName` de [MatchViewRepository], rendue seule : le service de
// rejeu n'a besoin que d'elle. Un nom que le référentiel ne connaît pas est absent du résultat.
type MedalNameLookup interface {
	LookupMedalMetaByName(ctx context.Context, namesEN []string) (map[string]domain.MedalNameMeta, error)
}
