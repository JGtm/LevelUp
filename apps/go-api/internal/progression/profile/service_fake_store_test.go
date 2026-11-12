package profile

// service_fake_store_test.go — le service de profil sur un faux port.ProfileStore : aucune base.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/prestige"
)

type fakeStore struct {
	title string

	lusr       port.ProfileLUSR
	lusrErr    error
	series     []float64
	points     []port.ProfileLUSRPoint
	count      int
	countErr   error
	radar      port.ProfileRadarBase
	windowIDs  []string
	windowErr  error
	awards     []port.ProfileAwardTotal
	awardsErr  error
	starts     []time.Time
	days       []port.ProfileMatchTime
	components []port.ProfileComponentAggregate
	compErr    error
	templates  []prestige.Template
	tplAvail   bool
	tplCalls   int
	objective  map[string]narrative.ObjectiveIndexInput
	objCalls   int
}

func (f *fakeStore) TitleSlug() string { return f.title }
func (f *fakeStore) LatestLUSR(context.Context) (port.ProfileLUSR, error) {
	return f.lusr, f.lusrErr
}
func (f *fakeStore) LUSRSeries(context.Context, time.Time, time.Time) ([]float64, error) {
	return f.series, nil
}
func (f *fakeStore) LUSRSeriesPoints(context.Context, time.Time, time.Time) ([]port.ProfileLUSRPoint, error) {
	return f.points, nil
}
func (f *fakeStore) LUSRComponents(context.Context) ([]port.ProfileComponentAggregate, error) {
	return f.components, f.compErr
}
func (f *fakeStore) CountWindowMatches(context.Context, string, time.Time, time.Time) (int, error) {
	return f.count, f.countErr
}
func (f *fakeStore) RadarBase(context.Context, string, time.Time, time.Time) (port.ProfileRadarBase, error) {
	return f.radar, nil
}
func (f *fakeStore) WindowMatchIDs(context.Context, time.Time, time.Time) ([]string, error) {
	return f.windowIDs, f.windowErr
}
func (f *fakeStore) PlayerWindowMatchIDs(context.Context, string, time.Time, time.Time) ([]string, error) {
	return f.windowIDs, f.windowErr
}
func (f *fakeStore) AwardTotals(context.Context, string, []string) ([]port.ProfileAwardTotal, error) {
	return f.awards, f.awardsErr
}
func (f *fakeStore) MatchStartTimes(context.Context, string, time.Time, time.Time) ([]time.Time, error) {
	return f.starts, nil
}
func (f *fakeStore) MatchDays(context.Context, string, time.Time, time.Time) ([]port.ProfileMatchTime, error) {
	return f.days, nil
}
func (f *fakeStore) ObjectiveIndexInputs(context.Context, []string, []string) (map[string]narrative.ObjectiveIndexInput, error) {
	f.objCalls++
	return f.objective, nil
}
func (f *fakeStore) KillDeathEvents(context.Context, []string) ([]canonical.HighlightEvent, error) {
	return nil, nil
}
func (f *fakeStore) SuggestionTemplates(context.Context, string) ([]prestige.Template, bool, error) {
	f.tplCalls++
	return f.templates, f.tplAvail, nil
}

var fakeNow = time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

func ptr(v float64) *float64 { return &v }

func TestService_LoadSurFauxMagasin(t *testing.T) {
	st := &fakeStore{lusr: port.ProfileLUSR{Mu: 1510, Sigma: 90, MatchesCount: MinMatchesForRating + 1}}
	for i := 0; i < 5; i++ {
		st.series = append(st.series, 1500+float64(i))
	}
	prof, err := NewService(st).Load(context.Background(), "u", "halo_infinite", 30, fakeNow)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if prof.LUSR.Mu != 1510 || prof.LUSR.Sigma != 90 || prof.Tier.IsEmpty() {
		t.Errorf("profil = %+v", prof.LUSR)
	}
}

func TestService_LoadRemonteLEchecDeLecture(t *testing.T) {
	boom := errors.New("lecture impossible")
	_, err := NewService(&fakeStore{lusrErr: boom}).Load(context.Background(), "u", "halo_infinite", 30, fakeNow)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "loadLUSRSnapshot") {
		t.Fatalf("err = %v", err)
	}
}

func TestService_BuildProfileSansMagasin(t *testing.T) {
	if _, err := NewService(nil).BuildProfile(context.Background(), "u", "halo_infinite", 30, fakeNow); err == nil {
		t.Fatal("un service sans magasin doit refuser BuildProfile")
	}
}

func TestService_RadarEtEngagementDepuisLeMagasin(t *testing.T) {
	st := &fakeStore{
		title: "halo_5", // sans capacité objectifs : aucune lecture d'index
		count: MinMatchesForProfile,
		radar: port.ProfileRadarBase{AvgKills: ptr(12), AvgDeaths: ptr(8), AvgAssists: ptr(4), AvgScore: ptr(1500), AvgKillingSpree: ptr(5), MatchCount: 20},
		starts: []time.Time{
			fakeNow.AddDate(0, 0, -20), fakeNow.AddDate(0, 0, -19), fakeNow.AddDate(0, 0, -10),
		},
	}
	svc := NewService(st)
	axes, err := svc.computeRadarAxes(context.Background(), "u", fakeNow.AddDate(0, 0, -30), fakeNow)
	if err != nil {
		t.Fatalf("computeRadarAxes: %v", err)
	}
	if axes[narrative.AxisCombat] != 12 || axes[narrative.AxisSurvival] != 4 || axes[narrative.AxisImpact] != 50 {
		t.Errorf("axes = %v", axes)
	}
	if _, objN := svc.computeObjectiveIndexAxis(context.Background(), "u", fakeNow.AddDate(0, 0, -30), fakeNow); objN != 0 || st.objCalls != 0 {
		t.Errorf("sans capacité objectifs : n=%d, lectures d'index=%d, attendu 0 et 0", objN, st.objCalls)
	}

	snap, err := svc.computeEngagementSimple(context.Background(), "u", fakeNow.AddDate(0, 0, -30), fakeNow)
	if err != nil {
		t.Fatalf("computeEngagementSimple: %v", err)
	}
	if snap.MaxGapDays != 9 || snap.Tier == "" {
		t.Errorf("engagement = %+v, attendu un écart maximal de 9 jours", snap)
	}
}

func TestService_RadarSansMatchEstVide(t *testing.T) {
	axes, err := NewService(&fakeStore{}).computeRadarAxes(context.Background(), "u", fakeNow.AddDate(0, 0, -30), fakeNow)
	if err != nil || axes != nil {
		t.Errorf("axes = %v err = %v, attendu nil sans match", axes, err)
	}
}

func TestService_ComposantesDuMagasin(t *testing.T) {
	st := &fakeStore{components: []port.ProfileComponentAggregate{
		{Name: "win_factor", Avg: ptr(0.5), Top20: ptr(0.8), LastAvg: ptr(0.7), FirstAvg: ptr(0.4), N: 30},
		{Name: "accuracy_delta", Avg: ptr(0.9), N: 0},                   // sans match : omise
		{Name: "medal_exploit", Avg: ptr(0.3), LastAvg: ptr(0.9), N: 5}, // tendance inconnue sans première moyenne
	}}
	rows, err := NewService(st).loadLUSRComponentsBreakdown(context.Background())
	if err != nil {
		t.Fatalf("loadLUSRComponentsBreakdown: %v", err)
	}
	if _, ok := rows["accuracy_delta"]; ok || len(rows) != 2 {
		t.Errorf("rows = %v, attendu win_factor et medal_exploit seulement", rows)
	}
	if got := rows["win_factor"]; got.currentAvg != 0.5 || got.top20 != 0.8 || got.trend < 0.299 || got.trend > 0.301 {
		t.Errorf("win_factor = %+v", got)
	}
	if rows["medal_exploit"].trend != 0 {
		t.Errorf("tendance sans première moyenne = %v, attendu 0", rows["medal_exploit"].trend)
	}

	st.compErr = errors.New("exclusions illisibles")
	if _, err := NewService(st).loadLUSRComponentsBreakdown(context.Background()); err == nil {
		t.Error("l'échec de lecture doit remonter")
	}
}

func TestService_SuggestionsFiltrentParLeviers(t *testing.T) {
	tpl := func(id string, long bool, comps ...string) prestige.Template {
		return prestige.Template{ID: id, IsLongTerm: long, LUSRComponents: comps}
	}
	st := &fakeStore{tplAvail: true, templates: []prestige.Template{
		tpl("b", false, "win_factor"),
		tpl("a", true, "win_factor", "accuracy_delta"),
		tpl("c", false, "medal_exploit"), // ne recoupe pas les leviers
		tpl("d", false, "accuracy_delta"),
		tpl("e", false, "win_factor"),
	}}
	prof := &PlayerProfile{Leverages: []ProgressionLeverage{{Component: "win_factor"}, {Component: "accuracy_delta"}}}
	ctx := context.Background()

	if err := NewService(st).selectSuggestedChallenges(ctx, prof, "halo_infinite"); err != nil {
		t.Fatalf("selectSuggestedChallenges: %v", err)
	}
	var got []string
	for _, s := range prof.SuggestedChallenges {
		got = append(got, s.TemplateID)
	}
	if strings.Join(got, ",") != "a,b,d" {
		t.Errorf("suggestions = %v, attendu [a b d] : long terme d'abord, puis par identifiant, 3 au plus, hors c", got)
	}

	// Sans levier : aucune lecture des modèles.
	st.tplCalls = 0
	if err := NewService(st).selectSuggestedChallenges(ctx, &PlayerProfile{}, "halo_infinite"); err != nil || st.tplCalls != 0 {
		t.Errorf("sans levier : err=%v lectures=%d, attendu aucune", err, st.tplCalls)
	}
	// Référentiels indisponibles : aucune suggestion.
	st.tplAvail = false
	prof2 := &PlayerProfile{Leverages: prof.Leverages}
	if err := NewService(st).selectSuggestedChallenges(ctx, prof2, "halo_infinite"); err != nil || len(prof2.SuggestedChallenges) != 0 {
		t.Errorf("référentiels absents : err=%v suggestions=%v", err, prof2.SuggestedChallenges)
	}
}

func TestService_CalendrierDActiviteDedoublonneParJour(t *testing.T) {
	d1 := time.Date(2026, 5, 10, 23, 30, 0, 0, time.UTC)
	st := &fakeStore{days: []port.ProfileMatchTime{
		{MatchID: "m1", At: d1}, {MatchID: "m1", At: d1}, {MatchID: "m2", At: d1.Add(time.Minute)},
		{MatchID: "m3", At: d1.Add(2 * time.Hour)}, // lendemain UTC
	}}
	cal, err := NewService(st).LoadActivityCalendar(context.Background(), "u", d1.AddDate(0, 0, -5), d1.AddDate(0, 0, 5))
	if err != nil {
		t.Fatalf("LoadActivityCalendar: %v", err)
	}
	if len(cal.Days) != 2 || cal.Days[0].Date != "2026-05-10" || cal.Days[0].Count != 2 || cal.Days[1].Count != 1 {
		t.Errorf("jours = %+v, attendu 2 jours : 2 matchs distincts puis 1", cal.Days)
	}
	if _, err := NewService(nil).LoadActivityCalendar(context.Background(), "u", d1, d1); err == nil {
		t.Error("sans magasin : erreur attendue")
	}
}
