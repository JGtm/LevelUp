package netguard

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/JGtm/LevelUp/libs/haloapi"
)

// Doer est le `haloapi.Doer` GARDÉ que LevelUp passe aux clients de la brique `libs/haloapi` :
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

// entetesAuthentification liste les en-têtes qui portent une identité (jeton Spartan, clearance,
// autorisation) et que `libs/haloapi` pose sur ses requêtes. Contrat : le [Doer] les retire de toute
// requête de redirection dont l'hôte (nom et port) diffère de celui de la requête d'origine ; sur le
// même hôte ils sont conservés. `net/http` ne retire seul que `Authorization` et `Cookie` ; l'en-tête
// Spartan et la clearance, propres à Halo, ne partiraient sinon vers le CDN d'images ou tout autre
// hôte tiers visé par un 302. Tout nouvel en-tête d'identité posé par la brique s'ajoute ici.
var entetesAuthentification = []string{
	"Authorization",
	"X-343-Authorization-Spartan",
	"343-Clearance",
}

// maxRedirections reprend la limite par défaut de `net/http` quand `Next` n'a pas de `CheckRedirect`.
const maxRedirections = 10

// Do vérifie la garde puis émet (chaque requête, reprises comprises, repasse par la garde). Les
// redirections suivies par `Next` passent par [Doer.verifieRedirection].
func (d Doer) Do(req *http.Request) (*http.Response, error) {
	if err := d.Verifie(req.Context()); err != nil {
		return nil, err
	}
	// Copie du client (jamais modifié en place : `Next` peut être partagé) dont seul `CheckRedirect`
	// change ; le `CheckRedirect` d'origine, s'il existe, est appelé ensuite.
	c := *d.Next
	c.CheckRedirect = d.verifieRedirection
	return c.Do(req)
}

// verifieRedirection est le `CheckRedirect` du [Doer] : il retire les en-têtes d'identité d'une
// redirection vers un autre hôte que la requête d'origine (`via[0]`), puis applique la politique de
// `Next` (celle de `net/http` : dix redirections au plus, quand `Next` n'en a pas).
func (d Doer) verifieRedirection(req *http.Request, via []*http.Request) error {
	if len(via) > 0 && !strings.EqualFold(req.URL.Host, via[0].URL.Host) {
		for _, nom := range entetesAuthentification {
			req.Header.Del(nom)
		}
	}
	if d.Next.CheckRedirect != nil {
		return d.Next.CheckRedirect(req, via)
	}
	if len(via) >= maxRedirections {
		return errors.New("netguard: arrêt après 10 redirections")
	}
	return nil
}
