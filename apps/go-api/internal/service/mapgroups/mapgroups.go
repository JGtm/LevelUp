// Package mapgroups lit la table des groupes de cartes pour les services : une carte de base et ses
// variantes déclarées (map_variants.toml) forment UNE ligne de toute statistique par carte. Feuille
// partagée par `service` et `service/teammates`, qui ne peuvent pas s'importer.
package mapgroups

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
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

// RefKey rend la clé de regroupement d'une carte canonique : l'identifiant de BASE de son groupe, à défaut
// son propre identifiant. "" sans carte ni identifiant.
func RefKey(ref *canonical.AssetReference, groups domain.GroupesDeCartes) string {
	if ref == nil || ref.ID == "" {
		return ""
	}
	return groups.Groupe(ref.ID)
}

// RefLabel rend le libellé d'une carte canonique dans `locale` : celui de la carte de BASE pour une carte
// de groupe (jamais le nom d'une variante), sinon `own`, le libellé propre que l'appelant aurait pris.
func RefLabel(ref *canonical.AssetReference, groups domain.GroupesDeCartes, locale i18n.Locale, own string) string {
	if ref != nil {
		if base, ok := groups.Libelle(ref.ID, locale); ok {
			return base
		}
	}
	return own
}
