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
	"sync"
	"time"

	"levelup/go-api/internal/migration"

	"golang.org/x/sync/singleflight"
)

// PlayerPoolConfig contient les chemins nécessaires pour ouvrir un PlayerDB.
type PlayerPoolConfig struct {
	Gamertag           string
	XUID               string
	TitleSlug          string // Sprint 44 : namespace titre (ex: "halo_infinite")
	PlayerDBPath       string
	SharedDBPath       string
	MetaDBPath         string
	SharedSocialDBPath string // shared_social.duckdb (médias, likes, favoris)
	UserTimezone       string // timezone IANA pour la lecture des TIMESTAMP (ex: "Europe/Paris")

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

	// SocialPersister est le chemin NOMINAL d'écriture sur SharedSocial via le
	// pattern Collect→Persist (ADR 0022). Le Persist() générique ne CHECKPOINT pas
	// systématiquement : le flush WAL est borné par le scheduler 5 min + le
	// CHECKPOINT shutdown (cf. #7659). NB (revue 2026-06-01 SS-02) : ce n'est PAS
	// l'unique chemin — les mutations notifications et Prestige écrivent encore en
	// direct, et il n'y a pas de sentinel AST sur les écritures (seulement l'ATTACH
	// est gardé). Nil si SharedSocial est nil.
	//
	// L'interface est définie ici (pas dans internal/persist) pour éviter
	// un cycle d'import : internal/persist importe déjà internal/platform/duckdb
	// via combined_persister.go. L'implémentation concrète
	// persist.SharedSocialPersister satisfait cette interface structurellement
	// et est injectée par main.go au boot après openPlayerDB.
	SocialPersister SocialPersister

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

	playerDB, err := openPlayerRWWithLockRetry(cfg.PlayerDBPath, cfg.UserTimezone)
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
	//
	// Récupération auto : openSharedSocialWithWALRecovery quarantines un WAL
	// non-rejouable (bug DuckDB #7659) puis retry. Cf. pool_shared_social_recovery.go
	// + ADR 0021. En cas d'échec persistant : socialDB=nil (dégradation graceful
	// existante), avec slog.Error pointant vers cmd/rebuild_shared_social.
	socialDB := openSharedSocialWithWALRecovery(ctx, cfg.SharedSocialDBPath, cfg.UserTimezone, cfg.Gamertag)
	if socialDB != nil {
		// Appliquer les migrations shared_social (idempotentes).
		if mErr := migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial); mErr != nil {
			slog.ErrorContext(ctx, "pool: migrations SharedSocial échouées",
				"path", cfg.SharedSocialDBPath, "gamertag", cfg.Gamertag, "err", mErr)
		}
	}

	// attachShared sur la conn player a été retiré (ADR 0016, commit 9c.5).
	// Toutes les queries shared passent désormais par SharedReader (Provider en
	// prod, LegacySharedReader pour les tests sans Provider injecté).
	// Sprint P0→P7 (2026-05-20) a finalisé la migration de tous les repos
	// applicatifs (media, post-sync, engagement, profile, career, explorer,
	// sessions, stats, leaderboard, exclusions, match_view).

	// La DB globale xbox_aliases a été CONSOLIDÉE dans shared.xuid_aliases
	// (refactor 2026-06-19, sous-commande `levelup consolidate-aliases`) : plus
	// aucun ATTACH `global`. La résolution xuid→gamertag passe entièrement par
	// shared (v_gamertag_lookup pour l'affichage, LookupXUIDByGamertag pour les
	// coéquipiers, invariant I13). Le champ de config GlobalXuidAliasesDBPath a
	// été retiré et le moteur de sync n'écrit plus dans ce store (2026-06-19).

	// attachShared sur SharedSocial retiré aussi.
	// media_repo passe désormais entièrement par SharedReader pour les queries
	// shared.* — plus aucune conn du pool ne porte d'ATTACH shared.
	//
	// 2026-05-25 : suppression de l'ATTACH global xbox_aliases sur SharedSocial.
	// Root cause des "INTERNAL Error: Failure while replaying WAL file:
	// DatabaseManager::GetDefaultDatabase" observés à chaque boot post-rebuild
	// Air : DuckDB bug #7659 — un ATTACH écrit dans le WAL de la base RW une
	// entrée non-rejouable au reboot. Conséquence : SharedSocial=nil partout,
	// rail média home vide, prestige_bundle_init_failed. Cycle vicieux relancé
	// à chaque restart serveur.
	//
	// Compromis fonctionnel : media_repo n'a plus accès direct à `global.
	// xuid_aliases` via JOIN SQL sur la conn social. Les rares features qui
	// résolvent xuid→gamertag pour les likers de médias devront passer par un
	// lookup Go (cache process-wide ou query séparée sur xbox_aliases RO).
	// Acceptable : feature peu utilisée, dégradation = afficher xuid brut
	// au lieu du gamertag pour les likers externes.
	_ = socialDB // ATTACH global supprimé — cf. commentaire ci-dessus

	// SharedReader (commit 8a) : par défaut, wrap pdb.Shared en mode legacy.
	// Si cfg.SharedReader fourni (mode B-swap), on l'utilise tel quel — le
	// caller a déjà arbitré le mode.
	sharedReader := cfg.SharedReader
	bSwapEnabled := sharedReader != nil
	if sharedReader == nil {
		sharedReader = LegacySharedReader(sharedDB)
	}

	// Phase 4 du refactor shared_social Collect→Persist (ADR 0022) :
	// instancier le SocialPersister via la factory injectée par main.go.
	// Si la factory n'est pas configurée (cas tests, bootstrap CLI), reste
	// nil et les repos retombent sur leur chemin legacy db.Exec.
	var socialPersister SocialPersister
	if socialDB != nil && SocialPersisterFactory != nil {
		socialPersister = SocialPersisterFactory(socialDB.SQLDb())
	}

	return &PlayerDB{
		Player:          playerDB,
		Shared:          sharedDB,
		SharedSocial:    socialDB,
		SocialPersister: socialPersister,
		Metadata:        metaDB,
		sharedDBPath:    cfg.SharedDBPath,
		userTimezone:    cfg.UserTimezone,
		bSwapEnabled:    bSwapEnabled,
		SharedReader:    sharedReader,
		XUID:            cfg.XUID,
		Gamertag:        cfg.Gamertag,
		TitleSlug:       cfg.TitleSlug,
	}, nil
}

// playerOpenAttempts / playerOpenDelay : retry borné de l'ouverture RW d'une player
// DB sur contention fichier. Même cause que le boot metadata (cmd/server/main.go
// metaOpenAttempts) : sur Windows, air force-kill le serveur au hot-reload (TASKKILL
// /F, send_interrupt non supporté → pas de shutdown gracieux) puis relance ; l'OS met
// ~1-2 s à libérer les HANDLEs DuckDB de l'ancien process. Le pool ouvre les player DB
// en lazy (spartan_cron, world-enrich…) APRÈS le boot metadata, donc tombe dans la même
// fenêtre sans bénéficier du retry de main.go. 12×500 ms = 6 s (aligné metaOpenAttempts).
const (
	playerOpenAttempts = 12
	playerOpenDelay    = 500 * time.Millisecond
)

// openPlayerRWWithLockRetry ouvre une player DB en RW en retentant brièvement sur
// IsFileLockError (handle non encore libéré par l'ancien process après un TASKKILL air).
// Si le lock persiste au-delà de la fenêtre (vraie 2e instance / CLI backfill), rend
// l'erreur d'origine — toujours classée IsFileLockError → agrégée par spartan_cron.
func openPlayerRWWithLockRetry(path string, timezone ...string) (*DB, error) {
	var db *DB
	var err error
	for attempt := range playerOpenAttempts {
		db, err = OpenReadWrite(path, timezone...)
		if err == nil || !IsFileLockError(err) || attempt == playerOpenAttempts-1 {
			return db, err
		}
		slog.Debug("pool: player DB verrouillée (handle non libéré par l'ancien process ?) — nouvelle tentative",
			"path", path, "attempt", attempt+1, "max", playerOpenAttempts)
		time.Sleep(playerOpenDelay)
	}
	return db, err
}

func ensurePlayerDBMigrations(path string) error {
	_ = migration.All()

	rwDB, err := openPlayerRWWithLockRetry(path)
	if err != nil {
		if IsFileLockError(err) {
			// Cause opérationnelle, pas une corruption : un AUTRE process tient
			// déjà ce fichier en écriture (CLI backfill, 2e instance serveur,
			// hot-reload Air pas encore libéré). DuckDB est mono-writer par
			// fichier. Message actionnable plutôt qu'un "open rw" opaque.
			return fmt.Errorf("open rw: player DB %q verrouillée par un autre process "+
				"(CLI backfill / 2e instance serveur encore ouverte ?) — fermer le writer concurrent puis relancer: %w", path, err)
		}
		return fmt.Errorf("open rw: %w", err)
	}
	defer rwDB.Close()

	if err := migration.RunForDB(rwDB.SQLDb(), migration.TargetPlayer); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
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
