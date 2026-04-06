# Plan refacto visualization — Items restants

> Créé le 2026-04-06  
> Statut global : **En attente**  
> Branche cible : `refactor/viz-cleanup`

---

## ⚠️ Prérequis avant de démarrer

La branche courante (`refactor/sessions-perf`) contient des **modifications non commitées** (feature cadence histogramme). Ces changes sont sans rapport avec ce plan. **Committer ou stasher ces changes avant de créer `refactor/viz-cleanup`**, sinon risque de mélanger les sujets sur la mauvaise branche.

```bash
git add <fichiers cadence>
git commit -m "feat(cadence): ..."
git checkout -b refactor/viz-cleanup
```

---

## Ordre d'exécution recommandé

```
1. #4  spartan_carnage backfill   (CLI, 5 min)
2. #10 PLR0913 dataclasses        (1h)       ← avant #9 : évite 2 passes sur timeseries_combat.py
3. #9  magic numbers hauteur      (import, 15 min)
4. #1  supprimer title=           (15 min)
5. #7  downsample centralisation  (45 min)
6. #8  maps_outcome split         (1h)
7. #2  migration SingleSeriesChartData  (plusieurs sessions)
```

---

## #4 — Backfill `spartan_carnage` *(5 min — CLI uniquement)*

Aucun code à modifier. Le mapping `citation_mappings` est déjà corrigé en DB.

**Prérequis** : Streamlit arrêté (fichier `metadata.duckdb` verrouillé sinon).

```bash
python scripts/backfill_data.py --all --citations --force-citations
```

**Validation** : vérifier que les lignes `match_citations` avec `citation_name_norm = 'spartan_carnage'`
contiennent désormais la somme des médailles de spree (pas `max_killing_spree` brut).

---

## #9 — Magic numbers de hauteur *(15 min)*

Les constantes existent déjà dans `src/visualization/_chart_series.py` :
- `HEIGHT_COMPACT = 320`
- `HEIGHT_TIMESERIES = 420`
- `HEIGHT_PROGRESSION = 400`

> **⚠️ Attention taille** : `timeseries_combat.py` est à **473L** (limite 500L). #7 va encore y ajouter des lignes.
> Faire #10 **avant** #9 pour potentiellement réduire ce fichier d'abord. Vérifier `wc -l` avant de committer.

### Fichiers à modifier

**`src/visualization/match_bars.py`** (2 occurrences) :
```python
# Avant
return apply_halo_plot_style(fig, height=320)

# Après — ajouter en tête : from src.visualization._chart_series import HEIGHT_COMPACT
return apply_halo_plot_style(fig, height=HEIGHT_COMPACT)
```

**`src/visualization/timeseries_combat.py`** (4 occurrences) :
```python
# Avant
fig.update_layout(height=420, ...)
return apply_halo_plot_style(fig, height=420)

# Après — ajouter en tête : from src.visualization._chart_series import HEIGHT_TIMESERIES
fig.update_layout(height=HEIGHT_TIMESERIES, ...)
return apply_halo_plot_style(fig, height=HEIGHT_TIMESERIES)
```

**`src/visualization/_timeseries_progression.py`** (2 occurrences) :
```python
# Avant
fig.update_layout(height=400, ...)
return apply_halo_plot_style(fig, height=400)

# Après — ajouter en tête : from src.visualization._chart_series import HEIGHT_PROGRESSION
fig.update_layout(height=HEIGHT_PROGRESSION, ...)
return apply_halo_plot_style(fig, height=HEIGHT_PROGRESSION)
```

**Validation** : `grep -rn "height=320\|height=420\|height=400" src/visualization/` → 0 résultat.

---

## #1 — Supprimer `title=` de `apply_halo_plot_style` *(15 min)*

**Prérequis** : confirmer que 0 caller actif — `grep -rn "apply_halo_plot_style.*title=" src/` → vide ✅

> **État partiel** : la docstring `title:` a déjà été retirée de `_timeseries_progression.py` (fait dans les changes cadence).
> Il reste à nettoyer `theme.py` : param, DeprecationWarning, bloc `if title is not None`.

### Fichier : `src/visualization/theme.py`

Supprimer :
1. Le param `title: str | None = None` de la signature
2. La ligne docstring dédiée à `title`
3. Le bloc `DeprecationWarning` (import `warnings` si plus utilisé ailleurs)
4. La ligne `if title is not None: fig.update_layout(title=title)`

```python
# Avant
def apply_halo_plot_style(
    fig: go.Figure,
    *,
    title: str | None = None,
    height: int | None = None,
) -> go.Figure:
    if title is not None and title != "":
        warnings.warn(...)
    ...
    if title is not None:
        fig.update_layout(title=title)

# Après
def apply_halo_plot_style(
    fig: go.Figure,
    *,
    height: int | None = None,
) -> go.Figure:
    ...
    # (supprimer les deux blocs title)
```

**Validation** : `python -m pytest tests/ -q --ignore=tests/integration` → vert.

---

## #10 — PLR0913 dans `_perf_progression.py` *(1h)*

### Fichier : `src/visualization/_plot_options.py`

Ajouter deux dataclasses après `PlotOptions` :

```python
@dataclass
class KdCiData:
    """Données pour les traces CI de la courbe KD cumulée."""
    x: list
    y_cumul: list
    y_upper: list
    y_lower: list
    y_match: list


@dataclass
class EwmaData:
    """Données pour les traces EWMA KD."""
    x: list
    y_kd: list
    y_ewma: list
    regression_data: dict | None
```

### Fichier : `src/visualization/_perf_progression.py`

**`_add_kd_ci_traces`** (actuellement 7 params) :
```python
# Avant
def _add_kd_ci_traces(  # noqa: PLR0913
    fig, x_values, y_cumul, y_upper, y_lower, y_match, lang
) -> None:

# Après
def _add_kd_ci_traces(fig: go.Figure, data: KdCiData, lang: str) -> None:
    # remplacer x_values → data.x, y_cumul → data.y_cumul, etc.
```

**`_add_ewma_traces`** (actuellement 6 params) :
```python
# Avant
def _add_ewma_traces(  # noqa: PLR0913
    fig, x_values, y_kd, y_ewma, regression_data, lang
) -> None:

# Après
def _add_ewma_traces(fig: go.Figure, data: EwmaData, lang: str) -> None:
    # remplacer x_values → data.x, y_kd → data.y_kd, etc.
```

Mettre à jour les call-sites (construire `KdCiData(...)` et `EwmaData(...)` avant l'appel).  
Supprimer les deux `# noqa: PLR0913`.

**Validation** : `python -m ruff check src/visualization/_perf_progression.py` → 0 violation PLR0913.

---

## #7 — Centraliser `_downsample_for_plot` *(45 min)*

### Étape 1 — Ajouter la fonction standalone dans `_chart_series.py`

> **État existant** : `_chart_series.py` a déjà `MAX_PLOT_POINTS = 200` (ligne 31) et `ChartData.downsample()` (ligne 103).
> Mais il n'y a pas de fonction **standalone** `downsample_for_plot(df)` pour les cas hors `ChartData`.
> `timeseries.py` a encore sa propre copie locale `_downsample_for_plot` (ligne 54) + `MAX_PLOT_POINTS` (ligne 51).
>
> **Action** : ajouter uniquement la fonction module-level dans `_chart_series.py` — ne pas toucher `ChartData.downsample()`.

```python
# À ajouter dans _chart_series.py, après MAX_PLOT_POINTS (déjà présent)

def downsample_for_plot(df: pl.DataFrame, max_points: int = MAX_PLOT_POINTS) -> pl.DataFrame:
    """Réduit le DataFrame pour le rendu graphique (conserve tendance).

    Garde le premier, le dernier, et un échantillonnage régulier entre les deux.
    """
    if len(df) <= max_points:
        return df
    step = len(df) // max_points
    indices = list(range(0, len(df), step))
    if indices[-1] != len(df) - 1:
        indices.append(len(df) - 1)
    return df[indices]
```

### Étape 2 — Mettre à jour l'appelant UI

Dans `src/ui/pages/timeseries.py` :
```python
# Supprimer la définition locale + MAX_PLOT_POINTS
# Ajouter l'import
from src.visualization._chart_series import downsample_for_plot, MAX_PLOT_POINTS
```

### Étape 3 — Appeler depuis les fonctions `plot_*` à risque

Ajouter `df = downsample_for_plot(df)` en première ligne dans :
- `plot_average_life(df, ...)` dans `timeseries_combat.py`
- `plot_spree_headshots_accuracy(df, ...)` dans `timeseries_combat.py`
- `plot_damage_dealt_taken(df, ...)` dans `timeseries_combat.py`
- `plot_assists_timeseries(df, ...)` dans `timeseries.py`

> Les fonctions multi-axes (`plot_timeseries`, `plot_per_minute_timeseries`)
> passent déjà par un downsample côté appelant (page `timeseries.py`) — ne pas doubler.

> **⚠️ Vérifier après cette étape** : `wc -l src/visualization/timeseries_combat.py` doit rester < 500.
> Si dépassement → extraire une ou deux fonctions helper avant de committer.

**Validation** : tests + charger une page timeline avec 500+ matchs et vérifier que Plotly est fluide.

---

## #8 — Finir le découpage de `maps_outcome.py` *(1h)*

### État actuel

`maps_outcome.py` = 363L. Fonctions encore présentes :

| Ligne | Nom | Extraction cible |
|-------|-----|-----------------|
| 37 | `_perf_color(v)` | `_maps_outcome_history.py` |
| 51 | `_empty_map_figure()` | reste dans `maps_outcome.py` (partagé) |
| 60 | `_prepare_timeline_df(...)` | `_maps_outcome_timeline.py` |
| 81 | `_add_timeline_traces(...)` | `_maps_outcome_timeline.py` |
| 119 | `plot_map_lollipop(...)` | reste dans `maps_outcome.py` |
| 203 | `plot_map_outcome_timeline(...)` | `_maps_outcome_timeline.py` |
| 290 | `plot_map_perf_vs_history(...)` | `_maps_outcome_history.py` |

### Étape 1 — Créer `_maps_outcome_timeline.py`

Déplacer : `_prepare_timeline_df`, `_add_timeline_traces`, `plot_map_outcome_timeline`.

### Étape 2 — Créer `_maps_outcome_history.py`

Déplacer : `_perf_color`, `plot_map_perf_vs_history`.

### Étape 3 — Réexporter depuis `maps_outcome.py`

```python
from src.visualization._maps_outcome_timeline import plot_map_outcome_timeline
from src.visualization._maps_outcome_history import plot_map_perf_vs_history

__all__ = [
    "plot_map_lollipop",
    "plot_map_outcome_timeline",
    "plot_map_perf_vs_history",
    "plot_map_winrate_bullet",  # déjà dans _maps_outcome_bullet.py
]
```

**Validation** : `python -m ruff check src/visualization/maps_outcome.py` + tests.

---

## #2 — Migrer timeseries vers `SingleSeriesChartData` *(plusieurs sessions)*

> **Dépendance** : faire #7 d'abord (downsample centralisé).
>
> **⚠️ Risque de régression élevé** : `timeseries_combat.py` est dense (multi-axes, bandes de confiance,
> heatmap cadence récemment ajoutée). Migrer **une fonction à la fois**, tester visuellement avant de passer
> à la suivante — ne pas batcher toute une phase sans validation intermédiaire.

### `SingleSeriesChartData` — rappel

```python
@dataclass
class SingleSeriesChartData:
    x: list[Any]
    y: list[float | None]
    y_smooth: list[float | None]
    height: int = HEIGHT_TIMESERIES

    @classmethod
    def from_series(cls, x, y, window=10, **kwargs) -> SingleSeriesChartData: ...
```

### Phase 2a — Fonctions simples (1 série Y) *(2h)*

Candidats : `plot_average_life`, `plot_accuracy_last_n`, `plot_assists_timeseries`.

Pattern de migration :
```python
# Avant
def plot_average_life(df: pl.DataFrame, lang: str = "fr") -> go.Figure:
    x = df["start_time"].to_list()
    y = df["avg_life"].to_list()
    # construction manuelle go.Bar/go.Scatter...

# Après
def plot_average_life(df: pl.DataFrame, lang: str = "fr") -> go.Figure:
    df = downsample_for_plot(df)
    data = SingleSeriesChartData.from_series(
        x=df["start_time"].to_list(),
        y=df["avg_life"].to_list(),
    )
    fig = go.Figure()
    fig.add_trace(go.Bar(x=data.x, y=data.y, ...))
    fig.add_trace(go.Scatter(x=data.x, y=data.y_smooth, ...))
    return apply_halo_plot_style(fig, height=data.height)
```

### Phase 2b — Fonctions multi-axes *(3h)*

Candidats : `plot_timeseries`, `plot_spree_headshots_accuracy`, `plot_damage_dealt_taken`,
`plot_shots_accuracy`, `plot_streak_chart`.

Ces fonctions ont plusieurs séries. Pattern :
- Créer un `SingleSeriesChartData` par axe/série principale
- Le rolling mean est géré uniformément via `from_series()`
- Le downsampling s'applique une fois au `df` en entrée

### Phase 2c — Fonctions de progression *(1h)*

Candidats : `plot_performance_timeseries`, `plot_rank_score`, `plot_lusr_timeseries`
dans `_timeseries_progression.py`.

**Validation globale** : `python -m pytest tests/ -q --ignore=tests/integration` + test visuel sur
les pages timeseries, career_lusr, win_loss.

---

## Checklist finale — Viz

- [ ] Changes cadence commitées sur `refactor/sessions-perf` avant de démarrer
- [ ] #4 backfill `spartan_carnage` exécuté
- [ ] #10 `KdCiData` + `EwmaData` créés, 0 `# noqa: PLR0913` restant
- [ ] #9 constantes hauteur importées (0 magic number)
- [ ] #1 `title=` supprimé de `theme.py` (param + DeprecationWarning + bloc if)
- [ ] #7 `downsample_for_plot` standalone dans `_chart_series.py`, appelé dans 4 fonctions `plot_*`
- [ ] #7 copie locale `_downsample_for_plot` + `MAX_PLOT_POINTS` supprimées de `timeseries.py`
- [ ] #8 `_maps_outcome_timeline.py` + `_maps_outcome_history.py` créés
- [ ] #2 phase 2a : 3 fonctions simples migrées vers `SingleSeriesChartData` (une par une, test visuel après chaque)
- [ ] #2 phase 2b : fonctions multi-axes migrées
- [ ] #2 phase 2c : fonctions progression migrées
- [ ] Tests passent (`pytest -q --ignore=tests/integration`)
- [ ] `timeseries_combat.py` reste sous 500L après #9 + #7
- [ ] 0 autre fichier `src/visualization/` dépasse 500L

---

---

# Plan refacto i18n assets — Traductions DB-first

> Créé le 2026-04-06  
> Statut global : **En attente**  
> Branche cible : `refactor/i18n-asset-labels`  
> **Aucun croisement avec le plan viz** — branches parallélisables.

---

## Contexte

En session précédente, le flux Discord a été corrigé : `LastMatchInfo` porte désormais
les IDs d'assets, `_discord_embed.py` résout via `asset_translations` selon `discord_lang`,
zéro colonne `_fr` hardcodée.

L'audit des call-sites UI restants a révélé 3 vrais problèmes et 1 duplication :

| # | Fichier | Nature |
|---|---------|--------|
| A | `explorer_enrich.py` | **Bug** — `playlist_fr` aliasé depuis EN même quand `playlist_name_fr` est présent |
| B | `_filters_apply.py` | **Sous-optimal** — `playlist_ui` appelle `translate_playlist_name()` quand `playlist_name_fr` est dans le DF |
| C | `filters_render.py` | **Sous-optimal** — options sidebar construites depuis EN, divergence possible avec `playlist_ui` |
| D | `helpers.py` + `mode_categories.py` | **DRY** — `normalize_mode_label` et `normalize_pair_name_to_mode_ui` dupliquent la même logique |

Les fichiers audités et déjà corrects (pas de travail à faire) :
`match_view.py`, `match_history.py`, `_session_compare_history.py`, `career_top_matches_render.py`.

---

## Ordre d'exécution recommandé

```
1. #A  explorer_enrich.py    (bug réel, 15 min)
2. #D  DRY normalize_mode    (avant #B+#C pour ne pas tripler la dette, 30 min)
3. #B  _filters_apply.py     (optimisation + cohérence, 20 min)
4. #C  filters_render.py     (cohérence sidebar, 15 min)
```

---

## #A — Corriger `explorer_enrich.py` *(15 min — bug)*

### Problème exact

`enrich_for_table()` ligne ~42 :
```python
if "playlist_fr" not in result.columns and "playlist_name" in result.columns:
    result = result.with_columns(pl.col("playlist_name").alias("playlist_fr"))
```
Quand `playlist_name_fr` est dans le DF (cas normal depuis `mv_player_matches` / `v_match_full`),
le nom EN est affiché directement. Même bug dans le bloc `enrich_for_career()` (~ligne 133).

Même problème pour `mode_ui` : si `pair_name_fr` est présent, il n'est pas prioritaire.

### Fichier : `src/ui/pages/explorer_enrich.py`

**Patch `playlist_fr`** (2 occurrences) :
```python
# Avant
if "playlist_fr" not in result.columns and "playlist_name" in result.columns:
    result = result.with_columns(pl.col("playlist_name").alias("playlist_fr"))

# Après
if "playlist_fr" not in result.columns and "playlist_name" in result.columns:
    if "playlist_name_fr" in result.columns:
        result = result.with_columns(
            pl.coalesce([
                pl.col("playlist_name_fr").cast(pl.Utf8),
                pl.col("playlist_name").cast(pl.Utf8),
            ]).alias("playlist_fr")
        )
    else:
        result = result.with_columns(pl.col("playlist_name").alias("playlist_fr"))
```

**Patch `mode_ui`** (2 occurrences) :
```python
# Avant
if "mode_ui" not in result.columns and "pair_name" in result.columns:
    _ee_mode_map = build_mapping(
        result["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
    )
    result = result.with_columns(
        pl.col("pair_name").cast(pl.Utf8)
        .replace_strict(_ee_mode_map, default=pl.col("pair_name").cast(pl.Utf8), ...)
        .alias("mode_ui")
    )

# Après — priorité pair_name_fr
if "mode_ui" not in result.columns and "pair_name" in result.columns:
    if "pair_name_fr" in result.columns:
        result = result.with_columns(
            pl.coalesce([
                pl.col("pair_name_fr").cast(pl.Utf8),
                pl.col("pair_name").cast(pl.Utf8),
            ]).alias("mode_ui")
        )
    else:
        _ee_mode_map = build_mapping(
            result["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
        )
        result = result.with_columns(
            pl.col("pair_name").cast(pl.Utf8)
            .replace_strict(_ee_mode_map, default=pl.col("pair_name").cast(pl.Utf8),
                            return_dtype=pl.Utf8)
            .alias("mode_ui")
        )
```

**Validation** : test unitaire — DF avec `playlist_name_fr="Partie rapide"` et
`playlist_name="Quick Play"` → `playlist_fr` doit valoir `"Partie rapide"`.

---

## #D — Consolider `normalize_mode_label` / `normalize_pair_name_to_mode_ui` *(30 min — DRY)*

### Problème exact

Deux fonctions quasi-identiques dans deux modules distincts :

| Point | `helpers.py::normalize_mode_label` | `mode_categories.py::normalize_pair_name_to_mode_ui` |
|-------|------------------------------------|------------------------------------------------------|
| Nettoyage suffixe | `clean_asset_label(pair_name)` | `_LABEL_SUFFIX_RE.match(raw)` |
| Traduction | `translate_pair_name(base, lang, normalize=normalize)` | `translate_pair_name(raw, lang)` — normalize hardcodé True |
| Strip "on Map" | ✅ | ✅ |
| Strip Forge/Ranked | ✅ | ✅ |
| Param `normalize` | ✅ exposé | ❌ absent |
| Import | module-level | local import (évite circular `analysis` → `app`) |

Call-sites :
- `normalize_mode_label` → `match_view.py`, `_filters_apply.py`, `filters_render.py`
- `normalize_pair_name_to_mode_ui` → `mode_categories.py::categorize_match` + tests

### Règle cible

`mode_categories.py` est dans `src/analysis/` → ne peut pas importer `src/app/helpers.py`.  
Direction correcte : `helpers.py::normalize_mode_label` délègue à `normalize_pair_name_to_mode_ui`.

### Fichier : `src/analysis/mode_categories.py`

Ajouter le param `normalize` à `normalize_pair_name_to_mode_ui` :
```python
# Avant
def normalize_pair_name_to_mode_ui(pair_name: str | None, lang: str = "fr") -> str | None:
    ...
    translated = translate_pair_name(raw, lang=lang)

# Après
def normalize_pair_name_to_mode_ui(
    pair_name: str | None, lang: str = "fr", *, normalize: bool = True
) -> str | None:
    ...
    translated = translate_pair_name(raw, lang=lang, normalize=normalize)
```

### Fichier : `src/app/helpers.py`

Faire déléguer `normalize_mode_label` :
```python
def normalize_mode_label(
    pair_name: str | None,
    *,
    lang: str = "fr",
    normalize: bool = True,
) -> str | None:
    """Normalise le label d'un mode de jeu (délègue à mode_categories)."""
    from src.analysis.mode_categories import normalize_pair_name_to_mode_ui
    return normalize_pair_name_to_mode_ui(pair_name, lang=lang, normalize=normalize)
```

Supprimer le corps dupliqué (les `re.sub`, `translate_pair_name`, `clean_asset_label`).  
Supprimer les imports module-level de `translate_pair_name` dans `helpers.py` s'ils ne sont
plus utilisés ailleurs dans le fichier.

**Validation** : `tests/test_helpers.py` et `tests/test_mode_categories.py` — même résultat
sur les mêmes inputs pour les deux fonctions.

---

## #B — Aligner `playlist_ui` dans `_filters_apply.py` *(20 min — optimisation)*

### Problème exact

Dans `_add_derived_columns()` :
- `playlist_fr` : COALESCE `playlist_name_fr + playlist_name` ✓
- `playlist_ui` : **toujours** `translate_playlist_name(clean_asset_label_fn(playlist_name))`,
  même quand `playlist_name_fr` est déjà dans le DF → appel DB inutile à chaque rendu.

`playlist_ui` alimente : filtre sidebar, `_apply_experience_filter`, checkboxes playlists.

### Fichier : `src/app/_filters_apply.py`

```python
# Avant
if "playlist_ui" not in dff.columns:
    _pui_map = build_mapping(
        dff["playlist_name"],
        lambda x: translate_playlist_name(clean_asset_label_fn(x), lang=get_lang()),
    )
    derived_exprs.append(
        pl.col("playlist_name").cast(pl.Utf8)
        .replace_strict(_pui_map, default=None, return_dtype=pl.Utf8)
        .alias("playlist_ui")
    )

# Après
if "playlist_ui" not in dff.columns:
    if "playlist_name_fr" in dff.columns:
        derived_exprs.append(
            pl.coalesce([
                pl.col("playlist_name_fr").cast(pl.Utf8),
                pl.col("playlist_name").cast(pl.Utf8),
            ]).alias("playlist_ui")
        )
    else:
        _pui_map = build_mapping(
            dff["playlist_name"],
            lambda x: translate_playlist_name(clean_asset_label_fn(x), lang=get_lang()),
        )
        derived_exprs.append(
            pl.col("playlist_name").cast(pl.Utf8)
            .replace_strict(_pui_map, default=None, return_dtype=pl.Utf8)
            .alias("playlist_ui")
        )
```

**Point de vigilance** : `_apply_experience_filter` classe les playlists en PvP/PvE/Custom.
Vérifier que la classification fonctionne avec des libellés FR (ex. `"Partie rapide"`).
Si `_apply_experience_filter` compare contre des strings EN hardcodées → ouvrir un ticket séparé.

**Validation** : `tests/test_filters_cascade.py` — experience_types passent avec les deux chemins
(DF avec et sans `playlist_name_fr`).

---

## #C — Cohérence sidebar dans `filters_render.py` *(15 min — optimisation)*

### Problème exact

`_compute_all_filter_options()` calcule les options des checkboxes sidebar :
```python
def _playlist_label(x: str) -> str:
    return str(translate_playlist_name(clean_asset_label_fn(x), lang=_lang))

_collect_unique_labels(base, "playlist_name", _playlist_label)
```
Collecte depuis la colonne EN et traduit — alors que `_add_derived_columns` peut avoir produit
`playlist_ui` à partir de `playlist_name_fr` directement.
Si un libellé FR n'est pas roundtrippable via `translate_playlist_name()`, les checkboxes
de la sidebar ne matchent plus `playlist_ui` → filtre silencieusement cassé.

Même logique que `map_name_fr` déjà traité correctement quelques lignes plus bas.

### Fichier : `src/app/filters_render.py`

```python
# Avant
_collect_unique_labels(base, "playlist_name", _playlist_label),

# Après — même pattern que map_name_fr
_collect_unique_labels(
    base,
    "playlist_name_fr" if "playlist_name_fr" in base.columns else "playlist_name",
    (lambda x: str(clean_asset_label_fn(x)))
        if "playlist_name_fr" in base.columns
        else _playlist_label,
),
```

**Point de vigilance** : si `playlist_name_fr` contient des nulls (matchs anciens sans traduction),
`_collect_unique_labels` les ignore déjà via `drop_nulls()`. Pas de régression attendue.

**Validation** : test manuel — vérifier que les checkboxes sidebar affichent les mêmes libellés
que la colonne `playlist_ui` du tableau.

---

## Checklist finale — i18n

- [ ] #A `explorer_enrich.py` : `playlist_fr` et `mode_ui` priorisent `*_fr` columns
- [ ] #D `normalize_mode_label` délègue à `normalize_pair_name_to_mode_ui` (0 duplication)
- [ ] #D `normalize_pair_name_to_mode_ui` accepte `normalize: bool = True`
- [ ] #B `playlist_ui` dans `_filters_apply.py` priorise `playlist_name_fr`
- [ ] #C options sidebar dans `filters_render.py` collectées depuis `playlist_name_fr` si disponible
- [ ] Aucun `translate_playlist_name()` ni `translate_pair_name()` appelé pour des colonnes déjà pré-calculées en FR
- [ ] Tests passent (`pytest -q --ignore=tests/integration`)
