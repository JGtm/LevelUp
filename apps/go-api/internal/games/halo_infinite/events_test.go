package halo_infinite

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// fakeEventsSource implémente EventsSource sans DB (injecte events + timeline).
type fakeEventsSource struct {
	events []canonical.HighlightEvent
	evErr  error
	tl     domain.MatchTimeline
	tlErr  error
}

func (f *fakeEventsSource) LoadHighlightEvents(context.Context, string) ([]canonical.HighlightEvent, error) {
	return f.events, f.evErr
}

func (f *fakeEventsSource) GetMatchTimeline(context.Context, string) (domain.MatchTimeline, error) {
	return f.tl, f.tlErr
}

// fixtureHE — highlight_events bruts d'un match : 2 paires kill/death (l'une
// pendant le countdown), 1 médaille, 1 event de mode. xuid = sens dépendant du
// type (tueur pour kill, victime pour death, joueur centrant pour medal/mode).
func fixtureHE() []canonical.HighlightEvent {
	return []canonical.HighlightEvent{
		{MatchID: "m1", EventType: "kill", XUID: "killerA", TimeMS: 8000},
		{MatchID: "m1", EventType: "death", XUID: "victimA", TimeMS: 8003},
		{MatchID: "m1", EventType: "medal", XUID: "killerA", TimeMS: 8050},
		{MatchID: "m1", EventType: "mode", XUID: "playerB", TimeMS: 9000},
		// Paire pendant le countdown (raw < T0) → corrigée négative → skippée.
		{MatchID: "m1", EventType: "kill", XUID: "k2", TimeMS: 1000},
		{MatchID: "m1", EventType: "death", XUID: "v2", TimeMS: 1002},
	}
}

func TestMapInfiniteEvents_PairsKillsAndCorrectsT0(t *testing.T) {
	tl := domain.NewMatchTimeline(600_000, 5000) // T0 = 5000 ms (countdown)
	evs := mapInfiniteEvents(fixtureHE(), tl, canonical.MatchEventOptions{})

	// 1 kill (la paire de countdown est skippée) + 1 medal + 1 impulse = 3.
	if len(evs) != 3 {
		t.Fatalf("events = %d, want 3 (kill + medal + impulse ; paire countdown skippée)", len(evs))
	}

	var kill *canonical.MatchEvent
	for i := range evs {
		if evs[i].Type == canonical.MatchEventKill {
			kill = &evs[i]
		}
	}
	if kill == nil {
		t.Fatal("aucun MatchEventKill")
	}
	if kill.Killer == nil || kill.Killer.XUID != "killerA" {
		t.Errorf("killer = %+v, want XUID killerA", kill.Killer)
	}
	if kill.Victim == nil || kill.Victim.XUID != "victimA" {
		t.Errorf("victim = %+v, want XUID victimA", kill.Victim)
	}
	// 8000 - 5000 = 3000.
	if kill.TimeMs != 3000 {
		t.Errorf("time_ms = %d, want 3000 (8000 - T0 5000)", kill.TimeMs)
	}
	// Dégradation Infinite : pas d'arme, pas de mécanique, pas de positions.
	if kill.Weapon != nil || kill.KillerLoc != nil || kill.VictimLoc != nil {
		t.Errorf("dégradation Infinite attendue : Weapon/Loc nil, got %+v", kill)
	}
	if kill.Kind != "" || kill.Headshot {
		t.Errorf("Kind/Headshot doivent rester zéro-value (mécanique inconnue), got Kind=%q Headshot=%v", kill.Kind, kill.Headshot)
	}
	// Tri par TimeMs croissant : kill(3000) < medal(3050) < impulse(4000).
	for i := 1; i < len(evs); i++ {
		if evs[i-1].TimeMs > evs[i].TimeMs {
			t.Errorf("events non triés par TimeMs: %d > %d", evs[i-1].TimeMs, evs[i].TimeMs)
		}
	}
}

func TestMapInfiniteEvents_MedalImpulse(t *testing.T) {
	tl := domain.NewMatchTimeline(600_000, 5000)
	evs := mapInfiniteEvents(fixtureHE(), tl, canonical.MatchEventOptions{})
	byType := map[canonical.MatchEventType]*canonical.MatchEvent{}
	for i := range evs {
		byType[evs[i].Type] = &evs[i]
	}
	if m := byType[canonical.MatchEventMedal]; m == nil || m.Player == nil || m.Player.XUID != "killerA" {
		t.Errorf("medal Player = %+v, want XUID killerA", m)
	} else if m.RefID != nil {
		t.Errorf("medal RefID doit être nil (id non porté par highlight_events), got %v", *m.RefID)
	}
	if im := byType[canonical.MatchEventImpulse]; im == nil || im.Player == nil || im.Player.XUID != "playerB" {
		t.Errorf("impulse Player = %+v, want XUID playerB", im)
	}
}

func TestMapInfiniteEvents_Filter(t *testing.T) {
	tl := domain.NewMatchTimeline(600_000, 5000)
	evs := mapInfiniteEvents(fixtureHE(), tl, canonical.MatchEventOptions{
		Types: []canonical.MatchEventType{canonical.MatchEventKill},
	})
	if len(evs) != 1 {
		t.Fatalf("filtre kill → %d events, want 1", len(evs))
	}
	if evs[0].Type != canonical.MatchEventKill {
		t.Errorf("event non-kill malgré le filtre: %q", evs[0].Type)
	}
}

func TestMapInfiniteEvents_NoT0Identity(t *testing.T) {
	// Sans T0 (MatchTimeline zéro-value) : aucun event n'est skippé, TimeMs = raw.
	evs := mapInfiniteEvents(fixtureHE(), domain.MatchTimeline{}, canonical.MatchEventOptions{})
	// 2 kills (les deux paires passent) + 1 medal + 1 impulse = 4.
	if len(evs) != 4 {
		t.Fatalf("sans T0 → %d events, want 4 (2 kills + medal + impulse)", len(evs))
	}
}

func TestMapInfiniteEvents_Empty(t *testing.T) {
	if evs := mapInfiniteEvents(nil, domain.MatchTimeline{}, canonical.MatchEventOptions{}); len(evs) != 0 {
		t.Errorf("entrée vide → 0 events, got %d", len(evs))
	}
}

func TestInfiniteEventLimitations(t *testing.T) {
	lim := infiniteEventLimitations()
	if len(lim) != 3 {
		t.Fatalf("limitations = %d, want 3", len(lim))
	}
	keys := map[string]bool{}
	for _, g := range lim {
		keys[g.CapabilityKey] = true
	}
	for _, want := range []games.CapabilityKey{
		games.CapMatchKillfeedPerKill, games.CapMatchEventsSpatial, games.CapMatchEventsTimeline,
	} {
		if !keys[string(want)] {
			t.Errorf("limitation manquante pour %q", want)
		}
	}
}

func TestAdapter_LoadMatchEvents_Live(t *testing.T) {
	src := &fakeEventsSource{events: fixtureHE(), tl: domain.NewMatchTimeline(600_000, 5000)}
	a := NewDataAdapter(nil, nil).WithEventsSource(src)
	tl, err := a.LoadMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("LoadMatchEvents: %v", err)
	}
	if tl.MatchID != "m1" {
		t.Errorf("match_id = %q, want m1", tl.MatchID)
	}
	if len(tl.Events) != 3 {
		t.Errorf("events = %d, want 3", len(tl.Events))
	}
	if len(tl.Limitations) != 3 {
		t.Errorf("limitations = %d, want 3 (dégradations Infinite reportées)", len(tl.Limitations))
	}
}

func TestAdapter_LoadMatchEvents_NilEventsSource(t *testing.T) {
	a := NewDataAdapter(nil, nil) // pas de WithEventsSource
	if _, err := a.LoadMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("events source nil → ErrCapabilityNotSupported, got %v", err)
	}
}

func TestAdapter_LoadMatchEvents_EmptyMatchID(t *testing.T) {
	a := NewDataAdapter(nil, nil).WithEventsSource(&fakeEventsSource{})
	if _, err := a.LoadMatchEvents(context.Background(), "  ", canonical.MatchEventOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("matchID vide → ErrCapabilityNotSupported, got %v", err)
	}
}

func TestAdapter_LoadMatchEvents_EventsErrorPropagated(t *testing.T) {
	src := &fakeEventsSource{evErr: errors.New("boom shared reader")}
	a := NewDataAdapter(nil, nil).WithEventsSource(src)
	if _, err := a.LoadMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{}); err == nil {
		t.Error("erreur de lecture highlight_events doit être propagée")
	}
}

func TestAdapter_LoadMatchEvents_TimelineErrorGraceful(t *testing.T) {
	// T0 indisponible → dégradation gracieuse (T0=0), pas d'échec dur.
	src := &fakeEventsSource{events: fixtureHE(), tlErr: errors.New("registry indispo")}
	a := NewDataAdapter(nil, nil).WithEventsSource(src)
	tl, err := a.LoadMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("erreur T0 doit dégrader gracieusement, got %v", err)
	}
	// Sans T0, les 2 paires passent → 4 events.
	if len(tl.Events) != 4 {
		t.Errorf("dégradation T0=0 → 4 events attendus, got %d", len(tl.Events))
	}
}
