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

import (
	"encoding/binary"
	"math"
)

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

// ---------------- Blend : table complete des couches et de leurs courbes ----------------
//
// STRUCTURE ETABLIE SUR LES OCTETS (vidage hexadecimal de l'etape 12, verifie champ par
// champ sur deux conteneurs distincts), apres la liste d'enfants :
//
//	u32 nombre de couches
//	  par couche :
//	    u32 ulLayerID
//	    u16 nombre de RTPC ; par RTPC :
//	        u32 RTPCID | u8 type | u8 accum | u8 ParamID | u32 curveID | u8 scaling
//	        u16 nombre de points | points x { f32 x | f32 y | u32 interp }
//	    u32 rtpcID du fondu | u8 type
//	    u32 nombre d'associations ; par association :
//	        u32 idEnfant | u32 nombre de points | points x 12 octets
//
// CE QUE LE BLEND JOUE (regle ecrite avant le code, cf. etape 18 du plan) :
//   - sans couche declaree : TOUS ses enfants, simultanement ;
//   - avec couches : chaque enfant associe est module par sa courbe de fondu selon un
//     parametre de jeu (la distance, typiquement). Le rejeu 2D ne gere pas ce parametre
//     (decision utilisateur) : on evalue chaque courbe A SON POINT LE PLUS A GAUCHE —
//     valeur minimale du parametre, donc la plus proche. Un enfant a gain lineaire nul y
//     est INAUDIBLE et ne joue pas ; un gain entre 0 et 1 devient un dB porte par l'enfant.
//   - un enfant sans association joue tel quel ;
//   - les RTPC DE COUCHE (volume par etat de jeu) sont IGNORES AVEC RAISON : leur valeur
//     neutre n'est pas dans la bank (0 chunk STMG reel sur 1305), l'evaluer a un x
//     arbitraire injecterait une constante fausse.

type pointCourbe struct {
	X, Y   float32
	Interp uint32
}

type courbe []pointCourbe

type assocBlend struct {
	Enfant uint32
	C      courbe // vide = l'enfant joue tel quel
}

type coucheBlend struct {
	ID     uint32
	RTPC   uint32 // parametre de jeu du fondu (0 = aucun)
	Assocs []assocBlend
}

type conteneurBlend struct {
	Lu   bool
	Cies []coucheBlend
}

// Audibles rend, pour les enfants du conteneur, le gain additionnel en dB au point de
// reference — ou l'ABSENCE de l'enfant s'il y est inaudible. Les enfants non associes
// jouent a 0 dB.
func (c conteneurBlend) Audibles(enfants []uint32) map[uint32]float64 {
	out := make(map[uint32]float64, len(enfants))
	for _, e := range enfants {
		out[e] = 0
	}
	for _, l := range c.Cies {
		for _, a := range l.Assocs {
			if len(a.C) == 0 {
				continue // association sans courbe : l'enfant joue tel quel
			}
			y := float64(a.C[0].Y) // valeur au point le plus a gauche
			if y <= 0 {
				delete(out, a.Enfant)
				continue
			}
			if y < 1 {
				out[a.Enfant] += 20 * math.Log10(y)
			}
		}
	}
	return out
}

// PiloteParRTPC dit si AU MOINS une couche declare une association a courbe : dans ce cas
// « le Blend joue toutes ses couches » est faux, la courbe tranche.
func (c conteneurBlend) PiloteParRTPC() bool {
	for _, l := range c.Cies {
		for _, a := range l.Assocs {
			if len(a.C) > 0 {
				return true
			}
		}
	}
	return false
}

// lireCourbe lit n points de 12 octets, VALIDES : x monotone croissant, valeurs finies et
// bornees, interpolation dans la nomenclature. Un seul point hors bornes rejette tout.
func lireCourbe(d []byte, off, n int) (courbe, int, bool) {
	if n < 0 || n > 255 || off+12*n > len(d) {
		return nil, off, false
	}
	out := make(courbe, 0, n)
	px := float32(math.Inf(-1))
	for i := 0; i < n; i++ {
		x := math.Float32frombits(binary.LittleEndian.Uint32(d[off+12*i:]))
		y := math.Float32frombits(binary.LittleEndian.Uint32(d[off+12*i+4:]))
		it := binary.LittleEndian.Uint32(d[off+12*i+8:])
		fx, fy := float64(x), float64(y)
		if math.IsNaN(fx) || math.IsInf(fx, 0) || math.IsNaN(fy) || math.IsInf(fy, 0) ||
			it > 0x20 || x < px || fy < -1e4 || fy > 1e4 {
			return nil, off, false
		}
		px = x
		out = append(out, pointCourbe{x, y, it})
	}
	return out, off + 12*n, true
}

// sauterRTPCListe traverse la liste des RTPC d'une couche en la validant, sans la garder
// (ignoree avec raison, cf. en-tete de section).
func sauterRTPCListe(d []byte, off int) (int, bool) {
	if off+2 > len(d) {
		return off, false
	}
	n := int(binary.LittleEndian.Uint16(d[off:]))
	off += 2
	if n > 64 {
		return off, false
	}
	for i := 0; i < n; i++ {
		// u32 RTPCID | u8 type | u8 accum | u8 ParamID | u32 curveID | u8 scaling | u16 n
		if off+14 > len(d) {
			return off, false
		}
		nb := int(binary.LittleEndian.Uint16(d[off+12:]))
		_, suite, ok := lireCourbe(d, off+14, nb)
		if !ok {
			return off, false
		}
		off = suite
	}
	return off, true
}

// lireBlend lit la table COMPLETE des couches d'un `Blend` (structure en tete de section).
// Chaque lecture est validee ; un seul champ hors bornes rejette le conteneur entier,
// compte par l'audit — jamais approxime.
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
	// SANS COUCHE DECLAREE, un `Blend` joue ses enfants tels quels.
	if nb == 0 {
		return conteneurBlend{Lu: true}
	}
	out := conteneurBlend{Lu: true}
	for i := 0; i < nb; i++ {
		if p+4 > len(d) {
			return conteneurBlend{}
		}
		l := coucheBlend{ID: binary.LittleEndian.Uint32(d[p:])}
		p += 4
		suite, ok := sauterRTPCListe(d, p)
		if !ok {
			return conteneurBlend{}
		}
		p = suite
		// u32 rtpcID du fondu | u8 type | u32 nombre d'associations
		if p+9 > len(d) {
			return conteneurBlend{}
		}
		l.RTPC = binary.LittleEndian.Uint32(d[p:])
		nAssoc := int(binary.LittleEndian.Uint32(d[p+5:]))
		p += 9
		if nAssoc < 0 || nAssoc > 512 {
			return conteneurBlend{}
		}
		for j := 0; j < nAssoc; j++ {
			if p+8 > len(d) {
				return conteneurBlend{}
			}
			enfant := binary.LittleEndian.Uint32(d[p:])
			nPts := int(binary.LittleEndian.Uint32(d[p+4:]))
			c, suite, ok := lireCourbe(d, p+8, nPts)
			if !ok {
				return conteneurBlend{}
			}
			// La courbe de fondu vise un enfant du conteneur : un identifiant inconnu
			// signe un mauvais offset, on rejette tout plutot que d'approximer.
			if !enfants[enfant] {
				return conteneurBlend{}
			}
			l.Assocs = append(l.Assocs, assocBlend{Enfant: enfant, C: c})
			p = suite
		}
		out.Cies = append(out.Cies, l)
	}
	return out
}
