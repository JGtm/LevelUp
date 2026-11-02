// Package handlers — desktop_strings.go : GET /api/v1/desktop/strings, les libellés de la coquille
// desktop (ADR 0045 D-7 et D-13, A4).
//
// La coquille ne porte aucun libellé de langue : ceux de l'écran de démarrage, des échecs, de
// l'arrêt et de la mise à jour viennent du catalogue des messages du serveur
// (internal/i18n/messages/desktop.toml), rendus dans la langue de la requête (en-tête
// X-LevelUp-Locale). La route n'est montée que sous l'entrée de la coquille et exige son cookie.
package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/i18n"
)

// DesktopStringsHandler sert les libellés de la coquille.
type DesktopStringsHandler struct{}

// NewDesktopStringsHandler crée le handler (sans état : le catalogue est embarqué).
func NewDesktopStringsHandler() *DesktopStringsHandler { return &DesktopStringsHandler{} }

// Mount enregistre la route au point de montage de l'appelant.
func (h *DesktopStringsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/desktop/strings", h.handle,
		humacore.Op("getDesktopStrings", "Labels of the desktop shell in the request language", "desktop"))
}

type desktopStringsOutput struct {
	Body struct {
		Locale  string            `json:"locale" doc:"Language the labels are written in."`
		Strings map[string]string `json:"strings" doc:"Label by identifier (starting, failed_*, stopping, stopped, update_available); update_available keeps the {version} placeholder."`
	}
}

func (h *DesktopStringsHandler) handle(ctx context.Context, _ *struct{}) (*desktopStringsOutput, error) {
	loc := i18n.FromContext(ctx)
	out := &desktopStringsOutput{}
	out.Body.Locale = string(loc)
	out.Body.Strings = map[string]string{
		"starting":                i18n.Msg(loc, "desktop.starting", nil),
		"failed_locked":           i18n.Msg(loc, "desktop.failed.locked", nil),
		"failed_migration_failed": i18n.Msg(loc, "desktop.failed.migration_failed", nil),
		"failed_schema_newer":     i18n.Msg(loc, "desktop.failed.schema_newer", nil),
		"failed_port":             i18n.Msg(loc, "desktop.failed.port", nil),
		"stopping":                i18n.Msg(loc, "desktop.stopping", nil),
		"stopped":                 i18n.Msg(loc, "desktop.stopped", nil),
		// Le gabarit voyage tel quel : la coquille connaît le numéro de version, pas le serveur.
		"update_available": i18n.Msg(loc, "desktop.update_available", i18n.Params{"version": "{version}"}),
	}
	return out, nil
}
