// Package config — player_resolver.go : résolution slug → PlayerDB.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// ErrPlayerNotFound est retourné quand le slug ne correspond à aucun joueur configuré.
var ErrPlayerNotFound = fmt.Errorf("joueur introuvable")

// sharedReaderForTitle résout le SharedReader (provider RO B-swap) du shared
// d'un titre donné, pour injection dans un PlayerPoolConfig.
//
// Pourquoi per-titre : en B-swap, cfg.SharedProvider est le provider du shared
// Halo Infinite (DefaultSlug) UNIQUEMENT. Un joueur d'un autre titre (Halo 5+)
// doit lire SON propre shared, sinon il lit le mauvais fichier. On résout donc
// le provider du path per-titre via le Manager (caché par path absolu).
//
// Garanties :
//   - DefaultSlug : SharedDBPath(DefaultSlug) == le path passé au boot →
//     Manager.For() retourne le MÊME provider que cfg.SharedProvider
//     (byte-identique, aucune nouvelle conn DuckDB ouverte).
//   - Mode legacy / kill-switch (SharedManager == nil) : fallback direct sur
//     cfg.SharedProvider (nil-safe — le pool repasse alors en mode legacy).
//   - Mode démo : le shared démo est mono-titre (HINF) et déjà ouvert par le
//     provider boot ; on conserve cfg.SharedProvider (le path démo réécrit au
//     boot ne correspond pas toujours à SharedDBPath title-aware).
//   - Échec d'ouverture per-titre (fichier shared du titre absent) : fallback
//     cfg.SharedProvider + warn. Le pool tolère un SharedReader pointant le
//     mauvais titre mieux qu'un crash de résolution (lecture vide vs panique).
func (cfg *AppConfig) sharedReaderForTitle(titleSlug string) duckdb.SharedReader {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	live := cfg.resolveLiveSharedReader(titleSlug)
	// Phase 4 : enveloppe dans le reader snapshot-préféré (lecture découplée du B-swap)
	// si câblé. Fallback live interne → jamais de régression si aucun snapshot. Pas en
	// démo (aucun snapshot produit, le wrap ne ferait que des fallbacks inutiles).
	if cfg.SnapshotReaderWrapper != nil && !cfg.DemoMode && live != nil {
		return cfg.SnapshotReaderWrapper(titleSlug, live)
	}
	return live
}

// resolveLiveSharedReader résout le SharedReader LIVE (provider B-swap) d'un titre.
func (cfg *AppConfig) resolveLiveSharedReader(titleSlug string) duckdb.SharedReader {
	// Mode legacy/kill-switch ou démo : pas de routage per-titre.
	if cfg.SharedManager == nil || cfg.DemoMode {
		return cfg.SharedProvider
	}
	path := title.NewPathResolver(cfg.RepoRoot).SharedDBPath(titleSlug)
	provider, err := cfg.SharedManager.For(path, cfg.UserTimezone)
	if err != nil {
		slog.Warn("sharedReaderForTitle: ouverture provider per-titre échouée, fallback Infinite",
			"title", titleSlug, "path", path, "err", err)
		return cfg.SharedProvider
	}
	return provider
}

// ResolvePlayer traduit un slug joueur en PlayerDB prêt à l'emploi.
// titleSlug est le titre courant (ex: "halo_infinite"). Si vide, DefaultSlug est utilisé.
// En mode démo, les fixtures de test sont utilisées (slug ignoré).
func ResolvePlayer(ctx context.Context, cfg *AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.DemoMode {
		return resolveDemoPlayer(ctx, cfg, slug, titleSlug)
	}
	return resolveRealPlayer(ctx, cfg, slug, titleSlug)
}

// demoPlayerDef décrit un joueur démo du stack (DemoPlayer + 2 coéquipiers).
type demoPlayerDef struct {
	Slug     string
	Gamertag string
	XUID     string
	Dir      string // sous-répertoire fixtures : DEMO, DEMO2, DEMO3
}

// DemoRoster : les 3 joueurs démo fixes. Cohérent avec seed-demo
// (demoXUIDForIndex / demoDirForIndex / DemoPlayer{,2,3}). Le main "demo-player"
// est l'index 0 ; les 2 coéquipiers principaux sont 2 et 3.
var DemoRoster = []demoPlayerDef{
	{Slug: "demo-player", Gamertag: "DemoPlayer", XUID: "0000000000000000", Dir: "DEMO"},
	{Slug: "demo-player-2", Gamertag: "DemoPlayer2", XUID: "0000000000000001", Dir: "DEMO2"},
	{Slug: "demo-player-3", Gamertag: "DemoPlayer3", XUID: "0000000000000002", Dir: "DEMO3"},
}

// demoDefForSlug retourne la def démo pour un slug OU un gamertag (résolution
// squad par gamertag), insensible à la casse. Fallback : le main (index 0).
func demoDefForSlug(slug string) demoPlayerDef {
	for _, d := range DemoRoster {
		if strings.EqualFold(d.Slug, slug) || strings.EqualFold(d.Gamertag, slug) {
			return d
		}
	}
	return DemoRoster[0]
}

// resolveDemoPlayer ouvre le joueur de démo depuis DemoFixturesDir.
//
// Supporte deux arborescences :
//   - Structure seed-demo (LEVELUP_DEMO_FIXTURES_DIR=data/demo) :
//     data/demo/players/DEMO/stats.duckdb
//     data/demo/warehouse/shared_matches_v2.duckdb
//     data/demo/warehouse/metadata.duckdb
//   - Structure plate legacy (tests/fixtures/ref_player/) :
//     stats.duckdb / shared_matches_v2.duckdb / metadata.duckdb à la racine
//
// En cas de fixture absente, retourne une erreur explicite avec la commande
// pour regénérer (go run ./cmd/levelup seed-demo).
func resolveDemoPlayer(ctx context.Context, cfg *AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	dir := cfg.DemoFixturesDir
	def := demoDefForSlug(slug) // DemoPlayer (main) ou un coéquipier DemoPlayer2/3

	// Résolution du chemin stats.duckdb : structure seed-demo (par joueur du
	// roster) d'abord, plate (fixtures legacy, main only) ensuite.
	statsPath := filepath.Join(dir, "players", def.Dir, "stats.duckdb")
	sharedPath := filepath.Join(dir, "warehouse", "shared_matches_v2.duckdb")
	metaPath := filepath.Join(dir, "warehouse", "metadata.duckdb")
	xuidBytes := def.XUID
	gamertag := def.Gamertag

	if _, err := os.Stat(statsPath); os.IsNotExist(err) {
		// Fallback : structure plate (fixtures commitées dans tests/) — main only.
		statsPath = filepath.Join(dir, "stats.duckdb")
		sharedPath = filepath.Join(dir, "shared_matches_v2.duckdb")
		metaPath = filepath.Join(dir, "metadata.duckdb")
		gamertag = "DemoPlayer"
		if xb, e := readXUIDFile(filepath.Join(dir, "xuid.txt")); e == nil {
			xuidBytes = xb
		}
	}

	if _, err := os.Stat(statsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"resolveDemoPlayer: fixture démo manquante (%s). "+
				"Générer avec : go run ./cmd/levelup seed-demo --gamertag JGtm (sortie dans data/demo). "+
				"Puis définir LEVELUP_DEMO_FIXTURES_DIR=data/demo.",
			statsPath,
		)
	}

	// SharedSocial démo : seed-demo produit data/demo/warehouse/shared_social.duckdb
	// au schéma canonique (migrations TargetSharedSocial). Le pipeline média EXIGE un
	// SharedSocial non-nil (sinon 0 média, pas de fallback Player DB — cf.
	// media_repo_q37_pipeline.go). Vide si la fixture est absente (démo legacy).
	sharedSocialPath := filepath.Join(dir, "warehouse", "shared_social.duckdb")
	if _, err := os.Stat(sharedSocialPath); os.IsNotExist(err) {
		sharedSocialPath = ""
	}
	pcfg := duckdb.PlayerPoolConfig{
		Gamertag:           gamertag,
		XUID:               xuidBytes,
		TitleSlug:          titleSlug,
		PlayerDBPath:       statsPath,
		SharedDBPath:       sharedPath,
		MetaDBPath:         metaPath,
		SharedSocialDBPath: sharedSocialPath,
		UserTimezone:       cfg.UserTimezone,
		// Per-titre (B-swap multi-titre) : provider du shared du TITRE courant.
		// DefaultSlug → byte-identique à cfg.SharedProvider (caché par path).
		SharedReader: cfg.sharedReaderForTitle(titleSlug),
	}
	return duckdb.GetOrOpen(ctx, pcfg)
}

// resolveRealPlayer trouve le joueur par slug dans db_profiles.json.
func resolveRealPlayer(ctx context.Context, cfg *AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	players, err := cfg.LoadPlayers(titleSlug)
	if err != nil {
		return nil, fmt.Errorf("ResolvePlayer: %w", err)
	}

	var found *domain.PlayerSummary
	for i := range players {
		if players[i].PlayerSlug == slug {
			found = &players[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%w: slug=%q", ErrPlayerNotFound, slug)
	}

	pcfg := buildPoolConfig(cfg, found, titleSlug)
	return duckdb.GetOrOpen(ctx, pcfg)
}

// buildPoolConfig construit un PlayerPoolConfig depuis un PlayerSummary.
// Utilise le PathResolver pour résoudre les chemins title-aware.
func buildPoolConfig(cfg *AppConfig, p *domain.PlayerSummary, titleSlug string) duckdb.PlayerPoolConfig {
	pr := title.NewPathResolver(cfg.RepoRoot)
	return duckdb.PlayerPoolConfig{
		Gamertag:           p.Gamertag,
		XUID:               p.XUID,
		TitleSlug:          titleSlug,
		PlayerDBPath:       pr.PlayerDBPath(titleSlug, p.Gamertag),
		SharedDBPath:       pr.SharedDBPath(titleSlug),
		MetaDBPath:         pr.MetadataDBPath(titleSlug),
		SharedSocialDBPath: pr.SharedSocialDBPath(titleSlug),
		UserTimezone:       cfg.UserTimezone,
		// Per-titre (B-swap multi-titre) : provider du shared du TITRE courant.
		// DefaultSlug → byte-identique à cfg.SharedProvider (caché par path).
		SharedReader: cfg.sharedReaderForTitle(titleSlug),
	}
}

// SharedDBPath retourne le chemin vers la base DuckDB partagée.
// titleSlug : titre courant, vide = halo_infinite.
func SharedDBPath(cfg *AppConfig, titleSlug string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.DemoMode {
		return filepath.Join(cfg.DemoFixturesDir, "shared_matches_v2.duckdb")
	}
	return title.NewPathResolver(cfg.RepoRoot).SharedDBPath(titleSlug)
}

// PlayerDBPath retourne le chemin vers la base DuckDB stats d'un joueur.
// titleSlug : titre courant, vide = halo_infinite.
func PlayerDBPath(cfg *AppConfig, titleSlug, gamertag string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return title.NewPathResolver(cfg.RepoRoot).PlayerDBPath(titleSlug, gamertag)
}

// readXUIDFile lit le XUID depuis un fichier texte (1 ligne).
func readXUIDFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	xuid := strings.TrimSpace(string(data))
	if xuid == "" {
		return "", fmt.Errorf("xuid.txt vide : %s", path)
	}
	return xuid, nil
}
