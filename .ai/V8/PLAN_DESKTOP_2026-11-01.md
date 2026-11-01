# Plan — Version Desktop de LevelUp (Tauri 2 + serveur Go en sidecar) — 2026-11-01

Statut : **plan, rien d'implémenté**. Sources : `.ai/ETUDE_DESKTOP_TAURI_2026-10-26.md` (l'étude)
et `.ai/AUDIT_COMPLEMENTAIRE_DESKTOP_2026-10-27.md` (l'audit, chantiers C1 à C12). Exécution sous le
contrat du skill `plan-execution` (ordre strict, aucune étape différée, statut `[x]` / `[~]` / `[!]`
sur chaque item, zéro fix hors périmètre, découvertes consignées en §10). Relecture `adversarial-review`
sur les lots marqués « à risque ».

Révision 2 (2026-11-01, soir) : contre-vérification par un relecteur en contexte frais (lecture seule),
17 constats intégrés ; les trois plus graves étaient le port aléatoire qui vide le stockage du navigateur
(A15), le lot C2 qui aurait cassé la relance après un arrêt forcé (verrou DuckDB réel, §9.1) et un contrat
coquille/serveur non observable (événements et marqueurs d'arrêt inexistants, §6).

Le pilote est orchestrateur seul : chaque lot est un brief fermé à un sous-agent (modèle indiqué par lot),
un worktree par agent, GOCACHE dédié (`go-build-<lot>`), pas de commit par l'agent, pas de `git add -A`.
Le pilote vérifie le compte rendu sur pièces avant de cocher. Les items marqués **(P)** sont au pilote
(push, run CI, mutation), les items **(U)** à l'utilisateur (gate visuel, décisions).

---

## 1. Objectif et critère de succès

Produire une application Windows installable (NSIS, par utilisateur), mono-utilisateur, autonome, qui
lance le serveur Go existant en processus enfant, affiche l'interface web existante dans une WebView2,
se ferme proprement (aucun fichier `*.wal` restant après l'arrêt, aucune récupération au lancement
suivant), se met à jour par l'updater signé de Tauri, et dont **le serveur reste le même binaire que
l'hébergé** (aucun contrat qui diverge).

Critères mesurables, par phase :

| Phase | Critère de succès |
|---|---|
| **P0 — serveur** | Les 12 chantiers de l'audit + port stable, mode `desktop`, verrou d'instance Go, séquence réveil livrés ; `go test ./...`, `-tags=integration -p 1` sur `migration`, `persist`, `sync` ; CI verte y compris la voie Windows CGO (C9) ; aucun changement de comportement sur le VPS (témoins §8 rejoués identiques). |
| **P1 — coquille** | `apps/desktop` construit en CI Windows ; test de contrat coquille/serveur vert des deux côtés (faux sidecar, faux lanceur) ; installeur NSIS produit par `release.yml` ; test de bout en bout CI : installation silencieuse, lancement, page servie, fermeture par `WM_CLOSE`, aucun `*.wal` restant, relancement. |
| **P2 — expérience** | Chaque fonction de l'étude §5 et §6 livrée avec son test côté Go et, pour la coquille, un test contre le faux sidecar. Gate visuel : témoins nommés par l'utilisateur. |

Hors périmètre : P3 (macOS, contribution aux rejeux, liens profonds), réécriture de l'historique git
des WAV (C11), chiffrement des jetons (D-8), binaire autonome C10 (D-11).

## 2. Rapport aux chantiers en cours

- **Publication des briques** (`.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`, lancement prévu le
  2026-11-02) : ses lots 2 et 3 réécrivent des imports dans `apps/go-api` (dont `cmd/server/main.go`),
  déplacent `sync/xbox_client.go` vers `libs/haloapi/xboxlive`, touchent `.air.toml`, et réservent
  l'**ADR 0043**. Règle d'ordre :
  - la **vague 1** de ce plan (L1 à L4) ne touche que `cmd/server`, `tokencache`, `media`, `ops`,
    `index.html` et `ci.yml` : elle peut courir en parallèle de la publication ;
  - **L5** (auth) et **L7** (extraction de `main.go`) attendent la fusion du lot 3 de la publication ;
  - `.air.toml` (L2) : rebaser sur la version de la publication si elle passe avant.
- **Branche** : `feat/desktop-p0` depuis `release/v8.1.0` (ou la release suivante si v8.1.0 est publiée
  entre-temps), un commit par lot, PR squash vers la release. P1 : `feat/desktop-p1`. Noms à confirmer.
- **CLAUDE.md** : la règle « Go/TS uniquement » est amendée par l'ADR 0046 (Rust limité à
  `apps/desktop/src-tauri`, quelques centaines de lignes, aucune logique métier) dans le lot L16.

## 3. Vérification sur pièces des deux documents (2026-11-01, HEAD `8f0b512c58`, deux passes)

| Constat | État | Écart ou précision |
|---|---|---|
| C1 signaux d'arrêt `boot_gate.go:197`, air sans `send_interrupt` Windows | tenu | `.air.toml:48-59` |
| C2 `migrations ignorées` en Debug ; `title_schema_version` jamais relu | tenu, **mais** | `main.go:469` ; le cas « base verrouillée » est RÉEL : `main.go:563-583` (12 × 500 ms après un arrêt forcé sous Windows), `platform/duckdb/db_recovery.go:66` `IsFileLockError` (antivirus, sauvegarde), sentinelle `ErrBaseTenueEnEcriture` ligne 79 ; `runMigrations` ouvre en RW SANS cette reprise (`main.go:1392`) |
| C3 store câblé en mode `xbox` seul (`server_apiv1.go:357`) ; `ErrSessionNotAuthenticated` toléré ; `persistRTATokens` | tenu | **non reproduit par exécution** (item 0.4) ; `multiUserTokens` est construit DANS la branche `xbox` (357-360) |
| C4 repli `time.Now().Add(fallbackTokenTTL)` | tenu, chemin déplacé | `internal/platform/auth/tokencache/player_token_cache.go:83` ; **Windows aussi concerné** : `runtime·nanotime1` lit `_INTERRUPT_TIME` (`sys_windows_amd64.s:200`, `time_windows.h:13`), horloge qui ne compte pas la veille |
| C5 archive 104 Go | **périmé** | 36 Go au 2026-11-01 (`du -sh data/cache/film_chunks`) ; chunks zlib tenu |
| C6 `main.go` 2 017 L, `main()` 1 210 L | tenu | hors `main()` il reste ~800 L (`runMigrations` 1354…) : « ≤ 500 L » exige de déplacer aussi ces fonctions |
| C7 `.DemoMode` 89, `.AuthMode` 55, `LEVELUP_ENV` 13 | tenu | `buildFeatureFlags` expose déjà `demo_mode`, `discord_configured`, `tailscale_enabled` |
| C8 `LEVELUP_ROOT` / `LEVELUP_DATA` morts (`Dockerfile:246-247`, compose `15,17,97,98`) | tenu | **`LEVELUP_REPO_ROOT` existe déjà** (`config.go:208,217`, `domain/title/repo_root.go:34`, `filmproc.EnvRepoRoot`) ; `RepoRoot` : 193 occurrences hors tests dans 90 fichiers d'`internal/`, 18 dans `cmd/server` ; `db_profiles.json` et `app_settings.json` ÉCRITS à la racine (`registry.go:1114,1119`) ; journaux par défaut `<repoRoot>/logs` (`observability/logging/config.go:83-86`, surcharge `LEVELUP_LOGS_DIR`) |
| C9 Windows : « aucun test » | **imprécis** | Windows joue déjà `go test` sans CGO sur `domain`, `analysis`, `games/weapons`, `contracttest` (`ci.yml:591-596`) et les briques `film`, `gamefiles` ; manque : CGO, `cmd/server`, fichiers Windows. `release.yml:57-69` compile déjà le serveur avec `CGO_ENABLED=1` sur `windows-latest` SANS action mingw : gcc est sur le runner |
| C9 fichiers propres à Windows | précisé | 8 `*_windows.go` dans 3 modules (`libs/film/filmproc`, `libs/film/internal/filelock`, `libs/gamefiles/himodule`, `internal/platform/diskfree`) + `cmd/levelup/cmd_compact_passes_remplacement_win.go` (suffixe `_win`) |
| C10 `release.yml:164`, aucun `go:embed` du site | tenu | — |
| C11 `static/sounds` 84 Mo, 184 WAV | tenu | l'extension `.wav` est aussi liée dans `cmd/weapon-sounds` (71 fichiers + goldens), `games/mappings/loader_vehicle_weapons.go:75`, `replaySoundAssets.guard.test.ts`, `exportSoundFamilies.ts`, `vehicleEngineSound.ts` (boucles) |
| C12 Google Fonts `index.html:23-25` ; 4 littéraux ffmpeg | tenu | — |
| Étude §3.3 bannière sur `srv.Addr` | tenu | `boot_gate.go:190,246` ; port pris = `os.Exit(1)` (190-195) |
| Étude §3.7 « `filmproc/solo.go` » | tenu, déplacé par D2 | `libs/film/filmproc/solo.go:99` `AcquireSolo`, verrou noyau `libs/film/internal/filelock`, fichier `film_decode.lock` sous `CacheRootDir()` |
| Étude §4.1 pas de tick initial | tenu | `scheduler/auto_sync_run.go:26-30` |
| Étude §3.5 secret public, CORS | tenu | `config.go:28,236,237` ; `authz.Enforced` (`authz.go:34`) ; `authz.InstanceLocked` (`authz.go:159`, lecture unique imposée par CLAUDE.md) |
| Enfants de cuisson | précisé | `internal/replaychild/replaychild.go:49` (`-film-child`), lancés par `exec.CommandContext` (`libs/film/filmproc/runner.go:209`) : l'annulation du contexte tue l'enfant ; à vérifier que le contexte de `replayartifacts/buildone.go:157` descend de celui annulé en `phaseScheduler` (`shutdown.go:112`). `SysProcAttr` déjà posé pour la priorité (`priority_windows.go:27-30`) ; aucun Job object |
| État de démarrage servi par `boot_gate` | partiel | 4 `bootStep` en 503 `server_starting` `Retryable: true` (`boot_gate.go:30-65,144-165`) ; aucun état d'échec ; le web réinterroge chaque seconde jusqu'à 120 s (`apps/web/src/app/serverStartup.ts:15-26`) puis « API injoignable » ; `serverStartup.test.ts:70` lit `boot_gate.go` |
| Marqueurs d'arrêt dans le journal | **inexistants** | « arrêt propre » n'est qu'un commentaire ; « replaying WAL » n'apparaît que dans une erreur de corruption (`pool_shared_social_recovery.go:54`) ; lignes réelles : `shutdown_total_duration_ms` (`shutdown.go:59`), « CHECKPOINT final … terminé » (`shutdown.go:192`) |
| `/bootstrap` porte `CapabilityMap` | tenu | `bootstrap_service.go:395`, type `domain/bootstrap.go:22`, 9 champs dont `CanManageInstance` constant |
| `PathResolver` à une racine | tenu | `registry.go:530-546` ; `BackupDir(slug, gamertag)` existe (ligne 810, exports Parquet) : ne pas réutiliser ce nom |
| `LoopbackOnly` | tenu | `middleware/loopback.go:54`, décide sur le pair TCP ; reconnu par `bare_routes_ratchet_test.go:50` |
| Playwright | précisé | 30 scénarios `apps/web/e2e/*.spec.ts` |
| Tâches de fond au démarrage | ajouté | `background_tasks.go:84-121` `demoPolicies` (garde-rail : chaque tâche déclarée) ; `taskAppRelease` lancé sous `context.Background()` (`main.go:1291`), donc hors du contexte d'arrêt |

## 4. Ce que les deux documents n'avaient pas vu (ajouts du plan)

- **A1 — Contrat coquille ↔ serveur testable des deux côtés.** Le contrat est écrit (§6) ; chaque côté
  se teste contre un faux de l'autre : un **faux sidecar** en Go (`apps/desktop/testdata/fake-sidecar`,
  son propre `go.mod`, `GOWORK=off`) et un **faux lanceur** en Go dans les tests de `cmd/server`.
- **A2 — Enfants de cuisson et mort de la coquille.** Pas de Job object : il tuerait le sidecar à la
  mort de la coquille, c'est-à-dire exactement le cas où la fermeture du tube stdin (fermé par l'OS à la
  mort du processus) doit déclencher l'arrêt propre. Les enfants sont tués par l'annulation du contexte
  (`exec.CommandContext`) ; le verrou noyau `film_decode.lock` est rendu par l'OS. Risque résiduel :
  un enfant orphelin après un crash DUR du serveur (durée bornée par le film en cours).
- **A3 — Mise à jour et fichiers tenus.** L'installeur ne peut pas remplacer un `.exe` en cours
  d'exécution : l'updater ne se lance qu'après `stopped` ; la copie des bases (C2) précède toute montée.
- **A4 — Chaînes de la coquille et i18n.** Aucun littéral de langue dans le Rust : libellés lus sur le
  serveur (`GET /api/v1/desktop/strings`, catalogue `internal/i18n/messages/desktop.toml`) ; pour les
  états où le serveur ne répond pas (démarrage, arrêt, mort), un **catalogue de repli généré à la
  compilation** depuis le même TOML (`build.rs`), dans toutes les langues de `config/locales.toml`.
  L'écran de démarrage embarqué n'affiche que le logo et une animation jusqu'à `ready`.
- **A5 — WebView2 runtime.** Pas garanti sous Windows 10 : `webviewInstallMode` Evergreen (bootstrapper).
- **A6 — `/bootstrap` a déjà `CapabilityMap` et `FeatureFlags`.** C7 ÉTEND `CapabilityMap` ; `FeatureFlags`
  garde `demo_mode` pour la compatibilité jusqu'à migration du front (règle 6 : pas de doublon neuf).
- **A7 — Pare-feu et exposition.** En mode `desktop`, l'hôte d'écoute est forcé à la boucle locale par
  le résolveur, quelle que soit la config.
- **A8 — Tests existants réutilisables.** `cmd/server` : 17 fichiers de test ; 30 scénarios Playwright ;
  `release.yml` compile déjà le serveur Windows avec CGO.
- **A9 — Version unique.** Le tag git alimente `-ldflags -X main.version`, `Cargo.toml`, `tauri.conf.json` ;
  une étape « version check » de `release.yml` refuse tout écart entre les quatre.
- **A10 — Tâche planifiée « rafraîchir seulement ».** Un mode du binaire serveur (`-mode refresh-only`),
  pas un second binaire ; prend le verrou d'instance (A16), n'ouvre aucune base, sort après rotation.
- **A11 — DuckDB hors ligne.** Aucune extension chargée (0 `INSTALL` / `LOAD`).
- **A12 — Secret de session.** En mode `desktop`, généré au premier lancement, écrit `0600` dans
  `<DataRoot>/auth/session_secret`, jamais le défaut public.
- **A13 — Outillage du poste.** `cargo` / `rustc` 1.97.1, Node 24.13.1 ; pas d'`apps/desktop`.
- **A14 — Journaux.** Racine données (`LEVELUP_LOGS_DIR` existe déjà) ; la coquille relaie stderr du
  sidecar vers `<DataRoot>/logs/desktop.log` et VIDE stdout en continu (un tube plein bloque le serveur).
- **A15 — Port stable.** L'origine web inclut le port : `localStorage`, `sessionStorage` et IndexedDB
  (297 occurrences dans 85 fichiers de `apps/web/src` : filtres, préférences du rejeu, langue de
  démarrage, notifications) seraient VIDÉS à chaque lancement avec `127.0.0.1:0`. Et le cookie, lui,
  n'est pas cloisonné par port : il partirait vers tout service local sur `127.0.0.1`. Règle : port
  mémorisé dans `<DataRoot>/desktop/port`, repli sur un port libre avec avertissement journalisé ;
  le mode `desktop` lie le cookie à un nom de cookie propre au port.
- **A16 — Verrou d'instance côté Go.** Le plugin single-instance de Tauri ne protège ni de la tâche
  `refresh-only`, ni d'un `go run`. Verrou noyau `<DataRoot>/levelup.lock` (patron `filelock`, recopié
  sous `internal/platform/instancelock` : `libs/film/internal/filelock` n'est pas importable).
- **A17 — Tâches de fond par profil.** `demoPolicies` devient une politique par profil de déploiement
  (cron Waypoint, Spartan, purge des rejeux, surveillance disque, sauvegarde) ; `taskAppRelease` rejoint
  le contexte d'arrêt.
- **A18 — Purge des rejeux × « ne pas archiver ».** La purge (`scheduler/replay_purge_cron.go`) suppose
  une recuisson depuis le cache film : si l'archive est désactivée (D-5), la rétention est forcée
  illimitée (combinaison refusée dans les réglages).
- **A19 — Licence et contenu de jeu.** `NOTICE` §2(b) (attribution atteignable depuis chaque écran), §4
  (images et sons du jeu : Microsoft), §6 (dépendances : ajouter `Cargo.lock`) ; fichier de mentions
  tierces dans l'installeur. **Règles d'usage du contenu de jeu Microsoft lues le 2026-11-01**
  (`xbox.com/en-US/developers/rules`, « Last Updated: January 2015 ») : (1) le lien de dons est
  explicitement permis (« You may use the Item on a page where you ask for optional donation
  requests ») ; une app gratuite doit rester gratuite et sans publicité (le cas) ; (2) mention
  OBLIGATOIRE, texte imposé : « Halo Infinite © Microsoft Corporation. LevelUp was created under
  Microsoft's "Game Content Usage Rules" using assets from Halo Infinite, and it is not endorsed by or
  affiliated with Microsoft. » + lien vers les règles, visible de quiconque voit l'Item (pied de page web
  ET installeur ET page de téléchargement) : la mention actuelle du pied de page n'a pas ce texte exact
  → item L17.1 ; (3) **sons** : les règles préviennent que les effets audio sont souvent sous licence
  tierce et ne sont pas couverts (seule la musique de Halo 3 est citée) : risque juridique propre aux
  184 sons de `static/sounds`, à trancher par l'utilisateur (retirer, ou assumer comme aujourd'hui sur
  le site) ; (4) pas de logo du jeu dans le logo de l'app ; (5) clause de licence accordée à Microsoft
  sur l'Item, à lire face à l'AGPL (note dans l'ADR 0045).
- **A20 — Application Azure unique** (`platform/auth/azure_credentials.go:30`, client public) : toutes
  les installations partagent le même identifiant ; dépendance externe à consigner dans l'ADR 0046.
- **A21 — Clé de signature de l'updater.** Perdue = plus aucune mise à jour automatique possible :
  procédure de sauvegarde et de rotation dans l'ADR 0046 ; clé publique allowlistée dans `.gitleaks.toml`.
- **A22 — Désinstallation.** Le désinstalleur ne supprime JAMAIS la racine de données (36 à 100 Go) ;
  l'identifiant Tauri (`info.lvelup.desktop`) est figé dès P1 (il nomme aussi le dossier WebView2).
- **A23 — Hygiène du dépôt pour Rust.** `.gitignore` (`target/`, `gen/`), `Cargo.lock` versionné,
  `dependabot.yml` (`cargo`), `cargo audit` en CI, cibles `Makefile`, `docs/COMMANDS.md` EN + FR,
  filtre `desktop` dans `ci.yml` (`ci_paths_filter_test.go`), actions épinglées par SHA dans tout
  workflow lisant `secrets.` (`archlint/ci_workflows_security_test.go`).
- **A24 — Veille sous Windows.** Prouvé par le toolchain : l'horloge monotone de Go sous Windows ne
  compte pas la veille (§3, C4) ; la détection « écart heure murale / monotone » (L15) vaut donc aussi
  sous Windows.

## 5. Décisions (confirmées par l'utilisateur le 2026-11-01 via la page de décisions ; 14 suivent la reco, D-8, D-9 et D-10 s'en écartent)

| # | Décision | Choix | Pourquoi |
|---|---|---|---|
| D-1 | Tauri ou Wails (étude q1) | **Tauri 2** | updater signé, NSIS, plugins mûrs |
| D-2 | Plateforme v1 (q2) | **Windows seul** | cible joueurs PC ; macOS = P3 |
| D-3 | Autonome ou client (q3) | **Autonome**, option « se connecter à un serveur » en P2 | valeur locale (rejeu, fin de match) |
| D-4 | C3-a : mise en commun en mode `none` | **Non** (sans décision = refus, ADR 0038) ; le profil `desktop` l'accorde d'office au seul utilisateur | cohérence ADR 0038 |
| D-5 | C5-a : archive des films | **Archiver par défaut** ; choix « ne pas archiver » dans l'assistant de premier lancement (`features/setup`), pas dans NSIS ; occupation affichée ; emplacement réglable | donnée irremplaçable ; choix informé |
| D-6 | C11-a : format des sons | **Mesurer d'abord** (L9 : AAC, Opus, FLAC sur 10 sons dont 3 boucles moteur), décision à la clôture de L9 | l'AAC ajoute une amorce qui peut casser les boucles |
| D-7 | C9 : budget Windows | voie CGO sur `release/**` et PR vers `main` seulement, `timeout-minutes: 15` | runners Windows plus chers |
| D-8 | Chiffrement des jetons (q5) | **OUI dès la v1 (décision utilisateur 2026-11-01, contre la reco)** : coffre Windows (DPAPI, portée utilisateur) derrière un port Go `TokenCipher` dans `platform/auth` ; ADR 0023 amendée ; lot L13b | choix de l'utilisateur ; le fichier reste illisible depuis un autre compte Windows ou une copie du disque |
| D-9 | ffmpeg (q6) | **LIVRÉ dans l'installeur (décision utilisateur 2026-11-01, contre la reco)** : build LGPL officiel, ~80 Mo, mentions et sources LGPL jointes ; résolveur C12 cherche d'abord le dossier de l'exécutable | prêt à l'emploi ; coût : 80 Mo par mise à jour |
| D-10 | Signature de code (q6) | **JAMAIS (décision utilisateur 2026-11-01)** : avertissement SmartScreen à chaque installation, documenté sur la page de téléchargement ; l'updater Tauri reste signé par sa propre clé (indépendante) | pas de budget ; réversible si la diffusion le justifie |
| D-11 | C10 binaire autonome | **Non retenu** (ressources Tauri) | reste au backlog |
| D-12 | ADR | **0044** profil de déploiement (C7), **amendement 0008** (C8), **0045** contrat coquille/sidecar, mode `desktop`, clé d'updater, Azure (0043 est réservé par la publication des briques) | escalade : contrat d'API, emplacement des données |
| D-13 | Port | **Stable, mémorisé** (A15) ; jamais `:0` sans repli journalisé | stockage navigateur, cookie |
| D-14 | Installation NSIS | **Par utilisateur** (`%LOCALAPPDATA%\Programs\LevelUp`), sans élévation ; données sous `%LOCALAPPDATA%\LevelUp` ; désinstallation conserve les données | updater sans UAC ; ressources en lecture seule par convention, pas par ACL |
| D-15 | Réglages (drapeaux ou variables) | **Variables d'environnement** pour les racines (`LEVELUP_DATA_DIR`, `LEVELUP_RESOURCES_DIR`, `LEVELUP_LOGS_DIR`), **drapeaux** pour le mode (`-mode desktop`, `-mode refresh-only`, `-arret-sur-stdin`) ; un seul fichier `cmd/server/flags.go`, créé par L2 et étendu par les lots suivants EN SÉQUENCE | aujourd'hui aucun parseur : `os.Args` testé à la main (`main.go:125,130`) |
| D-16 | Un seul concept de profil | `Deployment` (`hosted`, `desktop`, `demo`, `dev`) résolu par `internal/config` ; `AuthMode=desktop` et l'hôte loopback sont DÉRIVÉS par le résolveur, jamais posés à la main | trois formulations du même concept dans la première version du plan |
| D-17 | Rechargement air sous Windows | pré-commande d'air qui appelle le point d'arrêt (serveur arrêté pendant la reconstruction, comme aujourd'hui mais proprement) | le relais stdin ne survit pas au `TASKKILL /T` d'air |

## 6. Contrat coquille ↔ serveur (référence des lots L12, L13, L16)

1. **Lancement** : répertoire courant = racine ressources ; environnement `LEVELUP_DATA_DIR`,
   `LEVELUP_RESOURCES_DIR`, `LEVELUP_LOGS_DIR`, `LEVELUP_DESKTOP_SECRET=<aléa>` (jamais en argument) ;
   arguments `-mode desktop -arret-sur-stdin`. Le port vient de `<DataRoot>/desktop/port` (D-13).
2. **Événements** sur stdout, une ligne JSON chacun, écrits par un écrivain unique sous verrou (tout
   journal console va sur stderr) : `{"event":"listening","port":N}` au port lié (avant les
   migrations) ; `{"event":"ready","port":N}` quand le routeur est ouvert (`bootServer.openRouter`) ;
   `{"event":"failed","state":"migration_failed"|"schema_newer"|"locked"}` ; `{"event":"stopping"}` ;
   `{"event":"stopped","clean":true|false}` (`clean=false` : budget dépassé ou vidage persist en erreur).
3. **Première page** : la coquille montre son écran embarqué (logo) jusqu'à `ready`, puis charge
   `http://127.0.0.1:N/desktop/enter?secret=<aléa>` ; le serveur pose le cookie et redirige vers `/`.
   Sans cookie, toute route hors `/desktop/enter` répond 401 (`bare_routes_ratchet_test.go` : allowlist
   datée pour `/desktop/enter`) ; `/api/v1/desktop/strings` exige le cookie. Contrôle strict de `Host`.
   Sur `failed`, la coquille charge la page de démarrage du serveur (état servi en 503 non réessayable).
4. **Arrêt** : fermeture de stdin (chemin principal, aussi déclenché par l'OS si la coquille meurt) OU
   `POST /api/v1/internal/shutdown` (secret en en-tête, `LoopbackOnly`, comparaison à temps constant).
   Un seul passage (`sync.Once`) vers le chemin du signal, même budget de 40 s. La coquille attend
   `stopped` ou budget + 5 s, puis seulement tue.
5. **Observable d'un arrêt propre** : ligne de journal `arrêt: terminé` avec `clean=true` ET aucun
   fichier `*.wal` sous `<DataRoot>`. C'est la définition unique utilisée par L2, L6, L12, L17.
6. **Téléchargements** : gestionnaire de téléchargement de la WebView côté Rust (`on_download`), boîte
   « Enregistrer sous » native ; aucun changement dans `apps/web`.

## 7. Stratégie de test (développeur seul)

| Niveau | Ce qui est testé | Outil | Qui lance |
|---|---|---|---|
| Unitaire Go | chaque chantier, résolveur de déploiement, `PathResolver` à deux racines, mode `desktop`, verrou d'instance | `go test`, `httptest`, DuckDB `:memory:` | CI Linux + Windows CGO |
| Contrat côté serveur | EOF stdin → arrêt ; événements stdout ; `/desktop/enter` ; refus hors boucle locale | test Go lançant `cmd/server` compilé avec un faux lanceur | CI Windows CGO |
| Contrat côté coquille | `listening` / `ready` / `failed`, délais, arrêt, kill après budget, secret absent des arguments, stdout vidé | `cargo test` contre le faux sidecar | CI Windows |
| Bout en bout CI | installation NSIS silencieuse, lancement de l'exécutable, port lu dans le fichier, page servie, `WM_CLOSE`, aucun `*.wal`, relancement | script PowerShell, job `desktop-e2e` sur `release/**` (pas de `tauri-driver` : dépend d'un `msedgedriver` aligné sur le runtime) | CI Windows |
| Web | état `migration_failed`, cookie `/desktop/enter` (un scénario Playwright) | Vitest, Playwright | CI |
| Visuel | démarrage, barre des tâches, fermeture, mise à jour, désinstallation sur machine vierge | **témoins nommés par l'utilisateur** | utilisateur |

Règle : un lot sans test à son niveau n'est pas clos. La voie C9 est livrée AVANT tout lot à
comportement Windows (L2, L12 à L17).

## 8. Témoins avant/après (lot 0, scratchpad, joints à la PR)

- T1 : `go test ./... -count=1` du module go-api, liste des paquets `ok` (comparée après chaque vague).
- T2 : démarrage réel sur une copie de données ; journal filtré `grep -E "^time=.*level=INFO"
  logs/server.log | sed 's/^time=[^ ]* //' | sed -E 's/duration[_a-z]*=[0-9.a-z]+//g' > t2_avant.txt` ;
  après L7 : même commande, `diff t2_avant.txt t2_apres.txt` vide.
- T3 : `curl -s http://127.0.0.1:8000/api/v1/bootstrap` en `none`, `xbox`, démo, figés ; après L10 :
  identiques modulo le champ `deployment` (`jq 'del(.capabilities.deployment)'`).
- T4 : liste exhaustive des chemins rendus par `PathResolver` pour `halo_infinite` et un joueur (test
  Go jetable qui appelle chaque méthode exportée, sortie triée) ; après L11 : identique avec les deux
  racines égales à l'actuelle.
- T5 : `du -sh data/cache/film_chunks data static` ; `od -A x -t x1 -N 2 <un chunk_00.bin>`.
- T6 : `ls <DataRoot>/**/*.wal` après un arrêt par signal sur le poste (attendu : vide).

État au 2026-11-01 (lot 0, scratchpad de la session, fichiers `t1_avant.txt`, `t2_avant_{general,migration}.txt`
+ `t2_recette.txt`, `t3_{xbox,none}.json` (binaire local `8f0b512c58`) et `t3_demo_prod.json`
(`demo.lvelup.info`, v8.0.7 : témoin de forme seulement), `t4_avant.txt` (67 méthodes, 2 non appelées :
signatures non-chaîne), `t5` (36 Go de films, 63 Go de `data/`, 108 Mo de `static/`, chunk zlib `78 5e`),
`t6_avant.txt`) :
- T2 se lit dans les journaux par catégorie (`logs/general.log`, `logs/migration.log`, JSON, console au niveau
  WARN seulement) : la recette retire `time`, `source` (les noms de fonction changent avec C6) et les durées.
- T6 « avant » = l'état APRÈS un arrêt forcé (ce que fait air aujourd'hui, le signal propre n'existant pas
  sous Windows) : WAL laissés sur `metadata` (105 Ko), `monitoring` (4,8 Mo, présent avant et après les deux
  lancements : jamais résorbé), `shared_social`, 4 bases joueur. C'est le défaut C1 mesuré ; l'« après » de L2
  doit donner une liste vide hors `data/backups/`.

## 9. Lots

Légende modèle : **S** = Sonnet (implémentation), **H** = Haiku (mécanique, contrôlé sur pièces),
**O** = Opus (revue adversariale), **P** = pilote, **U** = utilisateur. « ∥ » = en parallèle.

### Vue d'ensemble

| Vague | Lots | Parallélisme | Dépend de |
|---|---|---|---|
| 0 | L0 relevé, témoins, décisions (P + U) | — | — |
| 1 | L1 C2 migrations (S) → L2 C1 arrêt (S) [séquence : mêmes fichiers] ∥ L3 C4 + C12 (H) ∥ L4 C9a voie Windows CGO (S) | 3 agents | L0 |
| 1b | L5 C3 (S) ∥ L6 C9b bout en bout arrêt (S) | 2 agents | L2, L4 ; publication lot 3 (L5) |
| 2 | L7 C6 extraction (S, trois sous-lots, **à risque**) ∥ L8 C11 sons (S, après L9) ∥ L9 mesures zstd et sons (H) ∥ ADR 0044 / 0008 / 0045 (P) | 2 agents + P | vague 1 fusionnée ; publication lot 3 |
| 2r | revue adversariale L1 + L7 (O) | 1 agent | L7 |
| 3 | L10 C7 profil (S) ∥ L11 C8 racines (S, trois sous-lots, **à risque**) | 2 agents | L7, ADR |
| 3r | revue adversariale L11 (O) | 1 agent | L11 |
| 4 | L12 port, événements, verrou d'instance (S) → L13 mode `desktop` (S) → L13b coffre Windows (S) ; ∥ L14 C5 archive (S) ∥ L15 réveil (S) | 3 agents | L10, L11 ; L13 après L12 ; L13b après L13 ; L15 après L5 |
| 5 (P1) | L16 coquille (S, **à risque**) → L17 CI, NSIS, updater (S) ; revue L16 (O) ; gate visuel (U) | 1 puis 1 | vague 4 |
| 6 (P2) | lots à écrire à la clôture de P1 (§9.18) | — | P1 |

Charge estimée : vagues 0 à 4 = 14 agents Sonnet/Haiku bornés (un lot = un brief, gates au paquet, pas
de `make gate-push` par agent) + 2 revues Opus ; P1 = 2 Sonnet + 1 Opus. Le pilote joue `make gate-push`
une fois par vague avant la PR. Les items « run vert », « mutation » et « push » sont au pilote.

### 9.0 L0 — Relevé, témoins, décisions (P ; 0.4 et 0.5 avec U)

- [x] 0.1 `feat/desktop-p0` créée depuis `release/v8.1.0` (`8f0b512c58`), worktree
      `../LevelUp-wt-desktop`, GOCACHE `go-build-desktop` (2026-11-01).
- [x] 0.2 Témoins T1 à T6 capturés le 2026-11-01 (état détaillé en §8). T1 : 194 paquets `ok`, 1 échec
      `internal/mapcatalog` (`TestAddOverlayEntryConcurrentDossierAbsentNePerdRien`, verrou de fichier
      sous charge) rejoué seul 3 fois vert : test non déterministe en run complet (§10), pas un défaut
      du commit.
- [x] 0.3 Vérifié (2026-11-01) : la cuisson post-sync passe par `sync/replayartifacts/artifacts.go:136`
      (`BuildOne: SpawnBuildOne`) avec le contexte du sync, qui dérive de `schedulerCtx`, annulé par
      `cancelScheduler` avant `phaseScheduler` (`shutdown.go:55,112,124-128`) ; l'enfant est lancé par
      `exec.CommandContext` (`libs/film/filmproc/runner.go:209`) : il est TUÉ à l'annulation (arrêt dur,
      sans conséquence : il rend des octets sans rien écrire, `buildone.go:150-152`). La voie admin
      (`wire/registry_replay_build.go:71`) suit le contexte de la requête HTTP. Rien à changer en L12.3
      hors un test qui fige « enfant absent après arrêt ».
- [x] 0.4 **C3 confirmé sur pièces le 2026-11-01** (chaîne fermée, pas d'exécution : une connexion
      Microsoft réelle est nécessaire pour l'exécuter, le pilote ne la détient pas) : en mode `none`,
      `server_apiv1.go:392` n'appelle pas `WithLinkStrategy`, donc `completeLogin`
      (`auth_login_complete.go:22`) reçoit une stratégie nulle et ne pose que la session ; les seuls
      écrivains du magasin après une connexion sont `xbox_auth_service.go:311` (stratégie SSO, mode
      `xbox` seul) et `auth_xbox_oauth.go:338` (flux OAuth, monté en mode `xbox` avec URI de
      redirection seulement) ; `refresh_user_xsts.go:74` ne fait que rafraîchir une entrée existante.
      L5 lancé tel quel. L'ordre « profil créé avant xuid connu » reste à relever dans le code de
      `features/setup` au brief de L5 (`queries.ts:60` crée le profil par `/setup/players`, le
      device-flow est à `queries.ts:25,44`).
- [x] 0.5 Décisions D-1 à D-17 confirmées par l'utilisateur le 2026-11-01 (page de décisions) ; lien de
      dons permis par les règles Microsoft (A19) ; **sons du jeu : GARDÉS** (décision utilisateur
      2026-11-01, « ok on garde »), même posture que le site, avec une ligne dans les mentions tierces
      (« effets sonores du jeu, propriété de Microsoft et de ses concédants, retirés sur simple
      demande ») → L17.1 ; branche `feat/desktop-p0` confirmée.
- Gate : témoins présents ; entrée `thought_log`.

### 9.1 L1 — C2 Migrations : verrou distingué, échec fatal, garde de version, copie (S)

Fichiers : `cmd/server/main.go:460-475,563-583,1354-1400`, `cmd/server/boot_gate.go`,
`internal/migration/registry.go`, nouveaux `internal/migration/version_guard.go`,
`internal/migration/backup_before_upgrade.go`, `internal/domain/title/registry.go` (une méthode),
`apps/web/src/app/serverStartup.ts` et ses manifests i18n.

- [ ] 1.1 La reprise bornée du verrou (`main.go:572`, 12 × 500 ms) est DÉPLACÉE avant `runMigrations`
      (une seule reprise, les deux appelants l'utilisent) ; l'erreur est classée par
      `duckdb.IsFileLockError` / `errors.Is(err, duckdb.ErrBaseTenueEnEcriture)`.
- [ ] 1.2 Après reprise épuisée (verrou) ou sur toute autre erreur de `runMigrations` :
      `slog.ErrorContext(ctx, "migration: échec, démarrage refusé", "err", err, "state", s)` ; le serveur
      RESTE debout sur la porte de démarrage avec un état d'échec (`locked`, `migration_failed`,
      `schema_newer`) servi en 503 **non réessayable** (`Retryable: false`, champ nouveau de `bootStep`),
      événement `failed` (§6.2) ; il sort avec un code non nul au signal, à l'EOF stdin ou au point
      d'arrêt. Supprimer la ligne `migrations ignorées` (`main.go:469`).
- [ ] 1.3 Garde de version : lecture de `title_schema_version` SANS ouverture en écriture (connexion
      lecture seule, fermée aussitôt) avant toute migration ; version > binaire = `ErrSchemaNewerThanBinary`
      → état `schema_newer`.
- [ ] 1.4 Copie avant montée : si des étapes sont en attente, copier `<base>.duckdb` + `<base>.duckdb.wal`
      (base FERMÉE, avant l'ouverture RW) vers `PathResolver.MigrationBackupsDir()` (nouveau, distinct de
      `BackupDir`), nom `<base>.v<version>.<date>.duckdb[.wal]`, rétention 2, refus si l'espace libre
      (`platform/diskfree`) est inférieur à la taille à copier (état `migration_failed`, cause journalisée).
- [ ] 1.5 Web : `serverStartup.ts` distingue un 503 non réessayable (arrêt de l'interrogation, écran
      explicite par état, libellés fr + en dans le manifest), et prolonge l'attente de 120 s tant que le
      serveur répond `migrating` (progression) ; `serverStartup.test.ts` mis à jour.
- [ ] 1.6 Tests Go : étape en échec → état `migration_failed`, journal Error, code de sortie non nul ;
      verrou tenu par un second processus de test → reprise puis `locked` ; base N+1 → `schema_newer` ;
      copie présente / absente / espace insuffisant.
- Gate : `go test ./internal/migration/... ./cmd/server/... ./internal/domain/title/...` ;
  `go test -tags=integration -p 1 ./internal/migration/...` ; `grep -n "migrations ignorées"
  apps/go-api/cmd/server/*.go` = 0 ; `cd apps/web && npm run test -- serverStartup && npm run typecheck`.
- Risque : démarrage prod → revue adversariale groupée avec L7 (vague 2r).

### 9.2 L2 — C1 Arrêt propre sous Windows (S ; après L1, mêmes fichiers)

Fichiers : `cmd/server/flags.go` (nouveau, D-15), `cmd/server/boot_gate.go`, `cmd/server/shutdown.go`,
`cmd/server/shutdown_stdin.go` (nouveau), `internal/api/handlers/internal_shutdown.go` (nouveau),
`internal/config/config.go` (`DesktopSecret` lu depuis l'environnement, jamais journalisé),
`apps/go-api/.air.toml`, `bare_routes_ratchet_test.go` (allowlist datée).

- [ ] 2.1 `flags.go` : parseur unique (`flag.NewFlagSet`), drapeaux `-arret-sur-stdin`, `-mode`
      (valeurs admises : vide, `desktop`, `refresh-only` ; les deux derniers refusés jusqu'à L12/L15
      avec une erreur explicite), compatibilité des tests `os.Args` existants (`-health-check`,
      `-film-child`).
- [ ] 2.2 `-arret-sur-stdin` : goroutine lisant stdin jusqu'à EOF → `requestShutdown(reason)`, fonction
      unique sous `sync.Once` que le signal et le point d'arrêt appellent aussi ; ligne de journal
      `arrêt: terminé` avec `clean` (§6.5) écrite en fin de `shutdown.go`.
- [ ] 2.3 `POST /api/v1/internal/shutdown` : `LoopbackOnly`, en-tête `X-LevelUp-Shutdown-Secret` comparé
      en temps constant à `cfg.DesktopSecret` ; 404 si non configuré, 403 si faux, 202 sinon ; injection
      de `requestShutdown` par le patron de `cfg.TitleReadyNotifier` (`main.go:1280`) ; route allowlistée
      dans le ratchet des routes nues avec date.
- [ ] 2.4 Air (D-17) : `pre_cmd` qui appelle le point d'arrêt avec le secret de dev (`.env.local`),
      tolère l'absence de serveur ; commentaire `.air.toml:48-59` réécrit.
- [ ] 2.5 Tests : EOF stdin → `requestShutdown` appelé une fois ; double déclenchement → une seule
      séquence ; handler : hors boucle locale 403, sans secret 404, mauvais secret 403, bon secret 202 ;
      `shutdown_budget_test.go` vert ; ratchet des routes nues vert.
- Gate : `go test ./cmd/server/... ./internal/api/... ./internal/config/...` ; **(P)** rechargement air
  sur le poste : ligne `arrêt: terminé clean=true`, T6 vide.

### 9.3 L3 — C4 horloge monotone + C12 police et résolveur ffmpeg (H)

- [ ] 3.1 `tokencache/player_token_cache.go:83` : `time.Now().Round(0).Add(fallbackTokenTTL)` ; test
      `exp.Round(0) == exp` ; commentaire citant la preuve (A24).
- [ ] 3.2 Recensement `grep -rn "time.Now().Add(" --include=*.go apps/go-api/internal | grep -v _test`
      avec le critère écrit : garder une ligne si la valeur est CONSERVÉE dans une structure (champ,
      map, cache) ET comparée plus tard à `time.Now()` ; jeter si consommée dans la même fonction
      (délai, contexte, backoff). Lignes gardées → même correction ; liste complète (gardées et jetées,
      avec la raison) dans le compte rendu, contrôlée par le pilote.
- [ ] 3.3 Roboto auto-hébergée : fichiers woff2 (400/500/700/900, sous-ensembles latin + latin-ext,
      licence Apache 2.0 jointe) FOURNIS PAR LE PILOTE dans le worktree sous `apps/web/public/fonts/`
      (pas de dépendance npm) ; `@font-face` dans le CSS global ; suppression de `index.html:23-25` ;
      garde-rail Vitest : `index.html` et `dist/index.html` sans `fonts.googleapis`.
- [ ] 3.4 Résolveur unique `internal/media/binpath.go` (`LEVELUP_FFMPEG_DIR`, puis dossier de
      l'exécutable, puis PATH) utilisé par `ffmpeg_slot.go:125`, `ffprobe.go:20`, `ops/healthcheck.go:93`,
      `ops/media_tooling.go:79-80` ; garde-rail archlint interdisant `"ffmpeg"` / `"ffprobe"` hors du
      résolveur dans `internal/` ET `cmd/` ; test de l'ordre de recherche.
- Gate : `go test ./internal/platform/auth/tokencache/... ./internal/media/... ./internal/ops/...
  ./internal/archlint/...` ; `cd apps/web && npm run build && grep -c fonts.g dist/index.html` = 0 ;
  `npm run test -- fonts`.

### 9.4 L4 — C9a Voie CI Windows avec CGO (S ; 4.3 et 4.4 P)

- [ ] 4.1 Job `go-test-windows-cgo` dans `ci.yml` : mêmes étapes de toolchain que `release.yml:44-69`
      (gcc du runner, PAS d'action mingw), `CGO_ENABLED=1`, `go build ./cmd/server`, puis `go test` sur
      `./cmd/server/...`, `./internal/platform/diskfree/...`, `./cmd/levelup/...` (fichier `_win.go`) et,
      depuis `libs/film` et `libs/gamefiles`, les paquets `filmproc`, `internal/filelock`, `himodule` ;
      `timeout-minutes: 15` ; déclencheur `release/**` et PR vers `main` (D-7). Règle au brief : un
      échec propre à Windows est CONSIGNÉ en §10, jamais corrigé dans ce lot.
- [ ] 4.2 `docs/COMMANDS.md` et `docs/FR/COMMANDS.md` : la voie décrite.
- [ ] 4.3 **(P)** Mutation jouée : un `*_windows.go` cassé sur une branche jetable → job rouge ; lien
      du run consigné.
- [ ] 4.4 **(P)** Run vert sur la branche du plan.

### 9.5 L5 — C3 Jeton longue durée en mode `none` (S ; après 0.4 et publication lot 3)

- [ ] 5.1 `internal/api/server_apiv1.go:357-360` : le magasin de jetons est construit HORS de la branche
      `xbox` et injecté dans les deux stratégies.
- [ ] 5.2 Nouveau service `internal/service/devicecode_completion.go` (couche service, pas handler) :
      à toute connexion par code d'appareil réussie, persistance du jeton dans le store pour un profil
      suivi (porte ADR 0035 D3, résolution par xuid APRÈS création du profil selon l'ordre relevé en 0.4),
      mise en commun : aucune décision (D-4) ; `XboxSSOLinkStrategy.persistRTATokens` l'appelle (plus de
      copie).
- [ ] 5.3 Tests : mode `none`, profil suivi → fichier écrit ; non suivi → rien ; mode `xbox` inchangé ;
      garde-rails `auth/sentinel_test.go`, `sync/no_legacy_source_used_test.go` verts.
- [ ] 5.4 `docs/INSTALL.md` + `docs/FR/INSTALL.md` alignés sur le comportement livré.
- Gate : `go test ./internal/service/... ./internal/api/... ./internal/platform/auth/... ./internal/sync/...`.

### 9.6 L6 — C9b Test de bout en bout de l'arrêt (S ; après L2 et L4 ; 6.2 P)

- [ ] 6.1 Test Go sous tag `e2e_windows` (`cmd/server/e2e_arret_windows_test.go`) : compile `cmd/server`,
      le lance avec `-arret-sur-stdin` et `LEVELUP_API_PORT` libre sur une racine de données vide
      (`LEVELUP_REPO_ROOT` = dossier temporaire avec `config/` copié), attend `/health` 200 (délai 120 s),
      ferme stdin, attend la sortie (délai 60 s), vérifie `arrêt: terminé clean=true` dans le journal et
      l'absence de `*.wal`, relance, attend `/health`, arrête. Ajouté au job C9a.
- [ ] 6.2 **(P)** Run vert ; durée consignée.

### 9.7 L7 — C6 Extraction de `main.go` (S, trois sous-lots, **à risque** ; après vague 1 et publication lot 3)

- [ ] 7.1 Liste des fonctions qui migrent, PRODUITE AU BRIEF par le pilote (`grep -n "^func " main.go`),
      cible par fichier : `boot_config.go` (configuration, journaux, drapeaux), `boot_migrations.go`
      (`runMigrations` et compagnes), `boot_providers.go` (DuckDB, auth, persist), `boot_tasks.go`
      (tâches de fond), `boot_router.go`, `boot_shutdown.go`.
- [ ] 7.2 Sous-lot a : configuration → migrations ; T2 rejoué, `diff` vide. Sous-lot b : fournisseurs,
      auth, persist ; T2. Sous-lot c : tâches de fond, routeur, arrêt ; T2. Chaque sous-lot = un commit.
      Déplacement PUR : aucun changement de comportement, `git diff --stat` limité à `cmd/server/`.
- [ ] 7.3 `main()` ≤ 80 L (enchaînement), `main.go` ≤ 500 L, aucune fonction neuve > 80 L (paramètres
      regroupés en struct `bootState` si > 5) ; baseline lint non accrue.
- Gate : `go test ./cmd/server/...` ; `golangci-lint run --new-from-merge-base=origin/<release>
  ./cmd/server/...` = 0 ; T2 `diff` vide ; `go run ./cmd/server` sur copie puis `curl -fsS
  http://127.0.0.1:8000/health`. Revue adversariale (O) sur L1 + L7.

### 9.8 L8 — C11 Sons compressés (S ; après L9 et décision D-6)

- [ ] 8.1 Outil Go `cmd/sounds-encode` (ffmpeg requis) : WAV → format retenu, même arborescence.
- [ ] 8.2 Tous les sites liés à `.wav` (liste fermée au brief : table des sons `features/match-replay/
      sound/*.ts`, `exportSoundFamilies.ts`, `vehicleEngineSound.ts`, `replaySoundAssets.guard.test.ts`,
      `games/mappings/loader_vehicle_weapons.go:75`, `cmd/weapon-sounds` et ses goldens,
      `livraison_orchestrate.go:39`) ; test Vitest : chaque entrée de la table pointe un fichier existant.
- [ ] 8.3 WAV supprimés de l'arbre ; `du -sh static/sounds` consigné.
- [ ] 8.4 **(U)** Écoute comparée sur 5 sons nommés par l'utilisateur, dont une boucle moteur.
- Gate : `npm run test`, `npm run build` ; `go test ./internal/games/mappings/... ./cmd/weapon-sounds/...`.

### 9.9 L9 — Mesures : zstd de l'archive, formats de sons (H, scripts Go sous le scratchpad, non versionnés)

- [ ] 9.1 Les 20 chunks les plus récents de `data/cache/film_chunks` : taille zlib, taille après
      décompression + zstd niveaux 3 et 19 ; table en §10.
- [ ] 9.2 10 sons (dont les 3 boucles moteur de `vehicleEngineSound.ts`) encodés en AAC, Opus (ogg), FLAC
      par ffmpeg ; taille et, pour les boucles, durée d'amorce mesurée (`ffprobe -show_packets`) ; table
      en §10 ; D-6 tranchée par le pilote sur cette table, confirmée par U.

### 9.10 L10 — C7 Profil de déploiement (S ; après ADR 0044)

- [ ] 10.1 `internal/config/deployment.go` : type `Deployment` (D-16) résolu une fois depuis les réglages
      actuels (`LEVELUP_ENV`, `LEVELUP_DEMO_MODE`, `AuthMode`, `-mode desktop`) ; dérive `AuthMode`,
      hôte d'écoute (A7), capacités fines : comptes et inscriptions, invitations, groupes, mise en commun,
      administration, sauvegarde restic, Discord, sortie réseau, mises à jour, téléchargement Windows.
- [ ] 10.2 `CapabilityMap` de `/bootstrap` étendue (A6), champ `deployment` ; OpenAPI régénéré
      (`make generate-types`) ; `FeatureFlags.demo_mode` conservé.
- [ ] 10.3 `demoPolicies` (`background_tasks.go:84`) généralisé en politique par profil (A17) ; garde-rail
      existant (chaque tâche déclarée) étendu aux quatre profils.
- [ ] 10.4 Garde-rail archlint : aucune NOUVELLE lecture de `.DemoMode` / `.AuthMode` / `LEVELUP_ENV`
      hors du résolveur (allowlist datée des 89 + 55 + 13 occurrences, compte figé).
- [ ] 10.5 Front : les 66 occurrences de `demo_mode|demoMode|auth_mode|authMode` dans 23 fichiers
      (hors tests et `generated.ts`, liste au brief) lisent les capacités.
- [ ] 10.6 Tests : chaque combinaison actuelle → mêmes capacités (T3) ; profil `desktop` → hôte loopback
      quelle que soit `LEVELUP_API_HOST`.
- Gate : `go test ./internal/config/... ./internal/service/... ./internal/archlint/... ./cmd/server/...` ;
  `make generate-types` sans diff hors `deployment` ; `cd apps/web && npm run typecheck && npm run test`.

### 9.11 L11 — C8 Deux racines (S, trois sous-lots, **à risque** ; après ADR 0008 amendée)

Définition : `ResourcesRoot` = la racine du dépôt aujourd'hui (`LEVELUP_RESOURCES_DIR`, repli
`LEVELUP_REPO_ROOT`, repli détection actuelle) ; `DataRoot` = ce qui est aujourd'hui `<repoRoot>/data`
(`LEVELUP_DATA_DIR`, défaut `<ResourcesRoot>/data`) ; `LogsRoot` = `LEVELUP_LOGS_DIR` (existant).

- [ ] 11.1 Sous-lot a : table COMPLÈTE chemin → racine produite au brief par le pilote (les 193 + 18
      usages de `RepoRoot`, les 41 jointures du littéral `"data"`, `server_apiv1.go:661,1565`,
      `main.go:1256`, `provision_title.go:165`, `config.go:230,241`) ; `PathResolver` reçoit les racines ;
      T4 identique avec racines par défaut.
- [ ] 11.2 Sous-lot b : ressources (référentiel du rejeu `data/titles/*/reference/`,
      `metadata-prebuilt.zip`, `config/`, `static/`, dist web) par `ResourcesRoot` ; données
      (`db_profiles.json`, `app_settings.json`, `.env.local`, sessions, auth, caches, journaux) par
      `DataRoot` / `LogsRoot` ; les enfants de cuisson reçoivent les trois racines (`replaychild.go:60,253`,
      `runner.go:189-200`).
- [ ] 11.3 Sous-lot c : `Dockerfile:246-247`, `docker-compose.yml:15,17,97,98` : variables mortes
      retirées, nouvelles posées, montage ressources en lecture seule ; `scripts/deploy.sh` relu (staging
      sous `data/`, piège `git clean`) ; `docs/INSTALL.md` EN + FR.
- [ ] 11.4 Tests : racines distinctes → aucun chemin de ressource sous `DataRoot`, aucun chemin de
      donnée sous `ResourcesRoot` ; garde-rail « pas de `filepath.Join(..., "data", ...)` » vert ; ratchet
      nouveau : `RepoRoot` plus lu hors `internal/config` et `PathResolver` (allowlist datée à réduire).
- Gate : `go test ./internal/... ./cmd/server/...` ; démarrage réel avec trois racines distinctes puis
  `curl -fsS http://127.0.0.1:8000/health` et une page de match ouverte (`/api/v1/...` 200). Revue
  adversariale (O).

### 9.12 L12 — Port stable, événements stdout, verrou d'instance (S ; après L7, L11)

- [ ] 12.1 Port : `<DataRoot>/desktop/port` lu, sinon port libre choisi puis écrit ; `startBootServer`
      (`boot_gate.go:190-195`) retourne le port lié (`ln.Addr()`) au lieu de sortir ; `os.Exit(1)` gardé
      seulement hors `-mode desktop`.
- [ ] 12.2 Événements §6.2 (`listening`, `ready`, `failed`, `stopping`, `stopped`) par un écrivain unique
      sous verrou, seulement en `-mode desktop` ; tout journal console sur stderr dans ce mode.
- [ ] 12.3 Arrêt des enfants : selon 0.3, rattacher la cuisson au contexte d'arrêt ; `taskAppRelease`
      (`main.go:1291`) rattaché au contexte des tâches ; `clean=false` si budget dépassé ou vidage persist
      en erreur.
- [ ] 12.4 Verrou d'instance `internal/platform/instancelock` (A16) : pris au démarrage en `desktop` et
      `refresh-only`, erreur typée `ErrInstanceDejaLancee` → `failed{state:"locked"}`.
- [ ] 12.5 Test « faux lanceur » : tube stdin, stdout lu ligne à ligne, séquence `listening` → `ready` →
      EOF → `stopping` → `stopped{clean:true}` ; second lancement pendant le premier → `failed locked` ;
      ajouté au job C9.
- Gate : `go test ./cmd/server/... ./internal/platform/instancelock/...` ; **(P)** voie Windows verte.

### 9.13 L13 — Mode `desktop` (S ; après L12, ADR 0045)

- [ ] 13.1 `config.Validate` (`main.go:339-345`) accepte `desktop` (dérivé par L10) y compris sous
      `LEVELUP_ENV=production` ; `authz.Enforced` reste faux pour `desktop` (mono-utilisateur, décision
      écrite dans l'ADR 0045).
- [ ] 13.2 `/desktop/enter?secret=` : pose le cookie (nom suffixé par le port, A15, durée 30 jours,
      renouvelé à chaque entrée) et redirige ; 401 ailleurs sans cookie ; `Host` strict ; CORS = origine
      exacte ; secret de session `<DataRoot>/auth/session_secret` (A12) ; `/api/v1/desktop/strings`
      (catalogue `messages/desktop.toml`, langue de la requête).
- [ ] 13.3 Mise en commun accordée d'office au seul utilisateur local (D-4) ; bannière de consentement
      masquée par la capacité ; libellés fr + en.
- [ ] 13.4 Tests `httptest` : chaque refus, le succès, `Host` étranger, secret absent de `os.Args` ;
      allowlist datée des routes nues ; un scénario Playwright `desktop-enter.spec.ts` ; `sentinel_test`.
- Gate : `go test ./internal/api/... ./internal/config/... ./internal/authz/...` ; `cd apps/web && npm run
  typecheck && npm run test && npx playwright test desktop-enter`.

### 9.13b L13b — Coffre Windows pour les jetons (S ; après L13 ; D-8)

- [ ] 13b.1 ADR 0023 amendée : port `TokenCipher` (`platform/auth`), implémentations `plain` (hébergé,
      conteneur Linux : comportement actuel) et `dpapi` (Windows, portée utilisateur courant,
      `CryptProtectData` via `golang.org/x/sys/windows`) ; choisie par le profil de déploiement
      (`desktop` = `dpapi`), jamais par `GOOS` seul.
- [ ] 13b.2 Format du fichier `watcher_tokens/{xuid}.json` : champ `cipher` (`plain` | `dpapi`) et
      champs sensibles (`OAuthRefreshToken`, jetons Spartan / XSTS) chiffrés en base64 ; lecture d'un
      fichier `plain` sur un profil `dpapi` = migration à la première écriture ; l'inverse = refus typé
      journalisé (un fichier chiffré copié sur un autre poste est illisible, c'est le but).
- [ ] 13b.3 Tests : aller-retour `plain` ; `dpapi` sous tag Windows (voie C9) ; garde-rails
      `auth/sentinel_test.go` et `no_legacy_source_used_test.go` verts ; `token-import` / `token-capture`
      écrivent par le même port.
- Gate : `go test ./internal/platform/auth/...` ; **(P)** voie Windows verte.

### 9.14 L14 — C5 Archive des films : emplacement, occupation, choix (S ; après L11)

- [ ] 14.1 `PathResolver.FilmArchiveDir()` = `film_chunks/` ET `film_manifests/` (l'unité d'archive est le
      film ; les faits restent sous `DataRoot`), réglable (`app_settings.json`, clé `film_archive_dir`),
      déplacement assisté refusé si la cible n'est pas vide ; `libs/film/filmcache` reçoit le dossier.
- [ ] 14.2 Occupation exposée dans les réglages ; alerte par `taskDiskWatch` existant (seuils 80 / 90 %),
      pas de second mécanisme.
- [ ] 14.3 Réglage « archiver les films » (défaut oui, D-5, proposé dans `features/setup`) ; non = faits et
      rejeu produits, chunks non conservés, rétention des rejeux forcée illimitée (A18) ; fr + en.
- [ ] 14.4 Tests `PathResolver` (archive hors racine) ; garde-rail de disposition `filmcache` vert ;
      test de la combinaison refusée.
- Gate : `go test ./internal/domain/title/... ./internal/service/... ./internal/scheduler/...` ;
  `cd libs/film && go test ./filmcache/...` ; `cd apps/web && npm run typecheck`.

### 9.15 L15 — Séquence réveil (S ; après L5, L12)

- [ ] 15.1 Rattrapage au lancement SOUS LA CAPACITÉ `desktop` seulement (le VPS redémarre à chaque
      déploiement : §1 « aucun changement sur le VPS ») : premier tick immédiat si la dernière sync est
      plus vieille que l'intervalle.
- [ ] 15.2 Vérification des jetons au boot (chaque profil suivi) : réseau absent = rien marqué ; mort =
      `reauth_required` + bannière ; vivant = RT rotaté écrit.
- [ ] 15.3 Sortie de veille côté Go : écart heure murale / monotone > 2 min (A24) → même séquence.
- [ ] 15.4 `-mode refresh-only` (A10) : verrou d'instance, aucune ouverture de base, rotation, sortie ;
      si le verrou est tenu (application ouverte) : sortie 0 sans rien faire (une seule rotation à la fois,
      ADR 0023).
- Gate : `go test ./internal/scheduler/... ./internal/platform/auth/... ./cmd/server/...`.

### 9.16 L16 — Coquille Tauri (S, **à risque** ; P1)

- [ ] 16.1 `apps/desktop/` (Tauri 2, `src-tauri/`, identifiant `info.lvelup.desktop`, A22) ; page
      embarquée minimale (logo + animation, écran « Fermeture en cours… » avec le catalogue de repli
      A4 généré par `build.rs` depuis `messages/desktop.toml`) ; plugins `single-instance`,
      `window-state`, `opener`, `updater` ; `webviewInstallMode` Evergreen (A5) ; sidecar lancé par
      `std::process::Command` (stdin tube, stdout vidé en continu, stderr relayé, répertoire courant =
      ressources, environnement §6.1).
- [ ] 16.2 Machine d'états : `listening` → `ready` → charge `/desktop/enter` ; `failed` → charge la page
      de démarrage du serveur ; `ExitRequested` → fermeture de stdin, attente `stopped` (budget + 5 s),
      puis kill ; téléchargements par `on_download` (§6.6).
- [ ] 16.3 Faux sidecar `apps/desktop/testdata/fake-sidecar` (Go, `go.mod` propre, `GOWORK=off`) : émet
      les événements, sert `/desktop/enter` et `/`, réagit à l'EOF ; variantes : jamais `ready`, `failed`,
      `stopped` tardif, crash.
- [ ] 16.4 `cargo test` contre le faux sidecar : chaque variante, kill après budget, secret absent des
      arguments, stdout vidé (sidecar bavard de 10 Mo).
- [ ] 16.5 Hygiène (A23) : `.gitignore`, `Cargo.lock`, `dependabot.yml`, `Makefile` (`desktop-build`,
      `desktop-test`), `docs/COMMANDS.md` EN + FR, filtre `desktop` dans `ci.yml` + `ci_paths_filter_test`,
      CLAUDE.md amendé (§2), `NOTICE` §6 (`Cargo.lock`).
- Gate : `cargo test` ; `cargo clippy -- -D warnings` ; `cargo audit` ; **(U)** lancement local avec le
  vrai serveur : témoins vidéo (démarrage, fermeture, `arrêt: terminé clean=true`, T6 vide).
  Revue adversariale (O).

### 9.17 L17 — CI Desktop, NSIS, updater (S ; après L16 ; 17.6 P, 17.7 U)

- [ ] 17.1 Job `desktop-build` (Windows) : build Go CGO, build Tauri, NSIS par utilisateur (D-14),
      désinstalleur qui conserve les données (A22) ; **ffmpeg et ffprobe LGPL livrés** à côté du
      binaire (D-9 : archive officielle épinglée par somme de contrôle, texte LGPL et lien vers les
      sources dans les mentions tierces) ; fichier de mentions tierces (A19) avec le **texte imposé par
      les règles Microsoft et le lien vers ces règles**, repris aussi dans le pied de page web
      (`lib/appLinks.ts`, manifest `common.toml`, fr + en) ; artefact.
- [ ] 17.2 `release.yml` : étape « version check » (tag, `Cargo.toml`, `tauri.conf.json`, `-ldflags`,
      A9) ; `latest.json` signé (clé Tauri en secret du dépôt, actions épinglées par SHA, clé publique
      allowlistée `.gitleaks.toml`) ; publication GitHub Releases ; section de notes FR présente (le
      workflow échoue sinon).
- [ ] 17.3 Job `desktop-e2e` (`release/**`) : script PowerShell : installation silencieuse, lancement de
      l'exécutable, port lu dans `<DataRoot>/desktop/port`, `/health` 200, `WM_CLOSE`, attente de sortie,
      `arrêt: terminé clean=true` dans `desktop.log`, aucun `*.wal`, relancement, `/health`, fermeture.
- [ ] 17.4 Updater : proposé, jamais imposé ; n'exécute l'installeur qu'après `stopped` (A3) ; changelog
      in-app (`docs/RELEASE_NOTES.md`) ; poids de la mise à jour mesuré et consigné (§10).
- [ ] 17.5 Page « Télécharger pour Windows » côté hébergé (capacité du profil `hosted`), fr + en, avec
      l'avertissement SmartScreen expliqué (D-10 : installeur non signé) et la mention Microsoft.
- [ ] 17.6 **(P)** Runs verts ; procédure de sauvegarde de la clé d'updater exécutée (ADR 0045).
- [ ] 17.7 **(U)** Installation sur une machine vierge, désinstallation, données conservées : témoins.

### 9.18 P2 — à planifier à la clôture de P1 (liste fermée des sujets)

Fin de match par détection du processus du jeu (énumération seule), mode jeu (pas de cuisson pendant
Halo), notifications natives relayées, barre des tâches, démarrage réduit avec Windows, tâche planifiée
`refresh-only`, ffmpeg téléchargeable (D-9), rapport de diagnostic, import depuis l'hébergé, option « se
connecter à un serveur », plafond mémoire du processus, CHECKPOINT plus fréquents en profil `desktop`,
**sauvegarde périodique en Parquet vers un dossier choisi** (remplace restic, étude §3.8), **fuseau et
langue tirés du système au premier lancement** (`user_timezone` vaut `Europe/Paris` en dur), poids des
mises à jour (ressources retéléchargées à chaque version).

## 10. Découvertes (notées, non traitées)

- **C2 observé en direct sur le poste (2026-11-01, témoin T2)** : à CHAQUE démarrage depuis le
  2026-05-20, l'étape de migration `purge_data_health_warning_notifs` sur `shared_social` échoue avec
  le bug ART #23645 (`Failed to delete all rows from index. Only deleted 0 out of 198 rows`, erreur
  `FATAL` DuckDB) et le serveur démarre quand même (une ligne ERROR dans `logs/migration.log`, rien en
  console). La base locale `shared_social` porte donc un index corrompu sur 198 notifications
  `data_health_warning` ; réparation = recette append-only (ADR 0026), HORS périmètre de ce plan.
  À vérifier sur le VPS (`logs/migration.log`, lecture seule, prévenir avant) : si la prod a la même
  ligne, c'est un incident C2 en cours.

- `internal/mapcatalog` : `TestAddOverlayEntryConcurrentDossierAbsentNePerdRien` échoue en run complet
  (« verrou d'ecriture impossible a poser — passage force », 2 cartes perdues sur 8) et passe seul : le
  passage forcé sans verrou est le comportement testé comme défaut, déclenché par la contention du run
  complet. Hors périmètre ; à signaler au chantier de publication (`mapcatalog/ugc.go` y bouge).
- `taskAppRelease` tourne sous `context.Background()` (`main.go:1291`) : hors du contexte d'arrêt
  (traité en L12.3, hors périmètre avant).
- Commentaire « API FastAPI exposée en local » dans `docker-compose.yml` (vestige Python ; audit).
- `.ai/project_map.md` gelé depuis avril 2026.
- Archive de films : 36 Go le 2026-11-01 contre 104 Go annoncés le 2026-10-27 (films rapatriés puis
  purge manuelle du 26/10 ; à remesurer en T5).

## 11. Protocole de reprise de session

1. Lire ce plan (§9, statuts) et la dernière entrée `thought_log` du chantier.
2. `git log --oneline -10` sur `feat/desktop-p0` ; `ListAgents` (lots en cours).
3. Reprendre au premier item sans statut de la vague en cours ; ne pas ouvrir la vague suivante tant que
   la précédente n'est pas fusionnée et ses témoins rejoués.
