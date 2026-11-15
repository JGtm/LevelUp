package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// fakeProgressionStore : dépôt en mémoire ; une erreur par lecture.
type fakeProgressionStore struct {
	streaks   []streaks.Streak
	pbs       []records.PersonalRecord
	history   []records.RecordHistory
	catalog   []milestones.CatalogEntry
	earned    []milestones.Earned
	errs      map[string]error
	gotTitle  string
	gotLimit  int
	gotCalled []string
}

func (f *fakeProgressionStore) note(name string) error {
	f.gotCalled = append(f.gotCalled, name)
	return f.errs[name]
}

func (f *fakeProgressionStore) ListStreaks(_ context.Context, title string) ([]streaks.Streak, error) {
	f.gotTitle = title
	return f.streaks, f.note("streaks")
}

func (f *fakeProgressionStore) ListPersonalBests(context.Context) ([]records.PersonalRecord, error) {
	return f.pbs, f.note("pbs")
}

func (f *fakeProgressionStore) ListRecordHistory(_ context.Context, title string, limit int) ([]records.RecordHistory, error) {
	f.gotTitle, f.gotLimit = title, limit
	return f.history, f.note("history")
}

func (f *fakeProgressionStore) ListMilestoneCatalog(_ context.Context, title string) ([]milestones.CatalogEntry, error) {
	f.gotTitle = title
	return f.catalog, f.note("catalog")
}

func (f *fakeProgressionStore) ListMilestonesEarned(_ context.Context, title string) ([]milestones.Earned, error) {
	f.gotTitle = title
	return f.earned, f.note("earned")
}

func newProgressionSvc(st port.ProgressionStore, resolveErr error) *ProgressionService {
	return NewProgressionService(func(_ context.Context, slug string) (port.ProgressionStore, error) {
		if resolveErr != nil || slug != "jg" {
			if resolveErr == nil {
				return nil, errors.New("inconnu")
			}
			return nil, resolveErr
		}
		return st, nil
	})
}

func progressionKind(t *testing.T, err error) port.ProgressionErrorKind {
	t.Helper()
	var pe *port.ProgressionError
	if !errors.As(err, &pe) {
		t.Fatalf("erreur %v : pas un *port.ProgressionError", err)
	}
	return pe.Kind
}

func TestProgressionService_JoueurIntrouvable(t *testing.T) {
	svc := newProgressionSvc(&fakeProgressionStore{}, nil)
	ctx := context.Background()
	_, err1 := svc.Streaks(ctx, "inconnu", "halo_infinite")
	_, err2 := svc.Records(ctx, "inconnu", "halo_infinite", 10)
	_, err3 := svc.Milestones(ctx, "inconnu", "halo_infinite")
	for i, err := range []error{err1, err2, err3} {
		if progressionKind(t, err) != port.ProgErrPlayerNotFound {
			t.Errorf("appel %d : nature %v", i, progressionKind(t, err))
		}
	}
}

func TestProgressionService_StreaksPoseLeTitreEtTypeLEchec(t *testing.T) {
	cause := errors.New("lecture impossible")
	st := &fakeProgressionStore{streaks: []streaks.Streak{{ID: "s1"}}}
	got, err := newProgressionSvc(st, nil).Streaks(context.Background(), "jg", "halo_5")
	if err != nil || len(got) != 1 || st.gotTitle != "halo_5" {
		t.Fatalf("Streaks = %v, %v, titre %q", got, err, st.gotTitle)
	}
	st.errs = map[string]error{"streaks": cause}
	_, err = newProgressionSvc(st, nil).Streaks(context.Background(), "jg", "halo_5")
	if progressionKind(t, err) != port.ProgErrListStreaks || !errors.Is(err, cause) {
		t.Fatalf("erreur = %v ; attendu ProgErrListStreaks portant la cause", err)
	}
}

func TestProgressionService_RecordsFiltreLesMetriquesHorsCatalogue(t *testing.T) {
	st := &fakeProgressionStore{
		pbs: []records.PersonalRecord{
			{Metric: "performance_score", Value: 90}, {Metric: "best_kda", Value: 3}},
		history: []records.RecordHistory{
			{ID: "h1", Metric: "best_kda"}, {ID: "h2", Metric: "performance_score"}},
	}
	got, err := newProgressionSvc(st, nil).Records(context.Background(), "jg", "halo_infinite", 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PersonalBests) != 1 || got.PersonalBests[0].Metric != "performance_score" {
		t.Errorf("records = %+v ; la métrique hors catalogue doit être écartée", got.PersonalBests)
	}
	if len(got.History) != 1 || got.History[0].ID != "h2" {
		t.Errorf("historique = %+v", got.History)
	}
	if st.gotLimit != 25 {
		t.Errorf("limite transmise = %d, attendu 25", st.gotLimit)
	}
}

func TestProgressionService_RecordsNatureDesEchecs(t *testing.T) {
	cause := errors.New("boom")
	for _, c := range []struct {
		step string
		want port.ProgressionErrorKind
	}{{"pbs", port.ProgErrListPersonalBests}, {"history", port.ProgErrListRecordHistory}} {
		st := &fakeProgressionStore{errs: map[string]error{c.step: cause}}
		_, err := newProgressionSvc(st, nil).Records(context.Background(), "jg", "halo_infinite", 10)
		if progressionKind(t, err) != c.want {
			t.Errorf("%s : nature %v, attendu %v", c.step, progressionKind(t, err), c.want)
		}
	}
	// Un échec des records n'ouvre pas la lecture de l'historique.
	st := &fakeProgressionStore{errs: map[string]error{"pbs": cause}}
	_, _ = newProgressionSvc(st, nil).Records(context.Background(), "jg", "halo_infinite", 10)
	for _, c := range st.gotCalled {
		if c == "history" {
			t.Errorf("l'historique est lu malgré l'échec des records : %v", st.gotCalled)
		}
	}
}

func TestProgressionService_MilestonesJointCatalogueEtJalonsFranchis(t *testing.T) {
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	st := &fakeProgressionStore{
		catalog: []milestones.CatalogEntry{
			{ID: "a", Title: i18n.FromColumns("A", "A")},
			{ID: "b", Title: i18n.FromColumns("B", "B")},
			{ID: "c", Title: i18n.FromColumns("C", "C")},
		},
		earned: []milestones.Earned{
			{MilestoneID: "a", EarnedAt: at},
			{MilestoneID: "b"}, // franchi, date inconnue
			{MilestoneID: "hors-catalogue", EarnedAt: at},
		},
	}
	got, err := newProgressionSvc(st, nil).Milestones(context.Background(), "jg", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Entry.ID != "a" || got[1].Entry.ID != "b" || got[2].Entry.ID != "c" {
		t.Fatalf("l'ordre du catalogue n'est pas conservé : %+v", got)
	}
	if !got[0].Earned || got[0].EarnedAt == nil || !got[0].EarnedAt.Equal(at) {
		t.Errorf("a : %+v", got[0])
	}
	if !got[1].Earned || got[1].EarnedAt != nil {
		t.Errorf("b (franchi sans date) : %+v", got[1])
	}
	if got[2].Earned || got[2].EarnedAt != nil {
		t.Errorf("c (non franchi) : %+v", got[2])
	}
}

func TestProgressionService_MilestonesNatureDesEchecs(t *testing.T) {
	cause := errors.New("boom")
	for _, c := range []struct {
		step string
		want port.ProgressionErrorKind
	}{{"catalog", port.ProgErrListCatalog}, {"earned", port.ProgErrListEarned}} {
		st := &fakeProgressionStore{errs: map[string]error{c.step: cause}}
		_, err := newProgressionSvc(st, nil).Milestones(context.Background(), "jg", "halo_infinite")
		if progressionKind(t, err) != c.want {
			t.Errorf("%s : nature %v, attendu %v", c.step, progressionKind(t, err), c.want)
		}
	}
}
