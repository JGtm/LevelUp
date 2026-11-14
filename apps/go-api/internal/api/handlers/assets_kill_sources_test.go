package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
)

// mockKillSourceIconService implémente port.KillSourceIconService pour les tests du handler.
type mockKillSourceIconService struct {
	items    []domain.KillSourceIconEntry
	err      error
	gotTitle string
}

func (m *mockKillSourceIconService) ListKillSourceIcons(_ context.Context, titleID string) ([]domain.KillSourceIconEntry, error) {
	m.gotTitle = titleID
	return m.items, m.err
}

func newKillSourceIconsRouter(svc *mockKillSourceIconService) *chi.Mux {
	r := chi.NewRouter()
	handlers.NewKillSourceIconsHandler(svc).Mount(r)
	return r
}

func TestKillSourceIconsHandler_ServesEntries(t *testing.T) {
	svc := &mockKillSourceIconService{items: []domain.KillSourceIconEntry{
		{Key: "k_a", Label: "Arme A", ImageURL: "/static/a.png", Tinted: true},
		{Key: "k_b", Label: "Arme B", ImageURL: "/static/b.png"},
	}}
	w := httptest.NewRecorder()
	newKillSourceIconsRouter(svc).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/un_titre/kill-sources", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d : %s", w.Code, w.Body.String())
	}
	if svc.gotTitle != "un_titre" {
		t.Errorf("titre transmis au service = %q, attendu un_titre", svc.gotTitle)
	}
	var got []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("décodage : %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("%d entrées, attendu 2", len(got))
	}
	for _, champ := range []string{"key", "label", "image_url", "tinted"} {
		if _, ok := got[0][champ]; !ok {
			t.Errorf("le champ %q manque au corps : la forme de réponse est {key, label, image_url, tinted}", champ)
		}
	}
	if got[0]["key"] != "k_a" || got[0]["image_url"] != "/static/a.png" || got[0]["tinted"] != true {
		t.Errorf("première entrée = %v", got[0])
	}
}

// Un titre sans capacité rend une liste vide : un tableau `[]`, jamais `null` ni une erreur.
func TestKillSourceIconsHandler_EmptyIsAnArray(t *testing.T) {
	for nom, items := range map[string][]domain.KillSourceIconEntry{"nil": nil, "vide": {}} {
		t.Run(nom, func(t *testing.T) {
			w := httptest.NewRecorder()
			newKillSourceIconsRouter(&mockKillSourceIconService{items: items}).
				ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/sans_capacite/kill-sources", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("attendu 200, obtenu %d", w.Code)
			}
			if body := strings.TrimSpace(w.Body.String()); body != "[]" {
				t.Errorf("corps = %q, attendu []", body)
			}
		})
	}
}

// La cause d'une erreur de service va au journal, le client reçoit le message fixe du code.
func TestKillSourceIconsHandler_ServiceErrorIsFixedMessage(t *testing.T) {
	svc := &mockKillSourceIconService{err: errors.New("detail interne secret")}
	w := httptest.NewRecorder()
	newKillSourceIconsRouter(svc).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/kill-sources", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret") {
		t.Errorf("le texte de l'erreur fuit au client : %s", w.Body.String())
	}
}
