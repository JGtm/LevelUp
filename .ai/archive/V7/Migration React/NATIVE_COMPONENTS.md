# NATIVE_COMPONENTS.md — Tables AG Grid et KPI Cards React

> Inventaire structuré et tasklist de migration des **tables de données** et **indicateurs/KPI cards**
> à implémenter en composants React natifs (AG Grid Community et CSS/SVG) plutôt qu'en `react-plotly.js`.
>
> **Source de référence** : `.ai/CHARTS_AND_TABLES.md` (inventaire exhaustif).
> **Raison d'être** : les tableaux et indicateurs simples sont de mauvais candidats pour `react-plotly.js`
> — ils sont plus performants, ergonomiques et maintenables en natif React.

---

## Stack cible

| Type | Lib | Statut |
|------|-----|--------|
| Tables de données riches | **AG Grid Community** | Figé — `DECISIONS.md` §3 |
| KPI cards / badges de stat | **React CSS + Tailwind v4** | Figé |
| Jauges arc (progression rang) | **CSS `conic-gradient` ou SVG** | À arbitrer à l'implémentation (sprint Career) |
| Grilles d'icônes (médailles) | **React CSS grid** | Figé |
| Expand panels du scoreboard | **shadcn/ui Collapsible** | Figé |

> **Note** : les graphiques complexes (timeseries, radars, heatmaps, distributions, KDE, subplots dual-Y…)
> restent **tous** en `react-plotly.js` via figures JSON backend. Voir `DECISIONS.md` §5.

---

## DoD transversal

> Applicable à **chaque item** de ce fichier, sans exception.
> Un item n'est `done` que si tous les points suivants sont cochés.

- [ ] Le schéma de données API est documenté dans `API_CONTRACTS.md` — payload rows + méta (pagination, totaux)
- [ ] L'endpoint fournit des **données brutes** (pas une figure `PlotlyFigurePayload`)
- [ ] Le composant React est placé dans `apps/web/src/features/<feature>/` (ou `components/` si transversal)
- [ ] Données vérifiées sur le **corpus de référence** contre les golden values (`tests/fixtures/golden_values/`)
- [ ] Aucune régression sur les **liens de navigation internes** (clic ligne → route, clic gamertag → Explorer)
- [ ] Le composant respecte le thème sombre Halo : fond `#1d2328`, texte blanc, palette cyan `#33D6FF` / rouge `#FF4B4B` / vert `#00DC82`
- [ ] Les trois états **loading / empty / error** sont gérés et visibles dans l'UI

---

## Récapitulatif par slice

| Slice | Items |
|-------|-------|
| Slice 2 — Career | [C1](#c1--gauge-progression-rang-xp-career), [C2](#c2--gauge-progression-rang-héros-career), [A2](#a2--top-10-meilleurs-matchs-career), [A3](#a3--top-10-pires-matchs-career), [A4](#a4--encounters-adversaires-récurrents-career) |
| Slice 3 — Match History | [A1](#a1--match-history--explorer-partagée) côté Match History |
| Slice 4 — Explorer | [A1](#a1--match-history--explorer-partagée) côté Explorer (réutilise le composant) |
| Slice 5 — Match View | [A5](#a5--scoreboard-du-match-match-view), [A6](#a6--encounters-du-match-match-view), [C3](#c3--kpi-cards-expected-vs-actual-match-view), [C4](#c4--bloc-rang-csrlusr-après-match-match-view), [C5](#c5--indicateur-kd-vs-nemesis-match-view), [E1](#e1--panneau-détail-joueur-expandable-match-view) |
| Slice 6+ — Teammates | [B1](#b1--récapitulatif-coéquipiers-teammates) |
| Slice 6+ — Session Compare | [B2](#b2--historique-de-session-session-compare), [D2](#d2--indicateurs-delta-cumulés-a-vs-b--4-session-compare) |
| Slice 6+ — Timeseries | [D1](#d1--indicateurs-régression-kd--3-timeseries) |
| Slice 6+ — Objectives | [D3](#d3--gauge-ratio-objectifs-objectives) |
| Slice 6+ — Citations | [B3](#b3--grille-médailles-citations) |
| Transversal | [D4](#d4--indicateur-performance-session-transversal) |

---

## Groupe A — Tables riches → AG Grid Community

> **Règle de choix AG Grid vs `<table>` HTML** :
> - AG Grid Community si ≥ 2 colonnes triables, colonnes custom (badges, liens, couleur conditionnelle),
>   ou données potentiellement paginées / virtualisées.
> - `<table>` Tailwind suffit pour les listes simples (≤ 5 colonnes, lecture seule, statiques).

---

### A1 — Match History + Explorer (partagée)

**Slices** : 3 (Match History) + 4 (Explorer)
**Source** : `§13.1 / §9.1` → `src/ui/pages/match_table_html.py::render_match_table_html`
**Statut** : `todo`

| Colonne | Type | Tri | Notes |
|---------|------|:---:|-------|
| # | int | ✅ | Index chronologique inversé |
| Date | datetime | ✅ | Format `DD/MM HH:mm` (FR) |
| Mode | string | ✅ | |
| Carte | string | ✅ | |
| K | int | ✅ | |
| D | int | ✅ | |
| A | int | ✅ | |
| Score | int | ✅ | |
| Résultat | enum | ✅ | Chip coloré + emoji |
| Durée | duration | ✅ | Format `MM:SS` |

**Comportements spécifiques** :

- Couleur de fond de ligne selon outcome : vert Victoire / rouge Défaite / bleu Égalité / gris DNF
- Clic ligne → navigation vers `/players/:playerSlug/matches/:matchId`
- Bouton export CSV scope-aware (Match History uniquement, pas Explorer)

**Tasklist A1** :

- [ ] DoD transversal complet (voir § ci-dessus)
- [ ] `API_CONTRACTS.md` mis à jour : schéma rows (10 colonnes) + payload méta (`total_count`, `has_more`)
- [ ] Composant `<MatchTable>` créé dans `apps/web/src/features/match-history/`
- [ ] Réutilisé tel quel dans `apps/web/src/features/explorer/` (pas de doublon)
- [ ] Couleur ligne : vert / rouge / bleu / gris selon outcome — pas seulement le chip Résultat
- [ ] Clic ligne navigue via TanStack Router (`navigate`)
- [ ] Export CSV présent dans Match History, absent dans Explorer
- [ ] Chip Résultat : emoji + label identiques à l'existant (✓ Victoire / ✗ Défaite / ═ Égalité / ? DNF)
- [ ] **Parité** : cardinalité lignes identique aux golden values `match_history_full.json` sur corpus de référence
- [ ] **Parité** : les 5 `match_id` de référence sont présents aux bons rangs / dates

---

### A2 — Top 10 meilleurs matchs (Career)

**Slice** : 2 (Career)
**Source** : `§1.5` → `src/ui/pages/career_top_matches_render.py`
**Statut** : `todo`

| Colonne | Type | Notes |
|---------|------|-------|
| # | int | Rang 1–10, fixe |
| Date | datetime | |
| Carte / Mode | string | Concaténé carte + mode |
| K | int | |
| D | int | |
| A | int | |
| Score perso | int | Valeur de tri (DESC) |
| Durée | duration | `MM:SS` |
| Résultat | enum | Chip coloré |
| K/D ratio | float | 1 décimale |
| Badges | tags | DOMINATION / HUMILIATION / REMONTADA / DÉBÂCLE / CONTRE-REMONTADA |

**Tasklist A2** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` mis à jour : schéma `TopMatchesRow[]`
- [ ] Composant `<TopMatchesTable>` créé dans `apps/web/src/features/career/`
- [ ] Tri Score DESC intégré et **non modifiable** par l'utilisateur (c'est un classement, pas un tableau interactif)
- [ ] Badges rendus en chips colorés :
  - `DOMINATION` → vert `#00DC82`
  - `HUMILIATION` → violet `#8B5CF6`
  - `REMONTADA` → bleu `#0072B2`
  - `DÉBÂCLE` → orange `#D55E00`
  - `CONTRE-REMONTADA` → cyan `#33D6FF`
- [ ] Clic ligne → navigation vers Match View
- [ ] Nombre de lignes : exactement `min(10, nb matchs disponibles)`
- [ ] **Parité** : match #1 (meilleur score) identique aux golden values `career.json`

---

### A3 — Top 10 pires matchs (Career)

**Slice** : 2 (Career)
**Source** : `§1.6` → `src/ui/pages/career_top_matches_render.py`
**Statut** : `todo`

> Identique à A2 avec tri Score ASC. Même composant React, prop `variant="worst"`.

**Tasklist A3** :

- [ ] DoD transversal complet
- [ ] Réutilise le composant `<TopMatchesTable>` de A2 — **pas de nouveau composant**
- [ ] Prop `variant` change le tri (ASC) et potentiellement le titre
- [ ] **Parité** : match #1 (pire score) identique aux golden values `career.json`

---

### A4 — Encounters adversaires récurrents (Career)

**Slice** : 2 (Career)
**Source** : `§1.7` → `src/ui/pages/career_encounters_render.py::render_encounters_section`
**Statut** : `todo`

| Colonne | Type | Notes |
|---------|------|-------|
| Gamertag | string | Lien → Explorer avec ce gamertag |
| Côté | enum | Allié / Ennemi — chip |
| N° rencontre | int | |
| Win Rate % | float | Affiché avec barre de progression |
| K/D | float | 2 décimales |
| Badges | tags | "Allié plus" / "Coriace" / "Noix dure" |

**Comportements spécifiques** :

- Filtres période configurables (7j / 30j / 90j / Tout) — état dans l'URL
- Légende badges inline sous le tableau

**Tasklist A4** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` mis à jour : endpoint accepte `?period=7d|30d|90d|all`, retourne `EncounterRow[]`
- [ ] Composant `<EncountersTable>` créé dans `apps/web/src/features/career/`
- [ ] Filtre période : dropdown ou tabs — **valeur dans l'URL** (`?encounter_period=30d`)
- [ ] Chip Côté : "Allié" vert / "Ennemi" rouge
- [ ] Chips badges identiques à ceux de A6 (composant partagé `<EncounterBadge>`)
- [ ] Légende badges rendue sous la table
- [ ] Clic Gamertag → route Explorer avec le gamertag pré-rempli en query param

---

### A5 — Scoreboard du match (Match View)

**Slice** : 5 (Match View)
**Source** : `§5.12` → `src/ui/pages/match_view_scoreboard.py`
**Statut** : `todo`

> La table la plus complexe de l'app — 18 colonnes, highlighting min/max + expand row.

| Colonne | Highlight | Notes |
|---------|:---------:|-------|
| Joueur | — | Gamertag, non comparé |
| Rang | — | Icône rang, non comparé |
| Score | ✅ max vert | |
| K | ✅ max vert | |
| D | ✅ **min** vert (inversé) | |
| A | ✅ max vert | |
| KDA | ✅ max vert | |
| Arme principale | — | Nom arme, non comparé |
| Spree | ✅ max vert | |
| Headshots | ✅ max vert | |
| Perfect Kills | ✅ max vert | |
| Tirs | — | Non comparé |
| Précision % | ✅ max vert | |
| Mêlée | ✅ max vert | |
| Arme lourde | — | Nom arme, non comparé |
| DMG infligés | ✅ max vert | |
| DMG subis | ✅ **min** vert (inversé) | |
| Durée vie moy. | ✅ max vert | `MM:SS` |

**Comportements spécifiques** :

- Highlighting automatique : max = fond vert semi-transparent, min = fond rouge semi-transparent, par colonne numérique — colonnes inversées : D, DMG subis
- Colonnes non comparées : Joueur, Rang, Arme principale, Arme lourde, Tirs
- Badge MVP (vert) sur la ligne avec score max, badge LVP (rouge/orange) sur la ligne avec score min
- Expand row → ouvre le panneau E1 (détail joueur)
- Deux sections visuellement séparées : Mon équipe / Équipe adverse

**Tasklist A5** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` mis à jour : `ScoreboardRow[]` avec les 18 colonnes + `team_id`
- [ ] Composant `<MatchScoreboard>` créé dans `apps/web/src/features/match-view/`
- [ ] Highlighting min/max fonctionnel avec les colonnes inversées correctement (D, DMG subis)
- [ ] Colonnes non comparées exclues du highlighting
- [ ] Badge MVP positionné sur la ligne avec `score = max(scores)`
- [ ] Badge LVP positionné sur la ligne avec `score = min(scores)`
- [ ] Expand row : clic sur une ligne ouvre `<PlayerDetailPanel>` (E1) en dessous
- [ ] Séparation visuelle claire entre les deux équipes (header ou séparateur)
- [ ] **Parité** : scoreboard de 3 matchs de référence vérifié — roster, K/D/A, scores

---

### A6 — Encounters du match (Match View)

**Slice** : 5 (Match View)
**Source** : `§5.18` → `src/ui/pages/match_view_encounters.py::render_encounter_section`
**Statut** : `todo`

> Version allégée de A4, sans filtre période — limitée aux adversaires/alliés du match courant.

| Colonne | Type | Notes |
|---------|------|-------|
| Gamertag | string | |
| Côté | enum | Allié / Ennemi — chip |
| N° rencontre | int | Badge ordinal |
| Win Rate % | float | |
| K/D | float | |
| Badges | tags | |

**Tasklist A6** :

- [ ] DoD transversal complet
- [ ] Réutilise le composant `<EncounterBadge>` de A4 — **pas de doublon**
- [ ] Badge ordinal `Nème rencontre` visible en info-bulle ou en sous-titre de ligne
- [ ] Légende badges inline
- [ ] Pas de filtre période (contrairement à A4)

---

### B1 — Récapitulatif coéquipiers (Teammates)

**Slice** : post-MVP
**Source** : `§4.9` → `src/ui/pages/teammates.py` (actuellement `st.dataframe`)
**Statut** : `todo`

| Colonne | Notes |
|---------|-------|
| Gamertag | Lien → profil joueur si disponible |
| Matchs communs | Tri par défaut DESC |
| Kills | |
| Morts | |
| Ratio | |
| Win Rate % | |
| Avg Life | `MM:SS` |
| Profil joueur | Lien externe ou internal si le joueur est configuré |

**Tasklist B1** :

- [ ] DoD transversal complet
- [ ] Tri Matchs communs DESC par défaut, modifiable par l'utilisateur
- [ ] Clic Gamertag → route Explorer ou profil joueur si disponible

---

### B2 — Historique de session (Session Compare)

**Slice** : post-MVP
**Source** : `§6.8` → `src/ui/pages/_session_compare_history.py` (actuellement `st.dataframe`)
**Statut** : `todo`

| Colonne | Notes |
|---------|-------|
| # | Index dans la session |
| Date | |
| Mode | |
| Carte | |
| K / D / A | |
| Score | |
| Résultat | Chip |
| Durée | `MM:SS` |

**Tasklist B2** :

- [ ] DoD transversal complet
- [ ] Réutilise le composant `<MatchTable>` de A1 avec un sous-ensemble de colonnes — **pas de doublon**
- [ ] Colonnes non présentes dans A1 (# session) ajoutées via prop `extraColumns`

---

### B3 — Grille médailles (Citations)

**Slice** : post-MVP
**Source** : `§8.2` → `src/ui/pages/citations.py`
**Statut** : `todo`

> Ce n'est **pas** une table AG Grid — c'est une grille CSS `grid` responsive.
> Elle remplace du HTML Streamlit `st.markdown` unsafe.

**Tasklist B3** :

- [ ] DoD transversal complet
- [ ] Layout responsive : `grid-cols-4` sur mobile → `grid-cols-6` sur desktop
- [ ] Chaque cellule : icône médaille (SVG ou image) + nom + count
- [ ] Triable par count DESC (tri côté client)
- [ ] **Parité** : nombre de médailles distinctes identique aux golden values sur corpus de référence

---

## Groupe C — KPI Cards et Gauges → React CSS/SVG (MVP)

---

### C1 — Gauge progression rang XP (Career)

**Slice** : 2 (Career)
**Source** : `§1.1` → `src/ui/components/career_progress_circle.py::create_career_progress_gauge`
**Statut** : `todo`

**Données nécessaires dans le payload** :

| Champ | Type | Description |
|-------|------|-------------|
| `rank_name` | string | Nom du rang actuel (FR) |
| `current_xp` | int | XP accumulée sur ce rang |
| `xp_for_next_rank` | int | XP totale du palier actuel |
| `progress_pct` | float | 0.0–1.0, dérivé des deux précédents |

**Comportement visuel** :

- Arc de cercle ou `conic-gradient` CSS (0–360°) — rendu sans Plotly
- 4 zones couleur selon `progress_pct` : rouge (0–25%) / orange (25–50%) / cyan (50–75%) / vert (75–100%)
- Valeur centrale affichée : `{progress_pct * 100:.0f}%`
- Sous-titre : `{current_xp} / {xp_for_next_rank} XP`
- Titre : `rank_name`
- Taille équivalente : ~280px de hauteur

**Tasklist C1** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` : champs `rank_name`, `current_xp`, `xp_for_next_rank`, `progress_pct` présents dans le payload Career
- [ ] Composant `<RankProgressGauge>` créé dans `apps/web/src/components/`
- [ ] Couleur de l'arc déterminée dynamiquement par les 4 seuils — **aucune couleur hardcodée**
- [ ] Valeur identique aux golden values `career.json` sur le corpus de référence

---

### C2 — Gauge progression rang Héros (Career)

**Slice** : 2 (Career)
**Source** : `§1.2` → `src/ui/components/career_progress_circle.py::create_hero_progress_gauge`
**Statut** : `todo`

> Identique à C1. Valeur : % de XP total cumulé vers le rang 272 (9 319 350 XP).
> Même composant React `<RankProgressGauge>` avec des props différentes.

**Données nécessaires** :

| Champ | Type | Description |
|-------|------|-------------|
| `total_xp_cumulated` | int | XP totale cumulée du joueur |
| `hero_progress_pct` | float | `total_xp / 9_319_350`, capé à 1.0 |

**Tasklist C2** :

- [ ] DoD transversal complet
- [ ] Réutilise `<RankProgressGauge>` avec `xpTarget=9_319_350`, `title="Héros"` — **pas de nouveau composant**
- [ ] `API_CONTRACTS.md` : champs `total_xp_cumulated`, `hero_progress_pct` dans le payload Career
- [ ] Valeur identique aux golden values sur le corpus de référence

---

### C3 — KPI cards Expected vs Actual (Match View)

**Slice** : 5 (Match View)
**Source** : `§5.0 Partie 1` → `src/ui/pages/match_view_charts.py::render_expected_vs_actual`
**Statut** : `todo`

> 3 cartes de métriques côte à côte : **Kills / Deaths / Assists**.
> Chaque carte affiche la valeur réelle, la valeur attendue (CSR/LUSR) et un delta coloré.

**Données nécessaires** :

| Champ | Type | Description |
|-------|------|-------------|
| `actual_kills` | int | |
| `expected_kills` | float | Peut être null si pas de données CSR |
| `actual_deaths` | int | |
| `expected_deaths` | float | |
| `actual_assists` | int | |
| `expected_assists` | float | |
| `has_expected_data` | bool | Si false, cartes grisées |

**Règles de couleur du delta** :

- Kills : delta vert si `actual > expected`, rouge sinon
- Deaths : delta vert si `actual < expected` (moins de morts = mieux), rouge sinon (**inversé**)
- Assists : delta vert si `actual > expected`, rouge sinon

**Tasklist C3** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` : 7 champs ci-dessus dans le payload Match View
- [ ] Composant `<StatExpectedCard>` créé dans `apps/web/src/features/match-view/`
- [ ] Les 3 cartes partagent le même composant, prop `metric` distingue "kills" / "deaths" / "assists"
- [ ] Règle d'inversion Deaths appliquée via prop `lowerIsBetter=true`
- [ ] Cartes grisées (opacité réduite) si `has_expected_data=false`
- [ ] **Parité** : valeurs vérifiées sur 2 matchs de référence avec données CSR disponibles

---

### C4 — Bloc rang CSR/LUSR après match (Match View)

**Slice** : 5 (Match View)
**Source** : `§5.17` → `src/ui/pages/match_view_rank.py::_build_match_rank_html`
**Statut** : `todo`

**Données nécessaires** :

| Champ | Type | Description |
|-------|------|-------------|
| `rank_tier` | string | Tier (ex. "Platinum") |
| `rank_sub` | int | Sous-rang (1–6) |
| `rank_points` | int | Points CSR |
| `delta_points` | int | Variation depuis le match précédent (±) |
| `had_bot_teammate` | bool | Note contextuelle si true |
| `rank_data_available` | bool | Composant masqué si false |

**Tasklist C4** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` : champs ci-dessus dans le payload Match View (null si indisponible)
- [ ] Composant `<MatchRankBadge>` créé dans `apps/web/src/features/match-view/`
- [ ] Badge delta : vert si `delta_points > 0`, rouge si < 0, gris si = 0
- [ ] Avertissement bot visible si `had_bot_teammate=true`
- [ ] Composant **entièrement masqué** (pas de placeholder vide) si `rank_data_available=false`

---

### C5 — Indicateur K/D vs nemesis (Match View)

**Slice** : 5 (Match View)
**Source** : `§5.11` → `src/visualization/_antagonist_duels.py::create_kd_indicator`
**Statut** : `todo`

> Simple indicateur chiffre + delta. Implémentable en React CSS sans Plotly.

**Données nécessaires** :

| Champ | Type | Description |
|-------|------|-------------|
| `kd_vs_nemesis` | float | K/D ratio vs l'adversaire principal de ce match |
| `nemesis_gamertag` | string | |
| `delta_vs_average` | float | Écart vs K/D moyen tous adversaires |

**Tasklist C5** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` : champs dans la section `antagonist` du payload Match View
- [ ] Composant `<KdIndicatorCard>` (ou réutilise `<StatExpectedCard>` de C3 si compatible)
- [ ] Delta coloré vert (favorable) / rouge (défavorable) avec signe `+/-` explicite
- [ ] Libellé : `K/D vs {nemesis_gamertag}`
- [ ] **Parité** : valeur vérifiée sur les matchs de référence avec un nemesis identifiable

---

## Groupe D — KPI Cards et Indicateurs (Post-MVP)

---

### D1 — Indicateurs régression K/D × 3 (Timeseries)

**Slice** : post-MVP
**Source** : `§3.19` → `src/visualization/_perf_progression.py::plot_regression_trend`
**Statut** : `todo`

3 cartes : **Pente K/D** / **Pente Win Rate** / **R²**

**Données nécessaires** :

| Champ | Type | Description |
|-------|------|-------------|
| `kd_slope` | float | Coefficient de régression K/D |
| `winrate_slope` | float | Coefficient de régression win rate |
| `r_squared` | float | 0.0–1.0 |
| `has_enough_for_trend` | bool | Conditionnel : cachées si false |

**Tasklist D1** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` mis à jour pour le payload Timeseries
- [ ] Alerte ⚠ visible si `r_squared < 0.3` (annotation "tendance non significative")
- [ ] Les 3 cartes **masquées** si `has_enough_for_trend=false`

---

### D2 — Indicateurs delta cumulés A vs B × 4 (Session Compare)

**Slice** : post-MVP
**Source** : `§6.6` → `src/visualization/_perf_session.py::create_cumulative_metrics_indicator`
**Statut** : `todo`

4 cartes : **Kills / Deaths / KDA / Win Rate**, valeur = delta Session B − Session A.

**Tasklist D2** :

- [ ] DoD transversal complet
- [ ] Delta Deaths : vert si B < A (moins de morts = mieux)
- [ ] Signe `+/-` explicite sur toutes les cartes

---

### D3 — Gauge ratio objectifs (Objectives)

**Slice** : post-MVP
**Source** : `§7.3` → `src/visualization/objective_charts.py::plot_objective_ratio_gauge`
**Statut** : `todo`

- Valeur : % du score total venant des objectifs (0–100%)
- Étiquette profil calculée : `< 20%` → Slayer / 20–40% → Polyvalent / `> 40%` → Support
- Seuils couleur : rouge `< 20%`, orange 20–40%, cyan 40–60%, vert `> 60%`

**Tasklist D3** :

- [ ] DoD transversal complet
- [ ] Réutilise `<RankProgressGauge>` de C1 si la structure d'arc est configurable, sinon nouveau composant
- [ ] Étiquette profil rendue sous la valeur chiffrée

---

### D4 — Indicateur performance session (transversal)

**Slice** : post-MVP
**Source** : `§11.3` → `src/visualization/participation_charts_extra.py::create_participation_indicator`
**Statut** : `todo`

- Valeur : score total de la session
- Delta : vs baseline historique (moyenne)

**Tasklist D4** :

- [ ] DoD transversal complet
- [ ] Composant transversal dans `apps/web/src/components/` (réutilisable across pages)

---

## Groupe E — Composants hybrides complexes

---

### E1 — Panneau détail joueur expandable (Match View)

**Slice** : 5 (Match View)
**Source** : `§5.19` → `src/ui/pages/match_view_scoreboard_detail.py::render_scoreboard_player_detail_html`
**Statut** : `todo`

> Panneau collapsible déclenché par le clic sur une ligne de A5 (scoreboard).
> Implémenté avec `shadcn/ui Collapsible`.

| Section | Contenu |
|---------|---------|
| Armes | Top armes par kills — nom + count + headshot % |
| Médailles | Grille icônes médailles avec count (variante de B3) |
| Citations | Liste citations gagnées dans ce match |
| Attendu | Comparatif réel vs. attendu CSR / historique (variante de C3) |
| Antagoniste | Kills / Deaths vs adversaire principal |
| Qualité données | Badge 🎮 "Données complètes" vs 🔗 "Données partagées seulement" |

**Tasklist E1** :

- [ ] DoD transversal complet
- [ ] `API_CONTRACTS.md` : sous-objet `player_detail` dans le payload Match View (peut être chargé en lazy si lourd)
- [ ] Composant `<PlayerDetailPanel>` dans `apps/web/src/features/match-view/`
- [ ] Implémenté avec `shadcn/ui Collapsible` — ouverture/fermeture via re-clic sur la ligne A5
- [ ] Les 6 sections s'enchaînent verticalement (pas d'onglets internes)
- [ ] Section "Attendu" conditionnelle : masquée si `has_expected_data=false`
- [ ] Badge qualité données affiché sur une ligne dédiée en bas du panneau

---

## Gaps documentés

> Éléments absents de l'inventaire `CHARTS_AND_TABLES.md` à documenter avant la slice concernée.

| Gap | Page | Priorité | Action requise |
|-----|------|----------|----------------|
| Affichage solo vs escouade (toggle / indicateur) | Win Loss / Career | Post-MVP | Documenter dans `CHARTS_AND_TABLES.md` avant la slice Win Loss |
| Bannières / cartes résumé session (Home Mission Control) | Home | Post-MVP | À documenter entièrement — page non encore inventoriée |
| Carrousel médias Last Match | Last Match | MVP (Slice 5) | Vérifier si la page Last Match a un rendu propre — section absente du CHARTS_AND_TABLES.md |
