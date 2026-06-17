# Index — Multi-titre : registre complet des axes + carte des phases

> **Rôle** : point d'entrée UNIQUE pour répertorier **tous** les axes multi-titre du projet —
> ceux déjà planifiés (registre A) et ceux **oubliés** révélés par l'audit 2026-06-13/14 (registre B, `MT-01..MT-26`).
> **Créé** : 2026-06-14.
>
> **Documents liés** :
> - Master data-path : [PLAN_TITLE_AGNOSTIC_REFACTORING.md](PLAN_TITLE_AGNOSTIC_REFACTORING.md) + [tracker](PLAN_TITLE_AGNOSTIC_TRACKER.md)
> - Périphérie (specs détaillées des axes oubliés) : [PLAN_MULTITITRE_PERIPHERY.md](PLAN_MULTITITRE_PERIPHERY.md)
> - Arcs cross-titre : [PLAN_CROSS_TITLE_ARCS_BACKEND.md](PLAN_CROSS_TITLE_ARCS_BACKEND.md) · Familles d'armes : [PLAN_WEAPON_FAMILY_CANONICAL.md](PLAN_WEAPON_FAMILY_CANONICAL.md)

**Légende statut** : ✅ done · 🟡 partial · ⬜ todo/différé. **Statut axe** : `gap` (aucun plan) · `partiel` (phase y touche, laisse un reste) · `tracké` · `no-action`.
**⚠ = donnée à RE-VÉRIFIER** (voir doctrine ci-dessous).

---

## ⚠ Doctrine RE-VÉRIFIER (s'applique à CHAQUE axe et CHAQUE phase, existante ou nouvelle)

Tout ce registre — pointeurs `file:line`, statut, scope — est une **carte datée (2026-06-13/14), PAS une vérité figée**.
La re-vérification a déjà corrigé plusieurs erreurs de la passe d'audit initiale (`weapon_labels` :564-568 ≠ :463-640 ; `discovery_client.go:20` = UGC pas stats ; audience Spartan `urn:343:s3:services` manquée ; `BuildEngine` = câblage unique watcher+scheduler+HTTP ; `MetadataDBPath(slug)` isole déjà par chemin).

**Avant d'exécuter une phase (y compris les phases EXISTANTES du master), l'agent DOIT** :
1. re-grep / re-lire chaque évidence contre `HEAD` ;
2. re-valider que le couplage existe encore et que les `file:line` sont à jour ;
3. re-scoper (le gap a pu rétrécir ou grossir) ;
4. consigner la dérive dans la PR.

**Ne jamais prendre une spec pour un copier-coller** : c'est une hypothèse à reconfirmer. Doctrine complète + méthode `expand → parity-gate → contract` : voir [PLAN_MULTITITRE_PERIPHERY.md](PLAN_MULTITITRE_PERIPHERY.md).

---

## Registre A — Axes DÉJÀ planifiés (chemin data-lecture du match)

| Axe | Phase | Doc | Statut |
|---|---|---|:-:|
| FieldKey + `fields.toml` (stats canoniques) | Phase 1 | master | 🟡 |
| DDL/schéma par titre | Phase 1.5 | master (+ [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md)) | ✅ |
| Pool tokens clé `(titleSlug,gamertag)` | Phase 1.6 | master | ✅ |
| `capabilities.toml` + loader + endpoint | Phase 1.7a | master | ✅ |
| Feature-matrix 3 états + cascade | Phase 1.7b | master | ✅ |
| Outillage diagnostic Lab | Phase 1.8 | master | ⬜ |
| Watcher / présence — routing par titre | Phase 1.9 | master | ⬜ |
| Services title-agnostic (canonical) | Phase 2 | master (+ [EXT-2](PLAN_MULTITITRE_PERIPHERY.md)) | 🟡 |
| Cleanup DTO (`*Raw` hors domain) | Phase 3a | master | 🟡 |
| Migration Huma (113 handlers) | Phase 3b | master | ⬜ |
| Sync flags FieldKey-based | Phase 4 | master | ⬜ |
| Frontend canonical-aware + `<FeatureGate>` | Phase 5 | master (+ [EXT-5](PLAN_MULTITITRE_PERIPHERY.md)) | ⬜ |
| Arcs Prestige cross-titre (table `arc_titles`) | — | [cross-title-arcs](PLAN_CROSS_TITLE_ARCS_BACKEND.md) | ⬜ |
| Famille d'armes canonique | — | [weapon-family](PLAN_WEAPON_FAMILY_CANONICAL.md) | 🟡 |

---

## Registre B — Axes OUBLIÉS (audit 2026-06-13/14)

> Statut `partiel` = une phase master touche le mécanisme mais laisse un reste concret (colonne **Reste**). Tous ⚠ à re-vérifier.

| ID | Axe | Sév. | Statut | Phase | Évidence (1 pointeur, ⚠ re-vérif) |
|---|---|:-:|:-:|---|---|
| MT-01 | Hosts d'ingestion API (stats/economy/gamecms/skill/UGC/discovery) en const | blocker | ✅ Exit Gate | [PMT-1](PLAN_MULTITITRE_PERIPHERY.md) — Expand bff9a1df3 + 6 axes Contract → cf2afefe2 (privacy `:443` + leaderboard web-scrape déférés/documentés) | `games.EndpointResolver` + `[endpoints]` constants.toml |
| MT-02 | Acquisition auth (XSTS audience, SISU titleID 144209987, clearance `titles/hi`, scopes) | blocker | done | [PMT-2](PLAN_MULTITITRE_PERIPHERY.md) ✅ 5 legs (XSTS/Spartan/Clearance/SISU/scopes) + store namespacé titre 873637195 | `AuthDescriptor` + `PathResolver.WatcherTokensDirFor(slug)` + `MigrateWatcherTokens` |
| MT-11 | Scheduler/auto-sync : `NewSyncEngine` écrit en `DefaultSlug` (slug résolu puis jeté) | blocker | ✅ Exit Gate | [PMT-3](PLAN_MULTITITRE_PERIPHERY.md) — PR-1→4 → a934cde77 (V1+V2 écrivent per-titre, gate clé composite ; watcher per-titre documenté/inexerçable) | `NewSyncEngineForTitle` + ctxkeys carrier + `gateKey` |
| MT-04 | `app_settings.json` global (CurrentCSRSeasonID, friends, sessions, toggles) | major | gap | [PMT-4](PLAN_MULTITITRE_PERIPHERY.md) | `internal/config/config.go:96-99` |
| MT-06 | Enum Outcome `2/3/1/4` baké en SQL + citations + front | major | partiel | [PMT-5](PLAN_MULTITITRE_PERIPHERY.md) — Expand ✅ 4bc694fd7 ; Contract : sites Go sûrs migrés → `domain.Outcome*` + **lint ratchet** `no_raw_outcome_literal_test.go` (allowlist décroissante) ; reste = SQL/threading différés (assists/perf data-path, services sans adapter, front) → onboarding 2e titre | seam `mappings.OutcomeMappingSet` + ratchet archlint |
| MT-08 | `XboxTitleIDFor` + achievements (`WHERE title_id='halo_infinite'`) | major | done | [PMT-6](PLAN_MULTITITRE_PERIPHERY.md) ✅ PR1+PR2 e7f06fe71 (lecture `title_id=?` + XboxTitleIDFor registry-driven) + PR3 flag CLI `--title` (NewSyncEngineForTitle) → MT-08 100% | `GetAchievementDefinitions(ctx, slug)` + `(r *Registry).XboxTitleIDFor` + `cmd_sync_achievements --title` |
| MT-03 | World-stats / leaderboard (repos, enricher, CLIs, `DEFAULT 'halo_infinite'`) | major | done | [PMT-7](PLAN_MULTITITRE_PERIPHERY.md) ✅ read-path (migration `title_slug` + `CapWorldLeaderboard` + port/repo/service + gating + oracle b) **+ write-path** (`InsertWorldCSRSnapshot`/`AccumulateWorldStats` threadés + cron + CLI `-title`) | `GetCSRWorldLeaderboard(titleSlug,…)` + `leaderboard_title_isolation_test` |
| MT-22 | Cycle de vie titre (`Status` défini, jamais lu/gaté) | major | done | [PMT-8](PLAN_MULTITITRE_PERIPHERY.md) ✅ cf27ff85f | `middleware/require_active_title.go` (gate 503) |
| MT-23 | Registre/ordre de migrations = 1 liste globale Halo lancée sur chaque titre | major | done | [PMT-9](PLAN_MULTITITRE_PERIPHERY.md) ✅ 743f9467c (relocation Phase 1.5 + routage `RunForTitleDB(slug)` + ledger `title_schema_version` ; 2 déviations doc : PK `name`, `canonicalOrder` reste runner) | `migration.TitleMigrationSet` + `RegisterMigrationSet` + oracle b synthetic_title_b |
| MT-05 | Observabilité sans dimension titre (expvar/error/player-API/perf endpoints) | major | done | [PMT-10](PLAN_MULTITITRE_PERIPHERY.md) ✅ (PR-1→4) | seam titré 3 collecteurs + logs `title` + endpoints `?title=` + émission sync |
| MT-26 | Discord : config globale (→PMT-4) + contenu 100% Halo (→PMT-11) | major | partiel | [PMT-11](PLAN_MULTITITRE_PERIPHERY.md) ✅ contenu/outcomes b571f1df5 (seam `NotifyLabels`) ; [PMT-4](PLAN_MULTITITRE_PERIPHERY.md) config = gap | `notify.NotifyLabels`+`OutcomeSource` ; footer/backfill restent Halo |
| MT-09 | Cutoffs factory `DefaultSlug` (`dataAdapterForPDB` → nil pour non-Halo) | major | done | [PMT-12](PLAN_MULTITITRE_PERIPHERY.md) ✅ 23d088103 | factory player-scoped par titre (`ServiceRegistry.playerDataBuilders`), allowlist archlint vide |
| MT-21 | Aucun validateur boot « TOML requis présents » par titre enregistré | minor | done | [PMT-12](PLAN_MULTITITRE_PERIPHERY.md) ✅ dcfc01c31 | `internal/games/mappings/validate.go` (capability-driven) |
| MT-12 | Front : constantes littérales `TITLE_SLUG='halo_infinite'` + lint absent | major | partiel | [EXT-5](PLAN_MULTITITRE_PERIPHERY.md)/[PMT-12](PLAN_MULTITITRE_PERIPHERY.md) | `apps/web/src/lib/staticAssets.ts:26` |
| MT-13 | Front : tables Halo client-side (teamNames, tier grids, badge `HINF`, 225 HP) | major | partiel | [EXT-5](PLAN_MULTITITRE_PERIPHERY.md) | `apps/web/src/lib/halo/teamNames.ts:17-27` |
| MT-07 | Modèle rangs/tiers carrière (272 rangs, 6 tiers, P80) non externalisé | major | différé-latent | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) — cosmétique/latent, extension Phase 2 ; vraie valeur seulement avec un 2e titre à grille divergente (décision 2026-06-17) | `internal/migration/career_rank_data.go` |
| MT-15 | Chaîne LUSR/perf + poids : `sync/skill_config.go` importe `halo_infinite` | major | différé-latent | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) — idem ; touche la chaîne LUSR (irréversible #1) → ne pas churner sans 2e titre | `internal/sync/skill_config.go` |
| MT-14 | Extraction JSON participant + assists + persist mono-DB | major | différé-latent | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) — idem (Phase 2) | `internal/sync/transforms.go` |
| MT-19 | Progression handlers `defaultProgressionTitleSlug` + PrestigeBundle 1 DB | major | différé-latent | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) — idem (Phase 2) | `internal/api/prestige_setup.go` |
| MT-10 | CLI/ops hors backup-restore-diagnose (healthcheck, gate, cmd/* sans `--title`) | major | différé-latent | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) — itérer `registry.All()` CHANGE le format de sortie diagnostic pour 0 valeur single-titre ; à faire au onboarding 2e titre | `internal/ops/healthcheck.go` + `validation/gate.go` |
| MT-16 | Tables metadata globales sans `title_id` (décision : path-isolation suffit ?) | major | done | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (stratégie nuancée : title_id sur caches d'assets génériques map_images/waypoint, path-isolation sur le canonique — ADR 0008) | commentaire step `add_map_images_registry` |
| MT-17 | `notification_preferences` sans dimension titre (décision de scoping) | minor | done | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (per-titre par isolation de chemin, pas de colonne title_id) | thought_log 2026-06-17 |
| MT-18 | Seed démo : préfixe `'Halo Infinite'` + résolution xuid halo-first | minor | différé-latent | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) — code seed sensible (regen destructif) + fallback déjà présent ; byte-identique Halo, à faire au onboarding 2e titre | `internal/ops/seed_demo.go:400` |
| MT-24 | Politique backup restic globale (1 repo/rétention, pas par titre) | minor | done | [PMT-13](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (rétention globale intentionnelle, découverte déjà per-titre) + test `TestDiscoverLevelUpDBs_MultiTitle` | `internal/ops/backup_service.go` |
| MT-25 | Cache HTTP / rate-limit sans dimension titre (bénin) | info | done | [PMT-13](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (rate-limit IP-invariant ; ETag par corps → title-correct) | `rate_limit.go` + `http_cache.go` |
| MT-20 | Adapter Halo `TitleSlug()` → `DefaultSlug` (auto-identité correcte) | info | done | [PMT-13](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (self-identity, gating via registre) | `internal/games/halo_infinite/adapter_data.go:66` |

---

## Carte des phases (nouvelles + extensions)

Specs complètes dans [PLAN_MULTITITRE_PERIPHERY.md](PLAN_MULTITITRE_PERIPHERY.md). Méthode `expand → parity-gate → contract` + oracle double (parité Halo golden + `synthetic_test_title`) pour CHAQUE phase.

| Phase | Titre | Axes | Sév. | Prérequis |
|---|---|---|:-:|---|
| **PMT-1** | Ingestion title-aware (hosts) | MT-01 | blocker | — (racine) |
| **PMT-2** | Acquisition auth par titre | MT-02 | blocker | — (racine) |
| **PMT-3** | Scheduler/sync titleSlug threading | MT-11 | blocker | PMT-1, PMT-2 |
| **PMT-4** | Settings par titre (overlay) + config Discord | MT-04, MT-26(cfg) | major | PMT-3 |
| **PMT-5** | Canonicalisation Outcome | MT-06 | major | — |
| **PMT-6** | Achievements par titre | MT-08 | major | PMT-1/2 (sync data) |
| **PMT-7** | World-stats / leaderboard par titre | MT-03 | major | PMT-3 |
| **PMT-8** | Cycle de vie du titre (Status) | MT-22 | major | — |
| **PMT-9** | Registre migrations + schema_version par titre | MT-23 | major | PMT-3 |
| **PMT-10** | Observabilité — dimension titre | MT-05 | major | — |
| **PMT-11** | Discord notifications (contenu) | MT-26(contenu) | major | — |
| **PMT-12** | Garde-fous & validateurs | MT-21, MT-09, MT-12(lint) | major | **après** PMT-3 (cutoffs) |
| **PMT-13** | Mineurs & bénins (décision documentée) | MT-24, MT-25, MT-20 | minor | — |
| **PMT-14** | Admin : gestion des titres (+ réhabilitation Lab) | MT-22 (+1.7a/b, 1.8) | major | PMT-3, 1.7a/b✅, PMT-4/8 |
| **EXT-1.5** | Extension Phase 1.5 (metadata/ops/seed/notif) | MT-16, MT-10, MT-18, MT-17 | major | PMT-3 |
| **EXT-2** | Extension Phase 2 (career/LUSR/extraction/prestige) | MT-07, MT-15, MT-14, MT-19 | major | PMT-3 |
| **EXT-5** | Extension Phase 5 (slug constants + tables Halo front) | MT-12, MT-13 | major | Phase 5 |

> **Statut clôture (2026-06-15, branche `feat/multititre-peripherie`)** : **PMT-8** ✅ (gate `RequireActiveTitle` 503 + seam restauré + oracle synthetic_b). **PMT-12** ✅ — MT-21 ✅ (validateur boot capability-driven + golden HI + oracle synthetic_b), **MT-09** ✅ (cutoffs → factory player-scoped par titre, allowlist archlint VIDE, parité Halo career.progression supported), **lint MT-12** ✅ (règle eslint `no-title-slug-literal`, warn). **PMT-14** ✅ — **volet A** ✅ (liste/détail/diagnostic+drift, toml-draft D10, CLI `levelup-titles diagnose`, admin-gating 401/403, page aide 2e titre) ; **volet C** ✅ (Lab monté + fail-closed + test anti-régression, livré 2026-06-14) ; **volet B** ✅ par construction (0 duplication ; atoms feature-local distincts — unifier violerait la modularité). **PMT-5** 🟡 — **Expand** ✅ (seam Outcome int↔canonique + oracle double) ; **Contract** (migration ~20 sites SQL/Go/front) en session dédiée (risque data-path, cf. note de reprise dans le tracker). **PMT-10** ✅ — observabilité dimension titre complète (PR-1 logs `title` + Expand 3 collecteurs + PR-2 endpoints `?title=` + PR-3 émission sync 61/65 + PR-4 LogsDir namespacing ; oracle double parité/routing ; coordinator gate process-wide laissé legacy).

> **Constat Lab (PMT-14 volet C) — RÉSOLU (2026-06-14, vérifié 2026-06-15)** : le Lab était cassé (backend complet mais non monté → `/lab/*` 404). Désormais **monté** (`server.go:680-683`), `requireAccess` **fail-closed**, test anti-régression `lab_routes_mounted_test.go` (chi.Walk), Contracts marqué « à retirer ». Modèle d'accès Lab = capability `can_manage_instance` (≠ admin role), documenté au montage. (L'audit 2026-06-15 le croyait absent : il ne lisait que le diff de la branche PMT-14, pas l'état de main.)

### Séquencement (relecteur de cohérence)

- **Racine du DAG = PMT-1 + PMT-2** : sans hosts ni auth title-aware, on ne peut ni fetcher ni authentifier un 2e titre → aucune phase aval n'est exerçable end-to-end (oracle b).
- **PMT-3 avant toute écriture per-title** (EXT-1.5, PMT-4, PMT-7, EXT-2, PMT-9) : sinon un 2e titre écrase les DB Halo (`NewSyncEngine` écrit en `DefaultSlug`, dette `auto_sync.go:838-841`).
- **MT-09 (cutoffs, PMT-12) après PMT-3** : ne pas livrer le validateur/lookup avant que le slug soit réellement threadé.
- **Indépendants** (parité Halo seule) : PMT-5, PMT-8, PMT-10, PMT-11, EXT-5, PMT-13.

### Bloquants pour un 2ᵉ titre réel (récap)

Au-delà du chemin data-path du master, **rien ne peut fonctionner pour un 2ᵉ titre sans** : Phase 1.5 (DDL par titre, le vrai 1er gros morceau) + **PMT-1** (hosts) + **PMT-2** (auth) + **PMT-3** (écriture par titre). Les `major` suivent ; les `minor`/`info` sont des décisions documentées.
