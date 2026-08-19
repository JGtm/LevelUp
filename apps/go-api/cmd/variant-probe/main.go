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
//	    --variant <assetId>[:<versionId>] [--variant ...]
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

func main() {
	var (
		variants  variantList
		player    = flag.String("player", "", "gamertag utilisé pour l'authentification (ADR 0023)")
		titleSlug = flag.String("title", titlePkg.DefaultSlug, "slug du titre")
		outDir    = flag.String("out", "", "dossier de dépôt des payloads (OBLIGATOIRE)")
		rateMS    = flag.Int("rate-ms", 1000, "délai entre deux requêtes (politesse)")
	)
	flag.Var(&variants, "variant", "assetId[:versionId] d'un UgcGameVariant (répétable, OBLIGATOIRE)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

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

	var failures []string
	for i, spec := range variants {
		if i > 0 && *rateMS > 0 {
			time.Sleep(time.Duration(*rateMS) * time.Millisecond)
		}
		if err := probeVariant(ctx, client, spec, *outDir); err != nil {
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

// probeVariant récupère un asset UgcGameVariant et dépose son JSON BRUT.
//
// Le brut est déposé tel quel : cette sonde cherche des champs qu'on ne connaît
// pas encore, un décodage typé les mangerait en silence.
func probeVariant(ctx context.Context, c *ugcClient, spec, outDir string) error {
	assetID, versionID := splitSpec(spec)
	raw, err := c.fetchVariantRaw(ctx, assetID, versionID)
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, assetID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("écriture %s: %w", path, err)
	}

	// Résumé lisible : ce que la sonde attend en premier (les fichiers référencés).
	var head struct {
		AssetID    string          `json:"AssetId"`
		VersionID  string          `json:"VersionId"`
		PublicName json.RawMessage `json:"PublicName"`
		Files      struct {
			Prefix            string   `json:"Prefix"`
			FileRelativePaths []string `json:"FileRelativePaths"`
		} `json:"Files"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return fmt.Errorf("décoder l'en-tête de %s: %w", assetID, err)
	}
	slog.InfoContext(ctx, "variant-probe: asset récupéré",
		"asset_id", head.AssetID, "version_id", head.VersionID,
		"name", string(head.PublicName), "bytes", len(raw),
		"files", len(head.Files.FileRelativePaths),
		"file_paths", strings.Join(head.Files.FileRelativePaths, " | "),
		"path", path)
	return nil
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
