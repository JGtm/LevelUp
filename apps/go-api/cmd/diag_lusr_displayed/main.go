//go:build cgo

// diag_lusr_displayed — vérifie END-TO-END le lissage d'affichage LUSR v2.
//
// Lit la séquence de paliers RÉELLEMENT persistée dans match_skill_rank
// (rating_type='LUSR') de chaque player DB, dans l'ordre chronologique
// (written_at), et compte les chutes brutales restantes. Après le backfill avec
// hystérésis, on attend ~0 chute −≥2 sous-paliers en un match.
//
// Lecture seule. Usage :
//
//	go run -tags cgo ./cmd/diag_lusr_displayed [-root ../..]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"

	_ "github.com/duckdb/duckdb-go/v2"
)

var players = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

func main() {
	root := flag.String("root", "../..", "racine du repo")
	flag.Parse()
	bnd := skillv2.DefaultTierBoundaries()
	ctx := context.Background()

	fmt.Println("# Vérification end-to-end — paliers affichés (match_skill_rank LUSR)")
	for _, gt := range players {
		path := fmt.Sprintf("%s/data/titles/halo_infinite/players/%s/stats.duckdb", *root, gt)
		db, err := sql.Open("duckdb", path+"?access_mode=read_only")
		if err != nil {
			fmt.Printf("\n## %s — open err: %v\n", gt, err)
			continue
		}
		groups := loadDisplayedByGroup(ctx, db, bnd)
		db.Close()
		var total int
		var down, up moveStats // agrégats POST-placement (rang établi)
		gnames := make([]string, 0, len(groups))
		for g := range groups {
			gnames = append(gnames, g)
		}
		sort.Strings(gnames)
		fmt.Printf("\n## %s\n", gt)
		fmt.Println("- rang final par groupe (lissé, sur le dernier match) :")
		for _, g := range gnames {
			gd := groups[g]
			total += len(gd.ords)
			s := countMoves(gd.ords, bnd, skillv2.PlacementMatches+1)
			down.downAbrupt += s.downAbrupt
			up.upAbrupt += s.upAbrupt
			up.tierUpBig += s.tierUpBig
			if s.downWorst < down.downWorst {
				down.downWorst = s.downWorst
			}
			if s.upBest > up.upBest {
				up.upBest = s.upBest
			}
			final := ""
			if len(gd.labels) > 0 {
				final = gd.labels[len(gd.labels)-1]
			}
			placement := ""
			if len(gd.ords) < skillv2.PlacementMatches {
				placement = fmt.Sprintf(" [EN PLACEMENT, %d restants]", skillv2.PlacementMatches-len(gd.ords))
			}
			fmt.Printf("    %-16s %-14s (%d matchs)%s\n", g, final, len(gd.ords), placement)
		}
		fmt.Printf("- POST-placement, en 1 match : CHUTES −≥2sp=%d (pire %d sp) | PICS +≥2sp=%d (plus gros bond +%d sp, dont franchissant un palier d'un coup=%d)\n",
			down.downAbrupt, -down.downWorst, up.upAbrupt, up.upBest, up.tierUpBig)
	}
}

type groupDisplayed struct {
	ords   []int
	labels []string
}

// loadDisplayedByGroup retourne, PAR playlist_group, la séquence chronologique
// des paliers affichés écrits par v2. Isole les lignes v2 via la présence d'un
// sibling rating_type='LUSR_V2' (les anciennes lignes v1 n'en ont pas) et prend
// la version courante de chaque match (rn=1). Le rang étant par groupe, on ne
// mesure JAMAIS de chute entre deux groupes différents.
func loadDisplayedByGroup(ctx context.Context, db *sql.DB, bnd []skillv2.TierBoundary) map[string]*groupDisplayed {
	rows, err := db.QueryContext(ctx, `
		WITH v2 AS (SELECT DISTINCT match_id FROM match_skill_rank WHERE rating_type='LUSR_V2'),
		ranked AS (
			SELECT playlist_group, tier, COALESCE(sub_tier,0) AS sub, tier_label, written_at, id, match_id,
			       ROW_NUMBER() OVER (PARTITION BY match_id ORDER BY written_at DESC, id DESC) rn
			FROM match_skill_rank
			WHERE rating_type='LUSR' AND tier IS NOT NULL
		)
		SELECT r.playlist_group, r.tier, r.sub, COALESCE(r.tier_label,'')
		FROM ranked r JOIN v2 ON v2.match_id = r.match_id
		WHERE r.rn = 1
		ORDER BY r.playlist_group, r.written_at ASC, r.id ASC`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "loadDisplayedByGroup:", err)
		return nil
	}
	defer rows.Close()
	out := map[string]*groupDisplayed{}
	for rows.Next() {
		var group, tier, label string
		var sub int
		if err := rows.Scan(&group, &tier, &sub, &label); err != nil {
			continue
		}
		g, ok := out[group]
		if !ok {
			g = &groupDisplayed{}
			out[group] = g
		}
		g.ords = append(g.ords, skillv2.TierOrdinal(bnd, tier, sub))
		g.labels = append(g.labels, label)
	}
	return out
}

// moveStats agrège les mouvements de rang par match, dans les deux sens.
type moveStats struct {
	downAbrupt int // matchs avec −≥2 sous-paliers en 1 match
	downWorst  int // pire chute (négatif)
	upAbrupt   int // matchs avec +≥2 sous-paliers en 1 match
	upBest     int // plus gros bond (positif)
	tierUpBig  int // matchs franchissant ≥1 palier complet vers le HAUT en 1 match (bond ≥2 sp)
	tierDown   int // franchissements de palier vers le bas (doux, pas-à-pas)
}

// countMoves compte les mouvements à partir de l'index `from` (pour ignorer le
// placement). Mesure montées ET descentes.
func countMoves(ords []int, bnd []skillv2.TierBoundary, from int) moveStats {
	bases := tierBases(bnd)
	if from < 1 {
		from = 1
	}
	var s moveStats
	for i := from; i < len(ords); i++ {
		d := ords[i] - ords[i-1]
		ti, tp := tierOf(ords[i], bases), tierOf(ords[i-1], bases)
		if d <= -2 {
			s.downAbrupt++
		}
		if d >= 2 {
			s.upAbrupt++
		}
		if d < s.downWorst {
			s.downWorst = d
		}
		if d > s.upBest {
			s.upBest = d
		}
		if ti < tp {
			s.tierDown++
		}
		if ti > tp && d >= 2 {
			s.tierUpBig++ // saut d'au moins un palier en un seul match
		}
	}
	return s
}

func tierBases(bnd []skillv2.TierBoundary) []int {
	bases := make([]int, len(bnd))
	acc := 0
	for i, b := range bnd {
		bases[i] = acc
		n := b.SubTiers
		if n < 1 {
			n = 1
		}
		acc += n
	}
	return bases
}

func tierOf(ord int, bases []int) int {
	idx := 0
	for i, b := range bases {
		if ord >= b {
			idx = i
		}
	}
	return idx
}
