package xblauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/haloapi"
)

// serveurJeton : un endpoint /token factice qui répond `repondre(secretPresent)` et compte ses
// appels.
func serveurJeton(t *testing.T, appels *atomic.Int32, repondre func(secretPresent bool) string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appels.Add(1)
		_ = r.ParseForm()
		_, _ = w.Write([]byte(repondre(r.PostForm.Get("client_secret") != "")))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const (
	reponseOK         = `{"access_token":"at","refresh_token":"rt-2","expires_in":3600}`
	reponseRevoque    = `{"error":"invalid_grant","error_description":"AADSTS70000: grant issued to a different client"}`
	reponseSecretRefu = `{"error":"invalid_request","error_description":"AADSTS90023: Public clients can't send a client secret."}`
	reponseConfig     = `{"error":"invalid_client","error_description":"app inconnue"}`
)

// TestRefreshWithRotation_IssueTypee : l'issue dit quel client a répondu et quelles reprises ont
// été tentées — c'est ce que `platform/auth` compte dans ses compteurs expvar.
func TestRefreshWithRotation_IssueTypee(t *testing.T) {
	cas := map[string]struct {
		secret         string
		azure, natif   func(bool) string
		famille        string
		publique, nat  bool
		erreur         bool
		appelsNatifMax int32
	}{
		"secret accepté": {secret: "s", azure: func(bool) string { return reponseOK }, natif: func(bool) string { return reponseOK },
			famille: FamilleAzure},
		"AADSTS90023 puis public": {secret: "s", natif: func(bool) string { return reponseOK },
			azure: func(avecSecret bool) string {
				if avecSecret {
					return reponseSecretRefu
				}
				return reponseOK
			}, famille: FamilleAzure, publique: true},
		"invalid_grant puis natif": {azure: func(bool) string { return reponseRevoque }, natif: func(bool) string { return reponseOK },
			famille: FamilleXboxNative, nat: true, appelsNatifMax: 1},
		"invalid_grant des deux côtés": {azure: func(bool) string { return reponseRevoque }, natif: func(bool) string { return reponseRevoque },
			nat: true, erreur: true, appelsNatifMax: 1},
		"erreur de configuration : pas de repli": {azure: func(bool) string { return reponseConfig }, natif: func(bool) string { return reponseOK },
			erreur: true},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			var nAzure, nNatif atomic.Int32
			az := serveurJeton(t, &nAzure, c.azure)
			na := serveurJeton(t, &nNatif, c.natif)
			o := OAuth{Doer: az.Client(), Azure: ClientAzure{ID: "id", Secret: c.secret}, URLAzure: az.URL, URLNatif: na.URL}
			out, err := o.RefreshWithRotation(context.Background(), "rt-1")
			if (err != nil) != c.erreur {
				t.Fatalf("err = %v, erreur attendue = %v", err, c.erreur)
			}
			if out.Famille != c.famille || out.RetentePublique != c.publique || out.RetenteNatif != c.nat {
				t.Errorf("issue = %+v, attendu famille=%q publique=%v natif=%v", out, c.famille, c.publique, c.nat)
			}
			if nNatif.Load() > c.appelsNatifMax {
				t.Errorf("%d appel(s) au client natif, au plus %d attendu(s)", nNatif.Load(), c.appelsNatifMax)
			}
			if c.erreur && ClassifyAuthError(err) == AuthErrorTransient {
				t.Errorf("erreur %v : la classe du refus Azure doit être conservée", err)
			}
		})
	}
}

// TestRefreshWithRotation_RefreshTokenVide : aucun appel, issue vide, pas d'erreur.
func TestRefreshWithRotation_RefreshTokenVide(t *testing.T) {
	var n atomic.Int32
	srv := serveurJeton(t, &n, func(bool) string { return reponseOK })
	out, err := OAuth{Doer: srv.Client(), URLAzure: srv.URL, URLNatif: srv.URL}.RefreshWithRotation(context.Background(), "")
	if err != nil || out != (RefreshOutcome{}) || n.Load() != 0 {
		t.Errorf("issue = %+v, err = %v, appels = %d : attendu issue vide, nil, 0", out, err, n.Load())
	}
}

// doerRefuse : un Doer qui refuse toute sortie, comme la garde du mode démo.
type doerRefuse struct{ appels atomic.Int32 }

var errSortieRefusee = errors.New("sortie refusée (test)")

func (d *doerRefuse) Do(*http.Request) (*http.Response, error) {
	d.appels.Add(1)
	return nil, haloapi.NonRetryable(errSortieRefusee)
}

// TestRefreshWithRotation_ErreurDefinitiveTraverse : une erreur définitive du Doer est rendue
// telle quelle (même message, reconnue par errors.Is), sans repli ni reprise.
func TestRefreshWithRotation_ErreurDefinitiveTraverse(t *testing.T) {
	d := &doerRefuse{}
	out, err := OAuth{Doer: d, Azure: ClientAzure{Secret: "s"}, URLAzure: "https://a.test", URLNatif: "https://n.test"}.
		RefreshWithRotation(context.Background(), "rt-1")
	if !errors.Is(err, errSortieRefusee) || !errors.Is(err, haloapi.ErrDefinitif) {
		t.Fatalf("err = %v, attendu l'erreur définitive du Doer", err)
	}
	if err.Error() != errSortieRefusee.Error() {
		t.Errorf("message = %q, attendu %q (non réécrit)", err.Error(), errSortieRefusee.Error())
	}
	if d.appels.Load() != 1 || out.RetentePublique || out.RetenteNatif {
		t.Errorf("appels = %d, issue = %+v : un seul appel, aucune reprise attendue", d.appels.Load(), out)
	}
}
