package teammates

// teammates_squad_intensity_team_lobby_test.go — les deux courbes de référence
// du profil d'intensité (lot 1, 2026-09-18) : `team` ne compte que les frags des
// alliés du joueur principal (main inclus), `lobby` compte tous les frags du
// match (les deux camps). Avant ce lot, la seule ligne agrégée (`all`) était en
// réalité le lobby, affichée sous le libellé « Équipe ».

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
)

const (
	tlMainXUID  = "x_main"
	tlAllyXUID  = "x_ally"
	tlEnemyA    = "x_enemy_a"
	tlEnemyB    = "x_enemy_b"
	tlDuration  = int64(1_000_000) // max(time_ms) du match m1 → dénominateur
	tlEarlyKill = int64(100_000)   // bucket 1
	tlLateKill  = int64(900_000)   // bucket 9
)

// tlFixture : 3 matchs (seuil intensityMinMatches), events sur m1 seulement :
// main + allié tuent tôt (bucket 1), les deux adversaires tuent tard (bucket 9,
// trois frags dont un à la durée exacte, borné au dernier bucket).
func tlFixture() (*mockSquadRepo, []domain.SquadMatchRow) {
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	repo := &mockSquadRepo{
		impactRows: []domain.ImpactEventRow{
			{MatchID: "m1", XUID: tlMainXUID, EventType: highlightevent.EventTypeKill, TimeMS: tlEarlyKill},
			{MatchID: "m1", XUID: tlAllyXUID, EventType: highlightevent.EventTypeKill, TimeMS: tlEarlyKill},
			{MatchID: "m1", XUID: tlEnemyA, EventType: highlightevent.EventTypeKill, TimeMS: tlLateKill},
			{MatchID: "m1", XUID: tlEnemyB, EventType: highlightevent.EventTypeKill, TimeMS: tlLateKill},
			{MatchID: "m1", XUID: tlEnemyB, EventType: highlightevent.EventTypeKill, TimeMS: tlDuration},
		},
	}
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: base, DurationSeconds: 1000},
		{MatchID: "m2", StartTime: base.Add(time.Hour), DurationSeconds: 1000},
		{MatchID: "m3", StartTime: base.Add(2 * time.Hour), DurationSeconds: 1000},
	}
	return repo, rows
}

func tlRowFor(t *testing.T, profile *domain.SquadIntensityProfile, key, matchID string) domain.SquadIntensityMatchRow {
	t.Helper()
	if profile == nil {
		t.Fatal("profil nil : au moins la ligne lobby porte un signal")
	}
	for _, r := range profile.Rows[key] {
		if r.MatchID == matchID {
			return r
		}
	}
	t.Fatalf("ligne %q introuvable pour %q (clés : %v)", matchID, key, profile.Options)
	return domain.SquadIntensityMatchRow{}
}

// La courbe `team` exclut les adversaires ; la courbe `lobby` les inclut. Aucune
// clé `all` ne subsiste dans le payload.
func TestBuildSquadIntensityProfile_TeamExcludesEnemies_LobbyIncludesThem(t *testing.T) {
	repo, rows := tlFixture()
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	mainTeam := map[string]map[string]struct{}{
		"m1": {tlMainXUID: {}, tlAllyXUID: {}},
	}

	got := svc.buildSquadIntensityProfile(context.Background(), rows, "main", nil, mainTeam)

	team := tlRowFor(t, got, domain.SquadIntensityKeyTeam, "m1")
	lobby := tlRowFor(t, got, domain.SquadIntensityKeyLobby, "m1")
	// Équipe : 2 frags alliés au bucket 1, rien au bucket 9 (adversaires exclus).
	if team.Phases[1] != 1 || team.Phases[9] != 0 {
		t.Errorf("team : want phases[1]=1 phases[9]=0 (adversaires exclus), got %v", team.Phases)
	}
	// Lobby : 3 frags adverses au bucket 9 (max) et 2 alliés au bucket 1 → 0.67.
	if lobby.Phases[9] != 1 || lobby.Phases[1] != 0.67 {
		t.Errorf("lobby : want phases[9]=1 phases[1]=0.67 (tout le match), got %v", lobby.Phases)
	}
	keys := map[string]bool{}
	for _, o := range got.Options {
		keys[o.Key] = true
	}
	if !keys[domain.SquadIntensityKeyTeam] || !keys[domain.SquadIntensityKeyLobby] || keys["all"] {
		t.Errorf("options : want team + lobby, sans `all`, got %v", got.Options)
	}
}

// Alliés non résolus (chargement Q32b en échec → map nil) : la ligne `team` est
// à phases nulles sur chaque match, la ligne `lobby` reste intacte.
func TestBuildSquadIntensityProfile_NoAllies_TeamEmpty_LobbyIntact(t *testing.T) {
	repo, rows := tlFixture()
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}

	got := svc.buildSquadIntensityProfile(context.Background(), rows, "main", nil, nil)

	team := tlRowFor(t, got, domain.SquadIntensityKeyTeam, "m1")
	if team.Phases != [intensityBuckets]float64{} {
		t.Errorf("team sans alliés résolus : want phases nulles, got %v", team.Phases)
	}
	lobby := tlRowFor(t, got, domain.SquadIntensityKeyLobby, "m1")
	if lobby.Phases[9] != 1 || lobby.Phases[1] != 0.67 {
		t.Errorf("lobby intact attendu, got %v", lobby.Phases)
	}
}
