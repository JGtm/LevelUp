package wire

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
)

func key(match string) participationKey {
	return participationKey{title: "halo_infinite", xuid: "xuid(1)", matchID: match}
}

// countingLoad rend un chargeur dont chaque appel est compté.
func countingLoad(calls *atomic.Int32, participated bool, err error) func(context.Context) (bool, error) {
	return func(context.Context) (bool, error) {
		calls.Add(1)
		return participated, err
	}
}

func TestParticipationCache_PositiveIsLoadedOnce(t *testing.T) {
	var c participationCache
	var calls atomic.Int32
	for i := 0; i < 2; i++ {
		got, err := c.check(context.Background(), key("m1"), countingLoad(&calls, true, nil))
		if err != nil || !got {
			t.Fatalf("appel %d : got=%v err=%v, attendu true sans erreur", i+1, got, err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("requêtes = %d, attendu 1 pour deux appels", n)
	}
}

func TestParticipationCache_KeyIsTitleXuidAndMatch(t *testing.T) {
	var c participationCache
	var calls atomic.Int32
	keys := []participationKey{
		key("m1"),
		key("m2"),
		{title: "halo_5", xuid: "xuid(1)", matchID: "m1"},
		{title: "halo_infinite", xuid: "xuid(2)", matchID: "m1"},
	}
	for _, k := range keys {
		if _, err := c.check(context.Background(), k, countingLoad(&calls, true, nil)); err != nil {
			t.Fatal(err)
		}
	}
	if n := calls.Load(); int(n) != len(keys) {
		t.Fatalf("requêtes = %d, attendu %d : un fait d'un autre titre, joueur ou match ne répond pas pour celui-ci", n, len(keys))
	}
}

func TestParticipationCache_FalseIsNeverCached(t *testing.T) {
	var c participationCache
	var calls atomic.Int32
	for i := 0; i < 2; i++ {
		got, err := c.check(context.Background(), key("m1"), countingLoad(&calls, false, nil))
		if err != nil || got {
			t.Fatalf("appel %d : got=%v err=%v, attendu false sans erreur", i+1, got, err)
		}
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("requêtes = %d, attendu 2 : un refus ne se mémorise pas", n)
	}
	// Le match est synchronisé ensuite : la participation devient vraie et se lit.
	got, err := c.check(context.Background(), key("m1"), countingLoad(&calls, true, nil))
	if err != nil || !got {
		t.Fatalf("après synchronisation : got=%v err=%v, attendu true", got, err)
	}
}

func TestParticipationCache_ErrorIsNeverCached(t *testing.T) {
	var c participationCache
	var calls atomic.Int32
	boom := errors.New("context deadline exceeded")
	for i := 0; i < 2; i++ {
		if _, err := c.check(context.Background(), key("m1"), countingLoad(&calls, false, boom)); !errors.Is(err, boom) {
			t.Fatalf("appel %d : err=%v, attendu l'erreur de la requête", i+1, err)
		}
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("requêtes = %d, attendu 2 : une erreur ne se mémorise pas", n)
	}
	// Une erreur accompagnée d'un true douteux n'est pas mémorisée non plus.
	if _, err := c.check(context.Background(), key("m2"), countingLoad(&calls, true, boom)); !errors.Is(err, boom) {
		t.Fatalf("err=%v, attendu l'erreur de la requête", err)
	}
	if c.has(key("m2")) {
		t.Fatal("une participation rendue avec une erreur a été mémorisée")
	}
}

func TestParticipationCache_SizeIsBoundedAndOldestIsEvicted(t *testing.T) {
	const capacity = 3
	c := participationCache{max: capacity}
	var calls atomic.Int32
	for i := 0; i < 10; i++ {
		if _, err := c.check(context.Background(), key("m"+strconv.Itoa(i)), countingLoad(&calls, true, nil)); err != nil {
			t.Fatal(err)
		}
		if len(c.seen) > capacity || len(c.order) > capacity {
			t.Fatalf("après %d ajouts : %d clés, %d positions, borne %d", i+1, len(c.seen), len(c.order), capacity)
		}
	}
	// Les trois plus récentes restent, les plus anciennes sont sorties.
	for _, m := range []string{"m7", "m8", "m9"} {
		if !c.has(key(m)) {
			t.Errorf("%s (récent) a été évincé", m)
		}
	}
	for _, m := range []string{"m0", "m6"} {
		if c.has(key(m)) {
			t.Errorf("%s (ancien) est resté", m)
		}
	}
	// Une entrée évincée se relit par la requête, puis se remémorise.
	before := calls.Load()
	if _, err := c.check(context.Background(), key("m0"), countingLoad(&calls, true, nil)); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != before+1 {
		t.Fatal("une entrée évincée n'a pas été relue par la requête")
	}
}

// TestParticipationCache_503StaysClosedOnUncachedFailure : branché au middleware, un échec
// non mémorisé reste un 503 fermé, et la requête suivante qui réussit passe puis se mémorise.
func TestParticipationCache_503StaysClosedOnUncachedFailure(t *testing.T) {
	var c participationCache
	var calls atomic.Int32
	failing := true
	check := func(ctx context.Context, _, matchID string) (bool, error) {
		return c.check(ctx, key(matchID), func(context.Context) (bool, error) {
			calls.Add(1)
			if failing {
				return false, errors.New("context deadline exceeded")
			}
			return true, nil
		})
	}
	r := chi.NewRouter()
	r.With(middleware.RequireMatchParticipation(check)).
		Get("/players/{player_slug}/matches/{match_id}", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	const matchID = "11111111-2222-3333-4444-555555555555"
	do := func() int {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/players/JGtm/matches/"+matchID, nil))
		return rec.Code
	}

	if code := do(); code != http.StatusServiceUnavailable {
		t.Fatalf("échec de la requête : %d, attendu 503 (fermé par défaut)", code)
	}
	if code := do(); code != http.StatusServiceUnavailable {
		t.Fatalf("second échec : %d, attendu 503 : l'échec n'est pas mémorisé", code)
	}
	failing = false
	if code := do(); code != http.StatusOK {
		t.Fatalf("requête rétablie : %d, attendu 200", code)
	}
	failing = true // la base retombe : la participation confirmée reste servie
	if code := do(); code != http.StatusOK {
		t.Fatalf("participation confirmée : %d, attendu 200 sans requête", code)
	}
	if n := calls.Load(); n != 3 {
		t.Fatalf("requêtes = %d, attendu 3 (deux échecs, un succès) ; la quatrième est servie par le cache", n)
	}
}
