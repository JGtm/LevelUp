//go:build integration

// match_view_cancellation_integration_test.go - E6.2a-F3 : une lecture de la vue match coupee
// par l'abandon de la requete est un WARN, jamais une ERREUR de panne (contrat de
// service/match_view_source_failures.go).

package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func contexteAbandonne() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// contexteDelaiExpire : un contexte DERIVE dont le delai interne est expire alors que la
// requete (le parent) vit.
func contexteDelaiExpire() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	_ = cancel
	return ctx
}

func TestMatchViewRepo_LectureAbandonnee_PasDErreur_DelaiInterne_Erreur(t *testing.T) {
	cas := []struct {
		nom      string
		lire     func(t *testing.T, ctx context.Context) error
		fragment string
	}{
		{"stats d'objectifs (Q12b)", func(t *testing.T, ctx context.Context) error {
			_, err := loadMatchObjectiveStats(ctx, openMemDBForProbe(t), "m1")
			return err
		}, "stats objectifs indisponibles"},
		{"sonde des stats d'Assaut", func(t *testing.T, ctx context.Context) error {
			_, err := loadMatchBombStats(ctx, openMemDBForProbe(t), "m1")
			return err
		}, "sonde des stats d'Assaut"},
		{"arme favorite du scoreboard", func(t *testing.T, ctx context.Context) error {
			NewMatchViewRepo(newTestPlayerDB(t), pTestXUID).attachTopWeapons(ctx, "m1",
				[]domain.ScoreboardRaw{{XUID: "x1"}})
			return nil
		}, "arme favorite du scoreboard"},
	}
	for _, c := range cas {
		t.Run(c.nom+"/client parti", func(t *testing.T) {
			buf := captureSlog(t)
			_ = c.lire(t, contexteAbandonne())
			if hasLogLevelContaining(t, buf, "ERROR", c.fragment) {
				t.Errorf("un client parti ne doit pas etre journalise en ERROR : %s", buf.String())
			}
			if !hasLogLevelContaining(t, buf, "DEBUG", c.fragment) {
				t.Errorf("DEBUG %q attendu : %s", c.fragment, buf.String())
			}
		})
		t.Run(c.nom+"/delai interne", func(t *testing.T) {
			buf := captureSlog(t)
			_ = c.lire(t, contexteDelaiExpire())
			if !hasLogLevelContaining(t, buf, "ERROR", c.fragment) {
				t.Errorf("un delai interne expire est une panne, ERROR %q attendu : %s", c.fragment, buf.String())
			}
		})
	}
}
