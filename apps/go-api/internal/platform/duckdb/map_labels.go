package duckdb

// map_labels.go — LE LIBELLÉ D'UNE CARTE, une seule règle pour l'Explorateur (son historique et les
// options de son filtre de cartes), l'onglet Tactique (vignettes, titre du plan, lien vers
// l'Explorateur) et le contexte du score d'engagement. Le lien « Voir les matchs de cette carte dans
// l'Explorateur » passe le libellé de la vignette en `?maps=`, que l'Explorateur compare au libellé
// de chacun de ses matchs : deux résolutions donneraient deux chaînes pour la même carte, et le lien
// ne filtrerait plus rien.
//
// LA RÈGLE (celle que l'Explorateur appliquait déjà) :
//  1. le nom FR du registre, quand il existe et diffère du nom EN ;
//  2. sinon la traduction de l'asset de la carte (`metadata.asset_translations`, par map_id), en
//     cascade fr-FR, fr, en-US, en puis toute langue (PreferredLangsForLocale), rognée ;
//  3. sinon le nom du registre.
//
// Garde-rail : archlint/map_label_single_resolution_test.go.

import (
	"context"
	"log/slog"
	"strings"
)

// traductionsDeCartes lit en UNE requête la traduction de chaque carte (map_id → libellé). Metadata
// absente ou aucune carte : nil. Un échec est journalisé puis dégradé — le registre nomme alors la
// carte.
func traductionsDeCartes(ctx context.Context, meta *DB, mapIDs []string) map[string]string {
	if meta == nil || len(mapIDs) == 0 {
		return nil
	}
	noms, err := NewMetadataRepoFromDB(meta).ResolveAssetNamesBulk(ctx, "map", mapIDs, PreferredLangsForLocale("fr"))
	if err != nil {
		slog.WarnContext(ctx, "cartes : traductions illisibles — libellés du registre",
			"cartes", len(mapIDs), "err", err)
		return nil
	}
	return noms
}

// libelleDeCarte applique la règle à une carte : ses noms FR et EN au registre ("" si absents) et
// sa traduction d'asset ("" si aucune).
func libelleDeCarte(registreFR, registreEN, traduction string) string {
	if !needsHomeAssetTranslation(registreFR, registreEN) {
		return registreFR
	}
	if t := strings.TrimSpace(traduction); t != "" {
		return t
	}
	if registreFR != "" {
		return registreFR
	}
	return registreEN
}
