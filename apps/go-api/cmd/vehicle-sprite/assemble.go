package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// cmdAssemble fond un chassis parent (`mode`) et un ou plusieurs objets-enfants (tourelle,
// canon, arme — chacun un `mode`) dans UN SEUL rendu vue de dessus. C'est « l'assemblage du
// package » : le z-buffer partage donne l'occlusion correcte, les pieces co-reperees dans le
// repere local du vehicule. Une translation de marqueur optionnelle par enfant corrige un
// decalage si le marqueur d'attache a ete extrait du render_model parent.
//
//	-chassis=0xHEX                mode du chassis parent
//	-children=0xHEX[:tx,ty,tz],... modes des objets-enfants (translation locale optionnelle, m)
//	-out=DIR -name=NOM            sortie <NOM>.png
//	-cellmm / -cote / -axe        echelle et projection (defaut top-down, ajuste a -cote)
func cmdAssemble(args []string) error {
	fs := flag.NewFlagSet("assemble", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir")
	variant := fs.String("variant", "any", "variante deploy")
	chassisHex := fs.String("chassis", "", "GlobalID du mode chassis (hex)")
	childSpec := fs.String("children", "", "modes enfants: 0xHEX[:tx,ty,tz],...")
	out := fs.String("out", ".", "dossier de sortie")
	name := fs.String("name", "assemble", "nom de fichier (sans extension)")
	batch := fs.String("batch", "", "plusieurs assemblages en un chargement: nom=chassis[+enfant...];... (enfant: 0xHEX[:tx,ty,tz])")
	cellmm := fs.Int("cellmm", 0, "mm/pixel FIXE (0 = ajuster a -cote)")
	cote := fs.Int("cote", 256, "cote px du plus grand cote")
	axe := fs.Int("axe", 2, "axe haut: 0=X 1=Y(profil) 2=Z(dessus)")
	cadre := fs.Float64("cadre", 0, "demi-emprise FIXE en metres (canevas commun -H..+H) ; 0 = auto")
	_ = fs.Parse(args)

	chemins, err := cheminsModules(*variant, listeModules(*mods))
	if err != nil {
		return err
	}
	fmt.Printf("ouverture de %d modules...\n", len(chemins))
	idx, err := himap.NewModuleIndex(chemins...)
	if err != nil {
		return err
	}
	opts := himap.OptionsSprite{AxeHaut: himap.AxeHaut(*axe), CotePx: *cote, CellMetres: float64(*cellmm) / 1000.0}
	if *cadre > 0 {
		// Canevas FIXE partage : deux pieces rendues au meme -cadre et -cellmm tombent sur la
		// MEME grille de pixels (le repere local du vehicule au centre), donc composables en 2D
		// meme si chassis et tourelle vivent dans des modules distincts (RAM : jamais charges
		// ensemble). Le compose2d superpose ensuite les PNG.
		mn := [2]float64{-*cadre, -*cadre}
		mx := [2]float64{*cadre, *cadre}
		opts.CadreMin, opts.CadreMax = &mn, &mx
		opts.MargePx = 0
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	if *batch != "" {
		return rendBatch(idx, *batch, *out, opts)
	}
	parts, err := partsAssemblage(idx, *chassisHex, *childSpec)
	if err != nil {
		return err
	}
	return rendUnAssemblage(parts, *name, *out, opts)
}

// rendBatch rend plusieurs assemblages en une seule ouverture de modules. Chaque entree
// `nom=chassis[+enfant...]` (separateur `;`) est un assemblage independant ; une piece dont
// le mode n'est pas dans l'index courant est signalee et sautee (elle sera rendue par l'autre
// passe de modules).
func rendBatch(idx *himap.ModuleIndex, spec, out string, opts himap.OptionsSprite) error {
	for _, e := range strings.Split(spec, ";") {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		i := strings.IndexByte(e, '=')
		if i <= 0 {
			fmt.Printf("  entree %q illisible\n", e)
			continue
		}
		nom := strings.TrimSpace(e[:i])
		ids := strings.Split(e[i+1:], "+")
		chassis := ids[0]
		children := ""
		if len(ids) > 1 {
			children = strings.Join(ids[1:], ",")
		}
		parts, err := partsAssemblage(idx, chassis, children)
		if err != nil {
			fmt.Printf("  %s: %v\n", nom, err)
			continue
		}
		if err := rendUnAssemblage(parts, nom, out, opts); err != nil {
			fmt.Printf("  %s: %v\n", nom, err)
		}
	}
	return nil
}

// rendUnAssemblage rasterise un assemblage et ecrit son PNG.
func rendUnAssemblage(parts []himap.PartAssemblage, nom, out string, opts himap.OptionsSprite) error {
	r, err := himap.RenduAssemblage(parts, opts)
	if err != nil {
		return err
	}
	chemin := filepath.Join(out, nom+".png")
	f, err := os.Create(chemin)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, himap.SpriteObjetPNG(r, opts)); err != nil {
		return err
	}
	fmt.Printf("assemble %d pieces -> %s (%dx%d)\n", len(parts), chemin, r.NX, r.NY)
	return nil
}

// partsAssemblage construit la liste des composants : le chassis (translation nulle) puis
// chaque enfant avec sa translation optionnelle. Chaque `mode` est extrait avec son blob et
// decode en asset.
func partsAssemblage(idx *himap.ModuleIndex, chassisHex, childSpec string) ([]himap.PartAssemblage, error) {
	var parts []himap.PartAssemblage
	chassis, err := chargeAsset(idx, chassisHex)
	if err != nil {
		return nil, fmt.Errorf("chassis %s: %w", chassisHex, err)
	}
	parts = append(parts, himap.PartAssemblage{Asset: chassis})
	for _, s := range strings.Split(childSpec, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		idPart, tr := s, [3]float64{}
		if i := strings.IndexByte(s, ':'); i > 0 {
			idPart = s[:i]
			tr = parseTranslation(s[i+1:])
		}
		a, err := chargeAsset(idx, idPart)
		if err != nil {
			fmt.Printf("  enfant %s non charge: %v\n", idPart, err)
			continue
		}
		parts = append(parts, himap.PartAssemblage{Asset: a, Translation: tr})
	}
	return parts, nil
}

// chargeAsset extrait un `mode` et decode son render_model.
func chargeAsset(idx *himap.ModuleIndex, idHex string) (*himap.RuntimeGeoAsset, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(idHex), 0, 32)
	if err != nil {
		return nil, fmt.Errorf("id illisible: %w", err)
	}
	tag, blob, err := idx.ExtractWithResources(uint32(v))
	if err != nil {
		return nil, err
	}
	return himap.NewRenderModelAsset(tag, blob)
}

// parseTranslation lit `tx,ty,tz` (metres, repere local vehicule).
func parseTranslation(s string) [3]float64 {
	var t [3]float64
	for i, p := range strings.Split(s, ",") {
		if i >= 3 {
			break
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil {
			t[i] = v
		}
	}
	return t
}
