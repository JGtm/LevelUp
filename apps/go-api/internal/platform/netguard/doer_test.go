package netguard_test

// doer_test.go — le `netguard.Doer` que LevelUp passe aux clients de la brique `libs/haloapi` :
// en mode démo, `Verifie` et `Do` refusent avec une erreur à la fois `ErrOffline` et
// `haloapi.ErrDefinitif` (la brique s'arrête dessus sans reprise ni attente), sans émettre ; hors
// mode démo, CHAQUE `Do` repasse par la garde (une bascule en démo entre deux requêtes bloque la
// seconde).

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/platform/netguard"

	"github.com/JGtm/LevelUp/libs/haloapi"
	"github.com/JGtm/LevelUp/libs/haloapi/haloapitest"
)

// doerDeTest rend un Doer gardé sur un enregistreur de requêtes.
func doerDeTest() (netguard.Doer, *haloapitest.Enregistreur) {
	e := &haloapitest.Enregistreur{}
	return netguard.Doer{Surface: "doer_test.get", Next: &http.Client{Transport: e}}, e
}

func requeteDeTest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://tiers.test/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// TestDoer_RefusDefinitifSansEmission : en mode démo, `Verifie` et `Do` rendent un refus reconnu
// comme `ErrOffline` ET comme `ErrDefinitif`, et aucune requête n'est émise.
func TestDoer_RefusDefinitifSansEmission(t *testing.T) {
	netguard.SetOffline(true)
	t.Cleanup(func() { netguard.SetOffline(false) })
	d, e := doerDeTest()

	var garde haloapi.Garde = d // la brique consulte la garde avant tout limiteur
	resp, errDo := d.Do(requeteDeTest(t))
	if resp != nil {
		t.Error("Do : une réponse est rendue malgré le refus")
	}
	for nom, err := range map[string]error{"Verifie": garde.Verifie(context.Background()), "Do": errDo} {
		if !errors.Is(err, netguard.ErrOffline) {
			t.Errorf("%s : erreur %v, attendu netguard.ErrOffline", nom, err)
		}
		if !errors.Is(err, haloapi.ErrDefinitif) {
			t.Errorf("%s : erreur %v, attendu haloapi.ErrDefinitif (sans quoi la brique retenterait)", nom, err)
		}
	}
	if got := e.Requetes(); len(got) != 0 {
		t.Errorf("mode démo : %d requête(s) émise(s) %q, attendu aucune", len(got), got)
	}
}

// TestDoer_GardeAChaqueRequete : hors mode démo `Do` émet et `Verifie` laisse passer ; chaque `Do`
// repasse par la garde, donc une bascule en mode démo entre deux requêtes bloque la seconde.
func TestDoer_GardeAChaqueRequete(t *testing.T) {
	netguard.SetOffline(false)
	t.Cleanup(func() { netguard.SetOffline(false) })
	d, e := doerDeTest()

	if err := d.Verifie(context.Background()); err != nil {
		t.Fatalf("Verifie hors mode démo : %v, attendu nil", err)
	}
	resp, err := d.Do(requeteDeTest(t))
	if err != nil {
		t.Fatalf("Do hors mode démo : %v", err)
	}
	_ = resp.Body.Close()
	if got := e.Requetes(); len(got) != 1 {
		t.Fatalf("hors mode démo : %d requête(s), attendu 1", len(got))
	}

	netguard.SetOffline(true)
	if _, err := d.Do(requeteDeTest(t)); !errors.Is(err, netguard.ErrOffline) || !errors.Is(err, haloapi.ErrDefinitif) {
		t.Errorf("Do après bascule en mode démo : erreur %v, attendu un refus ErrOffline définitif", err)
	}
	if got := e.Requetes(); len(got) != 1 {
		t.Errorf("après bascule en mode démo : %d requête(s), attendu toujours 1", len(got))
	}

	netguard.SetOffline(false)
	resp, err = d.Do(requeteDeTest(t))
	if err != nil {
		t.Fatalf("Do après retour hors mode démo : %v", err)
	}
	_ = resp.Body.Close()
	if got := e.Requetes(); len(got) != 2 {
		t.Errorf("après retour hors mode démo : %d requête(s), attendu 2", len(got))
	}
}
