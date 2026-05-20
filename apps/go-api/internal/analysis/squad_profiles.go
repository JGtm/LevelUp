// Package analysis — squad_profiles.go : profils radar et records joueurs escouade.
package analysis

import (
	"math"

	"levelup/go-api/internal/domain"
)

// Clés canoniques des axes/stats partagés entre les profils radar squad/teammates
// et les RecordSpec downstream. Externalisées pour la cohérence des map keys.
const (
	StatLabelKills       = "kills"
	StatLabelDeaths      = "deaths"
	StatLabelAssists     = "assists"
	StatLabelAccuracy    = "accuracy"
	StatLabelKDA         = "kda"
	StatLabelKillsPerMin = "kills_per_min"
)

// =============================================================================
// Profil de participation radar
// =============================================================================

// ComputeParticipationProfile calcule le profil radar à 6 axes d'un joueur
// depuis ses matchs communs en escouade.
//
// Axes : kills, deaths, assists, accuracy, kills_per_min, kda.
// Chaque valeur est la moyenne sur les matchs non-nuls.
// Miroir de TeammateStats.ParticipationProfile dans teammates_service.py.
func ComputeParticipationProfile(rows []domain.SquadMatchRow, name, color string) domain.ParticipationProfile {
	if len(rows) == 0 {
		return domain.ParticipationProfile{Name: name, Color: color, Values: map[string]float64{}}
	}

	var (
		sumKills, sumDeaths, sumAssists, sumAccuracy, sumKPM, sumKDA float64
		nKills, nDeaths, nAssists, nAccuracy, nKPM, nKDA             int
	)
	for _, r := range rows {
		sumKills += float64(r.Kills)
		nKills++
		sumDeaths += float64(r.Deaths)
		nDeaths++
		sumAssists += float64(r.Assists)
		nAssists++
		if r.Accuracy != nil {
			sumAccuracy += *r.Accuracy
			nAccuracy++
		}
		if r.TimePlayedSecs > 0 {
			kpm := float64(r.Kills) * 60.0 / float64(r.TimePlayedSecs)
			sumKPM += kpm
			nKPM++
		}
		if r.KDA != nil {
			sumKDA += *r.KDA
			nKDA++
		}
	}

	avg := func(s float64, n int) float64 {
		if n == 0 {
			return 0
		}
		return math.Round((s/float64(n))*100) / 100
	}

	return domain.ParticipationProfile{
		Name:  name,
		Color: color,
		Values: map[string]float64{
			StatLabelKills:       avg(sumKills, nKills),
			StatLabelDeaths:      avg(sumDeaths, nDeaths),
			StatLabelAssists:     avg(sumAssists, nAssists),
			StatLabelAccuracy:    avg(sumAccuracy, nAccuracy),
			StatLabelKillsPerMin: avg(sumKPM, nKPM),
			StatLabelKDA:         avg(sumKDA, nKDA),
		},
	}
}

// ComputeTeammateProfile calcule le profil radar depuis les lignes TeammateMatchRow.
func ComputeTeammateProfile(rows []domain.TeammateMatchRow, name, color string) domain.ParticipationProfile {
	if len(rows) == 0 {
		return domain.ParticipationProfile{Name: name, Color: color, Values: map[string]float64{}}
	}

	var (
		sumKills, sumDeaths, sumAssists, sumAccuracy, sumKPM, sumKDA float64
		nKills, nDeaths, nAssists, nAccuracy, nKPM, nKDA             int
	)
	for _, r := range rows {
		sumKills += float64(r.Kills)
		nKills++
		sumDeaths += float64(r.Deaths)
		nDeaths++
		sumAssists += float64(r.Assists)
		nAssists++
		if r.Accuracy != nil {
			sumAccuracy += *r.Accuracy
			nAccuracy++
		}
		if r.TimePlayedSecs > 0 {
			kpm := float64(r.Kills) * 60.0 / float64(r.TimePlayedSecs)
			sumKPM += kpm
			nKPM++
		}
		if r.Ratio != nil {
			sumKDA += *r.Ratio
			nKDA++
		}
	}

	avg := func(s float64, n int) float64 {
		if n == 0 {
			return 0
		}
		return math.Round((s/float64(n))*100) / 100
	}

	return domain.ParticipationProfile{
		Name:  name,
		Color: color,
		Values: map[string]float64{
			StatLabelKills:       avg(sumKills, nKills),
			StatLabelDeaths:      avg(sumDeaths, nDeaths),
			StatLabelAssists:     avg(sumAssists, nAssists),
			StatLabelAccuracy:    avg(sumAccuracy, nAccuracy),
			StatLabelKillsPerMin: avg(sumKPM, nKPM),
			StatLabelKDA:         avg(sumKDA, nKDA),
		},
	}
}

// =============================================================================
// Records escouade — miroir de squad_records.py
// =============================================================================

// squadMetrics définit les métriques de record et leur sens (false=max, true=min).
var squadMetrics = []struct {
	name    string
	fromRow func(domain.SquadMatchRow) *float64
	isMin   bool
}{
	{StatLabelKills, func(r domain.SquadMatchRow) *float64 { v := float64(r.Kills); return &v }, false},
	{StatLabelDeaths, func(r domain.SquadMatchRow) *float64 { v := float64(r.Deaths); return &v }, true},
	{StatLabelAssists, func(r domain.SquadMatchRow) *float64 { v := float64(r.Assists); return &v }, false},
	{StatLabelKDA, func(r domain.SquadMatchRow) *float64 { return r.KDA }, false},
	{StatLabelAccuracy, func(r domain.SquadMatchRow) *float64 { return r.Accuracy }, false},
}

// ComputeSquadRecords calcule les records individuels pour un joueur
// sur un ensemble de matchs communs.
// Retourne un map metric → valeur record (nil si aucune donnée).
func ComputeSquadRecords(rows []domain.SquadMatchRow) map[string]*float64 {
	result := make(map[string]*float64, len(squadMetrics))
	for _, m := range squadMetrics {
		result[m.name] = nil
	}
	if len(rows) == 0 {
		return result
	}
	for _, m := range squadMetrics {
		for _, row := range rows {
			v := m.fromRow(row)
			if v == nil {
				continue
			}
			cur := result[m.name]
			if cur == nil {
				cp := *v
				result[m.name] = &cp
				continue
			}
			if m.isMin && *v < *cur {
				cp := *v
				result[m.name] = &cp
			} else if !m.isMin && *v > *cur {
				cp := *v
				result[m.name] = &cp
			}
		}
	}
	return result
}

// ComputeTeammateRecords calcule les records d'un coéquipier (TeammateMatchRow).
func ComputeTeammateRecords(rows []domain.TeammateMatchRow) map[string]*float64 {
	type metricDef struct {
		name    string
		fromRow func(domain.TeammateMatchRow) *float64
		isMin   bool
	}
	metrics := []metricDef{
		{StatLabelKills, func(r domain.TeammateMatchRow) *float64 { v := float64(r.Kills); return &v }, false},
		{StatLabelDeaths, func(r domain.TeammateMatchRow) *float64 { v := float64(r.Deaths); return &v }, true},
		{StatLabelAssists, func(r domain.TeammateMatchRow) *float64 { v := float64(r.Assists); return &v }, false},
		{StatLabelKDA, func(r domain.TeammateMatchRow) *float64 { return r.Ratio }, false},
		{StatLabelAccuracy, func(r domain.TeammateMatchRow) *float64 { return r.Accuracy }, false},
	}
	result := make(map[string]*float64, len(metrics))
	for _, m := range metrics {
		result[m.name] = nil
	}
	for _, m := range metrics {
		for _, row := range rows {
			v := m.fromRow(row)
			if v == nil {
				continue
			}
			cur := result[m.name]
			if cur == nil {
				cp := *v
				result[m.name] = &cp
				continue
			}
			if m.isMin && *v < *cur {
				cp := *v
				result[m.name] = &cp
			} else if !m.isMin && *v > *cur {
				cp := *v
				result[m.name] = &cp
			}
		}
	}
	return result
}
