// Package duckdb — solo_lives_repo.go : les vies d'un ou de plusieurs joueurs pour la carte
// « Isolement » (Séries temporelles, Sessions : un joueur ; Vue match : chaque joueur de l'équipe) —
// port.SoloLivesRepository, port.CampLivesRepository.
//
// UN CHARGEMENT, CINQ REQUÊTES BORNÉES (ADR 0036 I2) : chaque vue `_latest` lue reçoit la liste
// des matchs liée en UNE constante `VARCHAR[]` sur son propre `match_id` (clauseListeMatchs) — un
// filtre posé sur une vue ne traverse ni la fenêtre d'une autre vue, ni une jointure. Les joueurs
// sont filtrés ensuite (un filtre sur un joueur ne descend pas sous la fenêtre). Vues `_latest`
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

// Les prédicats de liste sont posés par LoadLivesNearTeammateForPlayers : la liste des matchs (%s,
// sous la fenêtre de la vue), puis la liste des joueurs (%s, après la fenêtre). Le xuid sort en tête
// de chaque ligne propre à un joueur.
const (
	qSoloLives = `
SELECT l.xuid, l.match_id, l.start_ms, l.end_cause
FROM match_lives_latest l
WHERE %s AND %s
ORDER BY l.xuid, l.match_id, l.start_ms`

	qSoloMorts = `
SELECT c.victim_xuid, c.match_id, c.time_ms, c.nearest_teammate_m
FROM match_death_context_latest c
WHERE %s AND %s
ORDER BY c.victim_xuid, c.match_id, c.time_ms`

	// Les camps du tueur et de la victime : LEFT JOIN, une victime sans ligne (bot) garde un camp
	// NULL — le calcul écarte alors le frag, il ne le devine pas.
	qSoloFrags = `
SELECT e.feed_killer_xuid, e.match_id, e.time_ms, pk.team_id, pv.team_id
FROM match_kill_events_latest e
LEFT JOIN match_participants pk ON pk.match_id = e.match_id AND pk.xuid = e.feed_killer_xuid
LEFT JOIN match_participants pv ON pv.match_id = e.match_id AND pv.xuid = e.victim_xuid
WHERE %s AND e.publishable AND %s
ORDER BY e.feed_killer_xuid, e.match_id, e.time_ms`

	// Les matchs dont la dernière passe du journal n'est pas publiable : le drapeau est posé pour
	// la passe entière, une ligne suffit. Tous joueurs confondus — la publiabilité est celle du
	// journal, pas d'un tueur.
	qSoloJournalNonPubliable = `
SELECT DISTINCT e.match_id
FROM match_kill_events_latest e
WHERE %s AND NOT e.publishable`

	qSoloVariantes = `
SELECT mr.match_id, COALESCE(mr.game_variant_name, '')
FROM match_registry mr
WHERE %s`
)

// LoadLivesNearTeammate rend les vies, morts situées et frags de `xuid` sur `matchIDs`, les matchs
// au journal non publiable et la variante de chaque match. Liste ou joueur vide : rien à lire.
// Table absente : games.ErrCapabilityNotSupported.
func (r *SoloLivesRepo) LoadLivesNearTeammate(ctx context.Context, matchIDs []string, xuid string) (domain.ViesLues, error) {
	if len(matchIDs) == 0 || xuid == "" {
		return domain.ViesLues{Variantes: map[string]string{}}, nil
	}
	parJoueur, err := r.LoadLivesNearTeammateForPlayers(ctx, matchIDs, []string{xuid})
	return parJoueur[xuid], err
}

// LoadLivesNearTeammateForPlayers rend la lecture de CHAQUE joueur de `xuids` sur `matchIDs` (une
// entrée par joueur demandé, vide s'il n'a aucune ligne) : ses vies, morts situées et frags ; le
// journal non publiable et les variantes, communs à tous, sont posés sur chaque entrée. UNE lecture
// pour tous les joueurs (ADR 0036 I4). Liste vide : rien à lire. Table absente :
// games.ErrCapabilityNotSupported.
func (r *SoloLivesRepo) LoadLivesNearTeammateForPlayers(ctx context.Context, matchIDs, xuids []string) (map[string]domain.ViesLues, error) {
	out := make(map[string]domain.ViesLues, len(xuids))
	commun := domain.ViesLues{Variantes: map[string]string{}}
	for _, x := range xuids {
		out[x] = commun
	}
	if len(matchIDs) == 0 || len(xuids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, soloLivesQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("SoloLivesRepo: shared reader: %w", err)
	}
	defer release()
	lues := map[string]*domain.ViesLues{}
	of := func(xuid string) *domain.ViesLues {
		if lues[xuid] == nil {
			lues[xuid] = &domain.ViesLues{}
		}
		return lues[xuid]
	}
	if err := r.lireParJoueur(ctx, db, matchIDs, xuids, of); err != nil {
		return out, err
	}
	if err := r.lireJournalNonPubliable(ctx, db, matchIDs, &commun); err != nil {
		return out, err
	}
	clause, liste := clauseListeParJointure("mr.match_id", matchIDs)
	if err := r.lire(ctx, db, fmt.Sprintf(qSoloVariantes, clause), []any{liste}, func(rows *sql.Rows) error {
		var id, variante string
		err := rows.Scan(&id, &variante)
		commun.Variantes[id] = variante
		return err
	}); err != nil {
		return out, err
	}
	for _, x := range xuids {
		v := commun
		if l := lues[x]; l != nil {
			v.Vies, v.Morts, v.Frags = l.Vies, l.Morts, l.Frags
		}
		out[x] = v
	}
	return out, nil
}

// lireParJoueur lit les vies, les morts situées et les frags des joueurs `xuids`, rangés par joueur.
func (r *SoloLivesRepo) lireParJoueur(
	ctx context.Context, db *sql.DB, matchIDs, xuids []string, of func(xuid string) *domain.ViesLues,
) error {
	lire := func(gabarit, colMatch, colJoueur, extra string, scan func(*sql.Rows) error) error {
		clauseMatchs, listeMatchs := clauseListeMatchs(colMatch, matchIDs)
		clauseJoueurs, listeJoueurs := clauseListeParJointure(colJoueur, xuids)
		return r.lire(ctx, db, fmt.Sprintf(gabarit, clauseMatchs+extra, clauseJoueurs), []any{listeMatchs, listeJoueurs}, scan)
	}
	if err := lire(qSoloLives, "l.match_id", "l.xuid", "", func(rows *sql.Rows) error {
		var xuid string
		var v domain.VieLue
		if err := rows.Scan(&xuid, &v.MatchID, &v.StartMS, &v.EndCause); err != nil {
			return err
		}
		of(xuid).Vies = append(of(xuid).Vies, v)
		return nil
	}); err != nil {
		return err
	}
	if err := lire(qSoloMorts, "c.match_id", "c.victim_xuid", "", func(rows *sql.Rows) error { return scanMortSituee(rows, of) }); err != nil {
		return err
	}
	// La Campagne est exclue (D-5) : la fenêtre de la page l'exclut déjà, la lecture ne s'y fie pas.
	exclusion := excludeCampaignByMatchID(pdbTitleSlug(r.pdb), "e.match_id")
	return lire(qSoloFrags, "e.match_id", "e.feed_killer_xuid", exclusion, func(rows *sql.Rows) error { return scanFragLu(rows, of) })
}

// lireJournalNonPubliable nomme les matchs de la liste dont la dernière passe du journal des morts
// n'est pas publiable (liste liée sur la vue, ADR 0036 I2).
func (r *SoloLivesRepo) lireJournalNonPubliable(ctx context.Context, db *sql.DB, matchIDs []string, out *domain.ViesLues) error {
	clause, liste := clauseListeMatchs("e.match_id", matchIDs)
	return r.lire(ctx, db, fmt.Sprintf(qSoloJournalNonPubliable, clause), []any{liste}, func(rows *sql.Rows) error {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if out.JournalNonPubliable == nil {
			out.JournalNonPubliable = map[string]bool{}
		}
		out.JournalNonPubliable[id] = true
		return nil
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

// scanMortSituee lit une mort et la range chez sa victime ; une distance NULL (aucun coéquipier
// visible) reste nil.
func scanMortSituee(rows *sql.Rows, of func(xuid string) *domain.ViesLues) error {
	var xuid string
	var m domain.MortSituee
	var proche sql.NullFloat64
	if err := rows.Scan(&xuid, &m.MatchID, &m.TimeMS, &proche); err != nil {
		return err
	}
	if proche.Valid {
		v := proche.Float64
		m.PlusProcheM = &v
	}
	out := of(xuid)
	out.Morts = append(out.Morts, m)
	return nil
}

// scanFragLu lit un frag et le range chez son tueur ; un camp NULL reste nil.
func scanFragLu(rows *sql.Rows, of func(xuid string) *domain.ViesLues) error {
	var xuid string
	var f domain.FragLu
	var tueur, victime sql.NullInt64
	if err := rows.Scan(&xuid, &f.MatchID, &f.TimeMS, &tueur, &victime); err != nil {
		return err
	}
	out := of(xuid)
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
