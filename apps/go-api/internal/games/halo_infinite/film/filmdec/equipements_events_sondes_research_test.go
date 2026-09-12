package filmdec

// equipements_events_sondes_research_test.go — second volet de l instrument R5 (en-tete et
// usage : equipements_events_research_test.go) : croisement TEMPS SEUL, hypotheses de chaine
// de references, sonde de continuation de liste sur les tetes 103, et bilans parc.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// equipEvTempsSeul : croisement TEMPS SEUL pour les types de equipEvCanauxTemps — le type
// date-t-il le geste meme si sa ref0 ne rend pas le slot ? Dump occurrence par occurrence
// (tete hexa + hypotheses de chaine de refs) pour sourcer la grammaire plus tard.
func equipEvTempsSeul(t *testing.T, occs []equipEvOcc, truths []equipEvTruth,
	precAgg map[int][2]int, rappelAgg map[string]map[int][2]int) {
	t.Helper()
	for _, typ := range equipEvTypesPresents(occs) {
		prefixes := equipEvCanauxTemps[typ]
		if prefixes == nil {
			continue
		}
		var tot, ok int
		var deltas []int64
		canaux := map[string]int{}
		for _, o := range occs {
			if o.typ != typ {
				continue
			}
			tot++
			canal, slot, dt, found := equipEvMatchTemps(o.tsMS, prefixes, truths)
			marque := "AUCUN usage de ce canal dans le film"
			if canal != "" {
				marque = fmt.Sprintf("plus proche %s slot %d dt %+d ms (HORS fenetre)", canal, slot, dt)
			}
			if found {
				ok++
				canaux[canal]++
				deltas = append(deltas, dt)
				marque = fmt.Sprintf("plus proche %s slot %d dt %+d ms", canal, slot, dt)
			}
			t.Logf("    [type %d] @%d ms tete % X g0=%v w8=%d chA=%d chB=%d -> %s",
				typ, o.tsMS, o.tete[:equipEvMin(len(o.tete), 12)], o.slot8 >= 0, o.slot8,
				equipEvChaine(o.tete, 13), equipEvChaine(o.tete, 8), marque)
		}
		agg := precAgg[typ]
		agg[0] += ok
		agg[1] += tot
		precAgg[typ] = agg
		t.Logf("  TEMPS SEUL type %3d %-36s : precision %d/%d (fenetre %d ms) ; canaux %v ; dt med %s",
			typ, equipEvSuspects[typ], ok, tot, equipEvFenetreMS, canaux, equipEvMedianeMS(deltas))
		equipEvRappelTemps(t, typ, prefixes, occs, truths, rappelAgg)
	}
}

// equipEvRappelTemps : chaque usage des canaux vises a-t-il UNE occurrence du type dans la
// fenetre (temps seul, slot ignore) ?
func equipEvRappelTemps(t *testing.T, typ int, prefixes []string, occs []equipEvOcc,
	truths []equipEvTruth, rappelAgg map[string]map[int][2]int) {
	t.Helper()
	parCanal := map[string][2]int{}
	for _, tr := range truths {
		if !equipEvCanalVise(tr.canal, prefixes) {
			continue
		}
		trouve := false
		for _, o := range occs {
			if o.typ == typ && equipEvAbs(o.tsMS-tr.tMs) <= equipEvFenetreMS {
				trouve = true
				break
			}
		}
		a := parCanal[tr.canal]
		if trouve {
			a[0]++
		}
		a[1]++
		parCanal[tr.canal] = a
	}
	for canal, a := range parCanal {
		if rappelAgg[canal] == nil {
			rappelAgg[canal] = map[int][2]int{}
		}
		g := rappelAgg[canal][typ]
		g[0] += a[0]
		g[1] += a[1]
		rappelAgg[canal][typ] = g
		t.Logf("  TEMPS SEUL rappel %-22s par type %3d : %d/%d", canal, typ, a[0], a[1])
	}
}

func equipEvCanalVise(canal string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(canal, p) {
			return true
		}
	}
	return false
}

// equipEvMatchTemps : usage vise le plus proche en temps, slot ignore. Rend TOUJOURS le plus
// proche (canal vide = aucun usage de ce canal dans le film) ; le booleen dit si |dt| est
// dans la fenetre.
func equipEvMatchTemps(tsMS int64, prefixes []string, truths []equipEvTruth) (string, int, int64, bool) {
	best, canal, slot, found := int64(0), "", 0, false
	for _, tr := range truths {
		if !equipEvCanalVise(tr.canal, prefixes) {
			continue
		}
		dt := tsMS - tr.tMs
		if !found || equipEvAbs(dt) < equipEvAbs(best) {
			best, canal, slot, found = dt, tr.canal, tr.slot, true
		}
	}
	return canal, slot, best, found && equipEvAbs(best) <= equipEvFenetreMS
}

// equipEvChaine : hypothese de CHAINE de references « ref0 puis ref1 » — ref0 de largeur w0
// (l'objet cree ?), ref1 de largeur 8 base 512 (le porteur ?). Rend le slot candidat de ref1,
// ou -1 (porte fermee / tete trop courte). Une valeur n'est PAS une preuve : c'est une piste
// a re-sourcer cote exe (vtable+0x58 du descripteur du type).
func equipEvChaine(tete []byte, w0 uint) int {
	br := NewBitReader(tete)
	br.Skip(9)
	if br.Remaining() < 1 || !br.ReadBit() {
		return -1
	}
	if br.Remaining() < int(w0)+2+1+8+2 {
		return -1
	}
	br.Skip(int(w0) + 2) // index + generation de ref0
	if !br.ReadBit() {   // porte de ref1
		return -1
	}
	return int(br.ReadBits(8)) + 512
}

// equipEvSonde103 : la seule mesure a bas cout de la question « la tete est-elle le seul
// evenement du paquet ? ». Le type 103 a une charge PROUVEE de 0 bit (annexe A) : sous
// l'hypothese de largeur ref0/ref1 = 13 bits (les valeurs observees derivent avec
// l'allocation d'objets du monde — bande objets, pas bipede), l'evenement se termine et le
// bit suivant est le drapeau de continuation de la liste. HYPOTHESE, pas une grammaire
// sourcee : les largeurs exactes des domaines du type 103 restent a lire dans l'exe
// (vtable+0x58 du descripteur).
func equipEvSonde103(t *testing.T, occs []equipEvOcc, agg map[string]int) {
	t.Helper()
	for _, o := range occs {
		if o.typ != 103 {
			continue
		}
		br := NewBitReader(o.tete)
		br.Skip(9)
		if br.Remaining() < 1 || !br.ReadBit() {
			agg["ref0_fermee"]++
			continue
		}
		br.Skip(13 + 2)
		if br.Remaining() < 1 {
			agg["tete_trop_courte"]++
			continue
		}
		if br.ReadBit() { // porte ref1
			if br.Remaining() < 13+2 {
				agg["tete_trop_courte"]++
				continue
			}
			br.Skip(13 + 2)
		}
		if br.Remaining() < 1 {
			agg["tete_trop_courte"]++
			continue
		}
		if br.ReadBit() { // porte ref2 : largeur inconnue, on ne sait plus avancer
			agg["ref2_ouverte_indecodable"]++
			continue
		}
		if br.Remaining() < 1 {
			agg["tete_trop_courte"]++
			continue
		}
		if !br.ReadBit() { // drapeau de continuation de la liste
			agg["seul_evenement"]++
			continue
		}
		agg["suivi_d_un_autre"]++
		if br.Remaining() >= 7 {
			agg[fmt.Sprintf("suivant_type_%d", br.ReadBits(7))]++
		}
	}
}

func equipEvMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// equipEvBilanTemps : table parc du croisement temps seul.
func equipEvBilanTemps(t *testing.T, precAgg map[int][2]int, rappelAgg map[string]map[int][2]int) {
	t.Helper()
	t.Logf("== BILAN PARC — TEMPS SEUL, precision par type (occurrences datant un usage / total) ==")
	types := make([]int, 0, len(precAgg))
	for typ := range precAgg {
		types = append(types, typ)
	}
	sort.Ints(types)
	for _, typ := range types {
		a := precAgg[typ]
		t.Logf("  type %3d %-36s : %d/%d", typ, equipEvSuspects[typ], a[0], a[1])
	}
	t.Logf("== BILAN PARC — TEMPS SEUL, rappel par canal x type ==")
	canaux := make([]string, 0, len(rappelAgg))
	for c := range rappelAgg {
		canaux = append(canaux, c)
	}
	sort.Strings(canaux)
	for _, c := range canaux {
		for _, typ := range types {
			a := rappelAgg[c][typ]
			if a[1] == 0 {
				continue
			}
			t.Logf("  %-22s x type %3d %-36s : %d/%d", c, typ, equipEvSuspects[typ], a[0], a[1])
		}
	}
}

// equipEvBilan : tables parc precision / rappel avec denominateurs — la matiere du rapport.
func equipEvBilan(t *testing.T, precAgg map[int][2]int, rappelAgg map[string]map[int][2]int) {
	t.Helper()
	t.Logf("== BILAN PARC — PRECISION par type (apparies w8 / total tetes) ==")
	types := make([]int, 0, len(precAgg))
	for typ := range precAgg {
		types = append(types, typ)
	}
	sort.Ints(types)
	for _, typ := range types {
		a := precAgg[typ]
		t.Logf("  type %3d %-36s : %d/%d", typ, equipEvSuspects[typ], a[0], a[1])
	}
	t.Logf("== BILAN PARC — RAPPEL par canal x type (usages rappeles / usages mesures) ==")
	canaux := make([]string, 0, len(rappelAgg))
	for c := range rappelAgg {
		canaux = append(canaux, c)
	}
	sort.Strings(canaux)
	for _, c := range canaux {
		for _, typ := range types {
			a := rappelAgg[c][typ]
			if a[1] == 0 {
				continue
			}
			t.Logf("  %-22s x type %3d %-36s : %d/%d", c, typ, equipEvSuspects[typ], a[0], a[1])
		}
	}
}
