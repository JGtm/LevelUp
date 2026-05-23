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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

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

	n, err := InsertHighlightEvents(ctx, sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("InsertHighlightEvents: %w", err)
	}

	// Upsert XUID aliases from events.
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				_ = UpsertXUIDAlias(ctx, globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag)
			}
		}
	}

	if n > 0 {
		result.EventsInserted += n
		_ = MarkEventsLoaded(ctx, sharedDB, matchID)
	}

	// Fix Phase 1bis (mai 2026) : ne marquer MBitKillerVictim que si l'insert
	// a réellement réussi. Avant, l'insert + le mark étaient appelés
	// inconditionnellement avec `_ =` qui swallowait l'erreur — bit menteur
	// dormant, masqué tant que les events n'arrivaient pas (parser cassé).
	if pairsErr := InsertKillerVictimPairsFromEvents(ctx, sharedDB, matchID, events); pairsErr != nil {
		slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("killer_victim_pairs %s: %v", matchID, pairsErr))
		}
	} else {
		_ = MarkKillerVictimLoaded(ctx, sharedDB, matchID)
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
		slog.DebugContext(ctx, "processHighlightEvents: film absent ou chunk vide",
			"match_id", matchID, "found", found, "data_len", len(data),
		)
		// Marquer events_loaded=TRUE pour ne pas retenter à chaque sync : le
		// film 404 est définitif (Halo ne sauve pas le film de tous les matchs).
		if markErr := MarkEventsLoaded(ctx, sharedDB, matchID); markErr != nil {
			slog.DebugContext(ctx, "MarkEventsLoaded échoué (no-film)",
				"match_id", matchID, "err", markErr)
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

	n, err := InsertHighlightEvents(ctx, sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("InsertHighlightEvents: %w", err)
	}

	// Upsert les gamertags extraits depuis le film (source la plus fiable).
	// P5.3 : ecriture dans la DB globale xbox_aliases.
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

	if n > 0 {
		result.EventsInserted += n
		if markErr := MarkEventsLoaded(ctx, sharedDB, matchID); markErr != nil {
			slog.WarnContext(ctx, "MarkEventsLoaded échoué", "match_id", matchID, "err", markErr)
		}
	}

	pairsErr := InsertKillerVictimPairsFromEvents(ctx, sharedDB, matchID, events)
	if pairsErr != nil {
		slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
		// Non-bloquant : on continue.
	} else {
		if markErr := MarkKillerVictimLoaded(ctx, sharedDB, matchID); markErr != nil {
			slog.WarnContext(ctx, "MarkKillerVictimLoaded échoué", "match_id", matchID, "err", markErr)
		}
	}

	slog.DebugContext(ctx, "processHighlightEvents: terminé",
		"match_id", matchID,
		"film_version", filmMajorVersion,
		"events_parsed", len(events),
		"events_inserted", n,
		"aliases_upserted", aliasCount,
		"killer_victim_err", pairsErr,
	)
	return nil
}
