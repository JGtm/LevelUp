package service

import (
	"context"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// labelSkillRank pose le libellé de palier d'un rang de la vue du match, DÉDUIT de son code (Tier,
// SubTier) dans la langue de la requête. Les lecteurs ne renseignent pas SkillRankRaw.TierLabel : la
// colonne stockée `tier_label` est écrite dans la langue du moment de l'écriture (« Or VI », « Diamond
// 4 ») et ne se sert jamais telle quelle. Une ligne en placement porte le marqueur machine
// domain.TierPlacement, que le web remplace par son propre texte. nil sans effet.
func labelSkillRank(ctx context.Context, r *domain.SkillRankRaw) {
	if r == nil {
		return
	}
	r.TierLabel = analysis.SkillTierLabelOrPlacementMarker(strDeref(r.Tier), r.SubTier, i18n.FromContext(ctx))
}

// labelSkillRanks applique labelSkillRank à chaque rang d'une table par joueur.
func labelSkillRanks(ctx context.Context, ranks map[string]*domain.SkillRankRaw) {
	for _, r := range ranks {
		labelSkillRank(ctx, r)
	}
}
