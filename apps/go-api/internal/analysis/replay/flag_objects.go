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
// (`LabelCatalog.FlagObjects`) — jamais d'une constante ecrite ici.
//
// # Ce qu'est une VIE LIBRE, en une phrase
//
// Un record de creation, puis la piste delta de la MEME vie (slot, generation) jusqu'a ce
// qu'elle cesse — c'est-a-dire jusqu'a ce que quelqu'un le ramasse ou qu'il s'immobilise. La
// regle d'appariement creation -> piste est celle des armes au sol (`gwPickupLifeTrack`), et
// c'est deliberement la MEME : deux ecritures de cet appariement divergeraient au premier
// correctif, et le depot l'interdit.
//
// # CE FICHIER NE PUBLIE RIEN, ET C'EST UN RESULTAT DE MESURE (2026-08-18)
//
// La phase 2 du plan devait publier ces vies (`flagObjects`), dater le lacher volontaire et
// reposer l'etat `dropped` sur la piste libre. Le CONTROLE 3, ecrit AVANT la mesure, exigeait
// que >= 90 % des vies libres naissent a moins de 1,5 m d'un `flag_spawn` OU du porteur qui
// vient de finir. MESURE SUR LES TROIS FILMS CTF : 149/197 = 75,6 % — NON TENU. Le temoin, lui,
// tient largement (creations `ti=42` d'armes ordinaires : 122/950 = 12,8 %, seuil <= 20 %), donc
// la piste discrimine bel et bien — d'un facteur six — mais un quart des vies reste inexplique,
// et le diagnostic ecarte la re-creation sur place (3 cas sur 48).
//
// LA REGLE DU PLAN S'APPLIQUE TELLE QU'ELLE A ETE ECRITE : negatif ecrit, `flagObjects` non
// publie. Ce qui reste ici est ce que l'INSTRUMENT du controle appelle — la definition d'une vie
// libre et la fenetre de lacher —, rien de plus. Le detail de la mesure vit dans
// `drapeau_objet_controle_test.go` ; le registre des reports porte la condition de reprise.

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
func flagFreeLives(scan GroundWeaponScan, flags map[uint32]Label) []flagFreeLife {
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

// flagFreeLess est l'ordre TOTAL des vies libres : instant, puis slot et generation. Sans lui,
// le parcours de map rendrait une sortie differente a chaque execution.
func flagFreeLess(a, b flagFreeLife) bool {
	switch {
	case a.T0US != b.T0US:
		return a.T0US < b.T0US
	case a.Key.Slot != b.Key.Slot:
		return a.Key.Slot < b.Key.Slot
	}
	return a.Key.Gen < b.Key.Gen
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
