// Package mappings — units_test.go : tests pour IsKnownUnit et helpers de
// conversion d'unités (audit #4 round 2).
package mappings

import "testing"

func TestIsKnownUnit_AllValidUnits(t *testing.T) {
	t.Parallel()
	cases := []Unit{
		UnitCount,
		UnitRatio,
		UnitPercent,
		UnitSeconds,
		UnitMilliseconds,
		UnitNoneUnit,
	}
	for _, u := range cases {
		if !IsKnownUnit(u) {
			t.Errorf("IsKnownUnit(%q) = false, want true", u)
		}
	}
}

func TestIsKnownUnit_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	unknowns := []Unit{
		"unknown",
		"meters",
		"frames",
		"COUNT", // case-sensitive
		"hours",
	}
	for _, u := range unknowns {
		if IsKnownUnit(u) {
			t.Errorf("IsKnownUnit(%q) = true, want false", u)
		}
	}
}

func TestConvertValue_Identity(t *testing.T) {
	t.Parallel()
	// Identité pour chaque unité connue.
	for _, u := range []Unit{UnitCount, UnitRatio, UnitPercent, UnitSeconds, UnitMilliseconds} {
		out, ok := ConvertValue(123.4, u, u)
		if !ok || out != 123.4 {
			t.Errorf("identity %s: (%v, %v), want (123.4, true)", u, out, ok)
		}
	}
}

func TestConvertValue_Unsupported(t *testing.T) {
	t.Parallel()
	// Conversions non implémentées.
	cases := []struct {
		from, to Unit
	}{
		{UnitCount, UnitRatio},
		{UnitRatio, UnitSeconds},
		{UnitSeconds, UnitPercent},
		{UnitMilliseconds, UnitRatio},
	}
	for _, c := range cases {
		_, ok := ConvertValue(1, c.from, c.to)
		if ok {
			t.Errorf("ConvertValue(%q→%q) should be unsupported", c.from, c.to)
		}
	}
}

func TestConvertValue_Zero(t *testing.T) {
	t.Parallel()
	// 0 doit passer toutes les conversions.
	out, ok := ConvertValue(0, UnitRatio, UnitPercent)
	if !ok || out != 0 {
		t.Errorf("0 ratio→percent: (%v, %v)", out, ok)
	}
	out, ok = ConvertValue(0, UnitMilliseconds, UnitSeconds)
	if !ok || out != 0 {
		t.Errorf("0 ms→s: (%v, %v)", out, ok)
	}
}

func TestConvertValue_NegativeValues(t *testing.T) {
	t.Parallel()
	// La fonction n'a pas de clamping ; les valeurs négatives passent telles quelles.
	out, ok := ConvertValue(-0.5, UnitRatio, UnitPercent)
	if !ok || out != -50.0 {
		t.Errorf("-0.5 ratio→percent: got (%v, %v), want (-50, true)", out, ok)
	}
}

func TestIsKnownFormat_AllValidFormats(t *testing.T) {
	t.Parallel()
	cases := []Format{
		FormatInteger, FormatSignedInt,
		FormatPercent1, FormatPercent2, FormatKDR2,
		FormatDurationHMS, FormatSeconds,
		FormatString, FormatBoolean, FormatDateTime, FormatEnum,
	}
	for _, f := range cases {
		if !IsKnownFormat(f) {
			t.Errorf("IsKnownFormat(%q) = false, want true", f)
		}
	}
}

func TestIsKnownFormat_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	unknowns := []Format{
		"unknown",
		"percent_3",
		"INTEGER",
		"hex",
	}
	for _, f := range unknowns {
		if IsKnownFormat(f) {
			t.Errorf("IsKnownFormat(%q) = true, want false", f)
		}
	}
}
