package prestige

import (
	"encoding/json"
	"testing"
)

// Vérifie le marshaling/unmarshaling JSON et la validation pour chaque enum.
// Chaque test couvre : valeurs valides, valeurs invalides, roundtrip.

func TestChallengeStatus_RoundtripAndValidation(t *testing.T) {
	valid := []ChallengeStatus{
		StatusDraft, StatusActive, StatusCompleted, StatusExpired, StatusAbandoned, StatusArchived,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("ChallengeStatus %q should be valid", s)
		}
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshal %q: %v", s, err)
		}
		var got ChallengeStatus
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", s, err)
		}
		if got != s {
			t.Errorf("roundtrip mismatch: got %q want %q", got, s)
		}
	}

	// Invalide
	var invalid ChallengeStatus
	if err := json.Unmarshal([]byte(`"unknown_status"`), &invalid); err == nil {
		t.Error("expected error for invalid ChallengeStatus")
	}
}

func TestChallengeStatus_IsTerminal(t *testing.T) {
	cases := map[ChallengeStatus]bool{
		StatusDraft:     false,
		StatusActive:    false,
		StatusCompleted: true,
		StatusExpired:   true,
		StatusAbandoned: true,
		StatusArchived:  true,
	}
	for s, want := range cases {
		if got := s.IsTerminal(); got != want {
			t.Errorf("IsTerminal(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestTier_RoundtripAndValidation(t *testing.T) {
	valid := []Tier{TierNormal, TierHeroic, TierLegendary, TierMythic}
	for _, tier := range valid {
		if !tier.Valid() {
			t.Errorf("Tier %q should be valid", tier)
		}
		data, _ := json.Marshal(tier)
		var got Tier
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", tier, err)
		}
		if got != tier {
			t.Errorf("roundtrip mismatch: got %q want %q", got, tier)
		}
	}
	var invalid Tier
	if err := json.Unmarshal([]byte(`"diamond"`), &invalid); err == nil {
		t.Error("expected error for invalid Tier")
	}
}

func TestCadence_RoundtripAndValidation(t *testing.T) {
	valid := []Cadence{CadenceDaily, CadenceWeekly, CadenceMonthly, CadenceFree}
	for _, c := range valid {
		if !c.Valid() {
			t.Errorf("Cadence %q should be valid", c)
		}
		data, _ := json.Marshal(c)
		var got Cadence
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", c, err)
		}
		if got != c {
			t.Errorf("roundtrip mismatch: got %q want %q", got, c)
		}
	}
	var invalid Cadence
	if err := json.Unmarshal([]byte(`"yearly"`), &invalid); err == nil {
		t.Error("expected error for invalid Cadence")
	}
}

func TestEvalType_RoundtripAndValidation(t *testing.T) {
	valid := []EvalType{EvalThreshold, EvalCumulative}
	for _, e := range valid {
		if !e.Valid() {
			t.Errorf("EvalType %q should be valid", e)
		}
		data, _ := json.Marshal(e)
		var got EvalType
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", e, err)
		}
		if got != e {
			t.Errorf("roundtrip mismatch: got %q want %q", got, e)
		}
	}
	var invalid EvalType
	if err := json.Unmarshal([]byte(`"average"`), &invalid); err == nil {
		t.Error("expected error for invalid EvalType")
	}
}

func TestWindowType_RoundtripAndValidation(t *testing.T) {
	valid := []WindowType{WindowSession, WindowRollingDays, WindowDeadline, WindowMatchesInternal}
	for _, w := range valid {
		if !w.Valid() {
			t.Errorf("WindowType %q should be valid", w)
		}
		data, _ := json.Marshal(w)
		var got WindowType
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", w, err)
		}
		if got != w {
			t.Errorf("roundtrip mismatch: got %q want %q", got, w)
		}
	}
	var invalid WindowType
	if err := json.Unmarshal([]byte(`"forever"`), &invalid); err == nil {
		t.Error("expected error for invalid WindowType")
	}
}

func TestChallengeMode_RoundtripAndValidation(t *testing.T) {
	valid := []ChallengeMode{ModeLibre, ModePilote}
	for _, m := range valid {
		if !m.Valid() {
			t.Errorf("ChallengeMode %q should be valid", m)
		}
		data, _ := json.Marshal(m)
		var got ChallengeMode
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", m, err)
		}
		if got != m {
			t.Errorf("roundtrip mismatch: got %q want %q", got, m)
		}
	}
	var invalid ChallengeMode
	if err := json.Unmarshal([]byte(`"hybrid"`), &invalid); err == nil {
		t.Error("expected error for invalid ChallengeMode")
	}
}

func TestDataTier_RoundtripAndValidation(t *testing.T) {
	valid := []DataTier{DataFull, DataEstimated, DataTracking}
	for _, d := range valid {
		if !d.Valid() {
			t.Errorf("DataTier %q should be valid", d)
		}
		data, _ := json.Marshal(d)
		var got DataTier
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", d, err)
		}
		if got != d {
			t.Errorf("roundtrip mismatch: got %q want %q", got, d)
		}
	}
	var invalid DataTier
	if err := json.Unmarshal([]byte(`"partial"`), &invalid); err == nil {
		t.Error("expected error for invalid DataTier")
	}
}

func TestSquadMode_RoundtripAndValidation(t *testing.T) {
	valid := []SquadMode{SquadCollective, SquadCompetitive}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("SquadMode %q should be valid", s)
		}
		data, _ := json.Marshal(s)
		var got SquadMode
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %q: %v", s, err)
		}
		if got != s {
			t.Errorf("roundtrip mismatch: got %q want %q", got, s)
		}
	}
	var invalid SquadMode
	if err := json.Unmarshal([]byte(`"solo"`), &invalid); err == nil {
		t.Error("expected error for invalid SquadMode")
	}
}

func TestPalierColor(t *testing.T) {
	cases := map[Tier]string{
		TierNormal:    "#9CA3AF",
		TierHeroic:    "#3B82F6",
		TierLegendary: "#8B5CF6",
		TierMythic:    "#F59E0B",
	}
	for tier, want := range cases {
		if got := PalierColor(tier); got != want {
			t.Errorf("PalierColor(%q) = %q, want %q", tier, got, want)
		}
	}
	if got := PalierColor(Tier("invalid")); got != "" {
		t.Errorf("PalierColor(invalid) = %q, want empty", got)
	}
}

// TestIsValidChallengeSource verrouille le contrat de validation d'origine avant
// persistance (handler CreateChallenge) : seules "user"/"pilot_mode"/"coach" sont
// acceptées ; "unknown" (bucket d'agrégation, jamais écrit), la chaîne vide et
// toute valeur inconnue sont rejetées → repli "user" côté handler. Un bug qui
// ajouterait "unknown" au switch corromprait la ventilation historique du diag.
func TestIsValidChallengeSource(t *testing.T) {
	valid := []string{ChallengeSourceUser, ChallengeSourcePilotMode, ChallengeSourceCoach}
	for _, s := range valid {
		if !IsValidChallengeSource(s) {
			t.Errorf("IsValidChallengeSource(%q) = false, want true", s)
		}
	}
	invalid := []string{ChallengeSourceUnknown, "", "USER", "coach ", "admin", "match"}
	for _, s := range invalid {
		if IsValidChallengeSource(s) {
			t.Errorf("IsValidChallengeSource(%q) = true, want false", s)
		}
	}
}
