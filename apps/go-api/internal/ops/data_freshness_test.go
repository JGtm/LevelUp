// Package ops — data_freshness_test.go : évaluation pure DC-3 sur dataset
// hétérogène (à jour, périmé, jamais synchronisé, sync inconnu, DB en erreur).
package ops

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func tp(t time.Time) *time.Time { return &t }

func TestEvaluatePlayerFreshness_Heterogeneous(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	th := DefaultFreshnessThresholds() // 48 h / 6 h / 7 j

	cases := []struct {
		name string
		in   PlayerFreshnessInput
		want string
	}{
		{
			name: "joueur à jour (match récent, sync récent)",
			in: PlayerFreshnessInput{
				Gamertag:     "JGtm",
				LastMatchAt:  tp(now.Add(-2 * time.Hour)),
				LastSyncOKAt: tp(now.Add(-30 * time.Minute)),
			},
			want: domain.FreshnessStatusOK,
		},
		{
			name: "joueur inactif mais moteur vivant → ok (DC-3 : warn exige les DEUX)",
			in: PlayerFreshnessInput{
				Gamertag:     "Inactif",
				LastMatchAt:  tp(now.Add(-30 * 24 * time.Hour)),
				LastSyncOKAt: tp(now.Add(-1 * time.Hour)),
			},
			want: domain.FreshnessStatusOK,
		},
		{
			name: "match périmé 3 j + sync mort 12 h → warn",
			in: PlayerFreshnessInput{
				Gamertag:     "Perime",
				LastMatchAt:  tp(now.Add(-72 * time.Hour)),
				LastSyncOKAt: tp(now.Add(-12 * time.Hour)),
			},
			want: domain.FreshnessStatusWarn,
		},
		{
			name: "match périmé 10 j + sync mort → critical",
			in: PlayerFreshnessInput{
				Gamertag:     "Mort",
				LastMatchAt:  tp(now.Add(-10 * 24 * time.Hour)),
				LastSyncOKAt: tp(now.Add(-24 * time.Hour)),
			},
			want: domain.FreshnessStatusCritical,
		},
		{
			name: "jamais synchronisé (aucun match, sync inconnu) → critical",
			in:   PlayerFreshnessInput{Gamertag: "Nouveau"},
			want: domain.FreshnessStatusCritical,
		},
		{
			name: "sync inconnu (titre live-only) + match récent → ok",
			in: PlayerFreshnessInput{
				Gamertag:    "H5Player",
				LastMatchAt: tp(now.Add(-3 * time.Hour)),
			},
			want: domain.FreshnessStatusOK,
		},
		{
			name: "sync inconnu (titre live-only) + match 4 j → warn",
			in: PlayerFreshnessInput{
				Gamertag:    "H5Stale",
				LastMatchAt: tp(now.Add(-4 * 24 * time.Hour)),
			},
			want: domain.FreshnessStatusWarn,
		},
		{
			name: "DB inaccessible → unknown",
			in:   PlayerFreshnessInput{Gamertag: "Casse", CheckError: "shared DB inaccessible"},
			want: domain.FreshnessStatusUnknown,
		},
	}
	for _, tc := range cases {
		got := EvaluatePlayerFreshness(tc.in, now, th)
		if got.Status != tc.want {
			t.Errorf("%s : status = %q, attendu %q (reason=%q)", tc.name, got.Status, tc.want, got.Reason)
		}
	}
}

func TestFreshnessThresholdsFromSettings(t *testing.T) {
	// Défauts quand absent / invalide.
	th := FreshnessThresholdsFromSettings(map[string]interface{}{})
	if th != DefaultFreshnessThresholds() {
		t.Errorf("settings vides : attendu défauts, got %+v", th)
	}
	th = FreshnessThresholdsFromSettings(map[string]interface{}{
		"freshness_warn_match_hours": -5.0, "freshness_warn_sync_hours": "bogus",
	})
	if th != DefaultFreshnessThresholds() {
		t.Errorf("settings invalides : attendu défauts, got %+v", th)
	}

	// Surcharge valide (JSON numérique → float64).
	th = FreshnessThresholdsFromSettings(map[string]interface{}{
		"freshness_warn_match_hours":    24.0,
		"freshness_warn_sync_hours":     2.0,
		"freshness_critical_match_days": 3.0,
	})
	if th.WarnMatchAge != 24*time.Hour || th.WarnSyncAge != 2*time.Hour || th.CriticalMatchAge != 72*time.Hour {
		t.Errorf("surcharge : got %+v", th)
	}
}
