# Runbook — Premier déploiement Go en production (cutover Python → Go)

> Rédigé le 2026-06-03. Cible : VPS Ionos `/opt/levelup` (lvelup.info + demo.lvelup.info).
>
> Contexte : `main` = encore Python v6.5.0 ; toute la stack Go vit sur la branche de
> migration (~2377 commits devant main). Ce runbook couvre le cutover Go→main et le
> premier boot Go en prod. L'ancien `scripts/deploy.sh` (Streamlit) a été réécrit pour
> le Go le 2026-06-03 (healthcheck `GET :8000/health`, plus de Python/port 8501).

## Chaîne de déploiement (rappel)

```
push main → .github/workflows/deploy.yml → SSH deploy@VPS → scripts/deploy.sh
            → git reset --hard origin/main → docker compose up -d --build
            (Dockerfile = Vite + Go CGo/DuckDB, port 8000, GET /health)
deploy.yml job 2 (deploy-demo) → levelup seed-demo → docker compose up -d levelup-demo
```

## Phase 0 — Pré-vol LOCAL (avant tout upload / cutover)

- [ ] CI verte sur la branche : `cd apps/go-api && go test ./... && go vet ./...`
      puis `cd apps/web && npm run typecheck && npm run lint`.
- [ ] Chemins média en DB bien **relatifs** (`{slug}/...`, pas `C:\...`) :
      `go run apps/go-api/cmd/inspect_bp/main.go <shared_social.duckdb>` →
      `SELECT file_path FROM media_files LIMIT 20`. Si absolus Windows → relancer
      le backfill captures-base **localement** avant l'upload (sinon irrésolus sous Linux).
- [ ] (Optionnel — si les DBs ont un historique legacy/Python) Rebuild ART préventif
      LOCAL, serveur arrêté : `go run ./cmd/force_rebuild_art --all true` (CTAS swap
      non destructif ; go/no-go : rows avant == rows après). Skip si les DBs n'ont
      jamais été écrites que par le path batch Go. NB : ce binaire **n'est pas dans
      l'image Docker** → il se lance en local, d'où le rebuild avant upload.

## Phase 1 — Config VPS AVANT le deploy (`/opt/levelup`)

Seule la **config prod** est requise avant le boot — les **données viennent après** (Phase 5,
via stop/start). `.env.local` est préservé par `git clean --exclude=.env.local`.

- [ ] `.env.local` — ajouter le bloc **PRODUCTION** (cf. `.env.local.example`) :
      - `LEVELUP_ENV=production`
      - `LEVELUP_AUTH_MODE=xbox`
      - `LEVELUP_SESSION_SECRET=` ← `openssl rand -base64 48`
      - `LEVELUP_CORS_ORIGINS=https://lvelup.info` (+ www / autres origines réelles)
      - `LEVELUP_OAUTH_REDIRECT_URI=https://lvelup.info/auth/xbox/callback`
        (SSO Xbox Authorization Code Flow ; chemin **racine**, pas `/api/v1`)
- [ ] **Azure** (si SSO Xbox redirect-based activé) : App registration → Authentication →
      plateforme **Web** → ajouter le redirect URI **identique au caractère près** à
      `LEVELUP_OAUTH_REDIRECT_URI`. Scopes `Xboxlive.signin Xboxlive.offline_access`.
      Sans ça : `AADSTS50011` à l'étape `/authorize`. Le callback est exposé à la racine
      (`/auth/xbox/callback`) ET sous `/api/v1` — aucune réécriture `/auth/` à prévoir
      côté reverse proxy, `location / → backend` suffit.

## Phase 2 — Cutover (la branche Go devient `main`)

> ⚠️ **Topologie** : le dossier de travail Go est un **worktree lié** au repo Python
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp` (`.git` = fichier → `LevelUp/.git`). Le
> store d'objets vit dans le dossier Python. **Ne PAS déplacer/archiver `LevelUp/` avant la
> Phase 6.** La branche est entièrement pushée sur origin. `.env.local`, `app_settings.json`
> et les DBs/médias sont **locaux uniquement** → backup avant toute manip.

- [ ] Commit + push de tout le travail en cours (dont les patchs go-live de cette session).
- [ ] Cutover via merge `-s ours` : contenu Go conservé, historique Python préservé, **pas
      de force-push**. Sens critique — être sur la branche Go et merger `main` dedans :
      ```bash
      # Depuis le worktree Go (dossier courant), sur la branche Go, tout commité :
      git merge -s ours main -m "merge(-s ours): la branche Go devient la source de main"
      git branch -f main HEAD     # main (non checkout ailleurs) fast-forward vers le merge
      git push origin main        # déclenche deploy.yml
      ```
      Piège à éviter : faire `git checkout main && git merge -s ours <go>` ferait l'inverse
      (garderait le Python). C'est bien la branche Go qui doit être « ours ».

## Phase 3 — Déploiement (automatique via deploy.yml)

`deploy.sh` sur le VPS : `git reset origin/main` → rappel garde-fou prod (WARN si
`.env.local` pas en production) → stubs demo → `docker compose up -d --build` →
healthcheck `GET :8000/health` (**bloquant**) → check demo `:8001` (warn-only).
Puis job `deploy-demo` : regen données démo anonymisées + restart `levelup-demo`.

**L'app boote À VIDE** : sans données, le serveur auto-provisionne `metadata`, crée le
warehouse dir et démarre en mode `/setup` ; `/health` répond **200** → le deploy réussit même
sans DBs ([main.go:339-346](../apps/go-api/cmd/server/main.go#L339-L346)). Les données arrivent en Phase 5.

## Phase 4 — Vérifications post-deploy (app à vide)

- [ ] `docker compose ps` : `levelup` healthy.
- [ ] `docker compose logs --tail=50 levelup` → **PAS** de `configuration non sûre pour
      un déploiement multi-user exposé` (= garde-fou armé) ni de `FATAL ... index`.
- [ ] `curl https://lvelup.info/health` → 200 (version DuckDB ; 0 match attendu tant que pas de données).
- [ ] `demo.lvelup.info` répond.
- [ ] Une requête mutante depuis le navigateur (login Xbox) passe (CORS/CSRF OK pour l'origine prod).

## Phase 5 — Population des données (APRÈS deploy confirmé, via stop/start)

> Règle dure : ne jamais déposer/écraser un `.duckdb` dans un conteneur **qui tourne**
> (DuckDB tient ses handles + locks → corruption / `different configuration`). Toujours
> **stop → upload → start**. Pré-requis : Phase 0 faite sur les DBs (chemins média relatifs ;
> ART rebuild local si legacy).

```bash
docker compose stop levelup        # libère les handles DuckDB
#   ... upload des fichiers ci-dessous dans /opt/levelup/data/ ...
docker compose start levelup       # rouvre proprement TES bases
```

- [ ] `db_profiles.json` — profils joueurs (avec `xuid`).
- [ ] `app_settings.json` — `media_captures_base_dir` = **`/app/data/media`** (remplace la valeur Windows).
- [ ] Les **DBs DuckDB** — layout title-scopé, **identique à ton `data/` local**. Copier
      `data/global/` + `data/titles/halo_infinite/` vers `/opt/levelup/data/` en préservant
      l'arborescence (exclure `data/demo/`, régénéré par `deploy-demo`) :
      ```
      data/
      ├── global/xbox_aliases.duckdb
      └── titles/halo_infinite/
          ├── warehouse/{metadata,shared_matches_v2,shared_pve,shared_social}.duckdb
          └── players/{Gamertag}/stats.duckdb   (1 par joueur)
      ```
- [ ] Les **médias** : copier ta base locale de captures `{Gamertag}/*` vers
      `/opt/levelup/data/media/{Gamertag}/` (= base `/app/data/media`, cohérent avec les
      chemins relatifs `{slug}/{fichier}` en DB). Dossiers runtime (`logs`, `cache`,
      `sessions`, `wal`) créés par le conteneur — ne pas uploader.
- [ ] Vérifs données (après `start`) : `curl https://lvelup.info/health` → matchs > 0 ;
      un joueur et un média s'affichent.

## Rollback

- **Config non sûre** : le serveur refuse de booter (`os.Exit 1`) avec le détail des
  réglages manquants — corriger `.env.local` et relancer.
- **Incident ART** : re-rebuild **local** + re-upload de la DB (binaire hors image).
  Fallback d'urgence : `LEVELUP_PERSIST_BATCH=0` (réactive l'UPSERT ART-unsafe —
  temporaire uniquement, WARN au boot).
- **Retour Python** : redéployer l'ancien `main` Python — rollback structurel lourd,
  à éviter.

## Phase 6 — Réorganisation des dossiers (APRÈS la population des données)

But : un dossier standalone `LevelUp` = projet Go + data, Python archivé. Le dossier courant
étant un **worktree** de `LevelUp/.git`, on ne peut ni le renommer en place ni archiver le
dossier Python sans le détacher d'abord. Approche sûre (clone standalone + déplacement des
données, **aucune chirurgie git**) :

```bash
# 1. Backup d'abord : data/ (DBs+médias), .env.local, app_settings.json, db_profiles.json.
# 2. Clone standalone (main = contenu Go après Phase 2) :
git clone https://github.com/JGtm/LevelUp.git C:/Users/Guillaume/Downloads/Scripts/LevelUp-standalone
# 3. DÉPLACER (rename, instantané sur le même disque — pas de copie des Go de médias) les
#    données locales non trackées du worktree vers le clone :
#      data/  .env.local  app_settings.json  db_profiles.json
# 4. Vérifier que le clone démarre (serveur + data + médias OK).
# 5. Vérifier qu'aucun autre worktree (film-weapons, no-streamlit, .claude/worktrees/*) n'a
#    de travail non pushé, PUIS archiver l'ancien dossier Python `LevelUp/`
#    (zip ou rename → LevelUp-python-archive). À ce stade le clone est autonome.
# 6. Renommer LevelUp-standalone → LevelUp.
```

Note : « déplacer » (et non copier) les données évite de dupliquer plusieurs Go de médias.
Alternative « garder l'octet près du dossier courant » (conversion in-place worktree→standalone)
= chirurgie git fragile, déconseillée tant que le clone standalone fait le même résultat.

## Amélioration recommandée (post-go-live, non bloquante)

`deploy.sh` build l'image **sur le VPS** (Vite + Go CGo/DuckDB = lent). ROI réel :
build de l'image en CI → push GHCR → VPS fait `docker compose pull` au lieu de `--build`
(image reproductible, deploy rapide, échec de build attrapé avant la prod).
