# Plan détaillé — Refacto traductions d'assets : DB = source de vérité

**Date :** 2026-04-02  
**Branche de travail :** `refactor/asset-translations-db-first`  
**Backlog source :** `.ai/BACKLOG.md` — § "[refacto] Traductions d'assets"

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

**Item 3 — Carte DB-first : partiellement déjà là**  
Dans `_filters_apply.py::_add_derived_columns`, `normalize_map_label_fn` est déjà invoquée **en dernier recours**, après lecture de `map_name_fr` (colonne du DF issue de `v_match_full`). La route DB-first existe. L'item 3 se réduit à une vérification + nettoyage du nom.

**Ordre recommandé dans le backlog (1→2→4→3→5→6) — à réviser :**  
L'item 2 doit impérativement venir **après** l'item 4. Remplacer `_normalize_mode_label` par `normalize_mode_label` avant de découpler son `st.session_state` revient à introduire le couplage UI là où il n'existait pas encore.

### 1.3 Ordre corrigé

```
1 → 4 → 2 → 3 → (5 : vérification seule) → 6
```

---

## 2. Analyse technique détaillée par item

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

### 2.4 Item 3 — Vérifier et purger `normalize_map_label` comme chemin principal

**Situation actuelle dans `_filters_apply.py` :**  
`_add_derived_columns` (L336–L383) applique déjà la logique DB-first :
```
1. Si map_name_fr dans le DF (colonne v_match_full) → l'utiliser
2. Sinon fallback normalize_map_label_fn(map_name)
```

Ce qui correspond exactement à l'objectif du backlog.

**Ce qu'il reste à faire :**

a) **Audit des autres call-sites de `normalize_map_label` dans les filtres** :
- `filters.py` L87 : `build_mapping(df["map_name"], normalize_map_label)` — **problème potentiel** : ce `build_mapping` construit le mapping de labels *avant* l'application de `_add_derived_columns`. Si `map_name_fr` n'est pas encore dans le DF à ce stade, le mapping utilise uniquement `normalize_map_label(map_name)`. Vérifier si `v_match_full` est chargé avant ce `build_mapping`.
- `filters_render.py` L332, 342, 385, 399, 462 : `normalize_map_label_fn` propagée — usage légitime (render des options de filtres).

b) **Garder la guard UUID** : `normalize_map_label` doit rester le fallback explicite pour les cas où `map_name_fr` est `NULL` (map inédite, sync partiel). Ne pas la supprimer — juste ne plus l'appeler en chemin principal là où `map_name_fr` est disponible.

c) **Action concrète** : dans `filters.py::build_mapping` (L87), vérifier si on peut passer `map_name_fr` plutôt que `map_name` comme source. Si le DF exposé à ce point a déjà `map_name_fr`, permuter. Sinon, c'est déjà correct (fallback UUID guard).

**Risque :** faible. Le chemin DB-first était déjà le chemin principal — il s'agit d'une vérification et d'un potentiel ajustement dans `filters.py`.

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
| 1 | Créer `src/utils/strings.py` + déplacer `is_uuid_like` | `strings.py` (new), `translations.py`, `helpers.py` | 15 min | Nul |
| 4 | Découpler `normalize_mode_label` de `st.session_state` | `helpers.py`, `streamlit_app.py` (3 sites) | 45 min | Modéré |
| 2 | Supprimer `_normalize_mode_label` (après item 4) | `teammates_helpers.py` (2 lignes) | 10 min | Faible |
| 3 | Vérifier DB-first cartes dans `filters.py` | `filters.py` L87 (1 line au pire) | 30 min | Faible |
| 5 | **Aucun code** — valider + barrer le backlog | — | 5 min | Nul |
| 6 | Vérifier cache `build_mapping` dans `cache_filters.py` | `cache_filters.py` si absent | 15 min | Nul |

**Total estimé : ~2h** (vs 5h30 estimé dans le backlog — l'item 5 était déjà fait).

---

## 4. Checklist d'implémentation

### Phase 1 — `src/utils/strings.py` (Item 1)

- [ ] Créer `src/utils/strings.py` avec `is_uuid_like(s: str) -> bool`
- [ ] `translations.py` : supprimer `_is_uuid_like`, ajouter import `from src.utils.strings import is_uuid_like`, remplacer les 2 usages
- [ ] `helpers.py` : supprimer la définition locale, ajouter import
- [ ] Lancer `python -m pytest tests/ -q --ignore=tests/integration` → vert

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
- [ ] Vérifier visuellement la page Coéquipiers : labels de modes désormais complets (Forge/Ranked nettoyés)

### Phase 4 — Cartes DB-first (Item 3)

- [ ] Inspecter `filters.py` L87 : est-ce que `df["map_name_fr"]` est disponible dans le DF passé à `build_mapping` ?
  - Si oui → remplacer `df["map_name"]` par `df["map_name_fr"].fill_null(df["map_name"])`
  - Si non → confirmer que ce path est correct et laisser tel quel (garde UUID)
- [ ] Revalider `_filters_apply.py` : confirmer que `map_name_fr` est prioritaire avant `normalize_map_label_fn`
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
- **`st.session_state` interdit dans `src/app/helpers.py`** — la fonction helpers.py est testable sans Streamlit
- **`src/utils/strings.py`** doit rester un module pur (0 import Streamlit, 0 import DuckDB)
- **Pas de guard `_has_shared_view`** — les vues v6 sont garanties présentes
- **Tests obligatoires après chaque phase** : `python -m pytest tests/ -q --ignore=tests/integration`
- **Taille fichiers** : `helpers.py` et `teammates_helpers.py` restent sous 500L, fonctions sous 80L

---

## 6. Branche Git

```bash
git checkout -b refactor/asset-translations-db-first
```

Commits séquentiels par phase :
```
feat(utils): ajouter src/utils/strings.py avec is_uuid_like unifié
refactor(helpers): découpler normalize_mode_label de st.session_state
refactor(teammates): supprimer _normalize_mode_label, utiliser normalize_mode_label
refactor(filters): vérifier DB-first pour map_name_fr dans build_mapping
docs(backlog): barrer items 5 et 6 (déjà implémentés)
```
