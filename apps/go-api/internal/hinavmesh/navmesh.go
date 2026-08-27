package hinavmesh

// navmesh.go — DU FICHIER-TAG AU MAILLAGE, en coordonnees MONDE.
//
// hkaiNavMesh porte trois tableaux : `vertices` (hkVector4), `edges` (hkaiNavMesh::Edge)
// et `faces` (hkaiNavMesh::Face). Une face designe une tranche CONTIGUE du tableau des
// aretes, [startEdgeIndex, startEdgeIndex + numEdges) ; le contour du polygone est la
// suite des sommets `a` de ces aretes, dans l'ordre. Les polygones sont convexes et de 3
// a 10 cotes sur les cartes mesurees.
//
// Le seul element de disposition qui n'est PAS declare par le fichier est l'interieur du
// hkVector4 : la table des types le donne pour 16 octets sans membre. On lit donc 4
// float32 (x, y, z, w) — la taille de 16 est verifiee, et l'oracle des ancres tranche.

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Noms lus dans la reflexion du fichier-tag.
const (
	classeNavMesh    = "hkaiNavMesh"
	classeFace       = "hkaiNavMesh::Face"
	classeEdge       = "hkaiNavMesh::Edge"
	classeVecteur4   = "hkVector4"
	membreFaces      = "faces"
	membreAretes     = "edges"
	membreSommets    = "vertices"
	membreDebutArc   = "startEdgeIndex"
	membreNbAretes   = "numEdges"
	membreSommetA    = "a"
	membreSommetB    = "b"
	membreHaut       = "up"
	membreErosion    = "erosionRadius"
	tailleVecteur4   = 16
	cotesFaceMinimum = 3
)

// Point est une position en coordonnees MONDE (metres), dans le repere du jeu : Z vers
// le haut. Le vecteur `up` du maillage est lu et verifie, jamais suppose.
type Point struct{ X, Y, Z float64 }

// Face est un polygone convexe du maillage : des indices dans Maillage.Sommets, dans
// l'ordre du contour.
type Face struct{ Sommets []int32 }

// Maillage est le maillage de navigation d'une carte, en coordonnees MONDE.
type Maillage struct {
	Sommets []Point
	Faces   []Face
	// Min et Max sont l'emprise MESUREE sur les sommets. L'AABB que porte le fichier est
	// celle du CANEVAS Forge (plus de 400 m de cote) et ne decrit pas l'aire jouable.
	Min, Max Point
	// Haut est le vecteur `up` declare par le maillage. Mesure a (0,0,1) sur les cartes
	// testees : la projection vue de dessus se fait donc dans le plan XY.
	Haut Point
	// RayonErosion est le `erosionRadius` declare. Mesure a 0 sur les cartes testees :
	// le retrait du bord observe face aux ancres ne vient donc PAS de ce champ.
	RayonErosion float64
}

// Contour rend le polygone d'une face en coordonnees monde.
func (m *Maillage) Contour(f Face) []Point {
	pts := make([]Point, 0, len(f.Sommets))
	for _, i := range f.Sommets {
		pts = append(pts, m.Sommets[i])
	}
	return pts
}

// Triangles rend la soupe de triangles du maillage : chaque face convexe est eventail-ee
// depuis son premier sommet.
func (m *Maillage) Triangles() [][3]Point {
	tris := make([][3]Point, 0, len(m.Faces))
	for _, f := range m.Faces {
		for k := 1; k+1 < len(f.Sommets); k++ {
			tris = append(tris, [3]Point{
				m.Sommets[f.Sommets[0]], m.Sommets[f.Sommets[k]], m.Sommets[f.Sommets[k+1]],
			})
		}
	}
	return tris
}

// AireAuSol rend l'aire cumulee des faces PROJETEE dans le plan XY, en metres carres.
func (m *Maillage) AireAuSol() float64 {
	var aire float64
	for _, f := range m.Faces {
		var s float64
		n := len(f.Sommets)
		for k := 0; k < n; k++ {
			a, b := m.Sommets[f.Sommets[k]], m.Sommets[f.Sommets[(k+1)%n]]
			s += a.X*b.Y - b.X*a.Y
		}
		aire += math.Abs(s) / 2
	}
	return aire
}

// Decode lit un `navmesh.blob` complet et rend le maillage de navigation.
func Decode(blob []byte) (*Maillage, error) {
	charge, err := decompresse(blob)
	if err != nil {
		return nil, err
	}
	decoupe, err := regions(charge)
	if err != nil {
		return nil, err
	}
	for i, region := range decoupe {
		if len(region) < 8 || string(region[4:8]) != sectionTAG0 {
			continue
		}
		f, err := lireFichierTag(region)
		if err != nil {
			return nil, fmt.Errorf("hinavmesh: region %d: %w", i+1, err)
		}
		racine, err := f.racine()
		if err != nil {
			return nil, fmt.Errorf("hinavmesh: region %d: %w", i+1, err)
		}
		if f.nomType(racine.Type) != classeNavMesh {
			continue
		}
		return construitMaillage(f, racine)
	}
	return nil, fmt.Errorf("hinavmesh: aucune region ne porte un %s", classeNavMesh)
}

// construitMaillage assemble sommets, aretes et faces en un maillage exploitable.
func construitMaillage(f *fichierTag, racine itemHavok) (*Maillage, error) {
	sommets, err := lireSommets(f, racine)
	if err != nil {
		return nil, err
	}
	aretes, err := lireAretes(f, racine)
	if err != nil {
		return nil, err
	}
	faces, err := lireFaces(f, racine, aretes, len(sommets))
	if err != nil {
		return nil, err
	}
	m := &Maillage{Sommets: sommets, Faces: faces}
	if err := lireEntete(f, racine, m); err != nil {
		return nil, err
	}
	mesureEmprise(m)
	return m, nil
}

// lireSommets lit le tableau `vertices`.
func lireSommets(f *fichierTag, racine itemHavok) ([]Point, error) {
	it, err := f.tableau(racine, membreSommets)
	if err != nil {
		return nil, err
	}
	if nom := f.nomType(it.Type); nom != classeVecteur4 {
		return nil, fmt.Errorf("hinavmesh: %s est un tableau de %s, %s attendu", membreSommets, nom, classeVecteur4)
	}
	pas := f.types.taille(it.Type)
	if pas != tailleVecteur4 {
		return nil, fmt.Errorf("hinavmesh: %s fait %d octets, %d attendus", classeVecteur4, pas, tailleVecteur4)
	}
	brut, err := f.octets(it, pas)
	if err != nil {
		return nil, err
	}
	sommets := make([]Point, it.Compte)
	for i := range sommets {
		o := i * pas
		sommets[i] = Point{X: flottant(brut, o), Y: flottant(brut, o+4), Z: flottant(brut, o+8)}
	}
	return sommets, nil
}

// arete est une arete du maillage, reduite a ses deux sommets.
type arete struct{ A, B int32 }

// lireAretes lit le tableau `edges`.
func lireAretes(f *fichierTag, racine itemHavok) ([]arete, error) {
	it, err := f.tableau(racine, membreAretes)
	if err != nil {
		return nil, err
	}
	if nom := f.nomType(it.Type); nom != classeEdge {
		return nil, fmt.Errorf("hinavmesh: %s est un tableau de %s, %s attendu", membreAretes, nom, classeEdge)
	}
	pas := f.types.taille(it.Type)
	champA, err := f.champEntier(it.Type, membreSommetA, pas)
	if err != nil {
		return nil, err
	}
	champB, err := f.champEntier(it.Type, membreSommetB, pas)
	if err != nil {
		return nil, err
	}
	brut, err := f.octets(it, pas)
	if err != nil {
		return nil, err
	}
	aretes := make([]arete, it.Compte)
	for i := range aretes {
		o := i * pas
		aretes[i] = arete{A: int32(champA.lit(brut, o)), B: int32(champB.lit(brut, o))}
	}
	return aretes, nil
}

// lireFaces lit le tableau `faces` et reconstruit chaque polygone depuis les aretes.
func lireFaces(f *fichierTag, racine itemHavok, aretes []arete, nbSommets int) ([]Face, error) {
	it, err := f.tableau(racine, membreFaces)
	if err != nil {
		return nil, err
	}
	if nom := f.nomType(it.Type); nom != classeFace {
		return nil, fmt.Errorf("hinavmesh: %s est un tableau de %s, %s attendu", membreFaces, nom, classeFace)
	}
	pas := f.types.taille(it.Type)
	champDebut, err := f.champEntier(it.Type, membreDebutArc, pas)
	if err != nil {
		return nil, err
	}
	champNb, err := f.champEntier(it.Type, membreNbAretes, pas)
	if err != nil {
		return nil, err
	}
	brut, err := f.octets(it, pas)
	if err != nil {
		return nil, err
	}
	faces := make([]Face, 0, it.Compte)
	for i := 0; i < it.Compte; i++ {
		o := i * pas
		debut, nb := int(champDebut.lit(brut, o)), int(champNb.lit(brut, o))
		if nb < cotesFaceMinimum {
			return nil, fmt.Errorf("hinavmesh: face %d annonce %d cotes (minimum %d)", i, nb, cotesFaceMinimum)
		}
		if debut < 0 || debut+nb > len(aretes) {
			return nil, fmt.Errorf("hinavmesh: face %d vise les aretes %d..%d sur %d",
				i, debut, debut+nb, len(aretes))
		}
		contour := make([]int32, nb)
		for k := 0; k < nb; k++ {
			s := aretes[debut+k].A
			if s < 0 || int(s) >= nbSommets {
				return nil, fmt.Errorf("hinavmesh: face %d, arete %d: sommet %d sur %d", i, debut+k, s, nbSommets)
			}
			// Le contour doit BOUCLER : l'arete k finit ou l'arete k+1 commence. C'est
			// l'invariant qui distingue une lecture juste d'un decalage de champ, car un
			// mauvais offset donnerait des indices dans les bornes mais un contour ouvert.
			if suivante := aretes[debut+(k+1)%nb].A; aretes[debut+k].B != suivante {
				return nil, fmt.Errorf("hinavmesh: face %d, arete %d: contour ouvert (%d -> %d, %d attendu)",
					i, debut+k, aretes[debut+k].A, aretes[debut+k].B, suivante)
			}
			contour[k] = s
		}
		faces = append(faces, Face{Sommets: contour})
	}
	return faces, nil
}

// lireEntete lit les scalaires de l'objet racine : vecteur haut et rayon d'erosion.
func lireEntete(f *fichierTag, racine itemHavok, m *Maillage) error {
	haut, ok := f.types.membre(racine.Type, membreHaut)
	if !ok {
		return fmt.Errorf("hinavmesh: %s ne declare pas le membre %q", classeNavMesh, membreHaut)
	}
	// `up` est un hkaiUpVector, dont le seul membre est un hkVector4 a l'offset 0.
	interne, ok := f.types.membre(haut.Type, membreHaut)
	if !ok {
		return fmt.Errorf("hinavmesh: %s ne declare pas le membre %q", f.nomType(haut.Type), membreHaut)
	}
	base := racine.Offset + haut.Offset + interne.Offset
	if base < 0 || base+tailleVecteur4 > len(f.data) {
		return fmt.Errorf("hinavmesh: vecteur haut hors de DATA (offset %d)", base)
	}
	m.Haut = Point{X: flottant(f.data, base), Y: flottant(f.data, base+4), Z: flottant(f.data, base+8)}

	erosion, ok := f.types.membre(racine.Type, membreErosion)
	if !ok {
		return fmt.Errorf("hinavmesh: %s ne declare pas le membre %q", classeNavMesh, membreErosion)
	}
	pos := racine.Offset + erosion.Offset
	if pos < 0 || pos+4 > len(f.data) {
		return fmt.Errorf("hinavmesh: rayon d'erosion hors de DATA (offset %d)", pos)
	}
	m.RayonErosion = flottant(f.data, pos)
	return nil
}

// mesureEmprise calcule l'emprise sur les sommets REFERENCES par une face : un sommet
// orphelin ne doit pas elargir le fond de carte.
func mesureEmprise(m *Maillage) {
	inf := math.Inf(1)
	m.Min, m.Max = Point{inf, inf, inf}, Point{-inf, -inf, -inf}
	for _, f := range m.Faces {
		for _, i := range f.Sommets {
			p := m.Sommets[i]
			m.Min = Point{math.Min(m.Min.X, p.X), math.Min(m.Min.Y, p.Y), math.Min(m.Min.Z, p.Z)}
			m.Max = Point{math.Max(m.Max.X, p.X), math.Max(m.Max.Y, p.Y), math.Max(m.Max.Z, p.Z)}
		}
	}
}

// flottant lit un float32 petit-boutiste.
func flottant(b []byte, o int) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:])))
}

// champScalaire localise un membre entier dans un element de tableau.
type champScalaire struct{ offset, taille int }

// lit rend la valeur signee du champ pour l'element commencant a `base`.
func (c champScalaire) lit(b []byte, base int) int64 {
	switch c.taille {
	case 1:
		return int64(int8(b[base+c.offset]))
	case 2:
		return int64(int16(binary.LittleEndian.Uint16(b[base+c.offset:])))
	default:
		return int64(int32(binary.LittleEndian.Uint32(b[base+c.offset:])))
	}
}

// champEntier localise un membre entier et VERIFIE qu'il tient dans le pas du tableau.
func (f *fichierTag) champEntier(typ int, nom string, pas int) (champScalaire, error) {
	m, ok := f.types.membre(typ, nom)
	if !ok {
		return champScalaire{}, fmt.Errorf("hinavmesh: %s ne declare pas le membre %q", f.nomType(typ), nom)
	}
	taille := f.types.taille(m.Type)
	if taille != 1 && taille != 2 && taille != 4 {
		return champScalaire{}, fmt.Errorf("hinavmesh: le membre %q est un %s de %d octets, entier attendu",
			nom, f.nomType(m.Type), taille)
	}
	if m.Offset < 0 || m.Offset+taille > pas {
		return champScalaire{}, fmt.Errorf("hinavmesh: le membre %q est a +%d (%d octets) dans un pas de %d",
			nom, m.Offset, taille, pas)
	}
	return champScalaire{offset: m.Offset, taille: taille}, nil
}
