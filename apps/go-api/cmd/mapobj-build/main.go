// cmd/mapobj-build — construit le catalogue figé des objets d'objectif par carte.
//
// UN appel réseau par carte, résultat gelé sous data/titles/{slug}/reference/
// map_objectives.json (via PathResolver). Le rejeu d'un match reste 100 % hors ligne :
// il lit ce fichier, il n'appelle jamais l'UGC.
//
// Usage :
//
//	go run ./cmd/mapobj-build --player <Gamertag> --map-id <uuid> [--map-id <uuid>...]
//	go run ./cmd/mapobj-build --player <Gamertag> --all
//	go run ./cmd/mapobj-build --from-file <chemin.mvar> --map-id <uuid>   (hors ligne)
//	go run ./cmd/mapobj-build --refresh-from <dossier de .mvar>           (hors ligne, tout le catalogue)
//
// L'authentification suit ADR 0023 : MultiUserTokenStore d'abord, legacy ensuite,
// via auth.RefreshHaloTokensViaStoreFirst. AUCUNE re-capture de jeton.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

type mapIDList []string

func (m *mapIDList) String() string     { return strings.Join(*m, ",") }
func (m *mapIDList) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	var (
		mapIDs    mapIDList
		player    = flag.String("player", "", "gamertag utilisé pour l'authentification (ADR 0023)")
		titleSlug = flag.String("title", titlePkg.DefaultSlug, "slug du titre")
		all       = flag.Bool("all", false, "toutes les cartes distinctes de match_registry")
		fromFile  = flag.String("from-file", "", "parser un .mvar local au lieu d'appeler le réseau")
		refresh   = flag.String("refresh-from", "",
			"régénérer HORS LIGNE tout le catalogue depuis un dossier de .mvar locaux")
		rateMS = flag.Int("rate-ms", 1000, "délai entre deux requêtes réseau (politesse)")
		dryRun = flag.Bool("dry-run", false, "ne pas écrire le catalogue")
		// Sans ce dépôt, --refresh-from n'a aucune source : le chemin hors ligne exige un
		// dossier de .mvar que rien ne produisait. Les fichiers y sont nommés comme le
		// catalogue les enregistre (mvar_file), c'est ce que --refresh-from relit.
		saveMvar = flag.String("save-mvar", "",
			"déposer chaque .mvar téléchargé dans ce dossier (source de --refresh-from)")
	)
	flag.Var(&mapIDs, "map-id", "identifiant d'asset de carte (répétable)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fail(ctx, "config.Load", err)
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	outPath := resolver.MapObjectivesPath(*titleSlug)

	// Chemin hors ligne complet : re-parse du catalogue entier, aucune requête.
	// Il migre aussi le schéma, il ne passe donc pas par mergeExisting.
	if *refresh != "" {
		if err := refreshOffline(ctx, *refresh, outPath, *titleSlug, *dryRun); err != nil {
			fail(ctx, "régénération hors ligne", err)
		}
		return
	}

	cat := newCatalog(*titleSlug)
	if !*dryRun {
		if err := cat.mergeExisting(outPath); err != nil {
			fail(ctx, "fusion du catalogue existant", err)
		}
	}

	if *fromFile != "" {
		if len(mapIDs) != 1 {
			fail(ctx, "--from-file", errors.New("exige exactement un --map-id"))
		}
		if err := ingestLocal(ctx, cat, mapIDs[0], *fromFile); err != nil {
			fail(ctx, "ingestion locale", err)
		}
		finish(ctx, cat, outPath, *dryRun)
		return
	}

	targets, err := resolveTargets(ctx, cfg, mapIDs, *all)
	if err != nil {
		fail(ctx, "résolution des cartes cibles", err)
	}
	if len(targets) == 0 {
		fail(ctx, "aucune carte cible", errors.New("fournir --map-id ou --all"))
	}

	tokens, err := resolveTokens(ctx, cfg, *player, *titleSlug)
	if err != nil {
		fail(ctx, "authentification", err)
	}
	client := newUGCClient(tokens)

	var failures []string
	for i, t := range targets {
		if i > 0 && *rateMS > 0 {
			time.Sleep(time.Duration(*rateMS) * time.Millisecond)
		}
		if err := ingestRemote(ctx, cat, client, t, *saveMvar); err != nil {
			// Échec documenté, jamais contourné : une carte absente du catalogue
			// vaut mieux qu'un objectif affiché au mauvais endroit.
			slog.ErrorContext(ctx, "mapobj: carte non traitée",
				"map_id", t.mapID, "name", t.name, "err", err)
			failures = append(failures, fmt.Sprintf("%s (%s): %v", t.mapID, t.name, err))
			continue
		}
	}
	finish(ctx, cat, outPath, *dryRun)
	if len(failures) > 0 {
		slog.ErrorContext(ctx, "mapobj: cartes en échec", "n", len(failures),
			"detail", strings.Join(failures, " | "))
		os.Exit(1)
	}
}

type target struct {
	mapID     string
	versionID string
	name      string
}

// resolveTargets liste les cartes à traiter : explicites (--map-id) ou toutes
// celles présentes dans match_registry (--all).
func resolveTargets(ctx context.Context, cfg *config.AppConfig, explicit []string, all bool) ([]target, error) {
	if !all {
		out := make([]target, 0, len(explicit))
		for _, id := range explicit {
			out = append(out, target{mapID: id})
		}
		return out, nil
	}
	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(titlePkg.DefaultSlug)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			slog.WarnContext(ctx, "mapobj: fermeture DuckDB", "err", cerr)
		}
	}()
	if _, err := db.ExecContext(ctx,
		fmt.Sprintf("ATTACH '%s' AS s (READ_ONLY)", sharedPath)); err != nil {
		return nil, fmt.Errorf("attacher %s: %w", sharedPath, err)
	}
	rows, err := db.QueryContext(ctx, `
		SELECT map_id, ANY_VALUE(map_version_id), ANY_VALUE(map_name), COUNT(*) AS n
		FROM s.match_registry WHERE map_id IS NOT NULL
		GROUP BY map_id ORDER BY n DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []target
	for rows.Next() {
		var id, ver, name sql.NullString
		var n int
		if err := rows.Scan(&id, &ver, &name, &n); err != nil {
			return nil, err
		}
		out = append(out, target{mapID: id.String, versionID: ver.String, name: name.String})
	}
	return out, rows.Err()
}

// ingestRemote télécharge et ingère toutes les variantes .mvar d'une carte.
func ingestRemote(ctx context.Context, cat *catalog, c *ugcClient, t target, saveDir string) error {
	asset, err := c.fetchAsset(ctx, t.mapID, "")
	if err != nil {
		return err
	}
	files := asset.mvarPaths()
	if len(files) == 0 {
		return fmt.Errorf("aucun fichier .mvar dans l'asset (fichiers: %v)",
			asset.Files.FileRelativePaths)
	}
	// Une carte peut exposer plusieurs .mvar (Cliffhanger : map.mvar + ridgeline.mvar).
	// On retient celle qui porte le plus d'objets d'objectif, APRÈS avoir écarté les
	// variantes dont les objectifs sont rangés hors terrain (cf. variant.go).
	var best *mapEntry
	var bestVariant *mapvar.Variant
	parked := 0
	for _, rel := range files {
		buf, err := c.fetchMvar(ctx, asset, rel)
		if err != nil {
			return err
		}
		if err := saveVariantFile(ctx, saveDir, t.mapID, rel, buf); err != nil {
			return err
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			return fmt.Errorf("parser %s: %w", rel, err)
		}
		nObj := len(v.Objectives())
		slog.InfoContext(ctx, "mapobj: variante parsée",
			"map_id", t.mapID, "file", rel, "objets", len(v.Objects), "objectifs", nObj)
		if isParkedPalette(v) {
			parked++
			slog.WarnContext(ctx, "mapobj: variante écartée, objectifs rangés hors terrain",
				"map_id", t.mapID, "file", rel, "objectifs", nObj)
			continue
		}
		if best == nil || nObj > len(best.Objectives) {
			best = &mapEntry{
				MapID: t.mapID, VersionID: asset.VersionID, PublicName: asset.PublicName,
				MvarFile: rel, Module: strings.TrimSuffix(rel, ".mvar"),
				FetchedAt: time.Now().UTC(),
			}
			best.Objectives = v.Objectives()
			bestVariant = v
		}
	}
	if best == nil {
		// Carte absente du catalogue plutôt qu'objectif au mauvais endroit (fetch.go).
		return fmt.Errorf("aucune variante exploitable : %d fichier(s), %d écarté(s) pour objectifs rangés",
			len(files), parked)
	}
	cat.addVariant(best, bestVariant)
	return nil
}

// saveVariantFile dépose un .mvar téléchargé dans un SOUS-DOSSIER par map_id.
//
// Pas à plat : plusieurs cartes exposent un fichier nommé `map.mvar` (Vagabond et une
// variante de Highpower, mesuré le 2026-08-08). À plat, la seconde écrase la première et
// --refresh-from rendrait ensuite les objectifs d'une carte pour une autre, sans un mot.
// mvarPath (refresh.go) lit ce sous-dossier en priorité.
func saveVariantFile(ctx context.Context, dir, mapID, rel string, buf []byte) error {
	if dir == "" {
		return nil
	}
	dir = filepath.Join(dir, mapID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := rel
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	path := filepath.Join(dir, base)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return err
	}
	slog.InfoContext(ctx, "mapobj: .mvar déposé", "path", path, "octets", len(buf))
	return nil
}

// ingestLocal ingère un .mvar déjà présent sur disque (chemin hors ligne, pour
// rejouer un parse sans re-solliciter le réseau).
func ingestLocal(ctx context.Context, cat *catalog, mapID, path string) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	v, err := mapvar.Parse(buf)
	if err != nil {
		return err
	}
	base := path
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	e := &mapEntry{MapID: mapID, MvarFile: base,
		Module: strings.TrimSuffix(base, ".mvar"), FetchedAt: time.Now().UTC()}
	cat.addVariant(e, v)
	slog.InfoContext(ctx, "mapobj: variante locale ingérée",
		"map_id", mapID, "file", base, "objets", len(v.Objects), "objectifs", len(e.Objectives))
	return nil
}

func finish(ctx context.Context, cat *catalog, outPath string, dryRun bool) {
	if dryRun {
		slog.InfoContext(ctx, "mapobj: dry-run, rien écrit", "cartes", len(cat.Maps))
		return
	}
	if err := cat.write(outPath); err != nil {
		fail(ctx, "écriture du catalogue", err)
	}
	slog.InfoContext(ctx, "mapobj: catalogue écrit", "path", outPath, "cartes", len(cat.Maps))
}

func fail(ctx context.Context, what string, err error) {
	slog.ErrorContext(ctx, "mapobj: échec", "etape", what, "err", err)
	os.Exit(1)
}
