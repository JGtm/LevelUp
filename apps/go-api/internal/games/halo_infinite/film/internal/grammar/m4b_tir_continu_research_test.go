//go:build research

package grammar

// m4b_tir_continu_research_test.go — LOT M4b (campagne « retours rejeu », 2026-09-24) : LE TIR
// CONTINU TEL QUE LA MARCHE DE PRODUCTION LE REND. Mesure seule.
//
// Il n imite pas la production : il APPELLE [ScanMarcheDesTrames], la marche du frame-processeur
// que la cuisson emploie, et publie ce qu elle rend — les verdicts de vue C (lus, trous et leurs
// causes), les rafales par index de controle, et le gate G1 du lot sur le pilote designe :
//
//	pour chaque frag f : une rafale du pilote recouvre-t-elle [f - 2 s, f] ?
//	temoin -60 s        : une rafale recouvre-t-elle [f - 62 s, f - 60 s] ?
//	hors montures       : rafales du pilote hors des episodes S3_EPISODES (+ M4B_MONTURES)
//
// Rejouable (voie film, un film, racine de donnees TEMPORAIRE) :
//
//	MOUV511_FILM=<copie>/film_chunks/81c02726 MOUV511_BORNES=<copie>/map_quant_bounds.json \
//	MOUV511_CARTE=isolation S3_ORIGINE_US=451125221 S3_INDEX=2 \
//	S3_EPISODES=311-1088,2064-2869 M4B_MONTURES=2950-3120 S3_FRAGS=395,623,650,2159,2228,2402 \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestM4bTirContinu$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// m4bRecouvre dit si la part LUE d une rafale (hors de ses trous) recouvre les trames [a, b] :
// c est « une entree lue qui tire » dans la fenetre.
func m4bRecouvre(cad s3Cadre, r types.ContinuousFireBurst, a, b int) bool {
	debut := cad.trame(r.StartUS)
	for _, h := range r.Holes {
		if debut <= b && cad.trame(h.StartUS) >= a {
			return true
		}
		debut = cad.trame(h.EndUS)
	}
	return debut <= b && cad.trame(r.EndUS) >= a
}

// TestM4bTirContinu publie ce que la marche de production rend (en-tete du fichier).
func TestM4bTirContinu(t *testing.T) {
	cad := s3LireCadre(t)
	for _, s := range strings.Split(os.Getenv("M4B_MONTURES"), ",") {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			cad.episodes = append(cad.episodes, [2]int{a, b})
		}
	}
	tc := t516Cadre(t)
	m, err := ScanMarcheDesTrames(tc.fc)
	if err != nil {
		t.Fatalf("ScanMarcheDesTrames : %v", err)
	}
	st := m.ContinuousFireStats
	t.Logf("== VUE C : %d paquets · atteinte %d · FERMEE %d (%.1f %%) · trous %d en %d suites",
		st.Packets, st.Reached, st.Closed, 100*float64(st.Closed)/float64(max(st.Packets, 1)),
		st.Holes, st.HoleRuns)
	t.Logf("   trous : liste non localisee %d · vue B ouverte %d · debordement %d · kinds 1/2 %d · "+
		"bloc 0xbc %d · plafond %d · lue sans fermer %d", st.Unlocated, st.OpenViewB, st.StopOverflow,
		st.StopKind, st.StopBlockBC, st.StopCap, st.NotClosing)
	t.Logf("   entrees %d · avec bloc %d · qui tirent %d · rafales %d (touchees par un trou %d, trous interieurs %d) · "+
		"tir tenu tu %.1f s", st.Entries, st.WithAction, st.Firing, st.Bursts, st.BurstsWithHole, st.InnerHoles,
		float64(st.HeldHoleMS)/1000)
	ms := m.MovementStateStats
	t.Logf("== ETATS DE MOUVEMENT : records ti=35 %d · desyncs %d · lectures %d · paquets %d · "+
		"evenements %d (localises %d, non %d)", ms.Records, ms.Desyncs, ms.Read, ms.Packets,
		ms.EventPackets, ms.EventPacketsLocated, ms.EventPacketsUnlocated)
	m4bPublierParIndex(t, m.ContinuousFire, cad)
	m4bPublierGate(t, m.ContinuousFire, cad)
}

// m4bPublierParIndex publie, par index de controle et par bit, les rafales et leur duree.
func m4bPublierParIndex(t *testing.T, rs []types.ContinuousFireBurst, cad s3Cadre) {
	t.Helper()
	type agg struct {
		n, trou int
		ms      uint64
		armes   map[int]int
	}
	par := map[string]*agg{}
	for _, r := range rs {
		k := fmt.Sprintf("index %2d main %d %s %d", r.FilmIndex, r.Hand,
			map[bool]string{false: "gachette", true: "barillet"}[r.Barrel], r.Input)
		a := par[k]
		if a == nil {
			a = &agg{armes: map[int]int{}}
			par[k] = a
		}
		a.n++
		a.ms += (r.EndUS - r.StartUS) / 1000
		a.armes[r.Weapon]++
		if r.StartBound == types.ContinuousFireBoundHole || r.EndBound == types.ContinuousFireBoundHole {
			a.trou++
		}
	}
	cles := make([]string, 0, len(par))
	for k := range par {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	t.Logf("== RAFALES par bit (%d) :", len(rs))
	for _, k := range cles {
		a := par[k]
		t.Logf("   %s : %4d rafales · %7.1f s · coupees %d · armes %v", k, a.n, float64(a.ms)/1000,
			a.trou, a.armes)
	}
	tous := os.Getenv("M4B_TOUS") != ""
	for _, r := range rs {
		if r.FilmIndex != cad.index && !tous {
			continue
		}
		t.Logf("   index %d : t %d..%d (%.2f s) %s..%s entrees %d arme %d main %d %s %d trous %d",
			r.FilmIndex, cad.trame(r.StartUS), cad.trame(r.EndUS), float64(r.EndUS-r.StartUS)/1e6, r.StartBound,
			r.EndBound, r.Entries, r.Weapon, r.Hand,
			map[bool]string{false: "g", true: "b"}[r.Barrel], r.Input, len(r.Holes))
	}
}

// m4bPublierGate publie le gate G1 du lot.
func m4bPublierGate(t *testing.T, rs []types.ContinuousFireBurst, cad s3Cadre) {
	t.Helper()
	var par, temoin []string
	ok := 0
	for _, f := range cad.frags {
		n, nt := 0, 0
		for _, r := range rs {
			if r.FilmIndex != cad.index {
				continue
			}
			if m4bRecouvre(cad, r, f-20, f) {
				n++
			}
			if m4bRecouvre(cad, r, f-620, f-600) {
				nt++
			}
		}
		ok += s3B(n > 0)
		par = append(par, fmt.Sprintf("%d:%d", f, n))
		temoin = append(temoin, fmt.Sprintf("%d:%d", f-600, nt))
	}
	hors := 0
	for _, r := range rs {
		if r.FilmIndex != cad.index {
			continue
		}
		dedans := false
		for _, e := range cad.episodes {
			if cad.trame(r.StartUS) <= e[1] && cad.trame(r.EndUS) >= e[0] {
				dedans = true
			}
		}
		hors += s3B(!dedans)
	}
	t.Logf("== GATE G1 index %d : frags recouverts %d/%d %v · temoin -60 s %v · rafales hors "+
		"montures %d", cad.index, ok, len(cad.frags), par, temoin, hors)
}
