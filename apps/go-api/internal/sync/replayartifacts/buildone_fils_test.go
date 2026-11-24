package replayartifacts

import (
	"context"
	"testing"
)

// TestAvecFils_LaValeurAtteintChaqueRequete : le nombre de fils de l'installation est pose sur chaque requete de
// construction, quelle que soit la valeur que l'appelant y avait mise, avant qu'elle parte vers l'enfant.
func TestAvecFils_LaValeurAtteintChaqueRequete(t *testing.T) {
	var vus []int
	build := avecFils(func(_ context.Context, req BuildOneRequest) (BuildOneResult, error) {
		vus = append(vus, req.DecoderThreads)
		return BuildOneResult{}, nil
	}, 4)
	for _, depart := range []int{0, 1, 8} {
		if _, err := build(context.Background(), BuildOneRequest{DecoderThreads: depart}); err != nil {
			t.Fatal(err)
		}
	}
	for i, got := range vus {
		if got != 4 {
			t.Errorf("requete %d : %d fils, attendu 4", i, got)
		}
	}
	if len(vus) != 3 {
		t.Fatalf("%d requetes vues, attendu 3", len(vus))
	}
}
