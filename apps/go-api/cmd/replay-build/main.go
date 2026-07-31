// cmd/replay-build — génère l'artefact de rejeu 2D d'un match à partir des SEULS CHUNKS
// DU FILM (décodage offline des positions bipeds, zéro entrée Cheat Engine), plus un fond
// de carte optionnel (props Forge). Écrit data/cache/replays/{title}/{matchId}.json
// (résolu par PathResolver).
//
// Le décodage lourd est dans internal/analysis/filmdec, l'assemblage dans
// internal/analysis/replay. Ce binaire est l'assembleur HORS LIGNE ; l'API sert
// l'artefact tel quel.
//
// Usage:
//
//	replay-build --map <nom de carte> [--title slug] [--interval MS] [--geometry DIR] <matchId> [filmDir]
//
// PIÈGE : le paquet flag arrête l'analyse au premier argument positionnel — les options
// doivent précéder <matchId>.
//
// --map est OBLIGATOIRE : les bornes de déquantification sont propres à la carte (AABB de
// son BSP). Elles sont lues dans le catalogue versionné
// data/titles/{slug}/reference/map_quant_bounds.json (cmd/mapquant-build). Une carte
// absente du catalogue fait ÉCHOUER le build — un artefact aux bornes d'une autre carte
// serait faux d'un facteur d'échelle arbitraire, sans que rien ne le signale.
//
// Exemple (depuis apps/go-api, cache film dans le main tree) :
//
//	CGO_ENABLED=0 LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-build 000d5950 --map Cliffhanger
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

func main() {
	titleFlag := flag.String("title", title.DefaultSlug, "slug du titre")
	interval := flag.Int("interval", replay.DefaultFrameIntervalMS, "pas de temps du rejeu, en ms")
	geomDir := flag.String("geometry", "", "répertoire des CSV de géométrie (défaut : <repo>/.ai/re_dump ; vide si absent)")
	mapName := flag.String("map", "", "nom de carte du match (obligatoire : porte les bornes de déquantification)")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 || *mapName == "" {
		slog.Error("usage: replay-build --map <carte> [--title slug] [--interval MS] [--geometry DIR] <matchId> [filmDir] (les options précèdent le matchId)")
		os.Exit(2)
	}
	matchID := args[0]

	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		slog.Error("racine repo", "err", err)
		os.Exit(1)
	}
	entry, err := resolveMapEntry(repoRoot, *titleFlag, *mapName)
	if err != nil {
		slog.Error("bornes de la carte", "err", err, "map", *mapName)
		os.Exit(1)
	}
	worldRange := entry.Range()
	filmDir := filepath.Join(repoRoot, "data", "cache", "film_chunks", matchID)
	if len(args) >= 2 {
		filmDir = args[1]
	}
	if *geomDir == "" {
		*geomDir = filepath.Join(repoRoot, ".ai", "V7.5", "dumps")
	}

	doc, err := replay.BuildFromFilm(matchID, *titleFlag, filmDir, replay.Options{
		FrameIntervalMS: *interval,
		Geometry:        loadGeometry(*geomDir),
		Structure:       loadStructure(repoRoot, *titleFlag, entry.Module),
		WorldRange:      &worldRange,
	})
	if err != nil {
		slog.Error("décodage du film", "err", err, "filmDir", filmDir)
		os.Exit(1)
	}
	if len(doc.Tracks) == 0 {
		slog.Error("aucune trajectoire décodée — artefact non écrit", "filmDir", filmDir)
		os.Exit(1)
	}

	outPath := title.NewPathResolver(repoRoot).ReplayArtifactPath(*titleFlag, matchID)
	size, err := writeArtifact(outPath, doc)
	if err != nil {
		slog.Error("écriture artefact", "err", err, "path", outPath)
		os.Exit(1)
	}
	slog.Info("artefact rejeu écrit",
		"path", outPath, "tracks", len(doc.Tracks), "points", totalPoints(doc),
		"frames", doc.FrameCount, "intervalMs", doc.FrameIntervalMS, "durationMs", doc.DurationMS,
		"geometry", len(doc.Geometry), "structure", len(doc.Structure), "tirs", len(doc.Shots),
		"grenades", len(doc.Grenades), "projectiles", len(doc.Projectiles),
		"bytes", size, "match", matchID, "title", *titleFlag)
}

// resolveMapEntry lit l'entrée de la carte dans le catalogue versionné : bornes de
// déquantification ET module (qui sert ensuite de clé au fichier de structure — le lien
// nom affiché -> module n'est déclaré qu'à un seul endroit, cmd/mapquant-build).
// Une carte absente est une ERREUR : produire l'artefact avec les bornes d'une autre carte
// donnerait des coordonnées fausses d'un facteur d'échelle arbitraire.
func resolveMapEntry(repoRoot, titleSlug, mapName string) (filmdec.MapQuantEntry, error) {
	path := title.NewPathResolver(repoRoot).MapQuantBoundsPath(titleSlug)
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		return filmdec.MapQuantEntry{}, err
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		return filmdec.MapQuantEntry{}, err
	}
	slog.Info("bornes de carte chargées", "map", mapName, "module", entry.Module,
		"W", fmt.Sprintf("%d/%d/%d", entry.AxisWidths[0], entry.AxisWidths[1], entry.AxisWidths[2]))
	return entry, nil
}

// loadStructure charge le fond de carte STRUCTUREL figé de la carte (sols, plateformes,
// rampes, murs). Son absence n'est PAS fatale : toutes les cartes n'ont pas encore de
// fichier de structure, et un rejeu sans fond reste lisible.
func loadStructure(repoRoot, titleSlug, module string) []replay.Surface {
	path := title.NewPathResolver(repoRoot).MapStructurePath(titleSlug, module)
	ms, err := replay.LoadMapStructure(path)
	if err != nil {
		slog.Warn("structure de carte indisponible — artefact sans fond structurel",
			"err", err, "path", path)
		return nil
	}
	slog.Info("structure de carte chargée", "module", module, "emprises", len(ms.Surfaces),
		"couverture", fmt.Sprintf("%.1f %%", ms.Stats.CoveragePct), "path", path)
	return ms.Surfaces
}

// loadGeometry charge le fond de carte ; l'absence des CSV n'est PAS fatale (le rejeu
// reste lisible sans repères), mais elle est journalisée.
func loadGeometry(dir string) []replay.MapObject {
	objs, skipped, err := replay.LoadGeometry(dir)
	if err != nil {
		slog.Warn("géométrie de carte indisponible — artefact sans fond", "err", err, "dir", dir)
		return nil
	}
	slog.Info("géométrie de carte chargée", "objets", len(objs), "sansEmprise", skipped, "dir", dir)
	return objs
}

// writeArtifact sérialise le document et l'écrit (crée les répertoires parents) ; renvoie
// la taille en octets.
func writeArtifact(outPath string, doc replay.ReplayDocument) (int, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, err
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	return len(blob), os.WriteFile(outPath, blob, 0o644)
}

func totalPoints(doc replay.ReplayDocument) int {
	n := 0
	for _, tr := range doc.Tracks {
		n += len(tr.Points)
	}
	return n
}
