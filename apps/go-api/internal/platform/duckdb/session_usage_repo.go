// Package duckdb — session_usage_repo.go : lecture du résumé d'usage S1 pour
// l'agrégat de session (chantier session-usage S2).
//
// TROIS LECTURES, TOUTES SUR UN SCOPE FERMÉ DE match_id (aucun filtre temporel —
// le fragment timezone canonique ne s'applique donc pas ici) :
//   - match_usage_films_latest  : l'échelle de temps et les comptes de socle du
//     match — l'EXISTENCE de la ligne définit « match mesuré » ;
//   - match_usage_players_latest : les usages par (match, joueur) ;
//   - match_participants        : camp et présence à la fin (effectifs) — S1 ne
//     duplique pas l'effectif, à dessein (§3 du handoff).
//
// VUES `_latest` UNIQUEMENT (ADR 0026) : la table brute sert les passes
// précédentes. Les colonnes sont celles de
// internal/migration/steps_shared_usage_summary.go — vérifiées sur pièces.
package duckdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
)

// SessionUsageRepo lit les vues S1 sur le SharedReader du joueur. Injecté gated
// par la capability film.usage_summary (registry_pages) — nil ⇒ le bloc usage de
// la page Sessions dégrade en Available=false.
type SessionUsageRepo struct {
	pdb *PlayerDB
}

// NewSessionUsageRepo construit le repo à partir de la player DB (SharedReader).
func NewSessionUsageRepo(pdb *PlayerDB) *SessionUsageRepo {
	return &SessionUsageRepo{pdb: pdb}
}

const sessionUsageQueryTimeout = 15 * time.Second

// LoadUsageFilms retourne, par match_id, la ligne de grain match. Un match sans
// ligne n'est PAS mesuré : il manque de la map, ce n'est jamais une erreur.
func (r *SessionUsageRepo) LoadUsageFilms(ctx context.Context, matchIDs []string) (map[string]sessionusage.FilmRow, error) {
	out := make(map[string]sessionusage.FilmRow, len(matchIDs))
	if len(matchIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, sessionUsageQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: shared reader: %w", err)
	}
	defer release()

	// Seules les colonnes consommées par l'agrégat sont lues (0 code mort) :
	// frame_interval_ms et pad_named restent en table pour d'autres lecteurs.
	q := `SELECT match_id, duration_ms, pad_unnamed, powerup_pickups_json
	      FROM match_usage_films_latest
	      WHERE match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: films query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f sessionusage.FilmRow
		var powerups string
		if err := rows.Scan(&f.MatchID, &f.DurationMS, &f.PadUnnamed, &powerups); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: films scan: %w", err)
		}
		if f.PowerupPickups, err = countMapFromJSON(powerups); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: %s powerup_pickups_json: %w", f.MatchID, err)
		}
		out[f.MatchID] = f
	}
	return out, rows.Err()
}

// LoadUsagePlayers retourne les lignes (match, joueur) du scope. Les grenades
// (produites en S1) ne sont volontairement PAS lues : hors contrat de session.
func (r *SessionUsageRepo) LoadUsagePlayers(ctx context.Context, matchIDs []string) ([]sessionusage.PlayerRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, sessionUsageQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: shared reader: %w", err)
	}
	defer release()

	q := `SELECT match_id, xuid, grapple_pulls, camo_episodes, overshield_episodes,
	             dropped_objects, pad_pickups, deployed_json, pad_pickups_json
	      FROM match_usage_players_latest
	      WHERE match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: players query: %w", err)
	}
	defer rows.Close()
	var out []sessionusage.PlayerRow
	for rows.Next() {
		var p sessionusage.PlayerRow
		var deployed, pads string
		if err := rows.Scan(&p.MatchID, &p.XUID, &p.GrapplePulls, &p.CamoEpisodes,
			&p.OvershieldEpisodes, &p.DroppedObjects, &p.PadPickups, &deployed, &pads); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: players scan: %w", err)
		}
		if p.DeployedByFamily, err = countMapFromJSON(deployed); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: %s/%s deployed_json: %w", p.MatchID, p.XUID, err)
		}
		if p.PadPickupsByFamily, err = countMapFromJSON(pads); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: %s/%s pad_pickups_json: %w", p.MatchID, p.XUID, err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadParticipants retourne les participants du scope (bots inclus : ils
// occupent un slot du lobby, la parité les compte ; leurs gestes, eux, n'ont
// jamais de ligne d'usage — pas de xuid dans le film).
func (r *SessionUsageRepo) LoadParticipants(ctx context.Context, matchIDs []string) ([]sessionusage.ParticipantRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, sessionUsageQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: shared reader: %w", err)
	}
	defer release()

	// COALESCE(present_at_completion, TRUE) : les lignes antérieures au backfill
	// de présence n'ont pas la colonne renseignée — un participant enregistré est
	// alors réputé présent (même lecture optimiste que l'effectif affiché).
	// gamertag : sert à la résolution des coéquipiers suivis (contexte escouade).
	q := `SELECT match_id, xuid, COALESCE(gamertag, ''), team_id,
	             COALESCE(present_at_completion, TRUE)
	      FROM match_participants
	      WHERE match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: participants query: %w", err)
	}
	defer rows.Close()
	var out []sessionusage.ParticipantRow
	for rows.Next() {
		var p sessionusage.ParticipantRow
		if err := rows.Scan(&p.MatchID, &p.XUID, &p.Gamertag, &p.TeamID, &p.PresentAtCompletion); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: participants scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// countMapFromJSON décode une ventilation {"clé":n,...}. "{}" (la forme vide du
// persister — jamais NULL) rend nil : « aucun » sans allocation.
func countMapFromJSON(raw string) (map[string]int, error) {
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var m map[string]int
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return m, nil
}
