package handlers_test

// replay_demo_gate_sequence_test.go — la garde du rejeu en DÉMO sur la route de la séquence
// d'une mort, dont le match_id est dans le CORPS et non dans le chemin.
//
// Avant : ReplayGate ne lisait que le paramètre de chemin `match_id`, vide sur cette route, donc
// la séquence répondait toujours 404 en démo alors que le rejeu des matchs figés y est servi.
// La garde lit désormais le match_id du corps pour cette route, restitue le corps au handler, et
// refuse (404 identique au refus d'un match non figé) tout corps illisible ou démesuré.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/port"
)

// figesSequence : les matchs figés de la démo, par « titre/match ».
type figesSequence map[string]bool

func (f figesSequence) Allows(_ context.Context, titleSlug, matchID string) bool {
	return f[titleSlug+"/"+matchID]
}

// routeurSequenceDemo monte la route de la séquence derrière la garde démo, comme le groupe du
// rejeu de server_apiv1.go.
func routeurSequenceDemo(svc port.TacticalService, figes figesSequence) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		sub.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(ctxkeys.WithTitleSlug(req.Context(), "halo_infinite")))
			})
		})
		sub.Use(handlers.ReplayGate(figes))
		handlers.NewTacticalSequenceHandler(tacticalFactory(svc, nil)).Mount(sub)
	})
	return r
}

func sequenceDistante(r http.Handler, corps string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/players/jgtm/tactical/carte-a/sequence", strings.NewReader(corps))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:4444"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSequenceDemo_MatchFigeDuCorpsEstServi(t *testing.T) {
	svc := &fakeTacticalSvc{}
	r := routeurSequenceDemo(svc, figesSequence{"halo_infinite/3f2a-b1": true})
	w := sequenceDistante(r, corpsSequence)
	if w.Code != http.StatusOK {
		t.Fatalf("match figé lu dans le corps : statut %d (%s), attendu 200", w.Code, w.Body.String())
	}
	if got := svc.vuSequenceReq; got.MatchID != "3f2a-b1" || got.VictimXUID != "2533274812345" || got.TimeMs != 123456 {
		t.Errorf("le handler doit recevoir le corps INTACT après la garde, reçu %+v", got)
	}
}

func TestSequenceDemo_RefusIdentiques(t *testing.T) {
	grand := `{"match_id":"3f2a-b1","victim_xuid":"2533274812345","time_ms":1,"x":"` + strings.Repeat("a", 1<<20) + `"}`
	refus := map[string]string{
		"match non figé":     `{"match_id":"autre-02","victim_xuid":"2533274812345","time_ms":1}`,
		"match_id absent":    `{"victim_xuid":"2533274812345","time_ms":1}`,
		"match_id vide":      `{"match_id":"","victim_xuid":"2533274812345","time_ms":1}`,
		"corps non JSON":     `pas du json`,
		"corps vide":         ``,
		"match_id non texte": `{"match_id":123,"victim_xuid":"2533274812345","time_ms":1}`,
		"corps démesuré":     grand,
	}
	for nom, corps := range refus {
		svc := &fakeTacticalSvc{}
		w := sequenceDistante(routeurSequenceDemo(svc, figesSequence{"halo_infinite/3f2a-b1": true}), corps)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "replay_not_available") {
			t.Errorf("%s : statut %d (%s), attendu 404 replay_not_available", nom, w.Code, w.Body.String())
		}
		if svc.vuAppele || svc.vuSequenceReq.MatchID != "" {
			t.Errorf("%s : le refus doit précéder le service", nom)
		}
	}
}
