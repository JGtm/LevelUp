# Runbook — Go-live & release image flow (GHCR)

> EN-only (runbook policy). Operational reference for the production deploy pipeline
> of LevelUp on the Ionos VPS. Companion of `docs/RUNBOOK_DEPLOY_CHECKLIST.md`.

## Deploy flow (as of 2026-07-17)

`push` to `main` = automatic production deploy. The pipeline (`.github/workflows/deploy.yml`):

1. **`pre-check`** — actionlint on all workflows, `bash -n` on `scripts/*.sh`, ghost-dir guard.
   Also runs on pull requests.
2. **`build-image`** — builds the Docker image in CI (layer cache via GitHub Actions cache).
   - On **pull requests / branches**: build only (no registry push). Catches a broken
     `apt-get` / `npm ci` / `go build` on the PR, before it can break production.
   - On **push `main`**: build **and push** to GHCR
     (`ghcr.io/jgtm/levelup`, tags `latest` + `sha-<commit>`), via `GITHUB_TOKEN`
     (`packages: write`). No secret to provision — the token is auto-injected.
3. **`deploy`** (`needs: [pre-check, build-image]`, `main` push only) — SSHes to the VPS and
   runs `scripts/deploy.sh`, passing `LEVELUP_IMAGE=ghcr.io/jgtm/levelup@<digest>`
   (immutable digest of the image just built).
4. **`deploy-demo`** (`needs: deploy`) — regenerates the demo dataset.

`scripts/deploy.sh` startup path:

- If `LEVELUP_IMAGE` is set → `docker pull "$LEVELUP_IMAGE"`; on success it tags that image
  as the default compose names (`levelup-levelup:latest`, `levelup-levelup-demo:latest`) and
  starts with `docker compose up -d --no-build` (no build on the VPS).
- If the pull fails (no GHCR login, private package, network) **or** `LEVELUP_IMAGE` is
  unset → it falls back to the historical `docker compose up -d --build` (local build).
  This fallback is a **transitional kill-switch** (default flipped 2026-07-17, removal target
  2026-Q4); see the dated comment in `scripts/deploy.sh` for the removal criterion.

Because of the fallback, **the merge is safe with no prior VPS action**: the first deploy
after this change still serves production via a local build; the GHCR pull only becomes the
nominal path once the VPS can authenticate to GHCR (below).

## Activation — enable GHCR pull on the VPS (NOT done automatically)

Until one of the two options below is applied on the VPS, every deploy uses the local-build
fallback (correct, just slower and it does not catch build failures pre-prod). Pick one.

### Option A — private image + `docker login` with a read-only PAT (recommended)

Keeps the image private. One-time setup on the VPS as the `deploy` user:

1. On GitHub (owner `JGtm`): create a **fine-grained or classic PAT** with the single scope
   `read:packages`. Do NOT grant `write:packages` or `repo` — the VPS only pulls.
2. On the VPS, as the user that runs the deploy (`deploy`):

   ```bash
   # Store the PAT out of shell history, then log in non-interactively.
   echo "<PAT_read_packages>" | docker login ghcr.io -u JGtm --password-stdin
   ```

   This writes `~/.docker/config.json` for the `deploy` user. `scripts/deploy.sh` runs as
   `deploy`, so the credential is picked up automatically on the next deploy.
3. Verify (read-only): `docker pull ghcr.io/jgtm/levelup:latest` should succeed.

PAT rotation: repeat step 2 with the new token. If a pull starts failing after expiry, the
deploy silently falls back to a local build (production stays up) — watch for the
`[deploy] Pull GHCR échoué` log line as the rotation signal.

### Option B — make the package public (no VPS login)

Simpler, but the image becomes world-readable (it contains only built app code + assets,
no secrets — secrets live in `.env.local`, never baked into the image). On GitHub:

1. Go to the repo → **Packages** → `levelup` → **Package settings**.
2. **Change visibility** → **Public**.

Once public, `docker pull ghcr.io/jgtm/levelup@<digest>` works with no login and the
`--no-build` path activates on the next deploy with no VPS change.

## Rollback

- **Disable the CI image entirely**: nothing to do on the VPS — the deploy already falls back
  to a local build whenever the pull fails. To force it, revoke the PAT (Option A) or set the
  package back to private (Option B).
- **Revert the pipeline**: `git revert` the commit that introduced the `build-image` job; the
  deploy returns to a pure local-build flow.

## Removing the fallback (kill-switch retirement)

Once the GHCR pull has been the effective path for ≥ 4 consecutive deploys (no
`[deploy] Pull GHCR échoué` line in deploy logs / CI output), remove the local-build fallback
branch in `scripts/deploy.sh` together with its dated comment block, per project rule 11
(no feature flag left "for later").
