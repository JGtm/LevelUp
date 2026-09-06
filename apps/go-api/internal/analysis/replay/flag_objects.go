package replay

// flag_objects.go — LES VIES LIBRES DU DRAPEAU : la piste de l'objet quand PERSONNE ne le porte.
//
// # Le principe, et pourquoi il ne double pas `flag_carries.go`
//
// `flag_carries.go` lit le PORTEUR : un drapeau porte est a la position de son porteur, et rien
// de l'objet n'y est decode. Ce fichier-ci lit l'OBJET, et seulement quand il est LIBRE. Les
// deux se completent exactement la ou l'autre est aveugle :
//
//	PORTE    l'objet cesse de repliquer sa position (il suit son porteur) ; le calque des
//	         portages sait ou il est, le calque de l'objet ne voit rien.
//	LIBRE    l'objet replique sa position dans les paquets delta, comme tout objet du monde ;
//	         le calque des portages ne sait rien de lui (le lacher volontaire n'est date par
//	         aucun evenement), le calque de l'objet le suit a l'image.
//
// # D'ou vient la matiere, et pourquoi aucune lecture de film ne s'ajoute
//
// Le drapeau est un objet de l'archetype `ti=42` — LE MEME que les armes au sol. Le balayage du
// calque des socles (`build_ground_weapons.go`) rend donc deja tout : les records de CREATION
// (l'instant et le lieu ou l'objet apparait) et les PISTES delta de la meme bande de slots. Ce
// fichier ne fait que les trier par identite, et l'identite vient du manifeste du titre
// (`LabelCatalog.ObjectiveObjects`) — jamais d'une constante ecrite ici.
//
// # Ce qu'est une VIE LIBRE, en une phrase
//
// Un record de creation, puis la piste delta de la MEME vie (slot, generation) jusqu'a ce
// qu'elle cesse — c'est-a-dire jusqu'a ce que quelqu'un le ramasse ou qu'il s'immobilise. La
// regle d'appariement creation -> piste est celle des armes au sol (`gwPickupLifeTrack`), et
// c'est deliberement la MEME : deux ecritures de cet appariement divergeraient au premier
// correctif, et le depot l'interdit.
//
// # CE FICHIER NE PUBLIE PAS LA PISTE, ET C'EST UN RESULTAT DE MESURE (2026-08-18)
//
// La phase 2 du plan devait publier ces vies (`flagObjects`). Le CONTROLE 3, ecrit AVANT la
// mesure, exigeait que >= 90 % d'entre elles naissent a moins de 1,5 m d'un `flag_spawn` OU du
// porteur qui vient de finir. MESURE SUR LES TROIS FILMS CTF : 149/197 = 75,6 % — NON TENU. Le
// temoin, lui, tient largement (creations `ti=42` d'armes ordinaires soumises a la MEME regle :
// 122/950 = 12,8 %, seuil <= 20 %), donc la piste discrimine bel et bien — d'un facteur six —
// mais un quart des vies reste inexplique, et le diagnostic ecarte la re-creation sur place
// (3 cas sur 48). `flagObjects` n'est donc PAS publie : le detail vit dans
// `drapeau_objet_controle_test.go`, et le registre des reports porte la condition de reprise.
//
// # CE QUI EST LIVRE MALGRE TOUT, ET POURQUOI CE N'EST PAS UNE ENTORSE (arbitrage du 2026-08-18)
//
// Les deux CORRECTIONS que ces vies apportent au calque des portages sont livrees. Elles ne
// consomment pas la population que le controle refuse : elles ne se declenchent QUE sur les vies
// nees AUX PIEDS D'UN PORTEUR — c'est-a-dire exactement la sous-population que le controle
// VALIDE (la branche « porteur » de ses 75,6 %). Une vie nee a un socle est explicitement
// ecartee (`flagFreeNearSpawn`), une vie nee ailleurs ne passe pas la distance au porteur.
//
//	le LACHER VOLONTAIRE SE DATE — un portage que rien ne fermait (`carried_open`, une borne
//	  haute qui courait jusqu'a la fin de l'axe) se ferme a l'instant ou l'objet reapparait aux
//	  pieds de son porteur ;
//	le LACHER CHANGE DE PLACE — `dropped` passe de la derniere position du PORTEUR au dernier
//	  point de la piste LIBRE, la ou l'objet repose apres sa chute.
//
// LE CONTENU DE `flagCarries` CHANGE SANS QU'AUCUNE CLE NE BOUGE : c'est pour cela que le schema
// monte a 15. Un artefact 14 se lit « a re-cuire », pas « a jour ».

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// flagFreeLife est UNE vie libre : la creation, puis la piste repliquee jusqu'a sa fin.
type flagFreeLife struct {
	// ID est l'identifiant d'objet du manifeste (le mot MPP de 32 bits).
	ID uint32
	// Key est la vie au sens du film — LA PAIRE (slot, generation), jamais le slot seul.
	Key filmdec.EquipmentLifeKey
	// T0US est l'instant de CREATION, T1US le dernier instant REPLIQUE. Egaux quand la vie
	// n'a laisse aucun echantillon de position : l'objet est ne immobile (a son socle) et n'a
	// jamais bouge, ce qui est une vie libre parfaitement reelle, reduite a un point.
	T0US, T1US uint64
	// Pts sont les echantillons de position de la vie, tries, le premier etant celui de la
	// CREATION (le record de creation porte lui-meme sa position i0).
	Pts []flagFreeSample
}

// flagFreeSample est une position datee d'un objet libre.
type flagFreeSample struct {
	TUS  uint64
	X, Y float32
}

// First rend la position de CREATION de la vie.
func (l flagFreeLife) First() (float32, float32) {
	if len(l.Pts) == 0 {
		return 0, 0
	}
	return l.Pts[0].X, l.Pts[0].Y
}

// Last rend la DERNIERE position repliquee de la vie — la ou l'objet se trouve quand sa piste
// s'arrete, c'est-a-dire la ou il repose ou bien la ou on l'a pris.
func (l flagFreeLife) Last() (float32, float32) {
	if len(l.Pts) == 0 {
		return 0, 0
	}
	p := l.Pts[len(l.Pts)-1]
	return p.X, p.Y
}

// flagFreeLives rend les vies LIBRES des objets que le titre declare drapeaux, triees.
//
// TABLE VIDE, SCAN NON ABOUTI : aucune vie. Le calque se tait entierement plutot que de publier
// une piste dont on ne sait pas de quel objet elle est.
//
// LA FIN DE VIE D'UNE CLE EST LA CREATION SUIVANTE DE LA MEME CLE, et sinon l'infini. C'est la
// regle des armes au sol, et elle est ici SUFFISANTE : les pistes rendues par le decodeur sont
// deja decoupees par vie (`splitLives`), donc l'appariement par instant de depart le plus proche
// ne peut pas prendre la piste d'une vie ulterieure tant qu'une creation les separe.
func flagFreeLives(scan WorldObjectScan, flags map[uint32]Label) []flagFreeLife {
	if !scan.Scanned || len(flags) == 0 {
		return nil
	}
	byKey := map[filmdec.EquipmentLifeKey][]filmdec.EquipmentCreation{}
	ids := map[filmdec.EquipmentLifeKey]uint32{}
	for _, c := range scan.Creations {
		w, ok := gwPadsIdentity(c)
		if !ok || flags[w] == (Label{}) {
			continue
		}
		k := filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}
		byKey[k], ids[k] = append(byKey[k], c), w
	}
	tracks := gwTracksByKey(scan.Tracks)
	out := make([]flagFreeLife, 0, len(scan.Creations))
	for k, list := range byKey {
		sort.Slice(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i, c := range list {
			lifeEnd := uint64(math.MaxUint64)
			if i+1 < len(list) {
				lifeEnd = list[i+1].TimestampUS
			}
			out = append(out, flagFreeLifeOf(ids[k], k, c, tracks[k], lifeEnd))
		}
	}
	sort.Slice(out, func(i, j int) bool { return flagFreeLess(out[i], out[j]) })
	return out
}

// flagFreeLifeOf assemble UNE vie libre : sa creation, puis sa piste si elle en a une.
func flagFreeLifeOf(id uint32, k filmdec.EquipmentLifeKey, c filmdec.EquipmentCreation,
	tracks []filmdec.ProjectileTrack, lifeEnd uint64) flagFreeLife {
	l := flagFreeLife{ID: id, Key: k, T0US: c.TimestampUS, T1US: c.TimestampUS,
		Pts: []flagFreeSample{{TUS: c.TimestampUS, X: c.X, Y: c.Y}}}
	tr, moved := gwPickupLifeTrack(tracks, c.TimestampUS, lifeEnd)
	if !moved {
		return l
	}
	for _, p := range tr.Pts {
		if p.TimestampUS <= l.Pts[len(l.Pts)-1].TUS {
			continue // le premier point replique coincide avec la creation : un seul point
		}
		l.Pts = append(l.Pts, flagFreeSample{TUS: p.TimestampUS, X: p.X, Y: p.Y})
	}
	l.T1US = l.Pts[len(l.Pts)-1].TUS
	return l
}

// flagFreeLess est l'ordre TOTAL des vies libres : instant, puis slot et generation, PUIS la vie
// elle-meme. Sans lui, le parcours de map rendrait une sortie differente a chaque execution.
//
// POURQUOI LE DEPARTAGE PAR LA VIE (correction du 2026-09-02, item 0.4bis etendu de
// PLAN_CUISSON_PERF). Le triplet de tete N'EST PAS total : `out` est bati en iterant la MAP
// `byKey`, et une meme cle (slot, generation) peut porter DEUX creations au MEME instant — la
// fin de vie est alors la creation suivante, donc l'instant lui-meme, et les deux vies sont
// strictement ex aequo sur (T0US, slot, generation) tout en portant des positions differentes.
// `sort.Slice` n'etant pas stable, leur rang etait tire au sort a chaque execution.
//
// L'ORDRE N'EST PAS COSMETIQUE : `buildObjectiveObjects` retrie cette tranche avec un
// `sort.SliceStable`, qui RECONDUIT l'ordre d'entree pour les ex aequo — l'alea se serait donc
// propage tel quel jusqu'a l'artefact. Le departage n'utilise QUE des donnees de la vie (fin,
// identite d'objet, puis les echantillons dans l'ordre) : jamais une adresse memoire ni le rang
// d'iteration de la map. Deux vies que ce comparateur ne separe pas sont identiques champ pour
// champ. Meme patron que `lessTrack` (filmdec/projectiles.go) et `lessPlacement`
// (filmdec/equipment_placements.go).
func flagFreeLess(a, b flagFreeLife) bool {
	switch {
	case a.T0US != b.T0US:
		return a.T0US < b.T0US
	case a.Key.Slot != b.Key.Slot:
		return a.Key.Slot < b.Key.Slot
	case a.Key.Gen != b.Key.Gen:
		return a.Key.Gen < b.Key.Gen
	case a.T1US != b.T1US:
		return a.T1US < b.T1US
	case a.ID != b.ID:
		return a.ID < b.ID
	case len(a.Pts) != len(b.Pts):
		return len(a.Pts) < len(b.Pts)
	}
	for i := range a.Pts {
		if a.Pts[i] != b.Pts[i] {
			return flagFreeSampleLess(a.Pts[i], b.Pts[i])
		}
	}
	return false
}

// flagFreeSampleLess ordonne deux echantillons sur TOUS leurs champs — c'est ce qui rend le
// departage de `flagFreeLess` independant de l'ordre d'arrivee.
func flagFreeSampleLess(a, b flagFreeSample) bool {
	switch {
	case a.TUS != b.TUS:
		return a.TUS < b.TUS
	case a.X != b.X:
		return a.X < b.X
	}
	return a.Y < b.Y
}

// flagFreeDropWindowMS — l'ecart maximal, en millisecondes, entre la fin d'un portage et la
// naissance de la vie libre qui l'explique : UNE SECONDE.
//
// ECRIT AVANT LA MESURE (plan `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md`, controle 3). Le lacher
// est un evenement PHYSIQUE — le porteur tombe, l'objet est recree la — et non une transition de
// compteur : l'axe du rejeu avance par pas de 100 ms, les deux horloges (match et film) sont
// calees a la frame pres. Une seconde laisse dix pas de marge sans jamais rattraper le portage
// PRECEDENT, qui dure des dizaines de secondes.
//
// LA DISTANCE N'EST PAS UN SEUIL NEUF : c'est `originDropMaxDist` (1,5 m), celui de la regle du
// lacher, declare chez son proprietaire (equipment_placements.go). Les deux conditions valent
// ensemble — une vie libre qui commence au bon moment MAIS a l'autre bout de la carte n'est pas
// le drapeau que ce porteur vient de lacher.
const flagFreeDropWindowMS = 1000

// flagFreeNearSpawn dit si une vie libre nait A UN SOCLE.
//
// POURQUOI CE REFUS EST LA CONDITION DE TOUT CE QUI SUIT (arbitrage du 2026-08-18). Les deux
// corrections ci-dessous ne s'appliquent QU'AUX vies nees aux pieds d'un porteur — la seule
// sous-population que le controle 3 valide (les « porteur » tenues). Une vie nee a la base n'est
// pas un lacher : c'est un drapeau qui rentre, et un porteur tue juste devant le socle adverse
// suffirait a la confondre avec le sien. On l'ecarte d'abord, on regarde le porteur ensuite.
//
// LES SOCLES SONT CEUX QUE LA PRODUCTION CONNAIT (socles d'EQUIPE, cf. replaybuild/flagspawns.go).
// Carte hors catalogue : aucun socle, donc aucun refus — et la regle retombe sur la seule
// condition de distance au porteur, qui reste la bonne.
func flagFreeNearSpawn(spawns []FlagSpawn, x, y float32) bool {
	for _, s := range spawns {
		if sqDist(s.X, s.Y, x, y) <= originDropMaxDist*originDropMaxDist {
			return true
		}
	}
	return false
}

// closeByFreeLives DATE LE LACHER VOLONTAIRE — ce que rien d'autre ne sait faire.
//
// LA REGLE, ET POURQUOI ELLE EST SURE. Un portage que rien ne ferme court jusqu'a la fin de
// l'axe : c'est une BORNE HAUTE, publiee `carried_open`. Si l'objet drapeau REAPPARAIT pendant ce
// portage, AUX PIEDS du porteur, c'est qu'il ne le porte plus — un objet porte ne replique pas sa
// position, et un objet qui renait ailleurs n'est pas celui-ci. Les trois conditions valent
// ensemble : fenetre STRICTEMENT interieure au portage, distance au porteur, et naissance qui
// n'est PAS un socle.
//
// LE PORTEUR EST LU SUR SA PISTE PUBLIEE, la meme que celle sur laquelle le client dessine :
// c'est la seule position dont on soit sur qu'elle existe au rendu.
func closeByFreeLives(raws []flagCarryRaw, ctx flagCarryCtx, scan FlagCarryScan) ([]flagCarryRaw, int) {
	if len(scan.Free) == 0 {
		return raws, 0
	}
	idx := tracksByXUID(ctx.tracks, ctx.slotXUID)
	closed := 0
	for i := range raws {
		if raws[i].closed {
			continue
		}
		at, ok := flagFreeDropInside(raws[i], ctx, idx[raws[i].xuid], scan)
		if !ok {
			continue
		}
		raws[i].t1, raws[i].closed, raws[i].captured = at, true, false
		closed++
	}
	return raws, closed
}

// flagFreeDropInside rend l'instant (horloge du MATCH) de la PREMIERE vie libre qui commence
// pendant le portage, aux pieds de son porteur et hors de tout socle.
func flagFreeDropInside(r flagCarryRaw, ctx flagCarryCtx, tracks []Track,
	scan FlagCarryScan) (int64, bool) {
	for _, l := range scan.Free {
		f := frameOf(l.T0US, ctx.origin, ctx.step)
		at := ctx.matchMSOfFrame(f)
		if at <= r.t0 || at >= r.t1 {
			continue
		}
		x, y := l.First()
		if flagFreeNearSpawn(scan.Spawns, x, y) {
			continue
		}
		p, ok := pointOfXUIDAt(tracks, f)
		if !ok || sqDist(p.X, p.Y, x, y) > originDropMaxDist*originDropMaxDist {
			continue
		}
		return at, true
	}
	return 0, false
}

// repositionFlagDrops remplace la position de LACHER par le dernier point de la piste LIBRE.
//
// POURQUOI CE N'EST PAS LA MEME POSITION. `dropped` valait la derniere position du PORTEUR :
// c'est la ou il est mort, pas la ou l'objet repose. Un drapeau tombe, roule et s'immobilise ;
// sa piste libre le suit jusqu'a ce qu'il cesse d'emettre, c'est-a-dire jusqu'a son repos.
//
// UNE CAPTURE N'EST PAS UN LACHER : le drapeau rentre a sa base, et sa position vient du socle.
// Elle n'est donc jamais repositionnee.
func repositionFlagDrops(raws []flagCarryRaw, ctx flagCarryCtx, scan FlagCarryScan) int {
	if len(scan.Free) == 0 {
		return 0
	}
	moved := 0
	for i := range raws {
		if !raws[i].closed || raws[i].captured {
			continue
		}
		l, ok := flagFreeAtDrop(raws[i], ctx, scan)
		if !ok {
			continue
		}
		x, y := l.Last()
		if x == raws[i].x1 && y == raws[i].y1 {
			continue
		}
		raws[i].x1, raws[i].y1 = x, y
		moved++
	}
	return moved
}

// flagFreeAtDrop rend la vie libre qui EXPLIQUE ce lacher : nee dans la seconde qui l'entoure, a
// moins de `originDropMaxDist` du point de lacher publie, et pas a un socle. La plus proche en
// temps l'emporte.
//
// LA FENETRE EST SYMETRIQUE : on ne suppose pas l'ordre entre l'instant que les compteurs du
// statborg datent et celui ou le film cree l'objet, on mesure leur voisinage.
func flagFreeAtDrop(r flagCarryRaw, ctx flagCarryCtx, scan FlagCarryScan) (flagFreeLife, bool) {
	var best flagFreeLife
	bestGap, found := int64(math.MaxInt64), false
	for _, l := range scan.Free {
		at := ctx.matchMSOfFrame(frameOf(l.T0US, ctx.origin, ctx.step))
		gap := at - r.t1
		if gap < 0 {
			gap = -gap
		}
		if gap > flagFreeDropWindowMS || gap >= bestGap {
			continue
		}
		x, y := l.First()
		if flagFreeNearSpawn(scan.Spawns, x, y) {
			continue
		}
		if sqDist(r.x1, r.y1, x, y) > originDropMaxDist*originDropMaxDist {
			continue
		}
		best, bestGap, found = l, gap, true
	}
	return best, found
}

// flagHomecoming est LE DRAPEAU QUI RENTRE, DATE ET NOMME : la naissance d'une vie libre A UN
// SOCLE.
//
// # POURQUOI CETTE LECTURE EST NEUVE, ET CE QU'ELLE DEBLOQUE
//
// Le retour AUTOMATIQUE — le drapeau que plus personne ne touche et que le jeu ramene chez lui —
// n'est credite a AUCUN joueur : le statborg n'en porte rien, et le calque le laissait donc au
// sol jusqu'a la fin de l'axe (des laches de plus de deux minutes, qui n'ont jamais existe a
// l'ecran). La tentative anterieure cherchait le delai par un PROXY (l'ecart entre le lacher et
// la prise suivante au socle) et l'a trouve trop disperse pour trancher.
//
// L'OBJET, LUI, LE DIT. Un drapeau qui rentre est RE-CREE a son socle : c'est un record de
// creation, date a la frame, et le SOCLE LE NOMME — ce que `flag_returns` ne fait pas (cf.
// `applyFlagReturn`, qui s'abstient des que deux drapeaux sont au sol).
//
// LA MEME DISTANCE QUE PARTOUT : `originDropMaxDist`, celle de la regle du lacher. Aucun seuil
// neuf n'est introduit ici.
type flagHomecoming struct {
	// flag est l'indice du socle, donc du drapeau.
	flag int
	// at est l'instant sur l'horloge du MATCH, la meme que celle des portages.
	at int64
	// x, y est le point de naissance — il sert a ecarter le drapeau ADVERSE qui gisait la.
	x, y float32
}

// flagObjectHomecomings rend, triees, les rentrees que l'objet DATE. Vides quand la carte est
// hors du catalogue d'objectifs (aucun socle : aucune rentree ne se nomme).
func flagObjectHomecomings(scan FlagCarryScan, ctx flagCarryCtx) []flagHomecoming {
	if len(scan.Free) == 0 || len(scan.Spawns) == 0 {
		return nil
	}
	out := make([]flagHomecoming, 0, len(scan.Free))
	for _, l := range scan.Free {
		x, y := l.First()
		f, ok := flagSpawnAt(scan.Spawns, x, y)
		if !ok {
			continue
		}
		at := ctx.matchMSOfFrame(frameOf(l.T0US, ctx.origin, ctx.step))
		out = append(out, flagHomecoming{flag: f, at: at, x: x, y: y})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// flagSpawnAt rend l'indice du socle LE PLUS PROCHE d'un point, s'il est a moins de
// [originDropMaxDist]. Le plus proche, et non le premier : deux socles peuvent se toucher sur
// une carte etroite, et un drapeau ne rentre que chez lui.
func flagSpawnAt(spawns []FlagSpawn, x, y float32) (int, bool) {
	best, bestD := -1, float64(originDropMaxDist*originDropMaxDist)
	for i, s := range spawns {
		if d := sqDist(s.X, s.Y, x, y); d <= bestD {
			best, bestD = i, d
		}
	}
	return best, best >= 0
}
