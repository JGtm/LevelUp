# Audit complémentaire — ce que le Desktop fait revoir — 2026-10-27

Complète `.ai/ETUDE_DESKTOP_TAURI_2026-10-26.md` (l'étude). L'étude dit ce qu'il faut pour un
Desktop ; ce registre cadre les **12 chantiers** qu'elle fait apparaître, dont la plupart
corrigent un défaut qui coûte déjà en dev ou en prod. Aucun code modifié : chaque chantier
deviendra un plan (`plan-review`), exécuté sous `plan-execution`, relu sous
`adversarial-review` pour les lots à risque.

## Cadrage

- **Périmètre** : les chemins cités par chantier (liste fermée ci-dessous), état de
  `release/v8.1.0` au 2026-10-27 (`a208d0375`).
- **Doctrine de référence** : CLAUDE.md (règles 3, 5, 6, 7, 11 ; anti-patterns 3, 4, 10 ;
  règles ART n° 4 et 5), ADR 0013, 0023, 0035, 0038.
- **Vérification** : chaque constat a été rouvert sur pièces par le pilote (fichier, ligne,
  commande). **Vérification adverse par contexte frais NON faite** (coût en quota) : à lancer
  sur C1, C2 et C3 avant d'en faire des plans, si le user le souhaite.
- **Gravité** (échelle du skill `adversarial-audit`) donnée deux fois : **aujourd'hui** (dev et
  VPS) et **Desktop** (poste utilisateur sans opérateur).

## Synthèse

| # | Chantier | Aujourd'hui | Desktop | Taille | Dépend de |
|---|---|---|---|---|---|
| C1 | Arrêt propre sous Windows | P1 | P0 | M | — |
| C2 | Migrations : échec avalé, pas de garde de version | P0 | P0 | M | — |
| C3 | Jeton longue durée non gardé en mode `none` | P1 | P0 | S | décision C3-a |
| C4 | Cache Spartan : expiration sur horloge monotone | P2 | P1 | S | — |
| C5 | Films bruts : emplacement et quota d'archive | P2 | P1 | M | décision C5-a |
| C6 | Séquence de démarrage : `main.go` 2 016 lignes | P1 | P1 | L | — (facilite C1, C2, C7, C8) |
| C7 | Profil de déploiement | P2 | P1 | M | C6 conseillé |
| C8 | Racines ressources / données | P2 | P0 | L | C6 conseillé |
| C9 | Voie CI Windows avec CGO | P1 | P0 | M | — |
| C10 | Binaire autonome (ressources intégrées) | P2 | P2 | M | C8 |
| C11 | `static/` : 84 Mo de sons WAV | P2 | P2 | S | décision C11-a |
| C12 | Police auto-hébergée, chemin ffmpeg centralisé | P2 | P1 | S | — |

Ordre recommandé : **C2, C1, C3, C4** (défauts actuels, courts), puis **C9** (filet Windows
avant les refontes), **C6**, **C7**, **C8**, puis **C5, C12, C11, C10**.

---

## Constats retenus

### C1 — Arrêt propre sous Windows [aujourd'hui P1 · Desktop P0]

- **Où** : `apps/go-api/cmd/server/boot_gate.go:197` (`signal.Notify(sigCh, os.Interrupt,
  syscall.SIGTERM)`, seuls signaux d'arrêt) ; `apps/go-api/.air.toml:35-44` (air ne sait pas
  interrompre sous Windows : il tue le processus, `kill_delay` ignoré).
- **Règle** : CLAUDE.md, règles ART (bug DuckDB #7659 au rejeu du WAL, `cmd/duckdb_7659_repro`) ;
  budget d'arrêt de 40 s (`cmd/server/shutdown_budget.go`) qui ne sert jamais sous Windows hors
  console.
- **Conséquence** : chaque rechargement à chaud d'air sur le poste de dev ferme les bases
  DuckDB sans CHECKPOINT ni vidage du persist ; un sidecar Tauri tué (TerminateProcess) ferait
  de même sur chaque poste utilisateur, à chaque fermeture de l'app.
- **Reproduction** : `grep -n "send_interrupt\|ATTENTION Windows" apps/go-api/.air.toml` ;
  `grep -n "signal.Notify" apps/go-api/cmd/server/*.go`.
- **Traitement proposé** : un déclencheur d'arrêt de plus dans `boot_gate` : fermeture de
  stdin quand le serveur est lancé avec `-arret-sur-stdin` (option explicite, pour ne rien
  changer au conteneur) ; et un `POST /api/v1/internal/shutdown` réservé à la boucle locale
  (`LoopbackOnly`) et protégé par un secret passé au lancement. Air : pré-commande qui appelle
  ce point avant la reconstruction, ou lancement du serveur avec `-arret-sur-stdin` par un
  petit relais. Les deux déclencheurs passent par le même chemin que le signal (même budget).
- **Tests / gate** : test du déclencheur stdin (fermeture du tube → séquence d'arrêt appelée),
  test du point d'arrêt (refus hors boucle locale, refus sans secret) ; garde-rail existant
  `shutdown_budget_test.go` vert. Contrôle manuel : rechargement air sous Windows, journal
  « arrêt propre » présent, aucune relecture de WAL au démarrage suivant.
- **Décision** : aucune.

### C2 — Migrations : échec avalé, pas de garde de version [aujourd'hui P0 · Desktop P0]

- **Où** : `apps/go-api/cmd/server/main.go:467-468` : `if err := runMigrations(...); err != nil {
  slog.Debug("migrations ignorées (DB verrouillée), démarrage sans migration") }`.
  `runMigrations` rend l'erreur de toute migration ET des contrôles de démarrage
  `ValidateLUSRChainClassifierWired` / `ValidateObjectiveFamilyClassifierWired` (« fail-fast au
  boot », Lot B). `apps/go-api/internal/migration/registry.go:176-195` écrit
  `title_schema_version`, que rien ne relit.
- **Règle** : CLAUDE.md règle 3 (« jamais d'erreur avalée en silence : logger AVANT toute
  dégradation ») et anti-pattern 10 (swallowed error) : l'appelant jette l'erreur et écrit, au
  niveau Debug, un message qui suppose un verrou quelle que soit la cause. Seul l'échec d'une
  étape de schéma est journalisé en Error par le runner (`internal/migration/registry.go:376`) ;
  l'échec d'ouverture RW, des réconciliations (seeds, registre et noms d'armes) et des deux
  contrôles de classificateurs ne laisse aucune trace aux niveaux de journal de prod.
- **Conséquence** : une migration de schéma en échec en prod laisse démarrer le serveur sur un
  schéma partiel (une ligne Error, puis un démarrage normal) ; les contrôles « fail-fast » de
  classificateurs sont neutralisés en silence par leur appelant. Une restauration de sauvegarde ou un
  retour de version peut ouvrir une base plus récente que le binaire sans refus.
- **Reproduction** : `grep -n "migrations ignorées" apps/go-api/cmd/server/main.go` ;
  `grep -rn "title_schema_version" --include=*.go apps/go-api/internal | grep -v _test`
  (écritures seulement).
- **Traitement proposé** :
  1. distinguer le verrou (erreur typée du lease) du reste ; tout autre échec = journal Error
     avec l'erreur, puis arrêt du démarrage ;
  2. relire `title_schema_version` à l'ouverture : base plus récente que le binaire = refus
     typé et écran explicite (la page de démarrage sert déjà un état) ;
  3. copie des bases avant une montée de schéma (seulement si des étapes sont en attente),
     dans le dossier de sauvegardes ; rétention courte, nommée.
- **Tests / gate** : test du chemin d'erreur (étape en échec → démarrage refusé, journal Error) ;
  test de la garde de version (base à version N+1 → refus) ; test de la copie (étape en attente
  → copie présente, aucune étape → aucune copie) ; `go test -tags=integration` du paquet
  `migration`.
- **Dépendances** : vérifier que le cas « verrou » existe encore réellement au démarrage (le
  serveur est seul écrivain ; le cas visé était probablement un second serveur ou un outil CLI
  tenant la base) avant de garder une exception pour lui.
- **Décision** : aucune ; c'est un défaut de la règle 3.

### C3 — Jeton longue durée non gardé en mode `none` [aujourd'hui P1 · Desktop P0]

- **Où** : `apps/go-api/internal/api/server_apiv1.go:357` (le magasin de jetons n'est câblé que
  si `cfg.AuthMode == "xbox"`), `apps/go-api/internal/api/handlers/auth_login_complete.go:28`
  (`ErrSessionNotAuthenticated` toléré : seul le jeton Spartan est posé en session),
  `apps/go-api/internal/service/xbox_auth_service.go:287` (`persistRTATokens`, seul écrivain du
  store après une connexion web).
- **Règle** : ADR 0023 (source unique `watcher_tokens/{xuid}.json`) ; `docs/INSTALL.md` affirme
  que l'assistant y écrit le jeton (doc inversée, anti-pattern 9).
- **Conséquence** : en installation locale par défaut, la sync cesse quand le jeton de session
  expire ; seuls `token-capture` / `token-import` sèment le store.
- **Reproduction** : lecture des trois sites ci-dessus. **Non reproduit par exécution** : à
  confirmer d'abord par un premier lancement réel en mode `none` sur une copie de données
  vide (présence ou absence de `data/auth/watcher_tokens/<xuid>.json` après l'assistant).
- **Traitement proposé** : extraire la persistance du jeton de `XboxSSOLinkStrategy` vers une
  étape commune à toute connexion par code d'appareil réussie, gardée par la porte « profil
  suivi » (ADR 0035 D3) ; corriger `docs/INSTALL.md` et `docs/FR/INSTALL.md` (règle 15).
- **Tests / gate** : test handler du device-flow en mode `none` (jeton écrit dans le store
  pour un profil suivi, pas pour un profil non suivi) ; garde-rails `auth/sentinel_test.go` et
  `sync/no_legacy_source_used_test.go` verts.
- **Décision C3-a** : en mode `none`, le jeton entre-t-il dans la mise en commun (ADR 0038) ?
  Proposition : non, « sans décision » comme partout ; le mode Desktop (C7) l'accordera d'office.

### C4 — Cache Spartan : expiration sur horloge monotone [aujourd'hui P2 · Desktop P1]

- **Où** : `apps/go-api/internal/platform/halo/player_token_cache.go:82`
  (`return time.Now().Add(fallbackTokenTTL)`), comparé ligne 92 à `time.Now()`.
- **Règle** : la comparaison de deux `time.Time` portant une lecture monotone se fait sur
  l'horloge monotone, qui ne compte pas la mise en veille (macOS, Linux). Les expirations
  réelles (`SpartanExpiresAt`, désérialisées, sans lecture monotone) se comparent en heure
  murale et ne sont pas touchées.
- **Conséquence** : un poste qui sort de veille sert un jeton Spartan expiré tant que le repli
  de 50 min n'est pas écoulé en temps « éveillé » → 401 sur les lectures en direct.
- **Reproduction** : `grep -n "fallbackTokenTTL)" apps/go-api/internal/platform/halo/player_token_cache.go`.
- **Traitement proposé** : `time.Now().Round(0).Add(...)` (retire la lecture monotone) au repli ;
  recenser les autres expirations calculées par `time.Now().Add` et gardées en mémoire
  (`grep -rn "time.Now().Add(" --include=*.go apps/go-api/internal | grep -iv _test` puis tri à la
  main : expirations de jetons, de tentatives, de caches).
- **Tests / gate** : test unitaire qui vérifie que l'expiration de repli ne porte pas de lecture
  monotone (`exp.Round(0) == exp`) ; une horloge injectable n'est pas nécessaire.
- **Décision** : aucune.

### C5 — Films bruts : emplacement et quota d'archive [aujourd'hui P2 · Desktop P1]

- **Où** : `data/cache/film_chunks` = **104 Go mesurés** sur le poste de dev (4 046 films,
  environ 27 Mo par film), 144 Go pour tout `data/`. Doctrine du code :
  `apps/go-api/internal/games/halo_infinite/film/filmcache/write.go` (« les films EXPIRENT côté
  serveur Halo, ~29 % du corpus déjà perdus » : l'archive est IRREMPLAÇABLE) et
  `apps/go-api/internal/replaybuild/artifact_purge.go` (ne touche jamais aux films).
- **Correction de l'étude** : l'étude (§7, §11 q4) proposait une « purge automatique une fois
  les faits extraits ». **Écarté** : les faits se recalculent depuis le film à chaque
  amélioration du décodeur (recuisson) ; supprimer un film expiré côté Halo perd la donnée
  pour toujours.
- **Mesure** : les chunks sont stockés compressés en zlib (en-tête `78 5e`) ; un gzip de plus
  sur un chunk de 442 749 octets n'en gagne que 1,4 %. Recompresser l'existant tel quel ne
  libère rien ; un recodage après décompression (zstd) n'est pas mesuré.
- **Conséquence** : sur un poste utilisateur, l'archive grossit sans borne sur le disque
  système, sans que l'utilisateur le sache ni puisse la déplacer.
- **Reproduction** : `du -sh data/cache/film_chunks` ; `od -A x -t x1 -N 2 <un chunk_00.bin>`.
- **Traitement proposé** : (1) emplacement de l'archive réglable (un autre disque), via
  `PathResolver` ; (2) occupation affichée dans les réglages et alerte à un seuil ; (3) mesure,
  sur une copie, du gain d'un recodage zstd après décompression, avant toute décision de
  format. Aucune suppression automatique.
- **Tests / gate** : test `PathResolver` (archive hors de la racine de données) ; garde-rail
  existant sur la disposition du cache (`filmcache`) vert.
- **Décision C5-a** : sur Desktop, archiver les films par défaut (comportement actuel), ou
  proposer « ne pas archiver les films » à l'installation (les faits et le rejeu restent,
  la recuisson future devient impossible pour ces matchs) ?

### C6 — Séquence de démarrage : `main.go` de 2 016 lignes [aujourd'hui P1 · Desktop P1]

- **Où** : `apps/go-api/cmd/server/main.go` : 2 016 lignes, `func main()` de 1 210 lignes
  (ligne 116).
- **Règle** : CLAUDE.md règle 5 (fichier ≤ 500 L, fonction ≤ 80 L) et anti-pattern 3 (god file /
  god function). Dette gelée par la baseline de lint, à ne pas accroître.
- **Conséquence** : le Desktop ajoute des modes de démarrage (arrêt sur stdin, auth `desktop`,
  « rafraîchir seulement » pour la tâche planifiée, signal « prêt » avec port réel, garde de
  version) ; les greffer ici accroît la dette, et chaque étape reste intestable isolément.
- **Reproduction** : `wc -l apps/go-api/cmd/server/main.go` ;
  `awk '/^func main\(\)/{s=NR} s&&/^}/{print NR-s; exit}' apps/go-api/cmd/server/main.go`.
- **Traitement proposé** : extraire la séquence en étapes nommées (configuration, journaux,
  porte de démarrage, titres, migrations, fournisseurs DuckDB, auth, persist, tâches de fond,
  routeur, arrêt), chacune dans son fichier avec ses dépendances explicites ; `main()` ne fait
  que les enchaîner. Déplacement pur d'abord (un lot), changements de comportement ensuite
  (C1, C2) dans leurs propres lots.
- **Tests / gate** : tests existants de `cmd/server` verts sans modification (preuve du
  déplacement pur) ; ratchet de lint : `main.go` sous 500 lignes, aucune fonction nouvelle
  au-delà de 80 lignes ; démarrage réel local et page servie.
- **Décision** : aucune ; lot à risque (démarrage prod) → `adversarial-review`.

### C7 — Profil de déploiement [aujourd'hui P2 · Desktop P1]

- **Où** : décisions de comportement réparties sur trois réglages : `DemoMode` (89 occurrences
  hors tests), `LEVELUP_ENV` / production (30), `AuthMode` (55), sous `apps/go-api/internal` et
  `apps/go-api/cmd`.
- **Règle** : CLAUDE.md règle 6 (≤ 2 copies d'un même pattern) et anti-pattern 4 ; doctrine
  multi-titre « brancher sur des capacités, jamais sur une comparaison » transposée au
  déploiement.
- **Conséquence** : un quatrième contexte (Desktop) multiplierait les combinaisons ; le front
  n'a aucune source unique pour savoir quoi afficher (admin, invitations, groupes, mise en
  commun).
- **Reproduction** : `grep -rn "\.DemoMode\b" --include=*.go apps/go-api/internal apps/go-api/cmd | grep -v _test | wc -l`
  (idem pour les deux autres motifs).
- **Traitement proposé** : un type `Deployment` (hébergé, Desktop, démo, développement) résolu
  une fois au démarrage, exposant des capacités fines (comptes et inscriptions, invitations,
  groupes, mise en commun, administration, sauvegarde restic, Discord, sortie réseau) ; exposé
  par `/bootstrap` ; garde-rail interdisant les nouvelles lectures directes de `DemoMode` /
  `AuthMode` hors du résolveur (allowlist datée des occurrences existantes, à réduire par
  lots). Le front branche sur les capacités.
- **Tests / gate** : tests du résolveur (chaque combinaison de réglages actuelle → mêmes
  capacités qu'aujourd'hui) ; garde-rail grep ; contrat OpenAPI régénéré ; typecheck web.
- **Décision** : ADR à écrire (escalade : contrat d'API + doctrine).

### C8 — Racines ressources / données [aujourd'hui P2 · Desktop P0]

- **Où** : `apps/go-api/internal/domain/title/registry.go:558-1146` (`<repoRoot>/data/...` en
  dur) ; racine détectée par `db_profiles.example.json` (`apps/go-api/internal/config/config.go:602-628`) ;
  `LEVELUP_ROOT` et `LEVELUP_DATA` posés par `Dockerfile:234-235` et `docker-compose.yml:15,17,97,98`
  mais lus par **aucun** code Go (0 occurrence).
- **Règle** : CLAUDE.md règle 7 (0 code mort, configuration morte comprise) ; ADR 0008
  (isolation par chemin, tout passe par `PathResolver`).
- **Conséquence** : impossible d'installer dans un dossier en lecture seule ; le référentiel
  versionné du rejeu (`data/titles/*/reference/`, 42 Mo) vit sous la racine de données ; deux
  variables d'environnement font croire à un réglage qui n'existe pas.
- **Reproduction** : `grep -rn '"LEVELUP_ROOT"\|"LEVELUP_DATA"' --include=*.go apps/go-api | wc -l`
  (0) ; `grep -n "LEVELUP_ROOT\|LEVELUP_DATA=" Dockerfile docker-compose.yml`.
- **Traitement proposé** : `PathResolver` reçoit deux racines (ressources, lecture seule ;
  données, écriture) ; `LEVELUP_DATA_DIR` réel remplace les variables mortes (retirées du
  Dockerfile et du compose dans le même lot) ; le référentiel du rejeu passe sous la racine de
  ressources ; le conteneur le monte en lecture seule. Par défaut, les deux racines valent
  l'actuelle : aucun changement pour le dev ni le VPS.
- **Tests / gate** : tests `PathResolver` (deux racines distinctes, chemins de ressources
  jamais sous la racine de données) ; garde-rail existant « pas de `filepath.Join(..., "data",
  ...)` » vert ; démarrage réel avec racines séparées ; `scripts/deploy.sh` relu (staging sous
  `data/`, cf. piège `git clean` du dépôt VPS).
- **Décision** : ADR 0008 à amender (escalade : emplacement des données).

### C9 — Voie CI Windows avec CGO [aujourd'hui P1 · Desktop P0]

- **Où** : `.github/workflows/ci.yml:400-470`, job `go-build` en matrice Linux / Windows : sous
  Windows, seulement `go vet` d'un sous-ensemble et `go build ./contracttest/...`, tous deux
  avec `CGO_ENABLED=0`. Aucun test, aucun binaire serveur (DuckDB exige CGO).
- **Règle** : skill `delivery-checklist` (« gates dont l'équivalent CI n'existe pas », axe A8) ;
  le développement se fait sous Windows et la cible Desktop est Windows.
- **Conséquence** : tout ce qui est propre à Windows (renommage de fichier ouvert, verrous de
  fichiers, `filmproc/*_windows.go`, `cmd_compact_passes_remplacement_win.go`, et demain C1)
  n'est vérifié que sur le poste du user.
- **Reproduction** : `sed -n 400,470p .github/workflows/ci.yml | grep -n "CGO\|run:"`.
- **Traitement proposé** : une voie Windows CGO (toolchain mingw sur le runner) qui construit
  `cmd/server` et lance `go test` sur `cmd/server` et les paquets à fichiers `*_windows.go`,
  plus un test de bout en bout : démarrage sur données de test vides, migrations, page servie,
  arrêt par le déclencheur C1, redémarrage sans relecture de WAL. Déclenchée sur `release/**`
  et PR vers `main` (coût de runner).
- **Tests / gate** : la voie elle-même, rouge sur une mutation jouée (un `*_windows.go` cassé).
- **Décision** : budget de minutes Windows (runners plus chers que Linux).

### C10 — Binaire autonome [aujourd'hui P2 · Desktop P2]

- **Où** : `.github/workflows/release.yml:160-164` (« binaires non autosuffisants : requièrent
  ffmpeg, `config/` et `static/` adjacents ») ; le site compilé est servi depuis le disque
  (`LEVELUP_WEB_DIST`), aucun `go:embed` de l'interface.
- **Contrainte** : `go:embed` ne lit que sous le dossier du module (`apps/go-api/`) ; `config/`,
  `static/`, `data/titles/*/reference/` et `apps/web/dist` sont hors module → étape de copie
  avant compilation (dossier généré, ignoré par git).
- **Conséquence** : les binaires de release sont inutilisables seuls ; le Desktop devra de
  toute façon livrer ces dossiers (dans les ressources Tauri) si rien n'est intégré.
- **Traitement proposé** : option de compilation `embed` (étiquette de build) qui intègre les
  ressources via un `fs.FS` ; `PathResolver` (C8) lit la racine de ressources depuis ce `fs.FS`
  ou le disque. Sans l'étiquette, comportement actuel.
- **Décision** : utile seulement si on veut des binaires de release utilisables seuls ; pour le
  Desktop, les ressources Tauri suffisent. Taille estimée du binaire intégré : de l'ordre de
  200 Mo (non mesuré). À trancher après C8.

### C11 — `static/` : 84 Mo de sons WAV [aujourd'hui P2 · Desktop P2]

- **Où** : `static/` = 108 Mo, dont `static/sounds` = 84 Mo (184 fichiers `.wav`, sous
  `static/sounds/halo_infinite/`), versionnés dans git (989 fichiers suivis sous `static/`).
  Lecteurs : `apps/web/src/features/match-replay/sound/*.ts`.
- **Conséquence** : poids de l'installeur, de l'image Docker et du dépôt ; chaque son du rejeu
  se télécharge en WAV non compressé.
- **Reproduction** : `du -sh static/sounds` ;
  `find static/sounds -type f | sed 's/.*\.//' | sort | uniq -c`.
- **Traitement proposé** : conversion en format compressé lu par Web Audio
  (`decodeAudioData`) ; outil de conversion en Go (règle « Go/TS uniquement », ffmpeg déjà
  requis) ; les noms de fichiers changent → table des sons mise à jour ; écoute comparée sur
  un échantillon.
- **Décision C11-a** : format (Opus, gain maximal mais support à vérifier dans WebKit pour un
  futur macOS ; ou AAC, universel). Nettoyage de l'historique git des WAV : hors périmètre
  (réécriture d'historique, décision séparée).

### C12 — Police auto-hébergée, chemin ffmpeg centralisé [aujourd'hui P2 · Desktop P1]

- **Où** : `apps/web/index.html:23-25` (Roboto via `fonts.googleapis.com`) ; littéraux
  `"ffmpeg"` / `"ffprobe"` en quatre endroits : `apps/go-api/internal/media/ffmpeg_slot.go:125`,
  `apps/go-api/internal/media/ffprobe.go:20`, `apps/go-api/internal/ops/healthcheck.go:93`,
  `apps/go-api/internal/ops/media_tooling.go:79-80`.
- **Règle** : CLAUDE.md règle 6 (3+ copies → helper + garde-rail) pour ffmpeg ; pour la police,
  fonctionnement hors ligne du Desktop et appel tiers à chaque visite de l'hébergé.
- **Conséquence** : sans réseau, la police retombe sur une police système ; un ffmpeg livré à
  côté du binaire (Desktop) ne serait pas trouvé, le PATH étant la seule source.
- **Reproduction** : `grep -n "fonts.g" apps/web/index.html` ;
  `grep -rn '"ffmpeg"\|"ffprobe"' --include=*.go apps/go-api/internal | grep -v _test`.
- **Traitement proposé** : Roboto servie depuis `apps/web/public` (licence Apache 2.0, poids
  400/500/700/900 en woff2) ; un résolveur unique des binaires média (`LEVELUP_FFMPEG_DIR`,
  puis dossier du binaire, puis PATH) utilisé par les quatre sites, garde-rail grep interdisant
  le littéral ailleurs.
- **Tests / gate** : test du résolveur (ordre de recherche) ; garde-rail ; build web et
  vérification que la page ne fait plus d'appel à `fonts.googleapis.com`.
- **Décision** : aucune.

---

## Constats écartés

| Constat | Motif d'écart |
|---|---|
| Purge automatique des films une fois les faits extraits (étude §7 et §11 q4) | Les films expirent côté Halo et servent aux recuissons : donnée irremplaçable (`filmcache/write.go`). Remplacé par C5 (emplacement, quota affiché, pas de suppression). |
| Recompresser les chunks existants | Déjà en zlib : +1,4 % de gain mesuré sur un chunk. Seul un recodage après décompression reste à mesurer (C5, point 3). |
| Garde-rail de licence GPLv3 du serveur à revoir | Pas de défaut : le serveur n'a pas besoin du décodeur, le Desktop non plus (assets cuits hors ligne). |
| Notifications par interrogation toutes les 30 s → flux poussé (SSE) | Aucun défaut constaté ; l'interrogation suffit pour relayer des notifications natives. |
| Secret de session par défaut public | Déjà refusé en `LEVELUP_ENV=production` (`config.go`) ; le cas Desktop relève du mode d'auth `desktop` de l'étude (§3.5), pas d'un chantier séparé. |
| Rattrapage de sync au démarrage, vérification des jetons au boot | Propres au Desktop (le VPS tourne en continu) : relèvent de la séquence « réveil » de l'étude (§4), pas de ce registre. |

## Découvertes hors périmètre (notées, non traitées)

- Commentaire « API FastAPI exposée en local » dans `docker-compose.yml` (vestige Python).
- L'exception « DB verrouillée » de C2 vise un cas à recenser : quel processus peut tenir les
  bases au démarrage d'un serveur seul écrivain ?

## Suite

1. Le user tranche C3-a, C5-a, C11-a et le budget de C9.
2. Vérification adverse de C1, C2 et C3 (optionnelle, sur accord : coût en quota).
3. Premier plan : **C2 + C1 + C3 + C4** (défauts actuels, sans décision d'architecture), une
   branche, un commit par chantier ; puis C9 ; puis C6 → C7 → C8 avec leurs ADR.
