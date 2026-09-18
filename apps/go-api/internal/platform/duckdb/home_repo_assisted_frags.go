// Package duckdb — home_repo_assisted_frags.go : part des frags du joueur assistés par
// un coéquipier, par match, pour les tuiles de match de l'Accueil (Q26l).
//
// Sous-module de home_repo.go. Même vocabulaire et mêmes bornes que Q28c
// (relation_assists_repo.go) — le sens « reçues » de la page Relations, ramené à un
// seul match. Doctrine : domain/relation_assists.go (MatchAssistedFrags).
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

// Q26lHomeAssistedFragsTpl : pour chaque match du lot, les frags MESURÉS du joueur
// (lignes `publishable AND assist_known` où il est le tueur) et, parmi eux, ceux qu'un
// coéquipier a assistés, par tranche de part de dégâts.
//
// PORTÉE DES LIGNES : identique à Q28c — les frags servent de DÉNOMINATEUR aux
// assistances, ils se lisent sur la même population de lignes. Une assistance sans part
// mesurée (`assist_damage_pct` NULL) compte dans le total et dans aucune tranche : les
// comparaisons à NULL sont fausses dans les FILTER.
//
// PAS D'EXCLUSION DES BOTS : le tueur est le joueur suivi lui-même (`feed_killer_xuid`
// = xuid du profil), jamais un bot. Q28c exclut les bots côté COÉQUIPIER parce qu'il
// énumère les partenaires ; ici l'assistant n'est pas énuméré, seulement compté.
//
// Un match sans ligne mesurée pour le joueur ne sort pas (GROUP BY sur les lignes
// filtrées) : l'appelant le lit comme « non mesuré », jamais comme « 0 ».
//
// Format string, DANS CET ORDRE : bornes de tranche (low max, mid min, mid max,
// high min), puis les placeholders du lot de matchs. Placeholders ? : ?1 = xuid du
// joueur, puis les match_id.
const Q26lHomeAssistedFragsTpl = `
SELECT kv.match_id,
       COUNT(*)              AS frags_measured,
       COUNT(kv.assist_xuid) AS received_total,
       COUNT(kv.assist_xuid) FILTER (WHERE kv.assist_damage_pct < %d)                                    AS received_low,
       COUNT(kv.assist_xuid) FILTER (WHERE kv.assist_damage_pct >= %d AND kv.assist_damage_pct <= %d)   AS received_mid,
       COUNT(kv.assist_xuid) FILTER (WHERE kv.assist_damage_pct > %d)                                    AS received_high
FROM ` + KillEventsCanonicalTable + ` kv
WHERE kv.publishable AND kv.assist_known
  AND kv.feed_killer_xuid = ?
  AND kv.match_id IN (%s)
GROUP BY kv.match_id`

// buildHomeAssistedFragsQuery assemble Q26l et ses arguments. Retourne ("", nil) sur un
// lot vide.
func buildHomeAssistedFragsQuery(xuid string, matchIDs []string) (string, []any) {
	if len(matchIDs) == 0 {
		return "", nil
	}
	low, mid := domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct
	sqlText := fmt.Sprintf(Q26lHomeAssistedFragsTpl, low, low, mid, mid, Placeholders(len(matchIDs)))
	args := append([]any{xuid}, ToAnySlice(matchIDs)...)
	return sqlText, args
}

// LoadMatchAssistedFrags : part des frags du joueur assistés par un coéquipier, par match
// du lot (Q26l). Les matchs sans ligne mesurée pour le joueur sont absents de la map.
// Erreur propagée (pas de dégradation silencieuse ici : l'appelant journalise).
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
		if err := rows.Scan(&matchID, &a.FragsMeasured,
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
