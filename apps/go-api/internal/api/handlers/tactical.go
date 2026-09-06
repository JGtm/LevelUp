// Package handlers — tactical.go : les endpoints de l'onglet Tactique.
//
//	GET /players/{player_slug}/tactical/maps
//	GET /players/{player_slug}/tactical/{map_id}/raster?question=&qui=&<filtre>
//	GET /players/{player_slug}/tactical/{map_id}/background      (calage du fond)
//	GET /players/{player_slug}/tactical/{map_id}/background.png  (image du fond)
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
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/port"
)

// TacticalHandler sert l'onglet Tactique.
//
// DEUX SERVICES, ET C'EST VOULU. Les lectures tactiques viennent du TacticalService ; le
// FOND DE CARTE vient du ReplayService, qui possede deja la resolution carte -> image (une
// seule cascade dans le depot, cf. service/replay_map_background.go). Faire transiter le
// fond par le TacticalService aurait ete un service qui en appelle un autre — l'anti-pattern
// de couplage horizontal de `arch-rules`.
type TacticalHandler struct {
	newSvc    ServiceFactory[port.TacticalService]
	newReplay ServiceFactory[port.ReplayService]
}

// NewTacticalHandler construit le handler avec ses deux factories de service.
func NewTacticalHandler(
	newSvc ServiceFactory[port.TacticalService],
	newReplay ServiceFactory[port.ReplayService],
) *TacticalHandler {
	return &TacticalHandler{newSvc: newSvc, newReplay: newReplay}
}

// Mount enregistre les routes de l'onglet sur le sous-routeur chi.
//
// LE FOND DE CARTE N'EST PAS SOUS LE GARDE LOCAL DU REJEU (`LocalOnlyReplay`), et c'est
// deliberé : ce garde protege les TRAJECTOIRES decodees du film, dont la couverture n'est
// pas encore productionnalisable (cf. replay_local_gate.go). Une image de fond est une
// donnee de REFERENCE versionnee, extraite des fichiers de carte et non d'un film : la
// masquer hors localhost rendrait la grille des cartes vide en production sans rien
// proteger.
func (h *TacticalHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/tactical/maps", h.handleGetMaps,
		humacore.Op("getTacticalMaps", "Cartes jouees, pour la grille d'entree de l'onglet Tactique", "tactical"))
	huma.Get(api, "/tactical/{map_id}/raster", h.handleGetRaster,
		humacore.Op("getTacticalRaster", "Lecture de placement d'une carte (ou je meurs, ou je tue, ou je gagne)", "tactical"))
	huma.Get(api, "/tactical/{map_id}/background", h.handleGetMapBackground,
		humacore.Op("getTacticalMapBackground", "Calage du fond d'une carte de l'onglet Tactique", "tactical"))
	// Route chi nue : la charge utile est binaire, comme le fond du rejeu par match.
	r.Get("/tactical/{map_id}/background.png", h.handleGetMapBackgroundImage)
}

// TacticalFilterQuery porte le vocabulaire de filtre de l'Explorateur, et lui
// seul. Struct EMBARQUEE dans les deux entrees : les deux ecrans filtrent la meme
// chose, et deux declarations divergeraient au premier axe ajoute.
//
// PAS DE `session` (retrait du 2026-09-06, T11) : `MatchFilterSpec` porte bien un
// `SessionID`, mais `analysis.BuildNeighborsWhereClause` le range dans
// `IgnoredFilters` sans jamais l'appliquer — les sessions vivent dans la base
// JOUEUR (`player_match_enrichment`), que ces requetes shared ne joignent pas. Un
// parametre accepte, documente au contrat, et silencieusement sans effet est pire
// qu'un parametre absent : l'appelant croit avoir filtre. On n'accepte pas ce
// qu'on n'honore pas.
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

type tacticalMapInput struct {
	PlayerSlug string `path:"player_slug"`
	MapID      string `path:"map_id"`
}

type tacticalMapBackgroundOutput struct{ Body replaydoc.MapBackground }

// handleGetMapBackground retourne le CALAGE du fond d'une carte : metres par pixel, origine
// monde, taille de l'image. 404 quand la carte n'a pas de fond fige — absence NORMALE
// (toutes les cartes n'en ont pas), que le client traduit par une vignette sans image.
func (h *TacticalHandler) handleGetMapBackground(
	ctx context.Context, in *tacticalMapInput,
) (*tacticalMapBackgroundOutput, error) {
	svc, mapID, err := h.replayPourCarte(ctx, in.PlayerSlug, in.MapID)
	if err != nil {
		return nil, err
	}
	bg, err := svc.MapBackgroundForMap(ctx, mapID)
	if errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		return nil, humacore.NewError(http.StatusNotFound, "map_background_not_available",
			"aucun fond de carte pour cette carte")
	}
	if err != nil {
		return nil, mapTacticalError(ctx, err, "tactical.map_background")
	}
	return &tacticalMapBackgroundOutput{Body: *bg}, nil
}

// handleGetMapBackgroundImage sert le PNG du fond d'une carte, avec le meme cache que le
// fond par match : donnee de REFERENCE versionnee, qui ne change qu'a une re-cuisson.
// `private` parce que la route est derriere l'ownership joueur.
func (h *TacticalHandler) handleGetMapBackgroundImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := chi.URLParam(r, "player_slug")
	mapID := strings.TrimSpace(chi.URLParam(r, "map_id"))
	if mapID == "" {
		writeError(ctx, w, http.StatusBadRequest, "missing_map_id", "map_id est requis")
		return
	}
	svc, err := h.newReplay(ctx, slug)
	if err != nil {
		writeError(ctx, w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	blob, err := svc.MapBackgroundImageForMap(ctx, mapID)
	if errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		writeError(ctx, w, http.StatusNotFound, "map_background_not_available",
			"aucun fond de carte pour cette carte")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "tactique: image de fond en echec", "err", err, "map_id", mapID)
		writeError(ctx, w, http.StatusInternalServerError, "tactical_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(blob)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if _, err := w.Write(blob); err != nil {
		slog.WarnContext(ctx, "tactique: ecriture de l'image de fond interrompue",
			"err", err, "map_id", mapID, "player", slug)
	}
}

// replayPourCarte resout le service de rejeu du joueur et valide le map_id.
func (h *TacticalHandler) replayPourCarte(
	ctx context.Context, playerSlug, rawMapID string,
) (port.ReplayService, string, error) {
	svc, err := h.newReplay(ctx, playerSlug)
	if err != nil {
		return nil, "", humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	mapID := strings.TrimSpace(rawMapID)
	if mapID == "" {
		return nil, "", humacore.NewError(http.StatusBadRequest, "missing_map_id", "map_id est requis")
	}
	return svc, mapID, nil
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
