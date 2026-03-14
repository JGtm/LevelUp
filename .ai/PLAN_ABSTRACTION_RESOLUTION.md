# PLAN v6 — Couche d'Abstraction Complète pour la Résolution d'IDs

> **Version** : v6
> **Branche** : `refactor/id-resolution-cleanup` (créée à partir de `analysis/weapon-parser-rewrite` = v5.7)
> Créé le 2026-03-14 · Mis à jour le 2026-03-14.
> Couvre : cascade gamertag, noms d'assets, paires killer/victim, outcomes, médailles.

### Objectifs v6

1. **Centraliser** toute résolution ID → nom affiché via 3 vues SQL + fonctions Python
2. **Détecter les incohérences** : un même XUID affichant 2 gamertags différents selon la page, un map_name stale qui ne correspond plus à metadata
3. **Éliminer les redondances** : mêmes colonnes dupliquées dans 3-5 tables, mêmes COALESCE répétés dans 33+ fichiers
4. **Garantir un point unique de modification** : changer une source = modifier 1 vue SQL, pas 35 fichiers

---

## Problème Fondamental

Aujourd'hui, **les consommateurs (UI, analysis, scripts) lisent directement** les colonnes
résolues (`gamertag`, `map_name`, `killer_gamertag`, etc.) depuis les tables DuckDB brutes.
Les mêmes noms sont dupliqués dans 3 à 5 tables différentes, avec des qualités variables
et des risques de stale data.

### Audit quantitatif (2026-03-14, révisé par audit complémentaire)

| Domaine | Colonnes dupliquées | Tables sources | Fichiers impactés | Emplacements |
|---------|:-------------------:|:--------------:|:-----------------:|:------------:|
| **Gamertags** | `gamertag` | 3 (`xuid_aliases`, `match_participants`, `highlight_events`) | 33 | 64+ |
| **Noms assets** | `map_name`, `playlist_name`, `pair_name`, `game_variant_name` | 2 (`match_registry`, `metadata.duckdb`) | 35+ | 160+ |
| **Paires K/V** | `killer_gamertag`, `victim_gamertag` | 1 (`killer_victim_pairs`) + résolution | 14 prod + 10 tests | 82+ |
| **Outcomes** | n/a (pas de duplication de colonne) | 3 fonctions fragmentées | 5 | 8 |
| **Médailles** | n/a | 0 helper (manquant) | — | — |
| **TOTAL** | | | **~85 fichiers** | **~320 emplacements** |

**Conséquence** : changer une source de vérité = modifier des dizaines de fichiers.

### Objectif

Introduire une **couche d'abstraction unique** (vues SQL + fonctions Python) pour que :
1. **Tous les consommateurs** passent par un seul point de résolution
2. **Changer la source** = modifier 1 vue SQL ou 1 fonction Python
3. **Supprimer une colonne** dupliquée = modifier la vue → aucun impact consommateur

---

## Architecture Cible

```
┌───────────────────────────────────────────────────────────────┐
│                       CONSOMMATEURS                           │
│   Pages UI · Analysis · Scripts · Visualisation · Tests       │
└────────────────────────┬──────────────────────────────────────┘
                         │ ne lisent JAMAIS les tables brutes
                         │ directement pour des noms résolus
              ╔══════════╧════════════════════╗
              ║   COUCHE D'ABSTRACTION        ║
              ╠══════════════════════════════ ═╣
              ║                                ║
              ║  VUES SQL (dans shared.duckdb) ║
              ║  ┌─────────────────────────┐   ║
              ║  │ v_gamertag_lookup       │   ║  xuid → gamertag courant
              ║  │ v_match_full            │   ║  match_registry + noms résolus
              ║  │ v_killer_victim_full    │   ║  paires + gamertags résolus
              ║  │ mv_player_matches (MV)  │   ║  vue matérialisée joueur (existante)
              ║  └─────────────────────────┘   ║
              ║                                ║
              ║  PYTHON (DuckDBRepository)     ║
              ║  ┌─────────────────────────┐   ║
              ║  │ GamertagResolverMixin   │   ║  resolve_gamertag(), load_match_player_gamertags()
              ║  │ MetadataResolver        │   ║  resolve("map", id), resolve("playlist", id)
              ║  │ resolve_medal_name()    │   ║  medal_name_id → nom
              ║  │ resolve_weapon_display()│   ║  weapon_id → nom (existe déjà ✅)
              ║  │ get_outcome_map(lang)   │   ║  outcome_code → label (existe déjà ✅)
              ║  └─────────────────────────┘   ║
              ╚══════════╤═════════════════════╝
                         │
         ┌───────────────┼──────────────────────────┐
         ▼               ▼                          ▼
   ┌──────────┐   ┌──────────────────┐       ┌──────────────┐
   │ xuid_    │   │ match_registry   │       │ metadata.    │
   │ aliases  │   │ match_participants│      │ duckdb       │
   │ (vérité  │   │ killer_victim_   │       │ (référentiel │
   │ gamertag)│   │ pairs            │       │  maps, medal │
   └──────────┘   │ highlight_events │       │  playlists)  │
                  └──────────────────┘       └──────────────┘
                  shared_matches.duckdb
```

### Principe directeur

> **Les tables stockent des IDs. Les vues résolvent les noms.**
>
> Aucun consommateur ne doit lire `match_participants.gamertag` ou
> `match_registry.map_name` directement. Il passe par une vue qui
> fait le COALESCE / JOIN avec la source de vérité.
>
> Les colonnes `*_name` et `*_gamertag` dénormalisées dans les tables
> sont un **cache de confort** — elles peuvent être supprimées à terme
> sans casser aucun consommateur.

---

## Volet A — Gamertags (XUID → nom affiché)

### A.0 Audit de surface (révisé)

| Catégorie | Fichiers | Emplacements |
|-----------|:--------:|:------------:|
| SQL direct `match_participants.gamertag` | 8 prod | 13 requêtes |
| SQL direct `highlight_events.gamertag` | 5 prod | 11 requêtes |
| SQL direct `xuid_aliases` (sans vue) | 15 prod | 22 requêtes |
| Python `row["gamertag"]` / Polars `col("gamertag")` | 20+ | 40+ |
| **Total surface** | **33 fichiers** | **86+ emplacements** |

Fichiers les plus impactés :
- `src/data/repositories/_gamertag_resolver.py` (9 requêtes)
- `src/data/repositories/_roster_loader.py` (3 requêtes)
- `src/data/repositories/_encounter_loader.py` (1 requête)
- `src/data/repositories/_events_repo.py` (1 requête)
- `src/ui/pages/explorer_data.py` (3 requêtes, bypass le repo)
- `src/data/services/teammates_service.py` (1 requête)
- `src/ui/pages/teammates_impact.py` (1 requête, bypass le repo)

#### 🆕 Découverts par audit complémentaire (non dans le plan initial)

| Fichier | Hits | Détail |
|---------|:----:|--------|
| `src/data/repositories/_weapon_kills_repo.py` | 7 | `highlight_events` × 4, `xuid_aliases` × 1, `match_participants` × 2 |
| `src/ui/pages/career_encounters_data.py` | 3 | `match_participants.gamertag` × 3 |
| `src/data/repositories/_discord_queries.py` | 6 | `match_registry` + `match_participants` combinés |
| `src/data/repositories/_calibration_loaders.py` | 3 | Requêtes avec gamertag |
| `src/data/repositories/_performance.py` | 2 | xuid_aliases direct |
| `src/data/repositories/_skill_rating.py` | 2 | xuid_aliases direct |
| `src/data/services/aliases.py` | 1 | Cache global xuid_aliases |
| `src/data/services/media_helpers.py` | 1 | Load all xuid_aliases |
| `src/utils/xuid.py` | 1 | xuid_aliases direct |
| `src/ui/pages/setup_smoke_test_logic.py` | 3 | Requêtes diagnostiques |
| `scripts/backfill/orchestrator.py` | 12 | Requêtes backfill directes |

> **Impact** : ces fichiers seront couverts automatiquement une fois que les repositories
> qu'ils appellent utiliseront `v_gamertag_lookup`. Les accès `xuid_aliases` directs (15 fichiers)
> n'ont **pas besoin de migrer** vers la vue si leur but est de lire la source de vérité —
> la vue est un FULL OUTER JOIN qui inclut xuid_aliases.

### A.1 Vue SQL — `v_gamertag_lookup`

Créer dans `shared_matches.duckdb` :

```sql
CREATE OR REPLACE VIEW v_gamertag_lookup AS
SELECT
    xa.xuid,
    COALESCE(xa.gamertag, mp.gamertag) AS gamertag
FROM xuid_aliases xa
FULL OUTER JOIN (
    SELECT xuid, MAX(gamertag) AS gamertag
    FROM match_participants
    WHERE gamertag IS NOT NULL
    GROUP BY xuid
) mp ON xa.xuid = mp.xuid
WHERE COALESCE(xa.gamertag, mp.gamertag) IS NOT NULL;
```

**Pourquoi FULL OUTER JOIN** : couvre les XUIDs inconnus de `xuid_aliases` (bots, matchs
anciens pré-v5). Le COALESCE garantit : xuid_aliases prioritaire, match_participants fallback.

**Pourquoi pas highlight_events** : données corrompues (NUL bytes), qualité insuffisante.
Sera supprimée à terme (Phase A.4).

### A.2 Fonction Python unique — `GamertagResolverMixin` (refactored)

`_gamertag_resolver.py` existe déjà comme mixin du DuckDBRepository.
Le refactoring consiste à :

1. **Simplifier `resolve_gamertag()`** : remplacer la cascade 5-sources par une
   seule requête sur `v_gamertag_lookup` + fallback highlight_events nettoyé.

```python
def resolve_gamertag(self, xuid: str, match_id: str | None = None) -> str | None:
    """Résout un XUID → gamertag via la vue v_gamertag_lookup.

    Priorité : xuid_aliases > match_participants (géré par la vue).
    Fallback : highlight_events avec extraction ASCII (transitoire).
    """
    if not xuid:
        return None

    # Source unique : vue centralisée
    with self._shared_read() as conn:
        row = conn.execute(
            "SELECT gamertag FROM v_gamertag_lookup WHERE xuid = ?", (xuid,)
        ).fetchone()
        if row and row[0]:
            gt = _clean_gamertag_static(row[0])
            if gt:
                logger.debug("resolve_gamertag(%s): source=v_gamertag_lookup → %s", xuid, gt)
                return gt

    # Fallback transitoire : highlight_events (sera supprimé en A.4)
    gt = self._resolve_from_highlight_events(xuid, match_id)
    if gt:
        logger.debug("resolve_gamertag(%s): source=highlight_events → %s", xuid, gt)
        return gt

    logger.warning("resolve_gamertag(%s): aucune source", xuid)
    return None
```

2. **Simplifier `load_match_player_gamertags()`** : une seule requête JOIN
   au lieu de 4 requêtes séquentielles.

```python
def load_match_player_gamertags(self, match_id: str) -> dict[str, str]:
    """Retourne {xuid: gamertag} pour tous les joueurs d'un match.

    Utilise v_gamertag_lookup pour résoudre les noms courants.
    """
    result: dict[str, str] = {}

    with self._shared_read() as conn:
        rows = conn.execute("""
            SELECT mp.xuid, COALESCE(vg.gamertag, mp.gamertag) AS gamertag
            FROM match_participants mp
            LEFT JOIN v_gamertag_lookup vg ON mp.xuid = vg.xuid
            WHERE mp.match_id = ?
              AND mp.xuid IS NOT NULL
        """, (match_id,)).fetchall()

        for xuid, gt in rows:
            cleaned = _clean_gamertag_static(gt)
            if xuid and cleaned:
                result[xuid] = cleaned

    if not result:
        logger.debug("load_match_player_gamertags(%s): aucun joueur résolu", match_id)

    return result
```

### A.3 Migration des consommateurs directs

Les fichiers qui bypass le repo (SQL direct sur `gamertag`) doivent être migrés :

| Fichier | Requête actuelle | Migration |
|---------|-----------------|-----------|
| `explorer_data.py:67` | `SELECT gamertag FROM highlight_events UNION xuid_aliases` | → `SELECT gamertag FROM v_gamertag_lookup` |
| `explorer_data.py:106` | `SELECT xuid FROM highlight_events WHERE gamertag = ?` | → `SELECT xuid FROM v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?)` |
| `teammates_impact.py:51` | `SELECT ... gamertag FROM highlight_events` | → JOIN `v_gamertag_lookup` sur xuid |
| `teammates_service.py:76` | Query `highlight_events.gamertag` pour events kill/death | → Résoudre via xuid + JOIN `v_gamertag_lookup` (lors du drop `highlight_events.gamertag` en Wave 4) |
| `teammates_service.py:149` | `SELECT xuid FROM match_participants WHERE gamertag = ?` | → `SELECT xuid FROM v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?)` |
#### 🆕 Fichiers supplémentaires découverts

| Fichier | Requête actuelle | Migration |
|---------|-----------------|----------|
| `_weapon_kills_repo.py` | 4× `highlight_events.gamertag` + 2× `match_participants.gamertag` | → JOIN `v_gamertag_lookup` |
| `career_encounters_data.py` | 3× `match_participants.gamertag` | → JOIN `v_gamertag_lookup` |
| `_discord_queries.py` | `match_participants.gamertag` combiné | → JOIN `v_gamertag_lookup` |
| `_calibration_loaders.py` | 3× requêtes avec gamertag | → JOIN `v_gamertag_lookup` |
> **`_roster_loader.py`**, **`_encounter_loader.py`** et **`_events_repo.py`** : couverts
> **indirectement** par Commit 2 — ils appellent `self.resolve_gamertag()` via le mixin,
> pas de SQL direct sur les colonnes gamertag. Aucune modification requise dans ces fichiers.

#### Décision explicite — fichiers découverts sans commit assigné

| Fichier | Mécanisme | Décision |
|---------|-----------|----------|
| `_performance.py`, `_skill_rating.py` | Via mixin → `resolve_gamertag()` | ✅ Couverts par Commit 2 |
| `_calibration_loaders.py` | Via mixin / repo | ✅ Couvert par Commit 2 |
| `_weapon_kills_repo.py` | SQL direct `highlight_events.gamertag` ×4 | → Ajouter à Commit 3 |
| `_discord_queries.py` | SQL direct `match_participants.gamertag` + noms assets | → Commit 3 (gamertag) + Commit 5 (assets) |
| `media_library_data.py`, `match_view_logic.py` | Passent par `load_matches()` | ✅ Couverts par C.2.2 (`mv_player_matches`) |
| `setup_smoke_test_logic.py` | 3 requêtes diagnostiques intentionnellement directes | **Garder direct** — ne pas migrer |

### A.4 Suppression `highlight_events.gamertag`

Après A.1–A.3 :
- La vue `v_gamertag_lookup` ne l'utilise pas
- Le resolver ne l'utilise plus (ou en fallback transitoire)
- `explorer_data.py` migré sur la vue

→ Migration de schéma pour supprimer la colonne (migration step détaillé en Wave 4, Commit 8).

### A.5 Décision `match_participants.gamertag`

**→ ON GARDE** comme fallback dans la vue `v_gamertag_lookup`.

Raisons :
- Couvre les XUIDs absents de `xuid_aliases` (bots, données pré-v5, sync partiel)
- Dénormalisation utile en performance (évite un JOIN systématique dans les requêtes
  scoreboard qui lisent déjà match_participants)
- La vue COALESCE gère la priorité : si `xuid_aliases` a le XUID, son gamertag gagne

Le jour où on veut supprimer la colonne, on modifie la vue → aucun impact consommateur.

### A.6 Suppression wrappers XUID fragmentés

| Fonction à supprimer | Fichier | Raison |
|---------------------|---------|--------|
| `resolve_xuid_from_input()` | `main_helpers.py` | Wrapper 1 ligne vers `resolve_xuid_input()` |

| Fonction à garder | Fichier | Raison |
|-------------------|---------|--------|
| `resolve_xuid_from_db()` | `xuid.py` | Référence canonique (parsing + DB lookup) |
| `resolve_xuid_input()` | `data_loader.py` | Enveloppe avec fallback secrets (logique réelle) |
| `resolve_xuid()` | `profile.py` | Variante `PlayerIdentity` — utilisée par media_tab/media_library |
| `get_xuid_for_gamertag()` | `profile_api.py` | SPNKr API wrapper — rôle distinct (appel réseau) |

`resolve_xuid()` (profile.py) n'est **pas un doublon** de `resolve_xuid_input()` :
elle prend un `PlayerIdentity` en paramètre au lieu de lire les secrets. C'est une
variante légitime pour les pages média.

---

## Volet B — Outcomes (code résultat → label traduit)

### B.0 Audit de surface

| Fonction | Fichier | Appelants prod | Rôle |
|----------|---------|:--------------:|------|
| `get_outcome_name_fr()` | `refdata.py` | **0** (dead code) | Retourne nom FR statique |
| `get_outcome_map(lang)` | `i18n/__init__.py` | 5+ pages UI | Retourne `{code: label}` via `t()` |
| `resolve_outcome(row)` | `match_view_logic.py` | 1 (match_view) | `row → (code, label, couleur)` |
| `OUTCOME_TO_FR` dict | `refdata.py` | **0** (dead code) | Dict statique `{2: "Victoire", ...}` |

### B.1 Plan

1. **Supprimer** `get_outcome_name_fr()` et `OUTCOME_TO_FR` de `refdata.py` (dead code).
2. **Garder** `get_outcome_map(lang)` comme référence canonique (i18n, multilingue).
3. **Garder** `resolve_outcome(row)` — ce n'est pas un doublon, elle ajoute la résolution
   couleur (responsabilité UI légitime). Mais documenter qu'elle dépend de `get_outcome_map()`.
4. **Déplacer `get_outcome_map()`** vers `src/data/domain/_refdata_outcomes.py` (couche data)
   pour éviter l'import circulaire `data → ui` :

   > ⚠️ **Règle architecturale** : `data → ui` est interdit. La solution est de déplacer
   > `get_outcome_map()` dans la couche data en remplaçant l'appel `t()` par un dict
   > statique FR/EN (4 outcomes stables, ne changent pas). `src/ui/i18n/__init__.py`
   > devient un re-export de la version data.

   ```python
   # src/data/domain/_refdata_outcomes.py
   _OUTCOME_LABELS: dict[str, dict[int, str]] = {
       "fr": {1: "Égalité", 2: "Victoire", 3: "Défaite", 4: "Non terminé"},
       "en": {1: "Tie", 2: "Win", 3: "Loss", 4: "Did Not Finish"},
   }

   def get_outcome_map(lang: str = "fr") -> dict[int, str]:
       """Retourne {code: label} pour les outcomes. Aucune dépendance vers ui."""
       return _OUTCOME_LABELS.get(lang, _OUTCOME_LABELS["fr"])
   ```

   `src/ui/i18n/__init__.py` : remplacer l'implémentation existante par :
   ```python
   from src.data.domain._refdata_outcomes import get_outcome_map  # re-export
   ```

---

## Volet C — Noms d'Assets (map_name, playlist_name, etc.)

### C.0 Audit de surface

| Catégorie | Fichiers | Emplacements |
|-----------|:--------:|:------------:|
| SQL SELECT `*_name` depuis match_registry | 10 | 45 requêtes |
| Polars `col("map_name")` / `col("playlist_name")` | 25+ | 115+ |
| **Total surface** | **35+ fichiers** | **160+ emplacements** |

Fichiers les plus impactés :
- `src/app/_filters_cascade.py` (6 usages)
- `src/app/_filters_apply.py` (10 usages)
- `src/analysis/maps.py` (12 usages)
- `src/analysis/citations/custom_rules.py` (8 usages)
- `src/ui/pages/match_history.py` (4 usages)
- `src/ui/pages/explorer.py` (4 usages)

### C.1 État actuel : couverture partielle par `mv_player_matches`

La vue matérialisée `mv_player_matches` (définie dans `migrations.py:701`) contient déjà
les 4 colonnes `*_name` et est la source principale des pages UI via `load_matches()`.

**Mais** :
- La vue lit `match_registry.map_name` directement (pas de JOIN metadata) — si le nom
  est absent ou stale dans `match_registry`, la vue est incorrecte aussi
- Les scripts backfill/detection lisent `match_registry` directement, pas la vue
- Le mécanisme de JOIN metadata existe dans `_metadata_resolution.py` mais n'est pas
  utilisé par tous les chemins

### C.2 Plan complet

**Phase C.2.1 — Vue SQL `v_match_full`** (résolution à la lecture)

Remplace les accès directs à `match_registry` pour toute requête ayant besoin de noms :

```sql
-- Requiert ATTACH 'metadata.duckdb' AS meta à chaque connexion qui requête v_match_full.
-- ⚠️ Pas encore fait dans le repo — créer ensure_metadata_attached(conn) dans src/utils/db.py
--    sur le modèle de ensure_shared_attached() (ligne 128). À intégrer au Commit 1.
CREATE OR REPLACE VIEW v_match_full AS
SELECT
    mr.match_id,
    mr.start_time,
    mr.duration_seconds,
    mr.map_id,
    mr.playlist_id,
    mr.pair_id,
    mr.game_variant_id,
    mr.team_0_score,
    mr.team_1_score,
    mr.is_firefight,
    mr.is_ranked,
    mr.backfill_completed,
    mr.sync_spnkr_version,
    -- Noms EN : metadata prioritaire, match_registry fallback
    -- ⚠️ Garder EN obligatoirement : mark_firefight(), participation_radar et
    --    _is_objective_mode_from_pair_name() parsent ces colonnes avec des patterns EN.
    COALESCE(m.name_en,  mr.map_name)            AS map_name,
    COALESCE(p.name_en,  mr.playlist_name)       AS playlist_name,
    COALESCE(pp.name_en, mr.pair_name)           AS pair_name,
    COALESCE(gv.name_en, mr.game_variant_name)   AS game_variant_name,
    -- Noms FR : colonnes additionnelles pour l'affichage UI
    -- NULL si metadata.duckdb pas encore peuplée (avant Commit 0) → fallback dans translate_*()
    m.name_fr                                    AS map_name_fr,
    p.name_fr                                    AS playlist_name_fr,
    pp.name_fr                                   AS pair_name_fr,
    gv.name_fr                                   AS game_variant_name_fr,
    -- Colonnes de normalisation (v6+)
    gv.mode_name                                 AS mode_name,
    gv.mode_name_fr                              AS mode_name_fr,
    p.playlist_canonical_en                      AS playlist_canonical_en,
    p.playlist_canonical_fr                      AS playlist_canonical_fr
FROM match_registry mr
LEFT JOIN meta.maps m ON mr.map_id = m.asset_id
LEFT JOIN meta.playlists p ON mr.playlist_id = p.asset_id
LEFT JOIN meta.playlist_map_mode_pairs pp ON mr.pair_id = pp.asset_id
LEFT JOIN meta.game_variants gv ON mr.game_variant_id = gv.asset_id;
```

> **⚠️ Pré-requis (Commit 0)** : Les tables `meta.maps`, `meta.playlists`, `meta.playlist_map_mode_pairs`
> et `meta.game_variants` doivent exister dans `metadata.duckdb` et contenir `name_en`, `name_fr`
> avant que la vue soit pleinement opérationnelle. Avant Commit 0, les colonnes `*_fr` seront `NULL`
> et les colonnes EN tomberont en fallback sur `match_registry` (comportement actuel préservé).
>
> **Principe** : les colonnes `map_name` / `playlist_name` / `pair_name` / `game_variant_name`
> restent EN pour toute la logique métier. Les colonnes `*_fr` sont ajoutées **en plus** pour
> l'affichage. À terme, `translate_playlist_name()` pourra s'alimenter de `playlist_name_fr`
> directement au lieu des dicts hardcodés — sans toucher aux colonnes EN.
>
> La vue expose des colonnes avec les **mêmes noms** pour l'EN (`map_name`, `playlist_name`, etc.)
> → les consommateurs Python/Polars n'ont **rien à changer** dans leur code `col("map_name")`.

**Phase C.2.2 — Mise à jour `mv_player_matches`**

La vue matérialisée doit pointer sur `v_match_full` au lieu de `match_registry` :

```sql
-- Avant : FROM match_registry r JOIN match_participants p ...
-- Après  : FROM v_match_full r JOIN match_participants p ...
```

→ Les noms dans la vue matérialisée seront automatiquement les plus à jour.

**Phase C.2.3 — Migration des requêtes directes**

| Fichier | Requête actuelle | Migration |
|---------|-----------------|-----------|
| `scripts/backfill/strategies.py:386` | `SELECT mr.playlist_name` FROM `match_registry` | → FROM `v_match_full` |
| `scripts/backfill/strategies.py:630` | idem | → FROM `v_match_full` |
| `scripts/backfill/detection.py:408` | `... map_name IS NULL` FROM `match_registry` | → FROM `v_match_full` |
| `src/analysis/citations/_data_loader.py:88` | `LEFT JOIN match_registry` | → `LEFT JOIN v_match_full` |
#### 🆕 Fichiers supplémentaires découverts

| Fichier | Requête actuelle | Migration |
|---------|-----------------|----------|
| `scripts/backfill/orchestrator.py` | 12 requêtes directes `match_registry` + `match_participants` | → FROM `v_match_full` pour les noms |
| `scripts/backfill/detection.py` | 2 requêtes supplémentaires (hors 408) | → FROM `v_match_full` |
| `scripts/backfill/strategies.py` | 4 requêtes supplémentaires (hors 386/630) | → FROM `v_match_full` |
| `_discord_queries.py` | `match_registry.*_name` combiné | → `v_match_full` |
| `setup_smoke_test_logic.py` | 3 requêtes diagnostiques | → `v_match_full` (ou garder direct si diagnostic) |
| `career_data.py` | 2 requêtes avec noms assets | → Couvert par `mv_player_matches` (C.2.2) |
| `media_library_data.py` | 1 requête avec noms assets | → Couvert par `v_match_full` |
| `match_view_logic.py` | 1 requête avec noms assets | → Couvert par `v_match_full` |
| `multiplayer.py` | 3 requêtes `match_registry` directes | → FROM `v_match_full` |
> Les pages UI qui passent par `load_matches()` → `mv_player_matches` sont couvertes
> automatiquement par C.2.2.

> **⚠️ Impacts UI — dropdowns sidebar à surveiller impérativement lors de Volet C**
>
> Le pipeline de traduction des filtres cascade opère ainsi :
> `playlist_name` (DB, EN) → `translate_playlist_name()` → `playlist_ui` (FR, dropdown)
>
> La vue `v_match_full` retourne `public_name` (EN) via `COALESCE(x.public_name, mr.x_name)`.
> Le comportement est **identique à aujourd'hui** : `translate_playlist_name()` reçoit de l'EN et
> traduit en FR. **Pas de changement de comportement UI.**
>
> **Règle rappelée** : ne jamais injecter `name_fr` dans les colonnes `playlist_name` /
> `pair_name` de la vue — `mark_firefight()`, `participation_radar.py` et
> `_is_objective_mode_from_pair_name()` sont EN-dépendants et casseraient silencieusement.
>
> **Risque 1 — Mode EN** : ✅ OK — `translate_playlist_name(EN, "en")` lookup JSON → EN correct.
>
> **Risque 2 — `ranked_cond`** (`str.contains("classé|ranked")`) : ✅ OK — la colonne `playlist_ui`
> (post-traduction FR) contient "Arène classée" → "classé" matche.
>
> **Risque 3 — Détection Firefight** : ✅ OK — `_FIREFIGHT_PATTERNS` dans
> `checkbox_filter.py:461` inclut "baptême du feu" pour la `playlist_ui` (post-traduction).
> La `playlist_name` brute (EN) continue de contenir "Firefight" → `mark_firefight()` OK.
>
> **Risque 4 — `participation_radar.py` LIKE queries** : ✅ OK — `participation_radar.py` lit
> directement `shared.match_registry` (ligne 103), pas via `v_match_full`. Les LIKE
> `'%firefight%'` opèrent sur l'EN stocké dans `match_registry` → non impacté.
>
> **Risque 5 — `session_state` filter persistence** : ✅ OK — les valeurs stockées dans
> `session_state["filter_playlists"]` = labels `playlist_ui` (FR). Avant et après migration,
> `playlist_ui` reste FR.

**Phase C.2.4 — Suppression des colonnes `*_name` de `match_registry` (futur)**

Quand tous les consommateurs passent par `v_match_full` ou `mv_player_matches` :

1. Modifier la vue pour ne plus référencer `mr.map_name` dans le COALESCE → résolution
   uniquement depuis `metadata.duckdb`
2. Migration DuckDB : `ALTER TABLE match_registry DROP COLUMN map_name, playlist_name, ...`
3. Adapter le transformateur `_match.py` pour ne plus insérer les noms au sync

> ⚠️ Pas une priorité immédiate — les noms d'assets changent rarement (pas de stale data
> critique). Mais la vue est le **pré-requis**. Une fois en place, la suppression des colonnes
> ne sera qu'une formalité.

---

## Volet D — Paires Killer/Victim (killer_gamertag, victim_gamertag)

### D.0 Audit de surface (révisé)

| Catégorie | Fichiers | Emplacements |
|-----------|:--------:|:------------:|
| Schéma + Modèles | 4 | `_engine_connections.py`, `_batch_columns.py`, `_kv_types.py`, `models.py` |
| Écritures (INSERT) | 2 | `_shared_writes.py:75`, `strategies.py:183` |
| Lectures SQL | 1 prod | `_killer_victim_repo.py:73` |
| Polars GROUP BY / Agrégations | 5 prod | `_killer_victim_repo.py`, `_killer_victim_polars.py`, `_antagonist_kv.py`, `match_view_players_nemesis.py`, `friends_impact_heatmap.py` |
| Tests | 10 | Données de test, assertions, schéma |
| **Total surface** | **14 prod + 10 tests** | **82+ emplacements** |

#### 🆕 Découverts par audit complémentaire

| Fichier | Hits | Détail |
|---------|:----:|--------|
| `career_encounters_data.py` | 4 | `killer_victim_pairs` × 4 (bypass le repo) |
| `_encounter_loader.py` | 1 | `killer_victim_pairs` direct (supplémentaire) |
| `weapon_extraction_service.py` | 2 | `killer_gamertag` / `victim_gamertag` dans extraction |

### D.1 Problème

Les colonnes `killer_gamertag` et `victim_gamertag` dans `killer_victim_pairs` sont des
**snapshots figés** au moment du sync, exactement comme `match_participants.gamertag`.
Si un joueur change de gamertag, les paires K/V affichent l'ancien nom.

De plus, la source est `highlight_events.gamertag` (données brutes, possiblement corrompues
avec NUL bytes) → les paires K/V héritent des défauts de la source.

### D.2 Vue SQL — `v_killer_victim_full`

```sql
CREATE OR REPLACE VIEW v_killer_victim_full AS
SELECT
    kv.match_id,
    kv.killer_xuid,
    COALESCE(vk.gamertag, kv.killer_gamertag, kv.killer_xuid) AS killer_gamertag,
    kv.victim_xuid,
    COALESCE(vv.gamertag, kv.victim_gamertag, kv.victim_xuid) AS victim_gamertag,
    kv.kill_count,
    kv.time_ms,
    kv.is_validated
FROM killer_victim_pairs kv
LEFT JOIN v_gamertag_lookup vk ON kv.killer_xuid = vk.xuid
LEFT JOIN v_gamertag_lookup vv ON kv.victim_xuid = vv.xuid;
```

**Chaîne de résolution** :
1. `v_gamertag_lookup` (xuid_aliases → match_participants) = gamertag **courant**
2. `kv.killer_gamertag` (snapshot figé) = fallback si XUID inconnu
3. `kv.killer_xuid` (brut) = dernier recours

### D.3 Migration des consommateurs

| Fichier | Requête actuelle | Migration |
|---------|-----------------|-----------|
| `_killer_victim_repo.py:73` | `SELECT ... FROM killer_victim_pairs` | → FROM `v_killer_victim_full` |
| `career_encounters_data.py` | 4× `SELECT ... FROM killer_victim_pairs` (bypass le repo) | → FROM `v_killer_victim_full` |

Les opérations Polars en aval (`_killer_victim_polars.py`, `_antagonist_kv.py`,
`match_view_players_nemesis.py`) ne changent **rien** — elles lisent des colonnes
nommées `killer_gamertag` / `victim_gamertag` qui sortent maintenant de la vue avec
des noms à jour.

### D.4 Conservation des colonnes dans la table

Comme pour `match_participants.gamertag` : **on garde** les colonnes `killer_gamertag`
et `victim_gamertag` dans la table brute comme fallback dans la vue. Elles ne seront
supprimées que quand `xuid_aliases` couvrira 100% des XUIDs connus.

### D.5 Mise à jour du transformateur (écriture)

Pas de changement immédiat : le sync continue d'écrire `killer_gamertag` / `victim_gamertag`
dans `killer_victim_pairs`. C'est un cache — la vue résout toujours le nom courant en priorité.

> Future optimisation : arrêter d'écrire ces colonnes et compter uniquement sur la vue.
> Mais ce n'est pas bloquant maintenant.

---

## Volet E — Médailles (medal_name_id → nom)

### E.1 État actuel

Pas de helper centralisé dans le code Python. Les noms de médailles sont :
- Soit lus directement depuis l'API SPNKr
- Soit affichés comme identifiant brut en UI
- Soit résolus via `load_medal_name_maps()` dans `src/ui/medals.py` (JSON statique `static/medals/`)

> L'audit complémentaire a confirmé **69 emplacements** Polars utilisant `medal_name` dans l'UI,
> tous alimentés par le JSON statique. Ce helper est fonctionnel mais ne passe pas par DuckDB.

### E.2 Plan

Créer `src/analysis/_medal_data.py` (analogue à `_weapon_data.py`) :

```python
def resolve_medal_name(medal_name_id: int, lang: str = "fr") -> str:
    """Résout un medal_name_id en nom lisible depuis metadata.duckdb."""
```

Alimenté par la table `medals` de `metadata.duckdb` (si elle existe), sinon fallback `str(id)`.

> **Transition `load_medal_name_maps()`** : la fonction dans `src/ui/medals.py` (JSON statique)
> est **conservée dans le scope v6** pour compatibilité UI immédiate. Le nouveau `resolve_medal_name()`
> depuis `metadata.duckdb` sera la source de vérité cible. La refactorisation de `load_medal_name_maps()`
> pour appeler `resolve_medal_name()` est prévue comme **prochaine étape (v6.1+)** — pas un abandon,
> une évolution séquentielle.

> ⚠️ Pré-requis : vérifier que `metadata.duckdb` a bien une table `medals` avec les
> colonnes `medal_name_id` et `name_fr`/`name_en`. Sinon, créer le schéma + populer
> depuis l'API SPNKr (script `populate_metadata_from_discovery.py`).

---

## Impact Polars — Résultat de l'audit complémentaire

> **Conclusion : AUCUN changement Polars n'est nécessaire.**

Les 3 vues SQL résolvent les noms **avant** que les DataFrames Polars ne les reçoivent.
Les opérations Polars (`col("map_name")`, `col("killer_gamertag")`, `group_by("gamertag")`,
`replace_strict(...)`) continuent de fonctionner sans modification — les colonnes ont les
**mêmes noms**, juste des valeurs mieux résolues.

| Opération Polars | Source actuelle | Source avec vues | Code change ? |
|-----------------|-----------------|------------------|:-------------:|
| `pl.col("map_name")` → filtre | Table brute (possible NULL) | `v_match_full` (résolu) | ❌ Non |
| `pl.col("killer_gamertag")` → groupby | Lookup + combinaison Python | `v_killer_victim_full` + SQL JOIN | ❌ Non |
| `pl.col("gamertag")` → unique().to_list() | `match_participants` | Via `v_gamertag_lookup` dans les repos | ❌ Non |
| `build_mapping(df["playlist_name"], func)` | DataFrame brut | DataFrame résolu | ❌ Non |

**Patterns Polars audités** : 259 emplacements dans 30+ fichiers — aucun ne nécessite de migration.

Les `replace_strict()` i18n (traduction `playlist_name` → nom FR affiché) restent pertinents
même avec les vues : la vue résout le nom **anglais** depuis `metadata`, le `replace_strict()`
traduit en **français** pour l'UI.

---

## Autres Patterns ID→Nom — Résultat de l'audit complémentaire

L'audit a vérifié 8 patterns de résolution non couverts par le plan v6 :

| Pattern | Statut | Centralisé ? | Recommandation |
|---------|:------:|:------------:|----------------|
| Team ID (0–8 → "Eagle"/"Cobra") | ✅ Fonctionnel | `src/config.py` | Garder séparé (domaine config) |
| Rank/CSR Tier → label FR | ✅ Fonctionnel | `src/ui/i18n/ranks.py` | Garder séparé (domaine skill) |
| Playlist Groups (6 groupes) | ✅ Fonctionnel | `src/analysis/playlist_groups.py` | Catégorisation, pas résolution |
| Weapon ID → nom | ✅ Fonctionnel | `src/analysis/_weapon_data.py` | Déjà centralisé (v5.7) |
| Commendation rules | ✅ Fonctionnel | `metadata.duckdb` + Python custom | Complexe, garder séparé |
| Medal ID → nom | ✅ Fonctionnel | `src/ui/medals.py` (JSON) | Helper DuckDB en v6 (Volet E) |
| Personal Score ID → nom | ✅ Fonctionnel | `src/data/domain/_refdata_personal_scores.py` | Garder séparé |
| Label normalization | ✅ Fonctionnel | `src/app/helpers.py` | Pas une résolution ID |

> **Aucune lacune critique** : tous les patterns `*_id` → `*_name` sont déjà implémentés.
> Le plan v6 est **complet** pour son scope (centralisation + abstraction SQL).

---

## Stratégie DB : Travailler sur une copie `shared_matches_v2.duckdb`

### Principe

Plutôt que de modifier la DB de production directement, tout le travail v6 se fait
sur une **copie `shared_matches_v2.duckdb`**. La prod n'est jamais touchée jusqu'au
bascule final.

```
│ Production (intacte)            │  Développement v6                │
│ shared_matches.duckdb           │  shared_matches_v2.duckdb        │
│ Version v5.7 (current)          │  Version v6 (en cours)           │
│ Utilisée par l'app Streamlit    │  Utilisée sur la branche refactor│
│ Ne jamais modifier              │  Toutes les vues + DROP column  │
```

### Setup (avant Wave 1)

```bash
# 1. Copier la DB de prod
cp data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v2.duckdb

# 2. Pour tester la v2 avec l'app Streamlit : swap temporaire de fichiers (app arrêtée)
# Le chemin est codé dans src/utils/paths.py::SHARED_MATCHES_DB_FILENAME = "shared_matches.duckdb"
# Aucune variable d'environnement LEVELUP_SHARED_DB n'est supportée dans le code.
# → Renommer les fichiers temporairement :
mv data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_prod.duckdb
cp data/warehouse/shared_matches_v2.duckdb data/warehouse/shared_matches.duckdb
# L'app et les tests lisent shared_matches.duckdb normalement (= v2).
# Pour revenir à la prod :
#   mv data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v2.duckdb
#   mv data/warehouse/shared_matches_prod.duckdb data/warehouse/shared_matches.duckdb
```

### Bascule finale (après Wave 5 — audit OK)

```bash
# La v1 devient l'archive, la v2 devient prod
mv data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v1_backup_$(date +%Y%m%d).duckdb
mv data/warehouse/shared_matches_v2.duckdb data/warehouse/shared_matches.duckdb

# Remettre db_profiles.json sur le chemin par défaut (shared_matches.duckdb)
```

### Avantages

| Avant (modif directe) | Avec copie v2 |
|-----------------------|---------------|
| Backup manuel obligatoire avant commit 8 | La v1 est l'archive automatique |
| Erreur = rollback complexe | Rollback = supprimer v2, relancer avec v1 |
| App Streamlit potentiellement cassée pendant dev | App tourne sur v1 (stable) pendant tout le chantier |
| `DROP COLUMN` irréversible en prod | `DROP COLUMN` sur la copie seulement |

### Remarque sur les tests

Les tests unitaires créent leurs propres DB temporaires `tmp_path` — ils ne lisent
pas `shared_matches_v2.duckdb`. La stratégie v2 n'affecte donc pas la suite de tests.

---

## Ordonnancement par Commits

### Branche : `refactor/id-resolution-cleanup` (depuis `analysis/weapon-parser-rewrite`)

```bash
# Création de la branche v6
git checkout analysis/weapon-parser-rewrite
git checkout -b refactor/id-resolution-cleanup

# Setup DB v2 (avant de commencer) — arrêter l'app Streamlit avant la copie
cp data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v2.duckdb
```

#### Commit 0 — Pré-requis metadata (avant Wave 1)

| # | Commit | Volet | Risque | Action |
|:-:|--------|:-----:|:------:|--------|
| 0 | `chore(metadata): peupler maps/playlists/game_variants` | C (pré-requis) | FAIBLE | Modifier + exécuter `populate_metadata_from_discovery.py` |

> Les tables `meta.maps`, `meta.playlists`, `meta.playlist_map_mode_pairs` et `meta.game_variants`
> sont absentes de `metadata.duckdb` par défaut. Elles sont nécessaires pour que `v_match_full`
> résolve les noms depuis la source de vérité plutôt que depuis le cache `match_registry`.
>
> **Avant d'exécuter le script**, il faut enrichir son schéma avec les colonnes i18n et de
> normalisation décrites ci-dessous.

##### Enrichissement du schéma `game_variants`

La table actuelle n'a que `asset_id` et `public_name` (EN). Il faut ajouter :

| Colonne | Type | Dérivation |
|---------|------|-----------|
| `name_en` | `VARCHAR` | = `public_name` (alias explicite) |
| `name_fr` | `VARCHAR` | via `mode_translations` sur le `public_name` exact → mécanisme de migration one-shot |
| `mode_name` | `VARCHAR` | extrait programmatiquement : `TRIM(SPLIT_PART(SPLIT_PART(public_name, ':', 2), ' on ', 1))` |
| `mode_name_fr` | `VARCHAR` | `mode_translations.name_fr` du `mode_name` correspondant, ou `NULL` si absent |

**Règle d'extraction `mode_name`** :
- `"Arena:Attrition on Catalyst"` → `"Attrition"` ✅
- `"FFA Slayer"` (sans `:`) → colonne vide → écrire dans le fichier d'erreurs
- Résultat attendu : **27 `mode_name` distincts** sur 313 variantes

Les 313 entrées de `mode_translations` deviennent une **source de migration one-shot** pour peupler
`name_fr` et `mode_name_fr`. Une fois le populate effectué, `mode_translations` sera conservée
mais ne sera plus interrogée par les vues (obsolescence planifiée Wave 5).

##### Enrichissement du schéma `playlists`

La table actuelle n'a que `asset_id` et `public_name` (EN). Même souci de variantes qu'avec les
modes : 3 entrées EN distinctes partagent la même traduction FR "Grande bataille en équipe"
("Big Team Battle", "Big Team Battle: Refresh", "Big Team Social"). Ajouter :

| Colonne | Type | Dérivation |
|---------|------|-----------|
| `name_en` | `VARCHAR` | = `public_name` (alias explicite) |
| `name_fr` | `VARCHAR` | via `playlist_translations.name_fr` sur `playlist_id` exact |
| `playlist_canonical_en` | `VARCHAR` | `TRIM(SPLIT_PART(public_name, ':', 1))` (préfixe avant ":") |
| `playlist_canonical_fr` | `VARCHAR` | = `name_fr` (la traduction FR IS déjà le regroupement sémantique) |

**Différence avec les modes** : les playlists sont UUID-keyed (robustes), la `name_fr` issue de
`playlist_translations` sert directement de canonique FR. Le `playlist_canonical_en` est simplement
le préfixe avant ":" (ex: "Big Team Battle: Refresh" → "Big Team Battle").

> **Résultat attendu** : "Big Team Battle", "Big Team Battle: Refresh", "Big Team Social" →
> tous `playlist_canonical_fr = "Grande bataille en équipe"`, mais `playlist_canonical_en` distincts
> pour les deux derniers ("Big Team Battle" vs "Big Team Social").

##### Mécanisme de fichier d'erreurs

Lors du populate, tout cas non résolu doit être tracé dans **`metadata_populate_errors.txt`** à la
racine du projet (chemin relatif du repo). Ce fichier est destiné aux corrections manuelles.

Format :
```
[2025-01-15 14:32:00] game_variants | asset_id=abc123 | public_name="FFA Slayer" | raison=mode_name_extraction_failed (pas de ':')
[2025-01-15 14:32:01] playlists     | asset_id=xyz789 | public_name="Custom Mode" | raison=no_translation_found (UUID absent de playlist_translations)
```

Le fichier est **appendé** à chaque exécution (pas écrasé). L'utilisateur le consulte et apporte les
corrections directement dans `metadata.duckdb` via SQL si nécessaire.

> **Ce fichier ne bloque pas l'exécution** — le populate continue même si des entrées échouent.
> `mode_name` et `name_fr` restent `NULL` pour les cas non résolus.

> ```bash
> python scripts/populate_metadata_from_discovery.py
> ```
>
> **Pré-condition** : tokens API SPNKr configurés dans `.env.local` (`SPNKR_CLIENT_ID`, `SPNKR_CLIENT_SECRET`, `SPNKR_REFRESH_TOKEN`).

<details>
<summary>✅ Checklist Commit 0</summary>

- [ ] `grep -n "maps\|playlists\|game_variants\|name_fr\|name_en" scripts/populate_metadata_from_discovery.py`
  confirme que les tables cibles et colonnes i18n sont déjà gérées par le script — sinon
  identifier la fonction à enrichir avant de lancer quoi que ce soit
- [ ] `scripts/populate_metadata_from_discovery.py` modifié avec les nouvelles colonnes
- [ ] `python scripts/populate_metadata_from_discovery.py` exécuté sans erreur bloquante
- [ ] Table `maps` présente et peuplée dans `metadata.duckdb`
- [ ] Table `playlists` présente avec colonnes `name_fr`, `playlist_canonical_en`, `playlist_canonical_fr`
- [ ] Table `playlist_map_mode_pairs` présente et peuplée
- [ ] Table `game_variants` présente avec colonnes `name_fr`, `mode_name`, `mode_name_fr`
- [ ] 27 `mode_name` distincts dans `game_variants`
- [ ] `playlist_canonical_fr` correctement peuplé pour les 3 variantes BTB
- [ ] `metadata_populate_errors.txt` créé à la racine (peut être vide si tout résolu)
- [ ] Contenu du fichier d'erreurs consulté et cas notés si non-vide

</details>

> **⚠️ Impact UI / requêtes — non-bloquant mais à surveiller**
>
> La vue `v_match_full` utilise `COALESCE(x.public_name, mr.x_name)` → retourne **toujours de l'EN**.
> Le comportement actuel est **préservé à l'identique** : translate_playlist_name() reçoit de l'EN,
> traduit en FR, et le dropdown affiche du FR. Aucune régression.
>
> **RÈGLE ARCHITECTURALE — NE PAS ENFREINDRE** : la couche DB sert de l'EN (identifiants SPNKr
> stables), la traduction FR se fait uniquement à l'affichage. Ne pas injecter `name_fr` dans les
> colonnes `playlist_name` / `pair_name` / `map_name` de la vue `v_match_full`. Raisons :
>
> - `filters.py::mark_firefight` utilise `r"(?i)\bfirefight\b"` sur `playlist_name` brut → rate
>   "Baptême du feu"
> - `participation_radar.py` LIKE `'%firefight%'` et `'%btb%'` sur `pair_name` brut
> - `_is_objective_mode_from_pair_name()` parse les patterns EN ("ctf", "oddball", "capture"…)
>
> Ces fonctions d'analyse **doivent recevoir de l'EN** pour fonctionner correctement.
>
> **Pour afficher des noms FR** dans l'UI depuis les nouvelles colonnes `name_fr`, la bonne approche
> est d'ajouter des colonnes dédiées dans la vue uniquement pour l'affichage (sans écraser les
> colonnes EN) — ou de continuer à passer par `translate_playlist_name()` qui pourra un jour
> être alimentée depuis `metadata.duckdb` plutôt que des dicts hardcodés.
>
> `src/data/sync/metadata_resolver.py:193` détecte déjà `name_fr` dynamiquement → ✅ déjà prêt.
> `src/data/query/engine.py:362` et `_metadata_resolution.py:71-84` : ne pas basculer vers
> `COALESCE(name_fr, ...)` avant d'avoir migré toute la logique métier EN-dépendante.

Le travail est découpé en **waves** (groupes de commits) pour limiter le risque
et permettre de valider à chaque étape.

> **⚠️ Règle d'or : marquer les tâches au fil de l'eau.**
> Dès qu'un commit est poussé, cocher `[x]` dans la checklist correspondante **avant** de passer à la suite.
> Ne jamais démarrer la wave N+1 sans avoir coché **toutes** les cases de la wave N.
> Une case non cochée = soit la tâche n'est pas faite, soit il manque une validation.

#### Wave 1 — Fondation : vues SQL + refactor cascade (2 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 1 | `feat(db): vues v_gamertag_lookup + v_match_full + v_killer_victim_full` | A.1 + C.2.1 + D.2 | MOYEN | 2 (migrations.py + _engine_connections.py) |
| 2 | `refactor(resolver): cascade gamertag via v_gamertag_lookup` | A.2 | MOYEN | 1 (_gamertag_resolver.py) |

> Après cette wave : les vues existent, le resolver les utilise, mais les anciens
> chemins directs fonctionnent encore.

<details>
<summary>✅ Checklist Wave 1 — à valider avant de démarrer Wave 2</summary>

**Setup**
- [ ] Branche `refactor/id-resolution-cleanup` créée depuis `analysis/weapon-parser-rewrite`
- [ ] `shared_matches_v2.duckdb` copié avec succès

**Commit 1 — Vues SQL**
- [ ] `v_gamertag_lookup` créée dans `shared_matches_v2.duckdb`
- [ ] `v_match_full` créée dans `shared_matches_v2.duckdb`
- [ ] `v_killer_victim_full` créée dans `shared_matches_v2.duckdb`
- [ ] `tests/test_resolution_views.py` créé — 10 tests passent
- [ ] Logs de création des vues visibles (`logger.info(...)`)

**Commit 2 — Refactor resolver**
- [ ] `_gamertag_resolver.py` : cascade remplacée par `SELECT` sur `v_gamertag_lookup`
- [ ] `tests/test_gamertag_resolver.py` créé — 7 tests passent

**Validation globale**
- [ ] `python -m pytest tests/ -q --ignore=tests/integration` → 0 fail
- [ ] `tests/test_xuid_resolution_regression.py` (732L, existant) → 0 régression

</details>

#### Wave 2 — Migration des consommateurs directs (3 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 3 | `refactor(gamertag): consommateurs directs → v_gamertag_lookup` | A.3 | FAIBLE | 4 (explorer_data, teammates_impact, teammates_service, events_repo) |
| 4 | `refactor(kv): killer_victim_repo + career_encounters_data → v_killer_victim_full` | D.3 | FAIBLE | 2 (_killer_victim_repo.py, career_encounters_data.py) |
| 5 | `refactor(assets): requêtes directes match_registry → v_match_full` | C.2.2–C.2.3 | FAIBLE | 4 (strategies.py, detection.py, _data_loader.py, migrations.py pour mv_player_matches) |

> Après cette wave : **tous les consommateurs** passent par les vues.
> Les tables brutes ne sont plus lues directement pour des noms résolus.

<details>
<summary>✅ Checklist Wave 2 — à valider avant de démarrer Wave 3</summary>

**Commit 3 — Gamertag consommateurs**
- [ ] `explorer_data.py` migré → plus de `SELECT gamertag FROM match_participants / highlight_events` direct
- [ ] `teammates_impact.py` migré
- [ ] `teammates_service.py` migré
- [ ] `events_repo.py` migré
- [ ] `tests/test_explorer_data.py` créé — 3 tests passent

**Commit 4 — Killer/Victim consommateurs**
- [ ] `_killer_victim_repo.py` migré → utilise `v_killer_victim_full`
- [ ] `career_encounters_data.py` migré (4 accès directs `killer_victim_pairs`)
- [ ] `tests/test_killer_victim_views.py` créé — 3 tests passent

**Commit 5 — Assets consommateurs**
- [ ] `strategies.py` migré → utilise `v_match_full`
- [ ] `detection.py` migré
- [ ] `_data_loader.py` migré
- [ ] `migrations.py` (`mv_player_matches`) mis à jour
- [ ] `mv_player_matches` **re-matérialisée** (DROP + CREATE + INSERT) : la MV n'est **pas**
  auto-rafraîchie quand sa définition change. La migration `ensure_mv_player_matches()`
  doit inclure un `DROP TABLE IF EXISTS mv_player_matches` avant la recréation.
  Durée estimée : 10–30 s selon le volume.
- [ ] 2 tests supplémentaires dans `test_resolution_views.py` passent

**Vérification manuelle**
- [ ] `grep -rn "FROM match_participants WHERE" src/ scripts/ --include="*.py"` → 0 hit de lecture de noms résolus
- [ ] `grep -rn "FROM match_registry" src/ scripts/ --include="*.py"` → seuls les writes légitimes restent

**Validation globale**
- [ ] `python -m pytest tests/ -q --ignore=tests/integration` → 0 fail

</details>

#### Wave 3 — Nettoyage wrappers + dead code (2 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 6 | `refactor(xuid): supprimer wrapper resolve_xuid_from_input` | A.6 | FAIBLE | 3 (streamlit_app.py, main_helpers.py, __init__.py) |
| 7 | `refactor(outcome): supprimer dead code get_outcome_name_fr` | B | FAIBLE | 3 (refdata.py, test_refdata.py, +1 nouveau test) |

<details>
<summary>✅ Checklist Wave 3 — à valider avant de démarrer Wave 4</summary>

**Commit 6 — Suppression wrapper XUID**
- [ ] `resolve_xuid_from_input` absent de `__all__` dans `__init__.py`
- [ ] Appels dans `streamlit_app.py` et `main_helpers.py` remplacés par l'appel direct
- [ ] Tests existants `TestResolveXuidInput` passent sans modification

**Commit 7 — Centralisation Outcomes**
- [ ] `get_outcome_name_fr` supprimé de `refdata.py`
- [ ] `OUTCOME_TO_FR` supprimé de `refdata.py`
- [ ] `tests/test_outcome_resolution.py` créé — 11 tests passent
- [ ] `tests/test_refdata.py` mis à jour (3 assertions liées à `get_outcome_name_fr` supprimées)
- [ ] Aucune régression dans les pages UI qui affichent les outcomes

**Validation globale**
- [ ] `python -m pytest tests/ -q --ignore=tests/integration` → 0 fail

</details>

#### Wave 4 — Migration schéma + helpers (2 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 8 | `feat(migration): supprimer highlight_events.gamertag + nettoyer resolver` | A.4 | ÉLEVé | 7 (resolver, engine_connections, _events.py, migration step, __init__, explorer_data, +tests) |
| 9 | `feat(analysis): helper resolve_medal_name depuis metadata.duckdb` | E | FAIBLE | 2 + 1 test |

> ✅ **Aucun backup manuel requis** : on travaille sur `shared_matches_v2.duckdb`.
> La v1 de prod reste intacte. En cas de problème : `rm shared_matches_v2.duckdb` et recommencer.

<details>
<summary>✅ Checklist Wave 4 — à valider avant de démarrer Wave 5</summary>

**Commit 8 — Suppression `highlight_events.gamertag`**
- [ ] Step de migration créé dans `src/data/migration/steps/`
- [ ] Migration idempotente (appliquée 2×, pas d'erreur)
- [ ] `highlight_events.gamertag` absent de `shared_matches_v2.duckdb` (vérifier avec `information_schema.columns`)
- [ ] Resolver : branche fallback `highlight_events` retirée de `_gamertag_resolver.py`
- [ ] `tests/test_gamertag_resolver.py` : 3 nouveaux tests passent (`no_highlight_fallback`, `migration_drops_column`, `migration_idempotent`)
- [ ] Tests d'intégration mis à jour (`INSERT highlight_events` sans colonne `gamertag`)

**Commit 9 — Helper médailles**
- [ ] `resolve_medal_name()` implémenté dans un module dédié
- [ ] `tests/test_medal_data.py` créé — 5 tests passent (FR, EN, inconnu, sans DB, cache)

**Validation globale**
- [ ] `python -m pytest tests/ -q --ignore=tests/integration` → 0 fail

</details>

#### Wave 5 — Audit de clôture + nettoyage couche traduction (2 commits)

> **Obligatoire avant de merger sur main.**

| # | Commit | Objectif |
|:-:|--------|----------|
| 10 | `chore(audit): vérification finale abstraction v6` | Confirmer qu'aucun accès direct résiduel n'a été oublié |
| 11 | `refactor(i18n): supprimer couche traduction assets obsolète` | Éliminer les dicts/JSON remplacés par metadata.duckdb |

---

##### Commit 10 — Audit de clôture

**Procédure d'audit** :

```bash
# 1. Recherche résiduelle : accès directs aux colonnes dupliquées
grep -rn "match_participants.*gamertag\|highlight_events.*gamertag" src/ scripts/ --include="*.py"
grep -rn "match_registry.*map_name\|match_registry.*playlist_name" src/ scripts/ --include="*.py"
grep -rn "killer_victim_pairs.*killer_gamertag\|killer_victim_pairs.*victim_gamertag" src/ scripts/ --include="*.py"

# 2. Vérifier que les vues existent dans la DB v2
python -c "
import duckdb
conn = duckdb.connect('data/warehouse/shared_matches_v2.duckdb')
views = conn.execute(\"SELECT view_name FROM information_schema.views WHERE table_schema = 'main'\").fetchall()
expected = {'v_gamertag_lookup', 'v_match_full', 'v_killer_victim_full'}
found = {v[0] for v in views}
missing = expected - found
print('Vues OK :', expected & found)
print('Vues MANQUANTES :', missing)
conn.close()
"

# 3. Lancer la suite de tests complète
python -m pytest tests/ -q --ignore=tests/integration

# 4. Vérifier que highlight_events.gamertag n'existe plus dans v2
python -c "
import duckdb
conn = duckdb.connect('data/warehouse/shared_matches_v2.duckdb')
cols = conn.execute(\"SELECT column_name FROM information_schema.columns WHERE table_name='highlight_events'\").fetchall()
cols = [c[0] for c in cols]
assert 'gamertag' not in cols, 'ERREUR : gamertag encore présent dans highlight_events'
print('OK : gamertag absent de highlight_events')
conn.close()
"

# 5. Bascule finale (si tous les critères OK)
# ⚠️ Arrêter l'app Streamlit AVANT la bascule (accès concurrent → risque de corruption DB)
# Ctrl+C sur le process streamlit, ou : pkill -f "streamlit run" (Linux/Mac)
# Windows : fermer la fenêtre du terminal Streamlit
mv data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v1_backup_$(date +%Y%m%d).duckdb
mv data/warehouse/shared_matches_v2.duckdb data/warehouse/shared_matches.duckdb
echo "Bascule OK — v2 est maintenant la prod"
```

**Critères de succès — Checklist commit 10 (condition de merge)**

> Toutes les cases doivent être cochées avant de merger sur `main`. Sans exception.

- [ ] `grep` gamertag → 0 hit non légitime (writes dans `_shared_writes.py` / `strategies.py` exclus)
- [ ] `grep` map_name/playlist_name → 0 hit non légitime (writes exclus)
- [ ] `grep` killer_gamertag/victim_gamertag → 0 hit non légitime
- [ ] Les 3 vues présentes dans `shared_matches_v2.duckdb`
- [ ] `highlight_events.gamertag` absent du schéma v2
- [ ] `python -m pytest tests/ -q --ignore=tests/integration` → 0 fail
- [ ] Commit 10 `chore(audit)` rédigé avec le résultat des greps en corps de message
- [ ] App Streamlit arrêtée **avant** les commandes `mv` de bascule
- [ ] Bascule v2 → prod exécutée (`mv shared_matches.duckdb ...backup... && mv shared_matches_v2.duckdb shared_matches.duckdb`)
- [ ] `db_profiles.json` remis sur le chemin par défaut (`shared_matches.duckdb`)
- [ ] Thought log mis à jour avec le bilan final

---

##### Commit 11 — Nettoyage couche traduction assets (Wave 5)

Une fois Commit 0 exécuté et validé, les données de traduction sont dans `metadata.duckdb`
(colonnes `name_fr`, `mode_name_fr`, `playlist_canonical_fr`). La couche de fallback statique
devient obsolète et doit être supprimée proprement.

**Inventaire complet de ce qui devient obsolète :**

| Artifact | Fichier | Obsolète car | Action |
|----------|---------|-------------|--------|
| `PLAYLIST_FR` dict (18 entrées) | `src/ui/translations.py` | Remplacé par `playlists.name_fr` | Supprimer |
| `PLAYLIST_EN` dict (14 entrées) | `src/ui/translations.py` | Remplacé par `playlists.name_en` | Supprimer |
| `PAIR_FR` dict (vide, kept for compat) | `src/ui/translations.py` | Déjà vide | Supprimer |
| `static/i18n/playlists_fr.json` | `static/i18n/` | Remplacé par `playlists.name_fr` | Supprimer |
| `static/i18n/playlists_en.json` | `static/i18n/` | Remplacé par `playlists.name_en` | Supprimer |
| `mode_translations` table | `metadata.duckdb` | Données migrées vers `game_variants.mode_name_fr` | DROP TABLE (migration step) |
| `playlist_translations` table | `metadata.duckdb` | Données migrées vers `playlists.name_fr` | DROP TABLE (migration step) |
| `MetadataResolver._resolve_from_table()` fallback dynamique | `src/data/sync/metadata_resolver.py` | Schéma stable : colonnes `name_en`/`name_fr` garanties | Simplifier (supprimer autodétection) |
| `_metadata_resolution.py` dynamic join | `src/data/repositories/_metadata_resolution.py` | Remplacé par `v_match_full` | Supprimer si Volet C complet |

**Ce qui reste (non obsolète) :**

| Artifact | Raison |
|----------|--------|
| `translate_pair_name()` + `modes_fr.json` | `pair_name` = "Arena:CTF on Aquarius" — logique combinatoire préfixe+mode nécessaire sauf si `pair_name_fr` **complètement** peuplé dans `playlist_map_mode_pairs` (à évaluer) |
| `static/i18n/modes_fr.json`, `modes_en.json` | Alimentent `translate_pair_name()` — à garder tant que la logique combinatoire est active |
| `static/i18n/weapons_fr/en.json`, `ranks_fr/en.json`, `citations_fr/en.json`, `awards_fr/en.json` | Non concernés par ce plan |
| `translate_playlist_name()` signature | Garder mais réécrire : lire `playlist_name_fr` depuis le DataFrame au lieu du JSON |

**Réécriture cible de `translate_playlist_name()`** :

```python
def translate_playlist_name(name: str | None, lang: str = "fr") -> str | None:
    """Traduit un nom de playlist.

    Depuis v6 : les traductions sont dans metadata.duckdb (colonnes name_fr/name_en
    de v_match_full). Cette fonction ne sert plus que de fallback pour les valeurs non
    résolues (NULL dans la vue) et les UUIDs bruts.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None
    if _is_uuid_like(s):
        logger.warning("playlist_name non résolu (UUID brut) : %s — metadata.duckdb incomplet ?", s)
        return label("playlists", "_unknown", lang=lang)
    # Si on reçoit un nom EN et que la DB est peuplée, on ne devrait pas arriver ici.
    # Log pour détecter les cas résiduels.
    logger.debug("translate_playlist_name fallback pour '%s' (hors DB) — à investiguer", s)
    # Fallback minimal : retourner tel quel (EN ou FR selon ce qui arrive)
    return s
```

**Spécifications de logging obligatoires (à implémenter dès Commit 0) :**

| Situation | Niveau | Message |
|-----------|--------|---------|
| `mode_name` extraction échoue (pas de `:` dans `public_name`) | `WARNING` | `"mode_name extraction failed for game_variant asset_id=%s public_name=%s"` |
| UUID non résolu dans `translate_playlist_name()` après migration | `WARNING` | `"playlist_name unresolved UUID %s — metadata.duckdb may be incomplete"` |
| `translate_playlist_name()` reçoit un nom **EN** non trouvé en DB (fallback dict) | `DEBUG` | `"translate_playlist_name fallback to static dict for '%s'"` |
| `MetadataResolver.resolve()` hits le fallback (non trouvé en DB) | `DEBUG` | `"MetadataResolver: %s/%s not found in metadata.duckdb"` (déjà présent) |
| Commit 0 populate : N erreurs écrites dans `metadata_populate_errors.txt` | `WARNING` | `"populate_metadata: %d errors written to metadata_populate_errors.txt"` |

**Tests requis (à ajouter dans `tests/test_metadata_i18n.py`) :**

```python
# 1. v_match_full colonnes EN préservées pour la logique métier
def test_v_match_full_playlist_name_is_english(): ...
    # assert "Firefight" in playlist_name (pas "Baptême du feu")

# 2. v_match_full colonnes FR disponibles
def test_v_match_full_playlist_name_fr_populated(): ...
    # assert playlist_name_fr is not NULL pour matchs avec playlist connue

# 3. mark_firefight() fonctionne après migration (EN dans playlist_name)
def test_mark_firefight_still_works_after_v_match_full(): ...
    # build DataFrame with playlist_name="Firefight" → is_firefight=True

# 4. translate_playlist_name() devient passthrough pour noms déjà traduits
def test_translate_playlist_name_uuid_logs_warning(caplog): ...
    # assert WARNING logged, returns "_unknown" label

# 5. mode_name extraction : 27 modes distincts
def test_game_variants_mode_name_count(): ...
    # query metadata.duckdb → assert len(DISTINCT mode_name) == 27

# 6. Régression : playlist_canonical_fr regroupe bien les 3 variantes BTB
def test_btb_variants_share_canonical_fr(): ...
    # "Big Team Battle", "Big Team Battle: Refresh", "Big Team Social"
    # → all playlist_canonical_fr == "Grande bataille en équipe"

# 7. metadata_populate_errors.txt créé si extraction échoue
def test_populate_errors_file_created_on_failure(tmp_path): ...
    # mock une game_variant sans ":" → assert fichier créé avec 1 ligne
```

<details>
<summary>✅ Checklist Commit 11</summary>

- [ ] `PLAYLIST_FR`, `PLAYLIST_EN`, `PAIR_FR` supprimés de `translations.py`
- [ ] `static/i18n/playlists_fr.json` et `playlists_en.json` supprimés
- [ ] `translate_playlist_name()` réécrite (passthrough + warning UUID)
- [ ] `mode_translations` et `playlist_translations` droppées via migration step
- [ ] `MetadataResolver._resolve_from_table()` nettoyé (autodétection supprimée)
- [ ] `tests/test_metadata_i18n.py` créé avec les 7 tests ci-dessus
- [ ] `python -m pytest tests/test_metadata_i18n.py -v` → tous verts
- [ ] Aucun import de `PLAYLIST_FR`/`PLAYLIST_EN` résiduel (`grep -rn PLAYLIST_FR src/`)
- [ ] `grep -rn "playlists_fr.json\|playlists_en.json" src/` → 0 hit

</details>

---

##### Commit 11b — Refactor système de traduction des modes (Wave 5)

> **Objectif** : remplacer la logique JSON à 3 niveaux imbriqués (`_prefixes` + mode keys +
> `_pairs` overrides) par 4 tables dans `metadata.duckdb`. Ajouter une langue = N INSERTs SQL,
> zéro ligne de Python.

**`refactor(i18n): migrer modes_fr/en.json vers metadata.duckdb`**

**Problème actuel** : `modes_fr.json` + `translate_pair_name()` (80L, `noqa: C901`) gère
434 combinaisons implicites (14 préfixes × 31 modes) via une cascade 6 étapes difficile à
auditer et à étendre. Ajouter l'espagnol = créer `modes_es.json` + tester toutes les variantes.

###### DDL — 4 nouvelles tables dans `metadata.duckdb`

```sql
-- Noms localisés des catégories ("Arena" → "Arène", "BTB" → "Grande bataille...")
CREATE TABLE IF NOT EXISTS mode_prefix_names (
    prefix_en  VARCHAR NOT NULL,
    lang       VARCHAR NOT NULL,
    name       VARCHAR NOT NULL,
    PRIMARY KEY (prefix_en, lang)
);

-- Noms localisés des modes ("Slayer" → "Assassin", "CTF" → "Capture du drapeau")
CREATE TABLE IF NOT EXISTS mode_name_tr (
    mode_en    VARCHAR NOT NULL,
    lang       VARCHAR NOT NULL,
    name       VARCHAR NOT NULL,
    PRIMARY KEY (mode_en, lang)
);

-- Overrides pour cas non combinatoires (préfixes trompeurs, noms complexes sans ":")
CREATE TABLE IF NOT EXISTS mode_pair_overrides (
    pattern    VARCHAR NOT NULL,   -- ex: "Assault:Neutral Bomb" (sans " on <carte>")
    lang       VARCHAR NOT NULL,
    name       VARCHAR NOT NULL,
    PRIMARY KEY (pattern, lang)
);

-- Séparateur de la combinaison préfixe + mode selon la langue
CREATE TABLE IF NOT EXISTS mode_lang_settings (
    lang       VARCHAR NOT NULL PRIMARY KEY,
    separator  VARCHAR NOT NULL DEFAULT ' : '
);
```

Volume : **14 + 31 + 11 entrées × 2 langues = 112 lignes** au total. Cache dict Python
au premier appel (process-level), zéro requête répétée ensuite.

###### Migration depuis les JSON existants

Script à créer : `scripts/migrate_modes_json_to_db.py`

```python
"""Migration one-shot : modes_fr.json + modes_en.json → metadata.duckdb."""
import json
from pathlib import Path
import duckdb

ROOT = Path(__file__).parent.parent
LANGS = {"fr": ROOT / "static/i18n/modes_fr.json",
         "en": ROOT / "static/i18n/modes_en.json"}

def migrate(db_path: Path) -> None:
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("CREATE TABLE IF NOT EXISTS mode_prefix_names ...")  # DDL ci-dessus
        # ... (idem autres tables)

        for lang, path in LANGS.items():
            data = json.loads(path.read_text(encoding="utf-8"))
            sep = data.get("_separator", ": ")
            conn.execute("INSERT OR REPLACE INTO mode_lang_settings VALUES (?, ?)", [lang, sep])

            for prefix_en, name in data.get("_prefixes", {}).items():
                conn.execute(
                    "INSERT OR REPLACE INTO mode_prefix_names VALUES (?, ?, ?)",
                    [prefix_en, lang, name],
                )
            for mode_en, name in data.items():
                if not mode_en.startswith("_"):
                    conn.execute(
                        "INSERT OR REPLACE INTO mode_name_tr VALUES (?, ?, ?)",
                        [mode_en, lang, name],
                    )
            for pattern, name in data.get("_pairs", {}).items():
                # Normaliser : strip " on <carte>" pour uniformiser les clés
                key = pattern.split(" on ", 1)[0].strip()
                conn.execute(
                    "INSERT OR REPLACE INTO mode_pair_overrides VALUES (?, ?, ?)",
                    [key, lang, name],
                )
        print(f"Migration OK → {db_path}")

if __name__ == "__main__":
    migrate(ROOT / "data/warehouse/metadata.duckdb")
```

###### Nouvelle `translate_pair_name()` — ~30L, sans `noqa`

```python
# src/ui/translations.py  — version post-migration

def translate_pair_name(name: str | None, lang: str = "fr") -> str | None:
    """Traduit un pair_name depuis metadata.duckdb.

    Résolution en 3 étapes :
    1. Override exact (mode_pair_overrides) pour les cas non combinatoires
    2. Combinatoire générique : préfixe localisé + séparateur + mode localisé
    3. Mode seul (sans catégorie)
    """
    if not name:
        return None
    s = str(name).strip()
    if _is_uuid_like(s):
        logger.warning("pair_name UUID non résolu : %s", s)
        return _mode_label("_unknown", lang)

    no_map = s.split(" on ", 1)[0].strip()
    candidate = _normalize_pair_case(no_map)

    # 1) Override
    if result := _mode_db_lookup("mode_pair_overrides", candidate, lang):
        return result

    # 2) Combinatoire
    if ":" in candidate:
        prefix_en, mode_en = candidate.split(":", 1)
        sep = _mode_sep(lang)
        prefix_loc = _mode_db_lookup("mode_prefix_names", prefix_en.strip(), lang) or prefix_en.strip()
        mode_loc = _mode_db_lookup("mode_name_tr", mode_en.strip(), lang) or mode_en.strip()
        return f"{prefix_loc}{sep}{mode_loc}"

    # 3) Mode seul
    return _mode_db_lookup("mode_name_tr", candidate, lang) or candidate


# Helpers cachés (process-level, chargés une fois par langue)
@lru_cache(maxsize=8)
def _load_mode_tables(lang: str) -> dict[str, dict[str, str]]:
    """Charge les 4 tables mode depuis metadata.duckdb en mémoire."""
    from src.data.repositories._db_context import get_metadata_conn
    conn = get_metadata_conn()
    return {
        "mode_prefix_names":  dict(conn.execute("SELECT prefix_en, name FROM mode_prefix_names WHERE lang=?", [lang]).fetchall()),
        "mode_name_tr":       dict(conn.execute("SELECT mode_en, name FROM mode_name_tr WHERE lang=?", [lang]).fetchall()),
        "mode_pair_overrides": dict(conn.execute("SELECT pattern, name FROM mode_pair_overrides WHERE lang=?", [lang]).fetchall()),
        "separator":          (conn.execute("SELECT separator FROM mode_lang_settings WHERE lang=?", [lang]).fetchone() or (" : ",))[0],
    }

def _mode_db_lookup(table: str, key: str, lang: str) -> str | None:
    return _load_mode_tables(lang).get(table, {}).get(key)

def _mode_sep(lang: str) -> str:
    return _load_mode_tables(lang).get("separator", ": ")
```

###### Nettoyage après migration

| Action | Artifact |
|--------|----------|
| Supprimer | `static/i18n/modes_fr.json` |
| Supprimer | `static/i18n/modes_en.json` |
| Supprimer | `load_domain("modes", ...)` dans `translate_pair_name()` |
| Supprimer | logique à 6 étapes de l'ancienne `translate_pair_name()` |
| Supprimer | `_normalize_pair_case()` si non utilisée ailleurs |
| Conserver | `static/i18n/ranks_fr/en.json`, `weapons_fr/en.json`, etc. — hors scope |
| Archiver  | `scripts/migrate_modes_json_to_db.py` → `scripts/_archive/` après exécution |

**Ajouter une nouvelle langue** (exemple : espagnol) :

```sql
-- C'est tout ce qu'il faut faire :
INSERT INTO mode_lang_settings VALUES ('es', ' : ');
INSERT INTO mode_prefix_names VALUES ('Arena', 'es', 'Arena');
INSERT INTO mode_prefix_names VALUES ('BTB', 'es', 'Gran Batalla de Equipos');
-- ... (14 prefixes + 31 modes + 11 overrides = 56 lignes)
```

###### Tests — `tests/test_translate_pair_name.py`

```python
# 1. Combinatoire générique
def test_arena_slayer_fr():
    assert translate_pair_name("Arena:Slayer on Aquarius", "fr") == "Arène : Assassin"

def test_btb_ctf_fr():
    assert translate_pair_name("BTB:CTF on Fragmentation", "fr") == "Grande bataille en équipe : Capture du drapeau"

def test_arena_slayer_en():
    assert translate_pair_name("Arena:Slayer on Aquarius", "en") == "Arena: Slayer"

# 2. Override _pairs
def test_assault_neutral_bomb_override():
    assert translate_pair_name("Assault:Neutral Bomb on Curfew", "fr") == "Arène : Bombe neutre"

def test_survive_undead_override():
    # Cas sans ":" — ne passe PAS par le combinatoire
    assert translate_pair_name("Survive The Undead 3.0 on TFF | Night Of The Undead", "fr") == "Survivre aux morts-vivants 3.0"

# 3. Mode seul (sans catégorie)
def test_mode_alone_fr():
    assert translate_pair_name("Slayer", "fr") == "Assassin"

# 4. Strip " on <carte>" avant lookup
def test_strip_map_suffix():
    assert translate_pair_name("Arena:Oddball on Recharge", "fr") == "Arène : Oddball"

# 5. UUID → warning + label inconnu
def test_uuid_logs_warning(caplog):
    import logging
    with caplog.at_level(logging.WARNING):
        result = translate_pair_name("a446725e-b281-414c-a21e-12345678abcd", "fr")
    assert "UUID" in caplog.text
    assert result is not None  # label "_unknown"

# 6. None / vide → None
def test_none_input():
    assert translate_pair_name(None, "fr") is None
    assert translate_pair_name("", "fr") is None

# 7. Cache process-level : 2 appels identiques = 1 seule requête DB
def test_lru_cache_hit(mocker):
    load_spy = mocker.spy(translations, "_load_mode_tables")
    translate_pair_name("Arena:Slayer on Aquarius", "fr")
    translate_pair_name("Arena:CTF on Recharge", "fr")
    assert load_spy.call_count == 1  # chargé une fois, pas deux

# 8. Nouvelle langue absente → fallback gracieux (retourne EN brut)
def test_unknown_lang_fallback():
    result = translate_pair_name("Arena:Slayer on Aquarius", "de")
    assert result == "Arena : Slayer"  # préfixe + mode EN (pas de crash)

# 9. Régression : tous les _pairs existants traduits correctement
@pytest.mark.parametrize("raw,expected", [
    ("Community:Fiesta Slayer on High Ground", "Fiesta"),
    ("Husky Raid:Assault on Urban Raid", "Husky Raid"),
    ("Arena:Shotty Snipes Slayer", "Fusils snipers à grenaille"),
])
def test_pairs_overrides_regression(raw, expected):
    assert translate_pair_name(raw, "fr") == expected
```

<details>
<summary>✅ Checklist Commit 11b</summary>

- [ ] `scripts/migrate_modes_json_to_db.py` créé et exécuté sans erreur
- [ ] 4 tables présentes dans `metadata.duckdb` : `mode_prefix_names`, `mode_name_tr`, `mode_pair_overrides`, `mode_lang_settings`
- [ ] `SELECT COUNT(*) FROM mode_prefix_names` → 28 (14 × 2 langues)
- [ ] `SELECT COUNT(*) FROM mode_name_tr` → 62 (31 × 2 langues)
- [ ] `SELECT COUNT(*) FROM mode_pair_overrides` → 22 (11 × 2 langues)
- [ ] Nouvelle `translate_pair_name()` implémentée (~30L, sans `noqa: C901`)
- [ ] `_normalize_pair_case()` supprimée ou conservée si autre usage
- [ ] `load_domain("modes", ...)` supprimé de `translations.py`
- [ ] `static/i18n/modes_fr.json` et `modes_en.json` supprimés
- [ ] `scripts/migrate_modes_json_to_db.py` archivé → `scripts/_archive/`
- [ ] `tests/test_translate_pair_name.py` créé avec les 9 tests ci-dessus
- [ ] `python -m pytest tests/test_translate_pair_name.py -v` → tous verts
- [ ] `grep -rn "modes_fr.json\|modes_en.json\|load_domain.*modes" src/` → 0 hit
- [ ] Test de régression : `translate_pair_name("Arena:Slayer on Aquarius", "fr")` == `"Arène : Assassin"`

</details>

#### Récapitulatif

```
Pré-requis :
  Commit 0   ──  chore(metadata): peupler maps/playlists/game_variants dans metadata.duckdb

Wave 1 : Fondation (vues SQL + refactor resolver)
  Commit 1   ──  feat(db): vues v_gamertag_lookup + v_match_full + v_killer_victim_full
  Commit 2   ──  refactor(resolver): cascade gamertag via v_gamertag_lookup

Wave 2 : Migration consommateurs
  Commit 3   ──  refactor(gamertag): consommateurs directs → vue
  Commit 4   ──  refactor(kv): killer_victim_repo → v_killer_victim_full
  Commit 5   ──  refactor(assets): match_registry → v_match_full

Wave 3 : Nettoyage
  Commit 6   ──  refactor(xuid): supprimer wrapper inutile
  Commit 7   ──  refactor(outcome): supprimer dead code

Wave 4 : Migration schéma + médailles
  Commit 8   ──  feat(migration): supprimer highlight_events.gamertag
  Commit 9   ──  feat(analysis): helper resolve_medal_name

Wave 5 : Audit + nettoyage couche i18n
  Commit 10  ──  chore(audit): vérification finale abstraction v6
  Commit 11  ──  refactor(i18n): supprimer dicts/JSON playlists obsolètes
  Commit 11b ──  refactor(i18n): migrer modes_fr/en.json → metadata.duckdb
```

### Dépendances entre commits

```
Commit 0 (metadata pré-requis)
  └──→ Commit 1 (vues SQL)
        └──→ Commit 11b (migration modes → metadata.duckdb)

Commit 1 (vues SQL)
  ├──→ Commit 2 (resolver)
  │       └──→ Commit 3 (gamertag consommateurs)
  │               └──→ Commit 8 (drop highlight_events.gamertag)
  ├──→ Commit 4 (kv consommateurs)
  └──→ Commit 5 (assets consommateurs)
        └──→ Commit 11 (nettoyage playlists i18n)

Commit 6 (xuid wrapper)     ← indépendant
Commit 7 (outcomes)          ← indépendant
Commit 9 (médailles)         ← indépendant

Commit 10 (audit de clôture) ← dépend de TOUS les commits précédents
Commit 11 (playlists i18n)   ← après Commit 5 (assets via v_match_full)
Commit 11b (modes i18n)      ← après Commit 0 (metadata.duckdb peuplée)
                             ET Commit 11 (translations.py modifié en parallèle)
```

---

## Tests par Phase — TOUS les tests attendus

### Commit 1 — Vues SQL

**Fichier : `tests/test_resolution_views.py`** (nouveau)

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_v_gamertag_lookup_prefers_xuid_aliases` | Insert dans xuid_aliases (gt="Nouveau") et match_participants (gt="Ancien") → vue retourne "Nouveau" |
| 2 | `test_v_gamertag_lookup_fallback_match_participants` | XUID absent de xuid_aliases → retourne gt de match_participants |
| 3 | `test_v_gamertag_lookup_excludes_null` | Entrée sans gamertag nulle part → absente de la vue |
| 4 | `test_v_gamertag_lookup_full_outer_join_covers_both` | XUID dans xuid_aliases seul + XUID dans match_participants seul → les deux apparaissent |
| 5 | `test_v_match_full_resolves_map_name` | Insert map_id dans match_registry + public_name dans metadata → vue retourne public_name |
| 6 | `test_v_match_full_fallback_to_registry_name` | map_id absent de metadata → vue retourne match_registry.map_name |
| 7 | `test_v_match_full_all_four_names_resolved` | playlist, pair, game_variant resolus aussi |
| 8 | `test_v_killer_victim_full_resolves_gamertags` | killer_xuid dans xuid_aliases → killer_gamertag résolu à jour |
| 9 | `test_v_killer_victim_full_fallback_to_kv_gamertag` | killer_xuid absent de v_gamertag_lookup → retourne kv.killer_gamertag |
| 10 | `test_v_killer_victim_full_last_resort_xuid` | Les deux fallbacks null → retourne killer_xuid brut |
| 11 | `test_v_match_full_fr_columns_not_null` | Avec metadata peuplée : `map_name_fr`, `playlist_name_fr`, `game_variant_name_fr` non NULL pour un match avec IDs connus |

**Logging vérifié dans migrations.py** :
```python
logger.info("Vue v_gamertag_lookup créée dans shared_matches.duckdb")
logger.info("Vue v_match_full créée dans shared_matches.duckdb")
logger.info("Vue v_killer_victim_full créée dans shared_matches.duckdb")
```

### Commit 2 — Refactor Resolver

**Fichier : `tests/test_gamertag_resolver.py`** (nouveau, dédié)

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_resolve_gamertag_uses_view` | `repo.resolve_gamertag(xuid)` → retourne le gamertag de xuid_aliases même si match_participants a un autre nom |
| 2 | `test_resolve_gamertag_fallback_highlight_events` | XUID absent de la vue, présent dans highlight_events avec `"juan1\x00\x00"` → retourne `"juan1"` (transitoire) |
| 3 | `test_resolve_gamertag_returns_none` | XUID absent partout → None |
| 4 | `test_load_match_player_gamertags_prefers_aliases` | Joueur dans les deux sources → retourne nom xuid_aliases |
| 5 | `test_load_match_player_gamertags_merges_sources` | 3 joueurs répartis dans les sources → tous résolus |
| 6 | `test_load_match_player_gamertags_empty_match` | match_id inconnu → dict vide |
| 7 | `test_resolve_gamertags_batch` | Batch de 5 XUIDs → dict complet |

**Logging vérifié** :
```python
logger.debug("resolve_gamertag(%s): source=v_gamertag_lookup → %s", xuid, gt)
logger.debug("resolve_gamertag(%s): source=highlight_events → %s", xuid, gt)
logger.warning("resolve_gamertag(%s): aucune source", xuid)
logger.debug("load_match_player_gamertags(%s): %d joueurs résolus", match_id, len(result))
```

### Commit 3 — Gamertag consommateurs directs

**Fichier : `tests/test_explorer_data.py`** (nouveau)

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_get_all_gamertags_from_view` | Retourne gamertags de xuid_aliases + fallback match_participants — pas de SELECT highlight_events |
| 2 | `test_resolve_gamertag_to_xuid_from_view` | Résout via la vue |
| 3 | `test_get_all_gamertags_empty_db` | DB vide → liste vide, pas d'exception |

### Commit 4 — Killer/Victim consommateurs

**Fichier : `tests/test_killer_victim_views.py`** (nouveau)

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_load_killer_victim_pairs_uses_view` | `repo.load_career_antagonists()` retourne des gamertags mis à jour via la vue |
| 2 | `test_load_kv_pairs_fallback_snapshot` | Joueur absent de xuid_aliases → gamertag du snapshot dans kv_pairs |
| 3 | `test_teammates_service_uses_kv_view` | `build_teammate_connections()` résout les noms via la vue |

### Commit 5 — Assets consommateurs

**Ajout dans `tests/test_resolution_views.py`** :

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_mv_player_matches_uses_v_match_full` | La vue matérialisée retourne les noms résolus depuis metadata |
| 2 | `test_backfill_detection_uses_v_match_full` | Requête NULL detection passe par la vue |

### Commit 6 — Suppression wrapper XUID

| # | Test | Fichier | Objectif |
|:-:|------|---------|----------|
| 1 | `test_resolve_xuid_from_input_not_exported` | `test_app_sidebar.py` | Vérifie symbole absent de `__all__` |
| 2 | Tests existants `TestResolveXuidInput` | `test_app_sidebar.py` | Passent sans régression |

### Commit 7 — Centralisation Outcomes

**Fichier : `tests/test_outcome_resolution.py`** (nouveau)

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_get_outcome_map_fr_all_codes` | `get_outcome_map("fr")` contient {1, 2, 3, 4} |
| 2 | `test_get_outcome_map_en_all_codes` | Idem EN |
| 3 | `test_get_outcome_map_default_lang` | Sans arg → labels FR |
| 4 | `test_resolve_outcome_win` | `(2, "Victoire", "#3DFFB5")` |
| 5 | `test_resolve_outcome_loss` | `(3, "Défaite", "#FF4D6D")` |
| 6 | `test_resolve_outcome_tie` | `(1, "Égalité", couleur violet)` |
| 7 | `test_resolve_outcome_dnf` | `(4, "Non terminé", couleur violet)` |
| 8 | `test_resolve_outcome_none` | `(None, "-", couleur slate)` |
| 9 | `test_resolve_outcome_missing_key` | `{}` → `(None, "-", couleur slate)` |
| 10 | `test_outcome_enum_matches_map_keys` | `set(Outcome) == set(get_outcome_map().keys())` |
| 11 | `test_dead_code_removed` | `refdata` n'exporte plus `get_outcome_name_fr` ni `OUTCOME_TO_FR` |

**Mise à jour** : `tests/test_refdata.py` — supprimer les 3 assertions sur `get_outcome_name_fr`.

### Commit 8 — Suppression highlight_events.gamertag

**Ajout dans `tests/test_gamertag_resolver.py`** :

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_resolve_gamertag_no_highlight_fallback` | Après suppression, cascade s'arrête à la vue |
| 2 | `test_migration_drops_gamertag_column` | Applique migration → colonne absente |
| 3 | `test_migration_idempotent` | Migration 2 fois → pas d'erreur |

**Mise à jour tests intégration** :
- `tests/integration/test_app_data_to_chart_flow.py` — retirer `gamertag` des INSERT highlight_events
- `tests/integration/test_app_partial_data_to_chart_flow.py` — idem

### Commit 9 — Helper médailles

**Fichier : `tests/test_medal_data.py`** (nouveau)

| # | Test | Objectif |
|:-:|------|----------|
| 1 | `test_resolve_medal_name_known_fr` | Résolution nom FR depuis metadata |
| 2 | `test_resolve_medal_name_known_en` | Idem EN |
| 3 | `test_resolve_medal_name_unknown` | ID inconnu → `str(id)` |
| 4 | `test_resolve_medal_name_no_db` | DB absente → fallback, pas d'exception |
| 5 | `test_load_medal_names_cached` | 2 appels → 1 seule requête (mock) |

---

## Résumé Quantitatif

| Métrique | Valeur |
|----------|:------:|
| Waves | 5 (+ 1 pré-requis Commit 0 hors wave) |
| Commits | 12 (Commit 0 + Commits 1–9 + Commit 10 + Commits 11 et 11b) |
| Vues SQL créées | 3 (`v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`) |
| Vue matérialisée modifiée | 1 (`mv_player_matches`) |
| Fichiers production modifiés | ~25 |
| Fichiers production créés | 3 |
| Fichiers tests créés | 6 |
| Tests nouveaux | 43 |
| Tests mis à jour | 5 |
| Migrations DuckDB | 2 (vues + drop column) |
| Emplacements SQL couverts | ~320 (audit complémentaire) |
| Emplacements Polars impactés | 259 (aucun changement code requis) |
| Audit de clôture | Wave 5, commit 10 (grep + tests + vérification DB) |

### Bénéfice de l'abstraction complète

| Avant | Après |
|-------|-------|
| `gamertag` dupliqué dans 3 tables | Source unique : `v_gamertag_lookup` |
| `*_name` dupliqué dans match_registry + metadata | Source unique : `v_match_full` |
| `killer/victim_gamertag` figé au sync | Résolu live via `v_killer_victim_full` |
| Changer source gamertag = 33 fichiers | Modifier `v_gamertag_lookup` (1 vue) |
| Changer source noms assets = 35 fichiers | Modifier `v_match_full` (1 vue) |
| 3 fonctions outcome fragmentées | 1 canonique + 1 helper UI |
| 5 cascades gamertag dans le resolver | 1 requête SELECT sur la vue |
| Pas de helper médailles DuckDB | `resolve_medal_name()` centralisé |
| 259 opérations Polars lisant des noms | Aucun changement — mêmes colonnes, meilleures valeurs |
| **~320 emplacements lisant des colonnes dupliquées directement** | **Tous passent par 3 vues + 4 fonctions Python** |
