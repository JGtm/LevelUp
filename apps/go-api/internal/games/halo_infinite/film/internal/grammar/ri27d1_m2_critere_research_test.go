//go:build research

package grammar

// ri27d1_m2_critere_research_test.go — 2.7.d1 (`.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// commande I-critere ; decision U-0 de l utilisateur du 2026-10-08).
//
// LE CRITERE ECRIT DE LA BASCULE (« atterrissage bit-exact des 591 records ti=35 bornes au-dessus de
// 50 % »), rejoue dans le cadre de production de TestKF7EFullStateLoop (WalkKeyframeFullState, largeurs
// de la carte lues dans le film, SimStateComplet pose comme le fait TestKF7E), AVEC et SANS les bouchons
// de kf35ApplyStubs, pour REF (la marche de production a la tete) et (d+e) par les bascules du profil.
// Il publie aussi le nombre de records dont la marche a consomme une largeur de bouchon
// (lecture.LargeurBouchon) et les composants concernes.
//
//	KF35_ROOT=<film_chunks> go test -tags=research -count=1 -run '^TestRI27d1M2Critere$' -v ./...grammar/

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

func TestRI27d1M2Critere(t *testing.T) {
	films := kf35Films(t)
	defer poserBasculeDInstrument(func(g *GrammaireBalayage) { g.SimStateComplet = true })()
	type cas struct {
		nom       string
		scope, i0 bool
	}
	lesCas := []cas{
		{nom: "REF"},
		{nom: "(d+e) bascules du profil", scope: true, i0: true},
	}
	tot := map[string][3]int{} // exactes, chainees, bornees
	for _, f := range films {
		_, restorePrec := kf35bInstallPrecision(t, f.Name)
		for _, bouchons := range []bool{true, false} {
			var restoreStubs func()
			var stubbed []string
			if bouchons {
				stubbed, restoreStubs = kf35ApplyStubs(f, kf7dVariant)
			}
			for _, c := range lesCas {
				prev := poserBasculeDInstrument(func(g *GrammaireBalayage) {
					g.PorteeBaseline, g.GrammaireEcrivainI0 = c.scope, c.i0
				})
				tal := newKF7ETally()
				avecBouchon, parComp := 0, map[string]int{}
				for _, pay := range f.Pays {
					for _, b := range kf35BoundedRecs(pay) {
						tal.bounded++
						tr := WalkKeyframeFullState(pay, b.Rec.Bit, f.Reg, contexteDInstrument())
						vu := false
						for _, cp := range tr.Comps {
							if cp.Prov == lecture.LargeurBouchon {
								vu = true
								parComp[fmt.Sprintf("i%d %s", cp.Index, cp.Name)]++
							}
						}
						if vu {
							avecBouchon++
						}
						kf7eWalkOne(f, pay, b, kf7eCase{}, &tal)
					}
				}
				prev()
				cle := fmt.Sprintf("%s | bouchons=%v", c.nom, bouchons)
				x := tot[cle]
				tot[cle] = [3]int{x[0] + tal.exact, x[1] + tal.chained, x[2] + tal.bounded}
				t.Logf("CRITERE\t%s\t%s\tbouchons=%v\texactes=%d\tchainees=%d\tdesync=%d\tbornees=%d\ttaux=%.2f\tavec_bouchon=%d\tbouchons_poses=%v\tpar_composant=%v",
					f.Name, c.nom, bouchons, tal.exact, tal.chained, tal.desync, tal.bounded, tal.rate(), avecBouchon, stubbed, parComp)
				kf7eLogBreaks(t, tal.breaks, 4)
			}
			if restoreStubs != nil {
				restoreStubs()
			}
		}
		restorePrec()
	}
	for cle, x := range tot {
		t.Logf("TOTAL\t%s\texactes=%d\tchainees=%d\tbornees=%d\ttaux=%.2f", cle, x[0], x[1], x[2],
			100*float64(x[0]+x[1])/float64(x[2]))
	}
}
