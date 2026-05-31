// Package sync — engine_highlight_events.go : parse + insert highlight events.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - insertHighlightEventsFromData : parse + insert events à partir de données
//     déjà fetchées. Helper interne consommé par insertFetchedMatch (chemin
//     parallèle, evite un second appel API). Marque MBitKillerVictim si insert
//     réussi (fix Phase 1bis bit menteur).
//   - ProcessHighlightEvents : path standalone (télécharge le chunk puis parse
//   - insert). Exposé pour les outils de replay (cmd/replay_highlight_events,
//     events_heal, events_replay).
//
// Les deux fonctions partagent la même logique de parse_anomaly (chunk non-vide
// mais 0 events extraits → WARN + IncCounter expvar). Comportement INCHANGÉ —
// pur déplacement.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// filmRetryWindow : fenêtre pendant laquelle un film absent (404) est considéré
// comme « peut-être pas encore propagé côté Halo » et donc retenté au lieu
// d'être marqué définitivement absent. Au-delà, le film est jugé perdu (Halo ne
// conserve pas tous les films) et events_loaded passe à TRUE pour sortir du
// retry set. Tunable : 48h couvre largement la latence de propagation observée
// tout en bornant les retries. Cf. .ai/HANDOFF_sync_combat_completion.md
// (incident : un film simplement retardé était marqué définitif → perte).
const filmRetryWindow = 48 * time.Hour

// isNoFilmDefinitive décide, sur film absent (404), si on marque
// events_loaded=TRUE (définitif, sort du retry set) ou si on laisse FALSE
// (réessai au prochain cycle). Définitif si le match est plus vieux que
// filmRetryWindow OU si start_time est inconnu (NULL) — dans ce dernier cas on
// ne peut pas distinguer un retard d'une absence, on garde le comportement
// legacy (marquer). Best-effort : toute erreur de lecture → définitif pour ne
// jamais bloquer le pipeline.
func isNoFilmDefinitive(ctx context.Context, db *sql.DB, matchID string) bool {
	if db == nil {
		return true
	}
	var startTime sql.NullTime
	err := db.QueryRowContext(ctx,
		`SELECT start_time FROM match_registry WHERE match_id = ?`, matchID).Scan(&startTime)
	if err != nil || !startTime.Valid {
		return true // âge inconnu → comportement legacy (définitif)
	}
	return time.Since(startTime.Time) > filmRetryWindow
}

// insertHighlightEventsFromData parse et insère les highlight events à partir de données déjà fetchées.
// Helper utilisé par insertFetchedMatch pour injection de dépendance.
func insertHighlightEventsFromData(
	ctx context.Context,
	sharedDB, globalDB *sql.DB,
	matchID string,
	data []byte,
	filmMajorVersion int,
	result *domain.SyncResult,
) error {
	if len(data) == 0 {
		return nil // Pas de données — OK, pas d'erreur.
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		// Phase 4.3 métriques : compteur erreurs de parse (zlib invalide,
		// format film incorrect, etc.) pour observabilité prod.
		observability.IncCounter("highlight_events_parse_total_invalid_data")
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : on a téléchargé un chunk non-vide mais le parser
		// n'a rien extrait. Avant le fix bit-aligné (mai 2026), ce cas
		// était silencieusement loggé en DEBUG et faisait perdre tout
		// l'historique highlight events. Désormais : WARN + compteur
		// expvar pour qu'une regression soit immédiatement visible.
		observability.IncCounter("highlight_events_parse_anomaly_total")
		observability.IncCounter("highlight_events_parse_total_stale_cache")
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
		)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events", matchID, len(data), filmMajorVersion))
		}
		return nil
	}
	observability.IncCounter("highlight_events_parse_total_ok")

	// Upsert XUID aliases from events (DB globale, hors TX shared).
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				_ = UpsertXUIDAlias(ctx, globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag)
			}
		}
	}

	// Complétion combat atomique via persist (events + killer_victim + bits) sur
	// le writer RW shared — voir persistCombatCompletion. Remplace les écritures
	// db.Exec directes legacy (règle absolue : zéro écriture shared hors persist).
	n, err := persistCombatCompletion(ctx, sharedDB, matchID, events)
	if err != nil {
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("combat_completion %s: %v", matchID, err))
		}
		return fmt.Errorf("persistCombatCompletion: %w", err)
	}
	if n > 0 && result != nil {
		result.EventsInserted += n
	}

	return nil
}

// ProcessHighlightEvents télécharge le chunk highlight events, le parse et
// insère les events + paires killer/victim dans la shared DB.
// Retourne une erreur uniquement en cas de défaillance fatale (non-nil = warning dans processMatch).
//
// Exposé (majuscule) pour les outils de replay : cmd/replay_highlight_events.
func ProcessHighlightEvents(
	ctx context.Context,
	client HaloClient,
	sharedDB, globalDB *sql.DB,
	matchID string,
	result *domain.SyncResult,
) error {
	data, filmMajorVersion, found, err := client.GetHighlightEventsChunk(ctx, matchID)
	if err != nil {
		return fmt.Errorf("GetHighlightEventsChunk: %w", err)
	}
	if !found || len(data) == 0 {
		// Décision de marquage sur film absent (404) : définitif (match ancien
		// ou âge inconnu) → MarkEventsLoaded (sort du retry set) ; récent → on
		// laisse events_loaded=FALSE pour réessayer (le film n'est peut-être pas
		// encore propagé — éviter la perte définitive d'un film simplement
		// retardé, cf. .ai/HANDOFF_sync_combat_completion.md).
		definitive := isNoFilmDefinitive(ctx, sharedDB, matchID)
		slog.DebugContext(ctx, "processHighlightEvents: film absent ou chunk vide",
			"match_id", matchID, "found", found, "data_len", len(data),
			"definitive", definitive,
		)
		if definitive {
			if markErr := persist.NewEventsCompletionPersister(sharedDB).
				MarkNoFilmDefinitive(ctx, matchID, MBitEvents); markErr != nil {
				slog.DebugContext(ctx, "MarkNoFilmDefinitive échoué (no-film)",
					"match_id", matchID, "err", markErr)
			}
		}
		return nil
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		observability.IncCounter("highlight_events_parse_total_invalid_data")
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : chunk téléchargé non-vide mais 0 event parsé.
		// Voir insertHighlightEventsFromData pour la justification.
		observability.IncCounter("highlight_events_parse_anomaly_total")
		observability.IncCounter("highlight_events_parse_total_stale_cache")
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
		)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events", matchID, len(data), filmMajorVersion))
		}
		return nil
	}
	observability.IncCounter("highlight_events_parse_total_ok")

	// Upsert les gamertags extraits depuis le film (source la plus fiable).
	// P5.3 : ecriture dans la DB globale xbox_aliases. DB séparée du shared →
	// reste hors de la TX de complétion (best-effort, inchangé).
	aliasCount := 0
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				if uErr := UpsertXUIDAlias(ctx, globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag); uErr == nil {
					aliasCount++
				}
			}
		}
	}

	// Complétion combat via le persister orchestré (1 TX, writer RW). Remplace
	// les écritures db.Exec directes legacy (InsertHighlightEvents +
	// InsertKillerVictimPairsFromEvents + MarkEventsLoaded/MarkKillerVictimLoaded)
	// — règle absolue : zéro écriture shared hors package persist.
	n, err := persistCombatCompletion(ctx, sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("persistCombatCompletion: %w", err)
	}
	if n > 0 && result != nil {
		result.EventsInserted += n
	}

	slog.DebugContext(ctx, "processHighlightEvents: terminé",
		"match_id", matchID,
		"film_version", filmMajorVersion,
		"events_parsed", len(events),
		"events_inserted", n,
		"aliases_upserted", aliasCount,
	)
	return nil
}

// persistCombatCompletion construit les rows de complétion (highlight_events +
// killer_victim_pairs par-kill) depuis les events parsés et les écrit en UNE
// transaction atomique via persist.EventsCompletionPersister (writer RW shared).
// Retourne le nombre d'events insérés. Centralise la construction pour que les
// deux callers (ProcessHighlightEvents et insertHighlightEventsFromData) partagent
// exactement le même mapping.
func persistCombatCompletion(ctx context.Context, sharedDB *sql.DB, matchID string, events []analysis.HighlightEvent) (int, error) {
	hlRows := make([]persist.HLEventCompletion, 0, len(events))
	for _, ev := range events {
		hlRows = append(hlRows, persist.HLEventCompletion{
			XUID:      strconv.FormatUint(ev.XUID, 10),
			EventType: ev.EventType,
			TimeMS:    ev.TimeMS,
			TypeHint:  ev.TypeHint,
		})
	}

	// Paires killer→victim (forme par-kill, gamertags + time_ms) — même calcul
	// que la fonction legacy InsertKillerVictimPairsFromEvents (tolérance 5 ms).
	raw := make([]analysis.RawEvent, 0, len(events))
	for _, ev := range events {
		if ev.EventType != analysis.EventTypeKill && ev.EventType != analysis.EventTypeDeath {
			continue
		}
		raw = append(raw, analysis.RawEvent{
			EventType: ev.EventType,
			XUID:      strconv.FormatUint(ev.XUID, 10),
			Gamertag:  ev.Gamertag,
			TimeMS:    int64(ev.TimeMS),
		})
	}
	const toleranceMS = int64(5)
	pairs := analysis.ComputeKillerVictimPairs(raw, toleranceMS)
	kvRows := make([]persist.KVPairCompletion, 0, len(pairs))
	for _, p := range pairs {
		kvRows = append(kvRows, persist.KVPairCompletion{
			KillerXUID:     p.KillerXUID,
			KillerGamertag: p.KillerGT,
			VictimXUID:     p.VictimXUID,
			VictimGamertag: p.VictimGT,
			TimeMS:         p.TimeMS,
		})
	}

	return persist.NewEventsCompletionPersister(sharedDB).Persist(ctx, persist.EventsCompletionInput{
		MatchID:         matchID,
		Events:          hlRows,
		Pairs:           kvRows,
		MarkKV:          true,
		EventsBit:       MBitEvents,
		KillerVictimBit: MBitKillerVictim,
	})
}
