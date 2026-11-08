// Package handlers — media_missing.go : les médias dont aucun fichier n'existe plus sur le disque, et
// l'ouverture de l'explorateur sur le fichier d'un média.
//
// Routes, toutes sous /players/{player_slug} (groupe déjà gardé par RequirePlayerOwnership et
// RequireCapability(CapMedia)) :
//
//	GET  /media/missing                     → MediaMissingResponse
//	GET  /media/missing/candidates          → MediaMergeCandidatesResponse
//	POST /media/missing/merge               → MediaMergeResponse
//	POST /media/missing/forget              → MediaForgetResponse
//	POST /media/missing/confirm-bulk        → MediaConfirmBulkResponse
//	POST /media/reveal                      → MediaRevealResponse (404 sans la capacité du déploiement)
//
// Aucune règle d'autorisation ici : le handler traduit la session en identité de demandeur
// (resolveMediaRequester), la règle propriétaire/admin vit dans domain.CanModifyMedia et son
// application dans MediaService. Le middleware d'ownership ne peut pas la porter : sur une écriture il
// refuse déjà le visiteur et le co-membre de groupe (seul le like est une action du visiteur,
// player_visitor_actions.go), mais il laisse passer toute lecture du profil, d'où l'autorisation du
// service sur les deux routes GET.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MediaRescan relance la passe d'indexation des médias de gamertag sous ctx (le titre y est posé) :
// celle de service.NewLiveMediaScanHook.
type MediaRescan func(ctx context.Context, gamertag string)

// WithMediaRescan câble la passe lancée en arrière-plan après la confirmation d'une disparition
// massive. Sans elle, la confirmation est enregistrée et appliquée par la passe suivante, au
// prochain démarrage ou à la prochaine synchronisation.
func (h *MediaHandler) WithMediaRescan(f MediaRescan) *MediaHandler {
	h.rescan = f
	return h
}

// mountMissingRoutes enregistre les routes des médias introuvables sur api.
func (h *MediaHandler) mountMissingRoutes(api huma.API) {
	huma.Get(api, "/media/missing", h.handleGetMediaMissing,
		humacore.Op("getMissingMedia", "Media whose file is missing from disk", "media"))
	huma.Get(api, "/media/missing/candidates", h.handleGetMediaMissingCandidates,
		humacore.Op("getMissingMediaCandidates", "Entries a missing media can be merged into", "media"))
	huma.Post(api, "/media/missing/merge", h.handlePostMediaMissingMerge,
		humacore.Op("postMissingMediaMerge", "Merges a missing media into another entry", "media"))
	huma.Post(api, "/media/missing/forget", h.handlePostMediaMissingForget,
		humacore.Op("postMissingMediaForget", "Forgets a missing media", "media"))
	huma.Post(api, "/media/missing/confirm-bulk", h.handlePostMediaMissingConfirmBulk,
		humacore.Op("postMissingMediaConfirmBulk", "Confirms a bulk disappearance of media", "media"))
	huma.Post(api, "/media/reveal", h.handlePostMediaReveal,
		humacore.Op("postMediaReveal", "Opens the file explorer on a media file", "media"))
}

type mediaMissingListInput struct {
	PlayerSlug string `path:"player_slug"`
}
type mediaMissingListOutput struct{ Body *domain.MediaMissingResponse }

type mediaMissingCandidatesInput struct {
	PlayerSlug string `path:"player_slug"`
	FilePath   string `query:"file_path"`
}
type mediaMissingCandidatesOutput struct {
	Body *domain.MediaMergeCandidatesResponse
}

// mediaMissingBodyInput : {player_slug} + corps REQUIS (RawBody, décodage maison → 400 invalid_body,
// même motif que mediaAssociateInput).
type mediaMissingBodyInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type mediaMissingMergeOutput struct{ Body *domain.MediaMergeResponse }
type mediaMissingForgetOutput struct{ Body *domain.MediaForgetResponse }
type mediaMissingConfirmInput struct {
	PlayerSlug string `path:"player_slug"`
}
type mediaMissingConfirmOutput struct {
	Body *domain.MediaConfirmBulkResponse
}
type mediaRevealOutput struct{ Body *domain.MediaRevealResponse }

// missingService rend le service des médias introuvables du joueur de la route : le service que
// rend la fabrique, vu comme port.MediaMissingService.
func (h *MediaHandler) missingService(ctx context.Context, slug string) (port.MediaMissingService, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusNotFound, "player_not_found", err)
	}
	missing, ok := svc.(port.MediaMissingService)
	if !ok {
		return nil, humacore.NewError(http.StatusNotImplemented, "missing_media_unsupported",
			"missing media are not supported by this service")
	}
	return missing, nil
}

// mediaMissingError traduit une erreur du service des médias introuvables en erreur HTTP.
func mediaMissingError(ctx context.Context, err error) error {
	return mediaWriteError(ctx, err, "media_missing_forbidden", "media_missing_error")
}

// decodeMissingBody décode le corps JSON d'une écriture ; un corps illisible est un 400.
func decodeMissingBody(ctx context.Context, raw []byte, into any) error {
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(into); err != nil {
		return humacore.NewErrorFrom(ctx, http.StatusBadRequest, "invalid_body", err)
	}
	return nil
}

// handleGetMediaMissing liste les médias introuvables du joueur de la route.
func (h *MediaHandler) handleGetMediaMissing(ctx context.Context, in *mediaMissingListInput) (*mediaMissingListOutput, error) {
	svc, err := h.missingService(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	who, err := h.resolveMediaRequester(ctx, "list")
	if err != nil {
		return nil, err
	}
	resp, err := svc.ListMissingMedia(ctx, domain.MediaMissingListRequest{MediaRequester: who})
	if err != nil {
		return nil, mediaMissingError(ctx, err)
	}
	base := mediaCapturesBase(h.settingsStore)
	for i := range resp.Items {
		resp.Items[i].FilePath = h.filePathToURL(in.PlayerSlug, resp.Items[i].FilePath, base)
		resp.Items[i].ThumbnailPath = h.thumbnailURL(in.PlayerSlug, resp.Items[i].ThumbnailPath, base)
	}
	return &mediaMissingListOutput{Body: resp}, nil
}

// handleGetMediaMissingCandidates liste les fiches dans lesquelles fondre la fiche introuvable
// file_path.
func (h *MediaHandler) handleGetMediaMissingCandidates(
	ctx context.Context,
	in *mediaMissingCandidatesInput,
) (*mediaMissingCandidatesOutput, error) {
	if in.FilePath == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_file_path", "file_path query param is required")
	}
	svc, err := h.missingService(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	who, err := h.resolveMediaRequester(ctx, "list")
	if err != nil {
		return nil, err
	}
	resp, err := svc.ListMergeCandidates(ctx, domain.MediaMergeCandidatesRequest{
		MediaRequester: who,
		FilePath:       mediaServableURLToStoredPath(in.PlayerSlug, in.FilePath),
	})
	if err != nil {
		return nil, mediaMissingError(ctx, err)
	}
	base := mediaCapturesBase(h.settingsStore)
	resp.FilePath = in.FilePath
	for i := range resp.Candidates {
		resp.Candidates[i].FilePath = h.filePathToURL(in.PlayerSlug, resp.Candidates[i].FilePath, base)
		resp.Candidates[i].ThumbnailPath = h.thumbnailURL(in.PlayerSlug, resp.Candidates[i].ThumbnailPath, base)
	}
	return &mediaMissingCandidatesOutput{Body: resp}, nil
}

// handlePostMediaMissingMerge fond une fiche introuvable dans une fiche présente. Les deux chemins
// reçus sont des URL servables (celles de la liste et des candidats) : ils sont reversés vers la clé
// de stockage, et la réponse les répète tels que reçus.
func (h *MediaHandler) handlePostMediaMissingMerge(ctx context.Context, in *mediaMissingBodyInput) (*mediaMissingMergeOutput, error) {
	svc, err := h.missingService(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	var req domain.MediaMergeRequest
	if err := decodeMissingBody(ctx, in.RawBody, &req); err != nil {
		return nil, err
	}
	rawPath, rawTarget := req.FilePath, req.TargetFilePath
	req.FilePath = mediaServableURLToStoredPath(in.PlayerSlug, rawPath)
	req.TargetFilePath = mediaServableURLToStoredPath(in.PlayerSlug, rawTarget)
	if req.MediaRequester, err = h.resolveMediaRequester(ctx, "merge"); err != nil {
		return nil, err
	}
	resp, err := svc.MergeMissingMedia(ctx, req)
	if err != nil {
		return nil, mediaMissingError(ctx, err)
	}
	resp.FilePath, resp.TargetFilePath = rawPath, rawTarget
	BumpMediaFeedVersion()
	return &mediaMissingMergeOutput{Body: resp}, nil
}

// handlePostMediaMissingForget oublie une fiche introuvable.
func (h *MediaHandler) handlePostMediaMissingForget(ctx context.Context, in *mediaMissingBodyInput) (*mediaMissingForgetOutput, error) {
	svc, err := h.missingService(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	var req domain.MediaForgetRequest
	if err := decodeMissingBody(ctx, in.RawBody, &req); err != nil {
		return nil, err
	}
	rawPath := req.FilePath
	req.FilePath = mediaServableURLToStoredPath(in.PlayerSlug, rawPath)
	if req.MediaRequester, err = h.resolveMediaRequester(ctx, "forget"); err != nil {
		return nil, err
	}
	resp, err := svc.ForgetMissingMedia(ctx, req)
	if err != nil {
		return nil, mediaMissingError(ctx, err)
	}
	resp.FilePath = rawPath
	BumpMediaFeedVersion()
	return &mediaMissingForgetOutput{Body: resp}, nil
}

// handlePostMediaMissingConfirmBulk confirme la disparition massive du joueur de la route, puis
// relance sa passe d'indexation en arrière-plan pour qu'elle applique le marquage suspendu.
func (h *MediaHandler) handlePostMediaMissingConfirmBulk(
	ctx context.Context,
	in *mediaMissingConfirmInput,
) (*mediaMissingConfirmOutput, error) {
	svc, err := h.missingService(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	who, err := h.resolveMediaRequester(ctx, "confirm")
	if err != nil {
		return nil, err
	}
	resp, err := svc.ConfirmBulkMissing(ctx, domain.MediaConfirmBulkRequest{MediaRequester: who})
	if err != nil {
		return nil, mediaMissingError(ctx, err)
	}
	if resp.Confirmed {
		h.launchRescan(ctx, in.PlayerSlug)
	}
	return &mediaMissingConfirmOutput{Body: resp}, nil
}

// handlePostMediaReveal ouvre l'explorateur sur le fichier d'un média. Le chemin reçu n'est qu'une
// clé de recherche : le fichier ouvert est celui de la fiche en base.
func (h *MediaHandler) handlePostMediaReveal(ctx context.Context, in *mediaMissingBodyInput) (*mediaRevealOutput, error) {
	svc, err := h.missingService(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	var req domain.MediaRevealRequest
	if err := decodeMissingBody(ctx, in.RawBody, &req); err != nil {
		return nil, err
	}
	req.FilePath = mediaServableURLToStoredPath(in.PlayerSlug, req.FilePath)
	if req.MediaRequester, err = h.resolveMediaRequester(ctx, "reveal"); err != nil {
		return nil, err
	}
	resp, err := svc.RevealMedia(ctx, req)
	if err != nil {
		return nil, mediaMissingError(ctx, err)
	}
	return &mediaRevealOutput{Body: resp}, nil
}

// thumbnailURL rend l'URL servable d'une miniature stockée, nil si elle est absente.
func (h *MediaHandler) thumbnailURL(slug string, stored *string, base string) *string {
	if stored == nil {
		return nil
	}
	u := h.filePathToURL(slug, *stored, base)
	return &u
}

// rescansInFlight : les joueurs dont une passe lancée par une confirmation tourne encore (clé
// « titre/joueur »). Un second clic ne lance pas une seconde passe sur la même base.
var rescansInFlight sync.Map

// launchRescan relance en arrière-plan la passe d'indexation du joueur de la route, détachée de la
// requête (qui se termine avant elle) mais sous le titre de la requête, puis notifie la galerie.
func (h *MediaHandler) launchRescan(ctx context.Context, slug string) {
	if h.rescan == nil {
		return
	}
	titleSlug, gamertag, err := h.rescanTarget(ctx, slug)
	if err != nil {
		slog.WarnContext(ctx, "media_missing: passe non relancée après confirmation, elle aura lieu à la prochaine synchronisation",
			"player", slug, "err", err)
		return
	}
	key := titleSlug + "/" + gamertag
	if _, running := rescansInFlight.LoadOrStore(key, struct{}{}); running {
		return
	}
	bg := ctxkeys.WithTitleSlug(context.WithoutCancel(ctx), titleSlug)
	go func() {
		defer rescansInFlight.Delete(key)
		h.rescan(bg, gamertag)
		BumpMediaFeedVersion()
	}()
}

// rescanTarget rend le titre et le gamertag de la passe à relancer.
func (h *MediaHandler) rescanTarget(ctx context.Context, slug string) (titleSlug, gamertag string, err error) {
	if h.newPlayerCtx == nil {
		return ctxkeys.TitleSlug(ctx), slug, nil
	}
	return h.newPlayerCtx(ctx, slug)
}
