// Package handlers — tactical.go : les deux endpoints de l'onglet Tactique.
//
//	GET /players/{player_slug}/tactical/maps
//	GET /players/{player_slug}/tactical/{map_id}/raster?question=&qui=&<filtre>
//
// Montes sur le sous-routeur /players/{player_slug} — ownership (ADR 0029) et
// titre herites du groupe, comme /replay et /matches/{id}/events.
//
// ZERO LOGIQUE ICI. Le handler decode, delegue, traduit un refus en statut. Le
// vocabulaire de filtre est celui de l'Explorateur (playlist, mode, from, to,
// session, outcome, with_player) et sa validation reutilise les predicats deja
// poses par match_view.go — meme paquet, aucune regex recopiee.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TacticalHandler sert l'onglet Tactique via un TacticalService resolu par joueur.
type TacticalHandler struct {
	newSvc ServiceFactory[port.TacticalService]
}

// NewTacticalHandler construit le handler avec sa factory de service.
func NewTacticalHandler(newSvc ServiceFactory[port.TacticalService]) *TacticalHandler {
	return &TacticalHandler{newSvc: newSvc}
}

// Mount enregistre les deux routes via Huma sur le sous-routeur chi.
func (h *TacticalHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/tactical/maps", h.handleGetMaps,
		humacore.Op("getTacticalMaps", "Cartes jouees, pour la grille d'entree de l'onglet Tactique", "tactical"))
	huma.Get(api, "/tactical/{map_id}/raster", h.handleGetRaster,
		humacore.Op("getTacticalRaster", "Lecture de placement d'une carte (ou je meurs, ou je tue, ou je gagne)", "tactical"))
}

// TacticalFilterQuery porte le vocabulaire de filtre de l'Explorateur, et lui
// seul. Struct EMBARQUEE dans les deux entrees : les deux ecrans filtrent la meme
// chose, et deux declarations divergeraient au premier axe ajoute.
//
// EXPORTEE, et c'est OBLIGATOIRE : Huma lie les parametres par reflexion, et un
// champ embarque de type NON exporte n'est pas assignable — les filtres arrivaient
// alors tous vides, sans erreur (piege verifie par
// TestTacticalHandler_FiltreExplorateur).
type TacticalFilterQuery struct {
	Playlist   string `query:"playlist" doc:"Playlists, separees par une virgule."`
	Mode       string `query:"mode" doc:"Categories de mode, separees par une virgule."`
	From       string `query:"from" doc:"Borne basse (RFC3339)."`
	To         string `query:"to" doc:"Borne haute (RFC3339)."`
	Session    string `query:"session" doc:"Identifiant de session."`
	Outcome    string `query:"outcome" doc:"Issue : win | loss | draw | dnf."`
	WithPlayer string `query:"with_player" doc:"XUID (entier decimal) devant avoir participe au match."`
}

type tacticalMapsInput struct {
	PlayerSlug string `path:"player_slug"`
	TacticalFilterQuery
}

type tacticalMapsOutput struct{ Body domain.TacticalMapsPage }

type tacticalRasterInput struct {
	PlayerSlug string `path:"player_slug"`
	MapID      string `path:"map_id"`
	Question   string `query:"question" doc:"Lecture : morts | kills | gagne. Defaut : morts."`
	Qui        string `query:"qui" doc:"Axe : moi | escouade | adv. Defaut : moi."`
	TacticalFilterQuery
}

type tacticalRasterOutput struct{ Body domain.TacticalRaster }

// handleGetMaps retourne les cartes jouees par le joueur sous le filtre.
func (h *TacticalHandler) handleGetMaps(ctx context.Context, in *tacticalMapsInput) (*tacticalMapsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	page, err := svc.MapsPlayed(ctx, in.spec(ctx))
	if err != nil {
		return nil, mapTacticalError(ctx, err, "tactical.maps")
	}
	return &tacticalMapsOutput{Body: page}, nil
}

// handleGetRaster retourne la lecture de placement d'une carte.
func (h *TacticalHandler) handleGetRaster(ctx context.Context, in *tacticalRasterInput) (*tacticalRasterOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	mapID := strings.TrimSpace(in.MapID)
	if mapID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_map_id", "map_id est requis")
	}
	raster, err := svc.Raster(ctx, mapID,
		defautSiVide(in.Question, domain.TacticalQuestionMorts),
		defautSiVide(in.Qui, domain.TacticalQuiMoi),
		in.spec(ctx))
	if err != nil {
		return nil, mapTacticalError(ctx, err, "tactical.raster")
	}
	return &tacticalRasterOutput{Body: raster}, nil
}

// defautSiVide applique le defaut d'un parametre absent. Un parametre PRESENT mais
// hors vocabulaire n'est PAS remplace par le defaut : il est refuse par le service
// (400) — servir « morts » a qui a demande « temps » repondrait a une autre
// question sans le dire.
func defautSiVide(v, defaut string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return defaut
}

// mapTacticalError traduit les refus du service en statut HTTP.
//
// L'ordre compte : les refus TYPES d'abord (ils sont plus precis), la capability
// ensuite par le helper central MapCapabilityError (source unique, garde-rail
// no_capability_error_dup_test), le 500 en dernier recours.
func mapTacticalError(ctx context.Context, err error, probe string) error {
	switch {
	case errors.Is(err, domain.ErrTacticalCarteInconnue):
		return humacore.NewError(http.StatusNotFound, "tactical_map_unknown", err.Error())
	case errors.Is(err, domain.ErrTacticalQuestionInconnue):
		return humacore.NewError(http.StatusBadRequest, "tactical_question_unknown", err.Error())
	case errors.Is(err, domain.ErrTacticalQuiInconnu):
		return humacore.NewError(http.StatusBadRequest, "tactical_axis_unknown", err.Error())
	}
	if mapped, ok := MapCapabilityError(ctx, err, probe); ok {
		return mapped
	}
	slog.ErrorContext(ctx, "tactique: lecture en echec", "probe", probe, "err", err)
	return humacore.NewError(http.StatusInternalServerError, "tactical_error", err.Error())
}

// spec assemble le MatchFilterSpec de l'Explorateur.
//
// Un axe invalide est IGNORE avec un log (jamais un 400) — meme politique que
// parseNeighborsFilterSpec, dont ce code reutilise les predicats
// (playlistOrSessionPattern, xuidPattern, parseCsvFilterParam). Un filtre
// d'affichage qui ferait echouer la page pour une valeur mal formee rendrait un
// lien partage inutilisable.
func (f TacticalFilterQuery) spec(ctx context.Context) *domain.MatchFilterSpec {
	spec := &domain.MatchFilterSpec{
		PlaylistNames:  parseCsvFilterParam(ctx, f.Playlist, "playlist"),
		ModeCategories: parseCsvFilterParam(ctx, f.Mode, "mode"),
	}
	if v := strings.TrimSpace(f.Session); v != "" {
		affecterSiValide(ctx, "session", v, playlistOrSessionPattern.MatchString, &spec.SessionID)
	}
	if v := strings.TrimSpace(f.Outcome); v != "" {
		affecterSiValide(ctx, "outcome", v, analysis.IsValidOutcomeLabel, &spec.Outcome)
	}
	if v := strings.TrimSpace(f.WithPlayer); v != "" {
		affecterSiValide(ctx, "with_player", v, xuidPattern.MatchString, &spec.WithPlayerXuid)
	}
	spec.DateFrom = parseBorneDate(ctx, f.From, "from")
	spec.DateTo = parseBorneDate(ctx, f.To, "to")
	if spec.IsEmpty() {
		return nil
	}
	return spec
}

// affecterSiValide pose la valeur si elle passe le predicat, journalise sinon.
func affecterSiValide(ctx context.Context, param, valeur string, valide func(string) bool, cible **string) {
	if !valide(valeur) {
		slog.WarnContext(ctx, "tactique: filtre invalide ignore", "param", param, "value", valeur)
		return
	}
	v := valeur
	*cible = &v
}

// parseBorneDate lit une borne RFC3339 ; nil (avec log) si elle ne se lit pas.
func parseBorneDate(ctx context.Context, raw, param string) *time.Time {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		slog.WarnContext(ctx, "tactique: filtre invalide ignore",
			"param", param, "value", v, "err", err)
		return nil
	}
	return &t
}
