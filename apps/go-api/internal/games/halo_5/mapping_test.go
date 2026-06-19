package halo_5

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// fixtureMatches reproduit le shape REEL capture par la sonde live (cmd/probe-h5,
// JGtm) : un match arena que JGtm a GAGNE (son equipe = Rank 1, Score le plus haut).
const fixtureMatches = `{
  "Start":0,"Count":1,"ResultCount":1,
  "Results":[{
    "Id":{"MatchId":"5d16ff8d-43df-4300-8c87-ed83b03674d2","GameMode":1},
    "HopperId":"f0c9ef9a-48bd-4b24-9db3-2c76b4e23450",
    "MapId":"c7b7baf0-f206-11e4-ae9a-24be05e24f7e",
    "GameBaseVariantId":"a2949322-dc84-45ab-8454-cf94fb28c189",
    "MatchDuration":"PT5M41.7930011S",
    "MatchCompletedDate":{"ISO8601Date":"2023-04-05T00:00:00Z"},
    "Teams":[{"Id":0,"Score":2,"Rank":2},{"Id":1,"Score":3,"Rank":1}],
    "Players":[{"Player":{"Gamertag":"JGtm","Xuid":null},"TeamId":1,"Rank":1,"Result":3,"TotalKills":10,"TotalDeaths":14,"TotalAssists":11}],
    "IsTeamGame":true,"SeasonId":""
  }]
}`

// fixtureServiceRecord reproduit le shape REEL du service record arena : 1 playlist,
// pic CSR Diamant 5 (DesignationId 4, Tier 5).
const fixtureServiceRecord = `{
  "Results":[{"Id":"JGtm","ResultCode":0,"Result":{"ArenaStats":{
    "ArenaPlaylistStats":[{
      "PlaylistId":"2323b76a-db98-4e03-aa37-e171cfbdd1a4",
      "TotalKills":20,"TotalDeaths":39,"TotalAssists":5,"TotalHeadshots":15,
      "TotalShotsFired":90,"TotalShotsLanded":35,
      "TotalGamesCompleted":3,"TotalGamesWon":0,"TotalGamesLost":3,"TotalGamesTied":0,
      "TotalTimePlayed":"PT12M12.9155475S"
    }],
    "HighestCsrAttained":{"Tier":5,"DesignationId":4,"Csr":0,"PercentToNextTier":32}
  }}}]
}`

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestParseISO8601DurationSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want *int
	}{
		{"", nil},
		{"garbage", nil},
		{"PT5M41.7930011S", ptrInt(342)}, // 341.79 -> arrondi 342
		{"PT12M12.9155475S", ptrInt(733)},
		{"PT1H2M3S", ptrInt(3723)},
		{"PT9.35S", ptrInt(9)},
	}
	for _, c := range cases {
		got := parseISO8601DurationSeconds(c.in)
		if (got == nil) != (c.want == nil) {
			t.Fatalf("parse(%q) nullité = %v, want %v", c.in, got, c.want)
		}
		if got != nil && *got != *c.want {
			t.Errorf("parse(%q) = %d, want %d", c.in, *got, *c.want)
		}
	}
}

func TestDeriveOutcome(t *testing.T) {
	teamRank1 := 1
	teamRank2 := 2
	if got := deriveOutcome(1, &teamRank1); got != canonical.OutcomeWin {
		t.Errorf("team rank 1 -> %q, want win", got)
	}
	if got := deriveOutcome(1, &teamRank2); got != canonical.OutcomeLoss {
		t.Errorf("team rank 2 -> %q, want loss (rang d'equipe prioritaire)", got)
	}
	if got := deriveOutcome(1, nil); got != canonical.OutcomeWin {
		t.Errorf("FFA player rank 1 -> %q, want win", got)
	}
}

func TestMapMatchSummaries_RealShape(t *testing.T) {
	var resp H5MatchesResponse
	if err := json.Unmarshal([]byte(fixtureMatches), &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	out := mapMatchSummaries(&resp, "JGtm")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	m := out[0]
	if m.MatchID != "5d16ff8d-43df-4300-8c87-ed83b03674d2" {
		t.Errorf("MatchID = %q", m.MatchID)
	}
	if m.DurationSeconds == nil || *m.DurationSeconds != 342 {
		t.Errorf("DurationSeconds = %v, want 342", m.DurationSeconds)
	}
	// StartedAtUTC = fin (2023-04-05T00:00:00Z) - 342s = 2023-04-04T23:54:18Z.
	wantStart := time.Date(2023, 4, 4, 23, 54, 18, 0, time.UTC)
	if !m.StartedAtUTC.Equal(wantStart) {
		t.Errorf("StartedAtUTC = %s, want %s (fin - duree)", m.StartedAtUTC, wantStart)
	}
	// JGtm sur l'equipe 1 (Rank 1) -> WIN (data-grounded, pas l'enum Result).
	if m.Outcome != canonical.OutcomeWin {
		t.Errorf("Outcome = %q, want win", m.Outcome)
	}
	if m.MatchType != canonical.MatchTypeSocial {
		t.Errorf("MatchType = %q, want social", m.MatchType)
	}
	if m.Playlist == nil || m.Playlist.ID != "f0c9ef9a-48bd-4b24-9db3-2c76b4e23450" {
		t.Errorf("Playlist = %+v", m.Playlist)
	}
	if m.Map == nil || m.Map.Kind != "map" {
		t.Errorf("Map = %+v", m.Map)
	}
	if len(m.Teams) != 2 {
		t.Errorf("Teams len = %d, want 2", len(m.Teams))
	}
	// Divergences h5 : pas de pair_mode, pas de T0.
	if m.PairMode != nil || m.T0Ms != nil {
		t.Errorf("PairMode/T0Ms devraient etre nil en h5")
	}
}

func TestAggregatePlayerStats_RealShape(t *testing.T) {
	var resp H5ServiceRecordResponse
	if err := json.Unmarshal([]byte(fixtureServiceRecord), &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	s := aggregatePlayerStats(&resp, "JGtm")
	if s == nil {
		t.Fatal("aggregatePlayerStats nil")
	}
	if s.Identity.Gamertag != "JGtm" || s.Identity.XUID != "" {
		t.Errorf("Identity gamertag-keyee attendue, got %+v", s.Identity)
	}
	if s.MatchesPlayed != 3 || s.Losses != 3 || s.Wins != 0 {
		t.Errorf("games = %d W%d L%d", s.MatchesPlayed, s.Wins, s.Losses)
	}
	if s.Kills != 20 || s.Deaths != 39 || s.Assists != 5 {
		t.Errorf("K/D/A = %d/%d/%d", s.Kills, s.Deaths, s.Assists)
	}
	if s.WinRate == nil || !approxEq(*s.WinRate, 0) {
		t.Errorf("WinRate = %v, want 0", s.WinRate)
	}
	if s.KDR == nil || !approxEq(*s.KDR, 20.0/39.0) {
		t.Errorf("KDR = %v, want %.4f", s.KDR, 20.0/39.0)
	}
	if s.KDA == nil || !approxEq(*s.KDA, (20.0+5.0/3.0)/39.0) {
		t.Errorf("KDA = %v", s.KDA)
	}
	if s.Accuracy == nil || !approxEq(*s.Accuracy, 35.0/90.0) {
		t.Errorf("Accuracy = %v, want %.4f", s.Accuracy, 35.0/90.0)
	}
}

func TestMapCareerSnapshot_CSRDesignation(t *testing.T) {
	var resp H5ServiceRecordResponse
	if err := json.Unmarshal([]byte(fixtureServiceRecord), &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	snap := mapCareerSnapshot(&resp, "JGtm")
	if snap == nil {
		t.Fatal("mapCareerSnapshot nil")
	}
	if snap.RankTier == nil || *snap.RankTier != "Diamant" {
		t.Errorf("RankTier = %v, want Diamant (DesignationId 4)", snap.RankTier)
	}
	if snap.RankName == nil || *snap.RankName != "Diamant 5" {
		t.Errorf("RankName = %v, want \"Diamant 5\" (Tier 5)", snap.RankName)
	}
	// Csr=0 (sous-Onyx) -> HighestCSR non expose (la valeur brute n'a de sens qu'a Onyx).
	if snap.HighestCSR != nil {
		t.Errorf("HighestCSR = %v, want nil (Csr brut 0 sous Onyx)", snap.HighestCSR)
	}
}

func TestDesignationLabels(t *testing.T) {
	en, fr := designationLabels(5)
	if en != "onyx" || fr != "Onyx" {
		t.Errorf("designation 5 = (%q,%q), want (onyx,Onyx)", en, fr)
	}
	if en, fr := designationLabels(99); en != "" || fr != "" {
		t.Errorf("designation hors borne = (%q,%q), want vides", en, fr)
	}
}

func ptrInt(n int) *int { return &n }
