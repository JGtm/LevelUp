//go:build research

package grammar

// mouvement_5_7_phase_research_test.go — D OU VIENNENT LES LECTURES D `i55` (lot 5.7, sonde
// jetable de diagnostic). L instrument d aval compte `i55` sur 52 records de la boucle de
// composants, alors que la porte de publication tire 3 784 fois : les deux nombres ne peuvent
// pas decrire le meme chemin, et cette sonde dit lequel tire.

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestMouvement57Phase(t *testing.T) {
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
	phase := "liaison"
	compte := map[string]int{}
	parNom := map[string]int{}
	cfg := fc.CadreDeBalayage()
	cfg.Obs = &Observation{
		EtatMouvementHook: func(comp EtatMouvementComposant, _ uint32, _ []uint64) {
			compte[phase+":"+comp.String()]++
		},
	}
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		phase = "liaison"
		m533bLierMonde(w, data, pks)
		phase = "keyframe"
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			_ = WalkKeyframeWorld(pk.Payload(data))
		}
		phase = "delta"
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				phase = "localisation"
				debut = marchLocateStrict(pay, w, cfg)
				phase = "delta"
				if debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			for _, r := range recs {
				for _, c := range r.Trace.Comps {
					parNom[c.Name]++
					if r.TypeIndex == BipedTypeIndex {
						parNom["ti35:"+c.Name]++
					}
				}
			}
		}
	}
	for _, n := range []string{"unit-crouch-component", "biped-posture-physics-component",
		"biped-slide-component", "object-translational-velocity-dynamic-precision-component",
		"unit-control-component", "biped-mobility-action-component"} {
		t.Logf("  %-56s : COMPS tous archetypes %6d · dont ti=35 %6d", n, parNom[n],
			parNom["ti35:"+n])
	}
	keys := make([]string, 0, len(compte))
	for k := range compte {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("  HOOK %-70s : %6d", k, compte[k])
	}
}
