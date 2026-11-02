package middleware

// desktop_gate_test.go — l'entrée de la coquille desktop, hors routeur (ADR 0045 D-3, lot L13.4) :
// l'échéance et la signature du cookie, son renouvellement à chaque entrée, le nom lié au port, et
// l'entrée fermée quand le lanceur n'a pas donné de secret. Les refus sur la pile réelle sont dans
// internal/api (desktop_entry_cgo_test.go).

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	gateTestAuthority = "127.0.0.1:49555"
	gateTestPort      = 49555
	gateTestKey       = "cle-de-cookie-de-test-de-trente-deux-octets" // pragma: allowlist secret
	gateTestSecret    = "secret-du-lanceur"                           // pragma: allowlist secret
)

func newTestGate(now time.Time) *DesktopGate {
	g := NewDesktopGate(gateTestAuthority, gateTestPort, gateTestKey, gateTestSecret)
	g.now = func() time.Time { return now }
	return g
}

// serve fait passer une requête par le garde ; rend le statut et si le handler aval a été atteint.
func serve(g *DesktopGate, target string, cookie *http.Cookie) (int, bool) {
	reached := false
	h := g.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = g.authority
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, reached
}

// enter passe par EnterHandler et rend le cookie posé (nil si aucun).
func enter(g *DesktopGate, secret string) (*httptest.ResponseRecorder, *http.Cookie) {
	req := httptest.NewRequest(http.MethodGet, DesktopEnterPath+"?secret="+secret, nil)
	rec := httptest.NewRecorder()
	g.EnterHandler(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == DesktopCookieName(gateTestPort) {
			return rec, c
		}
	}
	return rec, nil
}

func TestDesktopCookieName_PorteLePort(t *testing.T) {
	if got := DesktopCookieName(8123); got != "levelup_session_8123" {
		t.Errorf("DesktopCookieName(8123) = %q", got)
	}
	if DesktopCookieName(1) == DesktopCookieName(2) {
		t.Error("deux ports donnent le même nom de cookie")
	}
}

// Le cookie expire à 30 jours, pas avant ; une valeur altérée ou signée par une autre clé ou un autre
// port est refusée.
func TestDesktopGate_EcheanceEtSignatureDuCookie(t *testing.T) {
	t0 := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	g := newTestGate(t0)
	_, cookie := enter(g, gateTestSecret)
	if cookie == nil {
		t.Fatal("aucun cookie posé")
	}
	if code, reached := serve(g, "/api/v1/x", cookie); code != http.StatusOK || !reached {
		t.Fatalf("cookie frais : %d (atteint=%v), attendu 200", code, reached)
	}

	justBefore := newTestGate(t0.Add(DesktopCookieTTL - time.Minute))
	if code, _ := serve(justBefore, "/api/v1/x", cookie); code != http.StatusOK {
		t.Errorf("29 jours 23 h 59 : %d, attendu 200", code)
	}
	expired := newTestGate(t0.Add(DesktopCookieTTL + time.Second))
	if code, reached := serve(expired, "/api/v1/x", cookie); code != http.StatusUnauthorized || reached {
		t.Errorf("après 30 jours : %d (atteint=%v), attendu 401", code, reached)
	}

	// Prolonger l'échéance sans connaître la clé : la signature ne couvre plus la valeur.
	exp, sig, _ := strings.Cut(cookie.Value, ".")
	expN, _ := strconv.ParseInt(exp, 10, 64)
	prolonged := &http.Cookie{Name: cookie.Name, Value: strconv.FormatInt(expN+86400*365, 10) + "." + sig}
	for name, c := range map[string]*http.Cookie{
		"échéance prolongée":       prolonged,
		"sans point":               {Name: cookie.Name, Value: exp},
		"échéance illisible":       {Name: cookie.Name, Value: "x." + sig},
		"signature vide":           {Name: cookie.Name, Value: exp + "."},
		"valeur vide":              {Name: cookie.Name, Value: ""},
		"signature tronquée":       {Name: cookie.Name, Value: exp + "." + sig[:len(sig)-2]},
		"mauvais nom (autre port)": {Name: DesktopCookieName(gateTestPort + 1), Value: cookie.Value},
	} {
		if code, reached := serve(g, "/api/v1/x", c); code != http.StatusUnauthorized || reached {
			t.Errorf("%s : %d (atteint=%v), attendu 401", name, code, reached)
		}
	}

	otherKey := NewDesktopGate(gateTestAuthority, gateTestPort, "une-autre-cle-de-trente-deux-octets-ou-plus", gateTestSecret) // pragma: allowlist secret
	otherKey.now = g.now
	if code, _ := serve(otherKey, "/api/v1/x", cookie); code != http.StatusUnauthorized {
		t.Errorf("cookie signé par une autre clé : %d, attendu 401", code)
	}
	// Même clé et même secret, autre port : la signature est liée au port.
	otherPort := NewDesktopGate("127.0.0.1:49556", gateTestPort+1, gateTestKey, gateTestSecret)
	otherPort.now = g.now
	moved := &http.Cookie{Name: DesktopCookieName(gateTestPort + 1), Value: cookie.Value}
	if code, _ := serve(otherPort, "/api/v1/x", moved); code != http.StatusUnauthorized {
		t.Errorf("cookie d'un autre port rejoué sous le nom du nouveau : %d, attendu 401", code)
	}
}

// Chaque lancement (donc chaque garde) renouvelle l'échéance : une entrée plus tardive donne un cookie qui
// vit plus longtemps.
func TestDesktopGate_ChaqueEntreeRenouvelleLEcheance(t *testing.T) {
	t0 := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	_, first := enter(newTestGate(t0), gateTestSecret)
	later := t0.Add(20 * 24 * time.Hour)
	g := newTestGate(later)
	_, second := enter(g, gateTestSecret)
	if first == nil || second == nil || first.Value == second.Value {
		t.Fatalf("les deux entrées devaient donner des cookies différents : %v / %v", first, second)
	}
	// À 35 jours après la première entrée, le premier cookie est mort, le second vit encore.
	g.now = func() time.Time { return t0.Add(35 * 24 * time.Hour) }
	if code, _ := serve(g, "/api/v1/x", first); code != http.StatusUnauthorized {
		t.Errorf("premier cookie à 35 jours : %d, attendu 401", code)
	}
	if code, _ := serve(g, "/api/v1/x", second); code != http.StatusOK {
		t.Errorf("cookie renouvelé à 35 jours : %d, attendu 200", code)
	}
}

// Sans secret du lanceur, l'entrée est fermée : 404, aucun cookie, même avec un secret vide présenté.
func TestDesktopGate_EntreeFermeeSansSecretDuLanceur(t *testing.T) {
	g := NewDesktopGate(gateTestAuthority, gateTestPort, gateTestKey, "")
	for _, secret := range []string{"", "n-importe-quoi"} {
		rec, cookie := enter(g, secret)
		if rec.Code != http.StatusNotFound || cookie != nil {
			t.Errorf("secret %q sans configuration : %d, cookie=%v ; attendu 404 sans cookie", secret, rec.Code, cookie)
		}
	}
	noKey := NewDesktopGate(gateTestAuthority, gateTestPort, "", gateTestSecret)
	if rec, cookie := enter(noKey, gateTestSecret); rec.Code != http.StatusNotFound || cookie != nil {
		t.Errorf("sans clé de cookie : %d, cookie=%v ; attendu 404 sans cookie", rec.Code, cookie)
	}
}

// Le secret n'est jamais dans la réponse : ni en-tête, ni corps, ni adresse de redirection.
func TestDesktopGate_LeSecretNeFuitPas(t *testing.T) {
	g := newTestGate(time.Now())
	for _, secret := range []string{gateTestSecret, "mauvais-secret-essaye"} {
		rec, _ := enter(g, secret)
		dump := rec.Header().Get("Location") + rec.Body.String()
		for k, v := range rec.Header() {
			dump += k + strings.Join(v, ",")
		}
		if strings.Contains(dump, secret) {
			t.Errorf("le secret %q figure dans la réponse : %s", secret, dump)
		}
	}
}

// Le secret d'entrée sert une seule fois : la seconde présentation du bon secret est refusée (403) sans
// cookie, une présentation fausse ne le consomme pas, et le cookie de la première entrée reste valide.
func TestDesktopGate_SecretAUsageUnique(t *testing.T) {
	g := newTestGate(time.Now())
	if rec, cookie := enter(g, "faux"); rec.Code != http.StatusForbidden || cookie != nil {
		t.Fatalf("faux secret : %d cookie=%v, attendu 403 sans cookie", rec.Code, cookie)
	}
	first, cookie := enter(g, gateTestSecret)
	if first.Code != http.StatusFound || cookie == nil {
		t.Fatalf("première entrée : %d cookie=%v, attendu 302 avec cookie", first.Code, cookie)
	}
	second, again := enter(g, gateTestSecret)
	if second.Code != http.StatusForbidden || again != nil {
		t.Errorf("seconde entrée : %d cookie=%v, attendu 403 sans cookie", second.Code, again)
	}
	if code, _ := serve(g, "/api/v1/x", cookie); code != http.StatusOK {
		t.Errorf("le cookie de la première entrée : %d, attendu 200", code)
	}
}

// Deux entrées simultanées avec le bon secret : une seule réussit.
func TestDesktopGate_SecretAUsageUniqueSousConcurrence(t *testing.T) {
	g := newTestGate(time.Now())
	const racers = 16
	codes := make(chan int, racers)
	for range racers {
		go func() {
			rec, _ := enter(g, gateTestSecret)
			codes <- rec.Code
		}()
	}
	won := 0
	for range racers {
		if <-codes == http.StatusFound {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d entrées réussies sur %d simultanées, attendu exactement 1", won, racers)
	}
}

// Le cookie est SameSite=Lax, pas Strict : la coquille entre depuis sa page embarquée (origine
// levelup-shell://), la redirection vers « / » est une navigation inter-sites et Chromium n'y enverrait
// pas un cookie Strict (401 sur la première page). HttpOnly et le chemin racine restent exigés.
func TestDesktopGate_CookieSameSiteLax(t *testing.T) {
	_, cookie := enter(newTestGate(time.Now()), gateTestSecret)
	if cookie == nil {
		t.Fatal("aucun cookie posé")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, attendu Lax", cookie.SameSite)
	}
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.Secure {
		t.Errorf("cookie = %+v, attendu HttpOnly, Path=/, sans Secure", cookie)
	}
}
