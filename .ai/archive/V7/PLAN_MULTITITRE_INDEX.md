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

## ✅ STATUT CONSOLIDÉ — 2026-06-19 (LIRE CECI D'ABORD — source unique)

> Les tableaux registre A/B ci-dessous restent le détail ; ce bloc est le résumé à jour. En cas de divergence, ce bloc fait foi (les statuts datés des tables ont pu prendre du retard). **Re-vérifié contre le code le 2026-06-19** (audit « que reste-t-il du master plan »).

**Le multi-titre est branchable day-one côté DONNÉES ET NAVIGATION.** Toute l'infra title-agnostic (boot, DB, capabilities, gating data, **switcher UI câblé**) est livrée et prouvée. Ajouter un 2ᵉ titre = déposer un dossier `config/titles/<slug>/` (title.toml + mappings + constants + auth) → découvert au boot, exposé dans `bootstrap.available_titles`, DB provisionnées, listé dans le sélecteur. Reste, pour un VRAI 2ᵉ titre, **un seul vrai gros morceau** : écrire son **adapter data + ses vraies sources** (intrinsèquement title-spécifique, hors infra — l'adapter `synthetic_title_b` renvoie `ErrCapabilityNotSupported` sur tous ses `Load*`). C'est le travail jour-1 (handoff [HANDOFF_HALO5_EXPERIMENTAL.md](HANDOFF_HALO5_EXPERIMENTAL.md)).

**✅ FAIT** — PMT-1→14, EXT-1.5, EXT-2, Phases 1.5 (+ **registre piloté par config + provisioning boot**), 1.6, 1.7a/b, **1.8 (Lab monté + testé)**, **1.9 routing/détection**, 2, 3a, **3b (166 routes Huma)**, **5 (gating front ~95% — Explorer + Timeseries rang gatés le 2026-06-18, voir axe UI)**. Livrés à 100% le 2026-06-18 : **PMT-4** (settings overlay GET/PATCH), **PMT-5** (outcomes SQL — tous repos), **PMT-11** (Discord title-aware). **Squelette `synthetic_title_b` complet** (coming_soon, niveau Halo).

**🟡 RESTE — 1 item externe (raison dure, pas du soft-deferral)** :
1. **PMT-2 — pool d'auth par titre (jambe finale)** : le watcher **détecte + route** un 2ᵉ titre, mais le pool partagé tient des tokens Spartan d'**audience Halo** ; le keying par titre est **inexerçable sans un vrai 2ᵉ titre** au périmètre auth distinct. Le reste de PMT-2 (descripteur `AuthDescriptor` + loader) est fait.

> **MàJ 2026-06-19** : l'item « Huma — génération auto openapi.yaml / bascule types.ts » est **résolu/clôturé**. La migration JSON Huma est à 100% (garde-fou `TestNoJSONRouteBypassesHuma`) ; l'`openapi.yaml` a été **complété** (MISSING 332→0, drift-detector) ; `types.ts` est **migré** vers `generated.ts` (228 shims + ratchet anti-doublon, Phase D). La **génération** auto reste DESCOPÉE (bloquant archi, valeur faible) → YAML manuel gardé honnête par le drift-detector. cf. [PLAN_WEB_API_TYPES_MIGRATION](PLAN_WEB_API_TYPES_MIGRATION.md) + thought_log 2026-06-18/19.

**🖥️ RESTE — UI multi-titre (audit 2026-06-19 vérifié + adversarial)** :
- **Sélecteur de titre (switcher) — ✅ CÂBLÉ** (PMT-8 / MT-22). `TitleSwitcher.tsx` est rendu dans `NavL1.tsx:197` (commit `c41e79a13`). Branché sur `buildTitleSwitcherEntries` (`coming_soon`/`archived`) + `switchTitle` + `setApiTitleSlug` → header `X-LevelUp-Title`. NO-OP <2 titres ; `coming_soon` affiché « bientôt disponible ». **L'ancien statut « NON câblé / code mort » de ce doc était PÉRIMÉ (corrigé 2026-06-19).**
- **Gating capability — ~95%** 🟢. Explorer (bloc CSR saison + filtre skill-tier) et Timeseries (charts rang `ranked||lusr`) gatés. Reste **cosmétique, non bloquant** : vocabulaire Halo en dur non gaté (admin Sync « API Halo »/« Watcher présence Xbox », Lab libellés « Waypoint », HomeHeroBanner images, `csrRankImageURL` motif `HINF-CSR`) → fallback Halo pour un 2e titre, **à sortir vers adapters par-titre quand un vrai 2e titre arrive** (YAGNI sans 2e titre : abstraction sans consommateur).
- **Lab** 🟢 : title-aware (header + `ctxkeys.TitleSlug`) + gaté admin (`can_manage_instance`). Mineur : queryKeys sans slug (fuite cache transitoire au switch) + vocab Waypoint Halo.
- **Section admin Titres** 🟢 : complète en lecture (liste + statut lifecycle + capabilities + feature-matrix + diagnostic DB). Onboarding = CLI (read-only assumé). Les autres pages admin ne sont pas title-scopées (acceptable mono-titre-de-fait).

**📦 FEATURES DISTINCTES (pas de l'infra multi-titre — features à part entière)** :
- **Arcs Prestige cross-titre** ✅ **LIVRÉ 2026-06-18** — [PLAN_CROSS_TITLE_ARCS_BACKEND.md](PLAN_CROSS_TITLE_ARCS_BACKEND.md). Backend-ready (table `arc_titles` + migration backfill idempotente + `ArcTitlesRepo` + point d'extension `creditTitlesFor`). Mono-titre observable inchangé. Hors périmètre (garde-fou plan) : arc réellement multi-titres + répartition PP = décision produit+UX pending 2e titre réel.
- **Famille d'armes canonique** 🟡 — [PLAN_WEAPON_FAMILY_CANONICAL.md](PLAN_WEAPON_FAMILY_CANONICAL.md). Partiel.
- **Phase 1 — FieldKey / `fields.toml`** 🟡 — partiel (stats canoniques).

**⛔ WON'T-DO (Phase 4 — clôturée par SUPPRESSION le 2026-06-18)** :
- **Phase 4 — Sync flags FieldKey** : le *refactor FieldKey* était **NO-GO** (fausse affordance, zéro consommateur prod). **Résolu non par câblage mais par SUPPRESSION 2026-06-18** : les 12 flags granulaires (`TeamMMR`/`Damage`/`HeadshotKills`…) + 5 groupes alias (`MMR`/`Expected`/`Combat`/`KillsDetail`/`CoreStats`) + leurs `Force*` + 34 flags CLI ont été retirés de `SyncScope`/`NewBackfillFlagSet`. Byte-identique fetch prouvé (`AllData` activait déjà `Accuracy`/`Shots`/`EnemyMMR` en direct ; flags supprimés lus par aucun chemin de fetch). Aucune capacité réelle perdue (flags directs `--accuracy`/`--shots`/`--enemy-mmr` conservés). cf. thought_log 2026-06-18.

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
| DDL/schéma par titre | Phase 1.5 | master (+ [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md)) | ✅ + **registre piloté par config + provisioning boot 2026-06-18** (worktree `feat/multititre-peripherie`) : `title.toml` manifest (`config_loader.go`) + `LoadTitlesIntoRegistry`/`NewRegistryFromConfig` (déposer un dossier config/titles/<slug>/ → titre découvert, zéro recompil) ; `SetDefaultRegistry` registre PARTAGÉ posé au boot → server/bootstrap-switcher/session-context/scheduler sur `DefaultRegistry()` ; boot `provisionAdditionalActiveTitles` (loop `reg.Active()`, skip `IsDefault`) crée+migre `data/titles/<slug>/` via `RunForTitleDB`, PvE gaté `CapFirefight`. Byte-identique mono-titre, archlint vert. cf. thought_log 2026-06-18 |
| Pool tokens clé `(titleSlug,gamertag)` | Phase 1.6 | master | ✅ |
| `capabilities.toml` + loader + endpoint | Phase 1.7a | master | ✅ |
| Feature-matrix 3 états + cascade | Phase 1.7b | master | ✅ |
| Outillage diagnostic Lab | Phase 1.8 | master | ✅ (re-vérif 2026-06-18 : Lab MONTÉ + testé — handler 3 routes `/lab/*` via Huma, service fail-closed `can_manage_instance`, provider title-aware, test anti-régression `lab_routes_mounted_test`. L'ancien ⬜ était périmé.) |
| Watcher / présence — routing par titre | Phase 1.9 | master | ⬜ |
| Services title-agnostic (canonical) | Phase 2 | master (+ [EXT-2](PLAN_MULTITITRE_PERIPHERY.md)) | ✅ |
| Cleanup DTO (`*Raw` hors domain) | Phase 3a | master | ✅ (2026-06-17 : `*Raw`=NO-OP cycle ; stats nullable `*T omitempty` ; **`HasExpectedData` retiré** — contrat coordonné Go+openapi+front, dérivation équivalente `expected_kills!=null||...`, regen gen/types.gen.go + generated.ts) |
| Migration Huma (~79 fichiers / ~139 routes, pas 113) | Phase 3b | master | ✅ **CONTRAT JSON MIGRÉ 2026-06-18 — 166 routes (12 workflows ultracode).** Inventaire (13 agents) : 169 routes. (a) **socle `humacore`** (sans cycle, partagé api+handlers : factory + erreur writeError + format byte-identique writeJSON NaN-safe) ; (b) **sous-routeur imbriqué PROUVÉ** (`TestHumaNestedSubrouterProbe` : middleware ownership/title hérité + path param parent `{player_slug}` lu, pas de bypass) → débloque ~70 routes player-scoped ; (c) **TOUTES les shapes prouvées+committées** : GET no-param/path/query, POST-body→200, body optionnel, **204** (Output `Status int`), PATCH/DELETE, **erreur+header** (`ErrorWithHeaders` 503 Retry-After), path param custom-parsé (string→400). **166 routes migrées** : tout le player-group (+prestige×26) + infra (bootstrap/players/asset-metadata/lab/help/multi-title/admin/health/session/diag/auth) + admin-ops (settings/setup/favorite/watcher/backfill/monitoring/data-quality/actions/logs) + sync×3 + media-feed. Goldens (recâblés vers Mount) + contract_test verts à chaque commit ; seuls échecs = device-flow E2E pré-existants (réseau). Écart délibéré unique : body malformé → 422 (routes sensibles en RawBody = 400 exact). **Reste sur chi par CONCEPTION (~20, non-JSON)** : multipart (media/upload, import/openspartan), binaire (media/files, images assets), redirects OAuth, device-flow, CSV export, home cached/ETag, battlepass/challenges NoStore, SPA catch-all. **Bascule openapi.yaml généré = DESCOPÉE** (bloquant archi investigué 2026-06-18 : Huma enregistre des chemins RELATIFS + ~133 instances + Registry/collisions + ~20 routes non-JSON hors Huma → spec auto incorrect sans ré-archi single-API qui sacrifierait l'héritage middleware ; valeur faible car YAML manuel gardé par contract_test + generated.ts 0 importeur). Garder le YAML manuel comme source de vérité ; cf. thought_log 2026-06-18 + PLAN_TITLE_AGNOSTIC_TRACKER |
| Sync flags FieldKey-based | Phase 4 | master | ⛔ **NO-GO (re-vérif 2026-06-17)** : les 12 champs stats `SyncScope` + `NewBackfillFlagSet` ont **ZÉRO consommateur prod** (mort-né) → refactor = churn byte-identique sur code mort. Reporté tant qu'aucun consommateur réel (CLI revival / 2e titre) |
| Frontend canonical-aware + `<FeatureGate>` | Phase 5 | master (+ [EXT-5](PLAN_MULTITITRE_PERIPHERY.md)) | ✅ **GATING CAPABILITY LIVRÉ 2026-06-18** (le « canonical-aware labels » restant était déjà en place via `useFieldLabel` → phase complète) (worktree `feat/multititre-peripherie`, 2 workflows ultracode : map 99→reconcile 35 vérifiés). Socle `apps/web/src/lib/capabilities/` (`useCapability`/`useTitleCapabilities`/`hasCapabilityIn` fail-open + `<FeatureGate>` + `<RouteCapabilityGate>` + `<FeatureUnavailable>`). Câblage **NO-OP halo_infinite** (11 caps) : nav (NavL1 sections+onglets, NavL2), route-page (media/career_/citations/season-pass/palmares/ascension), section/intra-composant (achievements, lusr-evolution, colonnes CSR/LUSR ranking, médias match-view, 4 sites engagement, home playlists+skill-peaks). **Ascension=lusr** (résout désaccord 3-agents). **firefight/forge=0** (self-hide/inexistant). typecheck+eslint ok, vitest 1902 pass. Reste : canonical-aware labels = déjà en place (`useFieldLabel`). cf. thought_log 2026-06-18 |
| Arcs Prestige cross-titre (table `arc_titles`) | — | [cross-title-arcs](PLAN_CROSS_TITLE_ARCS_BACKEND.md) | ⬜ |
| Famille d'armes canonique | — | [weapon-family](PLAN_WEAPON_FAMILY_CANONICAL.md) | 🟡 |

---

## Registre B — Axes OUBLIÉS (audit 2026-06-13/14)

> Statut `partiel` = une phase master touche le mécanisme mais laisse un reste concret (colonne **Reste**). Tous ⚠ à re-vérifier.

| ID | Axe | Sév. | Statut | Phase | Évidence (1 pointeur, ⚠ re-vérif) |
|---|---|:-:|:-:|---|---|
| MT-01 | Hosts d'ingestion API (stats/economy/gamecms/skill/UGC/discovery) en const | blocker | ✅ Exit Gate | [PMT-1](PLAN_MULTITITRE_PERIPHERY.md) — Expand bff9a1df3 + 6 axes Contract → cf2afefe2 (privacy `:443` + leaderboard web-scrape déférés/documentés) | `games.EndpointResolver` + `[endpoints]` constants.toml |
| MT-02 | Acquisition auth (XSTS audience, SISU titleID 144209987, clearance `titles/hi`, scopes) | blocker | done | [PMT-2](PLAN_MULTITITRE_PERIPHERY.md) ✅ 4 legs (XSTS/Spartan/Clearance/SISU/scopes) — leg 5 (store namespacé titre 873637195) **ANNULÉE (2026-06-25)** : tokens account-level partagés inter-titres ; store global `data/auth/watcher_tokens/` rétabli — cf. branche `fix/auth-tokens-title-agnostic` | `AuthDescriptor` ; ~~`PathResolver.WatcherTokensDirFor(slug)` + `MigrateWatcherTokens`~~ **supprimés (revert 2026-06-25)** — `WatcherTokensDir()` global |
| MT-11 | Scheduler/auto-sync : `NewSyncEngine` écrit en `DefaultSlug` (slug résolu puis jeté) | blocker | ✅ Exit Gate | [PMT-3](PLAN_MULTITITRE_PERIPHERY.md) — PR-1→4 → a934cde77 (V1+V2 écrivent per-titre, gate clé composite ; watcher per-titre documenté/inexerçable) | `NewSyncEngineForTitle` + ctxkeys carrier + `gateKey` |
| MT-04 | `app_settings.json` global (CurrentCSRSeasonID, friends, sessions, toggles) | major | **partiel** (quasi-done) ⚠ re-vérif 2026-06-17 | [PMT-4](PLAN_MULTITITRE_PERIPHERY.md) — PR-0 primitive (`22649f23e`) + PR-1 CSR season (`afef1195f`) + PR-2 Discord (`83953208f`) + PR-3a CSR UI (`ae7a1627b`) + **PR-3b sessions/coach per-titre** ✅ (`sessionComputeOptionsFor` par `p.TitleSlug` ; `readCoachProactiveMode(reg, pdb2.TitleSlug)` + progression threadée ; bug plan slug-joueur-vs-titre rattrapé par vérif adversariale). **FriendGamertags = cross_title_global ACTÉ** (footgun authz famille, 8 sites laissés sur `Load()`, décision commentée server.go). **Reste** : PR-3c (GET /settings title-aware ShowProgression/OutcomeExclude* — **différée**, no-op tant que pas de middleware titre sur `/settings` + flags front-only) | `settings.Store.ResolveForTitle` + `sessionComputeOptionsFor` |
| MT-06 | Enum Outcome `2/3/1/4` baké en SQL + citations + front | major | done (backend) | [PMT-5](PLAN_MULTITITRE_PERIPHERY.md) — Expand ✅ ; Contract backend ✅ **allowlist ratchet VIDE** : tous les sites Go → `domain.Outcome*`, dernier littéral SQL (`assists_model.go` filtre DNF) → paramètre lié `domain.OutcomeDNF` (`outcome != ?`). Le ratchet `no_raw_outcome_literal_test.go` interdit désormais toute régression sans exception. (Front TS = surface distincte, hors ratchet Go.) | seam `mappings.OutcomeMappingSet` + ratchet archlint (allowlist vide) |
| MT-08 | `XboxTitleIDFor` + achievements (`WHERE title_id='halo_infinite'`) | major | done | [PMT-6](PLAN_MULTITITRE_PERIPHERY.md) ✅ PR1+PR2 e7f06fe71 (lecture `title_id=?` + XboxTitleIDFor registry-driven) + PR3 flag CLI `--title` (NewSyncEngineForTitle) → MT-08 100% | `GetAchievementDefinitions(ctx, slug)` + `(r *Registry).XboxTitleIDFor` + `cmd_sync_achievements --title` |
| MT-03 | World-stats / leaderboard (repos, enricher, CLIs, `DEFAULT 'halo_infinite'`) | major | done | [PMT-7](PLAN_MULTITITRE_PERIPHERY.md) ✅ read-path (migration `title_slug` + `CapWorldLeaderboard` + port/repo/service + gating + oracle b) **+ write-path** (`InsertWorldCSRSnapshot`/`AccumulateWorldStats` threadés + cron + CLI `-title`) | `GetCSRWorldLeaderboard(titleSlug,…)` + `leaderboard_title_isolation_test` |
| MT-22 | Cycle de vie titre (`Status` défini, jamais lu/gaté) | major | done | [PMT-8](PLAN_MULTITITRE_PERIPHERY.md) ✅ cf27ff85f | `middleware/require_active_title.go` (gate 503) |
| MT-23 | Registre/ordre de migrations = 1 liste globale Halo lancée sur chaque titre | major | done | [PMT-9](PLAN_MULTITITRE_PERIPHERY.md) ✅ 743f9467c (relocation Phase 1.5 + routage `RunForTitleDB(slug)` + ledger `title_schema_version` ; 2 déviations doc : PK `name`, `canonicalOrder` reste runner) | `migration.TitleMigrationSet` + `RegisterMigrationSet` + oracle b synthetic_title_b |
| MT-05 | Observabilité sans dimension titre (expvar/error/player-API/perf endpoints) | major | done | [PMT-10](PLAN_MULTITITRE_PERIPHERY.md) ✅ (PR-1→4) | seam titré 3 collecteurs + logs `title` + endpoints `?title=` + émission sync |
| MT-26 | Discord : config globale (→PMT-4) + contenu 100% Halo (→PMT-11) | major | partiel | [PMT-11](PLAN_MULTITITRE_PERIPHERY.md) ✅ contenu/outcomes (seam `NotifyLabels`) + footer UNIFIÉ sur le même seam (`NotifyLabels.TitleName()` → `discordFooterText(labels)` : outcomes ET footer suivent le titre via `cfg.Labels`) ; [PMT-4](PLAN_MULTITITRE_PERIPHERY.md) config = gap | `notify.NotifyLabels`(+`TitleName`)+`LabelsFor(src,titleName)` ; reste = activation 2e-titre (poser `cfg.Labels` au trigger) + libellés backfill |
| MT-09 | Cutoffs factory `DefaultSlug` (`dataAdapterForPDB` → nil pour non-Halo) | major | done | [PMT-12](PLAN_MULTITITRE_PERIPHERY.md) ✅ 23d088103 | factory player-scoped par titre (`ServiceRegistry.playerDataBuilders`), allowlist archlint vide |
| MT-21 | Aucun validateur boot « TOML requis présents » par titre enregistré | minor | done | [PMT-12](PLAN_MULTITITRE_PERIPHERY.md) ✅ dcfc01c31 | `internal/games/mappings/validate.go` (capability-driven) |
| MT-12 | Front : constantes littérales `TITLE_SLUG='halo_infinite'` + lint absent | major | **done** (EXT-5 2026-06-17) | [EXT-5](PLAN_MULTITITRE_PERIPHERY.md)/[PMT-12](PLAN_MULTITITRE_PERIPHERY.md) ✅ : 6 littéraux `features/` migrés vers `useAppShellStore((s)=>s.currentTitleSlug)` / `DEFAULT_TITLE_SLUG` ; règle eslint `@levelup/no-title-slug-literal` passée **warn→error** (0 erreur résiduelle). Défauts `lib/`+`stores/` (client.ts, appShellStore.ts) = fallbacks légitimes hors-scope, gardés | `useAppShellStore` + `DEFAULT_TITLE_SLUG` (staticAssets.ts:26) |
| MT-13 | Front : tables Halo client-side (teamNames, tier grids, badge `HINF`, 225 HP) | major | partiel (**décision EXT-5**) | [EXT-5](PLAN_MULTITITRE_PERIPHERY.md) — **tables GARDÉES** (donnée structurelle Halo, exception documentée) : externaliser exigerait un NOUVEAU seam backend (kind `team`/`skill_tier_grid`/`gameplay_constant` ou champ `badge_image_url` sur l'entrée leaderboard — inexistant) = sur-engineering mono-titre. HALO_OUTLINE_COLORS + perf-tier = légitimes (pas MT-13). À externaliser au 2e titre via les seams existants (`useAssetLabel`/asset-URL). **⚠ MàJ 2026-06-20 — BACKEND sous-scopé par cette ligne** : le `225` (PV effectifs pour tuer) est AUSSI câblé dans le COMPUTE (`combat_yield.go`, `squad_breakdown_canonical.go`, SQL `post_sync_progression_queries.go`) + P80 calibrés Infinite + copy d'aide « convention Halo Infinite » — bien plus impactant que les tables front (c'est la donnée servie, pas que l'affichage). Trigger « 2e titre » ATTEINT (Halo 5, baseline 115 = bouclier 70 + armure 45, à valider data). → **plan dédié [PLAN_DAMAGE_MODEL_PER_TITLE.md](PLAN_DAMAGE_MODEL_PER_TITLE.md)**, à séquencer avec activation 1b | `apps/web/src/lib/halo/teamNames.ts:17-27` + `apps/go-api/internal/analysis/combat_yield.go:59-67` |
| MT-07 | Modèle rangs/tiers carrière (272 rangs, 6 tiers, P80) non externalisé | major | done | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) ✅ générateur déplacé dans le package de titre (`halomigrations.CareerRankTranslations`), `internal/migration` ne garde que le struct + seam `SetCareerRankTranslationsProvider` ; golden byte-identique (544 rows, FNV) + guard archlint no-title-import ; câblé server/levelup/seed-cli | `internal/games/halo_infinite/migrations/career_rank_data.go` |
| MT-15 | Chaîne LUSR/perf + poids : `sync/skill_config.go` importe `halo_infinite` | major | done | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) ✅ classification pair_name→chaîne LUSR déplacée dans `halo_infinite/skillchain` ; seam `SetLUSRChainClassifier` fail-loud (panic si non câblé, sortie persistée match_skill_rank) ; golden 48-cas + cross-check + sweep `go test ./...` ; 5 cmd LUSR + 3 TestMain câblés ; skill_config.go n'importe plus halo_infinite/analysis | `internal/games/halo_infinite/skillchain/classify.go` |
| MT-14 | Extraction JSON participant + assists + persist mono-DB | major | done | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) ✅ transforms.go/_helpers.go découplés de halo_infinite (mode_category = constantes locales, Option B ; colonne write-only opaque, audit workflow) ; golden TestDetermineModeCategoryTable + guard import | `internal/sync/{transforms.go,mode_category.go}` |
| MT-19 | Progression handlers `defaultProgressionTitleSlug` + PrestigeBundle 1 DB | major | partiel (de-magic) | [EXT-2](PLAN_MULTITITRE_PERIPHERY.md) — `defaultProgressionTitleSlug()` → `titlePkg.DefaultSlug` ✅. Reste : `PrestigeBundle` (`NewHaloBaselineProvider` + sous-système config Halo arcs/milestones/challenges TOML, derrière `PRESTIGE_ENABLED`) = threading title-awareness d'une FEATURE flaggée (PAS un import halo_infinite à découpler comme MT-07/14/15) ; invariant deadlock post-sync documenté → vraie valeur 2e-titre, Phase 2 | `internal/api/prestige_setup.go` |
| MT-10 | CLI/ops hors backup-restore-diagnose (healthcheck, gate, cmd/* sans `--title`) | major | done | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) ✅ healthcheck + gate itèrent `registry.All()` (diagnostic par titre ; `titleDataChecks` extrait + boucle ; noms préfixés du slug si >1 titre, mono-titre byte-identique) + oracle `TestTitleDataChecks_PerTitleLabeling`. (de-magic slug préalable inclus.) `cmd/diag_*` throwaway hors scope | `internal/ops/healthcheck.go` + `validation/gate.go` |
| MT-16 | Tables metadata globales sans `title_id` (décision : path-isolation suffit ?) | major | done | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (stratégie nuancée : title_id sur caches d'assets génériques map_images/waypoint, path-isolation sur le canonique — ADR 0008) | commentaire step `add_map_images_registry` |
| MT-17 | `notification_preferences` sans dimension titre (décision de scoping) | minor | done | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (per-titre par isolation de chemin, pas de colonne title_id) | thought_log 2026-06-17 |
| MT-18 | Seed démo : préfixe `'Halo Infinite'` + résolution xuid halo-first | minor | partiel (de-magic) | [EXT-1.5](PLAN_MULTITITRE_PERIPHERY.md) — slug `seed_demo.go:400` → `titlePkg.DefaultSlug` (byte-identique) ; le préfixe `haloInfinitePrefix` = nom de fichier capture Xbox réel (donnée title-spécifique légitime, conservé) ; regen destructif reste mono-titre par construction | `internal/ops/seed_demo.go:400` |
| MT-24 | Politique backup restic globale (1 repo/rétention, pas par titre) | minor | done | [PMT-13](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (rétention globale intentionnelle, découverte déjà per-titre) + test `TestDiscoverLevelUpDBs_MultiTitle` | `internal/ops/backup_service.go` |
| MT-25 | Cache HTTP / rate-limit sans dimension titre (bénin) | info | done | [PMT-13](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (rate-limit IP-invariant ; ETag par corps → title-correct) | `rate_limit.go` + `http_cache.go` |
| MT-20 | Adapter Halo `TitleSlug()` → `DefaultSlug` (auto-identité correcte) | info | done | [PMT-13](PLAN_MULTITITRE_PERIPHERY.md) ✅ décision consignée (self-identity, gating via registre) | `internal/games/halo_infinite/adapter_data.go:66` |
| MT-27 | Interface source de données par titre (`TitleSyncRunner` + core HTTP `platform/httpx` partagé + `KnownSet`) — **nouvel axe 2026-07, hors audit initial** | major | ⬜ gap (ADR rédigé) | — (ADR [0031](../../docs/adr/0031-title-data-source-boundary.md)) | dualité d'orchestration V2 (`internal/sync/v2/`, Infinite) vs `internal/games/halo_5/livesync.Runner` (H5) ; duplication retry/backoff/rate-limit ~150 L (`internal/sync/haloclient/halo_client_http.go` + `internal/games/halo_5/client.go`, constantes identiques 4/800ms/10s, `HTTPError` 2x). Amende ADR 0027. Impl = lots futurs (move+httpx AVANT Phase 1.6) |

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
| **EXT-5** | Extension Phase 5 (slug constants + tables Halo front) | MT-12 ✅, MT-13 (décision: gardé) | major | Phase 5 — **MT-12 livré 2026-06-17** (littéraux nettoyés + lint error) ; MT-13 = décision « garder, externaliser au 2e titre » ; `useCapability`/`FeatureGate` différés à Phase 5 |

> **Statut clôture (2026-06-15, branche `feat/multititre-peripherie`)** : **PMT-8** ✅ (gate `RequireActiveTitle` 503 + seam restauré + oracle synthetic_b). **PMT-12** ✅ — MT-21 ✅ (validateur boot capability-driven + golden HI + oracle synthetic_b), **MT-09** ✅ (cutoffs → factory player-scoped par titre, allowlist archlint VIDE, parité Halo career.progression supported), **lint MT-12** ✅ (règle eslint `no-title-slug-literal`, warn). **PMT-14** ✅ — **volet A** ✅ (liste/détail/diagnostic+drift, toml-draft D10, CLI `levelup-titles diagnose`, admin-gating 401/403, page aide 2e titre) ; **volet C** ✅ (Lab monté + fail-closed + test anti-régression, livré 2026-06-14) ; **volet B** ✅ par construction (0 duplication ; atoms feature-local distincts — unifier violerait la modularité). **PMT-5** 🟡 — **Expand** ✅ (seam Outcome int↔canonique + oracle double) ; **Contract** (migration ~20 sites SQL/Go/front) en session dédiée (risque data-path, cf. note de reprise dans le tracker). **PMT-10** ✅ — observabilité dimension titre complète (PR-1 logs `title` + Expand 3 collecteurs + PR-2 endpoints `?title=` + PR-3 émission sync 61/65 + PR-4 LogsDir namespacing ; oracle double parité/routing ; coordinator gate process-wide laissé legacy).

> **Constat Lab (PMT-14 volet C) — RÉSOLU (2026-06-14, vérifié 2026-06-15)** : le Lab était cassé (backend complet mais non monté → `/lab/*` 404). Désormais **monté** (`server.go:680-683`), `requireAccess` **fail-closed**, test anti-régression `lab_routes_mounted_test.go` (chi.Walk), Contracts marqué « à retirer ». Modèle d'accès Lab = capability `can_manage_instance` (≠ admin role), documenté au montage. (L'audit 2026-06-15 le croyait absent : il ne lisait que le diff de la branche PMT-14, pas l'état de main.)

### Séquencement (relecteur de cohérence)

- **Racine du DAG = PMT-1 + PMT-2** : sans hosts ni auth title-aware, on ne peut ni fetcher ni authentifier un 2e titre → aucune phase aval n'est exerçable end-to-end (oracle b).
- **PMT-3 avant toute écriture per-title** (EXT-1.5, PMT-4, PMT-7, EXT-2, PMT-9) : sinon un 2e titre écrase les DB Halo (`NewSyncEngine` écrit en `DefaultSlug`, dette `auto_sync.go:838-841`).
- **MT-09 (cutoffs, PMT-12) après PMT-3** : ne pas livrer le validateur/lookup avant que le slug soit réellement threadé.
- **Indépendants** (parité Halo seule) : PMT-5, PMT-8, PMT-10, PMT-11, EXT-5, PMT-13.

### Bloquants pour un 2ᵉ titre réel (récap)

Au-delà du chemin data-path du master, **rien ne peut fonctionner pour un 2ᵉ titre sans** : Phase 1.5 (DDL par titre, le vrai 1er gros morceau) + **PMT-1** (hosts) + **PMT-2** (auth) + **PMT-3** (écriture par titre). Les `major` suivent ; les `minor`/`info` sont des décisions documentées.
