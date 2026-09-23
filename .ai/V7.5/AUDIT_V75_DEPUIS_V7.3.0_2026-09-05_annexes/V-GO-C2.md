# Verification adverse V-GO-C2

HEAD `736ccf3c3`, branche `feat/v75`. Lecture seule. Aucun `go build/test/vet`, aucun npm.
Tous les fichiers cites ont ete rouverts a la ligne ; toutes les commandes ci-dessous ont ete
executees et leur sortie est reproduite.

---

## Constat 1 — aucun document de rejeu obtenu d'un film en CI : TIENT (enonce a preciser)

- **Ce que j'ai verifie**
  - `wc -l .../equivalence/minifilm.tsv .../equivalence/000d5950.tsv` →
    `8` et `50` lignes ; la premiere ligne de chacun est `# digest-grammar: 2` →
    **7 etapes contre 49**. Verifie sur pieces : `cut -f1 000d5950.tsv` rend bien 49 noms
    (`score`, `objectives`, ..., `artifact`) contre 7 dans `minifilm.tsv`
    (`fire`, `grenades`, `loadouts`, `inventory`, `deaths`, `playerIndices`, `projectiles`).
  - Appels REELS de `BuildFromFilm(` dans tout le module (grep, hors commentaires) : **11 sites**,
    dont 1 definition, 1 appel de production (`internal/replaybuild/replaybuild.go:239`), 1 appel
    d'outil (`cmd/zone-attribution/measure.go:95`), 1 shim de test
    (`internal/analysis/replay/film_shims_test.go:48`, `buildFromFilmDir`, sans appelant),
    3 dans des `*_research_test.go` **tous sous `os.Getenv("PICKUP_FILM")` + `t.Skip`**
    (`equipment_changes_film_research_test.go:21-23`, `weapon_changes_film_research_test.go:19-21`
    et `:71-73`), 1 lecture AST (`observe_test.go`), et **les 2 seuls appels executes en CI dans
    le paquet** :
    - `internal/analysis/replay/world_object_precision_guard_test.go:82` →
      `if _, err := BuildFromFilm(...); err == nil { t.Fatal(...) }` ;
    - `internal/analysis/replay/zero_disque_test.go:127-138` → `switch { case err == nil: t.Fatal ;
      case err.Error() != attendu: t.Fatalf }` avec `attendu = "aucun slot biped (ti=%d) dans les
      keyframes du film"`.
  - Seule cuisson reelle en CI : `internal/api/wire/build_queue_worker_binary_integration_test.go`
    (`//go:build integration && cgo`). Assertions de contenu, relues dans
    `assertArtefactLivreEtComplet` : `doc.SchemaVersion != replay.SchemaVersion || len(doc.Tracks)
    == 0`, `doc.ScoreTimeline == nil || len(doc.ScoreTimeline.Players) == 0`, `doc.Coverage == nil
    || doc.Coverage.Score == nil`. Les grandeurs « 5 joueurs de courbe de score et 92 actions
    d'objectif nommees » sont dans l'en-tete de la fonction et dans un `t.Logf`, **jamais dans un
    `if`**. S'y ajoute une preuve d'identite reelle : sha256 declare par l'ouvrier dans son
    `ResultJSON` vs sha256 recalcule sur l'artefact range.
- **Ce que j'ai cherche a decharge, sans le trouver**
  - `cmd/replay-worker/job_test.go` : son en-tete l'ecrit lui-meme — « le TRAVAIL de l'ouvrier
    exerce SANS decoder un vrai film [...] ici, aucun octet de film ». Zero cuisson.
  - `internal/replaybuild/` : 49 fonctions `Test*` relevees ; les seules qui touchent un film
    (`grenade_join_corpus_test.go`, `grenade_ambigu_sweep_test.go`) sont sous `REPLAY_CORPUS` /
    `FILM_SWEEP`.
  - `internal/ops/build_queue_cgo_test.go`, `internal/api/handlers/build_queue_e2e_cgo_test.go`,
    `internal/api/wire/build_queue_transport_e2e_cgo_test.go`,
    `internal/api/wire/registry_build_queue_cgo_test.go`,
    `internal/service/replay_map_background_registre_cgo_test.go` : `grep -n "chunk\|film\|Film"`
    ne rend que des commentaires et un `FilmShortMatchID` ; **aucun n'ouvre un octet de film**.
  - `grep -rn "film_e2e" --include='*.go'` → **3 lignes, toutes dans le meme fichier `wire`**.
    Un seul lecteur, confirme.
- **Nuance a porter au constat (l'auditeur se contredit dans son propre titre)** : la premiere
  clause « Aucun test de CI n'obtient un `ReplayDocument` a partir d'un film » est **litteralement
  fausse** — l'e2e `wire` en obtient un (`service.NewReplayService(...).GetReplay(...)` rend un
  `replay.ReplayDocument`, `internal/service/replay_service.go:110`) apres cuisson par le binaire
  ouvrier sur le film versionne. La mesure exacte est celle des « Chiffres mesures » du rapport
  (« dans son propre paquet : 2 appels, 0 document »), pas celle du titre.
- **Consequence reelle reformulee** : 42 des 49 balayages de `BuildFromFilm` n'ont aucune reference
  chiffree en CI, et la seule cuisson complete de la CI ne verifie que « non vide » plus l'identite
  octet a octet — l'enonce du titre doit etre reecrit « aucun test de CI n'assert de VALEUR sur un
  document obtenu d'un film ».

---

## Constat 2 — l'oracle API de la fixture e2e jamais confronte au film : REFUTE

- **Ce que j'ai verifie**
  - `internal/api/wire/testdata/film_e2e/c0a82e88/fixture.json` porte bien
    `facts.players[]` avec `kills`/`deaths`/`assists` par xuid (somme 20/20) et
    `facts.teamScores: [3, 0]`, et le test les charge en ENTREE
    (`chargerFixture` → `port.MatchPlayerFact{XUID, Kills, Deaths, Assists, TeamID}`).
  - `internal/analysis/replay/score_timeline.go:326-331` : `PlayerScore.Kills` vaut
    `kills.at(slot, c)` avec `kills := loadScoreSeries(recs, objectiveevents.KillsComponent, false)`
    — les compteurs viennent bien du **statborg DECODE DU FILM**, l'auditeur a raison sur ce point.
- **CE QUE L'AUDITEUR N'A PAS VU — la confrontation est faite, et le test asserte dessus**
  - `internal/analysis/replay/score_timeline.go:183` :
    `identity := objectiveevents.SlotIdentityFrom(recs, in.Lines)`.
  - `internal/analysis/objectiveevents/slotidentity.go:82-101` — le pont d'identite est une
    **egalite exacte du triplet** entre les compteurs decodes du film et les lignes de match :
    ```
    kills   := countsOf(recs, statSlotKey{coreKillsComp, sideA}, b)   // FILM
    deaths  := countsOf(recs, statSlotKey{coreKillsComp, sideB}, b)   // FILM
    assists := countsOf(recs, statSlotKey{coreAssistsComp, sideA}, b) // FILM
    ...
    if l.Kills == kills[slot] && l.Deaths == deaths[slot] && l.Assists == assists[slot] {
    ```
    Un slot dont le triplet FILM ne retrouve pas exactement une ligne API n'est **pas publie**.
  - `internal/replaybuild/matchfacts.go:88` + `:252-264` : `in.Lines` est exactement
    `playerLines(facts)` — c'est-a-dire les `kills`/`deaths`/`assists` de `fixture.json`,
    recopies un a un depuis `port.MatchFacts`.
  - Donc l'assertion DURE de la CI —
    `if doc.ScoreTimeline == nil || len(doc.ScoreTimeline.Players) == 0 { t.Fatal("artefact livre
    SANS compteurs de joueur ...") }` — **ne peut passer que si le decodage du film reproduit
    exactement, au frag et a la mort pres, au moins un triplet de l'oracle API**. Les deux chaines
    NE SONT PAS non comparees : la production les compare, et le test rougit sur le resultat de
    cette comparaison. La mesure ecrite en en-tete (« 5 joueurs de courbe de score ... contre 0
    sans ») est le compte des triplets film qui ont retrouve leur ligne API.
  - Corollaire : la panne que le constat annonce comme passant vert — « une perte de canal de
    score » — fait au contraire **echouer** ce test (aucun triplet, `Players` vide, `t.Fatal`).
- **Residu honnete (P3, pas P1)** : `facts.teamScores: [3, 0]` n'est effectivement confronte a
  rien (`doc.Coverage.Score.TeamIdentity` reste `unresolved` sur ce film, et le test l'ecrit
  comme « informatif ») ; et l'assertion porte sur « au moins un » joueur, pas sur les 5 mesures.
  Figer `5` serait un durcissement legitime — ce n'est pas l'absence de differentiel affirmee.
- **Consequence reelle reformulee** : le differentiel film↔API des frags/morts existe, est
  execute en CI et fait echouer le test s'il se rompt ; seul le score de camp (3-0) reste hors
  assertion.

---

## Constat 3 — le gate « presence des tests » aveugle sur tout le perimetre v7.5 : TIENT

- **Ce que j'ai verifie**
  - `grep -c "analysis/replay\|replaybuild\|replayartifacts\|killcollector\|objectiveevents"
    .ai/baselines/tests_pre_migration.jsonl` → **`0`** (exit 1, aucune correspondance).
  - `scripts/check_test_baseline.sh:7-9` (controle 1 = presence), `:135-141`
    (`extract_test_names` filtre `"Action":"(pass|fail|skip)"` → un test qui devient `skip`
    permanent reste « present »), `:22-25` (controle 3 = paquet qui ne compile pas).
  - `.github/workflows/ci.yml:376` : la CI n'invoque QUE
    `check_test_baseline.sh tests --from-jsonl baseline_current.jsonl` — donc les controles 1, 2
    et 3, jamais le mode `coverage`.
- **Ce que j'ai cherche a decharge, sans le trouver**
  - Le **controle 4 annonce « Coverage par package »** (`check_test_baseline.sh:12`) est une **doc
    inversee** : `check_coverage()` (`:416-462`) ne lit que la ligne `^total:` de
    `go tool cover -func`, un **seul chiffre global**. Et il n'est de toute facon pas invoque par
    la CI.
  - Le ratchet reellement joue est `apps/go-api/scripts/coverage_check.sh` (`ci.yml:407-408`) :
    lui aussi `awk '/^total:/'`, **global**, tolerance 0,1 pt, contre
    `apps/go-api/coverage_baseline.txt` = `69.0`. Le commentaire `ci.yml:395-400` confirme la
    marge de 1,7 pt (« valeur mesuree passee a 70,7 % et baseline posee a 69.0 »).
  - Aucun cliquet de comptage de tests par paquet nulle part :
    `grep -rn "func Test.*Freeze\|func Test.*Gel\|nombre de tests\|compte de tests"
    internal/archlint/*_test.go` → **vide**. Les seuls gels du paquet portent sur des variables
    de paquet (`filmdec_package_vars_test.go`), des litteraux (`no_raw_start_time_literal_test.go`)
    et un compte de FICHIERS racine (`sync_root_freeze_test.go`) — jamais sur des tests.
- **Consequence reelle reformulee** : supprimer ou transformer en `t.Skip` un test des paquets
  `analysis/replay`, `replaybuild`, `replayartifacts`, `killcollector`, `objectiveevents` ne
  declenche aucun controle nominatif ; le seul filet restant est un pourcentage global qui dispose
  de 1,7 pt de marge documentee.

---

## Constat 4 — deux tables append-only hors des deux garde-rails ART : TIENT

- **Ce que j'ai verifie** (boucle du rapport, rejouee mot pour mot) :
  ```
  kill_positions            : no_art=0  append_only_state=0
  match_weapon_hit_distance : no_art=0  append_only_state=0
  match_kill_events         : no_art=1  append_only_state=1
  match_bomb_stats          : no_art=1  append_only_state=1
  match_weapon_shots        : no_art=1  append_only_state=1
  match_usage_players       : no_art=1  append_only_state=1
  match_usage_films         : no_art=1  append_only_state=1
  ```
  - `internal/sync/no_art_patterns_test.go:68` (`var tablesProtegees`) : la liste s'arrete a
    `match_bomb_stats`, aucune des deux tables n'y figure ni dans un commentaire d'exclusion
    motivee (les deux seules exclusions ecrites nomment `player_records_history`,
    `media_likes_history`, `media_match_associations_history`).
  - `internal/sync/append_only_state_guard_test.go:12` — l'en-tete exige pourtant
    « Ajouter une table ici a CHAQUE conversion append-only (campagne en cours) » ;
    `:26` (`var appendOnlyStateTables`) ne les contient pas.
  - `TestNoRawDeleteOnAppendOnlyTables` (`no_art_patterns_test.go:287-292`) construit sa regex
    **a partir de `tablesProtegees`** : une table hors liste est litteralement invisible du scan.
- **Ce que j'ai cherche a decharge, sans le trouver**
  - `internal/games/halo_infinite/migrations/shared_kill_positions_appendonly_test.go` (3 tests,
    tag `cgo`) : verrouille la SPEC de la migration (cle fonctionnelle, preservation des lignes
    H5, dedup de la vue `_latest`) — ce n'est pas un scan de source contre un writer futur.
    Idem `internal/migration/steps_shared_weapon_hit_distance_test.go`.
  - `internal/migration/append_only_rebuild_test.go` : 6 tests sur le MECANISME generique du swap
    (`RebuildAppendOnly`, `RecoverOrphan`, rollback), aucune liste de tables.
  - Cliquet derivant les tables des vues `_latest` : `grep -rn '_latest"' internal/archlint/*_test.go`
    → **vide**. Il n'existe pas.
  - `internal/sync/art_upsert_patterns_test.go` et `combat_write_guard_test.go` : `grep -n
    "kill_positions\|match_weapon_hit_distance"` → **aucune occurrence**.
  - Verifie aussi que l'enrolement serait GRATUIT aujourd'hui :
    `grep -rn "kill_positions\|match_weapon_hit_distance" --include='*.go' internal/ cmd/
    | grep -v _test.go | grep -iE "delete|on conflict|insert or"` → **vide**. Les deux persisters
    sont bien INSERT-only ; ajouter les deux noms ne demanderait aucune entree d'allowlist.
- **Consequence reelle reformulee** : un `DELETE FROM kill_positions` ou un
  `INSERT ... ON CONFLICT DO UPDATE` sur `match_weapon_hit_distance` ecrit demain dans le hot path
  ne fait rougir aucun des deux scans — et l'enrolement qui fermerait le trou ne coute rien.

---

## Constat 5 — liste « FERMEE » d'enveloppes `dir` incomplete : TIENT (gravite → P2)

- **Ce que j'ai verifie**
  - `internal/archlint/no_film_reread_test.go:249` (`var enveloppesInterditesEnProduction`) —
    47 noms releves, **ni `ScanFilmWeaponShots` ni `ScanFilmWeaponDamages`**.
  - Definitions : `internal/analysis/filmdec/weapon_hits.go:192`
    (`func ScanFilmWeaponShots(dir string, n int)`) et `:232`
    (`func ScanFilmWeaponDamages(dir string, reg *Registry, n int)`) — les deux bouclent
    `for c := 1; c <= n; c++ { data, err := ReadFilmChunk(dir, c) ... }`.
  - Appels : `internal/sync/killcollector/hits.go:118` et `:122`, dans un paquet listé
    `paquetsDeProduction` (`no_film_reread_test.go:284-292`), et dont l'allowlist
    `appelsDEnveloppeAutorises` (`:309-312`) ne declare que `ReadFilmChunk` et `CountFilmChunks`.
  - **Le trou est plus large que dit** : l'inventaire complet des appels `filmdec.*` non-test de
    `killcollector` rend aussi `filmdec.FilmWeaponHitDistance` et `filmdec.DetectFilmWorldRange`,
    dont les signatures sont `FilmWeaponHitDistance(dir string, ...)`
    (`weapon_hit_distance_resolver.go:129`) et
    `DetectFilmWorldRange(dir, catalogPath, mapNameOverride string)` (`:144`) — **quatre**
    enveloppes `dir` hors liste, pas deux.
  - Asymetrie confirmee : la regle 4 (`sitesZlibAutorises`) est declaree « VERIFIEE DANS LES DEUX
    SENS » contre un balayage du depot ; la regle 3 ne verifie que les entrees MORTES de son
    allowlist (`:353-360`), jamais la completude de la liste d'enveloppes.
- **CE QUE L'AUDITEUR N'A PAS VU — la sous-clause (a) est fausse**
  - `.ai/V7.5/REGISTRE_REPORTS.md:15` porte le report, date, et **nomme les cinq appels**, ces
    deux-la compris : « 5 decompressions du film entier par match : `ReadFilmChunk`,
    `CountFilmChunks`, `ScanFilmWeaponShots`, `ScanFilmWeaponDamages`, `BuildBipedTracks` — plus
    une analyse de registre a lui seul », avec condition de reprise explicite (« le lot qui
    rallume la precision par arme [...] doit alors donner leur forme `Scan*(film)` a
    `ScanFilmWeaponShots`, `ScanFilmWeaponDamages` et `BuildBipedTracks` [...] et RETIRER les trois
    entrees d'allowlist »). L'ardoise de dette n'est donc **pas** comptee sur 2 appels au lieu de
    4 : elle est comptee sur 5, ailleurs que dans le fichier de test.
  - La raison ecrite existe et se verifie : `grep -rn "ConfigureFilmAccuracy" --include='*.go' .`
    → **aucun appelant de production** (uniquement `collector.go` qui la definit,
    `hits_capability_test.go` qui l'appelle, et des commentaires) ; et
    `config/titles/halo_infinite/mappings/capabilities.toml:80` :
    `"match.weapon.accuracy" = "not_exposed"`. La passe **ne s'execute pas** en production : le
    cout des relectures est nul aujourd'hui.
- **Consequence reelle reformulee** : la liste declaree FERMEE ne l'est pas et n'a aucun controle
  de completude, si bien qu'un futur appelant de production de `ScanFilmWeaponShots(dir, n)` ne
  ferait rougir personne — mais la dette existante est correctement inventoriee au registre et
  son cout d'execution est nul tant que la capability reste `not_exposed`. C'est un defaut de
  cliquet, pas une regression active : **P2**.

---

## Constat 6 — le garde auth ADR 0023 ne balaie que la racine de `internal/sync/` : REFUTE

- **Ce que j'ai verifie (le fait brut est exact)**
  - `internal/sync/no_legacy_source_used_test.go:37` : `entries, err := os.ReadDir(".")`, puis
    `:44` `if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go")
    { continue }`. Les sous-paquets ne sont effectivement jamais lus.
  - `internal/archlint/sync_root_freeze_test.go` gele bien le compte de fichiers racine, donc
    pousse le neuf vers les sous-paquets.
- **CE QUE L'AUDITEUR N'A PAS VU — le second garde-rail que CLAUDE.md nomme dans la meme phrase
  balaie TOUT le module**
  - `internal/platform/auth/sentinel_test.go:54-96` — `scanProductionGoFiles` fait
    `filepath.Walk(apiRoot, ...)` sur `apps/go-api` ENTIER (seuls `vendor`, `.git`,
    `node_modules`, `tmp` sont sautes), hors `_test.go`, avec anti-pourrissement
    (`if scanned == 0 { t.Fatal }`). Il est sans build tag — « execute en CI normale » (en-tete).
    Il porte cinq gardes, dont trois couvrent exactement les vecteurs que le constat cite :
    - **Guard 1** (`:110-121`) : le LITTERAL `SPNKR_OAUTH_REFRESH_TOKEN`, detection large assumee
      — « lecture, concatenation, format, ou simple mention » ; allowlist datee a 3 entrees.
    - **Guard 2** (`:138-147`) : `\bEnvRefreshTokenForGamertag\(`, 2 appelants autorises.
    - **Guard 3** (`:164-176`) : `\b\w+\.ReadOAuthRefreshToken\b`, **independant de l'alias
      d'import** (« le package est importe sous 5 noms differents ») — c'est-a-dire le **dernier
      lecteur existant** de `sync_meta.oauth_refresh_token`, allowlist a 2 entrees datees
      2026-10-01.
  - Autrement dit, deux des trois sources que le constat nomme
    (`SPNKR_OAUTH_REFRESH_TOKEN_*` et `sync_meta.oauth_refresh_token`) sont gardees dans
    **chaque sous-paquet de `internal/sync/`**, y compris ceux crees demain. La conclusion du
    constat — « l'invariant le plus sensible du depot est protege sur une surface qui retrecit a
    chaque extraction du god-package » — est fausse : la surface du garde principal est le module.
  - Verifie aussi que les autres motifs de la liste racine sont largement inertes :
    `grep -rn "RecordLegacySourceUsed\|MSALCacheJSON" --include='*.go' internal/ cmd/ | grep -v
    _test.go` → **vide dans les deux cas** ; les symboles n'existent plus, les reintroduire
    demanderait de les redefinir (l'en-tete du guard 3 l'ecrit : « leur simple reapparition ne
    compilerait pas »).
- **Residu honnete (P3)** : un fichier neuf d'un sous-paquet de `sync` qui ecrirait du SQL BRUT
  sur la cle `sync_meta` (`oauth_refresh_token` / `msal_token_cache`) sans passer par
  `ReadOAuthRefreshToken` echapperait aux deux gardes. Elargir le `os.ReadDir(".")` en
  `filepath.WalkDir` reste souhaitable — mais c'est un durcissement de bord, pas un P1.
- **Consequence reelle reformulee** : le perimetre etroit du garde de `sync` est reel, mais les
  vecteurs qu'il vise sont deja couverts a l'echelle du module par `platform/auth/sentinel_test.go` ;
  seul le SQL brut sur la cle `sync_meta` reste hors filet.

---

## Constat 7 — `ReplayPurgeCron.RunOnce` sans aucun test : TIENT

- **Ce que j'ai verifie**
  - `grep -c "RunOnce\|ReplayPurgeCron" internal/scheduler/replay_purge_cron_test.go` → **`0`**.
    Le fichier ne contient que `prepareReplayPurgeFixture` (helper), `TestPurgeReplayArtifacts:57`
    et `TestPurgeReplayArtifacts_DossierAbsent:80`, tous deux sur
    `purgeReplayArtifactsForTitle(...)`.
  - `grep -rn "RunOnce(" --include='*_test.go' .` sur tout le module : les appelants sont
    `asset_name_sweep_cron_test.go` (5 sites), `auto_sync_*_test.go`, `data_health_*` — **aucun
    pour `ReplayPurgeCron`**. `grep -rn "ReplayPurgeCron\|purgeReplayArtifacts"` hors du fichier
    source rend exactement 3 lignes : `cmd/server/main.go:1419` (cablage) et les 2 appels au
    helper dans le test.
  - Le code non couvert, relu : `internal/scheduler/replay_purge_cron.go:93-97`
    (`months := c.retention()` ; `if months <= 0 { ReportCronRun(...); return }`), `:99`
    (`cutoff := time.Now().UTC().AddDate(0, -months, 0)`), la boucle `for _, desc := range
    reg.Active()` (`:109`), le compteur `observability.AddInt("replay_purge_artifacts_deleted_total",
    ...)` (`:125`) et le `ReportCronRun` final. Le premier tick est bien immediat (`:72`
    `c.RunOnce(ctx)` avant le ticker).
  - Le reglage « 0 = illimite » est bien le defaut cable : `cmd/server/main.go:1419-1424`
    retourne `0` si le store est vide, et `internal/domain/settings.go:53` le documente.
  - **La godoc se dement elle-meme** : `:87` — « RunOnce execute un cycle complet (**exporte pour
    les tests**) », alors qu'aucun test ne l'appelle.
- **Ce que j'ai cherche a decharge, sans le trouver** : aucun test d'integration ailleurs
  (`internal/ops`, `internal/api/wire`, `cmd/server`) n'appelle `RunOnce` sur ce cron ; le patron
  existe pourtant a cote (`asset_name_sweep_cron_test.go:21,44,62,74,88`).
- **Consequence reelle reformulee** : le seul cron du depot qui supprime des fichiers a sa garde
  « retention 0 = ne rien toucher », sa boucle par titre et sa telemetrie entierement hors filet ;
  inverser cette garde fait tomber `cutoff` sur `time.Now()` et purge tous les artefacts datables
  au premier tick, CI verte.

---

## Bilan : 4 tiennent, 2 refutes, 1 requalifie

| # | Constat | Verdict |
|---|---|---|
| 1 | Aucun document obtenu d'un film en CI | **TIENT** (titre a reecrire : « aucune assertion de VALEUR » — un document EST obtenu par l'e2e `wire`) |
| 2 | Oracle API de la fixture jamais confronte | **REFUTE** — `SlotIdentityFrom` compare les triplets film↔API et la CI asserte (`t.Fatal`) sur le resultat |
| 3 | Gate de presence aveugle sur le perimetre v7.5 | **TIENT** (le controle 4 « par package » est une doc inversee : il est global, et la CI ne l'invoque pas) |
| 4 | 2 tables append-only hors des deux gardes ART | **TIENT** (enrolement gratuit : aucune violation existante) |
| 5 | Liste « FERMEE » d'enveloppes `dir` incomplete | **TIENT, gravite → P2** — le trou est reel (et vaut 4 enveloppes, pas 2), mais la dette est inventoriee au `REGISTRE_REPORTS.md:15` et la passe est inerte (`not_exposed`) |
| 6 | Garde auth limite a la racine de `sync/` | **REFUTE** — `platform/auth/sentinel_test.go` fait `filepath.Walk` sur tout `apps/go-api` ; residu P3 : SQL brut sur la cle `sync_meta` |
| 7 | `ReplayPurgeCron.RunOnce` sans test | **TIENT** (aggrave : la godoc dit « exporte pour les tests ») |
