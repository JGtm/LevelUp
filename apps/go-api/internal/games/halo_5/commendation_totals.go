package halo_5

// commendation_totals.go — TOTAUX à vie des commendations natives Halo 5 d'un joueur
// (AXE B, suite « persister le Progress absolu »). Le total courant d'une commendation
// = le progress ABSOLU du match le plus récent (lu via CommendationTotalsSource depuis
// shared.match_commendations), enrichi nom/icône/catégorie via les définitions
// (CommendationDefSource). Réponse PLATE (le handler groupe par catégorie).
//
// Interface définie côté package consommateur (h5), retour canonique : l'impl DuckDB
// la satisfait structurellement, zéro cycle (parité MatchHistorySource).

import (
	"context"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// CommendationTotalsSource lit les totaux à vie par commendation (dernier progress
// absolu) d'un joueur player-bound, depuis le shared h5 (read-only).
type CommendationTotalsSource interface {
	GetCommendationTotals(ctx context.Context) ([]canonical.CommendationTotal, error)
}

// LoadCommendationTotals retourne les totaux à vie des commendations natives du joueur
// (source player-bound), enrichis nom/icône/catégorie via les définitions. Source nil
// -> ErrCapabilityNotSupported (capability non câblée). Définitions absentes / lookup
// en erreur -> totaux BRUTS (ID + total, le front dégrade sur l'ID court).
func (a *DataAdapter) LoadCommendationTotals(ctx context.Context) ([]canonical.CommendationTotal, error) {
	if a.commendationTotals == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	totals, err := a.commendationTotals.GetCommendationTotals(ctx)
	if err != nil {
		return nil, err
	}
	a.enrichCommendationTotals(ctx, totals)
	return totals, nil
}

// enrichCommendationTotals peuple Name/IconURL/Category des totaux via les définitions
// (best-effort). N'écrase jamais un Name par du vide.
func (a *DataAdapter) enrichCommendationTotals(ctx context.Context, totals []canonical.CommendationTotal) {
	if a.commendationDefs == nil || len(totals) == 0 {
		return
	}
	ids := make([]string, 0, len(totals))
	for i := range totals {
		if totals[i].ID != "" {
			ids = append(ids, totals[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	defs, err := a.commendationDefs.LookupCommendations(ctx, ids)
	if err != nil {
		a.logger.DebugContext(ctx, "h5 LoadCommendationTotals: lookup défs échoué (totaux bruts)", "err", err)
		return
	}
	for i := range totals {
		d, ok := defs[totals[i].ID]
		if !ok {
			continue
		}
		if d.Name != "" {
			totals[i].Name = d.Name
		}
		totals[i].Category = d.Category
		totals[i].TierTargets = d.TierTargets
		if d.IconURL != "" {
			icon := d.IconURL
			totals[i].IconURL = &icon
		}
	}
}
