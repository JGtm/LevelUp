package halo5

import (
	"context"
	"errors"
	"net/http"
	"testing"

	haloapi "github.com/JGtm/halo-api"
	"github.com/JGtm/halo-api/haloapitest"
)

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

// limiteurCompteur compte ses appels.
type limiteurCompteur struct{ appels int }

func (l *limiteurCompteur) Wait(context.Context) error {
	l.appels++
	return nil
}

// TestGarde_RefusAvantLimiteur : une garde qui refuse arrête l'appel avant le limiteur, sans
// requête émise, en rendant son erreur telle quelle ; chemins JSON et binaire.
func TestGarde_RefusAvantLimiteur(t *testing.T) {
	refus := haloapi.NonRetryable(errors.New("sortie refusée"))
	envoyes := 0
	d := doerGarde{refus: refus, envoyes: &envoyes}
	l := &limiteurCompteur{}
	c := New(d, d, RetryHalo5()).WithLimiter(l)
	ctx := context.Background()
	if _, err := c.GetAppearance(ctx, Auth{Spartan: "s"}, "JGtm"); !errors.Is(err, refus) {
		t.Errorf("JSON : erreur %v, attendu le refus de la garde", err)
	}
	if _, _, err := c.GetEmblemPNG(ctx, Auth{Spartan: "s"}, "JGtm"); !errors.Is(err, refus) {
		t.Errorf("binaire : erreur %v, attendu le refus de la garde", err)
	}
	if envoyes != 0 || l.appels != 0 {
		t.Errorf("%d requête(s) émise(s), %d appel(s) au limiteur ; attendu 0 et 0", envoyes, l.appels)
	}
}

// TestNew_SansDoer : comme `infinite.New`, le constructeur n'exige rien ; le Doer est obligatoire
// (la brique n'en construit aucun) et un appel sans Doer panique.
func TestNew_SansDoer(t *testing.T) {
	c := New(nil, nil, RetryHalo5())
	defer func() {
		if recover() == nil {
			t.Error("un appel sans Doer doit paniquer")
		}
	}()
	_, _ = c.GetAppearance(context.Background(), Auth{}, "JGtm")
}

// TestAuth_JetonMasque : ni `%v` ni `slog` ne révèlent le jeton.
func TestAuth_JetonMasque(t *testing.T) {
	a := Auth{Spartan: "jeton-secret"}
	if got := a.String(); got != "Auth{Spartan:***}" {
		t.Errorf("String() = %q", got)
	}
	if got := a.LogValue().String(); got != "[spartan=***]" {
		t.Errorf("LogValue() = %q", got)
	}
}
