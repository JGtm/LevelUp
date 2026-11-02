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
   restart" downloads the signed installer, stops the server, then installs. The data folder is never touched.
5. **Uninstall.** "Settings > Apps > LevelUp > Uninstall" (or `uninstall.exe` in the program folder). The program is
   removed; `%LOCALAPPDATA%\LevelUp` stays, so a reinstall finds the data again. Delete that folder by hand to remove
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

`powershell -File apps/desktop/scripts/e2e-install.ps1 -Installer <setup.exe>` plays steps 1 to 5 on a throwaway data
root (silent install, two launches, close by `WM_CLOSE`, clean stop, silent uninstall, data kept). It refuses to run
when LevelUp is already installed. The `desktop-e2e` job of `.github/workflows/release.yml` runs the same script.

## Releasing (maintainer)

- A `vX.Y.Z` tag runs `release.yml`: version check (tag, `Cargo.toml`, `tauri.conf.json`, server `-ldflags`), signed
  installer, end-to-end test, then the GitHub release with `latest.json`. The repository secret
  `TAURI_SIGNING_PRIVATE_KEY` (and `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` when the key has a password) must exist: without
  it the job fails before compiling. Key backup and rotation: ADR 0045, "Implementation notes (L17)".
- A push on `release/**` builds the installer unsigned and plays the end-to-end test only.
