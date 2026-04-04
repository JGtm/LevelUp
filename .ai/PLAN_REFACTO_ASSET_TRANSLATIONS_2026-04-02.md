# Plan détaillé — Refacto architecture : i18n pipeline + abstraction ChartData

**Date :** 2026-04-02 (mis à jour 2026-04-04)
**Branche de travail :** `refactor/asset-translations-db-first`
**Backlog source :** `.ai/BACKLOG.md` — § "[refacto] Traductions d'assets"

> **État au 2026-04-04 :** Phase 0 complète + hotfixes UI supprimés. Reste : Phases 1→4 (Axe A) puis Phase 7 (Axe B).

---

## Vue d'ensemble : deux axes complémentaires

Ce plan adresse deux problèmes orthogonaux qui ont tous les deux causé des rustines lors du feature "records historiques" (branch `fix/map-ui-fr-mismatch`) :

| Axe | Couche | Symptôme observé | Solution |
|-----|--------|-----------------|----------|
| **A — i18n pipeline** | Données | `map_ui` absent de `df` → chaque page le recalcule localement (7 hotfixes posés) | `add_i18n_display_columns()` appelée une seule fois après `load_df_optimized()` |
| **B — ChartData** | Visualisation | Ajouter les records a requis de modifier 4 signatures + 3 implémentations distinctes | `ChartData` dataclass centralisant séries, couleurs, records, downsampling |

Les deux axes sont **indépendants** mais **se renforcent** : l'Axe A garantit `map_ui` dans le df avant qu'il entre dans `ChartData`, ce qui permet de supprimer les guards dynamiques `_map_col` encore présents dans `squad_records.py`.

**Ordre recommandé :** Phases 0→6 (Axe A) puis Phase 7 (Axe B). La Phase 7 peut démarrer en parallèle de la Phase 4 si besoin, mais pas avant la Phase 0.

---

## 0. Diagnostic racine — Pourquoi les fixes sont éparpillés

### 0.A — Pipeline i18n (Axe A)

```
load_df_optimized()          → df  (a map_name_fr, PAS map_ui)
       ↓
_filters_apply / _cascade    → dff (a map_ui) ← lang-aware ✅
       ↓
base, full_df, fr_sub...     → pas de map_ui  ← chaque page corrige localement ❌
```

**3 implémentations parallèles de la même logique** :

| Symbole | Fichier | Défaut |
|---------|---------|--------|
| `add_ui_columns()` | `filters.py` | Ignore `map_name_fr`, utilise toujours `map_name` EN |
| `_add_derived_columns()` | `_filters_apply.py` | Correcte mais uniquement appliquée à `dff` |
| `_vectorize_ui_columns()` | `_filters_cascade.py` | Idem |
| patches ad-hoc | `win_loss.py`, `teammates_views.py`, `friends_impact_heatmap.py`, `_teammates_trio.py` | Rustines symptomatiques |

**La vraie cause racine** : `df` (sortie de `load_df_optimized`) a déjà `map_name_fr` (depuis `v_match_full`), mais `map_ui` n'est jamais calculé à ce stade.

### 0.B — Architecture graphique (Axe B)

Les 5 graphes "barres multi-joueurs par match" de la page Escouade partagent les mêmes besoins transversaux, mais chaque fonction les implémente indépendamment :

| Besoin | Situation actuelle | Coût réel |
|--------|-------------------|-----------|
| Axe X normalisé (entiers + labels) | Chaque fonction reconstruit son propre idx_map | Logique dupliquée ×5 |
| Couleurs par joueur | 7-10 kwargs individuels par fonction | PLR0913 systématique |
| Records historiques | Ajout = 4 signatures modifiées + 3 implémentations | 2 sessions de travail |
| Records par carte | `_map_col` guard dans `squad_records.py` | Dette supplémentaire |
| Downsampling | `_downsample_for_plot` défini dans `timeseries.py` seulement | Pages escouade non protégées |

**Graphes concernés (§4 de `CHARTS_AND_TABLES.md`) :**

| Graphe | Fichier | barmode |
|--------|---------|---------|
| `plot_trio_metric` | `trio.py` | `group` |
| `plot_trio_kills_deaths` | `trio.py` | `group` |
| `plot_multi_metric_bars_by_match` | `match_bars.py` | `group` |
| `plot_hs_pk_stacked` | `teammates_hs_pk.py` | `overlay` |
| `_render_per_minute_stats` | `_teammates_trio_helpers.py` | catégoriel |

---

## 1. Revue de la section backlog

### 1.1 Ce qui est exact

| Item | Statut |
|------|--------|
| Deux implémentations dupliquées de `is_uuid_like` | ✅ Confirmé |
| `_normalize_mode_label` dans `teammates_helpers.py` est redondante | ✅ Confirmé (avec nuance — voir §1.2) |
| `normalize_mode_label` couple `st.session_state` à la logique métier | ✅ Confirmé |
| `translate_playlist_name` n'a pas de cache | ✅ Confirmé |

### 1.2 Corrections et nuances

**Item 2 — `_normalize_mode_label` :**
La version locale ne fait qu'appeler `translate_pair_name(pair_name)`, sans `clean_asset_label`, sans stripping `" on "`, sans nettoyage Forge/Ranked. Remplacer par `normalize_mode_label` est un **gain de justesse**, pas seulement un DRY. Dépend de l'Item 4.

**Item 5 — Armes : déjà implémenté**
`resolve_weapon_display` dans `src/analysis/_weapon_data.py` suit déjà l'ordre correct (fusion → DB → dicts Python). Aucun code à écrire — barrer dans le backlog.

**Item 3 — Carte DB-first : résolu structurellement par Item 0**
Avec `add_i18n_display_columns` appliqué dès le chargement, l'Item 3 se réduit à supprimer `add_ui_columns()` et les 7 hotfixes pré-Item0.

### 1.3 Ordre corrigé

```
0 (couche centralisée) → 1 (is_uuid_like) → 4 (découplage st) → 2 (teammates) → 3 (nettoyage) → (5 : vérification) → 6
```

---

## 2. Analyse technique — Axe A (i18n)

### 2.0 Item 0 — ~~Créer `src/app/i18n_columns.py`~~ ✅ COMPLÉTÉ

> **2026-04-04 — Déjà implémenté.** `src/app/i18n_columns.py` existe et est intégré dans
> `main_helpers.py:373`. Voir ci-dessous pour les divergences avec le plan initial.

**Pipeline effectif :**
```
load_df_optimized()            → df brut (map_name_fr présent, map_ui absent)
       ↓
add_i18n_display_columns(df, lang)   ← EN PLACE dans main_helpers.py:375
       ↓
df enrichi (map_ui, playlist_ui présents) ← stocké dans session_state ✅
       ↓
dff = apply_filters(df, ...)   → hérite automatiquement de map_ui ✅
base, full_df, ...             → héritent automatiquement de map_ui ✅
```

**Divergences par rapport au plan initial :**

| Point | Plan initial | Implémentation réelle |
|-------|-------------|----------------------|
| Signature | 6 params dont 4 callbacks fn | `(df, lang="fr")` seulement — autonome |
| `mode_ui` | Calculé ici via `normalize_mode_label_fn` | **Non calculé ici** — délégué à `_add_derived_columns` dans `_filters_apply.py` (pair_name brut ≠ label normalisé, décision correcte) |
| `playlist_ui` | Via callback | Via `coalesce([playlist_name_fr, playlist_name])` directement |

**Hotfixes pré-Item0 — état actuel :**

| Fichier | Hotfix | État |
|---------|--------|------|
| `win_loss.py` L261 | injection `map_ui` sur `base` | ✅ Supprimé |
| `teammates_views.py` L289 | injection `map_ui` sur `fr_sub` | ✅ Supprimé |
| `friends_impact_heatmap.py` | injection `map_ui` dans `plot_squad_map_heatmap` | ✅ Supprimé |
| `_teammates_trio.py` | injection `map_ui` sur `_me_full` | ✅ Supprimé |
| `_query_teammate_shared_stats` (SQL) | alias `map_ui` dans SELECT | ✅ Supprimé |
| `query_teammate_full_history` (SQL) | `JOIN v_match_full` + alias `map_ui` | ✅ Supprimé |
| `compute_squad_records_per_map` | guard dynamique `_map_col` | ⏳ Encore présente — à supprimer en Phase 4 |
| `add_ui_columns()` dans `filters.py` | L48 + caller L315 | ⏳ Encore présente — à supprimer en Phase 4 |
| `maps.py:89-90` | guard `"map_ui" if "map_ui" in df` | ⏳ Oubliée dans le plan initial — à supprimer en Phase 4 |

---

### 2.1 Item 1 — Unifier `is_uuid_like` dans `src/utils/strings.py`

| Fichier | Symbole | Pattern |
|---------|---------|---------|
| `src/ui/translations.py` L26 | `_is_uuid_like(s: str) -> bool` | regex interne |
| `src/app/helpers.py` L50 | `is_uuid_like(s: str) -> bool` | regex module-level |

Créer `src/utils/strings.py` avec `is_uuid_like`. Supprimer les deux copies. Risque : quasi nul.

---

### 2.2 Item 4 — Extraire `st.session_state` de `normalize_mode_label`

Transformer la signature (actuellement [helpers.py:55-72](src/app/helpers.py#L55)) :
```python
def normalize_mode_label(
    pair_name: str | None,
    *,
    lang: str = "fr",
    normalize: bool = True,
) -> str | None: ...
```
Adapter les 3 call-sites dans `streamlit_app.py` (lambda ou `functools.partial`). Risque : modéré.

> ⚠️ **Prérequis test** : écrire `tests/test_normalize_mode_label.py` qui appelle la fonction
> **sans** `st.session_state` actif — doit passer avant de modifier la signature (sinon aucun
> filet de sécurité pour détecter une régression).

---

### 2.3 Item 2 — Supprimer `_normalize_mode_label` (dépend de §2.2)

Supprimer `def _normalize_mode_label` (L40–L44) de `teammates_helpers.py`. Remplacer par `normalize_mode_label` importée. Risque : faible après item 4.

---

### 2.4 Item 3 — Nettoyer `normalize_map_label` et `add_ui_columns` (après §2.0)

- Supprimer `add_ui_columns()` dans `filters.py` (L48 + caller L315) — ⏳ à faire
- ~~Supprimer les 7 hotfixes UI~~ — ✅ déjà fait (voir §2.0)
- Garder la guard UUID dans `normalize_map_label` (fallback légitime pour maps inédites)
- Supprimer la guard `_map_col` dans `compute_squad_records_per_map` [squad_records.py:165](src/analysis/squad_records.py#L165) → utiliser `"map_ui"` directement — ⏳ à faire
- **Nouveau** : supprimer la guard similaire dans `maps.py:89-90` (`"map_ui" if "map_ui" in df_pl.columns else "map_name"`) → utiliser `"map_ui"` directement — ⏳ oubliée dans le plan initial

---

### 2.5 Item 5 — Armes (DÉJÀ IMPLÉMENTÉ)

`resolve_weapon_display` dans `src/analysis/_weapon_data.py` est déjà DB-first. Barrer dans le backlog.

---

### 2.6 Item 6 — Cache `translate_playlist_name`

Vérifier si `build_mapping(dfr["playlist_name"], translate_playlist_name)` dans `cache_filters.py` est sous `@st.cache_data`. Si oui : item terminé. Sinon : placer le cache autour du `build_mapping`.

---

## 3. Analyse technique — Axe B (ChartData)

### 3.0 Vue d'ensemble de l'abstraction

L'objectif est de remplacer les 8–12 kwargs individuels de chaque fonction de chart escouade par un objet `ChartData` qui transporte toutes les données + comportements transversaux.

```
Aujourd'hui                         Avec ChartData
─────────────────────────           ──────────────────────────
plot_trio_metric(               →   build_chart_data(series, metric, ...)
  records, records_per_map,         .add_record_overlays(fig)
  colors_by_name, n_players,        .downsample(max_points=200)
  player_names, xs, …)              → n kwargs → 1 objet
```

### 3.0b Risque import circulaire à surveiller

`ChartData.add_record_overlays()` importe lazily (dans la méthode) depuis
`src.visualization._squad_record_shapes`. L'import lazy évite le cycle au chargement,
mais il faut vérifier que `_squad_record_shapes` n'importe **rien** depuis `_chart_series.py`
(ni directement ni transitif). À valider avant de commencer Phase 7a.

### 3.1 Nouveau fichier `src/visualization/_chart_series.py` (~90 L)

```python
from __future__ import annotations
from dataclasses import dataclass, field
from typing import Literal
import plotly.graph_objects as go

HEIGHT_COMPACT: int = 320      # consolide les magic numbers dans match_bars.py etc.
HEIGHT_NORMAL: int = 420
MAX_PLOT_POINTS: int = 200     # déplace _downsample_for_plot de timeseries.py

@dataclass
class MatchSeries:
    """Une série de données pour un joueur sur l'axe X commun de matchs."""
    name: str                        # nom du joueur (= offsetgroup Plotly)
    x: list[int]                     # positions entières normalisées 0..N-1
    y: list[float | None]            # valeurs métriques
    color: str                       # couleur hex du joueur
    map_names: list[str | None]      # carte à chaque position x (records par carte)

@dataclass
class ChartData:
    """Container pour un graphe multi-joueurs par match."""
    series: list[MatchSeries]
    x_labels: list[str]              # labels affichés (#N<br>Carte ou date)
    barmode: Literal["group", "overlay", "categorical"]
    global_records: dict[str, float | None] = field(default_factory=dict)
    per_map_records: dict[str, dict[str, float | None]] = field(default_factory=dict)

    @property
    def player_names(self) -> list[str]:
        return [s.name for s in self.series]

    @property
    def colors_by_name(self) -> dict[str, str]:
        return {s.name: s.color for s in self.series}

    @property
    def tick_step(self) -> int:
        n = len(self.x_labels)
        return max(1, n // 10) if n else 1

    def add_record_overlays(self, fig: go.Figure) -> None:
        """Dispatche vers la bonne implémentation selon barmode."""
        from src.visualization._squad_record_shapes import (
            add_record_shapes, add_overlay_record_shapes,
        )
        if self.barmode == "overlay":
            xs_ref = self.series[0].x if self.series else []
            add_overlay_record_shapes(
                fig, xs=xs_ref,
                records=self.global_records,
                player_names=self.player_names,
                colors_by_name=self.colors_by_name,
            )
        elif self.barmode == "categorical":
            # pm chart : ghost bars avec labels catégoriels
            _add_categorical_record_bars(fig, self)
        else:  # group
            for s in self.series:
                add_record_shapes(
                    fig, xs=s.x,
                    records={s.name: self.global_records.get(s.name)},
                    player_names=self.player_names,
                    n_players=len(self.series),
                    colors_by_name=self.colors_by_name,
                    per_map_records=(
                        {s.name: self.per_map_records[s.name]}
                        if s.name in self.per_map_records else None
                    ),
                    map_names_per_x=[
                        s.map_names[i] for i in range(len(s.x))
                        if i < len(s.map_names)
                    ],
                )

    def downsample(self, max_points: int = MAX_PLOT_POINTS) -> "ChartData":
        """Réduit le nombre de matchs affichés si trop de points pour Plotly."""
        n = max(len(s.x) for s in self.series) if self.series else 0
        if n <= max_points:
            return self
        step = max(1, n // max_points)
        new_series = [
            MatchSeries(
                name=s.name, color=s.color,
                x=s.x[::step], y=s.y[::step],
                map_names=s.map_names[::step],
            )
            for s in self.series
        ]
        return ChartData(
            series=new_series,
            x_labels=self.x_labels[::step],
            barmode=self.barmode,
            global_records=self.global_records,
            per_map_records=self.per_map_records,
        )
```

**`_add_categorical_record_bars`** (fonction interne) : ghost `go.Bar` avec `x=_pm_metrics` (labels string), extrait de la logique actuellement inline dans `_teammates_trio_helpers.py`.

### 3.2 Migration des 5 fonctions de chart

Chaque fonction reçoit un `ChartData` **en plus** des kwargs existants (rétrocompatibilité via `chart_data: ChartData | None = None`). Si `chart_data` est fourni, on l'utilise ; sinon comportement actuel inchangé.

**Étape finale (après migration complète) :** supprimer les anciens kwargs `records`, `records_per_map`, `colors_by_name`, `n_players`, `player_names` qui sont maintenant dans `ChartData`.

| Fonction | Fichier | Priorité |
|----------|---------|----------|
| `plot_trio_metric` | `trio.py` | 1 |
| `plot_trio_kills_deaths` | `trio.py` | 1 |
| `plot_multi_metric_bars_by_match` | `match_bars.py` | 2 |
| `plot_hs_pk_stacked` | `teammates_hs_pk.py` | 2 |
| `_render_per_minute_stats` | `_teammates_trio_helpers.py` | 3 |

### 3.3 Construction de `ChartData` dans la couche UI

`ChartData` est construit dans `_teammates_trio.py` (qui a déjà les séries, couleurs, records) et passé aux fonctions de chart. Les helpers `render_trio_charts` / `render_metric_bar_charts` dans `teammates_charts.py` transmettent l'objet sans le manipuler.

### 3.4 Items backlog absorbés par la Phase 7

| Item backlog | Résolution |
|-------------|-----------|
| `_downsample_for_plot` centralisé | `ChartData.downsample()` |
| PLR0913 (trop d'arguments) | `ChartData` remplace les kwargs individuels |
| Magic numbers `HEIGHT_COMPACT = 320` | Constantes dans `_chart_series.py` |

### 3.5 Scope futur (Phase 8 — optionnel)

Les graphes solo de la page Timeseries (§3 de `CHARTS_AND_TABLES.md`) suivent le même pattern "barres par match" mais sans dimension multi-joueurs. Une variante `SingleSeriesChartData` pourrait centraliser le downsampling et les annotations de max. **Non prioritaire** — la valeur principale est sur la page Escouade.

---

## 4. Résumé des actions

| # | Axe | Item | Fichiers modifiés | Effort | Risque | État |
|---|-----|------|-------------------|--------|--------|------|
| **0** | A | **`src/app/i18n_columns.py`** | `i18n_columns.py`, `main_helpers.py` | **1h** | **Modéré** | ✅ Fait |
| 1 | A | `src/utils/strings.py` + `is_uuid_like` unifié | `strings.py` (new), `translations.py`, `helpers.py` | 15 min | Nul | ⏳ |
| 4 | A | Découpler `normalize_mode_label` de `st.session_state` | `helpers.py`, `streamlit_app.py` (3 sites) | 45 min | Modéré | ⏳ |
| 2 | A | Supprimer `_normalize_mode_label` (après item 4) | `teammates_helpers.py` | 10 min | Faible | ⏳ |
| 3 | A | Supprimer `add_ui_columns` + guards `_map_col` / `maps.py` | 3 fichiers | 30 min | Faible | ⏳ |
| 5 | A | **Aucun code** — valider + barrer le backlog | — | 5 min | Nul | ✅ Fait |
| 6 | A | Vérifier cache `build_mapping` dans `cache_filters.py` | — | 15 min | Nul | ✅ Fait |
| **7a** | B | **Créer `src/visualization/_chart_series.py`** | `_chart_series.py` (new) | **1h** | **Faible** | ⏳ |
| 7b | B | Migrer `plot_trio_metric` + `plot_trio_kills_deaths` | `trio.py`, `teammates_charts.py` | 1h30 | Modéré | ⏳ |
| 7c | B | Migrer `plot_multi_metric_bars_by_match` | `match_bars.py` | 45 min | Modéré | ⏳ |
| 7d | B | Migrer `plot_hs_pk_stacked` | `teammates_hs_pk.py` | 30 min | Faible | ⏳ |
| 7e | B | Migrer `_render_per_minute_stats` + ghost bars catégoriels | `_teammates_trio_helpers.py` | 45 min | Modéré | ⏳ |
| 7f | B | Supprimer anciens kwargs (après 7b-7e) | 5 fichiers | 30 min | Faible | ⏳ |

**Total estimé restant : ~5h** (Phases 1→4 ~40min + TeammatesService ~15min + Phase 7 ~4h20, Phases 0/5/6 déjà faites).

---

## 5. Checklist d'implémentation

### Phase 0 — Couche centralisée `src/app/i18n_columns.py` ✅ COMPLÉTÉE

- [x] Créer `src/app/i18n_columns.py` avec `add_i18n_display_columns(df, lang="fr")` — idempotente, 0 import Streamlit
- [x] Dans `main_helpers.py:375` : appelée juste après `load_df_optimized()`
- [x] Hotfixes UI dans `win_loss.py`, `teammates_views.py`, `friends_impact_heatmap.py`, `_teammates_trio.py`, requêtes SQL teammates — supprimés
- [ ] `TeammatesService.load_teammate_stats` (`src/data/services/teammates_service.py`) : la requête SQL inclut déjà `map_name_fr` et `pair_name_fr` (L111-112), mais `add_i18n_display_columns` n'est **pas** appelée sur le df retourné → `map_ui` absent. À corriger : appeler `add_i18n_display_columns(df, lang)` avant le `return` dans cette méthode.
- [ ] Écrire `tests/test_i18n_columns.py` : `map_ui` = `map_name_fr` si présent, fallback sinon, idempotence
- [ ] Tests → vert

### Phase 1 — `src/utils/strings.py`

- [ ] Créer `src/utils/strings.py` avec `is_uuid_like`
- [ ] `translations.py` : supprimer `_is_uuid_like`, importer depuis `src.utils.strings`
- [ ] `helpers.py` : supprimer définition locale, importer
- [ ] Tests → vert

### Phase 2 — Découpler `normalize_mode_label` (Item 4)

- [ ] **Prérequis** : écrire `tests/test_normalize_mode_label.py` sans `st.session_state` actif — doit passer avant toute modification
- [ ] Modifier signature dans [helpers.py:55](src/app/helpers.py#L55) : ajouter `lang: str = "fr"` et `normalize: bool = True`
- [ ] Supprimer les lignes `_settings = st.session_state...` ([helpers.py:71-72](src/app/helpers.py#L71))
- [ ] Adapter les 3 call-sites dans `streamlit_app.py`
- [ ] Vérifier `cache_filters.py` pour usage éventuel
- [ ] Tests → vert

### Phase 3 — Supprimer `_normalize_mode_label` (Item 2)

- [ ] Supprimer `def _normalize_mode_label` (L40–L44) dans `teammates_helpers.py`
- [ ] L203 : remplacer par `lambda p: normalize_mode_label(p, lang=..., normalize=...)`
- [ ] Tests → vert

### Phase 4 — Nettoyer code mort (Item 3, après Phase 0)

- [x] ~~Supprimer le patch dans `win_loss.py` L261~~ — déjà fait
- [x] ~~Supprimer le patch dans `teammates_views.py` L289~~ — déjà fait
- [x] ~~Supprimer le patch dans `friends_impact_heatmap.py`~~ — déjà fait
- [x] ~~Supprimer le patch `_me_full` dans `_teammates_trio.py`~~ — déjà fait
- [x] ~~Supprimer les alias SQL `map_ui` de `_query_teammate_shared_stats`~~ — déjà fait
- [x] ~~Supprimer les alias SQL `map_ui` de `query_teammate_full_history`~~ — déjà fait
- [ ] Supprimer `add_ui_columns()` dans [filters.py:48](src/app/filters.py#L48) + caller L315
- [ ] Supprimer la guard `_map_col` dans [squad_records.py:165](src/analysis/squad_records.py#L165) → `"map_ui"` direct
- [ ] **Nouveau** : supprimer la guard dans [maps.py:89-90](src/analysis/maps.py#L89) → `"map_ui"` direct
- [ ] Tests → vert

### Phase 5 — Validation armes (Item 5) ✅ COMPLÉTÉE

- [x] `src/analysis/_weapon_data.py:265` — `resolve_weapon_display` est bien DB-first (confirmé : appel `_resolve_weapon_cached()` en premier, fallback dicts Python)
- [x] Barrer item 5 dans `.ai/BACKLOG.md`

### Phase 6 — Cache playlist (Item 6) ✅ COMPLÉTÉE

- [x] Confirmé : `build_mapping(dfr["playlist_name"], translate_playlist_name)` dans `cache_filters.py:102` est appelée à l'intérieur de `_translate_playlist_pair_columns` (L95) → appelée par `_build_friend_df_from_match_ids_v4` → appelée par `cached_friend_matches_df` décorée `@st.cache_data`. Le cache est en place via la hiérarchie d'appels.
- [x] Barrer item 6 dans `.ai/BACKLOG.md`

### Phase 7 — Abstraction ChartData ⭐ AXE B

- [ ] **Prérequis** : vérifier que `_squad_record_shapes.py` n'importe rien depuis `_chart_series.py` (risque import circulaire — voir §3.0b)
- [ ] **7a** — Créer `src/visualization/_chart_series.py` avec `MatchSeries`, `ChartData`, `HEIGHT_COMPACT`, `HEIGHT_NORMAL`, `MAX_PLOT_POINTS`, `_add_categorical_record_bars`
- [ ] Écrire `tests/test_chart_series.py` : `add_record_overlays` dispatche correctement, `downsample` réduit les points, idempotence sur ChartData sans records
- [ ] **7b** — Migrer `plot_trio_metric` et `plot_trio_kills_deaths` (`trio.py`) : accepter `chart_data: ChartData | None = None` en plus des kwargs actuels ; `add_record_overlays` remplace les appels directs à `add_record_shapes`
- [ ] Adapter `_plot_trio_metric_chart` + `render_trio_charts` dans `teammates_charts.py` pour construire et passer `ChartData`
- [ ] **7c** — Migrer `plot_multi_metric_bars_by_match` (`match_bars.py`) : idem pattern ; `ChartData.downsample()` remplace éventuel futur downsampling inline
- [ ] **7d** — Migrer `plot_hs_pk_stacked` (`teammates_hs_pk.py`) : `barmode="overlay"`, `add_record_overlays` dispatche vers `add_overlay_record_shapes`
- [ ] **7e** — Migrer `_render_per_minute_stats` (`_teammates_trio_helpers.py`) : extraire les ghost bars catégoriels vers `_add_categorical_record_bars` dans `_chart_series.py` ; le chart reçoit un `ChartData(barmode="categorical")`
- [ ] **7f** — Supprimer les anciens kwargs `records`, `records_per_map`, `colors_by_name`, `n_players`, `player_names` des 5 fonctions (remplacés par `ChartData`)
- [ ] Vérifier taille `teammates_charts.py` < 500 L après migration
- [ ] Tests → vert (notamment `test_visualizations.py` + `test_squad_record_shapes.py`)

---

## 6. Contraintes architecture à respecter

- **Pandas PROSCRIT** — tout reste Polars
- **`src/app/i18n_columns.py`** — 0 import Streamlit, testable en isolation
- **`src/visualization/_chart_series.py`** — 0 import Streamlit, 0 accès DB, testable unitairement
- **`st.session_state` interdit dans `src/app/helpers.py`** après Phase 2
- **Pas de guard `_has_shared_view`** — les vues v6 sont garanties présentes
- **Tests obligatoires après chaque phase** : `python -m pytest tests/ -q --ignore=tests/integration --ignore=tests/e2e`
- **Taille** : nouveaux modules < 500 L, fonctions < 80 L
- **Rétrocompatibilité Phase 7** : `chart_data: ChartData | None = None` pendant la transition — supprimer en 7f seulement une fois tous les callers migrés

---

## 7. Branche Git

```bash
git checkout -b refactor/asset-translations-db-first
```

Commits séquentiels par phase :
```
feat(i18n): créer src/app/i18n_columns.py — couche centralisée map_ui/playlist_ui/mode_ui
feat(utils): ajouter src/utils/strings.py avec is_uuid_like unifié
refactor(helpers): découpler normalize_mode_label de st.session_state
refactor(teammates): supprimer _normalize_mode_label, utiliser normalize_mode_label
refactor(ui): supprimer add_ui_columns + rustines symptomatiques (code mort i18n)
docs(backlog): barrer items 5 et 6 (déjà implémentés / vérifiés)
feat(viz): créer _chart_series.py — MatchSeries, ChartData, downsample, record overlays
refactor(viz): migrer trio.py vers ChartData (plot_trio_metric, plot_trio_kills_deaths)
refactor(viz): migrer match_bars.py + teammates_hs_pk.py vers ChartData
refactor(viz): migrer _render_per_minute_stats vers ChartData (barmode=categorical)
refactor(viz): supprimer anciens kwargs records/colors (migration ChartData complète)
```

---

## 8. Plan V2 — Extensions post-V1

> **Prérequis** : V1 Phases 0→7f complètes. V2 s'exécute sur une nouvelle branche depuis `main`.
> **Date de rédaction** : 2026-04-04

### Vue d'ensemble V2

La V1 centralise les colonnes i18n et structure les données des graphes escouade.
La V2 tire les conséquences logiques de ces deux axes et adresse les violations de taille actives.

| Axe | Problème | Dépendance V1 | Effort | Valeur |
|-----|---------|--------------|--------|--------|
| **C** — Éliminer injection callback fn | `normalize_*_fn` transite encore comme Callable dans **49 sites** (7 fichiers) | Phase 2 terminée | ~2h | ⭐⭐⭐ |
| **D** — `mode_ui` centralisé + `_add_derived_columns` démantelée | God function noqa: C901/PLR0912, logique dupliquée dans `_filters_cascade` | Phase 2 + Axe C | ~2h | ⭐⭐⭐ |
| **E** — Résorber les 3 modules > 500L | Violations actives : `maps_outcome.py` 590L, `friends_impact_heatmap.py` 507L, `timeseries.py` 505L | Indépendant | ~3h | ⭐⭐ |
| **E′** — Surveillance préventive modules 450–500L | `session_compare_charts.py` 498L, `match_view_helpers.py` 495L, `teammates_charts.py` 491L (grossira avec ChartData) | V1 Phase 7 | — (monitorer) | ⭐ |
| **F** — ChartData solo timeseries | `_rolling_mean` import privé cross-module, magic numbers height incohérents, pas de downsampling centralisé | V1 Phase 7 + Axe E | ~2h | ⭐ |

**Total estimé V2 : ~9h** (C ~2h + D ~2h + E ~3h + F ~2h).

---

### Axe C — Éliminer le pattern callback fn injection

#### Diagnostic

Après V1 Phase 2, `normalize_mode_label(pair_name, lang=lang)` et `normalize_map_label` sont des fonctions pures. Pourtant elles continuent de voyager comme callables injectées à travers toute la pile :

```
streamlit_app.py
  → page_router.py           normalize_mode_label_fn: Callable
    → FilterSidebarCallbacks normalize_mode_label_fn, normalize_map_label_fn
      → filters_render.py    callbacks["normalize_mode_label_fn"](...)
        → MatchViewParams    normalize_mode_label_fn: Callable[[str], str]
          → match_view.py, explorer.py
```

**Sites effectifs identifiés (49 occurrences, 7 fichiers) :**

| Fichier | Occurrences | Nature |
|---------|-------------|--------|
| `src/app/_filters_apply.py` | 16 | fallback `_identity`, transmission, utilisation |
| `src/app/filters_render.py` | 10 | extraction depuis callbacks dict + transmission |
| `src/app/_filters_cascade.py` | 8 | `build_mapping(...)` avec la fn injectée |
| `src/app/_page_context.py` | 3 | typage TypedDict |
| `src/ui/pages/explorer.py` | 5 | extraction + appel |
| `src/ui/pages/match_view.py` | 5 | propagation + appel |
| `src/app/page_router.py` | 2 | construction du dict |

> **Écart vs estimation initiale** : 49 occurrences réelles vs 28 annoncées (+75%). L'effort ~2h reste valide car les sites sont mécaniques (find/replace guidé par les TypedDict), mais la revue est plus large qu'anticipée.

#### Solution

Remplacer l'injection de callbacks par des appels directs avec `lang` explicite :

```python
# Avant (V1)
params["normalize_mode_label_fn"](pair_name)

# Après (V2)
from src.app.helpers import normalize_mode_label
normalize_mode_label(pair_name, lang=get_lang())
```

- Supprimer `normalize_mode_label_fn` et `normalize_map_label_fn` de `FilterSidebarCallbacks` et `MatchViewParams`
- Supprimer le fallback `_identity` dans `_filters_apply.py` (L77–80) — appel direct
- Supprimer les champs correspondants dans `_page_context.py`
- Adapter `page_router.py` : ne plus construire ni passer ces clés

**Gain collatéral** : `apply_filters` (noqa: C901, PLR0912, PLR0913, PLR0915) perd 2 de ses 4 violations PLR0913 une fois les 2 callbacks retirés de sa signature.

---

### Axe D — Centraliser `mode_ui` et démanteler `_add_derived_columns`

#### Diagnostic

`mode_ui` est calculé dans deux endroits séparés :

| Fichier | Fonction | Violation |
|---------|---------|-----------|
| `src/app/_filters_apply.py:267` | `_add_derived_columns` | `# noqa: C901, PLR0912` |
| `src/app/_filters_cascade.py:170` | `_vectorize_ui_columns` | version **simplifiée** (59L vs 135L) : calcule seulement `playlist_ui`, `mode_ui`, `map_ui` sans `playlist_fr` ni `pair_fr`, avec logique facettée intégrée |

Après Axe C, `normalize_mode_label` est pure → `mode_ui` peut rejoindre `i18n_columns.py` via `coalesce([pair_name_fr, pair_name])` + normalisation, exactement comme `map_ui`.

#### Solution en 3 étapes

**D1 — Ajouter `mode_ui` dans `add_i18n_display_columns`**
```python
# Dans i18n_columns.py
if "mode_ui" not in df.columns and "pair_name" in df.columns:
    if lang == "fr" and "pair_name_fr" in df.columns:
        exprs.append(pl.coalesce([...]).alias("mode_ui"))
    # + normalize via map_elements(normalize_mode_label, lang=lang)
```
(Note : `pair_name_fr` est le nom brut asset, la normalisation reste nécessaire — mais elle est maintenant pure.)

**D2 — Décomposer `_add_derived_columns` → 3 fonctions à responsabilité unique**

```python
# Extrait de _add_derived_columns (noqa: C901, PLR0912 → supprimés)
def _compute_map_ui_column(dff, normalize_map_label_fn, lang) -> pl.Expr: ...
def _compute_mode_ui_column(dff, lang) -> pl.Expr: ...       # plus besoin de fn injectée
def _compute_playlist_ui_column(dff, lang) -> pl.Expr: ...
```

**D3 — Supprimer `_vectorize_ui_columns` dans `_filters_cascade.py`**
⚠️ N'est pas une copie identique de `_add_derived_columns` — c'est une version simplifiée (59L vs 135L) qui ne calcule pas `playlist_fr` / `pair_fr`. La remplacer par `add_i18n_display_columns(dropdown_base, lang)` couvre les colonnes UI (`map_ui`, `playlist_ui`) mais il faudra vérifier que `mode_ui` (D1) est aussi produit avant de supprimer `_vectorize_ui_columns`.

**D4 — Déplacer `_rolling_mean` vers `_timeseries_helpers.py`**
Actuellement défini dans `timeseries.py:27` et importé comme private depuis `timeseries_combat.py:23` (`from src.visualization.timeseries import _rolling_mean`). Import d'un symbole privé cross-module = dette. Le déplacer dans `_timeseries_helpers.py` où il rejoint `apply_chrono_xaxis` et `prepare_time_axis`.

---

### Axe E — Résorber les 3 modules > 500L (violations actives)

#### `maps_outcome.py` (590L) → split data / rendu

| Nouveau fichier | Contenu | Lignes estimées |
|----------------|---------|----------------|
| `src/visualization/_maps_outcome_data.py` | `_perf_color`, `_sort_by_map_order`, `_empty_map_figure`, `_prepare_timeline_df`, `_add_timeline_traces`, `_prepare_bullet_joined_data` | ~170L |
| `src/visualization/maps_outcome.py` | `plot_map_lollipop`, `plot_map_outcome_timeline`, `_add_bullet_bar_traces`, `_add_bullet_color_legend_traces`, `_add_bullet_overlay_traces`, `plot_map_winrate_bullet`, `plot_map_perf_vs_history` | ~420L |

Les 2 noqa PLR0913 dans `_add_bullet_bar_traces` et `_add_bullet_overlay_traces` sont liées à la nature intrinsèque des fonctions Plotly (nombreux paramètres de style) — les conserver avec annotation.

#### `friends_impact_heatmap.py` (507L) → split data / rendu

| Nouveau fichier | Contenu | Lignes estimées |
|----------------|---------|----------------|
| `src/visualization/_heatmap_data.py` | `build_impact_ranking_df`, `count_events_by_player`, `_top_maps_by_frequency`, `_order_maps_by_first_seen`, `_build_perf_matrix`, `_discrete_perf_colorscale` | ~175L |
| `src/visualization/friends_impact_heatmap.py` | `plot_friends_impact_heatmap` (noqa: C901, PLR0912, PLR0915 — complexité inhérente aux heatmaps multi-axes), `render_impact_summary_stats`, `plot_squad_map_heatmap` | ~330L |

#### `timeseries.py` (505L) → extraction helpers

Les helpers utilitaires migrent vers `_timeseries_helpers.py` (déjà existant) :

| Symbole | Source | Destination |
|---------|--------|------------|
| `_normalize_df` | `timeseries.py:22` | `_timeseries_helpers.py` |
| `_rolling_mean` | `timeseries.py:27` (cf. Axe D4) | `_timeseries_helpers.py` |
| `_build_kda_customdata` | `timeseries.py:43` | `_timeseries_helpers.py` |
| `_add_kda_traces` | `timeseries.py:68` | `_timeseries_helpers.py` |
| `_add_permin_rolling_lines` | `timeseries.py:276` | `_timeseries_helpers.py` |

Résultat : `timeseries.py` ne garde que les 4 fonctions publiques (`plot_timeseries`, `plot_assists_timeseries`, `plot_per_minute_timeseries`, `plot_accuracy_last_n`) → ~270L.

#### Périmètre de l'Axe E — modules hors scope (baseline connue)

`size_baseline.txt` documente les violations connues et acceptées en dehors du domaine visualisation/UI. Ces modules NE sont PAS dans le scope de ce plan :

| Module | Lignes (baseline) | Domaine | Raison hors scope |
|--------|-----------------|---------|------------------|
| `src/data/sync/engine.py` | 688L | sync | Architecture mixins déjà en place |
| `src/data/sync/transformers/_match.py` | 577L | sync | Transformers données brutes API |
| `src/data/repositories/_weapon_kills_repo.py` | 567L | repo | Domaine persistence |
| `src/data/sync/_match_processing.py` | 559L | sync | — |
| `src/data/sync/api_client.py` | 557L | sync | — |
| `src/data/sync/transformers/_helpers.py` | 515L | sync | — |
| `src/data/sync/_engine_connections.py` | 510L | sync | — |
| `src/data/services/teammates_service.py` | 537L | services | Domaine service (dette connue) |
| `src/analysis/weapon_parser.py` | 502L | analysis | Domaine analyse armes |

> Ces modules sont documentés dans `scripts/size_baseline.txt` comme dette technique connue. Les réduire est souhaitable mais relève d'un plan de refacto dédié (`refactor/sync-engine-cleanup`, etc.), pas de ce plan.

> **Note** : `src/visualization/distributions.py` contient `import pandas as pd` mais **sous garde `TYPE_CHECKING`** — c'est le pattern correct pour les annotations de type uniquement, pas une violation.

#### Modules 450–500L à surveiller (Axe E′)

Ces modules ne violent pas encore la règle mais sont à risque. Ceux marqués (baseline) sont déjà documentés comme dette connue :

| Module | Lignes | Baseline | Risque | Action préventive |
|--------|--------|----------|--------|-----------------|
| `session_compare.py` | 538L | ✅ oui | Déjà dépassé | Split `_session_compare_kpi.py` lors du prochain touch |
| `session_compare_charts.py` | 498L | non | Élevé — 1 commit peut dépasser | Split `_session_compare_annotations.py` prêt |
| `match_view_helpers.py` | 495L | non | Élevé | Split `_match_view_kpi.py` si dépasse |
| `teammates_charts.py` | 491L | non | Élevé — grossira avec ChartData | Mesurer après V1 Phase 7 ; split si > 500L |
| `match_impact_timeline.py` | 482L | non | Modéré — 2 god functions (C901+PLR0912+PLR0913+PLR0915) | `_match_impact_trace_builders.py` si dépasse |
| `timeseries_combat.py` | 481L | non | Modéré | `_timeseries_combat_helpers.py` si dépasse |
| `_perf_progression.py` | 476L | non | Faible | — |
| `_perf_session.py` | 470L | non | Faible | — |

> `match_impact_timeline.py` mérite une attention particulière : `plot_match_kill_death_timeline` et `plot_all_players_frags_timeline` ont toutes deux 3–4 violations noqa simultanées (C901, PLR0912, PLR0913, PLR0915). Non adressé dans V1. Si le fichier dépasse 500L, extraire les builders de traces dans `_match_impact_trace_builders.py`.

---

### Axe F — Étendre ChartData aux graphes solo timeseries

#### Diagnostic

Les 4 graphes timeseries solo (`plot_timeseries`, `plot_assists_timeseries`, `plot_per_minute_timeseries`, `plot_accuracy_last_n`) partagent les mêmes besoins non centralisés :

| Besoin | Situation actuelle |
|--------|------------------|
| Rolling mean (lissage) | `_rolling_mean` avec `window=10` codé à chaque call-site |
| Magic numbers height | `height=420` × 4 dans `timeseries_combat.py`, `height=400` × 2 dans `_timeseries_progression.py` — **incohérents** |
| Downsampling | Absent des graphes solo — seul `_downsample_for_plot` existe dans `timeseries.py` et n'est pas réutilisé |

> Note : les graphes solo utilisent un axe X chronologique (temps), pas des positions entières par match. `SingleSeriesChartData` sera une **dataclass séparée** de `ChartData` (pas d'héritage), plus simple.

#### Solution

**F1 — Harmoniser les magic numbers height**

Ajouter dans `_chart_series.py` (déjà créé en V1) :
```python
HEIGHT_TIMESERIES: int = 420   # graphes combat, timeseries standard
HEIGHT_PROGRESSION: int = 400  # courbes LUSR/performance (plus compact)
HEIGHT_MINI: int = 150         # mini-charts (participation_charts_extra.py)
```
Remplacer les 6 occurrences inline dans `timeseries_combat.py` et `_timeseries_progression.py`.

**F2 — `SingleSeriesChartData` dans `_chart_series.py`**

```python
@dataclass
class SingleSeriesChartData:
    """Container pour un graphe solo (KDA, per-minute, etc.)."""
    x: list[Any]                    # positions ou timestamps
    y: list[float | None]           # valeurs métriques
    y_smooth: list[float | None]    # rolling mean pré-calculée
    height: int = HEIGHT_TIMESERIES
    title: str = ""

    @classmethod
    def from_series(
        cls,
        x: list[Any],
        y: list[float | None],
        window: int = 10,
        **kwargs,
    ) -> "SingleSeriesChartData":
        """Construit avec rolling mean pré-calculée."""
        smooth = _rolling_mean_list(y, window)
        return cls(x=x, y=y, y_smooth=smooth, **kwargs)
```

**Graphes solo concernés :**

| Graphe | Fichier | window | Gain |
|--------|---------|--------|------|
| `plot_timeseries` | `timeseries.py` | 10 | supprime `_rolling_mean` inline |
| `plot_assists_timeseries` | `timeseries.py` | 10 | idem |
| `plot_per_minute_timeseries` | `timeseries.py` | 10 | idem × 3 traces |
| `plot_shots_accuracy` | `timeseries_combat.py` | 10 | idem |
| `plot_damage_dealt_taken` | `timeseries_combat.py` | 10 | idem |
| `plot_spree_headshots_accuracy` | `timeseries_combat.py` | 10 | idem |
| `plot_average_life` | `timeseries_combat.py` | 10 | idem |

> `plot_accuracy_last_n` (statique, pas de rolling) : compatible `SingleSeriesChartData` mais sans `y_smooth`.

---

## 9. Checklist V2

### Phase C — Éliminer injection callback fn

- [ ] Écrire test `tests/test_normalize_without_injection.py` : appels directs sans session_state
- [ ] Supprimer `normalize_mode_label_fn` + `normalize_map_label_fn` de `FilterSidebarCallbacks` et `MatchViewParams` (`_page_context.py`)
- [ ] Supprimer construction des clés dans `page_router.py:101+121`
- [ ] Adapter `filters_render.py` : appels directs `normalize_mode_label(x, lang=get_lang())`
- [ ] Adapter `_filters_apply.py` : supprimer fallback `_identity` L77–80, appels directs
- [ ] Adapter `_filters_cascade.py` : idem, supprimer params L344–345
- [ ] Adapter `explorer.py` + `match_view.py` : appels directs
- [ ] Vérifier que `apply_filters` perd ses violations PLR0913 (signature plus courte)
- [ ] Tests → vert

### Phase D — Centraliser mode_ui + démanteler _add_derived_columns

- [ ] **D1** — Ajouter `mode_ui` dans `add_i18n_display_columns` (`i18n_columns.py`)
- [ ] Mettre à jour `tests/test_i18n_columns.py` : `mode_ui` calculé correctement en fr/en
- [ ] **D2** — Extraire `_compute_map_ui_column`, `_compute_mode_ui_column`, `_compute_playlist_ui_column` depuis `_add_derived_columns` → supprimer noqa C901/PLR0912
- [ ] **D3** — Supprimer `_vectorize_ui_columns` dans `_filters_cascade.py` → remplacer par `add_i18n_display_columns`
- [ ] **D4** — Déplacer `_rolling_mean` de `timeseries.py:27` vers `_timeseries_helpers.py` ; adapter imports dans `timeseries_combat.py` et `_timeseries_progression.py`
- [ ] Tests → vert

### Phase E — Résorber modules > 500L

- [ ] **E1** — Créer `src/visualization/_maps_outcome_data.py` : extraire les 6 fonctions data de `maps_outcome.py`
- [ ] Vérifier `maps_outcome.py` < 500L après extraction
- [ ] **E2** — Créer `src/visualization/_heatmap_data.py` : extraire les 6 fonctions data de `friends_impact_heatmap.py`
- [ ] Vérifier `friends_impact_heatmap.py` < 500L après extraction
- [ ] **E3** — Extraire `_normalize_df`, `_build_kda_customdata`, `_add_kda_traces`, `_add_permin_rolling_lines` vers `_timeseries_helpers.py` (+ `_rolling_mean` depuis D4)
- [ ] Vérifier `timeseries.py` < 500L après extraction
- [ ] Vérifier `_timeseries_helpers.py` < 500L après ajouts
- [ ] Mesurer `teammates_charts.py` après V1 Phase 7 — split si > 500L
- [ ] Tests → vert

### Phase F — ChartData solo timeseries

- [ ] Ajouter `HEIGHT_TIMESERIES = 420`, `HEIGHT_PROGRESSION = 400`, `HEIGHT_MINI = 150` dans `_chart_series.py`
- [ ] Remplacer les 6 magic numbers height dans `timeseries_combat.py` et `_timeseries_progression.py`
- [ ] Créer `SingleSeriesChartData` + `from_series()` dans `_chart_series.py`
- [ ] Mettre à jour `tests/test_chart_series.py` : `from_series` pré-calcule le rolling, `HEIGHT_*` constantes exportées
- [ ] Migrer les 7 graphes solo listés en §Axe F vers `SingleSeriesChartData`
- [ ] Tests → vert

---

## 10. Branche Git V2

```bash
git checkout main
git pull
git checkout -b refactor/viz-pipeline-v2
```

Commits séquentiels :
```
refactor(app): éliminer injection callback fn normalize_*_label (Axe C)
feat(i18n): ajouter mode_ui dans add_i18n_display_columns (Axe D1)
refactor(filters): démanteler _add_derived_columns → 3 fonctions (Axe D2)
refactor(filters): supprimer _vectorize_ui_columns (Axe D3)
refactor(viz): déplacer _rolling_mean vers _timeseries_helpers (Axe D4)
refactor(viz): split maps_outcome.py → _maps_outcome_data.py (Axe E1)
refactor(viz): split friends_impact_heatmap.py → _heatmap_data.py (Axe E2)
refactor(viz): split timeseries.py → helpers extraits (Axe E3)
feat(viz): HEIGHT_* constantes + SingleSeriesChartData (Axe F)
refactor(viz): migrer graphes solo timeseries vers SingleSeriesChartData (Axe F)
```
