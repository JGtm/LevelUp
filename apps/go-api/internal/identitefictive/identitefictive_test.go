package identitefictive

import "testing"

// TestXuidDeterministeEtDansLaPlage : meme entree, meme sortie, toujours dans la plage ; et un
// vecteur fige, pour qu'un changement de graine ou de formule se voie.
func TestXuidDeterministeEtDansLaPlage(t *testing.T) {
	vus := map[uint64]uint64{}
	for i := range uint64(5000) {
		reel := PlageBas + i*7919
		f := Xuid(reel)
		if f != Xuid(reel) {
			t.Fatalf("%d : sortie differente pour la meme entree", reel)
		}
		if !DansLaPlage(f) {
			t.Fatalf("%d -> %d : hors de la plage [%#x, %#x[", reel, f, PlageBas, PlageHaut)
		}
		if autre, dup := vus[f]; dup {
			t.Fatalf("%d et %d rendent le meme fictif %d", reel, autre, f)
		}
		vus[f] = reel
	}
}

func TestXuidVecteurFige(t *testing.T) {
	const reel, attendu = uint64(2533274790395904), uint64(2550913680076637)
	if got := Xuid(reel); got != attendu {
		t.Fatalf("Xuid(%d) = %d", reel, got)
	}
}
