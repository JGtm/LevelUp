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

	"levelup/go-api/internal/analysis"
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

// ResetLyingBits détecte puis, si !dryRun, clear les bits menteurs.
// `db` doit pointer le shared : RO pour le dry-run, writer exclusif pour
// l'exécution. Idempotent : un second appel après exécution retourne des
// compteurs nuls.
//
// Écriture ROW-BY-ROW par match_id (ADR 0019/0026, anti-ART #23046) : cette fonction
// tourne IN-PROCESS (action admin data-quality) — un bulk UPDATE multi-row nu sur
// match_registry (même une mutation in-place, sans ON CONFLICT) touche N entrées de
// l'index PK en un statement = déclencheur ART direct. On SELECT donc les match_ids
// cibles puis on clear chacun par `WHERE match_id = ?`
// (garde-fou : no_art_patterns_test.go::TestNoBulkMultiRowUpdateOnCriticalTables).
func ResetLyingBits(ctx context.Context, db *sql.DB, dryRun bool) (LyingBitsResetResult, error) {
	out := LyingBitsResetResult{DryRun: dryRun}
	if db == nil {
		return out, fmt.Errorf("reset lying bits: shared DB nil")
	}

	// Détection (toujours) : la LISTE des match_ids par catégorie (mêmes prédicats
	// que CountDataQuality). Le compteur = len(liste).
	eventsIDs, err := selectLyingMatchIDs(ctx, db, fmt.Sprintf(`
		SELECT r.match_id FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, dqBitEvents))
	if err != nil {
		return out, fmt.Errorf("detect lying events: %w", err)
	}
	out.EventsBitsCleared = len(eventsIDs)

	// La PREUVE du detail des armes change de table selon le titre (cf.
	// analysis.WeaponEvidenceTable) : on sonde celle que la base CONTIENT. Aucune table
	// presente = aucun bit menteur a nettoyer, et surtout pas une requete vouee a l erreur.
	var weaponsIDs []string
	if evidence := analysis.WeaponEvidenceTable(ctx, db); evidence != "" {
		weaponsIDs, err = selectLyingMatchIDs(ctx, db, fmt.Sprintf(`
		SELECT r.match_id FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM %s w WHERE w.match_id = r.match_id)
	`, dqBitWeaponKills, evidence))
		if err != nil {
			return out, fmt.Errorf("detect lying weapons: %w", err)
		}
	}
	out.WeaponsBitsCleared = len(weaponsIDs)

	eventsLoadedIDs, err := selectLyingMatchIDs(ctx, db, `
		SELECT r.match_id FROM match_registry r
		WHERE COALESCE(r.events_loaded, FALSE) = TRUE
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`)
	if err != nil {
		return out, fmt.Errorf("detect lying events_loaded: %w", err)
	}
	out.EventsLoadedCleared = len(eventsLoadedIDs)

	if dryRun {
		return out, nil
	}

	// Exécution : clear row-by-row par match_id (le bit est un entier constant lié
	// en `?`, jamais un bulk set-based).
	for _, id := range eventsIDs {
		if _, err := db.ExecContext(ctx,
			`UPDATE match_registry SET backfill_completed = COALESCE(backfill_completed, 0) & ~? WHERE match_id = ?`,
			dqBitEvents, id); err != nil {
			return out, fmt.Errorf("clear lying events %s: %w", id, err)
		}
	}
	for _, id := range weaponsIDs {
		if _, err := db.ExecContext(ctx,
			`UPDATE match_registry SET backfill_completed = COALESCE(backfill_completed, 0) & ~? WHERE match_id = ?`,
			dqBitWeaponKills, id); err != nil {
			return out, fmt.Errorf("clear lying weapons %s: %w", id, err)
		}
	}
	for _, id := range eventsLoadedIDs {
		if _, err := db.ExecContext(ctx,
			`UPDATE match_registry SET events_loaded = FALSE WHERE match_id = ?`, id); err != nil {
			return out, fmt.Errorf("clear lying events_loaded %s: %w", id, err)
		}
	}

	return out, nil
}

// selectLyingMatchIDs exécute une requête de détection (SELECT match_id) et
// retourne la liste des match_ids. Utilisé par ResetLyingBits pour piloter le
// clear row-by-row.
func selectLyingMatchIDs(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
