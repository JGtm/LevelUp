package snapshot

// snapshot_readiness_eval.go — orchestration du marquage snapshot-ready (Phase 2).
//
// Charge les faits d'un match (player + shared, 2 requêtes Go-diff, JAMAIS d'ATTACH
// cross-DB), applique le prédicat pur isMatchSnapshotReady, et INSÈRE une row
// stage='snapshot' (INSERT pur append-only — zéro UPDATE/DELETE indexé, no_art OK)
// portant snapshot_ready_at + partial_reasons pour les matchs complets. Appelé en
// fin de runPostSyncPipeline (étape 6), best-effort, réévalué à chaque cycle.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
)

// snapshotGraceWindow : ancienneté au-delà de laquelle un match dont une dérivation
// reste bloquée est marqué READY de FORCE (partial_reasons = forced + blocked_*),
// pour ne jamais geler l'affichage (doctrine app autonome / zéro perte silencieuse).
// Défaut 60j — STRICTEMENT > filmRetryWindow (30j) pour laisser le no-film terminal
// se résoudre avant le forçage. Override : LEVELUP_SNAPSHOT_GRACE_HOURS.
func snapshotGraceWindow() time.Duration {
	const defaultHours = 60 * 24
	if v := os.Getenv("LEVELUP_SNAPSHOT_GRACE_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			return time.Duration(h) * time.Hour
		}
	}
	return defaultHours * time.Hour
}

// slugProducesWeaponKills : le titre produit-il le détail kills-par-arme
// (CapWeaponKills) ? Infinite oui, Halo 5 non. Gate la terminalité weapon du prédicat.
func slugProducesWeaponKills(slug string) bool {
	if slug == "" {
		slug = title.DefaultSlug
	}
	d := title.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(title.CapWeaponKills)
}

func slugHasFirefightCap(slug string) bool {
	if slug == "" {
		slug = title.DefaultSlug
	}
	d := title.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(title.CapFirefight)
}

func snapshotReadinessCaps(slug string) titleReadinessCaps {
	return titleReadinessCaps{
		hasLUSR:        slugHasLUSR(slug),
		hasWeaponKills: slugProducesWeaponKills(slug),
		hasFirefight:   slugHasFirefightCap(slug),
	}
}

// snapshotCandidate : un match sans snapshot_ready_at + ses faits PLAYER.
type snapshotCandidate struct {
	matchID string
	facts   matchReadinessFacts
}

// sharedSnapFacts : faits SHARED d'un match (registry + team count) + start_time (grâce).
type sharedSnapFacts struct {
	eventsLoaded      bool
	backfillCompleted int64
	isRanked          bool
	isFirefight       bool
	durationSeconds   int
	humanTeamCount    int
	startTime         time.Time
}

type snapshotReadyRow struct {
	matchID string
	reasons string
}

// EvaluateSnapshotReadiness marque les matchs complets du joueur (snapshot_ready_at).
// Retourne le nombre de matchs nouvellement marqués. Best-effort côté caller.
func EvaluateSnapshotReadiness(ctx context.Context, playerDB, sharedDB *sql.DB, xuid, titleSlug string) (int, error) {
	if playerDB == nil || sharedDB == nil || xuid == "" {
		return 0, nil
	}
	candidates, err := loadSnapshotCandidates(ctx, playerDB)
	if err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, nil
	}
	shared, err := loadSnapshotSharedFacts(ctx, sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	caps := snapshotReadinessCaps(titleSlug)
	cutoff := time.Now().Add(-snapshotGraceWindow())

	var ready []snapshotReadyRow
	var forced []string // matchs marqués ready DE FORCE après la fenêtre de grâce
	for _, cand := range candidates {
		sf, ok := shared[cand.matchID]
		if !ok {
			continue // match absent de shared (anormal) → on n'évalue pas
		}
		f := cand.facts
		f.eventsLoaded = sf.eventsLoaded
		f.backfillCompleted = sf.backfillCompleted
		f.isRanked = sf.isRanked
		f.isFirefight = sf.isFirefight
		f.durationSeconds = sf.durationSeconds
		f.humanTeamCount = sf.humanTeamCount
		agedOut := !sf.startTime.IsZero() && sf.startTime.Before(cutoff)
		isReady, reasons := isMatchSnapshotReady(f, caps, agedOut)
		if isReady {
			ready = append(ready, snapshotReadyRow{cand.matchID, marshalSnapshotReasons(reasons)})
			if reasonsContainForced(reasons) {
				forced = append(forced, cand.matchID)
			}
		}
	}
	if len(ready) == 0 {
		return 0, nil
	}
	// Le marquage DE FORCE après grâce = acceptation délibérée d'une dérivation jamais
	// converger (perte de donnée volontaire, doctrine app autonome) → log distinct
	// (module=snapshot) pour qu'un opérateur le voie au lieu d'un compteur global noyé.
	if len(forced) > 0 {
		snapshotLog.WarnContext(ctx, "snapshot: matchs marqués ready DE FORCE (grâce dépassée, dérivation bloquée)",
			"titleSlug", titleSlug, "xuid", xuid, "count", len(forced), "sample", forcedSample(forced))
	}
	return insertSnapshotReadyRows(ctx, playerDB, ready)
}

// reasonsContainForced indique si la liste de raisons inclut le marquage forcé.
func reasonsContainForced(reasons []string) bool {
	for _, r := range reasons {
		if r == snapReasonForced {
			return true
		}
	}
	return false
}

// forcedSample retourne un échantillon (max 10) de match_id forcés pour le log.
func forcedSample(forced []string) []string {
	if len(forced) > 10 {
		return forced[:10]
	}
	return forced
}

func marshalSnapshotReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "[]"
	}
	if b, err := json.Marshal(reasons); err == nil {
		return string(b)
	}
	return "[]"
}

// loadSnapshotCandidates : matchs du joueur SANS snapshot_ready_at + leurs faits player.
func loadSnapshotCandidates(ctx context.Context, playerDB *sql.DB) ([]snapshotCandidate, error) {
	rows, err := playerDB.QueryContext(ctx, `
		SELECT pme.match_id,
		       pme.performance_score IS NOT NULL,
		       pme.dominance_flag IS NOT NULL,
		       pme.psa_checked_at IS NOT NULL,
		       EXISTS(SELECT 1 FROM match_citations_latest c WHERE c.match_id = pme.match_id),
		       EXISTS(SELECT 1 FROM match_skill_rank_latest s WHERE s.match_id = pme.match_id AND s.rating_type = 'LUSR')
		FROM player_match_enrichment_latest pme
		WHERE pme.snapshot_ready_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("loadSnapshotCandidates: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []snapshotCandidate
	for rows.Next() {
		var c snapshotCandidate
		if err := rows.Scan(&c.matchID, &c.facts.perfScoreSet, &c.facts.dominanceSet,
			&c.facts.psaCheckedSet, &c.facts.citationsExist, &c.facts.lusrRowExists); err != nil {
			return nil, fmt.Errorf("loadSnapshotCandidates scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// loadSnapshotSharedFacts : faits shared des matchs du joueur, keyés par match_id.
func loadSnapshotSharedFacts(ctx context.Context, sharedDB *sql.DB, xuid string) (map[string]sharedSnapFacts, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.match_id,
		       COALESCE(mr.events_loaded, FALSE),
		       COALESCE(mr.backfill_completed, 0),
		       COALESCE(mr.is_ranked, FALSE),
		       COALESCE(mr.is_firefight, FALSE),
		       COALESCE(mr.duration_seconds, 0),
		       `+analysis.SQLStartTimeCanonical("mr")+`,
		       (SELECT COUNT(DISTINCT mp2.team_id) FROM match_participants mp2
		          WHERE mp2.match_id = mr.match_id AND mp2.team_id IS NOT NULL)
		FROM match_registry mr
		WHERE EXISTS(SELECT 1 FROM match_participants mp WHERE mp.match_id = mr.match_id AND mp.xuid = ?)`, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadSnapshotSharedFacts: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	out := make(map[string]sharedSnapFacts)
	for rows.Next() {
		var id string
		var f sharedSnapFacts
		var st sql.NullTime
		if err := rows.Scan(&id, &f.eventsLoaded, &f.backfillCompleted, &f.isRanked,
			&f.isFirefight, &f.durationSeconds, &st, &f.humanTeamCount); err != nil {
			return nil, fmt.Errorf("loadSnapshotSharedFacts scan: %w", err)
		}
		if st.Valid {
			f.startTime = st.Time
		}
		out[id] = f
	}
	return out, rows.Err()
}

// insertSnapshotReadyRows : INSERT pur stage='snapshot' (append-only) pour chaque
// match marqué ready. snapshot_ready_at + partial_reasons ; autres colonnes NULL
// (la vue _latest merge-on-read prend ces valeurs via le stage propriétaire 'snapshot').
func insertSnapshotReadyRows(ctx context.Context, playerDB *sql.DB, rows []snapshotReadyRow) (int, error) {
	tx, err := playerDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("snapshot readiness BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES (?, ?, ?, 'snapshot')`)
	if err != nil {
		return 0, fmt.Errorf("snapshot readiness prepare: %w", err)
	}
	defer stmt.Close() //nolint:errcheck
	now := time.Now()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx, r.matchID, now, r.reasons); err != nil {
			return 0, fmt.Errorf("snapshot readiness insert %s: %w", r.matchID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("snapshot readiness commit: %w", err)
	}
	return len(rows), nil
}
