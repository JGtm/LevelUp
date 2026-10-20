// Package ops - seed_demo_campaign.go : exclusion de la Campagne du corpus demo.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
)

// excludeCampaign retire de chaque voie du corpus (solo, escouade, classe, media) les matchs
// de Campagne du titre : un seul point pour l'ensemble que selectDynamicCorpus rend, quelle que
// soit la voie qui l'a retenu (un clip ou une session escouade peut ancrer un match de
// Campagne). Titre sans variante masquee : ensemble inchange. Echec de lecture : la voie est
// videe (jamais de Campagne dans la demo) apres un WARN.
func (dc *dynamicCorpus) excludeCampaign(ctx context.Context, sharedDBPath, titleSlug string) {
	for _, lane := range []struct {
		name string
		ids  *[]string
	}{{"solo", &dc.solo}, {"squad", &dc.squad}, {"ranked", &dc.ranked}, {"media", &dc.media}} { //nolint:goconst // nom de voie du corpus démo, distinct de playlist_group (synthGroupRanked)
		kept, err := dropCampaignMatchIDs(ctx, sharedDBPath, titleSlug, *lane.ids)
		if err != nil {
			slog.WarnContext(ctx, "seed-demo: exclusion de la Campagne impossible, voie videe",
				"lane", lane.name, "err", err)
			*lane.ids = nil
			continue
		}
		*lane.ids = kept
	}
}

// dropCampaignMatchIDs rend ids sans les matchs que le registre partage classe en Campagne
// pour titleSlug (un match absent du registre est conserve : rien ne dit qu'il en est un).
func dropCampaignMatchIDs(ctx context.Context, sharedDBPath, titleSlug string, ids []string) ([]string, error) {
	variants := analysis.CampaignExcludedVariantIDs(titleSlug)
	if len(ids) == 0 || len(variants) == 0 {
		return ids, nil
	}
	db, err := sql.Open("duckdb", sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open shared: %w", err)
	}
	defer db.Close()

	args := make([]any, 0, len(ids)+len(variants))
	for _, id := range ids {
		args = append(args, id)
	}
	for _, v := range variants {
		args = append(args, v)
	}
	rows, err := db.QueryContext(ctx, `SELECT match_id FROM match_registry
		WHERE match_id IN (`+placeholders(len(ids))+`)
		  AND game_variant_id IN (`+placeholders(len(variants))+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("query campagne: %w", err)
	}
	campaign, err := scanMatchIDColumn(rows)
	if err != nil {
		return nil, err
	}
	banned := make(map[string]struct{}, len(campaign))
	for _, id := range campaign {
		banned[id] = struct{}{}
	}
	kept := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, drop := banned[id]; !drop {
			kept = append(kept, id)
		}
	}
	return kept, nil
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
