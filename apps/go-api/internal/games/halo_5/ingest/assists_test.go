package ingest

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// killWithAssists construit un event kill canonique avec des assistants (gamertags).
func killWithAssists(killerGT, victimGT string, timeMs int, assists ...string) canonical.MatchEvent {
	ev := kill(killerGT, victimGT, "", timeMs)
	for _, a := range assists {
		ev.Assists = append(ev.Assists, canonical.PlayerIdentity{Gamertag: a})
	}
	return ev
}

func TestMapAssistEvents(t *testing.T) {
	events := []canonical.MatchEvent{
		killWithAssists("JGtm", "Enemy", 1000, "Mate", "Mate2"), // 2 assists
		killWithAssists("JGtm", "Enemy", 2000),                  // kill sans assist → 0 ligne
		killWithAssists("Mate", "Enemy", 3000, ""),              // assistant gamertag vide → ignoré
		medal("JGtm", "100", 4000),                              // non-kill → ignoré
	}
	resolve := fakeResolver(map[string]string{"Mate": "xMate", "Mate2": "xMate2"})

	rows := MapAssistEvents("m1", events, resolve)

	if len(rows) != 2 {
		t.Fatalf("rows = %d, attendu 2 (les 2 assists du 1er kill) — %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r.MatchID != "m1" || r.EventType != "assist" || r.TimeMS != 1000 {
			t.Errorf("ligne assist mal formée: %+v", r)
		}
		if r.XUID == nil {
			t.Errorf("xuid assistant attendu non-nil: %+v", r)
		}
	}
}
