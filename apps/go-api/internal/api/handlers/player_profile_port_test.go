// Package handlers_test — player_profile_port_test.go : les routes /profile et /activity-calendar sur un
// faux profile.PlayerProfileService : bornes des fenêtres, refus typés, projection.
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/profile"
)

// fakeProfileService note la fenêtre demandée et rend la valeur préparée.
type fakeProfileService struct {
	err error

	gotPlayer, gotTitle string
	gotWindow           int
	gotSince, gotUntil  time.Time
}

func (f *fakeProfileService) Build(_ context.Context, player, title string, window int, _ time.Time) (*profile.PlayerProfile, error) {
	f.gotPlayer, f.gotTitle, f.gotWindow = player, title, window
	if f.err != nil {
		return nil, f.err
	}
	return &profile.PlayerProfile{UserID: "xuid-1", TitleSlug: title}, nil
}

func (f *fakeProfileService) ActivityCalendar(_ context.Context, player string, since, until time.Time) (profile.ActivityCalendar, error) {
	f.gotPlayer, f.gotSince, f.gotUntil = player, since, until
	if f.err != nil {
		return profile.ActivityCalendar{}, f.err
	}
	return profile.ActivityCalendar{Since: "2026-08-01", Until: "2026-10-30", Days: []profile.ActivityDay{}}, nil
}

func profileRouter(svc profile.PlayerProfileService) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewPlayerProfileHandler(svc, "")
	r.Route("/api/v1/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	return r
}

func TestPlayerProfileHandler_FenetreParDefautEtBornes(t *testing.T) {
	for _, c := range []struct {
		query string
		want  int
	}{
		{"", handlers.ProfileWindowDays},
		{"?window_days=abc", handlers.ProfileWindowDays},
		{"?window_days=0", handlers.ProfileWindowDays},
		{"?window_days=3", 7},
		{"?window_days=45", 45},
		{"?window_days=999", 120},
	} {
		fake := &fakeProfileService{}
		code, body := getJSON(t, profileRouter(fake), "/api/v1/players/jg/profile"+c.query, "")
		if code != http.StatusOK || body["user_id"] != "xuid-1" {
			t.Fatalf("%q : %d %v", c.query, code, body)
		}
		if fake.gotWindow != c.want || fake.gotPlayer != "jg" || fake.gotTitle != "halo_infinite" {
			t.Errorf("%q : fenêtre %d, joueur %q, titre %q ; attendu %d, jg, halo_infinite", c.query, fake.gotWindow, fake.gotPlayer, fake.gotTitle, c.want)
		}
	}
}

func TestPlayerProfileHandler_CalendrierBorneLaFenetre(t *testing.T) {
	for _, c := range []struct {
		query string
		days  int
	}{
		{"", handlers.ActivityCalendarWindowDays},
		{"?days=2", 7},
		{"?days=60", 60},
		{"?days=500", 180},
	} {
		fake := &fakeProfileService{}
		code, body := getJSON(t, profileRouter(fake), "/api/v1/players/jg/activity-calendar"+c.query, "")
		if code != http.StatusOK || body["since"] != "2026-08-01" {
			t.Fatalf("%q : %d %v", c.query, code, body)
		}
		got := int(fake.gotUntil.Sub(fake.gotSince).Hours()/24 + 0.5)
		if got != c.days {
			t.Errorf("%q : fenêtre de %d jours, attendu %d", c.query, got, c.days)
		}
	}
}

func TestPlayerProfileHandler_Erreurs(t *testing.T) {
	cause := errors.New("boom")
	notFound := &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: cause}
	for _, c := range []struct {
		path   string
		err    error
		status int
		code   string
	}{
		{"profile", notFound, http.StatusNotFound, "player_not_found"},
		{"profile", cause, http.StatusInternalServerError, "build_profile_error"},
		{"activity-calendar", notFound, http.StatusNotFound, "player_not_found"},
		{"activity-calendar", cause, http.StatusInternalServerError, "activity_calendar_error"},
	} {
		code, body := getJSON(t, profileRouter(&fakeProfileService{err: c.err}), "/api/v1/players/jg/"+c.path, "")
		if code != c.status || body["code"] != c.code {
			t.Errorf("%s : %d %v ; attendu %d %s", c.path, code, body, c.status, c.code)
		}
	}
}
