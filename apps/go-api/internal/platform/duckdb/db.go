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
	"strings"
	"sync"
	"sync/atomic"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

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
		4,
		2,
		"OpenReadOnly",
		tz,
	)
}

// OpenReadWriteShared ouvre une base DuckDB en lecture-écriture avec un pool de connexions
// (4 conns max, comme OpenReadOnly). À utiliser pour les bases partagées (ex: metadata.duckdb)
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
	return openCachedDB("rw:"+path, path, path, 4, 2, "OpenReadWriteShared", tz)
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
	return openCachedDB("rw:"+path, path, path, 1, 1, "OpenReadWrite", tz)
}

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

// IsInvalidatedError détecte les erreurs qui marquent une connexion comme
// inutilisable et exigent un Reopen() pour récupérer. Trois classes :
//
//  1. FATAL DuckDB (ART corruption) :
//     "database has been invalidated because of a previous fatal error."
//     Cause racine : « Failed to delete all rows from index. Only deleted
//     N out of M rows. » sur un index ART avec valeurs NULL (duckdb#9277).
//
//  2. Handle fermée côté stdlib database/sql (corrigé 2026-05-25) :
//     "sql: database is closed" — retourné par toute opération sur un
//     *sql.DB après un Close(). Observé en prod 2026-05-25 11:20-11:23
//     en cascade massive (home + career + teammates + filters + explorer)
//     sur la player DB de JGtm. La DB physique reste saine, c'est juste
//     la handle Go périmée (close volontaire, swap RO↔RW, refcount cache).
//     Avant ce fix : WithReopenOnInvalidated retournait directement
//     l'erreur sans tenter Reopen → 500 cascade pendant plusieurs minutes.
//
//  3. Variantes driver-level "connection was closed".
//
// Exporté pour que les callers (repo, scheduler) puissent décider de :
//   - logger un incident métier (corruption d'index)
//   - tenter un Reopen() via WithReopenOnInvalidated
//   - propager au handler HTTP qui choisira la stratégie (503 + retry, etc.)
func IsInvalidatedError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "database has been invalidated") ||
		strings.Contains(s, "Failed to delete all rows from index") ||
		strings.Contains(s, "database must be restarted prior") ||
		strings.Contains(s, "sql: database is closed") ||
		strings.Contains(s, "connection was closed") ||
		strings.Contains(s, "database is closed")
}

// IsFileLockError détecte l'erreur DuckDB d'ouverture d'un fichier déjà
// verrouillé en écriture par un AUTRE process (mono-writer). Distinct de
// IsInvalidatedError (corruption/handle périmée) : ici la DB est saine, c'est
// une contention inter-process (CLI backfill concurrent, 2e instance serveur,
// hot-reload Air pas encore libéré). Permet aux callers d'émettre un message
// actionnable au lieu d'un "open rw" opaque (cf. spartan_cron Madina 2026-05-31).
func IsFileLockError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Could not set lock on file") ||
		strings.Contains(s, "Conflicting lock is held") ||
		strings.Contains(s, "different configuration") ||
		// Windows/DuckDB : un autre détenteur (process distinct OU instance in-process
		// avec une config différente) tient déjà le fichier. DuckDB ajoute toujours ce
		// marqueur EN — "File is already open in <exe> (PID N)" — indépendamment de la
		// locale OS (le message Win FR "utilisé par un autre processus" varie, pas lui).
		// Sans ça, le boot Air laissant un tmp/server.exe résiduel produit un WARN
		// par-joueur dans spartan_cron au lieu de l'ERROR agrégée. Cf. 2026-06-27.
		strings.Contains(s, "File is already open in")
}

// Reopen ferme la connexion actuelle et en ouvre une nouvelle avec les
// mêmes paramètres (DSN, max conns, timezone). Permet de récupérer d'une
// invalidation fatale sans redémarrer le serveur.
//
// Thread-safe via le mutex du cache. Tout autre *DB existant qui pointait
// sur la MÊME instance partagée sera également mis à jour (le pointeur
// sqlDB est remplacé in-place et la cache entry pointe sur le même *DB).
//
// Limitations :
//   - Les requêtes/transactions en cours sur l'ancien sqlDB échoueront
//     (comportement attendu : l'invalidation a déjà cassé ces requêtes).
//   - Si le ping de la nouvelle connexion échoue (fichier corrompu côté
//     OS), retourne l'erreur sans toucher au sqlDB existant.
func (db *DB) Reopen() error {
	if db == nil {
		return errors.New("duckdb.Reopen: nil DB")
	}
	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	// Construit le nouveau sqlDB avec la même config qu'à l'ouverture initiale.
	newSQLDB, err := openSQLDBFor(db.dsn, db.timezone, db.op, db.path)
	if err != nil {
		slog.Error("duckdb: Reopen a échoué (fichier inaccessible ?)",
			"path", db.path, "op", db.op, "err", err)
		return err
	}
	applyConnLimits(newSQLDB, db.maxOpenConns, db.maxIdleConns)

	// Ferme l'ancien sqlDB en best-effort (déjà invalidé côté DuckDB).
	if old := db.loadSQL(); old != nil {
		_ = old.Close()
	}
	db.sqlDB.Store(newSQLDB)
	db.closed.Store(false)

	// Restaure l'entrée cache pointant sur cette même *DB (refCount préservé
	// si déjà présent, sinon refCount=1).
	if db.cacheKey != "" {
		if cached, ok := openDBs[db.cacheKey]; ok {
			cached.db = db
		} else {
			openDBs[db.cacheKey] = &cachedDB{db: db, refCount: 1}
		}
	}

	slog.Info("duckdb: connexion ré-ouverte après invalidation",
		"path", db.path, "op", db.op)
	return nil
}

// WithReopenOnInvalidated exécute fn ; si fn renvoie une erreur
// d'invalidation détectée par IsInvalidatedError, fait un Reopen() et
// retry fn une fois. Sinon retourne l'erreur originale.
//
// Pattern à utiliser dans les repos qui font des opérations write sensibles
// (UPDATE/DELETE) sur des DB partagées process-level — ainsi un bug DuckDB
// transitoire n'invalide pas la DB pour toute la durée de vie du process.
//
// Le retry est borné à 1 : si la 2e tentative échoue aussi avec une
// invalidation, on remonte l'erreur (probablement une corruption durable
// du fichier qui nécessite intervention).
func (db *DB) WithReopenOnInvalidated(fn func() error) error {
	err := fn()
	if !IsInvalidatedError(err) {
		return err
	}
	slog.Warn("duckdb: connexion invalidée, tentative de reopen",
		"path", db.path, "err", err)
	if reopenErr := db.Reopen(); reopenErr != nil {
		return fmt.Errorf("reopen after invalidation: %w (original: %v)", reopenErr, err)
	}
	retryErr := fn()
	if retryErr != nil {
		// Retry échoué : incident à traiter par les ops (corruption persistante ?).
		slog.Error("duckdb: retry post-reopen a échoué",
			"path", db.path, "err", retryErr,
			"persistent_invalidation", IsInvalidatedError(retryErr))
	}
	return retryErr
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
func (db *DB) UpsertNoConflict(
	ctx context.Context,
	existsQuery string, existsArgs []any,
	updateQuery string, updateArgs []any,
	insertQuery string, insertArgs []any,
) error {
	return db.WithReopenOnInvalidated(func() error {
		var dummy int
		err := db.loadSQL().QueryRowContext(ctx, existsQuery, existsArgs...).Scan(&dummy)
		switch {
		case err == nil:
			_, execErr := db.loadSQL().ExecContext(ctx, updateQuery, updateArgs...)
			return execErr
		case errors.Is(err, sql.ErrNoRows):
			_, execErr := db.loadSQL().ExecContext(ctx, insertQuery, insertArgs...)
			return execErr
		default:
			return err
		}
	})
}

// openSQLDBFor construit un *sql.DB depuis un DSN + timezone, avec ping.
// Extrait de openCachedDB pour réutilisation par Reopen.
func openSQLDBFor(dsn, timezone, op, path string) (*sql.DB, error) {
	var sqlDB *sql.DB
	if timezone != "" {
		tz := timezone
		connector, err := duckdb.NewConnector(dsn, func(execer driver.ExecerContext) error {
			_, initErr := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
			return initErr
		})
		if err != nil {
			return nil, fmt.Errorf("duckdb.%s connector(%s): %w", op, path, err)
		}
		sqlDB = sql.OpenDB(connector)
	} else {
		var err error
		sqlDB, err = sql.Open("duckdb", dsn)
		if err != nil {
			return nil, fmt.Errorf("duckdb.%s(%s): %w", op, path, err)
		}
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("duckdb.%s ping(%s): %w", op, path, err)
	}
	return sqlDB, nil
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

// Close ferme la connexion DuckDB. À appeler au shutdown.
func (db *DB) Close() error {
	if db == nil || db.loadSQL() == nil {
		return nil
	}

	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	if db.closed.Load() {
		return nil
	}
	if db.cacheKey != "" {
		if cached, ok := openDBs[db.cacheKey]; ok {
			if cached.refCount > 1 {
				cached.refCount--
				return nil
			}
			delete(openDBs, db.cacheKey)
		}
	}
	db.closed.Store(true)
	return db.loadSQL().Close()
}

// QueryRow exécute une requête qui retourne exactement une ligne.
// L'erreur réelle de la requête est différée jusqu'à Scan ; on ne peut donc pas
// la logger ici. Les call sites critiques doivent capturer err sur Scan eux-mêmes.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.loadSQL().QueryRowContext(ctx, query, args...)
}

// Query exécute une requête qui retourne plusieurs lignes.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.loadSQL().QueryContext(ctx, query, args...)
	if err != nil {
		logDBError(ctx, "duckdb: query failed", db, query, err)
	}
	return rows, err
}

// Exec exécute une instruction sans valeur de retour.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	res, err := db.loadSQL().ExecContext(ctx, query, args...)
	if err != nil {
		logDBError(ctx, "duckdb: exec failed", db, query, err)
	}
	return res, err
}

// ExecRecovered exécute un Exec sous WithReopenOnInvalidated : si la connexion
// est invalidée par un bug DuckDB transitoire, fait un Reopen + retry une fois.
//
// À utiliser par défaut dans les repos qui écrivent sur des bases partagées
// process-level (shared_social.duckdb, notamment) — un seul incident de
// connexion ne doit pas casser l'API jusqu'au prochain restart.
func (db *DB) ExecRecovered(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var res sql.Result
	err := db.WithReopenOnInvalidated(func() error {
		var execErr error
		res, execErr = db.loadSQL().ExecContext(ctx, query, args...)
		return execErr
	})
	if err != nil {
		logDBError(ctx, "duckdb: exec failed (after recovery)", db, query, err)
	}
	return res, err
}

// QueryRecovered exécute une Query sous WithReopenOnInvalidated. Cf. ExecRecovered.
//
// ATTENTION : si l'invalidation se produit pendant l'itération des rows (donc
// après QueryRecovered a retourné), le wrapper ne peut plus reagir. Seule la
// requête initiale est protégée. Les rows.Next()/rows.Scan() restent
// vulnérables et doivent être audités dans les chemins critiques (rare).
func (db *DB) QueryRecovered(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	err := db.WithReopenOnInvalidated(func() error {
		var qerr error
		rows, qerr = db.loadSQL().QueryContext(ctx, query, args...)
		return qerr
	})
	if err != nil {
		logDBError(ctx, "duckdb: query failed (after recovery)", db, query, err)
	}
	return rows, err
}

// logDBError centralise le log d'erreur DuckDB avec contexte standard
// (path, op, query excerpt, err). Niveau Error — toute erreur DB est anormale
// par défaut ; les call sites qui tolèrent une erreur (ex. : fichier absent
// pour Stat) doivent éviter de passer par db.Query/Exec, ou wrapper l'err.
func logDBError(ctx context.Context, msg string, db *DB, query string, err error) {
	slog.ErrorContext(ctx, msg,
		"path", db.path,
		"op", db.op,
		"query_excerpt", queryExcerpt(query),
		"err", err,
	)
}

// queryExcerpt retourne les ~80 premiers caractères de la query, sur une seule
// ligne (newlines → espaces, espaces consécutifs collapsés). Évite de polluer
// la console avec des SQL multi-lignes tout en gardant assez d'info pour
// identifier la requête fautive.
func queryExcerpt(q string) string {
	const maxLen = 80
	// Collapse whitespace (newlines, tabs, multiples spaces) en un seul espace.
	collapsed := strings.Join(strings.Fields(q), " ")
	if len(collapsed) <= maxLen {
		return collapsed
	}
	// Tronquage avec ellipsis. Comptage byte = ok pour ASCII SQL ; runes pour
	// les cas avec accents dans les noms de tables/colonnes (rare).
	if cnt := 0; true {
		for i := range collapsed {
			if cnt == maxLen {
				return collapsed[:i] + "…"
			}
			cnt++
		}
	}
	return collapsed
}

// SQLDb retourne le *sql.DB sous-jacent (pour interop avec d'autres packages).
func (db *DB) SQLDb() *sql.DB { return db.loadSQL() }

// Path retourne le chemin de la base.
func (db *DB) Path() string { return db.path }
