package handlers

// settings_data.go — les données de l'application de bureau dans les réglages (lot L22) : validation du
// PATCH (capacités, bornes, dossier de sauvegarde), effet à chaud du plafond mémoire, et l'état vivant exposé
// par GET /settings/data (dernière sauvegarde, mémoire). POST /settings/data/backup/run lance une
// sauvegarde à la demande.

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/ops/parquetbackup"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/procmem"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// Codes des refus des réglages de données (le texte vient du catalogue `settings.data.*`).
const (
	codeDataNotOffered    = "data_setting_not_offered"
	codeBackupInterval    = "backup_interval_invalid"
	codeBackupKeep        = "backup_keep_invalid"
	codeMemoryLimit       = "memory_limit_invalid"
	codeBackupDirNotAbs   = "backup_dir_not_absolute"
	codeBackupDirMissing  = "backup_dir_missing"
	codeBackupDirNotWrite = "backup_dir_not_writable"
	codeBackupDirInData   = "backup_dir_inside_data"
	codeBackupDisabled    = "backup_disabled"
	codeBackupRunning     = "backup_already_running"
)

// validateData juge le PATCH sur les données : chaque clé suit SA capacité (ADR 0044 D-4), les valeurs
// restent dans leurs bornes et le dossier de sauvegarde est vérifié (existe, accessible en écriture, hors
// de la racine de données). `next` est le réglage APRÈS application.
func (h *SettingsHandler) validateData(ctx context.Context, req *domain.UpdateSettingsRequest, next *settings_platform.AppSettings) error {
	d := &req.DataUpdate
	if d.BackupDir == nil && d.BackupInterval == nil && d.BackupKeep == nil && d.MemoryLimitMB == nil {
		return nil
	}
	loc := i18n.FromContext(ctx)
	caps := h.cfg.Capabilities
	backupKey := d.BackupDir != nil || d.BackupInterval != nil || d.BackupKeep != nil
	if (backupKey && !caps.ParquetBackup) || (d.MemoryLimitMB != nil && !caps.MemoryLimit) {
		slog.WarnContext(ctx, "settings: réglage de données refusé (capacité absente)")
		return humacore.NewError(http.StatusBadRequest, codeDataNotOffered, i18n.Msg(loc, "settings.data.not_offered", nil))
	}
	if err := settings_platform.CheckData(next, d); err != nil {
		switch {
		case errors.Is(err, settings_platform.ErrBackupInterval):
			return humacore.NewError(http.StatusBadRequest, codeBackupInterval, i18n.Msg(loc, "settings.data.interval_invalid", nil))
		case errors.Is(err, settings_platform.ErrBackupKeep):
			return humacore.NewError(http.StatusBadRequest, codeBackupKeep, i18n.Msg(loc, "settings.data.keep_invalid", nil))
		default:
			return humacore.NewError(http.StatusBadRequest, codeMemoryLimit, i18n.Msg(loc, "settings.data.memory_invalid", nil))
		}
	}
	if d.BackupDir == nil {
		return nil
	}
	next.BackupDir = strings.TrimSpace(*d.BackupDir)
	if next.BackupDir == "" {
		return nil // désactivation : rien à vérifier
	}
	return h.validateBackupDir(ctx, next.BackupDir)
}

// validateBackupDir vérifie le dossier de sauvegarde choisi.
func (h *SettingsHandler) validateBackupDir(ctx context.Context, dir string) error {
	loc := i18n.FromContext(ctx)
	_, err := parquetbackup.ValidateDest(dir, h.cfg.PathResolver())
	switch {
	case err == nil:
		return nil
	case errors.Is(err, parquetbackup.ErrDestNotAbsolute):
		return humacore.NewError(http.StatusBadRequest, codeBackupDirNotAbs, i18n.Msg(loc, "settings.data.dir_not_absolute", nil))
	case errors.Is(err, parquetbackup.ErrDestMissing):
		return humacore.NewError(http.StatusBadRequest, codeBackupDirMissing, i18n.Msg(loc, "settings.data.dir_missing", i18n.Params{"dir": dir}))
	case errors.Is(err, parquetbackup.ErrDestInsideData):
		return humacore.NewError(http.StatusBadRequest, codeBackupDirInData, i18n.Msg(loc, "settings.data.dir_inside_data", nil))
	default:
		slog.WarnContext(ctx, "settings: dossier de sauvegarde non accessible en écriture", "dir", dir, "err", err)
		return humacore.NewError(http.StatusBadRequest, codeBackupDirNotWrite, i18n.Msg(loc, "settings.data.dir_not_writable", i18n.Params{"dir": dir}))
	}
}

// applyDataRuntime porte à chaud le plafond mémoire enregistré : les nouvelles connexions le reçoivent, les
// bases déjà ouvertes aussi (SET memory_limit).
func (h *SettingsHandler) applyDataRuntime(ctx context.Context, req *domain.UpdateSettingsRequest, cfg *settings_platform.AppSettings) {
	if req.MemoryLimitMB == nil || !h.cfg.Capabilities.MemoryLimit {
		return
	}
	duckdb.SetMemoryLimitMB(cfg.MemoryLimitMB)
	done, failed := duckdb.ApplyMemoryLimitToOpen(ctx)
	slog.InfoContext(ctx, "settings: plafond mémoire des bases appliqué",
		"memory_limit_mb", cfg.MemoryLimitMB, "effectif", duckdb.EffectiveMemoryLimit(), "bases", done, "echecs", failed)
}

// dataStatusOutput : le corps de GET /settings/data.
type dataStatusOutput struct {
	Body *domain.DataStatus
}

// handleGetDataStatus expose l'état vivant des données : dernière sauvegarde du dossier choisi et mémoire du
// processus. Les sections que le profil n'offre pas sont absentes.
// GET /settings/data
func (h *SettingsHandler) handleGetDataStatus(ctx context.Context, _ *struct{}) (*dataStatusOutput, error) {
	cfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "settings_load_error", err)
	}
	out := &domain.DataStatus{}
	if h.cfg.Capabilities.ParquetBackup {
		out.Backup = h.backupStatus(cfg)
	}
	if h.cfg.Capabilities.MemoryLimit {
		out.Memory = &domain.MemoryStatus{RSSMB: procmem.RSSMB(), LimitMB: cfg.MemoryLimitMB,
			EffectiveLimit: duckdb.EffectiveMemoryLimit()}
	}
	return &dataStatusOutput{Body: out}, nil
}

// backupStatus compose l'état de la sauvegarde périodique pour le réglage courant.
func (h *SettingsHandler) backupStatus(cfg *settings_platform.AppSettings) *domain.DataBackupStatus {
	dir := strings.TrimSpace(cfg.BackupDir)
	st := parquetbackup.Default().Status(dir)
	out := &domain.DataBackupStatus{Enabled: dir != "", Running: st.Running}
	if dir != "" {
		if check, err := parquetbackup.ValidateDest(dir, h.cfg.PathResolver()); err == nil {
			out.SameVolume = check.SameVolume
		}
	}
	if st.Last != nil {
		out.Last = &domain.DataBackupSummary{Name: st.Last.Name, CreatedAt: st.Last.CreatedAt, Bytes: st.Last.Bytes,
			Databases: st.Last.Databases, DurationMs: st.Last.DurationMs, Partial: st.Last.Partial, AppVersion: st.Last.AppVersion}
	}
	if st.LastAttempt != nil {
		out.LastAttempt = &domain.DataBackupAttempt{At: st.LastAttempt.At, OK: st.LastAttempt.OK, ErrorCode: st.LastAttempt.Code}
	}
	return out
}

// dataBackupRunOutput : le corps de POST /settings/data/backup/run.
type dataBackupRunOutput struct {
	Body *domain.DataBackupRunResponse
}

// handlePostDataBackupRun lance une sauvegarde maintenant, en arrière-plan (202) ; l'état se lit par
// GET /settings/data. Refusée hors de la capacité, sans dossier choisi (409) ou si une sauvegarde est en cours (409).
// POST /settings/data/backup/run
func (h *SettingsHandler) handlePostDataBackupRun(ctx context.Context, _ *struct{}) (*dataBackupRunOutput, error) {
	if !h.cfg.Capabilities.ParquetBackup {
		return nil, humacore.NewError(http.StatusBadRequest, codeDataNotOffered,
			i18n.Msg(i18n.FromContext(ctx), "settings.data.not_offered", nil))
	}
	switch err := parquetbackup.Default().Start(); {
	case err == nil:
		return &dataBackupRunOutput{Body: &domain.DataBackupRunResponse{Started: true}}, nil
	case errors.Is(err, parquetbackup.ErrDisabled):
		return nil, humacore.NewError(http.StatusConflict, codeBackupDisabled, "backup disabled")
	case errors.Is(err, parquetbackup.ErrAlreadyRunning):
		return nil, humacore.NewError(http.StatusConflict, codeBackupRunning, "backup running")
	default:
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, "backup_unavailable", err)
	}
}
