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
