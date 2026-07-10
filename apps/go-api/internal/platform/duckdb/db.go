// Package duckdb fournit l'accès aux bases DuckDB de LevelUp.
// CGO requis : compilé avec github.com/duckdb/duckdb-go/v2.
//
// Stratégie de connexion (Sprint 0) :
//   - Une connexion par base, avec sql.DB (pool natif Go).
//   - Les bases read-only utilisent "?access_mode=read_only".
//   - ATTACH est exécuté via une connexion dédiée (sql.Conn pinée).
//   - Les types critiques : UBIGINT→uint64, TIMESTAMP WITH TIME ZONE→time.Time,
//     VARCHAR→string, BOOLEAN→bool.
package duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// Limites ressources DuckDB appliquées à CHAQUE connexion (J2, 2026-07-05).
// Sans limite, DuckDB fixe memory_limit à ~80% de la RAM (~1.5 Go sur un VPS 2 Go) :
// un seul gros SELECT/backfill peut alors OOM le conteneur (prod = no-swap). En
// bornant memory_limit, DuckDB DÉBORDE sur disque au-delà (dégradation sûre, pas
// d'échec). Défaut CONSERVATEUR calibré sur le VPS prod (2 vCPU / 2 Go ; conteneur
// ~845 Mo au repos, ~256 Mo dispo). Override par env pour un hôte plus large :
//
//	LEVELUP_DUCKDB_MEMORY_LIMIT (ex. "2GB")   LEVELUP_DUCKDB_THREADS (ex. "4")
//
// Knob permanent (pas de date de retrait) ; critère : memory_limit doit rester
// < (RAM - baseline conteneur - marge démo/OS).
var (
	duckMemoryLimit = envOr("LEVELUP_DUCKDB_MEMORY_LIMIT", "512MB")
	duckThreads     = envIntOr("LEVELUP_DUCKDB_THREADS", 2)
)

// BudgetsSnapshot expose les bornes ressources DuckDB effectives (J2) pour
// l'observabilité — publiées sous /debug/vars levelup/duckdb_budgets, à côté de
// duckdb_pool_stats (J1). Valeurs statiques résolues au boot (env ou défauts) :
// permet de vérifier en un coup d'œil la config mémoire/threads/pool RÉELLEMENT
// appliquée sur un hôte donné (prérequis de la calibration measure-first).
func BudgetsSnapshot() map[string]any {
	return map[string]any{
		"memory_limit":         duckMemoryLimit,
		"threads":              duckThreads,
		"pool_max_open_shared": poolMaxOpenShared,
		"pool_max_idle_shared": poolMaxIdleShared,
		"pool_single_conn":     poolSingleConn,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// DB encapsule une connexion DuckDB ouverte.
//
// Les champs après cacheKey sont persistés pour permettre Reopen() de
// reconstruire la connexion avec exactement la même configuration suite à
// une invalidation fatale (cf. bug DuckDB ART/NULL, 2026-05-14).
type DB struct {
	// sqlDB et closed sont accédés en lecture SANS verrou (Query/Exec/SQLDb…) et
	// remplacés in-place sous openDBsMu (Reopen/openCachedDB sur ping-fail). On les
	// rend atomiques pour éliminer la data race lecteur/swap (revue P0 2026-06-02) :
	// un lecteur voit toujours soit l'ancien soit le nouveau handle publié.
	sqlDB    atomic.Pointer[sql.DB]
	path     string
	cacheKey string
	closed   atomic.Bool
	// Paramètres d'ouverture conservés pour Reopen().
	dsn          string
	maxOpenConns int
	maxIdleConns int
	timezone     string
	op           string
}

// loadSQL retourne le *sql.DB courant de façon lock-free (atomic.Load).
func (db *DB) loadSQL() *sql.DB { return db.sqlDB.Load() }

type cachedDB struct {
	db       *DB
	refCount int
}

var (
	openDBsMu sync.Mutex
	openDBs   = map[string]*cachedDB{}
)

// DumpCachedLeaks retourne un snapshot des entrées encore présentes dans le
// cache openDBs (clé → refCount). Pour diagnostic post-shutdown : si CloseAll
// a été appelé et que le map n'est pas vide, ça veut dire qu'un caller a
// oublié son Close() apparié → fuite de handle DuckDB → verrou au prochain
// hot-reload Air sur Windows.
//
// Phase 2 plan stabilisation 2026-05-22. Utilisé dans cmd/server/main.go
// après le bloc shutdown gracieux pour logger les leaks détectés.
func DumpCachedLeaks() map[string]int {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	if len(openDBs) == 0 {
		return nil
	}
	out := make(map[string]int, len(openDBs))
	for k, c := range openDBs {
		out[k] = c.refCount
	}
	return out
}

// PoolStatsSnapshot retourne les sql.DBStats de chaque handle DuckDB ouvert
// (clé de cache "ro:"/"rw:"+path → stats). Observabilité pool (J1, ADR 0009) :
// WaitCount/WaitDuration non nuls révèlent la contention sur le pool (lectures qui
// attendent une connexion libre) ; InUse proche de MaxOpenConnections = saturation.
// Snapshot instantané, sous verrou — appelé par l'endpoint expvar /debug/vars.
func PoolStatsSnapshot() map[string]sql.DBStats {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	out := make(map[string]sql.DBStats, len(openDBs))
	for k, c := range openDBs {
		if c.db == nil || c.db.closed.Load() {
			continue
		}
		if sdb := c.db.loadSQL(); sdb != nil {
			out[k] = sdb.Stats()
		}
	}
	return out
}

// LookupCachedDB retourne le *DB déjà ouvert pour path s'il est présent dans
// le cache process-wide, sinon (nil, false). Cherche d'abord la variante RW
// (clé "rw:"+path), puis RO ("ro:"+path).
//
// Usage : permet à un consommateur secondaire (ex. backup scheduler) de
// réutiliser une connexion existante au lieu d'ouvrir un 2e handle avec une
// config différente — DuckDB refuse `?access_mode=read_only` sur un fichier
// déjà ouvert en RW dans le même process ("Can't open a connection to same
// database file with a different configuration").
//
// Aucun incrément de refCount : c'est un emprunt non-possédant. Le caller ne
// doit pas appeler Close() sur le *DB retourné.
func LookupCachedDB(path string) (*DB, bool) {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	if cached, ok := openDBs["rw:"+path]; ok && cached.db != nil && !cached.db.closed.Load() {
		return cached.db, true
	}
	if cached, ok := openDBs["ro:"+path]; ok && cached.db != nil && !cached.db.closed.Load() {
		return cached.db, true
	}
	return nil, false
}

// OpenReadForQuery retourne un handle utilisable en LECTURE, en réutilisant le
// handle déjà en cache (RW ou RO) s'il existe — une lecture (SELECT) marche sur
// un handle RW. Évite l'échec "Can't open ... with a different configuration"
// quand un autre subsystem (pool, career live, backup cron) tient déjà le fichier
// en RW dans le même process : forcer un OpenReadOnly dans ce cas échoue (DuckDB
// refuse RO+RW concurrents sur un même fichier). Incident 2026-06-01 : la phase
// discovery du sync V2 (load known) échouait ainsi par intermittence sur les DB
// joueur (même classe que RC-A / ADR-0016).
//
// Le release retourné ne ferme le handle QUE s'il a été ouvert ici. Un handle
// emprunté au cache n'est pas fermé (sa durée de vie appartient à son
// propriétaire), conformément au contrat de LookupCachedDB.
func OpenReadForQuery(path string, timezone ...string) (*sql.DB, func(), error) {
	if cached, ok := LookupCachedDB(path); ok {
		return cached.SQLDb(), func() {}, nil
	}
	db, err := OpenReadOnly(path, timezone...)
	if err != nil {
		return nil, nil, err
	}
	return db.SQLDb(), func() { _ = db.Close() }, nil
}

// EvictAndCloseCached ferme et retire du cache process-wide les handles (RO et
// RW) ouverts pour ce chemin de fichier, INDÉPENDAMMENT du refCount. À appeler
// AVANT un os.RemoveAll de la base sur Windows : un handle ouvert verrouille le
// fichier et ferait échouer la suppression.
//
// Action DESTRUCTIVE volontaire : tout détenteur d'un handle évincé verra ses
// requêtes suivantes échouer — réservé à la PURGE délibérée d'une player DB
// (l'utilisateur supprime les données de ce titre). Retourne le nombre de
// handles effectivement fermés.
func EvictAndCloseCached(path string) int {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	closed := 0
	for _, key := range []string{"rw:" + path, "ro:" + path} {
		cached, ok := openDBs[key]
		if !ok || cached.db == nil {
			continue
		}
		delete(openDBs, key)
		if cached.db.closed.Swap(true) {
			continue // déjà fermé par ailleurs
		}
		if sqlDB := cached.db.loadSQL(); sqlDB != nil {
			_ = sqlDB.Close()
		}
		closed++
	}
	return closed
}

// sanitizeTimezone valide un nom de timezone IANA pour éviter les injections SQL.
// Retourne "" si la valeur contient des caractères non autorisés.
func sanitizeTimezone(tz string) string {
	for _, c := range tz {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '/' || c == '_' || c == '-' || c == '+':
		default:
			return ""
		}
	}
	return tz
}

// OpenReadOnly ouvre une base DuckDB en lecture seule.
// Le DSN est : "file.duckdb?access_mode=read_only".
// Une seule instance par chemin est maintenue (cache process-level).
// timezone (optionnel) : nom IANA (ex: "Europe/Paris") — appliqué via SET TimeZone sur chaque connexion.
func OpenReadOnly(path string, timezone ...string) (*DB, error) {
	tz := ""
	if len(timezone) > 0 {
		raw := timezone[0]
		tz = sanitizeTimezone(raw)
		if tz == "" && raw != "" {
			slog.Warn("duckdb: timezone invalide ignorée", "input", raw, "path", path)
		}
	}
	return openCachedDB(
		"ro:"+path,
		path,
		path+"?access_mode=read_only",
		poolMaxOpenShared,
		poolMaxIdleShared,
		"OpenReadOnly",
		tz,
	)
}

// OpenReadWriteShared ouvre une base DuckDB en lecture-écriture avec un pool de connexions
// (poolMaxOpenShared, comme OpenReadOnly). À utiliser pour les bases partagées (ex: metadata.duckdb)
// qui reçoivent des lectures concurrentes ET des écritures occasionnelles.
// Partage la même clé de cache que OpenReadWrite : si l'un est déjà ouvert, l'autre le réutilise.
func OpenReadWriteShared(path string, timezone ...string) (*DB, error) {
	tz := ""
	if len(timezone) > 0 {
		raw := timezone[0]
		tz = sanitizeTimezone(raw)
		if tz == "" && raw != "" {
			slog.Warn("duckdb: timezone invalide ignorée", "input", raw, "path", path)
		}
	}
	return openCachedDB("rw:"+path, path, path, poolMaxOpenShared, poolMaxIdleShared, "OpenReadWriteShared", tz)
}

// OpenReadWrite ouvre une base DuckDB en lecture-écriture.
// Utilisé pour les migrations au démarrage. UNE seule connexion : pas de pool.
// timezone (optionnel) : nom IANA (ex: "Europe/Paris") — appliqué via SET TimeZone sur chaque connexion.
func OpenReadWrite(path string, timezone ...string) (*DB, error) {
	tz := ""
	if len(timezone) > 0 {
		raw := timezone[0]
		tz = sanitizeTimezone(raw)
		if tz == "" && raw != "" {
			slog.Warn("duckdb: timezone invalide ignorée", "input", raw, "path", path)
		}
	}
	return openCachedDB("rw:"+path, path, path, poolSingleConn, poolSingleConn, "OpenReadWrite", tz)
}

// Limites de pool DuckDB (J8, 2026-07-05 : ex-magic 4/2/1). Le driver DuckDB est
// mono-process (ADR 0013) : les pools RO / shared-RW tolèrent quelques lectures
// concurrentes (poolMaxOpenShared) avec réutilisation au repos (poolMaxIdleShared) ;
// le writer RW exclusif reste à une seule connexion (poolSingleConn).
const (
	poolMaxOpenShared = 4
	poolMaxIdleShared = 2
	poolSingleConn    = 1
)

func openCachedDB(
	key, path, dsn string,
	maxOpenConns, maxIdleConns int,
	op string,
	timezone string,
) (*DB, error) {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	if cached, ok := openDBs[key]; ok {
		if err := cached.db.loadSQL().PingContext(context.Background()); err == nil {
			cached.refCount++
			return cached.db, nil
		}
		// ROOT CAUSE FIX (2026-05-25) : ping a échoué → l'ancien sqlDB n'est
		// plus utilisable. AVANT, on Close() + delete(cache) + ouvrait un
		// NOUVEAU wrapper *DB. Problème : globalPool.Player et les repos HTTP
		// gardent une référence vers l'ANCIEN wrapper, dont sqlDB est mort
		// → cascade "sql: database is closed" sur toutes les pages HTTP
		// jusqu'au prochain restart du serveur.
		//
		// FIX : swap le sqlDB IN-PLACE dans l'ancien wrapper (même pattern que
		// Reopen()). Les références externes voient automatiquement la nouvelle
		// handle sans changement.
		oldDB := cached.db
		slog.WarnContext(context.Background(),
			"duckdb: cache ping fail — swap in-place du sqlDB pour préserver les refs externes",
			"path", oldDB.path, "op", oldDB.op, "key", key)
		newSQLDB, err := openSQLDBFor(oldDB.dsn, oldDB.timezone, oldDB.op, oldDB.path)
		if err != nil {
			// Reopen impossible (fichier inaccessible, lock, etc.) — fallback :
			// délai standard, Close + delete + signal au caller.
			_ = oldDB.loadSQL().Close()
			delete(openDBs, key)
			slog.ErrorContext(context.Background(),
				"duckdb: cache ping fail + reopen échoué — handle perdue, caller doit retry",
				"path", oldDB.path, "op", oldDB.op, "err", err)
			return nil, err
		}
		applyConnLimits(newSQLDB, oldDB.maxOpenConns, oldDB.maxIdleConns)
		// Fermer l'ancien sqlDB en best-effort puis swap atomique.
		_ = oldDB.loadSQL().Close()
		oldDB.sqlDB.Store(newSQLDB)
		oldDB.closed.Store(false)
		cached.refCount++
		return oldDB, nil
	}

	sqlDB, err := openSQLDBFor(dsn, timezone, op, path)
	if err != nil {
		// Phase 2 plan stabilisation 2026-05-22 : démoté de Error à Debug. Cette
		// branche est principalement déclenchée par les retries au boot
		// (cmd/server/main.go OpenReadWriteShared metaPath × 12, AssetMetadata
		// × 3, etc.) où Air n'a pas encore libéré le HANDLE Windows post-SIGKILL.
		// Le caller a l'info pour logger au bon niveau (Warn sur tentative
		// intermédiaire, Error sur abandon final). Logger ici en Error spammait
		// 11 lignes ERROR pour 1 boot réussi.
		slog.Debug("duckdb: ouverture DB échouée",
			"path", path, "op", op, "dsn", dsn, "err", err)
		return nil, err
	}
	if timezone != "" {
		slog.Debug("duckdb: timezone appliquée", "timezone", timezone, "path", path)
	}
	applyConnLimits(sqlDB, maxOpenConns, maxIdleConns)

	db := &DB{
		path:         path,
		cacheKey:     key,
		dsn:          dsn,
		maxOpenConns: maxOpenConns,
		maxIdleConns: maxIdleConns,
		timezone:     timezone,
		op:           op,
	}
	db.sqlDB.Store(sqlDB)
	openDBs[key] = &cachedDB{db: db, refCount: 1}
	return db, nil
}

// UpsertNoConflict réalise un SELECT d'existence puis UPDATE ou INSERT, le tout
// sous WithReopenOnInvalidated.
//
// C'est le pattern anti-ART canonique pour les tables à clé naturelle sur une DB
// partagée process-level (metadata.duckdb : asset_index, waypoint_assets_raw,
// playlists_catalog…). Il combine les DEUX couches déjà en place dans le projet :
//
//  1. Éviter le déclencheur : pas de `INSERT ... ON CONFLICT DO UPDATE`, qui sur
//     DuckDB s'implémente en DELETE+INSERT et déclenche « Failed to delete all
//     rows from index » → FATAL-invalidation du handle pour tout le process
//     (cf. ADR 0019, art_upsert_patterns_test, thought_log 2026-05-30).
//  2. Auto-réparation : WithReopenOnInvalidated ré-ouvre + retry si le handle a
//     été invalidé par un autre writer.
//
// existsQuery doit retourner 0 ou 1 ligne (typiquement `SELECT 1 FROM t WHERE
// <pk...>`). Les écritures concurrentes sur la MÊME clé doivent être sérialisées
// par le caller (WriteQueue, write lease…) : ce helper ne protège pas la course
// SELECT→INSERT entre goroutines.
// UpsertRowNoConflict fait un SELECT d'existence puis UPDATE (si présent) ou INSERT
// (sinon) sur un *sql.DB brut — ART-safe : JAMAIS d'ON CONFLICT sur la PK (qui réécrit
// via l'index ART DuckDB, bug #23046). existsQuery doit retourner ≥1 ligne si la clé
// existe.
//
// SOURCE UNIQUE (K1d, dédup #6, 2026-07-05) : ce pattern était copié-collé dans
// ops/catalog_refresh.upsertNoConflict, service.CatalogFetcherService.upsertRowNoConflict
// et api/registry_catalog_expand.upsertPlaylistWeight. La méthode (*DB).UpsertNoConflict
// délègue ici depuis son wrapper reopen-on-invalidated.
func UpsertRowNoConflict(
	ctx context.Context, db *sql.DB,
	existsQuery string, existsArgs []any,
	updateQuery string, updateArgs []any,
	insertQuery string, insertArgs []any,
) error {
	var dummy int
	err := db.QueryRowContext(ctx, existsQuery, existsArgs...).Scan(&dummy)
	switch {
	case err == nil:
		_, execErr := db.ExecContext(ctx, updateQuery, updateArgs...)
		return execErr
	case errors.Is(err, sql.ErrNoRows):
		_, execErr := db.ExecContext(ctx, insertQuery, insertArgs...)
		return execErr
	default:
		return err
	}
}

func (db *DB) UpsertNoConflict(
	ctx context.Context,
	existsQuery string, existsArgs []any,
	updateQuery string, updateArgs []any,
	insertQuery string, insertArgs []any,
) error {
	return db.WithReopenOnInvalidated(func() error {
		return UpsertRowNoConflict(ctx, db.loadSQL(),
			existsQuery, existsArgs, updateQuery, updateArgs, insertQuery, insertArgs)
	})
}

// openSQLDBFor construit un *sql.DB depuis un DSN + timezone, avec ping.
// Extrait de openCachedDB pour réutilisation par Reopen.
func openSQLDBFor(dsn, timezone, op, path string) (*sql.DB, error) {
	// Toutes les connexions passent par le connector pour appliquer les limites
	// ressources (memory_limit + threads, J2) + la timezone. Auparavant seule la
	// branche timezone!="" avait un hook d'init → la borne mémoire manquait sur
	// les DBs ouvertes sans TZ (ex. metadata) = risque OOM sur VPS contraint.
	connector, err := duckdb.NewConnector(dsn, func(execer driver.ExecerContext) error {
		return applyDuckSessionInit(execer, timezone)
	})
	if err != nil {
		return nil, fmt.Errorf("duckdb.%s connector(%s): %w", op, path, err)
	}
	sqlDB := sql.OpenDB(connector)
	if err := sqlDB.PingContext(context.Background()); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("duckdb.%s ping(%s): %w", op, path, err)
	}
	return sqlDB, nil
}

// applyDuckSessionInit borne les ressources de CHAQUE connexion (J2) puis applique
// la timezone si fournie. memory_limit protège le conteneur d'un OOM ; threads borne
// le parallélisme au nombre de vCPU. Cf. vars duckMemoryLimit / duckThreads.
func applyDuckSessionInit(execer driver.ExecerContext, timezone string) error {
	ctx := context.Background()
	stmts := []string{
		"SET memory_limit='" + duckMemoryLimit + "'",
		"SET threads=" + strconv.Itoa(duckThreads),
	}
	if timezone != "" {
		stmts = append(stmts, "SET TimeZone='"+timezone+"'")
	}
	for _, s := range stmts {
		if _, err := execer.ExecContext(ctx, s, nil); err != nil {
			return fmt.Errorf("duckdb session init %q: %w", s, err)
		}
	}
	return nil
}

// applyConnLimits applique maxOpen/maxIdle avec la logique d'openCachedDB
// (defaults à 1, écrasés si > 1).
func applyConnLimits(sqlDB *sql.DB, maxOpenConns, maxIdleConns int) {
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if maxOpenConns > 1 {
		sqlDB.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 1 {
		sqlDB.SetMaxIdleConns(maxIdleConns)
	}
}
