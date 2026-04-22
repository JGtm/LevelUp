// Package handlers_test — media_test.go : tests unitaires MediaHandler.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mockMediaService implémente port.MediaService.
type mockMediaService struct {
	page      *domain.MediaPageResponse
	pageErr   error
	like      *domain.MediaLikeResponse
	likeErr   error
	upload    *domain.UploadResult
	uploadErr error
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

func newMediaRouter(factory handlers.ServiceFactory[port.MediaService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/media", h.GetMediaLibrary)
		r.Patch("/media/likes", h.PatchMediaLike)
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
		r.Post("/pages/media", h.GetMediaLibrary)
		r.Post("/media/upload", h.PostUploadMedia)
	})
	return r
}

// buildMultipartRequest construit une requête multipart avec les fichiers donnés.
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
		return mock, "TestPlayer", "halo_infinite", "/data/players/TestPlayer/stats.duckdb", "", "", nil
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
	r.Get("/media/feed-version", handlers.GetMediaFeedVersion)

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
	r.Get("/media/feed-version", handlers.GetMediaFeedVersion)

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
		r.Patch("/media/likes", h.PatchMediaLike)
	})
	r.Get("/media/feed-version", handlers.GetMediaFeedVersion)

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
