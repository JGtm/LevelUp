package himap

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// LE BALAYAGE — la regle vaut-elle sur les 37 cartes, ou seulement sur celle qui l'a vue naitre ?
//
// C'est le renversement de methode qui rend le chantier faisable : on calibre la REGLE une
// fois sur Cliffhanger, seule carte a posseder l'oracle fort des positions de joueur, et on
// valide la REGLE — pas ses parametres — sur toutes les autres avec l'oracle faible des ancres.
//
// Le critere est simple et ne demande aucun oeil humain : chaque ancre d'objectif doit trouver
// un sol sous elle, a `AncrageDecalageSol` pres. Une carte qui echoue signale une chaine
// cassee sur CETTE carte, pas un reglage a retoucher.
//
// Variables : BALAYAGE_CARTES (sous-ensemble, separe par des virgules), BALAYAGE_Z.

// toleranceAncre : ecart maximal admis entre une ancre ramenee au sol et le sol trouve.
// Vient de la dispersion mesuree de l'etalonnage (interquartile 0,46 m), doublee pour
// absorber les ancres legitimement posees en hauteur.
const toleranceAncre = 1.0

type bilanCarte struct {
	module   string
	ancres   int
	trouvees int
	sous1m   int
	ecart    float64
}

// TestBalayageDesCartes mesure la regle sur toutes les cartes du catalogue d'objectifs.
func TestBalayageDesCartes(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := chargeCatalogue(p)
	if err != nil {
		t.Fatal(err)
	}

	var modules []string
	vus := map[string]bool{}
	for _, e := range cat {
		if e.Module != "" && !vus[e.Module] {
			vus[e.Module] = true
			modules = append(modules, e.Module)
		}
	}
	sort.Strings(modules)
	if s := os.Getenv("BALAYAGE_CARTES"); s != "" {
		modules = strings.Split(s, ",")
	}
	t.Logf("%d modules distincts dans le catalogue", len(modules))

	var bilans []bilanCarte
	var absents []string
	for _, mod := range modules {
		chemin, ok := chercheModule(racine, mod)
		if !ok {
			absents = append(absents, mod)
			continue
		}
		b := jugeUneCarte(t, chemin, mod)
		bilans = append(bilans, b)
		t.Logf("%-34s ancres %3d · avec sol %3d · a moins d'1 m %3d · ecart median %+.2f m",
			b.module, b.ancres, b.trouvees, b.sous1m, b.ecart)
	}
	if len(absents) > 0 {
		t.Logf("%d modules sans fichier dans l'installation : %s",
			len(absents), strings.Join(absents, ", "))
	}

	total, bons := 0, 0
	for _, b := range bilans {
		total += b.ancres
		bons += b.sous1m
	}
	if total == 0 {
		t.Skip("aucune carte mesurable")
	}
	t.Logf("BILAN : %d cartes mesurees · %d/%d ancres sur un sol a moins d'1 m (%.1f %%)",
		len(bilans), bons, total, 100*float64(bons)/float64(total))
}

// chargeCatalogue rend les entrees du catalogue d'objectifs.
func chargeCatalogue(p string) ([]replay.MapObjectivesEntry, error) {
	cat, err := replay.LoadMapObjectives(p)
	if err != nil {
		return nil, err
	}
	out := make([]replay.MapObjectivesEntry, 0, len(cat.Maps))
	for _, e := range cat.Maps {
		out = append(out, e)
	}
	return out, nil
}

// chercheModule fait le pont entre le nom de module du CATALOGUE et le dossier de l'INSTALLATION.
//
// Ils ne coincident pas : le catalogue nomme « cliffhanger_ridgeline » ce que le jeu range sous
// « ridgeline ». Les deux moities du nom sont donc essayees, entiere puis par jeton, du plus
// specifique au plus general. Un module introuvable est SIGNALE, jamais suppose absent.
func chercheModule(racine, nom string) (string, bool) {
	dir, err := LevelsDir("pc")
	if err != nil {
		return "", false
	}
	dossiers, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	// Le jeu PREFIXE ses dossiers par le mode d'origine de la carte : `ctf_bazaar`,
	// `btb_highpower`, `sgh_interlock`, `va_behemoth`. Le catalogue, lui, nomme
	// « bazaar_map » ou « live_fire_sgh_interlock ». On enumere donc l'installation et on
	// apparie sur les jetons, plutot que de deviner un prefixe.
	generiques := map[string]bool{"map": true, "ranked": true, "": true, "-": true}
	var jetons []string
	for _, j := range strings.FieldsFunc(nom, func(r rune) bool { return r == '_' || r == '-' }) {
		if !generiques[j] {
			jetons = append(jetons, j)
		}
	}
	for _, d := range dossiers {
		if !d.IsDir() {
			continue
		}
		base := d.Name()
		p := filepath.Join(dir, base, base+"-rtx-new.module")
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if base == nom {
			return p, true
		}
		for _, j := range jetons {
			// Suffixe : `ctf_bazaar` porte `bazaar`. Egalite pour les dossiers sans prefixe.
			if base == j || strings.HasSuffix(base, "_"+j) {
				return p, true
			}
		}
	}
	_ = racine
	return "", false
}

// jugeUneCarte confronte les ancres d'une carte a sa geometrie reconstruite.
func jugeUneCarte(t *testing.T, cheminModule, module string) bilanCarte {
	t.Helper()
	b := bilanCarte{module: module}
	var ancres [][3]float64
	for _, e := range ancresDuModule(t, module) {
		for _, o := range e.Objectives {
			ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	b.ancres = len(ancres)
	if b.ancres == 0 {
		return b
	}
	vol := volumeDuModule(t, cheminModule, ancres)
	if vol == nil {
		return b
	}
	sol := vol.Floors(HeadroomBands)
	var ecarts []float64
	for _, a := range ancres {
		_, ecart, ok := sol.SolLePlusProche(a[0], a[1], a[2]-AncrageDecalageSol)
		if !ok {
			continue
		}
		b.trouvees++
		ecarts = append(ecarts, ecart)
		if ecart < toleranceAncre && ecart > -toleranceAncre {
			b.sous1m++
		}
	}
	// Une carte dont AUCUNE ancre ne trouve de sol est un cas reel, pas une impossibilite :
	// elle signale une chaine cassee sur cette carte. On la publie avec un ecart indefini
	// plutot que de faire tomber tout le balayage.
	if len(ecarts) == 0 {
		b.ecart = math.NaN()
		return b
	}
	b.ecart = mediane(ecarts)
	return b
}

// volumeDuModule rasterise une carte designee par le chemin de son module, avec les reglages
// retenus par l oracle : tous les modules, bornage a la boite de l instance.
//
// Le volume est BORNE au voisinage des ancres, en X/Y comme en Z. Ce n est pas une commodite :
// c est la meme regle qui cadre le dessin (cf. reference.go). Sans elle, une carte du balayage
// a reclame 11 324 x 12 308 x 240 cellules, soit 4 Go — l emprise du sbsp porte du decor
// lointain qui n interesse personne.
func volumeDuModule(t *testing.T, chemin string, ancres [][3]float64) *Volume {
	t.Helper()
	if len(ancres) == 0 {
		return nil
	}
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	zlo, zhi := math.Inf(1), math.Inf(-1)
	for _, a := range ancres {
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], a[k]-PorteeAncre)
			hi[k] = math.Max(hi[k], a[k]+PorteeAncre)
		}
		zlo = math.Min(zlo, a[2]-PorteeAncre)
		zhi = math.Max(zhi, a[2]+PorteeAncre)
	}
	return construitVolume(t, optionsCarte{
		CheminModule: chemin, ZMin: zlo, ZMax: zhi, BorneAABB: 0.5,
		EmpriseMin: &lo, EmpriseMax: &hi,
	})
}
