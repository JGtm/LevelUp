package persist

// kill_summaries_catchup.go — QUELS MATCHS ONT BESOIN DE LEURS RÉSUMÉS (ADR 0039, décisions 5-6).
//
// Un match est À RATTRAPER quand sa passe COURANTE du journal (`match_kill_events_latest`) n'a
// pas de tête qui la résume à la révision courante :
//   - aucune tête (passe écrite avant les résumés, par une migration historique ou par un
//     écrivain hors ligne) ;
//   - une tête à une autre révision que [killsummary.Rev] (règle de résumé modifiée) ;
//   - une tête qui résume une AUTRE passe que la courante (une passe plus récente a été écrite
//     sans ses résumés : la tête servie compte alors les morts d'une passe supplantée).
//
// Cette définition est UNIQUE : la commande de rattrapage (`levelup backfill-match-summaries`)
// et la jauge du contrôle de santé des données lisent le même texte SQL, donc la jauge tombe à
// zéro exactement quand la commande n'a plus rien à écrire.
//
// COÛT : la vue `_latest` du journal porte une fenêtre sur toute la table. Ce n'est pas une
// lecture de page : elle se lance depuis le contrôle de santé périodique et depuis la
// commande, jamais au démarrage ni sur un chemin de requête.

import (
	"context"
	"fmt"

	"levelup/go-api/internal/analysis/killsummary"
)

// pendingKillSummariesSQL : un identifiant par match à rattraper, ordre stable. Le `?` est la
// révision courante, liée plutôt que collée au texte.
const pendingKillSummariesSQL = `
	SELECT cur.match_id
	FROM (SELECT match_id, any_value(decode_pass) AS decode_pass
	      FROM match_kill_events_latest GROUP BY match_id) cur
	LEFT JOIN match_kill_heads_latest h ON h.match_id = cur.match_id
	WHERE h.match_id IS NULL OR h.summary_rev <> ? OR h.src_decode_pass <> cur.decode_pass
	ORDER BY cur.match_id`

// PendingKillSummaryMatches rend, triés, les matchs dont la passe courante n'a pas de tête à
// jour (voir l'en-tête du fichier). Lecture seule, par les vues `_latest`.
func PendingKillSummaryMatches(ctx context.Context, q rowQuerier) ([]string, error) {
	rows, err := q.QueryContext(ctx, pendingKillSummariesSQL, killsummary.Rev)
	if err != nil {
		return nil, fmt.Errorf("persist: matchs à rattraper (résumés): %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("persist: matchs à rattraper (scan): %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("persist: matchs à rattraper (rows): %w", err)
	}
	return out, nil
}

// CountPendingKillSummaries : le nombre de matchs à rattraper, sans rapatrier leurs
// identifiants. Même texte SQL que [PendingKillSummaryMatches] (compté autour de lui).
func CountPendingKillSummaries(ctx context.Context, q rowQuerier) (int, error) {
	rows, err := q.QueryContext(ctx, `SELECT COUNT(*) FROM (`+pendingKillSummariesSQL+`)`, killsummary.Rev)
	if err != nil {
		return 0, fmt.Errorf("persist: compte des matchs à rattraper (résumés): %w", err)
	}
	defer func() { _ = rows.Close() }()
	n := 0
	if rows.Next() {
		if err := rows.Scan(&n); err != nil {
			return 0, fmt.Errorf("persist: compte des matchs à rattraper (scan): %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("persist: compte des matchs à rattraper (rows): %w", err)
	}
	return n, nil
}
