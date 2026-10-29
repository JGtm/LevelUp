//go:build research

package replay

// instruments_partages_powerup_research_test.go — mesure du socle de powerup_socle_oracle_test.go, utilisee par des instruments research (J12.7 lint : inutilises dans le build par defaut). Deplacement pur.

import "testing"

// psSocleMesure rend la position ET l altitude du socle mesurees par la phase 1, ou saute
// l etape appelante.
func psSocleMesure(t *testing.T) (psPoint, float32) {
	t.Helper()
	dir := psArtDir(t)
	doc, ok := psLoadDoc(t, dir, "01e1f945")
	if !ok {
		t.Skipf("artefact 01e1f945 absent de %s : le socle n est pas mesure", dir)
	}
	r, ok := psSocleParRemontee(doc)
	if !ok {
		t.Skipf("la remontee ne rend pas de socle (%d episodes retenus, rayon %.3f m)",
			r.Garde, r.R)
	}
	return r.C, r.Z
}
