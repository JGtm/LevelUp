//go:build integration

// match_view_cancellation_integration_test.go - E6.2a-F3 : une lecture de la vue match coupee
// par l'abandon de la requete est un WARN, jamais une ERREUR de panne (contrat de
// service/match_view_source_failures.go).

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func contexteAbandonne() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestMatchViewRepo_LectureAbandonnee_JournaliseeEnWarn(t *testing.T) {
	cas := []struct {
		nom      string
		lire     func(t *testing.T) error
		fragment string
	}{
		{"stats d'objectifs (Q12b)", func(t *testing.T) error {
			_, err := loadMatchObjectiveStats(contexteAbandonne(), openMemDBForProbe(t), "m1")
			return err
		}, "stats objectifs indisponibles"},
		{"sonde des stats d'Assaut", func(t *testing.T) error {
			_, err := loadMatchBombStats(contexteAbandonne(), openMemDBForProbe(t), "m1")
			return err
		}, "sonde des stats d'Assaut"},
		{"arme favorite du scoreboard", func(t *testing.T) error {
			NewMatchViewRepo(newTestPlayerDB(t), pTestXUID).attachTopWeapons(contexteAbandonne(), "m1",
				[]domain.ScoreboardRaw{{XUID: "x1"}})
			return nil
		}, "arme favorite du scoreboard"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf := captureSlog(t)
			_ = c.lire(t)
			if hasLogLevelContaining(t, buf, "ERROR", c.fragment) {
				t.Errorf("une annulation ne doit pas etre journalisee en ERROR : %s", buf.String())
			}
			if !hasLogLevelContaining(t, buf, "WARN", c.fragment) {
				t.Errorf("WARN %q attendu : %s", c.fragment, buf.String())
			}
		})
	}
}
