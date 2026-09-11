package main

// crane.go — axe 1 TER : le temps de portage du CRANE d'Oddball, publie contre l'oracle API
// (`time_as_skull_carrier_seconds`).
//
// AJOUT DU LOT 6.7-B1 (2026-09-10). L'audit du 2026-09-10 ne pouvait pas le mesurer : aucun
// film Oddball n'avait d'artefact au parc (§2.2). Les quatre films Oddball du cache sont
// desormais cuits HORS LIGNE dans un parc de scratchpad, et cette sortie confronte leur calque
// a l'oracle exactement comme `axe1Portage` le fait du drapeau.

import (
	"fmt"

	"strings"
)

// axe1Crane rend, par film Oddball et par joueur, le portage publie et celui de l'oracle.
func (c *ctxAudit) axe1Crane() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "xuid", "gamertag",
		"periodes", "publie_s", "oracle_s", "ecart_s", "ratio", "depasse_oracle"}, "\t")}
	for _, f := range c.films {
		if len(f.art.SkullCarries) == 0 && famille(f.mode) != "Oddball" {
			continue
		}
		pf := float64(f.art.FrameIntervalMS) / 1000.0
		pub := map[string]float64{}
		nb := map[string]int{}
		for _, s := range f.art.SkullCarries {
			if s.XUID == "" {
				continue
			}
			pub[s.XUID] += float64(s.T1-s.T0+1) * pf
			nb[s.XUID]++
		}
		xs := map[string]bool{}
		for x := range pub {
			xs[x] = true
		}
		for x, r := range c.oracle[f.match] {
			if v, ok := c.oTSV.f(r, "time_as_skull_carrier_seconds"); ok && v > 0 {
				xs[x] = true
			}
		}
		for _, x := range triees(xs) {
			var orc float64
			if r, ok := c.oracle[f.match][x]; ok {
				orc, _ = c.oTSV.f(r, "time_as_skull_carrier_seconds")
			}
			ratio := ""
			if orc > 0 {
				ratio = fmt.Sprintf("%.3f", pub[x]/orc)
			}
			l = append(l, strings.Join([]string{f.court, famille(f.mode), f.mode, x,
				c.gt[f.match][x], fmt.Sprint(nb[x]), fmt.Sprintf("%.1f", pub[x]),
				fmt.Sprintf("%.1f", orc), fmt.Sprintf("%.1f", pub[x]-orc), ratio,
				fmt.Sprint(pub[x] > orc+0.05)}, "\t"))
		}
	}
	return l
}

// bilanCrane et bilanDrapeau resument, par film puis en total, ce que les deux calques de
// portage publient contre l'oracle — la ligne que le rapport de lot cite.
func (c *ctxAudit) bilanPortages() []string {
	l := []string{strings.Join([]string{"calque", "film", "famille", "periodes", "publie_s",
		"oracle_s", "ratio", "joueurs_au_dessus", "joueurs_mesures", "pire_ecart_xuid",
		"pire_ecart_s"}, "\t")}
	l = append(l, c.bilanUnCalque("crane", "time_as_skull_carrier_seconds", periodesCrane)...)
	l = append(l, c.bilanUnCalque("drapeau", "time_as_flag_carrier_seconds", periodesDrapeauCommeCrane)...)
	return l
}

// periodesCrane / periodesDrapeauCommeCrane ramenent les deux calques a la MEME forme (xuid,
// duree en frames) pour que le bilan n'ecrive la regle qu'une fois.
func periodesCrane(a *artefact) []periode {
	out := make([]periode, 0, len(a.SkullCarries))
	for _, s := range a.SkullCarries {
		if s.XUID == "" {
			continue
		}
		out = append(out, periode{xuid: s.XUID, t0: s.T0, t1: s.T1})
	}
	return out
}

func periodesDrapeauCommeCrane(a *artefact) []periode { return periodesDrapeau(a) }

func (c *ctxAudit) bilanUnCalque(nom, colonne string, lecture func(*artefact) []periode) []string {
	var out []string
	var tPub, tOrc float64
	tPer, tAuDessus, tMesures := 0, 0, 0
	for _, f := range c.films {
		periodes := lecture(f.art)
		if len(periodes) == 0 {
			continue
		}
		pf := float64(f.art.FrameIntervalMS) / 1000.0
		pub := map[string]float64{}
		for _, p := range periodes {
			pub[p.xuid] += float64(p.t1-p.t0+1) * pf
		}
		xs := map[string]bool{}
		for x := range pub {
			xs[x] = true
		}
		for x, r := range c.oracle[f.match] {
			if v, ok := c.oTSV.f(r, colonne); ok && v > 0 {
				xs[x] = true
			}
		}
		var sp, so, pireEcart float64
		auDessus, pire := 0, ""
		for _, x := range triees(xs) {
			var orc float64
			if r, ok := c.oracle[f.match][x]; ok {
				orc, _ = c.oTSV.f(r, colonne)
			}
			sp += pub[x]
			so += orc
			if pub[x] > orc+0.05 {
				auDessus++
			}
			if e := pub[x] - orc; e > pireEcart {
				pireEcart, pire = e, x
			}
		}
		ratio := ""
		if so > 0 {
			ratio = fmt.Sprintf("%.3f", sp/so)
		}
		out = append(out, strings.Join([]string{nom, f.court, famille(f.mode),
			fmt.Sprint(len(periodes)), fmt.Sprintf("%.1f", sp), fmt.Sprintf("%.1f", so), ratio,
			fmt.Sprint(auDessus), fmt.Sprint(len(xs)), pire,
			fmt.Sprintf("%.1f", pireEcart)}, "\t"))
		tPub += sp
		tOrc += so
		tPer += len(periodes)
		tAuDessus += auDessus
		tMesures += len(xs)
	}
	ratio := ""
	if tOrc > 0 {
		ratio = fmt.Sprintf("%.3f", tPub/tOrc)
	}
	out = append(out, strings.Join([]string{nom, "TOTAL", "", fmt.Sprint(tPer),
		fmt.Sprintf("%.1f", tPub), fmt.Sprintf("%.1f", tOrc), ratio,
		fmt.Sprint(tAuDessus), fmt.Sprint(tMesures), "", ""}, "\t"))
	return out
}
