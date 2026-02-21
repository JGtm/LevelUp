# Plan de refactoring des filtres — V2 (basé sur main)

> **Date** : 2026-02-20
> **Base** : branche `feature/v5.2`
> **Statut** : ✅ COMPLÉTÉ — v5.2 (2026-02-20)

---

## Contexte

Ce plan cible la branche `main` telle qu'elle est aujourd'hui. La branche expérimentale
`copilot/refactor-filter-selection-functionality` n'a jamais été mergée et contenait des
régressions ; elle est ignorée. Ce document est le plan unique de référence.

**Philosophie** : l'utilisateur choisit ses filtres une fois, l'app doit les restaurer
fidèlement — y compris quand de nouvelles playlists/modes/cartes apparaissent (mise à jour
du jeu, intégration PVE, nouvelles saisons PVP).

**Fichiers concernés** :
- `src/ui/filter_state.py` — Persistance (dataclass, save, load, apply)
- `src/app/filters_render.py` — Rendu sidebar (cascade, options, sauvegarde)
- `src/ui/components/checkbox_filter.py` — Composants checkbox
- `tests/test_filter_state.py` — Tests

---

## État actuel sur main — ce qui fonctionne ✅

- **Persistance JSON par joueur** : save/load/apply/clear OK, clés scopées via `_get_player_key`
- **Sauvegarde automatique** à chaque rendu (`render_filters_sidebar` L203-210)
- **Application au premier chargement** via `filters_loaded_key` (L127-139)
- **Vectorisation** : `build_mapping` + `replace_strict` (pas de `map_elements`)
- **`friends_tuple`** : retourné correctement par `_render_session_filter` (signature 4-tuple)
- **Bouton trio** : utilise `on_click` callback (pas de `st.rerun()` explicite)
- **Cascade scope1/scope2** : active (les options Modes dépendent des Playlists cochées,
  les options Cartes dépendent des Modes cochés)
- **Default à "dernière session"** si aucun filtre sauvegardé

---

## Problèmes à résoudre

### Problème 1 — Nouvelles options apparaissent décochées (bug principal)

**Symptôme** : l'utilisateur a "tout coché sauf Firefight". Après un sync qui ajoute une
nouvelle playlist "Slayer", au prochain chargement Slayer apparaît **décoché** car elle n'est
pas dans le JSON sauvegardé.

**Cause racine** : la persistance stocke des **valeurs brutes** (liste des sélectionnés)
au lieu de l'**intention** ("tout sauf X"). Le code actuel :

```python
# save_filter_preferences (filter_state.py L190-192)
playlists = st.session_state.get("filter_playlists")
if isinstance(playlists, (set, list)):
    preferences.playlists_selected = sorted(playlists)  # ← inclusions brutes
```

```python
# apply_filter_preferences (filter_state.py L293-294)
if preferences.playlists_selected is not None:
    st.session_state["filter_playlists"] = set(preferences.playlists_selected)  # ← restaure tel quel
```

Il n'y a aucune notion de mode include/exclude. Un utilisateur qui coche 11/12 playlists
sauvegarde 11 valeurs ; une 13e playlist ajoutée ne sera pas dans les 11 → décochée.

**Solution** : architecture **intent-based** — stocker le mode (include/exclude) et les
valeurs correspondantes. En mode exclude, on stocke ce qui est *décoché*, et au chargement
on calcule `all_options - exclusions` pour que les nouvelles options soient auto-cochées.

### Problème 2 — Cascade scope1/scope2 et désorientation

**Symptôme** : chaque clic sur une checkbox Playlist recalcule les options Modes disponibles
(et chaque clic Mode recalcule les Cartes). Résultat :
- Des modes/cartes **disparaissent** de la liste quand on décoche une playlist
- L'utilisateur perd ses sélections modes/cartes si les options deviennent indisponibles
- Le nettoyage dans `render_checkbox_filter` (L166-169) fait `current & set(options)` :
  si `options` rétrécit à cause de la cascade, des valeurs cochées sont silencieusement supprimées

**Code actuel** (`_render_cascade_filters`, L571-603) :
```python
# filters_render.py L571-603
scope1 = dropdown_base
if playlists_selected and len(playlists_selected) < len(playlist_values):
    scope1 = scope1.filter(pl.col("playlist_ui").fill_null("").is_in(playlists_selected))

mode_values = sorted({str(x).strip() for x in scope1["mode_ui"].drop_nulls().to_list() ...})
# mode_values calculé depuis scope1 → restreint par la sélection playlist ↑

scope2 = scope1
if modes_selected and len(modes_selected) < len(mode_values):
    scope2 = scope2.filter(pl.col("mode_ui").fill_null("").is_in(modes_selected))

map_values = sorted({str(x).strip() for x in scope2["map_ui"].drop_nulls().to_list() ...})
# map_values calculé depuis scope2 → restreint par mode + playlist ↑
```

**Solution** : supprimer la cascade — calculer Modes et Cartes depuis `dropdown_base` complet
(filtré uniquement par la fenêtre temporelle, pas par les sélections checkbox). L'avantage :
stabilité des options, pas de perte silencieuse de sélections.

### Problème 3 — Nouvelles options mid-session (transition Période ↔ Sessions)

**Symptôme** : l'utilisateur est en mode Sessions (3 matchs, 5 playlists visibles),
switch en Période (12 playlists disponibles). Les 7 nouvelles playlists apparaissent
décochées car `session_state["filter_playlists"]` ne contient que les 5 de la session.

**Cause** : le nettoyage à la L166-169 de `render_checkbox_filter` :
```python
# checkbox_filter.py L166-169
current_selection = st.session_state[session_key]
current_selection = current_selection & set(options)  # supprime les obsolètes
st.session_state[session_key] = current_selection
```
Ce nettoyage supprime les valeurs absentes des nouvelles `options`, mais **n'ajoute pas**
les nouvelles options. C'est correct pour le nettoyage, mais après Phase 2 (cascade supprimée),
ce sera la seule cause restante de "nouvelles playlists décochées" mid-session.

> **Note** : ce problème ne disparaît pas avec Phase 1 (la restore au chargement est correcte)
> ni avec Phase 2 (la cascade ne cause plus de disparitions, mais ne cause pas d'apparitions).
> Phase 3 est donc nécessaire pour le comportement mid-session.

**Solution** : dans `_render_cascade_filters`, après avoir calculé `playlist_values` et
**avant** d'appeler `render_checkbox_filter`, réconcilier les nouvelles options en se basant
sur les exclusions stockées — et non sur `set(options) - current_selection` (qui inclurait
des options délibérément décochées par l'utilisateur).

---

## Plan d'implémentation

### Phase 1 — Persistance intent-based (priorité haute)

**Objectif** : sauvegarder l'intention, pas les valeurs brutes

#### 1.1 Enrichir `FilterPreferences` avec les champs mode

```python
@dataclass
class FilterPreferences:
    filter_mode: str | None = None
    start_date: str | None = None
    end_date: str | None = None
    gap_minutes: int | None = None
    picked_session_label: str | None = None

    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None

    # AJOUT — mode d'interprétation des listes ci-dessus
    # "exclude" : playlists_selected contient ce qui est DÉCOCHÉ
    # "include" : playlists_selected contient ce qui est COCHÉ
    # None      : legacy (traiter comme "include" pour backward compat)
    playlists_mode: str | None = None
    modes_mode: str | None = None
    maps_mode: str | None = None
```

**Backward compat** : `from_dict` ignore les clés inconnues, et `*_mode = None` est
interprété comme `"include"` dans `apply_filter_preferences` → les JSON existants
sans `*_mode` continuent de fonctionner sans modification.

| Fichier | Modification |
|---------|-------------|
| `filter_state.py` | Ajouter 3 champs `*_mode` dans `FilterPreferences` |

#### 1.2 Créer `_detect_filter_mode()`

```python
def _detect_filter_mode(
    selected: set[str] | list[str],
    all_options: set[str] | list[str],
    current_mode: str = "include",
) -> str:
    """Détecte si l'utilisateur est en mode inclusion ou exclusion.

    Heuristique :
    - >70% cochés → exclude (on stocke les quelques décochés)
    - <30% cochés → include (on stocke les quelques cochés)
    - entre 30-70% → garde le mode actuel (hystérésis)

    Args:
        selected: Valeurs actuellement cochées.
        all_options: Toutes les valeurs disponibles.
        current_mode: Mode actuel (tie-break zone grise).

    Returns:
        "exclude" ou "include"
    """
    if not all_options:
        return "include"
    ratio = len(set(selected)) / len(set(all_options))
    if ratio > 0.7:
        return "exclude"
    elif ratio < 0.3:
        return "include"
    else:
        return current_mode
```

Le paramètre `current_mode` élimine le problème d'hystérésis : dans la zone grise (30-70%),
le mode ne bascule pas silencieusement.

| Fichier | Modification |
|---------|-------------|
| `filter_state.py` | Ajouter `_detect_filter_mode()` |

#### 1.3 Modifier `save_filter_preferences()` pour stocker le mode

Le save doit recevoir les `all_*` options pour pouvoir détecter le mode et stocker
correctement (exclusions en mode exclude, inclusions en mode include).

```python
def save_filter_preferences(
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
    *,
    all_playlists: list[str] | None = None,   # AJOUT
    all_modes: list[str] | None = None,        # AJOUT
    all_maps: list[str] | None = None,         # AJOUT
) -> None:
```

Logique de sauvegarde pour chaque filtre (playlists/modes/maps) :

```python
playlists = st.session_state.get("filter_playlists")
if isinstance(playlists, (set, list)) and all_playlists:
    current_mode = st.session_state.get("_playlists_filter_mode", "include")
    mode = _detect_filter_mode(playlists, all_playlists, current_mode)
    st.session_state["_playlists_filter_mode"] = mode  # persister pour hystérésis
    if mode == "exclude":
        preferences.playlists_selected = sorted(set(all_playlists) - set(playlists))
    else:
        preferences.playlists_selected = sorted(playlists)
    preferences.playlists_mode = mode
elif isinstance(playlists, (set, list)):
    # Fallback sans all_playlists (ne devrait pas arriver)
    preferences.playlists_selected = sorted(playlists)
    preferences.playlists_mode = "include"
```

| Fichier | Modification |
|---------|-------------|
| `filter_state.py` | Modifier signature + corps de `save_filter_preferences` |

#### 1.4 Modifier `apply_filter_preferences()` pour interpréter le mode

```python
def apply_filter_preferences(
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
    *,
    all_playlists: list[str] | None = None,   # AJOUT
    all_modes: list[str] | None = None,        # AJOUT
    all_maps: list[str] | None = None,         # AJOUT
) -> None:
```

Pour chaque filtre :
```python
if preferences.playlists_selected is not None:
    mode = preferences.playlists_mode or "include"  # backward compat
    exclusions = set(preferences.playlists_selected) if mode == "exclude" else set()

    if mode == "exclude" and all_playlists:
        # all - exclusions = ce qui doit être coché
        st.session_state["filter_playlists"] = set(all_playlists) - exclusions
    else:
        # mode include ou pas d'all_playlists disponible
        st.session_state["filter_playlists"] = set(preferences.playlists_selected)

    # Persister mode et exclusions pour la réconciliation mid-session (Phase 3)
    st.session_state["_playlists_filter_mode"] = mode
    st.session_state["_playlists_exclusions"] = exclusions
```

| Fichier | Modification |
|---------|-------------|
| `filter_state.py` | Modifier signature + corps de `apply_filter_preferences` |

#### 1.4b Mettre à jour `FILTER_DATA_KEYS`

Les nouvelles clés persistées dans `session_state` doivent être ajoutées à `FILTER_DATA_KEYS`
(L26-42 de `filter_state.py`) pour être nettoyées lors du changement de joueur :

```python
FILTER_DATA_KEYS: list[str] = [
    # ... clés existantes ...
    "_playlists_filter_mode",   # AJOUT
    "_modes_filter_mode",       # AJOUT
    "_maps_filter_mode",        # AJOUT
    "_playlists_exclusions",    # AJOUT
    "_modes_exclusions",        # AJOUT
    "_maps_exclusions",         # AJOUT
]
```

| Fichier | Modification |
|---------|-------------|
| `filter_state.py` | Ajouter 6 entrées dans `FILTER_DATA_KEYS` |

#### 1.5 Câbler dans `render_filters_sidebar()`

Le pré-calcul des options disponibles doit être fait **avant** le bloc `filters_loaded_key`
pour que `apply_filter_preferences` ait accès à `all_*` :

```python
# render_filters_sidebar — AVANT le bloc filters_loaded_key

# Pré-calcul des options disponibles (base large = toutes les données du joueur)
_pre_playlists = sorted({
    str(translate_playlist_name(clean_asset_label_fn(x))).strip()
    for x in base_for_filters["playlist_name"].drop_nulls().to_list()
    if str(x).strip()
})
_pre_modes = sorted({
    str(normalize_mode_label_fn(x)).strip()
    for x in base_for_filters["pair_name"].drop_nulls().to_list()
    if str(x).strip()
})
_pre_maps = sorted({
    str(normalize_map_label_fn(x)).strip()
    for x in base_for_filters["map_name"].drop_nulls().to_list()
    if str(x).strip()
})

if filters_loaded_key not in st.session_state:
    try:
        prefs = load_filter_preferences(xuid, db_path)
        if prefs is not None:
            apply_filter_preferences(
                xuid, db_path, preferences=prefs,
                all_playlists=_pre_playlists,
                all_modes=_pre_modes,
                all_maps=_pre_maps,
            )
        else:
            _apply_default_last_session(db_path, xuid, db_key, aliases_key)
        st.session_state[filters_loaded_key] = True
    except Exception:
        st.session_state[filters_loaded_key] = True
```

Et modifier l'appel `save_filter_preferences` (L203-210) pour passer les `all_*` :

```python
save_filter_preferences(
    xuid, db_path,
    all_playlists=playlist_values,  # calculé dans _render_cascade_filters
    all_modes=mode_values,
    all_maps=map_values,
)
```

**Point d'attention** : `playlist_values`, `mode_values`, `map_values` sont calculés dans
`_render_cascade_filters` et ne sont pas directement accessibles dans `render_filters_sidebar`.
Solution : faire remonter ces valeurs dans le retour de `_render_cascade_filters`, ou
calculer les options au niveau de `render_filters_sidebar` (pré-calcul ci-dessus
réutilisé pour le save aussi).

| Fichier | Modification |
|---------|-------------|
| `filters_render.py` | Pré-calcul + modification appels apply/save |

### Phase 2 — Supprimer la cascade scope1/scope2 (priorité haute)

**Objectif** : stabiliser les options, éviter les pertes silencieuses

#### 2.1 Modifier `_render_cascade_filters`

Remplacer :
```python
# ACTUEL — cascade
scope1 = dropdown_base
if playlists_selected and len(playlists_selected) < len(playlist_values):
    scope1 = scope1.filter(...)
# mode_values depuis scope1
scope2 = scope1
if modes_selected and len(modes_selected) < len(mode_values):
    scope2 = scope2.filter(...)
# map_values depuis scope2
```

Par :
```python
# NOUVEAU — pas de cascade
# mode_values depuis dropdown_base (pas scope1)
mode_values = sorted({
    str(x).strip() for x in dropdown_base["mode_ui"].drop_nulls().to_list()
    if str(x).strip()
})
# map_values depuis dropdown_base (pas scope2)
map_values = sorted({
    str(x).strip() for x in dropdown_base["map_ui"].drop_nulls().to_list()
    if str(x).strip()
})
```

Les variables `scope1` et `scope2` sont supprimées.

**Impact UX** : les options Modes et Cartes ne disparaissent/réapparaissent plus quand on
coche/décoche une playlist. Le filtrage réel est fait dans `apply_filters()` qui combine
tous les filtres.

| Fichier | Modification |
|---------|-------------|
| `filters_render.py` | Supprimer scope1/scope2, calculer modes/maps depuis dropdown_base |

#### 2.2 Faire remonter `all_*` depuis `_render_cascade_filters`

Modifier le retour de `_render_cascade_filters` pour inclure les options complètes :

```python
def _render_cascade_filters(...) -> tuple[
    list[str], list[str], list[str],   # selected
    list[str], list[str], list[str],   # all options
]:
```

Ou, plus simplement, retourner un petit dataclass/namedtuple.

Cela permet à `render_filters_sidebar` de passer les `all_*` corrects au `save`.

| Fichier | Modification |
|---------|-------------|
| `filters_render.py` | Modifier retour de `_render_cascade_filters` |

### Phase 3 — Réconciliation mid-session (priorité moyenne)

**Objectif** : gérer le cas Période ↔ Sessions sans perte de sélection, et absorber
automatiquement les nouveaux contenus (nouvelles playlists/modes/cartes ajoutés lors
d'une mise à jour du jeu ou d'une intégration PVE).

#### 3.1 Helper `_reconcile_filter_options` dans `filters_render.py`

La réconciliation doit se faire dans `_render_cascade_filters` — **pas** dans
`render_checkbox_filter`. Raisons :

1. `render_checkbox_filter` est un composant générique qui ne devrait pas avoir de logique métier
2. `_render_cascade_filters` a déjà accès à toutes les options calculées (`playlist_values`, etc.)
3. La réconciliation doit être faite **avant** que `render_checkbox_filter` ne lise `session_state`

**Piège à éviter** : `new_options = set(options) - current_selection` inclurait des options que
l'utilisateur a délibérément décochées. En mode exclude, elles seraient re-cochées à chaque render.
Il faut distinguer "vraiment nouveau" (jamais vu, doit être auto-coché) de "délibérément décoché"
(dans les exclusions stockées, ne doit pas être touché).

```python
# filters_render.py — helper privé

def _reconcile_filter_options(
    filter_key: str,
    options: list[str],
    mode_key: str,
    exclusions_key: str,
) -> None:
    """Auto-coche les options vraiment nouvelles en mode exclude.

    "Vraiment nouvelle" = dans `options`, absente de session_state[filter_key]
    ET absente des exclusions explicites. Cela garantit que :
    - Les nouvelles playlists/modes/cartes ajoutés par un sync sont auto-cochés ✓
    - Les options que l'utilisateur a délibérément décochées restent décochées ✓
    - En mode include, rien ne change (nouvelles options restent décochées) ✓

    Doit être appelé AVANT render_checkbox_filter / render_hierarchical_checkbox_filter.
    """
    if filter_key not in st.session_state:
        return  # pas encore initialisé → render_checkbox_filter s'en charge
    if st.session_state.get(mode_key, "include") != "exclude":
        return
    exclusions: set[str] = st.session_state.get(exclusions_key, set())
    current: set[str] = st.session_state[filter_key]
    truly_new = set(options) - current - exclusions
    if truly_new:
        st.session_state[filter_key] = current | truly_new
```

#### 3.2 Appels dans `_render_cascade_filters`

Après avoir calculé `playlist_values`, `mode_values`, `map_values` (après Phase 2),
et **avant** les appels aux composants checkbox :

```python
# Réconciliation — doit précéder tous les render_checkbox_filter
_reconcile_filter_options(
    "filter_playlists", playlist_values,
    "_playlists_filter_mode", "_playlists_exclusions",
)
_reconcile_filter_options(
    "filter_modes", mode_values,
    "_modes_filter_mode", "_modes_exclusions",
)
_reconcile_filter_options(
    "filter_maps", map_values,
    "_maps_filter_mode", "_maps_exclusions",
)

# Puis les render habituels
playlists_selected = render_checkbox_filter(
    label="Playlists", options=playlist_values, session_key="filter_playlists", ...
)
modes_selected = render_hierarchical_checkbox_filter(
    label="Modes", options=mode_values, session_key="filter_modes", ...
)
maps_selected = render_checkbox_filter(
    label="Cartes", options=map_values, session_key="filter_maps", ...
)
```

**Couvre automatiquement** :
- Nouveaux modes de jeu ajoutés par 343/Xbox (ex: saison avec nouvelle playlist PVP)
- Nouvelles cartes ajoutées au jeu
- Intégration future PVE (nouvelles playlists Firefight, modes PVE)
- Transition Période ↔ Sessions (plus/moins d'options visibles)

| Fichier | Modification |
|---------|-------------|
| `filters_render.py` | Ajouter `_reconcile_filter_options` + appels dans `_render_cascade_filters` |

### Phase 4 — Tests (priorité haute, parallélisable avec Phase 1-2)

| Test | Description |
|------|-------------|
| `test_detect_filter_mode` | Heuristique 70/30 + hystérésis zone grise |
| `test_save_load_include_mode` | Save sélection faible → JSON `include` → load correct |
| `test_save_load_exclude_mode` | Save "tout sauf FF" → JSON `exclude` + `["Firefight"]` → load correct |
| `test_apply_exclude_with_all` | Apply exclude + `all_playlists` → session_state = all - exclusions |
| `test_apply_exclude_without_all` | Apply exclude + `all_playlists=None` → fallback inclusions |
| `test_backward_compat_no_mode` | JSON legacy sans `*_mode` → traité comme include |
| `test_new_playlist_auto_included_exclude_mode` | Exclude "Firefight" + nouvelle playlist "Slayer" dans all → Slayer cochée |
| `test_new_playlist_not_included_include_mode` | Include "Ranked" + nouvelle playlist "Slayer" dans all → Slayer pas cochée |
| `test_cycle_complet_save_json_load_apply` | Save → JSON sur disque → load → apply → session_state cohérent |

| Fichier | Modification |
|---------|-------------|
| `tests/test_filter_state.py` | Ajouter tous les tests ci-dessus |

### Phase 5 — UX optionnelle (priorité basse)

| Tâche | Fichier | Détail |
|-------|---------|--------|
| Indicateur mode actif | `filters_render.py` | `st.caption("Mode exclusion : Firefight exclu")` |
| Compteurs de matchs par option | `checkbox_filter.py` | `f"{opt} ({count})"` — **attention perf** : ne pas faire un group_by par option, calculer en un seul passage sur dropdown_base |

---

## Architecture finale

### `FilterPreferences` (filter_state.py)

```python
@dataclass
class FilterPreferences:
    filter_mode: str | None = None
    start_date: str | None = None
    end_date: str | None = None
    gap_minutes: int | None = None
    picked_session_label: str | None = None

    # En mode "exclude" : contient ce qui est DÉCOCHÉ
    # En mode "include" (ou None/legacy) : contient ce qui est COCHÉ
    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None

    playlists_mode: str | None = None   # "exclude" | "include" | None (legacy=include)
    modes_mode: str | None = None
    maps_mode: str | None = None
```

### Flux de sauvegarde

```
Rendu sidebar terminé (playlists_selected, playlist_values connus)
  → save_filter_preferences(all_playlists=playlist_values)
      → _detect_filter_mode(selected, all, current_mode) → mode
      → if mode == "exclude":
            preferences.playlists_selected = sorted(all - selected)   ← exclusions
            preferences.playlists_mode = "exclude"
        else:
            preferences.playlists_selected = sorted(selected)          ← inclusions
            preferences.playlists_mode = "include"
      → JSON sauvegardé
```

### Flux de chargement

```
Ouverture app / changement joueur
  → base_for_filters = df.clone()
  → pré-calcul _pre_playlists, _pre_modes, _pre_maps (base large)
  → load_filter_preferences() → FilterPreferences
  → apply_filter_preferences(all_playlists=_pre_playlists, ...)
      → mode = prefs.playlists_mode  (ex: "exclude")
      → session_state["filter_playlists"] = set(_pre_playlists) - {"Firefight"}
         ↑ nouvelle playlist "Slayer" dans _pre_playlists → auto-cochée ✓
  → render_checkbox_filter(options=playlist_values)
      → nettoyage obsolètes : current & set(options)
      → réconciliation nouvelles : if exclude → auto-cocher les nouvelles ✓
```

### Flux mid-session (changement contexte ou nouveau contenu)

```
Cas A — Switch Sessions → Période (plus de playlists disponibles)
  → _render_cascade_filters recalcule playlist_values (12 au lieu de 5)
  → _reconcile_filter_options("filter_playlists", 12_playlists, ...)
      → mode   = session_state["_playlists_filter_mode"] = "exclude"
      → excl   = session_state["_playlists_exclusions"]  = {"Firefight"}
      → current = session_state["filter_playlists"] = {A,B,C,D,E}  (5 de la session)
      → truly_new = {12} − {A,B,C,D,E} − {"Firefight"} = {G,H,I,J,K,L}
      → session_state["filter_playlists"] = {A..E, G..L}  ← 11 cochées ✓
      → Firefight reste décochée car dans exclusions ✓
  → render_checkbox_filter(options=12_playlists) — lit session_state déjà à jour

Cas B — Sync ajoute une nouvelle playlist "PVE : Flood Warzone"
  → Au prochain chargement, _pre_playlists contient la nouvelle playlist
  → apply_filter_preferences : session_state["filter_playlists"] = _pre_playlists − exclusions
  → "PVE : Flood Warzone" pas dans exclusions → auto-cochée ✓

Cas C — Intégration future d'un nouveau type d'expérience
  → Voir section "Extensibilité" ci-dessous
```

---

## Extensibilité — Nouveaux types d'expérience (PVE, PVP classé, PVP non classé)

### Contexte

L'intégration future du PVE (Firefight étendu, modes PvE dédiés) et d'autres modes PVP
introduira de nouvelles playlists et modes dans la DB. Deux niveaux d'extensibilité :

---

### Niveau 1 — Nouvelles valeurs dans les filtres existants (géré automatiquement)

De nouvelles playlists/modes/cartes ajoutés par un sync apparaissent automatiquement
dans les filtres existants. Le mécanisme intent-based les gère sans code supplémentaire :

| Nouveau contenu | Mode exclude | Mode include |
|-----------------|-------------|--------------|
| Nouvelle playlist PVE "Flood Warzone" | Auto-cochée ✓ | Décochée (non dans include) ✓ |
| Nouvelle carte "Nexus" | Auto-cochée ✓ | Décochée ✓ |
| Nouveau mode "PVE : Sabotage" | Auto-coché ✓ | Décoché ✓ |

Aucune modification de code requise pour ce niveau.

---

### Niveau 2 — Nouvelle dimension de filtre (ex: sélecteur PVE / PVP / Ranked)

Si le volume de contenu PVE devient significatif, un sélecteur de **type d'expérience**
en haut de la sidebar permettrait de pré-filtrer `dropdown_base` avant que
playlists/modes/cartes ne soient calculés.

**Architecture recommandée : sélecteur `st.multiselect` comme pré-filtre**

```
sidebar
  ┌─ Type d'expérience ──────────────────┐
  │ ☑ PVP non classé                     │  ← filtre dropdown_base
  │ ☑ PVP classé                         │
  │ ☐ PVE                                │
  └──────────────────────────────────────┘
  ┌─ Playlists ──────────────────────────┐
  │ options calculées après pré-filtre   │
  └──────────────────────────────────────┘
```

Ce sélecteur agirait **avant** la fenêtre temporelle dans `_render_cascade_filters`,
réduisant `dropdown_base` aux matchs du type sélectionné.

**Implémentation** (quand le moment viendra) :

```python
# 1. FilterPreferences — ajouter le champ
@dataclass
class FilterPreferences:
    ...
    experience_types: list[str] | None = None  # ["PVP non classé", "PVP classé", "PVE"]
    experience_types_mode: str | None = None   # "exclude" | "include"

# 2. FILTER_DATA_KEYS — ajouter les clés
"filter_experience_types",
"_experience_types_filter_mode",
"_experience_types_exclusions",

# 3. _render_cascade_filters — pré-filtrer dropdown_base
experience_selected = render_checkbox_filter(
    label="Type d'expérience",
    options=["PVP non classé", "PVP classé", "PVE"],
    session_key="filter_experience_types",
)
# Mapper les types vers les playlists concernées et filtrer dropdown_base
if experience_selected:
    dropdown_base = _apply_experience_filter(dropdown_base, experience_selected)

# Puis calculer playlist_values, mode_values, map_values depuis le dropdown_base filtré
# La réconciliation _reconcile_filter_options s'applique normalement ensuite
```

**Note** : le type d'expérience étant une liste fixe (définie dans le code, pas depuis la DB),
il ne nécessite pas de `_pre_experience_types` au chargement. Les valeurs sont statiques.
Le mécanisme include/exclude s'applique quand même pour la persistance.

**Quand implémenter** : uniquement quand des données PVE réelles seront disponibles dans
`shared_matches.duckdb`. Ne pas anticiper sur des données inexistantes.

---

### Niveau 3 — Ajout d'un filtre totalement nouveau (ex: rang MMR, taille d'équipe)

Le pattern est identique pour n'importe quel nouveau filtre checkbox :

1. Ajouter `*_selected: list[str] | None` + `*_mode: str | None` dans `FilterPreferences`
2. Ajouter `"filter_*"`, `"_*_filter_mode"`, `"_*_exclusions"` dans `FILTER_DATA_KEYS`
3. Calculer les options disponibles (depuis `dropdown_base` ou source statique)
4. Appeler `_reconcile_filter_options("filter_*", options, ...)` avant le render
5. Appeler `render_checkbox_filter(session_key="filter_*", ...)`
6. Faire remonter `all_*` dans le retour de `_render_cascade_filters` → passer à `save`
7. Ajouter `all_*` dans la signature de `save_filter_preferences` et `apply_filter_preferences`

---

---

## Critères de succès

- [ ] Test "tout sauf Firefight" : fermer/rouvrir l'app → Firefight décoché, nouvelles playlists cochées
- [ ] Test "mode include" : sélection unique Ranked → fermer/rouvrir → seul Ranked coché
- [ ] Test mid-session : switch Sessions→Période → nouvelles playlists auto-cochées si mode exclude
- [ ] Test réconciliation : option délibérément décochée reste décochée après switch de contexte
- [ ] Test nouvelle playlist après sync : présente dans `_pre_playlists` → auto-cochée au prochain chargement
- [ ] `grep -r "scope1\|scope2" src/app/filters_render.py` = 0 résultat
- [ ] JSON legacy (sans `*_mode`) continue de fonctionner (backward compat)
- [ ] Suite pytest complète : OK

---

## Relations avec les autres docs

| Doc existant | Action |
|---|---|
| Docs de la branche `copilot/refactor-filter-selection-functionality` | Ignorer (branche non mergée) |
| `ANALYSE_FILTRES_SIDEBAR_2026-02-18.md` | Archiver (diagnostic intégré ici) |
| `SOLUTION_CONCRETE_EXCLUSION.md` | Archiver (architecture intégrée ici) |

---

**Dernière mise à jour** : 2026-02-20
**Auteur** : Revue humaine + Claude
**Statut** : ✅ Plan validé, implémentation Phase 1+2 prioritaire
