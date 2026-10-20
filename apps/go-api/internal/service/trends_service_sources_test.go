package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/equipmentusage"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

const (
	trendsMe   = "xuid-moi"
	trendsAlly = "xuid-allie"
)

// fakeTrendsObjectives sert les rôles et les prises nettes.
type fakeTrendsObjectives struct {
	rows     []sessionusage.ObjectiveRow
	rolesErr error
	grabsErr error
	roleCall atomic.Int32
	grabCall atomic.Int32
}

func (f *fakeTrendsObjectives) LoadObjectiveRoleRows(_ context.Context, _ []string) ([]sessionusage.ObjectiveRow, error) {
	f.roleCall.Add(1)
	return f.rows, f.rolesErr
}

func (f *fakeTrendsObjectives) LoadFlagGrabsNet(_ context.Context, _ []string) ([]sessionusage.FlagGrabsNetRow, error) {
	f.grabCall.Add(1)
	return nil, f.grabsErr
}

// fakeTrendsUsage sert l'usage de l'équipement et les participants.
type fakeTrendsUsage struct {
	players        []sessionusage.PlayerRow
	participants   []sessionusage.ParticipantRow
	playersErr     error
	participantErr error
	playersCall    atomic.Int32
	partCall       atomic.Int32
	partIDs        []string
}

func (f *fakeTrendsUsage) LoadUsageFilms(context.Context, []string) (map[string]sessionusage.FilmRow, error) {
	return nil, nil
}

func (f *fakeTrendsUsage) LoadUsagePlayers(context.Context, []string) ([]sessionusage.PlayerRow, error) {
	f.playersCall.Add(1)
	return f.players, f.playersErr
}

func (f *fakeTrendsUsage) LoadPadTiers(context.Context, []string) ([]sessionusage.PadTierRow, error) {
	return nil, nil
}

func (f *fakeTrendsUsage) LoadParticipants(_ context.Context, ids []string) ([]sessionusage.ParticipantRow, error) {
	f.partCall.Add(1)
	f.partIDs = append([]string(nil), ids...)
	return f.participants, f.participantErr
}

// fakeTrendsMedals sert les médailles et leurs noms.
type fakeTrendsMedals struct {
	rows       []port.MedalRow
	medalsErr  error
	defsErr    error
	medalCall  atomic.Int32
	defsCall   atomic.Int32
	defsLocale string
}

func (f *fakeTrendsMedals) LoadMedalsForMatchesByXUID(context.Context, string, port.MedalsByXUIDFilters) ([]port.MedalRow, error) {
	f.medalCall.Add(1)
	return f.rows, f.medalsErr
}

func (f *fakeTrendsMedals) LookupByIDs(_ context.Context, ids []int64, locale string) (map[int64]port.MedalDefinitionRow, error) {
	f.defsCall.Add(1)
	f.defsLocale = locale
	out := map[int64]port.MedalDefinitionRow{}
	for _, id := range ids {
		out[id] = port.MedalDefinitionRow{MedalID: id, Label: fmt.Sprintf("%s-%d", locale, id)}
	}
	return out, f.defsErr
}

// trendsSourceRows : six matchs solo récents m0..m5.
func trendsSourceRows() []canonical.PlayerMatchRow {
	rows := make([]canonical.PlayerMatchRow, 0, 6)
	for i := 0; i < 6; i++ {
		rows = append(rows, trendsRow(fmt.Sprintf("m%d", i), i+1, false, false))
	}
	return rows
}

// objectiveFixtures : rôles sur m0..m4 (le joueur et un allié), camps connus pour ces cinq matchs.
func objectiveFixtures() (*fakeTrendsObjectives, *fakeTrendsUsage) {
	obj := &fakeTrendsObjectives{}
	use := &fakeTrendsUsage{}
	zero := 0
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("m%d", i)
		obj.rows = append(obj.rows,
			sessionusage.ObjectiveRow{MatchID: id, XUID: trendsMe, Take: 1},
			sessionusage.ObjectiveRow{MatchID: id, XUID: trendsAlly, Take: 3})
		use.participants = append(use.participants,
			sessionusage.ParticipantRow{MatchID: id, XUID: trendsMe, TeamID: &zero, PresentAtCompletion: true},
			sessionusage.ParticipantRow{MatchID: id, XUID: trendsAlly, TeamID: &zero, PresentAtCompletion: true})
	}
	return obj, use
}

// equipmentFixtures : usage mesuré sur m0..m3.
func equipmentFixtures() *fakeTrendsUsage {
	fam := equipmentusage.EquipmentFamilyPowerupCamo
	use := &fakeTrendsUsage{}
	for i := 0; i < 4; i++ {
		use.players = append(use.players, sessionusage.PlayerRow{
			MatchID: fmt.Sprintf("m%d", i), XUID: trendsMe, CamoEpisodes: 1,
			TakenByFamily: map[string]int{fam: 2}, KeptByFamily: map[string]int{fam: 1},
		})
	}
	return use
}

func medalFixtures() *fakeTrendsMedals {
	return &fakeTrendsMedals{rows: []port.MedalRow{
		{XUID: trendsMe, MatchID: "m0", MedalID: 5, Count: 2},
		{XUID: trendsMe, MatchID: "m1", MedalID: 6, Count: 1},
	}}
}

func trendsSourcesService() *TrendsService {
	return newTrendsTestService(&fakeTrendsMatches{rows: trendsSourceRows()}).WithPlayerXUID(trendsMe)
}

func page(t *testing.T, s *TrendsService, locale string) domain.TrendsPageResponse {
	t.Helper()
	resp, err := s.GetPage(context.Background(), domain.TrendsQueryRequest{Locale: locale})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	return resp
}

func hasKey(resp domain.TrendsPageResponse, key string) bool {
	for _, ind := range resp.Indicators {
		if ind.Key == key {
			return true
		}
	}
	return false
}

func TestTrendsSources_AbsentesBlocsAbsents(t *testing.T) {
	resp := page(t, trendsSourcesService(), "")
	if hasKey(resp, domain.TrendsKeyObjectiveTakeShare) || hasKey(resp, domain.TrendsKeyEquipmentUsedShare) || len(resp.Medals) != 0 {
		t.Fatalf("aucune source : blocs attendus absents (%d médailles)", len(resp.Medals))
	}
}

func TestTrendsSources_ToutesLuesUneFoisEtMedaillesNommees(t *testing.T) {
	obj, use := objectiveFixtures()
	use.players = equipmentFixtures().players
	med := medalFixtures()
	s := trendsSourcesService().WithObjectives(obj, use).WithEquipmentUsage(use).WithMedals(med, med)
	resp := page(t, s, "en-US")

	if !hasKey(resp, domain.TrendsKeyObjectiveTakeShare) || !hasKey(resp, domain.TrendsKeyEquipmentUsedShare) {
		t.Fatalf("indicateurs d'objectif et d'équipement attendus")
	}
	if obj.roleCall.Load() != 1 || obj.grabCall.Load() != 1 || use.playersCall.Load() != 1 ||
		use.partCall.Load() != 1 || med.medalCall.Load() != 1 || med.defsCall.Load() != 1 {
		t.Fatalf("appels : rôles %d, prises %d, usage %d, participants %d, médailles %d, noms %d",
			obj.roleCall.Load(), obj.grabCall.Load(), use.playersCall.Load(), use.partCall.Load(),
			med.medalCall.Load(), med.defsCall.Load())
	}
	if len(resp.Medals) != 4 {
		t.Fatalf("blocs de médailles = %d", len(resp.Medals))
	}
	b := resp.Medals[3]
	if b.Days != 7 || len(b.Rows) != 2 || b.Rows[0].Name != "en-5" {
		t.Fatalf("bloc 7 j = %+v", b)
	}
	if med.defsLocale != "en" {
		t.Fatalf("locale des noms = %q, attendu en", med.defsLocale)
	}
	page(t, trendsSourcesService().WithMedals(med, med), "")
	if med.defsLocale != "fr" {
		t.Fatalf("locale par défaut = %q, attendu fr", med.defsLocale)
	}
}

func TestTrendsSources_ParticipantsSeulementPourLesMatchsAvecRole(t *testing.T) {
	obj, use := objectiveFixtures()
	page(t, trendsSourcesService().WithObjectives(obj, use), "")
	want := []string{"m0", "m1", "m2", "m3", "m4"}
	got := append([]string(nil), use.partIDs...)
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("participants lus pour %v, attendu %v", got, want)
	}
	// Aucune ligne de rôle : participants jamais lus.
	obj2, use2 := &fakeTrendsObjectives{}, &fakeTrendsUsage{}
	page(t, trendsSourcesService().WithObjectives(obj2, use2), "")
	if use2.partCall.Load() != 0 {
		t.Fatalf("participants lus sans ligne de rôle")
	}
}

func TestTrendsSources_LectureEnEchecBlocOmisPageServie(t *testing.T) {
	boom := errors.New("lecture impossible")
	cases := map[string]func() *TrendsService{
		"rôles": func() *TrendsService {
			obj, use := objectiveFixtures()
			obj.rolesErr = boom
			return trendsSourcesService().WithObjectives(obj, use)
		},
		"prises nettes": func() *TrendsService {
			obj, use := objectiveFixtures()
			obj.grabsErr = boom
			return trendsSourcesService().WithObjectives(obj, use)
		},
		"participants": func() *TrendsService {
			obj, use := objectiveFixtures()
			use.participantErr = boom
			return trendsSourcesService().WithObjectives(obj, use)
		},
		"équipement": func() *TrendsService {
			use := equipmentFixtures()
			use.playersErr = boom
			return trendsSourcesService().WithEquipmentUsage(use)
		},
		"médailles": func() *TrendsService {
			med := medalFixtures()
			med.medalsErr = boom
			return trendsSourcesService().WithMedals(med, med)
		},
		"noms de médailles": func() *TrendsService {
			med := medalFixtures()
			med.defsErr = boom
			return trendsSourcesService().WithMedals(med, med)
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			resp := page(t, build(), "")
			if hasKey(resp, domain.TrendsKeyObjectiveTakeShare) || hasKey(resp, domain.TrendsKeyEquipmentUsedShare) || len(resp.Medals) != 0 {
				t.Fatalf("bloc dégradé attendu absent (%d médailles)", len(resp.Medals))
			}
			if !hasKey(resp, domain.TrendsKeyMatchCount) {
				t.Fatal("le reste de la page doit être servi")
			}
		})
	}
}

// logRecorder garde les niveaux des enregistrements émis.
type logRecorder struct{ levels []slog.Level }

func (r *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (r *logRecorder) Handle(_ context.Context, rec slog.Record) error {
	if rec.Message != "trends: matchs chargés" {
		r.levels = append(r.levels, rec.Level)
	}
	return nil
}
func (r *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *logRecorder) WithGroup(string) slog.Handler      { return r }

func TestTrendsSources_Journalisation(t *testing.T) {
	prev := slog.Default()
	defer slog.SetDefault(prev)
	run := func(err error) []slog.Level {
		rec := &logRecorder{}
		slog.SetDefault(slog.New(rec))
		use := equipmentFixtures()
		use.playersErr = err
		page(t, trendsSourcesService().WithEquipmentUsage(use), "")
		return rec.levels
	}
	if got := run(errors.New("boom")); len(got) != 1 || got[0] != slog.LevelWarn {
		t.Fatalf("niveaux = %v, attendu un avertissement", got)
	}
	if got := run(fmt.Errorf("x: %w", games.ErrCapabilityNotSupported)); len(got) != 1 || got[0] != slog.LevelDebug {
		t.Fatalf("niveaux = %v, attendu un message de débogage", got)
	}
	if got := run(nil); len(got) != 0 {
		t.Fatalf("source absente ou saine : aucun journal attendu, reçu %v", got)
	}
}

func TestTrendsSources_SansXUIDAucuneLecture(t *testing.T) {
	obj, use := objectiveFixtures()
	med := medalFixtures()
	s := newTrendsTestService(&fakeTrendsMatches{rows: trendsSourceRows()}).
		WithObjectives(obj, use).WithEquipmentUsage(use).WithMedals(med, med)
	resp := page(t, s, "")
	if obj.roleCall.Load() != 0 || use.playersCall.Load() != 0 || med.medalCall.Load() != 0 || len(resp.Medals) != 0 {
		t.Fatal("sans xuid, aucune source ne doit être lue")
	}
}
