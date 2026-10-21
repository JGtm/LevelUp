// Package ops — media_capture_recalc.go : RECALCUL DE LA FENÊTRE DES MÉDIAS DÉJÀ INDEXÉS selon
// la règle par source (media_capture_source.go).
//
// CE QUE DIT UNE LIGNE ANTÉRIEURE. Avant la règle par source, l'indexation rangeait TOUJOURS
// l'heure lue H en capture_start_utc, et capture_end_utc = H + durée (aucun autre écrivain ne
// touche ces deux colonnes : réconciliation d'extension, HLS et suppression écrivent d'autres
// colonnes). Une ligne dont capture_time_source est NULL a donc H = capture_start_utc, quelle que
// soit sa source. Le recalcul reconnaît la source (motif du nom, sinon date du fichier), refait
// la fenêtre par captureWindow, écrit les deux bornes ET la source : la ligne marquée n'est plus
// jamais reprise. Rejouer le recalcul ne change rien (idempotent).
//
// NOM RECONNU MAIS HEURE DIFFÉRENTE. Si le fuseau est connu et que l'heure lue dans le nom
// diffère de capture_start_utc, H ne venait pas du nom (indexation sans fuseau, ou fuseau
// changé depuis) mais de la date du fichier : la source retenue est alors la date du fichier.
//
// QUI LE LANCE. IndexMedia, à chaque indexation (envoi, balayage après synchronisation, bouton
// « Scanner » des réglages), sous son verrou et sur son handle — le serveur se corrige donc sans
// arrêt. La commande `levelup media-capture-times` le lance hors serveur : simulation sur une
// base quelconque (une copie si le serveur tourne), ou écriture serveur arrêté.
package ops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// CaptureRecalcCount : compte d'une source dans un recalcul.
type CaptureRecalcCount struct {
	Examined   int // lignes non marquées de cette source
	Shifted    int // lignes dont le début ou la fin change
	NoDuration int // vidéos sans durée connue (posées ponctuellement)
}

// CaptureRecalcReport : bilan d'un recalcul, par clé de source (captureTimeSources).
type CaptureRecalcReport struct {
	DryRun bool
	// SkippedNoTimezone : recalcul non fait faute de fuseau (cf. recalcCaptureTimes) ; rien n'est
	// lu ni écrit, les lignes restent à reprendre.
	SkippedNoTimezone bool
	Examined          int
	Shifted           int
	BySource          map[string]CaptureRecalcCount
}

// SourceKeys rend les clés de source du bilan, triées (affichage stable).
func (r CaptureRecalcReport) SourceKeys() []string {
	keys := make([]string, 0, len(r.BySource))
	for k := range r.BySource {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// errNoMediaFilesTable : la base ne porte pas de table media_files (pas une base shared_social,
// ou aucun média n'y a jamais été indexé).
var errNoMediaFilesTable = errors.New("pas de table media_files : ce n'est pas une base shared_social, ou aucun média n'y a encore été indexé")

// legacyCaptureRow : une ligne de media_files indexée avant la règle par source.
type legacyCaptureRow struct {
	ID       int64
	FileName string
	Kind     string
	Start    time.Time
	End      sql.NullTime
	Duration sql.NullFloat64
}

// captureRecalcUpdate : la réécriture d'une ligne.
type captureRecalcUpdate struct {
	ID     int64
	Times  captureTimes
	Source string
}

// recalcCaptureTimes recalcule les lignes non marquées de media_files (cf. l'en-tête). dryRun :
// rien n'est écrit, le bilan dit ce qui changerait. Sans la colonne capture_time_source (base
// jamais indexée depuis la règle), la simulation tient toutes les lignes pour antérieures ;
// l'écriture exige la colonne (ensureMediaTables). Sans table media_files : erreur explicite.
//
// SANS FUSEAU, RIEN. Le fuseau dit si l'heure d'un nom daté vient bien du nom
// (legacyCaptureSource) ; sans lui, une ligne dont H venait de la date du fichier serait rangée
// sous la source du nom, et le marquage rendrait l'erreur définitive. Le recalcul est alors
// sauté (SkippedNoTimezone) et retenté au passage suivant.
//
// Une écriture est suivie d'un CHECKPOINT (ADR 0021) : les lignes réécrites ne restent pas au
// WAL jusqu'à la fin d'une indexation, ni quand IndexMedia sort avant son propre CHECKPOINT.
func recalcCaptureTimes(ctx context.Context, db *sql.DB, loc *time.Location, dryRun bool) (CaptureRecalcReport, error) {
	report := CaptureRecalcReport{DryRun: dryRun, BySource: map[string]CaptureRecalcCount{}}
	if loc == nil {
		report.SkippedNoTimezone = true
		return report, nil
	}
	cols, err := mediaFilesColumnSet(ctx, db)
	if err != nil {
		return report, err
	}
	if len(cols) == 0 {
		return report, errNoMediaFilesTable
	}
	hasMarker := cols["capture_time_source"]
	if !hasMarker && !dryRun {
		return report, errors.New("recalcCaptureTimes: colonne capture_time_source absente (ensureMediaTables non passé)")
	}
	rows, err := loadLegacyCaptureRows(ctx, db, hasMarker)
	if err != nil {
		return report, err
	}
	updates := make([]captureRecalcUpdate, 0, len(rows))
	for _, r := range rows {
		up, shifted := recalcCaptureRow(r, loc)
		c := report.BySource[up.Source]
		c.Examined++
		if shifted {
			c.Shifted++
			report.Shifted++
		}
		if r.Kind == mediaKindVideo && !r.Duration.Valid {
			c.NoDuration++
		}
		report.BySource[up.Source] = c
		report.Examined++
		updates = append(updates, up)
	}
	if dryRun || len(updates) == 0 {
		return report, nil
	}
	if err := writeCaptureRecalc(ctx, db, updates); err != nil {
		return report, err
	}
	if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		return report, fmt.Errorf("recalcul des heures de capture: CHECKPOINT: %w", err)
	}
	return report, nil
}

// recalcCaptureRow rend la réécriture d'une ligne antérieure et si sa fenêtre change.
func recalcCaptureRow(r legacyCaptureRow, loc *time.Location) (captureRecalcUpdate, bool) {
	src := legacyCaptureSource(r.FileName, r.Start, loc)
	inst := &captureInstant{At: r.Start, Source: src}
	win := captureWindow(r.Kind, inst, r.Duration.Float64, r.Duration.Valid)
	shifted := !sameInstant(win.Start, &r.Start) || !sameInstant(win.End, nullTimePtr(r.End))
	return captureRecalcUpdate{ID: r.ID, Times: win, Source: src.Key}, shifted
}

// legacyCaptureSource : la source de l'heure H d'une ligne antérieure (cf. l'en-tête, « nom
// reconnu mais heure différente »).
func legacyCaptureSource(name string, h time.Time, loc *time.Location) captureTimeSource {
	src, ok := captureSourceOfName(name)
	if !ok {
		return captureSourceByKey(captureSourceFileDate)
	}
	if loc != nil {
		if inst := captureInstantFromName(name, loc); inst != nil && !inst.At.Equal(h.UTC()) {
			return captureSourceByKey(captureSourceFileDate)
		}
	}
	return src
}

func sameInstant(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// loadLegacyCaptureRows lit les lignes non marquées qui ont une heure de capture.
func loadLegacyCaptureRows(ctx context.Context, db *sql.DB, hasMarker bool) ([]legacyCaptureRow, error) {
	q := `SELECT id, COALESCE(file_name, ''), COALESCE(kind, ''), capture_start_utc,
			capture_end_utc, duration_seconds
		FROM media_files WHERE capture_start_utc IS NOT NULL`
	if hasMarker {
		q += ` AND capture_time_source IS NULL`
	}
	rows, err := db.QueryContext(ctx, q+` ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("lecture des médias à recalculer: %w", err)
	}
	defer rows.Close()
	var out []legacyCaptureRow
	for rows.Next() {
		var r legacyCaptureRow
		if err := rows.Scan(&r.ID, &r.FileName, &r.Kind, &r.Start, &r.End, &r.Duration); err != nil {
			return nil, fmt.Errorf("lecture d'un média à recalculer: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// writeCaptureRecalc écrit les réécritures en une transaction. Les colonnes touchées ne portent
// aucun index (seuls player_slug, created_at et (player_slug, file_stem) en ont) : une mise à
// jour ligne à ligne par clé primaire, comme les autres écritures de media_files. La garde
// `capture_time_source IS NULL` laisse intacte une ligne marquée entre la lecture et l'écriture.
func writeCaptureRecalc(ctx context.Context, db *sql.DB, updates []captureRecalcUpdate) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("recalcul des heures de capture: begin: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE media_files
		SET capture_start_utc = ?, capture_end_utc = ?, capture_time_source = ?
		WHERE id = ? AND capture_time_source IS NULL`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("recalcul des heures de capture: prepare: %w", err)
	}
	defer stmt.Close()
	for _, u := range updates {
		if _, err := stmt.ExecContext(ctx, u.Times.Start, u.Times.End, u.Source, u.ID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("recalcul des heures de capture: média %d: %w", u.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("recalcul des heures de capture: commit: %w", err)
	}
	return nil
}

// logCaptureRecalc journalise un bilan qui a réécrit des lignes.
func logCaptureRecalc(ctx context.Context, player string, r CaptureRecalcReport) {
	if r.Examined == 0 {
		return
	}
	args := []any{"player", player, "examined", r.Examined, "shifted", r.Shifted}
	for _, k := range r.SourceKeys() {
		args = append(args, "source_"+k, r.BySource[k].Shifted)
	}
	slog.InfoContext(ctx, "IndexMedia: fenêtres de capture recalculées selon leur source", args...)
}

// CaptureRecalcOptions configure RecalcMediaCaptureTimes.
type CaptureRecalcOptions struct {
	DBPath   string // shared_social.duckdb du titre (ou une copie, en simulation)
	Timezone string // IANA : fuseau des heures écrites dans les noms (cf. legacyCaptureSource)
	DryRun   bool
}

// RecalcMediaCaptureTimes lance le recalcul hors serveur. Simulation : ouverture en lecture
// seule, rien n'est écrit. Écriture : ouverture en écriture par le cache process (échoue si un
// autre processus, le serveur, tient la base), colonnes assurées, recalcul, CHECKPOINT (ADR 0021).
// Le fuseau est exigé (sans lui, recalcCaptureTimes ne fait rien) ; une base sans table media_files
// est refusée avec errNoMediaFilesTable.
func RecalcMediaCaptureTimes(ctx context.Context, opts CaptureRecalcOptions) (CaptureRecalcReport, error) {
	tz := SanitizeMediaTimezone(opts.Timezone)
	if tz == "" {
		return CaptureRecalcReport{}, fmt.Errorf("fuseau %q absent ou invalide : il dit si l'heure d'un nom daté vient du nom (user_timezone de app_settings.json)", opts.Timezone)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return CaptureRecalcReport{}, fmt.Errorf("fuseau %q: %w", tz, err)
	}
	if opts.DryRun {
		handle, err := platform_duckdb.OpenReadOnly(opts.DBPath, tz)
		if err != nil {
			return CaptureRecalcReport{}, fmt.Errorf("ouverture en lecture de %s: %w", opts.DBPath, err)
		}
		defer handle.Close()
		report, err := recalcCaptureTimes(ctx, handle.SQLDb(), loc, true)
		if err != nil {
			return report, fmt.Errorf("%s : %w", opts.DBPath, err)
		}
		return report, nil
	}
	unlock := indexLock(opts.DBPath)
	defer unlock()
	handle, err := platform_duckdb.OpenReadWriteShared(opts.DBPath, tz)
	if err != nil {
		return CaptureRecalcReport{}, fmt.Errorf("ouverture en écriture de %s: %w", opts.DBPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()
	if err := ensureMediaTables(ctx, db); err != nil {
		return CaptureRecalcReport{}, err
	}
	report, err := recalcCaptureTimes(ctx, db, loc, false)
	if err != nil {
		return report, err
	}
	if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		return report, fmt.Errorf("recalcul des heures de capture: CHECKPOINT: %w", err)
	}
	return report, nil
}

// recalcBeforeIndex : le recalcul tel qu'IndexMedia le lance, avant l'association. L'échec est
// journalisé ici et rendu à l'appelant, qui le range dans les erreurs non fatales de
// l'indexation (retenté au passage suivant).
func recalcBeforeIndex(ctx context.Context, db *sql.DB, loc *time.Location, player string) error {
	report, err := recalcCaptureTimes(ctx, db, loc, false)
	if err != nil {
		slog.ErrorContext(ctx, "IndexMedia: recalcul des fenêtres de capture échoué",
			"player", player, "err", err)
		return err
	}
	if report.SkippedNoTimezone {
		slog.WarnContext(ctx, "IndexMedia: recalcul des fenêtres de capture sauté, fuseau inconnu (retenté au passage suivant)",
			"player", player)
		return nil
	}
	logCaptureRecalc(ctx, player, report)
	return nil
}
