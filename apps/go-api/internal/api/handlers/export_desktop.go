// Package handlers — export_desktop.go : l'export des données d'un joueur pour l'application de bureau (lot
// L22.b, ADR 0045), côté instance hébergée (capacité ExportForDesktop).
//
//   - POST /api/v1/export/desktop/token : le joueur CONNECTÉ obtient un jeton court (15 minutes) à usage unique ;
//   - GET /api/v1/export/desktop : l'archive du joueur, sur présentation du jeton (en-tête Authorization: Bearer,
//     depuis l'application de bureau, qui n'a pas de session) ou sur la session du joueur connecté.
//
// Le joueur exporté est TOUJOURS celui que le jeton ou la session désigne (ownership, ADR 0029) : aucune
// requête ne nomme un joueur, et sa base joueur se trouve par son XUID (le profil suivi qui le porte, ADR 0035),
// jamais par le gamertag d'une session. Jamais un jeton d'authentification Xbox, jamais un compte, jamais la donnée
// propre d'un autre profil (périmètre fermé d'internal/transfer). Un export RÉUSSI par heure et par joueur (un
// échec ne consomme pas le quota), un seul export à la fois par joueur, journalisé.
//
// L'export par SESSION exige l'en-tête X-Requested-With : une requête GET inter-sites (lien, image) ne peut pas le
// poser, elle ne déclenche donc pas un export à l'insu du joueur.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/transfer"
)

// ExportDesktopHandler sert les deux routes d'export.
type ExportDesktopHandler struct {
	cfg   *config.AppConfig
	store *transfer.TokenStore
	now   func() time.Time
}

// NewExportDesktopHandler crée le handler ; store tient les jetons d'export de l'instance.
func NewExportDesktopHandler(cfg *config.AppConfig, store *transfer.TokenStore) *ExportDesktopHandler {
	return &ExportDesktopHandler{cfg: cfg, store: store, now: time.Now}
}

// MountToken enregistre POST /export/desktop/token (Huma) ; r porte déjà RequireAuth.
func (h *ExportDesktopHandler) MountToken(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/export/desktop/token", h.handleToken,
		humacore.Op("postExportDesktopToken", "Get a one-shot export token for the desktop application", "export"))
}

// MountExport enregistre GET /export/desktop (réponse binaire). r ne porte PAS RequireAuth : la route
// s'authentifie elle-même, par jeton ou par session.
func (h *ExportDesktopHandler) MountExport(r chi.Router) {
	r.With(middleware.NoStore).Get("/export/desktop", h.Export)
}

type exportTokenOutput struct {
	Body *domain.ExportToken
}

// handleToken délivre un jeton d'export au joueur connecté.
func (h *ExportDesktopHandler) handleToken(ctx context.Context, _ *struct{}) (*exportTokenOutput, error) {
	sess := middleware.GetSession(ctx)
	if sess == nil || sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.XUID == "" {
		return nil, humacore.NewError(http.StatusUnauthorized, "halo_auth_required", "halo sign-in required")
	}
	g := transfer.Grant{Xuid: sess.LinkedHaloIdentity.XUID, Gamertag: sess.LinkedHaloIdentity.Gamertag}
	token, expires, err := h.store.Issue(g, h.now())
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, "export_token_unavailable", err)
	}
	slog.InfoContext(ctx, "export: one-shot export token issued", "xuid", g.Xuid, "expire", expires)
	return &exportTokenOutput{Body: &domain.ExportToken{Token: token, ExpiresAt: expires}}, nil
}

// Export écrit l'archive du joueur désigné par le jeton (ou la session) puis la sert.
// GET /export/desktop
func (h *ExportDesktopHandler) Export(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	grant, status, code, err := h.authenticate(r)
	if err != nil {
		writeErrorFrom(ctx, w, status, code, err)
		return
	}
	// Le quota d'un export par heure ne se consomme qu'APRÈS un export réussi ; l'export en cours se libère toujours.
	exported := false
	defer func() { h.store.Finish(grant.Xuid, exported, h.now()) }()
	dir := filepath.Join(h.cfg.PathResolver().CacheRootDir(), "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErrorFrom(ctx, w, http.StatusInternalServerError, "export_failed", err)
		return
	}
	f, err := os.CreateTemp(dir, "export-*.zip")
	if err != nil {
		writeErrorFrom(ctx, w, http.StatusInternalServerError, "export_failed", err)
		return
	}
	dest := f.Name()
	_ = f.Close()
	defer func() {
		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "export: temporary archive not removed", "path", dest, "err", err)
		}
	}()
	m, err := transfer.Export(ctx, transfer.ExportInput{Paths: h.cfg.PathResolver(), Slug: title.DefaultSlug,
		Xuid: grant.Xuid, Gamertag: grant.Gamertag, PlayerGamertag: h.trackedGamertag(ctx, grant.Xuid),
		AppVersion: h.cfg.AppVersion, Dest: dest})
	if err != nil {
		writeErrorFrom(ctx, w, http.StatusInternalServerError, "export_failed", err)
		return
	}
	exported = true
	out, err := os.Open(dest) //nolint:gosec // fichier créé ci-dessus
	if err != nil {
		writeErrorFrom(ctx, w, http.StatusInternalServerError, "export_failed", err)
		return
	}
	defer func() { _ = out.Close() }()
	slog.InfoContext(ctx, "export: archive servie", "xuid", grant.Xuid, "tables", len(m.Tables))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="levelup-export.zip"`)
	http.ServeContent(w, r, "levelup-export.zip", m.CreatedAt, out)
}

// authenticate rend le joueur de la requête : par jeton d'export (Bearer), sinon par la session connectée. Le
// refus porte son statut et son code fixe.
func (h *ExportDesktopHandler) authenticate(r *http.Request) (transfer.Grant, int, string, error) {
	if raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		g, err := h.store.Redeem(strings.TrimSpace(raw), h.now())
		switch err {
		case nil:
			return g, 0, "", nil
		case transfer.ErrRateLimited:
			return transfer.Grant{}, http.StatusTooManyRequests, "export_rate_limited", err
		default:
			return transfer.Grant{}, http.StatusUnauthorized, "export_token_invalid", err
		}
	}
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.XUID == "" {
		return transfer.Grant{}, http.StatusUnauthorized, "auth_required", transfer.ErrTokenInvalid
	}
	if r.Header.Get(exportSessionHeader) == "" {
		return transfer.Grant{}, http.StatusForbidden, "export_header_required", transfer.ErrTokenInvalid
	}
	g := transfer.Grant{Xuid: sess.LinkedHaloIdentity.XUID, Gamertag: sess.LinkedHaloIdentity.Gamertag}
	if !h.store.AllowSession(g.Xuid, h.now()) {
		return transfer.Grant{}, http.StatusTooManyRequests, "export_rate_limited", transfer.ErrRateLimited
	}
	return g, 0, "", nil
}

// exportSessionHeader : l'en-tête qu'une requête d'export par SESSION doit porter (valeur libre, non vide). Une
// requête inter-sites simple ne peut pas le poser : il écarte l'export déclenché par un lien tiers.
const exportSessionHeader = "X-Requested-With"

// trackedGamertag rend le gamertag du profil suivi qui porte CE xuid (la base joueur exportée est la sienne), ou
// "" quand aucun profil suivi ne le porte : le gamertag d'une session n'est jamais lu pour choisir une base.
func (h *ExportDesktopHandler) trackedGamertag(ctx context.Context, xuid string) string {
	players, err := h.cfg.LoadPlayers(title.DefaultSlug)
	if err != nil {
		slog.WarnContext(ctx, "export: profiles unreadable, player database not exported", "xuid", xuid, "err", err)
		return ""
	}
	for _, p := range players {
		if p.XUID == xuid {
			return p.Gamertag
		}
	}
	return ""
}
