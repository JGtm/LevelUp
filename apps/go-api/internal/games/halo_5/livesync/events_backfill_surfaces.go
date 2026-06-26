package livesync

// events_backfill_surfaces.go — backfill des surfaces DÉRIVÉES de la timeline d'events
// (weapon_accuracy + highlight_events assist/objectif) pour les matchs Halo 5 DÉJÀ
// collectés AVANT l'ajout de ces surfaces.
//
// Pourquoi un chemin dédié (et pas une re-collecte) : le collect normal est gated par
// l'idempotence match_registry — un 2e batch sur un match connu est intégralement
// skippé par SharedPersister (cf. ingest.CollectMatchBatch), donc les nouvelles tables
// ne seraient jamais peuplées sur l'existant. Comme personne ne (re)joue ces matchs,
// « new-matches-only » ne suffit pas : on RE-FETCH /events par match (endpoint
// PAR-MATCH → un seul token valide couvre tout l'historique partagé) et on écrit
// UNIQUEMENT les surfaces dérivées, idempotent (DELETE-puis-INSERT ciblé).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_5/ingest"
	"levelup/go-api/internal/persist"
)

// FetchEventsFunc récupère la timeline canonique d'un match (injectable → fake en
// test ; en prod = closure sur halo5.FetchCanonicalEvents(src, …)).
type FetchEventsFunc func(ctx context.Context, matchID string) ([]canonical.MatchEvent, error)

// EventsBackfillStats résume une passe de backfill des surfaces dérivées d'events.
type EventsBackfillStats struct {
	Matches    int // matchs candidats traités
	Updated    int // matchs ré-écrits avec au moins une surface
	FetchErr   int // timelines indisponibles (skip, retry ultérieur)
	WeaponRows int // lignes weapon_accuracy écrites
	EventRows  int // lignes highlight_events assist+objectif écrites
}

// RunEventsBackfill re-dérive weapon_accuracy + highlight_events (assist/objectif) pour
// les matchs Halo 5 EXISTANTS qui n'ont pas encore weapon_accuracy (collectés avant la
// surface). shared doit être ouvert en RW single-writer (serveur arrêté). fetch est
// injecté (prod = halo5.FetchCanonicalEvents sur une CaptureSource). maxMatches<=0 =
// tous. Best-effort par match : un fetch ou une écriture KO saute le match (compté),
// sans interrompre la passe — relancer reprend les matchs encore sans weapon_accuracy.
func RunEventsBackfill(ctx context.Context, shared *sql.DB, fetch FetchEventsFunc, maxMatches int, logger *slog.Logger) (EventsBackfillStats, error) {
	if logger == nil {
		logger = slog.Default()
	}
	resolveXUID, err := loadGamertagXUIDResolver(ctx, shared)
	if err != nil {
		return EventsBackfillStats{}, fmt.Errorf("backfill events: résolveur xuid: %w", err)
	}
	ids, err := matchesMissingWeaponAccuracy(ctx, shared, maxMatches)
	if err != nil {
		return EventsBackfillStats{}, fmt.Errorf("backfill events: énumération: %w", err)
	}

	stats := EventsBackfillStats{Matches: len(ids)}
	for i, id := range ids {
		events, ferr := fetch(ctx, id)
		if ferr != nil {
			stats.FetchErr++
			logger.WarnContext(ctx, "backfill events: timeline indisponible (skip)", "match_id", id, "err", ferr)
			continue
		}
		wacc := ingest.MapWeaponAccuracy(id, events, resolveXUID)
		hl := ingest.MapAssistEvents(id, events, resolveXUID)
		hl = append(hl, ingest.MapObjectiveImpulseEvents(id, events, resolveXUID)...)

		w, e, werr := WriteEventDerivedSurfaces(ctx, shared, id, wacc, hl)
		if werr != nil {
			logger.WarnContext(ctx, "backfill events: écriture échouée (match sauté)", "match_id", id, "err", werr)
			continue
		}
		if w > 0 || e > 0 {
			stats.Updated++
		}
		stats.WeaponRows += w
		stats.EventRows += e
		if (i+1)%200 == 0 {
			logger.InfoContext(ctx, "backfill events: progression",
				"fait", i+1, "total", len(ids), "maj", stats.Updated)
		}
	}
	logger.InfoContext(ctx, "backfill events: terminé",
		"matchs", stats.Matches, "maj", stats.Updated, "fetch_err", stats.FetchErr,
		"weapon_rows", stats.WeaponRows, "event_rows", stats.EventRows)
	return stats, nil
}

// WriteEventDerivedSurfaces écrit, dans UNE transaction, les surfaces dérivées d'events
// d'un match : weapon_accuracy + highlight_events (assist + objectif). Idempotent par
// DELETE-puis-INSERT CIBLÉ : purge weapon_accuracy du match (100% issu des events) +
// les highlight_events event_type IN ('assist','mode') du match — SANS toucher aux
// médailles/kills déjà persistés — puis ré-insère. Tables sans index (ART-safe, ADR 0026).
func WriteEventDerivedSurfaces(ctx context.Context, shared *sql.DB, matchID string,
	wacc []persist.WeaponAccuracyInsert, hl []persist.HighlightEventInsert) (weaponRows, eventRows int, err error) {
	tx, err := shared.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, `DELETE FROM weapon_accuracy WHERE match_id = ?`, matchID); err != nil {
		return 0, 0, fmt.Errorf("delete weapon_accuracy: %w", err)
	}
	if _, err = tx.ExecContext(ctx,
		`DELETE FROM highlight_events WHERE match_id = ? AND event_type IN ('assist','mode')`, matchID); err != nil {
		return 0, 0, fmt.Errorf("delete highlight_events: %w", err)
	}
	for _, r := range wacc {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO weapon_accuracy (match_id, xuid, weapon_id, shots_fired, shots_landed, drops)
			VALUES (?, ?, CAST(? AS UBIGINT), ?, ?, ?)`,
			r.MatchID, r.XUID, r.WeaponID, r.ShotsFired, r.ShotsLanded, r.Drops); err != nil {
			return 0, 0, fmt.Errorf("insert weapon_accuracy %s/%s/%d: %w", r.MatchID, r.XUID, r.WeaponID, err)
		}
	}
	for _, e := range hl {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint)
			VALUES (?, ?, ?, ?, ?)`,
			e.MatchID, e.EventType, e.TimeMS, e.XUID, e.DetailsJSON); err != nil {
			return 0, 0, fmt.Errorf("insert highlight_events %s/%s: %w", e.MatchID, e.EventType, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return len(wacc), len(hl), nil
}

// matchesMissingWeaponAccuracy : match_ids du shared H5 sans AUCUNE ligne
// weapon_accuracy (matchs collectés avant la surface). Récents d'abord. Le shared est
// per-titre (h5) → pas de filtre titre. maxMatches<=0 = tous.
func matchesMissingWeaponAccuracy(ctx context.Context, shared *sql.DB, maxMatches int) ([]string, error) {
	q := `SELECT r.match_id FROM match_registry r
	      WHERE NOT EXISTS (SELECT 1 FROM weapon_accuracy w WHERE w.match_id = r.match_id)
	      ORDER BY COALESCE(r.start_time_utc, r.start_time) DESC`
	if maxMatches > 0 {
		q += fmt.Sprintf(" LIMIT %d", maxMatches)
	}
	rows, err := shared.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// loadGamertagXUIDResolver construit le résolveur gamertag→xuid depuis shared.xuid_
// aliases : les events H5 sont gamertag-keyés (XUID vide, cf. h5EventIdentity), et les
// participants des matchs existants ont déjà leur alias. Gamertag inconnu → "" (la
// ligne dérivée garde xuid vide, cohérent avec le live quand le resolver échoue).
func loadGamertagXUIDResolver(ctx context.Context, shared *sql.DB) (func(string) string, error) {
	rows, err := shared.QueryContext(ctx,
		`SELECT gamertag, xuid FROM xuid_aliases WHERE gamertag <> '' AND xuid <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var gt, xuid string
		if err := rows.Scan(&gt, &xuid); err != nil {
			return nil, err
		}
		m[gt] = xuid
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return func(gt string) string { return m[gt] }, nil
}
