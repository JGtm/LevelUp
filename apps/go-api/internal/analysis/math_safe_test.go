package analysis

import (
	"encoding/json"
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// IsBadFloat
// ─────────────────────────────────────────────────────────────────────────────

func TestIsBadFloat(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want bool
	}{
		{"zero", 0.0, false},
		{"positive", 1.5, false},
		{"negative", -42.3, false},
		{"max_float64", math.MaxFloat64, false},
		{"smallest_nonzero_float64", math.SmallestNonzeroFloat64, false},
		{"NaN", math.NaN(), true},
		{"positive_infinity", math.Inf(1), true},
		{"negative_infinity", math.Inf(-1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsBadFloat(tc.in)
			if got != tc.want {
				t.Errorf("IsBadFloat(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SanitizeFloat (return 0 if bad)
// ─────────────────────────────────────────────────────────────────────────────

func TestSanitizeFloat_NormalValuesPassThrough(t *testing.T) {
	cases := []float64{0.0, 1.0, -1.0, 1.5, -42.3, math.MaxFloat64, math.SmallestNonzeroFloat64}
	for _, in := range cases {
		got := SanitizeFloat(in)
		if got != in {
			t.Errorf("SanitizeFloat(%v) = %v, want %v (pass-through)", in, got, in)
		}
	}
}

func TestSanitizeFloat_NaNBecomesZero(t *testing.T) {
	got := SanitizeFloat(math.NaN())
	if got != 0.0 {
		t.Errorf("SanitizeFloat(NaN) = %v, want 0.0", got)
	}
}

func TestSanitizeFloat_PositiveInfinityBecomesZero(t *testing.T) {
	got := SanitizeFloat(math.Inf(1))
	if got != 0.0 {
		t.Errorf("SanitizeFloat(+Inf) = %v, want 0.0", got)
	}
}

func TestSanitizeFloat_NegativeInfinityBecomesZero(t *testing.T) {
	got := SanitizeFloat(math.Inf(-1))
	if got != 0.0 {
		t.Errorf("SanitizeFloat(-Inf) = %v, want 0.0", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SanitizeNullableFloat (return nil if bad)
// ─────────────────────────────────────────────────────────────────────────────

func TestSanitizeNullableFloat_NilPassThrough(t *testing.T) {
	got := SanitizeNullableFloat(nil)
	if got != nil {
		t.Errorf("SanitizeNullableFloat(nil) = %v, want nil", got)
	}
}

func TestSanitizeNullableFloat_NormalPointerPreserved(t *testing.T) {
	v := 1.5
	got := SanitizeNullableFloat(&v)
	if got == nil {
		t.Fatal("SanitizeNullableFloat(&1.5) = nil, want &1.5")
	}
	if *got != 1.5 {
		t.Errorf("SanitizeNullableFloat(&1.5) deref = %v, want 1.5", *got)
	}
}

func TestSanitizeNullableFloat_NaNBecomesNil(t *testing.T) {
	v := math.NaN()
	got := SanitizeNullableFloat(&v)
	if got != nil {
		t.Errorf("SanitizeNullableFloat(&NaN) = %v, want nil", got)
	}
}

func TestSanitizeNullableFloat_InfBecomesNil(t *testing.T) {
	v := math.Inf(1)
	got := SanitizeNullableFloat(&v)
	if got != nil {
		t.Errorf("SanitizeNullableFloat(&+Inf) = %v, want nil", got)
	}
	w := math.Inf(-1)
	got2 := SanitizeNullableFloat(&w)
	if got2 != nil {
		t.Errorf("SanitizeNullableFloat(&-Inf) = %v, want nil", got2)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SafeRatio (calcule safe avec epsilon)
// ─────────────────────────────────────────────────────────────────────────────

func TestSafeRatio_NormalDivision(t *testing.T) {
	got := SafeRatio(10.0, 4.0)
	if got != 2.5 {
		t.Errorf("SafeRatio(10, 4) = %v, want 2.5", got)
	}
}

func TestSafeRatio_ZeroDenominator(t *testing.T) {
	got := SafeRatio(5.0, 0.0)
	if got != 0.0 {
		t.Errorf("SafeRatio(5, 0) = %v, want 0", got)
	}
}

func TestSafeRatio_NegligibleDenominator(t *testing.T) {
	got := SafeRatio(5.0, 1e-13) // < epsilon (1e-12)
	if got != 0.0 {
		t.Errorf("SafeRatio(5, 1e-13) = %v, want 0 (below epsilon)", got)
	}
}

func TestSafeRatio_NaNNumerator(t *testing.T) {
	got := SafeRatio(math.NaN(), 1.0)
	if got != 0.0 {
		t.Errorf("SafeRatio(NaN, 1) = %v, want 0 (NaN propagated)", got)
	}
}

func TestSafeRatio_HaloKDR_DeathsZero(t *testing.T) {
	// Cas reel : un joueur avec 5 kills et 0 deaths. KDR avant fix = +Inf.
	// Apres SafeRatio : 0.0 (semantique : pas de mesure significative).
	kdr := SafeRatio(5.0, 0.0)
	if IsBadFloat(kdr) {
		t.Errorf("KDR(5/0) = %v est encore bad, doit etre safe", kdr)
	}
}

func TestSafeRatio_HaloAccuracy_ShotsZero(t *testing.T) {
	// Cas reel : tir 0 / hits 0 → accuracy = NaN avant fix.
	acc := SafeRatio(0.0, 0.0)
	if acc != 0.0 {
		t.Errorf("accuracy(0/0) = %v, want 0", acc)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// json.Marshal compat (le but ultime de SanitizeFloat)
// ─────────────────────────────────────────────────────────────────────────────

func TestSanitizeFloat_JSONMarshalSafe(t *testing.T) {
	type sample struct {
		KDA      float64  `json:"kda"`
		Accuracy float64  `json:"accuracy"`
		KDR      *float64 `json:"kdr,omitempty"`
	}

	// Avant sanitize : KDA=NaN, accuracy=+Inf → json.Marshal ECHOUE.
	bad := sample{
		KDA:      math.NaN(),
		Accuracy: math.Inf(1),
	}
	_, err := json.Marshal(bad)
	if err == nil {
		t.Skip("Go a change le comportement json.Marshal(NaN) — sanitize plus necessaire ? Investigate.")
	}

	// Apres sanitize : tous les bad floats devenu 0 → Marshal OK.
	kdrVal := math.NaN()
	good := sample{
		KDA:      SanitizeFloat(bad.KDA),
		Accuracy: SanitizeFloat(bad.Accuracy),
		KDR:      SanitizeNullableFloat(&kdrVal),
	}
	raw, err := json.Marshal(good)
	if err != nil {
		t.Fatalf("Marshal apres sanitize doit reussir, got err: %v", err)
	}
	// KDR doit etre absent (nil + omitempty).
	if string(raw) != `{"kda":0,"accuracy":0}` {
		t.Errorf("JSON marshal post-sanitize = %s, want {\"kda\":0,\"accuracy\":0}", raw)
	}
}
