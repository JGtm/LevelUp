# ADR 0035 — Player directory: one identity key, one onboarding path, one read model

**Status**: Accepted (2026-09-15). To be amended at the closure of
`.ai/PLAN_ANNUAIRE_JOUEURS_2026-09-15.md` (step 7) with the measured outcome.

**Branch**: `wt/player-directory` (worktree `LevelUp-wt-player-directory`, base `feat/v75`)

**Relates to**: [ADR 0023](0023-auth-tokens-single-source.md) (token store is the only
credential source, keyed by xuid), [ADR 0029](0029-multi-user-player-ownership.md) (ownership =
`user.xuid == profile.xuid`, no `owner` field), [ADR 0008](0008-db-schema-multi-title-and-xuid-global.md)
(xuid is global, every path through `PathResolver`), [ADR 0027](0027-sync-pipeline-v2-cycle-orchestrator.md)
(sync entry points). Complements `.ai/PLAN_AMIS_PAR_JOUEUR_ET_INVITATIONS_2026-09-15.md`
(invitations carry a one-shot provisioning grant); does not supersede it.

---

## Context

On 2026-07-23 at 17:44:55 UTC an unknown Xbox account signed in on the production instance
through the Xbox SSO. The instance was not locked (`app_settings.json:instance_locked=false`,
never toggled since the lock shipped on 2026-06-08) and `can_self_provision` was `true`. The SSO
strategy (`internal/service/xbox_auth_service.go`) wrote three registries — an account in
`data/auth/users.json`, credentials in `data/auth/watcher_tokens/{xuid}.json`, and a live entry in
the watcher daemon — and never the fourth: the profile in `db_profiles.json`, which only the setup
wizard (`POST /setup/players`) creates.

Two hours later the watcher saw a new match for that account and submitted a sync. The sync engine
needs only `(gamertag, xuid)`: it derived a player-DB path from the gamertag, created
`data/titles/halo_infinite/players/<gamertag>/stats.duckdb`, and persisted 25 matches with their
participants into the shared warehouse. Everything downstream resolves a player through
`cfg.LoadPlayers()` (= `db_profiles.json`): post-sync notifications and Prestige logged
"joueur introuvable", the ownership middleware would have answered 403 on every page of that
player, the scheduler never synced them again, the watcher forgot them at the next restart, and
no admin page listed them — the Management page lists accounts, the monitoring pages list
profiles, nothing lists identities.

The stores themselves are sound: an account (who may log in), credentials (secrets, rotated,
mode 0600) and a tracking profile (what to sync, per title) have different lifecycles and
different sensitivity. What is missing is not one file but three properties:

1. a single identity key — profiles are keyed by slug/gamertag, accounts by username,
   credentials by xuid;
2. a single onboarding path — the SSO writes three registries in an order that lets a sync run
   before a profile exists;
3. a single read model — every consumer picks its own file, so an account without a profile is
   visible nowhere.

The lock check itself exists in three copies (`handlers/setup.go`, `handlers/user_auth.go`,
`service/xbox_auth_service.go` via a closure built in `api/server_apiv1.go`), which is the
third-copy threshold of CLAUDE.md rule 6.

## Decision

### D1 — The xuid is the identity key

Every cross-registry resolution goes by xuid. Gamertag and slug are display values and the
file-system path component (`PathResolver.PlayerDir(title, key)`), never a join key between
registries. `db_profiles.json` keeps its slug-keyed layout (no migration): the directory resolves
`xuid -> profiles` by scanning entries, which is cheap (tens of entries) and avoids a second
source of truth.

### D2 — A `PlayerDirectory` port fronts the registries

`internal/port/player_directory.go` declares the port; `internal/service/playerdirectory/`
implements it by composing the existing stores through small reader interfaces (profiles via
`config.AppConfig.LoadPlayers`, accounts via `userstore.Store`, credentials via
`auth.MultiUserTokenStore.LoadAll`, live tracking via the watcher daemon, disk via
`PathResolver`). Domain types live in `internal/domain/identity.go`:

- `IdentityRecord{XUID, Gamertag, Profiles []ProfileRef, Account *AccountRef, Token *TokenRef,
  Watched []string, Anomalies []IdentityAnomaly}`;
- typed anomalies: `account_without_profile`, `token_orphan` (credentials with neither account
  nor profile), `player_dir_orphan` (a player directory with no profile), `watched_without_profile`,
  and informational `profile_without_account` (friends tracked by an admin — normal) and
  `profile_without_token` (the auth pool lends credentials — normal).

The port exposes `List`, `Get(xuid)`, `HasTrackedProfile(title, xuid)`, `Onboard(req)` and
`Purge(xuid, opts)`. The storage behind the port may be unified later without touching any
consumer; that is the purpose of the port, not a promise to do it.

### D3 — No sync and no live tracking without a tracked profile

`sync.Coordinator.Submit` and `watcher.Daemon.AddPlayer` take a profile gate
(`func(ctx, titleSlug, xuid) bool`, backed by `HasTrackedProfile`: the couple exists in
`db_profiles.json`, is not `auth_only`, and has `sync_enabled != false` — i.e. the same filter as
`domain.SyncablePlayers`). A request for a player without a tracked profile is refused, logged at
WARN with `xuid`, `gamertag`, `title_slug`, and counted (`sync_refused_no_profile` expvar). The
HTTP sync handler already enforces this; the watcher path did not.

The Xbox SSO strategy still creates the account and stores the credentials (it must: the
refresh token is what makes the later profile usable), but it notifies the watcher only when the
gate passes. An account without a profile is a valid, inert state whose only exit is the setup
wizard (or an invitation grant, per the sibling plan).

### D4 — One onboarding path

`PlayerDirectory.Onboard` is the only caller of `ProfileService.CreatePlayer`: it creates the
profile and its directory, then notifies the watcher when the daemon is running, and logs each
stage. `POST /setup/players` calls `Onboard`. A ratchet forbids other callers of `CreatePlayer(`.
There is no compensation for a failed watcher notification: the profile is the source of truth
and the daemon reloads profiles at boot; the failure is logged at ERROR.

### D5 — The lock is decided in one place, and the safe defaults are the enforced defaults

`authz.InstanceLocked(envLocked bool, settings SettingsLoader) bool` replaces the three copies;
`api/server_apiv1.go` builds one resolver from it and injects it everywhere. A ratchet forbids
reading `.InstanceLocked` outside `authz/`, `config/`, `platform/settings/`, `domain/` and the
settings toggle handler.

Defaults when the key is absent from `app_settings.json` and ownership is enforced
(`authz.Enforced(demoMode, authMode)`): `instance_locked = true`, `can_self_provision = false`.
When not enforced (single-user, `auth_mode=none`, demo) the current defaults stay
(`false` / `true`). An admin is exempt from both `can_self_provision` and the lock on
`POST /setup/players`: adding a friend's profile is an admin act. Existing files that already
carry the keys are not rewritten — production is locked by its administrator, not by a deploy.

### D6 — Purge never touches the shared warehouse

`PlayerDirectory.Purge(xuid)` removes, in this order: live tracking, profile entries for every
title (and the player directories, handles evicted first), credentials, group memberships, the
account. It refuses to purge an admin, and it is dry-run unless confirmed. Matches already
persisted in `shared_matches_v2.duckdb` are never deleted by a purge: they carry other players'
data (opponents, teammates) and the shared warehouse is append-only by design (ADR 0026). A test
asserts the shared file's bytes are unchanged by a purge.

### D7 — One admin read model

`GET /admin/identities` returns the directory; the Management page renders it as an
"Identités" section with the anomalies as typed badges. That is the view that would have shown
the 2026-07-23 account the same evening.

## Consequences

- An unknown Xbox account can no longer trigger a sync, consume API quota, or write into the
  shared warehouse by merely signing in.
- The 2026-07-23 identity is purged with one command, and the procedure is reusable.
- Three lock copies become one call; the fourth would be caught by the ratchet.
- The sibling invitation plan (`PLAN_AMIS_PAR_JOUEUR_ET_INVITATIONS_2026-09-15.md`, step 5.4)
  must take the lock from `authz.InstanceLocked` and the profile creation through `Onboard`.
- Not done here, deliberately: unifying the three files into one store; an admin HTTP endpoint
  for purge (CLI only); multiple xuids per account (ADR 0029, deferred).
