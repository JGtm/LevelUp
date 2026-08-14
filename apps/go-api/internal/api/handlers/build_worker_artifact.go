// Package handlers — build_worker_artifact.go : LE TRANSPORT DE L'ARTEFACT.
//
//	POST /api/v1/internal/build-queue/artifact?job_id=…&worker_id=…
//	corps = l'artefact de rejeu, tel quel (application/json, ~2 Mo)
//
// SANS CETTE ROUTE, LA CHAÎNE NE SERT À RIEN. L'ouvrier savait déjà prendre un
// travail, le décoder et rendre un COMPTE RENDU — mais l'artefact restait chez
// lui. C'est ici qu'il arrive, et c'est le web qui le range : l'ouvrier n'a aucun
// port entrant, il pousse ; l'app est le seul point d'entrée et le seul lieu de
// stockage.
//
// L'IDENTITÉ DU JOB VOYAGE EN PARAMÈTRES D'URL, PAS DANS LE CORPS. Le corps EST
// l'artefact : l'envelopper dans un objet {job_id, artefact} obligerait à
// ré-encoder 2 Mo de JSON dans une chaîne JSON (double coût mémoire des deux
// côtés) pour ne rien gagner.
//
// CE QUI EST VÉRIFIÉ AVANT LA MOINDRE ÉCRITURE (chacun rend un refus distinct) :
//   - 413 : corps au-delà de domain.MaxBuildArtifactBytes — borné par Huma
//     LUI-MÊME (MaxBodyBytes), donc les octets excédentaires ne sont jamais lus ;
//   - 409 : le job n'existe pas, n'est pas en cours, ou appartient à un AUTRE
//     ouvrier (même refus que `complete` : le travail est périmé) ;
//   - 400 : le contenu n'est pas l'artefact attendu (illisible, mauvaise version
//     de schéma, mauvais match) — rien n'est écrit.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
)

// BuildQueueArtifactStorer valide et range l'artefact d'un job pris.
// Implémenté par ServiceRegistry.StoreBuildArtifact.
type BuildQueueArtifactStorer func(ctx context.Context, jobID, workerID string, blob []byte) (domain.BuildArtifactReceipt, error)

// buildQueueArtifactInput : l'identité du job en paramètres, l'artefact en corps
// brut. RawBody, et pas un type de corps décodé par Huma : le document n'a pas à
// être désérialisé DEUX fois (une par Huma, une par la validation) pour 2 Mo.
type buildQueueArtifactInput struct {
	JobID    string `query:"job_id"`
	WorkerID string `query:"worker_id"`
	RawBody  []byte
}

type buildQueueArtifactOutput struct {
	Body domain.BuildArtifactReceipt
}

// mountArtifactRoute enregistre le dépôt d'artefact sur l'API du protocole.
// Séparé de Mount pour que le plafond de corps (MaxBodyBytes) reste VISIBLE à
// l'enregistrement : c'est le garde-fou du disque du VPS, pas un détail.
func (h *BuildWorkerHandler) mountArtifactRoute(api huma.API) {
	huma.Post(api, "/build-queue/artifact", h.handleArtifact, humacore.Op(
		"postBuildQueueArtifact",
		"Protocole ouvrier — dépose l'artefact de rejeu construit pour un job pris (corps = l'artefact, taille bornée). Jeton d'ouvrier requis.",
		"build-queue"),
		humacore.MaxBody(domain.MaxBuildArtifactBytes))
}

// handleArtifact reçoit l'artefact d'un job en cours.
func (h *BuildWorkerHandler) handleArtifact(ctx context.Context, in *buildQueueArtifactInput) (*buildQueueArtifactOutput, error) {
	if h.storeArtifact == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "build_queue_unavailable",
			"File de construction indisponible (store non câblé).")
	}
	jobID, workerID := strings.TrimSpace(in.JobID), strings.TrimSpace(in.WorkerID)
	if jobID == "" || workerID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "job_id et worker_id requis.")
	}
	// Second garde-fou de taille, redondant avec MaxBodyBytes ET assumé comme tel :
	// le plafond ne doit pas dépendre d'un seul point de câblage.
	if len(in.RawBody) > domain.MaxBuildArtifactBytes {
		return nil, humacore.NewError(http.StatusRequestEntityTooLarge, "build_artifact_too_large",
			"Artefact au-delà du plafond accepté.")
	}

	receipt, err := h.storeArtifact(ctx, jobID, workerID, in.RawBody)
	switch {
	case errors.Is(err, ops.ErrBuildJobNotClaimed):
		slog.WarnContext(ctx, "build_queue: artefact d'un job non détenu refusé",
			"module", "monitoring", "job_id", jobID, "worker_id", workerID, "err", err)
		return nil, humacore.NewError(http.StatusConflict, "build_job_not_claimed",
			"Ce job n'est plus détenu par cet ouvrier (bail expiré ou job repris).")
	case errors.Is(err, domain.ErrBuildArtifactInvalid):
		// Le motif est rendu à l'ouvrier : c'est lui qui doit apprendre qu'il
		// construit avec un décodeur périmé, et personne d'autre ne le lui dira.
		slog.WarnContext(ctx, "build_queue: artefact invalide refusé",
			"module", "monitoring", "job_id", jobID, "worker_id", workerID,
			"bytes", len(in.RawBody), "err", err)
		return nil, humacore.NewError(http.StatusBadRequest, "build_artifact_invalid", err.Error())
	case err != nil:
		slog.ErrorContext(ctx, "build_queue: dépôt d'artefact échoué",
			"module", "monitoring", "job_id", jobID, "worker_id", workerID, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "build_artifact_store_error",
			"Impossible de ranger l'artefact.")
	}
	return &buildQueueArtifactOutput{Body: receipt}, nil
}
