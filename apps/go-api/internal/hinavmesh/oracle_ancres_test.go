package hinavmesh

// oracle_ancres_test.go — L'ORACLE : les ancres d'objectif tombent-elles sur le maillage ?
//
// Les ancres d'objectif d'une carte (drapeau, colline, zone, bombe...) sont du TERRAIN
// JOUE par definition : le jeu y pose un objet que des joueurs vont chercher a pied. Si
// le maillage decode est le bon, chaque ancre doit tomber DANS un polygone, a une altitude
// proche de celle du polygone. Une emprise qui n'enveloppe pas les ancres, ou des ecarts
// d'altitude metriques, signeraient un decodage faux — c'est la seule preuve qui vaille,
// et elle ne coute rien a rejouer sur une nouvelle carte.
//
// Les deux cartes temoins sont dans des reperes TOTALEMENT differents (Isolation autour de
// X -53..7 / Z 112..124, Kiken'na autour de X -187..-155 / Z 172..179) : aucun seuil ni
// aucune borne ne peut etre code en dur et passer sur les deux.

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"levelup/go-api/internal/testutil"
)

// ancre est une ancre d'objectif du referentiel des cartes.
type ancre struct {
	Role string `json:"role"`
	Pos  struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"pos"`
}

// referentielCartes est la forme utile de data/titles/halo_infinite/reference/map_objectives.json.
type referentielCartes struct {
	Maps map[string]struct {
		PublicName string  `json:"public_name"`
		Objectives []ancre `json:"objectives"`
	} `json:"maps"`
}

// carteTemoin est une carte dont le navmesh est verse en testdata.
type carteTemoin struct {
	assetID string
	nom     string
	// rolesHorsMaillage nomme les ancres tolerees hors de tout polygone. On les pin par
	// ROLE et pas par compte : si une AUTRE ancre sortait pendant que celle-ci rentre, un
	// simple compte laisserait passer la regression.
	rolesHorsMaillage []string
	raisonHors        string
	// ecartAltitudeMax borne l'ecart vertical entre une ancre et le polygone qui la porte.
	ecartAltitudeMax float64
}

var cartesTemoins = []carteTemoin{
	{
		assetID:           "01af558d-53ab-4f05-ba68-92d805fc6260",
		nom:               "Isolation",
		rolesHorsMaillage: []string{"assault_bomb"},
		raisonHors: "assault_bomb est posee en bordure : le navmesh s'arrete avant le mur, " +
			"l'ancre non. Ecart MESURE au bord du maillage : 2,03 m — et non 0,96 m comme le " +
			"donnait l'enquete du 2026-08-27, qui mesurait l'ecart a la BOITE ENGLOBANTE en Y " +
			"et non au bord reel",
		ecartAltitudeMax: 5.0,
	},
	{
		assetID:          "df7dbf08-b8de-4ade-9d7f-1947128c9ae4",
		nom:              "Kikenna",
		ecartAltitudeMax: 5.0,
	},
}

// TestOracleAncresDansLeMaillage est LA preuve du decodeur : sur chaque carte temoin,
// les ancres d'objectif tombent dans un polygone du maillage, a une altitude coherente.
func TestOracleAncresDansLeMaillage(t *testing.T) {
	ref := chargeReferentiel(t)
	for _, c := range cartesTemoins {
		t.Run(c.nom, func(t *testing.T) {
			m := decodeTemoin(t, c.assetID)
			carte, ok := ref.Maps[c.assetID]
			if !ok {
				t.Fatalf("%s absente du referentiel des cartes", c.assetID)
			}
			if len(carte.Objectives) == 0 {
				t.Fatalf("%s ne porte aucune ancre d'objectif", c.nom)
			}

			var hors []string
			var ecarts []float64
			for _, a := range carte.Objectives {
				dz, trouve := ecartAuMaillage(m, a.Pos.X, a.Pos.Y, a.Pos.Z)
				if !trouve {
					hors = append(hors, a.Role)
					continue
				}
				ecarts = append(ecarts, math.Abs(dz))
				if math.Abs(dz) > c.ecartAltitudeMax {
					t.Errorf("ancre %s a (%.2f, %.2f, %.2f) : ecart d'altitude %.2f m au polygone porteur, "+
						"maximum tolere %.2f m", a.Role, a.Pos.X, a.Pos.Y, a.Pos.Z, dz, c.ecartAltitudeMax)
				}
			}
			sort.Strings(hors)
			attendus := append([]string(nil), c.rolesHorsMaillage...)
			sort.Strings(attendus)
			if !slices.Equal(hors, attendus) {
				t.Errorf("ancres hors de tout polygone : %v ; attendues : %v (%s)",
					hors, attendus, c.raisonHors)
			}
			// Une exception toleree doit rester MESUREE : si l'ancre s'eloignait du bord,
			// ce ne serait plus un effet de bordure mais un trou dans le maillage.
			for _, a := range carte.Objectives {
				if !slices.Contains(hors, a.Role) {
					continue
				}
				d := distanceAuBord(m, a.Pos.X, a.Pos.Y)
				if d > toleranceBordure {
					t.Errorf("ancre %s a %.2f m du bord du maillage : au-dela de %.2f m ce n'est "+
						"plus un effet de bordure mais un trou", a.Role, d, toleranceBordure)
				}
				t.Logf("exception toleree : %s a %.2f m du bord du maillage", a.Role, d)
			}
			sort.Float64s(ecarts)
			t.Logf("%s : %d/%d ancres dans un polygone ; ecart d'altitude median %.3f m ; "+
				"emprise X[%.2f..%.2f] Y[%.2f..%.2f] Z[%.2f..%.2f] ; %d faces, %d sommets, %.0f m2",
				c.nom, len(ecarts), len(carte.Objectives), mediane(ecarts),
				m.Min.X, m.Max.X, m.Min.Y, m.Max.Y, m.Min.Z, m.Max.Z,
				len(m.Faces), len(m.Sommets), m.AireAuSol())
		})
	}
}

// TestOracleEmpriseEnveloppeLesAncres verifie la condition la plus grossiere et la plus
// dirimante : l'emprise XY du maillage contient les ancres. Un decodage qui rendrait des
// coordonnees aberrantes echouerait ici avant tout test fin.
func TestOracleEmpriseEnveloppeLesAncres(t *testing.T) {
	ref := chargeReferentiel(t)
	for _, c := range cartesTemoins {
		t.Run(c.nom, func(t *testing.T) {
			m := decodeTemoin(t, c.assetID)
			// Marge : le maillage s'arrete avant les murs, une ancre de bordure peut le
			// deborder de quelques decimetres.
			const marge = 1.5
			for _, a := range ref.Maps[c.assetID].Objectives {
				if a.Pos.X < m.Min.X-marge || a.Pos.X > m.Max.X+marge ||
					a.Pos.Y < m.Min.Y-marge || a.Pos.Y > m.Max.Y+marge {
					t.Errorf("ancre %s a (%.2f, %.2f) hors de l'emprise X[%.2f..%.2f] Y[%.2f..%.2f]",
						a.Role, a.Pos.X, a.Pos.Y, m.Min.X, m.Max.X, m.Min.Y, m.Max.Y)
				}
			}
		})
	}
}

// TestOracleMaillageSousLesCoques est le point qui motive tout le chantier : sur Isolation,
// la geometrie de rendu monte a Z 160 et les coques empilees a Z 136..160 peignent 82,7 %
// de l'image vue de dessus. Le maillage de navigation, lui, s'arrete bien en dessous : il
// n'y a aucune coque a peler.
func TestOracleMaillageSousLesCoques(t *testing.T) {
	m := decodeTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	// Altitude de la premiere couche de coques mesuree sur Isolation.
	const basDesCoques = 136.0
	if m.Max.Z >= basDesCoques {
		t.Errorf("le maillage monte a Z %.2f, soit dans la bande des coques (>= %.0f)", m.Max.Z, basDesCoques)
	}
	for _, s := range m.Sommets {
		if s.Z >= basDesCoques {
			t.Fatalf("un sommet est a Z %.2f, dans la bande des coques (>= %.0f)", s.Z, basDesCoques)
		}
	}
	t.Logf("Isolation : maillage entre Z %.2f et %.2f, soit %.1f m sous la premiere couche de coques",
		m.Min.Z, m.Max.Z, basDesCoques-m.Max.Z)
}

// ecartAuMaillage rend l'ecart d'altitude entre un point et le polygone qui le contient en
// XY, en gardant le polygone le plus proche verticalement. Le second retour est faux si
// aucun polygone ne contient le point.
func ecartAuMaillage(m *Maillage, x, y, z float64) (float64, bool) {
	meilleur, trouve := math.Inf(1), false
	for _, f := range m.Faces {
		contour := m.Contour(f)
		if !dansPolygone(x, y, contour) {
			continue
		}
		dz := z - altitudePlan(x, y, contour)
		if !trouve || math.Abs(dz) < math.Abs(meilleur) {
			meilleur, trouve = dz, true
		}
	}
	return meilleur, trouve
}

// dansPolygone teste l'appartenance d'un point au polygone, projete dans le plan XY,
// par la regle du nombre de croisements.
func dansPolygone(x, y float64, contour []Point) bool {
	dedans := false
	for i, j := 0, len(contour)-1; i < len(contour); j, i = i, i+1 {
		pi, pj := contour[i], contour[j]
		if (pi.Y > y) != (pj.Y > y) && x < (pj.X-pi.X)*(y-pi.Y)/(pj.Y-pi.Y)+pi.X {
			dedans = !dedans
		}
	}
	return dedans
}

// altitudePlan rend l'altitude du plan du polygone a l'aplomb de (x, y).
func altitudePlan(x, y float64, contour []Point) float64 {
	a, b, c := contour[0], contour[1], contour[2]
	ux, uy, uz := b.X-a.X, b.Y-a.Y, b.Z-a.Z
	vx, vy, vz := c.X-a.X, c.Y-a.Y, c.Z-a.Z
	nx, ny, nz := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
	if math.Abs(nz) < 1e-9 {
		return a.Z
	}
	return a.Z - (nx*(x-a.X)+ny*(y-a.Y))/nz
}

func mediane(v []float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	return v[len(v)/2]
}

// decodeTemoin decode le navmesh verse en testdata pour un asset.
func decodeTemoin(t *testing.T, assetID string) *Maillage {
	t.Helper()
	chemin := filepath.Join("testdata", assetID+".navmesh.blob")
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, versionne
	if err != nil {
		t.Fatalf("lecture de %s: %v", chemin, err)
	}
	m, err := Decode(blob)
	if err != nil {
		t.Fatalf("Decode(%s): %v", chemin, err)
	}
	return m
}

// chargeReferentiel lit le catalogue VERSIONNE des ancres d'objectif.
func chargeReferentiel(t *testing.T) referentielCartes {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot: %v", err)
	}
	chemin := filepath.Join(racine, "data", "titles", "halo_infinite", "reference", "map_objectives.json")
	brut, err := os.ReadFile(chemin) //nolint:gosec // fichier versionne, lecture seule
	if err != nil {
		t.Fatalf("lecture de %s: %v", chemin, err)
	}
	var ref referentielCartes
	if err := json.Unmarshal(brut, &ref); err != nil {
		t.Fatalf("%s illisible: %v", chemin, err)
	}
	return ref
}

// toleranceBordure borne l'ecart accepte pour une ancre hors maillage : au-dela, ce n'est
// plus le retrait du navmesh le long des murs mais un defaut de couverture. Cale sur la
// seule exception connue (assault_bomb d'Isolation, 2,03 m mesures) avec une marge courte :
// c'est un plafond a surveiller, pas un seuil a relever.
const toleranceBordure = 2.5

// distanceAuBord rend la distance XY d'un point au bord le plus proche du maillage.
func distanceAuBord(m *Maillage, x, y float64) float64 {
	meilleure := math.Inf(1)
	for _, f := range m.Faces {
		contour := m.Contour(f)
		for i := range contour {
			a, b := contour[i], contour[(i+1)%len(contour)]
			if d := distanceAuSegment(x, y, a, b); d < meilleure {
				meilleure = d
			}
		}
	}
	return meilleure
}

// distanceAuSegment rend la distance XY d'un point au segment [a, b].
func distanceAuSegment(x, y float64, a, b Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	long := dx*dx + dy*dy
	t := 0.0
	if long > 0 {
		t = math.Max(0, math.Min(1, ((x-a.X)*dx+(y-a.Y)*dy)/long))
	}
	return math.Hypot(x-(a.X+t*dx), y-(a.Y+t*dy))
}
