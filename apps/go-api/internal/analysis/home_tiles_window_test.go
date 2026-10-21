package analysis_test

import (
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
)

// tileRow construit une ligne canonique minimale : id et instant.
func tileRow(id string, at time.Time) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{Summary: canonical.MatchSummary{MatchID: id, StartedAtUTC: at}}
}

// tileHistory : n matchs, un par minute, du plus récent au plus ancien (m000 le plus récent).
func tileHistory(n int) []canonical.PlayerMatchRow {
	base := time.Date(2026, 10, 20, 21, 0, 0, 0, time.UTC)
	rows := make([]canonical.PlayerMatchRow, n)
	for i := range rows {
		rows[i] = tileRow(fmt.Sprintf("m%03d", i), base.Add(-time.Duration(i)*time.Minute))
	}
	return rows
}

func tileIDs(rows []canonical.PlayerMatchRow) []string {
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].Summary.MatchID
	}
	return out
}

// walkTilePages lit toute la liste page par page, comme le fait le client (un curseur rendu, un
// curseur repassé) et rend les identifiants dans l'ordre servi.
func walkTilePages(t *testing.T, ordered []canonical.PlayerMatchRow, pageSize int) []string {
	t.Helper()
	var got []string
	cursor := analysis.HomeTileCursor{}
	for pages := 0; ; pages++ {
		if pages > len(ordered)+2 {
			t.Fatalf("pagination sans fin après %d pages", pages)
		}
		w := analysis.WindowHomeTileRows(ordered, cursor, pageSize)
		got = append(got, tileIDs(w.Rows)...)
		if !w.HasMore {
			if w.NextBefore == "" && len(w.Rows) > 0 {
				t.Fatalf("dernière page sans curseur calculé")
			}
			return got
		}
		var ok bool
		cursor, ok = analysis.ParseHomeTileCursor(w.NextBefore)
		if !ok {
			t.Fatalf("curseur rendu illisible : %q", w.NextBefore)
		}
	}
}

func TestWindowHomeTileRows_LesPagesRecollentLaListe(t *testing.T) {
	ordered := analysis.OrderHomeTileRows(tileHistory(200))
	want := tileIDs(ordered)
	for _, size := range []int{1, 7, 30, 60, 199, 200, 500} {
		got := walkTilePages(t, ordered, size)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("pages de %d : la concaténation diffère de la liste (%d tuiles contre %d)", size, len(got), len(want))
		}
	}
}

func TestWindowHomeTileRows_BornesEtDerniereTuile(t *testing.T) {
	ordered := analysis.OrderHomeTileRows(tileHistory(65))
	w := analysis.WindowHomeTileRows(ordered, analysis.HomeTileCursor{}, 30)
	if len(w.Rows) != 30 || !w.HasMore || w.NextBefore == "" {
		t.Fatalf("tête : %d tuiles, has_more=%v, curseur %q", len(w.Rows), w.HasMore, w.NextBefore)
	}
	cur, _ := analysis.ParseHomeTileCursor(w.NextBefore)
	w2 := analysis.WindowHomeTileRows(ordered, cur, 30)
	cur2, _ := analysis.ParseHomeTileCursor(w2.NextBefore)
	w3 := analysis.WindowHomeTileRows(ordered, cur2, 30)
	if len(w3.Rows) != 5 || w3.HasMore {
		t.Fatalf("dernière page : %d tuiles, has_more=%v (attendu 5, false)", len(w3.Rows), w3.HasMore)
	}
	// Exactement 30 tuiles restantes : pas de page suivante vide annoncée.
	exact := analysis.OrderHomeTileRows(tileHistory(60))
	last := analysis.WindowHomeTileRows(exact, cur, 30)
	if len(last.Rows) != 30 || last.HasMore {
		t.Fatalf("liste de 60 : seconde page %d tuiles, has_more=%v (attendu 30, false)", len(last.Rows), last.HasMore)
	}
	if empty := analysis.WindowHomeTileRows(nil, analysis.HomeTileCursor{}, 30); len(empty.Rows) != 0 || empty.HasMore || empty.NextBefore != "" {
		t.Fatalf("liste vide : %+v", empty)
	}
}

// Deux matchs au même instant : l'ordre est total (match_id décroissant) et aucune page ne perd
// ni ne répète l'un des deux, même quand la frontière de page passe entre eux.
func TestWindowHomeTileRows_InstantsEgaux(t *testing.T) {
	at := time.Date(2026, 10, 20, 21, 0, 0, 123456000, time.UTC)
	rows := []canonical.PlayerMatchRow{
		tileRow("a", at), tileRow("c", at), tileRow("b", at), tileRow("z", at.Add(-time.Hour)), tileRow("y", at.Add(time.Hour)),
	}
	ordered := analysis.OrderHomeTileRows(rows)
	if got, want := fmt.Sprint(tileIDs(ordered)), "[y c b a z]"; got != want {
		t.Fatalf("ordre = %s, attendu %s", got, want)
	}
	for _, size := range []int{1, 2, 3} {
		if got := fmt.Sprint(walkTilePages(t, ordered, size)); got != "[y c b a z]" {
			t.Errorf("pages de %d : %s", size, got)
		}
	}
}

// Le curseur désigne une tuile, pas une position : des matchs arrivés en tête entre deux pages
// (synchro) ne décalent rien, et un match retiré au milieu ne fait rien perdre.
func TestWindowHomeTileRows_SynchroEntreDeuxPages(t *testing.T) {
	before := analysis.OrderHomeTileRows(tileHistory(100))
	first := analysis.WindowHomeTileRows(before, analysis.HomeTileCursor{}, 30)
	cursor, _ := analysis.ParseHomeTileCursor(first.NextBefore)

	// Trois matchs plus récents arrivent ; m050 (déjà après le curseur) disparaît.
	grown := tileHistory(100)
	newer := time.Date(2026, 10, 20, 22, 0, 0, 0, time.UTC)
	grown = append(grown, tileRow("n1", newer), tileRow("n2", newer.Add(time.Minute)), tileRow("n3", newer.Add(2*time.Minute)))
	filtered := grown[:0:0]
	for _, r := range grown {
		if r.Summary.MatchID != "m050" {
			filtered = append(filtered, r)
		}
	}
	after := analysis.OrderHomeTileRows(filtered)
	second := analysis.WindowHomeTileRows(after, cursor, 30)

	seen := map[string]bool{}
	for _, id := range tileIDs(first.Rows) {
		seen[id] = true
	}
	for _, id := range tileIDs(second.Rows) {
		if seen[id] {
			t.Errorf("doublon %s entre la tête et la page suivante", id)
		}
	}
	if got := second.Rows[0].Summary.MatchID; got != "m030" {
		t.Errorf("la page suivante doit reprendre à m030, elle commence à %s", got)
	}
	if got := second.Rows[20].Summary.MatchID; got != "m051" {
		t.Errorf("m050 retiré : la tuile 20 de la page suivante doit être m051, obtenu %s", got)
	}
}

func TestParseHomeTileCursor(t *testing.T) {
	if c, ok := analysis.ParseHomeTileCursor(""); !ok || !c.IsZero() {
		t.Errorf("curseur vide = début de liste, obtenu (%+v, %v)", c, ok)
	}
	at := time.Date(2026, 10, 20, 21, 3, 4, 123456789, time.UTC)
	enc := analysis.HomeTileCursor{At: at, MatchID: "id_avec_underscore"}.Encode()
	c, ok := analysis.ParseHomeTileCursor(enc)
	if !ok || !c.At.Equal(at) || c.MatchID != "id_avec_underscore" {
		t.Errorf("aller-retour : (%+v, %v) pour %q", c, ok, enc)
	}
	for _, bad := range []string{"n'importe quoi", "2026-10-20T21:00:00Z", "2026-10-20T21:00:00Z_", "hier_m1", "_m1"} {
		if _, ok := analysis.ParseHomeTileCursor(bad); ok {
			t.Errorf("curseur %q accepté", bad)
		}
	}
}

func TestOrderHomeTileRows_NeReordonnePasLaTrancheDeLAppelant(t *testing.T) {
	base := time.Date(2026, 10, 20, 21, 0, 0, 0, time.UTC)
	rows := []canonical.PlayerMatchRow{tileRow("old", base.Add(-time.Hour)), tileRow("", base), tileRow("new", base)}
	got := analysis.OrderHomeTileRows(rows)
	if fmt.Sprint(tileIDs(got)) != "[new old]" {
		t.Errorf("ordre = %v, attendu [new old] (ligne sans match_id écartée)", tileIDs(got))
	}
	if rows[0].Summary.MatchID != "old" || rows[1].Summary.MatchID != "" || rows[2].Summary.MatchID != "new" {
		t.Errorf("la tranche de l'appelant a été réordonnée : %v", tileIDs(rows))
	}
	sorted := tileHistory(3)
	if out := analysis.OrderHomeTileRows(sorted); &out[0] != &sorted[0] {
		t.Error("une liste déjà ordonnée doit être rendue telle quelle, sans copie")
	}
}

func TestFavoriteHomeTileRows(t *testing.T) {
	ordered := analysis.OrderHomeTileRows(tileHistory(10))
	got := analysis.FavoriteHomeTileRows(ordered, map[string]bool{"m007": true, "m002": true, "m004": false, "absent": true})
	if fmt.Sprint(tileIDs(got)) != "[m002 m007]" {
		t.Errorf("favoris = %v, attendu [m002 m007] dans l'ordre d'affichage", tileIDs(got))
	}
	if analysis.FavoriteHomeTileRows(ordered, nil) != nil {
		t.Error("sans favori : nil attendu")
	}
}
