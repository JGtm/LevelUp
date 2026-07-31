// Package analysis — comeback_objective.go : source de courbe de score ALTERNATIVE
// pour les modes objectif (CTF, Strongholds, KOTH, Oddball).
//
// Les modes objectif ne se gagnent pas aux kills : la courbe de score pertinente
// pour le dominance_flag vient des events objectif (captures, prises de zone…),
// pas de highlight_events. Ce fichier fournit la construction des ScoreSnapshot
// depuis ces events ; la détection du flag reste ComputeDominanceFlag (réutilisée
// telle quelle, MÊME forme de courbe que BuildScoreSnapshots côté kills).
package analysis

import "sort"

// ObjectiveScoreEvent est un event de score objectif neutre, découplé de
// internal/domain : l'appelant (package sync) mappe ses domain.ObjectiveEvent
// vers ce type avant d'appeler BuildObjectiveScoreSnapshots.
//
//   - TimeMS : instant du match en millisecondes.
//   - TeamID : équipe créditée (0 ou 1 ; tout autre TeamID est ignoré).
//   - Value  : points marqués par l'event (ex. +1 par capture CTF ; <=0 ignoré).
type ObjectiveScoreEvent struct {
	TimeMS int64
	TeamID int
	Value  int
}

// BuildObjectiveScoreSnapshots reconstruit la courbe de score d'un mode objectif
// depuis ses events, dans la MÊME forme que BuildScoreSnapshots (kills).
//
// Les events sont triés par TimeMS ASC, puis cumulés : un snapshot initial 0-0
// est émis, puis pour chaque event valide (TeamID ∈ {0,1} et Value > 0) la Value
// est ajoutée au score de l'équipe et un snapshot cumulé est émis. Les events
// hors {0,1} ou de Value <= 0 sont ignorés (aucun snapshot émis pour eux).
//
// Renvoie nil si events est vide. Le résultat est directement consommable par
// ComputeDominanceFlag (même package, même type ScoreSnapshot).
func BuildObjectiveScoreSnapshots(events []ObjectiveScoreEvent) []ScoreSnapshot {
	if len(events) == 0 {
		return nil
	}

	// Copie locale pour ne pas muter le slice de l'appelant lors du tri.
	sorted := make([]ObjectiveScoreEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TimeMS < sorted[j].TimeMS
	})

	snapshots := make([]ScoreSnapshot, 0, len(sorted)+1)
	// Snapshot initial (score 0-0 au début du match).
	snapshots = append(snapshots, ScoreSnapshot{TimeMS: 0, Team0Score: 0, Team1Score: 0})

	team0 := 0
	team1 := 0
	for _, e := range sorted {
		if e.Value <= 0 {
			continue
		}
		switch e.TeamID {
		case 0:
			team0 += e.Value
		case 1:
			team1 += e.Value
		default:
			continue // TeamID hors {0,1} : ignoré.
		}
		snapshots = append(snapshots, ScoreSnapshot{
			TimeMS:     e.TimeMS,
			Team0Score: team0,
			Team1Score: team1,
		})
	}
	return snapshots
}
