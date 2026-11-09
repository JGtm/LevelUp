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
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// serveurEnTetes rend un serveur qui consigne les en-têtes de la dernière requête reçue sur /fin et
// redirige (302) toute autre requête vers `cible` (adresse absolue ou chemin).
func serveurEnTetes(t *testing.T, cible func() string) (*httptest.Server, func() http.Header) {
	t.Helper()
	var mu sync.Mutex
	var vus http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fin" {
			mu.Lock()
			vus = r.Header.Clone()
			mu.Unlock()
			return
		}
		http.Redirect(w, r, cible(), http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	return srv, func() http.Header {
		mu.Lock()
		defer mu.Unlock()
		return vus
	}
}

// requeteAuthentifiee émet vers `url` une requête qui porte les en-têtes d'identité de la brique
// plus un en-tête neutre (`Accept`), par un Doer gardé sur un client sans `CheckRedirect`.
func requeteAuthentifiee(t *testing.T, url string, next *http.Client) {
	t.Helper()
	netguard.SetOffline(false)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer jeton")
	req.Header.Set("X-343-Authorization-Spartan", "spartan-jeton")
	req.Header.Set("343-Clearance", "clearance-jeton")
	req.Header.Set("Accept", "image/png")
	resp, err := netguard.Doer{Surface: "doer_test.redirection", Next: next}.Do(req)
	if err != nil {
		t.Fatalf("Do : %v", err)
	}
	_ = resp.Body.Close()
}

// TestDoer_RedirectionAutreHoteRetireIdentite : un 302 vers un autre hôte (second serveur, autre
// port) n'y réémet ni `Authorization`, ni le jeton Spartan, ni la clearance ; les en-têtes neutres
// suivent.
func TestDoer_RedirectionAutreHoteRetireIdentite(t *testing.T) {
	tiers, vusTiers := serveurEnTetes(t, func() string { return "" })
	origine, _ := serveurEnTetes(t, func() string { return tiers.URL + "/fin" })
	if origine.Listener.Addr().String() == tiers.Listener.Addr().String() {
		t.Fatal("les deux serveurs partagent la même adresse")
	}

	requeteAuthentifiee(t, origine.URL+"/depart", &http.Client{})

	got := vusTiers()
	if got == nil {
		t.Fatal("le serveur tiers n'a pas reçu la requête redirigée")
	}
	for _, nom := range []string{"Authorization", "X-343-Authorization-Spartan", "343-Clearance"} {
		if v := got.Get(nom); v != "" {
			t.Errorf("hôte tiers : %s = %q, attendu absent", nom, v)
		}
	}
	if got.Get("Accept") != "image/png" {
		t.Errorf("hôte tiers : Accept = %q, attendu conservé", got.Get("Accept"))
	}
}

// TestDoer_RedirectionMemeHoteGardeIdentite : un 302 vers le MÊME hôte conserve les en-têtes
// d'identité.
func TestDoer_RedirectionMemeHoteGardeIdentite(t *testing.T) {
	origine, vus := serveurEnTetes(t, func() string { return "/fin" })

	requeteAuthentifiee(t, origine.URL+"/depart", &http.Client{})

	got := vus()
	if got == nil {
		t.Fatal("la requête redirigée n'est pas arrivée")
	}
	if got.Get("X-343-Authorization-Spartan") != "spartan-jeton" || got.Get("343-Clearance") != "clearance-jeton" ||
		got.Get("Authorization") != "Bearer jeton" {
		t.Errorf("même hôte : en-têtes d'identité = %v, attendu conservés", got)
	}
}

// TestDoer_RedirectionRespecteLaPolitiqueDeNext : le `CheckRedirect` du client sous-jacent reste
// appliqué (ici : ne pas suivre), et le client partagé n'est pas modifié.
func TestDoer_RedirectionRespecteLaPolitiqueDeNext(t *testing.T) {
	tiers, vusTiers := serveurEnTetes(t, func() string { return "" })
	origine, _ := serveurEnTetes(t, func() string { return tiers.URL + "/fin" })
	next := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	requeteAuthentifiee(t, origine.URL+"/depart", next)

	if vusTiers() != nil {
		t.Error("la redirection a été suivie malgré le CheckRedirect de Next")
	}
	if next.CheckRedirect == nil {
		t.Error("le client Next a été modifié")
	}
}

// TestDoer_RedirectionAllerRetourRetireIdentite : dans une chaîne A -> B -> A, le retrait des en-têtes
// d'identité est collant (comme celui d'`Authorization` par `net/http`) : la requête finale reçue par
// A, de retour sur l'hôte d'origine, ne les porte pas ; les en-têtes neutres suivent.
func TestDoer_RedirectionAllerRetourRetireIdentite(t *testing.T) {
	var a *httptest.Server
	b, _ := serveurEnTetes(t, func() string { return a.URL + "/fin" })
	a, vusA := serveurEnTetes(t, func() string { return b.URL + "/saut" })

	requeteAuthentifiee(t, a.URL+"/depart", &http.Client{})

	got := vusA()
	if got == nil {
		t.Fatal("A n'a pas reçu la requête finale de la chaîne A -> B -> A")
	}
	for _, nom := range []string{"Authorization", "X-343-Authorization-Spartan", "343-Clearance"} {
		if v := got.Get(nom); v != "" {
			t.Errorf("retour sur l'hôte d'origine : %s = %q, attendu absent (retrait collant)", nom, v)
		}
	}
	if got.Get("Accept") != "image/png" {
		t.Errorf("retour sur l'hôte d'origine : Accept = %q, attendu conservé", got.Get("Accept"))
	}
}

// TestDoer_RedirectionsLimiteeADix : sans `CheckRedirect` sur `Next`, une boucle de redirections est
// coupée comme `net/http` le fait : dix requêtes reçues (l'origine plus neuf sauts ; le dixième saut est
// refusé), puis une erreur.
func TestDoer_RedirectionsLimiteeADix(t *testing.T) {
	netguard.SetOffline(false)
	var recues atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recues.Add(1)
		http.Redirect(w, r, srv.URL+"/boucle", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	// Délai borné : sans la limite, la boucle ne finirait pas et le test échouerait par le compte, pas par blocage.
	ctx, annule := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(annule)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/boucle", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := netguard.Doer{Surface: "doer_test.boucle", Next: &http.Client{}}.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("Do : aucune erreur sur une boucle de redirections")
	}
	if got := recues.Load(); got != 10 {
		t.Errorf("le serveur a reçu %d requête(s), attendu 10 (l'origine plus neuf sauts, le dixième étant refusé)", got)
	}
}
