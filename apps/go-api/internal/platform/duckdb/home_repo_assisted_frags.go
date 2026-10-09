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
// filtrée par joueur et par lot de matchs (au plus quelques dizaines d'identifiants).
const homeAssistedFragsTimeout = 10 * time.Second

// Q26lHomeAssistedFragsTpl : pour chaque match du lot dont le film PORTE l'assistance
// (`porteuses` : au moins une ligne `publishable AND assist_known`, tous tueurs
// confondus), les frags du joueur lus par le film et, parmi eux, ceux dont l'assistant est
// NOMMÉ (sqlAssistantNomme : bot ou joueur non résolu compris), par tranche de part.
//
// LIGNES DU JOUEUR : toutes ses lignes publiables de tueur, victime bot comprise. Une ligne
// dont l'assistance n'est pas lue n'entre dans aucun compte d'assistés ; elle compte dans
// `frags_film`, plancher de la base officielle posée par le service
// (domain.MatchAssistedFrags.WithOfficialFrags).
//
// Une assistance sans part mesurée (`assist_damage_pct` NULL) compte dans le total et dans
// aucune tranche : les comparaisons à NULL sont fausses dans les FILTER.
//
// Un match hors de `porteuses`, ou sans ligne de tueur pour le joueur, ne sort pas :
// l'appelant ne publie rien pour lui, jamais un « 0 ».
//
// Format string, DANS CET ORDRE : placeholders du lot de matchs, puis le prédicat
// d'assistant nommé et les bornes de tranche (low max, mid min, mid max, high min), chacune
// précédée du prédicat. Placeholders ? : les match_id, puis le xuid du joueur.
const Q26lHomeAssistedFragsTpl = `
WITH porteuses AS (
    SELECT DISTINCT match_id
    FROM ` + KillEventsCanonicalTable + `
    WHERE publishable AND assist_known AND match_id IN (%s)
)
SELECT kv.match_id,
       COUNT(*)                                                                                     AS frags_film,
       COUNT(*) FILTER (WHERE %s)                                                                   AS received_total,
       COUNT(*) FILTER (WHERE %s AND kv.assist_damage_pct < %d)                                     AS received_low,
       COUNT(*) FILTER (WHERE %s AND kv.assist_damage_pct >= %d AND kv.assist_damage_pct <= %d)     AS received_mid,
       COUNT(*) FILTER (WHERE %s AND kv.assist_damage_pct > %d)                                     AS received_high
FROM ` + KillEventsCanonicalTable + ` kv
WHERE kv.publishable
  AND kv.match_id IN (SELECT match_id FROM porteuses)
  AND kv.feed_killer_xuid = ?
GROUP BY kv.match_id`

// buildHomeAssistedFragsQuery assemble Q26l et ses arguments. Retourne ("", nil) sur un
// lot vide.
func buildHomeAssistedFragsQuery(xuid string, matchIDs []string) (string, []any) {
	if len(matchIDs) == 0 {
		return "", nil
	}
	low, mid := domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct
	nomme := sqlAssistantNomme("kv")
	sqlText := fmt.Sprintf(Q26lHomeAssistedFragsTpl, Placeholders(len(matchIDs)),
		nomme, nomme, low, nomme, low, mid, nomme, mid)
	args := append(ToAnySlice(matchIDs), xuid)
	return sqlText, args
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
