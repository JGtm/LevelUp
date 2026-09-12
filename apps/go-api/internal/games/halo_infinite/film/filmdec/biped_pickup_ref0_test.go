package filmdec

// biped_pickup_ref0_test.go — LOT 2 DU CHANTIER RAMASSAGE : QUI ramasse ? La reference ref0
// du type 9 (domaine 2, R(8), presente a 100 %) n'est pas identifiee a l'issue du lot 1.
//
// DEUX HYPOTHESES, ET IL FAUT LES DEPARTAGER, PAS SEULEMENT EN CONFIRMER UNE :
//
//	H-A : ref0 = LE RAMASSEUR. Le lecteur de l'exe (FUN_1406d3140) reconstruit la reference
//	      comme `(gen<<30) | (base + index)` ou `base = DAT_1451f98d0[dom*2]` est une valeur
//	      de RUNTIME. Le negatif du lot 1 (« ref0 n'est pas un slot de bipede, 25 valeurs
//	      distinctes ») portait sur l'index BRUT, sans la base : c'est EXACTEMENT le piege
//	      deja paye sur damage_aftermath, ou l'ajout de la base a transforme un « handle
//	      irresoluble » en slot de bipede. Il faut donc balayer la base.
//	H-B : ref0 = L'OBJET RAMASSE (l'instance monde : arme au sol, equipement).
//
// LA VERITE TERRAIN EXISTE DEJA, on ne la refabrique pas : le lot 1 a apparie 21 (000d5950)
// et 11 (00502e52) evenements type 9 a une emission i43..i46 portant LA MEME arme a moins de
// 500 ms. Or une emission i43..i46 est lue sur un record delta ANCRE : son slot de bipede est
// connu. Pour ces paires, LE RAMASSEUR EST CONNU. C'est contre lui qu'on juge.
//
// LEÇON damage_aftermath RESPECTEE : le taux de « liage » (l'index tombe-t-il dans une plage
// de bipedes ?) est un proxy FAIBLE. Le juge retenu ici est la correspondance EXACTE par
// evenement — `base + index == slot du ramasseur connu` — avec des temoins.
//
// Garde BIPED_PICKUP_FILM.

import (
	"sort"
	"testing"
)

// bpkPaire est un evenement type 9 apparie, sans ambiguite, a une prise dont le SLOT du
// bipede ramasseur est connu.
type bpkPaire struct {
	ev   bpkEvent
	slot uint32
	fam  uint32
}

// bpkPaires construit la verite terrain : pour chaque prise d'i43..i46 (taken/swapped), les
// evenements type 9 portant LA MEME arme a moins de bpkTolUS. On ne garde que les
// appariements SANS AMBIGUITE (exactement un candidat) — un appariement multiple ne permet
// pas de dire quel evenement porte quel ramasseur, et le compter fausserait le juge.
func bpkPaires(evs []bpkEvent, chg []HeldWeaponChange) (paires []bpkPaire, ambigus int) {
	for _, c := range chg {
		if c.Kind != HeldWeaponTaken && c.Kind != HeldWeaponSwapped {
			continue
		}
		var cand []bpkEvent
		for _, e := range evs {
			if !e.ObjetPresent || e.Objet != c.Family {
				continue
			}
			d := int64(e.TimestampUS) - int64(c.TimestampUS)
			if d < 0 {
				d = -d
			}
			if d <= bpkTolUS {
				cand = append(cand, e)
			}
		}
		switch len(cand) {
		case 0:
		case 1:
			paires = append(paires, bpkPaire{ev: cand[0], slot: c.Slot, fam: c.Family})
		default:
			ambigus++
		}
	}
	return paires, ambigus
}

// TestBipedPickupRef0Base — H-A. Si ref0 designe le ramasseur, alors il existe UNE base b
// telle que `b + ref0.index == slot du ramasseur` sur la quasi-totalite des paires. On ne
// balaye meme pas : on calcule directement l'ecart `slot - index` paire par paire et on
// regarde son histogramme. Une base constante donne un histogramme a UN pic.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	A1 — une valeur unique de (slot - index) couvre >= 70 % des paires non ambigues.
//	A2 — le TEMOIN (memes paires, appariement permute d'un cran : l'evenement i est associe
//	     au slot de la paire i+1) doit rester sous 25 % sur son propre mode. Sans ce temoin,
//	     un mode eleve pourrait n'etre qu'un effet de la faible cardinalite des slots.
//	A3 — H-A est RETENUE si A1 et A2 tiennent ET si la base trouvee est la MEME sur les deux
//	     films OU s'explique (c'est une valeur de runtime : elle PEUT differer, mais alors
//	     chaque film doit avoir son pic net).
//
// Si A1 echoue, H-A est REFUTEE pour une base constante, et le negatif est publie tel quel.
func TestBipedPickupRef0Base(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	evs := bpkCollecte(t, f)
	if len(evs) == 0 {
		t.Skip("aucun biped_pickup sur ce film")
	}
	ref := hwKeyframeRef(t, f.dir)
	chg, _, err := ScanFilmHeldWeaponChanges(f.dir, ref.setAt)
	if err != nil {
		t.Fatalf("balayage des changements d'arme : %v", err)
	}
	paires, ambigus := bpkPaires(evs, chg)
	t.Logf("== H-A : ref0 = LE RAMASSEUR ? · %s ==", f.dir)
	t.Logf("verite terrain : %d paires NON AMBIGUES (evenement type 9 <-> prise i43..i46 de la meme arme, <= %d ms) · %d appariements ambigus ecartes",
		len(paires), bpkTolUS/1000, ambigus)
	if len(paires) < 8 {
		t.Logf("VERDICT : %d paires, trop peu pour juger. Rien n'est conclu.", len(paires))
		return
	}

	ecarts := map[int64]int{}
	ecartsParClasse := map[uint64]map[int64]int{}
	for _, p := range paires {
		d := int64(p.slot) - int64(p.ev.Ref0)
		ecarts[d]++
		if ecartsParClasse[p.ev.Kind] == nil {
			ecartsParClasse[p.ev.Kind] = map[int64]int{}
		}
		ecartsParClasse[p.ev.Kind][d]++
	}
	// TEMOIN : le meme calcul avec l'appariement PERMUTE d'un cran.
	temoin := map[int64]int{}
	for i, p := range paires {
		q := paires[(i+1)%len(paires)]
		temoin[int64(q.slot)-int64(p.ev.Ref0)]++
	}

	mode, modeN := int64(0), 0
	for d, n := range ecarts {
		if n > modeN || (n == modeN && d < mode) {
			mode, modeN = d, n
		}
	}
	tMode, tModeN := int64(0), 0
	for d, n := range temoin {
		if n > tModeN || (n == tModeN && d < tMode) {
			tMode, tModeN = d, n
		}
	}
	t.Logf("ecart (slot du ramasseur - index de ref0) : %d valeurs distinctes sur %d paires — %s",
		len(ecarts), len(paires), bpkTopI64(ecarts, 10))
	t.Logf("  MODE : base = %d, sur %d / %d paires (%.1f %%)",
		mode, modeN, len(paires), bpkPct(modeN, len(paires)))
	t.Logf("TEMOIN (appariement permute d'un cran) : %d valeurs distinctes — mode %d sur %d (%.1f %%)",
		len(temoin), tMode, tModeN, bpkPct(tModeN, len(paires)))
	for k := uint64(0); k < 8; k++ {
		if m := ecartsParClasse[k]; len(m) > 0 {
			t.Logf("  classe R(3)=%d : %s", k, bpkTopI64(m, 6))
		}
	}
	// Rappel de la distribution brute, pour lire l'histogramme.
	slots, idxs := map[uint64]int{}, map[uint64]int{}
	for _, p := range paires {
		slots[uint64(p.slot)]++
		idxs[p.ev.Ref0]++
	}
	t.Logf("  slots des ramasseurs connus : %s", bpkTop(slots, 10))
	t.Logf("  index de ref0 apparies      : %s", bpkTop(idxs, 10))
	okA1 := bpkPct(modeN, len(paires)) >= 70
	okA2 := bpkPct(tModeN, len(paires)) < 25
	t.Logf("VERDICT A1 (un ecart unique >= 70 %%) : %s · A2 (temoin < 25 %%) : %s · H-A : %s",
		bpkVerdict(okA1), bpkVerdict(okA2), bpkVerdict(okA1 && okA2))
}

// bpkTopI64 rend les k entrees les plus frequentes d'un histogramme a cle signee.
func bpkTopI64(m map[int64]int, k int) string {
	type kv struct {
		k int64
		v int
	}
	s := make([]kv, 0, len(m))
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool {
		if s[i].v != s[j].v {
			return s[i].v > s[j].v
		}
		return s[i].k < s[j].k
	})
	if len(s) > k {
		s = s[:k]
	}
	out := ""
	for i, e := range s {
		if i > 0 {
			out += " · "
		}
		out += fmtI64(e.k, e.v)
	}
	return out
}

func fmtI64(k int64, v int) string {
	return itoa64(k) + " x" + itoa64(int64(v))
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [24]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
