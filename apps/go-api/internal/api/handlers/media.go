// Package handlers — media.go : handler HTTP pour la galerie médias.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// chi (préfixe /players/{player_slug} + middleware ownership/title/CapMedia hérités,
// lit {player_slug} parent) et enregistre les 5 routes JSON via huma.Post/Patch/Get.
// PostUploadMedia (multipart) et ServeMediaFile (fichier binaire) restent chi —
// hors scope JSON. Logique métier inchangée, seul le wrapping HTTP change.
//
// Endpoints :
//
//	POST /api/v1/players/{player_slug}/pages/media            → MediaPageResponse (Huma, body optionnel)
//	PATCH /api/v1/players/{player_slug}/media/likes            → MediaLikeResponse (Huma)
//	GET  /api/v1/players/{player_slug}/media/match-candidates  → MediaMatchCandidatesResponse (Huma)
//	POST /api/v1/players/{player_slug}/media/associate         → MediaAssociateResponse (Huma)
//	GET  /api/v1/players/{player_slug}/media/authors           → MediaAuthorsResponse (Huma)
//	POST /api/v1/players/{player_slug}/media/upload            → UploadResult (multipart, reste chi)
//	GET  /api/v1/players/{player_slug}/media/files/*           → fichier servi depuis captures (reste chi)
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/mediaaudio"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// Mount enregistre les 5 routes JSON via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title/CapMedia hérités). Seul
// /pages/media a un corps optionnel (MarkRequestBodyOptional). PostUploadMedia
// (multipart) et ServeMediaFile (binaire) restent enregistrés en chi par server.go.
func (h *MediaHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/media", h.handleGetMediaLibrary, humacore.Op("postMediaLibrary", "Galerie médias (filtres en body)", "media"))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/media")
	huma.Patch(api, "/media/likes", h.handlePatchMediaLike, humacore.Op("patchMediaLike", "Like / unlike d'un média", "media"))
	huma.Get(api, "/media/match-candidates", h.handleGetMediaMatchCandidates, humacore.Op("getMediaMatchCandidates", "Candidats de match pour associer un média", "media"))
	huma.Post(api, "/media/associate", h.handlePostMediaAssociate, humacore.Op("postMediaAssociate", "Associe un média à un match", "media"))
	huma.Get(api, "/media/authors", h.handleGetMediaAuthors, humacore.Op("getMediaAuthors", "Liste des auteurs de médias visibles pour ce joueur", "media"))
	huma.Get(api, "/media/audio-config", h.handleGetMediaAudioConfig, humacore.Op("getMediaAudioConfig", "Réglage des pistes audio des médias du joueur", "media"))
	huma.Put(api, "/media/audio-config", h.handlePutMediaAudioConfig, humacore.Op("putMediaAudioConfig", "Définit le réglage des pistes audio des médias du joueur", "media"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// mediaLibraryInput : {player_slug} parent + corps brut OPTIONNEL décodé maison.
type mediaLibraryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}
type mediaLibraryOutput struct{ Body *domain.MediaPageResponse }

// mediaLikeInput : {player_slug} + corps REQUIS (RawBody, décodage maison → 400 invalid_body).
type mediaLikeInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}
type mediaLikeOutput struct{ Body *domain.MediaLikeResponse }

// mediaCandidatesInput : {player_slug} + query file_path (requis) + window_minutes
// (string, parsé maison → fallback 15, jamais 422).
type mediaCandidatesInput struct {
	PlayerSlug    string `path:"player_slug"`
	FilePath      string `query:"file_path"`
	WindowMinutes string `query:"window_minutes"`
}
type mediaCandidatesOutput struct {
	Body *domain.MediaMatchCandidatesResponse
}

// mediaAssociateInput : {player_slug} + corps REQUIS.
type mediaAssociateInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}
type mediaAssociateOutput struct {
	Body *domain.MediaAssociateResponse
}

// mediaAuthorsInput : {player_slug} seul (pas de corps, pas de query).
type mediaAuthorsInput struct {
	PlayerSlug string `path:"player_slug"`
}
type mediaAuthorsOutput struct{ Body domain.MediaAuthorsResponse }

// mediaAudioConfigInput : {player_slug} seul (GET).
type mediaAudioConfigInput struct {
	PlayerSlug string `path:"player_slug"`
}

// mediaAudioConfigPutInput : {player_slug} + corps REQUIS (RawBody, décodage maison
// → 400 invalid_body ; même motif que mediaLikeInput pour éviter un schéma d'input Huma).
type mediaAudioConfigPutInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type mediaAudioConfigOutput struct {
	Body *domain.PlayerMediaAudioConfig
}

// maxUploadSize limite la taille totale d'un upload à 500 Mo.
const maxUploadSize = 500 << 20

// maxUploadFiles limite le nombre de fichiers par requête.
const maxUploadFiles = 50

// defaultMediaMatchWindowMinutes : fenêtre ± (minutes) d'association d'un média à un match
// par proximité temporelle, appliquée quand le client ne fournit pas window_minutes.
const defaultMediaMatchWindowMinutes = 15

// allowedMediaExts liste les extensions acceptées en upload.
var allowedMediaExts = map[string]bool{
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
	".png": true, ".jpg": true, ".jpeg": true, ".bmp": true, ".gif": true,
}

// MediaUploadContextFactory retourne service + gamertag + titleSlug + dbPath + sharedSocialDBPath + sharedMatchesDBPath.
// Utilisée par PostUploadMedia pour résoudre le répertoire captures.
type MediaUploadContextFactory func(ctx context.Context, slug string) (
	port.MediaService, string, string, string, string, string, error,
)

// MediaPlayerContextFactory résout slug → (titleSlug, gamertag) sans construire de service.
// Utilisée par les endpoints méta-données (ex : liste des auteurs).
type MediaPlayerContextFactory func(ctx context.Context, slug string) (titleSlug, gamertag string, err error)

// MediaProfilesProvider liste les profils connus pour un titre (db_profiles.json).
type MediaProfilesProvider func(ctx context.Context, titleSlug string) ([]domain.PlayerSummary, error)

// MediaHandler gère les endpoints de la galerie médias.
type MediaHandler struct {
	newSvc            ServiceFactory[port.MediaService]
	newUpload         MediaUploadContextFactory
	newPlayerCtx      MediaPlayerContextFactory
	loadProfiles      MediaProfilesProvider
	repoRoot          string
	settingsStore     *settings.Store
	notifierFor       NotificationsEmitterFactory // optionnel : émission media_added
	recipientResolver MediaRecipientResolver      // optionnel : fan-out aux autres joueurs
	demoMode          bool                        // true = upload figé (vitrine publique)
	isProduction      bool                        // défaut de rétention source (env LEVELUP_ENV=production)
	authEnforced      bool                        // true = multi-user authentifié (like sans session refusé)
}

// NewMediaHandler crée un MediaHandler.
// newUpload peut être nil : dans ce cas l'endpoint upload retourne 501.
func NewMediaHandler(
	newSvc ServiceFactory[port.MediaService],
	newUpload MediaUploadContextFactory,
	repoRoot string,
) *MediaHandler {
	return &MediaHandler{newSvc: newSvc, newUpload: newUpload, repoRoot: repoRoot}
}

// WithSettingsStore injecte le settings store pour lire media_captures_base_dir.
func (h *MediaHandler) WithSettingsStore(store *settings.Store) *MediaHandler {
	h.settingsStore = store
	return h
}

// WithDemoMode fige l'upload de médias en mode démo (vitrine publique partagée).
// Sans appel : false (upload autorisé).
func (h *MediaHandler) WithDemoMode(demo bool) *MediaHandler {
	h.demoMode = demo
	return h
}

// WithProduction fournit le défaut de résolution de la politique de rétention du
// source après transcodage HLS (config.ResolveMediaDeleteSource : env > store >
// isProd). Sans appel : false (conserver le source — défaut sûr hors prod).
func (h *MediaHandler) WithProduction(isProd bool) *MediaHandler {
	h.isProduction = isProd
	return h
}

// WithAuthEnforced déclare que l'authentification est APPLIQUÉE sur cette
// instance (multi-utilisateur : ni démo, ni auth_mode=none). Dans ce mode, un
// like sans joueur courant en session est refusé (401 like_requires_session)
// au lieu d'être écrit comme like anonyme non attribuable — cf. garde
// anti-silence de handlePatchMediaLike. Sans appel : false (comportement
// mono-utilisateur historique conservé).
func (h *MediaHandler) WithAuthEnforced(enforced bool) *MediaHandler {
	h.authEnforced = enforced
	return h
}

// WithAuthorsContext câble la résolution slug → (titleSlug, gamertag) et la liste
// des profils du titre. Optionnel : sert uniquement à enrichir le gamertag d'affichage
// dans GetMediaAuthors ; sans ces callbacks, l'endpoint retombe sur le player_slug
// comme libellé (la liste d'auteurs elle-même vient de la DB via le service).
func (h *MediaHandler) WithAuthorsContext(playerCtx MediaPlayerContextFactory, loadProfiles MediaProfilesProvider) *MediaHandler {
	h.newPlayerCtx = playerCtx
	h.loadProfiles = loadProfiles
	return h
}

// WithNotificationsEmitterFactory branche la factory d'émetteurs de notifications
// pour publier media_added après un upload réussi.
func (h *MediaHandler) WithNotificationsEmitterFactory(f NotificationsEmitterFactory) *MediaHandler {
	h.notifierFor = f
	return h
}

// MediaRecipientResolver retourne la liste des player_slug à notifier après un
// upload, à partir du sharedSocialDBPath et du sharedMatchesDBPath. Doit exclure
// l'uploader (passé via uploaderSlug).
type MediaRecipientResolver func(
	ctx context.Context,
	uploaderSlug, sharedSocialDBPath, sharedMatchesDBPath string,
	since time.Time,
) ([]string, error)

// WithMediaRecipientResolver branche le resolver de destinataires (fan-out).
// Si nil, l'émission media_added reste limitée à l'uploader.
func (h *MediaHandler) WithMediaRecipientResolver(r MediaRecipientResolver) *MediaHandler {
	h.recipientResolver = r
	return h
}

// mediaCoalesceWindow : fenêtre de coalescence des notifs media_added par
// acteur (DP5). Incident 2026-07-03 : 5 clips du même acteur en 5 min = 5 notifs.
const mediaCoalesceWindow = time.Hour

// emitMediaAdded émet media_added pour l'uploader puis fan-out vers les autres
// joueurs associés aux matchs concernés.
func (h *MediaHandler) emitMediaAdded(
	ctx context.Context,
	uploaderSlug, gamertag string,
	newIndexed int,
	since time.Time,
	sharedSocialDBPath, sharedMatchesDBPath string,
) {
	if h.notifierFor == nil || newIndexed <= 0 {
		return
	}
	// Fan-out aux autres joueurs (max ~5 destinataires).
	recipients := []string{}
	if h.recipientResolver != nil {
		var err error
		recipients, err = h.recipientResolver(ctx, uploaderSlug, sharedSocialDBPath, sharedMatchesDBPath, since)
		if err != nil {
			slog.WarnContext(ctx, "notifications: media_added recipients", "err", err)
		}
	}
	// Titre du contexte de requête (TitleExtractor). Le média et ses destinataires
	// vivent tous dans ce titre (shared_social/shared_matches sont per-titre) —
	// title-agnostic : on lit le titre ambiant, jamais de slug en dur.
	titleSlug := ctxkeys.TitleSlug(ctx)
	for _, slug := range recipients {
		if slug == uploaderSlug {
			continue // safety : exclure l'uploader (le resolver est censé le faire aussi)
		}
		em, err := h.notifierFor(ctx, slug)
		if err != nil || em == nil {
			continue
		}
		// Coalescence 1 h par acteur (DP5) : 5 clips du même joueur en 5 min →
		// 1 notif count=5 (incident 2026-07-03), jamais 5 notifs distinctes.
		_ = em.EmitCoalesced(ctx, notifications.EmitInput{
			Category:    notifications.CategoryMediaAdded,
			Severity:    notifications.SeverityInfo,
			TitleKey:    "notif.media_added.title",
			BodyKey:     "notif.media_added.body",
			Params:      map[string]any{"actor_name": gamertag, jsonKeyCount: newIndexed},
			TargetRoute: notifications.PlayerTargetRoute(titleSlug, slug, "media"),
			Actor:       &notifications.Actor{Name: gamertag},
			Source:      "media_handler",
		}, mediaCoalesceWindow)
	}
}

// GetMediaLibrary retourne la page paginée de la galerie médias.
// POST /api/v1/players/{player_slug}/pages/media
// Body (optionnel) : { "page": 1, "page_size": 24, "kind": "clip" }
func (h *MediaHandler) handleGetMediaLibrary(ctx context.Context, in *mediaLibraryInput) (*mediaLibraryOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	req := domain.MediaPageRequest{Page: 1}
	if len(in.RawBody) > 0 {
		if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	resp, err := svc.GetMediaPage(ctx, req)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "media_page_error", err.Error())
	}

	// Transformer les chemins absolus en URLs servables (mutation in-place inchangée).
	h.transformMediaURLs(in.PlayerSlug, resp.Items.Items)

	return &mediaLibraryOutput{Body: resp}, nil
}

// PostUploadMedia reçoit des fichiers via multipart/form-data, les sauvegarde
// dans le répertoire captures du joueur, puis déclenche l'indexation immédiate.
// POST /api/v1/players/{player_slug}/media/upload
func (h *MediaHandler) handleGetMediaMatchCandidates(ctx context.Context, in *mediaCandidatesInput) (*mediaCandidatesOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	if in.FilePath == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_file_path", "file_path query param requis")
	}
	// La galerie sert file_path sous forme d'URL servable (vidéo → playlist HLS
	// .../hls/<stem>/master.m3u8). On recompose le chemin RELATIF stocké pour que le
	// lookup media_files matche mf.file_path — sinon ErrNoRows → 500 « loading error ».
	filePath := mediaServableURLToStoredPath(in.PlayerSlug, in.FilePath)
	window := defaultMediaMatchWindowMinutes
	if in.WindowMinutes != "" {
		if n, err := strconv.Atoi(in.WindowMinutes); err == nil && n > 0 {
			window = n
		}
	}
	resp, err := svc.GetMatchCandidates(ctx, filePath, window)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "candidates_error", err.Error())
	}
	return &mediaCandidatesOutput{Body: resp}, nil
}

// PostMediaAssociate force l'association d'un média à un match précis.
// POST /api/v1/players/{player_slug}/media/associate { file_path, match_id }
func (h *MediaHandler) handlePostMediaAssociate(ctx context.Context, in *mediaAssociateInput) (*mediaAssociateOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	var req domain.MediaAssociateRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	// Symétrie avec /media/match-candidates : la galerie envoie l'URL servable
	// (vidéo → HLS). On recompose le chemin RELATIF stocké pour que l'UPDATE cible
	// la bonne ligne media_files (sinon 0 row → association silencieusement perdue).
	req.FilePath = mediaServableURLToStoredPath(in.PlayerSlug, req.FilePath)
	resp, err := svc.AssociateMediaToMatch(ctx, req)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "associate_error", err.Error())
	}
	out := &mediaAssociateOutput{Body: resp}
	BumpMediaFeedVersion()
	return out, nil
}

// GetMediaAuthors retourne la liste des auteurs sélectionnables dans le filtre
// Auteurs : les player_slug distincts présents dans shared_social.media_files
// (même source que la galerie), enrichis du gamertag d'affichage via db_profiles.json.
//
// Historique : cet endpoint scannait le filesystem (countMediaInDir) par gamertag,
// ce qui renvoyait une liste vide dès que les captures d'un auteur n'étaient pas
// présentes sur le disque local (multi-user) ou rangées en sous-dossiers — d'où le
// bug "Aucun auteur disponible" alors que la galerie affichait bien des médias. La
// source est désormais la DB, strictement cohérente avec ce que la galerie peut afficher.
// GET /api/v1/players/{player_slug}/media/authors
func (h *MediaHandler) handleGetMediaAuthors(ctx context.Context, in *mediaAuthorsInput) (*mediaAuthorsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	authors, err := svc.ListMediaAuthors(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "authors_query_error", err.Error())
	}

	// Enrichissement du gamertag d'affichage (best-effort) depuis db_profiles.json.
	// Sans contexte profils câblé (WithAuthorsContext), on retombe sur le player_slug.
	gamertagBySlug := h.authorGamertags(ctx, in.PlayerSlug)
	for i := range authors {
		if gt := gamertagBySlug[strings.ToLower(authors[i].PlayerSlug)]; gt != "" {
			authors[i].Gamertag = gt
		} else {
			authors[i].Gamertag = authors[i].PlayerSlug
		}
	}

	return &mediaAuthorsOutput{Body: domain.MediaAuthorsResponse{Authors: authors}}, nil
}

// handleGetMediaAudioConfig retourne le réglage audio média du joueur (rôle des
// pistes voix/jeu/autres, mode auto/manuel). Défaut auto si aucun sidecar.
// GET /api/v1/players/{player_slug}/media/audio-config
func (h *MediaHandler) handleGetMediaAudioConfig(ctx context.Context, in *mediaAudioConfigInput) (*mediaAudioConfigOutput, error) {
	store, err := h.audioConfigStore(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	// Load renvoie le défaut auto même en cas de sidecar illisible : on dégrade
	// proprement (200 + auto) tout en loggant, plutôt que d'échouer le réglage.
	cfg, loadErr := store.Load()
	if loadErr != nil {
		slog.WarnContext(ctx, "media audio-config: chargement échoué, défaut auto",
			"slug", in.PlayerSlug, "err", loadErr)
	}
	return &mediaAudioConfigOutput{Body: &cfg}, nil
}

// handlePutMediaAudioConfig valide et persiste le réglage audio média du joueur.
// Le champ updated_at est toujours réécrit côté serveur (autoritaire).
// PUT /api/v1/players/{player_slug}/media/audio-config
func (h *MediaHandler) handlePutMediaAudioConfig(ctx context.Context, in *mediaAudioConfigPutInput) (*mediaAudioConfigOutput, error) {
	store, err := h.audioConfigStore(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	var cfg domain.PlayerMediaAudioConfig
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&cfg); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := cfg.Validate(); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_audio_config", err.Error())
	}
	cfg.UpdatedAt = time.Now().UTC()
	if err := store.Save(cfg); err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "audio_config_save_error", err.Error())
	}
	slog.InfoContext(ctx, "media audio-config: enregistré",
		"slug", in.PlayerSlug, "mode", cfg.Mode, "tracks", len(cfg.TrackRoles))
	return &mediaAudioConfigOutput{Body: &cfg}, nil
}

// audioConfigStore résout slug → (titleSlug, gamertag) via newPlayerCtx puis construit
// le store du sidecar media_audio_config.json (PathResolver, pas de MediaRepository).
func (h *MediaHandler) audioConfigStore(ctx context.Context, slug string) (*mediaaudio.Store, error) {
	if h.newPlayerCtx == nil {
		return nil, humacore.NewError(http.StatusNotImplemented, "audio_config_not_configured",
			"résolution joueur non configurée")
	}
	titleSlug, gamertag, err := h.newPlayerCtx(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	path := titlePkg.NewPathResolver(h.repoRoot).PlayerMediaAudioConfigPath(titleSlug, gamertag)
	return mediaaudio.NewStore(path), nil
}

// authorGamertags retourne un index lower(slug|gamertag) → gamertag construit depuis
// db_profiles.json, pour afficher un libellé propre dans le filtre Auteurs. Best-effort :
// retourne une map vide si le contexte profils n'est pas câblé (WithAuthorsContext).
func (h *MediaHandler) authorGamertags(ctx context.Context, slug string) map[string]string {
	out := map[string]string{}
	if h.newPlayerCtx == nil || h.loadProfiles == nil {
		return out
	}
	titleSlug, _, err := h.newPlayerCtx(ctx, slug)
	if err != nil {
		return out
	}
	profiles, err := h.loadProfiles(ctx, titleSlug)
	if err != nil {
		return out
	}
	for _, p := range profiles {
		if p.Gamertag == "" {
			continue
		}
		out[strings.ToLower(p.PlayerSlug)] = p.Gamertag
		out[strings.ToLower(p.Gamertag)] = p.Gamertag
	}
	return out
}
