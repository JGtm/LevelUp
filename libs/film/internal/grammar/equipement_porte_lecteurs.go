package grammar

// equipement_porte_lecteurs.go — LES VALEURS DE QUATRE COMPOSANTS DE ti=37 QUE LA COUCHE DE CAPTURE
// REND (lot « equipement porte » du plan d ajustements du 2026-10-14) : l energie (i24), l activation
// (i21), le signal de controle (i22), la pile des poignees suivies (i28).
//
// UN SEUL LECTEUR PAR COMPOSANT : le dispatch (`consumeByName`) et la capture
// (`consumeByNameCapturing`) appellent la MEME fonction `decodeXxx` ci-dessous ;
// `TestCaptureConsumesSameBitsAsDispatch` le garde. Les crochets de [Observation.EquipmentStateHook]
// restent publies par ces lecteurs, comme avant.

import "github.com/JGtm/LevelUp/libs/film/types"

// EquipementLu est la valeur capturee d un composant d equipement de ti=37.
type EquipementLu struct {
	// Composant : l etiquette de registre du composant lu.
	Composant string
	// Porte : la porte de tete du composant s est OUVERTE sur une poignee — pour i21, la branche
	// `FUN_1408f0ac4(…,4)` (porte a 1) ; faux pour les composants sans porte.
	Porte bool
	// Valeur : R(14) pour i24 ; R(3) pour i21 porte fermee ; R(4) pour i22 ; n pour i28.
	Valeur uint64
	// Suivis : pour i28, les vies que designent ses entrees PRESENTES, dans l ordre de la pile.
	Suivis []types.LifeKey
}

// EquipementOf rend la valeur d equipement capturee, si c en est une.
func (c CompResult) EquipementOf() (EquipementLu, bool) {
	v, ok := c.Payload.(EquipementLu)
	return v, ok
}

// Les etiquettes de registre de i22 et i28.
const (
	compEquipmentControlSignal = "equipment-control-signal-component"
	compEquipmentTrackedStack  = "equipment-tracked-object-handles-stack-component"
)

// decodeEquipmentEnergy porte FUN_141087bec (ti=37 i24) : R(14), sans porte, range en objet+0x594
// (l ecrivain `FUN_142ee0594`, bit 0x18, le tire de l energie objet+0x510 bornee a [0, 1]).
func decodeEquipmentEnergy(br *Lecteur) EquipementLu {
	v := br.ReadBits(14)
	br.obs.publishEquipment(EquipEnergy, v, true)
	return EquipementLu{Composant: compEquipmentEnergy, Valeur: v}
}

// decodeEquipmentActivated porte le lecteur de ti=37 i21 : R(1) porte de polarite INVERSEE — a 0,
// R(3) ; a 1, le corps FUN_1408f0ac4(…,4) (pas de sonde, 9 + 2).
func decodeEquipmentActivated(br *Lecteur) EquipementLu {
	if !br.ReadBit() {
		v := br.ReadBits(3)
		br.obs.publishEquipment(EquipActivated, v, true)
		return EquipementLu{Composant: compEquipmentActivated, Valeur: v}
	}
	consume1408f0ac4(br, 4) // FUN_1408f0ac4(...,4) @140c1dcbb : PAS de sonde, 9 + 2
	br.obs.publishEquipment(EquipActivated, 0, false)
	return EquipementLu{Composant: compEquipmentActivated, Porte: true}
}

// decodeEquipmentControlSignal porte FUN_14101cd94 (ti=37 i22) : `FUN_14101d200` (R(4), range en
// objet+0x5a6) puis `FUN_1408f0ac4(dst+0x58c, br, 4)` — categorie 4 : pas de sonde, 9 bits de valeur.
// L ecrivain (`FUN_142ee0594`, bit 0x16) recopie objet+0x328 quand il change.
func decodeEquipmentControlSignal(br *Lecteur) EquipementLu {
	v := br.ReadBits(4)
	consume1408f0ac4(br, 4)
	return EquipementLu{Composant: compEquipmentControlSignal, Valeur: v}
}

// decodeEquipmentTrackedStack porte FUN_140f72dec (ti=37 i28) : R(4) n (`FUN_1424e1d48`), range en
// objet+0x544, puis n entrees FUN_1408f0ac4(…,0), chacune R(1) presence [si 1 : 13 + 2 bits, sans
// sonde]. Le pointeur avance de 8 octets AVANT de comparer a `+0x544 + (n+1)*8` : la boucle fait n
// tours, jamais n+1 (une trame qui porte ce composant ne ferme qu a cette condition).
//
// UNE ENTREE PRESENTE EST UN HANDLE : `FUN_1406d3140` categorie 0, dont la base est 0x200
// (`FUN_140d10bb0` pose 0x200 pour les categories 0 et 1). Elle designe la vie (slot, generation)
// d un objet que l equipement a engendre et suit.
func decodeEquipmentTrackedStack(br *Lecteur) EquipementLu {
	n := br.ReadBits(4)
	var suivis []types.LifeKey
	for i := uint64(0); i < n; i++ {
		if br.ReadBit() {
			h := br.readQuantStat(0) // FUN_1408f0ac4(...,0) @140f72e41 : PAS de sonde, 13 + 2
			suivis = append(suivis, types.LifeKey{Slot: h&0x3fffffff + trackedHandleBase, Gen: h >> 30})
		}
	}
	return EquipementLu{Composant: compEquipmentTrackedStack, Valeur: n, Suivis: suivis}
}

// trackedHandleBase est la base de la categorie 0 de `FUN_1406d3140` (`FUN_140d10bb0`).
const trackedHandleBase = 0x200

// Les bornes de la quantification de l energie (i24) : `FUN_141087bec` appelle `FUN_1406d84b4` avec
// min 0 (litteral), max `DAT_143cd8374` (1,0), largeur 0xe, sans exclusion, extremites exactes.
const (
	energieMin     float32 = 0
	energieMax     float32 = 1
	energieLargeur uint    = 14
)

// EnergieDEquipement rend la valeur de l energie (i24) d un quantum lu ([types.CarriedEquipmentRead.Energy]).
func EnergieDEquipement(q uint32) float32 {
	return DequantEndpoint(uint64(q), energieMin, energieMax, energieLargeur, false, true)
}
