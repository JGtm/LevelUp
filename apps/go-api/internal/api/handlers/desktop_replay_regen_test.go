package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replayregen"
)

// fakeRegen : la file, réduite à ce que les routes lui demandent.
type fakeRegen struct {
	status  domain.ReplayRegenStatus
	started int
	err     error
}

func (f *fakeRegen) Status() domain.ReplayRegenStatus { return f.status }
func (f *fakeRegen) Start() (domain.ReplayRegenStatus, error) {
	f.started++
	f.status.Running = true
	return f.status, f.err
}
func (f *fakeRegen) Pause() (domain.ReplayRegenStatus, error) {
	if f.err != nil {
		return domain.ReplayRegenStatus{}, f.err
	}
	f.status.Paused = true
	return f.status, nil
}
func (f *fakeRegen) Resume() (domain.ReplayRegenStatus, error) {
	if f.err != nil {
		return domain.ReplayRegenStatus{}, f.err
	}
	f.status.Paused = false
	return f.status, nil
}

func regenRouter(svc handlers.ReplayRegenService) http.Handler {
	r := chi.NewRouter()
	handlers.NewDesktopReplayRegenHandler(svc).Mount(r)
	return r
}

func regenCall(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestDesktopReplayRegen_EtatEtCommandes(t *testing.T) {
	svc := &fakeRegen{status: domain.ReplayRegenStatus{Checked: true, Stale: 12}}
	h := regenRouter(svc)

	w := regenCall(t, h, http.MethodGet, "/desktop/replay-regen")
	var st domain.ReplayRegenStatus
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &st) != nil || st.Stale != 12 || !st.Checked {
		t.Fatalf("GET = %d %s", w.Code, w.Body.String())
	}
	if svc.started != 0 {
		t.Fatal("lire l'état ne lance rien")
	}

	w = regenCall(t, h, http.MethodPost, "/desktop/replay-regen/start")
	if w.Code != http.StatusAccepted || svc.started != 1 {
		t.Fatalf("start = %d, démarrages = %d", w.Code, svc.started)
	}
	if w = regenCall(t, h, http.MethodPost, "/desktop/replay-regen/pause"); w.Code != http.StatusOK || !svc.status.Paused {
		t.Fatalf("pause = %d %s", w.Code, w.Body.String())
	}
	if w = regenCall(t, h, http.MethodPost, "/desktop/replay-regen/resume"); w.Code != http.StatusOK || svc.status.Paused {
		t.Fatalf("resume = %d %s", w.Code, w.Body.String())
	}
}

func TestDesktopReplayRegen_Refus(t *testing.T) {
	svc := &fakeRegen{err: replayregen.ErrNotRunning}
	h := regenRouter(svc)
	w := regenCall(t, h, http.MethodPost, "/desktop/replay-regen/pause")
	if w.Code != http.StatusConflict || dataErrCode(t, w.Body.String()) != "replay_regen_not_running" {
		t.Fatalf("pause sans file = %d %s", w.Code, w.Body.String())
	}
	svc.err = replayregen.ErrNotBound
	w = regenCall(t, h, http.MethodPost, "/desktop/replay-regen/start")
	if w.Code != http.StatusServiceUnavailable || dataErrCode(t, w.Body.String()) != "replay_regen_unavailable" {
		t.Fatalf("start sans liaison = %d %s", w.Code, w.Body.String())
	}
}
