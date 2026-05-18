package openspartan

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// Reader holds a read-only handle on an OpenSpartan SQLite database.
// It is safe for concurrent use by readers; Close is idempotent.
//
// Always defer Close after a successful Open.
type Reader struct {
	db   *sql.DB
	path string

	mu     sync.Mutex
	closed bool
}

// Open returns a Reader bound to the OpenSpartan SQLite file at path. The
// file is opened read-only via a `file:` URI with `mode=ro`, and the schema
// is asserted before returning. If the file is not a recognizable OpenSpartan
// database, Open returns an error wrapping ErrNotOpenSpartanDB.
func Open(path string) (*Reader, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("openspartan: resolve path %q: %w", path, err)
	}

	// modernc.org/sqlite accepts SQLite URI syntax. Forward slashes are
	// required for Windows paths inside the URI.
	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(5000)", filepath.ToSlash(absPath))
	db, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("openspartan: open sqlite at %q: %w", absPath, err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("openspartan: ping %q: %w", absPath, err)
	}

	r := &Reader{db: db, path: absPath}
	if err := detectSchema(context.Background(), db); err != nil {
		_ = db.Close()
		slog.Warn("openspartan: schema detection failed", "path", absPath, "err", err)
		return nil, err
	}
	slog.Info("openspartan: reader opened", "path", absPath)
	return r, nil
}

// Close releases the underlying SQLite handle. Safe to call multiple times.
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	return r.db.Close()
}

// Path returns the absolute path the reader was opened from.
func (r *Reader) Path() string { return r.path }

// MatchCount returns the number of rows in MatchStats. Useful to short-circuit
// downstream work if the database is empty.
func (r *Reader) MatchCount(ctx context.Context) (int, error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return 0, ErrReaderClosed
	}
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM MatchStats`).Scan(&n); err != nil {
		return 0, fmt.Errorf("openspartan: count MatchStats: %w", err)
	}
	return n, nil
}
