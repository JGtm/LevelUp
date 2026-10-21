// Package duckdb — home_repo_assisted_frags.go : part des frags du joueur assistés par
// un coéquipier, par match, pour les tuiles de match de l'Accueil (Q26l).
//
// Sous-module de home_repo.go. Même vocabulaire et mêmes bornes que Q28c
// (relation_assists_repo.go) — le sens « reçues » de la page Relations, ramené à un
// seul match. Règle des bases : domain/relation_assists.go.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// homeAssistedFragsTimeout borne la lecture Q26l : une agrégation sur la vue des morts
// bornée sous sa fenêtre au lot de matchs de l'Accueil (tout l'historique du joueur, plus
// d'un millier de matchs pour les plus anciens comptes).
const homeAssistedFragsTimeout = 10 * time.Second

// Q26lHomeAssistedFragsTpl : pour chaque match du lot dont le film PORTE l'assistance
// (`porteuses` : tête de résumé publiable avec au moins une mort à assistance connue,
// `match_kill_heads_latest`, ADR 0039), les frags du joueur lus par le film (`film`, S1
// `match_kill_pairs_latest` : toutes ses morts de tueur de la passe publiable, victime bot
// comprise) et, parmi eux, ceux dont l'assistant est NOMMÉ (`recus`, S2 `match_assist_pairs_latest` :
// bot ou joueur non résolu compris), par tranche de part.
//
// Une ligne dont l'assistance n'est pas lue n'entre dans aucun compte d'assistés ; elle compte dans
// `frags_film`, plancher de la base officielle posée par le service
// (domain.MatchAssistedFrags.WithOfficialFrags). Une assistance sans part mesurée compte dans le
// total et dans aucune tranche (`n_low` / `n_mid` / `n_high` ne portent que les parts mesurées).
//
// Un match hors de `porteuses`, ou sans ligne de tueur pour le joueur, ne sort pas : l'appelant ne
// publie rien pour lui, jamais un « 0 ». Un match porteur dont le joueur n'a aucun appui nommé sort
// avec des assistés à zéro (`LEFT JOIN`).
//
// PÉRIMÈTRE (ADR 0036 I2, ADR 0039 décision 7) : chaque lecture d'un résumé porte le lot en
// prédicat CONSTANT sur son `match_id` (clauseListeMatchs). Placeholders ? : le lot (porteuses), le
// lot et le xuid du joueur (film), le lot et le xuid du joueur (recus) ; trois formats `%s` dans cet
// ordre : le prédicat de lot de `porteuses`, de `film`, de `recus`.
const Q26lHomeAssistedFragsTpl = `
WITH porteuses AS (
    SELECT match_id
    FROM match_kill_heads_latest
    WHERE publishable AND n_assist_known > 0 AND %s
),
film AS (
    SELECT k.match_id, CAST(SUM(k.kills) AS BIGINT) AS frags_film
    FROM match_kill_pairs_latest k
    WHERE k.publishable
      AND %s
      AND k.match_id IN (SELECT match_id FROM porteuses)
      AND k.feed_killer_xuid = ?
    GROUP BY k.match_id
),
recus AS (
    SELECT a.match_id,
           CAST(SUM(a.n)      AS BIGINT) AS received_total,
           CAST(SUM(a.n_low)  AS BIGINT) AS received_low,
           CAST(SUM(a.n_mid)  AS BIGINT) AS received_mid,
           CAST(SUM(a.n_high) AS BIGINT) AS received_high
    FROM match_assist_pairs_latest a
    WHERE %s
      AND a.match_id IN (SELECT match_id FROM porteuses)
      AND a.feed_killer_xuid = ?
      AND (a.assist_xuid IS NOT NULL OR a.assist_gamertag IS NOT NULL)
    GROUP BY a.match_id
)
SELECT f.match_id,
       f.frags_film,
       COALESCE(r.received_total, 0) AS received_total,
       COALESCE(r.received_low, 0)   AS received_low,
       COALESCE(r.received_mid, 0)   AS received_mid,
       COALESCE(r.received_high, 0)  AS received_high
FROM film f
LEFT JOIN recus r ON r.match_id = f.match_id`

// buildHomeAssistedFragsQuery assemble Q26l et ses arguments. Retourne ("", nil) sur un
// lot vide.
func buildHomeAssistedFragsQuery(xuid string, matchIDs []string) (string, []any) {
	if len(matchIDs) == 0 {
		return "", nil
	}
	lotTete, argTete := clauseListeMatchs("match_id", matchIDs)
	lotFilm, argFilm := clauseListeMatchs("k.match_id", matchIDs)
	lotRecus, argRecus := clauseListeMatchs("a.match_id", matchIDs)
	sqlText := fmt.Sprintf(Q26lHomeAssistedFragsTpl, lotTete, lotFilm, lotRecus)
	return sqlText, []any{argTete, argFilm, xuid, argRecus, xuid}
}

// LoadMatchAssistedFrags : part des frags du joueur assistés par un coéquipier, par match
// du lot (Q26l). Les matchs dont le film ne porte pas l'assistance, ou sans frag lu pour le
// joueur, sont absents de la map. FragsOfficial reste à poser par l'appelant
// (WithOfficialFrags). Erreur propagée (pas de dégradation silencieuse ici : l'appelant
// journalise).
func (r *HomeRepo) LoadMatchAssistedFrags(ctx context.Context, matchIDs []string) (map[string]domain.MatchAssistedFrags, error) {
	out := make(map[string]domain.MatchAssistedFrags, len(matchIDs))
	sqlText, args := buildHomeAssistedFragsQuery(r.pdb.XUID, matchIDs)
	if sqlText == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, homeAssistedFragsTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("HomeRepo.LoadMatchAssistedFrags: shared reader: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("HomeRepo.LoadMatchAssistedFrags (Q26l): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			matchID string
			a       domain.MatchAssistedFrags
		)
		if err := rows.Scan(&matchID, &a.FragsFilm,
			&a.Received.Total, &a.Received.Low, &a.Received.Mid, &a.Received.High); err != nil {
			return nil, fmt.Errorf("HomeRepo.LoadMatchAssistedFrags (Q26l) scan: %w", err)
		}
		out[matchID] = a
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("HomeRepo.LoadMatchAssistedFrags (Q26l) rows: %w", err)
	}
	return out, nil
}
