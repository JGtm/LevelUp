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

// projectileMaxStepM borne le déplacement d'un projectile entre deux points de la grille
// (100 ms). Au-delà, ce n'est plus une lecture : c'est un artefact de déquantification.
//
// LE DÉFAUT MESURÉ, ET POURQUOI CE GARDE-FOU EXISTE. Sur les 76 artefacts du parc (schéma 51,
// 2026-09-11), 947 trajectoires sur 15 735 — 6,0 % — portent au moins un pas impossible, soit
// 4 901 pas. La signature est celle d'UN BIT du champ quantifié qui bascule : le saut vaut
// l'étendue de la carte sur un axe DIVISÉE PAR UNE PUISSANCE DE DEUX, l'autre axe ne bougeant
// pas d'un centimètre. Sur les quatre films Live Fire du parc (`sgh_interlock`, Y sur 12 bits),
// le saut vaut EXACTEMENT la moitié de l'étendue Y — 31,89 m pour 63,775 m, soit le bit de poids
// fort — et cette forme couvre 3 907 des 4 901 pas ; sur les cartes Forge, l'axe touché est
// plutôt X et le bit plus bas (étendue / 2^7 majoritaire).
//
// LA CAUSE EST EN AMONT, dans la déquantification (`filmdec`), et elle N'EST PAS corrigée ici :
// elle est caractérisée (`.ai/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`). Ce qui est corrigé
// ici est la PUBLICATION d'une position fausse, qui faisait tracer au client une droite en
// travers de toute la carte, à 300 m/s et plus.
//
// 10 m par pas, soit 100 m/s, laisse passer tout projectile du jeu (une grenade tient sous
// 20 m/s, une roquette sous 30) et ne coupe que l'impossible. C'est aussi le seuil que le
// filtre de vitesse des bipèdes emploie déjà, pour la même raison.
const projectileMaxStepM = 10

// buildProjectiles projette les trajectoires décodées sur la grille de frames du rejeu.
//
// DÉCIMATION : le film réplique à ~60 Hz, la grille du rejeu est à 10 Hz. On garde UN point
// par frame — le premier — plutôt que de moyenner : un projectile suit une parabole, et
// moyenner deux positions distantes de 100 ms couperait le sommet de l'arc.
//
// LE VOL S'ARRÊTE AU PREMIER PAS IMPOSSIBLE, il n'est pas recousu : après un basculement de
// bit, la suite du vol est ailleurs sur la carte et rien ne dit où le projectile est réellement
// passé. C'est la même règle que celle qui gouverne la fin d'un vol — on publie ce qui est lu,
// et on s'arrête là où le film cesse d'être lisible.
//
// LA SECONDE VALEUR DE RETOUR est la table index brut (rang dans `tracks`) -> index PUBLIÉ
// (rang dans la tranche rendue, après filtre et tri). C'est elle qui permet au lancer de
// grenade de publier son lien vers le projectile né de lui (Grenade.Proj) : l'appariement
// se fait sur les pistes brutes, l'artefact ne connaît que les publiées.
//
// LA TROISIÈME est le nombre de trajectoires COUPÉES, qui remonte à la couverture du document :
// un décodeur qui coupe sans le dire est un rejet avalé (cf. coverage.go). Elle compte aussi les
// coupures dont la trajectoire n'est pas publiée ensuite — sans quoi le compteur mentirait par
// omission.
func buildProjectiles(tracks []filmdec.ProjectileTrack, origin, step uint64) ([]Projectile, map[int]int, int) {
	if len(tracks) == 0 {
		return nil, nil, 0
	}
	tronquees := 0
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
		coupe := false
		for _, p := range tr.Pts {
			if p.TimestampUS < origin {
				continue
			}
			f := int((p.TimestampUS - origin) / step)
			if f == last {
				continue // un seul point par frame de la grille
			}
			if n := len(pts); n > 0 && planDist(round2(p.X), round2(p.Y), pts[n-1][1], pts[n-1][2]) > projectileMaxStepM {
				coupe = true
				break
			}
			last = f
			pts = append(pts, [3]float32{float32(f - t0), round2(p.X), round2(p.Y)})
		}
		if coupe {
			tronquees++
		}
		if len(pts) < 2 { // une trajectoire d'un seul point de grille ne se dessine pas
			continue
		}
		kept = append(kept, withRaw{
			// `Rest` CERTIFIE une fin de vol : un vol coupé n'a pas la sienne, et le dire
			// serait affirmer qu'on a vu le projectile s'immobiliser là.
			p:   Projectile{T0: t0, P: pts, Rest: !coupe && tr.Pts[len(tr.Pts)-1].AtRest},
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
	return out, pubByRaw, tronquees
}
