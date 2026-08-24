// Package duckdb — match_view_repo_assist_pairs.go : Q21d, les paires d'assistance d'un
// match (ASSISTANT -> TUEUR ASSISTÉ) et la PORTÉE de leur lecture.
//
// Fichier dédié plutôt qu'un ajout à queries_match.go / match_view_repo_extras.go : les
// deux sont déjà au-delà du seuil des 500 lignes (626 et 530), et la règle interdit
// d'accroître la dette gelée. La requête vit donc à côté de son unique lecteur.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// Q21dAssistPairs : les paires (assistant, tueur assisté) d'un match, ET les deux
// dénominateurs qui rendent une liste vide lisible.
//
// SŒUR de Q21b/Q21c (queries_match.go), mais d'une NATURE différente : celles-là rendent
// une ligne PAR MORT pour décorer le feed et s'apparient aux events par (tueur, instant) ;
// celle-ci rend un AGRÉGAT PAR MATCH. Conséquence directe et voulue : aucune clé
// temporelle ne sort d'ici, donc RIEN à recaler sur T0. Un agrégat par match_id ne
// s'apparie à rien — la correction T0 n'aurait ni objet ni prise.
//
// ─── LES DEUX DÉNOMINATEURS, ET POURQUOI DEUX ─────────────────────────────────────────
//
//	match_deaths     toutes les lignes du match, portées confondues. Zéro = le match n'est
//	                 jamais passé au décodeur de film (ou le titre n'en a pas). Le service
//	                 n'émet alors aucun bloc : il n'y a rien à dire.
//	measured_deaths  les lignes `publishable AND assist_known` — les morts dont
//	                 l'assistance est MESURÉE et publiable ligne à ligne. Zéro alors que
//	                 match_deaths > 0 = « non mesuré pour ce match » (le film est là,
//	                 l'assistance non — ou la passe n'est pas publiable ligne à ligne,
//	                 cas BTB). C'est un état à AFFICHER, jamais « aucune assistance ».
//
// ─── LA JOINTURE SUR TRUE N'EST PAS UNE COQUETTERIE ───────────────────────────────────
//
// `scope` rend TOUJOURS une ligne (c'est un COUNT sans GROUP BY) ; `pairs` peut n'en
// rendre aucune. Un `LEFT JOIN ... ON TRUE` fait donc sortir les dénominateurs MÊME
// quand il n'y a pas une seule paire — et c'est précisément le cas que le bloc doit
// savoir distinguer. Une requête qui ne rendrait que les paires perdrait l'information
// « mesuré, zéro assistant » au moment exact où elle est nécessaire.
//
// ─── LES FILTRES DE LA LISTE DE PAIRES ────────────────────────────────────────────────
//
//	publishable                  une paire nomme DEUX joueurs : c'est une lecture ligne à
//	                             ligne, pas un cumul anonyme. La portée l'exige.
//	assist_known                 sans lui on compterait des « pas d'assistant » jamais
//	                             observés.
//	assist_gamertag/xuid NOT NULL l'assistant nommé — le seul des trois états qu'une paire
//	                             sait représenter.
//
// `stolen_count` compte les morts où la part de dégâts de l'assistant DÉPASSE celle du
// tueur crédité. Les deux parts sont NULLABLES : `>` sur un NULL rend NULL, donc FALSE au
// FILTER — une mort dont une part manque n'est jamais comptée volée, et n'est jamais
// comptée « non volée » à tort non plus, puisqu'elle reste dans `assist_count`. Aucune des
// deux parts n'est plafonnée (mesures jusqu'à 228) : la comparaison porte sur l'ordre.
//
// Paramètres : ?1 = match_id (portée), ?2 = match_id (paires). Retourne 6 colonnes :
// match_deaths, measured_deaths, assist_xuid, assist_gamertag, feed_killer_xuid,
// assist_count, stolen_count — les cinq dernières NULL quand aucune paire ne sort.
const Q21dAssistPairs = `
WITH scope AS (
    SELECT
        COUNT(*)                                             AS match_deaths,
        COUNT(*) FILTER (WHERE publishable AND assist_known)  AS measured_deaths
    FROM ` + KillEventsCanonicalTable + `
    WHERE match_id = ?
),
pairs AS (
    SELECT
        assist_xuid,
        assist_gamertag,
        feed_killer_xuid,
        COUNT(*)                                                          AS assist_count,
        COUNT(*) FILTER (WHERE assist_damage_pct > killer_damage_pct)     AS stolen_count
    FROM ` + KillEventsCanonicalTable + `
    WHERE match_id = ?
      AND publishable
      AND assist_known
      AND assist_gamertag  IS NOT NULL
      AND assist_xuid      IS NOT NULL
      AND feed_killer_xuid IS NOT NULL
    GROUP BY assist_xuid, assist_gamertag, feed_killer_xuid
)
SELECT
    s.match_deaths,
    s.measured_deaths,
    p.assist_xuid,
    p.assist_gamertag,
    p.feed_killer_xuid,
    p.assist_count,
    p.stolen_count
FROM scope s
LEFT JOIN pairs p ON TRUE
ORDER BY p.assist_count DESC, p.assist_gamertag, p.feed_killer_xuid`

// GetMatchAssistPairs retourne les paires (assistant, tueur assisté) du match et la portée
// de leur lecture (Q21d). Exécutée sur SharedReader (ADR 0016, shared-only).
//
// Même dégradation gracieuse que Q21b/Q21c, et pour les mêmes raisons : reader
// indisponible ou table absente d'une base non migrée rendent une portée VIDE
// (MatchDeaths = 0), loggée. Le service n'émet alors aucun bloc et l'écran ne rend rien —
// l'état d'avant ce lot. Jamais une erreur : un titre sans décodeur de film n'est pas une
// panne.
func (r *MatchViewRepo) GetMatchAssistPairs(
	ctx context.Context,
	matchID string,
) ([]domain.MatchAssistPairRaw, domain.MatchAssistScopeRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var scope domain.MatchAssistScopeRaw

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "match_view: paires d'assistance indisponibles (shared reader)",
			"match_id", matchID, "err", err)
		return nil, scope, nil
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q21dAssistPairs, matchID, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match_view: paires d'assistance indisponibles (Q21d)",
			"match_id", matchID, "err", err)
		return nil, scope, nil
	}
	defer rows.Close()
	return scanAssistPairs(rows)
}

// scanAssistPairs lit le résultat de Q21d. Séparé du lecteur pour être testable sur une
// base DuckDB en mémoire, sans provider ni bail : c'est LA REQUÊTE et sa lecture que le
// test doit prouver, pas le routage.
func scanAssistPairs(rows *sql.Rows) ([]domain.MatchAssistPairRaw, domain.MatchAssistScopeRaw, error) {
	var (
		scope   domain.MatchAssistScopeRaw
		results []domain.MatchAssistPairRaw
	)
	for rows.Next() {
		var (
			matchDeaths, measured int
			ax, agt, kx           sql.NullString
			assistN, stolenN      sql.NullInt64
		)
		if err := rows.Scan(&matchDeaths, &measured, &ax, &agt, &kx, &assistN, &stolenN); err != nil {
			return nil, domain.MatchAssistScopeRaw{}, fmt.Errorf("MatchViewRepo.GetMatchAssistPairs scan: %w", err)
		}
		scope.MatchDeaths = matchDeaths
		scope.MeasuredDeaths = measured
		// Ligne de portée SEULE (aucune paire) : le LEFT JOIN ON TRUE laisse les cinq
		// colonnes de paire à NULL. C'est l'état « mesuré, zéro assistant nommé » —
		// on garde la portée et on n'invente pas de paire.
		if !ax.Valid || !agt.Valid || !kx.Valid {
			continue
		}
		results = append(results, domain.MatchAssistPairRaw{
			AssistXUID:     ax.String,
			AssistGamertag: agt.String,
			KillerXUID:     kx.String,
			AssistCount:    int(assistN.Int64),
			StolenCount:    int(stolenN.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, domain.MatchAssistScopeRaw{}, err
	}
	return results, scope, nil
}
