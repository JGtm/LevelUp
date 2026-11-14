package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// Contrat HTTP de `?encoding=columns` : le paramètre est absent (la forme normale, octet pour octet),
// puis présent (le corps passe par l'encodeur injecté), puis invalide ou sans corps à encoder.

const columnsMarker = "ENCODE:"

type encoderSpy struct {
	calls int
	input string
	err   error
}

func (s *encoderSpy) encode(body []byte) ([]byte, error) {
	s.calls++
	s.input = string(body)
	if s.err != nil {
		return nil, s.err
	}
	return append([]byte(columnsMarker), body...), nil
}

func columnsRouter(svc port.ReplayService, enc handlers.ReplayColumnEncoder) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewReplayHandler(func(context.Context, string) (port.ReplayService, error) { return svc, nil }, enc)
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	return r
}

func columnsGet(r *chi.Mux, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/players/"+testPlayerSlug+"/matches/000d5950/replay"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func columnsMock() *mockReplayService {
	return &mockReplayService{doc: replaydoc.ReplayDocument{
		SchemaVersion: 116, MatchID: "000d5950", TitleSlug: "halo_infinite", FrameCount: 2,
		Tracks: []replaydoc.Track{{Slot: 1, Points: []replaydoc.Point{{T: 0, X: 1, Y: 2}}}},
	}}
}

func TestReplayColumns_ParametreAbsentLaFormeNormaleInchangee(t *testing.T) {
	spy := &encoderSpy{}
	r := columnsRouter(columnsMock(), spy.encode)
	plain := columnsGet(r, "")
	reference := columnsGet(columnsRouter(columnsMock(), nil), "")
	if plain.Code != http.StatusOK || plain.Body.String() != reference.Body.String() {
		t.Fatalf("sans paramètre, la réponse doit être celle d'un handler sans encodeur : %d", plain.Code)
	}
	if spy.calls != 0 || strings.HasPrefix(plain.Body.String(), columnsMarker) {
		t.Fatalf("l'encodeur ne doit pas être appelé sans paramètre (appels : %d)", spy.calls)
	}
}

func TestReplayColumns_ParametrePresentLeCorpsPasseParLEncodeur(t *testing.T) {
	spy := &encoderSpy{}
	r := columnsRouter(columnsMock(), spy.encode)
	plain := columnsGet(r, "")
	cols := columnsGet(r, "?encoding=columns")
	if cols.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d : %s", cols.Code, cols.Body.String())
	}
	if spy.calls != 1 || spy.input != plain.Body.String() {
		t.Fatalf("l'encodeur doit recevoir exactement le corps de la forme normale (appels %d)", spy.calls)
	}
	if cols.Body.String() != columnsMarker+plain.Body.String() {
		t.Fatalf("le corps servi doit être la sortie de l'encodeur : %.120s", cols.Body.String())
	}
	if ct := cols.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("le type de contenu doit rester application/json (compression), reçu %q", ct)
	}
	for _, h := range []string{"X-Replay-Latest-Schema-Version", "X-Replay-Cadence-Stale"} {
		if cols.Header().Get(h) != plain.Header().Get(h) {
			t.Fatalf("l'en-tête %s doit rester celui de la forme normale", h)
		}
	}
}

func TestReplayColumns_ParametreInvalideRefuseSansEncoder(t *testing.T) {
	spy := &encoderSpy{}
	w := columnsGet(columnsRouter(columnsMock(), spy.encode), "?encoding=zip")
	if w.Code != http.StatusUnprocessableEntity || spy.calls != 0 {
		t.Fatalf("un encodage inconnu doit être refusé en 422 sans encoder : %d, appels %d", w.Code, spy.calls)
	}
}

func TestReplayColumns_UneErreurPasseSansEtreEncodee(t *testing.T) {
	spy := &encoderSpy{}
	mock := &mockReplayService{err: port.ErrReplayNotAvailable}
	w := columnsGet(columnsRouter(mock, spy.encode), "?encoding=columns")
	if w.Code != http.StatusNotFound || spy.calls != 0 || !strings.Contains(w.Body.String(), "replay_not_available") {
		t.Fatalf("un 404 ne passe pas par l'encodeur : %d appels %d corps %s", w.Code, spy.calls, w.Body.String())
	}
}

func TestReplayColumns_RefusDeLEncodeurServiLaFormeNormale(t *testing.T) {
	before := observability.LoadCounter("replay_columns_refused_total")
	spy := &encoderSpy{err: errors.New("collision")}
	r := columnsRouter(columnsMock(), spy.encode)
	w := columnsGet(r, "?encoding=columns")
	plain := columnsGet(columnsRouter(columnsMock(), nil), "")
	if w.Code != http.StatusOK || w.Body.String() != plain.Body.String() {
		t.Fatalf("un refus doit servir la forme normale : %d", w.Code)
	}
	if got := observability.LoadCounter("replay_columns_refused_total"); got != before+1 {
		t.Fatalf("le refus doit être compté : %d -> %d", before, got)
	}
}

func TestReplayColumns_SansEncodeurLaFormeNormale(t *testing.T) {
	w := columnsGet(columnsRouter(columnsMock(), nil), "?encoding=columns")
	plain := columnsGet(columnsRouter(columnsMock(), nil), "")
	if w.Code != http.StatusOK || w.Body.String() != plain.Body.String() {
		t.Fatalf("sans encodeur branché, la forme normale est servie : %d", w.Code)
	}
}
