package replay

import (
	"math"
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
	// Slot est le biped lanceur quand le pont le connaît (0 sinon). Il sert à relier le lancer
	// à une trajectoire ; il n'est PAS nécessaire pour situer le lancer.
	//
	// IL EST PUBLIÉ SUR LES DEUX BRANCHES DEPUIS LE 2026-09-11, et il ne l'était que sur la
	// branche biped : un lancer situé par son projectile sortait à zéro, et zéro RESSEMBLE à un
	// slot. Zéro reste donc « pont muet », mais il ne veut plus dire « position lue ailleurs ».
	Slot uint32 `json:"slot"`
	// Idx est l'index de joueur ÉCRIT dans le film. C'est lui l'auteur, toujours renseigné.
	Idx int `json:"i"`
	// X, Y sont la position du lancer.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// Rank est le RANG du type de grenade : un index dans ReplayDocument.GrenadeLabels,
	// la seule table qui les nomme.
	//
	// C'ÉTAIT UN NOM JUSQU'AU 2026-08-02, et c'est ce qui a produit la contradiction du
	// lot 3.1 : le lancer disait « Shock » là où le compteur porté du MÊME type disait
	// « Dynamo », sur la même fiche. Un index ne peut pas diverger de sa table.
	// Pas d'omitempty : le rang 0 (fragmentation) est une valeur, pas une absence.
	Rank int `json:"rank"`
	// Src dit d'où vient la position : GrenadeSrcProjectile ou GrenadeSrcBiped.
	Src string `json:"s"`
	// Proj est l'index, dans ReplayDocument.Projectiles, du projectile né de ce lancer
	// (appariement à ±200 ms, cf. grenadeBirthWindowUS). C'est le lien qui permet au
	// client de poser l'effet du type de grenade au point de REPOS du vol — la dernière
	// position répliquée, jamais un « impact » (aucun événement de détonation n'existe).
	//
	// POINTEUR, PAS int : le piège omitempty du dépôt — l'index 0 est un projectile
	// valide, un int l'omettrait exactement comme une absence de lien. Nil quand aucun
	// projectile n'apparie, ou quand celui qui appariait n'est pas publié (trop court).
	Proj *int `json:"proj,omitempty"`
}

// buildGrenades situe les lancers et rend la couverture.
//
// `pubProjByRaw` traduit l'index BRUT d'une piste de projectile (rang dans `proj`) vers son
// index PUBLIÉ (cf. buildProjectiles) : c'est lui qui alimente Grenade.Proj. Nil = aucun
// projectile publié, les lancers sortent sans lien — jamais un index qui ne pointe rien.
func buildGrenades(pos []filmdec.BipedPosition, throws []filmdec.GrenadeThrow,
	origin, step uint64, owner map[uint32]int, proj []filmdec.ProjectileTrack,
	pubProjByRaw map[int]int) ([]Grenade, LayerCoverage) {
	cov := LayerCoverage{Available: len(throws)}
	if len(throws) == 0 {
		return nil, cov
	}
	births := projectileBirths(proj)
	tracks := indexBySlot(pos)
	var out []Grenade
	for _, g := range throws {
		rank, known := g.Rank()
		if !known {
			// Le décodeur ne rend que des tags de sa liste blanche ; un tag hors rangs
			// serait un lancer qu'aucune table ne peut nommer. On ne le publie pas
			// plutôt que de le poser sur le rang 0 (fragmentation).
			cov.count(reasonNoSlot)
			continue
		}
		gr, rawProj, ok := locateThrow(g, births, tracks, owner)
		if !ok {
			cov.count(reasonNoSlot)
			continue
		}
		if rawProj >= 0 {
			if pub, published := pubProjByRaw[rawProj]; published {
				p := pub
				gr.Proj = &p
			}
		}
		cov.count(reasonAttached)
		gr.T = int((g.TimestampUS - origin) / step)
		gr.Idx = g.FilmIndex
		gr.Rank = rank
		out = append(out, gr)
	}
	// Tri TOTAL : deux lancers tombent souvent sur la même frame de la grille (10 Hz), et un
	// départage arbitraire suffit à changer l'artefact d'un octet à l'autre.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.T != b.T:
			return a.T < b.T
		case a.Idx != b.Idx:
			return a.Idx < b.Idx
		case a.Slot != b.Slot:
			return a.Slot < b.Slot
		case a.X != b.X:
			return a.X < b.X
		default:
			return a.Y < b.Y
		}
	})
	return out, cov
}

// locateThrow situe un lancer : d'abord par la naissance de son projectile, sinon par le biped
// de son auteur quand le pont le connaît. La seconde valeur est l'index BRUT de la piste de
// projectile appariée (-1 quand la position vient du biped) : c'est lui qui fonde le lien
// Grenade.Proj, une fois traduit en index publié par l'appelant.
//
// L'AUTEUR EST RÉSOLU EN PREMIER, ET C'EST UN CORRECTIF (2026-09-11). La naissance était choisie
// sur le TEMPS SEUL : `birthNear` prenait la plus proche dans une fenêtre de 200 ms, et le
// départage des naissances simultanées venait du tri, donc de X. Quand deux joueurs lancent dans
// la même fenêtre — banal —, le lancer recevait la position du projectile de l'AUTRE.
//
// MESURÉ SUR NOTRE PARC (banc `grenade_ecart_research_test.go`, distance du lancer publié au
// biped de son auteur au même instant) : sur `000d5950`, médiane 0,44 m mais 2 lancers sur 64
// au-delà de 4 m, pire cas 14,46 m ; sur les deux films Live Fire `0797ce72` et `21ece4d8`,
// médiane 25,42 m et 26,69 m et **100 %** des lancers au-delà de 4 m. Un lancer sur cinq tombe
// dans une fenêtre portant deux naissances ou plus (21,4 % / 19,4 % / 13,5 %) — la population
// exacte où le départage par le temps ne décide rien. À comparer à la mesure qui FONDE cette
// source : 0,77 unité entre une naissance et le biped de son auteur.
//
// LE BIPED DE L'AUTEUR EST DONC LE JUGE, quand le pont le donne : parmi les naissances de la
// fenêtre, on retient celle qui est à portée de sa main, et aucune si elle n'y est pas. Sans
// pont, la source reste utilisable — c'était sa raison d'être — mais une fenêtre qui porte
// PLUSIEURS naissances n'est plus tranchée au hasard : elle n'est pas publiée.
func locateThrow(g filmdec.GrenadeThrow, births []projectileBirth,
	tracks map[uint32]slotTrack, owner map[uint32]int) (Grenade, int, bool) {
	slot, author := authorBiped(g, tracks, owner)
	if b, ok := birthForThrow(births, g.TimestampUS, author); ok {
		// LE SLOT EST PORTÉ MÊME ICI, ET IL NE L'ÉTAIT PAS : la branche projectile rendait un
		// `Grenade` sans `Slot`, donc à zéro — et zéro RESSEMBLE à un slot, si bien qu'un
		// lecteur qui colore un lancer par son lanceur ne pouvait ni nommer personne, ni voir
		// l'ambiguïté. La POSITION ne dépend toujours pas du pont (c'est tout l'intérêt de
		// cette source) ; le slot, lui, est publié dès que le pont le connaît.
		return Grenade{Slot: slot, X: round2(b.s.X), Y: round2(b.s.Y), Src: GrenadeSrcProjectile}, b.raw, true
	}
	if author == nil {
		return Grenade{}, -1, false
	}
	return Grenade{Slot: slot, X: round2(author.X), Y: round2(author.Y), Src: GrenadeSrcBiped}, -1, true
}

// authorBiped rend le slot du lanceur et sa position répliquée, quand le pont et le film les
// donnent tous les deux. Le slot peut être connu sans que la position le soit (réplication trop
// lointaine) : le premier retour vaut alors le slot, le second nil.
func authorBiped(g filmdec.GrenadeThrow, tracks map[uint32]slotTrack,
	owner map[uint32]int) (uint32, *filmdec.BipedPosition) {
	slot, reason := slotFor(tracks, owner, g.FilmIndex, g.TimestampUS)
	if reason != reasonAttached {
		return 0, nil
	}
	p, d := tracks[slot].at(g.TimestampUS)
	if d > shotPosToleranceUS || !p.HasWorld {
		return slot, nil
	}
	return slot, &p
}

// grenadeAuthorRadiusM est la distance maximale acceptée entre la naissance d'un projectile et
// le biped de son lanceur, en mètres.
//
// LA MESURE QUI LE FONDE : 0,77 unité de médiane entre une naissance et le biped de son auteur,
// contre 6,4 pour un instant permuté. Quatre mètres laissent largement passer le signal (le banc
// mesure une médiane de 0,44 m sur `000d5950` après correctif du choix) tout en écartant le
// projectile d'un joueur voisin — et, incidemment, les naissances victimes du repli de quantum
// mesuré sur Live Fire, qui sautent d'une demi-étendue de carte (31,89 m).
//
// LE CLIENT NE PORTE PAS ENCORE DE JUMEAU : vérifié le 2026-09-11, `apps/web/src/features/
// match-replay/` n'a ni `grenadeArcs.ts` ni `ARC_ORIGIN_RADIUS_M` — les arcs de lancer n'y sont
// pas dessinés. Si ce seuil y naît un jour, il doit valoir CE nombre et le dire, faute de quoi
// les deux répondront différemment à la même question (« cette naissance est-elle celle de CE
// lanceur ? »).
const grenadeAuthorRadiusM = 4

// birthForThrow choisit la naissance qui appartient à CE lancer.
//
// AVEC L'AUTEUR : la plus proche de sa main, refusée au-delà de `grenadeAuthorRadiusM`. SANS
// l'auteur : une seule candidate est une lecture, plusieurs sont un tirage au sort — on
// s'abstient plutôt que de poser un lancer sur le projectile du voisin.
func birthForThrow(births []projectileBirth, at uint64,
	author *filmdec.BipedPosition) (projectileBirth, bool) {
	cands := birthsInWindow(births, at)
	if len(cands) == 0 {
		return projectileBirth{}, false
	}
	if author == nil {
		if len(cands) == 1 {
			return cands[0], true
		}
		return projectileBirth{}, false
	}
	best, bestD := projectileBirth{raw: -1}, math.MaxFloat64
	for _, c := range cands {
		if d := planDist(c.s.X, c.s.Y, author.X, author.Y); d < bestD {
			bestD, best = d, c
		}
	}
	if bestD > grenadeAuthorRadiusM {
		return projectileBirth{}, false
	}
	return best, true
}

// birthsInWindow rend TOUTES les naissances de la fenêtre, et pas seulement la plus proche dans
// le temps : c'est le fait qu'il y en ait plusieurs qui PORTE l'ambiguïté. Les rendre toutes est
// la condition pour que le biped de l'auteur puisse trancher.
//
// `births` est trié par instant (cf. projectileBirths), et ce tri reste TOTAL : il ne choisit
// plus la position publiée, mais il rend cette tranche reproductible d'une construction à
// l'autre.
func birthsInWindow(births []projectileBirth, at uint64) []projectileBirth {
	var lo uint64
	if at > grenadeBirthWindowUS {
		lo = at - grenadeBirthWindowUS
	}
	hi := at + grenadeBirthWindowUS
	i := sort.Search(len(births), func(k int) bool { return births[k].s.TimestampUS >= lo })
	var out []projectileBirth
	for ; i < len(births) && births[i].s.TimestampUS <= hi; i++ {
		out = append(out, births[i])
	}
	return out
}

// projectileBirth est la naissance d'une piste de projectile, avec l'index BRUT de sa piste
// (rang dans la tranche décodée) : c'est cette clé que buildProjectiles sait traduire en
// index publié.
type projectileBirth struct {
	s   filmdec.ProjectileSample
	raw int
}

// projectileBirths rend le premier point de chaque projectile, trié par instant.
//
// LE TRI EST TOTAL, ET C'EST LA CONDITION DE REPRODUCTIBILITÉ DE L'ARTEFACT. Plusieurs
// projectiles naissent au MÊME instant de réplication, et l'ordre d'arrivée vient d'une
// itération de map : sans départage stable, deux constructions du même film ne rendent pas la
// même tranche (mesuré : 12,72 / −187,11 contre 11,41 / 17,99 sur le lancer t=1580 de
// `01e1f945`). Départager par la position rend l'ordre indépendant de l'amont ; l'index brut
// ferme le dernier ex æquo depuis que la naissance porte aussi le LIEN vers sa piste.
//
// CE TRI NE CHOISIT PLUS LA POSITION PUBLIÉE, et c'est le correctif de `locateThrow` : le choix
// parmi les naissances d'une même fenêtre revient au biped de l'auteur, pas au rang dans la
// tranche. L'ordre reste requis — il rend `birthsInWindow` reproductible — mais il n'arbitre
// plus rien.
func projectileBirths(proj []filmdec.ProjectileTrack) []projectileBirth {
	out := make([]projectileBirth, 0, len(proj))
	for raw, p := range proj {
		if len(p.Pts) > 0 {
			out = append(out, projectileBirth{s: p.Pts[0], raw: raw})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].s, out[j].s
		switch {
		case a.TimestampUS != b.TimestampUS:
			return a.TimestampUS < b.TimestampUS
		case a.X != b.X:
			return a.X < b.X
		case a.Y != b.Y:
			return a.Y < b.Y
		case a.Z != b.Z:
			return a.Z < b.Z
		default:
			return out[i].raw < out[j].raw
		}
	})
	return out
}

// keepGrenadesOfPublishedTracks écarte les lancers rattachés à un biped SANS trajectoire
// publiée.
//
// UN LANCER SITUÉ PAR SON PROJECTILE N'EST PAS CONCERNÉ : il ne dépend d'aucune trajectoire,
// sa position est celle de l'objet lancé. Le filtrer reviendrait à jeter une donnée complète
// parce qu'une donnée voisine manque.
func keepGrenadesOfPublishedTracks(gren []Grenade, tracks []Track) []Grenade {
	return keepOfPublishedTracks(gren, tracks, func(g Grenade, published map[uint32]bool) bool {
		return g.Src == GrenadeSrcProjectile || published[g.Slot]
	})
}
