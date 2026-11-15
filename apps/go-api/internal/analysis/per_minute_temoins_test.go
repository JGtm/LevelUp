package analysis

import (
	"strconv"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// Témoins de caractérisation des cadences par minute : chaque test rend, pour des entrées fixes (durée non
// multiple de 60, durée nulle), la valeur exacte produite par le site, imprimée au bit près. Ils tiennent le
// comportement du site sur durée nulle (pointeur nil, saut, 0) indépendamment du helper PerMinute.

func temoinF(t *testing.T, label string, v float64) {
	t.Helper()
	t.Logf("%s = %s", label, strconv.FormatFloat(v, 'g', -1, 64))
}

func temoinP(t *testing.T, label string, v *float64) {
	t.Helper()
	if v == nil {
		t.Logf("%s = nil", label)
		return
	}
	temoinF(t, label, *v)
}

func TestTemoinMinute_KPIStats(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		mkRow(11, 7, 3, 937, canonical.OutcomeWin, nil, nil),
		mkRow(23, 13, 5, 1051, canonical.OutcomeLoss, nil, nil),
		mkRow(4, 2, 1, 59, canonical.OutcomeWin, nil, nil),
	}
	got := ComputeKPIStats(rows, 225)
	temoinF(t, "kpi.kpm", got.KillsPerMinute)
	temoinF(t, "kpi.dpm", got.DeathsPerMinute)
	temoinF(t, "kpi.apm", got.AssistsPerMinute)

	zero := ComputeKPIStats([]canonical.PlayerMatchRow{mkRow(9, 4, 2, 0, canonical.OutcomeWin, nil, nil)}, 225)
	temoinF(t, "kpi.zero.kpm", zero.KillsPerMinute)
	temoinF(t, "kpi.zero.dpm", zero.DeathsPerMinute)
	temoinF(t, "kpi.zero.apm", zero.AssistsPerMinute)
}

func TestTemoinMinute_ExplorerCadence(t *testing.T) {
	var stats domain.ExplorerTargetSampleStats
	setPerMinuteCadence(&stats, &domain.ParticipantStatsAggregate{Kills: 37, Deaths: 29, Assists: 11, TimePlayedSeconds: 1987})
	temoinP(t, "explorer.kpm", stats.KillsPerMin)
	temoinP(t, "explorer.dpm", stats.DeathsPerMin)
	temoinP(t, "explorer.apm", stats.AssistsPerMin)

	var zero domain.ExplorerTargetSampleStats
	setPerMinuteCadence(&zero, &domain.ParticipantStatsAggregate{Kills: 5, Deaths: 3, Assists: 1, TimePlayedSeconds: 0})
	temoinP(t, "explorer.zero.kpm", zero.KillsPerMin)
	temoinP(t, "explorer.zero.dpm", zero.DeathsPerMin)
	temoinP(t, "explorer.zero.apm", zero.AssistsPerMin)
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
	for i, s := range item.Slides {
		t.Logf("highlight.slide%d = %s", i, s.Value)
	}
	t.Logf("highlight.value = %s", item.Value)

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
	temoinP(t, "synth.kpm", got.KillsPerMin)

	z := 0
	zero := ComputeSynthesisKPIs([]legacymatch.SynthesisMatchRow{{Kills: 5, TimePlayedSecs: &z}}, false)
	temoinP(t, "synth.zero.kpm", zero.KillsPerMin)
	none := ComputeSynthesisKPIs([]legacymatch.SynthesisMatchRow{{Kills: 5}}, false)
	temoinP(t, "synth.nil.kpm", none.KillsPerMin)
}

func TestTemoinMinute_SynthesisKPIsCanonical(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		mkRow(11, 7, 3, 937, canonical.OutcomeWin, nil, nil),
		mkRow(23, 13, 5, 1051, canonical.OutcomeLoss, nil, nil),
		mkRow(4, 2, 1, 0, canonical.OutcomeWin, nil, nil),
	}
	got := ComputeSynthesisKPIsFromCanonical(rows, false, 225)
	temoinP(t, "canon.kpm", got.KillsPerMin)
	temoinP(t, "canon.dpm", got.DeathsPerMin)
	temoinP(t, "canon.apm", got.AssistsPerMin)

	zero := ComputeSynthesisKPIsFromCanonical([]canonical.PlayerMatchRow{mkRow(9, 4, 2, 0, canonical.OutcomeWin, nil, nil)}, false, 225)
	temoinP(t, "canon.zero.kpm", zero.KillsPerMin)
	temoinP(t, "canon.zero.dpm", zero.DeathsPerMin)
	temoinP(t, "canon.zero.apm", zero.AssistsPerMin)
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
	temoinF(t, "profil.squad.kpm", ComputeParticipationProfile(squad, "n", "c").Values[StatLabelKillsPerMin])
	temoinF(t, "profil.squad.zero.kpm", ComputeParticipationProfile(squad[3:], "n", "c").Values[StatLabelKillsPerMin])

	mates := []domain.TeammateMatchRow{
		{Kills: 11, TimePlayedSecs: 937},
		{Kills: 23, TimePlayedSecs: 1051},
		{Kills: 7, TimePlayedSecs: 59},
		{Kills: 9, TimePlayedSecs: 0},
	}
	temoinF(t, "profil.mate.kpm", ComputeTeammateProfile(mates, "n", "c").Values[StatLabelKillsPerMin])
	temoinF(t, "profil.mate.zero.kpm", ComputeTeammateProfile(mates[3:], "n", "c").Values[StatLabelKillsPerMin])
}
