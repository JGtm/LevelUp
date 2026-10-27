// Package canonical — assets.go : type référentiel pour l'Asset Drawer.
package canonical

import (
	"strconv"

	"levelup/go-api/internal/i18n"
)

// AssetMeta décrit un asset visuel (map, arme, médaille) exposé par l'Asset Drawer.
// C'est un type de réponse pur — aucune logique métier.
//
// Deux états, jamais mélangés : lu depuis la base, il porte les libellés dans TOUTES les langues
// (`Names`, `Descriptions`, `json:"-"`) ; servi, il porte le libellé de la langue de la requête
// (`Name`, `Description`), posé par Localized (ADR 0040 D-3, côté service, avec la `Locale` du
// contexte). `Name` et `Description` d'une valeur non localisée sont vides.
//
// Médailles Halo 5 : l'icône est un SPRITE (feuille + offset, API Metadata
// officielle) plutôt qu'une image dédiée. Les champs Sprite* portent la découpe ;
// le front rend via background-image + background-position. Tous omitempty :
// maps/armes (ImageURL direct) n'ont pas de sprite.
type AssetMeta struct {
	ID           string    `json:"id"`
	Names        i18n.Text `json:"-"`                     // nom par langue (donnée) ; une langue sans traduction est absente
	Descriptions i18n.Text `json:"-"`                     // description par langue (médailles) ; vide pour maps/armes
	Name         string    `json:"name"`                  // nom dans la langue de la requête (repli `en`)
	Description  string    `json:"description,omitempty"` // description dans la langue de la requête ; vide pour maps/armes
	ImageURL     string    `json:"image_url"`             // URL relative /api/v1/assets/... ou "" si pas d'image
	// ImageTinted : l'image est un MASQUE (dessin porté par l'alpha) que le front doit
	// teindre, et non une image finie. Cf. games.TitleAssetURLAdapter.WeaponImageIsTinted.
	ImageTinted  bool   `json:"image_tinted,omitempty"`
	SpriteSheet  string `json:"sprite_sheet,omitempty"`  // URL de la feuille de sprites (médailles h5)
	SpriteLeft   int    `json:"sprite_left,omitempty"`   // offset X dans la feuille
	SpriteTop    int    `json:"sprite_top,omitempty"`    // offset Y
	SpriteWidth  int    `json:"sprite_width,omitempty"`  // largeur de la découpe
	SpriteHeight int    `json:"sprite_height,omitempty"` // hauteur de la découpe
}

// KillSourceIcon décrit l'icône de la SOURCE DE DÉGÂT d'une mort — ce que le kill feed
// montre à côté du nom du tueur.
//
// Distincte de l'icône d'arme du tiroir d'assets, et pour deux raisons mesurées :
//   - la clé n'est pas la même. Une mort ne porte pas de `weapon_id` : elle porte un
//     identifiant d'EFFET de dégât, propre au format de film du titre. Le titre est seul
//     à savoir le traduire en image.
//   - le format n'est pas le même. Le feed lit en petit : les titres qui ont un atlas
//     dédié à cet usage doivent pouvoir le servir ici sans changer l'autre surface.
//
// Un titre sans réponse rend le zéro de ce type : le produit affiche alors le libellé
// seul. Aucune icône vaut toujours mieux que l'icône d'une autre arme.
type KillSourceIcon struct {
	// WeaponKey : clé canonique du registre d'armes du titre. Vide quand l'objet n'y
	// figure pas (variantes, objets hors registre) — l'icône reste valable.
	WeaponKey string `json:"weapon_key,omitempty"`
	// Label : le nom PROPRE de la source (BR75, Needler). Pas un libellé traduit : il
	// vient de la table de nommage du titre, seule à savoir ce que l'identifiant désigne.
	// Vide si le titre nomme l'objet sans certitude — l'icône peut alors rester servie.
	Label string `json:"label,omitempty"`
	// ImageURL : URL relative de l'image, "" si le titre n'en a pas pour cette source.
	ImageURL string `json:"image_url,omitempty"`
	// Tinted : l'image est un MASQUE à teindre, pas un visuel fini.
	// Cf. games.TitleAssetURLAdapter.WeaponImageIsTinted pour le pourquoi.
	Tinted bool `json:"image_tinted,omitempty"`
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

// Localized rend une copie dont `Name` et `Description` sont ceux de la langue `loc`, avec le repli
// des noms de jeu (`en`, ADR 0040 D-4). Les textes par langue (`Names`, `Descriptions`) restent
// portés : la copie peut être localisée une seconde fois dans une autre langue.
func (m AssetMeta) Localized(loc i18n.Locale) AssetMeta {
	m.Name = m.Names.Pick(loc)
	m.Description = m.Descriptions.Pick(loc)
	return m
}

// CanonicalName est le nom du jeu dans la langue de repli (`en`, ADR 0040 D-4) : la clé qui identifie
// une carte ou une arme auprès des adaptateurs d'images et de l'ordre du tiroir, jamais un texte à
// afficher (l'affichage passe par Name, résolu dans la langue de la requête).
func (m AssetMeta) CanonicalName() string {
	return m.Names[i18n.Fallback]
}
