package sync

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
)

// TestComputeMatchT0 couvre le wiring sync : extraction des ParticipationInfo
// depuis matchJSON + calcul T0 via timeline.ComputeT0. Les bots (xuid
// non-numérique) doivent être exclus et ne pas polluer le T0.
func TestComputeMatchT0(t *testing.T) {
	start := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	matchJSON := map[string]any{
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(2533274823110022)",
				"ParticipationInfo": map[string]any{
					"PresentAtBeginning": true,
					"FirstJoinedTime":    start.Add(28 * time.Second).Format(time.RFC3339),
				},
			},
			map[string]any{
				"PlayerId": "xuid(2535469190789936)",
				"ParticipationInfo": map[string]any{
					"PresentAtBeginning": true,
					"FirstJoinedTime":    start.Add(28200 * time.Millisecond).Format(time.RFC3339Nano),
				},
			},
			// Bot : present mais xuid non-numérique → exclu (sinon polluerait le spread).
			map[string]any{
				"PlayerId": "bid(1.0)",
				"ParticipationInfo": map[string]any{
					"PresentAtBeginning": true,
					"FirstJoinedTime":    start.Add(90 * time.Second).Format(time.RFC3339),
				},
			},
			// Latecomer : present_at_beginning=false → ignoré.
			map[string]any{
				"PlayerId": "xuid(2535443797720345)",
				"ParticipationInfo": map[string]any{
					"PresentAtBeginning": false,
					"FirstJoinedTime":    start.Add(200 * time.Second).Format(time.RFC3339),
				},
			},
		},
	}

	t0, q := computeMatchT0(matchJSON, start)
	if q != timeline.T0QualityOK {
		t.Errorf("quality = %s, want ok", q)
	}
	if t0 < 27000 || t0 > 29000 {
		t.Errorf("t0 = %dms, want ~28000 (bot/latecomer exclus)", t0)
	}
}

// TestComputeMatchT0_NoParticipation : sans ParticipationInfo, T0 no_data.
func TestComputeMatchT0_NoParticipation(t *testing.T) {
	start := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	matchJSON := map[string]any{
		"Players": []any{
			map[string]any{"PlayerId": "xuid(2533274823110022)"},
		},
	}
	_, q := computeMatchT0(matchJSON, start)
	if q.Computed() {
		t.Errorf("quality = %s, want non-computed (no_data)", q)
	}
}
