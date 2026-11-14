package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// deadlineRepo relève l'échéance du contexte avec lequel l'écriture de l'instantané est demandée.
type deadlineRepo struct {
	mockCareerLiveRepo
	deadline chan time.Time
}

func (r *deadlineRepo) InsertCareerProgressionPartial(ctx context.Context, xuid string, p *duckdb.CareerProgressionPartial) (bool, error) {
	d, _ := ctx.Deadline()
	r.deadline <- d
	return r.mockCareerLiveRepo.InsertCareerProgressionPartial(ctx, xuid, p)
}

// L'écriture de l'instantané de carrière attend l'écrivain de la base joueur plus longtemps que le
// budget des appels à l'API (careerLiveBgTimeout) : un recalcul des notes qui tient l'écrivain
// plusieurs minutes ne la fait plus échouer.
func TestCareerLive_PersistAttendLEcrivainAuDelaDuBudgetDeFetch(t *testing.T) {
	repo := &deadlineRepo{deadline: make(chan time.Time, 1)}
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{CurrentRank: 40, CurrentXP: 100},
		custom:   &syncpkg.SpartanCustomizationData{SpartanID: "SR-1"},
	}
	svc := NewCareerLiveService(repo, &mockIdentityBuilder{},
		func(context.Context) CareerFetcher { return fetcher }, NewCareerLiveCache(CareerLiveCacheConfig{}))

	debut := time.Now()
	svc.kickoffBackgroundRefresh("1234567890123456", "", &domain.HaloTokens{SpartanToken: "tok"}, "halo_infinite")

	select {
	case d := <-repo.deadline:
		if reste := d.Sub(debut); reste <= careerLiveBgTimeout+time.Minute {
			t.Errorf("échéance de l'écriture = %v après le départ, attendu au-delà du budget de fetch (%v) : %v",
				reste, careerLiveBgTimeout, careerLivePersistWait)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("l'instantané n'a pas été écrit")
	}
}
