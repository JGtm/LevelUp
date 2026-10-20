package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// cmdRender rend un vehi (ou tous) en PNG teintable dans -out.
func cmdRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir (basenames, virgule)")
	variant := fs.String("variant", "any", "variante deploy: any|pc|ds")
	out := fs.String("out", ".", "dossier de sortie des PNG")
	idHex := fs.String("id", "", "GlobalID du vehi (hex) ; vide = tous les vehi")
	axe := fs.Int("axe", 2, "axe haut du modele : 0=X 1=Y 2=Z")
	cote := fs.Int("cote", 192, "longueur cible du plus grand cote, en px")
	cellmm := fs.Int("cellmm", 0, "millimetres/pixel FIXE (echelle commune pour composition 2D) ; 0 = ajuster a -cote")
	nom := fs.String("nom", "", "prefixe de nom de fichier (sinon vehi_<id>)")
	curate := fs.String("curate", "", "liste hexid:nom,... a rendre avec un nom propre (ignore ceux non resolus)")
	_ = fs.Parse(args)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
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
	if *curate != "" {
		return rendCurate(idx, *curate, *out, opts)
	}

	ids, err := idsARendre(idx, *idHex)
	if err != nil {
		return err
	}
	fmt.Printf("%d vehi a rendre\n", len(ids))
	ok := 0
	for _, id := range ids {
		tag, err := idx.Extract(id)
		if err != nil {
			fmt.Printf("  %#08x : extraction vehi: %v\n", id, err)
			continue
		}
		base := nomFichier(*nom, len(ids) > 1, id, tag)
		if err := rendUn(idx, tag, id, filepath.Join(*out, base+".png"), opts); err != nil {
			fmt.Printf("  %#08x : %v\n", id, err)
			continue
		}
		ok++
	}
	fmt.Printf("%d/%d rendus\n", ok, len(ids))
	return nil
}

// rendCurate rend une liste `hexid:nom,...` avec un nom de fichier propre, en ignorant sans
// bruit les vehi que l'index courant ne resout pas (ils seront rendus par l'autre passe de
// modules). Ecrit dans un fichier <nom>.png.
func rendCurate(idx *himap.ModuleIndex, spec, out string, opts himap.OptionsSprite) error {
	ok := 0
	paires := strings.Split(spec, ",")
	for _, p := range paires {
		i := strings.IndexByte(p, ':')
		if i <= 0 {
			continue
		}
		v, err := strconv.ParseUint(strings.TrimSpace(p[:i]), 0, 32)
		if err != nil {
			fmt.Printf("  spec %q illisible: %v\n", p, err)
			continue
		}
		id, nom := uint32(v), strings.TrimSpace(p[i+1:])
		tag, err := idx.Extract(id)
		if err != nil {
			continue
		}
		if err := rendUn(idx, tag, id, filepath.Join(out, nom+".png"), opts); err != nil {
			fmt.Printf("  %s (%#08x) non rendu ici: %v\n", nom, id, err)
			continue
		}
		ok++
	}
	fmt.Printf("%d/%d curate rendus\n", ok, len(paires))
	return nil
}

// nomFichier derive le nom de base : le prefixe force s'il est donne et qu'on ne rend qu'un
// vehi, sinon `<famille>_<id>` (la classification par noms de maillage).
func nomFichier(forcePrefix string, plusieurs bool, id uint32, tag []byte) string {
	if forcePrefix != "" && !plusieurs {
		return forcePrefix
	}
	fam, _ := classeVehicule(chainesASCII(tag, 4))
	return fmt.Sprintf("%s_%08x", fam, id)
}

// idsARendre rend soit l'unique id demande, soit tous les vehi indexes (tries).
func idsARendre(idx *himap.ModuleIndex, idHex string) ([]uint32, error) {
	if idHex != "" {
		v, err := strconv.ParseUint(idHex, 0, 32)
		if err != nil {
			return nil, fmt.Errorf("id %q illisible: %w", idHex, err)
		}
		return []uint32{uint32(v)}, nil
	}
	ids := idx.EntreesDuGroupe(himap.GroupeVehi)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// rendUn execute la chaine complete pour un vehi (tag deja extrait) et ecrit le PNG.
func rendUn(idx *himap.ModuleIndex, tag []byte, id uint32, chemin string, opts himap.OptionsSprite) error {
	mid, grp, ok := himap.RefModeleVehicule(context.Background(), idx, tag)
	if !ok {
		return fmt.Errorf("aucun modele resolu")
	}
	if grp != himap.GroupeMode {
		return fmt.Errorf("modele en %s (attendu mode) %#08x", grp, mid)
	}
	mtag, blob, err := idx.ExtractWithResources(mid)
	if err != nil {
		return fmt.Errorf("extraction mode %#08x: %w", mid, err)
	}
	asset, err := himap.NewRenderModelAsset(mtag, blob)
	if err != nil {
		return fmt.Errorf("render_model %#08x: %w", mid, err)
	}
	r, err := himap.RenduObjetIsole(asset, opts)
	if err != nil {
		return err
	}
	img := himap.SpriteObjetPNG(r, opts)
	f, err := os.Create(chemin)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	fmt.Printf("  %#08x -> %s  (%dx%d, mode %#08x, %d sections)\n",
		id, filepath.Base(chemin), r.NX, r.NY, mid, asset.MeshCount())
	return nil
}
