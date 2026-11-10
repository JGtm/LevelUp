package infinite

// get_unique_test.go — les deux issues d'un GET unique (UGC, GameCMS) que les tests de forme ne
// figent pas : un corps 200 COUPÉ en cours de lecture n'est jamais rendu en succès
// ([*ErreurLecture]), et un échec d'émission du Doer est typé ([*ErreurTransport], message de la
// cause inchangé) sauf s'il est définitif (rendu tel quel).

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JGtm/halo-api"
	"github.com/JGtm/halo-api/haloapitest"
)

// errCoupure : la cause que rend un corps coupé.
var errCoupure = errors.New("connexion coupée en cours de lecture")

// corpsCoupe : un corps de réponse qui rend `reste` puis échoue avec [errCoupure].
type corpsCoupe struct{ reste string }

func (c *corpsCoupe) Read(p []byte) (int, error) {
	if c.reste == "" {
		return 0, errCoupure
	}
	n := copy(p, c.reste)
	c.reste = c.reste[n:]
	return n, nil
}

func (*corpsCoupe) Close() error { return nil }

// doerReponse : un Doer qui rend la réponse construite par la fonction, à chaque requête.
type doerReponse func() *http.Response

func (f doerReponse) Do(*http.Request) (*http.Response, error) { return f(), nil }

// appelUnique : un des trois appels à GET unique du client, avec l'URL qu'il demande.
type appelUnique struct {
	nom, url string
	appeler  func(c *Client) ([]byte, error)
}

func appelsUniques() []appelUnique {
	const urlCMS = "https://cms.test/hi/waypoint/file/images/x.png"
	return []appelUnique{
		{"FetchCarteUGC", "https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/a", func(c *Client) ([]byte, error) {
			carte, err := c.FetchCarteUGC(context.Background(), Auth{Spartan: "S"}, "a", "", identiteTest)
			if carte != nil {
				return []byte("carte rendue"), err
			}
			return nil, err
		}},
		{"TelechargerFichierUGC", "https://blob.test/m/x.mvar", func(c *Client) ([]byte, error) {
			return c.TelechargerFichierUGC(context.Background(), "https://blob.test/m", "x.mvar", identiteTest)
		}},
		{"TelechargerFichierGameCMS", urlCMS, func(c *Client) ([]byte, error) {
			return c.TelechargerFichierGameCMS(context.Background(), Auth{}, urlCMS)
		}},
	}
}

// TestGetUnique_CorpsCoupe : une réponse 200 dont le corps échoue en cours de lecture rend une
// [*ErreurLecture] (URL, cause), jamais les octets déjà reçus.
func TestGetUnique_CorpsCoupe(t *testing.T) {
	for _, a := range appelsUniques() {
		t.Run(a.nom, func(t *testing.T) {
			d := doerReponse(func() *http.Response {
				rep := haloapitest.Reponse(http.StatusOK, "")
				rep.Body = &corpsCoupe{reste: `{"AssetId":"a"`}
				return rep
			})
			octets, err := a.appeler(New(d, RetryAucune()))
			var el *ErreurLecture
			if !errors.As(err, &el) || el.URL != a.url || !errors.Is(err, errCoupure) {
				t.Fatalf("erreur %v, attendu *ErreurLecture de l'URL %s et de la cause de coupure", err, a.url)
			}
			if want := "lecture " + a.url + ": " + errCoupure.Error(); err.Error() != want {
				t.Errorf("message %q, attendu %q", err.Error(), want)
			}
			if len(octets) != 0 {
				t.Errorf("%d octet(s) rendu(s) malgré le corps coupé, attendu aucun", len(octets))
			}
		})
	}
}

// TestGetUnique_EchecEmission : un Doer qui échoue rend une [*ErreurTransport] (URL, message de la
// cause inchangé, non définitive) ; une erreur [haloapi.ErrDefinitif] du Doer est rendue telle
// quelle, sans enveloppe. Une seule émission, jamais de reprise.
func TestGetUnique_EchecEmission(t *testing.T) {
	cause := errors.New("connexion refusée")
	definitive := haloapi.NonRetryable(errors.New("sortie refusée"))
	for _, a := range appelsUniques() {
		t.Run(a.nom, func(t *testing.T) {
			d := &doerCompteur{err: cause}
			_, err := a.appeler(New(d, RetryAucune()))
			var et *ErreurTransport
			if !errors.As(err, &et) || et.URL != a.url || !errors.Is(err, cause) || errors.Is(err, haloapi.ErrDefinitif) {
				t.Fatalf("erreur %v, attendu *ErreurTransport de l'URL %s, non définitive, de cause %v", err, a.url, cause)
			}
			if err.Error() != cause.Error() {
				t.Errorf("message %q, attendu celui de la cause %q", err.Error(), cause.Error())
			}
			if d.n.Load() != 1 {
				t.Errorf("%d émission(s), attendu 1", d.n.Load())
			}

			dd := &doerCompteur{err: definitive}
			_, err = a.appeler(New(dd, RetryAucune()))
			if err != definitive { // identité voulue : rendue SANS enveloppe
				t.Errorf("erreur définitive du Doer : %v, attendu la même erreur, non enveloppée", err)
			}
			if errors.As(err, &et) {
				t.Errorf("erreur définitive enveloppée en *ErreurTransport : %v", err)
			}
		})
	}
}
