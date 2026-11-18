package halo_infinite

// killsource_class.go — LA CLASSE D'ARME D'UNE SOURCE DE DÉGÂT, celle de la Répartition des frags.
//
// DEUX SOURCES, ET AUCUNE DÉDUCTION.
//   - Une source dont le pont (`film/killicon`) nomme une arme prend la classe de cette arme au registre
//     canonique (`weapons.ClassesByKey`) : l'épée et le marteau sont `heavy`, les grenades de la liste du
//     jeu `grenade`, comme dans la Synthèse.
//   - Une source SANS clé d'arme prend la classe que le dictionnaire des sources de dégât (`damagetag`)
//     déclare pour son tag, et seulement pour les deux classes que la Synthèse compte par compteur API sans
//     arme : la mêlée générique (coup de crosse, dos) et la grenade que le registre ne porte pas (grenade à
//     pointes). Une source dont l'étiquette n'est pas publiable (statut ambigu) n'a pas de classe.
//
// Tout le reste (véhicule, objet, chute, écrasement, source inconnue) rend une classe vide : ni image ni nom
// n'entre jamais dans la décision.

import (
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/weapons"

	"github.com/JGtm/halo-infinite-film-decoder/damagetag"
)

// classesByWeaponKey est l'index statique clé d'arme -> classe, construit une fois : la décoration du fil
// l'interroge par élimination.
var classesByWeaponKey = sync.OnceValue(weapons.ClassesByKey)

// killSourceClass rend la classe de la source de dégât `tag` dont le pont a rendu la clé `weaponKey`
// (vide si le pont n'en nomme pas). Vide quand la classe n'est pas établie.
func killSourceClass(tag uint32, weaponKey string) string {
	if weaponKey != "" {
		return classesByWeaponKey()[weaponKey]
	}
	l, ok := damagetag.Lookup(tag)
	if !ok || !l.Publishable() {
		return ""
	}
	switch l.Class {
	case damagetag.ClassMelee:
		return domain.FragClassMelee
	case damagetag.ClassGrenade:
		return domain.FragClassGrenade
	}
	return ""
}
