# Lot A — phase 1 (publier le score dans le temps) : journal d'execution

> Executeur, 2026-08-18, worktree `../LevelUp-wt-score-film`, branche `wt/score-film`.
> Perimetre ferme : `LOTA_ARBITRAGE_PHASE0TER.md` §« Phase 1 — contenu tranche », items A.1.1 a
> A.1.5. A.1.0 est deja livre (commit `2fd5f8825`).

## Seuil du controle A.1.2, ECRIT AVANT LA MESURE

Le plan exige « accord des actions avec `match_objective_events` (ms) >= 95 % sur les 3 films
CTF » sans fixer de TOLERANCE. Elle est fixee ici, avant la mesure, et justifiee :

- **Tolerance : +/- 5 000 ms.** Le statborg reemet un compteur A SON rythme (intervalles mesures
  de 1 a 31 s, `cmd/zone-attribution/report.go`), et `incrementTimes` date l'unite gagnee sur
  l'EMISSION qui la revele — l'instant publie MAJORE donc l'action. La source de la base, elle,
  est le `burst` (la frame qui retransmet la table d'objectif). Les deux horloges sont la meme
  (l'horloge du film), mais les deux evenements ne sont pas emis au meme instant. Cinq secondes
  est la valeur deja employee comme temoin par le balayage de containment (`clockOffsets` +/-5 s).
- **Denominateur : les lignes `event_type = 'capture'` de `match_objective_events`**, seul grain
  comparable — la table ne porte, pour la famille `flag`, que les CAPTURES d'equipe, la ou le
  calque publie huit statistiques par joueur.
- **Sont aussi rapportes** : l'ecart median, et le taux a +/-1 s et +/-10 s, pour que le lecteur
  juge la tolerance et pas seulement le verdict.

## Ce qui est livre, fichier par fichier

### A.1.1 — `ReplayDocument.scoreTimeline` (schema 12)

| Fichier | Ce qu'il porte |
|---|---|
| `internal/analysis/replay/document_score.go` (NEUF) | la FORME publiee : `ScoreTick`, `ScoreRound`, `ScoreSeries`, `TeamScore`, `PlayerScore`, `ScoreTimeline`, `ScoreCoverage` ; l'oracle (`displayed`) et les trois identites de camp (`a` / `b` / `unresolved`) ; ce qui n'est PAS publie (les decrements du score personnel) |
| `internal/analysis/replay/score_timeline.go` (NEUF) | `ScoreInput` (l'entree de l'appelant), `scoreClock` (film -> frame, origine retranchee), `buildScoreTimeline` et ses aides ; assemblage PUR, aucune I/O |
| `internal/analysis/replay/score_team_identity.go` (NEUF) | D3 : `resolveTeamIdentity`, `identityByFinalScore` (a), `identityByFrags` (b) |
| `internal/analysis/replay/score_timeline_test.go` (NEUF) | 11 tests sur des `StatRecord` CONSTRUITS — zero octet de film |
| `objectiveevents/score.go` | `StatComponent` + les 5 emplacements nommes, `SeriesByRound`, `SeriesTotal` ; `ScoreCurveFrom` reecrite dessus ; `ScoreRoundsFrom` (0 appelant) remplacee |
| `objectiveevents/named.go`, `slotidentity.go` | `NamedEventsFrom` et `SlotIdentityFrom` EXPORTES : le constructeur decode le film UNE fois pour trois calques |
| `replay/document.go`, `coverage.go`, `build.go` | `SchemaVersion` 11 -> 12 + chronique, champ `ScoreTimeline`, `Coverage.Score`, `Options.Score` |
| `internal/port/services.go` | `MatchPlayerFact`, `MatchFacts`, `ReplayFactsRepo` — l'ENTREE que ni `analysis/` ni `replaybuild` ne vont chercher en base |
| `internal/platform/duckdb/replay_facts_repo.go` (NEUF) + son test `:memory:` | le lecteur MINIMAL : deux `SELECT` (registre + participants), lecture seule, 4 cas testes |
| `internal/replaybuild/matchfacts.go` (NEUF) | `readFilmStats` : `filmcache.OpenChunkDir` -> `StatRecordsCtx` -> score + identite + evenements, en UN decodage |
| `filmcache/filmcache.go` | `OpenChunkDir` : la remontee chunks -> racine du cache, declaree LA ou vit la disposition |
| `replaybuild.go` et les 5 appelants de `BuildMatch` | la signature prend `port.MatchFacts` |

Les cinq appelants et ce qu'ils fournissent :

| Appelant | Source des faits |
|---|---|
| `internal/api/wire/registry_replay_build.go` (action admin) | `ReplayFactsRepo` sur le handle shared DEJA ouvert, AVANT `closeAll()` |
| `internal/sync/replayartifacts/artifacts.go` (fil de l'eau) | `attachMatchFacts`, dans le MEME segment `WithRead` que la selection |
| `cmd/levelup/cmd_backfill_replay.go` (masse) | `chargerFaitsReplay`, une ouverture RO fermee AVANT la boucle de decodage |
| `cmd/replay-build/main.go` (unitaire) | `--facts <fichier.json>` — le binaire reste HORS LIGNE (compile et tourne sans CGO, verifie) |
| `cmd/replay-worker/job.go` (ouvrier distant) | AUCUNE : degradation ANNONCEE (`slog.WarnContext`), le payload de job ne porte pas encore les faits |

### A.1.2 — `Options.Objectives` en production, et l'origine retranchee

- `replaybuild/matchfacts.go` : `ObjectiveTypeOf(variante)` + `NamedEventsFrom` +
  `IdentifyNamedEvents(..., SlotIdentityFrom(...))`. Deux refus explicites journalises : mode sans
  famille d'objectif, et absence de lignes de match.
- `replay/objectives.go` : `buildObjectiveActions` prend desormais la `scoreClock` et RETRANCHE
  l'origine (report `:123` du registre) ; la fenetre borne l'axe des DEUX cotes.
- `replay/origin.go` : `originMSOf` — l'absence d'origine est un REPLI journalise, pas un zero
  neutre.
- `replay/objectives_test.go` : 2 tests neufs (origine retranchee ; refus de ce qui precede la
  frame 0).

### A.1.3 — suppression de `filmdec/statborg.go`

Supprimes : `ParseStatborgRecord`, `StatborgBinding`, `StatborgTarget`, `setDirty`,
`statborgSlots`, `statHeaderBits`, et le test `TestParseStatborgRecord_V8`. `ParseOptionalValue`
(primitive SANS rapport avec la chaine ti=6, hors du perimetre tranche par D1) est DEPLACEE dans
`bitreader.go` avec son test. `doc.go` du paquet dit desormais pourquoi le decodeur est parti.
`ecs_table.tsv` INCHANGEE (verifie : `git status` ne la voit pas) ; G1 vert.

### A.1.4 — contrat

- `SchemaVersion` 11 -> **12**, chronique dans `document.go` ET dans le garde `structure_test.go`.
- `wantReplayDocumentFields` 33 -> **34**, ligne de chronique ecrite ; 7 schemas ajoutes a
  `replaySchemas` (`ScoreTimeline`, `TeamScore`, `PlayerScore`, `ScoreSeries`, `ScoreRound`,
  `ScoreTick`, `ScoreCoverage`).
- `api/openapi.yaml` regenere (`go run ./cmd/openapi-gen`, 621 472 o), `generated.ts` regenere
  (`npm run generate-types`).
- **`NULLABLE_ARRAYS` : AUCUNE entree a ajouter, et c'est verifie et non suppose.** `scoreTimeline`
  est un OBJET (`components["schemas"]["ScoreTimeline"]`, optionnel, non nullable) ; la liste
  n'enumere que les TABLEAUX nullables de premier niveau. L'assertion de type `_ListeExhaustive`
  du contrat web echouerait a la compilation si un tableau manquait : `npx tsc -b --force` = 0.
- **Golden** : `assembly_000d5950.golden` change d'UNE ligne — `schema 11` -> `schema 12`. Aucune
  autre ligne ne bouge, et c'est la justification : le golden rejoue l'assemblage sur des entrees
  FIGEES qui ne portent aucun `ScoreInput`, donc ni courbe ni couverture de score n'y apparaissent.

### A.1.5 — controle apres cuisson

Artefacts ecrits **sous le worktree du lot** :
`C:\Users\Guillaume\Projects\LevelUp-wt-score-film\data\cache\replays\halo_infinite\` — jamais le
cache du principal (`LEVELUP_REPO_ROOT` pointe le worktree, les films sont lus par chemin ABSOLU
depuis le principal, en lecture seule).

| Temoin | Mode | Oracle affiche | `scoreTimeline` publie | identite | manches | joueurs | `objectives[]` |
|---|---|---|---|---|---|---|---|
| `000d5950` | Slayer | 43 / 50 | **43 / 50** | `a` | 1 | 6 | 0 (mode sans famille) |
| `530820e5` | CTF | 3 / 0 | **3** (equipe 0) ; l'equipe a 0 n'emet RIEN | `b` | 1 | 8 | **183** |
| `24dbb67d` | Oddball | 200 / 121 (manches 100/78 puis 100/43) | **200 / 121**, manches **100/78** puis **100/43** | `a` | 2 | 0 | 0 (mode sans famille) |

Couverture `coverage.score` coherente sur les trois (`oracle: displayed`, `truncated: false`,
`modeSupported: true`, `points` = 960 / 1 100 / 642).

Cout mesure (un film par processus, avant-plan, plafond 3 Go surveille — D17) : pic RAM **130 a
217 Mo**, duree 2 a 5 min par film (le decodage des positions domine).

Tailles (octets ; « avant » = l'artefact sans les trois ajouts, mesure par soustraction des
sous-documents publies) :

| Temoin | avant | apres | `scoreTimeline` | `objectives` | `coverage.score` | delta |
|---|---|---|---|---|---|---|
| `000d5950` | 2 252 049 | 2 270 742 | 18 580 | 0 | 113 | +0,83 % |
| `530820e5` | 1 464 984 | 1 499 597 | 21 563 | 12 936 | 114 | +2,36 % |
| `24dbb67d` | 1 504 395 | 1 516 241 | 11 733 | 0 | 113 | +0,79 % |
| `64e8adfa` | 2 681 489 | 2 681 865 | 256 | 0 | 120 | +0,01 % |
| `53ce4390` | 2 293 131 | 2 332 921 | 25 712 | 13 964 | 114 | +1,74 % |

Controle d'echelle : l'artefact `000d5950` de PROD (schema 11) pese 2 252 021 o, soit 28 o de
l'estimation « avant » ci-dessus — la methode de soustraction est juste a 0,001 % pres. Le seuil
A.0.4 (<= 60 Ko de plus par artefact en mediane) est TENU largement : le calque de score pese 11,7
a 25,7 Ko, actions d'objectif comprises 11,7 a 39,7 Ko.

## Controle A.1.2 : accord avec `match_objective_events`

Export lecture seule : `oracle_export_lotA_phase1.sql` -> `oracle_lotA_objective_events.tsv`
(8 lignes ; base ouverte en `-readonly`, aucune ecriture, serveur local laisse en place).

| Film | lignes oracle (`capture`) | `flag_captures` publiees | accord +/-5 s | ecarts |
|---|---|---|---|---|
| `64e8adfa` | 5 | **0** | 0/5 | — |
| `530820e5` | **0** | 3 | 0/0 | — |
| `53ce4390` | 3 | 3 | **3/3** | **0 ms, 0 ms, 0 ms** |

- **Sur le corpus mesurable — le seul film qui porte a la fois un oracle et une identite
  resoluble — l'accord est 3/3 = 100 %, a l'INSTANT EXACT (0 ms d'ecart).** La tolerance de 5 s
  ecrite avant la mesure n'a pas servi : le `burst` de la base et l'increment du statborg tombent
  sur la meme milliseconde.
- **Le seuil « >= 95 % sur les 3 films CTF » n'est PAS atteint au sens litteral : 3/8 = 37,5 %.**
  Les deux ecarts sont imputes a l'ORACLE et au CORPUS, pas au calque, et les deux etaient deja
  ecrits avant ce lot :
  - `64e8adfa` est le film **tronque de 24,6 s** que la phase 0 a EXCLU du corpus comparable
    (`LOTA_PHASE0.md` : couverture 0,97, identite (c), appariement joueurs 0/8). Le constructeur
    reproduit exactement ce diagnostic : `slotsApparies=0`, `identiteEquipes=unresolved`,
    266 evenements nommes et 0 identifie — la regle de prudence refuse d'attribuer.
  - `530820e5` n'a **aucune ligne** dans `match_objective_events` : le denominateur est vide (le
    peuplement live etait precisement le report `:12`, que ce lot ferme cote ARTEFACT et pas cote
    table). Ses 3 `flag_captures` publiees concordent avec `team_0_score = 3`.
- Aucun seuil n'a ete rebaisse. Le controle est statue `[!]` au sens litteral, TENU sur le corpus
  mesurable ; l'arbitrage appartient au superviseur.

## Decouvertes (hors perimetre — NOTEES, NON TRAITEES)

1. **Le garde-rail `TestUneSeuleSourceDisqueDeFilm` etait ROUGE en entrant sur la branche** :
   `objectiveevents/statborg_rounds_test.go` (arrive au commit A.1.0 `2fd5f8825`) implemente
   `FilmSource` sans entree d'allowlist. TRAITE, parce qu'il bloquait le gate de cloture (regle 7 :
   seule exception admise) : entree datee ajoutee, meme motif de cycle d'import que l'entree
   existante. A signaler au superviseur — le lot A.1.0 a livre avec ce gate rouge.
2. **Une equipe qui ne marque JAMAIS n'a pas de courbe** (CTF 3-0 : une seule `teams[]`). Le film
   ne transmet un compteur que lorsqu'il CHANGE ; publier une courbe plate a 0 serait une
   inference, pas une lecture. Consequence : le client ne peut pas dessiner la ligne du camp
   perdant. Decision de RENDU, elle appartient a la phase 2 (web).
3. **La preuve (a) echoue des qu'un camp reste a 0**, pour la meme raison : `identityByFinalScore`
   exige les DEUX finales. C'est (b) qui a resolu `530820e5`. Une regle « absent = 0 » serait une
   extension de D3, non tranchee.
4. **Oddball : aucun joueur publie** (`slotsApparies=0` sur `24dbb67d`). C'est le negatif de mode
   deja ecrit en phase 0-bis (A.0.3, Oddball 0/32) : le triplet frags/morts/assistances n'y apparie
   rien. Les courbes d'EQUIPE, elles, sont exactes.
5. **L'ouvrier distant construit sans faits.** Les porter exigerait de les ajouter au payload de
   `domain.BuildQueueJob`, donc de toucher le schema de la file — hors perimetre du lot A.
6. **`filmdec.ParseOptionalValue` n'a AUCUN appelant de production** (seul son test l'appelle). D1
   ne la nomme pas ; elle a ete deplacee, pas supprimee.
7. **Second decodage du film par artefact.** `readFilmStats` ajoute un balayage complet des chunks
   (grammaire des enregistrements d'entite) au decodage des positions. Cout mesure : le pic RAM par
   processus reste 130-217 Mo, sous le plafond. Une fusion des deux balayages serait un chantier de
   performance, pas un correctif.

## Gates (log persistant : `LOTA_phase1_gates.log`)

| Gate | Commande | Verdict |
|---|---|---|
| Compilation | `go build ./...` (CGO) | `EXIT_BUILD=0` |
| Vet | `go vet ./internal/... ./contracttest/... ./cmd/{replay-build,replay-worker,levelup}/...` | `EXIT_VET=0` |
| Analyse | `go test -count=1 ./internal/analysis/...` (CGO=0) | `EXIT_TEST_ANALYSIS=0` (16 paquets) |
| Chaine d'artefact | `go test -count=1 ./internal/replaybuild/ ./internal/games/halo_infinite/film/... ./contracttest/` | `EXIT_TEST_BUILDCHAIN=0` |
| Lecteur ajoute | `go test -tags=integration -run TestReplayFacts ./internal/platform/duckdb/` | `EXIT_TEST_DUCKDB_LECTEUR=0` (4/4) |
| API + fil de l'eau | `go test -count=1 ./internal/api/... ./internal/sync/replayartifacts/` | `EXIT_TEST_API=0` |
| Ratchet lint | `golangci-lint run --new-from-merge-base=origin/main ./...` | `EXIT_LINT=0` (0 issue) |
| Types web | `npx tsc -b --force` apres `generate-types` (`node_modules/.tmp` purge) | `EXIT_TSC=0` |
| Vitest rejeu | `npx vitest run src/features/match-replay` | `EXIT_VITEST_REPLAY=0` (47 fichiers, 657 tests) |

`internal/platform/duckdb` complet n'a PAS ete joue (paquet trop long) : le gate cible les tests du
lecteur ajoute, comme le prevoit le contrat du lot.

---

## Ronde 1 de revue (2026-08-18) — sept points P1 corriges

| Point | Correction | Fichier:ligne | Test ajoute |
|---|---|---|---|
| R1-1 | deduplication PAR VALEUR des courbes (le contrat disait « aux CHANGEMENTS seulement », 44,7 % des points `kills` et 46,3 % des `deaths` etaient des repetitions) | `replay/score_timeline.go:94-136` | `TestScoreTicksKeepOnlyChanges`, `TestScoreTicksSameFrameDifferentValues`, `TestScoreTimelinePlayerCountersHaveNoRepeats` |
| R1-2 | `coverage.originResolved` : l artefact DIT quand son axe de temps n est pas fiable | `replay/coverage.go:144-154`, `replay/origin.go:126-136`, `replay/build_score.go:17-25` | `TestCoverageSaysWhetherOriginIsResolved` (les deux sens) |
| R1-3 | les 3 appelants recoivent `port.ReplayFactsRepo` + assertion cote duckdb | `duckdb/replay_facts_repo.go:34`, `wire/registry_replay_build.go:145`, `replayartifacts/artifacts.go:207`, `cmd_backfill_replay.go:369` | assertion de compilation |
| R1-4 | code mort : `ParseOptionalValue` SUPPRIMEE ; `ScoreCurve` / `ScoreCurveFrom` / `PersonalScoreCurve` (+ `collectComponent`, `keepMonotoneBySlot`) descendues en helpers de test | `filmdec/bitreader.go`, `objectiveevents/score_instruments_test.go` (NEUF) | — |
| R1-5 | `build.go` **603 -> 585** (cablage dans `build_score.go`, `attachObjectiveActions` dans `objectives.go`), `document.go` **607 -> 539** (chronique v12 dans `document_score.go`, `NeutralDeath` dans `document_neutral_deaths.go`) | — | golden inchange |
| R1-6 | cinq familles de tests manquants | — | cf. ci-dessous |
| R1-7 | `RealRounds` : la contiguite tolere UN trou quand une manche coherente suit — les manches apres une manche 0 courte n etaient PAS publiees | `objectiveevents/statborg.go:400-455` | `TestRealRoundsGardeLesManchesApresUneManche0Courte` |

### R1-6 — les tests qui echoueraient si on inversait la condition

- **(a) branchement** : `replay/build_score_test.go` — avec entree, `scoreTimeline` ET
  `coverage.score` sont la ; sans entree, NI l un NI l autre. Le test a d abord ROUGI (les
  enregistrements synthetiques tombaient hors fenetre), ce qui prouve qu il mesure le cablage.
- **(b) ancrage** : `objectiveevents/statborg_guards_test.go` — un couple d en-tetes DEPAREILLE
  est refuse, un en-tete au-dela de `statMaxRound` est refuse. Vecteurs negatifs fabriques a
  partir des vecteurs REELS d A.1.0, un bit a la fois (`setBitsBE`).
- **(c) manches fantomes** : borne de domaine (valeur 2 104 rejetee + temoin sous la borne),
  contiguite (manche 5 isolee ecartee), seuil de coherence (emission isolee refusee).
- **(d) pont des faits** : `replaybuild/matchfacts_test.go` — ordre du triplet avec trois valeurs
  DISTINCTES, refus des deux cas (mode sans famille / aucune ligne) + temoin positif, camp -1
  ecarte et camp 0 conserve.
- **(e) courbes** : quatre compteurs EPINGLES a des valeurs distinctes deux a deux (le fixture a
  change pour cela), `coverage.score.points` egal au compte reel et non nul, et le cas
  Strongholds / KOTH — finales du film 200/126 contre registre 193/112 : (a) DECLINE, (b) resout,
  et la courbe publiee reste celle du FILM.

### Temoins re-cuits apres la ronde 1

| Temoin | `points` avant R1 | `points` apres R1 | repetitions restantes | oracle | `originResolved` | taille |
|---|---|---|---|---|---|---|
| `000d5950` Slayer | 960 | **708** (-26,3 %) | **0 / 656** | 43 / 50 | true | 2 270 742 -> 2 266 456 |
| `530820e5` CTF | 1 100 | **782** (-28,9 %) | **0 / 716** | 3 (le camp a 0 n emet rien) | true | 1 499 597 -> 1 494 172 |
| `24dbb67d` Oddball | 642 | **642** | **0 / 636** | 200 / 121 (100/78 puis 100/43) | true | 1 516 241 -> 1 516 263 |

L Oddball ne bouge pas, et c est coherent : il ne publie que des courbes d EQUIPE, dont le score
de mode est deja strictement croissant. Les valeurs d oracle, l identite des camps et les
183 actions d objectif de `530820e5` sont inchangees.

### Gates de la ronde 1

| Gate | Verdict |
|---|---|
| `go build ./...` (CGO) | `EXIT_BUILD=0` |
| `go vet ./internal/... ./contracttest/... ./cmd/...` | `EXIT_VET=0` |
| `go test ./internal/analysis/...` (CGO=0) | `EXIT_TEST_ANALYSIS=0` (16 paquets) |
| `go test ./internal/replaybuild/ ./contracttest/ ./internal/api/wire/ ./internal/games/halo_infinite/film/... ./internal/sync/replayartifacts/` | `EXIT_TEST_CHAINE=0` |
| `go test -tags=integration -run TestReplayFacts ./internal/platform/duckdb/` | `EXIT_TEST_DUCKDB_LECTEUR=0` |
| `golangci-lint run --new-from-merge-base=origin/main ./...` | `EXIT_LINT=0` (0 issue) |
| `npx tsc -b --force` | `EXIT_TSC=0` |

`generated.ts` et `openapi.yaml` ont bouge UNE fois (R1-2 : `Coverage` gagne `originResolved`),
et sont commites avec lui ; aucun drift depuis.
