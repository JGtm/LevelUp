// Package duckdb — relation_assists_repo.go : les assistances ÉCHANGÉES entre le joueur
// et chaque coéquipier, sur les matchs dont l'assistance est mesurée (Q28c).
//
// Un seul lecteur SQL, deux surfaces : la page Relations (tous les coéquipiers, scope de
// filtres optionnel) et l'historique des rencontres de la vue match (les seuls joueurs du
// match affiché, tout l'historique). Doctrine et tranches : domain/relation_assists.go.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// Q28cRelationAssistsTpl : pour chaque coéquipier, les assistances reçues et données,
// par tranche de part, et les dénominateurs (matchs mesurés, frags de chacun).
//
// PORTÉE DES LIGNES : `publishable AND assist_known`, pour les compteurs comme pour les
// frags. Une paire nomme deux joueurs (lecture ligne à ligne, cf. Q32d), et les frags
// servent de DÉNOMINATEUR aux assistances : ils doivent être lus sur la même population
// de lignes, sinon le pourcentage mélange deux portées.
//
// MATCHS COMPTÉS : ceux où le joueur et le coéquipier sont dans la MÊME équipe
// (`team_id` égal ; un team_id NULL ne s'égale à rien) ET qui portent au moins une ligne
// mesurée. Un match mesuré sans aucune assistance entre eux compte quand même : c'est
// « mesuré, zéro », pas « on ne sait pas ».
//
// Titres sans décodeur de film : `measured` est vide, donc aucune ligne ne sort — pas de
// branchement sur le slug. Le masquage Campagne n'est pas nécessaire pour la même raison
// (aucune ligne de film en Campagne).
//
// BORNAGE (ADR 0036 I2) : l'unique lecture de la vue `_latest` du kill-feed (CTE `lues`,
// matérialisée, d'où dérivent `measured` et les frags) porte la liste des matchs de la
// lecture, liée en UN paramètre constant sous la fenêtre. La liste est le périmètre (scope) quand il est fourni, sinon tous les matchs du
// joueur (QMatchsOuJoue) : `mates` ne garde que des matchs du joueur, donc la borne ne retire
// aucune ligne servie.
//
// JOINTURE PAR ÉGALITÉ SEULE : `kills` se joint à `mates` sur `match_id` ; chaque compteur
// nomme son tueur dans son FILTER (le joueur ou le coéquipier), donc les frags des autres
// tueurs du match rejoignent la ligne sans entrer dans aucun compte. Une disjonction dans la
// condition de jointure (`k.killer = me OR k.killer = partner`) fait choisir à DuckDB une
// jointure par boucles imbriquées, l'essentiel du coût de la lecture.
//
// Format string, DANS CET ORDRE : borne de `lues`, prédicat d'exclusion des bots
// (analysis.SQLIsNotBotCol), scopeClause (" AND mp.match_id IN (?,…)" ou ""),
// partnerClause (" AND p.xuid IN (…)" ou ""), puis les bornes de tranche
// (low max, mid min, mid max, high min) × 2 (reçues, données).
// Placeholders ? : ?1 = xuid du joueur, ?2 = liste de `lues`, puis ceux de scopeClause,
// puis ceux de partnerClause.
const Q28cRelationAssistsTpl = `
WITH me AS (
    SELECT CAST(? AS VARCHAR) AS xuid
),
lues AS MATERIALIZED (
    SELECT kv.match_id,
           kv.feed_killer_xuid  AS killer,
           kv.assist_xuid       AS assister,
           kv.assist_damage_pct AS pct
    FROM ` + KillEventsCanonicalTable + ` kv
    WHERE kv.publishable AND kv.assist_known AND %s
),
measured AS (
    SELECT DISTINCT match_id FROM lues
),
mates AS (
    SELECT DISTINCT mp.match_id, p.xuid AS partner
    FROM me
    JOIN match_participants mp ON mp.xuid = me.xuid
    JOIN match_participants p
        ON p.match_id = mp.match_id
       AND p.team_id  = mp.team_id
       AND p.xuid    <> mp.xuid
    WHERE %s
      AND mp.match_id IN (SELECT match_id FROM measured)%s%s
),
kills AS (
    SELECT * FROM lues
    WHERE match_id IN (SELECT match_id FROM mates)
)
SELECT
    m.partner,
    COUNT(DISTINCT m.match_id)                                                    AS matches_measured,
    COUNT(k.killer) FILTER (WHERE k.killer = me.xuid)                             AS my_frags,
    COUNT(k.killer) FILTER (WHERE k.killer = m.partner)                           AS partner_frags,
    COUNT(k.killer) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner)  AS received_total,
    COUNT(k.killer) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner AND k.pct < %d)                AS received_low,
    COUNT(k.killer) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner AND k.pct >= %d AND k.pct <= %d) AS received_mid,
    COUNT(k.killer) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner AND k.pct > %d)                AS received_high,
    COUNT(k.killer) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid)  AS given_total,
    COUNT(k.killer) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid AND k.pct < %d)                AS given_low,
    COUNT(k.killer) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid AND k.pct >= %d AND k.pct <= %d) AS given_mid,
    COUNT(k.killer) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid AND k.pct > %d)                AS given_high
FROM mates m
CROSS JOIN me
LEFT JOIN kills k
    ON k.match_id = m.match_id
GROUP BY m.partner`

// assistExchangeQuery : paramètres de lecture de Q28c.
type assistExchangeQuery struct {
	me string
	// scope : nil = tous les matchs ; non-nil = restreint à ces match_id (vide = rien).
	scope []string
	// borne : la liste liée sous les fenêtres `_latest` (cf. Q28cRelationAssistsTpl). Posée
	// par queryRelationAssists : le scope s'il est fourni, sinon les matchs du joueur.
	borne []string
	// partnersOfMatch : non vide = ne garder que les joueurs de ce match.
	partnersOfMatch string
}

// buildRelationAssistsQuery assemble le SQL et les arguments de Q28c. Retourne ("", nil)
// si le scope est non-nil et vide (aucun match en périmètre).
func buildRelationAssistsQuery(q assistExchangeQuery) (string, []any) {
	if q.scope != nil && len(q.scope) == 0 {
		return "", nil
	}
	borneClause, borneArg := clauseListeMatchs("kv.match_id", q.borne)
	args := []any{q.me, borneArg}
	scopeClause, partnerClause := "", ""
	if q.scope != nil {
		scopeClause = " AND mp.match_id IN (" + Placeholders(len(q.scope)) + ")"
		args = append(args, ToAnySlice(q.scope)...)
	}
	if q.partnersOfMatch != "" {
		partnerClause = " AND p.xuid IN (SELECT xuid FROM match_participants WHERE match_id = ?)"
		args = append(args, q.partnersOfMatch)
	}
	low, mid := domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct
	sqlText := fmt.Sprintf(Q28cRelationAssistsTpl, borneClause, analysis.SQLIsNotBotCol("p.xuid"),
		scopeClause, partnerClause,
		low, low, mid, mid,
		low, low, mid, mid)
	return sqlText, args
}

// queryRelationAssists exécute Q28c et indexe le résultat par xuid du coéquipier.
func queryRelationAssists(ctx context.Context, db *sql.DB, q assistExchangeQuery) (map[string]domain.RelationAssists, error) {
	out := map[string]domain.RelationAssists{}
	if q.scope != nil && len(q.scope) == 0 {
		return out, nil
	}
	q.borne = q.scope
	if q.borne == nil {
		matchs, err := matchsOuJoue(ctx, db, q.me)
		if err != nil {
			return nil, fmt.Errorf("relation assists (Q28c): %w", err)
		}
		q.borne = matchs
	}
	sqlText, args := buildRelationAssistsQuery(q)
	if sqlText == "" {
		return out, nil
	}
	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("relation assists (Q28c): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			partner string
			a       domain.RelationAssists
		)
		if err := rows.Scan(&partner, &a.MatchesMeasured, &a.MyFrags, &a.PartnerFrags,
			&a.Received.Total, &a.Received.Low, &a.Received.Mid, &a.Received.High,
			&a.Given.Total, &a.Given.Low, &a.Given.Mid, &a.Given.High,
		); err != nil {
			return nil, fmt.Errorf("relation assists (Q28c) scan: %w", err)
		}
		out[partner] = a
	}
	return out, rows.Err()
}

// GetRelationAssists : assistances échangées avec chaque coéquipier (page Relations).
// scope : même contrat que GetRelations (nil = tous ; vide = aucun match).
func (r *CareerRepo) GetRelationAssists(ctx context.Context, scope []string) (map[string]domain.RelationAssists, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelationAssists: shared reader: %w", err)
	}
	defer release()
	out, err := queryRelationAssists(ctx, db, assistExchangeQuery{me: r.pdb.XUID, scope: scope})
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelationAssists: %w", err)
	}
	return out, nil
}
