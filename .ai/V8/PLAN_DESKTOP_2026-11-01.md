# Plan — Version Desktop de LevelUp (Tauri 2 + serveur Go en sidecar) — 2026-11-01

Statut : **clos le 2026-11-09** (P0, P1 et P2 livrés ; versions 8.1.0 et 8.1.1 publiées, voir la clôture en §12 ; les suites vivent au backlog et au registre des reports). Sources : `.ai/ETUDE_DESKTOP_TAURI_2026-10-26.md` (l'étude)
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
- **CLAUDE.md** : la règle « Go/TS uniquement » est amendée par l'ADR 0045 (Rust limité à
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
  les installations partagent le même identifiant ; dépendance externe à consigner dans l'ADR 0045 (D-12).
- **A21 — Clé de signature de l'updater.** Perdue = plus aucune mise à jour automatique possible :
  procédure de sauvegarde et de rotation dans l'ADR 0045 (D-12) ; clé publique allowlistée dans `.gitleaks.toml`.
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
| D-6 | C11-a : format des sons | **Opus 96 kbit/s en Ogg** (décision utilisateur du 2026-11-01 sur la table L9 §10 : 5 % du WAV, boucles rendues à l'échantillon près ; AAC écarté car il ajoute 128 à 960 échantillons ; FLAC écarté, 35 %) ; risque accepté : Safari iOS sur le site hébergé | mesure L9 |
| D-7 | C9 : budget Windows | voie CGO sur `release/**`, les PR vers `main` (seules PR reçues par le workflow : `on.pull_request.branches: [main]`, une PR vers une branche de release ne déclenche RIEN) (la condition `feat/desktop-*` du 2026-11-01, posée « le temps du chantier », est RETIRÉE le 2026-11-04, L22c.3), `timeout-minutes: 15` | runners Windows plus chers |
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
| D-18 | Archive des films : compression (découverte L9.1) | **Oui, zstd-3** (décision utilisateur du 2026-11-01) : les nouveaux chunks sont écrits en zstd-3 à la réception, la lecture (`decfilm`/`source`) accepte brut, zlib et zstd ; les chunks existants sont recompressés par une CLI lancée par l'utilisateur, JAMAIS automatiquement ; lot L14 (items 14.5 à 14.7) | 96 % des chunks sont bruts ; −75 % mesuré, 3 s pour 20 films |

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
   Sans cookie, toute route hors `/desktop/enter` et `/health` (sonde de la coquille, sans donnée) répond 401 (`bare_routes_ratchet_test.go` : allowlist
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
- **Recette T2 corrigée (2026-11-01, fin de vague 1)** : fenêtre par DATE-HEURE UTC complète (`time` >= lancement et <= arrêt), jamais par heure seule ; après `del(.time,.source)` et retrait des durées, retirer aussi `event` (identifiant par cycle), `not_after` (jeton) et TRIER les lignes (les goroutines du démarrage s'entrelacent différemment d'un lancement à l'autre) ; la dernière ligne de la fenêtre peut différer (tâche de fond en cours à l'arrêt). Fin de vague 1 : `t2_avant2_*` (binaire `8f0b512c58`) et `t2_apres1_*` (L2) équivalents sous cette recette (98/97 lignes general, 92/92 migration) ; T3 bootstrap `xbox` identique octet pour octet avant/après ; T6 après = 0 WAL.
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

- [x] 1.1 La reprise bornée du verrou (`main.go:572`, 12 × 500 ms) est DÉPLACÉE avant `runMigrations`
      (une seule reprise, les deux appelants l'utilisent) ; l'erreur est classée par
      `duckdb.IsFileLockError` / `errors.Is(err, duckdb.ErrBaseTenueEnEcriture)`.
- [x] 1.2 Après reprise épuisée (verrou) ou sur toute autre erreur de `runMigrations` :
      `slog.ErrorContext(ctx, "migration: échec, démarrage refusé", "err", err, "state", s)` ; le serveur
      RESTE debout sur la porte de démarrage avec un état d'échec (`locked`, `migration_failed`,
      `schema_newer`) servi en 503 **non réessayable** (`Retryable: false`, champ nouveau de `bootStep`),
      événement `failed` (§6.2) ; il sort avec un code non nul au signal, à l'EOF stdin ou au point
      d'arrêt. Supprimer la ligne `migrations ignorées` (`main.go:469`).
- [x] 1.3 Garde de version : lecture de `title_schema_version` SANS ouverture en écriture (connexion
      lecture seule, fermée aussitôt) avant toute migration ; version > binaire = `ErrSchemaNewerThanBinary`
      → état `schema_newer`.
- [x] 1.4 Copie avant montée : si des étapes sont en attente, copier `<base>.duckdb` + `<base>.duckdb.wal`
      (base FERMÉE, avant l'ouverture RW) vers `PathResolver.MigrationBackupsDir()` (nouveau, distinct de
      `BackupDir`), nom `<base>.v<version>.<date>.duckdb[.wal]`, rétention 2, refus si l'espace libre
      (`platform/diskfree`) est inférieur à la taille à copier (état `migration_failed`, cause journalisée).
- [x] 1.5 Web : `serverStartup.ts` distingue un 503 non réessayable (arrêt de l'interrogation, écran
      explicite par état, libellés fr + en dans le manifest), et prolonge l'attente de 120 s tant que le
      serveur répond `migrating` (progression) ; `serverStartup.test.ts` mis à jour.
- [x] 1.6 Tests Go : étape en échec → état `migration_failed`, journal Error, code de sortie non nul ;
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

- [x] 2.1 `flags.go` : parseur unique (`flag.NewFlagSet`), drapeaux `-arret-sur-stdin`, `-mode`
      (valeurs admises : vide, `desktop`, `refresh-only` ; les deux derniers refusés jusqu'à L12/L15
      avec une erreur explicite), compatibilité des tests `os.Args` existants (`-health-check`,
      `-film-child`).
- [x] 2.2 `-arret-sur-stdin` : goroutine lisant stdin jusqu'à EOF → `requestShutdown(reason)`, fonction
      unique sous `sync.Once` que le signal et le point d'arrêt appellent aussi ; ligne de journal
      `arrêt: terminé` avec `clean` (§6.5) écrite en fin de `shutdown.go`.
- [x] 2.3 `POST /api/v1/internal/shutdown` : `LoopbackOnly`, en-tête `X-LevelUp-Shutdown-Secret` comparé
      en temps constant à `cfg.DesktopSecret` ; 404 si non configuré, 403 si faux, 202 sinon ; injection
      de `requestShutdown` par le patron de `cfg.TitleReadyNotifier` (`main.go:1280`) ; route allowlistée
      dans le ratchet des routes nues avec date.
- [x] 2.4 Air (D-17, RÉVISÉ le 2026-11-01 : `pre_cmd` tourne APRÈS le `TASKKILL` d'air 1.65, engine.go `stopBin` puis `buildRun`→`runPreCmd` ; remplacé par un lanceur `cmd/devlaunch` qui tient le stdin du serveur, serveur détaché de l'arbre, EOF = arrêt propre) ; ancien énoncé : `pre_cmd` qui appelle le point d'arrêt avec le secret de dev (`.env.local`),
      tolère l'absence de serveur ; commentaire `.air.toml:48-59` réécrit.
- [x] 2.5 Tests : EOF stdin → `requestShutdown` appelé une fois ; double déclenchement → une seule
      séquence ; handler : hors boucle locale 403, sans secret 404, mauvais secret 403, bon secret 202 ;
      `shutdown_budget_test.go` vert ; ratchet des routes nues vert.
- Gate : `go test ./cmd/server/... ./internal/api/... ./internal/config/...` ; **(P)** rechargement air
  sur le poste : ligne `arrêt: terminé clean=true`, T6 vide.

### 9.3 L3 — C4 horloge monotone + C12 police et résolveur ffmpeg (H)

- [x] 3.1 `tokencache/player_token_cache.go:83` : `time.Now().Round(0).Add(fallbackTokenTTL)` ; test
      `exp.Round(0) == exp` ; commentaire citant la preuve (A24).
- [x] 3.2 Recensement `grep -rn "time.Now().Add(" --include=*.go apps/go-api/internal | grep -v _test`
      avec le critère écrit : garder une ligne si la valeur est CONSERVÉE dans une structure (champ,
      map, cache) ET comparée plus tard à `time.Now()` ; jeter si consommée dans la même fonction
      (délai, contexte, backoff). Lignes gardées → même correction ; liste complète (gardées et jetées,
      avec la raison) dans le compte rendu, contrôlée par le pilote.
- [x] 3.3 Roboto auto-hébergée : fichiers woff2 (400/500/700/900, sous-ensembles latin + latin-ext,
      licence SIL OFL 1.1 jointe — Roboto est sous OFL sur Google Fonts, pas Apache ; police VARIABLE 100-900, deux fichiers) FOURNIS PAR LE PILOTE dans le worktree sous `apps/web/public/fonts/`
      (pas de dépendance npm) ; `@font-face` dans le CSS global ; suppression de `index.html:23-25` ;
      garde-rail Vitest : `index.html` et `dist/index.html` sans `fonts.googleapis`.
- [x] 3.4 Résolveur unique `internal/media/binpath.go` (`LEVELUP_FFMPEG_DIR`, puis dossier de
      l'exécutable, puis PATH) utilisé par `ffmpeg_slot.go:125`, `ffprobe.go:20`, `ops/healthcheck.go:93`,
      `ops/media_tooling.go:79-80` ; garde-rail archlint interdisant `"ffmpeg"` / `"ffprobe"` hors du
      résolveur dans `internal/` ET `cmd/` ; test de l'ordre de recherche.
- Gate : `go test ./internal/platform/auth/tokencache/... ./internal/media/... ./internal/ops/...
  ./internal/archlint/...` ; `cd apps/web && npm run build && grep -c fonts.g dist/index.html` = 0 ;
  `npm run test -- fonts`.

### 9.4 L4 — C9a Voie CI Windows avec CGO (S ; 4.3 et 4.4 P)

- [x] 4.1 Job `go-test-windows-cgo` dans `ci.yml` : mêmes étapes de toolchain que `release.yml:44-69`
      (gcc du runner, PAS d'action mingw), `CGO_ENABLED=1`, `go build ./cmd/server`, puis `go test` sur
      `./cmd/server/...`, `./internal/platform/diskfree/...`, `./cmd/levelup/...` (fichier `_win.go`) et,
      depuis `libs/film` et `libs/gamefiles`, les paquets `filmproc`, `internal/filelock`, `himodule` ;
      `timeout-minutes: 15` ; déclencheur `release/**` et pull requests (D-7). Règle au brief : un
      échec propre à Windows est CONSIGNÉ en §10, jamais corrigé dans ce lot.
- [x] 4.2 `docs/COMMANDS.md` et `docs/FR/COMMANDS.md` : la voie décrite.
- [x] 4.3 **(P)** Mutation jouée : un `*_windows.go` cassé sur une branche jetable → job rouge ; lien
      du run consigné.
- [x] 4.4 **(P)** Run vert sur la branche du plan. Fait le 2026-11-01 : run `41330468268` (commit `ef40d59e79`), job vert en 3 min 54 s (https://github.com/JGtm/LevelUp/actions/runs/41330468268/job/124919357210) ; 4.3 : branche jetable `feat/desktop-mutation-win` (test `_windows_test.go` en échec, un `_windows.go` cassé est arrêté avant le push par le hook `go-vet-cgo`), job ROUGE en 3 min 13 s (https://github.com/JGtm/LevelUp/actions/runs/41330598748/job/124919671555), branche supprimée.

### 9.5 L5 — C3 Jeton longue durée en mode `none` (S ; après 0.4 et publication lot 3)

- [x] 5.1 `internal/api/server_apiv1.go:357-360` : le magasin de jetons est construit HORS de la branche
      `xbox` et injecté dans les deux stratégies.
- [x] 5.2 Nouveau service `internal/service/devicecode_completion.go` (couche service, pas handler) :
      à toute connexion par code d'appareil réussie, persistance du jeton dans le store pour un profil
      suivi (porte ADR 0035 D3, résolution par xuid APRÈS création du profil selon l'ordre relevé en 0.4),
      mise en commun : aucune décision (D-4) ; `XboxSSOLinkStrategy.persistRTATokens` l'appelle (plus de
      copie).
- [x] 5.3 Tests : mode `none`, profil suivi → fichier écrit ; non suivi → rien ; mode `xbox` inchangé ;
      garde-rails `auth/sentinel_test.go`, `sync/no_legacy_source_used_test.go` verts.
- [x] 5.4 `docs/INSTALL.md` + `docs/FR/INSTALL.md` alignés sur le comportement livré.
- Gate : `go test ./internal/service/... ./internal/api/... ./internal/platform/auth/... ./internal/sync/...`.

### 9.6 L6 — C9b Test de bout en bout de l'arrêt (S ; après L2 et L4 ; 6.2 P)

- [x] 6.1 Test Go sous tag `e2e_windows` (`cmd/server/e2e_arret_desktop_test.go` (nom sans suffixe de plateforme : ratchet `no_accidental_goos_suffix_test.go` ; tag `e2e_windows && windows`)) : compile `cmd/server`,
      le lance avec `-arret-sur-stdin` et `LEVELUP_API_PORT` libre sur une racine de données vide
      (`LEVELUP_REPO_ROOT` = dossier temporaire avec `config/` copié), attend `/health` 200 (délai 120 s),
      ferme stdin, attend la sortie (délai 60 s), vérifie `arrêt: terminé clean=true` dans le journal et
      l'absence de `*.wal`, relance, attend `/health`, arrête. Ajouté au job C9a.
- [x] 6.2 **(P)** Run vert ; durée consignée. Fait le 2026-11-01 : run `41338448752` (commit `806e7cf0bb`), job Windows CGO vert avec l'étape de bout en bout (job entier 3 min 53 s sur le run précédent `41337064204`) ; test e2e local 18 à 22 s.

### 9.7 L7 — C6 Extraction de `main.go` (S, trois sous-lots, **à risque** ; après vague 1 et publication lot 3)

- [x] 7.1 Liste des fonctions qui migrent, PRODUITE AU BRIEF par le pilote (`grep -n "^func " main.go`),
      cible par fichier : `boot_config.go` (configuration, journaux, drapeaux), `boot_migrations.go`
      (`runMigrations` et compagnes), `boot_providers.go` (DuckDB, auth, persist), `boot_tasks.go`
      (tâches de fond), `boot_router.go`, `boot_shutdown.go`.
- [x] 7.2 Sous-lot a : configuration → migrations ; T2 rejoué, `diff` vide. Sous-lot b : fournisseurs,
      auth, persist ; T2. Sous-lot c : tâches de fond, routeur, arrêt ; T2. Chaque sous-lot = un commit.
      Déplacement PUR : aucun changement de comportement, `git diff --stat` limité à `cmd/server/`.
- [x] 7.3 `main()` ≤ 80 L (enchaînement), `main.go` ≤ 500 L, aucune fonction neuve > 80 L (paramètres
      regroupés en struct `bootState` si > 5) ; baseline lint non accrue.
- Gate : `go test ./cmd/server/...` ; `golangci-lint run --new-from-merge-base=origin/<release>
  ./cmd/server/...` = 0 ; T2 `diff` vide ; `go run ./cmd/server` sur copie puis `curl -fsS
  http://127.0.0.1:8000/health`. Revue adversariale (O) sur L1 + L7.

### 9.8 L8 — C11 Sons compressés (S ; après L9 et décision D-6)

- [x] 8.1 Outil Go `cmd/sounds-encode` (ffmpeg requis) : WAV → format retenu, même arborescence.
- [x] 8.2 Tous les sites liés à `.wav` (liste fermée au brief : table des sons `features/match-replay/
      sound/*.ts`, `exportSoundFamilies.ts`, `vehicleEngineSound.ts`, `replaySoundAssets.guard.test.ts`,
      `games/mappings/loader_vehicle_weapons.go:75`, `cmd/weapon-sounds` et ses goldens,
      `livraison_orchestrate.go:39`) ; test Vitest : chaque entrée de la table pointe un fichier existant.
- [x] 8.3 WAV supprimés de l'arbre ; `du -sh static/sounds` consigné.
- [x] 8.4 **(U)** Écoute comparée sur 5 sons nommés par l'utilisateur, dont une boucle moteur. Fait le 2026-11-02 : dossier `Downloadsecoute_sons_levelup` (WAV d'origine + Opus, page `ecoute.html`) ; verdict de l'utilisateur : « Sons ok, parfaits ».
- Gate : `npm run test`, `npm run build` ; `go test ./internal/games/mappings/... ./cmd/weapon-sounds/...`.

### 9.9 L9 — Mesures : zstd de l'archive, formats de sons (H, scripts Go sous le scratchpad, non versionnés)

- [x] 9.1 Les 20 chunks les plus récents de `data/cache/film_chunks` : taille zlib, taille après
      décompression + zstd niveaux 3 et 19 ; table en §10.
- [x] 9.2 10 sons (dont les 3 boucles moteur de `vehicleEngineSound.ts`) encodés en AAC, Opus (ogg), FLAC
      par ffmpeg ; taille et, pour les boucles, durée d'amorce mesurée (`ffprobe -show_packets`) ; table
      en §10 ; D-6 tranchée par le pilote sur cette table, confirmée par U.

### 9.10 L10 — C7 Profil de déploiement (S ; après ADR 0044)

- [x] 10.1 `internal/config/deployment.go` : type `Deployment` (D-16) résolu une fois depuis les réglages
      actuels (`LEVELUP_ENV`, `LEVELUP_DEMO_MODE`, `AuthMode`, `-mode desktop`) ; dérive `AuthMode`,
      hôte d'écoute (A7), capacités fines : comptes et inscriptions, invitations, groupes, mise en commun,
      administration, sauvegarde restic, Discord, sortie réseau, mises à jour, téléchargement Windows.
- [x] 10.2 `CapabilityMap` de `/bootstrap` étendue (A6), champ `deployment` ; OpenAPI régénéré
      (`make generate-types`) ; `FeatureFlags.demo_mode` conservé.
- [x] 10.3 `demoPolicies` (`background_tasks.go:84`) généralisé en politique par profil (A17) ; garde-rail
      existant (chaque tâche déclarée) étendu aux quatre profils.
- [x] 10.4 Garde-rail archlint : aucune NOUVELLE lecture de `.DemoMode` / `.AuthMode` / `LEVELUP_ENV`
      hors du résolveur (allowlist datée des 89 + 55 + 13 occurrences, compte figé).
- [x] 10.5 Front : les 66 occurrences de `demo_mode|demoMode|auth_mode|authMode` dans 23 fichiers
      (hors tests et `generated.ts`, liste au brief) lisent les capacités.
- [x] 10.6 Tests : chaque combinaison actuelle → mêmes capacités (T3) ; profil `desktop` → hôte loopback
      quelle que soit `LEVELUP_API_HOST`.
- Gate : `go test ./internal/config/... ./internal/service/... ./internal/archlint/... ./cmd/server/...` ;
  `make generate-types` sans diff hors `deployment` ; `cd apps/web && npm run typecheck && npm run test`.

### 9.11 L11 — C8 Deux racines (S, trois sous-lots, **à risque** ; après ADR 0008 amendée)

Définition : `ResourcesRoot` = la racine du dépôt aujourd'hui (`LEVELUP_RESOURCES_DIR`, repli
`LEVELUP_REPO_ROOT`, repli détection actuelle) ; `DataRoot` = ce qui est aujourd'hui `<repoRoot>/data`
(`LEVELUP_DATA_DIR`, défaut `<ResourcesRoot>/data`) ; `LogsRoot` = `LEVELUP_LOGS_DIR` (existant).

- [x] 11.1 Sous-lot a : table COMPLÈTE chemin → racine produite au brief par le pilote (les 193 + 18
      usages de `RepoRoot`, les 41 jointures du littéral `"data"`, `server_apiv1.go:661,1565`,
      `main.go:1256`, `provision_title.go:165`, `config.go:230,241`) ; `PathResolver` reçoit les racines ;
      T4 identique avec racines par défaut.
- [x] 11.2 Sous-lot b : ressources (référentiel du rejeu `data/titles/*/reference/`,
      `metadata-prebuilt.zip`, `config/`, `static/`, dist web) par `ResourcesRoot` ; données
      (`db_profiles.json`, `app_settings.json`, `.env.local`, sessions, auth, caches, journaux) par
      `DataRoot` / `LogsRoot` ; les enfants de cuisson reçoivent les trois racines (`replaychild.go:60,253`,
      `runner.go:189-200`).
- [x] 11.3 Sous-lot c : `Dockerfile:246-247`, `docker-compose.yml:15,17,97,98` : variables mortes
      retirées, nouvelles posées, montage ressources en lecture seule ; `scripts/deploy.sh` relu (staging
      sous `data/`, piège `git clean`) ; `docs/INSTALL.md` EN + FR.
- [x] 11.5 `LogsRoot` créé AVANT tout écrivain (découverte §10, 2026-11-01 : `crash_output` ouvre
      `logs/server.crash.log` avant la création de `logs/` sur une racine neuve) ; test : racine neuve → fichier de
      capture présent après le premier démarrage.
- [x] 11.4 Tests : racines distinctes → aucun chemin de ressource sous `DataRoot`, aucun chemin de
      donnée sous `ResourcesRoot` ; garde-rail « pas de `filepath.Join(..., "data", ...)` » vert ; ratchet
      nouveau : `RepoRoot` plus lu hors `internal/config` et `PathResolver` (allowlist datée à réduire).
- Gate : `go test ./internal/... ./cmd/server/...` ; démarrage réel avec trois racines distinctes puis
  `curl -fsS http://127.0.0.1:8000/health` et une page de match ouverte (`/api/v1/...` 200). Revue
  adversariale (O).

### 9.12 L12 — Port stable, événements stdout, verrou d'instance (S ; après L7, L11)

- [x] 12.1 Port : `<DataRoot>/desktop/port` lu (`PathResolver.DesktopPortPath`), sinon port libre choisi puis écrit
      atomiquement (Warn « port de repli » si l'ancien était occupé) ; `startBootServer` reçoit le listener lié
      (`cmd/server/desktop_port.go`, `boot_listen.go`) ; `os.Exit(1)` gardé hors `-mode desktop`, en desktop l'échec
      d'écoute annonce `failed{state:"port"}` ; `-mode desktop` n'est plus refusé (`refresh-only` ne l'est plus depuis L15).
- [x] 12.2 Événements §6.2 par un écrivain unique sous verrou (`cmd/server/desktop_events.go`), seulement en
      `-mode desktop` ; le gestionnaire console écrivait déjà sur stderr (vérifié), garde-rail `stdout_guard_test.go`.
- [x] 12.3 Arrêt des enfants : cuisson sous `schedulerCtx` (relevé 0.3 confirmé, test
      `libs/film/filmproc/runner_annulation_test.go` : contexte annulé = enfant tué) ; `taskAppRelease` sous
      `schedulerCtx` ET suivie par `schedulerWG` (`appReleaseTask`, test) ; `clean=false` sur budget dépassé ou vidage
      en erreur : déjà fait (L2), `shutdown_budget_test.go`.
- [x] 12.4 Verrou d'instance `internal/platform/instancelock` (patron recopié de `libs/film/internal/filelock`), pris
      avant le port, rendu avant `stopped` ; `ErrInstanceDejaLancee` -> `failed{state:"locked"}` ; test à deux
      processus. Prise en `refresh-only` : L15.
- [x] 12.6 Arrêt dans les premières secondes : `worldenrich.BuildEnricher/BuildMultiHaloSource/BuildHaloSource`
      reçoivent `ctx` (la résolution eager n'utilisait que `context.Background()`), la boucle de rafraîchissement
      du watcher tourne sous `schedulerCtx` et dans `schedulerWG` ; tests `refresh_loop_stop_test.go`,
      `wiring_ctx_test.go`.
- [x] 12.5 Test « faux lanceur » : `cmd/server/desktop_launcher_test.go` (`TestE2ELanceurDesktop`, tag
      `e2e_windows`, entre dans l'étape CI `-run TestE2E` existante).
- Gate : `go test ./cmd/server/... ./internal/platform/instancelock/...` ; **(P)** voie Windows verte.

### 9.13 L13 — Mode `desktop` (S ; après L12, ADR 0045)

- [x] 13.1 `config.Validate` accepte `desktop` (dérivé par L10, `refusesUnsafeConfig`) y compris sous
      `LEVELUP_ENV=production` ; `authz.Enforced` reste faux pour `desktop` (mono-utilisateur, ADR 0045 D-7).
      Secret de session (A12) : `<DataRoot>/auth/session_secret` (`PathResolver.SessionSecretPath`), généré au
      premier lancement (32 octets, `0600`), `Validate` refuse le défaut et un secret court en `desktop`
      (`validateStartup`). Tests `config_desktop_session_test.go`.
- [x] 13.2 `GET /desktop/enter?secret=` (capacité `ShellEntryGate`, `LoopbackOnly`) : cookie `levelup_session_<port>`
      (HMAC lié au port, 30 jours, renouvelé à chaque entrée, `HttpOnly`, `SameSite=Lax`), 302 vers `/` ;
      `middleware.DesktopGate` : Host strict (421), cookie exigé (401 `desktop_unauthorized`), `/health` ouvert ;
      CORS/CSRF = origine exacte du port lié (`AppConfig.SetBoundPort`) ; `GET /api/v1/desktop/strings`
      (`messages/desktop.toml`, fr + en). Écran web « accès expiré » sur le 401. Pas d'entrée d'allowlist des
      routes nues (route gardée par `LoopbackOnly`, `/health` déjà listée) : `TestBareRoutesRatchet_DesktopEntryRoutes`.
- [x] 13.3 Mise en commun accordée d'office : `DeviceCodeCompletion.WithLocalPoolGrant` (capacité
      `PoolGrantedToLocalUser`), base `desktop_single_user` ; bannière masquée par `can_pool_tokens` (faux en desktop
      depuis L10, aucun libellé neuf).
- [x] 13.4 Tests `httptest` sur la pile réelle, tests du garde, secret absent de `os.Args`
      (`TestParseServerFlags_AucunDrapeauNePorteLeSecret`), `desktop-enter.spec.ts`, `TestE2ELanceurDesktop` étendu.
- Gate : `go test ./internal/api/... ./internal/config/... ./internal/authz/...` ; `cd apps/web && npm run
  typecheck && npm run test && npx playwright test desktop-enter`.

### 9.13b L13b — Coffre Windows pour les jetons (S ; après L13 ; D-8)

- [x] 13b.1 ADR 0023 amendée : port `TokenCipher` (`platform/auth`), implémentations `plain` (hébergé,
      conteneur Linux : comportement actuel) et `dpapi` (Windows, portée utilisateur courant,
      `CryptProtectData` via `golang.org/x/sys/windows`) ; choisie par le profil de déploiement
      (`desktop` = `dpapi`), jamais par `GOOS` seul.
- [x] 13b.2 Format du fichier `watcher_tokens/{xuid}.json` : champ `cipher` (`plain` | `dpapi`) et
      champs sensibles (`OAuthRefreshToken`, jetons Spartan / XSTS) chiffrés en base64 ; lecture d'un
      fichier `plain` sur un profil `dpapi` = migration à la première écriture ; l'inverse = refus typé
      journalisé (un fichier chiffré copié sur un autre poste est illisible, c'est le but).
- [x] 13b.3 Tests : aller-retour `plain` ; `dpapi` sous tag Windows (voie C9) ; garde-rails
      `auth/sentinel_test.go` et `no_legacy_source_used_test.go` verts ; `token-import` / `token-capture`
      écrivent par le même port.
- Clôture L13b (2026-11-02). Écarts : (a) le port vit dans le sous-paquet `platform/auth/tokencipher` (alias
  `auth.TokenCipher`) ; (b) le choix est une capacité `TokenVault` et le chiffreur du processus est posé par le chargement
  de la configuration (`config_token_vault.go`), lu par `NewMultiUserTokenStore(dir)` : aucun des ~57 appelants ne change ;
  (c) un fichier `plain` n'écrit pas de champ `cipher` (fichiers hébergés identiques à l'octet) ; (d) le serveur refuse de
  démarrer si le profil exige le coffre et que le système ne l'offre pas ; (e) `token-import` / `token-capture` écrivent
  par le chiffreur du profil résolu : un outil lancé HORS du binaire serveur résout un profil non desktop donc `plain`
  (voir le journal).
- Gate : `go test ./internal/platform/auth/...` ; **(P)** voie Windows verte.

### 9.14 L14 — C5 Archive des films : emplacement, occupation, choix (S ; après L11)

- [x] 14.1 `PathResolver.FilmArchiveDir()` = `film_chunks/` ET `film_manifests/` (l'unité d'archive est le
      film ; les faits restent sous `DataRoot`), réglable (`app_settings.json`, clé `film_archive_dir`),
      déplacement assisté refusé si la cible n'est pas vide ; `libs/film/filmcache` reçoit le dossier.
- [x] 14.2 Occupation exposée dans les réglages ; alerte par `taskDiskWatch` existant (seuils 80 / 90 %),
      pas de second mécanisme.
- [x] 14.3 Réglage « archiver les films » (défaut oui, D-5, proposé dans `features/setup`) ; non = faits et
      rejeu produits, chunks non conservés, rétention des rejeux forcée illimitée (A18) ; fr + en.
- [x] 14.4 Tests `PathResolver` (archive hors racine) ; garde-rail de disposition `filmcache` vert ;
      test de la combinaison refusée.
- [x] 14.5 (D-18) Écriture des chunks en zstd-3 à la réception (`sync/filmfetch` ou le point unique d'écriture du cache de film : relève-le) ; lecture dans `libs/film` (`source` / `decfilm.Inflate`) acceptant brut, zlib ET zstd par en-tête magique (`28 b5 2f fd`), aucune autre couche ne lit les octets ; empreinte du film inchangée (les faits sont calculés sur les octets DÉCOMPRESSÉS : vérifier sur `replay-equiv`, 20 films identiques).
- [x] 14.6 (D-18) CLI `levelup film recompress` (ou sous-commande existante la plus proche) : recompresse en zstd-3 les chunks bruts et zlib existants, un film à la fois, écriture atomique (`.part` + renommage), reprise possible, journal par film, JAMAIS lancée automatiquement (décision utilisateur : purge et recompression manuelles) ; `--dry-run` qui chiffre le gain.
- [x] 14.7 (D-18) Tests : lecture des trois formats ; `replay-equiv` identique après recompression d'un film de l'harnais ; mesure consignée (`du -sh data/cache/film_chunks` avant/après sur le poste, lancée par l'utilisateur).
- Clôture L14 (2026-11-02). Écarts : (a) 14.1 le déplacement n'est pas fait par le réglage mais par `levelup film
  archive-move` (serveur arrêté) : le réglage refuse un changement tant que l'archive contient des films ou que la
  cible n'est pas vide ; (b) 14.6 l'écriture atomique est `atomicfile.WriteFileStrict` (temporaire du même dossier +
  fsync + renommage) et non un `.part` nommé ; (c) 14.7 : la mesure `du -sh data/cache/film_chunks` avant/après
  sur le cache réel reste à l'utilisateur ; la simulation sur copie de 3 films (57,3 Mio -> 14,9 Mio, -74 %) et
  `replay-equiv` identique avant/après sur le film 51101d1d (copie, 11,2 Mo -> 2,4 Mo) sont consignés au journal.
- Gate : `go test ./internal/domain/title/... ./internal/service/... ./internal/scheduler/...` ;
  `cd libs/film && go test ./filmcache/...` ; `cd apps/web && npm run typecheck`.

### 9.15 L15 — Séquence réveil (S ; après L5, L12)

- [x] 15.1 Rattrapage au lancement SOUS LA CAPACITÉ `desktop` seulement (le VPS redémarre à chaque
      déploiement : §1 « aucun changement sur le VPS ») : premier tick immédiat si la dernière sync est
      plus vieille que l'intervalle.
- [x] 15.2 Vérification des jetons au boot (chaque profil suivi) : réseau absent = rien marqué ; mort =
      `reauth_required` + bannière ; vivant = RT rotaté écrit.
- [x] 15.3 Sortie de veille côté Go : écart heure murale / monotone > 2 min (A24) → même séquence.
- [x] 15.4 `-mode refresh-only` (A10) : verrou d'instance, aucune ouverture de base, rotation, sortie ;
      si le verrou est tenu (application ouverte) : sortie 0 sans rien faire (une seule rotation à la fois,
      ADR 0023).
- Gate : `go test ./internal/scheduler/... ./internal/platform/auth/... ./cmd/server/...`.

### 9.16 L16 — Coquille Tauri (S, **à risque** ; P1)

- [x] 16.1 `apps/desktop/` (Tauri 2, `src-tauri/`, identifiant `info.lvelup.desktop`, A22) ; page
      embarquée minimale (logo + animation, écran « Fermeture en cours… » avec le catalogue de repli
      A4 généré par `build.rs` depuis `messages/desktop.toml`) ; plugins `single-instance`,
      `window-state`, `opener`, `updater` ; `webviewInstallMode` Evergreen (A5) ; sidecar lancé par
      `std::process::Command` (stdin tube, stdout vidé en continu, stderr relayé, répertoire courant =
      ressources, environnement §6.1).
- [x] 16.2 Machine d'états : `listening` → `ready` → charge `/desktop/enter` ; `failed` → charge la page
      de démarrage du serveur ; `ExitRequested` → fermeture de stdin, attente `stopped` (budget + 5 s),
      puis kill ; téléchargements par `on_download` (§6.6).
- [x] 16.3 Faux sidecar `apps/desktop/testdata/fake-sidecar` (Go, `go.mod` propre, `GOWORK=off`) : émet
      les événements, sert `/desktop/enter` et `/`, réagit à l'EOF ; variantes : jamais `ready`, `failed`,
      `stopped` tardif, crash.
- [x] 16.4 `cargo test` contre le faux sidecar : chaque variante, kill après budget, secret absent des
      arguments, stdout vidé (sidecar bavard de 10 Mo).
- [x] 16.5 Hygiène (A23) : `.gitignore`, `Cargo.lock`, `dependabot.yml`, `Makefile` (`desktop-build`,
      `desktop-test`), `docs/COMMANDS.md` EN + FR, filtre `desktop` dans `ci.yml` + `ci_paths_filter_test`,
      CLAUDE.md amendé (§2), `NOTICE` §6 (`Cargo.lock`).
- Gate : `cargo test` ; `cargo clippy -- -D warnings` ; `cargo audit` ; **(U)** lancement local avec le
  vrai serveur : témoins vidéo (démarrage, fermeture, `arrêt: terminé clean=true`, T6 vide).
  Revue adversariale (O).

### 9.17 L17 — CI Desktop, NSIS, updater (S ; après L16 ; 17.6 P, 17.7 U)

- [x] 17.1 Job `desktop-build` (Windows) : build Go CGO, build Tauri, NSIS par utilisateur (D-14),
      désinstalleur qui conserve les données (A22) ; **ffmpeg et ffprobe LGPL livrés** à côté du
      binaire (D-9 : archive officielle épinglée par somme de contrôle, texte LGPL et lien vers les
      sources dans les mentions tierces) ; fichier de mentions tierces (A19) avec le **texte imposé par
      les règles Microsoft et le lien vers ces règles**, repris aussi dans le pied de page web
      (`lib/appLinks.ts`, manifest `common.toml`, fr + en) ; artefact. FAIT 2026-11-02 : scripts `apps/desktop/scripts/`
      (`fetch-ffmpeg.ps1`, `stage-bundle.ps1`, `build-installer.ps1`), `tauri.bundle.conf.json`, crochet NSIS
      `src-tauri/windows/hooks.nsh`, `THIRD_PARTY_NOTICES.md`, mention Microsoft du pied de page (`MicrosoftNotice`),
      `LEVELUP_WEB_DIST` par défaut `<ResourcesRoot>/web` en desktop (`config_desktop.go`, test) ; job dans `release.yml`.
- [x] 17.2 `release.yml` : étape « version check » (tag, `Cargo.toml`, `tauri.conf.json`, `-ldflags`,
      A9) ; `latest.json` signé (clé Tauri en secret du dépôt, actions épinglées par SHA, clé publique
      allowlistée `.gitleaks.toml`) ; publication GitHub Releases ; section de notes FR présente (le
      workflow échoue sinon). FAIT 2026-11-02 : `check-version.sh` (tag/Cargo.toml/tauri.conf.json/`-ldflags` relus dans le
      binaire, testé en local avec écart), échec avant compilation sans le secret `TAURI_SIGNING_PRIVATE_KEY`, signature
      par le CLI (`tauri.updater.conf.json`, `.sig` produit en local avec la clé de génération), `latest.json` par `jq` dans
      `publish` (gabarit rejoué en local), clé publique dans `tauri.conf.json` et `.gitleaks.toml`, déclenchement aussi
      sur `release/**` (sans signature ni publication). Le run CI lui-même est 17.6 (pilote).
- [x] 17.3 Job `desktop-e2e` (`release/**`) : script PowerShell : installation silencieuse, lancement de
      l'exécutable, port lu dans `<DataRoot>/desktop/port`, `/health` 200, `WM_CLOSE`, attente de sortie,
      `arrêt: terminé clean=true` dans `desktop.log`, aucun `*.wal`, relancement, `/health`, fermeture. FAIT 2026-11-02 :
      `apps/desktop/scripts/e2e-install.ps1` (ajoute la désinstallation silencieuse et l'assertion « racine de données
      conservée ») joué en local sur l'installeur construit : exit 0, deux lancements sur le même port, `clean=true` deux
      fois, 0 WAL ; refus vérifié si une installation existe (exit 1) ; job `desktop-e2e` de `release.yml` (le journal
      qui porte la ligne est `general.log`, `desktop.log` la recopie). Cibles `make desktop-installer` / `desktop-e2e`.
- [x] 17.4 Updater : proposé, jamais imposé ; n'exécute l'installeur qu'après `stopped` (A3) ; changelog
      in-app (`docs/RELEASE_NOTES.md`) ; poids de la mise à jour mesuré et consigné (§10).
- [x] 17.5 Page « Télécharger pour Windows » côté hébergé (capacité du profil `hosted`), fr + en, avec
      l'avertissement SmartScreen expliqué (D-10 : installeur non signé) et la mention Microsoft. FAIT 2026-11-02 : route
      `/telecharger` (chemin anonyme), `features/download/` (dictionnaire `Record<Locale, T>`), capacité existante
      `can_offer_windows_download` (aucune capacité neuve), lien « Application Windows » du pied de page conditionné par la
      même capacité, lien vers `releases/latest`, tests Vitest (capacité vraie/fausse x fr/en, pied de page, chemin anonyme).
- [x] 17.6 **(P, au pilote)** Runs verts ; procédure de sauvegarde de la clé d'updater exécutée (ADR 0045). Premier run Release vert sur la release le 2026-11-02 (desktop-build + desktop-e2e sur runner Windows, sans signature) ; clé sauvegardée hors PC par l'utilisateur et secret `TAURI_SIGNING_PRIVATE_KEY` en place le 2026-11-02 ; le run signé aura lieu au premier tag. Procédure écrite dans l'ADR 0045 (« Implementation notes (L17) »), reports au registre.
- [x] 17.7 **(U)** PASSÉ le 2026-11-02 (sur ce poste, sans installation préalable ; journaux vérifiés par le pilote) — Installation sur une machine vierge, désinstallation, données conservées : témoins. À l'utilisateur, avec 18.4. PRÉPARÉ : mode d'emploi en cinq étapes dans `docs/RUNBOOK_DESKTOP.md`.

### 9.17b L18 — Interface bureau (S après maquette validée par U ; EN DERNIER, décision utilisateur du 2026-11-02)

Constat de l'utilisateur au premier lancement de la coquille : pied de page de site, barre de navigation de site,
image de couverture de l'accueil « ça ne fait pas app de bureau ». Décision : lot fait en dernier, après L17.

- [x] 18.1 Maquette (Claude Design ou artefact) validée par l'utilisateur AVANT tout code : accueil sans image de
      couverture (tableau de bord, en-tête compact), **barre latérale** (celle de la vue mobile, retenue par
      l'utilisateur) à la place de la barre haute, fenêtre « À propos » (version, licences, mention Microsoft
      obligatoire A19, lien de dons, « retirés sur simple demande » pour les sons). FAIT : maquette validée par
      l'utilisateur le 2026-11-02, avec cinq décisions : (1) barre latérale TOUJOURS dépliée (pas de mode icônes) ;
      (2) « À propos » = fenêtre MODALE dans l'application ; (3) dossier des données dans « À propos » : chemin affiché +
      bouton « Copier » (pas d'ouverture de l'Explorateur) ; (4) en-tête de l'accueil : bannière d'identité et indicateurs
      EXACTEMENT comme sur le site, seule l'image de couverture disparaît en bureau ; (5) le bouton « Rechercher une mise à
      jour » de « À propos » DOIT fonctionner réellement.
- [x] 18.2 Branché sur les capacités de `/bootstrap` (jamais sur le nom du profil, ADR 0044 D-4) : capacité fine
      `compact_chrome` (ou nom retenu) vraie en `desktop` ; pied de page retiré, éléments « site » de la navigation
      retirés (déjà portés par `can_offer_windows_download`, `can_use_accounts`), « À propos » ajouté ; le site hébergé
      ne change pas (tests fixtures `hosted` identiques). FAIT 2026-11-02 : capacité `can_use_compact_chrome`
      (`DeploymentCapabilities.CompactChrome`, vraie en `desktop` seulement, tests des quatre profils) ; `/bootstrap` porte
      `app_info` (`version` = version du serveur, `data_root` = racine des données) SEULEMENT sous cette capacité ;
      `AppShell` rend `SidebarNav` (barre latérale : marque, sélecteur de jeu, joueur + cloche, sections, pied Aide /
      Paramètres / thème / À propos / version) à la place de `NavL1`, sans `AppFooter` ; le rendu des sections est
      `NavSectionList`, PARTAGÉ avec le tiroir mobile (migré), le filtrage par capacité de titre est
      `useVisibleNavSections` (extrait de `NavL1`), la bascule de joueur `usePlayerSwitch` ; retirés du chrome bureau :
      déconnexion, Application Windows, CSinsight, Développeur ; accueil sans `HomeHeroBanner` (bannière d'identité et
      indicateurs inchangés). Décision `NavL2` : la barre d'onglets Carrière / Communauté est retirée en bureau (la barre
      latérale montre déjà les onglets de la section active), les filtres Stats restent (ce n'est pas une barre d'onglets).
      Site hébergé : tests existants de `NavL1`, `AppFooter`, `HomePage` inchangés et verts.
- [x] 18.2b Fenêtre « À propos » (`components/shell/AboutDialog.tsx`) : modale (rôle `dialog`, `aria-modal`, focus sur
      « Fermer » puis rendu au déclencheur, Tab bouclé, Échap et clic sur le fond ferment) ; logo, version, bloc mise à
      jour, colonnes Le projet (Nouveautés, Code source, Signaler un problème, Licence AGPL-3.0, Mentions tierces au tag de
      la version via `thirdPartyNoticesUrl`) / Soutenir / Données (chemin + « Copier » via `useCopyToClipboard`,
      Confidentialité, Conditions), mention Microsoft (`MicrosoftNotice`), ffmpeg LGPL, sons retirés sur simple demande,
      copyright ; liens externes par `lib/appLinks.ts` ; libellés dans `common.toml` (fr + en). FAIT 2026-11-02.
- [x] 18.2c Recherche de mise à jour qui fonctionne. FAIT 2026-11-02 : le bouton appelle `GET
      http://levelup-shell.localhost/check-update` (`lib/desktop/shellBridge.ts`, `fetch` en `no-cors`) ; le protocole
      embarqué répond 204 et lance la vérification dans la coquille (`render::route`, `update_manual.rs`,
      `updater::run_flow` partagé avec la vérification du lancement : proposée, jamais imposée, installée après l'arrêt
      propre) ; boîtes natives `update_none`, `update_check_failed`, `update_unavailable_debug`, `update_busy` (fr + en,
      `desktop.toml`, `desktop_strings.go`, catalogue de repli) ; une vérification à la fois (`UpdateBusy`). Écart avec le
      mécanisme proposé : pas de navigation ni de redirection `?back=` (la page ne quitte jamais l'application : rien à
      ramener, aucune URL de retour à valider), voir ADR 0045 notes L18. Tests : `render.rs` (seul le chemin exact),
      `navigation.rs`, `updater.rs` (machine d'états manuelle), `tests/update_manual.rs` (à jour, échec, proposée, verrou,
      indisponible).
- [x] 18.3 Libellés fr + en ; tokens sémantiques ; tests Vitest par capacité ; un scénario Playwright `desktop-chrome`.
      FAIT 2026-11-02 : Vitest `AppShell.chrome`, `SidebarNav`, `NavSectionList`, `AboutDialog`, `NavL2` (bureau),
      `HomePage.compactChrome`, `shellBridge`, `appLinks.thirdParty` ; Playwright `e2e/desktop-chrome.spec.ts` (4 cas, joués
      sur Vite local : verts).
- [x] 18.4 **(U)** PASSÉ le 2026-11-02 (retours installeur, lien de connexion, panneaux latéraux → L19) — Gate visuel sur l'application installée.
- Gate : `cd apps/web && npm run typecheck && npm run lint && npm run test` ; pilote : DevTools WebView2 (titre,
  URL) sur le binaire de dev.

### 9.17c L19 — Confort bureau après le gate visuel (S ; décisions utilisateur du 2026-11-02)

Retours de l'utilisateur à l'installation (17.7 et 18.4 passés le 2026-11-02 : premier lancement, connexion par code,
arrêt propre, désinstallation avec données conservées, vérifiés dans les journaux) :

- [x] 19.1 Installeur : après l'extraction, passer seul à la dernière page (crochet NSIS `SetAutoClose true` ou
      équivalent, vérifié sur le modèle NSIS de Tauri extrait du CLI) ; témoin : installation réelle.
      FAIT 2026-11-02 : le modèle extrait du CLI (`@tauri-apps/cli-win32-x64-msvc`) pose `!define
      MUI_FINISHPAGE_NOAUTOCLOSE` avant la page finale (commentaire du modèle : garder la page d'extraction pour le
      débogage) ; MUI2 (`Pages/Finish.nsh`) n'émet alors pas son `SetAutoClose true`, et le modèle ne le pose que pour
      le mode passif (fin de `Section Install`). Correction : crochet `NSIS_HOOK_POSTINSTALL` de `windows/hooks.nsh`
      = `SetAutoClose true` (instruction d'exécution évaluée à la fin de la section, après le crochet ; un échec garde
      la page). Vérifié : `.nsi` généré (`target/release/nsis/x64/installer.nsi`) inclut `hooks.nsh` (l.32) et appelle
      le crochet en fin de `Section Install` (l.3685) ; `e2e-install.ps1` sur l'installeur local : exit 0. Le clic
      réel de la page d'extraction est vérifié par l'utilisateur (19.5).
- [x] 19.2 Liens à nouvel onglet (`target=_blank`, dont le lien de connexion par code `microsoft.com/link`) :
      la fenêtre les envoie au navigateur par défaut (`on_new_window` de Tauri 2.11, même politique que
      `navigation.rs` : hors origine locale et écrans embarqués = navigateur, jamais une seconde fenêtre) ;
      test Rust de la décision ; témoin : clic sur le lien de connexion.
      FAIT 2026-11-02 : `navigation::decide_new_window` (`OpenExternally` / `Deny`, même politique que `decide` : externe
      -> navigateur, local, embarqué, blob, `javascript:`, `data:`, `file:` -> refus), `origin_for_log` (origine seule :
      le lien de connexion porte un code), `external_open_mode` (`LEVELUP_SHELL_OPEN_EXTERNAL=log` remplace l'ouverture du
      navigateur par le journal, pour les essais) ; `main.rs` : `.on_new_window(...)` renvoie toujours `Deny` et journalise
      `event=new_window url=<origine> decision=open|deny` puis `event=open_external origin=...`. **Cause réelle du lien
      muet** : `tauri_plugin_opener::init()` injecte par défaut un script qui attrape le clic sur `target=_blank`,
      `preventDefault()` puis appelle la commande `open_url` par le pont IPC, refusée à la page du serveur (aucune
      capacité) : le clic était avalé et `on_new_window` ne voyait jamais le lien (seul `window.open` y arrivait).
      Correction : `Builder::new().open_js_links_on_click(false)`. Témoin (binaire de débogage, front du poste,
      DevTools WebView2 9333, clic de souris de confiance sur le lien `microsoft.com/link` de `/setup`) : `desktop.log`
      `event=new_window url=https://www.microsoft.com decision=open` + `event=open_external origin=https://www.microsoft.com
      mode=log`, une seule page ; liens `target=_blank` vers `127.0.0.1:<port>`, `levelup-shell.localhost`, `javascript:` :
      `decision=deny` ; vers `example.org/a?code=SECRET` : `decision=open`, `SECRET` absent du journal (`l19/desktop_l192_temoin.log`).
      Avant la correction, mêmes clics : aucune ligne. Tests : 5 nouveaux dans `navigation.rs`.

- [x] 19.3 Référentiels en **panneau ancré** (décision utilisateur : option 1) : en profil bureau (capacité
      `can_use_compact_chrome`), le panneau des référentiels s'ouvre à droite du contenu et le POUSSE (pas de voile,
      la page se réorganise), largeur redimensionnable et mémorisée (`localStorage`), reste ouvert de page en page
      tant qu'on ne le ferme pas, champ de recherche en tête (celui du panneau actuel), bouton « Référentiels » au
      bas de la barre latérale + raccourci clavier ; le site hébergé garde son panneau coulissant actuel.
      FAIT 2026-11-02 : `AppShell` (capacité vraie) range `AssetDockedPanel` dans la rangée `[barre latérale | contenu | volet]`
      (il pousse le contenu : `aside` dans le flux, `shrink-0`, largeur en style, aucun voile) et ne monte plus le panneau
      coulissant ; capacité fausse : `AssetDrawer` et son mini-onglet inchangés, `HelpSplitButton` non touché. Contenu commun
      `AssetPanelContent` (onglets, recherche, `AssetGrid`/`AssetCard`, requêtes) partagé par le panneau du site et le volet
      (`searchFirst` : recherche en tête). Poignée `role="separator"` (souris, flèches, Maj = grand pas, Début/Fin), bornes
      280 à 640 px, borne haute réduite sur fenêtre étroite (`maxPanelWidth` : 240 de barre + 480 de contenu préservés),
      largeur en `localStorage` `levelup-asset-panel-width` (`panelWidth.ts`, lecture/écriture sous `try/catch`, alerte
      unique), état ouvert/fermé persisté par le store existant, focus dans la recherche à l'ouverture demandée
      (`focusToken`, jamais au démarrage), Échap dans le volet le ferme et rend le focus au bouton. Bouton « Référentiels » au
      pied de la barre latérale + raccourci `Ctrl+Maj+L` (`Ctrl+Maj+R` écarté : rechargement sans cache du moteur ;
      `Ctrl+Maj+L` sans accélérateur connu, rien dans la page ne l'utilise) en infobulle fr/en. Témoins sur l'application
      construite (CDP, clics et touches de confiance) : ouverture au bouton (focus dans la recherche), `Ctrl+Maj+L` ferme
      puis rouvre, Maj+Gauche x3 : 360 -> 552 px et `main` 648 px, Droite 536, Fin 640, Début 280, glisser de 140 px : 280 ->
      419 px, mémorisé ; navigation Carrière puis Explorer : volet ouvert, même largeur ; rechargement : volet ouvert, focus
      sur `body` (`l19/s1_ouvert.png`, `s4_page.png`). Tests : `AssetDockedPanel.test.tsx` (13), `panelWidth.test.ts` (6),
      `AppShell.chrome.test.tsx` (volet frère de la rangée, site inchangé), `SidebarNav.test.tsx`.

- [x] 19.4 « Signaler un problème » : entrée au bas de la barre latérale (à côté de Aide), formulaire actuel, GitHub
      s'ouvre dans le navigateur (dépend de 19.2).
      FAIT 2026-11-02 : entrée « Signaler un problème » au pied de `SidebarNav` (`common.shell.nav_feedback`, fr + en),
      ouvre `useFeedbackDrawerStore` ; le mini-onglet du bord du formulaire est masqué en bureau (capacité). Le formulaire
      ouvrait GitHub par `window.open(url, '_blank', 'noopener,noreferrer')` : avec `noopener` le retour est TOUJOURS `null`
      (constaté sur la fenêtre : refus de la nouvelle fenêtre = `null`), donc la branche « fenêtre bloquée » (copie de
      l'URL, formulaire laissé ouvert) s'exécutait à tort, sur le site aussi ; remplacé par un lien `target="_blank"` cliqué
      par programme (`openInNewTab`), qui passe par `on_new_window` ; branche de repli et clé `feedback_drawer.popup_blocked`
      supprimées (code mort). Témoin : formulaire rempli, clic « Ouvrir sur GitHub » : `desktop.log`
      `event=new_window url=https://github.com decision=open` + `event=open_external origin=https://github.com mode=log`.
      Tests : `FeedbackDrawer.test.tsx` (lien cliqué, formulaire refermé, mini-onglet masqué en bureau), `openInNewTab.test.ts`,
      `SidebarNav.test.tsx`.

- [x] 19.5 **(U)** PASSÉ (couvert par les gates utilisateur du 2026-11-03 sur les installeurs suivants : page d'extraction, lien de connexion, volet des référentiels, signalement) — À l'utilisateur (clic réel de la page d'extraction de l'installeur, volet des référentiels, lien de connexion, signalement).
- Gate : web `typecheck + lint + test`, Rust `fmt + clippy + test`, `e2e-install.ps1` sur l'installeur local.

### 9.18 P2 — à planifier à la clôture de P1 (liste fermée des sujets)

Fin de match par détection du processus du jeu (énumération seule), mode jeu (pas de cuisson pendant
Halo), notifications natives relayées, barre des tâches, démarrage réduit avec Windows, tâche planifiée
`refresh-only`, ffmpeg téléchargeable (D-9), rapport de diagnostic, import depuis l'hébergé, option « se
connecter à un serveur », plafond mémoire du processus, CHECKPOINT plus fréquents en profil `desktop`,
**sauvegarde périodique en Parquet vers un dossier choisi** (remplace restic, étude §3.8), **fuseau et
langue tirés du système au premier lancement** (`user_timezone` vaut `Europe/Paris` en dur), poids des
mises à jour (ressources retéléchargées à chaque version).

### 9.19 P2 — découpage VALIDÉ par l'utilisateur le 2026-11-02 (ordre L20 → L21 → L22 → tag → L23 ; TOUT dans la v8.1.0 ; réglages de L21 off par défaut)

P1 clos le 2026-11-02 (L16 à L19 fusionnés, 17.7 et 18.4 passés). La liste fermée ci-dessus est ventilée en quatre lots,
plus les découvertes de P1 qui changent ce que l'utilisateur verra. Pondération : P2 = 10 % du global.

| Lot | Contenu (liste fermée) | Pourquoi dans cet ordre |
|---|---|---|
| **L20 — Prêt pour la première release** (S) | (a) encodeur H.264 configurable : le build ffmpeg LGPL livré n'a pas libx264 ; `media/hls.go` et les rendus qui le demandent échouent sur l'installation de bureau → choix `h264_mf` (Windows) ou `libopenh264` par capacité, avec repli journalisé ; (b) fuseau et langue tirés du système au premier lancement (`user_timezone` vaut `Europe/Paris` en dur ; langue système hors de `locales.toml` = défaut `en`, validé par l'utilisateur le 2026-11-02) ; (c) notes de mise à jour dans la langue de la boîte (fr ET en depuis les deux changelogs) ; (d) échéance sur le téléchargement de l'installeur (le verrou de vérification ne doit pas rester pris) ; (e) toasts et barre de progression au-dessus du volet des référentiels ; (f) `HomeCitationsNearCompletion` pointe sur la route finale ; (g) CHECKPOINT plus fréquents en profil `desktop` (fermeture par la croix = arrêt propre déjà, mais une coupure de courant perd le WAL depuis le dernier CHECKPOINT) ; (h) **en-têtes de page uniformes** (retour utilisateur 19.5 : Ascension a titre, chapeau, sous-titre, Carrière / Communauté / Escouade n'ont rien, la page « saute » au changement de section ; vaut aussi pour le site) : un composant d'en-tête partagé, même hauteur et même espacement sur chaque section L1, titre = libellé de la section, chapeau fr/en par section. | Sans (a), la version de bureau publie des rendus qui échouent ; (b) et (c) sont les premiers défauts qu'un nouvel utilisateur hors France voit ; le reste est petit et déjà localisé. |
| **L21 — Vivre avec le jeu** (M) | (00) **connexion par le navigateur, sans code** (décision utilisateur 2026-11-02, accepte la modification Azure) : flux code d'autorisation + PKCE du site, adresse de retour loopback `http://localhost:<port>/auth/xbox/callback` servie par le serveur local (Azure : redirect URI `http://localhost/auth/xbox/callback` à ajouter par l'utilisateur sur l'application publique partagée, D-19 ; port dynamique admis par l'exemption loopback), aboutissement dans le même point d'arrivée que le code d'appareil (profil suivi, jetons chiffrés, `DeviceCodeCompletion` généralisé), bouton « Se connecter avec Microsoft » qui ouvre le navigateur, page « Connexion réussie, vous pouvez fermer cet onglet », l'app passe seule à l'accueil ; le code d'appareil reste en secours ; JAMAIS de page de connexion dans la fenêtre de l'app ; (0d) **aucun encodeur H.264** (Windows N sans Media Foundation, découverte L20) : message dans l'app (fr/en) qui explique et pointe le Media Feature Pack, rendus vidéo refusés proprement ; (0) **barre latérale repliable en icônes** (décision utilisateur 2026-11-02) : repliage MANUEL (bouton au pied de la barre + `Ctrl+B`, état mémorisé) et automatique sous ~1 100 px de largeur de fenêtre seulement (jamais au défilement), en mode icônes : infobulle au survol et menu volant des onglets pour les sections qui en ont ; icônes **Lucide** (ISC, glyphes retenus par l'utilisateur, copiés en SVG, pas la bibliothèque) : Accueil `layout-dashboard`, Solo `crosshair`, Escouade `swords`, Carrière `award`, Ascension `flame`, Communauté `globe`, Médias `film`, Explorer `search`, Référentiels `book-open`, Signaler un problème `bug`, Aide `book-open-text`, Paramètres `settings`, Thème `sun-moon`, À propos `info`, Replier `panel-left-close` ; les icônes du jeu restent réservées au contenu Halo (règles Microsoft) ; (0b) **capacités bureau de la communauté** : « Classements » masqué par capacité de déploiement, états vides honnêtes là où une page attend d'autres comptes (escouade : décision utilisateur 2026-11-02 = rien d'autre, les stats d'escouade dérivent des matchs communs déjà locaux ; relais VPS écarté) ; (a) détection du processus du jeu par énumération seule (jamais de lecture mémoire) : fin de match déclenche la synchronisation, et **mode jeu** = aucune cuisson de rejeu ni tâche lourde tant que Halo tourne ; (b) notifications natives Windows relayées depuis la cloche (opt-in dans Paramètres) ; (c) icône de la zone de notification : fermer la fenêtre réduit dans la zone (réglage, défaut = quitter), menu Ouvrir / Synchroniser / Quitter ; (d) démarrage réduit avec Windows (réglage, défaut off) ; (e) tâche planifiée `refresh-only` posée par l'app (réglage, défaut off) pour garder les jetons vivants quand l'app ne tourne pas. | Valeur propre à l'app de bureau (le site ne peut pas le faire). (c), (d) et (e) sont des réglages, jamais imposés. |
| **L22 — Données et diagnostic** (M) | (a) sauvegarde périodique en Parquet vers un dossier choisi (remplace restic ; étude §3.8), restauration documentée ; (b) import depuis l'instance hébergée (export côté hébergé par capacité, import au premier lancement ou dans Paramètres) ; (c) rapport de diagnostic (zip : journaux, versions, `app_info`, sans jetons) depuis « À propos » ; (d) plafond mémoire du processus (réglage, défaut = sans plafond, avec compteur). | Protège les 36 à 100 Go de données irremplaçables ; (b) est ce qui fera passer un joueur du site à l'app. |
| **L23 — Poids des mises à jour** (S, après la première release publiée) | Mesurer ce qui est retéléchargé à chaque version (installeur complet de 193 Mo dont ~80 Mo de ffmpeg), décider : ressources inchangées non réembarquées ou installeur différentiel ; `cargo audit` en CI. | Ne se mesure qu'avec deux releases publiées. **Mesure faite le 2026-11-04** sur quatre installeurs 8.1.0 successifs de la CI : [`L23_POIDS_MISES_A_JOUR_2026-11-04.md`](L23_POIDS_MISES_A_JOUR_2026-11-04.md) (ffmpeg = 47 Mo dans l'installeur, pas 80 ; images Halo 5 = 41 Mo ; 82 % du téléchargement identique d'une version à l'autre ; serveur sans `-s -w` : 11 Mo gagnables ; items §9.26). |
| **P3 (hors chantier)** | option « se connecter à un serveur » (D-3) ; ffmpeg téléchargeable (écarté par D-9) ; macOS (D-2). | Décisions déjà prises. |

Ordre proposé : L20 → L21 → L22 → première release signée (tag) → L23. Chaque lot = une PR vers la release, revue adversariale,
gate utilisateur sur l'installeur. Décisions de l'utilisateur (2026-11-02) : ordre validé ; L20, L21 et L22 entrent TOUS dans la v8.1.0 (L23 après le tag par nature) ; réglages de L21 off par défaut.

### 9.20 L20 — items

- [x] 20.a Encodeur H.264 sans libx264 : détection des encodeurs au démarrage (`ffmpeg -encoders`, une fois, en cache), ordre `libx264` -> `h264_mf` -> `libopenh264`, options propres à chacun, `media_tooling` exige « un encodeur H.264 parmi la liste » et journalise le retenu, aucun encodeur = erreur typée et transcodage refusé. Témoin : transcodage réel avec le ffmpeg LGPL du lot. FAIT 2026-11-02 : `media.SelectedH264Encoder` (liste `-encoders` lue une fois par chemin de binaire dans `AvailableEncoders`, essai réel sur une image par candidat, repli journalisé, `ErrNoH264Encoder` et transcodage HLS refusé avant tout lancement) ; ordre `libx264` -> `h264_mf` -> `libopenh264`, `h264_mf` et `libopenh264` à débit cible 12M, GOP 240 et images-clés forcées à la durée du segment (pas de preset ni de CRF) ; `media_tooling` n'exige plus libx264, journalise l'encodeur retenu, `check-env` ajoute une ligne. Témoin : `scratchpad/l20/witness_20a_*.log` (ffmpeg LGPL du lot, `h264_mf` puis `libopenh264`, sortie lue par ffprobe).
- [x] 20.b Fuseau et langue du système au premier lancement (capacité `desktop`, jamais le nom du profil) ; une valeur déjà écrite n'est jamais écrasée ; repli journalisé sur `Europe/Paris`. FAIT 2026-11-02 : capacité `AdoptSystemLocale` (vraie en `desktop` seulement) ; `internal/platform/syslocale` lit le registre (`TimeZoneKeyName`) et `GetUserPreferredUILanguages`, table CLDR 001 de 139 clés Windows vers IANA ; `config.adoptSystemLocale` valide par `time.LoadLocation` (repli journalisé `Europe/Paris`) et pose `UserTimezone` quand app_settings.json n'en porte pas ; `settings.Store.WithSystemDefaults` donne la langue et le fuseau système aux clés ABSENTES (aucune réécriture, une valeur écrite gagne). **Défaut trouvé et corrigé dans ce lot** : le serveur n'embarquait pas la base de fuseaux, donc `time.LoadLocation` échouait sur un poste sans Go (l'application installée) : `time/tzdata` importé dans `cmd/server`. Témoin : `scratchpad/l20/witness_20b_*.log`.
- [x] 20.c Notes de mise à jour dans la langue de la boîte : `latest.json` porte `notes` (FR) + `notes_en` ; la coquille choisit selon la langue. FAIT 2026-11-02 : `latest.json` porte `notes` (FR) et `notes_en` ; le workflow échoue avant compilation et à la publication si l'une des deux sections manque, le `jq` final refuse une clé vide ; la coquille lit `Update::raw_json` et choisit par `update_notes::notes_for_locale` (langue par défaut : `notes`, sinon `notes_<langue>`, `notes_<repli>`, `notes`), codes tirés de `locales.toml` par `build.rs`. 8 tests Rust. Témoin `scratchpad/l20/witness_20c_latest_json.log`.
- [x] 20.d Échéance sur le téléchargement de l'installeur (verrou de vérification rendu) ; décision d'abandon pure et testée. FAIT 2026-11-02 : pas d'échéance du builder sur le téléchargement (`Update.timeout` vaut `None`) ; chien de garde `download_watch` (60 s sans octet, 60 min au total, verdict pur, 7 tests) qui interrompt la tâche, rend l'erreur `download_stalled` ou `download_too_long`, le verrou est rendu et une vérification demandée dit `update_check_failed`. Choix : échéance d'INACTIVITÉ et non totale courte, pour ne pas couper un téléchargement lent mais vivant.
- [x] 20.e Toasts et barre de progression limités à la colonne du contenu en bureau (volet des référentiels non couvert). FAIT 2026-11-02 : en bureau la barre de progression est rendue DANS la colonne du contenu (`relative`, `absolute inset-x-0 top-0`) et les toasts se décalent de la largeur du volet (`useDockedPanelInset`, offset `{top, right}`) ; le site ne change pas (barre fixe `top-12`, offset 56 sans décalage). Tests `AppShell.chrome.test.tsx` et `TopProgressBar.test.tsx`.
- [x] 20.f `HomeCitationsNearCompletion` pointe sur la route finale `career/commendations`. FAIT 2026-11-02 : « Tout voir » de `HomeCitationsNearCompletion` pointe sur `career/citations` ou `career/commendations` (la route `citations` avait le même saut de redirection) ; test de navigation.
- [x] 20.g CHECKPOINT périodiques en profil `desktop` (`desktop_checkpoint_interval`, défaut 5 min), par l'écrivain titulaire, journalisés avec durée, échec compté (expvar) ; garde-rail archlint sur le site unique du CHECKPOINT. FAIT 2026-11-02 : capacité `FrequentCheckpoint` (desktop) et clé `desktop_checkpoint_interval` (défaut 5 min, bornes 30 s à 1 h) ; `duckdb.CheckpointHeld` checkpointe chaque base tenue en RW par le processus (cache de handles, jamais une ouverture propre) sous le bail `dblease` (base sautée et comptée `skipped_busy` pendant un lot d'écriture), échec journalisé WARN et compté (`levelup.checkpoint.periodic`) ; tâche `periodicCheckpointTask` suivie, `shared_social` exclue (déjà couverte) ; ratchet `archlint/checkpoint_single_site_test.go` (27 fichiers de sites existants au 2026-11-02). Test : écriture puis CHECKPOINT, aucun `.wal`.
- [x] 20.h En-têtes de page uniformes (`PageHeader`, huit sections L1, chapeau fr/en par section, hauteur fixe) + garde-rail par layout de section. FAIT 2026-11-02, forme 1 compacte, onglets et actions intégrés, web et bureau alignés, décision utilisateur 2026-11-02 : `components/foundations/PageHeader` (h1 unique 20 px, chapeau 13 px, `min-h-12` sur la zone titre, ligne d'onglets `h-11` RÉSERVÉE même sans onglets et collante quand elle en a ; sous-titre et actions retirés à la revue, onglets filtrés par `capability`, logique pure `pageHeaderLogic.ts`) ; `SectionHeader` posé par le layout joueur pour les sections de `L1_SECTIONS` (titre `common.nav.section_*`, chapeau `common.page_lead.<section>` fr et en, onglets de la section) ; l'accueil garde sa bannière d'identité (elle porte déjà le titre), l'escouade rend son en-tête dans `SquadLayout` (`headerInLayout`, l'onglet Emprise dépend des données) ; `AscensionLayout` ne garde que `TipsTicker`, `NavL2` devient `SoloFiltersBar` (filtres Stats seuls), h1 de pages retirés (Face-à-face, Leaderboard PP, session de Solo en h2). Garde-rail `SectionHeader.test.tsx` (une ligne par section de `L1_SECTIONS`, parité fr/en, chapeaux sans possessif ni pronom). Témoins `scratchpad/l20/witness_20h_offsets.log` et captures : h1 à 36 px et premier bloc à 152 px identiques sur Carrière, Escouade, Ascension, Solo, Communauté (108 px sans ligne d'onglets avant la revue : Médias et Explorer réservent désormais la ligne vide, voir la revue L20 au §10).
- [x] 20.i (U) Gate utilisateur sur l'installeur issu de la fusion. PASSÉ le 2026-11-03.

### 9.21 L21a — items (première moitié de L21, 2026-11-02 ; (a) à (e) : autre session)

- [x] 21.00 Connexion par le navigateur, sans code : capacité `can_login_via_browser` (desktop), `GET /api/v1/auth/xbox/login` rend `{authorize_url}` (code d'autorisation + PKCE S256, `redirect_uri` loopback `localhost:<port>`, `state` lié à la session de la coquille), `GET /auth/xbox/callback` ouvert sans cookie de coquille mais gardé par `state` (usage unique, 10 min) et Host loopback strict, aboutissement dans le point d'arrivée du code d'appareil (`AuthCompletion`), page de retour fr/en, bouton et sondage dans l'assistant, code d'appareil en secours. FAIT 2026-11-02 : capacité LoginViaBrowser (desktop) publiée par /bootstrap (can_login_via_browser) ; GET /api/v1/auth/xbox/login (Huma) ouvre une tentative du registre du code d'appareil liée à la session de la coquille (state 32 octets, PKCE S256, redirect_uri http://localhost:<port lié>/auth/xbox/callback) et rend authorize_url ; GET /auth/xbox/callback ouvert SANS cookie par le seul garde de la coquille (DesktopGate.AllowBrowserCallback : GET, chemin exact, Host localhost ou 127.0.0.1 du port lié) derrière LoopbackOnly, protégé par le state à usage unique de 10 minutes ; échange du code, chaîne XBL/XSTS/Spartan, tentative autorisée remise par le sondage existant à la même stratégie post-flux (DeviceCodeCompletion généralisé en AuthCompletion) ; page de retour fr/en (desktop.toml) ; BrowserLoginPanel dans l'étape de connexion de l'assistant (c'est elle, et non XboxLoginPage réservée au SSO du site, qui sert le bureau), code d'appareil en secours. Tests : handlers/auth_browser_login_test.go (flux complet, Azure simulé, state faux/rejoué/expiré, refus Microsoft, échec d'échange), middleware/desktop_gate_test.go, api/desktop_browser_login_cgo_test.go, ratchet des routes nues, attempt_browser_test.go, BrowserLoginPanel.test.tsx.
- [x] 21.0 Barre latérale repliable en icônes : bouton + `Ctrl+B`, état mémorisé, repli automatique sous 1 100 px sans écraser un choix manuel, infobulles, menu volant des onglets, icônes Lucide copiées (ISC), site hébergé inchangé. FAIT 2026-11-02 : barre latérale repliable en icônes (56 px) : `useSidebarCollapse` (trois états auto / replié manuel / déplié manuel, `localStorage` sous `try/catch`, repli automatique sous 1 100 px par `matchMedia`, jamais au défilement, Ctrl+B), `SidebarRail` (une icône par entrée, infobulle au survol et au focus, menu volant `role=menu` des onglets au survol ou au clic, flèches et Échap, sigle du jeu et initiale du joueur, cloche gardée, pied de barre en icônes et bouton de dépliage), bouton de repliement au pied de la barre dépliée ; 16 glyphes Lucide copiés (ISC) dans `components/icons/lucide.tsx` et mention dans `apps/desktop/THIRD_PARTY_NOTICES.md` (le pied de page web n'a pas de section licences) ; la flamme de `navL1Sections.tsx` est déjà celle de Lucide (même tracé), donc inchangée ; site hébergé inchangé (la barre n'existe que sous `can_use_compact_chrome`). Tests : `SidebarNav.collapse.test.tsx`, `sidebarCollapse.test.ts`, e2e `desktop-sidebar.spec.ts` sur `vite preview` (témoins `scratchpad/l21a/temoins/`).
- [x] 21.0b Capacités bureau de la communauté : « Classements » masqué par capacité, états vides honnêtes fr/en. FAIT 2026-11-02 : capacité de déploiement `CompareAccounts` (`can_compare_accounts`, vraie hors desktop) ; l'onglet « Classements » (`L1Tab.deploymentCapability`, lu par `useVisibleNavSections`) disparaît du `PageHeader`, de la barre latérale et du tiroir, la section Communauté mène à Relations et `/community` y redirige en bureau ; la page Leaderboard PP (adresse seule) affiche un bloc « Cette page compare plusieurs comptes ; l'application de bureau n'en suit qu'un. » fr/en au lieu d'un classement vide. Relations et Face-à-face dérivent des matchs : inchangés. Tests : `useVisibleNavSections.test.tsx`, `SectionHeader.test.tsx`, `LeaderboardPPPage.test.tsx`, `bootstrap_capabilities_profile_test.go`, e2e `desktop-sidebar.spec.ts`.
- [x] 21.0d Message sans encodeur H.264 : état exposé par l'API, bloc fr/en sur la page Médias, lien vers le Media Feature Pack. FAIT 2026-11-02 : `media_tooling.h264_encoder_missing` (verdict définitif de la sonde du démarrage : détection aboutie et aucun candidat utilisable ; un doute n'est pas un verdict) publié par /admin/health et par /bootstrap (`media_h264_missing`, absent quand un encodeur existe) ; `MediaNoEncoderNotice` sur la page Médias : bloc fr/en qui explique l'édition Windows N ou KN sans Media Foundation et pointe le « Media Feature Pack » (`MEDIA_FEATURE_PACK_URL` dans `lib/appLinks.ts`, lien `target=_blank`). Tests : `MediaNoEncoderNotice.test.tsx`, `MediaPage.test.tsx`, `media_tooling_test.go`, `bootstrap_media_h264_test.go`, e2e `desktop-sidebar.spec.ts`.
- [x] 21.z (U) Gate utilisateur : flux Azure réel, barre repliée, classements, message média. PASSÉ le 2026-11-03.

### 9.22 L21b — items (seconde moitié de L21, 2026-11-02 ; (a) à (e) du tableau 9.19)

Décisions utilisateur 2026-11-02 : réglages (c), (d), (e) DÉSACTIVÉS par défaut ; détection du jeu par énumération des processus seule (jamais de lecture mémoire) ; rien d'imposé.

- [x] 21.a Détection du processus du jeu et mode jeu (Go) : `internal/platform/gameproc` (énumération ToolHelp32, noms d'exécutables par titre `process_names`), sondage 15 s sous la capacité `GameProcessWatch`, état exposé et compteur expvar, mode jeu (porte `HeavyWorkAllowed` consultée avant chaque lancement : cuisson des rejeux, audits de santé, balayages, catalogue, classement mondial), fin de partie = synchronisation (anti-rebond 60 s), réglage `game_process_watch` (défaut ON) dans Paramètres fr/en. FAIT 2026-11-02 : `gameproc.Lister` (ToolHelp32 sous Windows via `lister_windows.go`, vide ailleurs ; comparaison insensible à la casse, liste voulue vide = jamais en cours), `Mode` (jauge `game_mode_running`, `HeavyWorkAllowed`, `WaitHeavyAllowed` : le cycle attend la fin de la partie par relevés toutes les 30 s, jamais un kill), `Watcher` (tics 15 s, passage `game_mode: enter` / `leave` journalisé, `OnGameClosed` avec anti-rebond de 60 s, réglage coupé = mode retombé sans synchronisation, énumération en panne = état conservé et panne journalisée une fois, compteurs `game_mode_*`) ; capacités `GameProcessWatch`, `ShellIntegration`, `ScheduledRefresh` (desktop seul, publiées par /bootstrap : `can_watch_game_process`, `can_integrate_shell`, `can_schedule_refresh`) ; `process_names` lu du `title.toml` (Halo 5 : `[]` explicite) et, pour le titre built-in dont le `title.toml` est ignoré (ADR 0025), posé dans `NewRegistry` (`HaloInfinite.exe`) : écart au libellé de la demande, qui supposait un `title.toml` pour Halo Infinite ; `Registry.GameProcessNames` rend l'union triée sans `slug ==` ; porte du mode jeu : `scheduler.runHeavy` devant les quatre boucles lourdes (`data_health`, `asset_name_sweep`, `catalog_refresh`, `world_leaderboard`, garde-rail `heavy_gate_test.go`) et la boucle de cuisson `replayartifacts.buildAll` (film persisté, cuisson reportée au rattrapage ; le cycle de synchronisation lui-même n'est PAS suspendu : réseau et insertions courtes) ; fin de partie : `RequestCatchUp` de l'ordonnanceur d'auto-sync (même chemin que la séquence de réveil ; le déclencheur de présence existant agit match par match jeu ouvert, ce déclencheur agit à la fermeture : pas de doublon) ; réglages `game_process_watch` (défaut ON), `native_notifications`, `close_to_tray`, `start_with_windows`, `scheduled_refresh` (défaut OFF) dans `domain.DesktopSettings`, refus 400 `desktop_setting_not_offered` hors capacité, `GET /settings/desktop` (`game_running`) ; onglet « Application » de Paramètres (capacité) avec la ligne de détection fr/en. Tests : `gameproc_test.go`, `lister_windows_test.go` (énumération réelle), `heavy_gate_test.go`, `cuisson_mode_jeu_test.go`, `boot_gameproc_test.go`, `settings_desktop_test.go`, `config_loader_test.go`, `DesktopIntegrationCard.test.tsx`, `SettingsPage.test.tsx`.
- [x] 21.b Notifications natives (Rust + Go) : réglage `native_notifications` (défaut OFF), événement stdout `notification{title, body, kind}` émis par un relais de la cloche (catégories retenues, texte du catalogue dans la langue du joueur), `tauri-plugin-notification` côté coquille, affichée seulement fenêtre masquée ou sans focus, clic = fenêtre au premier plan. FAIT 2026-11-02 : transport = événement stdout (un seul écrivain, ADR 0045 D-2), plutôt qu'un sondage HTTP que la coquille ne peut pas authentifier (le cookie de la coquille est `HttpOnly`) ; Go : `internal/notifications/native` (`Relay` = `notifications.ExternalForwarder`, branché par `notifications.Fanout(discord, native.Default())` dans `notifServiceFor`, inerte tant que le boot du profil desktop n'a pas installé son sink), catégories retenues `match_synced`, `sync_error`, `challenge_added|completed`, `objective_completed`, `streak_milestone`, `personal_record`, `career_rank`, `skill_tier` (les conseils du coach, médias, amis ne sortent pas), texte composé dans la langue du joueur par six paires de messages `desktop.notify.*` de `desktop.toml` (fr et en, tenus par le catalogue), réglage relu à chaque notification, compteurs `notifications_native_*` ; `desktopEvents.Notification`, `launchShellIntegration` (capacité `ShellIntegration`) ; Rust : `events::Toast` + `SidecarEvent::Notification`, `Action::Notify` (phase prête seulement), `Host::notify`, module pur `native_notice` (`WindowState`, `should_show` : retenue au premier plan, montrée masquée, réduite ou sans focus ; le journal ne porte jamais le texte), `toast_tauri` (plugin `tauri-plugin-notification` 2.4, dernière version compatible `rust-version 1.88`) ; clic : une notification Windows activée relance l'exécutable et le plugin d'instance unique appelle `focus_main` (montre, restaure, focus) : le plugin n'offre aucun rappel de clic sur le bureau, le comportement réel est à constater au gate 21.y ; ligne « Notifications Windows » de l'onglet Application fr/en. Tests : `relay_test.go`, `fanout_test.go`, `boot_shell_test.go`, `desktop_events_test.go` (forme exacte), `native_notice.rs` (5), `lifecycle/tests.rs` (2), `events.rs`, `tests/contract.rs` (2, faux serveur : `FAKE_EVENTS_AFTER_READY`), `DesktopIntegrationCard.test.tsx`.
- [x] 21.c Icône de la zone de notification (Rust) : réglage `close_to_tray` (défaut OFF), événement stdout `shell_settings` (lu au démarrage et à chaque changement), menu Ouvrir / Synchroniser maintenant / Quitter, la croix réduit quand le réglage est actif ; machine d'états testée. FAIT 2026-11-02 : transport des réglages = événement stdout `shell_settings` (un seul écrivain), relu côté serveur toutes les 5 s (`runShellSettingsLoop` : écriture initiale une fois prêt, puis seulement un changement ; PATCH ou édition du fichier, même effet), plutôt qu'un `GET /api/v1/desktop/settings` que la coquille ne peut pas appeler (cookie `HttpOnly`, aucune commande exposée) ; « Synchroniser maintenant » = commande `sync` écrite sur la stdin du serveur (`stdin_commands.go` : lignes avant l'EOF = commandes, l'EOF reste l'ordre d'arrêt) qui lance `AutoSyncScheduler.RunOnceTrigger("manual")`, un cycle à la fois, compté dans le groupe d'attente des tâches de fond ; Rust : `shell_settings.rs` (pure : `ShellState` décide la création, la recréation et le retrait de l'icône, `on_close_requested` rend `HideToTray` seulement si le réglage est actif ET que l'icône existe vraiment, jamais de fenêtre cachée sans icône ; menu Ouvrir / Synchroniser / Quitter, Quitter = séquence d'arrêt propre de la croix ; libellés du catalogue compilé `desktop.tray.*` dans la langue annoncée par le serveur, repli langue par défaut), `Action::ApplyShellSettings` / `Action::SendSync`, `Input::SyncRequested` (serveur prêt seulement), `Closer::request_sync`, `Sidecar::send_command`, `tray_tauri` (fonctionnalité `tray-icon`, clic gauche rouvre, menu au clic droit), `shell_tauri` (exécution sur la fenêtre : la croix cache, le retrait de l'icône ramène la fenêtre) ; ligne « Réduire dans la zone de notification à la fermeture » fr/en. Tests : `shell_settings.rs` (9 : croix, icône, échec de création, langue, menu, libellés), `lifecycle/tests.rs` (3), `events.rs`, `tests/contract.rs` (3, faux serveur : réglages annoncés, `sync` arrivé sur la stdin deux fois sans arrêter le serveur et fermeture propre ensuite, aucune commande avant `ready`), `stdin_commands_test.go`, `boot_shell_test.go` (boucle des réglages, cycle manuel), `desktop_events_test.go`, `desktop_entry_cgo_test.go` (libellés `tray_*` en fr et en), `DesktopIntegrationCard.test.tsx`.
- [x] 21.d Démarrage réduit avec Windows (Rust) : réglage `start_with_windows` (défaut OFF), `tauri-plugin-autostart` (entrée par utilisateur), argument `--minimized`, décision pure testée, aucune entrée réelle créée. FAIT 2026-11-02 : `autostart.rs` (pur : `reconcile(wanted, &dyn AutostartControl)` lit d'abord et n'écrit que pour changer ; réglage coupé sans entrée = aucune écriture, un refus ou une lecture impossible est rendu `Failed` et journalisé, jamais d'écriture à l'aveugle), `autostart_tauri.rs` (le plugin `tauri-plugin-autostart` 2.5, entrée `HKCU\...\Run` de l'utilisateur courant, lancée avec `--minimized`), appliqué à chaque annonce `shell_settings` (donc au lancement et au changement du réglage, depuis Paramètres par le serveur) ; `launch.rs` (pur : `started_minimized` lit argv[1..], `initial_window`), fenêtre créée cachée sous `--minimized`, `ShellState::start_minimized` crée l'icône tout de suite (implique l'icône) puis `on_window_shown` la retire quand `close_to_tray` est coupé ; le plugin de mémoire de fenêtre ne mémorise plus la visibilité (`!StateFlags::VISIBLE`, sinon il remontrerait la fenêtre cachée) ; une seconde ouverture `--minimized` (entrée de démarrage alors que l'application tourne) ne montre rien ; ligne « Démarrer avec Windows » fr/en. Aucune entrée réelle : tests sur un registre simulé, garde de texte (`tests/textes_de_configuration.rs`) imposant que seul `autostart_tauri.rs` derrière `reconcile` écrit l'entrée. Tests : `autostart.rs` (5), `launch.rs` (4), `shell_settings.rs` (4 de démarrage réduit), `textes_de_configuration.rs`, `DesktopIntegrationCard.test.tsx`.
- [x] 21.e Tâche planifiée `refresh-only` (Go) : réglage `scheduled_refresh` (défaut OFF), `schtasks` par `childproc.Command` derrière une interface, création et suppression idempotentes, état vérifié au démarrage et affiché dans Paramètres, mode `refresh-only` vérifié (verrou d'instance). FAIT 2026-11-02 : `internal/platform/schedtask` (`Manager` : `Ensure` = `/Create /TN LevelUp\Refresh /XML <fichier> /F`, `Remove` = lecture puis `/Delete /TN ... /F` (absente = succès sans appel), `Query` = liste CSV `/Query /FO CSV /NH` ; `ExecRunner` lance `System32\schtasks.exe` par `childproc.Command` avec une échéance de 30 s) ; écart au texte de la demande (`/SC DAILY|HOURLY`, `/TR`, `/RL LIMITED`) : définition XML UTF-16 (`/XML`), car `/TR` est borné à 261 caractères que trois chemins de racines dépassent, et seul le XML porte `StartWhenAvailable` (reprise après veille), l'absence de condition de batterie, `IgnoreNew`, `Hidden` et `RunLevel=LeastPrivilege` (= `/RL LIMITED`) ; l'action est `cmd.exe /d /s /c "set LEVELUP_DATA_DIR=... && set LEVELUP_RESOURCES_DIR=... && set LEVELUP_LOGS_DIR=... && levelup-server.exe -mode refresh-only"` (variables de racines) ; un chemin avec `"`, `%`, `&`, `^`, `!`, `<`, `>`, `|` ou retour à la ligne est refusé (`ErrUnsupportedPath`) ; `cmd/server/boot_schedtask.go` (capacité `ScheduledRefresh`) : au démarrage la boucle VÉRIFIE (réglage actif et tâche absente : posée ; coupé et tâche restante : retirée ; sinon rien), ensuite chaque changement pose ou retire, un échec est journalisé, exposé (`GET /settings/desktop` : `scheduled_refresh.state` = present, absent, error, unknown, et le code fixe `schtasks_failed` ou `unsupported_path`, jamais le texte du système) et non retenté à chaque tic ; carte « Garder la connexion vivante application fermée » et état fr/en ; `-mode refresh-only` vérifié : il prend le verrou d'instance avant tout et sort en 0 sans échange quand l'application tourne (`refresh_only_test.go` `TestRefreshOnly_VerrouTenu`, `e2e_refresh_only_test.go`). Tests : `schedtask_test.go` (arguments exacts, XML décodé, idempotence, chemins refusés, liste CSV), `schedtask_realschtasks_test.go` (étiquette `e2e_windows` : le vrai `schtasks.exe` accepte la définition sur une tâche `LevelUp-test-<aléa>` déclenchée dans 48 h et supprimée ; vu passer, aucune tâche `LevelUp\Refresh` créée), `boot_schedtask_test.go`, `settings_desktop_test.go`, `DesktopIntegrationCard.test.tsx`. Limites au §10 (désinstallation, console brève).
- [x] 21.y (U) Gate utilisateur : zone de notification, démarrage avec Windows, tâche planifiée, notification native, mode jeu pendant une partie. PASSÉ le 2026-11-03.

### 9.23 L22 — items (données et diagnostic, 2026-11-03 ; (a) à (d) du tableau 9.19)

Décisions de l'exécutant (2026-11-03, dans le cadre de la demande) : dossier de sauvegarde = champ texte validé par le serveur (pas de sélecteur natif : la page ne peut pas ouvrir de sélecteur et la coquille n'a aucun canal de commande depuis la page, cookie `HttpOnly`) ; import = persister INSERT-only dédié `persist.ImportPersister` (la recette OpenSpartan est typée et lit du SQLite) ; périmètre de l'export = liste fermée, lignes du joueur seulement ; plafond mémoire 0 = valeur par défaut du serveur (512 Mo), pas « sans limite ».

- [x] 22.a Sauvegarde périodique en Parquet vers un dossier choisi. FAIT 2026-11-03 : `internal/ops/parquetbackup` (jeu `levelup-<AAAAMMJJ-HHMM>/` : un Parquet zstd par table de chaque base listée par `ops.BackupDatabases` via `PathResolver`, schéma `_ddl_`, fichiers de configuration et d'auth tels quels, `manifest.json` avec SHA-256 et `excluded: film_archive` ; dossier `.partial-*` renommé à la fin ; rotation limitée aux dossiers `levelup-*` du dossier choisi), lecture par `duckdb.WithHeldRead` (bail d'écriture à attente bornée, même porte que `CheckpointHeld` ; base occupée sautée puis reprise une fois, sinon jeu partiel), exporteur `duckdbbackup.ExportTargetTables` (rangées comptées, apostrophe du chemin doublée), réglages `backup_dir`/`backup_interval`/`backup_keep` (`domain.DataSettings`, `settings/data.go`, validation serveur : absolu, existant, accessible en écriture, hors racine de données, même volume = avertissement), capacité `ParquetBackup` (`can_backup_to_folder`), boucle `cmd/server/boot_parquetbackup.go` (porte `WaitHeavyAllowed`), `GET /settings/data` + `POST /settings/data/backup/run`, expvar `levelup.backup.parquet*`, carte « Données » (`DataCard`), restauration `levelup backup restore --from --to [--dry-run] [--force]` (verrou d'instance, sommes vérifiées, importeur existant), `docs/RUNBOOK_DESKTOP.md` (cas disque mort -> nouveau PC), ADR 0045 D-24. Preuves : `parquetbackup_cgo_test.go` (sauvegarde -> restauration dans une racine temporaire = mêmes lignes, schéma/vue/séquence rendus, jeu altéré refusé, cible non vide refusée, base occupée = jeu partiel, rien exporté = rien gardé, rotation, même minute), `readonly_guard_test.go`, `cmd_backup_restore_cgo_test.go`, `boot_parquetbackup_test.go` (pas de passage pendant une partie), `settings_data_test.go`.
- [x] 22.b Import depuis l'instance hébergée. FAIT 2026-11-03 : hébergé (`ExportForDesktop`, `can_export_for_desktop`) `POST /export/desktop/token` (session, jeton aléatoire 256 bits, 15 min, usage unique, empreinte seule gardée) et `GET /export/desktop` (jeton Bearer ou session ; le joueur est toujours celui du jeton/de la session ; un export par heure et par joueur, 429, journalisé) ; `internal/transfer` (archive zip : manifeste + Parquet de `match_registry` des matchs du joueur, ses lignes de `match_participants`/`medals_earned`/`highlight_events`, `player_match_enrichment` ; liste fermée, jamais de jeton, de compte ni d'autre joueur) ; bureau (`ImportFromSite`, `can_import_from_site`) `POST`/`GET /import/site` (https seul, boucle locale en http pour les essais, aucune redirection, taille et durée bornées, entrées validées contre la liste fermée, sommes vérifiées, xuid de l'archive = joueur suivi) puis `persist.ImportPersister` (INSERT seul, une transaction par base, clé déjà présente jamais réécrite, intersection des colonnes) par `sync.AcquireSharedWriterStandalone` et le bail joueur ; `SiteImportPanel` (Réglages > Application et bloc facultatif de l'assistant, `StepInitialSync`) et `ExportForDesktopCard` (Réglages > Compte du site). Limites : titre par défaut seulement. Preuves : `transfer_cgo_test.go` (export -> import bout en bout sur bases temporaires, idempotence, autre joueur refusé, archive altérée/traversée/trop grosse refusées, jeton à usage unique par HTTP), `token_test.go`, `export_desktop_cgo_test.go`. ADR 0045 D-25.
- [x] 22.c Rapport de diagnostic. FAIT 2026-11-03 : `internal/ops/diagnostic` + `POST /api/v1/desktop/diagnostic` (`DiagnosticReport`, `can_report_diagnostic`) -> `<DataRoot>/diagnostics/levelup-diag-<AAAAMMJJ-HHMM>.zip` (fin de 5 Mo de chaque `*.log` dont `desktop.log`, `app_info.json`, `settings.json` expurgé, `scheduled_tasks.json`, `updater.json`, diagnostic de schéma de chaque base par la porte de lecture du serveur `ops.DiagnoseHandle`) ; deux barrières : clés interdites retirées des documents, `Scrub` sur le texte des journaux (en-têtes, JWT, jetons Microsoft, paires clé=valeur, paramètres d'adresse, valeurs exactes des secrets du processus) ; bouton « Rapport de diagnostic » + « Copier le chemin » dans `AboutDialog`. `[~]` « état de l'updater » : l'état détaillé vit dans la coquille (Rust), le rapport en porte la version et renvoie à `desktop.log` joint. Preuve : `diagnostic_cgo_test.go` (faux jeton, faux secret du lanceur, faux webhook, faux code et faux JWT semés dans journaux, réglages et documents joints : aucune entrée de l'archive n'en contient). ADR 0045 D-26.
- [x] 22.d Plafond mémoire du processus. FAIT 2026-11-03 : réglage `memory_limit_mb` (0 ou 1024 à 32768, capacité `MemoryLimit`, `can_limit_memory`) appliqué au `memory_limit` de DuckDB à chaque ouverture de connexion (`applyDuckSessionInit`), au démarrage avant la première base (`applyBootMemoryLimit`) et à chaud sur les bases ouvertes (`duckdb.ApplyMemoryLimitToOpen`) ; unité MiB ; expvar `levelup.process.rss_mb` (`internal/platform/procmem`, `GetProcessMemoryInfo`) ; « Mémoire actuelle : N Mo » dans la carte « Données ». Preuve : `memory_limit_test.go` (`current_setting('memory_limit')` lu sur des bases temporaires : 4.0 GiB après réglage, retour à 512MB à 0), `settings_data_test.go`.
- [x] 22.z (U) Gate utilisateur : sauvegarde vers un dossier réel, rapport de diagnostic, import depuis le site. PASSÉ le 2026-11-03.

### 9.24 L22b — retours du gate (décisions utilisateur 2026-11-03)

- [x] 22b.1 Retour d'enregistrement de l'onglet Application aligné sur le motif du site (le texte « Enregistré. » sous le formulaire est retiré). FAIT 2026-11-03 : le site n'a pas un motif unique ; retenu celui du plus grand nombre d'onglets (Apparence, Analyse, Notifications/Discord, Données et médias : `SettingsPage.handleChange`), un badge flottant en haut à droite « ✓ Enregistré » 2 s / « ✗ Erreur lors de la sauvegarde » 4 s ; seules la mise en commun du jeton (toast Sonner) et la synchronisation/le rattrapage (toasts de job) font autrement, et ce sont des actions, pas des réglages. Le motif est extrait en un module partagé (`saveFeedbackStore.ts` : état zustand + `flashSaveStatus` ; `SaveStatusBadge.tsx` monté une fois par `SettingsPage`, dont `handleChange` l'utilise aussi : plus de copie), et la carte Données (sauvegarde, mémoire) et la carte Intégration au poste (jeu, notifications, zone de notification, démarrage, tâche planifiée) l'appellent à chaque PATCH ; l'échec montre le badge ET garde le refus rédigé sous le champ (cause précise, absente du motif du site). Les textes `saved` « Enregistré. » de `dataText` et `desktopIntegrationText` sont supprimés (0 code mort ; celui de `filmArchiveText` appartient à l'onglet Données, hors périmètre). Tests : `SaveStatusBadge.test.tsx` (4 : repos, 2 s, 4 s en anglais, remplacement), `DataCard.test.tsx` et `DesktopIntegrationCard.test.tsx` (badge de succès et d'échec, plus de texte « Enregistré. »).
- [x] 22b.2 Rejeux à régénérer après une mise à jour : compteur « artefacts en retard » + file de fond (un film à la fois, jamais automatique, pause en mode jeu, reprise à l'ouverture), bandeau sur la page de rejeu seulement, ligne d'état dans Paramètres > Application. FAIT 2026-11-03 : la « table qui porte la version des artefacts » n'est pas la source : `match_usage_films_latest.artifact_schema` ne dit la version qu'APRÈS une dérivation (un artefact rangé sans résumé en est absent) et ignore les révisions de couche ; la vérité est le FICHIER rangé, lu par `replaybuild.ArtifactDigest` (les mêmes marques que les gardes de la cuisson) : en retard = `!UpToDate()` ET schéma <= celui du binaire (un artefact plus récent n'est pas refait : le garde anti-régression refuserait le résultat), film encore dans l'archive. Go : paquet `internal/replayregen` (`scan.go` décompte, `service.go` file) ; décompte lu en arrière-plan et mis en cache 10 min (`checked=false` tant qu'il n'est pas fait) ; file = UN film à la fois par `ServiceRegistry.RunReplayBuild` (action admin : enfant borné sous le verrou solo, `StoreArtifact`) puis `deriverArtefactRange`, via `wire/registry_replay_regen.go` (jamais une seconde recette) ; relit l'état du film avant de le cuire (idempotent) ; attend la porte du mode jeu avant chaque film (`gameproc.HeavyWorkAllowed`, `waiting_for_game`) ; décodage déjà en cours (`ErrDecodeBusy`, `ErrActionBusy`) = attente et nouvel essai (20 s, 30 fois) ; refus voulus de la cuisson (mode non géré, carte hors catalogue, clé inconnue, film non finalisé) = écarté pour le processus, le reste = `failed` ; `replay_regen_pending` dans `app_settings.json` (`Store.SetReplayRegenPending`, état interne jamais servi ni écrit par PATCH) posé au démarrage et à la reprise, effacé à la fin et à la pause, laissé par un arrêt du serveur : `ResumeIfPending` (boot, `cmd/server/boot_replayregen.go`) relance seulement une file que l'utilisateur avait lancée ; routes `GET /api/v1/desktop/replay-regen` et `POST .../start|pause|resume` (`handlers/desktop_replay_regen.go`, `mountDesktopReplayRegen`) sous la capacité `ReplayRegen` (`can_regenerate_replays`, desktop) ; journal `replay_regen:` + expvar `replay_regen_*`. Web : `components/replay-regen/` (hors de `features/` pour tenir le plafond cross-feature P8.5 : deux features l'utilisent) : `ReplayRegenPanel` UNIQUE (bandeau et carte), `ReplayRegenBanner` monté en tête de la page de rejeu (`replay.tsx`) sous la capacité, visible seulement si des rejeux sont en retard ou la file active, « Plus tard » masque l'offre pour la session (`sessionStorage`, une file lancée reste visible), `ReplayRegenCard` dans l'onglet Application, fr/en. Un rejeu ouvert avec un artefact en retard s'affiche comme avant. `make openapi-gen` + `generate-types` (champ `can_regenerate_replays`). Tests : `replayregen/service_test.go` (décompte, un film à la fois sous `-race`, démarrage jamais seul, idempotent, pause et reprise, mode jeu, reprise après redémarrage, arrêt du serveur, refus voulus, attente d'un décodage), `settings/replay_regen_test.go`, `handlers/desktop_replay_regen_test.go`, `api/desktop_replay_regen_mount_test.go`, `wire/registry_replay_regen_test.go`, `config/deployment_test.go` ; Vitest `components/replay-regen/replayRegen.test.tsx` (phases, bandeau sous capacité et `stale > 0`, « Plus tard », progression, pause et reprise, carte = même composant), `SettingsPage.test.tsx`.
- [x] 22b.z (U) Gate utilisateur : confirmation d'enregistrement de l'onglet Application, bandeau et file de régénération des rejeux. PASSÉ le 2026-11-04.

### 9.25 L22c — finitions avant le tag (2026-11-04)

Décisions de l'exécutant (2026-11-04, dans le cadre de la demande) : le nettoyage de désinstallation est un MODE DU SERVEUR LIVRÉ (`-mode uninstall-cleanup`), en Go (registre `HKCU\...\Run` + `StartupApproved` derrière une interface, `schedtask` existant), parce que le désinstalleur NSIS n'a sous la main que `levelup-server.exe` ; l'entrée de démarrage n'est retirée que si sa commande vise le dossier du programme désinstallé ; le crochet NSIS saute le flux de mise à jour (`$UpdateMode`) ; la preuve de bout en bout passe par un fichier témoin (`LEVELUP_UNINSTALL_WITNESS`, écrit seulement quand la variable est posée), l'e2e ne pose aucune entrée ni tâche réelle ; `cargo audit` tourne sur un job Linux dédié (`desktop-audit`), version épinglée, vulnérabilités bloquantes seulement ; la retenue des refus par règle passe par une dépendance du service (`Deps.RefusedByRule`), lue au registre avec le prédicat de la cuisson.

- [x] 22c.1 Désinstallation propre (entrée de démarrage + tâche planifiée). FAIT 2026-11-04 : `internal/platform/startupentry` (`RemoveForInstall`, `Store`, `CommandTargets` ; `registry_windows.go` sur `HKCU`, `registry_other.go` hors Windows), mode `-mode uninstall-cleanup` (`cmd/server/uninstall_cleanup_mode.go`, `flags.go`, `main.go` : ni configuration, ni base, ni port, ni verrou ; bilan sur stderr ; sortie 0/1), `NSIS_HOOK_PREUNINSTALL` dans `windows/hooks.nsh` (`nsExec::ExecToLog /TIMEOUT=60000`, hors `/UPDATE`, `LEVELUP_DATA_DIR` posée à `%LOCALAPPDATA%\LevelUp` si absente), `e2e-install.ps1` étendu (témoin + racines conservées), runbook corrigé (contournement retiré), ADR 0045 D-28, `docs/COMMANDS.md` + FR. Preuves : `startupentry_test.go`, `registry_windows_test.go` (clé privée aléatoire), `uninstall_cleanup_mode_test.go`, `flags_test.go`, gardes de texte Rust (`le_crochet_de_desinstallation_appelle_le_nettoyage_du_serveur`, `le_serveur_retire_l_entree_de_demarrage_sous_le_nom_que_la_coquille_pose`), installeur local + `e2e-install.ps1`.
- [x] 22c.2 `cargo audit` en CI. FAIT 2026-11-04 : job `desktop-audit` de `ci.yml` (Linux, filtre `desktop`, `cargo-audit` 0.22.2 installé `--locked` à version exacte et mis en cache, `cargo audit --file Cargo.lock`) ; une vulnérabilité fait échouer, les avis « non maintenu / unsound / yanked » (11 relevés, arbre transitif de tauri) restent des avertissements (essai local : sortie 0). `ci.yml` n'est pas un workflow sensible (ni secret ni `contents: write`) : pas d'épinglage par SHA exigé ; `ci_workflows_security_test.go` et `ci_paths_filter_test.go` verts.
- [x] 22c.3 Retrait de la condition datée `feat/desktop-*`. FAIT 2026-11-04 (commit EN DERNIER du lot) : le job `go-test-windows-cgo` ne tourne plus que sur `pull_request` et `release/**` ; commentaire daté retiré ; D-7 du plan, `docs/COMMANDS.md` + FR mis à jour (aucun test d'archlint ne citait la condition). Le pilote joue la voie Windows une dernière fois AVANT ce commit.
- [x] 22c.4 `levelup backfill-killsource` prend le verrou solo en refus immédiat. FAIT 2026-11-04 : `cmd/levelup/cmd_decode_verrou.go` (`prendreLeVerrouDeDecodage`, `verrouDeLaPasse`, `CodeSortieVerrouTenu = 3` (2 = `flag.ExitOnError`, revue V9), `sortirSur`), pris avant l'ouverture de la base sauf `--dry-run` et `--credit-only` (aucun décodage). Le grep `decfilm.Decode|Load` sous `cmd/` ne trouve en processus que cette passe dans `cmd/levelup` (`cmd/killsource` est un outil de diagnostic hors ligne sans base, voir §10) ; `TestDecodageEnProcessus_PrendLeVerrouSolo` tient la liste. Tests : refus immédiat avec détenteur nommé et sortie 2, libération, `--dry-run`/`--credit-only` non refusés, `--online` refusé. ADR 0045 D-29.
- [x] 22c.5 `filmArchiveText.saved` supprimé, le panneau d'archive passe au badge partagé. FAIT 2026-11-04 : `FilmArchivePanel.tsx` (`flashSaveStatus`, l'assistant monte son propre `SaveStatusBadge`), clé `saved` retirée du type et des deux langues, `FilmArchivePanel.test.tsx` (badge de succès, badge d'échec, refus rédigé gardé).
- [x] 22c.6 Rejeux Firefight d'avant le 2026-10-15 écartés d'emblée du décompte. FAIT 2026-11-04 : `replayregen.Deps.RefusedByRule` (`service.go`, `skipFor`), branché dans `wire/registry_replay_regen.go` (`unmanagedModeMatches` : `analysis.SQLModeNonGere`, le prédicat de la cuisson, forme courte). Tests : `TestScan_LesRefusParRegleSontEcartesDEmblee`, `TestScan_RefusParRegleIllisiblesRetombentSurLAncienComportement`, `TestQueryUnmanagedModeMatches_MemePredicatQueLaCuisson`.
- [x] 22c.z (U) PASSÉ le 2026-11-04 — Gate utilisateur : désinstallation avec démarrage Windows et tâche planifiée actifs (l'entrée et la tâche doivent disparaître, les données rester), puis un tag. À l'utilisateur.

### 9.26 L23 — poids des mises à jour (2026-11-04 ; rapport [`L23_POIDS_MISES_A_JOUR_2026-11-04.md`](L23_POIDS_MISES_A_JOUR_2026-11-04.md))

- [x] 23.1 Mesure. FAIT 2026-11-04 : composition de l'installeur par famille (non compressé et part compressée, méthode 7z LZMA d=8m mise à l'échelle du flux NSIS), diff SHA-256 de quatre installeurs successifs (runs 41586437921, 41722766811, 41816350092, 41825436589), relevé des composants ffmpeg requis, options (a) à (g) chiffrées, build `-ldflags "-s -w"` du serveur mesuré (-11 Mo). `cargo audit` en CI de la ligne L23 : déjà livré (22c.2).
- [x] 23.2 (U) Décision. PRISE le 2026-11-04 par l'utilisateur : (f) serveur sans symboles OUI ; (e) images Halo 5 en WebP qualité 85 OUI sous réserve d'un gate visuel ; (a), (b), (c) NON (D-9 maintenue : ffmpeg et ressources livrés, mises à jour complètes) ; (g) bootstrapper téléchargé NON.
- [x] 23.3 Mise en oeuvre des options retenues en 23.2. FAIT 2026-11-04 (f et e ci-dessous) : installeur 202 663 134 octets (193,3 Mo) vers 159 348 542 octets (152,0 Mo), -41,3 Mo (-21 %) ; `e2e-install.ps1` code 0. Gate visuel de (e) à l'utilisateur : `C:\Users\Guillaume\Downloads\webp-halo5-avant-apres\comparer.html` ; s'il refuse, le commit `images Halo 5 en WebP` est retiré (celui de (f) reste).
  - [x] 23.3-f Serveur sans symboles. FAIT 2026-11-04 : `-s -w` ajouté aux `-ldflags` (avec `-X main.version` conservé) de `release.yml` (job `build-go`, tous OS, et job `desktop-build`) et de `build-installer.ps1` ; `Makefile` (`go-api-build`, Linux) et `scripts/desktop-dev.ps1` (binaire de développement) gardent leurs symboles : ce ne sont pas des builds de release. Témoins : serveur 163 138 322 octets (155,6 Mo) vers 98 608 640 (94,0 Mo), `check-version.sh` vert sur le binaire `-s -w` (et rouge sur une mauvaise version) ; panic d'un binaire `-s -w` : noms de fonctions, fichiers et lignes intacts (contrat écrit dans le commentaire du workflow). `TestE2E` compile maintenant avec `-ldflags "-s -w"` (sans `-X main.version`, hors contrat testé) : vert.
  - [x] 23.3-e Images Halo 5 en WebP. FAIT 2026-11-04 (gate visuel de l'utilisateur en attente) : `scripts/convert-halo5-images.mjs` (ffmpeg `libwebp`, idempotent, retire les PNG ; `docs/COMMANDS.md` et FR), 1 257 PNG remplacés, 39,7 Mo vers 9,0 Mo (-30,7 Mo) ; références mises à jour (`SpartanCustomizerModal`, `HomeSpartanIdentityBanner`, `ExplorerTargetIdentityBanner`, `HomeHeroBanner`), garde-rail vitest `halo5Images.guard.test.ts`, test Go `TestSPAHandler_ImagesWebPDeLaPage` (`image/webp`). ÉCART À LA DÉCISION, justifié par mesure : seules les plaques (308, 32,0 Mo vers 4,3 Mo) et le fond (1) sont en qualité 85 ; les emblèmes (320, RGBA) et les vignettes (628) sont SANS PERTE, parce que `recolor.ts` lit R, V et B comme trois zones de couleur et que le sous-échantillonnage des chrominances d'un WebP avec perte fait baver une zone sur l'autre (après recoloration, pixels écartés de plus de 16 niveaux sur 255 : plaques 2,0 % en moyenne et 13,5 % pour la pire, emblèmes 3,5 %, vignettes d'emblème 16 %, contre 0 % sans perte). `teams/` reste en PNG (chemin `/titles/{slug}/teams/{id}.png` partagé avec Halo Infinite). Hors périmètre, non touchés : `libs/himap`, `mapcatalog`, fonds de carte du rejeu, PNG de `halo_infinite/`.

### 9.27 L24 — moteur et paquet de ressources séparés (décision utilisateur du 2026-11-04 ; à exécuter le dimanche 2026-11-08, AVANT le tag `v8.1.0`)

**Objectif.** Qu'une mise à jour ne retélécharge que ce qui change. Mesure L23 : sur 152 Mo d'installeur (après 23.3), le moteur
(serveur, coquille, front compilé) change à chaque version, le reste (ffmpeg, images, sons, référentiel, fonds de carte) presque jamais.
Cible : mise à jour courante ≈ 40 Mo (moteur seul) ; première installation inchangée (tout embarqué, D-9 tenue, application prête à l'emploi).

**Décisions déjà prises (utilisateur, 2026-11-04).**
- D-L24-1 : deux unités versionnées : le **moteur** (livré par l'updater Tauri, installeur NSIS) et le **paquet de ressources**
  (archive signée + manifeste : `ffmpeg/`, `static/`, `web/titles/`, `data/.../reference/`, `metadata-prebuilt.zip`, `config/`).
- D-L24-2 : **hébergé = paquet embarqué dans l'image Docker, JAMAIS téléchargé** ; le serveur hébergé vérifie le manifeste au démarrage et
  refuse de démarrer si la version du paquet est hors de la plage exigée par le moteur (même discipline qu'une migration fatale).
- D-L24-3 : **une seule racine de ressources** (`ResourcesRoot`, ADR 0008 amendée) et **un seul manifeste** pour les deux profils ;
  aucun code de page ne sait d'où viennent les ressources.
- D-L24-4 : le **Dockerfile** profite du découpage par ses couches : ressources copiées dans des couches basses (avant les binaires),
  moteur en dernier ; le VPS ne retire que les couches du moteur à chaque déploiement.
- D-L24-5 : en profil bureau, le paquet n'est retéléchargé QUE si son empreinte change, en arrière-plan, jamais pendant une partie
  (porte du mode jeu), vérifié par signature (même clé que l'updater) et somme SHA-256 avant d'être activé ; l'ancien paquet reste
  actif tant que le nouveau n'est pas complet et vérifié (bascule atomique de dossier), et un échec n'empêche jamais l'application de démarrer.

**Items.**
- [x] 24.1 (2026-11-08) Manifeste du paquet : `resources.manifest.json` (version sémantique du paquet, empreinte par fichier, taille, version minimale
      et maximale du moteur compatible) ; lu par `internal/config` au démarrage (profil-agnostique, capacité `ResourcePack`) ; refus de
      démarrer hors plage (événement `failed{state: resources_incompatible}` en bureau, erreur fatale journalisée en hébergé) ; tests.
- [x] 24.2 (2026-11-08) Version de ressources exigée par le moteur : constante générée au build (comme `main.version`), contrôlée par
      `check-version.sh` ; ratchet : une modification d'un fichier du paquet sans bump de version du paquet fait échouer la CI.
- [x] 24.3 (2026-11-08) CI et release : `release.yml` produit DEUX artefacts : l'installeur complet (première installation, inchangé) et
      `levelup-resources-<version>.zip` + `.sig` ; `latest.json` porte, en plus de l'installeur du moteur, l'URL, l'empreinte et la
      signature du paquet ; l'installeur NSIS de MISE À JOUR (`/UPDATE`) n'embarque que le moteur (deux configurations Tauri :
      `tauri.bundle.conf.json` complet pour la première installation, une variante moteur seul pour l'updater) — **point à vérifier en
      premier** : l'updater Tauri 2.12 ne connaît qu'un seul installeur par plateforme ; si deux installeurs sont impossibles à servir
      proprement, repli : l'installeur de l'updater reste unique mais n'embarque plus les ressources, et le premier lancement après
      installation neuve télécharge le paquet (contredit D-9 pour la seule première installation : à faire valider par l'utilisateur
      AVANT de coder).
- [x] 24.4 (2026-11-08, sauf le bouton de réessai de l'écran de la coquille : [!]) Côté bureau : vérification de l'empreinte du paquet au démarrage et après chaque mise à jour du moteur ; téléchargement en
      arrière-plan sous la porte du mode jeu, vérification signature + SHA-256, bascule atomique (`resources-<version>/` puis pointeur),
      purge de l'ancien après succès ; progression et erreurs fr/en dans Paramètres > Application ; tests (faux serveur de paquets,
      téléchargement interrompu, signature invalide, version incompatible).
- [x] 24.5 (2026-11-08, preuve = job CI `test-deploy-precheck.yml`, docker absent de ce poste) Dockerfile : réordonner les `COPY` de l'étage d'exécution : `config/`, `static/`, référentiel, `web/titles/` AVANT
      `levelup-server`, `levelup`, `web/dist` ; paquet embarqué tel quel (même manifeste) ; mesure : taille des couches tirées par le VPS
      entre deux déploiements avant/après (témoin `docker history`).
- [x] 24.6 (2026-11-08) ADR 0045 (amendement : paquet de ressources, D-9 précisé) et ADR 0008 (une racine, un manifeste) ; runbook ; `COMMANDS.md`
      EN + FR ; plan §10.
- [x] 24.7 (2026-11-08) Revue adversariale (deux relecteurs Opus : chaîne de livraison et signature ; démarrage et compatibilité hébergé/bureau) : 17 constats
      recevables (A1-A4, B1-B4, C1-C3, D x6, dont 2 graves), tous corrigés, chacun par un test vu rouge puis vert ; C3 partiel (voir ci-dessous).
- [x] 24.z (U) (2026-11-09, validé par l'utilisateur : « j'ai testé tout est bon ») Gate : installation neuve, puis mise à jour simulée (deux builds locaux de versions différentes) qui ne retélécharge pas
      le paquet ; puis paquet modifié qui est retéléchargé et activé ; déploiement Docker de la release sur une copie (taille tirée).

**Exécution (2026-11-08) — ce qui diffère du texte ci-dessus.**
- 24.1/24.2 : identité = EMPREINTE (SHA-256 du manifeste trié, `internal/resourcepack`, générateur unique `cmd/resources-manifest`), pas de version ni de plage min/max ni de ratchet « bump oublié » ; le moteur embarque l'empreinte par `-X main.resourcesFingerprint` ; `check-version.sh <v> <serveur> <empreinte>` la relit. Capacités `ResourcePackFatal` (hébergé, démo) et `UpdateApp` (bureau) ; pas de capacité « ResourcePack » : le bureau télécharge sous `UpdateApp`. Sans empreinte embarquée (dev, tests) : aucun contrôle, les suites existantes passent inchangées. Hébergé : empreinte différente ou manifeste absent = refus de démarrer (`failed{resources_incompatible}`, 503 `server_start_failed`) ; bureau : démarre sur le paquet présent + WARN + téléchargement en arrière-plan, paquet ABSENT téléchargé avant de servir puis refus si échec. Contenu du paquet : `config/` et le front compilé restent dans le moteur (§10, L24) ; le paquet hébergé est `static/` + `apps/web/dist/titles` (le référentiel vit dans le volume `./data`).
- 24.3 : trois sorties (`LevelUp_<v>_x64-setup.exe` complet, `LevelUp_<v>_x64-engine-setup.exe` moteur seul, `levelup-resources-<empreinte>.zip` + `.sig`), `tauri.engine.conf.json`, `latest.json` pointe l'installeur moteur et porte `resources` {fingerprint, url, sha256, size, signature}. Point à vérifier : aucun blocage, D-9 tenue (§10). Pas de `resources-<version>/` + pointeur : bascule au démarrage par renommage journalisé (§10 (d)). Le désinstalleur oublie le paquet : `-mode uninstall-cleanup` le retire (`resourcepack.Remove`).
- 24.4 : le SERVEUR télécharge (netguard `resources_pack`, hôtes GitHub en https, porte du mode jeu avant le premier octet, reprise `Range`, SHA-256, signature minisign relue en Go — format `ED` de `tauri signer sign`, prouvé sur une signature réelle —, extraction vérifiée fichier par fichier) ; la coquille (Rust, sans réseau) lit `resources` via `Update::raw_json`, écrit `resources <json>` sur la stdin du serveur (limite de ligne 256 -> 4096) et attend l'événement `resources` `ready`/`current` AVANT de télécharger et d'armer l'installeur du moteur ; échec, serveur arrêté ou délai de 6 h : rien n'est installé. Interface : Paramètres > Application > Ressources (fr/en, progression, erreurs par code, bouton Réessayer sur `/desktop/resources/retry`). **[!] Le bouton de réessai de l'écran `resources_incompatible` de la coquille n'est pas fait** : l'écran dit de relancer LevelUp (le démarrage réessaie le téléchargement) ; un bouton exige que la coquille se relance elle-même, ce qui ne se teste pas sans lancer la coquille (interdit ici hors e2e) ; à décider au gate 24.z.
- 24.5 : stage `pack-builder` (même générateur), serveur compilé EN DERNIER avec l'empreinte, couches du paquet (`static`, `apps/web/dist/titles`, manifeste) AVANT celles du moteur (`config`, `docs`, front, binaires) ; garde de construction dans l'image (manifeste = binaire) ; le job `docker-build` de `test-deploy-precheck.yml` prouve l'égalité et l'ordre par `docker history` et imprime la taille des couches. **Non jouée localement** (docker absent) ; la mesure « couches tirées par le VPS avant/après » reste à relever sur ce job et au gate 24.z.
- 24.6 : ADR 0045 D-30 (+ coût de la section Conséquences), ADR 0008 A-8, `docs/RUNBOOK_DESKTOP.md`, `docs/COMMANDS.md` + FR, CHANGELOG EN/FR, §10.
- **Mesures** (build local, empreinte `0e9d1bb3...cc3d`, témoin `scratchpad/l24/tailles.txt`) : installeur COMPLET 159 512 252 octets (152,1 Mo), installeur MOTEUR SEUL 31 675 133 octets (30,2 Mo), paquet `levelup-resources-<empreinte>.zip` 152 545 053 octets (145,5 Mo, deflate). La mise à jour courante passe de 152 Mo à 30,2 Mo (cible du plan : ~40 Mo).
- **Preuve exécutée** : `e2e-install.ps1 -EngineInstaller` (installation complète, paquet conforme au manifeste : 2 526 fichiers, 3 lancements avec arrêt propre, mise à jour moteur seul qui rend le fichier du moteur supprimé et ne rend PAS le fichier du paquet supprimé, désinstallation qui retire 2 525 fichiers du paquet et le dossier du programme entier) : code 0.

- **24.7 (2026-11-08) — corrections de la revue**, commits `fix(desktop L24.7)`. A1 (grave) : le journal de bascule porte un état `committed`, écrit d'un bloc et forcé sur le disque après la dernière unité basculée et avant toute suppression ; `Recover` finit en avant une bascule validée et ne défait que l'autre ; test = coupure du processus avant CHAQUE opération de la séquence (journal, renommages, validation, ménage), paquet cohérent et utilisable au démarrage suivant à chaque fois. A2 : un ménage qui échoue après validation (ancien paquet, dossier extrait, marqueur, journal) est un WARN repris au démarrage suivant, `Activate` rend nil. A3 : chaque erreur de retour arrière est journalisée, un retour arrière incomplet garde le journal, `Activate` refuse sur un journal non résolu, `Settle` n'active rien après une reprise en échec (`Outcome.RecoverErr`), `Purge` ne touche plus au journal ni à l'ancien paquet. A4 : `Sync` de chaque fichier extrait, du manifeste, du marqueur et du journal (`durable.go`). B1 (grave) : `Manager.Settle` relit le paquet installé après l'activation (états `active` et `restart_needed` justes, paquet absent : `current`) ; tests dans l'ordre réel du démarrage (`bootResourcePack`), deux variantes. B2 : la coquille réarme la minuterie de démarrage sur chaque événement `resources` (downloading, verifying) dont la phase ou le pourcentage avance, avant `ready` seulement ; l'événement porte `percent` côté coquille ; faux serveur `FAKE_PROGRESS_FOR_MS` ; tests de contrat (téléchargement plus long que le délai : pas de kill ; bloqué : délai) ; ADR 0045 D-30 mis à jour. B3 : une demande pour un autre paquet annule la préparation en cours (événement d'échec pour elle) et se lance à sa suite : toute demande aboutit à un événement terminal de son paquet. B4 : `Start` réserve et pose la phase avant de rendre la main, `Retry` rend la préparation lancée, un second essai reçoit 409 ; mock web aligné. C1 : `build-installer.ps1` passe le contenu de la clé dans l'environnement, une seule voie comme la CI (vu en échec avec `tauri signer sign`, puis en réussite). C2 : redirection au lieu de `cat |` (SC2002 : 1 avant, 0 après). C3 **partiel [!]** : le job exige désormais une copie de l'empreinte HORS des paramètres de compilation (plus d'occurrences de l'empreinte seule que de `-X main.resourcesFingerprint=<fp>`), joué sur deux binaires jetables ; le binaire ne RELIT pas sa valeur, faute d'un mode serveur qui l'affiche (en ajouter un touche `flags.go`, le garde-rail de la sortie standard et `COMMANDS` EN/FR : hors périmètre, §10). D : tests de somme à taille égale, fichier absent de l'archive, commentaire de confiance altéré, retour arrière d'une bascule échouée, `newResourcesClient` (hôte hors liste, http clair, trop de redirections), appel de `bootResourcePack` dans `bootConfigure` (garde-rail par AST : verrou < paquet < outillage) ; chaque test a été vu échouer avec le contrôle retiré (mutation temporaire, restaurée). Non rejoués : `e2e-install.ps1` et la construction d'un installeur complet (interdits ici), le job docker (docker absent de ce poste).

**Gates.** `go test` (config, cmd/server, sync), `-tags=e2e_windows`, `cargo test`, vitest, `check-version.sh`, build installeur complet +
moteur seul, `e2e-install.ps1` étendu (mise à jour sans paquet), `docker build` local + `docker history`, CI verte, revue.

**Hors périmètre.** Différentiel binaire du moteur (l'updater Tauri n'en a pas) ; ffmpeg réduit (option b de L23, écartée) ; macOS/Linux.

## 10. Découvertes (notées, non traitées)

- **L24 (2026-11-08) — point à vérifier 24.3 : lu dans le modèle NSIS réel (`src-tauri/target/release/nsis/x64/installer.nsi`, celui du build local du 2026-11-04) et dans `tauri-plugin-updater` 2.12.0. Verdict : AUCUN blocage, D-9 tenue, deux installeurs possibles, pas de téléchargement au premier lancement.**
  (a) **Un installeur sans ressources, lancé avec `/UPDATE`, laisse intacts les fichiers de ressources déjà installés : OUI.** Le plugin lance l'installeur avec `/UPDATE` (`updater.rs:893`) et rien d'autre ; dans le modèle, `PageLeaveReinstall` répond « In update mode, always proceeds without uninstalling » (`Goto reinst_done`, ligne 331) : l'ancien désinstalleur n'est PAS rejoué (le label `reinst_uninstall` n'est atteint en mode mise à jour que pour une migration depuis un MSI), la section `Install` ne fait que copier les `File` qu'elle embarque. Ce qu'elle n'embarque pas n'est ni supprimé ni réécrit. Le commentaire de `hooks.nsh` qui disait « il rejoue l'ancien désinstalleur avec /UPDATE » était faux : corrigé. Preuve exécutée : `e2e-install.ps1 -EngineInstaller` (supprime un fichier du moteur et un fichier du paquet avant la mise à jour ; le premier revient, le second non, les autres sommes sont intactes).
  (b) **Le désinstalleur réécrit par l'installeur moteur seul oublie les fichiers du paquet : OUI, un crochet est nécessaire.** `WriteUninstaller` est dans la section `Install` de l'installeur moteur seul, dont la section `Uninstall` ne contient que des `Delete` des fichiers qu'il embarque (ffmpeg, `static`, `web/titles`, référentiel : absents), puis `RMDir` sans `/r` : le dossier du programme resterait. Le crochet existe déjà (`NSIS_HOOK_PREUNINSTALL` lance `levelup-server.exe -mode uninstall-cleanup`, hors mise à jour) : le mode retire désormais le paquet (`resourcepack.Remove` : les fichiers que `resources.manifest.json` liste, le dossier de travail `.pack`, les dossiers vides, rien d'autre ; bilan `ressources=<n>-fichiers` dans le témoin), tenu par un test et par l'e2e (le dossier du programme doit disparaître ENTIÈREMENT après une mise à jour moteur seul).
  (c) **Où vit le paquet téléchargé : sous la racine de ressources, jamais sous la racine de données.** Les fichiers actifs sont à leur chemin habituel sous `%LOCALAPPDATA%\Programs\LevelUp` (là où l'installeur complet les pose) ; le paquet téléchargé attend dans `<racine de ressources>\.pack\` (archive partielle, extraction vérifiée, marqueur « prêt », journal de bascule). Le désinstalleur le retire (b) sans jamais toucher `%LOCALAPPDATA%\LevelUp` (A22 : la sentinelle de l'e2e le prouve toujours).
  (d) **Comment le serveur sert `web/` dont `assets` vient du moteur et `titles` du paquet : il n'a rien à faire.** Le paquet occupe les MÊMES chemins relatifs que l'installeur complet (`web/titles/`, `static/`, `data/titles/*/reference/`, ffmpeg à la racine) : sur disque, `web/` est la réunion de `web/assets` + `index.html` (moteur) et `web/titles` (paquet), `WebDistDir` reste `<racine>/web`, aucun lecteur n'apprend où sont les ressources (295 lecteurs de `RepoRoot` : une racine de paquet séparée en aurait touché la plupart). Écart au §9.27 : pas de `resources-<version>/` avec un pointeur ; la bascule est faite AU DÉMARRAGE (avant qu'un seul fichier du paquet soit ouvert) par renommage unité par unité (`static`, `web/titles`, `data/titles/<slug>/reference`, chaque fichier isolé, le manifeste en dernier) avec un journal qui permet de la défaire après un arrêt brutal ; l'ancien paquet reste actif jusqu'à ce démarrage.
  **Écarts au §9.27, tous décidés par le pilote le 2026-11-08 ou conséquences directes** : identité du paquet = empreinte au build (pas de version à bumper : pas de ratchet « bump oublié ») ; `config/` et le front compilé restent dans le moteur ; hébergé, le paquet de l'image est `static/` + `apps/web/dist/titles` (le référentiel vit dans le volume `./data` de la prod, jamais dans l'image : il n'a donc pas de manifeste hébergé) et son empreinte diffère de celle du bureau (ffmpeg Windows, référentiel) : un format, un générateur (`cmd/resources-manifest`), un lecteur, deux ensembles ; la coquille ne télécharge rien (le serveur le fait : hôtes de publication, signature minisign relue en Go, SHA-256, mode jeu), elle lit le bloc `resources` de `latest.json` (`Update::raw_json`) et attend l'événement `resources` `ready` avant d'armer l'installeur du moteur ; le bouton « Réessayer » de l'écran `resources_incompatible` de la coquille n'est pas fait (voir 24.4).
  **Découvertes L24.7, non traitées** : (a) `check-version.sh` lit les mêmes paramètres de compilation que l'ancien `grep` du job docker (`go version -m`) : un `-X` sur une variable renommée y passe aussi ; un mode serveur qui affiche l'empreinte embarquée (`-mode resources-fingerprint`, avec la table `flags.go`, le garde-rail de sortie standard et `COMMANDS` EN/FR) fermerait les deux ; (b) `Downloader` n'a aucun délai d'inactivité sur le CORPS d'une réponse : un transfert bloqué n'échoue que par la minuterie de la coquille (sans coquille, au lancement direct, il bloque le démarrage) ; (c) `tauri signer sign` avertit « Signing without an app version » : si l'updater passe un jour `requireSignedVersion`, il faudra `--app-version` sur la signature du paquet ET de l'installeur ; (d) `ResourcesBoard` ne retient que le DERNIER événement : un événement tardif d'un autre paquet après le `ready` du bon masquerait ce `ready` (l'ordre actuel des événements ne le produit pas, B3 les sérialise) ; (e) l'événement `waiting_game` ne réarme pas la minuterie de démarrage (voulu : rien n'avance), le démarrage ne lance jamais de préparation sous la porte du jeu (`PrepareNow`) ; (f) entre le dernier renommage et l'écriture de la validation, une coupure défait la bascule (état cohérent, reprise au démarrage suivant) : l'invariant « aucune coupure ne défait » vaut à partir de la validation.
  **Découvertes L24, non traitées** : (1) en mode mise à jour le modèle NSIS n'efface pas les fichiers de la version précédente que la nouvelle n'embarque plus : les `web/assets/*-<hash>.js` d'une version restent après chaque mise à jour (et après la désinstallation, `RMDir` échoue sur le dossier non vide) : antérieur à L24 (vrai pour L17 à L23), ~1,3 Mo par mise à jour ; le nettoyage de désinstallation pourrait retirer `web/assets` en entier, à décider ; (2) `Makefile` / `desktop-dev.ps1` compilent le serveur sans empreinte (aucun contrôle de paquet : voulu) ; (3) le téléchargement du paquet ne peut être joué de bout en bout que contre une vraie release publiée (hôtes GitHub) : au gate 24.z.

- **L23.3 (2026-11-04) — relevé, non traité** :
  - **Plaques en qualité 85 : fines rayures altérées.** Sur la pire des 308 (n° 101, rayures dorées très fines) 13,5 % des pixels s'écartent de plus de 16 niveaux après recoloration ; qualité 90 ou 95 n'y change presque rien (12,3 % et 11,6 %), la cause est le sous-échantillonnage des chrominances (propre au WebP avec perte). Seul le gate visuel de l'utilisateur tranche ; repli = plaques sans perte (environ 22 Mo au lieu de 4 Mo).
  - **PNG de `halo_infinite/` (6,6 Mo) et logos `teams/`** non convertis (hors décision) ; `teams/` partage son chemin avec Halo Infinite, il faudrait une extension par titre.
  - **`Makefile` (`go-api-build`, Linux) et `scripts/desktop-dev.ps1` (binaire de développement)** gardent leurs symboles : ce ne sont pas des builds de release.
  - **Les builds `-s -w` ne se lancent pas sous un débogueur symbolique** (delve) ; les traces de panique restent complètes (vérifié sur un binaire d'essai).

- **L22c (2026-11-04) — relevé, non traité** :
  - **`cmd/killsource`** (outil de diagnostic, `decfilm.Decode` sur un dossier de chunks, sans base ni réseau) ne prend pas le verrou solo : outil de recherche lancé à la main sur UN film, hors du périmètre « commande de `levelup` ». À prendre si l'utilisateur l'utilise pendant que l'application cuit.
  - **La désinstallation n'a pas été jouée contre un vrai registre `Run` ni un vrai planificateur** : l'e2e ne pose rien (trop risqué sur le poste) ; le registre est exercé sur une clé privée aléatoire, `schtasks` par un exécuteur simulé. À constater au gate 22c.z.
  - **`$UpdateMode` dans le désinstalleur** : variable du modèle NSIS de Tauri 2 posée par `/UPDATE` ; le garde `${If} $UpdateMode <> 1` compile (installeur local construit) mais le flux de mise à jour réel n'a pas été joué.
  - **Le décompte de régénération lit le registre une fois par décompte et par titre** (une requête sur `match_registry`) ; à mesurer au gate 22c.z sur les vraies données si besoin.

- **L22b (2026-11-03) — relevé, non traité** :
  - **Rejeux Firefight d'avant la décision du 2026-10-15** : comptés « en retard » jusqu'à leur premier refus (`ErrModeNonGere`), le décompte ne lisait pas le registre ; `levelup backfill-replay --purge-non-geres` les retire. TRAITÉ le 2026-11-04 (L22c.6) : le décompte les écarte d'emblée.
  - **Le panneau d'archive des films (onglet Données) gardait son propre texte « Enregistré. »** (`filmArchiveText.saved`) : TRAITÉ le 2026-11-04 (L22c.5), il passe au badge partagé.
  - **Le décompte lisait chaque rejeu rangé** (~1 à 2 Mo) : CORRIGÉ le 2026-11-03 (revue U8) : il lit les petits fichiers `<match>.meta` ; seul le premier décompte d'un parc d'avant ce fichier lit les artefacts, une fois (à constater au gate 22b.z sur les vraies données, jamais par une cuisson).

- **L22 (2026-11-03) — relevé du lot Données et diagnostic, non traité** :
  - **L'installeur ne livre PAS l'outil `levelup`** (seul `levelup-server` est en `externalBin`) : `levelup backup restore` ne se lance donc, sur un PC neuf, que depuis un dépôt (Go + compilateur C). Le cas « disque mort -> nouveau PC » du runbook le dit. TRANCHÉ ET FAIT le 2026-11-03 (revue S, décision pilote D-a) : `levelup-server.exe -mode restore --from --to [--dry-run] [--force]` (même fonction `parquetbackup.RunRestoreCommand` que `levelup backup restore`, devenu un alias ; runbook et `COMMANDS.md` EN+FR corrigés).
  - **Clé `backup_interval` partagée** avec le planificateur restic hébergé (durée Go, défaut `6h`) : la validation desktop (heures entières, 1 h à 168 h, vide = 24 h) ne porte que sur les clés qu'un PATCH écrit et la lecture ne réécrit jamais la valeur. Un `app_settings.json` hébergé gagne `backup_keep` (7) et les trois autres clés à sa prochaine écriture, sans effet faute de capacité.
  - **Boucle périodique non déclarée dans `taskPolicies`** : comme les boucles de L21 (jeu, tâche planifiée), elle est lancée par capacité (`ParquetBackup`), pas par la table des tâches (dont le test impose « hosted et dev lancent tout »).
  - **La sauvegarde tient le bail d'écriture de chaque base pendant son export** (un export de `shared_matches` peut durer des dizaines de secondes sur une grosse base : une écriture de synchronisation attend ce temps) ; durée à mesurer au gate 22.z sur les vraies données. CORRIGÉ le 2026-11-03 (revue S3) : le bail est pris et rendu PAR TABLE (`duckdbbackup.ExportTargetTablesHeld`), échéance de 2 minutes par table (dépassement = jeu partiel).
  - **Le document OpenAPI ne porte pas les routes sous capacité** (`/desktop/diagnostic`, `/import/site`, `/export/desktop*`, comme `/desktop/strings`) : leurs types web sont écrits à la main (`DiagnosticReport`, `SiteImportRequest`, `SiteImportStatus`, `ExportToken`). Revue T5 (2026-11-03) : un profil de documentation qui monte ces routes a été essayé puis ABANDONNÉ : le routeur de contrat est comparé au contrat dans les deux sens par six garde-rails (routes nues, routes documentées et enregistrées, document partagé, absence de la connexion par le navigateur hors desktop) ; y faire entrer des routes sous capacité demande de séparer le routeur de contrat du routeur de démonstration. Commentaires rendus vrais et types portés par `MANUAL_CAPABILITY_TYPES` (daté, décroissant) dans `lib/api/response-types.guard.test.ts`.
  - **Export hébergé : le filtre `xuid = '<chiffres>'` suppose le xuid stocké en chiffres seuls** dans `match_participants`, `medals_earned`, `highlight_events` (vérifié sur des fixtures, pas sur la base hébergée réelle) : à constater au premier export réel, côté site.
  - **Titre par défaut seulement** pour l'export/import (le fournisseur de la base partagée n'existe que pour lui, comme l'import OpenSpartan) ; Halo 5 n'est pas couvert. Les lignes de participants des AUTRES joueurs d'un match ne sont pas exportées (lecture stricte de « jamais les autres joueurs ») : la page d'un match importé montre le joueur seul jusqu'à la resynchronisation. CORRIGÉ le 2026-11-03 (revue S1, décision pilote D-b) : un match est exporté ENTIER (tous les participants, événements, journal des morts, rangs, stats d'objectif ; ADR 0045 D-25).
  - **Allowlist datée ajoutée** : `archlint/no_shared_writer_on_request_path_test.go` (+1 `internal/service/site_import.go`, import lancé par l'utilisateur, desktop seul).
  - `ops.DiagnoseDB` ouvre toujours en lecture seule forcée (CLI, serveur arrêté) ; le rapport utilise `ops.DiagnoseHandle` (handle prêté).
  - La mémoire du processus n'est pas plafonnée au niveau du système (décision de la demande) ; `memory_limit_mb` = 0 laisse les 512 Mo par défaut de chaque base, pas « sans limite ».


- **L21b (2026-11-02) — relevé de la seconde moitié de L21, non traité** :
  - **TRAITÉ le 2026-11-04 (L22c.1, ADR 0045 D-28) : mode `-mode uninstall-cleanup` appelé par `NSIS_HOOK_PREUNINSTALL`, hors `/UPDATE`.** Constat d'origine : la désinstallation laisse l'entrée de démarrage (`HKCU\...\Run`, valeur `LevelUp`) et la tâche planifiée `LevelUp\Refresh`** quand leurs réglages sont actifs : un crochet NSIS (`NSIS_HOOK_PREUNINSTALL`) ne s'est pas pu tester ici (le modèle de Tauri rejoue le désinstalleur dans le flux de mise à jour, il faudrait garder `/UPDATE`). Contournement documenté (`docs/RUNBOOK_DESKTOP.md`) : couper les réglages avant de désinstaller. À traiter avec un essai d'installeur réel.
  - **Console brève possible au passage de la tâche planifiée** : elle lance un programme console (`cmd.exe` puis `levelup-server.exe`) sous le compte interactif ; la tâche est `Hidden` mais Windows peut montrer une console un instant. À constater au gate 21.y ; si elle gêne, le mode `refresh-only` pourrait masquer sa console au démarrage.
  - **Clic sur une notification** : `tauri-plugin-notification` n'offre aucun rappel de clic sur le bureau ; l'activation relance l'exécutable et le plugin d'instance unique appelle `focus_main`. Comportement réel non constaté (aucune coquille lancée dans ce lot).
  - **`HaloInfinite.exe` et le `title.toml` du titre built-in** : la demande disait `process_names` dans `config/titles/halo_infinite/title.toml`, qui n'existe pas (le descripteur built-in fait foi, ADR 0025) ; le nom est posé dans `NewRegistry`, Halo 5 déclare `process_names = []`.
  - **La synchronisation d'un cycle automatique n'est pas suspendue pendant une partie** (réseau et insertions courtes) ; seuls la cuisson des rejeux et les quatre boucles lourdes attendent. À revoir si la mesure en partie montre une gêne.
  - **Archlint** : `lister_windows.go` et son test sont inscrits à `goosSuffixAllowed` (adaptateur Win32, daté 2026-11-02) ; deux littéraux accentués évités dans `settings.go` (résumé d'opération sans accents) pour ne pas relever le plafond de `no_french_label_literal_test.go`.

- **L21a (2026-11-02) — relevé de la première moitié de L21, non traité** :
  - **Risque à jouer au gate 21.z (flux Azure réel)** : le serveur de bureau n'envoie AUCUN secret (`SPNKR_AZURE_CLIENT_SECRET` absent, D-19) à l'échange du code. Si l'application Azure `e1cb35ab` n'accepte le code d'autorisation que comme client confidentiel (plateforme « Web »), Microsoft répondra `AADSTS70002` : la page d'échec du navigateur affichera `code_exchange_failed` et le journal `auth_browser_login: échange du code échoué` le détail. L'utilisateur a ajouté l'URI `http://localhost/auth/xbox/callback` à l'application publique : le test réel dit si la plateforme « client public / mobile et bureau » suffit. Le code d'appareil reste le secours.
  - Le jeton de rafraîchissement de ce flux est de la famille Azure (comme le SSO du site) ; `onRotated` ne persiste pas `client_family` à l'écriture. À regarder au premier rafraîchissement réel (`ExchangeRefreshTokenWithRotation` essaie Azure puis le client natif).
  - La demande disait `XboxLoginPage` ; c'est l'étape de connexion de l'assistant (`StepDeviceCode`) qui sert le bureau, `XboxLoginPage` étant réservée au SSO du site (`can_sign_in_with_xbox`, faux en bureau). Le panneau est dans `features/setup/BrowserLoginPanel.tsx`.
  - `GET /api/v1/auth/xbox/login` est un GET qui crée un état serveur (voulu par la demande). Il n'est atteignable qu'avec le cookie de la coquille (Host strict) et ne rend qu'une adresse ; une tentative en attente est remplacée au clic suivant, les plafonds du code d'appareil (20 au total, 3 par IP) s'appliquent.
  - La page de retour parle la langue de la page qui a lancé la connexion (le navigateur externe n'envoie pas `X-LevelUp-Locale`) ; un `state` inconnu ou rejoué tombe sur la langue par défaut.
  - La route cachée `community/prestige` (Leaderboard PP) n'a aucun lien dans la navigation ; en bureau elle dit « Cette page compare plusieurs comptes… ». Aucune autre page de la communauté n'attend d'autres comptes (Relations et Face-à-face dérivent des matchs).
  - `features/asset-drawer/panelWidth.ts` gardait `SIDEBAR_WIDTH = 240` pour borner le volet des référentiels : barre repliée (56 px) la borne haute était prudente de 184 px. Traité par la revue P (P8) : la borne lit la largeur réelle de la barre (`lib/layout/sidebarWidth`).
  - La flamme d'Ascension de `navL1Sections.tsx` a le même tracé que `flame` de Lucide : rien à remplacer.
  - `TestFetchPrivacy_BudgetExceededMemorized` (`internal/service`) a échoué une fois pendant que le gate web tournait en parallèle (attente de 103 ms contre un budget de 40 ms) ; vert seul, trois fois. Sensible à la charge, sans rapport avec le lot.
  - Un verdict « aucun encodeur H.264 » atteint APRÈS la sonde du démarrage (un transcodage qui ne trouve aucun encodeur) n'est publié qu'au démarrage suivant ; la sonde du démarrage couvre le cas de la machine Windows N.
  - Les glyphes Lucide ont été copiés de `lucide-static@0.552.0` (lecture HTTP de jsDelivr, aucune dépendance npm ajoutée).

- **C2 : le témoin T2 « avant » était trompeur (corrigé le 2026-11-01 à la clôture de L1)** : sa recette filtrait
  `logs/migration.log` sur l'heure seule (`HH:MM`), sans la date, et a ramené 8 lignes ERROR de
  `purge_data_health_warning_notifs` (bug ART #23645 sur `shared_social`) datées du 2026-05-20, pas du jour.
  Relecture du journal : cette étape a échoué en mai 2026, le serveur a démarré quand même (c'est bien C2),
  puis elle a été appliquée le 2026-06-25. Au 2026-11-01 la base locale est à jour (version 236, 0 étape en
  attente) et le serveur L1 démarre en 12 s (`t2_apres_l1`). Aucun index corrompu à réparer, aucune
  vérification VPS nécessaire. Recette T2 à rejouer avec un filtre sur la DATE à la comparaison de fin de vague.

- **Arrêt dans les premières secondes après le démarrage = `clean=false` (2026-11-01, gate L2 sur les données du
  poste)** : la phase `scheduler` de l'arrêt (3 s) expire quand la demande arrive pendant le rafraîchissement des
  jetons au démarrage (`oauth_refresh` enchaîne les joueurs, chaque échange ~1 s, et le suivi de présence continue
  après `watcher_daemon: arrêté`) : ces goroutines du `schedulerWG` n'honorent pas l'annulation pendant un
  échange HTTP. À 75 s après le démarrage : `clean=true` en 156 ms, aucun WAL. Hors périmètre de L2 (fichiers
  `platform/auth/refresh_loop.go`, `watcher`) ; à traiter avant L12 (la coquille fermée aussitôt ouverte doit
  rendre `stopped clean=true`) : item à ajouter à L12 ou lot dédié à la vague 2.
- **`make replay-equiv` : le film `11de8353` diffère de sa référence (2026-11-01, gate de fin de vague 1)** sur
  `spawnPoints` (attendu 0, obtenu 54), `spawnPointsState` (10 / 11) et `artifact` ; REPRODUIT à l'identique sur le
  checkout `release/v8.1.0` (`1628f75fff`, décodeur non touché par la vague 1) avec les mêmes données du poste :
  écart préexistant, dépendant des données de carte (points d'apparition) et non de ce chantier. Les 19 autres films
  sont identiques. À re-figer par le chantier qui a changé les points d'apparition, pas ici. Le gate-push joué dans un
  worktree sans `data/` échoue sur cette étape : la jouer avec `-repo-root <dépôt principal>`.
- **Chantier publication des briques fusionné dans `release/v8.1.0` le 2026-11-01** (`e2c449b38c` libs/haloapi + ADR 0043,
  `5c37ca56a3`, `1628f75fff`) : L5 et L7 ne sont plus bloqués ; la branche est rebasée dessus avant la PR.
- **Capture des plantages perdue au premier démarrage sur une racine neuve (2026-11-01, L6)** : `crash_output` ouvre
  `logs/server.crash.log` avant que `logs/` existe (créé plus tard par le handler multi-modules) : `WARN crash_output:
  ouverture fichier échouée`, aucune capture jusqu'au second démarrage. À corriger avec L11 (deux racines : `LEVELUP_LOGS_DIR`
  créé avant tout écrivain) ou en lot dédié.
- **L9.1 — Archive des films : 96 % des chunks sont stockés BRUTS, pas en zlib (mesure du 2026-11-01, scratchpad `l9/`)**.
  Recensement des 43 508 chunks de `data/cache/film_chunks` : 41 993 bruts, 1 515 zlib (en-tête `78`, les plus
  anciens) ; `decfilm.Inflate` accepte les deux (premier octet ≠ `0x78` = brut). Les 20 films les plus récents
  (607 chunks, 526 Mo) sont tous bruts : zstd-3 les ramène à 129 Mo (−75 %), zstd-19 à 120 Mo (−77 %) ; temps CLI
  zstd 1.5.7 : 3,4 s (niveau 3) contre 21,7 s (niveau 19) pour les 20 films. Sur les 40 seuls chunks zlib
  comparables : zstd-3 −23 % et zstd-19 −26 % par rapport au zlib. Conséquence : l'archive (36 Go au 2026-11-01)
  tiendrait en ~9 Go en zstd-3 ; décision à prendre pour L14 (D-18 : stockage zstd-3 des nouveaux chunks à la
  réception + recompression des chunks existants par une CLI lancée par l'utilisateur, jamais automatique ; lecture
  transparente dans `decfilm`/`source`). Tables complètes : `l9/table91_A.csv`, `table91_B.csv`, `cache_headers.csv`.
- **L9.2 — Formats de sons (10 sons, 3 boucles moteur `vehicle_scorpion_idle`, `vehicle_warthog_loop`,
  `vehicle_chopper_loop` ; WAV PCM 16 bits stéréo 48 kHz)** : total 8 419 492 o WAV → AAC 128k 715 378 (8,5 %),
  Opus 96k 426 471 (5,1 %), FLAC 2 910 701 (34,6 %). Boucles : Opus et FLAC rendent EXACTEMENT le nombre
  d'échantillons du WAV après décodage (le pré-skip Opus de 312 échantillons est absorbé par le décodeur) ; AAC
  rend 128 à 960 échantillons de TROP selon le son (amorce de 1 024 échantillons + remplissage de fin) : une boucle
  AAC a un raccord audible. Décodeur mesuré : ffmpeg 8.0.1 seul (pas le moteur du navigateur). Table :
  `l9/sons_table.csv`. D-6 : tranchée sur cette table (voir §5).

- **L7 (2026-11-01) — relevé de l'extraction, non traité** : `resolveCSRSeasonFromDB` (`boot_providers.go`) rend `""`
  sans journal si la requête échoue (règle 3) ; la constante `serverWriteTimeout` et son littéral restent dans
  `main.go` parce que `internal/api/wire/build_queue_writer_budget_test.go` lit `cmd/server/main.go` par son chemin ;
  logique métier logée dans les fichiers de démarrage, à sortir de `cmd/server` un jour : repli de jetons
  multi-utilisateur du watcher (`loadWatcherTokens`), fenêtre de rétention des rejeux (closure de
  `launchReplayPurgeCron`), rappels du pool de jetons (`poolCallbacks`, relèvent de `platform/auth`) ; un commentaire
  déplacé tel quel cite `.ai/archive/V7.5/DECOUVERTES_TACTIQUE_2026-09-07.md` (`boot_crons.go`) : relève du nettoyage
  des citations `.ai/` du BACKLOG. Témoin T2 rejoué par le pilote sur l'état FINAL (L7c) seulement, pas après chaque
  sous-lot : général 114 / 111 lignes, écarts = `janitor: fetch_cache purgé` (2 dossiers purgés au premier lancement,
  rien au second), deux lignes `oauth_refresh` (repli MSA natif selon la réponse d'Azure) et l'ordre de `disk_watch`
  (goroutine) ; migration 92 / 92 identiques ; T3 identique octet pour octet.

- **Revue adversariale 2r (2026-11-01, deux relecteurs Opus en contexte frais, L1 + L5 + L7)** : A (DuckDB / auth) :
  0 défaut sur les écritures et les jetons (16 conditions tiennent), 2 constats P2 corrigés dans le lot : la reprise du
  verrou refaisait la copie de sauvegarde à chaque essai (désormais une copie par base et par version et par
  démarrage), une copie interrompue par un arrêt brutal restait sous son nom définitif (désormais `.part` renommé à
  la fin, restes retirés par la rétention). B (extraction / tests) : extraction = déplacement pur confirmé (14
  conditions tiennent) ; 2 constats P1 corrigés : la garde de version lit `len(canonicalOrder)` qui BAISSERAIT à un
  squash et ferait refuser le démarrage de toutes les installations (`schema_newer`) → ratchet
  `migration/order_floor_test.go` (plancher 236, ne descend jamais ; un squash exige d'abord une version explicite
  et monotone : décision à prendre AVANT le prochain squash, ADR à écrire) ; la porte des jetons hors SSO ne testait
  que le titre par défaut avec le filtre du suivi live (synchro en pause ou profil `halo_5` seul → jeton jamais
  écrit) → `AppConfig.HasDeclaredProfile` (tout titre, pause et auth_only compris), porte du suivi live inchangée ;
  1 constat P2 écarté : `-mode desktop|refresh-only` refusé « jusqu'à L12/L15 » est la livraison en séquence
  décidée en D-15 / L2.1, pas un flag « pour plus tard ». Ronde 2 : relecture des seules corrections, non lancée
  (4 corrections de quelques lignes couvertes par des tests ; à relancer si l'utilisateur le demande).

- **L10 (2026-11-01) — relevé** : T3 `xbox` identique à avant hors `capabilities.deployment` et les 13 `can_*` neufs ;
  T3 `none` identique une fois `available_titles` TRIÉ : l'ordre de cette liste varie d'un lancement à l'autre
  (`BuildAvailableTitles` sur le registre, lot 0 `halo_5, halo_infinite`, après L7 et L10 `halo_infinite, halo_5`),
  préexistant et hors périmètre (ordre non déterministe à figer un jour par le rang du registre). Trois capacités de
  plus que l'ADR 0044 D-3 (`can_sign_in_with_xbox`, `can_edit_settings`, `can_upload_media`), `auth_mode` publié
  `none` en desktop, `deployment` sous `capabilities` : écarts documentés dans l'ADR (« Implementation notes »).
  Colonne `desktop` des tâches de fond : seul le classement mondial est coupé (scraping sur le jeton du joueur) ;
  `can_use_admin=false` en desktop, L13 peut le basculer. Allowlists du ratchet `deployment_single_resolver_test.go`
  (83 `.DemoMode`, 52 `.AuthMode`, 3 `LEVELUP_ENV` dans 23 fichiers) et de `demo_state_guard_test.go` à recompter à
  la fusion avec L11.

- **L8 (2026-11-01) — relevé** : `static/sounds` 84 Mo → 4,9 Mo (184 `.ogg`, `-check` 184/184 à l'échantillon près) ;
  devDependency `opus-decoder` 0.7.12 (MIT) pour mesurer le PCM dans les garde-rails Vitest (`ogg-opus-decoder` écarté :
  tire `codec-parser` LGPL) ; `libs/film/replay/weapon_burst_research_test.go` (tag `research`) lit encore des
  `hinf_*.wav` qui n'existent plus : instrument mort, à supprimer (règle 7) hors de ce lot ; Safari ne lit l'Ogg Opus
  qu'à partir de 18.4 (de mémoire, non vérifié) : risque accepté en D-6 ; l'absence de raccord des boucles est
  mesurée par ffmpeg, pas par un navigateur : l'écoute 8.4 de l'utilisateur tranche.

- **L11 (2026-11-01) — relevé et écarts acceptés par le pilote** : pont transitoire `title.InstallProcessRoots` (les
  ~100 services qui reçoivent `repoRoot string` et bâtissent leur résolveur obtiennent les racines du processus
  installées par `config.load()` ; `NewPathResolver(racine)` les rend quand `racine` est la racine de ressources
  installée) — documenté dans l'ADR 0008, s'éteint quand l'allowlist du ratchet `RepoRoot` (149 lectures dans
  91 fichiers) est vide ; `reference/generated/*` (cartes Forge écrites pendant que le serveur tourne) sous
  `DataRoot`, le reste de `reference/` sous `ResourcesRoot` (affine A-2) ; pas de montage en lecture seule du
  référentiel en prod pour cette raison ; Docker : le service prod pose `LEVELUP_DB_PROFILES` / `LEVELUP_APP_SETTINGS`
  pour garder les fichiers montés sous `/app` (A-3 déplacerait `db_profiles.json` sous `/app/data`), la démo pose
  `LEVELUP_LOGS_DIR=/app/data/demo/runtime/logs` → **à relire avant le prochain déploiement prod (push main =
  déploiement)** ; sans `LEVELUP_REPO_ROOT`, les journaux vont sous `<racine détectée>/logs` et plus sous le `logs/`
  du répertoire courant ; `data/cache/career-rank-image/` (images versionnées dans git mais servies par le cache
  d'assets) reste sous `DataRoot/cache` : absentes sur une installation à racines distinctes tant que l'installeur
  (L17) ne les copie pas, sinon retéléchargées ; `LEVELUP_WEB_DIST` reste une variable explicite. Hors périmètre :
  `cmd/replay-corpus-gate` pose `LEVELUP_REPO_ROOT` pour une racine jetable mais pas `LEVELUP_DATA_DIR` (ses enfants
  écriraient dans les vraies données si la variable est posée) ; une quinzaine d'outils `cmd/` ont leur copie de
  `loadEnvLocal` et une trentaine bâtissent `data/titles/...` à la main, hors des garde-rails. Table chemin → racine :
  scratchpad `l11_table_finale.tsv` (998 lignes). Fusion L10 + L11 : un conflit (`config.load` : signature
  `launchMode` de L10 + `ResolveRoots()` de L11), résolu à la main ; `config.go` reste à 629 lignes.

- **Revue adversariale 3r (2026-11-01, deux relecteurs Opus, L10 + L11)** : D (profil, accès, non-régression hébergée) :
  0 défaut, 14 conditions tiennent (table de vérité de `AuthDisabled` identique hors desktop, `LEVELUP_AUTH_MODE=desktop`
  posé à la main sur un serveur hébergé = démarrage refusé, colonnes `hosted`/`dev`/`demo` des tâches identiques,
  équivalences front, champs OpenAPI neufs `required` sans casse de client ancien). C (racines, chemins, pont) :
  2 constats corrigés dans le lot — P1 : l'enfant de cuisson `-film-child` est intercepté AVANT `config.Load`, le pont
  n'était jamais installé dans l'enfant qui écrivait ses faits de film sous `<Resources>/data` et cherchait l'overlay
  des socles au mauvais endroit (`config.InstallProcessRootsFromEnv()` appelé par les deux interceptions, test) ;
  P2 : `levelup check-env` / `healthcheck` lisaient `db_profiles.json` et `app_settings.json` par le PathResolver et
  non par les chemins résolus (`LEVELUP_DB_PROFILES` / `LEVELUP_APP_SETTINGS` du compose prod) → faux « introuvable »
  après déploiement (`HealthcheckOptions.DBProfilesPath/AppSettingsPath`, test). 16 conditions tiennent, dont : aucun
  volume Docker orphelin, démo inchangée, tests non contaminés par le pont. Remarque sans défaut : `taskAppRelease`
  est désormais annulé à l'arrêt (sous `schedulerCtx`), changement annoncé par l'ADR 0045 D-4.

- **L12 (2026-11-02) — relevé** : témoin du pilote sur les données du poste en `-mode desktop` : `listening` à 26 ms,
  `ready` à 10,1 s, arrêt 1 s après `ready` → `stopping` puis `stopped{clean:true}` 150 ms plus tard (12.6 tenu :
  l'arrêt dans les premières secondes est propre), relancement sur le MÊME port (`data/desktop/port`), 0 WAL, aucune
  ligne d'événement sur stderr. Cause réelle de l'ancien `clean=false` : résolution « eager » des comptes de
  `worldenrich.BuildHaloSource` sous `context.Background()` (un échange Microsoft par compte, dans le cron du
  classement mondial suivi par `schedulerWG`) et boucle de rafraîchissement du watcher hors `schedulerCtx` ; corrigé
  par propagation du contexte, sans attente bornée, les écritures de jeton obtenues avant l'annulation terminent.
  `taskAppRelease` désormais suivie par `schedulerWG` (elle écrit dans les bases joueur et pouvait finir après
  `CloseAll`). Écarts à l'ADR 0045 consignés dans l'ADR : verrou pris AVANT le port ; état `failed{state:"port"}`
  ajouté ; `stopped` émis après le rendu du verrou ; pas de `Sync` sur stdout (tube Windows bloquant). Hors
  périmètre : `pool.refresherLoop` détaché sous `context.Background()` (fenêtre d'une milliseconde pour une rotation
  coupée) ; un arrêt demandé pendant l'initialisation attend la fin de l'initialisation (porte de démarrage).

- **Revue adversariale 4r (2026-11-02, trois relecteurs Opus : E mode desktop + réveil, F archive + zstd, G coffre)** :
  E : 26 conditions tiennent, 3 constats corrigés (P1 : la sortie des enfants de cuisson était recopiée sur stdout du
  serveur, là où la coquille lit les événements → stderr + ratchet « aucun `os.Stdout` / `fmt.Print*` dans
  `internal/` » ; P2 : la route d'arrêt HTTP bloquée par le garde du cookie → exemptée, loopback + secret gardés ; P2 :
  la porte de démarrage servie à tout `Host` → `StrictHost` 421 sur la porte en desktop, sans cookie). F : 16
  conditions tiennent (aucune collision de l'en-tête zstd sur 43 508 chunks réels, golden `source` byte-identique,
  `--dry-run` sans écriture), 5 constats P2 corrigés (manifeste corrigé par chunk + reprise réparatrice ; cible par
  défaut d'`archive-move` admise sans film ; déplacement atomique par film via `<id>.part/` ; changement de dossier
  à chaud sur archive vide par `title.SetFilmArchiveRuntime`, hooks de cuisson et post-sync dynamiques ; une
  surveillance disque par VOLUME via `diskfree.SameVolume`). G : 16 conditions tiennent (aucun RT perdu ni écrasé,
  fichiers hébergés identiques à l'octet), 2 constats P2 corrigés (`watcher_tokens.json` scellé par le même
  chiffreur ; un seul avertissement par fichier scellé et par processus). Hors périmètre consigné : un fichier scellé
  illisible (dossier déplacé sur un autre PC, autre compte Windows) = joueur « sans jeton » sans aucune bannière → à
  prévoir (P2 ou L17 : bannière « jetons illisibles sur ce poste, reconnectez-vous »). Ronde 2 non lancée (corrections
  testées une à une).

- **L16 (2026-11-02) — relevé et gate du pilote** : revue H (Opus) : 17 conditions tiennent, 2 constats P2 corrigés
  (secret d'entrée désormais à USAGE UNIQUE côté serveur et chargé une seule fois côté coquille — il reste inscrit,
  mort, dans l'historique WebView2 `EBWebView\Default\History`, aucune interface wry pour l'effacer sans vider le
  stockage que A15 protège : documenté ADR 0045 D-3 ; rotation de `desktop.log` sans perte quand le fichier est tenu).
  Constat du pilote reproduit sur le poste (DevTools distant de WebView2, `--remote-debugging-port`) : la fenêtre finissait
  sur `/` en 401 `desktop_unauthorized` avec le cookie `SameSite=Strict` (navigation d'entrée initiée depuis la page
  embarquée `levelup-shell://`, autre site pour Chromium : cookie non envoyé sur la chaîne 302) → `SameSite=Lax`, test
  figé. Gate visuel : utilisateur (premier lancement : assistant puis matchs chargés) + pilote (deux lancements
  consécutifs : titre « LevelUp - Accueil », fermeture par `WM_CLOSE` → `stopped{clean:true}`, 0 WAL, 0 processus).
  Piège de dev : `tauri dev` redémarre l'application dès qu'un fichier `src-tauri` change (un agent qui édite pendant un
  essai de l'utilisateur tue le serveur en cours de migration) : pour les essais, lancer `target/debug/levelup-desktop.exe`
  avec l'environnement du script, jamais `-Tauri` pendant qu'un lot Rust est ouvert. Hors périmètre : `LEVELUP_WEB_DIST`
  sans défaut desktop (L17 tranche l'emplacement du front installé) ; `on_new_window` pour `target=_blank` ; `cargo audit`
  10 avis « unmaintained » sur des crates tiers.

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

- **L17 (2026-11-02) — décisions, mesures et relevé.** **Poids** : l'installeur NSIS produit en local pèse
  202 211 522 octets (192,84 Mio) ; l'updater télécharge ce même `.exe` en entier (aucune mise à jour différentielle) : 192,8 Mio
  par mise à jour. Contenu avant compression LZMA : serveur 160,8 Mo, ressources assemblées 281,6 Mo (dont ffmpeg et ses DLL
  165 Mo, `static/` 29 Mo, référentiel `data/titles` 45 Mo, front, config), coquille 4,9 Mo. **`LEVELUP_WEB_DIST`** : défaut
  `<ResourcesRoot>/web` posé par `config_desktop.go` quand le profil est desktop et la variable absente (la variable
  l'emporte ; test `TestLoad_DesktopWebDistParDefaut`). **Emplacement du programme** : Tauri 2 installe en `currentUser` sous
  `%LOCALAPPDATA%LevelUp`, c'est-à-dire LA RACINE DE DONNÉES par défaut (constaté sur l'installeur construit) : le crochet
  NSIS `src-tauri/windows/hooks.nsh` ramène ce seul défaut à `%LOCALAPPDATA%ProgramsLevelUp` (D-14) ; le désinstalleur de
  Tauri ne supprime que ses fichiers (un `Delete` par fichier, `RMDir` sans `/r`), prouvé par `e2e-install.ps1` (la racine de
  données jetable survit). **ffmpeg** : BtbN/FFmpeg-Builds `autobuild-2026-10-31-12-59`,
  `ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip`, SHA-256 `5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab`
  (celui du `checksums.sha256` de la release), épinglé dans `fetch-ffmpeg.ps1` ; `ffplay.exe` écarté. `data/cache/career-rank-image` NON livré
  (17 Mo par mise à jour ; le serveur les cherche sous la racine de données et les retélécharge à la demande). **Clé de
  l'updater** : paire générée hors dépôt, clé publique dans `tauri.conf.json` et `.gitleaks.toml` ; un secret GitHub ne
  pouvant pas être vide, `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` est facultatif (clé générée sans mot de passe) ; la clé privée du
  gate a un mot de passe vide : à REMPLACER par une clé protégée avant la première release publique (17.6, pilote).
  **Relevé, non traité** : (1) le build LGPL n'a PAS libx264 (`--disable-libx264`, `--enable-libopenh264` présent) alors que le serveur
  le déclare requis (`ops/media_tooling.go`, avertissement au démarrage) et le demande pour le transcodage HLS
  (`internal/media/hls.go:601`, `-c:v libx264`) et d'autres rendus : ces chemins échouent sur l'installation de bureau tant que le choix
  de l'encodeur n'est pas rendu configurable (openh264 ou `h264_mf`) ; (2) l'installeur étant non signé, l'updater n'est
  protégé que par sa clé minisign : tout est dans `latest.json` + `.sig` ; (3) la boîte de mise à jour utilise les boutons
  personnalisés de `rfd` (contrôles Windows v6, manifeste par défaut de tauri-build) : non vue à l'écran, la mise à jour réelle
  (téléchargement signé, arrêt, installeur) ne peut se jouer qu'avec deux releases publiées : à vérifier par le pilote en 17.6 ;
  (4) les notes de mise à jour sont en français seulement (section FR du changelog) quelle que soit la langue de la boîte ;
  (5) `cargo audit` n'est toujours pas en CI (ADR 0045 D-18) ; (6) un secret d'entrée mort reste dans l'historique WebView2 :
  inchangé (L16) ; (7) en CI les caches cargo du job `desktop-build` pèsent la cible de release entière : à surveiller (limite
  de 10 Go du dépôt) ; (8) un push sur `release/**` lance maintenant 2 jobs Windows lourds (~15 à 25 min) en plus de ci.yml.

- **Revue adversariale L17 (2026-11-02, deux relecteurs Opus) : 6 constats recevables (3 P1, 3 P2 retenus), 0 P0, corrigés.**
  (C1, P1) l'installeur de mise à jour ne part plus que si `stopped` est arrivé avec `clean=true` ET que le serveur n'a
  pas été tué : `updater::install_allowed` (fonction pure sur l'issue du superviseur, qui porte maintenant `kill_failed` ;
  `Host::exit_app` reçoit cette issue) ; sinon événement `update_skipped_unclean_stop reason=killed|kill_failed|unclean|no_stopped`
  dans `desktop.log`, boîte native (`update_skipped_unclean`, fr et en, catalogue de repli et route `/desktop/strings`) et sortie
  sans installer ; faux serveur : option `FAKE_STOP_CLEAN=false` ; tests `tests/updater.rs` (arrêt non propre ; serveur tué
  au budget) + test unitaire de la décision ; commentaire de `updater.rs` et phrase de l'ADR 0045 corrigés.
  (C2, P1) commentaire inversé de `desktop.toml` réécrit : les libellés de la boîte sont servis par la route ET présents dans le
  catalogue de repli compilé (la boîte peut s'afficher serveur arrêté). (C3, P2) un échec de `Update::install` est journalisé
  (`update_install_failed`) et montré dans une boîte native (`update_install_failed`, fr et en) ; le test de l'installation
  refusée vérifie le message. (C4, P1) `e2e-install.ps1` : vérifications préalables AVANT le `try/finally` (dossier, clé de
  registre NSIS `HKCU\Software\lvelup\LevelUp` et clés de désinstallation, processus en cours) avec refus code 1 sans aucun
  nettoyage ; le `finally` ne tue que les processus lancés par le script (arbre) et ne désinstalle que ce qu'il a installé ;
  témoin : dossier factice, clé factice et faux `levelup-server.exe` en cours, script sorti 1, rien tué ni retiré.
  (C5, P2) sentinelle `e2e-sentinelle.txt` posée dans `%LOCALAPPDATA%\LevelUp` avant l'installation, exigée après la
  désinstallation (l'assertion sur la racine jetable reste). (C6, P2) `cmd/updater-verify` (vérification minisign du `.sig` contre
  `plugins.updater.pubkey`, tests sur les deux algorithmes) et étape de `release.yml` juste après « Installeur et signature
  présents » ; joué sur l'installeur et le `.sig` du lot (succès, puis échec avec un octet modifié).
  **Relevé, non traité** : le fichier `.sig` de Tauri est le base64 du fichier minisign ENTIER (comme la clé publique de
  `tauri.conf.json`), pas le fichier minisign nu ; la clé `HKCU\Software\lvelup\LevelUp` (valeur par défaut = dossier
  d'installation) subsistait sur ce poste sans installation réelle (reste d'un essai antérieur) : le contrôle préalable l'accepte
  seulement quand elle désigne le dossier par défaut et que ce dossier est absent.

- **L18 (2026-11-02) — relevé et témoins.** Application construite (`scripts/desktop-dev.ps1`, `cargo build`, front
  `npm run build`) lancée sur `%LOCALAPPDATA%\LevelUp-dev` avec DevTools WebView2 (port 9333) : titre « LevelUp - Accueil »,
  URL `http://127.0.0.1:59184/fr/t/halo_infinite/players/JGtm/home` AVANT et APRÈS le clic « Rechercher une mise à jour »
  (identiques, la page ne bouge pas) ; `desktop.log` : `[shell] event=update_check_unavailable` à chaque clic (build de
  débogage : updater absent) ; boîte native « Mise à jour de LevelUp » lue par UI Automation : « La recherche de mise à jour
  n'est pas disponible dans ce build de développement. », fermée par `WM_CLOSE`, second clic : la boîte revient ; fermeture
  par `WM_CLOSE` : `stopped clean=true`, 0 WAL, 0 processus. Les issues « à jour », « échec » et « proposée » ne sont jouées
  que par les tests (`tests/update_manual.rs`, faux `UpdateSource`) : un build de débogage n'a pas d'updater, la vraie
  réponse de `latest.json` est affaire du pilote (17.6) sur un build de release. **Relevé, non traité** : (a) `target="_blank"`
  (lien « Règles d'usage… » de `MicrosoftNotice`, repris dans « À propos ») : la fenêtre ne le traite pas (`on_new_window`
  absent, déjà au relevé L16) : les liens du projet de « À propos » sont posés SANS `target`, la navigation passe par
  `navigation.rs` (navigateur par défaut) ; ce seul lien Microsoft garde `target="_blank"` ; (b) la version affichée est
  celle du serveur (`-ldflags -X main.version`) : `dev` sur le binaire de développement, `8.1.0` sur l'installeur ; (c)
  la première boîte native du premier lancement de la session de test n'a pas été retrouvée par l'énumération des fenêtres
  du processus (le journal avait bien la ligne) ; rejouée une seconde fois, la boîte est apparue à chaque clic.

- **Revue adversariale L18 (2026-11-02, un relecteur Opus) : 6 constats recevables (3 P1, 3 P2 retenus), 1 jeté (voile), 0 P0,
  corrigés.**
  (C1, P1) « À propos » : Échap et Tab ne tenaient plus quand le focus quittait la fenêtre (clic sur le chemin des données,
  bouton de mise à jour qui se désactive : `activeElement` = `body`) ; le clavier est écouté sur `window` (modèle
  `alert-dialog`), Tab ramène le focus dans la fenêtre quand il en sort, et le focus passe au titre (`tabIndex=-1`) quand le
  bouton se désactive ; 4 tests dont Échap et Tab sur `document.body`, chacun en échec sans la correction. (C2, P2) la
  vérification de mise à jour n'avait pas d'échéance : `UpdaterBuilder::timeout(CHECK_TIMEOUT)` de 30 s (constante nommée) sur
  la vérification, et la décision du verrou (`run_launch_check`) rend le verrou par garde RAII y compris sur panique
  (`catch_unwind` en test ; en release `panic = "abort"` ferme de toute façon la coquille). (C3, P2) Halo 5 : l'onglet Carrière
  réécrit visait la route de redirection `commendations` : il vise la route finale `career/commendations`, donc l'onglet est
  actif (test `SidebarNav` avec les capacités Halo 5 ; test `NavL1` aligné). (C4, P1) clé morte `common.about.dialog_aria`
  supprimée (fr et en), manifests régénérés. (C5, P1) `main.rs` 517 -> 374 L (`update_setup.rs`) et `updater.rs` 821 -> 456 L
  (`updater_tests.rs`, `updater_decision.rs` : `SkipReason`, `install_allowed`, messages d'après-arrêt, re-exportés par
  `updater`) ; `cargo test` identique (81 + 13 + 7 + 9) avant les tests ajoutés. (C6, P2) trous de tests comblés : vérification
  du lancement sautée quand le verrou est pris, `!enabled` mène à `Unavailable` (`ensure_available`), l'aiguillage
  `Route::CheckUpdate` (`render::reply`, 204 et rappel) et le fil séparé (`spawn_check_thread`), `NavL2` : filtres Stats rendus
  et Communauté masquée en bureau ; chaque test échoue condition inversée (vérifié une fois). **Jeté** : voile `bg-black/55` de la
  modale, même convention que `NavL1MobileMenu` et `alert-dialog` ; dette commune « jeton de voile à créer ».
  **Relevé, non traité** : (a) `HomeCitationsNearCompletion` navigue encore vers la route de redirection `.../commendations` (un
  saut de redirection, sans effet visible) ; (b) le téléchargement de l'installeur n'a pas d'échéance : un flux bloqué garde le
  verrou de vérification (une échéance totale couperait un téléchargement lent légitime ; il faudrait une échéance d'inactivité).

- **L19 (2026-11-02) — relevé, découvertes et témoins.** (a) Modèle NSIS de Tauri (extrait de
  `cli.win32-x64-msvc.node`, texte en clair) : `!define MUI_FINISHPAGE_NOAUTOCLOSE` juste avant `MUI_PAGE_FINISH` ; MUI2
  (`Contrib/Modern UI 2/Pages/Finish.nsh`) ne pose alors pas son `SetAutoClose true`, et le modèle ne le pose que pour le
  mode passif et la mise à jour (fin de `Section Install`) ; aucune option de configuration Tauri n'existe (`bundle.windows.nsis`
  : `installMode`, `installerHooks`, ...), d'où le crochet `NSIS_HOOK_POSTINSTALL` (`SetAutoClose true`). (b) **La cause du
  lien `target="_blank"` muet n'était pas l'absence de `on_new_window`** (relevé L16 et L18) : le plugin `opener` injecte
  par défaut un script de clic qui appelle `open_url` par IPC (refusé à la page : aucune capacité) ; désactivé par
  `open_js_links_on_click(false)`, voir 19.2. (c) `window.open(url, '_blank', 'noopener,noreferrer')` rend toujours `null` :
  le formulaire de signalement prenait cela pour « fenêtre bloquée » (copie de l'URL dans le presse-papier, formulaire non
  refermé) sur le site aussi ; remplacé par un lien cliqué par programme (19.4), garde-rail presse-papier
  (`useCopyToClipboard.guard.test.ts`) : entrée `FeedbackDrawer.tsx` retirée. (d) Écart de lecture de la demande : les
  référentiels ne s'ouvraient pas depuis `HelpSplitButton` (aide et notes de version) mais par le mini-onglet vertical du
  bord droit de `AssetDrawer` ; `HelpSplitButton` est resté intact. (e) **Raccourci** : `Ctrl+Maj+R` est le rechargement sans
  cache du moteur ; retenu `Ctrl+Maj+L`. Les touches de confiance envoyées par DevTools (CDP) ne passent pas par les
  accélérateurs natifs du moteur : l'absence de conflit natif repose sur la liste connue des accélérateurs, pas sur un
  essai ; le défaut est neutralisé (`preventDefault`) par précaution. (f) **Relevé, non traité** : en bureau les toasts
  (`top-right`) et la barre de progression (`left-60 right-0`) passent au-dessus du volet ancré ; l'infobulle du bouton est le
  seul endroit où le raccourci est écrit (la page d'aide n'en parle pas). (g) **Observation non expliquée** : au premier
  lancement du binaire de développement sur `LevelUp-dev`, une capture montrait le volet ouvert avec le focus dans la
  recherche, alors que `localStorage` le donnait fermé quelques secondes plus tard ; non reproduit ensuite (ouverture par
  bouton, par raccourci, rechargement : état et focus conformes) ; à surveiller au gate visuel (19.5). Témoins :
  `scratchpad/l19/` (`desktop_l192_temoin.log`, `desktop_l193_l194_temoin.log`, captures `s1_ouvert.png`, `s2_large.png`,
  `s3_drag.png`, `s4_page.png`, `s5_feedback.png`, `installer_genere.nsi`).
- **Revue adversariale L19 (2026-11-02, un relecteur Opus) : 3 constats recevables (2 P1, 1 P2 retenu), 0 P0, corrigés ; 4 trous de tests comblés par des tests de texte.**
  (C1, P1) doc inversée : les commentaires de `childproc`, de l'archlint `childproc_single_spawn`, de `filmproc/hideconsole_win.go` et du témoin `fenetre_e2e_win_test.go` affirmaient qu'un serveur lancé avec `CREATE_NO_WINDOW` donnait une fenêtre à chaque enfant console ; la mesure du lot dit le contraire (console cachée partagée par les enfants ; seul un parent sans aucune console, `DETACHED_PROCESS`, exécutable `-H windowsgui` ou service, donne une fenêtre par enfant). Les quatre commentaires disent le contrat vrai : aucun enfant n'ouvre de fenêtre quel que soit le mode de lancement du serveur (console visible, cachée, aucune), cas mesurés le 2026-11-02 ; le helper n'explique PAS, à lui seul, une fenêtre vue sur l'application installée. Aucun changement de code. (C2, P1) `SidebarNav` faisait 131 lignes : le pied de barre (Aide, référentiels, signalement, Paramètres, thème, À propos, version) est extrait dans `SidebarFooter.tsx` (chemin, langue et version en props ; stores des deux panneaux lus dans le pied, aucun store nouveau), `SidebarNav` passe à 79 lignes ; tests existants inchangés et verts. (C3, P2) le lien d'exemple de match de `IssueTable` (`target="_blank"` vers une route locale) était muet dans la fenêtre de bureau (nouvelle fenêtre refusée) : sous `can_use_compact_chrome` il navigue dans la même fenêtre (sans `target` ni `rel`), le site hébergé garde le nouvel onglet ; libellé sans « nouvel onglet » en bureau (clé `admin.dq.example_open_here`, fr et en) ; deux tests par capacité. (C4) tests de texte : `tests/textes_de_configuration.rs` (`SetAutoClose true` dans `NSIS_HOOK_POSTINSTALL` ; `.open_js_links_on_click(false)` ; la closure `on_new_window` rend `NewWindowResponse::Deny` en dernier, jamais `Allow` ni `Create` ; pas de runtime Tauri ni d'installeur en test) et `libs/film/filmproc/sansfenetre_source_test.go` (tout lanceur d'enfant du paquet appelle `sansFenetre`, vérifié en échec sans l'appel de `runner.go`).

- **L20 (2026-11-02) — relevé, décisions et témoins.** (a) **Le serveur n'embarquait pas la base de fuseaux** : sans installation de Go, `time.LoadLocation("Europe/Paris")` échoue (témoin `scratchpad/l20/witness_20b_tzdata.log`, binaire lancé avec `GOROOT` absent) ; chaque lecture de `user_timezone` des pages de tendances retombait sur son repli sur l'application installée ; corrigé dans 20.b par `time/tzdata` dans `cmd/server`. (b) **L'échéance du builder ne couvre pas le téléchargement** : `Update.timeout` vaut `None` à la sortie de `check` ; d'où le chien de garde d'inactivité de 20.d (60 s sans octet, 60 min au total) plutôt qu'une échéance totale courte qui aurait coupé un téléchargement lent. (c) **Encodeur H.264** : le build LGPL livré a `h264_mf` et `libopenh264`, pas `libx264` ; les trois produisent un HLS fMP4 lisible par ffprobe (`witness_20a_encodeurs.log`, 4 s par segment) ; `h264_mf` sort en profil Constrained Baseline. (d) **CHECKPOINT** : `shared_matches` n'est jamais vu par la tâche, la bascule B-swap tient le bail pendant tout son RW et ferme le handle avant de le rendre ; l'intervalle se lit au démarrage (pas de rechargement à chaud). (e) **Relevé, non traité** : un Windows N sans Media Foundation est écarté à l'essai d'encodage (journal), mais aucune boîte ne le dit à l'utilisateur ; `PageHeader.actions` et `subtitle` n'avaient pas d'appelant : retirés à la revue (les filtres Stats restent dans `SoloFiltersBar`, sous l'en-tête). (f) **Témoin 20.h sans lancer l'application** : consigne du coordinateur après des ouvertures répétées du navigateur de l'utilisateur ; le témoin des décalages est pris sur le build web servi par `vite preview`, bootstrap simulé, aucun serveur ni coquille ; la seule fois où un serveur a été lancé (`server_l20.exe`, sans coquille, racine de données vierge dans le scratchpad, 5 lancements de 20 s) il n'a ouvert aucun navigateur.

- **Revue adversariale L20 (2026-11-02, deux relecteurs Opus) : M 2 P1 + 3 P2, N 2 P1 + 7 P2, 0 P0, tous corrigés ; décisions pilote : ligne d'onglets réservée et collante, sous-titre/actions retirés.**
  Web (N) : (N1) les `Link` d'onglets portent `activeOptions={{ exact: true }}` (le routeur marquait actif tout préfixe : deux « page courante » sur les sous-onglets), testé avec un vrai routeur mémoire ; (N2) la ligne d'onglets (`h-11`, trait de séparation) est réservée sur toute section, vide sur Médias et Explorer, premier bloc de contenu à la même ordonnée partout (témoin `vite preview` headless : 116 px sur Carrière, Carrière/Citations, Solo, Ascension, Médias, Explorer) ; (N3) `nav_career_aria` et `nav_community_aria` supprimés, manifestes régénérés ; (N4) `subtitle` et `actions` retirés de `PageHeader` (ils reviendront avec la première page qui en a besoin), zone titre `min-h-12` ; (N5) test du layout joueur avec le vrai `SectionHeader`, test direct de `useDockedPanelInset`, rendu réel de `SquadLayout` qui trouve le h1 ; (N6) la ligne d'onglets est collante (`sticky top-0 z-30 bg-background`, frère de l'en-tête et non son enfant : collante dans l'en-tête seul elle partirait avec lui), sous la barre de progression, les tiroirs et les toasts (`z-50`) ; la barre haute du site n'est pas collante (barre fixe hors de la zone qui défile) ; conséquence reprise dans le lot : les bandeaux collants posés dessous se décalent de sa hauteur (`STICKY_BELOW_TABS_CLASS` : `SoloFiltersBar`, `SquadFilterBar`, barres de filtres de la synthèse et de `useLocalFilterBar`, `CampaignTracker` et `TendancesHorizonBar` avec marge, en-têtes L3 de `SessionDetailPage` en `calc`) ; la ligne vide d'une section sans onglets ne colle pas (44 px de vide épinglés ne servent à rien) ; (N7) `chooseH264` rend l'erreur du contexte (jamais `ErrNoH264Encoder`) quand l'échéance de 5 s du boot expire pendant l'essai, `check-env` dit « non vérifié » au lieu d'un faux `[KO]` ; (N8) commentaire de `media_tooling.go` ; (N9) le repli de `sectionWithHeader` sur `L1_SECTIONS` respecte la capacité de la section (titre sans `lusr`/`career`/`media` : ni titre ni onglets au-dessus de « fonction indisponible »). Revue CI du même lot : constantes nommées pour les options ffmpeg répétées (`-c:v`, `-preset`, `-crf`, `-b:v`, `-rc_mode`) et `libwebp` (goconst), les deux valeurs de la table de fuseaux ne sont citées qu'une fois dans le paquet (les tests passent par une constante).
  DuckDB, configuration, release (M) : (M1) `/bootstrap` sert la langue et le fuseau de la même source que le magasin de réglages : une règle unique « clé absente, vide ou non textuelle = non écrite » (`config.DeclaredString`, partagée par la configuration, `settings.Store` et `/bootstrap`) et les mêmes défauts (`SystemLocale.LangOr`, `TimezoneOr`) ; sous `AdoptSystemLocale`, langue du système ou `en` (langue inconnue, p. ex. pt-BR), fuseau du système ; hors capacité inchangé (`LEVELUP_LANG`, `Europe/Paris`) ; (M2) le ratchet `checkpoint_single_site_test.go` reconnaît toute instruction CHECKPOINT d'un littéral (bare, nommée, après « ; » dans un script, `CALL checkpoint()`, `force_checkpoint`) sans compter commentaires, identifiants Go, messages, noms de drapeau ou de tâche ; recompte réel : 4 vrais CHECKPOINT de `cmd/wal_forensic_compare` (+1 en script multi-ligne) et 1 de la migration `steps_shared_drop_weapon_kills.go` n'étaient pas comptés, 3 entrées de l'ancienne allowlist ne comptaient que des noms de drapeau (`cmd/backfill-world-player-stats`, `cmd/world-aliases-persist`) ou un nom de phase (`shutdown.go`) ; l'allowlist est maintenant un compte exact par fichier avec raison datée, et un fichier qui n'a plus de CHECKPOINT fait échouer le test ; (M3) `checkpointOne` relit l'état « fermé » du handle APRÈS la prise du bail (`skipped_closed`, plus compté en échec) ; (M4) un CHECKPOINT coupé par l'annulation du contexte à l'arrêt est compté `interrupted` à part, commentaire réécrit (le CHECKPOINT DuckDB est atomique, `CloseAll` checkpointe à la fermeture) ; (M5) la tâche périodique saute le `shared_social` de TOUS les titres (`PathResolver`), et la tâche sociale (5 min) les checkpointe tous, sinon celui de Halo 5 n'aurait plus eu aucun CHECKPOINT périodique : effet de bord assumé, le site hébergé checkpointe désormais aussi le `shared_social` de Halo 5 quand il est ouvert ; (M6) tests : handle RO jamais checkpointé, filtre `closed` et sortie sur `ctx.Err()`, `CheckpointInterval > 0` (`periodicCheckpointFor`), et boucle de `download_watched` extraite en `download_watch::watch_download` (résultat lu avant le verdict, abandon une seule fois, 5 tests sans réseau).

- **Revue adversariale L21a (2026-11-02, deux relecteurs Opus) : O 1 P1 + 1 P2, P 2 P1 + 7 P2, 0 P0, tous corrigés ; décision pilote P7.**
  Authentification (O) : (O1) `AuthCompletion.write` purge le cache process des HaloTokens (`tokencache.InvalidateCachedPlayerTokens`) après chaque écriture réussie, les trois chemins (`Persist`, `CompleteForTrackedProfile`, `SeedCredentials`), code d'appareil comme navigateur ; l'ADR et l'en-tête du handler l'affirmaient sans que le code le fasse ; tests : cache posé puis purgé pour chacun, et non purgé quand l'écriture échoue. (O2) le flux du navigateur utilise un client Azure PUBLIC construit explicitement (`ExchangeAuthorizationCodePublic`, `BuildAuthorizeURLPublic` de `platform/auth` : identifiant fixe `LevelUpClientID`, aucun secret), jamais `ResolveAzureOAuthClient` : un `.env.local` de dépôt chargé en `-mode desktop` sans `LEVELUP_DATA_DIR` ne fait plus d'une requête `/token` celle d'un client confidentiel ; tests : requête `/token` capturée (pas de `client_secret`, `client_id` attendu, avec et sans `SPNKR_AZURE_CLIENT_SECRET` / `LEVELUP_OAUTH_CLIENT_ID` dans l'environnement), garde de source du handler.
  Web et médias (P) : (P1) l'état et le sondage de la connexion par le navigateur vivent dans `StepDeviceCode` (`useBrowserLogin`), plus dans le panneau : finir la connexion dans l'onglet après avoir basculé vers le code d'appareil fait avancer l'étape (le serveur ne réclame la tentative autorisée qu'au sondage), test Vitest avec réponse du sondage retenue jusqu'après la bascule ; (P2) `HomeLink` unique pour les trois barres, garde-rail `homeLink.guard.test.ts` ; (P3) Échap ferme un menu épinglé à la souris, focus resté sur le bouton ; (P4) `Ctrl+B` ignoré pendant une saisie (`lib/dom/editableTarget`, partagé avec le rejeu) ; (P5) l'état de « À propos » et sa fenêtre sont tenus par `SidebarNav` ; (P6) le focus qui était dans la barre revient au bouton de bascule de la nouvelle barre ; (P7, décision du pilote) un choix manuel qui coïncide avec ce que produirait `auto` remet la préférence à `auto` (seul un choix contraire est mémorisé) ; (P8) la borne du volet des référentiels lit la largeur réelle de la barre (240 ou 56 px, `lib/layout/sidebarWidth` publié par `useSidebarCollapse`) ; (P9) test de handler : la sonde média du démarrage (variable `inspectMediaTooling`, posée par un test) atteint `media_h264_missing` de `/bootstrap` par le vrai routeur, le test échoue sans le branchement.

- **Revue adversariale L21b (2026-11-03, deux relecteurs Opus) : Q 2 P1 + 5 P2, R 1 P0 + 3 P1 + 2 P2, tous corrigés ; décisions pilote : auto-sync ON par défaut en desktop, nom de tâche par racine.**
  Serveur (R) : (R1 P0) `armGameWatch` fait le premier relevé du jeu avant toute tâche lourde et `Mode.ExpectFirstPoll` ferme la porte tant qu'aucun relevé n'a eu lieu (`main.go`, `boot_gameproc.go`, `gameproc/mode.go`) ; (R2) `runKillSource` passe par `HeavyWorkAllowed` (reportée, rattrapée par le backlog) ; ADR 0045 D-23 corrigée ; (R3) capacité `AutoSyncByDefault` + `Store.WithAutoSyncDefault`, `SyncOutcome`, compteur `game_mode_sync_skipped_total`, docs `CONFIGURATION.md` fr/en et runbook ; (R4) `reporterEnModeJeu` extraite de `buildAll` ; (R5) `Ensure /F` à chaque démarrage réglage actif ; (R6) `TaskNameFor(racine)` = `LevelUp\Refresh-<8 hex>`. Coquille (Q) : (Q1) `dispatch_tray_change` par `run_on_main_thread` + `ShellState::tray_locale` résolue ; (Q2) `launch::restores_maximized_*` ; (Q3) `RegisterOnce` ; (Q4) `autostart::EntryState` + lecture `winreg` ; (Q5) `Delivery::Requested` ; (Q6) `Relay.ForPlayer` + `OwnerFilter` (jeton de rafraîchissement) ; (Q7) `Action::RevealWindow` et `Action::SyncSkipped`. Tests de chacun nommés dans le commit. L'écart de nom de tâche (`LevelUp\Refresh` en 21.e) est remplacé par le nom à empreinte.

- **Revue adversariale L22 (2026-11-03, deux relecteurs Opus) : S 2 P1 + 4 P2, T 4 P1 + 5 P2, 0 P0, tous corrigés ; décisions pilote : matchs complets exportés, restauration par `-mode restore`, verrou d'instance pour tous les profils.**
  Serveur (S) : (S1) export du match entier, import borné aux matchs du joueur (`transfer/format.go`, `import.go`, `persist/import_persister.go` : `Where`, `OwnerCol`, colonne `id` locale non reprise), test d'intégration sur le vrai schéma (export, import, `knownset` ne redemande rien, rejouer n'écrit rien) ; (S2) verrou d'instance pris par tous les profils (`boot_listen.go`), la restauration refuse même sous `--force` (`parquetbackup/command.go`) ; (S3) bail par table et échéance par table (`pkg/duckdbbackup/exporter_held.go`) ; (S4) fichier hors manifeste refusé, WAL résiduel refusé sans `--force` et écarté AVEC sa base sous `--force` (`restore.go`, `importer.go`) ; (S5) quota consommé après un export réussi, un export à la fois, en-tête `X-Requested-With` pour l'export par session, base joueur trouvée par le xuid (`token.go`, `export_desktop.go`) ; (S6) plafond mémoire dit par base (ADR, `CONFIGURATION.md` fr/en, libellé web), rotation qui garde le dernier jeu complet (`rotate.go`), import filtré sur le xuid attendu.
  Web et diagnostic (T) : (T1) clés interdites ancrées sur des noms de secrets (`redact.go`) ; (T2) expurgation consciente du JSON, valeur bornée au guillemet (`ScrubLog`) ; (T3) test sentinelle à valeurs opaques derrière une clé sensible, chaque filtre de `Scrub` exercé seul (mutations vérifiées) ; (T4) `fillTemplate`/`substitute` uniques dans `lib/i18n/template.ts` + garde-rail ; (T5) voir la découverte du lot ci-dessus (abandonné, types gardés par une liste datée) ; (T6) résolution des filtres invalidée à la fin d'un import, caches de lecture joueur invalidés côté serveur (`AfterImport`) ; (T7) `site_unreachable`, `site_tls`, `site_not_found` fr/en ; (T8) 429 retiré, carte masquée sans identité Xbox liée ; (T9) fréquence réelle (« Toutes les 48 h »), état indisponible distinct d'« aucune sauvegarde », libellé mémoire « par base ouverte », import en cours qui bloque la synchronisation de l'assistant.

- **Revue adversariale L22b (2026-11-03, un relecteur Opus) : 1 P0 + 1 P1 + 6 P2, tous corrigés ; décisions pilote : verrou solo pour tout décodage dans le processus, la file cède à la synchro, décompte par fichiers `.meta`.**
  (U1 P0) l'étape kill source du post-sync prend `filmproc.AcquireSolo` (refus immédiat, passe reportée) avant `CollectMatches`, la cuisson après synchro qui trouve le verrou tenu REPORTE (jamais un échec), la file attend la fin d'un cycle (`Coordinator.SyncRunning`) entre deux films ; (U2 P1) chaque nouvel essai d'un film occupé repasse par `waitTurn` ; (U4) `finish` abaisse `running` avant le décompte, reprise d'une file finie = 409 ; (U5) `Store.Update`, un seul verrou d'écriture de `app_settings.json` ; (U6) l'artefact porte la version de l'application productrice, un artefact d'une version plus récente n'est jamais en retard ; (U7) décomptes numérotés ; (U8) `filmPresent` avant toute lecture, fichiers `.meta`, décompte au démarrage / fin de file / `POST .../rescan` seulement, ADR 0045 et runbook corrigés, mesure en test (2 000 artefacts : 33 Mo puis 108 Ko lus) ; (U9) garde-rail web du bandeau, test de `boot_replayregen.go`, test « deux décodages impossibles » contre la cuisson post-sync. Corrigé à part (CI) : `Registry.GameProcessNames` déterministe.
- **Revue adversariale L22c (2026-11-04, un relecteur Opus) : 3 P1 + 6 P2 + 1 P1 doc, tous corrigés.**
  (V1 P1) le désinstalleur de Tauri efface `Run\LevelUp` sans regarder ce qu'elle lance : `NSIS_HOOK_PREUNINSTALL` lit la valeur AVANT le nettoyage (`$LevelUpRunValue`), `NSIS_HOOK_POSTUNINSTALL` la réécrit quand elle ne visait pas `$INSTDIR\` (même règle que `CommandTargets`) ; (V2 P1) `CheckIfAppIsRunning` en tête du crochet : « Annuler » ne retire plus rien ; (V3 P1 doc) la garde `$UpdateMode` ne couvre que l'updater, une mise à jour manuelle ou `/P` retire puis repose au prochain lancement : ADR D-28, runbook, `COMMANDS` EN+FR ; (V4 P2) `schedtask.Manager.RemoveIfTargets` : la tâche n'est retirée que si son action (`/Query /XML`) lance le serveur de CE dossier, sinon `tache=foreign` ; (V5 P2) le témoin porte `exit=` et le bilan complet, `e2e-install.ps1` les asserte (dont le nom de la tâche de la racine jetable) ; la branche NSIS `SetEnvironmentVariable` (racine par défaut) n'est pas jouée sur le poste, tenue par un garde de texte, dit au runbook ; (V6 P2) `./internal/platform/startupentry/...` ajouté au job Windows CGO ; (V7 P2) garde Rust : `nsExec` DANS le bloc `$UpdateMode <> 1`, test Go du câblage (`newUninstallCleanupDeps`) ; (V8 P2 doc) le verrou solo est par racine de données (`<racine>/cache/film_decode.lock`), pas par machine ; (V9 P2) `CodeSortieVerrouTenu = 3` (2 = `flag.ExitOnError`) ; (V10 P1) `replayregen/service.go` 579 -> 476 lignes, décompte et ensemble des matchs écartés extraits dans `count.go`, sans changement de comportement.
- **Aléas de temps en CI (2026-11-02, L19 fusion et L20)** : run de la release `4d66a8fe58` rouge sur `TestE2ELanceurDesktop` (« aucun événement en 3m0s » après `listening`, le même commit passant le build et le test d'installation du workflow Release) ; run L20 `c76817d909` rouge sur le lot `film-grammar-reste` (600 s dépassées, ~170 s d'habitude, aucun fichier du décodeur touché). Les deux relances sont vertes : runners lents, pas le code. Budgets NON élargis (doctrine de `go-shards.txt` : rééquilibrer, pas allonger) ; à surveiller : deux occurrences de plus = rééquilibrer `grammar` ou isoler la migration du boot dans le test de lancement.

## 12. Clôture du chantier (2026-11-04)

**État** : P0, P1 et P2 livrés et fusionnés dans `release/v8.1.0` (L0 à L22c, PR #125 à #145 ; dernier commit `1da5888487`). Tous les gates utilisateur sont passés (22c.z le 2026-11-04, désinstallation avec démarrage Windows et tâche planifiée actifs, installeur du run
Release `41816350092`). Toutes les revues adversariales (2r, 3r, 4r, H, I, J, K, L, M, N, O, P, Q, R, S, T, U, V) sont consignées en §10,
tous leurs P0/P1 corrigés. Le workflow Release est vert sur chaque push de la release (build de l'installeur + installation/désinstallation sur
runner vierge) ; le run **signé sur tag** reste à jouer : c'est le tag `v8.1.0`, geste de l'utilisateur, **reporté** par lui le 2026-11-04 le temps
d'un autre chantier.

**Reste après le tag** : L23 mesuré et 23.3 livré (installeur 193 → 152 Mo, 2026-11-04) ; **L24 (moteur / paquet de ressources, §9.27) AVANT le tag, dimanche 2026-11-08** ; les reports et dettes du chantier sont au
`.ai/REGISTRE_REPORTS.md` (section « Chantier Desktop », 2026-11-04) et en §10.

**Marche à suivre du tag** (quand l'utilisateur le décide) : `docs/CHANGELOG.md` et `docs/FR/CHANGELOG.md` : poser la date sur `## [8.1.0]` ;
vérifier `Cargo.toml`, `tauri.conf.json` et le tag (`8.1.0`) ; `git tag v8.1.0` sur `release/v8.1.0` puis `git push origin v8.1.0` ; le workflow
Release construit, signe (`TAURI_SIGNING_PRIVATE_KEY`), vérifie la signature, publie l'installeur et `latest.json` ; puis fusion de la release dans
`main` (déploiement prod automatique : prévenir).

## 11. Protocole de reprise de session

1. Lire ce plan (§9, statuts) et la dernière entrée `thought_log` du chantier.
2. `git log --oneline -10` sur `feat/desktop-p0` ; `ListAgents` (lots en cours).
3. Reprendre au premier item sans statut de la vague en cours ; ne pas ouvrir la vague suivante tant que
   la précédente n'est pas fusionnée et ses témoins rejoués.
