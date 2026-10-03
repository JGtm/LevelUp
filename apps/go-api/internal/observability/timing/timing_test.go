package timing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock est une horloge pilotée par le test : les durées mesurées sont exactes, sans
// sommeil ni dépendance à la vitesse de la machine.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// timeSection ouvre `name`, avance l'horloge de `d`, puis ferme la section.
func timeSection(tm *Timings, clock *fakeClock, name string, d time.Duration) {
	stop := tm.Section(name)
	clock.advance(d)
	stop()
}

func newTestTimings() (*Timings, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)}
	return newTimings(clock.now), clock
}

// attrMap convertit la liste clé/valeur de LogAttrs en map (clés uniques attendues).
func attrMap(t *testing.T, attrs []any) map[string]any {
	t.Helper()
	if len(attrs)%2 != 0 {
		t.Fatalf("LogAttrs : nombre impair d'éléments (%d)", len(attrs))
	}
	out := make(map[string]any, len(attrs)/2)
	for i := 0; i < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok {
			t.Fatalf("LogAttrs : clé %v n'est pas une chaîne", attrs[i])
		}
		if _, dup := out[key]; dup {
			t.Fatalf("LogAttrs : clé %q en double", key)
		}
		out[key] = attrs[i+1]
	}
	return out
}

func TestNilTimingsIsNoop(t *testing.T) {
	var tm *Timings
	stop := tm.Section("x")
	stop() // ne doit pas paniquer
	if got := tm.Snapshot(); got != nil {
		t.Errorf("Snapshot sur nil = %v, attendu nil", got)
	}
	if got := tm.LogAttrs(); got != nil {
		t.Errorf("LogAttrs sur nil = %v, attendu nil", got)
	}
}

func TestFromContextWithoutTimingsIsNilAndSafe(t *testing.T) {
	if got := FromContext(context.Background()); got != nil {
		t.Fatalf("FromContext(Background) = %p, attendu nil", got)
	}
	var nilCtx context.Context
	if got := FromContext(nilCtx); got != nil {
		t.Fatalf("FromContext(nil) = %p, attendu nil", got)
	}
	// Le chemin réel d'un service hors requête HTTP : aucun effet, aucune panique.
	defer FromContext(context.Background()).Section("hors_requete")()
}

func TestWithTimingsRoundTrip(t *testing.T) {
	ctx, tm := WithTimings(context.Background())
	if tm == nil {
		t.Fatal("WithTimings a rendu un *Timings nil")
	}
	if got := FromContext(ctx); got != tm {
		t.Fatalf("FromContext = %p, attendu %p", got, tm)
	}
	func() {
		defer FromContext(ctx).Section("reelle")()
	}()
	snap := tm.Snapshot()
	if len(snap) != 1 || snap[0].Name != "reelle" || snap[0].Calls != 1 {
		t.Fatalf("Snapshot = %+v, attendu une section « reelle » appelée une fois", snap)
	}
}

func TestSectionAccumulatesDurationAndCalls(t *testing.T) {
	tm, clock := newTestTimings()
	timeSection(tm, clock, "echange", 30*time.Millisecond)
	timeSection(tm, clock, "echange", 12*time.Millisecond)
	snap := tm.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("Snapshot = %+v, attendu une seule section", snap)
	}
	if got := snap[0]; got.Name != "echange" || got.Duration != 42*time.Millisecond || got.Calls != 2 {
		t.Fatalf("section = %+v, attendu echange 42ms en 2 appels", got)
	}
}

func TestSnapshotSortedByDurationThenFirstAppearance(t *testing.T) {
	tm, clock := newTestTimings()
	timeSection(tm, clock, "courte", 5*time.Millisecond)
	timeSection(tm, clock, "egale_a", 20*time.Millisecond)
	timeSection(tm, clock, "longue", 90*time.Millisecond)
	timeSection(tm, clock, "egale_b", 20*time.Millisecond)
	var names []string
	for _, s := range tm.Snapshot() {
		names = append(names, s.Name)
	}
	want := "longue egale_a egale_b courte"
	if got := strings.Join(names, " "); got != want {
		t.Fatalf("ordre = %q, attendu %q", got, want)
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	tm, clock := newTestTimings()
	timeSection(tm, clock, "a", time.Millisecond)
	snap := tm.Snapshot()
	snap[0].Calls = 99
	if got := tm.Snapshot()[0].Calls; got != 1 {
		t.Fatalf("Calls après mutation de la copie = %d, attendu 1", got)
	}
}

func TestLogAttrsEmptyWhenNoSection(t *testing.T) {
	tm, _ := newTestTimings()
	if got := tm.LogAttrs(); got != nil {
		t.Fatalf("LogAttrs sans section = %v, attendu nil", got)
	}
}

func TestLogAttrsFormat(t *testing.T) {
	tm, clock := newTestTimings()
	timeSection(tm, clock, "impact_matrix", 3100*time.Millisecond)
	timeSection(tm, clock, "top_teammates", 3000*time.Millisecond)
	timeSection(tm, clock, "impact_matrix", 1500*time.Millisecond)
	got := attrMap(t, tm.LogAttrs())
	if got["total_ms"] != int64(7600) {
		t.Errorf("total_ms = %v (%T), attendu int64 7600", got["total_ms"], got["total_ms"])
	}
	if got["sections"] != "impact_matrix=4600 top_teammates=3000" {
		t.Errorf("sections = %q", got["sections"])
	}
	if got["calls"] != "impact_matrix=2" {
		t.Errorf("calls = %q, attendu \"impact_matrix=2\"", got["calls"])
	}
}

func TestLogAttrsOmitsCallsWhenEverySectionRanOnce(t *testing.T) {
	tm, clock := newTestTimings()
	timeSection(tm, clock, "a", 2*time.Millisecond)
	timeSection(tm, clock, "b", time.Millisecond)
	if _, ok := attrMap(t, tm.LogAttrs())["calls"]; ok {
		t.Fatal("calls présent alors qu'aucune section n'a été appelée deux fois")
	}
}

// Au-delà de maxLoggedSections, seules les plus longues sont listées, mais total_ms
// couvre toutes les sections, et calls ne cite que des sections listées.
func TestLogAttrsCapsListedSectionsButTotalCoversAll(t *testing.T) {
	tm, clock := newTestTimings()
	var total time.Duration
	for i := 1; i <= maxLoggedSections+3; i++ {
		d := time.Duration(i) * 10 * time.Millisecond
		timeSection(tm, clock, fmt.Sprintf("s%02d", i), d)
		total += d
	}
	// La plus courte (hors liste) est appelée deux fois : elle ne doit pas figurer dans calls.
	timeSection(tm, clock, "s01", time.Millisecond)
	total += time.Millisecond
	got := attrMap(t, tm.LogAttrs())
	if got["total_ms"] != total.Milliseconds() {
		t.Errorf("total_ms = %v, attendu %d", got["total_ms"], total.Milliseconds())
	}
	listed := strings.Fields(got["sections"].(string))
	if len(listed) != maxLoggedSections {
		t.Fatalf("%d sections listées, attendu %d : %v", len(listed), maxLoggedSections, listed)
	}
	if listed[0] != fmt.Sprintf("s%02d=%d", maxLoggedSections+3, (maxLoggedSections+3)*10) {
		t.Errorf("première section = %q, attendu la plus longue", listed[0])
	}
	if strings.Contains(got["sections"].(string), "s01=") {
		t.Error("s01 (la plus courte) ne devrait pas être listée")
	}
	if _, ok := got["calls"]; ok {
		t.Errorf("calls = %q : une section hors liste ne doit pas y figurer", got["calls"])
	}
}

func TestSectionsAreSafeConcurrently(t *testing.T) {
	_, tm := WithTimings(context.Background())
	const workers, perWorker = 8, 50
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				tm.Section("parallele")()
				_ = tm.LogAttrs()
			}
		}()
	}
	wg.Wait()
	snap := tm.Snapshot()
	if len(snap) != 1 || snap[0].Calls != workers*perWorker {
		t.Fatalf("Snapshot = %+v, attendu %d appels", snap, workers*perWorker)
	}
}
