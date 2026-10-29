// Package auth — xbl_doer.go : le `haloapi.Doer` gardé que la façade passe à la brique
// `xblauth` pour les surfaces sortantes qui étaient gardées par `netguard` avant le déplacement
// (plan D3 lot 2, décision D3-K) : rafraîchissement OAuth, démarrage et polling du device code.
//
// Les surfaces qui ne l'étaient pas (échange XBL/XSTS/Spartan, code d'autorisation, PeopleHub,
// profil Xbox) reçoivent le client nu, comme avant : leur exception est tenue par l'allowlist de
// `netguard_coverage_test.go`, au chemin du fichier de la brique qui émet.
package auth

import (
	"net/http"

	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/platform/netguard"
)

// netguardDoer appelle `netguard.Check` (mode démo) AVANT d'émettre par `next`. Une sortie
// refusée rend `netguard.ErrOffline` marquée définitive (`haloapi.NonRetryable`) : la brique la
// rend telle quelle, sans reprise ; `errors.Is(err, netguard.ErrOffline)` reste vrai.
type netguardDoer struct {
	// surface nomme le point de sortie dans le journal de netguard (une ligne par surface).
	surface string
	// next émet la requête ; il porte le délai.
	next *http.Client
}

// Do vérifie la garde puis émet.
func (d netguardDoer) Do(req *http.Request) (*http.Response, error) {
	if err := netguard.Check(req.Context(), d.surface); err != nil {
		return nil, haloapi.NonRetryable(err)
	}
	return d.next.Do(req)
}
