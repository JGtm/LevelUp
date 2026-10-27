package duckdb

// map_labels.go — LE LIBELLÉ D'UNE CARTE, une seule règle pour l'Explorateur (son historique et les
// options de son filtre de cartes), l'onglet Tactique (vignettes, titre du plan, lien vers
// l'Explorateur) et le contexte du score d'engagement. Le lien « Voir les matchs de cette carte dans
// l'Explorateur » passe le libellé de la vignette en `?maps=`, que l'Explorateur compare au libellé
// de chacun de ses matchs : deux résolutions donneraient deux chaînes pour la même carte, et le lien
// ne filtrerait plus rien.
//
// LA RÈGLE (celle que l'Explorateur appliquait déjà) :
//  1. le nom du registre dans la langue par défaut, quand il existe et diffère du nom de la langue
//     de repli ;
//  2. sinon la traduction de l'asset de la carte (`metadata.asset_translations`, par map_id) dans
//     cette langue, rognée ;
//  3. sinon le nom du registre (langue par défaut, puis langue de repli).
//
// Garde-rail : archlint/map_label_single_resolution_test.go.

import (
	"context"
	"strings"

	"levelup/go-api/internal/i18n"
)

// traductionsDeCartes lit en UNE requête les noms de chaque carte dans toutes les langues (map_id →
// noms par langue). Metadata absente ou aucune carte : nil. Un échec est journalisé et consigné
// (assetNameTexts) puis dégradé — le registre nomme alors la carte.
func traductionsDeCartes(ctx context.Context, meta *DB, mapIDs []string) map[string]i18n.Text {
	if meta == nil || len(mapIDs) == 0 {
		return nil
	}
	return assetNameTexts(ctx, meta, assetTypeMap, mapIDs)
}

// nomsDeCarte applique la règle à une carte et rend ses noms dans chaque langue : ses noms au registre
// (une entrée par langue lue) complétés par ses traductions d'asset (vides si aucune). Le nom de la
// langue par défaut est toujours posé (à défaut celui de la langue de repli) : c'est la clé des filtres.
func nomsDeCarte(registre, traductions i18n.Text) i18n.Text {
	t := registre.Clone()
	if t == nil {
		t = i18n.Text{}
	}
	fillUntranslated(&t, traductions)
	if strings.TrimSpace(t[i18n.Default]) == "" {
		t[i18n.Default] = t[i18n.Fallback]
	}
	return t
}

// libelleDeCarte rend le libellé de la carte dans la langue par défaut (la clé des filtres), à défaut
// celui de la langue de repli : nomsDeCarte[i18n.Default].
func libelleDeCarte(registre, traductions i18n.Text) string {
	return nomsDeCarte(registre, traductions)[i18n.Default]
}
