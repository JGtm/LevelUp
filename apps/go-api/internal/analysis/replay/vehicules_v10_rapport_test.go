package replay

// vehicules_v10_rapport_test.go — LE RAPPORT de l instrument V10 (la mesure vit dans
// `vehicules_v10_vitalite_test.go`). Separe pour tenir le seuil de 500 lignes par fichier.
//
// CE QU IL REND, DANS L ORDRE DES GATES ECRITS AVANT MESURE :
//
//	(a) SEPARATION des deux populations A (candidates) / B (temoin abandon) sur la derniere
//	    valeur d `i4` — histogrammes et meilleure coupure (Youden J) ;
//	(b) PALIER BAS TERMINAL : combien de vies en portent un, et la DISTRIBUTION de sa duree. Si
//	    l epave persiste, il y a un mode NON NUL de quelques secondes ; sinon, rien ;
//	(c) TEMOIN par decalage temporel (la meme grandeur 30 s avant la fin) ;
//	(d) CINEMATIQUE : le croisement vitalite terminale x vitesse terminale, qui separe l EPAVE
//	    (basse + arretee) de l ABANDON INTACT (pleine + arretee) et de la DISPARITION EN CONDUITE ;
//	(e) PLAUSIBILITE : la part de vies classees detruites doit rester une MINORITE (le garde-fou
//	    est la lecture fausse d avant V9, qui donnait 89 %).

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

func v10Rapport(t *testing.T, all []v10Vie) {
	t.Logf("\n########## V10 — VITALITE TERMINALE ET PALIER D EPAVE : %d vies avec i4 ##########",
		len(all))
	v10RapportSeparation(t, all)
	v10RapportPaliers(t, all)
	v10RapportCinematique(t, all)
	v10RapportLignes(t, all)
}

// v10RapportSeparation — gates (a) et (c).
func v10RapportSeparation(t *testing.T, all []v10Vie) {
	var a, b, aPente, bPente, temoinA, temoinB []float64
	for _, v := range all {
		switch v.classe {
		case "A_candidate":
			a = append(a, v.lastFrac)
			v10Push(&aPente, v.pente, v.penteOK)
			v10Push(&temoinA, v.temoin, v.temoinOK)
		case "B_abandon":
			b = append(b, v.lastFrac)
			v10Push(&bPente, v.pente, v.penteOK)
			v10Push(&temoinB, v.temoin, v.temoinOK)
		}
	}
	t.Logf("  (a) SEPARATION — derniere valeur d i4")
	t.Logf("      A (episode ouvert / ferme depuis <= %.0f s) : %s", v10OccupeS, v10Desc(a))
	t.Logf("          histogramme (10 buckets 0..1) : %v", v10Hist(a))
	t.Logf("      B (abandonnee depuis > %.0f s, ou jamais conduite) : %s", v10AbandonS, v10Desc(b))
	t.Logf("          histogramme (10 buckets 0..1) : %v", v10Hist(b))
	seuil, j, sens, spec := v10MeilleureCoupure(a, b)
	t.Logf("      MEILLEURE COUPURE (Youden J) : i4 final <= %.2f -> J=%.2f (sensibilite %.2f sur A, specificite %.2f sur B)",
		seuil, j, sens, spec)
	t.Logf("      PENTE terminale (fraction/s sur %.0f s) — A : %s", v10PenteS, v10Desc(aPente))
	t.Logf("      PENTE terminale — B : %s", v10Desc(bPente))
	t.Logf("  (c) TEMOIN TEMPOREL — la meme grandeur %.0f s AVANT la fin", v10TemoinS)
	t.Logf("      A : %s · histogramme %v", v10Desc(temoinA), v10Hist(temoinA))
	t.Logf("      B : %s · histogramme %v", v10Desc(temoinB), v10Hist(temoinB))
}

// v10RapportPaliers — gate (b) : le palier bas terminal, et sa duree.
func v10RapportPaliers(t *testing.T, all []v10Vie) {
	t.Logf("  (b) PALIER BAS TERMINAL")
	for s, seuil := range v10Seuils {
		var durees []float64
		nA, nB, nTot, n2 := 0, 0, 0, 0
		for _, v := range all {
			if v.palierN[s] == 0 {
				continue
			}
			nTot++
			if v.palierN[s] >= 2 {
				n2++
				durees = append(durees, v.palierS[s])
			}
			switch v.classe {
			case "A_candidate":
				nA++
			case "B_abandon":
				nB++
			}
		}
		t.Logf("      seuil %.2f : %d/%d vies portent un palier terminal (dont %d avec >= 2 lectures) · A=%d · B=%d",
			seuil, nTot, len(all), n2, nA, nB)
		t.Logf("          duree du palier (>= 2 lectures) : %s · histogramme en s [0-1,1-2,2-5,5-10,10-30,>30] %v",
			v10Desc(durees), v10HistDuree(durees))
	}
}

// v10RapportCinematique — gate (d) : le croisement vitalite x vitesse terminales.
func v10RapportCinematique(t *testing.T, all []v10Vie) {
	t.Logf("  (d) CINEMATIQUE — croisement (i4 final <= %.2f) x (vitesse finale < %.1f m/s), sur les vies dont la vitesse est lue",
		v10SeuilMoyen, v10ArretMPS)
	var epave, intactArret, basseRoule, pleineRoule int
	var sansV, jamaisRoule int
	for _, v := range all {
		if !v.vFinOK {
			sansV++
			continue
		}
		if v.vMaxOK && v.vMax < v10ArretMPS {
			jamaisRoule++
		}
		bas := v.lastFrac <= v10SeuilMoyen
		arret := v.vFin < v10ArretMPS
		switch {
		case bas && arret:
			epave++
		case !bas && arret:
			intactArret++
		case bas && !arret:
			basseRoule++
		default:
			pleineRoule++
		}
	}
	t.Logf("      EPAVE (basse + arretee) %d · ABANDON INTACT (pleine + arretee) %d · basse + en mouvement %d · pleine + en mouvement %d · vitesse non lue %d",
		epave, intactArret, basseRoule, pleineRoule, sansV)
	t.Logf("      (dont %d vies n ont JAMAIS roule de toute leur fenetre — vehicule jamais pris, pas une epave)",
		jamaisRoule)
	t.Logf("  (e) PLAUSIBILITE — part de vies « epave » sur les vies AVEC i4 : %d/%d = %.0f %%",
		epave, len(all), 100*v10Part(epave, len(all)))
}

func v10RapportLignes(t *testing.T, all []v10Vie) {
	sort.Slice(all, func(i, j int) bool { return all[i].lastFrac < all[j].lastFrac })
	t.Logf("  --- population A (candidates), toutes ---")
	for _, v := range all {
		if v.classe == "A_candidate" {
			t.Logf("    %s", v10Ligne(v))
		}
	}
	t.Logf("  --- les 20 vitalites finales les plus basses, toutes classes ---")
	for i, v := range all {
		if i >= 20 {
			break
		}
		t.Logf("    %s", v10Ligne(v))
	}
	if os.Getenv("V10_DUMP") != "" {
		t.Logf("  --- TOUTES les vies avec i4 (V10_DUMP), pour le recoupement manuel avec i11 ---")
		for _, v := range all {
			t.Logf("    %s vie=[%.1f..%.1f]s derniereLecture=%.1fs",
				v10Ligne(v), float64(v.loUS)/1e6, float64(v.hiUS)/1e6, float64(v.lastUS)/1e6)
		}
	}
}

func v10Ligne(v v10Vie) string {
	return fmt.Sprintf(
		"%s slot=%d gen=%d fam=%-9s n=%-4d i4fin=%.2f min=%.2f pente=%+.3f/s "+
			"palier10=%dx%.1fs palier25=%dx%.1fs vFin=%s vMax=%s ep=%d finApres=%.1fs %s",
		v.film, v.slot, v.gen, v10Ou(v.famille, "?"), v.nI4, v.lastFrac, v.minFrac, v.pente,
		v.palierN[0], v.palierS[0], v.palierN[1], v.palierS[1],
		v10Vit(v.vFin, v.vFinOK), v10Vit(v.vMax, v.vMaxOK), v.rides,
		v10ClampInf(v.finApres), v.classe)
}

func v10Vit(v float64, ok bool) string {
	if !ok {
		return "n/a"
	}
	return fmt.Sprintf("%.1f", v)
}

func v10Push(dst *[]float64, v float64, ok bool) {
	if ok {
		*dst = append(*dst, v)
	}
}

func v10Ou(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func v10ClampInf(v float64) float64 {
	if v >= v10Inf {
		return -1
	}
	return v
}

func v10Part(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func v10Desc(v []float64) string {
	if len(v) == 0 {
		return "n=0"
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	q := func(p float64) float64 { return c[int(p*float64(len(c)-1))] }
	return fmt.Sprintf("n=%d min=%.2f p10=%.2f mediane=%.2f p90=%.2f max=%.2f",
		len(c), c[0], q(0.10), q(0.50), q(0.90), c[len(c)-1])
}

func v10Hist(v []float64) [10]int {
	var h [10]int
	for _, x := range v {
		i := int(x * 10)
		if i > 9 {
			i = 9
		}
		if i < 0 {
			i = 0
		}
		h[i]++
	}
	return h
}

// v10HistDuree : buckets [0-1[, [1-2[, [2-5[, [5-10[, [10-30[, [30+[ secondes.
func v10HistDuree(v []float64) [6]int {
	var h [6]int
	for _, x := range v {
		switch {
		case x < 1:
			h[0]++
		case x < 2:
			h[1]++
		case x < 5:
			h[2]++
		case x < 10:
			h[3]++
		case x < 30:
			h[4]++
		default:
			h[5]++
		}
	}
	return h
}

// v10MeilleureCoupure cherche le seuil `i4 final <= s` qui maximise Youden J sur les deux
// populations. Rend le seuil, J, la sensibilite (part de A sous le seuil) et la specificite
// (part de B au-dessus).
func v10MeilleureCoupure(a, b []float64) (seuil, j, sens, spec float64) {
	if len(a) == 0 || len(b) == 0 {
		return 0, 0, 0, 0
	}
	best := -2.0
	for s := 0.0; s <= 1.0001; s += 0.01 {
		na, nb := 0, 0
		for _, x := range a {
			if x <= s {
				na++
			}
		}
		for _, x := range b {
			if x <= s {
				nb++
			}
		}
		se := float64(na) / float64(len(a))
		sp := 1 - float64(nb)/float64(len(b))
		if v := se + sp - 1; v > best {
			best, seuil, sens, spec = v, s, se, sp
		}
	}
	return seuil, best, sens, spec
}
