// cmd/mapcallouts-build — produit le CATALOGUE DES CALLOUTS (zones nommées officielles)
// des cartes intégrées, et l'écrit dans data/titles/{slug}/reference/map_callouts.json
// (PathResolver.MapCalloutsPath).
//
// LA CHAÎNE, portée des scripts de recherche callouts.py / callouts_all.py :
//
//	deploy/ds/levels/multi/<carte>/<carte>-rtx-new.module
//	  -> tag levl (internal/himap.ReadModuleCallouts : offsets documentés au champ près)
//	  -> jointure des libellés FR/EN par (carte, volume_index) sur callouts_i18n.csv
//	     (copie VERSIONNÉE — data/titles/{slug}/reference/callouts_i18n.csv — du CSV de
//	     recherche : 816/816 résolus par string_id, on ne re-extrait PAS uslg)
//	  -> classement grandes/fines par recouvrement (classify.go, étalonné sur le POC)
//	  -> Ridgeline : polygones remplacés par le dump DÉCOUPÉ versionné (decoupe.go)
//
// INVARIANTS MESURÉS, et le build ÉCHOUE s'ils bougent : 22 cartes avec zones, 816 zones,
// libellé résolu 816/816, string_id du CSV égal à celui du tag. Un catalogue partiel ne
// s'écrit pas.
//
// Usage : CGO_ENABLED=1 go run ./cmd/mapcallouts-build [--levels DIR] [--title slug]
//
//	[--i18n FILE] [--decoupe FILE] [--out FILE]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

// deployVariant : la variante sur laquelle la table levl a été établie et mesurée
// (callouts_all.py). La racine est résolue par himap.LevelsDir, jamais écrite en dur.
const deployVariant = "ds"

// Invariants du corpus (mesure callouts_all.py + CSV, 2026-08).
const (
	attenduCartes = 22
	attenduZones  = 816
)

func main() {
	levels := flag.String("levels", "", "racine des dossiers de cartes (.module) ; vide = installation détectée")
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	i18nPath := flag.String("i18n", "", "CSV des libellés (défaut : reference/callouts_i18n.csv du titre)")
	decoupePath := flag.String("decoupe", "", "dump découpé de Ridgeline (défaut : .ai/V7.5/dumps/callout_zones_ridgeline_clipped.json)")
	out := flag.String("out", "", "fichier de sortie (défaut : PathResolver.MapCalloutsPath)")
	flag.Parse()

	root, err := title.FindRepoRoot()
	if err != nil {
		slog.Error("racine repo", "err", err)
		os.Exit(1)
	}
	res := title.NewPathResolver(root)
	if *levels == "" {
		dir, lerr := himap.LevelsDir(deployVariant)
		if lerr != nil {
			slog.Error("installation du jeu", "err", lerr)
			os.Exit(1)
		}
		*levels = dir
		slog.Info("installation détectée", "levels", dir)
	}
	if *i18nPath == "" {
		*i18nPath = filepath.Join(filepath.Dir(res.MapCalloutsPath(*titleSlug)), "callouts_i18n.csv")
	}
	if *decoupePath == "" {
		*decoupePath = filepath.Join(root, ".ai", "V7.5", "dumps", "callout_zones_ridgeline_clipped.json")
	}
	outPath := *out
	if outPath == "" {
		outPath = res.MapCalloutsPath(*titleSlug)
	}

	labels, err := chargeLibelles(*i18nPath)
	if err != nil {
		slog.Error("libellés", "err", err)
		os.Exit(1)
	}
	decoupeModule, decoupeZones, err := chargeDecoupe(*decoupePath)
	if err != nil {
		slog.Error("dump découpé", "err", err)
		os.Exit(1)
	}

	cat := replay.MapCalloutsCatalog{
		SchemaVersion: replay.MapCalloutsSchemaVersion,
		TitleSlug:     *titleSlug,
		Source: "tag levl de " + *levels + " + callouts_i18n.csv (libellés uslg figés)" +
			" ; Ridgeline : polygones du dump découpé versionné" +
			" ; autres cartes à fond publié : découpe sur le masque praticable (internal/mapdecoupe)",
		Maps: map[string]replay.MapCalloutsEntry{},
		Brut: map[string][]replay.CalloutBrutZone{},
	}
	entries, err := os.ReadDir(*levels)
	if err != nil {
		slog.Error("dossier des cartes", "err", err, "levels", *levels)
		os.Exit(1)
	}
	totalZones := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		module := e.Name()
		modPath := filepath.Join(*levels, module, module+"-rtx-new.module")
		if _, statErr := os.Stat(modPath); statErr != nil {
			continue
		}
		cs, err := himap.ReadModuleCallouts(modPath)
		if err != nil {
			slog.Error("lecture des callouts", "err", err, "module", module)
			os.Exit(1)
		}
		if len(cs) == 0 {
			// Canevas Forge et assimilés : ZÉRO zone PAR CONSTRUCTION — pas d'entrée.
			continue
		}
		entry, err := construitEntree(module, cs, labels)
		if err != nil {
			slog.Error("jointure des libellés", "err", err, "module", module)
			os.Exit(1)
		}
		if module == decoupeModule {
			if err := appliqueDecoupe(&entry, decoupeZones); err != nil {
				slog.Error("dump découpé", "err", err, "module", module)
				os.Exit(1)
			}
		} else if err := decoupeCarte(&entry, res, *titleSlug, cat.Brut); err != nil {
			slog.Error("découpage sur le fond publié", "err", err, "module", module)
			os.Exit(1)
		}
		cat.Maps[module] = entry
		totalZones += len(entry.Zones)
		grandes := 0
		for _, z := range entry.Zones {
			if z.Big {
				grandes++
			}
		}
		slog.Info("carte lue", "module", module, "zones", len(entry.Zones),
			"grandes", grandes, "provenance", entry.Provenance)
	}
	if len(cat.Maps) != attenduCartes || totalZones != attenduZones {
		slog.Error("corpus inattendu — rien écrit",
			"cartes", len(cat.Maps), "zones", totalZones,
			"attendu", fmt.Sprintf("%d cartes / %d zones", attenduCartes, attenduZones))
		os.Exit(1)
	}

	blob, err := json.MarshalIndent(cat, "", " ")
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
	slog.Info("catalogue écrit", "path", outPath, "cartes", len(cat.Maps), "zones", totalZones)
}

// construitEntree joint les libellés et classe les zones d'une carte.
func construitEntree(module string, cs []himap.Callout, labels libelles) (replay.MapCalloutsEntry, error) {
	var shaped []shapedPoly
	for _, c := range cs {
		if c.HasShape && len(c.Polygon) >= 3 {
			shaped = append(shaped, shapedPoly{vi: c.VolumeIndex, poly: c.Polygon})
		}
	}
	big := classifyBig(shaped)

	entry := replay.MapCalloutsEntry{
		Module:     module,
		Provenance: replay.CalloutsProvenanceBrut,
		Zones:      make([]replay.CalloutZone, 0, len(cs)),
	}
	for _, c := range cs {
		lbl, err := labels.resolve(module, c)
		if err != nil {
			return replay.MapCalloutsEntry{}, err
		}
		z := replay.CalloutZone{
			VolumeIndex: c.VolumeIndex,
			Name:        c.Name,
			EN:          lbl.en,
			FR:          lbl.fr,
			X:           c.Pos[0],
			Y:           c.Pos[1],
			Z:           c.Pos[2],
			ZBottom:     c.ZBottom(),
			ZTop:        c.ZTop(),
			Big:         big[c.VolumeIndex],
		}
		if c.HasShape {
			z.Polygon = c.Polygon
		}
		entry.Zones = append(entry.Zones, z)
	}
	sort.Slice(entry.Zones, func(i, j int) bool {
		return entry.Zones[i].VolumeIndex < entry.Zones[j].VolumeIndex
	})
	return entry, nil
}
