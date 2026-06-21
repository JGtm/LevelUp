package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// mockMainTeamLoader implémente mainTeamParticipantsLoader pour les tests.
type mockMainTeamLoader struct {
	byMatch map[string][]domain.AllyParticipant
	err     error
	gotIDs  []string
}

func (m *mockMainTeamLoader) LoadMainTeamParticipants(_ context.Context, _ string, matchIDs []string) ([]domain.AllyParticipant, error) {
	m.gotIDs = append(m.gotIDs, matchIDs...)
	if m.err != nil {
		return nil, m.err
	}
	var out []domain.AllyParticipant
	for _, id := range matchIDs {
		out = append(out, m.byMatch[id]...)
	}
	return out, nil
}

func squadRow(matchID, label string, withFriends bool) canonical.PlayerMatchRow {
	lbl := label
	return canonical.PlayerMatchRow{
		Summary:    canonical.MatchSummary{MatchID: matchID},
		Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &lbl, IsWithFriends: withFriends},
	}
}

func TestSessionCoreTeammates_IntersectionExcludesRotating(t *testing.T) {
	// Zoe seulement dans m1 → exclue (pas dans tous les matchs). Alice/Bob partout.
	alliesByMatch := map[string][]string{
		"m1": {"Alice", "Bob", "Zoe"},
		"m2": {"Alice", "Bob"},
		"m3": {"Alice", "Bob"},
	}
	got := sessionCoreTeammates([]string{"m1", "m2", "m3"}, alliesByMatch, nil)
	want := []string{"Alice", "Bob"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("intersection: got %v want %v", got, want)
	}
}

func TestSessionCoreTeammates_CapAtMax(t *testing.T) {
	// 4 amis présents dans les 2 matchs → cappé à maxHomeSessionTeammates (3), ordre alpha.
	allies := []string{"Alice", "Bob", "Carol", "Dan"}
	alliesByMatch := map[string][]string{"m1": allies, "m2": allies}
	got := sessionCoreTeammates([]string{"m1", "m2"}, alliesByMatch, nil)
	want := []string{"Alice", "Bob", "Carol"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cap: got %v want %v", got, want)
	}
}

func TestSessionCoreTeammates_FriendRestriction(t *testing.T) {
	alliesByMatch := map[string][]string{
		"m1": {"Alice", "Random1"},
		"m2": {"Alice", "Random2"},
	}
	friendSet := map[string]struct{}{"alice": {}} // gamertags minuscules
	got := sessionCoreTeammates([]string{"m1", "m2"}, alliesByMatch, friendSet)
	want := []string{"Alice"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("friend restriction: got %v want %v", got, want)
	}
}

func TestSessionCoreTeammates_RotatingFriendsYieldsNil(t *testing.T) {
	// Aucun ami commun à tous les matchs → nil (→ /squad ouvre la session sans composition).
	alliesByMatch := map[string][]string{"m1": {"Alice"}, "m2": {"Bob"}}
	if got := sessionCoreTeammates([]string{"m1", "m2"}, alliesByMatch, nil); got != nil {
		t.Fatalf("rotating: want nil, got %v", got)
	}
}

func TestSessionCoreTeammates_NoneMatch(t *testing.T) {
	if got := sessionCoreTeammates([]string{"m1"}, map[string][]string{}, nil); got != nil {
		t.Fatalf("no allies: want nil, got %v", got)
	}
}

func TestEnrichSquadSessionsTeammates_AssignsCoreFriends(t *testing.T) {
	loader := &mockMainTeamLoader{byMatch: map[string][]domain.AllyParticipant{
		// Session A (m1,m2) : Alice+Bob dans les deux ; Random non-ami ; Me = joueur principal.
		"m1": {{MatchID: "m1", XUID: "main", Gamertag: "Me"}, {MatchID: "m1", XUID: "xa", Gamertag: "Alice"}, {MatchID: "m1", XUID: "xb", Gamertag: "Bob"}},
		"m2": {{MatchID: "m2", XUID: "xa", Gamertag: "Alice"}, {MatchID: "m2", XUID: "xb", Gamertag: "Bob"}, {MatchID: "m2", XUID: "xr", Gamertag: "Random"}},
		// Session B (m3) : Carol.
		"m3": {{MatchID: "m3", XUID: "xc", Gamertag: "Carol"}},
	}}
	s := &HomeService{
		xuid:                   "main",
		gamertag:               "Me",
		sessionTeammatesLoader: loader,
		sessionFriendsResolver: func(context.Context) []string { return []string{"Alice", "Bob", "Carol"} },
	}
	sessions := []domain.SessionSummaryItem{{SessionLabel: "A"}, {SessionLabel: "B"}}
	rows := []canonical.PlayerMatchRow{
		squadRow("m1", "A", true),
		squadRow("m2", "A", true),
		squadRow("m3", "B", true),
		squadRow("m_solo", "C", false), // solo : ignoré
	}

	s.enrichSquadSessionsTeammates(context.Background(), sessions, rows)

	if got, want := sessions[0].Teammates, []string{"Alice", "Bob"}; !reflect.DeepEqual(got, want) {
		t.Errorf("session A teammates: got %v want %v (Random exclu non-ami, Me=principal exclu)", got, want)
	}
	if got, want := sessions[1].Teammates, []string{"Carol"}; !reflect.DeepEqual(got, want) {
		t.Errorf("session B teammates: got %v want %v", got, want)
	}
	// m_solo (is_with_friends=false) ne doit pas avoir été chargé.
	for _, id := range loader.gotIDs {
		if id == "m_solo" {
			t.Errorf("le match solo m_solo ne doit pas être scanné, gotIDs=%v", loader.gotIDs)
		}
	}
}

func TestEnrichSquadSessionsTeammates_NoOpWithoutLoader(t *testing.T) {
	s := &HomeService{xuid: "main"} // loader nil
	sessions := []domain.SessionSummaryItem{{SessionLabel: "A"}}
	s.enrichSquadSessionsTeammates(context.Background(), sessions, []canonical.PlayerMatchRow{squadRow("m1", "A", true)})
	if sessions[0].Teammates != nil {
		t.Errorf("loader nil : Teammates doit rester nil, got %v", sessions[0].Teammates)
	}
}

func TestEnrichSquadSessionsTeammates_LoaderErrorDegradesGracefully(t *testing.T) {
	loader := &mockMainTeamLoader{err: errors.New("db down")}
	s := &HomeService{xuid: "main", gamertag: "Me", sessionTeammatesLoader: loader}
	sessions := []domain.SessionSummaryItem{{SessionLabel: "A"}}
	s.enrichSquadSessionsTeammates(context.Background(), sessions, []canonical.PlayerMatchRow{squadRow("m1", "A", true)})
	if sessions[0].Teammates != nil {
		t.Errorf("erreur loader : Teammates doit rester nil (dégradation), got %v", sessions[0].Teammates)
	}
}
