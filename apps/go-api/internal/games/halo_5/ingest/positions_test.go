package ingest

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func killWithLocs(killerGT, victimGT string, timeMs int, kloc, vloc *canonical.Vec3) canonical.MatchEvent {
	ev := kill(killerGT, victimGT, "", timeMs)
	ev.KillerLoc = kloc
	ev.VictimLoc = vloc
	return ev
}

func TestMapKillPositions(t *testing.T) {
	events := []canonical.MatchEvent{
		killWithLocs("Madina97294", "JGtm", 5000,
			&canonical.Vec3{X: 1, Y: 2, Z: 3}, &canonical.Vec3{X: 4, Y: 5, Z: 6}),
		killWithLocs("JGtm", "Madina97294", 8000,
			&canonical.Vec3{X: 7, Y: 8, Z: 9}, nil), // tueur seul, victime sans position
		kill("Madina97294", "JGtm", "", 9000), // aucune position → ignoré
		medal("Madina97294", "100", 1000),     // non-kill → ignoré
	}
	resolve := fakeResolver(map[string]string{"Madina97294": "xA", "JGtm": "xB"})

	rows := MapKillPositions("m1", events, resolve)

	if len(rows) != 2 {
		t.Fatalf("kill_positions: %d, attendu 2 — %+v", len(rows), rows)
	}
	r0 := rows[0]
	if r0.KillerXUID != "xA" || r0.TimeMS != 5000 {
		t.Errorf("ligne 0 identité/temps: %+v", r0)
	}
	if r0.KillerX == nil || *r0.KillerX != 1 || r0.VictimZ == nil || *r0.VictimZ != 6 {
		t.Errorf("positions tueur/victime mal mappées: %+v", r0)
	}
	if rows[1].VictimX != nil {
		t.Errorf("position victime attendue nil (absente): %+v", rows[1])
	}
	if rows[1].KillerX == nil || *rows[1].KillerX != 7 {
		t.Errorf("position tueur attendue présente: %+v", rows[1])
	}
}
