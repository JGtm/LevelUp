package duckdb

// tactical_repo_contextes.go — LE VOISINAGE DES MORTS D'UNE LISTE DE MATCHS, pour le placement
// d'une mort (seul / près d'un coéquipier) au détail d'une zone du plan tactique.
//
// Lecture de `match_death_context_latest` SEULE : le contexte de chaque mort, mesuré au sync par le
// collecteur de kills (distance au coéquipier visible le plus proche, coéquipiers vus et non vus).
// L'appariement à une mort précise (même victime, instant à ± 1,5 s) est fait par l'appelant
// (analysis/tactical.ContexteLePlusProche) : la base rend les lignes, elle ne tranche rien.
//
// LA LISTE BLANCHE EST EXIGÉE, et elle est liée en constantes sur le `match_id` de la vue : c'est
// la seule forme que DuckDB pousse sous la fenêtre de la dernière passe (ADR 0036 I2, garde-rail
// `TestTacticalRepo_ContextesDeMort_BorneEtNull`). Sans liste, la lecture parcourrait la table sur
// tout l'historique : elle est refusée.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// QTacticalContextes : %s = la liste des matchs, en constantes liées.
const QTacticalContextes = `
SELECT c.match_id, COALESCE(c.victim_xuid, '') AS victim_xuid, c.time_ms,
       c.nearest_teammate_m, c.teammates_visible, c.teammates_out_of_sight
FROM match_death_context_latest c
WHERE c.match_id IN (%s)
ORDER BY c.match_id, c.time_ms`

// ContextesDeMort rend le contexte de chaque mort des matchs de `q.Matchs` (liste blanche
// obligatoire ; vide = aucune ligne, aucune requête). Table absente → ErrCapabilityNotSupported.
func (r *TacticalRepo) ContextesDeMort(ctx context.Context, q domain.TacticalQuery) ([]domain.ContexteDeMort, error) {
	if !q.Matchs.Restreint() {
		return nil, fmt.Errorf("TacticalRepo.ContextesDeMort: liste blanche de matchs exigee")
	}
	ids := q.Matchs.IDs()
	if len(ids) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()
	db, release, err := r.ouvrir(ctx, q, "ContextesDeMort")
	if err != nil {
		return nil, err
	}
	defer release()

	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(QTacticalContextes, Placeholders(len(ids))), args...)
	if err != nil {
		return nil, r.degrader(ctx, "ContextesDeMort", err)
	}
	var out []domain.ContexteDeMort
	err = scanRows(ctx, rows, "TacticalRepo.ContextesDeMort", func(sc rowScanner) error {
		var c domain.ContexteDeMort
		var proche sql.NullFloat64
		if err := sc.Scan(&c.MatchID, &c.VictimXUID, &c.TimeMs, &proche, &c.Visibles, &c.HorsDeVue); err != nil {
			return err
		}
		c.PlusProcheM = nullFloatPtr(proche)
		out = append(out, c)
		return nil
	})
	return out, err
}
