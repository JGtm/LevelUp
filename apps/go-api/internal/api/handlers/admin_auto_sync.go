// Package handlers — admin_auto_sync.go : endpoints diagnostic du scheduler
// d'auto-sync. Permet de voir le résultat détaillé par joueur du dernier cycle
// (raison du skip/failure, compteurs, erreurs) sans avoir accès aux logs
// serveur, et de forcer un cycle on-demand pour reproduire un bug.
//
// Routes (montées sous /api/v1/admin/auto-sync/, protégées par RequireAdmin) :
//   - GET  /snapshot : retourne le dernier état mémorisé du scheduler
//   - POST /run      : force un cycle synchrone et retourne le snapshot mis à jour
//
// Le POST est synchrone et peut prendre plusieurs minutes (1 cycle = N joueurs
// × appel API Halo + DB writes). Le client doit augmenter son timeout en
// conséquence (`curl -m 600` recommandé).
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// AdminAutoSyncHandler expose les endpoints diagnostic de l'auto-sync.
type AdminAutoSyncHandler struct {
	scheduler *scheduler.AutoSyncScheduler
	cfg       *config.AppConfig
}

// NewAdminAutoSyncHandler crée un handler. scheduler doit être non nil
// (le caller dans server.go garde le wiring conditionnel).
// cfg permet à ProbeTokens de localiser la player DB pour lire sync_meta.
func NewAdminAutoSyncHandler(s *scheduler.AutoSyncScheduler, cfg *config.AppConfig) *AdminAutoSyncHandler {
	return &AdminAutoSyncHandler{scheduler: s, cfg: cfg}
}

// GetSnapshot retourne le snapshot mémorisé du dernier cycle.
// GET /api/v1/admin/auto-sync/snapshot
func (h *AdminAutoSyncHandler) GetSnapshot(w http.ResponseWriter, _ *http.Request) {
	snap := h.scheduler.Snapshot()
	writeJSON(w, http.StatusOK, snap)
}

// RunOnce force un cycle synchrone et retourne le snapshot mis à jour.
// POST /api/v1/admin/auto-sync/run
//
// Bloquant : peut prendre plusieurs minutes (N joueurs * appel API Halo +
// DB writes). Le scheduler est thread-safe, ce force-run n'interfère pas
// avec les cycles automatiques (ils ne tournent pas en parallèle car
// `Run()` séquence ses ticks via un ticker).
func (h *AdminAutoSyncHandler) RunOnce(w http.ResponseWriter, r *http.Request) {
	_ = h.scheduler.RunOnce(r.Context())
	snap := h.scheduler.Snapshot()
	writeJSON(w, http.StatusOK, snap)
}

// TokenProbeResult décrit l'état des sources de refresh_token pour un joueur.
type TokenProbeResult struct {
	Gamertag            string `json:"gamertag"`
	EnvVarKey           string `json:"env_var_key"`
	EnvVarPresent       bool   `json:"env_var_present"`
	EnvVarLen           int    `json:"env_var_len,omitempty"`
	EnvVarSHA256        string `json:"env_var_sha256,omitempty"` // identifie la valeur sans la révéler
	EnvVarHead          string `json:"env_var_head,omitempty"`   // 6 premiers chars (debug visuel)
	EnvVarTail          string `json:"env_var_tail,omitempty"`   // 6 derniers chars (debug visuel)
	MSALCachePresent    bool   `json:"msal_cache_present"`
	MSALCacheLen        int    `json:"msal_cache_len,omitempty"`
	DBRefreshPresent    bool   `json:"db_refresh_token_present"`
	DBRefreshLen        int    `json:"db_refresh_token_len,omitempty"`
	DBRefreshSHA256     string `json:"db_refresh_token_sha256,omitempty"`
	EnvVarExchangeOK    bool   `json:"env_var_exchange_ok"`
	EnvVarExchangeError string `json:"env_var_exchange_error,omitempty"`
	DBExchangeOK        bool   `json:"db_exchange_ok"`
	DBExchangeError     string `json:"db_exchange_error,omitempty"`
	DBPath              string `json:"db_path"`
}

// fingerprintToken retourne sha256 + head/tail tronqués pour identifier un
// token sans révéler sa valeur (utile pour comparer plusieurs lectures).
func fingerprintToken(s string) (sha string, head string, tail string) {
	if s == "" {
		return "", "", ""
	}
	sum := sha256.Sum256([]byte(s))
	sha = hex.EncodeToString(sum[:8]) // 16 hex chars suffisent pour comparer
	if len(s) >= 6 {
		head = s[:6]
	} else {
		head = s
	}
	if len(s) >= 6 {
		tail = s[len(s)-6:]
	} else {
		tail = s
	}
	return
}

// ProbeTokens diagnostic complet pour un joueur :
//   - quelle source de refresh_token est présente (env var, sync_meta) ?
//   - chaque source produit-elle un access_token quand échangée chez Microsoft ?
//
// GET /api/v1/_diag/auto-sync/probe?gamertag=JGtm
func (h *AdminAutoSyncHandler) ProbeTokens(w http.ResponseWriter, r *http.Request) {
	gamertag := r.URL.Query().Get("gamertag")
	if gamertag == "" {
		writeError(w, http.StatusBadRequest, "missing_gamertag", "query param gamertag requis")
		return
	}

	res := TokenProbeResult{Gamertag: gamertag}

	// Clé env var (même normalisation que defaultTokenReader).
	key := strings.ToUpper(gamertag)
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	res.EnvVarKey = "SPNKR_OAUTH_REFRESH_TOKEN_" + key

	envRT := os.Getenv(res.EnvVarKey)
	res.EnvVarPresent = envRT != ""
	res.EnvVarLen = len(envRT)
	res.EnvVarSHA256, res.EnvVarHead, res.EnvVarTail = fingerprintToken(envRT)

	// Lire la player DB pour msal_cache + db_refresh_token.
	dbPath := titlePkg.NewPathResolver(h.cfg.RepoRoot).PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	res.DBPath = dbPath
	db, err := duckdb.OpenReadWriteShared(dbPath)
	var msalCache, dbRT string
	if err == nil {
		defer db.Close() //nolint:errcheck // ref-count
		_ = db.SQLDb().QueryRowContext(r.Context(),
			"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&msalCache)
		_ = db.SQLDb().QueryRowContext(r.Context(),
			"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&dbRT)
	}
	res.MSALCachePresent = msalCache != ""
	res.MSALCacheLen = len(msalCache)
	res.DBRefreshPresent = dbRT != ""
	res.DBRefreshLen = len(dbRT)
	res.DBRefreshSHA256, _, _ = fingerprintToken(dbRT)

	// Tenter l'échange OAuth pour chaque source disponible.
	// IMPORTANT : Microsoft rotate le refresh_token à chaque exchange réussi.
	// Pour ne pas brûler le RT du caller, on persiste le RT rotaté retourné dans
	// sync_meta.oauth_refresh_token (et donc le scheduler pourra l'utiliser au
	// tick suivant).
	ctx := r.Context()
	if envRT != "" {
		accessToken, rotated, ferr := auth.ExchangeRefreshTokenWithRotation(ctx, envRT)
		if ferr != nil {
			res.EnvVarExchangeError = ferr.Error()
		} else {
			res.EnvVarExchangeOK = accessToken != ""
			if !res.EnvVarExchangeOK {
				res.EnvVarExchangeError = "Microsoft a retourné access_token vide"
			}
			if res.EnvVarExchangeOK && db != nil {
				toPersist := rotated
				if toPersist == "" {
					toPersist = envRT
				}
				_ = duckdb.WriteOAuthRefreshToken(ctx, db, toPersist)
			}
		}
	}
	if dbRT != "" {
		accessToken, rotated, ferr := auth.ExchangeRefreshTokenWithRotation(ctx, dbRT)
		if ferr != nil {
			res.DBExchangeError = ferr.Error()
		} else {
			res.DBExchangeOK = accessToken != ""
			if !res.DBExchangeOK {
				res.DBExchangeError = "Microsoft a retourné access_token vide"
			}
			if res.DBExchangeOK && db != nil && rotated != "" {
				_ = duckdb.WriteOAuthRefreshToken(ctx, db, rotated)
			}
		}
	}

	writeJSON(w, http.StatusOK, res)
}
