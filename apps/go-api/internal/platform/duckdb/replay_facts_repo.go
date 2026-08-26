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
	"strings"

	"levelup/go-api/internal/port"
)

// L'assertion fige le contrat : les appelants ne connaissent que l'interface du port, jamais
// ce type concret (même convention que port.ReplayMapNameRepo).
var _ port.ReplayFactsRepo = (*ReplayFactsRepo)(nil)

// Le même type sert la résolution des cibles de lien (notification « rejeux prêts ») : ce
// sont les deux mêmes tables, lues en lecture seule et courte. Un second repo aurait
// dupliqué l'ouverture, le contrat « n'ouvre ni ne ferme rien » et les pièges d'identité.
var _ port.ReplayLinkRepo = (*ReplayFactsRepo)(nil)

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

// registryFacts lit la ligne de registre : les scores des deux camps, le nom de variante et
// l'asset UGC de la carte (la clé du catalogue d'objectifs, cf. port.MatchFacts.MapID).
func (r *ReplayFactsRepo) registryFacts(ctx context.Context, matchID string) (port.MatchFacts, error) {
	var s0, s1 sql.NullInt64
	var variant, mapID sql.NullString
	err := r.shared.QueryRowContext(ctx,
		`SELECT team_0_score, team_1_score, game_variant_name, map_id FROM match_registry WHERE match_id = ?`,
		matchID).Scan(&s0, &s1, &variant, &mapID)
	if errors.Is(err, sql.ErrNoRows) {
		return port.MatchFacts{}, nil
	}
	if err != nil {
		return port.MatchFacts{}, fmt.Errorf("faits de rejeu : lecture match_registry : %w", err)
	}
	out := port.MatchFacts{GameVariantName: variant.String, MapID: strings.TrimSpace(mapID.String)}
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

// LinkTargetsForMatches résout, EN UNE REQUÊTE pour tout le lot, de quoi lier chaque match
// à sa page de rejeu : un joueur connu qui y a participé, et le nom de carte du registre.
//
// UNE JOINTURE À GAUCHE, ET C'EST LE POINT : un match sans participant connu doit quand
// même sortir, avec son nom de carte et un xuid vide. La notification affichera la ligne
// sans lien plutôt que d'escamoter le match.
//
// `MIN(xuid)` plutôt qu'un `LIMIT 1` implicite : quand plusieurs joueurs de l'instance ont
// joué le même match, le lien doit être STABLE d'un appel à l'autre (un ordre de lignes
// DuckDB ne l'est pas).
//
// Lecture SEULE et COURTE, sur un handle déjà ouvert par l'appelant (contrat du repo :
// il n'ouvre ni ne ferme rien).
func (r *ReplayFactsRepo) LinkTargetsForMatches(
	ctx context.Context, matchIDs, knownXUIDs []string,
) (map[string]port.ReplayLinkTarget, error) {
	out := map[string]port.ReplayLinkTarget{}
	if r == nil || r.shared == nil || len(matchIDs) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(matchIDs)+len(knownXUIDs))
	for _, id := range matchIDs {
		args = append(args, id)
	}
	join := ""
	if len(knownXUIDs) > 0 {
		for _, x := range knownXUIDs {
			args = append(args, x)
		}
		join = ` LEFT JOIN match_participants mp
			ON mp.match_id = mr.match_id AND mp.xuid IN (` + placeholders(len(knownXUIDs)) + `)`
	}
	xuidExpr := "NULL"
	if join != "" {
		xuidExpr = "MIN(mp.xuid)"
	}
	q := `SELECT mr.match_id, mr.map_name, ` + xuidExpr + `
		FROM match_registry mr` + join + `
		WHERE mr.match_id IN (` + placeholders(len(matchIDs)) + `)
		GROUP BY mr.match_id, mr.map_name`
	rows, err := r.shared.QueryContext(ctx, q, args...)
	if err != nil {
		return out, fmt.Errorf("cibles de lien de rejeu : lecture : %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id string
		var mapName, xuid sql.NullString
		if err := rows.Scan(&id, &mapName, &xuid); err != nil {
			return out, fmt.Errorf("cibles de lien de rejeu : lecture (scan) : %w", err)
		}
		out[id] = port.ReplayLinkTarget{
			MatchID: id,
			XUID:    strings.TrimSpace(xuid.String),
			MapName: strings.TrimSpace(mapName.String),
		}
	}
	return out, rows.Err()
}

// placeholders rend "?, ?, ?" pour n valeurs (n > 0 garanti par les appelants).
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
