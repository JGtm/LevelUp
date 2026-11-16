package config

import (
	"slices"
	"testing"
	"time"
)

func TestPublicBaseURLFromEnv(t *testing.T) {
	t.Setenv(envPublicBaseURL, "")
	if got := PublicBaseURLFromEnv(); got != "" {
		t.Errorf("variable absente : %q, attendu vide", got)
	}
	t.Setenv(envPublicBaseURL, "  https://levelup.example/app//  ")
	if got := PublicBaseURLFromEnv(); got != "https://levelup.example/app" {
		t.Errorf("espaces de bord et « / » finaux non retirés : %q", got)
	}
}

func TestFilmRetryWindowFromEnv(t *testing.T) {
	const def = 30 * 24 * time.Hour
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"", def},
		{"72", 72 * time.Hour},
		{"0", def},
		{"-5", def},
		{"abc", def},
	}
	for _, c := range cases {
		t.Setenv(envFilmRetryWindowHours, c.raw)
		if got := FilmRetryWindowFromEnv(def); got != c.want {
			t.Errorf("%s=%q : %v, attendu %v", envFilmRetryWindowHours, c.raw, got, c.want)
		}
	}
}

func TestSnapshotGraceWindowFromEnv(t *testing.T) {
	const def = 60 * 24 * time.Hour
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"", def},
		{"100", 100 * time.Hour},
		{"0", def},
		{"abc", def},
	}
	for _, c := range cases {
		t.Setenv(envSnapshotGraceHours, c.raw)
		if got := SnapshotGraceWindowFromEnv(def); got != c.want {
			t.Errorf("%s=%q : %v, attendu %v", envSnapshotGraceHours, c.raw, got, c.want)
		}
	}
}

func TestFreshFilmRetryDelaysFromEnv(t *testing.T) {
	def := []time.Duration{30 * time.Second, 30 * time.Second, 30 * time.Second}
	cases := []struct {
		raw  string
		want []time.Duration
	}{
		{"", def},
		{"0", nil},
		{"10,20", []time.Duration{10 * time.Second, 20 * time.Second}},
		{" 5 , x, -1, 7", []time.Duration{5 * time.Second, 7 * time.Second}},
		{"x,y", def},
	}
	for _, c := range cases {
		t.Setenv(envFreshFilmRetry, c.raw)
		if got := FreshFilmRetryDelaysFromEnv(def); !slices.Equal(got, c.want) {
			t.Errorf("%s=%q : %v, attendu %v", envFreshFilmRetry, c.raw, got, c.want)
		}
	}
}
