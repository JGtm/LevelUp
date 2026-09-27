# PLAN — Lot backlog du 2026-09-26 : musique de fin et son du rejeu + sept entrées du backlog

> **Statut : GO donné par l'utilisateur le 2026-09-26, avec UN SEUL agent à la fois, supervisé,
> dans le worktree dédié du plan.** Contrat d'exécution : skill `plan-execution`,
> précisé au §3 (en cas de divergence, ce plan fait foi).
>
> **Source** : `.ai/BACKLOG.md` (entrées numérotées 1 à 8 par l'utilisateur, dans l'ordre du
> fichier au 2026-09-26), plus deux défauts signalés par l'utilisateur le 2026-09-26 :
> - **11** : « sur le replay 2D je n'ai plus la musique victoire ou défaite qui se joue en fin
>   de partie » ; précisé ensuite : « j'entends bien la voix de l'annonceur et je vois le
>   résultat mais pas la musique ».
> - **12** : « le réglage est bien conservé en mémoire mais pour entendre le son quand je
>   recharge la page ou vais voir un autre rejeu, il faut que je désactive le son et le
>   réactive ».
>
> Les entrées 9 (blocs Élimination/Infection) et 10 (`weapon_labels` Halo 5) sont hors périmètre.
>
> **Pièces vérifiées le 2026-09-26** : cinq enquêtes en lecture seule sur `feat/v75` `ea5682373`.
> `origin/feat/v75` `d61443ef5` a 10 commits de plus (audit du décodeur J1-J2), sans effet sur
> les fichiers de ce plan. Requêtes en lecture seule sur `shared_matches_v2.duckdb`, serveur
> arrêté. Les numéros de ligne sont indicatifs : **les symboles font foi**, et chaque lot rouvre
> ses pièces avant de coder (règle 4 du contrat).

---

## 0. Résumé

| Lot | Item | Sujet | Ce que l'enquête a changé | Taille | Exécutant |
|---|---|---|---|---|---|
| **A1** | 11 | Musique de fin du rejeu 2D | Cause probable : le lecteur de la page soumet les sons de fin au plafond de 8 voix, la voix passe et la musique est refusée ; l'export en est exempté depuis le 29/08. Aucune option « musique » n'existe | S | opus-medium |
| **A2** | 12 | Son perdu au rechargement ou en changeant de rejeu | Le correctif du 27/08 ne réveille le son que sur « Lecture » / « Recommencer » ; les sons ne se chargent qu'après ce geste | S | opus-medium |
| **A3** | 5 | Dérogations mortes + README des graphes | 16 dérogations mortes (et non 1) ; le script ne tourne pas en CI | XS | sonnet-medium |
| **A4** | 7 | Musique d'intro au lancement | Le préambule d'1 s avant le coup d'envoi (livré le 02/09) accueille la queue sans attente ; piste source en 4 canaux | S | opus-medium |
| **A5** | 6 | Fins multi-équipes par couleur | **BLOQUÉ** : aucun match multi-équipes réel dans les données ; la table équipe → couleur annoncée n'existe nulle part | M | opus-medium |
| **B1** | 1 | Classement mondial perdu au boot | Pas de délai de 30 s (tir immédiat) ; `ErrDrainTimeout` n'existe pas | S | opus-high |
| **B2** | 3 | Découpeur SQL unique | 4 découpeurs qui exécutent du SQL (règle n°6 franchie), pas 2 | S | opus-high |
| **B3** | 2 | Index ART de `match_skill_rank` | 3 index, pas 2 ; des lectures brutes EMPRUNTENT l'index → lectures fausses prouvées le 13/09 | M | opus-high |
| **B4** | 4 | Garde-rail d'exclusion de la campagne | Les cas cités sont réglés ; l'angle mort est plus large : ~20 lecteurs hors garde, dont ~10 à corriger | M-L | opus-high |
| **B5** | 8 | Mode démo hermétique (fichiers) | Risque SOUS-ESTIMÉ en local : `monitoring.duckdb` ouvert en écriture, WAL réel rejoué dans les vraies player DB, fichiers réels supprimés | M | opus-high |
| **B-R** | — | Revue adversariale de la piste B | — | — | opus-high |

Deux familles de lots : **A = rejeu et web** (aucune commande Go), **B = backend Go**.
**UN SEUL agent à la fois** (consigne de l'utilisateur au GO du 2026-09-26), supervisé, dans le
worktree dédié du plan. Coût estimé : 10 à 12 lancements d'agents, l'un après l'autre.

---

## 1. Cadrage

### 1.1 Objectifs et critères de succès (mesurables)

| Item | Critère |
|---|---|
| 11 | Sur un rejeu dont la fin est dense, la musique de fin démarre (tampon de 9,93 / 10,59 / 11,67 s observé au démarrage) alors que 8 sons sont déjà en cours ; sur une fin calme, rien ne change. Écoute validée par l'utilisateur. |
| 12 | Préférence « son activé » : après un rechargement, le son est audible au plus tard 2 s après le premier geste sur la page, quel qu'il soit, sans toucher au bouton du son. En passant d'un rejeu à un autre sans rechargement, il est audible dès la lecture. Vérifié dans Chromium et Firefox, puis par l'utilisateur. |
| 5 | Le script échoue sur toute dérogation qui ne sert plus. `ALLOWED_CROSS_IMPORTS` ne contient plus aucune dérogation morte. Le README décrit l'emplacement réel des 12 wrappers. Le script tourne en CI. |
| 7 | « Lecture » depuis le préambule joue une seule fois la queue de la montée : la résolution tombe au coup d'envoi (±50 ms), à −18 LUFS. Rien ne se joue après une pause, un glissé de frise, un saut ou un lien tactique, ni dans l'export. Écoute validée par l'utilisateur. |
| 6 | (si débloqué) Sur un match réel à 3 équipes ou plus : écran de fin et réplique de l'annonceur à la couleur de l'équipe gagnante. FFA et matchs à 2 équipes inchangés. Visuel et écoute validés par l'utilisateur. |
| 1 | Premier tir du cron 2 min après le boot. Une vidange expirée (`ErrDrainTimeout`, erreur typée) donne jusqu'à 3 tentatives sans nouveau scrape. Une autre erreur ne donne aucune nouvelle tentative. Prouvé par test. |
| 3 | Un fragment fait uniquement d'un commentaire en fin de DDL passe `EnsurePlayerSchema` (test rouge avant, vert après). Un seul découpeur SQL dans le module, tenu par un garde-rail. |
| 2 | Mesure versionnée avec et sans index. Plus aucun `idx_msr_*` créé par une autorité non scellée, ni présent dans une player DB après migration ou `EnsurePlayerSchema`. Sonde, outil de réparation et `indexcheck` supprimés. Ratchet étendu. |
| 4 | Le garde balaie toutes les déclarations des cinq racines, y compris les littéraux locaux et `xuid IN (`. Chaque lecteur entrant est statué : correction avec un test de comportement rouge avant, ou dispense datée et justifiée. Aucune dispense périmée. |
| 8 | En mode démo, aucun fichier n'est créé, modifié ou supprimé hors de la racine démo (test sur un leurre + preuve de bout en bout). Les lectures hors racine se limitent à une liste explicite. |

### 1.2 Organisation

- **Une branche, un worktree** : `feat/backlog-2026-09-26`, worktree `../LevelUp-wt-backlog`,
  partis de `origin/feat/v75`. Les branches `feat/**` déclenchent la CI au push, pas les `wt/**`
  (cf. `reference-wt-branches-no-ci`).
- **Un seul agent à la fois** : un exécutant par lot, lancé seulement quand le lot précédent est
  vérifié et clos. La revue adversariale est aussi un agent : elle ne tourne jamais en même temps
  qu'un exécutant.
- **Ordre** : A1 → A2 → **fusion 1** → B1 → B2 → B3 → B4 → B5 → B-R (+ corrections) →
  **fusion 2** → A3 → A4 → A5 (si débloqué) → **fusion 3**.
  - Fusion 1 : les deux défauts du son, fonction de v7.5 à livrer avant la fusion v7.5 → main.
  - Fusion 2 : le backend, après la revue adversariale.
  - Fusion 3 : le reste.
- Chaque fusion se fait dans `feat/v75` (`merge --no-ff`) par le superviseur, après la CI de la
  branche verte au niveau job. La branche continue après une fusion.
- **Seule dérogation d'ordre prévue** : A2.0 exige un navigateur ET le serveur API. Si le serveur
  ne peut pas tourner quand A2 arrive (un autre processus tient les bases), B1 passe devant et A2
  reprend dès que le serveur tourne. La fusion 1 attend A2. La dérogation est consignée au
  journal.

### 1.3 Mise en place (superviseur, au GO)

1. `git fetch origin`, puis
   `git worktree add ../LevelUp-wt-backlog -b feat/backlog-2026-09-26 origin/feat/v75`.
2. Y copier ce fichier et le committer seul :
   `docs(.ai): plan backlog 2026-09-26 (items 1-8, 11, 12)`. Supprimer ensuite la copie non suivie
   du checkout principal (le principal est partagé et n'est pas touché).
3. `npm ci` dans `../LevelUp-wt-backlog/apps/web`.
4. **Serveur API pour les recettes navigateur** : celui du checkout principal sur `:8000`, démarré
   par le superviseur s'il ne répond pas (`Start-Process` détaché, une seule instance), après
   avoir vérifié qu'aucun autre processus ne tient les bases. Jamais par un exécutant, et jamais
   un binaire de worktree sur les données réelles.
5. Lancer A1.

### 1.4 Hors périmètre (explicite)

- Entrées 9 et 10 du backlog, et les entrées « gardées de côté » (réglage du janitor, Tauri,
  coach/Prestige).
- Refonte de `PathResolver` avec une racine de données distincte de la racine de configuration
  (~70 fichiers, 121 appels) : c'est le « lot C » de l'enquête de l'item 8. Il sera rattaché à
  l'entrée Tauri du backlog, qui a le même besoin d'inventaire.
- Lecteurs de `sync/**`, `ops/**` et `cmd/**` sans exclusion de la campagne : pipelines dérivés,
  triage séparé. La liste relevée est versée au backlog (B4.5).
- Rendre le découpeur SQL sensible aux chaînes `'…'` et aux commentaires `/* */` (B2.6 : limite
  écrite, entrée au backlog).
- Découvertes de l'enquête, §6.

---

## 2. Décisions tranchées (objection possible au GO, fermes ensuite)

- **D-1 (item 11).** Les sons de fin de match (voix d'annonceur et fanfare) échappent au plafond
  de voix dans le lecteur de la page, comme dans l'export depuis le 29/08, et n'occupent pas de
  place. La voix « manche terminée » reste soumise au plafond, comme dans l'export. Aucune
  option « musique » n'est ajoutée : il n'en existe pas (vérifié le 2026-09-26 ; les catégories
  de sons que l'on peut couper ne filtrent pas les sons de fin).
- **D-2 (item 1).**
  - Erreur sentinelle `ErrDrainTimeout`.
  - Nouvelle tentative bornée : 3 essais au total, délai de 30 s, uniquement sur cette erreur.
    Le scrape est gardé en mémoire.
  - Premier tir différé de 2 min, sur le patron `bootDelay` de `asset_name_sweep_cron` (2e copie,
    sous le seuil de la règle n°6).
  - Aucun signal « boot chaud » n'est créé : il n'en existe pas, et en inventer un sortirait du
    périmètre.
- **D-3 (item 3).**
  - Le cœur canonique est celui de `migration`, qui gagne une variante avec contexte.
  - `sync.execScript` devient un délégué d'une ligne, ce qui préserve les ~25 appels de test.
  - Aucun changement de sémantique du découpage : les DDL actuels ne contiennent ni `;` en
    commentaire, ni `--` dans un littéral, ni `/* */`.
- **D-4 (item 2).**
  - ~~Retrait des trois index si aucune forme de lecture ne dépasse 10 ms sans index sur une DB
    fichier de 12 000 lignes.~~ **AMENDÉ le 2026-09-27 par le superviseur** (STOP de B3.1) :
    retrait des trois index si, pour chacune des sept formes, la médiane **sans index ≤ médiane
    avec index + 2 ms**, sur une DB fichier de 12 000 lignes, plan lu par `EXPLAIN ANALYZE`
    (DB-13 : `EXPLAIN` seul ne montre jamais l'index). Sinon, arrêt et rapport.
  - **Raison de l'amendement** : le seuil absolu mesurait le coût de lecture de 12 000 lignes
    par le client Go et des listes `IN`. Ce coût est identique avec et sans index : même plan
    séquentiel pour F1-F3, sur un poste chargé à 100 % par une autre session. L'intention de D-4
    était de vérifier que le retrait ne ralentit rien, comme le relevé PSA (0,800 contre
    0,841 ms, même plan). Sur la mesure, le plus grand écart est de +1,1 ms. La seule forme qui
    emprunte un index (F4) est 10 fois plus RAPIDE sans lui, et la forme de l'incident du 13/09
    (C1/C2) est à égalité.
  - Les `DROP INDEX IF EXISTS` entrent aussi dans l'autorité rejouée à chaque ouverture, pour
    MSR **et** pour PSA. Raison : un binaire plus ancien (autre worktree, retour arrière) recrée
    les index par son `CREATE INDEX IF NOT EXISTS`, et la migration one-shot ne rejoue jamais.
    Sans la sonde, ce retour passerait inaperçu.
  - La baseline scellée reste intacte, avec une dispense datée. Le harnais `psarepro` est
    conservé.
- **D-5 (item 4).**
  - Le garde couvre `internal/platform/duckdb/**`, `internal/progression/**`,
    `internal/api/wire`, `internal/service/**` et `internal/analysis/**`.
  - Le critère inclut `xuid IN (`.
  - Les exclusions s'appliquent partout où un lecteur d'affichage agrège des matchs d'un
    joueur : c'est la règle produit du 2026-07-18. Le résolveur est neutre pour un titre sans
    variante de campagne, donc aucune comparaison de slug n'est nécessaire.
  - La lecture croisée d'un autre titre utilise `excludeAllCampaignByMatchID`.
  - Conséquence assumée : les paliers, les deltas, le prestige et les rencontres de Halo 5
    changent là où la campagne était comptée.
- **D-6 (item 5).** Toutes les dérogations mortes sont retirées. Le script détecte lui-même les
  dérogations qui ne servent plus, et il est ajouté au job web de la CI.
- **D-7 (item 8).**
  - Coupure en démo des tâches de fond qui écrivent ou suppriment hors de la racine, et
    dérivation des chemins de configuration et d'exécution depuis la racine démo.
  - Pas de connexion en démo : dossier `auth` vide sous la racine démo.
  - `.env.local` reste lu depuis `repoRoot`, parce qu'il porte `LEVELUP_DEMO_MODE` lui-même.
  - Le cache d'assets réel peut être lu, pas écrit, comme dans compose.
  - Magasin de monitoring en mémoire.
  - Écritures d'exécution sous `<racine démo>/runtime/`, ignoré par git.
- **D-8 (item 7).**
  - Extrait de la piste `402178411` : [R − 1,0 s ; R + 2,0 s] (borné à la fin de piste), où R
    est l'instant de résolution. Fondu d'entrée de 100 ms. −18 LUFS, −1 dBTP, stéréo PCM
    16 bits 48 kHz.
  - Déclenché seulement par « Lecture » ou « Recommencer » depuis le préambule, et par
    « Lecture » en fin de rejeu (rembobinage).
  - Absent de l'export. Une itération sur le point de coupe est permise après l'écoute de
    l'utilisateur.
- **D-9 (item 6).**
  - L'écran reste celui du joueur regardé : Victoire ou Défaite, avec le jeton `team-ally`.
    Cela suit l'amendement du 2026-08-26 et la décision D1 du 2026-08-28, postérieurs à l'entrée
    du backlog.
  - En défaite, une ligne « Victoire de l'équipe <Nom> » est ajoutée, avec le logo du gagnant.
  - La voix est la réplique de la couleur du gagnant pour TOUS les points de vue, comme en jeu ;
    la fanfare suit l'issue du joueur.
  - FFA et matchs à 2 équipes sont inchangés.
  - Une couleur non établie n'est pas mappée : le comportement actuel sert de repli.
- **D-10 (item 12).** Comportement attendu avec la préférence « son activé » :
  - le lecteur audio s'ouvre au PREMIER geste sur la page du rejeu, quel qu'il soit (clic,
    touche), et plus seulement sur « Lecture » ou « Recommencer » ;
  - il s'ouvre dès l'affichage si le document a déjà reçu un geste
    (`navigator.userActivation.hasBeenActive`, cas du passage d'un rejeu à un autre sans
    rechargement) ;
  - jamais de son sans geste préalable sur le document (politique d'autoplay des navigateurs) ;
  - le premier clic sur le bouton du son continue d'activer au lieu de couper (correctif du
    27/08) ;
  - si la mesure A2.0 montre que le chargement des sons dépasse 2 s après le geste, les fichiers
    se téléchargent dès l'affichage quand la préférence est « activé » (sans contexte audio), et
    se décodent à l'ouverture du lecteur.

---

## 3. Contrat d'exécution

### 3.1 Règles

1. Les 10 règles du skill `plan-execution` s'appliquent : ordre strict dans une piste, étape
   commencée = étape terminée, aucun report d'une action exécutable, vérification sur pièces
   avant de coder et avant de cocher, statut pour chaque item (`[x]` fait et vérifié, `[~]`
   couvert par un autre item à nommer, `[!]` non traité avec justification), zéro correctif hors
   périmètre.
2. **Clôture d'un lot** :
   - gate passé, avec la sortie consignée ;
   - tous les items statués ;
   - section du lot et journal de SA famille de lots (A ou B) mis à jour dans ce fichier, dans le commit ;
   - rapport au superviseur (fait / gates avec les lignes `EXIT_*` / découvertes / écarts).
   Le thought_log, le backlog et le registre des reports relèvent du superviseur (§3.4).
3. **Test rouge d'abord** : tout correctif porte un test qui échoue sur le code d'avant. Sa
   sortie rouge est consignée dans le rapport.
4. **STOP = remonter** : un STOP posé par un lot se remonte au superviseur, même si le jugement
   local semble bon. Il n'y a ni contournement ni adaptation discrète du périmètre.
5. **Découvertes** : section §6 de SA famille de lots, rien n'est traité.
6. **Fichiers d'une branche active voisine** : `feat/perf-perimetre` modifie `explorer_repo.go`,
   `squad_repo_annuaire.go`, `relations_repo.go`, `match_view_repo_*`,
   `no_raw_rating_reads_test.go` et `queries_*`. `feat/suite-audit-decodeur*` modifie
   `wire/registry_replay_build.go`, `replaybuild/*` et `wire/server_admin_monitoring.go`. Ces
   fichiers ne sont PAS modifiés par ce plan. Si un lot en a besoin : STOP.

### 3.2 Environnement

- **Go** (piste B uniquement ; la piste A ne lance aucune commande `go`) :
  - PowerShell, depuis `apps/go-api` du worktree :
    `$env:CGO_ENABLED = "1"; $env:GOCACHE = "$env:LOCALAPPDATA\go-build-backlog"`.
  - Le cache dédié évite les collisions avec les builds des autres sessions. Ne pas toucher à
    `CC` : le gcc winlibs de l'environnement est le seul qui lie DuckDB.
  - Une seule commande `go` à la fois.
  - Jamais `-race`.
- **Web** :
  - Purger `apps/web/node_modules/.tmp` avant tout typecheck de clôture (faux vert
    incrémental).
  - vitest hors bac à sable, avec `--pool=forks`.
  - `git checkout -- apps/web/src/routeTree.gen.ts` avant le staging.
  - Recettes navigateur : scripts Playwright ad hoc `apps/web/.tmp.*.mjs`, supprimés après usage.
    Chromium est installé ; Firefox s'installe par `npx playwright install firefox` (autorisé
    pour A2).
- **Gates** :
  - Sortie redirigée vers un log hors du dépôt (`$env:TEMP\backlog-gates\<lot>-<gate>.log`).
  - Code de sortie capturé sur la commande elle-même (`"EXIT_X=$LASTEXITCODE"`), jamais à
    travers un pipe.
  - Échecs Go cherchés avec le motif ancré `^--- FAIL:`.
- **Git** :
  - Staging explicite des fichiers du lot (jamais `git add -A`), jamais `git stash`.
  - Commits préfixés `backlog(<lot>): …` et terminés par
    `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
  - **Aucun push par l'exécutant.**
- **Aucune attente d'arrière-plan** : gates en avant-plan. Un résultat dont le rapport dépend ne
  s'attend jamais en tâche de fond.
- **Pas de Python** (ffmpeg pour l'audio : WinGet `Gyan.FFmpeg` 8.1.2), pas de SQLite.

### 3.3 Gates standard

- **GO-S** (`apps/go-api`) :
  - `go build ./...`
  - `go vet ./...`
  - `go test -count=1 <paquets du lot>`
  - `go test -tags=integration -count=1 -p 1 <paquets du lot>`
  - depuis la racine du worktree : `make go-api-lint`
- **GO-F** (lots qui touchent `sync/`, `migration/` ou `persist/`) : GO-S, plus
  `go test -count=1 ./...` et `go test -tags=integration -count=1 -p 1 ./...` complets.
- **WEB** (`apps/web`) : `npm run typecheck` (après purge), `npm run lint`, puis
  `node_modules/.bin/vitest run --pool=forks <filtres du lot>`. Suite complète à la clôture de
  la piste A.

### 3.4 Rôles

- **Exécutant** (un par lot, seul agent actif, dans le worktree du plan) :
  - code, tests, commits locaux ;
  - section de son lot + journal et découvertes de sa famille de lots (A ou B) dans ce fichier.
  - Ne touche jamais `.ai/thought_log.md`, `.ai/BACKLOG.md` ni `.ai/V7.5/REGISTRE_REPORTS.md`.
- **Superviseur** :
  - vérifie le rapport sur pièces, rejoue les gates, pousse la branche, lit la CI au niveau job
    (`gh run view <id> --json status,conclusion,jobs`) ;
  - fusionne dans `feat/v75` (`merge --no-ff`, pousse) ;
  - tient le thought_log, le backlog (entrée close déplacée dans « Récemment complété », entrée
    réécrite si besoin, découvertes versées) et le registre des reports ;
  - lance le lot suivant ;
  - supprime le worktree à la clôture du plan : jonctions retirées d'abord,
    `git worktree remove` sans `--force`.

### 3.5 Reprise de session

Ce fichier fait foi. Relire §3, puis le journal (§7, lots A et B), puis reprendre à la première
case non statuée du lot courant. Les décisions du §2 ne se re-décident pas.

---

## 4. Lots A — rejeu et web

### A1 — Musique de fin du rejeu 2D (item 11) — S

**Pièces** (dossier `apps/web/src/features/match-replay/`) :
- `sound/useReplaySound.ts:580-588` (`playEndMatch`) joue `[voix tirée, musique]` par
  `player.play(url)`.
- `sound/replayAudio.ts:315-320` : `play` refuse EN SILENCE quand
  `voices >= SOUND_MAX_VOICES` (8, `:83`).
- `sound/replayAudioMix.ts` : `MixedSound.conclusion` fait échapper la conclusion au plafond
  dans l'export (commit `206b36109`, 29/08). Le lecteur de la page n'a jamais reçu cette
  exception.
- Le commentaire de `applyVoiceCap` (« même comptabilité que le lecteur temps réel ») est donc
  faux.
- Les 14 WAV `end_*` sont en PCM stéréo 48 kHz 16 bits, et le garde-rail « décodable » est
  présent (`replaySoundAssets.guard.test.ts:557-584`).
- Déclenchement : `hooks/useReplayPlayback.ts:340-354`, uniquement quand la LECTURE franchit la
  borne de fin (décision D-C1).
- Cause probable (H1, non prouvée à l'oreille) : les tirs de fin de match remplissent les 8
  places. La voix prend la dernière, et la musique, jouée en second, est refusée. Cela colle au
  constat de l'utilisateur (voix entendue, résultat affiché, pas de musique).
- Aucune option « musique » n'existe dans le rejeu : les catégories de sons que l'on peut couper
  (`replay-sound-categories-off`) ne filtrent pas les sons de fin, et aucun réglage ne vise la
  musique (vérifié le 2026-09-26).
- Fichiers : `useReplaySound.ts` fait 675 lignes (au-dessus du seuil, gelé par la baseline) :
  **il ne doit pas grossir**. `replayAudio.ts` fait 362 lignes.

- [x] **A1.0 Confirmation déterministe AVANT tout code, sur la fixture réelle versionnée**
  `test/fixtures/go/replay_schema_71_000d5950.json.gz` (match `000d5950`, Slayer, JGtm
  équipe 0).
  - La fixture est chargée par le chargeur de fixtures existant, jamais par un nom de fichier en
    dur : `feat/suite-audit-decodeur-j4` fait passer ces fixtures au schéma 72.
  - Préféré : la piste sonore réelle du match est jouée jusqu'à la borne de fin à travers
    `useReplaySound`, avec un faux `AudioContext` dont l'horloge avance et qui déclenche `ended`
    à l'heure d'arrêt de chaque source. On relève le nombre de voix occupées à la borne, puis
    `endMatch()`.
  - Repli, si la simulation d'horloge sort du lot : compter, avec le planificateur réel et les
    enveloppes réelles, les sons dont la plage couvre l'instant de fin.
  - **H1 confirmée** si 8 voix sont occupées à la borne et que la musique est refusée alors que
    la voix passe. La version « préférée » devient le test de non-régression de A1.4 (rouge
    avant le correctif). Chiffres au journal.
  - **H1 réfutée : STOP**, rapport, aucun code. En particulier, une musique refusée alors que
    moins de 8 voix sont occupées signale une fuite du compteur ou une autre cause, que ce lot
    ne corrige pas.
  - Le navigateur n'est pas nécessaire ici. Au GO, une autre session faisait tourner
    `replay-equiv` et des builds Go : le serveur API n'est démarré que lorsque plus aucun
    processus ne tient les bases (§1.3).
  - **Statut (2026-09-26)** : H1 CONFIRMÉE à `SOUND_MAX_SPEED` (2×), RÉFUTÉE à 1× sur ce témoin
    — chiffres et portée au journal (§7, lots A) et en DA-3. Version « préférée » réalisée :
    `sound/endMatchVoiceCap.witness.test.tsx`.
- [x] **A1.1** `replayAudio.ts` : méthode `playConclusion(url)`. Elle joue hors plafond, avec
  l'enveloppe normale, sans compter de voix. La règle est nommée une seule fois, partagée avec
  `replayAudioMix.ts` (constante ou doc commune).
- [x] **A1.2** `useReplaySound.ts:586` : `player.play(url)` devient `player.playConclusion(url)`,
  sans ligne nette ajoutée.
- [x] **A1.3** Commentaire de `applyVoiceCap` (`replayAudioMix.ts:222-233`) rendu exact.
- [x] **A1.4 Tests** (rouges d'abord) :
  - (a) `useReplaySound.test.tsx`, bloc « la fin de partie » : 8 sons en cours puis `endMatch()`
    donnent 10 sources, dont `end_victory_music_01.wav`. Aujourd'hui : 8.
  - (b) `replayAudio.test.ts` : 8 `play` puis 2 `playConclusion` donnent 10 sons ; un `play`
    ordinaire ensuite est toujours refusé.
  - (c) Garde-rail de parité : la même situation saturée passée par `applyVoiceCap` (export) et
    par le lecteur (page) garde la conclusion des deux côtés.
  - (d) Le test de A1.0 sur la fixture réelle : musique refusée avant, jouée après.
  - **Statut** : (a) `useReplaySound.test.tsx:418`, (b) `replayAudio.test.ts:169`,
    (c) `replayAudioMix.test.ts:160`, (d) `endMatchVoiceCap.witness.test.tsx:195` (+ témoin 1×
    `:203`, vert avant et après). Sorties rouges au journal.
- [x] **A1.5 Recette navigateur**, seulement si `http://127.0.0.1:8000/health` répond (le
  superviseur démarre le serveur, jamais l'exécutant). Sinon, l'item reste ouvert au rapport et le
  superviseur le fait avant la fusion 1.
  - **Statut (2026-09-27)** : FAITE par l'exécutant de A2 (serveur démarré par le superviseur),
    **avec un témoin dense SUBSTITUÉ** : sur les données de ce poste, `000d5950` ne joue AUCUNE
    conclusion, à 1× comme à 2× (vue match partielle `scoreboard_empty`, issue « défaite » :
    `endMatchSoundSpec` rend `null`, aucune prise `end_*` n'est même téléchargée). Témoin dense de
    remplacement `ac03413d` (victoire de Chocoboflor) : voix ET fanfare démarrent à 2× et à 1×,
    7 à 8 sources actives à la borne. Chiffres au journal, écart en DA-5.
  - **Statut (2026-09-26)** : NON FAITE par l'exécutant — `/health` a rendu `000` (serveur
    arrêté). Reste au superviseur avant la fusion 1. ATTENTION (A1.0) : à 1× le témoin dense
    jouait déjà la fanfare ; la recette ne discrimine le correctif qu'à 2× (`SOUND_MAX_SPEED`).
  - Script Playwright ad hoc `apps/web/.tmp.recette-fin.mjs` (supprimé après) : Chromium,
    `--autoplay-policy=no-user-gesture-required` (pour isoler l'item 11 ; la recette de A2 garde
    la politique par défaut), Vite du worktree avec proxy vers `:8000`.
  - Son activé (`localStorage replay-sound-on`), vitesse 1×, curseur environ 20 s avant la fin ;
    la lecture doit franchir la borne.
  - Instrumentation de `AudioBufferSourceNode.prototype.start` (durée du tampon) et du nombre de
    sources réellement actives (écouteur `ended` ajouté par `addEventListener`).
  - Témoins : `000d5950-83d9-423f-ab55-d068a7237b9f` (fin dense : la musique démarre) et un match
    du cache dont les 5 dernières secondes portent au plus 2 tirs (fin calme : inchangé). Sortie
    au journal.
- [ ] **A1.6 Gate utilisateur (écoute)** : le témoin dense, et le match sur lequel l'utilisateur a
  constaté le défaut (qu'il nomme).
  - Laissé ouvert par l'exécutant (gate du superviseur). Question à poser avec : à quelle
    VITESSE le défaut a-t-il été constaté (cf. DA-3) ?
  - Réponse de l'utilisateur (2026-09-26) : il regarde à 1× ou à 2× au plus. L'écoute se fait
    donc aux DEUX vitesses. Si la fanfare manque encore à 1× après le correctif, c'est une autre
    cause : elle est rapportée, pas corrigée dans ce lot.

**Gate** : WEB (filtres `src/features/match-replay/sound`), recette A1.5, gate utilisateur A1.6.

### A2 — Son perdu au rechargement ou en changeant de rejeu (item 12) — S

**Pièces** (dossier `apps/web/src/features/match-replay/`) :
- `sound/useReplaySound.ts:10-26` (en-tête « DEUX ÉTATS, PAS UN », correctif du 27/08) : la
  PRÉFÉRENCE (`replay-sound-on`, persistée) revient du stockage local, le LECTEUR
  (`AudioContext`) ne peut naître que dans un geste.
- Aujourd'hui, seuls trois gestes ouvrent le lecteur :
  - « Lecture » et « Recommencer », via `onTransportGesture` (`hooks/useReplayPlayback.ts:434-449`,
    câblé à `sound.wake` dans `ui/ReplayCanvas.tsx:645`) ; la touche espace passe par
    `togglePlay` (`hooks/useReplayShortcuts.ts:117`) ;
  - le premier clic sur le bouton du son, qui ouvre au lieu de couper (`toggle`,
    `useReplaySound.ts:462-495`).
- `wake` (`:457-460`) ne fait rien si la préférence est « coupé » ou si un lecteur existe.
- Aucun son n'est chargé tant qu'aucun lecteur n'existe. `openPlayer` (`:425-446`) lance le
  préchargement de tous les sons du match au moment du geste, et un son demandé avant la fin de
  son chargement est sauté (« un son en retard sur son image est pire qu'un son manqué »).
- Le lecteur est détruit au démontage du composant (`:419-423`).
- Lecture automatique : réglage `replay-autoplay` (`settings/useReplaySettings.ts:72-73`, coupé
  par défaut). Activée, la lecture démarre sans aucun geste, donc sans lecteur.
- Le navigateur de l'utilisateur est Firefox. Playwright n'a que Chromium d'installé
  (`%LOCALAPPDATA%\ms-playwright`).
- Hypothèses, à trancher par la mesure :
  - (a) l'utilisateur démarre par un chemin qui n'ouvre pas le lecteur (lecture automatique,
    clic sur la frise, lien tactique) ;
  - (b) le lecteur s'ouvre bien, mais tous les sons des premières secondes sont sautés le temps
    du chargement, ce qui ressemble à une panne ;
  - (c) propre à Firefox : le contexte créé dans le geste reste suspendu ;
  - (d) le passage d'un rejeu à un autre démonte le composant, et le document, qui a pourtant
    déjà reçu un geste, n'en profite pas.

- [x] **A2.0 Reproduction AVANT tout code**, avec la politique d'autoplay PAR DÉFAUT (jamais
  `no-user-gesture-required`), dans Chromium et dans Firefox (`npx playwright install firefox`
  s'il est absent).
  - Préférence « son activé » posée dans le stockage local.
  - Chemins mesurés :
    1. rechargement complet puis « Lecture » ;
    2. rechargement puis touche espace ;
    3. rechargement puis clic sur la frise, puis lecture ;
    4. rechargement avec la lecture automatique activée ;
    5. passage d'un rejeu à un autre par un lien, sans rechargement.
  - Mesures pour chaque chemin : état de l'`AudioContext` après le geste, délai entre le geste et
    le premier `start()` d'une source (sur un passage où la piste porte des sons), nombre de sons
    sautés faute de chargement.
  - Verdict par chemin et par navigateur au journal. Si aucun chemin ne reproduit le défaut :
    **STOP**, rapport.
  - **Statut (2026-09-27)** : défaut REPRODUIT sur le chemin 4 (lecture automatique) et sur le
    chemin 5b ajouté (lecture automatique + lien vers un autre rejeu), dans les deux navigateurs :
    aucun contexte, aucun son, quel que soit le clic hors transport. Chemins 1, 2, 3 et 5 (avec
    « Lecture » ou Espace) : son en 0,19 à 2,2 s. Cause retenue : hypothèse (a), et (d) pour le
    lien. Décodage des sons ≤ 1,1 s (une passe à 2,6 s sous charge) : pas de téléchargement
    anticipé. Tableau au journal.
- [x] **A2.1 Correctif**, selon la cause mesurée et dans le cadre de D-10 :
  - ouverture au premier geste sur la page (écouteur unique `pointerdown` / `keydown` au niveau du
    document, posé seulement si la préférence est « activé » et qu'aucun lecteur n'existe,
    retiré ensuite) ;
  - ouverture dès l'affichage si `navigator.userActivation?.hasBeenActive` ;
  - téléchargement anticipé des sons (décodage à l'ouverture) seulement si A2.0 mesure un délai
    supérieur à 2 s.
  - **Statut** : `sound/useAudioUnlock.ts:40` (ouverture dès l'affichage si `hasBeenActive`),
    `:44-48` (écouteur unique, retiré au premier geste). **Écart** : `click` et `keyup` au lieu de
    `pointerdown` et `keydown`. Ces deux-là ouvrent le lecteur AVANT la bascule du bouton du son
    (et avant la touche M, écoutée par la fenêtre), qui le couperait : régression du 27/08 prouvée
    par test (journal). Pas de téléchargement anticipé (A2.0 : décodage < 2 s).
- [x] **A2.2** `useReplaySound.ts` ne grossit pas : la logique d'ouverture vit dans un module
  dédié (par exemple `sound/useAudioUnlock.ts`). L'en-tête « DEUX ÉTATS » est mis à jour avec la
  nouvelle liste des gestes.
  - **Statut** : 675 lignes avant, 675 après (`useAudioUnlock(on, wake)` en `:460`, en-tête
    « DEUX ÉTATS » `:18-23` réécrit avec la nouvelle liste des gestes).
- [x] **A2.3 Tests** (rouges d'abord, jsdom) :
  - préférence « activé », aucun lecteur : un `pointerdown` sur le document ouvre le lecteur une
    seule fois, et l'écouteur est retiré ;
  - `userActivation.hasBeenActive` vrai à l'affichage : le lecteur s'ouvre sans geste ;
  - préférence « coupé » : rien ne s'ouvre, rien ne se télécharge ;
  - non-régression du 27/08 : le premier clic sur le bouton du son ouvre et ne coupe pas.
  - **Statut** : `sound/useAudioUnlock.test.tsx` (9 cas, à travers `useReplaySound`) : clic `:81`,
    touche `:94`, écouteur retiré `:103`, `hasBeenActive` `:112` et `:121`, bouton du son `:136`,
    touche M `:149`, démontage `:159`, préférence « coupé » `:168`. Le `pointerdown` du texte est
    remplacé par `click` (écart de A2.1). Sorties rouges au journal.
- [!] **A2.4** Recette A2.0 rejouée après correctif : pour chaque chemin et dans les deux
  navigateurs, son audible au plus tard 2 s après le geste. Sortie au journal.
  - **Statut (2026-09-27)** : Chromium, 7 chemins sur 7 sous 2 s. Firefox : chemins 1, 2, 3, 5 et
    5b sous 2 s. Chemins 4 et 7 (lecture automatique, puis clic ou touche neutre) : le lecteur
    s'ouvre bien au geste (0 contexte avant, 1 après), mais le contexte de Firefox ne passe
    « running » que 0,8 à 4,3 s plus tard (4 passes : 0,8 / 2,2 / 2,9 / 4,3 s). La même latence a
    été mesurée AVANT le correctif sur le chemin 1 (0,8 à 1,4 s, et une passe sans démarrage en
    6 s). Le poste était chargé à 100 % par les tests Go du superviseur. La page ne peut pas créer
    le contexte avant le geste. Critère « ≤ 2 s » non tenu dans Firefox sous charge : A2.5 tranche
    à l'oreille (DA-7).
- [ ] **A2.5 Gate utilisateur** (Firefox, son usage réel) : recharger un rejeu avec le son activé
  donne du son sans toucher au bouton du son ; passer à un autre rejeu aussi.

**Gate** : WEB (filtres `src/features/match-replay`), recette A2.4, gate utilisateur A2.5.
**Fusion 1 après A2** : A1 et A2 fusionnés dans `feat/v75` par le superviseur, après la CI de
branche verte au niveau job.

### A3 — Dérogations mortes et README des graphes (item 5) — XS

**Pièces** :
- `tools/lint-cross-feature-imports.mjs` (racine du dépôt) : `ALLOWED_CROSS_IMPORTS`.
  - Entrées mortes : `admin=>lab`, `career=>compare`, `explorer=>compare`, `palmares=>compare`,
    `squad=>compare`, `career=>leaderboard`, `career=>citations`, `friends=>settings`,
    `squad=>settings`, `match-view=>settings`, `lab=>squad`, `squad=>timeseries`,
    `personal-stats=>{filters,synthesis,squad}` (l.170-172 ; le dossier `features/personal-stats/`
    n'existe plus).
  - `auth=>auth` (l.53) est morte par construction : l.303 saute les imports d'une feature vers
    elle-même.
- Le script ne détecte aucune dérogation morte (ratchet des violations seul, l.384 :
  « 7 <= plafond »).
- Il ne tourne qu'au pre-commit (`lefthook.yml:79-81`), ni en CI ni dans `gate-push`.
- `apps/web/src/components/charts/README.md` :
  - l.24 : « Wrappers 10–11 are kept in `features/timeseries/` » ; faux pour `FirstBloodLanes`
    (#11), qui vit dans `components/charts/` et sert squad, timeseries et session-detail ;
  - l.213 : « all 11 wrappers », alors qu'il y en a 12.

- [ ] **A3.1** Détection des dérogations mortes dans le script : collecter les clés réellement
  consommées pendant le balayage (même logique de saut que l.297, `cross-feature-allow`), puis
  `exit 1` en listant les entrées jamais servies. Démontrer le rouge sur l'état actuel (sortie
  au journal).
- [ ] **A3.2** Retirer les entrées que le script déclare mortes, et elles seules (liste
  ci-dessus, re-vérifiée au moment du lot), avec leurs commentaires devenus orphelins.
- [ ] **A3.3** README l.24 (wrapper 10 seul dans `features/timeseries/` ; `FirstBloodLanes` dans
  `components/charts/`, consommateurs cités) et l.213 (12 wrappers).
- [ ] **A3.4** `.github/workflows/ci.yml`, job web : étape `node tools/lint-cross-feature-imports.mjs`
  après `lint:colors`, depuis la racine du dépôt. Aucun déclencheur ni `paths-ignore` modifié
  (invariants `archlint`, vérifiés par la CI).

**Gate** : `node tools/lint-cross-feature-imports.mjs` rouge avant A3.2, puis exit 0 avec
toujours « 7 <= plafond » ; CI de branche verte, étape nouvelle comprise.

### A4 — Musique d'intro au lancement du rejeu (item 7) — S

**Pièces** :
- La piste `C:\Users\Guillaume\Desktop\Halo Infinite - Sons armes\_fin_partie\mus_mp_global_wav\402178411.wav`
  fait 16,00 s, en PCM 48 kHz 16 bits, **4 canaux** : le garde-rail « 2 canaux au plus » la
  refuserait, il faut la réduire en stéréo.
- Même piège pour `_fin_partie/livraison/end_*_music_01.wav`, restés en 4 canaux (les copies du
  dépôt sont les bonnes).
- La page s'ouvre en pause (lecture automatique coupée, `hooks/useReplayPlayback.ts:245`).
- Le curseur est posé à `leadInFrame`, 1 s avant le coup d'envoi (`LEAD_IN_MS`,
  `model/replayWindow.ts:81`).
- `togglePlay` (`:442-449`) rembobine en fin de rejeu. `restart` (`:436-440`) revient à
  `leadInFrame`. Tous deux appellent d'abord `onTransportGesture` (déblocage audio).
- `seekTo` est borné. `openAtFrame` (lien tactique) n'est appliqué qu'une fois.
- La fin est câblée dans `ReplayCanvas.tsx:645` (`onEnded: sound.endMatch`).
- Recette de normalisation : `.ai/V7.5/PLAN_REPLAY_CADRAGE_VICTOIRE.md:328-332` (mesure
  `loudnorm` de ffmpeg, gain = min(−18 − I, −1 − TP) pour une musique, gain linéaire,
  `pcm_s16le` 48 kHz).
- Garde-rails d'assets : `replaySoundAssets.guard.test.ts` (stem ↔ fichier `:115-161`, plafond
  `SOUND_CUT_MAX_S` `:400-406`, décodable `:541-585`).

- [ ] **A4.0** Mesure de la piste avec ffmpeg (énergie par fenêtre de 50 ms, `astats` ou
  `ebur128`) : repérer R, la dernière attaque forte de la montée. Tableau des mesures et R
  retenu au journal.
- [ ] **A4.1** Asset `static/sounds/halo_infinite/intro_music_01.wav` selon D-8 (fondu de sortie
  de 300 ms s'il est coupé avant la fin naturelle). Commande ffmpeg exacte au journal, pour être
  reproductible.
- [ ] **A4.2** `sound/introSound.ts` : `INTRO_MUSIC_STEM`, famille `music` pour les familles
  d'export (non joué dans l'export, décision D-8), préchargement par `soundURLsFor`.
- [ ] **A4.3** `useReplayPlayback.ts` : option `onStarted`, appelée par `togglePlay` et `restart`
  quand la lecture démarre avec `frameRef.current <= leadInFrame`, après l'éventuel rembobinage.
  Jamais sur une reprise, un glissé, un saut, `openAtFrame` ou une lecture automatique.
- [ ] **A4.4** Lecture par la voie ordinaire (`play`), avec le son activé et une vitesse d'au plus
  `SOUND_MAX_SPEED`. Toute ligne ajoutée à `useReplaySound.ts` est compensée par une extraction
  dans le même lot (le fichier ne grossit pas).
- [ ] **A4.5** Garde-rails d'assets étendus au nouveau stem (fichier ↔ stem, durée, décodable).
- [ ] **A4.6 Tests** (rouges d'abord) :
  - `useReplayPlayback.test.tsx` : un seul déclenchement au départ depuis le préambule ; aucun
    après pause/reprise, glissé, saut ou lien ; déclenchement après « Recommencer » et après
    « Lecture » en fin de rejeu.
  - `useReplaySound.test` : son coupé ou vitesse supérieure à 2× donnent le silence.
- [ ] **A4.7** Recette navigateur instrumentée (script de A1.0) : l'extrait démarre une seule
  fois, au clic. Puis **gate utilisateur (écoute)** : la résolution tombe au coup d'envoi. Une
  itération est permise sur R, d'après l'indication de l'utilisateur.

**Gate** : WEB (filtres `src/features/match-replay`), recette A4.7, gate utilisateur.

### A5 — Fins de partie multi-équipes par couleur (item 6) — M — BLOQUÉ

**Condition d'entrée**, vérifiée par le superviseur avant le lancement :
1. Au moins un match réel à 3 équipes ou plus, chaque équipe comptant 2 joueurs ou plus, joué
   par un joueur suivi, synchronisé, et dont le rejeu est cuit.
2. La couleur annoncée par le jeu pour l'équipe gagnante de ce match, relevée par l'utilisateur.

**État au 2026-09-26 : non remplie.**
- Un seul match de ce type dans toute la base (2025-08-13, aucun joueur suivi).
- Les deux matchs « 3+ équipes » des joueurs suivis sont une partie Survive The Undead et une
  Fiesta FFA de 2023.
- Aucun des 111 artefacts de rejeu ne porte d'équipe au-delà de 1.

Si la condition n'est pas remplie quand le plan arrive ici : A5 passe en `[!]` (report valide :
donnée que seul l'utilisateur produit), l'entrée du backlog est réécrite avec cette condition, et
la fusion 3 se fait sans A5.

**Pièces** (dossier `apps/web/src/features/match-replay/`) :
- `model/victoryLogic.ts:122` : `if (camps.length !== 2) return null`.
- Consommateurs : `ReplayVictoryOverlay.tsx:138,161`, `export/exportOverlayPanels.ts:135`,
  `export/useReplayCapture.ts:355`, `export/useReplayExport.ts:262`,
  `sound/endMatchSound.ts:139`.
- Aujourd'hui, un match à 3 équipes ou plus passe par `ffa: true` (`endMatchSound.ts:140-144`) :
  gagné, il joue « Vainqueur » avec la fanfare de victoire ; perdu, il est muet ; il n'a pas
  d'écran.
- `MatchScoreboardRow` porte `team_side`, `is_me`, `rank` et `outcome`
  (`lib/api/types.ts:2063-2108`). Aucune équipe gagnante dans l'en-tête.
- Sur le seul match multi-équipes réel de la base, seule l'équipe gagnante porte `outcome` = 2
  (victoire), ce qui va dans le sens de la règle A5.1.
- `lib/halo/teamNames.ts:17-50` : noms (Eagle … Observer) et couleurs hex qui ne correspondent
  PAS aux 8 couleurs de l'annonceur (un magenta, un doublon orange-rouge). La table équipe →
  couleur annoncée n'existe nulle part dans le dépôt.
- Logos `apps/web/public/titles/halo_infinite/teams/0..8.png` présents.
- 16 répliques extraites et converties (PCM stéréo 48 kHz 16 bits, NON normalisées) sous
  `_fin_partie/annonceur_{fr,en}_wav/`, avec les identifiants de l'entrée du backlog. En EN,
  `92374` = lime et `256805823` = cyan (mal transcrits « line » et « science »). En FR, mauve,
  verte, citron, jaune et orange ne sont transcrits que dans `ecoute_fin_partie.html:953-959`.

- [ ] **A5.0 Table équipe → couleur annoncée.**
  - Recherche hors ligne bornée à une passe dans les fichiers du jeu installé, avec les lecteurs
    de tags et de modules existants.
  - À défaut, table limitée aux couleurs établies par les matchs réels (condition 2).
  - Source de chaque ligne au journal.
  - Une couleur non établie n'est pas mappée.
- [ ] **A5.1** `victoryLogic.ts` : nouvelle lecture
  `readMultiTeamVictory(scoreboard, outcomeCode, subject)` → `{ outcome, mine, winner }`.
  - Multi-équipes = au moins 3 camps et au moins un camp de 2 joueurs ou plus. FFA = chaque camp
    compte un seul joueur (inchangé).
  - Gagnant = le `team_side` commun à toutes les lignes `outcome === 'win'` ; `null` s'il est
    ambigu, et le repli actuel s'applique.
  - `readVictory` reste inchangé : ses 14 cas figent la lecture à 2 équipes.
- [ ] **A5.2** `ReplayVictoryOverlay` selon D-9 : ligne « Victoire de l'équipe <Nom> » /
  « <Name> team wins » en FR et EN dans l'i18n de la feature, logo du gagnant par
  `teamLogoPath`, jetons de couleur uniquement (skill `color-tokens`).
- [ ] **A5.3** Export et capture : `exportOverlayPanels.ts`, `useReplayExport.ts`,
  `useReplayCapture.ts`.
- [ ] **A5.4** Son :
  - `END_TEAM_WIN_VOICE_STEMS` (team_id → locale → stems) pour les lignes établies en A5.0 ;
  - `EndMatchSoundSpec.winnerTeamID` ;
  - `endMatchSounds`, `endMatchSoundStems` et `soundFamiliesFor` étendus ;
  - FFA et 2 équipes inchangés.
- [ ] **A5.5** Assets `end_team_<couleur>_voice_{fr,en}_01.wav` (couleurs établies seulement) :
  −16 LUFS, −1 dBTP, stéréo PCM 16 bits 48 kHz, ffmpeg. Garde-rails : complétude (chaque
  team_id mappé a FR et EN), fichier ↔ stem, décodable.
- [ ] **A5.6 Tests** :
  - `victoryLogic.test` : multi-équipes gagné et perdu, gagnant ambigu, FFA inchangé, 2 équipes
    inchangé ;
  - `endMatchSound.test`, `ReplayVictoryOverlay.test`, `useReplaySound.test`.
- [ ] **A5.7 Gates utilisateur** (visuel et écoute) : le témoin multi-équipes, nommé par
  l'utilisateur ; non-régression sur un match à 2 équipes et un FFA gagné.

**Gate** : WEB (filtres `src/features/match-replay src/lib/halo`), gates utilisateur.

**Fusion 3 (fin du plan)** : suite vitest complète, typecheck purgé, CI de branche verte au
niveau job, fusion dans `feat/v75`, suppression du worktree et de la branche.

---

## 5. Lots B — backend Go

### B1 — Classement mondial perdu au boot (item 1) — S

**Pièces** (`apps/go-api/`) :
- `cmd/server/main.go:1394-1423` lance le cron.
- `Run` appelle `RunOnce` tout de suite (`internal/scheduler/world_leaderboard_cron.go:173`),
  puis un ticker de 24 h.
- Les « 30 s » du backlog ne sont pas un réglage : ~13 s pour construire l'enrichisseur en mode
  Eager, plus ~18 s de découverte et de scrape.
- `runOnceForTitle` (`:239-308`) :
  - scrape hors écriture, puis `persist` (`AcquireWriter`, étiquette
    `world_leaderboard_snapshot`, `:448`) ;
  - en cas d'échec : ERROR et retour, aucune nouvelle tentative, scrape perdu.
- `persistStats` (`:375`) prend aussi l'écrivain.
- Vidange : 5 s (`sharedprovider/provider.go:31`), `waitForDrain`
  (`provider_writer.go:251-263`).
- L'erreur n'est pas typée (`provider_writer.go:57`, `%w` sur `context.DeadlineExceeded`) :
  **`ErrDrainTimeout` n'existe pas** (`errors.go`).
- Le ratchet DT-5 (`archlint/no_error_text_classification_test.go`, présent dans la base)
  interdit de classer une erreur par son texte.
- Seul délai de boot existant : `asset_name_sweep_cron.go:24,104-109`.
- Idempotence :
  - INSERT pur en transaction (`leaderboard_world_repo.go:703-731`) ;
  - vue `world_csr_leaderboard_latest` par lot `max(fetched_at)` : une nouvelle tentative avec
    les mêmes entrées ne crée pas de doublon.
- Les deux messages du 20/09 ne sont plus sur ce disque (le worktree de l'époque est
  supprimé).
- Le cron fait 467 lignes, pour un plafond de 500.

- [x] **B1.1** `ErrDrainTimeout` dans `sharedprovider/errors.go`. À `provider_writer.go:57`,
  envelopper avec `%w: %w` (sentinelle + cause).
  → `errors.go:25-34` ; `provider_writer.go:57-62` : sentinelle posée seulement si le ctx de
  l'appelant est vivant (`ctx.Err() == nil`), sinon `ctx.Err()` enveloppé seul. Doc de
  l'interface `provider.go:105`.
- [x] **B1.2** Nouveau fichier `scheduler/world_leaderboard_persist_retry.go` avec
  `acquireWriterRetry(ctx, label)`, selon D-2.
  - Délai nommé `worldLeaderboardPersistRetryDelay = 30 * time.Second`.
  - `sleep` injectable (patron `historyretry.Sleep`), annulable par le contexte.
  - Sert `persist` ET `persistStats`.
  - WARN à chaque nouvelle tentative (`attempt`, `label`, `err`), un seul ERROR d'échec final.
  → `world_leaderboard_persist_retry.go:56` `acquireWriterRetry` (77 lignes, fonction de 22),
  seam `worldLeaderboardSleep` `:45`, constantes `:33-39`. Branché `world_leaderboard_cron.go:389`
  (`persistStats`) et `:463` (`persist`). L'ERROR final reste celui de l'appelant (un seul).
- [x] **B1.3** Premier tir différé : `worldLeaderboardBootDelay = 2 * time.Minute`, `select`
  annulable, option de test à 0.
  → champ `bootDelay` `world_leaderboard_cron.go:120`, défaut `:154`, `select` `:180-185`.
- [x] **B1.4 Tests** (rouges d'abord) :
  - `errors.Is(err, ErrDrainTimeout)` sur une vidange expirée (`reader_stall_metrics_test.go`,
    `SetDrainTimeoutForTest`, tag `integration`) ; un contexte parent annulé ne donne PAS
    `ErrDrainTimeout`.
  - `TestWorldLeaderboardCron_RetriesPersistOnDrainTimeout` : faux provider, deux
    `ErrDrainTimeout` puis succès → 1 lot inséré, `fetchCalls` inchangé, 2 attentes.
  - Erreur non transitoire → aucune nouvelle tentative.
  - 3 échecs → erreur remontée à `ReportCronRun`, aucune ligne.
  - Délai de boot respecté.
  - Les 14 tests `TestWorldLeaderboardCron_*` existants restent verts.
  → `reader_stall_metrics_test.go:124,157` ; `world_leaderboard_persist_retry_test.go` (6 tests,
  dont `RetryWaitIsCancellable` et `RetriesPersistStatsOnDrainTimeout` en plus de la liste).
  Sorties rouges au journal §7.
- [x] **B1.5** Godoc du cron à jour. `grep -rn world_leaderboard docs/` : tout guide qui décrit le
  cron est corrigé en EN et en FR.
  → en-tête du cron `:24-26`, `Run` `:168`, `runOnceForTitle`, `persist`, `persistStats` ;
  commentaire du câblage `cmd/server/main.go:1388-1390`. `grep` sur `docs/` : 0 occurrence de
  `world_leaderboard` ; aucun guide ne décrit le cron (seuls CHANGELOG/RELEASE_NOTES le citent,
  historiques) → rien à corriger en EN/FR.

**Gate** : GO-S sur `./internal/scheduler/ ./internal/platform/duckdb/sharedprovider/ ./internal/archlint/`.

### B2 — Un seul découpeur SQL (item 3) — S

**Pièces** (`apps/go-api/internal/`) :
- `migration.splitSQL` (`migration/helpers.go:240-267`, `isCommentOnly` `:271`) :
  - ignore un fragment fait uniquement de commentaires, et un `;` dans un `--` ;
  - ne gère ni les chaînes `'…'` ni `/* */` ;
  - exporté en `SplitSQL` (`helpers_export.go:58`) ; `ExecScript` (`:55`) n'a pas de contexte
    (`bootCtx()`, `helpers.go:225`).
- `sync.splitSQL` (`sync/schema.go:532-551`) :
  - coupe sur chaque `;` et passe un fragment de commentaire seul à DuckDB (« empty query ») ;
  - `trimSpace` maison (`:553`) ; `execScript` (`:522-529`) avec contexte et `truncate`
    (`:564`) ;
  - appelants : `EnsurePlayerSchema` (`:350`, `:408` via `recoverPlayerSchemaBoot`) et
    `EnsureSharedSchema` (`:437`, `:453`, `:456`) ; environ 25 appels de test dans une vingtaine
    de fichiers.
- `sync` importe déjà `migration` (`schema.go:32`) : pas de cycle.
- Autres découpeurs qui exécutent du SQL : `sync/skill/sqlexec_helpers_test.go:24` et
  `cmd/diag_exec/main.go:34`. Cela fait 4 copies.
- `migration/arbitration_clocks_utc_guard_test.go:299` analyse du SQL sans l'exécuter
  (dispense).
- Commentaire faux : `migration/steps_player_schema_authority.go:70-75`.
- `sync` est gelé (`sync_root_freeze_test.go` : aucun nouveau fichier dans `sync`). `schema.go`
  fait 569 lignes (baseline).

- [x] **B2.1** `migration.ExecScriptContext(ctx, db, script)`, cœur unique ; `ExecScript`
  délègue. Fait : `migration/helpers.go:231` `execScriptContext` (cœur), `:223` `execScript`
  délègue sous `bootCtx()` ; `helpers_export.go:61` `ExecScriptContext`, `:56` `ExecScript`.
- [x] **B2.2** `sync.execScript` devient un délégué d'une ligne. Suppression de `sync.splitSQL`,
  de `trimSpace`, de `truncate` (s'il n'a plus d'appelant, à vérifier) et de leurs tests
  unitaires (`schema_unit_test.go`). Fait : `sync/schema.go:523` ; `splitSQL`, `trimSpace` et
  `truncate` supprimés (aucun autre appelant dans `sync`, vérifié par grep) ; `schema.go`
  569 → 525 lignes ; `schema_unit_test.go` supprimé (il ne testait que ces trois fonctions).
- [x] **B2.3** `sync/skill/sqlexec_helpers_test.go` et `cmd/diag_exec/main.go` passent au
  découpeur canonique. Fait : `sqlexec_helpers_test.go` supprimé, ses deux appelants
  (`exclusion_filter_test.go:23`, `skill_rating_loaders_test.go:72`) appellent
  `migration.ExecScriptContext` ; `cmd/diag_exec/main.go:37` boucle sur `migration.SplitSQL`.
- [x] **B2.4** Garde-rail `internal/archlint/no_local_sql_splitter_test.go` :
  - interdit, hors `internal/migration/helpers*.go`, toute définition de découpeur ou
    d'exécuteur de script SQL, et `strings.Split(<x>, ";")` dans un fichier qui exécute du SQL ;
  - dispense datée pour `arbitration_clocks_utc_guard_test.go` ;
  - échec si aucun fichier n'est lu ;
  - mutation vérifiée (une copie réintroduite fait rougir).
  Fait (`:56`) : analyse AST de `internal/` et `cmd/`, tests compris. Définition = fonction
  nommée comme un découpeur ou un exécuteur de script, sauf délégué `return migration.X(...)` ;
  dans un fichier qui appelle `Exec`/`ExecContext`/`Query*` : `strings.Split*(…, ";")` et toute
  comparaison ou `case` sur `';'`. Dispense datée `:37` (plus un contrôle d'existence du
  fichier dispensé) ; `lus == 0` → échec (`:104`) ; ancre : `migration/helpers.go` doit être vu.
  Rouge sur le code d'avant (7 sites, les 3 copies) ; mutation dans `cmd/diag_exec` → rouge,
  retirée. Voir le journal.
- [x] **B2.5 Tests** (rouges d'abord ; aucun nouveau fichier dans `internal/sync/`) :
  - `TestEnsurePlayerSchema_TrailingCommentOnly` (tag `integration`, dans
    `schema_integration_test.go`) : surcharge
    `playerSchemaSQL` + `"\n-- note finale\n"`, restauration par `t.Cleanup`, sans
    `t.Parallel`. Aujourd'hui : « empty query ».
  - `TestExecScript_TrailingCommentOnly`.
  Fait : `schema_integration_test.go:45` et `:114`, rouges avant (« empty query »). En plus :
  `migration/helpers_extra_test.go:45` `TestExecScriptContext_HonoursContext` (contexte annulé).
- [x] **B2.6** Commentaire de `steps_player_schema_authority.go:70-75` réécrit. Godoc de
  `SplitSQL` : limites écrites (chaînes, `/* */`). Fait : `steps_player_schema_authority.go:71`
  (paragraphe « DÉCOUPAGE ») ; `helpers_export.go:72` (« LIMITES ») et renvoi depuis
  `helpers.go` `splitSQL`. Sémantique du découpage inchangée (D-3).

**Gate** : GO-F (paquets ciblés : `./internal/migration/ ./internal/sync/ ./internal/sync/skill/ ./internal/archlint/ ./cmd/diag_exec/`).

### B3 — Index ART de `match_skill_rank` (item 2) — M

**Pièces** (`apps/go-api/internal/` sauf mention) :
- **Trois index** : `idx_msr_match_lookup(match_id, rating_type, written_at)`,
  `idx_msr_rating_type(rating_type)`, `idx_msr_playlist(playlist_group)`. Table des player DB
  seulement.
- Autorités :
  - `sync/schema.go:111-113`, rejouée à chaque `OpenPlayerDB` ;
  - `games/halo_infinite/migrations/steps_player_match_skill_rank.go:109-111` (`lusrChainRework`)
    et `:179-181` (`applyAppendOnlyMatchSkillRank`) ;
  - baseline scellée `steps_player_baseline.go:308-309`, avec le golden
    `testdata/squash/player_block_golden.snapshot:212-213` et
    `TestSquashInvariant_PlayerBaselineEquivalent` ;
  - `cmd/purge_foreign_lusr_chain/purge.go:96-155` capture et rejoue les index.
- **La prémisse « tous les lecteurs passent par `_latest` » est fausse** :
  - `platform/duckdb/queries_career_encounters.go:485` (Q24LUSRHistory),
    `queries_home_citations.go:419` et `queries_squad.go:10`, allowlistés dans
    `no_raw_rating_reads_test.go:67-71` ;
  - `sync/csr_writes.go:299` ;
  - `sync/skill/skill_rating_loaders.go:164,186-189` ;
  - `skill_v2_canonical.go:216,251` ;
  - `skill_v2_metrics.go:131` ;
  - `sync/invariants/*.go`.
  - Les `playlist_group = ?` empruntent vraisemblablement `idx_msr_playlist`. L'incident du
    2026-09-13 (22 lignes par index contre 1 826 par scan, JGtm) prouve qu'un index désynchronisé
    et emprunté rend des lectures FAUSSES (delta LUSR, hystérésis de palier). Le retrait se
    justifie donc par la correction, pas seulement par l'inutilité.
- Recette PSA : `ceea58b39` (fusion `c5dbf70d1`).
  - Migration `drop_psa_secondary_art_indexes_v1` : `internal/migration/steps_player_schema_authority.go:179-190`,
    `order.go:129`, `order_dependency_test.go:50`.
  - Tests : `steps_player_drop_psa_secondary_indexes_test.go` et
    `sync/schema_authority_test.go:253-275`.
  - Ratchet `noSecondaryIndexTables` (`migration/metadata_art_surface_guard_test.go:38-60`). Il ne
    balaie que `internal/migration` et `internal/games`, **pas `sync/schema.go`**.
  - Banc EXPLAIN : `psa_index_repro_planprobe_test.go` (tag `psarepro`).
- À supprimer :
  - la sonde `scheduler/data_health_msr_index.go` et son test, les champs et appels de
    `data_health_check.go:70-79,210,217,232,246-267,346-349`, la jauge
    `data_health_msr_index_desync_keys` ;
  - `cmd/repair_msr_index/` ;
  - `platform/duckdb/indexcheck/` (ses seuls importeurs sont la sonde et l'outil) ;
  - `archlint/no_local_msr_axes_test.go` ;
  - dans `cmd/purge_foreign_lusr_chain` : `checkIndexCoherence`, `ForeignIndexed`, le message qui
    nomme `repair_msr_index` (`main.go:131-145`), et les tests `purge_test.go:209` et
    `:137-140` (l'assertion « au moins 3 index » rougira).
- Aucune mention dans `docs/`, le Makefile, la CI ou la baseline.
- Le harnais `psarepro` (1 488 lignes) reste : `challenge_snapshots` et `battlepass_snapshots`
  gardent des index.

- [x] **B3.1 Mesure versionnée AVANT tout retrait** : `psa_index_repro_msr_planprobe_test.go`
  (tag `psarepro`), DB FICHIER de 12 000 lignes réalistes. EXPLAIN et temps avec et sans index
  pour les sept formes de lecture listées ci-dessus. Critère D-4. Au-delà : **STOP**, index
  conservés, raison écrite.
  → Fait : `migration/psa_index_repro_msr_planprobe_test.go` (`TestMSRIndexRemovalPlanProbe`,
  critère D-4 asserté) + `psa_index_repro_msr_fixture_test.go` (DDL, remplissage, formes).
  **Premier verdict (critère d'origine) : NON TENU → STOP.** F1 (`Q24LUSRHistory`),
  F2 (citations, IN(200)) et F3 (escouade, IN(1000)) dépassent 10 ms SANS index (13,5 / 13,5 /
  16,7 ms, médiane client, passage final) — mais AUTANT avec index (14,9 / 18,1 / 18,4 ms) :
  plan séquentiel des deux côtés, 30 fois sur 30. Le dépassement ne vient pas de l'absence
  d'index. Seule forme qui emprunte un index : F4 (`rating_type = 'CSR'`), ~10× plus LENTE
  avec (20,3 ms contre 1,9 ms). Tableau complet et raison : journal §7 (B3).
  **Après l'amendement de D-4 (`99b241970`) : critère relatif TENU**, assertion réécrite
  (`msrProbeMargin`, 2 ms ; médianes absolues toujours journalisées) : `EXIT_MESURE=0`, plus
  grand écart « sans − avec » +0,22 ms (F3). Journal §7.
- [x] **B3.2** Migration `drop_msr_secondary_art_indexes_v1` : cible player, 3
  `DROP INDEX IF EXISTS`. `order.go` après `player_msr_view_latest_by_type_v1` ;
  `stepDependencies` vers `lusr_chain_rework_v1` (précédent `drop_career_xuid_art_index_v1`,
  `order.go:126`).
  → Step title-owned, dernier de la chaîne MSR : `games/halo_infinite/migrations/steps_player_match_skill_rank.go:73`,
  `dropMSRSecondaryARTIndexes` `:371` (découpeur canonique `migration.ExecScriptContext`).
  Il est title-owned et non global : une étape globale placée à cet endroit de `canonicalOrder`
  casserait `TestSortByCanonicalIsNoOpOnCurrentRegistry`. `order.go:100` ;
  `order_dependency_test.go:55`.
- [x] **B3.3** Retrait des `CREATE INDEX idx_msr_*` des autorités non scellées : `schema.go:111-113`,
  `steps_player_match_skill_rank.go:109-111` et `:179-181`. Baseline scellée INTACTE (précédent
  `idx_career_xuid`).
  → `sync/schema.go:113` (3 lignes → 1 commentaire ; 525 lignes avant, 525 après) ;
  `lusrChainRework` et `applyAppendOnlyMatchSkillRank` sans index, commentaires à jour.
  `steps_player_baseline.go` et son golden non touchés.
- [x] **B3.4** Convergence (D-4) : les `DROP INDEX IF EXISTS` des trois `idx_msr_*` ET des trois
  index PSA retirés le 2026-09-20 (noms relus dans la migration PSA) entrent dans l'autorité
  rejouée par `EnsurePlayerSchema`.
  → SOURCE UNIQUE `migration/steps_player_schema_authority.go:145-160`
  (`PlayerRetiredPSAIndexesDropSQL`, `PlayerRetiredMSRIndexesDropSQL`,
  `PlayerRetiredARTIndexesDropSQL`). Consommée par le step PSA (`:216`, son littéral est
  remplacé), par le step MSR et par `sync.playerSchemaSQL` (`schema.go:36`, en dernier).
- [x] **B3.5** Suppressions de la liste ci-dessus, avec imports, types et jauges.
  `no_raw_rating_reads_test.go` n'est PAS modifié (fichier de `feat/perf-perimetre`) : s'il
  mentionne un index retiré, découverte.
  → Supprimés :
  - `scheduler/data_health_msr_index.go` et son test ;
  - `cmd/repair_msr_index/` (3 fichiers) ;
  - `platform/duckdb/indexcheck/` (3 fichiers) ;
  - `archlint/no_local_msr_axes_test.go`.

  Débranchés dans `data_health_check.go` : les cinq champs `MSRIndex*`, la somme
  `WarningsTotal`, la jauge `data_health_msr_index_desync_keys`, les champs de journal et
  l'appel. Nouveau commentaire `:67-70`.

  Dans `purge_foreign_lusr_chain` :
  - `ForeignIndexed`, `indexMismatch`, `checkIndexCoherence` et le message
    `repair_msr_index` sont retirés ;
  - `TestCheckIndexCoherence_BlocksCommitOnDesync` et l'assertion « ≥ 3 index » sont
    supprimés ;
  - `TestCensus_ForcesScanOnHealthyDB` (`purge_test.go:161`) garde le compte par scan.

  `no_raw_rating_reads_test.go` : aucune mention d'index, non modifié.
- [x] **B3.6** Ratchet : `match_skill_rank` entre dans `noSecondaryIndexTables`, avec une
  dispense DATÉE pour `steps_player_baseline.go`. Balayage étendu à `internal/sync/schema.go`.
  Mutation vérifiée.
  → `migration/metadata_art_surface_guard_test.go` :
  - `:66` ajout de la table ;
  - `:72` `noSecondaryIndexExemptions`, dispense datée du 2026-09-27 avec son motif ; une
    dispense morte fait échouer le test ;
  - `:151` balayage étendu à `sync/schema.go` ; `Fatal` si le fichier est introuvable.

  Rouge sur le code d'avant (9 sites). Mutation : un mutant dans `sync/schema.go` et un dans
  la chaîne du titre donnent `EXIT_MUTANT=1`, les deux sites sont désignés ; mutants retirés.
- [x] **B3.7 Tests** (rouges d'abord) :
  - `TestPlayerSchemaAuthority_NoMatchSkillRankSecondaryIndex` (`sync/schema_authority_test.go`) ;
  - `steps_player_drop_msr_secondary_indexes_test.go` : retrait, idempotence, lignes et vues
    préservées ;
  - convergence B3.4 : DB migrée, index MSR et PSA recréés à la main, puis `EnsurePlayerSchema`
    → plus aucun.
  - Aucun nouveau fichier dans `internal/sync/` (ratchet de gel) : les tests de `sync` vont dans
    les fichiers existants.
  → Tests :
  - `sync/schema_authority_test.go:298` `…_NoMatchSkillRankSecondaryIndex` ;
  - `:317` `…_EnsureDropsRetiredARTIndexes` (MSR ET PSA recréés `IF NOT EXISTS`, avec
    précondition) ;
  - `games/halo_infinite/migrations/steps_player_drop_msr_secondary_indexes_test.go` : trois
    tests, `:80` chaîne neuve sans index, `:90` retrait et idempotence, `:111` lignes, deux vues,
    priorité CSR et INSERT sans id.

  Tous rouges sur le code d'avant, verts après (journal). Aucun nouveau fichier sous
  `internal/sync/`.
- [x] **B3.8** En-tête du harnais `psarepro` réécrit : véhicule de reproduction pour les index
  player restants.
  → `migration/psa_index_repro_test.go:13-24` (et la mention d'`indexcheck` `:158-159`).
- [x] **B3.9** Vérification sur COPIE d'une vraie player DB, jamais l'original (copie faite
  serveur principal arrêté, ce que le superviseur confirme avant). Migration puis `EnsurePlayerSchema` appliqués à la copie, par la CLI existante
  si elle migre une base désignée, sinon par un test d'intégration paramétré par une variable
  d'environnement (pas d'outil jetable). Attendu : `duckdb_indexes()` sans `idx_msr_*` ni index
  PSA, mêmes nombres de lignes, vues intactes.
  → Fait le 2026-09-27, sur décision du superviseur : `halo_infinite/JGtm` avec son `.wal`
  copié dans le même geste, et `halo_5/JGtm`, qui n'a pas de WAL.
  - Contrôles faits avant chaque copie : aucun serveur, `:8000` muet.
  - Après chaque copie : même taille, même date et même SHA-256 que l'original.
  - Aucun original n'a été ouvert.

  Aucune CLI ne migre une base désignée ; le véhicule est donc un test,
  `sync/schema_msr_views_test.go:220` `TestRetiredARTIndexes_RealPlayerDBCopy` (variable
  `LEVELUP_B3_PLAYER_DB_COPY`, sauté sans elle).
  - Il refuse tout chemin sous `data/titles/`.
  - Il accepte un `.wal` copié avec la base, et journalise l'issue de son rejeu.
  - Il applique `RunForDB(player)` puis `EnsurePlayerSchema` (`:170`).
  - Il compare index, lignes par table (hors `schema_migrations`), vues et leurs lignes.
  - Validé sur une base synthétique (`:240`) : rouge sur `82cd8871b`, vert après.

  - **`halo_infinite/JGtm`** (log `B3-9-copie-hi.log`, `EXIT_COPIE_HI=0`) :
    - WAL (745 o, 2026-09-23 19:56:45) : **rejeu RÉUSSI**, aucune erreur ART.
    - Avant : `idx_msr_match_lookup`, `idx_msr_rating_type` et `idx_msr_playlist` présents
      (les index PSA étaient déjà partis) ; 34 tables, 8 vues ; `match_skill_rank` 33 702
      lignes, `personal_score_awards` 3 971, `match_skill_rank_latest` 1 152,
      `match_skill_rank_latest_by_type` 2 287.
    - Migration : une seule étape appliquée, `drop_msr_secondary_art_indexes_v1`.
    - Après : aucun index retiré ; 34 tables et 8 vues, mêmes lignes partout.
  - **`halo_5/JGtm`** (log `B3-9-copie-h5.log`, `EXIT_COPIE_H5=1`) :
    - Avant : les six index retirés présents ; 32 tables, 8 vues ; `match_skill_rank` 7 657,
      `personal_score_awards` 234, `match_skill_rank_latest` 1 780,
      `match_skill_rank_latest_by_type` 3 924.
    - Migration : 12 étapes appliquées ; la base avait du retard sur la chaîne (DB-20).
    - Après : **aucun index retiré** ; `match_skill_rank`, `personal_score_awards` et les
      8 vues inchangés.
    - Deux écarts, qui ne viennent PAS de B3 :
      - `player_match_enrichment` passe de 56 365 à 57 763 lignes (INSERT append-only de
        `player_dominance_flag_reset_none_v1`) ;
      - `arc_titles`, vide, est supprimée (`drop_arc_titles`).
    - Preuve de l'attribution : une seconde copie (`h5-avant`, identique à l'original) passée
      au code d'avant B3 (`82cd8871b`, worktree jetable retiré ensuite ; log
      `B3-9-copie-h5-code-avant.log`) donne les MÊMES deux écarts. Elle applique 11 étapes,
      et les trois `idx_msr_*` y restent. La seule différence imputable à B3 est donc le
      retrait des index.

**STOP D-4 levé.** Le STOP posé à B3.1 sur le critère d'origine a été levé par l'amendement de
D-4 (§2, commit `99b241970`). Le critère relatif est tenu (B3.1), donc B3.2 à B3.9 reprennent.

**Gate** : GO-F, plus `go test -tags=psarepro -count=1 ./internal/migration/ -run MSR -v`.

### B4 — Garde-rail d'exclusion de la campagne (item 4) — M-L

**Pièces** (`apps/go-api/internal/`) :
- `platform/duckdb/campaign_exclusion_guard_test.go:69-156` :
  - ne balaie que les const/var de premier niveau nommées `Q…`, dans le seul répertoire
    courant ;
  - critère : `match_participants|mv_player_matches` + la sous-chaîne exacte `xuid = ?` ;
  - exemption : `campaignExclusionToken` ou une allowlist de 6 entrées (`:71-78`).
- Angles morts :
  - littéraux locaux ;
  - constantes non-`Q` (`q31CrossGameCooccurrenceTpl`, `qElapsedSecondsByMatchTpl`,
    `playerMatchesSharedBaseSelect`, `clauseCoequipier`, `h5MatchSummarySelect`) ;
  - sous-paquets `prestige/` et `halo5/` ;
  - lecteurs hors `platform/duckdb`.
- Mécanismes d'exclusion :
  - `platform/duckdb/campaign_exclusion.go:28-70` : jeton `/*__EXCLUDE_CAMPAIGN__*/`,
    `resolveCampaignExclusion`, `resolveCampaignExclusionByMatchID`, `excludeCampaignClause`,
    `excludeCampaignByMatchID`, `excludeAllCampaignByMatchID` ;
  - source : `analysis/campaign_exclusion.go:70-175` ;
  - hors paquet : `analysis.SQLExcludeCampaign*` et `profile.(*Service).campaignExcl`.
- Les deux cas cités par le backlog sont réglés : `GetLocalStats` est exclu
  (`compare_repo.go:123`), `GetCrossMatchSample` a été supprimé (`aa1a8dc2c`).

- [x] **B4.1 Nouveau balayage AST**, selon D-5 : `internal/archlint/campaign_exclusion_guard_test.go`
  (balayage multi-racines). Il REMPLACE le garde de `platform/duckdb`, qui est supprimé avec
  reprise à l'identique de ses 6 dispenses : pas deux gardes pour une même règle.
  - toutes les `FuncDecl` et `GenDecl`, quel que soit le nom ;
  - agrégat des `BasicLit` STRING d'une déclaration, plus résolution des `Ident` de constantes
    chaîne du paquet (concaténations, `fmt.Sprintf`, `WriteString`, variables séparées) ;
  - lecteur = `\b(match_participants|mv_player_matches)\b` ET `\bxuid\s*(=\s*\?|IN\s*\()`, hors
    forme `(… xuid = ?) AS …` ;
  - exclusion reconnue = appel d'un résolveur listé, `campaignExclusionToken` ou `campaignExcl` ;
  - dispenses de clé `chemin.go:Recv.Func`, avec justification datée et catégorie (mono-match /
    ensemble fourni par l'appelant / exclu au call site / sémantique) ;
  - échec sur une dispense qui ne correspond plus à rien ; `scanned > 0` par racine ; mutation
    vérifiée.
  - **Fait (2026-09-27)** : `TestCampaignExclusionGuard` (`archlint/campaign_exclusion_guard_test.go`,
    dispenses dans `campaign_exclusion_guard_dispenses_test.go`). Une constante nommée qui est
    elle-même un lecteur garde son verdict (son texte n'est pas recopié chez ses utilisateurs :
    un signalement, pas deux). Garde de `platform/duckdb` (`TestCampaignExclusionStructuralCoverage`)
    supprimé ; ses 6 dispenses reprises mot pour mot. Mutations : voir journal B4.
- [x] **B4.2 Corrections.** Chacune porte un cas dans `campaign_exclusion_behavior_test.go` (tag
  `integration`), rouge avant :
  1. `platform/duckdb/compare_repo.go:GetEncounterStats` (matchs communs A/B) ;
  2. `platform/duckdb/campaign_repo.go:(*CampaignSampleProvider).LoadAxisSamples` ;
  3. `platform/duckdb/engagement_score_repo_queries.go:ListRecentPvPMatchIDs` ;
  4. `platform/duckdb/cross_game_repo.go:q31CrossGameCooccurrenceTpl`, par
     `excludeAllCampaignByMatchID` ;
  5. `platform/duckdb/prestige/prestige_baseline_provider.go:RecentMatches` et `CumulativeSince` ;
  6. `api/wire/post_sync_progression_queries.go:loadProgressionSharedMatches`,
     `loadPlayerStats` (2 requêtes), `loadComebackContext` ;
  7. `api/wire/post_sync_deltas_snapshot.go:SnapshotPlayerState` (3 requêtes) ;
  8. `platform/duckdb/prestige_squad_match_provider.go:124,220` (`xuid IN`, roster).
  - **Fait** : les 8, chacun rouge sur le code d'avant puis vert (journal B4). Cas dans
    `platform/duckdb/`, `platform/duckdb/prestige/` et `api/wire/campaign_exclusion_behavior_test.go`,
    chacun joué pour `halo_5` (Campagne masquée) ET `halo_infinite` (résolveur neutre). Titre :
    celui de la base partagée lue (`pdbTitleSlug` / `pdb.TitleSlug`) ou le `titleSlug` déjà passé
    par l'appelant Prestige (paramètres `_` renommés). `loadPlayerStats` compte TROIS requêtes
    (la 3e, métriques de combat), toutes corrigées.
- [x] **B4.3 À statuer par lecture de l'appelant ou de la capability** :
  - `fanout_repo.go:CountCommonMatchesForXUID` : si l'ensemble fourni est exclu à la source,
    dispense avec renvoi `fichier:ligne` ; sinon, correction.
  - `leaderboard_world_repo.go:GetStatLeaderboard:612` (agrégat sans filtre xuid, hors critère) :
    si le lecteur ne sert qu'un titre sans variante de campagne, justification écrite ; sinon,
    correction.
  - **Statué** : `CountCommonMatchesForXUID` → dispense « ensemble fourni par l'appelant » :
    l'ensemble est `insertedMatchIDs` (seul appelant `service/fanout_service.go:73`), matchs
    nouvellement insérés, et la Campagne n'est plus collectée (`games/halo_5/capture.go:263`,
    `isExcludedH5GameMode` `:128-129`). `GetStatLeaderboard` : DÉJÀ exclu,
    `excludeCampaignByMatchID(titleSlug, "mp.match_id")` à `leaderboard_world_repo.go:620` —
    rien à faire.
- [x] **B4.4 Dispenses justifiées** :
  - `engagement_score_repo_queries.go:LoadMatchEngagementContext` (mono-match) ;
  - `csr_coverage_repo.go:countRankedMatchesInRegistry` (prédicat classé : la campagne n'est
    jamais classée) ;
  - `media_repo_filters.go:LoadMatchCandidatesForMedia` (appariement clip → match,
    sémantique) ;
  - `api/wire/registry_monitoring_freshness.go:lastMatchByXUID` (fraîcheur admin : la campagne
    est une activité réelle) ;
  - les 6 dispenses existantes, reprises à l'identique ;
  - les faux positifs éventuels (`media_repo_filters.go:loadMatchLobbies`,
    `analysis/match_filter.go:195`, `analysis/identity_annuaire.go:97`), si le motif les
    attrape.
  - **Fait** : les 4 dispenses, les 6 reprises, et 2 faux positifs attrapés
    (`BuildNeighborsWhereClause` exclu au call site, `AnnuaireNomsSQL` sémantique).
    `loadMatchLobbies` n'est pas attrapé (aucun filtre xuid) : pas de dispense. Pièces dans le
    champ `raison` de chaque dispense.
- [x] **B4.5** Tout autre entrant révélé par le balayage est statué selon les mêmes catégories.
  S'il n'entre dans aucune : **STOP**. La liste des lecteurs de `sync/`, `ops/` et `cmd/`
  (enquête : `sync/engagement.go:352`, `performance_helpers.go:228`, `session_recalc.go:30`,
  `skill/skill_rating_loaders.go:73`, `skill/skill_v2_shadow.go:434`, `assists_model.go:68`,
  `citations.go:379`, `friends_recompute.go:200`, `snapshot/snapshot_readiness_eval.go:208`,
  `csr_shared_backfill.go:194`, `ops/milestone_dates.go:294`) est écrite au journal pour le
  backlog. Rien n'y est corrigé.
  - **Fait** : 8 autres entrants, tous dans une catégorie (aucun STOP) : 3 « exclu au call
    site », 2 « sémantique », 3 « ensemble fourni par l'appelant » (détail au journal). Liste
    `sync/`, `ops/`, `cmd/` relevée par le garde lui-même (sonde temporaire, retirée) : au
    journal B4.
- [x] **B4.6** Aucune comparaison de slug ajoutée (ratchet `no_slug_comparison_test.go`). Les
  fichiers de `feat/perf-perimetre` ne sont pas touchés (§3.1.6).
  - **Fait** : aucune comparaison ajoutée (seul littéral de slug : les tests, en données) ;
    ratchet vert dans la gate archlint. Aucun fichier du §3.1.6 dans le diff.

**Gate** : GO-S sur `./internal/archlint/ ./internal/platform/duckdb/... ./internal/api/wire/ ./internal/progression/... ./internal/service/... ./internal/analysis/...`,
plus `go test -count=1 ./...` complet.

### B5 — Mode démo hermétique côté fichiers (item 8) — M

**Pièces** (`apps/go-api/`) :
- `internal/config/config.go:217-241` : `SessionDir`, `AuthDir`, `DBProfilesPath` et
  `AppSettingsPath` sont toujours dérivés de `repoRoot`.
- Réseau et bases :
  - `netguard.SetOffline(cfg.DemoMode)` (`cmd/server/main.go:331`) ;
  - les 4 bases passent par `demoWarehouseDBPath` (`cmd/server/demo_paths.go:27`) ;
  - les lecteurs par `config/player_resolver.go:58-117`.
- Ratchets existants : `demo_paths_test.go`, `netguard_coverage_test.go`,
  `archlint/no_data_path_join_test.go` (ne balaie que `internal/`, et `main.go:1373` y
  échappe).
- En prod, le conteneur `levelup-demo` est isolé par Docker (`docker-compose.yml:64-119`). La
  fuite réelle concerne donc le poste de dev et la CI :
  `scripts/demo-visual-harness.sh:165-170` passe le vrai checkout en `LEVELUP_REPO_ROOT`.
- **Écritures et suppressions hors racine en démo** :
  - logs (`logging/config.go:84`, crash log `main.go:227`) ;
  - sessions : `MkdirAll` et purge au boot puis toutes les 6 h (`platform/session/store.go:121`,
    `api/server.go:582-676`) ;
  - `data/global/player_friends.json` (`main.go:2296`) ;
  - **`data/global/monitoring.duckdb` ouvert en ÉCRITURE** (`ops/monitoring_store.go:72-75`,
    `main.go:1161`) ;
  - `data/global/admin_state/*` (`main.go:783-790`, `registry_monitoring_diskwatch.go:45`,
    boucle `main.go:1190`) ;
  - **WAL et `RecoverPending`, qui rejoue un WAL réel dans les VRAIES player DB**
    (`persist/queue.go:119`, `main.go:811-870`, `:853`) ;
  - janitor (`main.go:876-916`) ;
  - **purge des rejeux sur le vrai shared, avec `os.Remove`** (`scheduler/replay_purge_cron.go:120-190`,
    `main.go:1430`) ;
  - santé données (`main.go:1077`) ;
  - écrivain de la file de build (`api/wire/registry_build_queue.go:311-334`) ;
  - cache de l'aide (`api/handlers/help.go:158-167`) et `jobs.json` (`api/server.go:685`).
- **Lectures hors racine** :
  - tokens de `data/auth` ; `users.json`, `invites.json`, `groups.json` (`main.go:695,703`,
    `server.go:695`) ;
  - `migrateDefaultGroupAtBoot` lit le vrai `db_profiles` (`config/config_players.go:167`) ;
  - overlays de titre (`handlers/settings.go:154`) ;
  - rejeux et faits de film (`registry_pages.go:159,189`, `registry_film_facts.go:67`).
- **Lectures légitimes** : `config/titles/**`, `data/titles/*/reference/**`, `static/`,
  `WebDistDir`, README et changelog.
- Traduction des chemins démo en **3 copies** (règle n°6) : `cmd/server/demo_paths.go:27`,
  `internal/config/player_resolver.go:82`, `internal/ops/seed_demo_multititle.go:32`.

- [x] **B5.0** (tableau de 40 lignes au journal B, entrée « B5.0 » du 2026-09-27) Ré-inventaire, au début du lot, des tâches de fond et des chemins de `main.go` (le
  code bouge). Chacun est classé : coupé en démo / redirigé sous la racine démo / légitime.
  Tableau au journal ; il fait foi pour B5.2-B5.5.
- [x] **B5.1** (`internal/domain/title/demo_layout.go` `title.DemoLayout` ; garde-rail `internal/archlint/no_demo_layout_translation_test.go`, rouge 29 sites puis mutation ; copies supprimées : `cmd/server/demo_paths.go` demoWarehouseDBPath, `config/player_resolver.go` demoTitleDir + 3 traducteurs, `ops/seed_demo_multititle.go` demoTitleSubdir ; seed de la démo migré sur la disposition) Un seul helper de disposition démo, qui remplace les 3 copies, plus un garde-rail
  (test grep qui interdit la traduction hors du helper).
- [x] **B5.2** (`config/config.go` Load + `config/config_demo.go` statePaths, RuntimePaths, WatcherTokensDir, TitleSettingsPath, DemoLogsDir ; logs par `cmd/server/demo_paths.go` bootLogsConfig ; variable explicite gagne ; `git check-ignore` : `data/demo/runtime/x` → `.gitignore:132`, `tests/fixtures/demo-root/runtime/x` → `.gitignore:198`) `config.Load` en démo, sans variable explicite, selon D-7 :
  - `SessionDir` → `<démo>/runtime/sessions` ;
  - `AuthDir` → `<démo>/auth` ;
  - logs et crash log → `<démo>/runtime/logs` ;
  - `AppSettingsPath`, `DBProfilesPath` et overlays de titre → fichiers de la fixture.
  - `git check-ignore` vérifié pour `runtime/` sous `data/demo` et `tests/fixtures/demo-root`.
- [x] **B5.3** (`cmd/server/background_tasks.go` : table `demoPolicies`, `bootTasks.launch`/`cutInDemo`/`logCut` ; `PersistBatchAsync` forcé à false en démo dans `config.Load` ; `ops.MonitoringStoreInMemory` ; 16 tâches déclarées dont 12 coupées et 4 gardées, 5 étapes de boot coupées ; un seul log Info `demo_mode: tâches de fond coupées`) Tâches de fond coupées en démo, avec garde explicite et un seul log Info au boot
  qui liste ce qui est coupé :
  - janitor ;
  - file persist asynchrone (`PersistBatchAsync` forcé à false, aucun `RecoverPending`) ;
  - magasin de monitoring en mémoire ;
  - santé données ; purge des rejeux ; surveillance disque ; migration des amis ;
  - et toute autre tâche classée « coupée » en B5.0.
- [x] **B5.4** (`wire/registry_build_queue.go` sharedWriterForTitle → `config.SharedDBPath`) L'écrivain de la file de build (`sharedWriterForTitle`) passe par la même
  disposition démo que les 4 bases.
- [x] **B5.5** (`cfg.RuntimePaths()` : `api/server.go` jobs + amis, `server_apiv1.go` cache de l'aide, `cmd/server` amis + état admin, `registry_pages.go` rejeux + rasters, `registry_monitoring_resources.go` faits de film ; tokens par `cfg.WatcherTokensDir()` ; cache d'assets inchangé, lecture seule) Caches écrits en démo (aide, `jobs.json`) → `<démo>/runtime/`. Rejeux et faits de
  film lus sous la racine démo (aucun en démo, comme en prod). Cache d'assets : lecture seule
  autorisée.
- [x] **B5.6** (atteignable : `middleware/require_admin.go:18` transparent en démo → refus 403 `demo_mode_forbidden` dans `handlers/settings_backup.go` ; `settings_backup_demo_test.go`, rouge puis vert) Sauvegarde : établir si `POST /settings/backup/run` est atteignable en démo
  (comportement de `RequireAdmin` en démo). Si oui : refus en démo, avec un test.
- [x] **B5.7 Tests** (rouges d'abord ; sorties au journal B5) :
  - (1) `internal/config` : `Load()` avec un `LEVELUP_REPO_ROOT` leurre et le mode démo → aucun
    chemin d'exécution sous le leurre.
  - (2) `cmd/server` : le lancement des tâches de fond est extrait de `main()` vers une fonction
    testable, qui déclare le statut démo de chaque tâche. Sur un leurre à fichiers sentinelles,
    un manifeste avant/après (chemin, taille, date) doit montrer un écart nul, sans
    `monitoring.duckdb`.
  - (3) `demo_paths_test.go` étendu à tous les chemins de `cfg`.
  - (4) `archlint/no_data_path_join_test.go` étendu à `cmd/server` (`main.go:1373` corrigé, ou
    dispensé avec une justification datée).
- [x] **B5.8 Preuve de bout en bout** (NON JOUÉE par l'exécutant le 2026-09-27 : `http://127.0.0.1:8000/health` répondait 200 — serveur du checkout principal actif, que l'exécutant n'arrête pas ; à jouer par le superviseur, cf. journal B5), avec le serveur principal ARRÊTÉ (le superviseur le
  coordonne) : marqueur daté, `make demo-visual`, puis `find data logs -newer <marqueur>` doit
  être vide hors de la racine démo. Sortie au journal.
  - **JOUÉE par le superviseur le 2026-09-27 à 21 h 11**, sur le code de `94363c225` (B5 + B-C1
    à B-C10). Variante plus exigeante que le harnais : la racine du DÉPÔT est le vrai checkout
    principal, et non le worktree.
    - Mise en place : binaire `cmd/server` du worktree, `LEVELUP_REPO_ROOT` = checkout
      principal, `LEVELUP_DEMO_MODE=true`, `LEVELUP_DEMO_FIXTURES_DIR` = COPIE de
      `data/demo` sous `%TEMP%`, port 8010. Aucun serveur ne tournait sur :8000 et aucune
      écriture n'avait eu lieu dans `data/` ni `logs/` du principal pendant les 15 minutes
      précédentes.
    - Sollicitations :
      - `bootstrap` 200 (joueur démo) ;
      - `POST /setup/players`, `POST /settings/backup/run`,
        `DELETE /profiles/DemoPlayer/titles/halo_infinite/data`, `PATCH /watcher/subscriptions` :
        les quatre en 403 `demo_mode_forbidden` ;
      - `GET /admin/identities` 200, sans aucun gamertag réel ;
      - notes de version FR et EN 200, monitoring 200.
    - Puis 60 s de marche, et arrêt.
    - Résultat : **0 fichier créé ou modifié et 0 dossier créé** sous `data/` et `logs/` du
      checkout principal après le marqueur. 8 fichiers d'exécution sous
      `<copie démo>/runtime/` (data, logs, sessions).
    - Au passage : une ERROR de requête sur `titles/halo_5/warehouse/metadata.duckdb` de la
      fixture au boot. Elle est préexistante (fixture), sans lien avec l'hermétisme. Découverte.

**Gate** : GO-F (paquets ciblés : `./cmd/server/ ./internal/config/ ./internal/archlint/ ./internal/platform/netguard/ ./internal/scheduler/ ./internal/api/... ./internal/ops/ ./internal/persist/...`),
plus B5.8.

### B-R — Revue adversariale des lots B

- [ ] Skill `adversarial-review` sur les commits de B1 à B5 (plage `<fusion 1>..<fin de B5>` de
  `feat/backlog-2026-09-26`), contexte frais (opus-high), seul agent actif.
  - **Précision (superviseur, 2026-09-27)** : faute de fusion 1, la plage relue est le diff Go
    `d61443ef5..3de419efe -- apps/go-api docs/CONFIGURATION.md docs/FR/CONFIGURATION.md
    .ai/baselines`.
  - Le skill exige deux relecteurs pour un diff qui touche `sync`, `migration` et `persist`. Ils
    passent l'un APRÈS l'autre (un seul agent à la fois), aveugles l'un à l'autre :
    - relecteur 1 : lentilles L1 (DuckDB / anti-ART) et L3 (anti-patterns) ;
    - relecteur 2 : lentilles L6 (tests), L2 (multi-titre) et L4 (données).
  - **Relecteur 1 : 6 constats recevables, tous sur B5 (démo) ; 23 conditions vérifiées qui
    tiennent (B1-B4 : aucun constat).** Triage du superviseur :
    - **C1, P1** : `server_apiv1.go:416`, `NewProfileService(cfg.DBProfilesPath, cfg.RepoRoot)`.
      En démo sur un vrai checkout, `POST /setup/players` (profil azure_manual) puis
      `DELETE /profiles/{p}/titles/{t}/data` EFFACE `<dépôt>/data/titles/<t>/players/<p>`.
      Préexistant, mais dans l'objectif du lot → **corrigé**.
    - **C2, P2 → corrigé** : en démo, `db_profiles.json` et `app_settings.json` visent désormais
      la fixture, montée en écriture dans le conteneur de production. `POST /setup/players` et
      `PATCH /watcher/subscriptions` persistent donc des écritures anonymes sur l'hôte.
    - **C3, P2 → corrigé** : `server_player_directory.go:65`, `NewPathFS(cfg.RepoRoot)` : la
      section Identités de la démo liste les vrais dossiers de joueurs du poste.
    - **C4, P2 → corrigé après vérification** : `server_apiv1.go:1383`, un cache d'assets écrit
      sous le dépôt en démo (`PersistBinary`). Vérifier aussi que le téléchargement GameCMS est
      bien couvert par `netguard` ; si le réseau part vraiment en démo, c'est un trou de
      l'hermétisme RÉSEAU, corrigé dans le même lot.
    - **C5, P2 → corrigé** : `config.go` passe de 629 à 640 lignes, alors que la règle n°5 dit
      « ne pas accroître la dette ». Les ajouts vont dans `config_demo.go`.
    - **C6, P2 → corrigé** : le retrait convergent des index (B3.4) est silencieux, alors que le
      contrat de `sync/schema.go:21-28` exige que « toute action réelle est journalisée ».
    - **Code mort (non retenu par le relecteur, retenu ici, règle n°7)** :
      `title.DemoLayout.Root()` sans appelant → supprimé.
  - **Décision de correction** : C1 et C2 → en mode démo, les trois routes de mutation
    (`POST /setup/players`, `DELETE /profiles/.../data`, `PATCH /watcher/subscriptions`)
    répondent 403 `demo_mode_forbidden`, sur le modèle de B5.6 (sauvegarde). L'ouverture
    générale des actions admin en démo (DB-28) reste une entrée de backlog : c'est une décision
    de contrat d'API, hors de ce lot.
- [ ] Constats P0 et P1 : lot de corrections par un exécutant opus-high, sur la même branche,
  gates du lot concerné rejoués. Constats P2 : découvertes, versés au backlog.
  - Le lot de corrections (B-C) attend le relecteur 2, pour traiter les constats des deux
    relectures d'un seul coup. La ronde 2 relira ensuite les seules corrections (§8 du skill).
  - **Relecteur 2 : 2 constats recevables ; 18 conditions vérifiées qui tiennent.** Triage :
    - **R2-1, P1 (régression de B5)** : `api/wire/registry_pages.go:160`. En démo, le service de
      rejeu est enraciné sur `RuntimePaths()` (`<démo>/runtime`). Or il y lit aussi des DONNÉES
      VERSIONNÉES :
      - `MapBackgroundDir`, soit 218 fonds de carte ;
      - `TitleMappingsDir` (`replay_map_objectives.go:107`, `replay_vehicle_weapons.go:44`) ;
      - `replaylabels.Load` (`replay_weapon_labels.go:50`, `replay_vehicle_labels.go:126`) ;
      - `zonesPourIdentites` (`replay_map_callouts.go:52`) ;
      - `ReglesDepartsAleatoires` (`replay_weapon_tiers.go:46`).
      Résultat : l'onglet Tactique de la démo répond 404 sur les fonds de carte, et les
      catalogues du rejeu sont introuvables. C'est une régression par rapport à la base.
    - **R2-2, P2 → corrigé** : `handlers/prestige_squads.go:162,192`. Le web appelle `/squads`
      sans `title_slug` (`apps/web/src/lib/prestige.ts:478-481`), donc `SquadUsualContexts`
      reçoit `""` et l'exclusion de la campagne ne s'applique pas sur le vrai chemin HTTP. Le
      test passe `"halo_5"` en dur. Correction : le titre vient du `PlayerDB` résolu
      (`pdb.TitleSlug`), et le test passe par le chemin réel.
    - Non traité (découverte) : les lecteurs agrégés de `medals_earned` (`Q36aMedalTotals`,
      `queries_citations.go:38`) ne sont pas couverts par le critère D-5. À ajouter à l'entrée
      backlog des lecteurs non exclus, avec DB-26.

### B-C — Corrections de la revue adversariale (ronde 1)

Périmètre FERMÉ : un item par constat retenu, chacun avec un test de non-régression rouge avant.

- [x] **B-C1 (C1 + C2)** (`handlers/demo_guard.go` `refuseInDemo`, 4 appelants : `setup.go` handleCreatePlayer, `title_sync.go` Purge — `NewTitleSyncHandler(profiles, demoMode)`, câblé `server_apiv1.go` `cfg.DemoMode` —, `watcher_handler.go` handlePatchSubscriptions, `settings_backup.go` migré ; `demo_mutations_refused_test.go` rouge puis vert ; garde-rail `archlint/no_demo_forbidden_literal_test.go`, 4e copie, règle n°6, prouvé par mutation ; OpenAPI : réponse `default` ApiError, aucune déclaration, comme B5.6) : en mode démo, `POST /setup/players`, `DELETE /profiles/{p}/titles/{t}/data`
  et `PATCH /watcher/subscriptions` répondent 403 `demo_mode_forbidden`, sur le modèle de
  `settings_backup.go` (B5.6). Tests handler : 403 en démo, comportement inchangé hors démo.
  Contrat OpenAPI tenu comme pour B5.6.
- [x] **B-C2 (C3)** (`api/server_player_directory.go` buildPlayerDirectory : aucun témoin disque en démo — branche « collecte coupée » ; pas de `NewPathFS` sur la racine démo, dont les dossiers ne portent pas les clés de profil ; `server_player_directory_demo_test.go`, rouge puis vert, témoin hors démo) : en démo, la section Identités ne balaie pas les dossiers du dépôt :
  `NewPathFS` sur la racine démo, ou collecte des dossiers orphelins coupée en démo. Test sur un
  dépôt leurre avec un dossier de joueur réel, qui ne doit pas apparaître.
- [x] **B-C3 (C4)** (verdict réseau : PAS de trou — `fetchGameCMSImage` → `doGet` → `netguard.Check("gamecms_assets.get")` avant `httpClient.Do` (`assets/fetcher_gamecms.go:334`), seule source d'un payload binaire ; donc aucun `PersistBinary` en démo, aucun code modifié. Test `assets/demo_offline_no_persist_test.go` : 0 requête et cache vide en démo, persistance hors démo ; vert sur le code d'avant (rien à corriger), pouvoir discriminant prouvé par mutation : garde retiré → ce test ET `TestOutboundCallsAreNetguarded` rouges) : d'abord, vérifier si `fetchGameCMSImage` sort réellement sur le réseau en
  démo (couverture `netguard`). Ensuite, en démo, aucun `PersistBinary` sous le cache réel : les
  écritures vont sous `<démo>/runtime/`, ou n'ont pas lieu. Si le réseau sortait en démo : trou
  de l'hermétisme réseau, corrigé ici (le ratchet `netguard_coverage_test.go` doit le voir),
  avec son test.
- [x] **B-C4 (C5)** (`config.go` 640 → 629 lignes : `loadStatePaths`, `demoFixturesDirFromEnv`, `statePaths.backupConfig` et `statePaths.persistBatchAsync` dans `config_demo.go`, comportement identique ; gel `config/config_size_ratchet_test.go` ≤ 629, rouge à 640 puis vert) : `config.go` revient à au plus 629 lignes, les ajouts démo passant dans
  `config_demo.go`.
- [x] **B-C5 (C6)** (`sync/schemadrift/drift.go` : `Report` → `reportRetiredIndexes`, WARN `schema_drift_healed` `action=dropped` par index présent avant le soin et absent après ; `action=created` ajouté aux créations ; test d'intégration `sync/schema_index_retired_log_test.go` : 2 index recréés → 2 WARN exactement, rouge (0) puis vert ; base à jour → aucun WARN) : le retrait convergent d'un index par `EnsurePlayerSchema` est journalisé
  (`schema_drift_healed`, ou un log dédié de même contrat) seulement quand un index a vraiment
  été retiré. Test : index recréé, puis ouverture, puis log présent ; base à jour, puis
  ouverture, puis aucun log.
- [x] **B-C6 (code mort)** (`domain/title/demo_layout.go` : `Root()` retiré, aucun appelant ni test ; anti-résurrection `domain/title/demo_layout_no_root_test.go` par réflexion, rouge puis vert) : `title.DemoLayout.Root()` est supprimé.
- [x] **B-C7 (R2-1)** (`service/replay_service.go` : deux racines, `repoRoot` = données versionnées, `runtimeRoot` = artefacts (`ReplayArtifactPath`/`ReplayArtifactsDir`), `NewReplayServiceRoots` ; `NewReplayService` = deux racines égales, inchangé pour la CLI et les tests ; construction sortie de `registry_pages.go` (632 → 620 L) vers `wire/registry_replay_service.go` `replayServiceFrom` : `cfg.RepoRoot` + `cfg.RuntimePaths()` ; aucun fichier du §3.1.6 touché ; test HTTP par le vrai `ReplayHandler` `wire/registry_replay_service_demo_test.go` : démo → fond versionné 200, artefact sous runtime 200, artefact du dépôt 404 ; rouge 404 sur le fond avant, vert après ; témoin hors démo) : en démo, le service de rejeu lit les données VERSIONNÉES (fonds, mappings,
  libellés, zones, règles de tiers) depuis la racine du dépôt, et les artefacts d'exécution
  (rejeux, rasters, faits) depuis `<démo>/runtime/`. Test : en démo, le fond de carte d'une carte
  versionnée est servi (200), et un artefact est lu sous `runtime`.
- [x] **B-C8 (R2-2)** (`wire/prestige_lazy_service.go` `SquadUsualContexts` : `resolveWithPlayerDB`, titre = `pdb.TitleSlug`, le titre de l'appelant n'a plus voix ; test d'intégration par le vrai chemin HTTP `wire/prestige_squad_usual_contexts_title_test.go` : `GET /squads?user_id=…` sans `title_slug` → handler → service paresseux → bundle → fournisseur ; rouge `halo_5 : [Arene Campagne]` puis vert, témoin `halo_infinite` 2 playlists) : `SquadUsualContexts` prend le titre du `PlayerDB` résolu. Le test de
  comportement passe par le handler, sans slug en dur.

- [x] **B-C9 (CI de branche, run `36328284365`, job « Go Coverage + Baseline »)** (`config.LoadForCLI` dans `config/config_demo.go` : `DemoMode` tel que l'environnement le pose, AUCUNE redirection d'état B5.2/B5.5 — `statePaths.demo` faux, `demoState()` pour `RuntimePaths`/`WatcherTokensDir`/`TitleSettingsPath` ; `Load` inchangé pour le serveur ; `config.go` reste à 629 lignes (champ `stateFromRepo`, `load(fromRepo)`) ; `cmd/levelup/main.go` → `LoadForCLI` pour TOUTES les sous-commandes, statuées au journal ; E2E `ops/seed_demo_cli_test.go` joué sans la variable ET avec `LEVELUP_DEMO_MODE=true`, rouge `…/data/demo/db_profiles.json: … introuvable` (erreur de la CI) puis vert ; `config/config_cli_test.go` : chemins d'état en démo = hors démo, témoin serveur redirigé) : `seed-demo` sous `LEVELUP_DEMO_MODE=true` lit les vrais profils et les vraies bases ; son comportement ne dépend pas de la variable.
- [x] **B-C10 (inversion du choix de B-C9, DB-40)** (`config/config_demo.go` : `Load()` = chemins d'état du dépôt en démo, `DemoMode` lu de l'environnement, sémantique d'avant B5 ; `LoadServer()` = redirections B5.2/B5.5 + coupure de la file persist, appelé par `cmd/server/main.go:311` et lui seul ; `LoadForCLI` supprimé sans alias, `cmd/levelup/main.go` revenu à `Load()` ; inventaire des appelants au journal ; tests d'hermétisme `config_demo_hermetic_test.go` (tests renommés `TestLoadServer_*`) et `cmd/server/demo_paths_test.go` (`chargerCfg`, donc aussi le manifeste des tâches de fond) sur `LoadServer`, manifeste renforcé des écritures d'exécution du serveur (sessions, `jobs.json`) ; `config_cli_test.go` réécrit en `config_load_state_test.go` ; rouge `Load` en démo redirige puis vert ; mutation `LoadServer` sans redirection → les trois familles d'hermétisme rouges ; docs CONFIGURATION EN + FR) : la redirection démo des chemins d'état appartient au seul processus serveur, les autres binaires n'en héritent plus.

**Gate** : GO-F (paquets touchés, puis suites complètes), lint, et la baseline de tests si un test
est supprimé.
- [ ] **Fusion 2** : CI de branche verte au niveau job, fusion dans `feat/v75`, CI de `feat/v75`
  au niveau job. Puis **vérification sur données réelles par le superviseur**, après mise à jour
  du checkout principal :
  - redémarrage du serveur ;
  - logs de boot : premier tir du cron à +2 min, persistance réussie ou « snapshot récent
    présent » ;
  - aucune ERROR d'ouverture de player DB ;
  - `duckdb_indexes()` sans `idx_msr_*` sur les player DB.

---

## 6. Découvertes

### Lots A

- DA-1 (enquête) : `_fin_partie/livraison/end_*_music_01.wav` sur le Bureau sont encore en
  4 canaux. Les copies du dépôt sont les bonnes. Il faut toujours réduire en stéréo PCM.
- DA-2 (enquête) : les couleurs hex de `lib/halo/teamNames.ts` ne correspondent pas aux 8
  couleurs de l'annonceur. Ce sont des couleurs d'affichage ; A5.0 établit la table de
  l'annonceur à part. `teamNames.ts` n'est pas modifié sans preuve.
- DA-3 (A1.0, 2026-09-26) : H1 DÉPEND DE LA VITESSE sur le témoin `000d5950`. Simulation
  déterministe (piste réelle, durées réelles des WAV, horloge qui avance, 10 graines) : à 1×,
  3 à 4 voix occupées à la borne, voix ET fanfare jouées 10 fois sur 10 ; à 1,5×, 2 voix,
  tout passe ; à 2× (`SOUND_MAX_SPEED`), 6 à 7 voix, fanfare refusée 7 fois sur 10. Deux
  limites : la fixture ne porte pas les kills (vue match), dont les sons s'ajouteraient — les
  chiffres sont des minimums ; et rien ne dit à quelle vitesse l'utilisateur a constaté le
  défaut. S'il l'a constaté à 1× sur une fin comparable, une autre cause reste possible et
  A1.6 le dira. Non traité ici (hors périmètre de A1).
- DA-4 (A1.0) : la piste sonore du témoin au schéma 71 compte 334 événements (504 `shots` +
  rafales), là où les commentaires de `SOUND_MAX_VOICES` (`replayAudio.ts`) et de
  `replaySound.ts` citent « 483 tirs sonores » et « 46 sources refusées » : relevés datés du
  2026-08-15, antérieurs au modèle des rafales (M4b). Chiffres historiques, non corrigés.
- DA-5 (A1.5, 2026-09-27) : sur les données de ce poste, la vue match des matchs sondés est
  PARTIELLE (`scoreboard_empty`) : `readVictory` ne lit jamais deux camps, donc le rejeu n'y joue
  une conclusion que sur une VICTOIRE (réplique « Vainqueur » sans camps) et jamais sur une
  défaite ou une égalité — l'écran de fin est touché de la même façon. Le témoin `000d5950` n'y
  joue rien. Donnée du poste (cf. mémoire « plusieurs postes, jeux de données différents »), pas un
  défaut du code : non traité. L'écoute A1.6 se fait sur les données de l'utilisateur.
- DA-6 (A2.0, 2026-09-27) : sous Chromium, `page.evaluate` de Playwright s'exécute avec
  `userGesture=true` (sondé : `hasBeenActive` vrai avant tout geste). Une recette de son qui évalue
  du code dans la page avant son premier geste mesure donc un document déjà activé. À savoir pour
  les recettes à venir (A4 : musique d'intro).
- DA-7 (A2.4) : dans Firefox sans tête, sur ce poste chargé, le contexte audio passe `running`
  0,8 à 4,3 s après sa création dans un geste, plus lentement quand la lecture tourne déjà (lecture
  automatique). Les sons lancés entre-temps partent à la reprise. Hors de portée de la page (le
  contexte ne peut naître qu'au geste) ; à confirmer à l'oreille en A2.5.

### Lots B

- DB-1 (enquête) : `cmd/levelup/cmd_restore_csr.go:101` fait un DELETE sur une table
  append-only (ADR 0026).
- DB-2 (enquête) : `api/wire/post_sync_deltas_snapshot.go:238` contient le nombre magique
  `outcome = 2`. B4 modifie la même fonction sans le toucher.
- DB-3 (enquête) : l'entrée backlog de l'item 1 est inexacte sur trois points : tir immédiat,
  pas de délai de 30 s ; `ErrDrainTimeout` n'existe pas ; les logs du 20/09 ne sont plus
  disponibles. L'entrée sera réécrite à la clôture.
- DB-4 (enquête) : `.ai/V7.5/REGISTRE_REPORTS.md:543-544` cite `repair_psa_index` et
  `repair_msr_index` comme détecteurs périodiques. Ligne à réécrire par le superviseur à la
  clôture de B3.
- DB-5 (B1, 2026-09-26) : le compteur `swapFailuresTotal[drain_timeout]` et le WARN « drain
  timeout, rollback vers RO » (`provider_writer.go:53-56`) comptent aussi une vidange finie par
  le contexte de l'APPELANT (annulation, délai), que `ErrDrainTimeout` distingue désormais. La
  métrique mêle donc encombrement et arrêt demandé. Non traité (hors périmètre).
- DB-6 (B1) : `scheduler/world_leaderboard_cron_test.go` fait 613 lignes (> 500). Préexistant ;
  les nouveaux tests vont dans `world_leaderboard_persist_retry_test.go`. Non traité.
- DB-7 (B2, 2026-09-26) : `arbitration_clocks_utc_guard_test.go` n'appelle aucune méthode qui
  exécute du SQL. Avec le critère retenu par le garde-rail (fichier qui appelle `Exec*` ou
  `Query*`), il ne serait pas relevé sans sa dispense : la dispense demandée par le plan est
  donc préventive. Elle est gardée, datée, et le garde-rail échoue si le fichier disparaît.
- DB-8 (B2) : `TestLUSRV2Shadow_RafalesBornees_300Candidats` (`sync/skill`, tag `integration`)
  mesure une détention de rafale « < 2 s » à l'horloge murale. Sous la charge du poste (deux
  `replay-equiv` d'une autre session), des rafales montent à 2,00-2,04 s et le test rougit.
  Préexistant : rouge aussi sur `d61443ef5` (worktree jetable, log
  `B2-base-d61443ef5-skill.log`). Test fragile sous charge ; non traité.
- DB-9 (B2) : `cmd/diag_exec/main.go` ouvre la DB par un `sql.Open("duckdb", …)` direct en
  RW (anti-patron « bare connect »). Outil de diagnostic hors image ; préexistant, non traité.
- DB-10 (B2) : sur ce poste, `go test -tags=integration -p 1` de `internal/platform/duckdb`
  dure 872 s seul et `internal/sync` 577 s, contre un délai par défaut de 10 min : la suite
  complète sans `-timeout` les tue sous charge. La recette GO-F du plan (§3.3) ne fixe pas de
  `-timeout`. Non traité.
- DB-11 (B3.1, 2026-09-27) : `loadExistingCSRMatchIDs` (`sync/csr_writes.go`,
  `rating_type = 'CSR'`) emprunte `idx_msr_rating_type` à l'exécution (30 fois sur 30) et
  en est ~10× plus LENT : 20,3 ms avec l'index, 1,9 ms sans. C'est la seule des sept formes
  qui prend l'index. Tant que l'index reste, cette lecture est aussi exposée à une réponse
  fausse sur un index désynchronisé (mécanisme du 2026-09-13). Non traité.
- DB-12 (B3.1) : les témoins C1/C2 (`COUNT(*) WHERE playlist_group = 'h5_arena'`, forme de
  `cmd/purge_foreign_lusr_chain`) empruntent `idx_msr_playlist` sur une chaîne rare, ce qui
  confirme la prémisse du lot. F6a/F6b (`loadPreviousLUSRRating`,
  `loadPreviousDisplayedOrdinal`) restent séquentielles sur les chaînes courantes de la
  fixture. Non mesuré : une chaîne encore peu peuplée (premiers matchs d'un joueur dans une
  chaîne) les ferait vraisemblablement passer par l'index, avec un risque de delta ou de
  palier faux. Non traité.
- DB-13 (B3.1) : `EXPLAIN` affiche toujours un scan séquentiel en DuckDB 1.5.5. La stratégie
  se choisit à l'exécution et seul `EXPLAIN ANALYZE` la révèle (déjà écrit dans
  `RAPPORT_VOLET2_INDEX_PSA_2026-08-28.md` §3.1). Le libellé de B3.1 (« EXPLAIN et temps »)
  et la pièce « Banc EXPLAIN » (`psa_index_repro_planprobe_test.go`) conduiraient à conclure
  « index jamais emprunté ». La mesure B3.1 passe par `EXPLAIN ANALYZE`, calibré par un
  témoin PK. Non traité ailleurs.
- DB-14 (B3, 2026-09-27) : `migration/steps_player_lusr_components_append_only.go:17-20` garde
  `idx_lch_component` et `idx_lch_match` en invoquant « le même raisonnement que les idx_msr_* »
  (append-only = pas de surface ART). Ce raisonnement est réfuté deux fois par la mesure :
  PSA le 2026-09-20, MSR le 2026-09-27, la désynchronisation se reforme sur des INSERT purs.
  Le commentaire est désormais une doc inversée. La même question se pose pour les autres
  index secondaires des tables append-only (`idx_pme_match_lookup`, `idx_pcs_lookup`,
  `idx_lch_*`). Non traité.
- DB-15 (B3) : `cmd/purge_foreign_lusr_chain` capture et REJOUE tous les index présents sur
  `match_skill_rank` au moment du swap (`purgeForeignChain`). Sur une base où un binaire ancien
  aurait recréé les `idx_msr_*`, la purge les reposerait ; le prochain `EnsurePlayerSchema` les
  retire. Non traité (hors de la liste B3.5).
- DB-16 (B3) : plusieurs fixtures de test créent encore des `idx_msr_*` dans une DDL locale de
  schéma ancien, et ne représentent donc plus la prod. Fichiers : `sync/csr_art_repro_test.go`,
  `csr_backfill_integration_test.go`, `csr_writes_integration_test.go`,
  `lusrdb_helpers_test.go`, `sync/skill/skill_rating_loaders_test.go`,
  `persist/lusr_append_only_persister_test.go`,
  `games/halo_infinite/migrations/player_match_skill_rank_test.go`. Sans effet sur les tests.
  Non traité.
- DB-17 (B3.9) : les quatre player DB de `halo_infinite` du checkout principal ont un
  `stats.duckdb.wal` à côté d'elles (dernières écritures du 23/09). Le dernier arrêt du
  serveur n'a pas fait de CHECKPOINT, et aucune copie cohérente n'est possible sans rejouer ce
  WAL. Les player DB `halo_5` de JGtm, XxDaemonGamerxX et des joueurs de démo n'en ont pas.
  Non traité.
- DB-18 (B3.9) : aucune CLI ne migre une player DB désignée (`applyMigrationsOnDB` n'est
  exposée par aucune sous-commande prenant un chemin). Le véhicule de B3.9 est donc un test
  paramétré par `LEVELUP_B3_PLAYER_DB_COPY`. Non traité.
- DB-19 (B3.9, 2026-09-27) : le rejeu du WAL du 23/09 (`halo_infinite/JGtm`, 745 o)
  RÉUSSIT sur une copie, sans erreur ART. Le prochain démarrage du serveur rejouera ce même
  WAL sur l'original ; la copie ne montre aucun obstacle. Les trois autres player DB
  `halo_infinite` (DB-17) n'ont pas été essayées.
- DB-20 (B3.9) : la player DB `halo_5/JGtm` avait 12 étapes de retard sur la chaîne player.
  Parmi elles : `player_msr_view_latest_by_type_v1` (13/09), `drop_arc_titles`,
  `player_dominance_flag_reset_none_v1`, `create_personal_score_awards_player_v1`,
  `drop_psa_*` et les DEFAULT UTC. Cela confirme que les player DB Halo 5 sont hors de la
  boucle de migration du boot (commentaire de `sync/schema_msr_views_test.go`) : pour elles,
  seul `EnsurePlayerSchema` agit. Le retrait des index MSR et PSA les couvre donc par le
  soin (B3.4), pas par le step. Autre point : le véhicule migre par `RunForDB`, qui force
  le slug par défaut. Pour le type player, `halo_5` délègue aux étapes de `halo_infinite`, le
  résultat est donc le même. Non traité.
- DB-21 (B3.9) : `match_skill_rank` de `halo_infinite/JGtm` compte 33 702 lignes, 2,8 fois
  les 12 000 de la mesure D-4. Le critère relatif compare deux plans séquentiels qui croissent
  pareil avec le volume, mais il n'a pas été mesuré à ce volume. F4 (seul lecteur qui prenait
  l'index) y était plus lent AVEC l'index. Non traité.
- DB-22 (B4, 2026-09-27) : `service.FanoutService` (`BuildPlan`, `Execute`) n'a AUCUN appelant
  de production (seuls ses tests) : code mort au sens de la règle n°7. Non traité.
- DB-23 (B4) : `CampaignSampleProvider.LoadAxisSamples` (filtre `mr.start_time >= ? AND <= ?`,
  tri) et `EngagementScoreRepo.ListRecentPvPMatchIDs` (`mr.start_time IS NOT NULL`, tri) lisent
  `start_time` brut, contre la règle n°8. Non traité.
- DB-24 (B4) : `ListRecentPvPMatchIDs` ignore une erreur de `Scan` (`if err == nil`), et la
  lecture des coefficients juste au-dessus fait `continue` sur erreur, sans log (anti-patron
  n°10). Non traité.
- DB-25 (B4) : `QKillsBetweenPlayers` (frags échangés entre deux joueurs, lu par
  `GetEncounterStats` et l'Explorateur) n'exclut pas la Campagne. Hors critère (source
  `match_kill_events_latest`, pas `match_participants`). La base crédit de Halo 5 y verse
  `killer_victim_pairs` : des frags de Campagne peuvent donc compter dans ce duel, alors que
  le nombre de rencontres ne les compte plus. Non traité.
- DB-26 (B4) : `SnapshotPlayerState` → `loadEarnedMedalIDs` (`medals_earned`, hors critère)
  n'exclut pas la Campagne : une médaille obtenue seulement en Campagne compte comme « déjà
  obtenue ». Non traité.
- DB-27 (B4) : la sonde du garde sur `internal/sync`, `internal/ops` et `cmd` relève 56
  lecteurs sans exclusion (17 + 10 + 29), liste au journal B4. Tri séparé (§1.4). Non traité.
- DB-28 (B5, 2026-09-27) : `RequireAdmin` (et `RequireAuth`) sont transparents en démo
  (`middleware/require_admin.go:18`) : TOUTES les actions d'administration sont ouvertes à un
  visiteur de la démo publique. B5.6 ne ferme que `POST /settings/backup/run`. Les autres
  écritures déclenchées par requête gardent leurs chemins sous `LEVELUP_REPO_ROOT` (actions admin de
  `wire/registry_actions.go`, qualité des données, construction locale d'un rejeu par
  `wire/registry_replay_build.go` — fichier du §3.1.6 —, recalcul des sessions qui écrit les player
  DB de la fixture…). Non traité.
- DB-29 (B5) : la file de build lit l'artefact déposé sous le dépôt
  (`wire/registry_build_queue.go:393`, `requireArtifactBeforeSuccess`) et l'écrit par
  `registry_replay_build.go` (§3.1.6), alors que le lecteur du rejeu passe désormais par
  `cfg.RuntimePaths()`. Identique hors démo ; en démo, n'est atteignable qu'avec
  `LEVELUP_BUILD_WORKER_TOKEN` posé. Non traité.
- DB-30 (B5) : conséquence pour le conteneur `levelup-demo` de production : sessions, logs par
  module (jusqu'à 100 Mo × 4 par module) et caches d'exécution vont désormais sous
  `/app/data/demo/runtime/`, donc sur l'hôte dans `./data/demo/runtime/` (bind-mount RW), au lieu du
  système de fichiers éphémère du conteneur. Les sessions survivent donc au redémarrage du
  conteneur, et le disque du VPS porte ces fichiers. Non traité (à apprécier au déploiement).
- DB-31 (B5) : une variable explicite (`LEVELUP_SESSION_DIR`, `LEVELUP_AUTH_DIR`,
  `LEVELUP_LOGS_DIR`, `LEVELUP_DB_PROFILES`, `LEVELUP_APP_SETTINGS`, `LEVELUP_BACKUP_DIR`) garde la
  main en démo (contrat documenté) ; posée dans le `.env.local` du dépôt, elle ramènerait l'écriture
  sous le dépôt. Aucune n'y figure sur ce poste (vérifié le 2026-09-27). Non traité.
- DB-32 (B5) : lecteurs d'overlay de titre encore dérivés du dépôt, hors des chemins servis en
  démo : `wire/post_sync_deltas.go:219` (post-sync), `wire/registry_replay_notify.go:111` (boucle
  coupée en démo), `config.CSRSeasonIDForTitle` (`config.go:484`). Non traité.
- DB-33 (B5) : `internal/config/config.go` passe de 629 à 640 lignes (seuil 500, dette
  préexistante) ; la bascule démo vit dans `config_demo.go` pour limiter la croissance. Non traité.
  **Traité par B-C4 (2026-09-27)** : retour à 629 lignes, gel par `config/config_size_ratchet_test.go`.
- DB-34 (B-C1, 2026-09-27) : deux contrats de refus démo coexistent. `demo_mode_forbidden` (403,
  `handlers.refuseInDemo`, 4 routes) et `demo_mode_unsupported` (422, en ligne dans
  `handlers/settings.go` : `PATCH /settings`, `POST /settings/media/scan`). Non traité.
- DB-35 (B-C1) : `PATCH /profiles/{p}/titles/{t}/sync` (`TitleSyncHandler.SetSync`) reste ouvert en
  démo et écrit `db_profiles.json` (la fixture, montée en écriture en production), comme les
  routes fermées par B-C1. Hors des trois routes du triage ; famille de DB-28. Non traité.
- DB-36 (B-C1) : `WatcherHandler` construit son magasin de tokens mono-utilisateur sur
  `title.NewPathResolver(cfg.RepoRoot).WatcherTokensPath()` (`watcher_handler.go:55`), pas sur
  `<démo>/auth`. `POST /watcher/auth/start` est atteignable en démo (`RequireAdmin` transparent) et
  lance un device code flow, que l'allowlist netguard déclare « CLI uniquement »
  (`platform/auth/xbox_device_code.go`). Sortie réseau et écriture sous le dépôt non vérifiées.
  Non traité.
- DB-37 (B-C8) : le paramètre de requête `title_slug` de `GET /squads` (`listMySquadsInput`) n'a
  plus d'effet : l'indice d'escouade prend le titre du PlayerDB résolu. Le retirer change le
  contrat OpenAPI (et les types web générés). Non traité.
- DB-38 (B-C7) : l'overlay NON versionné des socles
  (`data/titles/<slug>/reference/generated/map_weapon_pads.json`, sortie de runtime) est lu par le
  service de rejeu à la racine du DÉPÔT, en démo comme ailleurs : il vit sous
  `data/titles/*/reference/**`, lecture classée légitime en B5. Non traité.
- DB-39 (B-C2) : en démo, la section Identités n'a plus de témoin disque : `dir_exists` et
  `db_exists` valent `false` pour tous les profils de la fixture (dont les dossiers portent un autre
  nom que la clé de profil : `DemoPlayer` → `DEMO`). Affichage honnête (aucune lecture), mais
  appauvri. Non traité.
- DB-40 (B-C9, 2026-09-27) : 42 autres binaires opérateurs de `cmd/*` appellent `config.Load()` (ex.
  `token-capture`, `token-import`, `restore`, `backup-once`, `h5-sync`, `refresh-metadata`). Sous
  `LEVELUP_DEMO_MODE=true`, ils suivent les redirections d'état du serveur démo (profils, réglages,
  auth, tokens vers `<démo>/…`) : `token-capture` écrirait par exemple dans le magasin de tokens de la
  démo. B-C9 ne couvre que `cmd/levelup`. Non traité.
  **Traité par B-C10 (2026-09-27)** : `config.Load()` ne redirige plus, seul `cmd/server` appelle
  `config.LoadServer()`.
- DB-41 (B-C9) : sous `LEVELUP_DEMO_MODE=true`, la CLI garde les effets de `DemoMode` antérieurs à B5 :
  `config.SharedDBPath`, `MetadataDBPath`, `PrestigeBundleDBPaths` et la résolution des player DB
  visent la fixture démo. Une sous-commande de sync lancée dans cet environnement écrirait donc dans
  la fixture. Préexistant, non traité.

---

## 7. Journal

### Lots A

**[2026-09-26] A1 — musique de fin du rejeu 2D (item 11) — exécutant opus, worktree du plan.**

- A1.0 : simulation `sound/endMatchVoiceCap.witness.test.tsx`. Fixture chargée par
  `goFixtureEntries()` + `loadGoFixture()`, normalisée par `testReplayDoc` (seule porte admise par
  `testDoc.guard.test.ts`). Borne = `replayWindow` avec l'en-tête mesuré (t0 18 465 ms, 478 s),
  soit l'image 4 929. `fetch` rend les octets des WAV de `static/sounds/halo_infinite/`. Le
  décodage lit la durée RIFF. L'horloge avance à 60 i/s murales et `ended` tombe à l'heure
  d'arrêt programmée. Hasard semé, kills vides (absents de la fixture).
  - Graine 20260926, 1× : 4/8 voix occupées à la borne, 333 sources tirées, conclusion jouée
    `[end_victory_voice_fr_02.wav, end_victory_music_01.wav]`. H1 NON reproduite à 1×.
  - Sonde temporaire (10 graines × 1× / 1,5× / 2×, supprimée) : voir DA-3. À 2×, graine 7919 :
    7/8 occupées, 328 sources, conclusion `[end_victory_voice_fr_02.wav]` — la voix prend la
    8e place, la fanfare est refusée. Critère « H1 confirmée » du plan rempli à 2× : le lot
    continue, avec l'écart signalé (DA-3).
- A1.1 : `replayAudio.ts:97` `soundOccupiesVoice` (règle unique + doctrine, consommée aussi par
  `applyVoiceCap`) ; `:339` `playConclusion` ; `:344` `start` privé commun à `play` et
  `playConclusion` (le compteur `voices` n'est touché que si la source occupe une voix).
- A1.2 : `useReplaySound.ts:586` → `player.playConclusion(url)`. 675 lignes avant, 675 après.
- A1.3 : `replayAudioMix.ts:226-238`. Le commentaire dit l'exception commune et son historique.
  `MixedSound.conclusion` renvoie à `soundOccupiesVoice` au lieu de répéter la doctrine.
- A1.4, sorties rouges (code de production inchangé, log `A1-4-rouge.log`), `EXIT_ROUGE=1`,
  4 échecs sur 71 :
  - (a) `expected [ FakeSource{ …(6) }, …(7) ] to have a length of 10 but got 8` ;
  - (b) et (c) `TypeError: p.playConclusion is not a function` ;
  - (d) `vitesse 2x — voix occupées à la borne : 7/8, sources tirées : 328, conclusion jouée :
    [end_victory_voice_fr_02.wav]: expected [...] to include 'end_victory_music_01.wav'`.
  - (d) a été re-vérifié rouge après le passage à `testReplayDoc`, en remettant un instant
    `player.play(url)` (log `A1-4d-rouge-testReplayDoc.log`).
  - Après correctif : `EXIT_VERT=0`, 71/71.
- Gates WEB (logs `$TEMP\backlog-gates\A1-*.log`) :
  - `EXIT_TYPECHECK=0` (après purge de `node_modules/.tmp`) ;
  - `EXIT_LINT=0` (26 avertissements, tous préexistants ; aucun sur une ligne touchée) ;
  - `EXIT_VITEST_SOUND=0` : 19 fichiers réussis et 1 sauté, 335 tests réussis et 1 sauté ;
  - `EXIT_VITEST_MATCH_REPLAY=0` : 216 fichiers réussis et 4 sautés, 3 179 tests réussis et 7 sautés.
  - Un premier lancement a buté sur « Timeout waiting for worker » (démarrage du worker, pas un
    test). Relancé tel quel : vert.
- A1.5 : `/health` → `000`. Non faite, reste au superviseur, à 2× (cf. statut de l'item). Faite le
  2026-09-27 par l'exécutant de A2 (entrée suivante).
- A1.6 : ouvert (superviseur).

**[2026-09-26] Superviseur — vérification de A1 et dérogation d'ordre.**

- Diff `a373539d9` relu : règle unique `soundOccupiesVoice`, `start` privé commun, compteur de
  voix touché seulement pour une source soumise au plafond ; `useReplaySound.ts` reste à 675
  lignes.
- Gates rejoués : `EXIT_TYPECHECK=0` (cache purgé), `EXIT_LINT=0` (0 erreur, 26 avertissements
  préexistants), `EXIT_VITEST=0` (216 fichiers, 3 179 tests réussis, 7 sautés préexistants).
- A1.5 et A1.6 restent ouverts ; ils se feront avant la fusion 1, à 2×.
- **Dérogation §1.2 appliquée** : à 21 h 11, une autre session fait toujours tourner deux
  `replay-equiv` et des builds Go. Le serveur API n'est pas démarrable, donc A2 attend et B1
  passe devant.

**[2026-09-27] A1.5 — recette navigateur de la musique de fin — exécutant opus (lot A2).**

- Environnement : API `:8000` démarrée par le superviseur (`/health` → 200), Vite du worktree
  détaché sur `:5174` (proxy vers `:8000`), script `apps/web/.tmp.recette-fin.mjs` (supprimé).
  Chromium sans tête, `--autoplay-policy=no-user-gesture-required`. Son activé, vitesse posée par
  `replay-speed`, curseur posé par la frise à 200 images (20 s) avant la borne, puis « Lecture ».
  Instrumentation : `AudioBufferSourceNode.prototype.start` (durée du tampon, fichier retrouvé par
  `fetch` → `decodeAudioData`), sources actives comptées par un écouteur `ended` posé par
  `addEventListener` (Chromium émet parfois deux `ended` pour une source : dédoublonnés).
- `000d5950` (JGtm), 2× et 1× : **aucune conclusion**, 0 prise `end_*` téléchargée. Sur ce poste
  la vue match est partielle (`partial_reasons` = `scoreboard_empty`, `events_empty`), l'issue
  est « défaite » (`outcome_code` 3, « Super Fiesta » sur Dévissage), donc `readVictory` ne lit pas
  deux camps et `endMatchSoundSpec` rend `null` (seule une victoire a une réplique sans camps).
  Aucun lien avec le correctif ; même constat sur les autres matchs du cache sondés (`28c9b538`,
  `4f77afc1`, `94a28b8b` : `scoreboard` vide). Voir DA-5.
- Témoin dense substitué `ac03413d-2c03-4e3e-9e9c-c10165e40be0` (Chocoboflor, victoire 50-48,
  `outcome_code` 2 → réplique « Vainqueur » + fanfare de victoire) :
  - 2×, passe 1 : voix `end_winner_voice_fr_01.wav` (1,26 s) et fanfare `end_victory_music_01.wav`
    (10,59 s) démarrées, **7 sources actives à la borne** ;
  - 2×, passe 2 : les deux démarrées, **8 sources actives à la borne** (le plafond est plein : avant
    le correctif, ni la voix ni la fanfare ne seraient passées) ;
  - 1× : les deux démarrées, **7 sources actives à la borne**.
- Témoin calme `4bd6de5a-98de-45cc-8c10-47f58280d2b9` (Chocoboflor, victoire 50-40, 0 tir dans les
  5 dernières secondes du film) : 2× → les deux démarrées, 5 sources actives à la borne ; 1× → les
  deux démarrées, 0 source active. Inchangé.
- Une passe isolée de `000d5950` à 1× a vu 8 sources lancées et aucune terminée (`ended` jamais
  reçu, contexte figé) ; rejouée deux fois, normale (60 et 48 `ended`). Aléa du navigateur sans
  tête sur un poste chargé, non retenu.

**[2026-09-27] A2 — son perdu au rechargement ou en changeant de rejeu (item 12) — exécutant opus.**

- Environnement :
  - même Vite `:5174` que A1.5, API `:8000` du superviseur ;
  - script `apps/web/.tmp.recette-son.mjs` (supprimé) ;
  - Chromium et Firefox 153 (`npx playwright install firefox`) SANS TÊTE, **aucun drapeau
    d'autoplay** ;
  - préférences : `replay-sound-on`=true, `replay-speed`=1 ;
  - rejeu A `ac03413d` (image 3 123), rejeu B `94a28b8b` (image 3 481), joueur Chocoboflor,
    passage dense (environ 95 tirs en 6 s).
- **Piège de mesure écarté (DA-6)** : avant le geste, AUCUN `page.evaluate`. Tout passe par la
  console de la page ; les gestes sont de vraies entrées (`mouse.click`, `keyboard.press`).
- Politique vérifiée par sonde, dans les deux navigateurs :
  - contexte créé hors geste → `suspended` ;
  - créé dans `pointerdown`, `click` ou `keydown` (touche « a ») → `running`.
  - Firefox met 0,4 à 1,2 s à passer `running` : un premier relevé à 400 ms disait `suspended` à
    tort.
- Mesures par chemin, sur une fenêtre de 6 s après le geste :
  - état du contexte ;
  - délai geste → premier `start()` ;
  - délai geste → contexte `running` ;
  - fin du dernier décodage ;
  - sons sautés = sons de la même fenêtre rejouée tiède (pause, même image, relecture) moins les
    sons à froid. Ce chiffre est bruité par le plafond de voix (valeurs négatives possibles).
- **A2.0, avant correctif** (log `$TEMP/backlog-gates/A2-0-recette-avant.log`) :

  | Chemin | Chromium | Firefox | Verdict |
  |---|---|---|---|
  | 1 rechargement + Lecture | running ; 1er son 0,35 à 1,4 s ; sautés 1 à 3 ; décodage ≤ 1,0 s | running à 0,8 / 1,0 s (une passe : jamais en 6 s, 23 sautés) ; 1er son 0,6 à 2,2 s | pas reproduit |
  | 2 rechargement + Espace | running ; 1er son 0,48 s ; sautés 1 | running à 1,4 s ; 1er son 0,5 à 1,3 s ; sautés 2 à 6 | pas reproduit |
  | 3 clic frise, puis Lecture | clic frise : 0 contexte ; 1er son 0,6 à 1,5 s après Lecture ; sautés 1 à 16 | clic frise : 0 contexte ; 1er son 0,9 s ; sautés 1 | pas reproduit (le clic frise n'ouvre rien) |
  | 4 lecture auto + clic neutre | **0 contexte, 0 son**, avant et après le clic | **0 contexte, 0 son** | **REPRODUIT** |
  | 5 A joué, lien vers B, Lecture | contexte neuf au geste ; 1er son 0,19 s ; sautés 1 | 1er son 0,46 s ; sautés 11 | pas reproduit |
  | 5b lecture auto, A, lien vers B | **B muet** : aucun contexte, 0 son | **B muet** | **REPRODUIT** |

- Cause retenue :
  - (a) la lecture automatique démarre sans geste de transport, et rien d'autre n'ouvre le
    lecteur ;
  - (d) le lien remonte le composant, qui ne profite pas du geste déjà reçu ;
  - (b) écartée : décodage ≤ 1,1 s (une passe à 2,6 s sous charge), donc pas de téléchargement
    anticipé ;
  - (c) écartée comme cause du silence : le contexte de Firefox démarre, en 0,8 à 1,4 s.
- A2.1 / A2.2 :
  - `sound/useAudioUnlock.ts` (53 lignes), appelé par `useReplaySound.ts:460`
    (`useAudioUnlock(on, wake)`) ;
  - `open` est lu par une ref : l'effet ne dépend que de la préférence ;
  - `useReplaySound.ts` : 675 → 675 lignes.
- **Écart au texte de A2.1 (`pointerdown`/`keydown` → `click`/`keyup`), prouvé** :
  - la variante `pointerdown`/`keydown` passée par les tests (log
    `A2-3-variante-pointerdown.log`, `EXIT_VARIANTE_POINTERDOWN=1`) casse « le premier clic sur
    le BOUTON DU SON active et ne coupe pas » et « la touche M active et ne coupe pas »
    (`expected false to be true` sur `on`) : le document ouvre le lecteur avant la bascule, qui
    coupe ;
  - `click` et `keyup` arrivent après le bouton (React, sous la racine) et après le raccourci
    (`keydown` sur la fenêtre) ;
  - la recette confirme qu'un contexte né dans `click` ou `keyup` démarre : l'activation est
    acquise dès `pointerdown` ou `keydown`.
- A2.3, sorties rouges (code de production inchangé, log `A2-3-rouge.log`), `EXIT_ROUGE=1`,
  4 échecs sur 9 :
  - clic : `expected +0 to be 1` ;
  - touche : `expected +0 to be 1` ;
  - écouteur retiré : `expected [] to deeply equal ArrayContaining ["click", "keyup"]` ;
  - `hasBeenActive` : `expected +0 to be 1`.
  - Les 5 autres (préférence « coupé », document jamais activé, bouton du son, touche M,
    démontage) sont des gardes, vertes avant comme après.
  - Après correctif : `EXIT_VERT=0`, 41/41 avec `useReplaySound.test.tsx`.
- **A2.4, après correctif** (log `A2-4-recette-apres.log`) :

  | Chemin | Chromium | Firefox |
  |---|---|---|
  | 1 rechargement + Lecture | running à 0,18 s ; 1er son 0,89 s ; sautés 4 | running à 0,94 s ; 1er son 0,64 s ; sautés 1 |
  | 2 rechargement + Espace | running à 0,17 s ; 1er son 0,60 s ; sautés 6 | running à 0,80 s ; 1er son 1,14 s ; sautés 2 |
  | 3 clic frise, puis Lecture | contexte ouvert AU CLIC FRISE ; 1er son 0,43 s après Lecture | ouvert au clic frise, running avant Lecture ; 1er son 0,79 s |
  | 4 lecture auto + clic neutre | 0 contexte avant ; running à 0,46 s ; 1er son 1,04 s | 0 contexte avant ; running à 0,81 s, puis 2,24 s (2 passes) |
  | 5 A joué, lien vers B, Lecture | contexte de B ouvert à l'affichage ; 1er son 0,13 s ; sautés 0 | idem ; 1er son 0,13 s ; sautés 0 |
  | 5b lecture auto, A, lien vers B | **B sonne sans geste** : 37 sons en 6 s, 1er à 0,15 s | **B sonne** : 27 sons, 1er à 0,27 s |
  | 7 lecture auto + touche « x » | running à 0,11 s ; 1er son 0,31 s | running à 4,35 s, puis 2,93 s (2 passes) |

  - En dev, `StrictMode` monte deux fois : le rejeu B compte deux contextes, dont le premier est
    fermé par le démontage simulé (3 au total avec A). Sans effet en production.
- Gates WEB (logs `$TEMP\backlog-gates\A2-*.log`) :
  - `EXIT_TYPECHECK=0` (après purge de `node_modules/.tmp`) ;
  - `EXIT_LINT=0` : 0 erreur, 26 avertissements préexistants, dont 4 `exhaustive-deps` sur
    `engine` dans `useReplaySound.ts`, sur des lignes non touchées ;
  - `EXIT_VITEST_MATCH_REPLAY=0` : 217 fichiers réussis et 4 sautés, 3 188 tests réussis et
    7 sautés.
- Nettoyage : Vite `:5174` arrêté (arbre du PID 23016), scripts `.tmp.*.mjs` supprimés,
  `routeTree.gen.ts` inchangé. Serveur `:8000` non touché.

### Lots B

**[2026-09-26] B1 — classement mondial perdu au boot (item 1) — exécutant opus, worktree du plan.**

- Méthode rouge d'abord : échafaudage minimal sans comportement (sentinelle déclarée non posée,
  seam `worldLeaderboardSleep`, constantes, champ `bootDelay` non lu), tests lancés, puis
  implémentation. Cache Go dédié `go-build-backlog`, une commande `go` à la fois.
- B1.1, sorties rouges (log `B1-1-rouge.log`, `EXIT_ROUGE=1`) :
  `TestProvider_DrainTimeoutIsTyped_integration` → `errors.Is(err, ErrDrainTimeout) = false,
  err = sharedprovider: drain inflight readers: context deadline exceeded`.
  `TestProvider_DrainCallerContextIsNotDrainTimeout_integration` est vert sur le code d'avant
  par construction (il ne produit jamais la sentinelle) : c'est la garde contre une enveloppe
  naïve. Preuve qu'il discrimine : mutant temporaire « sentinelle toujours posée » → rouge sur
  les deux sous-cas (`contexte appelant fini classé ErrDrainTimeout`, log
  `B1-1-mutant-naif.log`, `EXIT_MUTANT=1`), code remis. Après correctif : `EXIT_VERT=0`.
- B1.2/B1.3, sorties rouges (log `B1-2-3-rouge.log`, `EXIT_ROUGE=1`, 4 échecs) :
  - `RetriesPersistOnDrainTimeout` : `snapshots = 0 lignes / 0 lot(s), attendu 2 / 1` ;
    `AcquireWriter = 1 appels, attendu 3` ; `attentes = [], attendu 2 × 30s` ; `WARN de nouvelle
    tentative = 0, attendu 2` ; cycle rapporté en échec (`halo_infinite: persist snapshot: …
    drain timeout …`).
  - `PersistFailsAfterThreeDrainTimeouts` : `AcquireWriter = 1 appels, attendu 3 (bornage)` ;
    `attentes = [], attendu 2`.
  - `RetriesPersistStatsOnDrainTimeout` : `AcquireWriter = 2 appels / 0 attente(s), attendu 3 / 1` ;
    `stats persistées = 0, attendu 1`.
  - `RunWaitsBootDelay` : `cycle lancé avant le délai de boot (playlists=1, saison=2)`.
  - `NoRetryOnNonTransientWriterError` : vert avant (garde, l'ancien code ne retentait rien).
  - `RetryWaitIsCancellable` : ajouté après l'implémentation (garde de l'annulation ; il
    passerait aussi sur l'ancien code, qui ne faisait qu'un appel).
- Après correctif (log `B1-2-3-vert.log`) : 25 `TestWorldLeaderboardCron_*` verts — les 14
  historiques du cron, les 6 du garde-fou qualité et 5 nouveaux ; le 6e nouveau
  (`RetryWaitIsCancellable`) passe dans les gates du paquet complet.
- Fichiers : `world_leaderboard_cron.go` 467 → 482 lignes ; `world_leaderboard_persist_retry.go`
  77 lignes ; `ctxkeys` sort des imports du cron (l'étiquette est posée par `acquireWriterRetry`).
- Gates GO-S (logs `$TEMP\backlog-gates\B1-*.log`) : `EXIT_BUILD=0`, `EXIT_VET=0`,
  `EXIT_TEST=0` (scheduler, sharedprovider, archlint), `EXIT_INTEGRATION=0` (mêmes paquets,
  `-p 1`), `EXIT_LINT=0` (`0 issues.` ; un premier passage a relevé un gofmt sur le nouveau
  test, corrigé par `gofmt -w`, puis `EXIT_TEST_SCHEDULER_FINAL=0` sur l'état final). Aucune
  ligne `^--- FAIL:`.
- Écarts : deux retouches de doc hors des fichiers cités — interface `Provider.AcquireWriter`
  (`provider.go:105`) et commentaire de câblage `cmd/server/main.go:1388-1390`, qui décrivait
  un tir immédiat au boot. Deux tests de plus que la liste (stats enrichies, attente annulable).

**[2026-09-26] Superviseur — vérification de B1.**

- Diff `cd47936fc` relu : sentinelle posée seulement si le contexte de l'appelant est vivant ;
  nouvelle tentative bornée dans un fichier dédié ; délai de boot annulable.
- Gates rejoués avec le cache dédié : `EXIT_BUILD=0`, `EXIT_VET=0`, `EXIT_TEST=0` (scheduler,
  sharedprovider, archlint), `EXIT_INTEG=0` (scheduler, sharedprovider, `-p 1`), `EXIT_LINT=0`
  (`0 issues.`).
- À 21 h 46, l'autre session fait toujours tourner `replay-equiv` : A2 attend encore, B2 suit.

**[2026-09-26/27] B2 — un seul découpeur SQL (item 3) — exécutant opus, worktree du plan.**

- Ordre : tests rouges (B2.5) et garde-rail (B2.4) écrits d'abord sur le code d'avant, puis
  B2.1 → B2.6. Cache Go dédié `go-build-backlog`, une commande `go` à la fois.
- Sonde temporaire (retirée, log `B2-sonde-equivalence.log`, `EXIT_SONDE=0`) : l'ancien
  `sync.splitSQL` et `migration.SplitSQL` rendent les MÊMES instructions sur les quatre scripts
  que `sync` exécute (`playerSchemaSQL` 20, `sharedSchemaSQL` 12, `sharedViewsSQL` 1,
  `GamertagLookupViewSQL` 1). D-3 vérifiée sur pièces : aucun changement de découpage en prod.
- Sorties rouges sur le code d'avant :
  - `B2-5-rouge-sync.log`, `EXIT_ROUGE_SYNC=1` : `TestEnsurePlayerSchema_TrailingCommentOnly` →
    `EnsurePlayerSchema: DDL player échoué au boot même après réparation append-only (…):
    execScript: empty query (stmt="-- note finale")` ; `TestExecScript_TrailingCommentOnly` →
    `execScript: empty query (stmt="-- note finale")`.
  - `B2-4-rouge-archlint.log`, `EXIT_ROUGE_ARCHLINT=1` : 7 sites, les trois copies —
    `cmd/diag_exec/main.go:34` (Split « ; »), `sync/schema.go:522/532/537` (execScript, splitSQL,
    comparaison à `';'`), `sync/skill/sqlexec_helpers_test.go:15/24/26`. Aucun faux positif.
  - `TestExecScriptContext_HonoursContext` (nouveau symbole, donc non compilable sur le code
    d'avant) : prouvé par mutant — cœur remis sous `bootCtx()` → `err = <nil>, attendu
    context.Canceled` (log `B2-mutants.log`).
  - Mutation du garde-rail : une copie `strings.Split(script, ";")` + `db.Exec` réintroduite dans
    `cmd/diag_exec/main.go` → `cmd/diag_exec/main.go:96 strings.Split*(…, ";") dans un fichier qui
    exécute du SQL` (`EXIT_MUTANTS=1`). Les deux mutants retirés, grep `MUTANT B2` vide.
  - Après correctif : `B2-5-vert.log`, `EXIT_VERT=0`.
- Gates GO-F (logs `$TEMP\backlog-gates\B2-*.log`) :
  - `EXIT_BUILD=0`, `EXIT_VET=0`, `EXIT_TEST_LOT=0` (migration, sync, sync/skill, archlint,
    diag_exec).
  - `EXIT_INTEG_LOT=1` : seul échec `TestLUSRV2Shadow_RafalesBornees_300Candidats` (détention
    2,01-2,04 s > 2 s), PRÉEXISTANT (DB-8) : rouge aussi sur `d61443ef5` dans un worktree
    jetable (`EXIT_BASE_SKILL=1`), test indépendant du diff (`rw.Exec` direct). Rejoué seul :
    `EXIT_INTEG_SKILL_REJEU=0`.
  - `EXIT_TEST_COMPLET=1` : 187 paquets `ok`, 4 paquets rouges sous charge —
    `config` (`TestLoadPlayers_RacyWindowSnapshotNeverStored`), `mapcatalog` (deux tests de
    verrou concurrent, « passage forcé » à 2 s), `sync/haloclient`
    (`TestGetMatchFilm_ParallelDownloadFasterThanSequential`), `sync/skill` (DB-8). Rejoués
    seuls : `EXIT_TEST_REJEU=0` (les quatre verts). Aucun ne passe par le code du lot.
  - `EXIT_INTEG_COMPLET=1` : aucune ligne `^--- FAIL:` ; deux paquets tués au délai de 10 min
    (`platform/duckdb` 600,3 s, `sync` 600,3 s). Rejoués seuls avec `-timeout 40m` :
    `EXIT_INTEG_REJEU_SYNC=0` (576,5 s), `EXIT_INTEG_REJEU_DUCKDB=0` (872,3 s). Voir DB-10.
  - `EXIT_LINT=0` (`0 issues.`).
- Charge du poste : deux `replay-equiv` d'une autre session tournaient pendant les suites
  complètes (plus aucun à la fin).
- Écarts :
  - `sqlexec_helpers_test.go` SUPPRIMÉ plutôt que réécrit : ses deux appelants appellent
    `migration.ExecScriptContext` (un délégué de test n'aurait rien apporté).
  - Message d'erreur de `sync.execScript` : `(stmt=%q)` tronqué par `truncate` devient celui
    du cœur, `(stmt=%.80s)`. Aucun lecteur de ce texte (grep `stmt=`).
  - `cmd/diag_exec` hérite de la sémantique canonique : fragment de commentaires seul ignoré,
    `;` dans un `--` ne sépare plus.
  - Un test de plus que la liste (`TestExecScriptContext_HonoursContext`).
  - Les suites complètes ont dépassé les 10 min de l'outil : lancées au premier plan, basculées
    en tâche de fond par l'outil, attendues par une boucle au premier plan.

**[2026-09-27] Superviseur — vérification de B2.**

- Diff `edd0054d5` relu : `sync.execScript` délègue en une ligne à
  `migration.ExecScriptContext` ; `splitSQL`, `trimSpace` et `truncate` sont supprimés avec leurs
  tests ; le garde-rail AST couvre `internal/` et `cmd/`.
- Gates rejoués avec le cache dédié :
  - `EXIT_BUILD=0`, `EXIT_VET=0` ;
  - `EXIT_TEST=0` (migration, sync, sync/skill, archlint, diag_exec) ;
  - `EXIT_INTEG_SYNC=0` (filtre EnsurePlayerSchema, EnsureSharedSchema, PlayerSchemaAuthority,
    ExecScript) ;
  - `EXIT_INTEG_MIG=0` (migration, sync/skill : le test de rafales LUSR, DB-8, est passé cette
    fois) ;
  - `EXIT_LINT=0` (`0 issues.`).
- Les suites complètes restent celles de l'exécutant (échecs sous charge, verts rejoués seuls).
  La CI de branche sert d'autorité : branche poussée après cette vérification.
- Réponse de l'utilisateur sur la vitesse (1× ou 2× au plus) versée en A1.6.

**[2026-09-27] B3 — index ART de `match_skill_rank` (item 2) — exécutant opus, worktree du plan. STOP D-4 à B3.1.**

- Mesure : `migration/psa_index_repro_msr_planprobe_test.go` (`TestMSRIndexRemovalPlanProbe`,
  302 lignes) et `psa_index_repro_msr_fixture_test.go` (245 lignes), tag `psarepro`, inertes
  hors du tag (`go list` : `IgnoredGoFiles`).
  - DB fichier de 12 000 lignes, 5 053 matchs : les quatre chaînes de Halo Infinite
    pondérées, LUSR + LUSR_V2 par match social, CSR sur le classé, versions append-only
    réécrites, 22 matchs `h5_arena`.
  - Écriture par lots de 25 matchs en transaction, `CHECKPOINT` toutes les ~2 000 lignes,
    index posés avant le remplissage.
  - Copie du fichier, puis `DROP INDEX` des trois et `CHECKPOINT` sur la copie. Les deux
    bases sont rouvertes à froid.
  - Plan par `EXPLAIN ANALYZE`, 30 fois : `EXPLAIN` seul ne voit jamais l'index (DB-13).
  - Temps client : médiane et p90 de 30 exécutions (lignes consommées par `Scan`, sans
    autre travail), après 3 de chauffe. Temps moteur : médiane du « Total Time ».
  - Témoin C0 (PK) : Index Scan attendu des deux côtés, sinon `Fatal`.
  - Même résultat avec et sans index vérifié par une empreinte triée, calculée hors chrono.
- Défauts de méthode corrigés en cours de mesure :
  - le passage 1 chronométrait l'empreinte (`fmt.Sprint` et tri de chaque ligne) : écarté ;
  - le passage 2 détectait le plan par `EXPLAIN`, qui ne montre jamais l'index (le témoin
    C1 restait séquentiel alors que l'index était pris) : plan écarté, temps valides.
  - Passages valides pour les temps : 2, 3, 4 et final ; pour les plans : 3, 4 et final.
  - Logs `B3-1-mesure.log` et `B3-1-mesure-2.log`.
- Tableau, passage final (log `B3-1-mesure-final.log`, `EXIT_MESURE_FINAL=1` : le critère
  est asserté).
  - Colonnes : plan / médiane / p90 / moteur, avec index puis sans index.
  - Poste chargé à 100 % CPU (deux `replay-equiv-j45` d'une autre session).

  | Forme | Avec index | Sans index |
  |---|---|---|
  | F1 `Q24LUSRHistory` (12 000 l.) | SEQ 30/30 · 14,9 · 23,9 · 5,5 ms | SEQ 30/30 · **13,5** · 17,8 · 5,2 ms |
  | F2 citations `Q26gPlaylistPhaseAMSRTpl` IN(200) | SEQ 30/30 · 18,1 · 25,8 · 17,1 ms | SEQ 30/30 · **13,5** · 16,4 · 14,6 ms |
  | F3 escouade `QSquadExpectedWinProbTpl` IN(1000) | SEQ 30/30 · 18,4 · 20,2 · 23,6 ms | SEQ 30/30 · **16,7** · 29,0 · 22,6 ms |
  | F4 `loadExistingCSRMatchIDs` | **IDX 30/30** · 20,3 · 20,8 · 19,4 ms | SEQ 30/30 · 1,9 · 2,5 · 1,5 ms |
  | F5a `LoadExistingRatingIDs('LUSR')` | SEQ 30/30 · 4,0 · 5,5 · 2,3 ms | SEQ 30/30 · 2,4 · 5,7 · 0,9 ms |
  | F5b `loadExistingLUSRStates` | SEQ 30/30 · 4,6 · 6,1 · 5,0 ms | SEQ 30/30 · 3,9 · 4,5 · 3,4 ms |
  | F6a `loadPreviousLUSRRating` (arena_slayer) | SEQ 30/30 · 3,5 · 4,9 · 4,6 ms | SEQ 30/30 · 3,5 · 7,8 · 4,2 ms |
  | F6b `loadPreviousDisplayedOrdinal` (chaos) | SEQ 30/30 · 3,8 · 5,6 · 4,7 ms | SEQ 30/30 · 3,0 · 4,0 · 3,9 ms |
  | F7 `RunDualRowSentinel` | SEQ 30/30 · 5,8 · 9,8 · 4,3 ms | SEQ 30/30 · 4,8 · 6,8 · 3,2 ms |
  | S1-S3 invariants (hors critère) | SEQ · 2,9 à 3,6 ms | SEQ · 3,0 à 3,5 ms |
  | S4 vue `_latest` IN(200) (hors critère) | SEQ · 11,0 ms | SEQ · 10,0 ms |
  | C0 PK `id = 5` (témoin) | IDX 30/30 · 0,7 ms | IDX 30/30 · 0,6 ms |
  | C1/C2 `playlist_group = 'h5_arena'` (témoins) | **IDX 30/30** · 0,6 / 0,7 ms | SEQ 30/30 · 0,6 / 0,6 ms |

- Verdict : **critère D-4 NON TENU** → STOP, index conservés, B3.2 à B3.9 `[!]`.
  - F1, F2 et F3 dépassent 10 ms sans index dans les quatre passages valides (11,7 à 20,5 /
    12,1 à 18,5 / 13,9 à 19,1 ms), à la lettre du critère (temps client médian).
  - Ce dépassement ne vient PAS de l'absence d'index : même plan séquentiel avec et sans
    (30/30), temps équivalents. Le coût est la lecture de 12 000 lignes par le client Go (F1 :
    5 ms moteur) et le traitement des listes IN (F2, F3 : 15 à 24 ms moteur), avec ou sans
    index.
  - Mesure faite sous charge (100 % CPU). Sur un poste au repos, les valeurs baisseraient,
    sans garantie de passer sous 10 ms pour F3.
- Proposition au superviseur (décision hors de mon rôle, §2 « fermes ensuite ») : le critère
  absolu de 10 ms ne répond pas à la question que D-4 pose (« le retrait ralentit-il une
  lecture ? »). Critère relatif possible : sans index ≤ avec index + marge, forme par forme,
  et aucune forme qui passe de l'index au séquentiel avec une perte. Il est TENU sur les
  quatre passages avec une marge de 2 ms (plus grand écart « sans − avec » relevé : +1,1 ms,
  F1 au passage 2) ; F4 y gagne ~10×. Si D-4 est amendé, B3.2 reprend là, et l'assertion
  `CRITERE D-4` de la mesure se réécrit dans le même commit.
- Gates, sur l'état commité (deux fichiers de test sous tag, plan) :
  - `EXIT_BUILD=0`, `EXIT_VET=0`, `EXIT_VET_PSAREPRO=0` ;
  - `EXIT_LINT=0` (`0 issues.`) ;
  - gate psarepro `-run MSR` : `EXIT_MESURE_FINAL=1`, c'est le verdict D-4 et non une
    régression.
  - Suites `go test` complètes et d'intégration NON lancées : aucun code compilé hors du tag
    ne change, et le lot s'arrête avant tout code de production.
- Écarts :
  - fixture sortie dans un second fichier `psa_index_repro_msr_fixture_test.go` pour tenir le
    plafond de 500 lignes ;
  - formes supplémentaires (invariants, vue `_latest`) et témoins mesurés en plus des sept ;
  - F5 et F6 comptent chacune deux requêtes (a/b), toutes deux mesurées.

**[2026-09-27] B3 (reprise) — D-4 amendé par le superviseur (`99b241970`) : B3.1 au critère relatif, B3.2 à B3.8 faits, B3.9 bloqué.**

- B3.1 : l'assertion est réécrite (`msrProbeMargin` = 2 ms, sans index ≤ avec index + 2 ms,
  sur les sept formes) et l'en-tête renvoie à l'amendement. Commit `82cd8871b`.
  - Log `B3-1-mesure-relatif.log`, `EXIT_MESURE=0` ; plus grand écart « sans − avec »
    +0,22 ms (F3).
  - Gate 7 rejoué en fin de lot : `EXIT_PSAREPRO=0`. Le poste était moins chargé, et F4 avec
    index est tombé à 3,8 ms contre 1,8 ms sans. Plus grand écart +0,31 ms (F3).
- Rouge d'abord : tests B3.7, ratchet B3.6 et véhicule B3.9 écrits avant le code.
  - Sorties rouges sur le code d'avant (log `B3-7-rouge.log`) :
    - `EXIT_ROUGE_RATCHET=1` : 9 sites, six dans `steps_player_match_skill_rank.go` et trois
      dans `sync/schema.go`. La dispense de la baseline est consommée.
    - `EXIT_ROUGE_TITRE=1` : `3 idx_msr_* sur une player DB fraîchement migrée, attendu 0`,
      et `step drop_msr_secondary_art_indexes_v1 absent de la chaîne match_skill_rank`
      (×2).
    - `EXIT_ROUGE_SYNC=1` : `idx_msr_* présent après migrations + soin` (×3), et
      `… recréé par un binaire ancien et TOUJOURS présent après EnsurePlayerSchema` (×6 :
      trois MSR, trois PSA).
  - Véhicule B3.9 écrit après le code, donc son rouge est pris dans un worktree jetable à
    `82cd8871b` (retiré ensuite) : `EXIT_ROUGE_VEHICULE=1`, `index retirés encore présents
    après migrations + soin : [les six]`.
  - Après le code : `EXIT_VERT_MIGRATION=0`, `EXIT_VERT_TITRE=0`, `EXIT_VERT_SYNC=0`,
    `EXIT_VEHICULE=0` (logs `B3-7-vert.log`, `B3-9-vehicule-vert.log`).
- Mutation du ratchet (log `B3-6-mutation.log`) : un `CREATE INDEX … match_skill_rank` dans
  `sync/schema.go` et un dans la chaîne du titre donnent `EXIT_MUTANT=1`, les deux sites sont
  désignés. Fichiers restaurés depuis sauvegarde, grep `MUTANT` vide.
- Gates GO-F (logs `$TEMP\backlog-gates\B3-gate*.log`) :
  1. `EXIT_BUILD=0` ;
  2. `EXIT_VET=0` ;
  3. `EXIT_TEST_LOT=0` (migration, sync, halo_infinite/migrations, scheduler,
     purge_foreign_lusr_chain, archlint, platform/duckdb) ;
  4. `EXIT_INTEG_LOT=0` (mêmes paquets, `-p 1 -timeout 60m`) ;
  5. `EXIT_TEST_COMPLET=1` : 188 paquets `ok`, un seul rouge,
     `TestLUSRV2Shadow_RafalesBornees_300Candidats` (une rafale à 3,14 s). C'est DB-8,
     préexistant et rouge aussi sur `d61443ef5`. Rejoué seul : `EXIT_TEST_REJEU_SKILL=0` ;
  6. `EXIT_INTEG_COMPLET=1` : 189 paquets `ok`, aucun paquet tué au délai, un seul rouge, le
     même test DB-8 (rafales à 2,00-2,03 s). Rejoué seul : `EXIT_INTEG_REJEU_SKILL=0` ;
  7. `EXIT_PSAREPRO=0` ;
  8. `EXIT_LINT=0` (`0 issues.`).
- B3.9 : STOP sur la condition préalable, les quatre player DB `halo_infinite` ont un
  `.wal` (DB-17). Aucune copie faite, rien n'a été ouvert dans le checkout principal. Le
  véhicule est prêt et validé (cases B3).
- Écarts :
  - le step est title-owned (voir B3.2) ;
  - les noms des index retirés passent par une constante unique du paquet `migration`, que
    le step PSA consomme aussi ;
  - tests en plus de la liste : chaîne neuve sans index (titre), précondition du test de
    convergence, véhicule B3.9 et sa base synthétique ;
  - `sync/squash_convergence_test.go:210-214` : commentaire qui disait les index MSR
    « recréés par playerSchemaSQL » corrigé ;
  - `purge.go` : commentaires du scan forcé datés (index retiré, garde conservée).

**[2026-09-27] B3 — baseline de tests et B3.9 (décisions du superviseur).**

- Baseline (le run CI `36278286217` rougissait au job « Go Coverage + Baseline ») :
  - retrait de `.ai/baselines/tests_pre_migration.jsonl` des 12 tests de
    `internal/sync` supprimés par B2 : `TestSplitSQL_*` ×5, `TestTrimSpace_*` ×4,
    `TestTruncate_*` ×3 ;
  - 48 lignes (4 par test), 60 878 → 60 830 lignes, diff limité à ces 48 suppressions ;
  - les homonymes de `internal/migration` et `internal/notify` restent.
  - Balayage des 9 693 paires (paquet, test) de la baseline contre les `func Test…` du
    code : ces 12 sont les SEULES absentes.
  - Les tests supprimés par B3 sont postérieurs à la capture du 26/06 et absents de la
    baseline.
  - Preuve : `check_test_baseline.sh tests --from-jsonl` sur un run
    `go test -json -tags=integration -p 1` de 7 paquets (`EXIT_BASELINE_JSONL=0`), baseline
    restreinte à ces paquets dans un banc temporaire.
    - Avant retrait : `EXIT_BASELINE_AVANT=1`, « internal/sync : 12/850 absents ».
    - Après retrait : `EXIT_BASELINE_APRES=0`, « Tous les tests baseline présents ».
    - Logs `B3-baseline-check-avant.log` et `B3-baseline-check-apres.log`.
  - Commit dédié `e1e4232a2`.
- B3.9 : véhicule assoupli (WAL de la copie accepté, issue du rejeu journalisée) ; deux
  copies traitées, plus une copie de contrôle passée au code d'avant (détail dans la case
  B3.9).
  - Rejeu du WAL `halo_infinite` : réussi (DB-19).
  - Vérifications : `EXIT_SYNC_FINAL=0` (synthétique vert, réelle sautée sans variable),
    `EXIT_VET_SYNC=0`, `EXIT_LINT=0`.
- Écarts :
  - le passage `halo_5` rougit par construction : le véhicule exige les mêmes lignes
    partout, et la base avait 12 étapes de retard. Les deux écarts viennent d'étapes hors B3,
    et la copie de contrôle au code d'avant le prouve. Le véhicule n'a pas été relâché.
  - Une copie de plus que la consigne : `h5-avant`, identique à l'original, faite avec les
    mêmes contrôles, pour prouver l'attribution. Les copies restent sous
    `$env:TEMP\backlog-b3\` pour contre-vérification.

**[2026-09-27] Superviseur — vérification de B3 (mesure, retrait, baseline, B3.9).**

- Diffs relus :
  - `82cd8871b` : mesure au critère relatif ;
  - `7e9ef7c15` : retrait, soin convergent, ratchet, suppressions ;
  - `e1e4232a2` : baseline, exactement les 12 tests supprimés par B2, 48 lignes ;
  - `0b6c11f73` : B3.9.
- Gates rejoués avec le cache dédié : `EXIT_BUILD=0`, `EXIT_VET=0` (`./...`), tests unitaires du
  lot verts, `EXIT_INTEG=0` (migration, chaîne du titre, purge), `EXIT_INTEG_SYNC=0`,
  `EXIT_PSAREPRO=0`, `EXIT_LINT=0`.
- Un seul rouge : `internal/sync` tué au délai de 11 min, parce que le poste s'est mis en veille
  la nuit (5 h 48 de temps mur). Rejoué seul après le réveil : `ok` (103 s).
- DB-21 est apprécié sans nouvelle mesure. À 33 702 lignes au lieu de 12 000, les formes
  séquentielles coûtent environ 2,8 fois plus, mais pareil avec et sans index (même plan). Les
  seules formes qui empruntent un index sont :
  - F4, 10 fois plus lente AVEC l'index ;
  - C1/C2, à égalité à 12 000 lignes, un scan d'environ 1-2 ms attendu à 34 000.
  La marge de 2 ms n'est pas menacée, et l'argument de correction (index désynchronisé =
  lecture fausse) prime.

**[2026-09-27] B4 — garde-rail d'exclusion de la Campagne (item 4) — exécutant opus, worktree du plan.**

- Ordre : garde (B4.1) écrit d'abord et lancé sur le code d'avant (33 entrants, log
  `B4-1-premier-balayage.log`), cas de comportement écrits et rouges, puis corrections,
  dispenses, mutations, gates. Cache Go dédié `go-build-backlog`, une commande `go` à la fois.
- Garde : 5 racines, 5 588 déclarations balayées ; 83 lecteurs, 62 exclus, 21 dispensés
  (`platform/duckdb` 72/54, `progression` 4/4, `api/wire` 5/4, `service` 0/0, `analysis` 2/0).
- Entrants (lecteur → statut, catégorie, pièce dans la dispense) :
  - corrigés (B4.2) : `GetEncounterStats`, `LoadAxisSamples`, `ListRecentPvPMatchIDs`, `q31`
    (title-agnostic), `RecentMatches`, `CumulativeSince`, `candidateMatches`,
    `SquadUsualContexts`, `loadProgressionSharedMatches`, `loadPlayerStats` (3 requêtes),
    `loadComebackContext`, `SnapshotPlayerState` (3 requêtes) ;
  - mono-match : `Q17PlayerMatchStats`, `Q17bIsParticipant`, `Q26MatchExpectedStats`,
    `LoadMatchEngagementContext` ;
  - ensemble fourni par l'appelant : `Q25MatchParticipants`, `CountCommonMatchesForXUID`,
    `Q32bMainTeamParticipantsTemplate`, `LinkTargetsForMatches`, `matchsDesParticipants` ;
  - exclu au call site : `QRelationsPlayerWinRateTpl`, `Q25NeighborMatchesTemplate`,
    `BuildNeighborsWhereClause`, `playerMatchesSharedBaseSelect`,
    `Q42MapStatsSquadExtraExclusionFrag`, `clauseCoequipier` ;
  - sémantique : `countRankedMatchesInRegistry`, `LoadMatchCandidatesForMedia`,
    `lastMatchByXUID`, `AnnuaireNomsSQL`, `appendPlayerMatchSetFilters`,
    `buildRelationAssistsQuery`.
- Sorties rouges sur le code d'avant (Halo 5 ; Halo Infinite vert partout, résolveur neutre) :
  - `B4-2-rouge-duckdb.log`, `EXIT_ROUGE_DUCKDB=1` : `2 rencontres, attendu 1` ;
    `échantillons [5 50], attendu [5]` ; `[arena1 camp1], attendu [arena1]` ;
    `matchs communs avec B = 2, attendu 1`.
  - `B4-2-rouge-prestige.log`, `EXIT_ROUGE_PRESTIGE=1` : `matchs [camp1 arena1], attendu
    [arena1]` ; `cumul 55 sur 2 match(s), attendu 5 sur 1` ; `matchs d'escouade [arena1 camp1],
    attendu 1` ; `playlists usuelles [Arene Campagne], attendu 1`.
  - `B4-2-rouge-wire.log`, `EXIT_ROUGE_WIRE=1` : `matchs [arena1 camp1], attendu [arena1]` ;
    `matches_played / accuracy_threshold_days / combat_precision_matches = 2, attendu 1` ;
    dernier match `2026-09-02` au lieu du `2026-09-01` et match précédent présent ; `KD 55,
    attendu 5`, `taux de victoire 0.5, attendu 1`, `meilleur KDA sur "camp1"`, dernier match
    `2026-09-02`.
  - Après correctif : `B4-2-vert.log`, `EXIT_VERT=0`.
- Mutations du garde (logs `B4-1-mutation.log`, `B4-1-mutation-2.log`), toutes retirées
  (grep `MUTANT B4` vide) :
  - exclusion retirée de `GetEncounterStats` → `EXIT_MUTANT=1`,
    `compare_repo.go:CompareRepo.GetEncounterStats (ligne 213)` ;
  - exclusion retirée de `SquadUsualContexts` (forme `xuid IN`) → désigné (ligne 211), et une
    dispense fictive → `dispense périmée : …nexiste_pas.go:MUTANT` ; `EXIT_MUTANT_2=1`.
- Lecteurs de `sync/`, `ops/`, `cmd/` pour le backlog (sonde temporaire du garde sur ces
  racines, retirée ; log `B4-5-sonde-sync-ops-cmd.log`) — 56, rien corrigé :
  - `internal/sync` (17) : `assists_model.go:loadAssistsSamples`,
    `backfill.go:FindMatchesMissingParticipantBits`, `findMatchesInSharedAll`,
    `findMatchesInSharedDB`, `citations.go:loadMatchStats`, `comeback.go:loadMyTeamAndOutcome`,
    `csr_shared_backfill.go:loadRankedMatchesForSharedCSRBackfill`,
    `engagement.go:loadMatchesForEngagement`, `engine_backfills.go:loadAllMatchIDsForPlayer`,
    `enrichments.go:queryAnyBotMatchIDs`, `querySignificantBotMatchIDs`,
    `friends_recompute.go:loadMatchesWithFriends`, `performance_helpers.go:loadHistoryForPerf`,
    `session_recalc.go:sessionMatchesSQL`, `skill/skill_rating_loaders.go:loadLUSRMatchData`,
    `skill/skill_v2_shadow.go:loadShadowMatches`,
    `snapshot/snapshot_readiness_eval.go:loadSnapshotSharedFacts` ;
  - `internal/ops` (10) : `archive.go:ArchiveMatches`, `deleteArchivedParticipants`,
    `listArchivableYears`, `data_quality_examples.go:exampleMatchIDsFor`,
    `milestone_dates.go:loadCrossingMatches`, `seed_demo.go:selectRecentMatchIDs`,
    `sharedTablesWhere`, `seed_demo_corpus.go:applyUniversalAnonymization`,
    `selectRecentRankedMatchIDs`, `seed_demo_prestige.go:loadDemoCorpusStats` ;
  - `cmd` (29) : `audit_coverage:auditPlayer`, `backfill_all:loadMissingPSAMatches`,
    `backfill_kda_accuracy:main`, `backfill_participation_info:updateParticipationInfo`,
    `backfill_quit_timestamps:updateQuitTimestamps`, `backfill_time_played:commitUpdates`,
    `diag_backfill_dryrun:countMatchParticipants`, `countRankedForPlayer`,
    `diag_csr:diagUnrankedDates`, `diagUnrankedDatesXUID`, `diag_expected_kd:main`,
    `diag_highlight_match:main`, `diag_kpm:main`, `diag_lusr_player:loadMatches`,
    `diag_lusr_volatility:loadTrajectory`, `printPlayerCoverage`, `diag_orphan_session:main`,
    `diag_perfsim:universeSQL`, `diag_recent_match_sync:inspectMatch`,
    `diag_session_map_winrate:computeSquad`, `findSessionsOnDate`, `loadHistory`,
    `matchHasAllXUIDs`, `h5-roster-refetch:insertParticipant`,
    `h5-roster-topup:insertParticipant`, `levelup/cmd_backfill_h5_kill_mechanics:applyMechanicUpdates`,
    `lusr_v2_phase0:loadMatches`, `repair_data_consistency:chantier3BackfillXUIDAliases`,
    `seed-medal:main`.
  - Beaucoup sont des écritures ou des backfills (pas des lecteurs d'affichage) : le tri reste à
    faire (§1.4).
- Gates (logs `$TEMP\backlog-gates\B4-gate-*.log`) :
  - `EXIT_BUILD=0`, `EXIT_VET=0`, `EXIT_VET_INTEG=0` (lot, tag `integration`) ;
  - `EXIT_TEST_LOT=1` au premier passage : `TestMatchRegistryFixturesDeTestAlignees` (ma
    fixture Prestige déclarait `start_time_utc TIMESTAMP`) ; fixture passée en `TIMESTAMPTZ`,
    `EXIT_TEST_ARCHLINT=0`, cas Prestige rejoués verts (`EXIT_VERT_PRESTIGE=0`) ;
  - `EXIT_INTEG_LOT=1` au premier passage : 3 tests dont la fixture de shared n'avait pas de
    `match_registry` (`TestCountCrossTitleCooccurrences`,
    `TestCooccurrencesByXUID_PicksMaxTitle`, `…_SkipsTitleOnQueryError`) ; la table ajoutée
    aux deux fixtures (tout shared réel la porte), suite rejouée : `EXIT_INTEG_LOT_2=0` ;
  - `EXIT_TEST_COMPLET=0` (189 paquets `ok`, aucune ligne `^--- FAIL:`) ;
  - `EXIT_LINT=0` (`0 issues.`).
  - Aucun échec préexistant rencontré cette fois (DB-8 est passé).
- Écarts :
  - `loadPlayerStats` : 3 requêtes corrigées, pas 2 ;
  - `GetStatLeaderboard` : déjà exclu, rien fait ;
  - commentaires de `tactical_repo_univers.go` / `tactical_repo_ownership.go` qui décrivaient
    l'ancien garde (« ne balaye QUE les constantes Q ») réécrits au passé, avec renvoi au
    nouveau (doc inversée sinon) ;
  - deux fixtures de test existantes complétées d'une table `match_registry` ;
  - `candidateMatches` prend une struct `candidateQuery` (5 paramètres au plus) ;
  - le test `TestCampaignExclusionTokenWiredInStatQueries` (liste positive de 21 constantes)
    est conservé : le plan ne vise que le garde structurel et ses 6 dispenses ;
  - le rouge des cas Prestige a été pris avec `start_time_utc TIMESTAMP` ; la fixture est
    passée en `TIMESTAMPTZ` ensuite (type seul, lecteurs inchangés) ;
  - baseline de tests : aucune ligne à retirer, `TestCampaignExclusionStructuralCoverage` est
    absent de `.ai/baselines/tests_pre_migration.jsonl`.

**[2026-09-27] Superviseur — vérification de B4 et de A2, changement de fusion.**

- **B4** (`a9e2192ac`) :
  - diff échantillonné (rencontres, deltas notifiés, co-occurrence inter-titres) ;
  - gates rejoués : `EXIT_BUILD=0`, `EXIT_VET=0`, `EXIT_TEST=0` (archlint, platform/duckdb/...,
    api/wire, progression, service, analysis), `EXIT_INTEG=0` (filtre Campaign, CrossGame,
    Prestige, Relations sur les trois paquets de comportement) ;
  - lint local non conclusif : une autre session tenait golangci-lint (« parallel golangci-lint
    is running »), puis délai dépassé sous charge avec `--allow-parallel-runners`. Le lint de
    l'exécutant était vert ; la CI de branche fait foi.
- **A2** (`1fc63eca0`, `89f311191`) :
  - `useAudioUnlock` relu : `click`/`keyup` justifiés par la non-régression du 27/08,
    `hasBeenActive` pour le passage d'un rejeu à un autre ;
  - gates rejoués : `EXIT_TYPECHECK=0` (cache purgé), `EXIT_LINT=0`, vitest match-replay
    3 187/3 188, avec un seul échec, un délai de 5 s sur `carriedGlyphPulse.guard.test.ts` sous
    charge, rejoué seul : 3/3 verts.
- **Témoin de l'écoute A1.6** : `000d5950` ne joue aucune conclusion sur ce poste (vue match sans
  tableau des scores, DA-5). L'écoute se fait sur `ac03413d-2c03-4e3e-9e9c-c10165e40be0`, joueur
  Chocoboflor, victoire, fin dense, à 2× et à 1×. Le match où l'utilisateur a constaté le défaut
  est aussi à écouter, s'il le nomme.
- **Fusions** :
  - une seule branche, et B1-B4 ont été intercalés avant A2 (dérogation §1.2). La fusion 1
    (A1 + A2 seuls) n'est donc plus possible sans cherry-pick, lequel entrerait en conflit sur
    ce plan ;
  - décision : les fusions 1 et 2 sont regroupées. A1, A2 et B1 à B5 seront fusionnés ensemble,
    après la revue adversariale de B et les écoutes A1.6 / A2.5. La fusion 3 (A3, A4, A5) est
    inchangée ;
  - la v7.5 n'est pas fusionnée dans main à ce jour : le délai reste sans effet.
- Le serveur API tourne sur :8000 depuis 12 h 05 (checkout principal, accord de l'utilisateur,
  bases libres). B5.8 exige qu'il soit arrêté : c'est le superviseur qui le coordonne.

**[2026-09-27] B5.0 — ré-inventaire des tâches de fond et des chemins, avant tout code (exécutant opus, worktree du plan). Ce tableau fait foi pour B5.2-B5.5.**

Relevé sur `9d55d02b7`. Lignes de `cmd/server/main.go` sauf mention. « Réel » = sous `LEVELUP_REPO_ROOT`
(le vrai checkout sur le poste de dev et dans le harnais visuel). `<démo>` = `LEVELUP_DEMO_FIXTURES_DIR`.

| # | Tâche ou chemin | En démo avant B5 | Classement |
|---|---|---|---|
| 1 | Logs par module + crash log (`logging.LoadConfig`, l.203-240) | `<repo>/logs` (ou `logs/` du cwd) | redirigé `<démo>/runtime/logs` (B5.2) |
| 2 | Registre de titres et `config/titles/**` (l.378, 465), seeds Prestige et jalons (l.1692-1716), `static/`, `WebDistDir`, README, changelog | lecture | légitime |
| 3 | 4 bases warehouse (l.383-416) | fixture (`demoWarehouseDBPath`) | redirigé, déjà ; passe par le helper (B5.1) |
| 4 | Contrôle de la fixture joueur (l.428) | fixture | redirigé ; helper (B5.1) |
| 5 | `ensureWarehouseDir`, metadata pré-construite, titres additionnels, migrations player (l.447-516) | déjà gardés hors démo | légitime (garde existante) |
| 6 | `runMigrations` (l.466) | bases de la fixture | légitime (sous `<démo>`) |
| 7 | `users.json`, `groups.json` (l.695-703), `invites.json` (`api/server.go:694-697`) | `<repo>/data/auth` | redirigé `<démo>/auth` par `AuthDir` (B5.2) |
| 8 | Store d'amis `data/global/player_friends.json` (l.706, `api/server.go:691`) | réel, lu et écrit | redirigé `<démo>/runtime/…` (B5.5) |
| 9 | Tokens `data/auth/watcher_tokens` : `reauthStore` (l.714), `authStore` (`server_apiv1.go:1342`), SSO xbox (`server_apiv1.go:341`) | lecture réelle | redirigé `<démo>/auth/watcher_tokens`, vide (B5.2) |
| 10 | `app_settings.json` (l.730), `db_profiles.json` (`AdminPlayer`, l.2324 ; `config_players.go:167`), overlays de titre (`handlers/settings.go:154,309,317,507`) | réels sans variable explicite | redirigés vers la fixture (B5.2) |
| 11 | `migratePlayerFriendsAtBoot` (l.746) | écrit `player_friends.json` réel | coupé (B5.3) |
| 12 | `migrateDefaultGroupAtBoot` (l.750) | lit le vrai `db_profiles`, écrit `groups.json` | coupé (B5.3) |
| 13 | `buildAutoSyncPool` (l.763) | lit les tokens réels | coupé (pool nil), ce qui coupe aussi 19, l'orchestrateur V2 (l.1298) et 34 |
| 14 | Snapshot post-sync et journal des actions admin (l.783-789) | lus et écrits dans `data/global/admin_state` | redirigé `<démo>/runtime/…` (B5.5) |
| 15 | File persist asynchrone, ouvrier, `RecoverPending` (l.808-870) | WAL réel `data/wal`, rejeu dans les VRAIES player DB | coupé : `PersistBatchAsync` forcé à false (B5.3) |
| 16 | Janitor (l.876-915) : `data/sync_cache` + WAL | purge réelle au boot puis toutes les 24 h | coupé (B5.3) |
| 17 | Recovery WAL périodique (l.923) | — | coupé (suit 15) |
| 18 | `CHECKPOINT` périodique de `shared_social` (l.960) | base de la fixture | légitime |
| 19 | Re-scan du pool (l.988) | — | coupé (suit 13) |
| 20 | Watcher (l.1050) | lit `data/auth/watcher_tokens.json` réel | coupé (B5.3) |
| 21 | Scheduler d'auto-sync, `Run` (l.1066) | cycle + snapshot `admin_state` | coupé (l'objet reste construit pour le routeur) |
| 22 | Santé données (l.1077) | audit des bases réelles | coupé (B5.3) |
| 23 | Sauvegarde restic (l.1090) + `POST /settings/backup/run` | bases réelles ; `RequireAdmin` transparent en démo | refus en démo (B5.6) |
| 24 | Sessions : `MkdirAll` + purge au boot puis toutes les 6 h (`api/server.go:582-676`) | `data/sessions` réel | redirigé `<démo>/runtime/sessions` (B5.2) |
| 25 | `jobs.json` (`api/server.go:685`) | `data/cache/jobs.json` réel | redirigé `<démo>/runtime/…` (B5.5) |
| 26 | Cache de l'aide (`server_apiv1.go:325`) | écrit dans `data/cache` réel | redirigé `<démo>/runtime/…` (B5.5) |
| 27 | Cache d'assets (`server_apiv1.go:1383`) | lecture de `data/cache` | légitime, lecture seule (réseau coupé) |
| 28 | `monitoring.duckdb` (l.1161) : marqueur `server_boot`, puits des crons, flush des détections (l.1178), file de build | base réelle ouverte en ÉCRITURE | redirigé EN MÉMOIRE (B5.3) ; le flush reste actif sur la base mémoire |
| 29 | Surveillance disque (l.1190) | écrit `admin_state/disk_watch_state.json` réel | coupé (B5.3) |
| 30 | Notification des rejeux prêts (l.1199) | lit l'overlay de titre réel | coupé (B5.3) |
| 31 | Cron catalogue (l.1215) | réseau (coupé par netguard) puis écritures metadata | coupé (B5.3) |
| 32 | Balayage des noms d'assets (l.1247) | réseau | coupé (B5.3) |
| 33 | Cron classement mondial + enrichisseur (l.1395-1421) | `worldenrich.BuildEnricher` lit les tokens réels ; réseau | coupé (B5.3) |
| 34 | Cron Spartan (l.1346) ; `data/cache` à la main (l.1373) | — | coupé (suit 13) ; l.1373 corrigé (B5.7-4) |
| 35 | Purge des rejeux (l.1430) | `os.Remove` sur le vrai `data/cache/replays` d'après le vrai shared | coupé (B5.3) |
| 36 | `EmitAppReleaseForAllPlayers` (l.1469) | player DB de la fixture (résolveur démo) ; no-op en `dev` | légitime |
| 37 | `external.LogBootState` (l.1473) | lit `app_settings` de la fixture | légitime |
| 38 | Heartbeat (l.1508) | log seul | légitime |
| 39 | Écrivain de la file de build (`wire/registry_build_queue.go:311-334`) | vrai shared | redirigé vers la disposition démo (B5.4) |
| 40 | Rejeux et rasters (`registry_pages.go:159,189`), faits de film (`registry_monitoring_resources.go:117` → `FilmFactsDir`) | lecture de `data/cache` réel | redirigé `<démo>/runtime/…`, vide (B5.5) |

Hors du tableau : les écritures DÉCLENCHÉES PAR REQUÊTE hors `/settings/backup/run` (actions admin
ouvertes en démo par `RequireAdmin`) restent hors périmètre, versées aux découvertes (DB-28).

**[2026-09-27] B5 — mode démo hermétique côté fichiers (item 8) — exécutant opus, worktree du plan. B5.0 à B5.7 faits, B5.8 non jouée.**

- Ordre : tableau B5.0 au journal avant tout code (ci-dessus), puis garde-rails et tests rouges
  d'abord, sur le code d'avant ou sur un échafaudage sans comportement (méthodes aux sémantiques
  d'avant). Cache Go dédié `go-build-backlog`, une commande `go` à la fois.
- B5.1 : `title.DemoLayout` (`internal/domain/title/demo_layout.go`) remplace les trois copies ;
  le seed de la démo (`ops/seed_demo*.go`) est migré sur la disposition. Garde-rail
  `archlint/no_demo_layout_translation_test.go` (AST, `internal/` + `cmd/`) : noms de traducteurs,
  `filepath.Join` d'une base « demo/fixture » avec un segment de la disposition, tout segment de
  la disposition dans les fichiers de l'arbre démo ; base littérale (chemin relatif) exclue.
- B5.2 : défauts démo dans `config.Load` via `statePaths` (`config_demo.go`), une variable
  explicite garde la main ; `cfg.RuntimePaths()`, `cfg.WatcherTokensDir()`,
  `cfg.TitleSettingsPath()` rendent exactement les chemins d'avant hors démo. Logs : `bootLogsConfig`
  + `config.DemoLogsDir()` (lu avant `config.Load`).
- B5.3 : `cmd/server/background_tasks.go` porte le lancement (janitor, recovery WAL, checkpoint,
  re-scan du pool, heartbeat déplacés de `main.go`, qui passe de 2 366 à 2 211 lignes) et la table
  `demoPolicies` ; `bootTasks.cutInDemo` pour les étapes de boot (migrations amis et groupe, pool de
  tokens, file persist, watcher) ; un seul log Info `demo_mode: tâches de fond coupées` en fin de boot.
- Sorties rouges (logs `$TEMP\backlog-gates\B5-*.log`) :
  - garde B5.1, `B5-1-rouge-garde.log`, `EXIT_ROUGE_GARDE=1` : 29 sites (les 3 traducteurs, `main.go:428`,
    `config_players.go:120`, `player_resolver.go` ×9, seed ×14) ;
  - (4) `B5-7-4-rouge-datapath.log`, `EXIT_ROUGE_DATAPATH=1` :
    `cmd/server/main.go:1373  cacheRoot := filepath.Join(pr.RepoRoot(), "data", "cache")` ;
  - (1) `B5-7-1-rouge-config.log`, `EXIT_ROUGE_CONFIG=1` : 18 chemins « sous le dépôt LEURRE en mode
    démo » (SessionDir, AuthDir, UsersFilePath, DBProfilesPath, AppSettingsPath, Backup.BackupDir,
    WatcherTokensDir, TitleSettingsPath ×2, RuntimePaths ×7…), logs non redirigés, et
    `PersistBatchAsync = true en démo` ;
  - (3) worktree jetable à `9d55d02b7` + fichiers d'échafaudage seuls (retiré ensuite),
    `B5-7-3-rouge-demopaths-9d55d02b7.log`, `EXIT_ROUGE_DEMOPATHS_AVANT=1` : 21 chemins « sous le
    DÉPÔT en mode démo » (logs, cfg, runtime ×11) ; les 4 bases et la fixture joueur étaient déjà
    redirigées ;
  - (2) extraction sans gardes, `B5-7-2-rouge-taches.log`, `EXIT_ROUGE_TACHES=1` : `CRÉÉ
    data/global/monitoring.duckdb`, `CRÉÉ data/global/player_friends.json`, `SUPPRIMÉ
    data/sync_cache/sync.RunDelta_vieux` (+ son fichier), 7 tâches coupées LANCÉES ;
  - B5.6 `B5-6-rouge-backup.log`, `EXIT_ROUGE_BACKUP=1` : `statut 200, attendu 403`, `un cycle de
    sauvegarde a été lancé (1 énumération(s) de cibles)`.
  - Après correctif : `EXIT_VERT_CONFIG=0`, `EXIT_VERT_DEMOPATHS=0`, `EXIT_VERT_TACHES=0`,
    `EXIT_VERT_BACKUP=0`.
- Mutations (log `B5-mutations.log`), retirées ensuite (grep `MUTANT B5` vide) : une traduction
  `filepath.Join(cfg.DemoFixturesDir, "warehouse", …)` dans `demo_paths.go` → garde B5.1 rouge ;
  `data/cache` à la main dans `main.go` → `TestNoNewDataPathJoin` rouge ; janitor déclaré gardé →
  `SUPPRIMÉ data/sync_cache/…` et `statut démo 0, attendu 1` ; `EXIT_MUTANT_ARCHLINT=1`,
  `EXIT_MUTANT_TACHES=1`.
- Gates GO-F (logs `$TEMP\backlog-gates\B5-gate*.log`) :
  1. `EXIT_BUILD=0` ; 2. `EXIT_VET=0` ;
  3. `EXIT_TEST_LOT=1` au premier passage : deux garde-rails d'archlint relevaient MON diff
     (`TestNoNewFrenchLabelLiteral` : message FR du refus de sauvegarde → passé en anglais technique ;
     `TestNoGlobalFriendGamertagsKey` : clé legacy dans mon test → retirée, remplacée par une
     assertion sur la liste des coupées). Rejeu des 3 paquets touchés : `EXIT_TEST_LOT_REJEU=0` ; les
     11 autres paquets du lot étaient `ok` ;
  4. `EXIT_INTEG_LOT=0` (14 paquets, `-p 1`) ;
  5. `EXIT_TEST_COMPLET=1` : 186 `ok`, 3 rouges — `platform/duckdb`
     `TestNoUnauthorizedSharedSocialMention` (DÛ AU LOT : `background_tasks.go` et `demo_layout.go`
     mentionnent `shared_social` ; deux entrées datées ajoutées à la liste blanche), `mapcatalog`
     `TestAddOverlayEntryConcurrentNePerdPasDEntree` (« Accès refusé » au rename sous charge,
     préexistant, cf. B2) et `sync/skill` DB-8 (préexistant). Rejoués seuls : `EXIT_TEST_REJEU_DUCKDB=0`,
     `EXIT_TEST_REJEU_MAPCATALOG=0`, `EXIT_TEST_REJEU_SKILL=0` ;
  6. `EXIT_INTEG_COMPLET=0` (190 `ok`, aucune ligne `^--- FAIL:`, lancé après la liste blanche) ;
  7. `EXIT_LINT=2` au premier passage (goconst sur `"true"` déplacé dans `config_demo.go`, unparam
     sur le slug de `resolveBootDBPaths`) → corrigés ; `EXIT_BUILD_FINAL=0`, `EXIT_VET_FINAL=0`,
     `EXIT_TEST_FINAL=0` (cmd/server, config), `EXIT_INTEG_FINAL=0` (cmd/server, config, archlint),
     `EXIT_LINT_2=0` (`0 issues.`).
- B5.8 : NON JOUÉE. À l'arrivée sur l'item, `http://127.0.0.1:8000/health` répondait 200 (serveur du
  checkout principal). Rien n'a été arrêté ; le superviseur la fait jouer.
- Baseline de tests : aucun test supprimé présent dans `.ai/baselines/tests_pre_migration.jsonl`
  (les deux `TestDemoWarehouseDBPath_*` renommés en sont absents).
- Écarts :
  - `sessionComputeOptionsFor` prend désormais une fonction de chemin d'overlay (5 appels de test
    adaptés) ; les overlays de `handlers/settings.go:507/551` suivent donc aussi la fixture ;
  - hors du périmètre cité mais nécessaires : liste blanche `shared_social` de
    `platform/duckdb/no_attach_on_social_test.go` (2 entrées datées) ; `docs/CONFIGURATION.md` et
    `docs/FR/CONFIGURATION.md` (section « Chemins du mode démo », règle n°15) ; commentaire de
    `wire/prestige_setup.go:66` qui citait `ops.demoTitleSubdir` ;
  - tests en plus de la liste : `title/demo_layout_test.go`, variables explicites (config),
    déclaration des statuts (`TestBackgroundTasks_DeclarationDemo`), hors démo tout lancé,
    sauvegarde hors démo lancée ;
  - la migration des amis n'est plus rendue sensible dans le test manifeste (garde-rail de la clé
    legacy) : son sort est asserté par la liste des coupées.

**[2026-09-27] B-C — corrections de la revue adversariale, ronde 1 (exécutant opus, worktree du plan). B-C1 à B-C8 faits.**

- Méthode : un test par item, rouge sur le code d'avant (ou sur un échafaudage sans comportement :
  paramètre démo de `NewTitleSyncHandler` stocké non lu, `replayServiceFrom` à la sémantique d'avant),
  puis correctif, puis vert. Cache Go dédié `go-build-backlog`, une commande `go` à la fois. Logs sous
  `$TEMP\backlog-gates\BC-*.log`.
- Sorties rouges :
  - B-C1 `BC-1-rouge.log`, `EXIT_ROUGE_BC1=1` : `statut 201, attendu 403` (création de profil, l'annuaire
    a reçu la demande) ; `statut 200, attendu 403` + dossier du joueur effacé + entrée halo_5 retirée
    (purge) ; `statut 200, attendu 403` + `app_settings.json` écrit + abonnements du watcher modifiés.
    Témoins hors démo verts. Garde-rail `BC-1-rouge-garde.log`, `EXIT_ROUGE_GARDE_BC1=1` :
    `internal/api/handlers/settings_backup.go:29` (littéral de B5.6).
  - B-C2 `BC-2-rouge.log`, `EXIT_ROUGE_BC2=1` : `le dossier d'un joueur réel du dépôt apparaît dans les
    identités (VraiJoueur)`.
  - B-C3 : pas de rouge possible, il n'y avait rien à corriger (verdict réseau ci-dessous).
  - B-C4 `BC-4-rouge.log`, `EXIT_ROUGE_BC4=1` : `config.go fait 640 lignes, gel à 629`.
  - B-C5 `BC-5-rouge.log`, `EXIT_ROUGE_BC5=1` : `retrait de idx_msr_playlist : 0 ligne(s)
    schema_drift_healed action=dropped, attendu 1` (idem `idx_psa_match`), `WARN capturés = []`.
  - B-C6 `BC-6-rouge.log`, `EXIT_ROUGE_BC6=1` : `DemoLayout.Root() existe`.
  - B-C7 `BC-7-rouge.log` puis `BC-7-rouge-final.log` (version finale du test, construction d'avant
    remise le temps du passage), `EXIT_ROUGE_BC7_FINAL=1` : `démo : fond de carte versionné, statut
    404, attendu 200 — map_background_not_available`.
  - B-C8 `BC-8-rouge.log`, `EXIT_ROUGE_BC8=1` : `halo_5 : playlists habituelles [Arene Campagne],
    attendu 1`.
- Verdict réseau B-C3 : PAS de trou. `fetchGameCMSImage` → `doGet` → `netguard.Check(ctx,
  "gamecms_assets.get")` avant `httpClient.Do` ; GameCMS est la seule source d'un payload binaire, donc
  aucun `PersistBinary` en démo. Log `BC-3-verif-reseau-2.log` : `demo mode: external fetch skipped
  surface=gamecms_assets.get`, 0 requête, cache vide, `EXIT_VERIF_RESEAU_BC3=0`. Aucun code modifié.
- Mutations (retirées ensuite, `grep MUTANT` sans résultat du lot) :
  - B-C1 `BC-1-mutant.log`, `EXIT_MUTANT_BC1=1` : littéral ajouté dans `setup.go` → garde-rail rouge
    (`setup.go:462`). Une première tentative par chemin relatif .NET a échoué sans rien écrire
    (résolue contre le checkout principal, dossier inexistant) ; rejouée en chemin absolu.
  - B-C3 `BC-3-mutant.log`, `EXIT_MUTANT_BC3=1` : garde `netguard.Check` retiré de `doGet` → le test
    du lot rougit (1 requête, image écrite dans le cache) ET `TestOutboundCallsAreNetguarded` désigne
    `assets/fetcher_gamecms.go`.
- Gates GO-F (logs `BC-gate*.log`) :
  1. `EXIT_BUILD=0` ; 2. `EXIT_VET=0` ;
  3. `EXIT_TEST_LOT=1` au premier passage : `TestNoNewDataPathJoin`, DÛ AU LOT (B-C4 a déplacé le
     défaut `<repo>/data/demo` de `config.go` vers `config_demo.go`) → entrée datée ajoutée à
     l'allowlist (même site de bootstrap, pas un nouveau), `EXIT_TEST_ARCHLINT_REJEU=0` ; les 13 autres
     paquets étaient `ok`. `EXIT_INTEG_LOT=0` (14 paquets, `-p 1`) ;
  4. `EXIT_TEST_COMPLET=1` : 187 `ok`, 2 rouges préexistants et fragiles sous charge : `mapcatalog`
     (`Accès refusé` au rename, cf. B2/B5) et `sync/skill` DB-8 (rafale à 2,03 s). Rejoués seuls :
     `EXIT_TEST_REJEU_MAPCATALOG=0`, `EXIT_TEST_REJEU_SKILL=0` ;
  5. `EXIT_INTEG_COMPLET=0` (190 `ok`, aucune ligne `^--- FAIL:`) ;
  6. `make go-api-lint` : `0 issues.`, `EXIT_LINT=0`.
- Baseline de tests : aucun test supprimé.
- Écarts :
  - B-C1 : helper `handlers.refuseInDemo` + garde-rail `archlint/no_demo_forbidden_literal_test.go`
    (4e copie du refus, règle n°6) ; `settings_backup.go` (B5.6) migré dessus. Contrat OpenAPI tenu
    comme B5.6 : aucune déclaration (réponse `default` ApiError), `openapi.yaml` inchangé.
    `NewTitleSyncHandler` prend le mode démo au constructeur (2 appels de test adaptés).
  - B-C2 : branche « collecte coupée » retenue, pas un `PathFS` sur la racine démo (la fixture nomme
    ses dossiers autrement que ses clés de profil, cf. DB-39).
  - B-C3 : test de non-régression vert sur le code d'avant (rien à corriger) ; son pouvoir est prouvé
    par mutation, pas par un rouge d'avant.
  - B-C4 : gel de taille par test (`config/config_size_ratchet_test.go`) ; allowlist datée de
    `config_demo.go` dans `no_data_path_join_test.go` (hors liste du lot, relevée au gate).
  - B-C6 : test anti-résurrection par réflexion (`domain/title/demo_layout_no_root_test.go`).
  - B-C7 : construction du rejeu sortie de `registry_pages.go` (632 → 620 L, dette gelée) vers
    `wire/registry_replay_service.go` ; `service.NewReplayServiceRoots` ajouté, `NewReplayService`
    délègue avec deux racines égales (CLI et ~70 appels de test inchangés). Aucun fichier du §3.1.6
    touché. Le test HTTP ajoute un leurre (artefact du dépôt → 404 en démo) ; premier essai du leurre
    faussé par `FilmShortMatchID` (8 caractères : `match-demo` et `match-depot` = même fichier),
    identifiants corrigés.
  - B-C8 : le correctif vit dans `LazyPrestigeService` (seul point qui connaît le PlayerDB) ; le
    paramètre `title_slug` de la requête devient inerte (DB-37).

**[2026-09-27] B-C9 — `seed-demo` sous `LEVELUP_DEMO_MODE=true` (item ajouté par le superviseur sur la CI de branche rouge, run `36328284365`, commit `3de419efe`, job « Go Coverage + Baseline »).**

- Cause : la CI lance toute la suite Go avec `LEVELUP_DEMO_MODE=true` (`.github/workflows/ci.yml:436`).
  `TestSeedDemoCLI_E2E` lance le binaire avec `os.Environ()`, donc en démo ; depuis B5.2, `config.Load`
  y redirige `DBProfilesPath` vers `<repo>/data/demo/db_profiles.json`.
- Rouge (`BC-9-rouge.log`, `EXIT_ROUGE_BC9=1`) : sous-test `LEVELUP_DEMO_MODE_true` → `erreur: seed-demo:
  titres pour "JGtm": read profiles: open …\data\demo\db_profiles.json: Le chemin d'accès spécifié est
  introuvable.` (l'erreur de la CI) ; sous-test `sans_LEVELUP_DEMO_MODE` vert. Le test retire la
  variable de l'environnement hérité puis la pose selon le cas : il ne dépend plus du poste.
- Correctif : `config.LoadForCLI` (`config_demo.go`), appelé par `cmd/levelup/main.go`. `DemoMode`
  reflète toujours l'environnement (sémantique d'avant B5), mais aucune redirection d'état de B5.2/B5.5
  ne s'applique : profils, réglages et tout ce qui en dérive (webhook, fuseau, saison CSR, base médias,
  Prestige), auth, sessions, sauvegarde, file persist non coupée, `RuntimePaths`, `WatcherTokensDir`,
  `TitleSettingsPath`. `Load` (serveur) est inchangé. `config.go` reste à 629 lignes.
- Vert : `EXIT_VERT_BC9=0` (deux sous-tests), `EXIT_TEST_CONFIG_BC9=0` (`config_cli_test.go` : en démo,
  chemins d'état de `LoadForCLI` identiques à hors démo et jamais sous la racine démo ; témoin : `Load`
  redirige toujours).
- Sous-commandes de `cmd/levelup` statuées. `LoadForCLI` vaut pour TOUTES : ce sont des outils
  opérateurs sur les données réelles, aucune n'est un consommateur de la démo (le seul consommateur
  est le serveur démo) :
  - `seed-demo` (`DBProfilesPath`) : productrice de la démo, doit lire les vrais profils → corrigée ;
    `--synthetic` ne lit aucun profil (harnais visuel, CI e2e) → inchangée ;
  - `index-media` (`MediaCapturesBaseDir`, `UserTimezone`, tirés d'`app_settings`) : indexe les captures
    réelles avant `seed-demo` (regen de déploiement) → dépôt ;
  - `notify-version`, `notify-sync` (`AppSettingsPath`) : notifications réelles → dépôt ;
  - `check-env`, `gate-check` (`DBProfilesPath`, `AppSettingsPath`) : diagnostic du poste → dépôt ;
  - `add-title` (écrit `db_profiles.json`) : opérateur → dépôt ;
  - `identity list` / `identity purge` (`UsersFilePath`, `AuthDir`, `DBProfilesPath`) : annuaire réel
    (ADR 0035) → dépôt ;
  - `backfill-csr` et le moteur de pool (`CurrentCSRSeasonID`) ; `sync-*`, `backfill*`,
    `engagement-coefs`, `rebuild-pme-art`, `recompute-friends`, `backfill-squad-creators`,
    `sync-achievements` (`LoadPlayers` → `DBProfilesPath`) : opérateurs sur les vraies bases → dépôt.
    Leurs bases warehouse et player restent résolues selon `DemoMode`, comme avant B5 (DB-41) ;
  - `backfill-h5-kill-mechanics`, moteur de pool (`pr.WatcherTokensDir()`) : `PathResolver` du dépôt,
    non concernés par B5.2.
- Gates (logs `BC9-gate-*.log`) : `EXIT_BUILD_BC9=0`, `EXIT_VET_BC9=0`, `EXIT_TEST_BC9=0` (config, ops,
  cmd/levelup, archlint), `EXIT_INTEG_BC9=0` (`./internal/ops/ ./cmd/levelup/...` sans la variable),
  `EXIT_INTEG_DEMO_BC9=0` (même commande avec `LEVELUP_DEMO_MODE=true`, reproduction de la CI),
  `make go-api-lint` : `0 issues.`, `EXIT_LINT_BC9=0`.
- Baseline : `TestSeedDemoCLI_E2E` garde son nom (présent dans `tests_pre_migration.jsonl`), les cas
  sont des sous-tests ; aucun test supprimé.
- Découvertes : DB-40 (autres binaires `cmd/*` sur `config.Load`), DB-41 (effets de `DemoMode` sur les
  bases dans la CLI).

**[2026-09-27] B-C10 — la redirection démo des chemins d'état appartient au seul serveur (item ajouté par le superviseur ; inversion du choix de B-C9, d'après DB-40).**

- Décision : `config.Load()` retrouve la sémantique d'avant B5 pour les chemins d'état (profils,
  réglages et tout ce qui en dérive, auth, sessions, sauvegarde, caches d'exécution, file persist
  asynchrone non coupée) ; `DemoMode` reste lu de l'environnement, et les bases (fixture) se comportent
  comme avant B5 (DB-41, inchangé). `config.LoadServer()` applique les redirections de B5.2 et B5.5 et
  la coupure de la file persist. `config.LoadForCLI()` est supprimé sans alias. Mécanique :
  `load(fromRepo)` dans `config.go` (toujours 629 lignes) ; `Load` pose `stateFromRepo`, que
  `demoState()` lit pour `RuntimePaths` / `WatcherTokensDir` / `TitleSettingsPath`. Une `AppConfig`
  construite à la main avec `DemoMode` (tests) suit la disposition, comme le serveur.
- Inventaire des appelants de `config.Load()` (hors tests ; `grep` sur `apps/go-api`). Aucun appelant
  sous `internal/`. `config.DemoLogsDir()` (redirection des logs) n'est lu que par `cmd/server`
  (`demo_paths.go:66`).

  | Appelant | Processus | Statut |
  |---|---|---|
  | `cmd/server/main.go:311` | serveur API | → `LoadServer()` ; tout ce qui tourne dans le serveur reçoit ce `cfg` par injection (aucun autre `config.Load` dans le processus) |
  | `cmd/levelup/main.go:64` | CLI opérateur | `Load()` (retour d'avant B-C9) |
  | `cmd/backfill-team-rounds`, `backfill-team-scores`, `backfill-world-player-stats`, `backfill_kda_accuracy`, `backfill_objective_stats` | outils de rattrapage | `Load()`, données réelles |
  | `cmd/backup-once`, `restore`, `restore_one_player`, `cleanup_media_index`, `purge_player_media` | outils d'exploitation | `Load()`, données réelles |
  | `cmd/diag_appearance`, `diag_emblem_colors`, `diag_emblem_mapping`, `diag_live_economy`, `diag_matchstats_dump`, `probe-h5`, `probe-mcc` | diagnostics | `Load()`, données réelles |
  | `cmd/get-token`, `token-capture`, `token-import` | auth opérateur | `Load()` : magasin de tokens RÉEL (DB-40) |
  | `cmd/h5-appearance-backfill`, `h5-backfill`, `h5-csr-backfill`, `h5-csr-match-backfill`, `h5-enrich`, `h5-events-backfill`, `h5-kill-kind-backfill`, `h5-lusr-backfill`, `h5-metadata-fetch`, `h5-roster-refetch`, `h5-roster-topup`, `h5-sync`, `h5-teamscore-backfill` | outils Halo 5 | `Load()`, données réelles |
  | `cmd/mapobj-build`, `migrate-static-maps`, `populate-career-rank-images`, `prestige-tuning-analyze`, `refresh-career-ranks`, `refresh-metadata`, `refresh_golden_fixture`, `seed-rank-translations`, `world-aliases-persist` | outils de référentiels | `Load()`, données réelles |

  Tests appelants : `internal/config/*_test.go` (`Load` pour les tests hors démo et le nouveau test
  d'état, `LoadServer` pour l'hermétisme) et `cmd/server/demo_paths_test.go` (`LoadServer`).
- Tests :
  - rouge (`BC-10-rouge.log`, `EXIT_ROUGE_BC10=1`, échafaudage `LoadServer` = sémantique d'alors) :
    `TestLoad_DemoMode_CheminsDEtatDuDepot` → `SessionDir … \002\runtime\sessions en démo, attendu
    … \001\data\sessions`, idem `AuthDir`, `DBProfilesPath`, `AdminStateDir`, `FilmFacts`,
    `TitleSettingsPath`… ; vert après inversion (`EXIT_VERT_BC10=0`, config + cmd/server) ;
  - hermétisme sur `LoadServer` : `config_demo_hermetic_test.go` (tests renommés `TestLoadServer_*`,
    absents de la baseline) et `cmd/server/demo_paths_test.go` (`chargerCfg`, qui sert aussi le
    manifeste des tâches de fond) ;
  - mutation `LoadServer` = `Load` (aucune redirection), logs `BC-10-mutant.log` puis
    `BC-10-mutant-2.log`, `EXIT_MUTANT_BC10=1` : `TestLoadServer_DemoMode_AucunCheminDExecutionSousLeLeurre`,
    `TestLoadServer_DemoMode_Redirige`, `TestDemoBootPaths_TousSousLaRacineDemo` rouges. Au premier
    passage, le manifeste restait VERT : il n'exerçait que les coupures de B5.3, décidées par `DemoMode`.
    Il est renforcé des écritures d'exécution du serveur démo (répertoire et purge des sessions au
    boot, `jobs.json`) ; au second passage il rougit (`CRÉÉ data/cache/jobs.json`, `CRÉÉ
    data/sessions`). Mutant retiré (`grep MUTANT` vide dans `config_demo.go`) ;
  - E2E `TestSeedDemoCLI_E2E` : deux sous-tests verts dans les deux passages d'intégration.
- Gates (logs `BC10-gate-*.log`) : `EXIT_BUILD_BC10=0`, `EXIT_VET_BC10=0` ; unitaires (config,
  cmd/server, cmd/levelup, ops, archlint) `EXIT_TEST_BC10=0` et, sous `LEVELUP_DEMO_MODE=true`,
  `EXIT_TEST_DEMO_BC10=0` ; intégration des mêmes paquets (hors archlint) `EXIT_INTEG_BC10=0` et, sous la
  variable, `EXIT_INTEG_DEMO_BC10=0` ; suite unitaire complète sous `LEVELUP_DEMO_MODE=true` :
  `EXIT_TEST_COMPLET_DEMO_BC10=1`, 188 `ok`, un seul rouge préexistant, `sync/skill` DB-8 (rafale
  2,04 s), rejoué seul sous la variable : `EXIT_TEST_REJEU_SKILL_BC10=0` ; `make go-api-lint` :
  `0 issues.`, `EXIT_LINT_BC10=0`.
- Écarts : docs `CONFIGURATION.md` et `docs/FR/CONFIGURATION.md` (règle n°15) précisent que la
  redirection ne vaut que pour le serveur API ; manifeste des tâches de fond renforcé (ci-dessus) ;
  `config_cli_test.go` renommé `config_load_state_test.go`. Aucun test supprimé présent dans la
  baseline.
