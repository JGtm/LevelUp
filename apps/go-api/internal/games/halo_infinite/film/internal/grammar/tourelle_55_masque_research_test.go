//go:build research

package grammar

// tourelle_55_masque_research_test.go — LOT 5.5, POINT 2 : LE FILM DECLARE-T-IL `i41`/`i42` ?
//
// # LA QUESTION, ET POURQUOI LE MASQUE Y REPOND SANS PORTER UNE GRAMMAIRE
//
// L ecrivain (point 1) a rendu la grammaire de `ti=40 i41`/`i42` bit-exacte. Avant de la porter,
// il faut savoir si le film DECLARE ces composants : un composant dont aucun record ne pose le
// bit de masque ne se lira jamais, et le porter serait du code mort.
//
// LE MASQUE DE COMPOSANTS VOYAGE EN TETE DU RECORD, AVANT toute charge : il se lit donc sans
// porter aucun deserialiseur, et sans jamais desyncer sur un composant non porte. Le depot le
// capture deja (`BipedPosition.MaskBits` / `MaskOver`, sous `CaptureDirs`) — cette mesure ne fait
// que le VENTILER par index, sur la bande `ti=40`.
//
// # LES LARGEURS D AXE DE LA CARTE SONT INSTALLEES, ET CE N EST PAS FACULTATIF
//
// Piege documente au lot 5.3 (passation, point 6) et paye deux fois : un instrument qui marche
// des records SANS l entree de catalogue de la carte lit `i0` a 13/13/14 bits et tout se decale.
// Le contexte passe donc par `NewFilmContextForMap` + `PoserLargeursObjetDuMondeDepuisDecoupage`,
// exactement comme la cuisson (meme geste que `avantContexteDe`, lot 5.4).
//
// # CE QUE CETTE MESURE NE PEUT PAS DIRE, ET ELLE LE DIT
//
// `walkDeltaBipedPayload` est un CHERCHEUR D ANCRES : il ne retient qu un record dont `i0` est
// ABSOLU et de la region attendue. Un record `ti=40` qui porterait `i41`/`i42` SANS `i0` est donc
// invisible ici. Un ZERO rendu par cette passe est un negatif SUR CETTE POPULATION — c est dit
// avec son compte, jamais generalise.
//
// # REGIME
//
//	TOUR55_FILM=<abs>/data/cache/film_chunks/4f77afc1 TOUR55_CARTE="Flood Gulch" \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	    -run '^TestTourelle55Masque$' ./internal/games/halo_infinite/film/internal/grammar/
//
// UN SEUL FILM PAR INVOCATION, aucune base DuckDB, aucun artefact ecrit.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// tourelle55Index — les index dont la ventilation est commentee dans le rapport, parce que la
// question porte sur eux.
var tourelle55Index = map[int]string{
	2:  "i2  object-forward-and-up (l avant du chassis, lot 5.4)",
	21: "i21 unit-desired-aiming-vector (la visee — DU VEHICULE ici, pas de l occupant)",
	30: "i30 vehicle-auto-turret-triggers (le bloquant d image-cle de ti=40)",
	31: "i31 vehicle-auto-turret-aiming-vector",
	40: "i40 vehicle-equipment-turret-parent",
	41: "i41 vehicle-seats-override-PITCH  <-- CANDIDAT",
	42: "i42 vehicle-seats-override-YAW    <-- CANDIDAT",
}

func TestTourelle55Masque(t *testing.T) {
	dir, carte := os.Getenv("TOUR55_FILM"), os.Getenv("TOUR55_CARTE")
	if dir == "" || carte == "" {
		t.Skip("instrument de mesure : TOUR55_FILM et TOUR55_CARTE requis")
	}
	fc, plage := avantContexteDe(t, dir, carte)
	if fc == nil {
		t.Fatalf("contexte illisible")
	}
	if restore, err := InstallFilmFormatMPP(fc); err == nil {
		defer restore()
	}
	kf := ScanWorldObjectKeyframes(fc, VehicleTypeIndex)
	if len(kf.Band) == 0 {
		t.Fatalf("aucune bande ti=40 dans les images-cle")
	}
	opt := DefaultScanFilmOptions()
	opt.RequireTag1 = false
	opt.CaptureDirs = true
	opt.DynPrecOrientation = true
	if l := fc.ImposedLayout(); l != nil {
		opt.Layout = l
	}
	opt.WorldRange = plage
	pos, err := ScanBipedPositionsForBand(fc, NewSlotBand(kf.Band), opt)
	if err != nil {
		t.Fatalf("bande ti=40 illisible : %v", err)
	}
	parIndex, over, slotsParIndex := tourelle55Ventiler(pos)
	t.Logf("%s (carte %s) : %d records ti=40 a ancre i0 absolue, %d a masque tronque (index >= 64)",
		filepath.Base(dir), carte, len(pos), over)
	t.Log("--- VENTILATION DU MASQUE, index par index (part des records) ---")
	idx := make([]int, 0, len(parIndex))
	for i := range parIndex {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for _, i := range idx {
		n := parIndex[i]
		pc := 100 * float64(n) / float64(len(pos))
		t.Logf("  i%-2d  %7d  %6.2f %%  %d slot(s)   %s",
			i, n, pc, len(slotsParIndex[i]), tourelle55Index[i])
	}
	t.Log("--- REPONSE AUX DEUX CANDIDATS ---")
	for _, i := range []int{41, 42} {
		if parIndex[i] == 0 {
			t.Logf("  i%d : ZERO record sur %d — NEGATIF sur cette population", i, len(pos))
			continue
		}
		t.Logf("  i%d : %d records (%.2f %%), %d slots distincts — DECLARE par le film",
			i, parIndex[i], 100*float64(parIndex[i])/float64(len(pos)), len(slotsParIndex[i]))
	}
}

// tourelle55Ventiler compte, index par index, les records qui DECLARENT ce composant, et retient
// les slots distincts — un composant declare par un seul slot n est pas la meme chose qu un
// composant declare par tout le parc.
func tourelle55Ventiler(pos []BipedPosition) (map[int]int, int, map[int]map[uint32]bool) {
	parIndex := map[int]int{}
	slots := map[int]map[uint32]bool{}
	over := 0
	for _, p := range pos {
		if p.MaskOver {
			over++
		}
		for i := 0; i < 64; i++ {
			if p.MaskBits&(1<<uint(i)) == 0 {
				continue
			}
			parIndex[i]++
			if slots[i] == nil {
				slots[i] = map[uint32]bool{}
			}
			slots[i][p.Slot] = true
		}
	}
	return parIndex, over, slots
}
