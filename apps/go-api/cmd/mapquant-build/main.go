// cmd/mapquant-build — produit le catalogue des BORNES DE QUANTIFICATION PAR CARTE à
// partir des .module du jeu installé, et l'écrit dans
// data/titles/{slug}/reference/map_quant_bounds.json (PathResolver).
//
// Le film ne porte que des indices de quantum ; les bornes (AABB du BSP principal,
// `world bounds x/y/z` du tag sbsp) ne vivent que dans le module de la carte. Sans elles,
// aucune coordonnée monde n'est produite (refus explicite côté décodeur).
//
// Le lien nom de carte affiché -> dossier de module est déclaré ici, EXPLICITEMENT, et
// n'est retenu que lorsqu'il est établi hors de toute mesure (identité du nom, ou preuve
// externe citée). Les cartes dont le module n'est pas établi sont ABSENTES du catalogue :
// l'API préfère refuser que produire une coordonnée fausse.
//
// Usage : CGO_ENABLED=1 go run ./cmd/mapquant-build [--levels DIR] [--title slug] [--out FILE]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

// deployVariant : les bornes monde du BSP se lisent dans le build serveur dédié. La racine
// de l'installation est résolue par himap.LevelsDir (emplacements connus ou
// LEVELUP_HALO_DEPLOY), jamais écrite en dur ici.
const deployVariant = "ds"

// mapModule associe le nom de carte affiché (celui de match_registry.map_name) au dossier
// du module. Chaque entrée porte la RAISON pour laquelle le lien est tenu pour établi.
//
// Toutes les entrées ci-dessous sauf Cliffhanger sont l'identité du nom (le module porte
// le nom de la carte, éventuellement préfixé par sa famille : ctf_, sgh_, va_, btb_).
// Cliffhanger : le module `ridgeline` est le SEUL dont les bornes reproduisent la capture
// live de DAT_14462cbe0[0] sur le film 000d5950 (x[-41.103,72.109] y[-56.607,57.212]
// z[-84.371,53.180], accord à 2,5e-4 unité près) — preuve externe, pas un appariement de
// largeurs.
//
// Vagabond : le module `fo08_wetland` est établi par le `level_id` du .mvar, pas par le
// nom. `vagabond_fo08_wetland.mvar` porte level_id 88891201 (0x054C5F41) ; balayé sur les
// modules de `deploy/any` + `deploy/ds`, il rend EXACTEMENT UNE occurrence,
// `multi/fo08_wetland/fo08_wetland-rtx-new.module` à +0x28, groupe `levl`. Témoin de la
// méthode : le level_id de Catalyst (−1044063363) rend de même une seule occurrence,
// `multi/catalyst`. Preuve externe à toute mesure de largeur (plan maître §J0.2, 2026-07-31).
// Vagabond est une carte Forge : `fo08_wetland` est sa TOILE, et c'est bien la toile qui
// porte les bornes de déquantification.
//
// Les 6 entrées du 2026-08-13 (Corpo, Deadlock, Oasis, Prism, Recharge, Scarr) sont
// établies par la MÊME méthode level_id, désormais REJOUÉE en continu par le test
// gamefiles `TestPreuveLevelIDCartes` (internal/himap/sonde_levelid_gamefiles_test.go) :
// level_id lu dans `<carte>_map.mvar` (variante par défaut, nom sans module), corroboré par
// le fichier-lien `<carte>_<module>.mvar`, unicité exigée sur les 64 modules de
// `any/levels` + `ds/levels`. Level_id mesurés (2026-08-13, unicité 1/1 chacun) :
//
//	Corpo     426470249  (0x196B6B69) -> fo11_blank (Forge : sa toile porte les bornes)
//	Deadlock  -785503777 (0xD12E29DF) -> btb_drydock
//	Oasis     -611378397 (0xDB8F1B23) -> btb_exiled
//	Prism     2068765158 (0x7B4ED9E6) -> sgh_crystalcaves
//	Recharge  -687782121 (0xD7014717) -> sgh_blueprint
//	Scarr     799711266  (0x2FAAA022) -> btb_engine
//
// NON CATALOGUÉE malgré un module PROUVÉ : Live Fire (level_id 1253388187 / 0x4AB52F9B
// -> `sgh_interlock`, unicité 1/1, rejouée par le même test). Son module ds ne porte
// AUCUN tag sbsp (mesuré le 2026-08-13 : `himap.ErrAucunTagSbsp` — même exception que la
// cuisson des fonds, handoff cartes §1 ter) : la source déclarée des bornes n'existe pas
// pour cette carte, et une entrée sans bornes vraies serait une coordonnée devinée.
var mapModule = map[string]string{
	"Aquarius":      "ctf_aquarius",
	"Bazaar":        "ctf_bazaar",
	"Behemoth":      "va_behemoth",
	"Breaker":       "ctf_breaker",
	"Catalyst":      "catalyst",
	"Chasm":         "chasm",
	"Cliffhanger":   "ridgeline",
	"Corpo":         "fo11_blank",
	"Deadlock":      "btb_drydock",
	"Forbidden":     "ctf_forbidden",
	"Forest":        "forest",
	"Fragmentation": "btb_fragmentation",
	"Highpower":     "btb_highpower",
	"Illusion":      "ctf_illusion",
	"Launch Site":   "va_launchsite",
	"Oasis":         "btb_exiled",
	"Prism":         "sgh_crystalcaves",
	"Recharge":      "sgh_blueprint",
	"Scarr":         "btb_engine",
	"Streets":       "sgh_streets",
	"Vagabond":      "fo08_wetland",
}

func main() {
	levels := flag.String("levels", "", "racine des dossiers de cartes (.module) ; vide = installation détectée")
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	out := flag.String("out", "", "fichier de sortie (défaut : PathResolver.MapQuantBoundsPath)")
	flag.Parse()

	if *levels == "" {
		dir, err := himap.LevelsDir(deployVariant)
		if err != nil {
			slog.Error("installation du jeu", "err", err)
			os.Exit(1)
		}
		*levels = dir
		slog.Info("installation détectée", "levels", dir)
	}

	outPath := *out
	if outPath == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			slog.Error("racine repo", "err", err)
			os.Exit(1)
		}
		outPath = title.NewPathResolver(root).MapQuantBoundsPath(*titleSlug)
	}

	cat := filmdec.MapQuantCatalog{
		SchemaVersion: filmdec.MapQuantSchemaVersion,
		Source:        "world bounds x/y/z du tag sbsp principal, lus dans " + *levels,
		Maps:          map[string]filmdec.MapQuantEntry{},
	}
	names := make([]string, 0, len(mapModule))
	for n := range mapModule {
		names = append(names, n)
	}
	sort.Strings(names)
	missing := 0
	for _, name := range names {
		mod := mapModule[name]
		mods, _ := filepath.Glob(filepath.Join(*levels, mod, "*.module"))
		if len(mods) == 0 {
			slog.Error("module absent", "carte", name, "module", mod)
			missing++
			continue
		}
		bsps, err := himap.ReadModuleBSPBounds(mods[0])
		if err != nil {
			slog.Error("lecture des bornes", "err", err, "carte", name, "module", mod)
			missing++
			continue
		}
		main := bsps[0] // BSP principal = le plus gros tag sbsp
		if !main.Bounds.Valid() {
			slog.Error("AABB dégénérée", "carte", name, "module", mod)
			missing++
			continue
		}
		e := filmdec.MapQuantEntry{Module: mod}
		w := main.Bounds.AxisWidths()
		for ax := 0; ax < 3; ax++ {
			e.Min[ax] = float32(main.Bounds.Min[ax])
			e.Max[ax] = float32(main.Bounds.Max[ax])
			e.AxisWidths[ax] = uint(w[ax])
		}
		cat.Maps[filmdec.NormalizeMapName(name)] = e
		slog.Info("bornes lues", "carte", name, "module", mod,
			"W", fmt.Sprintf("%d/%d/%d", w[0], w[1], w[2]),
			"extent", fmt.Sprintf("%.3f/%.3f/%.3f", main.Bounds.Extent(0), main.Bounds.Extent(1), main.Bounds.Extent(2)))
	}
	if missing > 0 {
		slog.Error("catalogue incomplet — rien écrit", "manquantes", missing)
		os.Exit(1)
	}
	blob, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		slog.Error("sérialisation", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		slog.Error("création du répertoire", "err", err, "path", outPath)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, append(blob, '\n'), 0o644); err != nil {
		slog.Error("écriture", "err", err, "path", outPath)
		os.Exit(1)
	}
	slog.Info("catalogue écrit", "path", outPath, "cartes", len(cat.Maps))
}
