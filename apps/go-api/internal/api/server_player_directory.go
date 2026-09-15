// Package api — server_player_directory.go : assemblage de l'annuaire des
// joueurs (ADR 0035 D2).
//
// Wiring pur (aucune décision métier) : il branche les sources sur ce que le
// process possède déjà — les profils (`config.AppConfig`), les comptes
// (`userstore.Store`), les credentials (`MultiUserTokenStore`, ADR 0023), le
// suivi live (le daemon watcher), le disque (`PathResolver`) et, pour l'écriture,
// le créateur de profil (`service.ProfileService`, writer unique de
// `db_profiles.json`).
//
// Une source absente retire sa part de la réponse sans faire échouer l'endpoint.
// Le daemon est lu par petites interfaces (WatchedPlayers d'un côté, IsRunning +
// AddPlayer de l'autre) plutôt que par son type concret : le watcher n'est pas
// atteignable ici autrement que derrière DaemonController, et l'annuaire n'a
// besoin que de ces questions-là.
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

// playerDirectoryDeps regroupe ce que l'annuaire compose. Une structure plutôt
// qu'une liste de paramètres : elle a dépassé le seuil de 5 en gagnant le
// créateur de profil (CLAUDE.md règle 5).
type playerDirectoryDeps struct {
	cfg     *config.AppConfig
	users   *userstore.Store
	tokens  *auth_platform.MultiUserTokenStore
	daemon  watcher.DaemonController
	creator playerdirectory.ProfileCreator
}

// buildPlayerDirectory assemble l'annuaire : il sert GET /admin/identities en
// lecture et POST /setup/players en écriture (Onboard, ADR 0035 D4). Une SEULE
// instance pour les deux — le créateur de profil qu'elle porte est le writer
// unique de db_profiles.json.
func buildPlayerDirectory(d playerDirectoryDeps) port.PlayerDirectory {
	deps := playerdirectory.Deps{Creator: d.creator}
	if d.cfg != nil {
		deps.Profiles = d.cfg
		deps.FS = playerdirectory.NewPathFS(d.cfg.RepoRoot)
	}
	if d.users != nil {
		deps.Accounts = d.users
	}
	if d.tokens != nil {
		deps.Tokens = d.tokens
	}
	// Le watcher peut être désactivé (daemon nil) ou n'être qu'un contrôleur de
	// test : sans WatchedPlayers, l'annuaire se lit sur les registres fichiers.
	if w, ok := d.daemon.(playerdirectory.WatchedReader); ok {
		deps.Watched = w
	}
	// DaemonController porte déjà IsRunning + AddPlayer : la notification du
	// suivi live après création de profil n'a besoin de rien de plus. Interface
	// nil-safe côté annuaire, mais une interface NON NIL portant un pointeur nil
	// ne l'est pas — d'où le test explicite.
	if d.daemon != nil {
		deps.Watcher = d.daemon
	}
	return playerdirectory.New(deps)
}
