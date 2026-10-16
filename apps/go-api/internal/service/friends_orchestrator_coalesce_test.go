package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	gosync "sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/sync"
)

const friendsCoalesceProfiles = `{"version": "3.0", "profiles": {"halo_infinite": {
  "alice": {"db_path": "a.duckdb", "xuid": "222", "waypoint_player": "alice"},
  "bob": {"db_path": "b.duckdb", "xuid": "999", "waypoint_player": "bob"}}}}`

// friendsProbe observe les recalculs d'une base joueur : nombre d'appels, appels simultanés,
// liste d'amis reçue en dernier, et bloque le premier appel jusqu'à release.
type friendsProbe struct {
	calls, inFlight, peak atomic.Int32
	entered               chan struct{}
	release               chan struct{}
	mu                    gosync.Mutex
	lastFriends           []string
	err                   error
}

func newFriendsProbe() *friendsProbe {
	return &friendsProbe{entered: make(chan struct{}, 64), release: make(chan struct{})}
}

func (p *friendsProbe) recompute(_ context.Context, _, _, _ string, friends []string) (sync.FriendsRecomputeResult, error) {
	n := p.inFlight.Add(1)
	for {
		old := p.peak.Load()
		if n <= old || p.peak.CompareAndSwap(old, n) {
			break
		}
	}
	first := p.calls.Add(1) == 1
	p.mu.Lock()
	p.lastFriends = friends
	err := p.err
	p.mu.Unlock()
	if first {
		p.entered <- struct{}{}
		<-p.release
	}
	p.inFlight.Add(-1)
	return sync.FriendsRecomputeResult{MatchesPromoted: 1}, err
}

func newFriendsService(t *testing.T, loadFriends FriendsGamertagsLoader, probe *friendsProbe) *FriendsOrchestratorService {
	t.Helper()
	profilesPath := filepath.Join(t.TempDir(), "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(friendsCoalesceProfiles), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewFriendsOrchestratorService(&config.AppConfig{DBProfilesPath: profilesPath}, loadFriends)
	svc.recomputeDB = probe.recompute
	return svc
}

// N demandes rapprochées pour un même joueur : un recalcul en cours, puis UN seul rattrapage ;
// jamais deux recalculs de ce joueur en même temps, et le rattrapage relit la liste d'amis.
func TestRecomputeForPlayer_RapidRequestsCoalesce(t *testing.T) {
	probe := newFriendsProbe()
	var loads atomic.Int32
	svc := newFriendsService(t, func(string) ([]string, error) {
		return []string{"friend-v" + string(rune('0'+loads.Add(1)))}, nil
	}, probe)

	const requests = 8
	errs := make(chan error, requests)
	run := func() {
		_, err := svc.RecomputeForPlayer(context.Background(), "222")
		errs <- err
	}
	go run()
	<-probe.entered // le premier recalcul tient la base
	for range requests - 1 {
		go run()
	}
	time.Sleep(150 * time.Millisecond) // les sept autres se sont déposées derrière lui
	close(probe.release)
	for range requests {
		if err := <-errs; err != nil {
			t.Fatalf("RecomputeForPlayer : %v", err)
		}
	}

	if got := probe.calls.Load(); got != 2 {
		t.Fatalf("recalculs = %d, attendu 2 (un en cours + un rattrapage)", got)
	}
	if got := probe.peak.Load(); got != 1 {
		t.Fatalf("recalculs simultanés = %d, attendu 1", got)
	}
	probe.mu.Lock()
	defer probe.mu.Unlock()
	if len(probe.lastFriends) != 1 || probe.lastFriends[0] != "friend-v2" {
		t.Fatalf("le rattrapage doit relire la liste d'amis : reçu %v", probe.lastFriends)
	}
}

// Les joueurs ne se bloquent pas entre eux : la coalescence est par xuid.
func TestRecomputeForPlayer_DifferentPlayersRunIndependently(t *testing.T) {
	probe := newFriendsProbe()
	svc := newFriendsService(t, func(string) ([]string, error) { return nil, nil }, probe)

	done := make(chan struct{})
	go func() {
		_, _ = svc.RecomputeForPlayer(context.Background(), "222")
		close(done)
	}()
	<-probe.entered // le joueur 222 tient son recalcul

	finished := make(chan error, 1)
	go func() {
		_, err := svc.RecomputeForPlayer(context.Background(), "999")
		finished <- err
	}()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("RecomputeForPlayer(999) : %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("le recalcul d'un autre joueur attend celui de 222")
	}
	close(probe.release)
	<-done
}

// L'échec du recalcul remonte à l'appelant qui l'a lancé.
func TestRecomputeForPlayer_PropagatesTheError(t *testing.T) {
	probe := newFriendsProbe()
	probe.err = errors.New("base illisible")
	close(probe.release)
	svc := newFriendsService(t, func(string) ([]string, error) { return nil, nil }, probe)
	if _, err := svc.RecomputeForPlayer(context.Background(), "222"); err == nil {
		t.Fatal("l'échec du recalcul doit remonter")
	}
}
