# ADR 0045 — Desktop mode and the shell-server contract

Status: accepted (2026-11-01); implementation tracked in `.ai/V8/PLAN_DESKTOP_2026-11-01.md` (P0 delivered, lot L16 delivered on the branch,
lot L17 to come; each decision below says which).
Source: decisions D-1 to D-17 of the plan, confirmed by the maintainer on 2026-11-01 (D-8, D-9 and D-10 against the
recommendation); additions A1 to A24; section 6 of the plan (the contract).
Amends: ADR 0023 (D-11 below: the token file may be encrypted by a `TokenCipher` port, the single source stays the
single source); the Go/TypeScript-only rule of `CLAUDE.md` (D-18: Rust limited to `apps/desktop/src-tauri`).
Relates to: ADR 0044 (the `desktop` deployment profile), ADR 0008 as amended on 2026-11-01 (the two roots), ADR 0035
(no sync without a tracked profile), ADR 0038 (pooling consent), ADR 0013/0016 (one writer per database), ADR 0040
(languages are data), ADR 0041/0042/0043 (the bricks; the shell does not touch them).

## Vocabulary

- **Shell**: the Windows application in `apps/desktop` (Tauri 2). It owns the window, the updater and the installer; it
  holds no business logic.
- **Sidecar**: the Go server, the same binary as the hosted one (`cmd/server`), launched by the shell as a child
  process with `-mode desktop`.
- **Clean stop**: the single observable of section D-5.
- **Fake sidecar / fake launcher**: the test doubles of D-21, one for each side of the contract.

## Context

The goal is an installable, single-user, autonomous Windows application that runs the existing server and the existing
web interface, closes without leaving a write-ahead log, updates itself, and **does not fork the server**. Three facts
shape the contract. Windows has no usable stop signal for a process without a shared console (`os.Interrupt` does not
reach it, `SIGTERM` does not exist), and air's `TASKKILL /F /T` left WAL files on the maintainer's machine (witness T6).
Browser storage is keyed by origin, which includes the port: a random port empties `localStorage` and IndexedDB at every
launch (297 occurrences in 85 web files) while cookies, not partitioned by port, would leak to any local service. And a
database held by a killed process stays locked for a few seconds, so a relaunch after a forced stop is a normal case.

## Decision

### The contract (plan section 6)

**D-1 — Launch.** The shell starts the sidecar with the working directory set to the resources root, the environment
`LEVELUP_DATA_DIR`, `LEVELUP_RESOURCES_DIR`, `LEVELUP_LOGS_DIR` and `LEVELUP_DESKTOP_SECRET=<random>` (never an argument,
never logged), and the arguments `-mode desktop -arret-sur-stdin`. The port is read from `<DataRoot>/desktop/port` (D-9).
State: the single flag parser `cmd/server/flags.go` exists and recognises `-health-check`, `-arret-sur-stdin` and `-mode`;
`-mode desktop` is accepted since L12 and `-mode refresh-only` since L15 (both reach `config.LoadServer(launchMode)`; the two
resolve the `desktop` profile).

**D-2 — Events on stdout.** One JSON line per event, written by a single writer under a lock; in this mode every console
log goes to stderr. `{"event":"listening","port":N}` when the port is bound (before the migrations);
`{"event":"ready","port":N}` when the router opens (`bootServer.openRouter`);
`{"event":"failed","state":"migration_failed"|"schema_newer"|"locked"}`; `{"event":"stopping"}`;
`{"event":"stopped","clean":true|false}`. `locked` has two causes: a database file held by another process after the
bounded retry, and the instance lock of D-10. The first three states come from the migration step, which is fatal: a
failed step, a database whose schema is newer than the binary (`migration.InspectSchema`, `ErrSchemaNewerThanBinary`,
read without a write connection) or a lock that outlives the bounded retry refuse the start instead of serving data the
code does not understand, and a copy of each database about to be upgraded is taken first (`BackupBeforeUpgrade` into
`PathResolver.MigrationBackupsDir()`, two kept, refused when the free space is short). State: all of this exists, with
the 503 `server_start_failed`, `retryable:false` answer of the startup gate (`bootFailureState` in
`cmd/server/boot_failure.go`). The stdout writer (`cmd/server/desktop_events.go`, active in `-mode desktop` only) and
every event are delivered by L12; `failed` also carries the state `port` when no port can be bound (nothing serves the
gate then, the event is the only witness). The console log handler writes to stderr in every mode, so stdout carries
events only. Two more events (`notification`, `shell_settings`) and the commands the shell writes on stdin are defined by D-23 (L21b). `stopped` is written last, after the journals are closed, the token pool is closed and the instance lock is
released, so a relaunch triggered by `stopped` finds the lock free. The shell drains stdout continuously (a full pipe
blocks the server) and relays stderr to `<LogsRoot>/desktop.log`.

**D-3 — First page and access.** The shell shows an embedded screen (logo) until `ready`, then loads
`http://127.0.0.1:N/desktop/enter?secret=<random>`; the server sets the session cookie and redirects to `/`. The cookie
name carries the port (A15) and lasts 30 days, renewed at each entry. Without the cookie every route other than
`/desktop/enter` and `/health` answers 401 (one exception, under the capability `LoginViaBrowser`: the return of the
sign-in through the default browser, D-22; `/health` stays open because the shell and the end-to-end script probe it
before any cookie exists, and it carries no data); the `Host` header must be the exact loopback authority of the bound
port (otherwise 421, logged) and CORS allows that exact origin only. On `failed` the shell loads the page of the startup
gate, which answers before the router opens and says the state in a non-retryable 503. State: delivered by L13.
The whole entry hangs on one capability, `ShellEntryGate` (`internal/config/deployment.go`), so no consumer compares the
profile name. `middleware.DesktopGate` (`internal/api/middleware/desktop_gate.go`) is mounted right after
`PreserveRemoteAddr`: Host first (421 `desktop_host_refused`, `/health` and `/desktop/enter` included, which is what cuts
DNS rebinding), then the cookie (401 `desktop_unauthorized`, `retryable:false`, a path that `AmbiguousPath` flags never
passes as `/health`). The cookie is not a session: its value is `<expiry>.<hex HMAC-SHA256>` over `(port, expiry)` with
the session secret of D-7, checked without state, valid 30 days and renewed at each entry; it is `HttpOnly`,
`SameSite=Lax` (not Strict: the entry navigation is started from the embedded page, origin `levelup-shell://`, which is
another site for Chromium, so the redirect to `/` is a cross-site top-level navigation on which a Strict cookie is not
sent and `/` would answer 401; Lax travels on top-level navigations only, mutations stay behind the exact-origin CSRF
check, the strict `Host` and the loopback), not `Secure` (plain http on loopback). Its name is `levelup_session_<port>`. `GET /desktop/enter` sits
behind `LoopbackOnly`, compares `?secret=` with `LEVELUP_DESKTOP_SECRET` in constant time (403 wrong, 404 when the
launcher gave no secret), never logs it, and answers 302 to `/`.
**The entry secret is single-use per launch** (review of L16, 2026-11-02): WebView2 writes every URL it
visits, redirect sources included, into its history database
(`%LOCALAPPDATA%info.lvelup.desktopEBWebViewDefaultHistory`), so `/desktop/enter?secret=<64 hex>` stays on
disk in the clear. Measured on a launched shell: one `urls` row and one visit carry the secret, with a plain
navigation as well as with `location.replace` from the embedded page (the latter was also what exposed the Strict
cookie problem above: 401 on `/`). No setting of the window avoids
the record without losing the browser storage that A15 protects (a non-persistent profile empties it at each launch),
and clearing only the history needs a WebView2 interface that wry does not expose. Mitigation, which makes the record
harmless: `EnterHandler` accepts the correct secret once (an atomic swap, 403 and a `secret d'entrée déjà consommé`
warning afterwards, a wrong secret does not consume it), the shell loads the entry once per launch (a repeated
`ready` does not reload it, `lifecycle` test), the secret is drawn again at every launch, and the cookie carries
the session. What survives in the history is a dead secret of a previous launch. CORS and the CSRF origin list are
`AppConfig.SetBoundPort`'s output: the single origin `http://127.0.0.1:<bound port>`, `LEVELUP_CORS_ORIGINS` ignored.
Two exceptions and one extension, found by the wave 4 review: `POST /api/v1/internal/shutdown` is exempt from the
cookie (the shell cannot forge a signed `HttpOnly` cookie; the route keeps `LoopbackOnly`, the constant-time header
secret and the strict `Host`), and the startup gate, which answers before the router and its guard exist, is wrapped by
the same strict `Host` check (`middleware.StrictHost`, 421) without any cookie, since the shell loads the startup or
failure page before it has one. Cooking children (`internal/replaychild`) relay their output on stderr, never on the
server's stdout, which carries the D-2 events only (`internal/archlint/server_stdout_internal_test.go`).
The bare-route allowlist gets no entry: `/health` is already listed, and `/desktop/enter` is guarded by a recognised
guard (an entry would be flagged useless by the ratchet's own self-check); `TestBareRoutesRatchet_DesktopEntryRoutes`
proves instead that both desktop routes exist only under the capability and that every route of the desktop router sits
behind the gate. The web client recognises the 401 code and shows a non-retried "access expired, reopen LevelUp" screen.

**D-4 — Stop.** The principal path is the closing of the sidecar's stdin (the shell closes it; the OS closes it if the
shell dies). The alternative is `POST /api/v1/internal/shutdown`, mounted only when `LEVELUP_DESKTOP_SECRET` is set,
behind `LoopbackOnly` (decided on the TCP peer), with the secret in `X-LevelUp-Shutdown-Secret` compared in constant time
(404 not configured, 403 wrong secret or non-loopback peer, 202 and an asynchronous request otherwise). Signal, stdin and
HTTP all call `bootServer.requestShutdown`, a `sync.Once`: one pass, the same 40 s budget (`shutdownBudget`). The shell
waits for `stopped` or the budget plus 5 s, and only then kills. Background work that outlives the request joins the
shutdown context and the scheduler wait group (`taskAppRelease`, the token refresh loop of the watcher, the world
leaderboard enricher: L12.3 and L12.6). There is **no Job object**
(A2): it would kill the sidecar at the death of the shell, which is exactly when the stdin EOF must trigger a clean stop.
Replay build children are killed by context cancellation (`exec.CommandContext`) and their kernel lock is released by the
OS; the residual risk is an orphan child after a hard crash of the server, bounded by the film it was decoding. State:
delivered (`shutdown_stdin.go`, `internal/api/handlers/internal_shutdown.go`, `cmd/devlaunch` for air); L12 adds the test
that a cancelled context kills the build child (`libs/film/filmproc/runner_annulation_test.go`).

**D-5 — One definition of a clean stop.** The log line `arrêt: terminé` with `clean=true` **and** no `*.wal` file under
`<DataRoot>`. In code `clean` is "every phase finished in its time, the persist queue drained without error, and the
whole sequence fit in the budget" (`runShutdown`); the absence of WAL is checked by the tests that use the definition
(L2, L6, L12, L17) and by the end-to-end script. Nothing else is called a clean stop.

**D-6 — Downloads.** The WebView download handler (`on_download`) opens the native save dialog. No change in `apps/web`.

### Desktop mode (server side)

**D-7 — Single user.** The `desktop` profile (ADR 0044) is single-user: `authz.Enforced` is false, every profile of the
machine is the user's. The session secret is generated at the first launch and written `0600` to
`<DataRoot>/auth/session_secret` (mode bits are advisory on Windows; the file sits under the user's profile); the public
default secret is never used. A Microsoft sign-in by device code persists the refresh token for a tracked profile in
every authentication mode, `none` included (`service.AuthCompletion`, formerly `DeviceCodeCompletion`: the single arrival point of every Microsoft sign-in, D-22; the profile gate of ADR 0035; if the profile
does not exist yet, the tokens are kept in memory and replayed when the directory creates it); this is what keeps a
single-user installation syncing after the first hour. The listening host is forced to loopback (ADR 0044 D-2).
`/api/v1/desktop/strings` serves the shell's strings from `internal/i18n/messages/desktop.toml` in the request's
language (D-13). State: delivered by L13. `config.Validate` accepts the profile under `LEVELUP_ENV=production` (the
production refusal is the hosted profile's) but refuses, in `desktop` as in production, the public default secret or one
shorter than 32 bytes (`validateStartup`); `LEVELUP_SESSION_SECRET` is ignored with a warning in this profile; a file that
is empty, too short or equal to the public default is replaced with a warning, an unreadable one fails the start. The
route answers `{locale, strings}` with nine identifiers (`starting`, `failed_locked`, `failed_migration_failed`,
`failed_schema_newer`, `failed_port`, `failed_timeout` (the shell's own start-up deadline, L16), `stopping`, `stopped`, `update_available`); the last carries the `{version}`
placeholder untouched, the shell knows the number. It is mounted only under `ShellEntryGate` and requires the cookie.

**D-8 — Pooling for the local user (plan D-4).** In `desktop` the pooling decision is recorded as granted for the local
user, with a basis value of its own, and the consent banner is hidden by the capability. This is an exception to
ADR 0038 D4 (the choice is asked, not defaulted), and it holds because the only token in the pool is the user's own and
never leaves the machine. Without a decision the rule of ADR 0038 stays: refusal. State: delivered by L13. The capability
`PoolGrantedToLocalUser` (distinct from `PoolTokens`, which says the choice is *asked*) makes
`AuthCompletion.WithLocalPoolGrant` record `granted` with the basis `desktop_single_user` when a refresh token is
written for a token file that carries no decision; an existing decision (grant or refusal) is never overwritten and an
unreadable file gets no grant. The banner needs no new code: `PoolTokens` is already false in `desktop`, and
`PoolSharingBanner` reads `can_pool_tokens`. Known limits: a token written by the CLI (`token-import`, `token-capture`)
gets no grant; the in-memory pool learns the grant at its next rescan (at most 15 minutes), the owner's own token serving
its owner meanwhile (ADR 0038 D2).

**D-9 — Stable port (plan D-13, A15).** The port is stored in `<DataRoot>/desktop/port`; if it is taken, a free port is
chosen, written, and a warning is logged (the browser storage of that user is then lost once). Never `:0` without that
fallback. `startBootServer` receives the bound listener instead of exiting; the `os.Exit(1)` on a taken port stays outside
`-mode desktop`, where a listen failure is announced `failed{state:"port"}`. State: delivered by L12
(`cmd/server/desktop_port.go`, `PathResolver.DesktopPortPath`); the configuration's `APIPort` is set to the bound port. An
instance that lost the instance lock (D-10) binds a free port without writing the file: it must not change the port of the
instance that runs.

**D-10 — Instance lock and wake-up (A16, A10, A24).** A kernel lock `<DataRoot>/levelup.lock`
(`internal/platform/instancelock`, the pattern of `libs/film/internal/filelock`, which is not importable) is taken at
start in `desktop` and `refresh-only`; a second holder fails with `ErrInstanceDejaLancee` and `failed{state:"locked"}`.
**Amended 2026-11-03 (L22, review S2): EVERY server profile (hosted, dev, demo, desktop) takes the lock at boot** (`bindBootListener`, before the
port): there is never more than one server per data root, and `-mode restore` (D-24) must find the lock held by whatever server runs on the root it
replaces databases in, otherwise a `--force` restore would rename live databases. A second server on the same root is refused (`locked`).
The single-instance plugin of the shell protects neither the scheduled task nor a `go run`. `-mode refresh-only` is a
mode of the same binary: it takes the lock, opens no database, rotates the tokens and exits; if the lock is held it
exits 0 without doing anything (one rotation at a time, ADR 0023). Wake-up sequence, under the `desktop` capability only
(the VPS restarts at every deployment): an immediate first sync tick when the last sync is older than the interval; a
token check at boot for each tracked profile (no network: nothing is marked; dead: `reauth_required` and the banner;
alive: the rotated token is written); and the same sequence when the wall clock and the monotonic clock differ by more
than 2 minutes, because Go's monotonic clock on Windows does not count the sleep (`runtime·nanotime1` reads
`_INTERRUPT_TIME`). The cached-token deadline already drops the monotonic reading (`time.Now().Round(0)` in
`tokencache`). Lots L12, L15. State: the lock and its refusal are delivered by L12 (the lock is taken before the port,
released before `stopped`; an I/O error taking the lock refuses the start as `migration_failed`, the data root being
unusable). State of `refresh-only` and the wake-up sequence: delivered by L15. The capability `WakeSequence` (true in the
`desktop` profile only) arms `internal/wakeup` once the router is open: the token check (`deadtoken.Gate.Resolve` per
tracked profile, one profile after the other, no Halo exchange: dead = `reauth_required` and the banner, alive = rotated
token written atomically, no answer = nothing marked and a retry at the 30 s ticks with a growing delay up to 10 minutes,
the sync catch-up waiting with it), then `AutoSyncScheduler.CatchUpDue` / `RequestCatchUp` (the immediate tick runs inside
`Run`'s loop; the age of the last cycle, read back from the post-sync snapshot, is measured on the wall clock). The 30 s
tick compares the wall-clock and the monotonic advance (`SleepDetector`, gap above 2 minutes). `refresh-only`
(`cmd/server/refresh_only.go`) takes the lock before anything, opens no database and binds no port, writes nothing on
stdout, runs the same token check once and exits 0 whatever the tokens say; a held lock exits 0 without any exchange. Exit
1 only when the configuration cannot be read or the lock cannot be taken for another reason than a running instance.

**D-11 — Tokens encrypted by the Windows vault (plan D-8, amends ADR 0023).** A `TokenCipher` port in `platform/auth`
with two implementations: `plain` (hosted, Linux containers: today's behaviour) and `dpapi` (Windows,
`CryptProtectData`, current-user scope, through `golang.org/x/sys/windows`). The profile chooses, never `GOOS` alone. The
token file `watcher_tokens/{xuid}.json` carries a `cipher` field and its sensitive fields (`OAuthRefreshToken`, Spartan
and XSTS tokens) encrypted in base64. A `plain` file under a `dpapi` profile is rewritten encrypted at the next write;
the reverse is a typed, logged refusal, and an encrypted file copied to another account or machine is unreadable on
purpose. `token-import` and `token-capture` write through the same port. The single-source rule and its sentinels are
unchanged. Lot L13b.

Detail of the lot. The port lives in the leaf package `internal/platform/auth/tokencipher` (`Cipher`: `Name`, `Seal`,
`Open`; `Plain`; `NewDPAPI`, which fails with `ErrUnavailable` outside Windows). The profile decides through a new
capability, `TokenVault` (true in `desktop` only): the configuration loader calls `tokencipher.InstallForProfile` once,
from `internal/config/config_token_vault.go` (ratchet `archlint/token_cipher_single_decision_test.go`), and every
`NewMultiUserTokenStore(dir)` and `NewTokenStore(path)` read that process cipher at construction, so the server and every
command-line tool choose the same way; `NewMultiUserTokenStoreWithCipher` is for tests. The last resolved profile wins (a hosted or dev load puts
`plain` back); a load never fails on the vault (the desktop profile is exercised on Linux CI), `cmd/server`
(`tokenVaultRefusal`, after `LoadServer`, `refresh-only` included) refuses to start when the profile asks for the vault and
the system lacks it. File format: `cipher` is written for `dpapi` only (a `plain` store writes no field, so hosted files are
byte-identical to before; a missing field means `plain`); `OAuthRefreshToken`, `AccessToken`, `XSTSToken` and
`XSTSUserHash` become the standard base64 of the DPAPI blob (current-user scope, `CRYPTPROTECT_UI_FORBIDDEN`, fixed
entropy `levelup.watcher_tokens.v1`, description `LevelUp watcher token`); empty fields stay empty; identity, dates,
`pool_sharing` and the reauth state stay readable. The single-user watcher state file `data/auth/watcher_tokens.json`
(`TokenStore`: `access_token`, `xsts_token`, `xsts_user_hash`, no refresh token since ADR 0023 Phase 5) goes through the SAME
codec, cipher, `cipher` field and rules (it would otherwise keep in the clear three of the four fields sealed next to it); its
sealed fields are `AccessToken`, `XSTSToken` and `XSTSUserHash`, while `XSTSGamertag` and `XSTSXUID` stay readable. A test
lists every string field of `UserTokens` and of `StoredTokens` as sealed or declared non-sensitive. Reading: a `plain` file under a `dpapi` store is accepted and rewritten sealed at the next write, never at
load; a sealed file under a `plain` store, an unknown cipher, or a failed `Open` is `ErrTokenFileSealed` (logged once per
file AND per process, the key being the file path, because stores are rebuilt at every sync pass; every refusal is counted in
`levelup.auth.token_file_unreadable_total{sealed|decode}`; the scans and callers that used to warn or error on each read log a
sealed file at Debug only); `LoadAll` skips it and serves the other players. A partial `Upsert` (no refresh token) on an unreadable sealed file is refused rather than overwriting a possibly
recoverable credential; an `Upsert` that carries a fresh refresh token (a new sign-in) replaces it.

**D-12 — Film archive (plan D-5, A18).** Archived by default; the choice "do not archive" is offered in the first-launch
wizard (`features/setup`), never in the installer; occupancy is shown and the location is a setting. The unit of the
archive is the film: `PathResolver.FilmArchiveDir()` covers `film_chunks/` and `film_manifests/` (setting
`film_archive_dir`, a non-empty target refuses an assisted move). Without archive, facts and replay are still produced,
chunks are not kept, and the replay retention is forced to unlimited, because the purge assumes a re-cook from the film
cache; the combination "no archive, finite retention" is refused in the settings (and folded to unlimited on read, so
a hand-edited file cannot arm an irreversible purge). The disk alert is the existing `taskDiskWatch` (80 and 90 %),
which watches the archive volume with its own anti-spam state when the archive lives outside the data root.
Lot L14.

Detail of the lot. The archive root is the fourth field of `Roots` (`FilmArchive`, read once from
`app_settings.json:film_archive_dir` by `config.ResolveRoots`, absolute path only). Changing the folder in the settings
is refused while the archive holds films (the move is `levelup film archive-move`, server stopped, which records the
setting itself once every film is relocated; each film is copied to a `.part` folder of the target, then renamed in one
step, then removed from the origin, so a film is whole on one side only) and while the target is not empty (the default
location only has to hold no film). On an EMPTY archive the change applies at once (`title.SetFilmArchiveRuntime`, a
process variable read by every resolver, like the process roots): the first sync of the wizard writes to the new folder.
The disk watch keeps one watch per volume, not per path. Without
archive, a cooked film's chunks are removed by the post-sync cook right after its artifact is stored, never before. The
occupancy (`GET /settings/film-archive`) is measured once and cached for ten minutes with a 20 s walk budget.

**D-12b — Chunks stored in zstd-3 (plan D-18, 2026-11-01).** The single write point of film chunks is
`filmcache.Write` (every downloader goes through it); it stores a chunk in zstd level 3 when that shortens it, raw
otherwise. The manifest `size_bytes` keeps meaning "the size of what the writer received" (decoded bytes), so a
manifest written before this change stays valid after a chunk is recompressed, and the truncation check compares the
logical size (the size declared in the zstd frame header). The `source` layer, the only one that touches film bytes
(ADR 0034 D-2), recognises raw, zlib (`0x78`) and zstd (`28 b5 2f fd`) by header and is the only importer of the zstd
library (ratchet `no_raw_film_bytes_outside_source`). Existing chunks are converted by `levelup film recompress`,
launched by the user and never automatically (ratchet `no_auto_film_recompress`): one film at a time, each chunk
compressed, decoded back and compared before an atomic replacement, resumable. The source revision is unchanged (a
raw or zlib chunk reads to the same bytes; its golden fingerprint is re-recorded), and `replay-equiv` is identical on a
film before and after recompression.

### The shell

**D-13 — Tauri 2 (D-1), Windows only (D-2), autonomous (D-3).** The shell runs its own server; "connect to a server" is
a P2 option. No language literal in the Rust (A4): labels come from `GET /api/v1/desktop/strings`; for the states where
the server does not answer (start, stop, death) a fallback catalogue is generated at build time from the same TOML
(`build.rs`), in every language of `config/locales.toml`. macOS is P3.

**D-14 — WebView2 and plugins.** `webviewInstallMode` Evergreen (the bootstrapper): the runtime is not guaranteed on
Windows 10 (A5). Plugins: `single-instance`, `window-state`, `opener`, `updater`. Sidecar started with
`std::process::Command` (stdin pipe, stdout drained, stderr relayed). Lot L16.

**D-15 — Installer, version, identity (plan D-14, A9, A22).** NSIS, per user, no elevation: program in
`%LOCALAPPDATA%\Programs\LevelUp`, data in `%LOCALAPPDATA%\LevelUp` (ADR 0008 amendment); the uninstaller never deletes
the data root. The Tauri identifier `info.lvelup.desktop` is frozen from P1. One version: the git tag feeds
`-ldflags -X main.version`, `Cargo.toml` and `tauri.conf.json`, and a "version check" step of `release.yml` refuses any
difference among the four.

**D-16 — Updater signed by its own key; no code signing (plan D-10, A21, A3).** The installer is **not** code-signed:
SmartScreen warns at each installation, said on the download page. The updater is signed by a Tauri key pair
independent of any certificate; `latest.json` is signed, published on GitHub Releases, and the installer runs only after
a clean `stopped` (an installer cannot replace a running executable). The updater is offered, never imposed. The private key and
its password are repository secrets only as working copies (`TAURI_SIGNING_PRIVATE_KEY`,
`TAURI_SIGNING_PRIVATE_KEY_PASSWORD`); workflows reading `secrets.` pin their actions by SHA
(`ci_workflows_security_test.go`); the public key, which is not a secret, is allowlisted in `.gitleaks.toml`.
*Backup.* Two independent copies of the private key and password under the maintainer's control, outside the repository;
at each release the restore is proved by signing a test artifact from the backup copy and verifying it with the public
key embedded in the application (L17.6). *Rotation.* The configuration carries one public key, so a new key reaches
installed applications only through a release signed by the old key that embeds the new public key; the next release is
signed with the new one. If the old key is lost or compromised, no automatic update can carry the change: users
reinstall from the download page, which is why the backup is checked at every release.

**D-17 — ffmpeg shipped (plan D-9).** The official LGPL build (about 80 MB) is installed next to the server binary,
pinned by checksum, with the LGPL text and the link to the sources in the third-party notices. `media.ResolveFFmpeg`
looks in `LEVELUP_FFMPEG_DIR`, then the directory of the executable, then `PATH`. Cost: 80 MB per update.

**D-18 — Rust hygiene and the language rule (A23).** The `CLAUDE.md` rule "Go/TS only" is amended: Rust is allowed
under `apps/desktop/src-tauri` only, a few hundred lines, no business logic (everything decidable is decided by the
server). `.gitignore` (`target/`, `gen/`), `Cargo.lock` versioned and listed in `NOTICE` §6, Dependabot `cargo`,
`cargo audit` and `cargo clippy -- -D warnings` in CI, Makefile targets, `docs/COMMANDS.md` EN and FR, a `desktop` filter
in `ci.yml` (`ci_paths_filter_test.go`).

### External dependencies

**D-19 — One Azure application (A20).** All installations share the client identifier of the public client
(`platform/auth/azure_credentials.go`). It is an external dependency of every installation: if Microsoft revokes or
throttles it, all of them are affected at once. There is no per-install secret (a public client).

**D-20 — Microsoft content rules (A19).** Read on 2026-11-01 (`xbox.com/en-US/developers/rules`, last updated January
2015): a donation link is allowed on a page that uses the content, the application stays free and ad-free; the notice is
**mandatory** and worded by Microsoft ("Halo Infinite © Microsoft Corporation. LevelUp was created under Microsoft's
"Game Content Usage Rules" using assets from Halo Infinite, and it is not endorsed by or affiliated with Microsoft."),
with a link to the rules, visible to anyone who sees the content: web footer, installer and download page (L17.1);
no game logo in the application's logo; `NOTICE` §2(b), §4 and §6 updated. **Sounds**: the rules warn that audio effects
are often third-party licensed and not covered. The maintainer keeps the 184 sounds on 2026-11-01, the posture of the
website, with a line in the third-party notices ("game sound effects, property of Microsoft and its licensors, removed on
simple request"); the risk is the maintainer's. The licence the rules grant Microsoft over the content must be read
against the AGPL of the repository; this ADR records the question, it does not settle it.

### Sign-in through the default browser (L21)

**D-22 - Authorization code flow in the user's browser, no code to copy (L21.00, 2026-11-02).** Under the capability
`LoginViaBrowser` (true in `desktop` only, published as `can_login_via_browser`), the setup assistant offers "Sign in with
Microsoft": the page calls `GET /api/v1/auth/xbox/login` (with the shell cookie), receives `{attempt_id, authorize_url}` and
opens the URL by a `target="_blank"` link, which the shell sends to the default browser (D-13: the window never shows the
Microsoft page). The flow is the authorization code with PKCE S256 of the shared public client (D-19), **built explicitly**
(`platform/auth` `ExchangeAuthorizationCodePublic` and `BuildAuthorizeURLPublic`: the fixed client identifier of the shared
application, no secret, never read from the environment, so that a repository `.env.local` loaded by a desktop server cannot turn
the `/token` request into a confidential client's), the redirect URI is
`http://localhost:<bound port>/auth/xbox/callback` (the user added `http://localhost/auth/xbox/callback` to the Azure
application; the loopback exemption accepts any port, and the name is `localhost`, not the numeric address).

The attempt lives in the same registry as the device code (`AttemptStore.StartBrowser`), bound to the shell's session, with
a 256-bit `state`, the PKCE verifier and the language of the page. The return reaches the server **without the shell
cookie** (it is in the user's browser), so `DesktopGate.AllowBrowserCallback` lets exactly one request through without it:
`GET /auth/xbox/callback`, exact path (never an ambiguous one, D-3), `Host` equal to `localhost:<port>` or `127.0.0.1:<port>`
of the bound port; any other method, path, port or name keeps the Host and cookie checks (421, 401). The route is mounted
behind `LoopbackOnly` (the TCP peer, not a header) and protected by the `state`: single use (`ClaimBrowserState` removes it
and the verifier at the first presentation), valid 10 minutes (`BrowserStateTTL`), unknown, replayed or expired states render
the failure page without any outbound call. The code is exchanged with the verifier, the chain runs through
`platform/auth` and `libs/haloapi/xblauth` (XBL, XSTS, Spartan; the Xbox Live XSTS is best-effort as for the device code), and
the attempt becomes `authorized`. The assistant's polling (`GET /auth/device-flow/{attempt_id}`, unchanged) hands the attempt to
the session once and runs the same post-flow strategy as the device code: **`AuthCompletion` is the single arrival point**
(tracked profile created by `Onboard`, ADR 0035; tokens sealed by the `TokenCipher`, D-11; `InvalidateCachedPlayerTokens`;
pending tokens replayed at profile creation). The return page is minimal HTML in the language of the page that started the
sign-in (`desktop.toml`, `desktop.browser_login.*`), carries only a fixed error code on failure, and neither the `code`, the
`state` nor the verifier is ever logged. The device code stays the fallback ("Can't open the browser? Use a code instead"); the
hosted site keeps its own `/auth/xbox/login` redirect flow in mode `xbox`, untouched.

### Living with the game (L21b)

**D-23 - The desktop server watches the game, the shell lives in the notification area (L21.a to L21.e, 2026-11-02).**
Decisions of the user, 2026-11-02: the settings of items (b) to (e) are **off by default**, the game is detected by **process
enumeration only** (never a memory read), nothing is imposed. Four capabilities, all true in `desktop` only (D-4 of ADR 0044):
`GameProcessWatch` (`can_watch_game_process`), `ShellIntegration` (`can_integrate_shell`), `ScheduledRefresh`
(`can_schedule_refresh`) and `AutoSyncByDefault` (server side only, see "Game mode"); the settings are `game_process_watch` (default **on**: the detection alone has no visible effect), and
`native_notifications`, `close_to_tray`, `start_with_windows`, `scheduled_refresh` (default off), grouped in
`domain.DesktopSettings`. A PATCH of a key whose capability is absent is a 400 `desktop_setting_not_offered`. The "Application"
tab of the settings page exists under any of the three capabilities.

**Game mode (L21.a).** `internal/platform/gameproc`: a snapshot of the process list (`CreateToolhelp32Snapshot`, file names only,
nothing opened, nothing read), every 15 s, compared (case-insensitive) with the executable names the titles declare (`process_names`
of `title.toml`, `[]` for Halo 5 which has no PC client; the built-in Halo Infinite descriptor, whose `title.toml` is ignored by
ADR 0025, carries `HaloInfinite.exe` in `NewRegistry`; `Registry.GameProcessNames` is the union, no `slug ==`). `Mode` holds the
state (gauge `game_mode_running`, counters `game_mode_*`, `GET /settings/desktop` answers `game_running`) and the **gate**:
`HeavyWorkAllowed()` is consulted **before** each launch, never during one, so no task is ever killed. The cooking loop of the replay
artifacts (`replayartifacts.buildAll`) persists the films (they expire server side) and postpones the cook to the next catch-up
while the game runs; `scheduler.runHeavy` waits (it re-reads the state every 30 s) in front of the four heavy loops (`data_health`,
`asset_name_sweep`, `catalog_refresh`, `world_leaderboard`; the guard `heavy_gate_test.go` forbids a bare cycle call there). The
auto-sync cycle itself is not suspended: its network calls and short inserts go on. What it does that is heavy goes through the same
gate: the post-sync **kill source** step (1.57) decodes films inside the process, so while the game runs it is postponed like the
cooking (`runKillSource` answers 0 and counts `game_mode_heavy_deferred_total`; the backlog is read again from the database at every
cycle, so the first cycle outside a match catches it up). **The first reading is synchronous**: `armGameWatch` builds the watcher and
polls once BEFORE the first heavy task is launched, and until a reading has been made `HeavyWorkAllowed()` answers no
(`Mode.ExpectFirstPoll`, lifted by any poll, whatever its outcome), so a game already open at launch (reduced start, the ordinary case)
never sees an audit or a catalog refresh start in the middle of the match. The transition "game running to game closed" asks the
auto-sync scheduler for a catch-up (`RequestCatchUp`, the path of the wake-up sequence, D-10) with a 60 s debounce, **only if the auto-sync
is on**: `spnkr_auto_sync_enabled` is **on by default in `desktop`** (capability `AutoSyncByDefault`, a decision of the pilot, 2026-11-03:
`settings.Store.WithAutoSyncDefault`, absent key only, a written value, `true` or `false`, always wins and the file is never rewritten),
because the auto-sync loop ignores every tick while it is off. When it is off the closing logs "fin de partie : auto-sync désactivée,
rien à faire" and counts `game_mode_sync_skipped_total`; `game_mode_sync_triggered_total` and the "requested" line only say so when a
synchronization really is requested. The existing presence trigger (live match poller) acts per match while the game is open, this one
acts at the closing, so the two do not duplicate. Turning the setting off drops the mode without a sync; an enumeration that fails keeps
the previous state and is logged once.

**Events and commands (L21.b, L21.c).** Two events join D-2, **one line each, same single writer**:
`{"event":"notification","title","body","kind"}` and
`{"event":"shell_settings","native_notifications","close_to_tray","start_with_windows","locale"}` (written once the server is ready, then
at every change read every 5 s from the settings, whatever wrote them). The shell cannot call the server's HTTP API (its cookie is
`HttpOnly`, no command is exposed to the page), so both directions use the private pipes: server to shell by stdout events, shell to
server by **commands on stdin**, one per line, executed before the EOF, which stays the stop order (D-4): `sync` runs
`AutoSyncScheduler.RunOnceTrigger("manual")`, one cycle at a time, counted by the background wait group (`cmd/server/stdin_commands.go`).
An unknown command is logged and ignored.

**Native notifications (L21.b).** `internal/notifications/native` is a second `ExternalForwarder` (combined with the Discord relay by
`notifications.Fanout`), dormant until the desktop boot installs its sink. It retains the bell categories `match_synced`, `sync_error`,
`challenge_added|completed`, `objective_completed`, `streak_milestone`, `personal_record`, `career_rank`, `skill_tier`, composes the text
in the player's language from the message catalogue (`desktop.notify.*`, one title and one body per kind, complete in every language) and
re-reads the setting at each notification. It is limited to the **local player**: the notification service of every tracked profile is
wired to it (`Relay.ForPlayer(xuid)`), but a profile is local only when the token store carries a refresh token for its xuid (the account the
user signed in with, `OwnerFilter`, read at each notification); a followed friend who changes tier never becomes a Windows notification
(`notifications_native_foreign_total`), and without the filter nothing is relayed. The shell shows it with `tauri-plugin-notification` only when the window is hidden, minimized
or not focused (`native_notice`, pure); the journal carries the kind, never the text. The plugin queues the notification and discards the error of the underlying library, so
`outcome=requested` says the request left, never that Windows displayed it (`failed` remains only for the errors that prepare the
request). Known limit: the plugin has no click callback on the
desktop; a click on a toast re-launches the executable and the single-instance plugin calls `focus_main`, which shows, restores and focuses
the window (to be observed at gate 21.y).

**Notification area (L21.c, L21.d).** `shell_settings.rs` (pure) decides: the icon exists while `close_to_tray` is on, or while the window
hidden by a minimized start has not been shown; the cross hides the window **only if the setting is on and the icon really exists** (a
failed creation makes the cross quit: never a hidden window without its way back); **every creation, replacement or removal of the icon runs
on the main thread** (`shell_tauri::dispatch_tray_change`, `run_on_main_thread`): the hidden window of `tray-icon` belongs to the creating
thread and its destruction fails elsewhere, so a supervisor-thread destruction leaked it and left a ghost icon after an Explorer restart;
the language compared is the RESOLVED catalogue language (`ShellState::tray_locale`: the one the server announces, else the system's), so a
`--minimized` start whose icon already carries the right language is not recreated; the click handler of the menu is global to the
application and registered once (`RegisterOnce`), not at each creation; the menu is Open, Sync now (the `sync` command), Quit
(the same clean stop as the cross without an icon); the labels come from the catalogue compiled from `desktop.toml` (`desktop.tray.*`, also
served by `/desktop/strings`) in the language the server announces. `--minimized` (added by the startup entry) creates the window hidden
and the icon at once; showing the window removes the icon again if `close_to_tray` is off. The window-state plugin no longer saves the
visibility (it would show the hidden window), and under `--minimized` it does not restore the maximized state either (`maximize()` shows the
hidden window): that state is restored once, just before the window is first shown (`launch::restores_maximized_*`). A server that dies
while the window is hidden shows the window before the "stopped" screen (`Action::RevealWindow`), and a "Sync now" request that cannot
leave (server not ready) logs `sync_skipped phase=<phase>`. The startup entry (`tauri-plugin-autostart`, value `LevelUp` of the user's `Run` key, never
`HKLM`) is written only by `autostart::reconcile`, which reads first and writes only to change something: an off setting with no entry
writes nothing. It tells "present" (the `Run` value) from "approved" (`Explorer\StartupApproved\Run`, first byte even = enabled, odd = disabled by
the user in the Windows task manager, read with `winreg`): an on setting never rewrites an entry the user disabled (`left_disabled` in the journal),
an off setting deletes the entry whatever its approval state. A guard (`tests/textes_de_configuration.rs`) forbids any other file from calling `enable` or `disable`.

**Scheduled task (L21.e).** `internal/platform/schedtask`: when `scheduled_refresh` turns on, the server creates the per-user task
`LevelUp\Refresh-<8 hex>` (the suffix is a fingerprint of the data root: a server started on other roots, a test or a development run, never queries,
overwrites or deletes the user's task; every 6 hours, `LeastPrivilege`, hidden, `StartWhenAvailable`, no battery condition, 15 minutes at most) running
`cmd.exe /d /s /c "set LEVELUP_DATA_DIR=... && set LEVELUP_RESOURCES_DIR=... && set LEVELUP_LOGS_DIR=... && levelup-server.exe -mode refresh-only"`,
and deletes it when it turns off. Deviation from the plan text (`schtasks /Create /SC ... /TR ...`): the definition goes through an XML
file (`/XML`, UTF-16 LE), because `/TR` is limited to 261 characters, which three root paths exceed, and because only the XML carries the
resume-after-sleep and battery settings. `schtasks.exe` (System32 path, never the PATH) is launched by `childproc.Command` behind the
`Runner` interface; a path with `"`, `%`, `&`, `^`, `!`, `<`, `>`, `|` or a line break is refused (`unsupported_path`) rather than badly
quoted. The existence test lists the tasks (`/Query /FO CSV /NH`) instead of `/Query /TN`, whose exit code does not tell "absent" from a refusal and
whose message is localized. At start-up the loop **reconciles** (setting on: the task is always rewritten with `/F`, one call, so a reinstall in
another folder or a changed root never leaves a stale task shown as "present"; setting off and a task left over: delete it; otherwise nothing); afterwards each **change** creates or deletes. A failure is logged, exposed as a fixed code (`schtasks_failed`,
`unsupported_path`) by `GET /settings/desktop` (`scheduled_refresh.state` = `present|absent|error|unknown`) and not retried at every tick.
`-mode refresh-only` takes the instance lock before anything and exits 0 without an exchange when the application runs (D-10, tests
`refresh_only_test.go` and `e2e_refresh_only_test.go`). Limits: the task runs a console program, so Windows may flash a console window for
a moment (to be seen at gate 21.y); the uninstaller does not remove the task nor the startup entry (a NSIS hook could not be tested here:
turn the settings off before uninstalling, or delete them by hand).

### Data and diagnostics (L22)

**D-24 - The desktop backup is a set of Parquet files in a folder the user chooses; restic stays on the hosted server (L22.a, 2026-11-03).**
Under the capability `ParquetBackup` (`can_backup_to_folder`, desktop only; it replaces the restic cycle that `UseResticBackup` already
cuts in this profile) `internal/ops/parquetbackup` writes `<backup_dir>/levelup-<YYYYMMDD-HHMM>/`: one zstd Parquet per table of every
database `ops.BackupDatabases` lists (the four warehouse databases of each title, one database per player, the global aliases when the
file exists; keys and paths only through `PathResolver`), the schema of each database taken at the same instant
(`_ddl_<stamp>.json`, the format `duckdbbackup.ImportFromStaging` already restores "in the state of its epoch"), the configuration and
authentication files as they are (`db_profiles.json`, `app_settings.json`, `users.json`, `groups.json`, the token files, sealed by the
system vault when the profile has one) and a `manifest.json` (application version, tables, rows, duration, SHA-256 of every file, and
`excluded: ["film_archive"]`: the archived films are not copied, they weigh tens of GB and have their own archive). A set is built in a
`.partial-*` folder renamed at the end: a `levelup-*` folder always carries its manifest. Rotation keeps `backup_keep` sets and only
deletes direct children of the chosen folder whose name is a set name (never anything else), **and always keeps the latest COMPLETE (non-partial)
set**: a run of partial sets never evicts the last one that restores entirely. **Read-only by construction**: a database is read through
`duckdb.WithHeldRead`, which takes the database write lease with the same bounded wait as the periodic CHECKPOINT (a database whose lease
is held is skipped, then retried once with a longer wait at the end of the pass; a database still busy makes the set `partial`, said by
the manifest and by Settings). **The lease is taken and released PER TABLE** (`duckdbbackup.ExportTargetTablesHeld`), not held for a whole
database: the writers with a short wait (sync metadata 5 s, the persist sink) never wait longer than one table, and a handle opened only for the
export is closed between two tables. Each table has its own deadline (2 minutes): a table over it is left out and the set is `partial`; a lease
that cannot be taken in the middle of a database drops that database for the pass (retried whole with the longer wait), lends the process-wide cache handle (read-write included, `OpenReadForQuery`) and never opens read-only
by force; the package contains no write statement (guard `readonly_guard_test.go`). Settings: `backup_dir` (empty = off),
`backup_interval` (whole hours, `1h` to `168h`, empty = `24h`), `backup_keep` (1 to 60, default 7). The folder is validated by the
server (absolute, existing, writable, **not under the data root**; the same volume is accepted and flagged with a warning). The
`backup_interval` key is shared with the hosted restic planner (a Go duration such as `6h`): the desktop validation applies to the
keys a PATCH writes, the read never rewrites the value. The periodic loop (`cmd/server/boot_parquetbackup.go`, declared by capability like the
L21 loops, not in `taskPolicies`) wakes every minute, asks the heavy-work gate (`gameproc.Mode.WaitHeavyAllowed`: no backup during a game) and
writes a set when one is due; "Back up now" is `POST /settings/data/backup/run` (202), the state is `GET /settings/data`
(`backup.last` is read from the folder, so it survives a restart). Counters: expvar `levelup.backup.parquet` (`ok`, `partial`,
`failed`), `levelup.backup.parquet_last_duration_ms`, `levelup.backup.parquet_last_bytes`. **Restore**: `levelup-server.exe -mode restore --from <set> --to
<data root> [--dry-run] [--force]` (the server the installer ships: it does not ship `levelup.exe`; `levelup backup restore` is an alias that calls
the same function, `parquetbackup.RunRestoreCommand`, flags included: no copy). The command takes the instance lock of the target root and refuses
while it is held; **`--force` never bypasses the lock, only the existence of databases** (every server profile holds that lock since L22, D-10 amended).
It verifies every checksum first and **refuses a set whose `db/` folder holds a file the manifest does not list** (the importer takes the newest
Parquet of each table: an added file would be imported with no checksum), recreates each database from its Parquet through the existing importer,
puts the configuration back, and refuses a target that already holds one of the databases without `--force`. A leftover `<base>.wal` in the target root
is refused without `--force` (killed application: start it once for a clean stop, or delete the WAL if losing the last writes is acceptable); under
`--force` the WAL is set aside WITH its base and put back with it on failure, never replayed by the new database. The migrations of the next start
bring the databases up to date.
Tokens of another machine do not decrypt (system vault): the player signs in again. No native folder picker: the web page cannot open one
and the shell has no command channel from the page (the cookie is `HttpOnly`, D-23), so the folder is a text field validated by the
server; a `pick_dir` command and a `dir_picked` event would have been a new contract between shell and server for a field typed once.

**D-25 - Moving from the site to the application: an export by one-shot token, an import through `persist` (L22.b, 2026-11-03).**
Hosted side, capability `ExportForDesktop` (`can_export_for_desktop`: every profile but desktop and demo): `POST /export/desktop/token`
(signed-in player only) returns a random 256-bit token valid 15 minutes and usable **once** (only its SHA-256 is kept, in memory;
asking for a new one invalidates the previous), `GET /export/desktop` serves the archive to the owner of the token (header
`Authorization: Bearer`, from the application, which has no session) or of the session (which must send an `X-Requested-With` header: a cross-site
GET link cannot, so it never triggers an export unseen). **The player is always the one the token or the session designates**
(ownership, ADR 0029): no request names a player, and the player database is found by the player's **xuid** (the tracked profile that carries it, ADR 0035),
never by the gamertag of a session. One SUCCESSFUL export per hour and per player (429 afterwards, logged): the quota is taken after the export
succeeds (a failed export does not spend it), and only one export per player runs at a time. The archive
(`internal/transfer`, a zip of uncompressed entries) is a **closed list**, and **a match is exported WHOLE**: for every match where the player's xuid is among
the participants, ALL the rows of ALL the match tables (`match_registry`, `match_participants` of every participant, `medals_earned`, `weapon_accuracy`,
`highlight_events`, `killer_victim_pairs`, the kill journal `match_kill_events` with its summaries `match_kill_heads` / `match_kill_pairs` /
`match_assist_pairs`, `kill_positions`, `match_csrs`, `match_commendations`, `match_objective_stats`), plus `player_match_enrichment` of his player database.
A match imported by halves would stay incomplete for ever (the sync rule `knownset` counts it as known, `SharedPersister` skips a match whose
registry row exists, `events_loaded` would say the events are there): hence the whole match. "Never the other players" means never their OWN data (their
enrichments, their player database, a profile tracked on the same instance), not the other participants' rows of the player's matches. No token, no
account, no session, no reference table, no film-derived table (lives, usage, shots...: they are recomputed from the film the application loads itself);
`weapon_kills` is not in the list (Halo Infinite's schema no longer carries it; there is no Halo 5 export). Desktop side
(`ImportFromSite`, `can_import_from_site`): Settings > Application and an optional block of the first-run assistant take the site address
and the token (`POST /import/site`, 202, state by `GET /import/site`). The download refuses any scheme but https (http only to the
loopback), follows no redirect (the token only goes to the address typed), bounds the size and the time; the archive is verified before any write
(format, entry names matched against `^(shared|player)/[a-z0-9_]+\.parquet$` and the closed list, so no path traversal, total size, SHA-256 of every
table, and **the archive's xuid must be the tracked player's**). Only the matches the tracked xuid takes part in (read from the archive's own
`match_participants`) enter, and a player-table row only for his xuid; an archive of match tables without participants is refused. The write is `persist.ImportPersister`: INSERT only, no UPSERT, UPDATE or DELETE,
one transaction per database, a row whose key already exists locally is never rewritten (replaying an import inserts nothing), the common
columns of the file and of the local table only (an `id` the local table fills itself is not imported: the local sequence is not the origin's, and the `_latest`
views order passes by `(written_at, id)`; the file's order is kept), a missing mandatory column refuses the table. A table with no natural key is keyed
by `match_id`: a match is completed whole or not at all. After the import the application drops its player read caches (`InvalidatePlayerReadCaches`, as at the
end of a sync) and the page reloads the filter resolution of every tracked player. A failed connection gives a distinct code (`site_unreachable`,
`site_tls`, `site_not_found`), not a single `import_failed`. It goes through the application's existing write paths
(`sync.AcquireSharedWriterStandalone` with the `SharedProvider`, then the player lease and `OpenPlayerDB`). Limit: the default title only (the shared database
provider exists for it alone, as for the OpenSpartan import); the manifest carries the title so another one is a data change. The
OpenSpartan import recipe was not reused as is: it is typed (one Go struct per table) and reads a SQLite file; a Parquet table of another deployment, whose
schema may have drifted, needed the column intersection and the anti-join, in the same package and under the same rules.

**D-26 - The diagnostic report is a zip with no secret (L22.c, 2026-11-03).** Under `DiagnosticReport` (`can_report_diagnostic`),
`POST /api/v1/desktop/diagnostic` writes `<DataRoot>/diagnostics/levelup-diag-<YYYYMMDD-HHMM>.zip` (`internal/ops/diagnostic`): the end (5 MB) of every `*.log` of the logs root
(`desktop.log`, the shell journal, included: it carries the updater's state), `app_info.json` (version, profile, roots, capabilities, process and DuckDB
memory), `settings.json`, `scheduled_tasks.json`, `updater.json`, and for each database a schema diagnostic read through the server's read door
(`ops.DiagnoseHandle`; `DiagnoseDB` opens read-only by force and fails on a database the process holds). **Two barriers apply to everything that enters the
archive**: structured documents lose the value of any key whose WHOLE NAME is a secret name (`refresh_token`, `oauth_refresh_token`, `access_token`, `xsts_token`,
`spartan_token`, `authorization`, `cookie`, `webhook_url`, `webhook`, a bare `code`, anything containing `secret` or `password`, case and separators
normalised: `OAuthRefreshToken` is `oauth_refresh_token`), never a word inside a longer name (`players_with_token`, `scheduled_refresh`,
`spnkr_refresh_*`, `TokenVault`, `xsts_refreshed`, `discord_webhook_url_present` stay: they are counters, settings and capabilities the diagnosis needs), and log text goes
through `ScrubLog`: a line that is one JSON document (the per-module logs) is read as JSON, only the values of forbidden keys and the text of strings are
replaced, the key order is kept and the line stays valid JSON; any other line goes through `Scrub` (authorization headers, JWT and Microsoft refresh-token shapes, `key=value`
pairs whose KEY is forbidden by the same rule, the value bounded at its quote, sensitive URL parameters, the exact values of the process secrets). The sentinel test sows a fake token, launcher secret, webhook and
authorization code in logs, settings and attached documents and searches every entry; a second test sows a value with NO known token shape behind a
sensitive key (per-module JSON log, shell stderr relay, settings, a profiles document), so only the by-key filter can remove it, and unit tests exercise
each `Scrub` filter alone (mutation checked: inverting any filter turns a test red). The reply carries the path (button "Copy the path" in the About window).

**D-27 - The memory ceiling acts on DuckDB, not on the process (L22.d, 2026-11-03).** Setting `memory_limit_mb` (0 or 1024 to 32768, capability
`MemoryLimit`): DuckDB is what consumes the memory, so the lever is its `memory_limit`, applied at every connection open (`applyDuckSessionInit`),
at start-up before the first database opens, and live to the already open databases (`SET memory_limit`) when the setting changes. 0 means "the
server default" (`LEVELUP_DUCKDB_MEMORY_LIMIT`, else 512 MB), not "unlimited": the shipped default is already a ceiling. The value is in MiB (4096 = 4.0 GiB).
**The ceiling is PER OPEN DATABASE, not for the process**: every DuckDB file is a database instance of its own with its own memory, so N databases open at
once may use up to N times the ceiling. Settings ("Memory limit per open database"), `CONFIGURATION.md` and the field help say so.
No kill and no OS limit. The resident memory of the process (`GetProcessMemoryInfo` working set) is read at each `/debug/vars` read as
`levelup.process.rss_mb` and shown in Settings ("current memory: N MB").

### Tests

**D-21 — Strategy (plan section 7).** Each side of the contract is tested against a double of the other: a **fake
sidecar** in Go (`apps/desktop/testdata/fake-sidecar`, its own `go.mod`, `GOWORK=off`: events, `/desktop/enter`, EOF,
variants "never ready", `failed`, late `stopped`, crash, chatty 10 MB stdout) for `cargo test`, and a **fake launcher**
(stdin pipe, stdout read line by line) in the tests of `cmd/server` for the sequence `listening`, `ready`, EOF,
`stopping`, `stopped{clean:true}` and the second launch `failed locked`. The Windows CGO lane (`go-test-windows-cgo`)
runs the server tests and `TestE2EArretPropre` (tag `e2e_windows`); the `desktop-e2e` job on `release/**` runs a
PowerShell script (silent install, launch, port read from the file, `/health`, `WM_CLOSE`, clean stop, relaunch);
`tauri-driver` is not used (it needs an `msedgedriver` aligned with the runtime). Visual gates are witnesses named by the
maintainer on a clean machine. A lot without a test at its level is not closed.

## Known risks

- **Stop in the first seconds gave `clean=false`** (treated by L12.6). If the stop arrived while a token exchange of the
  start-up was running, the scheduler phase (3 s) expired: the world leaderboard cron resolved every tracked account
  with `context.Background()` (about 1 s per exchange, loop over the players), and the refresh loop of the watcher ran
  under the process context, outside the wait group. The exchanges now carry the cancellable context of the background
  tasks and the loop is tracked by the wait group; token writes never depend on the context, so a token obtained before
  the cancellation is written before the databases close (tests in `internal/platform/auth/refresh_loop_stop_test.go`
  and `internal/worldenrich/wiring_ctx_test.go`). Residual: a stop requested DURING the start-up waits for the end of the
  initialisation (boot-time token refresh included, 15 s per exchange at most), as the gate documents.
- **Crash capture lost at the first start on a fresh root.** `crash_output` opens `server.crash.log` before `logs/`
  exists: a warning, and no capture until the second start. This is the normal first start of every installation. Treated
  by L11 (`LogsRoot` created before any writer) or a dedicated lot.
- **SmartScreen at each installation** (D-16), **sound licensing** (D-20), **a single Azure application** (D-19).

## Implementation notes (L17, 2026-11-02)

- **Install folder (D-15, plan D-14).** Tauri 2 installs a `currentUser` NSIS package under
  `%LOCALAPPDATA%<product name>`, which is the default DATA root: program and data would share one folder. The hook
  `src-tauri/windows/hooks.nsh` (`NSIS_HOOK_PREINSTALL`) moves only that exact default to
  `%LOCALAPPDATA%ProgramsLevelUp`; a folder chosen by the user (`/D=`, directory page) or the one of a previous
  installation (registry, restored by the template) is never changed. The uninstaller generated by Tauri deletes the
  files it installed one by one and removes a folder only if it is empty (no recursive `RMDir`), and erases
  `%LOCALAPPDATA%info.lvelup.desktop` (the WebView2 profile) only when its "delete the application data" box is
  ticked, which silent mode never does: the data root survives. `apps/desktop/scripts/e2e-install.ps1` proves it.
- **Resources.** On Windows the resource folder of Tauri IS the install folder, so it becomes `ResourcesRoot`: `web/`
  (the compiled front; `config.DesktopWebDistDirName`, the default of `WebDistDir` under the desktop profile when
  `LEVELUP_WEB_DIST` is absent), `config/`, `static/`, the versioned reference `data/titles/*/reference` and
  `metadata-prebuilt.zip` (listed by `git ls-files`, never from the disk), ffmpeg and the notices.
  `data/cache/career-rank-image` is NOT shipped (17 MB per update for images the server downloads on demand). They are
  assembled by `scripts/stage-bundle.ps1` and named by `tauri.bundle.conf.json` (merged by `--config`, so a plain
  `cargo build` never needs them).
- **ffmpeg (D-17).** Pinned in `scripts/fetch-ffmpeg.ps1` by URL and SHA-256: BtbN/FFmpeg-Builds release
  `autobuild-2026-10-31-12-59`, `ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip`
  (`5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab`, the sum published in the release's
  `checksums.sha256`). LGPL v3, shared, without libx264. `ffplay.exe` is left out.
- **Release workflow.** `release.yml` also runs on `release/**` pushes (installer unsigned, end-to-end test, nothing
  published). `desktop-build` checks the version (`scripts/check-version.sh`: tag, `Cargo.toml`, `tauri.conf.json`
  and the `-ldflags` read back from the server binary), refuses to start without the signing secret on a tag, builds with
  `createUpdaterArtifacts` switched on by `--config` only for a tag, and `publish` builds `latest.json` (version, the FR
  changelog section as `notes`, `pub_date`, `platforms.windows-x86_64` with the signature and the installer URL).
- **Updater flow (D-16, plan A3).** `src-tauri/src/updater.rs` (no tauri import) holds the decisions: a pure state machine
  (`UpdateFlow`: idle, checking, offered, downloading, armed, done), the install gate (`InstallGate`) and the host wrapper
  (`UpdateHost`). The check starts once per launch, on its own thread, when the server has announced `ready` (the entry
  page is loaded); an error is logged and nothing else happens. The offer is a native dialog (two buttons named by the
  catalogue, closing it means "Later"; texts `update_title`, `update_install`, `update_later`, `update_notes_heading` and
  `update_available` of `desktop.toml`, compiled into the fallback catalogue, no language literal in the Rust); "Later" ends
  the proposal for this launch, nothing is downloaded. After "Install" the package is downloaded while the server keeps
  running, then the gate is armed and the ordinary clean close starts (stdin closed, `stopped`, process exit); the
  installer is launched by `exit_app` only after a clean stop: `stopped` with `clean=true` and the process gone without
  being killed (`install_allowed`, a pure function of the supervisor outcome). Otherwise (killed, kill failed,
  `clean=false`, no `stopped`) nothing is installed: `desktop.log` gets `update_skipped_unclean_stop` with the reason, a
  native box (`update_skipped_unclean`) says the update will be offered at the next launch, and the application exits. A
  refused install logs `update_install_failed` and shows `update_install_failed` (on Windows the installer quits the
  application). The notes shown are the `notes` field of `latest.json` (the French changelog section of
  the release). Debug builds never register the updater. `tests/updater.rs` plays the sequence against the fake sidecar.
- **Updater key: procedure (D-16, L17.6).** Generation: `npx tauri signer generate -w <file>` in `apps/desktop`; the
  public half goes to `plugins.updater.pubkey` of `tauri.conf.json` and to `.gitleaks.toml` in the same commit; the
  private half to the repository secret `TAURI_SIGNING_PRIVATE_KEY` (and `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` if it has
  a password; a GitHub secret cannot be empty). Backup: two copies of the private key file and of its password, on two
  media under the maintainer's control (for example a password manager and the offline backup drive), never in the
  repository, never in a build log. Restore proof, at every release: `npx tauri signer sign -f <copy of the key> [-p
  <password>] <any file>` then verify the `.sig` with the public key of the application (the updater itself rejects a
  bad signature: install a test build signed from the copy over a build carrying the public key). Rotation: release N
  signed with the old key embeds the new public key; release N+1 is signed with the new one; a lost or compromised key
  cannot be rotated this way (users reinstall from the download page).
  Release check: the Tauri CLI only warns when the secret key does not match `plugins.updater.pubkey`, so the release
  workflow runs `go run ./cmd/updater-verify` (minisign verification of the `.sig` against the embedded public key,
  key id and ed25519 signature) right after building the installer; a mismatch fails the release with an explicit error.


## Implementation notes (L18, 2026-11-02)

- **Desktop chrome.** The web client adopts the application chrome (sidebar instead of the top bar, no footer, a modal
  "About" window, home page without the cover image) on the capability `can_use_compact_chrome` of `/bootstrap`
  (`DeploymentCapabilities.CompactChrome`, true for the `desktop` profile only; ADR 0044 D-4: never on the profile name).
  `/bootstrap` also carries `app_info` (`version`, `data_root`) when, and only when, that capability is true: a hosted
  instance never publishes the path of its data.
- **Update check requested by the user.** The page runs in WebView2 without any shell command, so the only channel from
  page to shell is the embedded protocol. The "About" window calls `GET http://levelup-shell.localhost/check-update`
  (`fetch`, `no-cors`, response unread) and the page stays where it is. The protocol answers 204 and the shell runs the
  same sequence as the launch check (`updater::run_flow`: offered, never imposed, installed after the clean stop) on its own
  thread, because the native boxes block. Differences with the launch check: "up to date" and "failed" are said in a native
  box (`update_none`, `update_check_failed` of `desktop.toml`), a build without updater says why
  (`update_unavailable_debug`), and one check runs at a time (`UpdateBusy`, shared with the launch check; a request made
  meanwhile gets `update_busy`). Only the exact path `/check-update` triggers it (`render::route`); `navigation.rs` already
  keeps any other host out of the window. A navigation plus a 302 back to the page was not retained: it reloads the page (the
  open window and the page state are lost) and needs a return URL validated against the local origin; the no-cors call
  needs neither.

## Implementation notes (L19, 2026-11-02)

- **New-window requests.** The window never opens a second window. `WebviewWindowBuilder::on_new_window` always answers
  `Deny` and applies the navigation policy through `navigation::decide_new_window`: a link that `decide` would send to the
  default browser (`OpenExternally`) goes there, everything else (embedded screens, the local server, `blob:`, `javascript:`,
  `data:`, `file:`) is refused. The log carries `event=new_window url=<origin only> decision=open|deny`: an external URL can
  carry a code (the device-code sign-in link), so the path and the query are never written. `LEVELUP_SHELL_OPEN_EXTERNAL=log`
  replaces the browser launch by the log line, for automated runs.
- **Why `target="_blank"` used to do nothing.** `tauri-plugin-opener` injects by default a script that catches clicks on
  `target="_blank"` links, calls `preventDefault()` and invokes the `open_url` command over IPC. The server page has no
  capability, so the invocation is refused and the click was swallowed before the engine asked for a new window (only
  `window.open` reached `on_new_window`). The plugin is built with `open_js_links_on_click(false)`; the Rust side keeps
  using `AppHandle::opener()` for the launch. Ctrl+click and Shift+click, which the script also routed, now reach the engine
  as new-window requests and follow the same policy.
- **`window.open` and `noopener`.** A refused new window makes `window.open` return `null`, as `noopener` already does in a
  browser: the web client must not read that value as "blocked". The feedback form opens GitHub through a clicked
  `target="_blank"` link (`openInNewTab`).
- **Installer.** Tauri's NSIS template defines `MUI_FINISHPAGE_NOAUTOCLOSE` (the extraction page stays until a click, to
  read the details after an incident) and only sets `SetAutoClose true` for the passive mode. `NSIS_HOOK_POSTINSTALL`
  (`windows/hooks.nsh`) sets it for the interactive installer: the setting is evaluated when the install section ends, so a
  failure (`Abort`) still keeps the page and its log.

## Consequences

- The server stays one binary for hosted and desktop; what differs is the profile (ADR 0044), the roots (ADR 0008) and
  the launch arguments.
- Every stop path ends in one sequence, so the contract is the same for air, the shell, the end-to-end tests and a
  terminal Ctrl+C.
- The shell is small enough to be replaced; the contract (D-1 to D-6) is what a replacement must satisfy.
- Cost: a Windows CI lane with CGO, a second toolchain (Rust 1.97, Node 24), 80 MB more per update (ffmpeg), and a key to
  keep safe.

## Guard-rails

- Exist: `cmd/server/flags_test.go`, `shutdown_stdin_test.go` (one pass for any number of triggers),
  `shutdown_budget_test.go`, `boot_failure_test.go`, `internal/api/bare_routes_ratchet_test.go`
  (`TestBareRoutesRatchet_DesktopShutdownRouteIsGuarded`), `internal/archlint/media_binpath_single_resolver_test.go`,
  `internal/platform/auth/sentinel_test.go` and `internal/sync/no_legacy_source_used_test.go` (allowlists empty),
  `cmd/server/e2e_arret_windows_test.go` (`TestE2EArretPropre`, tag `e2e_windows`), the CI job `go-test-windows-cgo`.
- Created by L12: `cmd/server/desktop_launcher_test.go` (`TestE2ELanceurDesktop`, tag `e2e_windows`: the fake launcher),
  `internal/platform/instancelock/instancelock_test.go` (two processes), `cmd/server/desktop_events_test.go`,
  `cmd/server/desktop_port_test.go`, `cmd/server/stdout_guard_test.go`.
- Created by L13: `internal/api/desktop_entry_cgo_test.go` (every refusal, the success, `/health` open, the strings in
  both languages, CORS and CSRF on the exact origin, nothing mounted outside the profile),
  `internal/api/middleware/desktop_gate_test.go` (expiry, signature, renewal, closed entry),
  `TestBareRoutesRatchet_DesktopEntryRoutes`, `internal/config/config_desktop_session_test.go`,
  `internal/service/auth_completion_poolgrant_test.go`, `TestParseServerFlags_AucunDrapeauNePorteLeSecret`
  (`cmd/server/flags_test.go`: the secret never travels as an argument), `apps/web/e2e/desktop-enter.spec.ts`
  (Playwright, mocked server), and `TestE2ELanceurDesktop` extended (`GET /` 401, entry 302 with the cookie, bootstrap 200).
- Created by L13b (and extended by its review: watcher state file, process-wide log dedup): `internal/platform/auth/token_file_codec_test.go` (plain round trip, sealing of the sensitive fields only,
  migration at the first write, typed refusals, partial write on a sealed file, every string field classified),
  `internal/platform/auth/tokencipher/*_test.go` (`dpapi_vault_test.go` under the Windows tag), `token_file_vault_test.go`
  (real DPAPI store, Windows tag), `internal/config/config_token_vault_test.go`, `cmd/server/boot_token_vault_test.go`,
  `archlint/token_cipher_single_decision_test.go`.
- Created by L16: `apps/desktop/src-tauri/tests/contract.rs` (`cargo test` against the fake sidecar
  `apps/desktop/testdata/fake-sidecar`: nominal entry and clean close, never ready, `failed` with and without a port,
  crash before and after `ready`, late `stopped` killed after the budget, close during start-up, 10 MB of stdout,
  stable port, secret absent from arguments and journal), the state machine tests (`src/lifecycle/tests.rs`), the
  `desktop` filter and the `desktop-test` job of `ci.yml` (`ci_paths_filter_test.go` unchanged and green).
- Created by L17: the `desktop-build` and `desktop-e2e` jobs and the version check of `release.yml`, `apps/desktop/scripts/`
  (`check-version.sh`, `e2e-install.ps1`, `fetch-ffmpeg.ps1`, `stage-bundle.ps1`, `build-installer.ps1`), the NSIS hook, the
  default web dist of the desktop profile (`TestLoad_DesktopWebDistParDefaut`), `ci_workflows_security_test.go` (unchanged,
  green on the new workflow).
- Created by L21a: `internal/api/handlers/auth_browser_login_test.go` (the whole flow with a simulated Azure that checks PKCE
  S256 and the redirect URI: right, wrong, replayed and expired `state`, Microsoft refusal, exchange failure, the arrival
  point called once, the profile tokens written), `internal/api/middleware/desktop_gate_test.go` (the callback opens without a
  cookie for GET on the exact path and the loopback hosts only, and stays closed without the capability),
  `internal/api/desktop_browser_login_cgo_test.go` and `TestBareRoutesRatchet_DesktopEntryRoutes` (the two routes exist under
  the capability only), `internal/platform/auth/attempt_browser_test.go`, `apps/web/src/features/setup/BrowserLoginPanel.test.tsx`,
  `apps/web/e2e/desktop-sidebar.spec.ts` (Playwright on `vite preview`, mocked server).
- Created by L21b: `internal/platform/gameproc/gameproc_test.go` and `lister_windows_test.go` (the real enumeration sees the test process),
  `internal/scheduler/heavy_gate_test.go` (the four heavy loops go through `runHeavy`), `replayartifacts/cuisson_mode_jeu_test.go`,
  `cmd/server/boot_gameproc_test.go`, `boot_shell_test.go`, `stdin_commands_test.go`, `boot_schedtask_test.go`,
  `internal/notifications/native/relay_test.go`, `internal/platform/schedtask/schedtask_test.go` (exact `schtasks` arguments and task XML,
  idempotence, refused paths) and `schedtask_realschtasks_test.go` (tag `e2e_windows`: the real `schtasks.exe` accepts the definition, on a task
  named `LevelUp-test-<random>` that never fires and is deleted at the end), `internal/api/handlers/settings_desktop_test.go`,
  `apps/desktop/src-tauri/src/{shell_settings,native_notice,autostart,launch}.rs` tests, `tests/contract.rs` (notification, shell settings,
  `sync` on stdin, nothing before `ready`), `tests/textes_de_configuration.rs` (only `reconcile` writes the startup entry), the fake sidecar
  (`FAKE_EVENTS_AFTER_READY`, commands recorded), `apps/web/src/features/settings/DesktopIntegrationCard.test.tsx`.
- To create: `cargo audit` in CI.

## Implementation notes (L20, 2026-11-02)

- **Updater notes per language (D-16, L20.c).** `latest.json` carries `notes` (the changelog section of the default
  language, the only key an older client reads) and `notes_en` (the same version in `docs/CHANGELOG.md`). The release
  workflow fails, before compiling and again in `publish`, when either section is missing or empty, and the final `jq`
  check rejects an empty `notes` or `notes_en`. The shell reads the whole document (`Update::raw_json` of
  `tauri-plugin-updater` 2.12) and picks `notes` for the default language, otherwise `notes_<language>`, then
  `notes_<fallback language>` (codes from `config/locales.toml` through `build.rs`, none written in the Rust code), then
  `notes` (`update_notes.rs`, a pure function).
- **Installer download deadline (L20.d).** The download runs on the Tauri runtime while the update thread reads a
  progress counter: no byte for 60 s, or 60 minutes in total, aborts it (`download_stalled`, `download_too_long`), the
  verification lock is released and a requested check says `update_check_failed`. An inactivity limit, not a short total
  deadline, so a slow but live download of the 200 MB installer is never cut (`download_watch.rs`, a pure verdict).
- **Video encoder of the shipped ffmpeg (L20.a).** The LGPL build has no `libx264`: the HLS transcode picks the first
  usable of `libx264`, `h264_mf` (Media Foundation), `libopenh264`, each tried on one frame, the choice logged once per
  ffmpeg path; none usable means `ErrNoH264Encoder` and a refused transcode, never a panic.
- **System time zone and language at first launch (L20.b).** Capability `AdoptSystemLocale` (true under the desktop
  profile only): the Windows time zone key becomes an IANA name through the CLDR table of `platform/syslocale`, the UI
  language comes from `GetUserPreferredUILanguages`, and both are the default of the `user_timezone` and `lang` keys
  ABSENT from `app_settings.json` (`settings.Store.WithSystemDefaults`): a written value always wins and the file is not
  rewritten. The server embeds `time/tzdata`: without it `time.LoadLocation` fails on a machine with no Go installation,
  which is what an installed application is.
- **Periodic CHECKPOINT (L20.g).** Capability `FrequentCheckpoint` (desktop only) and key
  `desktop_checkpoint_interval` (default 5 minutes, bounds 30 s to 1 h): `duckdb.CheckpointHeld` runs `CHECKPOINT` on every
  database this process holds in read-write (the process-wide handle cache, never a second process, never an opening of
  its own) under the database write lease (`dblease`), so a persist or sync write burst, or a B-swap of `shared_matches`,
  makes it skip that database (counted `skipped_busy`). A failing CHECKPOINT is logged (WARN) and counted
  (`levelup.checkpoint.periodic`, `failed`), never swallowed. `shared_social` keeps its own 5-minute task and is skipped
  here. Guard: `archlint/checkpoint_single_site_test.go` forbids a CHECKPOINT statement outside `internal/platform/duckdb`
  except the existing sites of its dated allowlist.

## Implementation notes (L21a, 2026-11-02)

- **Browser sign-in (D-22, L21.00).** Detailed in D-22. The page that serves it is the setup assistant's sign-in step
  (`StepDeviceCode`, which switches to `BrowserLoginPanel` under the capability), not `XboxLoginPage`, which belongs to the
  hosted SSO.
- **Collapsible sidebar (L21.0).** `SidebarNav` is either the 240 px sidebar or the 56 px icon rail (`SidebarRail`). The
  preference has three states, `auto` (collapsed under `(max-width: 1099px)`, expanded above), `collapsed` and `expanded`
  (a manual choice, button at the foot of the sidebar or `Ctrl+B`, kept in `localStorage`); a manual choice is never
  overwritten by the window width and nothing collapses on scroll. The rail shows a tooltip on hover and focus, and a flyout
  `role="menu"` of the tabs of the sections that have some (hover or click, arrows, `Esc`); the game switch and the player
  shrink to initials. Icons are copied from Lucide (ISC, `components/icons/lucide.tsx`, notice in
  `apps/desktop/THIRD_PARTY_NOTICES.md`); the hosted site is unchanged (the sidebar exists under `can_use_compact_chrome` only).
- **Community without comparison (L21.0b).** Capability `CompareAccounts` (`can_compare_accounts`, false in `desktop`, D-7:
  one tracked account). The tab "Rankings" requires it (`L1Tab.deploymentCapability`, read by `useVisibleNavSections`), the
  section leads to "Relations" and `/community` redirects there; the Prestige leaderboard page, reachable by address only,
  says "This page compares several accounts; the desktop application follows only one." instead of an empty ranking.
- **No H.264 encoder (L21.0d).** The start-up probe of the media tooling publishes its definitive verdict (detection done, no
  usable candidate; a doubt is not a verdict) as `media_h264_missing` in `/bootstrap` and `h264_encoder_missing` in
  `/admin/health`; the Media page explains the Windows N and KN editions without Media Foundation and links Microsoft's
  Media Feature Pack. Known limit: a verdict reached later than the start-up probe (a transcode that finds no encoder) is not
  published until the next start.

## Implementation notes (L22, 2026-11-03)

- **Guard-rails created**: `internal/ops/parquetbackup/readonly_guard_test.go` (no write statement, no hand-built `data` path),
  `internal/ops/diagnostic/diagnostic_cgo_test.go` (no secret in the archive), `internal/transfer/transfer_cgo_test.go` (export, import,
  idempotence, other player's archive, tampered archive, path traversal, size bound, one-shot token over HTTP),
  `internal/platform/duckdb/memory_limit_test.go` (`current_setting('memory_limit')` read on a temporary database).
- **Found and left**: the `backup_interval` key is shared with the hosted restic planner (see D-24); `app_settings.json` of a hosted
  instance gains the key `backup_keep` and the three others on its next save (zero or default values, no effect without the capability).
  `ops.DiagnoseDB` still opens read-only by force (used by the CLI, server stopped).
- **No native folder picker, no download resume, no Halo 5 export**: see D-24 and D-25.

## Alternatives rejected

- **Wails** (D-1): no signed updater and a smaller plugin set than Tauri 2 for the NSIS and updater chain needed.
- **A Job object** (A2): see D-4.
- **Port `:0`** (A15, D-13): empties the browser storage at each launch.
- **Code signing** (D-10): no budget; reversible if the distribution justifies it.
- **A stop signal** (D-4): none usable on Windows without a shared console.
- **A standalone binary embedding the site** (C10, D-11 of the plan): not retained, resources travel as Tauri resources;
  stays in the backlog.
- **`tauri-driver` for the end-to-end test** (D-21): depends on a driver aligned with the installed WebView2.

## References

- Plan: `.ai/V8/PLAN_DESKTOP_2026-11-01.md` (sections 4, 5, 6, 7, 9.10 to 9.17, 10).
- Code: `apps/go-api/cmd/server/` (`flags.go`, `boot_gate.go`, `boot_failure.go`, `shutdown.go`, `shutdown_stdin.go`,
  `shutdown_budget.go`), `internal/api/handlers/internal_shutdown.go`, `internal/config/config_desktop.go`,
  `cmd/devlaunch/`, `internal/migration/` (`InspectSchema`, `BackupBeforeUpgrade`, `ErrSchemaNewerThanBinary`),
  `internal/service/auth_completion.go`, `internal/api/handlers/auth_browser_login.go`, `internal/media/binpath.go`, `.github/workflows/ci.yml`.
- ADR 0044, ADR 0008 (amendment), ADR 0023, ADR 0035, ADR 0038.
