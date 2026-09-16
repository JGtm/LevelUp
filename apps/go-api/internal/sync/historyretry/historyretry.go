// Package historyretry — rejeu borné d'une page d'historique de matchs.
//
// CONSTAT FONDATEUR (première passe réelle par le pool, 2026-09-16, 6 slots x 3 req/s).
// Un SEUL `HTTP 429` sur `GetMatchHistory(start=225)` a terminé toute la passe : le pool met
// le slot fautif en cooldown AIMD, la pagination faisait `break` sur la première erreur, et la
// passe se déclarait `status=success inserted=0`. Le client HTTP ne retente PAS un 429
// (re-taper l'API sous 10 s quand on est rate-limité n'ajoute que des 429) et compte sur le
// caller — qui ne retentait pas.
//
// POLITIQUE (D1, plan `.ai/PLAN_ROBUSTESSE_SYNC_2026-09-16.md`) :
//   - 429 : rejeu IMMÉDIAT de la même page. L'acquisition suivante saute le slot en cooldown
//     et sert un autre token ; avec un seul slot, le rejeu retombe sur le cas suivant.
//   - `pool.ErrNoHealthySlot` : tout le parc est en pause — attendre le cooldown du pool, borné
//     à WaitCap, puis rejouer.
//   - toute AUTRE erreur (dont 503, DÉJÀ retenté en interne par le client HTTP) : retour
//     immédiat, zéro rejeu ici — un second étage de rejeu doublerait la charge d'une API déjà
//     dégradée.
//
// Trois tentatives au total ; l'échec définitif remonte à l'appelant, qui le compte en
// `Errors` (et non en `Warnings`) : le statut de la passe devient `partial_success` ou
// `failure`, et la CLI sort en code non nul.
//
// Paquet à part (ADR 0027 / ratchet K3c `sync_root_freeze_test.go`) : `internal/sync` est un
// god-package gelé, le neuf va dans un sous-paquet cohésif.
package historyretry

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/sync/haloclient"
)

// Sleep — seam de paquet pour l'attente entre deux tentatives. Les tests le remplacent par un
// enregistreur de durées : aucune suite ne doit dormir pour de vrai.
var Sleep = time.Sleep

const (
	// Attempts : nombre TOTAL d'appels pour une page.
	Attempts = 3

	// NoSlotCooldown reprend le défaut de `pool.PoolOptions.GlobalCooldown` (30 s).
	// L'interface `pool.Pool` n'expose pas la valeur configurée : le client poolé ne peut pas
	// la lire, d'où cette constante nommée plutôt qu'un nombre nu.
	NoSlotCooldown = 30 * time.Second

	// WaitCap borne l'attente : au-delà d'une minute, mieux vaut rendre la main avec un verdict
	// honnête que tenir la base partagée en écriture pour rien.
	WaitCap = 60 * time.Second
)

// NoSlotWait rend l'attente appliquée quand le parc entier est en pause.
func NoSlotWait() time.Duration {
	return min(NoSlotCooldown, WaitCap)
}

// IsRateLimited indique si err est un 429 du client Halo, le seul statut rejoué immédiatement.
func IsRateLimited(err error) bool {
	var httpErr *haloclient.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == 429
}

// Page exécute `fetch` avec le rejeu borné décrit en tête de fichier. `gamertag` et `start` ne
// servent qu'au journal. Rend la dernière erreur rencontrée quand les tentatives sont épuisées.
func Page[T any](
	ctx context.Context,
	gamertag string,
	start int,
	fetch func() ([]T, error),
) ([]T, error) {
	var lastErr error
	for attempt := 1; attempt <= Attempts; attempt++ {
		entries, err := fetch()
		if err == nil {
			return entries, nil
		}
		lastErr = err

		rateLimited := IsRateLimited(err)
		noSlot := errors.Is(err, pool.ErrNoHealthySlot)
		if !rateLimited && !noSlot {
			// 503 inclus : déjà retenté par le client HTTP, on ne redouble pas.
			return nil, err
		}
		if attempt == Attempts {
			break
		}

		cause := "429"
		wait := time.Duration(0)
		if noSlot {
			cause = "pool sans slot sain"
			wait = NoSlotWait()
		}
		slog.WarnContext(ctx, "sync: page d'historique rejouée",
			"gamertag", gamertag, "start", start, "attempt", attempt,
			"cause", cause, "wait", wait, "err", err,
		)
		if wait > 0 {
			Sleep(wait)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
	}
	return nil, lastErr
}
