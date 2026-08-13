package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// projectiles.go — TRAJECTOIRES DE PROJECTILE projetées sur la grille du rejeu.
//
// SOURCE : filmdec.ScanFilmProjectiles — l'archétype ti=41, que le registre du film NOMME
// lui-même (`projectile-at-rest-state`, `projectile-tether-state`, `projectile-command_tick`).
// Position répliquée à ~60 Hz du départ à l'immobilisation.
//
// TÉMOIN : 65 des 70 lancers de grenade connus voient naître une trajectoire dans les 200 ms,
// contre 11 à 13 pour les mêmes lancers décalés en bloc.
//
// CE QUE CE CALQUE NE DIT PAS : l'IMPACT. Il n'existe aucun événement de détonation dans le
// film. Le dernier point est la DERNIÈRE POSITION RÉPLIQUÉE — pour une grenade à fragmentation
// la réplication cesse ~1,4 s après le lancer alors que la mèche court jusqu'à ~3 s. Le dernier
// point approche l'explosion parce que l'objet ne bouge plus, pas parce qu'on la lit. Le client
// doit écrire « dernière position connue », jamais « impact ».

// Projectile est une trajectoire de projectile, échantillonnée sur la grille du rejeu.
type Projectile struct {
	// T0 est l'index de frame du premier point, sur le même axe que Point.T.
	T0 int `json:"t0"`
	// P est la suite des points [dt, x, y] où dt est le décalage en frames depuis T0.
	// Format compact : une trajectoire porte des dizaines de points sur une seconde, et
	// répéter l'index absolu à chaque point doublerait le poids du document pour rien.
	P [][3]float32 `json:"p"`
	// Rest signale que le dernier point porte `projectile-at-rest-state` — le seul champ qui
	// CERTIFIE une fin de vol (78 fois sur 79 sur le film de référence). Sans lui, le vol
	// s'arrête parce que la réplication s'arrête, ce qui n'est pas la même chose.
	Rest bool `json:"rest,omitempty"`
}

// buildProjectiles projette les trajectoires décodées sur la grille de frames du rejeu.
//
// DÉCIMATION : le film réplique à ~60 Hz, la grille du rejeu est à 10 Hz. On garde UN point
// par frame — le premier — plutôt que de moyenner : un projectile suit une parabole, et
// moyenner deux positions distantes de 100 ms couperait le sommet de l'arc.
//
// LA SECONDE VALEUR DE RETOUR est la table index brut (rang dans `tracks`) -> index PUBLIÉ
// (rang dans la tranche rendue, après filtre et tri). C'est elle qui permet au lancer de
// grenade de publier son lien vers le projectile né de lui (Grenade.Proj) : l'appariement
// se fait sur les pistes brutes, l'artefact ne connaît que les publiées.
func buildProjectiles(tracks []filmdec.ProjectileTrack, origin, step uint64) ([]Projectile, map[int]int) {
	if len(tracks) == 0 {
		return nil, nil
	}
	type withRaw struct {
		p   Projectile
		raw int
	}
	kept := make([]withRaw, 0, len(tracks))
	for raw, tr := range tracks {
		if len(tr.Pts) < 3 || tr.Pts[0].TimestampUS < origin {
			continue
		}
		t0 := int((tr.Pts[0].TimestampUS - origin) / step)
		var pts [][3]float32
		last := -1
		for _, p := range tr.Pts {
			if p.TimestampUS < origin {
				continue
			}
			f := int((p.TimestampUS - origin) / step)
			if f == last {
				continue // un seul point par frame de la grille
			}
			last = f
			pts = append(pts, [3]float32{float32(f - t0), round2(p.X), round2(p.Y)})
		}
		if len(pts) < 2 { // une trajectoire d'un seul point de grille ne se dessine pas
			continue
		}
		kept = append(kept, withRaw{
			p:   Projectile{T0: t0, P: pts, Rest: tr.Pts[len(tr.Pts)-1].AtRest},
			raw: raw,
		})
	}
	// Tri TOTAL : T0 est un index de frame de la grille 10 Hz, donc les ex æquo sont la règle,
	// pas l'exception. Départager par la première position publiée puis par la longueur rend
	// l'ordre indépendant de celui des `tracks` reçues ; l'index brut ferme le dernier ex æquo
	// pour que le LIEN publié par un lancer ne dépende pas non plus du rang d'arrivée.
	sort.Slice(kept, func(i, j int) bool {
		a, b := kept[i].p, kept[j].p
		switch {
		case a.T0 != b.T0:
			return a.T0 < b.T0
		case a.P[0][1] != b.P[0][1]:
			return a.P[0][1] < b.P[0][1]
		case a.P[0][2] != b.P[0][2]:
			return a.P[0][2] < b.P[0][2]
		case len(a.P) != len(b.P):
			return len(a.P) < len(b.P)
		default:
			return kept[i].raw < kept[j].raw
		}
	})
	out := make([]Projectile, len(kept))
	pubByRaw := make(map[int]int, len(kept))
	for i, k := range kept {
		out[i] = k.p
		pubByRaw[k.raw] = i
	}
	return out, pubByRaw
}
