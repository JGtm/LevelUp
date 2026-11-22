package halo_5

// capture_known_run_test.go : le delta Halo 5 s'arrête après domain.DeltaKnownRun matchs connus
// d'affilée, pas au premier connu (D-23).

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"

	"github.com/JGtm/halo-api/halo5"
)

// listEntry est une entrée de la liste de l'API : un identifiant et son mode (1 = arène, 4 = Warzone).
type listEntry struct {
	id   string
	mode int
}

func arena(ids ...string) []listEntry {
	out := make([]listEntry, len(ids))
	for i, id := range ids {
		out[i] = listEntry{id: id, mode: 1}
	}
	return out
}

func ids(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%03d", prefix, i)
	}
	return out
}

// countingSource sert une liste paginée par page de 25 et note chaque décalage demandé.
type countingSource struct {
	*fakeH5Source
	starts []int
}

func (c *countingSource) GetPlayerMatches(ctx context.Context, gt string, start, count int) (*halo5.H5MatchesResponse, error) {
	c.starts = append(c.starts, start)
	return c.fakeH5Source.GetPlayerMatches(ctx, gt, start, count)
}

func newCountingSource(t *testing.T, list []listEntry) *countingSource {
	t.Helper()
	const pageSize = 25
	pages := map[int]*halo5.H5MatchesResponse{}
	for start := 0; start < len(list); start += pageSize {
		end := min(start+pageSize, len(list))
		parts := make([]string, 0, end-start)
		for _, e := range list[start:end] {
			parts = append(parts, matchJSONMode(e.id, e.mode))
		}
		raw := fmt.Sprintf(`{"Start":%d,"Count":%d,"ResultCount":%d,"Results":[%s]}`,
			start, len(parts), len(parts), strings.Join(parts, ","))
		var resp halo5.H5MatchesResponse
		if err := json.Unmarshal([]byte(raw), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		pages[start] = &resp
	}
	return &countingSource{fakeH5Source: &fakeH5Source{pages: pages}}
}

func collectedIDs(t *testing.T, src CaptureSource, known map[string]bool) ([]string, CaptureStats) {
	t.Helper()
	batches, stats, err := CollectRecentMatches(context.Background(), src, jgtmViewer(), idResolver,
		func(id string) bool { return known[id] }, CaptureOptions{PageSize: 25})
	if err != nil {
		t.Fatalf("CollectRecentMatches: %v", err)
	}
	out := make([]string, 0, len(batches))
	for _, b := range batches {
		out = append(out, b.Shared.Match.MatchID)
	}
	return out, stats
}

func knownSet(lists ...[]string) map[string]bool {
	k := map[string]bool{}
	for _, l := range lists {
		for _, id := range l {
			k[id] = true
		}
	}
	return k
}

func sameIDs(a, b []string) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

// Le joueur joue A seul, puis B avec un coéquipier ; le coéquipier range B en premier : la liste est
// [B, A, anciens connus], B est connu, A non. A est pris.
func TestCollectRecentMatches_ConnuRangeParUnCoequipier_LInconnuEnDessousEstPris(t *testing.T) {
	old := ids("old", 98)
	list := arena(append([]string{"B", "A"}, old...)...)
	src := newCountingSource(t, list)

	got, stats := collectedIDs(t, src, knownSet([]string{"B"}, old))
	if !sameIDs(got, []string{"A"}) {
		t.Errorf("collectés = %v, want [A]", got)
	}
	if !stats.StoppedOnKnown {
		t.Error("StoppedOnKnown = false : après A, 25 connus d'affilée arrêtent le delta")
	}
	if stats.TopSpan != 2 {
		t.Errorf("TopSpan = %d, want 2 (B et A sont arrivés)", stats.TopSpan)
	}
}

// 24 connus puis un inconnu : la suite est incomplète, l'inconnu est pris.
func TestCollectRecentMatches_VingtQuatreConnusPuisUnInconnu_LInconnuEstPris(t *testing.T) {
	known := ids("k", 24)
	rest := ids("r", 30)
	list := arena(append(append(append([]string{}, known...), "U"), rest...)...)
	got, _ := collectedIDs(t, newCountingSource(t, list), knownSet(known, rest))
	if !sameIDs(got, []string{"U"}) {
		t.Errorf("collectés = %v, want [U]", got)
	}
}

// 25 connus d'affilée : arrêt avant l'inconnu qui suit, une seule page lue.
func TestCollectRecentMatches_VingtCinqConnusDAffilee_Arret(t *testing.T) {
	if domain.DeltaKnownRun != 25 {
		t.Fatalf("DeltaKnownRun = %d, les scénarios supposent 25", domain.DeltaKnownRun)
	}
	known := ids("k", 25)
	rest := ids("r", 30)
	src := newCountingSource(t, arena(append(append(append([]string{}, known...), "U"), rest...)...))
	got, stats := collectedIDs(t, src, knownSet(known, rest))
	if len(got) != 0 || !stats.StoppedOnKnown {
		t.Errorf("collectés = %v, stoppedOnKnown = %v, want vide et vrai", got, stats.StoppedOnKnown)
	}
	if len(src.starts) != 1 {
		t.Errorf("pages lues %v, want 1", src.starts)
	}
}

// Une entrée écartée (Warzone) ne remet pas la suite à zéro ni ne l'allonge : 25 connus d'affilée de
// part et d'autre d'un Warzone arrêtent le delta, 24 non.
func TestCollectRecentMatches_EntreeEcarteeNeutre(t *testing.T) {
	build := func(after int) ([]listEntry, map[string]bool) {
		a, b := ids("a", 10), ids("b", after)
		list := append(arena(a...), listEntry{id: "wz", mode: 4})
		list = append(list, arena(b...)...)
		list = append(list, arena("U")...)
		list = append(list, arena(ids("z", 30)...)...)
		return list, knownSet(a, b, ids("z", 30))
	}
	list, known := build(15) // 10 + 15 = 25 connus autour du Warzone
	if got, _ := collectedIDs(t, newCountingSource(t, list), known); len(got) != 0 {
		t.Errorf("25 connus autour d'un Warzone : collectés = %v, want vide", got)
	}
	list, known = build(14) // 24 connus
	if got, _ := collectedIDs(t, newCountingSource(t, list), known); !sameIDs(got, []string{"U"}) {
		t.Errorf("24 connus autour d'un Warzone : collectés = %v, want [U]", got)
	}
}

// Appels d'API en historique complet : sans nouveau match une page, comme avant ; avec un nouveau match
// deux pages (avant : une), la seconde complète la suite de 25 connus.
func TestCollectRecentMatches_AppelsDAPI(t *testing.T) {
	all := ids("m", 100)

	src := newCountingSource(t, arena(all...))
	collectedIDs(t, src, knownSet(all))
	if len(src.starts) != 1 {
		t.Errorf("sans nouveau match : pages %v, want 1 (avant : 1)", src.starts)
	}

	src = newCountingSource(t, arena(all...))
	got, _ := collectedIDs(t, src, knownSet(all[1:]))
	if len(got) != 1 || len(src.starts) != 2 {
		t.Errorf("un nouveau match : collectés %v, pages %v, want 1 et 2 pages (avant : 1)", got, src.starts)
	}
}
