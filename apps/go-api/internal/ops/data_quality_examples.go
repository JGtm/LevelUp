// Package ops — data_quality_examples.go : résolution des match_id d'exemple
// (≤3) pour chaque ligne d'inconnu SERVIE (C7). Lecture seule, requête bornée
// par item — permet d'ouvrir la vue de match et de décider traduire/résoudre en
// connaissance de cause. Extrait de data_quality.go (seuil fichier).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// exampleMatchIDLimit : nombre max de match_id d'exemple par ligne d'inconnu.
const exampleMatchIDLimit = 3

// enrichExampleMatchIDs remplit ExampleMatchIDs (≤3, matchs les plus récents)
// pour chaque issue de la fenêtre servie. Best-effort : une erreur par item est
// loguée (WARN) et laisse la liste vide — un exemple manquant ne casse jamais la
// réponse. Ne sonde QUE les lignes rendues (fenêtre bornée par limit/offset).
func enrichExampleMatchIDs(ctx context.Context, sharedDB *sql.DB, kind string, items []DataQualityIssue) {
	if sharedDB == nil {
		return
	}
	for i := range items {
		ids, err := exampleMatchIDsFor(ctx, sharedDB, kind, items[i])
		if err != nil {
			slog.WarnContext(ctx, "data_quality: exemples de match indisponibles",
				"module", "monitoring", "kind", kind, "id", items[i].ID, "err", err)
			continue
		}
		items[i].ExampleMatchIDs = ids
	}
}

// exampleMatchIDsFor résout jusqu'à 3 match_id concrets pour une ligne d'inconnu.
// La sonde dépend du kind : colonne de match_registry (assets/mode/playlist) ou
// match_participants (xuid). Retourne nil sans erreur si la ligne n'a pas de clé
// sondable (kind hors périmètre, asset_kind inconnu, mode sans échantillon).
func exampleMatchIDsFor(ctx context.Context, sharedDB *sql.DB, kind string, it DataQualityIssue) ([]string, error) {
	switch kind {
	case "raw_uuids":
		col := rawUUIDColumnFor(it.AssetKind)
		if col == "" {
			return nil, nil
		}
		return queryExampleMatchIDs(ctx, sharedDB, fmt.Sprintf(
			`SELECT match_id FROM match_registry WHERE %s = ? ORDER BY %s DESC LIMIT %d`,
			col, dqTimestampExpr, exampleMatchIDLimit), it.ID)
	case "untranslated_modes":
		// L'ID est une clé de mode NORMALISÉE (non requêtable en SQL) ; on sonde
		// via l'échantillon de pair_name brut porté par Label (le plus fréquent).
		if it.Label == "" {
			return nil, nil
		}
		return queryExampleMatchIDs(ctx, sharedDB, fmt.Sprintf(
			`SELECT match_id FROM match_registry WHERE pair_name = ? ORDER BY %s DESC LIMIT %d`,
			dqTimestampExpr, exampleMatchIDLimit), it.Label)
	case "orphan_playlists":
		return queryExampleMatchIDs(ctx, sharedDB, fmt.Sprintf(
			`SELECT match_id FROM match_registry WHERE playlist_id = ? ORDER BY %s DESC LIMIT %d`,
			dqTimestampExpr, exampleMatchIDLimit), it.ID)
	case "orphan_xuids":
		return queryExampleMatchIDs(ctx, sharedDB, fmt.Sprintf(
			`SELECT DISTINCT match_id FROM match_participants WHERE xuid = ? LIMIT %d`,
			exampleMatchIDLimit), it.ID)
	}
	return nil, nil
}

// rawUUIDColumnFor mappe un asset_kind (playlist|map|pair|game_variant) vers sa
// colonne id de match_registry, en réutilisant rawUUIDColumns (source unique).
func rawUUIDColumnFor(assetKind string) string {
	for _, c := range rawUUIDColumns {
		if c.Kind == assetKind {
			return c.IDCol
		}
	}
	return ""
}

// queryExampleMatchIDs exécute une sonde d'exemples bornée et collecte les
// match_id non vides. Un seul argument : l'identifiant sondé.
func queryExampleMatchIDs(ctx context.Context, db *sql.DB, query string, arg any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr == nil && id != "" {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}
