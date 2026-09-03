package filmdec

// slot_set.go — LA BANDE DE SLOTS BIPEDE, EN TABLEAU INDEXE.
//
// POURQUOI. Un slot bipede tient sur [bipedSlotBits] bits : le domaine entier vaut 8 192
// valeurs, et une bande en compte une vingtaine. Les detecteurs d'en-tete
// (`matchBipedHeaderRaw` et ses repliques : rangs de capacite, camouflage, grappin,
// changements d'arme, decoupage i0, ramassages) interrogent cette bande UNE FOIS PAR BIT
// CANDIDAT du payload, soit des dizaines de millions de fois par film. En `map[uint32]bool`
// c'etait un hachage complet par candidat (`runtime.mapaccess1_fast32` pese 1,5 % du profil
// CPU du 2026-09-02, dont 40 % pour ce seul detecteur) ; en tableau de booleens c'est une
// indexation.
//
// POURQUOI UN STRUCT ET PAS UN `[]bool` NU. Parce que `len()` d'une bande veut dire « combien
// de slots » partout dans le depot (`if len(slots) == 0`), et que `len()` d'un tableau dense
// vaudrait 8 192. Le struct fait ECHOUER LA COMPILATION sur chaque usage de forme « map »
// au lieu de le laisser mentir : c'est le compilateur qui a trouve les sites a convertir,
// pas une relecture.

// SlotBand est une bande de slots dense : un booleen par slot possible.
type SlotBand struct {
	dense []bool
	n     int // nombre de slots presents — la grandeur que `len()` donnait sur la map
}

// slotBandDomain est la taille minimale du tableau : le domaine complet d'un slot de 13 bits.
// Une bande dont un slot depasserait ce domaine (les bandes d'objets du monde lisent leur
// slot dans le registre, pas dans un champ de 13 bits) fait grandir le tableau d'autant.
const slotBandDomain = 1 << bipedSlotBits

// NewSlotBand convertit un ensemble de slots en bande dense. C'est le SEUL point de passage :
// il se paie une fois par balayage, jamais dans une boucle de candidats. EXPORTE parce que
// [ScanBipedRecords] prend une bande et que ses appelants hors paquet relevent la leur.
func NewSlotBand(m map[uint32]bool) SlotBand {
	size := slotBandDomain
	for s, ok := range m {
		if ok && int(s) >= size {
			size = int(s) + 1
		}
	}
	out := SlotBand{dense: make([]bool, size)}
	for s, ok := range m {
		if ok && !out.dense[s] {
			out.dense[s] = true
			out.n++
		}
	}
	return out
}

// Has dit si le slot appartient a la bande.
func (b SlotBand) Has(slot uint32) bool { return int(slot) < len(b.dense) && b.dense[slot] }

// Count rend le nombre de slots de la bande — l'equivalent du `len()` de la map d'avant.
func (b SlotBand) Count() int { return b.n }

// Slots rend les slots de la bande, EN ORDRE CROISSANT — l'equivalent du parcours de la map
// d'avant, en deterministe.
func (b SlotBand) Slots() []uint32 {
	out := make([]uint32, 0, b.n)
	for s, in := range b.dense {
		if in {
			out = append(out, uint32(s))
		}
	}
	return out
}
