// Package handlers — admin_logs.go : viewer de logs du dashboard monitoring.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// chi (middleware RequireAuth/RequireAdmin/NoStore hérités) et enregistre les
// 2 GET via huma.Get. Logique métier inchangée (ops.ListLogModules /
// ops.TailModuleLog), seul le wrapping HTTP change.
//
// Routes (RequireAuth+RequireAdmin+NoStore) :
//   - GET /admin/monitoring/logs/modules : fichiers logs/{module}.log dispo
//   - GET /admin/monitoring/logs/tail?module=&n=&level=&contains=&since=
//
// Lecture par la fin chunkée (ops.TailModuleLog — budget 8 MiB, n ≤ 1000).
// Anti-boucle : ce handler ne logue qu'en DEBUG (chaque tail écrirait sinon
// dans http.log qu'on est précisément en train de lire).
//
// Les paramètres n et since sont pris en STRING pour reproduire le contrat
// d'origine (valeur non numérique / date non RFC3339 ignorée silencieusement),
// PAS le 422 de validation Huma qu'un `int`/`time.Time` typé produirait.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
)

// AdminLogsHandler sert le viewer de logs (logsDir résolu au boot via
// logging.LoadConfig — respecte LEVELUP_LOGS_DIR).
type AdminLogsHandler struct {
	logsDir string
}

// NewAdminLogsHandler construit le handler.
func NewAdminLogsHandler(logsDir string) *AdminLogsHandler {
	return &AdminLogsHandler{logsDir: logsDir}
}

// Mount enregistre les 2 routes via Huma sur le sous-routeur chi /admin
// (middleware RequireAuth/RequireAdmin/NoStore hérités).
func (h *AdminLogsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/monitoring/logs/modules", h.handleGetModules)
	huma.Get(api, "/monitoring/logs/tail", h.handleGetTail)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// logsTailInput : query params du tail filtré. n, since et before en STRING pour
// préserver le parsing lenient d'origine (valeur invalide ignorée). before =
// curseur arrière « charger plus » (offset octet renvoyé en next_offset).
type logsTailInput struct {
	Module   string `query:"module"`
	Level    string `query:"level"`
	Contains string `query:"contains"`
	N        string `query:"n"`
	Since    string `query:"since"`
	Before   string `query:"before"`
}

type logsModulesOutput struct{ Body domain.AdminLogModules }
type logsTailOutput struct{ Body domain.AdminLogTail }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetModules liste les modules de logs disponibles.
// GET /admin/monitoring/logs/modules.
func (h *AdminLogsHandler) handleGetModules(ctx context.Context, _ *struct{}) (*logsModulesOutput, error) {
	mods, err := ops.ListLogModules(h.logsDir)
	if err != nil {
		slog.DebugContext(ctx, "admin_logs: list modules failed", "err", err)
		return nil, humacore.NewError(http.StatusServiceUnavailable, "logs_unavailable",
			"Dossier de logs illisible.")
	}
	resp := domain.AdminLogModules{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Modules:     []domain.AdminLogModule{},
	}
	for _, m := range mods {
		resp.Modules = append(resp.Modules, domain.AdminLogModule{
			Module:     m.Module,
			SizeBytes:  m.SizeBytes,
			ModifiedAt: m.ModifiedAt.UTC().Format(time.RFC3339),
		})
	}
	return &logsModulesOutput{Body: resp}, nil
}

// handleGetTail retourne les dernières lignes filtrées d'un module.
// GET /admin/monitoring/logs/tail?module=sync&n=200&level=warn&contains=x&since=RFC3339.
func (h *AdminLogsHandler) handleGetTail(ctx context.Context, in *logsTailInput) (*logsTailOutput, error) {
	module := in.Module
	if module == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_module", "module requis.")
	}
	opts := ops.LogTailOptions{
		Level:    in.Level,
		Contains: in.Contains,
	}
	if in.N != "" {
		if n, err := strconv.Atoi(in.N); err == nil {
			opts.N = n
		}
	}
	if in.Since != "" {
		if t, err := time.Parse(time.RFC3339, in.Since); err == nil {
			opts.Since = t
		}
	}
	if in.Before != "" {
		if b, err := strconv.ParseInt(in.Before, 10, 64); err == nil && b > 0 {
			opts.BeforeOffset = b
		}
	}

	res, err := ops.TailModuleLog(h.logsDir, module, opts)
	if err != nil {
		slog.DebugContext(ctx, "admin_logs: tail failed", "module", module, "err", err)
		return nil, humacore.NewError(http.StatusBadRequest, "logs_tail_failed",
			"Lecture du module impossible (module inconnu ou dossier illisible).")
	}

	resp := domain.AdminLogTail{
		Module:       module,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Entries:      make([]domain.AdminLogEntry, 0, len(res.Entries)),
		ScannedBytes: res.ScannedBytes,
		Truncated:    res.Truncated,
		NextOffset:   res.NextOffset,
		HasMore:      res.HasMore,
	}
	for _, e := range res.Entries {
		entry := domain.AdminLogEntry{
			Level:     e.Level,
			Msg:       e.Msg,
			Module:    e.Module,
			RequestID: e.RequestID,
			EventID:   e.EventID,
			Err:       e.Err,
			Source:    e.Source,
			Fields:    e.Fields,
			Raw:       e.Raw,
		}
		if !e.Time.IsZero() {
			entry.Time = e.Time.UTC().Format(time.RFC3339Nano)
		}
		resp.Entries = append(resp.Entries, entry)
	}
	return &logsTailOutput{Body: resp}, nil
}
