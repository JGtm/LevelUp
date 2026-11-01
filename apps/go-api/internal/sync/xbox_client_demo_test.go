package sync

// xbox_client_demo_test.go — en mode démo, le client des succès Xbox construit par `NewXboxHTTPClient`
// n'émet aucune requête et rend un refus que `errors.Is` reconnaît comme `netguard.ErrOffline`. La
// garde n'est plus un `netguard.Check` local au client : elle tient au `netguard.Doer` posé par
// `nouvelleXboxAPI` ; ce test la prouve sur le constructeur de production, seul le `*http.Client`
// final étant remplacé par l'enregistreur.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/netguard"

	"github.com/JGtm/LevelUp/libs/haloapi/haloapitest"
)

func TestXboxClient_Demo_AucuneRequete(t *testing.T) {
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	c := NewXboxHTTPClient(&auth.XSTSResult{UserHash: "9876543210", Token: "jeton-xsts"}, xboxFigeesTitre)
	hc, ok := c.(*xboxHTTPClient)
	if !ok {
		t.Fatalf("NewXboxHTTPClient rend %T, attendu *xboxHTTPClient", c)
	}
	e := &haloapitest.Enregistreur{}
	hc.api = nouvelleXboxAPI(&http.Client{Transport: e})

	_, err := c.GetPlayerAchievements(context.Background(), xboxFigeesXuid, "en-US")

	if !errors.Is(err, netguard.ErrOffline) {
		t.Errorf("erreur %v, attendu netguard.ErrOffline", err)
	}
	if got := e.Requetes(); len(got) != 0 {
		t.Errorf("mode démo : %d requête(s) émise(s) %q, attendu aucune", len(got), got)
	}
}
