package analysis

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func TestSynthesizeKillEventsFromKVPairs_KillAndDeathPerPair(t *testing.T) {
	pairs := []KVSyntheticInput{
		{KillerXUID: "K1", VictimXUID: "V1", TimeMS: 5000},
		{KillerXUID: "K2", VictimXUID: "V2", TimeMS: 1000},
	}
	out := SynthesizeKillEventsFromKVPairs(pairs, "m1")
	if len(out) != 4 {
		t.Fatalf("attendu 4 events (2 paires × kill+death), obtenu %d", len(out))
	}
	// Tri par TimeMS croissant : la paire à 1000 doit venir avant celle à 5000.
	if out[0].TimeMS != 1000 || out[len(out)-1].TimeMS != 5000 {
		t.Errorf("events non triés par TimeMS : %d..%d", out[0].TimeMS, out[len(out)-1].TimeMS)
	}
	var kills, deaths int
	for _, e := range out {
		if e.MatchID != "m1" {
			t.Errorf("MatchID non rattaché : %q", e.MatchID)
		}
		switch canonical.HighlightEventType(e.EventType) {
		case canonical.EventKill:
			kills++
			if e.XUID == "" || e.KillerXUID == nil || *e.KillerXUID != e.XUID {
				t.Errorf("kill : XUID/KillerXUID incohérents (%q / %v)", e.XUID, e.KillerXUID)
			}
		case canonical.EventDeath:
			deaths++
			if e.XUID == "" || e.VictimXUID == nil || *e.VictimXUID != e.XUID {
				t.Errorf("death : XUID/VictimXUID incohérents (%q / %v)", e.XUID, e.VictimXUID)
			}
		default:
			t.Errorf("type inattendu : %q", e.EventType)
		}
	}
	if kills != 2 || deaths != 2 {
		t.Errorf("attendu 2 kills + 2 deaths, obtenu %d / %d", kills, deaths)
	}
}

func TestSynthesizeKillEventsFromKVPairs_KillCountMultiplies(t *testing.T) {
	pairs := []KVSyntheticInput{{KillerXUID: "K", VictimXUID: "V", TimeMS: 100, KillCount: 3}}
	out := SynthesizeKillEventsFromKVPairs(pairs, "m1")
	if len(out) != 6 { // 3 kills + 3 deaths
		t.Fatalf("KillCount=3 doit produire 6 events, obtenu %d", len(out))
	}
}

func TestSynthesizeKillEventsFromKVPairs_SkipsEmptyAndNil(t *testing.T) {
	if out := SynthesizeKillEventsFromKVPairs(nil, "m1"); out != nil {
		t.Errorf("nil input doit retourner nil, obtenu %v", out)
	}
	pairs := []KVSyntheticInput{{KillerXUID: "", VictimXUID: "V", TimeMS: 1}}
	if out := SynthesizeKillEventsFromKVPairs(pairs, "m1"); out != nil {
		t.Errorf("paire sans killer doit être ignorée, obtenu %v", out)
	}
}

func TestHasCanonicalKillOrDeath(t *testing.T) {
	medalsOnly := []canonical.HighlightEvent{{EventType: string(canonical.EventMedal)}}
	if HasCanonicalKillOrDeath(medalsOnly) {
		t.Error("médailles seules : ne doit PAS détecter kill/death")
	}
	withKill := []canonical.HighlightEvent{
		{EventType: string(canonical.EventMedal)},
		{EventType: string(canonical.EventKill)},
	}
	if !HasCanonicalKillOrDeath(withKill) {
		t.Error("présence d'un kill : doit détecter kill/death")
	}
}

func TestMergeAndSortCanonicalEvents(t *testing.T) {
	a := []canonical.HighlightEvent{{EventType: "medal", TimeMS: 3000}}
	b := []canonical.HighlightEvent{
		{EventType: "kill", TimeMS: 1000},
		{EventType: "death", TimeMS: 2000},
	}
	out := MergeAndSortCanonicalEvents(a, b)
	if len(out) != 3 {
		t.Fatalf("attendu 3 events fusionnés, obtenu %d", len(out))
	}
	if out[0].TimeMS != 1000 || out[1].TimeMS != 2000 || out[2].TimeMS != 3000 {
		t.Errorf("fusion non triée par TimeMS : %d, %d, %d", out[0].TimeMS, out[1].TimeMS, out[2].TimeMS)
	}
}
