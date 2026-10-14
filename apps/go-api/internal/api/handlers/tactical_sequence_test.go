package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

func newSequenceRouter(factory handlers.ServiceFactory[port.TacticalService]) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		handlers.NewTacticalSequenceHandler(factory).Mount(sub)
	})
	return r
}

func postSequence(r http.Handler, carte, corps string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/players/jgtm/tactical/"+carte+"/sequence", strings.NewReader(corps))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const corpsSequence = `{"match_id":"3f2a-b1","victim_xuid":"2533274812345","time_ms":123456}`

func TestSequence_Nominal(t *testing.T) {
	svc := &fakeTacticalSvc{sequence: domain.TacticalSequence{RayonM: 18, PasMs: 200}}
	w := postSequence(newSequenceRouter(tacticalFactory(svc, nil)), "carte-a", corpsSequence)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := svc.vuSequenceReq; got.MapID != "carte-a" || got.MatchID != "3f2a-b1" || got.VictimXUID != "2533274812345" || got.TimeMs != 123456 {
		t.Errorf("requete transmise = %+v", got)
	}
	if !strings.Contains(w.Body.String(), `"rayon_m":18`) {
		t.Errorf("corps = %s", w.Body.String())
	}
}

func TestSequence_Indisponible_Sans404Distinguable(t *testing.T) {
	// absence d'artefact, match d'un autre joueur, horloge non etablie : le service rend la meme
	// erreur ; une forme invalide est refusee AVANT le service, avec le MEME statut, code et message.
	svc := &fakeTacticalSvc{errSeq: port.ErrReplayNotAvailable}
	r := newSequenceRouter(tacticalFactory(svc, nil))
	absent := postSequence(r, "carte-a", corpsSequence)
	invalide := postSequence(r, "carte-a", `{"match_id":"../x","victim_xuid":"1","time_ms":1}`)
	carte := postSequence(r, "..%5Cx", corpsSequence)
	for nom, w := range map[string]*httptest.ResponseRecorder{"absent": absent, "forme": invalide, "carte": carte} {
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "replay_not_available") {
			t.Errorf("%s : %d %s", nom, w.Code, w.Body.String())
		}
	}
	if absent.Body.String() != invalide.Body.String() {
		t.Errorf("le refus de forme se distingue de l'absence :\n%s\n%s", absent.Body.String(), invalide.Body.String())
	}
	svc2 := &fakeTacticalSvc{}
	postSequence(newSequenceRouter(tacticalFactory(svc2, nil)), "carte-a", `{"match_id":"../x","victim_xuid":"1","time_ms":1}`)
	if svc2.vuAppele {
		t.Error("une entree hors vocabulaire ne doit atteindre ni le service ni le disque")
	}
}

func TestSequence_CapabilityAbsente_503(t *testing.T) {
	svc := &fakeTacticalSvc{errSeq: games.ErrCapabilityNotSupported}
	w := postSequence(newSequenceRouter(tacticalFactory(svc, nil)), "carte-a", corpsSequence)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
