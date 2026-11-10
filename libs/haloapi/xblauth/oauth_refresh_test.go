package xblauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JGtm/halo-api"
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
			// Jetons rendus : ceux de la réponse qui a abouti (reponseOK), aucun sur un échec.
			veutAT, veutRT := "at", "rt-2"
			if c.erreur {
				veutAT, veutRT = "", ""
			}
			if out.AccessToken != veutAT || out.RotatedRefreshToken != veutRT {
				t.Errorf("jetons rendus = (%q, %q), attendu (%q, %q)", out.AccessToken, out.RotatedRefreshToken, veutAT, veutRT)
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

// appelJournal : une requête reçue par l'un des deux endpoints factices.
type appelJournal struct {
	cible  string // "azure" ou "natif"
	rt     string // refresh_token présenté
	client string // client_id présenté
	secret bool   // client_secret présent
}

// journalAppels : l'ordre global des requêtes des deux endpoints.
type journalAppels struct {
	mu     sync.Mutex
	appels []appelJournal
}

func (j *journalAppels) ajouter(a appelJournal) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.appels = append(j.appels, a)
}

// ordre rend la suite des cibles jointes par « > », par exemple « natif>azure ».
func (j *journalAppels) ordre() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	cibles := make([]string, len(j.appels))
	for i, a := range j.appels {
		cibles[i] = a.cible
	}
	return strings.Join(cibles, ">")
}

// reponseHTTP : statut et corps d'une réponse factice.
type reponseHTTP struct {
	statut int
	corps  string
}

func ok200(corps string) func(bool) reponseHTTP {
	return func(bool) reponseHTTP { return reponseHTTP{http.StatusOK, corps} }
}

// secretRefuseSiPresent : Azure refuse le secret (AADSTS90023) et accepte le flux public.
func secretRefuseSiPresent(avecSecret bool) reponseHTTP {
	if avecSecret {
		return reponseHTTP{http.StatusOK, reponseSecretRefu}
	}
	return reponseHTTP{http.StatusOK, reponseOK}
}

// serveurJournal : un endpoint /token factice qui note chaque requête dans j sous `cible`.
func serveurJournal(t *testing.T, j *journalAppels, cible string, repondre func(secretPresent bool) reponseHTTP) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		secret := r.PostForm.Get("client_secret") != ""
		j.ajouter(appelJournal{cible: cible, rt: r.PostForm.Get("refresh_token"),
			client: r.PostForm.Get("client_id"), secret: secret})
		rep := repondre(secret)
		w.WriteHeader(rep.statut)
		_, _ = w.Write([]byte(rep.corps))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const (
	reponseRevoqueNatif = `{"error":"invalid_grant","error_description":"MSA: refresh token expiré"}`
	reponseVide         = `{}`
	reponseLimite       = `{"error":"temporarily_unavailable","error_description":"trop de requêtes"}`
)

// casNatifDabord : un scénario de TestRefreshWithRotation_NatifDabord.
type casNatifDabord struct {
	nom          string
	natifDabord  bool
	secret       string
	azure, natif func(bool) reponseHTTP
	ordre        string
	famille      string
	retenteNatif bool
	retentePub   bool
	erreur       bool
	classeErr    AuthErrorClass // "" = non vérifiée
	descErr      string         // sous-chaîne attendue du message d'erreur
}

func casNatifEnTete() []casNatifDabord {
	status500 := func(bool) reponseHTTP { return reponseHTTP{http.StatusInternalServerError, "boom"} }
	status429 := func(bool) reponseHTTP { return reponseHTTP{http.StatusTooManyRequests, reponseLimite} }
	return []casNatifDabord{
		{nom: "natif OK : un seul appel", natifDabord: true,
			azure: ok200(reponseOK), natif: ok200(reponseOK), ordre: "natif", famille: FamilleXboxNative},
		{nom: "natif révoqué puis Azure OK, sans second natif", natifDabord: true,
			azure: ok200(reponseOK), natif: ok200(reponseRevoqueNatif), ordre: "natif>azure", famille: FamilleAzure},
		{nom: "natif révoqué puis Azure révoqué : l'erreur d'Azure", natifDabord: true,
			azure: ok200(reponseRevoque), natif: ok200(reponseRevoqueNatif), ordre: "natif>azure",
			erreur: true, classeErr: AuthErrorRevoked, descErr: "AADSTS70000"},
		{nom: "natif révoqué, Azure avec secret refusé puis public", natifDabord: true, secret: "s",
			azure: secretRefuseSiPresent, natif: ok200(reponseRevoqueNatif), ordre: "natif>azure>azure",
			famille: FamilleAzure, retentePub: true},
		{nom: "natif en 500 : rendu tel quel, Azure intact", natifDabord: true,
			azure: ok200(reponseOK), natif: status500, ordre: "natif", erreur: true, classeErr: AuthErrorTransient},
		{nom: "natif en 429 : rendu tel quel, Azure intact", natifDabord: true,
			azure: ok200(reponseOK), natif: status429, ordre: "natif", erreur: true, classeErr: AuthErrorTransient},
		{nom: "natif en configuration puis Azure OK : Azure fait foi, sans second natif", natifDabord: true,
			azure: ok200(reponseOK), natif: ok200(reponseConfig), ordre: "natif>azure", famille: FamilleAzure},
		{nom: "natif en configuration puis Azure révoqué : l'erreur d'Azure", natifDabord: true,
			azure: ok200(reponseRevoque), natif: ok200(reponseConfig), ordre: "natif>azure",
			erreur: true, classeErr: AuthErrorRevoked, descErr: "AADSTS70000"},
		{nom: "réponse natif vide : jeton révoqué sans code, Azure répond", natifDabord: true,
			azure: ok200(reponseOK), natif: ok200(reponseVide), ordre: "natif>azure", famille: FamilleAzure},
	}
}

func casAzureDabord() []casNatifDabord {
	return []casNatifDabord{
		{nom: "défaut, Azure OK : un seul appel", azure: ok200(reponseOK), natif: ok200(reponseOK),
			ordre: "azure", famille: FamilleAzure},
		{nom: "défaut, Azure révoqué puis repli natif", azure: ok200(reponseRevoque), natif: ok200(reponseOK),
			ordre: "azure>natif", famille: FamilleXboxNative, retenteNatif: true},
		{nom: "défaut, Azure et natif révoqués : l'erreur d'Azure", azure: ok200(reponseRevoque),
			natif: ok200(reponseRevoqueNatif), ordre: "azure>natif", retenteNatif: true,
			erreur: true, classeErr: AuthErrorRevoked, descErr: "AADSTS70000"},
		{nom: "défaut, Azure en configuration : ni repli ni natif", azure: ok200(reponseConfig), natif: ok200(reponseOK),
			ordre: "azure", erreur: true, classeErr: AuthErrorConfig},
		{nom: "défaut, secret refusé puis public", secret: "s", azure: secretRefuseSiPresent, natif: ok200(reponseOK),
			ordre: "azure>azure", famille: FamilleAzure, retentePub: true},
	}
}

// TestRefreshWithRotation_NatifDabord : l'ordre des appels, la famille rendue, les reprises
// comptées et les jetons présentés, avec NatifDabord vrai puis faux (séquence inchangée).
func TestRefreshWithRotation_NatifDabord(t *testing.T) {
	for _, c := range append(casNatifEnTete(), casAzureDabord()...) {
		t.Run(c.nom, func(t *testing.T) {
			var j journalAppels
			az := serveurJournal(t, &j, "azure", c.azure)
			na := serveurJournal(t, &j, "natif", c.natif)
			o := OAuth{Doer: az.Client(), Azure: ClientAzure{ID: "id", Secret: c.secret}, URLAzure: az.URL,
				URLNatif: na.URL, NatifDabord: c.natifDabord}
			out, err := o.RefreshWithRotation(context.Background(), "rt-1")

			if got := j.ordre(); got != c.ordre {
				t.Fatalf("ordre des appels = %q, attendu %q", got, c.ordre)
			}
			if (err != nil) != c.erreur {
				t.Fatalf("err = %v, erreur attendue = %v", err, c.erreur)
			}
			if out.Famille != c.famille || out.RetenteNatif != c.retenteNatif || out.RetentePublique != c.retentePub {
				t.Errorf("issue = %+v, attendu famille=%q retenteNatif=%v retentePublique=%v",
					out, c.famille, c.retenteNatif, c.retentePub)
			}
			verifierIssue(t, c, out, err)
			verifierJetonsPresentes(t, &j)
		})
	}
}

// verifierIssue : jetons rendus sur un succès, classe et message de l'erreur sur un échec.
func verifierIssue(t *testing.T, c casNatifDabord, out RefreshOutcome, err error) {
	t.Helper()
	if !c.erreur {
		if out.AccessToken != "at" || out.RotatedRefreshToken != "rt-2" {
			t.Errorf("jetons rendus = (%q, %q), attendu (at, rt-2)", out.AccessToken, out.RotatedRefreshToken)
		}
		return
	}
	if out.AccessToken != "" || out.RotatedRefreshToken != "" {
		t.Errorf("jetons rendus sur un échec : %+v", out)
	}
	if c.classeErr != "" && ClassifyAuthError(err) != c.classeErr {
		t.Errorf("classe = %q, attendue %q (err = %v)", ClassifyAuthError(err), c.classeErr, err)
	}
	if c.descErr != "" && !strings.Contains(err.Error(), c.descErr) {
		t.Errorf("err = %v, attendu un message contenant %q", err, c.descErr)
	}
}

// verifierJetonsPresentes : le refresh_token n'est présenté qu'à des tentatives prévues, avec le
// client de l'endpoint visé ; le client natif n'envoie jamais de secret.
func verifierJetonsPresentes(t *testing.T, j *journalAppels) {
	t.Helper()
	j.mu.Lock()
	defer j.mu.Unlock()
	for i, a := range j.appels {
		if a.rt != "rt-1" {
			t.Errorf("appel %d (%s) : refresh_token %q, attendu rt-1", i, a.cible, a.rt)
		}
		switch a.cible {
		case "natif":
			if a.client != IDAppXboxNatif || a.secret {
				t.Errorf("appel natif %d : client=%q secret=%v, attendu le client natif sans secret", i, a.client, a.secret)
			}
		case "azure":
			if a.client != "id" {
				t.Errorf("appel azure %d : client=%q, attendu id", i, a.client)
			}
		}
	}
}

// TestRefreshWithRotation_NatifDabordErreurDefinitive : une sortie refusée par le Doer sur
// l'essai natif en tête est rendue telle quelle, sans appel Azure.
func TestRefreshWithRotation_NatifDabordErreurDefinitive(t *testing.T) {
	d := &doerRefuseURL{}
	out, err := OAuth{Doer: d, Azure: ClientAzure{Secret: "s"}, URLAzure: "https://a.test",
		URLNatif: "https://n.test", NatifDabord: true}.RefreshWithRotation(context.Background(), "rt-1")
	if !errors.Is(err, errSortieRefusee) || !errors.Is(err, haloapi.ErrDefinitif) {
		t.Fatalf("err = %v, attendu l'erreur définitive du Doer", err)
	}
	if len(d.urls) != 1 || d.urls[0] != "https://n.test" || out.RetentePublique || out.RetenteNatif {
		t.Errorf("urls = %v, issue = %+v : un seul appel, au natif, aucune reprise attendue", d.urls, out)
	}
}

// doerRefuseURL : un Doer qui refuse toute sortie et note l'adresse visée.
type doerRefuseURL struct{ urls []string }

func (d *doerRefuseURL) Do(r *http.Request) (*http.Response, error) {
	d.urls = append(d.urls, r.URL.String())
	return nil, haloapi.NonRetryable(errSortieRefusee)
}
