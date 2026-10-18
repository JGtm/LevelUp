package citationsrelance

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// fauxRecalcul : les ports de la relance, sans base.
type fauxRecalcul struct {
	mu           sync.Mutex
	suivis       []domain.PlayerSummary
	participants map[string][]string
	issues       map[string]domain.CitationsRelanceIssue // par xuid
	errs         map[string]error                        // par xuid
	appels       []string                                // "match/xuid"
	fait         chan struct{}
}

func (f *fauxRecalcul) Participants(_ context.Context, _, matchID string) ([]string, error) {
	return f.participants[matchID], nil
}

func (f *fauxRecalcul) FollowedPlayers(string) ([]domain.PlayerSummary, error) { return f.suivis, nil }

func (f *fauxRecalcul) RecomputeForPlayer(_ context.Context, _ string, p domain.PlayerSummary, matchID string) (domain.CitationsRelanceIssue, error) {
	f.mu.Lock()
	f.appels = append(f.appels, matchID+"/"+p.XUID)
	f.mu.Unlock()
	if f.fait != nil {
		f.fait <- struct{}{}
	}
	if err := f.errs[p.XUID]; err != nil {
		return "", err
	}
	return f.issues[p.XUID], nil
}

func suivi(xuid string) domain.PlayerSummary {
	return domain.PlayerSummary{Gamertag: "gt-" + xuid, XUID: xuid, SyncEnabled: true}
}

func TestTraiter_SeulsLesParticipantsSuivisSontRelances(t *testing.T) {
	enPause := suivi("x3")
	enPause.SyncEnabled = false
	authOnly := suivi("x4")
	authOnly.AuthOnly = true
	f := &fauxRecalcul{
		suivis:       []domain.PlayerSummary{suivi("x1"), suivi("x2"), enPause, authOnly, suivi("x9")},
		participants: map[string][]string{"m1": {"x1", "x2", "x3", "x4", "inconnu"}},
		issues: map[string]domain.CitationsRelanceIssue{
			"x1": domain.CitationsRelanceRecalculee, "x2": domain.CitationsRelanceJoueurOccupe,
		},
	}
	s := New(f)
	avant := observability.LoadCounterT("titre-a", CompteurRecalculees)
	occupesAvant := observability.LoadCounterT("titre-a", CompteurJoueurOccupe)

	s.traiter(context.Background(), demande{titleSlug: "titre-a", matchID: "m1"})

	if len(f.appels) != 2 || f.appels[0] != "m1/x1" || f.appels[1] != "m1/x2" {
		t.Fatalf("relancés : %v, attendu les deux participants suivis (ni en pause, ni auth seule, ni absent du match)", f.appels)
	}
	if got := observability.LoadCounterT("titre-a", CompteurRecalculees) - avant; got != 1 {
		t.Fatalf("recalculées : +%d, attendu +1", got)
	}
	if got := observability.LoadCounterT("titre-a", CompteurJoueurOccupe) - occupesAvant; got != 1 {
		t.Fatalf("joueurs occupés : +%d, attendu +1", got)
	}
}

func TestTraiter_UneErreurNArretePasLesAutresJoueurs(t *testing.T) {
	f := &fauxRecalcul{
		suivis:       []domain.PlayerSummary{suivi("x1"), suivi("x2")},
		participants: map[string][]string{"m1": {"x1", "x2"}},
		errs:         map[string]error{"x1": errors.New("base illisible")},
		issues:       map[string]domain.CitationsRelanceIssue{"x2": domain.CitationsRelanceRecalculee},
	}
	avant := observability.LoadCounterT("titre-b", CompteurErreurs)
	New(f).traiter(context.Background(), demande{titleSlug: "titre-b", matchID: "m1"})
	if len(f.appels) != 2 {
		t.Fatalf("relancés : %v, attendu les deux", f.appels)
	}
	if got := observability.LoadCounterT("titre-b", CompteurErreurs) - avant; got != 1 {
		t.Fatalf("erreurs : +%d, attendu +1", got)
	}
}

func TestSoumettre_DedoublonneEtBorne(t *testing.T) {
	s := New(&fauxRecalcul{})
	s.Soumettre(context.Background(), "titre-c", []string{"m1", "m1", "m2"})
	if got := len(s.file); got != 2 {
		t.Fatalf("en file : %d, attendu 2 (doublon écarté)", got)
	}
	avant := observability.LoadCounterT("titre-c", CompteurPerdues)
	ids := make([]string, FileCap)
	for i := range ids {
		ids[i] = "plein-" + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	s.Soumettre(context.Background(), "titre-c", ids)
	if got := len(s.file); got != FileCap {
		t.Fatalf("file : %d, attendu %d", got, FileCap)
	}
	if got := observability.LoadCounterT("titre-c", CompteurPerdues) - avant; got != 2 {
		t.Fatalf("perdues : +%d, attendu +2 (deux demandes au-delà de la borne)", got)
	}
}

func TestRun_TraiteLaFileEtSArreteAvecLeContexte(t *testing.T) {
	f := &fauxRecalcul{
		suivis:       []domain.PlayerSummary{suivi("x1")},
		participants: map[string][]string{"m1": {"x1"}},
		issues:       map[string]domain.CitationsRelanceIssue{"x1": domain.CitationsRelanceAJour},
		fait:         make(chan struct{}, 1),
	}
	s := New(f)
	ctx, annuler := context.WithCancel(context.Background())
	fini := make(chan struct{})
	go func() { s.Run(ctx); close(fini) }()

	s.Soumettre(ctx, "titre-d", []string{"m1"})
	select {
	case <-f.fait:
	case <-time.After(5 * time.Second):
		t.Fatal("la demande n'a pas été traitée")
	}
	// Une fois traité, le même match peut être redemandé (il n'est plus « en file »).
	s.mu.Lock()
	enFile := s.enFile[demande{titleSlug: "titre-d", matchID: "m1"}]
	s.mu.Unlock()
	if enFile {
		t.Fatal("un match traité reste marqué en file")
	}
	annuler()
	select {
	case <-fini:
	case <-time.After(5 * time.Second):
		t.Fatal("Run ne s'arrête pas avec son contexte")
	}
}
