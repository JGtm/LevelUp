package haloapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Retry : la politique de reprise d'un client de la brique, choisie par l'appelant à la
// construction. Les statuts 401/403 et 404/410 et une erreur [ErrDefinitif] ne sont JAMAIS
// retentés ; une erreur de transport (requête en échec, corps coupé) l'est toujours, dans la limite
// des essais.
//
// Les politiques nommées vivent dans les sous-paquets (`infinite.RetryProvider`,
// `infinite.RetryDiscovery`, `infinite.RetryHaloClient`, `halo5.RetryHalo5`) ; elles diffèrent à
// dessein (statuts retentés, 429, Retry-After, attente finale, limiteur) et ne convergent pas : en
// faire une seule serait un changement de comportement. L'appelant choisit par service.
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

// RetenteStatut applique [Retry.Retente] (nil : aucun statut retenté). La méthode ne peut pas
// porter le nom du champ.
func (r Retry) RetenteStatut(status int) bool {
	return r.Retente != nil && r.Retente(status)
}

// Delai rend l'attente après l'échec de l'essai n.
func (r Retry) Delai(n int, retryAfter time.Duration) time.Duration {
	d := r.DelaiBase * (1 << n)
	if r.HonoreRetryAfter && retryAfter > d {
		d = retryAfter
	}
	if r.Plafond > 0 && d > r.Plafond {
		d = r.Plafond
	}
	return d
}

// Attendre attend avant l'essai qui suit l'échec de l'essai n. Rend false quand la boucle doit
// s'arrêter sur la fin du contexte ([Retry.ArretSurContexte]).
func (r Retry) Attendre(ctx context.Context, n int, retryAfter time.Duration) bool {
	if n == r.Essais-1 && !r.AttenteFinale {
		return true
	}
	select {
	case <-ctx.Done():
		return !r.ArretSurContexte
	case <-time.After(r.Delai(n, retryAfter)):
		return true
	}
}

// Limiter borne le débit des requêtes. `*rate.Limiter` (`golang.org/x/time/rate`) la satisfait ;
// le limiteur appartient à l'appelant (un limiteur partagé entre clients borne leur débit commun).
type Limiter interface {
	Wait(ctx context.Context) error
}

// maxRetryAfter borne la valeur de l'en-tête Retry-After : une valeur aberrante ne fige pas
// l'appelant indéfiniment.
const maxRetryAfter = 5 * time.Minute

// ParseRetryAfter lit l'en-tête Retry-After (RFC 7231) : un entier de secondes ou une date HTTP.
// Rend une durée dans [0, maxRetryAfter] ; 0 si absent ou illisible.
func ParseRetryAfter(h string, now time.Time) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil {
		if secs <= 0 {
			return 0
		}
		if d := time.Duration(secs) * time.Second; d <= maxRetryAfter {
			return d
		}
		return maxRetryAfter
	}
	if t, err := http.ParseTime(h); err == nil {
		d := t.Sub(now)
		if d <= 0 {
			return 0
		}
		return min(d, maxRetryAfter)
	}
	return 0
}
