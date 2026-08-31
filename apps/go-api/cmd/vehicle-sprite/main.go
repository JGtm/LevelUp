// Command vehicle-sprite — rend, depuis les `.module` du jeu, une vue de dessus teintable
// par vehicule pilotable (lot V4, chantier « vehicules et tourelles »).
//
// CHAINE : `vehi` -> `hlmt` -> `mode` (render_model) -> triangles (himap.NewRenderModelAsset)
// -> rasterizer top-down local (himap.RenduObjetIsole) -> PNG silhouette+alpha
// (himap.SpriteObjetPNG). Tout le decodage vit dans internal/himap ; cette commande orchestre.
//
// Offline-pur (ooz/Kraken GPLv3) : jamais linke dans le serveur, seulement ici.
//
// Sous-commandes :
//
//	scan   : enumere les `vehi`, resout la chaine, dumpe les chaines ASCII (identification).
//	render : rend un `vehi` (ou tous) en PNG teintable dans un dossier de sortie.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/himap"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "scan":
		err = cmdScan(os.Args[2:])
	case "render":
		err = cmdRender(os.Args[2:])
	case "variantes":
		err = cmdVariantes(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: vehicle-sprite <scan|render> [flags]")
	fmt.Fprintln(os.Stderr, "  scan   -modules=a,b   liste les vehi et leur chaine")
	fmt.Fprintln(os.Stderr, "  render -out=DIR -id=0x..  rend un vehi (ou tous si -id absent)")
}

// modulesParDefaut : les modules qui portent les objets multijoueur, plus les globaux ou
// vivent des maillons partages (hlmt/mode references hors du module du vehicule). L'ordre
// donne la priorite : le module du contenu MP d'abord.
var modulesParDefaut = []string{
	"multiplayer-rtx-new.module",
	"multiplayer_r1-rtx-new.module",
	"multiplayer_r3-rtx-new.module",
	"globals-rtx-new.module",
	"common-rtx-new.module",
}

// cheminsModules resout une liste de specs de modules vers leurs chemins absolus. Chaque spec
// est soit `basename` (sous <variant>/globals), soit `autreVariant:basename` (pour indexer une
// autre variante — typiquement `pc:globals-rtx-new.module` qui porte la geometrie de rendu que
// `any` n'a pas). Un module absent est signale, jamais silencieux.
func cheminsModules(variant string, specs []string) ([]string, error) {
	root, err := himap.DeployRoot()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, s := range specs {
		v, base := variant, s
		if i := strings.IndexByte(s, ':'); i > 0 {
			v, base = s[:i], s[i+1:]
		}
		p := filepath.Join(root, v, "globals", base)
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("module %s introuvable (%s)", s, p)
		}
		out = append(out, p)
	}
	return out, nil
}

// listeModules decoupe un flag -modules=a,b ou rend la liste par defaut.
func listeModules(flagVal string) []string {
	if strings.TrimSpace(flagVal) == "" {
		return modulesParDefaut
	}
	var out []string
	for _, s := range strings.Split(flagVal, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
