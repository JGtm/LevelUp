// Package breakdown agrege les rows joueur par dimension (carte, mode,
// playlist) et calcule les compteurs W/L/T/DNF + winrate. C'est un helper pur :
// 0 dependance DB ni HTTP, tout consommateur projette ses propres rows
// (canonical.PlayerMatchRow, domain.MatchHistoryRow, etc.) vers breakdown.Row.
package breakdown

import "levelup/go-api/internal/games/canonical"

// Row est l'entree minimale consommee par les helpers de breakdown.
// Le service appelant construit le slice en projetant les champs necessaires
// depuis sa propre source. Les champs zero sont consideres comme absents
// (cf. ignore-rules dans chaque helper).
type Row struct {
	Outcome          canonical.Outcome
	MapID            string
	MapLabel         string
	ModeName         string
	ModeCategory     string
	PlaylistName     string
	PerformanceScore *float64
}

// Counts agrege les compteurs W/L/T/DNF + ratios sur un sous-ensemble de rows.
// Les ratios sont en [0,1] (jamais en pourcent). La somme WinRate + LossRate +
// TieRate + DNFRate vaut 1 quand Played > 0 et l'ensemble des outcomes connus
// est couvert ; les outcomes inconnus ne contribuent a aucun compteur.
type Counts struct {
	Played   int
	Wins     int
	Losses   int
	Ties     int
	DNF      int
	WinRate  float64
	LossRate float64
	TieRate  float64
	DNFRate  float64
}

// computeCounts retourne les compteurs pour un slice de rows. Les outcomes
// inconnus (vides ou non canoniques) ne sont compres dans aucun seau, mais
// comptent dans Played pour preserver la cardinalite reelle.
func computeCounts(rows []Row) Counts {
	c := Counts{Played: len(rows)}
	for _, r := range rows {
		switch r.Outcome {
		case canonical.OutcomeWin:
			c.Wins++
		case canonical.OutcomeLoss:
			c.Losses++
		case canonical.OutcomeTie:
			c.Ties++
		case canonical.OutcomeDNF:
			c.DNF++
		}
	}
	if c.Played > 0 {
		n := float64(c.Played)
		c.WinRate = float64(c.Wins) / n
		c.LossRate = float64(c.Losses) / n
		c.TieRate = float64(c.Ties) / n
		c.DNFRate = float64(c.DNF) / n
	}
	return c
}

// avgPerformanceScore calcule la moyenne des PerformanceScore non nil.
// Retourne nil si aucune row n'a de score.
func avgPerformanceScore(rows []Row) *float64 {
	var sum float64
	var n int
	for _, r := range rows {
		if r.PerformanceScore != nil {
			sum += *r.PerformanceScore
			n++
		}
	}
	if n == 0 {
		return nil
	}
	avg := sum / float64(n)
	return &avg
}
