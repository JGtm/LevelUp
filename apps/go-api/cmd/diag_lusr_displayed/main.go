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
		var total, abruptAll, abruptPost, worstPost int
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
			aAll, _, _ := countDrops(gd.ords, bnd, 1)
			aPost, _, wPost := countDrops(gd.ords, bnd, skillv2.PlacementMatches+1)
			abruptAll += aAll
			abruptPost += aPost
			if wPost < worstPost {
				worstPost = wPost
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
		fmt.Printf("- volatilité : chutes −≥2sp TOUT=%d, POST-placement=**%d** (pire %d sp)\n", abruptAll, abruptPost, -worstPost)
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

// countDrops compte les chutes à partir de l'index `from` (1-based sur les
// transitions) pour pouvoir ignorer la phase de placement.
func countDrops(ords []int, bnd []skillv2.TierBoundary, from int) (abrupt, tierDrops, worst int) {
	bases := tierBases(bnd)
	if from < 1 {
		from = 1
	}
	for i := from; i < len(ords); i++ {
		d := ords[i] - ords[i-1]
		if d <= -2 {
			abrupt++
		}
		if d < worst {
			worst = d
		}
		if tierOf(ords[i], bases) < tierOf(ords[i-1], bases) {
			tierDrops++
		}
	}
	return
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
