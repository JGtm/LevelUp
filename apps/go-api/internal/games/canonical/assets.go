// Package canonical — assets.go : type référentiel pour l'Asset Drawer.
package canonical

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
	SpriteSheet   string `json:"sprite_sheet,omitempty"`   // URL de la feuille de sprites (médailles h5)
	SpriteLeft    int    `json:"sprite_left,omitempty"`    // offset X dans la feuille
	SpriteTop     int    `json:"sprite_top,omitempty"`     // offset Y
	SpriteWidth   int    `json:"sprite_width,omitempty"`   // largeur de la découpe
	SpriteHeight  int    `json:"sprite_height,omitempty"`  // hauteur de la découpe
}
