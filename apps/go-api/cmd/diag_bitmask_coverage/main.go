// cmd/diag_bitmask_coverage — Sonde read-only sur la prod pour mesurer
// l'impact des dead writes du système de bitmask backfill.
//
// Mesure pour chaque filtre potentiellement cassé combien de matchs/participants
// ont la donnée déjà présente mais le bit non positionné. Ces chiffres
// constituent la baseline avant Phase 2 (fix WRITE) et Phase 4 (reset
// rétroactif) du plan PLAN_BITMASKS_AUDIT_FIX.md.
//
// Usage :
//
//	go run ./cmd/diag_bitmask_coverage
//
// Read-only sur shared_matches_v2.duckdb + shared_pve.duckdb. Skip
// gracieusement si shared_pve absent (multi-titres).
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

func main() {
	dataRoot := "../../data/titles/halo_infinite"
	sharedPath := filepath.Join(dataRoot, "warehouse", "shared_matches_v2.duckdb")
	pvePath := filepath.Join(dataRoot, "warehouse", "shared_pve.duckdb")

	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable : %s", sharedPath)
	}

	shared := openRO(sharedPath)
	defer shared.Close()

	fmt.Println("=== diag_bitmask_coverage ===")
	fmt.Println()

	// Totaux pour mise en perspective
	var totalMatches, totalParticipants, totalFirefights int
	_ = shared.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&totalMatches)
	_ = shared.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&totalParticipants)
	_ = shared.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE is_firefight = TRUE`).Scan(&totalFirefights)
	fmt.Printf("totaux : %d matchs (%d firefights), %d participants\n\n",
		totalMatches, totalFirefights, totalParticipants)

	// ── Skill — PBitTeamMMR (1<<0) sur match_participants ──
	var skillPBitDead int
	_ = shared.QueryRow(`
		SELECT COUNT(*) FROM match_participants
		WHERE team_mmr IS NOT NULL
		  AND (COALESCE(backfill_bits, 0) & 1) = 0
	`).Scan(&skillPBitDead)

	// Skill — bit 4 (BackfillFlags["skill"]) sur match_registry
	var skillMBitDead int
	_ = shared.QueryRow(`
		SELECT COUNT(*) FROM match_registry mr
		WHERE EXISTS (
		    SELECT 1 FROM match_participants mp
		    WHERE mp.match_id = mr.match_id AND mp.team_mmr IS NOT NULL
		)
		AND (COALESCE(mr.backfill_completed, 0) & 4) = 0
	`).Scan(&skillMBitDead)

	fmt.Printf("[skill] participants avec team_mmr rempli mais PBitTeamMMR=0 : %d / %d\n",
		skillPBitDead, totalParticipants)
	fmt.Printf("        matchs avec ≥1 team_mmr rempli mais BackfillFlags[skill]=0 (mr) : %d / %d\n\n",
		skillMBitDead, totalMatches)

	// ── Participants — bit 1<<9 = 512 ──
	var participantsBitDead int
	_ = shared.QueryRow(`
		SELECT COUNT(*) FROM match_registry mr
		WHERE EXISTS (SELECT 1 FROM match_participants WHERE match_id = mr.match_id)
		  AND (COALESCE(mr.backfill_completed, 0) & 512) = 0
	`).Scan(&participantsBitDead)
	fmt.Printf("[participants] matchs avec ≥1 participant inséré mais BackfillFlags[participants]=0 : %d / %d\n\n",
		participantsBitDead, totalMatches)

	// ── PVE stats — MBitPVEStats (1<<20 = 1048576) ──
	if _, err := os.Stat(pvePath); err == nil {
		pve, err := openROOptional(pvePath)
		if err != nil {
			fmt.Printf("[pve] DB lockée — skip (%v)\n\n", err)
		} else {
			defer pve.Close()
			// Compter via ATTACH cross-DB
			var pveBitDead int
			_, _ = shared.Exec(fmt.Sprintf(`ATTACH '%s' AS pve_db (READ_ONLY)`, pvePath))
			_ = shared.QueryRow(`
				SELECT COUNT(*) FROM match_registry mr
				WHERE mr.is_firefight = TRUE
				  AND EXISTS (
				    SELECT 1 FROM pve_db.pve_match_stats pms
				    WHERE pms.match_id = mr.match_id
				  )
				  AND (COALESCE(mr.backfill_completed, 0) & 1048576) = 0
			`).Scan(&pveBitDead)
			fmt.Printf("[pve] firefights avec ≥1 ligne pve_match_stats mais MBitPVEStats=0 : %d / %d\n\n",
				pveBitDead, totalFirefights)
			_, _ = shared.Exec(`DETACH pve_db`)
		}
	} else {
		fmt.Printf("[pve] shared_pve.duckdb absent — skip\n\n")
	}

	// ── Bits réellement écrits (sanity check) ──
	fmt.Println("=== bits effectivement positionnés (sanity) ===")
	for _, b := range []struct {
		label string
		bit   int64
	}{
		{"MBitEvents (1<<16)", 1 << 16},
		{"MBitKillerVictim (1<<19)", 1 << 19},
		{"MBitWeaponKills (1<<21)", 1 << 21},
		{"MBitFilmAbsent (1<<22)", 1 << 22},
		{"BackfillFlags[skill]=4 (Phase 2.a fixera)", 4},
		{"BackfillFlags[participants]=512 (Phase 2.b fixera)", 512},
		{"MBitPVEStats=1048576 (Phase 2.c fixera)", 1048576},
	} {
		var n int
		_ = shared.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE (COALESCE(backfill_completed, 0) & ?) != 0`, b.bit).Scan(&n)
		fmt.Printf("  %-50s : %d / %d matchs\n", b.label, n, totalMatches)
	}

	var pbitTeamSet int
	_ = shared.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE (COALESCE(backfill_bits, 0) & 1) != 0`).Scan(&pbitTeamSet)
	fmt.Printf("  %-50s : %d / %d participants\n", "PBitTeamMMR=1 (Phase 2.a fixera)", pbitTeamSet, totalParticipants)
}

func openRO(path string) *sql.DB {
	db, err := openROOptional(path)
	if err != nil {
		log.Fatalf("openRO(%s): %v", path, err)
	}
	return db
}

func openROOptional(path string) (*sql.DB, error) {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
