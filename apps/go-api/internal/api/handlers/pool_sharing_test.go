package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

type fakeUserLookup struct{ byXUID map[string]*domain.User }

func (f fakeUserLookup) Get(string) (*domain.User, error) { return nil, auth.ErrUserTokensNotFound }
func (f fakeUserLookup) GetByXUID(x string) (*domain.User, error) {
	if u, ok := f.byXUID[x]; ok {
		return u, nil
	}
	return nil, auth.ErrUserTokensNotFound
}

type recordingApplier struct {
	calls []struct {
		gamertag string
		shared   bool
	}
}

func (a *recordingApplier) SetPoolShared(gamertag string, shared bool) bool {
	a.calls = append(a.calls, struct {
		gamertag string
		shared   bool
	}{gamertag, shared})
	return true
}

// putPoolSharing monte le handler derrière une session portant `linked` (nil = anonyme).
func putPoolSharing(t *testing.T, store *auth.MultiUserTokenStore, applier handlers.PoolSharingApplier,
	linked *domain.HaloIdentity, body string) *httptest.ResponseRecorder {
	t.Helper()
	users := fakeUserLookup{byXUID: map[string]*domain.User{}}
	h := handlers.NewPoolSharingHandler(store, applier, users)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sess := &domain.SessionData{LinkedHaloIdentity: linked}
			next.ServeHTTP(w, req.WithContext(middleware.InjectSession(req.Context(), sess)))
		})
	})
	h.Mount(r)

	req := httptest.NewRequest(http.MethodPut, "/me/pool-sharing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPoolSharingHandler_GrantThenWithdraw(t *testing.T) {
	store := auth.NewMultiUserTokenStore(t.TempDir())
	if err := store.Upsert(&auth.UserTokens{XUID: "111", Gamertag: "Alice", OAuthRefreshToken: "rt"}); err != nil {
		t.Fatal(err)
	}
	applier := &recordingApplier{}
	me := &domain.HaloIdentity{XUID: "111", Gamertag: "Alice"}

	w := putPoolSharing(t, store, applier, me, `{"granted":true}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"granted"`) {
		t.Fatalf("grant: %d %s", w.Code, w.Body.String())
	}
	if got := store.PoolSharingState("111"); got != domain.PoolSharingGranted {
		t.Fatalf("état après accord = %q", got)
	}

	w = putPoolSharing(t, store, applier, me, `{"granted":false}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"declined"`) {
		t.Fatalf("withdraw: %d %s", w.Code, w.Body.String())
	}
	if got := store.PoolSharingState("111"); got != domain.PoolSharingDeclined {
		t.Fatalf("état après retrait = %q", got)
	}

	if len(applier.calls) != 2 || !applier.calls[0].shared || applier.calls[1].shared || applier.calls[1].gamertag != "Alice" {
		t.Fatalf("le pool doit recevoir chaque décision immédiatement: %+v", applier.calls)
	}
}

func TestPoolSharingHandler_RefusesWithoutLinkedIdentity(t *testing.T) {
	store := auth.NewMultiUserTokenStore(t.TempDir())
	applier := &recordingApplier{}

	w := putPoolSharing(t, store, applier, nil, `{"granted":true}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d %s", w.Code, w.Body.String())
	}
	if len(applier.calls) != 0 {
		t.Fatal("aucune décision ne doit atteindre le pool sans identité")
	}
}

func TestPoolSharingHandler_NoTokenIsConflict(t *testing.T) {
	store := auth.NewMultiUserTokenStore(t.TempDir())
	applier := &recordingApplier{}
	me := &domain.HaloIdentity{XUID: "222", Gamertag: "Bob"}

	w := putPoolSharing(t, store, applier, me, `{"granted":true}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d %s", w.Code, w.Body.String())
	}
	if len(applier.calls) != 0 {
		t.Fatal("rien n'a été persisté, le pool ne doit pas changer")
	}
}
