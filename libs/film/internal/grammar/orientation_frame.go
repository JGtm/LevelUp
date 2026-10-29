package grammar

import "math"

// orientation_frame.go — L AVANT D UN CHASSIS N EST PAS ECRIT : IL EST RECONSTRUIT.
//
// # CE QUE LE FILM ECRIT VRAIMENT POUR `object-forward-and-up`
//
// Le composant ne porte PAS deux vecteurs. Il porte UNE direction unitaire (cubemap) et UN
// ANGLE, et le moteur fabrique le second vecteur a partir des deux. C est la reponse au negatif
// du lot 5.2b.2, qui avait mesure la seule direction ecrite et l avait trouvee quasi VERTICALE
// (|z| median 0,960 a 0,981) : la direction ecrite est le vecteur HAUT, et l AVANT est la
// perpendiculaire construite.
//
// LA CHAINE, RELUE AU DECOMPILE (Ghidra, lecture seule, 2026-09-20) :
//
//	FUN_140c5f7ec (deser i2)   lit les bits, puis ecrit DEUX vec3 dans le composant :
//	                           base+0x18 <- param_2   = la PERPENDICULAIRE construite
//	                           base+0x24 <- param_3   = la DIRECTION lue (cubemap)
//	FUN_140c5f9c8 (0 bit)      dir  = FUN_1406d8b98(face, iu, iv, 0x13)  ou, porte posee,
//	                                  DAT_143b8f860 = (0, 0, 1)
//	                           theta = raw * DAT_143cd891c - DAT_143cd8918 + DAT_143cd97a0
//	                           FUN_1406d8678(&dir, theta, out=param_2) ; *param_3 = dir
//
// LES TROIS CONSTANTES DE L ANGLE, LUES DANS LE BINAIRE (`/read_memory`) :
// `DAT_143cd891c = 0x3cc90fdb = pi/128` (le pas de 8 bits sur un tour), `DAT_143cd8918 =
// 0x40490fdb = +pi`, `DAT_143cd97a0 = 0x3c490fdb = pi/256` (le demi-pas). Soit exactement
// `dequantMidpoint(raw, 8, -pi, +pi)` : la convention du MILIEU D INTERVALLE n est plus
// « retenue » pour ce champ, elle est MESUREE.
//
// LE DEFAUT VAUT (0, 0, 1). Quand la porte de direction est posee, le moteur prend
// `DAT_143b8f860 = (0, 0, 1)` (relu : `00000000 00000000 0000803f`). Un AVANT par defaut
// vertical n aurait aucun sens ; un HAUT par defaut vertical est l objet a plat. C est une
// preuve STRUCTURELLE, pas statistique, que la direction ecrite est le HAUT.
//
// LE MEME COUPLE (direction, angle) SE LIT AILLEURS, a d autres largeurs : le mode 1 d i2
// (`FUN_142e29bac` : R(30) direction + R(30) angle, chemin DOMINANT sur les builds recents) et
// la feuille 4 de l etat par defaut de `ti=40` (`FUN_140c1e79c` : R(19) + R(8)). La
// reconstruction ci-dessous leur sert a toutes.

// baseAxisX / baseAxisY sont les deux vecteurs de base entre lesquels FUN_1406d8678 choisit.
// Relus par leurs pointeurs : `PTR_DAT_14474c2f8 -> 0x14472a63c = (1, 0, 0)` et
// `PTR_DAT_14474c2e0 -> 0x14472a648 = (0, 1, 0)`.
var (
	baseAxisX = [3]float32{1, 0, 0}
	baseAxisY = [3]float32{0, 1, 0}
)

// orientationNormEpsilon = `DAT_143cd837c = 1,0e-4`, le seuil sous lequel FUN_1406d8678 laisse
// la perpendiculaire NON normalisee (il ne divise pas par une norme quasi nulle).
const orientationNormEpsilon = 1e-4

// RollAngleFromRaw dequantifie l angle de roulis d un couple (direction, angle) sur `bits` bits
// dans [-pi, +pi]. Les trois constantes du chemin de 8 bits sont relues dans le binaire (en-tete
// de ce fichier) et redonnent exactement cette formule ; le chemin de 30 bits passe par
// `FUN_1406d84b4(n = 0x1e, min = -pi, max = +pi)`, de meme convention.
func RollAngleFromRaw(raw uint32, bits uint) float32 {
	return dequantMidpoint(uint64(raw), bits, -math.Pi, math.Pi)
}

// ForwardFromUpRoll porte FUN_1406d8678 : il rend le vecteur UNITAIRE et PERPENDICULAIRE a `up`
// que le moteur ecrit a `base+0x18`, c est-a-dire L AVANT DU CHASSIS.
//
// La construction, a la lettre du decompile :
//
//  1. base = celui de (1,0,0) / (0,1,0) le MOINS aligne avec `up` (comparaison des |produits
//     scalaires|) ; l asymetrie d ordre du produit vectoriel est celle de l executable :
//     branche X -> `up x X`, branche Y -> `Y x up`. Elle n est pas cosmetique, elle fixe le SENS.
//  2. normalisation si la norme atteint 1,0e-4.
//  3. rotation de Rodrigues autour de `up`, d angle `roll` ; a +-pi exactement, l executable
//     court-circuite les appels trigonometriques (cos = -1, sin = 0 ; `DAT_143cd84ec`).
//  4. normalisation finale (FUN_1404fec88).
func ForwardFromUpRoll(up [3]float32, roll float32) [3]float32 {
	var v [3]float32
	if absf(dot3(up, baseAxisX)) < absf(dot3(up, baseAxisY)) {
		v = cross3(up, baseAxisX)
	} else {
		v = cross3(baseAxisY, up)
	}
	if n := norm3(v); n >= orientationNormEpsilon {
		v = [3]float32{v[0] / n, v[1] / n, v[2] / n}
	}
	cos, sin := float32(-1), float32(0)
	if roll != math.Pi && roll != -math.Pi {
		cos = float32(math.Cos(float64(roll)))
		sin = float32(math.Sin(float64(roll)))
	}
	// Rodrigues : v*cos + (up x v)*sin + up*(up.v)*(1-cos). Le terme axial est nul en
	// arithmetique exacte (v est perpendiculaire a up) ; il est conserve parce que
	// l executable le calcule, et avec lui l erreur flottante qu il rattrape.
	k := cross3(up, v)
	ax := dot3(v, up) * (1 - cos)
	out := [3]float32{
		v[0]*cos + up[0]*ax + k[0]*sin,
		v[1]*cos + up[1]*ax + k[1]*sin,
		v[2]*cos + up[2]*ax + k[2]*sin,
	}
	if n := norm3(out); n > 0 {
		out = [3]float32{out[0] / n, out[1] / n, out[2] / n}
	}
	return out
}

func dot3(a, b [3]float32) float32 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func cross3(a, b [3]float32) [3]float32 {
	return [3]float32{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func norm3(v [3]float32) float32 { return sqrt32(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]) }

// AimVector renvoie la direction unitaire décodée d i2 — le vecteur HAUT de l objet — et sa
// validité. LA LARGEUR SUIT LE MODE : 19 bits sur le chemin nominal, 30 sur le chemin « config »
// (mode 1). La décoder à 19 bits quand elle en fait 30 rend un vecteur arbitraire ; c est
// pourquoi la largeur n est plus une constante ici.
func (p BipedPosition) AimVector() ([3]float32, bool) {
	if p.AimDefault {
		return [3]float32{0, 0, 1}, true
	}
	if !p.HasAim {
		return [3]float32{}, false
	}
	return DecodeAimVectorChecked(p.AimRaw, FwdUpDirBits(p.FwdMode))
}

// ChassisForwardVector rend L AVANT DU CHASSIS : la perpendiculaire au vecteur HAUT que le
// moteur reconstruit avec l angle de roulis ([ForwardFromUpRoll]). ok est faux dès qu une des
// deux moitiés manque — en particulier sur le chemin « delta », qui n écrit qu un incrément
// d angle.
func (p BipedPosition) ChassisForwardVector() ([3]float32, bool) {
	up, ok := p.AimVector()
	if !ok || !p.HasRoll {
		return [3]float32{}, false
	}
	return ForwardFromUpRoll(up, RollAngleFromRaw(p.RollRaw, FwdUpRollBits(p.FwdMode))), true
}
