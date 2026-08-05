// cmd/mapstruct-build — fige la STRUCTURE d'une carte (sols, plateformes, rampes, murs)
// dans data/titles/{slug}/reference/map_structure/{module}.json (PathResolver), à partir du
// bloc `instanced geometry instances` du tag sbsp de son .module.
//
// POURQUOI : le fond de carte du rejeu venait des props de la variante Forge — 382 objets,
// 0,25 m² d'emprise médiane, 3,4 % de la zone de jeu couverts. Ce sont des accessoires. La
// vraie structure est dans le module du jeu.
//
// CE QUE CE BINAIRE PUBLIE : l'AABB MONDE de chaque instance, lue telle quelle (@0x7C de
// l'enregistrement). Aucune transformation n'est appliquée — le lien instance -> maillage
// n'étant pas résolu, on ne publie PAS de forme, seulement des boîtes englobantes.
//
// DÉPÔT DE JEU : utiliser deploy/pc/. Le build serveur dédié (deploy/ds/) est dépouillé du
// rendu : son bloc `instanced geometry instances` est VIDE (mesuré : data-block cible -1),
// il ne garde que `instanced physics instances`. Le compagnon `_hd1` de deploy/pc/ n'est
// pas un problème depuis le correctif dataBase de internal/himodule.
//
// Usage :
//
//	CGO_ENABLED=1 go run ./cmd/mapstruct-build [--levels DIR] [--maps "Cliffhanger,Streets"]
//	                                           [--title slug] [--out-dir DIR]
//
// --maps vide = toutes les cartes du catalogue map_quant_bounds.json (attention au poids :
// ~0,5 Mo par carte dans un dossier versionné).
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
)

// defaultLevelsDir : arborescence des cartes multijoueur d'une installation Steam.
// deploy/pc/ et non deploy/ds/ (cf. en-tête : le build dédié n'a pas la géométrie de rendu).
const defaultLevelsDir = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi`

func main() {
	levels := flag.String("levels", defaultLevelsDir, "racine des dossiers de cartes (.module)")
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	// Défaut = les cartes dont la structure est MESURÉE exploitable (couverture des bornes
	// monde = 100 %). Catalyst, Forest et Aquarius plafonnent à 40-49 % : leur structure vit
	// dans le maillage de rendu non instancié (bloc `meshes` @0x240, non lu). Les figer
	// donnerait un fond de carte troué qui se lirait comme une carte, d'où leur absence du
	// dépôt tant que ce bloc n'est pas décodé — les produire reste possible en les nommant.
	maps := flag.String("maps", "Cliffhanger,Streets", "noms de cartes séparés par des virgules ; vide = tout le catalogue")
	outDir := flag.String("out-dir", "", "répertoire de sortie (défaut : PathResolver.MapStructurePath)")
	flag.Parse()

	root, err := title.FindRepoRoot()
	if err != nil {
		slog.Error("racine repo", "err", err)
		os.Exit(1)
	}
	res := title.NewPathResolver(root)
	cat, err := filmdec.LoadMapQuantCatalog(res.MapQuantBoundsPath(*titleSlug))
	if err != nil {
		slog.Error("catalogue de bornes", "err", err)
		os.Exit(1)
	}
	// Le lien nom affiché -> module est déclaré UNE seule fois (cmd/mapquant-build) et lu
	// ici depuis le catalogue : deux tables divergeraient tôt ou tard.
	targets, err := selectModules(cat, *maps)
	if err != nil {
		slog.Error("sélection des cartes", "err", err)
		os.Exit(1)
	}

	failed := 0
	for _, t := range targets {
		modPath, err := findModule(*levels, t.module)
		if err != nil {
			slog.Error("module absent", "err", err, "module", t.module, "cartes", t.names)
			failed++
			continue
		}
		ms, err := extractStructure(modPath, t.module, t.names)
		if err != nil {
			slog.Error("extraction", "err", err, "module", t.module)
			failed++
			continue
		}
		outPath := res.MapStructurePath(*titleSlug, t.module)
		if *outDir != "" {
			outPath = filepath.Join(*outDir, t.module+".json")
		}
		size, err := writeStructure(outPath, ms)
		if err != nil {
			slog.Error("écriture", "err", err, "path", outPath)
			failed++
			continue
		}
		slog.Info("structure figée", "module", t.module, "cartes", t.names,
			"emprises", ms.Stats.Count, "aireMediane", ms.Stats.AreaMedian,
			"aireMax", ms.Stats.AreaMax,
			"couverture", fmt.Sprintf("%.1f %%", ms.Stats.CoveragePct),
			"couvertureSansGrandesBoites", fmt.Sprintf("%.1f %%", ms.Stats.CoveragePctCapped),
			"exclues", ms.Excluded, "octets", size, "path", outPath)
	}
	if failed > 0 {
		slog.Error("cartes en échec", "n", failed)
		os.Exit(1)
	}
}

// target est une carte à traiter : un module et les noms affichés qui le partagent.
type target struct {
	module string
	names  []string
}

// selectModules résout la liste de cartes demandée en modules, via le catalogue de bornes.
func selectModules(cat *filmdec.MapQuantCatalog, list string) ([]target, error) {
	byModule := map[string][]string{}
	for name, e := range cat.Maps {
		byModule[e.Module] = append(byModule[e.Module], name)
	}
	if strings.TrimSpace(list) == "" {
		out := make([]target, 0, len(byModule))
		for mod, names := range byModule {
			out = append(out, target{module: mod, names: sorted(names)})
		}
		return sortedTargets(out), nil
	}
	seen := map[string]bool{}
	var out []target
	for _, raw := range strings.Split(list, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		e, err := cat.Lookup(name)
		if err != nil {
			return nil, err
		}
		if seen[e.Module] {
			continue
		}
		seen[e.Module] = true
		out = append(out, target{module: e.Module, names: sorted(byModule[e.Module])})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune carte sélectionnée")
	}
	return out, nil
}

// findModule localise le .module d'un dossier de carte.
func findModule(levelsDir, module string) (string, error) {
	mods, err := filepath.Glob(filepath.Join(levelsDir, module, "*.module"))
	if err != nil {
		return "", err
	}
	if len(mods) == 0 {
		return "", fmt.Errorf("aucun .module dans %s", filepath.Join(levelsDir, module))
	}
	return mods[0], nil
}
