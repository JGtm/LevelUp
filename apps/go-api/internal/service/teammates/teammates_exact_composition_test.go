package teammates

// teammates_exact_composition_test.go — composition EXACTE (exclusivité).
//
// L'intersection garantit "le main a joué AVEC tous les sélectionnés" mais PAS
// "aucun autre coéquipier connu n'était présent". filterExactComposition ajoute
// l'exclusivité à partir de l'équipe alliée du main par match : un match où un
// coéquipier connu HORS sélection (extraPool) figure sur l'équipe du main est
// écarté ; les fills de lobby / bots / adversaires (hors pool) sont conservés.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func ally(matchID, xuid string) domain.AllyParticipant {
	return domain.AllyParticipant{MatchID: matchID, XUID: xuid}
}

func setXUIDs(xs ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		out[x] = struct{}{}
	}
	return out
}

// TestMatchHasExactComposition couvre le prédicat au cœur du fix.
func TestMatchHasExactComposition(t *testing.T) {
	extra := setXUIDs("xc", "xd") // coéquipiers connus hors sélection
	selected := []string{"xa", "xb"}

	cases := []struct {
		name string
		team map[string]struct{}
		want bool
	}{
		{"exactement {A,B}", setXUIDs("px", "xa", "xb"), true},
		{"{A,B} + fill hors pool", setXUIDs("px", "xa", "xb", "fill"), true},
		{"{A,B} + C connu hors sélection", setXUIDs("px", "xa", "xb", "xc"), false},
		{"un sélectionné absent (B sur l'équipe adverse)", setXUIDs("px", "xa", "fill"), false},
		{"équipe inconnue (nil)", nil, false},
	}
	for _, c := range cases {
		if got := matchHasExactComposition(c.team, extra, selected); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestMatchHasExactComposition_MonoSelection : compo mono {A}. Un match {A,B} avec
// B dans le pool est exclu ; {A} seul est inclus.
func TestMatchHasExactComposition_MonoSelection(t *testing.T) {
	extra := setXUIDs("xb")
	selected := []string{"xa"}
	if matchHasExactComposition(setXUIDs("px", "xa", "xb"), extra, selected) {
		t.Errorf("{A,B} avec B connu doit être exclu pour la compo mono {A}")
	}
	if !matchHasExactComposition(setXUIDs("px", "xa"), extra, selected) {
		t.Errorf("{A} seul doit être inclus pour la compo mono {A}")
	}
	if !matchHasExactComposition(setXUIDs("px", "xa", "randomFill"), extra, selected) {
		t.Errorf("{A} + fill hors pool doit être inclus pour la compo mono {A}")
	}
}

// TestFilterExactComposition_NilMapGraceful : mainTeamByMatch nil (chargement
// échoué / non tenté) => rows inchangés (dégradation gracieuse, page non blanchie).
func TestFilterExactComposition_NilMapGraceful(t *testing.T) {
	rows := []domain.SquadMatchRow{makeSquadRow("m1", "Bazaar", domain.OutcomeWin)}
	got := filterExactComposition(rows, nil, setXUIDs("xc"), []string{"xa"})
	if len(got) != 1 {
		t.Errorf("map nil doit laisser les rows intactes, got %d", len(got))
	}
}

// TestBuildExtraPoolXUIDs : pool = (top ∪ amis) \ sélection \ main.
func TestBuildExtraPoolXUIDs(t *testing.T) {
	top := []domain.TopTeammateRow{
		{XUID: "xa", Gamertag: "AllyA"},
		{XUID: "xb", Gamertag: "AllyB"},
		{XUID: "xc", Gamertag: "AllyC"},
	}
	friends := []string{"xf"}
	pool := buildExtraPoolXUIDs(top, friends, []string{"xa", "xb"}, "px")
	if _, ok := pool["xc"]; !ok {
		t.Errorf("xc (top non sélectionné) doit être dans le pool")
	}
	if _, ok := pool["xf"]; !ok {
		t.Errorf("xf (ami) doit être dans le pool")
	}
	if _, ok := pool["xa"]; ok {
		t.Errorf("xa (sélectionné) ne doit PAS être dans le pool")
	}
	if _, ok := pool["px"]; ok {
		t.Errorf("le main ne doit PAS être dans le pool")
	}
}

// TestResolveFriendXUIDs : amis résolus via la table gamertag→xuid des top rows,
// case-insensitive ; amis hors top ignorés.
func TestResolveFriendXUIDs(t *testing.T) {
	top := []domain.TopTeammateRow{{XUID: "xa", Gamertag: "AllyA"}}
	got := resolveFriendXUIDs([]string{"allya", "InconnuHorsTop"}, top)
	if len(got) != 1 || got[0] != "xa" {
		t.Errorf("resolveFriendXUIDs: want [xa], got %v", got)
	}
}

// TestBuildMainTeamXUIDSet : indexation match_id -> set(xuid).
func TestBuildMainTeamXUIDSet(t *testing.T) {
	allies := []domain.AllyParticipant{
		ally("m1", "px"), ally("m1", "xa"),
		ally("m2", "px"), ally("m2", "xa"), ally("m2", "xc"),
		ally("", "ignored"), ally("m3", ""), // entrées invalides ignorées
	}
	set := buildMainTeamXUIDSet(allies)
	if len(set["m1"]) != 2 {
		t.Errorf("m1: want 2 xuids, got %d", len(set["m1"]))
	}
	if _, ok := set["m2"]["xc"]; !ok {
		t.Errorf("m2 doit contenir xc")
	}
	if _, ok := set["m3"]; ok {
		t.Errorf("m3 (xuid vide) ne doit pas produire d'entrée")
	}
	if buildMainTeamXUIDSet(nil) != nil {
		t.Errorf("allies vide -> nil")
	}
}

// TestGetPage_ExactComposition_ExtraKnownTeammateExcluded : E2E. Composition
// {AllyA, AllyB} ; l'intersection donne {m1, m2}, mais m2 avait AllyC (top
// coéquipier hors sélection) sur l'équipe du main → m2 doit être écarté de TOUTES
// les surfaces (MapBreakdown + CompositionSessions). C'est le scénario du bug
// {JGtm, Chocoboflor} qui incluait à tort la session {…, Madina97294}.
func TestGetPage_ExactComposition_ExtraKnownTeammateExcluded(t *testing.T) {
	tS1 := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	tS2 := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)
	shared := []domain.SquadMatchRow{
		makeSquadRowSess("m1", "Bazaar", domain.OutcomeWin, "S_exact", tS1),
		makeSquadRowSess("m2", "Aquarius", domain.OutcomeWin, "S_with_C", tS2),
	}
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "xa", Gamertag: "AllyA", GamesTogether: 10},
			{XUID: "xb", Gamertag: "AllyB", GamesTogether: 10},
			{XUID: "xc", Gamertag: "AllyC", GamesTogether: 8}, // connu, PAS sélectionné
		},
		squadRowsByTeammate: map[string][]domain.SquadMatchRow{
			"xa": shared,
			"xb": shared,
		},
		// Équipe alliée du main par match : m2 embarque AllyC (xc).
		allyRows: []domain.AllyParticipant{
			ally("m1", "px"), ally("m1", "xa"), ally("m1", "xb"),
			ally("m2", "px"), ally("m2", "xa"), ally("m2", "xb"), ally("m2", "xc"),
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	maps := map[string]bool{}
	for _, row := range resp.MapBreakdown {
		maps[row.MapUI] = true
	}
	if maps["Aquarius"] {
		t.Errorf("le match m2 (avec AllyC hors sélection) a fui dans le MapBreakdown: %v", maps)
	}
	if !maps["Bazaar"] {
		t.Errorf("le match m1 (composition exacte) manque dans le MapBreakdown: %v", maps)
	}
	if len(resp.MapBreakdown) != 1 {
		t.Errorf("MapBreakdown: want 1 (m1 seul), got %d", len(resp.MapBreakdown))
	}

	sessions := map[string]bool{}
	for _, s := range resp.CompositionSessions {
		sessions[s.Label] = true
	}
	if sessions["S_with_C"] {
		t.Errorf("la session S_with_C (contenant AllyC) ne doit PAS être exposée pour la compo {A,B}")
	}
	if !sessions["S_exact"] || len(resp.CompositionSessions) != 1 {
		t.Errorf("CompositionSessions: want {S_exact}, got %v", sessions)
	}

	// Maillon 3 : le pool d'exclusion (extraPool = {xc}) est bien passé à
	// LoadMapStatsForSquad, et squadXUIDs = la sélection {xa, xb}.
	if len(repo.mapStatsExcludeXUIDs) != 1 || repo.mapStatsExcludeXUIDs[0] != "xc" {
		t.Errorf("LoadMapStatsForSquad excludeXUIDs: want [xc], got %v", repo.mapStatsExcludeXUIDs)
	}
	if len(repo.mapStatsSquadXUIDs) != 2 {
		t.Errorf("LoadMapStatsForSquad squadXUIDs: want 2 (xa,xb), got %v", repo.mapStatsSquadXUIDs)
	}
}
