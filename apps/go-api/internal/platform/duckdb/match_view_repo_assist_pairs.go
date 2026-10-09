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
	"time"

	"levelup/go-api/internal/domain"
)

// Q21dAssistPairs : les paires (assistant, tueur assisté) d'un match, ET les dénominateurs qui
// rendent une liste vide lisible.
//
// SŒUR de Q21b/Q21c (queries_match.go), mais d'une NATURE différente : celles-là rendent
// une ligne PAR MORT pour décorer le feed et s'apparient aux events par (tueur, instant) ;
// celle-ci rend un AGRÉGAT PAR MATCH. Conséquence directe et voulue : aucune clé
// temporelle ne sort d'ici, donc RIEN à recaler sur T0. Un agrégat par match_id ne
// s'apparie à rien — la correction T0 n'aurait ni objet ni prise.
//
// ─── LES DÉNOMINATEURS ────────────────────────────────────────────────────────────────
//
//	match_deaths     toutes les lignes du match, portées confondues. Zéro = le match n'est
//	                 jamais passé au décodeur de film (ou le titre n'en a pas). Le service
//	                 n'émet alors aucun bloc : il n'y a rien à dire.
//	measured_deaths  les lignes `publishable AND assist_known` — les morts dont
//	                 l'assistance est MESURÉE et publiable ligne à ligne. Zéro = le film du
//	                 match ne porte pas l'assistance (ou la passe n'est pas publiable ligne
//	                 à ligne, cas BTB) : le service n'émet pas de bloc.
//	publishable_deaths les lignes `publishable`, assistance connue ou non : le journal des
//	                 morts se lit ligne à ligne. Lu par la Vue match (frags pendant l'effet
//	                 d'un bonus), pas par les paires.
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
//	assistant nommé              sqlAssistantNomme : xuid OU gamertag d'assistant — le seul
//	                             des trois états qu'une paire sait représenter. Un BOT
//	                             assistant (gamertag seul) en est.
//	tueur nommé                  xuid OU gamertag de tueur : un BOT tueur (gamertag seul)
//	                             reçoit sa paire comme les autres.
//
// Un acteur sans xuid sort avec un xuid NULL (chaîne vide côté Go) ET son gamertag de
// film : le groupement porte sur les deux, donc deux bots ne fusionnent jamais.
//
// `stolen_count` compte les morts où la part de dégâts de l'assistant DÉPASSE celle du
// tueur crédité. Les deux parts sont NULLABLES : `>` sur un NULL rend NULL, donc FALSE au
// FILTER — une mort dont une part manque n'est jamais comptée volée, et n'est jamais
// comptée « non volée » à tort non plus, puisqu'elle reste dans `assist_count`. Aucune des
// deux parts n'est plafonnée (mesures jusqu'à 228) : la comparaison porte sur l'ordre.
//
// `avg_assist_pct` : la PART MOYENNE de participation de l'assistant sur les morts de la
// paire, arrondie à l'entier. AVG ignore nativement les parts NULL ; si AUCUNE mort de la
// paire ne porte de part, la colonne sort NULL et le champ publié reste absent — jamais un
// « 0 % » fabriqué. Même vocabulaire que le kill feed du rejeu (« part »), et même refus
// de plafonner que `stolen_count` (mesures jusqu'à 228) : c'est une moyenne de parts
// mesurées, pas un dégât chiffré.
//
// Paramètres : ?1 = match_id (portée), ?2 = match_id (paires). Retourne 10 colonnes :
// match_deaths, measured_deaths, publishable_deaths, assist_xuid, assist_gamertag,
// feed_killer_xuid, feed_killer_gamertag, assist_count, stolen_count, avg_assist_pct — les
// sept dernières NULL quand aucune paire ne sort.
const Q21dAssistPairs = `
WITH scope AS (
    SELECT
        COUNT(*)                                             AS match_deaths,
        COUNT(*) FILTER (WHERE publishable AND assist_known)  AS measured_deaths,
        COUNT(*) FILTER (WHERE publishable)                   AS publishable_deaths
    FROM ` + KillEventsCanonicalTable + `
    WHERE match_id = ?
),
pairs AS (
    SELECT
        kv.assist_xuid,
        kv.assist_gamertag,
        kv.feed_killer_xuid,
        kv.feed_killer_gamertag,
        COUNT(*)                                                             AS assist_count,
        COUNT(*) FILTER (WHERE kv.assist_damage_pct > kv.killer_damage_pct)  AS stolen_count,
        CAST(ROUND(AVG(kv.assist_damage_pct)) AS INTEGER)                    AS avg_assist_pct
    FROM ` + KillEventsCanonicalTable + ` kv
    WHERE kv.match_id = ?
      AND kv.publishable
      AND kv.assist_known
      AND (kv.assist_xuid IS NOT NULL OR kv.assist_gamertag IS NOT NULL)
      AND (kv.feed_killer_xuid IS NOT NULL OR kv.feed_killer_gamertag IS NOT NULL)
    GROUP BY kv.assist_xuid, kv.assist_gamertag, kv.feed_killer_xuid, kv.feed_killer_gamertag
)
SELECT
    s.match_deaths,
    s.measured_deaths,
    s.publishable_deaths,
    p.assist_xuid,
    p.assist_gamertag,
    p.feed_killer_xuid,
    p.feed_killer_gamertag,
    p.assist_count,
    p.stolen_count,
    p.avg_assist_pct
FROM scope s
LEFT JOIN pairs p ON TRUE
ORDER BY p.assist_count DESC, p.assist_gamertag, p.feed_killer_xuid, p.feed_killer_gamertag`

// GetMatchAssistPairs retourne les paires (assistant, tueur assisté) du match et la portée
// de leur lecture (Q21d). Exécutée sur SharedReader (ADR 0016, shared-only).
//
// UN ÉCHEC DE LECTURE EST UNE ERREUR, JAMAIS UNE PORTÉE VIDE : lecteur partagé indisponible,
// table absente d'une base non migrée, délai dépassé ou contexte annulé remontent enveloppés.
// Une portée à zéro rendue sur échec se lirait « aucune mort publiable » (journal des morts
// non publiable) alors que la lecture a seulement manqué ; l'appelant journalise l'erreur,
// dégrade, et la page dit « lecture indisponible ». Un match sans ligne de film rend, lui, une
// portée nulle sans erreur : c'est un résultat.
func (r *MatchViewRepo) GetMatchAssistPairs(
	ctx context.Context,
	matchID string,
) ([]domain.MatchAssistPairRaw, domain.MatchAssistScopeRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var scope domain.MatchAssistScopeRaw

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, scope, fmt.Errorf("MatchViewRepo.GetMatchAssistPairs: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q21dAssistPairs, matchID, matchID)
	if err != nil {
		return nil, scope, fmt.Errorf("MatchViewRepo.GetMatchAssistPairs: %w", err)
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
			matchDeaths, measured    int
			publishable              int
			ax, agt, kx, kgt         sql.NullString
			assistN, stolenN, avgPct sql.NullInt64
		)
		if err := rows.Scan(&matchDeaths, &measured, &publishable, &ax, &agt, &kx, &kgt, &assistN, &stolenN, &avgPct); err != nil {
			return nil, domain.MatchAssistScopeRaw{}, fmt.Errorf("MatchViewRepo.GetMatchAssistPairs scan: %w", err)
		}
		scope.MatchDeaths = matchDeaths
		scope.MeasuredDeaths = measured
		scope.PublishableDeaths = publishable
		// Ligne de portée SEULE (aucune paire) : le LEFT JOIN ON TRUE laisse les colonnes
		// de paire à NULL, compte compris (une paire sortie a un compte, jamais NULL).
		// C'est l'état « mesuré, zéro assistant nommé » : on garde la portée et on
		// n'invente pas de paire. Les xuids d'une paire, eux, peuvent être NULL un à un
		// (acteur sans xuid, nommé par son gamertag) : chaîne vide côté Go.
		if !assistN.Valid {
			continue
		}
		pair := domain.MatchAssistPairRaw{
			AssistXUID:     ax.String,
			AssistGamertag: agt.String,
			KillerXUID:     kx.String,
			KillerGamertag: kgt.String,
			AssistCount:    int(assistN.Int64),
			StolenCount:    int(stolenN.Int64),
		}
		if avgPct.Valid {
			v := int(avgPct.Int64)
			pair.AvgAssistPct = &v
		}
		results = append(results, pair)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.MatchAssistScopeRaw{}, err
	}
	return results, scope, nil
}
