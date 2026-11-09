package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// MapGroupReader lit la table des groupes de cartes d'un joueur : une carte de base et ses
// variantes déclarées (map_variants.toml). Implémenté par platform/duckdb.MapGroupsRepo.
//
// Source : le registre des matchs du titre du joueur (nom retenu par carte). Un titre sans suffixe
// déclaré rend un groupe vide : chaque identifiant reste seul, ce n'est pas une erreur.
type MapGroupReader interface {
	// MapGroups rend les groupes de cartes du titre. Une erreur de lecture est rendue à l'appelant,
	// qui décide de dégrader (variantes laissées seules) après l'avoir journalisée.
	MapGroups(ctx context.Context) (domain.GroupesDeCartes, error)
}
