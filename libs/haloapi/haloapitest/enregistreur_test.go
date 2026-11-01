package haloapitest

import (
	"io"
	"net/http"
	"testing"
)

func TestLigneDeRequete(t *testing.T) {
	avec, _ := http.NewRequest(http.MethodGet, "https://exemple.test/a/b?x=1", nil)
	avec.Header.Set("x-jeton", "v4=abc")
	avec.Header.Set("Accept", "application/json")
	if got, want := LigneDeRequete(avec), "GET https://exemple.test/a/b?x=1 [Accept=application/json; X-Jeton=v4=abc]"; got != want {
		t.Errorf("deux en-têtes : %q, attendu %q", got, want)
	}
	sans, _ := http.NewRequest(http.MethodPost, "https://exemple.test/", nil)
	if got, want := LigneDeRequete(sans), "POST https://exemple.test/ []"; got != want {
		t.Errorf("sans en-tête : %q, attendu %q", got, want)
	}
}

func TestEnregistreur(t *testing.T) {
	var appels []int
	e := &Enregistreur{
		Repondre: func(n int, _ *http.Request) *http.Response {
			return Reponse(200+n, "corps", "Retry-After", "3", "orphelin")
		},
		SurAppel: func(n int) { appels = append(appels, n) },
	}
	client := &http.Client{Transport: e}
	for _, chemin := range []string{"a", "b"} {
		resp, err := client.Get("https://exemple.test/" + chemin)
		if err != nil {
			t.Fatal(err)
		}
		corps, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.Request == nil || string(corps) != "corps" || resp.Header.Get("Retry-After") != "3" || len(resp.Header) != 1 {
			t.Errorf("réponse inattendue : %+v corps %q", resp, corps)
		}
	}
	if got := e.Requetes(); len(got) != 2 || got[0] != "GET https://exemple.test/a []" || got[1] != "GET https://exemple.test/b []" {
		t.Errorf("requêtes = %q", got)
	}
	if len(appels) != 2 || appels[0] != 1 || appels[1] != 2 {
		t.Errorf("SurAppel = %v, attendu [1 2]", appels)
	}
	// Sans réponse choisie : 200 au corps vide.
	rep, err := (&Enregistreur{}).RoundTrip(avec(t))
	if err != nil || rep.StatusCode != http.StatusOK {
		t.Errorf("réponse par défaut : %v, erreur %v", rep, err)
	}
}

func avec(t *testing.T) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "https://exemple.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
