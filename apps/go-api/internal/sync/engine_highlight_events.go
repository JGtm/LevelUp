// Package sync — engine_highlight_events.go : parse + insert highlight events.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - ProcessHighlightEvents : path standalone (télécharge le chunk puis parse
//   - insert). Exposé pour les outils de replay (cmd/replay_highlight_events,
//     events_heal, events_replay). Marque MBitKillerVictim si insert réussi.
//   - persistCombatCompletion : écriture atomique (events + killer_victim + bits)
//     sur le writer RW shared, via persist.EventsCompletionPersister.
//
// La logique de parse_anomaly (chunk non-vide mais 0 events extraits → WARN +
// IncCounter expvar) est portée par ProcessHighlightEvents.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// defaultFilmRetryWindow : fenêtre pendant laquelle un film absent (404) est
// retenté au lieu d'être marqué définitivement absent (events_loaded=TRUE sort
// le match du retry set). Au-delà, on suppose le film réellement indisponible.
//
// 30 jours (révisé 2026-06-01) : les films Halo sont conservés PLUSIEURS MOIS
// côté Waypoint (confirmé en prod), pas quelques heures comme le supposait
// l'ancienne valeur (48h). Le vrai danger n'est pas la latence de propagation
// (minutes) mais une PANNE PROLONGÉE d'écriture combat (cf. incidents RC-A/RC-E :
// shared servi read-only pendant >24h). Avec 48h, une telle panne marquait
// définitivement absents des films encore disponibles → perte permanente du
// match dans le retry set. 30 jours couvre toute panne réaliste tout en finissant
// par sortir les vrais films expirés. Cf. .ai/HANDOFF_sync_combat_completion.md.
const defaultFilmRetryWindow = 30 * 24 * time.Hour

// filmRetryWindow retourne la fenêtre effective. Override via
// LEVELUP_FILM_RETRY_WINDOW_HOURS (entier d'heures, > 0) — utile pour rallonger
// après une panne longue connue, ou raccourcir en test. Valeur invalide / absente
// → defaultFilmRetryWindow.
func filmRetryWindow() time.Duration {
	if v := os.Getenv("LEVELUP_FILM_RETRY_WINDOW_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			return time.Duration(h) * time.Hour
		}
	}
	return defaultFilmRetryWindow
}

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
	return time.Since(startTime.Time) > filmRetryWindow()
}

// highlightChunkFetcher : sous-ensemble de HaloClient nécessaire au fetch du
// chunk highlight events. Interface étroite → testable sans implémenter tout
// HaloClient.
type highlightChunkFetcher interface {
	GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error)
}

// freshFilmWaitWindow : un match plus récent que ce seuil est "frais" — son film
// highlight peut ne pas être encore publié au 1er fetch (propagation Halo ~1 min).
// fetchHighlightChunkResilient attend alors (retry borné) pour que le match
// s'enrichisse COMPLÈTEMENT dans le sync qui le découvre, plutôt que de
// s'afficher à moitié puis d'être rattrapé par la convergence un cycle plus tard
// (choix produit : complétude > latence). Au-delà (vieux match / backfill),
// aucune attente — le film est soit présent soit définitivement absent.
const freshFilmWaitWindow = 10 * time.Minute

// freshFilmRetryDelays retourne les intervalles de retry du fetch film d'un
// match frais. Variable (pas const) pour override en test.
//
// Réglable en PROD sans redéploiement via LEVELUP_FRESH_FILM_RETRY :
//   - "0"           → désactive l'attente (retombe sur la convergence)
//   - "30,30,30"    → CSV de secondes (intervalles entre tentatives)
//   - absent / vide → défaut ci-dessous
//
// Défaut : 30s × 3 (re-essais à +30s/+60s/+90s après le 1er fetch raté). Le
// watcher détecte un match en ~10s mais le film Halo se publie en ~1 min : 30s
// est une VALEUR DE DÉPART pour laisser le film arriver, à affiner en prod selon
// le taux de complétude au 1er passage. Cf. .ai/HANDOFF_ENRICHMENT_CONVERGENCE.md.
var freshFilmRetryDelays = func() []time.Duration {
	v := os.Getenv("LEVELUP_FRESH_FILM_RETRY")
	if v == "0" {
		return nil
	}
	if v != "" {
		var out []time.Duration
		for _, p := range strings.Split(v, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
				out = append(out, time.Duration(n)*time.Second)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []time.Duration{30 * time.Second, 30 * time.Second, 30 * time.Second}
}

// fetchHighlightChunkResilient récupère le chunk highlight events ; pour un match
// FRAIS dont le film n'est pas encore prêt (found=false SANS erreur), il
// re-essaie à court intervalle jusqu'à publication du film. Garantit qu'un match
// récent est enrichi du premier coup (events → killer_victim → frags par arme)
// au lieu d'être affiché à moitié. Respecte l'annulation du ctx ; ne re-essaie
// QUE pour les matchs récents (pas d'attente sur un backfill ou un film absent).
func fetchHighlightChunkResilient(
	ctx context.Context, client highlightChunkFetcher, matchID string, startTime time.Time,
) ([]byte, int, bool, error) {
	data, ver, found, err := client.GetHighlightEventsChunk(ctx, matchID)
	if err != nil || found || startTime.IsZero() || time.Since(startTime) > freshFilmWaitWindow {
		return data, ver, found, err
	}
	for i, d := range freshFilmRetryDelays() {
		select {
		case <-ctx.Done():
			return data, ver, found, ctx.Err()
		case <-time.After(d):
		}
		data, ver, found, err = client.GetHighlightEventsChunk(ctx, matchID)
		if err != nil || found {
			if found {
				slog.DebugContext(ctx, "highlight chunk prêt après attente film",
					"match_id", matchID, "retry", i+1)
			}
			return data, ver, found, err
		}
	}
	return data, ver, found, err
}

// ProcessHighlightEvents télécharge le chunk highlight events, le parse et
// insère les events + paires killer/victim dans la shared DB.
// Retourne une erreur uniquement en cas de défaillance fatale (non-nil = warning dans processMatch).
//
// Exposé (majuscule) pour les outils de replay : cmd/replay_highlight_events.
func ProcessHighlightEvents(
	ctx context.Context,
	client HaloClient,
	sharedDB *sql.DB,
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
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "highlight_events_parse_total_invalid_data")
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : chunk téléchargé non-vide mais 0 event parsé. Traitement :
		// définitif (vieux) → events_empty (sort du retry set sans mentir sur
		// events_loaded) ; récent → on retente (events_loaded reste FALSE).
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "highlight_events_parse_anomaly_total")
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "highlight_events_parse_total_stale_cache")
		definitive := isNoFilmDefinitive(ctx, sharedDB, matchID)
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
			"definitive", definitive,
		)
		if definitive && sharedDB != nil {
			if markErr := persist.NewEventsCompletionPersister(sharedDB).
				MarkEventsEmptyDefinitive(ctx, matchID, MBitEvents); markErr != nil {
				slog.DebugContext(ctx, "MarkEventsEmptyDefinitive échoué (parse_anomaly)",
					"match_id", matchID, "err", markErr)
			}
		}
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events (definitive=%v)", matchID, len(data), filmMajorVersion, definitive))
		}
		return nil
	}
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), "highlight_events_parse_total_ok")

	// Note alias : plus d'upsert xbox_aliases ici. Les gamertags des films
	// alimentent killer_victim_pairs (via persistCombatCompletion) que lit
	// v_gamertag_lookup ; le store global a été consolidé dans shared (2026-06-19).

	// Complétion combat via le persister orchestré (1 TX, writer RW). Remplace
	// les écritures db.Exec directes legacy (InsertHighlightEvents +
	// InsertKillerVictimPairsFromEvents + MarkEventsLoaded/MarkKillerVictimLoaded)
	// — règle absolue : zéro écriture shared hors package persist.
	n, err := persistCombatCompletion(ctx, sharedDB, matchID, events)
	if err != nil {
		// Log explicite : une écriture combat échouée (ex. shared read-only) ne
		// doit JAMAIS être silencieuse — c'est précisément la classe d'incident
		// (RC-A/RC-E) que ce module rend désormais visible. Le caller (events_heal)
		// la compte aussi en `failed`.
		slog.ErrorContext(ctx, "processHighlightEvents: persistCombatCompletion échoué",
			"match_id", matchID, "events_parsed", len(events), "err", err)
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
	)
	return nil
}

// persistCombatCompletion construit les rows de complétion (highlight_events +
// killer_victim_pairs par-kill) depuis les events parsés et les écrit en UNE
// transaction atomique via persist.EventsCompletionPersister (writer RW shared).
// Retourne le nombre d'events insérés. Centralise la construction du mapping pour
// ProcessHighlightEvents (unique caller).
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
