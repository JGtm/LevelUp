// Package api — server_build_worker.go : montage du PROTOCOLE OUVRIER.
//
// HORS DU GROUPE /admin, ET C'EST VOULU. Un ouvrier n'est pas un administrateur
// connecté : il n'a ni session, ni cookie, ni compte. Il présente un jeton dédié
// (LEVELUP_BUILD_WORKER_TOKEN) qui n'ouvre QUE ces trois routes — prendre un
// travail déjà résolu, battre, rendre un résultat. Aucun accès Halo, aucun accès
// base, aucune autre route.
//
// Sans jeton configuré, les trois routes répondent 503 : le dépôt est public, et
// une installation par défaut ne doit hériter d'aucune porte ouverte.
package wire

import (
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
)

// MountBuildWorkerRoutes monte
// /internal/build-queue/{claim,artifact,complete,heartbeat}.
// NoStore : ce sont des transitions d'état, jamais du contenu cacheable.
func MountBuildWorkerRoutes(r chi.Router, reg *ServiceRegistry, apiOpt humacore.MountOption) {
	h := handlers.NewBuildWorkerHandler(
		reg.cfg.BuildWorkerToken,
		reg.ClaimBuildJob, reg.CompleteBuildJob, reg.HeartbeatBuildWorker).
		WithArtifactStore(reg.StoreBuildArtifact)
	h.Mount(r.With(middleware.NoStore), apiOpt)
}
