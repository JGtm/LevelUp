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
//
// Les 2 pilotes du lot fonds par map_id (2026-08-13), même méthode level_id, unicité 1/1
// chacun, rejoués en continu par `TestPreuveLevelIDCartes` :
//
//	Starboard -747133697 (0xD377A4FF) -> fo03_space   (Forge : sa toile porte les bornes)
//	Dredge    2123870979 (0x7E97B303) -> fo06_deepsea (Forge : idem)
//
// La MASSE du même lot (33 cartes Forge, 2026-08-13) : même méthode, même sonde. Le
// level_id d'une carte Forge désigne son CANEVAS — les cartes d'un même canevas portent
// donc le MÊME level_id (mesuré, unicité 1/1 par carte via son fichier-lien) :
//
//	fo05_desert  1804860316 (0x6B93FB9C) : Banished Narrows, Cliffside, Domicile,
//	             Fortitude, Kaiketsu, Shiro, Sylvanus
//	fo08_wetland 88891201   (0x054C5F41) : Dynasty, High Ground, Isolation, Kiken'na,
//	             Nemesis, Origin, Perilous, Refuge, Smallhalla (+ Vagabond, témoin)
//	fo09_academy 1437677928 (0x55B13968) : Absolution, Command, Fortress, Houseki,
//	             Obituary, The Pit
//	fo11_blank   426470249  (0x196B6B69) : Critical Dewpoint, Curfew, Elevation,
//	             Empyrean, Goliath, Opulence, Salvation, Shogun, Solitude, Takamanohara
//	             (+ Corpo, témoin)
//	fo13_frost   -992358985 (0xC4D9CDB7) : Snowbound
var mapModule = map[string]string{
	"Absolution":        "fo09_academy",
	"Aquarius":          "ctf_aquarius",
	"Banished Narrows":  "fo05_desert",
	"Bazaar":            "ctf_bazaar",
	"Behemoth":          "va_behemoth",
	"Breaker":           "ctf_breaker",
	"Catalyst":          "catalyst",
	"Chasm":             "chasm",
	"Cliffhanger":       "ridgeline",
	"Cliffside":         "fo05_desert",
	"Command":           "fo09_academy",
	"Corpo":             "fo11_blank",
	"Critical Dewpoint": "fo11_blank",
	"Curfew":            "fo11_blank",
	"Deadlock":          "btb_drydock",
	"Domicile":          "fo05_desert",
	"Dredge":            "fo06_deepsea",
	"Dynasty":           "fo08_wetland",
	"Elevation":         "fo11_blank",
	"Empyrean":          "fo11_blank",
	"Forbidden":         "ctf_forbidden",
	"Forest":            "forest",
	"Fortitude":         "fo05_desert",
	"Fortress":          "fo09_academy",
	"Fragmentation":     "btb_fragmentation",
	"Goliath":           "fo11_blank",
	"High Ground":       "fo08_wetland",
	"Highpower":         "btb_highpower",
	"Houseki":           "fo09_academy",
	"Illusion":          "ctf_illusion",
	"Isolation":         "fo08_wetland",
	"Kaiketsu":          "fo05_desert",
	"Kiken'na":          "fo08_wetland",
	"Launch Site":       "va_launchsite",
	"Nemesis":           "fo08_wetland",
	"Oasis":             "btb_exiled",
	"Obituary":          "fo09_academy",
	"Opulence":          "fo11_blank",
	"Origin":            "fo08_wetland",
	"Perilous":          "fo08_wetland",
	"Prism":             "sgh_crystalcaves",
	"Recharge":          "sgh_blueprint",
	"Refuge":            "fo08_wetland",
	"Salvation":         "fo11_blank",
	"Scarr":             "btb_engine",
	"Shiro":             "fo05_desert",
	"Shogun":            "fo11_blank",
	"Smallhalla":        "fo08_wetland",
	"Snowbound":         "fo13_frost",
	"Solitude":          "fo11_blank",
	"Starboard":         "fo03_space",
	"Streets":           "sgh_streets",
	"Sylvanus":          "fo05_desert",
	"Takamanohara":      "fo11_blank",
	"The Pit":           "fo09_academy",
	"Vagabond":          "fo08_wetland",
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
