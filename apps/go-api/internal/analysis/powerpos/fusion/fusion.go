package fusion

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/analysis/tactical"
)

// NoeudGeo est ce que la fusion lit d'un noeud geometrique : son adresse de cellule, son
// altitude et son score (`geo.NoeudMesure.Score`, reglage geometrique fige).
type NoeudGeo struct {
	Col, Lig int
	Z        float64
	Score    float64
}

// CelluleGeo est une cellule du sol derive : le meilleur de ses noeuds.
type CelluleGeo struct {
	Col, Lig int
	// Score : le MAXIMUM des scores des noeuds de la cellule (l'etage qu'on tient).
	Score float64
	// ZMin, ZMax : l'emprise verticale des noeuds de la cellule.
	ZMin, ZMax float64
	NbNoeuds   int
}

// AgregeCellules ramene les noeuds a une cellule chacun, triees par adresse.
func AgregeCellules(noeuds []NoeudGeo) []CelluleGeo {
	parAdresse := map[tactical.Cellule]*CelluleGeo{}
	for _, n := range noeuds {
		adr := tactical.Cellule{Col: n.Col, Lig: n.Lig}
		c := parAdresse[adr]
		if c == nil {
			c = &CelluleGeo{Col: n.Col, Lig: n.Lig, Score: n.Score, ZMin: n.Z, ZMax: n.Z}
			parAdresse[adr] = c
		}
		c.NbNoeuds++
		c.Score = math.Max(c.Score, n.Score)
		c.ZMin, c.ZMax = math.Min(c.ZMin, n.Z), math.Max(c.ZMax, n.Z)
	}
	out := make([]CelluleGeo, 0, len(parAdresse))
	for _, c := range parAdresse {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		return out[i].Lig < out[j].Lig
	})
	return out
}

// Reglage porte les poids de la fusion, les valeurs d'absence et la selection.
type Reglage struct {
	// PoidsGeo, PoidsEmp : les poids des deux voies (somme 1 par convention).
	PoidsGeo float64
	PoidsEmp float64
	// EmpAbsent : emp_norm d'une cellule du sol derive qui n'est pas scorable.
	EmpAbsent float64
	// GeoAbsent : geo_norm d'une cellule scorable hors du sol derive.
	GeoAbsent float64
	// Selection : la regle v2 (`powerpos.Selectionne`) avec ses quantiles propres. Seuls les
	// champs de selection sont lus ; les poids empiriques de cette struct ne servent pas.
	Selection powerpos.Reglage
}

// Cellule est une cellule fusionnee.
type Cellule struct {
	// Cellule : l'adresse, le centre et les comptes empiriques (zero pour une cellule que
	// seule la geometrie connait).
	powerpos.Cellule
	GeoNorm, EmpNorm float64
	GeoPresent       bool
	EmpPresent       bool
	// ZMin, ZMax : l'emprise verticale geometrique (zero sans geometrie).
	ZMin, ZMax float64
	Score      float64
}

// Fusionne rend le score fusionne de chaque cellule connue d'au moins une voie, triees
// par score decroissant puis par adresse. `emp` est la sortie de `powerpos.Score` (les
// cellules scorables, deja notees par le reglage empirique fige).
func Fusionne(g tactical.Grille, geoCellules []CelluleGeo, emp []powerpos.CelluleScoree, r Reglage) []Cellule {
	geoNorm := normaliseGeo(geoCellules)
	empNorm := normaliseEmp(emp)
	parAdresse := map[tactical.Cellule]*Cellule{}
	for i, c := range geoCellules {
		adr := tactical.Cellule{Col: c.Col, Lig: c.Lig}
		x, y := g.Centre(adr)
		parAdresse[adr] = &Cellule{
			Cellule:    powerpos.Cellule{Col: c.Col, Lig: c.Lig, CentreX: x, CentreY: y},
			GeoNorm:    geoNorm[i],
			GeoPresent: true,
			EmpNorm:    r.EmpAbsent,
			ZMin:       c.ZMin, ZMax: c.ZMax,
		}
	}
	for i, e := range emp {
		adr := tactical.Cellule{Col: e.Col, Lig: e.Lig}
		c := parAdresse[adr]
		if c == nil {
			c = &Cellule{Cellule: e.Cellule, GeoNorm: r.GeoAbsent}
			parAdresse[adr] = c
		}
		c.Cellule = e.Cellule
		c.EmpNorm, c.EmpPresent = empNorm[i], true
	}
	out := make([]Cellule, 0, len(parAdresse))
	for _, c := range parAdresse {
		c.Score = r.PoidsGeo*c.GeoNorm + r.PoidsEmp*c.EmpNorm
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		return out[i].Lig < out[j].Lig
	})
	return out
}

// normaliseGeo ramene les scores geometriques dans [0, 1] par carte (p5..p95).
func normaliseGeo(cellules []CelluleGeo) []float64 {
	vals := make([]float64, len(cellules))
	for i, c := range cellules {
		vals[i] = c.Score
	}
	return geo.Normalise(vals)
}

// normaliseEmp ramene les scores empiriques dans [0, 1] par carte (p5..p95).
func normaliseEmp(emp []powerpos.CelluleScoree) []float64 {
	vals := make([]float64, len(emp))
	for i, c := range emp {
		vals[i] = c.Score
	}
	return geo.Normalise(vals)
}

// Selectionne applique la selection v2 au score fusionne et rend les positions.
func Selectionne(g tactical.Grille, cellules []Cellule, r Reglage) []powerpos.Position {
	return powerpos.Selectionne(g, CommeScorees(cellules), r.Selection)
}

// CommeScorees projette les cellules fusionnees sur le type que la selection v2 lit : le
// score fusionne prend la place du score empirique, les comptes suivent la cellule.
func CommeScorees(cellules []Cellule) []powerpos.CelluleScoree {
	out := make([]powerpos.CelluleScoree, 0, len(cellules))
	for _, c := range cellules {
		out = append(out, powerpos.CelluleScoree{Cellule: c.Cellule, Score: c.Score})
	}
	return out
}

// Bilan resume, pour une position, d'ou vient son score.
type Bilan struct {
	GeoMoyen, EmpMoyen float64
	// SansEmp, SansGeo : cellules mesurees de la position que l'une des voies ignorait.
	SansEmp, SansGeo int
	ZMin, ZMax       float64
}

// BilanDe rend le bilan d'une position sur les cellules fusionnees.
func BilanDe(p powerpos.Position, cellules []Cellule) Bilan {
	index := make(map[tactical.Cellule]Cellule, len(cellules))
	for _, c := range cellules {
		index[tactical.Cellule{Col: c.Col, Lig: c.Lig}] = c
	}
	var b Bilan
	n := 0
	b.ZMin, b.ZMax = math.Inf(1), math.Inf(-1)
	for _, adr := range p.Cellules {
		c, ok := index[adr]
		if !ok {
			continue
		}
		n++
		b.GeoMoyen += c.GeoNorm
		b.EmpMoyen += c.EmpNorm
		if !c.EmpPresent {
			b.SansEmp++
		}
		if !c.GeoPresent {
			b.SansGeo++
		} else {
			b.ZMin, b.ZMax = math.Min(b.ZMin, c.ZMin), math.Max(b.ZMax, c.ZMax)
		}
	}
	if n > 0 {
		b.GeoMoyen, b.EmpMoyen = b.GeoMoyen/float64(n), b.EmpMoyen/float64(n)
	}
	if math.IsInf(b.ZMin, 0) {
		b.ZMin, b.ZMax = 0, 0
	}
	return b
}
