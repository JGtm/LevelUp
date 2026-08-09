package himap

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// L'ORACLE — les positions de joueur REELLEMENT observees jugent la carte reconstruite.
//
// Pourquoi ce fichier existe. Toutes les statistiques internes essayees pour departager deux
// lectures de la geometrie (ecart aux bornes, longueur d'arete, saturation de plage) se sont
// revelees soit biaisees, soit incapables de separer — l'histoire du chantier est une
// collection de temoins tautologiques. La seule mesure qui ne puisse pas etre tautologique
// vient de DEHORS de la geometrie : un joueur qui a couru quelque part avait forcement du sol
// sous les pieds. 29 221 positions decodees du film servent d'arbitre.
//
// Ce que l'oracle tranche, en une seule passe :
//   - la carte a-t-elle des TROUS la ou l'on a joue (colonne sans aucun sol) ;
//   - la deQUANTIFICATION (u16 brut contre i16 decale) ;
//   - le FILTRE DE MODULE (module de la carte seul contre tous les modules).
//
// Variables : ORACLE_REPLAY (chemin du document de rejeu), ORACLE_Z (zmin,zmax),
// ORACLE_PNG (carte des positions manquees), ORACLE_CAS (sous-ensemble des cas a jouer).

// pointRejeu : une position de joueur du document de rejeu.
type pointRejeu struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type pisteRejeu struct {
	Points []pointRejeu `json:"points"`
}

type documentRejeu struct {
	Tracks []pisteRejeu `json:"tracks"`
}

// cheminRejeu cherche le document de rejeu : variable dediee, puis racine de depot declaree,
// puis remontee depuis le repertoire de test.
func cheminRejeu() (string, error) {
	if p := os.Getenv("ORACLE_REPLAY"); p != "" {
		return p, nil
	}
	const rel = "data/cache/replays/halo_infinite/000d5950.json"
	var candidats []string
	if r := os.Getenv("LEVELUP_REPO_ROOT"); r != "" {
		candidats = append(candidats, filepath.Join(r, rel))
	}
	if wd, err := os.Getwd(); err == nil {
		for d := wd; ; {
			candidats = append(candidats, filepath.Join(d, rel))
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	for _, c := range candidats {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("document de rejeu introuvable (poser ORACLE_REPLAY ou LEVELUP_REPO_ROOT)")
}

func chargePositions(t *testing.T) []pointRejeu {
	t.Helper()
	p, err := cheminRejeu()
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(p) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Skip(err)
	}
	var doc documentRejeu
	if err := json.Unmarshal(brut, &doc); err != nil {
		t.Fatal(err)
	}
	var out []pointRejeu
	for _, piste := range doc.Tracks {
		out = append(out, piste.Points...)
	}
	if len(out) == 0 {
		t.Fatalf("%s ne porte aucune position", p)
	}
	return out
}

// verdict : ce que l'oracle mesure sur une reconstruction.
type verdict struct {
	nom               string
	positions         int
	horsEmprise       int
	sansSol           int // colonne vide : un TROU dans la carte
	sous25, sous50    int
	sous100, sous200  int
	ecartMedianAbsolu float64
	// ecartMedianSigne separe DEUX diagnostics que la valeur absolue confond : une carte
	// fausse (ecart centre sur zero, disperse) et un repere decale (ecart serre autour
	// d'une valeur non nulle — z n'est alors pas l'altitude des pieds).
	ecartMedianSigne float64
}

func (v verdict) part(n int) float64 { return 100 * float64(n) / float64(v.positions) }

// jugeCarte confronte un volume de praticabilite aux positions observees.
func jugeCarte(nom string, sol *Volume, pos []pointRejeu) (verdict, []pointRejeu) {
	v := verdict{nom: nom, positions: len(pos)}
	ecarts := make([]float64, 0, len(pos))
	signes := make([]float64, 0, len(pos))
	manques := make([]pointRejeu, 0, 64)
	for _, p := range pos {
		_, ecart, ok := sol.SolLePlusProche(p.X, p.Y, p.Z)
		if !ok {
			if _, dans := sol.Colonne(p.X, p.Y); !dans {
				v.horsEmprise++
			} else {
				v.sansSol++
			}
			manques = append(manques, p)
			continue
		}
		d := math.Abs(ecart)
		ecarts = append(ecarts, d)
		signes = append(signes, ecart)
		for seuil, compteur := range map[float64]*int{0.25: &v.sous25, 0.50: &v.sous50, 1.00: &v.sous100, 2.00: &v.sous200} {
			if d < seuil {
				*compteur++
			}
		}
	}
	sort.Float64s(ecarts)
	sort.Float64s(signes)
	if len(ecarts) > 0 {
		v.ecartMedianAbsolu = ecarts[len(ecarts)/2]
		v.ecartMedianSigne = signes[len(signes)/2]
	}
	return v, manques
}

// TestOracleCarteCliffhanger joue les positions observees contre plusieurs reconstructions.
func TestOracleCarteCliffhanger(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	pos := chargePositions(t)
	t.Logf("oracle : %d positions de joueur", len(pos))

	zmin, zmax := VolumeZMin, VolumeZMax
	if s := os.Getenv("ORACLE_Z"); s != "" {
		fmt.Sscanf(s, "%f,%f", &zmin, &zmax)
	}

	// Un volume coute ~35 s a rasteriser, une praticabilite quelques dixiemes : on construit
	// chaque volume UNE fois et on lui applique tous les degagements a comparer.
	cas := []struct {
		nom        string
		carteSeule bool
		dq         Dequant
		borne      float64
		plafond    int
	}{
		{"tous modules · LOD 40k", false, DequantBrut, 0.5, 0},
		{"tous modules · LOD plein", false, DequantBrut, 0.5, 100000000},
		{"carte seule · LOD 40k", true, DequantBrut, 0.5, 0},
		{"carte seule · LOD plein", true, DequantBrut, 0.5, 100000000},
	}
	degagements := []int{2, 4}
	filtre := os.Getenv("ORACLE_CAS")
	reference := os.Getenv("ORACLE_REF")
	if reference == "" {
		reference = "tous modules · LOD plein/2"
	}

	var verdicts []verdict
	for _, c := range cas {
		if filtre != "" && !strings.Contains(c.nom, filtre) {
			continue
		}
		vol := construitVolume(t, optionsCarte{ZMin: zmin, ZMax: zmax, CarteSeule: c.carteSeule, Dequant: c.dq, BorneAABB: c.borne, Plafond: c.plafond})
		for _, d := range degagements {
			nom := fmt.Sprintf("%s/%d", c.nom, d)
			sol := vol.Floors(d)
			v, manques := jugeCarte(nom, sol, pos)
			verdicts = append(verdicts, v)
			if p := os.Getenv("ORACLE_PNG"); p != "" && nom == reference {
				ecritCarteDesManques(t, p, sol, pos, manques)
			}
		}
	}

	t.Log("cas                        | hors empr | SANS SOL | <25cm | <50cm | <1m   | <2m   | |med| | med signe")
	for _, v := range verdicts {
		t.Logf("%-26s | %8.1f%% | %7.1f%% | %4.1f%% | %4.1f%% | %4.1f%% | %4.1f%% | %.3f | %+.3f m",
			v.nom, v.part(v.horsEmprise), v.part(v.sansSol), v.part(v.sous25),
			v.part(v.sous50), v.part(v.sous100), v.part(v.sous200),
			v.ecartMedianAbsolu, v.ecartMedianSigne)
	}
}

// ecritCarteDesManques dessine la carte praticable et, en rouge, les positions qu'elle rate.
// C'est le diagnostic qui LOCALISE un trou (un pont absent, une aile non cartographiee) la
// ou un pourcentage ne fait que le compter.
func ecritCarteDesManques(t *testing.T, sortie string, sol *Volume, pos, manques []pointRejeu) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, sol.NX, sol.NY))
	for j := 0; j < sol.NY; j++ {
		for i := 0; i < sol.NX; i++ {
			c := color.RGBA{247, 248, 250, 255}
			for iz := 0; iz < sol.NZ; iz++ {
				if sol.Get(iz, j, i) {
					f := float64(iz) / float64(sol.NZ-1)
					c = color.RGBA{uint8(210 - 150*f), uint8(216 - 140*f), uint8(224 - 110*f), 255}
				}
			}
			img.Set(i, sol.NY-1-j, c)
		}
	}
	marque := func(p pointRejeu, c color.RGBA) {
		i := int((p.X - sol.Min[0]) / sol.Cell)
		j := int((p.Y - sol.Min[1]) / sol.Cell)
		for dj := -1; dj <= 1; dj++ {
			for di := -1; di <= 1; di++ {
				img.Set(i+di, sol.NY-1-(j+dj), c)
			}
		}
	}
	for _, p := range pos {
		marque(p, color.RGBA{70, 130, 90, 255})
	}
	for _, p := range manques {
		marque(p, color.RGBA{200, 40, 40, 255})
	}
	f, err := os.Create(sortie)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	fmt.Println("PNG des manques ecrit:", sortie)
}
