# Plan détaillé — Refacto traductions d'assets : DB = source de vérité

**Date :** 2026-04-02 (mis à jour 2026-04-03)
**Branche de travail :** `refactor/asset-translations-db-first`  
**Backlog source :** `.ai/BACKLOG.md` — § "[refacto] Traductions d'assets"

---

## 0. Diagnostic racine — Pourquoi les fixes sont éparpillés

### Le pipeline actuel (cassé)

```
load_df_optimized()          → df  (a map_name_fr, PAS map_ui)
       ↓
_filters_apply / _cascade    → dff (a map_ui) ← lang-aware ✅
       ↓
base, full_df, fr_sub...     → pas de map_ui  ← chaque page corrige localement ❌
```

**3 implémentations parallèles de la même logique** existent aujourd'hui :

| Symbole | Fichier | Défaut |
|---------|---------|--------|
| `add_ui_columns()` | `filters.py` | Ignore `map_name_fr`, utilise toujours `map_name` EN |
| `_add_derived_columns()` | `_filters_apply.py` | Correcte mais uniquement appliquée à `dff` |
| `_vectorize_ui_columns()` | `_filters_cascade.py` | Idem |
| patches ad-hoc | `win_loss.py`, `teammates_views.py`, `friends_impact_heatmap.py` | Rustines symptomatiques |

**La vraie cause racine** : `df` (sortie de `load_df_optimized`) a déjà `map_name_fr` (depuis `v_match_full`), mais `map_ui` n'est jamais calculé à ce stade. Toute sous-sélection de `df` avant filtrage est borgne, obligeant chaque page à patcher localement.

### La solution structurelle : Item 0

Créer **`src/app/i18n_columns.py`** avec une fonction unique `add_i18n_display_columns(df, lang)`, appelée **une seule fois** juste après `load_df_optimized()`. Ainsi :

- `df` (stocké dans `st.session_state`) a toujours `map_ui`, `playlist_ui`, `mode_ui`
- `dff` (sous-ensemble filtré de `df`) en hérite automatiquement — les 3 implémentations deviennent des no-ops (elles ont déjà les guards `if "map_ui" not in df.columns`)
- `base`, `full_df` et tout sous-ensemble de `df` en héritent aussi
- `fr_sub` (données coéquipiers, chargé séparément) est traité par un second appel à la même fonction

---

## 1. Revue de la section backlog

### 1.1 Ce qui est exact

| Item | Statut |
|------|--------|
| Deux implémentations dupliquées de `is_uuid_like` | ✅ Confirmé |
| `_normalize_mode_label` dans `teammates_helpers.py` est redondante | ✅ Confirmé (avec nuance — voir §1.2) |
| `normalize_mode_label` couple `st.session_state` à la logique métier | ✅ Confirmé |
| `translate_playlist_name` n'a pas de cache | ✅ Confirmé |

### 1.2 Corrections et nuances à apporter

**Item 2 — Diagnostic plus précis de `_normalize_mode_label` :**  
Le backlog dit "réimplémente `normalize_mode_label`". En réalité, c'est pire : la version locale ne fait qu'appeler `translate_pair_name(pair_name)`, sans `clean_asset_label`, sans stripping `" on "`, sans nettoyage Forge/Ranked. Remplacer par `normalize_mode_label` serait un **gain de justesse** (comportement plus complet), pas seulement un DRY. Attention : cela dépend de l'Item 4 — voir §2.3.

**Item 5 — Armes : déjà implémenté**  
`resolve_weapon_display` dans `src/analysis/_weapon_data.py` suit déjà l'ordre correct :
1. Fusion canonique (WEAPON_FUSION_MAP_ID)
2. DB via `_resolve_weapon_cached` (lru_cache 512, `metadata.duckdb → weapon_labels`)
3. Fallback dicts Python

**Aucun code à écrire pour l'item 5.** Le backlog l'a estimé à 1h — c'est une dette déjà apurée. À valider (§4.5) puis à barrer.

**Item 3 — Carte DB-first : résolu par Item 0**  
Avec `add_i18n_display_columns` appliqué dès le chargement, `map_name_fr` est toujours le chemin principal et `normalize_map_label_fn` reste le fallback explicite (cas map inédite / sync partiel). L'item 3 se réduit à supprimer `add_ui_columns()` de `filters.py` (qui ignorait `map_name_fr`) et vérifier `cache_filters.py` L87.

**Ordre recommandé dans le backlog (1→2→4→3→5→6) — à réviser :**  
L'item 2 doit impérativement venir **après** l'item 4. Remplacer `_normalize_mode_label` par `normalize_mode_label` avant de découpler son `st.session_state` revient à introduire le couplage UI là où il n'existait pas encore.

### 1.3 Ordre corrigé (avec Item 0 prérequis)

```
0 (couche centralisée) → 1 (is_uuid_like) → 4 (découplage st) → 2 (teammates) → 3 (nettoyage) → (5 : vérification) → 6
```

---

## 2. Analyse technique détaillée par item

### 2.0 Item 0 — Créer `src/app/i18n_columns.py` (couche centralisée) ⭐ PRÉREQUIS

**Motivation :**  
La multiplication des rustines locales (rustines ajoutées sur `win_loss.py`, `teammates_views.py`, `friends_impact_heatmap.py`) est le symptôme d'une architecture où la traduction arrive trop tard dans le pipeline. Ce refacto déplace le calcul de `map_ui` / `playlist_ui` / `mode_ui` au point de chargement.

**Pipeline cible :**
```
load_df_optimized()            → df brut (map_name_fr présent, map_ui absent)
       ↓
add_i18n_display_columns(df, lang)   ← NOUVEAU — appelée UNE SEULE FOIS
       ↓
df enrichi (map_ui, playlist_ui, mode_ui présents) ← stocké dans session_state
       ↓
dff = apply_filters(df, ...)   → hérite automatiquement de map_ui ✅
base, full_df, ...             → héritent automatiquement de map_ui ✅
fr_sub (teammates)             → add_i18n_display_columns() aussi ✅
```

**Nouveau fichier `src/app/i18n_columns.py` :**

```python
"""Couche centralisée d'enrichissement i18n des DataFrames de matchs.

Ce module est la source de vérité unique pour le calcul de map_ui,
playlist_ui et mode_ui. Il doit être appelé une seule fois après
load_df_optimized(), jamais dans les pages individuelles.
"""
from __future__ import annotations
import polars as pl


def add_i18n_display_columns(
    df: pl.DataFrame,
    lang: str,
    *,
    normalize_mode_label_fn,
    normalize_map_label_fn,
    translate_playlist_name_fn,
    clean_asset_label_fn,
) -> pl.DataFrame:
    """Ajoute map_ui, playlist_ui, mode_ui au DataFrame.

    Logique (par ordre de priorité pour chaque colonne) :
    - map_ui      : map_name_fr (DB) → normalize_map_label_fn(map_name) (fallback UUID)
    - playlist_ui : playlist_name_fr (DB) → translate_playlist_name_fn(playlist_name)
    - mode_ui     : pair_name_fr (DB) → normalize_mode_label_fn(pair_name)

    Cette fonction est idempotente : si la colonne existe déjà, elle n'est pas recalculée.
    """
    ...  # implémentation : cf. logique existante de _add_derived_columns
```

**Call-site : `streamlit_app.py` juste après `load_df_optimized()`**

```python
# Avant (actuel — map_ui calculé trop tard)
df = load_df_optimized(...)

# Après (centralisé)
df = load_df_optimized(...)
df = add_i18n_display_columns(
    df, lang=get_lang(),
    normalize_mode_label_fn=...,
    normalize_map_label_fn=...,
    translate_playlist_name_fn=translate_playlist_name,
    clean_asset_label_fn=clean_asset_label,
)
st.session_state["df"] = df
```

**Périmètre complet des hotfixes pré-Item0 (marqués `# Hotfix pré-Item0` dans le code, à supprimer après déploiement de l'Item 0) :**

| Fichier | Hotfix | Commit |
|---------|--------|--------|
| `win_loss.py` L261 | injection `map_ui` sur `base` | pré-df361f0 |
| `teammates_views.py` L289 | injection `map_ui` sur `fr_sub` | pré-df361f0 |
| `friends_impact_heatmap.py` | injection `map_ui` dans `plot_squad_map_heatmap` | 6df05c6 |
| `_teammates_trio.py` | injection `map_ui` sur `_me_full` | df361f0 |
| `_query_teammate_shared_stats` (SQL) | alias `map_ui` dans SELECT | df361f0 |
| `query_teammate_full_history` (SQL) | `JOIN v_match_full` + alias `map_ui` | df361f0 |
| `compute_squad_records_per_map` | guard dynamique `_map_col = "map_ui" if … else "map_name"` | df361f0 |

Le commit `df361f0` ("fix(maps): noms FR dans graphes après radar") a corrigé 4 composants du pipeline de la page "Complémentarité de l'escouade" où les graphes post-radar affichaient encore les noms EN :

1. **`f1_df/f2_df/f3_df`** — issus de `_query_teammate_shared_stats` : avaient `map_name_fr` mais pas `map_ui` → ajout de `COALESCE(r.map_name_fr, r.map_name, '') AS map_ui` dans le SELECT
2. **`_f1_full/_f2_full/_f3_full`** — issus de `query_teammate_full_history` : jointure sur `match_registry` (sans `map_name_fr`) → remplacée par `JOIN shared.v_match_full` + ajout `map_name_fr`/`map_ui`
3. **`_me_full`** — sous-ensemble de `df` (historique joueur principal) : `df` n'a pas `map_ui` avant filtrage → injection Python post-query dans `_teammates_trio.py`
4. **`records_per_map`** — dict de `compute_squad_records_per_map` : clés EN (`map_name`) vs labels FR des axes → guard dynamique `_map_col`

**Conséquences de l'Item 0 :**
- `_add_derived_columns` et `_vectorize_ui_columns` deviennent des no-ops pour `map_ui` (guards `if "map_ui" not in df.columns:` déjà présentes) — pas de suppression précoce, mais la logique n'est plus dupliquée
- `add_ui_columns()` dans `filters.py` devient obsolète → supprimer après validation
- Les 7 hotfixes listés ci-dessus → supprimer (code mort une fois l'Item 0 déployé)
- `fr_sub` dans `teammates_service.py` : appeler `add_i18n_display_columns(fr_sub, lang, ...)` dans `TeammatesService.load_teammate_stats` après la query SQL (remplace les `COALESCE` SQL partiels ajoutés dans df361f0)

**Risque :** modéré. `streamlit_app.py` est le point central — un test de régression end-to-end (filtres + affichage) est nécessaire. Les guards idempotentes dans `_add_derived_columns` et `_filters_cascade` protègent contre la double exécution.

---

### 2.1 Item 1 — Unifier `is_uuid_like` dans `src/utils/strings.py`

**Situation actuelle :**

| Fichier | Symbole | Pattern |
|---------|---------|---------|
| `src/ui/translations.py` L26 | `_is_uuid_like(s: str) -> bool` | import `re` lazy à l'intérieur |
| `src/app/helpers.py` L50 | `is_uuid_like(s: str) -> bool` | import `re` module-level |

Regex identique dans les deux cas : `^[a-f0-9]{8}(-[a-f0-9]{4}){0,3}(-[a-f0-9]{1,12})?$`

`src/utils/strings.py` **n'existe pas encore**.

**Callers post-migration :**
- `translations.py` L53, L77 → `_is_uuid_like` (interne) → devient `is_uuid_like` importé
- `helpers.py` L99 → `is_uuid_like` (local) → devient `is_uuid_like` importé

**Ce qu'il faut faire :**
1. Créer `src/utils/strings.py` avec `is_uuid_like` (+ import `re` module-level)
2. Dans `translations.py` : supprimer `_is_uuid_like`, importer `is_uuid_like` depuis `src.utils.strings`
3. Dans `helpers.py` : supprimer la définition locale, importer depuis `src.utils.strings`

**Risque :** quasi nul. Les deux callers restent dans leurs fichiers respectifs. Seul le symbole bouge.

---

### 2.2 Item 4 — Extraire `st.session_state` de `normalize_mode_label`

**Situation actuelle (helpers.py L53–L78) :**
```python
def normalize_mode_label(pair_name: str | None) -> str | None:
    ...
    _settings = st.session_state.get("app_settings")          # ← couplage UI/métier
    _normalize = getattr(_settings, "normalize_mode_labels", True) if _settings else True
    _translated = translate_pair_name(base, lang=get_lang(), normalize=_normalize)
```

**Callers dans `streamlit_app.py` (L673, L690, L711) :** passent `normalize_mode_label` comme callback `normalize_mode_label_fn=normalize_mode_label`.

**Ce qu'il faut faire :**

Transformer la signature pour accepter `lang` et `normalize` en paramètres explicites avec valeurs par défaut :

```python
def normalize_mode_label(
    pair_name: str | None,
    *,
    lang: str = "fr",
    normalize: bool = True,
) -> str | None:
    ...
    _translated = translate_pair_name(base, lang=lang, normalize=normalize)
```

Supprimer les imports `st` et `get_lang` de `helpers.py` si plus utilisés ailleurs.

**Adapter les 3 call-sites dans `streamlit_app.py` :**
```python
# Avant
normalize_mode_label_fn=normalize_mode_label

# Après (créer un wrapper dans streamlit_app.py, ou passer un partial)
_settings = st.session_state.get("app_settings")
_lang = get_lang()
_norm = getattr(_settings, "normalize_mode_labels", True) if _settings else True
normalize_mode_label_fn = lambda p: normalize_mode_label(p, lang=_lang, normalize=_norm)
```

> Ou utiliser `functools.partial` si la lambda n'est pas appropriée. À décider à l'implémentation.

**Risque :** modéré. La signature change — mais les call-sites sont peu nombreux (3 dans `streamlit_app.py` + 1 indirect dans `cache_filters.py` via callback). Revue des 3 sites obligatoire.

---

### 2.3 Item 2 — Supprimer `_normalize_mode_label` dans `teammates_helpers.py`

**Dépend de l'item 4 (§2.2).**

**Situation actuelle (`teammates_helpers.py` L40–L44) :**
```python
def _normalize_mode_label(pair_name: str | None) -> str | None:
    """Normalise un pair_name en label UI."""
    from src.ui.translations import translate_pair_name
    return translate_pair_name(pair_name) if pair_name else None
```

Appelée L203 : `_mode_map = build_mapping(friends_table["pair_name"], _normalize_mode_label)`

**Problème** : cette version locale appelle uniquement `translate_pair_name` — elle rate :
- `clean_asset_label` (suppression parenthèses, gestion None/whitespace)
- stripping `" on MapName"`
- regex Forge/Ranked
- respect de `normalize_mode_labels` du profil

**Ce qu'il faut faire (après item 4) :**
1. Supprimer la fonction `_normalize_mode_label` (L40–L44)
2. Importer `normalize_mode_label` depuis `src.app.helpers`
3. Adapter L203 :
   ```python
   _mode_map = build_mapping(
       friends_table["pair_name"],
       lambda p: normalize_mode_label(p, lang=get_lang(), normalize=_normalize_setting),
   )
   ```
   où `_normalize_setting` est lu depuis `st.session_state` une fois en début de fonction.

**Risque :** faible après item 4. Gain : label modes coéquipiers devient cohérent avec le reste de l'app (actuellement il manque le nettoyage Forge/Ranked).

---

### 2.4 Item 3 — Nettoyer `normalize_map_label` et `add_ui_columns` après Item 0

**Résolu structurellement par Item 0.** Ce qui reste :

a) **Supprimer `add_ui_columns()` dans `filters.py`** — cette fonction ignorait `map_name_fr` et sera rendue obsolète par Item 0. Vérifier qu'il n'y a plus de caller externe avant suppression (seul `filters.py` L315 l'appelait sur `dropdown_base` — à remplacer par un appel à `add_i18n_display_columns`).

b) **Supprimer les rustines symptomatiques** dans `win_loss.py` (L261), `teammates_views.py` (L289), `friends_impact_heatmap.py` — devenues code mort une fois que `df` porte toujours `map_ui`.

c) **Garder la guard UUID** dans `normalize_map_label` : reste le fallback légitime pour maps inédites ou sync partiel — ne pas la supprimer.

d) **`filters_render.py`** (L332, 342, 385, 399, 462) : `normalize_map_label_fn` propagée pour le render des options — usage légitime, conserver.

**Risque :** faible. Suppressions de code mort testées via `tests/test_imports.py` + suite complète.

---

### 2.5 Item 5 — Armes DB-first (DÉJÀ IMPLÉMENTÉ — validation seule)

`resolve_weapon_display` dans `src/analysis/_weapon_data.py` (L265–L295) suit déjà l'ordre correct :
```
fusion canonique → _resolve_weapon_cached (DB/lru_cache) → dicts Python
```

**Action :** aucun code. Barrer cet item dans le backlog. Optionnel : valider que `weapon_labels` couvre bien 100% des `WEAPON_INT_TO_NAME` connus via un script de diagnostic.

---

### 2.6 Item 6 — Cache `@st.cache_data` sur `translate_playlist_name`

**Situation actuelle (`translations.py` L34–L62) :**  
La fonction est presque un passthrough pur :
1. Guard None/vide
2. Guard UUID → `_UNKNOWN_PLAYLIST.get(lang, s)` (dict constant)
3. `return s` (passthrough)

Aucun appel SQL ou I/O. Le coût est négligeable. Le seul appel notable dans `cache_filters.py` est déjà sous `@st.cache_data`. Dans `_filters_apply.py`, elle est appelée dans `_add_derived_columns` uniquement en fallback (colonne `playlist_name_fr` absente du DF).

**Verdict :** le cache `@st.cache_data` apporterait peu de valeur ici — la fonction n'a pas d'état ni d'I/O. En revanche, si `build_mapping` dans `cache_filters.py` n'est pas lui aussi mis en cache, **c'est là** que le coût se paie.

**Action recommandée :** plutôt que wrapper `translate_playlist_name`, vérifier que `build_mapping(dfr["playlist_name"], translate_playlist_name)` dans `cache_filters.py` est bien sous un `@st.cache_data`. Si oui, item terminé sans code. Si non, placer le cache autour du `build_mapping`.

---

## 3. Résumé des actions

| # | Item | Fichiers modifiés | Effort réel | Risque |
|---|------|-------------------|-------------|--------|
| **0** | **Créer `src/app/i18n_columns.py` + call-site dans `streamlit_app.py`** | `i18n_columns.py` (new), `streamlit_app.py`, `teammates_service.py` | **1h** | **Modéré** |
| 1 | Créer `src/utils/strings.py` + déplacer `is_uuid_like` | `strings.py` (new), `translations.py`, `helpers.py` | 15 min | Nul |
| 4 | Découpler `normalize_mode_label` de `st.session_state` | `helpers.py`, `streamlit_app.py` (3 sites) | 45 min | Modéré |
| 2 | Supprimer `_normalize_mode_label` (après item 4) | `teammates_helpers.py` (2 lignes) | 10 min | Faible |
| 3 | Supprimer `add_ui_columns` + 7 hotfixes pré-Item0 (après item 0) | `filters.py`, `win_loss.py`, `teammates_views.py`, `friends_impact_heatmap.py`, `_teammates_trio.py`, `teammates_service.py`, `_teammates_history_queries.py`, `squad_records.py` | 30 min | Faible |
| 5 | **Aucun code** — valider + barrer le backlog | — | 5 min | Nul |
| 6 | Vérifier cache `build_mapping` dans `cache_filters.py` | `cache_filters.py` si absent | 15 min | Nul |

**Total estimé : ~2h40** (Item 0 ajouté, item 5 toujours sans code, Phase 4 étendue aux hotfixes df361f0).

---

## 4. Checklist d'implémentation

### Phase 0 — Couche centralisée `src/app/i18n_columns.py` ⭐ PREMIER

- [ ] Créer `src/app/i18n_columns.py` avec `add_i18n_display_columns(df, lang, *, normalize_mode_label_fn, normalize_map_label_fn, translate_playlist_name_fn, clean_asset_label_fn) -> pl.DataFrame`
  - Extraire la logique de `_add_derived_columns` (chemin `map_name_fr` prioritaire, fallback `normalize_map_label_fn`)
  - Idempotente : guard `if "map_ui" not in df.columns:` pour chaque colonne
  - 0 import Streamlit — module pur testable
- [ ] Dans `streamlit_app.py` : appeler `add_i18n_display_columns(df, lang=get_lang(), ...)` immédiatement après `load_df_optimized()`, avant de stocker dans `session_state["df"]`
- [ ] Dans `TeammatesService.load_teammate_stats` (`teammates_service.py`) : appeler `add_i18n_display_columns(fr_sub, lang, ...)` après la query SQL (remplace le `COALESCE` SQL partiel par la logique Python complète)
- [ ] Écrire un test unitaire `tests/test_i18n_columns.py` : vérifier que `map_ui` est `map_name_fr` quand présent, `normalize_map_label_fn(map_name)` sinon, et idempotence
- [ ] Tests → vert

### Phase 1 — `src/utils/strings.py` (Item 1)

- [ ] Créer `src/utils/strings.py` avec `is_uuid_like(s: str) -> bool`
- [ ] `translations.py` : supprimer `_is_uuid_like`, ajouter import `from src.utils.strings import is_uuid_like`, remplacer les 2 usages
- [ ] `helpers.py` : supprimer la définition locale, ajouter import
- [ ] Tests → vert

### Phase 2 — Découpler `st.session_state` de `normalize_mode_label` (Item 4)

- [ ] Modifier signature dans `helpers.py` : ajouter `lang: str = "fr"` et `normalize: bool = True`
- [ ] Supprimer les lignes `_settings = st.session_state...` et `_normalize = ...` — déplacer dans les call-sites
- [ ] Vérifier si `st` et `get_lang` sont encore utilisés ailleurs dans `helpers.py`; supprimer les imports si plus nécessaires
- [ ] Adapter les 3 call-sites dans `streamlit_app.py` (lambda ou `functools.partial`)
- [ ] Vérifier `cache_filters.py` si `normalize_mode_label` y est aussi appelée directement
- [ ] Tests → vert

### Phase 3 — Supprimer `_normalize_mode_label` (Item 2)

- [ ] Supprimer `def _normalize_mode_label` (L40–L44) dans `teammates_helpers.py`
- [ ] L203 : remplacer par `lambda p: normalize_mode_label(p, lang=..., normalize=...)` avec les paramètres corrects issus de `st.session_state`
- [ ] Importer `normalize_mode_label` depuis `src.app.helpers`
- [ ] Tests → vert

### Phase 4 — Nettoyer le code mort (Item 3, après Item 0)

- [ ] Supprimer `add_ui_columns()` dans `filters.py` (remplacée par `add_i18n_display_columns`) + caller `filters.py` L315 → remplacer par `add_i18n_display_columns`
- [ ] Supprimer le patch ad-hoc dans `win_loss.py` L261 (`# Hotfix pré-Item0`, code mort)
- [ ] Supprimer le patch ad-hoc dans `teammates_views.py` L289 (`# Hotfix pré-Item0`, code mort)
- [ ] Supprimer le patch ad-hoc dans `friends_impact_heatmap.py` (code mort)
- [ ] Supprimer le patch ad-hoc dans `_teammates_trio.py` `_me_full` (`# Hotfix pré-Item0`, code mort)
- [ ] Supprimer les alias SQL `map_ui` de `_query_teammate_shared_stats` (remplacé par l'appel `add_i18n_display_columns` sur `fr_sub`)
- [ ] Supprimer les alias SQL `map_ui` de `query_teammate_full_history` (idem)
- [ ] Supprimer la guard dynamique `_map_col` dans `compute_squad_records_per_map` (utiliser `"map_ui"` directement, garanti présent)
- [ ] Tests → vert

### Phase 5 — Validation armes (Item 5)

- [ ] Lire `src/analysis/_weapon_data.py` L265–L295 → confirmer DB-first en place
- [ ] Barrer item 5 dans `.ai/BACKLOG.md`

### Phase 6 — Cache playlist (Item 6)

- [ ] Lire `src/ui/cache_filters.py` L95 → confirmer si déjà sous `@st.cache_data`
- [ ] Si oui : barrer item 6 dans le backlog (rien à faire)
- [ ] Si non : wraper le `build_mapping(dfr["playlist_name"], translate_playlist_name)` dans un cache

---

## 5. Contraintes architecture à respecter

- **Pandas PROSCRIT** — aucun usage, tout reste Polars
- **`src/app/i18n_columns.py`** doit rester un module pur sans import Streamlit — testable en isolation
- **`st.session_state` interdit dans `src/app/helpers.py`** — la fonction helpers.py est testable sans Streamlit
- **`src/utils/strings.py`** doit rester un module pur (0 import Streamlit, 0 import DuckDB)
- **Pas de guard `_has_shared_view`** — les vues v6 sont garanties présentes
- **Tests obligatoires après chaque phase** : `python -m pytest tests/ -q --ignore=tests/integration`
- **Taille fichiers** : tous les nouveaux modules restent sous 500L, fonctions sous 80L

---

## 6. Branche Git

```bash
git checkout -b refactor/asset-translations-db-first
```

Commits séquentiels par phase :
```
feat(i18n): créer src/app/i18n_columns.py — couche centralisée map_ui/playlist_ui/mode_ui
feat(utils): ajouter src/utils/strings.py avec is_uuid_like unifié
refactor(helpers): découpler normalize_mode_label de st.session_state
refactor(teammates): supprimer _normalize_mode_label, utiliser normalize_mode_label
refactor(ui): supprimer add_ui_columns + rustines symptomatiques (code mort)
docs(backlog): barrer items 5 et 6 (déjà implémentés)
```
refactor(filters): vérifier DB-first pour map_name_fr dans build_mapping
docs(backlog): barrer items 5 et 6 (déjà implémentés)
```
