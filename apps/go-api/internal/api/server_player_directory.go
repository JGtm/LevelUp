// Package api — server_player_directory.go : assemblage de l'annuaire des
// joueurs (ADR 0035 D2).
//
// Wiring pur (aucune décision métier) : il branche les cinq sources de lecture
// sur ce que le process possède déjà — les profils (`config.AppConfig`), les
// comptes (`userstore.Store`), les credentials (`MultiUserTokenStore`, ADR 0023),
// le suivi live (le daemon watcher) et le disque (`PathResolver`).
//
// Une source absente retire sa part de la réponse sans faire échouer l'endpoint.
// Le daemon est lu par petite interface (WatchedPlayers) plutôt que par son type
// concret : le watcher n'est pas atteignable ici autrement que derrière
// DaemonController, et l'annuaire n'a besoin que de cette question-là.
//
// Extrait de server_apiv1.go, assembleur déjà exempté du seuil de 500 lignes :
// on n'y ajoute pas d'adaptateurs.
package api

import (
	"levelup/go-api/internal/config"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/playerdirectory"
	"levelup/go-api/internal/watcher"
)

// buildPlayerDirectory assemble l'annuaire servi par GET /admin/identities.
func buildPlayerDirectory(
	cfg *config.AppConfig,
	users *userstore.Store,
	tokens *auth_platform.MultiUserTokenStore,
	daemon watcher.DaemonController,
) port.PlayerDirectory {
	deps := playerdirectory.Deps{}
	if cfg != nil {
		deps.Profiles = cfg
		deps.FS = playerdirectory.NewPathFS(cfg.RepoRoot)
	}
	if users != nil {
		deps.Accounts = users
	}
	if tokens != nil {
		deps.Tokens = tokens
	}
	// Le watcher peut être désactivé (daemon nil) ou n'être qu'un contrôleur de
	// test : sans WatchedPlayers, l'annuaire se lit sur les registres fichiers.
	if w, ok := daemon.(playerdirectory.WatchedReader); ok {
		deps.Watched = w
	}
	return playerdirectory.New(deps)
}
