// Package sync — pooled_client_metrics.go : observabilité des appels API
// Halo du client poolé (dashboard monitoring P4).
//
// Chaque appel alimente :
//   - halo_api_ms_{call} (agrégat count/sum/avg/max — latence par type d'appel)
//   - halo_api_err_{call}_total (compteur d'erreurs par type d'appel)
//   - buckets globaux par classe : halo_api_429_total, halo_api_auth_total
//     (401/403), halo_api_5xx_total, halo_api_network_total (non-HTTP :
//     réseau/timeout/ctx), halo_api_other_total
//
// Observability-only : aucune influence sur le flux d'appels ni le pool.
package sync

import (
	"errors"
	"time"

	"levelup/go-api/internal/observability"
)

// haloAPICallNames : appels instrumentés (ordre d'affichage dashboard).
var haloAPICallNames = []string{
	"match_history",
	"match_stats",
	"match_skill",
	"film",
	"film_chunk",
	"career_rank",
	"player_csrs",
	"playlist_csr",
}

// HaloAPICallNames retourne la liste ordonnée des appels instrumentés
// (lecture des agrégats expvar par le registry monitoring).
func HaloAPICallNames() []string {
	out := make([]string, len(haloAPICallNames))
	copy(out, haloAPICallNames)
	return out
}

// observeHaloCall enregistre latence + classe d'erreur d'un appel API Halo.
// À appeler en fin de méthode du client poolé (start capturé en tête).
func observeHaloCall(call string, start time.Time, err error) {
	observability.RecordDurationMS("halo_api_ms_"+call, time.Since(start).Milliseconds())
	if err == nil {
		return
	}
	observability.IncCounter("halo_api_err_" + call + "_total")
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		observability.IncCounter("halo_api_network_total")
		return
	}
	switch {
	case httpErr.StatusCode == 429:
		observability.IncCounter("halo_api_429_total")
	case httpErr.StatusCode == 401 || httpErr.StatusCode == 403:
		observability.IncCounter("halo_api_auth_total")
	case httpErr.StatusCode >= 500:
		observability.IncCounter("halo_api_5xx_total")
	default:
		observability.IncCounter("halo_api_other_total")
	}
}
