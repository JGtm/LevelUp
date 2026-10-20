// medal_sheet.go — lecture de la feuille de sprites officielle des medailles :
// telechargement, deduction de la grille, decoupe, audit tuile par tuile et
// ancrage manuel. Extrait de medal_images.go (seuil de 500 lignes).
//
// Reference des points d acces et de la disposition des `spriteIndex` : voir le
// commentaire de tete de medal_images.go (article den.dev du 2023-10-11).
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// extractTiles ecrit des tuiles nommees `tuile_<index>.png` pour inspection
// visuelle — sert a VERIFIER a quelle medaille correspond une tuile absente du
// metadata.json avant de la figer avec --pin.
func extractTiles(ctx context.Context, host, slug, sheetPath string, tile int, list, outDir string) error {
	if outDir == "" {
		return fmt.Errorf("--extract-tiles exige --extract-dir")
	}
	sheetImg, grid, err := loadSpriteSheet(ctx, host, slug, sheetPath, tile)
	if err != nil {
		return err
	}
	for _, tok := range strings.Split(list, ",") {
		idx, perr := strconv.Atoi(strings.TrimSpace(tok))
		if perr != nil {
			return fmt.Errorf("indice de tuile %q invalide", tok)
		}
		sub, cerr := cropSprite(sheetImg, idx, grid)
		if cerr != nil {
			return cerr
		}
		path := filepath.Join(outDir, fmt.Sprintf("tuile_%d.png", idx))
		if werr := writePNG(path, sub); werr != nil {
			return werr
		}
		fmt.Printf("  ecrit %s\n", path)
	}
	return nil
}

// pinTiles ecrit l'icone de medailles ABSENTES du metadata.json, dont la tuile a
// ete etablie a la main (verification visuelle via --extract-tiles). La feuille
// devance le catalogue : sans cet ancrage, une medaille recente n'a pas d'image.
func pinTiles(ctx context.Context, host, slug, sheetPath string, tile int, pins, dir string) error {
	sheetImg, grid, err := loadSpriteSheet(ctx, host, slug, sheetPath, tile)
	if err != nil {
		return err
	}
	for _, tok := range strings.Split(pins, ",") {
		parts := strings.SplitN(strings.TrimSpace(tok), ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("ancrage %q invalide, attendu <id>:<spriteIndex>", tok)
		}
		id, ierr := strconv.ParseInt(parts[0], 10, 64)
		idx, xerr := strconv.Atoi(parts[1])
		if ierr != nil || xerr != nil {
			return fmt.Errorf("ancrage %q invalide, attendu <id>:<spriteIndex>", tok)
		}
		sub, cerr := cropSprite(sheetImg, idx, grid)
		if cerr != nil {
			return cerr
		}
		path := filepath.Join(dir, strconv.FormatInt(id, 10)+".png")
		if werr := writePNG(path, sub); werr != nil {
			return werr
		}
		fmt.Printf("  ecrit %s (tuile %d)\n", path, idx)
	}
	return nil
}

// auditSpriteSheet imprime, tuile par tuile, l'etat de la feuille officielle :
// medaille du catalogue qui la reference, vacuite (tuile entierement
// transparente) et icone locale identique au pixel pres. Sert a instruire les
// medailles absentes du catalogue mais presentes dans la feuille (piege
// documente par den.dev : la feuille et le JSON ne coincident pas).
func auditSpriteSheet(ctx context.Context, host, slug, sheetPath string, tile int, cat medalCatalog, dir string) error {
	sheetImg, grid, err := loadSpriteSheet(ctx, host, slug, sheetPath, tile)
	if err != nil {
		return err
	}
	rows := sheetImg.Bounds().Dy() / grid.Size
	byIndex := map[int]medalCatalogEntry{}
	for _, m := range cat.Medals {
		byIndex[m.SpriteIndex] = m
	}
	locals, err := loadLocalIcons(dir)
	if err != nil {
		return err
	}
	for idx := 0; idx < rows*grid.Columns; idx++ {
		sub, cerr := cropSprite(sheetImg, idx, grid)
		if cerr != nil {
			return cerr
		}
		state := "LIBRE"
		if m, ok := byIndex[idx]; ok {
			state = fmt.Sprintf("catalogue %d %s", m.NameID, m.Name.Value)
		} else if isBlank(sub) {
			state = "VIDE"
		}
		match := "-"
		if id, ok := matchLocalIcon(sub, locals); ok {
			match = strconv.FormatInt(id, 10)
		}
		fmt.Printf("tuile %3d  %-42s  icone locale identique : %s\n", idx, state, match)
	}
	return nil
}

// resolveTokensWithoutPlayerDB obtient les jetons Halo SANS ouvrir la base du
// joueur : `resolveTokens` (main.go) passe par config.ResolvePlayer, qui ouvre la
// player DB en écriture — interdit tant que le serveur la tient (ADR 0013,
// mono-writer). Ici db_profiles.json suffit pour le xuid, et le refresh passe par

// loadSpriteSheet telecharge la feuille officielle et deduit sa grille
// (colonnes = largeur / cote de tuile).
func loadSpriteSheet(ctx context.Context, host, slug, sheetPath string, tile int) (image.Image, medalSpriteSheet, error) {
	url := host + "/" + games.GamePrefix(slug) + "/" + strings.TrimLeft(sheetPath, "/")
	img, err := fetchPNG(ctx, url)
	if err != nil {
		return nil, medalSpriteSheet{}, fmt.Errorf("feuille de sprites: %w", err)
	}
	b := img.Bounds()
	if tile <= 0 || b.Dx()%tile != 0 {
		return nil, medalSpriteSheet{}, fmt.Errorf("taille de tuile %d incompatible avec la feuille %v", tile, b)
	}
	grid := medalSpriteSheet{Size: tile, Columns: b.Dx() / tile}
	fmt.Printf("feuille : %s — %dx%d, tuile %dpx, %d colonnes, %d lignes\n",
		url, b.Dx(), b.Dy(), grid.Size, grid.Columns, b.Dy()/tile)
	return img, grid, nil
}

// loadLocalIcons charge les icones deja versionnees, indexees par identifiant.
func loadLocalIcons(dir string) (map[int64]image.Image, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[int64]image.Image{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		id, perr := strconv.ParseInt(strings.TrimSuffix(e.Name(), ".png"), 10, 64)
		if perr != nil {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			return nil, rerr
		}
		img, derr := png.Decode(bytes.NewReader(data))
		if derr != nil {
			return nil, fmt.Errorf("decodage %s: %w", e.Name(), derr)
		}
		out[id] = img
	}
	return out, nil
}

// isBlank indique si une tuile est entierement transparente.
func isBlank(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				return false
			}
		}
	}
	return true
}

// matchLocalIcon cherche une icone locale identique au pixel pres a la tuile.
func matchLocalIcon(tile image.Image, locals map[int64]image.Image) (int64, bool) {
	ids := make([]int64, 0, len(locals))
	for id := range locals {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if sameImage(tile, locals[id]) {
			return id, true
		}
	}
	return 0, false
}

func sameImage(a, b image.Image) bool {
	ab, bb := a.Bounds(), b.Bounds()
	if ab.Dx() != bb.Dx() || ab.Dy() != bb.Dy() {
		return false
	}
	for y := 0; y < ab.Dy(); y++ {
		for x := 0; x < ab.Dx(); x++ {
			if a.At(ab.Min.X+x, ab.Min.Y+y) != b.At(bb.Min.X+x, bb.Min.Y+y) {
				return false
			}
		}
	}
	return true
}

// gamecmsHost resout l'hote GameCMS du titre via les manifestes (title-agnostic,
// aucun hote en dur).
func gamecmsHost(cfg *config.AppConfig, slug string) (string, error) {
	reg := mappings.NewRegistry()
	configRoot := cfg.RepoRoot
	if errs := reg.LoadFromConfigDir(configRoot, []string{slug, titlePkg.DefaultSlug}, nil); len(errs) != 0 {
		return "", fmt.Errorf("chargement manifestes %s: %v", configRoot, errs)
	}
	host, ok := games.NewMappingsEndpointResolver(reg, titlePkg.DefaultSlug).HostFor(slug, games.EndpointGameCMS)
	if !ok {
		return "", fmt.Errorf("hote gamecms non resolu pour %q", slug)
	}
	return strings.TrimRight(host, "/"), nil
}

func fetchPNG(ctx context.Context, url string) (image.Image, error) {
	body, err := fetchBytes(ctx, url)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(body))
}

// fetchBytes fait un GET authentifie GameCMS (jetons Spartan + clearance du ctx).
func fetchBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if tk := ctxkeys.HaloTokens(ctx); tk != nil {
		req.Header.Set("x-343-authorization-spartan", tk.SpartanToken)
		if tk.ClearanceToken != "" {
			req.Header.Set("343-clearance", tk.ClearanceToken)
		}
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: statut %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// cropSprite extrait la cellule `index` d'une feuille de sprites en grille.
func cropSprite(sheet image.Image, index int, meta medalSpriteSheet) (image.Image, error) {
	if meta.Size <= 0 || meta.Columns <= 0 {
		return nil, fmt.Errorf("metadonnees de feuille invalides (size=%d columns=%d)", meta.Size, meta.Columns)
	}
	x := (index % meta.Columns) * meta.Size
	y := (index / meta.Columns) * meta.Size
	r := image.Rect(x, y, x+meta.Size, y+meta.Size)
	sub, ok := sheet.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil, fmt.Errorf("image non decoupable")
	}
	if !r.In(sheet.Bounds()) {
		return nil, fmt.Errorf("cellule %d hors de la feuille %v", index, sheet.Bounds())
	}
	return sub.SubImage(r), nil
}

func writePNG(path string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
