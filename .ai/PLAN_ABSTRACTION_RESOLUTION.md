# PLAN v5.8 — Couche d'Abstraction Complète pour la Résolution d'IDs

> **Version** : v5.8
> **Branche** : `refactor/id-resolution-cleanup` (créée à partir de `analysis/weapon-parser-rewrite` = v5.7)
> Créé le 2026-03-14 · Mis à jour le 2026-03-14.
> Couvre : cascade gamertag, noms d'assets, paires killer/victim, outcomes, médailles.

### Objectifs v5.8

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
    SELECT DISTINCT xuid, gamertag
    FROM match_participants
    WHERE gamertag IS NOT NULL
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
| `teammates_service.py:149` | `SELECT xuid FROM match_participants WHERE gamertag = ?` | → `SELECT xuid FROM v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?)` |
#### 🆕 Fichiers supplémentaires découverts

| Fichier | Requête actuelle | Migration |
|---------|-----------------|----------|
| `_weapon_kills_repo.py` | 4× `highlight_events.gamertag` + 2× `match_participants.gamertag` | → JOIN `v_gamertag_lookup` |
| `career_encounters_data.py` | 3× `match_participants.gamertag` | → JOIN `v_gamertag_lookup` |
| `_discord_queries.py` | `match_participants.gamertag` combiné | → JOIN `v_gamertag_lookup` |
| `_calibration_loaders.py` | 3× requêtes avec gamertag | → JOIN `v_gamertag_lookup` |
Les fichiers qui passent par le repo (`_roster_loader.py`, `_encounter_loader.py`,
`_events_repo.py`) sont OK — ils seront mis à jour indirectement via le refactoring du mixin.

### A.4 Suppression `highlight_events.gamertag`

Après A.1–A.3 :
- La vue `v_gamertag_lookup` ne l'utilise pas
- Le resolver ne l'utilise plus (ou en fallback transitoire)
- `explorer_data.py` migré sur la vue

→ Migration de schéma pour supprimer la colonne (voir Phase D pour les détails migration).

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
4. **Ajouter un alias** dans `refdata.py` qui pointe vers `get_outcome_map` pour que les
   modules domain puissent l'utiliser sans importer `i18n` :
   ```python
   # Dans refdata.py — pour usage hors UI (scripts, analysis)
   def get_outcome_label(outcome: int, lang: str = "fr") -> str:
       """Retourne le label traduit d'un code outcome."""
       from src.ui.i18n import get_outcome_map
       return get_outcome_map(lang).get(outcome, "?")
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
-- Requiert ATTACH 'metadata.duckdb' AS meta (déjà fait dans le repo)
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
    -- Noms résolus : metadata prioritaire, match_registry fallback
    COALESCE(m.public_name, mr.map_name)            AS map_name,
    COALESCE(p.public_name, mr.playlist_name)        AS playlist_name,
    COALESCE(pp.public_name, mr.pair_name)           AS pair_name,
    COALESCE(gv.public_name, mr.game_variant_name)   AS game_variant_name,
FROM match_registry mr
LEFT JOIN meta.maps m ON mr.map_id = m.asset_id
LEFT JOIN meta.playlists p ON mr.playlist_id = p.asset_id
LEFT JOIN meta.map_mode_pairs pp ON mr.pair_id = pp.asset_id
LEFT JOIN meta.game_variants gv ON mr.game_variant_id = gv.asset_id;
```

> La vue expose des colonnes avec les **mêmes noms** (`map_name`, `playlist_name`, etc.)
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

## Volet E — Paires Killer/Victim (killer_gamertag, victim_gamertag)

### E.0 Audit de surface (révisé)

| Catégorie | Fichiers | Emplacements |
|-----------|:--------:|:------------:|
| Schéma + Modèles | 4 | `_engine_connections.py`, `_batch_columns.py`, `_kv_types.py`, `models.py` |
| Écritures (INSERT) | 2 | `_shared_writes.py:75`, `strategies.py:183` |
| Lectures SQL | 2 prod | `_killer_victim_repo.py:73`, `teammates_service.py:76` |
| Polars GROUP BY / Agrégations | 5 prod | `_killer_victim_repo.py`, `_killer_victim_polars.py`, `_antagonist_kv.py`, `match_view_players_nemesis.py`, `friends_impact_heatmap.py` |
| Tests | 10 | Données de test, assertions, schéma |
| **Total surface** | **14 prod + 10 tests** | **82+ emplacements** |

#### 🆕 Découverts par audit complémentaire

| Fichier | Hits | Détail |
|---------|:----:|--------|
| `career_encounters_data.py` | 4 | `killer_victim_pairs` × 4 (bypass le repo) |
| `_encounter_loader.py` | 1 | `killer_victim_pairs` direct (supplémentaire) |
| `weapon_extraction_service.py` | 2 | `killer_gamertag` / `victim_gamertag` dans extraction |

### E.1 Problème

Les colonnes `killer_gamertag` et `victim_gamertag` dans `killer_victim_pairs` sont des
**snapshots figés** au moment du sync, exactement comme `match_participants.gamertag`.
Si un joueur change de gamertag, les paires K/V affichent l'ancien nom.

De plus, la source est `highlight_events.gamertag` (données brutes, possiblement corrompues
avec NUL bytes) → les paires K/V héritent des défauts de la source.

### E.2 Vue SQL — `v_killer_victim_full`

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

### E.3 Migration des consommateurs

| Fichier | Requête actuelle | Migration |
|---------|-----------------|-----------|
| `_killer_victim_repo.py:73` | `SELECT ... FROM killer_victim_pairs` | → FROM `v_killer_victim_full` |
| `teammates_service.py:76` | `COALESCE(killer_gamertag, killer_xuid::TEXT)` | → FROM `v_killer_victim_full` (le COALESCE est dans la vue) |

Les opérations Polars en aval (`_killer_victim_polars.py`, `_antagonist_kv.py`,
`match_view_players_nemesis.py`) ne changent **rien** — elles lisent des colonnes
nommées `killer_gamertag` / `victim_gamertag` qui sortent maintenant de la vue avec
des noms à jour.

### E.4 Conservation des colonnes dans la table

Comme pour `match_participants.gamertag` : **on garde** les colonnes `killer_gamertag`
et `victim_gamertag` dans la table brute comme fallback dans la vue. Elles ne seront
supprimées que quand `xuid_aliases` couvrira 100% des XUIDs connus.

### E.5 Mise à jour du transformateur (écriture)

Pas de changement immédiat : le sync continue d'écrire `killer_gamertag` / `victim_gamertag`
dans `killer_victim_pairs`. C'est un cache — la vue résout toujours le nom courant en priorité.

> Future optimisation : arrêter d'écrire ces colonnes et compter uniquement sur la vue.
> Mais ce n'est pas bloquant maintenant.

---

## Volet D — Médailles (medal_name_id → nom)

### D.1 État actuel

Pas de helper centralisé dans le code Python. Les noms de médailles sont :
- Soit lus directement depuis l'API SPNKr
- Soit affichés comme identifiant brut en UI
- Soit résolus via `load_medal_name_maps()` dans `src/ui/medals.py` (JSON statique `static/medals/`)

> L'audit complémentaire a confirmé **69 emplacements** Polars utilisant `medal_name` dans l'UI,
> tous alimentés par le JSON statique. Ce helper est fonctionnel mais ne passe pas par DuckDB.

### D.2 Plan

Créer `src/analysis/_medal_data.py` (analogue à `_weapon_data.py`) :

```python
def resolve_medal_name(medal_name_id: int, lang: str = "fr") -> str:
    """Résout un medal_name_id en nom lisible depuis metadata.duckdb."""
```

Alimenté par la table `medals` de `metadata.duckdb` (si elle existe), sinon fallback `str(id)`.

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

L'audit a vérifié 8 patterns de résolution non couverts par le plan v5.8 :

| Pattern | Statut | Centralisé ? | Recommandation |
|---------|:------:|:------------:|----------------|
| Team ID (0–8 → "Eagle"/"Cobra") | ✅ Fonctionnel | `src/config.py` | Garder séparé (domaine config) |
| Rank/CSR Tier → label FR | ✅ Fonctionnel | `src/ui/i18n/ranks.py` | Garder séparé (domaine skill) |
| Playlist Groups (6 groupes) | ✅ Fonctionnel | `src/analysis/playlist_groups.py` | Catégorisation, pas résolution |
| Weapon ID → nom | ✅ Fonctionnel | `src/analysis/_weapon_data.py` | Déjà centralisé (v5.7) |
| Commendation rules | ✅ Fonctionnel | `metadata.duckdb` + Python custom | Complexe, garder séparé |
| Medal ID → nom | ✅ Fonctionnel | `src/ui/medals.py` (JSON) | Helper DuckDB en v5.8 (Volet D) |
| Personal Score ID → nom | ✅ Fonctionnel | `src/data/domain/_refdata_personal_scores.py` | Garder séparé |
| Label normalization | ✅ Fonctionnel | `src/app/helpers.py` | Pas une résolution ID |

> **Aucune lacune critique** : tous les patterns `*_id` → `*_name` sont déjà implémentés.
> Le plan v5.8 est **complet** pour son scope (centralisation + abstraction SQL).

---

## Stratégie DB : Travailler sur une copie `shared_matches_v2.duckdb`

### Principe

Plutôt que de modifier la DB de production directement, tout le travail v5.8 se fait
sur une **copie `shared_matches_v2.duckdb`**. La prod n'est jamais touchée jusqu'au
bascule final.

```
│ Production (intacte)            │  Développement v5.8              │
│ shared_matches.duckdb           │  shared_matches_v2.duckdb       │
│ Version v5.7 (current)          │  Version v5.8 (en cours)        │
│ Utilisée par l'app Streamlit    │  Utilisée sur la branche refactor│
│ Ne jamais modifier              │  Toutes les vues + DROP column  │
```

### Setup (avant Wave 1)

```bash
# 1. Copier la DB de prod
cp data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v2.duckdb

# 2. Pointer db_profiles.json vers la v2 pour le dev
# (modifier le champ "shared_db_path" ou équivalent)
# OU utiliser une variable d'environnement :
export LEVELUP_SHARED_DB=data/warehouse/shared_matches_v2.duckdb
```

> Vérifier comment `db_profiles.json` définit le chemin de `shared_matches.duckdb`
> et adapter la stratégie de surcharge en conséquence.

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
# Création de la branche v5.8
git checkout analysis/weapon-parser-rewrite
git checkout -b refactor/id-resolution-cleanup

# Setup DB v2 (avant de commencer)
cp data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v2.duckdb
```

Le travail est découpé en **waves** (groupes de commits) pour limiter le risque
et permettre de valider à chaque étape.

#### Wave 1 — Fondation : vues SQL + refactor cascade (2 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 1 | `feat(db): vues v_gamertag_lookup + v_match_full + v_killer_victim_full` | A.1 + C.2.1 + E.2 | MOYEN | 2 (migrations.py + _engine_connections.py) |
| 2 | `refactor(resolver): cascade gamertag via v_gamertag_lookup` | A.2 | MOYEN | 1 (_gamertag_resolver.py) |

> Après cette wave : les vues existent, le resolver les utilise, mais les anciens
> chemins directs fonctionnent encore.

#### Wave 2 — Migration des consommateurs directs (3 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 3 | `refactor(gamertag): consommateurs directs → v_gamertag_lookup` | A.3 | FAIBLE | 4 (explorer_data, teammates_impact, teammates_service, events_repo) |
| 4 | `refactor(kv): killer_victim_repo + teammates_service → v_killer_victim_full` | E.3 | FAIBLE | 2 (_killer_victim_repo.py, teammates_service.py) |
| 5 | `refactor(assets): requêtes directes match_registry → v_match_full` | C.2.2–C.2.3 | FAIBLE | 4 (strategies.py, detection.py, _data_loader.py, migrations.py pour mv_player_matches) |

> Après cette wave : **tous les consommateurs** passent par les vues.
> Les tables brutes ne sont plus lues directement pour des noms résolus.

#### Wave 3 — Nettoyage wrappers + dead code (2 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 6 | `refactor(xuid): supprimer wrapper resolve_xuid_from_input` | A.6 | FAIBLE | 3 (streamlit_app.py, main_helpers.py, __init__.py) |
| 7 | `refactor(outcome): supprimer dead code get_outcome_name_fr` | B | FAIBLE | 3 (refdata.py, test_refdata.py, +1 nouveau test) |

#### Wave 4 — Migration schéma + helpers (2 commits)

| # | Commit | Volet | Risque | Fichiers modif. |
|:-:|--------|:-----:|:------:|:---------------:|
| 8 | `feat(migration): supprimer highlight_events.gamertag + nettoyer resolver` | A.4 | ÉLEVé | 7 (resolver, engine_connections, _events.py, migration step, __init__, explorer_data, +tests) |
| 9 | `feat(analysis): helper resolve_medal_name depuis metadata.duckdb` | D | FAIBLE | 2 + 1 test |

> ✅ **Aucun backup manuel requis** : on travaille sur `shared_matches_v2.duckdb`.
> La v1 de prod reste intacte. En cas de problème : `rm shared_matches_v2.duckdb` et recommencer.

#### Wave 5 — Audit de clôture (1 commit)

> **Obligatoire avant de merger sur main.**

| # | Commit | Objectif |
|:-:|--------|----------|
| 10 | `chore(audit): vérification finale abstraction v5.8` | Confirmer qu'aucun accès direct résiduel n'a été oublié |

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
mv data/warehouse/shared_matches.duckdb data/warehouse/shared_matches_v1_backup_$(date +%Y%m%d).duckdb
mv data/warehouse/shared_matches_v2.duckdb data/warehouse/shared_matches.duckdb
echo "Bascule OK — v2 est maintenant la prod"
```

**Critères de succès** :
- [ ] `grep` ne trouve aucun accès direct non légitime (les writes dans `_shared_writes.py` / `strategies.py` sont légitimes)
- [ ] Les 3 vues sont présentes dans `shared_matches.duckdb`
- [ ] `highlight_events.gamertag` est absent du schéma
- [ ] Tous les tests passent (0 fail)
- [ ] Relancer les agents d'audit si `grep` retourne des hits inattendus

#### Récapitulatif

```
Wave 1 : Fondation (vues SQL + refactor resolver)
  Commit 1  ──  feat(db): vues v_gamertag_lookup + v_match_full + v_killer_victim_full
  Commit 2  ──  refactor(resolver): cascade gamertag via v_gamertag_lookup

Wave 2 : Migration consommateurs
  Commit 3  ──  refactor(gamertag): consommateurs directs → vue
  Commit 4  ──  refactor(kv): killer_victim_repo → v_killer_victim_full
  Commit 5  ──  refactor(assets): match_registry → v_match_full

Wave 3 : Nettoyage
  Commit 6  ──  refactor(xuid): supprimer wrapper inutile
  Commit 7  ──  refactor(outcome): supprimer dead code

Wave 4 : Migration schéma + médailles
  Commit 8  ──  feat(migration): supprimer highlight_events.gamertag
  Commit 9  ──  feat(analysis): helper resolve_medal_name

Wave 5 : Audit de clôture
  Commit 10 ──  chore(audit): vérification finale abstraction v5.8
```

### Dépendances entre commits

```
Commit 1 (vues SQL)
  ├──→ Commit 2 (resolver)
  │       └──→ Commit 3 (gamertag consommateurs)
  │               └──→ Commit 8 (drop highlight_events.gamertag)
  ├──→ Commit 4 (kv consommateurs)
  └──→ Commit 5 (assets consommateurs)

Commit 6 (xuid wrapper)     ← indépendant
Commit 7 (outcomes)          ← indépendant
Commit 9 (médailles)         ← indépendant

Commit 10 (audit de clôture) ← dépend de TOUS les commits précédents
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
| Waves | 5 |
| Commits | 10 |
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
