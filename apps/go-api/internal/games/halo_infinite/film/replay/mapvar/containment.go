// Package mapvar — containment.go : « ce point est-il DANS cette zone ? »
//
// C'est la brique manquante entre deux chantiers deja faits : les formes de zone
// (shape.go, 16 434 formes decodees) et les positions de joueur du rejeu 2D. Ni
// l'un ni l'autre ne repondait a la question, parce qu'elle exige les deux.
//
// # Le test est en 3D, et ce n'est pas un raffinement
//
// Les zones de Bastion mesurees dans le catalogue versionne s'etendent de 1,0 a
// 4,0 m au-dessus de leur centre et de 0,0 a 1,0 m en dessous. Une carte a
// etages (Streets, Cliffhanger) place des couloirs a l'aplomb d'une zone : un
// test 2D y declarerait « dedans » un joueur situe un etage plus haut.
//
// # L'orientation compte, et elle est TOUJOURS renseignee
//
// Mesure sur les 63 formes de zone Bastion et les 161 formes d'Extraction du
// catalogue : 0 vecteur Forward nul, 0 vecteur Up nul, 0 cas ou Forward est
// parallele a Up. L'axe Up est vertical (uz > 0,999) sur la totalite des formes
// de zone. Il n'existe donc AUCUNE forme de zone pour laquelle un repli sur les
// axes du monde serait necessaire — et c'est pourquoi le repli n'existe pas :
// [NewVolume] REFUSE une forme degeneree au lieu de deviner une geometrie.
// Ignorer Forward sur une zone d'Extraction tournee de -20 degres declarait
// « dedans » 31 % de positions qui sont dehors (shape.go).
//
// # Le volume se construit UNE fois, puis se teste N fois
//
// Le croisement pose des dizaines de milliers de positions contre une poignee de
// zones. La base orthonormee est donc calculee a la construction, pas a chaque
// point.
package mapvar

import (
	"errors"
	"math"
)

// ErrNoShape signale un objectif PONCTUEL : il n'a pas de forme, donc pas de
// volume. 65,8 % des objectifs sont dans ce cas (shape.go) — ce n'est pas une
// anomalie, et l'appelant doit le compter a part plutot que de l'assimiler a un
// echec de lecture.
var ErrNoShape = errors.New("mapvar: objectif ponctuel (aucune forme)")

// ErrDegenerateFrame signale une forme dont l'orientation ne permet pas de
// construire un repere : Up nul, ou Forward colineaire a Up. Mesure sur le
// catalogue : 0 occurrence sur les 224 formes de zone. Une occurrence future
// serait donc une nouveaute a expliquer, jamais un cas a rattraper par defaut.
var ErrDegenerateFrame = errors.New("mapvar: orientation degeneree (repere non constructible)")

// ErrMissingExtent signale une forme a laquelle manque la dimension de sa propre famille
// (une boite sans demi-cotes, un cylindre sans rayon). Cas d'un document relu d'un schema
// anterieur : sans ce refus, le volume serait d'extension nulle et ne contiendrait jamais
// rien — un zero silencieux, indistinguable d'un joueur qui n'y etait pas.
var ErrMissingExtent = errors.New("mapvar: forme sans dimension pour sa famille")

// degenerateEpsilon : en deca de cette norme, un vecteur d'orientation est tenu
// pour nul. Les vecteurs du fichier sont des unitaires ; 1e-6 separe un zero
// exact du bruit de la virgule flottante sans jamais rejeter un unitaire.
const degenerateEpsilon = 1e-6

// Volume est une forme de zone POSEE dans le monde : sa forme, son centre et sa
// base orthonormee, pretes pour le test d'appartenance.
type Volume struct {
	center Vec3
	// fwd, right, up forment une base orthonormee directe. up porte l'axe de
	// hauteur (et l'axe du cylindre) ; fwd porte la demi-largeur de la boite.
	fwd, right, up Vec3
	family         ShapeFamily
	halfX, halfY   float64
	radius         float64
	upZ, downZ     float64
}

// NewVolume pose une forme au centre donne et prepare son repere.
//
// Rend ErrNoShape si l'objectif est ponctuel, ErrDegenerateFrame si son
// orientation ne permet pas de construire une base.
func NewVolume(center Vec3, s *Shape) (Volume, error) {
	if s == nil {
		return Volume{}, ErrNoShape
	}
	up, ok := normalize(s.Up)
	if !ok {
		return Volume{}, ErrDegenerateFrame
	}
	// Gram-Schmidt : Forward est redresse dans le plan perpendiculaire a Up. Le
	// fichier les donne deja orthogonaux, mais les redresser coute une
	// soustraction et garantit que la base l'est vraiment — sans quoi une
	// derive de 1 degre biaiserait toutes les demi-largeurs.
	fwd, ok := normalize(sub(s.Forward, scale(up, dot(s.Forward, up))))
	if !ok {
		return Volume{}, ErrDegenerateFrame
	}
	v := Volume{
		center: center,
		fwd:    fwd,
		right:  cross(up, fwd),
		up:     up,
		family: s.Family,
		upZ:    s.UpZ,
		downZ:  s.DownZ,
	}
	// Les dimensions sont des POINTEURS parce qu'une boite n'a pas de rayon et un
	// cylindre pas de demi-cotes. Une forme relue d'un JSON ou la dimension de sa propre
	// famille manque n'est pas une forme plate : c'est une forme ILLISIBLE, et la laisser
	// passer donnerait un volume d'extension nulle qui ne contient jamais rien — un zero
	// silencieux, indistinguable d'un joueur qui n'y etait pas.
	switch s.Family {
	case ShapeBox:
		if s.HalfX == nil || s.HalfY == nil {
			return Volume{}, ErrMissingExtent
		}
		v.halfX, v.halfY = *s.HalfX, *s.HalfY
	case ShapeCylinder:
		if s.Radius == nil {
			return Volume{}, ErrMissingExtent
		}
		v.radius = *s.Radius
	}
	return v, nil
}

// Contains dit si un point du monde est dans le volume.
//
// Les bornes sont INCLUSIVES : un joueur exactement sur la limite est dedans.
// Le quantum du decodeur de positions vaut ~1,4 cm — a cette resolution,
// exclure la limite ne changerait aucun verdict reel, et l'inclure garde le
// test coherent avec la lecture « tailles pleines » etablie dans shape.go.
func (v Volume) Contains(p Vec3) bool {
	d := sub(p, v.center)
	lz := dot(d, v.up)
	if lz > v.upZ || lz < -v.downZ {
		return false
	}
	lx := dot(d, v.fwd)
	ly := dot(d, v.right)
	switch v.family {
	case ShapeBox:
		return math.Abs(lx) <= v.halfX && math.Abs(ly) <= v.halfY
	case ShapeCylinder:
		// Compare les carres : evite une racine par point, et le rayon est
		// toujours positif.
		return lx*lx+ly*ly <= v.radius*v.radius
	default:
		// Une famille inconnue n'a pas de geometrie : Shape() rend deja nil
		// dans ce cas, donc ce chemin ne devrait pas exister. Refuser plutot
		// que supposer.
		return false
	}
}

// DistanceTo rend la distance du point AU VOLUME, en metres. Zero quand le point est
// dedans — Contains(p) equivaut donc exactement a DistanceTo(p) == 0.
//
// # Pourquoi la distance au VOLUME et pas au CENTRE
//
// Les zones ne font pas la meme taille : rayon 1,7 a 5,0 m pour les zones de Bastion,
// demi-cotes jusqu'a 5,1 m. Comparer des distances au centre classerait systematiquement
// la plus petite zone comme « la plus proche » d'un joueur place entre deux — c'est un
// biais de taille, pas une mesure de proximite.
//
// Le calcul est le classique « ecart a la boite » : on annule chaque composante ou le
// point est deja dans les bornes, et on prend la norme de ce qui reste.
func (v Volume) DistanceTo(p Vec3) float64 {
	d := sub(p, v.center)
	lz := dot(d, v.up)
	dz := math.Max(0, math.Max(lz-v.upZ, -v.downZ-lz))
	lx := dot(d, v.fwd)
	ly := dot(d, v.right)
	switch v.family {
	case ShapeBox:
		dx := math.Max(0, math.Abs(lx)-v.halfX)
		dy := math.Max(0, math.Abs(ly)-v.halfY)
		return math.Sqrt(dx*dx + dy*dy + dz*dz)
	case ShapeCylinder:
		dr := math.Max(0, math.Hypot(lx, ly)-v.radius)
		return math.Hypot(dr, dz)
	default:
		// Famille inconnue : aucune geometrie, donc aucune distance mesurable. Rendre
		// l'infini plutot que zero — zero voudrait dire « dedans ».
		return math.Inf(1)
	}
}

// Center rend le centre du volume, tel qu'il a ete pose.
func (v Volume) Center() Vec3 { return v.center }

// Translate rend une COPIE du volume deplacee du vecteur donne, base inchangee.
//
// Sert au TEMOIN NEGATIF : la meme forme posee ailleurs doit attraper nettement
// moins de monde que la vraie. Sans ce temoin, « le joueur est dans la zone »
// n'est pas une mesure — une zone assez grande attrape tout le monde (etat de
// l'art, depart demi-extents / tailles pleines).
func (v Volume) Translate(d Vec3) Volume {
	v.center = Vec3{X: v.center.X + d.X, Y: v.center.Y + d.Y, Z: v.center.Z + d.Z}
	return v
}

// --- algebre vectorielle locale (aucune dependance externe) ---

func sub(a, b Vec3) Vec3 { return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z} }

func scale(a Vec3, k float64) Vec3 { return Vec3{X: a.X * k, Y: a.Y * k, Z: a.Z * k} }

func dot(a, b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func cross(a, b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// normalize rend le vecteur unitaire et true, ou le vecteur nul et false quand
// la norme est sous le seuil de degenerescence.
func normalize(a Vec3) (Vec3, bool) {
	n := math.Sqrt(dot(a, a))
	if n < degenerateEpsilon {
		return Vec3{}, false
	}
	return scale(a, 1/n), true
}
