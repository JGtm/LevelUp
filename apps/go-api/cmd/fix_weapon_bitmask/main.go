// fix_weapon_bitmask — réaligne match_registry.backfill_completed avec la
// réalité de weapon_kills. Idempotent et non-destructif :
//   - retire bit21 sur les matchs sans rows (faux positifs)
//   - pose bit21 sur les matchs avec rows (cohérence post-merge)
//   - ne touche PAS aux autres bits (events, medals, kvp, etc.)
//
// Usage:
//
//	go run . --local <path> --dry-run
//	go run . --local <path> --apply
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

const (
	bitWeaponKills       = 1 << 21 // 2097152 — MBitWeaponKills
	bitWeaponKillsNoFilm = 1 << 22 // 4194304 — MBitWeaponKillsNoFilm
)

func main() {
	localPath := flag.String("local", "", "chemin vers la DB locale (R/W)")
	dryRun := flag.Bool("dry-run", true, "compte uniquement, pas de modification")
	apply := flag.Bool("apply", false, "applique réellement les UPDATE")
	flag.Parse()

	if *localPath == "" {
		log.Fatal("--local est obligatoire")
	}
	if *apply {
		*dryRun = false
	}
	mode := "DRY RUN"
	if !*dryRun {
		mode = "APPLY"
	}
	if _, err := os.Stat(*localPath); err != nil {
		log.Fatalf("Local DB introuvable: %v", err)
	}

	fmt.Printf("=== fix_weapon_bitmask — mode=%s ===\n", mode)
	fmt.Printf("Local : %s\n\n", *localPath)

	db, err := sql.Open("duckdb", *localPath)
	if err != nil {
		log.Fatalf("ouverture: %v", err)
	}
	defer db.Close()

	// ÉTAT AVANT
	var totalMatches, withWK, bit21Set, bit21SetEmpty, bit21Missing int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&totalMatches)
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`).Scan(&withWK)
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry
		WHERE (COALESCE(backfill_completed, 0) & ?) != 0`, bitWeaponKills).Scan(&bit21Set)
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE (COALESCE(mr.backfill_completed, 0) & ?) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`, bitWeaponKills).Scan(&bit21SetEmpty)
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE (COALESCE(mr.backfill_completed, 0) & ?) = 0
		  AND EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`, bitWeaponKills).Scan(&bit21Missing)

	fmt.Println("[ÉTAT AVANT]")
	fmt.Printf("  Total matchs                              : %d\n", totalMatches)
	fmt.Printf("  Avec weapon_kills (rows présentes)        : %d\n", withWK)
	fmt.Printf("  bit21 set                                 : %d\n", bit21Set)
	fmt.Printf("  bit21 set MAIS table vide (faux positifs) : %d  ← À NETTOYER\n", bit21SetEmpty)
	fmt.Printf("  bit21 NON set MAIS table peuplée          : %d  ← À POSER\n", bit21Missing)

	if bit21SetEmpty == 0 && bit21Missing == 0 {
		fmt.Println("\n[OK] Bitmask déjà cohérent.")
		return
	}

	if *dryRun {
		fmt.Println("\n[DRY RUN] Aucune modification. Relancer avec --apply.")
		return
	}

	// APPLY — transaction unique
	fmt.Println("\n[APPLY] Réalignement bit21 dans une transaction…")
	t0 := time.Now()
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("BEGIN: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1) Retire bit21 sur les matchs sans rows (faux positifs).
	res1, err := tx.Exec(`
		UPDATE match_registry mr
		SET backfill_completed = COALESCE(backfill_completed, 0) & ~CAST(? AS BIGINT)
		WHERE (COALESCE(backfill_completed, 0) & ?) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`,
		bitWeaponKills, bitWeaponKills)
	if err != nil {
		log.Fatalf("UPDATE retrait bit21: %v", err)
	}
	cleared, _ := res1.RowsAffected()
	fmt.Printf("  bit21 retiré (faux positifs) : %d\n", cleared)
	if int(cleared) != bit21SetEmpty {
		log.Fatalf("MISMATCH cleared (%d) vs avant (%d) → ROLLBACK", cleared, bit21SetEmpty)
	}

	// 2) Pose bit21 sur les matchs avec rows mais sans bit (cohérence post-merge).
	res2, err := tx.Exec(`
		UPDATE match_registry mr
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE (COALESCE(backfill_completed, 0) & ?) = 0
		  AND EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`,
		bitWeaponKills, bitWeaponKills)
	if err != nil {
		log.Fatalf("UPDATE pose bit21: %v", err)
	}
	added, _ := res2.RowsAffected()
	fmt.Printf("  bit21 posé   (cohérence)     : %d\n", added)
	if int(added) != bit21Missing {
		log.Fatalf("MISMATCH added (%d) vs avant (%d) → ROLLBACK", added, bit21Missing)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("COMMIT: %v", err)
	}
	fmt.Printf("  COMMIT OK (durée: %s)\n", time.Since(t0))

	// VÉRIFICATION POST
	var bit21SetEmpty2, bit21Missing2 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE (COALESCE(mr.backfill_completed, 0) & ?) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`, bitWeaponKills).Scan(&bit21SetEmpty2)
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry mr
		WHERE (COALESCE(mr.backfill_completed, 0) & ?) = 0
		  AND EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)`, bitWeaponKills).Scan(&bit21Missing2)
	fmt.Println("\n[ÉTAT APRÈS]")
	fmt.Printf("  bit21 set MAIS table vide : %d (attendu 0)\n", bit21SetEmpty2)
	fmt.Printf("  bit21 NON set table peuplée : %d (attendu 0)\n", bit21Missing2)
}
