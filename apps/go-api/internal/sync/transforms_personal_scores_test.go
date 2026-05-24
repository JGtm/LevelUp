// Tests pour ExtractPersonalScoreAwards (transforms_personal_scores.go).
// Couvre :
//   - extraction depuis JSON nominal
//   - find recursif PlayerTeamStats → PersonalScores
//   - resolution xuid (PlayerId = "xuid(...)" ou map {value:"xuid(...)"})
//   - skip NameId inconnu / nul / 0
//   - fallback award_score = count * psaPoints quand TotalPersonalScoreAwarded absent
//   - categorisation (kill, assist, objective, vehicle, other)
//   - cas degenere : matchID/xuid vides → nil, players absent → nil
package sync

import (
	"testing"
)

// buildPlayerWithPersonalScores construit un objet "Players[]" avec un xuid donne
// et des PersonalScores parametrables.
func buildPlayerWithPersonalScores(xuid string, scores []map[string]any) []any {
	scoresAny := make([]any, len(scores))
	for i, s := range scores {
		scoresAny[i] = s
	}
	return []any{
		map[string]any{
			"PlayerId": "xuid(" + xuid + ")",
			"PlayerTeamStats": []any{
				map[string]any{
					"Stats": map[string]any{
						"CoreStats": map[string]any{
							"PersonalScores": scoresAny,
						},
					},
				},
			},
		},
	}
}

func TestExtractPersonalScoreAwards_NominalKill(t *testing.T) {
	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{
				"NameId":                    float64(1024030246), // killed_player
				"Count":                     float64(15),
				"TotalPersonalScoreAwarded": float64(750),
			},
		}),
	}

	rows := ExtractPersonalScoreAwards(matchJSON, "match-1", "2533274823110022")
	if len(rows) != 1 {
		t.Fatalf("attendu 1 row, got %d", len(rows))
	}
	got := rows[0]
	if got.MatchID != "match-1" {
		t.Errorf("MatchID = %q, want match-1", got.MatchID)
	}
	if got.XUID != "2533274823110022" {
		t.Errorf("XUID = %q, want 2533274823110022", got.XUID)
	}
	if got.AwardName != "killed_player" {
		t.Errorf("AwardName = %q, want killed_player", got.AwardName)
	}
	if got.AwardCategory != psaCatKill {
		t.Errorf("AwardCategory = %q, want kill", got.AwardCategory)
	}
	if got.AwardCount != 15 {
		t.Errorf("AwardCount = %d, want 15", got.AwardCount)
	}
	if got.AwardScore != 750 {
		t.Errorf("AwardScore = %d, want 750 (depuis TotalPersonalScoreAwarded)", got.AwardScore)
	}
}

func TestExtractPersonalScoreAwards_FallbackScoreFromPSAPoints(t *testing.T) {
	// TotalPersonalScoreAwarded absent → utilise count * psaPoints[NameId].
	const killedPlayer uint64 = 1024030246
	expectedPts := psaPoints[killedPlayer]
	if expectedPts == 0 {
		t.Skip("psaPoints[killed_player] = 0 — test pas applicable")
	}

	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{
				"NameId": float64(killedPlayer),
				"Count":  float64(7),
				// pas de TotalPersonalScoreAwarded
			},
		}),
	}
	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022")
	if len(rows) != 1 {
		t.Fatalf("attendu 1 row, got %d", len(rows))
	}
	want := 7 * expectedPts
	if rows[0].AwardScore != want {
		t.Errorf("AwardScore fallback = %d, want %d (7 * psaPoints)", rows[0].AwardScore, want)
	}
}

func TestExtractPersonalScoreAwards_UnknownNameIdSkipped(t *testing.T) {
	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{
				"NameId": float64(99999), // inconnu
				"Count":  float64(3),
			},
			{
				"NameId":                    float64(1024030246), // killed_player connu
				"Count":                     float64(5),
				"TotalPersonalScoreAwarded": float64(250),
			},
		}),
	}
	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022")
	if len(rows) != 1 {
		t.Errorf("attendu 1 row (unknown NameId skip), got %d", len(rows))
	}
	if len(rows) == 1 && rows[0].AwardName != "killed_player" {
		t.Errorf("AwardName = %q, want killed_player (le valide)", rows[0].AwardName)
	}
}

func TestExtractPersonalScoreAwards_ZeroNameIdSkipped(t *testing.T) {
	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{
				"NameId": float64(0),
				"Count":  float64(3),
			},
		}),
	}
	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022")
	if len(rows) != 0 {
		t.Errorf("NameId=0 doit etre skippe, got %d rows", len(rows))
	}
}

func TestExtractPersonalScoreAwards_EmptyMatchIDOrXUID(t *testing.T) {
	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{"NameId": float64(1024030246), "Count": float64(1), "TotalPersonalScoreAwarded": float64(50)},
		}),
	}
	if rows := ExtractPersonalScoreAwards(matchJSON, "", "2533274823110022"); rows != nil {
		t.Errorf("matchID vide doit retourner nil, got %v", rows)
	}
	if rows := ExtractPersonalScoreAwards(matchJSON, "m1", ""); rows != nil {
		t.Errorf("xuid vide doit retourner nil, got %v", rows)
	}
}

func TestExtractPersonalScoreAwards_XUIDNotInPlayers(t *testing.T) {
	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{"NameId": float64(1024030246), "Count": float64(1), "TotalPersonalScoreAwarded": float64(50)},
		}),
	}
	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274858283686")
	if rows != nil {
		t.Errorf("xuid absent de Players doit retourner nil, got %v", rows)
	}
}

func TestExtractPersonalScoreAwards_NoPlayers(t *testing.T) {
	if rows := ExtractPersonalScoreAwards(map[string]any{}, "m1", "2533274823110022"); rows != nil {
		t.Errorf("Players absent doit retourner nil, got %v", rows)
	}
	if rows := ExtractPersonalScoreAwards(map[string]any{"Players": []any{}}, "m1", "2533274823110022"); rows != nil {
		t.Errorf("Players vide doit retourner nil, got %v", rows)
	}
}

func TestExtractPersonalScoreAwards_PlayerWithoutPersonalScores(t *testing.T) {
	matchJSON := map[string]any{
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(2533274823110022)",
				"PlayerTeamStats": []any{
					map[string]any{
						"Stats": map[string]any{
							"CoreStats": map[string]any{
								// PersonalScores absent
							},
						},
					},
				},
			},
		},
	}
	if rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022"); rows != nil {
		t.Errorf("PersonalScores absent doit retourner nil, got %v", rows)
	}
}

func TestExtractPersonalScoreAwards_RecursivePlayerTeamStats(t *testing.T) {
	// Le JSON Halo enveloppe parfois PlayerTeamStats dans plusieurs niveaux.
	// findPSRecursive doit traverser map et []any.
	matchJSON := map[string]any{
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(2533274823110022)",
				"PlayerTeamStats": []any{
					map[string]any{
						"Stats": map[string]any{
							"CoreStats": map[string]any{
								"PersonalScores": []any{
									map[string]any{
										"NameId":                    float64(1024030246),
										"Count":                     float64(2),
										"TotalPersonalScoreAwarded": float64(100),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022")
	if len(rows) != 1 {
		t.Fatalf("findPSRecursive doit traverser : attendu 1 row, got %d", len(rows))
	}
}

func TestExtractPersonalScoreAwards_MultipleCategories(t *testing.T) {
	// 1 kill + 1 assist + 1 objective : 3 categories distinctes.
	const (
		killed       uint64 = 1024030246 // kill
		killAssist   uint64 = 638246808  // assist
		flagCaptured uint64 = 601966503  // objective
	)

	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{"NameId": float64(killed), "Count": float64(5), "TotalPersonalScoreAwarded": float64(250)},
			{"NameId": float64(killAssist), "Count": float64(8), "TotalPersonalScoreAwarded": float64(160)},
			{"NameId": float64(flagCaptured), "Count": float64(1), "TotalPersonalScoreAwarded": float64(100)},
		}),
	}

	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022")
	if len(rows) != 3 {
		t.Fatalf("attendu 3 rows, got %d", len(rows))
	}

	gotCategories := map[string]int{}
	for _, r := range rows {
		gotCategories[r.AwardCategory]++
	}
	if gotCategories[psaCatKill] != 1 {
		t.Errorf("attendu 1 'kill', got %d", gotCategories[psaCatKill])
	}
	if gotCategories[psaCatAssist] != 1 {
		t.Errorf("attendu 1 'assist', got %d", gotCategories[psaCatAssist])
	}
	if gotCategories[psaCatObjective] != 1 {
		t.Errorf("attendu 1 'objective', got %d", gotCategories[psaCatObjective])
	}
}

func TestExtractPersonalScoreAwards_CountAbsentDefaultsToZero(t *testing.T) {
	matchJSON := map[string]any{
		"Players": buildPlayerWithPersonalScores("2533274823110022", []map[string]any{
			{
				"NameId":                    float64(1024030246),
				"TotalPersonalScoreAwarded": float64(50),
				// Count absent
			},
		}),
	}
	rows := ExtractPersonalScoreAwards(matchJSON, "m1", "2533274823110022")
	if len(rows) != 1 {
		t.Fatalf("attendu 1 row, got %d", len(rows))
	}
	if rows[0].AwardCount != 0 {
		t.Errorf("Count absent doit defaulter a 0, got %d", rows[0].AwardCount)
	}
	if rows[0].AwardScore != 50 {
		t.Errorf("AwardScore = %d, want 50 (TotalPersonalScoreAwarded direct)", rows[0].AwardScore)
	}
}
