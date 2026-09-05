# Lot A — Faits du film (Go) — journal d'exécution

> Plan : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`, lot A. Branche `feat/v2-faits`, worktree
> `LevelUp-wt-v2-faits`. Contrat : skill `plan-execution`.
> Statuts : `[x]` fait et vérifié · `[~]` couvert ailleurs (réf.) · `[!]` non traité (justification).

## Tâche A-I

### [x] A.1 (P0-1) — révision du décodeur bumpée, garde-rail d'empreinte posé

**Vérification sur pièces avant de coder.** `internal/sync/killcollector/collector.go:65` portait
bien `KillSourceDecoderRev = "killsource-2026-07-31"`. Mesure du constat rejouée sur la branche :

```
git log --oneline v7.3.0..HEAD -- apps/go-api/internal/games/halo_infinite/film/killsource/ | wc -l
  -> 14
git log -G'KillSourceDecoderRev = ' --oneline v7.3.0..HEAD -- apps/go-api/internal/sync/killcollector/collector.go
  -> 36fc76835 refactor(sync): J4 — le collecteur va dans son sous-paquet  (déplacement de paquet, PAS un bump)
```

**Fait.** `KillSourceDecoderRev = "killsource-2026-09-05"`. Constante `killSourceDecoderFingerprint`
figée juste en dessous (sha256 des 20 sources non-test du paquet décodeur). Nouveau test
`internal/sync/killcollector/decoder_rev_fingerprint_test.go` :

- `TestKillSourceDecoderRevSuitLeDecodeur` — hache
  `internal/games/halo_infinite/film/killsource/` (récursif, `_test.go` et `testdata/` exclus,
  chemin relatif haché à côté du contenu, CRLF normalisés en LF pour que le gate soit identique
  sur la CI Linux et sur un poste Windows) et compare à la constante. Message d'échec : les DEUX
  gestes à faire (bumper la révision, recopier l'empreinte mesurée), avec la conséquence de
  chacun.
- `TestEmpreinteSourcesGoMord` — la preuve que le garde-rail mord, sur quatre propriétés : un
  octet de source changé fait bouger l'empreinte ; un renommage aussi ; un `_test.go` ou une
  fixture `testdata/` ne la bouge PAS ; le même contenu en CRLF donne la même valeur.

**Preuve par mutation manuelle** (exigée par le plan) :

```
printf '\n// mutation temporaire...\n' >> internal/games/halo_infinite/film/killsource/doc.go
go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur
  -> FAIL   attendue b272f221909247fd6f8e2c1cca01d4136ec9b317fbecd31488e3eda48bc59379
            mesurée  2f29297b39d9e59557efb68e4623bad41afaa6414d52b7390c35cbd52ac68d5c
git checkout -- apps/go-api/internal/games/halo_infinite/film/killsource/doc.go
go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur
  -> ok  levelup/go-api/internal/sync/killcollector  0.118s
```

**Question posée par le plan : les familles « tirs » et « positions » partagent-elles cette
révision pour leur REPRISE ?** Réponse mesurée : **elles n'ont aucune reprise du tout**. Le dépôt
ne contient que DEUX prédicats de reprise, et les deux lisent la même colonne :

| Site | Prédicat |
|---|---|
| `internal/sync/killcollector/postsync.go:375` (`conditionBacklog`) | `match_kill_events_latest.decoder_rev = KillSourceDecoderRev AND read_path <> credit-backfill` |
| `cmd/levelup/cmd_backfill_killsource.go:403` (`matchsAJour`) | idem |

Vérification : `grep -rn "decoder_rev" --include=*.go` hors tests ne rend, en contexte SQL, que ces
deux sites plus une migration (`steps_shared_kill_events_credit_base.go:136`, agrégat de rebuild).
Conséquences, toutes documentées ici et **non traitées dans ce lot** (le plan l'interdit
explicitement : « pas de nouvelle révision dans ce lot ») :

- `match_weapon_shots` porte sa propre constante `WeaponShotsDecoderRev = "filmshots-2026-08-01"`
  (`shots.go:49`), écrite sur chaque ligne — mais **aucun prédicat ne la lit**. La passe de tirs
  est une PASSAGÈRE de la passe de morts (un film téléchargé, quatre tables écrites,
  `collector.go:291`). Un changement du seul producteur de tirs ne déclenchera donc jamais de
  redécodage ; inversement le bump ci-dessus rend les tirs de tous les matchs à nouveau candidats.
- `match_weapon_hit_distance` porte `WeaponHitDistanceDecoderRev = "whd-v1"`
  (`internal/migration/steps_shared_weapon_hit_distance.go:75`) — même situation : écrite, jamais
  lue par une reprise.
- `kill_positions` **n'a pas de colonne `decoder_rev` du tout**, et c'est documenté comme un choix
  (`internal/persist/kill_position_persister.go:18` : la clé fonctionnelle + `written_at`
  suffisent). Elle n'a donc ni révision ni reprise propre : passagère elle aussi.

Autrement dit : les trois familles dépendent aujourd'hui de `KillSourceDecoderRev` pour être
redécodées, sans le dire nulle part. Le traitement (« une révision par famille de lignes »,
proposé par l'audit P0-1) reste ouvert — consigné en découverte D-1 ci-dessous.

**Gate A.1** (commandes exactes, dernière ligne de sortie) :

```
go build ./...                                       -> (aucune sortie) EXIT=0
go test ./internal/sync/...                          -> ok levelup/go-api/internal/sync/v2  15.427s   EXIT=0
golangci-lint run ./internal/sync/killcollector/...  -> 0 issues.  EXIT=0
```

### [x] A.2 (G4) — `kill_positions` et `match_weapon_hit_distance` enrôlées dans les deux listes

**Vérification sur pièces avant enrôlement** (les deux tables doivent être append-only AVEC vue
`_latest`, sinon l'enrôlement serait un mensonge) :

| Table | Migration | Forme | Vue `_latest` | Écrivains |
|---|---|---|---|---|
| `kill_positions` | `games/halo_infinite/migrations/steps_appendonly_misc.go:53` (rebuild G.2, 2026-08-30) | id PK `kill_positions_seq` + `written_at` | `kill_positions_latest`, `QUALIFY ROW_NUMBER() … PARTITION BY (match_id, killer_xuid, time_ms)` | `persist/kill_position_persister.go` (film Infinite) et `persist/shared_persister.go` `persistKillPositions` (builder Halo 5) — **INSERT purs tous les deux** |
| `match_weapon_hit_distance` | `migration/steps_shared_weapon_hit_distance.go:104` (créée append-only) | id PK seq + `decode_pass` + `decoder_rev` + `written_at` | `match_weapon_hit_distance_latest`, dernière PASSE par match | `persist/weapon_hit_distance_persister.go` — un seul statement, INSERT pur |

Note de forme : `kill_positions` n'a volontairement pas de `decoder_rev` (`written_at` arbitre,
cf. l'en-tête du persister) ; l'unité de génération y est LA LIGNE, alors que
`match_weapon_hit_distance` supersède par PASSE entière. Les deux formes sont couvertes par les
mêmes garde-rails (aucun n'inspecte la clé de partition).

**Fait.** Les deux noms ajoutés à `tablesProtegees` (`internal/sync/no_art_patterns_test.go`) et à
`appendOnlyStateTables` (`internal/sync/append_only_state_guard_test.go`), avec la justification
et la date en commentaire. **Aucune allowlist agrandie** : `allowlistArtPatterns`,
`allowlistRawDelete` et `allowlistMediaMutation` restent vides. Le `\b` final des motifs ne
déborde pas sur les vues (`_` est un caractère de mot) : les lectures via `_latest` ne sont pas
touchées.

**Preuve que l'enrôlement mord** (deux violations injectées, puis annulées) :

```
# ajout temporaire d'un `DELETE FROM kill_positions …` dans persist/kill_position_persister.go
# et d'un `INSERT INTO match_weapon_hit_distance … ON CONFLICT (…) DO UPDATE …` dans
# persist/weapon_hit_distance_persister.go
go test ./internal/sync/ -run 'TestNoRawDeleteOnAppendOnlyTables|TestNoMutationOnAppendOnlyStateTables|TestNoARTPatternsOnProtectedTables'
  -> FAIL  - DELETE FROM kill_positions dans internal/persist/kill_position_persister.go
           - INSERT ON CONFLICT/REPLACE/IGNORE sur match_weapon_hit_distance dans internal/persist/weapon_hit_distance_persister.go
           - table=match_weapon_hit_distance pattern_detected file=internal/persist/weapon_hit_distance_persister.go
           - table=kill_positions DELETE brut file=internal/persist/kill_position_persister.go
git checkout -- (les deux fichiers)
  -> ok  levelup/go-api/internal/sync  74.215s
```

**Gate A.2** :

```
go test ./internal/sync/ -run 'ART|AppendOnly|Mutation|Allowlist|Delete|Bulk' -v
  -> ok  levelup/go-api/internal/sync  44.897s   (11 tests, 0 FAIL, 0 SKIP)
golangci-lint run --new-from-merge-base=origin/main ./internal/sync/...
  -> 0 issues.  EXIT=0
```

Note : `golangci-lint run ./internal/sync/` NON ratcheté remonte 15 problèmes, tous
**préexistants** (goconst `weapon_kills`/`match_registry`, argument-limit de `citations.go` et
`engine_v2bridge.go`, `unused` de `convergence.go`/`engine_e2e_test.go`, SA4006 de `engine.go`) —
dette gelée par la baseline, aucune sur les deux fichiers modifiés. Le gate d'autorité est le
ratchet `--new-from-merge-base` (Makefile:307), vert.

### [x] A.3 (A0) — le runtime n'écrit plus le catalogue versionné

**Vérification sur pièces avant de coder.** `mvar_rattrapage.go` lisait
`PathResolver.MapWeaponPadsPath` (fichier suivi par git) et écrivait ce même chemin via
`mapcatalog.AddEntry` (un seul appelant de production, confirmé par grep). Le catalogue livré
fait 928 747 octets pour 72+ cartes.

**Conception retenue** (une entrée relue, une sortie jetable) :

| Rôle | Fichier | Producteur | Versionné |
|---|---|---|---|
| ENTRÉE relue en revue | `reference/map_weapon_pads.json` | `cmd/mapopads-build`, à la main | oui |
| SORTIE de runtime | `reference/generated/map_weapon_pads.json` | rattrapage `.mvar` au fetch de film | **non** (`.gitignore`) |

- `PathResolver.MapWeaponPadsOverlayPath(slug)` — nouveau chemin, documenté avec les deux dégâts
  qu'il évite (le `git reset --hard origin/main` de `scripts/deploy.sh` et le commit local qui
  avale +332 lignes de référence).
- `replay.LoadMapWeaponPadsMerged(versionne, overlay)` — la fusion À LA LECTURE, **l'entrée
  versionnée primant** sur la rattrapée. Overlay absent = cas nominal, silencieux ; overlay
  illisible = `slog.Warn` puis dégradation au seul versionné (un fichier jetable ne doit pas
  faire perdre les cartes relues) ; le versionné absent reste une erreur.
- `mapcatalog.AddEntry` → `mapcatalog.AddOverlayEntry(overlay, titleSlug, mapID, entry)` : elle
  ne connaît plus le chemin versionné du tout. Renversement de contrat assumé et testé :
  l'ancienne devait ÉCHOUER sur un fichier absent (créer un catalogue de zéro aurait effacé le
  titre) ; la nouvelle CRÉE l'overlay absent (c'est le premier rattrapage) mais refuse toujours
  d'écraser un overlay corrompu.
- Les trois lecteurs de production passent au chargeur fusionné : `mvar_rattrapage.go` (sinon une
  carte rattrapée serait re-téléchargée à chaque cycle), `replaybuild/spawnpoints.go` (la
  cuisson, destinataire réel du rattrapage) et `service/replay_map_weapon_pads.go` (le calque
  servi). `cmd/mapopads-build` continue de ne lire que le versionné : c'est lui qui le produit.
- `.gitignore` : `data/titles/*/reference/generated/`, avec la raison écrite. Vérifié :
  `git check-ignore -v data/titles/halo_infinite/reference/generated/map_weapon_pads.json`
  → `.gitignore:152`. **Le contenu déjà versionné n'a pas été touché** (`git status data/` vide).

**Garde-rail** : `internal/archlint/no_runtime_versioned_catalog_write_test.go`, trois tests.

1. `TestRuntimeNEcritPasLeCatalogueVersionne` — analyse **AST avec suivi de valeur** (pas un
   grep) sur tout `internal/` et `cmd/` non-test : repère `…MapWeaponPadsPath(…)`, suit la
   variable qui en reçoit le résultat dans la même fonction, et refuse que cette valeur atteigne
   un verbe d'écriture. Une valeur qui part dans un contexte non suivable (littéral composite,
   `return`, champ) est refusée aussi — un ratchet qui ne sait pas doit dire non. Exception
   unique et datée : `cmd/mapopads-build/` (la chaîne de fabrication, dont c'est le métier).
   **Les verbes d'écriture incluent les verbes FRANÇAIS** (`ajouter`, `ecrire`, `deposer`…) :
   sans eux, le ratchet aurait été vert sur `ajouterCarteAuCatalogue`, c'est-à-dire sur le défaut
   même qu'il doit garder.
2. `TestCatalogueVersionneNommeParLeResolverSeul` — ferme le contournement par le littéral :
   `"map_weapon_pads.json"` n'a le droit d'apparaître que dans `domain/title/registry.go`.
3. `TestRatchetCatalogueVersionneMord` — 5 sources refusées, 3 acceptées (dont la forme exacte de
   la production d'aujourd'hui, pour que le ratchet ne soit pas « vert par excès de sévérité »).

**Preuve par mutation sur le code réel** :

```
sed -i 's/ajouterCarteAuCatalogue(ctx, d, fetcher, overlayPath,/…, catPath,/' mvar_rattrapage.go
go test ./internal/archlint/ -run TestRuntimeNEcritPasLeCatalogueVersionne
  -> FAIL  internal/sync/replayartifacts/mvar_rattrapage.go ligne 153 : le chemin VERSIONNÉ
           (catPath) est passé à ajouterCarteAuCatalogue() — le runtime doit écrire l'overlay
(retour à overlayPath)
  -> ok  levelup/go-api/internal/archlint  0.309s
```

**Tests de comportement ajoutés** (au-delà du ratchet, qui ne lit que du code) :

- `mapcatalog` : `TestAddOverlayEntryNeTouchePasLeCatalogueVersionne` (contenu ET mtime du
  versionné inchangés après un rattrapage), `…CreeLOverlayAbsent`, `…EchoueSurOverlayCorrompu`,
  plus les trois tests d'origine portés sur l'overlay (refus de clé existante, ajout d'une clé
  neuve, concurrence à 8 écrivains).
- `analysis/replay` : quatre tests de `LoadMapWeaponPadsMerged` (overlay absent nominal, overlay
  complété, **le versionné prime**, overlay illisible → dégradation).
- `replayartifacts` : `TestRattrapageCarteInconnueAjouteSansToucherLesAutres` étendu (le fichier
  versionné ressort byte-identique, l'ajout est dans l'overlay) et
  `TestRattrapageCarteDejaDansLOverlayNeRetelechargePas` (deux cycles : le second ne fait AUCUN
  appel réseau — sans la lecture fusionnée, chaque cycle re-téléchargerait toutes les cartes
  déjà rattrapées).

## Gates de la tâche A-I

Tous joués en avant-plan, dans ce worktree, avec
`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-faits CGO_ENABLED=1` depuis `apps/go-api`.

| # | Commande | Dernière ligne |
|---|---|---|
| 1 | `go test ./internal/sync/... ./internal/analysis/replay/... ./internal/domain/... ./internal/archlint/...` | `ok levelup/go-api/internal/archlint 18.811s` — EXIT=0 |
| 2 | `go test ./internal/mapcatalog/... ./internal/replaybuild/... ./internal/service/...` (paquets touchés hors liste du plan) | `ok levelup/go-api/internal/service/teammates 0.529s` — EXIT=0 |
| 3 | `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` | `ok levelup/go-api/internal/persist 32.787s` — EXIT=0 |
| 4 | `golangci-lint run --timeout 5m --new-from-merge-base=origin/main ./internal/sync/... ./internal/analysis/replay/... ./internal/domain/title/... ./internal/mapcatalog/... ./internal/replaybuild/... ./internal/service/... ./internal/archlint/...` | `0 issues.` — EXIT=0 |
| 5 | `golangci-lint run --timeout 5m ./internal/mapcatalog/... ./internal/archlint/... ./internal/analysis/replay/...` (non ratcheté, paquets à code neuf) | `0 issues.` — EXIT=0 |
| 6 | `go build ./...` | (aucune sortie) — EXIT=0 |

Aucun test skippé, aucune variable d'environnement de film posée, aucune cuisson d'artefact,
aucune allowlist agrandie.

## Tâche A-II

### [x] A.4 (A1) — les dérivations se déclenchent sur « un artefact vient d'être rangé »

**Vérification sur pièces avant de coder** (le code avait bougé : les lignes citées par l'audit
sont désormais 321-322 → `artifacts.go` autour de 320) :

- `artifacts.go` : `if d.Placement == replaybuild.PlacementWorker { enqueueAll(...); return }`
  précédait bien `reporterT0Film`, `persisterResumesUsage`, `persisterStatsBombe`.
- `api/wire/registry_build_queue.go` `StoreBuildArtifact` : validait, rangeait, comptait,
  journalisait, `return`. **Aucune dérivation.**
- `replaybuild/placement.go` : `local` est REFUSÉ en production (`ErrLocalBuildInProduction`) et
  le défaut y est `worker` — la branche des dérivations est donc inatteignable en prod par
  construction, pas seulement par défaut.
- Le puits `SetArtifactStoredSink` existe mais est mono-emplacement et déjà pris par la
  notification Discord ; le ratchet `archlint/no_second_artifact_sink_test.go` interdit un second
  câblage. Il ne pouvait donc PAS porter les dérivations — d'où un point d'entrée dédié.

**Fait.** `internal/sync/replayartifacts/derivations.go` : `ArtefactRange` (le fait « un artefact
vient d'être rangé »), `DerivationsDeps` (sous-ensemble strict de `Deps` — le chemin ouvrier n'a
ni client film, ni segment de lecture, ni placement) et `Deriver`, appelé des DEUX rangeurs :

| Rangeur | Site d'appel |
|---|---|
| cuisson locale | `artifacts.go` `Run` → `Deriver(ctx, DerivationsDeps{...}, b.ranges)` |
| dépôt d'ouvrier | `registry_build_queue.go` `StoreBuildArtifact` → `r.deriverArtefactRange(...)` |

Côté wire : `sharedWriterForTitle(slug)` résout le provider **par titre**
(`cfg.SharedManager.For(SharedDBPath(slug))`, jamais `cfg.SharedProvider` nu qui est celui du
titre par défaut) et borne l'acquisition par `acquireWriterTimeout`, comme les actions admin.
Seam `replayDerivationsFn` (nil en production, même parti pris que `replayJobFactsFn`).

**Gain collatéral mesuré à la lecture** : les trois dérivations ouvraient CHACUNE l'artefact
(3 `os.ReadFile` + 2 désérialisations d'un document de ~2 Mo par match). `Deriver` lit et
désérialise **une fois** et passe le document aux projections ; `lireT0FilmArtefact`,
`projeterResumeUsage` et la lecture de `projeterStatsBombe` disparaissent avec leurs chemins
d'erreur (un document illisible est écarté à la lecture, journalisé, une seule fois).

**Test qui prouve l'appel sur le chemin ouvrier** :
`internal/api/wire/build_queue_derivations_cgo_test.go` (store monitoring DuckDB réel, rangement
par `replaybuild.StoreArtifact`) — `TestStoreBuildArtifact_DeclencheLesDerivations` vérifie
l'appel unique, le slug, l'identité du **job** (pas celle du document) et le chemin **rangé** ;
`TestStoreBuildArtifact_ArtefactRefuseNeDerivePas` vérifie qu'un dépôt refusé ne dérive rien.

**Preuve par mutation** :

```
# `r.deriverArtefactRange(...)` remplacé par `_ = stored.Path` dans StoreBuildArtifact
go test ./internal/api/wire/ -run TestStoreBuildArtifact_DeclencheLesDerivations
  -> FAIL  dérivations appelées 0 fois pour UN artefact rangé, attendu 1 —
           c'est exactement le maillon du constat A1
(retour à l'appel)
  -> ok  levelup/go-api/internal/api/wire  0.193s
```

**Gate A.4** :

```
go build ./...                                                     -> EXIT=0
go test ./internal/sync/... ./internal/api/... ./internal/archlint/...
  -> ok levelup/go-api/internal/archlint  17.636s   EXIT=0
go test -tags=integration -p 1 ./internal/sync/replayartifacts/...
  -> ok levelup/go-api/internal/sync/replayartifacts  16.085s   EXIT=0
golangci-lint run --new-from-merge-base=origin/main ./internal/sync/... ./internal/api/wire/...
  -> 0 issues.  EXIT=0
```

### [x] A.5 (A2) — un rattrapage unique des dérivés, à prédicat de digest

**Vérification sur pièces.** `backlog.go` `artefactPresent` (`os.Stat` + taille > 0) était bien
le seul filtre du rattrapage, et sa justification écrite tient toujours pour la CUISSON (le
prédicat complet lit un document de ~2 Mo, « ruineux sur soixante-quatre à chaque cycle »).
Il ne dit en revanche RIEN des dérivés — c'est le constat A2.

**Fait.** `internal/replaybuild/derivations_index.go` : l'index des dérivations, à côté du
digest de l'artefact.

- `DerivationsRev = "derivations-2026-09-06"` — à bumper quand ce que les dérivations ÉCRIVENT
  change (même doctrine que `KillSourceDecoderRev`) ; le rattrapage rejoue alors tout le parc,
  cinq artefacts par cycle.
- `DerivationsMark` (`rev`, `artifactBytes`, `artifactSchema`, `at`) sérialisée dans
  `<artefact>.derived.json` — un fichier À CÔTÉ, pas une colonne : le rattrapage tourne AVANT
  tout writer et le backfill CLI tourne sans base du tout.
- `DerivationsUpToDate(path)` = artefact présent ET marque présente ET même révision ET **même
  taille**. La taille (et non un hachage) parce qu'elle vient du `os.Stat` qu'on fait de toute
  façon : le prédicat reste jouable sur les 64 candidats d'un horizon à chaque cycle. Le
  compromis (un artefact re-cuit à la taille identique à l'octet près) est écrit dans l'en-tête,
  et le prochain bump de révision le rattrape.
- `RemoveDerivationsMark` : le geste « redemande les dérivations de ce match ».

`internal/sync/replayartifacts/derivations_backlog.go` : `rattraperDerivations` +
`candidatsDerivations`, appelés depuis `Run` **sur les DEUX placements** (en placement
« ouvrier » ce process ne cuit rien, mais les artefacts déposés sont sur le disque).

- Il **ne cuit rien** : il ne sélectionne que des artefacts DÉJÀ RANGÉS et n'appelle jamais la
  cuisson. Un artefact PÉRIMÉ est dérivé **tel quel** — la re-cuisson du corpus local (106
  artefacts, 9 versions de schéma) est l'arbitrage utilisateur daté du 2026-09-02 (registre des
  reports l. 17), que ce lot ne renverse pas.
- Bornes : l'horizon existant (`BacklogHorizon` = 64, **la même requête** que le rattrapage de
  cuisson — `requeteQueueRecente`) et le plafond existant (`maxPerCycle` = 5).
- `fenetreRetention(months)` extrait dans `backlog.go` : le calcul de la borne basse était sur
  le point d'exister en deux exemplaires (règle des ≤ 2 copies appliquée avant la 2e).
- Compteurs : `postsync_replay_derivations_rattrapees_total` et la jauge
  `postsync_replay_derivations_retard`, publiée **même à zéro**.

`Deriver` pose la marque en fin de passe (`marquerDerivations`), **même quand rien n'a été
écrit** : un match hors Assaut ou un document sans `t0FilmMs` sont des dérivations JOUÉES, pas
manquantes.

**Tests** : `internal/replaybuild/derivations_index_test.go` (7 cas du prédicat : artefact
absent, artefact vide, marque absente, marque illisible, marque courante, artefact re-cuit,
révision antérieure) et 5 tests d'intégration dans `backlog_integration_test.go` — dont
**`TestCandidatsDerivations_ConvergeApresDerivation`, la propriété qui fait tout tenir** : sans
marque, les mêmes cinq artefacts reviendraient à chaque cycle indéfiniment.

**Piège rencontré et corrigé** : `FilmShortMatchID` coupe au premier `-`, donc des identifiants
de test `m-1`, `m-2`… écrivent TOUS le même fichier d'artefact. Un des tests passait par
coïncidence ; les identifiants ont été refaits sans tiret.

**Gate A.5** :

```
go build ./...                                                            -> EXIT=0
go test ./internal/replaybuild/ -run Derivations -v                       -> 10 sous-tests PASS
go test -tags=integration -p 1 ./internal/sync/replayartifacts/... ./internal/replaybuild/...
  -> ok levelup/go-api/internal/replaybuild  0.532s   EXIT=0
golangci-lint run --new-from-merge-base=origin/main ./internal/sync/... ./internal/replaybuild/...
  -> 0 issues.  EXIT=0
```

### [x] A.6 (décision 1) — `match_player_positions` devient une projection de l'artefact

**Vérification sur pièces.** La table était écrite UNIQUEMENT par
`cmd/diag_weapons_v3 -positions -write` → `PlayerPositionsRepo.WriteMatch` (DELETE+INSERT dans
une transaction, sur le handle de LECTURE du pool). Sans PK, sans vue `_latest`. Lecteur unique :
`LoadMatch` → `GET /matches/{id}/positions` → `MatchPositionsHeatmap.tsx`.

**Fait, en quatre pièces :**

1. **Migration** `internal/migration/steps_shared_player_positions_appendonly.go` : rebuild CTAS
   (`ApplyAppendOnlyRebuild`) → `id` PK + `positions_pass` + vue `match_player_positions_latest`
   **par PASSE** (`QUALIFY positions_pass = FIRST_VALUE(...) OVER (PARTITION BY match_id ORDER BY
   written_at DESC, id DESC)`). Les lignes déjà en base reçoivent
   `positions_pass = 'legacy-diag'` : elles forment UNE génération cohérente au lieu d'être
   éclatées en autant de « passes » d'une ligne. Inscrite dans `order.go` juste après son
   créateur.
2. **Persister** `internal/persist/player_positions_persister.go`, sur le patron EXACT de
   `bomb_stats_persister.go` : une transaction, INSERT purs, `positions_pass` + `written_at`
   partagés par toute la passe (`newDecodePassID`, le même générateur que les autres passes),
   matchID vide refusé, passe vide ignorée ET journalisée. **La reprise sur un match déjà
   projeté écrit une NOUVELLE passe** — rien n'est lu, rien n'est effacé.
3. **Projection** `internal/sync/replayartifacts/positions.go`, appelée par `Deriver` (A.4) en
   DERNIER (c'est la plus volumineuse).
4. **Suppressions** (règle 7, zéro code mort) : `PlayerPositionsRepo.WriteMatch` et
   `writePlayerPositionsTx` ; les 4 tests d'écriture du repo ; le `-write` du mode positions de
   `cmd/diag_weapons_v3` (il **refuse** désormais explicitement, en disant où le travail se fait)
   et `cmd/diag_weapons_v3/write_conn.go` **entier** (101 L), devenu sans appelant.
   La lecture passe sur `match_player_positions_latest` (règle ART n°2).

**LA CADENCE : LA SEULE DÉCISION DE CE POINT, ET ELLE EST MESURÉE.** Le document publie une
position par vie et par frame (100 ms). Mesure faite sur les **106 artefacts du cache local** le
2026-09-06 :

| | par match |
|---|---|
| trajectoires BRUTES | moyenne **31 051**, médiane 29 167, max 129 096 |
| décimées à 20 s (`GrainPositionsMS`) | moyenne **215**, médiane 201, max 895 |

Le grain de 20 s n'est pas choisi : c'est celui que **le schéma de la table déclare depuis sa
création** (`steps_shared_player_positions.go` : « granularité ~20s du snapshot type-2 »), et son
unique lecteur binne en grille 20×20 — au-delà de quelques centaines de points, chaque ligne de
plus est du poids de base et de fil pour zéro pixel. Projeter les trajectoires brutes aurait
multiplié par ~145 le volume d'une table qui alimente une carte de chaleur. Le premier point de
chaque vie est toujours retenu (une vie plus courte que le grain existe quand même). **Signalé
comme question ouverte Q-2 ci-dessous** : c'est le seul endroit du lot où j'ai tranché un
paramètre produit, et il est réversible d'une constante.

**Forme de la table anti-ART** : append-only avec vue `_latest` **par passe** (et non à clé
naturelle comme `match_objective_events`) — le choix est dicté par le schéma lui-même, qui
déclare « pas de PK contraignante » parce qu'une position n'a pas de clé naturelle (deux joueurs
peuvent occuper le même point au même instant). `match_player_positions` est donc ajoutée aux
DEUX listes (`tablesProtegees`, `appendOnlyStateTables`) ; **aucune allowlist agrandie**.

**Tests** : `persist/player_positions_persister_test.go` (nominal + **reprise qui supersède sans
effacer** : 1 ligne servie par la vue, 4 en base + les deux refus) ;
`platform/duckdb/player_positions_repo_test.go` réécrit pour la lecture (`_latest`, seconde passe,
match absent, capability) ; `replayartifacts/positions_test.go` (6 cas de décimation + transport
des valeurs, dont « l'équipe n'est pas inventée »).

**Gate A-II** :

```
go build ./...                                                     -> EXIT=0
go test ./internal/sync/... ./internal/analysis/replay/... ./internal/domain/...
        ./internal/archlint/... ./internal/persist/... ./internal/platform/duckdb/...
        ./cmd/diag_weapons_v3/...
  -> ok levelup/go-api/internal/platform/duckdb/sharedprovider  0.443s   EXIT=0
go test ./internal/migration/... ./internal/games/...             -> ok (…/weapons 12.401s) EXIT=0
go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...
  -> ok levelup/go-api/internal/persist  31.454s   EXIT=0
go test -tags=integration -p 1 ./internal/migration/... ./internal/api/...
  -> ok levelup/go-api/internal/api/wire  15.302s   EXIT=0
golangci-lint run --new-from-merge-base=origin/main ./internal/sync/... ./internal/persist/...
        ./internal/migration/... ./internal/platform/duckdb/... ./internal/replaybuild/...
        ./internal/api/wire/... ./cmd/diag_weapons_v3/...
  -> 0 issues.  EXIT=0
go test ./internal/sync/ -run 'ART|AppendOnly|Mutation|Allowlist|Delete|Bulk'
  -> ok levelup/go-api/internal/sync  38.341s   EXIT=0  (match_player_positions enrôlée, 0 allowlist)
```

## Découvertes (hors périmètre, NON traitées)

- **D-1** — Aucune des trois familles passagères de la passe film (`match_weapon_shots`,
  `match_weapon_hit_distance`, `kill_positions`) n'a de prédicat de reprise propre : leur seule
  reprise possible passe par `KillSourceDecoderRev`, alors que deux d'entre elles portent une
  révision distincte qui n'est lue nulle part et que la troisième n'en a pas. Détail et mesures
  ci-dessus (A.1). Le plan interdit d'ajouter une révision dans ce lot.
- **D-2** — Les autres catalogues de `reference/` restent écrits UNIQUEMENT par leurs CLI
  (`map_objectives.json`, `map_positions_jouees.json`, `map_callouts.json`,
  `map_fond_reglages.json`) : vérifié, aucun autre chemin runtime n'écrit dans `reference/`.
  L'overlay et son ratchet ne couvrent donc aujourd'hui QUE `map_weapon_pads.json`, ce qui suffit
  au constat A0 ; le jour où un second rattrapage runtime apparaît, il faudra le même
  garde-rail — la structure `reference/generated/` est déjà prête et ignorée par git pour tout
  le dossier.
- **D-3** — Le lint non ratcheté de `internal/sync/` remonte 15 problèmes préexistants (détail
  en A.2). Hors périmètre du lot A ; ils appartiennent à la dette gelée par la baseline.
- **D-4 (demandée par le plan)** — `match_objective_events` (les captures CTF écrites par
  `cmd/diag_weapons_v3` en objective-events `-write`, via `ObjectiveEventsRepo.WriteMatch` en
  DELETE-then-INSERT) est restée **strictement hors périmètre**, comme le plan l'exige. Elle
  garde son `-write`, son repo, et son DELETE. Note de forme pour qui la reprendra : elle a une
  PK naturelle `(match_id, seq)` et AUCUNE vue `_latest` — c'est pourquoi
  `bomb_stats_persister.go` y écrit « seulement si le match n'a pas déjà de faits ». La convertir
  demande une sémantique de génération que son schéma ne porte pas : un chantier de schéma, pas
  une ligne de code.
- **D-5** — Les positions projetées n'ont **aucune clé de capability** (`film.*`), là où l'usage
  et l'Assaut en ont une (`film.usage_summary`, `film.bomb_stats`). Non traité : les fichiers
  `config/titles/**/capabilities.toml` et `internal/domain/title/**` appartiennent au lot C.
  Effet nul aujourd'hui — un titre sans décodeur de film n'a aucun artefact, donc aucune
  dérivation ne s'exécute pour lui.
- **D-6** — La table reste MATCH-LEVEL (pas de `xuid`) alors que le document, lui, nomme le
  porteur de chaque trajectoire. Nommer le joueur est désormais POSSIBLE (ce ne l'était pas avec
  l'ancien décodeur keyframe) mais changerait la forme d'une table déjà lue par la carte de
  chaleur — hors décision 1.
- **D-7 (risque d'intégration, à signaler au superviseur)** — La tâche A-I a modifié
  `internal/domain/title/registry.go` (ajout de `MapWeaponPadsOverlayPath`), fichier que le brief
  listait comme appartenant au lot C. Le lot C ajoutera `CapReplay` au même fichier : **conflit de
  merge probable, sur deux ajouts indépendants** (une méthode de chemin d'un côté, une capability
  de l'autre) — résolution attendue triviale, mais elle doit être anticipée à l'intégration.

## Questions ouvertes

- **Q-1 (pour le superviseur, pas bloquante)** — Le bump de `KillSourceDecoderRev` rend TOUT le
  parc à nouveau candidat au backlog de redécodage (1 325 films, 8 à 30 s par film). C'est
  l'effet VOULU du constat P0-1, mais c'est une charge de fond qui démarrera au premier cycle
  après le merge : à arbitrer au moment de l'intégration (horizon du backlog, ordre du plus
  récent au plus vieux déjà en place depuis le 2026-08-29).
- **Q-2 (A.6, le seul paramètre produit que j'ai tranché)** — La cadence d'échantillonnage des
  positions projetées, `GrainPositionsMS = 20 000`. Justification : c'est la granularité que le
  schéma de la table déclare depuis sa création, et elle reproduit l'ordre de grandeur de ce que
  la table portait (215 positions par match en moyenne contre 31 051 sans décimation, mesuré sur
  le corpus local). Si le produit veut une carte de chaleur plus dense, la constante est le seul
  point à changer — mais le volume écrit suit linéairement (grain 2 s ≈ 2 150 lignes par match,
  grain 100 ms ≈ 31 000). À confirmer ou à corriger, avec la mesure sous les yeux.
- **Q-3 (A.5, effet de bord du premier cycle après merge)** — Le rattrapage des dérivés va
  prendre en charge les 106 artefacts du cache local, cinq par cycle, jusqu'à convergence (~21
  cycles). Chacun ouvre un writer shared court et écrit jusqu'à ~900 positions. C'est borné et
  voulu, mais c'est un régime transitoire qu'il vaut mieux avoir vu venir.
