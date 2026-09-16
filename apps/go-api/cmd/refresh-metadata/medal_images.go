// medal_images.go — sous-commande `medal-images` : audite et complete le
// referentiel d'icones de medailles servi par LevelUp
// (`static/medals/{slug}/{medal_id}.png`).
//
// Pourquoi ici : la resolution de jetons (ADR 0023, store-first) et la
// convention de sous-commandes existent deja dans ce CLI ; la source officielle
// des medailles est le meme GameCMS que `refresh-metadata medals`.
//
// La sous-commande ne touche AUCUNE base DuckDB : elle lit le catalogue
// officiel (metadata.json) et le dossier d'icones, et n'ecrit que des PNG. Elle
// se lance donc serveur allume (modele mono-writer, ADR 0013).
//
// Points d'acces et disposition des `spriteIndex` : den.dev, « Halo Infinite
// Medal API: Infection, VIP, Extraction » (2023-10-11),
// https://den.dev/blog/halo-infinite-medals-api/ — catalogue
// `hi/Waypoint/file/medals/metadata.json`, feuille
// `hi/Waypoint/file/medals/images/medal_sheet_xl.png`, `spriteIndex` lu ligne
// par ligne. PIEGE DOCUMENTE PAR L'ARTICLE, toujours vrai au 2026-09-14 : la
// feuille porte des tuiles absentes du JSON (169 tuiles occupees pour
// 151 medailles catalogees). Le JSON reste la reference ; --pin ancre a la main
// une medaille verifiee mais non catalogee.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	authpkg "levelup/go-api/internal/platform/auth"
)

// defaultMedalMetadataPath : chemin GameCMS du catalogue des medailles, apres le
// prefixe de jeu (`hi`, `h5`). Surchargeable par --metadata-path si 343 deplace
// le fichier.
const (
	defaultMedalMetadataPath = "Waypoint/file/medals/metadata.json"
	// defaultMedalSheetPath : feuille de sprites « extra large » officielle.
	defaultMedalSheetPath = "Waypoint/file/medals/images/medal_sheet_xl.png"
	// defaultMedalTilePx : cote d'une tuile — mesure des 166 icones deja
	// versionnees dans static/medals/halo_infinite/ (256x256).
	defaultMedalTilePx = 256
)

// medalCatalog est la forme du catalogue officiel GameCMS
// (`/{prefix}/Waypoint/file/medals/metadata.json`). Les clefs JSON y sont en
// minuscule initiale (`medals`, `nameId`) — le decodage Go est insensible a la
// casse, les tags gardent la forme lisible.
type medalCatalog struct {
	Medals []medalCatalogEntry `json:"Medals"`
}

// medalSpriteSheet est la grille de la feuille de sprites, DEDUITE de l'image
// (cote de tuile impose, colonnes = largeur / cote) et non lue du JSON : la
// feuille devance le catalogue, la geometrie doit rester mesurable seule.
type medalSpriteSheet struct {
	Size    int
	Columns int
}

type medalCatalogEntry struct {
	NameID        int64          `json:"NameId"`
	Name          medalLocalized `json:"Name"`
	Description   medalLocalized `json:"Description"`
	SpriteIndex   int            `json:"SpriteIndex"`
	SortingWeight int            `json:"SortingWeight"`
	Difficulty    string         `json:"Difficulty"`
	Type          string         `json:"Type"`
	PersonalScore int            `json:"PersonalScore"`
}

type medalLocalized struct {
	Value        string            `json:"value"`
	Translations map[string]string `json:"translations"`
}

func runMedalImages(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("medal-images", flag.ExitOnError)
	titleID := fs.String("title-id", titlePkg.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	player := fs.String("player", "", "Gamertag pour la resolution des jetons (si SPARTAN_TOKEN absent)")
	outDir := fs.String("out-dir", "", "Dossier des icones (defaut : <repo>/static/medals/<slug>)")
	dumpRaw := fs.String("dump-raw", "", "Ecrit le catalogue officiel brut dans ce fichier")
	metaPath := fs.String("metadata-path", defaultMedalMetadataPath, "Chemin GameCMS du catalogue, apres le prefixe de jeu")
	sheetPath := fs.String("sprite-sheet-path", defaultMedalSheetPath, "Chemin GameCMS de la feuille de sprites")
	tile := fs.Int("tile", defaultMedalTilePx, "Cote d'une tuile de la feuille, en pixels")
	download := fs.Bool("download", false, "Telecharge les icones manquantes (sinon rapport seul)")
	audit := fs.Bool("audit-sheet", false, "Audite la feuille de sprites tuile par tuile (aucune ecriture)")
	extract := fs.String("extract-tiles", "", "Indices de tuiles a extraire pour inspection (ex: 55,56,57)")
	extractDir := fs.String("extract-dir", "", "Dossier de sortie des tuiles extraites")
	pin := fs.String("pin", "", "Medailles hors catalogue : <id>:<spriteIndex>[,<id>:<spriteIndex>...]")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	tokens, err := resolveTokensWithoutPlayerDB(ctx, cfg, *player, *titleID)
	if err != nil {
		return fmt.Errorf("resolution jetons: %w", err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, tokens, "")
	ctx = ctxkeys.WithTitleSlug(ctx, *titleID)

	host, err := gamecmsHost(cfg, *titleID)
	if err != nil {
		return err
	}
	raw, err := fetchBytes(ctx, host+"/"+games.GamePrefix(*titleID)+"/"+strings.TrimLeft(*metaPath, "/"))
	if err != nil {
		return fmt.Errorf("catalogue officiel: %w", err)
	}
	if *dumpRaw != "" {
		if werr := os.WriteFile(*dumpRaw, raw, 0o644); werr != nil {
			return fmt.Errorf("dump-raw: %w", werr)
		}
		fmt.Printf("catalogue brut ecrit : %s (%d octets)\n", *dumpRaw, len(raw))
	}

	var cat medalCatalog
	if err := json.Unmarshal(raw, &cat); err != nil {
		return fmt.Errorf("parse catalogue: %w", err)
	}

	dir := *outDir
	if dir == "" {
		dir = filepath.Join(cfg.RepoRoot, "static", "medals", *titleID)
	}
	missing, orphans, err := diffMedalIcons(cat, dir)
	if err != nil {
		return err
	}

	fmt.Printf("catalogue officiel : %d medailles — icones locales : %s\n", len(cat.Medals), dir)
	fmt.Printf("officielles SANS icone locale : %d %v\n", len(missing), missing)
	fmt.Printf("icones locales HORS catalogue  : %d %v\n", len(orphans), orphans)

	if *audit {
		return auditSpriteSheet(ctx, host, *titleID, *sheetPath, *tile, cat, dir)
	}
	if *extract != "" {
		return extractTiles(ctx, host, *titleID, *sheetPath, *tile, *extract, *extractDir)
	}
	if *pin != "" {
		return pinTiles(ctx, host, *titleID, *sheetPath, *tile, *pin, dir)
	}
	if !*download || len(missing) == 0 {
		return nil
	}
	return downloadMedalIcons(ctx, host, *titleID, *sheetPath, *tile, cat, missing, dir)
}

// resolveTokensWithoutPlayerDB obtient les jetons Halo SANS ouvrir la base du
// joueur : `resolveTokens` (main.go) passe par config.ResolvePlayer, qui ouvre la
// player DB en écriture — interdit tant que le serveur la tient (ADR 0013,
// mono-writer). Ici db_profiles.json suffit pour le xuid, et le refresh passe par
// le chemin canonique ADR 0023 (MultiUserTokenStore, aucune re-capture).
// Même raisonnement et même forme que cmd/mapobj-build/auth.go.
func resolveTokensWithoutPlayerDB(ctx context.Context, cfg *config.AppConfig, playerSlug, titleSlug string) (*domain.HaloTokens, error) {
	if envToken := os.Getenv("SPARTAN_TOKEN"); envToken != "" {
		return &domain.HaloTokens{
			SpartanToken:   envToken,
			ClearanceToken: os.Getenv("CLEARANCE_TOKEN"),
		}, nil
	}
	if playerSlug == "" {
		return nil, fmt.Errorf("SPARTAN_TOKEN absent ET --player non fourni")
	}
	players, err := cfg.LoadPlayers(titleSlug)
	if err != nil {
		return nil, fmt.Errorf("lecture db_profiles.json: %w", err)
	}
	var xuid, gamertag string
	for _, p := range players {
		if strings.EqualFold(p.PlayerSlug, playerSlug) || strings.EqualFold(p.Gamertag, playerSlug) {
			xuid, gamertag = p.XUID, p.Gamertag
			break
		}
	}
	if xuid == "" {
		return nil, fmt.Errorf("joueur %q absent (ou sans xuid) de db_profiles.json pour le titre %s", playerSlug, titleSlug)
	}
	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	result, err := authpkg.RefreshHaloTokensViaStoreFirst(ctx, store, authpkg.NewSISUProvider(), xuid, gamertag)
	if err != nil {
		return nil, err
	}
	tokens := authpkg.HaloTokensFromExchange(result)
	if tokens == nil || tokens.SpartanToken == "" {
		return nil, fmt.Errorf("aucun jeton exploitable pour %q (xuid %s) — diagnostiquer la chaîne de refresh, ne PAS re-capturer", playerSlug, xuid)
	}
	return tokens, nil
}

// diffMedalIcons compare les identifiants du catalogue officiel au contenu du
// dossier d'icones. Retourne les manquantes (catalogue sans PNG) et les
// orphelines (PNG sans entree au catalogue).
func diffMedalIcons(cat medalCatalog, dir string) (missing, orphans []int64, err error) {
	local := map[int64]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("lecture %s: %w", dir, err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".png") {
			continue
		}
		id, perr := strconv.ParseInt(strings.TrimSuffix(name, ".png"), 10, 64)
		if perr != nil {
			continue
		}
		local[id] = true
	}
	official := map[int64]bool{}
	for _, m := range cat.Medals {
		official[m.NameID] = true
		if !local[m.NameID] {
			missing = append(missing, m.NameID)
		}
	}
	for id := range local {
		if !official[id] {
			orphans = append(orphans, id)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	sort.Slice(orphans, func(i, j int) bool { return orphans[i] < orphans[j] })
	return missing, orphans, nil
}

// downloadMedalIcons decoupe les icones manquantes dans la feuille de sprites
// officielle et les ecrit en PNG, un fichier par medaille.
//
// La grille se deduit de la feuille elle-meme : colonnes = largeur / tuile. La
// REFERENCE reste le metadata.json — la feuille contient des tuiles sans entree
// au catalogue (constat den.dev), on ne decoupe donc que les `spriteIndex` du
// catalogue.
func downloadMedalIcons(ctx context.Context, host, slug, sheetPath string, tile int, cat medalCatalog, missing []int64, dir string) error {
	sheetImg, grid, err := loadSpriteSheet(ctx, host, slug, sheetPath, tile)
	if err != nil {
		return err
	}
	byID := map[int64]medalCatalogEntry{}
	for _, m := range cat.Medals {
		byID[m.NameID] = m
	}
	for _, id := range missing {
		m, ok := byID[id]
		if !ok {
			continue
		}
		sub, cerr := cropSprite(sheetImg, m.SpriteIndex, grid)
		if cerr != nil {
			return fmt.Errorf("decoupe %d: %w", id, cerr)
		}
		path := filepath.Join(dir, strconv.FormatInt(id, 10)+".png")
		if werr := writePNG(path, sub); werr != nil {
			return werr
		}
		fmt.Printf("  ecrit %s (spriteIndex %d, %s / %s)\n",
			path, m.SpriteIndex, m.Name.Value, m.Name.Translations["fr-FR"])
	}
	return nil
}
