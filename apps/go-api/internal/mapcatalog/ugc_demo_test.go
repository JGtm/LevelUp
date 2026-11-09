package mapcatalog

// ugc_demo_test.go — en mode démo, le client UGC construit par `NewClient` n'émet aucune requête, ni
// vers le Discovery ni vers le stockage blob, et rend un refus que `errors.Is` reconnaît comme
// `netguard.ErrOffline`. La garde n'est plus un `netguard.Check` local au client : elle tient au
// `netguard.Doer` posé par `nouvelleAPI` ; ce test la prouve sur le constructeur de production.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/netguard"

	"github.com/JGtm/LevelUp/libs/haloapi/haloapitest"
)

func TestClient_Demo_AucuneRequeteVersDiscoveryNiBlob(t *testing.T) {
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })

	e := &haloapitest.Enregistreur{}
	c := NewClient(&domain.HaloTokens{SpartanToken: "v4=SPARTAN"}).WithHTTPClient(&http.Client{Transport: e})

	_, errAsset := c.FetchAsset(context.Background(), ugcFigeesAsset, "")
	_, errBlob := c.FetchFileAt(context.Background(), ugcFigeesBlob, "MapVariant.mvar")

	for nom, err := range map[string]error{"FetchAsset": errAsset, "FetchFileAt": errBlob} {
		if !errors.Is(err, netguard.ErrOffline) {
			t.Errorf("%s : erreur %v, attendu netguard.ErrOffline", nom, err)
		}
	}
	if got := e.Requetes(); len(got) != 0 {
		t.Errorf("mode démo : %d requête(s) émise(s) %q, attendu aucune", len(got), got)
	}
}
