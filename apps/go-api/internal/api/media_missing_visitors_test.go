package api

// media_missing_visitors_test.go — qui atteint réellement les routes des médias introuvables.
//
// Pile réelle : RequirePlayerOwnership en propriété appliquée, MediaHandler, MediaService réel sur un
// faux dépôt. Mallory est co-membre de groupe de JGtm (famille) ; JGtm est propriétaire.
//
//   - Écritures (POST merge, forget, confirm-bulk, reveal) : le middleware refuse le co-membre (403
//     `authz: accès joueur refusé`) AVANT le handler ; aucune n'est une action du visiteur
//     (player_visitor_actions.go). C'est le test de famille (family_write_routes_test.go) qui tient
//     cette garantie sur toute route walkée ; celui-ci la prouve sur ces quatre routes avec le service.
//   - Lectures (GET missing, candidates) : le middleware laisse passer le co-membre, c'est le service
//     (CanModifyMedia) qui répond 403.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// missingRepoStub : le dépôt d'un joueur JGtm qui a une fiche introuvable et une fiche présente.
type missingRepoStub struct {
	port.MediaRepository
	reads int
}

func (*missingRepoStub) CurrentPlayerSlug() string { return "JGtm" }

func jgtmRef(path string) domain.MediaFileRef {
	return domain.MediaFileRef{
		MediaDeletionTarget: domain.MediaDeletionTarget{ID: 1, OwnerSlug: "JGtm", FilePath: path},
		Kind:                "video",
	}
}

func (r *missingRepoStub) ListMissingMedia(context.Context) ([]domain.MediaMissingItem, error) {
	r.reads++
	return []domain.MediaMissingItem{{FilePath: "JGtm/clip.mp4"}}, nil
}

func (r *missingRepoStub) LoadMissingTarget(_ context.Context, p string) (domain.MediaFileRef, error) {
	r.reads++
	return jgtmRef(p), nil
}

func (r *missingRepoStub) LoadMergeTarget(_ context.Context, p string) (domain.MediaFileRef, error) {
	r.reads++
	return jgtmRef(p), nil
}

func (r *missingRepoStub) ListMergeCandidates(context.Context, domain.MediaFileRef, int) ([]domain.MediaMergeCandidate, error) {
	r.reads++
	return nil, nil
}

func (r *missingRepoStub) LoadMediaForDeletion(_ context.Context, p string) (domain.MediaDeletionTarget, error) {
	r.reads++
	return domain.MediaDeletionTarget{ID: 1, OwnerSlug: "JGtm", FilePath: p}, nil
}

type noRemover struct{}

func (noRemover) RemoveMediaFiles(context.Context, []string) (int, error) { return 0, nil }

type countingRevealer struct{ opened int }

func (c *countingRevealer) Reveal(context.Context, string) error {
	c.opened++
	return nil
}

// missingStack assemble la pile réelle ; withReveal pose ou non la capacité d'ouvrir l'explorateur.
func missingStack(t *testing.T, withReveal bool) (http.Handler, *missingRepoStub, *countingRevealer) {
	t.Helper()
	cfg := requesterConfig(t, "password")
	users := requesterUsers{
		"mallory": {Username: "mallory", Role: domain.RoleUser, XUID: "222"},
		"jgtm":    {Username: "jgtm", Role: domain.RoleUser, XUID: "999"},
	}
	access := middleware.NewPlayerAccess(false, "password", playerOwnershipXUIDResolver(cfg), users,
		func(context.Context) map[string]bool { return map[string]bool{"222": true, "999": true} })

	repo := &missingRepoStub{}
	rev := &countingRevealer{}
	opts := []service.MediaOption{
		service.WithMediaTxOpener(duckdb.NewMediaTxOpener(nil, func() (*dblease.LeasedWriter, error) { return nil, context.Canceled })),
		service.WithMediaFileRemover(noRemover{}),
	}
	if withReveal {
		opts = append(opts, service.WithMediaFileRevealer(rev, func() string { return t.TempDir() }))
	}
	svc := service.NewMediaService(repo, "", opts...)
	factory := func(context.Context, string) (port.MediaService, error) { return svc, nil }
	h := handlers.NewMediaHandler(factory, nil, "").
		WithAuthEnforced(true).
		WithRequester(requesterPlayerSlugResolver(cfg, users)).
		WithUserLookup(users)
	r := chi.NewRouter()
	r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
		r.Use(middleware.RequirePlayerOwnership(access))
		h.Mount(r)
	})
	return r, repo, rev
}

func callAs(r http.Handler, username, method, route, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/players/JGtm"+route, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	current := "JGtm"
	sess := &domain.SessionData{SessionID: "s", Username: &username, CurrentPlayerSlug: &current}
	req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const missingBody = `{"file_path":"JGtm/clip.mp4","target_file_path":"JGtm/new.mp4"}`

var missingWriteRoutes = []string{"/media/missing/merge", "/media/missing/forget", "/media/missing/confirm-bulk", "/media/reveal"}

func TestMissingWriteRoutes_FamilyVisitorIsRefusedBeforeTheHandler(t *testing.T) {
	r, repo, rev := missingStack(t, true)
	for _, route := range missingWriteRoutes {
		w := callAs(r, "mallory", http.MethodPost, route, missingBody)
		if w.Code != http.StatusForbidden {
			t.Errorf("POST %s par un co-membre : statut %d, attendu 403 (%s)", route, w.Code, w.Body.String())
		}
	}
	if repo.reads != 0 || rev.opened != 0 {
		t.Errorf("le service a été atteint (%d lecture, %d ouverture) : le middleware devait refuser", repo.reads, rev.opened)
	}
}

func TestMissingReadRoutes_FamilyVisitorReachesTheServiceWhichRefuses(t *testing.T) {
	r, repo, _ := missingStack(t, true)
	for _, route := range []string{"/media/missing", "/media/missing/candidates?file_path=JGtm/clip.mp4"} {
		w := callAs(r, "mallory", http.MethodGet, route, "")
		if w.Code != http.StatusForbidden || !bytes.Contains(w.Body.Bytes(), []byte("media_missing_forbidden")) {
			t.Errorf("GET %s par un co-membre : statut %d (%s), attendu 403 media_missing_forbidden du service", route, w.Code, w.Body.String())
		}
	}
	if repo.reads > 1 {
		t.Errorf("%d lecture(s) de fiches : seule la fiche introuvable se charge pour juger le propriétaire", repo.reads)
	}
}

func TestMissingRoutes_OwnerReachesEveryRoute(t *testing.T) {
	r, _, rev := missingStack(t, true)
	for _, tc := range []struct{ method, route, body string }{
		{http.MethodGet, "/media/missing", ""},
		{http.MethodGet, "/media/missing/candidates?file_path=JGtm/clip.mp4", ""},
		{http.MethodPost, "/media/missing/confirm-bulk", ""},
		{http.MethodPost, "/media/reveal", `{"file_path":"JGtm/clip.mp4"}`},
	} {
		if w := callAs(r, "jgtm", tc.method, tc.route, tc.body); w.Code != http.StatusOK {
			t.Errorf("%s %s par le propriétaire : statut %d (%s)", tc.method, tc.route, w.Code, w.Body.String())
		}
	}
	if rev.opened != 1 {
		t.Errorf("explorateur ouvert %d fois, attendu 1", rev.opened)
	}
}

// Sans la capacité du déploiement, /media/reveal répond 404 même au propriétaire.
func TestMediaReveal_WithoutCapabilityIs404(t *testing.T) {
	r, repo, _ := missingStack(t, false)
	w := callAs(r, "jgtm", http.MethodPost, "/media/reveal", `{"file_path":"JGtm/clip.mp4"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("statut %d, attendu 404 (%s)", w.Code, w.Body.String())
	}
	if repo.reads != 0 {
		t.Errorf("la fiche a été lue (%d) alors que la capacité est absente", repo.reads)
	}
}
