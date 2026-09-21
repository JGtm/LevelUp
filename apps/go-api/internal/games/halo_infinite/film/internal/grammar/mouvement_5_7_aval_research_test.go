//go:build research

package grammar

// mouvement_5_7_aval_research_test.go — CE QUI SUIT `i55` DANS LE RECORD (lot 5.7).
//
// # LA QUESTION QUE CE FICHIER TRANCHE
//
// `i55` est le 56e des 64 composants du bipede. Si sa charge n est pas lue, les bits non lus
// decalent `i56` a `i63` et la queue du record est perdue. Le porter DOIT donc faire monter le
// nombre de records qui lisent un composant APRES `i55` — et si cela ne monte pas, c est que le
// manque ne coutait rien, ou que la grammaire posee est fausse.
//
// L instrument ne suppose rien : il compte, sur les records qui DECLARENT `i55`, ceux qui
// portent au moins un composant d index superieur, et il rend la distribution de ces index.
//
// Rejouable : memes variables que `TestMouvement57Posture`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m57AvalStat : ce que la passe d aval compte.
type m57AvalStat struct {
	ti35        int
	avecI55     int
	i55Porte    int
	avecAval    int
	indexAval   map[int]int
	indexNom    map[int]int
	dernierI55  int
	compsApres  int
	desync      int
	fautifs     map[int]int
	etalonI21   int
	etalonTotal int
}

// TestMouvement57Aval compte ce qui suit `i55` dans les records du bipede.
func TestMouvement57Aval(t *testing.T) {
	dir := os.Getenv("MOUV57_FILM")
	if dir == "" {
		t.Skip("MOUV57_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m57Contexte(t, film)
	reg, errR := fc.Registry()
	if errR != nil {
		t.Fatalf("registre : %v", errR)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	st := &m57AvalStat{indexAval: map[int]int{}, indexNom: map[int]int{}, fautifs: map[int]int{}}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m57AvalPaquet(pk, data, w, cfg, st)
		}
	}
	m57AvalRendre(t, st)
}

// m57AvalPaquet marche UN paquet et ventile ses records de bipede.
func m57AvalPaquet(pk FilmPacket, data []byte, w *World, cfg FrameConfig, st *m57AvalStat) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := 2
	if _, present := PacketHeadEventType(pay); present {
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			return
		}
	}
	recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		m57AvalRecord(r, st)
	}
}

// m57AvalRecord ventile UN record : `i55` present ? porte ? suivi ?
func m57AvalRecord(r FrameRecord, st *m57AvalStat) {
	st.ti35++
	st.etalonTotal++
	if r.DesyncAt >= 0 {
		st.desync++
		st.fautifs[r.DesyncAt]++
	}
	// LA CLE EST LE NOM, PAS L INDEX : le registre du film numerote les composants de
	// l archetype, et rien ne garantit que `biped-posture-physics-component` y porte l index 55.
	// Une premiere version de cette sonde comptait l index 55 et rendait 52 au lieu de 3 784.
	var idxI55 = -1
	var aval bool
	for _, c := range r.Trace.Comps {
		if c.Name == "unit-desired-aiming-vector-component" {
			st.etalonI21++
		}
		if c.Name != "biped-posture-physics-component" {
			continue
		}
		idxI55 = c.Index
		st.avecI55++
		st.indexNom[c.Index]++
		if c.Ported {
			st.i55Porte++
		}
	}
	if idxI55 < 0 {
		return
	}
	for _, c := range r.Trace.Comps {
		if c.Index > idxI55 {
			aval = true
			st.indexAval[c.Index]++
			st.compsApres++
		}
	}
	if aval {
		st.avecAval++
	} else {
		st.dernierI55++
	}
}

// m57AvalRendre publie le verdict.
func m57AvalRendre(t *testing.T, st *m57AvalStat) {
	t.Helper()
	t.Logf("AVAL D `i55` : %d records ti=35 (%d desynchronises, fautif %s) · etalon i21 %.1f %%",
		st.ti35, st.desync, m533cTable(st.fautifs),
		m533bPart(st.etalonI21, st.etalonTotal))
	t.Logf("  `i55` lu sur %d records, dont %d portes · %d (%.1f %%) portent AU MOINS UN "+
		"composant d index > 55 · %d (%.1f %%) s arretent a `i55`",
		st.avecI55, st.i55Porte, st.avecAval, m533bPart(st.avecAval, st.avecI55),
		st.dernierI55, m533bPart(st.dernierI55, st.avecI55))
	keys := make([]int, 0, len(st.indexAval))
	for k := range st.indexAval {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("i%d : %d", k, st.indexAval[k]))
	}
	t.Logf("  COMPOSANTS LUS APRES `i55` (%d au total) : %s", st.compsApres,
		strings.Join(parts, " · "))
	t.Logf("  INDEX DE REGISTRE PORTANT LE NOM `biped-posture-physics-component` : %s",
		m533cTable(st.indexNom))
}
