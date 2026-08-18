package duckdb

// replay_facts_repo.go — CE QUE LA BASE SAIT DU MATCH, POUR LE CONSTRUCTEUR D'ARTEFACT DE REJEU.
//
// POURQUOI CE REPO EXISTE. L'artefact de rejeu est décodé des SEULS chunks du film, et le film
// ne nomme personne : ses entités sont des slots. Deux ponts lui manquent donc, et tous deux
// vivent en base :
//
//	l'identité des JOUEURS   le triplet (frags, morts, assistances) de `match_participants`
//	                         apparie le slot d'entité au xuid — c'est la seule clé qui marche,
//	                         le numéro de slot n'en est pas une (deux espaces différents) ;
//	l'identité des CAMPS     `team_0_score` / `team_1_score` du registre désignent le camp de
//	                         chaque slot d'équipe quand ils diffèrent ; sinon c'est `team_id`
//	                         des participants, par la somme des frags.
//
// S'y ajoute `game_variant_name`, sans lequel aucune action d'objectif ne peut être NOMMÉE
// (la famille d'objectif se lit dans le nom de variante).
//
// LECTURE SEULE, COURTE, ET SANS AUCUNE ÉCRITURE : deux `SELECT` par match, les handles sont
// relâchés par l'appelant AVANT le décodage — on ne tient pas une lecture partagée pendant les
// dizaines de secondes que coûte un film.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"levelup/go-api/internal/port"
)

// ReplayFactsRepo lit les faits d'un match (registre + participants).
type ReplayFactsRepo struct {
	shared *sql.DB
}

// NewReplayFactsRepo crée le repo sur une connexion shared DÉJÀ ouverte (RO ou RW selon
// l'appelant : ce repo n'ouvre ni ne ferme rien).
func NewReplayFactsRepo(shared *sql.DB) *ReplayFactsRepo {
	return &ReplayFactsRepo{shared: shared}
}

// FactsForMatch retourne les faits du match.
//
// UN MATCH ABSENT DU REGISTRE N'EST PAS UNE ERREUR : il rend des faits VIDES, et l'appelant
// construit alors un artefact sans compteurs de joueur ni actions d'objectif. Le cas est réel
// (un film du cache dont le match n'a jamais été synchronisé) et il ne doit pas faire échouer
// une passe de construction.
func (r *ReplayFactsRepo) FactsForMatch(ctx context.Context, matchID string) (port.MatchFacts, error) {
	if r == nil || r.shared == nil {
		return port.MatchFacts{}, nil
	}
	facts, err := r.registryFacts(ctx, matchID)
	if err != nil {
		return port.MatchFacts{}, err
	}
	players, err := r.playerFacts(ctx, matchID)
	if err != nil {
		return port.MatchFacts{}, err
	}
	facts.Players = players
	return facts, nil
}

// registryFacts lit la ligne de registre : les scores des deux camps et le nom de variante.
func (r *ReplayFactsRepo) registryFacts(ctx context.Context, matchID string) (port.MatchFacts, error) {
	var s0, s1 sql.NullInt64
	var variant sql.NullString
	err := r.shared.QueryRowContext(ctx,
		`SELECT team_0_score, team_1_score, game_variant_name FROM match_registry WHERE match_id = ?`,
		matchID).Scan(&s0, &s1, &variant)
	if errors.Is(err, sql.ErrNoRows) {
		return port.MatchFacts{}, nil
	}
	if err != nil {
		return port.MatchFacts{}, fmt.Errorf("faits de rejeu : lecture match_registry : %w", err)
	}
	out := port.MatchFacts{GameVariantName: variant.String}
	// LES DEUX SCORES OU AUCUN : un seul des deux ne permet aucune comparaison, et publier
	// l'autre à zéro inventerait un score.
	if s0.Valid && s1.Valid {
		out.TeamScores = &[2]int{int(s0.Int64), int(s1.Int64)}
	}
	return out, nil
}

// playerFacts lit les lignes de match des participants.
func (r *ReplayFactsRepo) playerFacts(ctx context.Context, matchID string) ([]port.MatchPlayerFact, error) {
	rows, err := r.shared.QueryContext(ctx,
		`SELECT xuid, COALESCE(kills, 0), COALESCE(deaths, 0), COALESCE(assists, 0), COALESCE(team_id, -1)
		 FROM match_participants WHERE match_id = ? ORDER BY xuid`, matchID)
	if err != nil {
		return nil, fmt.Errorf("faits de rejeu : lecture match_participants : %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []port.MatchPlayerFact
	for rows.Next() {
		var p port.MatchPlayerFact
		if err := rows.Scan(&p.XUID, &p.Kills, &p.Deaths, &p.Assists, &p.TeamID); err != nil {
			return nil, fmt.Errorf("faits de rejeu : lecture match_participants (scan) : %w", err)
		}
		if p.XUID == "" {
			continue // sans xuid, la ligne ne peut apparier aucun slot
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
