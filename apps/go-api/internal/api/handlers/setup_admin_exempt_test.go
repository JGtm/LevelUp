// Package handlers_test — setup_admin_exempt_test.go : POST /setup/players, les
// deux gardes d'ouverture de l'instance et l'exemption ADMIN (ADR 0035 D5).
//
// Déclarer le profil d'un ami est un acte d'administration : un admin passe le
// verrou d'instance ET l'interrupteur d'auto-provisioning. Tout autre appelant
// subit les deux, dans cet ordre (provisioning d'abord, verrou ensuite).
package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// setupGuardCase décrit une combinaison (rôle de l'appelant, gardes) et le
// statut attendu.
type setupGuardCase struct {
	name             string
	role             string
	locked           bool
	canSelfProvision bool
	wantStatus       int
	wantCode         string // sous-chaîne attendue dans le corps ("" = non vérifié)
}

// newGuardRouter monte le SetupHandler avec un résolveur de verrou injecté et
// SANS middleware de session : la session (donc le rôle) est posée par requête
// via middleware.InjectSession.
func newGuardRouter(t *testing.T, locked, canSelfProvision bool, svc *mockProfileService) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
	}
	sessionStore := session_platform.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))

	appCfg, err := settingsStore.Load()
	if err != nil {
		t.Fatalf("settings.Load: %v", err)
	}
	appCfg.CanSelfProvision = canSelfProvision
	if err := settingsStore.Save(appCfg); err != nil {
		t.Fatalf("settings.Save: %v", err)
	}

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, svc).
		WithInstanceLock(func() bool { return locked })

	r := chi.NewRouter()
	h.Mount(r)
	return r
}

// postCreatePlayer joue POST /setup/players avec une session portant `role`
// ("" = pas de session du tout).
func postCreatePlayer(r *chi.Mux, role string) *httptest.ResponseRecorder {
	body := `{"gamertag":"AmiDuFoyer","profile_mode":"manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		username := "compte-" + role
		sess := &domain.SessionData{SessionID: "sess-setup", Username: &username, Role: &role}
		req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestSetupHandler_CreatePlayer_GuardMatrix — les 4 combinaisons de l'ADR 0035 D5
// plus le cas anonyme.
func TestSetupHandler_CreatePlayer_GuardMatrix(t *testing.T) {
	cases := []setupGuardCase{
		{
			name: "admin + instance verrouillée → 201",
			role: string(domain.RoleAdmin), locked: true, canSelfProvision: true,
			wantStatus: http.StatusCreated,
		},
		{
			name: "user + instance verrouillée → 403 instance_locked",
			role: string(domain.RoleUser), locked: true, canSelfProvision: true,
			wantStatus: http.StatusForbidden, wantCode: "instance_locked",
		},
		{
			name: "user + auto-provisioning coupé → 403 provisioning_disabled",
			role: string(domain.RoleUser), locked: false, canSelfProvision: false,
			wantStatus: http.StatusForbidden, wantCode: "provisioning_disabled",
		},
		{
			name: "admin + auto-provisioning coupé → 201",
			role: string(domain.RoleAdmin), locked: false, canSelfProvision: false,
			wantStatus: http.StatusCreated,
		},
		{
			name: "anonyme + instance verrouillée → 403 instance_locked",
			role: "", locked: true, canSelfProvision: true,
			wantStatus: http.StatusForbidden, wantCode: "instance_locked",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockProfileService{playerKey: "AmiDuFoyer"}
			r := newGuardRouter(t, tc.locked, tc.canSelfProvision, svc)

			w := postCreatePlayer(r, tc.role)

			if w.Code != tc.wantStatus {
				t.Fatalf("statut %d attendu, reçu %d : %s", tc.wantStatus, w.Code, w.Body.String())
			}
			if tc.wantCode != "" && !bytes.Contains(w.Body.Bytes(), []byte(tc.wantCode)) {
				t.Errorf("code %q attendu dans le corps : %s", tc.wantCode, w.Body.String())
			}
			created := svc.lastReq.Gamertag != ""
			if created != (tc.wantStatus == http.StatusCreated) {
				t.Errorf("CreatePlayer appelé=%v alors que le statut attendu est %d", created, tc.wantStatus)
			}
		})
	}
}

// TestSetupHandler_CreatePlayer_AdminFromUserStore — sans lookup câblé le rôle
// vient de la session ; avec un lookup câblé, c'est le STORE qui tranche (un
// rôle rétrogradé après l'ouverture de session ne donne plus l'exemption).
func TestSetupHandler_CreatePlayer_AdminFromUserStore(t *testing.T) {
	svc := &mockProfileService{playerKey: "AmiDuFoyer"}
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
	}
	sessionStore := session_platform.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, svc).
		WithInstanceLock(func() bool { return true }).
		WithUserLookup(demotedLookup{})

	r := chi.NewRouter()
	h.Mount(r)

	// La session prétend "admin" mais le store rend un compte rétrogradé.
	w := postCreatePlayer(r, string(domain.RoleAdmin))

	if w.Code != http.StatusForbidden {
		t.Fatalf("403 attendu (rôle du store = user), reçu %d : %s", w.Code, w.Body.String())
	}
	if svc.lastReq.Gamertag != "" {
		t.Error("CreatePlayer ne doit pas être appelé pour un compte rétrogradé")
	}
}

// demotedLookup rend un utilisateur au rôle standard quel que soit le nom
// demandé : simule un admin rétrogradé depuis l'ouverture de sa session.
type demotedLookup struct{}

func (demotedLookup) Get(username string) (*domain.User, error) {
	return &domain.User{Username: username, Role: domain.RoleUser}, nil
}

func (demotedLookup) GetByXUID(xuid string) (*domain.User, error) {
	return &domain.User{XUID: xuid, Role: domain.RoleUser}, nil
}
