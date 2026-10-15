package replay

import "testing"

// TestAucunePoseSansSonObjetSurLesBuilds : garde-rail « aucune pose sans position quand l objet est
// lisible ». Sur les builds figes (entrees, entites et equipement relus), chaque pose de faille publiee
// est situee par l objet que la pose cree : sur ces films il est lisible a chaque pose (123 sur 123 au
// parc), donc un repli du porteur ou une pose sans position y dit que la lecture de la faille a casse.
func TestAucunePoseSansSonObjetSurLesBuilds(t *testing.T) {
	poses := 0
	for _, b := range goldenBuilds() {
		g, entry := chargerGoldenBuild(t, b)
		doc := assemblerGoldenBuild(t, b, g, entry)
		for _, p := range doc.RiftPoses {
			poses++
			if p.Src != sourceFailleObjet || p.X == nil {
				t.Errorf("%s : pose %+v sans l objet qu elle cree", b.Short8, p)
			}
		}
	}
	if poses == 0 {
		t.Fatal("aucune pose de faille sur les builds : la garde ne garde rien")
	}
}
