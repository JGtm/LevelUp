package wire

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/replaybuild"
)

// fauxRegistry : un registre de matchs fixe et des noms EN de carte.
type fauxRegistry struct {
	hits    []port.ReplayRegistryMatch
	hitsErr error
	en      map[string]string
}

func (f fauxRegistry) MatchesByIDOrPrefix(context.Context, string) ([]port.ReplayRegistryMatch, error) {
	return f.hits, f.hitsErr
}

func (f fauxRegistry) EnglishMapName(_ context.Context, mapID string) (string, error) {
	if n, ok := f.en[mapID]; ok {
		return n, nil
	}
	return "", errors.New("inconnue")
}

func (f fauxRegistry) UnmanagedModeMatchIDs(context.Context) ([]string, error) { return nil, nil }

// TestReplayMatchIdentity_Decisions : identifiant vide, inconnu, ambigu, mode non géré, noms de carte (EN puis
// brut), échec de lecture rendu tel quel.
func TestReplayMatchIdentity_Decisions(t *testing.T) {
	ctx := context.Background()
	hit := port.ReplayRegistryMatch{ID: "full-1", RawMapName: "Recharge", MapID: "a"}

	if _, _, err := replayMatchIdentity(ctx, fauxRegistry{}, "  "); err == nil {
		t.Error("identifiant vide : erreur attendue")
	}
	if _, _, err := replayMatchIdentity(ctx, fauxRegistry{}, "zz"); err == nil {
		t.Error("match inconnu : erreur attendue")
	}
	if _, _, err := replayMatchIdentity(ctx, fauxRegistry{hits: []port.ReplayRegistryMatch{hit, hit}}, "ful"); err == nil {
		t.Error("préfixe ambigu : erreur attendue")
	}
	boom := errors.New("boom")
	if _, _, err := replayMatchIdentity(ctx, fauxRegistry{hitsErr: boom}, "x"); !errors.Is(err, boom) {
		t.Errorf("échec de lecture : %v", err)
	}
	firefight := hit
	firefight.UnmanagedMode = true
	if _, _, err := replayMatchIdentity(ctx, fauxRegistry{hits: []port.ReplayRegistryMatch{firefight}}, "full"); !errors.Is(err, replaybuild.ErrModeNonGere) {
		t.Errorf("mode non géré : %v", err)
	}

	id, names, err := replayMatchIdentity(ctx, fauxRegistry{hits: []port.ReplayRegistryMatch{hit}, en: map[string]string{"a": " Recharge EN "}}, "full")
	if err != nil || id != "full-1" || len(names) != 2 || names[0] != "Recharge EN" || names[1] != "Recharge" {
		t.Errorf("identité = %q %v %v, attendu nom EN puis brut", id, names, err)
	}
	if _, names, _ := replayMatchIdentity(ctx, fauxRegistry{hits: []port.ReplayRegistryMatch{hit}}, "full"); len(names) != 1 || names[0] != "Recharge" {
		t.Errorf("sans nom EN : %v, attendu le nom brut seul", names)
	}
}

// TestAssembleProgressionResults_CadencesParMinute : frags et score personnel par minute de jeu, 0 sans durée.
func TestAssembleProgressionResults_CadencesParMinute(t *testing.T) {
	at := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	_, inputs := assembleProgressionResults([]port.ProgressionMatchRow{
		{MatchID: "m1", StartTime: at, Kills: 12, PersonalScore: 1500, TimeSeconds: 600},
		{MatchID: "m2", StartTime: at, Kills: 5, PersonalScore: 900, TimeSeconds: 0},
	}, map[string]float64{"m1": 70}, 225)
	if len(inputs) != 2 {
		t.Fatalf("entrées = %d, attendu 2", len(inputs))
	}
	if m := inputs[0].Metrics; m[records.MetricKPM] != 1.2 || m[records.MetricPSPM] != 150 || m[records.MetricPerformanceScore] != 70 {
		t.Errorf("m1 : %v", m)
	}
	if m := inputs[1].Metrics; m[records.MetricKPM] != 0 || m[records.MetricPSPM] != 0 {
		t.Errorf("m2 (durée inconnue) : %v", m)
	}
}
