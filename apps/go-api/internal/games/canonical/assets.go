// Package canonical — assets.go : type référentiel pour l'Asset Drawer.
package canonical

// AssetMeta décrit un asset visuel (map, arme) exposé par l'Asset Drawer.
// C'est un type de réponse pur — aucune logique métier.
type AssetMeta struct {
	ID       string `json:"id"`
	NameEN   string `json:"name_en"`
	NameFR   string `json:"name_fr"`   // vide si traduction absente
	ImageURL string `json:"image_url"` // URL relative /api/v1/assets/... ou "" si pas d'image
}
