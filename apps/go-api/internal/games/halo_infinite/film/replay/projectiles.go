package replay

import (
	"cmp"
	"math"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// projectiles.go — TRAJECTOIRES DE PROJECTILE projetées sur la grille du rejeu, et l'EXPLOSION qui
// clôt chacune.
//
// SOURCE DE LA TRAJECTOIRE : grammar.ScanFilmProjectiles — l'archétype ti=41, que le registre du
// film NOMME lui-même (`projectile-at-rest-state`, `projectile-tether-state`,
// `projectile-command_tick`). Position répliquée à ~60 Hz du départ à l'immobilisation.
//
// TÉMOIN : 65 des 70 lancers de grenade connus voient naître une trajectoire dans les 200 ms,
// contre 11 à 13 pour les mêmes lancers décalés en bloc.
//
// L'EXPLOSION (lot detonations, 2026-10-22). Le film ÉCRIT la détonation — un message
// `projectile_detonate` (genre 5) ou `projectile_impact_effect` (genre 6) de la vue A
// (grammar/vue_a_detonations.go), avec sa position. CORRECTION D'UNE AFFIRMATION ANTÉRIEURE
// FAUSSE de cet en-tête (« ce calque ne lit pas la détonation ») : [attachDetonation] la lit et la
// rattache à la piste par proximité spatiale, parmi les détonations à portée temporelle de la
// dernière position publiée (mesure sur cinq films témoins, 727 lancers liés, 100 % appariés,
// médiane 17-35 ms après et 0,01 unité de la dernière position, jamais avant elle au-delà de la
// tolérance). Une piste sans détonation à portée (carte sans détonation lisible, vol tronqué) se
// publie quand même : [attachDetonation] pose alors la DERNIÈRE POSITION RÉPLIQUÉE, un repli NOMMÉ
// et COMPTÉ (`fallback.NomDetonationDernierePositionConnue`, ADR 0034 D-10) — jamais un « impact »
// tacite, et [Projectile.DetoLue] dit laquelle des deux voies a servi.

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
	// DetoT est l'index de frame de l'EXPLOSION, sur le même axe que Point.T (absolu, pas relatif
	// à T0) : soit l'instant du message de détonation/impact LU, soit, à défaut, celui du dernier
	// point de P (cf. [Projectile.DetoLue]).
	DetoT int `json:"detoT"`
	// DetoX/DetoY sont la position de l'EXPLOSION, mêmes axes que P.
	DetoX float32 `json:"detoX"`
	DetoY float32 `json:"detoY"`
	// DetoLue dit que DetoT/X/Y viennent d'un message de détonation/impact LU dans le film. Faux
	// = repli nommé (`fallback.NomDetonationDernierePositionConnue`) : DetoT/X/Y valent alors le
	// dernier point de P, l'approximation documentée (jamais une détonation affirmée).
	DetoLue bool `json:"detoLue,omitempty"`
	// Shot est l'index, dans ReplayDocument.Shots, du tir dont ce projectile est né. Absent = aucun
	// lien PUBLIÉ : projectile de grenade (son lien est Grenade.Proj), balle sans tir publié, ou
	// naissance ambiguë entre deux armes ou deux tireurs (cf. [linkShotsToProjectiles]). Pointeur :
	// 0 est un index valide, et omitempty l'effacerait.
	Shot *int `json:"shot,omitempty"`
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
// LA CAUSE EST EN AMONT, dans la déquantification (`grammar`), et elle N'EST PAS corrigée ici :
// elle est caractérisée (`.ai/archive/V7.5/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`). Ce qui est corrigé
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
func buildProjectiles(tracks []types.ProjectileTrack, origin, step uint64,
	detos []types.ProjectileDetonation, fb *fallback.Compteur) ([]Projectile, map[int]int, int) {
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
		var dernierPublie types.ProjectileSample
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
			last, dernierPublie = f, p
			pts = append(pts, [3]float32{float32(f - t0), round2(p.X), round2(p.Y)})
		}
		if coupe {
			tronquees++
		}
		if len(pts) < 2 { // une trajectoire d'un seul point de grille ne se dessine pas
			continue
		}
		proj := Projectile{T0: t0, P: pts,
			// `Rest` CERTIFIE une fin de vol : un vol coupé n'a pas la sienne, et le dire
			// serait affirmer qu'on a vu le projectile s'immobiliser là.
			Rest: !coupe && tr.Pts[len(tr.Pts)-1].AtRest}
		attachDetonation(&proj, dernierPublie, origin, step, detos, fb)
		kept = append(kept, withRaw{p: proj, raw: raw})
	}
	// Tri TOTAL : T0 est un index de frame de la grille 10 Hz, donc les ex æquo sont la règle,
	// pas l'exception. Départager par la première position publiée puis par la longueur rend
	// l'ordre indépendant de celui des `tracks` reçues ; l'index brut ferme le dernier ex æquo
	// pour que le LIEN publié par un lancer ne dépende pas non plus du rang d'arrivée.
	slices.SortFunc(kept, func(x, y withRaw) int {
		a, b := x.p, y.p
		return cmp.Or(
			cmp.Compare(a.T0, b.T0),
			cmp.Compare(a.P[0][1], b.P[0][1]),
			cmp.Compare(a.P[0][2], b.P[0][2]),
			cmp.Compare(len(a.P), len(b.P)),
			cmp.Compare(x.raw, y.raw),
		)
	})
	out := make([]Projectile, len(kept))
	pubByRaw := make(map[int]int, len(kept))
	for i, k := range kept {
		out[i] = k.p
		pubByRaw[k.raw] = i
	}
	return out, pubByRaw, tronquees
}

// detoSearchBeforeUS/detoSearchAfterUS bornent la fenêtre de recherche d'une détonation à portée
// de la dernière position publiée d'une piste.
//
// MESURE QUI LES FONDE (lot detonations, 2026-10-22, cinq films témoins — Origin, Dredge,
// Snowbound, Flood Gulch, Catalyst — 727 lancers liés à une piste) : 727/727 (100 %) trouvent une
// détonation à position dans cette fenêtre, médiane 17 à 35 ms APRÈS la dernière position publiée
// (jamais avant au-delà de la tolérance — un seul cas à -69 ms, dans la marge), médiane de
// distance 0,01 unité (min 0,01, max 3,58 sur 4 989 lancers liés). La borne haute (4 s) couvre la
// queue des mèches longues (frag/plasma) sans ouvrir sur l'explosion d'un AUTRE projectile du
// même paquet.
const (
	detoSearchBeforeUS uint64 = 500_000
	detoSearchAfterUS  uint64 = 4_000_000
)

// attachDetonation pose l'EXPLOSION d'une piste sur `p` : la détonation/impact LUE la plus proche
// dans l'espace parmi celles à portée temporelle de `dernierPublie` (sa dernière position publiée)
// — à défaut, le repli NOMMÉ et COMPTÉ [fallback.NomDetonationDernierePositionConnue] : cette
// dernière position elle-même, l'approximation que ce calque publiait SANS repli avant ce lot.
//
// `dernierPublie` est la valeur zéro (`TimestampUS` 0) pour une piste dont `pts` n'a reçu aucun
// point avant le premier pas impossible — impossible ici puisque l'appelant exige `len(pts) >= 2`
// avant d'appeler cette fonction, donc au moins un point publié.
func attachDetonation(p *Projectile, dernierPublie types.ProjectileSample, origin, step uint64,
	detos []types.ProjectileDetonation, fb *fallback.Compteur) {
	if best, ok := detonationLaPlusProche(dernierPublie, detos); ok {
		p.DetoT = int((best.TimestampUS - origin) / step) //nolint:gosec // index de frame d'un match
		p.DetoX, p.DetoY, p.DetoLue = round2(best.X), round2(best.Y), true
		return
	}
	fb.Declenche(fallback.NomDetonationDernierePositionConnue)
	dernier := p.P[len(p.P)-1]
	p.DetoT, p.DetoX, p.DetoY = p.T0+int(dernier[0]), dernier[1], dernier[2]
}

// detonationLaPlusProche rend, parmi les détonations À POSITION dont l'instant tombe dans
// [last - detoSearchBeforeUS, last + detoSearchAfterUS], celle la plus proche dans l'espace de
// `last`. PUR : aucune I/O, aucun état partagé — un film décodé deux fois en parallèle (ADR 0034
// D-5) rend le même résultat.
func detonationLaPlusProche(last types.ProjectileSample, detos []types.ProjectileDetonation) (types.ProjectileDetonation, bool) {
	var best types.ProjectileDetonation
	bestDist, trouve := math.MaxFloat64, false
	for _, d := range detos {
		if !d.HasPosition {
			continue
		}
		if d.TimestampUS+detoSearchBeforeUS < last.TimestampUS || d.TimestampUS > last.TimestampUS+detoSearchAfterUS {
			continue
		}
		dist := dist3([3]float32{d.X, d.Y, d.Z}, [3]float32{last.X, last.Y, last.Z})
		if !trouve || dist < bestDist {
			best, bestDist, trouve = d, dist, true
		}
	}
	return best, trouve
}
