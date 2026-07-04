// Package sync — transforms_extract_test.go : tests unitaires des extracteurs JSON.
//
// Couvre ExtractRegistry, ExtractParticipants, ExtractMedals (tous à 0%).
// Utilise des fixtures JSON statiques — aucune dépendance DB.
package sync

import (
	"testing"
)

// ── Fixtures ────────────────────────────────────────────────────────────────

// minimalMatchJSON retourne un JSON de match minimal valide.
func minimalMatchJSON() map[string]any {
	return map[string]any{
		"MatchId": "match-uuid-001",
		"MatchInfo": map[string]any{
			"StartTime":        "2025-03-15T18:30:00Z",
			"EndTime":          "2025-03-15T18:45:00Z",
			"Duration":         "PT15M",
			"PlayableDuration": "PT14M30S",
			"Playlist": map[string]any{
				"AssetId":    "playlist-uuid-001",
				"VersionId":  "playlist-ver-001",
				"PublicName": "Quick Play",
			},
			"MapVariant": map[string]any{
				"AssetId":    "map-uuid-001",
				"VersionId":  "map-ver-001",
				"PublicName": "Bazaar",
			},
			"PlaylistMapModePair": map[string]any{
				"AssetId":    "pair-uuid-001",
				"VersionId":  "pair-ver-001",
				"PublicName": "Quick Play: Slayer on Bazaar",
			},
			"UgcGameVariant": map[string]any{
				"AssetId":    "gv-uuid-001",
				"VersionId":  "gv-ver-001",
				"PublicName": "Slayer",
			},
		},
		"Players": []any{
			map[string]any{
				"PlayerId":   "xuid(1234567890)",
				"Gamertag":   "TestPlayer",
				"LastTeamId": float64(0),
				"Outcome":    float64(2),
				"Rank":       float64(1),
				"PlayerTeamStats": []any{
					map[string]any{
						"Stats": map[string]any{
							"CoreStats": map[string]any{
								"Kills":         float64(15),
								"Deaths":        float64(8),
								"Assists":       float64(5),
								"KDA":           float64(8.67), // API natif (15 + 5/3 − 8)
								"Accuracy":      float64(40.0), // API natif (80/200)
								"PersonalScore": float64(2500),
								"ShotsFired":    float64(200),
								"ShotsHit":      float64(80),
								"DamageDealt":   float64(3500.0),
								"DamageTaken":   float64(2800.0),
								"Medals": []any{
									map[string]any{"NameId": float64(622331684), "Count": float64(3)},
									map[string]any{"NameId": float64(2780740777), "Count": float64(1)},
								},
							},
						},
					},
				},
			},
			map[string]any{
				"PlayerId":   "xuid(9876543210)",
				"Gamertag":   "OpponentPlayer",
				"LastTeamId": float64(1),
				"Outcome":    float64(3),
				"Rank":       float64(2),
				"PlayerTeamStats": []any{
					map[string]any{
						"Stats": map[string]any{
							"CoreStats": map[string]any{
								"Kills":   float64(8),
								"Deaths":  float64(15),
								"Assists": float64(3),
								"KDA":     float64(-6.0), "Accuracy": float64(33.33),
								"PersonalScore": float64(1800),
								"ShotsFired":    float64(180),
								"ShotsHit":      float64(60),
								"DamageDealt":   float64(2800.0),
								"DamageTaken":   float64(3500.0),
								"Medals": []any{
									map[string]any{"NameId": float64(622331684), "Count": float64(1)},
								},
							},
						},
					},
				},
			},
		},
		"Teams": []any{
			map[string]any{
				"TeamId": float64(0),
				"Stats": map[string]any{
					"CoreStats": map[string]any{"Score": float64(50)},
				},
			},
			map[string]any{
				"TeamId": float64(1),
				"Stats": map[string]any{
					"CoreStats": map[string]any{"Score": float64(43)},
				},
			},
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ExtractRegistry
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractRegistry_Valid(t *testing.T) {
	row, err := ExtractRegistry(minimalMatchJSON(), "TestSyncer")
	if err != nil {
		t.Fatalf("ExtractRegistry error: %v", err)
	}
	if row.MatchID != "match-uuid-001" {
		t.Errorf("MatchID = %q", row.MatchID)
	}
	if row.StartTime.Year() != 2025 || row.StartTime.Month() != 3 || row.StartTime.Day() != 15 {
		t.Errorf("StartTime = %v", row.StartTime)
	}
	if row.FirstSyncBy != "TestSyncer" {
		t.Errorf("FirstSyncBy = %q", row.FirstSyncBy)
	}
	if row.PlaylistName == nil || *row.PlaylistName != "Quick Play" {
		t.Errorf("PlaylistName = %v", row.PlaylistName)
	}
	if row.MapName == nil || *row.MapName != "Bazaar" {
		t.Errorf("MapName = %v", row.MapName)
	}
	if row.DurationSeconds == nil || *row.DurationSeconds != 900 {
		t.Errorf("DurationSeconds = %v", row.DurationSeconds)
	}
	if row.PlayableDurationSeconds == nil || *row.PlayableDurationSeconds != 870 {
		t.Errorf("PlayableDurationSeconds = %v", row.PlayableDurationSeconds)
	}
	if row.EndTime == nil {
		t.Error("EndTime should not be nil")
	}
	// real_start_time vient désormais de ComputeT0 (first_joined_time des joueurs
	// présents au début), plus de playable_duration. Le fixture minimal n'a pas
	// de ParticipationInfo → T0 no_data → RealStartTime nil. Le calcul T0 lui-même
	// est couvert par compute_t0_test.go ; le wiring par TestComputeMatchT0.
	if row.RealStartTime != nil {
		t.Errorf("RealStartTime should be nil without ParticipationInfo (T0 derives from first_joined), got %v", row.RealStartTime)
	}
	if row.Team0Score == nil || *row.Team0Score != 50 {
		t.Errorf("Team0Score = %v", row.Team0Score)
	}
	if row.Team1Score == nil || *row.Team1Score != 43 {
		t.Errorf("Team1Score = %v", row.Team1Score)
	}
	if row.ModeCategory != "Other" {
		t.Errorf("ModeCategory = %q (expected Other for non-ranked slayer)", row.ModeCategory)
	}
	if row.IsRanked {
		t.Error("should not be ranked")
	}
	if row.IsFirefight {
		t.Error("should not be firefight")
	}
}

// Phase B du plan catalogue : extraction version_id depuis MatchInfo.
func TestExtractRegistry_VersionIDs(t *testing.T) {
	row, err := ExtractRegistry(minimalMatchJSON(), "TestSyncer")
	if err != nil {
		t.Fatalf("ExtractRegistry error: %v", err)
	}
	if row.PlaylistVersionID == nil || *row.PlaylistVersionID != "playlist-ver-001" {
		t.Errorf("PlaylistVersionID = %v", row.PlaylistVersionID)
	}
	if row.MapVersionID == nil || *row.MapVersionID != "map-ver-001" {
		t.Errorf("MapVersionID = %v", row.MapVersionID)
	}
	if row.PairVersionID == nil || *row.PairVersionID != "pair-ver-001" {
		t.Errorf("PairVersionID = %v", row.PairVersionID)
	}
	if row.GameVariantVersionID == nil || *row.GameVariantVersionID != "gv-ver-001" {
		t.Errorf("GameVariantVersionID = %v", row.GameVariantVersionID)
	}
}

// Phase B : VersionId absent du JSON → pointeur nil (NULL en DB), pas d'erreur.
func TestExtractRegistry_VersionIDsAbsent(t *testing.T) {
	j := map[string]any{
		"MatchId": "match-no-versions",
		"MatchInfo": map[string]any{
			"StartTime": "2025-03-15T18:30:00Z",
			"Playlist":  map[string]any{"AssetId": "p"},
			// pas de VersionId
		},
	}
	row, err := ExtractRegistry(j, "x")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	// strPtrNonEmpty("") retourne nil — convention pour insérer NULL en DB.
	if row.PlaylistVersionID != nil {
		t.Errorf("PlaylistVersionID attendu nil, got %v", *row.PlaylistVersionID)
	}
	if row.PairVersionID != nil {
		t.Errorf("PairVersionID attendu nil, got %v", *row.PairVersionID)
	}
}

func TestExtractRegistry_MissingMatchID(t *testing.T) {
	j := map[string]any{"MatchInfo": map[string]any{}}
	_, err := ExtractRegistry(j, "x")
	if err == nil {
		t.Error("expected error for missing MatchId")
	}
}

func TestExtractRegistry_MissingMatchInfo(t *testing.T) {
	j := map[string]any{"MatchId": "abc"}
	_, err := ExtractRegistry(j, "x")
	if err == nil {
		t.Error("expected error for missing MatchInfo")
	}
}

func TestExtractRegistry_MissingStartTime(t *testing.T) {
	j := map[string]any{
		"MatchId":   "abc",
		"MatchInfo": map[string]any{},
	}
	_, err := ExtractRegistry(j, "x")
	if err == nil {
		t.Error("expected error for missing StartTime")
	}
}

func TestExtractRegistry_RankedMatch(t *testing.T) {
	j := minimalMatchJSON()
	info := j["MatchInfo"].(map[string]any)
	info["Playlist"] = map[string]any{
		"AssetId":    "ranked-uuid",
		"PublicName": "Ranked Arena",
	}
	row, err := ExtractRegistry(j, "x")
	if err != nil {
		t.Fatal(err)
	}
	if !row.IsRanked {
		t.Error("should be ranked")
	}
}

func TestExtractRegistry_FirefightMatch(t *testing.T) {
	j := minimalMatchJSON()
	info := j["MatchInfo"].(map[string]any)
	info["GameVariantCategory"] = float64(22)
	row, err := ExtractRegistry(j, "x")
	if err != nil {
		t.Fatal(err)
	}
	if !row.IsFirefight {
		t.Error("should be firefight")
	}
}

func TestExtractRegistry_EndTimeFallback(t *testing.T) {
	j := minimalMatchJSON()
	info := j["MatchInfo"].(map[string]any)
	delete(info, "EndTime")
	row, err := ExtractRegistry(j, "x")
	if err != nil {
		t.Fatal(err)
	}
	if row.EndTime == nil {
		t.Error("EndTime should fallback to start+duration")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ExtractParticipants
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractParticipants_Valid(t *testing.T) {
	rows := ExtractParticipants(minimalMatchJSON())
	if len(rows) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(rows))
	}

	p1 := rows[0]
	if p1.XUID != "1234567890" {
		t.Errorf("player1 XUID = %q", p1.XUID)
	}
	if p1.Gamertag == nil || *p1.Gamertag != "TestPlayer" {
		t.Errorf("player1 Gamertag = %v", p1.Gamertag)
	}
	if p1.Kills == nil || *p1.Kills != 15 {
		t.Errorf("player1 Kills = %v", p1.Kills)
	}
	if p1.Deaths == nil || *p1.Deaths != 8 {
		t.Errorf("player1 Deaths = %v", p1.Deaths)
	}
	// KDA + Accuracy lus tels quels depuis l'API (CoreStats), plus aucun calcul.
	if p1.KDA == nil || *p1.KDA < 8.6 || *p1.KDA > 8.7 {
		t.Errorf("player1 KDA = %v, expected ~8.67 (API natif)", p1.KDA)
	}
	if p1.Accuracy == nil || *p1.Accuracy != 40.0 {
		t.Errorf("player1 Accuracy = %v, expected 40.0 (API natif)", p1.Accuracy)
	}
	if p1.TeamID == nil || *p1.TeamID != 0 {
		t.Errorf("player1 TeamID = %v", p1.TeamID)
	}
	if p1.Outcome == nil || *p1.Outcome != 2 {
		t.Errorf("player1 Outcome = %v", p1.Outcome)
	}
}

func TestExtractParticipants_ParticipationInfoBooleans(t *testing.T) {
	j := map[string]any{
		"MatchId": "abc",
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(1)", "PlayerType": float64(1),
				"ParticipationInfo": map[string]any{
					"TimePlayed":          "PT10M",
					"PresentAtBeginning":  true,
					"PresentAtCompletion": false,
					"JoinedInProgress":    false,
					"LeftInProgress":      true,
				},
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": float64(0),
					"Stats":  map[string]any{"CoreStats": map[string]any{}},
				}},
			},
			// Player without ParticipationInfo — all 4 booleans must remain nil.
			map[string]any{
				"PlayerId": "xuid(2)", "PlayerType": float64(1),
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": float64(0),
					"Stats":  map[string]any{"CoreStats": map[string]any{}},
				}},
			},
		},
	}
	rows := ExtractParticipants(j)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	p1 := rows[0]
	if p1.PresentAtBeginning == nil || !*p1.PresentAtBeginning {
		t.Errorf("p1 PresentAtBeginning = %v, want true", p1.PresentAtBeginning)
	}
	if p1.PresentAtCompletion == nil || *p1.PresentAtCompletion {
		t.Errorf("p1 PresentAtCompletion = %v, want false", p1.PresentAtCompletion)
	}
	if p1.JoinedInProgress == nil || *p1.JoinedInProgress {
		t.Errorf("p1 JoinedInProgress = %v, want false", p1.JoinedInProgress)
	}
	if p1.LeftInProgress == nil || !*p1.LeftInProgress {
		t.Errorf("p1 LeftInProgress = %v, want true", p1.LeftInProgress)
	}
	p2 := rows[1]
	if p2.PresentAtBeginning != nil || p2.PresentAtCompletion != nil ||
		p2.JoinedInProgress != nil || p2.LeftInProgress != nil {
		t.Errorf("p2 ParticipationInfo booleans should all be nil (no ParticipationInfo block)")
	}
}

// TestExtractParticipants_ParticipationInfoTimestamps vérifie le parsing des
// 2 timestamps absolus (LUSR v2 Phase 3-quit 2026-05-27) :
//   - FirstJoinedTime : toujours présent dans le payload API
//   - LastLeaveTime   : null si joueur encore présent à la fin
func TestExtractParticipants_ParticipationInfoTimestamps(t *testing.T) {
	j := map[string]any{
		"MatchId": "abc",
		"Players": []any{
			// p1 : quitter avec timestamp de départ.
			map[string]any{
				"PlayerId": "xuid(1)", "PlayerType": float64(1),
				"ParticipationInfo": map[string]any{
					"FirstJoinedTime":     "2025-03-15T18:30:00Z",
					"LastLeaveTime":       "2025-03-15T18:38:42Z",
					"PresentAtBeginning":  true,
					"PresentAtCompletion": false,
					"LeftInProgress":      true,
				},
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": float64(0),
					"Stats":  map[string]any{"CoreStats": map[string]any{}},
				}},
			},
			// p2 : joueur présent jusqu'à la fin → LastLeaveTime absent du JSON.
			map[string]any{
				"PlayerId": "xuid(2)", "PlayerType": float64(1),
				"ParticipationInfo": map[string]any{
					"FirstJoinedTime":     "2025-03-15T18:30:00Z",
					"PresentAtBeginning":  true,
					"PresentAtCompletion": true,
				},
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": float64(0),
					"Stats":  map[string]any{"CoreStats": map[string]any{}},
				}},
			},
			// p3 : pas de ParticipationInfo du tout → les 2 timestamps doivent rester nil.
			map[string]any{
				"PlayerId": "xuid(3)", "PlayerType": float64(1),
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": float64(0),
					"Stats":  map[string]any{"CoreStats": map[string]any{}},
				}},
			},
		},
	}
	rows := ExtractParticipants(j)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	p1 := rows[0]
	if p1.FirstJoinedTime == nil {
		t.Errorf("p1 FirstJoinedTime = nil, want set")
	} else if got := p1.FirstJoinedTime.UTC().Format("15:04:05"); got != "18:30:00" {
		t.Errorf("p1 FirstJoinedTime UTC time = %q, want 18:30:00", got)
	}
	if p1.LastLeaveTime == nil {
		t.Errorf("p1 LastLeaveTime = nil, want set (quitter)")
	} else if got := p1.LastLeaveTime.UTC().Format("15:04:05"); got != "18:38:42" {
		t.Errorf("p1 LastLeaveTime UTC time = %q, want 18:38:42", got)
	}
	p2 := rows[1]
	if p2.FirstJoinedTime == nil {
		t.Errorf("p2 FirstJoinedTime = nil, want set")
	}
	if p2.LastLeaveTime != nil {
		t.Errorf("p2 LastLeaveTime = %v, want nil (joueur présent à la fin)", p2.LastLeaveTime)
	}
	p3 := rows[2]
	if p3.FirstJoinedTime != nil || p3.LastLeaveTime != nil {
		t.Errorf("p3 timestamps doivent rester nil (pas de ParticipationInfo block)")
	}
}

func TestExtractParticipants_MissingMatchID(t *testing.T) {
	j := map[string]any{"Players": []any{}}
	if rows := ExtractParticipants(j); rows != nil {
		t.Error("expected nil when MatchId missing")
	}
}

func TestExtractParticipants_NoPlayers(t *testing.T) {
	j := map[string]any{"MatchId": "abc"}
	rows := ExtractParticipants(j)
	if len(rows) != 0 {
		t.Errorf("expected 0 participants, got %d", len(rows))
	}
}

func TestExtractParticipants_DuplicateXUID(t *testing.T) {
	j := map[string]any{
		"MatchId": "abc",
		"Players": []any{
			map[string]any{"PlayerId": "xuid(111)", "Gamertag": "P1", "PlayerTeamStats": []any{}},
			map[string]any{"PlayerId": "xuid(111)", "Gamertag": "P1dup", "PlayerTeamStats": []any{}},
		},
	}
	rows := ExtractParticipants(j)
	if len(rows) != 1 {
		t.Errorf("duplicate XUID should be deduplicated, got %d", len(rows))
	}
}

func TestExtractParticipants_InvalidPlayerID(t *testing.T) {
	j := map[string]any{
		"MatchId": "abc",
		"Players": []any{
			map[string]any{"PlayerId": "not-xuid", "PlayerTeamStats": []any{}},
		},
	}
	rows := ExtractParticipants(j)
	if len(rows) != 0 {
		t.Error("invalid PlayerId should be skipped")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ExtractMedals
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractMedals_Valid(t *testing.T) {
	rows := ExtractMedals(minimalMatchJSON())
	if len(rows) != 3 {
		t.Fatalf("expected 3 medal rows (2 for P1, 1 for P2), got %d", len(rows))
	}

	// Verify first player's medals
	found := 0
	for _, r := range rows {
		if r.XUID == "1234567890" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("player1 should have 2 medals, got %d", found)
	}
}

func TestExtractMedals_MissingMatchID(t *testing.T) {
	j := map[string]any{"Players": []any{}}
	if rows := ExtractMedals(j); rows != nil {
		t.Error("expected nil when MatchId missing")
	}
}

func TestExtractMedals_ZeroCount(t *testing.T) {
	j := map[string]any{
		"MatchId": "abc",
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(111)",
				"PlayerTeamStats": []any{
					map[string]any{
						"Stats": map[string]any{
							"CoreStats": map[string]any{
								"Medals": []any{
									map[string]any{"NameId": float64(123), "Count": float64(0)},
								},
							},
						},
					},
				},
			},
		},
	}
	rows := ExtractMedals(j)
	if len(rows) != 0 {
		t.Error("zero-count medals should be skipped")
	}
}

func TestExtractMedals_NoPlayerTeamStats(t *testing.T) {
	j := map[string]any{
		"MatchId": "abc",
		"Players": []any{
			map[string]any{
				"PlayerId":        "xuid(111)",
				"PlayerTeamStats": []any{},
			},
		},
	}
	rows := ExtractMedals(j)
	if len(rows) != 0 {
		t.Errorf("expected 0 medals without CoreStats, got %d", len(rows))
	}
}
