// merge_weapon_kills — restaure les weapon_kills perdus depuis une DB VPS
// vers la DB locale. Idempotent et non-destructif :
//   - n'écrase JAMAIS les rows locales existantes (clause NOT EXISTS)
//   - opère en transaction unique avec rollback en cas d'erreur
//   - mode --dry-run : compte sans modifier
//
// Usage:
//
//	go run . --vps <path> --local <path> --dry-run
//	go run . --vps <path> --local <path> --apply
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	vpsPath := flag.String("vps", "", "chemin vers la DB VPS (READ_ONLY)")
	localPath := flag.String("local", "", "chemin vers la DB locale (R/W)")
	dryRun := flag.Bool("dry-run", true, "compte uniquement, pas de modification")
	apply := flag.Bool("apply", false, "applique réellement le merge (sinon dry-run)")
	flag.Parse()

	if *vpsPath == "" || *localPath == "" {
		log.Fatal("--vps et --local sont obligatoires")
	}
	if *apply {
		*dryRun = false
	}
	mode := "DRY RUN"
	if !*dryRun {
		mode = "APPLY"
	}

	if _, err := os.Stat(*vpsPath); err != nil {
		log.Fatalf("VPS DB introuvable: %v", err)
	}
	if _, err := os.Stat(*localPath); err != nil {
		log.Fatalf("Local DB introuvable: %v", err)
	}

	fmt.Printf("=== merge_weapon_kills — mode=%s ===\n", mode)
	fmt.Printf("VPS   : %s\n", *vpsPath)
	fmt.Printf("Local : %s\n\n", *localPath)

	// Ouverture locale en R/W (DuckDB exclusif → vérifie que le serveur est arrêté).
	local, err := sql.Open("duckdb", *localPath)
	if err != nil {
		log.Fatalf("ouverture locale R/W: %v", err)
	}
	defer local.Close()

	// ATTACH VPS en READ_ONLY.
	if _, err := local.Exec(fmt.Sprintf("ATTACH '%s' AS vps (READ_ONLY)", *vpsPath)); err != nil {
		log.Fatalf("ATTACH VPS: %v", err)
	}
	defer func() {
		if _, err := local.Exec("DETACH vps"); err != nil {
			log.Printf("[WARN] DETACH vps: %v", err)
		}
	}()

	// 1) ÉTAT AVANT
	var localBefore, vpsTotal int
	_ = local.QueryRow(`SELECT COUNT(*) FROM weapon_kills`).Scan(&localBefore)
	_ = local.QueryRow(`SELECT COUNT(*) FROM vps.weapon_kills`).Scan(&vpsTotal)

	var localMatchesBefore, vpsMatches int
	_ = local.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM weapon_kills`).Scan(&localMatchesBefore)
	_ = local.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM vps.weapon_kills`).Scan(&vpsMatches)

	fmt.Println("[ÉTAT AVANT]")
	fmt.Printf("  Local weapon_kills   : %d lignes / %d matchs distincts\n", localBefore, localMatchesBefore)
	fmt.Printf("  VPS   weapon_kills   : %d lignes / %d matchs distincts\n", vpsTotal, vpsMatches)

	// 2) DELTA — combien de rows VPS sont absentes du local (NOT EXISTS sur match+xuid+time_ms)
	var deltaRows, deltaMatches int
	if err := local.QueryRow(`
		SELECT COUNT(*) FROM vps.weapon_kills v
		WHERE NOT EXISTS (
			SELECT 1 FROM weapon_kills l
			WHERE l.match_id = v.match_id
			  AND l.xuid     = v.xuid
			  AND l.time_ms  = v.time_ms
		)`).Scan(&deltaRows); err != nil {
		log.Fatalf("count delta rows: %v", err)
	}
	if err := local.QueryRow(`
		SELECT COUNT(DISTINCT v.match_id) FROM vps.weapon_kills v
		WHERE NOT EXISTS (
			SELECT 1 FROM weapon_kills l
			WHERE l.match_id = v.match_id
			  AND l.xuid     = v.xuid
			  AND l.time_ms  = v.time_ms
		)`).Scan(&deltaMatches); err != nil {
		log.Fatalf("count delta matches: %v", err)
	}

	fmt.Printf("\n[DELTA À RESTAURER]\n")
	fmt.Printf("  Lignes à insérer     : %d\n", deltaRows)
	fmt.Printf("  Matchs concernés     : %d\n", deltaMatches)

	if deltaRows == 0 {
		fmt.Println("\n[OK] Aucun delta — rien à insérer.")
		return
	}

	// 3) Échantillon de 5 matchs concernés pour vérification visuelle
	fmt.Println("\n[ÉCHANTILLON 10 matchs concernés]")
	rows, err := local.Query(`
		SELECT v.match_id, COUNT(*) AS rows_to_insert,
		       COUNT(DISTINCT v.xuid) AS xuid_count
		FROM vps.weapon_kills v
		WHERE NOT EXISTS (
			SELECT 1 FROM weapon_kills l
			WHERE l.match_id = v.match_id
			  AND l.xuid     = v.xuid
			  AND l.time_ms  = v.time_ms
		)
		GROUP BY v.match_id
		ORDER BY rows_to_insert DESC
		LIMIT 10`)
	if err != nil {
		log.Fatalf("sample query: %v", err)
	}
	for rows.Next() {
		var mid string
		var n, xc int
		_ = rows.Scan(&mid, &n, &xc)
		fmt.Printf("  %s  rows=%d  xuids=%d\n", mid[:8], n, xc)
	}
	rows.Close()

	if *dryRun {
		fmt.Println("\n[DRY RUN] Aucune modification. Relancer avec --apply pour exécuter.")
		return
	}

	// 4) APPLY — transaction unique, rollback automatique sur erreur
	fmt.Println("\n[APPLY] Exécution du INSERT … NOT EXISTS dans une transaction…")
	t0 := time.Now()
	tx, err := local.Begin()
	if err != nil {
		log.Fatalf("BEGIN: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.Exec(`
		INSERT INTO weapon_kills
		SELECT v.* FROM vps.weapon_kills v
		WHERE NOT EXISTS (
			SELECT 1 FROM weapon_kills l
			WHERE l.match_id = v.match_id
			  AND l.xuid     = v.xuid
			  AND l.time_ms  = v.time_ms
		)`)
	if err != nil {
		log.Fatalf("INSERT failed (rollback automatique): %v", err)
	}
	inserted, _ := res.RowsAffected()
	fmt.Printf("  rows insérées (RowsAffected) : %d\n", inserted)

	if int(inserted) != deltaRows {
		log.Fatalf("MISMATCH delta vs inserted (%d vs %d) → ROLLBACK", deltaRows, inserted)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("COMMIT: %v", err)
	}
	fmt.Printf("  COMMIT OK (durée: %s)\n", time.Since(t0))

	// 5) VÉRIFICATION POST-MERGE
	var localAfter, localMatchesAfter int
	_ = local.QueryRow(`SELECT COUNT(*) FROM weapon_kills`).Scan(&localAfter)
	_ = local.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM weapon_kills`).Scan(&localMatchesAfter)
	fmt.Println("\n[ÉTAT APRÈS]")
	fmt.Printf("  Local weapon_kills   : %d lignes (+%d) / %d matchs (+%d)\n",
		localAfter, localAfter-localBefore, localMatchesAfter, localMatchesAfter-localMatchesBefore)

	// 6) Vérification du match canari 8faf5c41
	const canary = "8faf5c41-0af2-4102-b687-60b297afc1c7"
	var canaryRows int
	_ = local.QueryRow(`SELECT COUNT(*) FROM weapon_kills WHERE match_id = ?`, canary).Scan(&canaryRows)
	fmt.Printf("  Match canari %s : %d lignes (attendu : 90)\n", canary[:8], canaryRows)
}
