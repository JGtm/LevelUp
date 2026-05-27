//go:build cgo

// Replay LUSR v1 avec capture des diagnostics Phase 0.
//
// Le replay est volontairement une copie locale de la logique sync/skill_rating.go
// (et de cmd/diag_lusr_player/replay.go) pour deux raisons :
//  1. les fonctions internes du package sync (trueskillUpdate, computeEnemyStrength)
//     ne sont pas exportées ;
//  2. on a besoin d'intercepter mu_before / mu_opp à chaque match, ce que le
//     batch de prod ne fait pas.
//
// Math identique au prod : InitialMU, Beta, KElo, Tau, etc. importés depuis
// internal/sync via les constantes publiques.
package main

import (
	"database/sql"
	"log/slog"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	lusync "levelup/go-api/internal/sync"
)

// ── Types & loaders SQL ────────────────────────────────────────────────────

type matchRow struct {
	matchID        string
	startTime      time.Time
	pairName       string
	outcome        *int
	kills          float64
	deaths         float64
	assists        float64
	killsExpected  float64
	deathsExpected float64
	damageDealt    float64
	damageTaken    float64
	accuracy       float64
	teamID         *int
}

type participantRow struct {
	xuid          string
	teamID        *int
	killsExpected float64
}

// loadMatches : mêmes filtres que sync.loadLUSRMatchData (LUSR-éligible).
func loadMatches(db *sql.DB, xuid string) []matchRow {
	rows, err := db.Query(`
		SELECT
			mr.match_id, mr.start_time, COALESCE(mr.pair_name, ''),
			mp.outcome, COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
			COALESCE(mp.assists, 0),
			COALESCE(mp.kills_expected, 0), COALESCE(mp.deaths_expected, 0),
			COALESCE(mp.damage_dealt, 0), COALESCE(mp.damage_taken, 0),
			COALESCE(mp.accuracy, 0), mp.team_id
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		ORDER BY mr.start_time ASC`, xuid)
	if err != nil {
		slog.Error("loadMatches", "err", err, "xuid", xuid)
		return nil
	}
	defer rows.Close()
	var out []matchRow
	for rows.Next() {
		var m matchRow
		var outcome sql.NullInt64
		var teamID sql.NullInt64
		if err := rows.Scan(
			&m.matchID, &m.startTime, &m.pairName,
			&outcome, &m.kills, &m.deaths, &m.assists,
			&m.killsExpected, &m.deathsExpected,
			&m.damageDealt, &m.damageTaken,
			&m.accuracy, &teamID,
		); err != nil {
			continue
		}
		if outcome.Valid {
			v := int(outcome.Int64)
			m.outcome = &v
		}
		if teamID.Valid {
			v := int(teamID.Int64)
			m.teamID = &v
		}
		out = append(out, m)
	}
	return out
}

// loadParticipants : chunks 500 pour éviter binder param limit.
func loadParticipants(db *sql.DB, matchIDs []string) map[string][]participantRow {
	out := map[string][]participantRow{}
	if len(matchIDs) == 0 {
		return out
	}
	const chunk = 500
	for start := 0; start < len(matchIDs); start += chunk {
		end := start + chunk
		if end > len(matchIDs) {
			end = len(matchIDs)
		}
		batch := matchIDs[start:end]
		ph := strings.Repeat("?,", len(batch))
		ph = ph[:len(ph)-1]
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := db.Query(
			"SELECT match_id, xuid, team_id, COALESCE(kills_expected, 0) "+
				"FROM match_participants WHERE match_id IN ("+ph+")", args...)
		if err != nil {
			slog.Error("loadParticipants", "err", err, "chunk", start)
			continue
		}
		for rows.Next() {
			var mid string
			var p participantRow
			var teamID sql.NullInt64
			if err := rows.Scan(&mid, &p.xuid, &teamID, &p.killsExpected); err != nil {
				continue
			}
			if teamID.Valid {
				v := int(teamID.Int64)
				p.teamID = &v
			}
			out[mid] = append(out[mid], p)
		}
		rows.Close()
	}
	return out
}

// loadTrackedTeammateCounts : pour chaque match du joueur, retourne le nombre
// de coéquipiers AUTRES que lui dont l'xuid est dans le set tracké (= squad size proxy).
func loadTrackedTeammateCounts(db *sql.DB, playerXUID string, matchIDs []string, trackedXUIDs map[string]bool) map[string]int {
	out := map[string]int{}
	if len(matchIDs) == 0 || len(trackedXUIDs) == 0 {
		return out
	}
	// Pour chaque match : compter parts de team_id = team du joueur et xuid ∈ tracked \ {self}.
	// On utilise les rows déjà chargées via loadParticipants serait plus simple ; ici relecture indépendante.
	const chunk = 500
	for start := 0; start < len(matchIDs); start += chunk {
		end := start + chunk
		if end > len(matchIDs) {
			end = len(matchIDs)
		}
		batch := matchIDs[start:end]
		ph := strings.Repeat("?,", len(batch))
		ph = ph[:len(ph)-1]
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := db.Query(
			"SELECT match_id, xuid, team_id FROM match_participants WHERE match_id IN ("+ph+")", args...)
		if err != nil {
			slog.Error("loadTrackedTeammateCounts", "err", err)
			continue
		}
		type mpKey struct{ matchID string }
		playerTeam := map[mpKey]int{}
		others := map[mpKey][]struct {
			xuid   string
			teamID int
		}{}
		for rows.Next() {
			var mid, x string
			var tid sql.NullInt64
			if err := rows.Scan(&mid, &x, &tid); err != nil {
				continue
			}
			if !tid.Valid {
				continue
			}
			k := mpKey{mid}
			if x == playerXUID {
				playerTeam[k] = int(tid.Int64)
			} else if trackedXUIDs[x] {
				others[k] = append(others[k], struct {
					xuid   string
					teamID int
				}{x, int(tid.Int64)})
			}
		}
		rows.Close()
		for k, t := range playerTeam {
			for _, o := range others[k] {
				if o.teamID == t {
					out[k.matchID]++
				}
			}
		}
	}
	return out
}

// loadMatchDurations : durée du match en secondes (pour kill rate par minute).
func loadMatchDurations(db *sql.DB, matchIDs []string) map[string]int {
	out := map[string]int{}
	if len(matchIDs) == 0 {
		return out
	}
	const chunk = 500
	for start := 0; start < len(matchIDs); start += chunk {
		end := start + chunk
		if end > len(matchIDs) {
			end = len(matchIDs)
		}
		batch := matchIDs[start:end]
		ph := strings.Repeat("?,", len(batch))
		ph = ph[:len(ph)-1]
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := db.Query(
			"SELECT match_id, COALESCE(duration_seconds, 0) FROM match_registry "+
				"WHERE match_id IN ("+ph+")", args...)
		if err != nil {
			continue
		}
		for rows.Next() {
			var mid string
			var dur int
			if rows.Scan(&mid, &dur) == nil {
				out[mid] = dur
			}
		}
		rows.Close()
	}
	return out
}

// ── Replay & math ──────────────────────────────────────────────────────────

type localState struct {
	mu                  float64
	sigma               float64
	matchCount          int // dans la même chaine
	lastMatchTime       *time.Time
	accuracyHistory     []float64
	damageEffHistory    []float64
	offConvHistory      []float64
	defResHistory       []float64
}

func newState() *localState { return &localState{mu: lusync.InitialMU, sigma: lusync.InitialSigma} }

// replayAndCollect rejoue le LUSR v1 chronologiquement et émet une observation
// par match LUSR-éligible. Capture mu/sigma AVANT update + P(win) prédite.
func replayAndCollect(
	matches []matchRow,
	parts map[string][]participantRow,
	teammateCounts map[string]int,
	durations map[string]int,
	playerXUID, gamertag string,
) []observation {
	states := map[string]*localState{}
	totalPlayed := 0
	prevKillRate := math.NaN()
	out := make([]observation, 0, len(matches))

	for _, m := range matches {
		chain := lusync.GetLUSRChain(m.pairName)
		if chain == "" {
			continue
		}
		st, ok := states[chain]
		if !ok {
			st = newState()
			states[chain] = st
		}
		if st.lastMatchTime != nil {
			days := m.startTime.Sub(*st.lastMatchTime).Hours() / 24.0
			st.sigma = applyDecay(st.sigma, days)
		}

		pp := parts[m.matchID]
		muBefore, sigmaBefore := st.mu, st.sigma
		muOpp := estimateMuOpp(st.mu, m.teamID, pp)
		predicted := sigmoid((muBefore - muOpp) / (2.0 * lusync.Beta))
		actual := outcomeToActual(m.outcome)

		o := observation{
			gamertag:       gamertag,
			xuid:           playerXUID,
			matchID:        m.matchID,
			startTime:      m.startTime,
			chain:          chain,
			muBefore:       muBefore,
			sigmaBefore:    sigmaBefore,
			muOpp:          muOpp,
			predictedPWin:  predicted,
			actualWin:      actual,
			priorMatchTot:  totalPlayed,
			priorMatchMode: st.matchCount,
			teamSize:       1 + teammateCounts[m.matchID],
			prevKillRate:   prevKillRate,
		}
		out = append(out, o)

		// Update state via la même math que le prod (composite + trueskillUpdate).
		st.mu, st.sigma = trueskillStep(st, m, pp, muOpp)
		st.matchCount++
		t := m.startTime
		st.lastMatchTime = &t
		totalPlayed++
		prevKillRate = killRate(m, durations[m.matchID])
	}
	return out
}

func outcomeToActual(o *int) float64 {
	if o == nil {
		return 0
	}
	switch *o {
	case 2:
		return 1.0
	case 1:
		return 0.5
	default:
		return 0.0
	}
}

func killRate(m matchRow, durSec int) float64 {
	if durSec <= 0 {
		return math.NaN()
	}
	return m.kills / (float64(durSec) / 60.0)
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func applyDecay(sigma, days float64) float64 {
	capped := math.Min(days, float64(lusync.MaxInactivityDays))
	if capped <= lusync.InactivityThresholdDay {
		return sigma
	}
	added := lusync.InactivitySigmaPerDay * (capped - lusync.InactivityThresholdDay)
	return clamp(sigma+added, lusync.MinSigma, lusync.MaxSigma)
}

// estimateMuOpp : portage allégé de sync.computeEnemyStrength.
func estimateMuOpp(playerMU float64, playerTeam *int, parts []participantRow) float64 {
	enemyKEs := make([]float64, 0, len(parts))
	allKEs := make([]float64, 0, len(parts))
	for _, p := range parts {
		if p.killsExpected <= 0 {
			continue
		}
		allKEs = append(allKEs, p.killsExpected)
		if playerTeam != nil && p.teamID != nil && *p.teamID != *playerTeam {
			enemyKEs = append(enemyKEs, p.killsExpected)
		}
	}
	if len(enemyKEs) == 0 {
		return playerMU
	}
	avg, std := 0.0, 0.0
	if len(allKEs) > 0 {
		for _, k := range allKEs {
			avg += k
		}
		avg /= float64(len(allKEs))
		for _, k := range allKEs {
			d := k - avg
			std += d * d
		}
		std = math.Sqrt(std / float64(len(allKEs)))
	}
	sum := 0.0
	for _, ke := range enemyKEs {
		z := 0.0
		if std >= 1e-6 {
			z = (ke - avg) / std
		}
		sum += playerMU + lusync.IndividualMUAlpha*z
	}
	return sum / float64(len(enemyKEs))
}

// trueskillStep : reproduit le pipeline composite + trueskillUpdate du prod.
func trueskillStep(st *localState, m matchRow, parts []participantRow, muOpp float64) (float64, float64) {
	avgAcc := rollingAvg(st.accuracyHistory)
	avgDmg := rollingAvg(st.damageEffHistory)
	avgOff := rollingAvg(st.offConvHistory)
	avgDef := rollingAvg(st.defResHistory)
	composite := computeComposite(m, avgAcc, avgDmg, avgOff, avgDef)
	newMU, newSigma := trueskillUpdate(st.mu, st.sigma, muOpp, composite)
	updateHistories(st, m)
	return newMU, newSigma
}

func updateHistories(st *localState, m matchRow) {
	appendHist(&st.accuracyHistory, m.accuracy)
	total := m.damageDealt + m.damageTaken
	if total > 0 {
		appendHist(&st.damageEffHistory, clamp(m.damageDealt/total, 0, 1))
	}
	if m.damageDealt > 0 {
		off := 225.0 * (m.kills + m.assists/3.0) / m.damageDealt
		if off > 0 {
			appendHist(&st.offConvHistory, off)
		}
	}
	if m.damageTaken > 0 && m.deaths > 0 {
		def := m.damageTaken / (225.0 * m.deaths)
		if def > 0 {
			appendHist(&st.defResHistory, def)
		}
	}
}

// computeComposite : portage allégé du composite v1 (sans medal_exploit ni carry-adj
// — négligeable pour la précision de Phase 0, et carry-adj n'utilise pas l'enemyAvgKE
// dont on n'a pas besoin ici).
func computeComposite(m matchRow, avgAcc, avgDmg, avgOff, avgDef *float64) float64 {
	w := lusync.CompositeWeights
	type entry struct{ score, weight float64 }
	entries := []entry{}
	if m.killsExpected > 0 {
		entries = append(entries, entry{sigRatio(m.kills, m.killsExpected), w[lusync.MetricKeyKillsVsExpected]})
	}
	if m.deathsExpected > 0 {
		entries = append(entries, entry{sigRatio(m.deathsExpected, math.Max(1.0, m.deaths)), w[lusync.MetricKeyDeathsVsExpected]})
	}
	if m.outcome != nil {
		var s float64
		switch *m.outcome {
		case 2:
			s = 1.0
		case 1:
			s = 0.5
		case 3:
			s = 0.0
		case 4:
			s = 0.15
		default:
			s = 0.5
		}
		entries = append(entries, entry{s, w[lusync.MetricKeyWinFactor]})
	}
	total := m.damageDealt + m.damageTaken
	if total > 0 {
		raw := clamp(m.damageDealt/total, 0.0, 1.0)
		s := raw
		if avgDmg != nil && *avgDmg > 0 {
			s = sigRatio(raw, *avgDmg)
		}
		entries = append(entries, entry{s, w[lusync.MetricKeyDamageEfficiency]})
	}
	if m.accuracy > 0 && avgAcc != nil && *avgAcc > 0 {
		entries = append(entries, entry{sigRatio(m.accuracy, *avgAcc), w[lusync.MetricKeyAccuracyDelta]})
	}
	if m.damageDealt > 0 {
		off := 225.0 * (m.kills + m.assists/3.0) / m.damageDealt
		if off > 0 {
			ref := analysis.OffensiveConversionP80
			if avgOff != nil && *avgOff > 1e-9 {
				ref = *avgOff
			}
			entries = append(entries, entry{sigRatio(off, ref), w[lusync.MetricKeyOffensiveConv]})
		}
	}
	if m.damageTaken > 0 && m.deaths > 0 {
		def := m.damageTaken / (225.0 * m.deaths)
		if def > 0 {
			ref := analysis.DefensiveResistanceP80
			if avgDef != nil && *avgDef > 1e-9 {
				ref = *avgDef
			}
			entries = append(entries, entry{sigRatio(def, ref), w[lusync.MetricKeyDefensiveResist]})
		}
	}
	if len(entries) == 0 {
		return 0.5
	}
	sum, tw := 0.0, 0.0
	for _, e := range entries {
		sum += e.score * e.weight
		tw += e.weight
	}
	if tw < 1e-12 {
		return 0.5
	}
	return clamp(sum/tw, 0.0, 1.0)
}

func trueskillUpdate(mu, sigma, muOpp, actualScore float64) (float64, float64) {
	expected := 1.0 / (1.0 + math.Exp(-(mu-muOpp)/(2.0*lusync.Beta)))
	deltaMU := lusync.KElo * (actualScore - expected)
	newMU := math.Max(lusync.MinRating, mu+deltaMU)

	const sigmaOpp = lusync.DefaultOpponentSigma
	c2 := 2.0*lusync.Beta*lusync.Beta + sigma*sigma + sigmaOpp*sigmaOpp
	c := math.Sqrt(c2)
	eps := drawMargin(lusync.Beta)
	w := wWin(0.0, eps/c)
	sigma2 := sigma * sigma
	deltaSig2 := sigma2 * (sigma2 / c2) * w

	newSig := math.Sqrt(math.Max(lusync.MinSigma*lusync.MinSigma, sigma2-deltaSig2))
	newSig = math.Min(math.Sqrt(newSig*newSig+lusync.Tau*lusync.Tau), lusync.MaxSigma)
	return newMU, newSig
}

func drawMargin(beta float64) float64 {
	const drawProb = 0.06
	p := (drawProb + 1.0) / 2.0
	if p >= 1.0 {
		return 8.0 * beta
	}
	return math.Sqrt(-2.0*math.Log(1.0-p)) * beta
}

func wWin(t, eps float64) float64 {
	v := vWin(t, eps)
	x := t - eps
	return v * (v + x)
}

func vWin(t, eps float64) float64 {
	x := t - eps
	denom := 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
	if denom < 1e-10 {
		return -x
	}
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi) / denom
}

func rollingAvg(h []float64) *float64 {
	if len(h) < lusync.MinMatchesForAccuracyDelta {
		return nil
	}
	s := 0.0
	for _, v := range h {
		s += v
	}
	a := s / float64(len(h))
	return &a
}

func appendHist(h *[]float64, v float64) {
	*h = append(*h, v)
	if len(*h) > lusync.AccuracyHistorySize {
		*h = (*h)[len(*h)-lusync.AccuracyHistorySize:]
	}
}

func sigRatio(num, denom float64) float64 {
	if denom <= 0 {
		return 0.5
	}
	r := num / denom
	return clamp(r/(1.0+r), 0.0, 1.0)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
