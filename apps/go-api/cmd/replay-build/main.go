// cmd/replay-build — génère l'artefact de rejeu 2D d'un match à partir des SEULS CHUNKS
// DU FILM (décodage offline des positions bipeds, zéro entrée Cheat Engine), plus un fond
// de carte optionnel (props Forge). Écrit data/cache/replays/{title}/{matchId}.json
// (résolu par PathResolver).
//
// L'ASSEMBLAGE (catalogue de bornes + libellés + structure + écriture) vit dans
// `internal/replaybuild`, partagé avec `levelup backfill-replay`, l'action admin et
// l'étape post-sync — ce binaire n'est que son enveloppe CLI unitaire.
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
//	CGO_ENABLED=0 LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-build --map Cliffhanger 000d5950
package main

import (
	"flag"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/replaybuild"
)

func main() {
	titleFlag := flag.String("title", title.DefaultSlug, "slug du titre")
	interval := flag.Int("interval", 0, "pas de temps du rejeu, en ms (0 = défaut)")
	geomDir := flag.String("geometry", "",
		"répertoire des CSV de props Forge (défaut : PathResolver.MapGeometryDir du titre)")
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
	builder, err := replaybuild.NewBuilder(repoRoot, *titleFlag)
	if err != nil {
		slog.Error("préparation du builder", "err", err, "title", *titleFlag)
		os.Exit(1)
	}
	builder.WithFrameInterval(*interval)
	if *geomDir != "" {
		builder.WithGeometryDir(*geomDir)
	}

	// La disposition du cache film n'est déclarée que dans filmcache (garde-rail).
	filmDir := filmcache.ChunkDir(title.NewPathResolver(repoRoot).CacheRootDir(), title.FilmShortMatchID(matchID))
	if len(args) >= 2 {
		filmDir = args[1]
	}

	out, err := builder.BuildMatch(matchID, []string{*mapName}, filmDir)
	if err != nil {
		slog.Error("construction de l'artefact", "err", err, "filmDir", filmDir, "match", matchID)
		os.Exit(1)
	}
	slog.Info("artefact rejeu écrit",
		"path", out.ArtifactPath, "tracks", out.Tracks, "module", out.Module,
		"bytes", out.Bytes, "match", matchID, "title", *titleFlag)
}
