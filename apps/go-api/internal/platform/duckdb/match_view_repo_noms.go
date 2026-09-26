// Package duckdb — match_view_repo_noms.go : les lectures NOMMÉES de la vue Match — events
// highlight (Q21) et rencontres (Q23) —, qui nomment leurs lignes par l'annuaire du match en
// portée base (lot A du plan perf « lectures par périmètre », 2026-09-26, ADR 0036 I1). Extraites
// de match_view_repo_extras.go dans ce lot (fichier gelé au-delà de 500 lignes).
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// medalNameFromRawJSON extrait le nom anglais de la médaille du raw_json d'un
// event `medal` de highlight_events ({"medal_name": "Odin's Raven", ...}).
// Nil si le JSON est vide, illisible ou sans champ medal_name non vide — l'event
// reste servi, simplement anonyme (le service ne résout alors rien).
func medalNameFromRawJSON(raw string) *string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var payload struct {
		MedalName string `json:"medal_name"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	name := strings.TrimSpace(payload.MedalName)
	if name == "" {
		return nil
	}
	return &name
}

// GetMatchEvents retourne les events highlight du match (Q21).
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *MatchViewRepo) GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil
	}
	defer release()
	stop := timing.FromContext(ctx).Section("match_events")
	results, ok, err := lireEvenementsDuMatch(ctx, sharedDB, matchID)
	stop()
	if err != nil || !ok {
		return nil, err
	}
	// Noms : l'annuaire du match en portée base (lot A, plus de jointure sur la vue des noms) ;
	// seuls les events portant un xuid sont nommés.
	stop = timing.FromContext(ctx).Section("match_events_annuaire")
	defer stop()
	nommes := make([]*domain.EventRaw, 0, len(results))
	for i := range results {
		if x := results[i].XUID; x != nil && *x != "" {
			nommes = append(nommes, &results[i])
		}
	}
	if err := nommerLignesPorteeBase(ctx, sharedDB, []string{matchID}, nommes, accesLigne[*domain.EventRaw]{
		xuid:   func(e *domain.EventRaw) string { return *e.XUID },
		match:  func(*domain.EventRaw) string { return matchID },
		nommer: func(e **domain.EventRaw, gt string) { (*e).Gamertag = &gt },
	}); err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEvents: %w", err)
	}
	return results, nil
}

// lireEvenementsDuMatch lit Q21. ok=false : la requête a échoué (table absente sur une base non
// migrée) — l'appelant rend vide, comme avant le lot A.
func lireEvenementsDuMatch(ctx context.Context, db *sql.DB, matchID string) ([]domain.EventRaw, bool, error) {
	rows, err := db.QueryContext(ctx, Q21MatchEventsWithXUID, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match_view: events indisponibles (Q21)", "match_id", matchID, "err", err)
		return nil, false, nil
	}
	defer rows.Close()

	var results []domain.EventRaw
	for rows.Next() {
		var e domain.EventRaw
		var medalRaw sql.NullString
		if err := rows.Scan(&e.EventType, &e.TimeMS, &e.XUID, &medalRaw); err != nil {
			return nil, false, fmt.Errorf("MatchViewRepo.GetMatchEvents scan: %w", err)
		}
		if medalRaw.Valid {
			e.MedalName = medalNameFromRawJSON(medalRaw.String)
		}
		results = append(results, e)
	}
	return results, true, rows.Err()
}

// GetMatchEncounters retourne l'historique de rencontres avec les participants (Q23).
// Exécutée sur SharedReader (ADR 0016) — Q23 lit match_participants ; les noms viennent de
// l'annuaire du match en portée base (lot A, plus de jointure sur la vue des noms).
func (r *MatchViewRepo) GetMatchEncounters(ctx context.Context, matchID, myXUID string) ([]domain.EncounterRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters: shared reader: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("match_encounters")
	results, err := r.lireRencontresDuMatch(ctx, sharedDB, matchID, myXUID)
	stop()
	if err != nil {
		return nil, err
	}
	stop = timing.FromContext(ctx).Section("match_encounters_annuaire")
	defer stop()
	if err := nommerLignesPorteeBase(ctx, sharedDB, []string{matchID}, results, accesLigne[domain.EncounterRaw]{
		xuid:   func(e domain.EncounterRaw) string { return e.XUID },
		match:  func(domain.EncounterRaw) string { return matchID },
		nommer: func(e *domain.EncounterRaw, gt string) { e.Gamertag = gt },
	}); err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters: %w", err)
	}
	return results, nil
}

// lireRencontresDuMatch lit Q23 (lignes sans nom).
func (r *MatchViewRepo) lireRencontresDuMatch(ctx context.Context, db *sql.DB, matchID, myXUID string) ([]domain.EncounterRaw, error) {
	// Masquage Campagne (Halo 5) : count_together agrège l'historique du joueur
	// (me.xuid) → forme by-match-id sur me.match_id. No-op Infinite. Item backlog H1.
	q := resolveCampaignExclusionByMatchID(Q23MatchEncounters, r.pdb.TitleSlug, "me.match_id")
	rows, err := db.QueryContext(ctx, q,
		matchID, myXUID, // this_match WHERE
		matchID, myXUID, // my_team WHERE
		myXUID, // me.xuid = ?
	)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters: %w", err)
	}
	defer rows.Close()

	var results []domain.EncounterRaw
	for rows.Next() {
		var enc domain.EncounterRaw
		if err := rows.Scan(&enc.XUID, &enc.IsBot, &enc.CountTogether, &enc.IsAlly); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounters scan: %w", err)
		}
		results = append(results, enc)
	}
	return results, rows.Err()
}
