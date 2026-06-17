package watcher

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
)

// TestMatchQueue_PreservesTitleSlug (Phase 1.9) : Enqueue ne doit PAS dropper le
// TitleSlug en reconstruisant `filtered`. Sans ça, le CoordinatorRequest retombe
// sur halo_infinite même pour un 2e titre suivi (bug latent corrigé).
func TestMatchQueue_PreservesTitleSlug(t *testing.T) {
	q := NewMatchQueue(10)
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}, TitleSlug: "halo_mcc"})
	select {
	case req := <-q.Dequeue():
		if req.TitleSlug != "halo_mcc" {
			t.Fatalf("filtered.TitleSlug = %q, want halo_mcc (droppé par Enqueue ?)", req.TitleSlug)
		}
	default:
		t.Fatal("rien en file")
	}
}

// TestQueueSyncTrigger_TitleFromCtx (Phase 1.9) : le trigger lit le titre depuis
// le ctx (posé par startPoller) → MatchRequest.TitleSlug. Ctx sans titre ⇒
// halo_infinite (ctxkeys défaut, byte-identique mono-titre).
func TestQueueSyncTrigger_TitleFromCtx(t *testing.T) {
	q := NewMatchQueue(10)
	trig := &queueSyncTrigger{queue: q, gamertag: "p1", xuid: "x1"}
	if err := trig.TriggerSync(ctxkeys.WithTitleSlug(context.Background(), "halo_mcc"), "p1", "x1", []string{"m1"}); err != nil {
		t.Fatal(err)
	}
	if req := <-q.Dequeue(); req.TitleSlug != "halo_mcc" {
		t.Fatalf("TitleSlug = %q, want halo_mcc (depuis le ctx)", req.TitleSlug)
	}

	// Ctx sans titre → défaut halo_infinite (garde byte-identique).
	q2 := NewMatchQueue(10)
	trig2 := &queueSyncTrigger{queue: q2, gamertag: "p2", xuid: "x2"}
	if err := trig2.TriggerSync(context.Background(), "p2", "x2", []string{"m2"}); err != nil {
		t.Fatal(err)
	}
	if req := <-q2.Dequeue(); req.TitleSlug != "halo_infinite" {
		t.Fatalf("TitleSlug = %q, want halo_infinite (défaut)", req.TitleSlug)
	}
}

// titleCapturingFetcher capture le ctxkeys.TitleSlug du ctx reçu par le fetch.
type titleCapturingFetcher struct {
	got chan string
}

func (f *titleCapturingFetcher) FetchRecentMatchIDs(ctx context.Context, _ string, _ int) ([]string, error) {
	select {
	case f.got <- ctxkeys.TitleSlug(ctx):
	default:
	}
	return nil, nil
}

// TestPlayerWatcher_TitleSlug_RoutesFetchCtx (Phase 1.9) : SetTitleSlug pose le
// titre sur le ctx du poller → le fetch match-history le voit (routing host
// PMT-1 via ctxkeys). Sans SetTitleSlug ⇒ halo_infinite (défaut byte-identique).
// Le poll seed immédiat (MatchPoller.Run) appelle le fetch tout de suite.
func TestPlayerWatcher_TitleSlug_RoutesFetchCtx(t *testing.T) {
	cases := []struct {
		name, set, want string
	}{
		{"titre configuré", "halo_mcc", "halo_mcc"},
		{"non configuré (défaut)", "", "halo_infinite"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := &titleCapturingFetcher{got: make(chan string, 1)}
			pw := NewPlayerWatcher("p1", "x1", fetcher, newMockSyncTrigger())
			if tc.set != "" {
				pw.SetTitleSlug(tc.set)
			}
			pw.OnPresenceActive(context.Background()) // → startPoller → poll seed immédiat
			defer pw.stopPoller()

			select {
			case got := <-fetcher.got:
				if got != tc.want {
					t.Fatalf("ctx du fetch TitleSlug = %q, want %q", got, tc.want)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("fetch non appelé dans les 2s")
			}
		})
	}
}
