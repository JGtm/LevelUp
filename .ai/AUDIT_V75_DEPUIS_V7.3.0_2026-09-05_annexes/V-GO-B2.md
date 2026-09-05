# Vérification adverse V-GO-B2

Cadre : lecture seule sur `feat/v75` @ `736ccf3c3`. Aucune compilation, aucun test exécuté.
Chaque verdict est adossé à un fichier:ligne rouvert, pas au texte de l'audit.

---

## Constat 1 — Chaîne du rejeu sans aucune clé de capability : TIENT (gravité → P2)

**Ce que j'ai vérifié**

- `apps/go-api/internal/games/capabilities.go:13-41` — `AllCapabilityKeys()` énumère
  **25** clés ; les 5 `film.*` sont `film.kill_source`, `film.weapon_shots`,
  `film.kill_positions`, `film.usage_summary`, `film.bomb_stats`. Aucune ne concerne
  l'artefact de rejeu, la timeline d'objectif ni les positions keyframe.
- `git diff a2719a68c..HEAD -- internal/domain/title/registry.go | grep '^[+-].*Capability = '`
  → **sortie vide**. Zéro `Capability` title-level ajoutée depuis v7.3.0. Le même diff sur
  `games/adapter.go` rend exactement les **5** `+ Cap Film... CapabilityKey = "film.*"`.
- Reproduction de l'audit rejouée :
  `grep -rn 'CapFilmKillSource|CapFilmUsageSummary|CapFilmBombStats|caps\.Has|Capability'
  internal/api/handlers/replay.go internal/service/replay_service.go
  internal/sync/replayartifacts/artifacts.go` → **0 résultat**. Confirmé.
- `internal/api/server_apiv1.go:697-706` : le groupe des 4 routes `/replay*` porte
  `r.Use(handlers.LocalOnlyReplay)` et **rien d'autre** ; le commentaire assume
  (« Hors sous-groupe capability : la disponibilité EST la présence d'artefact »).
  `grep -n 'RequireCapability' server_apiv1.go` → seulement `CapCareer` (679),
  `CapAchievements` (686), `CapEngagement` (717), `CapMedia` (789). Le
  `r.Route("/players/{player_slug}"` de la ligne 636 n'est gardé par aucune capability :
  les 4 routes de rejeu sont bien montées pour tous les titres.
- `config/titles/halo_5/title.toml` : `capabilities = [ "matchmaking", "ranked", ... ]` —
  halo_5 déclare bien `matchmaking`, la clé sur laquelle la route web garde
  (`.../matches/$matchId/replay.tsx:56-58`, `RouteCapabilityGate capability="matchmaking"`).
- Étape 1.58 : `replayartifacts/artifacts.go:392-413` (`armee`) ne teste que `Placement`,
  `WithRead`, `Fetcher` — de la configuration et du câblage ; `:334` `replaybuild.NewBuilder`
  est bien une sonde de **présence de fichier**. Aucune capability.

**Ce que l'auditeur n'a pas vu (deux portes, aucune ne réfute le fond)**

1. **La conséquence produit est surévaluée : aucun lien ne mène à la page de rejeu pour
   halo_5.** Les deux seuls composants de lien rendent `null` quand l'artefact n'existe pas —
   `apps/web/src/lib/match-nav/MatchReplayLink.tsx:36` (`if (!available) return null`) et
   `apps/web/src/features/match-view/MatchHeader.replayLink.tsx:36`, dont l'en-tête se dit
   lui-même « la porte d'entrée du rejeu 2D, et la SEULE ». La valeur `available` vient de
   `replay_available` (`service/replay_service.go`, `os.Stat` sur
   `ReplayArtifactPath(titleSlug, …)`, chemin title-scopé) : elle est structurellement
   `false` pour halo_5. La page n'est donc atteignable **qu'en tapant l'URL à la main**.
   S'y ajoute `handlers.LocalOnlyReplay` (`replay_local_gate.go:117-126`) qui 404 toute
   requête non-loopback : hors poste local, les 4 endpoints sont morts pour **tous** les
   titres. L'affirmation « la page de rejeu s'ouvre pour tout match Halo 5 [et] l'utilisateur
   voit un état vide » est donc vraie seulement pour une URL saisie manuellement.
2. **Le manque est nommé dans un plan, mais ce n'est PAS un report décidé.**
   `.ai/PLAN_FINALISATION_REJEU_2D.md:414-417`, §3.5 « Poser la capability » :
   « `[ ]` Déclarer `film.replay2d` et y brancher la route et le lien. Aujourd'hui la seule
   porte est un 404 sur un fichier absent : un titre qui ne sait pas produire de rejeu n'a
   aucun moyen de le dire. » — case **NON cochée**, plan du 2026-07-31, contrat
   `plan-execution` affiché en tête. `grep -i replay2d .ai/V7.5/REGISTRE_REPORTS.md` ne rend
   que deux lignes sans rapport (bases du worktree ; calque des armes au sol) : **aucune
   entrée du registre ne porte ce report, donc aucune condition de reprise**. Au regard des
   règles 2 et 3 du skill `plan-execution` (« ne jamais différer une étape exécutable
   maintenant » / « statuer chaque item »), c'est une étape non terminée, pas une décision.
   Le constat est donc **confirmé par la documentation du dépôt**, pas atténué par elle.
   (`PLAN_MASTER_FILM_KILLFEED_REJEU.md:854` renvoie à cette même finalisation 3.5.)

**Conséquence réelle reformulée** — Le fond tient intégralement (zéro clé, 4 routes montées
pour tous les titres, 1.58 gardée par du disque et de la configuration), mais le dommage
observable aujourd'hui se réduit à une page vide pour qui tape l'URL à la main sur un poste
local : gravité à abaisser à **P2**, l'écart étant doctrinal (exigence 5) et non fonctionnel.

---

## Constat 2 — Doc inversée `/positions` et `/objective-events` : TIENT (P1 confirmé, périmètre élargi)

**Ce que j'ai vérifié — la chaîne exacte demandée, maillon par maillon**

1. **Le câblage est inconditionnel.** `internal/api/wire/registry_pages.go:113-117` : le
   commentaire promet « Titre sans film / tables absentes → le repo remonte
   `ErrCapabilityNotSupported` et l'endpoint rend un 503 propre », puis
   `.WithObjectiveEventsRepo(duckdb.NewObjectiveEventsRepo(pdb)).WithPlayerPositionsRepo(...)`
   sans le moindre `if`.
2. **Le service ne peut plus dégrader.** `internal/service/match_view_service.go:463-479` :
   `GetObjectiveEvents` / `GetMatchPositions` ne rendent `games.ErrCapabilityNotSupported`
   que si `s.objectiveEventsRepo == nil` — or il n'est jamais nil (point 1).
3. **Sur table VIDE mais EXISTANTE, `LoadMatch` rend `(nil, nil)`, pas une erreur.**
   `internal/platform/duckdb/objective_events_repo.go` : `loadObjectiveEventRows` fait un
   `SELECT ... WHERE match_id = ?`, la boucle `rows.Next()` ne s'exécute pas, `out` reste nil
   et l'erreur est nil ; `attachObjectiveEventPlayers` sort immédiatement
   (`if len(bySeq) == 0 { return nil }`). `isTableNotFoundErr → ErrCapabilityNotSupported`
   n'est atteint **que** si la table est absente. Le handler
   (`handlers/match_view_objective_events.go:41-44,95`) mappe alors `nil` par
   `toObjectiveEventsDTO` qui fait `make([]objectiveEventDTO, 0, 0)` → **HTTP 200 `[]`**.
   C'est exactement le point que la consigne demandait de trancher : **`[]`, pas 503**.
4. **La table EST bien créée dans le shared halo_5.**
   `internal/games/halo_5/migrations/metadata.go:311-317` : `OwnsTarget` rend `true`
   uniquement pour `migration.TargetMetadata`. `internal/migration/registry.go:223-230` :
   `RunForTitleDB` — « le set n'isole PAS ce target (OwnsTarget==false) → fallback complet
   (registre global Halo + titleStepsProvider) ». `cmd/server/main.go:1841-1855`
   (`provisionAdditionalTitle`) appelle `migration.RunForTitleDB(db, slug,
   migration.TargetShared)` sur `pr.SharedDBPath(slug)` pour tout titre additionnel. Les
   tables du film sont donc dans `data/titles/halo_5/warehouse/shared_matches_v2.duckdb`.
   Le registre `halo_5/migrations/` ne contient que `metadata.go` et `milestones.go` —
   aucun step `shared`.

**Ce qui confirme, et que l'auditeur n'a pas relevé : la doc inversée est à TROIS endroits,
pas un.** Aux deux hooks web s'ajoute la même promesse fausse :
`apps/web/src/features/match-view/queries.ts:22-26` — « Dégradation propre : l'endpoint
renvoie 503 (capability_not_supported) pour les titres sans film » — et `:47-51` pour
`useMatchPositions`. Ni l'un ni l'autre n'a de garde de capability dans `enabled`
(`enabled: enabled && !!playerSlug && !!matchId`) : sur l'onglet Chronologie d'un match
halo_5, les deux requêtes partent et reçoivent 200 `[]`. Le constat est donc **plus large**
que ce que l'audit décrit, pas plus étroit.

**Conséquence réelle reformulée** — Sur halo_5 les deux endpoints répondent 200 `[]` là où le
code (serveur ET client) promet un 503 : le client ne peut pas distinguer « ce titre ne sait
pas produire ce calque » de « ce match n'en a pas », et `http_capability_not_supported_total`
reste à zéro. P1 maintenu.

---

## Constat 3 — Mise en file et appel UGC avant la sonde de titre : TIENT PARTIELLEMENT (gravité → P2) ; le volet UGC est RÉFUTÉ

**Ce que j'ai vérifié**

- L'ordre allégué est exact. `internal/sync/replayartifacts/artifacts.go:294-336` :
  `armee` (295) → `selectionnerLeTravail` (300) → `rattraperCartesAbsentes` (322) →
  `if d.Placement == PlacementWorker { enqueueAll(...); return }` (323-326) →
  `replaybuild.NewBuilder` (334, commenté « SONDE DE TITRE, PAS UN CONSTRUCTEUR »).
- `selectBuildWork` (`replayartifacts/backlog.go:205-249`) sélectionne sur
  `match_registry WHERE match_id IN (…)` + fenêtre de rétention : **rien** n'exige un film.
  La sortie précoce « sélection vide » n'est donc pas une porte de titre.

**RÉFUTÉ — le volet « appel à l'API UGC ».** `rattraperCartesAbsentes`
(`replayartifacts/mvar_rattrapage.go:80-110`) porte une **seconde sonde de titre**, en amont
de tout réseau : elle charge
`title.NewPathResolver(d.RepoRoot).MapWeaponPadsPath(d.TitleSlug)` =
`data/titles/{slug}/reference/map_weapon_pads.json`
(`internal/domain/title/registry.go:797-799`) et **retourne** si le fichier est illisible,
avant d'atteindre le moindre `FetchMvarForMap`. Le commentaire du fichier nomme
explicitement le cas et l'assume : « un titre qui n'a pas encore de catalogue de cartes
(Halo 5). Le rattrapage journalise alors UN WARN et UN echec PAR CYCLE, ce qui est bruyant
mais assume ». Sur pièce : `ls data/titles/halo_5/` rend `players/ warehouse/` — pas de
`reference/`, alors que `git ls-files data/titles/` en compte 232 pour halo_infinite. Donc
**zéro aller-retour réseau UGC** pour un titre sans catalogue. L'affirmation « la seule sonde
de titre » est fausse : il y en a deux, et c'est justement la première qui garde l'appel
réseau que l'audit lui impute.

**TIENT — le volet `enqueueAll`.** `replayartifacts/cuisson.go:34-105` ne consulte aucune
capability ni aucun catalogue de titre ; `wire/registry_build_queue.go:46-64`
(`EnqueueReplayBuild`) accepte n'importe quel `titleSlug` et fait
`resolveFilmChunkURLs` — un aller-retour réseau par match — dont l'échec est journalisé en
`slog.Debug` et compté `refuses`.

**Portée réelle aujourd'hui : nulle, et l'auditeur l'avait vu à moitié.** halo_5 n'atteint
jamais ce code : `handlers/sync_handler.go:264-269` (`runnerFor`) et
`scheduler/auto_sync_run.go:125,313,445` routent par `livesync.HandlesTitle` /
`RunnerForTitle` (`games/halo_5/livesync/wire.go:24-46`, `runnerBuilders` = `{halo5.TitleSlug}`)
vers un `*livesync.Runner` dédié, et le hook 1.58 n'est installé que sur un `*sync.SyncEngine`
(`sync_handler.go:210`, `scheduler/auto_sync_engine.go:147-152`). S'y ajoutent deux verrous :
`enqueueAll` sort si `d.Enqueue == nil`, et `PlacementWorker` exige
`LEVELUP_BUILD_WORKER_TOKEN` (`replaybuild/placement.go:72-77`).

**Conséquence réelle reformulée** — Il reste un vrai défaut d'ordre, réduit de moitié : en
placement `worker`, un futur titre sans film enfilerait un job (et une résolution de manifeste
réseau) par match avant la sonde `NewBuilder` ; le rattrapage `.mvar`, lui, est déjà gardé
par titre. Aucun titre existant n'y passe. Gravité **P2**.

---

## Constat 4 — Deux sites servent le paquet Infinite à tout titre franchissant la capability : TIENT (gravité → P2)

**Ce que j'ai vérifié**

- `internal/sync/killcollector/classifier.go:33-44` : après
  `if !caps.Has(games.CapFilmKillSource) { return nil }`, la fonction rend
  `halo.NewKillSourceRegistry()` — aucune lecture du `slug`, qui est pourtant un paramètre.
  L'en-tête du fichier revendique « La garde est la capability, jamais le slug » : la garde
  l'est, la RÉSOLUTION ne l'est pas.
- `internal/replaybuild/replaybuild.go:431` : `adapter := halo.NewAssetURLAdapter()` en dur
  dans `(*Builder).neutralDeaths`, alors que le `Builder` porte `b.titleSlug`.
  Vérifié aussi que `NeutralDeathIcon` est **absent** de l'interface
  `games.TitleAssetURLAdapter` (`internal/games/adapter.go:309-345`) et n'existe que sur le
  type concret Infinite (`games/halo_infinite/adapter_asset_urls.go:265`) — l'import direct
  n'est donc pas un raccourci, c'est le seul chemin actuel.
- **Un autre titre PEUT déclarer `film.kill_source` aujourd'hui.**
  `games.LoadCapabilityMap` (`internal/games/capabilities.go:97-111`) →
  `CapabilityMapFromMappings` (`:59-83`) ne valide que deux choses : la clé appartient à
  `AllCapabilityKeys()` et le statut est `supported|degraded|not_exposed`. Les 5 clés
  `film.*` sont dans ce vocabulaire canonique : **rien n'interdit à
  `config/titles/<slug>/mappings/capabilities.toml` de les déclarer**. La fixture
  `config/titles/synthetic_title_b/mappings/capabilities.toml` déclare bien **15** clés
  (recomptées) et **aucune** `film.*` — donc aucune fixture n'exerce ce cas.

**Ce qui atténue, et que l'auditeur n'a pas dit** : le déclencheur n'est pas atteignable par
une simple édition de TOML. Les deux sites vivent dans des paquets qui sont **entièrement**
Infinite par leurs imports (`replaybuild.go:38-40` = `halo`, `film/killsource`,
`replaylabels` ; `killcollector` = `film/killsource`, `film/filmcache`,
`halo_infinite/ingest`), et le second exige en plus un `replaybuild.NewBuilder` réussi
(bornes de carte + `replay_labels.toml` du titre). Un 2e titre qui déclarerait la clé
n'obtiendrait pas « la mauvaise table » : il obtiendrait la chaîne de décodage d'Infinite
dans son ensemble, et elle échouerait sur son format de film. Le piège est donc réel mais il
suppose un chantier, pas une ligne de TOML.

**Conséquence réelle reformulée** — Le gate de capability est franchi par le vocabulaire, la
résolution par titre ne l'est pas : deux sites rendent des objets Infinite sans regarder le
slug. Latent, à effet nul aujourd'hui et non déclenchable par la seule déclaration d'une
capability : gravité **P2**, à relever le jour où un 2e titre à film est ouvert.

---

## Constat 5 — Décodeur du film et tables Infinite sous `internal/analysis/` : TIENT (P1)

**Ce que j'ai vérifié — les chiffres sont reproduits à l'unité**

Comptage non-test par paquet (`ls .../*.go | grep -v _test | xargs wc -l`) :
`filmdec` **108 / 25 637** · `replay` **114 / 26 834** · `objectiveevents` 12 / 3 175 ·
`sessionusage` 5 / 884 · `filmsource` 3 / 499 · `weaponv3` 4 / 368 · `positions` 2 / 275 —
soit **248 fichiers / 57 672 lignes**. Identique au chiffre publié.
`git diff --name-status a2719a68c..HEAD` sur ces sept répertoires, hors `_test.go` :
**298 `A`, 0 `M`** — le « 0 modifié » est confirmé (le décompte d'ajouts de l'audit, 256,
est plus bas que le mien, sans doute par exclusion des `testdata` ; sans incidence).

- `internal/analysis/filmdec/doc.go:1-3` : « Package filmdec decodes the **Halo Infinite**
  Theater film replication stream … reverse-engineered statically from **HaloInfinite.exe**
  via Ghidra ». Aucune ambiguïté sur la nature title-specific.
- `internal/analysis/replay/vehicle_families.go:79` : `vehicleFamilyByChassis` avec les
  GlobalIDs de tag `vehi` Infinite (`0x00002705` Warthog, `0x000025aa` Mongoose, …).
- `internal/analysis/replay/usage_summary_families.go:78-81` : `usageWallPanelIDs =
  {"0x528fce46","0x686b40c9"}`, dont le commentaire admet lui-même « même table que
  `WALL_PANEL_IDS` côté web (placementWall.ts), deuxième et dernière copie tolérée ».
- Antériorité confirmée : `.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md:204-229`, « [P1] Décodeurs
  Halo-Infinite-only sous `internal/analysis/` (contre ADR 0012) », avec sa propre
  vérification adverse « TIENT AVEC NUANCE » et, ligne 229 puis 497-498, la mention
  « **Décision : escalade utilisateur** (décision d'architecture) » / « Placement
  `filmdec`/`positions`/`replay` vs ADR 0012 : déplacement de paquets ou ADR d'exemption ».

**Ce que j'ai cherché comme décharge, et qui n'existe pas**

- **Aucune ADR ni aucun commentaire de paquet ne justifie l'emplacement.** ADR 0012
  (`docs/adr/0012-halo-only-adapters-extraction.md`) dit l'inverse en toutes lettres :
  « Le package `analysis/` ne contient plus que des algorithmes purs cross-titre » ;
  ADR 0031:71 la reprend (« Halo-only code lives under `internal/games/<slug>/` »).
  Aucune ADR d'exemption n'a été écrite depuis.
- **La justification « aucun import de `games/` = cross-titre par construction » est
  FACTUELLEMENT vraie mais n'est écrite nulle part** : `grep 'games/halo'` sur
  `analysis/filmdec/` et `analysis/replay/` (hors tests) ne rend que **trois commentaires**
  (`replay/catalog.go:9,159`, `replay/vehicle_families.go:23`), zéro import réel. Elle n'est
  donc ni documentée ni figée : `archlint/no_temporal_title_import_test.go` ne couvre que
  `analysis/temporal` (vérifié dans son en-tête), `archlint/filmsource_leaf_test.go` ne
  couvre que `analysis/filmsource`, et `filmdec_package_vars_test.go` est un ratchet de
  variables, pas de frontière.
- **Aucune entrée du registre.** `grep -i 'adr 0012|placement.*filmdec|déplacement de
  paquets'` sur `.ai/V7.5/REGISTRE_REPORTS.md` et `.ai/thought_log.md` → rien. L'escalade du
  2026-08-06 n'a donc pas été convertie en report daté avec condition de reprise : elle est
  restée sans suite pendant que le périmètre passait à 248 fichiers.
- Élément de coût que l'auditeur n'a pas cité, et qui joue plutôt en faveur du constat qu'il
  ne l'atténue : `archlint/filmsource_leaf_test.go:1-20` documente un cycle d'import
  **vérifié** (`games/halo_infinite/film/filmcache` importe `analysis/objectiveevents`, et
  cinq tests internes de `filmdec` importent `objectiveevents`/`filmcache`). Autrement dit,
  le paquet de titre dépend déjà de la couche « agnostique » — l'inversion de frontière est
  installée dans les deux sens.

**Conséquence réelle reformulée** — 57 672 lignes de format Halo Infinite (dont le décodeur
lui-même et deux tables d'identifiants `vehi`/`eqip` en dur) vivent sous le paquet que
l'ADR 0012 réserve au cross-titre, sans exemption écrite ni garde-rail, et le chantier v7.5
a multiplié le périmètre au lieu de le résorber depuis que le point a été escaladé le
2026-08-06. P1 maintenu.

---

## Bilan : 4 tiennent, 0 réfuté, 1 tient partiellement (un volet réfuté) — 3 requalifiés en gravité

| # | Verdict | Gravité |
|---|---|---|
| 1 | TIENT (fond intact ; conséquence produit surévaluée — aucun lien ne mène à la page, garde loopback) | P1 → **P2** |
| 2 | TIENT — chaîne vérifiée maillon par maillon : table présente + vide ⇒ **200 `[]`**, jamais 503 ; doc inversée à **3** endroits | **P1** |
| 3 | TIENT PARTIELLEMENT — volet `enqueueAll` confirmé ; **volet UGC RÉFUTÉ** (2e sonde de titre en tête de `rattraperCartesAbsentes`) ; inatteignable aujourd'hui | P1 → **P2** |
| 4 | TIENT — les deux sites vérifiés, et un titre tiers PEUT déclarer `film.kill_source` ; mais le piège suppose un décodeur, pas une ligne de TOML | P1 → **P2** |
| 5 | TIENT — chiffres reproduits à l'unité, aucune ADR/exemption/garde-rail/entrée de registre ne couvre l'emplacement | **P1** |

Aucun constat n'est réfuté en entier. Le seul énoncé factuellement faux de la série est
« l'étape 1.58 appelle l'API UGC avant sa seule sonde de titre » : il y a deux sondes, et la
première garde précisément cet appel.
