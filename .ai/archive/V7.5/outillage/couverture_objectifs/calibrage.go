package main

// calibrage.go — le CONTROLE de l'instrument avant toute affirmation.
//
// Deux moities a calibrer, et chacune a son temoin :
//
//  1. LECTURE DE L'ARTEFACT — le compte de spans `carried` recalcule depuis `flagCarries`
//     doit egaler le compteur publie `coverage.flagCarries.carries`. Un ecart signifie que
//     l'instrument ne lit pas la meme chose que le producteur.
//  2. LECTURE DE L'ORACLE — les valeurs Oddball citees au §3.3 du rapport
//     `RAPPORT_ODDBALL_FANTOMES_2026-09-10.md` (1 249,0 s au total sur 4 films, et les trois
//     valeurs par joueur 51,1 / 62,3 / 40,8 s) doivent ressortir du TSV.
//
// Le calibrage de reference demande par l'instruction — les 82,2 % de l'oracle sur les films
// Oddball CUITS — n'est PAS reproductible : aucun film Oddball n'a d'artefact au parc (le
// recensement le prouve, colonne `skullCarries` a zero sur les 64). Seule la moitie ORACLE de ce
// chiffre est verifiable ici.

import (
	"fmt"
	"sort"
	"strings"
)

func (c *ctxAudit) calibrage() []string {
	l := []string{"# calibrage 1 : spans `carried` recalcules vs coverage.flagCarries.carries",
		strings.Join([]string{"film", "spans_carried_recalcules", "coverage_carries",
			"coverage_openings", "accord"}, "\t")}
	dAccord, total := 0, 0
	for _, f := range c.films {
		if len(f.art.FlagCarries) == 0 || f.cov.FlagCarries == nil {
			continue
		}
		n := 0
		for _, fc := range f.art.FlagCarries {
			for _, s := range fc.Spans {
				if s.State == "carried" || s.State == "carried_open" {
					n++
				}
			}
		}
		ok := n == f.cov.FlagCarries.Carries
		if ok {
			dAccord++
		}
		total++
		l = append(l, strings.Join([]string{f.court, fmt.Sprint(n),
			fmt.Sprint(f.cov.FlagCarries.Carries), fmt.Sprint(f.cov.FlagCarries.Openings),
			fmt.Sprint(ok)}, "\t"))
	}
	l = append(l, fmt.Sprintf("# accord %d/%d", dAccord, total), "")

	l = append(l, "# calibrage 2 : oracle Oddball (rapport 6.2 §3.3 — 1249,0 s sur 4 films)",
		strings.Join([]string{"match", "joueurs", "somme_time_as_skull_carrier_s"}, "\t"))
	oddball := []string{"d9781168", "51ebbc0f", "c88ec007", "43716616"}
	var somme float64
	for _, court := range oddball {
		var s float64
		n := 0
		for m, par := range c.oracle {
			if !strings.HasPrefix(m, court) {
				continue
			}
			for _, r := range par {
				if v, ok := c.oTSV.f(r, "time_as_skull_carrier_seconds"); ok {
					s += v
					if v > 0 {
						n++
					}
				}
			}
		}
		somme += s
		l = append(l, fmt.Sprintf("%s\t%d\t%.1f", court, n, s))
	}
	l = append(l, fmt.Sprintf("TOTAL\t\t%.1f", somme), "")

	l = append(l, "# calibrage 3 : les trois porteurs cites au rapport 6.2 §3.3",
		strings.Join([]string{"match", "xuid", "oracle_s", "attendu_rapport"}, "\t"))
	temoins := [][3]string{
		{"d9781168", "2533274977810162", "51,1"},
		{"43716616", "2533274978052136", "62,3"},
		{"c88ec007", "2535448682342105", "40,8"},
	}
	for _, t := range temoins {
		for m, par := range c.oracle {
			if !strings.HasPrefix(m, t[0]) {
				continue
			}
			if r, ok := par[t[1]]; ok {
				v, _ := c.oTSV.f(r, "time_as_skull_carrier_seconds")
				l = append(l, fmt.Sprintf("%s\t%s\t%.1f\t%s", t[0], t[1], v, t[2]))
			}
		}
	}
	return l
}

// statsPubliees rend, par film, le compte de chaque `stat` du calque `objectives` — la table
// qui expose l'anomalie du compteur `assists` sur certains films.
func (c *ctxAudit) statsPubliees() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "duree_s", "stat", "publie",
		"par_seconde"}, "\t")}
	for _, f := range c.films {
		n := map[string]int{}
		for _, o := range f.art.Objectives {
			n[o.Stat]++
		}
		cles := make([]string, 0, len(n))
		for k := range n {
			cles = append(cles, k)
		}
		sort.Strings(cles)
		for _, k := range cles {
			ps := ""
			if f.duree > 0 {
				ps = fmt.Sprintf("%.2f", float64(n[k])/float64(f.duree))
			}
			l = append(l, strings.Join([]string{f.court, famille(f.mode), f.mode,
				fmt.Sprint(f.duree), k, fmt.Sprint(n[k]), ps}, "\t"))
		}
	}
	return l
}
