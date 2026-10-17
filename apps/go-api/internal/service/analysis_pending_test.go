package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

type probeRepoFactice struct {
	probes map[string]domain.AnalysisProbe
	err    error
	appels int
	lots   [][]string
}

func (r *probeRepoFactice) LoadAnalysisProbes(_ context.Context, ids []string) (map[string]domain.AnalysisProbe, error) {
	r.appels++
	r.lots = append(r.lots, append([]string(nil), ids...))
	return r.probes, r.err
}

var instantService = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func resolveurDeTest(repo port.AnalysisProbeRepository, replayExpected bool) *AnalysisPendingResolver {
	return NewAnalysisPendingResolver(repo, replayExpected).WithClock(func() time.Time { return instantService })
}

func synchroIlYA(d time.Duration, mod func(*domain.AnalysisProbe)) domain.AnalysisProbe {
	t := instantService.Add(-d)
	p := domain.AnalysisProbe{FirstSyncAt: &t, KillSourceDone: true, HasCitationRows: true}
	if mod != nil {
		mod(&p)
	}
	return p
}

func TestAnalysisPendingResolver_Resolve(t *testing.T) {
	repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{
		"fini":     synchroIlYA(time.Hour, nil),
		"sansArme": synchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.KillSourceDone = false }),
		"ancien":   synchroIlYA(25*time.Hour, func(p *domain.AnalysisProbe) { p.KillSourceDone = false }),
	}}
	r := resolveurDeTest(repo, true)
	replays := port.ReplayAvailability{title.FilmShortMatchID("fini"): {}}
	got := r.Resolve(context.Background(), []string{"fini", "sansArme", "ancien", "absent"},
		ReplayLookup{Known: true, Has: replays.Has})
	if len(got) != 1 || !reflect.DeepEqual(got["sansArme"], []string{domain.AnalysisPartKillSource, domain.AnalysisPartReplay}) {
		t.Fatalf("pending = %v, want {sansArme: [kill_source replay]}", got)
	}
	if repo.appels != 1 {
		t.Fatalf("lectures = %d, want 1 par lot", repo.appels)
	}
}

func TestAnalysisPendingResolver_NilEtLotVide(t *testing.T) {
	var nul *AnalysisPendingResolver
	if got := nul.Resolve(context.Background(), []string{"m"}, ReplayLookup{}); got != nil {
		t.Fatalf("résolveur nil : %v", got)
	}
	repo := &probeRepoFactice{}
	if got := resolveurDeTest(repo, true).Resolve(context.Background(), nil, ReplayLookup{}); got != nil || repo.appels != 0 {
		t.Fatalf("lot vide : %v, %d lectures", got, repo.appels)
	}
	if NewAnalysisPendingResolver(nil, true) != nil {
		t.Fatal("dépôt nil : le constructeur doit rendre nil")
	}
}

func TestAnalysisPendingResolver_LectureEnEchec(t *testing.T) {
	repo := &probeRepoFactice{err: errors.New("boom")}
	if got := resolveurDeTest(repo, true).Resolve(context.Background(), []string{"m"}, ReplayLookup{}); len(got) != 0 {
		t.Fatalf("lecture en échec : aucun indicateur attendu, got %v", got)
	}
}

func TestHomeService_AnalysisPending_UneLectureEtDeuxListes(t *testing.T) {
	repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{
		"m1": synchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.HasCitationRows = false }),
		"m2": synchroIlYA(time.Hour, nil),
	}}
	s := &HomeService{analysisPending: resolveurDeTest(repo, false)}
	recent := []domain.RecentMatchItem{{MatchID: "m1"}, {MatchID: "m2"}}
	favorites := []domain.RecentMatchItem{{MatchID: "m1", IsFavorite: true}}
	s.applyAnalysisPendingToRecentItems(context.Background(), nil, recent, favorites)

	wantParts := []string{domain.AnalysisPartCitations}
	if !recent[0].AnalysisPending || recent[1].AnalysisPending || !favorites[0].AnalysisPending {
		t.Fatalf("recent=%+v favoris=%+v", recent, favorites)
	}
	if !reflect.DeepEqual(recent[0].AnalysisPendingParts, wantParts) || !reflect.DeepEqual(favorites[0].AnalysisPendingParts, wantParts) ||
		recent[1].AnalysisPendingParts != nil {
		t.Fatalf("causes: recent=%v favoris=%v", recent, favorites)
	}
	if repo.appels != 1 || len(repo.lots[0]) != 2 {
		t.Fatalf("un seul lot dédoublonné attendu, got %d lectures, lots=%v", repo.appels, repo.lots)
	}
}

func TestHomeService_AnalysisPending_SansResolveur(t *testing.T) {
	s := &HomeService{}
	items := []domain.RecentMatchItem{{MatchID: "m1"}}
	s.applyAnalysisPendingToRecentItems(context.Background(), nil, items)
	if items[0].AnalysisPending {
		t.Fatal("titre sans la capability : le champ doit rester faux")
	}
}

func TestHomeService_AnalysisPending_RejeuAbsentEtEnsembleNonResolu(t *testing.T) {
	repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{"m1": synchroIlYA(time.Hour, nil)}}
	s := &HomeService{analysisPending: resolveurDeTest(repo, true)}

	// Ensemble résolu et vide : le rejeu manque -> en cours.
	items := []domain.RecentMatchItem{{MatchID: "m1"}}
	s.applyAnalysisPendingToRecentItems(context.Background(), port.ReplayAvailability{}, items)
	if !items[0].AnalysisPending || !reflect.DeepEqual(items[0].AnalysisPendingParts, []string{domain.AnalysisPartReplay}) {
		t.Fatalf("rejeu absent d'un ensemble résolu : attendu en cours pour replay, got %+v", items[0])
	}
	// Ensemble non résolu (nil) : la condition rejeu est ignorée.
	items = []domain.RecentMatchItem{{MatchID: "m1"}}
	s.applyAnalysisPendingToRecentItems(context.Background(), nil, items)
	if items[0].AnalysisPending {
		t.Fatal("ensemble de rejeux non résolu : ne pas conclure à un rejeu manquant")
	}
}

func TestMatchViewService_HeaderAnalysisPending(t *testing.T) {
	repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{"m1": synchroIlYA(time.Hour, nil)}}
	s := &MatchViewService{analysisPending: resolveurDeTest(repo, true), replaySvc: &stubReplayService{}}

	h := domain.MatchViewHeader{MatchID: "m1", ReplayAvailable: false}
	s.applyHeaderAnalysisPending(context.Background(), &h)
	if !h.AnalysisPending || !reflect.DeepEqual(h.AnalysisPendingParts, []string{domain.AnalysisPartReplay}) {
		t.Fatalf("rejeu absent, service de rejeu câblé : attendu en cours pour replay, got %+v", h)
	}
	h = domain.MatchViewHeader{MatchID: "m1", ReplayAvailable: true}
	s.applyHeaderAnalysisPending(context.Background(), &h)
	if h.AnalysisPending {
		t.Fatal("tout est fait : attendu faux")
	}
	// Service de rejeu non câblé : l'absence ne dit rien.
	s.replaySvc = nil
	h = domain.MatchViewHeader{MatchID: "m1"}
	s.applyHeaderAnalysisPending(context.Background(), &h)
	if h.AnalysisPending {
		t.Fatal("service de rejeu non câblé : ne pas conclure à un rejeu manquant")
	}
	// Sans résolveur.
	s.analysisPending = nil
	h = domain.MatchViewHeader{MatchID: "m1"}
	s.applyHeaderAnalysisPending(context.Background(), &h)
	if h.AnalysisPending {
		t.Fatal("sans résolveur : faux")
	}
}

type observateurFactice struct{ match string }

func (o observateurFactice) DecodingMatchID() string { return o.match }

func TestMatchViewService_HeaderAnalysisState(t *testing.T) {
	repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{
		"m1": synchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.KillSourceDone = false }),
		"m2": synchroIlYA(time.Hour, nil),
	}}
	cas := []struct {
		nom      string
		match    string
		decodant port.FilmDecodeObserver
		want     string
	}{
		{"décodage de ce match", "m1", observateurFactice{"m1"}, domain.AnalysisStateInProgress},
		{"décodage d'un autre match", "m1", observateurFactice{"m9"}, domain.AnalysisStateQueued},
		{"aucun décodage", "m1", observateurFactice{""}, domain.AnalysisStateQueued},
		{"observateur non câblé", "m1", nil, domain.AnalysisStateQueued},
		{"analyse finie : champ absent même si le match est décodé", "m2", observateurFactice{"m2"}, ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			r := resolveurDeTest(repo, false).WithDecodeObserver(c.decodant)
			s := &MatchViewService{analysisPending: r}
			h := domain.MatchViewHeader{MatchID: c.match}
			s.applyHeaderAnalysisPending(context.Background(), &h)
			if h.AnalysisState != c.want {
				t.Fatalf("AnalysisState = %q, want %q (header %+v)", h.AnalysisState, c.want, h)
			}
		})
	}
}
