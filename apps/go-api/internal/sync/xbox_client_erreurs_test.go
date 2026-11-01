package sync

// Messages d'erreur du client Xbox Achievements v2 recomposés depuis les erreurs typées de la brique
// `haloapi/xboxlive` (lot 2.4 du plan PUBLICATION) : mêmes textes qu'avant la coupe.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/platform/netguard"

	"github.com/JGtm/LevelUp/libs/haloapi"
	"github.com/JGtm/LevelUp/libs/haloapi/xboxlive"
)

// doerQuiEchoue : un Doer dont l'émission échoue avec `err`.
type doerQuiEchoue struct{ err error }

func (d doerQuiEchoue) Do(*http.Request) (*http.Response, error) { return nil, d.err }

func TestErreurXbox_Messages(t *testing.T) {
	cause := errors.New("réseau coupé")
	cas := map[string]struct {
		doer    haloapi.Doer
		attendu string
	}{
		"échec d'émission": {doerQuiEchoue{cause}, "xbox_client: HTTP do: réseau coupé"},
		"sortie refusée":   {doerQuiEchoue{haloapi.NonRetryable(netguard.ErrOffline)}, netguard.ErrOffline.Error()},
		"JSON invalide": {
			&http.Client{Transport: xboxFigeesEnregistreur([]string{"pas du JSON"}, nil)},
			"xbox_client: JSON decode: invalid character 'p' looking for beginning of value",
		},
	}
	for nom, c := range cas {
		hc := &xboxHTTPClient{api: xboxlive.New(c.doer)}
		_, err := hc.GetPlayerAchievements(context.Background(), xboxFigeesXuid, "en-US")
		if err == nil {
			t.Fatalf("%s : erreur attendue", nom)
		}
		if err.Error() != c.attendu {
			t.Errorf("%s : message %q, attendu %q", nom, err.Error(), c.attendu)
		}
		if nom == "sortie refusée" && (!errors.Is(err, netguard.ErrOffline) || !errors.Is(err, haloapi.ErrDefinitif)) {
			t.Errorf("%s : %v doit rester reconnue (ErrOffline, ErrDefinitif)", nom, err)
		}
	}
}
