package analysis

import (
	"testing"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/canonical"
)

func TestFirstBlood(t *testing.T) {
	events := []ImpactEvent{
		{TimeMS: 500, EventType: highlightevent.EventTypeDeath, ActorXUID: "v"},
		{TimeMS: 3000, EventType: highlightevent.EventTypeKill, ActorXUID: "b"},
		{TimeMS: 1200, EventType: highlightevent.EventTypeKill, ActorXUID: "a"},
	}
	if fb := FirstBlood(events); fb == nil || fb.ActorXUID != "a" || fb.TimeMS != 1200 {
		t.Fatalf("premier frag attendu a@1200, obtenu %+v", fb)
	}
	if fb := FirstBlood(events[:1]); fb != nil {
		t.Fatalf("sans frag : nil attendu, obtenu %+v", fb)
	}
}

func TestFirstDeath(t *testing.T) {
	events := []ImpactEvent{
		{TimeMS: 400, EventType: highlightevent.EventTypeKill, ActorXUID: "k"},
		{TimeMS: 3000, EventType: highlightevent.EventTypeDeath, ActorXUID: "b"},
		{TimeMS: 1200, EventType: highlightevent.EventTypeDeath, ActorXUID: "a"},
	}
	if fd := FirstDeath(events); fd == nil || fd.ActorXUID != "a" || fd.TimeMS != 1200 {
		t.Fatalf("premiere mort attendue a@1200, obtenu %+v", fd)
	}
	if fd := FirstDeath(events[:1]); fd != nil {
		t.Fatalf("sans mort : nil attendu, obtenu %+v", fd)
	}
}

func hlKill(matchID, killer string, ms int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{
		MatchID: matchID, EventType: string(canonical.EventKill), TimeMS: ms, XUID: killer, KillerXUID: &killer,
	}
}

func hlDeath(matchID, victim string, ms int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{
		MatchID: matchID, EventType: string(canonical.EventDeath), TimeMS: ms, XUID: victim, VictimXUID: &victim,
	}
}

func TestCountFirstKillsAndDeaths(t *testing.T) {
	const me = "me"
	events := []canonical.HighlightEvent{
		// m1 : mon frag ouvre le match, la premiere mort est la mienne apres.
		hlKill("m1", me, 1000), hlKill("m1", "x", 2000), hlDeath("m1", "y", 1000), hlDeath("m1", me, 1500),
		// m2 : un adversaire ouvre le match et je meurs en premier.
		hlKill("m2", "x", 800), hlDeath("m2", me, 800), hlKill("m2", me, 5000),
		// m3 : un frag de compte a rebours (instant negatif) ne compte pas ; le premier vrai frag est le mien.
		hlKill("m3", "x", -500), hlDeath("m3", me, -500), hlKill("m3", me, 700), hlDeath("m3", "y", 700),
		// m4 : frag de bot (tueur vide) : personne n'est credite, la mort de bot non plus.
		hlKill("m4", "", 300), hlDeath("m4", "bot", 300), hlKill("m4", me, 900), hlDeath("m4", me, 900),
		// m6 : tueur absent de la source (bot sans xuid) : ouvre le match, personne n est credite.
		{MatchID: "m6", EventType: string(canonical.EventKill), TimeMS: 100}, hlKill("m6", me, 400),
		// m5 : un medaille n'est ni frag ni mort.
		{MatchID: "m5", EventType: string(canonical.EventMedal), TimeMS: 10, XUID: me, PlayerXUID: &[]string{me}[0]},
	}
	fk, fd := CountFirstKillsAndDeaths(events, me)
	if fk != 2 { // m1, m3
		t.Errorf("premiers frags = %d, attendu 2", fk)
	}
	if fd != 1 { // m2 (m1 : y meurt en premier ; m3 : y ; m4 : bot)
		t.Errorf("premieres morts = %d, attendu 1", fd)
	}
	if fk, fd := CountFirstKillsAndDeaths(nil, me); fk != 0 || fd != 0 {
		t.Errorf("sans evenement : 0/0 attendu, obtenu %d/%d", fk, fd)
	}
}
