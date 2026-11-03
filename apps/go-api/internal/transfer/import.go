package transfer

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/persist"
)

// Bornes de l'import.
const (
	// DefaultMaxUncompressed : taille maximale cumulée des tables d'une archive (8 Gio).
	DefaultMaxUncompressed = int64(8) << 30
	maxManifestBytes       = 1 << 20
)

// Acquire rend un handle d'écriture sur une base, sous son bail, et la fonction qui le rend.
type Acquire func(ctx context.Context) (*sql.DB, func(), error)

// ImportInput décrit un import.
type ImportInput struct {
	// ZipPath : l'archive à importer.
	ZipPath string
	// ExpectXuid : le joueur suivi localement ; l'archive d'un autre joueur est refusée.
	ExpectXuid string
	// AcquireShared et AcquirePlayer : les chemins d'écriture existants de l'application (fournisseur de la
	// base partagée, bail et ouverture de la base joueur). AcquirePlayer nil : les tables joueur sont sautées.
	AcquireShared, AcquirePlayer Acquire
	// WorkDir : dossier de travail (extraction) ; vide = dossier temporaire du système.
	WorkDir string
	// MaxUncompressed : borne de la taille cumulée (zéro = [DefaultMaxUncompressed]).
	MaxUncompressed int64
}

// ImportResult résume un import.
type ImportResult struct {
	Manifest *Manifest
	// Inserted : lignes insérées par table ("shared/match_registry").
	Inserted map[string]int64
}

// Import vérifie l'archive (manifeste, joueur attendu, tables du périmètre, sommes SHA-256, taille) puis
// écrit ses tables par persist.ImportPersister : INSERT seulement, une transaction par base, une ligne déjà
// présente n'est jamais réécrite. Rien n'est écrit avant la fin de la vérification.
func Import(ctx context.Context, in ImportInput) (*ImportResult, error) {
	work, err := os.MkdirTemp(in.WorkDir, "levelup-import-")
	if err != nil {
		return nil, fmt.Errorf("dossier de travail : %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	m, err := extract(in, work)
	if err != nil {
		return nil, err
	}
	expect, err := NormalizeXuid(in.ExpectXuid)
	if err != nil {
		return nil, err
	}
	if m.Xuid != expect {
		slog.WarnContext(ctx, "transfer: archive d'un autre joueur refusée", "attendu", expect, "archive", m.Xuid)
		return nil, ErrXuidMismatch
	}
	res := &ImportResult{Manifest: m, Inserted: map[string]int64{}}
	for _, scope := range []struct {
		name string
		acq  Acquire
	}{{ScopeShared, in.AcquireShared}, {ScopePlayer, in.AcquirePlayer}} {
		if err := importScope(ctx, scope.name, scope.acq, m, work, res); err != nil {
			return res, err
		}
	}
	slog.InfoContext(ctx, "transfer: import terminé", "xuid", m.Xuid, "titre", m.Title, "tables", len(res.Inserted))
	return res, nil
}

// importScope écrit les tables d'une base dans une seule transaction.
func importScope(ctx context.Context, scope string, acq Acquire, m *Manifest, work string, res *ImportResult) error {
	var tables []persist.ImportTable
	for _, t := range m.Tables {
		if t.Scope != scope {
			continue
		}
		spec, err := specFor(t.Scope, t.Name)
		if err != nil {
			return err
		}
		tables = append(tables, persist.ImportTable{Name: t.Name, File: filepath.Join(work, filepath.FromSlash(t.File)), Key: spec.Key})
	}
	if len(tables) == 0 {
		return nil
	}
	if acq == nil {
		slog.WarnContext(ctx, "transfer: base non branchée, tables sautées", "scope", scope, "tables", len(tables))
		return nil
	}
	db, release, err := acq(ctx)
	if err != nil {
		return fmt.Errorf("écriture de la base %s : %w", scope, err)
	}
	defer release()
	counts, err := persist.NewImportPersister(db).Persist(ctx, tables)
	if err != nil {
		return fmt.Errorf("import de la base %s : %w", scope, err)
	}
	for name, n := range counts {
		res.Inserted[scope+"/"+name] = n
	}
	return nil
}

// extract décompresse l'archive dans work en vérifiant tout : seuls le manifeste et les entrées du périmètre
// sont lus (jamais un nom libre : pas de traversée de dossier), la taille est bornée à l'extraction, chaque
// fichier correspond à sa somme du manifeste.
func extract(in ImportInput, work string) (*Manifest, error) {
	zr, err := zip.OpenReader(in.ZipPath)
	if err != nil {
		return nil, fmt.Errorf("%w : %v", ErrBadArchive, err)
	}
	defer func() { _ = zr.Close() }()
	m, err := readManifest(zr)
	if err != nil {
		return nil, err
	}
	limit := in.MaxUncompressed
	if limit <= 0 {
		limit = DefaultMaxUncompressed
	}
	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		byName[f.Name] = f
	}
	var total int64
	for _, t := range m.Tables {
		if !entryName.MatchString(t.File) || t.File != t.Scope+"/"+t.Name+".parquet" {
			return nil, fmt.Errorf("%w : entrée %q", ErrBadArchive, t.File)
		}
		if _, err := specFor(t.Scope, t.Name); err != nil {
			return nil, err
		}
		zf := byName[t.File]
		if zf == nil {
			return nil, fmt.Errorf("%w : entrée %q absente", ErrBadArchive, t.File)
		}
		total += int64(zf.UncompressedSize64)
		if total > limit || zf.UncompressedSize64 > uint64(limit) {
			return nil, ErrTooLarge
		}
		if err := extractFile(zf, filepath.Join(work, filepath.FromSlash(t.File)), t, limit); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// readManifest lit et valide le manifeste.
func readManifest(zr *zip.ReadCloser) (*Manifest, error) {
	for _, f := range zr.File {
		if f.Name != ManifestFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%w : %v", ErrBadArchive, err)
		}
		defer func() { _ = rc.Close() }()
		data, err := io.ReadAll(io.LimitReader(rc, maxManifestBytes+1))
		if err != nil || len(data) > maxManifestBytes {
			return nil, fmt.Errorf("%w : manifeste", ErrBadArchive)
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("%w : manifeste : %v", ErrBadArchive, err)
		}
		if m.Format != FormatVersion {
			return nil, ErrFormatTooNewer
		}
		if _, err := NormalizeXuid(m.Xuid); err != nil {
			return nil, err
		}
		sort.SliceStable(m.Tables, func(i, j int) bool { return m.Tables[i].File < m.Tables[j].File })
		return &m, nil
	}
	return nil, fmt.Errorf("%w : manifeste absent", ErrBadArchive)
}

// extractFile écrit une entrée et vérifie sa taille et sa somme.
func extractFile(zf *zip.File, dst string, t Table, limit int64) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	rc, err := zf.Open()
	if err != nil {
		return fmt.Errorf("%w : %v", ErrBadArchive, err)
	}
	defer func() { _ = rc.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // nom validé par entryName
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(rc, limit+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("%w : %v", ErrBadArchive, err)
	}
	if n > limit {
		return ErrTooLarge
	}
	sum, size, err := hashFile(dst)
	if err != nil {
		return err
	}
	if sum != t.SHA256 || size != t.Bytes {
		return fmt.Errorf("%w : somme de %s", ErrBadArchive, t.File)
	}
	return nil
}
