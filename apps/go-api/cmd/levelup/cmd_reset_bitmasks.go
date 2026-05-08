// cmd_reset_bitmasks.go — Sous-commande `levelup reset-bitmasks`.
//
// Phase 4 du plan PLAN_BITMASKS_AUDIT_FIX (mai 2026). One-shot rétroactif
// pour positionner les bits skill/participants/PVE sur les matchs existants
// dont la donnée est déjà présente — évite que `levelup backfill --skill`
// re-traite tout l'historique au premier run après le déploiement Phase 2.
//
// Idempotent. Read-only par défaut (--dry-run obligatoire pour voir l'impact
// avant --apply).
//
// Usage :
//
//	levelup reset-bitmasks --dry-run
//	levelup reset-bitmasks --apply
package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"

	"database/sql"
)

func runResetBitmasks(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("reset-bitmasks", flag.ExitOnError)
	apply := fs.Bool("apply", false, "Appliquer les UPDATE (sinon dry-run, lecture seule)")
	dryRun := fs.Bool("dry-run", false, "Afficher les chiffres sans modifier la DB (équivalent à --apply absent)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *apply && *dryRun {
		return fmt.Errorf("--apply et --dry-run sont mutuellement exclusifs")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	pvePath := filepath.Join(cfg.RepoRoot, "data", "titles", titlePkg.DefaultSlug, "warehouse", "shared_pve.duckdb")

	mode := "DRY-RUN"
	if *apply {
		mode = "APPLY"
	}
	fmt.Printf("=== levelup reset-bitmasks (%s) ===\n", mode)
	fmt.Printf("shared : %s\n\n", sharedPath)

	if !*apply {
		// Dry-run : ouvrir en read-only.
		shared, err := duckdbpkg.OpenReadOnly(sharedPath)
		if err != nil {
			return fmt.Errorf("open shared RO: %w", err)
		}
		defer shared.Close()
		return previewReset(shared.SQLDb(), pvePath)
	}

	// Apply : ouvrir en RW (échoue si server tient le lock).
	shared, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (server LevelUp tourne ?): %w", err)
	}
	defer shared.Close()
	return applyReset(shared.SQLDb(), pvePath)
}

// previewReset compte combien de rows seraient mises à jour par chaque
// UPDATE, sans toucher la DB.
func previewReset(db *sql.DB, pvePath string) error {
	// Skill — participants
	var skillParts int
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM match_participants
		WHERE team_mmr IS NOT NULL
		  AND (COALESCE(backfill_bits, 0) & ?) != ?
	`, skillBitsForReset, skillBitsForReset).Scan(&skillParts)
	fmt.Printf("[skill]        match_participants à updater (PBit skill) : %d\n", skillParts)

	// Skill — registry
	var skillReg int
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM match_registry mr
		WHERE EXISTS (
		    SELECT 1 FROM match_participants mp
		    WHERE mp.match_id = mr.match_id AND mp.team_mmr IS NOT NULL
		)
		AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
	`, backfillFlagSkillForReset).Scan(&skillReg)
	fmt.Printf("[skill]        match_registry  à updater (BackfillFlags[skill]) : %d\n", skillReg)

	// Participants
	var partReg int
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM match_registry mr
		WHERE EXISTS (SELECT 1 FROM match_participants WHERE match_id = mr.match_id)
		  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
	`, backfillFlagParticipantsForReset).Scan(&partReg)
	fmt.Printf("[participants] match_registry à updater (BackfillFlags[participants]) : %d\n", partReg)

	// PVE — via ATTACH cross-DB
	pveReg := 0
	_, attErr := db.Exec(fmt.Sprintf(`ATTACH '%s' AS pve_db (READ_ONLY)`, pvePath))
	if attErr == nil {
		_ = db.QueryRow(`
			SELECT COUNT(*) FROM match_registry mr
			WHERE mr.is_firefight = TRUE
			  AND EXISTS (
			    SELECT 1 FROM pve_db.pve_match_stats WHERE match_id = mr.match_id
			  )
			  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
		`, mBitPVEStatsForReset).Scan(&pveReg)
		_, _ = db.Exec(`DETACH pve_db`)
	}
	fmt.Printf("[pve]          match_registry à updater (MBitPVEStats) : %d\n", pveReg)

	fmt.Println("\nLance `levelup reset-bitmasks --apply` pour appliquer ces UPDATEs.")
	return nil
}

// applyReset exécute les UPDATEs idempotents.
func applyReset(db *sql.DB, pvePath string) error {
	// Skill participants
	r1, err := db.Exec(`
		UPDATE match_participants
		SET backfill_bits = COALESCE(backfill_bits, 0) | ?
		WHERE team_mmr IS NOT NULL
		  AND (COALESCE(backfill_bits, 0) & ?) != ?
	`, skillBitsForReset, skillBitsForReset, skillBitsForReset)
	if err != nil {
		return fmt.Errorf("UPDATE skill participants: %w", err)
	}
	n1, _ := r1.RowsAffected()
	fmt.Printf("[skill] participants updated : %d\n", n1)

	// Skill registry
	r2, err := db.Exec(`
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE EXISTS (
		    SELECT 1 FROM match_participants mp
		    WHERE mp.match_id = match_registry.match_id AND mp.team_mmr IS NOT NULL
		)
		AND (COALESCE(backfill_completed, 0) & ?) = 0
	`, backfillFlagSkillForReset, backfillFlagSkillForReset)
	if err != nil {
		return fmt.Errorf("UPDATE skill registry: %w", err)
	}
	n2, _ := r2.RowsAffected()
	fmt.Printf("[skill] registry updated : %d\n", n2)

	// Participants registry
	r3, err := db.Exec(`
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE EXISTS (SELECT 1 FROM match_participants WHERE match_id = match_registry.match_id)
		  AND (COALESCE(backfill_completed, 0) & ?) = 0
	`, backfillFlagParticipantsForReset, backfillFlagParticipantsForReset)
	if err != nil {
		return fmt.Errorf("UPDATE participants registry: %w", err)
	}
	n3, _ := r3.RowsAffected()
	fmt.Printf("[participants] registry updated : %d\n", n3)

	// PVE — uniquement si shared_pve.duckdb accessible
	_, attErr := db.Exec(fmt.Sprintf(`ATTACH '%s' AS pve_db`, pvePath))
	if attErr == nil {
		r4, err := db.Exec(`
			UPDATE match_registry
			SET backfill_completed = COALESCE(backfill_completed, 0) | ?
			WHERE is_firefight = TRUE
			  AND EXISTS (
			    SELECT 1 FROM pve_db.pve_match_stats WHERE match_id = match_registry.match_id
			  )
			  AND (COALESCE(backfill_completed, 0) & ?) = 0
		`, mBitPVEStatsForReset, mBitPVEStatsForReset)
		if err != nil {
			return fmt.Errorf("UPDATE pve registry: %w", err)
		}
		n4, _ := r4.RowsAffected()
		fmt.Printf("[pve] registry updated : %d\n", n4)
		_, _ = db.Exec(`DETACH pve_db`)
	} else {
		fmt.Printf("[pve] DB inaccessible — skip (%v)\n", attErr)
	}

	fmt.Println("\nReset terminé. Idempotent — ré-exécution sans dommage.")
	return nil
}

// Constantes locales (dupliquées depuis internal/sync — évite l'import lourd
// pour un one-shot CLI).
const (
	pBitTeamMMR                      = 1 << 0
	pBitEnemyMMR                     = 1 << 1
	pBitKillsExp                     = 1 << 2
	pBitDeathsExp                    = 1 << 3
	skillBitsForReset                = pBitTeamMMR | pBitEnemyMMR | pBitKillsExp | pBitDeathsExp
	backfillFlagSkillForReset        = 1 << 2  // 4
	backfillFlagParticipantsForReset = 1 << 9  // 512
	mBitPVEStatsForReset             = 1 << 20 // 1048576
)
