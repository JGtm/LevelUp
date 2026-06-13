package assetnames

import (
	"context"
	"strings"
)

// resolveOne résout un asset pour toutes les langues demandées. Retourne :
//   - resolved : ≥1 langue a été fetchée puis upsertée avec succès
//   - skipped  : toutes les langues étaient déjà présentes (aucun fetch)
//   - errCount : nombre d'erreurs fetch/upsert (niveau langue)
//
// Best-effort : une erreur (ou un nom vide) sur une langue n'interrompt pas les
// autres. Un nom vide n'est PAS une erreur (l'asset peut ne pas avoir de libellé
// localisé) — il n'est simplement pas écrit.
func resolveOne(
	ctx context.Context,
	f Fetcher,
	s Store,
	ref AssetRef,
	titleID string,
	langs []string,
) (resolved, skipped bool, errCount int) {
	allFresh := true
	for _, lang := range langs {
		if ctx.Err() != nil {
			allFresh = false
			break
		}
		fresh, err := s.ExistsFresh(ctx, ref.AssetType, ref.AssetID, lang)
		if err != nil {
			errCount++
			allFresh = false
			continue
		}
		if fresh {
			continue
		}
		allFresh = false

		name, ferr := f.FetchName(ctx, ref.AssetType, titleID, ref.AssetID, ref.VersionID, lang)
		if ferr != nil {
			errCount++
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue // pas de nom localisé pour cette langue — non bloquant
		}
		if uerr := s.Upsert(ctx, ref.AssetType, ref.AssetID, lang, name); uerr != nil {
			errCount++
			continue
		}
		resolved = true
	}
	skipped = allFresh && !resolved
	return resolved, skipped, errCount
}
