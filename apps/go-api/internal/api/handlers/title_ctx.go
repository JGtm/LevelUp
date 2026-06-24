// Package handlers — title_ctx.go : résolution title-aware PAR REQUÊTE du slug de
// titre pour les handlers progression/coach/profile/campaign.
//
// Ces handlers sont construits avec un slug de FALLBACK (typ. halo_infinite, le
// défaut historique mono-titre). Le middleware TitleExtractor pose le titre réel
// de la requête dans le contexte (X-LevelUp-Title → session → défaut). Les filtres
// de requête (catalog/history/earned/profile/campaign par title_id) doivent suivre
// CE titre, pas la constante de boot — sinon un joueur sur Halo 5 lit un catalogue
// Infinite. La résolution du player DB est déjà title-aware (registry.resolve lit
// ctxkeys.TitleSlug) ; ce helper aligne le FILTRAGE applicatif sur la même source.
package handlers

import (
	"context"

	"levelup/go-api/internal/ctxkeys"
)

// requestTitleSlug retourne le slug de titre de la requête s'il est présent dans
// le contexte (posé par TitleExtractor), sinon le fallback fourni (slug de boot du
// handler). HINF byte-identique : sans header titre, ctxkeys.TitleSlugIfSet est
// false → fallback = halo_infinite (l'ancienne constante).
func requestTitleSlug(ctx context.Context, fallback string) string {
	if slug, ok := ctxkeys.TitleSlugIfSet(ctx); ok {
		return slug
	}
	return fallback
}
