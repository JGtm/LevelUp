// Package analysis — tests purs de BuildSampleStats (explorer_target_stats.go).
//
// Le KDA per-match est NET ((k + a/3) − d) pour TOUS les titres ; l'agrégat est donc
// la moyenne NETTE ((Σk + Σa/3) − Σd)/N (peut être négatif), JAMAIS le quotient.
//
// Couvre la matrice :
//   - cas standard : tous les ratios calculables (KDA = moyenne nette)
//   - deaths=0 → KDR nil (mais KDA reste calculé, net)
//   - shots_fired=0 → accuracy nil
//   - kills=0 → headshot_rate nil
//   - sampleSize=0 → retourne nil
//   - agg=nil → retourne nil
//   - rendement combat : OffensiveConversion + DefensiveResistance présents/absents
package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestBuildSampleStats_StandardCase(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{
		Kills: 100, Deaths: 50, Assists: 30,
		Wins: 7, Losses: 2, Draws: 1,
		ShotsFired: 800, ShotsHit: 400,
		DamageDealt: 12000, DamageTaken: 8000,
		HeadshotKills: 25, MeleeKills: 5,
		PowerWeaponKills: 15, GrenadeKills: 10,
		TimePlayedSeconds: 600, PersonalScore: 15000,
	}
	medals := &domain.MedalCountsAggregate{Total: 142, Unique: 12, PerfectKills: 8}

	got := BuildSampleStats(agg, medals, 10, 225)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	if got.SampleSize != 10 {
		t.Errorf("SampleSize = %d, want 10", got.SampleSize)
	}
	if got.Kills != 100 || got.Deaths != 50 {
		t.Errorf("Kills/Deaths = %d/%d, want 100/50", got.Kills, got.Deaths)
	}
	if got.TotalMedals != 142 || got.UniqueMedals != 12 {
		t.Errorf("Medals = %d/%d, want 142/12", got.TotalMedals, got.UniqueMedals)
	}

	// KDA = moyenne NETTE ((100 + 30/3) − 50)/10 = (110 − 50)/10 = 60/10 = 6.0
	if got.KDA == nil || *got.KDA != 6.0 {
		t.Errorf("KDA = %v, want 6.0 (moyenne nette)", got.KDA)
	}
	// KDR = 100 / 50 = 2.0
	if got.KDR == nil || *got.KDR != 2.0 {
		t.Errorf("KDR = %v, want 2.0", got.KDR)
	}
	// WinRate = 7 / (7+2+1) = 0.7
	if got.WinRate == nil || *got.WinRate != 0.7 {
		t.Errorf("WinRate = %v, want 0.7", got.WinRate)
	}
	// Accuracy = 400 / 800 = 0.5
	if got.Accuracy == nil || *got.Accuracy != 0.5 {
		t.Errorf("Accuracy = %v, want 0.5", got.Accuracy)
	}
	// HeadshotRate = 25 / 100 = 0.25
	if got.HeadshotRate == nil || *got.HeadshotRate != 0.25 {
		t.Errorf("HeadshotRate = %v, want 0.25", got.HeadshotRate)
	}
	if got.OffensiveConversion == nil || *got.OffensiveConversion <= 0 {
		t.Errorf("OffensiveConversion attendue > 0, got %v", got.OffensiveConversion)
	}
	if got.DefensiveResistance == nil || *got.DefensiveResistance <= 0 {
		t.Errorf("DefensiveResistance attendue > 0, got %v", got.DefensiveResistance)
	}
	// Cadence par minute : 600s = 10 min → Kills/min = 100/10 = 10, etc.
	if got.KillsPerMin == nil || *got.KillsPerMin != 10.0 {
		t.Errorf("KillsPerMin = %v, want 10", got.KillsPerMin)
	}
	if got.DeathsPerMin == nil || *got.DeathsPerMin != 5.0 {
		t.Errorf("DeathsPerMin = %v, want 5", got.DeathsPerMin)
	}
	if got.AssistsPerMin == nil || *got.AssistsPerMin != 3.0 {
		t.Errorf("AssistsPerMin = %v, want 3", got.AssistsPerMin)
	}
	// AvgPersonalScore = 15000 / 10 = 1500
	if got.AvgPersonalScore == nil || *got.AvgPersonalScore != 1500.0 {
		t.Errorf("AvgPersonalScore = %v, want 1500", got.AvgPersonalScore)
	}
	if got.PerfectKills != 8 {
		t.Errorf("PerfectKills = %d, want 8", got.PerfectKills)
	}
}

func TestBuildSampleStats_TimePlayedZero(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{
		Kills: 10, Deaths: 5, Assists: 3,
		TimePlayedSeconds: 0,
	}
	got := BuildSampleStats(agg, nil, 2, 225)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	if got.KillsPerMin != nil || got.DeathsPerMin != nil || got.AssistsPerMin != nil {
		t.Errorf("cadence par minute attendue nil quand TimePlayedSeconds=0 ; got %v/%v/%v",
			got.KillsPerMin, got.DeathsPerMin, got.AssistsPerMin)
	}
}

func TestBuildSampleStats_DeathsZero(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{
		Kills: 10, Deaths: 0, Assists: 3,
		ShotsFired: 50, ShotsHit: 20,
		DamageDealt: 1500, DamageTaken: 0,
		HeadshotKills: 5,
	}
	got := BuildSampleStats(agg, nil, 2, 225)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	// KDA est NET et toujours calculé (même deaths=0) : ((10 + 3/3) − 0)/2 = 5.5.
	if got.KDA == nil || *got.KDA != 5.5 {
		t.Errorf("KDA (net) attendu 5.5 même quand deaths=0 ; got %v", got.KDA)
	}
	// KDR (k/d) reste nil quand deaths=0 (division par zéro).
	if got.KDR != nil {
		t.Errorf("KDR doit être nil quand deaths=0 ; got %v", got.KDR)
	}
	// L'accuracy reste calculable.
	if got.Accuracy == nil {
		t.Error("Accuracy doit être calculable même si deaths=0")
	}
	// DefensiveResistance nul car damage_taken=0.
	if got.DefensiveResistance != nil {
		t.Errorf("DefensiveResistance attendue nil quand damage_taken=0 ; got %v", got.DefensiveResistance)
	}
}

// KDA = FDA NET moyen ((k+a/3)−d)/N pour TOUS les titres (per-match net partout),
// JAMAIS le quotient. Ici ((100 + 30/3) − 50)/10 = 6.0. KDR reste calculé.
func TestBuildSampleStats_NetFDA_Universal(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{
		Kills: 100, Deaths: 50, Assists: 30,
		Wins: 7, Losses: 2, Draws: 1,
		ShotsFired: 800, ShotsHit: 400,
	}
	got := BuildSampleStats(agg, nil, 10, 115)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	wantKDA := (100.0 + 30.0/3.0 - 50.0) / 10.0 // 6.0
	if got.KDA == nil || *got.KDA != wantKDA {
		t.Errorf("KDA = %v, want FDA NET moyen %v", got.KDA, wantKDA)
	}
	if got.KDR == nil || *got.KDR != 2.0 {
		t.Errorf("KDR (non ambigu) doit rester calculé = 2.0, got %v", got.KDR)
	}
}

func TestBuildSampleStats_ShotsZero(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{
		Kills: 10, Deaths: 5,
		ShotsFired: 0, ShotsHit: 0,
	}
	got := BuildSampleStats(agg, nil, 1, 225)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	if got.Accuracy != nil {
		t.Errorf("Accuracy attendue nil quand shots_fired=0 ; got %v", got.Accuracy)
	}
}

func TestBuildSampleStats_KillsZero(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{
		Kills: 0, Deaths: 5,
		HeadshotKills: 0,
	}
	got := BuildSampleStats(agg, nil, 1, 225)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	if got.HeadshotRate != nil {
		t.Errorf("HeadshotRate attendue nil quand kills=0 ; got %v", got.HeadshotRate)
	}
	// KDR nil aussi (kills=0 et deaths=5 → 0/5 = 0, mais convention : KDR n'a de sens que quand kills>0 OU deaths>0)
	// Notre code retourne KDR=0.0 quand kills=0 et deaths>0. C'est OK : kdr=0 est une info utile (le joueur n'a pas tué).
	if got.KDR == nil || *got.KDR != 0.0 {
		t.Errorf("KDR attendu 0 quand kills=0 deaths>0 ; got %v", got.KDR)
	}
}

func TestBuildSampleStats_NilOrZeroSample(t *testing.T) {
	t.Run("agg nil", func(t *testing.T) {
		if got := BuildSampleStats(nil, nil, 5, 225); got != nil {
			t.Errorf("agg nil → attendu nil, got %+v", got)
		}
	})
	t.Run("sampleSize 0", func(t *testing.T) {
		agg := &domain.ParticipantStatsAggregate{Kills: 10, Deaths: 5}
		if got := BuildSampleStats(agg, nil, 0, 225); got != nil {
			t.Errorf("sampleSize=0 → attendu nil, got %+v", got)
		}
	})
	t.Run("sampleSize négatif", func(t *testing.T) {
		agg := &domain.ParticipantStatsAggregate{Kills: 10, Deaths: 5}
		if got := BuildSampleStats(agg, nil, -1, 225); got != nil {
			t.Errorf("sampleSize<0 → attendu nil, got %+v", got)
		}
	})
}

func TestBuildSampleStats_MedalsNil(t *testing.T) {
	agg := &domain.ParticipantStatsAggregate{Kills: 10, Deaths: 5}
	got := BuildSampleStats(agg, nil, 1, 225)
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	if got.TotalMedals != 0 || got.UniqueMedals != 0 {
		t.Errorf("Medals attendus 0/0 quand medals=nil, got %d/%d", got.TotalMedals, got.UniqueMedals)
	}
}

func TestBuildSampleStats_WinRateDNFExclu(t *testing.T) {
	// Wins=3, Losses=1, Draws=0 → played=4, wr=0.75. Les DNF ne sont pas comptés.
	agg := &domain.ParticipantStatsAggregate{
		Kills: 1, Deaths: 1,
		Wins: 3, Losses: 1, Draws: 0,
	}
	got := BuildSampleStats(agg, nil, 5, 225) // 5 matchs au total mais seulement 4 jouables
	if got == nil {
		t.Fatal("BuildSampleStats attendu non-nil")
	}
	if got.WinRate == nil || *got.WinRate != 0.75 {
		t.Errorf("WinRate = %v, want 0.75 (3/4)", got.WinRate)
	}
}
