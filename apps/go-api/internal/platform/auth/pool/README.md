# Token Pool — Parallel Halo API Sync

Package `pool` manages a shared pool of Halo API tokens with two acquisition policies, automatic refresh, and HTTP error backoff.

## Architecture

### Four Layers

1. **Discovery** (`discovery.go`)
   - Scans the MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json) without network validation
   - Single source since ADR 0023 Phase 5 (2026-08-25): no env var, no sync_meta, no mono-user store
   - Returns `[]CredentialSource` (gamertag, xuid, refresh token)

2. **Resolver** (`resolver.go`)
   - Exchanges `CredentialSource → ResolvedTokens` (Spartan + Clearance)
   - Caches tokens for ~3h30 (Spartan token lifetime ~4h)
   - Pipeline: `TryOAuthRefreshWithRotation(RefreshToken)` → `provider.Exchange(accessToken)`

3. **Pool** (`pool.go`)
   - Maintains N tokens alive with round-robin or pinned access
   - Goroutine refresher auto-reactivates unhealthy tokens
   - Global cooldown (30s default) on 429/503 errors

4. **Client Adapter** (`../../../sync/pooled_client.go`)
   - Implements `sync.HaloClient` interface
   - Uses `PolicyAnyPublic` for public endpoints (round-robin)
   - Uses `PolicyPinnedPlayer` for privacy-gated endpoints (token owner only)

---

## Usage

### Creating a Pool

```go
// Scan the MultiUserTokenStore for credentials
discovery := pool.NewDiscoveryWithStore(cfg, resolver, titleSlug, tokenStore)
sources, err := discovery.Scan(ctx)
if err != nil {
    return err
}

// Create resolver with caching
provider := auth.NewSISUProvider()
resolver := auth.NewResolver(provider, 0) // 0 = default TTL ~3h30

// Build pool
pool, err := auth.NewPool(ctx, resolver, sources, auth.PoolOptions{
    MaxSize:     0, // 0 = all sources
    PerTokenRPS: 1,
})
if err != nil {
    return err
}
defer pool.Close()

// Use with sync engine
client := sync.NewPooledHaloClient(pool, gamertag, xuid)
engine := sync.NewSyncEngine(repoRoot, gamertag, xuid, &domain.HaloTokens{}, provider)
engine.SetCustomClient(client)
syncResult, err := engine.RunDelta(ctx, opts)
```

### Acquisition Policies

**PolicyAnyPublic** — Round-robin distribution for public endpoints:
- `GetMatchHistory(gamertag, ...)` — accepts any gamertag in URL
- `GetMatchStats(matchID)` — public stats
- `GetMatchFilm(matchID)` — public film

```go
lease, err := pool.Acquire(ctx, auth.PolicyAnyPublic, "")
if err != nil {
    return err
}
defer lease.Release()
client := halo.NewHaloAPIClient(lease.Tokens)
stats, err := client.GetMatchStats(ctx, matchID)
```

**PolicyPinnedPlayer** — Token of specific player only:
- `GetCareerRank(xuid)` — privacy-gated to token owner
- Fails gracefully with `(nil, nil)` if token absent or stale

```go
lease, err := pool.Acquire(ctx, auth.PolicyPinnedPlayer, gamertag)
if err != nil {
    // gamertag has no token or token is unhealthy → skip silently
    return nil, nil
}
defer lease.Release()
client := halo.NewHaloAPIClient(lease.Tokens)
rank, err := client.GetCareerRank(ctx, xuid)
```

---

## HTTP Error Backoff

### Global Cooldown (429 / 503)

When the Halo API returns:
- **429 (Too Many Requests)** — rate limit exceeded
- **503 (Service Unavailable)** — API degraded

The pool immediately:
1. Marks **all slots unhealthy** (no new Acquire succeeds)
2. Suspends refresher goroutine for `GlobalCooldown` (default 30s)
3. After cooldown expires, refresher resumes and reactivates tokens

```go
// In PooledHaloClient
if statusCode == 429 || statusCode == 503 {
    pool.OnHTTPError(statusCode) // Non-blocking
}
```

### Per-Token Refresh (401 / 403)

When a specific token returns 401/403:
1. `MarkUnhealthy(gamertag, error)` — exclude token temporarily
2. Refresher goroutine calls `Resolver.Refresh(gamertag)` asynchronously after GlobalCooldown
3. On success, token is reactivated automatically

```go
if statusCode == 401 || statusCode == 403 {
    pool.MarkUnhealthy(lease.Gamertag, errors.New("401 unauthorized"))
}
```

---

## Configuration

### PoolOptions

```go
type PoolOptions struct {
    MaxSize         int           // 0 = all sources, 1+ = limit pool size
    PerTokenRPS     int           // requests/sec per token (total = PerTokenRPS × Size())
    RefreshInterval time.Duration // re-exchange before expiration (default 3h30)
    GlobalCooldown  time.Duration // delay after 429/503 (default 30s)
}
```

### CLI Flags

```bash
# Create pool with auto-detected tokens
levelup sync-delta --all --token-pool-size 0

# Disable pool (use standard serial flow)
levelup sync-delta --all --token-pool-size 1

# Limit pool to 2 tokens
levelup sync-delta --all --token-pool-size 2 --rps 1
```

---

## Implementation Notes

### Thread-Safety

- **Slots** protected by `slot.mu RWMutex` (healthy flag, resolved tokens)
- **Round-robin** uses buffered channel + simple index counter (no contention)
- **Cooldown** protected by `cooldownMu` for state checks
- **Refresher** runs in single goroutine (no concurrent refreshes on same slot)

### Order Preservation (Parallel Fetch)

When `engine.go` fetches matches in parallel via `errgroup.SetLimit(pool.Size())`:
1. Fetches can complete out-of-order
2. Inserts remain **sequential** using indexed array + mutex (maintains DB transaction order)
3. `fetchedMatch` container holds unpacked match data (pure, no DB access during fetch)
4. `insertFetchedMatch` serializes inserts to preserve event ordering in DB

### Automatic Refresh Timing

Refresher loop runs every 10 seconds and:
1. Checks if global cooldown has expired → resumes refreshes
2. Finds unhealthy slots → spawns goroutine for `Resolver.Refresh`
3. On success, marks slot healthy and updates `resolved.Tokens`

---

## Testing

### Unit Tests

```bash
go test ./internal/platform/auth/pool/... -v
```

Covers:
- Pool creation and initialization
- Round-robin fairness (TestPoolRoundRobinDistribution)
- Pinned player lookup
- MarkUnhealthy + async refresh
- Concurrent acquire without races
- Context deadline handling
- OnHTTPError for 429/503/other codes

### Integration Tests

```bash
# Full sync with pool
levelup sync-delta --all --token-pool-size 0 --max-matches 50

# Smoke test pool disabled
levelup sync-delta --all --token-pool-size 1 --max-matches 50
```

**Expected improvements**:
- 3 tokens × public endpoints = ~3× RPS throughput
- Parallel fetch within each match page
- Estimated speedup: ~3× on total sync time (bottleneck shifts to DuckDB inserts)

---

## Future Enhancements

1. **Metrics export**: Add expvar counters (acquire success/fail, refresh attempts, cooldown triggers)
2. **Adaptive cooldown**: Increase GlobalCooldown dynamically on repeated 429s
3. **Token health scoring**: Track error rates per token, rotate unhealthy tokens to back of queue
4. **Resolver cache sharing**: Share cache across multiple Pool instances (current: per-pool)
