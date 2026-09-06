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

## Corrections après revue (A-R1, 2026-09-06)

> Verdict de revue : sept constats recevables (C1, C2 en P1 ; C3 à C7 en P2), douze mutations
> jouées, vingt et une conditions vérifiées qui tiennent. Chaque correction ci-dessous est
> prouvée par la mutation ou le test que le verdict prescrivait : **rouge d'abord, vert
> ensuite**, la mesure du rouge est citée.

### [x] Correction 1 (C1, P1) — la marque de dérivation ne se pose que sur ce qui a été écrit

**Vérification sur pièces.** `derivations.go:132` appelait `marquerDerivations` en fin de passe,
sans condition, et le commentaire de la fonction ne justifiait que les cas « rien à écrire »
(match hors Assaut, document sans `t0FilmMs`, titre sans capability) — jamais le cas « writer
indisponible ». `registry_build_queue.go:250-252` documentait pourtant l'inverse (« le rattrapage
les rejouera »).

**Le défaut, mesuré.** Test jetable du verdict rejoué en test permanent : `Deriver` avec
`AcquireWriter == nil`, puis avec un `AcquireWriter` en erreur → les quatre projections
journalisent, incrémentent leurs compteurs `*Echecs`, écrivent **zéro ligne**, et la marque se
pose quand même. `DerivationsUpToDate` rend alors `true` **à jamais** et `candidatsDerivations`
exclut ce match définitivement.

**Fait.** `bilanDerivations` (derivations.go) enregistre PAR MATCH ce que les familles n'ont pas
pu écrire, et distingue trois états :

| État | Conduite |
|---|---|
| writer indisponible (nil, ou acquisition en erreur) | **aucune marque sur tout le lot**, WARN avec le compte d'échecs |
| échec d'un match (capabilities illisibles, INSERT refusé, état du T0 illisible) | **ce match seul** n'est pas marqué, WARN nominatif |
| rien à écrire (titre sans clé `film.*`, document sans `t0FilmMs` ni trajectoire) | dérivation JOUÉE, **marquée** — la rejouer serait du travail pur perte |

Les quatre familles alimentent le bilan (`echecT0`, `echecUsage`, `echecBombe`,
`echecPositions` pour l'indisponibilité ; `b.echec(matchID)` pour un échec unitaire). Le rejeu
d'un match non marqué reste sûr : trois familles sont append-only (une passe neuve supersède) et
le report du T0 porte sa garde « déjà à la même valeur ».

**Preuve** (`derivations_marque_test.go`, rouge avant / vert après) :

```
go test ./internal/sync/replayartifacts/ -run TestDeriver_
  AVANT -> FAIL  aucun_writer_cable    : marque posee sans ecriture (constat C1)
           FAIL  acquisition_en_erreur : marque posee sans ecriture (constat C1)
  APRES -> ok  levelup/go-api/internal/sync/replayartifacts  0.150s
```

Trois tests : les deux sous-cas du writer indisponible, une famille en échec (capabilities
illisibles) qui ne marque pas SON match, et la contre-épreuve « rien à écrire se marque quand
même » — sans elle, le rattrapage rejouerait ces matchs à chaque cycle indéfiniment.

### [x] Correction 2 (C2, P1) — le rattrapage tourne même quand il n'y a rien à cuire

**Vérification sur pièces.** `artifacts.go` : `if len(work) == 0 { … return }` précédait bien les
DEUX seuls appels à `rattraperDerivations` (chemin ouvrier après `enqueueAll`, chemin local après
`buildAll`).

**Le défaut, mesuré.** C'est **exactement l'état converge** que le constat A2 vise : une instance
dont les artefacts ont été cuits AVANT ce lot, donc sans marque. Test jetable du verdict rejoué
en test permanent (2 matchs au registre, artefact posé pour chacun, aucune marque) :

```
travail de cuisson selectionne = 0
candidats a la derivation      = 2
marques posees par Run         = 0 / 2
```

`JaugeDerivationsRetard` n'était pas publiée non plus : « tout est dérivé » et « le rattrapage ne
tourne pas » redevenaient indistinguables — l'ambiguïté exacte que la jauge dit fermer.

**Fait.** `Run` pose `defer rattraperDerivations(ctx, d)` juste après la garde `armee`, et délègue
la cuisson à `cuireLeCycle`. Le `defer` n'est pas une commodité : c'est ce qui rend le rattrapage
**total sur toutes les sorties** (sélection vide, titre sans catalogue de bornes, chemin ouvrier,
chemin local) au lieu de deux points d'appel qu'une troisième sortie contournerait à nouveau. Les
deux appels internes disparaissent — un seul site.

**Preuve** (`backlog_integration_test.go`, `TestRun_SelectionDeCuissonVide_RattrapeQuandMeme`) :

```
go test -tags=integration -p 1 ./internal/sync/replayartifacts/ -run TestRun_SelectionDeCuissonVide
  AVANT -> FAIL  cuittout1 : aucune marque posee par Run (constat C2)
                 cuittout2 : aucune marque posee par Run (constat C2)
  APRES -> ok  levelup/go-api/internal/sync/replayartifacts  10.269s
```

Le test passe **par `Run`**, là où les tests du lot appelaient `candidatsDerivations` directement
et ne pouvaient donc pas voir le trou. Il vérifie aussi que rien n'est mis en file : le rattrapage
ne cuit pas.

### [x] Correction 3 (C3, P2) — le verrou du catalogue crée son dossier et ne confond plus ENOENT

**Vérification sur pièces.** `mapcatalog/store.go` `prendreVerrou` ouvrait `<overlay>.lock` en
`O_CREATE|O_EXCL` sans `MkdirAll`. Or l'overlay vit sous `reference/generated/`, ignoré par git
(`.gitignore:152`) : il n'existe pas sur un checkout neuf ni sur une instance fraîchement
déployée — l'état NOMINAL du premier rattrapage.

**Le défaut, mesuré** (mutation `j` du verdict, rejouée avant correction) :

```
go test ./internal/mapcatalog/ -run 'Verrou|DossierAbsent'
  -> WARN mapcatalog: verrou d'ecriture tenu trop longtemps — passage force  attente=2s  (x5)
  -> FAIL 1 carte(s) conservee(s) sur 8
     + 3 renames en « Acces refuse »
```

Deux secondes d'attente inutile, un WARN qui **ment sur la cause**, puis une écriture sans
exclusion mutuelle — précisément le trou que `TestAddOverlayEntryConcurrentNePerdPasDEntree`
prétend fermer, sauf qu'il travaille sur un dossier déjà créé.

**Fait.** `MkdirAll` du dossier avant le verrou ; un verrou tenu est désormais un `EEXIST` **et
rien d'autre** — toute autre erreur (droits, dossier disparu) passe en force IMMÉDIATEMENT avec
un journal qui nomme l'erreur réelle, au lieu d'attendre deux secondes pour se tromper de
diagnostic.

**Preuve** (`internal/mapcatalog/verrou_test.go`, deux tests) : le verrou EXISTE réellement sur
dossier absent (c'est la différence entre « posé » et « passage forcé ») et se prend en quelques
millisecondes ; 8 rattrapages simultanés sur dossier absent conservent 8 cartes. Mesures après
correction :

```
--- PASS: TestAddOverlayEntryCreeLOverlayAbsent (0.01s)        [2,06 s avant]
--- PASS: TestPrendreVerrouCreeLeDossierEtNAttendPas (0.00s)
--- PASS: TestAddOverlayEntryConcurrentDossierAbsentNePerdRien (0.19s)   8/8 cartes
ok  levelup/go-api/internal/mapcatalog  0.700s   — aucun WARN dans la sortie
```

### [x] Correction 4 (C4, P2) — la conversion append-only des positions a enfin un garde-rail

**Vérification sur pièces.** `steps_shared_player_positions_appendonly.go` n'avait aucun fichier
de test, alors que toutes les autres conversions append-only en ont un
(`steps_shared_bomb_stats_test.go`, `steps_player_append_only_csr_snapshots_test.go`,
`games/halo_infinite/migrations/shared_kill_positions_appendonly_test.go`).

**Fait.** `steps_shared_player_positions_appendonly_test.go`, trois tests sur DuckDB. La table de
départ est créée par la **migration réelle** — `shared_match_player_positions_v1`, résolue par son
nom dans le registre (`All()`) : aucune DDL recopiée, qui aurait dérivé le jour où la vraie aurait
bougé (leçon du dépôt : « DDL de test recopiées = dérive indétectable »). Base sur fichier
temporaire (`openTmpDB`, la convention des autres tests de migration) et non `:memory:` :
`database/sql` gère un POOL et une base DuckDB en mémoire est propre à chaque connexion — le CTAS
et la relecture pourraient tomber sur deux bases différentes.

Les trois propriétés : trois lignes legacy écrites à trois instants **différents** survivent en
UNE passe et sont les trois servies par la vue ; le DDL est idempotent ; une passe neuve supersède
toute la génération legacy sans effacer une ligne (5 brutes, 2 servies).

**Preuve par la mutation du verdict** (mutation `i`, jouée puis annulée) :

```
SyntheticCols: `'legacy-diag' AS positions_pass`  ->  `CAST(written_at AS VARCHAR) AS positions_pass`
go test ./internal/migration/ -run TestPositionsAppendOnly
  -> FAIL  1 ligne(s) servie(s) par match_player_positions_latest, attendu 3
     FAIL  1 ligne(s) servie(s) apres deux passages, attendu 3
(retour)
  -> ok  levelup/go-api/internal/migration  49.274s
```

C'est la mutation qui laissait **toute la suite verte** avant ce test, pendant qu'en production la
carte de chaleur de chaque match déjà rempli se serait réduite à un point.

### [x] Correction 5 (C5, P2) — l'équipe des positions projetées, jointe depuis la base

**Vérification sur pièces — et elle a réfuté l'hypothèse de la revue.** Le verdict suggérait que
« le roster / l'identité de slot du document donne l'équipe de chaque trajectoire ». Ce n'est pas
le cas : `analysis/replay/document.go:1172-1175` documente `Track.Team` comme valant -1 parce que
**l'équipe n'est pas dans le film**, et `RosterEntry.Name` le redit (« ce qu'il ne donne PAS, et
que seule la base porte : l'équipe »). `build.go:502` pose `Team: -1` sans condition. Il n'y a
donc **aucun champ d'équipe à lire dans l'artefact** — consigne respectée : ne pas l'inventer.

**Les deux moitiés du constat, traitées :**

1. **La doc était inversée** (anti-pattern n°9). L'en-tête affirmait que -1 était « la même valeur
   non attribuée que l'ancien décodeur produisait ». Faux : `analysis/positions/positions.go:74`
   appelait `assignTeamsBestEffort`, qui attribue 0/1 dès qu'un écart franc sépare deux groupes
   sur l'axe X (`positions/team.go:19-45`) — un **devinement spatial**, jamais l'équipe réelle,
   mais pas -1 non plus. L'en-tête dit désormais ce qui est, et nomme la mesure.
2. **Le filtre serait devenu du code mort.** `MatchPositionsHeatmap.tsx:119-133,148-158` ne rend le
   filtre Global / Équipe A / Équipe B que si au moins une position porte `team != -1`.

**Fait.** L'équipe est **jointe**, par le xuid que le document nomme sur chaque vie, contre
`match_participants` — la même jointure que celle que le client fait déjà. Le lecteur est
`port.ReplayFactsRepo`, qui EST le lecteur de « ce que la base sait du match » pour le rejeu,
camps compris (règle 14 : réutiliser l'existant). La lecture se fait sur le handle **writer**,
dans le même segment court — même règle et même raison que le report du coup d'envoi (t0film.go).

La projection reste **pure et hors writer** : `projeterPositions` emporte le porteur de chaque
ligne (`passePositionsPrete.porteurs`, jamais écrit en base — la table est match-level par
schéma), et `appliquerEquipes` fait la jointure ensuite. L'équipe publiée par le film **prime**
quand elle existe ; `EquipeInconnue` (-1) reste pour une vie anonyme ou un xuid hors participants.

**Preuves.** Trois tests unitaires (`TestPoserEquipes_DeuxSlotsDeuxCamps` : deux slots, deux camps
→ lignes à 0 et 1, anonyme et xuid inconnu restent à -1 ; `…EquipeDuDocumentPrime` ; la parité
porteurs/lignes) plus un test d'intégration de bout en bout sur base migrée. Mutation jouée :

```
situees := appliquerEquipes(ctx, db, prets)   ->   debranche
go test -tags=integration -p 1 ./internal/sync/replayartifacts/ -run TestPersisterPositions_EquipeJointeDepuisLaBase
  -> FAIL  repartition = map[-1:3], attendue map[-1:1 0:1 1:1]
(retour) -> PASS (2.44s)
```

### [x] Correction 6 (C6, P2) — les marques ne sont plus prises pour des artefacts

**Vérification sur pièces.** `<short8>.derived.json` est déposée dans `ReplayArtifactsDir`, à côté
de `<short8>.json`, et finit par `.json` : les deux consommateurs qui filtrent sur ce suffixe la
comptaient comme un artefact.

| Consommateur | Effet mesuré |
|---|---|
| `scheduler/replay_purge_cron.go:152-160` | `short = "<short8>.derived"`, absent du registre → `unknown++` : une ligne INFO à CHAQUE passage dès qu'une dérivation existe, et une marque qui SURVIT à la purge de son artefact |
| `cmd/backfill_t0_film/artefacts.go:103` | chaque marque tombe en « sans match_id » : jusqu'à 2× le corpus en entrées fantômes, alors que la propriété AFFICHÉE du bilan est « chaque artefact tombe dans EXACTEMENT une catégorie — c'est ce qui rend le total vérifiable » |

**Fait.** Un prédicat **commun** — `replaybuild.EstMarqueDerivations(nom)`, à côté du suffixe
désormais exporté — plutôt qu'un littéral recopié dans chaque appelant (règle des ≤ 2 copies : ce
serait le troisième exemplaire, et la première évolution du suffixe en aurait laissé un en
arrière). Le cron ignore les marques ET supprime celle de l'artefact qu'il purge ; celles des
artefacts conservés restent, sinon le rattrapage rejouerait leurs dérivations.

**Preuve par mutation** (les deux gardes débranchées, puis rétablies) :

```
go test ./cmd/backfill_t0_film/ ./internal/scheduler/ -run 'Marques|Scanner'
  -> FAIL  4 verdict(s) [aaaa0001.derived.json aaaa0001.json bbbb0002.derived.json bbbb0002.json], attendu 2
  -> FAIL  purge = (purged 1, kept 1, unknown 4), attendu (1, 1, 1)
           + la marque de l'artefact purge a survecu
(retour)
  -> ok  levelup/go-api/cmd/backfill_t0_film  1.192s
     ok  levelup/go-api/internal/scheduler    1.855s
```

### [x] Correction 7 (C7, P2) — une seule acquisition du writer pour les quatre familles

**Vérification sur pièces.** Les quatre sites d'acquisition (`t0film.go:108`, `usage.go:118`,
`bombstats.go:150`, `positions.go:133`), `acquireWriterTimeout = 60 s`
(`registry_actions.go:49`), `WriteTimeout: 30 * time.Second` (`cmd/server/main.go:1457`) —
confirmés à la ligne près. Le commentaire de `deriverArtefactRange` justifiait le synchrone par
« un segment writer COURT, relâché aussitôt » : cela décrit UNE acquisition, pas quatre en file
derrière un lease.

**Fait, deux pièces.**

1. **`writerUnique`** (derivations.go) mémoïse la source pour la passe : au plus UNE acquisition,
   un seul retrait. La fonction de retrait rendue aux familles est un **no-op** — sinon la
   première à rendre la main relâcherait le lease sous les pieds des trois suivantes ; le vrai
   retrait est posé en `defer` par `Deriver`. L'acquisition reste **paresseuse** : une passe sans
   rien à écrire n'ouvre aucun segment (propriété qu'un regroupement naïf aurait perdue).
   L'erreur est mémoïsée elle aussi — pas de réessai par les trois familles suivantes.
2. **`acquireWriterDepot` = 8 s** remplace `acquireWriterTimeout` (60 s) **sur le seul chemin du
   dépôt d'ouvrier**. Le temps de réponse du dépôt est donc majoré par 8 s plus les écritures d'UN
   match, très en dessous des 30 s du serveur.

**Le choix du synchrone borné plutôt que de l'asynchrone après réponse est documenté** sur
`deriverArtefactRange` : une goroutine détachée demanderait un `context.WithoutCancel` (le
contexte de la requête meurt avec elle), une borne sur le nombre de dérivations en vol et un arrêt
propre au shutdown — trois états à surveiller pour la même garantie de réponse. L'abandon sur
délai est **sûr par la correction 1** : la marque n'est pas posée, donc le rattrapage rejoue.

**Preuves.**

```
go test ./internal/sync/replayartifacts/ -run TestDeriver_UnSeulSegmentWriterPourLesQuatreFamilles
  AVANT -> FAIL  writer acquis 3 fois pour UNE passe de derivations, attendu 1
  APRES -> PASS
```

Plus : « rien à écrire » → 0 acquisition (la propriété à ne pas perdre) ; et sur base réelle,
`TestDeriver_UnSegmentAcquisEtRelacheUneFois` — 1 acquisition, 1 retrait, et les écritures des
trois familles vérifiées EN BASE (un `release` prématuré les aurait fait échouer en silence).

Le budget HTTP a son propre garde-rail, `build_queue_writer_budget_test.go` : il lit le
`WriteTimeout` dans `cmd/server/main.go` (les deux paquets ne peuvent pas s'importer) et exige que
`acquireWriterDepot` en laisse la moitié pour les écritures et la réponse.

```
const acquireWriterDepot = 8 * time.Second  ->  20 * time.Second
  -> FAIL  acquireWriterDepot = 20s pour un WriteTimeout serveur de 30s (constat C7)
(retour) -> PASS
```

## Gates des corrections (tous joués en avant-plan, dans ce worktree)

`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-faits`,
`GOLANGCI_LINT_CACHE=/c/Users/Guillaume/AppData/Local/golangci-v2-faits`, `CGO_ENABLED=1`, depuis
`apps/go-api`.

| # | Commande | Dernière ligne |
|---|---|---|
| 1 | `go build ./...` | (aucune sortie) — EXIT=0 |
| 2 | `go test -count=1 ./internal/sync/... ./internal/replaybuild/... ./internal/mapcatalog/... ./internal/migration/... ./internal/persist/... ./internal/platform/duckdb/... ./internal/api/wire/... ./internal/scheduler/... ./cmd/backfill_t0_film/...` | `ok levelup/go-api/cmd/backfill_t0_film 0.262s` — EXIT=0 |
| 3 | `go test -tags=integration -p 1 -count=1 ./internal/sync/... ./internal/persist/... ./internal/migration/... ./internal/api/...` | `ok levelup/go-api/internal/api/wire 18.698s` — EXIT=0 |
| 4 | `go test ./internal/sync/ -run 'ART\|AppendOnly\|Mutation\|Allowlist\|Delete\|Bulk' -v` | `ok levelup/go-api/internal/sync 55.970s` — 11 tests PASS, 0 SKIP |
| 5 | `golangci-lint run --timeout 10m --new-from-merge-base=origin/main ./...` | `0 issues.` — EXIT=0 |

Allowlists anti-ART revérifiées **vides** après corrections : `allowlistArtPatterns`,
`allowlistRawDelete`, `allowlistMediaMutation`. Aucun test skippé, aucune variable de film posée,
aucune cuisson d'artefact.

## Découvertes de la passe de corrections (hors périmètre, NON traitées)

- **D-8** — Le lint global émet un avertissement permanent : « Found unknown linters in //nolint
  directives: gosec — limit/placeholders maîtrisés, plr0913 — coordinator function ». Deux
  directives préexistantes écrivent `//nolint:gosec — raison` avec un tiret cadratin au lieu du
  séparateur `//` : `internal/api/wire/registry_weapon_coverage.go:109` et
  `internal/platform/duckdb/explorer_repo.go:579`. Le linter lit alors la raison comme un nom de
  linter, donc **ces deux `nolint` ne suppriment rien** — elles sont décoratives. Hors périmètre
  des sept corrections ; correction attendue triviale (remplacer `—` par `//`).
- **D-9** — `golangci-lint` laisse un verrou `%TEMP%\golangci-lint.lock` qui survit à l'arrêt du
  processus : deux invocations rapprochées se refusent l'une l'autre (« parallel golangci-lint is
  running »). Rencontré une fois pendant cette passe ; contourné en attendant la fin du processus.
  À signaler si le gate CI le rencontre.
- **D-10** — La table locale `match_player_positions` est toujours **vide** (0 ligne) : la
  jointure d'équipe de la correction 5 n'a donc aucun effet rétroactif observable localement. Elle
  se remplira au fil de l'eau, et le rattrapage des dérivés (correction 2) prendra les 106
  artefacts du cache en ~21 cycles.
