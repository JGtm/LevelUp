package ingest

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func medal(gamertag, refID string, timeMs int) canonical.MatchEvent {
	r := refID
	return canonical.MatchEvent{
		Type:   canonical.MatchEventMedal,
		TimeMs: timeMs,
		Player: &canonical.PlayerIdentity{Gamertag: gamertag},
		RefID:  &r,
	}
}

// resolver factice : map gamertag → xuid ; "" si absent (non résolu).
func fakeResolver(m map[string]string) func(string) string {
	return func(gt string) string { return m[gt] }
}

func TestMapMedalEvents_AggregateAndTimeline(t *testing.T) {
	events := []canonical.MatchEvent{
		medal("Madina97294", "100", 5000),
		medal("Madina97294", "100", 9000), // même joueur + même médaille → count=2
		medal("Madina97294", "200", 12000),
		medal("JGtm", "100", 3000),
		{Type: canonical.MatchEventKill, TimeMs: 1000}, // non-médaille → ignoré
	}
	resolve := fakeResolver(map[string]string{"Madina97294": "2533274858283686", "JGtm": "2535400000000000"})

	agg, tl := MapMedalEvents("match-1", events, resolve)

	// Agrégat : 3 lignes distinctes (Madina/100=2, Madina/200=1, JGtm/100=1).
	if len(agg) != 3 {
		t.Fatalf("agrégat: %d lignes, attendu 3 — %+v", len(agg), agg)
	}
	if agg[0].XUID != "2533274858283686" || agg[0].MedalNameID != 100 || agg[0].Count != 2 {
		t.Errorf("1re ligne médaille: %+v, attendu Madina/100/count=2", agg[0])
	}
	if agg[1].MedalNameID != 200 || agg[1].Count != 1 {
		t.Errorf("2e ligne: %+v, attendu medal 200 count 1", agg[1])
	}

	// Timeline : 4 events médaille horodatés (un par médaille gagnée).
	if len(tl) != 4 {
		t.Fatalf("timeline: %d events, attendu 4", len(tl))
	}
	for _, e := range tl {
		if e.MatchID != "match-1" || e.EventType != string(canonical.MatchEventMedal) {
			t.Errorf("event mal formé: %+v", e)
		}
		if e.XUID == nil || *e.XUID == "" {
			t.Errorf("xuid résolu attendu non vide: %+v", e)
		}
		if e.DetailsJSON == nil {
			t.Errorf("type_hint (medal id) attendu: %+v", e)
		}
	}
	if tl[0].TimeMS != 5000 {
		t.Errorf("1er event time_ms = %d, attendu 5000", tl[0].TimeMS)
	}
}

func TestMapMedalEvents_UnresolvedXUIDStillPersisted(t *testing.T) {
	events := []canonical.MatchEvent{medal("UnknownPlayer", "100", 5000)}
	resolve := fakeResolver(nil) // résout rien

	agg, tl := MapMedalEvents("m", events, resolve)

	if len(agg) != 1 || agg[0].XUID != "" {
		t.Fatalf("agrégat avec xuid vide attendu: %+v", agg)
	}
	if len(tl) != 1 || tl[0].XUID != nil {
		t.Fatalf("timeline avec xuid NULL attendu: %+v", tl)
	}
}

func TestMapMedalEvents_NonNumericRefIgnored(t *testing.T) {
	events := []canonical.MatchEvent{medal("Madina97294", "not-a-number", 5000)}
	agg, tl := MapMedalEvents("m", events, fakeResolver(map[string]string{"Madina97294": "x"}))
	if len(agg) != 0 || len(tl) != 0 {
		t.Fatalf("médaille à id non numérique doit être ignorée: agg=%+v tl=%+v", agg, tl)
	}
}
