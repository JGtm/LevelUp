// Package service — squad_service_v2_history.go : helper Tableau historique
// pour la page Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE Phase P9, sections audit
// "tableau historique").
//
// Le tableau liste les matchs partages du squad avec : date, mode, carte,
// outcome (du main), durée, et stats par joueur (K/D/A) pour chaque membre.
//
// Le helper consomme l'intersection deja calculee (sharedMatches +
// rowsByPlayer) — pas d'acces DB requis.
package service

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// HistoryTableRow est une ligne du tableau historique Squad V2.
//
// PlayerStats est indexee par gamertag, contient les K/D/A du joueur sur ce
// match. Les joueurs absents du match (ou leur stats nil) ne sont pas
// inseres dans la map (le rendu front affiche "-" sur les colonnes absentes).
type HistoryTableRow struct {
	MatchID         string                       `json:"match_id"`
	StartedAtUTC    time.Time                    `json:"started_at_utc"`
	DurationSeconds *int                         `json:"duration_seconds,omitempty"`
	// GameplayDurationSeconds : durée réelle de gameplay (countdown retranché),
	// préférée à DurationSeconds par le front pour un affichage homogène.
	GameplayDurationSeconds *int                 `json:"gameplay_duration_seconds,omitempty"`
	MapLabel        string                       `json:"map_label,omitempty"`
	ModeLabel       string                       `json:"mode_label,omitempty"`
	PlaylistLabel   string                       `json:"playlist_label,omitempty"`
	MainOutcome     canonical.Outcome            `json:"main_outcome"`
	PlayerStats     map[string]HistoryPlayerCell `json:"player_stats"`
}

// HistoryPlayerCell est la cellule par joueur du tableau historique.
type HistoryPlayerCell struct {
	Kills   *int     `json:"kills,omitempty"`
	Deaths  *int     `json:"deaths,omitempty"`
	Assists *int     `json:"assists,omitempty"`
	KDA     *float64 `json:"kda,omitempty"`
	Outcome string   `json:"outcome,omitempty"` // "win"/"loss"/"tie"/"dnf"
}

// BuildHistoryTable construit le tableau historique des matchs partages
// (intersection du squad).
//
//	sharedMatches : ordonne le tableau (date desc apres tri).
//	rowsByPlayer  : indexe par gamertag, fournit les stats par joueur par
//	                match (lookup via PlayerMatchRow.Summary.MatchID).
//	squadOrder    : conservee dans HistoryTableRow.PlayerStats (les keys de
//	                la map suivent cet ordre cote front).
//
// Tri retourne : StartedAt desc (matchs recents en haut). Stable.
func BuildHistoryTable(
	sharedMatches []domain.SquadSharedMatch,
	rowsByPlayer map[string][]canonical.PlayerMatchRow,
	squadOrder []string,
) []HistoryTableRow {
	if len(sharedMatches) == 0 {
		return nil
	}
	// Indexer rowsByPlayer par (gt, match_id).
	rowByGTMatch := make(map[string]map[string]canonical.PlayerMatchRow, len(rowsByPlayer))
	for gt, rows := range rowsByPlayer {
		byID := make(map[string]canonical.PlayerMatchRow, len(rows))
		for _, r := range rows {
			byID[r.Summary.MatchID] = r
		}
		rowByGTMatch[gt] = byID
	}

	out := make([]HistoryTableRow, 0, len(sharedMatches))
	for _, sm := range sharedMatches {
		row := HistoryTableRow{
			MatchID:      sm.MatchID,
			StartedAtUTC: sm.StartedAt,
			MainOutcome:  sm.Outcome,
			PlayerStats:  make(map[string]HistoryPlayerCell, len(squadOrder)),
		}
		// Hydrater map / mode / playlist depuis le row du main si dispo.
		if mainRows := rowByGTMatch[firstNonEmpty(squadOrder)]; mainRows != nil {
			if m, ok := mainRows[sm.MatchID]; ok {
				if d := m.Summary.DurationSeconds; d != nil {
					dCopy := *d
					row.DurationSeconds = &dCopy
				}
				if gp := m.Summary.GameplayDurationSeconds(); gp != nil {
					row.GameplayDurationSeconds = gp
				}
				if m.Summary.Map != nil {
					row.MapLabel = m.Summary.Map.DefaultLabel
				}
				if m.Summary.GameVariant != nil {
					row.ModeLabel = m.Summary.GameVariant.DefaultLabel
				}
				if m.Summary.Playlist != nil {
					row.PlaylistLabel = m.Summary.Playlist.DefaultLabel
				}
			}
		}
		// Cellules par joueur.
		for _, gt := range squadOrder {
			byID := rowByGTMatch[gt]
			pr, ok := byID[sm.MatchID]
			if !ok {
				continue
			}
			cell := HistoryPlayerCell{
				Kills:   copyIntPtr(pr.Self.Kills),
				Deaths:  copyIntPtr(pr.Self.Deaths),
				Assists: copyIntPtr(pr.Self.Assists),
				KDA:     copyFloatPtr(pr.Self.KDA),
				Outcome: string(pr.Self.Outcome),
			}
			row.PlayerStats[gt] = cell
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAtUTC.After(out[j].StartedAtUTC)
	})
	return out
}

func firstNonEmpty(s []string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

func copyIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyFloatPtr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
