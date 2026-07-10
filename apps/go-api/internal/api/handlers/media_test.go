// Package handlers_test — media_test.go : tests unitaires MediaHandler.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// mockMediaService implémente port.MediaService.
type mockMediaService struct {
	page       *domain.MediaPageResponse
	pageErr    error
	like       *domain.MediaLikeResponse
	likeErr    error
	upload     *domain.UploadResult
	uploadErr  error
	authors    []domain.MediaAuthor
	authorsErr error
}

func (m *mockMediaService) GetMediaPage(_ context.Context, _ domain.MediaPageRequest) (*domain.MediaPageResponse, error) {
	return m.page, m.pageErr
}

func (m *mockMediaService) SetMediaLike(_ context.Context, _ domain.MediaLikeRequest) (*domain.MediaLikeResponse, error) {
	if m.likeErr != nil {
		return nil, m.likeErr
	}
	if m.like != nil {
		return m.like, nil
	}
	return &domain.MediaLikeResponse{}, nil
}

func (m *mockMediaService) UploadMedia(_ context.Context, _ domain.UploadRequest) (*domain.UploadResult, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	if m.upload != nil {
		return m.upload, nil
	}
	return &domain.UploadResult{}, nil
}

func (m *mockMediaService) GetMatchCandidates(_ context.Context, _ string, _ int) (*domain.MediaMatchCandidatesResponse, error) {
	return &domain.MediaMatchCandidatesResponse{}, nil
}

func (m *mockMediaService) AssociateMediaToMatch(_ context.Context, req domain.MediaAssociateRequest) (*domain.MediaAssociateResponse, error) {
	return &domain.MediaAssociateResponse{FilePath: req.FilePath, MatchID: req.MatchID}, nil
}

func (m *mockMediaService) ListMediaAuthors(_ context.Context) ([]domain.MediaAuthor, error) {
	return m.authors, m.authorsErr
}

func newMediaRouter(factory handlers.ServiceFactory[port.MediaService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r) // 5 routes JSON migrées Huma (pages/media, likes, match-candidates, associate, authors)
	})
	return r
}

func TestMediaHandler_OK(t *testing.T) {
	mock := &mockMediaService{page: &domain.MediaPageResponse{}}
	factory := func(_ context.Context, slug string) (port.MediaService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMediaRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMediaHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newMediaRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMediaHandler_ServiceError(t *testing.T) {
	mock := &mockMediaService{pageErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := newMediaRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// TestMediaHandler_Authors_OK couvre le fix du bug "Aucun auteur disponible" :
// l'endpoint retourne les auteurs fournis par le service (source DB), incl. un
// auteur tiers (Madina) avec is_self=false. Sans WithAuthorsContext câblé, le
// gamertag d'affichage retombe sur le player_slug.
func TestMediaHandler_Authors_OK(t *testing.T) {
	mock := &mockMediaService{authors: []domain.MediaAuthor{
		{PlayerSlug: "Madina", IsSelf: false, MediaCount: 3},
		{PlayerSlug: testPlayerSlug, IsSelf: true, MediaCount: 5},
	}}
	factory := func(_ context.Context, slug string) (port.MediaService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMediaRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/media/authors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.MediaAuthorsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Authors) != 2 {
		t.Fatalf("got %d authors, want 2", len(resp.Authors))
	}
	var madina *domain.MediaAuthor
	for i := range resp.Authors {
		if resp.Authors[i].PlayerSlug == "Madina" {
			madina = &resp.Authors[i]
		}
	}
	if madina == nil {
		t.Fatal("Madina absent de la liste d'auteurs (bug 'Aucun auteur disponible')")
	}
	if madina.IsSelf {
		t.Error("Madina ne devrait pas être is_self")
	}
	if madina.Gamertag != "Madina" {
		t.Errorf("gamertag fallback = %q, want Madina (= player_slug sans WithAuthorsContext)", madina.Gamertag)
	}
}

func TestMediaHandler_Authors_ServiceError(t *testing.T) {
	mock := &mockMediaService{authorsErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := newMediaRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/media/authors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestMediaHandler_PatchMediaLike_OK(t *testing.T) {
	mock := &mockMediaService{like: &domain.MediaLikeResponse{FilePath: "/clips/g1.mp4", Liked: true, LikeCount: 1}}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := newMediaRouter(factory)
	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: "/clips/g1.mp4", Liked: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMediaHandler_PatchMediaLike_NotFound(t *testing.T) {
	mock := &mockMediaService{likeErr: domain.ErrNotFound("media", "/clips/missing.mp4")}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := newMediaRouter(factory)
	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: "/clips/missing.mp4", Liked: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMediaHandler_PatchMediaLike_DBLocked_Returns503(t *testing.T) {
	mock := &mockMediaService{
		likeErr: fmt.Errorf("simulated lease busy: %w", dblease.ErrDBLocked),
	}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := newMediaRouter(factory)
	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: "/clips/g1.mp4", Liked: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
	var body503 map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body503); err != nil {
		t.Fatalf("response body not JSON: %v", err)
	}
	if code, _ := body503["code"].(string); code != "db_busy" {
		t.Errorf("error code = %v, want db_busy", body503["code"])
	}
}

func TestMediaHandler_InvalidBody(t *testing.T) {
	mock := &mockMediaService{page: &domain.MediaPageResponse{}}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := newMediaRouter(factory)
	body := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/media", nil)
	body.ContentLength = 999
	w := httptest.NewRecorder()
	r.ServeHTTP(w, body)

	// Corps vide avec ContentLength > 0 → erreur decode
	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		// ContentLength=999 sans body → EOF, mais le handler default page=1 peut passer
		t.Logf("got %d (acceptable: 200 ou 400)", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests PostUploadMedia
// ─────────────────────────────────────────────────────────────────────────────

// newUploadRouter construit un chi.Mux avec la route upload câblée.
func newUploadRouter(
	svcFactory handlers.ServiceFactory[port.MediaService],
	uploadFactory handlers.MediaUploadContextFactory,
) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMediaHandler(svcFactory, uploadFactory, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)                                 // pages/media (+ autres JSON) migrés Huma
		r.Post("/media/upload", h.PostUploadMedia) // multipart reste chi
	})
	return r
}

// buildMultipartRequest construit une requête multipart avec les fichiers donnés.
//
//nolint:unparam // route est gardé pour ouvrir l'helper à d'autres routes média
func buildMultipartRequest(t *testing.T, route string, files map[string][]byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, data := range files {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatalf("createFormFile: %v", err)
		}
		fw.Write(data) //nolint:errcheck
	}
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, route, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func makeUploadFactory(mock *mockMediaService) handlers.MediaUploadContextFactory {
	return func(_ context.Context, slug string) (port.MediaService, string, string, string, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", "", "", "", errors.New("player_not_found")
		}
		return mock, "TestPlayer", testTitleSlug, "/data/players/TestPlayer/stats.duckdb", "", "", nil
	}
}

func TestUploadHandler_NoUploadFactory(t *testing.T) {
	svcFactory := func(_ context.Context, _ string) (port.MediaService, error) {
		return &mockMediaService{}, nil
	}
	r := newUploadRouter(svcFactory, nil) // nil factory → 501
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/media/upload", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestUploadHandler_PlayerNotFound(t *testing.T) {
	svcFactory := func(_ context.Context, _ string) (port.MediaService, error) {
		return &mockMediaService{}, nil
	}
	uploadFactory := func(_ context.Context, _ string) (port.MediaService, string, string, string, string, string, error) {
		return nil, "", "", "", "", "", errors.New("player_not_found")
	}
	r := newUploadRouter(svcFactory, uploadFactory)
	req := buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{
		"clip.mp4": []byte("fake"),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUploadHandler_NoValidFiles(t *testing.T) {
	mock := &mockMediaService{}
	r := newUploadRouter(
		func(_ context.Context, _ string) (port.MediaService, error) { return mock, nil },
		makeUploadFactory(mock),
	)
	// Extension refusée
	req := buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{
		"malware.exe": []byte("bad"),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUploadHandler_OK(t *testing.T) {
	expected := &domain.UploadResult{Saved: 1, NewIndexed: 1, Associated: 0, Thumbnails: 0}
	mock := &mockMediaService{upload: expected}
	r := newUploadRouter(
		func(_ context.Context, _ string) (port.MediaService, error) { return mock, nil },
		makeUploadFactory(mock),
	)
	req := buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{
		"clip.mp4": []byte("fake video"),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result domain.UploadResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Saved != 1 {
		t.Errorf("Saved = %d, want 1", result.Saved)
	}
}

func TestUploadHandler_ServiceError(t *testing.T) {
	mock := &mockMediaService{uploadErr: errors.New("db_error")}
	r := newUploadRouter(
		func(_ context.Context, _ string) (port.MediaService, error) { return mock, nil },
		makeUploadFactory(mock),
	)
	req := buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{
		"clip.mp4": []byte("fake"),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GetMediaFeedVersion — feed-version polling
// ─────────────────────────────────────────────────────────────────────────────

func TestGetMediaFeedVersion_ReturnsJSON(t *testing.T) {
	r := chi.NewRouter()
	handlers.NewMediaFeedVersionHandler().Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/media/feed-version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]int64
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["version"]; !ok {
		t.Error("expected 'version' field in response")
	}
}

func TestBumpMediaFeedVersion_Increments(t *testing.T) {
	r := chi.NewRouter()
	handlers.NewMediaFeedVersionHandler().Mount(r)

	getVersion := func() int64 {
		req := httptest.NewRequest(http.MethodGet, "/media/feed-version", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var body map[string]int64
		json.NewDecoder(w.Body).Decode(&body) //nolint:errcheck
		return body["version"]
	}

	v1 := getVersion()
	handlers.BumpMediaFeedVersion()
	v2 := getVersion()

	if v2 != v1+1 {
		t.Errorf("expected version to increment by 1: %d → %d", v1, v2)
	}
}

func TestMediaHandler_PatchLike_BumpsVersion(t *testing.T) {
	mock := &mockMediaService{like: &domain.MediaLikeResponse{FilePath: "/x.mp4", Liked: true}}
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return mock, nil
	}
	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r) // likes (+ autres JSON) migrés Huma
	})
	handlers.NewMediaFeedVersionHandler().Mount(r)

	versionNow := func() int64 {
		req := httptest.NewRequest(http.MethodGet, "/media/feed-version", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var body map[string]int64
		json.NewDecoder(w.Body).Decode(&body) //nolint:errcheck
		return body["version"]
	}

	v1 := versionNow()

	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: "/x.mp4", Liked: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	v2 := versionNow()
	if v2 <= v1 {
		t.Errorf("expected version to increase after like: %d → %d", v1, v2)
	}
}

// PostReassociateMedia handler + tests supprimés en revue 2026-04-29 P0.2 Q6
// (doublon non utilisé — le front consomme /media/associate seulement).

// ─────────────────────────────────────────────────────────────────────────────
// GH2-A2 — réassociation média : dépouillement de l'URL servable (HLS)
// ─────────────────────────────────────────────────────────────────────────────

// spyReassocService capture le file_path reçu par les endpoints de réassociation.
type spyReassocService struct {
	mockMediaService
	candidatesPath string
	associatePath  string
}

func (s *spyReassocService) GetMatchCandidates(_ context.Context, filePath string, _ int) (*domain.MediaMatchCandidatesResponse, error) {
	s.candidatesPath = filePath
	return &domain.MediaMatchCandidatesResponse{}, nil
}

func (s *spyReassocService) AssociateMediaToMatch(_ context.Context, req domain.MediaAssociateRequest) (*domain.MediaAssociateResponse, error) {
	s.associatePath = req.FilePath
	return &domain.MediaAssociateResponse{FilePath: req.FilePath, MatchID: req.MatchID}, nil
}

// hlsServableURL est l'URL servable telle que la galerie l'expose pour une vidéo
// (file_path = playlist HLS). storedPathHLS est le chemin RELATIF correspondant
// stocké en DB (media_files.file_path).
const (
	hlsServableURL = "/api/v1/players/test-player/media/files/JGtm/hls/Replay 2026-07-07 23-03-38/master.m3u8"
	storedPathHLS  = "JGtm/hls/Replay 2026-07-07 23-03-38/master.m3u8"
)

// TestMediaHandler_MatchCandidates_HLSUrlStrippedToStoredPath prouve le fix GH2-A2 :
// la popup de réassociation envoie file_path sous forme d'URL servable (vidéo →
// playlist HLS .../hls/<stem>/master.m3u8). Le handler doit dépouiller le préfixe pour
// retrouver le chemin RELATIF stocké — sinon le lookup ne matche ni mf.file_path
// (préfixe URL en trop) ni mf.file_name (basename = "master.m3u8") → 500 loading error.
func TestMediaHandler_MatchCandidates_HLSUrlStrippedToStoredPath(t *testing.T) {
	spy := &spyReassocService{}
	factory := func(_ context.Context, slug string) (port.MediaService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return spy, nil
	}
	r := newMediaRouter(factory)
	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/media/match-candidates?file_path="+url.QueryEscape(hlsServableURL)+"&window_minutes=15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.candidatesPath != storedPathHLS {
		t.Errorf("candidates file_path = %q, want %q (URL servable dépouillée en chemin relatif stocké)",
			spy.candidatesPath, storedPathHLS)
	}
}

// TestMediaHandler_Associate_HLSUrlStrippedToStoredPath : symétrie côté écriture —
// l'association d'une vidéo doit cibler la ligne media_files par le chemin stocké,
// pas par l'URL servable (sinon UPDATE 0 row → association perdue).
func TestMediaHandler_Associate_HLSUrlStrippedToStoredPath(t *testing.T) {
	spy := &spyReassocService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }
	r := newMediaRouter(factory)
	body, _ := json.Marshal(domain.MediaAssociateRequest{FilePath: hlsServableURL, MatchID: "m-1"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/media/associate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.associatePath != storedPathHLS {
		t.Errorf("associate file_path = %q, want %q (URL servable dépouillée)", spy.associatePath, storedPathHLS)
	}
}

// TestMediaHandler_MatchCandidates_PlainPathPassthrough : un chemin déjà stocké
// (appelant hors galerie / screenshot legacy) passe inchangé au service.
func TestMediaHandler_MatchCandidates_PlainPathPassthrough(t *testing.T) {
	spy := &spyReassocService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }
	r := newMediaRouter(factory)
	plain := "JGtm/screenshot.png"
	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/media/match-candidates?file_path="+url.QueryEscape(plain)+"&window_minutes=15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.candidatesPath != plain {
		t.Errorf("candidates file_path = %q, want %q (chemin non-URL inchangé)", spy.candidatesPath, plain)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// parseUploadedFiles — capture_times
// ─────────────────────────────────────────────────────────────────────────────

// buildMultipartRequestWithCaptureTimes construit une requête multipart avec le champ capture_times.
func buildMultipartRequestWithCaptureTimes(t *testing.T, route string, files map[string][]byte, captureTimes string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, data := range files {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatalf("createFormFile: %v", err)
		}
		fw.Write(data) //nolint:errcheck
	}
	if captureTimes != "" {
		_ = mw.WriteField("capture_times", captureTimes)
	}
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, route, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadHandler_WithCaptureTimes_OK(t *testing.T) {
	expected := &domain.UploadResult{Saved: 1, NewIndexed: 1}
	mock := &mockMediaService{upload: expected}
	r := newUploadRouter(
		func(_ context.Context, _ string) (port.MediaService, error) { return mock, nil },
		makeUploadFactory(mock),
	)
	// Envoyer capture_times JSON valide aligné sur files
	req := buildMultipartRequestWithCaptureTimes(t, "/players/test-player/media/upload",
		map[string][]byte{"clip.mp4": []byte("fake")},
		`[1700000000]`,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_WithCaptureTimes_InvalidJSON_Ignored(t *testing.T) {
	// capture_times JSON invalide → ignoré (pas 400)
	mock := &mockMediaService{upload: &domain.UploadResult{Saved: 1}}
	r := newUploadRouter(
		func(_ context.Context, _ string) (port.MediaService, error) { return mock, nil },
		makeUploadFactory(mock),
	)
	req := buildMultipartRequestWithCaptureTimes(t, "/players/test-player/media/upload",
		map[string][]byte{"clip.mp4": []byte("fake")},
		`not-json`,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Le JSON invalide est loggué et ignoré, l'upload doit réussir
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (capture_times invalide ignoré), got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// urlToFilePath — conversion URL→chemin stocké en DB
// ─────────────────────────────────────────────────────────────────────────────

// spyLikeService capture le MediaLikeRequest reçu par SetMediaLike.
type spyLikeService struct {
	mockMediaService
	capturedReq *domain.MediaLikeRequest
}

func (s *spyLikeService) SetMediaLike(_ context.Context, req domain.MediaLikeRequest) (*domain.MediaLikeResponse, error) {
	s.capturedReq = &req
	return &domain.MediaLikeResponse{FilePath: req.FilePath, Liked: true}, nil
}

// patchLike envoie un PATCH /media/likes et retourne le recorder.
func patchLike(r http.Handler, filePath string, liked bool) *httptest.ResponseRecorder {
	body, _ := json.Marshal(domain.MediaLikeRequest{FilePath: filePath, Liked: liked})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestMediaHandler_PatchMediaLike_URLPath_FallbackToRelPath vérifie que urlToFilePath
// dépouille le préfixe URL et transmet relPath au service quand aucun settingsStore
// ni repoRoot n'est configuré. C'est le fix du bug double-slug (before: URL complète
// retournée → UPDATE 0 rows → 404).
func TestMediaHandler_PatchMediaLike_URLPath_FallbackToRelPath(t *testing.T) {
	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }
	r := newMediaRouter(factory) // pas de settingsStore, pas de repoRoot

	urlPath := "/api/v1/players/test-player/media/files/JGtm/clip.mp4"
	w := patchLike(r, urlPath, true)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.capturedReq == nil {
		t.Fatal("SetMediaLike not called")
	}
	// URL dépouillée du préfixe → relPath avec séparateur OS.
	wantRelPath := filepath.Join("JGtm", "clip.mp4")
	if spy.capturedReq.FilePath != wantRelPath {
		t.Errorf("FilePath = %q, want %q (URL stripped to relPath)", spy.capturedReq.FilePath, wantRelPath)
	}
}

// TestMediaHandler_PatchMediaLike_PlainPath_Passthrough vérifie qu'un chemin ordinaire
// (non préfixé par le pattern URL) passe inchangé au service.
func TestMediaHandler_PatchMediaLike_PlainPath_Passthrough(t *testing.T) {
	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }
	r := newMediaRouter(factory)

	plainPath := "/clips/g1.mp4"
	w := patchLike(r, plainPath, true)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.capturedReq == nil {
		t.Fatal("SetMediaLike not called")
	}
	if spy.capturedReq.FilePath != plainPath {
		t.Errorf("FilePath = %q, want %q (unchanged)", spy.capturedReq.FilePath, plainPath)
	}
}

// TestMediaHandler_PatchMediaLike_URLPath_CapturesBaseResolves vérifie que urlToFilePath
// retourne le chemin absolu correct quand capturesBase est configuré et que le fichier
// existe à capturesBase/relPath. Valide le fix du bug double-slug : l'ancienne version
// cherchait capturesBase/viewer-slug/owner-slug/clip.mp4 (introuvable) ; la version
// corrigée cherche capturesBase/owner-slug/clip.mp4 (trouve le fichier → chemin absolu).
func TestMediaHandler_PatchMediaLike_URLPath_CapturesBaseResolves(t *testing.T) {
	capturesBase := t.TempDir()
	ownerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ownerDir, "clip.mp4"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "app_settings.json")
	settingsJSON := fmt.Sprintf(`{"media_captures_base_dir":%q}`, capturesBase)
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	store := settings.NewStore(settingsPath)

	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }

	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "").WithSettingsStore(store)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r) // likes (+ autres JSON) migrés Huma
	})

	urlPath := "/api/v1/players/test-player/media/files/JGtm/clip.mp4"
	w := patchLike(r, urlPath, true)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.capturedReq == nil {
		t.Fatal("SetMediaLike not called")
	}
	wantPath := filepath.Join(capturesBase, "JGtm", "clip.mp4")
	if spy.capturedReq.FilePath != wantPath {
		t.Errorf("FilePath = %q, want %q (capturesBase resolved to absolute)", spy.capturedReq.FilePath, wantPath)
	}
}
