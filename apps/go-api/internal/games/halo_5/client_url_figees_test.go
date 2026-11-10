package halo_5

// Témoin T2 « URL figées » du client Halo 5 (plan PUBLICATION, lot 1.2) : pour chaque méthode, la
// requête exacte que le client pose sur le fil — méthode, URL absolue, en-têtes triés — pour des
// identifiants sains, écrite en littéral. Le client est déplacé sans changement de comportement
// (lot 2.1) : ces chaînes ne bougent pas.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/JGtm/halo-api"
	"github.com/JGtm/halo-api/halo5"
	"github.com/JGtm/halo-api/haloapitest"
)

const (
	figeesJeton   = "v4=abc.def.ghi"
	figeesMatchID = "6a2e1c0f-9d34-4b8a-8f17-3c5e7d9b0a12"
)

// clientEnregistreur : un client Halo 5 sans attente de limiteur, branché sur `e`.
func clientEnregistreur(e *haloapitest.Enregistreur) *Client {
	return NewClient(figeesJeton, 0).
		WithHTTPClient(&http.Client{Transport: e}).
		WithLimiter(rate.NewLimiter(rate.Inf, 1))
}

const (
	entetesJSON     = " [Accept=application/json; User-Agent=cpprestsdk/2.4.0; X-343-Authorization-Spartan=" + figeesJeton + "]"
	entetesBinaires = " [Accept=image/png,image/*; User-Agent=cpprestsdk/2.4.0; X-343-Authorization-Spartan=" + figeesJeton + "]"
)

// TestURLFigees_Halo5_Methodes : les sept méthodes Get*, une requête chacune, sans reprise.
func TestURLFigees_Halo5_Methodes(t *testing.T) {
	cas := []struct {
		nom     string
		appeler func(c *Client) error
		attendu string
	}{
		{"GetPlayerMatches", func(c *Client) error {
			_, err := c.GetPlayerMatches(context.Background(), "JGtm", 0, 25)
			return err
		}, "GET https://spartanstats.svc.halowaypoint.com/h5/players/JGtm/matches?start=0&count=25&include-times=true&auth=st" + entetesJSON},
		{"GetServiceRecords", func(c *Client) error {
			_, err := c.GetServiceRecords(context.Background(), "JGtm", "arena")
			return err
		}, "GET https://spartanstats.svc.halowaypoint.com/h5/servicerecords/arena?players=JGtm&auth=st" + entetesJSON},
		{"GetMatchCarnage", func(c *Client) error {
			_, err := c.GetMatchCarnage(context.Background(), figeesMatchID, "arena")
			return err
		}, "GET https://spartanstats.svc.halowaypoint.com/h5/arena/matches/" + figeesMatchID + "?auth=st" + entetesJSON},
		{"GetMatchEvents", func(c *Client) error {
			_, err := c.GetMatchEvents(context.Background(), figeesMatchID)
			return err
		}, "GET https://spartanstats.svc.halowaypoint.com/h5/matches/" + figeesMatchID + "/events?auth=st" + entetesJSON},
		{"GetAppearance", func(c *Client) error {
			_, err := c.GetAppearance(context.Background(), "JGtm")
			return err
		}, "GET https://haloplayer.svc.halowaypoint.com/h5/profiles/JGtm/appearance?auth=st" + entetesJSON},
		{"GetSpartanRenderPNG", func(c *Client) error {
			_, _, err := c.GetSpartanRenderPNG(context.Background(), "JGtm")
			return err
		}, "GET https://haloplayer.svc.halowaypoint.com/h5/profiles/JGtm/spartan?auth=st" + entetesBinaires},
		{"GetEmblemPNG", func(c *Client) error {
			_, _, err := c.GetEmblemPNG(context.Background(), "JGtm")
			return err
		}, "GET https://haloplayer.svc.halowaypoint.com/h5/profiles/JGtm/emblem?auth=st" + entetesBinaires},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response {
				return haloapitest.Reponse(http.StatusOK, "{}")
			}}
			if strings.HasSuffix(c.nom, "PNG") {
				e.Repondre = func(int, *http.Request) *http.Response {
					return haloapitest.Reponse(http.StatusOK, "PNG", "Content-Type", "image/png")
				}
			}
			if err := c.appeler(clientEnregistreur(e)); err != nil {
				t.Fatalf("%s : %v", c.nom, err)
			}
			if len(e.Requetes()) != 1 || e.Requetes()[0] != c.attendu {
				t.Errorf("requêtes = %q\nattendu    %q", e.Requetes(), c.attendu)
			}
			if strings.Contains(strings.ToLower(e.Requetes()[0]), "343-clearance") {
				t.Errorf("343-clearance ne doit jamais être posé en Halo 5 : %s", e.Requetes()[0])
			}
		})
	}
}

// TestURLFigees_Halo5_Encodage : le gamertag est échappé en chemin (PathEscape) dans l'historique et
// les profils, en requête (QueryEscape) dans `players=` ; l'identifiant de match en chemin.
func TestURLFigees_Halo5_Encodage(t *testing.T) {
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response { return haloapitest.Reponse(http.StatusOK, "{}") }}
	c := clientEnregistreur(e)
	ctx := context.Background()
	if _, err := c.GetPlayerMatches(ctx, "Sir Foo/é", 50, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetServiceRecords(ctx, "Sir Foo/é", "arena"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetAppearance(ctx, "Sir Foo/é"); err != nil {
		t.Fatal(err)
	}
	attendu := []string{
		"GET https://spartanstats.svc.halowaypoint.com/h5/players/Sir%20Foo%2F%C3%A9/matches?start=50&count=10&include-times=true&auth=st" + entetesJSON,
		"GET https://spartanstats.svc.halowaypoint.com/h5/servicerecords/arena?players=Sir+Foo%2F%C3%A9&auth=st" + entetesJSON,
		"GET https://haloplayer.svc.halowaypoint.com/h5/profiles/Sir%20Foo%2F%C3%A9/appearance?auth=st" + entetesJSON,
	}
	if strings.Join(e.Requetes(), "\n") != strings.Join(attendu, "\n") {
		t.Errorf("requêtes =\n%s\nattendu\n%s", strings.Join(e.Requetes(), "\n"), strings.Join(attendu, "\n"))
	}
}

// TestURLFigees_Halo5_RedirectionCDN : le 302 de l'emblème vers un autre hôte est suivi par le
// *http.Client. Faits figés : la seconde requête ne porte PAS `auth=st`, garde `User-Agent` et `Accept`
// (copiés de la première), reçoit `Referer` de la première URL, et ne porte PAS le jeton Spartan (le
// `netguard.Doer` le retire sur changement d'hôte).
func TestURLFigees_Halo5_RedirectionCDN(t *testing.T) {
	e := &haloapitest.Enregistreur{Repondre: func(n int, _ *http.Request) *http.Response {
		if n == 1 {
			return haloapitest.Reponse(http.StatusFound, "", "Location", "https://image.halocdn.com/h5/emblems/264_46_46_24?hash=sig")
		}
		return haloapitest.Reponse(http.StatusOK, "PNG", "Content-Type", "image/png")
	}}
	octets, ct, err := clientEnregistreur(e).GetEmblemPNG(context.Background(), "JGtm")
	if err != nil || string(octets) != "PNG" || ct != "image/png" {
		t.Fatalf("GetEmblemPNG = %q, %q, %v", octets, ct, err)
	}
	attendu := []string{
		"GET https://haloplayer.svc.halowaypoint.com/h5/profiles/JGtm/emblem?auth=st" + entetesBinaires,
		"GET https://image.halocdn.com/h5/emblems/264_46_46_24?hash=sig [Accept=image/png,image/*; Referer=https://haloplayer.svc.halowaypoint.com/h5/profiles/JGtm/emblem?auth=st; User-Agent=cpprestsdk/2.4.0]",
	}
	if strings.Join(e.Requetes(), "\n") != strings.Join(attendu, "\n") {
		t.Errorf("requêtes =\n%s\nattendu\n%s", strings.Join(e.Requetes(), "\n"), strings.Join(attendu, "\n"))
	}
}

// TestURLFigees_Halo5_StatutsTerminaux : 401, 403, 404 et 410 ne sont pas retentés (un seul essai),
// rendus en `*HTTPError` du statut ; le binaire suit la même règle ; un 200 au corps vide est une
// erreur du chemin binaire.
func TestURLFigees_Halo5_StatutsTerminaux(t *testing.T) {
	for _, statut := range []int{401, 403, 404, 410} {
		e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response { return haloapitest.Reponse(statut, "") }}
		c := clientEnregistreur(e)
		_, err := c.GetAppearance(context.Background(), "JGtm")
		var he *halo5.HTTPError
		if !errors.As(err, &he) || he.StatusCode != statut || len(e.Requetes()) != 1 {
			t.Errorf("JSON %d : erreur %v, %d essai(s), attendu *HTTPError %d et 1 essai", statut, err, len(e.Requetes()), statut)
		}
		_, _, err = c.GetEmblemPNG(context.Background(), "JGtm")
		if !errors.As(err, &he) || he.StatusCode != statut || len(e.Requetes()) != 2 {
			t.Errorf("binaire %d : erreur %v, %d essai(s) au total, attendu *HTTPError %d et 2 essais (un par appel)", statut, err, len(e.Requetes()), statut)
		}
	}
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response { return haloapitest.Reponse(http.StatusOK, "") }}
	var he *halo5.HTTPError
	if _, _, err := clientEnregistreur(e).GetEmblemPNG(context.Background(), "JGtm"); !errors.As(err, &he) || he.Err.Error() != "corps vide" {
		t.Errorf("200 au corps vide : erreur %v, attendu *HTTPError « corps vide »", err)
	}
}

// TestURLFigees_Halo5_ReprisePuisEchec : tout autre statut (500 comme 429) est retenté, quatre essais
// en tout, puis « 4 tentatives echouees » enveloppant le dernier `*HTTPError`. L'attente entre essais
// (800 ms de base) est coupée en annulant le contexte au premier appel : `Retry.Attendre` rend aussitôt
// sur `ctx.Done()` et la boucle continue, les quatre essais ont lieu quand même (fait Go figé).
func TestURLFigees_Halo5_ReprisePuisEchec(t *testing.T) {
	for _, statut := range []int{500, 429} {
		ctx, annuler := context.WithCancel(context.Background())
		e := &haloapitest.Enregistreur{
			Repondre: func(int, *http.Request) *http.Response { return haloapitest.Reponse(statut, "") },
			SurAppel: func(int) { annuler() },
		}
		c := clientEnregistreur(e)
		c.api = c.api.WithLimiter(nil) // un limiteur rend l'erreur du contexte annulé avant l'essai suivant
		_, err := c.GetAppearance(ctx, "JGtm")
		url := "https://haloplayer.svc.halowaypoint.com/h5/profiles/JGtm/appearance?auth=st"
		attendu := fmt.Sprintf("GetAppearance(JGtm): doGet %s: 4 tentatives echouees: HTTP %d: %s", url, statut, url)
		if err == nil || err.Error() != attendu {
			t.Errorf("statut %d : erreur %v, attendu %q", statut, err, attendu)
		}
		var he *halo5.HTTPError
		if !errors.As(err, &he) || he.StatusCode != statut || len(e.Requetes()) != 4 {
			t.Errorf("statut %d : %d essai(s), attendu 4 et un *HTTPError %d", statut, len(e.Requetes()), statut)
		}
	}
}

// TestURLFigees_Halo5_RetryAfterHonore : un 503 portant `Retry-After: 1` fait attendre au moins une
// seconde (plus que les 800 ms de base) avant le second essai, qui réussit.
func TestURLFigees_Halo5_RetryAfterHonore(t *testing.T) {
	e := &haloapitest.Enregistreur{Repondre: func(n int, _ *http.Request) *http.Response {
		if n == 1 {
			return haloapitest.Reponse(http.StatusServiceUnavailable, "", "Retry-After", "1")
		}
		return haloapitest.Reponse(http.StatusOK, "{}")
	}}
	debut := time.Now()
	if _, err := clientEnregistreur(e).GetAppearance(context.Background(), "JGtm"); err != nil {
		t.Fatal(err)
	}
	if ecoule := time.Since(debut); len(e.Requetes()) != 2 || ecoule < 950*time.Millisecond {
		t.Errorf("%d essai(s) en %v, attendu 2 essais séparés d'au moins 1 s (Retry-After)", len(e.Requetes()), ecoule)
	}
}

// TestURLFigees_Halo5_LectureDeRetryAfter : la fenêtre Retry-After lue (`haloapi.ParseRetryAfter`)
// puis plafonnée à 10 s par la politique Halo 5 : secondes entières ; une date HTTP PASSÉE, le texte,
// zéro et le négatif donnent 0. Depuis la coupe (lot 2.1), une date HTTP future est lue aussi.
func TestURLFigees_Halo5_LectureDeRetryAfter(t *testing.T) {
	for entree, attendu := range map[string]time.Duration{
		"":    0,
		"3":   3 * time.Second,
		" 7 ": 7 * time.Second,
		"99":  10 * time.Second,
		"0":   0,
		"-5":  0,
		"abc": 0,
		// l'ancien client rendait 0 pour une date (différence annoncée)
		time.Now().Add(time.Hour).UTC().Format(http.TimeFormat): 10 * time.Second,
	} {
		if got := min(haloapi.ParseRetryAfter(entree, time.Now()), halo5.RetryHalo5().Plafond); got != attendu {
			t.Errorf("Retry-After lu (%q) = %v, attendu %v", entree, got, attendu)
		}
	}
}
