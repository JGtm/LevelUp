// Package handlers — sync_job_errors.go : traduction des refus typés des jobs de sync
// (port.SyncJobError) en réponses HTTP, pour les routes de sync, de rattrapage et de recalcul des
// sessions.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/port"
)

// Codes d'erreur des jobs de sync.
const (
	errCodeSyncAlreadyActive = "sync_already_active"
	errCodeAuthRequired      = "auth_required"
	errCodePlayerNotFound    = "player_not_found"
	// errCodeHaloTokensMissing : session authentifiée mais sans jeton Halo. Statut 403, jamais 401 : un
	// 401 `auth_required` signifie « session expirée » pour la coquille web, qui recharge alors la page
	// vers la connexion — une session valide sans jetons y bouclerait.
	errCodeHaloTokensMissing = "halo_tokens_missing"
)

// syncJobHTTPError traduit un refus d'un job de sync en réponse HTTP : statut, code et message par
// nature de refus. Une erreur qui n'est pas un port.SyncJobError est une panne : 500.
func syncJobHTTPError(ctx context.Context, err error) error {
	var je *port.SyncJobError
	if !errors.As(err, &je) {
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "sync_failed", err)
	}
	switch je.Kind {
	case port.SyncErrSettingsLoad:
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "settings_load_error", je.Err)
	case port.SyncErrProfilesLoad:
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "profiles_load_error", je.Err)
	case port.SyncErrPlayersLoad:
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "players_load_error", je.Err)
	case port.SyncErrCooldown:
		return cooldownError(je.RetryAfter)
	case port.SyncErrPlayerNotFound:
		return humacore.NewError(http.StatusNotFound, errCodePlayerNotFound,
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", je.PlayerSlug))
	case port.SyncErrPlayerNotFoundForTitle:
		return humacore.NewError(http.StatusNotFound, errCodePlayerNotFound,
			fmt.Sprintf("Joueur %q introuvable pour le titre %q dans db_profiles.json.", je.PlayerSlug, je.TitleSlug))
	}
	if r, ok := syncJobRefusals[je.Kind]; ok {
		return humacore.NewError(r.status, r.code, r.message)
	}
	return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "sync_failed", je)
}

// syncJobRefusal : la réponse de client d'un refus dont le message est fixe.
type syncJobRefusal struct {
	status  int
	code    string
	message string
}

// syncJobRefusals : statut, code et message de chaque refus de client à message fixe.
var syncJobRefusals = map[port.SyncJobErrorKind]syncJobRefusal{
	port.SyncErrInitialDisabled: {http.StatusForbidden, "initial_sync_disabled",
		"Le lancement d'une sync initiale est désactivé sur cette instance."},
	port.SyncErrInvalidMaxMatches: {http.StatusBadRequest, "invalid_max_matches",
		"max_matches doit etre entre 1 et 2000."},
	port.SyncErrInitialAlreadyActive: {http.StatusConflict, errCodeSyncAlreadyActive,
		"Une sync initiale est deja en cours pour ce joueur."},
	port.SyncErrDeltaAlreadyActive: {http.StatusConflict, errCodeSyncAlreadyActive,
		"Une synchronisation est déjà en cours pour ce joueur."},
	port.SyncErrAlreadyRunning: {http.StatusConflict, errCodeSyncAlreadyActive,
		"Une synchronisation de ce joueur est déjà en cours (watcher ou auto-sync)."},
	port.SyncErrBackfillAlreadyActive: {http.StatusConflict, "backfill_already_active",
		"Un backfill est déjà en cours pour ce joueur."},
	port.SyncErrAuthRequired: {http.StatusUnauthorized, errCodeAuthRequired, "Connexion requise."},
	port.SyncErrHaloTokensMissing: {http.StatusForbidden, errCodeHaloTokensMissing,
		"Aucun jeton Halo pour cette session : connecte le compte Xbox."},
	port.SyncErrNoPlayers: {http.StatusNotFound, "no_players", "Aucun joueur configuré dans db_profiles.json."},
}

// cooldownError construit l'erreur 429 sync_cooldown, avec un en-tête Retry-After (secondes, arrondi au
// plafond).
func cooldownError(retry time.Duration) error {
	secs := int(retry.Seconds()) + 1
	return huma.ErrorWithHeaders(
		humacore.NewError(http.StatusTooManyRequests, "sync_cooldown",
			fmt.Sprintf("Synchronisation déjà déclenchée récemment — réessayez dans %d s.", secs)),
		http.Header{headerRetryAfter: []string{strconv.Itoa(secs)}},
	)
}
