package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/slots"
	"levelup/go-api/internal/port"
)

// gatedUploadService bloque UploadMedia jusqu'à la libération et mesure la concurrence.
type gatedUploadService struct {
	mockMediaService
	release  chan struct{}
	entered  chan struct{}
	inFlight atomic.Int32
	peak     atomic.Int32
}

func (g *gatedUploadService) UploadMedia(ctx context.Context, _ domain.UploadRequest) (*domain.UploadResult, error) {
	n := g.inFlight.Add(1)
	for {
		p := g.peak.Load()
		if n <= p || g.peak.CompareAndSwap(p, n) {
			break
		}
	}
	g.entered <- struct{}{}
	select {
	case <-g.release:
	case <-ctx.Done():
	}
	g.inFlight.Add(-1)
	return &domain.UploadResult{Saved: 1}, nil
}

func newSlotRouter(svc port.MediaService, pool *slots.Pool, wait time.Duration) http.Handler {
	h := handlers.NewMediaHandler(
		func(context.Context, string) (port.MediaService, error) { return svc, nil },
		func(context.Context, string) (port.MediaService, string, string, string, string, string, error) {
			return svc, "TestPlayer", testTitleSlug, "/data/players/TestPlayer/stats.duckdb", "", "", nil
		}, "").WithUploadSlots(pool, wait)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/media/upload", h.PostUploadMedia)
	})
	return r
}

func postClip(t *testing.T, r http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{"clip.mp4": uploadHeadMP4}))
	return w
}

// Au plus deux envois en vol ; les suivants attendent leur tour puis aboutissent.
func TestUploadHandler_AtMostTwoUploadsInFlight(t *testing.T) {
	svc := &gatedUploadService{release: make(chan struct{}), entered: make(chan struct{}, 8)}
	r := newSlotRouter(svc, slots.New(2), 5*time.Second)

	var wg sync.WaitGroup
	codes := make(chan int, 5)
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- postClip(t, r).Code
		}()
	}
	// Deux entrent ; les trois autres restent en attente.
	<-svc.entered
	<-svc.entered
	time.Sleep(150 * time.Millisecond)
	if got := svc.peak.Load(); got != 2 {
		t.Fatalf("envois en vol = %d, attendu 2", got)
	}
	close(svc.release) // libère tout : les suivants entrent à leur tour
	go func() {
		for range 3 {
			<-svc.entered
		}
	}()
	wg.Wait()
	close(codes)
	for c := range codes {
		if c != http.StatusOK {
			t.Errorf("statut %d, attendu 200 (l'envoi en attente doit aboutir)", c)
		}
	}
	if got := svc.peak.Load(); got != 2 {
		t.Fatalf("pic d'envois en vol = %d, attendu 2", got)
	}
}

// Un envoi qui n'obtient pas de place dans le délai reçoit un 503 clair, jamais une erreur muette.
func TestUploadHandler_BusyAnswers503WithRetryAfter(t *testing.T) {
	svc := &gatedUploadService{release: make(chan struct{}), entered: make(chan struct{}, 8)}
	pool := slots.New(1)
	r := newSlotRouter(svc, pool, 80*time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		postClip(t, r)
	}()
	<-svc.entered // la seule place est tenue

	w := postClip(t, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("statut %d, attendu 503: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("le 503 doit porter Retry-After")
	}
	if !strings.Contains(w.Body.String(), "upload_busy") {
		t.Errorf("le corps doit porter le code upload_busy: %s", w.Body.String())
	}
	close(svc.release)
	<-done
	if pool.InUse() != 0 {
		t.Errorf("places encore tenues après la fin: %d", pool.InUse())
	}
}
