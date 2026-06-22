package analysis

import (
	"reflect"
	"testing"
)

func TestPerfectKillMedalIDs_HaloInfinite(t *testing.T) {
	got := PerfectKillMedalIDs("halo_infinite")
	want := []int64{1512363953}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HINF: got %v, want %v", got, want)
	}
}

func TestPerfectKillMedalIDs_Halo5_AggregatesSixIDs(t *testing.T) {
	got := PerfectKillMedalIDs("halo_5")
	// 6 ids attendus, triés croissants. Perfection (3592822316) EXCLUE.
	want := []int64{370413844, 1080468863, 2279899989, 3098362934, 3653057799, 3992195104}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("h5: got %v, want %v", got, want)
	}
	if len(got) != 6 {
		t.Fatalf("h5: got %d ids, want 6", len(got))
	}
	for _, id := range got {
		if id == 3592822316 {
			t.Fatalf("h5: « Perfection » (3592822316) ne doit PAS être incluse")
		}
	}
}

func TestPerfectKillMedalIDs_UnknownSlugFallsBackToDefault(t *testing.T) {
	wantHINF := PerfectKillMedalIDs("halo_infinite")
	for _, slug := range []string{"", "   ", "halo_3_unknown", "title_x"} {
		got := PerfectKillMedalIDs(slug)
		if !reflect.DeepEqual(got, wantHINF) {
			t.Errorf("slug %q: got %v, want HINF default %v", slug, got, wantHINF)
		}
	}
}

func TestPerfectKillMedalIDs_ReturnedSliceIsCopy(t *testing.T) {
	a := PerfectKillMedalIDs("halo_5")
	a[0] = -1
	b := PerfectKillMedalIDs("halo_5")
	if b[0] == -1 {
		t.Fatal("mutation de la slice retournée a corrompu la source")
	}
}

func TestPerfectKillMedalInClause(t *testing.T) {
	if got, want := PerfectKillMedalInClause("medal_name_id", "halo_infinite"),
		"medal_name_id IN (1512363953)"; got != want {
		t.Errorf("HINF: got %q, want %q", got, want)
	}
	got := PerfectKillMedalInClause("me.medal_name_id", "halo_5")
	want := "me.medal_name_id IN (370413844, 1080468863, 2279899989, 3098362934, 3653057799, 3992195104)"
	if got != want {
		t.Errorf("h5: got %q, want %q", got, want)
	}
}
