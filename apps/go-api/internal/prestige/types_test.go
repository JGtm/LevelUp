package prestige

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// Vérifie le roundtrip JSON sur chaque struct persistée.
// Si un champ est cassé (mauvais tag, type incompatible), le roundtrip diverge.

func TestChallenge_JSONRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	committed := now.Add(time.Minute)
	completed := now.Add(time.Hour)
	original := Challenge{
		ID:                    "ch_001",
		UserID:                "user_42",
		TitleSlug:             "halo_infinite",
		ArcID:                 "arc_slayer",
		Position:              2,
		TemplateID:            "halo_infinite.weekly.kda_3sessions",
		Metric:                "FieldKDA",
		Target:                1.5,
		TargetPerMember:       0,
		WindowType:            WindowSession,
		WindowValue:           "3",
		Cadence:               CadenceWeekly,
		EvalType:              EvalThreshold,
		Mode:                  ModeLibre,
		Tier:                  TierHeroic,
		DataTier:              DataFull,
		Label:                 "Slayer Lv.2",
		Status:                StatusCompleted,
		CreatedAt:             now,
		CommittedAt:           &committed,
		CompletedAt:           &completed,
		LastPalierRecomputeAt: &committed,
		IsPrivate:             false,
	}
	roundtripJSON(t, original, &Challenge{})
}

func TestArc_JSONRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	completed := now.Add(time.Hour)
	original := Arc{
		ID:          "arc_001",
		UserID:      "user_42",
		TitleSlug:   "halo_infinite",
		Title:       "Le Slayer",
		Description: "Devenir la force offensive",
		IsPreset:    true,
		PresetID:    "halo_infinite.slayer",
		CreatedAt:   now,
		CompletedAt: &completed,
	}
	roundtripJSON(t, original, &Arc{})
}

func TestMomentCard_JSONRoundtrip(t *testing.T) {
	original := MomentCard{
		ID:          "mc_001",
		ChallengeID: "ch_001",
		BlobPath:    "/blobs/2026/04/mc_001.png",
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &MomentCard{})
}

func TestPrestigeEvent_JSONRoundtrip(t *testing.T) {
	original := PrestigeEvent{
		ID:         "pe_001",
		UserID:     "user_42",
		TitleSlug:  "halo_infinite",
		SourceType: SourceChallenge,
		SourceID:   "ch_001",
		PPAmount:   75,
		Tier:       TierHeroic,
		CreatedAt:  time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &PrestigeEvent{})
}

func TestUserPrestige_JSONRoundtrip(t *testing.T) {
	original := UserPrestige{
		UserID:       "user_42",
		TitleSlug:    "halo_infinite",
		TotalPP:      1820,
		CurrentLevel: 2,
		UpdatedAt:    time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &UserPrestige{})
}

func TestTemplate_JSONRoundtrip(t *testing.T) {
	original := Template{
		ID:              "halo_infinite.daily.kda_session",
		TitleSlug:       "halo_infinite",
		Metric:          "FieldKDA",
		WindowType:      WindowSession,
		WindowValue:     "1",
		Cadence:         CadenceDaily,
		EvalType:        EvalThreshold,
		ModeFilter:      "universal",
		LabelEN:         "Stay sharp",
		LabelFR:         "Reste affûté",
		NormalTarget:    1.10,
		HeroicTarget:    1.35,
		LegendaryTarget: 1.60,
		MythicTarget:    2.00,
		SchemaVersion:   1,
		UpdatedAt:       time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &Template{})
}

func TestPresetArc_JSONRoundtrip(t *testing.T) {
	original := PresetArc{
		ID:            "halo_infinite.slayer",
		TitleSlug:     "halo_infinite",
		TitleEN:       "The Slayer",
		TitleFR:       "Le Slayer",
		DescriptionEN: "Become the force",
		DescriptionFR: "Deviens la force",
		SchemaVersion: 1,
		UpdatedAt:     time.Now().UTC().Truncate(time.Second),
		Steps: []PresetArcStep{
			{PresetArcID: "halo_infinite.slayer", Position: 1, TemplateID: "tpl1", TargetTier: TierNormal},
			{PresetArcID: "halo_infinite.slayer", Position: 2, TemplateID: "tpl2", TargetTier: TierHeroic},
		},
	}
	roundtripJSON(t, original, &PresetArc{})
}

func TestSquadChallenge_JSONRoundtrip(t *testing.T) {
	expires := time.Now().UTC().Add(7 * 24 * time.Hour).Truncate(time.Second)
	original := SquadChallenge{
		ID:              "sc_001",
		SquadID:         "squad_001",
		TemplateID:      "halo_infinite.weekly.wins",
		TitleSlug:       "halo_infinite",
		Mode:            SquadCollective,
		EvalType:        EvalCumulative,
		WindowType:      WindowDeadline,
		WindowValue:     "2026-05-01",
		TargetPerMember: 10.0,
		ExpiresAt:       &expires,
		CreatedBy:       "user_42",
		CreatedAt:       time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &SquadChallenge{})
}

func TestSquadChallengeParticipant_JSONRoundtrip(t *testing.T) {
	completed := time.Now().UTC().Truncate(time.Second)
	original := SquadChallengeParticipant{
		SquadChallengeID: "sc_001",
		UserID:           "user_42",
		ChosenTier:       TierLegendary,
		DataTier:         DataFull,
		CurrentValue:     12.0,
		CompletedAt:      &completed,
		IsPrivate:        false,
		JoinedAt:         time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &SquadChallengeParticipant{})
}

func TestPrestigeTelemetry_JSONRoundtrip(t *testing.T) {
	original := PrestigeTelemetry{
		ID:                     "pt_001",
		UserID:                 "user_42",
		ChallengeID:            "ch_001",
		EventType:              TelemetryCompleted,
		Palier:                 TierHeroic,
		StretchRatio:           1.42,
		BaselineValue:          1.18,
		Mode:                   ModeLibre,
		Cadence:                CadenceWeekly,
		EvalType:               EvalThreshold,
		TimeSinceCreateSeconds: 86400,
		CreatedAt:              time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &PrestigeTelemetry{})
}

func TestBaselineState_JSONRoundtrip(t *testing.T) {
	last := time.Now().UTC().Add(-30 * 24 * time.Hour).Truncate(time.Second)
	original := BaselineState{
		UserID:                   "user_42",
		TitleSlug:                "halo_infinite",
		Metric:                   "FieldKDA",
		LastMatchAt:              &last,
		IsStale:                  false,
		RecoveryMatchesRemaining: 0,
		UpdatedAt:                time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &BaselineState{})
}

func TestBaseline_JSONRoundtrip(t *testing.T) {
	original := Baseline{
		UserID:     "user_42",
		TitleSlug:  "halo_infinite",
		Metric:     "FieldKDA",
		Value:      1.18,
		MatchCount: 20,
		DataTier:   DataFull,
		ComputedAt: time.Now().UTC().Truncate(time.Second),
	}
	roundtripJSON(t, original, &Baseline{})
}

func TestLevel_JSONRoundtrip(t *testing.T) {
	original := Level{
		Index:           2,
		Name:            "Vétéran",
		ThresholdPP:     1500,
		NextThresholdPP: 3000,
		ProgressRatio:   0.213,
	}
	roundtripJSON(t, original, &Level{})
}

// roundtripJSON marshals original then unmarshals into target and compares
// via reflect.DeepEqual (handles pointers, slices, time.Time correctly).
func roundtripJSON[T any](t *testing.T, original T, target *T) {
	t.Helper()
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal: %v\njson: %s", err, data)
	}
	if !reflect.DeepEqual(original, *target) {
		t.Errorf("roundtrip mismatch:\nwant: %+v\n got: %+v\njson: %s", original, *target, data)
	}
}
