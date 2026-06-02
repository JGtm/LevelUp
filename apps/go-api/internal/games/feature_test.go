package games

import (
	"testing"

	"levelup/go-api/internal/domain/feature"
)

func TestComputeFeatureMatrix_Cascade(t *testing.T) {
	caps := CapabilityMap{
		CapMatchHistory:       CapSupported,
		CapMatchDetailCore:    CapSupported,
		CapScoreboardExtra:    CapNotExposed, // enrichissement absent → match_detail degraded
		CapMatchSkillSnapshot: CapDegraded,   // primaire degraded → skill_rating degraded
		CapCareerProgression:  CapSupported,
		CapPveFirefight:       CapSupported,
		CapTimeseries:         CapNotExposed, // primaire not_exposed → timeseries unavailable
		CapCitationsEngine:    CapNotExposed,
		CapEngagement:         CapSupported,
	}

	m := ComputeFeatureMatrix(caps)

	want := map[feature.Key]feature.Status{
		feature.KeyMatchHistory: feature.StatusAvailable,
		feature.KeyMatchDetail:  feature.StatusDegraded, // core supported + scoreboard.extra absent
		feature.KeySkillRating:  feature.StatusDegraded, // primaire degraded
		feature.KeyCareer:       feature.StatusAvailable,
		feature.KeyPveStats:     feature.StatusAvailable,
		feature.KeyTimeseries:   feature.StatusUnavailable, // primaire not_exposed
		feature.KeyCitations:    feature.StatusUnavailable,
		feature.KeyEngagement:   feature.StatusAvailable,
	}
	if len(m) != len(want) {
		t.Fatalf("matrice = %d features, want %d", len(m), len(want))
	}
	for k, w := range want {
		if got := m.Get(k); got != w {
			t.Errorf("feature %q = %q, want %q", k, got, w)
		}
	}
}

// TestComputeFeatureMatrix_MissingCapsGracefulDegradation : un titre qui ne
// déclare AUCUNE capability → toutes les features unavailable (pas de panic, pas
// de feature exposée à tort).
func TestComputeFeatureMatrix_MissingCapsGracefulDegradation(t *testing.T) {
	m := ComputeFeatureMatrix(CapabilityMap{})
	for _, k := range AllFeatureKeys() {
		if got := m.Get(k); got != feature.StatusUnavailable {
			t.Errorf("feature %q sans capability = %q, want unavailable", k, got)
		}
	}
}

// TestComputeFeatureMatrix_EnhancementSupported : si l'enrichissement passe
// supported, match_detail devient available.
func TestComputeFeatureMatrix_EnhancementSupported(t *testing.T) {
	caps := CapabilityMap{
		CapMatchDetailCore: CapSupported,
		CapScoreboardExtra: CapSupported,
	}
	if got := ComputeFeatureMatrix(caps).Get(feature.KeyMatchDetail); got != feature.StatusAvailable {
		t.Errorf("match_detail (core+extra supported) = %q, want available", got)
	}
}

// TestComputeFeatureMatrix_EnhancementDegraded : un enrichissement degraded
// (≠ not_exposed) dégrade aussi la feature quand la primaire est supported —
// la cascade traite tout enrichissement non-supported pareil.
func TestComputeFeatureMatrix_EnhancementDegraded(t *testing.T) {
	caps := CapabilityMap{
		CapMatchDetailCore: CapSupported,
		CapScoreboardExtra: CapDegraded,
	}
	if got := ComputeFeatureMatrix(caps).Get(feature.KeyMatchDetail); got != feature.StatusDegraded {
		t.Errorf("match_detail (core supported + extra degraded) = %q, want degraded", got)
	}
}

func TestAllFeatureKeys_Sorted(t *testing.T) {
	keys := AllFeatureKeys()
	if len(keys) != 8 {
		t.Errorf("AllFeatureKeys() = %d, want 8", len(keys))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Errorf("AllFeatureKeys() non trié à %d: %q >= %q", i, keys[i-1], keys[i])
		}
	}
}
