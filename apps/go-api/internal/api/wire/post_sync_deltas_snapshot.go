// Package api — post_sync_deltas_snapshot.go : capture de l'état joueur
// (PlayerSnapshot) avant/après sync. Extrait de post_sync_deltas.go (refactor god-file).
package wire

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// PlayerSnapshot capture l'état pertinent d'un joueur pour la détection delta (type du port, lu par
// duckdb.PlayerSnapshotStore).
type PlayerSnapshot = port.PlayerSnapshot

// SnapshotPlayerState lit l'état courant nécessaire à la détection delta.
// Robuste aux tables vides ou colonnes manquantes (renvoie zero-values).
//
// `citationsSvc` peut être nil — dans ce cas les compteurs citation_tier /
// citation_mastery restent à 0 (pas d'émission).
func SnapshotPlayerState(
	ctx context.Context,
	pdb *duckdb.PlayerDB,
	citationsSvc port.CitationsService,
) (*PlayerSnapshot, error) {
	if pdb == nil || pdb.Player == nil {
		return &PlayerSnapshot{}, nil
	}
	s := &PlayerSnapshot{SkillTierByPlaylist: map[string]string{}}
	var reader port.PlayerSnapshotReader = duckdb.NewPlayerSnapshotStore(pdb)

	// Rang carrière, awards, citations brutes, défis, niveaux de classement, battle pass (base du joueur).
	reader.FillPlayerCounters(ctx, s)

	// Citations / commendations agrégées via le service (réutilise la chaîne
	// Q34 + Q35 + MergeCitationTotals → garantit que la sémantique tiers/mastery
	// est identique à celle de la page Citations). citationsSvc peut être nil.
	if citationsSvc != nil {
		page, err := citationsSvc.GetCitationsPage(ctx)
		if err != nil {
			slog.DebugContext(ctx, "snapshot: citations page", "err", err)
		} else if page != nil {
			for _, item := range page.Citations {
				s.CitationTotalEarnedTiers += item.EarnedTiers
				if item.MasteryPct >= 100.0 {
					s.CitationMasteryCount++
				}
			}
		}
	}

	// KD agrégé, winrate, meilleur KDA, dernier match, médailles (base partagée — SharedReader, ADR 0016).
	reader.FillSharedAggregates(ctx, s)

	return s, nil
}

// thresholdCrossed retourne true si une métrique est passée au-dessus d'un palier
// (granularité step) entre deux snapshots, vers le haut uniquement.
//
// Exemple : before=0.99, after=1.04, step=0.05 → crosses 1.00 → true.
//
//	before=1.04, after=0.99, step=0.05 → descente → false.
