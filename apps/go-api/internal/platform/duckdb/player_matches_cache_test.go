// player_matches_cache_test.go — historique canonique enrichi derrière le cache des
// lectures joueur (plan perf 2026-09-23, lot L5b, D5b.4). Aucune base : source
// factice.
package duckdb

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fakePlayerMatches compte ses chargements et rend des lignes neuves à chaque
// appel, avec des AssetReference porteuses de libellés (comme l'adapter enrichi).
type fakePlayerMatches struct {
	calls      atomic.Int32
	lobbyCalls atomic.Int32
	delay      time.Duration
}

func (f *fakePlayerMatches) LoadPlayerMatches(_ context.Context, _, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	f.calls.Add(1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	ref := func(kind, id, fr string) *canonical.AssetReference {
		return &canonical.AssetReference{Kind: kind, ID: id, DefaultLabel: id, Labels: map[string]string{"fr": fr, "en": id}}
	}
	return []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{MatchID: "m1", Map: ref("map", "Aquarius", "Aquarius FR"),
			Playlist: ref("playlist", "Ranked", "Classé"), PairMode: ref("pair", "Slayer", "Assassin")}},
		{Summary: canonical.MatchSummary{MatchID: "m2", Map: ref("map", "Streets", "Rues")}},
	}, nil
}

func (f *fakePlayerMatches) LobbySizesAtCompletion(_ context.Context, _ string, ids []string) (map[string]int, error) {
	f.lobbyCalls.Add(1)
	return map[string]int{ids[0]: 8}, nil
}

func newTestPlayerMatches(src *fakePlayerMatches, xuid string) *CachedPlayerMatchesRepo {
	return &CachedPlayerMatchesRepo{
		inner: src,
		cache: newPlayerReadCache("player_matches_cache", clonePlayerMatchRows),
		scope: testScope(xuid, "halo_infinite", "p"),
	}
}

func TestCachedPlayerMatchesRepo_HitPerFilters(t *testing.T) {
	t.Parallel()
	src := &fakePlayerMatches{}
	repo := newTestPlayerMatches(src, "x1")
	oneYear, oneMonth := temporal.Period1Y, temporal.Period1M
	for i := 0; i < 2; i++ {
		for _, f := range []port.PlayerMatchFilters{{}, {Period: &oneYear}, {Period: &oneMonth}} {
			if rows, err := repo.LoadPlayerMatches(context.Background(), "halo_infinite", "GT", f); err != nil || len(rows) != 2 {
				t.Fatalf("chargement : %d lignes, err=%v", len(rows), err)
			}
		}
	}
	if got := src.calls.Load(); got != 3 {
		t.Errorf("chargements = %d, want 3 (une variante par jeu de filtres, puis hits)", got)
	}
}

func TestCachedPlayerMatchesRepo_OutcomeOrderInsensitive(t *testing.T) {
	t.Parallel()
	src := &fakePlayerMatches{}
	repo := newTestPlayerMatches(src, "x1")
	a := port.PlayerMatchFilters{OutcomeIn: []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss}}
	b := port.PlayerMatchFilters{OutcomeIn: []canonical.Outcome{canonical.OutcomeLoss, canonical.OutcomeWin}}
	_, _ = repo.LoadPlayerMatches(context.Background(), "", "", a)
	_, _ = repo.LoadPlayerMatches(context.Background(), "", "", b)
	if got := src.calls.Load(); got != 1 {
		t.Errorf("mêmes filtres dans un autre ordre : %d chargements, want 1", got)
	}
}

// TestCachedPlayerMatchesRepo_ReturnsCopies : re-enrichir (écrire Labels, DefaultLabel,
// IconURL comme EnrichCanonicalAssetTranslations), trier ou modifier les lignes
// rendues ne touche ni le cache ni les autres lecteurs.
func TestCachedPlayerMatchesRepo_ReturnsCopies(t *testing.T) {
	t.Parallel()
	repo := newTestPlayerMatches(&fakePlayerMatches{}, "x1")
	for i := 0; i < 2; i++ { // 1re lecture = chargement, 2e = cache
		rows, _ := repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
		rows[0].Summary.Map.Labels["fr"] = "muté"
		rows[0].Summary.Map.DefaultLabel = "muté"
		rows[0].Summary.Map.IconURL = "muté"
		rows[0].Summary.Playlist.Labels["fr"] = "muté"
		rows[0].Summary.PairMode.Labels["fr"] = "muté"
		rows[0].Summary.MatchID = "muté"
		sort.Slice(rows, func(a, b int) bool { return rows[a].Summary.MatchID > rows[b].Summary.MatchID })
	}
	rows, _ := repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	m := rows[0].Summary
	if m.MatchID != "m1" || m.Map.Labels["fr"] != "Aquarius FR" || m.Map.DefaultLabel != "Aquarius" || m.Map.IconURL != "" ||
		m.Playlist.Labels["fr"] != "Classé" || m.PairMode.Labels["fr"] != "Assassin" {
		t.Errorf("cache modifié par un lecteur : %+v / map=%+v", m.MatchID, *m.Map)
	}
}

// TestCachedPlayerMatchesRepo_ConcurrentReEnrichment : des requêtes concurrentes qui
// ré-enrichissent leurs lignes (Home, Synthèse) écrivent chacune dans SES maps — un
// Labels partagé ferait « concurrent map writes » (erreur fatale du runtime).
func TestCachedPlayerMatchesRepo_ConcurrentReEnrichment(t *testing.T) {
	t.Parallel()
	repo := newTestPlayerMatches(&fakePlayerMatches{}, "x1")
	_, _ = repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, _ := repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
			for i := 0; i < 200; i++ {
				rows[0].Summary.Map.Labels["fr"] = "ré-enrichi"
				rows[1].Summary.Map.Labels["en"] = "re-enriched"
			}
		}()
	}
	wg.Wait()
}

func TestCachedPlayerMatchesRepo_ConcurrentMissesCoalesce(t *testing.T) {
	t.Parallel()
	src := &fakePlayerMatches{delay: 50 * time.Millisecond}
	repo := newTestPlayerMatches(src, "x1")
	var wg sync.WaitGroup
	for g := 0; g < 25; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
		}()
	}
	wg.Wait()
	if got := src.calls.Load(); got != 1 {
		t.Errorf("chargements = %d, want 1 (coalescence)", got)
	}
}

// TestCachedPlayerMatchesRepo_LobbySizesDelegates : la capacité optionnelle lue par
// SessionPageService (assertion de type sur ce jeu de méthodes) reste exposée.
func TestCachedPlayerMatchesRepo_LobbySizesDelegates(t *testing.T) {
	t.Parallel()
	src := &fakePlayerMatches{}
	var repo port.PlayerMatchesRepository = newTestPlayerMatches(src, "x1")
	provider, ok := repo.(interface {
		LobbySizesAtCompletion(ctx context.Context, slug string, matchIDs []string) (map[string]int, error)
	})
	if !ok {
		t.Fatal("CachedPlayerMatchesRepo n'expose plus LobbySizesAtCompletion : le breakdown de placements de la page Sessions disparaîtrait")
	}
	sizes, err := provider.LobbySizesAtCompletion(context.Background(), "halo_infinite", []string{"m1"})
	if err != nil || sizes["m1"] != 8 || src.lobbyCalls.Load() != 1 {
		t.Errorf("délégation : %v, err=%v, appels=%d", sizes, err, src.lobbyCalls.Load())
	}
}

func TestCachedPlayerMatchesRepo_InvalidatePlayer(t *testing.T) {
	t.Parallel()
	src := &fakePlayerMatches{}
	repo := newTestPlayerMatches(src, "x1")
	_, _ = repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	repo.InvalidatePlayer("halo_infinite", "GT")
	_, _ = repo.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	if got := src.calls.Load(); got != 2 {
		t.Errorf("chargements = %d, want 2 après InvalidatePlayer", got)
	}
}

// TestInvalidatePlayerReadCaches_PlayerMatches : la fonction appelée par le post-sync
// vide aussi le cache process-wide de l'historique, pour le seul joueur/titre visé.
func TestInvalidatePlayerReadCaches_PlayerMatches(t *testing.T) {
	t.Parallel()
	target, other := &fakePlayerMatches{}, &fakePlayerMatches{}
	repoTarget := &CachedPlayerMatchesRepo{inner: target, cache: playerMatchesReadCache,
		scope: testScope("l5b-test-pm-target", "halo_infinite", "p")}
	repoOther := &CachedPlayerMatchesRepo{inner: other, cache: playerMatchesReadCache,
		scope: testScope("l5b-test-pm-other", "halo_infinite", "p")}
	for _, r := range []*CachedPlayerMatchesRepo{repoTarget, repoOther, repoTarget, repoOther} {
		_, _ = r.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	}
	InvalidatePlayerReadCaches(context.Background(), "l5b-test-pm-target", "halo_infinite")
	_, _ = repoTarget.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	_, _ = repoOther.LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	if target.calls.Load() != 2 || other.calls.Load() != 1 {
		t.Errorf("chargements cible=%d autre=%d, want 2 et 1", target.calls.Load(), other.calls.Load())
	}
}

// TestClonePlayerMatchRows_ReferenceInventory fige l'inventaire des champs
// référence de canonical.PlayerMatchRow (pointeurs de struct, slices, maps) : chacun
// est soit COPIÉ par clonePlayerMatchRows, soit PARTAGÉ parce qu'aucun consommateur
// n'écrit à travers lui (grep du 2026-09-23). Un champ référence ajouté au type fait
// échouer ce test : décider copie ou partage, puis l'inscrire ici. Les pointeurs de
// scalaires (*int, *float64, *string, *bool, *int64) sont partagés en bloc.
func TestClonePlayerMatchRows_ReferenceInventory(t *testing.T) {
	want := map[string]string{
		"Summary.Playlist":                "copié",
		"Summary.Playlist.Labels":         "copié",
		"Summary.Map":                     "copié",
		"Summary.Map.Labels":              "copié",
		"Summary.GameVariant":             "copié",
		"Summary.GameVariant.Labels":      "copié",
		"Summary.PairMode":                "copié",
		"Summary.PairMode.Labels":         "copié",
		"Summary.Teams":                   "partagé",
		"Summary.Teams.ParticipantsXUIDs": "partagé",
		"Enrichment.FriendsXUIDs":         "partagé",
		"Enrichment.SkillSnapshot":        "partagé",
	}
	got := map[string]bool{}
	collectReferenceFields(reflect.TypeOf(canonical.PlayerMatchRow{}), "", got)
	for path := range got {
		if _, ok := want[path]; !ok {
			t.Errorf("champ référence non inventorié : %s — le copier dans clonePlayerMatchRows ou le déclarer partagé", path)
		}
	}
	for path := range want {
		if !got[path] {
			t.Errorf("champ inventorié disparu du type : %s — mettre l'inventaire à jour", path)
		}
	}

	// Les champs « copié » le sont réellement : aucune référence commune après clone.
	src, _ := (&fakePlayerMatches{}).LoadPlayerMatches(context.Background(), "", "", port.PlayerMatchFilters{})
	src[0].Summary.GameVariant = &canonical.AssetReference{Labels: map[string]string{"fr": "v"}}
	cl := clonePlayerMatchRows(src)
	for name, pair := range map[string][2]*canonical.AssetReference{
		"Playlist": {src[0].Summary.Playlist, cl[0].Summary.Playlist}, "Map": {src[0].Summary.Map, cl[0].Summary.Map},
		"GameVariant": {src[0].Summary.GameVariant, cl[0].Summary.GameVariant}, "PairMode": {src[0].Summary.PairMode, cl[0].Summary.PairMode},
	} {
		if pair[0] == pair[1] || reflect.ValueOf(pair[0].Labels).Pointer() == reflect.ValueOf(pair[1].Labels).Pointer() {
			t.Errorf("Summary.%s partagé après clone", name)
		}
	}
}

// collectReferenceFields parcourt les champs de t et note les chemins des champs
// référence : pointeurs de struct (parcourus), slices et maps (éléments struct
// parcourus). time.Time et les pointeurs de scalaires sont ignorés.
func collectReferenceFields(t reflect.Type, prefix string, out map[string]bool) {
	timeType := reflect.TypeOf(time.Time{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		path := prefix + f.Name
		ft := f.Type
		switch ft.Kind() {
		case reflect.Struct:
			if ft != timeType {
				collectReferenceFields(ft, path+".", out)
			}
		case reflect.Pointer:
			if ft.Elem().Kind() == reflect.Struct && ft.Elem() != timeType {
				out[path] = true
				collectReferenceFields(ft.Elem(), path+".", out)
			}
		case reflect.Slice, reflect.Map:
			out[path] = true
			if ft.Elem().Kind() == reflect.Struct && ft.Elem() != timeType {
				collectReferenceFields(ft.Elem(), path+".", out)
			}
		}
	}
}

func TestFiltersCacheKey_Stable(t *testing.T) {
	t.Parallel()
	a := port.PlayerMatchFilters{
		OutcomeIn:           []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss},
		ExcludeFriendsXUIDs: []string{"x1", "x2"},
		MapIDs:              []string{"m1", "m2"},
	}
	b := port.PlayerMatchFilters{
		OutcomeIn:           []canonical.Outcome{canonical.OutcomeLoss, canonical.OutcomeWin},
		ExcludeFriendsXUIDs: []string{"x2", "x1"},
		MapIDs:              []string{"m2", "m1"},
	}
	if filtersCacheKey(a) != filtersCacheKey(b) {
		t.Error("permuted slices should produce same cache key")
	}
}

func TestFiltersCacheKey_DistinctValues(t *testing.T) {
	t.Parallel()
	a := port.PlayerMatchFilters{Limit: 10}
	b := port.PlayerMatchFilters{Limit: 20}
	if filtersCacheKey(a) == filtersCacheKey(b) {
		t.Error("distinct values should produce distinct keys")
	}
}
