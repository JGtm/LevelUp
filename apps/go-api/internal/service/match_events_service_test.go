package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// fakeEventsAdapter satisfait games.TitleDataAdapter via l'embedding nil de
// l'interface (seul LoadMatchEvents est appelé par le service ; les autres
// méthodes paniqueraient si appelées — elles ne le sont pas).
type fakeEventsAdapter struct {
	games.TitleDataAdapter
	tl  *canonical.MatchEventTimeline
	err error
}

func (f fakeEventsAdapter) LoadMatchEvents(context.Context, string, canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	return f.tl, f.err
}

type fakeGTResolver struct {
	m   map[string]string
	err error
}

func (f fakeGTResolver) ResolveGamertags(context.Context, []string) (map[string]string, error) {
	return f.m, f.err
}

func xuidOnlyTimeline() *canonical.MatchEventTimeline {
	return &canonical.MatchEventTimeline{
		MatchID: "m1",
		Events: []canonical.MatchEvent{
			{Type: canonical.MatchEventKill, TimeMs: 1000,
				Killer: &canonical.PlayerIdentity{XUID: "xK"},
				Victim: &canonical.PlayerIdentity{XUID: "xV"}},
			{Type: canonical.MatchEventMedal, TimeMs: 1100,
				Player: &canonical.PlayerIdentity{XUID: "xK"}},
		},
	}
}

func TestGetMatchEvents_NilAdapter(t *testing.T) {
	svc := NewMatchEventsService(nil, nil)
	if _, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("adapter nil → ErrCapabilityNotSupported, got %v", err)
	}
}

func TestGetMatchEvents_CapabilityErrorPropagated(t *testing.T) {
	svc := NewMatchEventsService(fakeEventsAdapter{err: games.ErrCapabilityNotSupported}, nil)
	if _, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("erreur capability adapter doit être propagée, got %v", err)
	}
}

func TestGetMatchEvents_NilTimeline(t *testing.T) {
	svc := NewMatchEventsService(fakeEventsAdapter{tl: nil, err: nil}, nil)
	tl, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("nil timeline ne doit pas être une erreur : %v", err)
	}
	if tl == nil || tl.MatchID != "m1" || len(tl.Events) != 0 {
		t.Errorf("nil timeline → timeline vide non nulle avec MatchID, got %+v", tl)
	}
}

func TestGetMatchEvents_EnrichesGamertags(t *testing.T) {
	resolver := fakeGTResolver{m: map[string]string{"xK": "KillerGT", "xV": "VictimGT"}}
	svc := NewMatchEventsService(fakeEventsAdapter{tl: xuidOnlyTimeline()}, resolver)
	tl, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	kill := tl.Events[0]
	if kill.Killer.Gamertag != "KillerGT" || kill.Victim.Gamertag != "VictimGT" {
		t.Errorf("gamertags non enrichis : killer=%q victim=%q", kill.Killer.Gamertag, kill.Victim.Gamertag)
	}
	if tl.Events[1].Player.Gamertag != "KillerGT" {
		t.Errorf("medal Player gamertag non enrichi : %q", tl.Events[1].Player.Gamertag)
	}
	// L'XUID est conservé (le chokepoint front a besoin des deux).
	if kill.Killer.XUID != "xK" {
		t.Errorf("XUID doit être conservé après enrichissement, got %q", kill.Killer.XUID)
	}
}

func TestGetMatchEvents_NilResolver_NoEnrich(t *testing.T) {
	svc := NewMatchEventsService(fakeEventsAdapter{tl: xuidOnlyTimeline()}, nil)
	tl, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if tl.Events[0].Killer.Gamertag != "" {
		t.Errorf("resolver nil → gamertag laissé vide (masqué au rendu), got %q", tl.Events[0].Killer.Gamertag)
	}
}

func TestGetMatchEvents_ResolverError_Graceful(t *testing.T) {
	resolver := fakeGTResolver{err: errors.New("v_gamertag_lookup indispo")}
	svc := NewMatchEventsService(fakeEventsAdapter{tl: xuidOnlyTimeline()}, resolver)
	tl, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("erreur resolver doit dégrader gracieusement, got %v", err)
	}
	if tl.Events[0].Killer.Gamertag != "" {
		t.Errorf("resolver en erreur → identités inchangées, got %q", tl.Events[0].Killer.Gamertag)
	}
}

func TestGetMatchEvents_GamertagKeyed_NoResolveNeeded(t *testing.T) {
	// Style Halo 5 : identités déjà gamertag-keyées (XUID vide) → rien à résoudre.
	tlIn := &canonical.MatchEventTimeline{
		MatchID: "m1",
		Events: []canonical.MatchEvent{
			{Type: canonical.MatchEventKill, TimeMs: 1000,
				Killer: &canonical.PlayerIdentity{Gamertag: "JGtm"},
				Victim: &canonical.PlayerIdentity{Gamertag: "Madina97294"}},
		},
	}
	// resolver qui paniquerait s'il était appelé avec des xuids (m nil) — ici non appelé.
	svc := NewMatchEventsService(fakeEventsAdapter{tl: tlIn}, fakeGTResolver{m: nil})
	tl, err := svc.GetMatchEvents(context.Background(), "m1", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if tl.Events[0].Killer.Gamertag != "JGtm" {
		t.Errorf("identité gamertag-keyée doit rester intacte, got %q", tl.Events[0].Killer.Gamertag)
	}
}
