package teammates

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakePending — un résolveur d'analyse en cours à table fixe ; il note les identifiants demandés.
type fakePending struct {
	table  map[string][]string
	asked  []string
	called bool
}

func (f *fakePending) ResolvePending(_ context.Context, ids []string) map[string][]string {
	f.called = true
	f.asked = append([]string(nil), ids...)
	return f.table
}

var _ port.AnalysisPendingMatches = (*fakePending)(nil)

func blocEmprise(unavailable string, films ...bool) *domain.SquadEmpriseBlock {
	b := &domain.SquadEmpriseBlock{FilmUnavailable: unavailable}
	for i, has := range films {
		b.Matches = append(b.Matches, domain.SquadEmpriseMatch{MatchID: string(rune('a' + i)), HasFilm: has})
	}
	return b
}

func TestNoteFilmGaps_SeparateEnCoursEtNonAnalyse(t *testing.T) {
	// a : film lu ; b, c : sans film, analyse en cours ; d : sans film, rien en cours.
	b := blocEmprise("", true, false, false, false)
	fake := &fakePending{table: map[string][]string{"b": {domain.AnalysisPartKillSource}, "c": {domain.AnalysisPartCitations}}}
	s := (&TeammatesService{}).WithAnalysisPending(fake)

	s.noteFilmGaps(context.Background(), b)

	if b.MatchesFilmPending != 2 || b.MatchesFilmMissing != 1 {
		t.Fatalf("en cours=%d manquant=%d, want 2 et 1", b.MatchesFilmPending, b.MatchesFilmMissing)
	}
	if got := len(fake.asked); got != 3 {
		t.Fatalf("le résolveur ne doit être interrogé que sur les matchs sans film : %v", fake.asked)
	}
}

func TestNoteFilmGaps_SansResolveurToutEstNonAnalyse(t *testing.T) {
	b := blocEmprise("", false, false)
	(&TeammatesService{}).noteFilmGaps(context.Background(), b)
	if b.MatchesFilmPending != 0 || b.MatchesFilmMissing != 2 {
		t.Fatalf("en cours=%d manquant=%d, want 0 et 2", b.MatchesFilmPending, b.MatchesFilmMissing)
	}
}

func TestNoteFilmGaps_TitreSansFilmNeCompteRien(t *testing.T) {
	b := blocEmprise(domain.EmpriseFilmUnsupported, false, false)
	fake := &fakePending{table: map[string][]string{"a": {domain.AnalysisPartKillSource}}}
	(&TeammatesService{}).WithAnalysisPending(fake).noteFilmGaps(context.Background(), b)
	if b.MatchesFilmPending != 0 || b.MatchesFilmMissing != 0 {
		t.Fatalf("un titre sans résumé d'usage n'attend rien : en cours=%d manquant=%d", b.MatchesFilmPending, b.MatchesFilmMissing)
	}
	if fake.called {
		t.Fatal("aucune lecture d'avancement pour un titre sans film")
	}
}

func TestNoteFilmGaps_TousLesFilmsLusNeComptentRien(t *testing.T) {
	b := blocEmprise("", true, true)
	fake := &fakePending{}
	(&TeammatesService{}).WithAnalysisPending(fake).noteFilmGaps(context.Background(), b)
	if b.MatchesFilmPending != 0 || b.MatchesFilmMissing != 0 || fake.called {
		t.Fatalf("rien à compter ni à lire : %+v appelé=%v", b, fake.called)
	}
}
