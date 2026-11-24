package livesync

// resolver_retry.go — la résolution gamertag -> xuid Halo 5 patiente sous la limite de débit de Xbox Live
// au lieu d'abandonner.
//
// Constat (mesure d'une passe de rattrapage de 51 matchs) : PeopleHub accepte environ 10 requêtes par 15 s
// et par compte ; une passe qui résout des centaines de gamertags prenait des 429, et un gamertag non
// résolu fait SAUTER son participant du match (resolve-or-skip, halo5.mapCarnageParticipants) : le match
// est rangé sans lui, puis jamais revu par le delta (il est connu). Deux protections, sans toucher la
// brique halo-api (dont l'erreur ne porte ni statut typé ni en-tête Retry-After : le texte « peoplehub
// HTTP 429 » est la seule trace) :
//
//   - cadence : chaque résolveur de compte espace ses requêtes de resolveMinGap, sous la limite observée ;
//   - patience : sur 429 de tous les comptes, la résolution attend (recul doublé de resolveBackoffFirst à
//     resolveBackoffMax) et réessaie, sans dépasser resolveRetryBudget d'attente cumulée par passe. Passé ce
//     budget, elle rend l'erreur comme avant : l'identité reste portée par le gamertag.
//
// Les deux sont comptés (h5_xuid_resolve_*) et journalisés.

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/observability"
)

const (
	// resolveMinGap espace deux requêtes d'un même compte : 15 s / 9, soit 9 requêtes par fenêtre de 15 s,
	// sous les 10 observées.
	resolveMinGap = 15 * time.Second / 9
	// resolveBackoffFirst et resolveBackoffMax : premier recul sur 429 et plafond du recul (la fenêtre de la limite).
	resolveBackoffFirst = 3 * time.Second
	resolveBackoffMax   = 15 * time.Second
	// resolveRetryBudget borne l'attente cumulée sur 429 d'une passe (un RunDelta) : au-delà, la résolution
	// abandonne comme avant, pour que la passe ne reste pas bloquée sur un service durablement saturé.
	resolveRetryBudget = 2 * time.Minute

	metricResolveRetry  = "h5_xuid_resolve_retry_total"
	metricResolveGiveUp = "h5_xuid_resolve_giveup_total"
)

// resolveSleep attend d (annulable) ; les tests le remplacent pour ne pas dormir. resolveNow est l'horloge
// de la cadence.
var (
	resolveSleep = func(ctx context.Context, d time.Duration) error {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	resolveNow = time.Now
)

// isRateLimitedErr dit si l'erreur d'un résolveur est une limitation de débit (429). Le message est la
// seule trace : la brique rend « peoplehub HTTP 429 », sans type (même lecture que worldenrich).
func isRateLimitedErr(err error) bool { return err != nil && strings.Contains(err.Error(), "429") }

// pacedResolver espace les requêtes d'un résolveur (un compte) de resolveMinGap.
type pacedResolver struct {
	inner XUIDResolver
	mu    sync.Mutex
	next  time.Time // instant avant lequel la prochaine requête ne part pas
}

// newPacedResolver enveloppe un résolveur de compte.
func newPacedResolver(inner XUIDResolver) *pacedResolver { return &pacedResolver{inner: inner} }

func (p *pacedResolver) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	p.mu.Lock()
	now := resolveNow()
	wait := time.Duration(0)
	if p.next.After(now) {
		wait = p.next.Sub(now)
	}
	p.next = now.Add(wait + resolveMinGap)
	p.mu.Unlock()
	if wait > 0 {
		if err := resolveSleep(ctx, wait); err != nil {
			return "", err
		}
	}
	return p.inner.ResolveXUID(ctx, gamertag)
}

// retryingResolver réessaie une résolution limitée en débit (429) après un recul, dans la limite d'une
// attente cumulée resolveRetryBudget pour la passe.
type retryingResolver struct {
	inner  XUIDResolver
	mu     sync.Mutex
	waited time.Duration
}

func newRetryingResolver(inner XUIDResolver) *retryingResolver {
	return &retryingResolver{inner: inner}
}

func (r *retryingResolver) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	backoff := resolveBackoffFirst
	for {
		xuid, err := r.inner.ResolveXUID(ctx, gamertag)
		if !isRateLimitedErr(err) {
			return xuid, err
		}
		if !r.reserve(backoff) {
			observability.IncCounter(metricResolveGiveUp)
			slog.WarnContext(ctx, "h5 sync: résolution xuid abandonnée — attente cumulée sur 429 épuisée",
				"gamertag", gamertag, "budget", resolveRetryBudget)
			return "", err
		}
		observability.IncCounter(metricResolveRetry)
		slog.InfoContext(ctx, "h5 sync: résolution xuid limitée en débit (429) — attente puis nouvel essai",
			"gamertag", gamertag, "wait", backoff)
		if serr := resolveSleep(ctx, backoff); serr != nil {
			return "", serr
		}
		backoff = min(backoff*2, resolveBackoffMax)
	}
}

// reserve ajoute d à l'attente cumulée de la passe et rend vrai si elle reste dans resolveRetryBudget.
func (r *retryingResolver) reserve(d time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.waited+d > resolveRetryBudget {
		return false
	}
	r.waited += d
	return true
}
