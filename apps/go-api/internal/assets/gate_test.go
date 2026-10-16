package assets

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// spyResolver compte les appels sortants (Get) et sert un stock local fixe.
type spyResolver struct {
	local map[string]Resolved
	gets  atomic.Int32
	err   error
}

func (s *spyResolver) LookupLocal(_ context.Context, ref Ref) (Resolved, bool) {
	r, ok := s.local[ref.String()]
	return r, ok
}

func (s *spyResolver) Get(_ context.Context, ref Ref) (Resolved, error) {
	s.gets.Add(1)
	if r, ok := s.local[ref.String()]; ok {
		return r, nil
	}
	if s.err != nil {
		return Resolved{}, s.err
	}
	return Resolved{Payload: URLPayload{URL: "https://cdn.example.test/x.png"}, Source: SourceRemote}, nil
}

// mapCatalog classe par table : kind+id → verdict ; défaut inconnu.
type mapCatalog struct {
	verdicts map[string]Verdict
	calls    atomic.Int32
}

func (c *mapCatalog) Classify(_ context.Context, ref Ref) Verdict {
	c.calls.Add(1)
	return c.verdicts[ref.String()]
}

var (
	medalUnknown = Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "999999"}
	medalKnown   = Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}
	bpUnverif    = Ref{Kind: KindBPTrackImage, TitleID: "halo_infinite", ID: "/tracks/a.png", Variant: "tracks"}
)

func newTestGate(local map[string]Resolved) (*Gate, *spyResolver, *mapCatalog) {
	spy := &spyResolver{local: local}
	cat := &mapCatalog{verdicts: map[string]Verdict{
		medalKnown.String(): VerdictKnown,
		bpUnverif.String():  VerdictUnverifiable,
	}}
	return NewGate(spy, cat), spy, cat
}

// Un identifiant absent du catalogue local répond ErrNotFound sans aucun appel sortant,
// quel que soit le nombre de requêtes et même pour un appelant de confiance.
func TestGate_UnknownIDNeverReachesTheFetcher(t *testing.T) {
	g, spy, _ := newTestGate(nil)
	for _, trusted := range []bool{false, true} {
		for range 50 {
			if _, err := g.Get(context.Background(), medalUnknown, trusted); !errors.Is(err, ErrNotFound) {
				t.Fatalf("trusted=%v: err = %v, attendu ErrNotFound", trusted, err)
			}
		}
	}
	if got := spy.gets.Load(); got != 0 {
		t.Fatalf("appels sortants = %d, attendu 0", got)
	}
}

// Le refus d'un identifiant inconnu est mémorisé (cache négatif) : le catalogue n'est
// pas réinterrogé à chaque requête.
func TestGate_UnknownIDIsNegativelyCached(t *testing.T) {
	g, _, cat := newTestGate(nil)
	for range 20 {
		_, _ = g.Get(context.Background(), medalUnknown, false)
	}
	if got := cat.calls.Load(); got != 1 {
		t.Fatalf("interrogations du catalogue = %d, attendu 1", got)
	}
}

// Un identifiant connu est résolu (fetch autorisé), pour tout appelant.
func TestGate_KnownIDIsFetched(t *testing.T) {
	g, spy, _ := newTestGate(nil)
	if _, err := g.Get(context.Background(), medalKnown, false); err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := spy.gets.Load(); got != 1 {
		t.Fatalf("appels sortants = %d, attendu 1", got)
	}
}

// Un asset déjà en local est servi sans consulter le catalogue ni la source distante.
func TestGate_LocalHitSkipsCatalogAndFetcher(t *testing.T) {
	local := map[string]Resolved{medalUnknown.String(): {Payload: BinaryPayload{Bytes: []byte{1}}, Source: SourceLocalFile}}
	g, spy, cat := newTestGate(local)
	res, err := g.Get(context.Background(), medalUnknown, false)
	if err != nil || res.Source != SourceLocalFile {
		t.Fatalf("res = %+v err = %v", res, err)
	}
	if spy.gets.Load() != 0 || cat.calls.Load() != 0 {
		t.Fatalf("gets=%d catalogue=%d, attendu 0/0", spy.gets.Load(), cat.calls.Load())
	}
}

// Un kind sans catalogue local n'est récupéré à distance que pour un appelant de
// confiance ; un anonyme est servi depuis le local seulement.
func TestGate_UnverifiableKindFetchesOnlyForTrustedCallers(t *testing.T) {
	g, spy, _ := newTestGate(nil)
	if _, err := g.Get(context.Background(), bpUnverif, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("anonyme: err = %v, attendu ErrNotFound", err)
	}
	if spy.gets.Load() != 0 {
		t.Fatalf("anonyme: appels sortants = %d, attendu 0", spy.gets.Load())
	}
	if _, err := g.Get(context.Background(), bpUnverif, true); err != nil {
		t.Fatalf("appelant de confiance: err = %v", err)
	}
	if spy.gets.Load() != 1 {
		t.Fatalf("appelant de confiance: appels sortants = %d, attendu 1", spy.gets.Load())
	}
}

// Un 404 amont sur un identifiant connu est mémorisé : pas de nouvel appel sortant.
func TestGate_UpstreamNotFoundIsNegativelyCached(t *testing.T) {
	g, spy, _ := newTestGate(nil)
	spy.err = ErrNotFound
	for range 10 {
		if _, err := g.Get(context.Background(), medalKnown, false); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	}
	if got := spy.gets.Load(); got != 1 {
		t.Fatalf("appels sortants = %d, attendu 1", got)
	}
}

// Une indisponibilité amont n'est pas mémorisée (elle peut cesser).
func TestGate_UpstreamErrorIsNotCached(t *testing.T) {
	g, spy, _ := newTestGate(nil)
	spy.err = ErrUpstreamUnavailable
	for range 3 {
		_, _ = g.Get(context.Background(), medalKnown, false)
	}
	if got := spy.gets.Load(); got != 3 {
		t.Fatalf("appels sortants = %d, attendu 3", got)
	}
}

// Le cache négatif est borné.
func TestNegativeCache_IsBounded(t *testing.T) {
	c := newNegativeCache(8, time.Minute)
	for i := range 100 {
		c.add(Ref{Kind: KindMedalImage, TitleID: "t", ID: string(rune('a'+i%26)) + string(rune('0'+i/26))}.String())
	}
	if n := c.len(); n > 8 {
		t.Fatalf("taille = %d, plafond 8", n)
	}
}

// Une référence mal formée est refusée avant le stock local, le catalogue et la source
// distante, et n'entre pas dans le cache négatif.
func TestGate_InvalidRefIsRefusedBeforeAnyWork(t *testing.T) {
	g, spy, cat := newTestGate(nil)
	ref := Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: "/../../secret"}
	if _, err := g.Get(context.Background(), ref, true); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("err = %v, attendu ErrInvalidRef", err)
	}
	if spy.gets.Load() != 0 || cat.calls.Load() != 0 || g.refused.len() != 0 {
		t.Fatalf("travail engagé : appels sortants=%d, catalogue=%d, refus mémorisés=%d",
			spy.gets.Load(), cat.calls.Load(), g.refused.len())
	}
}

// Un catalogue illisible (base indisponible, table absente) ne vaut pas « inconnu » : la
// référence est refusée sans appel sortant, mais n'entre pas dans le cache négatif — la
// requête suivante interroge de nouveau le catalogue.
func TestGate_CatalogReadErrorIsNotNegativelyCached(t *testing.T) {
	unreadable := NewDuckDBIndexStore(t.TempDir()) // un dossier : la base ne s'ouvre pas
	t.Cleanup(unreadable.Close)
	cat := NewMetadataCatalog(unreadable, func(string) bool { return true })
	spy := &spyResolver{}
	g := NewGate(spy, cat)

	for range 2 {
		if _, err := g.Get(context.Background(), medalKnown, true); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, attendu ErrNotFound", err)
		}
	}
	if n := g.refused.len(); n != 0 {
		t.Fatalf("%d refus mémorisé(s) sur une erreur de lecture du catalogue, attendu 0", n)
	}
	if spy.gets.Load() != 0 {
		t.Fatalf("appels sortants = %d, attendu 0", spy.gets.Load())
	}
}
