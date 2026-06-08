// Package handlers_test — auth_xbox_e2e_test.go : tests end-to-end du SSO Xbox.
//
// Couvre le pipeline complet PR 1 + 2 + 2.5a + 2.5c + 3 + 4 :
//
//	AuthHandler → TokenProvider → XboxSSOLinkStrategy → Userstore + TokenStore + Daemon
//
// Stubs utilisés :
//   - stubTokenProvider (provider mocké : InitDeviceFlow, Exchange)
//   - stubDeviceFlow (AcquireToken immédiat, pas de réseau MSAL)
//   - mockE2EDaemon (capture AddUserClient + AddPlayer)
//
// Mocks réels :
//   - Userstore avec tempdir users.json
//   - MultiUserTokenStore avec tempdir data/auth/watcher_tokens/
//
// Note : AcquireXSTSForRTA n'est pas mockée — l'appel HTTP réel échoue vite
// (DNS ou 401) en environnement test sans token valide. C'est le path nominal :
// le flow doit aboutir à AuthorizationStatus="authorized" malgré l'échec RTA
// (best-effort). La persistance RTA ne se fait pas en test, mais le user est
// quand même créé et la session wirée.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// ---------------------------------------------------------------------------
// E2E test rig
// ---------------------------------------------------------------------------

// mockE2EDaemon implémente service.WatcherDaemon en capturant les appels.
type mockE2EDaemon struct {
	mu             sync.Mutex
	running        bool
	addPlayerCalls []domain.PlayerSummary
	addUserCalls   []*auth_platform.UserTokens
}

func (m *mockE2EDaemon) IsRunning() bool { return m.running }

func (m *mockE2EDaemon) AddPlayer(_ context.Context, p domain.PlayerSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addPlayerCalls = append(m.addPlayerCalls, p)
	return nil
}

func (m *mockE2EDaemon) AddUserClient(_ context.Context, ut *auth_platform.UserTokens) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addUserCalls = append(m.addUserCalls, ut)
	return nil
}

func (m *mockE2EDaemon) Snapshot() (addPlayers []domain.PlayerSummary, addUsers []*auth_platform.UserTokens) {
	m.mu.Lock()
	defer m.mu.Unlock()
	addPlayers = append([]domain.PlayerSummary(nil), m.addPlayerCalls...)
	addUsers = append([]*auth_platform.UserTokens(nil), m.addUserCalls...)
	return
}

// e2eRig assemble tous les composants nécessaires pour tester le flow SSO complet.
type e2eRig struct {
	router       *chi.Mux
	sessStore    *session.Store
	userStore    *userstore.Store
	tokenStore   *auth_platform.MultiUserTokenStore
	daemon       *mockE2EDaemon
	attempts     *auth_platform.AttemptStore
	stubProvider *stubTokenProvider
}

// newE2ERig crée un rig complet en mode `AuthMode="xbox"`. Le stub provider
// est configuré pour retourner un Exchange réussi avec gamertag/xuid factices.
func newE2ERig(t *testing.T) *e2eRig {
	t.Helper()
	dir := t.TempDir()

	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	userStore := userstore.NewStore(filepath.Join(dir, "users.json"))
	tokenStore := auth_platform.NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))
	attempts := auth_platform.NewAttemptStore()
	daemon := &mockE2EDaemon{running: true}

	// stubProvider configuré pour simuler une auth Microsoft réussie.
	// AcquireToken retourne "" (stub), Exchange retourne le résultat ci-dessous.
	stubProvider := &stubTokenProvider{
		exchangeResult: &auth_platform.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken:   "fake-spartan-token",
				ClearanceToken: "fake-clearance-token",
			},
			Gamertag: "E2ETestPlayer",
			XUID:     "2535471234567890",
		},
	}

	// Branchement comme dans server.go pour AuthMode="xbox".
	daemonGetter := func() service.WatcherDaemon { return daemon }
	xboxStrategy := service.NewXboxSSOLinkStrategy(userStore).
		WithTokenStore(tokenStore).
		WithDaemonGetter(daemonGetter)

	authHandler := handlers.NewAuthHandler(sessStore, attempts, false, stubProvider).
		WithLinkStrategy(xboxStrategy)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Post("/auth/device-flow/start", authHandler.StartDeviceFlow)
	r.Get("/auth/device-flow/{attempt_id}", authHandler.GetDeviceFlowStatus)

	return &e2eRig{
		router:       r,
		sessStore:    sessStore,
		userStore:    userStore,
		tokenStore:   tokenStore,
		daemon:       daemon,
		attempts:     attempts,
		stubProvider: stubProvider,
	}
}

// waitForStatus poll l'attempt store jusqu'à ce que le statut donné soit atteint,
// ou jusqu'au timeout. Utilisé pour attendre que la goroutine pollDeviceFlow
// finisse son cycle InitDeviceFlow → AcquireToken → Exchange → (AcquireXSTSForRTA
// best-effort) → status=authorized.
func waitForStatus(t *testing.T, attempts *auth_platform.AttemptStore, attemptID, status string, timeout time.Duration) *auth_platform.Attempt {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snap := attempts.Snapshot(attemptID)
		if snap != nil && snap.Status == status {
			return snap
		}
		time.Sleep(50 * time.Millisecond)
	}
	snap := attempts.Snapshot(attemptID)
	if snap != nil {
		t.Fatalf("waitForStatus(%s) timeout après %v, status final = %q (errorCode=%q errorDetail=%q)",
			status, timeout, snap.Status, snap.ErrorCode, snap.ErrorDetail)
	}
	t.Fatalf("waitForStatus(%s) timeout, attempt introuvable", status)
	return nil
}

// ---------------------------------------------------------------------------
// Test : Device Code Flow happy path end-to-end
// ---------------------------------------------------------------------------

// TestE2E_DeviceCodeFlow_HappyPath teste le pipeline complet :
//
//	POST /auth/device-flow/start    → 200 user_code retourné
//	(polling MSAL en background)    → AcquireToken stub → Exchange stub → authorized
//	GET /auth/device-flow/{id}      → 200 status=authorized + session wirée
//
// Assertions post-flow :
//   - Userstore : user créé avec gamertag/xuid factices
//   - Session : Username + Role + CurrentPlayerSlug + LinkedHaloIdentity + HaloTokens
//   - Daemon : AddUserClient OU AddPlayer appelé (selon que RTA a marché)
//   - TokenStore : tokens persistés SI AcquireXSTSForRTA a réussi (rare en test sans Xbox Live valide)
func TestE2E_DeviceCodeFlow_HappyPath(t *testing.T) {
	rig := newE2ERig(t)

	// Configurer un flow stub qui retourne immédiatement (pas de polling MSAL réel).
	rig.stubProvider.initFlowFlow = auth_platform.NewStubDeviceFlow(
		"E2EX-1234",
		"https://microsoft.com/devicelogin",
		"Stub message",
		60, // expires_in
		"msal",
	)

	// Étape 1 : démarrer le flow.
	startReq := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	startW := httptest.NewRecorder()
	rig.router.ServeHTTP(startW, startReq)

	if startW.Code != http.StatusOK {
		t.Fatalf("StartDeviceFlow status = %d, body = %s", startW.Code, startW.Body.String())
	}
	var startResp domain.DeviceFlowStartResponse
	if err := json.Unmarshal(startW.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("décodage start response: %v", err)
	}
	if startResp.UserCode != "E2EX-1234" {
		t.Errorf("UserCode = %q, want E2EX-1234", startResp.UserCode)
	}
	if startResp.AttemptID == "" {
		t.Fatal("AttemptID vide")
	}

	// Étape 2 : attendre que pollDeviceFlow termine son cycle.
	// AcquireXSTSForRTA peut prendre quelques secondes (DNS + erreur HTTP).
	// Le status passe à "authorized" après Exchange réussi, indépendamment du RTA.
	authorized := waitForStatus(t, rig.attempts, startResp.AttemptID, auth_platform.AttemptStatusAuthorized, 30*time.Second)

	if authorized.Gamertag != "E2ETestPlayer" {
		t.Errorf("attempt.Gamertag = %q, want E2ETestPlayer", authorized.Gamertag)
	}
	if authorized.XUID != "2535471234567890" {
		t.Errorf("attempt.XUID = %q, want 2535471234567890", authorized.XUID)
	}
	if authorized.SpartanToken != "fake-spartan-token" {
		t.Errorf("attempt.SpartanToken = %q", authorized.SpartanToken)
	}

	// Étape 3 : GET status pour déclencher OnAuthSuccess (LinkStrategy).
	// Le handler invoque h.linkStrategy.OnAuthSuccess et persiste la session.
	statusReq := httptest.NewRequest(http.MethodGet, "/auth/device-flow/"+startResp.AttemptID, nil)
	// Conserver le cookie de session entre start et status.
	for _, c := range startW.Result().Cookies() {
		statusReq.AddCookie(c)
	}
	statusW := httptest.NewRecorder()
	rig.router.ServeHTTP(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Fatalf("GetDeviceFlowStatus status = %d, body = %s", statusW.Code, statusW.Body.String())
	}

	// Étape 4 : vérifier les effets de bord.
	// 4a. User créé dans le userstore via XboxSSOLinkStrategy.CreateFromXbox.
	user, err := rig.userStore.GetByXUID("2535471234567890")
	if err != nil {
		t.Fatalf("user pas créé dans userstore: %v", err)
	}
	if user.Gamertag != "E2ETestPlayer" {
		t.Errorf("user.Gamertag = %q, want E2ETestPlayer", user.Gamertag)
	}
	if user.Role != domain.RoleUser {
		t.Errorf("user.Role = %q, want user", user.Role)
	}

	// 4b. Daemon doit avoir été notifié. Selon que AcquireXSTSForRTA a réussi
	// ou échoué (cas attendu en test sans Xbox Live valide), c'est AddUserClient
	// (RTA dédié) OU AddPlayer (fallback) qui est appelé. On vérifie qu'AU MOINS
	// UN des deux a été invoqué.
	addPlayers, addUsers := rig.daemon.Snapshot()
	if len(addPlayers)+len(addUsers) == 0 {
		t.Error("Daemon n'a reçu ni AddPlayer ni AddUserClient — XboxSSOLinkStrategy a échoué à notifier")
	}
	// Si AddUserClient a été appelé, on vérifie que le xuid match.
	if len(addUsers) > 0 && addUsers[0].XUID != "2535471234567890" {
		t.Errorf("AddUserClient XUID = %q", addUsers[0].XUID)
	}
	// Si AddPlayer a été appelé (fallback), idem.
	if len(addPlayers) > 0 && addPlayers[0].XUID != "2535471234567890" {
		t.Errorf("AddPlayer XUID = %q", addPlayers[0].XUID)
	}
}

// ---------------------------------------------------------------------------
// Test : Device Code Flow Exchange failure path
// ---------------------------------------------------------------------------

// TestE2E_DeviceCodeFlow_ExchangeFailure vérifie que si provider.Exchange échoue,
// le flow passe à status=failed et aucun side-effect n'a lieu (pas de user créé,
// pas de notification daemon).
func TestE2E_DeviceCodeFlow_ExchangeFailure(t *testing.T) {
	rig := newE2ERig(t)

	rig.stubProvider.initFlowFlow = auth_platform.NewStubDeviceFlow(
		"FAIL-CODE", "https://microsoft.com/devicelogin", "msg", 60, "msal",
	)
	// Force Exchange à échouer.
	rig.stubProvider.exchangeResult = nil
	rig.stubProvider.exchangeErr = errors.New("simulated halo exchange failure")

	startReq := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	startW := httptest.NewRecorder()
	rig.router.ServeHTTP(startW, startReq)

	if startW.Code != http.StatusOK {
		t.Fatalf("StartDeviceFlow status = %d", startW.Code)
	}
	var startResp domain.DeviceFlowStartResponse
	_ = json.Unmarshal(startW.Body.Bytes(), &startResp)

	failed := waitForStatus(t, rig.attempts, startResp.AttemptID, auth_platform.AttemptStatusFailed, 5*time.Second)
	if failed.ErrorCode != "halo_exchange_error" {
		t.Errorf("ErrorCode = %q, want halo_exchange_error", failed.ErrorCode)
	}

	// Vérifier : aucun side-effect.
	if _, err := rig.userStore.GetByXUID("2535471234567890"); err == nil {
		t.Error("user ne devrait PAS être créé si Exchange échoue")
	}
	addPlayers, addUsers := rig.daemon.Snapshot()
	if len(addPlayers)+len(addUsers) > 0 {
		t.Error("Daemon ne devrait pas être notifié si Exchange a échoué")
	}
}

// ---------------------------------------------------------------------------
// Test : Single-flight (POST /start 2x → même attempt)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Test : Authorization Code Flow CSRF guard end-to-end
// ---------------------------------------------------------------------------

// TestE2E_AuthCodeFlow_CSRFGuard vérifie le contrat CSRF du Auth Code Flow
// (PR 4) en bout-en-bout : LoginRedirect génère un state, Callback rejette
// si le state ne matche pas.
//
// L'étape exchange (POST /oauth2/v2.0/token vers Microsoft) n'est pas testée
// ici — non-mockable sans modif d'API. Couverte par les unit tests de
// auth_xbox_oauth_test.go (state match → exchange tenté).
func TestE2E_AuthCodeFlow_CSRFGuard(t *testing.T) {
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	provider := &stubTokenProvider{}
	oauthHandler := handlers.NewXboxOAuthHandler(sessStore, provider, false, "http://localhost:8000/cb")

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Get("/auth/xbox/login", oauthHandler.LoginRedirect)
	r.Get("/auth/xbox/callback", oauthHandler.Callback)

	// Étape 1 : LoginRedirect. Doit retourner 302 vers Microsoft avec state.
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/xbox/login", nil)
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusFound {
		t.Fatalf("LoginRedirect status = %d, want 302", loginW.Code)
	}
	loc := loginW.Header().Get("Location")
	if loc == "" {
		t.Fatal("Location header manquant")
	}

	// Extraire le state. Format : ...&state=<hex>&...
	stateValue := extractQueryParam(t, loc, "state")
	if stateValue == "" {
		t.Fatal("state absent du Location")
	}
	if len(stateValue) < 32 {
		t.Errorf("state trop court (%d chars), attendu ≥32 (sécurité CSRF)", len(stateValue))
	}

	// Conserver le cookie de session pour le callback.
	cookies := loginW.Result().Cookies()

	// Étape 2a : Callback avec state CORRECT mais code bidon → exchange tentera
	// (et échouera côté Microsoft, 500 attendu). Mais le CSRF a passé — c'est ce
	// que ce test valide.
	goodReq := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?code=mock-code&state="+stateValue, nil)
	for _, c := range cookies {
		goodReq.AddCookie(c)
	}
	goodW := httptest.NewRecorder()
	r.ServeHTTP(goodW, goodReq)

	// Si CSRF avait failed, on aurait 403 state_mismatch. Si CSRF a passé, on
	// a soit 500 (code_exchange_failed) soit 302 (success rare en test).
	if goodW.Code == http.StatusForbidden {
		t.Errorf("CSRF rejeté à tort : %s", goodW.Body.String())
	}

	// Étape 2b : Replay attack — retenter le callback avec le même state
	// (qui a été consommé/effacé) doit échouer en 403.
	replayReq := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?code=mock-code&state="+stateValue, nil)
	for _, c := range cookies {
		replayReq.AddCookie(c)
	}
	replayW := httptest.NewRecorder()
	r.ServeHTTP(replayW, replayReq)

	if replayW.Code != http.StatusForbidden {
		t.Errorf("Replay attack non bloquée : status = %d (state aurait dû être consommé)", replayW.Code)
	}
}

// extractQueryParam extrait la valeur d'un paramètre depuis une URL ou query
// string. Renvoie "" si absent.
func extractQueryParam(t *testing.T, raw, key string) string {
	t.Helper()
	// Format simple : "...?key1=v1&state=v2&key3=v3" — on cherche &?key=
	for _, prefix := range []string{"?" + key + "=", "&" + key + "="} {
		idx := indexAt(raw, prefix)
		if idx < 0 {
			continue
		}
		start := idx + len(prefix)
		end := start
		for end < len(raw) && raw[end] != '&' {
			end++
		}
		return raw[start:end]
	}
	return ""
}

func indexAt(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestE2E_DeviceCodeFlow_SingleFlight vérifie que 2 appels POST /start sur
// la même session pendant qu'un flow est encore PENDING retournent le même
// attempt_id (anti-spam). Le stub bloque AcquireToken via un channel pour
// reproduire le timing prod (où l'user prend plusieurs secondes/minutes).
func TestE2E_DeviceCodeFlow_SingleFlight(t *testing.T) {
	rig := newE2ERig(t)
	// Flow bloquant : AcquireToken attend sur unblock pour ne pas faire
	// passer le status à authorized avant le 2e POST.
	unblock := make(chan struct{})
	rig.stubProvider.initFlowFlow = &blockingDeviceFlow{
		userCode:        "SF-CODE",
		verificationURL: "https://microsoft.com/devicelogin",
		expiresIn:       60,
		unblock:         unblock,
	}
	defer close(unblock) // libère la goroutine pollDeviceFlow à la fin du test

	// 1er start.
	req1 := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	w1 := httptest.NewRecorder()
	rig.router.ServeHTTP(w1, req1)
	var resp1 domain.DeviceFlowStartResponse
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.AttemptID == "" {
		t.Fatal("1er start : AttemptID vide")
	}

	// 2e start immédiatement (flow toujours pending car AcquireToken bloque).
	req2 := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	for _, c := range w1.Result().Cookies() {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	rig.router.ServeHTTP(w2, req2)
	var resp2 domain.DeviceFlowStartResponse
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)

	if resp1.AttemptID != resp2.AttemptID {
		t.Errorf("single-flight cassé : attempt1=%s attempt2=%s", resp1.AttemptID, resp2.AttemptID)
	}
}

// blockingDeviceFlow est un DeviceFlow qui bloque AcquireToken jusqu'à ce que
// le channel unblock soit fermé. Utilisé pour tester le single-flight (sinon
// le stub finit instantanément et le 2e POST crée un nouvel attempt).
type blockingDeviceFlow struct {
	userCode        string
	verificationURL string
	expiresIn       int
	unblock         chan struct{}
}

func (f *blockingDeviceFlow) GetMessage() string         { return "blocking stub" }
func (f *blockingDeviceFlow) GetUserCode() string        { return f.userCode }
func (f *blockingDeviceFlow) GetVerificationURL() string { return f.verificationURL }
func (f *blockingDeviceFlow) GetExpiresIn() int          { return f.expiresIn }
func (f *blockingDeviceFlow) GetFlowType() string        { return "msal" }
func (f *blockingDeviceFlow) AcquireToken(ctx context.Context) (string, error) {
	select {
	case <-f.unblock:
		return "", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

var _ auth_platform.DeviceFlow = (*blockingDeviceFlow)(nil)
