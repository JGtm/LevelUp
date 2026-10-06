// Package duckdb — solo_lives_repo.go : les vies d'un joueur pour la carte « Mes vies : près d'un
// coéquipier ou seul » des Séries temporelles (plan
// `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L3.3 ; port.SoloLivesRepository).
//
// UN CHARGEMENT, QUATRE REQUÊTES BORNÉES (ADR 0036 I2) : chaque vue `_latest` lue reçoit la liste
// des matchs liée en UNE constante `VARCHAR[]` sur son propre `match_id` (clauseListeMatchs) — un
// filtre posé sur une vue ne traverse ni la fenêtre d'une autre vue, ni une jointure. Le joueur est
// filtré ensuite (une égalité sur un joueur ne descend pas sous la fenêtre). Vues `_latest`
// seulement (ADR 0026), jamais `v_gamertag_lookup` (I1) ; camps et variantes viennent de tables
// ordinaires (`match_participants`, `match_registry`).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// SoloLivesRepo lit les vies d'un joueur sur le SharedReader de son PlayerDB.
type SoloLivesRepo struct {
	pdb *PlayerDB
}

// NewSoloLivesRepo construit le repo à partir de la player DB (SharedReader).
func NewSoloLivesRepo(pdb *PlayerDB) *SoloLivesRepo {
	return &SoloLivesRepo{pdb: pdb}
}

const soloLivesQueryTimeout = 15 * time.Second

// Les prédicats de liste sont posés par LoadLivesNearTeammate (%s) ; le joueur est le paramètre
// qui suit la liste.
const (
	qSoloLives = `
SELECT l.match_id, l.start_ms, l.end_cause
FROM match_lives_latest l
WHERE %s AND l.xuid = ?
ORDER BY l.match_id, l.start_ms`

	qSoloMorts = `
SELECT c.match_id, c.time_ms, c.nearest_teammate_m
FROM match_death_context_latest c
WHERE %s AND c.victim_xuid = ?
ORDER BY c.match_id, c.time_ms`

	// Les camps du tueur et de la victime : LEFT JOIN, une victime sans ligne (bot) garde un camp
	// NULL — le calcul écarte alors le frag, il ne le devine pas.
	qSoloFrags = `
SELECT e.match_id, e.time_ms, pk.team_id, pv.team_id
FROM match_kill_events_latest e
LEFT JOIN match_participants pk ON pk.match_id = e.match_id AND pk.xuid = e.feed_killer_xuid
LEFT JOIN match_participants pv ON pv.match_id = e.match_id AND pv.xuid = e.victim_xuid
WHERE %s AND e.publishable AND e.feed_killer_xuid = ?
ORDER BY e.match_id, e.time_ms`

	qSoloVariantes = `
SELECT mr.match_id, COALESCE(mr.game_variant_name, '')
FROM match_registry mr
WHERE %s`
)

// LoadLivesNearTeammate rend les vies, morts situées et frags de `xuid` sur `matchIDs`, et la
// variante de chaque match. Liste ou joueur vide : rien à lire. Table absente :
// games.ErrCapabilityNotSupported.
func (r *SoloLivesRepo) LoadLivesNearTeammate(ctx context.Context, matchIDs []string, xuid string) (domain.ViesLues, error) {
	out := domain.ViesLues{Variantes: map[string]string{}}
	if len(matchIDs) == 0 || xuid == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, soloLivesQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("SoloLivesRepo: shared reader: %w", err)
	}
	defer release()
	lire := func(gabarit, col, extra string, scan func(*sql.Rows) error) error {
		clause, liste := clauseListeMatchs(col, matchIDs)
		return r.lire(ctx, db, fmt.Sprintf(gabarit, clause+extra), []any{liste, xuid}, scan)
	}
	if err := lire(qSoloLives, "l.match_id", "", func(rows *sql.Rows) error {
		var v domain.VieLue
		err := rows.Scan(&v.MatchID, &v.StartMS, &v.EndCause)
		out.Vies = append(out.Vies, v)
		return err
	}); err != nil {
		return out, err
	}
	if err := lire(qSoloMorts, "c.match_id", "", func(rows *sql.Rows) error { return scanMortSituee(rows, &out) }); err != nil {
		return out, err
	}
	// La Campagne est exclue (D-5) : la fenêtre de la page l'exclut déjà, la lecture ne s'y fie pas.
	exclusion := excludeCampaignByMatchID(pdbTitleSlug(r.pdb), "e.match_id")
	if err := lire(qSoloFrags, "e.match_id", exclusion, func(rows *sql.Rows) error { return scanFragLu(rows, &out) }); err != nil {
		return out, err
	}
	clause, liste := clauseListeParJointure("mr.match_id", matchIDs)
	return out, r.lire(ctx, db, fmt.Sprintf(qSoloVariantes, clause), []any{liste}, func(rows *sql.Rows) error {
		var id, variante string
		err := rows.Scan(&id, &variante)
		out.Variantes[id] = variante
		return err
	})
}

// lire exécute une requête et passe chaque ligne à `scan` ; table absente : capability absente.
func (r *SoloLivesRepo) lire(ctx context.Context, db *sql.DB, requete string, args []any, scan func(*sql.Rows) error) error {
	rows, err := db.QueryContext(ctx, requete, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "SoloLivesRepo: table des vies absente", "err", err)
			return games.ErrCapabilityNotSupported
		}
		return fmt.Errorf("SoloLivesRepo: query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return fmt.Errorf("SoloLivesRepo: scan: %w", err)
		}
	}
	return rows.Err()
}

// scanMortSituee lit une mort ; une distance NULL (aucun coéquipier visible) reste nil.
func scanMortSituee(rows *sql.Rows, out *domain.ViesLues) error {
	var m domain.MortSituee
	var proche sql.NullFloat64
	if err := rows.Scan(&m.MatchID, &m.TimeMS, &proche); err != nil {
		return err
	}
	if proche.Valid {
		v := proche.Float64
		m.PlusProcheM = &v
	}
	out.Morts = append(out.Morts, m)
	return nil
}

// scanFragLu lit un frag ; un camp NULL reste nil.
func scanFragLu(rows *sql.Rows, out *domain.ViesLues) error {
	var f domain.FragLu
	var tueur, victime sql.NullInt64
	if err := rows.Scan(&f.MatchID, &f.TimeMS, &tueur, &victime); err != nil {
		return err
	}
	if tueur.Valid {
		v := int(tueur.Int64)
		f.CampTueur = &v
	}
	if victime.Valid {
		v := int(victime.Int64)
		f.CampVictime = &v
	}
	out.Frags = append(out.Frags, f)
	return nil
}
