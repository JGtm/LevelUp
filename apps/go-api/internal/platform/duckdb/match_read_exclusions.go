// Package duckdb — match_read_exclusions.go : masquage READ-SIDE de certains
// game_variant par titre.
//
// Halo 5 : les matchs de Campagne (PvE, hors scope de l'app) ne sont plus collectés
// (cf. internal/games/halo_5.isExcludedH5GameMode), mais des matchs historiques
// subsistent dans le shared DB. On les masque à la LECTURE plutôt que de purger la
// base (décision produit). Identification par game_variant_id (GUID stables de l'API
// Metadata Halo 5 ; cf. metadata.asset_translations : "Campaign" / "Campaign Score
// Attack"). Le shared DB étant isolé par titre (data/titles/{slug}/warehouse/), ce
// filtre n'impacte aucun autre titre.
package duckdb

import (
	"fmt"
	"strings"
)

// readExcludedGameVariantIDs : game_variant_id masqués à la lecture, par titre.
var readExcludedGameVariantIDs = map[string][]string{
	"halo_5": {
		"00000003-0000-0010-8000-00aa00389b71", // Campaign
		"67ffc2ff-a50e-4e5d-ae08-b40e3d961061", // Campaign Score Attack
	},
}

// ExcludedVariantClause retourne une clause SQL
// ` AND COALESCE(<alias>.game_variant_id, ”) NOT IN (?, ?, ...)` + ses args pour le
// titre donné, ou ("", nil) si le titre n'a aucun mode masqué. `alias` est l'alias de
// la table match_registry / v_match_full dans la requête appelante (ex. "r").
func ExcludedVariantClause(titleSlug, alias string) (string, []any) {
	ids := readExcludedGameVariantIDs[titleSlug]
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	clause := fmt.Sprintf(" AND COALESCE(%s.game_variant_id, '') NOT IN (%s)",
		alias, strings.Join(placeholders, ","))
	return clause, args
}
