package infinite

import (
	"context"
	"net/http"
	"time"
)

// Retry : la politique de reprise d'un [Client], choisie par l'appelant à la construction. Les
// statuts 401/403 et 404/410 et une erreur [haloapi.ErrDefinitif] ne sont JAMAIS retentés ; une
// erreur de transport (requête en échec, corps coupé) l'est toujours, dans la limite des essais.
//
// Les trois politiques nommées ([RetryProvider], [RetryDiscovery], [RetryHaloClient]) sont celles
// que LevelUp emploie ; elles diffèrent à dessein (statuts retentés, 429, Retry-After, attente
// finale, limiteur) et ne convergent pas : en faire une seule serait un changement de comportement.
// L'appelant choisit par service.
type Retry struct {
	// Essais : nombre maximal de requêtes envoyées.
	Essais int
	// Retente dit si un statut non-200 (hors 401/403/404/410) est retenté ; nil : aucun.
	Retente func(status int) bool
	// HonoreRetryAfter : l'attente avant l'essai suivant vaut au moins la fenêtre Retry-After.
	HonoreRetryAfter bool
	// DelaiBase : attente après l'échec de l'essai n (compté depuis 0) = DelaiBase·2^n.
	DelaiBase time.Duration
	// Plafond : borne de l'attente (Retry-After compris) ; 0 = sans borne.
	Plafond time.Duration
	// AttenteFinale : attendre aussi après l'échec du DERNIER essai (RetryProvider et RetryHaloClient).
	AttenteFinale bool
	// ArretSurContexte : une fin de contexte pendant une attente rend aussitôt `ctx.Err()` ; sinon la
	// boucle poursuit et les essais suivants échouent sur le contexte.
	ArretSurContexte bool
	// LimiteurUneFois : le limiteur est attendu une seule fois, avant le premier essai ; sinon avant
	// chaque essai.
	LimiteurUneFois bool
	// IgnoreErreurLimiteur : une erreur du limiteur n'arrête pas la requête ; sinon elle est rendue.
	IgnoreErreurLimiteur bool
}

// Délais communs aux trois politiques.
const (
	delaiBaseReprise = 800 * time.Millisecond
	plafondReprise   = 10 * time.Second
	essaisProvider   = 3
	essaisDiscovery  = 3
	essaisHaloClient = 4
	premierStatut5xx = http.StatusInternalServerError
)

// RetryProvider : la politique des services du fournisseur (passe de combat, défis, service
// record, confidentialité, médailles, saisons, `FetchMatchStats`). 3 essais ; TOUT statut
// non-200 hors 401/403/404/410 est retenté, 429 compris ; Retry-After ignoré ; attente
// 800 ms·2^n bornée à 10 s, y compris après le dernier essai ; limiteur avant chaque essai, son
// erreur arrête.
func RetryProvider() Retry {
	return Retry{
		Essais:        essaisProvider,
		Retente:       func(int) bool { return true },
		DelaiBase:     delaiBaseReprise,
		Plafond:       plafondReprise,
		AttenteFinale: true,
	}
}

// RetryDiscovery : la politique des actifs Discovery UGC. 3 essais ;
// seuls les 5xx sont retentés (429 et tout autre 4xx sont rendus) ; Retry-After ignoré ; attente
// 800 ms puis 1,6 s entre les essais, aucune après le dernier ; une fin de contexte pendant
// l'attente rend `ctx.Err()` ; le limiteur n'est attendu qu'une fois, son erreur arrête.
func RetryDiscovery() Retry {
	return Retry{
		Essais:           essaisDiscovery,
		Retente:          func(s int) bool { return s >= premierStatut5xx },
		DelaiBase:        delaiBaseReprise,
		ArretSurContexte: true,
		LimiteurUneFois:  true,
	}
}

// RetryHaloClient : la politique du client de synchronisation (historique, stats, skill, CSR,
// carrière, apparence, playlists, manifestes de film ; ses essais et délais servent aussi aux blobs
// du CDN).
// 4 essais ; 429 est rendu IMMÉDIATEMENT (avec sa fenêtre Retry-After : le pool de l'appelant gère
// le refroidissement) ; tout autre statut non-200 est retenté ; attente 800 ms·2^n, au moins la
// fenêtre Retry-After, bornée à 10 s, y compris après le dernier essai ; limiteur avant chaque
// essai, son erreur ignorée.
func RetryHaloClient() Retry {
	return Retry{
		Essais:               essaisHaloClient,
		Retente:              func(s int) bool { return s != http.StatusTooManyRequests },
		HonoreRetryAfter:     true,
		DelaiBase:            delaiBaseReprise,
		Plafond:              plafondReprise,
		AttenteFinale:        true,
		IgnoreErreurLimiteur: true,
	}
}

// retente applique [Retry.Retente] (nil : aucun statut retenté).
func (r Retry) retente(status int) bool {
	return r.Retente != nil && r.Retente(status)
}

// delai rend l'attente après l'échec de l'essai n.
func (r Retry) delai(n int, retryAfter time.Duration) time.Duration {
	d := r.DelaiBase * (1 << n)
	if r.HonoreRetryAfter && retryAfter > d {
		d = retryAfter
	}
	if r.Plafond > 0 && d > r.Plafond {
		d = r.Plafond
	}
	return d
}

// attendre attend avant l'essai qui suit l'échec de l'essai n. Rend false quand la boucle doit
// s'arrêter sur la fin du contexte ([Retry.ArretSurContexte]).
func (r Retry) attendre(ctx context.Context, n int, retryAfter time.Duration) bool {
	if n == r.Essais-1 && !r.AttenteFinale {
		return true
	}
	select {
	case <-ctx.Done():
		return !r.ArretSurContexte
	case <-time.After(r.delai(n, retryAfter)):
		return true
	}
}
