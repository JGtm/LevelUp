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
  - Retrait des trois index si aucune forme de lecture ne dépasse **10 ms sans index** sur une
    DB fichier de 12 000 lignes. Sinon, arrêt et rapport.
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
- [!] **A1.5 Recette navigateur**, seulement si `http://127.0.0.1:8000/health` répond (le
  superviseur démarre le serveur, jamais l'exécutant). Sinon, l'item reste ouvert au rapport et le
  superviseur le fait avant la fusion 1.
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

- [ ] **A2.0 Reproduction AVANT tout code**, avec la politique d'autoplay PAR DÉFAUT (jamais
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
- [ ] **A2.1 Correctif**, selon la cause mesurée et dans le cadre de D-10 :
  - ouverture au premier geste sur la page (écouteur unique `pointerdown` / `keydown` au niveau du
    document, posé seulement si la préférence est « activé » et qu'aucun lecteur n'existe,
    retiré ensuite) ;
  - ouverture dès l'affichage si `navigator.userActivation?.hasBeenActive` ;
  - téléchargement anticipé des sons (décodage à l'ouverture) seulement si A2.0 mesure un délai
    supérieur à 2 s.
- [ ] **A2.2** `useReplaySound.ts` ne grossit pas : la logique d'ouverture vit dans un module
  dédié (par exemple `sound/useAudioUnlock.ts`). L'en-tête « DEUX ÉTATS » est mis à jour avec la
  nouvelle liste des gestes.
- [ ] **A2.3 Tests** (rouges d'abord, jsdom) :
  - préférence « activé », aucun lecteur : un `pointerdown` sur le document ouvre le lecteur une
    seule fois, et l'écouteur est retiré ;
  - `userActivation.hasBeenActive` vrai à l'affichage : le lecteur s'ouvre sans geste ;
  - préférence « coupé » : rien ne s'ouvre, rien ne se télécharge ;
  - non-régression du 27/08 : le premier clic sur le bouton du son ouvre et ne coupe pas.
- [ ] **A2.4** Recette A2.0 rejouée après correctif : pour chaque chemin et dans les deux
  navigateurs, son audible au plus tard 2 s après le geste. Sortie au journal.
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

- [ ] **B2.1** `migration.ExecScriptContext(ctx, db, script)`, cœur unique ; `ExecScript`
  délègue.
- [ ] **B2.2** `sync.execScript` devient un délégué d'une ligne. Suppression de `sync.splitSQL`,
  de `trimSpace`, de `truncate` (s'il n'a plus d'appelant, à vérifier) et de leurs tests
  unitaires (`schema_unit_test.go`).
- [ ] **B2.3** `sync/skill/sqlexec_helpers_test.go` et `cmd/diag_exec/main.go` passent au
  découpeur canonique.
- [ ] **B2.4** Garde-rail `internal/archlint/no_local_sql_splitter_test.go` :
  - interdit, hors `internal/migration/helpers*.go`, toute définition de découpeur ou
    d'exécuteur de script SQL, et `strings.Split(<x>, ";")` dans un fichier qui exécute du SQL ;
  - dispense datée pour `arbitration_clocks_utc_guard_test.go` ;
  - échec si aucun fichier n'est lu ;
  - mutation vérifiée (une copie réintroduite fait rougir).
- [ ] **B2.5 Tests** (rouges d'abord ; aucun nouveau fichier dans `internal/sync/`) :
  - `TestEnsurePlayerSchema_TrailingCommentOnly` (tag `integration`, dans
    `schema_integration_test.go`) : surcharge
    `playerSchemaSQL` + `"\n-- note finale\n"`, restauration par `t.Cleanup`, sans
    `t.Parallel`. Aujourd'hui : « empty query ».
  - `TestExecScript_TrailingCommentOnly`.
- [ ] **B2.6** Commentaire de `steps_player_schema_authority.go:70-75` réécrit. Godoc de
  `SplitSQL` : limites écrites (chaînes, `/* */`).

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

- [ ] **B3.1 Mesure versionnée AVANT tout retrait** : `psa_index_repro_msr_planprobe_test.go`
  (tag `psarepro`), DB FICHIER de 12 000 lignes réalistes. EXPLAIN et temps avec et sans index
  pour les sept formes de lecture listées ci-dessus. Critère D-4. Au-delà : **STOP**, index
  conservés, raison écrite.
- [ ] **B3.2** Migration `drop_msr_secondary_art_indexes_v1` : cible player, 3
  `DROP INDEX IF EXISTS`. `order.go` après `player_msr_view_latest_by_type_v1` ;
  `stepDependencies` vers `lusr_chain_rework_v1` (précédent `drop_career_xuid_art_index_v1`,
  `order.go:126`).
- [ ] **B3.3** Retrait des `CREATE INDEX idx_msr_*` des autorités non scellées : `schema.go:111-113`,
  `steps_player_match_skill_rank.go:109-111` et `:179-181`. Baseline scellée INTACTE (précédent
  `idx_career_xuid`).
- [ ] **B3.4** Convergence (D-4) : les `DROP INDEX IF EXISTS` des trois `idx_msr_*` ET des trois
  index PSA retirés le 2026-09-20 (noms relus dans la migration PSA) entrent dans l'autorité
  rejouée par `EnsurePlayerSchema`.
- [ ] **B3.5** Suppressions de la liste ci-dessus, avec imports, types et jauges.
  `no_raw_rating_reads_test.go` n'est PAS modifié (fichier de `feat/perf-perimetre`) : s'il
  mentionne un index retiré, découverte.
- [ ] **B3.6** Ratchet : `match_skill_rank` entre dans `noSecondaryIndexTables`, avec une
  dispense DATÉE pour `steps_player_baseline.go`. Balayage étendu à `internal/sync/schema.go`.
  Mutation vérifiée.
- [ ] **B3.7 Tests** (rouges d'abord) :
  - `TestPlayerSchemaAuthority_NoMatchSkillRankSecondaryIndex` (`sync/schema_authority_test.go`) ;
  - `steps_player_drop_msr_secondary_indexes_test.go` : retrait, idempotence, lignes et vues
    préservées ;
  - convergence B3.4 : DB migrée, index MSR et PSA recréés à la main, puis `EnsurePlayerSchema`
    → plus aucun.
  - Aucun nouveau fichier dans `internal/sync/` (ratchet de gel) : les tests de `sync` vont dans
    les fichiers existants.
- [ ] **B3.8** En-tête du harnais `psarepro` réécrit : véhicule de reproduction pour les index
  player restants.
- [ ] **B3.9** Vérification sur COPIE d'une vraie player DB, jamais l'original (copie faite
  serveur principal arrêté, ce que le superviseur confirme avant). Migration puis `EnsurePlayerSchema` appliqués à la copie, par la CLI existante
  si elle migre une base désignée, sinon par un test d'intégration paramétré par une variable
  d'environnement (pas d'outil jetable). Attendu : `duckdb_indexes()` sans `idx_msr_*` ni index
  PSA, mêmes nombres de lignes, vues intactes.

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

- [ ] **B4.1 Nouveau balayage AST**, selon D-5 : `internal/archlint/campaign_exclusion_guard_test.go`
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
- [ ] **B4.2 Corrections.** Chacune porte un cas dans `campaign_exclusion_behavior_test.go` (tag
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
- [ ] **B4.3 À statuer par lecture de l'appelant ou de la capability** :
  - `fanout_repo.go:CountCommonMatchesForXUID` : si l'ensemble fourni est exclu à la source,
    dispense avec renvoi `fichier:ligne` ; sinon, correction.
  - `leaderboard_world_repo.go:GetStatLeaderboard:612` (agrégat sans filtre xuid, hors critère) :
    si le lecteur ne sert qu'un titre sans variante de campagne, justification écrite ; sinon,
    correction.
- [ ] **B4.4 Dispenses justifiées** :
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
- [ ] **B4.5** Tout autre entrant révélé par le balayage est statué selon les mêmes catégories.
  S'il n'entre dans aucune : **STOP**. La liste des lecteurs de `sync/`, `ops/` et `cmd/`
  (enquête : `sync/engagement.go:352`, `performance_helpers.go:228`, `session_recalc.go:30`,
  `skill/skill_rating_loaders.go:73`, `skill/skill_v2_shadow.go:434`, `assists_model.go:68`,
  `citations.go:379`, `friends_recompute.go:200`, `snapshot/snapshot_readiness_eval.go:208`,
  `csr_shared_backfill.go:194`, `ops/milestone_dates.go:294`) est écrite au journal pour le
  backlog. Rien n'y est corrigé.
- [ ] **B4.6** Aucune comparaison de slug ajoutée (ratchet `no_slug_comparison_test.go`). Les
  fichiers de `feat/perf-perimetre` ne sont pas touchés (§3.1.6).

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

- [ ] **B5.0** Ré-inventaire, au début du lot, des tâches de fond et des chemins de `main.go` (le
  code bouge). Chacun est classé : coupé en démo / redirigé sous la racine démo / légitime.
  Tableau au journal ; il fait foi pour B5.2-B5.5.
- [ ] **B5.1** Un seul helper de disposition démo, qui remplace les 3 copies, plus un garde-rail
  (test grep qui interdit la traduction hors du helper).
- [ ] **B5.2** `config.Load` en démo, sans variable explicite, selon D-7 :
  - `SessionDir` → `<démo>/runtime/sessions` ;
  - `AuthDir` → `<démo>/auth` ;
  - logs et crash log → `<démo>/runtime/logs` ;
  - `AppSettingsPath`, `DBProfilesPath` et overlays de titre → fichiers de la fixture.
  - `git check-ignore` vérifié pour `runtime/` sous `data/demo` et `tests/fixtures/demo-root`.
- [ ] **B5.3** Tâches de fond coupées en démo, avec garde explicite et un seul log Info au boot
  qui liste ce qui est coupé :
  - janitor ;
  - file persist asynchrone (`PersistBatchAsync` forcé à false, aucun `RecoverPending`) ;
  - magasin de monitoring en mémoire ;
  - santé données ; purge des rejeux ; surveillance disque ; migration des amis ;
  - et toute autre tâche classée « coupée » en B5.0.
- [ ] **B5.4** L'écrivain de la file de build (`sharedWriterForTitle`) passe par la même
  disposition démo que les 4 bases.
- [ ] **B5.5** Caches écrits en démo (aide, `jobs.json`) → `<démo>/runtime/`. Rejeux et faits de
  film lus sous la racine démo (aucun en démo, comme en prod). Cache d'assets : lecture seule
  autorisée.
- [ ] **B5.6** Sauvegarde : établir si `POST /settings/backup/run` est atteignable en démo
  (comportement de `RequireAdmin` en démo). Si oui : refus en démo, avec un test.
- [ ] **B5.7 Tests** (rouges d'abord) :
  - (1) `internal/config` : `Load()` avec un `LEVELUP_REPO_ROOT` leurre et le mode démo → aucun
    chemin d'exécution sous le leurre.
  - (2) `cmd/server` : le lancement des tâches de fond est extrait de `main()` vers une fonction
    testable, qui déclare le statut démo de chaque tâche. Sur un leurre à fichiers sentinelles,
    un manifeste avant/après (chemin, taille, date) doit montrer un écart nul, sans
    `monitoring.duckdb`.
  - (3) `demo_paths_test.go` étendu à tous les chemins de `cfg`.
  - (4) `archlint/no_data_path_join_test.go` étendu à `cmd/server` (`main.go:1373` corrigé, ou
    dispensé avec une justification datée).
- [ ] **B5.8 Preuve de bout en bout**, avec le serveur principal ARRÊTÉ (le superviseur le
  coordonne) : marqueur daté, `make demo-visual`, puis `find data logs -newer <marqueur>` doit
  être vide hors de la racine démo. Sortie au journal.

**Gate** : GO-F (paquets ciblés : `./cmd/server/ ./internal/config/ ./internal/archlint/ ./internal/platform/netguard/ ./internal/scheduler/ ./internal/api/... ./internal/ops/ ./internal/persist/...`),
plus B5.8.

### B-R — Revue adversariale des lots B

- [ ] Skill `adversarial-review` sur les commits de B1 à B5 (plage `<fusion 1>..<fin de B5>` de
  `feat/backlog-2026-09-26`), contexte frais (opus-high), seul agent actif.
- [ ] Constats P0 et P1 : lot de corrections par un exécutant opus-high, sur la même branche,
  gates du lot concerné rejoués. Constats P2 : découvertes, versés au backlog.
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
- A1.5 : `/health` → `000`. Non faite, reste au superviseur, à 2× (cf. statut de l'item).
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
