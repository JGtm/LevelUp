// Package ops — seed_demo_multititle.go : orchestrateur multi-titre de la démo.
//
// La démo n'est plus mono-titre : on seede N titres (Halo Infinite + Halo 5…) dans
// le MÊME arbre data/demo, puis on écrit UNE fois db_profiles.json v3 (une entrée
// par couple titre × gamertag démo) + app_settings.json.
//
// Layout de sortie (cf. demoTitleSubdir) :
//   - titre par défaut (halo_infinite) → PLAT, byte-identique à la démo mono-titre :
//     data/demo/warehouse/*.duckdb, data/demo/players/DEMO/stats.duckdb
//   - titre additionnel (halo_5)       → title-scopé (miroir du PathResolver prod) :
//     data/demo/titles/halo_5/warehouse/*.duckdb, data/demo/titles/halo_5/players/DEMO/…
//
// La résolution démo (internal/config) lit ce layout title-aware (cf. Phase 2).
package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
)

// demoTitleSubdir retourne le sous-répertoire de sortie d'un titre dans l'arbre démo.
// Titre par défaut (ou slug vide) → outDir tel quel (layout PLAT legacy, byte-identique
// mono-titre). Titre additionnel → outDir/titles/{slug}/ (miroir PathResolver prod).
func demoTitleSubdir(outDir, slug string) string {
	if slug == "" || slug == titlePkg.DefaultSlug {
		return outDir
	}
	return filepath.Join(outDir, "titles", slug)
}

// demoPlayerDBRelPath retourne le chemin relatif (à la racine démo) de la player DB
// démo d'un titre — cohérent avec demoTitleSubdir, pour db_profiles.json.
func demoPlayerDBRelPath(slug, demoDir string) string {
	if slug == "" || slug == titlePkg.DefaultSlug {
		return filepath.ToSlash(filepath.Join("data", "players", demoDir, "stats.duckdb"))
	}
	return filepath.ToSlash(filepath.Join("data", "titles", slug, "players", demoDir, "stats.duckdb"))
}

// TitleSeedSpec décrit un titre à seeder dans la démo.
type TitleSeedSpec struct {
	Slug         string // ex: "halo_infinite", "halo_5"
	Gamertag     string // gamertag source pour ce titre (ex: "JGtm")
	MaxMatches   int    // matchs récents à extraire (0 → DefaultMaxMatches)
	MaxMedia     int    // clips à extraire (0 → DefaultMaxMedia)
	IncludeMedia bool   // extraire les médias HLS de ce titre
}

// SeedDemoMultiOptions configure l'orchestrateur multi-titre.
type SeedDemoMultiOptions struct {
	RepoRoot     string          // racine du repo source (où vivent data/titles/… et db_profiles.json)
	OutDir       string          // racine de sortie démo (ex: data/demo)
	ProfilesPath string          // db_profiles.json source (résout (slug,gamertag)→xuid,db_path)
	ServiceTag   string          // Spartan ID affiché sous le gamertag DEMO
	Titles       []TitleSeedSpec // titres à seeder (le 1er DOIT être le titre par défaut)
}

// SeedDemoMultiResult résume l'exécution multi-titre.
type SeedDemoMultiResult struct {
	OutDir    string
	PerTitle  map[string]SeedDemoResult
	Skipped   []string // titres ignorés (gamertag non configuré / source absente)
	Duration  time.Duration
	ConfigsOK bool
}

// SeedDemoMulti seede tous les titres dans le même arbre démo puis écrit les configs
// v3 une seule fois. Le titre par DÉFAUT est obligatoire (échec → fatal) ; un titre
// additionnel en échec est loggué et ignoré (la démo reste utilisable sur les autres).
func SeedDemoMulti(ctx context.Context, opts SeedDemoMultiOptions) (SeedDemoMultiResult, error) {
	start := time.Now()
	res := SeedDemoMultiResult{OutDir: opts.OutDir, PerTitle: map[string]SeedDemoResult{}}
	if len(opts.Titles) == 0 {
		return res, fmt.Errorf("seed-demo multi: aucun titre")
	}

	pr := titlePkg.NewPathResolver(opts.RepoRoot)
	byTitle := map[string][]seededDemoPlayer{}
	mediaEnabled := false

	for _, ts := range opts.Titles {
		isDefault := ts.Slug == "" || ts.Slug == titlePkg.DefaultSlug
		slug := ts.Slug
		if slug == "" {
			slug = titlePkg.DefaultSlug
		}

		xuid, playerRel, rerr := ResolveSourceXUIDForTitle(opts.ProfilesPath, slug, ts.Gamertag)
		if rerr != nil {
			if isDefault {
				return res, fmt.Errorf("seed-demo multi: résolution titre défaut %q: %w", slug, rerr)
			}
			slog.WarnContext(ctx, "seed-demo multi: titre ignoré (gamertag non configuré)",
				"title", slug, "gamertag", ts.Gamertag, "err", rerr)
			res.Skipped = append(res.Skipped, slug)
			continue
		}
		sourcePlayerDB := filepath.Join(opts.RepoRoot, playerRel)

		sopts := SeedDemoOptions{
			SourcePlayerDB: sourcePlayerDB,
			SourceSharedDB: pr.SharedDBPath(slug),
			SourceMetaDB:   pr.MetadataDBPath(slug),
			SourceXUID:     xuid,
			OutDir:         opts.OutDir,
			TitleSlug:      slug,
			MaxMatches:     ts.MaxMatches,
			SourceLabel:    ts.Gamertag,
			ServiceTag:     opts.ServiceTag,
			IncludeMedia:   ts.IncludeMedia,
			MaxMedia:       ts.MaxMedia,
			// Racine des player DBs du titre (…/players) pour emprunter une identité Spartan.
			SourcePlayersDir: filepath.Dir(filepath.Dir(sourcePlayerDB)),
			ProfilesPath:     opts.ProfilesPath,
			RepoRoot:         opts.RepoRoot,
			SkipConfigs:      true, // configs écrites une fois en fin d'orchestration
		}

		tres, terr := SeedDemo(ctx, sopts)
		if terr != nil {
			if isDefault {
				return res, fmt.Errorf("seed-demo multi: titre défaut %q: %w", slug, terr)
			}
			slog.WarnContext(ctx, "seed-demo multi: titre additionnel en échec, ignoré",
				"title", slug, "err", terr)
			res.Skipped = append(res.Skipped, slug)
			continue
		}
		res.PerTitle[slug] = tres
		byTitle[slug] = tres.SeededPlayers
		if tres.MediaCopied > 0 {
			mediaEnabled = true
		}
		slog.InfoContext(ctx, "seed-demo multi: titre seedé",
			"title", slug, "matches", len(tres.MatchIDs), "players", len(tres.SeededPlayers), "media", tres.MediaCopied)
	}

	if len(byTitle) == 0 {
		return res, fmt.Errorf("seed-demo multi: aucun titre seedé")
	}

	if err := writeDemoConfigsV3(opts.OutDir, byTitle, opts.ServiceTag, mediaEnabled); err != nil {
		return res, fmt.Errorf("seed-demo multi: write configs v3: %w", err)
	}
	res.ConfigsOK = true
	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "seed-demo multi: terminé",
		"titles", len(byTitle), "skipped", len(res.Skipped), "duration", res.Duration)
	return res, nil
}

// TitlesForGamertag retourne les slugs de titres où le gamertag a un profil dans
// db_profiles.json (= titres où il a des données → titres à seeder dans la démo).
// Le titre par défaut est placé en TÊTE s'il est présent ; les autres sont triés
// (ordre stable). v2.1 plat → uniquement le titre par défaut.
func TitlesForGamertag(profilesPath, gamertag string) ([]string, error) {
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return nil, fmt.Errorf("read profiles: %w", err)
	}
	var v3 struct {
		Version  string                             `json:"version"`
		Profiles map[string]map[string]profileEntry `json:"profiles"`
	}
	if json.Unmarshal(data, &v3) == nil && v3.Version >= "3.0" {
		var others []string
		hasDefault := false
		for slug, byGT := range v3.Profiles {
			if _, ok := byGT[gamertag]; !ok {
				continue
			}
			if slug == titlePkg.DefaultSlug {
				hasDefault = true
				continue
			}
			others = append(others, slug)
		}
		sort.Strings(others)
		var out []string
		if hasDefault {
			out = append(out, titlePkg.DefaultSlug)
		}
		out = append(out, others...)
		if len(out) == 0 {
			return nil, fmt.Errorf("gamertag %q absent de tous les titres dans %s", gamertag, profilesPath)
		}
		return out, nil
	}
	// v2.1 plat : titre défaut seulement.
	return []string{titlePkg.DefaultSlug}, nil
}

// ResolveSourceXUIDForTitle lit db_profiles.json et retourne le xuid + le chemin
// player DB pour le couple (titleSlug, gamertag).
//
// v3.0 (nested par titre) : profiles.{slug}.{gamertag}.{xuid,db_path}.
// v2.1 (plat, legacy)     : uniquement valide pour le titre par défaut → délègue à
//
//	ResolveSourceXUIDFromProfiles (compat tests/anciens db_profiles).
func ResolveSourceXUIDForTitle(profilesPath, titleSlug, gamertag string) (xuid, playerDBPath string, err error) {
	data, rerr := os.ReadFile(profilesPath)
	if rerr != nil {
		return "", "", fmt.Errorf("read profiles: %w", rerr)
	}
	var v3 struct {
		Version  string                             `json:"version"`
		Profiles map[string]map[string]profileEntry `json:"profiles"`
	}
	if json.Unmarshal(data, &v3) == nil && v3.Version >= "3.0" {
		byGT, ok := v3.Profiles[titleSlug]
		if !ok {
			return "", "", fmt.Errorf("titre %q absent de %s", titleSlug, profilesPath)
		}
		p, ok := byGT[gamertag]
		if !ok {
			return "", "", fmt.Errorf("gamertag %q absent du titre %q dans %s", gamertag, titleSlug, profilesPath)
		}
		if p.XUID == "" {
			return "", "", fmt.Errorf("profile %q/%q sans xuid", titleSlug, gamertag)
		}
		return p.XUID, p.DBPath, nil
	}
	// Fallback v2.1 (plat) : seul le titre par défaut est représentable.
	if titleSlug == titlePkg.DefaultSlug {
		return ResolveSourceXUIDFromProfiles(profilesPath, gamertag)
	}
	return "", "", fmt.Errorf("db_profiles v2.1 (plat) ne porte pas le titre %q", titleSlug)
}

// resolvePlayerDBByXUIDForTitle retourne le chemin DB relatif du joueur ayant ce xuid
// SOUS LE TITRE donné (strict — pas de fallback cross-titre : un coéquipier absent du
// titre courant est ignoré, plutôt que de seeder ses stats d'un autre titre). v2.1
// plat → délègue à resolvePlayerDBByXUID (titre défaut uniquement).
func resolvePlayerDBByXUIDForTitle(profilesPath, titleSlug, xuid string) (dbRelPath string, found bool, err error) {
	data, rerr := os.ReadFile(profilesPath)
	if rerr != nil {
		return "", false, fmt.Errorf("read profiles: %w", rerr)
	}
	var v3 struct {
		Version  string                             `json:"version"`
		Profiles map[string]map[string]profileEntry `json:"profiles"`
	}
	if json.Unmarshal(data, &v3) == nil && v3.Version >= "3.0" {
		byGT, ok := v3.Profiles[titleSlug]
		if !ok {
			return "", false, nil
		}
		for _, p := range byGT {
			if p.XUID == xuid && p.DBPath != "" {
				return p.DBPath, true, nil
			}
		}
		return "", false, nil
	}
	// v2.1 plat → uniquement titre défaut.
	if titleSlug == titlePkg.DefaultSlug {
		return resolvePlayerDBByXUID(profilesPath, xuid)
	}
	return "", false, nil
}

// writeDemoConfigsV3 écrit db_profiles.json v3.0 (profiles.{slug}.{gamertag}) + app_settings.json
// à la racine démo, agrégeant les player DB seedées de tous les titres.
func writeDemoConfigsV3(outDir string, byTitle map[string][]seededDemoPlayer, serviceTag string, mediaEnabled bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	profiles := make(map[string]any, len(byTitle))
	for slug, players := range byTitle {
		byGT := make(map[string]any, len(players))
		for _, s := range players {
			byGT[s.Gamertag] = map[string]any{
				"db_path":         demoPlayerDBRelPath(slug, s.Dir),
				"xuid":            s.XUID,
				"waypoint_player": s.Gamertag,
			}
		}
		profiles[slug] = byGT
	}
	doc := map[string]any{
		"version":        "3.0",
		"admin":          DefaultDemoMainGamertag,
		"warehouse_path": "data/warehouse",
		"metadata_db":    "data/warehouse/metadata.duckdb",
		"profiles":       profiles,
	}
	if err := writeJSONFile(filepath.Join(outDir, "db_profiles.json"), doc); err != nil {
		return fmt.Errorf("db_profiles.json: %w", err)
	}
	if err := writeJSONFile(filepath.Join(outDir, "app_settings.json"), demoAppSettings(serviceTag, mediaEnabled)); err != nil {
		return fmt.Errorf("app_settings.json: %w", err)
	}
	return nil
}
