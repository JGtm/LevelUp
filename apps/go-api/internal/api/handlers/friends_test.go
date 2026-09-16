// Package handlers_test — friends_test.go : la liste d'amis d'un profil joueur.
//
// Le montage reproduit celui de la production : le groupe `/players/{player_slug}`
// porte `RequirePlayerOwnership` (couche A, ADR 0029) et le handler pose sa propre
// garde d'écriture (D4). Les deux portes sont testées ENSEMBLE parce qu'elles
// répondent à deux questions distinctes — « ce profil t'est-il accessible ? » et
// « peux-tu l'écrire ? » — et que c'est leur composition qui décide.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/friendstore"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

// Le parc de test : deux profils joueurs, trois comptes (propriétaire,
// co-membre de groupe, étranger) plus un admin.
const (
	ownerSlug = "Alice"
	ownerGT   = "Alice"
	ownerXUID = "xuid-alice"
	otherSlug = "Bob"
	otherXUID = "xuid-bob"
)

type friendsFixture struct {
	router  *chi.Mux
	session *session.Store
	users   *userstore.Store
	friends *friendstore.FriendStore
}

// newFriendsFixture monte le handler sous le MÊME chokepoint qu'en production.
// `authMode` vaut "xbox" (propriété appliquée) ou "none" (désactivée).
// friendsOption ajuste le handler de la fixture (ex. brancher un recomputer).
type friendsOption func(*handlers.FriendsHandler)

func newFriendsFixture(t *testing.T, authMode string, opts ...friendsOption) *friendsFixture {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	friends := friendstore.NewFriendStore(filepath.Join(dir, "player_friends.json"))

	xuidBySlug := map[string]string{ownerSlug: ownerXUID, otherSlug: otherXUID}
	gtBySlug := map[string]string{ownerSlug: ownerGT, otherSlug: "Bob"}
	resolveXUID := func(_ context.Context, slug string) (string, bool) {
		x, ok := xuidBySlug[slug]
		return x, ok
	}
	resolveGT := func(_ context.Context, slug string) (string, bool) {
		g, ok := gtBySlug[slug]
		return g, ok
	}
	// Cercle de groupe : Bob est co-membre d'Alice (il VOIT son profil).
	family := func(ctx context.Context) map[string]bool {
		sess := middleware.GetSession(ctx)
		if sess == nil || sess.LinkedHaloIdentity == nil {
			return nil
		}
		if sess.LinkedHaloIdentity.XUID == otherXUID {
			return map[string]bool{ownerXUID: true, otherXUID: true}
		}
		return nil
	}

	h := handlers.NewFriendsHandler(friends, users,
		handlers.PlayerXUIDResolver(resolveXUID), resolveGT, false, authMode)
	for _, o := range opts {
		o(h)
	}

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Use(middleware.RequirePlayerOwnership(false, authMode,
			middleware.PlayerXUIDResolver(resolveXUID), users, family))
		h.Mount(r)
	})
	return &friendsFixture{router: r, session: sessStore, users: users, friends: friends}
}

// linkedUser crée un compte lié à un xuid et retourne son cookie de session.
func (f *friendsFixture) linkedUser(t *testing.T, username, gamertag, xuid string, role domain.UserRole) *http.Cookie {
	t.Helper()
	if _, err := f.users.CreateFromXbox(gamertag, xuid); err != nil {
		t.Fatalf("création compte %s : %v", username, err)
	}
	if role == domain.RoleAdmin {
		u, err := f.users.GetByXUID(xuid)
		if err != nil {
			t.Fatalf("relecture compte %s : %v", username, err)
		}
		if err := f.users.SetRole(u.Username, domain.RoleAdmin); err != nil {
			t.Fatalf("passage admin %s : %v", username, err)
		}
	}
	sess := f.session.New()
	sess.LinkedHaloIdentity = &domain.HaloIdentity{XUID: xuid, Gamertag: gamertag}
	if err := f.session.Save(sess); err != nil {
		t.Fatalf("sauvegarde session : %v", err)
	}
	return &http.Cookie{Name: session.CookieName, Value: f.session.SignCookie(sess.SessionID)}
}

func (f *friendsFixture) do(t *testing.T, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func decodeFriends(t *testing.T, w *httptest.ResponseRecorder) domain.PlayerFriends {
	t.Helper()
	var out domain.PlayerFriends
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("décodage réponse : %v (corps = %s)", err, w.Body.String())
	}
	return out
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body.Code
}

// ─── Propriétaire ────────────────────────────────────────────────────────────

func TestFriends_Owner_GetEmptyThenPut(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	w := f.do(t, http.MethodGet, "/players/Alice/friends", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("GET = %d, corps = %s", w.Code, w.Body.String())
	}
	got := decodeFriends(t, w)
	if got.XUID != ownerXUID || len(got.Gamertags) != 0 || !got.CanEdit {
		t.Errorf("GET = %+v ; want xuid=%s, liste vide, can_edit=true", got, ownerXUID)
	}

	w = f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Charlie","Delta"]}`, cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT = %d, corps = %s", w.Code, w.Body.String())
	}
	put := decodeFriends(t, w)
	if len(put.Gamertags) != 2 || put.Gamertags[0] != "Charlie" || !put.CanEdit {
		t.Errorf("PUT = %+v ; want [Charlie Delta], can_edit=true", put)
	}
	if put.UpdatedAt == "" {
		t.Error("PUT : updated_at vide")
	}
	// La liste est bien persistée pour le joueur, pas pour l'instance.
	stored, _ := f.friends.Get(ownerXUID)
	if len(stored) != 2 {
		t.Errorf("store = %v, want 2 entrées", stored)
	}
	if other, _ := f.friends.Get(otherXUID); len(other) != 0 {
		t.Errorf("la liste a fuité vers l'autre joueur : %v", other)
	}
}

func TestFriends_Put_NormalisesAndExcludesSelf(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	w := f.do(t, http.MethodPut, "/players/Alice/friends",
		`{"gamertags":["  Charlie ","CHARLIE","alice","Delta"]}`, cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT = %d, corps = %s", w.Code, w.Body.String())
	}
	got := decodeFriends(t, w)
	if len(got.Gamertags) != 2 || got.Gamertags[0] != "Charlie" || got.Gamertags[1] != "Delta" {
		t.Errorf("PUT = %v ; want [Charlie Delta] (trim, dédoublonnage, soi exclu)", got.Gamertags)
	}
}

func TestFriends_Put_ReplacesWholeList(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	if w := f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Charlie"]}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("1er PUT = %d", w.Code)
	}
	w := f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Delta"]}`, cookie)
	got := decodeFriends(t, w)
	if len(got.Gamertags) != 1 || got.Gamertags[0] != "Delta" {
		t.Errorf("PUT = %v ; want [Delta] (remplacement complet)", got.Gamertags)
	}
}

// ─── Refus ───────────────────────────────────────────────────────────────────

func TestFriends_Stranger_403PlayerForbidden(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "eve", "Eve", "xuid-eve", domain.RoleUser)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		body := ""
		if method == http.MethodPut {
			body = `{"gamertags":["Charlie"]}`
		}
		w := f.do(t, method, "/players/Alice/friends", body, cookie)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s étranger = %d, want 403 (corps = %s)", method, w.Code, w.Body.String())
		}
		if code := errorCode(t, w); code != "player_forbidden" {
			t.Errorf("%s étranger : code = %q, want player_forbidden", method, code)
		}
	}
}

func TestFriends_GroupCoMember_ReadsButCannotWrite(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	owner := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)
	if w := f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Charlie"]}`, owner); w.Code != http.StatusOK {
		t.Fatalf("PUT propriétaire = %d", w.Code)
	}
	coMember := f.linkedUser(t, "bob", "Bob", otherXUID, domain.RoleUser)

	w := f.do(t, http.MethodGet, "/players/Alice/friends", "", coMember)
	if w.Code != http.StatusOK {
		t.Fatalf("GET co-membre = %d, want 200 (corps = %s)", w.Code, w.Body.String())
	}
	got := decodeFriends(t, w)
	if len(got.Gamertags) != 1 {
		t.Errorf("GET co-membre = %v, want la liste d'Alice", got.Gamertags)
	}
	if got.CanEdit {
		t.Error("can_edit=true pour un co-membre de groupe : il lit, il n'écrit pas (D4)")
	}

	w = f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Delta"]}`, coMember)
	if w.Code != http.StatusForbidden {
		t.Fatalf("PUT co-membre = %d, want 403 (corps = %s)", w.Code, w.Body.String())
	}
	if code := errorCode(t, w); code != "friends_forbidden" {
		t.Errorf("PUT co-membre : code = %q, want friends_forbidden", code)
	}
	// L'écriture refusée n'a rien changé.
	stored, _ := f.friends.Get(ownerXUID)
	if len(stored) != 1 || stored[0] != "Charlie" {
		t.Errorf("liste après refus = %v, want [Charlie]", stored)
	}
}

func TestFriends_Admin_ReadsAndWritesAnyProfile(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "root", "Root", "xuid-root", domain.RoleAdmin)

	w := f.do(t, http.MethodGet, "/players/Alice/friends", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("GET admin = %d, corps = %s", w.Code, w.Body.String())
	}
	if !decodeFriends(t, w).CanEdit {
		t.Error("can_edit=false pour un admin (D4)")
	}
	w = f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Charlie"]}`, cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT admin = %d, corps = %s", w.Code, w.Body.String())
	}
}

// ─── Validation ──────────────────────────────────────────────────────────────

func TestFriends_Put_RejectsOversizedList(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	var sb strings.Builder
	sb.WriteString(`{"gamertags":[`)
	for i := 0; i <= domain.MaxFriendGamertags; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"Ami`)
		sb.WriteString(string(rune('A' + i%26)))
		sb.WriteString(string(rune('a' + i/26)))
		sb.WriteString(`"`)
	}
	sb.WriteString(`]}`)

	w := f.do(t, http.MethodPut, "/players/Alice/friends", sb.String(), cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PUT 51 entrées = %d, want 400 (corps = %s)", w.Code, w.Body.String())
	}
	if code := errorCode(t, w); code != "invalid_friends_too_many" {
		t.Errorf("code = %q, want invalid_friends_too_many", code)
	}
}

func TestFriends_Put_RejectsTooLongGamertag(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	long := strings.Repeat("x", domain.MaxFriendGamertagLen+1)
	w := f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["`+long+`"]}`, cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PUT gamertag trop long = %d, want 400 (corps = %s)", w.Code, w.Body.String())
	}
	// Le motif de refus est PORTÉ PAR LE CODE : l'API n'envoie aucune phrase, le
	// texte lisible vit côté web (features/friends/errors.ts).
	if code := errorCode(t, w); code != "invalid_friends_gamertag_too_long" {
		t.Errorf("code = %q, want invalid_friends_gamertag_too_long", code)
	}
}

func TestFriends_Put_EmptyGamertagsAreDropped(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	w := f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["","   ","Charlie"]}`, cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT = %d, corps = %s", w.Code, w.Body.String())
	}
	got := decodeFriends(t, w)
	if len(got.Gamertags) != 1 || got.Gamertags[0] != "Charlie" {
		t.Errorf("PUT = %v, want [Charlie]", got.Gamertags)
	}
}

func TestFriends_Put_InvalidJSON400(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	w := f.do(t, http.MethodPut, "/players/Alice/friends", `{ pas du json`, cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PUT corps invalide = %d, want 400 (corps = %s)", w.Code, w.Body.String())
	}
}

// TestFriends_UnknownSlug : un slug inconnu est arrêté par le middleware AVANT
// le handler, avec un 403 uniforme — fail-closed délibéré (S7 / audit A1-m1 :
// un 404 distinct servirait d'oracle d'existence à un utilisateur authentifié).
// Seul un admin traverse et reçoit le 404 player_not_found du handler.
func TestFriends_UnknownSlug(t *testing.T) {
	f := newFriendsFixture(t, "xbox")
	user := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	w := f.do(t, http.MethodGet, "/players/Inconnu/friends", "", user)
	if w.Code != http.StatusForbidden {
		t.Fatalf("GET slug inconnu (utilisateur) = %d, want 403 (corps = %s)", w.Code, w.Body.String())
	}
	if code := errorCode(t, w); code != "player_forbidden" {
		t.Errorf("code = %q, want player_forbidden", code)
	}

	admin := f.linkedUser(t, "root", "Root", "xuid-root", domain.RoleAdmin)
	w = f.do(t, http.MethodGet, "/players/Inconnu/friends", "", admin)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET slug inconnu (admin) = %d, want 404 (corps = %s)", w.Code, w.Body.String())
	}
	if code := errorCode(t, w); code != "player_not_found" {
		t.Errorf("code admin = %q, want player_not_found", code)
	}
}

// ─── Propriété non appliquée ─────────────────────────────────────────────────

func TestFriends_AuthModeNone_WriteIsFree(t *testing.T) {
	f := newFriendsFixture(t, "none")

	w := f.do(t, http.MethodGet, "/players/Alice/friends", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET sans session en auth_mode=none = %d, corps = %s", w.Code, w.Body.String())
	}
	if !decodeFriends(t, w).CanEdit {
		t.Error("can_edit=false alors que la propriété n'est pas appliquée")
	}
	w = f.do(t, http.MethodPut, "/players/Alice/friends", `{"gamertags":["Charlie"]}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT sans session en auth_mode=none = %d, corps = %s", w.Code, w.Body.String())
	}
}

// recorderRecomputer capture chaque appel de RecomputeForPlayer (xuid) sur un
// canal, pour synchroniser le test avec la goroutine du handler.
type recorderRecomputer struct{ calls chan string }

func (r *recorderRecomputer) RecomputeForPlayer(_ context.Context, xuid string) (int64, error) {
	r.calls <- xuid
	return 0, nil
}

func awaitRecompute(t *testing.T, rec *recorderRecomputer) string {
	t.Helper()
	select {
	case x := <-rec.calls:
		return x
	case <-time.After(2 * time.Second):
		t.Fatal("RecomputeForPlayer non appelé dans les 2 s")
		return ""
	}
}

// Constat P0 de revue (2026-09-16) : retirer le DERNIER ami doit déclencher le
// recalcul is_with_friends (démotion convergente) exactement comme un ajout. Le
// handler doit appeler le recomputer avec le xuid du profil pour un PUT non vide
// PUIS pour un PUT vide ; une liste inchangée ne déclenche rien.
func TestFriends_Put_TriggersRecomputeIncludingEmptyList(t *testing.T) {
	rec := &recorderRecomputer{calls: make(chan string, 4)}
	f := newFriendsFixture(t, "xbox", func(h *handlers.FriendsHandler) { h.WithRecomputer(rec) })
	cookie := f.linkedUser(t, "alice", ownerGT, ownerXUID, domain.RoleUser)

	if w := f.do(t, http.MethodPut, "/players/"+ownerSlug+"/friends", `{"gamertags":["Bob"]}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("PUT [Bob] = %d (corps = %s)", w.Code, w.Body.String())
	}
	if got := awaitRecompute(t, rec); got != ownerXUID {
		t.Fatalf("recompute xuid = %q, want %q", got, ownerXUID)
	}

	if w := f.do(t, http.MethodPut, "/players/"+ownerSlug+"/friends", `{"gamertags":[]}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("PUT [] = %d (corps = %s)", w.Code, w.Body.String())
	}
	if got := awaitRecompute(t, rec); got != ownerXUID {
		t.Fatalf("recompute (liste vide) xuid = %q, want %q", got, ownerXUID)
	}

	// Liste inchangée : aucun recalcul.
	if w := f.do(t, http.MethodPut, "/players/"+ownerSlug+"/friends", `{"gamertags":[]}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("PUT [] bis = %d", w.Code)
	}
	select {
	case x := <-rec.calls:
		t.Fatalf("recalcul inattendu (xuid %q) pour une liste inchangée", x)
	case <-time.After(300 * time.Millisecond):
	}
}
