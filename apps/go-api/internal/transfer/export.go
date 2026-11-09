package transfer

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// exportLeaseWait : attente du bail d'écriture d'une base à exporter (un lot d'écriture en cours se termine
// vite ; l'export est demandé par un joueur qui attend).
const exportLeaseWait = 15 * time.Second

// exportedScope : une base exportée, son chemin et ses tables.
type exportedScope struct {
	scope, path string
	specs       []tableSpec
}

// ExportInput décrit un export.
type ExportInput struct {
	// Paths : le résolveur de chemins de l'instance qui exporte.
	Paths *title.PathResolver
	// Slug : le titre exporté.
	Slug string
	// Xuid et Gamertag : le joueur CONNECTÉ (jamais lu d'une requête libre : de sa session).
	Xuid, Gamertag string
	// PlayerGamertag : le gamertag du profil suivi qui porte CE xuid, trouvé par le xuid (ADR 0035) ; il
	// désigne la base joueur exportée. Vide : aucun profil suivi pour ce xuid, la portée joueur est sautée
	// (jamais la base d'un autre profil choisie par un gamertag de session).
	PlayerGamertag string
	AppVersion     string
	// Dest : le fichier zip à écrire.
	Dest string
	Now  func() time.Time
}

// Export écrit dans in.Dest l'archive du joueur et rend son manifeste. Les bases se LISENT sous leur bail
// d'écriture (duckdb.WithHeldRead) ; aucune n'est modifiée. Le fichier est écrit sous un nom temporaire puis
// renommé.
func Export(ctx context.Context, in ExportInput) (*Manifest, error) {
	xuid, err := NormalizeXuid(in.Xuid)
	if err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp(filepath.Dir(in.Dest), ".export-work-")
	if err != nil {
		return nil, fmt.Errorf("dossier de travail : %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	now := time.Now
	if in.Now != nil {
		now = in.Now
	}
	m := &Manifest{Format: FormatVersion, Title: in.Slug, Xuid: xuid, Gamertag: in.Gamertag,
		CreatedAt: now().UTC(), AppVersion: in.AppVersion}
	scopes := []exportedScope{{ScopeShared, in.Paths.SharedDBPath(in.Slug), sharedTables}}
	if in.PlayerGamertag != "" {
		scopes = append(scopes, exportedScope{ScopePlayer, in.Paths.PlayerDBPath(in.Slug, in.PlayerGamertag), playerTables})
	} else {
		slog.WarnContext(ctx, "transfer: aucun profil suivi pour ce xuid, base joueur non exportée", "xuid", xuid)
	}
	for _, s := range scopes {
		if _, err := os.Stat(s.path); err != nil {
			slog.WarnContext(ctx, "transfer: base absente, portée ignorée", "scope", s.scope, "path", s.path)
			continue
		}
		err := duckdb.WithHeldRead(ctx, s.path, exportLeaseWait, func(db *sql.DB) error {
			return exportScope(ctx, db, s.scope, s.specs, xuid, work, m)
		})
		if err != nil {
			return nil, fmt.Errorf("export de la base %s : %w", s.scope, err)
		}
	}
	if err := writeArchive(in.Dest, work, m); err != nil {
		return nil, err
	}
	return m, nil
}

// exportScope exporte les tables d'une base qui existent, filtrées pour le joueur.
func exportScope(ctx context.Context, db *sql.DB, scope string, specs []tableSpec, xuid, work string, m *Manifest) error {
	if err := os.MkdirAll(filepath.Join(work, scope), 0o755); err != nil {
		return err
	}
	for _, s := range specs {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = ?`, s.Name).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			slog.WarnContext(ctx, "transfer: table absente, ignorée", "scope", scope, "table", s.Name)
			continue
		}
		rel := scope + "/" + s.Name + ".parquet"
		where := ""
		if scope == ScopeShared {
			where = " WHERE " + fmt.Sprintf(matchScopeFilter, "'"+xuid+"'")
		}
		q := fmt.Sprintf(`COPY (SELECT * FROM %q%s) TO '%s' (FORMAT PARQUET, COMPRESSION 'zstd')`,
			s.Name, where, strings.ReplaceAll(filepath.Join(work, filepath.FromSlash(rel)), "'", "''"))
		var rows int64
		if err := db.QueryRowContext(ctx, q).Scan(&rows); err != nil {
			return fmt.Errorf("table %s : %w", s.Name, err)
		}
		sum, size, err := hashFile(filepath.Join(work, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		m.Tables = append(m.Tables, Table{Scope: scope, Name: s.Name, File: rel, Rows: rows, Bytes: size, SHA256: sum})
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path) //nolint:gosec // fichier de travail de l'export
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// writeArchive assemble le zip (entrées non compressées) sous un nom temporaire, puis le renomme.
func writeArchive(dest, work string, m *Manifest) error {
	tmp := dest + ".partial"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec // fichier de l'export
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	err = func() error {
		for _, t := range m.Tables {
			if err := addStored(zw, t.File, filepath.Join(work, filepath.FromSlash(t.File))); err != nil {
				return err
			}
		}
		data, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return err
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: ManifestFile, Method: zip.Deflate})
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}()
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func addStored(zw *zip.Writer, name, src string) error {
	in, err := os.Open(src) //nolint:gosec // fichier de travail de l'export
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}
