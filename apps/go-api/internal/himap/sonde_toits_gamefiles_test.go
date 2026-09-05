//go:build gamefiles

package himap

// SONDE DES TOITS — cherche le critere UNIVERSEL qui separe un PLAFOND (a remplacer par le sol
// qui est dessous) d'un ROCHER HAUT valide (l'identite de Cliffhanger, a ne pas toucher).
//
// Methode : cuire les cartes jugees au gate du 2026-08-10 — defectueuses (Illusion, Prism,
// Aquarius, et Chasm/Forbidden non satisfaisantes) ET validees (Cliffhanger, Catalyst,
// Behemoth « nickel » malgre son arche) — avec la voie de reference ARMEE mais NON appliquee,
// puis mesurer par pixel l'ecart de la surface haute a la reference et l'existence d'une
// surface praticable dessous. Le critere de production ne se decide que sur ces chiffres.
//
// PISTE REFUTEE, ne pas rejouer : l'ecart MEDIAN aux ancres ne predit pas le defaut
// (Behemoth -12,73 m est nickel, Forbidden -0,35 m est touche).
//
// Lancement CIBLE (jamais ./internal/himap/ nu) :
//
//	go test ./internal/himap/ -run TestSondeToits -timeout 45m -v

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// cartesSondeToits : cles = dossiers installes, dans l'ordre du verdict utilisateur.
var cartesSondeToits = []string{
	"ridgeline",        // VALIDEE — les rochers hauts sont son identite
	"catalyst",         // VALIDEE
	"va_behemoth",      // nickel malgre l'arche (ecart median -12,73 m)
	"ctf_illusion",     // DEFECTUEUSE : on ne voit que les toits
	"sgh_crystalcaves", // DEFECTUEUSE (Prism)
	"ctf_aquarius",     // DEFECTUEUSE
	"ctf_forbidden",    // partiellement touchee
	"chasm",            // non satisfaisante apres la bascule de tranche
}

// ancresParCleInstallee reconstruit les ancres de PRODUCTION : union dedupliquee des entrees du
// catalogue qui designent le meme dossier installe (meme regle que cmd/mapfond-build).
func ancresParCleInstallee(t *testing.T) map[string]struct {
	chemin string
	ancres [][3]float64
} {
	t.Helper()
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := replay.LoadMapObjectives(p)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(cat.Maps))
	for id := range cat.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := map[string]struct {
		chemin string
		ancres [][3]float64
	}{}
	connues := map[string]map[[3]float64]bool{}
	for _, id := range ids {
		e := cat.Maps[id]
		chemin, ok := ChercheModuleInstalle(e.Module)
		if !ok {
			continue
		}
		cle := filepath.Base(filepath.Dir(chemin))
		c := out[cle]
		c.chemin = chemin
		if connues[cle] == nil {
			connues[cle] = map[[3]float64]bool{}
		}
		for _, o := range e.Objectives {
			a := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
			if !connues[cle][a] {
				connues[cle][a] = true
				c.ancres = append(c.ancres, a)
			}
		}
		out[cle] = c
	}
	return out
}

func TestSondeToits(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	cibles := ancresParCleInstallee(t)
	for _, cle := range cartesSondeToits {
		c, ok := cibles[cle]
		if !ok {
			t.Errorf("%s : absente du catalogue resolu", cle)
			continue
		}
		cle := cle
		t.Run(cle, func(t *testing.T) {
			mesureToits(t, racine, c.chemin, c.ancres)
		})
	}
}

func mesureToits(t *testing.T, racine, chemin string, ancres [][3]float64) {
	t.Helper()
	s := NewSurfaceReference(ancres)
	r := CadreSurAncres(ancres)
	zJeu := MedianeZ(ancres) - AncrageDecalageSol
	r.Tranche(TrancheDeJeu(zJeu))
	r.ArmeReference(s)
	var b BilanCuisson
	if err := peupleDepuisModule(context.Background(), r, &b, OptionsCuisson{
		RacineDeploy: racine, CheminModule: chemin, Ancres: ancres,
	}); err != nil {
		t.Skipf("cuisson impossible : %v", err)
	}

	seuilsS := []float64{4, 6, 8, 10, 12, 15, 20}
	seuilsT := []float64{2, 3, 4, 6}
	compte := make([][]int, len(seuilsS))
	for i := range compte {
		compte[i] = make([]int, len(seuilsT))
	}
	nMat, nAlt, nMatProche, nAltProche := 0, 0, 0, 0
	var hautsAlt []float64 // zHaut-ref des cellules qui ont une alternative praticable
	for j := 0; j < r.NY; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := 0; i < r.NX; i++ {
			zHaut, zProche, ref, ok := r.CandidatsReference(i, j)
			if !ok {
				continue
			}
			nMat++
			proche := s.DistanceAncre(r.Min[0]+(float64(i)+0.5)*r.Cell, y) <= PorteeAncre
			if proche {
				nMatProche++
			}
			alternative := zHaut-zProche >= 2 && math.Abs(zProche-ref) <= 3
			if alternative {
				nAlt++
				hautsAlt = append(hautsAlt, zHaut-ref)
				if proche {
					nAltProche++
				}
			}
			for si, S := range seuilsS {
				if zHaut-ref < S {
					continue
				}
				for ti, T := range seuilsT {
					if math.Abs(zProche-ref) <= T && zHaut-zProche >= 2 {
						compte[si][ti]++
					}
				}
			}
		}
	}
	t.Logf("matiere %d px · alternative praticable (gap>=2, |zProche-ref|<=3) : %d px (%.1f %%) · "+
		"pres des ancres : %d/%d px (%.1f %%)",
		nMat, nAlt, 100*float64(nAlt)/float64(max(nMat, 1)),
		nAltProche, nMatProche, 100*float64(nAltProche)/float64(max(nMatProche, 1)))
	if len(hautsAlt) > 0 {
		t.Logf("zHaut-ref des cellules a alternative : q10 %.1f · q25 %.1f · mediane %.1f · q75 %.1f · q90 %.1f",
			Centile(hautsAlt, 0.10), Centile(hautsAlt, 0.25), Centile(hautsAlt, 0.5),
			Centile(hautsAlt, 0.75), Centile(hautsAlt, 0.90))
	}
	for si, S := range seuilsS {
		var ligne strings.Builder
		fmt.Fprintf(&ligne, "S=%4.1f :", S)
		for ti, T := range seuilsT {
			fmt.Fprintf(&ligne, "  T<=%.0f : %5.1f %%", T, 100*float64(compte[si][ti])/float64(max(nMat, 1)))
		}
		t.Log(ligne.String())
	}
	mesureAncresToits(t, r, ancres)
}

// mesureAncresToits journalise, ancre par ancre, la surface HAUTE dessinee a son aplomb, en
// metres au-dessus de son sol. Un plafond se lit ici directement.
func mesureAncresToits(t *testing.T, r *Rendu, ancres [][3]float64) {
	t.Helper()
	var dessus []float64
	couvertes := 0
	for _, a := range ancres {
		i := int((a[0] - r.Min[0]) / r.Cell)
		j := int((a[1] - r.Min[1]) / r.Cell)
		if z, ok := r.Altitude(i, j); ok {
			d := z - (a[2] - AncrageDecalageSol)
			dessus = append(dessus, d)
			if d >= 4 {
				couvertes++
			}
		}
	}
	if len(dessus) > 0 {
		t.Logf("ANCRES : %d/%d sous une surface a >=4 m au-dessus de leur sol · surplomb q50 %.1f m · q90 %.1f m",
			couvertes, len(dessus), Centile(dessus, 0.5), Centile(dessus, 0.90))
	}
}
