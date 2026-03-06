## Plan : Logging avancé — famille match_view

### TL;DR

Les 7 fichiers `match_view_*` totalisent **38 blocs `except`** dont **34 silencieux** (89 %), **0 log structurel** (entry point, métriques, décisions de rendu), et 3 fichiers sans même `import logging`. Le plan ajoute du logging à **5 niveaux** : infrastructure manquante, exceptions silencieuses, points d'entrée métier, requêtes DB, et décisions de rendu conditionnel. Convention projet : `%s` lazy, messages en français, `warning` pour les erreurs récupérables, `debug` pour le reste, `exc_info=True` réservé aux erreurs rares. Helper `log_duration()` existant dans `src/utils/log_config.py` pour le benchmarking.

---

### Steps

#### 1. Infrastructure — ajouter `import logging` + `logger` aux 3 fichiers manquants

- `match_view.py` : ajouter `import logging` + `logger = logging.getLogger(__name__)` en haut (après les imports stdlib existants)
- `match_view_charts.py` : idem
- `match_view_participation.py` : idem

#### 2. Les 6 blocs `except` critiques — passer de silence total à `logger.warning` avec `exc_info=True`

| Fichier | Ligne | Action |
|---------|-------|--------|
| `match_view.py` | L546 | `pass` → `logger.warning("Lecture player_match_enrichment échouée match=%s", match_id, exc_info=True)` |
| `match_view_participation.py` | L59 | `return` → `logger.warning("render_participation_section: échec repo match=%s xuid=%s", match_id, xuid, exc_info=True)` + `return` |
| `match_view_participation.py` | L180 | `pass` → `logger.warning("render_participation_comparison: échec", exc_info=True)` |
| `match_view_players.py` | L532 | `pass` → `logger.warning("Rendu chart Killer-Victim échoué match=%s", match_id, exc_info=True)` |
| `match_view_players.py` | L675 | `he = None` → ajout `logger.warning("Chargement highlight_events échoué match=%s", match_id, exc_info=True)` |
| `match_view_charts.py` | L328 | `pass` → `logger.debug("Calcul perfect kills historique échoué", exc_info=True)` |

#### 3. Les 14 blocs `except` medium — passer à `logger.warning` ou `logger.debug`

| Fichier | Ligne | Niveau | Message |
|---------|-------|--------|---------|
| `match_view_players.py` | L58 | `debug` | `"_has_table_duckdb: échec vérification table=%s db=%s", table_name, db_path` |
| `match_view_players.py` | L71 | `warning` | `"Chargement stats joueurs échoué match=%s", match_id` |
| `match_view_players.py` | L84 | `warning` | `"Chargement scoreboard échoué match=%s", match_id` |
| `match_view_players.py` | L453 | `debug` | `"Résolution gamertags échouée match=%s", match_id` |
| `match_view_players.py` | L463 | `warning` | `"Chargement killer_victim_pairs échoué match=%s", match_id` |
| `match_view_players.py` | L486 | `debug` | `"Fallback compute_killer_victim_pairs échoué match=%s", match_id` |
| `match_view_charts.py` | L148 | `debug` | `"Conversion ratio K/D/A échouée", exc_info=True` |
| `match_view.py` | L87 | `debug` | `"Module skill_rating_config non disponible"` |
| `match_view.py` | L290 | `warning` | `"CitationEngine.aggregate_for_display échoué match=%s", match_id` |
| `match_view.py` | L301 | `debug` | `"Citations: chargement compteurs globaux échoué"` |
| `match_view_helpers.py` | L46 | `debug` | `"Conversion datetime échouée: %s", dt_value` |
| `match_view_helpers.py` | L136 | `warning` | `"Scan dossier médias échoué: %s", dir_path` |
| `match_view_helpers.py` | L131 | `debug` | `"os.stat échoué pour: %s", full` |
| `match_view_helpers.py` | L264 | `debug` | `"Conversion epoch timestamp échouée"` |

#### 4. Les 12 blocs `except` mineurs — ajouter `logger.debug` systématique

Tous les `except` restants (conversions numériques, fallbacks cosmétiques, debug flags) reçoivent un `logger.debug` bref avec contexte minimal. Pas de `exc_info=True` pour ceux-là (overhead inutile).

#### 5. Logs de point d'entrée — tracer les fonctions clés

Ajouter un `logger.debug` au début des **4 fonctions de rendu principales** pour tracer l'appel avec le `match_id` :

| Fonction | Fichier | Message |
|----------|---------|---------|
| `render_match_view` | `match_view.py` | `"render_match_view: match=%s xuid=%s db=%s", match_id, xuid, db_path` |
| `render_participation_section` | `match_view_participation.py` | `"render_participation_section: match=%s xuid=%s", match_id, xuid` |
| `render_encounter_section` | `match_view_encounters.py` | `"render_encounter_section: match=%s xuid=%s", match_id, self_xuid` |
| `_render_match_citations_section` | `match_view.py` | `"_render_match_citations_section: match=%s xuid=%s", match_id, xuid` |

#### 6. Logs de résultat DB — tracer les chargements significatifs

Ajouter un `logger.debug` **après** les appels DB/repository réussis pour tracer ce qui a été chargé :

| Point | Message |
|-------|---------|
| `match_view.py` L682-683 | Après chargement `pm` + `medals_last` : `"Match %s: pm=%s medals=%d", match_id, "ok" if pm else "vide", len(medals_last or [])` |
| `match_view.py` L536-545 | Après query `player_match_enrichment` : `"match=%s had_bot=%s perf_score=%s", match_id, _had_bot_teammate, _stored_perf_score` |
| `match_view_players.py` L461-463 | Après chargement KV pairs : `"Pairs KV chargées match=%s: %d lignes", match_id, len(pairs_df) if pairs_df is not None else 0` |
| `match_view_encounters.py` L285 | Après chargement scoreboard : `"Scoreboard rencontre match=%s: %d joueurs", match_id, len(players)` |

#### 7. Logs de décisions de rendu — tracer les `return` silencieux

Ajouter un `logger.debug` **avant chaque early return** qui rend une section invisible sans feedback utilisateur :

| Fichier | Ligne | Condition | Message |
|---------|-------|-----------|---------|
| `match_view_players.py` | L112 | table absente | `"render_team_dominance_section: skip (table absente) match=%s", match_id` |
| `match_view_players.py` | L145 | < 2 équipes | `"render_team_dominance_section: skip (<2 équipes) match=%s", match_id` |
| `match_view_players.py` | L446 | match_id vide | `"_render_antagonist_chart: skip (match_id vide)"` |
| `match_view_players.py` | L670 | table absente | `"render_kd_timeline_section: skip (table absente) match=%s", match_id` |
| `match_view_players.py` | L678 | events vides | `"render_kd_timeline_section: skip (events vides) match=%s", match_id` |
| `match_view_participation.py` | L51 | pas de PSA | `"render_participation_section: skip (pas de personal_score_awards) match=%s", match_id` |
| `match_view_participation.py` | L54 | df vide | `"render_participation_section: skip (df vide) match=%s", match_id` |
| `match_view_participation.py` | L166 | profiles vides | `"render_participation_comparison: skip (profiles vides)"` |
| `match_view_encounters.py` | L270 | IDs manquants | `"render_encounter_section: skip (match_id ou xuid manquant)"` |
| `match_view_encounters.py` | L291 | players vides | `"render_encounter_section: skip (players vides) match=%s", match_id` |
| `match_view_encounters.py` | L296 | pas de cibles | `"render_encounter_section: skip (0 adversaires non-amis) match=%s", match_id` |
| `match_view_encounters.py` | L299 | df encounters vide | `"render_encounter_section: skip (df encounters vide) match=%s", match_id` |

#### 8. Performance — `log_duration` sur l'enveloppe principale

Utiliser le context manager `log_duration()` existant pour envelopper `render_match_view` et le `st.spinner` de chargement de données :

```python
from src.utils.log_config import log_duration

with log_duration("render_match_view", logger, threshold_ms=200):
    ...
```

Seulement sur `render_match_view` et le bloc `with st.spinner` (L682-683) — pas besoin de mesurer chaque sous-section.

---

### Verification

1. **Syntaxe** : `python -m py_compile src/ui/pages/match_view.py` (et les 6 autres)
2. **Tests existants** : `python -m pytest tests/ -q --ignore=tests/integration` — aucune régression
3. **Vérification manuelle** : ouvrir un match dans l'app, vérifier dans `data/logs/app.log` que les logs `debug` apparaissent avec les `match_id` corrects
4. **Simulation erreur** : renommer temporairement `shared_matches.duckdb` → les `warning` doivent apparaître dans le log au lieu d'un silence total
5. **Ruff** : `python -m ruff check src/ui/pages/match_view*.py --no-fix`

### Decisions

- **`warning` (pas `error`)** pour les `except` critiques : cohérent avec la convention projet (seuls les crashs fatals utilisent `error`)
- **`debug` pour les early returns** : ce sont des situations normales (ex: pas de données PSA), pas des erreurs
- **`%s` lazy (pas f-strings)** : tendance récente du projet, évite l'évaluation inutile si le log level est désactivé
- **`exc_info=True` sélectif** : uniquement sur les 6 critiques + quelques medium où la stacktrace aide — pas sur les conversions numériques triviales
- **Messages en français** : cohérent avec le reste du projet
- **Pas de refactoring des `contextlib.suppress`** : les 3 occurrences (thumbnail, datetime, gamertag) restent en `suppress` avec ajout d'un `logger.debug` juste avant le `suppress` quand le contexte est utile
- **`log_duration` uniquement sur `render_match_view`** : évite le bruit ; une seule mesure suffit pour identifier si le rendu est lent

### Résumé quantitatif

| Catégorie | Ajouts |
|-----------|:------:|
| `import logging` + `logger` | **3 fichiers** |
| `except` → `logger.warning` | **~10** |
| `except` → `logger.debug` | **~22** |
| Entry point logs | **4** |
| Post-DB result logs | **4** |
| Early return logs | **12** |
| `log_duration` | **1** |
| **Total logs ajoutés** | **~56** |

---
---

## Plan combiné : Refactoring modularité UI + Logging match_view

> Fusionné le 4 mars 2026. Les deux plans ciblent les mêmes fichiers — chaque fichier n'est touché qu'**une seule fois**.

### Contexte — Audit modularité UI

**23 fichiers > 500L**, **30 fonctions > 80L**, **4 fonctions dupliquées 3× chacune**, **6 patterns `has_table` inline**. Le plan refactoring corrige les god files et centralise les utilitaires dupliqués.

### Synergie des deux plans

| Fichier | Refactoring prévu | Logging prévu |
|---------|:---:|:---:|
| `match_view.py` (916L) | Split `render_match_view` (488L), extraire `match_view_rank.py` | +import logger, 6 logs, log_duration |
| `match_view_players.py` (1164L) | Extraire `_data.py` + `_scoreboard.py` | +14 logs (except + early return) |
| `match_view_charts.py` | — | +import logger, 2 logs |
| `match_view_participation.py` | — | +import logger, 6 logs |
| `match_view_helpers.py` | (18 `st.*` calls = dette identifiée) | +4 logs |
| `match_view_encounters.py` | — | +5 logs |
| `career.py` (1126L) | Extraire `_data.py` + `_logic.py` | (hors scope logging) |
| `src/utils/db.py` | +`has_table()`, +`is_duckdb_v4_path()` | — |
| `cache_loaders.py`, `cache_social.py` | Supprimer copies `_is_duckdb_v4_path` | — |
| `teammates_helpers.py`, `win_loss.py` | Supprimer copies `_normalize_mode_label`, `_format_score_label` | — |
| `career_ranks.py`, `aliases.py`, `session_compare.py`, `media_library_data.py` | Remplacer `information_schema` inline par `has_table()` | — |

---

### Phase 1 — Fondations partagées (pré-requis)

Centraliser les utilitaires dupliqués **avant** de toucher les pages.

| # | Action | Fichiers impactés |
|---|--------|-------------------|
| 1.1 | Ajouter `has_table(conn, table_name) → bool` dans `src/utils/db.py` | `src/utils/db.py` |
| 1.2 | Déplacer `_is_duckdb_v4_path()` → `src/utils/db.py` ; remplacer les 3 copies par un import | `cache_loaders.py`, `cache_social.py`, `match_view_players.py` |
| 1.3 | Remplacer `_normalize_mode_label()` dans 2 fichiers par `from src.app.helpers import normalize_mode_label` | `teammates_helpers.py`, `win_loss.py` |
| 1.4 | Remplacer `_format_score_label()` dans 2 fichiers par `from src.ui.formatting import format_score_label` | `teammates_helpers.py`, `objective_analysis.py` |
| 1.5 | Remplacer les 6 patterns `information_schema.tables` inline par `has_table()` | `match_view_players.py`, `career_ranks.py`, `aliases.py`, `session_compare.py`, `media_library_data.py` (×2) |

**Vérification** : `python -m pytest -q --ignore=tests/integration`

---

### Phase 2 — Split `match_view_players.py` (1164L) + logging

Découper **et** instrumenter en un seul pass.

| # | Action | Résultat |
|---|--------|----------|
| 2.1 | Créer `match_view_players_data.py` : `_load_match_players_stats()`, `_load_match_scoreboard()` + imports DB | ~120L, utilise `has_table()` de phase 1 |
| 2.2 | Créer `match_view_scoreboard.py` : `_get_scoreboard_cols()`, `_sb_numeric_value()`, `_compute_scoreboard_extremes()`, `_sb_cell_class()`, `_fmt_scoreboard_cell()`, `render_match_scoreboard()` | ~350L |
| 2.3 | Réduire `match_view_players.py` résiduel : `render_team_dominance_section`, `render_nemesis_section`, `render_roster_section`, `render_match_impact_section`, `render_kd_timeline_section` | ~500L |
| 2.4 | **Logging** dans les 3 fichiers résultants | |

Logs ajoutés dans cette phase :

| Fichier résultat | Logs |
|---|---|
| `match_view_players_data.py` | `_has_table` debug, chargement stats warning, scoreboard warning |
| `match_view_scoreboard.py` | gamertags debug, KV pairs warning, fallback debug, highlight_events warning |
| `match_view_players.py` | chart KV warning, early returns debug (table absente, <2 équipes, match_id vide, events vides) |

---

### Phase 3 — Split `match_view.py` (916L) + logging complet

| # | Action | Résultat |
|---|--------|----------|
| 3.1 | Extraire `_build_match_rank_html()` (176L) → `match_view_rank.py` | ~210L |
| 3.2 | Découper `render_match_view()` (488L) en sous-fonctions : `_render_header()`, `_render_stats_tabs()`, `_render_medals_tab()`, `_render_enrichment_section()` | ~500L résiduel |
| 3.3 | `import logging` + `logger` | |
| 3.4 | Logging plan LOG steps 2-7 : entry points, except critiques (PME L546, citations L290, compteurs L301, skill_rating L87), post-DB (pm+medals, query PME), `log_duration` sur `render_match_view` | |

---

### Phase 4 — Logging des fichiers match_view non-splittés

| # | Fichier | Actions |
|---|---------|---------|
| 4.1 | `match_view_charts.py` | +`import logging` + logger, +2 logs (conversion ratio debug, perfect kills debug) |
| 4.2 | `match_view_participation.py` | +`import logging` + logger, +entry point, +2 except critiques (L59, L180), +3 early returns (PSA, df vide, profiles vides) |
| 4.3 | `match_view_helpers.py` | +4 logs debug (datetime L46, os.stat L131, scan warning L136, epoch L264) |
| 4.4 | `match_view_encounters.py` | +entry point, +4 early returns (IDs manquants, players vides, 0 cibles, df vide), +post-DB scoreboard |

---

### Phase 5 — Split `career.py` (1126L)

Hors scope logging, dans le plan modularité pur.

| # | Action | Résultat |
|---|--------|----------|
| 5.1 | Créer `career_data.py` : `_load_career_data()`, `_load_career_history()`, `_load_pre_sync_match_dates()`, `_load_post_sync_match_count()`, `_load_lusr_snapshot()`, `_load_lusr_history()`, `_load_other_players_histories()` | ~250L |
| 5.2 | Créer `career_logic.py` : `_compute_estimated_xp_curve()`, `_compute_active_xp_per_day()`, `_compute_hero_projections()`, `_create_xp_history_chart()`, `_get_pg_labels()` | ~300L |
| 5.3 | Réduire `career.py` résiduel : `render_career_page()`, `_render_lusr_section()` (refactorisée) | ~400L |

---

### Duplications éliminées — Détail

| Fonction | Copies actuelles | Action |
|----------|:---:|---|
| `_is_duckdb_v4_path()` | 3× (`cache_loaders.py`, `cache_social.py`, `match_view_players.py`) | → `src/utils/db.py` |
| `_normalize_mode_label()` | 3× (`app/helpers.py` canonical, `teammates_helpers.py`, `win_loss.py`) | → import canonical |
| `format_score_label()` / `_format_score_label()` | 3× (`formatting.py` canonical, `teammates_helpers.py`, `objective_analysis.py`) | → import canonical |
| `has_table` pattern (`information_schema.tables`) | 6× inline (`match_view_players.py`, `career_ranks.py`, `aliases.py`, `session_compare.py`, `media_library_data.py` ×2) | → `has_table()` dans `src/utils/db.py` |

---

### Résumé quantitatif combiné

| Métrique | Modularité | Logging | **Combiné** |
|----------|:---:|:---:|:---:|
| Fichiers créés | 5 | 0 | **5** |
| Fichiers modifiés | ~10 | 7 | **~12** (dédupliqués) |
| Fonctions centralisées | 4 | 0 | **4** |
| Copies supprimées | 12 | 0 | **12** |
| Logs ajoutés | 0 | ~56 | **~56** |
| God files éliminés | 3 | 0 | **3** |
| Lignes nettes | ~-200 (duplication) | ~+80 (logs) | **~-120 net** |

### Points de vigilance

- **Pas de double-touch** : chaque fichier est touché une seule fois dans une seule phase
- **Les imports des nouveaux modules** (`match_view_players_data`, `career_data`, etc.) doivent utiliser les helpers centralisés de phase 1
- **Les logs dans les fichiers splittés** utilisent `logger = logging.getLogger(__name__)` — le `__name__` changera avec le nouveau module
- **`match_view_helpers.py`** a 18 appels `st.*` : dette identifiée mais pas traitée ici (renommage en `match_view_media_render.py` à planifier séparément)
- **`safe_chart_render` inconsistant** (53/73 appels plotly sans) : pas traité ici, à uniformiser dans un pass dédié

### Vérification globale

1. `python -m py_compile` sur chaque fichier créé/modifié
2. `python -m pytest -q --ignore=tests/integration` — zéro régression
3. `python -m ruff check src/ui/pages/match_view*.py src/ui/pages/career*.py --no-fix`
4. Vérification manuelle : ouvrir un match dans l'app → logs debug avec match_id dans `data/logs/app.log`
5. `wc -l` sur chaque fichier résultant → tous sous 500L
