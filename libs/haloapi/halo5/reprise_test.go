package halo5

// reprise_test.go — la politique de reprise de Halo 5 et ce qu'elle fige dans le transport : les
// paramètres de [RetryHalo5], le limiteur attendu avant CHAQUE essai (son erreur rendue telle
// quelle), un corps 200 coupé retenté, un 200 vide sur le chemin JSON, les gardes d'arguments vides.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/JGtm/LevelUp/libs/haloapi/haloapitest"
)

// retryRapide : la politique de Halo 5 avec une attente de base négligeable (les paramètres de la
// politique réelle sont figés par [TestRetryHalo5_Parametres] ; ici on ne mesure que le nombre
// d'essais).
func retryRapide() (essais int, nouveau func(d *http.Client) *Client) {
	pol := RetryHalo5()
	pol.DelaiBase = time.Microsecond
	return pol.Essais, func(d *http.Client) *Client { return New(d, d, pol) }
}

// reponsesFixes : un enregistreur qui rend à chaque requête la réponse construite par `rep`.
func reponsesFixes(rep func() *http.Response) (*http.Client, *haloapitest.Enregistreur) {
	e := &haloapitest.Enregistreur{Repondre: func(int, *http.Request) *http.Response { return rep() }}
	return &http.Client{Transport: e}, e
}

// TestRetryHalo5_Parametres : les paramètres EXACTS de la politique de Halo 5.
func TestRetryHalo5_Parametres(t *testing.T) {
	r := RetryHalo5()
	if r.Essais != 4 {
		t.Errorf("Essais = %d, attendu 4", r.Essais)
	}
	if !r.AttenteFinale {
		t.Error("AttenteFinale = false : Halo 5 attend aussi après le dernier essai")
	}
	if !r.HonoreRetryAfter {
		t.Error("HonoreRetryAfter = false")
	}
	if r.ArretSurContexte || r.LimiteurUneFois || r.IgnoreErreurLimiteur {
		t.Errorf("ArretSurContexte=%v LimiteurUneFois=%v IgnoreErreurLimiteur=%v, attendu false/false/false",
			r.ArretSurContexte, r.LimiteurUneFois, r.IgnoreErreurLimiteur)
	}
	for n, veut := range []time.Duration{800 * time.Millisecond, 1600 * time.Millisecond, 3200 * time.Millisecond, 6400 * time.Millisecond} {
		if got := r.Delai(n, 0); got != veut {
			t.Errorf("Delai(%d, 0) = %v, attendu %v", n, got, veut)
		}
	}
	if got := r.Delai(0, 5*time.Second); got != 5*time.Second {
		t.Errorf("Delai(0, 5 s) = %v, attendu la fenêtre Retry-After de 5 s", got)
	}
	if got := r.Delai(0, 99*time.Second); got != 10*time.Second {
		t.Errorf("Delai(0, 99 s) = %v, attendu le plafond de 10 s", got)
	}
	if got := r.Delai(4, 0); got != 10*time.Second {
		t.Errorf("Delai(4, 0) = %v, attendu le plafond de 10 s", got)
	}
	for _, statut := range []int{400, 429, 500, 502, 503} {
		if !r.RetenteStatut(statut) {
			t.Errorf("RetenteStatut(%d) = false : Halo 5 retente tout statut non-200 hors 401/403/404/410", statut)
		}
	}
}

// limiteurEchec : un limiteur qui compte ses appels et rend `err` (nil : laisse passer).
type limiteurEchec struct {
	appels int
	err    error
}

func (l *limiteurEchec) Wait(context.Context) error {
	l.appels++
	return l.err
}

// TestLimiteur_AvantChaqueEssai : le limiteur est attendu avant CHAQUE essai (4 essais sur 500 : 4
// appels), et son erreur est rendue telle quelle, sans requête émise.
func TestLimiteur_AvantChaqueEssai(t *testing.T) {
	_, nouveau := retryRapide()

	d, e := reponsesFixes(func() *http.Response { return haloapitest.Reponse(http.StatusInternalServerError, "") })
	l := &limiteurEchec{}
	_, err := nouveau(d).WithLimiter(l).GetAppearance(context.Background(), Auth{Spartan: "s"}, "JGtm")
	if err == nil || len(e.Requetes()) != 4 || l.appels != 4 {
		t.Errorf("erreur %v, %d requête(s), %d attente(s) du limiteur, attendu une erreur, 4 et 4", err, len(e.Requetes()), l.appels)
	}

	cause := errors.New("limiteur saturé")
	d, e = reponsesFixes(func() *http.Response { return haloapitest.Reponse(http.StatusOK, "{}") })
	l = &limiteurEchec{err: cause}
	for nom, appel := range map[string]func(c *Client) error{
		"JSON": func(c *Client) error {
			_, err := c.GetAppearance(context.Background(), Auth{Spartan: "s"}, "JGtm")
			return err
		},
		"binaire": func(c *Client) error {
			_, _, err := c.GetEmblemPNG(context.Background(), Auth{Spartan: "s"}, "JGtm")
			return err
		},
	} {
		if err := appel(nouveau(d).WithLimiter(l)); !errors.Is(err, cause) {
			t.Errorf("%s : erreur %v, attendu l'erreur du limiteur", nom, err)
		}
	}
	if len(e.Requetes()) != 0 || l.appels != 2 {
		t.Errorf("%d requête(s), %d attente(s) du limiteur, attendu 0 et 2 (une par appel, aucune reprise)", len(e.Requetes()), l.appels)
	}
}

// corpsCoupe : un corps de réponse qui rend `reste` puis échoue avec `err`.
type corpsCoupe struct {
	reste string
	err   error
}

func (c *corpsCoupe) Read(p []byte) (int, error) {
	if c.reste == "" {
		return 0, c.err
	}
	n := copy(p, c.reste)
	c.reste = c.reste[n:]
	return n, nil
}

func (*corpsCoupe) Close() error { return nil }

// TestCorpsCoupeRetente : un 200 dont le corps échoue en cours de lecture est retenté (4 essais),
// puis échoue en enveloppant l'erreur de lecture ; aucun octet n'est rendu, sur les deux chemins.
func TestCorpsCoupeRetente(t *testing.T) {
	essais, nouveau := retryRapide()
	coupure := errors.New("connexion coupée")
	d, e := reponsesFixes(func() *http.Response {
		rep := haloapitest.Reponse(http.StatusOK, "")
		rep.Body = &corpsCoupe{reste: `{"Gamertag":"JG`, err: coupure}
		return rep
	})
	c := nouveau(d)

	app, errJSON := c.GetAppearance(context.Background(), Auth{Spartan: "s"}, "JGtm")
	if app != nil || !errors.Is(errJSON, coupure) || !strings.Contains(errJSON.Error(), "4 tentatives echouees") {
		t.Errorf("JSON : résultat %v, erreur %v, attendu aucun résultat et 4 tentatives échouées sur la coupure", app, errJSON)
	}
	octets, _, errBin := c.GetEmblemPNG(context.Background(), Auth{Spartan: "s"}, "JGtm")
	if len(octets) != 0 || !errors.Is(errBin, coupure) || !strings.Contains(errBin.Error(), "4 tentatives echouees") {
		t.Errorf("binaire : %d octet(s), erreur %v, attendu aucun octet et 4 tentatives échouées sur la coupure", len(octets), errBin)
	}
	if got := len(e.Requetes()); got != 2*essais {
		t.Errorf("%d requête(s), attendu %d (4 essais par appel)", got, 2*essais)
	}
}

// TestCorpsVideJSON_UnEssai : un 200 au corps vide sur le chemin JSON n'est pas une erreur du
// transport (contrairement à un rendu PNG) : un seul essai, puis l'erreur de décodage du JSON vide.
func TestCorpsVideJSON_UnEssai(t *testing.T) {
	_, nouveau := retryRapide()
	d, e := reponsesFixes(func() *http.Response { return haloapitest.Reponse(http.StatusOK, "") })
	_, err := nouveau(d).GetAppearance(context.Background(), Auth{Spartan: "s"}, "JGtm")
	var he *HTTPError
	if err == nil || errors.As(err, &he) || !strings.Contains(err.Error(), "GetAppearance decode:") {
		t.Errorf("erreur %v, attendu une erreur « GetAppearance decode: » et pas un *HTTPError", err)
	}
	if got := len(e.Requetes()); got != 1 {
		t.Errorf("%d requête(s), attendu 1 (le décodage n'est pas retenté)", got)
	}
}

// TestArgumentsVides : chacune des sept méthodes refuse un identifiant vide (ou blanc) avant toute
// requête, avec son message.
func TestArgumentsVides(t *testing.T) {
	_, nouveau := retryRapide()
	ctx, a := context.Background(), Auth{Spartan: "s"}
	cas := []struct {
		nom, veut string
		appeler   func(c *Client) error
	}{
		{"GetAppearance", "GetAppearance: gamertag vide", func(c *Client) error {
			_, err := c.GetAppearance(ctx, a, "  ")
			return err
		}},
		{"GetSpartanRenderPNG", "GetSpartanRenderPNG: gamertag vide", func(c *Client) error {
			_, _, err := c.GetSpartanRenderPNG(ctx, a, "  ")
			return err
		}},
		{"GetEmblemPNG", "GetEmblemPNG: gamertag vide", func(c *Client) error {
			_, _, err := c.GetEmblemPNG(ctx, a, "  ")
			return err
		}},
		{"GetPlayerMatches", "GetPlayerMatches: gamertag vide", func(c *Client) error {
			_, err := c.GetPlayerMatches(ctx, a, "  ", 0, 25)
			return err
		}},
		{"GetServiceRecords gamertag", "GetServiceRecords: gamertag vide", func(c *Client) error {
			_, err := c.GetServiceRecords(ctx, a, "  ", "arena")
			return err
		}},
		{"GetServiceRecords recordType", "GetServiceRecords: recordType vide", func(c *Client) error {
			_, err := c.GetServiceRecords(ctx, a, "JGtm", "  ")
			return err
		}},
		{"GetMatchCarnage matchID", "GetMatchCarnage: matchID/mode vide", func(c *Client) error {
			_, err := c.GetMatchCarnage(ctx, a, "", "arena")
			return err
		}},
		{"GetMatchCarnage mode", "GetMatchCarnage: matchID/mode vide", func(c *Client) error {
			_, err := c.GetMatchCarnage(ctx, a, "m1", "  ")
			return err
		}},
		{"GetMatchEvents", "GetMatchEvents: matchID vide", func(c *Client) error {
			_, err := c.GetMatchEvents(ctx, a, "")
			return err
		}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d, e := reponsesFixes(func() *http.Response { return haloapitest.Reponse(http.StatusOK, "{}") })
			if err := c.appeler(nouveau(d)); err == nil || err.Error() != c.veut {
				t.Errorf("erreur %v, attendu %q", err, c.veut)
			}
			if got := e.Requetes(); len(got) != 0 {
				t.Errorf("%d requête(s) émise(s) %q, attendu aucune", len(got), got)
			}
		})
	}
}
