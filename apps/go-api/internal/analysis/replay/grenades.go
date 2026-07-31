package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// grenades.go — LANCERS DE GRENADE.
//
// LE LANCER PORTE DÉJÀ SON AUTEUR. Comme l'événement de tir, il écrit le `FilmIndex` du
// lanceur : celui-ci n'est ni deviné ni voté. Ce fichier n'a donc jamais eu à trouver QUI a
// lancé.
//
// CE QUI ÉTAIT MAL CONÇU, ET QUI EST CORRIGÉ ICI. Pour dessiner le lancer, on prenait la
// position du BIPED du lanceur — ce qui exigeait de connaître son biped, donc de passer par le
// pont, donc d'échouer quand la vie n'était pas nommée. **Sept lancers sur soixante-dix étaient
// perdus pour cette seule raison.**
//
// Or le lancer fait naître un PROJECTILE, dont la position est décodée, et **dont le premier
// point est la main du lanceur** : la position cherchée est déjà là, sans le pont. Mesuré :
// la naissance est à 0,77 unité du biped auteur (médiane), contre 6,4 pour un instant permuté
// et 33,9 pour un biped tiré au hasard.
//
// LA HIÉRARCHIE DES SOURCES, dans cet ordre et pour cette raison :
//
//	1. la naissance du PROJECTILE     position de l'objet lancé, décodée — aucun pont
//	2. la position du BIPED du lanceur si le pont le connaît, et si aucun projectile n'apparie
//
// La seconde n'est pas un repli voté : c'est la même grandeur lue ailleurs. Le champ `Src` dit
// laquelle a servi, pour que l'écran puisse les distinguer s'il le veut.

// grenadeBirthWindowUS est la fenêtre dans laquelle un projectile doit naître après un lancer
// pour qu'on les tienne pour le même événement. Mesuré : 65 des 70 lancers apparient à
// ±200 ms, contre 11 à 13 pour les mêmes lancers décalés en bloc dans le temps.
const grenadeBirthWindowUS = 200_000

// Sources d'une position de lancer, publiées telles quelles dans l'artefact.
const (
	// GrenadeSrcProjectile : position du projectile à sa naissance — la main du lanceur.
	GrenadeSrcProjectile = "projectile"
	// GrenadeSrcBiped : position du biped du lanceur, quand aucun projectile n'apparie.
	GrenadeSrcBiped = "biped"
)

// Grenade est un lancer de grenade, situé dans le temps et l'espace.
type Grenade struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le biped lanceur quand il est connu (0 sinon). Il sert à relier le lancer à une
	// trajectoire ; il n'est PAS nécessaire pour situer le lancer.
	Slot uint32 `json:"slot"`
	// Idx est l'index de joueur ÉCRIT dans le film. C'est lui l'auteur, toujours renseigné.
	Idx int `json:"i"`
	// X, Y sont la position du lancer.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// Kind est le type de grenade ("Fragmentation", "Plasma", "Shock", "Spike").
	Kind string `json:"k"`
	// Src dit d'où vient la position : GrenadeSrcProjectile ou GrenadeSrcBiped.
	Src string `json:"s"`
}

// buildGrenades situe les lancers et rend la couverture.
func buildGrenades(pos []filmdec.BipedPosition, throws []filmdec.GrenadeThrow,
	origin, step uint64, owner map[uint32]int, proj []filmdec.ProjectileTrack) ([]Grenade, LayerCoverage) {
	cov := LayerCoverage{Available: len(throws)}
	if len(throws) == 0 {
		return nil, cov
	}
	births := projectileBirths(proj)
	tracks := indexBySlot(pos)
	var out []Grenade
	for _, g := range throws {
		gr, ok := locateThrow(g, births, tracks, owner)
		if !ok {
			cov.count(reasonNoSlot)
			continue
		}
		cov.count(reasonAttached)
		gr.T = int((g.TimestampUS - origin) / step)
		gr.Idx = g.FilmIndex
		gr.Kind = g.Name()
		out = append(out, gr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].T < out[j].T })
	return out, cov
}

// locateThrow situe un lancer : d'abord par la naissance de son projectile, sinon par le biped
// de son auteur quand le pont le connaît.
func locateThrow(g filmdec.GrenadeThrow, births []filmdec.ProjectileSample,
	tracks map[uint32]slotTrack, owner map[uint32]int) (Grenade, bool) {
	if b, ok := birthNear(births, g.TimestampUS); ok {
		return Grenade{X: round2(b.X), Y: round2(b.Y), Src: GrenadeSrcProjectile}, true
	}
	slot, reason := slotFor(tracks, owner, g.FilmIndex, g.TimestampUS)
	if reason != reasonAttached {
		return Grenade{}, false
	}
	p, d := tracks[slot].at(g.TimestampUS)
	if d > shotPosToleranceUS || !p.HasWorld {
		return Grenade{}, false
	}
	return Grenade{Slot: slot, X: round2(p.X), Y: round2(p.Y), Src: GrenadeSrcBiped}, true
}

// projectileBirths rend le premier point de chaque projectile, trié par instant.
func projectileBirths(proj []filmdec.ProjectileTrack) []filmdec.ProjectileSample {
	out := make([]filmdec.ProjectileSample, 0, len(proj))
	for _, p := range proj {
		if len(p.Pts) > 0 {
			out = append(out, p.Pts[0])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// birthNear rend la naissance de projectile la plus proche de `at`, si elle tombe dans la
// fenêtre.
func birthNear(births []filmdec.ProjectileSample, at uint64) (filmdec.ProjectileSample, bool) {
	if len(births) == 0 {
		return filmdec.ProjectileSample{}, false
	}
	i := sort.Search(len(births), func(k int) bool { return births[k].TimestampUS >= at })
	best, bestD := filmdec.ProjectileSample{}, uint64(1)<<62
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(births) {
			continue
		}
		if d := absDiffU64(births[k].TimestampUS, at); d < bestD {
			bestD, best = d, births[k]
		}
	}
	return best, bestD <= grenadeBirthWindowUS
}

// keepGrenadesOfPublishedTracks écarte les lancers rattachés à un biped SANS trajectoire
// publiée.
//
// UN LANCER SITUÉ PAR SON PROJECTILE N'EST PAS CONCERNÉ : il ne dépend d'aucune trajectoire,
// sa position est celle de l'objet lancé. Le filtrer reviendrait à jeter une donnée complète
// parce qu'une donnée voisine manque.
func keepGrenadesOfPublishedTracks(gren []Grenade, tracks []Track) []Grenade {
	if len(gren) == 0 {
		return nil
	}
	live := make(map[uint32]bool, len(tracks))
	for _, t := range tracks {
		live[t.Slot] = true
	}
	out := gren[:0]
	for _, g := range gren {
		if g.Src == GrenadeSrcProjectile || live[g.Slot] {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
