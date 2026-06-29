package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

type stubLocalSearch struct {
	results []domain.GamertagSearchResult
	err     error
	calls   int
}

func (s *stubLocalSearch) Search(_ context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	s.calls++
	return s.results, s.err
}

type stubResolver struct {
	xuid  string
	err   error
	calls int
}

func (r *stubResolver) ResolveXUID(_ context.Context, _ string) (string, error) {
	r.calls++
	return r.xuid, r.err
}

func TestLiveFallback_NilResolver_Passthrough(t *testing.T) {
	local := &stubLocalSearch{results: []domain.GamertagSearchResult{{Gamertag: "Foo", XUID: "1"}}}
	svc := NewLiveFallbackGamertagSearch(local, nil)
	got, err := svc.Search(context.Background(), "Bar")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("attendu 1 résultat (passthrough local), got %d", len(got))
	}
}

func TestLiveFallback_ExactLocalMatch_SkipsResolver(t *testing.T) {
	local := &stubLocalSearch{results: []domain.GamertagSearchResult{{Gamertag: "Foo", XUID: "1"}}}
	res := &stubResolver{xuid: "999"}
	svc := NewLiveFallbackGamertagSearch(local, res)
	got, err := svc.Search(context.Background(), "foo") // casse différente, match exact
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.calls != 0 {
		t.Fatalf("résolveur ne doit PAS être appelé sur match exact local (calls=%d)", res.calls)
	}
	if len(got) != 1 {
		t.Fatalf("attendu 1 résultat, got %d", len(got))
	}
}

func TestLiveFallback_LocalEmpty_AppendsSynthetic(t *testing.T) {
	local := &stubLocalSearch{results: nil}
	res := &stubResolver{xuid: "2533274812345678"}
	svc := NewLiveFallbackGamertagSearch(local, res)
	got, err := svc.Search(context.Background(), "NeverSeen")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.calls != 1 {
		t.Fatalf("résolveur attendu 1 appel, got %d", res.calls)
	}
	if len(got) != 1 {
		t.Fatalf("attendu 1 résultat synthétique, got %d", len(got))
	}
	r := got[0]
	if r.Gamertag != "NeverSeen" || r.XUID != "2533274812345678" || !r.ExactMatch || r.Score != 0 {
		t.Fatalf("résultat synthétique inattendu: %+v", r)
	}
}

func TestLiveFallback_Throttle_DegradesAndNegCaches(t *testing.T) {
	local := &stubLocalSearch{results: nil}
	res := &stubResolver{err: errors.New("xbox profile HTTP 429: rate limited")}
	svc := NewLiveFallbackGamertagSearch(local, res)

	got, err := svc.Search(context.Background(), "Throttled")
	if err != nil {
		t.Fatalf("throttle ne doit jamais propager d'erreur: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("attendu 0 résultat (local vide), got %d", len(got))
	}
	// 2e recherche identique : cache négatif → pas de ré-appel résolveur.
	if _, err := svc.Search(context.Background(), "Throttled"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.calls != 1 {
		t.Fatalf("cache négatif attendu : 1 seul appel résolveur, got %d", res.calls)
	}
}

func TestLiveFallback_NegCacheExpires(t *testing.T) {
	local := &stubLocalSearch{results: nil}
	res := &stubResolver{err: errors.New("not found")}
	svc := NewLiveFallbackGamertagSearch(local, res)
	base := time.Unix(1_700_000_000, 0)
	svc.now = func() time.Time { return base }
	if _, err := svc.Search(context.Background(), "Gone"); err != nil {
		t.Fatal(err)
	}
	// après expiration du TTL négatif → ré-appel autorisé.
	svc.now = func() time.Time { return base.Add(liveFallbackNegTTL + time.Second) }
	if _, err := svc.Search(context.Background(), "Gone"); err != nil {
		t.Fatal(err)
	}
	if res.calls != 2 {
		t.Fatalf("après expiry, attendu 2 appels résolveur, got %d", res.calls)
	}
}

func TestLiveFallback_NotFound_DegradesNoError(t *testing.T) {
	local := &stubLocalSearch{results: []domain.GamertagSearchResult{{Gamertag: "Sub", XUID: "1"}}}
	res := &stubResolver{err: errors.New("xbox profile: aucun profil pour gamertag")}
	svc := NewLiveFallbackGamertagSearch(local, res)
	got, err := svc.Search(context.Background(), "Unknownnn")
	if err != nil {
		t.Fatalf("not-found ne doit pas propager: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("attendu résultats locaux inchangés, got %d", len(got))
	}
}

func TestLiveFallback_ImplausibleQuery_SkipsResolver(t *testing.T) {
	local := &stubLocalSearch{}
	res := &stubResolver{xuid: "1"}
	svc := NewLiveFallbackGamertagSearch(local, res)
	for _, q := range []string{"ab", "x", "a*b", "  "} {
		if _, err := svc.Search(context.Background(), q); err != nil {
			t.Fatalf("q=%q err: %v", q, err)
		}
	}
	if res.calls != 0 {
		t.Fatalf("résolveur ne doit pas être appelé sur query implausible (calls=%d)", res.calls)
	}
}

func TestLiveFallback_DedupByXUID(t *testing.T) {
	local := &stubLocalSearch{results: []domain.GamertagSearchResult{{Gamertag: "AliasOld", XUID: "777"}}}
	res := &stubResolver{xuid: "777"} // même xuid, libellé différent
	svc := NewLiveFallbackGamertagSearch(local, res)
	got, err := svc.Search(context.Background(), "AliasNew")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("dédup par xuid attendu (pas d'ajout), got %d", len(got))
	}
}

func TestLiveFallback_LocalError_Propagates(t *testing.T) {
	local := &stubLocalSearch{err: errors.New("db down")}
	res := &stubResolver{xuid: "1"}
	svc := NewLiveFallbackGamertagSearch(local, res)
	if _, err := svc.Search(context.Background(), "Whatever"); err == nil {
		t.Fatal("une erreur locale réelle doit être propagée")
	}
	if res.calls != 0 {
		t.Fatalf("résolveur ne doit pas être appelé si le local échoue (calls=%d)", res.calls)
	}
}

func TestIsPlausibleGamertag(t *testing.T) {
	cases := map[string]bool{
		"Foo":         true,
		"Foo Bar":     true,
		"Player 12":   true,
		"Gamer#1234":  true,
		"ab":          false, // trop court
		"   ":         false, // que des espaces
		"a*b":         false, // symbole interdit
		"DemoPlayer2": true,
	}
	for q, want := range cases {
		if got := isPlausibleGamertag(q); got != want {
			t.Errorf("isPlausibleGamertag(%q)=%v, want %v", q, got, want)
		}
	}
}

func TestHasExactMatch(t *testing.T) {
	rs := []domain.GamertagSearchResult{{Gamertag: "Foo"}, {Gamertag: "Bar"}}
	if !hasExactMatch(rs, "foo") {
		t.Error("match exact casse-insensible attendu")
	}
	if hasExactMatch(rs, "Baz") {
		t.Error("pas de match attendu pour Baz")
	}
}
