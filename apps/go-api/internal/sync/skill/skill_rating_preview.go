package skill

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
)

type LUSRPlaylistPreview struct {
	PlaylistGroup string
	OldMU         float64 // rating persisté actuel (0 si pas de seed)
	OldSigma      float64
	NewMU         float64 // dernier rating qui serait écrit
	NewSigma      float64
	MatchCount    int                // nombre de matchs qui contribueraient à ce playlist_group
	ComponentAvgs map[string]float64 // moyenne par composante sur tous les matchs du groupe
}

// DeltaMU retourne NewMU - OldMU. Positif = LUSR remonte (joueur sous-évalué
// avant), négatif = LUSR descend.
func (p LUSRPlaylistPreview) DeltaMU() float64 { return p.NewMU - p.OldMU }

// LUSRDryRunReport agrège le résultat d'une exécution dry-run.
type LUSRDryRunReport struct {
	XUID             string
	MatchesProcessed int
	Playlists        []LUSRPlaylistPreview
}

// HasChanges retourne true si au moins un playlist_group montre un delta
// significatif (> 1.0 MU pour filtrer le bruit numérique).
func (r *LUSRDryRunReport) HasChanges() bool {
	for _, p := range r.Playlists {
		if absF(p.DeltaMU()) > 1.0 {
			return true
		}
	}
	return false
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// BatchComputeLUSRPreview est le pendant dry-run de BatchComputeLUSRWithMedals :
// reproduit les étapes 1-6 (load + compute) mais court-circuite l'écriture
// (étape 7). À la place, agrège un LUSRDryRunReport qui compare l'état
// persisté à l'état qui serait écrit.
//
// Toujours en mode force=true (recompute depuis zéro pour pouvoir comparer
// l'ensemble du résultat avec l'état actuel — un dry-run incrémental serait
// trivialement vide).
func BatchComputeLUSRPreview(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	medalExploitByMatch map[string]float64,
) (*LUSRDryRunReport, error) {
	report := &LUSRDryRunReport{XUID: xuid}

	// Étapes 1-2 identiques à BatchComputeLUSRWithMedals.
	matches, err := loadLUSRMatchData(ctx, sharedDB, xuid)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return report, nil
	}
	excluded, err := LoadExcludedMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, fmt.Errorf("BatchComputeLUSRPreview: %w", err)
	}
	if len(excluded) > 0 {
		filtered := matches[:0]
		for _, m := range matches {
			if !excluded[m.MatchID] {
				filtered = append(filtered, m)
			}
		}
		matches = filtered
		if len(matches) == 0 {
			return report, nil
		}
	}
	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}
	participantsByMatch, err := loadLUSRParticipants(ctx, sharedDB, matchIDs)
	if err != nil {
		return nil, err
	}

	// État actuel (persisté) — comparaison du Before.
	oldStates := loadExistingLUSRStates(ctx, playerDB)

	// Recompute depuis zéro (force=true, pas de seed).
	results := computeSkillRatingsBatch(ctxkeys.TitleSlug(ctx), matches, participantsByMatch,
		map[string]*PlayerState{}, medalExploitByMatch)

	// Agréger par playlist_group : dernier résultat chronologique = état final.
	// computeSkillRatingsBatch maintient un état interne mais ne le retourne pas ;
	// on reconstruit l'état final en prenant le rating du DERNIER match traité
	// par playlist_group (les matches sont déjà triés chronologiquement par
	// loadLUSRMatchData → COALESCE start_time_utc).
	finalByPG := make(map[string]*lusrResult, len(results))
	countByPG := make(map[string]int, len(results))
	compSums := make(map[string]map[string]float64)
	compCounts := make(map[string]map[string]int)
	for i := range results {
		r := &results[i]
		finalByPG[r.PlaylistGroup] = r
		countByPG[r.PlaylistGroup]++
		if compSums[r.PlaylistGroup] == nil {
			compSums[r.PlaylistGroup] = make(map[string]float64)
			compCounts[r.PlaylistGroup] = make(map[string]int)
		}
		for comp, val := range r.Components {
			compSums[r.PlaylistGroup][comp] += val
			compCounts[r.PlaylistGroup][comp]++
		}
	}

	// Construire le rapport — un PlaylistPreview par playlist_group.
	report.MatchesProcessed = len(results)
	for pg, r := range finalByPG {
		avgs := make(map[string]float64, len(compSums[pg]))
		for comp, sum := range compSums[pg] {
			avgs[comp] = sum / float64(compCounts[pg][comp])
		}
		preview := LUSRPlaylistPreview{
			PlaylistGroup: pg,
			NewMU:         r.RatingValue,
			NewSigma:      r.RatingDeviation,
			MatchCount:    countByPG[pg],
			ComponentAvgs: avgs,
		}
		if old, ok := oldStates[pg]; ok && old != nil {
			preview.OldMU = old.MU
			preview.OldSigma = old.Sigma
		}
		report.Playlists = append(report.Playlists, preview)
	}
	return report, nil
}

// computeSkillRatingsBatch calcule mu/sigma pour chaque match séquentiellement.
// medalExploitByMatch : map optionnelle match_id → score brut médailles.
