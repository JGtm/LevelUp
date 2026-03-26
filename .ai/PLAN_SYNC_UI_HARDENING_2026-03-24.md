# Plan détaillé — Hardening du sync via l'app (UI)

**Date :** 2026-03-24
**Branche de travail :** fix/sync-ui-hardening-plan

---

## 1) Contexte

Le flux de sync UI (sidebar Streamlit) est fonctionnel, mais la revue technique a mis en évidence :

1. Un risque d'ambiguïté UX sur les succès partiels (erreurs réelles possibles malgré message vert).
2. Un coût évitable en mode delta (chargement d'état DB potentiellement lourd avant le HEAD check).
3. Une distinction auth implicite qui mérite d'être rendue explicite dans le code.
4. Une invalidation cache potentiellement inutile en cas d'échec par joueur.
5. Un écart de fraîcheur perçue : le label "Mis à jour il y a XXX" peut rester obsolète pour les joueurs non actifs après un sync global.

Un audit code approfondi de la chaîne complète (sidebar → `sync_all_players_duckdb` → `_sync_all_players_loop` → `sync_player_duckdb` → `DuckDBSyncEngine._sync_internal` → mixins) a confirmé ces constats et en a révélé d'autres :

6. **Event loop recréée par joueur** : `asyncio.run()` appelé N fois dans la boucle multi-joueurs au lieu d'une seule event loop partagée.
7. **Absence de gestion transactionnelle** : aucun `BEGIN TRANSACTION` / `ROLLBACK` — les commits intermédiaires (`_maybe_batch_commit`) créent un risque de corruption soft en cas de crash.
8. **`contextlib.suppress(Exception)` abusif** : ≥12 occurrences dans la chaîne sync masquent des erreurs critiques (CHECKPOINT raté, connexions non libérées).
9. **Connexion bare `duckdb.connect`** dans `_compute_dominance_post_sync` — fuite de file handle possible.
10. **`_load_existing_match_ids` en 3 requêtes séquentielles** + intersections Python — consolidable en une seule requête SQL.
11. **Fan-out synchrone et non borné** : `_enrich_other_registered_players()` bloque le spinner UI pendant le post-traitement de tous les coéquipiers.
12. **Compteur `remaining` consomme les matchs skippés en mode full** : un sync full avec beaucoup de matchs existants épuise son quota sans rien récupérer de nouveau.
13. **Code mort** : `_sync_duckdb_player()` n'est plus appelé dans le flux app (remplacé par `sync_player_duckdb`).

Ce plan vise à améliorer robustesse + observabilité + perf, sans casser les comportements métier existants.

---

## 2) Contrainte métier prioritaire (auth)

### 2.1 Réalité de l'API Halo Infinite

L'API Halo distingue deux familles d'endpoints :

1. **Endpoints "any-token"** : un seul spartan_token valide (n'importe quel joueur) suffit pour récupérer les données de match de **tous** les joueurs.
   - match_history, match_stats, skill/MMR, highlight_events, film metadata, film chunks, assets (maps/playlists/variants), gamertag↔XUID resolution, match_count
   - Conséquence : en sync multi-joueurs, on peut récupérer toutes les données de match de joueur B avec le token de joueur A.

2. **Endpoints "player-gated"** : nécessitent les tokens du joueur cible lui-même.
   - `economy.svc/.../careerranks/careerrank1` — Career rank, XP, progression
   - `economy.get_player_customization()` — Spartan ID (bannière, backdrop, emblème, adornment, service tag, nameplate)
   - Si on n'a pas les tokens du joueur cible, ces données sont simplement **skippées** (pas d'erreur bloquante).

### 2.2 Règle à conserver

Le système doit conserver les deux logiques :

- **Sync match data** : fonctionne avec le token global (token de l'utilisateur connecté ou premier refresh_token disponible). Un seul token valide suffit pour syncer les matchs de **tous** les joueurs du profil.
- **Sync identity data** (career rank + spartan ID) : ne fonctionne que si les tokens spécifiques du joueur cible sont disponibles (env var `SPNKR_OAUTH_REFRESH_TOKEN_<GT>` ou cache DB `sync_meta`).

Le sync ne doit **jamais** échouer globalement parce qu'un joueur n'a pas de token per-player. Les données de match se récupèrent quand même. Seules les données player-gated sont skippées avec un warning.

### 2.3 Implication architecture

On ne fusionne pas ces deux logiques. On les distingue par **catégorie d'endpoint** :

- `token_scope="any"` : tout appel API qui accepte n'importe quel spartan_token valide. Pas besoin du token du joueur cible.
- `token_scope="player"` : appel API qui exige les tokens du joueur concerné. Skip clean si indisponible.

Le vocabulaire `auth_mode="required"` / `auth_mode="optional"` de la version précédente était ambigu : il suggérait que certaines étapes peuvent tourner *sans aucun token*, ce qui est faux. Tous les appels API nécessitent *un* token — la question est de savoir *lequel*.

**Terminologie retenue** :
- `token_scope="any"` → un token global suffit
- `token_scope="player"` → token du joueur cible obligatoire
- Pas de `"optional"` ambigu

---

## 3) Objectifs / Non-objectifs

### 3.1 Objectifs

1. Rendre le statut de sync fidèle (succès total vs partiel vs échec).
2. Réduire le coût de sync delta pour les joueurs déjà à jour.
3. Clarifier les chemins auth (token_scope any vs player — cf. Phase C).
4. Éviter les invalidations cache non nécessaires.
5. Ajouter des métriques et logs permettant d'expliquer chaque décision de pipeline.
6. Garantir la cohérence de l'indicateur "Mis à jour il y a XXX" pour chaque joueur après sync global.
7. Consolider le runtime async (une seule event loop pour la boucle multi-joueurs).
8. Éliminer les masquages d'erreurs (`suppress(Exception)`) et les connexions bare.
9. Améliorer l'intégrité crash-safe (transactions explicites).
10. Rendre le fan-out post-sync non bloquant pour l'UX.
11. **Zéro workaround / fallback excessif** : chaque correction traite la cause racine. Pas de guard temporaire, pas de retry aveugle, pas de fallback silencieux qui masque un vrai problème.
12. **Couverture logging exemplaire** : chaque décision prise par le pipeline (skip, short-circuit, auth decision, rollback, fan-out) doit être traçable dans les logs sans reproduire localement.
13. **Couverture tests rigoureuse** : chaque phase doit fournir des tests positifs, négatifs et de cas limites. Les nouveaux chemins de code doivent atteindre ≥90% de couverture branches.
14. **Garantir que tous les enrichissements locaux sont effectués** pour tous les joueurs partageant des matchs, en éliminant les conflits de connexion shared et en étendant le fanout aux enrichissements manquants.

### 3.2 Non-objectifs

1. Refactor complet de DuckDBSyncEngine ou des mixins.
2. Changement de schéma DB.
3. Changement de stratégie API globale (RPS/parallélisme) hors ajustements mineurs.
4. Suppression des capacités legacy utiles à l'exploitation.
5. Suppression de la distinction token_scope (any/player) — cf. Phase C.

### 3.3 Principes anti-workaround & anti-fallback

Ces règles s'appliquent transversalement à toutes les phases :

1. **Pas de retry aveugle** : si une opération échoue, logger l'erreur et la propager. Pas de `time.sleep(N) + retry` sans raison documentée et bornée.
2. **Pas de guard temporaire** : pas de `if asyncio.get_event_loop().is_running(): fallback_sync()`. Si le contexte d'exécution est maîtrisé (c'est le cas — thread principal Streamlit), le coder pour ce contexte uniquement.
3. **Pas de fallback silencieux** : si on ne peut pas faire l'opération, c'est une erreur (loggée + propagée), pas un `return None` discret.
4. **Pas de cache de secours** : si `session_state` perd le marqueur fan-out entre deux rerun, ce n'est pas un workaround acceptable — persister en DB si nécessaire, ou accepter de reporter au prochain sync.
5. **Pas de `suppress(Exception)` résiduel** : à la fin du hardening, il ne doit rester aucun `suppress(Exception)` dans la chaîne sync. Les seuls `except Exception` tolérés sont des `try/except` avec log explicite.
6. **Pas de code "au cas où"** : chaque branche de code doit être atteignable et testée. Pas de `except` pour des cas théoriques sans test prouvant que le cas existe.
7. **Pas de duplication de logique pour contourner un bug** : corriger le bug à la source, pas à l'endroit de consommation.

Ces principes sont des critères de revue bloquants pour chaque PR de ce plan.

### 3.4 Exigences de logging

1. **Chaque phase** doit avoir un bloc "Logs attendus" dans sa section 11.x avec les lignes de log exactes à vérifier.
2. **Niveaux** : `INFO` pour les décisions normales (sync_start, sync_end, status), `WARNING` pour les erreurs récupérables (auth optional sans token, CHECKPOINT échoué), `ERROR` pour les échecs critiques (auth required sans token, DB corrompue).
3. **Format** : clé=valeur structuré (ex: `event=sync_start mode=delta players=3`). Pas de format libre.
4. **Testabilité** : les tests doivent asserter la présence des messages de log (via `caplog` ou mock logger). Un changement de comportement silencieux sans log = bug.

### 3.5 Exigences de couverture tests

1. **Chaque nouvelle fonction** doit avoir au minimum un test positif + un test négatif + un test de cas limite.
2. **Couverture branches** : ≥90% sur tout le code ajouté/modifié. Vérifiable via `pytest --cov` avec `--cov-fail-under` sur les fichiers touchés.
3. **Pas de test qui passe en l'absence de code** : chaque test doit échouer si on supprime la fonctionnalité qu'il couvre (test de pertinence).
4. **Tests d'erreur obligatoires** : chaque `try/except` ajouté doit avoir un test qui déclenche le `except` et vérifie le comportement (log émis, valeur de retour, propagation).
5. **Mocking minimal** : mocker l'API (réseau) et les fichiers, pas la logique interne. Les tests doivent exercer le vrai code DuckDB en mémoire (`:memory:` ou tmpdir).
6. **Non-régression** : avant de modifier une fonction existante, s'assurer qu'il existe un test couvrant son comportement actuel. Si absent, l'ajouter d'abord.

---

## 4) Plan d'implémentation détaillé

## Phase A — Statut de sync fiable et lisible

### A.1 Modèle de résultat

- Introduire explicitement 3 états de sortie :
  - success (0 erreur)
  - partial_success (au moins 1 insertion mais erreurs non vides)
  - failure (aucune insertion utile + erreurs bloquantes)
- Conserver la compatibilité descendante de SyncResult.to_message().

### A.2 Affichage UI

- Côté Streamlit :
  - success -> st.success(...)
  - partial_success -> st.warning(...) + résumé des erreurs
  - failure -> st.error(...)

### A.3 Critères d'acceptation

1. Un sync avec erreurs non vides n'apparaît plus en succès plein.
2. Le message final contient un sous-résumé d'erreurs (au moins les 1-2 premières).
3. Aucun changement régressif sur les cas déjà à jour.

---

## Phase B — Optimisation delta (HEAD-first réel) + consolidation requêtes

### B.1 Principe

- En delta, faire d'abord un check API HEAD (count=1) + match DB le plus récent.
- Si identique : sortie immédiate sans charger l'ensemble complet des existing_ids.
- Charger existing_ids uniquement si le HEAD indique qu'il y a du nouveau (ou incertitude).

### B.2 Ajustements prévus

- Déplacer/retarder le calcul de existing_ids dans le chemin qui en a besoin.
- Ajouter un compteur de télémétrie (logs) :
  - delta_head_short_circuit=true|false
  - existing_ids_loaded=true|false

### B.3 Consolidation de `_load_existing_match_ids`

Actuellement, 3 requêtes séquentielles :
1. `SELECT match_id, personal_score FROM shared.match_participants WHERE xuid = ?`
2. `SELECT DISTINCT match_id FROM player_match_enrichment`
3. `SELECT DISTINCT match_id FROM personal_score_awards`

Puis intersections Python en mémoire — charge ~15 000 rows pour un joueur avec 5000 matchs.

**Cible** : une seule requête SQL avec ATTACH + JOINs côté DuckDB :
```sql
-- Sur la connexion player DB, ATTACH shared en lecture seule
SELECT mp.match_id
FROM shared.match_participants mp
JOIN player_match_enrichment e ON e.match_id = mp.match_id
WHERE mp.xuid = ?
  AND (EXISTS (SELECT 1 FROM personal_score_awards p WHERE p.match_id = mp.match_id)
       OR COALESCE(mp.personal_score, 0) = 0)
```

**Gain attendu** : -70% mémoire Python, ~3x plus rapide (exécution vectorisée DuckDB).

### B.4 Fix `remaining` en mode full

Bug actuel : un match déjà existant en mode full décrémente `remaining`, consommant le quota sans récupérer de données nouvelles. Un full sync `max_matches=200` sur un joueur avec 500 matchs en DB épuise son quota sur les 200 premiers matchs (tous skippés).

**Fix** : ne décrémenter `remaining` que pour les matchs effectivement traités (insérés). Les skipped ne consomment que `start` (pagination), pas le quota.

### B.5 Critères d'acceptation

1. Joueur à jour : 1 appel historique API, puis stop rapide.
2. Pas de changement de résultat métier sur les cas non à jour.
3. Gain de temps observé sur sync delta no-op.
4. `_load_existing_match_ids` exécute une seule requête SQL.
5. Mode full avec matchs existants ne gaspille plus le quota `max_matches`.
6. **Test de parité obligatoire** : `test_load_existing_match_ids_parity` couvre 4 cas — match normal, match `personal_score=0` sans awards, match partial (enrichment sans awards), match complet — et vérifie que la nouvelle requête retourne exactement les mêmes `match_id` que l'ancienne implémentation 3 requêtes.
7. **Commentaire inline obligatoire** sur la condition `OR COALESCE(mp.personal_score, 0) = 0` : documenter que ce comportement est un changement intentionnel par rapport à l'ancienne intersection stricte (les matchs 0-score sans awards sont désormais considérés comme traités).

---

## Phase C — Clarification explicite des chemins auth (token_scope any vs player)

### C.1 Décision

- Rendre explicite le scope de token utilisé par chaque étape de sync :
  - `token_scope="any"` : un token global valide suffit (match data)
  - `token_scope="player"` : token du joueur cible obligatoire (career rank, spartan ID)

### C.2 Matrice complète des endpoints (figée)

| Endpoint API | Données | token_scope | Comportement sans token joueur |
|---|---|---|---|
| `stats.get_match_history()` | Historique matchs | `any` | Fonctionne avec token global |
| `stats.get_match_stats()` | Stats d'un match | `any` | Fonctionne avec token global |
| `skill.get_match_skill()` | CSR/MMR | `any` | Fonctionne avec token global |
| `film.read_highlight_events()` | Events film (kills, etc.) | `any` | Fonctionne avec token global |
| `discovery_ugc.get_film_by_match_id()` | Film metadata | `any` | Fonctionne avec token global |
| Film chunk download | Chunks film (Azure blob) | aucun | URL pré-signée |
| `profile.get_user_by_gamertag()` | GT→XUID | `any` | Fonctionne avec token global |
| `profile.get_users_by_id()` | XUIDs→profiles | `any` | Fonctionne avec token global |
| `halostats.svc/.../matches/count` | Nombre matchs | `any` | Fonctionne avec token global |
| `discovery_ugc.get_*()` | Assets (maps, playlists) | `any` | Fonctionne avec token global |
| `gamecms_hacs.get_career_reward_track()` | Métadonnées rangs | `any` | Fonctionne avec token global |
| **`economy.svc/.../careerranks/...`** | **Career rank, XP** | **`player`** | **Skip + warning** |
| **`economy.get_player_customization()`** | **Spartan ID** | **`player`** | **Skip + warning** |

### C.3 Hiérarchie de résolution de tokens (code actuel dans `_tokens.py`)

1. `SPNKR_OAUTH_REFRESH_TOKEN_<GT_NORMALISÉ>` (env var per-player)
2. Cache DB `sync_meta` (per-player, persisté après Device Code Flow)
3. `SPNKR_OAUTH_REFRESH_TOKEN` (env var global — fallback)
4. `SPNKR_SPARTAN_TOKEN` + `SPNKR_CLEARANCE_TOKEN` (tokens manuels directs)

Les étapes 1-2 fournissent un **token du joueur cible** (→ `token_scope="player"` OK).
Les étapes 3-4 fournissent un **token global** (→ `token_scope="any"` OK, `token_scope="player"` skip).

### C.4 Gouvernance des erreurs

- **Aucun token du tout** (ni global ni per-player) : erreur bloquante avant le sync. Remédiation = reconnexion Xbox (Device Code Flow).
- **Token global présent mais pas de token per-player** : le sync match data fonctionne normalement. Career rank et spartan ID sont skippés avec warning structuré.
- **Token per-player présent** : tout fonctionne (match data + career rank + spartan ID).

### C.5 Critères d'acceptation

1. Chaque appel API dans le code est annoté avec son `token_scope`.
2. Aucune régression : le sync match data fonctionne toujours avec un token global seul.
3. Les skips player-gated sont visibles dans les logs (pas silencieux).
4. Les tests couvrent les 3 cas : aucun token, token global seul, token per-player.

---

## Phase D — Invalidation cache/mtime plus fine

### D.1 Changement

- Mettre à jour mtime/caches uniquement après succès utile du joueur concerné.
- Conserver l'invalidation globale finale si le résultat global est succès/partiel selon stratégie retenue.

### D.2 Critères d'acceptation

1. Pas de rafraîchissement inutile lorsque la sync d'un joueur échoue avant écriture.
2. Les vues se rafraîchissent bien dès qu'il y a écriture réelle.

---

## Phase E — Observabilité, diagnostics et hygiène d'erreurs

### E.1 Logs structurés

Ajouter des logs normalisés (une ligne par décision-clé) :

- sync_mode=delta|full
- token_scope=any|player (remplace auth_mode)
- delta_head_short_circuit=true|false
- status=success|partial_success|failure
- matches_inserted, warnings_count, errors_count

### E.2 KPIs de suivi

1. Durée moyenne d'un delta no-op
2. Taux de succès partiel
3. Taux d'échec auth
4. Nombre moyen d'appels API par sync

### E.3 Nettoyage `suppress(Exception)` (audit code)

≥12 occurrences de `contextlib.suppress(Exception)` dans la chaîne sync masquent des erreurs qui devraient être loggées, même si non bloquantes.

**Localisation des cas critiques** :

| Fichier | Méthode | Risque masqué |
|---------|---------|---------------|
| `_engine_connections.py` | `close()` (CHECKPOINT) | WAL non flushé |
| `_sync_duckdb_ops.py` | `_activate_sync_mode()` | Connexions R/O non libérées → conflit R/W |
| `engine.py` | `_run_post_sync_compute()` | shared_conn non fermée |
| `engine.py` | `_compute_dominance_post_sync()` | Connexion bare non nettoyée |
| `engine.py` | `_run_lusr_post_sync()` | shared_conn zombie |

**Principe** : remplacer chaque `suppress(Exception)` par un `try/except Exception as e: logger.debug(...)` minimum. Les erreurs de CHECKPOINT et de release de connexion doivent être loggées en `warning`, pas masquées.

### E.4 Fix connexion bare `duckdb.connect`

Dans `_compute_dominance_post_sync()` (engine.py L489) :
```python
# Avant (bare connect)
_sconn = _ddb.connect(str(shared_path), read_only=True)
try: ...
finally: _sconn.close()

# Après (context manager)
with _ddb.connect(str(shared_path), read_only=True) as _sconn:
    ...
```

**Impact** : élimine un risque de fuite de file handle si l'exception survient entre `connect()` et le `try`.

### E.5 Nettoyage hygiène bitmasks (`constants.py` / `migrations.py`)

Deux anomalies résiduelles dans le système de bitmasks, sans impact fonctionnel mais sources de confusion :

**Anomalie 1 — bit `performance_scores` à sémantique incorrecte**

`_compute_backfill_mask()` ([src/data/sync/_match_processing_helpers.py](src/data/sync/_match_processing_helpers.py)) pose `BACKFILL_FLAGS["performance_scores"]` (1<<4) sur `match_registry.backfill_completed` (shared DB, granularité par match). Or `performance_score` est un calcul **par joueur** stocké dans `player_match_enrichment` — sa granularité est joueur × match, incompatible avec un bit par match. Le bit n'est relu par personne pour décider de recalculer (la détection se fait via `IS NULL` dans `player_match_enrichment`). Il pollue `backfill_completed` avec une sémantique trompeuse.

**Fix** : supprimer la ligne dans `_compute_backfill_mask()` :
```python
# Supprimer :
bf_mask |= BACKFILL_FLAGS["performance_scores"]
```

**Anomalie 2 — collision de bits documentée mais non bloquée**

`BACKFILL_FLAGS["lusr"]` (1<<16) et `BACKFILL_FLAGS["csr"]` (1<<17) **collisionnent** avec `MatchBits.EVENTS` (1<<16) et `MatchBits.ASSETS` (1<<17). C'est documenté comme "jamais écrit en production" mais rien n'empêche un futur appel à `compute_backfill_mask("lusr", ...)` de corrompre silencieusement les bits `EVENTS`/`ASSETS`.

**Fix** : supprimer directement ces entrées de `BACKFILL_FLAGS`. Un renommage `_OBSOLETE_*` laisse du code mort qui traîne — supprimer est la seule action nette.

Vérification préalable obligatoire :
```bash
grep -r '"lusr"\|"csr"\|"performance_scores"' src/data/sync/ --include="*.py"
```
Si le grep retourne 0 résultat (hors `BACKFILL_FLAGS` lui-même) : supprimer.

Ajouter un test de non-existence permanent :
```python
def test_backfill_flags_no_collision_with_match_bits():
    from src.data.sync.migrations import BACKFILL_FLAGS
    assert "lusr" not in BACKFILL_FLAGS, "collision avec MatchBits.EVENTS (1<<16)"
    assert "csr" not in BACKFILL_FLAGS, "collision avec MatchBits.ASSETS (1<<17)"
    assert "performance_scores" not in BACKFILL_FLAGS, "sémantique incorrecte (granularité joueur)"
```

Ce test échoue immédiatement si quelqu'un les rajoute accidentellement.

**Fichiers impactés** :
- `src/data/sync/_match_processing_helpers.py` — supprimer `BACKFILL_FLAGS["performance_scores"]` de `_compute_backfill_mask()`
- `src/data/sync/migrations.py` — supprimer ou renommer les entrées `"lusr"` et `"csr"` de `BACKFILL_FLAGS`

**Risque** : nul — le bit `performance_scores` n'est jamais relu, et les entrées `lusr`/`csr` ne sont pas utilisées en production. Aucun test existant ne doit en dépendre (vérifier via grep avant suppression).

---

## Phase F — Cohérence de l'indicateur "Mis à jour il y a XXX" (multi-joueurs)

### F.1 Problème à adresser

Le sync via l'app (`sync_all_players_duckdb`) itère sur **tous** les profils de `db_profiles.json` et appelle `sync_player_duckdb()` pour chacun. Chaque joueur passe par son propre `_sync_internal` → `_save_sync_metadata()`. **L'écriture de `last_sync_at` est donc correcte pour tous les joueurs.**

Le problème est dans l'**affichage** :

- **Joueur actif** : après le sync, le timestamp est stocké dans `session_state` → l'UI affiche le label depuis `session_state` (pas depuis DB). Label à jour. ✅
- **Autres joueurs** (switch de profil) : l'UI appelle `get_sync_metadata()` → lit depuis `meta.sync_meta` (mauvaise DB, `metadata.duckdb`) au lieu de la player DB locale → retourne toujours `None`. Label stale ou absent. ❌

**Il y a un seul bug : le lecteur `get_sync_metadata()` lit depuis la mauvaise table.**

### F.2 Objectif

Corriger `get_sync_metadata()` pour qu'il lise depuis la bonne source. C'est un fix de 3 lignes dans un seul fichier.

### F.3 Source canonique — déjà existante, lecteur cassé

- **Écriture** : `_save_sync_metadata()` dans `engine.py` écrit `key='last_sync_at', value=<ISO timestamp>` dans `sync_meta` de `stats.duckdb` du joueur, inconditionnellement après chaque sync. ✅
- **Lecteur cassé** : `DiagnosticMixin.get_sync_metadata()` dans `_diagnostic_repo.py` (L25) lit depuis `meta.sync_meta WHERE xuid = ?` (colonne `last_sync_at`) — c'est `metadata.duckdb` (référentiels), pas la player DB. Schéma incompatible, résultat toujours `None`. ❌

### F.4 Fix — Corriger `get_sync_metadata()` dans `_diagnostic_repo.py`

```python
# Avant (lit depuis meta.sync_meta — mauvaise DB, mauvais schéma)
if "meta" in self._attached_dbs:
    result = conn.execute(
        "SELECT last_sync_at FROM meta.sync_meta WHERE xuid = ?",
        [self._xuid],
    ).fetchone()
    last_sync = result[0] if result else None

# Après (lit depuis sync_meta locale du joueur — clé/valeur)
try:
    result = conn.execute(
        "SELECT value FROM sync_meta WHERE key = 'last_sync_at'",
    ).fetchone()
    last_sync = result[0] if result else None
except Exception:
    last_sync = None
```

Retirer la garde `if "meta" in self._attached_dbs` — inutile, la lecture est locale.

**Fichier unique impacté** : `src/data/repositories/_diagnostic_repo.py`

### F.5 Critères d'acceptation

1. `get_sync_metadata()` lit depuis `sync_meta WHERE key = 'last_sync_at'` (player DB locale), pas depuis `meta.sync_meta`.
2. Après un sync global, le switch de profil affiche un `last_sync_at` à jour pour tous les joueurs — sans rerun secondaire.
3. En cas d'échec d'un joueur (exception avant `_save_sync_metadata`), son label reste à l'ancienne valeur.
4. Player DB sans entrée `last_sync_at` → retourne `{"last_sync_at": None, ...}` sans exception.

### F.6 Tests ciblés

1. **Test lecteur corrigé** : player DB `:memory:` avec `sync_meta key='last_sync_at' value='2026-01-01T12:00:00Z'` → `get_sync_metadata()["last_sync_at"]` retourne `'2026-01-01T12:00:00Z'`, pas `None`.
2. **Test sync global 2 joueurs** : simuler `_save_sync_metadata()` pour 2 players → `get_sync_metadata()` retourne la bonne valeur pour chacun après switch.
3. **Test joueur sans sync** : player DB sans entrée `last_sync_at` → `{"last_sync_at": None, ...}`, pas d'exception.
4. **Test non-régression** : joueur actif après sync → `session_state` path inchangé (le fix ne touche que le path DB-read).

---

## Phase G — Consolidation runtime async + fan-out non bloquant + dead code

### G.1 Event loop unique pour la boucle multi-joueurs

**Problème** : `_sync_all_players_loop` appelle `sync_player_duckdb()` → `asyncio.run()` pour chaque joueur. Chaque `asyncio.run()` crée et détruit une event loop complète. Pour N joueurs = N event loops successives (overhead, impossibilité de paralléliser).

**Cible** : un seul `asyncio.run()` au sommet de `sync_all_players_duckdb()`, avec une boucle async interne :

```python
# Avant
def _sync_all_players_loop(...):
    for gamertag, profile in profiles.items():
        ok, msg = sync_player_duckdb(gamertag=gamertag, ...)  # asyncio.run() chaque fois

# Après
async def _sync_all_players_loop_async(...):
    for gamertag, profile in profiles.items():
        ok, msg = await sync_player_duckdb_async(gamertag=gamertag, ...)

def sync_all_players_duckdb(...):
    ...
    return asyncio.run(_sync_all_players_loop_async(...))
```

**Fichiers** : `src/ui/sync.py`, `src/ui/_sync_duckdb_ops.py`

**Compatibilité Phase C** : cette refonte ne change pas les chemins auth. Chaque joueur continue à utiliser ses tokens propres (ou le token global si pas de token per-player) dans le même ordre qu'aujourd'hui. Le token_scope (any/player) est évalué indépendamment par joueur, inchangé.

### G.2 Fan-out post-sync non bloquant

**Problème** : `_enrich_other_registered_players()` est synchrone et séquentiel. Pour 5 coéquipiers × 50 matchs, ça rallonge le temps perçu de sync significativement derrière le spinner UI.

**Options** :

1. **Option A (recommandée)** : exécuter le fan-out dans un `run_in_executor` (thread pool) après le `st.rerun()`, via un marqueur `session_state` + flag de persistance DB qui déclenche le fan-out au prochain cycle Streamlit.
2. **Option B** : exécuter le fan-out comme tâche background lancée par un thread daemon, avec un indicateur "Enrichissement en cours..." dans la sidebar.
3. **Option C (minimaliste)** : garder synchrone mais borner à max 3 joueurs et timeout par joueur.

**Recommandation** : Option A avec persistance DB — découple l'UX du fan-out et résiste au redémarrage du process Streamlit.

**Détail de la persistance** : après le sync principal et avant le `st.rerun()`, poser un flag dans `sync_meta` du joueur principal :

```python
# Dans sync_meta de stats.duckdb — posé avant st.rerun()
conn.execute("INSERT OR REPLACE INTO sync_meta (key, value, updated_at) VALUES ('fanout_pending', 'true', now())")
```

Au démarrage du cycle Streamlit suivant (sidebar render), checker et exécuter si présent :

```python
if repo.get_sync_meta("fanout_pending") == "true":
    _enrich_other_registered_players(last_inserted_ids_from_registry)
    repo.clear_sync_meta("fanout_pending")
```

`last_inserted_ids` se reconstruit depuis `match_registry WHERE inserted_at > (last_fanout_at - 5min)`.

**Cas process restart** : si Streamlit redémarre entre le sync et le cycle suivant, le flag `fanout_pending` est en DB et sera lu au prochain démarrage. Pas de perte silencieuse.

**Commentaire inline obligatoire** dans la fonction qui pose le flag :
```python
# Fan-out opportuniste : posé en DB pour résister à un process restart.
# Si le flag est perdu (corruption), le fan-out sera rattrapé au prochain
# sync complet du coéquipier concerné. Comportement acceptable.
```

**Fichiers** : `src/data/sync/engine.py`, `src/data/sync/_engine_fanout.py`, `streamlit_app.py`

**Compatibilité Phase C** : le fan-out n'utilise pas l'API Halo (pas d'auth requise). Ce sont des calculs locaux (perf_scores, sessions, citations) sur les DBs joueur. Mode auth = `optional` par nature : aucun token n'est nécessaire, aucune interaction avec la matrice auth de Phase C.

### G.3 Suppression du code mort

`_sync_duckdb_player()` dans `_sync_duckdb_ops.py` (L126-L173) n'est plus appelé dans le flux app. Il a été remplacé par `sync_player_duckdb()` qui passe par `sync_player_duckdb_async()`.

**Action** : supprimer `_sync_duckdb_player` + sa coroutine interne `_run_duckdb_player_sync_async` (L65-L120), ou les marquer `DEPRECATED` avec date de suppression si des scripts externes les utilisent encore.

**Vérification préalable** : `grep -r "_sync_duckdb_player\|_run_duckdb_player_sync_async" --include="*.py"` pour confirmer qu'aucun autre caller n'existe.

### G.4 Critères d'acceptation

1. Un seul `asyncio.run()` pour le sync de N joueurs.
2. Fan-out non bloquant : le st.rerun() côté utilisateur survient avant le fan-out.
3. Code mort supprimé ou marqué deprecated.
4. Pas de régression sur les tests existants.
5. Pas d'impact sur les chemins auth (Phase C) — le fan-out ne fait aucun appel API (token_scope=n/a), l'event loop unique ne change pas l'ordre d'évaluation des tokens.

---

## Phase H — Intégrité transactionnelle (hardening avancé)

### H.1 Problème

Aucun `BEGIN TRANSACTION` / `ROLLBACK` dans le moteur. Le code fait des `commit()` intermédiaires (`_maybe_batch_commit`) et un `commit()` final. Impact en cas de crash :

- Shared DB : matchs partiellement écrits (registry sans tous les participants, events sans killer_victim_pairs).
- Player DB : enrichments orphelins sans personal_score_awards correspondantes.

Le rattrapage partiel existant (`_load_existing_match_ids` vérifie le triple JOIN enrichment ∩ scored ∩ shared) couvre les cas les plus fréquents, mais un match avec registry + participants mais sans enrichment player ne sera pas re-traité (il passe par `_process_known_match` qui skip si `participants_loaded=True`).

### H.2 Stratégie proposée

**Option A (par match)** : transaction par `_process_single_match`. Granularité maximale, mais sur un sync d'une heure (400-500 matchs), l'overhead de 400-500 COMMIT/BEGIN peut être perceptible.

**Option B (par batch de 50, recommandée)** : transaction autour de chaque batch de 50 matchs (aligné sur `batch_commit_size`). En cas de crash, on perd au maximum 50 matchs (quelques minutes de fetch), pas l'intégralité du sync. C'est le bon compromis entre granularité de récupération et overhead transactionnel.

**Si les benchmarks montrent un overhead > 5% avec N=50** : augmenter à N=100. Ne pas dépasser 100 — au-delà, un crash peut représenter 10-15 minutes de travail perdu.

**Implémentation** : repurposer `_maybe_batch_commit` comme point de commit transactionnel :

```python
def _maybe_batch_commit(self, count: int) -> None:
    """Commit le batch courant et ouvre une nouvelle transaction."""
    if count % self._batch_commit_size == 0:
        shared_conn = self._get_shared_connection()
        shared_conn.execute("COMMIT")
        shared_conn.execute("BEGIN TRANSACTION")
        logger.debug("event=batch_commit count=%d", count)

# En amont, ouvrir la première transaction au début du traitement :
# shared_conn.execute("BEGIN TRANSACTION")
# En aval, COMMIT final après le dernier batch.
```

**Trade-off assumé** : un crash dans les 50 premiers matchs d'un batch perd ces matchs. Ils seront ré-insérés au prochain sync (détectés comme manquants via `_load_existing_match_ids`). Ce comportement doit être documenté par un commentaire dans `_process_matches`.

### H.3 Contrainte DuckDB

DuckDB ne supporte qu'un seul writer à la fois. L'`asyncio.Lock` (`_shared_db_lock`) garantit déjà la sérialisation des écritures. Les transactions explicites n'introduisent donc pas de contention supplémentaire.

`_maybe_batch_commit` **n'est pas supprimé** — il est repurposé comme gestionnaire de batch transactionnel (COMMIT + BEGIN). Sa fréquence reste calée sur `batch_commit_size` (défaut=50).

### H.4 Compatibilité Phase C

Les transactions ne modifient pas les chemins auth. L'auth est évaluée en amont (avant l'écriture), dans les appels API. La transaction ne porte que sur les écritures DuckDB post-fetch.

### H.5 Critères d'acceptation

1. Un crash simulé dans un batch de 50 matchs ne laisse pas de données partielles dans shared pour les matchs du batch interrompu.
2. Les matchs des batches précédents (déjà committés) sont correctement retrouvés par `_load_existing_match_ids` après redémarrage.
3. **Benchmark obligatoire avant merge (gate de merge)** : exécuter `tests/bench/test_transaction_overhead.py` avec N=50 matchs en `:memory:`, 10 runs. Overhead médian ≤ 5% vs baseline sans transaction. Résultat à coller dans la description PR. Si overhead > 5% avec N=50, monter à N=100 et re-benchmarker.
4. Le commentaire inline dans `_process_matches` documente le trade-off batch (50 matchs max perdus en cas de crash).

### H.6 Prérequis

Phase G.1 (event loop unique) facilite l'implémentation car la gestion des locks asyncio est simplifiée avec une seule event loop.

---

## Phase I — Extraction de `_run_post_sync_pipeline()` (réduction God function)

### I.1 Problème

`_sync_internal()` dans `src/data/sync/engine.py` est signalée par Ruff avec les violations `C901` (complexité cyclomatique), `PLR0912` (trop de branches) et `PLR0915` (trop de statements). La fonction orchestre à la fois :

- La récupération des tokens
- Le traitement des matchs (`_process_matches`)
- Le commit DuckDB
- Le pipeline post-sync complet : citations + perf_scores + sessions + dominance + LUSR + fan-out
- La gestion des erreurs + métadonnées de sync

Cet entrelacement rend la fonction quasi-impossible à tester par segment, contraint toute modification à relire 150+ lignes, et interdit d'exécuter le post-sync de façon autonome (par ex. dans un backfill manuel).

### I.2 Cible

Extraire un bloc `_run_post_sync_pipeline()` qui regroupe toutes les étapes post-commit :

```python
# src/data/sync/engine.py

async def _sync_internal(self, options: SyncOptions) -> SyncResult:
    """Orchestre le sync : fetch → commit → post-pipeline."""
    result = SyncResult()
    try:
        tokens = self._resolve_tokens(options)
        async with create_api_client(tokens, options.rps) as client:
            inserted_ids = await self._process_matches(client, options, result)

        if inserted_ids:
            await self._commit_and_save_meta(options, len(inserted_ids))
            await self._run_post_sync_pipeline(inserted_ids, options, result)
        else:
            self._save_sync_meta_no_new(options)

    except Exception as e:
        logger.error("sync_internal_error", error=str(e), exc_info=True)
        result.errors.append(str(e))
    finally:
        result.set_finished()
    return result


async def _run_post_sync_pipeline(
    self,
    inserted_ids: list[str],
    options: SyncOptions,
    result: SyncResult,
) -> None:
    """Pipeline post-sync : citations, perf, sessions, dominance, LUSR, fan-out.

    Appelable indépendamment (backfill manuel, test unitaire).
    """
    await self._run_post_sync_compute(inserted_ids)        # citations + perf + sessions
    self._compute_dominance_post_sync()                    # dominance flag
    self._detach_shared_from_player_conn()                 # prérequis LUSR
    await self._run_lusr_post_sync(inserted_ids)           # LUSR / CSR
    self._enrich_other_registered_players(inserted_ids)   # fan-out
```

**Résultat attendu** : `_sync_internal` descend à ~40 lignes, `_run_post_sync_pipeline` ≤ 30 lignes, chacun testable indépendamment.

### I.3 Périmètre

**Fichier principal** : `src/data/sync/engine.py`

Aucune modification des mixins ni de l'API publique de `DuckDBSyncEngine`. La signature de `sync_delta()` et `sync_full()` reste inchangée.

### I.4 Règle d'ordre dans `_run_post_sync_pipeline`

L'ordre des appels n'est pas libre — il reflète des dépendances de données réelles :

1. `_run_post_sync_compute` → doit passer avant LUSR (perf_scores requis pour LUSR en certaines configs)
2. `_compute_dominance_post_sync` → peut aller avant ou après LUSR (pas de dépendance)
3. `_detach_shared_from_player_conn` → **obligatoire** avant `_run_lusr_post_sync` (LUSR ouvre shared en mode propre — sinon Binder Error)
4. `_run_lusr_post_sync` → après detach
5. `_enrich_other_registered_players` → en dernier (lit shared en read-only, doit être libre)

**Cette contrainte doit être documentée par des commentaires inline dans la fonction, pas seulement ici.** Forme recommandée :

```python
# ORDRE CONTRAINT — ne pas réarranger
# 1. perf + sessions requis en amont de LUSR
await self._run_post_sync_compute(inserted_ids)
# 2. detach obligatoire avant LUSR (sinon Binder Error sur shared)
self._detach_shared_from_player_conn()
# 3. LUSR (nécessite shared détaché)
await self._run_lusr_post_sync(inserted_ids)
# 4. fan-out (shared libéré, read-only)
self._enrich_other_registered_players(inserted_ids)
```

### I.5 Critères d'acceptation

1. `_sync_internal` ≤ 50 lignes (Ruff `PLR0915` / `C901` éteint sans `# noqa`).
2. `_run_post_sync_pipeline` est `async def` autonome, testable via mock de chaque étape.
3. Pas de régression sur les tests existants (`pytest -q --ignore=tests/integration`).
4. L'ordre contraint est documenté par commentaires inline.
5. `_run_post_sync_pipeline` peut être appelée directement dans un script de backfill sans déclencher l'intégralité de `_sync_internal`.

### I.6 Prérequis

Aucun prérequis bloquant. Peut être implementée indépendamment de G et H, mais le merge après G (event loop unique) réduit les conflits potentiels sur `engine.py`.

### I.7 Logs attendus

| Événement | Niveau | Message structuré |
|-----------|--------|-------------------|
| Entrée pipeline | `debug` | `post_sync_pipeline_start` + `match_count=N` |
| Sortie pipeline | `debug` | `post_sync_pipeline_end` + `duration_ms=X` |
| Erreur dans une étape | `warning` | `post_sync_step_error` + `step=<nom>` + `error=<msg>` (non bloquant) |

---

## Phase J — Garantie d'enrichissement complet pour tous les joueurs sur matchs partagés

### J.1 Problème

Après un sync, le pipeline post-sync exécute 6 enrichissements locaux (performance_score, sessions, citations, dominance, LUSR, weapon_kills extraction). Ce sont des calculs **purement locaux** — lecture de `shared_matches.duckdb` + écriture dans `stats.duckdb` du joueur — sans aucun appel réseau. Ils ne devraient **jamais** échouer.

Or, ils échouent silencieusement dans un scénario précis et fréquent :

1. L'utilisateur lance un sync **depuis l'app Streamlit** (dashboard ouvert)
2. Streamlit maintient une connexion **R/O** sur `shared_matches.duckdb` (requêtes de lecture UI)
3. `_get_shared_connection()` tente d'ouvrir shared en **R/W** (systématiquement, même pour des SELECT)
4. DuckDB interdit R/W + R/O simultané → `"unique file handle conflict"`
5. Après 2 retry, retourne `None` → les enrichissements qui dépendent de cette connexion retournent `0` silencieusement
6. Le sync est rapporté comme succès

**Conséquence directe** : le joueur principal ET les coéquipiers (via fanout) se retrouvent avec des matchs sans `performance_score`, sans `session_id`, etc. La session la plus récente n'apparaît pas dans le graphe escouade (bug critique #1 du backlog).

**Audit des 6 enrichissements et leur vulnérabilité :**

| Enrichissement | Mode connexion shared | Vulnérable au conflit R/W ? | Inclus dans le fanout ? |
|---|---|:---:|:---:|
| **Performance scores** | `_get_shared_connection()` → R/W | ✅ OUI — cause racine | ✅ |
| **LUSR** | `_get_shared_connection()` → R/W | ✅ OUI | ❌ |
| **Sessions** | Connexion R/O fraîche propre | ❌ Non | ✅ |
| **Citations** | Connexion R/O fraîche propre | ❌ Non | ✅ |
| **Dominance** | `shared_conn` passé en paramètre | ⚠️ Dépend du caller | ❌ |
| **Weapon kills** | Optionnel (skip si None) | ❌ Non (API-dépendant) | ❌ |

**Observation clé** : sessions et citations ouvrent *déjà* leur propre connexion R/O fraîche — elles ne sont pas touchées par le bug. Le problème est **concentré** sur `_get_shared_connection()` qui ouvre systématiquement en R/W.

### J.2 Stratégie : fix root cause + vérification défensive

Pas de tables de journal ou de tracking complexes — ces calculs sont locaux et **doivent simplement marcher**. La stratégie est :

1. **Corriger `_get_shared_connection()`** pour supporter le mode R/O
2. **Ouvrir en R/O** partout où seuls des SELECT sont exécutés
3. **Ajouter une passe de vérification** post-pipeline légère
4. **LEFT JOIN défensif** dans le graphe escouade
5. **Étendre le fanout** aux enrichissements manquants (dominance, LUSR)

### J.3 Fix 1 — Paramètre `shared_read_only` sur `DuckDBSyncEngine`

**Fichier** : `src/data/sync/_engine_connections.py`

Ajouter un attribut `_shared_read_only: bool` au constructeur, propagé à `_get_shared_connection()` :

```python
# _engine_connections.py — _get_shared_connection()
def _open_shared() -> duckdb.DuckDBPyConnection:
    return duckdb.connect(str(self._shared_db_path), read_only=self._shared_read_only)
```

**Fichier** : `src/data/sync/engine.py`

Ajouter `shared_read_only: bool = False` au `__init__` de `DuckDBSyncEngine` :

```python
def __init__(self, ..., shared_read_only: bool = False) -> None:
    ...
    self._shared_read_only = shared_read_only
```

**Règle** :
- Sync principal (écrit dans shared) → `shared_read_only=False` (défaut, inchangé)
- Fanout (ne lit que shared) → `shared_read_only=True`
- `batch_compute_performance_scores()` / `batch_compute_lusr()` quand appelés post-sync → la connexion est déjà ouverte en R/W par le sync principal, pas de conflit
- Fanout crée un **nouvel** engine → celui-ci doit être R/O

### J.4 Fix 2 — Fanout en R/O

**Fichier** : `src/data/sync/_engine_fanout.py` — `_run_other_player_enrichment()`

```python
# Avant
engine = DuckDBSyncEngine(
    player_db_path=player_db_path,
    xuid=xuid,
    gamertag=gamertag,
    shared_db_path=shared_path,
)

# Après
engine = DuckDBSyncEngine(
    player_db_path=player_db_path,
    xuid=xuid,
    gamertag=gamertag,
    shared_db_path=shared_path,
    shared_read_only=True,  # ← Fanout ne fait que lire shared
)
```

**Impact** : élimine le conflit R/W vs R/O entre Streamlit et le fanout. Le fanout peut tourner librement même si le dashboard est ouvert.

### J.5 Fix 3 — Étendre le fanout aux enrichissements manquants

Actuellement le fanout ne calcule que 3 enrichissements sur 6 pour les coéquipiers :

| Enrichissement | Fanout actuel | Action |
|---|:---:|---|
| Performance scores | ✅ | Conserver |
| Sessions | ✅ | Conserver |
| Citations | ✅ | Conserver |
| Dominance | ❌ | **Ajouter** |
| LUSR | ❌ | **Ajouter** |
| Weapon kills | ❌ | Ne pas ajouter (API-dépendant, hors scope fanout) |

**Fichier** : `src/data/sync/_engine_fanout.py` — `_run_other_player_enrichment()`

Ajouter après les citations :

```python
# Dominance (calcul local, R/O sur shared)
self._run_other_dominance(gamertag, xuid, player_db_path, player_conn)

# LUSR (calcul local via shared R/O)
lusr_count = engine.batch_compute_lusr(force=False)
if lusr_count > 0:
    logger.info("fanout [%s]: %d LUSR calculé(s)", gamertag, lusr_count)
```

Avec un nouveau helper `_run_other_dominance()` calqué sur `_run_other_sessions()` :

```python
def _run_other_dominance(
    self, gamertag: str, xuid: str, db_path: Path, conn: duckdb.DuckDBPyConnection
) -> None:
    try:
        from src.data.dominance_backfill import compute_dominance_for_player
        with duckdb.connect(str(shared_path), read_only=True) as shared_ro:
            result = compute_dominance_for_player(conn, shared_ro, xuid)
        logger.info(
            "fanout [%s]: dominance — %d traité(s)",
            gamertag, result.get("processed", 0),
        )
    except Exception as exc:
        logger.warning("fanout [%s]: dominance échoué (non bloquant): %s", gamertag, exc)
```

### J.6 Fix 4 — Passe de vérification post-pipeline

Après le pipeline complet (perf + sessions + citations + dominance + LUSR + fanout), ajouter une vérification légère qui **détecte** les enrichissements manquants sans les recalculer (juste du logging d'alerte).

**Fichier** : `src/data/sync/engine.py` — en fin de `_run_post_sync_pipeline()` (ou équivalent actuel)

```python
def _verify_enrichment_completeness(self, inserted_ids: list[str]) -> None:
    """Vérification post-pipeline : détecte les matchs sans enrichissement."""
    if not inserted_ids:
        return
    conn = self._get_connection()
    placeholders = ", ".join(["?" for _ in inserted_ids])

    # Matchs insérés mais sans performance_score
    missing_perf = conn.execute(
        f"SELECT COUNT(*) FROM player_match_enrichment "
        f"WHERE match_id IN ({placeholders}) AND performance_score IS NULL",
        inserted_ids,
    ).fetchone()[0]

    # Matchs insérés mais sans session_id
    missing_sess = conn.execute(
        f"SELECT COUNT(*) FROM player_match_enrichment "
        f"WHERE match_id IN ({placeholders}) AND session_id IS NULL",
        inserted_ids,
    ).fetchone()[0]

    if missing_perf > 0 or missing_sess > 0:
        logger.warning(
            "event=enrichment_incomplete perf_missing=%d sessions_missing=%d total=%d",
            missing_perf, missing_sess, len(inserted_ids),
        )
    else:
        logger.info(
            "event=enrichment_complete total=%d", len(inserted_ids),
        )
```

**Principe** : pas de retry automatique, juste de la **visibilité**. Si on voit `enrichment_incomplete` dans les logs, c'est qu'il y a un bug de connexion à investiguer — pas un état normal.

### J.7 Fix 5 — LEFT JOIN défensif dans le graphe escouade

**Fichier** : `src/analysis/_performance_squad.py` — `_join_perf_frames()`

Indépendamment du fix R/O, le graphe escouade doit **tolérer** un score manquant plutôt que de supprimer le match entier :

```python
# Avant
joined = joined.join(part, on="match_id", how="inner")

# Après
joined = joined.join(part, on="match_id", how="left")
```

Avec le calcul `squad_perf` qui ignore les nulls (Polars `mean()` ignore les nulls par défaut) :
- Session avec 3 joueurs scorés → moyenne des 3 ✅
- Session avec 2 joueurs scorés sur 3 → moyenne des 2 ✅ (mieux qu'invisible)
- Session avec 0 joueur scoré → filtrée par le `.filter()` existant ✅

### J.8 Critères d'acceptation

1. `_get_shared_connection()` respecte `self._shared_read_only` (R/O ou R/W selon le paramètre).
2. Le fanout crée ses engines avec `shared_read_only=True` — plus jamais de conflit R/W quand Streamlit est ouvert.
3. Le fanout exécute 5 enrichissements (perf, sessions, citations, dominance, LUSR) au lieu de 3.
4. La passe de vérification post-pipeline log `enrichment_complete` ou `enrichment_incomplete`.
5. Le graphe escouade affiche la dernière session même si un coéquipier n'a pas de score (LEFT JOIN).
6. **Aucune table de journal** ajoutée — ces calculs sont locaux et doivent simplement fonctionner.
7. Zéro régression : les tests existants passent.

### J.9 Prérequis

Aucun prérequis bloquant. Peut être implémentée indépendamment des autres phases. Recommandé comme **priorité haute** car c'est le fix du bug critique #1 du backlog (dernière session escouade invisible).

### J.10 Fichiers impactés

| Fichier | Modification |
|---------|-------------|
| `src/data/sync/engine.py` | Ajouter `shared_read_only` au `__init__` + `_verify_enrichment_completeness()` |
| `src/data/sync/_engine_connections.py` | `_get_shared_connection()` utilise `self._shared_read_only` |
| `src/data/sync/_engine_fanout.py` | `shared_read_only=True` + `_run_other_dominance()` + LUSR fanout |
| `src/analysis/_performance_squad.py` | `how="inner"` → `how="left"` dans `_join_perf_frames()` |

### J.11 Logs attendus

| Événement | Niveau | Message structuré |
|-----------|--------|-------------------|
| Vérification réussie | `info` | `event=enrichment_complete total=N` |
| Enrichissements manquants | `warning` | `event=enrichment_incomplete perf_missing=N sessions_missing=N total=N` |
| Fanout dominance | `info` | `fanout [GT]: dominance — N traité(s)` |
| Fanout LUSR | `info` | `fanout [GT]: N LUSR calculé(s)` |

### J.12 Tests ciblés

1. **Test R/O fanout** : créer un engine avec `shared_read_only=True` → vérifier que `_get_shared_connection()` ouvre en `read_only=True`.
2. **Test conflit éliminé** : ouvrir shared en R/O (simule Streamlit) + créer un engine fanout R/O → la connexion réussit (plus de conflit).
3. **Test vérification post-pipeline** : insérer des matchs sans `performance_score` → `_verify_enrichment_completeness` log `enrichment_incomplete`.
4. **Test LEFT JOIN** : `_join_perf_frames()` avec un joueur manquant → le match apparaît dans le résultat (pas filtré).
5. **Test fanout étendu** : vérifier que le fanout appelle dominance + LUSR pour chaque coéquipier.
6. **Test non-régression** : sync principal avec `shared_read_only=False` (défaut) → comportement identique à l'existant.

---

## 5) Stratégie de tests

### 5.0 Principes généraux

1. **Couverture branches** : ≥90% sur tous les fichiers touchés par ce plan. Vérifier via `python -m pytest --cov=src --cov-report=term-missing` filtré sur les fichiers concernés.
2. **Assertions de logs** : chaque test d'intégration doit vérifier la présence des messages de log attendus via `caplog` (pytest). Un comportement silencieux = test échoué.
3. **DuckDB en mémoire ou tmpdir** : tous les tests unitaires et d'intégration utilisent `:memory:` ou `tmp_path` pour l'isolation. Pas de test dépendant de données réelles.
4. **Mocking ciblé** : seul le réseau (API SPNKr) et le filesystem externe sont mockés. La logique DuckDB, les transactions et les context managers sont testés pour de vrai.
5. **Chaque `try/except` ajouté = 1 test `except`** : vérifier que le chemin d'erreur est exercé et produit le log/comportement attendu.
6. **Fichier de test dédié par phase** : `tests/test_sync_phase_a.py`, `tests/test_sync_phase_b.py`, etc. pour faciliter le rollback par lot.

## 5.1 Tests unitaires

1. Classification du résultat success/partial/failure.
2. Delta HEAD-first :
   - cas à jour (short-circuit)
   - cas nouveau match (charge existing_ids ensuite)
3. Matrice auth :
   - étape required sans token -> erreur attendue
   - étape optional sans token -> warning + continuation
4. Fraîcheur : `get_sync_metadata()` lit depuis `sync_meta WHERE key='last_sync_at'` (player DB locale), pas `meta.sync_meta` — retourne le timestamp écrit par `_save_sync_metadata()` pour chaque joueur.
5. `_load_existing_match_ids` : parité entre ancienne impl (3 requêtes) et nouvelle (1 SQL) sur 4 cas (match normal, 0-score sans awards, partial, complet).
6. `remaining` en mode full : 200 existants + 50 nouveaux → 50 traités.
7. `suppress(Exception)` remplacés : injection d'erreur CHECKPOINT → warning loggé (assert `caplog` contient le message).
8. Transaction par batch de 50 : crash simulé dans un batch → les batches précédents sont intacts, le batch interrompu est absent.
9. **Logging assertions** : chaque décision-clé (short-circuit, auth decision, skip, status) produit une ligne de log vérifiable.
10. **Test négatif fan-out** : erreur pendant le fan-out ne propage pas au résultat sync principal (isolation).
11. `_run_post_sync_pipeline` autonome : appelée directement avec `inserted_ids` mocké → chaque étape (perf, LUSR, fan-out) reçoit bien les ids sans passer par `_sync_internal`.
12. Ordre contraint post-sync : mock des 4 étapes + vérification de l'ordre d'appel via `call_args_list` (garantit que `_detach_shared_from_player_conn` précède `_run_lusr_post_sync`).
13. `_get_shared_connection()` avec `_shared_read_only=True` → `duckdb.connect(read_only=True)`.
14. `_verify_enrichment_completeness` : matchs sans `performance_score` → log `enrichment_incomplete`.
15. `_join_perf_frames` LEFT JOIN : joueur manquant → match conservé dans le résultat.
16. Fanout étendu : vérification que dominance + LUSR sont appelés pour chaque coéquipier.

## 5.2 Tests d'intégration

1. Sync multi-joueurs depuis UI : agrégation des statuts individuels.
2. Vérification invalidation cache uniquement sur écritures effectives.
3. Vérification fraîcheur "Mis à jour il y a XXX" pour tous les joueurs synchronisés.
4. Sync multi-joueurs : un seul `asyncio.run()` (pas N event loops).
5. Fan-out différé : st.rerun() survient avant le fan-out, enrichissement se déclenche au cycle suivant.
6. **Vérification logs bout-en-bout** : un sync complet (début → fin) doit produire au minimum : `sync_start`, `player_sync_start` ×N, `player_sync_end` ×N, `sync_end`. Si l'un manque = régression.
7. **Injection d'erreur DB** : simuler un crash DuckDB mid-write (close forced) → vérifier que le rollback transactionnel laisse la DB propre.
8. **Fanout R/O sous charge Streamlit** : ouvrir shared en R/O (simule Streamlit UI) + lancer fanout engine R/O → aucun conflit, enrichissements complets.
9. **Vérification post-sync bout-en-bout** : sync 2 joueurs avec matchs partagés → vérifier que les 2 joueurs ont `performance_score`, `session_id`, `dominance_flag` non-NULL pour les matchs communs.

## 5.3 Non-régression

1. Cas déjà à jour
2. Cas nouveaux matchs
3. Cas auth expirée
4. Cas un joueur OK, un joueur KO
5. Cas switch de joueur immédiat après sync global
6. Full sync avec matchs existants : le quota n'est plus gaspillé par les skips
7. Crash mid-batch (H) : les batches précédents (N×50 matchs) sont intacts, le batch en cours est rollbacké
8. **Couverture branches vérifiée** : `python -m pytest --cov=src/data/sync --cov=src/ui/sync.py --cov=src/ui/_sync_duckdb_ops.py --cov-fail-under=90` doit passer sur le code ajouté/modifié.
9. **Aucun `suppress(Exception)` résiduel** : `grep -r "suppress(Exception)" src/data/sync/ src/ui/sync.py src/ui/_sync_duckdb_ops.py` doit retourner 0 résultat.

---

## 6) Plan de rollout

## Lot 1 (faible risque)

- Phase E (logs + suppress cleanup + bare connect fix + nettoyage bitmasks E.5).
- Phase D (mtime/caches).

## Lot 2 (risque moyen)

- Phase A (statut/UX).

## Lot 3 (risque moyen/élevé)

- Phase B (delta HEAD-first + consolidation SQL + fix remaining).

## Lot 4 (risque moyen, métier sensible)

- Phase C (clarification auth token_scope any/player).

## Lot 5 (risque moyen)

- Phase F (fraîcheur multi-joueurs / indicateur "Mis à jour il y a XXX").

## Lot 6 (risque moyen, refonte structurelle)

- Phase G (event loop unique + fan-out non bloquant + dead code).

## Lot 7 (risque moyen/élevé, hardening avancé)

- Phase H (transactions explicites par batch de 50 matchs). Pré-requis : Lot 6 (Phase G). Gate de merge : benchmark < 5% overhead.

## Lot 8 (risque faible, dette structurelle)

- Phase I (extraction `_run_post_sync_pipeline` — réduction God function `_sync_internal`). Aucun prérequis bloquant ; recommandé après Lot 6 pour limiter les conflits sur `engine.py`.

## Lot 9 (risque faible/moyen, fix critique — PRIORITÉ HAUTE)

- Phase J (garantie d'enrichissement pour matchs partagés — fix root cause R/W→R/O + LEFT JOIN défensif + fanout étendu). Aucun prérequis bloquant. **Peut être implémenté en premier** car il corrige le bug critique #1 du backlog (dernière session escouade invisible) sans dépendre des autres phases.

---

## 7) Risques & mitigations

1. Risque UX : trop de warnings partiels.
Mitigation : seuil de synthèse + message compact + détails en logs.

2. Risque perf : optimisation delta mal placée.
Mitigation : feature flag temporaire pour HEAD-first si nécessaire.

3. Risque métier auth : confusion entre optionnel et requis.
Mitigation : matrice auth centralisée + tests explicites par étape.

4. Risque observabilité : bruit de logs.
Mitigation : niveaux adaptés (info décisionnel, debug détail).

5. Risque de cohérence fraîcheur : timestamp avancé sans écriture réelle.
Mitigation : mise à jour conditionnée au résultat joueur + tests échec partiel.

6. Risque event loop : `asyncio.run()` dans un contexte où une event loop tourne déjà (Streamlit).
Mitigation : `asyncio.run()` est appelé depuis le thread principal Streamlit (garanti sync, hors event loop). C'est le contexte maîtrisé — on le code pour ce contexte. **Pas de guard `is_running()` ni de fallback** : si le contexte change, c'est un bug à corriger en amont, pas à contourner. Test: vérifier que `asyncio.run()` fonctionne dans un thread non-async (unit test dédié).

7. Risque fan-out différé : perte du contexte fan-out si le process Streamlit redémarre entre le sync et le prochain cycle.
Mitigation : le flag `fanout_pending` est persisté dans `sync_meta` (player DB) avant le `st.rerun()`. Résiste au process restart. Si corrompue (cas extrême), le fan-out sera rattrapé au prochain sync du coéquipier concerné. Pas de queue complexe — juste un boolean en DB.

8. Risque transactions : overhead COMMIT par batch sur grands volumes.
Mitigation : batch de 50 matchs (pas par match). Benchmark obligatoire avant merge — gate de merge si overhead > 5% avec N=50. Monter à N=100 si nécessaire. DuckDB = COMMIT très léger (WAL mode, pas de fsync par COMMIT).

9. Risque Phase J : `shared_read_only=True` dans le fanout empêche d'écrire dans shared (si un futur enrichissement devait écrire dans shared).
Mitigation : les 5 enrichissements du fanout (perf, sessions, citations, dominance, LUSR) écrivent **exclusivement** dans la player DB locale — jamais dans shared. Si un futur enrichissement nécessite d'écrire dans shared, il devra être exécuté dans le pipeline principal (pas le fanout). Le paramètre `shared_read_only` encode cette contrainte architecturale explicitement.

---

## 8) Checklist d'exécution

### Implémentation

- [ ] Définir enum/statut de sync (A)
- [ ] Adapter rendu UI selon statut (A)
- [ ] Implémenter delta HEAD-first lazy existing_ids (B)
- [ ] Consolider `_load_existing_match_ids` en une seule requête SQL (B)
- [ ] Fix `remaining` en mode full (B)
- [ ] Annoter les endpoints API avec token_scope (any/player) (C)
- [ ] Logger token_resolved + player_gated_skip (C)
- [ ] Ajuster invalidation mtime/caches (D)
- [ ] Ajouter logs/KPIs structurés (E)
- [ ] Remplacer `suppress(Exception)` critiques par try/except + log (E)
- [ ] Fix connexion bare dans `_compute_dominance_post_sync` (E)
- [ ] Supprimer `BACKFILL_FLAGS["performance_scores"]` de `_compute_backfill_mask()` (E.5)
- [ ] Supprimer `"lusr"` et `"csr"` de `BACKFILL_FLAGS` (grep préalable + test de non-existence) (E.5)
- [ ] Corriger `get_sync_metadata()` dans `_diagnostic_repo.py` — lire depuis `sync_meta WHERE key='last_sync_at'` (player DB locale, pas `meta.sync_meta`) (F)
- [ ] Vérifier le rendu "Mis à jour il y a XXX" au switch de profil après sync global (F)
- [ ] Event loop unique dans `sync_all_players_duckdb` (G)
- [ ] Découpler fan-out du spinner UI + persister `fanout_pending` dans `sync_meta` avant `st.rerun()` (G)
- [ ] Ajouter commentaire inline "fan-out opportuniste, persisté en DB" dans la fonction qui pose le flag (G)
- [ ] Supprimer `_sync_duckdb_player` + `_run_duckdb_player_sync_async` (G)
- [ ] Transactions explicites par batch de 50 matchs dans shared (H) — repurposer `_maybe_batch_commit`
- [ ] Benchmark `tests/bench/test_transaction_overhead.py` N=50 — gate de merge (H)
- [ ] Extraire `_run_post_sync_pipeline()` depuis `_sync_internal` (I)
- [ ] Ajouter `shared_read_only` à `DuckDBSyncEngine.__init__` + `_get_shared_connection()` (J)
- [ ] Fanout avec `shared_read_only=True` (J)
- [ ] Étendre fanout : dominance + LUSR pour les coéquipiers (J)
- [ ] LEFT JOIN dans `_join_perf_frames()` (J)
- [ ] Passe de vérification `_verify_enrichment_completeness()` (J)
- [ ] **Mettre à jour `docs/SYNC_CALL_TREE.md`** pour refléter l'état final de la chaîne d'appels (à faire en dernier, après Lot 8)

### Qualité logging

- [ ] Chaque phase a ses événements de log structurés implémentés (cf. "Logs attendus" dans 11.x)
- [ ] Format clé=valeur respecté partout (pas de format libre)
- [ ] Niveaux de log corrects : INFO (décisions), WARNING (erreurs récupérables), ERROR (critiques)
- [ ] Un sync bout-en-bout produit les 5+ événements minimum (sync_start → sync_end)

### Qualité tests

- [ ] Fichiers de test dédiés par phase créés (`tests/test_sync_phase_{a..h}.py`)
- [ ] Chaque `try/except` ajouté a un test `except` correspondant
- [ ] Assertions `caplog` sur tous les événements de log définis
- [ ] Couverture branches ≥90% vérifiée (`pytest --cov`)
- [ ] Tests DuckDB en mémoire / tmpdir (pas de dépendance données réelles)
- [ ] Valider non-régression sur sync UI multi-joueurs
- [ ] Benchmark perf transactions (H) < 5% overhead

### Anti-workaround

- [ ] Zéro `suppress(Exception)` résiduel : `grep -r "suppress(Exception)" src/data/sync/ src/ui/_sync_duckdb_ops.py` = 0
- [ ] Zéro `except: pass` / `except Exception: pass` sans log
- [ ] Zéro guard `is_running()` / fallback async
- [ ] Zéro code mort commenté (supprimé, pas commenté)
- [ ] Pas de retry aveugle (chaque retry est borné + documenté)
- [ ] Pas de `return None` silencieux sur chemin d'erreur

---

## 9) Décision clé validée dans ce plan

Le refactor ne doit pas supprimer la logique de récupération partielle possible sans auth spécifique au joueur cible.

La bonne direction est :

- mieux distinguer les chemins d'auth
- mieux diagnostiquer pourquoi une donnée est ou non récupérée
- conserver les deux capacités métier existantes

---

## 10) Addendum demandé utilisateur (2026-03-24)

Point confirmé : en sync global, l'indicateur de fraîcheur doit être correct pour tous les joueurs synchronisés, pas seulement pour le joueur actif au moment du clic.

Ce besoin est intégré dans la Phase F et devient un critère de validation obligatoire.

---

## 11) Détail d'implémentation par phase (niveau exécutable)

Cette section détaille exactement quoi changer, dans quel ordre, et comment vérifier.

## 11.1 Phase A — Statut de sync (success / partial / failure)

### Cible technique

Unifier la sémantique de résultat entre moteur, agrégation multi-joueurs et UI.

### Fichiers pressentis

1. src/data/sync/models_sync.py
2. src/ui/sync.py
3. streamlit_app.py

### Work breakdown

1. Introduire une représentation explicite du statut global (enum ou champ status calculé).
2. Conserver la propriété success pour compatibilité, mais ajouter partial_success explicite.
3. Adapter le résumé multi-joueurs pour distinguer :
  - toutes les sync OK,
  - au moins une sync KO mais certaines OK,
  - toutes KO.
4. Adapter l'affichage sidebar pour afficher warning sur succès partiel.

### Règles métier à verrouiller

1. Si errors non vide et matches_inserted > 0 : partial_success.
2. Si errors non vide et matches_inserted == 0 : failure.
3. Si errors vide : success.

### Test cases détaillés

1. Resultat moteur : 3 cas unitaires sur status.
2. Agrégateur multi-joueurs :
  - 2 joueurs OK -> success
  - 1 OK / 1 KO -> partial_success
  - 2 KO -> failure
3. UI : mapping status -> success/warning/error.
4. **Test de log** : vérifier que `event=sync_end status=partial_success` apparaît dans `caplog` quand 1 joueur KO.
5. **Test négatif** : `matches_inserted=0` et `errors=[]` → `success` (pas `failure`).

### Logs attendus (Phase A)

```
event=player_sync_end gamertag=... status=success|partial_success|failure inserted=N errors=N
event=sync_end status=success|partial_success|failure ok=X ko=Y duration_s=...
```
Chaque test d'intégration Phase A doit asserter ces lignes via `caplog`.

---

## 11.2 Phase B — Delta HEAD-first + consolidation requêtes + fix remaining

### Cible technique

Éviter le coût de _load_existing_match_ids sur les no-op delta. Consolider les 3 requêtes séquentielles en une seule. Corriger le décomptage remaining en mode full.

### Fichiers pressentis

1. src/data/sync/engine.py
2. src/data/sync/_match_processing.py
3. src/data/sync/_engine_connections.py

### Work breakdown

1. Extraire un pré-check delta dédié :
  - latest_api_match_id
  - latest_db_match_id
  - décision short-circuit.
2. Déplacer le chargement existing_ids après ce pré-check.
3. Si short-circuit positif : terminer le sync sans charger existing_ids.
4. Journaliser la décision pour audit perf.
5. Refactorer `_load_existing_match_ids` : remplacer les 3 requêtes Python par une requête SQL unifiée avec JOIN/subquery (voir B.3).
6. Corriger `_process_matches` : `remaining` ne décrémente plus pour les matchs skippés en mode full.

### Pseudo-flux cible

1. if delta_mode:
2.   head_api = get_match_history(count=1)
3.   head_db = latest_match_in_db
4.   if head_api == head_db: return already_up_to_date
5. existing_ids = _load_existing_match_ids()   # une seule requête SQL
6. poursuivre le pipeline normal.

### Garde-fous

1. Si head_api absent ou erreur API : fallback pipeline normal (pas de short-circuit).
2. Ne jamais short-circuiter en mode full.
3. La requête SQL consolidée doit retourner exactement le même set de match_ids que les 3 requêtes actuelles (test de non-régression par comparaison).

### Changement `remaining` dans `_process_matches`

Avant :
```python
if match_id in existing_ids:
    result.matches_skipped += 1
    start += 1
    remaining -= 1  # ← consomme le quota
    continue
```

Après :
```python
if match_id in existing_ids:
    if delta_mode:
        logger.info("[DELTA] Match %s déjà connu — arrêt", match_id)
        delta_stop = True
        break
    result.matches_skipped += 1
    start += 1
    # remaining inchangé : skip ne consomme pas le quota
    continue
```

### Test cases détaillés

1. delta + head identique -> stop, existing_ids non chargé.
2. delta + head différent -> existing_ids chargé, sync continue.
3. delta + API head indisponible -> pas de crash, sync continue.
4. `_load_existing_match_ids` : test de parité entre ancienne impl (3 requêtes) et nouvelle (1 requête SQL) sur un jeu de données réel.
5. Full sync avec 200 matchs existants + 50 nouveaux : les 50 sont traités (remaining non consommé par les skips).
6. **Test log short-circuit** : `caplog` contient `event=delta_head_check short_circuit=true` quand head identique.
7. **Test log non-short-circuit** : `caplog` contient `event=delta_head_check short_circuit=false existing_ids_loaded=true` quand head diffère.
8. **Test négatif API head** : API retourne erreur → pas de crash, sync continue normalement (pas de fallback silencieux — le warning est loggé).
9. **Test parité SQL** : exécuter les 3 anciennes requêtes + la nouvelle sur les mêmes données → résultat identique (test de non-régression avant suppression).

### Logs attendus (Phase B)

```
event=delta_head_check gamertag=... short_circuit=true|false
event=existing_ids_loaded gamertag=... count=N source=sql_consolidated
event=remaining_tracking mode=full total=N processed=N skipped=N
```

---

## 11.3 Phase C — Auth explicite (token_scope any vs player)

### Cible technique

Rendre explicite le scope de token nécessaire par étape du pipeline sync, en s'appuyant sur la réalité de l'API Halo Infinite :
- **`token_scope="any"`** : un spartan_token global suffit (données de match, stats, film, assets).
- **`token_scope="player"`** : les tokens du joueur cible sont obligatoires (career rank, XP, spartan ID).

### Fichiers pressentis

1. `src/data/sync/api_client.py` — annoter chaque méthode API avec `token_scope`
2. `src/data/sync/_tokens.py` — hiérarchie de résolution de tokens (déjà implémentée, à documenter)
3. `src/data/sync/_career.py` — sync career rank (player-gated)
4. `src/data/sync/engine.py` — orchestration : séparer clairement les phases any-token vs player-token
5. `src/ui/profile_api.py` — customisation/spartan ID (player-gated)
6. `src/auth/provider.py` — obtention tokens (inchangé)

### Matrice endpoint complète (figée)

| Étape sync | Endpoint API | token_scope | Comportement sans token joueur |
|---|---|---|---|
| Historique matchs | `stats.get_match_history()` | `any` | OK avec token global |
| Stats d'un match | `stats.get_match_stats()` | `any` | OK avec token global |
| CSR/MMR | `skill.get_match_skill()` | `any` | OK avec token global |
| Events film | `film.read_highlight_events()` | `any` | OK avec token global |
| Film metadata | `discovery_ugc.get_film_by_match_id()` | `any` | OK avec token global |
| Film chunks | Azure blob download | aucun | URL pré-signée |
| GT→XUID | `profile.get_user_by_gamertag()` | `any` | OK avec token global |
| XUIDs→profiles | `profile.get_users_by_id()` | `any` | OK avec token global |
| Match count | `halostats.svc/.../matches/count` | `any` | OK avec token global |
| Assets | `discovery_ugc.get_*()` | `any` | OK avec token global |
| Career rank metadata | `gamecms_hacs.get_career_reward_track()` | `any` | OK avec token global |
| **Career rank + XP** | **`economy.svc/.../careerranks/...`** | **`player`** | **Skip + warning** |
| **Spartan ID** | **`economy.get_player_customization()`** | **`player`** | **Skip + warning** |

Les post-traitements locaux (perf_scores, sessions, citations, dominance, fan-out) ne font aucun appel API et ne nécessitent aucun token.

### Hiérarchie de résolution de tokens (`_tokens.py`)

| Priorité | Source | Résultat |
|---|---|---|
| 1 | `SPNKR_OAUTH_REFRESH_TOKEN_<GT>` (env var per-player) | token_scope=player OK |
| 2 | Cache sync_meta dans stats.duckdb (per-player, Device Code Flow) | token_scope=player OK |
| 3 | `SPNKR_OAUTH_REFRESH_TOKEN` (env var global) | token_scope=any OK, player SKIP |
| 4 | `SPNKR_SPARTAN_TOKEN` + `SPNKR_CLEARANCE_TOKEN` (manuels) | token_scope=any OK, player SKIP |

### Conséquence en sync multi-joueurs

En sync global (N joueurs), le scénario nominal est :
1. **Token global valide** → toutes les données de match de tous les joueurs sont récupérées (token_scope=any).
2. Pour chaque joueur individuellement :
   - Si token per-player disponible → career rank + spartan ID récupérés.
   - Si pas de token per-player → career rank + spartan ID skippés (warning), mais match data complète.
3. Le sync ne doit **jamais échouer** pour un joueur parce qu'il n'a pas de token per-player. Seules les données player-gated sont skippées.

### Work breakdown

1. **Annoter les méthodes API** dans `api_client.py` avec un docstring ou un commentaire `# token_scope: any|player` sur chaque méthode publique.
2. **Séparer les phases dans le engine** : le match processing (any-token) et le career/spartan sync (player-token) sont déjà distincts dans le code. S'assurer que c'est explicite et loggé.
3. **Logger le scope de token résolu** pour chaque joueur au début de son sync : `event=token_resolved gamertag=... scope=any|player source=env_per_player|db_cache|env_global|manual`.
4. **Logger les skips player-gated** : quand career rank ou spartan ID est skippé par manque de token per-player, logger un warning structuré (pas silencieux).
5. **Ne pas introduire de fallback** : si le token per-player n'est pas disponible, on skip. Pas de retry, pas de prompt UI, pas de fallback vers un token global pour un endpoint player-gated (ça ne marcherait pas de toute façon).

### Changements concrets (détail minimum)

#### api_client.py — annotations

```python
async def get_match_history(self, player, match_type, start, count):
    """Historique matchs. token_scope: any."""
    ...

async def get_career_rank_progression(self, xuid):
    """Career rank + XP. token_scope: player (tokens du joueur cible requis)."""
    ...
```

#### engine.py — log resolution

```python
# Au début de _sync_internal, après résolution tokens
player_tokens = get_tokens_for_player(self._gamertag)
has_player_scope = player_tokens is not None
logger.info(
    "event=token_resolved gamertag=%s scope=%s source=%s",
    self._gamertag,
    "player" if has_player_scope else "any",
    token_source,
)
```

#### _career.py — log skip explicite

```python
if player_tokens is None:
    logger.warning(
        "event=player_gated_skip gamertag=%s step=career_rank reason=no_player_token",
        self._gamertag,
    )
    return None  # Skip explicite, pas silencieux — le warning est le contrat
```

### Matrice de décision erreur (3 scénarios)

| Scénario | Token global | Token per-player | Match data | Career rank | Spartan ID |
|---|---|---|---|---|---|
| Full auth | ✅ | ✅ | ✅ sync | ✅ sync | ✅ sync |
| Global only | ✅ | ❌ | ✅ sync | ⚠️ skip + warning | ⚠️ skip + warning |
| No token | ❌ | ❌ | ❌ erreur bloquante | ❌ erreur bloquante | ❌ erreur bloquante |

### Test cases détaillés

1. **Token global + token per-player** → match data + career rank + spartan ID tous récupérés. Aucun warning.
2. **Token global seul, pas de per-player** → match data OK. Career rank et spartan ID skippés. `caplog` contient `event=player_gated_skip step=career_rank` et `event=player_gated_skip step=spartan_id`.
3. **Aucun token** → erreur bloquante **avant le sync**. Message actionnable (reconnexion Xbox).
4. **Test log token_resolved** : `caplog` contient `event=token_resolved scope=player` ou `scope=any` selon le cas.
5. **Test non-régression match data** : sync avec token global seul → même nombre de matchs insérés qu'avec token per-player (les matchs ne dépendent pas du token per-player).
6. **Test pas de fallback silencieux** : un endpoint player-gated sans token per-player ne retourne jamais de données partielles ou erronées — il retourne `None` avec warning.
7. **Test multi-joueurs** : 3 joueurs, token per-player pour 1 seul → match data pour 3, career rank pour 1 seul. Le status global = `success` (pas `partial_success` — le skip player-gated n'est pas une erreur).

### Logs attendus (Phase C)

```
event=token_resolved gamertag=... scope=any|player source=env_per_player|db_cache|env_global|manual
event=player_gated_skip gamertag=... step=career_rank|spartan_id reason=no_player_token
event=no_token_available gamertag=... remediation=device_code_flow
```

### Vérification anti-workaround (Phase C)

Après implémentation, confirmer :
- [ ] Pas de fallback d'un endpoint player-gated vers un token global (ça ne marchera pas)
- [ ] Pas de retry si token per-player absent (ce n'est pas un problème transitoire)
- [ ] Pas de prompt UI pour saisir un token pendant le sync
- [ ] Les skips player-gated sont des warnings, pas des erreurs (le sync match data est complet)

---

## 11.4 Phase D — Invalidation cache et mtime conditionnelle

### Cible technique

Ne rafraîchir que ce qui a réellement été modifié.

### Fichiers pressentis

1. src/ui/sync.py
2. src/app/cache_control.py

### Work breakdown

1. Conditionner os.utime et invalidation locale au succès joueur.
2. Conserver l'invalidation globale finale mais l'associer au statut global.
3. Ne pas avancer de marqueur de fraîcheur sur un joueur KO.

### Test cases détaillés

1. Joueur KO -> pas d'utime joueur.
2. Joueur OK -> utime présent.
3. Mixte OK/KO -> seuls les OK sont invalidés/timestampés.
4. **Test log invalidation** : `caplog` contient `event=cache_invalidated gamertag=... reason=sync_success` pour les OK.
5. **Test log skip invalidation** : `caplog` contient `event=cache_skip gamertag=... reason=sync_failed` pour les KO.
6. **Test négatif** : un joueur KO ne touche ni `mtime` ni `cache` — vérifier que les fichiers n'ont pas été modifiés (stat avant/après).

### Logs attendus (Phase D)

```
event=cache_invalidated gamertag=... reason=sync_success
event=cache_skip gamertag=... reason=sync_failed|no_new_data
```

---

## 11.5 Phase E — Observabilité, métriques, diagnostic terrain et hygiène

### Cible technique

Rendre tout diagnostic possible depuis les logs sans reproduire localement. Éliminer les masquages d'erreurs silencieux.

### Fichiers pressentis

1. src/ui/sync.py
2. src/data/sync/engine.py
3. src/data/sync/_match_processing.py
4. src/ui/_sync_indicator.py
5. src/data/sync/_engine_connections.py
6. src/ui/_sync_duckdb_ops.py

### Schéma de logs conseillé (clé=valeur)

1. event=sync_start sync_scope=all_players mode=delta players_count=N
2. event=player_sync_start gamertag=... token_scope=any|player
3. event=delta_head_check gamertag=... short_circuit=true|false
4. event=player_sync_end gamertag=... status=... inserted=... errors=...
5. event=sync_end status=... duration_s=... ok=X ko=Y partial=Z

### KPIs exploitables

1. p50/p95 durée sync par joueur
2. taux short-circuit delta
3. taux partial_success
4. ratio KO auth / KO non-auth

### Work breakdown supplémentaire (audit code)

#### E.3a — Remplacement `suppress(Exception)` → `try/except + log`

Cibles classées par criticité :

1. **Critique** (`warning`) : `close()` dans `ConnectionMixin` (CHECKPOINT), `_activate_sync_mode()` dans `_sync_duckdb_ops.py` (release connexions).
2. **Important** (`warning`) : `_run_post_sync_compute()`, `_run_lusr_post_sync()`, `_compute_dominance_post_sync()` (fermetures shared_connection).
3. **Acceptable** (`debug`) : `_update_match_participant_bits()`, `_upsert_event_aliases()` (erreurs non structurelles).

Chaque remplacement suit le pattern :
```python
# Avant
with contextlib.suppress(Exception):
    resource.close()

# Après
try:
    resource.close()
except Exception:
    logger.warning("Fermeture %s échouée", resource_name, exc_info=True)
```

#### E.3b — Fix connexion bare `_compute_dominance_post_sync`

Remplacer le `duckdb.connect() + try/finally` par un context manager `with`.

### Test cases ajoutés

1. Injection d'erreur sur `CHECKPOINT` → vérifier que le warning est loggé.
2. Injection d'erreur sur `release_all_db_connections` → vérifier que le sync continue mais logge.
3. **Test zéro suppress résiduel** : `grep -c "suppress(Exception)"` sur les fichiers de la chaîne sync = 0 après cette phase.
4. **Test context manager** : `_compute_dominance_post_sync` utilise `with` — simuler une exception dans le body et vérifier que la connexion est bien fermée (`__exit__` appelé).
5. **Test exhaustivité logs** : un sync complet de bout en bout doit produire les 5 événements structurés minimum : `sync_start`, `player_sync_start`, `player_sync_end`, `sync_end` + au moins un événement décisionnel (`delta_head_check` ou `token_resolved`).

### Vérification anti-workaround (Phase E)

Après implémentation, confirmer :
- [ ] Aucun `suppress(Exception)` résiduel dans `src/data/sync/` et `src/ui/_sync_duckdb_ops.py`
- [ ] Aucun `except: pass` ni `except Exception: pass` (sans log)
- [ ] Chaque `except Exception as e:` a un `logger.warning(...)` ou `logger.error(...)` avec `exc_info=True`
- [ ] Aucun `return None` silencieux dans un chemin d'erreur

---

## 11.6 Phase F — Indicateur “Mis à jour il y a XXX” multi-joueurs

### Cible technique

Assurer une fraîcheur juste et homogène pour tous les joueurs synchronisés, même non actifs dans l'UI au moment du sync.

### Fichiers pressentis

1. src/ui/_sync_indicator.py
2. src/ui/sync.py
3. src/data/sync/engine.py
4. src/data/repositories/_diagnostic_repo.py (lecture fraîcheur)

### Work breakdown détaillé

1. Définir la source canonique de last_sync_at par joueur.
2. Documenter la hiérarchie de fallback (si legacy).
3. Mettre à jour la boucle sync global pour persister last_sync_at joueur par joueur après succès.
4. Faire lire l'indicateur UI exclusivement via la source canonique.
5. Vérifier au switch de profil que la fraîcheur affichée suit bien la DB du profil sélectionné.

### Cas limites à couvrir

1. Sync global interrompu au milieu.
2. Joueur sans match nouveau mais sync réussie (freshness doit quand même être cohérente selon règle métier retenue).
3. Joueur absent/invalide dans db_profiles.
4. Horodatages avec timezone incohérente.

### Test cases détaillés

1. 3 joueurs, 3 succès -> 3 timestamps mis à jour.
2. 3 joueurs, 2 succès / 1 échec -> seulement 2 timestamps mis à jour.
3. Switch joueur immédiat après sync -> “Mis à jour il y a ...” correct pour chacun.
---

## 11.7 Phase G — Consolidation runtime async + fan-out + dead code

### Cible technique

Éliminer le overhead de N event loops, découpler le fan-out de l'UX, et supprimer le code mort.

### Fichiers pressentis

1. src/ui/sync.py (`sync_all_players_duckdb`, `_sync_all_players_loop`)
2. src/ui/_sync_duckdb_ops.py (`sync_player_duckdb`, `_sync_duckdb_player`, `_run_duckdb_player_sync_async`)
3. src/data/sync/engine.py (`_enrich_other_registered_players`)
4. src/data/sync/_engine_fanout.py
5. streamlit_app.py (déclenchement fan-out différé)

### Work breakdown

#### G.1 — Event loop unique

1. Convertir `_sync_all_players_loop` en `async def _sync_all_players_loop_async`.
2. Remplacer les appels `sync_player_duckdb()` (sync) par `await sync_player_duckdb_async()`.
3. Placer le seul `asyncio.run()` dans `sync_all_players_duckdb()`.
4. Supprimer le wrapper sync `sync_player_duckdb()` s'il n'est plus appelé ailleurs, sinon le garder comme convenience wrapper.

#### G.2 — Fan-out non bloquant (Option A : session_state marker)

1. Dans `_sync_internal()`, ne plus appeler `_enrich_other_registered_players()` directement.
2. Retourner `inserted_match_ids` dans le `SyncResult`.
3. Côté `streamlit_app.py`, après le `st.rerun()`, vérifier un marqueur `session_state["_fanout_pending"]` :
   - Si présent : lancer le fan-out en arrière-plan (thread ou executor).
   - Supprimer le marqueur une fois lancé.
4. Le fan-out écrit directement dans les player DBs (pas d'interaction shared R/W).

**Compatibilité Phase C** : aucun impact. Le fan-out ne fait aucun appel API — pas de token nécessaire. Les post-traitements locaux (perf_scores, sessions, citations) sont des calculs purs sur les DBs joueur, sans interaction avec la matrice token_scope définie en Phase C.

#### G.3 — Suppression code mort

1. Confirmer par grep que `_sync_duckdb_player` et `_run_duckdb_player_sync_async` n'ont aucun caller actif.
2. Supprimer les deux fonctions.
3. Retirer les re-exports correspondants de `src/ui/sync.py __all__`.

### Test cases détaillés

1. Sync 3 joueurs : un seul `asyncio.run()` observé (pas 3 event loops créées).
2. Fan-out : après `st.rerun()`, le fan-out se déclenche au cycle suivant sans bloquer le rendu.
3. Fan-out en erreur : n'affecte pas le résultat sync principal.
4. `_sync_duckdb_player` absent du codebase après nettoyage.
5. Aucune régression sur le flux auth : les tokens sont toujours évalués par joueur dans la boucle async.
6. **Test log event loop** : `caplog` contient `event=async_loop_start players=N` une seule fois (pas N fois).
7. **Test log fan-out** : `caplog` contient `event=fanout_deferred match_ids=N` quand le fan-out est déféré.
8. **Test fan-out isolé** : erreur dans le fan-out → le résultat sync principal reste `success`. Vérifier que `event=fanout_error` est loggé (pas silencieux).
9. **Test grep dead code** : `grep -r "_sync_duckdb_player\|_run_duckdb_player_sync_async" --include="*.py" src/` retourne 0 résultat après cleanup.
10. **Test anti-workaround** : pas de `asyncio.get_event_loop().is_running()` dans le code. Vérifier par grep.

### Logs attendus (Phase G)

```
event=async_loop_start players=N mode=delta|full
event=async_loop_end duration_s=... status=success|partial_success|failure
event=fanout_deferred gamertag=... match_ids=N
event=fanout_start gamertag=... targets=N
event=fanout_complete gamertag=... targets=N duration_s=...
event=fanout_error gamertag=... error=...
```

### Vérification anti-workaround (Phase G)

Après implémentation, confirmer :
- [ ] Pas de guard `is_running()` ni de fallback async
- [ ] Pas de `try: asyncio.run() except: fallback()`
- [ ] Le fan-out différé n'utilise pas de TTL ni de mécanisme de queue complexe
- [ ] Le code mort est supprimé (pas commenté, pas marqué deprecated avec date lointaine)

---

## 11.8 Phase H — Intégrité transactionnelle

### Cible technique

Garantir qu'un crash ne laisse pas de données partielles dans shared_matches.duckdb.

### Fichiers pressentis

1. src/data/sync/_match_processing.py (`_process_single_match`, `_process_new_match`, `_process_known_match`)
2. src/data/sync/_match_processing_helpers.py (`_maybe_batch_commit`)
3. src/data/sync/_shared_writes.py

### Work breakdown

1. Ouvrir `BEGIN TRANSACTION` sur `shared_conn` au début du traitement d'un batch de 50 matchs.
2. Repurposer `_maybe_batch_commit` : à chaque `count % batch_commit_size == 0`, exécuter `COMMIT` + `BEGIN TRANSACTION` (nouveau batch).
3. En cas d'exception dans un match : `ROLLBACK` du batch en cours → aucune trace partielle pour les matchs du batch interrompu. Les batches précédents (déjà committés) sont préservés.
4. `COMMIT` final après le dernier batch (même si incomplet — ex: 223 matchs avec batch_size=50 → 4 batches complets + 1 batch de 23).
5. Vérifier que `_shared_db_lock` (asyncio.Lock) protège toujours la section critique (une seule transaction à la fois).
6. Ajouter un commentaire inline documentant le trade-off : "un crash dans un batch perd au max 50 matchs — re-insérés au prochain sync via `_load_existing_match_ids`".

### Interaction avec les autres phases

- **Phase B** : la transaction par batch est compatible avec le lazy-load de `existing_ids` (les IDs ne sont lus qu'avant la boucle, pas pendant une transaction).
- **Phase C** : aucune interaction — l'auth n'est pas touchée par les transactions DuckDB. Les transactions portent uniquement sur les écritures post-fetch.
- **Phase G** : l'event loop unique simplifie la gestion du lock asyncio (un seul contexte d'exécution).

### Garde-fous

1. DuckDB = un seul writer actif → pas de risque de deadlock entre transactions.
2. Les écritures dans la player DB (enrichments) sont HORS de la transaction shared → un rollback shared n'affecte pas les enrichments déjà écrits. Acceptable : `_load_existing_match_ids` vérifie les deux côtés et re-traitera les matchs orphelins.
3. Monitoring : logger le nombre de rollbacks par sync pour détecter les patterns d'erreur.

### Test cases détaillés

1. Injection d'erreur dans `_insert_shared_participants` dans le batch 2 → les matchs du batch 1 sont en DB, les matchs du batch 2 sont rollbackés.
2. Crash simulé au milieu d'un batch → les batches précédents sont intacts, le batch en cours est absent.
3. **Benchmark obligatoire (gate de merge)** : `tests/bench/test_transaction_overhead.py`, N=50 matchs `:memory:`, 10 runs, overhead médian ≤ 5%. Si > 5%, passer à N=100 et re-benchmarker.
4. **Test log transaction** : `caplog` contient `event=batch_commit count=50` à chaque commit de batch.
5. **Test log rollback** : `caplog` contient `event=batch_rollback count=N error=...` quand un batch échoue.
6. **Test compteur rollback** : `event=sync_summary rollbacks=N` en fin de sync pour monitoring.
7. **Test intégrité cross-DB** : après rollback d'un batch shared, la player DB n'a pas d'enrichment orphelin pour les match_ids du batch (ou ils sont détectés par `_load_existing_match_ids` pour re-traitement).
8. **Test batch final incomplet** : 123 matchs avec batch_size=50 → 2 batches complets committés + 1 batch de 23 committé en fin de boucle.

### Logs attendus (Phase H)

```
event=batch_begin batch=1
event=batch_commit batch=1 count=50 duration_ms=...
event=batch_rollback batch=2 count=23 error=...
event=sync_summary total=N committed=N rollbacks=N
```

### Vérification anti-workaround (Phase H)

Après implémentation, confirmer :
- [ ] `_maybe_batch_commit` repurposé (COMMIT+BEGIN), pas supprimé ni dupliqué
- [ ] Pas de double-commit (pas de `commit()` Python en plus du COMMIT SQL)
- [ ] Le rollback est un vrai ROLLBACK SQL, pas un "cleanup best-effort" en Python
- [ ] Benchmark `tests/bench/test_transaction_overhead.py` exécuté et résultat dans la PR
---

## 12) Plan de livraison en tickets (prêt sprint)

## Ticket T1 — Statut sync explicite

1. Scope : Phase A
2. Risque : moyen
3. Sortie attendue : status success/partial/failure visible dans UI + tests.
4. **Critère logging** : les 2 événements `player_sync_end` et `sync_end` sont testés via `caplog`.
5. **Critère tests** : ≥5 tests unitaires (3 status moteur + 3 agrégation multi-joueurs + 1 log assert).

## Ticket T2 — Delta HEAD-first

1. Scope : Phase B
2. Risque : moyen/élevé
3. Sortie attendue : no-op delta plus rapide + logs short-circuit.
4. **Critère logging** : `delta_head_check`, `existing_ids_loaded`, `remaining_tracking` testés.
5. **Critère tests** : ≥9 tests (5 delta + parité SQL + remaining + 2 logs).
6. **Anti-workaround** : si API head indisponible, pas de retry — on continue le flux normal avec log warning.

## Ticket T3 — Auth mode explicite

1. Scope : Phase C
2. Risque : moyen
3. Sortie attendue : token_scope any/player formalisé + annotations endpoints + tests.
4. **Critère logging** : `token_resolved` et `player_gated_skip` par joueur testés.
5. **Critère tests** : ≥7 tests (3 scénarios token + 2 logs + 1 non-régression match data + 1 multi-joueurs).
6. **Anti-workaround** : pas de fallback token global pour un endpoint player-gated. Pas de retry si token per-player absent.

## Ticket T4 — Cache/mtime conditionnels

1. Scope : Phase D + E partiel
2. Risque : faible
3. Sortie attendue : invalidation uniquement quand pertinent + logs.
4. **Critère logging** : `cache_invalidated` et `cache_skip` testés.
5. **Critère tests** : ≥6 tests (3 invalidation + 2 logs + 1 négatif).
6. **Anti-workaround** : zéro `suppress(Exception)` résiduel dans la chaîne sync.

## Ticket T5 — Freshness multi-joueurs

1. Scope : Phase F
2. Risque : moyen
3. Sortie attendue : indicateur “Mis à jour il y a XXX” correct pour tous les profils synchronisés.
4. **Critère logging** : `freshness_updated` et `freshness_skip` testés.
5. **Critère tests** : ≥7 tests.

## Ticket T6 — Consolidation runtime async + dead code

1. Scope : Phase G
2. Risque : moyen
3. Sortie attendue : event loop unique + fan-out non bloquant + `_sync_duckdb_player` supprimé.
4. **Critère logging** : `async_loop_start`, `fanout_deferred`, `fanout_complete` testés.
5. **Critère tests** : ≥10 tests.
6. **Anti-workaround** : zéro guard `is_running()`, zéro code mort commenté.

## Ticket T7 — Intégrité transactionnelle

1. Scope : Phase H
2. Risque : moyen/élevé
3. Prérequis : T6 (Phase G)
4. Sortie attendue : transaction explicite par batch de 50 matchs + `_maybe_batch_commit` repurposé (COMMIT+BEGIN) + benchmark < 5% overhead (gate de merge). Monter à N=100 si overhead > 5% avec N=50.
5. **Critère logging** : `transaction_commit`, `transaction_rollback`, `sync_summary` testés.
6. **Critère tests** : ≥8 tests.
7. **Anti-workaround** : le rollback est un vrai ROLLBACK SQL, pas un cleanup Python.

## Ticket T8 — Garantie enrichissement matchs partagés (Phase J)

1. Scope : Phase J
2. Risque : faible/moyen
3. **Priorité : HAUTE** — corrige le bug critique #1 du backlog (dernière session escouade invisible).
4. Prérequis : aucun (implémentable indépendamment)
5. Sortie attendue : fanout R/O (plus de conflit shared), fanout étendu (5 enrichissements), LEFT JOIN défensif, vérification post-pipeline.
6. **Critère logging** : `enrichment_complete` / `enrichment_incomplete` testés + fanout dominance/LUSR.
7. **Critère tests** : ≥6 tests (R/O engine, conflit éliminé, verification, LEFT JOIN, fanout étendu, non-régression).
8. **Anti-workaround** : pas de table de journal — fix la cause racine (connexion R/O au lieu de R/W).

---

## 13) Critères Go/No-Go avant merge

### Critères fonctionnels

1. Tous les tests unitaires nouveaux passent.
2. Tests d'intégration multi-joueurs passent.
3. Cas partiel (1 OK / 1 KO) validé manuellement en UI.
4. Freshness valide sur changement de profil après sync global.
5. Aucune régression auth (token_scope any/player conservé, match data toujours récupérable avec token global seul).
6. Fan-out ne bloque plus le spinner UI.
7. Benchmark transactions Phase H : overhead < 5%.

### Critères qualité (bloquants)

8. **Couverture branches ≥90%** sur tous les fichiers touchés (vérifié par `pytest --cov`).
9. **Zéro `suppress(Exception)`** dans la chaîne sync (vérifié par grep automatisé).
10. **Zéro `except: pass`** ni `except Exception: pass` sans log.
11. **Chaque `try/except` ajouté a un test** qui exerce le chemin `except`.
12. **Tous les événements de log structurés** définis dans les sections "Logs attendus" sont assertés dans les tests.
13. **Zéro workaround** : pas de guard `is_running()`, pas de retry aveugle, pas de fallback silencieux, pas de code "au cas où" non testé.
14. **Zéro code mort** : les fonctions supprimées ne sont plus référencées (vérifié par grep).
15. **Chaque événement décisionnel** (skip, short-circuit, auth, rollback, fan-out) est traçable dans les logs d'un sync réel.
16. **`docs/SYNC_CALL_TREE.md` à jour** : l'arbre d'appels reflète la chaîne finale (event loop unique, `_run_post_sync_pipeline`, fan-out découplé). Vérifié manuellement avant merge.

No-Go si l'un de ces points échoue.

---

## 14) Procédure de rollback

1. Garder les changements phase par phase (un commit par phase ou sous-phase critique).
2. En cas de régression : revert du ticket concerné uniquement.
3. Priorité rollback :
  - d'abord Phase H si corruption/incohérence transactionnelle,
  - puis Phase G si régression event loop ou fan-out,
  - puis Phase B si perf/comportement delta anormal,
  - puis Phase C si ambiguïté auth,
  - puis Phase F si incohérence de fraîcheur.
4. Maintenir les logs ajoutés (Phase E) même en rollback partiel pour aider le diagnostic.
5. Le suppress(Exception) cleanup (Phase E) et le bare connect fix n'ont aucun risque de rollback — ce sont des corrections unilatérales.
6. **Règle de rollback tests** : si un lot est revert, ses fichiers de test (`tests/test_sync_phase_X.py`) sont conservés en état `xfail` (pas supprimés) pour documenter la régression et faciliter la réintégration.
7. **Règle de rollback logs** : les événements de log ajoutés par une phase revertée sont conservés (jamais rollbackés). Les logs ne cassent rien et aident toujours le diagnostic.

---

## 15) Règles transversales rappelées

Ces règles s'appliquent à TOUTES les phases sans exception :

1. **Pas de `suppress(Exception)`** — utiliser `try/except Exception as e: logger.warning(...)` avec `exc_info=True`.
2. **Pas de `return None` silencieux** sur un chemin d'erreur — logger et propager ou retourner un objet d'erreur explicite.
3. **Pas de retry sans raison** — si une op doit être retryée, documenter pourquoi (ex: contention DuckDB) et borner à max 1 retry.
4. **Chaque branche `except`** doit être exercée par un test.
5. **Chaque décision** (skip, short-circuit, fallback, auth, invalidation) doit produire une ligne de log structurée.
6. **Mocking minimal** : tester le vrai code DuckDB, mocker uniquement le réseau.
7. **Nommage prédictif** : les événements de log utilisent le format `event=xxx key=value` sans format libre.

---

## 16) Bugs résolus par ce plan + backfill post-déploiement

### 16.1 Symptômes corrigés par phase

| Symptôme observé dans l'app | Phase | Cause racine |
|---|---|---|
| Dernière session escouade **invisible** dans le graphe | J | Conflit R/W + INNER JOIN : si 1 coéquipier n'a pas de score, tout le match disparaît |
| `performance_score` NULL pour les matchs récents | J.3 | `_get_shared_connection()` tente toujours R/W → conflit avec Streamlit R/O → retourne `None` silencieusement |
| `dominance_flag` NULL pour les matchs récents | J.3 | Même cause (connexion `shared_conn` passée au caller, issue du même `_get_shared_connection()`) |
| LUSR manquant dans `match_skill_rank` | J.3 | Même cause |
| Coéquipiers sans `performance_score` / dominance / LUSR après sync | J.4 + J.5 | Fanout crée un engine R/W → même conflit + dominance/LUSR non inclus dans le fanout |
| Sync affiché "succès" alors qu'il y a des erreurs d'enrichissement | A | Pas d'état `partial_success` — toute sortie sans exception levée = vert |
| Quota full sync gaspillé sur des matchs déjà en DB | B.4 | `remaining` décrémenté sur les matchs skippés au lieu des matchs effectivement traités |
| Indicateur "Mis à jour il y a XXX" désynchronisé en multi-joueurs | F | Pas de source canonique `last_sync_at` par joueur — l'UI lit une valeur stale |

### 16.2 Ce qui N'est PAS cassé par le bug actuel

| Enrichissement | Impact du bug | Raison |
|---|:---:|---|
| Sessions (`session_id`) | ❌ Non touché | Ouvre sa propre connexion R/O fraîche, indépendamment de `_get_shared_connection()` |
| Citations (`match_citations`) | ❌ Non touché | Même raison |
| Weapon kills | ❌ Non touché | Optionnel (film-dépendant), connexion gérée séparément |

### 16.3 Backfill post-déploiement

**Qui est concerné ?**

Tous les joueurs ayant fait un sync depuis l'UI Streamlit **pendant que le dashboard était ouvert** — ce qui est le cas nominal en usage quotidien. En pratique : **tous les joueurs enregistrés dans `db_profiles.json`** ont possiblement des entrées NULL dans les colonnes ci-dessous.

#### Diagnostic avant backfill (requêtes DuckDB par joueur)

```sql
-- Matchs sans performance_score
SELECT COUNT(*) FROM player_match_enrichment WHERE performance_score IS NULL;

-- Matchs sans dominance_flag (si la colonne existe)
SELECT COUNT(*) FROM player_match_enrichment WHERE dominance_flag IS NULL;

-- Matchs sans LUSR (en comparant avec les matchs que l'on aurait dû avoir)
SELECT COUNT(*) FROM player_match_enrichment e
LEFT JOIN match_skill_rank r ON r.match_id = e.match_id
WHERE r.match_id IS NULL;
```

#### Commandes de backfill

À lancer **depuis la CLI** (pas depuis l'UI) pour éviter tout conflit de handle tant que Phase J n'est pas déployée. Après déploiement de Phase J, peut être lancé depuis l'UI sans risque.

```bash
# Cas standard — recalcule uniquement les entrées NULL (ne réécrase pas ce qui existe)
python scripts/backfill_data.py --all --performance-scores --dominance --lusr

# Si on suspecte des valeurs incorrectes (pas seulement NULL) — force le recalcul complet
python scripts/backfill_data.py --all --force-performance-scores --force-dominance --force-lusr

# Pour un seul joueur (diagnostic ciblé)
python scripts/backfill_data.py --player <Gamertag> --performance-scores --dominance --lusr
```

**Sessions et citations** : pas de backfill nécessaire (enrichissements non touchés par le bug).

#### Ordre recommandé

1. **Déployer Phase J** en premier (fix root cause) — sinon le backfill lancé depuis l'UI souffrira du même bug.  
2. **Lancer le backfill CLI** (`--all --performance-scores --dominance --lusr`) pour rattraper le gap historical.
3. **Vérifier** avec les requêtes de diagnostic ci-dessus que les NULL ont disparu.
4. **Optionnel** : vérifier que le graphe escouade affiche bien la dernière session après backfill.
