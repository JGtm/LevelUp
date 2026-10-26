// Package mapgroups lit la table des groupes de cartes pour les services : une carte de base et ses
// variantes déclarées (map_variants.toml) forment UNE ligne de toute statistique par carte. Feuille
// partagée par `service` et `service/teammates`, qui ne peuvent pas s'importer.
package mapgroups

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Load lit la table des groupes de cartes. Lecteur non câblé (titre sans variante, test) : aucun
// groupe. Une lecture en échec est journalisée puis dégradée — les variantes restent alors seules, la
// page se sert plutôt qu'en erreur.
func Load(ctx context.Context, reader port.MapGroupReader) domain.GroupesDeCartes {
	if reader == nil {
		return domain.GroupesDeCartes{}
	}
	groups, err := reader.MapGroups(ctx)
	if err != nil {
		slog.WarnContext(ctx, "cartes : groupes de variantes illisibles, variantes laissées seules", "err", err)
		return domain.GroupesDeCartes{}
	}
	return groups
}
