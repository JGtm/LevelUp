# Problèmes de logique identifiés — 2026-03-03

Liste des bugs et dettes techniques détectés par analyse statique du code.
À traiter en priorité lors d'un prochain sprint dédié.

---

## 🔴 Bug majeur

### 1. Incohérence du calcul `win_rate`

**Fichiers concernés** :
- `src/data/query/analytics.py` (lignes 163, 193, 460, 487)
- `src/data/query/trends.py` (lignes 244, 275, 308)

**Problème** : deux formules incompatibles selon la page :

```sql
-- analytics.py : dénominateur = TOUS les matchs (y compris DNF)
SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END) * 1.0 / COUNT(*) as win_rate

-- trends.py : dénominateur = WIN + LOSS uniquement
SUM(...) / NULLIF(SUM(CASE WHEN outcome IN (2, 3) THEN 1 ELSE 0 END), 0) as win_rate
```

**Impact** : un joueur avec des matchs DNF voit un `win_rate` différent sur les stat cards vs les graphes de tendance.

**Correction attendue** : uniformiser sur `NULLIF(SUM(CASE WHEN outcome IN (2, 3) THEN 1 ELSE 0 END), 0)` (exclure TIE et DNF du dénominateur) dans tous les fichiers.

---

## 🟠 Dettes techniques

### 2. Compatibility guard obsolète : `_PERF_SCORE_AVAILABLE`

**Fichier** : `src/data/sync/_performance.py` (lignes 17–32)

```python
try:
    import polars as pl
    from src.analysis.performance_score import compute_relative_performance_score
    _PERF_SCORE_AVAILABLE = True
except ImportError:
    ...
    _PERF_SCORE_AVAILABLE = False
```

Polars et `performance_score` sont des dépendances obligatoires du projet. Ce guard ne sera jamais `False`. Anti-pattern "compatibility guard forever" (CLAUDE.md §2).

**Correction** : supprimer le try/except, importer directement.

---

### 3. Dead code : `_ensure_performance_score_column()`

**Fichier** : `src/data/sync/_performance.py` (ligne 36)

```python
def _ensure_performance_score_column(self) -> None:
    """V5 finale - Plus nécessaire ... conservée pour compatibilité."""
    pass
```

Méthode vide conservée "pour compatibilité" sans justification active. Anti-pattern "dead code museum".

**Correction** : supprimer la méthode + vérifier qu'elle n'est appelée nulle part.

---

### 4. Magic number `outcome == 4` dans le transformeur

**Fichier** : `src/data/sync/transformers/_match.py` (ligne 187)

```python
left_early = outcome == 4  # DidNotFinish
```

L'enum `Outcome.DID_NOT_FINISH` existe dans `src/data/domain/refdata.py` mais n'est pas utilisé ici.

**Correction** : `from src.data.domain.refdata import Outcome` puis `left_early = outcome == Outcome.DID_NOT_FINISH`.

---

### 5. NaN-check fragile dans `match_view.py`

**Fichier** : `src/ui/pages/match_view.py` (ligne 501)

```python
outcome_code = int(last_outcome) if last_outcome == last_outcome else None
```

Idiome NaN flottant (`NaN != NaN`). Si `last_outcome` est `None` ou `str`, la condition est `True` et `int(None)` lève une `TypeError`.

**Correction** : `outcome_code = int(last_outcome) if last_outcome is not None else None` (avec `try/except` si le type n'est pas garanti).

---

### 6. Magic numbers SQL dans `analytics.py`

**Fichier** : `src/data/query/analytics.py` (8+ occurrences)

Les valeurs `2` (WIN) et `3` (LOSS) sont dupliquées en dur dans toutes les requêtes SQL. L'enum `Outcome` existe mais n'est pas utilisé dans les f-strings SQL.

**Correction** : définir des constantes `_WIN = Outcome.WIN.value` et `_LOSS = Outcome.LOSS.value` en tête de fichier, les injecter dans les requêtes via paramètres ou f-string centralisé.

---

## Priorité suggérée

| # | Sévérité | Effort | Priorité |
|---|----------|--------|----------|
| 1 | Bug visible | Faible | ⭐ Sprint prochain |
| 5 | Bug potentiel | Très faible | ⭐ Sprint prochain |
| 4 | Dette | Très faible | Sprint suivant |
| 3 | Dead code | Très faible | Sprint suivant |
| 2 | Guard obsolète | Faible | Sprint suivant |
| 6 | Lisibilité | Moyen | Backlog |

---

# Audit Traductions / i18n — 2026-03-03

Analyse réalisée par scanning statique + exécution du registre i18n.

**Périmètre analysé** :
- `src/ui/i18n/` : `common.py`, `pages.py`, `widgets.py`, `viz.py`, `cli.py`, `data_labels.py`, `__init__.py`
- `src/ui/translations.py` (dicts hardcodés : `PAIR_FR`, `PLAYLIST_FR/EN`)
- `static/i18n/` : JSON playlists, modes, ranks, awards, citations

**Chiffres clés** : 1 228 clés, 134 aliases, 0 traduction FR ou EN manquante.

---

## 🔴 Bugs de données

### i18n-1. Clés tronquées dans `PAIR_FR` (typos)

**Fichier** : `src/ui/translations.py`

Deux entrées ont leurs premiers caractères coupés — elles ne matcheront jamais :

```python
# Clé tronquée — "Firefight:" est manquant au début
"ght:Heroic King of the Hill on Vallaheim Firefight": "Baptême du feu : ..."

# Clé tronquée — "S" manquant au début
"urvive The Undead 3.0 on TFF | Night Of The Undead": "Survivre aux morts-vivants 3.0"
```

**Correction** : restaurer les clés complètes :
- `"Firefight:Heroic King of the Hill on Vallaheim Firefight"`
- `"Survive The Undead 3.0 on TFF | Night Of The Undead"`

---

## 🟠 Redondances majeures

### i18n-2. `PAIR_FR` : 342 entrées inutiles sur 399 (86%)

**Fichier** : `src/ui/translations.py`

`PAIR_FR` contient 399 entrées dont **342 sont 100% redondantes** avec les entrées génériques déjà présentes dans le même dict.

Exemple concret :

```python
# Générique (utile)
"Arena:CTF": "Arène : Capture du drapeau",

# 25 entrées spécifiques identiques (inutiles — le fallback strip "on ..." les couvre déjà)
"Arena:CTF on Aquarius": "Arène : Capture du drapeau",
"Arena:CTF on Bazaar": "Arène : Capture du drapeau",
"Arena:CTF on Chasm": "Arène : Capture du drapeau",
# ... 22 autres identiques
```

La fonction `translate_pair_name()` fait déjà `split(" on ", 1)[0]` pour le fallback générique — donc ces 342 entrées ne seront **jamais atteintes** en pratique (le générique répond avant).

**Seules 11 entrées spécifiques** apportent une vraie valeur (cas sans générique correspondant) :  
`Arena:VIP on Catalyst`, `Ranked:CTF 3 Captures on Argyle`, les deux `Fiesta:Slayer on *-Forge`, etc.

**Correction** : supprimer les 342 entrées redondantes. Le dict passe de 399 → 57 entrées.  
Gain de lisibilité majeur + maintenance réduite.

---

### i18n-3. Doublon exact : `tm_session_trend` dans deux modules

**Fichiers** : `src/ui/i18n/pages.py` ET `src/ui/i18n/widgets.py`

La clé `tm_session_trend` est définie avec les mêmes valeurs dans les deux modules :
```python
{"fr": "Tendance de session", "en": "Session trend"}
```
Le `__init__.py` loggue ce genre de collision en WARNING. La version de `pages` prend la priorité (premier chargé).

**Correction** : supprimer l'entrée dans `widgets.py`.

---

## 🟡 Points d'attention (non-bugs)

### i18n-4. 115 clés avec FR == EN (traductions identiques)

115 clés ont une traduction EN strictement identique à FR (termes neutres ou jargon Halo invariable) :
`"Score"`, `"Date"`, `"Performance"`, `"Playlist"`, `"Ratio"`, `"Arena"`, `"BTB"`, noms de médailles, etc.

Ce n'est pas un bug — ce sont des termes valides qui ne traduisent pas. Mais cela gonfle inutilement le registre.

**Optimisation possible** : supporter une valeur `null`/absent pour EN signifiant "utiliser FR". Effort élevé vs bénéfice faible — **classer en backlog**.

---

### i18n-5. `PAIR_FR` : aucune couverture JSON (architecture à deux vitesses)

`translate_pair_name()` cherche d'abord dans `modes_{lang}.json`, puis dans `PAIR_FR`. Mais aucune des 399 entrées de `PAIR_FR` n'est couverte par les JSON actuels — le JSON est donc toujours bypassé pour les pairs.

Conséquence : les traductions EN des pairs passent par l'algorithme générique (`_EN_GENERIC_MODES` + `_EN_PREFIXES`) sans dict exhaustif, alors que le FR a le dict complet. **Asymétrie FR/EN pour les noms de pairs.**

**Direction à terme** : migrer les 57 entrées utiles de `PAIR_FR` (après nettoyage i18n-2) dans `static/i18n/modes_fr.json` pour unifier la source de vérité.

---

### i18n-6. Architecture : `translations.py` = couche legacy non migrée

`translations.py` est explicitement décrit comme "fallback historique" dans sa docstring. Il contient encore :
- `PLAYLIST_FR/EN` (~15 entrées + UUIDs)
- `PAIR_FR` (399 entrées)
- `_EN_GENERIC_MODES` (30 entrées)
- `_EN_PREFIXES` (14 entrées)

La migration vers les JSON `static/i18n/` est amorcée mais pas terminée. Les fonctions `translate_playlist_name()` et `translate_pair_name()` maintiennent les deux couches en parallèle.

---

## Résumé i18n

| # | Type | Impact | Effort | Action |
|---|------|--------|--------|--------|
| i18n-1 | Bug données | Moyen | Très faible | ⭐ Corriger maintenant |
| i18n-2 | Redondance | Faible (maintenance) | Faible | Sprint prochain |
| i18n-3 | Doublon clé | Faible | Très faible | Sprint prochain |
| i18n-4 | Bruit registre | Négligeable | Élevé | Backlog |
| i18n-5 | Asymétrie FR/EN | Moyen (UX EN) | Moyen | Backlog |
| i18n-6 | Dette migration | Moyen (maintenance) | Élevé | Backlog |
