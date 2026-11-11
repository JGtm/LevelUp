package xboxlive

// succes_test.go — la page de succès Xbox Live : URL de la première et de la seconde page (clés
// `skipItems` puis `titleId`, `xuid(...)` ni échappé ni encodé), page sans `titleId`, les quatre
// en-têtes, statut non-200 avec le corps entier, échec d'émission, garde refusée sans requête, jeton
// masqué.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	haloapi "github.com/JGtm/halo-api"
	"github.com/JGtm/halo-api/haloapitest"
)

const (
	xuidTest   = "2535428000000001"
	titreTest  = "1144039928"
	hoteTest   = "https://succes.test"
	cheminTest = hoteTest + "/users/xuid(" + xuidTest + ")/achievements"
	autoTest   = "XBL3.0 x=9876543210;jeton-xsts"
	entetes    = " [Accept=application/json; Accept-Language=fr-FR; Authorization=" + autoTest + "; X-Xbl-Contract-Version=2]"
	pageUne    = `{"achievements":[{"id":"1","name":"Un"}],"pagingInfo":{"totalRecords":2,"continuationToken":"suite"}}`
)

var authTest = Auth{Autorisation: autoTest}

func clientSur(e *haloapitest.Enregistreur) *Client {
	return New(&http.Client{Transport: e}).WithEndpoints(Endpoints{Succes: hoteTest})
}

func TestGetSuccesPage_URLEtEntetes(t *testing.T) {
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
		return haloapitest.Reponse(http.StatusOK, pageUne)
	}}
	c := clientSur(e)
	rep, err := c.GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{TitleID: titreTest})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Achievements) != 1 || rep.Achievements[0].Name != "Un" || rep.PagingInfo.ContinuationToken != "suite" {
		t.Errorf("réponse = %+v, attendu l'élément Un et le jeton de suite", rep)
	}
	if _, err := c.GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{TitleID: titreTest, SkipItems: 100}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{}); err != nil {
		t.Fatal(err)
	}
	attendu := []string{
		"GET " + cheminTest + "?titleId=" + titreTest + entetes,
		"GET " + cheminTest + "?skipItems=100&titleId=" + titreTest + entetes,
		"GET " + cheminTest + entetes,
	}
	if got := strings.Join(e.Requetes(), "\n"); got != strings.Join(attendu, "\n") {
		t.Errorf("requêtes =\n%s\nattendu\n%s", got, strings.Join(attendu, "\n"))
	}
}

func TestGetSuccesPage_LangueVideEtHoteParDefaut(t *testing.T) {
	e := &haloapitest.Enregistreur{}
	c := New(&http.Client{Transport: e})
	if _, err := c.GetSuccesPage(context.Background(), authTest, xuidTest, "", Page{}); err == nil {
		t.Fatal("un corps vide n'est pas du JSON : erreur de décodage attendue")
	}
	attendu := "GET https://achievements.xboxlive.com/users/xuid(" + xuidTest + ")/achievements" +
		" [Accept=application/json; Accept-Language=; Authorization=" + autoTest + "; X-Xbl-Contract-Version=2]"
	if got := e.Requetes(); len(got) != 1 || got[0] != attendu {
		t.Errorf("requêtes = %v, attendu [%s]", got, attendu)
	}
}

func TestGetSuccesPage_StatutEnErreur(t *testing.T) {
	const refus = "corps de refus\nsur deux lignes"
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
		return haloapitest.Reponse(http.StatusInternalServerError, refus)
	}}
	_, err := clientSur(e).GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{})
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusInternalServerError || string(he.Body) != refus || he.URL != cheminTest {
		t.Fatalf("erreur = %v, attendu un HTTPError 500 au corps entier et à l'URL de la page", err)
	}
	if want := "HTTP 500: " + refus; err.Error() != want {
		t.Errorf("message = %q, attendu %q", err.Error(), want)
	}
	if len(e.Requetes()) != 1 {
		t.Errorf("%d requête(s), attendu 1 : aucune reprise", len(e.Requetes()))
	}
}

// doerQuiEchoue : un Doer dont l'émission échoue.
type doerQuiEchoue struct{ err error }

func (d doerQuiEchoue) Do(*http.Request) (*http.Response, error) { return nil, d.err }

func TestGetSuccesPage_EchecDEmission(t *testing.T) {
	cause := errors.New("réseau coupé")
	_, err := New(doerQuiEchoue{cause}).GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{})
	var et *ErreurTransport
	if !errors.As(err, &et) || !errors.Is(err, cause) || err.Error() != cause.Error() || errors.Is(err, haloapi.ErrDefinitif) {
		t.Errorf("erreur = %v, attendu un ErreurTransport au message du Doer, non définitif", err)
	}
	definitive := haloapi.NonRetryable(cause)
	_, err = New(doerQuiEchoue{definitive}).GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{})
	if !errors.Is(err, haloapi.ErrDefinitif) || errors.As(err, &et) {
		t.Errorf("erreur = %v, attendu l'erreur définitive du Doer telle quelle", err)
	}
}

// doerGarde : un Doer qui porte une garde refusant toute sortie et compte les requêtes reçues.
type doerGarde struct {
	refus   error
	envoyes *int
}

func (d doerGarde) Verifie(context.Context) error { return d.refus }

func (d doerGarde) Do(*http.Request) (*http.Response, error) {
	*d.envoyes++
	return haloapitest.Reponse(http.StatusOK, "{}"), nil
}

func TestGetSuccesPage_GardeRefuseSansRequete(t *testing.T) {
	refus := haloapi.NonRetryable(errors.New("sortie refusée"))
	envoyes := 0
	_, err := New(doerGarde{refus: refus, envoyes: &envoyes}).GetSuccesPage(context.Background(), authTest, xuidTest, "fr-FR", Page{})
	if !errors.Is(err, refus) || err.Error() != "sortie refusée" || envoyes != 0 {
		t.Errorf("erreur = %v, %d requête(s) ; attendu le refus de la garde tel quel et 0 requête", err, envoyes)
	}
}

func TestAuth_JetonMasque(t *testing.T) {
	a := Auth{Autorisation: "jeton-secret"}
	if got := a.String(); got != "Auth{Autorisation:***}" {
		t.Errorf("String() = %q", got)
	}
	if got := a.LogValue().String(); got != "[autorisation=***]" {
		t.Errorf("LogValue() = %q", got)
	}
}
