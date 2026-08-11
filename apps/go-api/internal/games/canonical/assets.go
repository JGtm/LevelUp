// Package canonical — assets.go : type référentiel pour l'Asset Drawer.
package canonical

import "strconv"

// AssetMeta décrit un asset visuel (map, arme, médaille) exposé par l'Asset Drawer.
// C'est un type de réponse pur — aucune logique métier.
//
// Médailles Halo 5 : l'icône est un SPRITE (feuille + offset, API Metadata
// officielle) plutôt qu'une image dédiée. Les champs Sprite* portent la découpe ;
// le front rend via background-image + background-position. Tous omitempty :
// maps/armes (ImageURL direct) n'ont pas de sprite.
type AssetMeta struct {
	ID            string `json:"id"`
	NameEN        string `json:"name_en"`
	NameFR        string `json:"name_fr"`                  // vide si traduction absente
	Description   string `json:"description,omitempty"`    // description EN (médailles) ; vide pour maps/armes
	DescriptionFR string `json:"description_fr,omitempty"` // description FR (médailles) ; vide si absente
	ImageURL      string `json:"image_url"`                // URL relative /api/v1/assets/... ou "" si pas d'image
	// ImageTinted : l'image est un MASQUE (dessin porté par l'alpha) que le front doit
	// teindre, et non une image finie. Cf. games.TitleAssetURLAdapter.WeaponImageIsTinted.
	ImageTinted  bool   `json:"image_tinted,omitempty"`
	SpriteSheet  string `json:"sprite_sheet,omitempty"`  // URL de la feuille de sprites (médailles h5)
	SpriteLeft   int    `json:"sprite_left,omitempty"`   // offset X dans la feuille
	SpriteTop    int    `json:"sprite_top,omitempty"`    // offset Y
	SpriteWidth  int    `json:"sprite_width,omitempty"`  // largeur de la découpe
	SpriteHeight int    `json:"sprite_height,omitempty"` // hauteur de la découpe
}

// WeaponID décode l'ID d'un asset ARME en identifiant numérique. Second retour
// faux si l'ID n'est pas un entier décimal (asset non-arme, ID vide).
//
// ParseUint puis conversion, PAS ParseInt : un weapon_id Halo Infinite est un
// UINT64 (jusqu'à 18 273 449 274 082 158 495) qui dépasse la borne d'int64.
// ParseInt échouerait sur les deux tiers du référentiel — dont VK78 Commando.
// int64 porte le MOTIF BINAIRE, comme partout ailleurs dans le domaine
// (cf. platform/duckdb/weapon_resolver.go, qui refait FormatUint pour requêter).
func (m AssetMeta) WeaponID() (int64, bool) {
	v, err := strconv.ParseUint(m.ID, 10, 64)
	if err != nil {
		return 0, false
	}
	return int64(v), true //nolint:gosec // motif binaire conservé, convention du domaine
}
