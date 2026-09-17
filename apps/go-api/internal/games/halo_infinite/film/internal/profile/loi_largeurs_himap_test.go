//go:build cgo

package profile_test

// loi_largeurs_himap_test.go — LA CONFRONTATION DES DEUX COPIES DE LA LOI, DERRIERE LE TAG `cgo`.
//
// Sorti de loi_largeurs_test.go le 2026-09-17 (CI rouge sur la fusion du lot 3.4) : ce test importe
// `himap`, qui tire `himodule` puis `ooz` — un paquet CGO sans aucun fichier compilable sous
// CGO_ENABLED=0. Or le job « Go Build + Test » de la CI vet `film/internal/profile/...` avec
// CGO_ENABLED=0 (ci.yml, etape `go vet`) : le paquet de test entier mourait sur « build constraints
// exclude all Go files in internal/ooz », et avec lui les six autres tests de la loi, qui n ont
// besoin de rien. Le tag `cgo` ne desarme rien : le job Coverage (CGO_ENABLED=1) et tout poste local
// jouent ce test ; seul le vet sans CGO l ignore, comme il ignore deja himap lui-meme
// (memoire du projet : « vet himap/himodule = CGO obligatoire »).

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/himap"
)

// TestLoiHimapEtLoiDuProfilSAccordent — LE GARDE-RAIL DES DEUX COPIES.
//
// `himap.Bounds.AxisWidths` (le PRODUCTEUR hors ligne, qui lit les `.module` en `float64`) et
// `profile.LargeursAxeDuNiveau` (le CONSOMMATEUR a l execution, en `float32` comme le moteur)
// portent la meme loi et ne peuvent pas la partager : `profile` est une feuille. Elles sont
// donc confrontees sur les 79 cartes commises ET sur les deux gardes.
func TestLoiHimapEtLoiDuProfilSAccordent(t *testing.T) {
	cat := chargerCatalogueDesCartes(t)
	for nom, e := range cat.Maps {
		var b himap.Bounds
		for ax := 0; ax < 3; ax++ {
			b.Min[ax], b.Max[ax] = float64(e.Min[ax]), float64(e.Max[ax])
		}
		h := b.AxisWidths()
		p := profile.LargeursAxeDuNiveau(bornesDe(e), profile.NiveauPositionDObjet)
		for ax := 0; ax < 3; ax++ {
			if uint(h[ax]) != p[ax] {
				t.Errorf("%s axe %d : himap %d, profile %d", nom, ax, h[ax], p[ax])
			}
		}
	}
	// LES DEUX GARDES, sur la meme carte synthetique que le test ci-dessus.
	for _, etendue := range []float64{4194304.0 / 60 * 0.99, 4194304.0 / 60 * 4, math.Exp2(30)} {
		b := himap.Bounds{Max: [3]float64{etendue, etendue, etendue}}
		if got := b.AxisWidths(); got != [3]int{22, 22, 22} {
			t.Errorf("himap, etendue %.0f : %v, attendu 22/22/22 (garde 2^22)", etendue, got)
		}
	}
}
