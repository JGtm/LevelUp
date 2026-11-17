package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
)

const genTestMatch = "0edb8512-1111-4222-8333-444455556666"

// fakeReplayGeneration : un port qui rend ce qu'on lui pose et enregistre ce qu'on lui demande.
type fakeReplayGeneration struct {
	st      domain.ReplayGenerationStatus
	created bool
	err     error
	gotSlug string
	gotID   string
}

func (f *fakeReplayGeneration) RequestGeneration(_ context.Context, slug, id string) (domain.ReplayGenerationStatus, bool, error) {
	f.gotSlug, f.gotID = slug, id
	return f.st, f.created, f.err
}

func (f *fakeReplayGeneration) GenerationStatus(_ context.Context, slug, id string) (domain.ReplayGenerationStatus, error) {
	f.gotSlug, f.gotID = slug, id
	return f.st, f.err
}

func newGenerationRouter(svc port.ReplayGeneration, factoryErr error) http.Handler {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		NewReplayGenerationHandler(func(context.Context, string) (port.ReplayGeneration, error) {
			return svc, factoryErr
		}).Mount(r)
	})
	return r
}

func doGeneration(h http.Handler, method, matchID string) *httptest.ResponseRecorder {
	suffix := map[string]string{http.MethodPost: "generate", http.MethodGet: "generation"}[method]
	req := httptest.NewRequest(method, "/players/P/matches/"+matchID+"/replay/"+suffix, strings.NewReader(""))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

type generationBody struct {
	Created *bool                          `json:"created"`
	Status  *domain.ReplayGenerationStatus `json:"status"`
	Code    string                         `json:"code"`
	Message string                         `json:"message"`
	Retry   *bool                          `json:"retryable"`
	Details map[string]string              `json:"details"`
}

func decodeGeneration(t *testing.T, w *httptest.ResponseRecorder) generationBody {
	t.Helper()
	var b generationBody
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("corps illisible %q : %v", w.Body.String(), err)
	}
	return b
}

// POST : 202 créée ou doublon, 200 si le rejeu est déjà rangé ; le titre vient du contexte, le match du chemin.
func TestReplayGeneration_PostSucces(t *testing.T) {
	cas := []struct {
		nom     string
		st      domain.ReplayGenerationStatus
		created bool
		want    int
	}{
		{"créée", domain.ReplayGenerationStatus{State: domain.ReplayGenerationQueued, Position: 1}, true, http.StatusAccepted},
		{"doublon en file", domain.ReplayGenerationStatus{State: domain.ReplayGenerationQueued, Position: 2}, false, http.StatusAccepted},
		{"doublon en cours", domain.ReplayGenerationStatus{State: domain.ReplayGenerationRunning, Phase: domain.ReplayGenPhaseBuilding}, false, http.StatusAccepted},
		{"déjà rangé", domain.ReplayGenerationStatus{State: domain.ReplayGenerationReady}, false, http.StatusOK},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := &fakeReplayGeneration{st: c.st, created: c.created}
			w := doGeneration(newGenerationRouter(svc, nil), http.MethodPost, genTestMatch)
			if w.Code != c.want {
				t.Fatalf("statut = %d, attendu %d : %s", w.Code, c.want, w.Body.String())
			}
			b := decodeGeneration(t, w)
			if b.Created == nil || *b.Created != c.created || b.Status == nil || b.Status.State != c.st.State {
				t.Fatalf("corps = %s", w.Body.String())
			}
			if svc.gotID != genTestMatch || svc.gotSlug == "" {
				t.Fatalf("service appelé avec slug=%q id=%q", svc.gotSlug, svc.gotID)
			}
		})
	}
}

// POST : les refus, avec leur code, leur nature réessayable et leur raison ; aucun texte d'erreur ne part au client.
func TestReplayGeneration_PostRefus(t *testing.T) {
	cas := []struct {
		nom       string
		err       error
		want      int
		code      string
		retryable bool
		reason    string
		secret    string
	}{
		{"film indisponible", domain.ErrReplayFilmUnavailable, http.StatusConflict, "replay_film_unavailable", false, "", ""},
		{"non pris en charge, raison", &domain.ReplayGenerationNotSupportedError{Reason: domain.ReplayGenReasonNoToken}, http.StatusConflict, "replay_generation_not_supported", false, domain.ReplayGenReasonNoToken, ""},
		{"non pris en charge, sans raison", domain.ErrReplayGenerationNotSupported, http.StatusConflict, "replay_generation_not_supported", false, "", ""},
		{"file pleine", fmt.Errorf("x: %w", domain.ErrReplayGenerationQueueFull), http.StatusTooManyRequests, "replay_generation_queue_full", true, "", ""},
		{"capacité absente", fmt.Errorf("x: %w", games.ErrCapabilityNotSupported), http.StatusServiceUnavailable, "capability_not_supported", false, "", ""},
		{"file non reliée", domain.ErrReplayGenerationUnbound, http.StatusServiceUnavailable, "replay_generation_unavailable", false, "", ""},
		{"autre erreur", errors.New("SECRET-INTERNE disque"), http.StatusInternalServerError, "replay_generation_error", false, "", "SECRET-INTERNE"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := &fakeReplayGeneration{err: c.err}
			w := doGeneration(newGenerationRouter(svc, nil), http.MethodPost, genTestMatch)
			if w.Code != c.want {
				t.Fatalf("statut = %d, attendu %d : %s", w.Code, c.want, w.Body.String())
			}
			b := decodeGeneration(t, w)
			if b.Code != c.code || b.Message == "" {
				t.Fatalf("corps = %s, attendu le code %s", w.Body.String(), c.code)
			}
			if c.want == http.StatusTooManyRequests && (b.Retry == nil || *b.Retry != c.retryable) {
				t.Fatalf("retryable = %v, attendu %v", b.Retry, c.retryable)
			}
			if b.Details[detailKeyReason] != c.reason {
				t.Fatalf("details = %v, attendu la raison %q", b.Details, c.reason)
			}
			if c.secret != "" && strings.Contains(w.Body.String(), c.secret) {
				t.Fatalf("le texte de l'erreur ne doit pas partir au client : %s", w.Body.String())
			}
		})
	}
}

// Identifiant mal formé : 400 avant tout appel du service ; profil introuvable : 404.
func TestReplayGeneration_IdentifiantEtProfil(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		svc := &fakeReplayGeneration{}
		w := doGeneration(newGenerationRouter(svc, nil), method, "pas-un-match-zzz")
		if w.Code != http.StatusBadRequest || decodeGeneration(t, w).Code != "invalid_match_id" {
			t.Fatalf("%s : statut %d corps %s, attendu 400 invalid_match_id", method, w.Code, w.Body.String())
		}
		if svc.gotID != "" {
			t.Fatalf("%s : le service ne doit pas être appelé", method)
		}
		w = doGeneration(newGenerationRouter(nil, errors.New("inconnu")), method, genTestMatch)
		if w.Code != http.StatusNotFound || decodeGeneration(t, w).Code != "player_not_found" {
			t.Fatalf("%s : statut %d corps %s, attendu 404 player_not_found", method, w.Code, w.Body.String())
		}
	}
}

// GET : 200 pour chaque état, le corps est l'état ; capacité absente : 503.
func TestReplayGeneration_GetUnEtatParCas(t *testing.T) {
	for _, st := range []domain.ReplayGenerationStatus{
		{State: domain.ReplayGenerationGeneratable},
		{State: domain.ReplayGenerationQueued, Position: 3, WaitingForGame: true, Paused: true},
		{State: domain.ReplayGenerationRunning, Phase: domain.ReplayGenPhaseDownloading},
		{State: domain.ReplayGenerationFailed, Reason: domain.ReplayGenReasonFetchFailed, Retryable: true},
		{State: domain.ReplayGenerationFilmUnavailable},
		{State: domain.ReplayGenerationNotSupported, Reason: domain.ReplayGenReasonModeUnmanaged},
		{State: domain.ReplayGenerationReady},
	} {
		w := doGeneration(newGenerationRouter(&fakeReplayGeneration{st: st}, nil), http.MethodGet, genTestMatch)
		if w.Code != http.StatusOK {
			t.Fatalf("%s : statut %d : %s", st.State, w.Code, w.Body.String())
		}
		var got domain.ReplayGenerationStatus
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got != st {
			t.Fatalf("%s : corps %s (%v), attendu %+v", st.State, w.Body.String(), err, st)
		}
	}
	w := doGeneration(newGenerationRouter(&fakeReplayGeneration{err: games.ErrCapabilityNotSupported}, nil), http.MethodGet, genTestMatch)
	if w.Code != http.StatusServiceUnavailable || decodeGeneration(t, w).Code != "capability_not_supported" {
		t.Fatalf("capacité absente : statut %d corps %s", w.Code, w.Body.String())
	}
	w = doGeneration(newGenerationRouter(&fakeReplayGeneration{err: errors.New("boom")}, nil), http.MethodGet, genTestMatch)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("erreur interne : statut %d", w.Code)
	}
}

// Base occupée par une synchronisation (bascule du fournisseur partagé en échec) : l'état comme la demande rendent
// 503 db_busy avec Retry-After par le classement commun des erreurs, jamais 500 replay_generation_error.
func TestReplayGeneration_BaseOccupeeRend503AvecRetryAfter(t *testing.T) {
	occupee := fmt.Errorf("registre des matchs: %w", sharedprovider.ErrSwapTimeout)
	for _, methode := range []string{http.MethodGet, http.MethodPost} {
		w := doGeneration(newGenerationRouter(&fakeReplayGeneration{err: occupee}, nil), methode, genTestMatch)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s : statut %d, attendu 503 : %s", methode, w.Code, w.Body.String())
		}
		if got := decodeGeneration(t, w).Code; got != codeDBBusy {
			t.Errorf("%s : code %q, attendu %q", methode, got, codeDBBusy)
		}
		if w.Header().Get("Retry-After") == "" {
			t.Errorf("%s : Retry-After absent", methode)
		}
	}
}
