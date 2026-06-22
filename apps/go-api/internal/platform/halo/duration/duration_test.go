package duration

import (
	"errors"
	"testing"
	"time"
)

// TestParseISO8601Duration couvre la grammaire canonique + le cas qui divergeait
// (P1D / P2D sans marqueur T) qui doit désormais parser de façon COHÉRENTE.
func TestParseISO8601Duration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"PT10M", 10 * time.Minute},
		{"PT46S", 46 * time.Second},
		{"PT11M59.25S", 11*time.Minute + 59*time.Second + 250*time.Millisecond},
		{"PT1H30M", time.Hour + 30*time.Minute},
		{"PT1M8.2S", time.Minute + 8*time.Second + 200*time.Millisecond},
		{"PT1H2M3S", time.Hour + 2*time.Minute + 3*time.Second},
		// Cas de la divergence : jours purs, T optionnel.
		{"P1D", 24 * time.Hour},
		{"P2D", 48 * time.Hour},
		{"P1DT2H", 24*time.Hour + 2*time.Hour},
		{"P1DT7H50M24.6360455S", 24*time.Hour + 7*time.Hour + 50*time.Minute + 24*time.Second + 636*time.Millisecond},
	}
	for _, tc := range cases {
		got, err := ParseISO8601Duration(tc.in)
		if err != nil {
			t.Errorf("ParseISO8601Duration(%q) erreur: %v", tc.in, err)
			continue
		}
		// 1 ms de slop pour les conversions flottantes.
		diff := got - tc.want
		if diff < -time.Millisecond || diff > time.Millisecond {
			t.Errorf("ParseISO8601Duration(%q): want %v, got %v", tc.in, tc.want, got)
		}
	}
}

func TestParseISO8601Duration_Invalid(t *testing.T) {
	// "P" et "PT" matchent la grammaire mais sans composante → invalides.
	invalid := []string{"P", "PT", "P1Y", "P1W", "10M", "not-a-duration", "PT1.5M", "garbage", "PT5MX"}
	for _, in := range invalid {
		if _, err := ParseISO8601Duration(in); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("ParseISO8601Duration(%q): want ErrInvalidDuration, got %v", in, err)
		}
	}
}

// TestSeconds : parité openspartan/mapper (troncature vers le bas).
func TestSeconds(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"", 0, false},
		{"PT11M59.25S", 719, false}, // 719.25 tronqué → 719
		{"PT1H30M", 5400, false},
		{"P1D", 86400, false},
		{"PT", 0, true},
		{"garbage", 0, true},
	}
	for _, c := range cases {
		got, err := Seconds(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("Seconds(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && got != c.want {
			t.Errorf("Seconds(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestSecondsFloat(t *testing.T) {
	got, err := SecondsFloat("PT16.48S")
	if err != nil {
		t.Fatalf("SecondsFloat erreur: %v", err)
	}
	if got < 16.479 || got > 16.481 {
		t.Errorf("SecondsFloat(PT16.48S) = %v, want ~16.48", got)
	}
	if _, err := SecondsFloat("PT"); !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("SecondsFloat(PT) want ErrInvalidDuration, got %v", err)
	}
}

// TestSecondsInt64 : parité Compare (dégradation gracieuse 0, troncature).
func TestSecondsInt64(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"garbage", 0},
		{"P1DT7H50M24.6360455S", 1*86400 + 7*3600 + 50*60 + 24}, // 114624, tronqué
		{"PT30M", 1800},
		{"PT1H", 3600},
		{"PT45.9S", 45}, // tronqué
		{"P2D", 172800}, // cas divergent désormais cohérent
		{"PT0S", 0},
		{"P", 0},  // P nu → invalide → 0 (gracieux)
		{"PT", 0}, // idem
	}
	for _, c := range cases {
		if got := SecondsInt64(c.in); got != c.want {
			t.Errorf("SecondsInt64(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestSecondsRoundedBoundedPtr : parité Halo 5 (arrondi + borne 24h + nil).
func TestSecondsRoundedBoundedPtr(t *testing.T) {
	cases := []struct {
		in   string
		want *int
	}{
		{"", nil},
		{"garbage", nil},
		{"PT5M41.7930011S", ptr(342)}, // 341.79 → arrondi 342
		{"PT12M12.9155475S", ptr(733)},
		{"PT1H2M3S", ptr(3723)},
		{"PT9.35S", ptr(9)},
		{"PT", nil},
		{"P", nil},
		{"PT25H", nil},                   // > 24h plausible → nil
		{"PT99999999999999999999S", nil}, // overflow → nil (pas un négatif absurde)
		{"P1D", ptr(86400)},              // 24h pile, dans la borne
	}
	for _, c := range cases {
		got := SecondsRoundedBoundedPtr(c.in)
		if (got == nil) != (c.want == nil) {
			t.Fatalf("SecondsRoundedBoundedPtr(%q) nullité = %v, want %v", c.in, got, c.want)
		}
		if got != nil && *got != *c.want {
			t.Errorf("SecondsRoundedBoundedPtr(%q) = %d, want %d", c.in, *got, *c.want)
		}
	}
}

// TestMillisBounded : parité events Halo 5 (ms fractionnaires bornés).
func TestMillisBounded(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"PT33.2154416S", 33215, true},
		{"PT5M41.7899986S", 341790, true},
		{"PT0.0950007S", 95, true},
		{"PT", 0, false},
		{"", 0, false},
		{"garbage", 0, false},
		{"PT25H", 0, false}, // hors borne
	}
	for _, c := range cases {
		got, ok := MillisBounded(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("MillisBounded(%q) = (%d,%v), want (%d,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestSecondsFloatBoundedPtr(t *testing.T) {
	if got := SecondsFloatBoundedPtr("PT16.48S"); got == nil || *got < 16.479 || *got > 16.481 {
		t.Errorf("SecondsFloatBoundedPtr(PT16.48S) = %v, want ~16.48", got)
	}
	if got := SecondsFloatBoundedPtr("PT"); got != nil {
		t.Errorf("SecondsFloatBoundedPtr(PT) = %v, want nil", got)
	}
	if got := SecondsFloatBoundedPtr("PT25H"); got != nil {
		t.Errorf("SecondsFloatBoundedPtr(PT25H) = %v, want nil (hors borne)", got)
	}
}

func ptr(n int) *int { return &n }
