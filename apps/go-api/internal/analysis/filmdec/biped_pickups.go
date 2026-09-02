package filmdec

import "levelup/go-api/internal/analysis/filmsource"

// biped_pickups.go — LES RAMASSAGES, lus dans l'ÉVÉNEMENT NATIF `biped_pickup` de la bobine.
//
// CE QUE C'EST. Un paquet delta ne porte pas que des records d'entités : il commence par une
// LISTE D'ÉVÉNEMENTS typés, et le type 9 de cette liste est `biped_pickup` — l'enregistrement
// que le moteur écrit quand un bipède ramasse quelque chose. Il donne l'instant (celui du
// paquet, à la milliseconde), le RAMASSEUR et l'identifiant de CATALOGUE de l'objet.
//
// CE QUE ÇA CORRIGE. Le canal `weapon-state-type-info` (i43..i46, cf. held_weapon_changes.go)
// est précis mais il ne voit pas tout : les images-clés révèlent des arrivées d'arme qu'il
// n'explique pas. Et `padPickups` ne publie qu'un intervalle de vingt secondes sans joueur.
// L'événement natif comble les deux : il date à la milliseconde ET il nomme le ramasseur.
//
// POURQUOI CE N'EST PAS UN HOOK, contrairement aux autres canaux de ce paquet. Les hooks
// (`SetHeldWeaponHook`, `SetDesiredWeaponSetHook`…) existent parce que leurs composants se
// lisent PENDANT la traversée d'un record d'entité : il faut se greffer sur le décodeur. La
// liste d'événements, elle, vit AVANT la trame de records, en tête de payload — ces bits ne
// sont lus par AUCUN consommateur existant. Le balayage ci-dessous est donc autonome : il
// n'installe rien, ne modifie aucun global de décodage, et ne peut pas changer ce que les
// autres canaux lisent.
//
// LA GRAMMAIRE, LUE DANS L'EXE (HaloInfinite.exe, chantier RAMASSAGE lot 1) :
//
//	octet de tête = 0xC0 | (type>>1) : le type 9 partage 0xC4 avec le type 8
//	(`biped_board_vehicle`), départagés par le bit 8.
//
//	[1 bit configuration][1 bit continuation][R(7) type = 9]
//	ref0 : R(1) porte ; si 1 : R(8) index + R(2) génération   <- LE RAMASSEUR, domaine 2
//	ref1 : R(1) porte ; si 1 : R(13) + R(2)                    <- domaine 8 (jamais présente)
//	ref2 : R(1) porte ; si 1 : R(13) + R(2)                    <- domaine 7 (jamais présente)
//	R(3) classe · R(1) porte ; si 1 : R(32) identifiant de catalogue
//	[1 bit fin de liste]                        total modal = 50 bits après le champ de type
//
// Sources : table des descripteurs à `ctx+0x210 + type*8` (`FUN_140e453b4`) → type 9 =
// 0x144724e18 ; vtable 0x143d0d758 dont l'entrée +0x08 est l'unique fonction référençant la
// chaîne "biped_pickup" ; domaines à `vtable+0x58` (0x1410f92bc) ; charge à `vtable+0x68`
// (`FUN_141037828`). Le lecteur de référence (`FUN_1406d3140`) reconstruit
// `(gen<<30) | (base + index)` — d'où la base ci-dessous.
//
// CE QUE ÇA NE DONNE PAS :
//
//   - l'INSTANCE monde de l'objet, donc le SOCLE d'origine. L'événement porte l'identifiant de
//     CATALOGUE, pas un handle : mesuré et tranché (l'hypothèse « ref0 = l'objet » est réfutée,
//     `512 + ref0` vaut le slot du RAMASSEUR sur 32/32 paires de vérité terrain).
//   - les événements type 9 qui ne sont PAS EN TÊTE de leur liste. Décoder le deuxième
//     événement d'une liste exige la grammaire de la charge du premier, et toutes les familles
//     ne sont pas percées. Ce balayage voit donc les type 9 dont l'octet de tête vaut 0xC4 ;
//     c'est une borne INFÉRIEURE du rappel, et elle est mesurée (`Stats.MultiEvent`).
//
// HORS LIGNE par construction (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// UN SEUL décodage filmdec à la fois par process (verrou partagé, cf. LockProcessDecode).

const (
	// bipedPickupPacketByte est l'octet de tête d'un paquet dont la liste d'événements
	// s'ouvre sur un type 8 ou 9.
	bipedPickupPacketByte = 0xC4
	// bipedPickupType / bipedBoardVehicleType : les deux types que cet octet porte.
	bipedPickupType       = 9
	bipedBoardVehicleType = 8
	// bipedPickupRefBaseDom2 est la BASE de plage du domaine 2, celle que le lecteur de l'exe
	// ajoute à l'index pour reconstruire la référence. Établie par mesure : l'écart
	// `slot du ramasseur connu - index de ref0` vaut 512 sur 21/21 puis 11/11 paires de
	// vérité terrain (deux films), UNE SEULE valeur distincte, témoin d'appariement permuté à
	// 14-18 %. C'est la même base que celle de la référence domaine 1 de `damage_aftermath`.
	bipedPickupRefBaseDom2 = 512
	// bipedPickupIdxBits est la largeur de l'index du domaine 2. C'est une valeur de RUNTIME
	// (table `DAT_1451f98d0/d4` de l'exe, peuplée au chargement de carte), donc PAS une
	// constante du format — comme FrameConfig.IDLowBits, qui vaut 9 ou 11 selon le film.
	// Balayée contre l'oracle de trame sur [4,14] et sur deux films : 8 rend 60,0 % de trames
	// exactes contre 3,5 % pour la meilleure largeur voisine, sur les DEUX films. Le garde-fou
	// contre un film où elle différerait est le rejet ci-dessous (slot hors bande de bipèdes).
	bipedPickupIdxBits = 8
	// bipedPickupWideRefBits est la largeur des index des domaines 7 et 8 (ref1 et ref2).
	// Aucune des deux n'a jamais été observée présente ; la largeur ne sert qu'à consommer
	// correctement les bits si elle l'était un jour.
	bipedPickupWideRefBits = 13
	// bipedPickupClassBits / bipedPickupCatalogBits : la charge du type 9.
	bipedPickupClassBits   = 3
	bipedPickupCatalogBits = 32
)

// BipedPickup est UN ramassage, daté, attribué et nommé.
type BipedPickup struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk localise l'événement dans le film.
	Chunk int
	// Slot est le slot du bipède RAMASSEUR : il désigne une VIE, pas un joueur. Même espace
	// de slots que HeldWeaponChange.Slot — c'est celui que l'assemblage relie au joueur.
	Slot uint32
	// CatalogID est l'identifiant de CATALOGUE de l'objet ramassé (le R(32) de la charge).
	// Ce n'est PAS un handle du monde : la même valeur se retrouve d'un match à l'autre.
	// Pour les armes, il vaut la FAMILLE d'arme telle que HeldWeaponChange.Family la publie —
	// mesuré : 100 % des familles vues par i43..i46 sont dans l'ensemble des CatalogID.
	CatalogID uint32
	// Class est le R(3) de tête de charge. Il sépare les ramassages d'ARME des autres : les
	// classes 0 et 1 portent une famille d'arme d'i43..i46 dans 63 à 72 % des cas, les classes
	// 2 et 3 dans 0,0 % — sur 118 événements de deux films. Voir BipedPickupIsWeaponClass.
	Class uint8
}

// BipedPickupIsWeaponClass dit si la classe désigne un ramassage d'ARME.
//
// LA SÉPARATION EST MESURÉE, PAS SUPPOSÉE : classes 0 et 1 → 63-72 % d'armes connues du canal
// i43..i46 ; classes 2 et 3 → 0,0 % sur 118 événements, deux films. Ce que les classes 2 et 3
// désignent (équipement, grenades, consommables) n'est pas nommé par le catalogue d'armes.
// Ce que 0 distingue de 1, et 2 de 3, n'est PAS établi.
func BipedPickupIsWeaponClass(c uint8) bool { return c <= 1 }

// BipedPickupStats dit ce que le balayage a vu ET ce qu'il a REFUSÉ. Le second compte autant :
// la largeur d'index du domaine 2 est une valeur de runtime, et un film qui la porterait
// différente produirait des slots hors de la bande de bipèdes. Ces rejets sont la sentinelle.
type BipedPickupStats struct {
	// Packets est le nombre de paquets delta dont l'octet de tête vaut 0xC4.
	Packets int
	// Type9 / Type8 / OtherType ventilent le type lu en tête de liste.
	Type9, Type8, OtherType int
	// Published est le nombre de ramassages rendus.
	Published int
	// MultiEvent compte les listes qui portent un AUTRE événement après le type 9. Il mesure
	// ce que ce balayage ne peut pas voir : un type 9 en deuxième position d'une liste ouverte
	// par une autre famille lui échappe entièrement.
	MultiEvent int
	// RefusedNoRef / RefusedNoCatalog / RefusedOffBand comptent les rejets. Aucun des trois
	// n'a jamais été observé non nul sur le corpus de référence — une valeur non nulle est un
	// signal, pas un détail.
	RefusedNoRef, RefusedNoCatalog, RefusedOffBand int
	// UnexpectedWideRef compte les événements dont ref1 ou ref2 est présente. Jamais observé :
	// une valeur non nulle dénonce un cadrage faux ou un build différent.
	UnexpectedWideRef int
}

// bipedPickupReadRef consomme une référence gardée de largeur w (plus R(2) de génération).
func bipedPickupReadRef(br *BitReader, w uint) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(w)
	br.Skip(2) // génération
	return idx, true
}

// ScanFilmBipedPickups décode tous les ramassages natifs du film de `dir`.
//
// Le balayage est autonome : il ne lit que la tête des payloads delta et n'installe aucun
// hook. Il ne peut donc pas altérer ce que les autres canaux décodent.
//
// ScanFilmBipedPickups est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanBipedPickups].
func ScanFilmBipedPickups(dir string) ([]BipedPickup, BipedPickupStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, BipedPickupStats{}, err
	}
	return ScanBipedPickups(NewFilmContext(film))
}

// ScanBipedPickups décode les ramassages natifs d'un film DEJA CHARGE.
func ScanBipedPickups(fc *FilmContext) ([]BipedPickup, BipedPickupStats, error) {
	var st BipedPickupStats
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	// La bande de bipèdes sert de SENTINELLE : un slot reconstruit hors bande dénonce une
	// largeur d'index inadaptée à ce film. Son absence n'empêche pas de décoder, elle
	// désactive seulement le rejet — et on le dit dans les stats.
	band := fc.BipedSlots()

	var out []BipedPickup
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != bipedPickupPacketByte {
				continue
			}
			st.Packets++
			p, ok := decodeBipedPickup(pay, &st)
			if !ok {
				continue
			}
			if len(band) > 0 && !band[p.Slot] {
				st.RefusedOffBand++
				continue
			}
			p.TimestampUS, p.Chunk = pk.TimestampUS, c
			out = append(out, p)
			st.Published++
		}
	}
	return out, st, nil
}

// decodeBipedPickup consomme l'événement de tête d'un payload 0xC4. Rend (ramassage, ok) ;
// `ok` est faux dès que la lecture n'est pas celle qu'on attend — on ne publie jamais un
// ramassage deviné.
func decodeBipedPickup(pay []byte, st *BipedPickupStats) (BipedPickup, bool) {
	// LE PRÉAMBULE FAIT 9 BITS : configuration(1) + continuation(1) + type R(7). Il se lit en
	// ligne plutôt que par une constante — celle-ci n'avait aucun lecteur et la CI l'a relevée
	// (golangci-lint, `unused`) ; le fait, lui, reste écrit ici.
	br := NewBitReader(pay)
	br.Skip(1)         // bit de configuration
	if !br.ReadBit() { // continuation : un événement suit
		return BipedPickup{}, false // liste vide : impossible pour 0xC4, mais on ne le suppose pas
	}
	switch typ := int(br.ReadBits(7)); typ {
	case bipedPickupType:
		st.Type9++
	case bipedBoardVehicleType:
		st.Type8++
		return BipedPickup{}, false
	default:
		st.OtherType++
		return BipedPickup{}, false
	}
	idx, ok := bipedPickupReadRef(br, bipedPickupIdxBits)
	if !ok {
		st.RefusedNoRef++
		return BipedPickup{}, false
	}
	_, wide1 := bipedPickupReadRef(br, bipedPickupWideRefBits)
	_, wide2 := bipedPickupReadRef(br, bipedPickupWideRefBits)
	if wide1 || wide2 {
		st.UnexpectedWideRef++
	}
	class := uint8(br.ReadBits(bipedPickupClassBits))
	if !br.ReadBit() {
		st.RefusedNoCatalog++
		return BipedPickup{}, false
	}
	catalog := uint32(br.ReadBits(bipedPickupCatalogBits))
	if br.ReadBit() { // fin de liste : 1 = un autre événement suit
		st.MultiEvent++
	}
	return BipedPickup{
		Slot:      uint32(bipedPickupRefBaseDom2 + int(idx)),
		CatalogID: catalog,
		Class:     class,
	}, true
}
