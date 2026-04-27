// Package analysis — comeback.go : détection des badges narratifs de match.
//
// Port Go de src/analysis/comeback_analysis.py.
// Calcule le dominance_flag (0–5) depuis la courbe de score reconstruite
// à partir des kill-events de highlight_events.
//
// Valeurs de dominance_flag :
//
//	0 — Aucun badge
//	1 — DOMINATION   : victoire avec avance constante ≥ seuil
//	2 — HUMILIATION  : défaite avec retard constant ≥ seuil
//	3 — REMONTADA    : victoire après avoir été mené ≥ seuil
//	4 — DÉBÂCLE      : défaite après avoir mené ≥ seuil
//	5 — CONTRE-REMONTADA : victoire malgré un second retard adverse
package analysis

import (
	"log/slog"
)

// DominanceFlagNone indique l'absence de badge narratif.
const DominanceFlagNone = 0

// Constantes de dominance_flag (valeurs Halo Infinite).
const (
	DominanceFlagDomination      = 1
	DominanceFlagHumiliation     = 2
	DominanceFlagRemontada       = 3
	DominanceFlagDebacle         = 4
	DominanceFlagContreRemontada = 5
)

// MedalSteaktacularID est l'ID de la médaille Steaktacular (5+ kills de suite).
// Déclenche DOMINATION (gagnée par mon équipe) ou HUMILIATION (gagnée par l'adversaire).
// Miroir de MEDAL_STEAKTACULAR_ID = 1169390319 dans src/analysis/_medal_verdicts.py.
const MedalSteaktacularID int64 = 1169390319

// Constantes d'outcome complémentaires (OutcomeWin=2, OutcomeLoss=3 déjà déclarées dans performance_score.go).
const (
	OutcomeTie = 1
	OutcomeDNF = 4
)

// KillEvent représente un kill tiré de highlight_events, avec le team_id résolu.
// Le team_id doit être résolu par la couche backfill depuis match_participants.
type KillEvent struct {
	TimeMS int64
	TeamID int // team_id du joueur qui a fait le kill
}

// ScoreSnapshot représente l'état du score à un instant t du match.
type ScoreSnapshot struct {
	TimeMS     int64
	Team0Score int
	Team1Score int
}

// sensitivityThresholds mappe une sensibilité sur [leadPct, comebackPct].
// leadPct : pourcentage du score final max pour déclencher Domination/Humiliation.
// comebackPct : amplitude de retournement pour Remontada/Débâcle/Contre-Remontada.
var sensitivityThresholds = map[string][2]float64{
	"relaxed":  {0.25, 0.20},
	"standard": {0.40, 0.35},
	"strict":   {0.60, 0.55},
}

// defaultSensitivity est la sensibilité par défaut si une valeur invalide est passée.
const defaultSensitivity = "standard"

// BuildScoreSnapshots reconstruit la courbe de score depuis les kill-events.
//
// events doit être trié par TimeMS ASC. Chaque kill incrémente le score de l'équipe
// du tueur. Un snapshot est émis avant le premier kill (score 0-0) puis après chaque
// kill. Le score des deux équipes est tracé avec Team0Score et Team1Score.
func BuildScoreSnapshots(events []KillEvent) []ScoreSnapshot {
	if len(events) == 0 {
		return nil
	}
	snapshots := make([]ScoreSnapshot, 0, len(events)+1)
	// Snapshot initial (score 0-0 au début du match)
	snapshots = append(snapshots, ScoreSnapshot{TimeMS: 0, Team0Score: 0, Team1Score: 0})

	team0 := 0
	team1 := 0
	for _, e := range events {
		if e.TeamID == 0 {
			team0++
		} else {
			team1++
		}
		snapshots = append(snapshots, ScoreSnapshot{
			TimeMS:     e.TimeMS,
			Team0Score: team0,
			Team1Score: team1,
		})
	}
	return snapshots
}

// ComputeDominanceFlag calcule le dominance_flag (0–5) à partir des snapshots.
//
// Paramètres :
//   - snapshots      : courbe de score (issue de BuildScoreSnapshots)
//   - playerTeamID   : team_id du joueur (0 ou 1)
//   - playerOutcome  : code de résultat Halo (2=Win, 3=Loss, 1=Tie, 4=DNF)
//   - sensitivity    : "relaxed" | "standard" | "strict"
//   - excludeBots    : si true et hadBotTeammate=true, retourne 0
//   - hadBotTeammate : indique la présence d'un bot côté joueur
//
// La détection respecte l'ordre de priorité :
// CONTRE_REMONTADA > REMONTADA > DÉBÂCLE > DOMINATION > HUMILIATION
func ComputeDominanceFlag(
	snapshots []ScoreSnapshot,
	playerTeamID int,
	playerOutcome int,
	sensitivity string,
	excludeBots bool,
	hadBotTeammate bool,
	matchID string, // pour le logging seulement
) int {
	if excludeBots && hadBotTeammate {
		slog.Debug("comeback: match exclu (bot teammate)",
			"match_id", matchID,
			"sensitivity", sensitivity,
		)
		return DominanceFlagNone
	}

	if len(snapshots) == 0 {
		slog.Warn("comeback: events vides, flag=0 par défaut", "match_id", matchID)
		return DominanceFlagNone
	}

	// Seuls Win et Loss sont analysés. Tie/DNF → 0.
	if playerOutcome != OutcomeWin && playerOutcome != OutcomeLoss {
		return DominanceFlagNone
	}

	thresholds := getThresholds(sensitivity)
	leadPct := thresholds[0]
	comebackPct := thresholds[1]

	// Score final (dernier snapshot) pour normaliser le seuil.
	finalSnap := snapshots[len(snapshots)-1]
	finalMax := max(finalSnap.Team0Score, finalSnap.Team1Score)
	if finalMax == 0 {
		return DominanceFlagNone
	}

	leadThreshold := float64(finalMax) * leadPct
	comebackThreshold := float64(finalMax) * comebackPct

	// Analyser tous les snapshots SAUF le dernier ("before end").
	var maxPlayerLead, maxEnemyLead float64
	hadEnemyLeadBeforeEnd := false
	hadPlayerLeadBeforeEnd := false

	n := len(snapshots)
	for i := 0; i < n-1; i++ {
		s := snapshots[i]
		pl := playerLead(s, playerTeamID)
		if pl > maxPlayerLead {
			maxPlayerLead = pl
		}
		if -pl > maxEnemyLead {
			maxEnemyLead = -pl
		}
		if -pl >= comebackThreshold {
			hadEnemyLeadBeforeEnd = true
		}
		if pl >= comebackThreshold {
			hadPlayerLeadBeforeEnd = true
		}
	}

	playerWon := playerOutcome == OutcomeWin

	// Priorité : CONTRE_REMONTADA > REMONTADA > DÉBÂCLE > DOMINATION > HUMILIATION
	flag := DominanceFlagNone
	switch {
	case playerWon && hadPlayerLeadBeforeEnd && hadEnemyLeadBeforeEnd:
		flag = DominanceFlagContreRemontada
	case playerWon && hadEnemyLeadBeforeEnd:
		flag = DominanceFlagRemontada
	case !playerWon && hadPlayerLeadBeforeEnd:
		flag = DominanceFlagDebacle
	case playerWon && maxPlayerLead >= leadThreshold:
		flag = DominanceFlagDomination
	case !playerWon && maxEnemyLead >= leadThreshold:
		flag = DominanceFlagHumiliation
	}

	slog.Debug("comeback: match analysé",
		"match_id", matchID,
		"flag", flag,
		"sensitivity", sensitivity,
		"had_bot", hadBotTeammate,
		"excluded", excludeBots && hadBotTeammate,
		"max_player_lead", maxPlayerLead,
		"max_enemy_lead", maxEnemyLead,
		"lead_threshold", leadThreshold,
		"comeback_threshold", comebackThreshold,
	)

	return flag
}

// getThresholds retourne [leadPct, comebackPct] pour la sensibilité donnée.
// Retourne les seuils "standard" si la sensibilité est invalide.
func getThresholds(sensitivity string) [2]float64 {
	if t, ok := sensitivityThresholds[sensitivity]; ok {
		return t
	}
	return sensitivityThresholds[defaultSensitivity]
}

// playerLead retourne le lead du joueur (positif = joueur mène, négatif = ennemi mène).
func playerLead(s ScoreSnapshot, playerTeamID int) float64 {
	if playerTeamID == 0 {
		return float64(s.Team0Score - s.Team1Score)
	}
	return float64(s.Team1Score - s.Team0Score)
}

// max retourne le plus grand des deux entiers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
