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
// La colonne kills utilisée ici est ajustée par mvpKills (frags bruts moins
// assassinat / charge spartane / coup au sol) — seule la valeur de départage
// MVP/LVP est concernée, pas les frags affichés.
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
				bestCount[i] += c.weight
			case worst:
				worstCount[i] += c.weight
			}
		}
	}

	mvp := pickBest(humans, bestCount, mvpMinCells)
	lvp := pickBest(humans, worstCount, mvpMinCells)
	if mvp != "" && lvp != "" && mvp == lvp {
		mvp = pickBestExcluding(humans, bestCount, mvpMinCells, lvp)
	}
	return ScoreboardExtremes{MVPXUID: mvp, LVPXUID: lvp}
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

type scoreboardCol struct {
	vals     []float64
	inverted bool
	weight   int
}

// mvpKills retourne le nombre de frags retenu pour DÉPARTAGER le MVP/LVP :
// frags bruts moins les mécaniques de kill exclues (assassinat, charge spartane
// / shoulder_bash, coup au sol / ground_pound). Ces mécaniques ne reflètent pas
// la performance de tir qui doit peser dans le badge MVP/LVP.
//
// N'affecte QUE la valeur de départage : les frags par-match affichés ailleurs
// (colonne Frags du scoreboard) restent inchangés. Title-agnostic : ces trois
// champs sont nil hors Halo 5 (Infinite ne les fournit pas) → aucun effet sur
// Infinite. Le résultat est borné à 0 (défense en profondeur contre une donnée
// incohérente où la somme des mécaniques dépasserait les frags bruts).
func mvpKills(r domain.ScoreboardRaw) int {
	k := r.Kills - intPtrOrZero(r.AssassinationKills) -
		intPtrOrZero(r.ShoulderBashKills) - intPtrOrZero(r.GroundPoundKills)
	if k < 0 {
		return 0
	}
	return k
}

// intPtrOrZero déréférence un *int nullable en retournant 0 si nil.
func intPtrOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
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
		kills[i] = float64(mvpKills(r))
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
		{kills, false, 2},
		{deaths, true, 2},
		{assists, false, 1},
		{kda, false, 3},
		{accuracy, false, 1},
		{score, false, 3},
		{damageDealt, false, 2},
		{damageTaken, true, 2},
		{headshots, false, 1},
		{spree, false, 1},
		{perfect, false, 1},
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

func pickBestExcluding(humans []domain.ScoreboardRaw, counts []int, minCount int, excludeXUID string) string {
	bestIdx, bestVal := -1, 0
	for i, c := range counts {
		if humans[i].XUID == excludeXUID {
			continue
		}
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
