package filmdec

import (
	"fmt"
	"math"
)

// Décodage OFFLINE des ÉVÉNEMENTS DE TIR du film (record d'event type 105).
//
// CE QU'EST CE RECORD (RE Ghidra, dispatcher EventDispatch_Generic_ReadType7 @0x14080AADE) :
// chaque paquet de type 0 commence par un CHAMP DE TYPE D'EVENT sur 7 BITS ; le
// désérialiseur est `vtable[type] + 0x68` (table de handlers @0x144724A90). Le type 105
// pointe sur FUN_14080C1F8 — c'est le « record de dégât » déjà identifié par ailleurs dans
// le projet. Autrement dit : le « fire event » et le « record de dégât 0xd2 » sont LE MÊME
// RECORD, et il n'existe QUE lorsqu'un dégât est appliqué (il n'y a pas de record de tir
// manqué : « touché » est une propriété de tous les records lus ici).
//
// Conséquence pratique majeure : `payload[0] >> 1` donne le type en O(1). Le balayage
// bit-à-bit par « marqueur 11 bits » de l'ancienne génération de scanners est inutile — ce
// prétendu marqueur n'était que l'en-tête du record (type + deux drapeaux constants).
//
// LAYOUT (bitstream MSB-first, offsets en BITS depuis le début du payload, sans aucun
// alignement octet) :
//
//	bits 0..6    (7)  type d'event = 105
//	bit  7       (1)  variante : 0 = record long (porte l'arme) ; 1 = record court
//	bit  8       (1)  drapeau « bloc supplémentaire »
//	bits 9..35        en-tête + préambule de référence d'entité (non modélisé ici)
//	bits 36..40  (5)  INDEX DE L'ATTAQUANT x 2  -> index de joueur = valeur >> 1
//	bits 41..43       cause / slot de dégât
//	bits 44..75  (32) ARME : moitié haute (famille)
//	bits 76..107 (32) ARME : moitié basse (variante)  -> weapon_id 64 bits du projet
//	bits 108..112 (5) cinq drapeaux ; 110 = « compteurs nuls », 111 et 112 = deux portes
//	bits 113..142 (30) VISÉE — direction cubemap 30 bits, lue SEULEMENT si
//	                   bit110 == 1 && bit111 == 0 && bit112 == 0 (chemin « record vide »).
//
// Hors de ce chemin, la visée existe toujours mais après des boucles de longueur VARIABLE
// (composantes de dégât, liste des cibles) dont une largeur vient d'une table remplie au
// runtime : elle n'est donc PAS localisable hors ligne. Ce décodeur ne devine pas — il
// n'expose la visée que sur le chemin sûr (mesuré : 19 % des records longs sur 000d5950,
// 10 % sur 01e1f945).
//
// NON RÉSOLU, à ne pas prétendre : la VICTIME n'est pas décodée (elle vit dans la liste des
// cibles, de largeur runtime). Un record type 105 dit qui tire, avec quoi, quand, et vers
// où — pas qui est touché.

// FireEventType est le type d'event (champ 7 bits) du record de tir / dégât.
const FireEventType = 105

// Offsets et largeurs du record type 105, en bits depuis le début du payload.
const (
	fireVariantBit  = 7
	fireAttackerBit = 36
	fireAttackerW   = 5
	fireWeaponHiBit = 44
	fireWeaponLoBit = 76
	fireWeaponW     = 32
	fireFlagsBit    = 108
	fireFlagsCount  = 5
	fireAimBit      = 113
	// FireHeadBits est la longueur MINIMALE, en bits, d'un payload pour que la tête du
	// record tienne : le dernier champ obligatoire est le cinquième drapeau (bit 112).
	// C'est la garde de decodeFireEvent — voir son commentaire.
	FireHeadBits = fireFlagsBit + fireFlagsCount // 113
	// FireAimBits est la largeur du champ de visée du record long (0x1E au site d'appel
	// FUN_1406D8288 dans FUN_14080C1F8).
	FireAimBits uint = 30
)

// FireEvent est un événement de tir décodé (tête du record type 105).
type FireEvent struct {
	// Chunk / PacketIndex localisent l'event dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet, en microsecondes — MÊME horloge que
	// BipedPosition.TimestampUS, donc directement croisable avec les positions.
	TimestampUS uint64
	// Variant vaut 0 pour le record long (celui qui porte l'arme) et 1 pour le record
	// court. Le court NE porte PAS d'arme (mesuré : 3/313) : ce n'est pas un tir manqué,
	// sa sémantique n'est pas établie — il n'est pas émis par ce décodeur.
	Variant int
	// FilmIndex est l'index de joueur du TIREUR, tel que LE FILM l'écrit (0..7 en arène).
	//
	// LE NOM DIT SON STATUT, ET C'EST DÉLIBÉRÉ. Il s'appelait `PlayerIndex`, ce qui laissait
	// croire à une identité de joueur. C'en est une NUMÉROTATION INTERNE AU FILM : elle ne
	// coïncide avec aucun ordre que nous fabriquons, et surtout pas avec le tri alphabétique
	// du roster. Avoir confondu les deux a produit une « découverte sur le format » qui n'en
	// était pas une — l'écart mesuré était celui de notre propre tri.
	//
	// L'IDENTITÉ D'UN JOUEUR EST SON XUID. Toute jointure passe par lui ; cet index ne sert
	// qu'à regrouper les événements d'un même tireur À L'INTÉRIEUR d'un film.
	FilmIndex int
	// WeaponID est l'identifiant global 64 bits de l'arme : clé directe de
	// metadata.weapon_labels.weapon_id et de analysis.WeaponIDToName.
	WeaponID uint64
	// Flags porte les bits 108..112 (diagnostic).
	Flags [fireFlagsCount]uint8
	// HasAim indique que la visée a pu être lue (chemin « record vide » uniquement).
	HasAim bool
	// Aim est la direction UNITAIRE monde du tir (cubemap 30 bits déquantifié).
	Aim [3]float32
}

// ScanFilmFireEvents décode tous les événements de tir LONGS des chunks du film de dir.
// Aucun balayage bit-à-bit : un paquet de type 0 = un event, son type est dans le premier
// octet. Les chunks illisibles sont ignorés (le film peut être partiel).
func ScanFilmFireEvents(dir string) ([]FireEvent, error) {
	n := CountFilmChunks(dir)
	var out []FireEvent
	read := 0
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		read++
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != FireEventType || int(pay[0])&1 != 0 {
				continue // autre type d'event, ou variante courte (sans arme)
			}
			e, ok := decodeFireEvent(pay)
			if !ok {
				continue // record tronqué : voir la garde de decodeFireEvent
			}
			e.Chunk, e.PacketIndex, e.TimestampUS = c, p.Index, p.TimestampUS
			out = append(out, e)
		}
	}
	if read == 0 {
		return nil, fmt.Errorf("aucun chunk film lisible dans %s", dir)
	}
	return out, nil
}

// ReadAttackerIndex lit l'index d'attaquant d'un payload de record type 105, aux MÊMES offsets
// que `decodeFireEvent`. Rend -1 si le payload est trop court.
//
// EXPORTÉ POUR LA RECHERCHE, ET POUR UNE RAISON PRÉCISE. La variante COURTE du record 105 n'est
// pas émise par ce décodeur (sa sémantique n'est pas établie), mais savoir si elle porte un
// index d'attaquant AU MÊME ENDROIT est justement ce qui permettra de trancher ce qu'elle est.
// Sans cet accesseur, l'instrument de recherche devrait recopier `fireAttackerBit` et
// `fireAttackerW` — une seconde copie d'un offset de bit, exactement ce que ce chantier a payé
// cher ailleurs. Un seul endroit déclare ces offsets, et c'est ce fichier.
func ReadAttackerIndex(pay []byte) int {
	if len(pay)*8 < fireAttackerBit+fireAttackerW {
		return -1
	}
	return int(readBitsAt(pay, fireAttackerBit, fireAttackerW)) >> 1
}

// decodeFireEvent lit la tête du record type 105 d'un payload de paquet. Rend ok=false, sans
// rien lire, si le payload est trop court pour porter cette tête.
//
// LA GARDE N'EST PAS DÉFENSIVE « au cas où » — le chemin était atteignable en production.
// `ScanFilmFireEvents` n'exige que `p.Size >= 1` ; un paquet delta tronqué dont le premier
// octet vaut 0xD2 (type 105, variante longue) suffisait à faire lire jusqu'au bit 112, et
// `readBitsAt` indexe le tableau SANS borne — contrairement à `PeekBits`, qui rend 0 au-delà.
// Un film tronqué par un téléchargement partiel faisait donc paniquer le décodeur ; en J4 il
// tourne dans un collecteur de fond du process de sync, où une panique coûte le process entier.
//
// Le paquet a été audité pour ce même motif : ses frères sont sains. `scanGrenadeThrows` et
// `scanProjectileRecords` bornent leur balayage (`len(pay)*8 - <bits du record>`) ET lisent par
// `PeekBits` ; `offline_aim.go` teste `at+n > total` avant chaque composant ; `offline_biped.go`
// et `i0_layout.go` bornent leur boucle sur `total`. `decodeFireEvent` était le seul à lire à
// des offsets FIXES derrière une garde de taille qui ne les couvrait pas.
func decodeFireEvent(pay []byte) (FireEvent, bool) {
	if len(pay)*8 < FireHeadBits {
		return FireEvent{}, false
	}
	e := FireEvent{Variant: int(pay[0]) & 1}
	e.FilmIndex = int(readBitsAt(pay, fireAttackerBit, fireAttackerW)) >> 1
	e.WeaponID = uint64(readBitsAt(pay, fireWeaponHiBit, fireWeaponW))<<32 |
		uint64(readBitsAt(pay, fireWeaponLoBit, fireWeaponW))
	for i := 0; i < fireFlagsCount; i++ {
		e.Flags[i] = uint8(readBitsAt(pay, fireFlagsBit+i, 1))
	}
	// Chemin sûr uniquement : les trois drapeaux qui commandent la position du champ.
	if e.Flags[2] == 1 && e.Flags[3] == 0 && e.Flags[4] == 0 &&
		len(pay)*8 >= fireAimBit+int(FireAimBits) {
		if v, ok := DecodeAimVectorChecked(readBitsAt(pay, fireAimBit, int(FireAimBits)), FireAimBits); ok {
			e.HasAim, e.Aim = true, v
		}
	}
	return e, true
}

// AimHeadingDeg renvoie le cap de visée du tir dans le plan XY, en degrés dans [0,360[,
// même origine et même sens que atan2(Y, X) des positions déquantifiées.
func (e FireEvent) AimHeadingDeg() (float64, bool) {
	if !e.HasAim {
		return 0, false
	}
	h := math.Atan2(float64(e.Aim[1]), float64(e.Aim[0])) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h, true
}
