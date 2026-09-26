# Plan de portage de la page Citations — Python v7/cockpit -> Go + React

> Audit comparatif rigoureux et plan de portage par phases.
> Branche source : `v7/cockpit` (Streamlit/Python).
> Branche cible : `feat/multi-title-adapters-and-mappings` (Go + React + recharts/Plotly).
> Date d'audit : 2026-04-26.

> **Note d'amendement — 2026-04-27** : ce plan est **partiellement supersedé** par
> [`PLAN_META_FOUNDATIONS_GO.md`](./PLAN_META_FOUNDATIONS_GO.md). Avant toute
> implémentation à partir de ce plan, consulter le méta-plan pour les fondations
> communes : `LoadPlayerMatches(filters)` au lieu de SQL ad hoc pour le scope
> `compute_wins_*`, stack chart **ECharts** (le `MedalsDistributionChart`
> server-side Plotly migre côté client en `<Histogram>` ECharts), manifest i18n
> centralisé. Réécriture complète prévue en **Phase 2** du méta-plan.

### Statut des sections de ce plan vis-à-vis du méta-plan

| Section / Phase | Statut | Action |
|---|---|---|
| Phase 0 — Audit / inventaire | À conserver | Contexte historique. |
| Phase 1 — Port `CitationEngine` + 12 `custom_functions` + composites | À conserver | Spécifique Citations, non couvert par fondations. |
| Phase 2 — Backfill `match_citations` (`compute_and_store_for_match`) | À conserver | Spécifique. |
| Phase 2.x — `MedalsDistributionChart` server-side Plotly | Obsolète | Migration côté client en `<Histogram>` ECharts (méta-plan § 5.2). |
| Phase 2.x — Incohérence shape API ↔ shape Frontend | À conserver (fix) | Bug à corriger ; alignement sur DTO consolidé. |
| R03 — `compute_wins_*` regex localisée FR | À refactorer | Utiliser `LoadPlayerMatches(playlistRegex)` partagé (méta-plan § 5.3). |
| R10 — `total_enemy_kills` PvE | À conserver | Spécifique PvE. |
| R14 — `weapon_labels` UBIGINT | À conserver | Spécifique mapping armes. |
| Phase 3 — Anneaux progression CSS / grille catégorisée | À conserver | UI spécifique citations. |
| Backfill flag CLI `--citations` | À conserver | Spécifique. |

---

## 0. Synthèse exécutive

La page Citations en Python v7/cockpit est **deux pages combinées** dans un seul écran :
1. **Commendations Halo 5** (citations dérivées des matchs Infinite via le moteur `CitationEngine`) — anneaux de progression CSS, groupage par catégorie/sous-catégorie, recherche full-text, mastery par tiers, 100+ items affichés en grille de 8 colonnes par catégorie.
2. **Médailles Halo Infinite** — distribution Plotly horizontale des 25 plus fréquentes + grille complète avec icônes locales 8 colonnes, descriptions au survol, deltas si filtre actif.

La version Go actuelle réduit cela à :
- une `CareerCitationsTab` qui consomme `useCitationsPage(playerSlug, …, 'hub-all')` ;
- une `CitationsPage` qui consomme la même hook avec filtres globaux ;
- un endpoint `POST /api/v1/players/{slug}/pages/citations` qui répond avec `{ citations: CitationItem[], categories: string[], total_count }` — **shape qui ne correspond même pas** à ce que le frontend déstructure (`commendations, medals_summary, deltas, distribution_chart`).

**Diagnostic clé** : la page Go mélange l'onglet "Citations" (résumé Carrière) avec une vue "Citations" autonome, mais aucune des deux ne porte les fonctionnalités riches du v7/cockpit. Le moteur `CitationEngine` n'a pas été porté en Go — la table `match_citations` est lue mais jamais alimentée. `mapping_type` est partiel (`medal` uniquement). `tier_targets` est chargé mais jamais parsé. Les citations composites n'existent pas. Les 12 fonctions custom (`compute_bulldozer`, `compute_wins_*`, `compute_annexion_forcee`, `compute_flag_em_down`, `compute_hijack`, `compute_vandalism`, `compute_*_destroyer`) ne sont pas portées. L'incohérence shape-API/shape-Frontend prouve que cette page n'a probablement jamais été testée bout en bout.

**Écart fonctionnel** :

| Bloc Python (v7/cockpit) | Go actuel | État | Priorité |
|---|---|:-:|:-:|
| Section Commendations H5 — grille catégorisée 100+ items | Liste plate sans hiérarchie | absent | P0 |
| Anneaux progression CSS (`os-citation-ring`) avec image + tier + counter + delta | Barre progression linéaire | dégradé | P1 |
| Groupage par `category` puis `subcategory` (Arme→UNSC/Paria/Forerunner/Grenade…) | Aucun | absent | P0 |
| Ordre catégories/sous-catégories i18n (FR/EN) | Tri alphabétique | absent | P1 |
| Recherche full-text sur `name + description + category` | Aucune | absent | P2 |
| Calcul mastery `(level, counter, is_master, ratio)` à partir de `tier_targets` CSV | Aucun (frontend reçoit déjà `mastery_pct`) | partiel — mais le backend ne calcule rien | P0 |
| Citations composites (`composite_children` JSON, mastery comptée par enfants masterisés) | Absent | absent | P1 |
| 12 fonctions custom (`CUSTOM_FUNCTIONS` registry) | Absent | absent | P0 |
| Backfill `match_citations` — `compute_and_store_for_match` | Absent (flag `--citations` câblé mais sans implémentation) | absent | P0 |
| Section Médailles HI — distribution Plotly horizontale top 25 | `distribution_chart` câblé en `null` | absent | P1 |
| Grille médailles 8 colonnes avec icônes PNG locales | Grille 4-8 colonnes basique sans icônes | dégradé | P1 |
| Métriques (citations obtenues, matchs analysés, médailles distinctes/totales) | KPIs partiels (Maîtrise moyenne, Total médailles…) | divergent | P2 |
| Delta filtré vs complet (`is_filtered` → encadré "+N") | Bloc deltas existant côté React | partiel | P3 |
| Shape API ↔ Shape Frontend | **Incohérence majeure** : Go renvoie `{citations, categories}`, React lit `{commendations, medals_summary, deltas, distribution_chart}` | bug | P0 |
| Images commendations (statique `static/commendations/h5g/*.png`) | Handler Go `commendation_handler.go` existe (servi sous `/static/commendations/`) | OK | — |

**Audit exhaustif (§9)** : sur 87 citations total, **80 sont portables à 100%**, 5 ont des lacunes critiques en données/schéma Go :
- **R10** : `player_vs_everything` — colonne `total_enemy_kills` n'existe pas en `pve_match_stats` → à disable ou implémenter.
- **R11-R12** : Ambiguïté award names + localisation playlist names → à clarifier sur corpus réel.
- **R13** : medal_id 9000000001 (Avenger) factice → à disable.
- **R14** : weapon_labels UBIGINT → 25 citations weapon_stat cassées si mapping weapon_id → nom absent → **CRITIQUE**

**Conclusion** : la page Go est aujourd'hui **non fonctionnelle** au-delà de l'affichage d'une liste de commendations issue d'un agrégat partiel. La couche backend doit être réécrite intégralement (moteur de calcul + stockage + agrégation enrichie), puis la couche frontend recâblée sur le nouveau contract. Checklist pré-Phase-2 (§8.3, §9.5) garante 100% de portabilité une fois exécutée.

---

## 1. Cartographie source (v7/cockpit)

### 1.1 Fichiers Python impliqués

```
src/ui/pages/citations.py                     (197 L) ← entry — page complète
src/ui/commendations.py                       (446 L) ← rendu HTML grille H5G + mastery
src/ui/medals.py                              (200+ L) ← grille HI + icônes b64
src/visualization/distributions.py            (374 L) ← plot_medals_distribution
src/data/citation_definitions.py              (76 L)  ← chargement citation_mappings
src/data/medal_definitions.py                          ← noms/desc médailles
src/data/citations_backfill.py                         ← backfill par-match
src/analysis/citations/__init__.py
src/analysis/citations/_data_loader.py                 ← load_match_medals/stats/awards/df/highlight_events/pve/weapon_kills
src/analysis/citations/engine.py              (491 L) ← CitationEngine (compute + aggregate + persist)
src/analysis/citations/composite.py           (57 L)  ← _apply_composite_citations
src/analysis/citations/custom_rules.py        (315 L) ← CUSTOM_FUNCTIONS registry (12 fonctions)
scripts/populate_citation_mappings.py                  ← seed metadata.duckdb
scripts/investigation/diagnose_citations.py
docs/FR/CITATIONS.md                                   ← documentation référence
```

### 1.2 Tables DuckDB

| Table | DB | Rôle |
|---|---|---|
| `citation_mappings` | `metadata.duckdb` | 14+ règles : `citation_name_norm`, `citation_name_display`, `mapping_type` ∈ {`medal`,`stat`,`pve_stat`,`weapon_stat`,`award`,`custom`,`composite`}, `medal_id`, `medal_ids` (CSV), `stat_name`, `award_name`, `award_category`, `custom_function`, `composite_children` (JSON list), `tier_targets` (CSV des paliers), `category`, `subcategory`, `description`, `image_path`, `enabled` |
| `match_citations` | `data/players/{gt}/stats.duckdb` | Stockage par-match : `(match_id, citation_name_norm, value)` PK composé. Marqueur `_processed=1`. |
| `medals_earned` | `shared_matches_v2.duckdb` | `(match_id, xuid, medal_name_id, count)` |
| `medal_definitions` + `medal_translations` | `metadata.duckdb` | Noms médailles BCP-47 |
| `personal_score_awards` | `data/players/{gt}/stats.duckdb` | Awards (`award_name, award_category, award_count, award_score`) — source des `mapping_type='award'` |
| `highlight_events` | `shared_matches_v2.duckdb` | Events filmés `(time_ms, event_type)` — source de `compute_annexion_forcee` |
| `pve_match_stats` | `shared_pve.duckdb` | Stats Firefight — source des `mapping_type='pve_stat'` |
| `v_weapon_kills` | view sur `shared_matches_v2` | Weapon kills par `effective_weapon_id` — source de `mapping_type='weapon_stat'` |

### 1.3 Pipeline complet

```
Sync match → backfill_citations(player) → CitationEngine.compute_and_store_for_match(match_id)
                                              ↓
                                   Lit: medals_earned, match_participants, personal_score_awards,
                                        highlight_events, pve_match_stats, v_weapon_kills
                                              ↓
                                   Pour chaque mapping enabled:
                                     dispatch via mapping_type
                                       ↓
                                       INSERT OR REPLACE INTO match_citations VALUES (?, norm, value)
                                              ↓
                                   Mark _processed=1
                                              ↓
Page UI → CitationEngine.aggregate_for_display(match_ids=None|filtered)
            └─ SELECT citation_name_norm, SUM(value) FROM match_citations [WHERE match_id IN ?] GROUP BY 1
            └─ _apply_composite_citations(result, mappings)  ← ajoute les composites
              ↓
          Render section Commendations H5 (grille catégorisée + mastery)
          Render section Médailles HI (Plotly + grille icônes)
```

### 1.4 Logique de mastery

`_compute_mastery_display(current_count, tiers)` retourne `(label, counter, is_master, ratio)` :

- `tiers = sorted(parse_csv(tier_targets))` — ex : `"10,20,30,50,100"` → `[10,20,30,50,100]`.
- Si `current_count >= tiers[-1]` → **Maître** ; ratio = 1.0.
- Sinon trouve le palier suivant non atteint, ratio = `(current - prev) / (next - prev)`, label = "Niveau N+1" où N = nb de tiers déjà franchis.
- Pour les composites : `composite_total = nb d'enfants activés` et progression directe `current/total`.

### 1.5 Logique custom functions (registre `CUSTOM_FUNCTIONS`)

Toutes prennent `(df=None, awards=None, highlight_events=None)` et retournent un `int` :

| Fonction | Logique synthétique |
|---|---|
| `compute_bulldozer` | KDA > 8 sur Slayer/Assassin (hors Firefight/BTB) — DataFrame match |
| `compute_wins_ctf` | `outcome=2` ET playlist matche `ctf|capture.*drapeau|drapeau.*neutre|neutral.*flag` |
| `compute_wins_firefight` | `outcome=2` ET playlist matche `firefight\|baptême\|bapteme` |
| `compute_wins_slayer` | `outcome=2` ET playlist matche `slayer\|assassin` |
| `compute_wins_strongholds` | `outcome=2` ET playlist matche `stronghold\|bases` |
| `compute_annexion_forcee` | Walk de `highlight_events` : streaks de 3 events `mode` sans `death` entre. Fallback : `awards["zone_captured"] // 3`. |
| `compute_flag_em_down` | `awards["runner_stopped"] + ["Porteur arrêté"] + ["Flag Carrier Kill"] + ["Flag Carrier Killed"]` |
| `compute_hijack` | Awards `startswith("hijacked_")` ou `hijack/skyjack` dans le nom |
| `compute_vandalism` | Awards `startswith("destroyed_")` ou `destroyed/destruction` dans le nom |
| `compute_wraith_destroyer` | `awards["destroyed_wraith"] + legacy fallbacks` |
| `compute_mongoose_destroyer` | `awards["destroyed_mongoose"] + legacy fallbacks` |
| `compute_warthog_destroyer` | `destroyed_warthog + destroyed_rocket_warthog + legacy fallbacks` |

### 1.6 Logique composite (`_apply_composite_citations`)

Pour chaque `mapping_type='composite'` :
- Parser `composite_children` (JSON list de `citation_name_norm`).
- Pour chaque enfant :
  - Si `tier_targets` vide → masterisé si `count > 0` ;
  - Sinon → masterisé si `count >= max(tiers)`.
- `result[norm] = nb_enfants_masterisés` si `> 0`.

### 1.7 Données loaders (`_data_loader.CitationDataLoaderMixin`)

| Méthode | Source | Retour |
|---|---|---|
| `load_match_medals(match_id)` | `shared.medals_earned` filtré xuid | `dict[medal_id, count]` |
| `load_match_stats(match_id)` | `shared.match_participants ⨝ match_registry` filtré xuid | `dict[col_name, value]` |
| `load_match_awards(match_id)` | `personal_score_awards` filtré xuid+match | `dict[award_name, sum(award_count)]` |
| `load_match_df(match_id)` | `shared.match_participants ⨝ match_registry` filtré xuid | `pl.DataFrame` 1 ligne enrichie playlist/variant |
| `load_match_highlight_events(match_id)` | `shared.highlight_events` filtré xuid+match, ORDER BY time_ms | `list[(time_ms, event_type)]` |
| `load_match_pve_stats(match_id)` | `shared_pve.pve_match_stats` filtré xuid+match | `dict[col, value]` |
| `load_match_weapon_kills(match_id)` | `shared.v_weapon_kills` filtré xuid+match GROUP BY effective_weapon_id | `dict[weapon_id, kills]` |

### 1.8 Rendu UI Streamlit (`citations.py`)

Structure verticale (~200 L) :
1. Garde sur DataFrame vide.
2. Calcul `is_filtered` (longueur dff vs df_full).
3. **Bloc 1 — Commendations H5** :
   - 3 metrics : `cit_obtained` (total), `cit_matches_analyzed`, vide.
   - `render_h5g_commendations_section()` (446 L) :
     - Charge `_load_citations_from_db()` (cache 300s).
     - Engine.aggregate_for_display(full + filtered).
     - Pour chaque cit : compute progress (composite ou tiers), construit item `{name, norm, description, category, subcategory, image_path, tiers, composite_total}`.
     - Filtre texte (`st.text_input` sur name/desc/category).
     - Group by `category` puis `subcategory` selon ordre `_CATEGORY_ORDER_BY_LANG[lang]`.
     - Render header `<h2>` par catégorie, `<h4>` par sous-catégorie, puis `_render_citation_row(items, cols=8)` (anneaux CSS avec image en data-URI base64, tooltip description, level label, counter, delta vert si filtré).
4. **Bloc 2 — Médailles HI** :
   - 3 metrics : `cit_distinct_medals`, `cit_total_medals`, `cit_matches_analyzed`.
   - `top_medals_fn(db_path, xuid, match_ids, top_n=None)` → `[(medal_name_id, count)]`.
   - Si non vide :
     - Mappe medal_id → label via `medal_label(nid, lang)` (`medal_translations` BCP-47).
     - **`plot_medals_distribution(top, names, top_n=25, lang)`** : barres horizontales Plotly, dégradé bleu cyan (`rgba(53,208,255,*)`), height adaptative `25*len + 80`.
     - `render_medals_grid(items, cols=8, deltas?, lang, descriptions?)` : icônes locales `static/medals/icons/{nid}.png` en `data-uri` base64, fallback `<div>?</div>` si manquante, label + counter + delta vert si filtré.

### 1.9 Catégories / sous-catégories (FR / EN)

```python
_CATEGORY_ORDER = {
  "fr": ["Mode de jeu", "Multijoueur", "Arme", "Spartan Companies", "Véhicule", "Ennemi"],
  "en": ["Game mode", "Multiplayer", "Weapon", "Spartan Companies", "Vehicle", "Enemy"],
}
_SUBCAT_ORDER = {
  "Arme":    ["Général", "UNSC", "Paria", "Forerunner", "Grenade"],
  "Weapon":  ["General", "UNSC", "Paria", "Forerunner", "Grenade"],
  "Véhicule":["Général", "UNSC", "Covenant"],
  "Vehicle": ["Général", "UNSC", "Covenant"],
  "Ennemi":  ["Covenant", "Banished"],
  "Enemy":   ["Covenant", "Banished"],
}
_SUBCAT_DISPLAY = {
  "en": {"Général": "General", "Paria": "Banished"},
  "fr": {"Paria": "Parias"},
}
```

### 1.10 Clés i18n requises

`citations_halo5_title`, `citations_medals_title`, `citations_medals_caption`,
`citations_no_medals`, `citations_medals_distribution`, `citations_medals_grid`,
`citations_no_progress`, `cit_obtained`, `cit_matches_analyzed`,
`cit_distinct_medals`, `cit_total_medals`, `cit_search`, `cit_search_placeholder`,
`cit_mastery_master`, `cit_mastery_level` (template `Niveau {level}` / `Level {level}`),
`mv_medal_fallback`, `no_matches`, `no_data_filter`, `tm_computing_medals_all`.

---

## 2. Cartographie cible (Go + React — état actuel)

### 2.1 Fichiers Go

```
apps/go-api/internal/domain/citations.go            (104 L) — types CitationItem, CitationsPageResponse, CommendationItem
apps/go-api/internal/analysis/citations.go          (169 L) — MergeCitationTotals, MergeMedalSummary, ExtractCategories, GroupCommendationsByCategory
apps/go-api/internal/analysis/citation_snippets.go  (135 L) — BuildCitationSnippets (utilisé par MatchView, parseTierTargets, computeTierProgress)
apps/go-api/internal/api/handlers/citations.go      (129 L) — POST /pages/citations + /pages/commendations
apps/go-api/internal/api/commendation_handler.go    (55 L)  — sert /static/commendations/*.png
apps/go-api/internal/service/citations_service.go   (76 L)  — orchestration
apps/go-api/internal/platform/duckdb/citations_repo.go (130 L) — Q34/Q35/Q36a/Q36b
apps/go-api/internal/platform/duckdb/queries_home_citations.go — SQL Q34→Q36b
```

### 2.2 Fichiers React

```
apps/web/src/features/citations/CitationsPage.tsx     (181 L) — page filtrée
apps/web/src/features/citations/queries.ts            (22 L)  — useCitationsPage
apps/web/src/features/career/CareerCitationsTab.tsx   (199 L) — onglet Carrière (scope global)
apps/web/src/components/ui/citation-progress-ring.tsx          — composant anneau (à vérifier réutilisé)
apps/web/src/lib/api/types.ts:1837-1871               — types CommendationSummary, MedalSummary, CitationsDeltas, CitationsPageResponse
apps/web/src/routes/players/$playerSlug/profile/citations.tsx  — redirect legacy → career?tab=citations
```

### 2.3 Routes

- `/players/{slug}/career?tab=citations` → `CareerHubPage` → `CareerCitationsTab` (filterContext = DEFAULT, hub-all).
- `/players/{slug}/profile/citations` → redirect.
- **Aucune route ne sert directement `CitationsPage.tsx`** — composant orphelin ?

À vérifier : `git grep -n "CitationsPage" apps/web/src/routes/`.

### 2.4 Shape backend vs frontend (incohérence)

**Go renvoie** (`CitationsPageResponse`) :
```ts
{ citations: CitationItem[], categories: string[], total_count: number }
// CitationItem = { name_norm, name_display, category, total, image_path?, description? }
```

**Frontend lit** (`CitationsPageResponse` dans `apps/web/src/lib/api/types.ts:1866`) :
```ts
{
  commendations: CommendationSummary[],   // {key, label, category, current_value, color, icon_path, tier_label, mastery_pct}
  medals_summary: MedalSummary[],         // {medal_name_id, name, count_filtered, count_total, description}
  deltas: { filtered_total, unfiltered_total, delta_count },
  distribution_chart: PlotlyFigurePayload | null,
}
```

Ces deux contrats **ne se chevauchent sur aucun champ**. Le React déstructure des champs qui n'existent pas dans la réponse JSON. La page n'a probablement jamais affiché autre chose que ses placeholders ou ses erreurs silencieuses (commendations.length = 0 → `EmptyStateNotice`).

### 2.5 État `commendation_handler.go`

OK — sert `static/commendations/h5g/*.png` avec gestion URL-encoding pour caractères accentués. Fonctionnel, à conserver.

### 2.6 Ce qui existe vraiment côté Go

- `citation_snippets.go:parseTierTargets` et `computeTierProgress` — réutilisables pour la mastery !
- `citations_repo.go` — lit `match_citations` (Q35), `medals_earned` (Q36a), `citation_mappings` (Q34, Q36b filtré `mapping_type='medal'` uniquement).
- Aucun écrivain dans `match_citations`.
- Aucune des 12 fonctions custom.
- Aucune logique composite.
- Aucun support `mapping_type` autre que `medal` (filtre dur Q36b ligne 409 du fichier `queries_home_citations.go`).
- `tier_targets` chargé dans Q34 mais jamais consommé pour calculer mastery (le service renvoie `Total` brut, le frontend affiche `mastery_pct` qu'il n'a pas).

---

## 3. Liste exhaustive des écarts

### 3.1 Backend (Go)

| # | Écart | Fichiers concernés | Impact UI | Priorité |
|--|---|---|---|:-:|
| B01 | Pas de moteur de calcul `CitationEngine` | `internal/analysis/citations/*` (à créer) | `match_citations` jamais alimentée → totals = 0 | P0 |
| B02 | Pas de backfill `compute_and_store_for_match` | `internal/data/citations_backfill.go` (à créer), wiring dans engine sync | flag `--citations` factice | P0 |
| B03 | Pas de dispatch `mapping_type` (sauf `medal` partiel) | `compute_citation_for_match()` à écrire | seules les médailles font progresser les commendations | P0 |
| B04 | Aucune des 12 custom functions portées | `internal/analysis/citations/custom_rules.go` (à créer) | citations Bulldozer, Annexion forcée, Flag em down, Hijack, Vandalism, Wraith/Mongoose/Warthog destroyer, Wins/mode → toutes à 0 | P0 |
| B05 | Logique composite absente | `internal/analysis/citations/composite.go` (à créer) | citations composites jamais débloquées | P1 |
| B06 | `tier_targets` non parsé pour mastery | `analysis/citations.go:MergeCitationTotals` | frontend reçoit `mastery_pct=null` → barre vide | P0 |
| B07 | Q34 ne charge pas `subcategory`, `composite_children`, `medal_ids`, `stat_name`, `award_name`, `award_category`, `custom_function` | `queries_home_citations.go:Q34CitationMappings` + `domain.CitationMappingRow` | impossible de calculer / regrouper | P0 |
| B08 | Q36b filtre `WHERE mapping_type = 'medal'` | `queries_home_citations.go:Q36b` ligne 409 | autres types ignorés | P0 |
| B09 | Shape `CitationsPageResponse` incohérent avec frontend | `domain/citations.go`, `service/citations_service.go`, `handlers/citations.go` | page non fonctionnelle | P0 |
| B10 | Pas de génération `distribution_chart` (Plotly) | nouveau builder à écrire | bloc Distribution médailles vide | P1 |
| B11 | Pas d'agrégation `medals_summary` filtré vs total | service à étendre | grille médailles non comparée | P1 |
| B12 | Pas de gestion `is_filtered` (full vs filtered match_ids) | service à étendre | deltas faux | P2 |
| B13 | Pas de loaders match-level (`load_match_medals`, `load_match_awards`, `load_match_highlight_events`, `load_match_pve_stats`, `load_match_weapon_kills`, `load_match_df`) | `internal/platform/duckdb/citations_loader_repo.go` (à créer) | engine ne peut rien lire | P0 |
| B14 | Tests `citations_service_test.go` ne couvrent pas le nouveau contract | tests à réécrire | couverture fictive | P0 |
| B15 | `medal_translations` BCP-47 non implémenté pour la chaîne médailles | cf. `MIGRATION_GAP_PYTHON_TO_GO.md §3.6` | noms médailles dégradés | P2 |

### 3.2 Frontend (React)

| # | Écart | Fichiers | Priorité |
|--|---|---|:-:|
| F01 | `CitationsPage.tsx` et `CareerCitationsTab.tsx` consomment des champs inexistants dans la réponse Go | les deux fichiers + `lib/api/types.ts` | P0 (bloquant) |
| F02 | Pas de groupage par catégorie/sous-catégorie | `CitationsPage`, `CareerCitationsTab` | P0 |
| F03 | Anneau de progression CSS absent (composant `citation-progress-ring.tsx` à vérifier mais non utilisé visiblement dans les pages) | nouveau composant `CommendationCard.tsx` | P1 |
| F04 | Pas de recherche full-text sur commendations | `CitationsPage` | P2 |
| F05 | Pas d'ordre catégories i18n (FR : Mode de jeu, Multijoueur, Arme, Spartan Companies, Véhicule, Ennemi) | constantes à créer dans `features/citations/categoryOrder.ts` | P1 |
| F06 | Pas de chart distribution médailles (Plotly) | `CitationsPage` (Plotly déjà utilisé ailleurs — `PlotlyChart`) | P1 |
| F07 | Grille médailles sans icônes | composant `MedalGridItem.tsx` à créer ; servir `static/medals/icons/{nid}.png` via handler | P1 |
| F08 | Pas de tooltip description (anneau + médaille) | composant card | P2 |
| F09 | Pas de delta visuel (`+N`) sur items filtrés | composants card | P2 |
| F10 | Pas de métriques globales (citations obtenues, distinct/total médailles, matchs analysés) | header de la page | P2 |
| F11 | Pas de toggle full-text → vue compacte | page | P3 |

### 3.3 Données / migration

| # | Écart | Action | Priorité |
|--|---|---|:-:|
| D01 | `metadata.duckdb` doit être seedé avec les 100+ commendations H5 + `subcategory` + `composite_children` | porter `scripts/populate_citation_mappings.py` en commande Go `levelup data citations seed` ou import direct du parquet | P0 |
| D02 | `match_citations` doit être créé pour chaque joueur | migration ou auto-create dans engine (cf. `_create_match_citations_if_needed`) | P0 |
| D03 | Backfill historique `match_citations` pour matchs déjà ingérés | commande CLI `levelup backfill citations --player <gt>` | P0 |
| D04 | Images `static/commendations/h5g/*.png` doivent être présentes | déjà dans le repo (cf. `git ls-tree v7/cockpit | grep h5g | wc -l`) — vérifier sur cible | P1 |

---

## 4. Plan de portage par phases

### Phase 1 — Aligner le contract API (P0 — bloque tout le reste)

**Objectif** : que le frontend reçoive enfin la shape qu'il déstructure.

**Branche** : `feat/citations-page-go-portage` (créer depuis `feat/multi-title-adapters-and-mappings`).

#### 1.1 Étendre le domain Go

Dans `apps/go-api/internal/domain/citations.go`, remplacer / compléter :

```go
// Réponse alignée sur apps/web/src/lib/api/types.ts:1866
type CitationsPageResponse struct {
    Commendations     []CommendationSummary    `json:"commendations"`
    MedalsSummary     []MedalSummary           `json:"medals_summary"`
    Deltas            CitationsDeltas          `json:"deltas"`
    DistributionChart *PlotlyFigurePayload     `json:"distribution_chart,omitempty"`
}

type CommendationSummary struct {
    Key          string   `json:"key"`             // citation_name_norm
    Label        string   `json:"label"`           // citation_name_display (i18n résolu)
    Category     *string  `json:"category"`
    Subcategory  *string  `json:"subcategory,omitempty"`
    CurrentValue int      `json:"current_value"`
    Color        *string  `json:"color"`           // null par défaut
    IconPath     *string  `json:"icon_path"`       // /static/commendations/h5g/...
    TierLabel    *string  `json:"tier_label"`      // "Niveau 3" | "Master"
    MasteryPct   *float64 `json:"mastery_pct"`     // 0..100
    Description  *string  `json:"description,omitempty"`
    Delta        int      `json:"delta,omitempty"` // si filtré
    IsMaster     bool     `json:"is_master,omitempty"`
}

type MedalSummary struct {
    MedalNameID   int64   `json:"medal_name_id"`
    Name          string  `json:"name"`
    CountFiltered int     `json:"count_filtered"`
    CountTotal    int     `json:"count_total"`
    Description   *string `json:"description,omitempty"`
    IconPath      *string `json:"icon_path,omitempty"` // /static/medals/icons/{id}.png
}

type CitationsDeltas struct {
    FilteredTotal   int `json:"filtered_total"`
    UnfilteredTotal int `json:"unfiltered_total"`
    DeltaCount      int `json:"delta_count"`
}
```

Étendre `CitationsPageRequest` :
```go
type CitationsPageRequest struct {
    Filters  *FilterContextInput `json:"filters,omitempty"` // déjà dans le domain pour d'autres pages
    Search   string              `json:"search,omitempty"`
    Category string              `json:"category,omitempty"`
}
```

#### 1.2 Étendre `CitationMappingRow` (B07)

Compléter avec : `Subcategory *string`, `CompositeChildren *string`, `MedalIDs *string`, `StatName *string`, `AwardName *string`, `AwardCategory *string`, `CustomFunction *string`, `Enabled bool`.

Mettre à jour `Q34CitationMappings` pour SELECT toutes ces colonnes.

#### 1.3 Adapter `service.GetCitationsPage`

Nouvelle signature :
```go
func (s *CitationsService) GetCitationsPage(
    ctx context.Context,
    xuid string,
    req domain.CitationsPageRequest,
) (*domain.CitationsPageResponse, error)
```

Pipeline :
1. Charger mappings (Q34 enrichi).
2. Charger `match_citations` totaux full + filtered selon `req.Filters` → resolver match_ids depuis `FilterRepo`.
3. Charger `medals_earned` totaux full + filtered.
4. Charger `medal_definitions` + `medal_translations` (BCP-47 fr/en) — option D15 incluse, sinon défaut `medal_definitions`.
5. Appliquer `_apply_composite_citations`.
6. Construire `CommendationSummary[]` :
   - Pour chaque mapping enabled :
     - `current_value = totals[norm_full]`.
     - Calculer mastery via `tier_targets` ou `composite_total`.
     - Résoudre `tier_label` ("Maître" si is_master sinon "Niveau N+1").
     - Résoudre `icon_path` depuis `image_path`.
     - Filtrer par `req.Search` et `req.Category` si fournis.
7. Construire `MedalsSummary[]` (sorted by count_filtered DESC).
8. Construire `Deltas` (somme totale full vs filtered).
9. Construire `DistributionChart` (Plotly figure JSON pour top-25 médailles, miroir de `plot_medals_distribution`).

#### 1.4 Mettre à jour les tests existants

`citations_service_test.go`, `citations_test.go` (handlers), `handlers_extra_test.go`, `handlers_internal_test.go`, `extra_coverage_test.go` (DB) — réécrire pour le nouveau contract. Snapshot tests sur la shape JSON pour bloquer toute régression.

#### 1.5 Recâbler le frontend

`CitationsPage.tsx` et `CareerCitationsTab.tsx` : les déstructurations actuelles deviennent valides — ajouter logging d'erreur si une clé manque (pour CI). Pas de gros refactor à ce stade (Phase 3 fera la refonte UI).

**Critère de sortie Phase 1** : `npm run typecheck` + `npm run test:e2e --grep citations` passent ; la page affiche les commendations existantes (uniquement type `medal` pour l'instant) avec progression linéaire et grille médailles basique.

---

### Phase 2 — Porter le moteur de calcul (P0)

**Objectif** : calculer et persister `match_citations` pour tous les `mapping_type`, replicating exactement la sémantique Python.

#### 2.1 Loaders match-level

Créer `internal/platform/duckdb/citations_loader_repo.go` :

| Méthode | Source SQL | Type Go retourné |
|---|---|---|
| `LoadMatchMedals(ctx, xuid, matchID)` | `SELECT medal_name_id, count FROM shared.medals_earned WHERE xuid=? AND match_id=?` | `map[int64]int` |
| `LoadMatchStats(ctx, xuid, matchID)` | jointure `shared.match_participants p ⨝ shared.match_registry r` filtré | `map[string]any` |
| `LoadMatchAwards(ctx, xuid, matchID)` | `SELECT award_name, SUM(award_count) FROM personal_score_awards WHERE xuid=? AND match_id=? GROUP BY 1` | `map[string]int` |
| `LoadMatchHighlightEvents(ctx, xuid, matchID)` | `SELECT time_ms, event_type FROM shared.highlight_events WHERE xuid=? AND match_id=? ORDER BY time_ms` | `[]struct{TimeMs int64; EventType string}` |
| `LoadMatchPVEStats(ctx, xuid, matchID)` | `SELECT * FROM shared_pve.pve_match_stats WHERE xuid=? AND match_id=?` | `map[string]any` |
| `LoadMatchWeaponKills(ctx, xuid, matchID)` | `SELECT effective_weapon_id, SUM(kills) FROM shared.v_weapon_kills WHERE xuid=? AND match_id=? GROUP BY 1` | `map[uint64]int` |

#### 2.2 Custom functions (12 fonctions)

Créer `internal/analysis/citations/custom_rules.go` avec un registre :

```go
type CustomFn func(ctx CustomCtx) int

type CustomCtx struct {
    MatchStats       map[string]any    // playlist_name, game_variant_name, kda, kills, deaths, outcome, ...
    Awards           map[string]int
    HighlightEvents  []HighlightEvent
}

var CustomFunctions = map[string]CustomFn{
    "compute_bulldozer":          computeBulldozer,
    "compute_wins_ctf":           computeWinsCTF,
    "compute_wins_firefight":     computeWinsFirefight,
    "compute_wins_slayer":        computeWinsSlayer,
    "compute_wins_strongholds":   computeWinsStrongholds,
    "compute_annexion_forcee":    computeAnnexionForcee,
    "compute_flag_em_down":       computeFlagEmDown,
    "compute_hijack":             computeHijack,
    "compute_vandalism":          computeVandalism,
    "compute_wraith_destroyer":   computeWraithDestroyer,
    "compute_mongoose_destroyer": computeMongooseDestroyer,
    "compute_warthog_destroyer":  computeWarthogDestroyer,
}
```

Les regex Python (`(?i)slayer|assassin`, `(?i)firefight|baptême|bapteme`) → `regexp.MustCompile` Go avec flag `(?i)`. Les patterns CTF/strongholds doivent matcher exactement les mêmes strings.

Tests unitaires obligatoires sur fixtures DuckDB (réutiliser `testdata/` du projet) — au moins 1 test par fonction custom + 1 test pour le dispatch global.

#### 2.3 Composite

Créer `internal/analysis/citations/composite.go` portant `_apply_composite_citations`. Test sur fixture avec 1 composite + 3 enfants dont 2 masterisés.

#### 2.4 Engine `Compute*ForMatch`

Créer `internal/analysis/citations/engine.go` :

```go
type Engine struct {
    repo CitationsLoaderRepo
}

func (e *Engine) ComputeAllForMatch(ctx, xuid, matchID, mappings) (map[string]int, error)
func (e *Engine) ComputeAndStore(ctx, xuid, matchID) (int, error)
```

Dispatch par `mapping_type` :
- `medal` → somme `medal_ids` (CSV) ou `medal_id` simple via `LoadMatchMedals`.
- `stat`, `pve_stat`, `weapon_stat` → `LoadMatchStats` + `LoadMatchPVEStats` + `LoadMatchWeaponKills` (clé `weapon_kills:{name}` après résolution `effective_weapon_id` → nom canonique via `meta.weapon_labels`).
- `award` → `LoadMatchAwards`.
- `custom` → `CustomFunctions[name](ctx)`.
- `composite` → 0 par-match (calculé en agrégation).

Stockage : `INSERT OR REPLACE INTO match_citations VALUES (?, ?, ?)` + marqueur `_processed=1`.

#### 2.5 Backfill CLI

Créer `apps/go-api/cmd/levelup/cmd_backfill_citations.go` :

```bash
levelup backfill citations --player <gt> [--force] [--match <id>]
```

Pipeline :
1. Lister `match_ids` du joueur (via `match_participants` xuid).
2. Filtrer ceux non encore dans `match_citations` (sauf `--force`).
3. Pour chaque batch (size 50), `Engine.ComputeAndStore`.
4. Logguer progress (`charmbracelet/log` déjà présent).

Ajouter un hook post-sync dans le pipeline existant (équivalent du `backfill_citations()` post-sync Python — voir `src/data/citations_backfill.py`).

#### 2.6 Seed `citation_mappings`

Importer le contenu canonique depuis le corpus v7/cockpit. Deux options :

**Option A (recommandée)** — exporter `citation_mappings` du `metadata.duckdb` source :
```bash
duckdb data/warehouse/metadata.duckdb -c "COPY citation_mappings TO 'apps/go-api/migrations/seed/citation_mappings.parquet' (FORMAT PARQUET)"
```
Puis charger ce parquet dans la migration Go (`internal/migration/steps_metadata.go`).

**Option B** — porter `scripts/populate_citation_mappings.py` en `cmd_data citations seed` Go avec une grosse table de constantes inline (~100 entrées). Plus volumineux, mais self-contained.

#### 2.7 Tests d'intégration

Créer `internal/analysis/citations/engine_integration_test.go` :
- Fixture metadata + shared + player avec 5 matchs et 3 commendations connues.
- Run `Engine.ComputeAndStore` sur chaque match.
- Assert : counts agrégés == valeurs Python attendues (capturer un run Python sur la même fixture comme oracle, stocker en JSON).

**Critère de sortie Phase 2** : `levelup backfill citations --player <gt>` fait passer un joueur de 14 commendations à 100+, dont au moins toutes les `custom` et `composite` débloquées. Diff Go vs Python ≤ 1% par citation (tolérance pour `compute_annexion_forcee` qui est explicitement une approximation).

---

### Phase 3 — Refonte UI complète (P1)

**Objectif** : porter fidèlement la richesse visuelle Streamlit en React.

#### 3.1 Composants à créer

```
apps/web/src/features/citations/
├── CitationsPage.tsx                  ← refonte
├── CareerCitationsTab.tsx             ← refonte (variante sans filtres)
├── components/
│   ├── CommendationCard.tsx           ← anneau CSS (image + ring conique + tier + counter + delta)
│   ├── CommendationGrid.tsx           ← grille 8 cols groupée par catégorie/sous-catégorie
│   ├── CommendationSearch.tsx         ← input recherche full-text
│   ├── MedalGridItem.tsx              ← icône + label + counter + delta
│   ├── MedalsDistributionChart.tsx    ← wrapper PlotlyChart pour distribution_chart
│   └── CitationsMetrics.tsx           ← 3 metrics header (citations obtenues / matchs / médailles)
├── constants.ts                       ← CATEGORY_ORDER_BY_LANG, SUBCAT_ORDER, SUBCAT_DISPLAY
├── grouping.ts                        ← groupCommendationsByCategory + sortSubcategories
└── queries.ts                         ← inchangé (modulo nouveau type)
```

#### 3.2 Anneau de progression CSS

Reproduire `os-citation-ring` (cf. `commendations.py` lignes 200-240) :

```css
.commendation-ring {
  --p: 0; /* 0..1 */
  --ring-color: hsl(var(--primary));
  width: 96px; height: 96px;
  border-radius: 50%;
  background:
    conic-gradient(var(--ring-color) calc(var(--p) * 360deg), hsl(var(--muted)) 0)
    padding-box,
    var(--img) center/cover no-repeat content-box;
  border: 4px solid transparent;
  position: relative;
}
.commendation-ring--master {
  --ring-color: hsl(var(--rarity-legendary));
  /* halo doré, animation pulse optionnelle */
}
```

Utiliser le système de tokens (cf. CLAUDE.md §20) : `--ring-color` mappé sur `tokenCssVar('progress')` ou `tokenCssVar('rarity-legendary')`.

#### 3.3 Groupage par catégorie/sous-catégorie

Constantes (porter exactement) :
```ts
export const CATEGORY_ORDER_BY_LANG: Record<'fr'|'en', string[]> = {
  fr: ['Mode de jeu', 'Multijoueur', 'Arme', 'Spartan Companies', 'Véhicule', 'Ennemi'],
  en: ['Game mode', 'Multiplayer', 'Weapon', 'Spartan Companies', 'Vehicle', 'Enemy'],
}
export const SUBCAT_ORDER: Record<string, string[]> = {
  Arme:     ['Général', 'UNSC', 'Paria', 'Forerunner', 'Grenade'],
  Weapon:   ['General', 'UNSC', 'Paria', 'Forerunner', 'Grenade'],
  // ... idem pour Véhicule/Vehicle, Ennemi/Enemy
}
```

Fonction `groupCommendationsByCategory(items, lang)` :
1. `Map<category, Map<subcategory|null, items[]>>`.
2. Tri catégorie via `CATEGORY_ORDER_BY_LANG[lang]`, puis catégories restantes triées alpha.
3. Tri sous-catégorie via `SUBCAT_ORDER[category]`, puis restantes alpha, `null` à la fin.

Rendu :
```tsx
{groupedCategories.map(({ category, subcategories }) => (
  <section key={category}>
    <h2 className="...">{category}</h2>
    {subcategories.map(({ subcategory, items }) => (
      <div key={subcategory ?? '_'}>
        {subcategory && <h4>{translateSubcategory(subcategory, lang)}</h4>}
        <CommendationGrid items={items} cols={8} />
      </div>
    ))}
  </section>
))}
```

#### 3.4 Distribution médailles (Plotly)

Côté backend : sérialiser un `PlotlyFigurePayload` (déjà type existant `apps/web/src/lib/api/types.ts`). Mirror exact de `plot_medals_distribution` (top-25, barres horizontales, dégradé `rgba(53,208,255, alpha)`, height adaptative `25*N+80`).

Côté frontend : `<PlotlyChart figure={data.distribution_chart} />`.

Tests visuels : utiliser le harness Playwright existant (`apps/web/e2e/citations.spec.ts` à créer).

#### 3.5 Grille médailles

Reproduire `render_medals_grid` :
- 8 colonnes responsive (4/6/8 selon viewport — le code v5 utilise déjà `cols=8` Streamlit).
- Icônes locales : étendre `commendation_handler.go` (ou créer `medal_icon_handler.go`) pour servir `static/medals/icons/{nid}.png`.
- Fallback `<div>?</div>` si icône manquante.
- Tooltip `name : description`.
- Delta vert `+N` si filtré.

#### 3.6 Recherche full-text

Input texte (debounced 200ms) → POST `/pages/citations` avec `{search: "..."}`. Backend filtre côté Go (le seed est petit, pas besoin d'index FTS). Couverture : nom, description, catégorie. Sensibilité : insensible casse + accents (utiliser `golang.org/x/text/unicode/norm` NFKD comme Python).

#### 3.7 Métriques

3 KPI cards en haut :
- Citations obtenues (somme `current_value` sur `is_master=true`) — clarifier la sémantique Python : c'est `sum(citations_full.values())` toutes valeurs confondues, **pas** uniquement masterisées.
- Matchs analysés (count des matchs filtrés ou full).
- Médailles distinctes / Total médailles.

#### 3.8 i18n

Ajouter à `apps/web/src/features/notifications/i18n.ts` (ou nouveau `features/citations/i18n.ts`) les clés portées du `shared.py` :

```ts
export const citationsI18n = {
  fr: {
    citations_halo5_title: 'Commendations Halo 5',
    citations_medals_title: 'Médailles Halo Infinite',
    citations_medals_distribution: 'Distribution des médailles',
    citations_medals_grid: 'Grille des médailles',
    citations_no_medals: 'Aucune médaille obtenue.',
    cit_obtained: 'Citations obtenues',
    cit_matches_analyzed: 'Matchs analysés',
    cit_distinct_medals: 'Médailles distinctes',
    cit_total_medals: 'Total médailles',
    cit_search: 'Recherche',
    cit_search_placeholder: 'Filtrer par nom, description ou catégorie',
    cit_mastery_master: 'Maître',
    cit_mastery_level: 'Niveau {level}',
    no_data_filter: 'Aucune donnée pour le filtre actuel.',
  },
  en: { /* miroir */ },
}
```

#### 3.9 Tokens couleurs

Conformer à CLAUDE.md §20 : aucun hex/Tailwind color class. Anneau utilise `tokenCssVar('progress')` (en cours) et `tokenCssVar('rarity-legendary')` (master). Delta vert utilise `tokenCssVar('success')`.

**Critère de sortie Phase 3** :
- Tests visuels Playwright passent (regression snapshots).
- Page Carrière → Citations en mode `?tab=citations` affiche 100+ commendations groupées par 6 catégories, distribution Plotly, grille médailles avec icônes.
- Lighthouse a11y ≥ 95.

---

### Phase 4 — Intégration au sync + déploiement (P0/P1)

#### 4.1 Hook post-sync

Dans `apps/go-api/internal/sync/engine.go` (ou équivalent), après chaque batch de matchs synchronisés, déclencher `Engine.ComputeAndStore` pour chaque nouveau `match_id`. Comportement = `_run_citations_backfill_after_sync` Python (cf. `engine.py` ligne 1700+).

Configuration : flag `SyncScope.Citations` déjà câblé — implémenter le branchement réel.

#### 4.2 Migration des joueurs existants

Commande one-shot :
```bash
levelup backfill citations --all-players
```

Documenté dans `docs/MIGRATION_GAP_PYTHON_TO_GO.md` § Citations.

#### 4.3 Performance / cache

- Mappings en cache mémoire (TTL 5 min) — cf. `_load_citations_from_db` Python (`@st.cache_data(ttl=300)`).
- Réponse `/pages/citations` peut être mise en cache 60s pour les vues `hub-all` (sans filtre).

#### 4.4 Observabilité

Logs structurés sur :
- Nb mappings chargés.
- Nb matchs traités par run de backfill.
- Custom function dispatchées (counter par `function_name`).
- Erreurs de parsing `tier_targets` / `composite_children`.

---

## 5. Risques & points ouverts

Voir section §9.4 pour **R10-R15 (audit exhaustif citations)** — les risques majeurs pour 100% de portabilité.

| # | Risque | Severity | Mitigation |
|--|---|:-:|---|
| R01 | `compute_annexion_forcee` est explicitement une approximation Python (~90% fiable). Le porter "à l'identique" propage l'imprécision. | 🟡 P1 | Documenter dans le code Go ; étudier en parallèle si l'API Halo expose enfin la vraie citation. |
| R02 | Le seed `citation_mappings` peut diverger entre v7/cockpit metadata.duckdb et la cible Go. | 🟡 P1 | Phase 2.6 option A (export parquet depuis metadata source) garantit la fidélité. |
| R03 | Les noms de playlists/variants utilisés par les regex `compute_wins_*` sont localisés (français) — un joueur Steam EN aura des matchs sans match. | 🟡 P1 | Vérifier le corpus actuel ; étendre les patterns (cf. R12). |
| R04 | La résolution `weapon_kills:{name}` Python utilise un nom canonique EN résolu via `shared.v_weapon_kills`. Côté Go : `meta.weapon_labels` UBIGINT. Doit produire la même clé. | 🔴 P0 | Cf. R14 (audit §9.5). Tests d'intégration sur les 5 citations `weapon_stat` les plus jouées. |
| R05 | Pas de gestion BCP-47 pour `medal_translations` côté Go. | 🟡 P2 | Hors scope direct citations, tracker sous une autre tâche (cf. `MIGRATION_GAP_PYTHON_TO_GO.md §3.6`). |
| R06 | `CitationsPage.tsx` peut être un composant orphelin (pas de route) — vérifier. | 🟡 P2 | `git grep -n "CitationsPage" apps/web/src/routes/`. |
| R07 | Le frontend déstructure aujourd'hui sans erreur car les champs absents = `undefined` puis fallback empty state. Pas d'alarme ⇒ régression silencieuse possible. | 🔴 P0 | Ajouter un test contractuel `npm run test:contract -- citations` qui valide la shape de réponse (Phase 1). |
| R08 | Volumétrie `match_citations` : ~14 lignes par match × N matchs × N joueurs. Pour 100k matchs : ~1.4M lignes. | 🟢 P3 | OK — Python opère déjà à cette échelle. Index sur `(match_id, citation_name_norm)` (PK) suffit. |
| R09 | Les images commendations H5G sont URL-encodées sur disque (`%C3%89crasement.png`). Le handler Go gère déjà ce cas. | 🟢 P3 | OK. |
| **R10** | `player_vs_everything` — stat_name `total_enemy_kills` n'existe **PAS** en `pve_match_stats` | 🔴 P0 | Cf. §9.2.2 — disable ou implémenter logique SUM(grunt+elite+…) |
| **R11** | `carrier_killed` ambiguïté — deux citations mappent, données divergentes possible | 🟡 P1 | Cf. §9.2.3 — audit real awards SQL sur joueur test |
| **R12** | Playlist names mélangés FR/EN — regex `compute_wins_*` incomplètes | 🟡 P1 | Cf. §9.2.4 — étendre patterns regex |
| **R13** | medal_id 9000000001 (Avenger) factice — jamais trouvée | 🟡 P1 | Cf. §9.2.5 — disable au seed |
| **R14** | weapon_labels UBIGINT — résolution effective_weapon_id → nom 25 citations cassées si absent | 🔴 P0 | Cf. §9.2.7 — vérifier Q et mapping dans Phase 2.1 |
| **R15** | `power_weapon_kills` colonne — jamais utilisée mais à vérifier | 🟡 P2 | Aucune impact actuel |

---

## 6. Estimation de charge

| Phase | Tâche | Estimation |
|---|---|---:|
| 1 | Aligner contract API + recâbler frontend | 1.5 j |
| 2 | Loaders + custom functions (12) + composite + engine + backfill CLI + seed | 4-5 j |
| 3 | Refonte UI (anneau CSS, groupage, search, distribution Plotly, grille médailles, i18n) | 3 j |
| 4 | Hook post-sync + observabilité + migration joueurs existants + docs | 1 j |
| | **Total** | **9.5-10.5 j** |

Phasage recommandé en branches séparées (cf. CLAUDE.md "1 tâche = 1 branche") :
- `feat/citations-go-engine` (Phase 1 + 2 + 4 — backend + sync) → PR.
- `feat/citations-ui-refonte` (Phase 3 — UI) → PR (peut démarrer dès Phase 1 mergée).

---

## 7. Critères d'acceptation globaux

- [ ] La page Carrière → Citations affiche 100+ commendations groupées en 6 catégories avec sous-catégories ordonnées.
- [ ] Chaque carte affiche : image, anneau de progression conique, label `Niveau N` ou `Maître`, compteur (`X/Y` ou `X` masterisé), delta vert si filtré.
- [ ] La recherche full-text filtre instantanément (debounce 200ms).
- [ ] Le bloc Distribution affiche un Plotly horizontal top-25.
- [ ] La grille médailles affiche les icônes locales avec tooltip + delta.
- [ ] `match_citations` est alimentée automatiquement post-sync.
- [ ] `levelup backfill citations --player <gt>` fait passer un joueur existant à un volume de citations comparable au Python (delta ≤ 1% modulo R01).
- [ ] La shape API matche exactement `apps/web/src/lib/api/types.ts:CitationsPageResponse` (test contractuel CI).
- [ ] Les 12 custom functions ont chacune un test unitaire avec fixture connue.
- [ ] Aucun fichier ne dépasse 500 L, aucune fonction ne dépasse 80 L (CLAUDE.md §13/14).
- [ ] i18n FR + EN couverts pour toutes les clés portées du § 1.10.

---

## 9. Audit exhaustif des citations — 87 citations à 100%

> **Vérification fonctionnelle** : chaque citation Python → données disponibles en Go → portabilité garantie.

### 9.1 Inventaire complet

**Total : 87 citations** (80-85 activées, 5-7 disabled)

| Type | Nombre | Activées | Sources données Go requises |
|---|---:|---:|---|
| **stat** (direct match_participants) | 5 | 5 | shared.match_participants colonnes : `assists`, `melee_kills`, `headshot_kills`, `kills`, `power_weapon_kills` |
| **pve_stat** (Firefight) | 10 | 7 | shared_pve.pve_match_stats colonnes : `grunt_kills`, `elite_kills`, `jackal_kills`, `hunter_kills`, `boss_kills`, `total_enemy_kills` |
| **award** (joueur objectives) | 5 | 5 | personal_score_awards : `zone_captured`, `carrier_killed`, `flag_returned`, `zone_secured` |
| **award** (vehicle) | 7 | 7 | personal_score_awards : `destroyed_banshee`, `destroyed_ghost`, `destroyed_scorpion`, `destroyed_wasp`, `destroyed_wraith`, `destroyed_mongoose`, `destroyed_warthog`, `destroyed_rocket_warthog` |
| **custom** (formules) | 11 | 11 | Voir §9.3 — highlight_events, match_stats, awards, formules regex |
| **medal** (simple id) | 8 | 8 | shared.medals_earned + IDs fixes |
| **medal_ids** (CSV combiné) | 3 | 3 | shared.medals_earned + parsing CSV |
| **weapon_stat** (per-kill) | 25 | 25 | shared.v_weapon_kills + résolution `effective_weapon_id` → nom canonique EN |
| **composite** (validation enfants) | 7 | 7 | Citation dépend de ses enfants masterisés |

### 9.2 Dépendances par mapping_type

#### 9.2.1 **stat** — 5 citations (100% portables ✅)

| Citation | stat_name | Source | Colonne shared.match_participants | Status |
|---|---|---|---|---|
| `assistant` | `assists` | shared MP | `assists` (INT) | ✅ |
| `close_combat` | `melee_kills` | shared MP | `melee_kills` (INT) | ✅ |
| `melee_fighter` | `melee_kills` | shared MP | `melee_kills` (INT) | ✅ (doublon avec `close_combat`) |
| `headshot` | `headshot_kills` | shared MP | `headshot_kills` (INT) | ✅ |
| `spartan_killer` | `kills` | shared MP | `kills` (INT) | ✅ |

**Risque** : `power_weapon_kills` utilisée dans aucune citation, mais mentionnée en variable Streamlit. À vérifier en Go si colonne existe.

#### 9.2.2 **pve_stat** — 10 citations (7 enabled, 3 disabled)

Tous depuis `shared_pve.pve_match_stats` (colonnes confirmées par skill db-schema) :

| Citation | stat_name | Colonne (PVE) | Enabled | Status | Notes |
|---|---|---|---|---|---|
| `grunt_slayer` | `grunt_kills` | `grunt_kills` | ✅ | ✅ | Covenant — common |
| `elite_slayer` | `elite_kills` | `elite_kills` | ✅ | ✅ | Covenant — common |
| `jackal_slayer` | `jackal_kills` | `jackal_kills` | ✅ | ✅ | Covenant — common |
| `hunter_slayer` | `hunter_kills` | `hunter_kills` | ✅ | ✅ | Covenant — common |
| `like_a_boss` | `boss_kills` | `boss_kills` | ✅ | ✅ | Boss — general |
| `player_vs_everything` | `total_enemy_kills` | **MANQUANT** | ✅ | ⚠️ **R10** | **Colonne `total_enemy_kills` n'existe PAS en PVE** — retournera 0 ou erreur |
| `sentinel_slayer` | `sentinel_kills` | **MANQUANT** | ❌ | disabled | Pas en `pve_match_stats` |
| `brute_slayer` | `brute_kills` | `brute_kills` (existe mais…) | ❌ | disabled | Données rares ou NULL |
| `skimmer_slayer` | `skimmer_kills` | `skimmer_kills` (existe mais…) | ❌ | disabled | Données rares ou NULL |
| `marine_slayer` | `marine_kills` | **MANQUANT** | ❌ | disabled | Marines = alliés, pas ennemis |

**R10 — `player_vs_everything` cassée** : citation `enabled=true` mais `stat_name='total_enemy_kills'` n'existe pas. Revient probablement à `SUM(grunt + elite + jackal + hunter + brute + skimmer + crawler + soldier + knight + warden_kills)` — à clarifier.

#### 9.2.3 **award** — 12 citations (objectives + vehicles)

Tous depuis `personal_score_awards` (player DB) :

| Citation | award_name | Données | Status |
|---|---|---|---|
| `charge` | `zone_captured` | Strongholds — captures | ✅ |
| `flag_defender` | `carrier_killed` | CTF — tuer porteur | ⚠️ **R11** |
| `got_you` | `flag_returned` | CTF — ramener drapeau | ⚠️ **R11** |
| `stakeholder` | `zone_secured` | Strongholds — défendre zone | ✅ |
| `flag_carrier_hunter` | `carrier_killed` | CTF — dual source ? | ⚠️ **R11** |
| `banshee_destroyer` | `destroyed_banshee` | Destroys vehicle | ✅ |
| `ghost_destroyer` | `destroyed_ghost` | Destroys vehicle | ✅ |
| `scorpion_destroyer` | `destroyed_scorpion` | Destroys vehicle | ✅ |
| `wasp_destroyer` | `destroyed_wasp` | Destroys vehicle | ✅ |
| `wraith_destroyer` (direct award, non custom) | `destroyed_wraith` | Destroys vehicle | ✅ |

**R11 — Ambiguïté `carrier_killed` vs `flag_carrier_kill`** : deux citations différentes (`flag_defender` + `flag_carrier_hunter`) mappent sur `carrier_killed`. À vérifier que les données les expriment identiquement. Legacy pourrait avoir `"Flag Carrier Kill"` (anglais).

#### 9.2.4 **custom** — 11 citations (formules métier)

| Citation | Fonction custom | Dépendances | Portabilité | Status |
|---|---|---|---|---|
| `bulldozer` | `compute_bulldozer` | playlist_name, game_variant_name, kda | Regex `(?i)slayer\|assassin` + filtre firefight/btb | ✅ |
| `forced_annexation` | `compute_annexion_forcee` | highlight_events + awards | Walk events `mode`/`death` | ✅ (approximation ~90%) |
| `flag_em_down` | `compute_flag_em_down` | awards → `runner_stopped` + legacy fallbacks | Combinaison d'awards | ✅ |
| `grand_theft` | `compute_hijack` | awards → `hijacked_*` | Grep awards prefix | ✅ |
| `positive_contribution` | `compute_bulldozer` (alias) | playlist_name, kda | Alias → même fonction | ✅ |
| `flag_victory` | `compute_wins_ctf` | playlist_name, outcome | Regex CTF + outcome=2 | ⚠️ **R12** |
| `slayer_victory` | `compute_wins_slayer` | playlist_name, outcome | Regex Slayer/Assassin + outcome=2 | ⚠️ **R12** |
| `strongholds_victory` | `compute_wins_strongholds` | playlist_name, outcome | Regex Stronghold/Bases + outcome=2 | ⚠️ **R12** |
| `vandalism` | `compute_vandalism` | awards → `destroyed_*` | Grep awards prefix | ✅ |
| `wraith_destroyer` (custom, non award) | `compute_wraith_destroyer` | awards → `destroyed_wraith` + fallbacks | Awards + fallbacks legacy | ✅ |
| `mongoose_destroyer` | `compute_mongoose_destroyer` | awards → `destroyed_mongoose` + fallbacks | Awards + fallbacks legacy | ✅ |
| `warthog_destroyer` | `compute_warthog_destroyer` | awards → `destroyed_warthog` + `destroyed_rocket_warthog` + fallbacks | Aggregation | ✅ |

**R12 — Localisation des noms playlist** : les regex Python matchent du français (ex: `"drapeau"`, `"bases"`) et de l'anglais (ex: `"flag"`, `"stronghold"`). Le corpus réel peut être mélangé (API en en-US, UI en FR). À vérifier sur un joueur test.

#### 9.2.5 **medal** — 8 citations (IDs)

| Citation | medal_id | Nom | Trouvé en HI ? | Status |
|---|---|---|---|---|
| `splatter` | 221693153 | Splatter | ✅ | ✅ |
| `driver` | 3169118333 | Vehicle Violence | ✅ | ✅ |
| `assassin` | 548533137 | Backstab | ✅ | ✅ |
| `frag_grenade` | 2648272972 | Frag Grenade | ✅ | ✅ |
| `plasma_grenade` | 3655682764 | Plasma Grenade | ✅ | ✅ |
| `eagle_eye` / `im_just_perfect` | 1512363953 | Perfection | ✅ | ✅ |
| `the_reaper` | 2625820422 | Postmortem | ✅ | ✅ |
| `too_fast_for_you` | 2123530881 | Turncoat | ✅ | ✅ |
| `avenger` | 9000000001 | (factice) | ❌ | ❌ **R13** |

**R13 — medal_id 9000000001 factice** : ID inventée pour "Avenger" (tuer qui m'a tué). Aucune médaille réelle. Citation retournera toujours 0 ou doit être disabled. À clarifier au seed.

#### 9.2.6 **medal_ids** (CSV) — 3 citations (combinaisons)

| Citation | medal_ids (CSV) | Médailles HI correspondantes | Status |
|---|---|---|---|
| `spartan_carnage` | `2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471` | Killing sprees (x8) | ✅ |
| `opportunist` | `622331684,2063152177,4261842076,2137071619,1486797009,1430343434,2242633421` | Multi-kills (x7) | ✅ |
| `lucky` | `3905838030,3091261182` | Luck + Empty Mag (x2) | ✅ |

Parsing CSV obligatoire en Go.

#### 9.2.7 **weapon_stat** — 25 citations (per-kill par arme)

Tous utilisent `stat_name = "weapon_kills:<weapon_name_canonical>"`.

| Catégorie | Armes | Noms canoniques requis | Status |
|---|---:|---|---|
| **UNSC (10)** | BR75, MA40 AR, Sidekick, Commando, Sniper, SPNKr, Bulldog, Bandit Evo, Hydra, Mutilator | `weapon_labels` v5.4 (UBIGINT id → name) | ✅ si mapping existe |
| **Parias (9)** | Stalker, Needler, Energy Sword, Mangler, Skewer, Gravity Hammer, Pulse Carbine, Ravager, Plasma Pistol | `weapon_labels` v5.4 | ✅ si mapping existe |
| **Forerunner (5)** | Heatwave, Cindershot, Sentinel Beam, Disruptor, Shock Rifle | `weapon_labels` v5.4 | ✅ si mapping existe |

**R14 — weapon_labels UBIGINT** : les clés `weapon_kills:...` en `citation_mappings` utilisent le nom canonique EN (ex: `"BR75"`), pas l'ID filmshell. Le loader `LoadMatchWeaponKills` doit faire : `SELECT effective_weapon_id, SUM(kills) FROM shared.v_weapon_kills WHERE xuid=? AND match_id=? GROUP BY 1` → puis **convertir `effective_weapon_id` en nom canonique** via `metadata.weapon_labels`. Si le mapping `effective_weapon_id ↔ nom` est cassé, les 25 citations retourneront 0.

### 9.3 Composite — 7 citations (validation enfants)

| Composite | Enfants (count) | Type enfants | Status |
|---|---:|---|---|
| `covenant_destroyer` | 7 | `pve_stat` | Dépend des 3 disabled (`sentinel_kills`, `brute_slayer`, `skimmer_slayer`) → partiel |
| `grenade_mastery` | 2 | `medal` | `frag_grenade`, `plasma_grenade` → ✅ |
| `vehicle_mastery` | 9 | `medal` + `custom` | splash + driver + 7 destroyers → ✅ |
| `human_weapons_mastery` | 10 | `weapon_stat` | UNSC → ✅ si weapon_labels OK |
| `paria_weapons_mastery` | 9 | `weapon_stat` | Parias → ✅ si weapon_labels OK |
| `forerunner_weapons_mastery` | 5 | `weapon_stat` | Forerunner → ✅ si weapon_labels OK |
| `all_weapons_mastery` | 4 | `composite` | Agrège les 3 composites armes + grenade → ✅ si enfants OK |

---

## 9.4 Lacunes récapitulatives

| # | Risque | Severity | Impact Go | Mitigation |
|---|---|:-:|---|---|
| **R10** | `player_vs_everything` stat_name `total_enemy_kills` n'existe pas en pve_match_stats | 🔴 P0 | Citation retourne 0 toujours | Disabled cette citation OR implémenter logique SUM(grunt+elite+…) dans loader |
| **R11** | `carrier_killed` vs `flag_carrier_kill` — ambiguité noms awards | 🟡 P1 | Deux citations mappent sur même award, données divergentes possible | Audit real awards SQL sur joueur test ; normaliser noms |
| **R12** | Playlist names mélangés FR/EN — regex Python incomplètes | 🟡 P1 | Formules `compute_wins_*` n'activent jamais sur corpus FR | Étendre patterns regex : `ctf\|capture.*drapeau\|drapeau.*neutre\|capture du drapeau\|neutral.*flag` |
| **R13** | medal_id 9000000001 (Avenger) est factice | 🟡 P1 | Citation jamais matchée, retourne 0 | Disabled `avenger` au seed OU implémenter logique custom (tuer qui m'a tué) |
| **R14** | weapon_labels UBIGINT — résolution effective_weapon_id → nom | 🔴 P0 | 25 citations weapon_stat cassées si mapping absent | Charger Q via JOIN metadata.weapon_labels ; tester sur 3-5 armes fréquentes |
| **R15** | `power_weapon_kills` colonne — référencée mais jamais utilisée | 🟡 P2 | Aucune citation PVP | Si colonne existait, serait portée automatiquement |

---

## 9.5 Checklist pré-Phase 2

**Vérifications à faire AVANT de implémenter le backfill** :

```bash
# 1. Vérifier l'existence des colonnes match_participants en shared
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "DESCRIBE shared.match_participants" | grep -E "assists|melee_kills|power_weapon|headshot"

# 2. Vérifier player_score_awards pour noms awards (FR vs EN)
duckdb data/titles/halo_infinite/players/TestGT/stats.duckdb \
  "SELECT DISTINCT award_name FROM personal_score_awards WHERE award_name LIKE '%flag%' OR award_name LIKE '%drapeau%' LIMIT 20"

# 3. Vérifier v_weapon_kills et effective_weapon_id → nom mapping
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT DISTINCT effective_weapon_id FROM shared.v_weapon_kills LIMIT 3" && \
  "SELECT * FROM metadata.weapon_labels WHERE weapon_id IN (SELECT DISTINCT effective_weapon_id FROM shared.v_weapon_kills LIMIT 3)"

# 4. Compter les Firefight matches — vérifier pve_match_stats non-empty
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT COUNT(DISTINCT match_id) FROM shared.highlight_events WHERE event_type LIKE '%firefight%' OR mode_variant_name LIKE '%Firefight%'"

# 5. Test formula compute_wins_ctf — vérifier playlist_name patterns
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT DISTINCT playlist_name FROM shared.match_registry LIMIT 50" | grep -i "ctf\|drapeau\|flag"

# 6. Vérifier medal_id 9000000001 — ne devrait pas exister
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT COUNT(*) FROM shared.medals_earned WHERE medal_name_id = 9000000001"
```

Si une vérification échoue (retour 0, NULL, error), **ajouter une note en comment dans le code Go** et disable la citation concernée en seed.

---

## 8. Annexes

### 8.1 Mapping rapide Python → Go

| Python | Go (cible) |
|---|---|
| `src/analysis/citations/engine.py:CitationEngine` | `internal/analysis/citations/engine.go:Engine` |
| `src/analysis/citations/composite.py:_apply_composite_citations` | `internal/analysis/citations/composite.go:ApplyComposites` |
| `src/analysis/citations/custom_rules.py:CUSTOM_FUNCTIONS` | `internal/analysis/citations/custom_rules.go:CustomFunctions` |
| `src/analysis/citations/_data_loader.py:CitationDataLoaderMixin` | `internal/platform/duckdb/citations_loader_repo.go:CitationsLoaderRepo` |
| `src/data/citations_backfill.py` | `internal/data/citations_backfill.go` + `cmd/levelup/cmd_backfill_citations.go` |
| `src/ui/pages/citations.py:render_citations_page` | `apps/web/src/features/citations/CitationsPage.tsx` |
| `src/ui/commendations.py:render_h5g_commendations_section` | `apps/web/src/features/citations/components/CommendationGrid.tsx` |
| `src/ui/commendations.py:_compute_mastery_display` | déjà partiel dans `analysis/citation_snippets.go:computeTierProgress` — à étendre |
| `src/ui/medals.py:render_medals_grid` | `apps/web/src/features/citations/components/MedalGrid.tsx` |
| `src/visualization/distributions.py:plot_medals_distribution` | builder Go `internal/analysis/citations/medals_distribution_chart.go` (output `PlotlyFigurePayload`) |

### 8.2 Fichiers à créer / modifier

**Créer** (Go) :
- `internal/analysis/citations/engine.go`
- `internal/analysis/citations/composite.go`
- `internal/analysis/citations/custom_rules.go`
- `internal/analysis/citations/medals_distribution_chart.go`
- `internal/platform/duckdb/citations_loader_repo.go`
- `internal/data/citations_backfill.go`
- `cmd/levelup/cmd_backfill_citations.go`
- Tests : `*_test.go` correspondants + `engine_integration_test.go`.

**Modifier** (Go) :
- `internal/domain/citations.go` (nouveaux types)
- `internal/platform/duckdb/queries_home_citations.go` (Q34 enrichi, Q36b sans filtre `mapping_type`)
- `internal/platform/duckdb/citations_repo.go` (lectures full+filtered)
- `internal/service/citations_service.go` (orchestration nouvelle shape)
- `internal/api/handlers/citations.go` (request avec `filters`, `search`, `category`)
- `internal/port/services.go` (signature `GetCitationsPage` étendue)
- `internal/migration/steps_metadata.go` (seed `citation_mappings`)

**Créer** (frontend) :
- `apps/web/src/features/citations/components/CommendationCard.tsx`
- `apps/web/src/features/citations/components/CommendationGrid.tsx`
- `apps/web/src/features/citations/components/CommendationSearch.tsx`
- `apps/web/src/features/citations/components/MedalGridItem.tsx`
- `apps/web/src/features/citations/components/MedalsDistributionChart.tsx`
- `apps/web/src/features/citations/components/CitationsMetrics.tsx`
- `apps/web/src/features/citations/constants.ts`
- `apps/web/src/features/citations/grouping.ts`
- `apps/web/src/features/citations/i18n.ts`
- E2E : `apps/web/e2e/citations.spec.ts`.

**Modifier** (frontend) :
- `apps/web/src/features/citations/CitationsPage.tsx` (refonte)
- `apps/web/src/features/career/CareerCitationsTab.tsx` (refonte)
- `apps/web/src/features/citations/queries.ts` (paramètres search/category)
- `apps/web/src/lib/api/types.ts` (synchroniser éventuels champs ajoutés : `subcategory`, `is_master`, `delta`).

**Documentation** :
- `docs/MIGRATION_GAP_PYTHON_TO_GO.md` (§ 2.6 → mettre à jour status après merge).
- `docs/FR/CITATIONS.md` (porter en français, à jour pour Go).
- `docs/COMMENDATIONS.md` (équivalent EN).
- `.ai/thought_log.md` (entrée par phase mergée).

### 8.3 Vérifications préalables avant Phase 1

**Étape -1 : Audit données (cf. §9.5)** — exécuter **avant Phase 2** pour garantir 100% de portabilité.

```bash
# 1. Confirmer que CitationsPage.tsx est routée (ou non)
git grep -n "CitationsPage" apps/web/src/routes/

# 2. Confirmer que static/commendations/h5g/*.png sont présents
ls apps/go-api/static/commendations/h5g/ | wc -l   # attendu : 100+

# 3. Confirmer que metadata.duckdb dispose de citation_mappings (avec audit)
duckdb data/titles/halo_infinite/warehouse/metadata.duckdb \
  "SELECT mapping_type, COUNT(*), SUM(CASE WHEN enabled THEN 1 ELSE 0 END) FROM citation_mappings GROUP BY mapping_type"

# 4. Confirmer la version actuelle de match_citations sur un joueur test
duckdb data/titles/halo_infinite/players/<gt>/stats.duckdb \
  "SELECT COUNT(*) FROM match_citations; SELECT COUNT(DISTINCT citation_name_norm) FROM match_citations"

# 5. Vérifier colonnes match_participants — R12 (power_weapon_kills)
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "DESCRIBE shared.match_participants" | grep -E "assists|melee_kills|power_weapon|headshot"

# 6. Vérifier personal_score_awards — R11 (award names FR vs EN)
duckdb data/titles/halo_infinite/players/<gt>/stats.duckdb \
  "SELECT DISTINCT award_name FROM personal_score_awards WHERE award_name LIKE '%flag%' OR award_name LIKE '%drapeau%' OR award_name LIKE '%zone%' LIMIT 30"

# 7. Vérifier weapon_labels — R14 (effective_weapon_id → nom)
duckdb data/titles/halo_infinite/warehouse/metadata.duckdb \
  "SELECT COUNT(*) FROM weapon_labels; SELECT * FROM weapon_labels LIMIT 5"

# 8. Vérifier pve_match_stats colonnes — R10 (total_enemy_kills)
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "DESCRIBE shared_pve.pve_match_stats" 2>/dev/null || echo "shared_pve.duckdb pas attaché — recréer connexion"

# 9. Vérifier playlist_name patterns — R12 (compute_wins_*) 
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT DISTINCT playlist_name FROM shared.match_registry LIMIT 100" | grep -iE "ctf|drapeau|flag|slayer|assassin|stronghold|bases"

# 10. Vérifier medal_id 9000000001 — R13 (Avenger)
duckdb data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT COUNT(*) FROM shared.medals_earned WHERE medal_name_id = 9000000001"
```

**Résultat attendu** :
- (3) ≥ 75 citations activées (`stat`, `pve_stat`, `award`, `medal`, `weapon_stat`, `custom`)
- (5-9) Colonnes retrouvées ✅ — si non, documenter lacune en R##
- (10) Count = 0 (ID factice) — OK
