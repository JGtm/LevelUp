# Plan — Refactoring vers une architecture title-agnostic complète (v2.5)

**Objectif** : retirer/ajouter un champ d'un titre = **1 ligne dans `fields.toml` (mapping) + 1 modif dans le `TitleDataAdapter` correspondant**, sans toucher aux services, repos cross-titre, OpenAPI, ni frontend.

**Critère de succès opérationnel** : une CI step « build + tests pour `synthetic_test_title` » s'exécute avec un dossier `internal/games/synthetic_test_title/` minimal + un set de TOML mappings réduit. Toutes les routes répondent (200 ou 404 capability) et tous les tests existants passent. Aucun changement requis dans `internal/service/`, `internal/api/`, `apps/web/` au-delà d'une route capability-gated.

**Branche cible** : `refactor/title-agnostic-services` (créée depuis `main`).

**Statut** : v2.5 (2026-05-18) — Revue après ~10 jours de développement intensif (OpenSpartan, Xbox SSO, perf/LUSR chains, career-live, etc.). Décisions D11 (OpenSpartan = bootstrap one-shot), D12 (migration CapabilityMap code → TOML), D13 (gel nouveaux handlers en Huma dès Phase 3b start) ajoutées. Phase 1.7 découpée en **1.7a (capabilities binaires TOML, light)** + **1.7b (3 états + cascade, lourd)**. Phase 1.5 ré-estimée (~doubling) à cause de l'accumulation `internal/migration/steps_*.go` Halo-only ajoutés depuis. Phase 3b ré-estimée (~80 → ~113 handlers). Phase 2 enrichie d'un test de continuité OpenSpartan-primed → Waypoint-fed. Effort total : 62-82 j (vs 50-67 j en v2.4).

---

## 0bis. Périmètre ÉLARGI — registre multi-titre complet (audit 2026-06-14)

> Ce master couvre le **chemin data-lecture du match**. L'audit 2026-06-14 a révélé ~26 axes multi-titre **supplémentaires** (ingestion, acquisition auth, scheduler, settings, achievements, world-stats, outcome, observabilité, Discord, cycle de vie, registre de migrations…) — voir l'**index** [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md) (registre `MT-01..MT-26`) et les specs détaillées [PLAN_MULTITITRE_PERIPHERY.md](PLAN_MULTITITRE_PERIPHERY.md) (`PMT-1..14` + extensions `EXT-1.5/2/5`). **PMT-14** = section admin « Titres » + réhabilitation du **Lab cassé** (non monté dans `server.go`).
>
> **⚠ Doctrine RE-VÉRIFIER (s'applique aussi aux phases de CE master)** : les pointeurs `file:line` sont une carte datée, pas une vérité figée. Re-grep/re-valider chaque évidence contre `HEAD` AVANT d'exécuter une phase (existante ou nouvelle). Une spec est une hypothèse à reconfirmer, jamais un copier-coller.
>
> **Méthode** : `expand → parity-gate → contract` avec **oracle double** (parité Halo golden + fixture `synthetic_test_title`). **Bloquants 2ᵉ titre hors data-path** : `PMT-1` (hosts ingestion) + `PMT-2` (acquisition auth) + `PMT-3` (écriture sync par titre) — racine du DAG.

## 0. Doctrine — alignement avec l'ADR 0011 + acquis ADR 0012

L'ADR 0011 a tranché : **`canonical.*` reste minimal**, et trois adapters distincts collaborent côté service :

| Adapter | Rôle | Ce qu'il NE FAIT PAS |
|---|---|---|
| `TitleDataAdapter` | Charge la data brute → `canonical.*` | Pas de label i18n, pas d'URL |
| `TitleSemanticAdapter` | Labels FR/EN, RankCatalog, Outcomes | Pas de data brute |
| `TitleAssetURLAdapter` | URLs map / medal / CSR rank | Pas de DB |

Ce plan **respecte cette frontière** : il ne pousse PAS `canonical.PlayerMatchRow` directement dans le DTO HTTP. Le DTO HTTP reste un **view-model service** qui combine les 3 sources.

**Acquis ADR 0012 (2026-04-29)** : la logique Halo-only de `internal/analysis/` (préfixes mode_category, citations custom) a déjà été extraite vers `internal/games/halo_infinite/`. Le hook `analysis.RegisterCustomDispatcher` est en place. Ce plan **construit dessus** — il ne reprend pas ce travail.

**Acquis stub multi-titre** : [synthetic_title_b/adapter.go](apps/go-api/internal/games/synthetic_title_b/adapter.go) implémente déjà les 3 adapters + un `CapabilityMap` (binaire) + un test d'isolation. Ce stub servira de **base au futur `synthetic_test_title`** introduit par la validation finale §8.11.

Ce que le plan vise restent :

1. Que les **column names DB** ne fuitent plus dans les services (Phase 2).
2. Que les **types Halo-specific** dans `domain/match_view.go` (`MatchExpectedStats`, `ExpectedStatsRaw`, etc.) soient remplacés par des types canonical reasonably nullable (Phase 3a).
3. Que les **flags sync** ne soient plus énumérés en dur par champ Halo (Phase 4).
4. Que le **schéma DB** soit isolé par titre, pas en silo dans `internal/migration/steps_*.go` partagé (Phase 1.5).
5. Que le **handler layer** ne maintienne plus de YAML OpenAPI manuel (Phase 3b, Huma).
6. Que les **capabilities** soient déclarées en TOML (déjà partiellement en code, à migrer — Phase 1.7a/b).

Différence clé avec v1 : le plan v2 **conserve les types domain de view-model** (header, summary tab, scoreboard row…) — ils sont la composition canonique × semantic × assetURL. Ce qu'on retire de `domain/`, c'est uniquement les types `*Raw` et les champs Halo-only sans pendant canonical.

---

## 1. Diagnostic — fuites actuelles de l'abstraction

Sur la modif récente `drop assists_expected/assists_stddev` (Halo Infinite ne renvoie pas ces champs), j'ai dû toucher 14 fichiers dans 8 couches. Diagnostic toujours valable en mai 2026.

| Couche | Fichier | Type de fuite |
|---|---|---|
| **DDL shared** | `internal/migration/steps_*.go` (20+ fichiers, dont `steps_player_lusr_chain_rework.go`, `steps_player_perf_chain.go`, `steps_player_prestige.go`, `steps_player_notifications.go`, `steps_metadata_prestige_seed.go` ajoutés depuis la v2.4) | `CREATE/ALTER COLUMN` Halo-specific dans la DB partagée multi-titres. La dette s'aggrave à chaque feature Halo-specific livrée. |
| **SQL inline** | `platform/duckdb/queries_match.go` (Q12, Q26, Q26MatchExpectedStats) | `SELECT mp.assists_expected, ...` codé en dur |
| **Magic constants Halo** | `queries_match.go` Q12 (`medal_name_id = 1512363953` pour Perfect Kills) | Constante Halo-only inline dans une query cross-titre |
| **Scan** | `platform/duckdb/match_view_repo.go` | `row.Scan(&s.AssistsExpected)` couplé à l'ordre du SELECT |
| **Domain (view-model)** | `domain/match_view.go` (`MatchExpectedStats`, `MatchScoreboardRow.ExpectedKills`) | Champs Halo-specific exposés au DTO HTTP |
| **Domain (raw)** | `domain/match_view.go` (9 types `*Raw` : `MatchMetaRaw`, `PlayerMatchStatsRaw`, `ScoreboardRaw`, `BulkMedalRaw`, `BulkWeaponKillRaw`, `MatchEnrichmentRaw`, `MedalRaw`, `EventRaw`, `WeaponKillRaw`, +`MatchHistAvgRow`) | Types frontière repo↔service mais hébergés dans `domain/`. Fichier toujours à 799 lignes. |
| **OpenAPI** | `api/openapi.yaml` | Schema `MatchExpectedStats.expected_assists` édité manuellement (~113 routes maintenues à la main — +33 depuis la v2.4 à cause de OpenSpartan, Xbox SSO, achievements, CSR per-match, etc.) |
| **Generated** | `internal/api/gen/types.gen.go` | Auto-généré depuis OpenAPI manuel — divergence inévitable |
| **Handlers chi** | `internal/api/handlers/*.go` (**113 fichiers** vs 80 en v2.4) | Style `func(w, r)` avec validation manuelle (regex, parse query) — pas d'inférence OpenAPI |
| **Service** | `service/match_view_service.go`, 77 occurrences de `"halo_infinite"` dans `internal/service/` | `out.ExpectedAssists = e.AssistsExpected` ; comparaisons slug hardcodées |
| **Sync flags** | `sync/scope.go`, `sync/backfill_flags.go`, `sync/backfill_cli.go` | `PBitAssistsExp`, `--assists-expected`, `scope.AssistsExpected` |
| **Tests** | 4+ fichiers | refs à `PBitAssistsExp` / `scope.AssistsExpected` / `assists_expected` |

**Causes racines** : identiques à la v2.4 ; aucune n'a été résolue côté code applicatif. Le compteur de fichiers/lignes touchés pour ajouter un field a même empiré (steps_*.go a doublé en taille).

---

## 2. Architecture cible

### 2.1 Topologie (alignée ADR 0011)

```
┌──────────────────────────────────────────────────────────────────┐
│ apps/web/                                                        │
│  - lit JSON DTO via TanStack Query                               │
│  - useFieldLabel(FieldKey, locale) / useCapability(cap)          │
│  - omet/grise les sections quand field nil ou capability absente │
└─────────────────────────────────┬────────────────────────────────┘
                                  │ JSON DTO (view-model)
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│ internal/api/handlers/ (Huma)                                    │
│  - reçoit Input typé via path:/query:/body: tags                 │
│  - appelle Service via port.*Service                             │
│  - sérialise le DTO retourné                                     │
│  - 0 logique métier, 0 SQL                                       │
└─────────────────────────────────┬────────────────────────────────┘
                                  │ domain.*ViewResponse
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│ internal/service/                                                │
│  compose 3 sources :                                             │
│   - port.*Repository    → canonical.* (data brute)               │
│   - games.TitleSemantic → labels FR/EN, RankCatalog, Outcomes    │
│   - games.TitleAssetURL → URLs map / medal / CSR rank            │
│  retourne domain.*ViewResponse (view-model composé)              │
│  dégrade gracieusement si capability absente                     │
└──────┬─────────────────────────┬──────────────────────────────┬──┘
       ▼                         ▼                              ▼
┌──────────────┐        ┌────────────────┐         ┌────────────────────┐
│ analysis/    │        │ port/          │         │ internal/games/    │
│  algos purs  │        │  *Repository   │         │  halo_infinite/    │
│  0 IO        │        │  *Service      │         │  adapter_data.go   │
│  prend       │        │  interfaces    │         │  adapter_semantic  │
│  canonical   │        └────────┬───────┘         │  adapter_asset_url │
└──────────────┘                 │ impl            │  ddl/*.sql          │
                                 ▼                 │  capabilities.toml │
                ┌────────────────────────────────┐ └────────────────────┘
                │ internal/platform/duckdb/      │
                │  - implémente port.*Repository │
                │  - construit SQL via FieldKey  │
                │  - DDL chargée via TitleDataAdapter.MigrationSteps()
                │  - 0 SQL exposé en dehors      │
                └────────────────────────────────┘
```

### 2.2 Règle d'or

> **Aucune fonction dans `service/`, `api/handlers/` ou `apps/web/` ne doit jamais voir un nom de colonne DB, un type Halo-specific, ou un enum Halo (medal_id, mode_prefix). Tout passe par `canonical.*` + `FieldKey` + `Capability` ou par un des 3 adapters.**

### 2.3 Comportement nullable cohérent

- Tous les fields canonical optionnels sont **`*T` pointer**.
- Le frontend gère `null` partout (`omitempty` côté Go, `T | null` côté TS, fallback texte ou skeleton dans la UI).
- Une capability absente → field jamais renvoyé (`omitempty`) → frontend ne l'affiche pas.
- Distinction explicite côté repo : « field absent du mapping TOML » vs « valeur NULL en DB » (cf. décision D2 ci-dessous).

---

## 3. Phase 0 — Décisions techniques + alignement ADR (BLOQUANTES)

**Effort** : 2-3 jours
**Livrable** : 9 décisions tranchées + ADRs mises à jour + branche créée + plan v2.5 figé.

### Décisions tranchées

| ID | Sujet | Décision | Justification |
|---|---|---|---|
| **D1** | `canonical.Value` (Phase 2) | Wrapper typé `Value{Kind, Int, Float, Str, Bool, Time}` | Compromis lisibilité/perf, pattern-match côté service via switch sur Kind, évite le runtime cast |
| **D2** | Field absent vs NULL | `map[FieldKey]*Value` : présent dans le map = field supporté ; `*Value = nil` = NULL en DB ; absent du map = capability non supportée | Sémantique explicite, testable, permet de vérifier la dégradation gracieuse |
| **D3** | Schéma DB multi-titre (Phase 1.5) | DB physique par titre (`data/titles/{slug}/warehouse/...`) | Cohérent ADR 0008 (isolation par chemin FS). Le `PathResolver` retourne déjà la bonne DB. DDL dans `internal/games/{slug}/ddl/` |
| **D4** | OpenAPI gen (Phase 3) | **Huma intégré au plan** : migrer ~113 handlers chi vers Huma (vs 80 en v2.4). OpenAPI 3.1 auto-généré par construction, validation des inputs auto, plus jamais de YAML manuel | Décision ambitieuse mais bénéfice scale linéairement avec #handlers — donc +33 handlers depuis v2.4 renforce le ROI, pas l'inverse. Coût total révisé à 62-82 j. |
| **D5** | Codegen TS canonical (Phase 7) | Script `tools/codegen/canonical-ts/` : lit `canonical/fields.go` (go/ast) → écrit `apps/web/src/lib/canonical/fields.ts`. Single source Go, CI lint vérifie l'idempotence | Évite la dérive entre Go et TS. Compatible avec l'output OpenAPI Huma. |
| **D6** | Stratégie de migration progressive (Phase 2) | Service par service en PR atomique : ancien path supprimé dans la même PR que la migration | Pas de feature flag (évite la dette « deux paths à maintenir »). Critère de mergeabilité par phase = service migré + tests passent |
| **D7** | Granularité du status feature (Phase 1.7b) | 3 états : `available` / `degraded` / `unavailable` + reason humaine. Une feature `degraded` rend des données partielles avec badge UI explicite | Permet de modéliser le cas réel : feature qui tourne avec subset (ex. `synthesis.weapon_breakdown` sans medals = OK mais moins riche). Plus nuancé qu'un binaire on/off |
| **D8** | Source de la disponibilité data (Phase 1.7) | TOML déclaratif uniquement : l'opérateur du titre déclare `[data] match_events = true` dans `capabilities.toml`. Test CI de cohérence : pour chaque `data=true`, la table existe + a ≥ 1 row sur fixture `halo_full` | Simple, rapide, pas de query DB au boot. Le risque de dérive (TOML dit true mais table vide) est mitigé par le test de cohérence |
| **D9** | Phasage de l'outillage diagnostic | Phase 1.8 dédiée (3-4 j) après Phase 1.7b. Outillage opérateur, différable | Sépare clairement le runtime (utilisé par les services produit) du tooling opérateur (utilisé une fois par titre ajouté). |
| **D10** | Mode d'export TOML draft depuis Lab | Copie presse-papier uniquement. Le bouton génère le bloc `[data]` formaté, l'opérateur colle manuellement dans `capabilities.toml` et fait `git commit` | Préserve le versioning Git (D8). Pas d'écriture serveur, pas de risque de conflits, audit trail = git log standard. L'opérateur reste le seul auteur du TOML |
| **D11** | OpenSpartan = bootstrap one-shot (NOUVELLE v2.5) | OpenSpartan import est une **source d'ingestion bootstrap unique** (Day-0, premier lancement), pas un pipeline continu. Le `TitleDataAdapter` est l'autorité du schéma DB ; OpenSpartan **écrit** vers la même DDL que Waypoint. Le `port.MatchFieldRepository` lit indifféremment Day-0 (OpenSpartan-primed) et Day-1+ (Waypoint-fed) | Évite de modéliser un 2ᵉ pipeline parallèle dans `SyncScope` ou Phase 4. Le seul test de parité requis (Phase 2) est : « une DB primée OpenSpartan + sync'd Waypoint sur les mêmes match_id retourne les mêmes valeurs canonical » |
| **D12** | Migration `CapabilityMap` code → TOML (NOUVELLE v2.5) | Le `CapabilityMap` actuel ([games/adapter.go:36-54](apps/go-api/internal/games/adapter.go#L36)) est **binaire** et déclaré en Go. Phase 1.7a le **réécrit en TOML** sans casser l'API existante (les `games.Cap*` constants restent les keys, valeurs lues depuis `capabilities.toml` au boot). Phase 1.7b **étend** vers 3 états + cascade. | Évite un double système (code + TOML) en parallèle. Migration en 2 PR : (a) move CapabilityMap → TOML, (b) extend to 3-state. Compat préservée pour les consumers actuels. |
| **D13** | Gel des nouveaux handlers en Huma (NOUVELLE v2.5) | Dès que **Phase 3b est démarrée**, tout nouveau handler créé (feature non listée dans le plan) doit être **directement écrit en Huma** — pas en chi. Lint CI bloquant : `tests/lint/no_new_chi_handler_test.go` qui git-diff vs branche `phase-3b-start-tag` et rejette tout nouveau `func(w, r)`. | Sans D13, la dette se creuse pendant la migration (rythme actuel : ~30 handlers en 2 semaines). Avec D13, le solde restant est plafonné. |

### Tâches Phase 0

- [ ] **ADR 0014 (à créer)** : « Title-agnostic services + DDL isolation ». Acte D2, D3, D6, D11 (OpenSpartan bootstrap). Complète ADR 0011 sans la contredire.
- [ ] **ADR 0015 (à créer)** : « Adoption de Huma pour OpenAPI 3.1 auto-généré ». Documente D4 + D13 (gel des nouveaux handlers). Bénéfices long terme (validation auto, plus de YAML manuel) vs coût (~113 handlers). Alternatives écartées : swag, kin-openapi, Fuego.
- [ ] **ADR 0016 (à créer)** : « Feature Matrix — capability cascading multi-titre ». Documente le modèle data capabilities + feature capabilities + arbre de dépendances + 3 états (available / degraded / unavailable). Décisions D7, D8, D12. Mention migration depuis CapabilityMap code.
- [ ] **ADR 0017 (à créer)** : « Title Diagnostic — outillage Lab read-only ». Documente la séparation diagnostic (D9) + le choix copie presse-papier vs écriture serveur (D10). Pattern : TOML reste source de vérité Git, le Lab n'écrit jamais.
- [ ] Créer la branche `refactor/title-agnostic-services` depuis `main`.
- [ ] Ajouter ce plan en référence dans `CLAUDE.md` § Décisions architecturales.
- [ ] Entrée `thought_log.md` documentant les 13 décisions et leur justification.

---

## 4. Phases d'exécution

### Phase 1 — Étendre `canonical/fields.go` à 100% des champs services (rapide)

**Effort** : 2-3 jours
**Risque** : faible (additif uniquement)
**Livrable** : tous les FieldKeys que les services lisent existent dans `canonical/fields.go`, mappés vers les colonnes DB Halo Infinite via TOML, pour **toutes les tables shared**, pas seulement `match_participants`.

**État actuel** : [canonical/fields.go](apps/go-api/internal/games/canonical/fields.go) ne fait que 158 lignes — additions raisonnables. Aucun TOML mapping `fields.toml` n'existe encore côté `internal/games/halo_infinite/`.

- [ ] Inventaire exhaustif :
  ```bash
  rg "p\.\w+|mp\.\w+|mr\.\w+|me\.\w+|w\.\w+|kvp\.\w+" \
     apps/go-api/internal/platform/duckdb/queries_*.go > /tmp/cols.txt
  ```
- [ ] Pour chaque colonne référencée, vérifier que le `canonical.FieldKey` existe ; sinon, ajouter dans `canonical/fields.go`.
- [ ] Créer `config/titles/halo_infinite/mappings/fields.toml` (n'existe pas) avec le mapping `field_key → db_column → table` pour les 5 tables :
  - `match_participants` (~31 colonnes)
  - `medals_earned` (medal_id, count)
  - `highlight_events` (event_type, time_ms, actor_xuid)
  - `killer_victim_pairs` (killer_xuid, victim_xuid, kill_count)
  - `weapon_kills` (weapon_id, kills)
- [ ] Pour les **constantes Halo magiques** (`medal_name_id = 1512363953` pour Perfect Kills, mode prefixes Assassin/Fiesta/BTB…) : créer `config/titles/halo_infinite/constants.toml` (`perfect_kill_medal_id`, `mode_prefixes`) lu au boot par l'adapter Halo.
- [ ] Test garde-fou : `canonical/fields_test.go` vérifie l'exhaustivité des FieldKeys référencés par les TOMLs (lint custom déjà existant `Lint multi-titres`).

**Critère de complétion** : `grep -rE "p\.\w+|mp\.\w+|mr\.\w+" internal/platform/duckdb/queries_*.go` produit une liste 100% couverte par `fields.toml`. Aucun magic number Halo dans `queries_*.go`.

### Phase 1.5 — DDL et schéma multi-titre (lourd, effort doublé)

> **Extensions audit 2026-06-14 (à RE-VÉRIFIER)** — au-delà du déplacement DDL, cf. [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) : colonnes `title_id` metadata (MT-16, **décision** car `MetadataDBPath(slug)` isole déjà par chemin), audit ops/CLI élargi à `healthcheck`+`gate`+`cmd/*` (MT-10), seed démo (MT-18), scoping `notification_preferences` (MT-17). + le **registre/ordre de migrations** reste 1 liste globale Halo → [PMT-9](PLAN_MULTITITRE_PERIPHERY.md) (MT-23). Index : [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md).

**Effort** : **8-10 jours** (vs 4-5 j en v2.4)
**Risque** : élevé (touche **20+ fichiers** de migration accumulés)
**Livrable** : chaque titre possède ses propres DDL ; le `MigrationRunner` est paramétré par `titleSlug`.

**Pourquoi l'effort a doublé** : depuis la v2.4, le répertoire [internal/migration/](apps/go-api/internal/migration/) a accumulé des steps Halo-specific qui n'existaient pas. Liste exhaustive à migrer :

| Catégorie | Fichiers à migrer vers `internal/games/halo_infinite/ddl/` |
|---|---|
| **Métadata** | `steps_metadata.go`, `steps_metadata_assists_model.go`, `steps_metadata_catalog.go`, `steps_metadata_citation_fix.go`, `steps_metadata_playlist_fr.go`, `steps_metadata_prestige.go`, `steps_metadata_prestige_seed.go` |
| **Player DB (Halo-only)** | `steps_player.go`, `steps_player_assists_model.go`, `steps_player_lusr_chain_rework.go`, `steps_player_notifications.go`, `steps_player_perf_chain.go`, `steps_player_prestige.go` |
| **Engagement** | `steps_engagement.go` |
| **Shared (historique)** | Tout DDL Halo-only restant dans `migration/` |

- [ ] Déplacer ces fichiers vers `internal/games/halo_infinite/ddl/{steps_shared.sql, steps_player.sql, steps_metadata.sql, steps_pve.sql, steps_engagement.sql, steps_perf_chain.sql, steps_lusr_chain.sql, steps_prestige.sql, steps_notifications.sql, ...}`.
- [ ] `MigrationRunner` lit les steps depuis le `TitleDataAdapter` (méthode `MigrationSteps() []MigrationStep`). Plus de DDL hardcoded dans `internal/migration/`.
- [ ] Schema versioning par titre : `meta.schema_version` dans chaque DB, par titre. Conserver compat avec `schema_version` existant (migration step `0000_rename_schema_version_to_titled.sql`).
- [ ] **OpenSpartan bootstrap** (D11) : vérifier que [internal/service/openspartan_post_import_service.go](apps/go-api/internal/service/openspartan_post_import_service.go) et le mapper [openspartan_mapper.go](apps/go-api/internal/openspartan/) écrivent vers les **mêmes tables/colonnes** que les steps déplacés. Sinon, divergence Day-0 vs Day-1+.
- [ ] Test : ajouter un titre fixture `synthetic_test_title` (peut être un alias du `synthetic_title_b` existant étendu) avec un schéma minimal (3 colonnes : kills, deaths, match_id) et vérifier que le `MigrationRunner` crée la DB sans toucher au code partagé.
- [ ] Audit `internal/ops/` (backup, restore, diagnose) : ces outils doivent itérer sur tous les titres enregistrés. Effort spécifique 1-2 j inclus dans le total.

**Critère de complétion** : `internal/migration/` ne contient plus aucune DDL Halo-specific. Ajouter un titre = créer son dossier `ddl/` + l'enregistrer. Le test `OpenSpartanImport_E2E_test.go` continue de passer (rows OpenSpartan respectent la DDL HI).

### Phase 1.6 — Pool tokens multi-titre (NOUVELLE 2026-05-24)

**Effort** : ~1-2 j
**Contexte** : aujourd'hui un seul titre (Halo Infinite). `internal/platform/auth/pool/` gère un pool de Spartan tokens par joueur mais **sans dimension `titleSlug`** explicite. Quand un 2e titre arrivera (Reach via MCC, autre jeu Microsoft), il faudra que le pool puisse stocker des tokens spécifiques par couple `(titleSlug, gamertag)` — un même joueur peut avoir des credentials différents selon le jeu (différents Spartan endpoints, scopes OAuth potentiellement différents).

**Acquis** : `Pool` a déjà une méthode `SetCurrentTitle(slug)` qui prépare le terrain — il faut étendre pour vraiment supporter plusieurs titres simultanés (pas juste un "currentTitle" mono-valué).

**Lien avec sprint auth unification** : si le sprint Option E du `.ai/PLAN_AUTH_PROVIDER_UNIFICATION.md` est livré avant Phase 1.6 (très probable), le refactor unification a déjà identifié `TokenProvider` comme source de vérité + `Pool` comme cache. Étendre la clé du cache de `gamertag` vers `(titleSlug, gamertag)` est la touche naturelle à ce moment-là.

**Tâches** :
- [ ] `Pool` interne : `sync.Map[string]Slot` → `sync.Map[poolKey]Slot` avec `poolKey = struct{TitleSlug, Gamertag string}`. Méthodes `Get`, `HasPlayer` prennent (titleSlug, gamertag).
- [ ] `Discovery.Scan()` (ou son remplaçant post-unification) itère sur **tous les titres enregistrés** dans le `title.Registry`, pas seulement le slug courant.
- [ ] `PooledHaloClient` paramétré par titleSlug en plus de gamertag/xuid.
- [ ] Tests : 2 titres fixtures (halo_infinite + synthetic_title_b), même gamertag, vérifier que les 2 entries pool coexistent sans conflit.
- [ ] Garde compat : si appelé sans titleSlug (anciens chemins), défaut sur `title.DefaultSlug` jusqu'à migration complète.

**Critère de complétion** : ajouter un 2e titre n'oblige plus à recréer un pool ad-hoc — le pool existant accommode automatiquement le nouveau slug via Discovery.

### Phase 1.7a — TOML capabilities binaires (light, NOUVELLE v2.5)

**Effort** : 2 jours
**Risque** : faible (additif, compat avec `CapabilityMap` existant)
**Livrable** : le `CapabilityMap` actuel ([games/adapter.go:51](apps/go-api/internal/games/adapter.go#L51)) est lu depuis `capabilities.toml` au boot. Les `games.Cap*` constants restent les keys (pas de breaking). Endpoint `GET /api/v1/title/capabilities` retourne le map du titre actif.

- [ ] Créer `config/titles/halo_infinite/capabilities.toml` minimal (section `[capabilities]` binaire uniquement) :
  ```toml
  [capabilities]
  match_history       = true
  match_detail_core   = true
  match_skill_snapshot = true
  career_progression  = true
  pve_firefight       = true
  timeseries          = true
  engagement          = true
  ```
- [ ] Créer `config/titles/synthetic_title_b/capabilities.toml` (subset, prouve la dégradation).
- [ ] Loader : `internal/games/{slug}/capabilities.go::LoadCapabilities() games.CapabilityMap`. Les adapters `halo_infinite/adapter_data.go` et `synthetic_title_b/adapter.go` consomment ce loader au lieu de hardcoder la map.
- [ ] Handler `GET /api/v1/title/capabilities` (existe déjà ? Sinon créer). Format JSON `{ "capability_key": "supported" | "not_exposed" }`.
- [ ] **Aucune extension** (3 états, cascade, data vs feature) — c'est Phase 1.7b.
- [ ] Test : suppression d'une ligne du TOML synthetic → endpoint reflète l'absence + service dégrade.

**Critère de complétion** : 0 `games.CapabilityMap{...}` hardcodé dans `internal/games/*/adapter*.go`. Toutes les maps proviennent du TOML.

### Phase 1.7b — Feature Matrix 3 états + cascade (moyen, ex-Phase 1.7)

**Effort** : 4-5 jours
**Risque** : moyen (touche services + frontend, mais isolé via interface)
**Livrable** : système 2 niveaux qui modélise « data capability » (donnée brute peuplée) + « feature capability » (feature produit) avec arbre de dépendances déclaratif. Status à 3 états (`available` / `degraded` / `unavailable`). Exposé au frontend via `/api/v1/title/feature_matrix`.

**Pré-requis** : Phase 1.7a mergée (TOML capabilities binaires en place).

#### 1.7b.1 Modèle de données (extension du TOML 1.7a)

`config/titles/{slug}/capabilities.toml` étendu :

```toml
# Section [capabilities] : binaire, conservée pour compat (Phase 1.7a).
[capabilities]
match_history       = true
...

# Section [data] : disponibilité brute des sources DB par titre (NOUVELLE 1.7b).
[data]
match_events     = true   # highlight_events table peuplée
match_skill      = true
weapon_kills     = true
medals           = true
killer_victim    = true
csr              = true
lusr             = true
pve_firefight    = true
match_film       = false  # ex: pas de film côté Halo Infinite

# Section [feature.<key>] : features produit + leurs dépendances data.
# requires        : données obligatoires (absent → unavailable)
# degraded_without: données optionnelles (absent → degraded avec reason)
[feature."match_view.cadence"]
requires = ["match_events"]

[feature."match_view.tug_of_war"]
requires = ["match_events"]

[feature."match_view.impact_roles"]
requires = ["match_events"]

[feature."match_view.killer_victim"]
requires = ["killer_victim", "match_events"]

[feature."match_view.expected_stats"]
requires = ["match_skill"]

[feature."synthesis.weapon_breakdown"]
requires = ["weapon_kills"]
degraded_without = ["medals"]

[feature."engagement_score"]
requires = ["match_events"]

[feature."career.csr_progression"]
requires = ["csr"]

[feature."career.lusr_progression"]
requires = ["lusr"]
```

#### 1.7b.2 Algorithme de calcul

Au boot du serveur, pour chaque titre enregistré :

```
pour chaque feature F :
  manquants_requis  = { d ∈ F.requires        | data[d] == false }
  manquants_optnels = { d ∈ F.degraded_without | data[d] == false }

  si manquants_requis non vide :
    status = "unavailable"
    reason = "Requires {join(manquants_requis)} data"
  sinon si manquants_optnels non vide :
    status = "degraded"
    reason = "{join(manquants_optnels)} data not available"
  sinon :
    status = "available"
    reason = ""
```

Le résultat est un `map[FeatureKey]FeatureStatus` figé au boot, exposé via injection (DI) sous `port.FeatureChecker`.

#### 1.7b.3 Couches Go

- [ ] `internal/domain/feature/` : types `FeatureKey`, `FeatureStatus` (`available` / `degraded` / `unavailable`), `FeatureMatrix`.
- [ ] `internal/games/{slug}/feature_matrix.go` : loader TOML.
- [ ] `internal/port/feature_checker.go` :
  ```go
  type FeatureChecker interface {
      Status(ctx context.Context, key FeatureKey) FeatureStatus
      Matrix(ctx context.Context) FeatureMatrix
  }
  ```
- [ ] `internal/service/feature_matrix_service.go` : implémente `port.FeatureMatrixService`, expose la matrice complète (consommée par handler `/api/v1/title/feature_matrix`).
- [ ] `internal/api/handlers/feature_matrix.go` : handler **Huma** `GET /api/v1/title/feature_matrix` (si Phase 3b démarrée ; sinon chi).

#### 1.7b.4 Pattern d'usage côté service

Tout service avec une feature gated :

```go
func (s *MatchViewService) buildCadence(ctx context.Context, ...) (*ChartSeries, error) {
    if s.features.Status(ctx, "match_view.cadence").IsUnavailable() {
        return nil, nil  // omitempty côté DTO
    }
    if s.features.Status(ctx, "match_view.cadence").IsDegraded() {
        slog.WarnContext(ctx, "feature degraded",
            "feature", "match_view.cadence",
            "reason", s.features.Status(ctx, "match_view.cadence").Reason)
    }
    // calcul normal
}
```

#### 1.7b.5 Pattern d'usage côté frontend (anticipé Phase 5)

```tsx
const status = useFeatureStatus("match_view.cadence");
// → { status: "degraded" | "available" | "unavailable", reason: string }

<FeatureGate feature="match_view.cadence">
  <Cadence data={...} />
</FeatureGate>

// FeatureGate logic :
//   available   → render children
//   degraded    → render children + <Badge variant="partial" tooltip={reason} />
//   unavailable → render <FeatureUnavailable reason={reason} /> ou null selon prop hidden
```

#### 1.7b.6 Tests obligatoires

- [ ] `feature_matrix_test.go` : algorithme de calcul, 4 cas (tous available, 1 degraded, 1 unavailable, cascade).
- [ ] **Test de cohérence TOML ↔ DB** : pour chaque `data=true` dans `capabilities.toml`, vérifier sur fixture `halo_full` que la table existe + a ≥ 1 row. CI fail si dérive.
- [ ] **Test snapshot feature_matrix** : pour Halo Infinite et `synthetic_test_title`, golden file de la matrice complète. Diff = breaking change documenté.
- [ ] Test parité multi-titre : la suite Halo passe avec toutes les features `available`. La suite synthetic passe avec seulement les features compatibles avec son TOML.

**Critère de complétion** :
- `capabilities.toml` Halo Infinite exhaustif (toutes les data + toutes les features).
- `capabilities.toml` synthetic_test_title minimal (subset).
- `port.FeatureChecker` injecté dans tous les services qui ont une feature gated.
- Route `GET /api/v1/title/feature_matrix` opérationnelle, snapshot stable.
- Aucun service n'utilise `if titleSlug == "halo_infinite"` pour gater une feature (lint custom).

### Phase 1.8 — Outillage diagnostic Lab (moyen, différable)

**Effort** : 3-4 jours
**Risque** : faible (read-only, pas de mutation)
**Livrable** : page Lab avec section « Title Capabilities Diagnostic » qui montre la réalité DB (rows count par table), la compare au TOML déclaré, expose les drifts. Bouton « Export TOML draft » qui copie un bloc `[data]` prêt à coller dans le presse-papier. CLI `levelup-titles diagnose` headless équivalent.

**Note v2.5** : Phase 1.8 reste différable sans bloquer Phase 2+. Si l'urgence est de migrer les services, faire 0 → 1.7b → 2 directement et revenir à 1.8 plus tard.

#### 1.8.1 Workflow opérateur (ajout d'un titre)

```
1. Créer internal/games/halo_mcc/{adapter_data,adapter_semantic,ddl}/
2. Créer config/titles/halo_mcc/capabilities.toml MINIMAL (tout false)
3. Sync une fois → la DB se peuple
4. Ouvrir Lab → "Title Capabilities Diagnostic" → titre = halo_mcc
5. Voir le tableau "Réalité DB" : ce qui peuple vs ce qui est vide
6. Cliquer "Export TOML draft" → presse-papier contient le bloc [data] correct
7. Coller dans capabilities.toml, ajuster les commentaires, git commit
```

#### 1.8.2 Couches Go

- [ ] `internal/domain/diagnostic/` : `DiagnosticReport`, `DataCapabilityStatus` (declared/actual/drift), `FeatureDiscrepancy`.
- [ ] `internal/port/table_inspector.go` :
  ```go
  type TableInspector interface {
      CountRows(ctx context.Context, slug, table string) (int64, error)
      ListExpectedTables(slug string) []string  // depuis le mapping fields.toml
  }
  ```
- [ ] `internal/port/title_diagnostic_service.go` :
  ```go
  type TitleDiagnosticService interface {
      RunDiagnostic(ctx context.Context, slug string) (*DiagnosticReport, error)
      GenerateTOMLDraft(ctx context.Context, slug string) (string, error)
  }
  ```
- [ ] `internal/platform/duckdb/table_inspector.go` : impl. Utilise `PathResolver` pour atteindre la bonne DB par titre. Read-only.
- [ ] `internal/service/title_diagnostic_service.go` : compose `TableInspector` + `FeatureChecker` (de Phase 1.7b) + le mapping `fields.toml`.
- [ ] `internal/api/handlers/lab_diagnostic.go` : handler Huma `GET /api/v1/lab/title/{slug}/diagnostic` + `GET /api/v1/lab/title/{slug}/toml-draft`. Auth admin requise.
- [ ] **Aucune écriture côté serveur** (D10). Le bloc TOML est sérialisé en string et renvoyé tel quel.

#### 1.8.3 CLI complémentaire

- [ ] `cmd/levelup-titles/diagnose.go` : sous-commande qui appelle `service.RunDiagnostic` directement (pas via HTTP), output text-table (défaut) ou JSON (`--format=json`).

```bash
levelup-titles diagnose --slug halo_mcc
# Output:
# Data capabilities:
#   match_events     declared=false  actual=true (1234 rows)  ← DRIFT
#   match_skill      declared=true   actual=true   (567 rows) OK
#   weapon_kills     declared=true   actual=false (0 rows)    ← DRIFT
#
# Features:
#   match_view.cadence       calculated=unavailable  actual=available  ← UPGRADE
#   synthesis.weapon_breakdown calculated=available actual=unavailable ← REGRESSION
```

#### 1.8.4 Frontend Lab section

- [ ] Composant `apps/web/src/features/lab/TitleDiagnosticSection.tsx` :
  - Selector titre (current title par défaut, switch possible si admin)
  - Tableau `<DataCapabilitiesTable>` : key / declared / actual / drift / last sync
  - Tableau `<FeatureDiscrepanciesTable>` : key / status calculé / status réel / discrepancy
  - Bouton `<Button onClick={copyTomlDraft}>Export TOML draft →</Button>` qui :
    - fetch `/api/v1/lab/title/{slug}/toml-draft`
    - copie dans `navigator.clipboard.writeText(...)`
    - toast "TOML draft copié — collez dans capabilities.toml"
  - Aucun bouton qui écrit sur disque côté serveur (D10).
- [ ] Tests Vitest : rendering correct des 3 états (no drift / drifts / vide), copie presse-papier mockée, hidden si user non admin.

#### 1.8.5 Tests obligatoires

- [ ] `title_diagnostic_service_test.go` : 4 scénarios — no drift, drift data only, drift feature only, cascade (data drift → feature drift propagé).
- [ ] `table_inspector_test.go` : impl DuckDB sur `:memory:`, 3 cas (table absente / présente vide / présente avec rows).
- [ ] Test handler Huma `lab_diagnostic_test.go` : auth admin requise (401 sans), réponse JSON conforme schema OpenAPI.
- [ ] Test CLI `diagnose_test.go` : output stable sur fixtures (golden files).
- [ ] Test frontend Vitest : `TitleDiagnosticSection.test.tsx` couvre les 3 états + le clipboard.

#### 1.8.6 Logging spécifique

- `slog.InfoContext(ctx, "title_diagnostic: report generated", "title", slug, "data_drifts", n, "feature_drifts", m)` à chaque run.
- Pas de log Warn par drift individuel (bruit) — le rapport agrégé suffit.

**Critère de complétion** :
- Lab section opérationnelle, screenshot dans la PR.
- CLI `levelup-titles diagnose` produit un output stable testable.
- Aucun endpoint ne mute le TOML côté serveur (lint : pas de `os.WriteFile` dans handlers Lab).
- Les 3 layers (port + service + impl + handler + frontend) testés indépendamment.

### Phase 1.9 — Watcher / présence multi-title routing (NOUVELLE 2026-06-13, post-v2.5)

**Effort** : 3-5 j
**Risque** : moyen (touche `internal/watcher/daemon`, la `MatchQueue` et la signature `Coordinator`/`SyncTrigger`), mais **entièrement additif derrière une garde `DefaultSlug`** → comportement mono-titre strictement identique tant qu'un seul titre est enregistré.
**Branche cible** : `refactor/title-agnostic-services` (commit dédié) ou `feat/watcher-title-routing` si livrée isolément.
**Prérequis** : Phase 1.6 (pool tokens clé `(titleSlug, gamertag)`) — **livrée**. Pleinement exerçable seulement quand un 2e titre est enregistré (Registry + son `MatchFetcher`), mais **toute la plomberie se pose dès maintenant** derrière la garde `DefaultSlug`, testée avec un Registry à 2 titres fixtures.

**Contexte / diagnostic (audit code 2026-06-13)** : la *détection* de présence est déjà title-agnostic, mais toute la chaîne en aval est câblée Halo Infinite. Le poller « reconnaît un titre tracké » sans « router par titre ».

| Maillon | État | Fuite mono-titre |
|---|---|---|
| Détection présence | title-agnostic | `daemon.makePresenceHandler` → `titleReg.MatchPresence(titleID)` → `MatchByXboxTitleID` itère **tous** les titres enregistrés ; `TitleDescriptor` porte déjà `XboxTitleID` + `SteamAppID` |
| `MatchFetcher` | Halo-only | 1 seul fetcher partagé = `HaloMatchFetcher` (API Halo `GetMatchHistory`), câblé au boot (`cmd/server/main.go` ~l.1728) puis injecté dans `DaemonConfig.MatchFetcher` |
| `PlayerWatcher` | pas de titre | aucun champ `titleSlug` ; `OnPresenceActive` ne propage pas le `td.Slug` matché |
| Chaîne sync | pas de titre | `MatchRequest` / `CoordinatorRequest` = `{Gamertag, XUID, MatchIDs}` sans `TitleSlug` → sync sur titre par défaut |
| Steam | inactif | `MatchBySteamAppID` non utilisé ; `SteamPoller` implémenté mais non câblé (note W8) |

**Décision de modèle** : **1 `PlayerWatcher` par gamertag** (pas par `(gamertag, titleSlug)`). Un humain joue à un seul jeu à la fois sur un device, et l'event de présence dit lequel. Le watcher mémorise le **titre actif courant** (`activeTitleSlug`) et route match-poll + sync dessus ; un changement de titre redémarre le `MatchPoller` contre le fetcher du nouveau titre. Variante `(gamertag, titleSlug)` écartée (lourde, sans bénéfice — pas de jeu simultané).

**Tâches** (ordre de risque croissant) :
- [ ] **Propager le titre matché** : `makePresenceHandler` passe `td.Slug` à `OnPresenceActive(ctx, titleSlug)`. `PlayerWatcher` stocke `activeTitleSlug` (sous `mu`). Exposer via `WatcherStatus.Players[].title` (observabilité UI / WatcherCard).
- [ ] **`MatchFetcher` par titre** : remplacer le champ unique `DaemonConfig.MatchFetcher` par un `MatchFetcherResolver` (`FetcherFor(titleSlug) (MatchFetcher, bool)`). `HaloMatchFetcher` = impl `halo_infinite`. `startPoller` résout le fetcher via `activeTitleSlug` ; **si aucun fetcher** (titre sans support) → log Warn + rester Idle (ne JAMAIS poller l'API Halo par défaut pour un autre titre). Câblage `main.go` : resolver `{halo_infinite: HaloMatchFetcher}`. Garde compat : resolver nil / titleSlug vide → `DefaultSlug`.
- [ ] **Threader `TitleSlug` dans la chaîne sync** : ajouter `TitleSlug` à `MatchRequest` (queue), `CoordinatorRequest`, et `SyncTrigger.TriggerSync(ctx, titleSlug, gamertag, xuid, matchIDs)`. Le `SyncRunner` cible la bonne DB via `PathResolver(titleSlug)`. Compat : `TitleSlug` vide → `title.DefaultSlug` (même garde que Phase 1.6).
- [ ] **(Optionnel) Steam fallback title-aware** : si/quand le `SteamPoller` est câblé, `MatchBySteamAppID(gameid)` résout le titre, même threading. Hors scope tant que W8 n'est pas activé.
- [ ] **Garde-fou lint** : aucun `if titleSlug == "halo_infinite"` dans `internal/watcher/` (lint `no_slug_comparison` déjà actif). Routage via resolver + `PathResolver` uniquement.

**Tests** (par couche) :
- [ ] `daemon_test.go` : Registry à 2 titres fixtures ; event présence avec le `XboxTitleID` du titre B → `activeTitleSlug == "B"` + fetcher résolu = mock B (pas Halo).
- [ ] `player_watcher_test.go` : changement HI → B redémarre le `MatchPoller` contre le fetcher B ; B → titre sans fetcher → Idle sans aucun fetch.
- [ ] Queue / Coordinator : `TitleSlug` propagé jusqu'au `SyncRunner` (mock) ; vide → `DefaultSlug`.
- [ ] **Non-régression** : Registry à 1 titre → `WatcherStatus` byte-identique à aujourd'hui (snapshot).

**Logging** : `slog.InfoContext(ctx, "watcher: titre actif", "gamertag", gt, "title", slug)` à chaque transition de titre (clé standard `"title"`). Pas de log par poll.

**Critère de complétion** : avec un Registry à 2 titres + 2 fetchers mock, un event de présence sur le titre B route match-poll **et** sync vers B. Avec `halo_infinite` seul enregistré, comportement strictement identique à aujourd'hui (zéro régression mono-titre). Aucun `MatchFetcher` Halo invoqué pour un titre non-Halo.

### Phase 2 — Repository abstrait par FieldKey (moyen, +1 j OpenSpartan)

> **Extensions audit 2026-06-14 (à RE-VÉRIFIER)** — la canonicalisation déborde des données match, cf. [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) : modèle rangs/tiers carrière (MT-07), chaîne LUSR/poids qui **importe** `halo_infinite` depuis `internal/sync` (MT-15), extraction JSON participant + persist mono-DB (MT-14), progression `defaultProgressionTitleSlug` + PrestigeBundle (MT-19). + le pipeline **world-stats** est hors des 7 services → [PMT-7](PLAN_MULTITITRE_PERIPHERY.md) (MT-03). Index : [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md).

**Effort** : 6-8 jours (vs 5-7 j en v2.4 ; +1 j test OpenSpartan continuity D11)
**Risque** : moyen (refactor des repos existants, mais migration progressive service-par-service)
**Livrable** : nouvelle interface `port.MatchFieldRepository` qui prend des `[]FieldKey`, retourne `map[FieldKey]*canonical.Value`. Implémentation DuckDB qui résout le SELECT via TOML mapping.

- [ ] **Bench préliminaire** (1 j, BLOQUANT) : implémenter une version POC de `LoadMatchFields` avec SELECT dynamique sur Q12 (scoreboard, 4 joins). Comparer perf vs Q12 actuel sur un dataset de 1000 matchs. Si > 15% slowdown → garder Q12 spécialisé pour les hot paths, n'utiliser le SELECT dynamique que pour les routes peu sollicitées.
- [ ] Créer `internal/port/match_field_repository.go` :
  ```go
  type MatchFieldRepository interface {
      LoadMatchSummary(ctx, matchID) (*canonical.MatchSummary, error)
      LoadMatchParticipant(ctx, matchID, xuid) (*canonical.PlayerMatchRow, error)
      LoadMatchFields(ctx, matchID, xuid, []FieldKey) (map[FieldKey]*canonical.Value, error)
      LoadScoreboardFields(ctx, matchID, []FieldKey) ([]map[FieldKey]*canonical.Value, error)
  }
  ```
- [ ] Implémenter `internal/platform/duckdb/match_field_repo.go` :
  - Construit le SELECT dynamique depuis le TOML mapping.
  - Map les rows vers `*canonical.Value` (D1 = wrapper typé).
  - **Sémantique D2** : FieldKey absente du map = capability non supportée ; FieldKey présente avec `*Value = nil` = NULL en DB ; FieldKey présente avec valeur = OK.
- [ ] Tests integration `match_field_repo_integration_test.go` :
  - Halo Infinite : tous les FieldKeys retournent valeurs ou NULL cohérents.
  - `synthetic_test_title` : seules les FieldKeys déclarées dans son TOML sont résolues, les autres absentes du map.
- [ ] **NOUVEAU — Test OpenSpartan continuity (D11)** : `openspartan_waypoint_continuity_test.go` :
  - Fixture A : DB primée par `ImportFromOpenSpartan` sur 20 matchs.
  - Fixture B : Même DB + sync Waypoint sur 5 matchs additionnels (Day-1+).
  - Pour chaque match dans A∪B, `MatchFieldRepository.LoadMatchFields(...)` retourne le même set de FieldKeys avec valeurs valides (ou NULL cohérents).
  - Critère : aucun FieldKey présent dans les rows Waypoint qui soit absent dans les rows OpenSpartan (ou si différence : doc explicite dans le mapper OpenSpartan).
- [ ] Migration progressive (D6) : chaque service migré en PR séparée :
  1. `match_view_service.go` (pilote, plus complexe)
  2. `synthesis_service.go`
  3. `home_service.go`
  4. `explorer_service.go`
  5. `match_history_service.go`
  6. `career_service.go`
  7. `timeseries_service.go`

**Critère de complétion** : aucun service n'importe `internal/platform/duckdb` directement. Tous dépendent de `port.MatchFieldRepository` ou des autres ports existants. Test OpenSpartan continuity PASS.

### Phase 3 — Nettoyage DTO + migration vers Huma (lourd, +33 handlers)

**Effort** : 22-32 jours (vs 18-25 j v2.4 ; +5-7 j à cause de +33 handlers)
**Risque** : élevé (impact contrat front + rewrite ~113 handlers)
**Livrable** : `domain/match_view.go` ne contient plus que les view-model DTO propres ; tous les handlers chi sont migrés vers Huma ; `api/openapi.yaml` est auto-généré et le client TS front aussi.

> **Pourquoi fusionner Phase 3 (OpenAPI) et ex-Phase 4 (DTO clean)** : Huma génère l'OpenAPI depuis les types Input/Output. Si on migre les handlers vers Huma AVANT de nettoyer les DTOs, on définit les Output structs sur les types `domain` actuels (Halo-specific) puis on les rewrite — soit deux passes sur 113 handlers. Fusionner = un seul passage par handler.

#### Phase 3a — Cleanup DTO (5-7 j)

- [ ] Déplacer les 9 types `*Raw` ([domain/match_view.go:505-662](apps/go-api/internal/domain/match_view.go#L505) : `MatchMetaRaw`, `PlayerMatchStatsRaw`, `ScoreboardRaw`, `BulkMedalRaw`, `BulkWeaponKillRaw`, `MatchEnrichmentRaw`, `MedalRaw`, `EventRaw`, `WeaponKillRaw`, +`MatchHistAvgRow`) de `domain/match_view.go` vers `platform/duckdb/raw_types.go`. Ils ne traversent plus la frontière service.
- [ ] Réécrire `MatchExpectedStats` : tous les champs `*float64 omitempty`. Pas de `HasExpectedData bool` (le front teste `expected_kills !== null`).
- [ ] Idem `MatchScoreboardRow` : les 30+ champs deviennent `*T omitempty`.
- [ ] Le service `match_view_service.go` consulte le `TitleDataAdapter` (via `port.MatchFieldRepository`) puis compose le DTO ; les champs absents → nil → JSON omit.

#### Phase 3b — Migration Huma (17-25 j, +33 handlers depuis v2.4)

**Inventaire actuel** : `find apps/go-api/internal/api/handlers -name "*.go" | wc -l` = **113 fichiers** (incl. tests). Le scope a cru de ~30 handlers depuis la v2.4 (achievements, OpenSpartan import, Xbox SSO oauth, CSR par-match, citations, LUSR chain, career-live, perf-chain, admin-auto-sync).

- [ ] **Setup Huma sur chi existant** : créer `huma.NewAPI(chi)` adapter, sans toucher aux routes existantes. Phase 3b démarre avec 0 handler migré, OpenAPI vide. **Poser le tag git `phase-3b-start` sur ce commit** — c'est le référent pour le lint D13.
- [ ] **Activer le lint D13** : `tests/lint/no_new_chi_handler_test.go` qui git-diff vs `phase-3b-start` et rejette tout nouveau handler chi non listé dans `tests/lint/handlers_migration_progress.json`. Mise à jour de ce JSON au fur et à mesure de la migration.
- [ ] **Pattern de migration handler** : pour chaque handler, créer un struct `Input` (path/query/body params via tags `path:`, `query:`, `body:`, `header:`) et un struct `Output` (réponse). Le corps du handler devient `func(ctx, *Input) (*Output, error)`.
  ```go
  type MatchViewInput struct {
      PlayerSlug string `path:"player_slug"`
      MatchID    string `path:"match_id"`
      Playlist   string `query:"playlist,omitempty"`
  }
  type MatchViewOutput struct {
      Body domain.MatchViewResponse
  }
  func (h *Handler) MatchView(ctx context.Context, in *MatchViewInput) (*MatchViewOutput, error) {
      resp, err := h.svc.GetMatchView(ctx, in.PlayerSlug, in.MatchID, in.Playlist)
      if err != nil { return nil, mapErrorToHuma(err) }
      return &MatchViewOutput{Body: *resp}, nil
  }
  ```
- [ ] **Migration progressive par groupe de handlers** (ordre de risque croissant) — **inventaire à jour mai 2026** :
  1. Handlers simples GET sans body (~45 handlers : health, bootstrap, settings, gamertag, prestige, achievements, citations, lab, admin, etc.) — ~4-5 j
  2. Handlers GET avec query params filtrés (~35 handlers : match_history, sessions, explorer, timeseries, career, compare, synthesis, season_pass, etc.) — ~6-7 j
  3. Handlers POST/PUT avec body (~20 handlers : sync, watcher, match_favorite, match_exclusion, openspartan import, xbox-sso oauth, notifications mark-read, etc.) — ~4-5 j
  4. Handlers complexes (~13 handlers : match_view, synthesis détaillé, squad_v2, season_pass full, career_live, admin auto-sync diag, etc.) — ~4-5 j
- [ ] **Validation des inputs** : les tags `minLength:`, `maxLength:`, `pattern:`, `enum:` dans Input sont vérifiés par Huma avant d'appeler le handler. Supprimer les regex manuels (`playlistOrSessionPattern` etc.) au profit de tags Huma.
- [ ] **Mapping erreurs** : créer `mapErrorToHuma(err) error` qui convertit `port.ErrCapabilityNotSupported` → `huma.Error404NotFound`, `port.ErrInvalidInput` → `huma.Error400BadRequest`, etc.
- [ ] **Suppression `api/openapi.yaml` manuel** : le YAML est désormais généré par `huma.NewAPI(...)` au boot, exposé sur `/openapi.yaml` et committé via `go generate ./tools/openapi-export`. CI fail si diff.
- [ ] **Régénération client TS frontend** : `apps/web/src/lib/api/types.gen.ts` régénéré depuis le nouveau YAML via `openapi-typescript`. Régénération automatique en CI.
- [ ] **Tests de contrat snapshot** : pour chaque route, JSON de réponse comparé à un golden file (fixtures Halo + synthetic_test_title).
- [ ] **Smoke test E2E** : 9 routes critiques (home, match-view, match-history, sessions, palmares, season-pass, synthesis, career, openspartan-import) renvoient toujours du JSON valide vs le schema généré.

**Critère de complétion** :
- `domain/match_view.go` ne contient plus aucun type `*Raw` ni aucun champ marqué « Halo Infinite : pas d'expected_assists ».
- `api/openapi.yaml` est généré + git-tracked ; aucune édition manuelle ; CI fail si diff vs `huma.NewAPI` au boot.
- 100% des handlers utilisent le pattern `func(ctx, *Input) (*Output, error)` Huma.
- Aucun handler n'utilise plus directement `chi.URLParam`, `r.URL.Query()`, ou `json.NewEncoder(w).Encode(...)`.
- Lint D13 actif : un PR introduisant un handler chi rouge.

### Phase 4 — Sync flags génériques (moyen)

**Effort** : 5-6 jours (inchangé)
**Risque** : moyen (touche le CLI utilisateur)
**Livrable** : `SyncScope` est partiellement FieldKey-based (pour les champs stats) et garde des flags top-level pour les concerns non-FieldKey (sessions, citations, engagement, dry-run, etc.).

**Note D11** : OpenSpartan import est hors-scope de `SyncScope` — c'est un import bootstrap one-shot via `POST /import/openspartan`, pas un mode de sync. Phase 4 ne le touche pas.

- [ ] **Découpler** :
  - **Champs FieldKey-based** : `TeamMMR`, `KillsExpected`, `DeathsExpected`, `Damage`, `AvgLife`, `GrenadeKills`, `MeleeKills`, `PowerWeaponKills`, `HeadshotKills`, `MaxSpree`, `KDARecalc`, `TimePlayed` → remplacés par `Fields []canonical.FieldKey`.
  - **Champs « stratégie de sync » non-FieldKey** : `Medals`, `Events`, `Skill`, `KillerVictim`, `Sessions`, `Citations`, `EngagementScores`, `LUSR`, `CSR`, `SkillRank`, `ComebackBadges`, `PlayableDuration`, `Aliases`, `Assets`, `PVEStats`, `Weapons` → restent flags top-level (groupés dans `SyncScope.Operations`).
  - **Options générales** : `DryRun`, `MaxMatches`, `RequestsPerSec`, `DetectionMode` → inchangés.
- [ ] CLI : `--field <key>` répétable + `--operation <name>` répétable. Aliases historiques préservés via map (`--mmr` → `--field team_mmr --field enemy_mmr`, `--skill` → `--field` set + `--operation skill`).
- [ ] `backfill_flags.go::PBit*` : générer le bitmask dynamiquement depuis le TOML capabilities au boot. Le mapping `bit_position ↔ FieldKey` est versionné (changement = bump schema_version + script de migration de la table `backfill_completed`).
- [ ] `FindMatchesMissingData` prend `[]FieldKey` + `[]Operation` et construit le `WHERE` dynamiquement. Bench (cf. Phase 2) pour vérifier que les index DuckDB existants tiennent.
- [ ] Deprecation warning sur les vieilles options CLI : 2 versions, retrait à v6.5.

**Critère de complétion** : ajouter un nouveau field stats = 1 ligne dans le TOML + 1 ligne dans le `TitleDataAdapter`. Le CLI le détecte automatiquement. Les opérations restent enumérées (pas de scope creep).

### Phase 5 — Frontend canonical-aware + FeatureGate (moyen)

> **Extensions audit 2026-06-14 (à RE-VÉRIFIER)** — au-delà des hooks, cf. [EXT-5](PLAN_MULTITITRE_PERIPHERY.md) : retrait des constantes littérales `TITLE_SLUG='halo_infinite'` + fallback silencieux `DEFAULT_TITLE_SLUG` (MT-12, + lint front anti-littéral) ; externalisation des tables Halo client-side (teamNames/outline-colors/tier grids/badge `HINF`/225 HP, MT-13). Index : [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md).

**Effort** : 7-9 jours (inchangé)
**Risque** : moyen (ajout d'abstractions front, mais OpenAPI gen capture les changes)
**Livrable** : composants UI utilisent `useFieldLabel(FieldKey)` / `useCapability(cap)` / `useFeatureStatus(featureKey)` au lieu d'accéder directement aux propriétés JSON par hardcoded path. `<FeatureGate>` masque/dégrade automatiquement les sections selon le `feature_matrix` du titre actif.

- [ ] **Codegen TS depuis canonical Go** (D5) : script `tools/codegen/canonical-ts/` qui lit `canonical/fields.go` et écrit `apps/web/src/lib/canonical/fields.ts`. CI lint vérifie que le fichier généré est à jour.
- [ ] **Client API TS auto-généré** depuis l'OpenAPI Huma (Phase 3b) via `openapi-typescript`. Remplace le client manuel actuel.
- [ ] Hook `useFieldLabel(field, locale)` lit le manifest i18n exposé via API `/api/v1/title/manifest` (déjà existant côté back via TitleSemanticAdapter).
- [ ] Hook `useCapability(cap)` consulte `/api/v1/title/capabilities` (Phase 1.7a).
- [ ] Hook `useFeatureStatus(featureKey)` consulte `/api/v1/title/feature_matrix` (Phase 1.7b, cached au boot via TanStack Query infinite stale time). Retourne `{ status, reason }`.
- [ ] Composant `<FeatureGate feature="..." [hidden]>` à 3 modes :
  - `available` → render children sans modification
  - `degraded` → render children + `<PartialDataBadge tooltip={reason} />`
  - `unavailable` → render `<FeatureUnavailable reason={reason} />` (skeleton avec explication) ou `null` si `hidden` prop fourni
- [ ] Composants `<StatRow field="kills_expected" value={...} />` qui se masquent automatiquement si value undefined.
- [ ] Capability gating au routeur : `<Route capability="lusr">` n'affiche pas la route si titre n'expose pas LUSR.
- [ ] Tests Vitest sur les hooks + tests Playwright sur la dégradation `synthetic_test_title`.

**Critère de complétion** : ajouter un titre = uploader son `i18n manifest` + `capabilities.toml` côté back. Le front l'utilise sans nouveau code TS.

---

## 5. Tests — exigences strictes (BLOQUANTES)

> **Règle absolue** : chaque seuil ci-dessous est BLOQUANT pour l'exit d'une phase. Aucune dérogation. Si un seuil n'est pas tenu, la phase n'est PAS close — le travail continue jusqu'à ce que le seuil soit atteint OU une dérogation est documentée dans `thought_log.md` AVEC date de remediation engagée.

### 5.1 Seuils de couverture par couche (mesurés via `go test -coverprofile`)

| Couche | Couverture min | Type de test | Métrique de qualité supplémentaire |
|---|:-:|---|---|
| `internal/games/canonical/` | **95%** | Unitaire pur (zero IO) | 100% des FieldKey référencés dans TOMLs ont un test de round-trip |
| `internal/analysis/` | **90%** | Unitaire pur | Property-based (gopter ou rapid) sur ratios, KDA, accuracy |
| `internal/games/{slug}/adapter_*.go` | **85%** | Unitaire avec fixtures JSON réelles | Chaque `Load*()` testé sur ≥3 fixtures (best/typical/edge case) |
| `internal/games/{slug}/ddl/` | **100%** des steps | Test MigrationRunner sur DB vide | Schema produit comparé à un golden file via `pragma_table_info` |
| `internal/port/` | N/A (interfaces) | Mocks utilisables | Lint : aucune impl directe de `port.*` n'est référencée hors `platform/` |
| `internal/platform/duckdb/match_field_repo.go` | **85%** | Integration `:memory:` + dataset réaliste | Test pour CHAQUE FieldKey du TOML : présent+valeur, présent+NULL, absent du TOML |
| `internal/service/` | **80%** | Tests avec mocks `port.*` + fakes `games.*` | Tous les chemins de dégradation `ErrCapabilityNotSupported` testés |
| `internal/api/handlers/` (Huma) | **85%** | `huma.NewTestAPI` + golden snapshot JSON | 100% des routes : test happy path + 1 test d'erreur (404 / 400 / 500 mappé) |
| `apps/web/` (hooks + features) | **75%** | Vitest unit + Playwright E2E | Critical path : home, match-view, sessions, season-pass — Playwright systématique |

**Outils de mesure obligatoires** :
- Go : `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` — CI fail si seuil non tenu par package.
- TS : `vitest --coverage` avec `coverage.thresholds` configurés dans `vitest.config.ts` — CI fail si seuil non tenu.

### 5.2 Tests de non-régression OBLIGATOIRES (par phase)

#### Phase 2 — migration service par service vers `port.MatchFieldRepository`

Pour CHAQUE PR migrant un service :

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot JSON avant/après** | Pour 10 matchs réels (5 PVP + 5 PVE), capturer la réponse JSON avant la migration (sur `main`) et après. | `diff -u before.json after.json` = vide OU diff documenté ligne par ligne dans la PR avec justification |
| **Snapshot SQL avant/après** | Logger les queries DuckDB exécutées (via `slog` Debug) avant/après. | Plan d'exécution comparable (pas de `Sequential Scan` introduit là où il n'y en avait pas) |
| **Bench latence** | `go test -bench` sur le handler complet pour 100 matchs. | p95 latency post-migration ≤ p95 pre-migration × 1.10 (slowdown max 10%) |
| **Test de parité multi-titre** | Sur `synthetic_test_title` (subset fields), même handler répond 200 + JSON valide. | Pas de panic, pas de 500, FieldKeys absents → omitempty respecté |
| **Test OpenSpartan continuity (NOUVEAU v2.5, D11)** | DB primée OpenSpartan + 5 matchs Waypoint additionnels. Handler répond pour les 2 sous-ensembles. | Pas de FieldKey present-on-Waypoint / absent-on-OpenSpartan non documenté |

#### Phase 3a — cleanup DTO

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot JSON 100% routes** | Pour 5 fixtures par route (1 par profil joueur représentatif), avant/après. | Diff = uniquement des champs nouvellement nullables qui passent de `0`/`""` à `null` (omitempty), JAMAIS un champ qui disparaît |
| **Test typescript front** | Le client TS regen passe `tsc --noEmit` sur tout `apps/web/src/`. | 0 erreur TS |
| **Snapshot OpenAPI YAML** | Diff entre YAML manuel actuel et YAML produit par les types Go nouvellement nettoyés. | Diff documenté champ par champ dans la PR |

#### Phase 3b — migration Huma

Pour CHAQUE groupe de handlers migré :

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot JSON full** | Toutes les routes du groupe testées via `httptest` (via Huma) ET via `chi` (route legacy si encore présente) sur les mêmes fixtures. | Bytes identiques OU diff documenté |
| **Test des middlewares** | Pour chaque middleware existant (auth, CSRF, slog HTTP, rate-limit, title extractor, LoopbackOnly pour admin), test `httptest` confirme l'invocation sur une route Huma. | 6/6 middlewares passent |
| **Test des erreurs** | Chaque erreur mappée (`ErrCapabilityNotSupported` → 404, `ErrInvalidInput` → 400, etc.) testée. | Status code + body conformes au schema OpenAPI Huma |
| **Test client TS regen** | `openapi-typescript` régénère le client TS sans erreur, `apps/web/` compile encore. | Build front passe, snapshot des types front committé |
| **Test validation Huma** | Pour chaque route, 3 inputs invalides (manque param, format faux, valeur hors enum). | Réponse 400 + body lisible, pas de panic, pas de 500 |
| **Lint D13 actif (NOUVEAU v2.5)** | Tentative d'ajouter un handler chi sans le déclarer dans `handlers_migration_progress.json` → CI rouge. | Lint custom passe sur tous les commits intermédiaires |

#### Phase 4 — sync flags

| Test | Modalité | Critère pass |
|---|---|---|
| **Compatibility CLI** | Tous les anciens flags (`--mmr`, `--skill`, `--assists-expected`, etc.) sont parsés et produisent le même bitmask `backfill_completed` qu'avant. | Test snapshot bitmask sur 20 combinaisons de flags |
| **Idempotence backfill** | Lancer backfill 2 fois avec le même scope → la 2e run ne fait aucun appel API ni écriture DB. | `slog` capture confirme 0 appel API en 2e run |

#### Phase 5 — frontend canonical-aware

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot rendu Playwright** | Capture screenshot par route critique (home, match-view, sessions, season-pass) sur Halo Infinite + synthetic_test_title. | Diff visuel automatique (pixelmatch) ≤ 0.5% sur Halo. synthetic_test_title : sections capability-gated absentes |
| **Test capability gating** | 4 routes sans capability requise → 404 propre côté back, route absente du routeur front. | Pas d'erreur console, pas de blank page |
| **Test codegen TS** | `make canonical-ts-gen` produit un fichier identique au commit. | CI fail si diff |

### 5.3 Tests de parité multi-titre (exécutés à CHAQUE phase)

Le job CI `synthetic_test_title-parity` lance la suite COMPLÈTE des tests avec `LEVELUP_TITLE=synthetic_test_title`. Critères :

- Toutes les routes répondent (200 ou 404 capability) — JAMAIS 500.
- Toutes les FieldKeys absentes du TOML synthetic sont omises du JSON via `omitempty`.
- Aucun test n'utilise `if titleSlug == "halo_infinite"` (lint custom `tests/lint/no_slug_comparison_test.go`).
- Le frontend chargé avec `LEVELUP_TITLE=synthetic_test_title` n'a aucune erreur console.

### 5.4 Cas de dégradation explicitement testés

Pour chaque phase, les cas suivants sont obligatoirement couverts par un test (pas implicites, écrits noir sur blanc) :

- Title sans `expected_kills` → service retourne `*MatchScoreboardRow.ExpectedKills = nil` → JSON omit → UI masque la case → pas d'erreur 500.
- Title sans capability `match_film` → endpoint `/match/{id}/film` retourne 404 + body `{"capability":"match_film","supported":false}`.
- Title sans capability `lusr` → home page n'affiche pas le panneau LUSR (sans casser le reste).
- Title sans capability `firefight` → routes PVE absentes du routeur front, 404 côté back.
- Repo retourne `ErrCapabilityNotSupported` → handler dégrade au lieu de paniquer.
- Adapter retourne `nil` → service compose un DTO partiel sans crash.
- **DB OpenSpartan-only (Day-0)** → toutes les routes critiques (home, match-view, match-history) répondent 200, dégradation OK pour features dépendantes de Waypoint-only data (ex. live career rank).

### 5.5 Datasets de tests OBLIGATOIRES

Pour les tests d'intégration et de non-régression, **trois datasets obligatoires** :

- **Halo réaliste** (`testdata/integration/halo_full/`) : 50 matchs hétérogènes (PVP + PVE, ranked + social, plusieurs maps/playlists, plusieurs joueurs, ≥1 match avec données partielles/NULL).
- **Synthetic minimal** (`testdata/integration/synthetic/`) : 10 matchs avec subset de FieldKeys (kills, deaths, match_id, durations uniquement).
- **OpenSpartan-primed** (`testdata/integration/openspartan_primed/`, NOUVEAU v2.5) : DB issue d'un `ImportFromOpenSpartan` sur fixture SQLite snapshot, 20 matchs Day-0 + 5 matchs Waypoint Day-1+. Sert au test de continuité Phase 2.

CI échoue si un test d'intégration tourne sur fixtures < ces datasets.

---

## 6. Logging — exigences strictes (BLOQUANTES)

> **Règle absolue** : tout code introduit par ce plan respecte 100% des règles ci-dessous. Lint CI fail si violation. Aucune dérogation.

### 6.1 Lint CI obligatoires (à activer en Phase 0)

| Lint | Cible | Action si violation |
|---|---|---|
| `forbidigo` ban `fmt.Println`, `fmt.Printf`, `fmt.Print` | tout `internal/` et `cmd/` | CI fail |
| `forbidigo` ban `log.Print*`, `log.Fatal*`, `log.Panic*` (stdlib `log`) | tout `internal/` et `cmd/` | CI fail |
| `forbidigo` ban `panic(` hors `init()` ou tests | tout `internal/` et `cmd/` | CI fail |
| `revive` règle `unused-parameter` sur les handlers Huma | `internal/api/handlers/` | CI fail |
| Lint custom `slog-context-required` | toute fonction qui prend `ctx context.Context` doit utiliser `slog.*Context` (pas `slog.*` sans contexte) | CI fail |
| Lint custom `error-must-be-logged-or-returned` | tout `err != nil` doit soit `return err` (avec wrap), soit `slog.ErrorContext` | CI fail |
| **Lint custom `no-new-chi-handler` (NOUVEAU D13)** | tout nouveau handler `func(w, r)` ajouté après `phase-3b-start` doit être déclaré dans `handlers_migration_progress.json` | CI fail |

### 6.2 Standards de logging par opération

**Erreur non-triviale** (toute erreur autre que `io.EOF`, `sql.ErrNoRows` quand attendu, `context.Canceled`) :
```go
slog.ErrorContext(ctx, "match_field_repo: scan failed",
    "err", err,
    "title", titleSlug,
    "match_id", matchID,
    "xuid", xuid,
    "operation", "load_match_fields")
return fmt.Errorf("scan match %s: %w", matchID, err)
```

**Opération significative** (DB query > 100ms, API call externe, sync de plus de 1 match, migration step) :
```go
slog.InfoContext(ctx, "sync: backfill batch completed",
    "title", titleSlug,
    "player", gamertag,
    "match_count", len(matches),
    "duration", time.Since(start),
    "operation", "backfill_batch")
```

**Capability absente** (titre ne supporte pas un field/feature) — émis 1 fois par boot via `sync.Once` par couple `(title, capability)` :
```go
slog.WarnContext(ctx, "title_data_adapter: capability unsupported",
    "title", titleSlug,
    "capability", capName,
    "consumer", "match_view_service")
```

**Trace de debug** (utilisateur final ne voit pas, mais utile pour diag) :
```go
slog.DebugContext(ctx, "match_field_repo: load fields",
    "title", titleSlug,
    "match_id", matchID,
    "field_count", len(fields),
    "fields_requested", fieldKeys)
```

### 6.3 Clés structurées normalisées (whitelist exhaustive)

Toute nouvelle clé hors whitelist nécessite mise à jour de cette section + entrée `thought_log.md`.

| Clé | Type | Usage |
|---|---|---|
| `err` | error | Toujours présent en cas d'erreur |
| `title` | string (slug) | Identifiant titre courant |
| `match_id` | string | UUID du match |
| `xuid` | string | XUID Xbox du joueur |
| `player` | string (gamertag) | Display name du joueur |
| `field` | string (FieldKey) | FieldKey canonical concerné |
| `capability` | string (CapabilityKey) | Capability concernée |
| `operation` | string | Nom de l'opération (`load_match_fields`, `backfill_batch`, `migration_step`, etc.) |
| `duration` | time.Duration | Durée d'opération significative |
| `match_count` | int | Nombre de matchs (batch) |
| `field_count` | int | Nombre de FieldKeys |
| `route` | string | Path HTTP (Huma) |
| `status` | int | HTTP status code |
| `consumer` | string | Service qui consomme (pour traces inter-couches) |
| `source` (NOUVEAU v2.5) | string | `"openspartan"` ou `"waypoint"` pour distinguer l'origine d'une row |

### 6.4 Métriques expvar obligatoires (par phase)

L'observabilité ne se résume pas aux logs. Pour chaque nouvelle couche introduite, exposer via `expvar` (cohérent ADR 0009) :

| Métrique | Phase d'introduction | Format |
|---|---|---|
| `match_field_repo.load_duration_ms` | Phase 2 | Histogram (p50/p95/p99) |
| `match_field_repo.fields_unsupported_count` | Phase 2 | Counter par couple `(title, field)` |
| `huma.request_count_by_status` | Phase 3b | Counter par couple `(route, status)` |
| `huma.request_duration_ms` | Phase 3b | Histogram par route |
| `huma.validation_failures_count` | Phase 3b | Counter par route |
| `migration_runner.steps_applied` | Phase 1.5 | Counter par titre |
| `sync.field_backfill_count` | Phase 4 | Counter par FieldKey |
| `feature_checker.status_count_by_status` | Phase 1.7b | Gauge `(title, status)` |
| `openspartan_import.rows_written` (NOUVEAU v2.5) | Phase 1.5 (compat) | Counter par titre |

**Test obligatoire** : un test `expvar_smoke_test.go` par phase qui vérifie que les métriques sont exposées et incrémentent correctement après une opération.

### 6.5 Vérifications automatiques en fin de phase

```bash
# 1. Aucun fmt.Println résiduel
rg "fmt\.Print(ln|f)?\(" apps/go-api/internal/ apps/go-api/cmd/ \
  | grep -v "_test.go" \
  | grep -v "// approved:" \
  && echo "FAIL: fmt.Print* found" && exit 1

# 2. Aucun log.Printf résiduel
rg "^[^/]*\blog\.(Print|Fatal|Panic)" apps/go-api/internal/ apps/go-api/cmd/ \
  && echo "FAIL: stdlib log usage found" && exit 1

# 3. Tout slog dans contexte ctx utilise *Context
rg "slog\.(Debug|Info|Warn|Error)\(" apps/go-api/internal/ apps/go-api/cmd/ \
  | grep -v "_test.go" \
  && echo "FAIL: slog without Context found" && exit 1

# 4. Aucune erreur silencieuse
rg "err != nil \{[\s]*\}" apps/go-api/internal/ apps/go-api/cmd/ \
  && echo "FAIL: silent error swallow found" && exit 1
```

Ces 4 vérifications sont en CI à partir de Phase 0 et BLOQUANTES.

---

## 7. Multi-titres — exigences capability

- [ ] Chaque FieldKey est listée dans `config/titles/{slug}/mappings/fields.toml` (ou héritée d'un default si extension future).
- [ ] Si une route HTTP nécessite une capability et le titre actif ne la supporte pas → handler renvoie `404 ErrCapabilityNotSupported` (pas de panic, pas de 500).
- [ ] Le frontend appelle `GET /api/v1/title/capabilities` au boot pour connaître les features dispos.
- [ ] **Aucune** comparaison `if titleSlug == "halo_infinite"` dans le code applicatif (77 occurrences actuelles dans `internal/service/` à éliminer). Tout via `HasCapability(cap)` ou `FeatureChecker.Status()`. Lint custom Phase 0 : `tests/lint/no_slug_comparison_test.go`.

---

## 8. Phase Exit Gate — règles strictes (BLOQUANTES)

> **Aucune phase ne peut être déclarée "close" tant que TOUS les items du Exit Gate sont DONE et datés.**
>
> - Pas de "TODO", "report à plus tard", "OK pour MVP", "partiel". Chaque item est binaire : DONE ou NOT DONE.
> - Aucun item n'est optionnel. Si un item ne s'applique pas à un cas, ça doit être DONE-N/A avec justification écrite dans `thought_log.md` AVANT de fermer la phase.
> - Le tag git `phase-{N}-exit` n'est posé que quand le tableau exit gate est 100% DONE.

### 8.1 Format de l'Exit Gate

Chaque phase a son tableau Exit Gate au format suivant :

```
| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| Description courte de l'item | DONE | 2026-05-20 | commit abc123 / PR #42 / CI run #99 | GS |
```

- **Statut** : `DONE` ou `NOT DONE` (jamais "WIP", "PARTIAL", "TBD"). `DONE-N/A` autorisé uniquement avec justification thought_log.
- **Date** : YYYY-MM-DD du jour où l'item est passé en DONE.
- **Evidence** : lien commit hash + PR + CI run qui prouve.
- **Validateur** : initiales du validateur humain (Guillaume = GS).

### 8.2 Items communs à TOUTES les phases (Exit Gate transverse)

Ces 12 items sont obligatoires en fin de chaque phase, en plus des items spécifiques :

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| 1. `go test ./... -race` passe sans erreur | NOT DONE | | | |
| 2. `go vet ./...` sans warning | NOT DONE | | | |
| 3. Lint logging CI (cf. §6.5) passe (4 vérifs) | NOT DONE | | | |
| 4. Couverture par couche ≥ seuil §5.1 (mesurée) | NOT DONE | | | |
| 5. Job CI `synthetic_test_title-parity` passe | NOT DONE | | | |
| 6. Tests de non-régression de la phase passent (cf. §5.2) | NOT DONE | | | |
| 7. Datasets `halo_full`, `synthetic`, `openspartan_primed` à jour (cf. §5.5) | NOT DONE | | | |
| 8. Aucun fichier nouveau > 500 lignes | NOT DONE | | | |
| 9. Aucune fonction nouvelle > 80 lignes | NOT DONE | | | |
| 10. Métriques expvar §6.4 exposées + test smoke OK | NOT DONE | | | |
| 11. Entrée `thought_log.md` rédigée (date, décision, résultats, prochaine étape) | NOT DONE | | | |
| 12. Tag git `phase-{N}-exit` posé sur le HEAD de la branche | NOT DONE | | | |

### 8.3 Exit Gate Phase 0 — décisions et setup

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| ADR 0014 (title-agnostic + DDL isolation) créé | NOT DONE | | | |
| ADR 0015 (Huma + D13 gel handlers) créé | NOT DONE | | | |
| ADR 0016 (Feature Matrix + D12 migration code→TOML) créé | NOT DONE | | | |
| ADR 0017 (Title Diagnostic + D10 presse-papier) créé | NOT DONE | | | |
| Branche `refactor/title-agnostic-services` créée | NOT DONE | | | |
| Plan référencé dans `CLAUDE.md` § Décisions architecturales | NOT DONE | | | |
| 7 lints CI §6.1 activés et BLOQUANTS (dont D13 placeholder désactivé jusqu'au tag `phase-3b-start`) | NOT DONE | | | |
| Job CI `synthetic_test_title-parity` créé (vide est OK pour Phase 0) | NOT DONE | | | |
| Datasets `testdata/integration/halo_full/` + `synthetic/` + `openspartan_primed/` créés | NOT DONE | | | |
| 12 items de §8.2 (Exit Gate transverse) | NOT DONE | | | |

### 8.4 Exit Gate Phase 1 — FieldKey exhaustifs (5 tables)

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| Inventaire colonnes `queries_*.go` produit (`/tmp/cols.txt` committé) | NOT DONE | | | |
| 100% colonnes inventoriées ont un FieldKey dans `canonical/fields.go` | NOT DONE | | | |
| 100% FieldKeys ont une section dans `fields.toml` (5 tables) | NOT DONE | | | |
| `constants.toml` Halo créé (medal IDs, mode prefixes) | NOT DONE | | | |
| Test `fields_test.go::TestExhaustiveTOMLCoverage` passe | NOT DONE | | | |
| Aucun `medal_name_id = <int>` inline dans `queries_*.go` | NOT DONE | | | |
| Couverture `canonical/` ≥ 95% | NOT DONE | | | |
| Property-based test sur ratios (KDA, accuracy) | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.5 Exit Gate Phase 1.5 — DDL par titre (effort doublé v2.5)

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `internal/games/halo_infinite/ddl/steps_*.sql` créés (toutes les catégories §Phase 1.5) | NOT DONE | | | |
| `internal/migration/` ne contient plus de DDL Halo-specific (20+ fichiers déplacés) | NOT DONE | | | |
| `MigrationRunner` accepte un `TitleDataAdapter.MigrationSteps()` paramètre | NOT DONE | | | |
| `synthetic_test_title/ddl/` créé (schema minimal, base = synthetic_title_b) | NOT DONE | | | |
| Test : `MigrationRunner` crée la DB synthetic from scratch sans toucher au code partagé | NOT DONE | | | |
| Test golden : schema produit = `pragma_table_info` snapshot | NOT DONE | | | |
| **Test OpenSpartan continuity** : import D11 + sync Waypoint sur même DB → schema cohérent | NOT DONE | | | |
| `internal/ops/` (backup, restore, diagnose) auditée et adaptée multi-titre | NOT DONE | | | |
| Couverture `internal/games/{slug}/ddl/` 100% des steps | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.55 Exit Gate Phase 1.7a — TOML capabilities binaires (NOUVELLE v2.5)

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `config/titles/halo_infinite/capabilities.toml` créé (section [capabilities] binaire) | NOT DONE | | | |
| `config/titles/synthetic_title_b/capabilities.toml` créé (subset) | NOT DONE | | | |
| Loader `internal/games/{slug}/capabilities.go::LoadCapabilities()` créé | NOT DONE | | | |
| Adapters `halo_infinite/adapter_data.go` + `synthetic_title_b/adapter.go` consomment le loader (plus de hardcode) | NOT DONE | | | |
| Endpoint `GET /api/v1/title/capabilities` retourne le map du titre actif | NOT DONE | | | |
| Test : suppression ligne TOML synthetic → endpoint reflète + service dégrade | NOT DONE | | | |
| Aucun `games.CapabilityMap{...}` hardcodé dans `internal/games/*/adapter*.go` (lint custom) | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.56 Exit Gate Phase 1.7b — Feature Matrix 3 états + cascade

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `internal/domain/feature/` créé (FeatureKey, FeatureStatus à 3 états, FeatureMatrix) | NOT DONE | | | |
| `port.FeatureChecker` interface créée | NOT DONE | | | |
| Loader TOML `internal/games/halo_infinite/feature_matrix.go` créé (section [data] + [feature.*]) | NOT DONE | | | |
| `capabilities.toml` Halo Infinite étendu (section [data] + section [feature.*] exhaustives) | NOT DONE | | | |
| `capabilities.toml` synthetic_test_title étendu (subset minimal) | NOT DONE | | | |
| Algo de calcul implémenté + 4 cas testés (all-available, degraded, unavailable, cascade) | NOT DONE | | | |
| Test cohérence TOML ↔ DB : pour chaque `data=true`, table existe + ≥ 1 row sur fixture halo_full | NOT DONE | | | |
| Test snapshot feature_matrix Halo + synthetic (golden files) | NOT DONE | | | |
| Handler `GET /api/v1/title/feature_matrix` opérationnel (Huma si Phase 3b démarrée, chi sinon) | NOT DONE | | | |
| `FeatureChecker` injecté dans les services qui en ont besoin | NOT DONE | | | |
| Aucun service n'utilise `if titleSlug == ...` pour gater une feature (lint custom) | NOT DONE | | | |
| Couverture `internal/domain/feature/` ≥ 95%, services impactés ≥ 80% | NOT DONE | | | |
| Métriques expvar `feature_checker.status_count_by_status` exposées | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.57 Exit Gate Phase 1.8 — Outillage diagnostic Lab

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `internal/domain/diagnostic/` créé (DiagnosticReport, DataCapabilityStatus, FeatureDiscrepancy) | NOT DONE | | | |
| `port.TableInspector` + `port.TitleDiagnosticService` interfaces créées | NOT DONE | | | |
| Impl DuckDB `table_inspector.go` (read-only, count rows) | NOT DONE | | | |
| Service `title_diagnostic_service.go` (compose TableInspector + FeatureChecker + fields mapping) | NOT DONE | | | |
| Handler Huma `GET /api/v1/lab/title/{slug}/diagnostic` (auth admin) | NOT DONE | | | |
| Handler Huma `GET /api/v1/lab/title/{slug}/toml-draft` (auth admin, retourne string) | NOT DONE | | | |
| CLI `cmd/levelup-titles/diagnose.go` (output text-table + --format=json) | NOT DONE | | | |
| Frontend `TitleDiagnosticSection.tsx` créé dans `apps/web/src/features/lab/` | NOT DONE | | | |
| Bouton Export TOML draft : copie presse-papier OK, toast confirmation | NOT DONE | | | |
| **Aucune écriture serveur** (lint : pas de `os.WriteFile` dans handlers Lab) | NOT DONE | | | |
| Tests service : 4 scénarios (no drift / drift data / drift feature / cascade) | NOT DONE | | | |
| Tests CLI : golden files sur fixture halo_full | NOT DONE | | | |
| Tests Vitest : 3 états + clipboard mocké + hidden si non-admin | NOT DONE | | | |
| Couverture `internal/service/title_diagnostic_service.go` ≥ 85% | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.6 Exit Gate Phase 2 — `port.MatchFieldRepository` + 7 services migrés

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `port.MatchFieldRepository` interface créée | NOT DONE | | | |
| `platform/duckdb/match_field_repo.go` impl créée | NOT DONE | | | |
| Bench préliminaire : `LoadMatchFields` p95 ≤ Q12 actuel × 1.15 | NOT DONE | | | |
| Test integration `:memory:` couvre : présent+valeur, présent+NULL, absent du TOML (3 scénarios par FieldKey) | NOT DONE | | | |
| **Test OpenSpartan continuity (NOUVEAU v2.5)** : import + Waypoint sync → mêmes FieldKeys ou diff documenté | NOT DONE | | | |
| Service 1 migré : `match_view_service.go` (PR atomique avec snapshots avant/après) | NOT DONE | | | |
| Service 2 migré : `synthesis_service.go` | NOT DONE | | | |
| Service 3 migré : `home_service.go` | NOT DONE | | | |
| Service 4 migré : `explorer_service.go` | NOT DONE | | | |
| Service 5 migré : `match_history_service.go` | NOT DONE | | | |
| Service 6 migré : `career_service.go` | NOT DONE | | | |
| Service 7 migré : `timeseries_service.go` | NOT DONE | | | |
| Aucun service n'importe `internal/platform/duckdb` directement (lint custom) | NOT DONE | | | |
| Snapshot JSON pour 10 matchs réels : diff = vide ou justifié | NOT DONE | | | |
| Bench latence : aucun handler avec slowdown > 10% | NOT DONE | | | |
| Couverture `internal/service/` ≥ 80%, `internal/platform/duckdb/match_field_repo.go` ≥ 85% | NOT DONE | | | |
| Métriques `match_field_repo.load_duration_ms` + `.fields_unsupported_count` exposées | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.7 Exit Gate Phase 3a — cleanup DTO

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| 9 types `*Raw` déplacés vers `platform/duckdb/raw_types.go` | NOT DONE | | | |
| `domain/match_view.go` ne contient plus de type `*Raw` | NOT DONE | | | |
| `MatchExpectedStats` 100% nullable (`*float64 omitempty`) | NOT DONE | | | |
| `MatchScoreboardRow` 100% nullable (30+ champs) | NOT DONE | | | |
| `domain/match_view.go` ne contient plus de commentaire « Halo Infinite : pas de... » | NOT DONE | | | |
| `domain/match_view.go` ≤ 500 lignes (vs 799 actuelles) | NOT DONE | | | |
| Snapshot JSON 100% routes : diff documenté champ par champ | NOT DONE | | | |
| `tsc --noEmit` sur `apps/web/` passe | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.8 Exit Gate Phase 3b — migration Huma (113 handlers)

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `huma.NewAPI(chi)` adapter en place | NOT DONE | | | |
| Tag git `phase-3b-start` posé | NOT DONE | | | |
| Lint D13 (`no-new-chi-handler`) activé et BLOQUANT | NOT DONE | | | |
| Groupe 1 (handlers GET simples) migré : ~45 handlers | NOT DONE | | | |
| Groupe 2 (handlers GET + query params) migré : ~35 handlers | NOT DONE | | | |
| Groupe 3 (handlers POST/PUT) migré : ~20 handlers | NOT DONE | | | |
| Groupe 4 (handlers complexes) migré : ~13 handlers | NOT DONE | | | |
| **100% handlers migrés** (~113) — aucun `func(w http.ResponseWriter, r *http.Request)` métier hors middlewares | NOT DONE | | | |
| Aucun `chi.URLParam` hors middlewares | NOT DONE | | | |
| Aucun `r.URL.Query()` hors middlewares | NOT DONE | | | |
| Aucun `json.NewEncoder(w).Encode` hors middlewares | NOT DONE | | | |
| `mapErrorToHuma` créé avec 100% des erreurs port mappées | NOT DONE | | | |
| `api/openapi.yaml` généré par `huma.NewAPI` au boot, committé via `go generate` | NOT DONE | | | |
| CI fail si `openapi.yaml` diff entre boot et commit | NOT DONE | | | |
| Client TS frontend regen via `openapi-typescript`, build front passe | NOT DONE | | | |
| Snapshot JSON full : 100% routes, bytes identiques OU diff documenté | NOT DONE | | | |
| Test des 6 middlewares chi (auth, CSRF, slog, rate-limit, title, LoopbackOnly) avec route Huma : 6/6 OK | NOT DONE | | | |
| Test validation Huma : 3 inputs invalides par route, 0 panic, 0 status 500 | NOT DONE | | | |
| Couverture `internal/api/handlers/` ≥ 85% | NOT DONE | | | |
| Métriques `huma.request_*` exposées | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.9 Exit Gate Phase 4 — sync flags génériques

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `SyncScope.Fields []FieldKey` introduit | NOT DONE | | | |
| `SyncScope.Operations []string` introduit (flags non-FieldKey) | NOT DONE | | | |
| 12 champs FieldKey-based retirés de `SyncScope` (TeamMMR, KillsExpected, ...) | NOT DONE | | | |
| CLI `--field <key>` répétable + `--operation <name>` répétable | NOT DONE | | | |
| Aliases historiques préservés (test snapshot bitmask sur 20 combinaisons) | NOT DONE | | | |
| Deprecation warnings sur les vieilles options CLI (date butoir v6.5 documentée) | NOT DONE | | | |
| `FindMatchesMissingData` accepte `[]FieldKey` + `[]Operation` | NOT DONE | | | |
| Test idempotence backfill : 2e run = 0 appel API + 0 écriture DB | NOT DONE | | | |
| Métrique `sync.field_backfill_count` par FieldKey exposée | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.10 Exit Gate Phase 5 — frontend canonical-aware

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `tools/codegen/canonical-ts/` créé | NOT DONE | | | |
| `apps/web/src/lib/canonical/fields.ts` généré + committé | NOT DONE | | | |
| CI lint fail si `fields.ts` ≠ sortie codegen | NOT DONE | | | |
| Client API TS regen depuis OpenAPI Huma, build front passe | NOT DONE | | | |
| Hook `useFieldLabel(field, locale)` créé + tests Vitest | NOT DONE | | | |
| Hook `useCapability(cap)` créé + tests Vitest | NOT DONE | | | |
| Hook `useFeatureStatus(featureKey)` créé + tests Vitest (3 états) | NOT DONE | | | |
| Composant `<FeatureGate feature=... [hidden]>` créé + tests (3 modes : available render, degraded render+badge, unavailable masque/skeleton) | NOT DONE | | | |
| Composant `<StatRow field=... value=... />` créé + tests | NOT DONE | | | |
| Capability gating au routeur (`<Route capability="...">`) | NOT DONE | | | |
| Snapshot Playwright Halo : diff visuel ≤ 0.5% sur 4 routes critiques | NOT DONE | | | |
| Snapshot Playwright synthetic_test_title : capability-gated sections absentes | NOT DONE | | | |
| Aucun composant ne hardcode un FieldKey en string littéral (lint custom) | NOT DONE | | | |
| Couverture `apps/web/src/lib/canonical/` ≥ 90%, hooks ≥ 80% | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.11 Validation finale (cross-phase)

À la clôture de la Phase 5, ajouter une PR « validate: synthetic_test_title E2E » qui :

1. Crée un dossier `internal/games/synthetic_test_title/` complet (data + semantic + asset_url + ddl) — peut être un alias / extension de `synthetic_title_b` existant.
2. Définit un `fields.toml` avec un subset de FieldKeys (kills, deaths, match_id, durations).
3. Lance la suite COMPLÈTE des tests existants (Go + TS) avec `LEVELUP_TITLE=synthetic_test_title`.

Cette PR est le **critère de réussite final**. Elle doit passer SANS modifier :
- `internal/service/`
- `internal/api/`
- `apps/web/src/{features,components,routes}/`

Si une de ces zones doit être modifiée pour faire passer la PR, c'est que le plan a échoué — retour en Phase {N} pour corriger.

**Validation finale** : ajouter un titre `synthetic_test_title` avec un sous-ensemble de fields (kills, deaths, match_id, durations) ; vérifier que :
- Toutes les routes répondent sans 500.
- Tous les fields canonical absents → JSON omit ou `null`.
- Le front masque les sections concernées sans erreur console.
- Le pipeline CI `synthetic_test_title-build-and-test` est vert.
- Aucune modification dans `internal/service/`, `internal/api/`, `apps/web/src/{features,components}/` n'a été nécessaire au-delà de l'enregistrement du titre dans le Resolver.

---

## 9. Blockers / risques connus

| Risque | Probabilité | Mitigation |
|---|:-:|---|
| Migration Huma sur 113 handlers casse une route en prod | Élevée | Migration par groupes (4 groupes), tests de contrat snapshot par route avant/après, smoke test E2E systématique. Possibilité de revert handler-par-handler si régression. |
| Validation Huma plus stricte qu'avant rejette des inputs auparavant tolérés | Moyenne | Audit des regex/whitelist actuels (ex: `playlistOrSessionPattern`), porter en tags Huma à l'identique. Tests des cas limites avec inputs de prod. |
| Client TS regen change la forme des types (camelCase, optionals) → cascade front | Élevée | Phase 3b ne casse pas le contrat JSON (omitempty conservé). Test snapshot des fixtures front avant/après regen. Coordonner avec Phase 5. |
| Bench Phase 2 montre slowdown > 15% sur SELECT dynamique | Moyenne | Garder `Q12` spécialisé pour les hot paths (match_view), n'utiliser FieldKey-based que pour les routes secondaires. Documenter la décision. |
| Migration des sync flags casse les scripts utilisateur (`--mmr`, `--skill`) | Moyenne | Aliases historiques préservés via map, deprecation warning, retrait à v6.5 |
| ADR 0011 conflictuel avec le plan | Faible (résolu Phase 0) | Phase 0 crée ADR 0014 |
| Coût total dépassé (62 j vs estimé 82 j) | Élevée | Phasage strict, chaque phase mergeable indépendamment, possibilité de geler après Phase 3a (cible « DTO propres + Huma reporté ») |
| DDL par titre rompt les outils ops existants (backup, restore, diagnose) | Moyenne | Audit `internal/ops/` en Phase 1.5, adapter `BackupRunner` pour itérer sur tous les titres enregistrés |
| Codegen TS canonical (D5) divergence avec Go | Faible | CI lint compare le fichier généré au commit, fail si diff |
| Huma compatibility avec middlewares chi existants (auth, CSRF, slog HTTP) | Moyenne | Huma utilise `huma.NewAPI(chi)` adapter — les middlewares chi continuent de tourner. Tester explicitement chaque middleware en début de Phase 3b. |
| **Nouveaux handlers ajoutés pendant Phase 3b creusent la dette (NOUVEAU v2.5)** | Élevée | D13 + lint custom `no-new-chi-handler` bloque tout nouveau handler chi après tag `phase-3b-start`. Tous les nouveaux handlers (feature non listée) doivent être écrits directement en Huma. |
| **Double système Capability code + TOML pendant migration 1.7a (NOUVEAU v2.5)** | Faible | D12 : migration en 2 PR atomiques (move puis extend). Pas de phase intermédiaire où les deux coexistent en runtime. |
| **OpenSpartan import écrit une DDL divergente de Waypoint (NOUVEAU v2.5)** | Moyenne | Phase 1.5 audite explicitement le mapper OpenSpartan. Phase 2 ajoute un test de continuité (D11) qui détecte tout drift FieldKey-level. |

---

## 10. Anti-patterns à NE PAS faire

- **God refactor en une seule PR** — chaque phase = 1 ou plusieurs PR mergeables.
- **Casser l'API publique sans deprecation** — tous les changements de schema OpenAPI passent par un cycle deprecate → remove (2 versions).
- **Pousser `canonical.PlayerMatchRow` directement dans le DTO HTTP** — viole ADR 0011. Le DTO HTTP reste un view-model service composé.
- **Schema EAV** — perd les bénéfices colonnar de DuckDB. Préférer une DB par titre (Phase 1.5).
- **Cas particuliers `if title == "halo_infinite"`** — toujours via capability. Lint custom CI.
- **Forcer la canonicalisation des features encore en chantier** (ex. nouvelles features Halo Infinite-only). Phase finale uniquement quand le périmètre est stable.
- **Mélanger flags FieldKey-based et flags non-FieldKey dans `SyncScope` sans découplage explicite** (Phase 4).
- **Régénérer OpenAPI manuellement après Phase 3** — la gen doit être en CI, pas dans la mémoire des devs.
- **Ajouter un nouveau handler chi après tag `phase-3b-start` (NOUVEAU v2.5)** — D13 + lint bloquant. Tous les nouveaux handlers doivent être écrits directement en Huma.
- **Modéliser OpenSpartan comme un pipeline continu (NOUVEAU v2.5)** — c'est un bootstrap one-shot (D11). Ne pas l'ajouter à `SyncScope` ni en faire un mode de sync.

---

## 11. Effort total estimé (révisé v2.5)

- Phase 0 : 2-3 j
- Phase 1 : 2-3 j
- Phase 1.5 : **8-10 j** (vs 4-5 j v2.4 — doubling dû à l'accumulation `internal/migration/steps_*.go`)
- Phase 1.7a : **2 j** (NOUVELLE v2.5 — TOML capabilities binaires light)
- Phase 1.7b : 4-5 j (ex-Phase 1.7 — Feature Matrix 3 états + cascade)
- Phase 1.8 : 3-4 j (différable)
- Phase 2 : **6-8 j** (vs 5-7 j — +1 j test OpenSpartan continuity)
- Phase 3a : 5-7 j (cleanup DTO)
- Phase 3b : **17-25 j** (vs 13-18 j v2.4 — +33 handlers depuis v2.4, scope ~113 au total)
- Phase 4 : 5-6 j
- Phase 5 : 7-9 j

**Total** : **62-82 jours-personne**, étalable sur 4-5 mois sans blocage du reste du dev.

**Fenêtre minimale viable** : Phases 0 → 3a = **31-43 j** (vs 25-34 j v2.4) → état « services title-agnostic + DTO propres + feature matrix opérationnelle + diagnostic Lab pour onboarding nouveau titre, mais OpenAPI manuel ». Phase 3b (Huma) peut être différée d'un trimestre si le ROI n'est pas clair, MAIS dans ce cas :
- Le client TS front reste désynchronisé du back.
- D13 (gel des nouveaux handlers en Huma) reste inactif → la dette continue de croître au rythme observé (~15 handlers/mois).
- Plus on attend pour démarrer 3b, plus son coût grossit. **Recommandation v2.5 : ne pas différer 3b de plus de 2 trimestres** sous peine de scope creep ingérable.

**Phase 1.8 peut être différée** sans bloquer Phase 2+ : c'est de l'outillage opérateur, pas une dépendance des services. Si l'urgence est de migrer les services, faire 0 → 1.7a → 1.7b → 2 directement et revenir à 1.8 plus tard.

**ROI Huma seul** : validation auto, gen permanente, élimination de la dette OpenAPI manuel (qui se cumule à chaque feature — +33 routes en 2 semaines fin avril/début mai 2026, signe que le rythme est soutenu). Décisif.

**ROI title-agnostic** : marginal tant qu'il n'y a qu'1 titre. Décisif dès le 2e titre — chaque titre ajouté coûte ~1-2 jours (capabilities.toml + DataAdapter + ddl/) au lieu de plusieurs semaines.

---

## 12. Pré-requis avant démarrage

- [ ] Pas d'autre big refactor en parallèle (notamment côté frontend canonical pipeline).
- [ ] Cap définie sur les fields à inclure dans canonical (pas de scope creep en cours de Phase 1).
- [ ] Validation par toi (Guillaume) du phasage et des 13 décisions Phase 0 — quels risques tu acceptes, lesquels tu repousses.
- [ ] Décision sur la fenêtre minimale viable (Phases 0-3a, 31-43 j) vs full sweep (0-5, 62-82 j).
- [ ] Gel sur les features Halo-Infinite-only majeures pendant Phase 1.5 (pour éviter d'accumuler encore des `steps_player_*.go`).

---

## 13. Premier pas concret quand tu reprendras

```bash
git checkout main
git pull
git checkout -b refactor/title-agnostic-services

# Phase 0 — ADRs et setup (décisions D1-D13 actées)
# 1. Créer ADR 0014 (title-agnostic + DDL isolation + OpenSpartan bootstrap)
# 2. Créer ADR 0015 (adoption Huma + D13 gel nouveaux handlers)
# 3. Créer ADR 0016 (Feature Matrix + D12 migration code→TOML)
# 4. Créer ADR 0017 (Title Diagnostic + D10 presse-papier)
# 5. Entrée thought_log.md actant les 13 décisions
# Commit : "docs(adr): record decisions for title-agnostic refactor v2.5 (D1-D13)"

# Phase 1 — inventaire FieldKey (5 tables shared)
# rg "p\.\w+|mp\.\w+|mr\.\w+|me\.\w+|w\.\w+|kvp\.\w+" \
#    apps/go-api/internal/platform/duckdb/queries_*.go > /tmp/cols.txt
# Pour chaque ligne, vérifier/ajouter dans canonical/fields.go + halo_infinite/mappings/fields.toml
# Extraire les magic constants Halo (medal_id 1512363953, mode prefixes) vers constants.toml
# Commit : "feat(canonical): exhaustive FieldKey coverage for Halo Infinite (5 tables)"
```

---

## 14. Changelog

- **v2.5 (2026-05-18)** :
  - Revue après ~10 j de développement intensif (OpenSpartan import, Xbox SSO multi-user, achievements, citations enrichies, LUSR chain rework, perf-chain, career-live SwR, CSR per-match).
  - **3 nouvelles décisions** :
    - **D11** : OpenSpartan import = bootstrap one-shot (Day-0), pas un pipeline continu. TitleDataAdapter reste l'autorité du schéma. Hors-scope `SyncScope`.
    - **D12** : Migration `CapabilityMap` code → TOML en 2 PR atomiques (move puis extend 3 états). Évite double système.
    - **D13** : Gel des nouveaux handlers en Huma dès `phase-3b-start` tag. Lint custom bloquant `no-new-chi-handler`. Stoppe la croissance de la dette pendant la migration (rythme actuel ~15 handlers/mois).
  - **Phase 1.7 découpée en 1.7a (2 j, TOML capabilities binaires light) + 1.7b (4-5 j, ex-Phase 1.7 3 états + cascade)**.
  - **Phase 1.5 ré-estimée 4-5 j → 8-10 j** : doubling dû à l'accumulation `internal/migration/steps_*.go` (steps_player_lusr_chain_rework, steps_player_perf_chain, steps_player_prestige, steps_player_notifications, steps_metadata_prestige_seed ajoutés depuis v2.4). Liste exhaustive incluse.
  - **Phase 3b ré-estimée 13-18 j → 17-25 j** : 80 handlers → 113 handlers (+33 depuis v2.4). Migration groupes recalibrés (45 / 35 / 20 / 13).
  - **Phase 2 enrichie** : +1 j pour test de continuité OpenSpartan-primed → Waypoint-fed (D11). Nouveau dataset `testdata/integration/openspartan_primed/`.
  - **§9 risques** : 3 nouveaux risques (scope creep handlers, double système Capability code+TOML, OpenSpartan DDL divergente).
  - **§5.5 datasets** : 3 datasets obligatoires (vs 2 en v2.4) avec ajout `openspartan_primed`.
  - **Acquis ADR 0012** documenté en §0 : la sortie de `analysis/` Halo-only est déjà faite ; le plan construit dessus.
  - **synthetic_title_b existant** identifié comme base du futur `synthetic_test_title` §8.11.
  - **Effort total révisé** : 62-82 j (vs 50-67 j v2.4). Fenêtre minimale viable : 31-43 j (vs 25-34 j v2.4).
  - **Recommandation forte** : ne pas différer Phase 3b de plus de 2 trimestres sous peine de scope creep ingérable.
- **v2.4 (2026-05-07)** :
  - Décisions D9 + D10 actées (Phase 1.8 dédiée + export presse-papier).
  - Phase 1.8 NOUVELLE : outillage diagnostic Lab read-only. Couches Go (domain/diagnostic, port.TableInspector, port.TitleDiagnosticService, impl DuckDB, handler Huma admin) + CLI `levelup-titles diagnose` + frontend `TitleDiagnosticSection` avec bouton « Export TOML draft » (copie presse-papier uniquement). Effort 3-4 j.
  - ADR 0017 (Title Diagnostic) à créer en Phase 0.
  - Exit Gate Phase 1.8 ajouté avec 15 items dont lint « pas d'`os.WriteFile` dans handlers Lab » (garde-fou D10).
  - Effort total révisé : 50-67 j (vs 47-63 j v2.3).
  - Note opérationnelle : Phase 1.8 peut être différée sans bloquer Phase 2+ (outillage opérateur, pas dépendance des services).
- **v2.3 (2026-05-07)** :
  - Décisions D7 et D8 actées (3 états + TOML déclaratif).
  - Phase 1.7 NOUVELLE : Feature Matrix — modèle 2 niveaux (data capabilities + feature capabilities) + arbre de dépendances déclaratif TOML + 3 états (`available` / `degraded` / `unavailable` + reason). Algorithme de cascade au boot. Effort 4-5 j.
  - Phase 5 enrichie : `useFeatureStatus` + `<FeatureGate>` (3 modes : render / render+badge / skeleton). +1 j (7-9 j vs 6-8 j v2.2).
  - ADR 0016 (Feature Matrix) à créer en Phase 0.
  - Exit Gate Phase 1.7 ajouté avec 14 items (incl. test cohérence TOML ↔ DB).
  - Effort total révisé : 47-63 j (vs 42-57 j v2.2).
- **v2.2 (2026-05-06)** :
  - **§5 Tests refondu** : seuils de couverture chiffrés par couche (BLOQUANTS), 5 sous-sections (seuils, non-régression par phase, parité multi-titre, dégradation, datasets obligatoires)
  - **§6 Logging refondu** : 6 lints CI bloquants, standards par opération, whitelist de clés structurées, métriques expvar par phase, 4 vérifications shell de fin de phase
  - **§8 Phase Exit Gate** : refonte complète. Tableau binaire DONE/NOT DONE daté + validateur + evidence par phase. 12 items transverses + items spécifiques. Aucun item optionnel. Tag git `phase-{N}-exit` sur HEAD.
  - **§8.11 Validation finale** : PR `synthetic_test_title` E2E qui doit passer sans modifier service/api/features front. Critère de réussite ultime.
  - Aucun changement du phasage ni de l'effort total.
- **v2.1 (2026-05-06)** :
  - 6 décisions D1-D6 actées (cf. session interactive 2026-05-06) :
    - D1 : `Value{Kind, Int, Float, Str, Bool, Time}` (wrapper typé)
    - D2 : `map[FieldKey]*Value` (présent+nil = NULL, absent = unsupported)
    - D3 : DB physique par titre
    - D4 : **Huma intégré au plan** (rewrite ~80 handlers) au lieu de swag/kin-openapi
    - D5 : Codegen Go → TS pour canonical
    - D6 : Service par service en PR atomique, pas de feature flag
  - Phase 3 fusionne ex-Phases 3 et 4 (cleanup DTO + migration Huma) en Phase 3a + 3b
  - Renumérotation : ex-Phase 5 → Phase 4, ex-Phase 7 → Phase 5 (Phase 6 EAV skipped retirée)
  - ADR 0015 (Huma) ajouté en Phase 0
  - Effort total révisé : 42-57 j (vs 36-49 j en v2)
  - Fenêtre minimale viable décalée à Phase 3a (~18-25 j)
  - Risques mis à jour avec spécifiques Huma (validation stricte, middlewares chi, client TS regen)
- **v2 (2026-05-06)** :
  - Phase 0 enrichie avec 6 décisions techniques bloquantes (D1-D6) + spike OpenAPI gen
  - Alignement explicite avec ADR 0011 (3 adapters préservés, view-models côté service)
  - Phase 1 étendue à 5 tables shared (pas seulement match_participants) + magic constants Halo
  - Phase 1.5 NOUVELLE : DDL par titre (cohérence ADR 0008)
  - Phases 3 et 4 réordonnées : OpenAPI gen avant réécriture domain (évite cascade YAML manuel)
  - Phase 5 découplée : flags FieldKey-based vs flags « stratégie sync »
  - Phase 7 : codegen TS canonical (D5) au lieu de duplication manuelle
  - Effort total révisé : 36-49 j (vs 21-31 j en v1, sous-estimé d'un facteur ~1.5)
  - Fenêtre minimale viable Phases 0-4 (~25-35 j) introduite
  - CI gate `synthetic_test_title-build-and-test` formalisé
  - Anti-pattern « pousser canonical dans le DTO » ajouté
- **v1 (2026-05-06)** : version initiale, voir historique git.

---

**Auteur** : Claude (session 2026-05-06).
**Revue v2** : après audit plan v1 (cf. session du 2026-05-06).
**Revue v2.5** : après audit codebase 10 j plus tard (cf. session du 2026-05-18) — scope grossi, OpenSpartan/Xbox SSO/perf-chain/lusr-chain/career-live ajoutés, 113 handlers, 20+ steps_*.go accumulés.
**À traiter par** : Guillaume, plus tard.
