package halo_infinite

// killsource_icons.go — le RÉFÉRENTIEL « arme -> pictogramme du fil d'éliminations », énuméré.
//
// MÊME RÉSOLUTION QUE `KillSourceIcon`, PARCOURUE : chaque entrée est ce que `KillSourceIcon` rend
// pour une source de dégât de l'arme, jamais une table à part. Ce qui n'a pas de clé de registre
// (grenade, mêlée) ou pas d'image n'entre pas : le référentiel sert à retrouver un pictogramme PAR
// CLÉ, et une entrée sans clé n'est retrouvable par personne.

import (
	"sort"

	"levelup/go-api/internal/games/canonical"

	"github.com/JGtm/halo-infinite-film-decoder/killicon"
)

// KillSourceIconsByKey rend une entrée par weapon_key, triée par clé (port.KillSourceIconCatalog).
//
// Plusieurs sources de dégât peuvent porter la même clé (variantes d'une arme) ; elles servent toutes
// le même pictogramme, la première source rencontrée fixe l'entrée. L'invariant « une clé, une
// image » est tenu par `killsource_icons_test.go`.
func (a *AssetURLAdapter) KillSourceIconsByKey() []canonical.KillSourceIcon {
	byKey := map[string]canonical.KillSourceIcon{}
	for _, tag := range killicon.ResolvedTags() {
		ic, ok := a.KillSourceIcon(tag)
		if !ok || ic.WeaponKey == "" || ic.ImageURL == "" {
			continue
		}
		if _, seen := byKey[ic.WeaponKey]; !seen {
			byKey[ic.WeaponKey] = ic
		}
	}
	out := make([]canonical.KillSourceIcon, 0, len(byKey))
	for _, ic := range byKey {
		out = append(out, ic)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WeaponKey < out[j].WeaponKey })
	return out
}
