// Package ops — media_associate.go : algorithme d'association média↔match
// SANS ATTACH cross-DB.
//
// Contexte : l'ancienne implémentation faisait `ATTACH 'shared_matches.duckdb'
// AS shared_matches (READ_ONLY)` sur la connexion RW de shared_social.duckdb
// pour réaliser une jointure SQL directe. Ce ATTACH écrit dans le WAL une
// entrée qui n'est PAS rejouable au prochain boot (bug DuckDB #7659 :
// "WAL Replay fails when attach alias changes"). Conséquence : si le process
// est killé brutalement (Air rebuild Windows), le boot suivant échoue avec
// "INTERNAL Error: Calling DatabaseManager::GetDefaultDatabase with no
// default database set" → SharedSocial=nil pour tous les joueurs → rail média
// vide partout.
//
// Nouvelle approche (ADR 0016 conforme) :
//   1. Charger les fenêtres temporelles des matchs depuis shared_matches_v2
//      sur une connexion RO indépendante (DuckDB autorise N readers RO).
//   2. Charger les media_files candidats (capture_start_utc not null, pas déjà
//      associés) depuis shared_social via la connexion existante.
//   3. Calculer les associations en Go (computeAssociations) — algorithme pur,
//      isolable, testable.
//   4. Bulk-insérer les associations dans shared_social via la connexion
//      existante (INSERT OR IGNORE).
//
// Plus aucun ATTACH → plus aucune corruption WAL possible.

package ops

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// matchTimeWindow représente la fenêtre temporelle d'un match. Tous les
// timestamps sont en UTC.
type matchTimeWindow struct {
	MatchID  string
	StartUTC time.Time
	EndUTC   time.Time
}

// unassocMediaRow représente un média à associer.
type unassocMediaRow struct {
	MediaFileID     int64
	CaptureStartUTC time.Time
}

// mediaMatchAssoc est le résultat de l'algorithme d'association : un média est
// rattaché à UN match (le meilleur selon le scoring). DeltaSeconds est la
// distance au DÉBUT du match (cohérent avec le SQL d'origine).
type mediaMatchAssoc struct {
	MediaFileID  int64
	MatchID      string
	DeltaSeconds int
}

// loadMatchTimeWindows lit les fenêtres temporelles des matchs depuis
// shared_matches_v2.duckdb sans ATTACH cross-DB.
//
// Acquiert le handle via un RecoveringReader (platform/duckdb) : il réutilise le
// handle process-wide (RW ou RO) s'il existe et ne ré-ouvre en RO que si AUCUN
// handle n'est en cache. C'est crucial — l'ancien fallback
// `sql.Open(... access_mode=read_only)` forçait une ouverture RO d'un fichier
// potentiellement tenu en RW dans le même process (sync / reindex concurrent),
// ce qui échoue avec "file is being used by another process" / "different
// configuration". Conséquence observée (2026-06-03) : l'association d'un upload
// concurrent d'un reindex échouait silencieusement → médias sans match. Même
// classe d'incident que l'OpenReadOnly forcé corrigé par cette famille
// d'ouverture (ADR-0016 / discovery sync V2 2026-06-01).
//
// RecoveringReader plutôt qu'un instantané nu (2026-08-25) : le handle emprunté
// au cache est NON POSSÉDÉ, et le B-swap RO→RW déclenché par un writer sync le
// ferme PENDANT la requête ou l'itération → « sql: database is closed »
// (~2,7 ERROR/j en prod : `ops.IndexMedia: association média↔match échouée`,
// err `load match windows: query match_registry`). Do ré-ouvre et rejoue la
// lecture une fois — cf. read_recovery.go.
//
// Filtre WHERE start_time_utc/end_time_utc IS NOT NULL pour garantir des
// timestamps valides. Le fallback `start_time AT TIME ZONE 'UTC'` couvre les
// rares matchs pré-migration add_start_time_utc_to_match_registry.
func loadMatchTimeWindows(ctx context.Context, sharedMatchesPath string) ([]matchTimeWindow, error) {
	reader, err := platform_duckdb.OpenRecoveringReader(sharedMatchesPath)
	if err != nil {
		return nil, fmt.Errorf("ouverture shared_matches pour lecture: %w", err)
	}
	defer reader.Close()

	q := `
		SELECT
			match_id,
			` + analysis.SQLStartTimeCanonical("") + ` AS start_utc,
			COALESCE(end_time_utc,   end_time   AT TIME ZONE 'UTC') AS end_utc
		FROM match_registry
		WHERE ` + analysis.SQLStartTimeCanonical("") + ` IS NOT NULL
		  AND COALESCE(end_time_utc,   end_time   AT TIME ZONE 'UTC') IS NOT NULL
	`

	var windows []matchTimeWindow
	if err := reader.Do(ctx, func(db *sql.DB) error {
		// Rejouable (contrat de Do) : le retry post-ré-ouverture repart d'une
		// liste vide, sinon il dupliquerait les fenêtres déjà scannées.
		windows = nil

		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return fmt.Errorf("query match_registry: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var w matchTimeWindow
			if err := rows.Scan(&w.MatchID, &w.StartUTC, &w.EndUTC); err != nil {
				return fmt.Errorf("scan match_registry row: %w", err)
			}
			windows = append(windows, w)
		}
		return rows.Err()
	}); err != nil {
		return nil, err
	}
	return windows, nil
}

// loadUnassociatedMedia retourne les media_files dont capture_start_utc est
// non-null ET qui ne sont pas déjà dans media_match_associations.
//
// Les médias SUPPRIMÉS (item 3.1) sont exclus : les associer produirait un
// INSERT dans media_match_associations_history, table append-only — un event
// écrit là n'est jamais retirable, et il porterait sur un média qui n'existe
// plus sur le disque.
func loadUnassociatedMedia(ctx context.Context, db *sql.DB) ([]unassocMediaRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT mf.id, mf.capture_start_utc
		FROM media_files mf
		WHERE mf.capture_start_utc IS NOT NULL
		  AND `+platform_duckdb.MediaVisiblePredicate("mf")+`
		  AND mf.id NOT IN (SELECT media_file_id FROM media_match_associations_latest)
	`)
	if err != nil {
		return nil, fmt.Errorf("query media_files candidates: %w", err)
	}
	defer rows.Close()

	var candidates []unassocMediaRow
	for rows.Next() {
		var c unassocMediaRow
		if err := rows.Scan(&c.MediaFileID, &c.CaptureStartUTC); err != nil {
			return nil, fmt.Errorf("scan media_files row: %w", err)
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// computeAssociations est l'algorithme PUR d'association média→match. Pour
// chaque média, trouve le meilleur match dans la fenêtre [start - buffer, end +
// buffer]. Préserve la sémantique exacte du SQL d'origine (ROW_NUMBER OVER
// PARTITION BY mf.id) :
//
//  1. PRIORITÉ : match qui CONTIENT capture_start_utc (sans buffer) bat un
//     match dont seul le buffer englobe la capture. Évite qu'un replay enregistré
//     1 min après un match court ne soit attribué au match précédent.
//  2. TIE-BREAK : distance au CENTRE du match (asc). Un replay enregistré à
//     la fin d'un match a un delta naturel ~match_duration/2 par rapport au
//     début ; trier par "delta vs start_time" donnerait des faux positifs sur
//     le match suivant.
//  3. TIE-BREAK FINAL : match_id alphabétique (déterministe).
//
// DeltaSeconds renvoyé = distance au DÉBUT du match (positif, en secondes) —
// cohérent avec la sémantique stockée historiquement dans
// media_match_associations.delta_seconds.
//
// Pure function : aucun side-effect, aucun I/O. Testable en isolation.
func computeAssociations(media []unassocMediaRow, matches []matchTimeWindow, bufferMin int) []mediaMatchAssoc {
	if bufferMin <= 0 {
		bufferMin = 2
	}
	buffer := time.Duration(bufferMin) * time.Minute

	out := make([]mediaMatchAssoc, 0, len(media))
	for _, m := range media {
		bestIdx := -1
		bestContained := false
		bestDistCenter := time.Duration(1<<62 - 1)
		bestMatchID := ""

		for i, w := range matches {
			if m.CaptureStartUTC.Before(w.StartUTC.Add(-buffer)) {
				continue
			}
			if m.CaptureStartUTC.After(w.EndUTC.Add(buffer)) {
				continue
			}

			contained := !m.CaptureStartUTC.Before(w.StartUTC) && !m.CaptureStartUTC.After(w.EndUTC)
			center := w.StartUTC.Add(w.EndUTC.Sub(w.StartUTC) / 2)
			distCenter := absDuration(m.CaptureStartUTC.Sub(center))

			if isBetterMatch(bestIdx, contained, bestContained, distCenter, bestDistCenter, w.MatchID, bestMatchID) {
				bestIdx = i
				bestContained = contained
				bestDistCenter = distCenter
				bestMatchID = w.MatchID
			}
		}

		if bestIdx == -1 {
			continue
		}
		best := matches[bestIdx]
		deltaStart := int(absDuration(m.CaptureStartUTC.Sub(best.StartUTC)).Seconds())
		out = append(out, mediaMatchAssoc{
			MediaFileID:  m.MediaFileID,
			MatchID:      best.MatchID,
			DeltaSeconds: deltaStart,
		})
	}
	return out
}

// isBetterMatch encapsule la règle de tri (contained > distCenter asc > matchID asc).
// Premier paramètre : -1 si aucun "best" encore. Extrait pour lisibilité +
// testabilité fine si besoin.
func isBetterMatch(bestIdx int, contained, bestContained bool, distCenter, bestDistCenter time.Duration, matchID, bestMatchID string) bool {
	if bestIdx == -1 {
		return true
	}
	if contained != bestContained {
		return contained // true (contained) bat false (buffer only)
	}
	if distCenter != bestDistCenter {
		return distCenter < bestDistCenter
	}
	return matchID < bestMatchID
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// bulkInsertAssociations insère les associations calculées dans
// media_match_associations via une transaction. INSERT OR IGNORE protège
// contre les races (un autre IndexMedia concurrent pourrait avoir inséré
// entre loadUnassociatedMedia et l'insert).
//
// Pas d'ATTACH — la connexion db est celle de shared_social (RW).
func bulkInsertAssociations(ctx context.Context, db *sql.DB, assocs []mediaMatchAssoc) (int, error) {
	if len(assocs) == 0 {
		return 0, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	// APPEND-ONLY : INSERT event auto dans _history (plus d'INSERT OR IGNORE legacy).
	// Dédup garantie par loadUnassociatedMedia (forward-only via la vue _latest).
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO media_match_associations_history
			(media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at)
		VALUES (?, ?, ?, FALSE, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
	`)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, a := range assocs {
		res, err := stmt.ExecContext(ctx, a.MediaFileID, a.MatchID, a.DeltaSeconds)
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("insert assoc media=%d match=%s: %w", a.MediaFileID, a.MatchID, err)
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return inserted, nil
}
