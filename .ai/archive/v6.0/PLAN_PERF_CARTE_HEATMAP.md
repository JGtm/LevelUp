# PLAN — Performance par carte vs historique + Heatmap joueur × carte

> Créé le 2026-03-18. Branche : `refactor/id-resolution-cleanup`

---

## Alternatives visuelles pour le graphe "par carte" (à choisir)

> Contexte : le bar empilé Win/Loss actuel est monochrome sur des sessions courtes (n=1 par carte → 100% vert ou 100% rouge, aucune richesse visuelle).
> Les 3 options ci-dessous restent centrées sur la dimension **outcome (V/D)**, pas sur la performance/skill.

### Option A — Lollipop chart (dot plot)

**Principe** : une ligne horizontale par carte + un cercle coloré (vert=victoire, rouge=défaite) à son extrémité.

```python
def plot_map_lollipop(
    df_breakdown: DataFrameLike,   # colonnes : map_name, win_rate, loss_rate, matches
    lang: str = "fr",
) -> go.Figure:
```

- Trace `go.Scatter` mode `"lines+markers"` : ligne de (0, map_name) à (win_rate, map_name)
- Marqueur : taille proportionnelle à `matches` (min 12px, max 24px), couleur = vert si `win_rate >= 0.5` sinon rouge
- Annotation texte sur le marqueur : `f"{win_rate:.0%} ({matches})"` si matches > 1, sinon `"V"` / `"D"`
- Trier les cartes par `win_rate` desc
- Ligne de référence verticale à x=0.5 (couleur gris, dash="dot")
- Fonctionne en mode `absolute_counts` : x = wins (entier), ref line à `matches/2`

**Avantages** : propre, élégant, outcome-focused, lisible avec n=1. Facilement extensible (2ème marqueur = partenaire).

**Fichier** : `src/visualization/maps.py` (~40L supplémentaires → ~250L total)

---

### Option B — Timeline dots par carte

**Principe** : chaque carte = une ligne horizontale de cercles triés chronologiquement (1 cercle = 1 match), colorés par outcome. Les matchs de la session courante sont mis en évidence (taille × 1.5, contour blanc).

```python
def plot_map_outcome_timeline(
    df_matches: DataFrameLike,     # colonnes : map_name, start_time, outcome, match_id
    session_match_ids: list[str],  # match_ids de la session courante à mettre en évidence
    lang: str = "fr",
) -> go.Figure:
```

- Trier les cartes par fréquence desc (max 15 cartes)
- Pour chaque carte : une trace `go.Scatter` mode `"markers"`, x = index temporel (0, 1, 2…), y = map_name
- Couleur par outcome : vert=WIN, rouge=LOSS, violet=TIE/DNF
- Points session courante : `marker_size=14`, `marker_line_width=2`, `marker_line_color="white"`
- Points historiques : `marker_size=8`, `marker_opacity=0.5`
- Tooltip : date + outcome + playlist
- Axe X masqué (c'est un ordre temporel, pas une valeur)

**Avantages** : le plus riche visuellement pour les sessions courtes — contextualise chaque résultat dans l'historique. Répond à "est-ce qu'on gagne souvent sur cette carte ?".

**Données** : nécessite `df_matches` brut (matchs individuels) plutôt que `df_breakdown` agrégé. Déjà disponible dans `sub_all` / `dff`.

**Fichier** : `src/visualization/maps.py` (~55L supplémentaires → ~265L total)

---

### Option C — Bullet chart win rate session vs historique

**Principe** : pour chaque carte, une barre grise fine = win rate historique, un marqueur coloré = résultat de la session (0% ou 100% si n=1). Reste 100% outcome — aucune métrique de perf/skill.

```python
def plot_map_winrate_bullet(
    bd_current: DataFrameLike,   # breakdown session : map_name, win_rate, matches
    bd_history: DataFrameLike,   # breakdown historique : map_name, win_rate, matches
    lang: str = "fr",
) -> go.Figure | None:
```

- Jointure `bd_current` ← `bd_history` sur `map_name`
- Si aucune carte commune → retourner `None`
- Barre horizontale grise translucide = `win_rate` historique (label : "Historique X%")
- Marqueur vertical (ligne) sur la barre = `win_rate` session (vert si > hist, rouge si < hist, neutre si égal)
- Ligne de référence à x=0.5 (50%)
- Trier par `bd_history.win_rate` desc
- Tooltip : "Session : V% | Historique : V% (N matchs)"

**Avantages** : répond à "est-ce qu'on gagne habituellement sur cette carte ?" sans sortir du domaine outcome. Différent des features 1 & 2 du plan principal (celles-ci sont sur `performance_avg`, pas sur `win_rate`).

**Données** : mêmes que `plot_map_ratio_with_winloss` + un `bd_history` (déjà calculable via `compute_map_breakdown` sur le scope non filtré).

**Fichier** : `src/visualization/maps.py` (~45L supplémentaires → ~255L total)

---

## Objectifs

### Feature 1 — Heatmap joueur × carte (Page Teammates)

Heatmap 2D Plotly (`go.Heatmap`) sur la Page Teammates :
- **Axe Y** : joueurs de l'escouade (moi + coéquipier(s))
- **Axe X** : cartes jouées ensemble (triées par fréquence desc)
- **Couleur** : score de performance moyen (`performance_avg`) de chaque joueur sur chaque carte
- **Tooltip** : "Joueur X — Carte Y : perf=Z.Z (N matchs)"
- Affiché **toujours** dans la section "Par carte" de la vue multi-coéquipiers (≥ 2 joueurs)
- Visible aussi en vue 1 coéquipier (axe Y = 2 lignes : moi + coéquipier)

### Feature 2 — Performance par carte vs historique (Win/Loss + Teammates)

**Page Win/Loss** :
- Pour chaque carte jouée dans les filtres actuels, afficher 2 barres horizontales groupées :
  - Barre 1 : performance moyenne **dans la sélection courante** (`dff`)
  - Barre 2 : performance moyenne **historique totale** sur cette carte (`base`)
- Delta visible : montre les cartes surperformées vs sous-performées par rapport à la normale

**Page Teammates (vue 1 coéquipier + vue escouade)** :
- Même principe, mais la métrique = performance moyenne de l'escouade
- Barre 1 : perf escouade sur les matchs de la sélection avec ces coéquipiers
- Barre 2 : perf escouade sur **tous** leurs matchs communs (non filtré)

### Nettoyage Win/Loss

- Supprimer le `st.radio` scope (SpartanA/SpartanB/Moi toutes parties) → toujours utiliser `dff`
- Supprimer le `st.selectbox` de métrique → afficher **toutes** les métriques en séquentiel
- Supprimer `get_friend_scope_df` dans `win_loss_service.py` (dead code)
- Garder le slider `min_matches`

---

## Fichiers impactés

| Fichier | Action |
|---------|--------|
| `src/visualization/maps.py` (211L) | Ajouter `plot_map_perf_vs_history` (~50L → total ~260L) |
| `src/visualization/friends_impact_heatmap.py` (356L) | Ajouter `plot_squad_map_heatmap` (~60L → total ~416L) |
| `src/data/services/win_loss_service.py` (246L) | Supprimer `get_friend_scope_df`, modifier `compute_map_breakdown` |
| `src/ui/pages/win_loss.py` (344L) | Refactor `_render_ratio_by_map_section` |
| `src/ui/pages/teammates_views.py` (448L) | Ajouter heatmap + perf vs historique dans `_render_map_breakdown` et vue single |
| `src/ui/i18n/pages/wl.py` | Supprimer clés `wl_scope_*`, ajouter `wl_perf_vs_history_caption` |
| `src/ui/i18n/pages/teammates.py` | Ajouter `tm_map_heatmap`, `tm_perf_vs_history` |

---

## Plan détaillé par étape

### Étape 1 — `plot_map_perf_vs_history` dans `maps.py`

```python
def plot_map_perf_vs_history(
    bd_current: DataFrameLike,   # breakdown filtré
    bd_history: DataFrameLike,   # breakdown historique total
    lang: str = "fr",
) -> go.Figure | None:
```

- Jointure `bd_current` ← `bd_history` sur `map_name` (inner join = cartes présentes dans les deux)
- Si aucune carte commune → retourner `None`
- Barres horizontales groupées (`barmode="group"`) :
  - Trace "Sélection actuelle" : `performance_avg` from `bd_current`, couleur cyan
  - Trace "Historique" : `performance_avg` from `bd_history`, couleur gris (#888)
- Trier par `bd_current.performance_avg` desc
- Taille max : 20 cartes
- Ajouter ligne de référence verticale à x=0

### Étape 2 — `plot_squad_map_heatmap` dans `friends_impact_heatmap.py`

```python
def plot_squad_map_heatmap(
    series: list[tuple[str, pl.DataFrame]],  # (nom_joueur, df_matchs)
    lang: str = "fr",
) -> go.Figure | None:
```

- Pour chaque (joueur, df) : calculer `compute_map_breakdown(df)` → `performance_avg` par carte
- Construire matrice `players × cartes` (Polars pivot)
- Cartes absentes pour un joueur → NaN (Plotly les affiche en blanc/gris)
- Trier les cartes par fréquence totale desc (max 15 cartes)
- `go.Heatmap` avec colorscale personnalisée :
  - Rouge `SCORE_THRESHOLDS["bad"]` → Cyan `SCORE_THRESHOLDS["excellent"]`
- Tooltip : "Joueur — Carte : perf=X.X (N matchs)"
- Légende de la colorscale en bas

### Étape 3 — `win_loss_service.py`

**Modifier `compute_map_breakdown`** :
```python
@staticmethod
def compute_map_breakdown(
    base_scope: pl.DataFrame,
    min_matches: int,
    df_history: pl.DataFrame | None = None,  # nouveau paramètre
) -> MapBreakdownResult:
    breakdown = compute_map_breakdown(base_scope, df_history=df_history)
    ...
```

**Supprimer** :
- `get_friend_scope_df` (entièrement)

### Étape 4 — `win_loss.py` : refactor `_render_ratio_by_map_section`

**Signature** : retirer le paramètre `base` de la signature (passer directement dans la fonction)

**Supprimer** :
- `st.radio` scope (+ ses options SpartanA/SpartanB)
- `st.selectbox` métrique
- Import local de `WinLossService.get_friend_scope_df`

**Nouvel affichage séquentiel** (tout sur `dff` vs `base`) :
```
1. st.subheader("V/D par carte")
   → plot_map_ratio_with_winloss(breakdown_current, absolute_counts=True)

2. st.subheader("Performance vs historique")
   → plot_map_perf_vs_history(breakdown_current, breakdown_history)

3. (si pas is_session_scope) st.subheader("Ratio F/M par carte")
   → plot_map_comparison(breakdown_current, "ratio_global", ...)

4. (si pas is_session_scope) st.subheader("Précision par carte")
   → plot_map_comparison(breakdown_current, "accuracy_avg", ...)
```

**Calcul** :
- `breakdown_current` = `compute_map_breakdown(dff, min_matches, df_history=base)`
- `breakdown_history` = `compute_map_breakdown(base, min_matches=1, df_history=base)`

**Clés i18n à supprimer** : `wl_scope_label`, `wl_scope_me_filtered`, `wl_scope_me_all`
**Clés i18n à ajouter** : `wl_perf_vs_history_caption`

### Étape 5 — `teammates_views.py` : vue escouade + vue 1 coéquipier

#### Vue multi (`_render_map_history_section` → `_render_map_breakdown`)

Passer `full_squad_df` (non filtré) en plus de `sub_all` (filtré) :

```python
# Dans _render_map_history_section :
full_squad_df = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(all_match_ids)))
_render_map_breakdown(sub_all, full_squad_df, breakdown_current, min_matches, ctx)
```

Dans `_render_map_breakdown` :
```
1. plot_map_ratio_with_winloss(breakdown_current)          ← déjà présent
2. plot_map_perf_vs_history(bd_current, bd_history)        ← NOUVEAU
3. plot_squad_map_heatmap(series)                          ← NOUVEAU (si ≥ 2 joueurs)
```

Pour `bd_history` :
- `compute_map_breakdown(full_squad_df, df_history=full_squad_df)`

#### Vue 1 coéquipier (`render_single_teammate_view`)

Après `_render_shared_stats_metrics`, ajouter :
```python
shared_full = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(shared_ids))
_render_single_map_section(sub, shared_full, series, lang)
```

Nouvelle fonction `_render_single_map_section` :
```
1. st.subheader("Par carte")
2. plot_map_ratio_with_winloss(breakdown_current)
3. plot_map_perf_vs_history(bd_current, bd_history)
4. plot_squad_map_heatmap(series)
```

**Clés i18n à ajouter** : `tm_map_heatmap`, `tm_perf_vs_history`

---

## Contraintes de taille

| Fichier | Avant | Estimé après | Statut |
|---------|------:|-------------:|--------|
| `maps.py` | 211L | ~265L | ✅ |
| `friends_impact_heatmap.py` | 356L | ~420L | ✅ |
| `win_loss_service.py` | 246L | ~200L | ✅ (suppression) |
| `win_loss.py` | 344L | ~310L | ✅ (suppression) |
| `teammates_views.py` | 448L | ~490L | ✅ (limite 500L) |

---

## Séquence de commits

```
feat(viz): plot_map_perf_vs_history + plot_squad_map_heatmap
feat(ui/win-loss): affichage séquentiel cartes + suppression scope radio/selectbox
feat(ui/teammates): heatmap joueur×carte + perf vs historique vues single et multi
```

---

## Tests à valider après implémentation

- `_render_ratio_by_map_section` sans erreur quand `dff` est vide
- `plot_map_perf_vs_history` retourne `None` si aucune carte commune
- `plot_squad_map_heatmap` retourne `None` si `series` vide ou toutes cartes vides
- Pas de régression sur les tests existants `test_win_loss_page.py`
