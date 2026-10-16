// Package handlers — media_likes.go : like / unlike d'un média.
//
// Extrait de media.go (v7.3 lot 2, item 1.5) : le like porte trois
// responsabilités distinctes du reste de la galerie — la conversion de l'URL
// servable vers la clé de stockage, la résolution du liker (identité sociale) et
// la notification du propriétaire. Les regrouper ici garde media.go sous le seuil
// de 500 lignes et rend la frontière lisible.
//
// Endpoint : PATCH /api/v1/players/{player_slug}/media/likes
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
)

// handlePatchMediaLike persiste le like/unlike d'un média.
// PATCH /api/v1/players/{player_slug}/media/likes
func (h *MediaHandler) handlePatchMediaLike(ctx context.Context, in *mediaLikeInput) (*mediaLikeOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusNotFound, "player_not_found", err)
	}

	var req domain.MediaLikeRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusBadRequest, "invalid_body", err)
	}

	// Le frontend reçoit les file_path déjà transformés en URL HTTP
	// (cf. transformMediaURLs / filePathToURL). Quand il rejoue ce file_path
	// dans une mutation (like, réassociation), on reverse la transformation vers
	// le chemin RELATIF stocké en DB — exactement comme /media/match-candidates
	// et /media/associate, via l'unique helper mediaServableURLToStoredPath.
	//
	// Historique du bug (v7.3 lot 2, item 1.5) : ce call-site passait par
	// urlToFilePath, qui re-résolvait un chemin ABSOLU disque (os.Stat sous
	// media_captures_base_dir) et appliquait filepath.FromSlash. Depuis la
	// migration des paths (cmd/migrate-media-paths), media_files.file_path est
	// relatif forward-slash ({owner_slug}/{rel}) : l'UPDATE ne touchait plus
	// aucune ligne → 404 systématique dès que le fichier existait sur le disque
	// du serveur (donc en prod, où il existe toujours).
	rawPath := req.FilePath
	req.FilePath = mediaServableURLToStoredPath(in.PlayerSlug, req.FilePath)
	slog.DebugContext(ctx, "media_like: path resolved",
		"slug", in.PlayerSlug, "raw", rawPath, "resolved", req.FilePath)

	if err := h.resolveLikerIdentity(ctx, in.PlayerSlug, &req); err != nil {
		return nil, err
	}

	resp, err := svc.SetMediaLike(ctx, req)
	if err != nil {
		return nil, mediaLikeError(ctx, err)
	}

	// La réponse écho le file_path REÇU (URL servable) : le cache client indexe
	// ses items par cette URL. Renvoyer le chemin stocké ferait échouer toute
	// mise à jour ciblée côté client (likers / compteur jamais appliqués).
	resp.FilePath = rawPath
	out := &mediaLikeOutput{Body: resp}
	BumpMediaFeedVersion()

	// Notification au owner si quelqu'un d'autre a liké son média.
	if req.Liked && req.LikerSlug != "" {
		h.emitMediaLiked(ctx, req.FilePath, req.LikerSlug, req.LikerGamertag)
	}
	return out, nil
}

// resolveLikerIdentity fixe le liker du like : c'est le DEMANDEUR (profil possédé par
// l'utilisateur de la session, media_requester.go), et lui seul. Le like étant
// par-viewer, l'identité du liker décide de QUI verra ce cœur allumé.
//
// Ordre, aligné sur la lecture (le registre résout le viewer des galeries par le même
// résolveur — lire et écrire doivent désigner le même viewer) :
//
//  1. Le demandeur. Il ÉCRASE tout `liker_slug` reçu dans le corps : sans cela un
//     client pourrait liker au nom d'un tiers ou consommer son quota social.
//  2. Propriété appliquée et aucun profil possédé : 401 VISIBLE (garde anti-silence),
//     jamais un repli sur le corps.
//  3. Hors propriété appliquée (mono-utilisateur / démo) sans demandeur : le corps,
//     s'il porte un liker, puis le propriétaire de la page. Sans liker, un like ne
//     s'écrirait NULLE PART (le like global anonyme n'a plus de stockage).
func (h *MediaHandler) resolveLikerIdentity(ctx context.Context, playerSlug string, req *domain.MediaLikeRequest) error {
	requester := h.requesterSlug(ctx)
	if requester != "" || h.authEnforced {
		req.LikerSlug = requester
		// Le libellé affiché dans « ♥ Alice et Bob » se déduit du slug, jamais du
		// corps : sinon un client choisirait le nom sous lequel son like apparaît.
		req.LikerGamertag = ""
	}
	if req.LikerSlug == "" && h.authEnforced {
		slog.WarnContext(ctx, "media_like refusé: aucun profil possédé par la session",
			"slug", playerSlug, "file_path", req.FilePath)
		return humacore.NewError(http.StatusUnauthorized, "like_requires_session",
			"Aucun joueur courant en session : impossible d'attribuer ce like.")
	}
	if req.LikerSlug == "" {
		// Mono-utilisateur / démo : la page consultée est celle du joueur local.
		slog.WarnContext(ctx, "media_like sans session — liker replié sur le propriétaire de la page",
			"slug", playerSlug, "file_path", req.FilePath)
		req.LikerSlug = playerSlug
	}
	if req.LikerGamertag == "" {
		req.LikerGamertag = h.resolveLikerGamertag(ctx, req.LikerSlug)
	}
	return nil
}

// mediaLikeError traduit une erreur de service en erreur HTTP : 503 + Retry-After
// sur base occupée (dblease), 400/404 sur erreur métier typée, 500 sinon.
func mediaLikeError(ctx context.Context, err error) error {
	if errors.Is(err, dblease.ErrDBLocked) {
		return errDBBusy()
	}
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "bad_request":
			return humacore.NewError(http.StatusBadRequest, apiErr.Code, apiErr.Message)
		case "not_found":
			return humacore.NewError(http.StatusNotFound, apiErr.Code, apiErr.Message)
		}
	}
	return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "media_like_error", err)
}

// emitMediaLiked notifie le owner d'un média quand quelqu'un d'autre le like.
// Le owner_slug est déduit du file_path (cf. ownerSlugFromFilePath).
// Silent fail si notifierFor absent, liker == owner, ou owner indéterminable.
func (h *MediaHandler) emitMediaLiked(ctx context.Context, filePath, likerSlug, likerGamertag string) {
	if h.notifierFor == nil {
		return
	}
	ownerSlug := ownerSlugFromFilePath(filePath)
	if ownerSlug == "" || ownerSlug == likerSlug {
		return
	}
	em, err := h.notifierFor(ctx, ownerSlug)
	if err != nil || em == nil {
		return
	}
	displayName := likerGamertag
	if displayName == "" {
		displayName = likerSlug
	}
	// Titre ambiant de la requête (like) — le média liké appartient à ce titre.
	titleSlug := ctxkeys.TitleSlug(ctx)
	_ = em.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategoryMediaLiked,
		Severity:    notifications.SeverityInfo,
		TitleKey:    "notif.media_liked.title",
		BodyKey:     "notif.media_liked.body",
		Params:      map[string]any{"actor_name": displayName},
		TargetRoute: notifications.PlayerTargetRoute(titleSlug, ownerSlug, "media"),
		Actor:       &notifications.Actor{Name: displayName},
		Source:      "media_handler",
	})
}

// ownerSlugFromFilePath extrait le slug du joueur propriétaire depuis le chemin.
// Conventions reconnues :
//   - <slug>/file.ext                      (format CANONIQUE stocké post-migration)
//   - .../Captures/<slug>/file.ext         (capturesBase utilisateur, ex. Windows Captures)
//   - .../players/<slug>/captures/file.ext (chemin interne data/)
//
// Retourne "" si la convention n'est pas détectée.
func ownerSlugFromFilePath(filePath string) string {
	clean := filepath.Clean(filePath)
	parts := strings.Split(clean, string(filepath.Separator))
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "Captures") && i+1 < len(parts)-1 {
			return parts[i+1]
		}
		if parts[i] == "captures" && i > 0 && i+1 < len(parts) {
			// .../<slug>/captures/file.ext → slug est avant
			return parts[i-1]
		}
	}
	// Format canonique stocké en DB ({owner_slug}/{rel}, relatif) : le premier
	// segment EST le propriétaire. Sans cette branche, la notification
	// media_liked disparaît silencieusement dès que le like porte le chemin
	// stocké plutôt qu'un absolu legacy (item 1.5).
	if !filepath.IsAbs(clean) && len(parts) > 1 && parts[0] != "" && parts[0] != "." && parts[0] != ".." {
		return parts[0]
	}
	return ""
}
