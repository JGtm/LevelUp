// Package playerresolve ouvre le PlayerDB d'un joueur à partir de la configuration (ResolvePlayer) et résout le
// lecteur de la base partagée d'un titre. C'est du câblage : il compose config (chemins, profils suivis, mode
// démo) et platform/duckdb (pool de PlayerDB, fournisseur B-swap), que config n'importe plus.
package playerresolve

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// SharedReaderForTitle est le wrapper exporté de sharedReaderForTitle : résout le
// SharedReader RO (B-swap) du shared du titre courant. Utilisé au composition root
// pour un décompte de matchs title-aware (bootstrap) sans que le package service
// n'importe platform/duckdb. DefaultSlug → byte-identique à cfg.SharedProvider.
func SharedReaderForTitle(cfg *config.AppConfig, titleSlug string) duckdb.SharedReader {
	return sharedReaderForTitle(cfg, titleSlug)
}

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
//   - Sans Manager (SharedManager == nil : outils en ligne de commande, tests) : fallback direct sur
//     cfg.SharedProvider (nil-safe — le pool repasse alors en mode legacy).
//   - Mode démo : le shared du titre PAR DÉFAUT est celui ouvert au boot (provider
//     boot, byte-identique) ; un titre ADDITIONNEL (Halo 5…) résout le provider de
//     SON shared démo title-scopé (data/demo/titles/{slug}/warehouse/…).
//   - Échec d'ouverture per-titre (fichier shared du titre absent) : fallback
//     cfg.SharedProvider + warn. Le pool tolère un SharedReader pointant le
//     mauvais titre mieux qu'un crash de résolution (lecture vide vs panique).
func sharedReaderForTitle(cfg *config.AppConfig, titleSlug string) duckdb.SharedReader {
	// Sans Manager : pas de routage per-titre.
	if cfg.SharedManager == nil {
		return cfg.SharedProvider
	}
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	// Démo : titre par défaut = provider boot (shared démo plat, byte-identique) ;
	// titre additionnel = provider du shared démo title-scopé.
	if cfg.DemoState() {
		if title.IsDefaultSlug(titleSlug) {
			return cfg.SharedProvider
		}
		return sharedProviderForPath(cfg, titleSlug, cfg.DemoLayout().SharedDBPath(titleSlug))
	}
	return sharedProviderForPath(cfg, titleSlug, cfg.PathResolver().SharedDBPath(titleSlug))
}

// sharedProviderForPath résout (via le Manager, caché par path) le SharedReader d'un
// shared à un chemin donné. Fallback cfg.SharedProvider + warn si l'ouverture échoue.
func sharedProviderForPath(cfg *config.AppConfig, titleSlug, path string) duckdb.SharedReader {
	provider, err := cfg.SharedManager.For(path, cfg.UserTimezone)
	if err != nil {
		slog.Warn("sharedReaderForTitle: ouverture provider per-titre échouée, fallback défaut",
			"title", titleSlug, "path", path, "err", err)
		return cfg.SharedProvider
	}
	return provider
}

// ResolvePlayer traduit un slug joueur en PlayerDB prêt à l'emploi.
// titleSlug est le titre courant (ex: "halo_infinite"). Si vide, DefaultSlug est utilisé.
// Sur le serveur démo (demoState), les fixtures du roster démo sont utilisées ; un binaire
// opérateur chargé par Load() résout toujours un vrai profil, même sous LEVELUP_DEMO_MODE=true.
func ResolvePlayer(ctx context.Context, cfg *config.AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.DemoState() {
		return resolveDemoPlayer(ctx, cfg, slug, titleSlug)
	}
	return resolveRealPlayer(ctx, cfg, slug, titleSlug)
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
func resolveDemoPlayer(ctx context.Context, cfg *config.AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	dir := cfg.DemoFixturesDir
	def := config.DemoDefForSlug(slug) // DemoPlayer (main) ou un coéquipier DemoPlayer2/3

	// Résolution du chemin stats.duckdb : structure seed-demo (par joueur du
	// roster, title-scopée) d'abord, plate (fixtures legacy, main only) ensuite.
	statsPath := cfg.DemoLayout().PlayerDBPath(titleSlug, def.Dir)
	sharedPath := cfg.DemoLayout().SharedDBPath(titleSlug)
	metaPath := cfg.DemoLayout().MetadataDBPath(titleSlug)
	xuidBytes := def.XUID
	gamertag := def.Gamertag

	// Fallback structure plate (fixtures commitées dans tests/) — RÉSERVÉ au titre
	// par défaut (un titre additionnel sans fixture ne doit JAMAIS retomber sur les
	// DB du titre par défaut : on préfère l'erreur explicite ci-dessous).
	if _, err := os.Stat(statsPath); os.IsNotExist(err) && title.IsDefaultSlug(titleSlug) {
		statsPath = filepath.Join(dir, "stats.duckdb")
		sharedPath = filepath.Join(dir, "shared_matches_v2.duckdb")
		metaPath = filepath.Join(dir, "metadata.duckdb")
		gamertag = "DemoPlayer"
		if xb, e := config.ReadXUIDFile(filepath.Join(dir, "xuid.txt")); e == nil {
			xuidBytes = xb
		}
	}

	if _, err := os.Stat(statsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"resolveDemoPlayer: fixture démo manquante (%s, titre=%s). "+
				"Générer avec : go run ./cmd/levelup seed-demo --gamertag JGtm (sortie dans data/demo). "+
				"Puis définir LEVELUP_DEMO_FIXTURES_DIR=data/demo",
			statsPath, titleSlug,
		)
	}

	// SharedSocial démo (title-scopé) : seed-demo produit shared_social.duckdb au
	// schéma canonique (migrations TargetSharedSocial). Le pipeline média EXIGE un
	// SharedSocial non-nil (sinon 0 média, pas de fallback Player DB — cf.
	// media_repo_q37_pipeline.go). Vide si la fixture est absente (titre sans média).
	sharedSocialPath := cfg.DemoLayout().SharedSocialDBPath(titleSlug)
	if _, err := os.Stat(sharedSocialPath); os.IsNotExist(err) {
		sharedSocialPath = ""
	}
	pcfg := duckdb.PlayerPoolConfig{
		Gamertag:           gamertag,
		XUID:               xuidBytes,
		TitleSlug:          titleSlug,
		ModeSource:         title.DefaultRegistry().ModeSourceFor(titleSlug),
		PlayerDBPath:       statsPath,
		SharedDBPath:       sharedPath,
		MetaDBPath:         metaPath,
		SharedSocialDBPath: sharedSocialPath,
		UserTimezone:       cfg.UserTimezone,
		// Per-titre (B-swap multi-titre) : provider du shared du TITRE courant.
		// DefaultSlug → byte-identique à cfg.SharedProvider (caché par path).
		SharedReader: sharedReaderForTitle(cfg, titleSlug),
	}
	return duckdb.GetOrOpen(ctx, pcfg)
}

// resolveRealPlayer trouve le joueur par slug dans db_profiles.json.
func resolveRealPlayer(ctx context.Context, cfg *config.AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
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
		return nil, fmt.Errorf("%w: slug=%q", config.ErrPlayerNotFound, slug)
	}

	pcfg := buildPoolConfig(cfg, found, titleSlug)
	return duckdb.GetOrOpen(ctx, pcfg)
}

// buildPoolConfig construit un PlayerPoolConfig depuis un PlayerSummary.
// Utilise le PathResolver pour résoudre les chemins title-aware.
func buildPoolConfig(cfg *config.AppConfig, p *domain.PlayerSummary, titleSlug string) duckdb.PlayerPoolConfig {
	pr := cfg.PathResolver()
	return duckdb.PlayerPoolConfig{
		Gamertag:           p.Gamertag,
		XUID:               p.XUID,
		TitleSlug:          titleSlug,
		ModeSource:         title.DefaultRegistry().ModeSourceFor(titleSlug),
		PlayerDBPath:       pr.PlayerDBPath(titleSlug, p.Gamertag),
		SharedDBPath:       pr.SharedDBPath(titleSlug),
		MetaDBPath:         pr.MetadataDBPath(titleSlug),
		SharedSocialDBPath: pr.SharedSocialDBPath(titleSlug),
		UserTimezone:       cfg.UserTimezone,
		// Per-titre (B-swap multi-titre) : provider du shared du TITRE courant.
		// DefaultSlug → byte-identique à cfg.SharedProvider (caché par path).
		SharedReader: sharedReaderForTitle(cfg, titleSlug),
	}
}
