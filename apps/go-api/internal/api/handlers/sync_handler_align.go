// Package handlers — sync_handler_align.go : alignement du sync manuel DELTA sur
// l'auto-sync (pool de tokens partagé) + cooldown anti-spam.
//
// Extrait de sync_handler.go (qui dépassait 500 lignes) pour regrouper la logique
// « le sync manuel = un déclencheur à la demande de l'auto-sync » :
//   - EngineBuilder / newPooledEngine : MÊME moteur (PooledHaloClient du pool partagé)
//     que l'auto-sync — l'auth ne dépend plus des HaloTokens de session.
//   - cooldown : guardManualDeltaSync / tryManualSyncCooldown / writeCooldown.
//
// Cf. ADR 0023 (MultiUserTokenStore = source unique des tokens) + thought_log 2026-06-14.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	go_sync "levelup/go-api/internal/sync"
)

// defaultManualSyncCooldown borne la fréquence des syncs manuels delta (bouton
// « Synchroniser » de Settings → /sync/all, et /players/{slug}/sync). Choix
// utilisateur 2026-06-14 : 5 min — le sync manuel n'est qu'un déclencheur à la
// demande de l'auto-sync, inutile de marteler l'API Halo plus souvent. Une 2e
// demande dans la fenêtre répond 429 + Retry-After. N'affecte PAS le sync
// initial (onboarding). Override en test via WithManualSyncCooldown.
const defaultManualSyncCooldown = 5 * time.Minute

// EngineBuilder construit un *go_sync.SyncEngine fully-wired pour (gamertag, xuid).
//
// En production, server.go y injecte AutoSyncScheduler.BuildEngine — l'UNIQUE
// source de vérité du wiring moteur. Le sync manuel delta (StartSyncAll /
// StartDeltaSync) construit alors EXACTEMENT le même moteur que l'auto-sync :
// même PooledHaloClient (pool de tokens partagé, source unique ADR 0023), même
// post-sync runner, même batch queue. L'auth est entièrement déléguée au pool →
// le sync manuel ne dépend plus des HaloTokens de session (qui peuvent être
// nil/expirés alors que le store, lui, a un RT valide).
//
// Nil → fallback legacy newEngineFor (tests, ou bootstrap sans pool).
type EngineBuilder func(ctx context.Context, gamertag, xuid string) *go_sync.SyncEngine

// WithEngineBuilder branche le constructeur de moteur partagé avec l'auto-sync
// (AutoSyncScheduler.BuildEngine). À appeler depuis server.go. Une fois injecté,
// le sync manuel delta (StartSyncAll / StartDeltaSync) passe par le PooledHaloClient
// du pool de tokens au lieu des HaloTokens de session. Nil → fallback legacy.
func (h *SyncHandler) WithEngineBuilder(b EngineBuilder) *SyncHandler {
	h.engineBuilder = b
	return h
}

// WithManualSyncCooldown surcharge la fenêtre anti-spam du sync manuel delta.
// 0 désactive le cooldown (tests qui enchaînent plusieurs déclenchements).
func (h *SyncHandler) WithManualSyncCooldown(d time.Duration) *SyncHandler {
	h.cooldown = d
	return h
}

// guardManualDeltaSync applique les pré-conditions communes au sync manuel DELTA
// (StartSyncAll / StartDeltaSync) :
//  1. session présente (être connecté) — sinon 401 « Connexion requise ». L'auth
//     Halo elle-même est déléguée au pool (cf. newPooledEngine), pas à la session.
//  2. cooldown anti-spam par clé — sinon 429 + Retry-After.
//
// Retourne false si une réponse d'erreur a déjà été écrite (le caller doit return).
// Centralise la logique partagée par les deux endpoints (DRY + handlers < 80 lignes).
func (h *SyncHandler) guardManualDeltaSync(w http.ResponseWriter, r *http.Request, cooldownKey string) bool {
	if sess := middleware.GetSession(r.Context()); sess == nil {
		writeError(r.Context(), w, http.StatusUnauthorized, "auth_required", "Connexion requise.")
		return false
	}
	if retry, ok := h.tryManualSyncCooldown(cooldownKey); !ok {
		slog.InfoContext(r.Context(), "sync_handler: sync manuel throttlé par cooldown",
			"key", cooldownKey, "retry_after_s", int(retry.Seconds())+1)
		h.writeCooldown(r.Context(), w, retry)
		return false
	}
	return true
}

// tryManualSyncCooldown vérifie ET enregistre (atomiquement) le cooldown anti-spam
// du sync manuel delta pour `key`. Retourne (retryAfter>0, false) si un sync de la
// même clé a été déclenché il y a moins de h.cooldown — le caller répond 429. Sinon
// mémorise l'instant courant et retourne (0, true). Toujours (0, true) si le cooldown
// est désactivé (h.cooldown<=0). Le marquage est posé même si la suite échoue (claim
// refusé, joueur introuvable) : c'est volontaire — la fenêtre anti-spam couvre la
// TENTATIVE, pas seulement les syncs réellement exécutés.
func (h *SyncHandler) tryManualSyncCooldown(key string) (time.Duration, bool) {
	if h.cooldown <= 0 {
		return 0, true
	}
	h.cooldownMu.Lock()
	defer h.cooldownMu.Unlock()
	if last, ok := h.lastManualAt[key]; ok {
		if elapsed := time.Since(last); elapsed < h.cooldown {
			return h.cooldown - elapsed, false
		}
	}
	h.lastManualAt[key] = time.Now()
	return 0, true
}

// writeCooldown répond 429 avec un header Retry-After (secondes, arrondi au plafond).
func (h *SyncHandler) writeCooldown(ctx context.Context, w http.ResponseWriter, retry time.Duration) {
	secs := int(retry.Seconds()) + 1
	w.Header().Set("Retry-After", fmt.Sprintf("%d", secs))
	writeError(ctx, w, http.StatusTooManyRequests, "sync_cooldown",
		fmt.Sprintf("Synchronisation déjà déclenchée récemment — réessayez dans %d s.", secs))
}

// newPooledEngine construit le moteur du sync manuel DELTA (StartSyncAll /
// StartDeltaSync) via le MÊME wiring que l'auto-sync : engineBuilder =
// AutoSyncScheduler.BuildEngine → PooledHaloClient (pool de tokens partagé,
// source unique ADR 0023) + post-sync runner + batch queue. L'auth vient du pool,
// pas de la session. Fallback legacy (newEngineFor, tokens vides) si aucun builder
// n'est injecté (tests / bootstrap sans pool).
//
// À distinguer de StartInitialSync, qui reste sur les HaloTokens de session :
// l'onboarding cible un joueur fraîchement connecté, pas encore présent dans le
// pool (Discovery ne tourne qu'au boot) — le pool échouerait à résoudre ses tokens.
func (h *SyncHandler) newPooledEngine(ctx context.Context, gamertag, xuid string) *go_sync.SyncEngine {
	if h.engineBuilder != nil {
		slog.DebugContext(ctx, "sync_handler: moteur sync manuel construit via le pool partagé (aligné auto-sync)",
			"gamertag", gamertag, "xuid", xuid)
		return h.engineBuilder(ctx, gamertag, xuid)
	}
	// Garde-fou observable : en prod server.go injecte toujours le builder. Un
	// fallback ici signale un câblage manquant (le moteur legacy a des tokens vides
	// → l'auth échouera) — WARN pour le repérer dans logs/handlers.log.
	slog.WarnContext(ctx, "sync_handler: EngineBuilder absent — fallback moteur legacy (tokens vides, auth probablement en échec)",
		"gamertag", gamertag, "xuid", xuid)
	return h.newEngineFor(gamertag, xuid, &domain.HaloTokens{})
}
