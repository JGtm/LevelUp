package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himap"
)

// vocabPermutations : noms candidats de permutation de vehicule. Un StringId de nom est le
// murmur3 (mapvar.LabelHash) de la chaine ; on hache ce vocabulaire et on apparie pour nommer
// les variantes. Sur le Warthog, seuls `default` et `unarmed` (= Razorback) resolvent ; les
// trois variantes d'arme gardent leur StringId brut (mappees a la forme, cf. rapport).
var vocabPermutations = []string{
	"default", "base", "standard", "unarmed", "no_turret", "none", "empty",
	"rocket", "rockethog", "rocket_turret", "gauss", "gauss_turret", "chaingun", "gun_turret",
	"razorback", "cargo", "troop", "gungoose", "twin_gun", "mongoose", "warthog",
}

// cmdVariantes marche les regions/permutations d'un `mode` (render_model) et rend UNE image par
// VARIANTE (nom de permutation, partage entre regions). Pour une variante, chaque region
// contribue les sections de sa permutation portant ce nom, sinon (absente ou heritante) celles
// de la permutation de BASE. Toutes les variantes sont rendues au MEME cadre (comparables),
// traits noirs inclus. C'est le levier qui distingue Rockethog/Gauss/chaingun/Razorback.
func cmdVariantes(args []string) error {
	fs := flag.NewFlagSet("variantes", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir")
	variant := fs.String("variant", "any", "variante deploy")
	idHex := fs.String("id", "", "GlobalID du mode (hex)")
	out := fs.String("out", ".", "dossier de sortie")
	cellmm := fs.Int("cellmm", 10, "mm/pixel")
	baseHex := fs.String("base", "0x42c9679f", "StringId de la permutation de BASE")
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
	v, _ := strconv.ParseUint(*idHex, 0, 32)
	baseU, _ := strconv.ParseUint(*baseHex, 0, 32)
	tag, blob, err := idx.ExtractWithResources(uint32(v))
	if err != nil {
		return err
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		return err
	}
	regions, err := himap.ModeRegions(tag)
	if err != nil {
		return err
	}
	return rendVariantes(asset, regions, uint32(baseU), float64(*cellmm)/1000.0, *out, uint32(v))
}

// rendVariantes rend chaque variante d'un modele au meme cadre.
func rendVariantes(asset *himap.RuntimeGeoAsset, regions []himap.Region, base uint32, cell float64, out string, modeID uint32) error {
	dico := construitDico()
	noms := nomsVariantes(regions)
	fmt.Printf("mode %#08x : %d sections, %d regions, %d variantes\n",
		modeID, asset.MeshCount(), len(regions), len(noms))

	full, err := himap.RenduObjetIsole(asset, himap.OptionsSprite{CellMetres: cell})
	if err != nil {
		return err
	}
	cmin := full.Min
	cmax := [2]float64{full.Min[0] + float64(full.NX)*full.Cell, full.Min[1] + float64(full.NY)*full.Cell}

	for _, nom := range noms {
		set := sectionsVariante(regions, nom, base)
		lbl := dico[nom]
		if lbl == "" {
			lbl = fmt.Sprintf("%08x", nom)
		}
		fmt.Printf("variante %#08x [%s] : %d sections\n", nom, lbl, len(set))
		o := himap.OptionsSprite{CellMetres: cell, SectionsChoisies: set, CadreMin: &cmin, CadreMax: &cmax}
		r, err := himap.RenduObjetIsole(asset, o)
		if err != nil {
			fmt.Printf("  rendu KO: %v\n", err)
			continue
		}
		ecrirePNG(filepath.Join(out, fmt.Sprintf("var_%s_%08x.png", lbl, nom)), himap.SpriteObjetPNG(r, o))
	}
	return nil
}

// construitDico hache le vocabulaire et rend un index StringId -> nom lisible.
func construitDico() map[uint32]string {
	d := map[uint32]string{}
	for _, s := range vocabPermutations {
		d[uint32(mapvar.LabelHash(s))] = s
	}
	return d
}

// nomsVariantes rend les noms de permutation distincts, dans l'ordre de premiere apparition.
func nomsVariantes(regions []himap.Region) []uint32 {
	vus := map[uint32]bool{}
	var out []uint32
	for _, r := range regions {
		for _, p := range r.Permutations {
			if !vus[p.Name] {
				vus[p.Name] = true
				out = append(out, p.Name)
			}
		}
	}
	return out
}

// sectionsVariante assemble les sections d'une variante : par region, la permutation portant ce
// nom, sinon la permutation de base. Une permutation dont SectionIndex < 0 n'est PAS vide : elle
// HERITE de la base (convention render_model) — sans quoi la variante « unarmed » (Razorback)
// perdait ses roues. On retombe donc sur la base des que la perm de la variante est absente OU
// heritante.
func sectionsVariante(regions []himap.Region, nom, base uint32) map[int]bool {
	set := map[int]bool{}
	for _, r := range regions {
		p := permParNom(r, nom)
		if p == nil || p.SectionIndex < 0 {
			p = permParNom(r, base)
		}
		if p == nil || p.SectionIndex < 0 {
			continue
		}
		for s := p.SectionIndex; s < p.SectionIndex+p.SectionCount; s++ {
			set[s] = true
		}
	}
	return set
}

func permParNom(r himap.Region, nom uint32) *himap.Permutation {
	for i := range r.Permutations {
		if r.Permutations[i].Name == nom {
			return &r.Permutations[i]
		}
	}
	return nil
}

func ecrirePNG(chemin string, img image.Image) {
	f, err := os.Create(chemin)
	if err != nil {
		fmt.Printf("  ecriture KO: %v\n", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Printf("  encode KO: %v\n", err)
	}
}
