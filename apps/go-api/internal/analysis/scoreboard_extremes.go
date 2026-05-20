// Package analysis — scoreboard_extremes.go : détection MVP et LVP d'un match.
//
// Port Go de src/ui/pages/match_view_scoreboard.py:_compute_mvp_lvp.
// MVP = joueur humain avec le plus de cellules « best » (≥ 2) sur l'ensemble
// des colonnes de stats numériques. LVP = symétrique sur les cellules « worst ».
// Les bots (xuid commençant par "bid(") sont exclus.
package analysis

import (
	"math"
	"strings"

	"levelup/go-api/internal/domain"
)

// ScoreboardExtremes contient les XUID du MVP et du LVP.
// Les deux champs sont vides si moins de 2 joueurs humains ou si aucun
// joueur n'atteint le seuil minimum de cellules (mvpMinCells).
type ScoreboardExtremes struct {
	MVPXUID string
	LVPXUID string
}

// mvpMinCells est le nombre minimum de cellules « best » (ou « worst ») pour
// qu'un joueur soit désigné MVP (ou LVP). Port de la constante Python.
const mvpMinCells = 2

// ComputeMVPLVP désigne le MVP (plus de cellules « best ») et le LVP (plus de
// cellules « worst ») parmi les joueurs humains du scoreboard.
//
// Colonnes considérées (higher-is-better) :
//
//	kills, assists, kda, accuracy, personal_score, damage_dealt,
//	headshot_kills, max_killing_spree, perfect_kills
//
// Colonnes inversées (lower-is-better = best) :
//
//	deaths, damage_taken
func ComputeMVPLVP(scoreboard []domain.ScoreboardRaw) ScoreboardExtremes {
	humans := humanPlayers(scoreboard)
	if len(humans) < 2 {
		return ScoreboardExtremes{}
	}

	cols := scoreboardColumns(humans)
	bestCount := make([]int, len(humans))
	worstCount := make([]int, len(humans))

	for _, c := range cols {
		minV, maxV := colMinMax(c.vals)
		if math.IsNaN(minV) || minV == maxV {
			continue
		}
		for i, v := range c.vals {
			if math.IsNaN(v) {
				continue
			}
			best := maxV
			worst := minV
			if c.inverted {
				best, worst = minV, maxV
			}
			switch v {
			case best:
				bestCount[i]++
			case worst:
				worstCount[i]++
			}
		}
	}

	mvp := pickBest(humans, bestCount, mvpMinCells)
	lvp := pickBest(humans, worstCount, mvpMinCells)
	return ScoreboardExtremes{MVPXUID: mvp, LVPXUID: lvp}
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

type scoreboardCol struct {
	vals     []float64
	inverted bool
}

func humanPlayers(rows []domain.ScoreboardRaw) []domain.ScoreboardRaw {
	out := make([]domain.ScoreboardRaw, 0, len(rows))
	for _, r := range rows {
		if !strings.HasPrefix(r.XUID, "bid(") {
			out = append(out, r)
		}
	}
	return out
}

func scoreboardColumns(rows []domain.ScoreboardRaw) []scoreboardCol {
	n := len(rows)
	fv := func(p *float64) float64 {
		if p == nil {
			return math.NaN()
		}
		return *p
	}
	fi := func(p *int) float64 {
		if p == nil {
			return math.NaN()
		}
		return float64(*p)
	}

	kills := make([]float64, n)
	deaths := make([]float64, n)
	assists := make([]float64, n)
	kda := make([]float64, n)
	accuracy := make([]float64, n)
	score := make([]float64, n)
	damageDealt := make([]float64, n)
	damageTaken := make([]float64, n)
	headshots := make([]float64, n)
	spree := make([]float64, n)
	perfect := make([]float64, n)

	for i, r := range rows {
		kills[i] = float64(r.Kills)
		deaths[i] = float64(r.Deaths)
		assists[i] = float64(r.Assists)
		kda[i] = fv(r.KDA)
		accuracy[i] = fv(r.Accuracy)
		score[i] = fv(r.PersonalScore)
		damageDealt[i] = fv(r.DamageDealt)
		damageTaken[i] = fv(r.DamageTaken)
		headshots[i] = fi(r.HeadshotKills)
		spree[i] = fi(r.MaxKillingSpree)
		perfect[i] = float64(r.PerfectKills)
	}

	return []scoreboardCol{
		{kills, false},
		{deaths, true},
		{assists, false},
		{kda, false},
		{accuracy, false},
		{score, false},
		{damageDealt, false},
		{damageTaken, true},
		{headshots, false},
		{spree, false},
		{perfect, false},
	}
}

func colMinMax(vals []float64) (min, max float64) {
	min, max = math.NaN(), math.NaN()
	for _, v := range vals {
		if math.IsNaN(v) {
			continue
		}
		if math.IsNaN(min) || v < min {
			min = v
		}
		if math.IsNaN(max) || v > max {
			max = v
		}
	}
	return
}

func pickBest(humans []domain.ScoreboardRaw, counts []int, minCount int) string {
	bestIdx, bestVal := -1, 0
	for i, c := range counts {
		if c > bestVal {
			bestVal = c
			bestIdx = i
		}
	}
	if bestIdx < 0 || bestVal < minCount {
		return ""
	}
	return humans[bestIdx].XUID
}
