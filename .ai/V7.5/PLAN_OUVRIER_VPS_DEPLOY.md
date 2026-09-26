# Plan — Deploiement de l'ouvrier de rejeu sur le VPS puissant (csstat)

> Lot `ouvrier-vps`, branche `wt/ouvrier-vps`, base `c67b58f6b`. Ecrit le 2026-08-25.
> Execution sous le contrat du skill `plan-execution` (ordre strict, une etape a la fois,
> aucun report d'une action executable maintenant, chaque case statuee `[x]`/`[~]`/`[!]`,
> zero fix hors perimetre). En cas de divergence, le skill fait foi.

## Objectif

Livrer les FICHIERS VERSIONNES qui permettront de deployer `apps/go-api/cmd/replay-worker`
sur le second VPS (« csstat ») a chaque push sur `main` : un job GitHub Actions, un script
de deploiement joue sur csstat, une unite systemd versionnee, un runbook d'activation.

Rien n'est active dans ce lot. Le workflow ne s'executera qu'apres fusion sur `main`
(release v7.5) et l'unite est livree DESACTIVEE : le lot pose la mecanique, pas la mise en
service.

## Critere de succes

1. Les 4 fichiers existent, se tiennent, et decrivent un chemin de deploiement complet.
2. Gates verts, exit codes reels consignes (actionlint 1.7.12 pinne, `bash -n`, archlint).
3. Le garde-rail `ci_deploy_triggers_test.go` reste vert : aucun motif ajoute au
   `paths-ignore` de `deploy.yml`.
4. Registre `.ai/V7.5/REGISTRE_REPORTS.md` + `.ai/thought_log.md` a jour, commits propres,
   aucun push.

## Repartition superviseur / agent

| Qui | Quoi |
|---|---|
| **Superviseur** | Tout le provisionnement machine : compte `deploy` sur csstat, toolchain Go dans `/usr/local/go`, clone https public du depot vers `/opt/levelup`, secrets GitHub `VPS2_HOST` / `VPS2_SSH_KEY`, sudoers restreint, pose de `/etc/levelup-worker.env`, installation (par LIEN) de l'unite systemd en etat DESACTIVE, jeton stage sur le VPS prod. |
| **Agent (ce lot)** | Uniquement des fichiers du depot, dans le worktree. Aucune connexion SSH, aucun `gh secret`, aucun acces reseau hors telechargement d'actionlint pour le gate. Aucun push, aucun rebase, aucun merge. |

## Faits de reconnaissance (etablis par le superviseur, cites, non re-verifies)

- **VPS ouvrier « csstat »** : Debian 12 bookworm x86_64, 6 vCPU, 7,7 Gio RAM, 129 Go libres,
  gcc 12.2 + git 2.39 + systemd 252 presents. **Go ABSENT** — le superviseur installe la
  toolchain stable dans `/usr/local/go` (d'ou le `PATH` explicite du script, cf. D2).
- **Connexion CI** : utilisateur `deploy`, secrets GitHub `VPS2_HOST` / `VPS2_SSH_KEY`.
  Ce lot ne fait que les REFERENCER.
- **Depot public** (`JGtm/LevelUp`) : clone https sans cle sur csstat, a `/opt/levelup`,
  propriete `deploy`.
- **URL prod** : `https://lvelup.info` ; l'ouvrier pollera `https://lvelup.info/api/v1/internal`.
- **VPS prod « lvelup »** : rien a y faire dans ce lot (jeton stage par le superviseur,
  inerte).

## Decisions (fermes — ne pas re-decider)

- **D1 — job `deploy-worker` dans `.github/workflows/deploy.yml`.** `needs: [deploy]`,
  condition `github.ref == 'refs/heads/main' && !failure() && !cancelled()` (meme forme que
  le job `deploy` : la condition n'est PAS cosmetique, elle couvre le cas d'un `needs`
  legitimement SAUTE — cf. commentaire de `deploy`, lignes 244-254). `appleboy/ssh-action@v1`,
  `host: ${{ secrets.VPS2_HOST }}`, `username: deploy`, `key: ${{ secrets.VPS2_SSH_KEY }}`,
  `command_timeout: 40m` (dimensionne pour un build Go CGO A FROID : le defaut appleboy de
  10 min avait tue le build v7.2.0 en plein `go build`, cf. commentaire du job `deploy`),
  `timeout-minutes: 50` (doit rester AU-DESSUS du `command_timeout`, meme garde que `deploy`).
  Script = `bash /opt/levelup/scripts/deploy-worker.sh`. **AUCUN motif ajoute au
  `paths-ignore`** (invariant D29 documente en tete du workflow, garde-rail
  `apps/go-api/internal/archlint/ci_deploy_triggers_test.go`).
- **D2 — `scripts/deploy-worker.sh` versionne**, joue SUR csstat par le job. `set -euo pipefail`,
  en-tete de doc dans le style de `scripts/deploy.sh`. Idempotent : `git fetch` + `reset --hard
  origin/main`, `go build` du paquet `./cmd/replay-worker` (CGO requis — DuckDB transitif ;
  `PATH` etendu a `/usr/local/go/bin`), installation dans `/opt/levelup/bin/replay-worker` par
  REMPLACEMENT ATOMIQUE (build vers un fichier temporaire du meme repertoire puis `mv`),
  rafraichissement de l'unite systemd si le fichier versionne a change
  (`sudo systemctl daemon-reload`), puis :
  - `systemctl is-enabled levelup-worker` = `enabled` -> `sudo systemctl restart levelup-worker` ;
  - sinon message explicite « service desactive : binaire mis a jour, pas de restart » et
    **EXIT 0**. C'est le chemin NOMINAL tant que l'activation n'a pas eu lieu.
- **D3 — unite systemd versionnee `packaging/systemd/levelup-worker.service`** : `User=deploy`,
  `EnvironmentFile=/etc/levelup-worker.env` (root 600 — lu par systemd AVANT le drop de
  privileges, c'est voulu), `ExecStart=/opt/levelup/bin/replay-worker --url
  https://lvelup.info/api/v1/internal --id csstat --mem-limit-gib 3 --work
  /var/lib/levelup-worker/work`, `StateDirectory=levelup-worker`,
  `WorkingDirectory=/opt/levelup`, `Restart=on-failure` + `RestartSec` raisonnable,
  `[Install] WantedBy=multi-user.target`.
- **D3-bis — AMENDEMENT OBLIGATOIRE A D3 (verifie sur pieces, 2026-08-25)** : l'`ExecStart`
  de D3 ne demarrerait JAMAIS. `apps/go-api/cmd/replay-worker/main.go:73` declare
  `--repo` avec pour defaut `os.Getenv("LEVELUP_REPO_ROOT")`, et `main.go:87-90` sort en
  **exit 2** (« racine du depot absente ») si la valeur est vide. `WorkingDirectory=` ne
  renseigne PAS cette racine. L'unite porte donc `--repo /opt/levelup` EXPLICITE dans
  l'`ExecStart` : ecrit dans le fichier versionne, il ne depend pas du contenu d'un fichier
  d'environnement pose a la main sur la machine.
- **D3-ter — installation de l'unite PAR LIEN, consequence du sudoers restreint.** Le compte
  `deploy` a un sudoers limite a `daemon-reload` et `restart levelup-worker` : il ne peut donc
  PAS recopier l'unite dans `/etc/systemd/system/`. « Rafraichir l'unite si le fichier
  versionne a change » n'a de sens que si `/etc/systemd/system/levelup-worker.service` est un
  LIEN vers `/opt/levelup/packaging/systemd/levelup-worker.service` : le `git reset` met alors
  a jour le contenu pointe, et `daemon-reload` suffit a le prendre en compte. Le script gere
  les trois etats reels (lien / copie divergente / absent) et n'echoue sur aucun.
- **D4 — `docs/RUNBOOK_REPLAY_WORKER.md`, EN-ONLY** (regle du depot n°15 : ADRs et runbooks
  sans traduction FR). Architecture, etat provisionne, sequence d'activation a la release
  v7.5, rollback, et le point nginx a verifier sur la prod.
- **D5 — aucune activation dans ce lot** : aucun changement de comportement de la prod ; le
  workflow ne s'executera qu'apres fusion sur `main`. Dit dans les commentaires du job.

## Etapes

### Etape 1 — Plan (ce fichier)

- [x] Lire AVANT d'ecrire : `scripts/deploy.sh`, `.github/workflows/deploy.yml` (en entier),
      `.github/workflows/test-deploy-precheck.yml`, `apps/go-api/cmd/replay-worker/main.go`,
      `job.go`, `apps/go-api/internal/replaybuild/placement.go`,
      `apps/go-api/internal/api/handlers/build_worker.go`, `replay_local_gate.go`,
      section « COMMANDE DE REMEDIATION » de `.ai/V7.5/PLAN_OUVRIER_DISTANT.md`,
      `packaging/nginx/*.conf`, `apps/go-api/internal/archlint/ci_deploy_triggers_test.go`.
- [x] Ecrire objectif, critere de succes, faits de recon, D1-D5, etapes, gates, reprise.
- [x] Signaler immediatement le defaut D3 (`--repo` manquant) et la contrainte sudoers
      (installation par lien) — faits AVANT toute ecriture de l'unite.

Gate : le fichier existe et porte les cases des etapes 2 a 4.

### Etape 2 — Les 4 fichiers

- [x] `scripts/deploy-worker.sh` (D2 + D3-ter).
- [x] `packaging/systemd/levelup-worker.service` (D3 + D3-bis).
- [x] Job `deploy-worker` dans `.github/workflows/deploy.yml` (D1 + D5).
- [x] `docs/RUNBOOK_REPLAY_WORKER.md` (D4).

Gate : les 4 fichiers existent ; aucun emoji dans les fichiers livres (regle n°4) ; le
`paths-ignore` de `deploy.yml` est inchange (verification par `git diff`).

**Gate PASSE (2026-08-25)** : `git diff -- .github/workflows/deploy.yml` = 45 insertions,
**0 ligne touchant `paths-ignore`** ; recherche de plages Unicode emoji sur les 4 fichiers
livres = aucune correspondance ; `git status --short` ne montre que les fichiers de ce lot.

### Etape 3 — Gates

Commandes NUES, exit code reel consigne. **Jamais de pipe pour rendre un verdict** (lecon
payee deux fois : `cmd | grep | head` -> SIGPIPE -> faux vert).

- [x] actionlint **1.7.12 EXACT**, memes deux lignes que le step « Validation workflows » de
      `deploy.yml` / `test-deploy-precheck.yml` :
      ```bash
      bash <(curl -sSf https://raw.githubusercontent.com/rhysd/actionlint/v1.7.12/scripts/download-actionlint.bash) 1.7.12
      ./actionlint -color
      ```
      **EXIT 0** (2026-08-25). Telechargement : `Done: 1.7.12`, `built with go1.26.1 compiler
      for windows/amd64`, exit 0. Analyse : **aucune sortie, exit 0** (zero finding sur les
      8 workflows). Binaire supprime de l'arbre apres le gate (`rm -f actionlint.exe`),
      `git status` verifie.
- [x] `bash -n scripts/deploy-worker.sh` -> **EXIT 0**.
- [x] `shellcheck scripts/deploy-worker.sh` -> **DISPONIBLE** (winget koalaman.shellcheck),
      **EXIT 0**, aucune sortie.
- [x] `cd apps/go-api && go test ./internal/archlint/` -> **EXIT 0**
      (`ok levelup/go-api/internal/archlint 3.646s`) : le garde-rail des declencheurs D29
      reste vert avec le diff.
- [~] `systemd-analyze verify` : **indisponible sous Windows**. A jouer par le superviseur
      sur csstat : `systemd-analyze verify /opt/levelup/packaging/systemd/levelup-worker.service`.
      Renvoi ecrit aussi dans `docs/RUNBOOK_REPLAY_WORKER.md` §6.

Toutes les commandes ont ete jouees NUES, exit code lu directement — aucun pipe pour rendre
un verdict.

### Etape 4 — Cloture

- [x] Ligne au registre `.ai/V7.5/REGISTRE_REPORTS.md` (lot ouvrier-vps : fichiers livres,
      activation = condition de reprise a la release).
- [x] Entree en tete de `.ai/thought_log.md` ([2026-08-25], titre, statut, decision,
      resultats avec exit codes, prochaine etape).
- [x] Toutes les cases de ce plan statuees.
- [x] 2 commits : `ouvrier-vps(V1)` (plan) et `ouvrier-vps(V2)` (les 4 fichiers + cloture).
      Uniquement les fichiers de ce lot. **Aucun push.**

## Ce que le superviseur doit provisionner EXACTEMENT (contrat du script)

Le script et l'unite livres attendent ces chemins et ces droits, au caractere pres :

| Element | Valeur attendue |
|---|---|
| Racine du clone | `/opt/levelup`, proprietaire `deploy` |
| Toolchain Go | `/usr/local/go/bin/go` (le script etend `PATH` a `/usr/local/go/bin`) |
| Binaire produit | `/opt/levelup/bin/replay-worker` (repertoire cree par le script) |
| Nom d'unite | `levelup-worker` (aucune variante : le script compare `is-enabled` a `enabled`) |
| Installation de l'unite | **par lien** : `sudo systemctl link /opt/levelup/packaging/systemd/levelup-worker.service` puis `sudo systemctl daemon-reload`, et **rester DESACTIVEE** |
| Fichier d'environnement | `/etc/levelup-worker.env`, root, 0600, portant `LEVELUP_BUILD_WORKER_TOKEN=` |
| Dossier d'etat | cree par systemd via `StateDirectory=levelup-worker` -> `/var/lib/levelup-worker` (le sous-dossier `work/` est cree au premier job) |
| sudoers de `deploy` | `systemctl daemon-reload` **et** `systemctl restart levelup-worker`, sans mot de passe (le script les appelle en non-interactif) |
| Secrets GitHub | `VPS2_HOST`, `VPS2_SSH_KEY` |

Le jeton pose dans `/etc/levelup-worker.env` doit etre BYTE-IDENTIQUE a celui qui sera
branche cote prod (`/opt/levelup/.env.local` sur lvelup) : la comparaison est a temps
constant, toute divergence rend 401.

## Ce que la verification sur pieces a etabli (a citer, pas a re-verifier)

1. **`--work` explicite -> les morceaux sont nettoyes apres chaque job.** `main.go:99-108` :
   le defaut de `--work` est le cache film du depot ; `keepsFilms = sameDir(workDir, repoCache)`.
   Avec `--work /var/lib/levelup-worker/work` (different de `/opt/levelup/data/cache`),
   `keepsFilms` est faux, donc `cleanupFilm` (`job.go:282-299`) supprime le dossier des
   morceaux apres CHAQUE job. Confirme : un ouvrier distant ne remplit pas son disque.
2. **Jeton** : `--token` a pour defaut `os.Getenv("LEVELUP_BUILD_WORKER_TOKEN")`
   (`main.go:71`) ; absent -> **exit 2** (`main.go:83-86`). Il vient donc de
   `/etc/levelup-worker.env`, jamais de la ligne de commande (pas de secret dans `ps`).
3. **Exit codes** : `0` arret propre ; `1` arret sur erreur de protocole (`main.go:125-128`) ;
   `2` configuration manquante (jeton ou racine du depot) ; `3` `memory_exceeded`
   (`job.go:47`, `exitCodeMemoryExceeded`).
4. **`Restart=on-failure` est SUR vis-a-vis de l'exit 3** : la sentinelle appelle
   `reportMemoryExceeded` (`job.go:135-138`), qui joue `completeMemoryExceeded`
   (`job.go:143-153`) — le compte rendu `error_code=memory_exceeded` part au serveur sur un
   contexte FRAIS — **PUIS** `os.Exit(3)`. Le job est donc deja rendu `failed` avec son motif
   explicite avant l'arret : un redemarrage ne le reprend pas en boucle, il repart sur la file.
5. **Ce que le jeton d'ouvrier ouvre** : QUATRE routes sous `/api/v1/internal/build-queue/`
   (`claim`, `artifact`, `complete`, `heartbeat` — `build_worker.go:1-27`, `Mount` 88-103),
   toutes derriere le middleware `RequireWorkerToken` (`build_worker.go:254-271`, comparaison
   a temps constant). Sans jeton configure : **503** ; jeton non conforme : **401**. Ni token
   Halo, ni base, ni port entrant chez l'ouvrier.
6. **nginx prod (fichiers versionnes)** : `packaging/nginx/levelup.conf:85-90` proxifie TOUT
   `location /api/` vers `http://127.0.0.1:8000` (`client_max_body_size 2g`,
   `proxy_read_timeout 3600s`). **Aucune directive `deny`/`allow`, aucune mention de
   `internal`** dans les deux confs du depot (`levelup.conf`, `demo.conf`) : rien n'y bloque
   `/api/v1/internal` depuis l'exterieur, et le corps volumineux + les timeouts longs
   couvrent le depot d'artefact. **Limite honnete** : ces confs sont installees A LA MAIN
   (`sudo cp`, cf. leur en-tete) et completees par certbot ; le fichier du depot ne prouve
   pas ce qui tourne. D'ou la commande de verification ecrite dans le runbook.
7. **Garde local du rejeu** : `replay_local_gate.go:74` — `LEVELUP_REPLAY_PUBLIC=1` leve un
   middleware (`LocalOnlyReplay`, lignes 110-119) qui repond **404 `replay_not_available`** a
   toute requete dont l'adresse de connexion TCP n'est pas loopback. Il ne concerne QUE les
   routes de rejeu servies au navigateur ; il n'a aucun effet sur le protocole ouvrier.
8. **Garde-rail declencheurs** : `ci_deploy_triggers_test.go` n'inspecte que le bloc
   `on: push:` des deux workflows (`pushTriggerBlock`, lignes 175-230). Ajouter un JOB ne le
   touche pas ; ajouter un motif au `paths-ignore` de `deploy.yml`, si.

## Decouvertes (consignees, NON traitees — regle n°7)

1. **Deux emplacements pour les unites systemd.** `scripts/systemd/` existe deja (units
   `levelup-docker-prune`, `levelup-restic-backup` + README). D3 impose `packaging/systemd/`.
   La distinction est defendable — `scripts/systemd/` = unites de l'HOTE prod recopiees a la
   main (« NOT managed by git deploy », README ligne 6), `packaging/systemd/` = artefact
   deploye par la CI et rafraichi par lien — mais elle n'est ecrite nulle part. Non traite
   dans ce lot : le fichier livre porte un renvoi en commentaire.
2. **`scripts/deploy.sh` et `deploy.yml` contiennent des emojis** dans des chaines de log,
   alors que la regle n°4 du CLAUDE.md les interdit dans les fichiers versionnes. Dette
   preexistante, hors perimetre. Les fichiers de ce lot n'en portent aucun.
3. **Les URL pre-signees vieillissent DANS la file** (deja au registre via
   `PLAN_OUVRIER_DISTANT.md` §6.2). Consequence directe pour l'ouvrier distant : un job qui
   attend plus longtemps que la validite du CDN Azure echouera au telechargement. Rappele
   dans le runbook, non traite ici.

## Protocole de reprise

L'avancement, ce sont les cases de ce fichier. A la reprise : relire le contrat
`plan-execution`, puis ce plan, reprendre a la premiere case non statuee de l'etape courante,
apres avoir rejoue le gate de l'etape (le code a pu bouger : rouvrir les fichiers cibles
avant d'ecrire). Les decisions D1 a D5 (et leurs amendements D3-bis / D3-ter) sont FERMES :
elles ne se re-decident pas, elles s'appliquent.
