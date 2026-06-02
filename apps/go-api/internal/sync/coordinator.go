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
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
)

// Clés expvar du gate (namespace "levelup" sur /debug/vars, cardinalité bornée
// — clés fixes, jamais par gamertag, cf. ADR 0009).
const (
	metricGateGranted   = "sync_gate_claims_granted_total"   // claims auto/HTTP accordés
	metricGateCoalesced = "sync_gate_claims_coalesced_total" // claims auto/HTTP cédés (double-fetch évité)
	metricGateInflight  = "sync_gate_inflight"               // jauge claims gate en cours (une valeur figée = fuite)
)

// staleClaimThreshold : au-delà, un claim est considéré potentiellement FUITÉ
// (release jamais appelé → joueur jamais re-synchronisé). Choisi > syncJobTimeout
// HTTP (30 min) pour ne pas alerter sur un sync initial volumineux légitime.
const staleClaimThreshold = 45 * time.Minute

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
	// GateSnapshot retourne un cliché d'observabilité (compteurs + claims en vol
	// avec leur âge) pour le diagnostic (/_diag/auto-sync/snapshot) et la détection
	// de claim fuité.
	GateSnapshot() GateSnapshotData
}

// GateClaimInfo décrit un claim en vol (diag). Source : "watcher" (Submit) ou
// "gate" (TryClaim auto/HTTP).
type GateClaimInfo struct {
	Gamertag string `json:"gamertag"`
	Source   string `json:"source"`
	AgeMs    int64  `json:"age_ms"`
	Stale    bool   `json:"stale"` // âge >= staleClaimThreshold → potentiellement fuité
}

// GateSnapshotData est un cliché de l'état du gate de déduplication cross-source.
type GateSnapshotData struct {
	InflightWatcher int             `json:"inflight_watcher"`
	InflightGate    int             `json:"inflight_gate"`
	GrantedTotal    int64           `json:"granted_total"`
	CoalescedTotal  int64           `json:"coalesced_total"`
	StaleCount      int             `json:"stale_count"`
	Claims          []GateClaimInfo `json:"claims,omitempty"`
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

// GateSnapshot retourne un cliché vide (aucun claim suivi).
func (NopSyncGate) GateSnapshot() GateSnapshotData { return GateSnapshotData{} }

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
	inFlightMu  sync.Mutex           // protège inFlight, gateClaims ET closing
	inFlight    map[string]time.Time // claims WATCHER (Submit/run) — clé normalisée → début du claim
	gateClaims  map[string]time.Time // claims AUTO/HTTP (TryClaim) — clé normalisée → début du claim
	closing     bool                 // levé par BeginShutdown : TryClaim refuse tout claim
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
		inFlight:    make(map[string]time.Time),
		gateClaims:  make(map[string]time.Time),
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
	if _, busy := c.inFlight[key]; busy {
		c.inFlightMu.Unlock()
		slog.InfoContext(ctx, "coordinator: sync watcher déjà en cours, requête ignorée (dedup)",
			"gamertag", req.Gamertag,
			"match_count", len(req.MatchIDs),
			"event", evID,
		)
		return false
	}
	c.inFlight[key] = time.Now()
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
	if c.closing {
		c.inFlightMu.Unlock()
		return nil, false // refus de shutdown : ne compte pas comme coalesced
	}
	_, w := c.inFlight[key]
	_, g := c.gateClaims[key]
	if w || g {
		c.inFlightMu.Unlock()
		// Cédé : un sync du joueur est déjà en vol (watcher ou auto/HTTP). C'est
		// LA métrique de valeur — chaque coalesced = un double-fetch API évité.
		observability.IncCounter(metricGateCoalesced)
		return nil, false
	}
	c.gateClaims[key] = time.Now()
	c.gateWG.Add(1) // sous inFlightMu + !closing → jamais concurrent à gateWG.Wait
	c.inFlightMu.Unlock()
	observability.IncCounter(metricGateGranted)
	observability.AddInt(metricGateInflight, 1)
	return sync.OnceFunc(func() {
		c.inFlightMu.Lock()
		delete(c.gateClaims, key)
		c.inFlightMu.Unlock()
		c.gateWG.Done()
		observability.AddInt(metricGateInflight, -1)
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
	_, w := c.inFlight[key]
	_, g := c.gateClaims[key]
	return w || g
}

// GateSnapshot — cf. SyncGate. Cliché d'observabilité : compteurs cumulés (lus
// depuis expvar) + claims en vol avec leur âge et un flag stale (potentiellement
// fuité). Sans effet de bord ; verrou bref sur inFlightMu.
func (c *Coordinator) GateSnapshot() GateSnapshotData {
	now := time.Now()
	snap := GateSnapshotData{
		GrantedTotal:   observability.LoadCounter(metricGateGranted),
		CoalescedTotal: observability.LoadCounter(metricGateCoalesced),
	}
	c.inFlightMu.Lock()
	snap.InflightWatcher = len(c.inFlight)
	snap.InflightGate = len(c.gateClaims)
	snap.Claims = make([]GateClaimInfo, 0, len(c.inFlight)+len(c.gateClaims))
	collect := func(m map[string]time.Time, source string) {
		for gt, started := range m {
			age := now.Sub(started)
			stale := age >= staleClaimThreshold
			if stale {
				snap.StaleCount++
			}
			snap.Claims = append(snap.Claims, GateClaimInfo{
				Gamertag: gt, Source: source, AgeMs: age.Milliseconds(), Stale: stale,
			})
		}
	}
	collect(c.inFlight, "watcher")
	collect(c.gateClaims, "gate")
	c.inFlightMu.Unlock()
	return snap
}

// InFlightCount retourne le nombre de syncs en cours (watcher + auto/HTTP).
func (c *Coordinator) InFlightCount() int {
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	return len(c.inFlight) + len(c.gateClaims)
}
