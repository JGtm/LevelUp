package narrative

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func ev(matchID, eventType string, killer, victim *string, timeMS int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{
		MatchID:    matchID,
		EventType:  eventType,
		TimeMS:     timeMS,
		KillerXUID: killer,
		VictimXUID: victim,
	}
}

func sptr(s string) *string { return &s }

func TestComputeFirstEventsPerMatch_Empty(t *testing.T) {
	t.Parallel()
	got := ComputeFirstEventsPerMatch(nil, "p1", nil)
	if len(got) != 0 {
		t.Errorf("expected empty/nil, got %+v", got)
	}
}

func TestComputeFirstEventsPerMatch_KillOnly(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, 5000),
		ev("m1", "kill", &p1, &other, 2000), // earlier
		ev("m1", "kill", &p1, &other, 9000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	if got[0].FirstKillMS == nil || *got[0].FirstKillMS != 2000 {
		t.Errorf("firstKill: %v", got[0].FirstKillMS)
	}
	if got[0].FirstDeathMS != nil {
		t.Errorf("firstDeath should be nil, got %v", *got[0].FirstDeathMS)
	}
}

func TestComputeFirstEventsPerMatch_DeathOnly(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	killer := "killer"
	events := []canonical.HighlightEvent{
		ev("m1", "death", &killer, &p1, 3000),
		ev("m1", "death", &killer, &p1, 1500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].FirstDeathMS == nil || *got[0].FirstDeathMS != 1500 {
		t.Errorf("firstDeath: %v", got[0].FirstDeathMS)
	}
}

func TestComputeFirstEventsPerMatch_KillAndDeath(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, 4000),
		ev("m1", "death", &other, &p1, 2500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if *got[0].FirstKillMS != 4000 {
		t.Errorf("firstKill: %d", *got[0].FirstKillMS)
	}
	if *got[0].FirstDeathMS != 2500 {
		t.Errorf("firstDeath: %d", *got[0].FirstDeathMS)
	}
}

func TestComputeFirstEventsPerMatch_OtherPlayerIgnored(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", sptr("someone_else"), &other, 1000),
		ev("m1", "death", sptr("someone_else"), sptr("another"), 2000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 (events present even without our player), got %d", len(got))
	}
	if got[0].FirstKillMS != nil || got[0].FirstDeathMS != nil {
		t.Error("firstKill/firstDeath should be nil for other players' events")
	}
}

func TestComputeFirstEventsPerMatch_FirstKillFirstDeathTypes(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "first_kill", &p1, &other, 1500),
		ev("m1", "first_death", &other, &p1, 1500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if *got[0].FirstKillMS != 1500 || *got[0].FirstDeathMS != 1500 {
		t.Errorf("first_kill/first_death types should map: %+v", got[0])
	}
}

func TestComputeFirstEventsPerMatch_MultiMatchSorted(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m_b", "kill", &p1, &other, 2000),
		ev("m_a", "kill", &p1, &other, 3000),
		ev("m_c", "kill", &p1, &other, 1000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if got[0].MatchID != "m_a" || got[1].MatchID != "m_b" || got[2].MatchID != "m_c" {
		t.Errorf("match order: %s / %s / %s", got[0].MatchID, got[1].MatchID, got[2].MatchID)
	}
}

func TestComputeFirstEventsPerMatch_WithMatchIDsListIncludesEmpty(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, 2000),
		// pas d'event sur m2
	}
	got := ComputeFirstEventsPerMatch(events, p1, []string{"m1", "m2", "m3"})
	if len(got) != 3 {
		t.Fatalf("matchIDs forces 1 row per ID, got %d", len(got))
	}
	if got[0].MatchID != "m1" || got[0].FirstKillMS == nil {
		t.Errorf("m1: %+v", got[0])
	}
	if got[1].MatchID != "m2" || got[1].FirstKillMS != nil || got[1].FirstDeathMS != nil {
		t.Errorf("m2 should be empty placeholder, got %+v", got[1])
	}
	if got[2].MatchID != "m3" || got[2].FirstKillMS != nil {
		t.Errorf("m3 should be empty placeholder, got %+v", got[2])
	}
}

func TestComputeFirstEventsPerMatch_IgnoresEmptyMatchID(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("", "kill", &p1, &other, 1000), // ignored
		ev("m1", "kill", &p1, &other, 2000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 || got[0].MatchID != "m1" {
		t.Errorf("empty matchID should be skipped, got %+v", got)
	}
}

// TestComputeFirstEventsPerMatch_SkipsPreT0Events : apres correction T0, un
// event au TimeMS negatif (countdown pre-gameplay) ne doit jamais etre retenu
// comme premier frag/mort. Le minimum doit porter sur les events de gameplay.
func TestComputeFirstEventsPerMatch_SkipsPreT0Events(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, -3000),  // pre-T0 (countdown) — ignore
		ev("m1", "kill", &p1, &other, 8000),   // premier frag gameplay
		ev("m1", "death", &other, &p1, -1500), // pre-T0 — ignore
		ev("m1", "death", &other, &p1, 12000), // premiere mort gameplay
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	if got[0].FirstKillMS == nil || *got[0].FirstKillMS != 8000 {
		t.Errorf("firstKill should skip pre-T0 event, got %v", got[0].FirstKillMS)
	}
	if got[0].FirstDeathMS == nil || *got[0].FirstDeathMS != 12000 {
		t.Errorf("firstDeath should skip pre-T0 event, got %v", got[0].FirstDeathMS)
	}
}

// TestComputeFirstEventsPerMatch_AllPreT0 : si tous les events du joueur sont
// pre-T0 (negatifs), aucun premier frag/mort n'est retenu (nil/nil).
func TestComputeFirstEventsPerMatch_AllPreT0(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, -2000),
		ev("m1", "death", &other, &p1, -500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, []string{"m1"})
	if len(got) != 1 {
		t.Fatalf("want 1 row (placeholder), got %d", len(got))
	}
	if got[0].FirstKillMS != nil || got[0].FirstDeathMS != nil {
		t.Errorf("all pre-T0 events should yield nil/nil, got %+v", got[0])
	}
}

// ---------------------------------------------------------------------------
// ComputeFirstEventsByActor (surface multi-joueurs : page Escouade)
// ---------------------------------------------------------------------------

func actorEv(matchID, xuid string, isKill bool, timeMS int64) FirstEventActor {
	return FirstEventActor{MatchID: matchID, XUID: xuid, IsKill: isKill, TimeMS: timeMS}
}

// TestComputeFirstEventsByActor_MinPerPlayerAndMatch : le minimum est pris par
// (joueur, match, type) ; chaque joueur demande obtient une row par matchID.
func TestComputeFirstEventsByActor_MinPerPlayerAndMatch(t *testing.T) {
	t.Parallel()
	events := []FirstEventActor{
		actorEv("m1", "a", true, 9000),
		actorEv("m1", "a", true, 4000), // min pour a/m1
		actorEv("m1", "a", false, 15000),
		actorEv("m1", "b", true, 2000),
		actorEv("m2", "a", false, 7000),
	}
	got := ComputeFirstEventsByActor(events, []string{"a", "b"}, []string{"m1", "m2"})
	if len(got) != 2 {
		t.Fatalf("want 2 joueurs, got %d", len(got))
	}
	rowsA := got["a"]
	if len(rowsA) != 2 || rowsA[0].MatchID != "m1" || rowsA[1].MatchID != "m2" {
		t.Fatalf("ordre des matchs non preserve pour a: %+v", rowsA)
	}
	if rowsA[0].FirstKillMS == nil || *rowsA[0].FirstKillMS != 4000 {
		t.Errorf("a/m1 firstKill want 4000, got %v", rowsA[0].FirstKillMS)
	}
	if rowsA[0].FirstDeathMS == nil || *rowsA[0].FirstDeathMS != 15000 {
		t.Errorf("a/m1 firstDeath want 15000, got %v", rowsA[0].FirstDeathMS)
	}
	if rowsA[1].FirstKillMS != nil || rowsA[1].FirstDeathMS == nil || *rowsA[1].FirstDeathMS != 7000 {
		t.Errorf("a/m2 want (nil, 7000), got %+v", rowsA[1])
	}
	rowsB := got["b"]
	if len(rowsB) != 2 || rowsB[1].FirstKillMS != nil || rowsB[1].FirstDeathMS != nil {
		t.Errorf("b/m2 sans event doit rester nil/nil, got %+v", rowsB)
	}
}

// TestComputeFirstEventsByActor_SkipsPreT0AndUnknownPlayers : un TimeMS negatif
// (countdown apres correction T0) est ignore, et un joueur hors liste n'est
// jamais servi.
func TestComputeFirstEventsByActor_SkipsPreT0AndUnknownPlayers(t *testing.T) {
	t.Parallel()
	events := []FirstEventActor{
		actorEv("m1", "a", true, -3000), // countdown
		actorEv("m1", "a", true, 12000),
		actorEv("m1", "z", true, 1000), // joueur non demande
	}
	got := ComputeFirstEventsByActor(events, []string{"a"}, []string{"m1"})
	if _, ok := got["z"]; ok {
		t.Error("un joueur hors liste ne doit pas etre servi")
	}
	rows := got["a"]
	if len(rows) != 1 || rows[0].FirstKillMS == nil || *rows[0].FirstKillMS != 12000 {
		t.Errorf("a/m1 firstKill want 12000 (countdown ignore), got %+v", rows)
	}
}

// TestComputeFirstEventsByActor_PlayerWithoutEvent : un joueur demande sans
// aucun event obtient quand meme sa serie (rows toutes nulles) — la decision de
// masquer la bande appartient a l'appelant.
func TestComputeFirstEventsByActor_PlayerWithoutEvent(t *testing.T) {
	t.Parallel()
	got := ComputeFirstEventsByActor(nil, []string{"a"}, []string{"m1", "m2"})
	rows, ok := got["a"]
	if !ok || len(rows) != 2 {
		t.Fatalf("want 2 rows placeholder pour a, got %+v", got)
	}
	for _, r := range rows {
		if r.FirstKillMS != nil || r.FirstDeathMS != nil {
			t.Errorf("row sans event doit etre nil/nil, got %+v", r)
		}
	}
}
