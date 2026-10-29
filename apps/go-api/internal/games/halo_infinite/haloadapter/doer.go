package haloadapter

import (
	"net/http"

	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/platform/netguard"
)

// Les surfaces sortantes vers Halo, une par point de sortie : chacune a son `haloapi.Doer` gardé
// et sa ligne dans le journal de `netguard` (une par surface et par processus).
const (
	surfaceAPIGet           = "halo_api.get"
	surfaceAPIBlob          = "halo_api.download_blob"
	surfaceProviderGet      = "halo_provider.get"
	surfaceDiscoveryGet     = "halo_discovery.get"
	surfaceNameplateMapping = "nameplate_resolver.mapping"
	surfaceNameplateEmblem  = "nameplate_resolver.emblem"
	surfaceClassementPage   = "waypoint_leaderboard.scrape"
	surfaceClassementSaison = "waypoint_leaderboard.discover_season"
)

// netguardDoer appelle `netguard.Check` (mode démo) AVANT d'émettre par `next`. Une sortie
// refusée rend `netguard.ErrOffline` marquée définitive (`haloapi.NonRetryable`) : la brique la
// rend aussitôt, sans reprise ni attente ; `errors.Is(err, netguard.ErrOffline)` reste vrai.
type netguardDoer struct {
	// surface nomme le point de sortie dans le journal de netguard.
	surface string
	// next émet la requête ; il porte le délai (relu à chaque appel : un `*http.Client` partagé
	// dont on change le transport est vu).
	next *http.Client
}

// Do vérifie la garde puis émet.
func (d netguardDoer) Do(req *http.Request) (*http.Response, error) {
	if err := netguard.Check(req.Context(), d.surface); err != nil {
		return nil, haloapi.NonRetryable(err)
	}
	return d.next.Do(req)
}
