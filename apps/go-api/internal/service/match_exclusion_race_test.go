package service

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// serialExclusionRepo tient l'état du flag sous un verrou (la base joueur et son verrou
// d'écriture). La première demande qui lit l'état est retenue, état lu, jusqu'à hold :
// une demande concurrente arrive pendant qu'elle décide.
type serialExclusionRepo struct {
	mockExclusionRepo
	lock    sync.Mutex // verrou d'écriture de la base joueur
	state   sync.Mutex // protège excluded
	reads   atomic.Int32
	arrived atomic.Int32 // demandes arrivées au dépôt
	entered chan struct{}
	hold    chan struct{}
}

func (r *serialExclusionRepo) readState(matchID string) bool {
	r.state.Lock()
	defer r.state.Unlock()
	return r.excluded[matchID]
}

func (r *serialExclusionRepo) writeState(matchID string, excluded bool) {
	r.state.Lock()
	defer r.state.Unlock()
	r.excluded[matchID] = excluded
	r.sets++
}

// pause retient la première lecture d'état.
func (r *serialExclusionRepo) pause() {
	if r.reads.Add(1) == 1 {
		close(r.entered)
		<-r.hold
	}
}

// SetExclusion lit l'état, décide et écrit sous le verrou, comme le dépôt DuckDB.
func (r *serialExclusionRepo) SetExclusion(_ context.Context, matchID string, excluded bool) (bool, error) {
	r.arrived.Add(1) // la demande est arrivée au dépôt, avant le verrou
	r.lock.Lock()
	defer r.lock.Unlock()
	current := r.readState(matchID)
	r.pause()
	if current == excluded {
		return false, nil
	}
	r.writeState(matchID, excluded)
	return true, nil
}

// Exclure puis réinclure un match par deux demandes qui se chevauchent : la seconde décide
// sur l'état que laisse la première (lecture et décision sous le verrou de la base joueur),
// jamais sur un état lu avant son écriture. L'état final est celui de la dernière demande.
func TestMatchExclusionService_OverlappingExcludeIncludeEndsOnTheLastDemand(t *testing.T) {
	repo := &serialExclusionRepo{
		mockExclusionRepo: mockExclusionRepo{
			excluded: map[string]bool{},
			registry: map[string]domain.MatchRegistryInfo{"m1": {MatchID: "m1"}},
		},
		entered: make(chan struct{}),
		hold:    make(chan struct{}),
	}
	svc := NewMatchExclusionService(repo, nil)
	ctx := context.Background()

	excludeDone := make(chan error, 1)
	go func() { excludeDone <- svc.SetExclusion(ctx, "m1", true) }()
	<-repo.entered // l'exclusion a lu l'état (non exclu) et n'a pas encore écrit

	includeDone := make(chan error, 1)
	go func() { includeDone <- svc.SetExclusion(ctx, "m1", false) }()
	// La réinclusion est arrivée : terminée (elle a décidé sans attendre) ou au verrou.
	deadline := time.Now().Add(5 * time.Second)
	for len(includeDone) == 0 && repo.arrived.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatal("la réinclusion n'est jamais arrivée")
		}
		runtime.Gosched()
	}
	close(repo.hold)

	if err := <-excludeDone; err != nil {
		t.Fatal(err)
	}
	if err := <-includeDone; err != nil {
		t.Fatal(err)
	}
	if repo.readState("m1") {
		t.Fatal("état final : exclu, alors que la réinclusion est arrivée après l'exclusion (décision sur un état périmé)")
	}
}
