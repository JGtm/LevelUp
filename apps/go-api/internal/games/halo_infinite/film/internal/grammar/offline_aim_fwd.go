package grammar

// offline_aim_fwd.go — LA CAPTURE D'i2 `object-forward-and-up-DYNAMIC-PRECISION` PAR LE
// BALAYAGE OFFLINE (extrait de offline_aim.go le 2026-09-20, lot 5.2b.2 : le fichier atteignait
// le seuil de 500 lignes du dépôt). Même fichier LOGIQUE, autre fichier PHYSIQUE.
//
// CE QUE CE COMPOSANT DONNE, ET CE QU'IL NE DONNE PAS — mesure du lot 5.2b.2, deux films.
//
// Il porte QUATRE chemins (cf. components_dynprec_orientation.go) et deux seulement sont
// empruntés sur `ti=40` : le mode 0 (direction cubemap de 19 bits) et le mode 1 (direction
// cubemap de 30 bits). Le mode 2 (« keep », deux vec3 float32 bruts) n'est JAMAIS emprunté :
// 0 record sur 144 385 (`4f77afc1`) et 0 sur 105 392 (`a349fea8`).
//
// LA DIRECTION QU'ILS PORTENT N'EST PAS L'AVANT DU CHÂSSIS. Sur `4f77afc1` elle est QUASI
// VERTICALE — |z| médian 0,960 (19 bits) et 0,981 (30 bits), 77 % et 94 % au-dessus de 0,9 :
// c'est le vecteur HAUT que le composant nomme, et son azimut au sol n'est que la direction de
// la pente. L'écart au déplacement est alors indiscernable du témoin par permutation
// (104,5° contre 88,7°). Le détail et le tableau des deux films sont au plan, item 5.2b.2.
//
// LES CHAMPS SONT CAPTURÉS QUAND MÊME, et c'est délibéré : un négatif qui ne se rejoue pas
// n'est pas une mesure. `vehicule_orientation_research_test.go` les lit, et lui seul.

// fwdDir30Bits : la largeur de la direction packée du chemin « config » d'i2 dyn.-préc.
// (`FUN_1406d8288(..., 0x1e)`). Elle vaut 30, comme l'argument de l'appel.
const fwdDir30Bits uint = 30

// ForwardVector30 rend la direction unitaire du chemin « config » (mode 1) d'i2 dyn.-préc., et
// sa validité. C'est le MÊME dépaqueteur cubemap que [BipedPosition.AimVector], à une largeur
// de 30 bits au lieu de 19 — la largeur que l'écrivain passe, pas une supposition.
func (p BipedPosition) ForwardVector30() ([3]float32, bool) {
	if !p.HasFwdDir30 {
		return [3]float32{}, false
	}
	return DecodeAimVectorChecked(p.FwdDir30Raw, fwdDir30Bits)
}

// readForwardComponentDynPrec consomme i2 `object-forward-and-up-DYNAMIC-PRECISION-component`
// (FUN_140c5f7ec, ti=38/39/40/43). La grammaire n'est PAS réécrite ici : on repositionne le
// lecteur de l'appelant et on appelle son unique détenteur (components_dynprec_orientation.go).
//
// AUCUNE LARGEUR NE CHANGE avec la capture du lot 5.2b.2 : les trois chemins consomment
// exactement ce qu'ils consommaient, et `br.BitPos()` sort au même bit.
func readForwardComponentDynPrec(br *Lecteur, at, total int, out *componentDirs, param uint32) (int, bool) {
	br.SetBitPos(at)
	v, ok := decodeObjectForwardAndUpDynPrec(br, param)
	if !ok || br.BitPos() > total {
		return at, false
	}
	if v.HasDir {
		out.HasAim = true
		out.AimRaw = v.DirRaw
	}
	out.FwdMode = v.Mode
	if v.HasDir30 {
		out.HasFwdDir30, out.FwdDir30Raw = true, v.Dir30Raw
	}
	if v.HasVecs {
		out.HasFwdVecs, out.FwdVec1, out.FwdVec2 = true, v.Vec1, v.Vec2
	}
	return br.BitPos(), true
}
