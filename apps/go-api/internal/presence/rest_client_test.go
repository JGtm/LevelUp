package presence

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fixtureOnlineHaloInfinite est le payload exact observé en prod 2026-05-25
// quand JGtm jouait à Halo Infinite. Format devices[].titles[] (cf.
// ParsePresencePayload dans event_parser.go).
const fixtureOnlineHaloInfinite = `{
  "xuid":"2533274823110022",
  "state":"Online",
  "devices":[{
    "type":"WindowsOneCore",
    "titles":[{
      "id":"2043073184",
      "name":"Halo Infinite",
      "placement":"Full",
      "state":"Active",
      "lastModified":"2026-05-25T19:53:30.5573644"
    }]
  }]
}`

const fixtureOfflineWithLastSeen = `{
  "xuid":"2533274823110022",
  "state":"Offline",
  "lastSeen":{
    "deviceType":"WindowsOneCore",
    "titleId":"2043073184",
    "titleName":"Halo Infinite",
    "timestamp":"2026-05-25T20:00:36.8996648"
  }
}`

const fixtureOnlineOtherGame = `{
  "xuid":"2533274858283686",
  "state":"Online",
  "devices":[{
    "type":"WindowsOneCore",
    "titles":[{
      "id":"2076696971",
      "name":"Counter-Strike 2",
      "placement":"Full",
      "state":"Active",
      "lastModified":"2026-05-25T18:20:30.7939601"
    }]
  }]
}`

// newTestServer monte un httptest qui répond avec le body fourni et le status.
// Permet de simuler n'importe quelle réponse Xbox sans dépendance réseau.
func newTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Asserts utiles : l'URL contient bien xuid( et l'auth header est passé.
		if !strings.Contains(r.URL.Path, "xuid(") {
			t.Errorf("URL path = %q, attendu contient xuid(", r.URL.Path)
		}
		if r.URL.Query().Get("level") != "all" {
			t.Errorf("level query = %q, attendu all", r.URL.Query().Get("level"))
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Authorization header manquant")
		}
		if r.Header.Get("x-xbl-contract-version") != "3" {
			t.Errorf("x-xbl-contract-version = %q, attendu 3", r.Header.Get("x-xbl-contract-version"))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// presenceClientPointingTo crée un PresenceClient dont le httpClient redirige
// toutes les requêtes vers le serveur de test (au lieu de la vraie URL Xbox).
// Astuce : on remplace le Transport pour réécrire le Host.
func presenceClientPointingTo(serverURL string) *PresenceClient {
	c := NewPresenceClient("XBL3.0 x=fakehash;faketoken")
	c.httpClient = &http.Client{
		Transport: &rewriteTransport{target: serverURL},
		Timeout:   restPresenceHTTPTimeout,
	}
	return c
}

// rewriteTransport réécrit toute requête sortante vers serverURL/<path>.
// Permet de tester sans monkey-patch ni URL mocking complexe.
type rewriteTransport struct {
	target string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Préserve le path + query d'origine, change uniquement scheme+host.
	newURL := rt.target + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header.Clone()
	return http.DefaultTransport.RoundTrip(newReq)
}

// ─── Tests ──────────────────────────────────────────────────────────────

func TestPresenceClient_GetPresence_OnlineHaloInfinite(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, fixtureOnlineHaloInfinite)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	event, err := client.GetPresence(context.Background(), "2533274823110022")
	if err != nil {
		t.Fatalf("GetPresence error = %v", err)
	}
	if event.PresenceState != "Online" {
		t.Errorf("PresenceState = %q, attendu Online", event.PresenceState)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil — parser n'a pas reconnu le payload")
	}
	if event.PresenceDetail.TitleID != "2043073184" {
		t.Errorf("TitleID = %q, attendu 2043073184", event.PresenceDetail.TitleID)
	}
	if event.PresenceDetail.TitleName != "Halo Infinite" {
		t.Errorf("TitleName = %q", event.PresenceDetail.TitleName)
	}
	if event.PresenceDetail.State != "Active" {
		t.Errorf("State = %q, attendu Active", event.PresenceDetail.State)
	}
}

func TestPresenceClient_GetPresence_OfflineLastSeen(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, fixtureOfflineWithLastSeen)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	event, err := client.GetPresence(context.Background(), "2533274823110022")
	if err != nil {
		t.Fatalf("GetPresence error = %v", err)
	}
	if event.PresenceState != "Offline" {
		t.Errorf("PresenceState = %q, attendu Offline", event.PresenceState)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail attendu nil pour Offline+lastSeen")
	}
}

func TestPresenceClient_GetPresence_OtherGame(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, fixtureOnlineOtherGame)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	event, err := client.GetPresence(context.Background(), "2533274858283686")
	if err != nil {
		t.Fatalf("GetPresence error = %v", err)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil")
	}
	if event.PresenceDetail.TitleID != "2076696971" {
		t.Errorf("TitleID = %q, attendu 2076696971 (CS2)", event.PresenceDetail.TitleID)
	}
	// Le titre CS2 n'est pas dans notre registre Halo — c'est au handler
	// watcher de l'ignorer (la responsabilité du client est juste de parser).
}

func TestPresenceClient_GetPresence_HTTPError_401_AuthExpired(t *testing.T) {
	ts := newTestServer(t, http.StatusUnauthorized, `{"error":"XSTS expired"}`)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	_, err := client.GetPresence(context.Background(), "xuid")
	if err == nil {
		t.Fatal("attendu erreur, reçu nil")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("erreur attendue de type *HTTPError, reçu %T: %v", err, err)
	}
	if !httpErr.IsAuthExpired() {
		t.Errorf("IsAuthExpired() = false sur 401")
	}
	if httpErr.IsTransient() {
		t.Errorf("IsTransient() = true sur 401, attendu false")
	}
}

func TestPresenceClient_GetPresence_HTTPError_429_RateLimit(t *testing.T) {
	ts := newTestServer(t, http.StatusTooManyRequests, ``)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	_, err := client.GetPresence(context.Background(), "xuid")
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("erreur attendue *HTTPError, reçu %T", err)
	}
	if !httpErr.IsRateLimited() {
		t.Errorf("IsRateLimited() = false sur 429")
	}
}

func TestPresenceClient_GetPresence_HTTPError_500_Transient(t *testing.T) {
	ts := newTestServer(t, http.StatusInternalServerError, `{"error":"upstream"}`)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	_, err := client.GetPresence(context.Background(), "xuid")
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("erreur attendue *HTTPError, reçu %T", err)
	}
	if !httpErr.IsTransient() {
		t.Errorf("IsTransient() = false sur 500")
	}
}

func TestPresenceClient_GetPresence_EmptyXUID(t *testing.T) {
	client := NewPresenceClient("auth")
	_, err := client.GetPresence(context.Background(), "")
	if err == nil {
		t.Fatal("attendu erreur pour xuid vide")
	}
}

func TestPresenceClient_UpdateAuth(t *testing.T) {
	client := NewPresenceClient("old-auth")
	if client.AuthHeader() != "old-auth" {
		t.Errorf("AuthHeader() = %q, attendu old-auth", client.AuthHeader())
	}
	client.UpdateAuth("new-auth")
	if client.AuthHeader() != "new-auth" {
		t.Errorf("après UpdateAuth, AuthHeader() = %q, attendu new-auth", client.AuthHeader())
	}
}

// Assure que le contexte annulé fait retourner l'erreur sans réseau réel.
func TestPresenceClient_GetPresence_ContextCanceled(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, fixtureOnlineHaloInfinite)
	defer ts.Close()

	client := presenceClientPointingTo(ts.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetPresence(ctx, "xuid")
	if err == nil {
		t.Fatal("attendu erreur sur ctx annulé")
	}
}
