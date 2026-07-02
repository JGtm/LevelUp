# Audit d'architecture — apps/go-api/internal/ + config/titles/

> Date : 2026-07-02 — Revue seule, aucune correction appliquée.
> Méthodologie : audit multi-agent (27 agents, ~3,75 M tokens, 1 228 lectures) — 10 dimensions
> parallèles (couches x4, structure x2, title-agnosticism x2, perf DuckDB x2) + 3 dimensions
> complémentaires ajoutées par un critique de complétude (contrat HTTP/OpenAPI, cohérence
> capabilities coarse/fine, modèle de connexions). Chaque lot de findings contre-vérifié
> adversarialement à la ligne près : 17 faux positifs éliminés, sévérités recalibrées.
> Le vérificateur de la dimension capabilities a échoué (erreur API) ; ses 4 claims structurants
> ont été re-vérifiés manuellement au grep — tous confirmés.
>
> Bilan : 173 findings confirmés — 0 Bloquant, 55 Majeurs, 118 Mineurs.
> 21 findings correspondent à de la dette déjà trackée ADR/plans, marqués [TRACKÉ].

## Résumé exécutif

| Axe | Santé | Verdict |
|---|---|---|
| 1. Respect des couches | Moyen | `handlers/`, `middleware/`, `port/`, `domain/` sains. La racine `api/` (39 fichiers, ~9,5 k L) est devenue une deuxième couche service de facto (SQL dans 9 fichiers, pipeline post-sync, runners admin, cascade auth). 3 foyers dans `service/`. |
| 2. Structure | Moyen | Fondations propres, mais 5 god packages (platform/duckdb 143 fichiers, service 127, sync 111, api racine 39, analysis) et une dizaine de god-files dont `NewRouter` ~1 470 L. |
| 3. Title-agnosticism | Moyen | Socle solide (archlint, registres par slug, pipeline TOML fonctionnel) mais des fuites HINF actives sur les surfaces H5 et un ratchet anti-slug contournable. |
| 4. Performance DuckDB | Moyen | Paramétrage SQL exemplaire (0 injection), N+1 des hot paths disciplinés (santé Bon). Vrai sujet : ~8 lecteurs de `match_skill_rank` brut (piège `_latest`), un `LoadAll` full-history par hit, handle player RW 1-connexion partagé HTTP+sync. |

**Pourquoi 0 Bloquant** : les deux candidats ont été rétrogradés sur pièces. L'UPSERT per-match
OpenSpartan sur shared est sérialisé sous writer lease et déclaré légitime par l'allowlist du
tripwire ART ; le `OpenReadOnly` des runners admin partage en réalité le handle caché hors
fenêtres de swap. Les risques restent réels mais conditionnels, pas systémiques.

**Forces confirmées (à préserver)** : `port/` propre, pureté d'imports parfaite dans `domain/` /
`analysis/`, garde-fous archlint réels avec allowlists vides ou décroissantes, dette ADR 0025
`home_service` -> `PersistSink` déjà résolue (`port.HomePersistSink`), write-path sync massivement
assaini (append-only + generation_id), zéro SQL non paramétré sur des valeurs externes.

---

## Findings Majeurs (55)

### Axe 1 — Respect des couches

#### La racine api/ : deuxième couche service de facto

1. **Pipeline post-sync progression : SQL + formules produit dans api/** —
   `apps/go-api/internal/api/post_sync_progression_queries.go:139`. Loaders SQL shared/player et
   formules produit (offensive_conversion, seuils P80 en constantes locales) dans la couche HTTP ;
   l'orchestrateur `post_sync_progression.go` fait 598 L. Nuance vérifiée : l'accès shared passe
   bien par SharedReader (bon pattern B-swap). Reco : extraire `service/postsync/` (ou
   `progression/orchestrator/`), SQL vers des repos `platform/duckdb`, formules vers `analysis/`.
2. **SnapshotPlayerState : 9 requêtes inline dont match_skill_rank lu BRUT** —
   `apps/go-api/internal/api/post_sync_deltas_snapshot.go:74`. Capture d'état joueur (career, PSA,
   citations, skill tier, KD) = travail repo+service exécuté depuis api/ ; la lecture brute
   l.142-153 peut servir une ligne périmée -> transition skill_tier fantôme dans les notifications
   push (le détecteur l.151 n'a de surcroît aucun tiebreak à start_time égal). Reco :
   `PlayerSnapshotRepo` côté platform + bascule sur `match_skill_rank_latest`.
3. **Écritures sync_meta sans lease dblease (ADR 0013), helper dupliqué x2** —
   `apps/go-api/internal/api/notifications_title_ready.go:141` + copie dans
   `notifications_boot.go:112`. `OpenReadWrite` direct sans `AcquirePlayerWriterTimeout` ;
   title_ready s'exécute en fin de sync live -> écriture non sérialisée avec un sync tenant le
   writer. Reco : helper unique côté platform, sous lease (pattern `prestige_lazy_service.go:119`).
4. **ExpandPlaylistChildren : runner complet (fetch API + SELECT shared + DDL + upserts) dans api/**
   — `apps/go-api/internal/api/registry_catalog_expand.go:94`. Écritures ART-safe (vérifié), mais
   DDL hors `internal/migration` (piège « CREATE TABLE IF NOT EXISTS + PK ») et 3e copie du pattern
   upsert ART-safe. Reco : déplacer vers ops/ ou service/, DDL vers migration, factoriser.
5. **Cascade complète de refresh tokens (métier ADR 0023) dans api/** —
   `apps/go-api/internal/api/registry_auth.go:169`. ~130 L dupliquant la sémantique du helper
   canonique `RefreshHaloTokensViaStoreFirst` ; la suppression Phase 5 du fallback legacy devra
   toucher deux implémentations. Reco : déplacer vers `platform/auth`.
6. **dataQualityHandles : OpenReadOnly direct sur shared, en conflit avec les fenêtres RW du
   provider (ADR 0016)** — `apps/go-api/internal/api/registry_data_quality.go:33`. Hors swap le
   handle est partagé (vérifié), mais pendant une fenêtre RW : erreurs « different configuration »
   et un refcount RO retenu par un agrégat admin long peut faire échouer le swap du sync prod.
   Alimente 5+ runners admin. Reco : lire via `cfg.SharedProvider` (pattern
   `acquireProgressionSharedRead`).
7. **Métier Explorer CSR dans la factory DI, divergence i18n déjà concrétisée** —
   `apps/go-api/internal/api/registry_pages.go:380`. Duplication de
   `sync.augmentWithActiveRankedCSRs` avec `res.PlaylistName = pl.NameFR` alors que l'original
   (`sync/career.go:255`) pose `NameEN` — les deux copies ont déjà divergé. Reco : une seule
   implémentation partagée + résolution du nom via semantic adapter/locale.

#### service/

8. **Couplage service->service concret : HomeService -> *CareerLiveService sur le chemin HTTP
   chaud** — `apps/go-api/internal/service/home_service.go:264`. Anti-pattern explicitement
   interdit par arch-rules ; le chemin live de prod n'est pas mockable. Reco : interface
   consumer-side étroite (`spartanIdentityProvider`), pattern `explorer_service.go:65`.
9. **Persistance catalogue complète (~200 L de SQL + *sql.DB) dans CatalogFetcherService** —
   `apps/go-api/internal/service/catalog_fetcher_service.go:73`. 7 écritures inline sur 6 tables
   metadata, exposé HTTP (drain) et CLI ; connaissance anti-ART dupliquée hors de persist/. Même
   dérive dans `openspartan_post_import_service.go:170`. Reco : CatalogRepository
   (platform/duckdb) ou Persister dédié, via port.*.
10. **Le service construit le client HTTP Halo du moteur sync ; types sync dans le contrat
    CareerFetcher** — `apps/go-api/internal/service/career_live_fetcher.go:150`. 5 fichiers
    `career_live_*` importent internal/sync ; HaloAPIClient est Infinite-only -> bloque la
    généralisation career-live à H5. Reco : factory injectée côté registry, promouvoir
    CareerRankData/SpartanCustomizationData en types domain/.
11. **Chemin './data/players' en dur, layout legacy non title-aware** —
    `apps/go-api/internal/service/openspartan_import_service.go:481` (+ câblage `server.go:1344`).
    Le stash friends atterrit hors `data/titles/{slug}/players` ; défaut relatif dépendant du cwd.
    Reco : PathResolver injecté.
12. [TRACKÉ] **Write-IO média direct dans media_service.go + media_index_service.go** —
    `apps/go-api/internal/service/media_service.go:357`. Seule exemption restante au critère
    Phase 2 ADR 0025, trackée par l'allowlist décroissante de `no_duckdb_import_test.go`. Reco :
    exécuter le plan de l'allowlist (extraire MediaRepository/MediaStore, vider l'allowlist).

#### analysis/ et chemins physiques

13. **Pipeline film Halo Infinite entier resté dans analysis/** —
    `apps/go-api/internal/analysis/weapon_data.go:3` + weapon_scanner/parser/correlation/
    reconciliation.go, highlight_event_parser.go, spawn_detection.go, kill_attribution.go.
    Contredit l'état final déclaré par ADR 0012 ; corpus 100 % spécifique au format film Infinite
    (H5 a des weapon_kills natifs), non tracké comme dette. Reco : migrer vers
    `games/halo_infinite/film/` (migration mécanique, vérifiée).
14. **Bypasses du PathResolver : ~8 sites composent data/... à la main** —
    `apps/go-api/internal/api/server.go:290` (`data/cache/jobs.json` alors que
    `PathResolver.JobsCachePath()` retourne exactement ce chemin), :651, :1112, :1344 ;
    `ops/seed_demo.go:392` ; `ops/seed_demo_multititle.go:43` (ré-implémente le layout
    `data/titles/{slug}/players`). Le contrat du resolver (`domain/title/registry.go:383`)
    interdit explicitement ces joins. Reco : appels PathResolver + garde-rail archlint.

#### Écritures à risque ART

15. **Import OpenSpartan (chemin HTTP) : UPSERT ON CONFLICT DO UPDATE per-match sur shared, hors
    BatchBuilder** — `apps/go-api/internal/service/openspartan_import_service.go:318` -> rejoue
    les UPSERT legacy de `sync/writes.go:68`, non gated par LEVELUP_PERSIST_BATCH. Rétrogradé de
    Bloquant après vérification (sérialisé sous AcquireWriter, allowlist explicite du tripwire),
    mais c'est le dernier chemin prod du pattern déclencheur ART #23046 sur la DB au plus gros
    blast radius. Reco : router writeOneMatch vers persist.SharedPersister (INSERT-only +
    pre-check registry), comme le livesync H5.
16. **UPDATE bulk multi-row nu sur match_registry, atteignable via endpoint admin HTTP, hors
    couverture du tripwire** — `apps/go-api/internal/sync/backfill_registry_names.go:98` (caller
    in-process : `api/registry_actions.go:94`). Le profil exact décrit par no_art_patterns_test.go
    comme vrai déclencheur ART, mais la regex du tripwire ne matche que la forme
    `UPDATE ... FROM (VALUES ...)`. Précédent réel : les UPDATE nus qui ont FATAL-invalidé
    metadata.duckdb. Reco : convertir en row-by-row par match_id ou réserver au CLI
    serveur-arrêté ; étendre le tripwire aux UPDATE multi-row nus sur les tables critiques.

### Axe 2 — Structure

17. **God package platform/duckdb : 143 fichiers non-test / 62,6 k L dans un seul package Go** —
    aucune frontière compilateur entre les repos home/career/media/prestige ; les halo5_*.go y
    cohabitent. Reco : duckdb/core + extraction par domaine (1 domaine = 1 PR, commencer par
    prestige) ; halo5_*.go vers games/halo_5/ ou duckdb/halo5/.
18. **God package service/ : 127 fichiers plats** — couplages inter-services non vérifiables par
    le compilateur. Reco : sous-packages par feature, absence d'import croisé vérifiable par
    archlint ; commencer par teammates (13 fichiers).
19. **God package sync/ : 111 fichiers racine malgré sync/v2/ (ADR 0027)** —
    `apps/go-api/internal/sync/engine.go:13`. Reco : extraire sync/skill/ (17 fichiers),
    sync/snapshot/ (6), geler la racine par ratchet, router le neuf vers v2.
20. **Racine api/ : 39 fichiers / 9 483 L mêlant DI, boot, post-sync** —
    `apps/go-api/internal/api/registry.go:26`, 18 registry_*.go + 7 post_sync_*.go. Reco :
    api/wire/ pour la DI, post_sync vers service (cf. n° 1) ; cible < 10 fichiers racine.
21. **Client HTTP Halo Infinite codé en dur dans sync/** —
    `apps/go-api/internal/sync/halo_client.go:1` + 6 fichiers halo_client_*/halo_skill*. URLs
    halowaypoint dans le moteur « générique », contraire à ADR 0012/0025 (games/halo_5/client.go
    montre la cible). Reco : déplacer vers platform/halo/ ou games/halo_infinite/client/.
22. **NewRouter ~1 470 lignes** (`apps/go-api/internal/api/server.go:245`, fichier 1 717 L) —
    stores + middlewares + registre titres + chargements SQL + ~30 services + ~80 routes ; le
    doc-comment est physiquement rattaché à la mauvaise fonction (vérifié). Facteur 18x le seuil
    de 80 L, sans exemption. Reco : buildStores/buildTitleRuntime/applyMiddlewares/mountXxx —
    NewRouter < 100 L.
23. **SyncEngine.run() = 483 L** (`apps/go-api/internal/sync/engine.go:179`) — init + leases +
    DBs + fetch + persist + reconcile, avec un raisonnement d'ordre des defer fragile (deadlock
    shared RW documenté en commentaire). Reco : poursuivre le précédent engine_backfills.go
    (4 sous-étapes nommées).
24. **auto_sync.go 1 083 L : scheduler + factory engine + métriques + convergence**
    (`apps/go-api/internal/scheduler/auto_sync.go:717`) — la résolution runner-par-titre (sensible
    activation H5) cohabite avec la télémétrie ; BuildEngine (~90 L) et RunOnceTrigger (~117 L)
    sans exemption. Reco : scission en 3 fichiers.
25. **SeedDemo() = 203 L sans exemption, fichier 1 027 L multi-responsabilités**
    (`apps/go-api/internal/ops/seed_demo.go:224`) — sur le chemin de déploiement où des incidents
    regen-demo sont documentés. Reco : scission extract/configs/identity + phases nommées.

#### Contrat HTTP & gouvernance (dimension complémentaire)

26. **internal/api/gen = dead code périmé** (`apps/go-api/internal/api/gen/types.gen.go:3`) —
    2 536 L, 0 importeur Go (vérifié repo-wide), spec modifiée le 2026-06-30 sans regen,
    3 exclusions de tooling entretenues pour rien, et le message d'erreur du drift-test prescrit
    encore `make gen`. Anti-pattern « dead code museum ». Reco : supprimer gen/ + cible make +
    exclusions ; garder openapi.yaml -> openapi-typescript + drift-test Huma comme unique chaîne.
27. [TRACKÉ] **22 schémas OpenAPI DIVERGENT passent la CI en silence**
    (`apps/go-api/internal/api/openapi_schema_drift_test.go:111`) — confirmé par exécution réelle
    du test (DIVERGENT=22 dont BootstrapResponse, seul MISSING>0 échoue) ; generated.ts front
    potentiellement faux sur 22 types. Reco : résorber via le mode emit prévu, puis durcir le gate
    (t.Errorf si divergent > 0 — ratchet déjà annoncé en commentaire).

### Axe 3 — Title-agnosticism

28. **MediaRepo applique la classification de modes Halo Infinite à TOUS les titres** —
    `apps/go-api/internal/platform/duckdb/media_repo.go:112` (+ translations :240, q37_enrich).
    Des médias H5 existent -> catégories fausses/vides sur la page Médias H5. Couplage direct
    platform/duckdb -> games/halo_infinite alors que le seam d'injection existe
    (analysis.PairNamePrefixesFunc). Reco : injecter la classification par titre au wiring.
29. **Providers CSR Explorer/Compare câblés HINF quel que soit le titre** —
    `apps/go-api/internal/api/registry_pages.go:360` (+ :812 Compare). Pour un joueur H5, l'encart
    cible affiche le CSR Halo Infinite — fuite de données cross-titre. Reco : gate capability
    (csr.live) ou map slug->provider (pattern MT-09).
30. **La regex du ratchet no_slug_comparison est contournée par l'alias titlePkg. et
    TitleSlug(ctx)** — `apps/go-api/internal/archlint/no_slug_comparison_test.go:35`. Vérifié :
    `sync/comeback.go:34/95` et `sync/coordinator.go:316` passent sous le radar (sites justifiés
    par commentaire — le défaut est leur non-détection). La garantie « allowlist vide » est
    illusoire. Reco : élargir la regex (`(?:\w+\.)?DefaultSlug` + cas TitleSlug(ctx)), allowlister
    les sites détectés.
31. **Labels d'outcome FR en dur en 3 copies (service + analysis + notify/discord), seam TOML
    contourné** — `apps/go-api/internal/service/match_history_service.go:34` +
    `analysis/home_locale.go:55`. Incohérence EN déjà réelle : la home émet "Victory/Defeat" quand
    outcomes.toml (et notify) disent "Win/Loss". Reco :
    resolver.Semantic(slug).Outcomes().Label(...) partout, littéraux en failsafe.
32. **URL Waypoint halo-infinite en dur alors que le TitleAssetURLAdapter est injecté dans la
    fonction** — `apps/go-api/internal/service/match_view_builders_header.go:59` +
    `match_history_service_enrich.go:278`. Lien mort visible sur chaque Match View H5 en prod.
    Reco : URL derrière l'adapter, capability/None pour les titres sans page match.
33. **Labels KPI en dur, FR/EN mélangés dans une même réponse** —
    `apps/go-api/internal/service/compare_service.go:472` (~20 labels),
    `session_compare_service.go:414` ("Win Rate" + "Précision" dans la même réponse — anglicisme
    en prod), `timeseries_service_tabs.go:48`. Ces clés existent dans fields.toml avec labels
    EN+FR. Reco : FieldMappingSet du titre ou key-only + labelling front via /field-mappings.
34. [TRACKÉ] **fields.toml halo_5 resté au stade squelette : 5 FieldKeys vs 59 pour Infinite,
    titre actif en prod** — `config/titles/halo_5/mappings/fields.toml:5`. La prémisse
    « coming_soon » du header est périmée ; le contrat « section absente = non supporté » est
    devenu faux. Reco : compléter + test de parité FieldKeys requis vs déclarés par titre actif.
35. [TRACKÉ] **5 handlers Ascension épinglés DefaultSlug — le titre du contexte est ignoré** —
    `apps/go-api/internal/api/server.go:1554` (+1579/1584/1588/1609/1614). Sous titre H5 :
    /streaks, /records, /milestones, /profile, /patterns, /campaign servent des données Infinite
    (dette MT-19 trackée, mais elle bloque le chantier actif Phase 1b). Reco :
    ctxkeys.TitleSlug(ctx) ou a minima RequireCapability -> 503 propre.
36. [TRACKÉ] **Le seam auth.toml par titre n'est consommé par AUCUN code de prod** —
    `apps/go-api/internal/platform/auth/halo_exchange.go:71` : toute la chaîne d'échange est
    câblée DefaultHaloAuthDescriptor() ; LoadAuthDescriptor a zéro call-site hors tests
    (re-vérifié). H5 fonctionne par coïncidence d'audiences ; MT-02 marqué « done » à tort. Reco :
    résolution au boot par titre actif, fallback Default réservé au titre par défaut.
37. **Contradiction coarse/fine en prod : H5 déclare la coarse engagement pendant que la fine
    engagement.score = not_exposed** — `config/titles/halo_5/title.toml:48` vs
    `config/titles/halo_5/mappings/capabilities.toml:28` (re-vérifié). Routes ouvertes + front
    affiché + livesync qui calcule, pendant que /feature-matrix annonce indisponible. Aucune garde
    ne relie les deux référentiels. Reco : réconcilier + test générique des paires miroir
    coarse<->fine.
38. **Le titre par défaut échappe au mécanisme TOML : manifest halo_infinite codé en dur, tout
    title.toml déposé serait ignoré en silence** — `apps/go-api/internal/domain/title/registry.go:259`
    + skip muet `config_loader.go:185` (re-vérifié). Deux sources de vérité selon le titre. Reco :
    WARN explicite si un TOML infinite existe, ou parity-test built-in vs TOML versionné.

### Axe 4 — Performance DuckDB

#### Famille « lecture brute vs vue _latest » (piège ADR 0026 documenté, ~8 chemins HTTP)

39. **Home (Q26) : LEFT JOIN match_skill_rank brut -> rating des tuiles non déterministe** —
    `apps/go-api/internal/platform/duckdb/queries_home_citations.go:100` ; le merge Go « dernière
    ligne scannée gagne » sans ORDER BY. H5 (3 lignes/match) le plus exposé. Reco : joindre
    match_skill_rank_latest.
40. **Historique (Q5PlayerSkillRankHistory) : tier affiché non déterministe** —
    `apps/go-api/internal/platform/duckdb/queries_career.go:172` ; le bricolage winProb de
    mergeHistorySkillRanks prouve que le problème est connu sans être traité à la racine. Reco :
    _latest, le bricolage devient inutile.
41. **Compare : MAX(rating_value) LUSR sur table brute -> ATH périmée incorrigible** —
    `apps/go-api/internal/platform/duckdb/compare_repo.go:129` (+ :170). Après un re-backfill à la
    baisse (campagnes réelles), l'ATH ne redescend jamais. Reco : _latest aux 2 sites.
42. **Leaderboard local : charge toutes les versions brutes** —
    `apps/go-api/internal/platform/duckdb/leaderboard_repo.go:65` — doublons dans les agrégats.
    Reco : _latest en Phase A.
43. **Patterns : loadSkillRanks brut, map écrasée en ordre arbitraire** —
    `apps/go-api/internal/platform/duckdb/patterns_repo.go:245`. Reco : _latest.
44. **LoadAll recharge TOUT l'historique à chaque hit de page (pas de LIMIT, IN-lists de milliers
    de placeholders)** — `apps/go-api/internal/platform/duckdb/match_history_repo.go:32`. Nuance
    vérifiée : le push-LIMIT naïf est impossible (le calcul de placement exige l'ensemble
    chronologique). Reco : cache par joueur invalidé post-sync, ou matérialiser le placement
    séparément.
45. **Fan-out notifications : OpenReadOnly direct sur shared** —
    `apps/go-api/internal/api/registry_notifications.go:162`. Pendant une fenêtre RW du B-swap :
    échec intermittent -> notifications d'upload média non fan-outées, précisément dans la fenêtre
    post-match. Reco : OpenReadForQuery (existe pour exactement ce cas, incident 2026-06-01
    documenté dans db.go).

#### N+1 sur chemins HTTP interactifs

46. **Match View : Q29 GetHistoryForAvg exécutée par ami tracké du scoreboard** —
    `apps/go-api/internal/service/match_view_data_loaders.go:386`. Jusqu'à ~8 exécutions d'une
    requête qui scanne match_participants + medals_earned par clic sur un match, chaque appel
    ré-acquérant le SharedReader. Reco : GetHistoryForAvgBulk (IN + ROW_NUMBER PARTITION BY xuid),
    un seul Get du reader.
47. **Page escouade : LoadSquadMatches par coéquipier sélectionné** —
    `apps/go-api/internal/service/teammates_service.go:185`. 4 coéquipiers = 4 passes complètes
    (jointure double sur match_participants) sur une page déjà lourde. Reco :
    LoadSquadMatchesBulk groupé par teammate_xuid + lookup gamertags batch.

#### Modèle de connexions

48. **Player DB : handle RW unique à 1 connexion partagé entre toutes les lectures HTTP et les
    écritures sync/persist** — `apps/go-api/internal/platform/duckdb/db.go:227`. Toutes les
    lectures player d'un joueur (163 usages dans 60 fichiers) se sérialisent sur UNE connexion et
    queuent derrière chaque batch d'INSERT pendant un sync. Vérifié : maillon absent de
    .ai/V7/PLAN_CONTENTION_SYNC_SERVICE.md (qui ne traite que le provider shared). Reco : exporter
    d'abord sql.DBStats (WaitCount/WaitDuration) par handle, puis petit pool lecture (2-4 conns)
    après audit des UPSERT player reposant sur l'effet de bord MaxOpenConns(1), ou généraliser le
    modèle sharedprovider aux player DBs.
49. **Aucune configuration d'instance DuckDB (memory_limit, threads) pour un process
    multi-instances** — `apps/go-api/internal/platform/duckdb/db.go:507`. Chaque DSN distinct =
    une instance avec les défauts (80 % de la RAM et tous les cœurs PAR instance) x 8-15 instances
    -> surengagement mémoire/CPU sur le VPS, facteur aggravant plausible de la lenteur générale.
    Reco : budget par classe de DB dans openSQLDBFor (params DSN), valeurs exposées dans /health.
50. [TRACKÉ] **Pipeline V1 (fallback automatique encore actif) : writer RW shared tenu pendant
    toute la boucle de fetch réseau** — `apps/go-api/internal/sync/engine.go:266`. V2-par-défaut +
    budget lecture livrés, mais le fallback V1 automatique ramène la contention précisément quand
    le système est déjà dégradé. Reco : suivre le plan tracké (Chantier 1 + §8) — resserrer la
    fenêtre RW du chemin V1 ou supprimer le fallback après soak.

---

## Findings Mineurs (118, regroupés)

### Couches — handlers/api

- `api/handlers/bootstrap.go:23`, `api/handlers/title_sync.go:30` — dépendances concrètes
  *service.X au lieu de port.* (port.BootstrapService existe ; port.ProfileService incomplet).
- `api/handlers/progression.go:162`, `api/handlers/campaign.go:221`,
  `api/handlers/admin_auto_sync.go:206`, `api/handlers/sync_handler.go:165` — handlers qui
  composent des repos/services platform (+ merge catalogue x earned dans le handler progression).
- `api/server.go:146` — SQL boot-time dans api/ (loadTitleAssetDrawerData + loadCSRBadgeResolver).
- `api/registry_weapon_coverage.go:102` — slug ?title= concaténé dans le SQL (non exploitable en
  l'état) + SQL métier dans api/.
- `api/post_sync_deltas_snapshot.go:236` — « BestKDA » en quotient (kills+assists)/deaths, non
  conforme ADR 0006.
- `api/commendation_handler.go:4` — handler HTTP à la racine api/ au lieu de api/handlers/.

### Couches — service/analysis/domain

- Algos purs dans service/ : `service/engagement_timeseries_binning.go:65` (172 L stateless),
  `service/timeseries_service_aggregations.go:58`,
  `service/teammates_squad_charts_intensity_perminute.go:141`,
  `service/match_history_placement.go:35` (regex dupliquée depuis platform, cycle assumé).
- `service/friends_orchestrator_service.go:102` — invoque sync.RecomputeIsWithFriends (moteur
  d'écriture) au lieu d'un port.
- `service/catalog_fetcher_service.go:197` — référentiel halo_infinite/rankedplaylists sans gate
  titre dans le drain multi-titre.
- `service/engagement_admin_service.go:28` — liste de modes dupliquée service/sync.
- Impuretés analysis/ : `analysis/combat_yield.go:28` (état global atomique),
  `analysis/comeback.go:130` (slog dans fonctions pures), `analysis/sql_fragments.go:26`,
  `analysis/perfect_kills.go:28` (registre par slug + fragment SQL IN),
  `analysis/identity.go:162` (générateur CREATE OR REPLACE VIEW), `analysis/citations.go:174`
  (URLs d'assets), `analysis/world_stats.go:153` (parsing conventions Waypoint = TitleDataAdapter
  de fait), `analysis/mode_label.go:49` (playlists HINF dupliquées vs games/),
  `analysis/home_kpis.go:12` (dépendance legacymatch transitionnelle).
- Domain : `domain/achievement_categories.go:27` (référentiels par titre en dur),
  `domain/job.go:92` (littéral "halo_infinite" au lieu de title.DefaultSlug).

### Couches — SQL hors platform

- `sync/writes.go:68` — chemin d'écriture legacy ON CONFLICT conservé (documenté, cf. Majeur 15).
- `notify/notifiers.go:100` — NotifyNewMedia : fonction MORTE (0 caller) avec bare sql.Open sur
  shared_social + UPDATE media_files hors SharedSocialPersister — à supprimer.
- `progression/profile/queries.go:40` — SQL de lecture sur chemin HTTP hors platform/duckdb +
  filtre temporel non canonique (start_time brut).
- `worldenrich/wiring.go:33`, `platform/auth/pool/discovery.go:229` — bare connect RO sur une
  player DB potentiellement tenue RW (échec silencieux / legacy tokens).
- `games/halo_5/livesync/csr_match.go:69` — écritures per-match H5 directes (match_skill_rank,
  career_progression) hors BatchBuilder/Persister.
- `sync/no_art_patterns_test.go:184` — exclusion ops/ du guard-rail fondée sur « exécuté hors
  serveur », hypothèse fausse (plomberie média ops tourne in-process via services HTTP).
- `sync/schema.go:22` — DDL bootstrap maintenue dans sync/ en parallèle de migration.

### Structure

- `notify/` vs `notifications/` — doublon nomenclatural, deux domaines distincts.
- `prestige/` (41 fichiers, 9 366 L) et `campaign/` au top-level hors progression/.
- `worldenrich/wiring.go:7` — package de câblage top-level qui ouvre DuckDB en direct.
- `assets/` / `assetnames/` / `media/` — frontières réelles mais non documentées globalement.
- `metadata/` (2 fichiers) — nom trop générique pour des garde-fous d'import.
- `openspartan/` — lecteur de source externe, plutôt sous platform/.
- [TRACKÉ] `legacymatch/types.go:17` — types transitionnels dupliquant canonical (échéance doc).
- [TRACKÉ] `migration/steps_metadata.go` — 82 steps_*.go Halo-specific (ADR 0025 Phase 1.5).
- `domain/title/registry.go:412` — PathResolver (filesystem) hébergé dans domain/ (débattable).
- God-files secondaires : `games/halo_infinite/migrations/steps.go:27` (~1 270 L déclaratives),
  `api/handlers/prestige.go:77` (1 019 L, 28 routes de 4 sous-domaines),
  `sync/skill_v2_shadow.go:234` (774 L, SQL + TrueSkill2 + persist),
  `platform/duckdb/persist_sink.go:35` (745 L, 5 familles), `games/halo_infinite/adapter_data.go:76`
  (746 L), `platform/duckdb/db.go:31` (661 L), `api/registry_pages.go` (850 L, re-approche le
  seuil), `platform/auth/pool/pool.go:580` (3 fonctions > 80 L sans exemption),
  `prestige/service.go:276` (CreateChallenge 95 L),
  `games/halo_infinite/migrations/steps_player_base.go:21` (~670 L déclaratives).

### Title-agnosticism

- `platform/duckdb/home_repo_skill_peak.go:516` — fallback badge CSR HINF cross-titre.
- `platform/duckdb/match_view_repo_neighbors_skill.go:63` — préfixes de catégorie HINF en dur
  pour tout titre (neighbors Match View).
- `platform/duckdb/halo_ranks_loader.go:132` — LoadRankCatalog force title.DefaultSlug.
- `api/server.go:780` — callbacks image map/arme de l'Asset Drawer ignorent le titre.
- `service/world_stats_enricher.go:15` — leaderboard mondial câblé HINF sans gate capability.
- `api/handlers/sync_handler.go:29` — couche handlers couplée à games/halo_5/livesync.
- `api/registry_catalog_adapter_check.go:48` — HasCatalogAdapter vrai uniquement pour HINF.
- `api/registry_career.go:220` — commentaire obsolète (hard-gate supprimé).
- [TRACKÉ] `platform/auth/pool/types.go:23` — pool auth clé par gamertag seul (Phase 1.6).
- [TRACKÉ] `migration/order.go:162` — steps nommés Halo dans le runner neutre (Phase 1.5).
- `config/titles/halo_5/mappings/outcomes.toml:9` — raw_code absent : agrégations win/loss H5 en
  fallback legacy au lieu du chemin nominal MT-06.
- `config/titles/halo_5/mappings/assets.toml:13` — IDs de mode en slugs divergents de la
  convention (entrées probablement mortes).
- `api/server.go:1344` — stash OpenSpartan sur layout legacy data/players (3 copies du chemin).
- `api/server.go:290` / :651 — data/cache à la main ; CacheRootDir() absent du PathResolver.
- `ops/seed_demo.go:392` — data/media en dur au lieu de PathResolver.MediaDataDir() (existante).
- `scheduler/data_health_check.go:257` — pattern filepath.Join(TitleDataDir, "players") copié sur
  6 sites ; helper PlayersRootDir(slug) manquant.
- `config/config.go:205` — double mécanisme de résolution data/auth et data/sessions (env vs
  PathResolver).
- [TRACKÉ] `service/match_view_service.go:54` — couleurs hex d'outcome en dur (plan cleanup MV3).
- `config/titles/halo_5/mappings/capabilities.toml:10` — header périmé (« PHASE 1a : seul
  career.progression câblé ») contredit le corps (8 supported).
- `config/titles/synthetic_title_b/mappings/capabilities.toml:17` — fixture de test dans l'arbre
  config runtime, TOML contredit son adapter Go, sans test de parité.
- `games/capabilities.go:13` — clé fine absente = not_exposed silencieux ; pas d'exigence de
  complétude ; clés sans consommateur de gating.
- `api/server_titles_additional.go:110` — erreur de conversion capabilities.toml en WARN +
  fallback silencieux vers la map codée en dur.
- `domain/title/registry.go:76` — 3e mécanisme [damage_model] en miroir manuel des coarse caps
  (règle SSI non gardée).
- `domain/title/registry.go:249` — ServiceConfigIDFor : fonction morte (ignore son paramètre,
  retourne "").

### Performance

- N+1 cron/sync batchables (lecture seule) : `sync/engine.go:696` (SELECT EXISTS par match inséré,
  reconcile post-drain), `sync/skill_v2_helpers.go:28` (LoadState par joueur, 8/match),
  `service/relations_moments_service.go:140` (GetRivalTimeline, N=3),
  `service/fanout_service.go:73` (CountCommonMatchesForXUID par joueur),
  `sync/session_recalc.go:80` (lookup xuid_aliases par gamertag),
  `sync/backfill_registry_names.go:157` (lookup + UPDATE par asset_id),
  `api/handlers/prestige.go:718` (ListSquadMembers + SquadUsualContexts par squad),
  `api/registry_catalog_expand.go:94` (3 requêtes par entry, write cron).
- Ré-implémentations locales du « latest » : `platform/duckdb/queries_home_citations.go:448`
  (Q26g sans tiebreak de récence), `platform/duckdb/player_matches_loaders.go:190` (match_csrs),
  `platform/duckdb/queries_squad.go:10` (MAX(expected_win_prob) brut),
  `platform/duckdb/halo5_career_source.go:34` (meilleur CSR à vie sur snapshots bruts),
  `platform/duckdb/csr_coverage_repo.go:63` (COUNT(*) brut gonfle le diag).
- `platform/duckdb/queries_home_citations.go:26` — CTE perfect de Q26 agrège medals_earned sur
  tout l'historique pour une liste LIMIT 150.
- `sync/aggregates.go:37` — mv_map_stats : rebuild à chaque sync, AUCUN lecteur Go, nom trompeur
  (agrégat sur table brute).
- Constantes SQL mortes au pattern interdit (préfixe shared. / lecture brute) :
  `platform/duckdb/queries_career.go:7` (Q4/Q4MV), `queries_home_citations.go:283` (Q26e),
  `queries_career_encounters.go:446` (Q24 + doc fausse).
- `platform/duckdb/db.go:192` — limites de pool 4/2 en constantes magiques, sans observabilité
  d'attente.
- `api/registry_relations_cross_game.go:81` — OpenReadForQuery cross-titre : emprunt non-possédant
  d'un handle que le provider de l'autre titre peut fermer en plein vol.

### Contrat / gouvernance

- `api/middleware/contract_validate.go:34` — ContractValidate ne valide aucun schéma et bufferise
  TOUS les corps, y compris hors /api/v1/.
- `api/middleware/read_budget.go:7` — middleware couplé à platform/duckdb/sharedprovider (clé de
  contexte du swap-budget).
- `service/media_service.go:338` — ReassociateMedia : dead code (route supprimée 2026-04-29,
  méthode + interface + types conservés).
- `api/server.go:773` — câblage Halo 5 copy-paste au boot (double lecture metadata + fallbacks
  d'image épinglés HINF).

---

## TOP 10 des actions prioritaires

1. **Basculer tous les lecteurs de rating sur les vues _latest** (Majeurs 39-43 + snapshot
   skill-tier + 5 sites mineurs) — quick win à plus fort impact utilisateur : ratings non
   déterministes visibles sur Home, historique, compare, leaderboard, patterns et notifications,
   aggravé pour H5.
2. **Fermer les 2 derniers trous ART** : router l'import OpenSpartan vers persist.SharedPersister
   (n° 15) et convertir/couvrir le bulk UPDATE de backfill_registry_names + étendre le tripwire
   (n° 16).
3. **Réparer et étendre archlint** : regex no_slug_comparison (n° 30), + 3 nouvelles règles — pas
   de SQL/Open* dans api/, pas de filepath.Join(..."data"...) hors resolver, parité coarse<->fine
   des capabilities. Meilleur levier du repo : les ratchets existants ont prouvé qu'ils tiennent.
4. **Débrancher les 3 fuites HINF actives sur les surfaces H5** : classification MediaRepo
   (n° 28), providers CSR Explorer/Compare (n° 29), WaypointURL (n° 32) — bugs visibles en prod
   multi-titre aujourd'hui.
5. **Extraire le pipeline post-sync de api/** vers service/postsync + repos platform (n° 1-2), et
   la cascade auth vers platform/auth (n° 5).
6. **Compléter les manifests halo_5** (fields.toml 5 -> ~60, raw_code outcomes) + réconcilier
   coarse/fine engagement + câbler LoadAuthDescriptor (n° 34, 36, 37).
7. **Instrumenter puis desserrer le goulot player-DB** : sql.DBStats par handle d'abord, puis pool
   de lecture 2-4 conns, et poser memory_limit/threads par classe d'instance DuckDB (n° 48-49) —
   impact direct sur la lenteur prod.
8. **Batcher les 2 N+1 HTTP chauds** : Q29 bulk sur le match view, LoadSquadMatchesBulk sur la
   page escouade (n° 46-47).
9. **Assainir le contrat HTTP** : supprimer internal/api/gen mort, résorber les 22 schémas
   DIVERGENT puis durcir le drift-test (n° 26-27).
10. **Entamer le découpage des god packages par le plus rentable** : racine api/ (wire/ +
    postsync), puis service/ par feature avec test d'imports croisés, ratchet de gel sur la racine
    sync/ (n° 17-20, 22).

## Recommandations techniques

- **Le ratchet archlint est le meilleur outil de gouvernance du repo — investir dedans.** Partout
  où un test-garde existe (no_duckdb_import, no_art_patterns, no_slug_comparison, drift OpenAPI),
  la dette est contenue ou décroissante ; partout où il n'existe pas (SQL dans api/, chemins
  data/, parité TOML), elle a proliféré. Chaque règle d'architecture de ce rapport devrait finir
  soit corrigée, soit encodée en ratchet avec allowlist décroissante datée.
- **Une seule source de vérité par référentiel.** Trois mécanismes de capabilities coexistent
  (coarse Go/TOML, fine TOML, flags [damage_model]) sans garde de cohérence, et le titre par
  défaut vit hors TOML avec skip silencieux. À trancher : soit le TOML fait foi pour tous les
  titres (parity-test pour le built-in), soit l'asymétrie est verrouillée par un WARN au boot et
  documentée dans l'ADR 0025.
- **Le pattern « interface consumer-side étroite » (explorer_service) mérite d'être le standard**
  pour tout besoin cross-service — il résout à la fois le couplage HomeService->CareerLiveService
  et la contamination des contrats par les types internal/sync.
- **La contention prod a probablement deux étages, pas un.** Le plan CONTENTION_SYNC_SERVICE
  traite le provider shared ; le handle player RW 1-conn et l'absence totale de tuning d'instances
  DuckDB (mémoire/threads par défaut x 8-15 instances sur le VPS) sont le second étage, non
  tracké. Mesurer (DBStats) avant de toucher, mais traiter les deux.
- **Purge de dead code ciblée** (règle projet « 0 module mort ») : api/gen, NotifyNewMedia,
  ReassociateMedia, ServiceConfigIDFor, constantes SQL mortes Q4/Q26e/Q24, entrées assets.toml H5
  mortes — une PR de suppression pure, sans risque.
- **Priorisation multi-titre** : les items qui bloquent concrètement Halo 5 ne sont pas les god
  packages mais les fuites sémantiques (n° 28-38). À périmètre d'effort égal, les traiter avant
  les refactors structurels — petits, localisés, user-visible.
