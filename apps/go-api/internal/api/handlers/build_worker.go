// Package handlers — build_worker.go : LE PROTOCOLE OUVRIER de la file durable de
// construction (piste F §1/§2 du plan film/killfeed/rejeu).
//
// Quatre routes, sous /api/v1/internal/build-queue/ :
//
//	POST /claim     — prendre le prochain job (rend le travail RÉSOLU, ou rien)
//	POST /artifact  — DÉPOSER l'artefact construit (build_worker_artifact.go)
//	POST /complete  — rendre le résultat d'un job pris
//	POST /heartbeat — signe de vie + prolongation du bail du job en cours
//
// L'ORDRE COMPTE : l'artefact d'abord, le compte rendu ensuite. Un job n'est
// `succeeded` que si son artefact est arrivé et valide — le rendu de résultat ne
// ment donc jamais sur la présence du fichier.
//
// L'OUVRIER TIRE, IL N'EST JAMAIS APPELÉ. Aucun port entrant chez lui, aucun
// système de fichiers partagé, aucun Redis : quatre POST HTTPS suffisent. C'est le
// patron de csstat porté sur la pile Go existante.
//
// CE QUE CE JETON N'OUVRE PAS. Le jeton d'ouvrier (LEVELUP_BUILD_WORKER_TOKEN)
// ne donne accès ni aux tokens Halo, ni à la base, ni à quoi que ce soit d'autre
// que « prendre un travail déjà résolu et en rendre le résultat ». Le web a déjà
// résolu le manifeste et mis des URL CDN PRÉ-SIGNÉES dans le job : c'est
// précisément ce qui rend sûr de faire tourner un ouvrier n'importe où.
//
// SANS JETON CONFIGURÉ, LES QUATRE ROUTES RÉPONDENT 503. Le dépôt est public :
// une installation par défaut ne doit hériter d'aucune porte ouverte.
package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
)

// BuildQueueClaimer prend le prochain job pour un ouvrier (nil = file vide).
// Implémenté par ServiceRegistry.ClaimBuildJob.
type BuildQueueClaimer func(ctx context.Context, workerID, hostname, version string) (*domain.BuildQueueJob, error)

// BuildQueueCompleter enregistre le résultat d'un job.
// Implémenté par ServiceRegistry.CompleteBuildJob.
type BuildQueueCompleter func(ctx context.Context, req ops.CompleteBuildJobRequest) error

// BuildQueueBeater enregistre un battement d'ouvrier (et prolonge le bail).
// Implémenté par ServiceRegistry.HeartbeatBuildWorker.
type BuildQueueBeater func(ctx context.Context, req ops.HeartbeatRequest) error

// BuildWorkerHandler porte le protocole ouvrier.
type BuildWorkerHandler struct {
	token    string
	claim    BuildQueueClaimer
	complete BuildQueueCompleter
	beat     BuildQueueBeater
	// storeArtifact range l'artefact rendu (build_worker_artifact.go). Nil → la
	// route de dépôt répond 503 : sans elle, l'ouvrier construirait sans jamais
	// livrer, et le protocole ne servirait à rien.
	storeArtifact BuildQueueArtifactStorer
}

// NewBuildWorkerHandler construit le handler. token vide → les routes du
// protocole répondent 503 (protocole non ouvert).
func NewBuildWorkerHandler(token string, claim BuildQueueClaimer, complete BuildQueueCompleter, beat BuildQueueBeater) *BuildWorkerHandler {
	return &BuildWorkerHandler{token: strings.TrimSpace(token), claim: claim, complete: complete, beat: beat}
}

// WithArtifactStore branche le dépôt d'artefact (route /artifact). Chaînable.
func (h *BuildWorkerHandler) WithArtifactStore(store BuildQueueArtifactStorer) *BuildWorkerHandler {
	h.storeArtifact = store
	return h
}

// Mount enregistre les routes du protocole sur le sous-routeur donné.
//
// Le jeton est vérifié par un MIDDLEWARE, donc AVANT que Huma ne regarde le
// corps de la requête. Ce n'est pas un détail de style : validé après, un appelant
// non authentifié obtiendrait des 422 détaillant le schéma attendu — une porte
// fermée ne doit pas décrire ce qu'il y a derrière.
func (h *BuildWorkerHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r.With(h.RequireWorkerToken), opts...)
	huma.Post(api, "/build-queue/claim", h.handleClaim, humacore.Op(
		"postBuildQueueClaim",
		"Protocole ouvrier — prend le prochain job de construction et rend son travail résolu (URL CDN pré-signées). Jeton d'ouvrier requis.",
		"build-queue"))
	huma.Post(api, "/build-queue/complete", h.handleComplete, humacore.Op(
		"postBuildQueueComplete",
		"Protocole ouvrier — rend le résultat d'un job pris (succès ou échec). Jeton d'ouvrier requis.",
		"build-queue"))
	huma.Post(api, "/build-queue/heartbeat", h.handleHeartbeat, humacore.Op(
		"postBuildQueueHeartbeat",
		"Protocole ouvrier — signe de vie de l'ouvrier et prolongation du bail du job en cours. Jeton d'ouvrier requis.",
		"build-queue"))
	h.mountArtifactRoute(api)
}

// ─── Contrats d'entrée/sortie ────────────────────────────────────────────────

// BuildQueueClaimRequest : l'ouvrier se présente et demande du travail.
type BuildQueueClaimRequest struct {
	WorkerID string `json:"worker_id"`
	Hostname string `json:"hostname,omitempty"`
	Version  string `json:"version,omitempty"`
}

// BuildQueueClaimResponse : le job pris, ou rien. Job nil + Job==nil signifie
// « file vide » — un 200 avec un corps vide, PAS une erreur : un ouvrier au repos
// est le cas nominal, il ne doit pas remplir les logs d'erreurs.
type BuildQueueClaimResponse struct {
	Job *domain.BuildQueueJob `json:"job,omitempty"`
	// LeaseSeconds : le temps dont dispose l'ouvrier avant que son job ne
	// retourne en file. Publié pour qu'il n'ait pas à deviner la cadence de
	// battement à laquelle il doit tenir.
	LeaseSeconds int `json:"lease_seconds"`
}

// BuildQueueCompleteRequest : le résultat rendu.
type BuildQueueCompleteRequest struct {
	JobID        string `json:"job_id"`
	WorkerID     string `json:"worker_id"`
	Succeeded    bool   `json:"succeeded"`
	ResultJSON   string `json:"result_json,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// BuildQueueHeartbeatRequest : le signe de vie.
type BuildQueueHeartbeatRequest struct {
	WorkerID   string `json:"worker_id"`
	Hostname   string `json:"hostname,omitempty"`
	Version    string `json:"version,omitempty"`
	JobID      string `json:"job_id,omitempty"`
	Note       string `json:"note,omitempty"`
	JobsDone   int64  `json:"jobs_done,omitempty"`
	JobsFailed int64  `json:"jobs_failed,omitempty"`
}

// BuildQueueAckResponse : accusé d'un rendu ou d'un battement.
type BuildQueueAckResponse struct {
	OK bool `json:"ok"`
}

// Les trois entrées ne déclarent PAS l'en-tête Authorization : il est consommé
// par RequireWorkerToken avant que Huma ne voie la requête.
type buildQueueClaimInput struct {
	Body BuildQueueClaimRequest
}
type buildQueueCompleteInput struct {
	Body BuildQueueCompleteRequest
}
type buildQueueHeartbeatInput struct {
	Body BuildQueueHeartbeatRequest
}

type buildQueueClaimOutput struct{ Body BuildQueueClaimResponse }
type buildQueueAckOutput struct{ Body BuildQueueAckResponse }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleClaim prend le prochain job de la file.
func (h *BuildWorkerHandler) handleClaim(ctx context.Context, in *buildQueueClaimInput) (*buildQueueClaimOutput, error) {
	if h.claim == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "build_queue_unavailable",
			"File de construction indisponible (store non câblé).")
	}
	workerID := strings.TrimSpace(in.Body.WorkerID)
	if workerID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "worker_id requis.")
	}
	job, err := h.claim(ctx, workerID, in.Body.Hostname, in.Body.Version)
	if err != nil {
		slog.ErrorContext(ctx, "build_queue: prise de job échouée",
			"module", "monitoring", "worker_id", workerID, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "build_queue_claim_error",
			"Impossible de prendre un job dans la file.")
	}
	return &buildQueueClaimOutput{Body: BuildQueueClaimResponse{
		Job:          job,
		LeaseSeconds: int(domain.BuildJobLeaseDuration.Seconds()),
	}}, nil
}

// handleComplete enregistre le résultat d'un job pris.
func (h *BuildWorkerHandler) handleComplete(ctx context.Context, in *buildQueueCompleteInput) (*buildQueueAckOutput, error) {
	if h.complete == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "build_queue_unavailable",
			"File de construction indisponible (store non câblé).")
	}
	if strings.TrimSpace(in.Body.JobID) == "" || strings.TrimSpace(in.Body.WorkerID) == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "job_id et worker_id requis.")
	}
	err := h.complete(ctx, ops.CompleteBuildJobRequest{
		JobID:        in.Body.JobID,
		WorkerID:     in.Body.WorkerID,
		Succeeded:    in.Body.Succeeded,
		ResultJSON:   in.Body.ResultJSON,
		ErrorCode:    in.Body.ErrorCode,
		ErrorMessage: in.Body.ErrorMessage,
	})
	if errors.Is(err, ops.ErrBuildJobNotClaimed) {
		// 409 et pas 400 : l'ouvrier n'a rien fait de mal, son travail est
		// simplement PÉRIMÉ (bail expiré, job repris par un autre). Il doit jeter
		// son résultat et redemander un job — un 400 lui ferait croire à un bug.
		slog.WarnContext(ctx, "build_queue: résultat périmé refusé",
			"module", "monitoring", "job_id", in.Body.JobID, "worker_id", in.Body.WorkerID, "err", err)
		return nil, humacore.NewError(http.StatusConflict, "build_job_not_claimed",
			"Ce job n'est plus détenu par cet ouvrier (bail expiré ou job repris).")
	}
	if err != nil {
		slog.ErrorContext(ctx, "build_queue: rendu de résultat échoué",
			"module", "monitoring", "job_id", in.Body.JobID, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "build_queue_complete_error",
			"Impossible d'enregistrer le résultat du job.")
	}
	return &buildQueueAckOutput{Body: BuildQueueAckResponse{OK: true}}, nil
}

// handleHeartbeat enregistre un signe de vie et prolonge le bail du job en cours.
func (h *BuildWorkerHandler) handleHeartbeat(ctx context.Context, in *buildQueueHeartbeatInput) (*buildQueueAckOutput, error) {
	if h.beat == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "build_queue_unavailable",
			"File de construction indisponible (store non câblé).")
	}
	if strings.TrimSpace(in.Body.WorkerID) == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "worker_id requis.")
	}
	if err := h.beat(ctx, ops.HeartbeatRequest{
		WorkerID: in.Body.WorkerID, Hostname: in.Body.Hostname, Version: in.Body.Version,
		JobID: in.Body.JobID, Note: in.Body.Note,
		JobsDone: in.Body.JobsDone, JobsFailed: in.Body.JobsFailed,
	}); err != nil {
		slog.ErrorContext(ctx, "build_queue: battement d'ouvrier non enregistré",
			"module", "monitoring", "worker_id", in.Body.WorkerID, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "build_queue_heartbeat_error",
			"Impossible d'enregistrer le battement de l'ouvrier.")
	}
	return &buildQueueAckOutput{Body: BuildQueueAckResponse{OK: true}}, nil
}

// RequireWorkerToken ferme les quatre routes du protocole derrière le jeton
// d'ouvrier (Bearer). Comparaison à temps constant : un jeton partagé se devine à
// l'oreille sur un comparateur naïf.
//
// Sans jeton configuré → 503 : la feature n'existe pas sur cette instance. Avec
// un jeton présenté qui ne correspond pas → 401.
func (h *BuildWorkerHandler) RequireWorkerToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.token == "" {
			writeWorkerAuthError(w, http.StatusServiceUnavailable, "build_queue_disabled",
				"Le protocole ouvrier n'est pas ouvert sur cette instance (aucun jeton configuré).")
			return
		}
		presented := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer"))
		if subtle.ConstantTimeCompare([]byte(presented), []byte(h.token)) != 1 {
			slog.WarnContext(r.Context(), "build_queue: jeton d'ouvrier refusé",
				"module", "monitoring", "path", r.URL.Path, "ip", r.RemoteAddr)
			writeWorkerAuthError(w, http.StatusUnauthorized, "invalid_worker_token",
				"Jeton d'ouvrier invalide.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeWorkerAuthError écrit le corps d'erreur du contrat API ({code, message,
// retryable}) depuis le middleware, où huma.StatusError n'est pas disponible.
func writeWorkerAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		jsonKeyCode: code, jsonKeyMessage: message, jsonKeyRetryable: false,
	})
}
