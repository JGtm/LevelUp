// Package ops — lying_bits_reset.go : reset des bits menteurs du ledger
// match_registry.backfill_completed (et du flag events_loaded).
//
// Un bit « menteur » est posé (= donnée réputée chargée) alors que la table
// correspondante est vide. Le heal filtre sur ces flags : tant que le bit est
// set, le match est skip → il ne reçoit jamais ses events/weapons. Le clear
// débloque la convergence au prochain sync delta.
//
// Extrait du chantier 4 de cmd/repair_data_consistency pour être appelable
// in-process depuis l'action admin (writer shared sérialisé) ET depuis le CLI.
package ops

import (
	"context"
	"database/sql"
	"fmt"
)

// LyingBitsResetResult — compteurs des matchs concernés par catégorie. En
// dry-run, les *Cleared comptent ce qui SERAIT corrigé (COUNT seul) ; en
// exécution, ce qui A ÉTÉ corrigé. Le WHERE de détection et d'UPDATE est
// identique et le writer est exclusif, donc les deux coïncident.
type LyingBitsResetResult struct {
	DryRun              bool
	EventsBitsCleared   int
	WeaponsBitsCleared  int
	EventsLoadedCleared int
}

// Total agrège les trois catégories.
func (r LyingBitsResetResult) Total() int {
	return r.EventsBitsCleared + r.WeaponsBitsCleared + r.EventsLoadedCleared
}

// ResetLyingBits détecte (COUNT) puis, si !dryRun, clear les bits menteurs.
// `db` doit pointer le shared : RO pour le dry-run, writer exclusif pour
// l'exécution. Idempotent : un second appel après exécution retourne des
// compteurs nuls. Les UPDATE ne sont PAS un ON CONFLICT (mutation in-place
// d'un INTEGER/BOOLEAN existant — pas de risque ART, pas de PK touchée).
func ResetLyingBits(ctx context.Context, db *sql.DB, dryRun bool) (LyingBitsResetResult, error) {
	out := LyingBitsResetResult{DryRun: dryRun}
	if db == nil {
		return out, fmt.Errorf("reset lying bits: shared DB nil")
	}

	// Détection (toujours) : un COUNT par catégorie, mêmes prédicats que les
	// UPDATE ci-dessous et que CountDataQuality (dqBitEvents/dqBitWeaponKills).
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, dqBitEvents)).Scan(&out.EventsBitsCleared); err != nil {
		return out, fmt.Errorf("count lying events: %w", err)
	}
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id)
	`, dqBitWeaponKills)).Scan(&out.WeaponsBitsCleared); err != nil {
		return out, fmt.Errorf("count lying weapons: %w", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM match_registry r
		WHERE COALESCE(r.events_loaded, FALSE) = TRUE
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`).Scan(&out.EventsLoadedCleared); err != nil {
		return out, fmt.Errorf("count lying events_loaded: %w", err)
	}

	if dryRun {
		return out, nil
	}

	// Clear MBitEvents.
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) & ~%d
		WHERE (COALESCE(backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = match_registry.match_id)
	`, dqBitEvents, dqBitEvents)); err != nil {
		return out, fmt.Errorf("clear lying events: %w", err)
	}
	// Clear MBitWeaponKills.
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) & ~%d
		WHERE (COALESCE(backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = match_registry.match_id)
	`, dqBitWeaponKills, dqBitWeaponKills)); err != nil {
		return out, fmt.Errorf("clear lying weapons: %w", err)
	}
	// Clear events_loaded booléen menteur.
	if _, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET events_loaded = FALSE
		WHERE COALESCE(events_loaded, FALSE) = TRUE
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = match_registry.match_id)
	`); err != nil {
		return out, fmt.Errorf("clear lying events_loaded: %w", err)
	}

	return out, nil
}
