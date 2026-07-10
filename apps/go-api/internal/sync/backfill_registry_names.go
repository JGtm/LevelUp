// Package sync — backfill_registry_names.go : backfill des noms d'assets
// (playlist, map, pair, game_variant) dans match_registry pour les rows où
// le nom est égal à l'UUID (cas du fallback UUID dans ExtractRegistry, avant
// l'introduction de EnrichRegistryFromMetadata 2026-05-09).
//
// Stratégie : pour chaque colonne *_name dont la valeur == *_id, on lookup
// metadata.asset_translations[lang='en-US'] pour récupérer le nom canonique
// EN, puis on UPDATE. Si asset_translations n'a pas de ligne pour cet UUID,
// on conserve la valeur (UUID) — la donnée reste en l'état mais sans dégrader.
//
// Utilisation : cmd/backfill_registry_names CLI, ou intégration future à un
// flag SyncScope si le besoin émerge.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games"
)

// BackfillRegistryStats résume les compteurs après un run de backfill.
type BackfillRegistryStats struct {
	PlaylistsScanned int
	PlaylistsFixed   int
	MapsScanned      int
	MapsFixed        int
	PairsScanned     int
	PairsFixed       int
	VariantsScanned  int
	VariantsFixed    int
}

// Total retourne la somme des fixed pour log/affichage.
func (s BackfillRegistryStats) Total() int {
	return s.PlaylistsFixed + s.MapsFixed + s.PairsFixed + s.VariantsFixed
}

// BackfillRegistryNames remplace les *_name == *_id dans match_registry par
// le nom canonique en-US depuis metadata.asset_translations. Idempotent :
// re-exécuter sur une DB déjà nettoyée est un no-op.
//
// metadataDB peut être nil → log warning et retourne stats vides (tests).
func BackfillRegistryNames(ctx context.Context, sharedDB, metadataDB *sql.DB) (BackfillRegistryStats, error) {
	var stats BackfillRegistryStats
	if metadataDB == nil {
		slog.WarnContext(ctx, "BackfillRegistryNames: metadata DB nil — abort")
		return stats, nil
	}
	if sharedDB == nil {
		return stats, fmt.Errorf("BackfillRegistryNames: sharedDB nil")
	}

	type column struct {
		assetType string
		idCol     string
		nameCol   string
		scanned   *int
		fixed     *int
	}
	cols := []column{
		{games.AssetKindPlaylist, "playlist_id", "playlist_name", &stats.PlaylistsScanned, &stats.PlaylistsFixed},
		{games.AssetKindMap, "map_id", "map_name", &stats.MapsScanned, &stats.MapsFixed},
		{games.AssetKindPair, "pair_id", "pair_name", &stats.PairsScanned, &stats.PairsFixed},
		{games.AssetKindGameVariant, "game_variant_id", "game_variant_name", &stats.VariantsScanned, &stats.VariantsFixed},
	}

	for _, c := range cols {
		fixed, scanned, err := backfillOneColumn(ctx, sharedDB, metadataDB, c.assetType, c.idCol, c.nameCol)
		if err != nil {
			return stats, fmt.Errorf("backfill %s: %w", c.assetType, err)
		}
		*c.scanned = scanned
		*c.fixed = fixed
	}

	// Étape finale : pour les paires dont le nom est resté un GUID (asset
	//_translations[pair] absent — cas du nouveau contenu Halo), on CONSTRUIT
	// "{game_variant} on {map}" depuis les colonnes voisines, désormais résolues
	// par les passes ci-dessus. Miroir du fallback sync-time dans
	// EnrichRegistryFromMetadata (constructPairName). Idempotent.
	constructed, err := backfillPairNamesByConstruction(ctx, sharedDB)
	if err != nil {
		return stats, fmt.Errorf("backfill pair construction: %w", err)
	}
	stats.PairsFixed += constructed
	return stats, nil
}

// backfillPairNamesByConstruction réécrit pair_name = "{game_variant_name} on
// {map_name}" pour les rows où pair_name est resté un GUID (== pair_id) ALORS
// que game_variant_name et map_name sont, eux, résolus (≠ leur id). Idempotent.
// Retourne le nombre de rows mises à jour.
//
// Row-by-row par match_id (ADR 0019/0026, anti-ART #23046) : SELECT des match_ids
// cibles + de leur pair_name construit, puis N UPDATE sérialisés `WHERE match_id = ?`.
// JAMAIS de bulk UPDATE multi-row nu (set-based) sur match_registry — même sous
// write-lease, un statement touchant N entrées d'index est le déclencheur ART direct
// (garde-fou : no_art_patterns_test.go::TestNoBareBulkUpdateOnCriticalTables).
func backfillPairNamesByConstruction(ctx context.Context, sharedDB *sql.DB) (int, error) {
	const sel = `
		SELECT match_id, game_variant_name || ' on ' || map_name AS pair
		FROM match_registry
		WHERE pair_id IS NOT NULL
		  AND pair_name = pair_id
		  AND game_variant_name IS NOT NULL AND game_variant_name <> game_variant_id
		  AND map_name IS NOT NULL AND map_name <> map_id`
	rows, err := sharedDB.QueryContext(ctx, sel)
	if err != nil {
		return 0, err
	}
	type pairUpdate struct{ matchID, pair string }
	var updates []pairUpdate
	for rows.Next() {
		var mid, pair sql.NullString
		if err := rows.Scan(&mid, &pair); err != nil {
			rows.Close()
			return 0, err
		}
		if mid.Valid && mid.String != "" && pair.Valid && pair.String != "" {
			updates = append(updates, pairUpdate{mid.String, pair.String})
		}
	}
	rows.Close()

	n := 0
	for _, u := range updates {
		res, err := sharedDB.ExecContext(ctx,
			`UPDATE match_registry SET pair_name = ? WHERE match_id = ?`, u.pair, u.matchID)
		if err != nil {
			return n, err
		}
		if c, _ := res.RowsAffected(); c > 0 {
			n++
		}
	}
	if n > 0 {
		slog.InfoContext(ctx, "BackfillRegistryNames: pairs construites depuis game_variant + map",
			"rows", n)
	}
	return n, nil
}

// backfillOneColumn : (1) liste les asset_ids distincts dont *_name == *_id,
// (2) lookup les noms en-US depuis asset_translations, (3) UPDATE match_registry.
//
// Retourne (fixed, scanned, error). scanned = nb d'asset_ids distincts à fixer
// initialement ; fixed = nb d'UPDATEs effectifs (peut être inférieur si
// asset_translations n'a pas le nom).
func backfillOneColumn(
	ctx context.Context,
	sharedDB, metadataDB *sql.DB,
	assetType, idCol, nameCol string,
) (fixed, scanned int, err error) {
	// Étape 1 : liste des asset_ids distincts où name == id.
	q1 := fmt.Sprintf(`
		SELECT DISTINCT %s
		FROM match_registry
		WHERE %s IS NOT NULL
		  AND %s = %s`, idCol, idCol, nameCol, idCol)
	rows, err := sharedDB.QueryContext(ctx, q1)
	if err != nil {
		return 0, 0, fmt.Errorf("query distinct %s: %w", idCol, err)
	}
	var ids []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		if id.Valid && id.String != "" {
			ids = append(ids, id.String)
		}
	}
	rows.Close()
	scanned = len(ids)
	if scanned == 0 {
		return 0, 0, nil
	}

	// Étape 2 : lookup asset_translations[en-US] pour chaque asset_id.
	for _, id := range ids {
		var name sql.NullString
		err := metadataDB.QueryRowContext(ctx, `
			SELECT name FROM asset_translations
			WHERE asset_type = ? AND asset_id = ? AND lang = 'en-US'
			LIMIT 1`, assetType, id).Scan(&name)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			slog.WarnContext(ctx, "BackfillRegistryNames: lookup en-US failed",
				"asset_type", assetType, "asset_id", id, "err", err)
			continue
		}
		if !name.Valid {
			continue
		}
		canonical := name.String
		if canonical == "" || canonical == id {
			continue
		}
		// Étape 3 : UPDATE match_registry pour cet asset_id.
		q3 := fmt.Sprintf(`UPDATE match_registry SET %s = ? WHERE %s = ? AND %s = ?`,
			nameCol, idCol, nameCol)
		res, err := sharedDB.ExecContext(ctx, q3, canonical, id, id)
		if err != nil {
			slog.WarnContext(ctx, "BackfillRegistryNames: UPDATE failed",
				"asset_type", assetType, "asset_id", id, "err", err)
			continue
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			fixed++
			slog.InfoContext(ctx, "BackfillRegistryNames: fixed",
				"asset_type", assetType, "asset_id", id, "name", canonical, "rows", n)
		}
	}
	return fixed, scanned, nil
}
