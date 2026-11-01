package netguard

import (
	"context"
	"net/http"

	"levelup/go-api/internal/haloapi"
)

// Doer est le `haloapi.Doer` GARDÉ que LevelUp passe aux clients de la brique `internal/haloapi` :
// il appelle `netguard.Check` (mode démo) AVANT d'émettre par `Next`. Une sortie refusée rend
// [ErrOffline] marquée définitive (`haloapi.NonRetryable`) : la brique la rend aussitôt, sans
// reprise ni attente ; `errors.Is(err, netguard.ErrOffline)` reste vrai. Il implémente aussi
// `haloapi.Garde`, que la brique consulte avant tout limiteur.
//
// Un client de la brique construit par go-api reçoit ce type en littéral à l'appel de son
// constructeur (`archlint/haloapi_doer_garde_test.go`).
type Doer struct {
	// Surface nomme le point de sortie dans le journal de netguard.
	Surface string
	// Next émet la requête ; il porte le délai (relu à chaque appel : un `*http.Client` partagé
	// dont on change le transport est vu).
	Next *http.Client
}

// Verifie consulte la garde (`haloapi.Garde`) : la brique l'appelle au début de chaque appel, avant
// toute attente du limiteur, pour qu'une sortie refusée ne consomme aucun jeton.
func (d Doer) Verifie(ctx context.Context) error {
	if err := Check(ctx, d.Surface); err != nil {
		return haloapi.NonRetryable(err)
	}
	return nil
}

// Do vérifie la garde puis émet (chaque requête, reprises comprises, repasse par la garde).
func (d Doer) Do(req *http.Request) (*http.Response, error) {
	if err := d.Verifie(req.Context()); err != nil {
		return nil, err
	}
	return d.Next.Do(req)
}
