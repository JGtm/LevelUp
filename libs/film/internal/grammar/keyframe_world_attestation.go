package grammar

// keyframe_world_attestation.go — L'ATTESTATION PAR ENTREES LIBRES : la preuve d'arrivee exacte d'une
// ancre d'image-cle.
//
// La table d'image-cle compte UNE entree par slot (`keyframe_entrees.go`) ; un slot libere y est une
// entree de 108 bits dont l'identifiant, l'archetype et le mot +0xc valent `0xffffffff`
// ([estUnSlotLibere]). Entre un record de slot p et le record SUIVANT de slot s, la table porte donc
// EXACTEMENT s-p-1 entrees libres, contigues, juste devant le second. Un candidat de slot s qui a ce
// compte devant lui est attendu a cette position par la table ; une fausse ancre prise dans le corps
// d'un record n'a pas, devant elle, les s-p-1 entrees que son slot exige (le corps d'un record n'est
// pas une suite d'entrees libres de 108 bits alignees sur elle).
//
// La regle pose une clef de tete a l'election ([kfCand.betterThan]) : un candidat attesté bat tout
// candidat non attesté, quel que soit son slot ou sa generation. Un ecart de slot nul (s == p+1) ne
// prouve rien : zero entree libre est vrai de toute position, ce cas reste juge par le voisin
// immediat et l'election d'avant.

// keyframeEntreeLibreBits est la taille, en bits, d'une entree sans archetype de la table
// d'image-cle : l'en-tete de 108 bits de l'ecrivain (`profile.KeyframeEnTeteBits`), sans corps.
const keyframeEntreeLibreBits = 108

// entreesLibresAvant compte les entrees libres contigues qui precedent immediatement le bit `q`, en
// remontant au plus jusqu'a `plafond` entrees et sans passer sous le bit 0. Le compte est
// exact : il s'arrete a la premiere entree qui n'est pas libre.
func entreesLibresAvant(pay []byte, q, plafond int) int {
	n := 0
	for n < plafond {
		p := q - (n+1)*keyframeEntreeLibreBits
		if p < 0 || !estUnSlotLibere(pay, p, len(pay)*8, keyframeEntreeLibreBits) {
			break
		}
		n++
	}
	return n
}

// estAttestee dit si le candidat de `bit` et de `slot`, cherche apres un record de slot `prevSlot`,
// a devant lui EXACTEMENT slot-prevSlot-1 entrees libres (au moins une : cf. l'en-tete).
func (r *kfRecherche) estAttestee(bit, slot, prevSlot int) bool {
	ecart := slot - prevSlot - 1
	if ecart < 1 {
		return false
	}
	return entreesLibresAvant(r.buf, bit, ecart+1) == ecart
}
