package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/platform/dblease"
)

// campaign_rebaseline_repo.go — lecture des campagnes en cours et remplacement de leur point de départ
// (recalage au démarrage, campaign.Service.ApplyObjectiveAxisRev).

// ListOpen liste les campagnes en cours (active ou paused) du joueur sur un titre, les plus récentes
// d'abord. Scopé user_id + title_slug ; l'index idx_campaign_user_title couvre le filtre.
func (r *CampaignRepo) ListOpen(ctx context.Context, userID, titleSlug string) ([]campaign.ImprovementCampaign, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, campaignSelectColumns+`
		WHERE user_id = ? AND title_slug = ?
		  AND status IN ('active', 'paused')
		ORDER BY started_at DESC
	`, userID, titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []campaign.ImprovementCampaign
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RestartMeasure remplace le point de départ d'une campagne en cours (valeur, échantillons, matchs
// depuis le départ) et remet son évaluation à vide : valeur courante brute et lissée, test de
// progression, date d'évaluation, confirmation et suggestion de clôture. Une campagne close (statut
// hors active et paused) n'est pas touchée.
//
// Écriture sous le verrou d'écriture de la base joueur (lease KindPlayer, ADR 0013) : sérialisée avec
// le sync. UPDATE de colonnes non indexées, filtré par la clé primaire (même motif que
// UpdateEvaluation) : ni UPSERT ni index touché (ADR 0019).
func (r *CampaignRepo) RestartMeasure(ctx context.Context, id string, start campaign.MeasureStart) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	w, err := dblease.AcquireWriter(r.db.db.SQLDb(), r.db.db.Path(), dblease.KindPlayer, dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("CampaignRepo.RestartMeasure: lease: %w", err)
	}
	defer w.Release()
	_, err = r.db.ExecRecovered(ctx, `
		UPDATE improvement_campaign
		SET snapshot_value = ?, snapshot_sample = ?, matches_since_start = ?,
		    current_value_raw = NULL, current_value_lowess = NULL,
		    last_evaluated_at = NULL, mann_whitney_p = NULL,
		    progression_confirmed = FALSE, auto_closure_suggested = FALSE,
		    auto_closure_reason = NULL
		WHERE id = ? AND status IN ('active', 'paused')
	`, start.SnapshotValue, start.SnapshotSample, start.MatchesSinceStart, id)
	if err != nil {
		return fmt.Errorf("CampaignRepo.RestartMeasure: %w", err)
	}
	return nil
}
