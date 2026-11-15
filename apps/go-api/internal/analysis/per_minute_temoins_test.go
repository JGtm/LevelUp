package analysis

import (
	"strconv"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// Témoins de caractérisation des cadences par minute : chaque test compare, pour des entrées fixes (durée non
// multiple de 60, durée nulle), la valeur produite par le site à la valeur attendue au bit près (chaîne
// 'g', -1 : la plus courte qui relit exactement le flottant). Ils tiennent le comportement du site sur durée
// nulle (pointeur nil, saut, 0) indépendamment du helper PerMinute. Les valeurs attendues sont celles que les
// sites rendaient avant leur passage par PerMinute.

func temoinF(t *testing.T, label string, v float64, want string) {
	t.Helper()
	got := strconv.FormatFloat(v, 'g', -1, 64)
	t.Logf("%s = %s", label, got)
	if got != want {
		t.Errorf("%s = %s, attendu %s", label, got, want)
	}
}

func temoinP(t *testing.T, label string, v *float64, want string) {
	t.Helper()
	if v == nil {
		t.Logf("%s = nil", label)
		if want != "nil" {
			t.Errorf("%s = nil, attendu %s", label, want)
		}
		return
	}
	temoinF(t, label, *v, want)
}

func temoinS(t *testing.T, label, got, want string) {
	t.Helper()
	t.Logf("%s = %s", label, got)
	if got != want {
		t.Errorf("%s = %s, attendu %s", label, got, want)
	}
}

func TestTemoinMinute_KPIStats(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		mkRow(11, 7, 3, 937, canonical.OutcomeWin, nil, nil),
		mkRow(23, 13, 5, 1051, canonical.OutcomeLoss, nil, nil),
		mkRow(4, 2, 1, 59, canonical.OutcomeWin, nil, nil),
	}
	got := ComputeKPIStats(rows, 225)
	temoinF(t, "kpi.kpm", got.KillsPerMinute, "1.1138251099169516")
	temoinF(t, "kpi.dpm", got.DeathsPerMinute, "0.6448461162677088")
	temoinF(t, "kpi.apm", got.AssistsPerMinute, "0.26380068392769906")

	zero := ComputeKPIStats([]canonical.PlayerMatchRow{mkRow(9, 4, 2, 0, canonical.OutcomeWin, nil, nil)}, 225)
	temoinF(t, "kpi.zero.kpm", zero.KillsPerMinute, "0")
	temoinF(t, "kpi.zero.dpm", zero.DeathsPerMinute, "0")
	temoinF(t, "kpi.zero.apm", zero.AssistsPerMinute, "0")
}

func TestTemoinMinute_ExplorerCadence(t *testing.T) {
	var stats domain.ExplorerTargetSampleStats
	setPerMinuteCadence(&stats, &domain.ParticipantStatsAggregate{Kills: 37, Deaths: 29, Assists: 11, TimePlayedSeconds: 1987})
	temoinP(t, "explorer.kpm", stats.KillsPerMin, "1.1172622043281328")
	temoinP(t, "explorer.dpm", stats.DeathsPerMin, "0.8756919979869149")
	temoinP(t, "explorer.apm", stats.AssistsPerMin, "0.33215903371917466")

	var zero domain.ExplorerTargetSampleStats
	setPerMinuteCadence(&zero, &domain.ParticipantStatsAggregate{Kills: 5, Deaths: 3, Assists: 1, TimePlayedSeconds: 0})
	temoinP(t, "explorer.zero.kpm", zero.KillsPerMin, "nil")
	temoinP(t, "explorer.zero.dpm", zero.DeathsPerMin, "nil")
	temoinP(t, "explorer.zero.apm", zero.AssistsPerMin, "nil")
}

func TestTemoinMinute_HighlightTuile(t *testing.T) {
	window := []canonical.PlayerMatchRow{
		mkRow(11, 7, 3, 937, canonical.OutcomeWin, nil, nil),
		mkRow(23, 13, 5, 1051, canonical.OutcomeLoss, nil, nil),
		mkRow(99, 99, 99, 0, canonical.OutcomeLoss, nil, nil),
	}
	item := buildPerMinuteHighlightCanonical(window)
	if item == nil {
		t.Fatal("tuile attendue")
	}
	if len(item.Slides) != 3 {
		t.Fatalf("slides = %d, attendu 3", len(item.Slides))
	}
	for i, want := range []string{"1.03", "0.60", "0.24"} {
		temoinS(t, "highlight.slide"+strconv.Itoa(i), item.Slides[i].Value, want)
	}
	temoinS(t, "highlight.value", item.Value, "1.03")

	if got := buildPerMinuteHighlightCanonical([]canonical.PlayerMatchRow{mkRow(9, 4, 2, 0, canonical.OutcomeWin, nil, nil)}); got != nil {
		t.Errorf("durée nulle : tuile nil attendue, obtenu %+v", got)
	} else {
		t.Log("highlight.zero = nil")
	}
}

func synthRows() []legacymatch.SynthesisMatchRow {
	a, b, z := 937, 1051, 0
	return []legacymatch.SynthesisMatchRow{
		{Kills: 11, Deaths: 7, TimePlayedSecs: &a},
		{Kills: 23, Deaths: 13, TimePlayedSecs: &b},
		{Kills: 5, Deaths: 3, TimePlayedSecs: &z},
	}
}

func TestTemoinMinute_SynthesisKPIsLegacy(t *testing.T) {
	got := ComputeSynthesisKPIs(synthRows(), false)
	temoinP(t, "synth.kpm", got.KillsPerMin, "1.18")

	z := 0
	zero := ComputeSynthesisKPIs([]legacymatch.SynthesisMatchRow{{Kills: 5, TimePlayedSecs: &z}}, false)
	temoinP(t, "synth.zero.kpm", zero.KillsPerMin, "nil")
	none := ComputeSynthesisKPIs([]legacymatch.SynthesisMatchRow{{Kills: 5}}, false)
	temoinP(t, "synth.nil.kpm", none.KillsPerMin, "nil")
}

func TestTemoinMinute_SynthesisKPIsCanonical(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		mkRow(11, 7, 3, 937, canonical.OutcomeWin, nil, nil),
		mkRow(23, 13, 5, 1051, canonical.OutcomeLoss, nil, nil),
		mkRow(4, 2, 1, 0, canonical.OutcomeWin, nil, nil),
	}
	got := ComputeSynthesisKPIsFromCanonical(rows, false, 225)
	temoinP(t, "canon.kpm", got.KillsPerMin, "1.15")
	temoinP(t, "canon.dpm", got.DeathsPerMin, "0.66")
	temoinP(t, "canon.apm", got.AssistsPerMin, "0.27")

	zero := ComputeSynthesisKPIsFromCanonical([]canonical.PlayerMatchRow{mkRow(9, 4, 2, 0, canonical.OutcomeWin, nil, nil)}, false, 225)
	temoinP(t, "canon.zero.kpm", zero.KillsPerMin, "nil")
	temoinP(t, "canon.zero.dpm", zero.DeathsPerMin, "nil")
	temoinP(t, "canon.zero.apm", zero.AssistsPerMin, "nil")
}

// Les profils moyennent des cadences par match (kills * 60 / secondes dans l'ancienne forme) puis arrondissent
// à 2 décimales : l'arrondi affiché est la valeur comparée.
func TestTemoinMinute_Profils(t *testing.T) {
	squad := []domain.SquadMatchRow{
		{Kills: 11, TimePlayedSecs: 937},
		{Kills: 23, TimePlayedSecs: 1051},
		{Kills: 7, TimePlayedSecs: 59},
		{Kills: 9, TimePlayedSecs: 0},
	}
	temoinF(t, "profil.squad.kpm", ComputeParticipationProfile(squad, "n", "c").Values[StatLabelKillsPerMin], "3.05")
	temoinF(t, "profil.squad.zero.kpm", ComputeParticipationProfile(squad[3:], "n", "c").Values[StatLabelKillsPerMin], "0")

	mates := []domain.TeammateMatchRow{
		{Kills: 11, TimePlayedSecs: 937},
		{Kills: 23, TimePlayedSecs: 1051},
		{Kills: 7, TimePlayedSecs: 59},
		{Kills: 9, TimePlayedSecs: 0},
	}
	temoinF(t, "profil.mate.kpm", ComputeTeammateProfile(mates, "n", "c").Values[StatLabelKillsPerMin], "3.05")
	temoinF(t, "profil.mate.zero.kpm", ComputeTeammateProfile(mates[3:], "n", "c").Values[StatLabelKillsPerMin], "0")
}
