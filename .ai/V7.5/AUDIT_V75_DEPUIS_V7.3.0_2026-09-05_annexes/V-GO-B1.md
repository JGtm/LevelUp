# Vérification adverse V-GO-B1

Cadre : lecture seule, arbre `feat/v75` (HEAD `081871f09`, code identique à `736ccf3c3`).
Doctrine relue : `.claude/skills/arch-rules/SKILL.md` (couches L6-L20, note racine `api/` L22-24,
table « où va ce code » L28-38, patron Port L40-56, anti-patterns L58-75, testabilité L77-84).
Audit antérieur relu : `.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md` (523 L), périmètre
`v7.2.0 → 634e079f2` incluant explicitement le merge `feat/replay2d-prod` du 2026-08-05.

## Constat préliminaire de méthode (il conditionne C4 et C6)

`git merge-base --is-ancestor a2719a68c 634e079f2` → OUI. `a2719a68c` (tag `v7.3.0`) date du
**2026-08-04**, `634e079f2` (HEAD de l'audit 06/08) du **2026-08-05**. Or :

```
git ls-tree -r --name-only a2719a68c -- apps/go-api/internal/analysis/replay/   → VIDE
git ls-tree -r --name-only 634e079f2 -- apps/go-api/internal/analysis/replay/   → 24 fichiers
git ls-tree -r --name-only a2719a68c -- apps/go-api/internal/persist/ | grep kill_events → VIDE
git ls-tree -r --name-only 634e079f2 -- apps/go-api/internal/persist/ | grep kill_events → 7 fichiers
```

La base choisie par l'audit G3 tombe **un jour avant** le merge qui a fait entrer tout le
chantier rejeu/killsource dans `main`. Tout ce chantier ressort donc mécaniquement comme
« créé après v7.3.0 », alors qu'il a été **couvert par le périmètre de l'audit du 06/08**, qui
avait un auditeur A4 dédié « frontières de couches ». Ce n'est pas un détail de date : c'est ce
qui fait passer pour neufs des fichiers déjà audités et non retenus.

---

## Constat 1 — SQL brut / réseau authentifié / runner dans la racine `api/` : TIENT (gravité → P2)

- **Ce que j'ai vérifié** :
  - `api/wire/registry_replay_build.go:92-95` (`SELECT match_id, map_name, map_id FROM match_registry`)
    et `:127-129` (`SELECT name FROM asset_translations`) — SQL écrit à la main, confirmé.
  - `:27` `var replayBuildMu gosync.Mutex`, `:32` `RunReplayBuild` — runner single-flight, confirmé.
  - `registry_build_queue.go:46` `EnqueueReplayBuild`, `:151` `syncpkg.NewHaloAPIClient(...)`,
    `:281` `requireArtifactBeforeSuccess`, `:336` `reclaimBuildQueue` — confirmés.
  - Création : `git log --diff-filter=A a2719a68c..736ccf3c3` → `e166d4f25` et `9fd8df040`. Les deux
    fichiers sont bien **neufs**, l'écart de l'audit 06/08 (« SQL brut dans `internal/api/wire/*.go`
    | Lignes antérieures », ligne 446) ne les couvre effectivement pas.
  - Exemption cherchée : aucun commentaire de justification, aucune entrée dans les **35** ratchets
    de `internal/archlint/` (`ls internal/archlint/*_test.go | wc -l` = 35).
- **Ce que l'auditeur n'a pas vu** (n'annule pas le constat, en abaisse la gravité) :
  1. **Le motif « runner dans `api/wire` » est massivement antérieur.** `grep -n "^func (r \*ServiceRegistry) Run" internal/api/wire/*.go`
     → **12** runners, dont **10 antérieurs** (`registry_actions.go:54,119,254,303,385`,
     `registry_catalog_drain.go:44`, `registry_invariants.go:34`, `registry_monitoring.go:342`,
     `registry_monitoring_diskwatch.go:44` — boucle infinie —, `registry_monitoring_store.go:90`).
     Le SQL brut dans `api/wire` l'est aussi : 8 fichiers en portent, **6 antérieurs**
     (`post_sync_deltas_snapshot.go`, `post_sync_progression_queries.go`, `registry_catalog_expand.go`,
     `registry_notifications.go`, `registry_weapon_coverage.go` — 2026-07-06 ;
     `registry_monitoring_freshness.go` — 2026-07-10).
  2. **Incohérence interne du tableau G3** : `registry_replay_notify.go` (`RunReplayNotifyLoop`,
     `:82`, même campagne, même paquet, boucle de fond) est classé « OK — câblage » tandis que
     `RunReplayBuild` est un P1. Les deux sont des runners au sens de la note racine `api/`.
  3. **La conséquence centrale est fausse sur pièces.** « aucune de ces règles n'est joignable par
     un mock de port » / « les deux règles qui empêchent un job de mentir [n'ont] aucun contrat où
     être mockées » :
     - la frontière de sécurité est extraite derrière une **interface** `filmChunkResolver`
       (`registry_build_queue.go:136`) et une **fonction pure** `chunksFromResolver` (`:160`),
       toutes deux testées en direct (`registry_build_queue_cgo_test.go:207,228,236,244`) ;
     - `requireArtifactBeforeSuccess` est exercée bout-en-bout :
       `api/wire/build_queue_transport_e2e_cgo_test.go:149` (« succès annoncé sans artefact : status
       %d (attendu 409) ») ;
     - `reclaimBuildQueue` s'appuie sur `ReclaimExpiredBuildJobs`, testé
       (`ops/build_queue_cgo_test.go:244`).
     Les « seams privés » `r.replayFilmResolver` / `r.replayJobFactsFn` ne sont pas un contournement :
     ils sont documentés `registry.go:123-128` avec leur précédent de dépôt explicite
     (`sync.engine.customClient`), et le chemin de prod les laisse à nil.
  4. La sous-affirmation qui, elle, **tient sans réserve** : `replayMatchIdentity` duplique la
     cascade de noms de carte de `platform/duckdb/replay_map_repo.go` (`MapKeysForMatch:57`), et le
     commentaire `registry_replay_build.go:85-86` l'admet (« même ordre que ReplayMapRepo »). Rien ne
     tient les deux alignées.
- **Conséquence réelle reformulée** : deux fichiers neufs prolongent dans `api/wire` un motif de
  dérive vieux de deux mois (10 runners, 6 fichiers à SQL antérieurs) et y recopient une cascade de
  résolution de carte que `platform/duckdb` porte déjà — mais contrairement à ce qui est écrit, les
  règles métier du protocole ouvrier sont bien couvertes par des tests et par une frontière
  d'interface.

---

## Constat 2 — Trois services injectent `port.ReplayService` : TIENT (gravité → P2)

- **Ce que j'ai vérifié** :
  - `service/match_view_service.go:193`, `service/home_service.go:85`,
    `service/match_history_service.go:126` : les trois champs existent. Appels confirmés :
    `home_service_enrichment.go:49`, `match_history_service.go:207`,
    `match_view_data_loaders.go:364`.
  - `git grep "replaySvc\|port.ReplayService" a2719a68c -- apps/go-api/internal/service/` → **vide**.
    Le « 0 à v7.3.0 » de l'auditeur est exact.
- **Ce que l'auditeur n'a pas vu / a mal établi** :
  1. **Le compte est faux : il y en a QUATRE, pas trois.** `service/teammates/teammates_service.go:79`
     (`replaySvc port.ReplayService`, `WithReplay` `:145`, appel `:157`), ajouté le 2026-08-24
     (`d60397f64`). La repro proposée (`grep -rn "replaySvc port.ReplayService" apps/go-api/internal/service/`)
     le rend pourtant. Cela **aggrave** l'argument R6 mais invalide le chiffre publié (et le
     « Couplages horizontaux service→service neufs : 3 » des Chiffres mesurés).
  2. **La dépendance est une interface de `port/`, pas un type de service concret.** L'anti-pattern
     cité (`SKILL.md:66-69`) illustre `s.sessionSvc.GetSessions(...)`, un champ de service concret ;
     ici c'est le mécanisme de découplage prescrit qui est employé.
  3. **Les trois conséquences annoncées sont fausses sur pièces** :
     - « Tester `MatchViewService` exige un faux qui implémente tout `port.ReplayService` » — NON :
       le champ est optionnel et nil-tolérant (`match_view_builders_header.go:234` :
       `if h == nil || svc == nil || matchID == ""` → retour), `NewMatchViewService`
       (`match_view_service.go:197`) ne le prend pas, et **aucun** test de `MatchViewService`
       n'implémente l'interface.
     - « une 7ᵉ méthode … casse la compilation de trois services qui ne s'en servent pas » — NON :
       en Go, ajouter une méthode à une interface casse les **implémenteurs**, pas les consommateurs
       qui n'en détiennent que le type. Le seul implémenteur de prod est `service.replayService`.
     - « un changement de signature de `GetReplay` » — aucun des quatre ne l'appelle.
     - Le coût réel d'un faux est payé **une fois** : `stubReplayService`
       (`service/match_history_replay_test.go:22-50`, ~30 L) est partagé par `home` et
       `match_history`.
  4. Le type de remplacement suggéré existe bien (`port/replay_availability.go:12`) — vérifié, et son
     en-tête `:2-3` dit effectivement le rôle. Ce point du constat tient.
- **Conséquence réelle reformulée** : quatre services (et non trois) reçoivent l'interface entière
  du service de rejeu là où une interface de disponibilité à une méthode suffirait — une largeur de
  contrat inutile, sans le coût de test ni la fragilité de compilation avancés par le constat.

---

## Constat 3 — `service/replay_weapon_labels.go` importe `games/halo_infinite/replaylabels` : TIENT (gravité → P2)

- **Ce que j'ai vérifié** :
  - `service/replay_weapon_labels.go:34` (import) et `:46` (`replaylabels.Load(s.repoRoot, s.titleSlug)`)
    — confirmés ; l'en-tête `:24` affirme bien « TITLE-AGNOSTIC PAR CONSTRUCTION ».
  - Le service est construit **sans garde de capability** : `api/wire/registry_pages.go:147-150`
    (`replayServiceFor` → `service.NewReplayService(pdb.TitleSlug, …)`), donc `s.titleSlug` peut valoir
    `halo_5`. Le constat est exact sur ce point.
- **La piste de réfutation suggérée par le mandat est CLOSE** : `replaylabels` n'est pas
  title-agnostic malgré les apparences.
  - Son propre en-tête le déclare : `games/halo_infinite/replaylabels/catalog.go:1-2` — « assemble le
    CATALOGUE DE LIBELLÉS du rejeu 2D **d'Halo Infinite** » — et `:11-14` invoque explicitement
    l'ADR 0011 pour justifier le placement dans la couche titre.
  - Il **importe** `halo "levelup/go-api/internal/games/halo_infinite"` et l'utilise :
    `catalog.go:138` `adapter := halo.NewAssetURLAdapter()` ; il appelle aussi
    `weapons.FilmshellWeaponKeysByFamily()` (`:50,56`), registre d'armes du film Infinite.
- **Ce qui, en revanche, borne la gravité** :
  - **Aucune conséquence fonctionnelle aujourd'hui** : `config/titles/halo_5/mappings/` ne contient
    pas `replay_labels.toml` (seul `halo_infinite` l'a). `Load` échoue donc sur le 2ᵉ fichier
    (`catalog.go:44-47`) **avant** d'atteindre l'adaptateur Infinite ; le service journalise
    (`replay_weapon_labels.go:48`) et rend un document sans clés. Dégradation propre, conforme.
  - **L'argument R6 est faible** : les 3 imports `service/ → games/halo_*` portent sur **2** paquets
    (`rankedplaylists` ×2 dans `catalog_fetcher_service.go:20` et `world_stats_enricher.go:17`,
    `replaylabels` ×1). `replaylabels` est la 1ʳᵉ occurrence de son espèce, pas la 3ᵉ.
  - **Aucun écart antérieur à invoquer** : l'audit 06/08 a au contraire **retenu** en P1 la famille
    ADR 0012 (`.ai/AUDIT_…:204-232`, décision « escalade utilisateur »). Le constat n'est donc pas
    réfutable par précédent.
- **Élément confirmant que l'auditeur n'a pas cité** : `replaylabels/catalog.go:16-17` écrit
  « HORS LIGNE : ce chargement lit des fichiers ; il appartient à `cmd/replay-build`, **jamais à un
  chemin de requête** » — or `resolveWeaponLabels` est appelé sur le chemin de requête. L'import
  contredit le contrat écrit du paquet cible.
- **Conséquence réelle reformulée** : une arête `service/ → games/halo_infinite/` de plus,
  structurellement gênante pour ADR 0012/0031, mais sans effet observable en production (Halo 5
  dégrade proprement, faute de `replay_labels.toml`).

---

## Constat 4 — `analysis/replay` lit le disque et garde un cache de process : RÉFUTÉ

- **Ce que j'ai vérifié** : les 8 sites d'I/O disque cités existent bien
  (`callouts_catalog.go:155`, `geometry.go:115`, `map_background.go:123`,
  `map_background_index.go:190,308`, `map_weapon_pads_catalog.go:130`,
  `objectives_catalog.go:78`, `structure.go:82`), ainsi que `indexFondsMu`/`indexFondsCache`
  (`map_background_index.go:291-292`) et `signatureIndatable` (`:303`).
- **Ce que l'auditeur n'a pas vu** :
  1. **Erreur de fait sur le périmètre : 2 des « 7 fichiers créés après v7.3.0 » sont antérieurs.**
     `git log --diff-filter=A` : `geometry.go` et `structure.go` datent du **2026-07-31**
     (`2044b7139`, « porter l additif du rejeu 2D sur la branche saine »). Et ils étaient présents,
     **avec les mêmes lectures disque et aux mêmes lignes**, dans l'arbre audité le 06/08 :
     `git grep "os\." 634e079f2 -- 'apps/go-api/internal/analysis/replay/*.go'` →
     `geometry.go:90 os.Open`, `structure.go:82 os.ReadFile`, `deaths_source.go:40 os.ReadFile`.
     L'audit du 06/08 a examiné ce paquet (P1 ADR 0012, `.ai/AUDIT_…:204-232`) et **n'a pas retenu**
     la violation de pureté.
  2. **La violation « 0 side-effect » dans `analysis/` est antérieure à la campagne et plus large** :
     `analysis/filmdec/film_packets.go:50,95` (`os.ReadFile`, `os.Stat`, 2026-07-31),
     `filmdec/map_bounds.go:97`, `filmdec/keyframe_entity_queue.go:49`.
  3. **L'« état de process dans `analysis/` » est lui aussi antérieur** :
     `analysis/combat_yield.go:28` `var excludeAssistsFromYield atomic.Bool` (2026-06-18),
     `filmdec/decode_gate.go:23` `var processDecodeMu sync.Mutex`,
     `filmdec/registry_fingerprint.go:102` `var registryWarned sync.Map`.
  4. **Le préjudice annoncé pour le cache est contredit par le code.** « non rejouable en parallèle
     sans se marcher dessus » : le cache est protégé par `indexFondsMu` (`:291`), **clé par
     répertoire** (`indexFondsCache map[string]…`), et invalidé par une signature de contenu
     (nom+taille+mtime, `signatureDossier:329-342`) sans TTL — le raisonnement est écrit `:280-289`.
     Deux tests sur deux répertoires temporaires distincts ne peuvent pas se croiser. Le fichier a
     même prévu le cas dégénéré : un sidecar indatable rend une signature volontairement non
     reproductible (`:337`).
  5. **Le volet `service/` du constat est écarté par le document lui-même** : ligne 221 de G3,
     « `service/*.go` accédant au système de fichiers | Motif établi et antérieur ». C4 le remet
     pourtant dans son titre et dans son « Où ». Contradiction interne.
- **Conséquence réelle reformulée** : le constat re-décrit, sur une base de comparaison décalée d'un
  jour, une propriété du paquet `analysis/replay` déjà auditée le 06/08 et déjà portée par
  l'escalade ADR 0012 ouverte — sans le préjudice de concurrence annoncé, que le mutex et la clé par
  répertoire excluent.

---

## Constat 5 — File de jobs live câblée `handlers → ops` sans passer par `port/` : RÉFUTÉ

- **Ce que j'ai vérifié** :
  - `api/handlers/build_worker.go:48,52,56` — les trois seams sont des **types fonction déclarés
    dans le paquet `handlers`** (`BuildQueueClaimer`, `BuildQueueCompleter`, `BuildQueueBeater`), plus
    `build_worker_artifact.go:42` `BuildQueueArtifactStorer`. Le handler n'appelle **aucune** fonction
    de `ops` : il appelle le seam injecté. L'import `ops` ne sert qu'à deux DTO plats
    (`ops.CompleteBuildJobRequest`, 6 champs scalaires, `ops/build_queue.go:186-194` ;
    `ops.HeartbeatRequest`, 7 champs, `:246-256`) et à la sentinelle `ops.ErrBuildJobNotClaimed`.
  - `grep -n "BuildQueue" internal/port/*.go` → un seul hit, un commentaire (`services.go:232`) qui
    porte d'ailleurs sur `domain.BuildQueuePayload`, pas sur l'absence de port. Le constat est exact
    sur ce point de forme.
- **Ce que l'auditeur n'a pas vu** :
  1. **La conséquence centrale est démentie par un test qui existe déjà.** « Le handler ne peut pas
     être monté contre un mock de port » : `api/handlers/build_worker_test.go:44-58`
     (`okWorkerHandler`) construit le handler sur **trois stubs anonymes**, et `serveBuildWorker`
     (`:25-30`) le monte sur un routeur `chi` avec `httptest.NewRecorder()` — c'est exactement la
     stratégie prescrite par `SKILL.md:83` (« `httptest.NewRecorder` + mock `port.*Service` »).
     Six tests HTTP en découlent (`:73,86,107,129,150,165`).
  2. **Le motif « seam fonction déclaré dans `handlers` » est la norme du dépôt** :
     `grep -c "^type [A-Za-z]* func(" internal/api/handlers/*.go` → **31 fichiers** en déclarent
     (`admin_monitoring.go` 9, `media.go` 4, `sync_handler.go` 3, `prestige.go` 2…). Un type fonction
     déclaré côté consommateur **est** un port au sens Go.
  3. **Incohérence interne du tableau G3** : `admin_actions_replay_build.go` (3 seams fonction, même
     campagne, même registry, même mécanique) est classé « OK (pattern admin_actions établi) ».
  4. **Le précédent `handlers → ops` est antérieur et bien plus fort** : `handlers/admin_logs.go`
     (2026-06-12) **appelle** directement `ops.ListLogModules` (`:83`), `ops.LogTailOptions` (`:110`)
     et `ops.TailModuleLog` (`:130`) depuis le handler ; `handlers/progression.go` importe `ops`
     depuis le 2026-07-26. `build_worker.go` ne fait rien d'équivalent.
- **Résidu (non retenu)** : les deux structs de requête gagneraient à vivre en `domain/`, où
  `domain.BuildQueueJob` / `BuildQueuePayload` / `BuildArtifactReceipt` sont déjà. C'est un
  déplacement de DTO, pas un court-circuit de couche.
- **Conséquence réelle reformulée** : le sous-système est bel et bien découplé par des seams
  injectables et testé en `httptest` ; il ne reste qu'un emplacement discutable pour deux DTO,
  avec un précédent plus grave et non contesté à côté.

---

## Constat 6 — Fusionneur crédit/film : algo pur dans `persist/`, SQL dans `persist/`, seuil dans `migration/`, 2ᵉ implémentation SQL : RÉFUTÉ

- **Ce que j'ai vérifié** : tous les emplacements cités existent
  (`persist/kill_events_merge.go:167,223,237,273,298` ; `kill_events_credit.go:117,191,221-229,346` ;
  `kill_events_film_pass.go:53,63` ; `migration/steps_shared_weapon_hit_distance.go:80`
  `const WeaponHitsMinShots = 8` consommé par `persist/weapon_hit_distance_persister.go:42,54` ;
  `migration/steps_shared_kill_events_credit_base.go:115-148,240-277`).
- **Ce que l'auditeur n'a pas vu — la thèse causale du constat est FAUSSE sur pièces** :
  1. Le constat écrit : « l'arête `persist → migration` n'existe que pour lire un entier
     (`WeaponHitsMinShots`) ; **la supprimer rend l'import `migration → persist` possible** et la
     duplication inutile. » Or `grep -h "^package " internal/persist/*_test.go | sort | uniq -c`
     → **28 fichiers, tous `package persist`** (tests **in-package**), dont **10+** importent
     `internal/migration` (`kill_events_persister_test.go:21`, `shared_persister_test.go:32`,
     `bomb_stats_persister_test.go:20`, `player_persister_test.go:28`, `pve_persister_test.go:15`,
     `usage_summary_persister_test.go:21`, `metadata_persister_test.go:14`,
     `kill_position_persister_test.go:19`, `events_completion_persister_test.go:24`,
     `post_sync_enrichment_persister_test.go:10`) — c'est la doctrine du dépôt (fixtures construites
     par les migrations réelles). Un import `migration → persist` créerait donc un **cycle de
     compilation réel** dans le binaire de test de `persist`, indépendamment de `WeaponHitsMinShots`.
     Le commentaire du code (`steps_shared_kill_events_credit_base.go:149-151` : « `migration` ne peut
     importer ni `persist` (**cycle a la compilation de ses tests**) ») est exact ; l'auditeur l'a lu
     comme portant sur l'arête de production. **Le traitement v2 proposé ne fonctionne pas.**
  2. **La duplication a déjà été vue, nommée et jugée par l'audit du 06/08.**
     `.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md:340-353` : « [P2] La passe kill-events recomposée hérite du
     `publishable` du film … Où : `internal/persist/kill_events_merge.go:185` + **miroir SQL**
     `internal/migration/steps_shared_kill_events_credit_base.go:136,250,266` ». L'audit a donc eu les
     deux implémentations sous les yeux, les a désignées comme miroirs, et n'en a retenu qu'un **défaut
     de contenu (P2, backlog)** — pas une violation de frontière. Ce n'est pas le seul écart A2 cité
     par G3 (ligne 458), c'est un constat P2 distinct que G3 ne mentionne pas.
  3. **Les fichiers ne sont pas des produits de la campagne** : `kill_events_merge.go`,
     `kill_events_credit.go`, `kill_events_film_pass.go` et `kill_events_persister.go` sont **tous
     présents à `634e079f2`** (cf. constat préliminaire). Ils ont donc été soumis à l'auditeur A4 du
     06/08, qui n'a soulevé ni l'emplacement de l'algo pur ni les SELECT.
  4. **Le SELECT dans `persist/` est un motif établi de trois mois** : 12 fichiers non-test en
     portent, dont **7 antérieurs** — `shared_persister.go`, `player_persister.go`,
     `pve_persister.go` (2026-05-23), `lusr_append_only_persister.go` (2026-05-24),
     `shared_social_persister.go` (2026-05-25), `events_completion_persister.go` (2026-05-31),
     `shared_social_persister_batch.go` (2026-06-02).
  5. **Le déplacement vers `analysis/killmerge/` se heurte à ADR 0030.**
     `MergeCreditAndFilm(base, film KillSourceBatch) (KillSourceBatch, MergeStats, error)`
     (`kill_events_merge.go:167`) prend **et rend** l'agrégat d'écriture de `persist`
     (`kill_events_persister.go:126`). ADR 0030 (`docs/adr/0030-persist-write-aggregates.md`, D-1)
     décide que l'agrégat d'écriture est une « closed, single-package construction … Enforcement
     lives entirely **inside `internal/persist`** … no new package ». Loger dans `analysis/` une
     fonction dont la signature est l'agrégat d'écriture inverserait le sens des couches
     (`analysis → persist`) ou obligerait à sortir l'agrégat de `persist`.
  6. **« Rien ne les tient alignées » est excessif** : chacune est testée par **mutation** contre la
     même liste de propriétés, énumérée à l'identique dans les deux en-têtes —
     `steps_shared_kill_events_credit_base_test.go:13-19` (5 tests : base crédit = liste des morts,
     le film enrichit sans remplacer, orphelins conservés, dédup sur l'identité, révision du
     décodeur, idempotence) et `persist/kill_events_merge_test.go`. Il manque un harnais
     d'équivalence croisé, pas des tests.
  7. **Les contrats d'erreur divergents sont motivés et écrits** :
     `steps_shared_kill_events_credit_base.go:165-172` — la migration tourne **au boot sur toutes les
     bases**, une erreur y bloquerait le démarrage de l'application ; le fusionneur en mémoire, lui,
     travaille match par match pendant un cycle de sync où l'échec est rattrapable. Ce n'est pas
     « deux contrats volontairement différents » sans raison, c'est une décision datée et justifiée.
- **Résidu qui tient, mais hors P1** : `WeaponHitsMinShots = 8`, seuil de publication, est déclaré
  dans `migration/` (à côté de `WeaponHitDistanceDecoderRev`, `:75`) et non en `domain/`. Le
  consommateur en fait explicitement « sa copie UNIQUE » (`weapon_hit_distance_persister.go:44`),
  donc conforme à R6. Emplacement discutable, sans conséquence — un P3.
- **Conséquence réelle reformulée** : la duplication Go/SQL est structurellement **imposée** par un
  cycle d'import que le remède proposé ne lève pas, elle avait déjà été identifiée et jugée par
  l'audit du 06/08, et les emplacements incriminés dans `persist/` prolongent un motif antérieur au
  périmètre.

---

## Bilan : 1 tient, 3 réfutés, 2 requalifiés

- **Tient tel quel** : aucun au P1 annoncé.
- **Tient, gravité abaissée à P2** : C1 (SQL/runner dans `api/wire` — letter breach réelle sur des
  fichiers neufs, mais toutes les conséquences de testabilité sont fausses et le motif a 10
  précédents), C2 (couplage horizontal réel — mais quatre sites et non trois, et les trois préjudices
  annoncés sont techniquement faux).
- **Tient, gravité abaissée à P2** (compté dans les requalifiés) : C3 — la piste de réfutation
  « `replaylabels` n'a rien d'Infinite » est close (le paquet importe `games/halo_infinite` et se
  déclare Infinite), mais l'effet en production est nul (Halo 5 n'a pas `replay_labels.toml`) et
  l'argument R6 ne compte qu'une occurrence, pas trois.
- **Réfutés** : C4 (2 des 7 fichiers sont antérieurs et étaient sous les yeux de l'audit 06/08 ; le
  motif I/O et l'état de process dans `analysis/` sont antérieurs ; le préjudice de concurrence est
  contredit par le mutex + la clé par répertoire ; le volet `service/` est écarté par le document
  lui-même), C5 (le handler EST monté sur des stubs en `httptest` par un test existant ; le motif
  seam-fonction est la norme du dépôt et le précédent `handlers → ops` est antérieur et plus fort),
  C6 (la thèse causale et le remède sont faux — le cycle vient des 28 tests in-package de `persist` ;
  la duplication avait été identifiée comme « miroir SQL » et jugée P2 de contenu par l'audit 06/08).

**Défaut de méthode commun** : la base `a2719a68c` (v7.3.0, 2026-08-04) tombe la veille du merge
`feat/replay2d-prod` (2026-08-05). Elle fait apparaître comme neuf tout un chantier que l'audit du
2026-08-06 avait déjà dans son périmètre — ce qui fonde à tort les phrases « ne couvre pas ce
constat » de C4 et C6.
