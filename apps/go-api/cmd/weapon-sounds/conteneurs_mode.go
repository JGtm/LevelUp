package main

// conteneurs_mode.go — LE MODE DE LECTURE D'UN CONTENEUR DE TYPE 5, et pourquoi il manquait.
//
// LE CINQUIEME OUBLI DE FORMAT. Le type 5 de Wwise s'appelle `CAkRanSeqCntr` : il couvre DEUX
// comportements, et le champ `eMode` dit lequel.
//
//	eMode = 0  ALEATOIRE  : les enfants sont des VARIANTES, le moteur en TIRE une
//	eMode = 1  SEQUENCE   : les enfants sont des PHASES, le moteur les joue DANS L'ORDRE
//
// Tout l'outil suppose le premier depuis son premier jour (`arbre.go` : « Random (type 5,
// mode aleatoire) »), et l'audit n'a jamais mesure que la table de POIDS — laquelle ne dit
// rien du mode. Sur un conteneur en SEQUENCE, rendre « une variante » ne rend donc pas un
// tiers du geste : il rend UN TIERS DU GESTE, ce qui n'est pas la meme chose qu'une variante
// et ne ressemble a rien.
//
// CE QUI A DECLENCHE LA MESURE, et c'est une oreille : sur les 23 gestes du translocateur
// quantique rendus le 2026-08-26, l'utilisateur n'en reconnait AUCUN comme la pose de la
// balise, et decrit le vrai geste ainsi — « c'est comme si on le chargeait, ca monte en
// intensite, et ensuite il est pose ». Une montee suivie d'une pose, c'est la description
// exacte d'une SEQUENCE de phases, pas d'un tirage de variantes.
//
// L'ANCRE EST LA LISTE D'ENFANTS, comme partout ailleurs dans ce fichier-ci et ses voisins.
// Dans `AkRanSeqCntrInitialValues`, les quatre octets qui PRECEDENT immediatement le nombre
// d'enfants sont, dans l'ordre :
//
//	u16 wAvoidRepeatCount | u8 eTransitionMode | u8 eRandomMode | u8 eMode | u8 byBitVector
//	                        <---------------------- off-4 .. off-1 ---------------------->
//	u32 ulNumChilds  <- `off`, deja localise par `positionEnfants`
//	... ids des enfants ...
//	u16 ulNumPlaylistItem | { u32 ulPlayID, s32 weight } * n   <- deja lu par `lirePoidsAleatoire`
//
// La position est donc DEDUITE de deux ancres deja validees (la liste d'enfants et la table
// de poids qui la suit immediatement), pas postulee.
//
// LE CONTROLE EST ECRIT AVANT LA MESURE, et il est refutable : si l'offset est faux, `eMode`
// prendra des valeurs hors {0, 1} et le drapeau des bits sortira de ses cinq bits. Le mode
// `audit-modes` imprime ces deux taux de PLAUSIBILITE avant tout resultat — un decodage qui
// ne les tient pas ne doit rien conclure.

import "encoding/binary"

// Les bits de `byBitVector`, dans l'ordre du SDK.
const (
	bitUtilisePoids   = 1 << 0 // bIsUsingWeight
	bitRemetAChaqueOn = 1 << 1 // bResetPlayListAtEachPlay
	bitRepartArriere  = 1 << 2 // bIsRestartBackward
	bitContinu        = 1 << 3 // bIsContinuous
	bitGlobal         = 1 << 4 // bIsGlobal
	bitsConnus        = bitUtilisePoids | bitRemetAChaqueOn | bitRepartArriere | bitContinu | bitGlobal
)

// modeRanSeq : ce qu'on sait lire du mode de lecture d'un conteneur de type 5.
type modeRanSeq struct {
	Lu bool
	// Sequence : les enfants sont des PHASES jouees dans l'ordre, pas des variantes.
	Sequence bool
	// Continu : la sequence (ou le tirage) enchaine ses elements dans UNE lecture. `false`
	// = mode « pas a pas », un element par declenchement.
	Continu bool
	// UtilisePoids : la table de poids est effectivement consultee (elle existe toujours).
	UtilisePoids bool
	// Bits : le drapeau brut, garde pour que le controle de plausibilite reste verifiable.
	Bits    byte
	Enfants int
}

// PhasesEnchainees dit si le conteneur joue ses enfants BOUT A BOUT en une seule lecture.
// C'est la seule question a laquelle le rendu doit repondre : dans ce cas, un geste ne se
// rend pas en tirant un enfant, il se rend en les CONCATENANT.
func (m modeRanSeq) PhasesEnchainees() bool { return m.Lu && m.Sequence && m.Continu }

// lireModeRanSeq lit `eMode` et `byBitVector` a l'ancre de la liste d'enfants.
//
// Rend `Lu: false` des que la lecture n'est pas PLAUSIBLE : `eMode` hors {0, 1} ou des bits
// inconnus dans le drapeau. Un decodage douteux doit se declarer douteux — c'est ce qui rend
// le taux de plausibilite du mode `audit-modes` interpretable.
func lireModeRanSeq(d []byte, connu func(uint32) bool) modeRanSeq {
	off, n := positionEnfants(d, connu)
	if off < 4 || n < 1 {
		return modeRanSeq{}
	}
	eMode := d[off-2]
	bits := d[off-1]
	if eMode > 1 || bits&^bitsConnus != 0 {
		return modeRanSeq{Bits: bits, Enfants: n}
	}
	return modeRanSeq{
		Lu:           true,
		Sequence:     eMode == 1,
		Continu:      bits&bitContinu != 0,
		UtilisePoids: bits&bitUtilisePoids != 0,
		Bits:         bits,
		Enfants:      n,
	}
}

// ordreDeSequence rend les enfants DANS L'ORDRE DE LA LISTE DE LECTURE, pas dans celui de la
// liste d'enfants — c'est la liste de lecture qui porte l'ordre des phases.
//
// La table de poids (`lirePoidsAleatoire`) valide deja que ces identifiants sont des enfants
// declares ; ici on garde en plus leur RANG, que la table jette parce qu'elle est indexee par
// identifiant. Rend nil si la liste n'est pas lisible : sans ordre, pas de sequence.
func ordreDeSequence(d []byte, connu func(uint32) bool) []uint32 {
	off, n := positionEnfants(d, connu)
	if off < 0 || n < 1 {
		return nil
	}
	enfants := map[uint32]bool{}
	for i := 0; i < n; i++ {
		enfants[binary.LittleEndian.Uint32(d[off+4+4*i:])] = true
	}
	p := off + 4 + 4*n
	if p+2 > len(d) {
		return nil
	}
	nb := int(binary.LittleEndian.Uint16(d[p:]))
	p += 2
	if nb < 1 || nb > 512 || p+8*nb > len(d) {
		return nil
	}
	out := make([]uint32, 0, nb)
	for i := 0; i < nb; i++ {
		id := binary.LittleEndian.Uint32(d[p+8*i:])
		if !enfants[id] {
			return nil
		}
		out = append(out, id)
	}
	return out
}
