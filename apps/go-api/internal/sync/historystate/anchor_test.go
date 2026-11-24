package historystate_test

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"
)

func idsFrom(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%03d", prefix, i)
	}
	return out
}

// L'ancre se range et se relit avec le marque-page ; un écrasement sans ancre l'efface.
func TestCursor_AncreAllerRetour(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	want := historystate.Cursor{Offset: 300, LastMatchID: "m-300", HeadID: "m-000"}
	if !historystate.SaveCursor(ctx, db, want) {
		t.Fatal("SaveCursor")
	}
	got, ok := historystate.LoadCursor(ctx, db)
	if !ok || got != want || !got.HasAnchor() {
		t.Fatalf("LoadCursor = (%+v, %v), attendu (%+v, true) avec ancre", got, ok, want)
	}
	// Les échecs rangés à côté ne touchent pas l'ancre.
	historystate.SaveFailures(ctx, db, []historystate.Failure{{MatchID: "x", Attempts: 1}})
	if got, _ := historystate.LoadCursor(ctx, db); got != want {
		t.Errorf("après SaveFailures : %+v, attendu %+v", got, want)
	}
	// Une écriture sans ancre l'efface.
	historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 400, LastMatchID: "m-400"})
	if got, _ := historystate.LoadCursor(ctx, db); got.HasAnchor() || got.Offset != 400 {
		t.Errorf("écriture sans ancre : %+v, attendu 400 sans ancre", got)
	}
}

// Format compatible : un marque-page rangé avant l'ancre se relit, sans ancre.
func TestCursor_AncienFormatSansAncre(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	if err := syncmeta.Set(ctx, db, historystate.CursorKey,
		`{"offset":250,"last_match_id":"m-250","saved_at":"2026-11-22T10:00:00Z"}`); err != nil {
		t.Fatal(err)
	}
	got, ok := historystate.LoadCursor(ctx, db)
	if !ok || got.Offset != 250 || got.LastMatchID != "m-250" || got.HasAnchor() {
		t.Fatalf("LoadCursor = (%+v, %v), attendu 250/m-250 sans ancre", got, ok)
	}
	if n, found := got.ArrivedSince(idsFrom("m", 50)); found || n != 0 {
		t.Errorf("sans ancre : ArrivedSince = (%d, %v), attendu (0, false)", n, found)
	}
}

// ArrivedSince : position de l'ancienne tête dans le haut de la liste, toutes entrées comptées.
func TestArrivedSince(t *testing.T) {
	top := idsFrom("n", 40)
	cases := []struct {
		name  string
		head  string
		wantN int
		found bool
	}{
		{"aucune arrivée", "n000", 0, true},
		{"trois arrivées", "n003", 3, true},
		{"trente arrivées", "n030", 30, true},
		{"dernière entrée lue", "n039", 39, true},
		{"ancre perdue", "absente", 0, false},
		{"sans ancre", "", 0, false},
	}
	for _, c := range cases {
		cur := historystate.Cursor{Offset: 100, LastMatchID: "x", HeadID: c.head}
		n, found := cur.ArrivedSince(top)
		if n != c.wantN || found != c.found {
			t.Errorf("%s : (%d, %v), attendu (%d, %v)", c.name, n, found, c.wantN, c.found)
		}
	}
	if n, found := (historystate.Cursor{Offset: 1, LastMatchID: "x", HeadID: "a"}).ArrivedSince(nil); found || n != 0 {
		t.Errorf("liste vide : (%d, %v)", n, found)
	}
}

func TestResumeOffset_EtBorne(t *testing.T) {
	c := historystate.Cursor{Offset: 200, LastMatchID: "x", HeadID: "h"}
	if got := c.ResumeOffset(30); got != 230 {
		t.Errorf("ResumeOffset(30) = %d, attendu 230", got)
	}
	if got := c.ResumeOffset(-4); got != 200 {
		t.Errorf("ResumeOffset(-4) = %d, attendu 200", got)
	}
	if got := historystate.AnchorScanLimit(25); got != historystate.AnchorMaxPages*25 {
		t.Errorf("AnchorScanLimit(25) = %d", got)
	}
}
