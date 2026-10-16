// media_upload_transverse_test.go — l'échéance de lecture d'un envoi de médias reportée à
// travers la pile transverse RÉELLE (applyTransverseMiddlewares, server.go).
//
// Contrat éprouvé : le handler d'envoi reporte l'échéance de lecture de la connexion
// (http.ResponseController.SetReadDeadline) une fois sa place obtenue. Le contrôleur
// remonte les enveloppes du ResponseWriter par leur méthode Unwrap : une enveloppe de la
// pile qui ne l'expose pas rend ErrNotSupported, et un envoi qui a attendu sa place
// au-delà du ReadTimeout du serveur échoue. Les tests passent par la pile assemblée
// (session, journal, compression, ...) et un vrai http.Server, pas par le handler seul.
package api

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	session_platform "levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/slots"
	"levelup/go-api/internal/port"
)

// transverseTestOrigin est l'origine autorisée par le CSRF de la pile de test.
const transverseTestOrigin = "http://localhost:5173"

const (
	transverseUploadPath   = "/api/v1/players/{player_slug}/media/upload"
	transverseDeadlinePath = "/api/v1/probe/read-deadline"
)

// instantUploadService répond aussitôt à un envoi ; toute autre méthode du port panique.
type instantUploadService struct{ port.MediaService }

func (instantUploadService) UploadMedia(context.Context, domain.UploadRequest) (*domain.UploadResult, error) {
	return &domain.UploadResult{Saved: 1}, nil
}

// newTransverseUploadServer sert la route d'envoi (place prise dans pool) et une sonde
// d'échéance derrière la pile transverse de production et un vrai http.Server dont
// l'échéance de lecture vaut readTimeout.
func newTransverseUploadServer(t *testing.T, pool *slots.Pool, readTimeout time.Duration) *httptest.Server {
	t.Helper()
	cfg := &config.AppConfig{CORSOrigins: []string{transverseTestOrigin}}
	store := session_platform.NewStore(t.TempDir(), session_platform.DefaultTTL,
		"secret-de-test-de-trente-deux-octets!") // pragma: allowlist secret
	r := chi.NewRouter()
	applyTransverseMiddlewares(r, cfg, store, middleware.SecureCookiePolicy{}, titlePkg.DefaultRegistry())

	svc := instantUploadService{}
	media := handlers.NewMediaHandler(
		func(context.Context, string) (port.MediaService, error) { return svc, nil },
		func(context.Context, string) (port.MediaService, string, string, string, string, string, error) {
			return svc, "TestPlayer", titlePkg.DefaultSlug, t.TempDir(), "", "", nil
		}, "").WithUploadSlots(pool, 5*time.Second)
	r.Post(transverseUploadPath, media.PostUploadMedia)
	r.Post(transverseDeadlinePath, func(w http.ResponseWriter, _ *http.Request) {
		if err := http.NewResponseController(w).SetReadDeadline(time.Now().Add(time.Minute)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewUnstartedServer(r)
	srv.Config.ReadTimeout = readTimeout
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

// postFromOrigin émet un POST accepté par le CSRF (origine autorisée).
func postFromOrigin(t *testing.T, url, contentType string, body []byte) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", transverseTestOrigin)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(got)
}

// transverseClip rend le corps multipart d'un clip MP4 de 1 Mio et son Content-Type : au-delà
// du tampon de lecture du serveur, le corps se lit bien sur la connexion.
func transverseClip(t *testing.T) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("files", "clip.mp4")
	if err != nil {
		t.Fatal(err)
	}
	head := []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom")
	if _, err := fw.Write(append(head, make([]byte, 1<<20)...)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), mw.FormDataContentType()
}

// La pile transverse laisse un handler atteindre la connexion pour reporter son échéance
// de lecture.
func TestTransverseStack_ReadDeadlineReachesTheConnection(t *testing.T) {
	srv := newTransverseUploadServer(t, slots.New(1), 5*time.Second)
	status, got := postFromOrigin(t, srv.URL+transverseDeadlinePath, "", nil)
	if status != http.StatusNoContent {
		t.Fatalf("statut %d, attendu 204 : SetReadDeadline refusé à travers la pile : %s", status, got)
	}
}

// Un envoi qui attend sa place au-delà du ReadTimeout du serveur est servi à travers la
// pile de production : son corps se lit dans le budget reporté après l'obtention de la place.
func TestTransverseStack_UploadWaitingBeyondReadTimeoutIsServed(t *testing.T) {
	pool := slots.New(1)
	release, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	srv := newTransverseUploadServer(t, pool, 200*time.Millisecond)
	time.AfterFunc(600*time.Millisecond, release) // la place se libère après l'échéance de lecture

	body, contentType := transverseClip(t)
	status, got := postFromOrigin(t, srv.URL+"/api/v1/players/test-player/media/upload", contentType, body)
	if status != http.StatusOK {
		t.Fatalf("statut %d, attendu 200 : %s", status, got)
	}
}
