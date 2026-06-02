package pool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/observability/logging"
)

// poolImpl implémente Pool avec round-robin PolicyAnyPublic et pinned PolicyPinnedPlayer.
// Gère N tokens en parallèle, chacun avec son rate limiter (RPS par token).
type poolImpl struct {
	resolver Resolver

	// Slots : chaque slot = 1 token réactivé + rate limiter (HaloAPIClient).
	slots     []*slot
	slotsByGt map[string]int // gamertag → slot index (pour PolicyPinnedPlayer)
	slotMu    sync.RWMutex

	// Round-robin pour PolicyAnyPublic — canal buffered.
	anyPublicChan chan int // indice slot

	// Configuration.
	maxSize         int
	perTokenRPS     int
	refreshInterval time.Duration
	globalCooldown  time.Duration

	// État global de cooldown (429/503).
	coolingDown    bool
	cooldownUntil  time.Time
	cooldownMu     sync.Mutex
	cooldownCancel context.CancelFunc // canceller le refresher loop pour respecter cooldown
	cooldownCtx    context.Context
	consecutive429 int // compteur backoff exponentiel 429 (protégé par cooldownMu)

	// Goroutine refresher en arrière-plan.
	stopOnce sync.Once
	stopCh   chan struct{}
}

// slot encapsule l'état d'un token dans le pool.
// Chaque slot possède son propre *rate.Limiter (Option 2 de l'audit 2026-05-21) :
// PerTokenRPS appliqué par identité distincte → throughput global = PerTokenRPS × Size().
type slot struct {
	gamertag    string
	xuid        string
	resolved    *ResolvedTokens
	limiter     *rate.Limiter // rate-limit par-token, partagé entre tous les leases sortants
	mu          sync.RWMutex
	healthy     bool // faux après 401/403, remis vrai après Refresh réussi
	lastRefresh time.Time
}

// NewPool crée un Pool à partir d'une liste de CredentialSources découvertes.
// opts.MaxSize = 0 → utiliser tous les sources. opts.PerTokenRPS = 0 → défaut 1 RPS.
func NewPool(
	ctx context.Context,
	resolver Resolver,
	sources []CredentialSource,
	opts PoolOptions,
) (Pool, error) {
	// Appliquer les valeurs par défaut.
	if opts.PerTokenRPS == 0 {
		opts.PerTokenRPS = 1
	}
	if opts.RefreshInterval == 0 {
		opts.RefreshInterval = 3*time.Hour + 30*time.Minute
	}
	if opts.GlobalCooldown == 0 {
		opts.GlobalCooldown = 30 * time.Second
	}

	// Limiter à MaxSize.
	poolSize := len(sources)
	if opts.MaxSize > 0 && poolSize > opts.MaxSize {
		poolSize = opts.MaxSize
	}

	if poolSize == 0 {
		return nil, fmt.Errorf("pool: aucune source de credential pour créer un pool")
	}

	// Créer les slots.
	slots := make([]*slot, poolSize)
	slotsByGt := make(map[string]int)

	for i := 0; i < poolSize; i++ {
		src := sources[i]

		// Résoudre la source au boot.
		resolved, err := resolver.Resolve(ctx, src)
		if err != nil {
			slog.WarnContext(ctx, "pool: impossible de résoudre token au boot, skip slot",
				"gamertag", src.Gamertag, "err", err)
			// Skip ce slot si la résolution échoue au boot.
			slots = slots[:len(slots)-1]
			poolSize--
			i--
			continue
		}

		slots[i] = &slot{
			gamertag:    src.Gamertag,
			xuid:        src.XUID,
			resolved:    resolved,
			limiter:     rate.NewLimiter(rate.Limit(opts.PerTokenRPS), 1),
			healthy:     true,
			lastRefresh: time.Now(),
		}
		slotsByGt[src.Gamertag] = i
	}

	if poolSize == 0 {
		return nil, fmt.Errorf("pool: aucun slot créé (toutes les résolutions ont échoué)")
	}

	// Créer le cooldown context.
	cooldownCtx, cancel := context.WithCancel(context.Background())

	p := &poolImpl{
		resolver:        resolver,
		slots:           slots,
		slotsByGt:       slotsByGt,
		anyPublicChan:   make(chan int, poolSize),
		maxSize:         opts.MaxSize,
		perTokenRPS:     opts.PerTokenRPS,
		refreshInterval: opts.RefreshInterval,
		globalCooldown:  opts.GlobalCooldown,
		stopCh:          make(chan struct{}),
		cooldownCtx:     cooldownCtx,
		cooldownCancel:  cancel,
	}

	// Initialiser le canal round-robin avec tous les slots.
	for i := 0; i < poolSize; i++ {
		p.anyPublicChan <- i
	}

	slog.InfoContext(ctx, "pool: créé",
		"size", len(p.slots), "perTokenRPS", opts.PerTokenRPS)

	// Lancer le refresher en arrière-plan (ne pas attendre).
	go p.refresherLoop(context.Background())

	return p, nil
}

// Acquire implémente Pool.Acquire() avec support PolicyAnyPublic et PolicyPinnedPlayer.
func (p *poolImpl) Acquire(ctx context.Context, policy AcquirePolicy, pinnedGamertag string) (*Lease, error) {
	switch policy {
	case PolicyAnyPublic:
		return p.acquireAnyPublic(ctx)
	case PolicyPinnedPlayer:
		return p.acquirePinnedPlayer(ctx, pinnedGamertag)
	default:
		return nil, fmt.Errorf("pool: policy inconnue %v", policy)
	}
}

// acquireAnyPublic : round-robin parmi les slots sains.
func (p *poolImpl) acquireAnyPublic(ctx context.Context) (*Lease, error) {
	maxRetries := len(p.slots) // Éviter boucle infinie si tous les slots sont malsains.

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case slotIdx := <-p.anyPublicChan:
			slot := p.slots[slotIdx]

			// Vérifier que le slot est sain.
			slot.mu.RLock()
			healthy := slot.healthy
			slot.mu.RUnlock()

			if !healthy {
				// Slot malsain, remettre dans le canal et essayer le suivant.
				p.anyPublicChan <- slotIdx
				continue
			}

			// Slot sain, créer un Lease.
			return &Lease{
				Tokens:   slot.resolved.Tokens,
				Gamertag: slot.gamertag,
				Limiter:  slot.limiter,
				Release: func() {
					// Remettre le slot dans le canal.
					p.anyPublicChan <- slotIdx
				},
			}, nil

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Tous les slots essayés sont malsains.
	return nil, fmt.Errorf("pool: aucun slot sain disponible (PolicyAnyPublic)")
}

// acquirePinnedPlayer : lookup par gamertag, retourne ErrNoTokenForPlayer si absent ou malsain.
//
// même si le lookup en mémoire pure n'a pas besoin de timeout/cancel aujourd'hui.
//
//nolint:unparam // ctx maintenu pour cohérence avec l'interface caller (Acquire(ctx))
func (p *poolImpl) acquirePinnedPlayer(ctx context.Context, gamertag string) (*Lease, error) {
	p.slotMu.RLock()
	slotIdx, ok := p.slotsByGt[gamertag]
	p.slotMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pool: %q n'a pas de token pinné", gamertag)
	}

	slot := p.slots[slotIdx]

	// Vérifier l'état de santé du slot.
	slot.mu.RLock()
	healthy := slot.healthy
	slot.mu.RUnlock()

	if !healthy {
		return nil, fmt.Errorf("pool: token %q est malsain (401/403), en refresh", gamertag)
	}

	return &Lease{
		Tokens:   slot.resolved.Tokens,
		Gamertag: gamertag,
		Limiter:  slot.limiter,
		Release:  func() {}, // No-op pour PolicyPinnedPlayer.
	}, nil
}

// Size implémente Pool.Size().
func (p *poolImpl) HasPlayer(gamertag string) bool {
	p.slotMu.RLock()
	defer p.slotMu.RUnlock()
	_, ok := p.slotsByGt[gamertag]
	return ok
}

func (p *poolImpl) Size() int {
	p.slotMu.RLock()
	defer p.slotMu.RUnlock()
	return len(p.slots)
}

// AddOrUpdateSource implémente Pool.AddOrUpdateSource() — hot-add ou refresh
// d'un slot par gamertag (E.v2, cf. PLAN_AUTH_PROVIDER_UNIFICATION.md).
//
// Pre-Resolve hors lock pour ne pas bloquer Acquire() pendant l'appel réseau
// (qui peut prendre 1-3s pour Exchange OAuth).
func (p *poolImpl) AddOrUpdateSource(ctx context.Context, src CredentialSource) error {
	if src.Gamertag == "" {
		return fmt.Errorf("pool: AddOrUpdateSource gamertag vide")
	}

	resolved, err := p.resolver.Resolve(ctx, src)
	if err != nil {
		return fmt.Errorf("pool: AddOrUpdateSource resolve %s: %w", src.Gamertag, err)
	}

	p.slotMu.Lock()
	defer p.slotMu.Unlock()

	// Update existing slot in-place.
	if idx, exists := p.slotsByGt[src.Gamertag]; exists {
		s := p.slots[idx]
		s.mu.Lock()
		s.resolved = resolved
		s.xuid = src.XUID
		s.healthy = true
		s.lastRefresh = time.Now()
		s.mu.Unlock()
		slog.InfoContext(ctx, "pool: slot updated",
			"gamertag", src.Gamertag, "source", src.Source)
		return nil
	}

	// New slot : check MaxSize cap.
	if p.maxSize > 0 && len(p.slots) >= p.maxSize {
		return fmt.Errorf("pool: AddOrUpdateSource pool full (maxSize=%d)", p.maxSize)
	}

	newSlot := &slot{
		gamertag:    src.Gamertag,
		xuid:        src.XUID,
		resolved:    resolved,
		limiter:     rate.NewLimiter(rate.Limit(p.perTokenRPS), 1),
		healthy:     true,
		lastRefresh: time.Now(),
	}
	newIdx := len(p.slots)
	p.slots = append(p.slots, newSlot)
	p.slotsByGt[src.Gamertag] = newIdx

	// Best-effort push dans le canal round-robin (capacité fixe au boot).
	// Si plein → slot reachable uniquement via PolicyPinnedPlayer.
	select {
	case p.anyPublicChan <- newIdx:
	default:
		slog.DebugContext(ctx, "pool: anyPublicChan plein, slot reachable seulement via PolicyPinnedPlayer",
			"gamertag", src.Gamertag, "new_size", len(p.slots))
	}

	slog.InfoContext(ctx, "pool: slot ajouté (hot-add)",
		"gamertag", src.Gamertag, "source", src.Source, "new_size", len(p.slots))
	return nil
}

// MarkUnhealthy invalide un token après 401/403 et déclenche un Resolver.Refresh asynchrone.
func (p *poolImpl) MarkUnhealthy(gamertag string, reason error) {
	p.slotMu.RLock()
	slotIdx, ok := p.slotsByGt[gamertag]
	p.slotMu.RUnlock()

	if !ok {
		slog.WarnContext(context.Background(), "pool: MarkUnhealthy: gamertag inconnu",
			"gamertag", gamertag)
		return
	}

	slot := p.slots[slotIdx]

	// Marquer malsain.
	slot.mu.Lock()
	slot.healthy = false
	slot.mu.Unlock()

	slog.WarnContext(context.Background(), "pool: token marqué malsain",
		"gamertag", gamertag, "reason", reason)

	// Déclencher un refresh asynchrone (le refresherLoop s'en chargera dans son prochain cycle).
}

// maxBackoffShift borne le décalage exponentiel du cooldown 429 (globalCooldown<<shift).
// maxCooldown borne la durée totale d'un cooldown (Retry-After ou backoff).
const (
	maxBackoffShift = 4
	maxCooldown     = 5 * time.Minute
)

// OnHTTPError signale une erreur HTTP (429/503) et déclenche un cooldown global.
// Non-bloquant : tous les tokens sont marqués malsains et le refresher est suspendu.
// retryAfter > 0 = durée du header Retry-After (prioritaire) ; sinon backoff
// exponentiel sur globalCooldown pour les 429 répétés, borné à maxCooldown.
func (p *poolImpl) OnHTTPError(statusCode int, retryAfter time.Duration) {
	if statusCode != 429 && statusCode != 503 {
		// Ignorer les autres codes d'erreur.
		return
	}

	p.cooldownMu.Lock()
	defer p.cooldownMu.Unlock()

	// Durée de cooldown : Retry-After prioritaire, sinon backoff exponentiel.
	var dur time.Duration
	honored := false
	switch {
	case retryAfter > 0:
		dur = retryAfter
		p.consecutive429 = 0
		honored = true
	case statusCode == 429:
		shift := p.consecutive429
		if shift > maxBackoffShift {
			shift = maxBackoffShift
		}
		dur = p.globalCooldown << shift
		p.consecutive429++
	default:
		dur = p.globalCooldown
	}
	if dur > maxCooldown {
		dur = maxCooldown
	}
	newUntil := time.Now().Add(dur)

	// Déjà en cooldown : n'écraser QUE si le nouveau délai est plus tardif (un
	// Retry-After plus long ne doit pas être ignoré). Les métriques ne sont
	// comptées que si le cooldown est RÉELLEMENT (ré)appliqué — un 429 ignoré
	// pendant un cooldown plus long ne doit pas incrémenter cooldowns_total ni
	// retry_after_honored_total.
	if p.coolingDown && time.Now().Before(p.cooldownUntil) {
		if newUntil.After(p.cooldownUntil) {
			p.cooldownUntil = newUntil
			cooldownExtendedTotal.Add(1)
			recordCooldownMetrics(statusCode, honored)
			lastCooldownSeconds.Set(int64(dur.Seconds()))
			slog.WarnContext(context.Background(), "pool: cooldown prolongé",
				"status", statusCode, "cooldown_s", dur.Seconds(), "retry_after_honored", honored)
		} else {
			slog.DebugContext(context.Background(), "pool: OnHTTPError ignoré pendant cooldown plus long",
				"status", statusCode)
		}
		return
	}

	// Déclencher le cooldown global.
	p.coolingDown = true
	p.cooldownUntil = newUntil
	recordCooldownMetrics(statusCode, honored)
	lastCooldownSeconds.Set(int64(dur.Seconds()))

	slog.WarnContext(context.Background(), "pool: cooldown global déclenché",
		"status", statusCode, "cooldown_s", dur.Seconds(), "retry_after_honored", honored)

	// Marquer tous les tokens comme malsains (non-bloquant).
	for _, slot := range p.slots {
		slot.mu.Lock()
		slot.healthy = false
		slot.mu.Unlock()
	}

	slog.InfoContext(context.Background(), "pool: tous les tokens marqués malsains (cooldown)",
		"count", len(p.slots))
}

// Close implémente Pool.Close().
func (p *poolImpl) Close() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		p.cooldownCancel()
		slog.InfoContext(context.Background(), "pool: fermé")
	})
}

// refresherLoop en arrière-plan, rafraîchit les tokens malsains ou proches de l'expiration.
func (p *poolImpl) refresherLoop(baseCtx context.Context) {
	// Sprint B1 commit 17 : event_id sur la loop globale. Les opérations
	// individuelles (Refresh par slot) génèrent leur propre sous-event_id
	// via Resolver.Refresh.
	baseCtx, loopID := logging.WithEvent(baseCtx, "auth.refresher_loop")
	slog.InfoContext(baseCtx, "pool: refresher loop démarré", "event", loopID)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return

		case <-ticker.C:
			// Vérifier cooldown global.
			p.cooldownMu.Lock()
			if p.coolingDown && time.Now().Before(p.cooldownUntil) {
				p.cooldownMu.Unlock()
				continue
			}
			if p.coolingDown {
				p.coolingDown = false
				p.consecutive429 = 0 // reset du backoff à la sortie saine du cooldown
				slog.InfoContext(baseCtx, "pool: cooldown global levé")
			}
			p.cooldownMu.Unlock()

			// Parcourir les slots et refresh les malsains.
			for i, slot := range p.slots {
				slot.mu.RLock()
				healthy := slot.healthy
				slot.mu.RUnlock()

				if !healthy {
					// Asynchronously refresh sans bloquer la loop.
					go func(slotIdx int, gt string) {
						ctx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
						defer cancel()

						refreshed, err := p.resolver.Refresh(ctx, gt)
						if err != nil {
							slog.ErrorContext(ctx, "pool: refresh échoué",
								"gamertag", gt, "err", err)
							return
						}

						// Mettre à jour le slot.
						slot := p.slots[slotIdx]
						slot.mu.Lock()
						slot.resolved = refreshed
						slot.healthy = true
						slot.lastRefresh = time.Now()
						slot.mu.Unlock()

						slog.InfoContext(ctx, "pool: token rafraîchi avec succès",
							"gamertag", gt)
					}(i, slot.gamertag)
				}
			}
		}
	}
}
