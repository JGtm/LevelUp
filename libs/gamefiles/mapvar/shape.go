// Package mapvar — shape.go : la FORME d'un objectif surfacique.
//
// ORIGINE (méthode, pas devinette) — le sac de propriétés #8 de chaque objet
// porte, à `#0[0] → #0[0]`, une structure jamais décodée :
//
//	#0  i32  famille de forme   2 = cylindre, 3 = boîte
//	#5  i32  dimension A        virgule fixe 16.16
//	#6  i32  dimension B        virgule fixe 16.16
//	#7  i32  hauteur au-dessus du centre
//	#8  i32  profondeur au-dessous du centre
//
// Le pas de 65536 est ÉTABLI : 393216 = 6 × 65536 exactement, et 39,2 % des
// 62 299 valeurs brutes des 199 variantes sont des multiples exacts de 65536
// (valeurs rondes de concepteur). Aucune autre famille que 2 ou 3 n'apparaît
// sur les 16 434 formes mesurées.
//
// TAILLES PLEINES, PAS DEMI-EXTENTS — le départ a demandé deux mesures
// indépendantes et concordantes (état de l'art §Q2) :
//
//  1. 11 coïncidences exactes cylindre/boîte sous la lecture « pleine » contre
//     1 sous la lecture « demi », sur 79 paires de même carte et même rôle ;
//  2. l'oracle d'exécution : aux 4 instants du relevé terrain de Vagabond, le
//     rapport signal/témoin vaut 6,0 en lecture pleine contre 1,4 en demi —
//     sous la lecture demi, le témoin négatif attrape autant voire plus de
//     joueurs que les vraies zones.
//
// La conversion se fait donc UNE fois, ici, et le contrat ne publie que des
// demi-extents : aucun consommateur n'a à rejouer ce départ. Le brut est
// conservé à côté du dérivé — si le départ était un jour infirmé, on
// recalcule sans ré-extraire les 199 cartes.
package mapvar

// ShapeFamily est la famille géométrique d'une zone.
type ShapeFamily string

// Les deux seules familles observées sur 16 434 formes.
const (
	ShapeBox      ShapeFamily = "box"
	ShapeCylinder ShapeFamily = "cylinder"
)

// Codes de famille tels qu'écrits dans le fichier.
const (
	shapeCodeCylinder = 2
	shapeCodeBox      = 3
)

// fixedPointDivisor : les dimensions sont en virgule fixe 16.16.
const fixedPointDivisor = 65536.0

// ShapeRaw conserve les entiers du fichier, non convertis. C'est la donnée du
// jeu ; le reste de Shape est notre interprétation.
type ShapeRaw struct {
	Family int32 `json:"family"`
	S5     int64 `json:"s5"`
	S6     int64 `json:"s6"`
	S7     int64 `json:"s7"`
	S8     int64 `json:"s8"`
}

// Shape est la forme d'un objectif surfacique, en mètres et en DEMI-EXTENTS.
//
// `Forward` et `Up` sont TOUJOURS présents : une boîte alignée sur les axes du
// monde n'est jamais une réponse acceptable. Mesuré sur Cliffhanger, ignorer
// `Forward` sur une zone d'Extraction tournée de -20° déclare « dedans » 31 %
// de positions qui sont dehors.
type Shape struct {
	Family ShapeFamily `json:"family"`
	// Boîte uniquement : demi-largeur le long de Forward, demi-profondeur.
	HalfX *float64 `json:"half_x,omitempty"`
	HalfY *float64 `json:"half_y,omitempty"`
	// Cylindre uniquement : rayon.
	Radius *float64 `json:"radius,omitempty"`
	// Extension verticale, mesurée depuis le centre.
	UpZ     float64  `json:"up_z"`
	DownZ   float64  `json:"down_z"`
	Forward Vec3     `json:"forward"`
	Up      Vec3     `json:"up"`
	Raw     ShapeRaw `json:"raw"`
}

// readShape décode le sac de forme d'un objet gameplay.
//
// Rend nil quand il n'y a pas de forme — ce qui est le cas NORMAL des
// objectifs ponctuels : 100 % des objectifs surfaciques en portent une
// (zones d'Extraction 434/434, repères Ravitaillement 186/186, zones Bastion
// 430/431), 0 % des ponctuels (apparitions de drapeau 0/669, socles 0/855).
// Un consommateur qui reçoit nil doit afficher un POINT, jamais un disque par
// défaut : 65,8 % des objectifs sont des points, un rayon inventé créerait
// 2 155 zones qui n'existent pas.
func readShape(gp Value) *ShapeRaw {
	lst, ok := gp.Field(0)
	if !ok || len(lst.Items) == 0 {
		return nil
	}
	sh := lst.Items[0]
	kind, ok := sh.Field(0)
	if !ok {
		return nil
	}
	raw := ShapeRaw{Family: int32(kind.Int)}
	raw.S5 = shapeSlot(sh, 5)
	raw.S6 = shapeSlot(sh, 6)
	raw.S7 = shapeSlot(sh, 7)
	raw.S8 = shapeSlot(sh, 8)
	return &raw
}

// shapeSlot lit un emplacement numérique du sac de forme. Chaque emplacement
// est une structure enveloppant sa valeur au champ 0 ; un emplacement absent
// vaut 0 (Bond omet les défauts).
func shapeSlot(sh Value, id uint16) int64 {
	f, ok := sh.Field(id)
	if !ok {
		return 0
	}
	v, ok := f.Field(0)
	if !ok {
		return 0
	}
	return v.Int
}

// Shape convertit le brut d'un objet en forme orientée, en mètres.
//
// Rend nil si l'objet ne porte pas de forme, ou si la famille n'est ni 2 ni 3 :
// une famille inconnue reste SANS forme, on ne devine pas une géométrie.
func (o Object) Shape() *Shape {
	if o.ShapeRaw == nil {
		return nil
	}
	r := *o.ShapeRaw
	s := &Shape{
		UpZ:     float64(r.S7) / fixedPointDivisor,
		DownZ:   float64(r.S8) / fixedPointDivisor,
		Forward: o.Forward,
		Up:      o.Up,
		Raw:     r,
	}
	switch r.Family {
	case shapeCodeBox:
		s.Family = ShapeBox
		hx := float64(r.S5) / fixedPointDivisor / 2
		hy := float64(r.S6) / fixedPointDivisor / 2
		s.HalfX, s.HalfY = &hx, &hy
	case shapeCodeCylinder:
		s.Family = ShapeCylinder
		// Le cylindre publie un RAYON, pas un diamètre : pas de division par 2.
		rad := float64(r.S5) / fixedPointDivisor
		s.Radius = &rad
	default:
		return nil
	}
	return s
}
