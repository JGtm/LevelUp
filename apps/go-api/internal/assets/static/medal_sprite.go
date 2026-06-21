package static

import "strconv"

// MedalSprite décrit l'icône d'une médaille servie comme sprite (feuille + offset).
//
// Modèle Halo 5 : l'icône d'une médaille n'est pas un PNG dédié mais une découpe
// d'une feuille de sprites (table medal_definitions : sprite_sheet_url + offsets).
// Le front rend via background-image + background-position (cf. AssetCard).
type MedalSprite struct {
	SheetURL string
	Left     int
	Top      int
	Width    int
	Height   int
}

// medalSpriteResolver : hook title-aware injecté au boot. Retourne le sprite d'une
// médaille pour un titre (Halo 5), ou nil si le titre sert des PNG (Halo Infinite).
//
// Miroir EXACT du csrBadgeResolver (platform/duckdb) : posé UNE fois au boot,
// lu en read-only ensuite — aucun accès concurrent au-delà du set initial.
var medalSpriteResolver func(titleSlug string, medalID int64) *MedalSprite

// SetMedalSpriteResolver câble le résolveur (boot). nil = comportement PNG only (HINF).
func SetMedalSpriteResolver(f func(titleSlug string, medalID int64) *MedalSprite) {
	medalSpriteResolver = f
}

// MedalImage résout l'icône d'une médaille de façon title-aware : sprite si le titre
// en fournit un (h5), sinon URL PNG /static/medals/{slug}/{id}.png (HINF, inchangé).
//
// pngURL est vide quand un sprite est retourné. Sans résolveur câblé (ou titre non
// couvert), le comportement est strictement identique à static.URL(KindMedal, ...) —
// HINF byte-identique.
func MedalImage(titleSlug string, medalID int64) (pngURL string, sprite *MedalSprite) {
	if medalSpriteResolver != nil {
		if sp := medalSpriteResolver(titleSlug, medalID); sp != nil && sp.SheetURL != "" {
			return "", sp
		}
	}
	return URL(KindMedal, titleSlug, strconv.FormatInt(medalID, 10), ".png"), nil
}
