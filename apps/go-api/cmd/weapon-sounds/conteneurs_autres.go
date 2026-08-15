package main

// conteneurs_autres.go — les deux autres conteneurs a statuer avant de corriger quoi que
// ce soit, conformement au gate 12a du plan : LU / IGNORE AVEC RAISON / A LIRE.
//
// L'inventaire montre qu'il reste, APRES la liste d'enfants, des octets jamais lus :
// 40 en moyenne sur un `RandomSequence` (max 546) et 56 sur un `Blend` (max 373). Deux
// questions en decoulent, et chacune change ou non le rendu :
//
//   - `RandomSequence` : la table de lecture porte un POIDS par enfant. Sans elle, les
//     variantes sont tirees uniformement alors que le jeu en favorise certaines.
//   - `Blend` : ses couches peuvent etre pilotees par un RTPC (fondu enchaine selon une
//     valeur de jeu, typiquement la distance). Si c'est le cas, « le Blend joue TOUTES ses
//     couches » est faux, et le rendu empile des couches que le moteur attenue.
//
// Ce fichier MESURE ces deux points. Il ne les corrige pas : le plan veut l'inventaire
// complet avant le correctif.

import "encoding/binary"

// poidsAleatoire : la table de lecture d'un `RandomSequence`, quand elle est lisible.
//
//	u16 nombre | pour chacun : u32 idEnfant | i32 poids
//
// Poids Wwise : 50000 = poids par defaut. Valide en exigeant que tous les identifiants
// cites appartiennent a la liste d'enfants deja resolue.
type poidsAleatoire struct {
	Lu    bool
	Poids map[uint32]int32
}

// Uniforme dit si tous les poids sont egaux : dans ce cas ne pas lire la table ne change
// rien au rendu, et l'ignorer devient un choix justifie plutot qu'un oubli.
func (p poidsAleatoire) Uniforme() bool {
	var ref int32
	premier := true
	for _, v := range p.Poids {
		if premier {
			ref, premier = v, false
			continue
		}
		if v != ref {
			return false
		}
	}
	return true
}

func lirePoidsAleatoire(d []byte, connu func(uint32) bool) poidsAleatoire {
	off, n := positionEnfants(d, connu)
	if off < 0 {
		return poidsAleatoire{}
	}
	enfants := map[uint32]bool{}
	for i := 0; i < n; i++ {
		enfants[binary.LittleEndian.Uint32(d[off+4+4*i:])] = true
	}
	p := off + 4 + 4*n
	if p+2 > len(d) {
		return poidsAleatoire{}
	}
	nb := int(binary.LittleEndian.Uint16(d[p:]))
	p += 2
	if nb < 1 || nb > 512 || p+8*nb > len(d) {
		return poidsAleatoire{}
	}
	out := make(map[uint32]int32, nb)
	for i := 0; i < nb; i++ {
		id := binary.LittleEndian.Uint32(d[p+8*i:])
		if !enfants[id] {
			return poidsAleatoire{}
		}
		out[id] = int32(binary.LittleEndian.Uint32(d[p+8*i+4:]))
	}
	return poidsAleatoire{Lu: true, Poids: out}
}

// coucheBlend : une couche d'un `Blend`, avec le parametre de jeu qui la pilote.
type coucheBlend struct {
	ID      uint32
	RTPC    uint32 // 0 = aucune automation : la couche joue telle quelle
	NbAssoc int
}

type conteneurBlend struct {
	Lu   bool
	Cies []coucheBlend
}

// PiloteParRTPC dit si AU MOINS une couche declare une automation. Si aucune ne le fait —
// y compris quand le conteneur ne declare aucune couche — « le Blend joue toutes ses
// couches » est exact et le rendu actuel est correct pour lui.
func (c conteneurBlend) PiloteParRTPC() bool {
	for _, l := range c.Cies {
		if l.RTPC != 0 {
			return true
		}
	}
	return false
}

// lireBlend lit la table des couches qui suit la liste d'enfants d'un `Blend`.
//
//	u32 nombre de couches
//	  pour chacune : u32 ulLayerID | u16 nombre de RTPC | les RTPC | ...
//
// Le nombre de couches vaut souvent ZERO : le conteneur joue alors ses enfants tels quels.
// L'identifiant de couche est PROPRE a la couche — le valider contre les enfants, comme le
// faisait la premiere version, faisait echouer le lecteur sur les 303 conteneurs.
func lireBlend(d []byte, connu func(uint32) bool) conteneurBlend {
	off, n := positionEnfants(d, connu)
	if off < 0 {
		return conteneurBlend{}
	}
	enfants := map[uint32]bool{}
	for i := 0; i < n; i++ {
		enfants[binary.LittleEndian.Uint32(d[off+4+4*i:])] = true
	}
	p := off + 4 + 4*n
	if p+4 > len(d) {
		return conteneurBlend{}
	}
	nb := int(binary.LittleEndian.Uint32(d[p:]))
	p += 4
	if nb < 0 || nb > 64 {
		return conteneurBlend{}
	}
	// SANS COUCHE DECLAREE, un `Blend` joue ses enfants tels quels : c'est le cas simple,
	// et c'est celui que le rendu actuel suppose partout.
	if nb == 0 {
		return conteneurBlend{Lu: true}
	}
	// AVEC couches, la premiere porte son identifiant PROPRE — pas celui d'un enfant, et
	// c'est l'erreur qui faisait echouer ce lecteur sur les 303 conteneurs. Vient ensuite
	// une liste de RTPC (`u16` de tete) : son effectif suffit a repondre a la seule
	// question posee ici, y a-t-il une automation ? Le detail des courbes n'est pas lu.
	if p+6 > len(d) {
		return conteneurBlend{}
	}
	nRTPC := int(binary.LittleEndian.Uint16(d[p+4:]))
	if nRTPC < 0 || nRTPC > 64 {
		return conteneurBlend{}
	}
	rtpc := uint32(0)
	if nRTPC > 0 {
		if p+10 > len(d) {
			return conteneurBlend{}
		}
		rtpc = binary.LittleEndian.Uint32(d[p+6:])
	}
	return conteneurBlend{Lu: true, Cies: []coucheBlend{{
		ID: binary.LittleEndian.Uint32(d[p:]), RTPC: rtpc, NbAssoc: nRTPC,
	}}}
}
