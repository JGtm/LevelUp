package infinite

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/haloapi"
)

// sansDelai rend la politique avec des attentes nulles (les séquences testent le NOMBRE d'essais ;
// les durées sont testées par [Retry.delai]).
func sansDelai(r Retry) Retry {
	r.DelaiBase = 0
	return r
}

// serveurScripte sert, requête après requête, les statuts de `statuts` (le dernier se répète) ;
// `retryAfter` est posé sur chaque réponse non vide. Rend le serveur et le compteur de requêtes.
func serveurScripte(t *testing.T, retryAfter string, statuts ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		i := int(n.Add(1)) - 1
		code := statuts[min(i, len(statuts)-1)]
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func getTest(c *Client, srv *httptest.Server) ([]byte, error) {
	return c.get(context.Background(), requete{url: srv.URL + "/x", auth: Auth{Spartan: "s"}})
}

// TestRetry_Delais : les attentes EXACTES des trois politiques (800 ms·2^n, Retry-After, plafond).
func TestRetry_Delais(t *testing.T) {
	cas := []struct {
		nom  string
		r    Retry
		n    int
		ra   time.Duration
		veut time.Duration
	}{
		{"provider n=0", RetryProvider(), 0, 0, 800 * time.Millisecond},
		{"provider n=1", RetryProvider(), 1, 0, 1600 * time.Millisecond},
		{"provider n=2", RetryProvider(), 2, 0, 3200 * time.Millisecond},
		{"provider plafond", RetryProvider(), 4, 0, 10 * time.Second},
		{"provider ignore Retry-After", RetryProvider(), 0, 7 * time.Second, 800 * time.Millisecond},
		{"discovery n=0", RetryDiscovery(), 0, 0, 800 * time.Millisecond},
		{"discovery n=1", RetryDiscovery(), 1, 0, 1600 * time.Millisecond},
		{"discovery sans plafond", RetryDiscovery(), 5, 0, 25600 * time.Millisecond},
		{"discovery ignore Retry-After", RetryDiscovery(), 0, 7 * time.Second, 800 * time.Millisecond},
		{"haloclient n=3", RetryHaloClient(), 3, 0, 6400 * time.Millisecond},
		{"haloclient Retry-After", RetryHaloClient(), 0, 7 * time.Second, 7 * time.Second},
		{"haloclient Retry-After plafonné", RetryHaloClient(), 0, 30 * time.Second, 10 * time.Second},
		{"haloclient plafond", RetryHaloClient(), 4, 0, 10 * time.Second},
	}
	for _, c := range cas {
		if got := c.r.delai(c.n, c.ra); got != c.veut {
			t.Errorf("%s : delai = %v, veut %v", c.nom, got, c.veut)
		}
	}
}

// TestRetry_Parametres : essais, statuts retentés et options des trois politiques.
func TestRetry_Parametres(t *testing.T) {
	p, d, h := RetryProvider(), RetryDiscovery(), RetryHaloClient()
	if p.Essais != 3 || d.Essais != 3 || h.Essais != 4 {
		t.Fatalf("essais = %d/%d/%d, veut 3/3/4", p.Essais, d.Essais, h.Essais)
	}
	for _, s := range []int{400, 429, 500, 503} {
		if !p.retente(s) {
			t.Errorf("provider doit retenter %d", s)
		}
	}
	for s, veut := range map[int]bool{400: false, 429: false, 500: true, 503: true} {
		if d.retente(s) != veut {
			t.Errorf("discovery retente(%d) = %v", s, !veut)
		}
	}
	for s, veut := range map[int]bool{400: true, 429: false, 500: true, 503: true} {
		if h.retente(s) != veut {
			t.Errorf("haloclient retente(%d) = %v", s, !veut)
		}
	}
	if !p.AttenteFinale || d.AttenteFinale || !h.AttenteFinale {
		t.Error("AttenteFinale : provider et haloclient oui, discovery non")
	}
	if !d.ArretSurContexte || p.ArretSurContexte || h.ArretSurContexte {
		t.Error("ArretSurContexte : discovery seule")
	}
	if !d.LimiteurUneFois || p.LimiteurUneFois || h.LimiteurUneFois {
		t.Error("LimiteurUneFois : discovery seule")
	}
	if !h.IgnoreErreurLimiteur || p.IgnoreErreurLimiteur || d.IgnoreErreurLimiteur {
		t.Error("IgnoreErreurLimiteur : haloclient seul")
	}
	if (Retry{}).retente(500) {
		t.Error("Retente nil : aucun statut retenté")
	}
}

// TestGet_Sequences : nombre de requêtes et verdict pour chaque politique, séquence par séquence.
func TestGet_Sequences(t *testing.T) {
	cas := []struct {
		nom       string
		r         Retry
		statuts   []int
		requetes  int32
		succes    bool
		statutErr int
	}{
		{"provider 429 retenté", RetryProvider(), []int{429, 429, 200}, 3, true, 0},
		{"provider 500 épuisé", RetryProvider(), []int{500}, 3, false, 500},
		{"provider 400 retenté", RetryProvider(), []int{400, 200}, 2, true, 0},
		{"discovery 429 rendu", RetryDiscovery(), []int{429, 200}, 1, false, 429},
		{"discovery 400 rendu", RetryDiscovery(), []int{400, 200}, 1, false, 400},
		{"discovery 5xx retentés", RetryDiscovery(), []int{502, 503, 200}, 3, true, 0},
		{"discovery 5xx épuisé", RetryDiscovery(), []int{503}, 3, false, 503},
		{"haloclient 429 immédiat", RetryHaloClient(), []int{429, 200}, 1, false, 429},
		{"haloclient 503 puis 200", RetryHaloClient(), []int{503, 200}, 2, true, 0},
		{"haloclient 503 épuisé", RetryHaloClient(), []int{503}, 4, false, 503},
	}
	for _, c := range cas {
		srv, n := serveurScripte(t, "", c.statuts...)
		body, err := getTest(New(srv.Client(), sansDelai(c.r)), srv)
		if got := n.Load(); got != c.requetes {
			t.Errorf("%s : %d requêtes, veut %d", c.nom, got, c.requetes)
		}
		if c.succes {
			if err != nil || string(body) != `{"ok":true}` {
				t.Errorf("%s : body=%q err=%v", c.nom, body, err)
			}
			continue
		}
		var he *HTTPError
		if !errors.As(err, &he) || he.StatusCode != c.statutErr {
			t.Errorf("%s : err=%v, veut HTTPError %d", c.nom, err, c.statutErr)
		}
	}
}

// TestGet_AuthEtAbsenceJamaisRetentes : 401/403/404/410 en un seul essai, pour les trois politiques.
func TestGet_AuthEtAbsenceJamaisRetentes(t *testing.T) {
	for _, r := range []Retry{RetryProvider(), RetryDiscovery(), RetryHaloClient()} {
		for code, auth := range map[int]bool{401: true, 403: true, 404: false, 410: false} {
			srv, n := serveurScripte(t, "", code, 200)
			_, err := getTest(New(srv.Client(), sansDelai(r)), srv)
			if n.Load() != 1 {
				t.Errorf("statut %d : %d requêtes, veut 1", code, n.Load())
			}
			if IsAuthError(err) != auth || IsNotFound(err) == auth || IsFilmGoneErr(err) == auth {
				t.Errorf("statut %d : IsAuthError=%v IsNotFound=%v", code, IsAuthError(err), IsNotFound(err))
			}
		}
	}
}

// TestGet_RetryAfterRendu : le 429 du client de synchronisation porte sa fenêtre Retry-After.
func TestGet_RetryAfterRendu(t *testing.T) {
	srv, n := serveurScripte(t, "7", 429)
	_, err := getTest(New(srv.Client(), sansDelai(RetryHaloClient())), srv)
	var he *HTTPError
	if !errors.As(err, &he) || he.RetryAfter != 7*time.Second || n.Load() != 1 {
		t.Fatalf("err=%v requêtes=%d, veut 429 RetryAfter=7s en 1 requête", err, n.Load())
	}
	if string(he.Body) != `{"ok":true}` {
		t.Errorf("Body = %q", he.Body)
	}
}

// doerCompteur : un Doer qui rend `err` (et compte ses appels).
type doerCompteur struct {
	n   atomic.Int32
	err error
}

func (d *doerCompteur) Do(*http.Request) (*http.Response, error) {
	d.n.Add(1)
	return nil, d.err
}

// TestGet_ErrDefinitifArreteSansAttente : une sortie refusée par le Doer arrête la boucle au premier
// essai, sans attente, l'erreur d'origine reconnaissable.
func TestGet_ErrDefinitifArreteSansAttente(t *testing.T) {
	cause := errors.New("sortie refusée")
	for _, r := range []Retry{RetryProvider(), RetryDiscovery(), RetryHaloClient()} {
		d := &doerCompteur{err: haloapi.NonRetryable(cause)}
		debut := time.Now()
		_, err := New(d, r).get(context.Background(), requete{url: "http://x.invalid/y"})
		if d.n.Load() != 1 || !errors.Is(err, haloapi.ErrDefinitif) || !errors.Is(err, cause) {
			t.Errorf("appels=%d err=%v", d.n.Load(), err)
		}
		if time.Since(debut) > 500*time.Millisecond {
			t.Errorf("attente après une erreur définitive : %v", time.Since(debut))
		}
	}
}

// TestGet_TransportRetente : une erreur de transport est retentée jusqu'à épuisement.
func TestGet_TransportRetente(t *testing.T) {
	d := &doerCompteur{err: errors.New("connexion coupée")}
	_, err := New(d, sansDelai(RetryHaloClient())).get(context.Background(), requete{url: "http://x.invalid/y"})
	if d.n.Load() != 4 || err == nil || !strings.Contains(err.Error(), "4 tentatives échouées") {
		t.Fatalf("appels=%d err=%v", d.n.Load(), err)
	}
}

// TestGet_AttenteFinale : provider attend après le dernier échec, discovery non.
func TestGet_AttenteFinale(t *testing.T) {
	for _, c := range []struct {
		r       Retry
		attente bool
	}{{RetryProvider(), true}, {RetryDiscovery(), false}} {
		r := c.r
		r.Essais, r.DelaiBase = 1, 300*time.Millisecond
		srv, _ := serveurScripte(t, "", 503)
		debut := time.Now()
		_, _ = getTest(New(srv.Client(), r), srv)
		ecoule := time.Since(debut)
		if c.attente != (ecoule >= 250*time.Millisecond) {
			t.Errorf("AttenteFinale=%v : écoulé %v", c.attente, ecoule)
		}
	}
}

// TestGet_ArretSurContexte : discovery rend ctx.Err() pendant l'attente ; provider poursuit et ses
// essais suivants échouent sur le contexte.
func TestGet_ArretSurContexte(t *testing.T) {
	for _, c := range []struct {
		r      Retry
		appels int32
	}{{RetryDiscovery(), 1}, {RetryProvider(), 3}} {
		r := c.r
		r.DelaiBase = 5 * time.Second
		d := &doerCompteur{err: errors.New("réseau")}
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_, err := New(d, r).get(ctx, requete{url: "http://x.invalid/y"})
		cancel()
		if d.n.Load() != c.appels {
			t.Errorf("ArretSurContexte=%v : %d appels, veut %d", r.ArretSurContexte, d.n.Load(), c.appels)
		}
		if r.ArretSurContexte && !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("discovery : err=%v, veut context.DeadlineExceeded", err)
		}
	}
}

// limiteurCompteur compte ses attentes et rend `err`.
type limiteurCompteur struct {
	n   atomic.Int32
	err error
}

func (l *limiteurCompteur) Wait(context.Context) error { l.n.Add(1); return l.err }

// TestGet_Limiteur : une fois (discovery) ou avant chaque essai ; erreur rendue ou ignorée.
func TestGet_Limiteur(t *testing.T) {
	srv, _ := serveurScripte(t, "", 503)
	for _, c := range []struct {
		r     Retry
		waits int32
	}{{RetryDiscovery(), 1}, {RetryProvider(), 3}, {RetryHaloClient(), 4}} {
		l := &limiteurCompteur{}
		_, _ = getTest(New(srv.Client(), sansDelai(c.r)).WithLimiter(l), srv)
		if l.n.Load() != c.waits {
			t.Errorf("Essais=%d : %d attentes du limiteur, veut %d", c.r.Essais, l.n.Load(), c.waits)
		}
	}
	echec := errors.New("limiteur")
	srv2, n := serveurScripte(t, "", 200)
	_, err := getTest(New(srv2.Client(), RetryProvider()).WithLimiter(&limiteurCompteur{err: echec}), srv2)
	if !errors.Is(err, echec) || n.Load() != 0 {
		t.Errorf("provider : err=%v requêtes=%d, veut l'erreur du limiteur sans requête", err, n.Load())
	}
	_, err = getTest(New(srv2.Client(), RetryHaloClient()).WithLimiter(&limiteurCompteur{err: echec}), srv2)
	if err != nil || n.Load() != 1 {
		t.Errorf("haloclient : err=%v requêtes=%d, veut l'erreur ignorée", err, n.Load())
	}
}

// TestGet_Entetes : Accept, Accept-Language donné par l'appelant, jetons non vides seulement.
func TestGet_Entetes(t *testing.T) {
	var mu sync.Mutex
	var vus []http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		vus = append(vus, r.Header.Clone())
		mu.Unlock()
	}))
	defer srv.Close()
	c := New(srv.Client(), RetryHaloClient())
	_, _ = c.get(context.Background(), requete{url: srv.URL, auth: Auth{Spartan: "sp", Clearance: "cl"}, langue: "xx-YY"})
	_, _ = c.get(context.Background(), requete{url: srv.URL})
	if len(vus) != 2 {
		t.Fatalf("%d requêtes", len(vus))
	}
	h := vus[0]
	if h.Get(enteteAccept) != typeJSON || h.Get(enteteLangue) != "xx-YY" || h.Get(enteteSpartan) != "sp" || h.Get(enteteClearance) != "cl" {
		t.Errorf("en-têtes = %v", h)
	}
	for _, k := range []string{enteteLangue, enteteSpartan, enteteClearance} {
		if _, ok := vus[1][http.CanonicalHeaderKey(k)]; ok {
			t.Errorf("en-tête %s posé sans valeur", k)
		}
	}
}

// TestAuth_JetonsMasques : ni `%v` ni un journal `slog` ne révèlent un jeton.
func TestAuth_JetonsMasques(t *testing.T) {
	a := Auth{Spartan: "jeton-spartan-secret", Clearance: "jeton-clearance-secret"}
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "auth", a)
	for _, rendu := range []string{fmt.Sprintf("%v", a), fmt.Sprintf("%+v", a), buf.String()} {
		if strings.Contains(rendu, "secret") {
			t.Errorf("rendu %q contient un jeton", rendu)
		}
	}
}

// TestParseRetryAfter : secondes, date HTTP, borne, illisible.
func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	// Les huit premiers cas portent les noms du test du client de synchronisation d'avant la brique.
	cas := []struct {
		nom  string
		h    string
		veut time.Duration
	}{
		{"empty", "", 0},
		{"seconds", "120", 120 * time.Second},
		{"zero", "0", 0},
		{"negative", "-5", 0},
		{"garbage", "abc", 0},
		{"clamp_over_max", "99999", maxRetryAfter},
		{"http_date_future", now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second},
		{"http_date_past", now.Add(-time.Minute).Format(http.TimeFormat), 0},
		{"secondes_bordees_d_espaces", " 3 ", 3 * time.Second},
		{"une_heure_bornee", strconv.Itoa(3600), maxRetryAfter},
		{"texte_libre", "n'importe quoi", 0},
		{"date_http_bornee", now.Add(time.Hour).Format(http.TimeFormat), maxRetryAfter},
		{"douze_secondes", "12", 12 * time.Second},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ParseRetryAfter(c.h, now); got != c.veut {
				t.Errorf("ParseRetryAfter(%q) = %v, veut %v", c.h, got, c.veut)
			}
		})
	}
}

// TestWith_CopieImmuable : un With* ne modifie pas le client d'origine.
func TestWith_CopieImmuable(t *testing.T) {
	c := New(&doerCompteur{}, RetryProvider())
	l := &limiteurCompteur{}
	c2 := c.WithLimiter(l).WithEndpoints(DefaultEndpoints().Avec(CleStats, "http://autre"))
	if c.limiter != nil || c.Endpoints().Hote(CleStats) == "http://autre" {
		t.Error("le client d'origine a changé")
	}
	if c2.limiter != l || c2.retry.Essais != 3 || c2.Endpoints().Hote(CleStats) != "http://autre" {
		t.Error("la copie ne porte pas les valeurs")
	}
}
