# PLAN — Intégration des branches actives dans l'architecture cuisson-perf, puis merge et push de feat/v75

> Date : 2026-09-05. Branche `wt/cuisson-perf` (worktree `LevelUp-wt-cuisson-perf`), HEAD de départ
> `31b754363`. Exécution sous le contrat du skill `plan-execution` (ordre strict, aucun report
> d'un item exécutable, statut par item, vérification sur pièces — rappel opposable au §7).
> Pilote : cette session (orchestrateur-vérificateur) ; exécution et revue par agents Opus
> (conflits, balayages) ou Sonnet (tâches bornées) ; le pilote commite (jamais de stash, jamais de
> hook contourné). Relu par `plan-review` le 2026-09-05 (verdict : exécutable après corrections —
> toutes fondues ici ; l'addendum `*_REVUE.md` est supprimé à la clôture de B).

## §0 Mandat (utilisateur, 2026-09-05)

« Le sujet optimisation doit reprendre le travail des véhicules et du translocateur/répulseur et
des nouvelles stats pour l'intégrer proprement à son refacto (branches actives non mergées avec
feat/v75). Quand ce sera fait tu mergeras dans feat/v75 [...] et quand tu auras fini, tu push et
vérifieras que tout est vert, même si ce n'est pas de ta faute. » Contraintes ajoutées : « propre,
solide et pérenne, si jamais tu dois faire des ajustements » ; « si tu peux orchestrer plusieurs
étapes en parallèle, go » ; **`feat/v75` est GELÉ par l'utilisateur pour la durée de
l'intégration** (« C'est gelé, je touche plus à rien », 15 h).

## §1 État des lieux (mesuré le 2026-09-05, 14 h 30)

| Branche | Avance / retard sur feat/v75 | Base | Schéma | Périmètre | Décision |
|---|---|---|---|---|---|
| `feat/v75` (origin `7fb4b60a1`) | — (65 commits depuis notre dernière réconciliation `ceabaad67`) | — | **38** | translocateur, propulseur, charges d'équipement (3 balayages ANCIENNE forme appelés dans `build.go:308/457/473`), vue match, himodule mmap, CI | **Étape A** — FAIT (`eb80a4f0a`) |
| `origin/feat/v75-vehicules-sons` (28/339) ∪ `wt/vehicule-deadstate` (25/339 : 4 DERRIÈRE `sons` + 1 devant, `0b5141b8a`, tests + notes) | voir cellule | `14a115bb1` (base commune des deux têtes : `f4c3ed417`) | **31** (bump propre à la branche) | 65 fichiers filmdec + 14 replay non-test, **8 balayages ancienne forme** (5 sites de production dans `build_vehicles.go`), `build.go` (+Vehicles, attachVehicles, attachVehicleShots, buildShots à 3 retours), `document.go` +97, web match-replay ×50, sons ×66, assets ×20, 3 outils cmd, `openapi.yaml` +282 et `generated.ts` +132 (GÉNÉRÉS) | **Étape B** — le gros morceau |
| `wt/session-usage` (10) | 10 / 22 | `da616828f` | 38 (inchangé) | paquet `sessionusage`, `replay/usage_summary*.go` (projection du DOCUMENT, aucun balayage), persister INSERT-only + 2 tables append-only neuves + vues `_latest`, migration, capability `film.usage_summary`, crochet dans `replayartifacts/{cuisson,artifacts,journal}.go` + `usage.go`, CLI `backfill-usage-summary`, killcollector (classifier, postsync), web session-detail (~2 100 l.), `openapi.yaml` +307 et `generated.ts` +140 (GÉNÉRÉS) | **Étape C** |
| `wt/game-changers` (4) | 4 / 15 | `ca55f0ed7` | — | web uniquement (match-replay ×12, ui ×3) | **Étape D** |
| `wt/assaut-stats` (3) | 3 / 34 | `146f1d92e` | 38 | `replay/bomb_stats.go`, `bomb_arms.go` + tests de recherche ; **aucun appelant dans tout go-api** ; worktree avec 6 fichiers NON commités (recherche du 04/09) | **[!] non intégrée** (D5) |
| `wt/r9-repulseur`, `wt/r12-repulseur`, `wt/son-propulseur`, `wt/lint-filmdec`, `wt/himap-mmap`, `wt/trame-film` | 0 | — | — | déjà dans feat/v75 | arrivées par l'étape A |
| `main` | 9 commits hors feat/v75 (hotfix 7.3.1 classement mondial) | — | — | — | hors mandat — signalé à l'utilisateur |

Corpus du harnais : 13 films (`internal/analysis/replay/testdata/equivalence/CORPUS.txt`),
**48 étapes observées depuis A** (45 + `translocations`, `abilityImpulses`, `abilityCharges`),
marqueur `# digest-grammar: 2`. Intersections web mesurées : D ∩ B = `match-replay/i18n.ts`,
`i18nContract.ts` ; D ∩ C = `match-replay/MatchEquipmentUsageSection.tsx` ; B ∩ C = les deux
fichiers générés + `lib/api/types.ts` + `thought_log.md`.

## §2 Décisions

- **D1 — Ordre logique** : A (amont) → B (véhicules) → C (usage de session) → D (game-changers) →
  E (gates + revue de branche) → F (merge, push, CI). L'ordre est confirmé par la relecture (le
  recouvrement web B↔D tient en deux fichiers i18n, B↔C en fichiers générés). Voir D10 pour
  l'exécution PARALLÈLE de B, C, D.
- **D2 — Rejouer, jamais réintroduire** : tout balayage en ancienne forme (`ScanFilm*(dir)`)
  arrivant d'une branche se réécrit en `Scan*(film, ...)` lisant `film.Packets(i)` et le
  `FilmContext` (bande de slots, layout catalogue, registre) ; une enveloppe `(dir)` n'existe que
  pour les tests qui l'appellent (liste fermée d'archlint) et se supprime sans appelant. Chaque
  nouveau balayage de production reçoit son étape observée (`BuildFromFilmSteps` + `opt.observe`).
  Les garde-rails (observe_test, no_film_reread, filmsource_leaf, no_recomputed_film_context,
  ratchet des vars, liste des enveloppes) sont la checklist : un rouge = un oubli, on corrige dans
  le sens de l'intention de la branche, jamais en élargissant une allowlist sans justification datée.
- **D3 — Un seul bump de schéma, et le numéro n'est pas acquis** : A a absorbé le 38 amont ; B porte
  le **39** (champs véhicules SUR le 38 ; le 31 de la branche et ses trois blocs de doc de version
  29/30/31 fondus en un seul bloc 39) ; C et D ne bumpent pas. `feat/v75` a bumpé sept fois en trois
  jours et a subi une renumérotation par collision (`6ce0fcc2a`, 29 → 30) : (a) le 39 est ANNONCÉ à
  l'utilisateur à l'ouverture de B (fait, 15 h — et l'amont est gelé) ; (b) F.1 re-vérifie
  `const SchemaVersion` sur `origin/feat/v75` AVANT le merge ; si ≥ 39, notre bump devient N+1 et
  les 13 TSV sont re-figés puis re-vérifiés (le numéro entre dans le digest `artifact`).
  **Complément D13 (16 h 20)** : le travail non commité d'Assaut porte LUI AUSSI `38 → 39`
  (armements pausables au document). Les deux 39 se FONDENT en un seul à G.1 (aucun des deux
  n'est déployé) : un seul bloc de doc de version 39 = véhicules + armements/portage ; les TSV
  sont re-figés à G.1 en changement déclaré (films d'Assaut du corpus), le numéro ne bouge pas.
- **D4 — Preuve par le harnais à chaque étape** : passe complète SANS `-update` d'abord ; la
  comparaison du harnais est POSITIONNELLE (`parent.go`, ligne i contre ligne i) : quand une étape
  s'insère, le diff exploitable est **PAR NOM D'ÉTAPE** (copie des TSV avant / après). Chaque écart
  doit être **corrélé** à l'intention de la branche (quels films, quelle étape, pourquoi ces
  films-là) — un écart inexpliqué = arrêt et rapport ; puis `-update`, puis vérification 13/13 ;
  diff des comptes au §5. Une ligne AJOUTÉE n'est pas un écart, une ligne MANQUANTE l'est : gate de
  lignes (48 après A, **49 après B**, partout). Une étape sans changement attendu (C, D) rend 13/13
  identiques SANS `-update` — tout écart y est une ANOMALIE (le diff Go de C sur le document est un
  renommage pur `padFamilyKey` → `PadWeaponFamilyKey`).
- **D5 — `wt/assaut-stats` n'est pas intégrée** : `BuildBombStats`/`BuildBombArms` n'ont aucun
  appelant hors de leurs tests (imports `sort`, `strconv`, `objectiveevents` seulement : bibliothèque
  non câblée — table `match_bomb_stats`, capability `film.bomb_stats` et écriture au sync restent à
  faire), **aucune dépendance au refacto** (post-décodage, rien à rejouer), et le worktree porte de
  la recherche non commitée. L'intégrer poserait du code mort dans feat/v75 (anti-pattern n°1) ; la
  branche mergera à la clôture de son chantier (nuance : elle modifie aussi
  `bombe_portage_gate_test.go`, 12 l. — pas strictement additive). Ajustement signalé.
- **D6 — Champ `structure` de l'artefact : hors périmètre** — quatre sites web subsistent, dont un
  lecteur métier (`features/match-replay/replayLogic.ts:274` repli des bornes ;
  `replayNormalize.ts:131/250/407`). Retrait = décision produit + web + schéma : au registre des
  reports.
- **D7 — mmap** : la projection mémoire vit dans `internal/himodule/projection_*.go` (modules de
  cartes), pas dans la cuisson des films ; `filmsource` lit des chunks zlib de quelques dizaines de
  Ko : aucun gain à attendre. Pas d'action ; consigné.
- **D8 — Merge final et CI** : fetch frais ; contrôle D3(b) ; si `feat/v75` a bougé malgré le gel,
  mini-réconciliation (protocole A) AVANT le merge ; merge dans le worktree partagé uniquement s'il
  est propre ; `git push origin feat/v75` (jamais `main`) ; CI verte **au niveau JOB sur la liste
  MESURÉE au push** (`gh run view --json jobs`) : `Frontend`, `Go Build + Test` (matrice), `Go Lint`,
  `Go Lease Enforcement`, `Go Coverage + Baseline`, `OpenAPI Lint`, `Go Contract Test`, et — parce
  que C touche `internal/persist/**` et `internal/platform/duckdb/**` — `ADR 0021 Gate —
  shared_social invariants` (workflow séparé, déclenché par paths). `E2E React (Playwright)` ne
  tourne PAS sur un push (`if: pull_request`) : son absence n'est pas un rouge. Un rouge
  préexistant se corrige aussi (mandat).
- **D8bis — Re-mesure de l'amont au début de chaque étape** (même gelé) : `git fetch`, puis commits
  depuis A, `const SchemaVersion`, et `git diff eb80a4f0a..origin/feat/v75 | grep -oE
  'ScanFilm[A-Za-z]+\('`. Delta non nul = mini-réconciliation IMMÉDIATE. Mesure écrite au §6, même
  nulle.
- **D8ter — Ce que la prod sert entre le push et la re-cuisson, écrit et assumé** : le 39 périme tous
  les artefacts (`cmd_backfill_replay.go:79`) MAIS aucune re-cuisson de masse ne part seule — le
  rattrapage du fil de l'eau sélectionne par `os.Stat` (présence du fichier), pas par schéma
  (`replayartifacts/backlog.go`). Corpus MIXTE 38/39 : calque véhicules absent sur l'historique,
  présent sur les matchs cuits après déploiement ; le front lit `schemaVersion` comme un nombre
  (`replaySchemaLogic.ts`) — dégradation gracieuse. Assumé jusqu'à l'accord utilisateur sur
  `levelup backfill-replay --only-existing` (jamais lancé d'office).
- **D9 — Union des deux branches véhicules** : les deux têtes sont DISJOINTES hors `thought_log.md`
  (base commune `f4c3ed417` ; `0b5141b8a` = 2 tests neufs + 2 notes) : l'ordre ne change pas le
  contenu. `origin/feat/v75-vehicules-sons` D'ABORD parce que ses 4 commits de CI réparent le lint
  `unused` et deux gardes archlint (état intermédiaire vert), puis `wt/vehicule-deadstate` (trivial).
- **D10 — Exécution PARALLÈLE de B, C, D en worktrees dédiés** (demande utilisateur, 15 h 40) :
  trois worktrees dérivés du commit A `eb80a4f0a` — `LevelUp-wt-integ-vehicules`
  (`wt/integ-vehicules`), `LevelUp-wt-integ-usage` (`wt/integ-usage`), `LevelUp-wt-integ-gamechangers`
  (`wt/integ-gamechangers`) — jonctions `data/cache/{film_chunks,film_manifests,mvar,replays}` vers
  le cache principal, `LEVELUP_REPO_ROOT` pointé sur le worktree (pas de `db_profiles.json`),
  **`GOCACHE` propre à chaque worktree** (`<wt>/tmp/gocache` — des builds Go concurrents sur un
  cache partagé l'ont déjà corrompu), verrou solo de cuisson partagé par construction (le cache est
  commun : les cuissons se sérialisent, c'est voulu). Chaque agent commite dans SA branche
  d'intégration (commit final de l'étape, autorisé par le pilote pour ces branches jetables) ; le
  pilote FUSIONNE ensuite séquentiellement dans `wt/cuisson-perf` : B, puis C, puis D — conflits
  attendus et bornés (§1 intersections : fichiers générés → regénération D11 ; `i18n.ts` → union
  des clés ; `MatchEquipmentUsageSection.tsx` → les deux intentions ; `thought_log.md` → union) ;
  après chaque fusion : build, tests des paquets touchés, harnais SANS `-update` (attendu : les TSV
  de B) ; puis E sur le résultat fusionné. Les trois worktrees et branches d'intégration sont
  supprimés à la clôture de E.
- **D11 — Contrat OpenAPI : `openapi.yaml` et `generated.ts` sont GÉNÉRÉS** (`openapi.yaml` porte
  l'en-tête « NE PAS ÉDITER À LA MAIN — `make openapi-gen` — verrouillé par
  `TestOpenAPIYAMLIsUpToDate` »). B et C les modifient tous deux. Un conflit de merge sur l'un des
  deux ne se résout JAMAIS à la main : on prend `--ours` (indifférent), puis on REGÉNÈRE et on
  committe le résultat de la machine : `make openapi-gen && make generate-types && make
  openapi-check`. Gate (B, C, et chaque fusion D10) : `go test ./internal/api/ -run
  TestOpenAPIYAMLIsUpToDate -count=1` · `CGO_ENABLED=0 go test ./contracttest/... -run TestContract
  -count=1` · `node tools/check-generated-types-fresh.mjs` (aucun job CI ne vérifie la fraîcheur de
  `generated.ts`). Si `make openapi-gen` produit un diff APRÈS résolution, la résolution Go était
  incomplète : corriger le handler/DTO, jamais le YAML.
- **D12 — Gate web = ce que fait le job CI `Frontend`, pas moins** (B, C, D, chaque fusion, E) :
  `npm run typecheck && npm run lint && npm run lint:fields` · `npm run test` (vitest COMPLET) ·
  `npm run build` · `node tools/knip-ratchet.mjs` (ratchet code mort — il casse typiquement à la
  réconciliation de plusieurs branches ; un export neuf non consommé se câble ou se supprime,
  JAMAIS relever le plafond).

- **D13 — Stats d'Assaut : D5 est RENVERSÉE par l'utilisateur (16 h 10)** : « tu peux quand même
  traiter ce point ». État réel mesuré : la branche COMMITÉE ne porte que le noyau pur (E1/E2),
  mais le worktree porte 29 fichiers modifiés + 8 non suivis NON COMMITÉS (E2-bis, E2-ter, E3 :
  migration `match_bomb_stats`, `BombStatsPersister`, câblage `BatchBuilder`, durcissements ART —
  « Complétés » au journal du 04/09, gates passés selon lui, jamais commités : la session s'est
  arrêtée avant). Rien n'est écrit en base, rien n'est servi, rien n'est affiché : la
  fonctionnalité N'EST PAS opérationnelle aujourd'hui. Reprise en Étape G (après les fusions D10,
  avant E) : sécuriser le travail non commité, intégrer, puis exécuter E4 (crochet au sync), E5
  (API + web), E6 (backfill, clôture) du plan Assaut, qui fait foi pour les décisions tranchées.

## §3 Étapes

### Étape A — Réconciliation amont (`feat/v75` `7fb4b60a1` → `wt/cuisson-perf`) — CLOSE
- [x] A.1 Merge `origin/feat/v75` (`7fb4b60a1`, 65 commits, 324 fichiers, 5 conflits résolus
  « sémantique amont dans notre architecture »). PREUVE : `--diff-filter=U` vide, aucun marqueur.
  Détail au §6.
- [x] A.2 Les 3 balayages amont en nouvelle forme : `ScanTranslocatorTeleports(film, entry)`,
  `ScanAbilityImpulses(fc)`, `ScanAbilityCharges(fc)` ; enveloppes `(dir)` gardées pour leurs seuls
  tests, liste archlint **40 → 43** ; `BuildFromFilmSteps` **31 → 34**. Aucune allowlist élargie,
  ratchet des vars inchangé à 116.
- [x] A.3 Gates — gofmt vide · build 0 · vet 0 · 66 paquets ok · intégration `-p 1` exit 0 · lint
  281 issues, 0 sur un fichier touché.
- [x] A.4 Harnais — diff PAR NOM : 3 étapes neuves, `equipmentChanges` 13/13 (récupération gatée),
  `positions` sur EXACTEMENT les 2 films à translocations > 0, `artifact` 13/13 (37 → 38) ;
  `-update` ; vérification **13/13** ; **48 étapes par TSV**. Témoins 17,9 / 21,0 / 20,5 s.
- [x] A.5 Commit du pilote : **`eb80a4f0a`** (vérifications sur pièces : 0 conflit, 0 marqueur,
  0 `ScanFilm*(filmDir` dans `build_from_film.go`, 48 lignes × 13 TSV, build/vet/tests verts).

### Étape B — Véhicules (union D9) — schéma 39
- [x] B.0 D8bis — `git fetch origin` : **`origin/feat/v75` inchangé à `7fb4b60a1`**, ZÉRO commit
  amont depuis l'étape A (`git log --oneline 7fb4b60a1..origin/feat/v75` rend vide). Aucune
  mini-réconciliation. Merge lancé sur cette base.
- [x] B.1 Merge `origin/feat/v75-vehicules-sons` (`1e3d459d1`, 28 commits, base `14a115bb1`) puis
  ajout du second parent `wt/vehicule-deadstate` (`0b5141b8a`) — **387 fichiers, 26 conflits**,
  résolus « sémantique véhicules dans NOTRE architecture ET sur le schéma 38 amont ». Détail
  fichier par fichier au §6. PREUVE : `git diff --name-only --diff-filter=U` VIDE,
  `git grep '^<<<<<<< '` VIDE.
- [x] B.2 Les balayages en nouvelle forme, les CINQ sites de production de `build_vehicles.go`
  compris ; trois enveloppes SANS APPELANT supprimées (N-6) ; quatre enveloppes ajoutées à la
  liste fermée d'archlint (**43 → 47**) ; ratchet des vars de `filmdec` **116 → 118** (deux vars
  de la branche, chacune justifiée) ; UNE étape observée `vehicles` (`BuildFromFilmSteps`
  **34 → 35**). Tableau des migrations au §6. Gardes vertes : `./internal/archlint/` **ok**.
- [x] B.3 `SchemaVersion = 39` (une fois, `document.go`, commentaire : les trois montées 29/30/31
  du chantier fondues en une, posées sur le 38 — décision D3) ; `make openapi-gen` +
  `make generate-types` + `make openapi-check` **exit 0** ; `contracttest` **ok**
  (`wantReplayDocumentFields` 52 → 54).
- [x] B.4 Gates — `gofmt -l .` VIDE · `go build ./...` **EXIT_BUILD=0** · `go vet ./...`
  **EXIT_VET=0** · suites `./internal/analysis/... ./internal/replaybuild/
  ./internal/games/halo_infinite/film/... ./internal/sync/... ./internal/archlint/
  ./internal/api/... ./cmd/...` **EXIT_TESTS=0** (71 paquets `ok`). Web : détail au §6.
- [x] B.5 Harnais — passe 1 SANS `-update` : 0 identique / 13 différents ; **diff PAR NOM** (la
  comparaison est POSITIONNELLE) : `vehicles` NEUVE 13/13, `artifact` bouge 13/13, `killsource`
  bouge **4/13** — les 4 films dont la bande `ti=40` est peuplée, et eux seuls : c'est la
  correction déclarée du lot V9 de la branche (N-8), pas un effet de l'intégration ; AUCUNE autre
  étape ne bouge, aucune ne disparaît. Corrélation MESURÉE : `artifact` dépasse son plancher de
  +450 octets sur EXACTEMENT les 5 films à `vehicles` non vide. Puis `-update` (13 écrits) et
  passe de vérification **13 identiques, 0 différent, 0 écarté, 0 échec, 0 illisible**. Gate de
  lignes : 49 étapes sur les 13 TSV ; `minifilm.tsv` INTACT. Témoin BTB obligatoire : `084a804d`
  (180 vies, 109 478 échantillons). Tableau complet au §5.
- [x] B.6 Commit du pilote — un seul commit de merge à DEUX parents.

### Étape C — Usage de session (`wt/session-usage`)
- [x] C.0 D8bis — `git fetch origin` : `origin/feat/v75` **INCHANGÉE** à `7fb4b60a1`,
  `git rev-list --count HEAD..origin/feat/v75` = **0**. Aucune mini-réconciliation à faire.
  Cible du merge : `wt/session-usage` = `c4f7d5417`, base de merge `da616828f`, **10 commits**
  (conformes au §1).
- [x] C.1 Merge `--no-commit` : **68 fichiers**, **3 conflits** seulement (les crochets de
  `artifacts.go` et `journal.go` ont fusionné SEULS, ainsi que `killcollector`,
  `no_art_patterns_test.go`, `migration/order.go` et `openapi.yaml`). Détail et résolutions au §6.
  PREUVE : `git diff --name-only --diff-filter=U` VIDE, `git grep '^<<<<<<< HEAD$'` VIDE.
  D11 respecté : `openapi.yaml`/`generated.ts` **régénérés**, jamais édités —
  `make openapi-gen` (666 456 o, diff NUL contre l'auto-merge) · `make generate-types` ·
  `make openapi-check` code 0 · `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate` **ok** ·
  `CGO_ENABLED=0 go test ./contracttest/... -run TestContract` **ok** ·
  `node tools/check-generated-types-fresh.mjs` code **0**.
- [x] C.2 Vérifié SUR PIÈCES (preuves au §6) : persister INSERT-only, deux tables append-only
  avec leurs vues `_latest` (`films_latest` créée AVANT `players_latest`), tous les lecteurs de
  production sur les vues, gate par `film.usage_summary` (jamais `slug ==`), aucun
  `OpenReadWrite` neuf hors du régime « serveur arrêté » des backfills CLI.
- [x] C.2bis **NON** — `match_usage_players`/`match_usage_films` n'entrent PAS dans
  `internal/migration/append_only_rebuild.go` (justification N-6 au §4). En revanche l'étape **5**
  de la recette ADR 0026 manquait : les deux tables sont inscrites à `appendOnlyStateTables`
  (`internal/sync/append_only_state_guard_test.go`), sur le patron de `match_kill_events` /
  `match_weapon_shots`. `internal/sync` **ok**.
- [x] C.3 Gates — `gofmt -l .` VIDE (**EXIT_GOFMT=0**) · `go build ./...` **EXIT_BUILD=0** ·
  `go vet ./...` **EXIT_VET=0** · suite Go des 12 périmètres **EXIT_TESTS=0** (101 paquets `ok`,
  0 FAIL) · intégration `-p 1` **EXIT_INTEG=0** (15 paquets `ok`, dont `replayartifacts` avec
  `usage_integration_test.go` et `platform/duckdb` avec
  `session_usage_aggregate_integration_test.go`) · `golangci-lint run` : **336 issues** au total,
  **0 sur un fichier touché par la RÉSOLUTION** (croisement mesuré : les 5 fichiers résolus à la
  main n'apparaissent dans aucune ligne du rapport). MAIS le lint QUI FAIT FOI (ratchet CI
  `--new-from-merge-base=origin/main`) rend **5 issues, toutes dans deux fichiers NEUFS de la
  branche** — détail et attribution en N-10 au §4. Web D12 COMPLET : `npm run typecheck` **0**, `npm run lint` **0**,
  `npm run lint:fields` **0**, `npm run test` **0** (574 fichiers, 5 991 tests passés,
  14 skippés), `npm run build` **0**, `node tools/knip-ratchet.mjs` **0** (0/0/0 au plafond 0).
  Seuils : `usageLogic.ts` **486 L**, `SessionUsageSection.tsx` **417 L** — sous 500, passés tels
  quels. Aucun fichier n'a franchi 500 L par la résolution (mesuré HEAD vs merge, §4 N-7).
- [x] C.4 Harnais — passe SANS `-update` : **13 identiques, 0 différent** (verdict complet au §5).
  Aucun `-update`, aucun refigeage.
- [x] C.5 Commit du merge dans `wt/integ-usage`.

### Étape D — Game-changers (`wt/game-changers`, web)
- [x] D.0 `git fetch origin` — **0 commit** entre `eb80a4f0a` et `origin/feat/v75` (amont
  toujours gelé, mesuré). `git merge --no-ff --no-commit wt/game-changers` — base de merge
  **`ca55f0ed7`** (celle du §1), **4 commits**, **16 fichiers** (`.ai/thought_log.md`,
  `.ai/PLAN_REPLI_GAME_CHANGERS_2026-09-05.md` neuf, 3 `components/ui/collapsed-items-toggle*`
  neufs, 12 `features/match-replay`). Détail au §6.
- [x] D.1 Merge automatique SANS AUCUN CONFLIT (`git status` : « All conflicts fixed » —
  aucun marqueur `<<<<<<<`, `git grep` VIDE) : les croisements annoncés avec B/C
  (`i18n.ts`, `i18nContract.ts`, `MatchEquipmentUsageSection.tsx`) ne se sont pas manifestés
  ici, B et C n'étant pas encore mergées dans cette branche — laissés au pilote à la fusion
  finale, comme prescrit. `.ai/thought_log.md` fusionné seul par git en UNION (nos entrées en
  tête à la ligne 1, les leurs à leur ancre d'origine plus bas, aucune perdue des deux côtés).
- [x] D.2 Gates web D12 complets, tous verts : `npm run typecheck` **exit 0** ·
  `npm run lint` **exit 0** (23 warnings préexistants, 0 erreur, aucun sur un fichier touché) ·
  `npm run lint:fields` **exit 0** (220 labels FR+EN, 1643 fichiers scannés, 0 violation) ·
  `npm run test` **exit 0** (577 fichiers / 6008 tests passés, 14 skipped, 0 échec) ·
  `npm run build` **exit 0** (avertissements de taille de chunk préexistants) ·
  `node tools/knip-ratchet.mjs` **exit 0** (files 0/0, exports 0/0, types 0/0 — le nouveau
  `collapsed-items-toggle.tsx` est consommé par `equipmentUsageColumns.ts` /
  `MatchEquipmentUsageSection.tsx` / `MatchPadControlSection.tsx`, aucun export mort neuf).
  PREUVE harnais [~] : `git diff --stat eb80a4f0a -- apps/go-api` **VIDE** — rien côté Go,
  harnais non rejoué (étape déclarée sans changement attendu, prouvé sur pièces).
- [x] D.3 Commit du pilote : merge commit sur `wt/integ-gamechangers`, SHA au §6.

### Étape G — Stats d'Assaut : REPRISE du chantier `wt/assaut-stats` et mise en service (D13)
Exécutée APRÈS les trois fusions D10 (elle crochète la même zone que C : `replayartifacts`,
`persist`), AVANT E. Plan de référence : `.ai/V7.5/PLAN_ASSAUT_STATS_2026-09-04.md` (sur la
branche, il fait foi pour les décisions §2 : table dédiée `match_bomb_stats`, append-only +
`_latest`, INSERT-only via `BatchBuilder`, événements datés dans `match_objective_events`,
capability `film.bomb_stats`, désamorçage HORS LOT, aucune cuisson en lot).
- [ ] G.0 Sécuriser le travail NON COMMITÉ de l'autre session dans `LevelUp-wt-assaut-stats`
  (29 fichiers modifiés + 8 non suivis = lots E2-bis, E2-ter, E3 « Complétés » au journal mais
  jamais commités) : vérification sur pièces (build, vet, tests) puis commit SUR `wt/assaut-stats`
  au nom du lot, sans rien y changer.
- [ ] G.1 Merge `wt/assaut-stats` dans `wt/cuisson-perf` (fusionnée B+C+D) ; conflits attendus :
  `document.go` (champs armements/portage sur le 39), `persist/{batch,builder,combined_persister}.go`
  (C a ajouté le persister d'usage au même endroit), `no_art_patterns_test.go` et
  `append_only_state_guard_test.go` (deux durcissements datés : union), `migration/order.go`,
  `replaybuild/{matchfacts,zones}.go`, journaux. Harnais : écart déclaré si le document bouge
  (armements pausables, `assembly_000d5950.golden` a bougé sur la branche) — corrélé aux films
  d'Assaut du corpus, sinon anomalie.
- [ ] G.2 E4 — branchement au sync : dans `replayartifacts`, projeter les stats bombe du
  DOCUMENT RANGÉ (patron EXACT de `usage.go` arrivé par C : projection après rangement, lecture
  du fichier rangé, writer court après toute cuisson, via `persist`), gate par capability
  `film.bomb_stats` (TOML du titre), dégradation gracieuse (film absent, mode non-Assaut,
  capability absente → rien d'écrit, `slog.InfoContext` qui dit pourquoi), aucune erreur avalée,
  `maxPerCycle` et verrou inchangés. Test du crochet + intégration `-p 1`.
- [ ] G.3 E5 — lecture et API : repo `platform/duckdb` sur `match_bomb_stats_latest`, port, service
  (zéro SQL inline), handler Huma (zéro logique métier), capability → `ErrCapabilityNotSupported`
  en réponse partielle propre ; `make openapi-gen && make generate-types && make openapi-check`
  (D11).
- [ ] G.4 E5 — web : affichage dans la fiche de match sur le patron des autres modes à objectif
  (section objectifs, vues arrivées par l'amont), `useFieldLabel()`, i18n FR **et** EN, query key
  dans `lib/query/keys.ts`, tokens de couleur seulement ; D12 complet.
- [ ] G.5 E6 — backfill : `levelup backfill-replay` suffit-il (le crochet projette tout artefact
  cuit) ? sinon sous-commande dédiée sur le patron de `backfill-usage-summary` (C). JAMAIS lancé
  sur le parc. Registre des reports (désamorçage, condition « corpus portant un désamorçage
  avéré ») ; cases du plan Assaut statuées ; thought_log ; commit.

### Étape E — Filet complet et revue de branche (sur `wt/cuisson-perf` fusionnée)
- [ ] E.1 FILET COMPLET, pièce par pièce, un code de sortie consigné par ligne (EXIT_*=0 dans un log
  persistant, jamais un pipe) : `make gate-push` (son ratchet lint compare à `origin/main` : tout le
  delta v75 s'affiche, ce n'est pas une régression) · `gofmt -l .` vide · `go vet ./...` · `go build
  ./...` · `go test ./... -count=1` · `go test -tags=integration -p 1 ./internal/sync/... -count=1` ·
  `make openapi-check` · D12 complet · harnais 13/13 identiques SANS `-update` (49 lignes).
- [ ] E.2 Revue adversariale du diff d'intégration (`eb80a4f0a..HEAD`, contexte frais) ;
  corrections ; seconde lecture.
- [ ] E.3 Journal, registre des reports (D5, D6, D7), thought_log ; suppression des trois worktrees
  et branches d'intégration ; commit.

### Étape F — Merge, push, CI
- [ ] F.1 `git fetch` ; D3(b) numéro de schéma amont ; si `origin/feat/v75` ≠ `7fb4b60a1` :
  mini-réconciliation (protocole A) + harnais + gates ; commit.
- [ ] F.2 Worktree partagé propre → `git merge --no-ff wt/cuisson-perf` dans `feat/v75` ; sinon
  attendre/signaler.
- [ ] F.3 `git push origin feat/v75` ; suivi `gh run watch` ; tout job rouge (même préexistant)
  se diagnostique et se corrige ; re-push jusqu'au vert au niveau job sur la liste mesurée (D8).
- [ ] F.4 Rapport à l'utilisateur : ce qui est intégré, les écarts nommés, D5/D6/D7/D8ter, la
  proposition de re-cuisson de masse (jamais lancée d'office), la note sur `main`, le sort des
  branches absorbées (`origin/feat/v75-vehicules-sons` est publiée : proposer sa suppression ;
  `wt/cuisson-perf` poussée comme trace ou non — décision utilisateur).

## §4 Hors périmètre / découvertes
- N-1 `wt/assaut-stats` : D5 (+ nuance `bombe_portage_gate_test.go`).
- N-2 champ `structure` : D6 (quatre sites web).
- N-3 mmap himodule : D7.
- N-4 `main` a 9 commits hors `feat/v75` (hotfix 7.3.1) — à ramener dans feat/v75 un jour ; hors
  mandat.
- N-5 les trois outils `cmd/{weapon-sounds,vs-measure,vehicle-sprite}` de la branche véhicules
  arrivent tels quels ; à statuer par leur auteur (outillage de recherche vs production).
- **N-6 (étape C, C.2bis) — `match_usage_players` / `match_usage_films` N'ENTRENT PAS dans
  `internal/migration/append_only_rebuild.go`, et c'est la recette elle-même qui le dit.** Le
  helper est un **SWAP DE CONVERSION** (`legacy → append-only`) : son en-tête l'écrit
  (« factorise le SWAP de conversion, commun à 7 tables »), et `applyAppendOnlyRebuild` rend
  **no-op sur une table absente** (étape 2 de son contrat : « table absente → no-op ») — il ne
  CRÉE jamais rien. Les deux tables du chantier sont NET-NEUVES, écrites append-only dès leur
  DDL (PK technique `id` sur séquence, `summary_pass`/`summary_rev`/`written_at`, vues
  `_latest`) : il n'y a aucun héritage à convertir, donc rien à passer au helper. Le dépôt a
  déjà quatre précédents exactement de cette forme, tous hors du helper :
  `match_objective_stats`, `match_kill_events`, `match_weapon_shots` et
  `match_weapon_hit_distance`. Ce qui manquait de la recette ADR 0026, c'est son **étape 5**
  (inscription au garde-rail `internal/sync/append_only_state_guard_test.go`) — FAITE en C.2bis.
- **N-7 (étape C) — quatre fichiers Go touchés dépassent 500 L, TOUS déjà au-dessus avant le
  merge** (mesuré `git show HEAD:<f> | wc -l` contre l'arbre) : `cmd_backfill_killsource.go`
  516 → **505** (la branche le RACCOURCIT), `games/halo_infinite/adapter_data.go` 501 → 506,
  `service/session_page_service.go` 844 → 856, `sync/no_art_patterns_test.go` 542 → 550. Dette
  gelée par la baseline lint, aucune résolution ne franchit le seuil : rien à traiter ici.
- **N-8 (étape C) — `match_weapon_hit_distance` n'est PAS inscrite à `appendOnlyStateTables`**
  alors que `match_kill_events` et `match_weapon_shots`, ses sœurs du même chantier, le sont
  (`internal/sync/append_only_state_guard_test.go`). Trou PRÉEXISTANT à l'étape C (la table
  arrive de la branche précision, remise le 2026-09-01), hors périmètre : consigné, non traité.
- **N-9 (étape C) — le ratchet D-3 de l'ADR 0030 (allowlist datée des sites `OpenReadWrite`)
  N'EXISTE PAS dans le code.** L'ADR est « Accepted » et le décrit (§D-3, §Implémentation
  point 2), mais aucun test n'énumère les appelants : grep exhaustif du dépôt, aucun fichier de
  garde. La vérification C.2 « aucun `OpenReadWrite` hors allowlist » a donc été faite À LA
  MAIN : la branche ajoute **un seul** site de production, `cmd/levelup/cmd_backfill_usage_summary.go:103`,
  CLI hors ligne sous précondition « serveur arrêté » écrite en tête de fichier — même régime
  que `cmd_backfill_killsource.go` qui le précède. Le second ajout est un `_test.go`
  d'intégration. Hors périmètre : poser le ratchet est un chantier ADR 0030 à part entière.
- **N-10 (étape C) — CINQ ISSUES DE LINT QUI ROUGIRONT LE JOB CI `Go Lint`.** Le lint qui fait
  foi est `golangci-lint run --new-from-merge-base=origin/main` (Makefile `go-api-lint`, job CI
  `go-lint`) : la dette gelée reste invisible, seules les issues AJOUTÉES rougissent. Mesuré sur
  l'arbre de merge, il rend **5** issues (`EXIT_LINT_RATCHET=1`), toutes dans **deux fichiers
  NET-NEUFS apportés par `wt/session-usage`**, aucune dans un fichier résolu à la main :
  - `internal/analysis/replay/usage_summary_families.go` : `grenade_frag` ×17, `grapple` ×8
    (« constante `eqUsesGrappleFamily` existe déjà »), `powerup_camo` ×4, `wall` ×11 (goconst) ;
  - `internal/analysis/sessionusage/usage.go:179` : préallocation de `keys` (prealloc).

  NON TRAITÉ à l'étape C, à dessein : le mandat de l'étape est la résolution du merge, pas la
  réécriture du code de la branche (aucun fix hors périmètre). Ce sont des corrections pures
  (extraction de constantes, capacité de slice) sans effet sur la sortie. **Condition de
  reprise : étape E** (E.1 `make gate-push`), qui ne peut pas passer avec ces cinq rouges — et
  D8 exige le vert au niveau job avant le push de F.3.

  **ÉTEINTES le 2026-09-05, commit `983b696b1`** (worktree `wt/cuisson-perf`). Une constante
  nommée par clé de famille répétée (`usageFamily*` dans `usage_summary_families.go`) ; dans
  `metricKeys` (`sessionusage/usage.go`), calcul des familles déployées avant l'allocation de
  `keys` (`make` + `append` au lieu du slice littéral qui grossissait par `append`). Piège
  mesuré en cours de résolution : `--uniq-by-line` masquait DEUX occurrences goconst
  supplémentaires (`thruster` ×13, `powerup_overshield` ×7) tant que `grapple`/`powerup_camo`
  restaient en littéral sur la même ligne physique — apparues au relevé cache-nettoyé une fois
  les quatre premières corrigées, mêmes constantes, même traitement. Gate final mesuré : `gofmt
  -l .` vide, `go vet` propre, `go test ./internal/analysis/replay/ ./internal/analysis/sessionusage/`
  vert, `golangci-lint run --new-from-merge-base=origin/main` (cache nettoyé) **0 issue**.
- N-11 la comparaison du harnais est positionnelle (§5 A) : une amélioration « diff par nom »
  de `cmd/replay-equiv` serait pérenne — hors périmètre de l'intégration, au registre.
- N-12 **Trois entrées de balayage du chantier véhicules arrivaient SANS AUCUN APPELANT** (tests
  compris) : `ScanFilmVehicleCreationsForBand`, `ScanFilmKeyframeRecordSpans` et
  `ScanFilmVehicleOccupancy` — cette dernière avec sa forme film, dont elle était l'unique
  consommateur. Elles ont été **supprimées** à l'étape B (règle D2 « une enveloppe sans appelant
  est supprimée » + règle 0 du dépôt), et NON inscrites à la liste fermée d'archlint. Rien de la
  MESURE n'est perdu : les moitiés pures et testées restent (`KeyframeRecordSpans`,
  `VehicleKeyframeStates`, `FindKeyframeBlockInsertion`), et l'en-tête de chaque fichier dit ce
  qui a été retiré et pourquoi. À signaler à l'auteur du chantier.
- N-13 `apps/web/.../ReplayCanvas.tsx` est **exactement à son plafond** (664 lignes pour 665) après
  le câblage véhicules. La prochaine addition devra être payée par une extraction, comme les dix-sept
  précédentes.
- N-14 **L'ÉCART `killsource` de l'étape B est une CORRECTION DÉCLARÉE DE LA BRANCHE, pas une
  régression de l'intégration** : `filmdec/traverse.go` routait les deux composants
  `-dynamic-precision-` d'orientation de `ti=38/39/40/43` vers les désérialiseurs du BIPÈDE ; le
  lot V9 du chantier (Ghidra, validé 6/6 contre la table ECS live) les remet sur les leurs. Le
  digest `killsource` bouge donc sur les 4 films du corpus dont la bande `ti=40` est peuplée, et
  sur eux seuls. Chiffres et corrélation au §5. **À signaler à l'utilisateur** : le kill-feed des
  matchs à véhicules change avec cette livraison, dans le sens de la correction.

## §5 Diffs des comptes (par étape)

### Étape A — réconciliation amont `7fb4b60a1` (2026-09-05)

**LA COMPARAISON DU HARNAIS EST POSITIONNELLE** (`cmd/replay-equiv/parent.go`, `comparer` : ligne
`i` contre ligne `i`), et l'étape `translocations` s'insère en **tête** de `BuildFromFilmSteps`
(le balayage tourne AVANT les positions, parce qu'il arme l'exemption du filtre de vitesse). La
passe 1 nomme donc « première étape en écart : `translocations` » sur les 13 films — un artefact
de décalage, pas un diagnostic. Le diff exploitable est celui **PAR NOM D'ÉTAPE**, entre la copie
des 13 TSV de référence prise avant refigeage et les TSV refigés. Il ne fait bouger que **six
noms**, et **aucune étape ne disparaît** :

| étape | films touchés | nature | cause amont |
|---|---|---|---|
| `translocations` | **13/13** | NOUVELLE | `filmdec/transloc_events.go` — l'événement 117 `EquipmentTranslocatorTeleportEffects`. Contenu NON VIDE sur 2 films seulement : `64e8adfa` (4 événements), `60ae07c4` (2) ; 0 sur les 11 autres |
| `abilityImpulses` | **13/13** | NOUVELLE | `filmdec/ability_impulses.go` — corps `tag==1` d'i57/i59 (usage du propulseur). Non vide sur 6 films : `000d5950` 86, `1c4c63c2` 42, `084a804d` 8, `64e8adfa` 7, `7344d24f` 2, `53ce4390` 1 |
| `abilityCharges` | **13/13** | NOUVELLE | `filmdec/ability_charges.go` — emplacements armés d'i56. Non vide sur 9 films (`084a804d` 148, `000d5950` 91, `1c4c63c2` 86, `53ce4390` 83, `a349fea8` 73, `64e8adfa` 35, `696a9d7c` 28, `7344d24f` 17, `01e1f945` 14) |
| `equipmentChanges` + `.stats` | **13/13** | CHANGE | RÉCUPÉRATION GATÉE des émissions i48 manquées (`filmdec/equipment_recovery.go`, neuf de l'amont) : `EquipmentChange` gagne `Recovered` et `Gap`, `EquipmentChangeStats` gagne `Recovered`. Le compte monte sur 8 films (`084a804d` et `a349fea8` +9, `000d5950` et `53ce4390` +2, quatre autres +1) et reste identique sur 5 — sur ces 5 le sha bouge quand même, **par les deux champs ajoutés au struct**, pas par une lecture qui change |
| `positions` | **2/13** | CHANGE | EXEMPTION DU FILTRE DE VITESSE par les téléportations (décision D2 amont) : `60ae07c4` 267 365 → 267 371 (+6), `64e8adfa` 293 811 → 293 823 (+12) |
| `artifact` | **13/13** | CHANGE (octets) | schéma **37 → 38** + les trois calques neufs et leurs couvertures. +364 à +7 769 octets |

**CORRÉLATION MESURÉE, PAS DÉDUITE** — c'est elle qui autorise le refigeage :

1. `positions` bouge sur **EXACTEMENT** les 2 films dont `translocations > 0`, et sur **AUCUN**
   des 11 films à `translocations == 0`. 13/13, zéro exception. Et le delta suit le nombre
   d'événements : 2 téléportations → +6 positions, 4 → +12 (3 enregistrements récupérés par
   téléportation, le même rapport sur les deux films). C'est l'exemption, et rien d'autre :
   sur un film sans tête 117 le filtre est bit à bit celui d'avant, ce que les 11 autres prouvent.
2. Les zéros des trois calques neufs sont des zéros **MESURÉS**, pas des balayages qui n'ont pas
   tourné : le journal des 13 films rend `composantAbsent=false` sur les 26 lignes
   (13 × impulsions + 13 × charges) — le composant est déclaré partout, il n'y a simplement eu ni
   impulsion ni charge sur ces matchs-là.
3. `artifact` : le plancher est **+364 octets**, et il tombe sur exactement les 3 films qui ont
   0 translocation, 0 impulsion ET 0 charge (`51101d1d`, `9f57c612`, `d9781168`) — c'est le coût
   fixe des calques vides, du numéro de schéma et des blocs de couverture. Les autres montent en
   proportion de ce qu'ils portent.

**AUCUN AUTRE DIGEST NE BOUGE** : `fire`, `loadouts`, `pickups`, `inventory`, `inventoryDeltas`,
`abilityRanks`, `camoStates`, `grappleReads`, `zoomEvents`, `placements`, `pads`, `carrierMarks`,
`zoneReads`, `bombReads`, `grenades`, `projectiles`, `deaths`, `playerIndices`, `clockOrigin`,
`score`, `objectives`, `flag`, `zones`, `killSource`, `neutralDeaths`, `spawnPoints` et les six
canaux delta sont **IDENTIQUES AU BIT PRÈS** sur les 13 films — la preuve que l'architecture du
chantier a traversé le merge sans dommage.

**GATE DE LIGNES (IMP-3 de la revue) : les 13 TSV portent 48 étapes, tous** (45 + 3), et
`minifilm.tsv` ses 7. Une ligne AJOUTÉE n'est pas un écart ; une ligne MANQUANTE le serait — et
`observe_test.go` rougirait avant le harnais.

**MINI-BOBINE (étage CI) : AUCUN REFIGEAGE.** `TestEquivalenceMiniFilm` ne couvre que sept étapes
(`fire`, `grenades`, `loadouts`, `inventory`, `deaths`, `playerIndices`, `projectiles`) : aucune ne
bouge, contrairement au 2026-09-03 où `fire` avait changé. Le test passe sur la référence existante.

**DURÉES ET PICS (passe de vérification)** : 5,1 s (`51101d1d`) à 1 min 58,7 s (`1c4c63c2`) ;
pics 0,10 à 0,66 Gio. Témoins : `01e1f945` **17,9 s**, `7344d24f` **21,0 s**, `696a9d7c`
**20,5 s** — le gain du chantier est intact après le merge (cible du plan : < 100 s).


### Étape B — véhicules (union D9), schéma 39 (2026-09-05)

**LE DIFF EXPLOITABLE EST CELUI PAR NOM D'ÉTAPE** (la comparaison du harnais est POSITIONNELLE, et
`vehicles` s'insère APRÈS `pads`, donc au milieu du fichier) : copie des 13 TSV prise avant
refigeage, comparée nom par nom aux TSV refigés. **AUCUNE étape ne disparaît**, et seuls **TROIS
noms** bougent :

| étape | films touchés | nature | cause |
|---|---|---|---|
| `vehicles` | **13/13** | NOUVELLE | le calque `ti=40` du chantier. Contenu VIDE (sha `f73d0cef…`, `VehicleScan{}`) sur 8 films sans aucun slot `ti=40` aux images-clés ; contenu propre sur 5 : `084a804d` (180 vies, 109 478 échantillons, 78 événements, 64 297 visées), `a349fea8`, `1c4c63c2` (67 vies), `53ce4390` (26 vies) et `60ae07c4` (4 slots recensés, **0 échantillon** — bande présente, entités muettes) |
| `artifact` | **13/13** | CHANGE (octets) | schéma **38 → 39** + le calque véhicules. **Plancher mesuré : +450 octets**, et il tombe sur les 8 films à `vehicles` VIDE (`000d5950`, `01e1f945`, `51101d1d`, `64e8adfa`, `696a9d7c`, `7344d24f`, `9f57c612`, `d9781168`) — le coût fixe du numéro de schéma et du calque vide. `60ae07c4` fait **+452** : deux octets de plus, exactement le bloc de couverture « balayé, rien trouvé ». Les 4 films à véhicules montent de **+301 553** (`53ce4390`) à **+1 980 061** (`084a804d`) |
| `killsource` | **4/13** | CHANGE | **ÉCART HORS DE L'ATTENDU, MESURÉ ET EXPLIQUÉ** — voir ci-dessous |

**CORRÉLATION MESURÉE, PAS DÉDUITE** : `artifact` dépasse le plancher sur **EXACTEMENT** les
5 films dont `vehicles` n'est pas vide, et sur **AUCUN** des 8 autres. 13/13, zéro exception. Le
sha de `vehicles` est LE MÊME (`f73d0cef…`) sur les 8 films vides et DISTINCT sur les 5 autres :
le zéro est un zéro MESURÉ (le balayage a tourné et n'a rien trouvé), pas un balayage qui n'aurait
pas tourné.

**L'ÉCART `killsource`, ET POURQUOI IL N'EST PAS UNE RÉGRESSION DE L'INTÉGRATION.** Il tombe sur
`084a804d`, `1c4c63c2`, `53ce4390` et `a349fea8` — les **QUATRE films dont la bande `ti=40` est
peuplée**, et sur aucun autre (`60ae07c4`, dont la bande existe mais dont les entités ne
répliquent pas, ne bouge pas). Sa cause est une **CORRECTION DÉCLARÉE DE LA BRANCHE**, mesurée par
elle et arrivée avec elle (`filmdec/traverse.go`, lot V9 du 2026-09-03) : les deux composants
`-dynamic-precision-` d'orientation de `ti=38/39/40/43` étaient routés vers les désérialiseurs du
BIPÈDE, ce qui amputait `i2` de ses bits de tête et `i3` de son gate externe. Les vrais
désérialiseurs (`FUN_140c5f7ec` / `FUN_140d87740`) ont été résolus statiquement sous Ghidra et
validés 6/6 contre la table ECS live (`components_dynprec_orientation.go`). Le marcheur de records
que `killsource` emprunte (`consumeByName`) traverse donc ces deux composants autrement — et
seulement là où des entités `ti=38/39/40/43` en émettent. Les deux autres fichiers du chemin
`killsource` touchés par la branche (`frame_records.go`, `components_object.go`) sont **neutres au
bit près** (plomberie de sonde et extraction d'un décodeur : vérifié sur pièces).

**LES QUATRE FILMS À VÉHICULES, CHIFFRÉS** :

| film | variante | vies `ti=40` | échantillons | `artifact` | `killsource` |
|---|---|---|---|---|---|
| `084a804d` | BTB Heavies : CTF | 180 | 109 478 | 7 275 954 → **9 256 015** (+1 980 061) | bouge |
| `a349fea8` | BTB Heavies : Total Control | — | — | 6 897 326 → **8 709 326** (+1 812 000) | bouge |
| `1c4c63c2` | BTB : One Flag CTF | 67 | 111 822 | 7 433 969 → **8 301 577** (+867 608) | bouge |
| `53ce4390` | CTF : Arena (Behemoth) | 26 | 28 072 | 2 696 470 → **2 998 023** (+301 553) | bouge |
| `60ae07c4` | Ranked : Oddball (Live Fire) | 4 recensées, **0** échantillon | 0 | 3 035 677 → **3 036 129** (+452) | IDENTIQUE |

**AUCUN AUTRE DIGEST NE BOUGE** : `score`, `objectives`, `vip`, `skull`, `bomb`, `flag`, `zones`,
`zoneRoles`, `spawnPoints`, `spawnPointsState`, `neutralDeaths`, `killRefs`, `translocations`,
`positions`, `fire`, `loadouts`, `heldWeaponChanges`, `pickups`, `inventory`, `inventoryDeltas`,
`abilityRanks`, `equipmentChanges`, `camoStates`, `grappleReads`, `abilityImpulses`,
`abilityCharges`, `zoomEvents`, `placements`, `pads`, `carrierMarks`, `zoneReads`, `bombReads`,
`grenades`, `projectiles`, `deaths`, `playerIndices`, `clockOrigin` et les six canaux `.stats` sont
**IDENTIQUES AU BIT PRÈS** sur les 13 films. En particulier `positions` ne bouge pas : la
correction `i2`/`i3` dyn.-préc. ne touche que le marcheur ECS de `killsource`, jamais le nuage
bipède (qui lit `i0` AVANT `i2`).

**GATE DE LIGNES : les 13 TSV portent 49 étapes, tous** (48 + `vehicles`), et `minifilm.tsv` ses 7.

**MINI-BOBINE (étage CI) : AUCUN REFIGEAGE.** `TestEquivalenceMiniFilm` ne couvre que sept étapes
(`fire`, `grenades`, `loadouts`, `inventory`, `deaths`, `playerIndices`, `projectiles`) : aucune ne
bouge.

**PASSES ET DURÉES.** Passe 1 SANS `-update` : **0 identique / 13 différents / 0 écarté / 0 échec /
0 illisible** ; passe `-update` : 13 écrits, BILAN **13 identiques** ; passe de VÉRIFICATION sans
`-update` : **13 identiques, 0 différent, 0 écarté, 0 échec, 0 illisible** (exit 0). Témoins de la
passe de vérification : `01e1f945` **18,2 s**, `7344d24f` **22,2 s**, `696a9d7c` **22,9 s** —
cible < 100 s tenue (le gain du chantier survit au merge). Pics 0,08 à 0,67 Gio ; le plus long est
`084a804d` à 2 min 35,6 s, le BTB à 180 véhicules.

### Étape C — usage de session `wt/session-usage` `c4f7d5417` (2026-09-05)

**AUCUN REFIGEAGE, ET C'EST LE RÉSULTAT ATTENDU (D4, item C.4).** Passe SANS `-update` :

```
BILAN : 13 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s) (harnais)
```

Les treize, nommément : `000d5950`, `01e1f945`, `64e8adfa`, `7344d24f`, `696a9d7c`, `084a804d`,
`1c4c63c2`, `53ce4390`, `d9781168`, `9f57c612`, `60ae07c4`, `51101d1d`, `a349fea8` — tous
« identique ». `git status` sur `internal/analysis/replay/testdata/` rend VIDE : pas un octet
de référence n'a bougé.

**POURQUOI ZÉRO ÉCART ÉTAIT PRÉVISIBLE, ET CE QUI LE PROUVE.** La branche ne touche que deux
fichiers du chemin de cuisson, et le diff y est un **RENOMMAGE PUR** :
`padFamilyKey` -> `PadWeaponFamilyKey` (`pad_pickup_dating.go` : la signature, deux appels dans
`datePadPickups`, un bloc de commentaire ajouté ; `document_pickups.go` : une seule ligne de
commentaire citant l'ancien nom). Le corps de la fonction est inchangé au caractère près, et le
seul appelant de production reste `datePadPickups`. Le résumé d'usage lui-même
(`replay/usage_summary.go`) est **HORS ARTEFACT** : `import "sort"` et rien d'autre,
`BuildUsageSummary(doc)` est une fonction pure document -> lignes, appelée depuis
`replayartifacts`, jamais depuis `BuildFromFilm`. Aucune étape observée n'est ajoutée :
`BuildFromFilmSteps` reste à **34** et les 13 TSV gardent leurs 48 étapes.

**DURÉES ET PICS** (passe unique, à titre de veille — le gain du chantier tient) : `01e1f945`
**25,1 s** / 0,21 Gio, `7344d24f` **26,8 s** / 0,22 Gio, `696a9d7c` **26,3 s** / 0,22 Gio ;
plancher `51101d1d` 7,5 s / 0,08 Gio, plafond `a349fea8` 4 min 38,9 s / 0,51 Gio (film-bombe du
lot 4b, régime connu).

### Fusions D10 — (attendu : 13/13 identiques sans `-update` sur les TSV de B, après chaque fusion)

- Fusion D (`1c1b6026f`, 15 h 55) : zéro octet Go, plan en union ; gates web déjà rendus dans le worktree D sur le même contenu.
- Fusion C (`9423b9ba4`, 16 h 54) : conflits documents seuls ; typecheck web exit 0, `go build` OK, `go test` replayartifacts / replay / persist / archlint ok, **harnais 13/13 identiques sans `-update`** (les TSV sont encore ceux de A : B n'est pas fusionnée).

## §6 Journal
- 2026-09-05 14 h 40 — plan écrit après inventaire mesuré (§1) ; relecture `plan-review` lancée
  en parallèle de l'étape A (geste déjà rodé le 03/09).
- 2026-09-05 — **ÉTAPE A exécutée (A.1 à A.4), rien de commité** (mandat du pilote : l'arbre est
  résolu, l'index prêt). Cible `7fb4b60a1`, 324 fichiers, 5 conflits.

  **Les 5 conflits et leur résolution** (règle : la SÉMANTIQUE amont est préservée intégralement,
  dans NOTRE architecture) :

  | conflit | ce que l'amont voulait | résolution |
  |---|---|---|
  | `analysis/replay/build.go` | ajouter 5 champs à `Options` (`AbilityImpulses`/`Stats`, `AbilityCharges`/`Stats`, `Translocations`) et 3 balayages à `BuildFromFilm` (l.308/457/473) | Le bloc en conflit EST celui que le lot 1 a sorti de `build.go`. Vérifié sur pièces (diff bloc-à-bloc base `ceabaad67` contre amont) : l'amont n'y touche QUE ces deux choses. Bloc supprimé chez nous ; les 5 champs REJOUÉS dans `options.go` au même rang (commentaires amont inclus), les 3 balayages dans `build_from_film.go`. Les hunks amont de `BuildFromPositions` (calques `translocations`, `abilityImpulses`, `abilityCharges` + leurs couvertures gatées par `Scanned`) sont hors du bloc déplacé et ont fusionné seuls |
  | `filmdec/ability_rank.go` | extraire `abilityScanSetup` + `resolveAbilityScan(dir)` + `walkAbilityEmissionsWith(s, …)` pour que `ScanFilmEquipmentChanges` résolve le film UNE fois pour ses DEUX passes (strict, puis récupération gatée) ; scinder `walkComponentsAt` hors de `walkRecordComponents` | L'intention amont — résoudre une fois, partager — EST celle de notre `FilmContext`. La structure amont est reprise TELLE QUELLE, son champ `dir string` devenant `fc *FilmContext` et `slots map[uint32]bool` devenant `SlotBand` ; `resolveAbilityScan(fc)` lit `fc.ChunkNumbers/BipedSlots/I0Layout/bipedArchetype` (messages d'erreur du lot 1 conservés) ; `ReadFilmChunk(s.dir, c)` → `s.fc.ChunkAt(c)`. `walkComponentsAt` pris tel quel. La résolution unique amont est donc conservée ET élargie : le contexte la partage aussi avec les balayages hors de ce setup |
  | `filmdec/equipment_changes.go` | remplacer l'accumulation en ligne par DEUX PASSES (strict + récupération) et une fusion pure `assembleEquipmentChanges` | Corps amont pris INTÉGRALEMENT ; seule la tête change : `ScanEquipmentChanges(fc, bornAt)` fait `resolveAbilityScan(fc)`, et `ScanFilmEquipmentChanges(dir, …)` redevient l'enveloppe D2 (`LoadDir` + `NewFilmContext`). L'import `sort` amont fusionné avec notre import `filmsource` |
  | `.ai/thought_log.md` | deux blocs insérés aux mêmes ancres | UNION : nos entrées en tête, puis celles de l'amont — aucune perdue des deux côtés ; ligne de séparation rétablie |
  | `.ai/V7.5/REGISTRE_REPORTS.md` | une ligne de report neuve (`TestBancCliffhanger` rouge préexistant du corpus `gamefiles`) | UNION du tableau : notre ligne `hits.go` puis la sienne |

  **TROIS BALAYAGES MIGRÉS (D2)**, avec leur enveloppe conservée et son unique raison d'être :

  | ancien (production) | nouveau | enveloppe `(dir)` gardée pour |
  |---|---|---|
  | `ScanFilmTranslocatorTeleports(dir, entry)` — `build.go:308` | `ScanTranslocatorTeleports(film, entry)` — `transloc_events.go`, patron de `ScanZoomEvents` (`FilmChunkNumbers`/`FilmChunkAt`) | `replay/golden_inputs_test.go`, `filmdec/transloc_exemption_film_test.go`, `filmdec/transloc_positions_film_test.go` |
  | `ScanFilmAbilityImpulses(dir)` — `build.go:457` | `ScanAbilityImpulses(fc)` — `ability_impulses.go` | `replay/golden_inputs_test.go` |
  | `ScanFilmAbilityCharges(dir)` — `build.go:473` | `ScanAbilityCharges(fc)` — `ability_charges.go` | `replay/golden_inputs_test.go` |

  Le quatrième point d'entrée du même setup, `scanEquipmentRecovery`, ne lit plus le disque non
  plus : sa borne aux chunks des fenêtres est conservée (elle borne le TRAVAIL, marche bit à bit,
  pas seulement l'E/S) mais elle relit `s.fc.ChunkAt(c)`, en mémoire.

  **GARDES TOUCHÉES** :
  - `archlint/no_film_reread_test.go` (règle 3) — `enveloppesInterditesEnProduction` **40 → 43** :
    les trois enveloppes ci-dessus, avec la justification datée et la vérification d'homonymie
    exigée par le fichier (aucune méthode ne porte ces noms). L'allowlist
    `appelsDEnveloppeAutorises` (2 entrées `hits.go`, amont du 03/09) est **inchangée** et reste
    vivante — le test la vérifie dans les deux sens.
  - `replay/observe.go` — `BuildFromFilmSteps` **31 → 34** (`translocations` en tête,
    `abilityImpulses` et `abilityCharges` après `grappleReads`). Pas d'étape `.stats` pour les deux
    dernières : leurs témoins `Absent`/`Scanned` voyagent dans `Options` et sont couverts par le
    digest `artifact`, comme `pads`/`carrierMarks`/`zoneReads`/`bombReads`.
  - `filmdec/equipment_state.go` — commentaire de `EquipmentArchetypeDir` : « la seule des 40
    enveloppes » → « 40 au lot 6, 43 depuis le 2026-09-05 » (doc tenue à jour dans le commit qui
    la périme).
  - **AUCUNE allowlist élargie.** `archlint/filmdec_package_vars_test.go` : le ratchet reste à
    **116** — MESURÉ, l'amont n'ajoute AUCUNE variable de paquet à `filmdec` (le test passe sans
    Logf « a baissé », donc le compte est exactement 116). `no_recomputed_film_context_test.go`,
    `filmsource_leaf_test.go`, `no_rewritten_slot_band_test.go`, `no_unbounded_film_loop_test.go` :
    verts sans modification.

  **CINQ RUPTURES DE COMPILATION LAISSÉES PAR L'AUTO-MERGE**, toutes dans des tests NEUFS de
  l'amont appelant des symboles supprimés au lot 1 — intention amont rejouée, jamais contournée
  (même classe qu'`inflateChunk` et `worldObjectSlotBand` le 2026-09-03) :
  - `filmdec/i48_manques_research_test.go` : `bipedSlotBand(dir, …)` → `bipedSlotBandDir`,
    `bipedArchetype(dir)` → `bipedArchetypeDir`, `slots map[uint32]bool` → `SlotBand`
    (`len()` → `.Count()`, `s.slots[slot]` → `s.slots.Has(slot)`) ;
  - `filmdec/r8_i54_research_test.go` et `filmdec/r12_socle_research_test.go` : mêmes deux shims,
    même bascule de type (`r11_journal_research_test.go` en dépendait et se répare avec eux) ;
  - `filmdec/r9_i22_signal_research_test.go` : `worldObjectSlotBand(dir, n, ti)` →
    `worldObjectSlotBandDir`, `EquipmentArchetype(dir)` → `EquipmentArchetypeDir` ;
  - `filmdec/r9_ti37_identite_research_test.go` : `worldObjectSlotBand(dir, …)` → `…Dir` ;
  - `replay/ability_charges_film_test.go` et `replay/ability_impulses_film_test.go` : ils
    rejouaient la chaîne de production depuis un RÉPERTOIRE. Un shim `buildFromFilmDir` a été
    ajouté à `replay/film_shims_test.go` (fichier `_test.go` : le compilateur interdit qu'un
    chemin de production l'emprunte) ; il ÉCHOUE sur un répertoire illisible plutôt que de
    construire sur un film nil, parce que ces instruments comparent à des relevés Theater.
    Mesure inchangée dans les sept cas.

  **`replay.SchemaVersion` : aucune collision** — base 37, notre branche 37 (le chantier ne bumpe
  jamais, c'est un refacto), amont **38**. Le merge prend 38.

  **RESTE DOUTEUX / À SURVEILLER** : (a) la comparaison positionnelle du harnais rend son message
  « première étape en écart » inexploitable dès qu'une étape s'insère AILLEURS QU'À LA FIN — ce
  n'est pas un bug (le fichier de digests est une séquence figée) mais l'étape B, qui ajoutera
  `vehicles`, devra refaire le diff par NOM ; (b) l'étape `translocations` est observée AVANT
  `positions` parce que le balayage arme l'exemption du filtre — si un lot futur déplaçait ce
  balayage, l'ordre de `BuildFromFilmSteps` devrait suivre.
- 2026-09-05 15 h 39 — **A.5 : commit `eb80a4f0a`** après vérifications sur pièces du pilote.
- 2026-09-05 15 h 45 — Relecture `plan-review` fondue (BL-1 → D11, BL-2 → D3, BL-3 → D8, IMP-1/2/3
  → B.2/B.5/D4, IMP-4 → D12, IMP-5 → D8bis (+ gel utilisateur), IMP-6 → D.1/D10, IMP-7 → E.1,
  MIN-1..10 → D5/D6/D9/§1/D8ter/C.4/C.2bis/C.3/§7/F.4). D10 ajouté : B, C, D en PARALLÈLE dans trois
  worktrees dérivés de `eb80a4f0a` (créés, jonctions posées, `npm ci` lancé). Agents lancés :
  B (Opus), C (Opus), D (Sonnet).
- 2026-09-05 — **ÉTAPE D exécutée et commitée** (worktree dédié `LevelUp-wt-integ-gamechangers`,
  branche `wt/integ-gamechangers`, HEAD de départ `eb80a4f0a`). D.0 : `git fetch origin` — 0 commit
  d'écart `eb80a4f0a..origin/feat/v75` (amont toujours gelé). `git merge --no-ff --no-commit
  wt/game-changers` (base `ca55f0ed7`, 4 commits, 16 fichiers) : **merge automatique SANS AUCUN
  CONFLIT** — `git status` rend « All conflicts fixed », `git grep '^<<<<<<< HEAD$'` VIDE. Les
  croisements redoutés avec les étapes B/C (`i18n.ts`, `i18nContract.ts`,
  `MatchEquipmentUsageSection.tsx`) ne se sont pas matérialisés : cette branche ne porte encore
  aucune des deux, rien à arbitrer ici — reste au pilote à la fusion finale. `.ai/thought_log.md`
  a fusionné seul en union (nos entrées en tête, celles de `wt/game-changers` à leur ancre
  d'origine, aucune perdue).

  **Fichiers apportés** : `.ai/PLAN_REPLI_GAME_CHANGERS_2026-09-05.md` (neuf, plan de repli
  « game changers »), 3 `apps/web/src/components/ui/collapsed-items-toggle.{tsx,test.tsx,
  guard.test.ts}` (extraction du contrôle « Voir plus (N) », 3e usage → règle n°6), 12 fichiers
  `apps/web/src/features/match-replay/` dont `gameChangers.ts`/`.test.ts` (nouveaux — la liste
  tranchée par vote utilisateur des familles d'équipement/armes « game changer ») et les
  adaptations de `equipmentUsageColumns.ts`, `MatchEquipmentUsageSection.tsx`,
  `MatchPadControlSection.tsx`, `padControlLogic.ts`, `i18n.ts`, `i18nContract.ts`.

  **Gates D.2 — tous verts, un code de sortie par ligne** : `npm run typecheck` exit 0 ·
  `npm run lint` exit 0 (23 warnings préexistants, 0 erreur, aucun sur un fichier touché par ce
  merge) · `npm run lint:fields` exit 0 (220 labels FR+EN, 1643 fichiers scannés, 0 violation) ·
  `npm run test` exit 0 (**577 fichiers, 6008 tests passés, 14 skipped, 0 échec**) ·
  `npm run build` exit 0 (avertissements de taille de chunk préexistants, non nouveaux) ·
  `node tools/knip-ratchet.mjs` (depuis la racine) exit 0 — **files 0/0, exports 0/0, types
  0/0** : le nouveau `collapsed-items-toggle.tsx` est bien câblé (consommé par
  `equipmentUsageColumns.ts`, `MatchEquipmentUsageSection.tsx`, `MatchPadControlSection.tsx`),
  aucun export mort neuf, plafond jamais relevé.

  **Preuve harnais [~]** : `git diff --stat eb80a4f0a -- apps/go-api` rend VIDE — zéro octet
  touché côté Go par ce merge (branche 100% web) ; le harnais d'équivalence (13 films) reste
  valide sans rejeu, comme prévu pour une étape sans changement attendu côté artefact.

  **Douteux / hors périmètre signalé, non traité ici (règle n°5 zéro fix opportuniste)** : le
  journal `wt/game-changers` (commit `c972b8be0`) documente un lot I abandonné sur clarification
  utilisateur (exposition `weapon_key` pour répliquer le repli aux graphes de performance) — décision
  déjà actée par cette branche avant le merge, rien à statuer côté intégration.

  Commit du pilote : merge de fusion sur `wt/integ-gamechangers`, message français, voir
  `git log --oneline -1` pour le SHA (rapporté à l'utilisateur en clôture de tâche).

- 2026-09-05 — **ÉTAPE C exécutée (C.0 à C.5)**, worktree `LevelUp-wt-integ-usage`, branche
  `wt/integ-usage`, HEAD de départ `eb80a4f0a`.

  **C.0 / D8bis.** `git fetch origin` : `origin/feat/v75` **toujours `7fb4b60a1`**,
  `HEAD..origin/feat/v75` = **0 commit**. Rien à réconcilier. Cible : `wt/session-usage`
  `c4f7d5417`, base `da616828f`, 10 commits, **67 fichiers** (+8 737 / −87).

  **LES TROIS CONFLITS ET LEUR RÉSOLUTION** (règle : la sémantique de la branche est préservée
  intégralement, dans NOTRE architecture) :

  | conflit | ce que la branche voulait | résolution |
  |---|---|---|
  | `sync/replayartifacts/cuisson.go` | dans `buildAll`, après `b.construits++` : empiler `rapportUsage{matchID, path: out.Path}` | Le bloc en conflit EST celui que le lot 5 a sorti de `buildAll` vers `cuireUnMatch` (items 5.5/5.6). Bloc de la branche supprimé ; le crochet **REJOUÉ dans `cuireUnMatch`**, immédiatement après le report du T0, sur `out.stored.Path` (notre `storedOne{stored, dur, peak}` au lieu du `stored` nu). Le champ `usage []rapportUsage` du bilan avait fusionné SEUL. Doctrine de `usage.go` respectée sur les quatre points : fichier RANGÉ et non blob candidat, projection après rangement, toutes les projections avant `AcquireWriter`, segment writer court après TOUTE cuisson (`Run` : `reporterT0Film` -> `persisterResumesUsage` -> `publierBilan`, fusionné seul dans `artifacts.go`) |
  | `cmd/levelup/main.go` | `case "backfill-usage-summary"` + sa ligne d'aide | UNION, aux deux endroits : la commande de la branche prend sa place au contact de `backfill-replay` (son voisin thématique), notre `replay-facts-export` la suit |
  | `.ai/thought_log.md` | deux blocs insérés aux mêmes ancres | UNION : nos entrées en tête, puis celles de la branche — aucune perdue des deux côtés |

  **CE QUI A FUSIONNÉ SEUL, ET QUI A ÉTÉ RELU** : `replayartifacts/artifacts.go` (l'appel à
  `persisterResumesUsage` est bien APRÈS `reporterT0Film` et AVANT `publierBilan`),
  `replayartifacts/journal.go` (deux compteurs), `killcollector/{classifier,postsync}.go`
  (centralisation de la lecture des capabilities dans `games.LoadCapabilityMap` + garde-rail
  `capability_loader_guard_test.go`), `no_art_patterns_test.go` (**le durcissement daté du
  2026-09-04 est intact** : le diff contre HEAD est purement additif, +8 lignes, les deux
  nouvelles tables dans `tablesProtegees`), `migration/order.go`, `openapi.yaml`.

  **DEUX AJUSTEMENTS AU-DELÀ DES MARQUEURS**, tous deux dans le sens de la doctrine du dépôt :
  - `migration/order.go` : l'auto-merge avait glissé l'entrée `shared_match_usage_summary_v1`
    ENTRE le commentaire « Table SOEUR de match_weapon_shots… » et l'entrée qu'il décrit
    (`shared_match_weapon_hit_distance_v1`) — doc inversée (anti-pattern n°9). Chaque bloc de
    commentaire a été remis au contact de son entrée ; l'ORDRE des entrées, lui, ne bouge pas
    (il est dicté par `TestSortByCanonicalIsNoOpOnCurrentRegistry`).
  - `sync/append_only_state_guard_test.go` : étape **5** de la recette ADR 0026, manquante à
    l'arrivée de la branche — `match_usage_players` et `match_usage_films` inscrites à
    `appendOnlyStateTables`, sur le patron de `match_kill_events`/`match_weapon_shots`
    (cf. C.2bis et N-6).

  **D11 — L'OPENAPI N'A PAS ÉTÉ TOUCHÉ À LA MAIN.** `make openapi-gen` régénère 666 456 octets
  et le diff contre l'auto-merge est **NUL** (l'auto-merge était déjà la sortie du générateur) ;
  `make generate-types` ; `make openapi-check` code 0 ;
  `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate` **ok** ;
  `CGO_ENABLED=0 go test ./contracttest/... -run TestContract` **ok** ;
  `node tools/check-generated-types-fresh.mjs` code **0**. +307 lignes d'OpenAPI, conformes au §1.

  **C.2 — LES PREUVES, MESURÉES, PAS DÉDUITES** :

  1. **Persister INSERT-only.** `internal/persist/usage_summary_persister.go` ne porte que deux
     `const` SQL, `INSERT INTO match_usage_films` et `INSERT INTO match_usage_players`, dans une
     transaction unique (`BeginTx` / `Commit`, `Rollback` différé). Grep exhaustif du module :
     `(UPDATE|DELETE FROM|ON CONFLICT).*match_usage` rend **VIDE**. La passe entière (ligne film
     + lignes joueurs) porte le même `summary_pass` et le même `written_at` — la re-projection
     est atomique à la lecture.
  2. **Append-only + vues `_latest`, dans le bon ordre.** `steps_shared_usage_summary.go` :
     deux tables à PK technique `id` sur séquence, `written_at` par défaut UTC, un index par
     table ; `applyMatchUsageSummary` exécute les scripts dans l'ordre
     `players`, `films`, **`films_latest`, puis `players_latest`** — et le commentaire dit
     pourquoi : `players_latest` se DÉFINIT sur `films_latest` (JOIN sur
     `match_id` + `summary_pass`), l'autorité de passe étant la ligne film, toujours écrite.
  3. **Lecteurs sur les vues UNIQUEMENT.** Grep `match_usage_[a-z]*` hors migration et hors
     tests : les seuls `FROM` de production sont
     `platform/duckdb/session_usage_repo.go:58` (`FROM match_usage_films_latest`), `:95`
     (`FROM match_usage_players_latest`) et `cmd/levelup/cmd_backfill_usage_summary.go:162`
     (`FROM match_usage_films_latest`). **Aucune lecture brute.** Toutes les autres occurrences
     sont des commentaires ou les deux INSERT du persister.
  4. **Capability, jamais un slug.** `games.CapFilmUsageSummary = "film.usage_summary"`, déclarée
     dans `AllCapabilityKeys`, dans le repli d'`adapter_data.go` et dans
     `config/titles/halo_infinite/mappings/capabilities.toml` ; absente de Halo 5 (pas de
     décodeur de film). Trois gates, tous par la clé : production post-sync
     (`replayartifacts/usage.go:88`), backfill CLI (`cmd_backfill_usage_summary.go:122`),
     exposition de la page (`api/wire/registry_pages.go:298` -> repo nil, donc bloc absent, jamais
     un 500). Ratchet `archlint/no_slug_comparison_test.go` **vert** (paquet `internal/archlint`
     `ok`, 22,7 s).
  5. **`OpenReadWrite`.** Un seul site de production neuf,
     `cmd/levelup/cmd_backfill_usage_summary.go:103`, CLI hors ligne sous précondition « serveur
     arrêté » écrite en tête de fichier — même régime que `cmd_backfill_killsource.go`. Le
     ratchet D-3 de l'ADR 0030 n'existe pas dans le code (N-9) : la vérification est manuelle et
     dite comme telle.

  **RESTE DOUTEUX / À SURVEILLER** : (a) **N-10** — cinq issues sous le ratchet CI du lint, dans
  deux fichiers neufs de la branche ; elles rougiront `Go Lint` et `make gate-push` tant qu'elles
  ne sont pas éteintes (étape E) ; (b) l'entrée thought_log S1 de la branche (2026-09-04) décrit
  encore le crochet comme vivant dans `buildAll` — c'est un journal historique, il n'a pas été
  réécrit, mais la lecture du code fait foi : le crochet vit dans `cuireUnMatch` depuis cette
  résolution ; (c) le résumé n'est produit QUE pour les artefacts cuits DANS LE CYCLE — le parc
  déjà sur disque relève de `levelup backfill-usage-summary`, qui n'a PAS été lancé ici (aucune
  base ouverte de toute l'étape).
- 2026-09-05 — **ÉTAPE B exécutée (B.0 à B.5), un seul commit de merge** (union D9 : deux parents,
  `origin/feat/v75-vehicules-sons` `1e3d459d1` et `wt/vehicule-deadstate` `0b5141b8a`, sur
  `eb80a4f0a`). D8bis : `origin/feat/v75` MESURÉ inchangé à `7fb4b60a1` — zéro commit amont.
  387 fichiers, 26 conflits.

  **LES 26 CONFLITS ET LEUR RÉSOLUTION** (règle : la SÉMANTIQUE véhicules est préservée
  intégralement, dans NOTRE architecture, et POSÉE SUR LE 38 amont) :

  | conflit | ce que la branche voulait | résolution |
  |---|---|---|
  | `replay/build.go` (2 blocs) | ajouter `Vehicles VehicleScan` à `Options`, un appel de décodage à `BuildFromFilm`, et — dans `BuildFromPositions` — `buildShots` à 3 retours, `attachVehicles`, `attachVehicleShots`, le journal sur `doc.Coverage.Shots` | Le premier bloc EST celui que le lot 1 a sorti de `build.go`. Diff bloc-à-bloc base `14a115bb1` contre la branche : elle n'y touche QUE ces deux choses. Bloc supprimé ; le champ REJOUÉ dans `options.go` au rang de `Pads`, l'appel dans `build_from_film.go` juste après `pads`. Les hunks de `BuildFromPositions` avaient fusionné SEULS (vérifié : `shotOrphans`, `attachVehicles`, `attachVehicleShots`, `shotsPub` en place) ; le second conflit est l'UNION du bloc « impulsions + charges » (à nous) et de la ligne `shotsPub := doc.Coverage.Shots` (à elle) |
  | `replay/document.go` | trois blocs de doc de version (29 véhicules, 30 tirs embarqués, 31 visée d'occupant) et `SchemaVersion = 31` | Notre bloc 29→38 est conservé INTÉGRALEMENT ; les trois blocs de la branche FONDENT EN UN, « CE QUE LA VERSION 39 PORTE… EN TROIS TEMPS », avec les renvois de numéro corrigés (« un artefact 38 doit se lire à re-cuire ») ; `const SchemaVersion = 39`. Les champs `Vehicles`, `VehicleLabels` et `Shot.Vehicle` avaient fusionné seuls |
  | `replay/structure_test.go` | les trois mêmes blocs + `SchemaVersion != 31` | Même geste : nos blocs jusqu'à 38, puis les trois de la branche renumérotés `v39 (1)/(2)/(3)` sous une introduction qui dit la fusion, puis `!= 39` |
  | `contracttest/replay_contract_test.go` | `wantReplayDocumentFields` 44 → 46 (`vehicles`, `vehicleLabels`) | Notre 52 + les deux champs = **54**, avec l'entrée de chronique « 52 → 54 » et « Les vingt et une fois » |
  | `replay/testdata/assembly_000d5950.golden` | `schema 31` | `schema 39` |
  | `filmdec/offline_biped.go` (4 hunks) | scinder `ScanFilmBipedPositions` en `bipedScanChunks` / `bipedI0Layout` / `scanBipedChunks` pour ouvrir `ScanFilmBipedPositionsForBand(dir, band, opt)`, et ajouter l'option `DynPrecOrientation` | Structure de la branche reprise TELLE QUELLE, en forme FILM : `ScanBipedPositions(film, opt)` délègue à `ScanBipedPositionsForBand(film, band, opt)` ; les trois helpers prennent `*filmsource.Film` ; la bande est une `SlotBand` dense (le type du lot 2) et non une map. `DynPrecOrientation` conservé mot pour mot, sa grammaire RÉSOLUE UNE FOIS PAR PAYLOAD (hors de la boucle de records) au lieu d'une fois par record |
  | `filmdec/offline_aim.go` | `scanRecordDirs(pay, …, g dirsGrammar)` + deux lecteurs dyn.-préc. qui allouent chacun leur `BitReader` | Corps de la branche pris INTÉGRALEMENT ; la tête garde NOTRE lecteur partagé : `scanRecordDirs(br, …, g)`, et `readForwardComponentDynPrec` / `readAngularVelocityComponentDynPrec` reçoivent `br` au lieu d'allouer — deux allocations de moins par record, exactement la raison d'être du lecteur partagé |
  | `filmdec/offline_aim_test.go` | deux appels à `scanRecordDirs` | Signature à cinq paramètres, lecteur partagé |
  | `.ai/thought_log.md` (2 blocs) | deux blocs insérés aux mêmes ancres | UNION : nos entrées en tête, puis celles de la branche, ligne de séparation rétablie ; l'entrée V13 du dead-state (second parent) insérée en tête de la section du chantier |
  | `.ai/V7.5/REGISTRE_REPORTS.md` | une ligne de report (cap i2 réfuté) | UNION du tableau |
  | `.gitignore` | un commentaire sur `.gocache-*/` | UNION (le motif était déjà chez nous, le commentaire est repris) |
  | `api/openapi.yaml`, `web/lib/api/generated.ts` | contrat régénéré au schéma 31 | D11 : `--ours` puis **régénération** (`make openapi-gen && make generate-types && make openapi-check`), JAMAIS à la main |
  | 13 fichiers `web/features/match-replay/` | le calque véhicules dans le tiroir, le contrat, la normalisation, les marqueurs, le cône | UNION partout. Trois résolutions non triviales : (a) `ReplaySettingsLayers.tsx` — la branche pose une liste PLATE, nous des `LayerGroup` ; la bascule `vehicles` entre dans le groupe TERRAIN (l'intention écrite de la branche : « meubles du terrain, pas l'enjeu du mode ») ; (b) `replayAimCone.ts` — la branche extrait `drawAimSector` pour que le calque véhicules réutilise le cône, nous avions ajouté la LUNETTE (ouverture variable + gain d'opacité) ; le secteur extrait reçoit deux options de plus, `halfAngle` et `alphaBoost`, et le cône du pion les lui passe : une seule géométrie, les deux mécaniques intactes ; (c) `ReplayCanvas.tsx` — `showNames` n'existe plus (le calque des noms a quitté le tiroir le 02/09, toujours allumé) : le calque véhicules reçoit `showNames: true` |

  **DEUX RUPTURES DE COMPILATION LAISSÉES PAR L'AUTO-MERGE**, toutes deux dans des tests de la
  branche appelant des symboles disparus au lot 1 — intention rejouée, jamais contournée :
  `filmdec/event_list_test.go` redéclarait `itoa` (le nôtre vit dans
  `lot1_visee_calib_research_test.go` : le doublon est retiré, la mesure est identique) ; et
  `killsource/vehicules_v10_deadstate_test.go` appelait `DirChunks(dir)`, supprimé au lot 1 —
  remplacé par `filmsource.LoadDir(dir, nil)`, la source que `loadFilm` prend désormais.

  **LES BALAYAGES MIGRÉS (D2)**, avec l'enveloppe conservée et son unique raison d'être :

  | ancien (site) | nouveau | enveloppe `(dir)` gardée pour |
  |---|---|---|
  | `ScanFilmWorldObjectKeyframes(dir, 40)` — `build_vehicles.go:84` | `ScanWorldObjectKeyframes(film, VehicleTypeIndex)` — existait déjà (`world_object_census.go:79`) | déjà à la liste depuis le lot 1 |
  | `ScanFilmVehicleCreationsForBand(dir, wr, band)` — `build_vehicles.go:90` | `ScanVehicleCreationsForBand(fc, wr, band)` — `vehicle_creation.go` | AUCUNE : l'enveloppe n'avait pas d'appelant, elle est SUPPRIMÉE (N-6) |
  | `ScanFilmBipedPositionsForBand(dir, band, opt)` — `build_vehicles.go:96` | `ScanBipedPositionsForBand(film, band, opt)` — `offline_biped.go` | `filmdec/offline_biped_test.go`, `vehicle_creation_test.go`, huit instruments `vehicules_v*` de `filmdec` et sept de `replay` |
  | `ScanFilmBipedAimOnly(dir)` — `build_vehicles.go:125` | `ScanBipedAimOnly(fc)` — `offline_aim_only.go` (bande bipède du CONTEXTE) | `filmdec/vehicules_v11_scan_test.go` |
  | `ScanFilmVehicleEvents(dir)` — `build_vehicles.go:139` | `ScanVehicleEvents(fc)` — `event_list.go` (bande bipède du CONTEXTE) | `filmdec/event_list_test.go`, `event_list_board_test.go`, `replay/vehicules_v2b_occupant_test.go`, `vehicules_v3_destruction_test.go` |
  | `ScanFilmVehicleCreations(dir, wr)` | `ScanVehicleCreations(fc, wr)` | `replay/vehicules_v2b_cooldown_test.go` |
  | `ScanFilmKeyframeRecordSpans(dir)` | `ScanKeyframeRecordSpans(film)`, puis SUPPRIMÉ (N-6) | — |
  | `ScanFilmVehicleOccupancy(dir)` | `ScanVehicleOccupancy(film)`, puis SUPPRIMÉ (N-6) | — |

  `decodeFilmVehicleScan` ne touche plus le disque : elle prend le `*FilmContext` et fait dessus
  ses CINQ lectures (recensement, créations, nuage, événements, visées) ; le découpage d'i0 du
  nuage `ti=40` vient désormais du CONTEXTE (`ImposedLayout`) et non de l'auto-détection — la même
  règle du catalogue que les positions bipèdes du même film (lot 3). Le nom garde le préfixe
  `decodeFilm*` de ses cinq sœurs (`decodeFilmPadScans`, `decodeFilmPlacements`, …), qui est aussi
  ce que lit `observe_test.go`.

  **UNE FACTORISATION, PARCE QUE C'ÉTAIT LA TROISIÈME COPIE** : la boucle de marche des records de
  création existait pour l'équipement et pour l'arme au sol ; le véhicule en aurait été la
  troisième. `runCreationWalk(fc, w, &st)` est devenu le point de passage UNIQUE des trois, chacun
  gardant ses refus en propre (règle n°6 du dépôt).

  **GARDES TOUCHÉES** :
  - `archlint/no_film_reread_test.go` — `enveloppesInterditesEnProduction` **43 → 47** :
    `ScanFilmBipedPositionsForBand`, `ScanFilmBipedAimOnly`, `ScanFilmVehicleEvents`,
    `ScanFilmVehicleCreations`, avec la justification datée et la vérification d'homonymie exigée
    par le fichier (aucune méthode ne porte ces noms — vérifié). Les TROIS enveloppes sans appelant
    ne sont PAS entrées à la liste : elles ont été supprimées (N-6).
  - `archlint/filmdec_package_vars_test.go` — ratchet **116 → 118**, avec les deux variables
    nommées et justifiées : `unitRefHook` (une SONDE, nil en production, le patron déjà compté du
    paquet) et `vehicleMediaFrameBits` (une TABLE DE GRAMMAIRE mesurée). L'intégration n'en ajoute
    AUCUNE de son fait — la seule erreur sentinelle qu'elle allait poser est restée locale.
  - `archlint/no_recomputed_film_context_test.go` — l'entrée
    `offline_biped.go/ScanBipedPositions -> DetectI0LayoutOf` devient
    `offline_biped.go/bipedI0Layout -> DetectI0LayoutOf` : MÊME appel, MÊME valeur, le site a
    seulement été extrait pour que les deux entrées le partagent. Aucune allowlist élargie.
  - `archlint/no_rewritten_slot_band_test.go` — une entrée d'allowlist de plus,
    `vehicle_creation_test.go`, pour sa bande FANTÔME : c'est le NÉGATIF d'une bande (le témoin
    d'ancrage), pas une quatrième règle — exactement le cas déjà admis pour
    `equipment_creation_test.go`.
  - `replay/observe.go` — `BuildFromFilmSteps` **34 → 35** : `vehicles` entre APRÈS `pads`, à
    l'endroit exact où le balayage tourne. **Pas d'étape pour les tirs en véhicule ni pour la
    couverture** : ils vivent dans `BuildFromPositions`, que l'observateur ne couvre pas, et le
    digest `artifact` les porte.
  - `web/placementFamily.guard.test.ts` — plafond de `ReplayCanvas.tsx` **INCHANGÉ à 665** ; le
    fichier en fait 664 après le câblage véhicules (N-7).
  - `no_unbounded_film_loop_test.go`, `filmsource_leaf_test.go`, `no_rewritten_slot_band_test.go`
    (règle 2), `no_art_patterns_test.go` : verts sans modification.
## §7 Contrat d'exécution (rappel opposable)
Statuts : `[x]` fait · `[~]` couvert ailleurs (avec la référence) · `[!]` non traité (avec la
justification écrite). AUCUNE case vide à la clôture d'une étape. Ordre strict : l'étape N+1 ne
commence pas avant le gate de N (B, C, D sont parallèles entre elles par D10, mais chacune est
strictement ordonnée en interne, et E ne commence qu'après les trois fusions). Zéro fix hors
périmètre : la découverte va au §4, pas dans le diff. Jamais d'allowlist élargie sans
justification datée, jamais de plafond de ratchet relevé. REPRISE DE SESSION : lire §5 (dernier
diff de comptes consigné) puis §6 (journal) ; la première case non cochée de la première étape non
close est le point de reprise ; les worktrees D10 se listent par `git worktree list | grep integ`.
