package main

// hirc_noeuds.go — lecture COMPLETE d'un `NodeBaseParams` : bus, parent, et TOUTES les
// proprietes, decodees ou non.
//
// POURQUOI CE FICHIER (lot V3E, 2026-09-02). `proprietes.go` ne retient de chaque noeud que
// quatre proprietes (volume, hauteur, LPF, delai) et JETTE le reste, y compris les deux
// champs qui decident du gain reel d'une couche :
//
//	OverrideBusId  — le bus de sortie du noeud (son volume s'ajoute au chemin) ;
//	DirectParentID — le parent dans la hierarchie actor-mixer (idem, en remontant).
//
// `arbre.go` ne somme que le chemin DESCENDANT (evenement -> conteneurs -> Sound). Le gain
// qu'il publie est donc un gain RELATIF entre couches d'un meme evenement, pas le gain de
// chemin complet. Tant que toutes les couches ont le meme parent et le meme bus, les deux
// coincident ; des qu'elles n'ont pas le meme, le melange est faux. Ce fichier lit ce qu'il
// faut pour trancher, et RIEN N'EST JETE : une propriete dont l'identifiant n'est pas
// connu est rendue avec ses octets bruts (§ « propriete non decodee » des rapports).
//
// LAYOUT lu (Wwise 2019+, identique a `proprietes.go` mais complet) :
//
//	u8  bIsOverrideParentFX
//	u8  uNumFx
//	si uNumFx > 0 : u8 bitsFXBypass, puis uNumFx x 7 octets
//	u8  bOverrideAttachmentParams
//	u32 OverrideBusId
//	u32 DirectParentID
//	u8  byBitVector
//	AkPropBundle        : u8 n, n x u8 id, n x f32
//	AkPropBundle RANGED : u8 n, n x u8 id, n x (f32 min, f32 max)

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// nomsAkProp : identifiants de proprietes Wwise (`AkPropID`) pour les versions de bank
// 113 et suivantes. La table est DECLAREE, pas devinee au cas par cas : un identifiant
// absent d'ici est rendu « inconnu » avec ses octets bruts plutot qu'ignore en silence.
//
// ATTENTION — desaccord avec `proprietes.go`. Celui-ci lit l'identifiant 17 comme
// « InitialDelay » ; dans cette table 17 est `Probability` et le delai initial est 59.
// Le desaccord est CONSIGNE et tranche par la mesure : `hirc-event` imprime le releve des
// identifiants reellement presents (`-releve`), et le rapport V3E dit ce qui a ete vu.
var nomsAkProp = map[byte]string{
	0: "Volume", 1: "LFE", 2: "Pitch", 3: "LPF", 4: "HPF",
	5: "BusVolume", 6: "MakeUpGain", 7: "Priority", 8: "PriorityDistanceOffset",
	11: "MuteRatio", 12: "PAN_LR", 13: "PAN_FR", 14: "CenterPCT",
	15: "DelayTime", 16: "TransitionTime", 17: "Probability", 18: "DialogueMode",
	19: "UserAuxSendVolume0", 20: "UserAuxSendVolume1", 21: "UserAuxSendVolume2",
	22: "UserAuxSendVolume3", 23: "GameAuxSendVolume", 24: "OutputBusVolume",
	25: "OutputBusHPF", 26: "OutputBusLPF",
	27: "HDRBusThreshold", 28: "HDRBusRatio", 29: "HDRBusReleaseTime",
	30: "HDRBusGameParam", 31: "HDRBusGameParamMin", 32: "HDRBusGameParamMax",
	33: "HDRActiveRange", 34: "LoopStart", 35: "LoopEnd",
	36: "TrimInTime", 37: "TrimOutTime", 38: "FadeInTime", 39: "FadeOutTime",
	40: "FadeInCurve", 41: "FadeOutCurve", 42: "LoopCrossfadeDuration",
	43: "CrossfadeUpCurve", 44: "CrossfadeDownCurve",
	45: "MidiTrackingRootNote", 46: "MidiPlayOnNoteType", 47: "MidiTransposition",
	48: "MidiVelocityOffset", 49: "MidiKeyRangeMin", 50: "MidiKeyRangeMax",
	51: "MidiVelocityRangeMin", 52: "MidiVelocityRangeMax", 53: "MidiChannelMask",
	54: "PlaybackSpeed", 55: "MidiTempoSource", 56: "MidiTargetNode",
	57: "AttachedPluginFXID", 58: "Loop", 59: "InitialDelay",
	60: "UserAuxSendLPF0", 61: "UserAuxSendLPF1", 62: "UserAuxSendLPF2",
	63: "UserAuxSendLPF3", 64: "UserAuxSendHPF0", 65: "UserAuxSendHPF1",
	66: "UserAuxSendHPF2", 67: "UserAuxSendHPF3", 68: "GameAuxSendLPF",
	69: "GameAuxSendHPF", 70: "AttenuationID",
}

// nomAkProp rend le nom d'une propriete, ou « inconnu_<n> » si la table ne la porte pas.
func nomAkProp(id byte) string {
	if n, ok := nomsAkProp[id]; ok {
		return n
	}
	return fmt.Sprintf("inconnu_%d", id)
}

// propBrute : une propriete telle qu'elle est ECRITE, plus sa lecture en flottant. Les
// octets sont conserves pour que le rapport puisse citer une propriete non decodee.
type propBrute struct {
	ID     byte    `json:"id"`
	Nom    string  `json:"nom"`
	Valeur float32 `json:"valeur"`
	Octets string  `json:"octets"`
	// Bits : la valeur relue en entier non signe. Certaines proprietes NE SONT PAS des
	// flottants — `AttenuationID` (70) et `AttachedPluginFXID` (57) portent un identifiant
	// de 32 bits ; les lire en flottant donne des nombres absurdes (5,6e-30). Les deux
	// lectures sont publiees, la brute fait foi.
	Bits uint32 `json:"bits"`
	// Min/Max ne sont renseignes que pour le paquet RANGED (fourchette par lecture).
	Min    float32 `json:"min,omitempty"`
	Max    float32 `json:"max,omitempty"`
	Ranged bool    `json:"ranged,omitempty"`
}

// nodeBase : le `NodeBaseParams` complet d'un objet HIRC.
type nodeBase struct {
	Lu           bool        `json:"lu"`
	NumFx        int         `json:"num_fx"`
	BusOverride  uint32      `json:"bus_override"`
	ParentDirect uint32      `json:"parent_direct"`
	Props        []propBrute `json:"props,omitempty"`
	Ranged       []propBrute `json:"ranged,omitempty"`
	// Echec : pourquoi la lecture n'a pas abouti, quand `Lu` est faux. Un noeud non lu
	// n'est jamais compte comme « gain 0 » en silence.
	Echec string `json:"echec,omitempty"`
	// Fin : offset juste apres le paquet RANGED (debut de PositioningParams).
	Fin int `json:"-"`
}

// gain rend la valeur d'une propriete de gain, en dB, et si elle est presente.
func (n nodeBase) prop(id byte) (float32, bool) {
	for _, p := range n.Props {
		if p.ID == id {
			return p.Valeur, true
		}
	}
	return 0, false
}

// fourchetteDe rend la fourchette RANGED d'une propriete, si le noeud en declare une.
func (n nodeBase) fourchetteDe(id byte) (float32, float32, bool) {
	for _, p := range n.Ranged {
		if p.ID == id {
			return p.Min, p.Max, true
		}
	}
	return 0, 0, false
}

// gainPropre : le gain que le noeud ajoute a tout ce qui passe par lui — Volume + MakeUpGain.
//
// Les deux s'additionnent dans le moteur (le makeup gain est applique apres les effets du
// noeud, le volume avant : leur somme est le gain de voix du noeud). Les ignorer l'un ou
// l'autre est ce qui rendait les gains « approximatifs » avant ce lot.
func (n nodeBase) gainPropre() float64 {
	var g float64
	if v, ok := n.prop(propVolume); ok {
		g += float64(v)
	}
	if v, ok := n.prop(propMakeUpGain); ok {
		g += float64(v)
	}
	return g
}

// propMakeUpGain : `AkPropID_MakeUpGain`. Absent de `proprietes.go`, donc jamais somme
// avant ce lot.
const propMakeUpGain = 6

// propBusVolume : `AkPropID_BusVolume`, le volume propre d'un objet `Bus`.
const propBusVolume = 5

// gainDeBus : ce qu'un objet `Bus` ajoute a tout ce qui le traverse. Un bus porte son
// niveau dans `BusVolume` et non dans `Volume` — les deux sont sommes plutot que choisis,
// pour qu'un bus qui declarerait les deux ne perde ni l'un ni l'autre.
func (n nodeBase) gainDeBus() float64 {
	g := n.gainPropre()
	if v, ok := n.prop(propBusVolume); ok {
		g += float64(v)
	}
	return g
}

// propInitialDelay : `AkPropID_InitialDelay` selon la table 113+. C'est le DECALAGE de
// depart d'un noeud, celui que le mandat appelle « offset de couche ».
const propInitialDelay = 59

// decalageNodeBaseSound : longueur de `AkBankSourceData` devant le NodeBaseParams d'un
// Sound (pluginID 4 + streamType 1 + sourceID 4 + inMemoryMediaSize 4 + sourceBits 1).
const decalageNodeBaseSound = 14

// lireNodeBase decode un `NodeBaseParams` a l'offset donne.
//
// La lecture est STRUCTURELLE (bornes, nombre de proprietes, flottants finis) et non
// semantique : une propriete hors plage n'invalide pas le paquet, elle est rendue telle
// quelle. C'est le choix inverse de `lirePaquetLarge`, et il est deliberé : ce mode-ci sert
// a MESURER ce que portent les banques, pas a filtrer ce qu'on sait deja lire.
func lireNodeBase(d []byte, off int) nodeBase {
	var nb nodeBase
	if off+2 > len(d) {
		nb.Echec = "charge utile trop courte pour NodeInitialFxParams"
		return nb
	}
	nFx := int(d[off+1])
	if nFx > 8 {
		nb.Echec = fmt.Sprintf("uNumFx=%d invraisemblable", nFx)
		return nb
	}
	p := off + 2
	if nFx > 0 {
		p += 1 + nFx*7
	}
	if p+10 > len(d) {
		nb.Echec = "charge utile trop courte pour bus/parent"
		return nb
	}
	nb.NumFx = nFx
	nb.BusOverride = binary.LittleEndian.Uint32(d[p+1:])
	nb.ParentDirect = binary.LittleEndian.Uint32(d[p+5:])
	p += 10
	props, suite, err := lirePaquetHirc(d, p, 1)
	if err != "" {
		nb.Echec = "paquet de proprietes : " + err
		return nb
	}
	nb.Props = props
	ranged, fin, err2 := lirePaquetHirc(d, suite, 2)
	if err2 != "" {
		// Le paquet simple est lu, le RANGED non : on garde ce qui est acquis et on le DIT.
		nb.Lu = true
		nb.Fin = suite
		nb.Echec = "paquet RANGED : " + err2
		return nb
	}
	for i := range ranged {
		ranged[i].Ranged = true
	}
	nb.Ranged = ranged
	nb.Lu = true
	nb.Fin = fin
	return nb
}

// lirePaquetHirc lit un `AkPropBundle` de `largeur` flottants par propriete.
// Rend les proprietes, l'offset suivant, et une raison d'echec (vide = succes).
func lirePaquetHirc(d []byte, off, largeur int) ([]propBrute, int, string) {
	if off < 0 || off >= len(d) {
		return nil, off, "offset hors charge utile"
	}
	n := int(d[off])
	if n > 32 {
		return nil, off, fmt.Sprintf("cProps=%d invraisemblable", n)
	}
	debutIDs := off + 1
	debutVals := debutIDs + n
	fin := debutVals + n*4*largeur
	if fin > len(d) {
		return nil, off, fmt.Sprintf("paquet de %d propriete(s) deborde la charge utile", n)
	}
	out := make([]propBrute, 0, n)
	for i := 0; i < n; i++ {
		id := d[debutIDs+i]
		base := debutVals + i*largeur*4
		bits := make([]uint32, largeur)
		vals := make([]float32, largeur)
		for c := 0; c < largeur; c++ {
			bits[c] = binary.LittleEndian.Uint32(d[base+c*4:])
			vals[c] = math.Float32frombits(bits[c])
			if math.IsNaN(float64(vals[c])) || math.IsInf(float64(vals[c]), 0) {
				return nil, off, fmt.Sprintf("propriete %d : flottant non fini", id)
			}
		}
		pb := propBrute{ID: id, Nom: nomAkProp(id), Valeur: vals[0], Bits: bits[0], Octets: hexaLE(bits)}
		if largeur == 2 {
			pb.Min, pb.Max = min32(vals[0], vals[1]), max32(vals[0], vals[1])
		}
		out = append(out, pb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, fin, ""
}

// hexaLE rend les octets d'une suite de mots 32 bits, dans l'ordre du fichier.
func hexaLE(mots []uint32) string {
	var s string
	for i, m := range mots {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%02x %02x %02x %02x",
			byte(m), byte(m>>8), byte(m>>16), byte(m>>24))
	}
	return s
}

// lireBusHirc decode un objet `Bus` (type 8) ou `AuxBus` (19).
//
// LAYOUT (CAkBus) : u32 OverrideBusId ; si 0, u32 idDeviceShareset ; puis AkPropBundle.
// Les deux dispositions sont ESSAYEES et la premiere qui rend un paquet structurellement
// lisible est retenue — meme methode que le reste du parseur.
func lireBusHirc(d []byte) nodeBase {
	var nb nodeBase
	if len(d) < 4 {
		nb.Echec = "charge utile de bus trop courte"
		return nb
	}
	nb.ParentDirect = binary.LittleEndian.Uint32(d)
	offs := []int{4}
	if nb.ParentDirect == 0 {
		offs = []int{8, 4}
	}
	for _, p := range offs {
		props, suite, err := lirePaquetHirc(d, p, 1)
		if err != "" {
			continue
		}
		nb.Props = props
		nb.Lu = true
		nb.Fin = suite
		return nb
	}
	nb.Echec = "aucune disposition de CAkBus ne rend un paquet lisible"
	return nb
}

// wemsSous rend les `.wem` atteignables sous un noeud, tries. Sert a montrer ce que joue
// CHAQUE etat d'un `Switch` : un hachage d'etat ne se juge que par ses medias.
func (d *dumpeurV3E) wemsSous(n uint32) []uint32 {
	set := map[uint32]bool{}
	vus := map[uint32]bool{}
	var descendre func(uint32)
	descendre = func(id uint32) {
		if vus[id] {
			return
		}
		vus[id] = true
		if w, ok := d.bk.Sons[id]; ok {
			set[w] = true
		}
		for _, e := range d.bk.Enfants[id] {
			descendre(e)
		}
		if c, ok := d.bk.Switchs[id]; ok {
			for _, p := range c.Paquets {
				for _, e := range p.Enfants {
					descendre(e)
				}
			}
		}
	}
	descendre(n)
	out := make([]uint32, 0, len(set))
	for w := range set {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
