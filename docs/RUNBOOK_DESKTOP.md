# Runbook: LevelUp for Windows (desktop application)

Operator notes for installing, locating the data and uninstalling the Windows application (ADR 0045). The test on a
clean machine (plan item L17.7) follows these five steps and records witnesses (screenshots or a short video).

## Install, find the data, uninstall

1. **Install.** Run `LevelUp_X.Y.Z_x64-setup.exe` from the GitHub release (or `make desktop-installer` locally).
   Windows SmartScreen warns because the installer is not code-signed: "More info", then "Run anyway". The installation
   is per user, no administrator rights: the program goes to `%LOCALAPPDATA%\Programs\LevelUp`.
2. **First start.** Start "LevelUp" from the Start menu. The first launch creates the data folder and runs the setup
   assistant; closing the window stops the server cleanly (the window shows "LevelUp is closing" for a few seconds).
3. **Where the data is.** Everything the application writes lives in `%LOCALAPPDATA%\LevelUp`: databases
   (`titles\`), tokens (`auth\`, sealed for the Windows account), the film archive (`cache\`), logs (`logs\`,
   `desktop.log` is the shell journal). The program folder holds no user data.
4. **Update.** The application offers a new version once the server is ready; "Later" does nothing, "Install and
   restart" downloads the signed installer, stops the server, then installs. The data folder is never touched. The installer empties `web\assets` of the
   program folder first (the front's files are named by a content hash, so those of the previous version would otherwise stay next to the new ones), after
   checking that LevelUp is not running ("Cancel" removes nothing).
5. **Uninstall.** "Settings > Apps > LevelUp > Uninstall" (or `uninstall.exe` in the program folder). The program is
   removed, `web` and `config` of the program folder included (even files a previous version left there); `%LOCALAPPDATA%\LevelUp` stays, so a reinstall finds the data again. Delete that folder by hand to remove
   everything (it can reach tens of GB because of the film archive).

## Window conveniences

- **References pane.** "References" at the bottom of the sidebar (or `Ctrl+Shift+L`) opens the maps, weapons and medals
  pane to the right of the page and pushes the page; it stays open from page to page until it is closed (button, `Esc`
  inside it, or the same shortcut). Drag its left edge, or focus the edge and use the arrow keys (`Shift` for larger
  steps, `Home` and `End` for the bounds), to resize it from 280 to 640 px; the width is remembered.
- **Links that open in a new tab** (Microsoft sign-in, rules of use, "Report a problem") open in the default browser; the
  window never opens a second window. `desktop.log` records `event=new_window url=<origin only> decision=open|deny`.
- **Sidebar in icons.** The button at the foot of the sidebar, or `Ctrl+B`, folds it to a 56 px rail of icons (a tooltip on
  hover and focus; sections with tabs open a flyout menu of them, arrows and `Esc` work). Below about 1,100 px of window width
  it folds by itself and unfolds when the window widens; a choice made by hand (button or `Ctrl+B`) is never overridden by the
  width and survives a restart. Scrolling never folds it.
- **Community.** The "Rankings" tab is absent (the application follows one account, so there is nothing to rank); "Relations"
  and "Head-to-head" are derived from the local matches and stay.
- **Media without H.264.** On a Windows N or KN edition (no Media Foundation) the Media page says that clips the browser cannot
  play as they are cannot be converted, and links Microsoft's Media Feature Pack; installing it and restarting LevelUp fixes it.

## Living with the game (Settings > Application)

Everything below is in the "Application" tab of the settings, absent on the site. Nothing touches Windows until it is turned on.

- **Detect the running game** (on by default). LevelUp reads the list of Windows processes every 15 s (program names only, never the
  game's memory). While `HaloInfinite.exe` runs ("game mode"), replay building and the heavy background tasks wait for the match to
  end (including the film decoding that follows a sync), and a sync starts when the game closes (at most one per minute) provided
  automatic sync is on, which it is by default in the desktop application. A game already open when LevelUp starts is taken into account
  before any heavy task starts. The journal says `game_mode: enter` and `game_mode: leave`;
  the tab says whether the game is running.
- **Windows notifications** (off). The bell notifications (sync done, sync error, challenge, streak, record, rank) also show as Windows
  notifications when the window is minimized, hidden or behind another application, never while LevelUp is in front, and only for the
  account you signed in with (a followed friend never triggers one). A click on one brings the window back.
- **Minimize to the notification area when closing** (off). The cross hides LevelUp in the notification area and the server keeps
  running; the icon menu offers Open, Sync now and Quit (the same clean stop as the cross without the icon). The icon exists only while
  this is on. If the icon cannot be created, the cross quits instead of hiding the window.
- **Start with Windows** (off). Adds a per-user startup entry (`LevelUp` in the `Run` key of the current user, never machine-wide) that
  starts LevelUp minimized in the notification area (`--minimized`); turning it off deletes the entry. If you disabled the entry yourself in the
  Windows task manager (Startup apps), LevelUp respects it and does not turn it back on (the journal says `left_disabled`).
- **Keep the sign-in alive while the application is closed** (off). Creates the per-user scheduled task `LevelUp\Refresh-<8 hex>` (the suffix is a fingerprint of the data folder, so a test or a second
  installation on other folders never touches yours; every 6 hours,
  no administrator rights, hidden) that runs `levelup-server.exe -mode refresh-only` to rotate the Microsoft tokens; with the application
  open it exits at once (the instance lock). The tab shows whether the task is present, absent or in error. To inspect it by hand:
  `schtasks /Query /FO LIST | findstr /i "LevelUp\Refresh"` to read its exact name, then `schtasks /Query /TN "<name>" /V /FO LIST`; to remove it
  without LevelUp: `schtasks /Delete /TN "<name>" /F`. At each start with the setting on, the task is rewritten (a reinstall in another
  folder never leaves it pointing to the old one).
- **Uninstalling.** The uninstaller removes both by itself: before deleting the program files it runs `levelup-server.exe -mode uninstall-cleanup`, which
  deletes the `LevelUp` value of `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` (and its `StartupApproved` value; only when it points into the
  folder being uninstalled) and the scheduled task of the data root (only when its action runs the server of that folder: two installations of the same
  user share the default data root, so the other one's task stays). It never touches the data folder. Nothing to turn off first. If you cancel at "LevelUp
  is running", nothing has been removed yet. The entry of ANOTHER copy (a development build, another folder) is left alone: the template of Tauri deletes
  the `Run` value without looking, so the uninstaller reads it first and writes it back when it did not point into the uninstalled folder.
  **Updates.** The automatic update (the updater) never calls the old uninstaller: your entry and your task survive. A MANUAL update (the installer
  launched by hand with "uninstall before" ticked, which is the default) or a passive reinstall (`/P`) does call it: the entry and the task are removed,
  then put back at the next launch of the application if the setting is on. If the cleanup could not run (the program files were already damaged), remove
  them by hand with the commands above and by deleting the `LevelUp` value of the `Run` key.

## Your data: backup, restore, move from the site, diagnostic (Settings > Application)

- **Backup to a folder.** "Backup" in Settings > Application: type the absolute path of a folder (the page cannot open a folder
  picker; the server checks that it exists, is writable and is **not inside the application data folder**). The same disk as the
  data is accepted with a warning: a disk failure would take both. The application writes a **set** every `backup_interval` (default
  24 h, 1 h to 7 days) and keeps the last `backup_keep` (default 7): `<folder>\levelup-<YYYYMMDD-HHMM>\` holds one compressed Parquet
  file per table of every database (`db\`), the configuration, accounts and token files (`config\`) and `manifest.json` (version,
  tables, rows, SHA-256 of each file). The archived **films are not copied** (they weigh tens of GB): copy `film_archive_dir` yourself.
  "Back up now" starts a set at once. No set starts while the game runs (it waits for the end of the game). A set marked *partial*
  means a database stayed busy until the end of the pass (the next pass retries). Only `levelup-*` folders of the chosen folder are ever deleted.
- **Disk dead, new PC.** 1. Install LevelUp on the new PC and close it without signing in. 2. Plug the disk (or copy the latest
  `levelup-*` set). 3. In a console, with the application closed:
  the installer ships `levelup-server.exe` (not the `levelup` command-line tool), and the server restores by itself:
  `"%LOCALAPPDATA%\Programs\LevelUp\levelup-server.exe" -mode restore --from D:\Backups\levelup-20261103-0930 --to "%LOCALAPPDATA%\LevelUp"`
  (adjust the path of `levelup-server.exe` to where the installer put it; add `--dry-run` first to check the set; `--force` replaces databases
  already there, **never** a running application). The command verifies every checksum, refuses while the application or any server runs on that
  folder (instance lock), recreates the databases and puts the configuration back. If it refuses because of a leftover `.wal` file, the
  application was killed: start it once and close it normally, or delete that `.wal` if losing the last writes is acceptable. 4. Start LevelUp: the migrations bring
  the databases up to date. 5. **Sign in again**: the token files are sealed by the Windows vault of the *old* account and machine and
  cannot be read on another one. 6. Films: copy the archive folder back and set `film_archive_dir` if it moved.
- **Import from the site.** On the hosted site, Settings > Account > "Export to the desktop application" gives a **one-shot token valid
  15 minutes** (one export per hour). In the application: Settings > Application > "Import from the site" (or the optional block of the
  first-run assistant, once the account is tracked): the site address (https) and the token. The application downloads the archive,
  checks it (manifest, sums, and that it is **your** account's), then adds the matches it does not have (a match already present is never
  duplicated; replaying the import adds nothing). It carries your matches WHOLE (every participant, every highlight and kill, the ranks and objective stats of the match) and your player
  enrichments; not tokens or accounts, and never another tracked player's own data. If the site cannot be reached the page says why (site
  not answering, certificate refused, unknown address). Default title only for now.
- **Diagnostic report.** About window > "Diagnostic report": writes `<data folder>\diagnostics\levelup-diag-<YYYYMMDD-HHMM>.zip`
  (the end of each log, versions, settings, scheduled task, schema of each database) and shows its path with a "Copy the path" button;
  attach the zip to a report. **No token, secret or authorization code goes into it** (forbidden keys removed, log text scrubbed; a test
  checks it). Look inside before sending if in doubt.
- **Memory ceiling.** Settings > Application > "Memory": the ceiling of each DuckDB database in MiB (`0` = the server default, 512 MB;
  otherwise 1,024 to 32,768), applied at once to the open databases. It acts on DuckDB, which is what uses the memory: a heavy page or a
  recompute spills to disk beyond it instead of growing. "Current memory" shows the process working set (also `levelup.process.rss_mb`
  in `/debug/vars`).

## Replays after an update (Settings > Application)

After an update, replays already built may be behind the new version: they stay readable, but do not carry what the update adds. LevelUp never
rebuilds them by itself.

- **The banner.** On a replay page, a banner says "N replays to regenerate after the update" with **Regenerate** and **Later** (Later hides it for the
  session). It appears only when some are behind; replays whose film is no longer in the film archive are not counted (nothing to rebuild them from).
- **The same line in Settings > Application** ("Replays" card): the count, or "All replays are up to date", with the same buttons.
- **While it runs**, the banner shows `done of total` with **Pause** / **Resume**. One film at a time, in the background. It never starts a film
  while the game runs (the line says it is waiting for the game to end). **Pause** lets the film in progress finish.
- **At the end of a pass**, the replays it could not rebuild (film no longer archived) are compressed on disk, one file at a time, behind the same waits (pause,
  game, sync). The card then says how many were compressed and how many could not be (those stay readable). After an update, expect no raw `<match>.json` left in the
  replays folder (`data/cache/replays/<title>/`) and less space used. On an exFAT or FAT32 drive the files are laid without hard links
  (`replay_artifact_link_fallback_total` counts them).
- **If you close LevelUp while it runs**, it picks up again at the next opening. A paused queue does not.
- **A replay that cannot be rebuilt** (map outside the catalogue, unfinished film) is set aside and not counted again until the next launch; others
  that fail are counted ("N replays could not be regenerated") and can be offered again.
- **It gives way to a synchronization**: between two films it waits for a running sync cycle to end, and a sync (kill-source step, replay cooking) that
  finds a film being decoded postpones its decode to the next cycle (one `INFO` line, never an error). Two films are never decoded at once.
- **The count is cheap.** Each built replay has a small `<match>.meta` file beside it (versions only); the count reads those, not the replays. At the
  first count after an update from an older version the replays are read once to write them (a few seconds per thousand, behind the game gate). The
  count runs at startup, at the end of the queue, and on `POST /api/v1/desktop/replay-regen/rescan`; opening a page never starts it.
- **Going back to an earlier version**: replays built by a newer version are not offered for regeneration (the older decoder would redo them).
- **Journal** (`logs`): lines starting with `replay_regen:`; counters `replay_regen_*`, `replay_artifact_header_*` in `/debug/vars`.

## Resource pack (images, sounds, reference data, ffmpeg)

The installation is two units (ADR 0045 D-30). The **engine** (application, server, compiled front, `config/`) changes at every release and is what an
update installs. The **resource pack** (`static/`, `web/titles/`, the replay reference, `metadata-prebuilt.zip`, ffmpeg) changes rarely and is identified by its
**fingerprint** (SHA-256 of `resources.manifest.json`, computed at build; there is no version number to maintain). The first installation is complete and
works offline; an ordinary update downloads the engine only (about 30 MB instead of 152).

- **Settings > Application > Resources** shows the state: up to date, waiting for the end of a match, downloading (percent), checking, ready (applied at the next
  start of LevelUp) or failed, with a **Try again** button when the pack does not match this version or a download failed.
- **When an update changes the pack**: after you accept the update, the application downloads and verifies the new pack **first** (signature, SHA-256, every
  file against its manifest), still on the old one; only then does it download and install the engine that requires it. If the pack cannot be prepared nothing is
  installed and the application stays open; the update is offered again at the next launch. Nothing is downloaded during a match (the download waits for it).
- **At the next start** the new pack becomes the active one (renamed into place before any file is opened). If the computer stops in the middle, the next start undoes
  the half-done swap and retries; once every folder is swapped the swap is final, and a stop or a locked file during the cleanup only leaves the cleanup to the
  next start (`resources: ménage de la bascule incomplet`, a warning, not a failure).
- **A pack that must be downloaded before the window opens** (first launch without pack) may take longer than 2 minutes: as long as the percentage moves, the shell
  waits; a download that makes no progress for 2 minutes is stopped and the screen says the start timed out.
- **A pack that does not match the engine** (an interrupted update, files deleted by hand): LevelUp starts anyway on what is installed, logs a warning
  (`resources: paquet différent de celui du moteur`) and downloads the right pack in the background. A pack that is **missing** altogether is downloaded before the
  window opens; if that fails the screen says the resources are missing or incompatible: check the connection and start LevelUp again to retry.
- **The data root is never involved**: the downloaded pack is staged in the program folder (`.pack`), and the uninstaller removes the pack (through
  `-mode uninstall-cleanup`) without touching `%LOCALAPPDATA%\LevelUp`.
- **Hosted server**: the pack is inside the Docker image, never downloaded; a pack different from the engine's refuses to start (`resources_incompatible`, state in
  the 503 of the start page and in the log). The `Deploy Pre-Check` workflow proves that the manifest and the binary carry the same fingerprint and that the pack layers
  sit below the engine layers (`docker history`).
- **Journal** (`logs`): lines starting with `resources:`; the shell logs `event=resources_requested`, `event=server_event kind=resources ...`,
  `event=update_download_failed err=... resources_failed code=...` when a pack refuses an update.

## Signing in (Microsoft account)

The setup assistant's first step opens **the default browser** on the Microsoft sign-in page ("Sign in with Microsoft"); after
the consent the browser shows "Signed in, you can close this tab" and the application moves on by itself (ADR 0045 D-22). The
return address is `http://localhost:<port>/auth/xbox/callback`, where `<port>` is the application's own stable port; the Azure
application must list `http://localhost/auth/xbox/callback` (a port is accepted on `localhost`). If the browser cannot be
opened, "Can't open the browser? Use a code instead" falls back to the code to type on the Microsoft page. A failure page in the
browser carries a short error code (`state_expired` after 10 minutes, `oauth_denied` if the consent was refused,
`code_exchange_failed` when Microsoft did not answer): close the tab and start again from the application. Never re-capture
tokens to "repair" a sign-in (ADR 0023). The `code` and the `state` never reach the logs.

## Automated proof

`powershell -File apps/desktop/scripts/e2e-install.ps1 -Installer <setup.exe> [-EngineInstaller <engine-setup.exe>]` plays steps 1 to 5 on a throwaway data
root (silent install, pack checked against its manifest, two launches, with `-EngineInstaller` an engine-only update that must keep the pack, remove two planted files of a "previous front" in `web\assets`, leave the list of front files unchanged and every `assets/` file named by `web\index.html` present, and a third launch, close by `WM_CLOSE`, clean stop, silent uninstall, data kept, and the uninstaller's cleanup
call proven by a witness file: `LEVELUP_UNINSTALL_WITNESS`, whose line must carry the full report: `exit=0`, `ok=true`, `ressources=<n>-fichiers`, `moteur=<n>-fichiers` (n >= 3), `demarrage=absent` or
`foreign` (never `removed`), `tache=absent`, the task name of the throwaway root, the root and the program folder). It creates no startup entry and no
scheduled task. It refuses to run when LevelUp is already installed. The `desktop-e2e` job of `.github/workflows/release.yml` runs the same script.
**Limit of the update step**: the engine-only installer is of the installed version (version B simulated), so this is not a real upgrade from one published version to the next; the old front files are planted by the script. **Not played by the script**: the uninstaller's branch "`LEVELUP_DATA_DIR` absent" (it sets `LEVELUP_DATA_DIR` itself to the default
`%LOCALAPPDATA%\LevelUp`; playing it would aim the cleanup at the real data root of the machine), nor the rewrite of a foreign `Run` value. Both are held
by the text guards of `apps/desktop/src-tauri/tests/textes_de_configuration.rs` (presence, form and order of the lines of the hooks).

## Releasing (maintainer)

- A `vX.Y.Z` tag runs `release.yml`: version check (tag, `Cargo.toml`, `tauri.conf.json`, server `-ldflags` including the pack fingerprint), the pack
  (manifest, fingerprint, archive), the **full** installer (first installation, download page) and the **engine-only** installer (what the updater installs), the
  signatures (engine installer and pack, same key), end-to-end test (full installation, then an engine-only update), then the single `updates` release (engine installer, pack and its signature, `latest.json`, which
  points at the engine installer and carries the `resources` block of the pack) and the version page (full installer, servers). An installed 8.1.0 is not offered this update and takes the full installer once. The repository secret
  `TAURI_SIGNING_PRIVATE_KEY` (and `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` when the key has a password) must exist: without
  it the job fails before compiling. Key backup and rotation: ADR 0045, "Implementation notes (L17)".
- A push on `release/**` builds the installer unsigned and plays the end-to-end test only.
