# ADR 0035 — Player directory: one identity key, one onboarding path, one read model

**Status**: Accepted (2026-09-15). Amended at closure (2026-09-16) after two adversarial
review rounds — see the *Amended at closure* notes under D2, D3, D5, D6 and the Outcome section.

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

The port exposes `List`, `Get(key)`, `Onboard(req)` and `Purge(key, opts)`. The storage behind
the port may be unified later without touching any consumer; that is the purpose of the port,
not a promise to do it.

*Amended at closure (2026-09-16, adversarial review round 1):* the port does **not** expose
`HasTrackedProfile`. The single definition of "tracked" is `config.AppConfig.HasTrackedProfile`
(`domain.SyncablePlayers`, looked up by xuid); the gates of D3 read it directly and the
directory reuses it through its `ProfilesReader` — a delegating method on the port had no
production caller and was removed as dead code. `Get`/`Purge` take a *key*: a xuid, or the
gamertag of an identity that has **no** xuid anywhere (an orphan player directory), never the
gamertag of an identity that has one. When two accounts in `users.json` carry the same xuid
(a real production case: a password account and an SSO account for the same person), the
first one read is `Account`, the others are `DuplicateAccounts`, and the record carries the
warning anomaly `account_duplicate`; nothing is overwritten or hidden.

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

*Amended at closure (2026-09-16, adversarial review round 1):* the gate cuts both ways. Pausing
a title (`PATCH /profiles/{slug}/titles/{title}/sync {enabled:false}`) or purging it (`DELETE
.../data`) removes the (xuid, title) couple from the watcher (`Daemon.RemovePlayerTitle`), and
re-enabling it adds the couple back. Without that, the poller of a paused title survived until
the next restart and every match it detected was refused by the gate — incrementing
`sync_refused_no_profile`, the counter meant to signal an unknown identity, and raising
`watched_without_profile` in the directory. A legitimate administrative action must never trip
the intrusion signal.

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

*Amended at closure (2026-09-16):* the lock is also exposed as an admin switch on the
Management page (`PATCH /settings {instance_locked}` already existed, no page used it — the
production instance had to be locked by editing the file). When the lock is forced by the
environment (`LEVELUP_INSTANCE_LOCKED`), a request to lift it from the settings is refused with
`409 instance_lock_forced` and nothing is written: the file cannot open what the environment
closes, and the switch says so instead of silently snapping back.

### D6 — Purge never touches the shared warehouse

`PlayerDirectory.Purge(xuid)` removes, in this order: live tracking, profile entries for every
title (and the player directories, handles evicted first), credentials, group memberships, the
account. It refuses to purge an admin, and it is dry-run unless confirmed. Matches already
persisted in `shared_matches_v2.duckdb` are never deleted by a purge: they carry other players'
data (opponents, teammates) and the shared warehouse is append-only by design (ADR 0026). A test
asserts the shared file's bytes are unchanged by a purge.

*Amended at closure (2026-09-16):* a purge removes **every** account carrying the xuid
(principal and duplicates) and refuses if **any** of them is an administrator; an identity
without a xuid (an orphan player directory) is purged by its gamertag and only its directories
are touched; a directory that cannot be removed is logged with the OS error before the report
marks it as not removed.

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

## Outcome (closure, 2026-09-16)

Delivered on `wt/player-directory` in seven steps (three implementation agents, one at a time,
then the pilot's closure). Production was locked by hand on 2026-09-15 19:44 UTC (in-place
edit of `app_settings.json`, same inode, backup kept) before any code shipped.

- Ratchets added: `no_bare_instance_lock_read`, `no_direct_profile_create`,
  `no_duckdb_import_playerdirectory`, `no_users_json_literal` — the last one caught a fourth
  copy of the `users.json` literal (`cmd/admin`) the same minute it was written.
- Adversarial review, two rounds with fresh contexts: round 1 (access / anti-patterns /
  multi-title) raised 5 admissible findings, all P1, all fixed — the most consequential being
  that pausing or purging a title left its poller alive and, with the new gate, tripped the
  intrusion counter; round 2 (tests / front + re-verification of the 8 fixes) raised 0 P0/P1,
  2 P2 and 3 reserves, all fixed or accepted with a test. The loop converged (5 → 0).
- Pilot review register: `.ai/REVUE_ANNUAIRE_JOUEURS_2026-09-15.md` (R1-R6, A1-A8, B1-B5).
- Gates at rest: `go test ./...` green (3 min 34 s warm; the first cold pass of
  `internal/sync` needs `-timeout 30m`, 501 s), `-tags=integration ./internal/sync/...
  ./internal/persist/...` green, `golangci-lint --new-from-merge-base` 0 issue, `tsc` clean,
  vitest 717 files / 7 707 tests green, `openapi-check` clean, no hard-coded colour.
- Left open, on purpose: a group *owned* by a purged identity is reported as a failed purge
  step rather than transferred or deleted (product decision pending); three local
  `token_orphan` fixtures from 2026-08-20; the settings toggle still trusts `sess.Role`
  (pre-existing, plan §10); the invitation path that gives a beta-tester a way in on a locked
  instance is the sibling plan's job.
