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
	// Variation : le paquet RANGED du meme noeud, c'est-a-dire de combien le moteur
	// deplace ces valeurs A CHAQUE LECTURE. C'est ce qui fait que deux coups consecutifs
	// ne sont pas le meme fichier joue deux fois.
	Variation fourchetteSon
}

// fourchette : l'ecart autorise autour de la valeur nominale, en OFFSETS SIGNES
// (unite de la propriete : dB pour le volume, centiemes de demi-ton pour la hauteur).
//
// Les deux composantes du paquet RANGED sont rendues ORDONNEES (`Bas <= Haut`) et non
// retranchees du nominal : le format ne dit pas lequel des deux champs est le minimum, et
// le chantier ne postule rien. `statsSignesVariation` compte les signes observes a
// l'execution pour que la premiere sortie reelle tranche l'interpretation sur pieces.
type fourchette struct {
	Bas  float32 `json:"bas"`
	Haut float32 `json:"haut"`
}

// Nulle dit si la fourchette n'autorise aucun ecart — le cas le plus frequent.
func (f fourchette) Nulle() bool { return f.Bas == 0 && f.Haut == 0 }

// fourchetteSon : les fourchettes RANGED retenues pour le rejeu (volume et hauteur).
// Le plan n'en demande pas d'autres : filtrage et delai ne sont pas rejoues in-app.
type fourchetteSon struct {
	VolumeDB fourchette `json:"volume_db"`
	PitchCts fourchette `json:"pitch_cents"`
	Lu       bool       `json:"-"`
}

// Nulle dit si le noeud ne fait varier ni le volume ni la hauteur.
func (f fourchetteSon) Nulle() bool { return f.VolumeDB.Nulle() && f.PitchCts.Nulle() }

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

// lireProprietes decode les proprietes d'un objet `Sound` (NodeBaseParams a +14,
// apres la source audio).
func lireProprietes(d []byte) proprietesSon {
	return lireProprietesA(d, 14)
}

// lireProprietesConteneur decode les proprietes d'un CONTENEUR : sa charge utile COMMENCE
// par NodeBaseParams, il n'y a pas de source audio devant.
func lireProprietesConteneur(d []byte) proprietesSon {
	return lireProprietesA(d, 0)
}

// lireProprietesA decode un NodeBaseParams a l'offset donne. Meme validation par
// plausibilite dans les deux cas : un layout qui a derive echoue au lieu de mentir.
func lireProprietesA(d []byte, off int) proprietesSon {
	var out proprietesSon
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
	// Le paquet RANGED suit : c'est lui qui porte l'ecart applique a chaque lecture.
	// Illisible, il signale que le layout a derive — on garde alors la lecture simple
	// sans pretendre connaitre la variation.
	out.Variation = lireVariation(d, suite)
	return out
}

// lireVariation decode le paquet RANGED a l'offset donne.
//
// Chaque propriete y occupe DEUX float32 (contre un seul dans le paquet simple). Le
// lecteur historique ne prenait que la premiere composante : appele en largeur 2, il
// lisait donc une borne et jetait l'autre.
func lireVariation(d []byte, off int) fourchetteSon {
	var out fourchetteSon
	vals, _, ok := lirePaquetLarge(d, off, 2)
	if !ok {
		return out
	}
	out.Lu = true
	for id, v := range vals {
		f := fourchette{Bas: min32(v[0], v[1]), Haut: max32(v[0], v[1])}
		noterSignesVariation(v[0], v[1])
		switch id {
		case propVolume:
			out.VolumeDB = f
		case propPitch:
			out.PitchCts = f
		}
	}
	return out
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// lirePaquetProps lit un AkPropBundle de `largeur` float32 par propriete.
// Rend les valeurs (premiere composante), l'offset suivant, et la plausibilite.
func lirePaquetProps(d []byte, off, largeur int) (map[byte]float32, int, bool) {
	larges, fin, ok := lirePaquetLarge(d, off, largeur)
	if !ok {
		return nil, off, false
	}
	out := make(map[byte]float32, len(larges))
	for id, v := range larges {
		out[id] = v[0]
	}
	return out, fin, true
}

// lirePaquetLarge lit un AkPropBundle et rend TOUTES les composantes de chaque propriete.
// Rend les valeurs, l'offset suivant, et la plausibilite.
func lirePaquetLarge(d []byte, off, largeur int) (map[byte][]float32, int, bool) {
	if off < 0 || off >= len(d) || largeur < 1 {
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
	out := make(map[byte][]float32, n)
	for i := 0; i < n; i++ {
		id := d[debutIDs+i]
		composantes := make([]float32, largeur)
		for c := 0; c < largeur; c++ {
			v := math.Float32frombits(binary.LittleEndian.Uint32(d[debutVals+(i*largeur+c)*4:]))
			if !plausibleProp(id, v) {
				return nil, off, false
			}
			composantes[c] = v
		}
		out[id] = composantes
	}
	return out, fin, true
}
