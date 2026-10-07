package replay

// zone_states_owner_nom.go — LE CANAL DE PROPRIETE D'UNE ZONE, DESIGNE PAR SON NOM.
//
// # CE QUE LE FILM DIT
//
// Chaque propriete reseau `ti=13` porte un NOM (`i0`, « propertyName », identifiant de chaine
// R(32)), que l'etat complet de chaque image-cle ecrit et que le balayage pose sur chaque lecture
// d'image-cle ([grammar.ManagedPropertyRead.Name]). Ce nom est l'identite de la propriete, pas son
// numero de slot. Les proprietes d'une zone d'objectif portent des noms qui se repondent — un BLOC
// par zone :
//
//	index de zone (tag 4) · cle de nommage (tag 5) · PROPRIETAIRE (tag 4) · pousseur (tag 4) ·
//	JAUGE (tag 3)
//
// [zoneBlocsNommes] est ce vocabulaire, reduit a ce que le rejeu lit : la CLE, le PROPRIETAIRE, le
// POUSSEUR et la JAUGE de chaque bloc. La jauge d une zone etant appariee par les captures (cf.
// pairGaugeSlots), son nom designe le proprietaire et le pousseur sans vote, sans seuil et sans
// aucune capture concordante — une zone prise une seule fois a son proprietaire et son pousseur
// comme une zone disputee. En colline, le DESIGNATEUR est la cle du bloc de l objet de mode : son
// nom designe le proprietaire de la colline (hillOwnerSlotOf).
//
// # POURQUOI LE NOM ET PAS LE VOISINAGE DE SLOTS
//
// Un numero de slot est un ordre d'allocation du moteur : il dit QUAND la propriete a ete creee,
// pas CE QU'ELLE EST. Le nom le dit. Un nom hors vocabulaire (autre build, mode a plus de zones) ne
// designe rien : la zone retombe alors sur le vote, repli nomme et compte (cf. zoneOwnerSlotsOf).
//
// # CE QUE CE VOCABULAIRE N'EST PAS
//
// Les noms ne sont pas resolus en clair : ce sont des empreintes relevees dans les films, par
// l'instrument `grammar/zone_proprietaire_noms_research_test.go`, et confrontees au vote la ou il
// aboutit (mesure au journal `.ai/thought_log.md`, 2026-10-07).

import (
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// zoneBlocNomme est le nom de chaque propriete que le rejeu lit dans un bloc de zone : la CLE de
// nommage (en colline, le designateur), le PROPRIETAIRE, le POUSSEUR et la JAUGE.
type zoneBlocNomme struct {
	cle, proprietaire, pousseur, jauge uint32
}

// zoneBlocsNommes est le vocabulaire des blocs de zone d'objectif, dans l'ordre des blocs.
var zoneBlocsNommes = []zoneBlocNomme{
	{cle: 1535194732, proprietaire: 904941267, pousseur: 2767827992, jauge: 1868372999},
	{cle: 679735806, proprietaire: 2914281175, pousseur: 1378231552, jauge: 2534649937},
	{cle: 1693197998, proprietaire: 31084060, pousseur: 4114389774, jauge: 316609505},
}

// zoneNoms est le nom de chaque slot lu aux images-cles, et l index inverse nom -> slot.
type zoneNoms struct {
	parSlot, parNom map[uint32]uint32
}

// zoneNomsDesSlots rend le nom de chaque slot lu aux images-cles, et l index inverse nom -> slot.
//
// UN NOM AMBIGU NE DESIGNE RIEN : un slot qui porte deux noms, ou un nom porte par deux slots, est
// ecarte des deux tables — le rattachement n'est publie que s'il est univoque.
func zoneNomsDesSlots(keyReads []grammar.ManagedPropertyRead) zoneNoms {
	parSlot := map[uint32]uint32{}
	conflit := map[uint32]bool{}
	for _, r := range keyReads {
		if !r.Named {
			continue
		}
		slot := r.Slot
		if n, ok := parSlot[slot]; ok && n != r.Name {
			conflit[slot] = true
		}
		parSlot[slot] = r.Name
	}
	parNom := map[uint32]uint32{}
	nomConflit := map[uint32]bool{}
	for _, s := range sortedZoneSlots(parSlot) {
		if conflit[s] {
			delete(parSlot, s)
			continue
		}
		n := parSlot[s]
		if _, ok := parNom[n]; ok {
			nomConflit[n] = true
		}
		parNom[n] = s
	}
	for n := range nomConflit {
		delete(parNom, n)
	}
	for s, n := range parSlot {
		if nomConflit[n] {
			delete(parSlot, s)
		}
	}
	return zoneNoms{parSlot: parSlot, parNom: parNom}
}

// zoneBlocDuSlot rend le bloc du vocabulaire auquel appartient le slot, par le role que son nom y
// tient (`role` extrait ce nom d'un bloc), ou faux quand le slot n'a pas de nom univoque ou que ce
// nom n'est pas au vocabulaire.
func zoneBlocDuSlot(slot uint32, noms zoneNoms, role func(zoneBlocNomme) uint32) (zoneBlocNomme, bool) {
	nom, ok := noms.parSlot[slot]
	if !ok {
		return zoneBlocNomme{}, false
	}
	i := slices.IndexFunc(zoneBlocsNommes, func(b zoneBlocNomme) bool { return role(b) == nom })
	if i < 0 {
		return zoneBlocNomme{}, false
	}
	return zoneBlocsNommes[i], true
}

// zoneProprietaireNomme rend le slot du proprietaire du bloc dont `jauge` est la jauge, ou faux
// quand le nom de la jauge n'est pas au vocabulaire ou que le proprietaire n'est pas dans le film.
func zoneProprietaireNomme(jauge uint32, noms zoneNoms) (uint32, bool) {
	b, ok := zoneBlocDuSlot(jauge, noms, func(b zoneBlocNomme) uint32 { return b.jauge })
	if !ok {
		return 0, false
	}
	slot, ok := noms.parNom[b.proprietaire]
	return slot, ok
}

// zonePousseurNomme rend le slot du POUSSEUR du bloc dont `jauge` est la jauge, ou faux (memes
// cas que [zoneProprietaireNomme]).
func zonePousseurNomme(jauge uint32, noms zoneNoms) (uint32, bool) {
	b, ok := zoneBlocDuSlot(jauge, noms, func(b zoneBlocNomme) uint32 { return b.jauge })
	if !ok {
		return 0, false
	}
	slot, ok := noms.parNom[b.pousseur]
	return slot, ok
}

// zoneProprietaireDeCle rend le slot du proprietaire du bloc dont `cle` est la cle de nommage —
// en colline, le DESIGNATEUR —, ou faux (memes cas que [zoneProprietaireNomme]).
func zoneProprietaireDeCle(cle uint32, noms zoneNoms) (uint32, bool) {
	b, ok := zoneBlocDuSlot(cle, noms, func(b zoneBlocNomme) uint32 { return b.cle })
	if !ok {
		return 0, false
	}
	slot, ok := noms.parNom[b.proprietaire]
	return slot, ok
}

// zoneProprietaires est le resultat du rattachement des canaux de propriete : le canal de chaque
// zone, et ce que le nom et le vote en ont dit.
type zoneProprietaires struct {
	// slot : zone -> canal de propriete retenu.
	slot map[int]uint32
	// nommees : zones dont le canal est designe par le nom.
	nommees int
	// votees : zones dont le canal vient du vote, faute de nom (repli).
	votees int
	// discordantes : zones nommees dont le vote a elu un AUTRE canal.
	discordantes []zoneDiscordance
}

// zoneDiscordance est une zone ou la regle de repli (vote du proprietaire, election du pousseur,
// voisin du designateur en colline) designe un AUTRE canal que le nom. Le nom est retenu.
type zoneDiscordance struct {
	ref int
	// canal nomme le role du canal en cause : [zoneCanalProprietaire] ou [zoneCanalPousseur].
	canal        string
	nomme, autre uint32
}

// Les roles de canal d une discordance, tels que le journal les nomme.
const (
	zoneCanalProprietaire = "proprietaire"
	zoneCanalPousseur     = "pousseur"
)

// zoneOwnerSlotsOf rattache a chaque zone appariee son canal de propriete : PAR LE NOM d'abord, PAR
// LE VOTE en repli.
//
// LE VOTE EST AUSSI LE CONTROLE DU NOM : la ou les deux designent un canal, ils sont confrontes, et
// une discordance se compte (le nom est retenu — c'est ce que le film dit de la propriete ; le vote
// n'en est qu'une correlation). Un canal deja designe par le nom n'est jamais repris par le vote
// d'une autre zone : une zone n'a qu'un proprietaire, un canal qu'une zone.
func zoneOwnerSlotsOf(gaugeSlot, vote map[int]uint32, noms zoneNoms) zoneProprietaires {
	out := zoneProprietaires{slot: map[int]uint32{}}
	tenus := map[uint32]bool{}
	refs := make([]int, 0, len(gaugeSlot))
	for ref := range gaugeSlot {
		refs = append(refs, ref)
	}
	slices.Sort(refs)
	for _, ref := range refs {
		s, ok := zoneProprietaireNomme(gaugeSlot[ref], noms)
		if !ok {
			continue
		}
		out.slot[ref], tenus[s] = s, true
		out.nommees++
		if v, voted := vote[ref]; voted && v != s {
			out.discordantes = append(out.discordantes, zoneDiscordance{ref: ref, canal: zoneCanalProprietaire, nomme: s, autre: v})
		}
	}
	for _, ref := range refs {
		if _, ok := out.slot[ref]; ok {
			continue
		}
		v, ok := vote[ref]
		if !ok || tenus[v] {
			continue
		}
		out.slot[ref], tenus[v] = v, true
		out.votees++
	}
	return out
}
