# ADR 0038 — Token pooling is opt-in, per player

Status: accepted (2026-10-16)

## Context

The sync runs on a pool of Microsoft tokens (`internal/platform/auth/pool`). Every endpoint the
sync client uses is acquired in `PolicyAnyPublic`, a round-robin over all tokens, so the per-account
rate limit Microsoft applies is multiplied by the pool size. Until now every player who connected
an account joined the pool, and the privacy policy said the token was used "solely to query the
official Halo API on your behalf" — which the pool contradicted: a sync request for one player
could run on another player's token.

## Decision

1. **A token enters the public round-robin only on a recorded consent.** The decision lives in the
   player's own token file (`auth.UserTokens.PoolSharing`: granted, decided_at, basis), next to the
   credential it governs. No decision means no pooling. `Discovery` is the only production
   producer of `CredentialSource` and sets `PoolExcluded` unless `SharesWithPool()`.
2. **A withheld token still serves its owner.** The slot exists; `PolicyAnyPublic` skips it
   (`slot.sharedLeaseData`), `PolicyPinnedPlayer` still reaches it. `Pool.IsWithheld` tells
   callers; `Pool.SetShared` applies a change at once, the 15-minute rescan being the safety net.
3. **Reciprocity.** A sync run for a player who withholds their token uses only that token
   (`PooledHaloClient.WithOwner`, `acquire`), at that token's own rate; the Halo 5 live sync does
   not fall back to the pool for them. Refusing costs speed, never access. Not covered, by
   design: the live-presence match poller and CLI operations, which have no owning player.
4. **The choice is asked, not defaulted.** `bootstrap.pool_sharing` is `undecided` until the
   connected user decides; the shell banner offers two equal buttons, none preselected
   (`PoolSharingBanner`). Withdrawal is in Settings, Sync tab (`PUT /me/pool-sharing`).
5. **Pre-existing players are grandfathered, explicitly.** `levelup pool-sharing grandfather`
   records `basis = grandfathered_explicit` for the players the operator names (`--players`, or
   `--all` deliberately) who have a token and no decision, only for
   agreements collected outside the application, and never overwrites a decision.
6. **The texts follow the behaviour.** Privacy policy section "Token pooling", the new terms page
   (`/terms`) and the consent texts (`features/settings/poolSharingText.ts`) say the same thing;
   a test pins the shared section and its key claims.

## Consequences

- `Upsert` merge-preserves `PoolSharing` like the refresh token: a partial write must not erase a
  consent. Only `SetPoolSharing` changes it, including to withdraw.
- If every token is withheld, `PolicyAnyPublic` returns `ErrNoHealthySlot`; owners' own syncs keep
  working through their pinned token, but profiles followed without a connected account are no
  longer synced at all, and neither are the live reads of pages. This is accepted: refusing to
  share limits what the instance can do. The terms say so.
- For a withheld owner, an unavailable own token (429 cooldown, refresh) surfaces as
  `ErrNoHealthySlot`: that token is their whole pool, so `historyretry` waits and replays instead
  of ending the pass.
- Consent changes from the application reach the pool by xuid (`Pool.SetShared`), the only
  identity key (ADR 0035): the account's gamertag may differ from the profile's.
- Withdrawing from the pool does not delete the token: it is still the player's own credential.
- Microsoft sees pool requests as coming from the owning account. The terms say so.
