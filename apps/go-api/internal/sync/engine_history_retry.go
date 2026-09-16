// Package sync — engine_history_retry.go : rejeu borné d'une page d'historique.
//
// CONSTAT FONDATEUR (première passe réelle par le pool, 2026-09-16, 6 slots x 3 req/s).
// Un SEUL `HTTP 429` sur `GetMatchHistory(start=225)` a terminé toute la passe : le pool
// met le slot fautif en cooldown AIMD, `paginateAndPersistHistory` faisait `break` sur la
// première erreur d'historique, et la passe se déclarait `status=success inserted=0`. Le
// client HTTP ne retente PAS un 429 (`halo_client_http.go` : re-taper l'API sous 10 s quand
// on est rate-limité n'ajoute que des 429) et compte sur le caller — qui ne retentait pas.
//
// Politique (D1, plan `.ai/PLAN_ROBUSTESSE_SYNC_2026-09-16.md`) :
//   - 429 : rejeu IMMÉDIAT de la même page. L'acquisition suivante saute le slot en
//     cooldown et sert un autre token ; avec un seul slot, le rejeu retombe sur le cas
//     suivant (plus aucun slot sain).
//   - `pool.ErrNoHealthySlot` : tout le parc est en pause — attendre le cooldown du pool,
//     borné à historyNoSlotWaitCap, puis rejouer.
//   - toute AUTRE erreur (dont 503, DÉJÀ retenté en interne par `doGet`) : retour immédiat,
//     zéro rejeu ici — un second étage de rejeu sur un 503 doublerait la charge d'une API
//     déjà dégradée.
//
// Trois tentatives au total ; l'échec définitif remonte à l'appelant, qui le compte en
// `Errors` (et non plus en `Warnings`) : `SyncResult.Status()` rend alors `partial_success`
// ou `failure`, la CLI un code de sortie non nul.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/auth/pool"
)

// historyRetrySleep — seam de paquet pour l'attente entre deux tentatives. Les tests le
// remplacent par un enregistreur de durées : aucune suite ne doit dormir pour de vrai.
var historyRetrySleep = time.Sleep

const (
	// historyFetchAttempts : nombre TOTAL d'appels à GetMatchHistory pour une page.
	historyFetchAttempts = 3

	// historyNoSlotCooldown reprend le défaut de PoolOptions.GlobalCooldown (30 s,
	// `pool.go`). L'interface `pool.Pool` n'expose pas la valeur configurée : le client
	// poolé ne peut pas la lire, d'où cette constante nommée plutôt qu'un nombre nu.
	historyNoSlotCooldown = 30 * time.Second

	// historyNoSlotWaitCap borne l'attente : au-delà d'une minute, mieux vaut rendre la
	// main avec un verdict honnête que tenir la base partagée en écriture pour rien.
	historyNoSlotWaitCap = 60 * time.Second
)

// historyNoSlotWait rend l'attente appliquée quand le parc entier est en pause.
func historyNoSlotWait() time.Duration {
	return min(historyNoSlotCooldown, historyNoSlotWaitCap)
}

// isHistoryRateLimited indique si err est un 429 du client Halo (rate limit), le seul
// statut rejoué immédiatement par fetchHistoryPage.
func isHistoryRateLimited(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == 429
}

// fetchHistoryPage récupère UNE page d'historique avec le rejeu borné décrit en tête de
// fichier. Rend la dernière erreur rencontrée quand les tentatives sont épuisées.
//
// L'endpoint /hi/players/{player}/matches exige strictement le format xuid(NNN) (Grunt
// StatsModule.GetMatchHistory + SPNKr) : passer le gamertag rend une réponse stale figée
// (incident du 2026-05-20).
func (e *SyncEngine) fetchHistoryPage(
	ctx context.Context,
	in *historyPaginationInputs,
	start int,
) ([]MatchHistoryEntry, error) {
	var lastErr error
	for attempt := 1; attempt <= historyFetchAttempts; attempt++ {
		entries, err := in.client.GetMatchHistory(
			ctx, fmt.Sprintf("xuid(%s)", e.xuid), in.opts.MatchType, start, historyPageSize,
		)
		if err == nil {
			return entries, nil
		}
		lastErr = err

		rateLimited := isHistoryRateLimited(err)
		noSlot := errors.Is(err, pool.ErrNoHealthySlot)
		if !rateLimited && !noSlot {
			// 503 inclus : déjà retenté par le client HTTP, on ne redouble pas.
			return nil, err
		}
		if attempt == historyFetchAttempts {
			break
		}

		cause := "429"
		wait := time.Duration(0)
		if noSlot {
			cause = "pool sans slot sain"
			wait = historyNoSlotWait()
		}
		slog.WarnContext(ctx, "sync: page d'historique rejouée",
			"gamertag", e.gamertag, "start", start, "attempt", attempt,
			"cause", cause, "wait", wait, "err", err,
		)
		if wait > 0 {
			historyRetrySleep(wait)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
	}
	return nil, lastErr
}
