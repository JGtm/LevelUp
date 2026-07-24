// Package service — career_live_cache_test.go : tests unitaires du cache
// TTL + singleflight pour le flow live carrière.
//
// L'horloge est injectée (`now func() time.Time`) pour éliminer toute
// dépendance à time.Sleep — les tests TTL sont déterministes.
package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	syncpkg "levelup/go-api/internal/sync"
)

// testSlug : titre par défaut utilisé pour la clé composite (xuid, titleSlug).
const testSlug = "halo_infinite"

// fakeClock retourne le temps lu depuis un pointeur partagé. Sans verrou —
// les tests qui le manipulent le font de manière séquentielle ou explicite.
type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newTestCache(t *testing.T, progressTTL, customTTL time.Duration) (*CareerLiveCache, *fakeClock) {
	t.Helper()
	clk := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	cache := NewCareerLiveCache(CareerLiveCacheConfig{
		ProgressTTL:      progressTTL,
		CustomizationTTL: customTTL,
		Now:              clk.Now,
	})
	return cache, clk
}

func TestCareerLiveCache_Defaults(t *testing.T) {
	cache := NewCareerLiveCache(CareerLiveCacheConfig{})
	if cache.progressTTL != DefaultCareerProgressTTL {
		t.Errorf("progressTTL default = %v, want %v", cache.progressTTL, DefaultCareerProgressTTL)
	}
	if cache.customTTL != DefaultCareerCustomizationTTL {
		t.Errorf("customTTL default = %v, want %v", cache.customTTL, DefaultCareerCustomizationTTL)
	}
}

func TestCareerLiveCache_ProgressMiss_ThenHit(t *testing.T) {
	cache, _ := newTestCache(t, 5*time.Minute, 6*time.Hour)
	if _, hit := cache.GetProgress("xuid1", testSlug); hit {
		t.Fatal("miss expected on first Get")
	}
	cache.PutProgress("xuid1", testSlug, &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1000})
	got, hit := cache.GetProgress("xuid1", testSlug)
	if !hit {
		t.Fatal("hit expected after Put")
	}
	if got.CurrentRank != 42 || got.CurrentXP != 1000 {
		t.Errorf("cached value mismatch: %+v", got)
	}
}

func TestCareerLiveCache_ProgressExpires(t *testing.T) {
	cache, clk := newTestCache(t, 5*time.Minute, 6*time.Hour)
	cache.PutProgress("xuid1", testSlug, &syncpkg.CareerRankData{CurrentRank: 1})

	clk.Advance(4 * time.Minute)
	if _, hit := cache.GetProgress("xuid1", testSlug); !hit {
		t.Fatal("hit expected before TTL")
	}
	clk.Advance(2 * time.Minute) // total 6 min > 5 min TTL
	if _, hit := cache.GetProgress("xuid1", testSlug); hit {
		t.Fatal("miss expected after TTL")
	}
}

func TestCareerLiveCache_CustomizationExpires(t *testing.T) {
	cache, clk := newTestCache(t, 5*time.Minute, 6*time.Hour)
	cache.PutCustomization("xuid1", testSlug, &syncpkg.SpartanCustomizationData{SpartanID: "SR-001"})

	clk.Advance(5 * time.Hour)
	if _, hit := cache.GetCustomization("xuid1", testSlug); !hit {
		t.Fatal("hit expected before customization TTL")
	}
	clk.Advance(2 * time.Hour) // total 7 h > 6 h TTL
	if _, hit := cache.GetCustomization("xuid1", testSlug); hit {
		t.Fatal("miss expected after customization TTL")
	}
}

func TestCareerLiveCache_DistinctXUIDs(t *testing.T) {
	cache, _ := newTestCache(t, 5*time.Minute, 6*time.Hour)
	cache.PutProgress("a", testSlug, &syncpkg.CareerRankData{CurrentRank: 1})
	cache.PutProgress("b", testSlug, &syncpkg.CareerRankData{CurrentRank: 2})
	if _, hit := cache.GetProgress("c", testSlug); hit {
		t.Fatal("miss expected for unknown xuid")
	}
	a, hitA := cache.GetProgress("a", testSlug)
	b, hitB := cache.GetProgress("b", testSlug)
	if !hitA || !hitB || a.CurrentRank != 1 || b.CurrentRank != 2 {
		t.Errorf("isolation broken: a=%+v hit=%v b=%+v hit=%v", a, hitA, b, hitB)
	}
}

// TestCareerLiveCache_DistinctTitles vérifie l'isolation par titre (V72-29) :
// un même xuid porte des entrées indépendantes selon le titre, et une lecture
// sur un autre titre que celui écrit renvoie un miss (jamais de fuite cross-titre).
func TestCareerLiveCache_DistinctTitles(t *testing.T) {
	cache, _ := newTestCache(t, 5*time.Minute, 6*time.Hour)
	cache.PutProgress("shared-xuid", "halo_infinite", &syncpkg.CareerRankData{CurrentRank: 272})
	cache.PutProgress("shared-xuid", "halo_5", &syncpkg.CareerRankData{CurrentRank: 152})

	hi, hitHI := cache.GetProgress("shared-xuid", "halo_infinite")
	h5, hitH5 := cache.GetProgress("shared-xuid", "halo_5")
	if !hitHI || !hitH5 || hi.CurrentRank != 272 || h5.CurrentRank != 152 {
		t.Errorf("isolation par titre cassée: hi=%+v hit=%v h5=%+v hit=%v", hi, hitHI, h5, hitH5)
	}

	// Un titre jamais écrit sous ce xuid ne doit rien renvoyer (pas de fuite).
	if _, hit := cache.GetProgress("shared-xuid", "halo_mcc"); hit {
		t.Error("miss attendu pour un titre non écrit (fuite cross-titre)")
	}
}

// TestCareerLiveCache_DoProgress_SingleflightDedupes vérifie que plusieurs
// goroutines appelant DoProgress simultanément pour le même (xuid, titre) voient
// fn invoqué une seule fois (pattern singleflight).
func TestCareerLiveCache_DoProgress_SingleflightDedupes(t *testing.T) {
	cache, _ := newTestCache(t, 5*time.Minute, 6*time.Hour)

	var calls int32
	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	const concurrency = 8
	results := make([]*syncpkg.CareerRankData, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		done.Add(1)
		go func(idx int) {
			defer done.Done()
			start.Wait()
			results[idx], errs[idx] = cache.DoProgress("xuid-sf", testSlug, func() (*syncpkg.CareerRankData, error) {
				atomic.AddInt32(&calls, 1)
				// Simule un fetch lent pour que les autres goroutines aient le
				// temps d'arriver sur le même singleflight.
				time.Sleep(10 * time.Millisecond)
				return &syncpkg.CareerRankData{CurrentRank: 7}, nil
			})
		}(i)
	}
	start.Done()
	done.Wait()

	got := atomic.LoadInt32(&calls)
	if got != 1 {
		t.Errorf("singleflight: fn appelé %d fois, attendu 1", got)
	}
	for i := 0; i < concurrency; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d: erreur inattendue %v", i, errs[i])
		}
		if results[i] == nil || results[i].CurrentRank != 7 {
			t.Errorf("goroutine %d: result mismatch %+v", i, results[i])
		}
	}
}

// TestCareerLiveCache_DoProgress_ErrorNotCached vérifie qu'une erreur ne
// pollue pas le cache : un Do suivant doit refaire l'appel.
func TestCareerLiveCache_DoProgress_ErrorNotCached(t *testing.T) {
	cache, _ := newTestCache(t, 5*time.Minute, 6*time.Hour)

	var calls int
	wantErr := errors.New("first call fails")

	_, err := cache.DoProgress("xuid-err", testSlug, func() (*syncpkg.CareerRankData, error) {
		calls++
		return nil, wantErr
	})
	if err != wantErr {
		t.Errorf("première erreur attendue %v, obtenue %v", wantErr, err)
	}

	got, err := cache.DoProgress("xuid-err", testSlug, func() (*syncpkg.CareerRankData, error) {
		calls++
		return &syncpkg.CareerRankData{CurrentRank: 1}, nil
	})
	if err != nil || got == nil || got.CurrentRank != 1 {
		t.Errorf("second call: got=%+v err=%v", got, err)
	}
	if calls != 2 {
		t.Errorf("fn appelé %d fois, attendu 2 (erreur ne doit pas cacher)", calls)
	}
}

func TestCareerLiveCache_DoCustomization_SingleflightDedupes(t *testing.T) {
	cache, _ := newTestCache(t, 5*time.Minute, 6*time.Hour)

	var calls int32
	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	const concurrency = 8

	for i := 0; i < concurrency; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			_, _ = cache.DoCustomization("xuid-sf-c", testSlug, func() (*syncpkg.SpartanCustomizationData, error) {
				atomic.AddInt32(&calls, 1)
				time.Sleep(10 * time.Millisecond)
				return &syncpkg.SpartanCustomizationData{SpartanID: "tag"}, nil
			})
		}()
	}
	start.Done()
	done.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("singleflight customization: fn appelé %d fois, attendu 1", got)
	}
}
