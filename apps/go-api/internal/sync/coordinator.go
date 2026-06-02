// Package sync — coordinator.go : coordination des syncs déclenchés par le watcher.
//
// Le SyncCoordinator :
//   - Consomme les MatchRequests depuis la MatchQueue
//   - Garantit au plus N syncs simultanés via sémaphore
//   - Empêche les syncs concurrents sur le même joueur (inFlight map)
//   - Appelle le SyncEngine pour chaque joueur+matchIDs
//
// COORDINATION CROSS-SOURCE (revue 2026-06-01 → unifiée 2026-06-02, AS-1/D3-02 ;
// durcie après revue adversariale) : la dédup repose sur DEUX maps avec une
// PRIORITÉ ASYMÉTRIQUE — le watcher n'est jamais bloqué, auto/HTTP cèdent.
//   - `inFlight` (WATCHER) : alimentée par Submit. Le watcher est PRIORITAIRE
//     fraîcheur : son Submit ne dédoublonne QUE contre d'autres syncs watcher du
//     même joueur (jamais contre auto/HTTP). Il pose toujours son claim et lance
//     son RunDelta — le lease dblease KindPlayer sérialise au besoin face à un
//     auto/HTTP concurrent (double travail borné, mais le match frais est écrit
//     dans le même cycle).
//   - `gateClaims` (AUTO + HTTP) : alimentée par TryClaim (interface SyncGate).
//     Avant leur RunDelta, ces sources demandent un claim ; TryClaim renvoie
//     ok=false si le joueur est en vol côté watcher (`inFlight`) OU côté auto/HTTP
//     (`gateClaims`) — donc auto/HTTP CÈDENT au watcher ET entre eux (auto = skip
//     re-tenté au prochain tick ; HTTP = 409 synchrone). Pas de double fetch.
//
// Le claim borne le double-TRAVAIL (fetch API + recompute). Le lease dblease
// KindPlayer (pris EN AVAL, dans RunDelta) borne la double-ÉCRITURE (corruption).
// Invariant d'ordre : TryClaim AVANT le lease, release APRÈS (defer LIFO côté
// caller). TryClaim ne prend JAMAIS le leaseMutex → aucun cycle claim↔lease.
//
// SHUTDOWN : `closing` (sous inFlightMu) est levé par BeginShutdown() avant tout
// WaitInFlight(). Une fois levé, TryClaim refuse tout nouveau claim AVANT
// gateWG.Add — garantit qu'aucun Add ne court concurremment à gateWG.Wait (sinon
// data race / panic WaitGroup) et que Wait ne rend pas la main prématurément.
// Deux WaitGroup DISTINCTS : `wg` ne tracke que les run() watcher (attendu par
// daemon.Stop via Wait(), invariant COORD-1) ; `gateWG` tracke les claims auto/
// HTTP (attendu par WaitInFlight() pour qu'ils finissent avant duckdb.CloseAll()).
package sync

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"levelup/go-api/internal/observability/logging"
)

// SyncRunner exécute le sync pour un joueur avec les match_ids donnés.
type SyncRunner interface {
	RunSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error
}

// SyncGate est le point de déduplication cross-source pour l'auto-sync scheduler
// et le handler HTTP. Ils l'interrogent AVANT de lancer un RunDelta, afin de céder
// si un sync du même joueur est déjà en vol — côté watcher OU côté auto/HTTP. Le
// watcher ne passe PAS par cette interface (il est prioritaire : Submit direct).
// *Coordinator l'implémente ; NopSyncGate est le no-op (dédup désactivée, legacy).
type SyncGate interface {
	// TryClaim réserve le joueur s'il n'est pas déjà en vol (ni watcher ni
	// auto/HTTP). Retourne (release, true) en cas de succès — le caller DOIT
	// `defer release()` ; sinon (nil, false). release est idempotent. Purement
	// mémoire : jamais bloquant, pas d'IO, donc pas de ctx. Renvoie toujours
	// (nil,false) après BeginShutdown().
	TryClaim(gamertag string) (release func(), ok bool)
	// IsInFlight indique si un sync du joueur est en cours (watcher ou auto/HTTP).
	// Pré-check best-effort (TOCTOU) — la garantie réelle d'exclusion est TryClaim.
	IsInFlight(gamertag string) bool
	// WaitInFlight bloque jusqu'à la fin des claims auto/HTTP. À appeler au
	// shutdown APRÈS BeginShutdown() et l'annulation des ctx de sync, sous un
	// timeout dur côté caller. (Les syncs watcher sont attendus séparément par
	// daemon.Stop.)
	WaitInFlight()
	// BeginShutdown bascule le gate en mode arrêt : tout TryClaim ultérieur échoue.
	// À appeler AVANT WaitInFlight pour qu'aucun nouveau claim ne puisse survenir
	// concurremment au drain (évite une data race / un retour prématuré de Wait).
	BeginShutdown()
}

// NopSyncGate est un SyncGate no-op : TryClaim réussit toujours (aucune dédup
// cross-source). Câblé quand le watcher est désactivé (pas de Coordinator
// partagé) — préserve le comportement legacy (le lease reste le seul rempart).
type NopSyncGate struct{}

// TryClaim accorde toujours le claim (pas de dédup).
func (NopSyncGate) TryClaim(string) (func(), bool) { return func() {}, true }

// IsInFlight renvoie toujours false (pas de suivi).
func (NopSyncGate) IsInFlight(string) bool { return false }

// WaitInFlight ne bloque jamais (aucun claim suivi).
func (NopSyncGate) WaitInFlight() {}

// BeginShutdown est un no-op (aucun claim à figer).
func (NopSyncGate) BeginShutdown() {}

// Garde-fou compile-time : les deux types satisfont l'interface.
var (
	_ SyncGate = (*Coordinator)(nil)
	_ SyncGate = NopSyncGate{}
)

// CoordinatorRequest est une demande de sync entrante.
type CoordinatorRequest struct {
	Gamertag string
	XUID     string
	MatchIDs []string
}

// Coordinator coordonne les syncs avec contrôle de concurrence.
type Coordinator struct {
	runner      SyncRunner
	maxParallel int
	sem         chan struct{}
	inFlightMu  sync.Mutex      // protège inFlight, gateClaims ET closing
	inFlight    map[string]bool // claims WATCHER (Submit/run) — clé normalisée
	gateClaims  map[string]bool // claims AUTO/HTTP (TryClaim) — clé normalisée
	closing     bool            // levé par BeginShutdown : TryClaim refuse tout claim
	onComplete  func(gamertag string, err error)
	// wg track les goroutines run() en vol. Le caller (watcher daemon) appelle
	// Wait() au shutdown — APRÈS avoir annulé le ctx des syncs — pour ne pas
	// rendre la main tant qu'un RunSync écrit encore une DB (sinon write-after
	// duckdb.CloseAll → handle orphelin / WAL #7659, cf. revue COORD-1 2026-06-01).
	wg sync.WaitGroup
	// gateWG track les claims AUTO/HTTP (gateClaims). DISTINCT de wg (run() watcher).
	// Attendu par WaitInFlight() au shutdown pour que les RunDelta auto/HTTP
	// gate-claimés finissent avant duckdb.CloseAll(). gateWG.Add n'a lieu que sous
	// inFlightMu ET tant que !closing → jamais concurrent à gateWG.Wait.
	gateWG sync.WaitGroup
}

// NewCoordinator crée un coordinateur avec limite de syncs parallèles.
func NewCoordinator(runner SyncRunner, maxParallel int) *Coordinator {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &Coordinator{
		runner:      runner,
		maxParallel: maxParallel,
		sem:         make(chan struct{}, maxParallel),
		inFlight:    make(map[string]bool),
		gateClaims:  make(map[string]bool),
	}
}

// SetOnComplete définit un callback appelé après chaque sync (succès ou erreur).
func (c *Coordinator) SetOnComplete(fn func(gamertag string, err error)) {
	c.onComplete = fn
}

// Submit soumet une requête de sync.
// Retourne immédiatement — le sync est exécuté en goroutine.
// Retourne false si le joueur a déjà un sync en cours.
func (c *Coordinator) Submit(ctx context.Context, req CoordinatorRequest) bool {
	// Sprint B1 commit 19 : event_id sur l'arrivée au coordinator. Trace
	// le maillon entre `watcher.trigger:<gt>` (dequeue) et la sync effective
	// (sync.RunDelta dans c.runner.RunSync). Permet de répondre "pourquoi
	// cette requête a-t-elle été dédupée ?".
	ctx, evID := logging.WithEvent(ctx, "coordinator.submit:"+req.Gamertag)

	// Le watcher est PRIORITAIRE : son claim ne dédoublonne QUE contre un autre
	// sync watcher du même joueur (jamais contre auto/HTTP). Il n'est donc jamais
	// bloqué par un claim auto/HTTP — il pose son claim et lance son RunDelta ; le
	// lease KindPlayer sérialise au besoin (le match frais est écrit dans le cycle).
	key := normGT(req.Gamertag)
	c.inFlightMu.Lock()
	if c.inFlight[key] {
		c.inFlightMu.Unlock()
		slog.InfoContext(ctx, "coordinator: sync watcher déjà en cours, requête ignorée (dedup)",
			"gamertag", req.Gamertag,
			"match_count", len(req.MatchIDs),
			"event", evID,
		)
		return false
	}
	c.inFlight[key] = true
	c.inFlightMu.Unlock()

	slog.InfoContext(ctx, "coordinator: requête acceptée",
		"gamertag", req.Gamertag,
		"match_count", len(req.MatchIDs),
		"event", evID,
	)
	c.wg.Add(1)
	go c.run(ctx, req)
	return true
}

// TryClaim — cf. SyncGate. Réserve un slot AUTO/HTTP pour le joueur s'il n'est en
// vol NI côté watcher (inFlight) NI côté auto/HTTP (gateClaims) — donc auto/HTTP
// cèdent au watcher ET entre eux. Renvoie (nil,false) après BeginShutdown. release
// (sync.OnceFunc) retire le gateClaim + décrémente gateWG, idempotent même sur panic.
func (c *Coordinator) TryClaim(gamertag string) (func(), bool) {
	key := normGT(gamertag)
	c.inFlightMu.Lock()
	if c.closing || c.inFlight[key] || c.gateClaims[key] {
		c.inFlightMu.Unlock()
		return nil, false
	}
	c.gateClaims[key] = true
	c.gateWG.Add(1) // sous inFlightMu + !closing → jamais concurrent à gateWG.Wait
	c.inFlightMu.Unlock()
	return sync.OnceFunc(func() {
		c.inFlightMu.Lock()
		delete(c.gateClaims, key)
		c.inFlightMu.Unlock()
		c.gateWG.Done()
	}), true
}

// WaitInFlight — cf. SyncGate. Bloque jusqu'à la fin des claims AUTO/HTTP. Distinct
// de Wait() (qui n'attend que les run() watcher, COORD-1). À appeler après
// BeginShutdown() pour garantir l'absence de gateWG.Add concurrent.
func (c *Coordinator) WaitInFlight() {
	c.gateWG.Wait()
}

// BeginShutdown — cf. SyncGate. Fige le gate : tout TryClaim ultérieur échoue, donc
// plus aucun gateWG.Add ne peut survenir. À appeler AVANT WaitInFlight.
func (c *Coordinator) BeginShutdown() {
	c.inFlightMu.Lock()
	c.closing = true
	c.inFlightMu.Unlock()
}

// normGT normalise la clé de dédup (insensible à la casse et aux espaces de
// bord). Cohérent avec la clé du lease dblease KindPlayer, dérivée du gamertag.
func normGT(gamertag string) string {
	return strings.ToLower(strings.TrimSpace(gamertag))
}

// Wait bloque jusqu'à ce que tous les syncs en vol (goroutines run) soient
// terminés. À appeler par le daemon au shutdown APRÈS l'annulation du ctx (qui
// fait abandonner les RunSync en cours), de préférence sous un timeout dur.
func (c *Coordinator) Wait() {
	c.wg.Wait()
}

// run exécute le sync watcher avec sémaphore. Le claim inFlight (posé par Submit)
// est libéré en defer de tête pour couvrir aussi le chemin ctx.Done() (sémaphore
// non acquis).
func (c *Coordinator) run(ctx context.Context, req CoordinatorRequest) {
	defer c.wg.Done()
	defer c.releaseInFlight(req.Gamertag)
	// Acquérir le sémaphore (watcher uniquement — borne le parallélisme des syncs
	// déclenchés par le watcher ; auto/HTTP ne consomment pas ce sémaphore).
	select {
	case c.sem <- struct{}{}:
	case <-ctx.Done():
		return
	}

	defer func() {
		<-c.sem // libérer le sémaphore
	}()

	slog.InfoContext(ctx, "coordinator: démarrage sync",
		"gamertag", req.Gamertag,
		"match_count", len(req.MatchIDs),
		"parallel_slots", c.maxParallel,
	)

	err := c.runner.RunSync(ctx, req.Gamertag, req.XUID, req.MatchIDs)
	if err != nil {
		slog.ErrorContext(ctx, "coordinator: sync échoué",
			"gamertag", req.Gamertag,
			"err", err,
		)
	} else {
		slog.InfoContext(ctx, "coordinator: sync terminé",
			"gamertag", req.Gamertag,
		)
	}

	if c.onComplete != nil {
		c.onComplete(req.Gamertag, err)
	}
}

// releaseInFlight retire le joueur de la map inFlight watcher (clé normalisée).
func (c *Coordinator) releaseInFlight(gamertag string) {
	c.inFlightMu.Lock()
	delete(c.inFlight, normGT(gamertag))
	c.inFlightMu.Unlock()
}

// IsInFlight vérifie si un joueur a un sync en cours, watcher OU auto/HTTP (clé
// normalisée). Best-effort (TOCTOU) — la garantie d'exclusion est TryClaim.
func (c *Coordinator) IsInFlight(gamertag string) bool {
	key := normGT(gamertag)
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	return c.inFlight[key] || c.gateClaims[key]
}

// InFlightCount retourne le nombre de syncs en cours (watcher + auto/HTTP).
func (c *Coordinator) InFlightCount() int {
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	return len(c.inFlight) + len(c.gateClaims)
}
