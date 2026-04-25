package mappings

import (
	"math"
	"testing"
)

func TestFormatValueInteger(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in  any
		out string
	}{
		{int(42), "42"},
		{int64(0), "0"},
		{float64(3.7), "3"},
		{int(-5), "-5"},
	}
	for _, tc := range cases {
		got, err := FormatValue(tc.in, FormatInteger)
		if err != nil {
			t.Errorf("FormatValue(%v, integer) err: %v", tc.in, err)
		}
		if got != tc.out {
			t.Errorf("FormatValue(%v, integer) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

func TestFormatValueSignedInt(t *testing.T) {
	t.Parallel()
	got, _ := FormatValue(15, FormatSignedInt)
	if got != "+15" {
		t.Errorf("signed_int(15) = %q, want +15", got)
	}
	got, _ = FormatValue(-3, FormatSignedInt)
	if got != "-3" {
		t.Errorf("signed_int(-3) = %q, want -3", got)
	}
	got, _ = FormatValue(0, FormatSignedInt)
	if got != "+0" {
		t.Errorf("signed_int(0) = %q, want +0", got)
	}
}

func TestFormatValuePercent(t *testing.T) {
	t.Parallel()
	got, _ := FormatValue(42.5, FormatPercent1)
	if got != "42.5%" {
		t.Errorf("percent_1(42.5) = %q", got)
	}
	got, _ = FormatValue(0.123, FormatPercent2)
	if got != "0.12%" {
		t.Errorf("percent_2(0.123) = %q", got)
	}
}

func TestFormatValueKDR(t *testing.T) {
	t.Parallel()
	got, _ := FormatValue(1.234, FormatKDR2)
	if got != "1.23" {
		t.Errorf("kdr_2(1.234) = %q", got)
	}
}

func TestFormatValueDurationHMS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		secs any
		want string
	}{
		{0, "0s"},
		{45, "45s"},
		{60, "1m00s"},
		{125, "2m05s"},
		{3661, "1h01m01s"},
	}
	for _, tc := range cases {
		got, err := FormatValue(tc.secs, FormatDurationHMS)
		if err != nil {
			t.Errorf("duration_hms(%v) err: %v", tc.secs, err)
		}
		if got != tc.want {
			t.Errorf("duration_hms(%v) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

func TestFormatValueNilAndCorrupted(t *testing.T) {
	t.Parallel()
	if got, _ := FormatValue(nil, FormatInteger); got != "" {
		t.Errorf("nil → %q, want empty", got)
	}
	if got, _ := FormatValue(math.NaN(), FormatPercent1); got != "" {
		t.Errorf("NaN → %q, want empty", got)
	}
	if got, _ := FormatValue(math.Inf(1), FormatKDR2); got != "" {
		t.Errorf("Inf → %q, want empty", got)
	}
}

func TestFormatValueBoolean(t *testing.T) {
	t.Parallel()
	if got, _ := FormatValue(true, FormatBoolean); got != "true" {
		t.Errorf("true → %q", got)
	}
	if got, _ := FormatValue(false, FormatBoolean); got != "false" {
		t.Errorf("false → %q", got)
	}
}

func TestFormatValueUnknownFormat(t *testing.T) {
	t.Parallel()
	_, err := FormatValue(42, Format("not_a_format"))
	if err == nil {
		t.Errorf("expected error for unknown format")
	}
}

func TestConvertValue(t *testing.T) {
	t.Parallel()
	// Identité
	out, ok := ConvertValue(0.5, UnitRatio, UnitRatio)
	if !ok || out != 0.5 {
		t.Errorf("identité ratio = (%f, %v)", out, ok)
	}

	// ratio → percent
	out, ok = ConvertValue(0.42, UnitRatio, UnitPercent)
	if !ok || math.Abs(out-42.0) > 1e-9 {
		t.Errorf("ratio→percent(0.42) = (%f, %v)", out, ok)
	}

	// percent → ratio
	out, ok = ConvertValue(75, UnitPercent, UnitRatio)
	if !ok || math.Abs(out-0.75) > 1e-9 {
		t.Errorf("percent→ratio(75) = (%f, %v)", out, ok)
	}

	// ms → s
	out, ok = ConvertValue(1500, UnitMilliseconds, UnitSeconds)
	if !ok || math.Abs(out-1.5) > 1e-9 {
		t.Errorf("ms→s(1500) = (%f, %v)", out, ok)
	}

	// s → ms
	out, ok = ConvertValue(2.5, UnitSeconds, UnitMilliseconds)
	if !ok || math.Abs(out-2500.0) > 1e-9 {
		t.Errorf("s→ms(2.5) = (%f, %v)", out, ok)
	}

	// non supporté
	if _, ok = ConvertValue(1, UnitCount, UnitSeconds); ok {
		t.Errorf("count→seconds devrait être non supporté")
	}
}

// FuzzFormatValue garantit qu'aucune entrée ne fait paniquer le formatter.
func FuzzFormatValue(f *testing.F) {
	f.Add(int64(0), "integer")
	f.Add(int64(-1), "signed_int")
	f.Add(int64(1234567890), "duration_hms")
	f.Add(int64(0), "percent_1")

	f.Fuzz(func(t *testing.T, n int64, formatStr string) {
		_, _ = FormatValue(n, Format(formatStr))
	})
}

// FuzzLoadFieldsFromBytes garantit que le loader ne panique pas sur des TOML
// arbitraires : il retourne une erreur typée ou un set valide.
func FuzzLoadFieldsFromBytes(f *testing.F) {
	f.Add([]byte(minimalValidTOML))
	f.Add([]byte(""))
	f.Add([]byte("[meta]\n"))
	f.Add([]byte("not toml at all"))

	f.Fuzz(func(t *testing.T, raw []byte) {
		_, _ = LoadFieldsFromBytes("fuzz.toml", raw)
	})
}

// TestFormatRoundTripPercent vérifie que percent → ratio → percent garde
// la valeur d'entrée à 1e-9 près (property-based léger).
func TestFormatRoundTripPercent(t *testing.T) {
	t.Parallel()
	values := []float64{0, 0.5, 1, 25, 50, 75, 99.99, 100}
	for _, v := range values {
		ratio, ok := ConvertValue(v, UnitPercent, UnitRatio)
		if !ok {
			t.Errorf("percent→ratio(%v) ok=false", v)
		}
		back, ok := ConvertValue(ratio, UnitRatio, UnitPercent)
		if !ok {
			t.Errorf("ratio→percent ok=false")
		}
		if math.Abs(back-v) > 1e-9 {
			t.Errorf("round-trip percent(%v) → ratio → percent = %v", v, back)
		}
	}
}
