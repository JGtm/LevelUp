package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

func mkSharedMatchHistory(matchID string, started time.Time, mainOutcome canonical.Outcome) domain.SquadSharedMatch {
	return domain.SquadSharedMatch{
		MatchID:   matchID,
		StartedAt: started,
		Outcome:   mainOutcome,
	}
}

func mkPMRowHistory(
	gt, matchID string,
	started time.Time,
	durationSec int,
	kills, deaths, assists int,
	mapLabel, modeLabel string,
	outcome canonical.Outcome,
) canonical.PlayerMatchRow {
	d := durationSec
	k, dth, a := kills, deaths, assists
	row := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:         matchID,
			StartedAtUTC:    started,
			DurationSeconds: &d,
		},
		Self: canonical.MatchParticipant{
			Identity: canonical.PlayerIdentity{Gamertag: gt},
			Kills:    &k,
			Deaths:   &dth,
			Assists:  &a,
			Outcome:  outcome,
		},
	}
	if mapLabel != "" {
		row.Summary.Map = &canonical.AssetReference{ID: "map_" + mapLabel, DefaultLabel: mapLabel}
	}
	if modeLabel != "" {
		row.Summary.GameVariant = &canonical.AssetReference{ID: "mode_" + modeLabel, DefaultLabel: modeLabel}
	}
	return row
}

func TestBuildHistoryTable_SortsByDateDesc(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	t3 := t1.Add(2 * time.Hour)
	shared := []domain.SquadSharedMatch{
		mkSharedMatchHistory("m1", t1, canonical.OutcomeWin),
		mkSharedMatchHistory("m2", t3, canonical.OutcomeLoss),
		mkSharedMatchHistory("m3", t2, canonical.OutcomeTie),
	}
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkPMRowHistory("main", "m1", t1, 600, 10, 5, 0, "Aquarius", "Slayer", canonical.OutcomeWin),
			mkPMRowHistory("main", "m2", t3, 600, 8, 7, 0, "Recharge", "CTF", canonical.OutcomeLoss),
			mkPMRowHistory("main", "m3", t2, 600, 12, 9, 0, "Live Fire", "Strongholds", canonical.OutcomeTie),
		},
	}
	rows := BuildHistoryTable(shared, rowsByPlayer, []string{"main"}, i18n.Default)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	// Tri date desc : m2 (t3) > m3 (t2) > m1 (t1)
	if rows[0].MatchID != "m2" || rows[1].MatchID != "m3" || rows[2].MatchID != "m1" {
		t.Errorf("date desc order want [m2,m3,m1], got [%s,%s,%s]",
			rows[0].MatchID, rows[1].MatchID, rows[2].MatchID)
	}
}

func TestBuildHistoryTable_HydratesMapModeFromMain(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	shared := []domain.SquadSharedMatch{mkSharedMatchHistory("m1", t0, canonical.OutcomeWin)}
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowHistory("main", "m1", t0, 600, 10, 5, 3, "Aquarius", "Slayer", canonical.OutcomeWin)},
	}
	rows := BuildHistoryTable(shared, rowsByPlayer, []string{"main"}, i18n.Default)
	if rows[0].MapLabel != "Aquarius" {
		t.Errorf("MapLabel want Aquarius, got %s", rows[0].MapLabel)
	}
	if rows[0].ModeLabel != "Slayer" {
		t.Errorf("ModeLabel want Slayer, got %s", rows[0].ModeLabel)
	}
	if rows[0].DurationSeconds == nil || *rows[0].DurationSeconds != 600 {
		t.Errorf("Duration want 600, got %v", rows[0].DurationSeconds)
	}
}

func TestBuildHistoryTable_PlayerCellsByGamertag(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	shared := []domain.SquadSharedMatch{mkSharedMatchHistory("m1", t0, canonical.OutcomeWin)}
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowHistory("main", "m1", t0, 600, 10, 5, 3, "Aquarius", "Slayer", canonical.OutcomeWin)},
		"f1":   {mkPMRowHistory("f1", "m1", t0, 600, 5, 8, 2, "Aquarius", "Slayer", canonical.OutcomeWin)},
	}
	rows := BuildHistoryTable(shared, rowsByPlayer, []string{"main", "f1"}, i18n.Default)
	cells := rows[0].PlayerStats
	if len(cells) != 2 {
		t.Fatalf("want 2 player cells, got %d", len(cells))
	}
	mainCell := cells["main"]
	if mainCell.Kills == nil || *mainCell.Kills != 10 {
		t.Errorf("main kills want 10, got %v", mainCell.Kills)
	}
	if mainCell.Outcome != "win" {
		t.Errorf("main outcome want win, got %s", mainCell.Outcome)
	}
	f1Cell := cells["f1"]
	if f1Cell.Deaths == nil || *f1Cell.Deaths != 8 {
		t.Errorf("f1 deaths want 8, got %v", f1Cell.Deaths)
	}
}

func TestBuildHistoryTable_PlayerAbsentFromMatchSkipsCell(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	shared := []domain.SquadSharedMatch{mkSharedMatchHistory("m1", t0, canonical.OutcomeWin)}
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowHistory("main", "m1", t0, 600, 10, 5, 3, "", "", canonical.OutcomeWin)},
		// f1 n'a pas le match m1
	}
	rows := BuildHistoryTable(shared, rowsByPlayer, []string{"main", "f1"}, i18n.Default)
	if _, hasMain := rows[0].PlayerStats["main"]; !hasMain {
		t.Errorf("main expected in PlayerStats")
	}
	if _, hasF1 := rows[0].PlayerStats["f1"]; hasF1 {
		t.Errorf("f1 should NOT be in PlayerStats (no row m1)")
	}
}

func TestBuildHistoryTable_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := BuildHistoryTable(nil, nil, []string{"main"}, i18n.Default); got != nil {
		t.Errorf("nil shared: want nil, got %v", got)
	}
}

// Les libellés de carte, de mode et de sélection suivent la langue de la requête : le même match
// donne le nom français sous fr et le nom anglais sous en ; une langue sans nom retombe sur la langue
// de repli.
func TestBuildHistoryTable_LibellesDansLaLangueDeLaRequete(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	shared := []domain.SquadSharedMatch{mkSharedMatchHistory("m1", t0, canonical.OutcomeWin)}
	row := mkPMRowHistory("main", "m1", t0, 600, 10, 5, 3, "Aquarius", "Slayer", canonical.OutcomeWin)
	row.Summary.Map.Labels = map[string]string{"fr": "Aquarium", "en": "Aquarius"}
	row.Summary.GameVariant.Labels = map[string]string{"fr": "Assassin", "en": "Slayer"}
	row.Summary.Playlist = &canonical.AssetReference{ID: "pl", DefaultLabel: "Quick Play", Labels: map[string]string{"fr": "Partie rapide"}}
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{"main": {row}}

	fr := BuildHistoryTable(shared, rowsByPlayer, []string{"main"}, i18n.Locale("fr"))[0]
	if fr.MapLabel != "Aquarium" || fr.ModeLabel != "Assassin" || fr.PlaylistLabel != "Partie rapide" {
		t.Errorf("fr : carte %q, mode %q, sélection %q", fr.MapLabel, fr.ModeLabel, fr.PlaylistLabel)
	}
	en := BuildHistoryTable(shared, rowsByPlayer, []string{"main"}, i18n.Locale("en"))[0]
	if en.MapLabel != "Aquarius" || en.ModeLabel != "Slayer" || en.PlaylistLabel != "Quick Play" {
		t.Errorf("en : carte %q, mode %q, sélection %q (la sélection sans nom en retombe sur le nom par défaut)", en.MapLabel, en.ModeLabel, en.PlaylistLabel)
	}
}
