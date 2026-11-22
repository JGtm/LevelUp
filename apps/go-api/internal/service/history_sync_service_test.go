package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

type fakeHistoryProfiles struct {
	players []domain.PlayerSummary
	err     error
}

func (f fakeHistoryProfiles) LoadPlayers(...string) ([]domain.PlayerSummary, error) {
	return f.players, f.err
}

type fakeHistoryProgress struct {
	byTitle map[string]port.HistoryProgress
	errFor  map[string]error
	calls   []string
}

func (f *fakeHistoryProgress) ReadHistoryProgress(_ context.Context, titleSlug, gamertag, xuid string) (port.HistoryProgress, error) {
	f.calls = append(f.calls, titleSlug+"/"+gamertag+"/"+xuid)
	if err := f.errFor[titleSlug]; err != nil {
		return port.HistoryProgress{}, err
	}
	return f.byTitle[titleSlug], nil
}

type fakeSyncActivity struct {
	enabled bool
	next    time.Time
	passes  map[string]port.SyncPass
}

func (f fakeSyncActivity) AutoSyncEnabled() bool  { return f.enabled }
func (f fakeSyncActivity) NextCycleAt() time.Time { return f.next }
func (f fakeSyncActivity) LastPass(title, gt string) (port.SyncPass, bool) {
	p, ok := f.passes[title+"/"+gt]
	return p, ok
}

var (
	histT0     = time.Date(2026, 11, 22, 14, 0, 0, 0, time.UTC)
	histOldest = time.Date(2019, 9, 3, 18, 20, 0, 0, time.UTC)
)

func histNames(slug string) string {
	return map[string]string{"halo_infinite": "Halo Infinite", "halo_5": "Halo 5: Guardians"}[slug]
}

func newHistoryService(players []domain.PlayerSummary, prog *fakeHistoryProgress, act fakeSyncActivity) *HistorySyncService {
	return NewHistorySyncService(fakeHistoryProfiles{players: players}, prog, act, histNames)
}

func historyProfile(slug, title, gt, xuid string) domain.PlayerSummary {
	return domain.PlayerSummary{PlayerSlug: slug, TitleSlug: title, Gamertag: gt, XUID: xuid, SyncEnabled: true}
}

// Un joueur suivi sur deux titres : un élément par titre (trié), les autres joueurs et les profils
// d'authentification seule sont ignorés, libellés dans la langue de la requête.
func TestHistorySyncStatus_ParTitreEtLangue(t *testing.T) {
	authOnly := historyProfile("JGtm", "halo_5", "JGtm", "1")
	authOnly.AuthOnly = true
	players := []domain.PlayerSummary{
		historyProfile("JGtm", "halo_5", "JGtm", "2533"),
		historyProfile("JGtm", "halo_infinite", "JGtm", "2533"),
		historyProfile("Autre", "halo_infinite", "Autre", "9999"),
		authOnly,
	}
	prog := &fakeHistoryProgress{byTitle: map[string]port.HistoryProgress{
		"halo_infinite": {MatchCount: 6812, OldestAt: histOldest, Complete: false},
		"halo_5":        {MatchCount: 120, OldestAt: histOldest.Add(-24 * time.Hour), Complete: true},
	}}
	act := fakeSyncActivity{enabled: true, next: histT0, passes: map[string]port.SyncPass{
		"halo_infinite/JGtm": {At: histT0.Add(-10 * time.Minute), MatchesInserted: 200, Status: port.SyncPassOK},
	}}
	svc := newHistoryService(players, prog, act)

	cases := []struct {
		locale   string
		catching string
		complete string
		pass     string
	}{
		{"fr", "Rattrapage en cours", "Complet", "Réussie"},
		{"en", "Catching up", "Complete", "Succeeded"},
	}
	for _, c := range cases {
		t.Run(c.locale, func(t *testing.T) {
			got, err := svc.HistorySyncStatus(ctxkeys.WithLocale(context.Background(), c.locale), "JGtm")
			if err != nil {
				t.Fatalf("HistorySyncStatus: %v", err)
			}
			if len(got.Titles) != 2 || got.Titles[0].TitleSlug != "halo_5" || got.Titles[1].TitleSlug != "halo_infinite" {
				t.Fatalf("titres = %+v, attendu halo_5 puis halo_infinite", got.Titles)
			}
			h5, hi := got.Titles[0], got.Titles[1]
			if h5.State != domain.HistoryStateComplete || h5.StateLabel != c.complete || h5.TitleName != "Halo 5: Guardians" || h5.MatchCount != 120 {
				t.Errorf("halo_5 = %+v", h5)
			}
			if h5.LastPass != nil {
				t.Errorf("halo_5 : aucune passe connue, LastPass = %+v", h5.LastPass)
			}
			if hi.State != domain.HistoryStateCatchingUp || hi.StateLabel != c.catching || hi.MatchCount != 6812 {
				t.Errorf("halo_infinite = %+v", hi)
			}
			if hi.OldestMatchAt == nil || !hi.OldestMatchAt.Equal(histOldest) {
				t.Errorf("plus ancien = %v, attendu %v", hi.OldestMatchAt, histOldest)
			}
			if hi.LastPass == nil || hi.LastPass.MatchesAdded != 200 || hi.LastPass.Status != "ok" || hi.LastPass.StatusLabel != c.pass {
				t.Errorf("dernière passe = %+v", hi.LastPass)
			}
			if !got.AutoSyncEnabled || got.NextCycleAt == nil || !got.NextCycleAt.Equal(histT0) {
				t.Errorf("calendrier = (%v, %v)", got.AutoSyncEnabled, got.NextCycleAt)
			}
		})
	}
	for _, call := range prog.calls {
		if call == "halo_infinite/Autre/9999" {
			t.Error("l'état d'un autre joueur ne doit pas être lu")
		}
	}
}

// Aucun titre suivi : liste vide (jamais nulle), 200.
func TestHistorySyncStatus_AucunTitreSuivi(t *testing.T) {
	svc := newHistoryService([]domain.PlayerSummary{historyProfile("Autre", "halo_infinite", "Autre", "9")}, &fakeHistoryProgress{}, fakeSyncActivity{})
	got, err := svc.HistorySyncStatus(context.Background(), "JGtm")
	if err != nil {
		t.Fatalf("HistorySyncStatus: %v", err)
	}
	if got.Titles == nil || len(got.Titles) != 0 {
		t.Errorf("Titles = %#v, attendu une liste vide non nulle", got.Titles)
	}
	if got.NextCycleAt != nil {
		t.Errorf("NextCycleAt = %v, attendu absent tant que la boucle n'a pas démarré", got.NextCycleAt)
	}
}

// Une lecture en échec sur un titre rend ce titre « inconnu » sans faire échouer la réponse ; un titre
// en pause se dit en pause, ses compteurs lus tout de même.
func TestHistorySyncStatus_EchecDeLectureEtPause(t *testing.T) {
	paused := historyProfile("JGtm", "halo_5", "JGtm", "2533")
	paused.SyncEnabled = false
	players := []domain.PlayerSummary{historyProfile("JGtm", "halo_infinite", "JGtm", "2533"), paused}
	prog := &fakeHistoryProgress{
		errFor:  map[string]error{"halo_infinite": errors.New("base verrouillée")},
		byTitle: map[string]port.HistoryProgress{"halo_5": {MatchCount: 50, Complete: false}},
	}
	got, err := newHistoryService(players, prog, fakeSyncActivity{}).HistorySyncStatus(ctxkeys.WithLocale(context.Background(), "fr"), "JGtm")
	if err != nil {
		t.Fatalf("HistorySyncStatus: %v", err)
	}
	h5, hi := got.Titles[0], got.Titles[1]
	if hi.State != domain.HistoryStateUnknown || hi.StateLabel != "Inconnu" || hi.MatchCount != 0 {
		t.Errorf("halo_infinite en échec de lecture = %+v", hi)
	}
	if h5.State != domain.HistoryStatePaused || h5.StateLabel != "Synchronisation en pause" || h5.MatchCount != 50 {
		t.Errorf("halo_5 en pause = %+v", h5)
	}
}

func TestHistorySyncStatus_ProfilsIllisibles(t *testing.T) {
	svc := NewHistorySyncService(fakeHistoryProfiles{err: errors.New("db_profiles illisible")}, &fakeHistoryProgress{}, fakeSyncActivity{}, nil)
	if _, err := svc.HistorySyncStatus(context.Background(), "JGtm"); err == nil {
		t.Fatal("attendu une erreur quand les profils sont illisibles")
	}
}

// Chaque issue de passe a son libellé dans les deux langues ; sans nom de titre déclaré, l'identifiant
// du titre en tient lieu.
func TestHistorySyncStatus_LibellesDesPasses(t *testing.T) {
	cases := []struct {
		status port.SyncPassStatus
		fr     string
		en     string
	}{
		{port.SyncPassOK, "Réussie", "Succeeded"},
		{port.SyncPassPartial, "Partielle", "Partial"},
		{port.SyncPassFailed, "Échec", "Failed"},
		{port.SyncPassSkipped, "Différée", "Deferred"},
	}
	for _, c := range cases {
		act := fakeSyncActivity{passes: map[string]port.SyncPass{"halo_infinite/JGtm": {At: histT0, Status: c.status}}}
		svc := NewHistorySyncService(
			fakeHistoryProfiles{players: []domain.PlayerSummary{historyProfile("JGtm", "", "JGtm", "2533")}},
			&fakeHistoryProgress{}, act, nil)
		for loc, want := range map[string]string{"fr": c.fr, "en": c.en} {
			got, err := svc.HistorySyncStatus(ctxkeys.WithLocale(context.Background(), loc), "JGtm")
			if err != nil || len(got.Titles) != 1 {
				t.Fatalf("HistorySyncStatus: %v, %+v", err, got)
			}
			if got.Titles[0].LastPass.StatusLabel != want {
				t.Errorf("%s/%s : libellé %q, attendu %q", c.status, loc, got.Titles[0].LastPass.StatusLabel, want)
			}
			if got.Titles[0].TitleSlug != "halo_infinite" || got.Titles[0].TitleName != "halo_infinite" {
				t.Errorf("titre par défaut = %+v", got.Titles[0])
			}
		}
	}
}
