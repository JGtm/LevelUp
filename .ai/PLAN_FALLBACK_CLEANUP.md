# Plan de stabilisation — Suppression des fallbacks excessifs

**Branche cible :** `refactor/id-resolution-cleanup` (existante)
**Stratégie Git :** 1 branche confirmée, N commits séquentiels
**Date :** 2026-03-17

---

## Contexte

Analyse complète du code `src/` révèle quatre familles de fallbacks dangereux :

1. **Guards sur vues v6 garanties** — Le copilot-instructions.md interdit explicitement les `_has_shared_view` / `_has_shared_table` sur les vues/tables v6. Ces guards existent encore dans plusieurs mixins.
2. **Chemins v4/v3 morts** — Les branches qui ciblent des tables locales supprimées en v5.1 (`match_stats`, `match_participants`, `medals_earned` locales) sont du dead code qui complexifie la lecture et peut masquer des erreurs.
3. **Dead code SQLite dans `ui/multiplayer.py`** — ~370 lignes jamais exécutées depuis v5.
4. **`except Exception: pass` sans log métier** — Absorbe silencieusement des erreurs dans des fonctions de calcul critiques (moteur citations, chargement stats).

---

## Vue d'ensemble des phases

| Phase | Scope | Commits | Risque |
|-------|-------|---------|--------|
| **Ph-1** | Guards vues v6 + méthodes fallback gamertag/KV | 2 commits | Faible |
| **Ph-2** | Branche v4/v3 dans `_get_match_source()` | 1 commit | Moyen |
| **Ph-3** | Fallbacks tables locales citations + legacy_compat | 2 commits | Faible |
| **Ph-4** | Dead code SQLite `multiplayer.py` | 1 commit | Faible |
| **Ph-5** | `getattr(settings,...)` → accès direct Pydantic | 1 commit | Très faible |
| **Ph-6** | Logging sur `except Exception` métier | 1 commit | Très faible |

---

## Phase 1 — Suppression guards vues v6 garanties

### Fichiers concernés

- `src/data/repositories/_gamertag_resolver.py`
- `src/data/repositories/_killer_victim_repo.py`
- `src/data/repositories/_career_encounters_repo.py`

### 1.A — `_gamertag_resolver.py`

#### Situation actuelle

```python
# L72 — appel à has_shared_view au lieu de supposer garanti
if self._has_shared_view("v_gamertag_lookup"):
    # chemin normal v6
else:
    # Fallback pré-vue : shared.xuid_aliases puis shared.match_participants
    resolved = self._resolve_gamertag_without_view(conn, xuid, match_id)
    if resolved:
        return resolved
```

La méthode `_resolve_gamertag_without_view()` (L97+) est entièrement du dead code en v6.

#### Changement

- Supprimer le `if/else` — conserver uniquement le chemin `v_gamertag_lookup`.
- Supprimer la méthode `_resolve_gamertag_without_view()`.
- Dans `get_all_gamertags()` (L216) : supprimer la guard `_has_shared_table("v_gamertag_lookup")` — c'est une vue, non une table, et elle est garantie.

#### Après

```python
def resolve_gamertag(self, xuid: str | None, *, match_id: str | None = None) -> str | None:
    if not xuid:
        return None
    conn = self._get_connection()
    xuid = str(xuid).strip()
    try:
        result = conn.execute(
            "SELECT gamertag FROM shared.v_gamertag_lookup WHERE xuid = ?",
            [xuid],
        ).fetchone()
        if result and result[0]:
            cleaned = _clean_gamertag_static(result[0])
            if cleaned:
                return cleaned
    except Exception:
        logger.warning("resolve_gamertag(%s): échec v_gamertag_lookup", xuid, exc_info=True)
    logger.warning("resolve_gamertag(%s): aucune source", xuid)
    return None
```

> **Note logging :** L'`except` passe de silencieux à `WARNING` avec `exc_info=True` — c'est une erreur inattendue en v6 (la vue est garantie).

### 1.B — `_killer_victim_repo.py`

#### Situation actuelle

```python
# L66
if self._has_shared_view("v_killer_victim_full"):
    table_ref = "shared.v_killer_victim_full"
elif self._has_shared_table("killer_victim_pairs"):
    table_ref = "shared.killer_victim_pairs"
# (même pattern L162, L168)
```

#### Changement

- Remplacer par `table_ref = "shared.v_killer_victim_full"` direct.
- Supprimer les `elif` et les `has_killer_victim_pairs()` fallback sur table locale.

#### Après (load_killer_victim_pairs_as_polars)

```python
table_ref = "shared.v_killer_victim_full"
```

```python
def has_killer_victim_pairs(self) -> bool:
    conn = self._get_connection()
    try:
        row = conn.execute("SELECT 1 FROM shared.v_killer_victim_full LIMIT 1").fetchone()
        return row is not None
    except Exception:
        logger.warning("has_killer_victim_pairs: erreur v_killer_victim_full", exc_info=True)
        return False
```

### 1.C — `_career_encounters_repo.py`

#### Situation actuelle

```python
def _get_kv_source_shared(self) -> str:
    if self._has_shared_table("v_killer_victim_full"):  # BUG : c'est une vue
        return "shared.v_killer_victim_full"
    return "shared.killer_victim_pairs"  # fallback mort
```

#### Changement

- Supprimer la méthode `_get_kv_source_shared()`.
- Remplacer ses usages par la constante `"shared.v_killer_victim_full"`.

---

### Tests Phase 1

**Fichier :** `tests/test_gamertag_resolver.py` (existant — à enrichir)

#### Nouveaux cas à ajouter

```
TestResolveGamertag_V6Guaranteed
├── test_resolve_uses_only_v_gamertag_lookup
│     Vérifie que resolve_gamertag() émet une requête vers v_gamertag_lookup
│     et retourne le bon gamertag.
│     Fixture : shared in-memory avec v_gamertag_lookup créée comme vue.
├── test_resolve_without_view_method_removed
│     Affirme que _resolve_gamertag_without_view n'existe plus sur le mixin.
├── test_resolve_logs_warning_on_db_error (NEW)
│     Simule une erreur DuckDB (DROP VIEW v_gamertag_lookup) → vérifie qu'un
│     WARNING est loggé avec exc_info (caplog pytest).
└── test_get_all_gamertags_no_has_shared_table_guard (NEW)
      Vérifie que get_all_gamertags() fonctionne sans guard préliminaire.
      Fixture : shared vide → retourne [] sans exception.
```

**Fichier :** `tests/test_killer_victim_antagonists.py` (existant — à enrichir)

```
TestKillerVictimRepo_V6
├── test_load_kv_uses_view_directly (NEW)
│     Vérifie que load_killer_victim_pairs_as_polars() ne cherche pas
│     killer_victim_pairs en fallback.
│     Fixture : shared avec UNIQUEMENT v_killer_victim_full (pas la table).
├── test_has_kv_pairs_true_when_view_has_data (NEW)
│     has_killer_victim_pairs() → True si la vue a des lignes.
├── test_has_kv_pairs_false_when_view_empty (NEW)
│     has_killer_victim_pairs() → False si la vue est vide.
└── test_career_encounters_kv_source_constant (NEW)
      Vérifie que load_top_encountered() utilise v_killer_victim_full
      sans appel à _has_shared_table.
```

**Pattern de fixture (à réutiliser) :**

```python
def _make_shared_v6(tmp_path: Path) -> Path:
    """shared_matches.duckdb minimal v6 avec toutes les vues garanties."""
    db_path = tmp_path / "shared_matches.duckdb"
    conn = duckdb.connect(str(db_path))
    conn.execute("CREATE TABLE match_registry (...)")
    conn.execute("CREATE TABLE match_participants (...)")
    conn.execute("CREATE TABLE xuid_aliases (...)")
    conn.execute("CREATE TABLE killer_victim_pairs (...)")
    # Vues v6 garanties
    conn.execute("CREATE VIEW v_gamertag_lookup AS ...")
    conn.execute("CREATE VIEW v_killer_victim_full AS ...")
    conn.execute("CREATE VIEW v_match_full AS ...")
    conn.execute("CREATE VIEW v_weapon_kills AS ...")
    conn.close()
    return db_path
```

Cette fixture sera centralisée dans `tests/conftest.py` sous le nom `tmp_shared_v6`.

---

## Phase 2 — Suppression branche v4/v3 dans `_get_match_source()`

### Fichier concerné

- `src/data/repositories/_match_queries.py`

### Situation actuelle

`_get_match_source()` contient trois branches :
1. XUID vide → fallback table locale `match_stats` (v3/v4)
2. shared absent → fallback table locale `match_stats` (v4)
3. `mv_player_matches` présente → chemin normal v5.1+

La méthode `_get_match_table_name()` scanne `information_schema` à chaque appel.

### Changement

Supprimer les branches 1 et 2. En v6 :
- Un XUID vide = une erreur de configuration, pas un chemin silencieux.
- shared absent = une erreur de déploiement, pas un fallback.

```python
def _get_match_source(self, conn) -> tuple[str, list[str], bool]:
    """Retourne l'expression FROM pour les matchs (v6 : mv_player_matches uniquement)."""
    if not self._xuid or self._xuid.strip() == "":
        raise RuntimeError(
            "XUID non configuré — impossible de charger les matchs. "
            "Vérifiez la configuration du joueur."
        )
    if not self.has_shared:
        raise RuntimeError(
            "shared_matches.duckdb indisponible. "
            "Fermez les scripts en cours puis relancez l'app."
        )
    source = """(SELECT
        match_id, start_time, map_id, map_name,
        playlist_id, playlist_name, pair_id, pair_name,
        game_variant_id, game_variant_name, outcome, team_id,
        kda, max_killing_spree, headshot_kills, avg_life_seconds,
        time_played_seconds, kills, deaths, assists, accuracy,
        my_team_score, enemy_team_score,
        team_mmr, enemy_mmr,
        personal_score, is_firefight, is_ranked
    FROM shared.mv_player_matches
    WHERE xuid = ?
    ) AS match_stats"""
    return source, [self._xuid], True
```

Supprimer également `_get_match_table_name()`.

> **Impact load_matches() :** Le `try/except` avec fallback SQL sans jointures (L~570) devient redondant — le simplifier en propagant l'exception directement. La requête ne devrait jamais échouer en v6 sauf bug réel.

### Tests Phase 2

**Fichier :** `tests/test_v5_match_queries.py` (existant — à enrichir)

```
TestMatchSource_V6
├── test_load_matches_from_mv_player_matches (EXISTANT — inchangé)
├── test_no_fallback_to_local_match_stats (NEW)
│     shared_db présente, DB joueur SANS match_stats → load_matches() fonctionne.
│     Vérifie qu'aucune requête sur match_stats locale n'est émise.
├── test_empty_xuid_raises_runtime_error (NEW)
│     DuckDBRepository avec xuid="" → _get_match_source() lève RuntimeError.
│     Message doit contenir "XUID non configuré".
├── test_no_shared_raises_runtime_error (NEW)
│     shared DB absente (shared_db_path=Path("/nonexistent")) →
│     _get_match_source() lève RuntimeError.
│     Message doit contenir "shared_matches.duckdb indisponible".
└── test_get_match_table_name_removed (NEW)
      Vérifie que _get_match_table_name n'existe plus sur le mixin.
```

**Fichier :** `tests/test_duckdb_repository_v5.py` (existant — à enrichir)

```
TestGetMatchCount_V6
├── test_get_match_count_uses_shared_only (NEW)
│     Vérifie que get_match_count() retourne 0 si shared vide,
│     sans accès à une table locale.
└── test_get_match_count_shared_unavailable_returns_zero (EXISTANT — comportement inchangé)
```

---

## Phase 3 — Suppression fallbacks tables locales citations + legacy_compat

### 3.A — `citations/_data_loader.py`

#### Situation actuelle

Chaque méthode `load_match_*` a une double logique :
- Chemin shared (nominal)
- Fallback sur table locale (dead code depuis v5.1)

Les tables locales `medals_earned`, `match_stats`, `match_participants` sont dans la liste des **8 tables supprimées en v5.1**.

#### Changement

Dans `load_match_medals()`, `load_match_stats()`, `load_match_df()`, `load_match_highlight_events()` :

- Supprimer les blocs `else:` / blocs `if tables:` qui ciblent les tables locales.
- Garder uniquement le chemin `shared_alias`.
- Si `shared_alias` est None ou que `_shared_has_table` retourne False → retourner directement `{}` / `[]` / `pl.DataFrame()` avec un log `DEBUG`.

> **Note :** `_shared_has_table` dans ce contexte est implémenté **localement** dans `CitationEngine` (pas via `SchemaIntrospectionMixin`). Ce mixin peut également être simplifié : en v6, toutes les tables sont garanties, donc `_shared_has_table` peut être remplacé par un accès direct (voir Ph-3 bonus).

**Bonus Ph-3 :** Supprimer `_shared_has_table()` de `CitationEngine` et appel direct aux tables — ou conserver comme vérification défensive avec log `WARNING` si absent (acceptable).

### 3.B — `_legacy_compat.py` — `_collect_xuids_local()`

#### Situation actuelle

`_collect_xuids_local()` itère sur `highlight_events`, `match_participants`, `antagonists` en tables locales — toutes supprimées en v5.1.

#### Changement

- Supprimer `_collect_xuids_local()`.
- Dans `list_other_player_xuids()` : supprimer l'appel et simplifier à `_collect_xuids_shared()` uniquement.

```python
def list_other_player_xuids(self, limit: int = 500) -> list[str]:
    conn = self._get_connection()
    xuids: set[str] = set()
    try:
        self._collect_xuids_shared(conn, xuids, limit)
        return list(xuids)[:limit]
    except Exception as e:
        logger.warning("Erreur list_other_player_xuids: %s", e, exc_info=True)
        return []
```

### Tests Phase 3

**Fichier :** `tests/test_citation_engine.py` (existant — à enrichir)

```
TestCitationDataLoader_V6
├── test_load_match_medals_from_shared (EXISTANT — à valider toujours OK)
├── test_load_match_medals_no_local_fallback (NEW)
│     shared_alias présent, medals_earned dans shared → OK.
│     Vérifier qu'aucune requête vers une table locale medals_earned n'est émise.
│     (mock conn + assertion execute calls)
├── test_load_match_stats_no_local_fallback (NEW)
│     Même pattern pour load_match_stats().
├── test_load_match_df_no_local_fallback (NEW)
│     Même pattern pour load_match_df().
└── test_load_highlight_events_empty_when_no_shared (NEW)
      shared_alias=None → retourne [] immédiatement.
```

**Fichier :** `tests/test_duckdb_repository.py` (existant — à enrichir)

```
TestListOtherPlayerXuids_V6
├── test_list_other_xuids_from_shared_only (NEW)
│     shared avec match_participants peuplé → liste correcte.
│     DB joueur sans tables locales → pas d'exception.
├── test_collect_xuids_local_method_removed (NEW)
│     Vérifie que _collect_xuids_local n'existe plus sur LegacyCompatMixin.
└── test_list_other_xuids_shared_unavailable_returns_empty (EXISTANT — vérifier)
```

---

## Phase 4 — Dead code SQLite `ui/multiplayer.py`

### Situation actuelle

`src/ui/multiplayer.py` contient ~370 lignes dont :
- `PlayerInfo` dataclass (utilisée uniquement par le code Legacy SQLite)
- Toutes les fonctions de scan SQLite (`_list_players_sqlite`, etc.)
- `DuckDBPlayerInfo` + scan des dossiers joueurs (potentiellement utile ?)
- `is_multi_player_db()` → `return False` hardcodé
- `render_player_selector()` → `return None` hardcodé

#### Analyse avant suppression

Avant de supprimer, vérifier **quels symboles sont importés ailleurs** :

```bash
grep -r "from src.ui.multiplayer\|import multiplayer" src/ tests/ --include="*.py"
```

Conserver uniquement ce qui est importé et utilisé activement.

#### Changement probable

- Supprimer `PlayerInfo`, `is_multi_player_db()`, `render_player_selector()`, fonctions SQLite.
- Conserver `DuckDBPlayerInfo` si utilisé dans la sélection de joueur (à vérifier).
- Réduire le module à <100 lignes avec uniquement le code v5 actif.

### Tests Phase 4

**Fichier :** `tests/test_multiplayer.py` (existant — adapter)

```
TestMultiplayer_V5Only
├── test_is_multi_player_db_removed_or_always_false (ADAPTER)
│     Si la fonction est supprimée : vérifier AttributeError.
│     Si conservée (par précaution) : vérifier return False.
├── test_player_info_legacy_class_removed (NEW)
│     Vérifie que PlayerInfo n'existe plus dans le module.
└── test_duckdb_player_info_still_exists_if_used (NEW)
      Vérifie que DuckDBPlayerInfo est encore présent (si conservé).
```

---

## Phase 5 — `getattr(settings, ...)` → accès direct Pydantic

### Fichiers concernés

- `src/app/sidebar.py` (L222-L253, ~15 occurrences)
- `src/app/main_helpers.py` (L112-L207, ~10 occurrences)
- `src/app/profile.py` (L203-L245, ~10 occurrences)

### Situation actuelle

```python
bool(getattr(settings, "spnkr_refresh_backfill_medals", False))
int(getattr(settings, "profile_assets_auto_refresh_hours", 0) or 0)
```

Si `AppSettings` est un `BaseModel` Pydantic v2, les champs ont déjà des valeurs par défaut déclarées. `getattr(..., False)` masque :
- Un champ potentiellement absent du modèle (bug silencieux)
- Une faute de frappe dans le nom du champ

#### Pré-requis

Avant de changer, vérifier que tous les attributs existent bien dans `AppSettings` :

```bash
grep -n "spnkr_refresh_backfill_\|profile_api_\|profile_assets_\|media_enabled\|prefer_spnkr" src/config.py
```

#### Changement

Remplacer `getattr(settings, "field", default)` par `settings.field`.

Si un champ n'existe pas dans `AppSettings`, l'**ajouter** avec sa valeur par défaut dans `AppSettings` — c'est la stabilisation en amont.

```python
# Avant
bool(getattr(settings, "spnkr_refresh_backfill_medals", False))
# Après
settings.spnkr_refresh_backfill_medals
```

### Tests Phase 5

**Fichier :** `tests/test_settings_backfill.py` (existant — à enrichir)

```
TestAppSettingsFields
├── test_all_sidebar_settings_fields_declared (NEW)
│     Instancie AppSettings() avec valeurs par défaut et vérifie que tous les
│     champs accédés dans sidebar.py existent (hasattr check + pas de AttributeError).
│     Liste des champs couverts documentée dans le test.
├── test_all_profile_settings_fields_declared (NEW)
│     Même chose pour main_helpers.py et profile.py.
└── test_settings_no_getattr_access (NEW — qualité de code)
      ast.parse sur sidebar.py, main_helpers.py, profile.py → assert aucun
      nœud ast.Call avec func=getattr et args[0]=settings.
      (Test de régression pour empêcher la réintroduction de getattr(settings...))
```

---

## Phase 6 — Logging sur `except Exception: pass` métier

### Objectif

Les `except Exception: pass` / `except Exception: return X` dans des fonctions de **calcul métier** (pas I/O externe) doivent au minimum émettre un `logger.debug(...)` avec `exc_info=True`. Les plus critiques méritent `WARNING`.

### Matrice de décision

| Niveau | Contexte | Nouveau comportement |
|--------|----------|----------------------|
| `WARNING` + `exc_info=True` | Erreur inattendue en opération normale (resolve_gamertag, load_match_medals) | Anomalie à investiguer |
| `DEBUG` + `exc_info=True` | Opération optionnelle (enrichissement, cache) | Informatif uniquement |
| Inchangé | I/O externe réseau/fichier (discord, tailscale, media_helpers) | Attendu, pas de bruit |

### Changements fichier par fichier

**`src/analysis/citations/engine.py` — PRIORITY HIGH**

```python
# L142 — dans la boucle d'évaluation des règles
except Exception:
    continue
# → Ajouter :
except Exception:
    logger.debug(
        "Règle citation ignorée (exception) pour match_id=%s rule=%s",
        match_id, rule_name, exc_info=True
    )
    continue

# L145 — retour None sur calcul citation
except Exception:
    return None
# → Ajouter :
except Exception:
    logger.warning(
        "Erreur calcul citation (match_id=%s) — retourne None",
        match_id, exc_info=True
    )
    return None

# L156 — retour False sur évaluation
except Exception:
    return False
# → Ajouter :
except Exception:
    logger.warning(
        "Erreur évaluation citation (match_id=%s) — retourne False",
        match_id, exc_info=True
    )
    return False

# L301 — retour 0 sur compteur
except Exception:
    return 0
# → Ajouter :
except Exception:
    logger.debug("Erreur compteur citation — retourne 0", exc_info=True)
    return 0
```

**`src/app/data_loader.py`** — L67, L133, L198 (init session state)
→ `logger.debug("...", exc_info=True)` — informatif, pas bloquant.

**`src/app/state.py`** — L283, L293
→ `logger.debug("...", exc_info=True)`.

**`src/app/_filters_friends.py`** — L133, L202
→ `logger.debug("...", exc_info=True)`.

### Tests Phase 6

**Fichier :** `tests/test_citation_engine.py` (existant — à enrichir)

```python
# Utiliser caplog de pytest pour vérifier les logs

TestCitationEngineLogging
├── test_rule_exception_logs_debug (NEW)
│     Injecter une règle qui lève une exception.
│     Vérifier que caplog contient un message DEBUG avec le nom de la règle.
│     Vérifier que le calcul continue (les autres règles sont évaluées).
├── test_compute_failure_logs_warning (NEW)
│     Forcer une erreur dans le calcul global d'une citation.
│     Vérifier caplog niveau WARNING avec exc_info.
└── test_evaluation_failure_logs_warning (NEW)
      Forcer une erreur dans l'évaluation booléenne.
      Vérifier caplog niveau WARNING.
```

**Exemple de test caplog :**

```python
import pytest

def test_rule_exception_logs_debug(caplog):
    with caplog.at_level(logging.DEBUG, logger="src.analysis.citations.engine"):
        engine = CitationEngine(...)
        # Injecter une règle cassée
        engine._rules["test_rule"] = lambda **kw: 1 / 0
        result = engine.compute(...)
    assert any("Règle citation ignorée" in r.message for r in caplog.records)
    assert any(r.levelname == "DEBUG" for r in caplog.records)
```

---

## Phase bonus — `citations/custom_rules.py` : legacy award keys

### Situation actuelle

```python
zone_captures = (
    awards.get("zone_captured", 0)
    or awards.get("Zone capturée", 0)   # Legacy fallback (données pré-migration)
    or awards.get("Zone Capture", 0)
)
```

### Question à trancher avant d'agir

Sont présentes dans la DB des entrées avec les clés françaises / en majuscules ?

```sql
-- Requête de diagnostic (à lancer une fois)
SELECT DISTINCT award_name FROM personal_score_awards
WHERE award_name IN ('Zone capturée', 'Zone Capture', 'Porteur arrêté',
                     'Flag Carrier Kill', 'Flag Carrier Killed');
```

**Si résultat vide** (probable) → supprimer les fallbacks.
**Si résultat non vide** → ajouter une migration de normalisation, puis supprimer.

### Changement après confirmation

```python
zone_captures = awards.get("zone_captured", 0)
# Les clés legacy ('Zone capturée', etc.) ont été supprimées en v5.1
# Voir migration add_migration_technical_ids
```

### Tests

**Fichier :** `tests/test_custom_citations.py` (existant)

```
TestCustomCitations_V6Keys
├── test_annexion_forcee_uses_technical_id_only (NEW)
│     Passer awards={"zone_captured": 9} → résultat = 3.
│     Passer awards={"Zone capturée": 9} → résultat = 0 (clé inconnue ignorée).
└── test_flag_em_down_uses_technical_id_only (NEW)
      Passer awards={"runner_stopped": 2} → résultat = 2.
      Passer awards={"Porteur arrêté": 2} → résultat = 0.
```

---

## Considérations transversales sur le logging

### Configuration logger recommandée

Tous les modules modifiés utilisent `logger = logging.getLogger(__name__)`. Pas de changement nécessaire sur la configuration.

### Niveaux à utiliser

```
DEBUG   — Opération optionnelle échouée (enrichissement, cache, résolution secondaire)
INFO    — (réservé aux actions utilisateur, pas utilisé ici)
WARNING — Erreur inattendue dans un chemin normalement fiable en v6
ERROR   — (réservé aux pannes critiques bloquantes; hors scope de ce refactoring)
```

### Pattern standard pour les `except` avec log

```python
# Pattern recommandé — à utiliser systématiquement
except Exception:
    logger.warning("Contexte : description courte (param=%s)", valeur, exc_info=True)
    return valeur_par_defaut
```

L'argument `exc_info=True` est **obligatoire** sur tous les `except Exception` nouvellement loggés — il permet de voir la traceback dans les logs de développement.

---

## Plan d'exécution et commits

```
Commit 1 : refactor(resolver): supprimer guards v_gamertag_lookup + _resolve_without_view
  Fichiers : _gamertag_resolver.py
  Tests : test_gamertag_resolver.py (+ nouveaux cas)

Commit 2 : refactor(kv-repo): supprimer guards v_killer_victim_full + _get_kv_source_shared
  Fichiers : _killer_victim_repo.py, _career_encounters_repo.py
  Tests : test_killer_victim_antagonists.py (+ nouveaux cas)

Commit 3 : refactor(match-queries): supprimer branches v4/v3 + _get_match_table_name
  Fichiers : _match_queries.py
  Tests : test_v5_match_queries.py (+ nouveaux cas)

Commit 4 : refactor(citations): supprimer fallbacks tables locales dans data_loader
  Fichiers : citations/_data_loader.py
  Tests : test_citation_engine.py (+ nouveaux cas)

Commit 5 : refactor(legacy-compat): supprimer _collect_xuids_local
  Fichiers : _legacy_compat.py
  Tests : test_duckdb_repository.py (+ nouveaux cas)

Commit 6 : refactor(multiplayer): supprimer dead code SQLite legacy
  Fichiers : ui/multiplayer.py
  Tests : test_multiplayer.py (adapter)

Commit 7 : refactor(settings): remplacer getattr(settings,...) par accès direct Pydantic
  Fichiers : sidebar.py, main_helpers.py, profile.py
  Tests : test_settings_backfill.py (+ nouveaux cas)

Commit 8 : fix(logging): ajouter logs sur except Exception métier
  Fichiers : citations/engine.py, data_loader.py, state.py, _filters_friends.py,
             _gamertag_resolver.py (ajusté depuis commit 1)
  Tests : test_citation_engine.py (+ cas caplog)

Commit 9 (conditionnel) : refactor(citations): supprimer legacy award keys après diagnostic SQL
  Fichiers : citations/custom_rules.py
  Tests : test_custom_citations.py (+ nouveaux cas)
```

---

## Checklist avant PR

- [ ] Tests passent : `python -m pytest --ignore=tests/integration -q`
- [ ] Aucun nouveau fichier dépasse 500 lignes
- [ ] Aucune nouvelle fonction dépasse 80 lignes
- [ ] Aucun `import pandas` introduit
- [ ] Aucun `getattr(settings, ...)` réintroduit
- [ ] Tous les nouveaux `except Exception` loggent avec `exc_info=True`
- [ ] `_resolve_gamertag_without_view` absent du codebase
- [ ] `_get_match_table_name` absent du codebase
- [ ] `_collect_xuids_local` absent du codebase
- [ ] thought_log.md mis à jour avec les décisions de chaque commit
