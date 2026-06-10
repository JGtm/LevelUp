package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// pick retourne le bucket (season, playlist) ou échoue le test s'il est absent.
func pick(t *testing.T, buckets []domain.WorldPlayerSeasonStats, season, playlist string) domain.WorldPlayerSeasonStats {
	t.Helper()
	for _, b := range buckets {
		if b.SeasonID == season && b.PlaylistID == playlist {
			return b
		}
	}
	t.Fatalf("bucket (%s, %s) introuvable dans %+v", season, playlist, buckets)
	return domain.WorldPlayerSeasonStats{}
}

func TestNormalizeSeasonID(t *testing.T) {
	cases := map[string]string{
		"Csr/Seasons/CsrSeason13-2.json": "csrseason13-2",
		"Csr/Seasons/CsrSeason12-1.json": "csrseason12-1",
		"Seasons/Season6.json":           "season6", // hors-CSR → ne matchera aucun snapshot
		"CsrSeason13-2":                  "csrseason13-2",
		"":                               "",
	}
	for in, want := range cases {
		if got := NormalizeSeasonID(in); got != want {
			t.Errorf("NormalizeSeasonID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractPlayerMatchStat(t *testing.T) {
	match := map[string]any{
		"MatchInfo": map[string]any{
			"SeasonId": "Csr/Seasons/CsrSeason13-2.json",
			"Playlist": map[string]any{"AssetId": "edfef3ac-9cbe-4fa2-b949-8f29deafd483"},
		},
		"Players": []any{
			map[string]any{"PlayerId": "xuid(999)"}, // autre joueur
			map[string]any{
				"PlayerId": "xuid(2533274895653213)",
				"Outcome":  float64(2), // Win
				"PlayerTeamStats": []any{
					map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
						"Kills": float64(15), "Deaths": float64(8), "Assists": float64(4),
					}}},
				},
				"ParticipationInfo": map[string]any{"TimePlayed": "PT10M39.203S"},
			},
		},
	}

	st, ok := ExtractPlayerMatchStat(match, "2533274895653213")
	if !ok {
		t.Fatal("ExtractPlayerMatchStat: joueur introuvable alors qu'il est présent")
	}
	if st.SeasonID != "csrseason13-2" {
		t.Errorf("SeasonID = %q, want csrseason13-2", st.SeasonID)
	}
	if st.PlaylistID != "edfef3ac-9cbe-4fa2-b949-8f29deafd483" {
		t.Errorf("PlaylistID = %q", st.PlaylistID)
	}
	if st.Outcome != 2 {
		t.Errorf("Outcome = %d, want 2", st.Outcome)
	}
	if st.Kills != 15 || st.Deaths != 8 || st.Assists != 4 {
		t.Errorf("K/D/A = %d/%d/%d, want 15/8/4", st.Kills, st.Deaths, st.Assists)
	}
	if st.PlaytimeSec < 639.2 || st.PlaytimeSec > 639.21 {
		t.Errorf("PlaytimeSec = %v, want ~639.203 (10*60+39.203)", st.PlaytimeSec)
	}

	if _, ok := ExtractPlayerMatchStat(match, "111111"); ok {
		t.Error("ExtractPlayerMatchStat: xuid absent devrait retourner ok=false")
	}
}

func TestAccumulateWorldStats(t *testing.T) {
	const arena = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"
	const slayer = "dcb2e24e-05fb-4390-8076-32a0cdb4326e"
	stats := []PlayerMatchStat{
		{SeasonID: "csrseason13-2", PlaylistID: arena, Outcome: 2, Kills: 15, Deaths: 8, Assists: 4, PlaytimeSec: 600},
		{SeasonID: "csrseason13-2", PlaylistID: arena, Outcome: 3, Kills: 10, Deaths: 12, Assists: 2, PlaytimeSec: 600},
		{SeasonID: "csrseason13-2", PlaylistID: arena, Outcome: 1, Kills: 5, Deaths: 5, Assists: 1, PlaytimeSec: 300},
		// Autre playlist (ne doit pas fusionner avec Arena).
		{SeasonID: "csrseason13-2", PlaylistID: slayer, Outcome: 2, Kills: 20, Deaths: 10, Assists: 5},
		// Autre saison.
		{SeasonID: "csrseason12-1", PlaylistID: arena, Outcome: 4, Kills: 1, Deaths: 1},
		// Ignorés : saison/playlist vides.
		{SeasonID: "", PlaylistID: arena, Outcome: 2, Kills: 99},
		{SeasonID: "season6", PlaylistID: "", Outcome: 2, Kills: 99},
	}

	out := AccumulateWorldStats("Alpha", stats)
	if len(out) != 3 {
		t.Fatalf("attendu 3 buckets (13-2/arena, 13-2/slayer, 12-1/arena), got %d : %+v", len(out), out)
	}

	byKey := map[string]struct{}{}
	for _, b := range out {
		byKey[b.SeasonID+"|"+b.PlaylistID] = struct{}{}
		if b.Gamertag != "Alpha" || b.TitleSlug != "halo_infinite" {
			t.Errorf("bucket meta inattendu: %+v", b)
		}
	}

	arenaCur := pick(t, out, "csrseason13-2", arena)
	if arenaCur.MatchCount != 3 {
		t.Errorf("13-2/arena MatchCount = %d, want 3", arenaCur.MatchCount)
	}
	if arenaCur.WinCount != 1 || arenaCur.LossCount != 1 || arenaCur.TieCount != 1 || arenaCur.DnfCount != 0 {
		t.Errorf("13-2/arena W/L/T/D = %d/%d/%d/%d, want 1/1/1/0",
			arenaCur.WinCount, arenaCur.LossCount, arenaCur.TieCount, arenaCur.DnfCount)
	}
	if arenaCur.Kills != 30 || arenaCur.Deaths != 25 || arenaCur.Assists != 7 {
		t.Errorf("13-2/arena K/D/A = %d/%d/%d, want 30/25/7", arenaCur.Kills, arenaCur.Deaths, arenaCur.Assists)
	}
	if arenaCur.PlaytimeSec != 1500 {
		t.Errorf("13-2/arena playtime = %d, want 1500", arenaCur.PlaytimeSec)
	}

	prevArena := pick(t, out, "csrseason12-1", arena)
	if prevArena.DnfCount != 1 || prevArena.MatchCount != 1 {
		t.Errorf("12-1/arena = %+v, want 1 match / 1 DNF", prevArena)
	}
}
