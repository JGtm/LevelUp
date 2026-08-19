// cmd/variant-probe — SONDE (lot de mesure v7.5, 2026-08-19). PAS un outil de
// production : il ne produit aucune référence, n'écrit dans aucune base, et ne
// tourne jamais tout seul.
//
// Question mesurée : l'asset `UgcGameVariants` d'un mode porte-t-il l'ACTIVATION
// des socles d'armes (Cliffhanger : 17 poses au fichier de carte, 10 actives en
// CTF, 0 en Super Fiesta) ? Cf. .ai/V7.5/replay2d/PLAN_SONDE_VARIANT_SOCLES.md.
//
// GARDE : aucun défaut ne tape le réseau. Sans --variant ET --out explicites,
// l'outil sort en erreur sans émettre la moindre requête.
//
// Usage :
//
//	go run ./cmd/variant-probe --player <Gamertag> --out <dossier> \
//	    --variant <assetId>[:<versionId>] [--variant ...] [--fetch-files] [--engine]
//
// Authentification : ADR 0023 strict — MultiUserTokenStore via
// auth.RefreshHaloTokensViaStoreFirst, AUCUNE re-capture, aucune écriture de
// jeton. Un 401 arrête l'outil et remonte l'erreur exacte.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

type variantList []string

func (v *variantList) String() string     { return strings.Join(*v, ",") }
func (v *variantList) Set(s string) error { *v = append(*v, s); return nil }

// probeOpts regroupe les options d'une passe de sonde (évite la fonction à
// 6 paramètres — seuil du CLAUDE.md).
type probeOpts struct {
	outDir     string
	segment    string
	fetchFiles bool
	engine     bool
}

func main() {
	var (
		variants  variantList
		player    = flag.String("player", "", "gamertag utilisé pour l'authentification (ADR 0023)")
		titleSlug = flag.String("title", titlePkg.DefaultSlug, "slug du titre")
		outDir    = flag.String("out", "", "dossier de dépôt des payloads (OBLIGATOIRE)")
		rateMS    = flag.Int("rate-ms", 1000, "délai entre deux requêtes (politesse)")
		fetchAll  = flag.Bool("fetch-files", false, "télécharger aussi les fichiers référencés par l'asset")
		engine    = flag.Bool("engine", false, "suivre EngineGameVariantLink et récupérer l'asset moteur")
		scanDir   = flag.String("scan", "", "mode HORS LIGNE : analyser les payloads deja deposes dans ce dossier (aucune requete)")
		segment   = flag.String("segment", variantSegment, "segment discovery interroge (ugcGameVariants, mapModePairs, maps...)")
	)
	flag.Var(&variants, "variant", "assetId[:versionId] d'un UgcGameVariant (répétable, OBLIGATOIRE)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	// Mode hors ligne : ni auth ni reseau, il ne lit que des fichiers deposes.
	if strings.TrimSpace(*scanDir) != "" {
		if err := runScan(ctx, *scanDir); err != nil {
			fail(ctx, "scan hors ligne", err)
		}
		return
	}

	// Garde de sonde : rien d'implicite, aucune requête sans cible explicite.
	if len(variants) == 0 || strings.TrimSpace(*outDir) == "" {
		fail(ctx, "arguments", fmt.Errorf("--variant et --out sont obligatoires (sonde : aucun défaut réseau)"))
	}

	cfg, err := config.Load()
	if err != nil {
		fail(ctx, "config.Load", err)
	}
	tokens, err := resolveTokens(ctx, cfg, *player, *titleSlug)
	if err != nil {
		// ADR 0023 : on ne répare pas l'auth ici, on remonte l'erreur exacte.
		fail(ctx, "authentification (ADR 0023 : ne PAS re-capturer, diagnostiquer)", err)
	}
	client := newUGCClient(tokens)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(ctx, "création du dossier de sortie", err)
	}

	opts := probeOpts{outDir: *outDir, fetchFiles: *fetchAll, engine: *engine, segment: *segment}
	var failures []string
	for i, spec := range variants {
		if i > 0 && *rateMS > 0 {
			time.Sleep(time.Duration(*rateMS) * time.Millisecond)
		}
		if err := probeVariant(ctx, client, spec, opts); err != nil {
			slog.ErrorContext(ctx, "variant-probe: variant non récupéré", "spec", spec, "err", err)
			failures = append(failures, fmt.Sprintf("%s: %v", spec, err))
		}
	}
	if len(failures) > 0 {
		slog.ErrorContext(ctx, "variant-probe: échecs", "n", len(failures),
			"detail", strings.Join(failures, " | "))
		os.Exit(1)
	}
}

// assetHead est la partie du document d'asset que la sonde exploite. Le reste du
// JSON n'est PAS décodé : il est déposé BRUT, cette sonde cherche des champs
// qu'on ne connaît pas encore et un décodage typé les mangerait en silence.
type assetHead struct {
	AssetID    string          `json:"AssetId"`
	VersionID  string          `json:"VersionId"`
	PublicName json.RawMessage `json:"PublicName"`
	Files      assetFiles      `json:"Files"`
	EngineLink struct {
		AssetID   string     `json:"AssetId"`
		VersionID string     `json:"VersionId"`
		Name      string     `json:"PublicName"`
		Files     assetFiles `json:"Files"`
	} `json:"EngineGameVariantLink"`
}

type assetFiles struct {
	Prefix            string   `json:"Prefix"`
	FileRelativePaths []string `json:"FileRelativePaths"`
}

// probeVariant récupère un UgcGameVariant, dépose son JSON BRUT, puis suit les
// pistes demandées (fichiers référencés, asset moteur).
func probeVariant(ctx context.Context, c *ugcClient, spec string, opts probeOpts) error {
	assetID, versionID := splitSpec(spec)
	head, err := c.dumpAsset(ctx, opts.segment, assetID, versionID, opts.outDir)
	if err != nil {
		return err
	}
	if opts.fetchFiles {
		c.dumpFiles(ctx, head.Files, filepath.Join(opts.outDir, "files", assetID))
	}
	if opts.engine && head.EngineLink.AssetID != "" {
		if err := c.probeEngine(ctx, head, opts); err != nil {
			return fmt.Errorf("asset moteur %s: %w", head.EngineLink.AssetID, err)
		}
	}
	return nil
}

// probeEngine récupère l'asset EngineGameVariants pointé par un variant.
func (c *ugcClient) probeEngine(ctx context.Context, head *assetHead, opts probeOpts) error {
	slog.InfoContext(ctx, "variant-probe: lien moteur",
		"from", head.AssetID, "engine_asset", head.EngineLink.AssetID,
		"engine_version", head.EngineLink.VersionID, "engine_name", head.EngineLink.Name,
		"files_dans_le_lien", len(head.EngineLink.Files.FileRelativePaths))
	eng, err := c.dumpAsset(ctx, engineSegment, head.EngineLink.AssetID, head.EngineLink.VersionID, opts.outDir)
	if err != nil {
		return err
	}
	if opts.fetchFiles {
		c.dumpFiles(ctx, eng.Files, filepath.Join(opts.outDir, "files", eng.AssetID))
	}
	return nil
}

// dumpAsset récupère un document d'asset, l'écrit BRUT, et retourne son en-tête.
func (c *ugcClient) dumpAsset(ctx context.Context, segment, assetID, versionID, outDir string) (*assetHead, error) {
	raw, err := c.fetchAssetRaw(ctx, segment, assetID, versionID)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(outDir, segment+"_"+assetID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return nil, fmt.Errorf("écriture %s: %w", path, err)
	}
	var head assetHead
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("décoder l'en-tête de %s: %w", assetID, err)
	}
	slog.InfoContext(ctx, "variant-probe: asset récupéré",
		"segment", segment, "asset_id", head.AssetID, "version_id", head.VersionID,
		"name", string(head.PublicName), "bytes", len(raw),
		"files", len(head.Files.FileRelativePaths),
		"file_paths", strings.Join(head.Files.FileRelativePaths, " | "),
		"path", path)
	return &head, nil
}

// dumpFiles télécharge les fichiers référencés par un asset. Un échec par fichier
// est loggué et n'interrompt pas la passe : la sonde veut l'inventaire, pas la
// perfection.
func (c *ugcClient) dumpFiles(ctx context.Context, files assetFiles, dir string) {
	if len(files.FileRelativePaths) == 0 {
		slog.InfoContext(ctx, "variant-probe: aucun fichier référencé", "prefix", files.Prefix)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.ErrorContext(ctx, "variant-probe: création du dossier de fichiers", "dir", dir, "err", err)
		return
	}
	for _, rel := range files.FileRelativePaths {
		body, err := c.fetchBlob(ctx, files.Prefix, rel)
		if err != nil {
			slog.ErrorContext(ctx, "variant-probe: fichier non téléchargé", "file", rel, "err", err)
			continue
		}
		dest := filepath.Join(dir, strings.ReplaceAll(rel, "/", "_"))
		if err := os.WriteFile(dest, body, 0o644); err != nil {
			slog.ErrorContext(ctx, "variant-probe: écriture du fichier", "dest", dest, "err", err)
		}
	}
}

// splitSpec découpe "assetId[:versionId]".
func splitSpec(spec string) (string, string) {
	if i := strings.Index(spec, ":"); i > 0 {
		return spec[:i], spec[i+1:]
	}
	return spec, ""
}

func fail(ctx context.Context, what string, err error) {
	slog.ErrorContext(ctx, "variant-probe: échec", "etape", what, "err", err)
	os.Exit(1)
}
