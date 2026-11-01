# ADR 0044 — Deployment profile: one concept, resolved once

Status: accepted (2026-11-01); implementation tracked in `.ai/V8/PLAN_DESKTOP_2026-11-01.md` (lot L10; the pieces
that read the profile arrive with L12 to L15).
Source: decision D-16 of the plan (confirmed by the maintainer on 2026-11-01), audit worksite C7, additions A6, A7 and
A17.
Relates to: ADR 0045 (desktop mode and shell contract: the first consumer of the `desktop` profile), ADR 0008 as
amended on 2026-11-01 (the two roots, derived from the same profile), ADR 0029 and 0035 (ownership and the instance
lock, read through `authz`), ADR 0038 (pooling consent), ADR 0011/0025 (title-agnostic rule: branch on capabilities).

## Vocabulary

- **Deployment**: the way one server process is run. Four values: `hosted` (the public or private instance behind a
  reverse proxy, the VPS and the Docker image), `desktop` (the server launched by the Windows shell, single user),
  `demo` (the public read-only demo instance), `dev` (a developer's machine).
- **Resolver**: the one function of `internal/config` that turns the raw inputs into a `Deployment` and everything
  derived from it. It runs once, in `LoadServer`, and the result is a field of `AppConfig`.
- **Capability**: one fine-grained fact the rest of the code branches on ("accounts and registrations exist",
  "token pooling is offered"). A capability is decided by the resolver, never recomputed by a consumer.

## Context

Three settings describe, today, what kind of instance is running, and they are read all over the server:
`AppConfig.DemoMode` (`LEVELUP_DEMO_MODE`), `AppConfig.AuthMode` (`LEVELUP_AUTH_MODE`: `none`, `password`, `xbox`,
default `none`) and `AppConfig.Environment` (`LEVELUP_ENV`). On 2026-11-01 the non-test Go code of `apps/go-api` reads
`.DemoMode` about 90 times, `.AuthMode` about 55 to 60 times and names `LEVELUP_ENV` 13 times (the counts differ by a
few units with the pattern used; D-6 freezes them with the ratchet's own pattern). Each reader re-derives a meaning from
the combination:

- `authz.Enforced(demoMode, authMode)` is true only for `password` and `xbox` outside the demo;
- `Validate` refuses an unsafe configuration only when `LEVELUP_ENV=production` and the demo is off, and
  `SecurityWarnings` treats `AuthMode=none` as unsafe there;
- `IsExposedDeployment` reads the host and `LEVELUP_ENV` again to choose a log level;
- `demoPolicies` (`cmd/server/background_tasks.go`) declares, per background task and boot step, whether the demo keeps
  or cuts it;
- `/bootstrap` already publishes `CapabilityMap` (nine `can_*` booleans, one of them, `CanManageInstance`, constant)
  and `FeatureFlags.demo_mode`.

A fourth way of running the server arrives with the Windows application (ADR 0045): single user, no account system,
loopback only, tokens pooled for the local user, a different set of background tasks. Adding it as a fourth reader axis
(a boolean `DesktopMode` next to `DemoMode`, an `AuthMode` value set by hand, a host forced by the launcher) would make
every existing branch ask a new question and would let the combinations drift. The first version of the plan carried
three formulations of this one concept; the maintainer asked for one (D-16).

## Decision

**D-1 — One enum, four values, one resolver.** `internal/config/deployment.go` defines `Deployment`
(`hosted`, `desktop`, `demo`, `dev`) and a resolver called once by `LoadServer`. Inputs: `LEVELUP_ENV`,
`LEVELUP_DEMO_MODE`, `LEVELUP_AUTH_MODE` and the launch mode (`-mode desktop`, passed by `cmd/server` as a parameter
after its single flag parser has run: `internal/config` never reads `os.Args`). Resolution order, first match wins:
`-mode desktop` gives `desktop`; `LEVELUP_DEMO_MODE=true` gives `demo`; a `LEVELUP_ENV` other than empty or
`development` (the split `IsExposedDeployment` already uses) gives `hosted`; otherwise `dev`. `-mode desktop` together
with `LEVELUP_DEMO_MODE=true` is refused at startup with an explicit error (the two have no common meaning). The
precedence and this refusal are written by this ADR; the plan fixes the four values and the inputs.

**D-2 — Derived, never set by hand.** The resolver derives, from the profile and the raw inputs:

- `AuthMode`: for `desktop` the value `desktop`, which is not an accepted value of `LEVELUP_AUTH_MODE` (it exists only
  as a derived value; `Validate` accepts it, including under `LEVELUP_ENV=production`). For the other profiles the
  existing rules are unchanged. `authz.Enforced` stays false for `desktop` (single user: every profile of the machine is
  the user's), through the same code path as `none`.
- the listening host (A7): in `desktop` the loopback address, whatever `LEVELUP_API_HOST` says; a conflicting value is
  ignored and logged at Warn once. The other profiles keep `LEVELUP_API_HOST`.
- the fine-grained capabilities of D-3, and the data and resource roots when `desktop` (ADR 0008, amendment).
An explicit `LEVELUP_AUTH_MODE` under `-mode desktop` is ignored with a Warn, for the same reason as the host: a
leftover variable in a user's environment must not change the security model of a local application.

**D-3 — `CapabilityMap` of `/bootstrap` is extended; nothing new is added next to it (A6).** New capabilities, named in
the existing `can_*` style and fixed at L10: accounts and registrations, invitations, groups, token pooling,
administration, restic backup, Discord, outbound network, updates, Windows download. The nine existing fields keep their
meaning; the response gains a `deployment` field (the profile name, for display and diagnostics only). OpenAPI and the
generated web types are regenerated. `FeatureFlags.demo_mode` (and the top-level `demo_mode` of the bootstrap
response) stay until the web client reads capabilities instead (L10.5); they are then removed from the response in the
same change that removes their last reader, which fixes the removal date.

**D-4 — Consumers branch on capabilities, never on the profile.** Same rule as for titles (`HasCapability`, never
`slug == "..."`): the name `deployment` is published for display, and no code outside the resolver compares a
`Deployment` value. A consumer that needs a new distinction asks for a new capability.

**D-5 — Background tasks are declared per profile (A17).** `demoPolicies` becomes a policy per profile: for each
background task and boot step (Waypoint cron, Spartan cron, replay purge, disk watch, backup and the others), keep or
cut, for each of the four profiles. The rule "a task absent from the table is cut" stays; the existing test requiring
every task name to be declared is extended to the four profiles. The columns `hosted`, `dev` and `demo` reproduce
today's behaviour exactly (no change on the VPS, T3); the `desktop` column is set at L10. `taskAppRelease`, started
today under `context.Background()`, joins the shutdown context (ADR 0045, D-4).

**D-6 — A ratchet closes the door (C7).** No new read of `.DemoMode`, `.AuthMode` or `LEVELUP_ENV` outside the
resolver. The existing reads go in a dated allowlist whose count is frozen at the value measured when the ratchet is
created, and which only goes down. A read that moves into the resolver leaves the list; a new one fails the build.
This is the "centralise, then forbid the old literal" pattern of rule 6 of `CLAUDE.md`, and extends the intent of
`config/env_centralization_test.go`.

**D-7 — The demo keeps its own layout.** The `demo` profile selects the existing hermetic layout (`title.DemoLayout`,
state under the demo root, `config_demo.go`); the resolver reads `LEVELUP_DEMO_MODE` through the function that is today
its single source (`demoModeFromEnv`) and does not change what a demo instance reads or writes.

## Consequences

- **One place to read, one to change.** A new way of running the server is a new value and a new column in D-5's table,
  not a new axis in 160 call sites. The combinations that have no meaning (`desktop` with the demo) are refused where
  they are formed.
- **No behaviour change outside `desktop`.** Witness T3 (the `/bootstrap` body in `none`, `xbox`, demo and frozen
  modes) is identical before and after L10 except for the `deployment` field (`jq 'del(.capabilities.deployment)'`).
  The existing mode combinations map to the same capabilities (test of L10.6).
- **Cost.** The 160 existing reads are not migrated by this ADR: they stay on the allowlist and move when their code is
  next touched. The front end keeps `demo_mode` and `auth_mode` readers (66 occurrences in 23 files on 2026-11-01)
  until L10.5.
- **`LEVELUP_ENV=production` no longer forces the unsafe-configuration refusal on `desktop`.** The check stays tied to
  `hosted`; `desktop` is safe by construction (loopback, secret per launch, ADR 0045).
- **Not covered.** Per-capability authorisation of a request stays with `authz` and the existing middlewares; the
  profile only decides which features exist.

## Guard-rails

In `apps/go-api/internal/archlint/` unless a path says otherwise. The tests below are created by L10 and are named here
so that the plan and the code agree; the file names are fixed at L10.

- `deployment_single_resolver_test.go` (to create, L10.4): no new `.DemoMode` / `.AuthMode` / `LEVELUP_ENV` read outside
  `internal/config/deployment.go` and the dated allowlist; the allowlist count is frozen and the test fails when an
  entry is useless (the pattern of `bare_routes_ratchet_test.go`); no comparison with a `Deployment` value outside the
  resolver (D-4).
- `internal/config/deployment_test.go` (to create, L10.6): every combination of today's settings yields the capabilities
  of today (T3); the `desktop` profile forces the loopback host whatever `LEVELUP_API_HOST` holds; `-mode desktop` with
  the demo is refused; a `LEVELUP_AUTH_MODE` under `desktop` is ignored.
- `cmd/server/background_tasks_test.go` (exists, extended at L10.3): every task and boot step is declared for each of
  the four profiles.
- `internal/config/config_size_ratchet_test.go` (exists): `config.go` is frozen in size, which is why the resolver is a
  file of its own.
- `internal/archlint/no_slug_comparison_test.go` (exists): the title-agnostic ratchet this ADR mirrors for profiles.
- Web, L10.5: the 66 reads of `demo_mode|demoMode|auth_mode|authMode` read capabilities; `make generate-types` shows no
  diff outside `deployment`.

## Alternatives rejected

- **A boolean `DesktopMode` next to `DemoMode`.** A fourth reader axis: each existing `if DemoMode` would have to decide
  what `DesktopMode` means for it, and `desktop` with the demo would be representable.
- **A new `AuthMode` value and a host set by the launcher.** Two settings that must agree, written by two different
  actors (the shell and the user's environment); a mismatch exposes a no-account server on a network interface.
- **One environment variable per feature** (`LEVELUP_ACCOUNTS=false`, `LEVELUP_POOLING=true`, ...). The profile is the
  unit the product reasons about; per-feature variables recreate the combinatorial space and make the T3 witness
  meaningless.
- **Leaving the three readers as they are and documenting the desktop combination.** Rejected by D-16: the three
  formulations in the first plan were exactly that, and each had a different answer to "what is a desktop instance".

## References

- Plan: `.ai/V8/PLAN_DESKTOP_2026-11-01.md` (D-16, C7, A6, A7, A17, §9.10).
- Code: `apps/go-api/internal/config/config.go` (`DemoMode`, `AuthMode`, `Environment`, `Validate`,
  `IsExposedDeployment`), `config_auth.go` (`knownAuthModes`), `config_demo.go`, `config_desktop.go`;
  `internal/authz/authz.go` (`Enforced`, `InstanceLocked`); `internal/domain/bootstrap.go` (`CapabilityMap`,
  `FeatureFlags`); `internal/service/bootstrap_service.go` (`buildFeatureFlags`); `cmd/server/background_tasks.go`
  (`demoPolicies`).
- ADR 0045 (desktop mode), ADR 0008 (amendment of 2026-11-01), ADR 0038 (pooling consent).
