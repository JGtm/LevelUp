package handlers_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/slots"
	"levelup/go-api/internal/port"
)

// instantUploadService répond aussitôt à un envoi.
type instantUploadService struct{ mockMediaService }

func (*instantUploadService) UploadMedia(context.Context, domain.UploadRequest) (*domain.UploadResult, error) {
	return &domain.UploadResult{Saved: 1}, nil
}

// newDeadlineServer sert la route d'envoi derrière un vrai http.Server dont l'échéance de
// lecture (ReadTimeout) vaut readTimeout.
func newDeadlineServer(t *testing.T, h *handlers.MediaHandler, readTimeout time.Duration) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Post("/players/{player_slug}/media/upload", h.PostUploadMedia)
	srv := httptest.NewUnstartedServer(r)
	srv.Config.ReadTimeout = readTimeout
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func newDeadlineHandler(pool *slots.Pool, wait time.Duration) *handlers.MediaHandler {
	svc := &instantUploadService{}
	return handlers.NewMediaHandler(
		func(context.Context, string) (port.MediaService, error) { return svc, nil },
		func(context.Context, string) (port.MediaService, string, string, string, string, string, error) {
			return svc, "TestPlayer", testTitleSlug, "/data/players/TestPlayer/stats.duckdb", "", "", nil
		}, "").WithUploadSlots(pool, wait)
}

// multipartClip rend le corps multipart d'un clip de 1 Mio et son Content-Type.
func multipartClip(t *testing.T) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("files", "clip.mp4")
	if err != nil {
		t.Fatal(err)
	}
	// 1 Mio : au-delà du tampon de lecture du serveur, le corps se lit bien sur la connexion.
	clip := append(append([]byte{}, uploadHeadMP4...), make([]byte, 1<<20)...)
	if _, err := fw.Write(clip); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), mw.FormDataContentType()
}

// Un envoi qui attend sa place au-delà de l'échéance de lecture du serveur est servi : son
// corps se lit après l'obtention de la place, jamais un 413 « trop volumineux » pour un
// fichier de quelques octets.
func TestUploadHandler_SlotWaitBeyondReadTimeoutIsServed(t *testing.T) {
	pool := slots.New(1)
	release, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	srv := newDeadlineServer(t, newDeadlineHandler(pool, 5*time.Second), 200*time.Millisecond)
	time.AfterFunc(600*time.Millisecond, release) // la place se libère après l'échéance de lecture

	body, contentType := multipartClip(t)
	resp, err := http.Post(srv.URL+"/players/test-player/media/upload", contentType, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200 : %s", resp.StatusCode, got)
	}
}

// Un corps qui n'arrive pas dans le budget de lecture reçoit 503 upload_busy + Retry-After,
// jamais un 413 trompeur.
func TestUploadHandler_BodyReadDeadlineAnswers503(t *testing.T) {
	h := newDeadlineHandler(slots.New(1), 5*time.Second).WithUploadBodyReadBudget(200 * time.Millisecond)
	srv := newDeadlineServer(t, h, 0)

	body, contentType := multipartClip(t)
	// Connexion brute : l'en-tête annonce le corps entier, seule sa première moitié part.
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	head := fmt.Sprintf("POST /players/test-player/media/upload HTTP/1.1\r\nHost: x\r\n"+
		"Content-Type: %s\r\nContent-Length: %d\r\n\r\n", contentType, len(body))
	if _, err := conn.Write(append([]byte(head), body[:len(body)/2]...)); err != nil {
		t.Fatal(err)
	}
	// Sans budget de lecture, le serveur attendrait la suite du corps indéfiniment.
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("aucune réponse du serveur : %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("statut %d, attendu 503 : %s", resp.StatusCode, got)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("le 503 doit porter Retry-After")
	}
	if !strings.Contains(string(got), "upload_busy") {
		t.Errorf("code attendu upload_busy : %s", got)
	}
}
