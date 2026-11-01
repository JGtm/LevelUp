package halo_5

// client_demo_test.go — en mode démo, le client Halo 5 construit par `NewClient` n'émet aucune
// requête, ni sur le chemin JSON ni sur le chemin binaire, et rend un refus que `errors.Is` reconnaît
// comme `netguard.ErrOffline`. La garde n'est plus un `netguard.Check` local au client : elle tient
// au `netguard.Doer` posé par `nouvelleAPI` sur chacun des deux chemins ; ce test la prouve sur le
// constructeur de production.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/haloapi/haloapitest"
	"levelup/go-api/internal/platform/netguard"
)

func TestClient_Demo_AucuneRequeteSurLesDeuxChemins(t *testing.T) {
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	e := &haloapitest.Enregistreur{}
	c := NewClient("jeton", 0).WithHTTPClient(&http.Client{Transport: e})

	_, errJSON := c.GetPlayerMatches(context.Background(), "JGtm", 0, 25)
	_, _, errBinaire := c.GetEmblemPNG(context.Background(), "JGtm")

	for nom, err := range map[string]error{"JSON": errJSON, "binaire": errBinaire} {
		if !errors.Is(err, netguard.ErrOffline) {
			t.Errorf("chemin %s : erreur %v, attendu netguard.ErrOffline", nom, err)
		}
	}
	if got := e.Requetes(); len(got) != 0 {
		t.Errorf("mode démo : %d requête(s) émise(s) %q, attendu aucune", len(got), got)
	}
}
