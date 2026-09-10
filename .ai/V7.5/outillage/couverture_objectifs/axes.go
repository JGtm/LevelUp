package main

// axes.go — les cinq axes de mesure de l audit, un par sortie TSV. Chacun ne lit que des
// artefacts deja cuits et l oracle TSV : aucune base, aucune recuisson.

import (
	"fmt"
	"sort"
	"strings"
)

// --- axe 1 : temps de portage publie vs oracle -------------------------------

// periode est un portage : une suite maximale de spans `carried`/`carried_open` du meme xuid.
type periode struct {
	xuid   string
	t0, t1 int // frames, bornes incluses
	ouvert bool
}

// periodesDrapeau rend les portages du drapeau, par regroupement des spans contigus.
func periodesDrapeau(a *artefact) []periode {
	var out []periode
	for _, fc := range a.FlagCarries {
		var cur *periode
		for _, s := range fc.Spans {
			porte := s.State == "carried" || s.State == "carried_open"
			if !porte || s.XUID == "" {
				cur = nil
				continue
			}
			if cur != nil && cur.xuid == s.XUID && s.T0 == cur.t1+1 {
				cur.t1 = s.T1
				cur.ouvert = cur.ouvert || s.State == "carried_open"
				continue
			}
			out = append(out, periode{xuid: s.XUID, t0: s.T0, t1: s.T1,
				ouvert: s.State == "carried_open"})
			cur = &out[len(out)-1]
		}
	}
	return out
}

// spansPortesSansXUID compte les images portees dont le porteur n'est PAS nomme.
func spansPortesSansXUID(a *artefact) (spans, frames int) {
	for _, fc := range a.FlagCarries {
		for _, s := range fc.Spans {
			if (s.State == "carried" || s.State == "carried_open") && s.XUID == "" {
				spans++
				frames += s.T1 - s.T0 + 1
			}
		}
	}
	return
}

func (c *ctxAudit) axe1Portage() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "xuid", "gamertag",
		"periodes", "publie_s", "oracle_s", "ecart_s", "ratio", "depasse_oracle"}, "\t")}
	for _, f := range c.films {
		if famille(f.mode) != "CTF" {
			continue
		}
		pf := float64(f.art.FrameIntervalMS) / 1000.0
		pub := map[string]float64{}
		nb := map[string]int{}
		for _, p := range periodesDrapeau(f.art) {
			pub[p.xuid] += float64(p.t1-p.t0+1) * pf
			nb[p.xuid]++
		}
		// tous les joueurs de l'oracle, meme ceux a 0 publie
		xs := map[string]bool{}
		for x := range pub {
			xs[x] = true
		}
		for x, r := range c.oracle[f.match] {
			if v, ok := c.oTSV.f(r, "time_as_flag_carrier_seconds"); ok && v > 0 {
				xs[x] = true
			}
		}
		for _, x := range triees(xs) {
			var orc float64
			if r, ok := c.oracle[f.match][x]; ok {
				orc, _ = c.oTSV.f(r, "time_as_flag_carrier_seconds")
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

// --- axe 1 bis : zones -------------------------------------------------------

func (c *ctxAudit) axe1Zones() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "zones_publiees",
		"images_possedees_eq0", "images_possedees_eq1", "s_possession_eq0", "s_possession_eq1",
		"oracle_time_in_zones_total_s", "oracle_zone_scoring_ticks_total",
		"joueurs_oracle_temps_non_nul", "publie_par_joueur"}, "\t")}
	for _, f := range c.films {
		fam := famille(f.mode)
		if fam != "Strongholds" && fam != "TotalControl" && fam != "KOTH" {
			continue
		}
		pf := float64(f.art.FrameIntervalMS) / 1000.0
		var fr0, fr1 int
		for _, z := range f.art.ZoneStates {
			for _, s := range z.Spans {
				n := s.T1 - s.T0 + 1
				switch s.Owner {
				case 0:
					fr0 += n
				case 1:
					fr1 += n
				}
			}
		}
		var oTemps, oTics float64
		nOracle := 0
		for _, r := range c.oracle[f.match] {
			if v, ok := c.oTSV.f(r, "time_in_zones_seconds"); ok {
				oTemps += v
				if v > 0 {
					nOracle++
				}
			}
			if v, ok := c.oTSV.f(r, "zone_scoring_ticks"); ok {
				oTics += v
			}
		}
		l = append(l, strings.Join([]string{f.court, fam, f.mode,
			fmt.Sprint(len(f.art.ZoneStates)), fmt.Sprint(fr0), fmt.Sprint(fr1),
			fmt.Sprintf("%.1f", float64(fr0)*pf), fmt.Sprintf("%.1f", float64(fr1)*pf),
			fmt.Sprintf("%.1f", oTemps), fmt.Sprintf("%.0f", oTics),
			fmt.Sprint(nOracle), "0 (aucun calque par joueur)"}, "\t"))
	}
	return l
}

// --- axe 2 : actions publiees vs oracle --------------------------------------

// actions confrontees : la cle est le `stat` publie dans `objectives`, la valeur la colonne
// oracle correspondante.
var actionsSuivies = [][2]string{
	{"flag_captures", "flag_captures"},
	{"flag_grabs", "flag_grabs"},
	{"flag_steals", "flag_steals"},
	{"flag_returns", "flag_returns"},
	{"flag_secures", "flag_secures"},
	{"flag_carriers_killed", "flag_carriers_killed"},
	{"flag_capture_assists", "flag_capture_assists"},
	{"zone_captures", "zone_captures"},
	{"zone_secures", "zone_secures"},
	{"skull_grabs", "skull_grabs"},
}

func (c *ctxAudit) axe2Actions() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "action", "xuid", "gamertag",
		"publie", "oracle", "ecart"}, "\t")}
	for _, f := range c.films {
		fam := famille(f.mode)
		if fam == "SansObjectif" || fam == "INCONNU" {
			continue
		}
		pub := map[string]map[string]int{}
		for _, o := range f.art.Objectives {
			if pub[o.Stat] == nil {
				pub[o.Stat] = map[string]int{}
			}
			pub[o.Stat][o.XUID]++
		}
		for _, a := range actionsSuivies {
			xs := map[string]bool{}
			for x := range pub[a[0]] {
				xs[x] = true
			}
			for x, r := range c.oracle[f.match] {
				if v, ok := c.oTSV.i(r, a[1]); ok && v > 0 {
					xs[x] = true
				}
			}
			if len(xs) == 0 {
				continue
			}
			for _, x := range triees(xs) {
				o := 0
				if r, ok := c.oracle[f.match][x]; ok {
					o, _ = c.oTSV.i(r, a[1])
				}
				p := pub[a[0]][x]
				l = append(l, strings.Join([]string{f.court, fam, f.mode, a[0], x,
					c.gt[f.match][x], fmt.Sprint(p), fmt.Sprint(o), fmt.Sprint(p - o)}, "\t"))
			}
		}
	}
	return l
}

// --- axe 3 : identite par manche ---------------------------------------------

func (c *ctxAudit) axe3Identite() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "manches_couverture",
		"manches_score_indice", "obj_available", "obj_attached", "obj_noSlot",
		"pct_actions_non_nommees", "flag_openings", "flag_carries", "flag_noBridge",
		"spans_portes_sans_xuid", "images_portees_sans_xuid", "bridge_livesNamed",
		"bridge_livesTotal", "actions_objectif_oracle_total", "actions_objectif_publiees"}, "\t")}
	for _, f := range c.films {
		fam := famille(f.mode)
		if fam == "SansObjectif" || fam == "INCONNU" {
			continue
		}
		pct := ""
		if f.cov.Objectives.Available > 0 {
			pct = fmt.Sprintf("%.1f", 100*float64(f.cov.Objectives.NoSlot)/
				float64(f.cov.Objectives.Available))
		}
		fo, fca, fnb := "", "", ""
		if f.cov.FlagCarries != nil {
			fo = fmt.Sprint(f.cov.FlagCarries.Openings)
			fca = fmt.Sprint(f.cov.FlagCarries.Carries)
			fnb = fmt.Sprint(f.cov.FlagCarries.NoBridge)
		}
		ns, nf := spansPortesSansXUID(f.art)
		// actions d'objectif : total oracle vs total publie, toutes familles confondues
		var oTot, pTot int
		for _, r := range c.oracle[f.match] {
			for _, a := range actionsSuivies {
				if v, ok := c.oTSV.i(r, a[1]); ok {
					oTot += v
				}
			}
		}
		suivi := map[string]bool{}
		for _, a := range actionsSuivies {
			suivi[a[0]] = true
		}
		for _, o := range f.art.Objectives {
			if suivi[o.Stat] {
				pTot++
			}
		}
		l = append(l, strings.Join([]string{f.court, fam, f.mode,
			fmt.Sprint(f.cov.Score.Rounds), fmt.Sprint(manchesIndice(f.art)),
			fmt.Sprint(f.cov.Objectives.Available), fmt.Sprint(f.cov.Objectives.Attached),
			fmt.Sprint(f.cov.Objectives.NoSlot), pct, fo, fca, fnb,
			fmt.Sprint(ns), fmt.Sprint(nf),
			fmt.Sprint(f.cov.Bridge.LivesNamed), fmt.Sprint(f.cov.Bridge.LivesTotal),
			fmt.Sprint(oTot), fmt.Sprint(pTot)}, "\t"))
	}
	return l
}

// manchesIndice rend le nombre de manches lu dans le FIL DE SCORE de l artefact — une source
// independante de `coverage.score.rounds`, publiee a cote d elle pour les confronter.
// et on publie -1 quand elle est absente (le lecteur sait alors que c'est non prouve).
func manchesIndice(a *artefact) int { return manchesDuFilScore(a) }

// --- axe 4 : bornage des periodes --------------------------------------------

func (c *ctxAudit) axe4Bornage() []string {
	l := []string{strings.Join([]string{"film", "famille", "xuid", "gamertag", "periodes",
		"publie_s", "oracle_s", "ecart_s", "ecart_par_periode_s", "publie_demi_fenetre_s",
		"demi_fenetre_depasse_oracle"}, "\t")}
	for _, f := range c.films {
		if famille(f.mode) != "CTF" {
			continue
		}
		pf := float64(f.art.FrameIntervalMS) / 1000.0
		pub := map[string]float64{}
		nb := map[string]int{}
		for _, p := range periodesDrapeau(f.art) {
			pub[p.xuid] += float64(p.t1-p.t0+1) * pf
			nb[p.xuid]++
		}
		for _, x := range triees(clesDe(pub)) {
			var orc float64
			if r, ok := c.oracle[f.match][x]; ok {
				orc, _ = c.oTSV.f(r, "time_as_flag_carrier_seconds")
			}
			ecart := pub[x] - orc
			parP := ""
			if nb[x] > 0 {
				parP = fmt.Sprintf("%.2f", ecart/float64(nb[x]))
			}
			// simulation : une demi-fenetre de tic (0,5 s) ajoutee aux DEUX bornes
			demi := pub[x] + float64(nb[x])*1.0
			l = append(l, strings.Join([]string{f.court, famille(f.mode), x,
				c.gt[f.match][x], fmt.Sprint(nb[x]), fmt.Sprintf("%.1f", pub[x]),
				fmt.Sprintf("%.1f", orc), fmt.Sprintf("%.1f", ecart), parP,
				fmt.Sprintf("%.1f", demi), fmt.Sprint(demi > orc+0.05)}, "\t"))
		}
	}
	return l
}

// --- axe 5 : objet porte sans position du porteur ----------------------------

// positionAt reproduit `posOfPlayerAt` (apps/web/.../model/livesPosition.ts:54) : une vie qui
// couvre l'image rend une position ; sinon la derniere position d'une vie close depuis moins de
// `deathFrames` images ; sinon rien.
func aUnePosition(vies []track, frame, deathFrames int) bool {
	for _, v := range vies {
		if len(v.Points) == 0 {
			continue
		}
		d := v.Points[0].T
		fin := v.EndFrame
		if fin == 0 {
			fin = v.Points[len(v.Points)-1].T
		}
		if frame >= d && frame <= fin {
			return true
		}
	}
	for _, v := range vies {
		if len(v.Points) == 0 {
			continue
		}
		fin := v.EndFrame
		if fin == 0 {
			fin = v.Points[len(v.Points)-1].T
		}
		if d := frame - fin; d >= 0 && d <= deathFrames {
			return true
		}
	}
	return false
}

func (c *ctxAudit) axe5SansPosition() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "images_portees",
		"images_muettes", "pct_muettes", "periodes", "periodes_avec_trou",
		"images_muettes_expliquees_vehicule"}, "\t")}
	for _, f := range c.films {
		if famille(f.mode) != "CTF" || len(f.art.FlagCarries) == 0 {
			continue
		}
		vies := map[string][]track{}
		for _, t := range f.art.Tracks {
			if t.XUID != "" {
				vies[t.XUID] = append(vies[t.XUID], t)
			}
		}
		df := 15 // KILLPOS_WINDOW_MS 1500 / 100 ms
		var portees, muettes, avecTrou, vehicule int
		ps := periodesDrapeau(f.art)
		for _, p := range ps {
			trou := false
			for fr := p.t0; fr <= p.t1; fr++ {
				portees++
				if aUnePosition(vies[p.xuid], fr, df) {
					continue
				}
				muettes++
				trou = true
				if embarque(f.art, p.xuid, fr) {
					vehicule++
				}
			}
			if trou {
				avecTrou++
			}
		}
		pct := ""
		if portees > 0 {
			pct = fmt.Sprintf("%.1f", 100*float64(muettes)/float64(portees))
		}
		l = append(l, strings.Join([]string{f.court, famille(f.mode), f.mode,
			fmt.Sprint(portees), fmt.Sprint(muettes), pct, fmt.Sprint(len(ps)),
			fmt.Sprint(avecTrou), fmt.Sprint(vehicule)}, "\t"))
	}
	return l
}

// --- utilitaires -------------------------------------------------------------

func triees(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func clesDe(m map[string]float64) map[string]bool {
	out := map[string]bool{}
	for k := range m {
		out[k] = true
	}
	return out
}
