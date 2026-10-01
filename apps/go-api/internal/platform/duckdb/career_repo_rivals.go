// Package duckdb — career_repo_rivals.go : les rivaux de la page Carrière (top némésis, top
// souffre-douleur). Sorti de career_repo_encounters.go au lot B du plan perf (2026-09-27).
//
// UNE LECTURE, DEUX CLASSEMENTS. Les deux listes sont deux tris du MÊME agrégat (un adversaire,
// ses frags, ses morts, ses matchs partagés) : Q27 le lit une fois, borné aux matchs du joueur
// (QMatchsDuJoueurTpl en UNE constante sous la fenêtre `_latest` du kill-feed, clauseListeMatchs,
// ADR 0036 I2), et classerRivaux trie en Go avec l'ordre exact de l'ancien `ORDER BY … LIMIT 10`
// SQL. Avant, la page lisait deux fois la fenêtre entière du kill-feed, une par classement.
package duckdb

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// rivauxParClassement : la taille de chaque classement servi (l'ancien `LIMIT 10` de Q27).
const rivauxParClassement = 10

// rivalLu : une ligne de Q27 avant projection — `match` est le match de la rencontre où
// l'annuaire cherche un adversaire que seul le kill-feed connaît (cf. Q27CareerRivalsTpl).
type rivalLu struct {
	domain.CareerRivalRawRow
	match string
}

// GetRivals retourne les top némésis (par deaths DESC) et top souffre-douleur
// (par frags DESC), 10 chacun, depuis le kill-feed canonique via SharedReader.
// Pas de seuil min — le ratio est calculé côté service.
func (r *CareerRepo) GetRivals(ctx context.Context) (nemeses, victims []domain.CareerRivalRawRow, err error) {
	ctx, cancel := context.WithTimeout(ctx, careerRivalsTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetRivals: shared reader: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("rivals")
	matchs, tous, err := r.lireRivaux(ctx, db)
	stop()
	if err != nil {
		return nil, nil, err
	}
	nem := classerRivaux(tous, func(l rivalLu) int { return l.Deaths })
	vic := classerRivaux(tous, func(l rivalLu) int { return l.Frags })
	// Noms : UN annuaire pour les deux listes (un même adversaire y figure souvent deux fois).
	lus := make([]*rivalLu, 0, len(nem)+len(vic))
	for i := range nem {
		lus = append(lus, &nem[i])
	}
	for i := range vic {
		lus = append(lus, &vic[i])
	}
	stop = timing.FromContext(ctx).Section("rivals_annuaire")
	err = nommerLignes(ctx, db, matchs, lus, accesLigne[*rivalLu]{
		xuid:   func(l *rivalLu) string { return l.XUID },
		match:  func(l *rivalLu) string { return l.match },
		nommer: func(l **rivalLu, gt string) { (*l).Gamertag = gt },
	})
	stop()
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetRivals: %w", err)
	}
	return projeterRivaux(nem), projeterRivaux(vic), nil
}

// classerRivaux rend une COPIE des `rivauxParClassement` premières lignes selon `cle`
// décroissante, puis matchs partagés décroissants, puis xuid croissant — l'ordre total de
// l'ancien `ORDER BY <cle> DESC, match_count DESC, opp_xuid ASC` (octet par octet, comme la
// collation binaire de DuckDB). `tous` n'est pas modifié : les deux classements le partagent.
func classerRivaux(tous []rivalLu, cle func(rivalLu) int) []rivalLu {
	tri := slices.Clone(tous)
	slices.SortFunc(tri, func(a, b rivalLu) int {
		if c := cmp.Compare(cle(b), cle(a)); c != 0 {
			return c
		}
		if c := cmp.Compare(b.MatchCount, a.MatchCount); c != 0 {
			return c
		}
		return cmp.Compare(a.XUID, b.XUID)
	})
	return tri[:min(len(tri), rivauxParClassement)]
}

// projeterRivaux rend les lignes du contrat (nil pour une liste vide, comme la lecture d'origine).
func projeterRivaux(lus []rivalLu) []domain.CareerRivalRawRow {
	if len(lus) == 0 {
		return nil
	}
	out := make([]domain.CareerRivalRawRow, len(lus))
	for i, l := range lus {
		out[i] = l.CareerRivalRawRow
	}
	return out
}

// lireRivaux lit les matchs du joueur puis Q27 bornée à ces matchs : TOUS les adversaires, SANS
// nom. Sans match : aucune ligne, sans requête (`IN ()` n'est pas du SQL valide).
func (r *CareerRepo) lireRivaux(ctx context.Context, db *sql.DB) ([]string, []rivalLu, error) {
	matchs, err := matchsDeLHistorique(ctx, db, r.historique())
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetRivals: %w", err)
	}
	if len(matchs) == 0 {
		return nil, nil, nil
	}
	x := r.pdb.XUID
	liste, listeArg := clauseListeMatchs("kv.match_id", matchs)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(Q27CareerRivalsTpl, liste), x, x, x, x, x, listeArg, x)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetRivals: %w", err)
	}
	defer rows.Close()

	var results []rivalLu
	for rows.Next() {
		var l rivalLu
		if err := rows.Scan(&l.XUID, &l.Frags, &l.Deaths, &l.MatchCount, &l.match); err != nil {
			return nil, nil, fmt.Errorf("CareerRepo.GetRivals scan: %w", err)
		}
		results = append(results, l)
	}
	return matchs, results, rows.Err()
}
