package assetnames

import (
	"context"
	"errors"
	"testing"
)

// mockFetcher compte les appels et renvoie un nom par (assetID|lang).
type mockFetcher struct {
	calls int
	names map[string]string // clé assetID|lang → nom ("" = pas de nom)
	err   error             // si non nil, FetchName échoue toujours
}

func (m *mockFetcher) FetchName(_ context.Context, _, _, assetID, _, lang string) (string, error) {
	m.calls++
	if m.err != nil {
		return "", m.err
	}
	return m.names[assetID+"|"+lang], nil
}

// mockStore simule asset_translations en mémoire.
type mockStore struct {
	fresh    map[string]bool   // clé assetType|assetID|lang → déjà présent
	upserts  map[string]string // clé assetType|assetID|lang → nom écrit
	freshErr error
}

func newMockStore() *mockStore {
	return &mockStore{fresh: map[string]bool{}, upserts: map[string]string{}}
}

func (m *mockStore) ExistsFresh(_ context.Context, assetType, assetID, lang string) (bool, error) {
	if m.freshErr != nil {
		return false, m.freshErr
	}
	return m.fresh[assetType+"|"+assetID+"|"+lang], nil
}

func (m *mockStore) Upsert(_ context.Context, assetType, assetID, lang, name string) error {
	m.upserts[assetType+"|"+assetID+"|"+lang] = name
	return nil
}

func TestResolve_MultiLangUpsert(t *testing.T) {
	f := &mockFetcher{names: map[string]string{
		"pl1|fr-FR": "Partie rapide",
		"pl1|en-US": "Quick Play",
	}}
	s := newMockStore()
	res, err := Resolve(context.Background(), f, s,
		[]AssetRef{{AssetType: "playlist", AssetID: "pl1"}}, Config{TitleID: "hi"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Requested != 1 || res.Resolved != 1 {
		t.Fatalf("compteurs: %+v", res)
	}
	if got := s.upserts["playlist|pl1|fr-FR"]; got != "Partie rapide" {
		t.Errorf("upsert fr-FR = %q", got)
	}
	if got := s.upserts["playlist|pl1|en-US"]; got != "Quick Play" {
		t.Errorf("upsert en-US = %q", got)
	}
}

func TestResolve_Dedup(t *testing.T) {
	f := &mockFetcher{names: map[string]string{"m1|fr-FR": "Aquarius", "m1|en-US": "Aquarius"}}
	s := newMockStore()
	refs := []AssetRef{
		{AssetType: "map", AssetID: "m1"},
		{AssetType: "map", AssetID: "m1", VersionID: "v2"}, // doublon
	}
	res, _ := Resolve(context.Background(), f, s, refs, Config{TitleID: "hi"})
	if res.Requested != 1 {
		t.Fatalf("dedup: Requested = %d, want 1", res.Requested)
	}
	if f.calls != 2 { // 1 asset × 2 langs
		t.Fatalf("dedup: fetch calls = %d, want 2", f.calls)
	}
}

func TestResolve_SkipFresh(t *testing.T) {
	f := &mockFetcher{names: map[string]string{}}
	s := newMockStore()
	s.fresh["playlist|pl1|fr-FR"] = true
	s.fresh["playlist|pl1|en-US"] = true
	res, _ := Resolve(context.Background(), f, s,
		[]AssetRef{{AssetType: "playlist", AssetID: "pl1"}}, Config{TitleID: "hi"})
	if res.Skipped != 1 || res.Resolved != 0 {
		t.Fatalf("skip-fresh: %+v", res)
	}
	if f.calls != 0 {
		t.Fatalf("skip-fresh: %d fetch calls, want 0", f.calls)
	}
}

func TestResolve_Cap(t *testing.T) {
	f := &mockFetcher{names: map[string]string{}}
	s := newMockStore()
	refs := []AssetRef{
		{AssetType: "map", AssetID: "m1"},
		{AssetType: "map", AssetID: "m2"},
		{AssetType: "map", AssetID: "m3"},
	}
	res, _ := Resolve(context.Background(), f, s, refs, Config{TitleID: "hi", MaxAssets: 2})
	if res.Requested != 3 || res.Capped != 1 {
		t.Fatalf("cap: %+v", res)
	}
}

func TestResolve_BestEffortFetchError(t *testing.T) {
	// 1 asset résout, l'autre échoue au fetch → erreurs comptées, pas de panic.
	f := &mockFetcher{
		names: map[string]string{"ok|fr-FR": "Bon", "ok|en-US": "Good"},
		err:   nil,
	}
	s := newMockStore()
	// On force l'échec uniquement via un fetcher qui erre pour "bad" : on simule
	// avec un fetcher dédié par sous-test.
	res, _ := Resolve(context.Background(), f, s,
		[]AssetRef{{AssetType: "playlist", AssetID: "ok"}}, Config{TitleID: "hi"})
	if res.Resolved != 1 {
		t.Fatalf("best-effort ok: %+v", res)
	}

	fErr := &mockFetcher{err: errors.New("boom")}
	res2, _ := Resolve(context.Background(), fErr, newMockStore(),
		[]AssetRef{{AssetType: "playlist", AssetID: "bad"}}, Config{TitleID: "hi"})
	if res2.Resolved != 0 || res2.Errors == 0 {
		t.Fatalf("best-effort error: %+v", res2)
	}
}

func TestResolve_EmptyNameNotWritten(t *testing.T) {
	f := &mockFetcher{names: map[string]string{}} // tout vide
	s := newMockStore()
	res, _ := Resolve(context.Background(), f, s,
		[]AssetRef{{AssetType: "map", AssetID: "m1"}}, Config{TitleID: "hi"})
	if res.Resolved != 0 {
		t.Fatalf("empty-name: Resolved = %d, want 0", res.Resolved)
	}
	if len(s.upserts) != 0 {
		t.Fatalf("empty-name: %d upserts, want 0", len(s.upserts))
	}
}

func TestResolve_NilDeps(t *testing.T) {
	res, err := Resolve(context.Background(), nil, nil,
		[]AssetRef{{AssetType: "map", AssetID: "m1"}}, Config{})
	if err != nil || res.Requested != 0 {
		t.Fatalf("nil deps: %+v err=%v", res, err)
	}
}

func TestResolve_FreshErrorCountsAsError(t *testing.T) {
	f := &mockFetcher{names: map[string]string{"m1|fr-FR": "X", "m1|en-US": "X"}}
	s := newMockStore()
	s.freshErr = errors.New("db down")
	res, _ := Resolve(context.Background(), f, s,
		[]AssetRef{{AssetType: "map", AssetID: "m1"}}, Config{TitleID: "hi"})
	if res.Errors == 0 {
		t.Fatalf("fresh-error: Errors = 0, want > 0 (%+v)", res)
	}
	if res.Skipped != 0 {
		t.Fatalf("fresh-error: Skipped = %d, want 0", res.Skipped)
	}
}
