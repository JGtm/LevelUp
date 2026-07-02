package pool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/ratebudget"
)

// poolImpl implémente Pool avec round-robin PolicyAnyPublic et pinned PolicyPinnedPlayer.
// Gère N tokens en parallèle, chacun avec son rate limiter (RPS par token).
type poolImpl struct {
	resolver Resolver

	// titleSlug : titre propriétaire du pool (Phase 1.6). Un pool est mono-titre
	// (Discovery scanne les sources d'UN titre) ; titleSlug compose la clé des
	// slots pour interdire toute collision/contamination cross-titre.
	titleSlug string

	// Slots : chaque slot = 1 token réactivé + rate limiter (HaloAPIClient).
	slots []*slot
	// slotsByKey : clé composite (titleSlug, gamertag) → slot index (PolicyPinnedPlayer).
	// Cf. gtKey. La clé inclut le titre pour le multi-titres (Phase 1.6).
	slotsByKey map[string]int
	slotMu     sync.RWMutex

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
	// rateLimitedUntil : le token a pris un 429 (rate-limit d'API, PAS un échec
	// d'auth) — il reste VALIDE mais est skippé par l'acquisition jusqu'à cette
	// date. Auto-résorption temporelle, SANS re-exchange (un 429 ne justifie pas
	// de refaire la chaîne XBL/XSTS). Distinct de healthy (401/403 = token mort).
	rateLimitedUntil time.Time
	// lastAttempt : dernière tentative de refresh (réactive ou proactive), pour
	// throttler le refresh proactif near-expiry.
	lastAttempt time.Time
}

// leaseData retourne, sous UN SEUL RLock, si le slot est acquérable MAINTENANT
// (sain — pas de 401/403 — ET hors cooldown de rate-limit 429) et les données
// nécessaires à un Lease capturées de façon COHÉRENTE. Capturer resolved/limiter
// ici (et non hors verrou comme avant) élimine la data race avec le refresher
// async et AddOrUpdateSource, qui réassignent slot.resolved sous slot.mu. Le
// *ResolvedTokens pointé est immuable après création (le refresher REMPLACE le
// pointeur, ne le mute pas), donc lire resolved.Tokens après le RUnlock est sûr.
func (s *slot) leaseData(now time.Time) (resolved *ResolvedTokens, gamertag string, limiter *rate.Limiter, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.healthy || now.Before(s.rateLimitedUntil) {
		return nil, "", nil, false
	}
	return s.resolved, s.gamertag, s.limiter, true
}

// gtKey construit la clé composite (titleSlug, gamertag) du registre de slots
// (Phase 1.6). Le séparateur NUL ne peut apparaître dans un gamertag Xbox ni
// dans un slug, ce qui évite toute collision entre deux titres pour un même
// gamertag. Un titleSlug vide (sources legacy/tests sans titre) dégrade
// proprement vers une clé gamertag-only — comportement historique inchangé.
func gtKey(titleSlug, gamertag string) string {
	return titleSlug + "\x00" + gamertag
}

// poolTitleOf retourne le premier titleSlug non vide parmi les sources — le
// titre propriétaire du pool (les sources d'un même scan partagent ce titre).
func poolTitleOf(sources []CredentialSource) string {
	for _, s := range sources {
		if s.TitleSlug != "" {
			return s.TitleSlug
		}
	}
	return ""
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

	// Titre propriétaire du pool (Phase 1.6) — compose la clé des slots.
	poolTitle := poolTitleOf(sources)

	// Créer les slots. Une source en échec est SKIPPÉE et on passe à la
	// SUIVANTE (fix 2026-06-11 : l'ancienne boucle `i--` + `poolSize--`
	// retentait le même index en boucle et abandonnait silencieusement toutes
	// les sources situées après la première en échec — cf. burst de 7
	// tentatives DankerGlue au boot, .ai/PLAN_AUTH_WARNING_NOISE.md).
	slots := make([]*slot, 0, poolSize)
	slotsByKey := make(map[string]int)

	for _, src := range sources[:poolSize] {
		resolved, err := resolver.Resolve(ctx, src)
		if err != nil {
			slog.WarnContext(ctx, "pool: impossible de résoudre token au boot, skip slot",
				"gamertag", src.Gamertag, "err", err)
			continue
		}

		slotsByKey[gtKey(poolTitle, src.Gamertag)] = len(slots)
		slots = append(slots, &slot{
			gamertag: src.Gamertag,
			xuid:     src.XUID,
			resolved: resolved,
			// Budget PAR COMPTE partagé process-wide (sujet 2 T1) : tous les
			// consommateurs du même xuid (pool, career_live, worldenrich)
			// attendent sur le MÊME token bucket — le pool voit la vraie pression.
			limiter:     ratebudget.ForXUID(src.XUID, float64(opts.PerTokenRPS)),
			healthy:     true,
			lastRefresh: time.Now(),
		})
	}
	poolSize = len(slots)

	if poolSize == 0 {
		return nil, fmt.Errorf("pool: aucun slot créé (toutes les résolutions ont échoué)")
	}

	// Créer le cooldown context.
	cooldownCtx, cancel := context.WithCancel(context.Background())

	p := &poolImpl{
		resolver:        resolver,
		titleSlug:       poolTitle,
		slots:           slots,
		slotsByKey:      slotsByKey,
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
			// Lire p.slots sous slotMu : AddOrUpdateSource peut append (réalloc du
			// backing array) concurremment (re-scan Discovery 15min).
			p.slotMu.RLock()
			slot := p.slots[slotIdx]
			p.slotMu.RUnlock()

			// Sain ET hors cooldown 429 ? Sinon remettre dans le canal et tenter
			// le suivant. Un slot rate-limité (429 sur SON token) est skippé sans
			// pénaliser les autres — fini le scorched-earth d'un 429 isolé. La
			// capture des tokens/limiter est cohérente (sous un seul RLock).
			resolved, gamertag, limiter, ok := slot.leaseData(time.Now())
			if !ok {
				p.anyPublicChan <- slotIdx
				continue
			}

			// Slot sain, créer un Lease.
			return &Lease{
				Tokens:   resolved.Tokens,
				Gamertag: gamertag,
				Limiter:  limiter,
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
	// Capturer l'index ET le pointeur de slot sous slotMu (append concurrent possible).
	p.slotMu.RLock()
	slotIdx, ok := p.slotsByKey[gtKey(p.titleSlug, gamertag)]
	var slot *slot
	if ok {
		slot = p.slots[slotIdx]
	}
	p.slotMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pool: %q n'a pas de token pinné", gamertag)
	}

	// Sain ET hors cooldown 429 ? Un token rate-limité (429) est momentanément
	// indisponible sans être marqué mort — le caller réessaiera. Capture cohérente
	// des tokens/limiter sous un seul RLock (anti-race refresher).
	resolved, _, limiter, acq := slot.leaseData(time.Now())
	if !acq {
		return nil, fmt.Errorf("pool: token %q indisponible (auth invalide ou rate-limité), réessayer", gamertag)
	}

	return &Lease{
		Tokens:   resolved.Tokens,
		Gamertag: gamertag,
		Limiter:  limiter,
		Release:  func() {}, // No-op pour PolicyPinnedPlayer.
	}, nil
}

// Size implémente Pool.Size().
func (p *poolImpl) HasPlayer(gamertag string) bool {
	p.slotMu.RLock()
	defer p.slotMu.RUnlock()
	_, ok := p.slotsByKey[gtKey(p.titleSlug, gamertag)]
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

	// Phase 1.6 : un pool est mono-titre. Rejeter toute source d'un autre titre
	// pour ne jamais servir le token d'un titre étranger via ce pool.
	if src.TitleSlug != "" && p.titleSlug != "" && src.TitleSlug != p.titleSlug {
		return fmt.Errorf("pool: AddOrUpdateSource titre %q ≠ titre du pool %q (cross-title interdit)",
			src.TitleSlug, p.titleSlug)
	}

	resolved, err := p.resolver.Resolve(ctx, src)
	if err != nil {
		return fmt.Errorf("pool: AddOrUpdateSource resolve %s: %w", src.Gamertag, err)
	}

	p.slotMu.Lock()
	defer p.slotMu.Unlock()

	// Pool initialement sans titre (ex. construit vide) : adopter celui de la source.
	if p.titleSlug == "" && src.TitleSlug != "" {
		p.titleSlug = src.TitleSlug
	}
	key := gtKey(p.titleSlug, src.Gamertag)

	// Update existing slot in-place.
	if idx, exists := p.slotsByKey[key]; exists {
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
		gamertag: src.Gamertag,
		xuid:     src.XUID,
		resolved: resolved,
		// Budget PAR COMPTE partagé (sujet 2 T1) — cf. NewPool.
		limiter:     ratebudget.ForXUID(src.XUID, float64(p.perTokenRPS)),
		healthy:     true,
		lastRefresh: time.Now(),
	}
	newIdx := len(p.slots)
	p.slots = append(p.slots, newSlot)
	p.slotsByKey[key] = newIdx

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
	slotIdx, ok := p.slotsByKey[gtKey(p.titleSlug, gamertag)]
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

// Cooldown per-token sur 429 (rate-limit d'API imputable à UN token précis).
const (
	// perToken429BaseCooldown : durée par défaut de mise à l'écart d'un token
	// rate-limité quand le serveur ne fournit pas de Retry-After.
	perToken429BaseCooldown = 2 * time.Second
	// perToken429MaxCooldown : plafond d'un cooldown per-token.
	perToken429MaxCooldown = 60 * time.Second
)

// Refresh proactif near-expiry : la sonde de fond rafraîchit un token SAIN dont
// l'expiry approche, pour qu'aucune requête ne DÉCOUVRE l'expiration par un échec.
const (
	proactiveRefreshThreshold   = 15 * time.Minute
	proactiveRefreshMinInterval = 60 * time.Second
)

// aimdRestorePerTick : pas de restauration additive du débit d'un compte sain
// par tick du refresher (10s) → ~+0,6 RPS/min, plafonné au nominal (ratebudget).
// Lent volontairement : après un 429, on re-teste la limite en douceur.
const aimdRestorePerTick = 0.1

// acquirableForRestore : le compte du slot mérite une restauration AIMD —
// sain (pas de 401/403) ET hors cooldown 429.
func (s *slot) acquirableForRestore() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy && !time.Now().Before(s.rateLimitedUntil)
}

// xuidSnapshot lit le xuid sous lock (AddOrUpdateSource peut le réécrire).
func (s *slot) xuidSnapshot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.xuid
}

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

	// Si le cooldown précédent a EXPIRÉ, on en est sortis sainement → repartir du
	// palier de base (ne pas dépendre du seul tick 10s du refresher pour reset le
	// backoff). Sans ça, un 429 frais juste après l'expiration réutiliserait un
	// consecutive429 gonflé → cooldown de 5min au lieu de ~30s.
	if p.coolingDown && !time.Now().Before(p.cooldownUntil) {
		p.coolingDown = false
		p.consecutive429 = 0
	}

	// Durée de cooldown : max(Retry-After, backoff exponentiel), planchée à
	// globalCooldown. Le backoff N'EST PLUS neutralisé par la présence d'un
	// Retry-After (ancien bug : Retry-After=1s remettait consecutive429=0 → thrash
	// loop d'1s à vie). Des rate-errors répétées escaladent donc toujours, même
	// quand le serveur renvoie un petit Retry-After. NB : consecutive429 n'est
	// incrémenté QUE dans les branches qui posent/prolongent réellement le cooldown
	// (pas dans la branche « ignoré pendant un cooldown plus long ») — sinon un
	// burst de 429 concurrents sur-escaladerait le backoff jusqu'à maxCooldown.
	honored := retryAfter > 0
	shift := p.consecutive429
	if shift > maxBackoffShift {
		shift = maxBackoffShift
	}
	dur := p.globalCooldown << shift
	if retryAfter > dur {
		dur = retryAfter
	}
	if dur < p.globalCooldown {
		dur = p.globalCooldown // plancher : jamais sous le cooldown de base
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
			p.consecutive429++ // escalade uniquement quand on prolonge réellement
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
	p.consecutive429++ // escalade uniquement quand on déclenche réellement
	recordCooldownMetrics(statusCode, honored)
	lastCooldownSeconds.Set(int64(dur.Seconds()))

	slog.WarnContext(context.Background(), "pool: cooldown global déclenché",
		"status", statusCode, "cooldown_s", dur.Seconds(), "retry_after_honored", honored)

	// Marquer tous les tokens comme malsains (non-bloquant). Snapshot de p.slots
	// sous slotMu (append concurrent possible) — ordre cooldownMu→slotMu, sans cycle.
	p.slotMu.RLock()
	slots := p.slots
	p.slotMu.RUnlock()
	for _, slot := range slots {
		slot.mu.Lock()
		slot.healthy = false
		slot.mu.Unlock()
	}

	slog.InfoContext(context.Background(), "pool: tous les tokens marqués malsains (cooldown)",
		"count", len(slots))
}

// On429ForToken implémente Pool.On429ForToken : cooldown TEMPOREL borné sur le seul
// token fautif, sans cooldown global ni re-exchange. Le token reste valide (un 429
// est un throttle d'API, pas un échec d'auth) — il est juste skippé par l'acquisition
// jusqu'à expiration du cooldown, puis redevient acquérable seul. Les autres tokens
// continuent de servir : fini le scorched-earth où un 429 isolé mettait les 7 en pause.
func (p *poolImpl) On429ForToken(gamertag string, retryAfter time.Duration) {
	if gamertag == "" {
		// Sans token identifiable, filet global borné plutôt qu'ignorer le signal.
		p.OnHTTPError(429, retryAfter)
		return
	}
	p.slotMu.RLock()
	slotIdx, ok := p.slotsByKey[gtKey(p.titleSlug, gamertag)]
	p.slotMu.RUnlock()
	if !ok {
		// Gamertag hors pool (source retirée) : ne PAS nuke le pool, juste tracer.
		slog.DebugContext(context.Background(), "pool: On429ForToken gamertag inconnu, ignoré",
			"gamertag", gamertag)
		return
	}

	cooldown := retryAfter
	if cooldown <= 0 {
		cooldown = perToken429BaseCooldown
	}
	if cooldown > perToken429MaxCooldown {
		cooldown = perToken429MaxCooldown
	}
	until := time.Now().Add(cooldown)

	p.slotMu.RLock()
	slot := p.slots[slotIdx]
	p.slotMu.RUnlock()
	slot.mu.Lock()
	if until.After(slot.rateLimitedUntil) {
		slot.rateLimitedUntil = until
	}
	xuid := slot.xuid
	slot.mu.Unlock()

	// AIMD (sujet 2 T2) : multiplicative decrease — le débit du COMPTE est
	// divisé par 2 (plancher ratebudget.minRPS). Comme le limiteur est partagé
	// par-xuid (T1), le ralentissement s'applique immédiatement à TOUS les
	// consommateurs du compte (pool, career_live, worldenrich). La restauration
	// additive se fait au tick du refresher (comptes sains uniquement).
	newRPS := ratebudget.HalveRPS(xuid)

	slog.WarnContext(context.Background(), "pool: token rate-limité (429) — cooldown per-token + AIMD",
		"gamertag", gamertag, "cooldown_s", cooldown.Seconds(), "rps_after_halve", newRPS)
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

			// Snapshot de p.slots sous RLock : AddOrUpdateSource peut append
			// (réalloc du backing array) concurremment (re-scan Discovery 15min).
			p.slotMu.RLock()
			slots := p.slots
			p.slotMu.RUnlock()

			// Parcourir les slots : refresh les malsains (réactif) ET, PROACTIVEMENT,
			// les tokens sains dont l'expiry approche — pour qu'aucune requête ne
			// découvre l'expiration par un échec (sonde de santé de fond).
			// AIMD (sujet 2 T2) : restauration additive du débit des comptes SAINS
			// et hors cooldown 429 (+aimdRestorePerTick par tick 10s, plafonnée au
			// nominal par ratebudget).
			for _, sl := range slots {
				if sl.acquirableForRestore() {
					ratebudget.RestoreStep(sl.xuidSnapshot(), aimdRestorePerTick)
				}
			}
			for _, sl := range slots {
				sl.mu.RLock()
				healthy := sl.healthy
				lastAttempt := sl.lastAttempt
				var expiresAt time.Time
				if sl.resolved != nil {
					expiresAt = sl.resolved.ExpiresAt
				}
				sl.mu.RUnlock()

				// Proactif : token SAIN proche de l'expiry, throttlé pour ne pas
				// re-exchanger en boucle si l'échange échoue.
				nearExpiry := !expiresAt.IsZero() && time.Until(expiresAt) < proactiveRefreshThreshold
				proactive := healthy && nearExpiry && time.Since(lastAttempt) > proactiveRefreshMinInterval

				if !healthy || proactive {
					// reason distingue le refresh réactif (token 401/403 mort) du refresh
					// PROACTIF de la sonde de fond — pour qu'un opérateur voie dans logs/
					// que la sonde tourne (répond à la demande « savoir en background si
					// un token est sain »).
					reason := "reactive"
					if proactive {
						reason = "proactive"
						slog.DebugContext(baseCtx, "pool: sonde santé — refresh proactif near-expiry déclenché",
							"gamertag", sl.gamertag, "expires_in_s", time.Until(expiresAt).Seconds())
					}
					sl.mu.Lock()
					sl.lastAttempt = time.Now()
					sl.mu.Unlock()
					// Asynchronously refresh sans bloquer la loop. On passe le *slot
					// capturé (pas d'index) → aucune lecture de p.slots hors lock.
					go func(s *slot, gt, reason string) {
						ctx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
						defer cancel()

						refreshed, err := p.resolver.Refresh(ctx, gt)
						if err != nil {
							// Échec permanent récent servi par le cache négatif :
							// Debug (la loop repasse toutes les 10s, le WARN initial
							// a déjà été émis par le resolver).
							if errors.Is(err, ErrPermanentAuthFailure) {
								slog.DebugContext(ctx, "pool: refresh court-circuité (échec permanent récent)",
									"gamertag", gt, "reason", reason)
								return
							}
							slog.ErrorContext(ctx, "pool: refresh échoué",
								"gamertag", gt, "reason", reason, "err", err)
							return
						}

						// Mettre à jour le slot (pointeur capturé, pas de p.slots hors lock).
						s.mu.Lock()
						s.resolved = refreshed
						s.healthy = true
						s.lastRefresh = time.Now()
						s.mu.Unlock()

						slog.InfoContext(ctx, "pool: token rafraîchi avec succès",
							"gamertag", gt, "reason", reason)
					}(sl, sl.gamertag, reason)
				}
			}
		}
	}
}
