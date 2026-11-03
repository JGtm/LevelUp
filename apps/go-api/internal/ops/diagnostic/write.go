package diagnostic

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/procmem"
)

// Réglages du rapport.
const (
	// DirName : sous-dossier de la racine de données qui reçoit les rapports.
	DirName = "diagnostics"
	// DefaultMaxLogBytes : la fin de chaque journal retenue (5 Mo).
	DefaultMaxLogBytes = 5 << 20
	namePrefix         = "levelup-diag-"
	nameLayout         = "20060102-1504"
	perDBTimeout       = 30 * time.Second
)

// Input rassemble ce que le rapport décrit. Le paquet ne lit ni l'environnement ni la configuration : tout est
// passé par l'appelant.
type Input struct {
	// Paths : le résolveur de chemins (racine de données, bases).
	Paths *title.PathResolver
	// LogsRoot : le dossier des journaux (fichiers .log).
	LogsRoot string
	// AppVersion et Deployment : la version et le profil, pour l'affichage.
	AppVersion, Deployment string
	// Capabilities : les capacités du profil (sérialisées, sans secret par construction).
	Capabilities any
	// SettingsFile : chemin d'app_settings.json (lu puis expurgé).
	SettingsFile string
	// Extras : documents supplémentaires, nom de fichier (sans extension) -> valeur (tâches planifiées, mise à jour).
	Extras map[string]any
	// Secrets : valeurs exactes à effacer partout (secret du lanceur, secret de session).
	Secrets []string
	// MaxLogBytes : la fin de chaque journal retenue (zéro = [DefaultMaxLogBytes]).
	MaxLogBytes int64
	// Now : l'horloge (tests) ; nil = time.Now.
	Now func() time.Time
}

func (in *Input) now() time.Time {
	if in.Now != nil {
		return in.Now()
	}
	return time.Now()
}

// piece : un document du rapport et la fonction qui le produit.
type piece struct {
	name string
	make func() ([]byte, error)
}

// Write produit le rapport dans <racine de données>/diagnostics/levelup-diag-<AAAAMMJJ-HHMM>.zip et rend son
// chemin. L'archive est écrite sous un nom temporaire puis renommée : un fichier `.zip` est toujours complet.
func Write(ctx context.Context, in Input) (string, error) {
	dir := filepath.Join(in.Paths.DataRoot(), DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("dossier des rapports : %w", err)
	}
	final, err := freshName(dir, in.now())
	if err != nil {
		return "", err
	}
	tmp := final + ".partial"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec // sous la racine de données
	if err != nil {
		return "", fmt.Errorf("rapport : %w", err)
	}
	zw := zip.NewWriter(f)
	buildErr := fill(ctx, zw, in)
	if cerr := zw.Close(); buildErr == nil {
		buildErr = cerr
	}
	if cerr := f.Close(); buildErr == nil {
		buildErr = cerr
	}
	if buildErr != nil {
		_ = os.Remove(tmp)
		return "", buildErr
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rapport, renommage final : %w", err)
	}
	slog.InfoContext(ctx, "diagnostic: rapport écrit", "path", final)
	return final, nil
}

// freshName rend un nom de fichier libre pour l'instant now.
func freshName(dir string, now time.Time) (string, error) {
	base := namePrefix + now.Format(nameLayout)
	for i := 1; i < 100; i++ {
		name := base + ".zip"
		if i > 1 {
			name = fmt.Sprintf("%s-%d.zip", base, i)
		}
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p, nil
		}
	}
	return "", fmt.Errorf("trop de rapports à la même minute dans %s", dir)
}

// fill écrit chaque pièce du rapport. Une pièce qui échoue est remplacée par une note dans l'archive (le
// rapport d'une machine en difficulté doit sortir quand même) ; seule l'écriture du zip lui-même est fatale.
func fill(ctx context.Context, zw *zip.Writer, in Input) error {
	names := make([]string, 0, len(in.Extras))
	for n := range in.Extras {
		names = append(names, n)
	}
	sort.Strings(names)
	pieces := make([]piece, 0, 2+len(names))
	pieces = append(pieces,
		piece{"app_info.json", func() ([]byte, error) { return jsonDoc(appInfo(in), in.Secrets) }},
		piece{"settings.json", func() ([]byte, error) { return settingsDoc(in) }})
	for _, n := range names {
		v := in.Extras[n]
		pieces = append(pieces, piece{n + ".json", func() ([]byte, error) { return jsonDoc(v, in.Secrets) }})
	}
	for _, p := range pieces {
		if err := addPiece(ctx, zw, p); err != nil {
			return err
		}
	}
	if err := addDatabases(ctx, zw, in); err != nil {
		return err
	}
	return addLogs(ctx, zw, in)
}

// addPiece écrit une pièce ; son échec laisse une note `<nom>.error.txt`.
func addPiece(ctx context.Context, zw *zip.Writer, p piece) error {
	data, err := p.make()
	if err != nil {
		slog.WarnContext(ctx, "diagnostic: pièce non produite", "piece", p.name, "err", err)
		return writeEntry(zw, p.name+".error.txt", []byte("pièce non produite : "+Scrub(err.Error(), nil)))
	}
	return writeEntry(zw, p.name, data)
}

func writeEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// jsonDoc sérialise v, l'expurge (clés interdites, valeurs exactes) et l'indente.
func jsonDoc(v any, secrets []string) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return json.MarshalIndent(RedactValue(decoded, secrets), "", "  ")
}

// settingsDoc : app_settings.json expurgé (un fichier absent ou illisible est dit, pas tu).
func settingsDoc(in Input) ([]byte, error) {
	if in.SettingsFile == "" {
		return jsonDoc(map[string]string{"note": "aucun fichier de réglages"}, in.Secrets)
	}
	raw, err := os.ReadFile(in.SettingsFile)
	if errors.Is(err, os.ErrNotExist) {
		return jsonDoc(map[string]string{"note": "app_settings.json absent : valeurs par défaut"}, in.Secrets)
	}
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("app_settings.json illisible : %w", err)
	}
	return json.MarshalIndent(RedactValue(decoded, in.Secrets), "", "  ")
}

// appInfo : la version, le profil, les racines, les capacités et l'état du processus.
func appInfo(in Input) map[string]any {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	roots := in.Paths.Roots()
	return map[string]any{
		"app_version":     in.AppVersion,
		"deployment":      in.Deployment,
		"generated_at":    in.now().UTC().Format(time.RFC3339),
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"go_version":      runtime.Version(),
		"cpus":            runtime.NumCPU(),
		"goroutines":      runtime.NumGoroutine(),
		"heap_alloc_mb":   ms.HeapAlloc >> 20,
		"process_rss_mb":  procmem.RSSMB(),
		"duckdb_limit":    duckdb.EffectiveMemoryLimit(),
		"roots":           map[string]string{"resources": roots.Resources, "data": roots.Data, "logs": roots.Logs},
		"capabilities":    in.Capabilities,
		"film_archive":    in.Paths.FilmArchiveDir(),
		"report_excludes": "jetons, secrets, codes d'autorisation, contenu des bases",
	}
}

// addDatabases écrit le diagnostic de schéma de chaque base, lu par la porte de lecture du serveur.
func addDatabases(ctx context.Context, zw *zip.Writer, in Input) error {
	dbs, err := ops.BackupDatabases(in.Paths)
	if err != nil {
		return writeEntry(zw, "databases/_error.txt", []byte("liste des bases impossible : "+Scrub(err.Error(), nil)))
	}
	for _, db := range dbs {
		name := "databases/" + strings.NewReplacer(":", "_", " ", "_").Replace(db.Key) + ".txt"
		if err := writeEntry(zw, name, []byte(diagnoseOne(ctx, db))); err != nil {
			return err
		}
	}
	return nil
}

// diagnoseOne rend le diagnostic d'une base, ou la raison pour laquelle il n'a pas pu être fait.
func diagnoseOne(ctx context.Context, db ops.BackupDB) string {
	cctx, cancel := context.WithTimeout(ctx, perDBTimeout)
	defer cancel()
	h, release, err := duckdb.OpenReadForQuery(db.Path)
	if err != nil {
		slog.WarnContext(ctx, "diagnostic: base non lue", "key", db.Key, "err", err)
		return "base non lue : " + Scrub(err.Error(), nil) + "\n"
	}
	defer release()
	report, err := ops.DiagnoseHandle(cctx, h, ops.DiagnoseOptions{DBPath: db.Key, Verbose: true})
	if err != nil {
		slog.WarnContext(ctx, "diagnostic: diagnostic de base échoué", "key", db.Key, "err", err)
		return ops.FormatDiagnoseReport(report) + "\nerreur : " + Scrub(err.Error(), nil) + "\n"
	}
	return ops.FormatDiagnoseReport(report)
}

// addLogs écrit la fin de chaque journal (`*.log`, `*.log.1`) du dossier des journaux, expurgée.
func addLogs(ctx context.Context, zw *zip.Writer, in Input) error {
	limit := in.MaxLogBytes
	if limit <= 0 {
		limit = DefaultMaxLogBytes
	}
	var files []string
	for _, pat := range []string{"*.log", "*.log.1"} {
		m, err := filepath.Glob(filepath.Join(in.LogsRoot, pat))
		if err != nil {
			return err
		}
		files = append(files, m...)
	}
	sort.Strings(files)
	for _, p := range files {
		data, err := tailFile(p, limit)
		if err != nil {
			slog.WarnContext(ctx, "diagnostic: journal non lu", "file", p, "err", err)
			data = []byte("journal non lu : " + Scrub(err.Error(), nil))
		}
		if err := writeEntry(zw, "logs/"+filepath.Base(p), []byte(ScrubLog(string(data), in.Secrets))); err != nil {
			return err
		}
	}
	return nil
}

// tailFile rend les limit derniers octets de path, commençant à une ligne entière.
func tailFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // journal du dossier des journaux
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	start := int64(0)
	if st.Size() > limit {
		start = st.Size() - limit
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if start > 0 {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}
	return data, nil
}
