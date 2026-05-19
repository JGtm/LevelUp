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

		_ = pdb.Shared.Close()
		if pdb.SharedSocial != nil {
			_ = pdb.SharedSocial.Close()
		}
		_ = pdb.Metadata.Close()
		globalPool.Delete(key)
		return true
	})
}

// openPlayerDB ouvre et initialise un PlayerDB complet.
func openPlayerDB(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	if err := ensurePlayerDBMigrations(cfg.PlayerDBPath); err != nil {
		return nil, fmt.Errorf("pool: migrate player db %s: %w", cfg.Gamertag, err)
	}

	playerDB, err := OpenReadWrite(cfg.PlayerDBPath, cfg.UserTimezone)
	if err != nil {
		return nil, fmt.Errorf("pool: open player db %s: %w", cfg.Gamertag, err)
	}

	// OpenReadOnly pour partager l'instance RO avec main.go::sharedDB et permettre
	// l'ATTACH RO de shared sur les connexions player DB. Le mode RW exclusif
	// (OpenReadWriteShared) verrouille le fichier au niveau process et bloque
	// tout ATTACH ultérieur avec "Unique file handle conflict", cassant les
	// pages HTTP qui font des JOINs `shared.match_registry`.
	//
	// Trade-off : le sync engine ouvre shared en RW via OpenSharedDB lors de
	// RunDelta, ce qui peut entrer en conflit "different configuration" avec
	// l'instance RO ouverte ici. C'est documenté dans main.go.
	sharedDB, err := OpenReadOnly(cfg.SharedDBPath, cfg.UserTimezone)
	if err != nil {
		_ = playerDB.Close()
		return nil, fmt.Errorf("pool: open shared db: %w", err)
	}
	// Les migrations shared sont gérées par runMigrations() dans main.go.

	// metadata.duckdb est ouverte en RW partagé : le DuckDBIndexStore (assets) a besoin d'y
	// écrire (table asset_index). Les deux utilisent la clé cache "rw:path", donc une seule
	// sql.DB est créée. OpenReadWriteShared garantit maxOpenConns=4 pour les lectures concurrentes.
	metaDB, err := OpenReadWriteShared(cfg.MetaDBPath, cfg.UserTimezone)
	if err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		return nil, fmt.Errorf("pool: open metadata db: %w", err)
	}

	// SharedSocial est optionnel : absent si le fichier n'existe pas encore.
	// Ouvert en read-write pour permettre les écritures (favoris, likes).
	var socialDB *DB
	if cfg.SharedSocialDBPath != "" {
		socialDB, err = OpenReadWrite(cfg.SharedSocialDBPath, cfg.UserTimezone)
		if err != nil {
			// Non bloquant : la DB sera créée lors de la prochaine migration.
			socialDB = nil
		} else {
			// Appliquer les migrations shared_social (idempotentes).
			_ = migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial)
		}
	}

	// ATTACH shared sur la connexion player RW pour les requêtes join (sync engine et handlers HTTP)
	if err := attachShared(ctx, playerDB, cfg.SharedDBPath); err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		_ = metaDB.Close()
		return nil, fmt.Errorf("pool: attach shared on player db: %w", err)
	}

	// P5.3 : ATTACH global xbox_aliases (mapping xuid→gamertag global Microsoft).
	// Non-bloquant : si la DB globale n'existe pas encore (avant migration), les
	// requêtes qui font `JOIN global.xuid_aliases` retomberont sur NULL.
	if cfg.GlobalXuidAliasesDBPath != "" {
		if err := attachGlobalXuidAliases(ctx, playerDB, cfg.GlobalXuidAliasesDBPath); err != nil {
			// Non bloquant — log déjà émis dans attachGlobalXuidAliases.
			_ = err
		}
	}

	// ATTACH shared_matches_v2 sur SharedSocial pour les JOIN match_registry dans Q37.
	if socialDB != nil && cfg.SharedDBPath != "" {
		if err := attachShared(ctx, socialDB, cfg.SharedDBPath); err != nil {
			// Non bloquant : les colonnes map/mode seront NULL.
			_ = err
		}
	}
	// P5.3 : ATTACH global xbox_aliases sur SharedSocial aussi (media_repo fait
	// `JOIN global.xuid_aliases` sur les likers de médias).
	if socialDB != nil && cfg.GlobalXuidAliasesDBPath != "" {
		_ = attachGlobalXuidAliases(ctx, socialDB, cfg.GlobalXuidAliasesDBPath)
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

// attachShared attache shared_matches_v2.duckdb sur une connexion player sous
// l'alias `shared`. Le code SQL applicatif référence partout `shared.X` pour
// les tables match_registry, match_participants, etc.
//
// Cas spécial DuckDB-Go : quand main.go ouvre déjà `sharedDB` séparément via
// OpenReadWriteShared, le fichier `shared_matches_v2.duckdb` est auto-attaché
// dans chaque nouvelle connexion DuckDB du process sous l'alias auto-nommé
// `shared_matches_v2` (filename sans .duckdb). Tenter `ATTACH ... AS shared`
// échoue alors avec "Unique file handle conflict".
//
// Workaround : si on détecte cette situation, créer un schéma `shared` peuplé
// de VIEWs qui redirigent vers les tables de `shared_matches_v2`. Le code SQL
// applicatif fonctionne alors transparentement.
func attachShared(ctx context.Context, db *DB, sharedPath string) error {
	_, err := db.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath),
	)
	if err == nil {
		return nil
	}

	// L'ATTACH a échoué. Vérifier si shared est déjà accessible (idempotence
	// classique : autre goroutine a déjà attaché).
	var count int
	if pingErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM shared.match_registry").Scan(&count); pingErr == nil {
		return nil
	}

	// Pas accessible via `shared.*`. Si DuckDB-Go a auto-attaché shared_matches_v2
	// (cas du conflict "Unique file handle"), créer des VIEWs alias.
	if pingErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM shared_matches_v2.match_registry").Scan(&count); pingErr == nil {
		if aliasErr := createSharedAliasViews(ctx, db); aliasErr != nil {
			return fmt.Errorf("attach shared: alias views creation failed: %w (attach err: %v)", aliasErr, err)
		}
		return nil
	}

	// Ni `shared` ni `shared_matches_v2` accessibles → vraie erreur.
	return fmt.Errorf("attach shared: %w", err)
}

// sharedAliasTables liste les tables de `shared_matches_v2.duckdb` qui doivent
// être exposées sous le schéma `shared` quand l'ATTACH direct n'est pas possible
// (cf. attachShared).
//
// Cette liste DOIT rester en sync avec sharedSchemaSQL (sync/schema.go) :
// toute table ajoutée dans le schéma shared doit être ajoutée ici sinon le
// code applicatif qui interroge `shared.X` plantera quand le workaround est
// actif.
var sharedAliasTables = []string{
	"match_registry",
	"match_participants",
	"highlight_events",
	"medals_earned",
	"killer_victim_pairs",
	"xuid_aliases",
	"weapon_kills",
}

// createSharedAliasViews crée le schéma `shared` (s'il n'existe pas) puis
// expose chaque table de sharedAliasTables comme une VIEW pointant vers
// `shared_matches_v2.<table>`. Idempotent (CREATE OR REPLACE).
func createSharedAliasViews(ctx context.Context, db *DB) error {
	if _, err := db.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS shared"); err != nil {
		return fmt.Errorf("create schema shared: %w", err)
	}
	for _, table := range sharedAliasTables {
		stmt := fmt.Sprintf(
			"CREATE OR REPLACE VIEW shared.%s AS SELECT * FROM shared_matches_v2.%s",
			table, table,
		)
		if _, err := db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("create view shared.%s: %w", table, err)
		}
	}
	return nil
}

// attachGlobalXuidAliases attache la DB globale `xbox_aliases.duckdb` sous
// l'alias `global` (P5.3). Tolère l'absence du fichier (avant migration) :
// la table sera créée via init schema si nécessaire.
func attachGlobalXuidAliases(ctx context.Context, db *DB, globalPath string) error {
	if _, err := os.Stat(globalPath); err != nil {
		// Fichier absent — créer une DB globale vide avec le schéma minimal.
		// Idempotent : si une autre instance vient de la créer, ATTACH échouera
		// proprement et on retombera sur le ping.
		if err := initGlobalXuidAliasesSchema(globalPath); err != nil {
			return fmt.Errorf("init global db: %w", err)
		}
	}
	_, err := db.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS global", globalPath),
	)
	if err != nil {
		// Vérifier accessibilité — déjà attachée, OK.
		var count int
		pingErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM global.xuid_aliases").Scan(&count)
		if pingErr != nil {
			return fmt.Errorf("attach global: %w (attach err: %v)", pingErr, err)
		}
	}
	return nil
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
	_, err = db.SQLDb().Exec(`
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
