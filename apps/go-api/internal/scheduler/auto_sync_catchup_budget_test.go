// auto_sync_catchup_budget_test.go — le budget de rattrapage d'un titre live-only est par cycle et par titre :
// les joueurs en rattrapage se le partagent (au moins 25 matchs chacun, 8 joueurs au plus), les autres
// attendent le cycle suivant, les moins récemment passés d'abord.
package scheduler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/historystate"
)

func h5Players(n int) []domain.PlayerSummary {
	out := make([]domain.PlayerSummary, 0, n)
	for i := 1; i <= n; i++ {
		gt := fmt.Sprintf("H5-%02d", i)
		out = append(out, domain.PlayerSummary{Gamertag: gt, XUID: gt, PlayerSlug: gt, TitleSlug: halo5.TitleSlug})
	}
	return out
}

func incompleteAll(players []domain.PlayerSummary) *fakeHistoryReader {
	m := map[string]bool{}
	for _, p := range players {
		m[p.Gamertag] = true
	}
	return &fakeHistoryReader{incomplete: m}
}

func TestPlanLiveCatchUp_RepartitLeBudgetDuTitre(t *testing.T) {
	budget := historystate.PassBudget()
	cases := []struct {
		catching  int
		wantShare int
		wantWait  int
	}{
		{1, budget, 0},
		{2, budget / 2, 0},
		{4, budget / 4, 0},
		{8, minCatchUpShare, 0},
		{9, minCatchUpShare, 1},
		{12, minCatchUpShare, 4},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d joueurs en rattrapage", c.catching), func(t *testing.T) {
			s := newH5Scheduler(t)
			players := h5Players(c.catching)
			s.WithHistoryReader(incompleteAll(players))
			plan := s.planLiveCatchUp(context.Background(), players)
			if len(plan.wait) != c.wantWait {
				t.Errorf("joueurs en attente = %d, attendu %d", len(plan.wait), c.wantWait)
			}
			served := 0
			for _, p := range players {
				if plan.waits(p) {
					continue
				}
				served++
				if got := plan.optionsFor(p).MaxMatches; got != c.wantShare {
					t.Errorf("part de %s = %d, attendu %d", p.Gamertag, got, c.wantShare)
				}
			}
			if served*c.wantShare > budget {
				t.Errorf("%d joueurs servis à %d dépassent le budget du titre (%d)", served, c.wantShare, budget)
			}
		})
	}
}

// Un joueur dont l'historique est complet garde les options par défaut et ne consomme aucune part ; sans lecteur
// d'historique, personne n'est en rattrapage ; un titre du moteur V2 n'entre pas dans le plan.
func TestPlanLiveCatchUp_SeulsLesJoueursEnRattrapageDuTitreLiveCompent(t *testing.T) {
	s := newH5Scheduler(t)
	players := append(h5Players(3), domain.PlayerSummary{Gamertag: "JGtm", TitleSlug: "halo_infinite"})
	s.WithHistoryReader(&fakeHistoryReader{incomplete: map[string]bool{"H5-01": true, "JGtm": true}})
	plan := s.planLiveCatchUp(context.Background(), players)
	if got := plan.optionsFor(players[0]).MaxMatches; got != historystate.PassBudget() {
		t.Errorf("seul joueur en rattrapage du titre : part = %d, attendu le budget entier", got)
	}
	for _, p := range players[1:] {
		if got := plan.optionsFor(p).MaxMatches; got != domain.DefaultSyncOptions().MaxMatches {
			t.Errorf("%s : MaxMatches = %d, attendu la valeur par défaut", p.Gamertag, got)
		}
	}

	s.historyReader = nil
	if plan := s.planLiveCatchUp(context.Background(), players); len(plan.maxMatches)+len(plan.wait) != 0 {
		t.Errorf("sans lecteur d'historique, le plan doit être vide : %+v", plan)
	}
}

// Au-delà de 8 joueurs, ceux dont la dernière passe est la plus ancienne (ou inconnue) sont servis d'abord :
// chacun finit par passer.
func TestPlanLiveCatchUp_LesMoinsRecemmentPassesPassentDabord(t *testing.T) {
	s := newH5Scheduler(t)
	players := h5Players(10)
	s.WithHistoryReader(incompleteAll(players))
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	// H5-01 et H5-02 viennent de passer ; les autres ont passé plus tôt ou jamais.
	s.recordPass(halo5.TitleSlug, "H5-01", port.SyncPass{At: now, Status: port.SyncPassOK})
	s.recordPass(halo5.TitleSlug, "H5-02", port.SyncPass{At: now.Add(-time.Minute), Status: port.SyncPassOK})
	s.recordPass(halo5.TitleSlug, "H5-03", port.SyncPass{At: now.Add(-time.Hour), Status: port.SyncPassOK})

	plan := s.planLiveCatchUp(context.Background(), players)
	var waiting []string
	for _, p := range players {
		if plan.waits(p) {
			waiting = append(waiting, p.Gamertag)
		}
	}
	sort.Strings(waiting)
	if len(waiting) != 2 || waiting[0] != "H5-01" || waiting[1] != "H5-02" {
		t.Errorf("en attente = %v, attendu les deux passes les plus récentes [H5-01 H5-02]", waiting)
	}
}

// Le cycle : chaque joueur servi reçoit sa part dans opts.MaxMatches, celui qui attend ne lance pas de runner,
// reste en rattrapage (cadence courte) et garde sa dernière passe connue.
func TestSyncPlayersConcurrent_TransmetLaPartEtFaitAttendre(t *testing.T) {
	s := newH5Scheduler(t)
	players := h5Players(10)
	s.WithHistoryReader(incompleteAll(players))
	s.recordPass(halo5.TitleSlug, "H5-10", port.SyncPass{At: time.Now(), Status: port.SyncPassOK})
	s.recordPass(halo5.TitleSlug, "H5-09", port.SyncPass{At: time.Now(), Status: port.SyncPassOK})

	var mu sync.Mutex
	got := map[string]int{}
	s.liveRunner = func(ctx context.Context, _, gt, _ string) (DeltaRunner, context.Context, func(), error) {
		return optsRecorder{record: func(o domain.SyncOptions) {
			mu.Lock()
			got[gt] = o.MaxMatches
			mu.Unlock()
		}}, ctx, func() {}, nil
	}
	tally := s.syncPlayersConcurrent(context.Background(), players)

	if len(got) != 8 {
		t.Fatalf("%d joueurs servis, attendu 8 (les deux dernières passes attendent)", len(got))
	}
	for gt, n := range got {
		if n != minCatchUpShare {
			t.Errorf("%s : MaxMatches = %d, attendu %d", gt, n, minCatchUpShare)
		}
	}
	if _, ran := got["H5-09"]; ran {
		t.Error("H5-09 attend le cycle suivant : son runner ne doit pas tourner")
	}
	if tally.skipped != 2 || tally.synced != 8 {
		t.Errorf("synced/skipped = %d/%d, attendu 8/2", tally.synced, tally.skipped)
	}
	if len(tally.pending) != 10 {
		t.Errorf("%d joueurs en rattrapage, attendu 10 (ceux qui attendent restent en rattrapage)", len(tally.pending))
	}
	if pass, ok := s.LastPass(halo5.TitleSlug, "H5-09"); !ok || pass.Status != port.SyncPassOK {
		t.Errorf("la dernière passe de H5-09 doit rester celle d'avant : %+v (%v)", pass, ok)
	}
}

type optsRecorder struct{ record func(domain.SyncOptions) }

func (r optsRecorder) RunDelta(_ context.Context, o domain.SyncOptions) (domain.SyncResult, error) {
	r.record(o)
	return domain.SyncResult{HistoryPending: true}, nil
}
