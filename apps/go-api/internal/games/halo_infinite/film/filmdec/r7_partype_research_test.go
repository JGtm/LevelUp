package filmdec

// r7_partype_research_test.go — lot R7 : L'ORACLE DE TRAME APPLIQUE TYPE PAR TYPE.
//
// POURQUOI. Un oracle global peut rester bon alors qu'UNE grammaire est fausse : les listes
// ou le type fautif n'apparait pas portent la moyenne. Ce test isole chaque type : il compare
// la profondeur de trame des listes QUI CONTIENNENT le type a celle des listes qui ne le
// contiennent PAS, sur le meme film et le meme cadrage.
//
// LECTURE : une grammaire juste laisse les deux profondeurs du meme ordre. Une grammaire
// fausse ecroule la profondeur des listes qui traversent le type — et, symptome jumeau, fait
// exploser le nombre d'occurrences du type (la marche part en vrille et « relit » le meme
// type indefiniment).
//
// SEUIL ECRIT AVANT LA MESURE : un type est declare SUSPECT si la profondeur des listes qui
// le contiennent tombe sous 50 % de celle des listes qui ne le contiennent pas, avec au moins
// 30 trames de chaque cote.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// r7TypesDemandes rend le filtre R7_TYPES ("104,42,43") ou nil pour « tous les types vus ».
func r7TypesDemandes() map[int]bool {
	v := os.Getenv("R7_TYPES")
	if v == "" {
		return nil
	}
	out := map[int]bool{}
	for _, s := range strings.Split(v, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			out[n] = true
		}
	}
	return out
}

// TestR7ParType isole la responsabilite de chaque grammaire dans l'oracle de trame.
func TestR7ParType(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	release := LockProcessDecode()
	defer release()
	filtre := r7TypesDemandes()
	type mesure struct{ avec, sans r7TrameStat }
	agg := map[int]*mesure{}
	occurrences := map[int]int{}
	for _, id := range ids {
		reg, chunks, err := r7Chargements(filepath.Join(root, id))
		if err != nil || len(chunks) == 0 {
			t.Logf("film %s : illisible (%v) — ignore", id, err)
			continue
		}
		cfg := DefaultFrameConfig()
		cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
		ctx := cartes[id]
		// Quels types apparaissent dans ce film ? (pour ne mesurer que ceux-la)
		presents := map[int]bool{}
		for _, data := range chunks {
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 {
					continue
				}
				evs, _, _, _ := r7Marche(pay, ctx)
				for _, e := range evs {
					presents[e.Typ] = true
					occurrences[e.Typ]++
				}
			}
		}
		var types []int
		for typ := range presents {
			if filtre == nil || filtre[typ] {
				types = append(types, typ)
			}
		}
		sort.Ints(types)
		for _, typ := range types {
			if agg[typ] == nil {
				agg[typ] = &mesure{}
			}
			contient := r7ContientType(typ)
			avec, _ := r7OracleFilm(reg, chunks, ctx, cfg, contient, 0)
			sans, _ := r7OracleFilm(reg, chunks, ctx, cfg,
				func(evs []r7Ev) bool { return !contient(evs) }, 0)
			agg[typ].avec.cumule(avec)
			agg[typ].sans.cumule(sans)
		}
	}
	var types []int
	for typ := range agg {
		types = append(types, typ)
	}
	sort.Slice(types, func(i, j int) bool { return occurrences[types[i]] > occurrences[types[j]] })
	t.Logf("")
	t.Logf("%-4s %-38s %8s %10s %10s %8s", "type", "nom", "occurr.", "prof AVEC", "prof SANS", "verdict")
	for _, typ := range types {
		m := agg[typ]
		a, s := m.avec.profondeur(), m.sans.profondeur()
		verdict := "ok"
		switch {
		case m.avec.paquets < 30 || m.sans.paquets < 30:
			verdict = "trop peu"
		case a < 0.5*s:
			verdict = "SUSPECT"
		}
		t.Logf("%-4d %-38s %8d %10.3f %10.3f %8s  (n avec=%d, sans=%d)",
			typ, r7Noms[typ], occurrences[typ], a, s, verdict, m.avec.paquets, m.sans.paquets)
	}
}

// r7Tranches : les tranches de longueur de liste examinees par TestR7ParLongueur.
var r7Tranches = [][2]int{{1, 1}, {2, 3}, {4, 7}, {8, 15}, {16, 31}, {32, 1 << 30}}

// TestR7ParLongueur juge la marche PAR LONGUEUR DE LISTE. Les listes tres longues sont le
// symptome classique d'une marche partie en vrille : si elles sont reelles, leur trame doit
// aller aussi loin que celle des listes courtes ; si elles sont un artefact, leur trame
// s'effondre. SEUIL ECRIT D'AVANCE : une tranche est SUSPECTE si sa profondeur tombe sous
// 50 % de celle de la tranche « une seule entree », avec au moins 30 trames.
func TestR7ParLongueur(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	release := LockProcessDecode()
	defer release()
	stats := make([]r7TrameStat, len(r7Tranches))
	for _, id := range ids {
		reg, chunks, err := r7Chargements(filepath.Join(root, id))
		if err != nil || len(chunks) == 0 {
			continue
		}
		cfg := DefaultFrameConfig()
		cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
		ctx := cartes[id]
		for i, tr := range r7Tranches {
			lo, hi := tr[0], tr[1]
			st, _ := r7OracleFilm(reg, chunks, ctx, cfg,
				func(evs []r7Ev) bool { return len(evs) >= lo && len(evs) <= hi }, 0)
			stats[i].cumule(st)
		}
	}
	ref := stats[0].profondeur()
	t.Logf("%-12s %8s %12s %10s", "longueur", "trames", "profondeur", "verdict")
	for i, tr := range r7Tranches {
		verdict := "ok"
		switch {
		case stats[i].paquets < 30:
			verdict = "trop peu"
		case stats[i].profondeur() < 0.5*ref:
			verdict = "SUSPECT"
		}
		t.Logf("%2d..%-9d %8d %12.3f %10s", tr[0], tr[1], stats[i].paquets,
			stats[i].profondeur(), verdict)
	}
}
