# Plan de release v8.0.0 — 2026-10-17

Sources : page Notion « Backlog LevelUp », section « Pour la v8 (ex-v7.5) — Séquence à dérouler à
la release, dans l'ordre » (relue le 2026-10-17) ; `docs/RUNBOOK_DEPLOY_CHECKLIST.md` ;
`.ai/PLAN_SECURITE_2026-10-15.md` §5 ; `docs/RUNBOOK_REPLAY_WORKER.md`. Contrat d'exécution :
skill **`plan-execution`** (fait foi), `delivery-checklist` à chaque clôture d'étape.

## 0. Cadre

**Objectif** : `feat/v75` en prod sous **v8.0.0**, avec les bases locales passées par toutes les
passes de rattrapage puis copiées sur le VPS, l'ouvrier csstat actif, la mise en commun des jetons
à 9 accords, sans perte de donnée prod hors des pertes assumées ci-dessous.

**Critère de succès** : chaque item statué ; `/health` 200 en prod ; contrôles de l'étape 6 tous
verts ; items de la séquence Notion cochés.

**Branche / worktree** : `feat/release-v8` dans `../LevelUp-wt-release-v8`, créée depuis
`origin/feat/v75` APRÈS l'étape 0. Fusion dans `feat/v75` sur accord de l'utilisateur. La fusion
`feat/v75` → `main` (PR, squash) reste **le geste de l'utilisateur** : push `main` = déploiement.

### Décisions tranchées par l'utilisateur

- **D1 (17/10)** : version **v8.0.0** — tag `v8.0.0`, `## [8.0.0]` dans les changelogs, bloc
  `**v8.0 — …**` dans les notes de version (la clé servie est tronquée à `vX.Y`).
- **D2 (17/10)** : **Nuzzles est déclaré en prod**, profil indépendant, **sans amis**. Déclaré par
  l'onboarding admin APRÈS le premier boot (le semis des amis et du groupe par défaut a alors déjà
  tourné : il n'hérite de rien). Sa base copiée est reprise telle quelle (`CreatePlayer` ne fait
  qu'un `MkdirAll`). Sans jeton propre, il dépend du pool (étape 5.8 avant 5.9).
- **D3 (17/10, précisée le même jour)** : recuisson en local des **films des matchs joués depuis
  le 2025-09-01** (date canonique du match), **plus tous les films des rejeux de la démo** quelle
  que soit leur date. Uniquement par `levelup backfill-replay` (un processus par film, plafond
  mémoire `filmproc`), jamais de boucle shell, jamais deux passes de cuisson en parallèle.
- **D5 (17/10)** : la recuisson est l'occasion de **recalculer les citations qui lisent le film**.
  Vérifié dans le code le 17/10 :
  - « Presque, ça ne compte pas » (`shieldless_kills`) et « Jouets éducatifs » (`pad_pickups`)
    lisent le résumé d'usage, qui n'existe que là où un artefact est sur disque (report D7 du plan
    citations : 127 matchs mesurés sur 926 pour JGtm). Après 2.3 et 2.4, le post-sync de chaque
    joueur recalcule SEUL les citations des matchs dont le film a été dérivé après elles
    (`citationsfilm.SelectMatchesFilmPlusRecent`, un cycle de latence) : aucune commande à écrire.
  - « Premier coup » (règle « Premier sang ») lit `highlight_events`, pas le film : déjà calculée
    sur tous les matchs au recalcul complet du 16/10. Rien à faire.
- **D6 (17/10)** : **démo** = un rejeu par mode présent dans les films, plus **un match 4v4 avec
  plusieurs véhicules** ; tous les films de la démo cuits en local ; les matchs ajoutés entrent
  dans le corpus de la démo.
- **D8 (utilisateur, 2026-10-17, « Oui je confirme »)** : rejeu **public** en prod dès la v8.0.0,
  en connaissance du critère non atteint ci-dessous (arbitrage disponibilité contre précision,
  à consigner aussi dans `replay_local_gate.go` lors de 7.3b s'il n'est pas retiré). Contexte :
  rejeu public en prod à la release
  (`LEVELUP_REPLAY_PUBLIC=1`, déjà dans `.env.local` prod depuis le 23/09). Le runbook la classe
  « décision produit » : à la dernière mesure, le critère du garde n'était pas atteint
  (`64e8adfa` à 87,39 % de couverture des tirs pour 88 % exigés). Sans cette variable, le rejeu
  répond 404 à tout visiteur et l'ouvrier construit quand même les artefacts.
- **D7 (17/10)** : les matchs et rejeux ajoutés à la démo **survivent aux releases suivantes**
  (émission additive, magasin `data/demo_films/` persistant sur le VPS, hors du `git clean`), et
  **les films de la démo se cuisent en local** à chaque montée du schéma de rejeu, jamais sur le
  VPS sauf en secours.
- **D4 (17/10)** : aucun travail de code avant que la session « backlog des 12 corrections » ait
  fini et fusionné.
- Rappels antérieurs : tout en local puis copie, aucune passe sur le VPS (23/09) ; les 24 matchs
  présents seulement en prod (XxGdakilla187xX, hors de tout profil) sont perdus à la copie,
  assumé (23/09) ; mise en commun : 9 comptes désignés, **jamais XxGdakilla187xX** (16/10) ; droit
  applicable des conditions non précisé (16/10).

### Décisions de conception proposées par ce plan (l'utilisateur peut s'y opposer)

- **P1 — Bascule des bases AVANT le déploiement, prod arrêtée** (Notion la place après). Le
  premier boot v8 se fait directement sur les bonnes bases, sans migrer puis jeter les anciennes,
  et le build sur le VPS (2 Go de RAM) profite de la mémoire libérée. Coût : prod coupée pendant
  l'attente de la CI et le build (~1 h, à annoncer). Démo non coupée.
- **P2 — Staging hors du dépôt** : `/opt/levelup-staging/` (le `git clean` de `deploy.sh` a
  effacé `/opt/levelup/data.incoming/` le 09/10).
- **P3 — Tag `v8.0.0` poussé dès la fusion squash** : `deploy.sh` lit `git describe --tags`
  après l'attente de la CI ; poussé tout de suite, le tag est en place à temps.
- **Retour arrière** : revert du commit squash + push, puis `data/titles.pre-v8` (et
  `cache/replays.pre-v8`) remis en place, prod relancée. Le snapshot restic de 5.3 couvre le reste.

## 1. Étapes

Statuts : `[x]` fait · `[~]` couvert ailleurs (référence) · `[!]` non traité (justification).
Une étape est **close** quand tous ses items sont statués ET que son gate est passé.

### Étape 0 — Gel (attente)

- [x] 0.1 La session « backlog des 12 corrections » a fusionné dans `feat/v75` (clôture
      `00349dc8c`, suites `36366f4af`, `f032e0193`) ; CI verte au niveau job sur le dernier commit
      de code `2ed8b05b1` (runs 39216028758 CI, 39216028760 Deploy Pre-Check, 39216028612
      gitleaks ; E2E sauté, réservé aux PR vers `main`) — vérifié le 2026-10-19.
- [x] 0.1b Le lot **« délai du débrief »** a fusionné dans `feat/v75` (`1b01444a1`, docs
      `96019bdd8`) ; CI du commit de fusion verte sur tous les jobs (run 39131709680, E2E sauté :
      réservé aux PR vers `main`) — vérifié le 2026-10-18. **La preuve de l'objectif (< 5 min) est
      reportée par décision de l'utilisateur** (`.ai/REGISTRE_REPORTS.md`, section du lot) : elle se
      lit au tableau admin « Délai du débrief » après un match → reprise en 2.10 (local) et 6.7c
      (prod). À savoir pour le déploiement : le post-sync V2 prend désormais le verrou d'écriture
      de la base joueur, et une tâche de fond « relance des citations du film » démarre au boot
      (coupée en démo) → surveillées en 5.7 et 6.8.
- [x] 0.2 `ListAgents` le 2026-10-19 : 6 sessions, toutes au repos ; checkout principal à
      `6e4ebfc80` = `origin/feat/v75`, arbre propre. La preuve reportée du lot délai du débrief est
      faite depuis (`6e4ebfc80` : 2 min 49 s en local, 2 min 45 s par l'ouvrier).

**Gate** : `git log origin/feat/v75 -1` = commit de fusion de l'autre session, CI verte.

### Étape 1 — Code et docs (worktree `feat/release-v8`)

- [x] 1.1 (2026-10-19, `29f623091` sur `feat/rv8-groupes`) **La migration était déjà faite** depuis
      `3b2db1574` (V72-01 H4-H5) : les 7 routes `/groups` sont déclarées par `huma.*`, aucun
      `writeJSON` (vérifié par le superviseur : 7 déclarations, 0 route chi-native) ; l'exception
      de l'allowlist était périmée. Seule action : retrait de l'entrée. Gates de l'exécutant :
      `go test ./internal/api/...` 0, `go vet` 0, `openapi-gen -check` 0, `tsc` 0, vitest groupes
      + garde de fraîcheur 0 ; golangci : 49 constats préexistants, aucun dans le fichier touché.
      Texte d'origine de l'item : **Groupes au format standard de l'API** : migrer les routes de
      `internal/api/handlers/groups.go` (345 L) vers Huma (path params `{id}`/`{xuid}`, flux
      d'invitation Xbox), comportement de `groups_test.go` conservé, `openapi.yaml` +
      `make generate-types` régénérés, appelants web adaptés, entrée `groups.go` retirée de
      l'allowlist de `json_huma_coverage_test.go`.
      Gate : `go test ./internal/api/...` exit 0 ; `make check-types` exit 0 ; vitest des fichiers
      groupes touchés ; `golangci-lint` 0 issue sur le paquet.
- [~] 1.2 et 1.3 (version, notes, README) : déplacés en **étape 3b**, dernière étape avant le VPS
      (utilisateur, 2026-10-18), pour couvrir tout ce qui aura fusionné, correctifs de l'étape 3
      compris.
- [x] 1.3b (2026-10-19, `54a2402db` sur `feat/rv8-rejeu-cli`) Fait. `--since` (date canonique,
      borne incluse, sans date = écarté, compté, journalisé ; lecture `OpenReadForQuery` de
      `match_registry`), `--match` (résolveur partagé avec killsource et vehicle-takes, préfixes ≥ 8),
      `--demo` (`ops.DemoReplayMatchIDs`, manifestes du `--title`, liste vide = refus).
      **`--since` + `--demo` = union** (le cas D3 se lance donc en UNE commande :
      `backfill-replay --since 2025-09-01 --demo`) ; `--match` seul (refusé avec les deux autres).
      Gates de l'exécutant : `go test ./cmd/levelup/` 0, tests ops ciblés 0, archlint 0, `go vet` 0,
      golangci ratchet (`--new-from-merge-base=origin/main`) 0 ; aucun `.py` ni CRLF dans le diff
      (vérifié par le superviseur).
      Texte d'origine : **Recuisson bornée (D3)** : `backfill-replay` n'a ni filtre de date ni liste de matchs
      (constaté le 17/10). Ajouter `--since AAAA-MM-JJ` (date canonique du match, fragment SQL
      partagé de timezone, jamais `start_time` brut) et `--match id1,id2,...` (même forme que
      `backfill-vehicle-takes`), et `--demo` qui lit la liste `replay_matches` des manifestes
      `config/demo/*/` (aucune liste d'identifiants recopiée à la main), combinables avec
      `--dry-run`, `--limit`, `--force`.
      Gate : tests unitaires du filtre (bornes incluses, match sans date écarté et compté) ;
      `go test ./cmd/levelup/` exit 0. Le `--dry-run --since 2025-09-01` sur les vraies données
      est la première commande de 2.3 (serveur arrêté), pas un gate de l'exécutant (son worktree
      n'a pas les données).
- [x] 1.3c (2026-10-19, `e9e90d7a1` sur `feat/rv8-demo`) Fait. Famille `vehicles` : 8 présents au
      départ (`present_at_beginning`, bots compris, repli sur tous les participants si le drapeau
      est NULL) en exactement deux camps de 4 ; `COUNT(DISTINCT family) >= 2` sur
      `match_vehicle_takes_latest` (`take`, mesuré, `family <> 'unknown'`) ; départage montées,
      date canonique, id. Émission additive (`mergeDemoReplayPicks`) : famille figée conservée,
      famille absente remplie, famille inconnue conservée (WARN), idempotente. Corpus : déjà garanti
      (`CorpusMatchIDs` = union avec les rejeux figés), test ajouté ; dédoublonnage d'un match sous
      deux familles dans `seedDemoReplays`. Gates de l'exécutant : `go test ./internal/ops/` 0,
      intégration `-p 1` 0, `go vet` 0, archlint 0 ; mutations (seuil à 1, dédoublonnage retiré)
      rendues rouges.
      Texte d'origine : **Rejeu « 4v4 avec plusieurs véhicules » de la démo (D6)** : nouvelle famille dans
      `internal/ops/seed_demo_replay_picks.go`, choisie par règle déterministe et non par nom de
      variante : match à 8 joueurs (4 contre 4), au moins 2 types de véhicules distincts montés
      (`match_vehicle_takes`), départage par le plus grand nombre de montées, puis le plus récent,
      puis le plus petit `match_id`. Le match de chaque rejeu figé est garanti dans le corpus de la
      démo (vérifier ; l'ajouter s'il n'y est pas). Strings UI éventuelles en FR et EN.
      **Émission ADDITIVE (D7)** : aujourd'hui `EmitDemoReplayPicks` remplace toute la liste
      (`m.Corpus.ReplayMatches = picks`), donc une réémission future changerait les rejeux pour des
      matchs plus récents. Désormais une famille déjà figée garde son match ; seules les familles
      vides sont remplies ; un retrait se fait à la main dans le manifeste.
      Gate : tests de `pickDemoReplays` (candidat retenu, candidat à un seul type de véhicule
      écarté, départage) et de l'émission (rejeu existant conservé face à un candidat plus récent,
      famille vide remplie) ; `go test ./internal/ops/` exit 0.
- [x] 1.3d (2026-10-19 : `8e1540403` + `2d6580168` sur `feat/rv8-demo-film`, fusionnés dans
      `feat/release-v8` `d394589ac` (CI 39264886409 verte), puis dans `feat/v75` `ec9805253` par-dessus
      le commit de l'utilisateur `66add1866` (cockpit Tactique) ; CI 39267242985 verte.) 14 tables dérivées du
      film extraites (usages, bonus par palier, véhicules, vies, contexte et coéquipiers à la mort,
      placement, prises de drapeau, ouvertures de kill, stats de bombe, événements d'objectif,
      positions) + `personal_score_awards` (base joueur : radar et score d'objectif),
      `player_squad_offset` (couples du corpus seulement) et `player_assists_model` (sans
      identité) ; colonnes d'identité déclarées (garde de schéma + contrôle des valeurs de fin de
      seed) ; mutations rouges par table à identité. Gates de l'exécutant verts ; gate du
      superviseur sur l'état fusionné : `go vet ./internal/ops/` 0, `go test ./internal/ops/
      ./cmd/levelup/` ok, intégration `-p 1 ./internal/ops/` ok (104 s). Non extraites, avec
      raison : `match_weapon_shots`, `match_weapon_hit_distance` (aucun lecteur) ;
      `match_objective_score_timeline` n'existe plus. **À vérifier en 2.9** : comptes source =
      démo par match du corpus sur ces tables ; seed sans `ErrDemoIdentityLeak` ; à l'écran,
      Emprise (véhicules, bonus), Tactique, chronologie d'objectifs, carte de chaleur, radar,
      score d'objectif non nul, synergies d'Escouade.
      Texte d'origine : **Démo : données dérivées du film** (D-9 ; décision de l'utilisateur du 2026-10-19 :
      « on corrige avant la release ») : inventaire des tables (et fichiers) que lisent la vue
      match (onglet Emprise, usages, citations du film) et la page Tactique ; extraction de chacune
      dans la démo, filtrée sur le corpus, colonnes d'identité déclarées et anonymisées, vues
      `_latest` intactes ; garde d'anonymisation étendue (mutation rouge). Exécutant Sonnet,
      worktree `LevelUp-wt-rv8-demo-film`, branche `feat/rv8-demo-film` depuis `2b4953ce5`.
      **Amendement d'ordre (2026-10-19)** : 1.3d ne bloque que 2.9 (seul `seed-demo` en dépend) ;
      les passes 2.1 à 2.8 partent sans l'attendre, avec un binaire construit depuis `2b4953ce5`
      (2.9 reconstruit le binaire après la fusion de 1.3d).
      Gate : `go test ./internal/ops/` + intégration `-p 1` 0, archlint 0, ratchet lint 0 ; fusion
      dans `feat/v75` (accord de l'utilisateur pour le lot : le « on corrige » du 19/10), CI verte.
- [x] 1.4 (2026-10-19, `6b12cf885`) Fait : squash + `git revert <squash-commit>` ; étape « Replay
      schema bump: cook the demo locally » (`--demo --dry-run` puis `--demo`, serveur arrêté, accord
      utilisateur ; rsync `--files-from` des seuls artefacts de la démo ; contrôle : rejeux du seed
      `copie`/`conserve`, jamais `perime`/`recuit`/`absent`).
      Texte d'origine : **Runbook de déploiement** : `docs/RUNBOOK_DEPLOY_CHECKLIST.md` prescrit « merge commit,
      no squash » et un retour arrière `git revert -m 1`, faux depuis que GitHub n'autorise que le
      squash (CLAUDE.md, stratégie de branches §8). Aligner sur le squash (revert simple du commit).
      Ajouter l'étape permanente **« Replay schema bump: cook the demo locally »** (D7) :
      `levelup backfill-replay --demo` en local, puis rsync des artefacts de la démo vers
      `/opt/levelup/data/cache/replays/` du VPS AVANT le push sur `main` ; le seed de la démo les
      copie alors au lieu de recuire sur le VPS. Contrôle : bilan du job `deploy-demo` sans rejeu
      « périmé » ni « recuit ». (Runbook EN-only, règle docs/FR.)
- [ ] 1.5 (2026-10-19 : CI de `feat/release-v8` @ `8c5edda15` VERTE sur tous les jobs, runs
      39257297620 CI, 39257297550 Deploy Pre-Check, 39257297544 gitleaks ; `npx playwright test`
      local contre `make dev` : 34 passés, 47 sautés (specs DEMO_MODE), 0 échec, 12,8 min ;
      accord de l'utilisateur le 19/10 ; fusion `--no-ff` `2b4953ce5` poussée sur `feat/v75` ;
      CI de la fusion VERTE (run 39259914637 CI, Deploy Pre-Check, gitleaks : success). [x])
      Fusion `feat/release-v8` → `feat/v75` sur accord ; CI verte au niveau job ;
      `npx playwright test` en local (l'E2E ne tourne en CI que sur la PR vers `main`).
- [x] 1.6 (2026-10-19) **Décision de l'utilisateur du 19/10** (« ok avec tes recos, on ne garde
      que wt/power-positions ») appliquée : étude + maquette « Tendances » versées dans `.ai/`
      (`ETUDE_TENDANCES_2026-09-30.md`, avec en annexe l'entrée de journal du 30/09 jamais
      versionnée ; maquette identique octet pour octet) ; recherche 2.7.d1 copiée dans
      `Downloads/LevelUp_recherche_ri27d1/` (6 fichiers neufs + 3 modifiés + patch, comparés
      octet pour octet) ; worktrees `imagecle` (jonctions retirées d'abord), `equip-porte` et
      `trend-analysis` retirés ; branches `feat/cg-l2`, `feat/cg-l9`, `feat/cg2-ln`, `feat/cg2-lp`,
      `claude/trend-analysis-halo-stats-8771a8`, `feat/faille-pose`, `feat/imagecle-etat-complet`,
      `feat/tactique-coequipiers` supprimées ; `wt/power-positions` GARDÉE (locale et `origin`).
      Gate : `git worktree list` = principal + release + `rv8-demo-film` (lot 1.3d en cours) ;
      cache de films 90 030 + 2 879, identique.
      Premier passage : Cache de films avant/après :
      90 030 morceaux + 2 879 manifestes, identique. 11 worktrees propres et entièrement fusionnés
      retirés (8 jonctions retirées d'abord par `(Get-Item).Delete()`, `remove` sans `--force`),
      plus le worktree de gate oublié `Temp\replay-corpus-gate-3878327983` ; 56 branches locales
      fusionnées supprimées ; 54 branches `origin/` fusionnées supprimées ; `disk-hygiene` :
      1,98 Go libérés (dont les trois caches de lot rv8). **Soumis à l'utilisateur** :
      3 worktrees avec du travail non commité (`trend-analysis-halo-stats-8771a8` : étude et
      maquette « Tendances » du 30/09, jamais versionnées ; `LevelUp-wt-equip-porte` : ébauche de la
      fin de faille, remplacée par `3881e58ad` dans `feat/v75` ; `LevelUp-wt-imagecle` : fichiers
      de recherche 2.7.d1, tag `research`, non versionnés par conception) et 5 branches non
      fusionnées (`feat/cg-l2`, `feat/cg-l9`, `feat/cg2-ln`, `feat/cg2-lp` : lots « non retenus »
      de la campagne grammaire ; `wt/power-positions` : docs du 20/09, aussi sur `origin`).
      Texte d'origine : **Nettoyage des worktrees, branches et caches** (après 1.5) :
      - worktrees : pour chacun des 11 hors checkout principal (`git worktree list`, sauf
        `LevelUp-wt-release-v8`, retiré en 7.6) : commits non fusionnés dans `feat/v75`
        (`git log feat/v75..<branche>`) ; s'il y en a → liste soumise à l'utilisateur, rien supprimé ;
        sinon jonctions retirées d'abord (`(Get-Item).Delete()`), puis `git worktree remove` SANS
        `--force` ;
      - branches locales fusionnées (`git branch --merged feat/v75`) supprimées ; non fusionnées →
        liste soumise ;
      - branches distantes fusionnées supprimées (`git push origin --delete`) sur la même règle ;
      - `scripts/disk-hygiene.ps1` en simulation, puis `-Apply` sur accord.
      Gate : `git worktree list` = principal + release ; cache de films intact (même nombre de
      fichiers avant/après).

### Étape 2 — Passes locales (code gelé = `feat/v75` après 1.5, serveur local arrêté)

Binaire construit une seule fois : `go build -o <scratchpad>/levelup_v8.exe ./cmd/levelup` depuis
`apps/go-api` (jamais `go build ./...`). Ordre imposé par les dépendances entre passes.

- [x] 2.1 (2026-10-19) Binaire `levelup_v8.exe` construit depuis `2b4953ce5` (scratchpad de la
      session). air (pid 29516) et `tmp/server.exe` (pid 3852) arrêtés ; port 8000 libre ; aucun
      verrou de décodage. Vite laissé en route (sans base).
- [x] 2.2 (2026-10-19) Simulation : 239 cartes (117 à télécharger, 1 ajoutable, 121 couvertes) ;
      hors ligne : Harvest ajoutée (18 zones nommées) ; complet : 78 ajoutées (2 925 zones dont
      2 924 nommées), 36 en échec ; relance : 200 couvertes, les 36 mêmes en échec, TOUTES en
      HTTP 404 (cartes Forge retirées de Waypoint : définitif, pas une panne), 3 sans zone ; un
      `string_id` sans libellé (`0x4C39AD38`). `map_callouts.json` modifié (md5 avant
      `5bbba3d4…`, après `de5c583a…`).
      Texte d'origine : **Zones nommées des cartes Forge** : `backfill-map-callouts --dry-run`, puis
      `--hors-ligne`, puis sans option. Gate : bilan chiffré final, `map_callouts.json` modifié.
- [x] 2.3 (résultat au journal, 19/10) **Recuisson** (D3) : `backfill-replay --since 2025-09-01 --demo --dry-run` (union, cf.
      1.3b ; les rejeux déjà figés de la démo sont cuits dans la même passe ; 2.9 ne recuit que les
      rejeux nouvellement émis, dont le 4v4 à véhicules) (nombre de films ;
      ~30 s par film, 2 742 films en cache au total le 17/10), puis la même sans `--dry-run` et
      sans `--force` (cuit les manquants et les périmés, reprend après interruption).
      Puis `backfill-replay --match` sur les sept films du corpus nommé du garde de rejeu
      (`replay_local_gate.go` : 5 sur 7 sans artefact en local le 17/10), pour pouvoir mesurer 7.3b.
      Gate : le dry-run rejoué = 0 film à cuire ; échecs listés et statués.
- [!] 2.3b (19/10 : 43 restent appauvris, limite du décodeur, voir journal et D-12) **Artefacts appauvris** (`RUNBOOK_REPLAY_WORKER.md` §4 étape 2, à faire AVANT le
      premier passage de l'ouvrier ; les artefacts locaux partent en prod) :
      `backfill-replay --repair-impoverished --dry-run`, puis sans `--dry-run`.
      Gate : le dry-run rejoué = 0 artefact appauvri.
- [x] 2.4 (résultat au journal, 19/10) `backfill-usage-summary` (lit les artefacts : après 2.3).
- [x] 2.5 (résultat au journal, 19/10) `backfill-pad-tiers --force` (lit les artefacts : après 2.4).
- [x] 2.6 (résultat au journal, 19/10) `backfill-vehicle-takes --dry-run`, puis `--force`. Gate : soirée Big Team Battle, onglet
      Emprise, lignes « Véhicules » chiffrées.
- [x] 2.7 (résultat au journal, 19/10) `backfill-killsource`, puis `--status`. Gate : 0 erreur.
- [x] 2.8 (résultat au journal, 19/10) **Inventaire fermé des autres passes** — chacune jouée ou statuée `[~]`/`[!]` :
      `backfill-bomb-stats`, `backfill-flag-grabs-net`, `backfill-h5-kill-mechanics`,
      `backfill-is-ranked`, `backfill-medailles-feed`, `backfill-registry-names`,
      `backfill-squad-creators` (dry-run d'abord ; 0 à écrire = `[x]` « rien à faire »).
- [x] 2.8b (résultat au journal, 19/10) **Dérivés Halo 5 recalculés hors Campagne** (ajouté le 2026-10-18 depuis le plan backlog du
      2026-10-16, runbook R2-R4 ; `--title` honoré depuis ce plan) : `levelup_v8.exe backfill --all
      --title halo_5 --engagement-scores --force`, puis `--perf --force`, puis `--assists-model --force`
      (une commande par passe, dans cet ordre). Gate : chaque passe sort 0, nombre de joueurs Halo 5
      traités = profils Halo 5 non `auth_only` de `db_profiles.json`. Citations : sans objet pour Halo 5
      (`citations.engine` non déclaré, la passe répond « non supporte »).
- [x] 2.8c (résultat au journal, 19/10) **Dates de paliers Halo 5** (runbook R4b) : depuis `apps/go-api`,
      `go run -tags cgo ./cmd/backfill_milestone_dates --data-root ../../data/titles/halo_5` (simulation),
      puis la même avec `--apply`. Le chargeur exclut la Campagne. Gate : bilan sans erreur ; une date
      non dérivable est NULL (aucune date affichée), jamais fausse.
- [x] 2.8d (résultat au journal, 19/10) **LUSR v2 de tous les joueurs, les deux titres** (runbook R5-R6 ; la v1 est supprimée depuis
      le plan backlog du 2026-10-16) : `levelup_v8.exe backfill --all --lusr`, puis
      `levelup_v8.exe backfill --all --title halo_5 --lusr`. Réécrit `match_skill_rank` (v2) et les 8
      composantes de `lusr_component_history` ; remplace les lignes v1 laissées par d'anciennes
      exclusions (servies sinon par le graphe d'évolution). Gate : 0 erreur ; mesure de référence :
      JGtm, 1 108 matchs, 47 s.
- [x] 2.8e **Indicateur « données de compétence » des 157 matchs synchronisés du 25 au 30 mai**
      (ajouté le 2026-10-18 depuis le plan backlog du 2026-10-16) : la donnée est en base (MMR d'équipe
      présent, mesuré sur la copie du 16/10 : JGtm 137 matchs, Chocoboflor 87, Madina97294 75,
      XxDaemonGamerxX 10), seul le bit skill (4) de `backfill_completed` manque. Posé en local le
      2026-10-18 par `levelup reset-bitmasks --dry-run` puis `--apply` (serveur arrêté), donc dans la copie
      des bases vers le VPS : RIEN à faire en prod. Gate de release : `levelup_v8.exe reset-bitmasks
      --dry-run` annonce 0 ligne `match_registry` à mettre à jour (vérifié le 2026-10-18 à 23:43 : 0).
- [x] 2.9 (résultat au journal, 19/10 ; contrôle à l écran de la démo -> 3.1 et 5.11) **Démo (D6)** :
      - inventaire des modes présents dans les films où joue le joueur source de la démo : une
        ligne par famille avec son match retenu ; les variantes qu'aucune famille ne reconnaît sont
        listées à l'utilisateur (ajouter une famille ou non) ;
      - `seed-demo --emit-replay-picks` (après 2.6 : la règle véhicules lit `match_vehicle_takes`),
        manifeste `config/demo/JGtm/halo_infinite.json` relu puis committé sur `feat/release-v8` ;
      - `backfill-replay --demo` : les films de la démo antérieurs au
        2025-09-01 sont cuits eux aussi. Sur le VPS, le seed reprend alors l'artefact du dépôt
        source (copié en 5.5) au lieu de recuire sur 2 Go de RAM ; puis 2.4, 2.5 et 2.6 rejoués
        (rapides, ils ne reprennent que ce qui manque ou a changé) pour ces films ;
      - `seed-demo` en local : remplit `data/demo_films/<titre>/`, bilan sans rejeu « périmé » ni
        absent.
      Gate : un rejeu par famille + le rejeu véhicules au bilan, tous au dernier schéma ; démo locale
      ouverte, un rejeu par mode s'affiche.
- [x] 2.10 (19/10, voir journal) Serveur local relancé (Start-Process détaché, port 8000) ; **un cycle de synchro complet
      par joueur suivi** pour le recalcul des citations du film (D5). Gate : logs
      `citations post-sync: film dérivé après les citations, recalcul` pour chaque joueur (ou la
      tâche de fond « relance des citations du film » du lot délai du débrief) ; si un match a été
      joué entre-temps, lire au tableau admin « Délai du débrief » la preuve reportée du lot
      (< 5 min, verrou de décodage libre) ; puis,
      sur une copie de la base partagée, `shieldless_kills IS NULL` ≈ 0 parmi les matchs depuis le
      2025-09-01 qui ont un film ; les citations « Presque, ça ne compte pas » et « Jouets
      éducatifs » augmentent sur la page Citations.
- [x] 2.10b (19/10, voir journal) **Sessions recalculées, les deux titres** (runbook R8 ; serveur relancé, route admin, pas de
      forme CLI) : dans la console du navigateur connecté en admin,
      `await (await fetch('/api/v1/settings/sessions/recalculate', {method: 'POST', credentials: 'include'})).json()`.
      Depuis le plan backlog du 2026-10-16, chaque profil est recalculé sur les bases de SON titre (avant,
      un profil Halo 5 ne l'était jamais). Gate : réponse sans erreur, sessions Halo 5 présentes sur la
      page Sessions d'un joueur Halo 5.
- [x] 2.11 (19/10, voir journal ; à REFAIRE en 5.2 avant le rsync) Serveur local arrêté, vidage des journaux des bases (ouverture/fermeture → 0 `.wal` sous
      `data/titles`), `levelup healthcheck` tout OK, serveur relancé.

**Gate** : `find data/titles -name '*.wal' | wc -l` = 0 ; healthcheck OK.

### Étape 3 — Contrôles de l'utilisateur (gate utilisateur)

- [ ] 3.1 Les 42 contrôles visuels et les 7 contrôles techniques de l'encadré Notion, sur le
      local à jour. Claude prépare une fiche avec un lien direct par contrôle (match, page, onglet).
- [ ] 3.2 Chaque anomalie : correctif (nouvelle branche, retour en 1.5) ou report en v8.1 —
      décision de l'utilisateur. Un correctif du décodeur fait remonter le schéma : rejouer 2.3 à
      2.7 (2.3 ne recuit alors que les artefacts périmés).

**Gate** : « ok release » de l'utilisateur, daté.

### Étape 3b — Docs et captures (dernière étape avant le VPS, branche `feat/release-v8`)

Ouverte quand l'étape 3 est close (plus aucun correctif attendu) ET que l'utilisateur a fourni
ses nouvelles captures d'écran (annoncées le 2026-10-18).

- [ ] 3b.1 **Numéro de version** : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`
      `## [7.5.0] - Unreleased` → `## [8.0.0] - <date de fusion>` ; `docs/RELEASE_NOTES.md` +
      `docs/FR/RELEASE_NOTES.md` `**v7.5 — …**` → `**v8.0 — …**` ; badge `Version-7.3.2` des deux
      README → `8.0.0` ; mentions « v7.5 » des README.
      Gate : `git grep -n "7\.5\.0\|v7\.5\|7\.3\.2"` sur `README.md docs/` = mentions historiques
      seulement.
- [ ] 3b.2 **Contenu** depuis la dernière mise à jour DU CONTENU de chaque fichier (repérée par
      `git log -p`, en respectant les formulations retouchées à la main par l'utilisateur), en
      couvrant tous les lots fusionnés jusqu'à l'étape 3 (dont « délai du débrief » et « messages
      de la vue match ») :
      - What's new de `README.md` (EN) + `docs/FR/README.md` (FR) : public JOUEUR ;
      - changelogs EN + FR : public technique ;
      - notes de version de l'app EN + FR (`docs/RELEASE_NOTES.md`, `docs/FR/RELEASE_NOTES.md`).
      Gate : extraction des blocs sur les fichiers réels (script node, `extractWhatsNewBlocks`) =
      nombre de blocs attendu, bloc `v8.0` présent dans les deux langues ; parité FR/EN des sections.
- [ ] 3b.3 **Captures d'écran** fournies par l'utilisateur : rangées dans `docs/screenshots/`
      (dossier partagé par les deux README) ; une capture qui remplace une existante garde son nom
      de fichier, une nouvelle est référencée dans LES DEUX README avec un texte alternatif FR et EN ;
      capture retirée = fichier supprimé et références retirées. Taille raisonnable (PNG compressé).
      Gate : chaque image référencée existe (`grep` des chemins des deux README contre le
      dossier), aucun fichier orphelin dans `docs/screenshots/`.
- [ ] 3b.4 Fusion `feat/release-v8` → `feat/v75` sur accord, CI verte.

### Étape 4 — Préparation du VPS (prod intacte ; prévenir avant chaque commande)

- [x] 4.1 (19/10 : revérifié, tout présent ; `replay_build_location` absent = worker) Lecture seule de `/opt/levelup/.env.local` : `LEVELUP_ENV=production`,
      `LEVELUP_BUILD_WORKER_TOKEN`, `LEVELUP_REPLAY_PUBLIC=1` présents (constaté le 17/10) ;
      `replay_build_location` absent de `app_settings.json` prod (constaté le 17/10) = défaut
      `worker` en production : le VPS ne décode jamais, il met en file pour csstat.
- [x] 4.2 (19/10, voir journal) **csstat** : l'unité `levelup-worker.service` (aujourd'hui un LIEN vers
      `/opt/levelup/packaging/…`, constaté le 17/10) devient une copie root (runbook §7 de
      `RUNBOOK_REPLAY_WORKER.md`), `systemd-analyze verify`. Sans ça, le job `deploy-worker` passe au
      rouge. Gate : fichier régulier `root:root`, verify exit 0.
- [~] 4.3 (19/10, premier passage ; complété par 5.2, serveur local arrêté) **Staging** (P2) : `/opt/levelup-staging/` (deploy:deploy), rsync msys64 + ssh msys64
      `--delete` de `data/titles/`, `data/cache/replays/`, `data/global/xbox_aliases.duckdb`,
      `data/demo_films/`. Gate : nombre de bases DuckDB et d'artefacts = local, 0 `.wal`.
- [ ] 4.4 Suppression de `/opt/levelup-data.incoming-restes-2026-09-23` (sur accord).
- [ ] 4.5 **Répétition du premier boot** : binaire serveur final sur une racine à part en local
      (copie de `data/titles` + `config/` + `static/` + `app_settings.json`/`db_profiles.json` de
      prod, sans `data/auth`), `LEVELUP_REPO_ROOT`/`LEVELUP_LOGS_DIR`/`LEVELUP_API_PORT`.
      Gate : boot sans FATAL, `friends: … created=N` (N > 0), groupe par défaut non vide.

### Étape 5 — Jour J (fenêtre annoncée, utilisateur disponible, chaque geste prévenu)

- [ ] 5.1 Aucune passe en cours, ni en local ni sur le VPS (`fuser` sur les bases partagées).
- [ ] 5.2 Delta : vidage des journaux en local, puis rsync `--delete` vers le staging (rapide).
- [ ] 5.3 Snapshot restic manuel sur le VPS, identifiant noté ici.
- [ ] 5.4 `docker compose stop levelup` (la démo reste servie).
- [ ] 5.5 **Bascule** : `data/titles` → `data/titles.pre-v8`, `data/cache/replays` →
      `data/cache/replays.pre-v8`, puis les dossiers du staging mis à leur place, alias et
      `data/demo_films` idem ; `chown -R deploy:deploy`. **Jamais** `data/auth`,
      `data/global/player_friends.json`, `app_settings.json`, `db_profiles.json`.
- [ ] 5.6 PR `feat/v75` → `main` (description), CI + E2E vertes → **squash par l'utilisateur** →
      tag `v8.0.0` poussé aussitôt sur le commit squash (P3).
- [ ] 5.7 Suivi du déploiement : attente de la CI, démo puis prod, conteneurs durcis
      (`docker compose ps` + logs : aucun « permission denied » ni « cannot read directory » sur
      `data/auth` ou `film_facts`), sortie de `deploy.sh` = `LEVELUP_APP_VERSION=v8.0.0`,
      `/health` 200 ; logs du boot : la tâche « relance des citations du film » démarre en prod
      et PAS dans la démo ; aucun conflit de verrou de la base joueur au premier post-sync.
- [ ] 5.8 **Mise en commun des jetons** (juste après le boot) : `pool-sharing list` ;
      `pool-sharing grandfather --players JGtm,Chocoboflor,XxDaemonGamerxX,Madina97294,DankerGlue,GeleJugefi,Trimbutton,QuiteSiren,UppedJoker`
      (simulation), puis la même avec `--yes` ; `list` = 9 `granted`, XxGdakilla187xX `undecided` ;
      `docker compose restart levelup`.
- [ ] 5.9 **Nuzzles** (D2) : onboarding admin (xuid `2535472547643888`), aucun ami, hors du groupe.
- [ ] 5.10 **Ouvrier** (D’ABORD, en root sur csstat, une fois le déploiement de l’ouvrier passé : `install -o root -g root -m 0644 /opt/levelup/packaging/systemd/levelup-worker.service /etc/systemd/system/levelup-worker.service && systemctl daemon-reload && systemd-analyze verify …`, cf. 4.2 ; puis séquence de `RUNBOOK_REPLAY_WORKER.md` §4, qui fait foi) : test d'accès à `/api/v1/internal/build-queue/claim` (503 sans jeton, 401
      jeton faux ; jamais 403 `csrf_rejected` ni 404) ; `allow <IP csstat>; deny all;` dans
      `location /api/v1/internal/` + `nginx -t && systemctl reload nginx` ; sur csstat
      `systemctl enable --now levelup-worker` ; `journalctl -u levelup-worker` ; un job de bout en
      bout (claim → décodage → artefact) visible dans le tableau admin.
- [ ] 5.11 Démo : un rejeu s'ouvre sur demo.lvelup.info.
- [ ] 5.12 Conf nginx versionnée (`packaging/nginx/`) = VPS (`sites-available/levelup`,
      `demo-levelup`, `snippets/levelup-assets-v.conf`, `conf.d/rate_limit.conf`) ; écart →
      correction du fichier versionné (branche dédiée).

### Étape 6 — Contrôles après le boot

- [ ] 6.1 `schema_migrations` de `metadata.duckdb` : migrations médailles passées ; la médaille
      VIP s'affiche « Clash of Kings » avec son image.
- [ ] 6.2 `data/global/player_friends.json` existe avec N entrées ; groupe par défaut non vide.
- [ ] 6.3 +1 h : logs `post-sync: kill source` → `sans_killfeed=0`, `ecrits` > 0,
      `backlog_restant` en baisse.
- [ ] 6.4 Synchro de Nuzzles réussie (`auto_sync: sync delta réussie`), plus de `joueur en échec`.
- [ ] 6.5 Logs `pool: créé` à la bonne taille ; bannière de mise en commun pour un nouvel inscrit ;
      Réglages > Synchronisation affiche la carte ; `/privacy` et `/terms` sans compte (FR et EN).
- [ ] 6.6 Pied de page « Licence AGPL-3.0 », confidentialité parle d'AGPL.
- [ ] 6.7 Onglet Emprise d'une soirée BTB : véhicules chiffrés. Rejeu Lattice ou Origin classé :
      zones nommées en français ; onglet Tactique : grappes nommées.
- [ ] 6.7b **Rejeu en prod** : depuis un navigateur hors du VPS, un rejeu s'ouvre depuis la vue
      d'un match (pas de `404 replay_not_available`) ; un match joué après le boot reçoit son rejeu
      par l'ouvrier (file `queued` → `running` → `succeeded` au tableau admin) ; un visiteur qui
      n'a pas joué le match n'obtient pas le rejeu (contrôle de participation).
- [ ] 6.7c **Délai du débrief en prod** : premier match joué après le boot, la mesure de bout en
      bout du lot « délai du débrief » (vue admin) montre toutes les données en moins de 5 min
      après la fin du match.
- [ ] 6.8 Post-déploiement du runbook : `/debug/vars` admin seulement, ffmpeg présent, premier
      cycle de synchro complet, pages clés FR + EN, endpoint device-code joignable.
- [ ] 6.9 Items de la séquence Notion cochés (sur demande de l'utilisateur).

### Étape 7 — Suivi et clôture

- [ ] 7.1 +48 h : réponses 401, 403, 413, 429 dans les logs nginx et serveur ; hausse sur des pages
      normales = protection à desserrer.
- [ ] 7.2 +7 jours : `memory.peak` des deux conteneurs (référence 16/10 : prod 1 134 Mio, démo
      334 Mio) → proposition de `mem_limit`.
- [ ] 7.3 Audit des `ON CONFLICT` sur les tables héritées sans clé primaire (registre daté, ne
      corrige pas — skill `adversarial-audit`).
- [ ] 7.3b **Garde « rejeu local seulement »** (`replay_local_gate.go`, à réexaminer au plus tard
      le 2026-11-08) : mesurer son critère de retrait sur les artefacts recuits des sept films du
      corpus nommé (couverture des tirs >= 88 % et pont nominal sur les sept). Atteint → retrait du
      garde ET de `LEVELUP_REPLAY_PUBLIC` (branche dédiée, règle n°11 : pas de kill-switch
      éternel). Non atteint → décision de l'utilisateur, datée dans le fichier.
- [ ] 7.4 Suppression de `data/titles.pre-v8` et `cache/replays.pre-v8` sur le VPS (sur accord,
      après 7.1).
- [ ] 7.5 `thought_log` + mémoire de clôture ; `.ai/REGISTRE_REPORTS.md` alimenté.
- [ ] 7.6 Worktree `LevelUp-wt-release-v8` retiré (jonctions d'abord), branche supprimée.

**Hors plan, sur demande explicite seulement** : retrait des captures mémoire du jeu, anonymisation
des gamertags des fixtures, réécriture de l'historique (push forcé sur `main` = déploiement).

## 2. Découvertes (consignées, non traitées hors périmètre)

- **D-1 (17/10)** : la page Notion dit le staging « PRÉPARÉ le 23/09 » ; il a été effacé par le
  déploiement du 09/10. Remplacé par 4.3.
- **D-2 (17/10)** : l'item Notion « Véhicules » place le rattrapage « après la copie des bases »,
  donc sur le VPS ; contraire à la décision du 23/09 (tout en local). Placé en 2.6.
- **D-3 (17/10)** : la prod n'a pas de `data/global/xbox_aliases.duckdb` (seulement un `.bak`) ;
  la bascule 5.5 apporte celui du local.
- **D-4 (17/10)** : `RUNBOOK_DEPLOY_CHECKLIST.md` prescrit un merge commit ; traité en 1.4 (dans le
  périmètre : le runbook est exécuté par ce plan).
- **D-5 (17/10)** : à chaque montée du schéma de rejeu, la régénération de la démo sur le VPS
  recuisait les films de la démo sur place (plafond `demoReplaySoftLimit` = 1 Gio ; au-delà le
  film est abandonné et l'ancien artefact reste servi, « périmé »). Un 4v4 avec véhicules risque
  de dépasser. Traité par D7 (cuisson locale + rsync, 1.3b et 1.4) ; la recuisson sur le VPS reste
  le filet de secours.
- **D-6 (17/10)** : le report « Presque, ça ne compte pas mesurée seulement là où l'artefact est
  sur disque » (registre, plan citations D7) est levé pour les matchs depuis le 2025-09-01 par 2.3,
  2.4 et 2.10 ; à mettre à jour au registre en 7.5 (les matchs antérieurs restent non mesurés).

- **D-7 (19/10, exécutant 1.1)** : `groups.go` — `handleRenameGroup` avale l'erreur de
  `h.groups.Get` (`updated, _ :=`, règle n°3) ; `handleGenerateInvite` ignore sans log l'erreur
  de `json.Unmarshal` (corps optionnel) ; les corps POST/PATCH lus en `RawBody` n'apparaissent pas
  typés dans `openapi.yaml`. Non traité (hors périmètre). L'item Notion « page des groupes au
  format standard » est à cocher comme déjà réalisé.

- **D-8 (19/10, exécutant 1.3b)** : le texte d'aide de `--match` est recopié dans trois
  sous-commandes (parseur partagé, aide non ; règle n°6 sans garde-rail) ; `printUsage` > 80 L et
  `main` complexité 54 (dette gelée) ; `registry.go` 1 144 L. L'exécutant signale qu'un Python
  lancé depuis Windows a réécrit des fichiers en CRLF pendant son travail (remis en LF, rien de
  Python versionné) — rappel : pas de Python, même comme outil d'édition.

- **D-9 (19/10, exécutant 1.3c, vérifié par le superviseur)** : la démo n'extrait AUCUNE table
  dérivée du film (`sharedTablesWhere`, `seed_demo.go:181` : ni `match_vehicle_takes`, ni
  `match_usage_*`, ni `match_pad_tiers`, ni `match_objective_score_timeline`). Les rejeux de la
  démo s'affichent (ils lisent l'artefact), mais les onglets Emprise / Tactique / usages des
  matchs de la démo seront vides ou « non mesurés », y compris pour le rejeu à véhicules (D6).
  Préexistant, hors périmètre : soumis à l'utilisateur le 19/10.
- **D-10 (19/10)** : la constante `replay.VehicleFamilyUnknown` est recopiée dans `ops`
  (`demoVehicleFamilyUnknown`, ratchet de façade du film) sans test qui les lie ; une dispense de
  `campaign_exclusion_guard_dispenses_test.go` cite un numéro de ligne (fragile).

- **D-11 (19/10, recuisson)** : anomalies de lecture journalisées en ERROR sur 26 films recuits (23 « entrée(s) du roster présente(s) SANS ÉQUIPE lue », 2 « vies PUBLIÉES sans identité », 1 « bot sans place ») ; artefacts produits. Liste : `replay_cuisson.log` du scratchpad de la session ; à recouper avec le contrôle visuel n°32 (`b1ad85eb`, 3 équipes).
- **D-12 (19/10)** : 43 artefacts restent « appauvris » après `--repair-impoverished` (registre d identité vide : aucune vie nommée ; ex. `07af7c78`, `0e640074`, `279ac3dd`, `29206c7c`, `2c16c553`, `f80cdf71`, `f10237b1`, `44e14331`). Le runbook de l ouvrier suppose la réparation suffisante : ce n est pas le cas pour ces films.
- **D-13 (19/10, instruit)** : `--assists-model --force` rend `n_modes=0` pour Halo 5 parce que la régression OLS (`sync/assists_model.go` : assists ~ kills + morts + dégâts infligés + dégâts subis + écart de MMR) est SINGULIÈRE : Halo 5 ne fournit ni `damage_taken` ni `team_mmr`/`enemy_mmr` (JGtm : 1 970 matchs, 1 970 assistances, 0 dégât subi, 0 MMR), deux colonnes constantes à 0, `fitOLS` rend nil. Les assistances Halo 5 existent ; seul le modèle personnel d « assistances attendues » manque (repli populationnel). **Attendu : les assistances attendues sont hors capacités de Halo 5 (utilisateur, 19/10).** Reste un défaut mineur : la passe `assists-model` (`cmd/levelup/cmd_backfill_titled.go`, `passAssistsModel`) n a pas de garde de capacité (contrairement à `citations`), et aucune clé de `capabilities.toml` ne nomme les assistances attendues ; elle tourne donc pour rien sur Halo 5. Hors périmètre.

- **D-14 (19/10)** : au premier démarrage du serveur après les passes, la mise en écriture de la base partagée pour la relecture killsource échoue (« drain inflight readers », « drain timeout, rollback vers RO ») pendant ~1 min, le temps du recalcul de masse des citations ; 3 films reportés au cycle suivant, rien ensuite. En prod, même premier boot avec les bases recuites : à surveiller en 5.7 (échecs transitoires attendus, persistants = anomalie).

- **D-15 (19/10, BLOQUANT POTENTIEL pour la prod)** : sur le serveur local, la prise d écriture de la base partagée Infinite échoue tant que des lectures sont en vol (« drain timeout, rollback vers RO »), et une fois `RestoreSharedAfterSwap: OpenReadOnly failed` survenu (20:38), TOUTE prise d écriture échoue (« open RW after RO close : base tenue par un autre processus ») jusqu au redémarrage, sans autre processus sur la machine (vérifié). En prod, le même état bloquerait les écritures de la synchro. À instruire avant la release (lot dédié, proposé à l utilisateur le 19/10).

## 3. Reprise de session

Relire `plan-execution`, puis ce fichier : reprendre à la première case vide de la première étape
non close. État git : `git log --oneline -10 feat/release-v8`. État des passes : `--status` de
`backfill-killsource`, `--dry-run` de `backfill-replay`.

## 4. Journal

(à remplir à chaque clôture d'étape : date, preuves, gates)

- **2026-10-19 — Étape 0 close.** Gate : `origin/feat/v75` = `6e4ebfc80`, CI verte sur le dernier
  commit de code. Worktrees créés depuis `6e4ebfc80` : `LevelUp-wt-release-v8`
  (`feat/release-v8`, intégration + plan), et un par exécutant de l'étape 1 :
  `LevelUp-wt-rv8-groupes` (`feat/rv8-groupes`, 1.1), `LevelUp-wt-rv8-rejeu-cli`
  (`feat/rv8-rejeu-cli`, 1.3b + 1.4), `LevelUp-wt-rv8-demo` (`feat/rv8-demo`, 1.3c). Exécutants
  Sonnet (préférence quota du 09/10), effort high.
- **2026-10-19 — Étape 1, items 1.1 à 1.4 faits.** Trois branches fusionnées (`--no-ff`, sans
  conflit) dans `feat/release-v8` : `eb33e29b3`, `7c1313d63`, `208133fed`. Gate du superviseur
  sur l'état fusionné : `go vet` (cmd/levelup, ops, handlers) 0 ; `go test -count=1
  ./cmd/levelup/ ./internal/ops/ ./internal/api/handlers/ ./internal/archlint/` tous ok. Push et
  CI de `feat/release-v8` ensuite, puis 1.5 sur accord.
- **2026-10-19 — Étape 1 close.** 1.1 à 1.6 statués ; `feat/v75` = `ec9805253`, CI verte ; E2E
  local 34/0. Étape 2 démarrée par amendement (2.1, 2.2 faits, 2.3 en cours depuis 15:42 :
  2 260 films à cuire, 733 faits à 16:28, 0 dépassement mémoire).
- **2026-10-19 — Étape 2, passes locales (binaire `levelup_v8.exe` @ `2b4953ce5`, puis `levelup_v8b.exe` @ `aa4871dcd` pour 2.9).**
  - 2.3 : 15:42 → 19:23, **2 249 construits**, 250 déjà à jour, 0 mort mémoire, 0 mort subite. Restent
    11 non cuisables, définitivement : 9 « carte hors catalogue » (Forge sans bornes : Kusini Bay ×2,
    Exiled ×2, Perdition, Narrows, Night Of The Undead…, retirées de Waypoint, cf. 2.2) et 2 films
    sans chunk de données (`f3e3112f`, `e869bcdf`). Dry-run rejoué : ces 11 seuls. Les 7 films du
    corpus du garde (7.3b) : `--match … --dry-run` = 0 à cuire, 7 à jour. Anomalies de lecture
    journalisées en ERROR, artefacts produits quand même : 23 « roster SANS EQUIPE », 2 « vies
    PUBLIEES sans identité », 1 « bot sans place » (D-11).
  - 2.3b `[!]` : 43 artefacts appauvris ; `--repair-impoverished` les reconstruit (43, 5 s, depuis les
    faits frais) mais ils RESTENT appauvris : le registre d'identité ne nomme aucune vie (ex.
    `07af7c78` Vacancy classé : `viesNommees=0`, `repli_table_identite_vide`). Limite du décodeur,
    pas de la passe : rejeux servis sans compteurs de joueur (D-12).
  - 2.4 usage : 2 433 écrits, 66 à jour, 0 échec (1 min 23). 2.5 pad-tiers `--force` : 2 498 écrits,
    0 échec. 2.6 véhicules : dry-run 0 (présence en base), donc `--force` après recuisson : 2 499
    écrits, 2 181 prises, 1 329 frags d'engin dont 106 non appariés, 0 échec. Contrôle Emprise à
    l'écran -> 2.10 / 3.1.
  - 2.7 killsource : 12 films (8 écrits, 751 morts, 3 sans kill-feed, 1 clé inconnue), 0 erreur ;
    crédit 9 243 matchs.
  - 2.8 : bomb-stats `--force` 8 écrits (un artefact recuit se re-projette par `--force`) ;
    flag-grabs-net `--force` 347 écrits ; registry-names 959 colonnes réparées sur 713 matchs
    (`game_variant_name` 713, `pair_name` 246 ; 5 `playlist_name` sans source) ; is-ranked 0 ;
    medailles-feed 0 identifié (passe inopérante, connue depuis le 23/09) ; squad-creators 0 ;
    h5-kill-mechanics `[~]` : one-shot déjà joué en juillet, simulation complète > 15 min (réseau),
    échantillon `--limit 20` : 0 ligne à corriger.
  - 2.8b Halo 5 (4 joueurs = profils non `auth_only`) : engagement 5 091, perf 5 032, assists-model
    `n_modes=0` pour les 4 (D-13). 2.8c : 40 dates recalculées, 1 NULL, `--apply` fait. 2.8d LUSR :
    Infinite 4 898 lignes (5 joueurs), 1 WARN EP non convergé (BTB `6222be27`) ; Halo 5 0 ligne :
    attendu, `match.skill.snapshot = "not_exposed"` pour halo_5 (aucune entrée de note).
  - 2.9 : émission additive : 9 familles conservées, `vehicles` remplie par `4898d586` (Behemoth,
    16 lignes de prise, 24 prises) ; escalation et extraction sans candidat ; variante sans famille
    `CASTLE WARS` (soumise à l'utilisateur). Manifeste reporté sur `feat/release-v8`. `--demo` :
    10 à jour. `seed-demo` : 0 fuite, 10 rejeux `copie`, Infinite 68 matchs ; base démo : véhicules
    67 matchs, usages 67, paliers 67, vies 68, contexte de mort 68 ; `4898d586` : 16 lignes =
    source.
  - 2.10 : air relancé à 20:20 (journaux du serveur sous `apps/go-api/logs/` : `LEVELUP_LOGS_DIR` non
    posé par ce lancement), `/health` 200. Citations du film recalculées pour les 5 joueurs entre 20:22
    et 20:23 (Chocoboflor 574, Madina97294 580, XxDaemonGamerxX 39, JGtm 1 139, Nuzzles 1 158 matchs).
    Sur une COPIE de la base partagée : depuis le 2025-09-01, 2 477 matchs résumés, 56 avec
    `shieldless_kills` NULL (186 lignes sur 20 157 : les artefacts appauvris de D-12 et voisins) ;
    avant : 109 résumés, tous NULL (rejeux non recuits, attendu). Au démarrage, 3 échecs killsource
    « drain inflight readers » + 6 « drain timeout, rollback vers RO » entre 20:22:21 et 20:23:27,
    aucun ensuite (D-14). Preuve du délai du débrief : aucun match joué depuis le boot (faite le
    19/10 matin, cf. 0.2).
  - 2.10b : route admin appelée depuis un Chrome où l'utilisateur s'est connecté (19/10). 1er passage
    (20:38) : 11 731 matchs mis à jour, mais 4 joueurs Infinite en erreur ; 2e et 3e passages : tous
    les joueurs Infinite en erreur (« open RW after RO close : base tenue par un autre processus »,
    puis `RestoreSharedAfterSwap: OpenReadOnly failed`) ; après redémarrage du serveur, 4e passage :
    Halo 5 recalculé (3 818 matchs) mais Infinite en « drain timeout » ; 5e passage, serveur calme
    (20:45) : 0 erreur, 0 changement (sessions Infinite déjà justes). D-15.
  - 2.11 : serveur arrêté ; 7 WAL restants (arrêt brutal de Windows) vidés par CHECKPOINT (outil
    jetable compilé hors dépôt, supprimé) ; 0 `.wal` sous `data/titles` ; `healthcheck` TOUT OK ;
    serveur relancé, `/health` 200.
- **2026-10-19 — Étape 2 close** (2.3b `[!]` justifié, D-11 à D-15 consignés).
- **2026-10-19 — Étape 4 avancée par amendement** (l'utilisateur : « continue tant que tu peux » ;
  4.1 à 4.4 ne dépendent pas de l'étape 3, la copie est rafraîchie en 5.2).
  - 4.2 : sur csstat, `/etc/systemd/system/levelup-worker.service` était un lien PENDANT vers
    `/opt/levelup/packaging/systemd/…`, fichier ABSENT du checkout de csstat (`97a109b0d`, branche
    `main` : l'unité n'existe que dans `feat/v75`). Lien retiré (ouvrier ni activé ni actif) ;
    copie impossible avant la release. `deploy-worker.sh` de la v8 traite une unité ABSENTE comme
    « phase pré-activation » (message, pas d'échec) et ne refuse qu'un LIEN : le risque de rouge au
    déploiement est levé. L'installation de la copie root + `systemd-analyze verify` passent en 5.10.
  - 4.1 : `.env.local` prod : `LEVELUP_ENV=production`, `LEVELUP_REPLAY_PUBLIC=1`, jeton de l'ouvrier
    présent ; `replay_build_location` absent de `app_settings.json`.
  - 4.3 : `/opt/levelup-staging/` créé (deploy:deploy). rsync msys64 : il faut
    `-o UserKnownHostsFile=/c/Users/Guillaume/.ssh/known_hosts` (sinon « Host key verification
    failed », le ssh msys64 lit un autre `known_hosts`). Copié : 3,1 Go de bases (15 sur 23 : les 8
    bases tenues par le serveur local sont verrouillées par Windows, « Device or resource busy »),
    5,4 Go de rejeux (2 928 fichiers, = local), 250 Mo de films de la démo, alias. Les 8 bases
    manquantes partent en 5.2, serveur arrêté.
