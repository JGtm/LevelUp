# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-13.

---

## 🔴 Dette Technique (code source)

### Migration : noms d'assets résolus → IDs bruts en BDD
> Dans `match_registry`, les noms d'assets sont stockés en parallèle des IDs bruts (redondance + risque de stale data). À terme, l'UI doit résoudre les noms à la lecture depuis `metadata.duckdb`, pas les lire depuis les colonnes `*_name`.

**Contexte** : Au moment de l'insertion (sync initial), les noms publics (ex. `"Aquarius"`, `"Ranked Arena"`) sont récupérés depuis l'API SPNKr et stockés directement en BDD — en plus de l'ID brut. La `weapon_kills` (v5.7) et `medals_earned` montrent le bon modèle : ID brut uniquement, résolution à la lecture.

**Colonnes concernées dans `shared_matches.duckdb`** :

| Table | Colonnes ID (OK) | Colonnes nom résolu (à migrer) |
|-------|-----------------|-------------------------------|
| `match_registry` | `map_id`, `playlist_id`, `pair_id`, `game_variant_id` | `map_name`, `playlist_name`, `pair_name`, `game_variant_name` |
| `match_participants` | `xuid` | `gamertag` (redondant avec `xuid_aliases`) |
| `highlight_events` | `xuid` | `gamertag` (peut devenir stale si alias change) |

**Modèles de référence (déjà corrects)** :
- `medals_earned.medal_name_id` → UBIGINT, résolution via `metadata.duckdb`
- `weapon_kills.weapon_id` → UBIGINT post v5.7 (migré depuis `weapon_name`)

**Actions** :
- [ ] Auditer les usages UI/query des colonnes `*_name` dans `match_registry` pour identifier ce qui lit directement le nom stocké vs ce qui joint `metadata.duckdb`
- [ ] Créer une vue `v_match_registry` qui résout les noms à la lecture via JOIN sur les tables de référence `metadata.duckdb` (maps, playlists, game_variants)
- [ ] Migrer les requêtes consommatrices (pages Streamlit, repositories) vers la vue — supprimer les colonnes `*_name` de `match_registry` une fois toutes les requêtes migrées
- [ ] Auditer les usages UI/query des colonnes `*_name` dans `match_registry` pour identifier ce qui lit directement le nom stocké vs ce qui joint `metadata.duckdb`
- [ ] Créer une vue `v_match_registry` qui résout les noms à la lecture via JOIN sur les tables de référence `metadata.duckdb` (maps, playlists, game_variants)
- [ ] Migrer les requêtes consommatrices (pages Streamlit, repositories) vers la vue — supprimer les colonnes `*_name` de `match_registry` une fois toutes les requêtes migrées
- [ ] `match_participants.gamertag` et `highlight_events.gamertag` : voir item dédié ci-dessous
- [ ] Ajouter un test de non-régression : aucune colonne `*_name` dans les nouvelles tables shared (hors `xuid_aliases`)

**Complexité** : L (impact UI + repositories + migrations)  
**Fichiers clés** : [`src/data/sync/migrations.py`](../src/data/sync/migrations.py), [`src/data/sync/_shared_writes.py`](../src/data/sync/_shared_writes.py), [`src/data/sync/transformers/_match.py`](../src/data/sync/transformers/_match.py), `data/warehouse/shared_matches.duckdb`

---

### Cohérence XUID↔Gamertag — source de vérité et stale data

> **Diagnostic complet (2026-03-13)** : trois emplacements stockent la relation XUID→Gamertag dans `shared_matches.duckdb`, avec des rôles et qualités différentes, et une cascade de résolution dont l'ordre est inversé par rapport à la logique attendue.

#### État actuel des 3 sources

| Table | Rôle réel | Qualité | Stale possible |
|-------|-----------|---------|:--------------:|
| `xuid_aliases.gamertag` | Gamertag **courant** — source de vérité, UPSERT à chaque sync | ✅ Normalisé | Non |
| `match_participants.gamertag` | Snapshot figé à la date du sync du match | ✅ Normalisé | **Oui** |
| `highlight_events.gamertag` | Champ brut de l'API events, sans normalisation complète | ⚠️ NUL bytes connus | **Oui** |

#### Problème identifié : cascade inversée dans `_gamertag_resolver.py`

La fonction `load_match_player_gamertags()` ([`src/data/repositories/_gamertag_resolver.py`](../src/data/repositories/_gamertag_resolver.py)) utilise cet ordre de priorité :

1. `shared.match_participants` ← **prioritaire** (snapshot figé)
2. local `match_participants` (fallback v4 legacy)
3. `shared.highlight_events` (données corrompues)
4. `shared.xuid_aliases` ← source de vérité… en **dernier**

Conséquence : si un joueur change de gamertag, le scoreboard affiche l'ancien nom alors que la page Rencontres (qui fait `COALESCE(xuid_aliases, match_participants)`) affiche le bon — incohérence visuelle entre pages pour le même joueur.

#### Plan d'implémentation

**Étape 1 — Fix immédiat, sans migration (priorité haute)**

- [ ] Dans `_gamertag_resolver.py`, inverser l'ordre de la cascade dans `load_match_player_gamertags()` :
  ```
  Nouvel ordre : xuid_aliases → match_participants → highlight_events
  ```
- [ ] Vérifier que `resolve_gamertag()` (source #1 = `match_participants`) applique le même correctif
- [ ] Valider que la page Rencontres reste cohérente (son `COALESCE(xuid_aliases, match_participants)` est déjà correct)
- [ ] Ajouter un test unitaire dans `tests/` vérifiant que `load_match_player_gamertags` préfère `xuid_aliases` quand les deux sources diffèrent

**Étape 2 — Suppression de `highlight_events.gamertag` (priorité moyenne)**

- [ ] Confirmer qu'aucun code ne lit `highlight_events.gamertag` directement hors de `_gamertag_resolver.py`
- [ ] Retirer la colonne `gamertag` du schéma `highlight_events` dans `_engine_connections.py`
- [ ] Retirer la colonne `gamertag` du transformateur `transformers/_events.py`
- [ ] Supprimer la source #3 (`highlight_events`) dans `load_match_player_gamertags()` — `xuid_aliases` couvre le même cas avec meilleure qualité
- [ ] Créer une migration `add_drop_highlight_events_gamertag.py` dans `src/data/migration/steps/`
- [ ] Ajouter l'import dans `src/data/migration/steps/__init__.py`

**Étape 3 — Statut de `match_participants.gamertag` (à décider)**

- [ ] Décider si la colonne est conservée comme fallback historique ou supprimée :
  - **Conserver** : utile pour XUIDs absents de `xuid_aliases` (matchs très anciens, bots, edge cases API) — mais jamais comme source prioritaire
  - **Supprimer** : simplifie le schéma, force le passage complet par `xuid_aliases`
- [ ] Si conservée : s'assurer que son rôle est documenté en commentaire dans les schémas et le resolver
- [ ] Si supprimée : créer la migration correspondante et retirer du transformateur `_match.py`

**Étape 4 — Test de non-régression**

- [ ] Test : un gamertag modifié dans `xuid_aliases` est affiché correctement dans toutes les pages (scoreboard, Rencontres, antagonistes, sélecteur joueur)
- [ ] Test : pas de régression sur les matchs anciens dont le joueur n'est plus dans `xuid_aliases`

**Complexité** : M (étape 1 = S, étapes 2-3 = M chacune)  
**Fichiers clés** : [`src/data/repositories/_gamertag_resolver.py`](../src/data/repositories/_gamertag_resolver.py), [`src/data/sync/_engine_connections.py`](../src/data/sync/_engine_connections.py), [`src/data/sync/transformers/_events.py`](../src/data/sync/transformers/_events.py), [`src/data/migration/steps/`](../src/data/migration/steps/)

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-13 | Couverture tests `migrations.py` (lacunes v5.5–v5.7) |
| 2026-03-13 | Conflit `shared_matches.duckdb` — sync depuis UI Streamlit |
| 2026-03-13 | Hover thumbnail sur les noms de cartes (tableaux HTML) |
| 2026-03-13 | Détection de langue système dans `LevelUp.sh` / `LevelUp.bat` |
| 2026-03-13 | [UI] Heatmap performance par joueur × carte — Page Teammates |
| 2026-03-13 | [UI] Performance par carte vs historique — vues escouade et joueur |
| 2026-03-13 | Audit Pandas → Polars — résidus nettoyés |
| 2026-03-13 | Kwargs legacy SyncScope — nettoyage différé |
| 2026-03-13 | Traductions FR manquantes dans migration metadata |
| 2026-03-13 | Images citations d'armes incorrectes |
| 2026-03-08 | Bug #0 : match invisible post-sync — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | Bug #1 : `win_rate` unifié sur `NULLIF(WIN+LOSS, 0)` dans `analytics.py` et `trends.py` |
| 2026-03-08 | Bug #5 : NaN-check fragile dans `match_view.py` → `is not None` |
| 2026-03-08 | Dette #2 : guard obsolète `_PERF_SCORE_AVAILABLE` supprimé dans `_performance.py` |
| 2026-03-08 | Dette #3 : dead code `_ensure_performance_score_column()` supprimé |
| 2026-03-08 | Dette #4 : magic number `outcome == 4` → `Outcome.DID_NOT_FINISH` |
| 2026-03-08 | Dette #6 : magic SQL `2`/`3` → constantes `_WIN`/`_LOSS` dans `analytics.py` |
| 2026-03-08 | i18n-1 : clés tronquées `PAIR_FR` restaurées dans `translations.py` |
| 2026-03-08 | i18n-2 : 342 entrées redondantes supprimées de `PAIR_FR` (399 → 57) |
| 2026-03-08 | i18n-3 : doublon `tm_session_trend` supprimé dans `widgets.py` |
| 2026-03-08 | Kwargs legacy SyncScope — dépréciés + `scope=SyncScope(...)` opérationnel |
| 2026-03-08 | `career.py` migré vers `get_cached_repository_st()` (plus de `duckdb.connect()` nu) |
| 2026-03-12 | `custom_rules.py:103` — TODO disparu, `compute_annexion_forcee` implémentée |
| 2026-03-12 | Pandas éliminé du code métier — résidus légitimes uniquement |
| 2026-03-08 | Perf UI — vues matérialisées reconstruites uniquement post-sync dans `engine.py` |
| 2026-03-08 | Perf UI — lazy-loading `match_view` via `st.tabs` + `@fragment_if_available` |
| 2026-03-08 | Perf UI — pagination SQL `LIMIT/OFFSET` sur `mv_player_matches` |
| 2026-03-08 | Perf UI — projections Polars fines par page dans `cache_loaders.py` |
| 2026-03-08 | i18n câblage `t()` dans les pages/widgets Streamlit |
| 2026-03-08 | CI/CD — détection de régression + pre-commit hook |
| 2026-02-26 | Quick wins perf UI (cache TTL, `@lru_cache`, `@st.cache_data`) |
| 2026-02-25 | v5.3 LUSR stabilisation + UI Carrière |
| 2026-02-25 | i18n Phase 1b — traductions EN registres |
| 2026-02-20 | v5.2 : Filtres intent-based + Stats PvE Firefight |
| 2026-02-17 | Release v5.1 — architecture shared-only |
| 2026-02-15 | Remédiation P0/P1 sécurité SQL + conformité Streamlit |
