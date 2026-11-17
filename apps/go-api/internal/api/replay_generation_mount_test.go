package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// genMountMatch : un identifiant de match de forme Halo.
const genMountMatch = "0edb8512-1111-4222-8333-444455556666"

// genMountSpy : un port qui enregistre les profils pour lesquels le service a été demandé.
type genMountSpy struct{ slugs []string }

func (s *genMountSpy) factory(_ context.Context, slug string) (port.ReplayGeneration, error) {
	s.slugs = append(s.slugs, slug)
	return genMountService{}, nil
}

type genMountService struct{}

func (genMountService) RequestGeneration(context.Context, string, string) (domain.ReplayGenerationStatus, bool, error) {
	return domain.ReplayGenerationStatus{State: domain.ReplayGenerationQueued, Position: 1}, true, nil
}

func (genMountService) GenerationStatus(context.Context, string, string) (domain.ReplayGenerationStatus, error) {
	return domain.ReplayGenerationStatus{State: domain.ReplayGenerationGeneratable}, nil
}

// newGenMountRouter monte les routes comme dans server_apiv1.go : sous /players/{player_slug}, derrière la garde de
// propriété, avec la garde de participation en garde de route.
func newGenMountRouter(generate bool, access *middleware.PlayerAccess, participates bool, spy *genMountSpy) http.Handler {
	cfg := &config.AppConfig{}
	cfg.Capabilities.GenerateReplays = generate
	r := chi.NewRouter()
	r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
		r.Use(middleware.RequirePlayerOwnership(access))
		participated := middleware.RequireMatchParticipation(
			func(context.Context, string, string) (bool, error) { return participates, nil })
		mountReplayGeneration(r, cfg, handlers.ServiceFactory[port.ReplayGeneration](spy.factory), participated)
	})
	return r
}

func genMountDo(h http.Handler, method, player, user string) *httptest.ResponseRecorder {
	suffix := map[string]string{http.MethodPost: "generate", http.MethodGet: "generation"}[method]
	req := httptest.NewRequest(method, "/api/v1/players/"+player+"/matches/"+genMountMatch+"/replay/"+suffix, strings.NewReader(""))
	if user != "" {
		current := player
		req = req.WithContext(middleware.InjectSession(req.Context(),
			&domain.SessionData{SessionID: "s", Username: &user, CurrentPlayerSlug: &current}))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// Sans la capacité GenerateReplays, ni la route d'écriture ni celle de lecture n'existent (404) ; avec elle, elles
// répondent.
func TestMountReplayGeneration_SousLaCapaciteSeulement(t *testing.T) {
	for _, tc := range []struct {
		nom      string
		generate bool
		post     int
		get      int
	}{
		{"sans la capacité", false, http.StatusNotFound, http.StatusNotFound},
		{"avec la capacité", true, http.StatusAccepted, http.StatusOK},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			spy := &genMountSpy{}
			h := newGenMountRouter(tc.generate, openAccess(), true, spy)
			if w := genMountDo(h, http.MethodPost, "p", ""); w.Code != tc.post {
				t.Fatalf("POST = %d, attendu %d : %s", w.Code, tc.post, w.Body.String())
			}
			if w := genMountDo(h, http.MethodGet, "p", ""); w.Code != tc.get {
				t.Fatalf("GET = %d, attendu %d : %s", w.Code, tc.get, w.Body.String())
			}
			if !tc.generate && len(spy.slugs) != 0 {
				t.Fatal("sans la capacité, le service ne doit jamais être construit")
			}
		})
	}
}

// La propriété du profil s'applique à la demande (mode xbox, propriété appliquée) : le propriétaire et
// l'administrateur passent, un co-membre de groupe est refusé en écriture (403) mais lit (200) ; le service n'est
// construit pour aucune demande refusée.
func TestMountReplayGeneration_ProprieteAppliquee(t *testing.T) {
	users := ambUsers{
		"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
		"root":  {Username: "root", Role: domain.RoleAdmin, XUID: "1"},
	}
	profiles := map[string]string{"alice": "222", "bob": "999"}
	access := middleware.NewPlayerAccess(false, "xbox",
		func(_ context.Context, slug string) (string, bool) { x, ok := profiles[slug]; return x, ok },
		users, func(context.Context) map[string]bool { return map[string]bool{"222": true, "999": true} })

	for _, tc := range []struct {
		nom    string
		method string
		player string
		user   string
		want   int
		built  int
	}{
		{"propriétaire écrit sur son profil", http.MethodPost, "alice", "alice", http.StatusAccepted, 1},
		{"co-membre écrit sur le profil d'un autre", http.MethodPost, "bob", "alice", http.StatusForbidden, 0},
		{"co-membre lit le profil d'un autre", http.MethodGet, "bob", "alice", http.StatusOK, 1},
		{"administrateur écrit sur un profil", http.MethodPost, "bob", "root", http.StatusAccepted, 1},
		{"profil inconnu refusé à un utilisateur", http.MethodPost, "inconnu", "alice", http.StatusForbidden, 0},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			spy := &genMountSpy{}
			w := genMountDo(newGenMountRouter(true, access, true, spy), tc.method, tc.player, tc.user)
			if w.Code != tc.want || len(spy.slugs) != tc.built {
				t.Fatalf("%s %s en %s = %d (service construit %d fois), attendu %d et %d : %s",
					tc.method, tc.player, tc.user, w.Code, len(spy.slugs), tc.want, tc.built, w.Body.String())
			}
			if tc.want == http.StatusForbidden && !strings.Contains(w.Body.String(), "player_forbidden") {
				t.Fatalf("refus attendu player_forbidden : %s", w.Body.String())
			}
		})
	}
}

// Un match que le joueur du chemin n'a pas joué : 404 match_not_participant, le service n'est pas construit.
func TestMountReplayGeneration_NonParticipant(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		spy := &genMountSpy{}
		w := genMountDo(newGenMountRouter(true, openAccess(), false, spy), method, "p", "")
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "match_not_participant") || len(spy.slugs) != 0 {
			t.Fatalf("%s = %d (service construit %d fois) : %s", method, w.Code, len(spy.slugs), w.Body.String())
		}
	}
}

// Le montage vit dans le groupe du rejeu de server_apiv1.go, donc derrière la capacité de rejeu du titre
// (RequireCapability(CapReplay)), la garde de démo du rejeu et celles du groupe joueur.
func TestMountReplayGeneration_DansLeGroupeDuRejeu(t *testing.T) {
	raw, err := os.ReadFile("server_apiv1.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	group := strings.Index(src, "middleware.RequireCapability(titleRegistry, titlePkg.CapReplay)")
	mount := strings.Index(src, "mountReplayGeneration(r, cfg, reg.ReplayGeneration, participated, playerOpt)")
	end := strings.Index(src, "handlers.NewTacticalSequenceHandler(reg.Tactical)")
	if group < 0 || mount < 0 || end < 0 || !(group < mount && mount < end) {
		t.Fatalf("mountReplayGeneration doit être dans le groupe CapReplay du rejeu (groupe %d, montage %d, fin %d)", group, mount, end)
	}
}
