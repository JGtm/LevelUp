package halo_5

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// fixtureEvents — slice RÉELLE (shapes exactes capturées par cmd/probe-h5 sur un
// match arena de JGtm) couvrant tous les types d'events + un type inconnu (skip).
const fixtureEvents = `{"GameEvents":[
 {"RoundIndex":0,"TimeSinceStart":"PT0.0950007S","EventName":"RoundStart"},
 {"Player":{"Gamertag":"JGtm","Xuid":null},"WeaponAttachmentIds":[2758383128],"WeaponStockId":669296699,"TimeSinceStart":"PT24.2979846S","EventName":"WeaponPickup"},
 {"Player":{"Gamertag":"JGtm","Xuid":null},"TimeSinceStart":"PT24.2819841S","EventName":"PlayerSpawn"},
 {"IsHeadshot":true,"IsMelee":false,"IsGroundPound":false,"IsShoulderBash":false,"IsWeapon":true,"Killer":{"Gamertag":"Madman684844","Xuid":null},"KillerAgent":1,"KillerWeaponAttachmentIds":[2758383128],"KillerWeaponStockId":2650887244,"KillerWorldLocation":{"x":24.908062,"y":-41.9863129,"z":-7.5632887},"Victim":{"Gamertag":"Madina97294","Xuid":null},"VictimAgent":1,"VictimStockId":0,"VictimWorldLocation":{"x":24.669899,"y":-41.7344475,"z":-7.5619984},"TimeSinceStart":"PT33.2154416S","EventName":"Death"},
 {"MedalId":3001183151,"Player":{"Gamertag":"Madman684844","Xuid":null},"TimeSinceStart":"PT33.2204424S","EventName":"Medal"},
 {"ImpulseId":2556889090,"Player":{"Gamertag":"Madman684844","Xuid":null},"TimeSinceStart":"PT34.0640328S","EventName":"Impulse"},
 {"IsHeadshot":false,"IsMelee":true,"IsWeapon":false,"Killer":{"Gamertag":"JGtm","Xuid":null},"Victim":{"Gamertag":"Pancakeflips","Xuid":null},"TimeSinceStart":"PT40S","EventName":"Death"},
 {"Player":{"Gamertag":"JGtm","Xuid":null},"WeaponStockId":669296699,"TimeSinceStart":"PT50S","EventName":"WeaponDrop"},
 {"TimeSinceStart":"PT60S","EventName":"SomethingBrandNew"},
 {"RoundIndex":0,"TimeSinceStart":"PT5M41.7899986S","EventName":"RoundEnd"}
]}`

func parseFixture(t *testing.T) *h5MatchEventsResponse {
	t.Helper()
	var r h5MatchEventsResponse
	if err := json.Unmarshal([]byte(fixtureEvents), &r); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return &r
}

func TestMapH5Events_Kill(t *testing.T) {
	evs := mapH5Events(parseFixture(t), canonical.MatchEventOptions{})
	// 10 events, 1 inconnu (skip) → 9 mappés.
	if len(evs) != 9 {
		t.Fatalf("events mappés = %d, want 9 (le type inconnu doit être skippé)", len(evs))
	}

	// 1er kill = headshot arme.
	var kill *canonical.MatchEvent
	for i := range evs {
		if evs[i].Type == canonical.MatchEventKill {
			kill = &evs[i]
			break
		}
	}
	if kill == nil {
		t.Fatal("aucun event kill")
	}
	if kill.Killer == nil || kill.Killer.Gamertag != "Madman684844" {
		t.Errorf("killer = %+v, want Madman684844", kill.Killer)
	}
	if kill.Victim == nil || kill.Victim.Gamertag != "Madina97294" {
		t.Errorf("victim = %+v, want Madina97294", kill.Victim)
	}
	if kill.Kind != canonical.KillKindWeapon {
		t.Errorf("kind = %q, want weapon", kill.Kind)
	}
	if !kill.Headshot {
		t.Error("headshot devrait être true")
	}
	if kill.Weapon == nil || kill.Weapon.ID != "2650887244" {
		t.Errorf("weapon = %+v, want ID 2650887244", kill.Weapon)
	}
	if kill.KillerLoc == nil || kill.VictimLoc == nil {
		t.Error("positions monde devraient être présentes (Halo 5 natif)")
	}
	if kill.TimeMs != 33215 {
		t.Errorf("time_ms = %d, want 33215 (PT33.2154416S)", kill.TimeMs)
	}
}

func TestMapH5Events_MeleeAndIdentities(t *testing.T) {
	evs := mapH5Events(parseFixture(t), canonical.MatchEventOptions{})
	// 2e kill = melee.
	var meleeFound bool
	for _, e := range evs {
		if e.Type == canonical.MatchEventKill && e.Kind == canonical.KillKindMelee {
			meleeFound = true
			if e.Weapon != nil {
				t.Error("kill melee ne devrait pas porter d'arme")
			}
		}
	}
	if !meleeFound {
		t.Error("le kill melee n'a pas été mappé en KillKindMelee")
	}
}

func TestH5KillKind_AssassinationPriority(t *testing.T) {
	cases := []struct {
		name string
		ev   h5GameEvent
		want canonical.KillKind
	}{
		// L'assassinat est aussi tagué IsMelee par l'API → il DOIT primer.
		{"assassination bat melee", h5GameEvent{IsAssassination: true, IsMelee: true}, canonical.KillKindAssassination},
		{"assassination seul", h5GameEvent{IsAssassination: true}, canonical.KillKindAssassination},
		{"melee sans assassinat", h5GameEvent{IsMelee: true}, canonical.KillKindMelee},
		{"groundpound", h5GameEvent{IsGroundPound: true}, canonical.KillKindGroundPound},
		{"shoulderbash", h5GameEvent{IsShoulderBash: true}, canonical.KillKindShoulderBash},
		{"weapon par défaut", h5GameEvent{IsWeapon: true}, canonical.KillKindWeapon},
	}
	for _, c := range cases {
		ev := c.ev
		if got := h5KillKind(&ev); got != c.want {
			t.Errorf("%s: h5KillKind = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMapH5Events_MedalImpulseRound(t *testing.T) {
	evs := mapH5Events(parseFixture(t), canonical.MatchEventOptions{})
	byType := map[canonical.MatchEventType]*canonical.MatchEvent{}
	for i := range evs {
		byType[evs[i].Type] = &evs[i]
	}
	if m := byType[canonical.MatchEventMedal]; m == nil || m.RefID == nil || *m.RefID != "3001183151" {
		t.Errorf("medal RefID = %+v, want 3001183151", m)
	}
	if im := byType[canonical.MatchEventImpulse]; im == nil || im.RefID == nil || *im.RefID != "2556889090" {
		t.Errorf("impulse RefID = %+v, want 2556889090", im)
	}
	if re := byType[canonical.MatchEventRoundEnd]; re == nil || re.Round == nil || *re.Round != 0 {
		t.Errorf("round_end Round = %+v, want 0", re)
	}
	// PT5M41.79S → 341790 ms.
	if re := byType[canonical.MatchEventRoundEnd]; re != nil && re.TimeMs != 341790 {
		t.Errorf("round_end time_ms = %d, want 341790", re.TimeMs)
	}
}

func TestMapH5Events_Filter(t *testing.T) {
	evs := mapH5Events(parseFixture(t), canonical.MatchEventOptions{Types: []canonical.MatchEventType{canonical.MatchEventKill}})
	if len(evs) != 2 {
		t.Fatalf("filtre kill → %d events, want 2", len(evs))
	}
	for _, e := range evs {
		if e.Type != canonical.MatchEventKill {
			t.Errorf("event non-kill malgré le filtre: %q", e.Type)
		}
	}
}

func TestAdapter_LoadMatchEvents_Live(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{ev: parseFixture(t)}), nil)
	tl, err := a.LoadMatchEvents(context.Background(), "match-123", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("LoadMatchEvents: %v", err)
	}
	if tl.MatchID != "match-123" {
		t.Errorf("match_id = %q, want match-123", tl.MatchID)
	}
	if len(tl.Events) != 9 {
		t.Errorf("events = %d, want 9", len(tl.Events))
	}
}

func TestAdapter_LoadMatchEvents_404Graceful(t *testing.T) {
	notFound := &HTTPError{StatusCode: http.StatusNotFound, URL: "x", Err: errors.New("absent")}
	a := NewDataAdapter(srcFactory(&fakeSource{evErr: notFound}), nil)
	tl, err := a.LoadMatchEvents(context.Background(), "m", canonical.MatchEventOptions{})
	if err != nil {
		t.Fatalf("404 doit dégrader (timeline vide), pas erreur dure : %v", err)
	}
	if tl == nil || len(tl.Events) != 0 {
		t.Errorf("404 → timeline vide attendue, got %+v", tl)
	}
}

func TestAdapter_LoadMatchEvents_NilFactory(t *testing.T) {
	a := NewDataAdapter(nil, nil)
	if _, err := a.LoadMatchEvents(context.Background(), "m", canonical.MatchEventOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("nil factory → ErrCapabilityNotSupported, got %v", err)
	}
}

func TestParseISO8601DurationMs(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"PT33.2154416S", 33215, true},
		{"PT5M41.7899986S", 341790, true},
		{"PT0.0950007S", 95, true},
		{"PT", 0, false},
		{"", 0, false},
		{"garbage", 0, false},
	}
	for _, c := range cases {
		got, ok := parseISO8601DurationMs(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseISO8601DurationMs(%q) = (%d,%v), want (%d,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
