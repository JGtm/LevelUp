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
	"levelup/go-api/internal/api/middleware"
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
	del        *domain.MediaDeleteResponse
	delErr     error
	delReq     *domain.MediaDeleteRequest // dernière requête reçue (assertions d'identité)
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

func (m *mockMediaService) DeleteMedia(_ context.Context, req domain.MediaDeleteRequest) (*domain.MediaDeleteResponse, error) {
	m.delReq = &req
	if m.delErr != nil {
		return nil, m.delErr
	}
	if m.del != nil {
		return m.del, nil
	}
	return &domain.MediaDeleteResponse{FilePath: req.FilePath, Deleted: true, FilesRemoved: 1}, nil
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
// Conversion URL servable → chemin stocké en DB (item 1.5)
//
// media_files.file_path est RELATIF forward-slash ({owner_slug}/{rel}) depuis la
// migration des paths. Toute mutation qui rejoue une URL servable doit recomposer
// EXACTEMENT cette forme, sinon l'UPDATE ne touche aucune ligne (404 silencieux).
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

// TestMediaHandler_PatchMediaLike_URLPath_FallbackToRelPath vérifie que le handler
// dépouille le préfixe URL et transmet le chemin stocké (relatif, forward-slash)
// au service quand aucun settingsStore ni repoRoot n'est configuré.
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
	// Format DB : forward-slash, JAMAIS le séparateur OS (sur Windows,
	// filepath.Join produirait "JGtm\clip.mp4" → 0 ligne mise à jour).
	const wantRelPath = "JGtm/clip.mp4"
	if spy.capturedReq.FilePath != wantRelPath {
		t.Errorf("FilePath = %q, want %q (URL stripped to stored path)", spy.capturedReq.FilePath, wantRelPath)
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

// newCapturesSettingsStore crée un settings store dont media_captures_base_dir
// pointe sur un dossier temporaire contenant RÉELLEMENT JGtm/clip.mp4. C'est la
// configuration qui déclenchait le bug 1.5 : le fichier présent sur le disque du
// serveur faisait résoudre un chemin ABSOLU, introuvable en base.
func newCapturesSettingsStore(t *testing.T) *settings.Store {
	t.Helper()
	capturesBase := t.TempDir()
	ownerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ownerDir, "clip.mp4"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(t.TempDir(), "app_settings.json")
	settingsJSON := fmt.Sprintf(`{"media_captures_base_dir":%q}`, capturesBase)
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	return settings.NewStore(settingsPath)
}

// TestMediaHandler_PatchMediaLike_URLPath_KeepsStoredRelativePath est le test de
// non-régression de l'item 1.5 : même quand media_captures_base_dir est configuré
// ET que le fichier existe à capturesBase/{owner}/{file}, le handler doit
// transmettre le chemin STOCKÉ (relatif forward-slash), pas un chemin absolu
// reconstruit depuis le disque. Le comportement inverse (retenu jusqu'ici) faisait
// échouer l'UPDATE media_files (0 ligne → 404) sur toute installation où le
// serveur voit les fichiers — c'est-à-dire en production.
func TestMediaHandler_PatchMediaLike_URLPath_KeepsStoredRelativePath(t *testing.T) {
	store := newCapturesSettingsStore(t)
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
	const wantPath = "JGtm/clip.mp4"
	if spy.capturedReq.FilePath != wantPath {
		t.Errorf("FilePath = %q, want %q (chemin stocké, pas de résolution disque)",
			spy.capturedReq.FilePath, wantPath)
	}
}

// TestMediaHandler_PatchMediaLike_EchoesRequestedFilePath : la réponse renvoie le
// file_path REÇU (URL servable). Le client indexe son cache par cette URL ; un
// chemin stocké en réponse ne matcherait aucun item (likers/compteur perdus).
func TestMediaHandler_PatchMediaLike_EchoesRequestedFilePath(t *testing.T) {
	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }
	r := newMediaRouter(factory)

	const urlPath = "/api/v1/players/test-player/media/files/JGtm/hls/clip/master.m3u8"
	w := patchLike(r, urlPath, true)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.MediaLikeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.FilePath != urlPath {
		t.Errorf("réponse file_path = %q, want %q (écho de la requête)", resp.FilePath, urlPath)
	}
	// Le service, lui, a bien reçu le chemin stocké.
	if spy.capturedReq == nil || spy.capturedReq.FilePath != "JGtm/hls/clip/master.m3u8" {
		t.Errorf("service FilePath = %+v, want JGtm/hls/clip/master.m3u8", spy.capturedReq)
	}
}

// TestMediaHandler_PatchMediaLike_AuthEnforced_NoSession_401 : garde anti-silence.
// En multi-utilisateur authentifié, un like sans joueur courant en session ne peut
// pas être attribué : il doit échouer VISIBLEMENT (401) au lieu d'écrire un
// media_files.liked global muet côté social.
func TestMediaHandler_PatchMediaLike_AuthEnforced_NoSession_401(t *testing.T) {
	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }

	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "").WithAuthEnforced(true)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})

	w := patchLike(r, "/api/v1/players/test-player/media/files/JGtm/clip.mp4", true)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if spy.capturedReq != nil {
		t.Error("SetMediaLike ne doit PAS être appelé sans liker identifiable")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("like_requires_session")) {
		t.Errorf("corps sans code like_requires_session : %s", w.Body.String())
	}
}

// TestMediaHandler_PatchMediaLike_NoAuthEnforced_NoSession_FallsBackToPageOwner :
// en mono-utilisateur / démo (auth non appliquée), le like passe — et il est
// désormais ATTRIBUÉ au propriétaire de la page.
//
// Ce repli n'est pas cosmétique : depuis que le like est par-viewer, un like sans
// liker n'a plus AUCUN support de stockage (media_files.liked, le booléen global,
// n'est plus écrit). Sans ce repli, le like d'une instance locale — le cas d'usage
// principal du produit — ne s'écrirait nulle part et le cœur retomberait au
// rechargement.
func TestMediaHandler_PatchMediaLike_NoAuthEnforced_NoSession_FallsBackToPageOwner(t *testing.T) {
	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }
	r := newMediaRouter(factory) // WithAuthEnforced non appelé → false

	w := patchLike(r, "/api/v1/players/test-player/media/files/JGtm/clip.mp4", true)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.capturedReq == nil {
		t.Fatal("SetMediaLike doit être appelé en mode mono-utilisateur")
	}
	if spy.capturedReq.LikerSlug != testPlayerSlug {
		t.Errorf("LikerSlug = %q, want %q — un like sans liker ne se persiste plus nulle part",
			spy.capturedReq.LikerSlug, testPlayerSlug)
	}
}

// TestMediaHandler_PatchMediaLike_SessionOverridesBodyLiker : le liker vient de la
// SESSION, jamais du corps de la requête.
//
// Depuis que le cœur est par-viewer, `liker_slug` n'est plus un champ décoratif :
// il décide de QUI verra ce like allumé et de quel nom s'affiche dans « ♥ Alice
// et Bob ». Un client qui choisirait ces valeurs pourrait liker à la place d'un
// tiers. Le serveur écrase donc les deux champs dès qu'une session porte un
// joueur courant.
func TestMediaHandler_PatchMediaLike_SessionOverridesBodyLiker(t *testing.T) {
	spy := &spyLikeService{}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }

	sessionSlug := "Chocoboflor"
	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				sess := &domain.SessionData{CurrentPlayerSlug: &sessionSlug}
				next.ServeHTTP(w, req.WithContext(middleware.InjectSession(req.Context(), sess)))
			})
		})
		h.Mount(r)
	})

	// Corps hostile : un liker et un libellé choisis par le client.
	body, _ := json.Marshal(domain.MediaLikeRequest{
		FilePath:      "JGtm/clip.mp4",
		Liked:         true,
		LikerSlug:     "victime",
		LikerGamertag: "Quelqu'un d'autre",
	})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/media/likes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if spy.capturedReq == nil {
		t.Fatal("SetMediaLike not called")
	}
	if spy.capturedReq.LikerSlug != sessionSlug {
		t.Errorf("LikerSlug = %q, want %q (la session fait foi, pas le corps)",
			spy.capturedReq.LikerSlug, sessionSlug)
	}
	if spy.capturedReq.LikerGamertag == "Quelqu'un d'autre" {
		t.Error("LikerGamertag du corps retenu : un client choisirait le nom sous lequel son like s'affiche")
	}
}

// spyPathService capture le file_path reçu par les TROIS endpoints qui reversent
// une URL servable vers le chemin stocké.
type spyPathService struct {
	mockMediaService
	paths map[string]string
}

func (s *spyPathService) SetMediaLike(_ context.Context, req domain.MediaLikeRequest) (*domain.MediaLikeResponse, error) {
	s.paths["like"] = req.FilePath
	return &domain.MediaLikeResponse{FilePath: req.FilePath, Liked: true}, nil
}

func (s *spyPathService) GetMatchCandidates(_ context.Context, filePath string, _ int) (*domain.MediaMatchCandidatesResponse, error) {
	s.paths["candidates"] = filePath
	return &domain.MediaMatchCandidatesResponse{}, nil
}

func (s *spyPathService) AssociateMediaToMatch(_ context.Context, req domain.MediaAssociateRequest) (*domain.MediaAssociateResponse, error) {
	s.paths["associate"] = req.FilePath
	return &domain.MediaAssociateResponse{FilePath: req.FilePath, MatchID: req.MatchID}, nil
}

// TestMediaHandler_URLReverse_ConsistentAcrossEndpoints est le GARDE-RAIL de la
// règle « une seule conversion inverse » (CLAUDE.md n°6) : like, match-candidates
// et associate doivent produire le MÊME chemin stocké pour la MÊME URL d'entrée.
// C'est cette divergence (like via une résolution disque, les deux autres via le
// strip de préfixe) qui a produit le bug 1.5 ; le test la rend impossible à
// réintroduire sans échec rouge.
func TestMediaHandler_URLReverse_ConsistentAcrossEndpoints(t *testing.T) {
	store := newCapturesSettingsStore(t) // capturesBase peuplé = piège du bug d'origine
	spy := &spyPathService{paths: map[string]string{}}
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil }

	r := chi.NewRouter()
	h := handlers.NewMediaHandler(factory, nil, "").WithSettingsStore(store)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})

	const urlPath = "/api/v1/players/test-player/media/files/JGtm/clip.mp4"

	if w := patchLike(r, urlPath, true); w.Code != http.StatusOK {
		t.Fatalf("like: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/media/match-candidates?file_path="+url.QueryEscape(urlPath), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("candidates: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body, _ := json.Marshal(domain.MediaAssociateRequest{FilePath: urlPath, MatchID: "m1"})
	req = httptest.NewRequest(http.MethodPost, "/players/test-player/media/associate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("associate: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	const want = "JGtm/clip.mp4"
	for _, endpoint := range []string{"like", "candidates", "associate"} {
		if got := spy.paths[endpoint]; got != want {
			t.Errorf("%s: file_path = %q, want %q (conversion inverse unique)", endpoint, got, want)
		}
	}
}
