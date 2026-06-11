// Package handlers — admin_logs.go : viewer de logs du dashboard monitoring.
//
// Routes (RequireAuth+RequireAdmin+NoStore) :
//   - GET /admin/monitoring/logs/modules : fichiers logs/{module}.log dispo
//   - GET /admin/monitoring/logs/tail?module=&n=&level=&contains=&since=
//
// Lecture par la fin chunkée (ops.TailModuleLog — budget 8 MiB, n ≤ 1000).
// Anti-boucle : ce handler ne logue qu'en DEBUG (chaque tail écrirait sinon
// dans http.log qu'on est précisément en train de lire).
package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

// GetModules liste les modules de logs disponibles.
// GET /admin/monitoring/logs/modules.
func (h *AdminLogsHandler) GetModules(w http.ResponseWriter, r *http.Request) {
	mods, err := ops.ListLogModules(h.logsDir)
	if err != nil {
		slog.DebugContext(r.Context(), "admin_logs: list modules failed", "err", err)
		writeError(r.Context(), w, http.StatusServiceUnavailable, "logs_unavailable",
			"Dossier de logs illisible.")
		return
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
	writeJSON(w, http.StatusOK, resp)
}

// GetTail retourne les dernières lignes filtrées d'un module.
// GET /admin/monitoring/logs/tail?module=sync&n=200&level=warn&contains=x&since=RFC3339.
func (h *AdminLogsHandler) GetTail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	module := q.Get("module")
	if module == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_module", "module requis.")
		return
	}
	opts := ops.LogTailOptions{
		Level:    q.Get("level"),
		Contains: q.Get("contains"),
	}
	if raw := q.Get("n"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			opts.N = n
		}
	}
	if raw := q.Get("since"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			opts.Since = t
		}
	}

	res, err := ops.TailModuleLog(h.logsDir, module, opts)
	if err != nil {
		slog.DebugContext(r.Context(), "admin_logs: tail failed", "module", module, "err", err)
		writeError(r.Context(), w, http.StatusBadRequest, "logs_tail_failed",
			"Lecture du module impossible (module inconnu ou dossier illisible).")
		return
	}

	resp := domain.AdminLogTail{
		Module:       module,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Entries:      make([]domain.AdminLogEntry, 0, len(res.Entries)),
		ScannedBytes: res.ScannedBytes,
		Truncated:    res.Truncated,
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
	writeJSON(w, http.StatusOK, resp)
}
