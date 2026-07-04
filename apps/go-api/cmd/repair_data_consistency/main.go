//go:build cgo

// repair_data_consistency — outil one-shot de réparation des inconsistances
// résiduelles détectées par diag_db_health (2026-05-08).
//
// Couvre 3 chantiers :
//
//  1. PLAYER MIGRATIONS — force l'application des migrations TargetPlayer
//     sur TOUTES les player DBs sous `data/titles/halo_infinite/players/*`.
//     Compense le fait que `ensurePlayerDBMigrations` ne tourne qu'à
//     l'ouverture du pool, donc une player DB jamais ouverte depuis le reboot
//     reste dans son ancien schéma. Spécifiquement : applique
//     `cleanup_spartan_customization_garbage_urls` aux DBs qui ont raté le
//     premier passage.
//
//  2. MATCH_REGISTRY UUIDs — vide les colonnes `map_name` et `pair_name`
//     contenant un UUID brut (sync n'a pas résolu). Le code-side cascade
//     resolver (`MatchViewRepo.resolveAssetName`) reprend la résolution via
//     `asset_translations` à l'affichage, donc une fois le brut purgé, les
//     UI affichent les noms propres au lieu de l'UUID.
//
//  3. XUID_ALIASES BACKFILL — cross-match : pour chaque xuid orphelin
//     (présent en match_participants mais absent de xuid_aliases ET sans
//     gamertag dans match_participants), tente de retrouver son gamertag
//     dans match_participants (lignes rares avec gamertag non-null pour le
//     même xuid). Si trouvé, INSERT dans shared.xuid_aliases.
//
// Read-write. NE PAS lancer pendant que le serveur Air tourne (locks).
//
// Usage :
//
//	go run -tags cgo ./cmd/repair_data_consistency           # exécution
//	go run -tags cgo ./cmd/repair_data_consistency --dry-run # plan seulement
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Afficher les actions sans modifier les DBs")
	dataRoot := flag.String("data-root", "../../data", "Racine des données (default ../../data)")
	flag.Parse()

	mode := "EXECUTE"
	if *dryRun {
		mode = "DRY-RUN"
	}
	fmt.Printf("══ repair_data_consistency [%s] ══\n\n", mode)

	titleDir := filepath.Join(*dataRoot, "titles", "halo_infinite")
	if err := chantier1ForcePlayerMigrations(filepath.Join(titleDir, "players"), *dryRun); err != nil {
		fmt.Printf("[chantier 1] ERR: %v\n", err)
	}
	fmt.Println()
	if err := chantier2CleanupRegistryUUIDs(filepath.Join(titleDir, "warehouse", "shared_matches_v2.duckdb"), *dryRun); err != nil {
		fmt.Printf("[chantier 2] ERR: %v\n", err)
	}
	fmt.Println()
	if err := chantier3BackfillXUIDAliases(filepath.Join(titleDir, "warehouse", "shared_matches_v2.duckdb"), *dryRun); err != nil {
		fmt.Printf("[chantier 3] ERR: %v\n", err)
	}
	fmt.Println()
	if err := chantier4ResetLyingBits(filepath.Join(titleDir, "warehouse", "shared_matches_v2.duckdb"), *dryRun); err != nil {
		fmt.Printf("[chantier 4] ERR: %v\n", err)
	}
	fmt.Println()
	fmt.Println("══ FIN ══")
}

// ─────────────────────────────────────────────────────────────────────────
// Chantier 4 — Reset des bits menteurs (oubli post-fix parser)
// ─────────────────────────────────────────────────────────────────────────

// chantier4ResetLyingBits délègue à ops.ResetLyingBits (logique partagée avec
// l'action admin POST /admin/actions/lying-bits/reset). Clear les bits
// MBitEvents (16) et MBitWeaponKills (21) de match_registry.backfill_completed
// pour les matchs où le bit est set mais la table correspondante est vide, plus
// `events_loaded=TRUE` menteur. Le heal filtre sur ces flags : tant que le bit
// est set, le match est skip → reset débloque la convergence au prochain sync.
func chantier4ResetLyingBits(sharedPath string, dryRun bool) error {
	fmt.Println("┌─ [chantier 4] Reset bits menteurs ─────────────────────")
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return err
	}
	defer rwDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := ops.ResetLyingBits(ctx, rwDB.SQLDb(), dryRun)
	if err != nil {
		return err
	}

	fmt.Printf("│ MBitEvents (16) menteurs       : %d matchs\n", res.EventsBitsCleared)
	fmt.Printf("│ MBitWeaponKills (21) menteurs  : %d matchs\n", res.WeaponsBitsCleared)
	fmt.Printf("│ events_loaded=TRUE menteur     : %d matchs\n", res.EventsLoadedCleared)
	if dryRun {
		fmt.Println("│ [dry] would clear these bits → heal retentera au prochain sync")
	} else {
		fmt.Printf("│ ✓ bits clearés (total %d matchs) → candidats au heal au prochain sync\n", res.Total())
	}
	fmt.Println("└────────────────────────────────────────────────────────")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────
// Chantier 1 — Force player migrations
// ─────────────────────────────────────────────────────────────────────────

func chantier1ForcePlayerMigrations(playersDir string, dryRun bool) error {
	fmt.Println("┌─ [chantier 1] Force player migrations ─────────────────")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		return err
	}
	var dbs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(playersDir, e.Name(), "stats.duckdb")
		if _, err := os.Stat(path); err == nil {
			dbs = append(dbs, path)
		}
	}
	sort.Strings(dbs)
	fmt.Printf("│ %d player DBs trouvées\n", len(dbs))

	if dryRun {
		for _, p := range dbs {
			fmt.Printf("│   [dry] would run TargetPlayer migrations on : %s\n", filepath.Base(filepath.Dir(p)))
		}
		fmt.Println("└────────────────────────────────────────────────────────")
		return nil
	}

	migration.All() // force registration
	for _, p := range dbs {
		name := filepath.Base(filepath.Dir(p))
		rwDB, err := duckdb.OpenReadWrite(p)
		if err != nil {
			fmt.Printf("│ ✗ %s : open : %v\n", name, err)
			continue
		}
		if err := migration.RunForDB(rwDB.SQLDb(), migration.TargetPlayer); err != nil {
			fmt.Printf("│ ✗ %s : run migrations : %v\n", name, err)
			rwDB.Close()
			continue
		}
		rwDB.Close()
		fmt.Printf("│ ✓ %s : migrations appliquées\n", name)
	}
	fmt.Println("└────────────────────────────────────────────────────────")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────
// Chantier 2 — Cleanup match_registry UUIDs résiduels
// ─────────────────────────────────────────────────────────────────────────

func chantier2CleanupRegistryUUIDs(sharedPath string, dryRun bool) error {
	fmt.Println("┌─ [chantier 2] Cleanup match_registry UUIDs ────────────")
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return err
	}
	defer rwDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`

	var mapCount, pairCount int
	_ = rwDB.QueryRow(ctx, `SELECT COUNT(*) FROM match_registry WHERE map_name ~ ?`, uuidPattern).Scan(&mapCount)
	_ = rwDB.QueryRow(ctx, `SELECT COUNT(*) FROM match_registry WHERE pair_name ~ ?`, uuidPattern).Scan(&pairCount)
	fmt.Printf("│ map_name UUID brut    : %d matchs\n", mapCount)
	fmt.Printf("│ pair_name UUID brut   : %d matchs\n", pairCount)

	if dryRun {
		fmt.Printf("│ [dry] would NULL these columns (code-side resolve cascade kicks in)\n")
		fmt.Println("└────────────────────────────────────────────────────────")
		return nil
	}

	res, err := rwDB.Exec(ctx, `UPDATE match_registry SET map_name = NULL WHERE map_name ~ ?`, uuidPattern)
	if err != nil {
		return fmt.Errorf("UPDATE map_name: %w", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("│ ✓ map_name vidé sur %d lignes\n", n)

	res, err = rwDB.Exec(ctx, `UPDATE match_registry SET pair_name = NULL WHERE pair_name ~ ?`, uuidPattern)
	if err != nil {
		return fmt.Errorf("UPDATE pair_name: %w", err)
	}
	n, _ = res.RowsAffected()
	fmt.Printf("│ ✓ pair_name vidé sur %d lignes\n", n)
	fmt.Println("│ → resolveAssetName (côté MatchViewRepo + home) cascade")
	fmt.Println("│   désormais via asset_translations à l'affichage.")
	fmt.Println("└────────────────────────────────────────────────────────")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────
// Chantier 3 — Backfill xuid_aliases cross-match
// ─────────────────────────────────────────────────────────────────────────

func chantier3BackfillXUIDAliases(sharedPath string, dryRun bool) error {
	fmt.Println("┌─ [chantier 3] Backfill xuid_aliases cross-match ───────")
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return err
	}
	defer rwDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Identifier les xuids orphelins : présents en match_participants mais
	// absents de xuid_aliases (ou avec alias vide). Exclure les bots `bid(%`.
	rows, err := rwDB.Query(ctx, `
		SELECT DISTINCT mp.xuid
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
	`)
	if err != nil {
		return fmt.Errorf("orphan query: %w", err)
	}
	var orphans []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err == nil {
			orphans = append(orphans, x)
		}
	}
	rows.Close()
	fmt.Printf("│ %d xuids orphelins à investiguer\n", len(orphans))

	// Pour chaque orphelin, chercher un gamertag fiable. Sources testées
	// dans l'ordre :
	//  1. match_participants.gamertag (rare en prod : ~0.1% peuplés)
	//  2. killer_victim_pairs.killer_gamertag / victim_gamertag (parsing
	//     highlight events stocke explicitement le gamertag de l'actor)
	type pair struct {
		xuid     string
		gamertag string
		source   string
	}
	var resolved []pair
	var stillOrphan []string
	for _, x := range orphans {
		// Source 1 : match_participants
		var gt sql.NullString
		err := rwDB.QueryRow(ctx, `
			SELECT MAX(gamertag) FROM match_participants
			WHERE xuid = ?
			  AND gamertag IS NOT NULL
			  AND gamertag != ''
			  AND gamertag != xuid
		`, x).Scan(&gt)
		if err == nil && gt.Valid && gt.String != "" {
			resolved = append(resolved, pair{xuid: x, gamertag: gt.String, source: "match_participants"})
			continue
		}
		// Source 2 : killer_victim_pairs (killer_gamertag + victim_gamertag)
		err = rwDB.QueryRow(ctx, `
			SELECT MAX(gt) FROM (
				SELECT killer_gamertag AS gt FROM killer_victim_pairs
				WHERE killer_xuid = ? AND killer_gamertag IS NOT NULL
				  AND killer_gamertag != '' AND killer_gamertag != killer_xuid
				UNION
				SELECT victim_gamertag AS gt FROM killer_victim_pairs
				WHERE victim_xuid = ? AND victim_gamertag IS NOT NULL
				  AND victim_gamertag != '' AND victim_gamertag != victim_xuid
			)
		`, x, x).Scan(&gt)
		if err == nil && gt.Valid && gt.String != "" {
			resolved = append(resolved, pair{xuid: x, gamertag: gt.String, source: "killer_victim_pairs"})
			continue
		}
		stillOrphan = append(stillOrphan, x)
	}

	// Stats par source
	bySource := map[string]int{}
	for _, p := range resolved {
		bySource[p.source]++
	}
	fmt.Printf("│ résolvables (total)                 : %d\n", len(resolved))
	for s, n := range bySource {
		fmt.Printf("│   via %-25s : %d\n", s, n)
	}
	fmt.Printf("│ vraiment orphelins (sans source)    : %d\n", len(stillOrphan))

	if dryRun {
		for i, p := range resolved {
			if i >= 5 {
				fmt.Printf("│   ... et %d autres\n", len(resolved)-5)
				break
			}
			fmt.Printf("│   [dry] would upsert xa[%s] = %s\n", p.xuid, p.gamertag)
		}
		if len(stillOrphan) > 0 {
			fmt.Printf("│ [dry] orphelins sans source (échantillon) :\n")
			for i, x := range stillOrphan {
				if i >= 5 {
					fmt.Printf("│   ... et %d autres\n", len(stillOrphan)-5)
					break
				}
				fmt.Printf("│   - %s\n", x)
			}
		}
		fmt.Println("└────────────────────────────────────────────────────────")
		return nil
	}

	// Upsert effectif
	now := time.Now().UTC()
	upserted := 0
	for _, p := range resolved {
		src := "repair_" + p.source
		_, err := rwDB.Exec(ctx, `
			INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (xuid) DO UPDATE SET
				gamertag   = EXCLUDED.gamertag,
				last_seen  = EXCLUDED.last_seen,
				source     = EXCLUDED.source,
				updated_at = EXCLUDED.updated_at
		`, p.xuid, p.gamertag, now, src, now)
		if err == nil {
			upserted++
		}
	}
	fmt.Printf("│ ✓ %d alias upsertés (source=repair_*)\n", upserted)
	if len(stillOrphan) > 0 {
		fmt.Printf("│ ⚠ %d xuids restent orphelins (aucune source en DB) :\n", len(stillOrphan))
		for i, x := range stillOrphan {
			if i >= 10 {
				fmt.Printf("│   ... et %d autres (voir log complet via --dry-run)\n", len(stillOrphan)-10)
				break
			}
			fmt.Printf("│   - %s\n", x)
		}
		fmt.Println("│ → ces xuids n'ont jamais eu leur gamertag synced ;")
		fmt.Println("│   nécessitent un fetch Microsoft Xbox profile API")
		fmt.Println("│   (cmd/migrate-xuid-aliases-global ou intégration future).")
	}
	fmt.Println("└────────────────────────────────────────────────────────")
	return nil
}
