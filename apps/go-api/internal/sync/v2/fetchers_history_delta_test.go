// Package v2 — fetchers_history_delta_test.go : le delta du dessus se garde quand la lecture du rattrapage
// échoue, les pages passent par le rejeu borné des 429, les rattrapages d'un cycle se lisent un joueur à la fois.
package v2

import (
	"context"
	"errors"
	"fmt"
	gosync "sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historyretry"
	"levelup/go-api/internal/sync/historystate"
)

func rateLimited() error {
	return &haloadapter.HTTPError{StatusCode: 429, URL: "https://halostats/matches", Err: errors.New("HTTP 429")}
}

// flakyClient est un pagedClient dont les lectures de page sont scriptées : `fail` rend l'erreur d'un appel
// (numéro de l'appel depuis 1, par décalage), nil = la page est servie.
type flakyClient struct {
	*pagedClient
	calls map[int]int
	fail  func(start, call int) error
}

func newFlakyClient(ids []string, fail func(start, call int) error) *flakyClient {
	return &flakyClient{pagedClient: newPagedClient(ids), calls: map[int]int{}, fail: fail}
}

func (c *flakyClient) GetMatchHistory(ctx context.Context, g, m string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	c.calls[start]++
	if err := c.fail(start, c.calls[start]); err != nil {
		return nil, err
	}
	return c.pagedClient.GetMatchHistory(ctx, g, m, start, count)
}

func listWith(c HaloClient, known map[string]bool, hist HistoryState) (Listing, error) {
	p := NewMatchListProvider(func(string, string) HaloClient { return c }, "matchmaking", 25, 20)
	return p.ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}, known, hist)
}

// Le rattrapage ne peut pas lire sa page de reprise : les nouveaux matchs du delta sont rendus et annoncés comme
// nouveaux, le joueur n'est pas en échec, aucune position ni fin d'historique n'est proposée.
func TestListCatchUp_LectureEnEchec_GardeLesNouveauxMatchs(t *testing.T) {
	old := matchIDs("m", 1000)
	fresh := matchIDs("n", 7)
	known := knownOfIDs(old[:600])
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: old[599], HeadID: old[0]}
	c := newFlakyClient(append(append([]string{}, fresh...), old...), func(start, _ int) error {
		if start == 606 { // la page de reprise du marque-page 600 + 7 nouveaux, moins l'entrée de contrôle
			return rateLimited()
		}
		return nil
	})

	l, err := listWith(c, known, HistoryState{Incomplete: true, Cursor: cursor})
	if err != nil {
		t.Fatalf("la lecture du rattrapage en échec ne doit pas faire échouer le joueur : %v", err)
	}
	if !equalSlice(l.Unknown, fresh) || !equalSlice(l.Arrived, fresh) {
		t.Errorf("Unknown = %v, Arrived = %v, want les %d nouveaux matchs", l.Unknown, l.Arrived, len(fresh))
	}
	if l.EndSeen || l.Proposed != nil {
		t.Errorf("EndSeen = %v, Proposed = %+v : rien ne doit être rangé pour le rattrapage", l.EndSeen, l.Proposed)
	}
}

// Un joueur sans match connu : le delta est borné à une passe minimale et, faute d'avoir rejoint la zone connue,
// ne passe pas pour des matchs neufs (rien n'est annoncé).
func TestListCatchUp_LectureEnEchec_JoueurSansConnusNAnnonceRien(t *testing.T) {
	c := newFlakyClient(matchIDs("m", 500), func(start, _ int) error {
		if start >= 25 {
			return rateLimited()
		}
		return nil
	})
	l, err := listWith(c, map[string]bool{}, HistoryState{Incomplete: true})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(l.Unknown) != catchUpMinBudget || len(l.Arrived) != 0 {
		t.Errorf("Unknown = %d, Arrived = %d, want %d et 0", len(l.Unknown), len(l.Arrived), catchUpMinBudget)
	}
}

// Une lecture du delta qui échoue garde le comportement d'avant : le joueur est en échec.
func TestListCatchUp_DeltaEnEchec_LeJoueurEchoue(t *testing.T) {
	c := newFlakyClient(matchIDs("m", 100), func(int, int) error { return rateLimited() })
	if _, err := listWith(c, map[string]bool{}, HistoryState{Incomplete: true}); err == nil {
		t.Fatal("err = nil, want l'échec de la lecture du delta")
	}
}

// Le delta et le rattrapage lisent la même page de tête : une seule fois.
func TestListCatchUp_PageDeTeteLueUneSeuleFois(t *testing.T) {
	old := matchIDs("m", 1000)
	c := newFlakyClient(old, func(int, int) error { return nil })
	hist := HistoryState{Incomplete: true, Cursor: &historystate.Cursor{Offset: 600, LastMatchID: old[599], HeadID: old[0]}}
	if _, err := listWith(c, knownOfIDs(old[:600]), hist); err != nil {
		t.Fatal(err)
	}
	if c.calls[0] != 1 {
		t.Errorf("page 0 lue %d fois, want 1", c.calls[0])
	}
}

// Une page en 429 est rejouée par historyretry.Page, au delta (historique complet) comme au rattrapage ; un
// 429 durable rend l'erreur après historyretry.Attempts lectures.
func TestListUnknown_429RejoueSurChaqueLectureDePage(t *testing.T) {
	ids := matchIDs("m", 100)
	for _, hist := range []HistoryState{{}, {Incomplete: true}} {
		t.Run(fmt.Sprintf("incomplet=%v", hist.Incomplete), func(t *testing.T) {
			c := newFlakyClient(ids, func(start, call int) error {
				if start == 0 && call < historyretry.Attempts {
					return rateLimited()
				}
				return nil
			})
			l, err := listWith(c, map[string]bool{}, hist)
			if err != nil || len(l.Unknown) == 0 {
				t.Fatalf("Listing = %d inconnus, err = %v, want la page rejouée puis lue", len(l.Unknown), err)
			}
			if c.calls[0] != historyretry.Attempts {
				t.Errorf("page 0 lue %d fois, want %d", c.calls[0], historyretry.Attempts)
			}

			durable := newFlakyClient(ids, func(int, int) error { return rateLimited() })
			if _, err := listWith(durable, map[string]bool{}, hist); !historyretry.IsRateLimited(err) {
				t.Errorf("err = %v, want le 429 rendu après les tentatives", err)
			}
			if durable.calls[0] != historyretry.Attempts {
				t.Errorf("429 durable : page 0 lue %d fois, want %d", durable.calls[0], historyretry.Attempts)
			}
		})
	}
}

// slowClient mesure combien de lectures profondes (décalage >= 25 : le rattrapage, jamais le delta borné à
// une page) se chevauchent.
type slowClient struct {
	*pagedClient
	active, peak *atomic.Int32
}

func (c *slowClient) GetMatchHistory(ctx context.Context, g, m string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	if start >= 25 {
		n := c.active.Add(1)
		defer c.active.Add(-1)
		for {
			p := c.peak.Load()
			if n <= p || c.peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return c.pagedClient.GetMatchHistory(ctx, g, m, start, count)
}

// Plusieurs joueurs découvrent en parallèle : leurs rattrapages se lisent un à la fois.
func TestListCatchUp_UnJoueurALaFois(t *testing.T) {
	var active, peak atomic.Int32
	p := NewMatchListProvider(func(string, string) HaloClient {
		return &slowClient{pagedClient: newPagedClient(matchIDs("m", 500)), active: &active, peak: &peak}
	}, "matchmaking", 25, 20)
	var wg gosync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			prof := PlayerProfile{Gamertag: fmt.Sprintf("p%d", i), XUID: fmt.Sprint(i)}
			if _, err := p.ListUnknownMatches(context.Background(), prof, map[string]bool{}, HistoryState{Incomplete: true}); err != nil {
				t.Errorf("joueur %d : %v", i, err)
			}
		}()
	}
	wg.Wait()
	if peak.Load() != 1 {
		t.Errorf("%d rattrapages lus en même temps, want 1", peak.Load())
	}
}

// L'attente du tour de lecture s'arrête avec le contexte.
func TestListCatchUp_AttenteDuTourAnnulable(t *testing.T) {
	p := NewMatchListProvider(func(string, string) HaloClient { return newPagedClient(matchIDs("m", 100)) }, "matchmaking", 25, 20)
	p.(*matchListProviderV2).catchUp <- struct{}{} // le tour est tenu
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := p.ListUnknownMatches(ctx, PlayerProfile{Gamertag: "alice", XUID: "999"}, map[string]bool{}, HistoryState{Incomplete: true})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}
