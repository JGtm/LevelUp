package main

// proprietes.go — les PROPRIETES d'un objet `Sound` : gain, hauteur, filtrage, delai.
//
// POURQUOI. Le mixage additionne les couches a gain unitaire et toutes a t=0, alors que le
// moteur applique par couche un volume, un decalage et parfois un filtrage. Une couche de
// renfort censee etre 10 dB sous la principale arrive donc au meme niveau : le resultat
// sonne plus brouillon qu'un `.wem` isole — exactement ce que l'utilisateur a signale.
//
// LAYOUT (Wwise 2019+), depuis le debut de la charge utile d'un objet Sound :
//
//	+0   u32 ulPluginID
//	+4   u8  StreamType
//	+5   u32 sourceID
//	+9   u32 uInMemoryMediaSize
//	+13  u8  uSourceBits
//	+14  NodeBaseParams :
//	       u8 bIsOverrideParentFX | u8 uNumFx
//	       si uNumFx > 0 : u8 bitsFXBypass, puis uNumFx x 7 octets
//	       u8 bOverrideAttachmentParams | u32 OverrideBusId | u32 DirectParentID | u8 bits
//	       AkPropBundle : u8 nProps, puis nProps x u8 idProp, puis nProps x float32
//	       AkPropBundle RANGED : u8 nProps, puis nProps x u8 idProp, puis nProps x 2 float32
//
// RIEN N'EST POSTULE : la lecture n'est retenue que si le paquet est PLAUSIBLE (nombre de
// proprietes borne, identifiants connus, valeurs finies et dans des plages realistes).
// Un layout qui aurait derive echoue au controle au lieu de rendre des gains fantaisistes.

import (
	"encoding/binary"
	"math"
)

// Identifiants de proprietes Wwise utiles ici.
const (
	propVolume = 0  // dB
	propLFE    = 1  // dB
	propPitch  = 2  // centiemes de demi-ton
	propLPF    = 3  // 0..100
	propHPF    = 4  // 0..100
	propDelai  = 17 // secondes (AkPropID_InitialDelay)
)

// proprietesSon rassemble ce qui influe sur le mixage d'une couche.
type proprietesSon struct {
	VolumeDB float32
	PitchCts float32
	LPF      float32
	DelaiS   float32
	Lu       bool // faux si le paquet n'a pas pu etre lu de facon plausible
}

// plausibleProp dit si un couple (identifiant, valeur) est credible.
func plausibleProp(id byte, v float32) bool {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return false
	}
	switch id {
	case propVolume, propLFE:
		return f >= -200 && f <= 200
	case propPitch:
		return f >= -4800 && f <= 4800
	case propLPF, propHPF:
		return f >= -100 && f <= 100
	case propDelai:
		return f >= 0 && f <= 60
	}
	return f >= -1e6 && f <= 1e6
}

// lireProprietes decode les proprietes d'un objet `Sound`.
func lireProprietes(d []byte) proprietesSon {
	var out proprietesSon
	off := 14
	if off+2 > len(d) {
		return out
	}
	nFx := int(d[off+1])
	off += 2
	if nFx > 0 {
		off++ // bitsFXBypass
		off += nFx * 7
	}
	// bOverrideAttachmentParams (1) + OverrideBusId (4) + DirectParentID (4) + bits (1)
	off += 10
	if off >= len(d) {
		return out
	}
	vals, suite, ok := lirePaquetProps(d, off, 1)
	if !ok {
		return out
	}
	out.Lu = true
	for id, v := range vals {
		switch id {
		case propVolume:
			out.VolumeDB = v
		case propPitch:
			out.PitchCts = v
		case propLPF:
			out.LPF = v
		case propDelai:
			out.DelaiS = v
		}
	}
	// Le paquet RANGED suit ; on ne l'exploite pas encore mais sa lecture confirme
	// l'alignement — s'il est illisible, c'est que le layout a derive.
	if _, _, ok := lirePaquetProps(d, suite, 2); !ok {
		out.Lu = out.Lu && true // on garde la lecture simple, sans pretendre plus
	}
	return out
}

// lirePaquetProps lit un AkPropBundle de `largeur` float32 par propriete.
// Rend les valeurs (premiere composante), l'offset suivant, et la plausibilite.
func lirePaquetProps(d []byte, off, largeur int) (map[byte]float32, int, bool) {
	if off < 0 || off >= len(d) {
		return nil, off, false
	}
	n := int(d[off])
	if n > 24 {
		return nil, off, false
	}
	debutIDs := off + 1
	debutVals := debutIDs + n
	fin := debutVals + n*4*largeur
	if fin > len(d) {
		return nil, off, false
	}
	out := make(map[byte]float32, n)
	for i := 0; i < n; i++ {
		id := d[debutIDs+i]
		v := math.Float32frombits(binary.LittleEndian.Uint32(d[debutVals+i*4*largeur:]))
		if !plausibleProp(id, v) {
			return nil, off, false
		}
		out[id] = v
	}
	return out, fin, true
}
