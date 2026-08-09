package himap

import (
	"math"
	"testing"
)

// POURQUOI LIVE FIRE NE PLACE AUCUNE DE SES 28 ANCRES SUR UN SOL.
//
// Le balayage du 2026-08-09 donne 0/14 sur `live_fire_sgh_interlock` ET sur sa variante
// classee, quand les trois autres cartes `sgh` (recharge/blueprint, prism/crystalcaves,
// streets/streets) rendent 13/14, 8/14 et 10/14. Le defaut est donc propre a cette carte, pas
// a la famille.
//
// Ce diagnostic compare, pour une carte fautive et une carte temoin, les deux reperes qui
// doivent coincider : l'emprise declaree du sbsp et l'emprise des ancres. S'ils sont
// disjoints, la question n'est plus « pourquoi le sol manque » mais « pourquoi les ancres sont
// ailleurs », ce qui est un tout autre probleme.
func TestDiagnosticLiveFire(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	for _, module := range []string{"live_fire_sgh_interlock", "recharge_sgh_blueprint", "cliffhanger_ridgeline"} {
		chemin, ok := chercheModule(racine, module)
		if !ok {
			t.Logf("%-26s module introuvable", module)
			continue
		}
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%-26s lecture KO : %v", module, err)
			continue
		}

		lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
		hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
		n := 0
		for _, e := range ancresDuModule(t, module) {
			for _, o := range e.Objectives {
				n++
				p := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
				for k := 0; k < 3; k++ {
					lo[k] = math.Min(lo[k], p[k])
					hi[k] = math.Max(hi[k], p[k])
				}
			}
		}
		t.Logf("%s — %d bsp · %d ancres", module, len(bsps), n)
		t.Logf("   ancres  X [%+7.1f ; %+7.1f]  Y [%+7.1f ; %+7.1f]  Z [%+7.1f ; %+7.1f]",
			lo[0], hi[0], lo[1], hi[1], lo[2], hi[2])
		for i, b := range bsps {
			t.Logf("   bsp %d  X [%+7.1f ; %+7.1f]  Y [%+7.1f ; %+7.1f]  Z [%+7.1f ; %+7.1f]  %d instances",
				i, b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				b.Bounds.Min[2], b.Bounds.Max[2], len(b.Instances))
		}
	}
}
