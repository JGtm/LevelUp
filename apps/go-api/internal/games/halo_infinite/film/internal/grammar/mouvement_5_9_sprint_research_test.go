//go:build research

package grammar

// mouvement_5_9_sprint_research_test.go — LA FENTE DU SPRINT, VERIFIEE PAR LE CONTENU (lot
// 5.9.5). Scinde de `mouvement_5_9_research_test.go` par DEPLACEMENT PUR le 2026-09-21 : le
// fichier passait 500 lignes, et le ratchet de taille refuse de grandir. Aucune fonction n est
// modifiee ; elles changent seulement de fichier, dans le MEME paquet.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ---------------------------------------------------------------------------
// 5.9.5 — LA FENTE DU SPRINT, VERIFIEE PAR LE CONTENU
//
// L image NOMME les trois fentes (`FUN_1407e9ce4` aiguille sur le groupe de tag et appelle un
// desenregistreur qui teste l index actif contre SA fente) :
//
//	'saev' esquive -> FUN_14319d0ac, fente 0 ; 'sasp' SPRINT -> FUN_14319d1ec, fente 1 ;
//	'sagh' grappin -> FUN_14319d14c, fente 2.
//
// Le flux d `i57` ecrit `bloc+3 = R(2) - 1`, donc le BRUT `2` designe la fente 1 — le sprint.
// Ce qui suit le VERIFIE sur le contenu : la vitesse au sol pendant les intervalles de fente 1
// doit se separer de celle du reste du film, et les intervalles de fente 2 doivent tomber sur
// les paires de grappin.
// ---------------------------------------------------------------------------

// m59BrutSprint / m59BrutGrappin / m59BrutEsquive : les valeurs BRUTES d `i57` (flux = fente+1).
const (
	m59BrutAucune  = 0
	m59BrutEsquive = 1
	m59BrutSprint  = 2
	m59BrutGrappin = 3
)

// m59Intervalle est UN episode de capacite active, par vie.
type m59Intervalle struct {
	slot   uint32
	t0, t1 uint64
}

// TestMouvement59FenteDuSprint confronte les intervalles de la fente 1 a la vitesse au sol.
func TestMouvement59FenteDuSprint(t *testing.T) {
	rec, ok := m57PasseRetenue(t)
	if !ok {
		return
	}
	m59Repartition(t, rec)
	eps := m59IntervallesDe(rec, m59BrutSprint)
	t.Logf("FENTE 1 ('sasp') : %d intervalles sur %d vies · duree %s", len(eps), m59NbVies(eps),
		m59Durees(eps))
	m59VitesseDansEtHors(t, rec, eps)
	jug, muets := m59Jugeables(rec, eps)
	t.Logf("JUGEABLES : %d intervalles sur %d ; %d ECARTES parce que la vie a cesse de "+
		"transmettre plus de %d us pendant l intervalle — `i57` ne voyage que sur CHANGEMENT, "+
		"donc une fente reste armee pendant un silence de replication, et un tel intervalle "+
		"n est pas un geste mesurable (borne de tenue deja validee au 5.9.4)",
		len(jug), len(eps), muets, types.SpartanJumpHoldMaxUS)
	t.Logf("  duree des JUGEABLES : %s", m59Durees(jug))
	m59ScoreSprint(t, rec, jug)
	m59VitesseAtteinte(t, rec, jug)
	m59AccroupiExclusif(t, rec, eps)
	gr := m59IntervallesDe(rec, m59BrutGrappin)
	t.Logf("FENTE 2 ('sagh', controle gratuit) : %d intervalles sur %d vies · duree %s",
		len(gr), m59NbVies(gr), m59Durees(gr))
	m59VitesseDansEtHors(t, rec, gr)
}

// m59Repartition publie le compte des lectures par valeur brute.
func m59Repartition(t *testing.T, rec *m57Rec) {
	t.Helper()
	par := map[uint64]int{}
	for _, c := range rec.capa {
		par[c.brut]++
	}
	noms := map[uint64]string{m59BrutAucune: "aucune fente", m59BrutEsquive: "fente 0 'saev'",
		m59BrutSprint: "fente 1 'sasp' SPRINT", m59BrutGrappin: "fente 2 'sagh' grappin"}
	for v := uint64(0); v < 4; v++ {
		t.Logf("  brut %d = %-24s : %5d lectures", v, noms[v], par[v])
	}
}

// m59IntervallesDe replie les lectures d `i57` en intervalles pour UNE valeur brute.
//
// UNE LECTURE EST UNE TRANSITION : la fente active ne voyage que quand elle CHANGE. Un
// intervalle s ouvre sur la valeur cherchee et se ferme sur la premiere valeur differente de la
// MEME vie. Un intervalle jamais ferme est JETE (meme refus que le saut derive).
func m59IntervallesDe(rec *m57Rec, brut uint64) []m59Intervalle {
	parSlot := map[uint32][]m57Cap{}
	for _, c := range rec.capa {
		parSlot[c.slot] = append(parSlot[c.slot], c)
	}
	var slots []uint32
	for s := range parSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []m59Intervalle
	for _, s := range slots {
		cs := parSlot[s]
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].ts < cs[j].ts })
		ouvert := uint64(0)
		actif := false
		for _, c := range cs {
			switch {
			case c.brut == brut && !actif:
				ouvert, actif = c.ts, true
			case c.brut != brut && actif:
				out = append(out, m59Intervalle{slot: s, t0: ouvert, t1: c.ts})
				actif = false
			}
		}
	}
	return out
}

// m59NbVies compte les vies distinctes portant un intervalle.
func m59NbVies(eps []m59Intervalle) int {
	vies := map[uint32]struct{}{}
	for _, e := range eps {
		vies[e.slot] = struct{}{}
	}
	return len(vies)
}

// m59VitesseDansEtHors publie la distribution de la vitesse au sol DANS les intervalles et HORS,
// avec leurs medianes et leurs deciles. Le SEUIL ne se choisit pas : il se LIT sur ces deux
// distributions.
func m59VitesseDansEtHors(t *testing.T, rec *m57Rec, eps []m59Intervalle) {
	t.Helper()
	if len(eps) == 0 {
		t.Logf("  aucune vitesse a confronter : zero intervalle")
		return
	}
	parSlot := map[uint32][]m59Intervalle{}
	for _, e := range eps {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	var dans, hors []float64
	for _, v := range rec.vit {
		dedans := false
		for _, e := range parSlot[v.slot] {
			if v.ts >= e.t0 && v.ts <= e.t1 {
				dedans = true
				break
			}
		}
		if dedans {
			dans = append(dans, v.sol)
		} else {
			hors = append(hors, v.sol)
		}
	}
	t.Logf("  vitesse au sol DANS : %s", m59Quantiles(dans))
	t.Logf("  vitesse au sol HORS : %s", m59Quantiles(hors))
}

// m59Quantiles rend n, mediane, p10 et p90 d une serie.
func m59Quantiles(xs []float64) string {
	if len(xs) == 0 {
		return "n=0"
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	q := func(f float64) float64 { return cp[int(f*float64(len(cp)-1))] }
	return fmt.Sprintf("n=%d · p10 %.2f · mediane %.2f · p90 %.2f m/s", len(cp), q(0.10), q(0.50),
		q(0.90))
}

// m59ScoreSprint note les intervalles de la fente 1 contre L ETIQUETTE PHYSIQUE : un PLATEAU de
// vitesse au sol tenu plus d une demi-seconde, au-dessus d un seuil qui n est pas CHOISI mais LU
// sur la mesure — le milieu entre la valeur mediane des plateaux tenus HORS intervalle (`Vm`, la
// marche) et celle des plateaux tenus DEDANS (`Vs`, le sprint).
func m59ScoreSprint(t *testing.T, rec *m57Rec, eps []m59Intervalle) {
	t.Helper()
	pls := m575Plateaux(rec)
	if len(pls) == 0 || len(eps) == 0 {
		t.Logf("SCORE : impossible (%d plateaux, %d intervalles)", len(pls), len(eps))
		return
	}
	parSlot := map[uint32][]m59Intervalle{}
	for _, e := range eps {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	var dedans, dehors []float64
	for _, p := range pls {
		if m59Recouvre(parSlot[p.slot], p.t0, p.t1) {
			dedans = append(dedans, p.valeur)
		} else {
			dehors = append(dehors, p.valeur)
		}
	}
	vm, vs := m59Mediane(dehors), m59Mediane(dedans)
	seuil := (vm + vs) / 2
	t.Logf("SEUIL LU, PAS CHOISI : Vm %.2f (mediane de %d plateaux hors intervalle) · "+
		"Vs %.2f (mediane de %d plateaux dedans) · seuil (Vm+Vs)/2 = %.2f m/s",
		vm, len(dehors), vs, len(dedans), seuil)
	var hauts []m575Plateau
	for _, p := range pls {
		if p.valeur >= seuil {
			hauts = append(hauts, p)
		}
	}
	parSlotH := map[uint32][]m59Intervalle{}
	for _, p := range hauts {
		parSlotH[p.slot] = append(parSlotH[p.slot], m59Intervalle{slot: p.slot, t0: p.t0, t1: p.t1})
	}
	var bons int
	for _, e := range eps {
		if m59Recouvre(parSlotH[e.slot], e.t0, e.t1) {
			bons++
		}
	}
	var couverts int
	for _, p := range hauts {
		if m59Recouvre(parSlot[p.slot], p.t0, p.t1) {
			couverts++
		}
	}
	t.Logf("SCORE FENTE 1 CONTRE LE PLATEAU HAUT (>= %.2f m/s, tenu > %.1f s) : "+
		"%d intervalles, %d sur un plateau haut (precision %.1f %%) · "+
		"%d plateaux hauts, %d couverts (rappel %.1f %%)",
		seuil, m575PlateauMinS, len(eps), bons, m533bPart(bons, len(eps)),
		len(hauts), couverts, m533bPart(couverts, len(hauts)))
}

// m59Recouvre dit si l un des intervalles chevauche [t0, t1].
func m59Recouvre(eps []m59Intervalle, t0, t1 uint64) bool {
	for _, e := range eps {
		if e.t0 <= t1 && t0 <= e.t1 {
			return true
		}
	}
	return false
}

// m59Mediane rend la mediane d une serie (zero si vide).
func m59Mediane(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	return cp[len(cp)/2]
}

// m59AccroupiExclusif verifie que les intervalles de la fente 1 ne recouvrent PAS l accroupi :
// on ne sprinte pas accroupi, et c est un controle que le film peut trancher seul.
func m59AccroupiExclusif(t *testing.T, rec *m57Rec, eps []m59Intervalle) {
	t.Helper()
	parSlot := map[uint32][]m59Intervalle{}
	for _, e := range eps {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	var poses, dedans int
	for _, a := range rec.accr {
		if !a.on {
			continue
		}
		poses++
		if m59Recouvre(parSlot[a.slot], a.ts, a.ts) {
			dedans++
		}
	}
	t.Logf("EXCLUSION ACCROUPI : %d poses d accroupi, %d tombent DANS un intervalle de fente 1 "+
		"(%.1f %%) — on ne sprinte pas accroupi", poses, dedans, m533bPart(dedans, poses))
}

// m59Durees rend les quantiles de duree des intervalles, en secondes. C EST LE CHIFFRE QUI DIT
// SI L INTERVALLE EST UN GESTE OU UN SILENCE DE REPLICATION : `i57` ne voyage que sur
// changement, donc un intervalle long peut n etre qu une fente restee armee pendant que la vie
// ne transmettait plus.
func m59Durees(eps []m59Intervalle) string {
	if len(eps) == 0 {
		return "n=0"
	}
	xs := make([]float64, 0, len(eps))
	for _, e := range eps {
		xs = append(xs, float64(e.t1-e.t0)/1e6)
	}
	sort.Float64s(xs)
	q := func(f float64) float64 { return xs[int(f*float64(len(xs)-1))] }
	return fmt.Sprintf("p10 %.2f s · mediane %.2f s · p90 %.2f s · max %.1f s",
		q(0.10), q(0.50), q(0.90), xs[len(xs)-1])
}

// m59Jugeables ecarte les intervalles que le film ne permet pas de juger : ceux pendant lesquels
// la vie a cesse de transmettre sa vitesse plus longtemps que la borne de tenue.
//
// CE N EST PAS UN FILTRE DE COMMODITE. `i57` ne voyage que sur CHANGEMENT : une fente reste
// armee tant que rien ne la change, y compris pendant que la vie ne replique plus. L intervalle
// brut le plus long de `bfecd02b` dure 166,7 s — ce n est pas un sprint, c est un silence. La
// production, elle, borne ses intervalles aux FENETRES DE VIE (`trackFrameWindows`) et les
// ferme a la mort ; l instrument de grammaire n a pas les vies, il emploie donc la borne de
// tenue, qui est la meme regle vue depuis le flux.
func m59Jugeables(rec *m57Rec, eps []m59Intervalle) ([]m59Intervalle, int) {
	parSlot := map[uint32][]m57Vit{}
	for _, v := range rec.vit {
		parSlot[v.slot] = append(parSlot[v.slot], v)
	}
	for s := range parSlot {
		vs := parSlot[s]
		sort.SliceStable(vs, func(i, j int) bool { return vs[i].ts < vs[j].ts })
	}
	var out []m59Intervalle
	var muets int
	for _, e := range eps {
		if m59Continue(parSlot[e.slot], e.t0, e.t1) {
			out = append(out, e)
			continue
		}
		muets++
	}
	return out, muets
}

// m59Continue dit si la vie a transmis sa vitesse sur tout [t0, t1] sans trou depassant la borne.
func m59Continue(vs []m57Vit, t0, t1 uint64) bool {
	prec := t0
	var vu bool
	for _, v := range vs {
		if v.ts < t0 {
			continue
		}
		if v.ts > t1 {
			break
		}
		if v.ts-prec > types.SpartanJumpHoldMaxUS {
			return false
		}
		prec, vu = v.ts, true
	}
	return vu && t1-prec <= types.SpartanJumpHoldMaxUS
}

// m59VitesseAtteinte confronte la vitesse MAXIMALE atteinte pendant chaque intervalle de fente 1
// a celle d un TEMOIN APPARIE : la meme vie, la meme duree, cinq secondes plus tot.
//
// C EST LA MESURE QUI TRANCHE, et elle ne depend d aucun seuil. Un score « intervalle contre
// plateau » punit les intervalles courts : la vitesse monte par une RAMPE
// (`spartan_ability_sprint_seconds_to_full_speed`, cf. `FUN_1431a2c94`), donc un sprint d une
// demi-seconde n atteint jamais un plateau TENU une demi-seconde. Le temoin apparie, lui, pose
// la seule question qui compte : pendant ces instants-la, cette vie-la va-t-elle plus vite ?
func m59VitesseAtteinte(t *testing.T, rec *m57Rec, eps []m59Intervalle) {
	t.Helper()
	parSlot := map[uint32][]m57Vit{}
	for _, v := range rec.vit {
		parSlot[v.slot] = append(parSlot[v.slot], v)
	}
	for s := range parSlot {
		vs := parSlot[s]
		sort.SliceStable(vs, func(i, j int) bool { return vs[i].ts < vs[j].ts })
	}
	var dans, temoin []float64
	var gagne, perd, exaequo, sansTemoin int
	for _, e := range eps {
		a, okA := m59VMax(parSlot[e.slot], e.t0, e.t1)
		if !okA {
			continue
		}
		dans = append(dans, a)
		d := e.t1 - e.t0
		if e.t0 < 5_000_000+d {
			sansTemoin++
			continue
		}
		b, okB := m59VMax(parSlot[e.slot], e.t0-5_000_000-d, e.t0-5_000_000)
		if !okB {
			sansTemoin++
			continue
		}
		temoin = append(temoin, b)
		switch {
		case a > b+0.05:
			gagne++
		case b > a+0.05:
			perd++
		default:
			exaequo++
		}
	}
	t.Logf("VITESSE MAX ATTEINTE, intervalle : %s", m59Quantiles(dans))
	t.Logf("VITESSE MAX ATTEINTE, temoin (meme vie, meme duree, -5 s) : %s", m59Quantiles(temoin))
	t.Logf("APPARIEMENT : %d intervalles PLUS RAPIDES que leur temoin · %d moins rapides · "+
		"%d a egalite (a 0,05 m/s pres) · %d sans temoin → %.1f %% de victoires",
		gagne, perd, exaequo, sansTemoin, m533bPart(gagne, gagne+perd+exaequo))
}

// m59VMax rend la vitesse au sol maximale lue dans [t0, t1].
func m59VMax(vs []m57Vit, t0, t1 uint64) (float64, bool) {
	var max float64
	var vu bool
	for _, v := range vs {
		if v.ts < t0 {
			continue
		}
		if v.ts > t1 {
			break
		}
		if !vu || v.sol > max {
			max, vu = v.sol, true
		}
	}
	return max, vu
}
