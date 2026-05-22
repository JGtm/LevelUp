// Package duckdb — PlayerPool : pool de connexions DuckDB par joueur.
//
// Architecture :
//   - Player : stats.duckdb en lecture/écriture (sync engine et handlers HTTP) avec shared attaché.
//   - ReadDB() retourne Player (connexion unique — DuckDB interdit RW+RO sur le même fichier).
//   - Shared : shared_matches_v2.duckdb en lecture seule.
//   - Metadata : metadata.duckdb en lecture seule (PersistSink ouvre sa propre connexion RW).
//   - SharedSocial : shared_social.duckdb en lecture/écriture (médias, likes, favoris).
//   - Les migrations shared sont exécutées par runMigrations() dans main.go, pas ici.
//   - Le pool est global au processus et threadsafe (sync.Map + singleflight).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"levelup/go-api/internal/migration"

	"golang.org/x/sync/singleflight"
)

// PlayerPoolConfig contient les chemins nécessaires pour ouvrir un PlayerDB.
type PlayerPoolConfig struct {
	Gamertag                string
	XUID                    string
	TitleSlug               string // Sprint 44 : namespace titre (ex: "halo_infinite")
	PlayerDBPath            string
	SharedDBPath            string
	MetaDBPath              string
	SharedSocialDBPath      string // shared_social.duckdb (médias, likes, favoris)
	GlobalXuidAliasesDBPath string // P5.3 : data/global/xbox_aliases.duckdb (mapping xuid→gamertag global Microsoft)
	UserTimezone            string // timezone IANA pour la lecture des TIMESTAMP (ex: "Europe/Paris")

	// SharedReader (sprint sharedprovider, commit 8a) : si non-nil, expose ce
	// SharedReader dans PlayerDB.SharedReader. Sinon (mode legacy par défaut),
	// un wrapper LegacySharedReader(pdb.Shared) est utilisé.
	//
	// Au commit 8k (retrait de attachShared et de PlayerDB.Shared), ce champ
	// devra être obligatoire pour le mode B-swap. En attendant, le caller
	// peut injecter un sharedprovider.Provider (qui satisfait l'interface
	// SharedReader structurellement) sans cycle d'import.
	SharedReader SharedReader
}

// PoolKey retourne la clé unique du pool pour ce joueur.
// Format : "{title_slug}:{gamertag}" ou "{gamertag}" si TitleSlug vide (legacy).
func (c PlayerPoolConfig) PoolKey() string {
	if c.TitleSlug != "" {
		return c.TitleSlug + ":" + c.Gamertag
	}
	return c.Gamertag
}

// PlayerDB regroupe les connexions DB nécessaires pour un joueur.
type PlayerDB struct {
	Player       *DB // stats.duckdb du joueur (RW, avec shared attaché) — sync engine et handlers HTTP
	Shared       *DB // shared_matches_v2.duckdb
	SharedSocial *DB // shared_social.duckdb (médias, likes, favoris de matchs)
	Metadata     *DB // metadata.duckdb (RO)

	// sharedDBPath (commit 8f) — chemin du fichier shared, stocké pour permettre
	// le re-OpenReadOnly + re-attachShared après un swap RW du Provider.
	sharedDBPath string
	// userTimezone stocké aussi pour les ré-ouvertures.
	userTimezone string
	// bSwapEnabled (commit 8f) : true si cfg.SharedReader != nil au moment
	// de openPlayerDB (mode B-swap). False = mode legacy avec
	// LegacySharedReader(pdb.Shared). Permet à Prepare/Restore d'être
	// no-op en mode legacy.
	bSwapEnabled bool

	// SharedReader (commit 8a) sert les lectures shared via un contrat
	// uniforme : Get(ctx) (*sql.DB, releaseFn, error).
	//
	// Initialisé dans openPlayerDB :
	//   - Si cfg.SharedReader != nil : utilise le SharedReader injecté
	//     (typiquement sharedprovider.Provider en mode B-swap).
	//   - Sinon : LegacySharedReader(pdb.Shared) — wrapper no-op autour de
	//     la conn RO classique. Comportement identique à pre-commit-8a.
	//
	// Les repos migrés aux commits 8c+ consommeront ce champ au lieu de
	// PlayerDB.Shared directement. Au commit 8k (retrait de attachShared),
	// seul ce champ subsistera pour les lectures shared.
	SharedReader SharedReader

	XUID      string
	Gamertag  string
	TitleSlug string // Sprint 44 : titre associé
}

// ReadDB retourne la connexion de lecture (Player RW — connexion unique par joueur).
func (pdb *PlayerDB) ReadDB() *DB {
	return pdb.Player
}

// SharedReadDB retourne le SharedReader effectif, avec fallback automatique
// sur LegacySharedReader(pdb.Shared) si SharedReader n'est pas initialisé
// (cas des tests qui construisent un PlayerDB manuellement sans passer par
// openPlayerDB).
//
// Les repos migrés (commits 8c+) DOIVENT utiliser cette méthode au lieu
// d'accéder directement à pdb.SharedReader, pour rester compatibles avec
// les tests existants. Au commit 8k (retrait de pdb.Shared), cette méthode
// deviendra inutile — pdb.SharedReader sera toujours non-nil.
func (pdb *PlayerDB) SharedReadDB() SharedReader {
	if pdb.SharedReader != nil {
		return pdb.SharedReader
	}
	return LegacySharedReader(pdb.Shared)
}

// GlobalPool est le registre process-level des PlayerDB par gamertag (slug).
var globalPool sync.Map // map[string]*PlayerDB

// sfGroup empêche les créations concurrentes de pools pour un même joueur.
var sfGroup singleflight.Group

// GetOrOpen retourne le PlayerDB existant pour ce joueur, ou l'ouvre.
// Thread-safe : singleflight garantit qu'un seul openPlayerDB par clé de pool.
// Clé de pool : "{title_slug}:{gamertag}" (Sprint 44) ou "{gamertag}" (legacy).
func GetOrOpen(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	key := cfg.PoolKey()
	if pdb, ok := globalPool.Load(key); ok {
		return pdb.(*PlayerDB), nil
	}

	// singleflight : une seule goroutine ouvre; les autres attendent le résultat.
	result, err, _ := sfGroup.Do(key, func() (interface{}, error) {
		// Vérifier à nouveau après avoir gagné le lock singleflight.
		if pdb, ok := globalPool.Load(key); ok {
			return pdb.(*PlayerDB), nil
		}
		pdb, openErr := openPlayerDB(ctx, cfg)
		if openErr != nil {
			return nil, openErr
		}
		globalPool.Store(key, pdb)
		return pdb, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*PlayerDB), nil
}

// LookupFromPool retourne le PlayerDB existant pour une clé de pool donnée.
// Format de la clé : "{title_slug}:{gamertag}" (ou "{gamertag}" pour legacy).
// Retourne nil, false si le joueur n'est pas ouvert dans le pool.
func LookupFromPool(key string) (*PlayerDB, bool) {
	v, ok := globalPool.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*PlayerDB), true
}

// IteratePool parcourt tous les PlayerDB ouverts dans le pool.
// La fonction f reçoit chaque PlayerDB ; retourner false arrête l'itération.
func IteratePool(f func(*PlayerDB) bool) {
	globalPool.Range(func(_, value any) bool {
		return f(value.(*PlayerDB))
	})
}

// CloseAll ferme toutes les connexions du pool. À appeler au shutdown.
func CloseAll() {
	globalPool.Range(func(key, value any) bool {
		pdb := value.(*PlayerDB)
		_ = pdb.Player.Close()

		if pdb.Shared != nil {
			_ = pdb.Shared.Close()
		}
		if pdb.SharedSocial != nil {
			_ = pdb.SharedSocial.Close()
		}
		_ = pdb.Metadata.Close()
		globalPool.Delete(key)
		return true
	})
}

// openPlayerDB ouvre et initialise un PlayerDB complet. Boot per-player :
// migrations + open RW + ATTACH shared/social/metadata + SharedReader bswap vs
// legacy + xuid_aliases global optionnel. Splitter casserait l'atomicité du
// boot (cleanup en cascade sur erreur).
//
//nolint:gocyclo // assemblage boot avec cleanup en cascade, cohésion requise.
func openPlayerDB(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	if err := ensurePlayerDBMigrations(cfg.PlayerDBPath); err != nil {
		return nil, fmt.Errorf("pool: migrate player db %s: %w", cfg.Gamertag, err)
	}

	playerDB, err := OpenReadWrite(cfg.PlayerDBPath, cfg.UserTimezone)
	if err != nil {
		return nil, fmt.Errorf("pool: open player db %s: %w", cfg.Gamertag, err)
	}

	// pdb.Shared : conn RO directe sur shared_matches_v2.duckdb. Utilisée
	// UNIQUEMENT en mode legacy (cfg.SharedReader nil) où LegacySharedReader
	// wrap cette conn pour servir l'interface SharedReader. En mode B-swap
	// (cfg.SharedReader != nil), le Provider possède son propre handle et
	// pdb.Shared n'est jamais consulté — on évite donc de l'ouvrir (sinon
	// son file handle bloquerait le swap RW du Provider).
	var sharedDB *DB
	if cfg.SharedReader == nil {
		sharedDB, err = OpenReadOnly(cfg.SharedDBPath, cfg.UserTimezone)
		if err != nil {
			_ = playerDB.Close()
			return nil, fmt.Errorf("pool: open shared db: %w", err)
		}
	}
	// Les migrations shared sont gérées par runMigrations() dans main.go.

	// metadata.duckdb est ouverte en RW partagé : le DuckDBIndexStore (assets) a besoin d'y
	// écrire (table asset_index). Les deux utilisent la clé cache "rw:path", donc une seule
	// sql.DB est créée. OpenReadWriteShared garantit maxOpenConns=4 pour les lectures concurrentes.
	metaDB, err := OpenReadWriteShared(cfg.MetaDBPath, cfg.UserTimezone)
	if err != nil {
		_ = playerDB.Close()
		if sharedDB != nil {
			_ = sharedDB.Close()
		}
		return nil, fmt.Errorf("pool: open metadata db: %w", err)
	}

	// SharedSocial est optionnel : absent si le fichier n'existe pas encore.
	// Ouvert en OpenReadWriteShared (maxOpenConns=4) — comme metadata.duckdb :
	// lectures concurrentes (handlers HTTP media/likes/favoris/ascension) +
	// écritures occasionnelles (post-sync records, INSERTs unitaires). Tous les
	// autres call sites runtime (registry_notifications, post_sync_deltas,
	// prestige_setup) doivent aussi utiliser OpenReadWriteShared pour partager
	// la même clé cache "rw:path" et éviter "different configuration" errors
	// (mix avec OpenReadOnly historiquement, fixé en commit 21).
	var socialDB *DB
	if cfg.SharedSocialDBPath != "" {
		socialDB, err = OpenReadWriteShared(cfg.SharedSocialDBPath, cfg.UserTimezone)
		if err != nil {
			// Non bloquant : la DB sera créée lors de la prochaine migration.
			// On log Warn pour garder une trace (l'absence du fichier au boot est
			// normale, mais une erreur ouverture sur fichier existant indique
			// verrou / corruption qu'on veut voir).
			slog.WarnContext(ctx, "pool: ouverture SharedSocial échouée (dégradation: socialDB=nil)",
				"path", cfg.SharedSocialDBPath, "gamertag", cfg.Gamertag, "err", err)
			socialDB = nil
		} else {
			// Appliquer les migrations shared_social (idempotentes).
			if mErr := migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial); mErr != nil {
				slog.ErrorContext(ctx, "pool: migrations SharedSocial échouées",
					"path", cfg.SharedSocialDBPath, "gamertag", cfg.Gamertag, "err", mErr)
			}
		}
	}

	// attachShared sur la conn player a été retiré (ADR 0016, commit 9c.5).
	// Toutes les queries shared passent désormais par SharedReader (Provider en
	// prod, LegacySharedReader pour les tests sans Provider injecté).
	// Sprint P0→P7 (2026-05-20) a finalisé la migration de tous les repos
	// applicatifs (media, post-sync, engagement, profile, career, explorer,
	// sessions, stats, leaderboard, exclusions, match_view).

	// P5.3 : ATTACH global xbox_aliases (mapping xuid→gamertag global Microsoft).
	// Non-bloquant : si la DB globale n'existe pas encore (avant migration), les
	// requêtes qui font `JOIN global.xuid_aliases` retomberont sur NULL.
	if cfg.GlobalXuidAliasesDBPath != "" {
		if err := attachGlobalXuidAliases(ctx, playerDB, cfg.GlobalXuidAliasesDBPath); err != nil {
			// Non bloquant : si l'attach échoue, les queries `JOIN global.xuid_aliases`
			// retomberont sur NULL (tolérance prévue côté query).
			slog.WarnContext(ctx, "pool: ATTACH global xbox_aliases échoué (player conn)",
				"path", cfg.GlobalXuidAliasesDBPath, "gamertag", cfg.Gamertag, "err", err)
		}
	}

	// attachShared sur SharedSocial retiré aussi.
	// media_repo passe désormais entièrement par SharedReader pour les queries
	// shared.* — plus aucune conn du pool ne porte d'ATTACH shared.
	// P5.3 : ATTACH global xbox_aliases sur SharedSocial aussi (media_repo fait
	// `JOIN global.xuid_aliases` sur les likers de médias).
	if socialDB != nil && cfg.GlobalXuidAliasesDBPath != "" {
		if err := attachGlobalXuidAliases(ctx, socialDB, cfg.GlobalXuidAliasesDBPath); err != nil {
			slog.WarnContext(ctx, "pool: ATTACH global xbox_aliases échoué (social conn)",
				"path", cfg.GlobalXuidAliasesDBPath, "gamertag", cfg.Gamertag, "err", err)
		}
	}

	// SharedReader (commit 8a) : par défaut, wrap pdb.Shared en mode legacy.
	// Si cfg.SharedReader fourni (mode B-swap), on l'utilise tel quel — le
	// caller a déjà arbitré le mode.
	sharedReader := cfg.SharedReader
	bSwapEnabled := sharedReader != nil
	if sharedReader == nil {
		sharedReader = LegacySharedReader(sharedDB)
	}

	return &PlayerDB{
		Player:       playerDB,
		Shared:       sharedDB,
		SharedSocial: socialDB,
		Metadata:     metaDB,
		sharedDBPath: cfg.SharedDBPath,
		userTimezone: cfg.UserTimezone,
		bSwapEnabled: bSwapEnabled,
		SharedReader: sharedReader,
		XUID:         cfg.XUID,
		Gamertag:     cfg.Gamertag,
		TitleSlug:    cfg.TitleSlug,
	}, nil
}

// openSharedSocialDB ouvre la DB SharedSocial (optionnelle). Retourne nil en
// cas d'échec (non bloquant : absence normale au boot, fichier créé en sync).
//
//nolint:unused // WIP refacto Phase 3 stabilisation pool — péremption 2026-06-22 (à câbler dans openPlayerDB ou supprimer).
func openSharedSocialDB(ctx context.Context, cfg PlayerPoolConfig) *DB {
	if cfg.SharedSocialDBPath == "" {
		return nil
	}
	socialDB, err := OpenReadWrite(cfg.SharedSocialDBPath, cfg.UserTimezone)
	if err != nil {
		// Non bloquant : la DB sera créée lors de la prochaine migration.
		slog.WarnContext(ctx, "pool: ouverture SharedSocial échouée (dégradation: socialDB=nil)",
			"path", cfg.SharedSocialDBPath, "gamertag", cfg.Gamertag, "err", err)
		return nil
	}
	if mErr := migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial); mErr != nil {
		slog.ErrorContext(ctx, "pool: migrations SharedSocial échouées",
			"path", cfg.SharedSocialDBPath, "gamertag", cfg.Gamertag, "err", mErr)
	}
	return socialDB
}

// attachGlobalXuidAliasesIfConfigured attache la DB globale xbox_aliases sur
// les conn player + social. Non-bloquant : silencieux si la DB n'existe pas
// (tolérance prévue côté query — retombe sur NULL).
//
// P5.3 : mapping xuid→gamertag global Microsoft pour les JOIN global.xuid_aliases.
//
//nolint:unused // WIP refacto Phase 3 stabilisation pool — péremption 2026-06-22 (à câbler dans openPlayerDB ou supprimer).
func attachGlobalXuidAliasesIfConfigured(
	ctx context.Context, playerDB, socialDB *DB, cfg PlayerPoolConfig,
) {
	if cfg.GlobalXuidAliasesDBPath == "" {
		return
	}
	if err := attachGlobalXuidAliases(ctx, playerDB, cfg.GlobalXuidAliasesDBPath); err != nil {
		slog.WarnContext(ctx, "pool: ATTACH global xbox_aliases échoué (player conn)",
			"path", cfg.GlobalXuidAliasesDBPath, "gamertag", cfg.Gamertag, "err", err)
	}
	if socialDB != nil {
		if err := attachGlobalXuidAliases(ctx, socialDB, cfg.GlobalXuidAliasesDBPath); err != nil {
			slog.WarnContext(ctx, "pool: ATTACH global xbox_aliases échoué (social conn)",
				"path", cfg.GlobalXuidAliasesDBPath, "gamertag", cfg.Gamertag, "err", err)
		}
	}
}

func ensurePlayerDBMigrations(path string) error {
	_ = migration.All()

	rwDB, err := OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw: %w", err)
	}
	defer rwDB.Close()

	if err := migration.RunForDB(rwDB.SQLDb(), migration.TargetPlayer); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// attachGlobalState : sync.Once-gated state for global xbox_aliases attach.
// Phase 3 plan stabilisation 2026-05-22 : process-level idempotence.
//
// Contexte : avant ce fix, attachGlobalXuidAliases était appelé à chaque
// ouverture player + social conn (4 joueurs × 2 conns = 8 appels). Toutes
// les tentatives sauf la 1ère échouaient avec "Unique file handle conflict:
// Cannot attach 'global' - the database file ... is already attached by
// database 'global'" — cf. AUDIT_DUCKDB_ATTACH_2026-05-21.md §1.
//
// Cause racine : DuckDB instance partagée au niveau process. ATTACH sur un
// fichier donné est process-wide, pas per-conn ni per-sql.DB.
//
// Fix : sync.Once par globalPath. Le 1er appel attache, les suivants
// no-opent (le catalog DuckDB conserve l'alias process-wide ; toutes les
// conns futures peuvent l'utiliser).
type attachGlobalState struct {
	once sync.Once
	err  error // erreur du 1er appel (préservée pour les suivants)
}

var (
	attachGlobalMu     sync.Mutex
	attachGlobalByPath = map[string]*attachGlobalState{}
)

// resetGlobalAttachState efface l'état sync.Once par path. Utilisé par les
// tests qui ouvrent/ferment des DBs globales successivement (sinon le 2e
// test du run récupère l'état "already attached" du 1er).
//
//nolint:unused // WIP test helper Phase 3 — péremption 2026-06-22 (à câbler dans tests sync.Once ou supprimer).
func resetGlobalAttachState() {
	attachGlobalMu.Lock()
	defer attachGlobalMu.Unlock()
	attachGlobalByPath = map[string]*attachGlobalState{}
}

// attachGlobalXuidAliases attache la DB globale `xbox_aliases.duckdb` sous
// l'alias `global` (P5.3). Tolère l'absence du fichier (avant migration) :
// la table sera créée via init schema si nécessaire.
//
// Idempotent process-level via sync.Once : seul le 1er appel par path
// déclenche l'ATTACH ; les suivants reprennent le résultat du 1er.
func attachGlobalXuidAliases(ctx context.Context, db *DB, globalPath string) error {
	// Récupérer (ou créer) l'état sync.Once pour ce path.
	attachGlobalMu.Lock()
	state, ok := attachGlobalByPath[globalPath]
	if !ok {
		state = &attachGlobalState{}
		attachGlobalByPath[globalPath] = state
	}
	attachGlobalMu.Unlock()

	state.once.Do(func() {
		if _, err := os.Stat(globalPath); err != nil {
			// Fichier absent — créer une DB globale vide avec le schéma minimal.
			if err := initGlobalXuidAliasesSchema(globalPath); err != nil {
				state.err = fmt.Errorf("init global db: %w", err)
				return
			}
		}
		if _, err := db.Exec(ctx,
			fmt.Sprintf("ATTACH '%s' AS global", globalPath),
		); err != nil {
			// Vérifier accessibilité — peut-être déjà attaché par un autre process
			// (concurrent CLI tool) ou par une lecture antérieure non passée par
			// nous. Si oui, OK ; sinon erreur réelle.
			var count int
			if pingErr := db.QueryRow(ctx,
				"SELECT COUNT(*) FROM global.xuid_aliases").Scan(&count); pingErr != nil {
				state.err = fmt.Errorf("attach global: %w (attach err: %v)", pingErr, err)
				return
			}
		}
	})
	return state.err
}

// initGlobalXuidAliasesSchema crée la DB globale et la table xuid_aliases si
// absentes. Appelé par attachGlobalXuidAliases en pré-condition (avant la
// première run du script de migration).
func initGlobalXuidAliasesSchema(globalPath string) error {
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		return err
	}
	db, err := OpenReadWrite(globalPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.SQLDb().ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR NOT NULL,
			last_seen TIMESTAMP NOT NULL DEFAULT now()
		)
	`)
	return err
}

// XUID du joueur depuis sync_meta (fallback si cfg.XUID vide).
func ResolveXUID(ctx context.Context, db *DB) (string, error) {
	var xuid string
	err := db.QueryRow(ctx, "SELECT value FROM sync_meta WHERE key = 'xuid'").Scan(&xuid)
	if err != nil {
		return "", fmt.Errorf("resolve xuid: %w", err)
	}
	return xuid, nil
}
