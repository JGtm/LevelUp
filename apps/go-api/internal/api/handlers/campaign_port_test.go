// Package handlers_test — campaign_port_test.go : les routes /campaigns sur un faux
// port.CampaignService : correspondance des refus métier, paramètres transmis, projections.
package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/port"
)

// fakeCampaignService rend l'erreur préparée à chaque appel et note le dernier appel.
type fakeCampaignService struct {
	err     error
	one     campaign.ImprovementCampaign
	list    []campaign.ImprovementCampaign
	called  string
	player  string
	title   string
	id      string
	started campaign.StartParams
}

func (f *fakeCampaignService) Start(_ context.Context, player string, p campaign.StartParams, _ time.Time) (campaign.ImprovementCampaign, error) {
	f.called, f.player, f.started = "start", player, p
	return f.one, f.err
}

func (f *fakeCampaignService) Active(_ context.Context, player, title string) (campaign.ImprovementCampaign, error) {
	f.called, f.player, f.title = "active", player, title
	return f.one, f.err
}

func (f *fakeCampaignService) Ended(_ context.Context, player, title string) ([]campaign.ImprovementCampaign, error) {
	f.called, f.player, f.title = "ended", player, title
	return f.list, f.err
}

func (f *fakeCampaignService) ByID(_ context.Context, player, id string) (campaign.ImprovementCampaign, error) {
	f.called, f.player, f.id = "byid", player, id
	return f.one, f.err
}

func (f *fakeCampaignService) Pause(_ context.Context, player, id string) error {
	f.called, f.player, f.id = "pause", player, id
	return f.err
}

func (f *fakeCampaignService) Resume(_ context.Context, player, id string) error {
	f.called, f.player, f.id = "resume", player, id
	return f.err
}

func (f *fakeCampaignService) Close(_ context.Context, player, id string, _ time.Time) error {
	f.called, f.player, f.id = "close", player, id
	return f.err
}

func (f *fakeCampaignService) Abandon(_ context.Context, player, id string, _ time.Time) error {
	f.called, f.player, f.id = "abandon", player, id
	return f.err
}

func campaignRouter(svc port.CampaignService) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewCampaignHandler(svc, "")
	r.Route("/api/v1/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	return r
}

func doCampaign(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/players/jg"+path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCampaignHandler_StartTransmetLeCorpsEtRepond201(t *testing.T) {
	fake := &fakeCampaignService{one: campaign.ImprovementCampaign{ID: "c1", Axis: "combat"}}
	rec := doCampaign(campaignRouter(fake), http.MethodPost, "/campaigns", `{"axis":"combat","axis_kind":"radar","playlist_group":"ranked"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d : %s", rec.Code, rec.Body.String())
	}
	if fake.player != "jg" || fake.started.Axis != "combat" || fake.started.AxisKind != campaign.AxisKindRadar ||
		fake.started.PlaylistGroup != "ranked" || fake.started.TitleSlug != "halo_infinite" || fake.started.UserID != "" {
		t.Fatalf("appel = joueur %q, params %+v", fake.player, fake.started)
	}
}

func TestCampaignHandler_ActiveSansCampagneRendNull(t *testing.T) {
	fake := &fakeCampaignService{err: campaign.ErrNotFound}
	rec := doCampaign(campaignRouter(fake), http.MethodGet, "/campaigns/active", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "null" {
		t.Fatalf("%d %q ; attendu 200 et null", rec.Code, rec.Body.String())
	}
}

func TestCampaignHandler_HistoriqueProjetteLeDelta(t *testing.T) {
	final := 7.0
	fake := &fakeCampaignService{list: []campaign.ImprovementCampaign{{
		ID: "c1", Axis: "combat", AxisKind: campaign.AxisKindRadar, Status: campaign.StatusCompleted,
		SnapshotValue: 5, CurrentValueRaw: &final,
	}}}
	rec := doCampaign(campaignRouter(fake), http.MethodGet, "/campaigns/history", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d : %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"count":1`, `"id":"c1"`, `"snapshot_value":5`} {
		if !bytes.Contains(rec.Body.Bytes(), []byte(want)) {
			t.Errorf("le corps ne contient pas %s : %s", want, rec.Body.String())
		}
	}
}

func TestCampaignHandler_RefusMetierEnReponsesHTTP(t *testing.T) {
	cause := errors.New("boom")
	notFound := &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: cause}
	for _, c := range []struct {
		nom, method, path, body string
		err                     error
		status                  int
		code                    string
	}{
		{"démarrage : déjà active", http.MethodPost, "/campaigns", `{"axis":"a","axis_kind":"radar"}`, campaign.ErrAlreadyActive, http.StatusConflict, "campaign_already_active"},
		{"démarrage : axe invalide", http.MethodPost, "/campaigns", `{"axis":"a","axis_kind":"radar"}`, campaign.ErrInvalidAxis, http.StatusBadRequest, "invalid_axis"},
		{"démarrage : panne", http.MethodPost, "/campaigns", `{"axis":"a","axis_kind":"radar"}`, cause, http.StatusInternalServerError, "start_campaign_error"},
		{"démarrage : joueur introuvable", http.MethodPost, "/campaigns", `{"axis":"a","axis_kind":"radar"}`, notFound, http.StatusNotFound, "player_not_found"},
		{"active : panne", http.MethodGet, "/campaigns/active", "", cause, http.StatusInternalServerError, "get_active_error"},
		{"active : joueur introuvable", http.MethodGet, "/campaigns/active", "", notFound, http.StatusNotFound, "player_not_found"},
		{"historique : panne", http.MethodGet, "/campaigns/history", "", cause, http.StatusInternalServerError, "list_campaigns_error"},
		{"historique : joueur introuvable", http.MethodGet, "/campaigns/history", "", notFound, http.StatusNotFound, "player_not_found"},
		{"détail : campagne absente", http.MethodGet, "/campaigns/c1", "", campaign.ErrNotFound, http.StatusNotFound, "campaign_not_found"},
		{"détail : panne", http.MethodGet, "/campaigns/c1", "", cause, http.StatusInternalServerError, "get_campaign_error"},
		{"détail : joueur introuvable", http.MethodGet, "/campaigns/c1", "", notFound, http.StatusNotFound, "player_not_found"},
		{"pause : campagne absente", http.MethodPost, "/campaigns/c1/pause", "", campaign.ErrNotFound, http.StatusNotFound, "campaign_not_found"},
		{"reprise : transition refusée", http.MethodPost, "/campaigns/c1/resume", "", campaign.ErrInvalidStatus, http.StatusConflict, "invalid_status_transition"},
		{"clôture : panne", http.MethodPost, "/campaigns/c1/close", "", cause, http.StatusInternalServerError, "campaign_transition_error"},
		{"abandon : joueur introuvable", http.MethodPost, "/campaigns/c1/abandon", "", notFound, http.StatusNotFound, "player_not_found"},
	} {
		rec := doCampaign(campaignRouter(&fakeCampaignService{err: c.err}), c.method, c.path, c.body)
		if rec.Code != c.status || !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"`+c.code+`"`)) {
			t.Errorf("%s : %d %s ; attendu %d %s", c.nom, rec.Code, rec.Body.String(), c.status, c.code)
		}
	}
}

func TestCampaignHandler_TransitionsRepondent204(t *testing.T) {
	for _, c := range []struct{ path, called string }{
		{"/campaigns/c1/pause", "pause"}, {"/campaigns/c1/resume", "resume"},
		{"/campaigns/c1/close", "close"}, {"/campaigns/c1/abandon", "abandon"},
	} {
		fake := &fakeCampaignService{}
		rec := doCampaign(campaignRouter(fake), http.MethodPost, c.path, "")
		if rec.Code != http.StatusNoContent || fake.called != c.called || fake.player != "jg" || fake.id != "c1" {
			t.Errorf("%s : %d, appel %q joueur %q id %q", c.path, rec.Code, fake.called, fake.player, fake.id)
		}
	}
}
