package haloapi

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

// TestParseRetryAfter : secondes, date HTTP, borne, illisible.
func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	// Les huit premiers cas portent les noms du test du client de synchronisation d'avant la brique.
	cas := []struct {
		nom  string
		h    string
		veut time.Duration
	}{
		{"empty", "", 0},
		{"seconds", "120", 120 * time.Second},
		{"zero", "0", 0},
		{"negative", "-5", 0},
		{"garbage", "abc", 0},
		{"clamp_over_max", "99999", maxRetryAfter},
		{"http_date_future", now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second},
		{"http_date_past", now.Add(-time.Minute).Format(http.TimeFormat), 0},
		{"secondes_bordees_d_espaces", " 3 ", 3 * time.Second},
		{"une_heure_bornee", strconv.Itoa(3600), maxRetryAfter},
		{"texte_libre", "n'importe quoi", 0},
		{"date_http_bornee", now.Add(time.Hour).Format(http.TimeFormat), maxRetryAfter},
		{"douze_secondes", "12", 12 * time.Second},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ParseRetryAfter(c.h, now); got != c.veut {
				t.Errorf("ParseRetryAfter(%q) = %v, veut %v", c.h, got, c.veut)
			}
		})
	}
}
