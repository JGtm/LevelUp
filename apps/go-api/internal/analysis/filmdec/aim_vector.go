package filmdec

import "math"

// Décodage d'une DIRECTION UNITAIRE empaquetée (cubemap 6 faces) — port bit-exact de
// FUN_1406d8288 (HaloInfinite.exe), trianguléе par FUN_1406d8228 (divmod nu),
// FUN_1406d8b98 (même reconstruction, faces en littéraux) et FUN_1407eaf1c (l'encodeur).
//
// C'est le déquantificateur commun à TOUS les champs de direction du film :
//   - i1 object-translational-velocity / i3 angular-velocity : R(19) packed dir
//   - i2 object-forward-and-up : R(19) packed dir (cap/facing du biped)
//   - i17 player-control-aiming (ti=5) : R(19)
//   - FUN_14076dc04 du default-state biped : R(19)
//   - aim des fire events : R(30)
//
// GRAMMAIRE (p = largeur en bits du champ lu = index de précision) :
//
//	F    = faceSize[p]              // table .data 0x1447084D0 + 8*p (offset +0)
//	face = code / F ; rem = code % F
//	N    = gridSize[p]              // même entrée, offset +4 (peuplée au runtime)
//	iu   = rem / N ; iv = rem % N   // 2e DIVMOD : LA 2e COORDONNÉE
//	step = 2 / (N-1)
//	a = iu*step - 1 + step/2 (0 exact si iu*2 == N-2)
//	b = iv*step - 1 + step/2 (0 exact si iv*2 == N-2)
//	face 0 (+X): (1,a,b)  1 (+Y): (a,1,b)  2 (+Z): (a,b,1)
//	face 3 (-X): (-1,a,b) 4 (-Y): (a,-1,b) 5 (-Z): (a,b,-1)
//	puis normalisation L2 (epsilon 1e-4). AUCUN changement de signe sur a,b.
//
// TABLES : faceSize(p) = floor(2^p / 6) — lu octet à octet dans le .exe pour p=6..30.
// gridSize(p) = floor(sqrt(2^p / 6)) - 1 — dérivé de l'initialiseur FUN_14038bc40
// (gridSize = (int)sqrtf(K_p) - 1, K_p = 2^p/6 en float32). Les deux sont figées en dur
// ci-dessous : zéro dépendance runtime, zéro recalcul.
//
// ÉCART AVEC LA VERSION COMMUNAUTAIRE (dev externe) : elle utilise FACE_SIZE = 0xAAA8000
// (faux : 6*0xAAA8000 = 2^30-65536, la vraie valeur est 0x0AAAAAAA) et pose la 2e
// coordonnée à 0 (elle n'avait pas le 2e divmod), d'où jusqu'à 45° d'erreur aux coutures.
// Elle applique aussi des négations sur les coordonnées libres : l'exe n'en applique
// aucune (invisible chez elle puisque sa 2e coordonnée était nulle).

// cubemapMinWidth / cubemapMaxWidth bornent le domaine légal de p : sous 6 la table
// voisine contient des descripteurs d'octets (entry[5] = (0,0) -> division par zéro),
// au-delà de 30 on sort de la table (0x1447085C8 contient autre chose).
const (
	cubemapMinWidth uint = 6
	cubemapMaxWidth uint = 30
)

// cubemapTable[p-6] = {faceSize, gridSize} pour p = 6..30. Colonne faceSize relevée
// octet à octet à 0x1447084D0+8*p (= floor(2^p/6)) ; colonne gridSize dérivée de
// FUN_14038bc40 (= floor(sqrt(2^p/6)) - 1 ; nulle en statique, peuplée au map-load).
var cubemapTable = [cubemapMaxWidth - cubemapMinWidth + 1][2]int32{
	{10, 2}, {21, 3}, {42, 5}, {85, 8}, {170, 12}, // p=6..10
	{341, 17}, {682, 25}, {1365, 35}, {2730, 51}, {5461, 72}, // p=11..15
	{10922, 103}, {21845, 146}, {43690, 208}, {87381, 294}, {174762, 417}, // p=16..20
	{349525, 590}, {699050, 835}, {1398101, 1181}, {2796202, 1671}, {5592405, 2363}, // p=21..25
	{11184810, 3343}, {22369621, 4728}, {44739242, 6687}, {89478485, 9458}, // p=26..29
	{178956970, 13376}, // p=30
}

// cubemapParams renvoie (faceSize, gridSize) pour une largeur de champ p, et ok=false
// hors du domaine légal [6,30].
func cubemapParams(width uint) (faceSize, gridSize int, ok bool) {
	if width < cubemapMinWidth || width > cubemapMaxWidth {
		return 0, 0, false
	}
	e := cubemapTable[width-cubemapMinWidth]
	return int(e[0]), int(e[1]), true
}

// DecodeAimVector déquantifie une direction empaquetée de `width` bits en un vecteur
// UNITAIRE dans le repère monde (port de FUN_1406d8288). Les valeurs hors domaine
// (largeur illégale, face >= 6) renvoient la sentinelle de l'exe (0,0,1) ; utiliser
// DecodeAimVectorChecked pour distinguer une sentinelle d'un vrai +Z.
func DecodeAimVector(encoded uint32, width uint) (x, y, z float32) {
	v, _ := DecodeAimVectorChecked(encoded, width)
	return v[0], v[1], v[2]
}

// DecodeAimVectorChecked décode comme DecodeAimVector et signale la validité. ok=false
// signifie : largeur hors table, ou face >= 6 — ce dernier cas est le CLIQUET
// D'ALIGNEMENT du flux (un curseur de bits décalé produit presque toujours face >= 6).
func DecodeAimVectorChecked(encoded uint32, width uint) ([3]float32, bool) {
	faceSize, n, ok := cubemapParams(width)
	if !ok || n < 2 {
		return [3]float32{0, 0, 1}, false
	}
	code := int(encoded)
	face := code / faceSize
	rem := code - faceSize*face
	if face < 0 || face > 5 {
		return [3]float32{0, 0, 1}, false // DAT_143cf7758 : sentinelle "invalide"
	}
	iu := rem / n
	iv := rem - n*iu

	step := 2.0 / float32(n-1)
	a := cubemapCoord(iu, n, step)
	b := cubemapCoord(iv, n, step)

	var v [3]float32
	switch face {
	case 0: // +X
		v = [3]float32{1, a, b}
	case 1: // +Y
		v = [3]float32{a, 1, b}
	case 2: // +Z
		v = [3]float32{a, b, 1}
	case 3: // -X
		v = [3]float32{-1, a, b}
	case 4: // -Y
		v = [3]float32{a, -1, b}
	default: // 5 : -Z
		v = [3]float32{a, b, -1}
	}
	return normalizeCubemap(v), true
}

// cubemapCoord calcule une coordonnée de cellule centrée : c = 2*(i+0.5)/(N-1) - 1, avec
// le snap exact à zéro de l'exe (i*2 == N-2 : nettoyage d'erreur flottante).
func cubemapCoord(i, n int, step float32) float32 {
	if i*2 == n-2 {
		return 0
	}
	return float32(i)*step - 1.0 + step*0.5
}

// normalizeCubemap applique la normalisation L2 "safe" de l'exe (epsilon 1e-4 =
// DAT_143cd837c). La norme brute vaut toujours >= 1 (l'axe dominant vaut ±1), donc
// le garde-fou ne se déclenche jamais en pratique.
func normalizeCubemap(v [3]float32) [3]float32 {
	m := float32(sqrt32(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]))
	if m < 1e-4 {
		return v
	}
	return [3]float32{v[0] / m, v[1] / m, v[2] / m}
}

// DecodeAimVectorFlat décode SANS la 2e coordonnée : c'est exactement la version du dev
// externe (b forcé à 0). Conservée UNIQUEMENT pour chiffrer le gain apporté par le 2e
// divmod dans les outils de validation — n'appartient à aucun chemin de production.
func DecodeAimVectorFlat(encoded uint32, width uint) ([3]float32, bool) {
	faceSize, n, ok := cubemapParams(width)
	if !ok || n < 2 {
		return [3]float32{0, 0, 1}, false
	}
	code := int(encoded)
	face := code / faceSize
	rem := code - faceSize*face
	if face < 0 || face > 5 {
		return [3]float32{0, 0, 1}, false
	}
	iu := rem / n
	step := 2.0 / float32(n-1)
	a := cubemapCoord(iu, n, step)
	var v [3]float32
	switch face {
	case 0:
		v = [3]float32{1, a, 0}
	case 1:
		v = [3]float32{a, 1, 0}
	case 2:
		v = [3]float32{a, 0, 1}
	case 3:
		v = [3]float32{-1, a, 0}
	case 4:
		v = [3]float32{a, -1, 0}
	default:
		v = [3]float32{a, 0, -1}
	}
	return normalizeCubemap(v), true
}

// EncodeAimVector porte FUN_1407eaf1c (l'inverse) : il ne sert qu'à valider le décodeur
// par aller-retour (test unitaire), aucun chemin de production ne l'appelle.
func EncodeAimVector(v [3]float32, width uint) (uint32, bool) {
	faceSize, n, ok := cubemapParams(width)
	if !ok || n < 2 {
		return 0, false
	}
	ax, ay, az := absf(v[0]), absf(v[1]), absf(v[2])
	var face int
	var c0, c1 float32
	switch {
	case ay <= ax && az <= ax: // |X| dominant
		face = 3
		if v[0] > 0 {
			face = 0
		}
		c0, c1 = v[1]/ax, v[2]/ax
	case az <= ay: // |Y| dominant
		face = 4
		if v[1] > 0 {
			face = 1
		}
		c0, c1 = v[0]/ay, v[2]/ay
	default: // |Z| dominant
		face = 5
		if v[2] > 0 {
			face = 2
		}
		c0, c1 = v[0]/az, v[1]/az
	}
	step := 2.0 / float32(n-1)
	iu := clampi(int((c0+1.0)/step), 0, n-2)
	iv := clampi(int((c1+1.0)/step), 0, n-2)
	return uint32(face*faceSize + n*iu + iv), true
}

// sqrt32 : racine carrée calculée en float64 puis ramenée en float32 (l'exe utilise
// sqrtf ; l'écart est sous l'ULP pour ce domaine, où la norme vaut toujours >= 1).
func sqrt32(f float32) float32 {
	if f <= 0 {
		return 0
	}
	return float32(math.Sqrt(float64(f)))
}
