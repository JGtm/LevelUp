# Plan détaillé — Weapon Parser v2 (rewrite)

> Date : 2026-03-12  
> Branche cible : `analysis/weapon-parser-rewrite`  
> Statut : PLAN — en attente de validation avant implémentation

---

## Table des matières

1. [Objectifs et non-objectifs](#1-objectifs-et-non-objectifs)
2. [Architecture cible](#2-architecture-cible)
3. [Phase 0 — Exploration non-POV](#3-phase-0--exploration-non-pov)
4. [Phase 1 — Migration schéma (reconciled_as)](#4-phase-1--migration-schéma)
5. [Phase 2 — Parser v2 (couche pure)](#5-phase-2--parser-v2-couche-pure)
6. [Phase 3 — Service v2 (orchestration)](#6-phase-3--service-v2-orchestration)
7. [Phase 4 — Réconciliation API découplée](#7-phase-4--réconciliation-api-découplée)
8. [Phase 5 — Repository v2](#8-phase-5--repository-v2)
9. [Stratégie de logging](#9-stratégie-de-logging)
10. [Stratégie de tests](#10-stratégie-de-tests)
11. [Plan de migration des données](#11-plan-de-migration-des-données)
12. [Risques et mitigations](#12-risques-et-mitigations)
13. [Checklist de livraison](#13-checklist-de-livraison)

---

## 1. Objectifs et non-objectifs

### Objectifs

| # | Objectif | Mesure de succès |
|---|----------|-----------------|
| O1 | **Unifier l'attribution** : tous les joueurs du lobby utilisent le même pipeline (fire events quand disponible, formula_a en fallback) | Taux de `confidence=high` ≥ 70 % sur tout le lobby (vs ~12.5 % actuellement) |
| O2 | **Éliminer la perte de données film** : `weapon_id` ne doit jamais être écrasé par un sentinel | 0 overwrites de hex réels ; sentinels dans `reconciled_as` uniquement |
| O3 | **Claim-and-remove** : chaque fire event ne peut être attribué qu'à un seul kill | 0 doublons `(fire_event → kill)` vérifiable par test |
| O4 | **Cross-chunk** : un kill à la frontière de chunk trouve les fire events du chunk précédent | Test avec kill à t=19050ms corrélé à fire event t=18800ms |
| O5 | **Logging structuré** : chaque décision d'attribution est traçable sans relancer le parser | Logs JSON par match exportables pour audit |
| O6 | **Tests déterministes** : toute régression produit un test rouge immédiat | ≥ 150 tests, couverture ≥ 90 % sur parser + service |
| O7 | **Corriger Bug F** : `_inject_missing_sentinels` ne reclassifie plus les kills avec hex réel | 0 kill `confidence=low` + hex réel converti en sentinel |

### Non-objectifs (hors scope du rewrite)

| # | Exclusion | Raison |
|---|-----------|--------|
| N1 | ~~Attribution des adversaires~~ | **Conditionnel Phase 0** — si la résolution de `player_index` s'avère fiable pour tous les joueurs (pas seulement les coéquipiers), les adversaires peuvent rentrer dans le scope. Décision prise en fin de Phase 0 uniquement. |
| N2 | ~~Parsing de melee events film~~ | **Piste Phase 0** — à explorer en même temps que les fire events non-POV. Si les melee events sont détectables et fiables en POV, ils remplacent la détection par médailles pour le POV. Décision en fin de Phase 0. |
| N3 | Fusion `weapon_kills` ↔ `killer_victim_pairs` | **JAMAIS.** Décision architecturale définitive et documentée (2026-03-12) : les deux tables restent séparées, liées par la clé naturelle `(match_id, xuid, time_ms)`. |
| N4 | Découverte de nouveaux weapon_ids | Tâche d'investigation film distincte, pas du rôle du parser. Le parser stocke les hex inconnus as-is pour résolution future. |
| N5 | Optimisation ProcessPoolExecutor | Évaluer après v2, seulement si CPU > 40 % mesuré en production. |
| N6 | **Scan grenade/explosive events film** | Hors scope v2 — architecture préparée : `attribution_path` réserve la valeur `"grenade_event"` et `correlate_kills()` expose un point d'extension explicite avant le fallback formula_a (voir §5.2). Implémentation conditionnelle à la découverte du marqueur film grenades. |

---

## 2. Architecture cible

### 2.1 Modules et responsabilités

```
src/analysis/
├── weapon_parser.py              # RÉÉCRIT — Couche pure, 0 I/O, 0 DB  (≤ 600 L)
│   ├── scan_fire_events_all()        # Section 2 — scanner match-level (0 filtre pi, tous joueurs)
│   ├── scan_melee_events()           # Section 2 — melee events film (marqueur 0xd340)
│   ├── scan_grenade_events_all()     # FUTUR (v3+) — grenade/explosive events film (marqueur à découvrir)
│   ├── scan_formula_a()              # Section 1 raw — instance handles (compat tests)
│   ├── scan_formula_a_ns()           # Section 1 NS layer — TYPE IDs (fallback formula_a)
│   ├── build_weapon_timeline()       # Timeline raw par chunk/pi (compat tests)
│   ├── build_weapon_timeline_ns()    # Timeline NS (TYPE IDs) par chunk/pi
│   ├── map_b2_to_player()            # {b2_value → pi} via NS Section 1 (pipeline unifié)
│   ├── group_events_by_pi()          # Dispatche all events → {pi: [events]}
│   ├── correlate_kills()             # Corrélation claim-and-remove (fire_event / formula_a)
│   └── compute_confidence()          # Zones A/B/C par arme
│
├── _weapon_data.py               # ÉTENDU — 36 armes confirmées  (≤ 450 L)
│   ├── WEAPON_ID_MAP                 # 36 entrées (source : acurtis166, mars 2026)
│   ├── WEAPON_TIMING_BY_ID           # Inchangé
│   ├── MELEE_MEDALS                  # +Ninja, +Pancake
│   └── GRENADE_MEDALS                # Inchangé
│
├── packet_index.py               # INCHANGÉ — Indexation paquets
├── player_index.py               # INCHANGÉ — Résolution pi↔xuid
│
├── reconciliation.py             # NOUVEAU — Post-traitement découplé  (≤ 200 L)
│   ├── reconcile_api_aggregates()    # Steps 4a/4b/4c (corrigé)
│   └── assign_sentinels()            # Écriture reconciled_as, jamais weapon_id
│
└── _parser_logging.py            # NOUVEAU — Logging structuré par match  (≤ 100 L)

src/data/services/
└── weapon_extraction_service.py  # RÉÉCRIT — Orchestration async

src/data/repositories/
└── _weapon_kills_repo.py         # MODIFIÉ — Support reconciled_as + vue

src/data/sync/
└── migrations.py                 # MODIFIÉ — add reconciled_as column

src/data/migration/steps/
└── add_reconciled_as.py          # NOUVEAU — Migration step
```

### 2.2 Flux de données (diagramme)

```
                    ┌─────────────────────────────────────┐
                    │    highlight_events (kills, t_ms)    │
                    │    match_participants (API aggs)      │
                    │    medals_earned (melee/grenade)      │
                    └─────────────┬───────────────────────┘
                                  │ SQL batch (2 requêtes)
                                  ▼
┌──────────────────────────────────────────────────────────┐
│              SERVICE v2 (orchestration async)             │
│                                                          │
│  1. Charger kill_times + médailles (batch SQL)           │
│  2. Filtrer chunks nécessaires (kill windows)            │
│  3. Télécharger chunks (async, sémaphore ×5)             │
│  4. Résoudre player_indices (METADATA → acurtis)         │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │           PARSER v2 (couche pure, 0 I/O)           │  │
│  │                                                    │  │
│  │  5a. Scan Phase (par chunk) :                      │  │
│  │      - index_chunk() → paquets                     │  │
│  │      - scan_fire_events_all(chunk) → all events    │  │
│  │      - scan_melee_events(chunk) → melee events POV │  │
│  │      - scan_formula_a_ns(chunk) → NS timeline      │  │
│  │                                                    │  │
│  │  5b. Accumulation + dispatch cross-chunk :         │  │
│  │      b2_to_pi = map_b2_to_player(events, ns_tl)   │  │
│  │      fire_events_by_pi = group_events_by_pi(...)   │  │
│  │      timeline_ns[chunk][pi] = TYPE IDs NS          │  │
│  │      melee_events[] = accumulation cross-chunk     │  │
│  │                                                    │  │
│  │  5c. Correlation Phase (claim-and-remove) :        │  │
│  │      Même logique pour TOUS les joueurs du lobby : │  │
│  │      - Chercher fire event non-claimé [t-5s, t]    │  │
│  │      - Si trouvé → attribution "fire_event"        │  │
│  │      - Sinon → fallback NS Section 1 ("formula_a") │  │
│  │      - compute_confidence() → high/medium/low/none │  │
│  │                                                    │  │
│  │  Sortie : list[KillAttribution]                    │  │
│  └────────────────────────────────────────────────────┘  │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │      RÉCONCILIATION (post-traitement optionnel)    │  │
│  │                                                    │  │
│  │  6a. Comparer film vs API aggregates               │  │
│  │  6b. assign_sentinels() → reconciled_as            │  │
│  │      RÈGLE : weapon_id JAMAIS modifié              │  │
│  │      RÈGLE : seulement si confidence ∈ {low, none} │  │
│  │                                                    │  │
│  │  Sortie : list[KillAttribution] enrichi            │  │
│  └────────────────────────────────────────────────────┘  │
│                                                          │
│  7. Écriture DB via Repository (write_lock)              │
│  8. Mark bit WEAPON_KILLS                                │
│  9. Flush logs structurés                                │
└──────────────────────────────────────────────────────────┘
```

### 2.3 Dataclass centrale — KillAttribution

```python
from dataclasses import dataclass

@dataclass(slots=True)
class KillAttribution:
    """Résultat d'attribution d'un kill à une arme."""
    match_id: str
    xuid: str
    time_ms: int
    weapon_id: int | None        # Hex film brut (UBIGINT). JAMAIS écrasé.
    reconciled_as: int | None    # Sentinel API (0/1/2). NULL si pas de réconciliation.
    delta_ms: int | None         # Écart kill↔fire event (fire_event uniquement)
    confidence: str              # "high" | "medium" | "low" | "none"
    attribution_path: str        # "fire_event" | "melee_event" | "grenade_event" (v3+) | "formula_a" | "none"
    swap_detected: bool          # Swap intra-chunk détecté (formula_a)
    delayed_damage: bool         # delta_ms > travel_max (fire_event)
    player_index: int | None     # pi résolu pour ce joueur
    source_chunk_idx: int | None # Index du chunk source du fire event

    @property
    def effective_weapon_id(self) -> int | None:
        """Arme effective = COALESCE(reconciled_as, weapon_id)."""
        return self.reconciled_as if self.reconciled_as is not None else self.weapon_id
```

---

## 3. Phase 0 — Exploration non-POV

> **Prérequis obligatoire** avant de figer l'architecture finale.  
> Réf. : `.ai/NON_POV_FIRE_EVENTS_CONCLUSIONS_2026-03-12.md`

### 3.1 Objectif

Mesurer si les fire events Section 2 sont fiables pour les coéquipiers non-POV
(player_index ≠ 1) sur un échantillon représentatif de matchs réels.

### 3.2 Protocole

| Étape | Action | Sortie |
|-------|--------|--------|
| E0.1 | Sélectionner 20 matchs diversifiés (4v4, BTB, Firefight) avec weapon_kills existant | Liste de match_ids |
| E0.2 | Pour chaque match, scanner les fire events de TOUS les player_index (0–7) via `scan_all_players()` | `dict[pi, list[fire_event]]` par match |
| E0.3 | Pour chaque pi non-POV, compter : events trouvés, events corrélables à un kill (window 5s), taux de couverture | Table `(match_id, pi, is_pov, n_events, n_correlable, coverage_pct)` |
| E0.4 | Comparer avec l'attribution T1 existante (Formula A) : combien de kills sont améliorés, dégradés, inchangés | Table `(match_id, pi, t1_only, fire_only, fire_better, t1_better, equal)` |
| E0.5 | Décision go/no-go pour convergence Path A unique | Seuil : coverage ≥ 60 % sur T1 → go ; sinon → hybrid maintenu |

### 3.3 Script

```python
# scripts/experimental/explore_non_pov_fire_events.py
# NE PAS écrire en DB — lecture seule + export CSV/JSON
```

### 3.4 Livrables

- Fichier CSV : `data/investigation/non_pov_fire_events_exploration.csv`
- Fichier CSV : `data/investigation/non_pov_t1_vs_fire_comparison.csv`
- Fichier JSON : `data/investigation/non_pov_exploration_summary.json`
- Mise à jour de `.ai/NON_POV_FIRE_EVENTS_CONCLUSIONS_2026-03-12.md` avec résultats quantitatifs
- Décision documentée dans `.ai/thought_log.md`

### 3.5 Critère de sortie

> **Note** : Ces critères ont été évalués via les sections 3.6–3.8. La découverte du b2_stream (inv #131) a rendu le critère de "couverture non-POV" caduc — la décision finale est le pipeline unifié décrit en §3.8.

| Scénario | Décision architecture v2 |
|----------|--------------------------|
| Coverage non-POV ≥ 60 % ET fire_better > t1_better | **Pipeline unifié** : fire events prioritaires, formula_a en fallback |
| Coverage non-POV < 60 % OU fire_better ≤ t1_better | **Hybrid** : POV = fire events, coequipiers = formula_a seul |

### 3.6 Résultats (exécution 2026-03-12)

> 20 matchs analysés, 0 erreurs, 1782 kills totaux, 40 melee events POV.

| Métrique | POV (pi=1) | Non-POV (pi≠1) |
|----------|:----------:|:--------------:|
| Joueurs avec kills | 20 | 147 |
| Total kills | 222 | 1 560 |
| Fire events détectés | 24 373 | **1** |

### 3.7 Découverte critique (2026-03-13) — marqueur pi=1 = match-level

**Contexte** : comparaison directe sur match `147ffd4d` (Super Fiesta Bazaar, 10 joueurs).

| Source | Fire events Film |
|--------|:---:|
| acurtis — Σ tous joueurs | **1 178** |
| Notre `scan_fire_events(pi=1)` | **1 177** |
| Notre `scan_fire_events(pi≠1)` | 0 |

**Conclusion** : `scan_fire_events(pi=1)` est un scanner **match-level**, pas un scanner par joueur.
Le marqueur `_build_marker(pi=1)` = `0b10100100110` est un bit structurel commun à tous les fire events de la Section 2, quelle que soit la valeur de player_index dans le filmshell.

**Action Phase 0 ajoutée** : comprendre comment acurtis extrait l'attribution par joueur dans le filmshell Section 2 (champ distinct dans le paquet ? offset différent du marqueur ? autre section ?)
| Kills corrélés | 183 | 1 |
| Coverage | **82.4 %** | **0.1 %** |

**Comparaison T1 vs Fire (non-POV)** :
| Verdict | Count | % |
|---------|:-----:|:-:|
| neither | 973 | 62.4 % |
| t1_only | 586 | 37.6 % |
| fire_better | **0** | 0 % |
| different | 1 | 0.06 % |

**Décision intermédiaire (révisée par §3.8)** :
- Constat : les fire events Section 2 ne portent pas de player_index — ils sont match-level.
- Ce constat a déclenché l'investigation inv #131 sur le b2_stream, qui a abouti au pipeline unifié (§3.8).
- **N1 (adversaires)** : toujours à évaluer — aucune donnée concluante pour l'instant.
- **N2 (melee events)** : 40 événements détectés sur 20 matchs — signal exploitable.

### 3.8 Découverte T2 path — b2_stream dispatch (2026-03-13)

**Conséquence directe de inv #131** : puisque `byte[1] = 0x26` est constant pour tous les fire events
(marqueur structurel match-level, pas un filtre par joueur), la distinction par joueur passe par
`b2_stream` (byte[2]) — un discriminant stable par vie/arme par joueur.

**Mécanisme T2** :

1. `scan_fire_events_all(chunk)` capture **tous** les fire events du match en un seul scan.
2. `scan_formula_a_ns(chunk)` ou `build_weapon_timeline_ns(chunks)` scanne la **couche
   nibble-shifted de la Section 1** (pas le raw) pour extraire les TYPE IDs présents dans
   `WEAPON_ID_MAP`. Chaque entrée encode `pi = ns[wid_pos - 1] >> 5` et filtre les fire
   events via `ns[wid_pos - 5] != 0x26`.
3. `map_b2_to_player(events, timeline_ns, chunks)` — pour chaque fire event, cherche le pi
   dont le TYPE ID correspond à `event["weapon_bytes"]` dans le chunk couvrant le timestamp
   de l'event, vote par majorité par valeur de `b2`. Retourne `{b2_value: pi}`.
4. `group_events_by_pi(all_events, b2_to_pi)` dispatche les events → `{pi: [fire_events]}`.

**Couverture mesurée (match 147ffd4d)** : ~21 % des fire events résolus (255/1177).
pi=6 (shoxyy) : 179 events attribués vs 182 `shots_fired` API (quasi-exact ✓).
~79 % des fire events tombent en **fallback formula_a** (b2 non encore résolu sur ce match, baseline améliorable).

**⚠️ Distinction critique raw vs NS** :

| Scanner | Couche | Retourne | Dans WEAPON_ID_MAP ? |
|---------|--------|----------|:-------------------:|
| `scan_formula_a()` | raw bytes | instance handles | ❌ jamais |
| `scan_formula_a_ns()` | nibble-shifted | TYPE IDs | ✅ oui |
| `build_weapon_timeline()` | raw | handles par chunk/pi | ❌ jamais |
| `build_weapon_timeline_ns()` | nibble-shifted | TYPE IDs par chunk/pi | ✅ oui |

> **Décision architecture v2** : le fallback `formula_a` utilise `build_weapon_timeline_ns`
> (TYPE IDs NS) pour que `confidence=high`/`medium` soient atteignables. Tant que le
> fallback utilise la couche raw, **100 % des kills formula_a ont `confidence="low"`** —
> les branches `high`/`medium` sont du code mort.

**Architecture bi-path (pipeline unifié)** :

> Même algorithme pour tous les joueurs du lobby — POV ou non-POV, il n'y a pas de distinction dans le code.

| Étape | Tous les joueurs | Couverture |
|-------|-----------------|:----------:|
| **1. Scan** | `scan_fire_events_all()` — 1 seul scan, capture tous les fire events du match | 100 % des events |
| **2. Dispatch** | `map_b2_to_player()` — b2_stream → pi via NS Section 1, pour chaque joueur | ≥21 %+ corrects (baseline 1 match, pas un plafond) |
| **3a. fire_event** | fire event corrélable dans la fenêtre 5s → corrélation temporelle directe (claim-and-remove) | = kills avec fire event disponible |
| **3b. formula_a** | pas de fire event corrélable → NS Section 1 snapshot (TYPE ID, sans timestamp précis) | fallback pour les kills sans fire event dans la fenêtre |

**Conséquence** : les labels "Path A" et "T2" sont supprimés. `attribution_path` a 4 valeurs (+ 1 réservée v3+) :
- `fire_event` : fire event corrélable trouvé dans la fenêtre (tous joueurs confondus)
- `melee_event` : melee event film corrélé (tous joueurs confondus)
- `grenade_event` *(FUTUR v3+)* : grenade/explosive event film corrélé — extension anticipée, marqueur film à découvrir
- `formula_a` : pas de fire event dans la fenêtre, snapshot NS (tous joueurs confondus)
- `none` : aucune source film disponible

La "couverture POV élevée" (82%) vs "non-POV plus basse" n'est pas une différence d'algorithme — c'est simplement l'observation empirique que le POV a statistiquement plus de fire events dans la fenêtre de ses kills.

---

## 4. Phase 1 — Migration schéma

### 4.1 Nouvelle colonne `reconciled_as`

```sql
-- Dans ensure_weapon_kills_table() (migrations.py)
ALTER TABLE weapon_kills ADD COLUMN IF NOT EXISTS reconciled_as UBIGINT;
```

### 4.2 Vue `v_weapon_kills`

```sql
CREATE OR REPLACE VIEW v_weapon_kills AS
SELECT *,
       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
FROM weapon_kills;
```

### 4.3 Nouvelle colonne `attribution_path`

```sql
ALTER TABLE weapon_kills ADD COLUMN IF NOT EXISTS attribution_path VARCHAR DEFAULT 'unknown';
```

### 4.4 Fichier de migration step

```
src/data/migration/steps/add_reconciled_as.py
```

```python
from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

register(Migration(
    name="add_reconciled_as",
    target_db="shared",
    description="Ajoute reconciled_as + vue v_weapon_kills + attribution_path",
    apply_schema=ensure_weapon_kills_reconciled_as,
))
```

### 4.5 Import dans `__init__.py`

```python
from src.data.migration.steps import add_reconciled_as  # noqa: F401
```

### 4.6 Tests migration

| Test | Vérifie |
|------|---------|
| `test_add_reconciled_as_fresh_db` | Colonne créée sur table neuve |
| `test_add_reconciled_as_idempotent` | Exécution ×2 sans erreur |
| `test_add_reconciled_as_preserves_data` | Données existantes intactes, `reconciled_as = NULL` partout |
| `test_v_weapon_kills_view_coalesce` | `effective_weapon_id = weapon_id` quand `reconciled_as IS NULL` |
| `test_v_weapon_kills_view_override` | `effective_weapon_id = reconciled_as` quand `reconciled_as IS NOT NULL` |
| `test_attribution_path_default` | Valeur par défaut = `'unknown'` pour lignes existantes |

---

## 5. Phase 2 — Parser v2 (couche pure)

> Fichier : `src/analysis/weapon_parser.py`  
> Contrainte : **0 I/O, 0 import DB, 0 import asyncio**.  
> Seules dépendances externes : `bitstring`, collections stdlib.

### 5.1 Fonctions conservées (refactorisées)

| Fonction v1 | Statut v2 | Changements |
|-------------|-----------|-------------|
| `scan_formula_a()` | ✅ Conservée (raw) | Retourne instance handles. Utilisée uniquement comme stub de compatibilité. |
| `build_weapon_timeline()` | ✅ Conservée (raw) | Timeline raw par chunk/pi. Gardée pour compat tests. **Doit être remplacée par NS.** |
| `scan_formula_a_ns()` | 🆕 Nouvelle | Section 1 couche nibble-shifted → TYPE IDs + pi. Requis pour le fallback formula_a et pour résoudre b2_stream. |
| `build_weapon_timeline_ns()` | 🆕 Nouvelle | Timeline NS (TYPE IDs) par chunk/pi. Utilisée par `map_b2_to_player` et fallback formula_a. |
| `scan_fire_events_all()` | 🔄 Renommée | Ex-`scan_fire_events` : scanner **match-level**, 0 filtre pi. Capture tous les fire events du lobby. |
| `map_b2_to_player()` | 🆕 Nouvelle | `{b2_value → pi}` via NS Section 1 — **pipeline unifié tous joueurs**. Intégrée dans `weapon_parser.py`. |
| `group_events_by_pi()` | 🆕 Nouvelle | Dispatche all events → `{pi: [events]}` selon b2_to_pi. Intégrée dans `weapon_parser.py`. |
| `scan_melee_events()` | 🆕 Nouvelle | Marqueur `0xd340`, couche NS. POV uniquement. Même structure que fire events + champ animation. |
| `scan_grenade_events_all()` | 🔮 FUTUR (v3+) | Scanner match-level grenades/explosifs. Même pattern que `scan_fire_events_all()`. Conditionnel à la découverte du marqueur film. |
| `find_chunk_at_time()` | ✅ Conservée | Inchangée. |
| `find_frame_positions()` | ⚠️ Dépréciée | Remplacée par `build_packet_estimator()` (packet_index.py). Gardée pour compat tests. |
| `build_frame_estimator()` | ⚠️ Dépréciée | Idem. Gardée pour compat tests. |
| `_get_confidence()` | ✅ Conservée | Renommée `compute_confidence()` (publique). |
| `_check_zone_b_swap()` | ✅ Conservée | Renommée `check_zone_b_swap()` (publique). Re-évaluer W1 vs W2 (voir how_it_works). |

### 5.2 Nouvelles fonctions

#### `correlate_kills(kills, fire_events_by_pi, timeline, swap_pis, timing, player_pi_map) → list[KillAttribution]`

Cœur de la v2. Remplace `correlate_kills_to_weapons()`.

```python
def correlate_kills(
    kills: list[dict],
    fire_events_by_pi: dict[int, list[dict]],
    melee_events_by_pi: dict[int, list[dict]],   # melee events POV (vide pour non-POV)
    timeline: dict[int, dict[int, bytes]],
    swap_pis: dict[int, set[int]],
    timing: list[tuple[int, int, int]],
    player_pi_map: dict[str, int],  # xuid → pi
    *,
    kill_window_ms: int = KILL_WINDOW_MS,
    log_callback: Callable | None = None,
    # FUTUR v3+ : grenade_events_by_pi: dict[int, list[dict]] | None = None,
) -> list[KillAttribution]:
```

**Algorithme (claim-and-remove) — même logique pour TOUS les joueurs du lobby** :

```
1. Trier kills par time_ms croissant
2. Par joueur (xuid/pi), construire les pools d'events (tous mutable, pré-triés par timestamp_ms) :
   - fire_events_by_pi  (après dispatch b2_stream universel)
   - melee_events_by_pi (POV uniquement pour l'instant)
   - [FUTUR v3+] grenade_events_by_pi
3. Pour chaque kill — dispatch en cascade, ordre de priorité strict :
   a. pi = player_pi_map[kill.xuid]

   ► Étape M — Melee (si kill.medals ∩ MELEE_MEDALS ≠ ∅) :
      melee_pool = melee_events_by_pi.get(pi, [])
      Si melee_event non-claimé dans [kill.time_ms - kill_window_ms, kill.time_ms] :
      → claim, attribution_path = "melee_event", confidence = "high", weapon_id = event.weapon_id
      → fin dispatch pour ce kill

   ► [FUTUR v3+] Étape G — Grenade (si kill.medals ∩ GRENADE_MEDALS ≠ ∅) :
      # ── Point d'extension : tout nouveau type d'event film s'insère ici, ──
      # ── avant le fallback formula_a, avec le même pattern claim-and-remove. ──
      grenade_pool = grenade_events_by_pi.get(pi, [])  # kwarg optionnel
      Si grenade_event non-claimé dans [kill.time_ms - kill_window_ms, kill.time_ms] :
      → claim, attribution_path = "grenade_event", confidence = "high", weapon_id = None  # grenade ≠ loadout
      → fin dispatch pour ce kill

   ► Étape F — Fire event :
      pool = fire_events_by_pi.get(pi, [])  (mutable, pré-trié)
      Si fire event non-claimé dans [kill.time_ms - kill_window_ms, kill.time_ms] :
      → claim, weapon_id = event.weapon_id, delta_ms = kill.time_ms - event.timestamp_ms
      → confidence = compute_confidence(weapon_id, delta_ms)
      → attribution_path = "fire_event"  ← même valeur POV ou non-POV
      → check_zone_b_swap() si confidence == "medium"
      → fin dispatch pour ce kill

   ► Étape A — Fallback NS Section 1 (formula_a) :
      chunk_idx = find_chunk_at_time(timing, kill.time_ms)
      weapon_id = timeline_ns[chunk_idx].get(pi)  ← TYPE ID (NS)
      swap_detected = pi in swap_pis.get(chunk_idx, set())
      confidence = "high" si pas swap, "medium" si swap
      attribution_path = "formula_a"

   ► Étape ∅ — Aucune source :
      weapon_id = None, confidence = "none", attribution_path = "none"

   log_callback(kill, attribution, decision_details) si fourni
4. Retourner list[KillAttribution]
```

**Invariants vérifiés par assertion** :
- Chaque fire event est claimé au plus une fois
- Chaque kill produit exactement un KillAttribution
- `len(output) == len(kills)`

#### `scan_fire_events_all(chunk_data, start_ms, duration_ms, packets) → list[dict]`

Scanner match-level. Capture **tous** les fire events du chunk sans filtre player_index
(byte[1] = `0x26` est constant pour l'ensemble des joueurs, cf. inv #131).

```python
def scan_fire_events_all(
    chunk_data: bytes,
    start_ms: int,
    duration_ms: int,
    packets: list | None = None,
) -> list[dict]:
    """Scanne tous les fire events du chunk (match-level, 0 filtre pi).

    Retourne une liste de dicts avec les champs :
      timestamp_ms, weapon_bytes, weapon_id, fire_counter, b2_stream, is_burst_end, is_hit.
    La clé de déduplication pour les armes automatiques (BR75, MA40) est
    (weapon_id, fire_counter) — deux entrées par salve avec b2_stream différents.
    """
```

**Déduplication** : `(weapon_id, fire_counter)` par chunk. Les armes automatiques (BR75,
MA40 AR) génèrent deux entrées par tir avec `b2_stream` différents mais même `fire_counter`.

---

#### `scan_melee_events(chunk_data, start_ms, duration_ms, packets) → list[dict]`

Scanner POV pour les melee events film. Marqueur `0xd340` dans la couche nibble-shifted.
Structure identique aux fire events + champ `animation_type` (`5` ou `d`).

```python
def scan_melee_events(
    chunk_data: bytes,
    start_ms: int,
    duration_ms: int,
    packets: list | None = None,
) -> list[dict]:
    """Scanne les melee events POV (marqueur 0xd340).

    Champs retournés : timestamp_ms, weapon_bytes, weapon_id, animation_type.
    Permet d'attribuer les kills melee directement depuis le film sans médailles.
    """
```

**Scope** : POV uniquement (même couche nibble-shifted que fire events → même restriction).

---

#### `scan_formula_a_ns(chunk_data) → list[dict]`

Scanner Section 1 en couche nibble-shifted. Retourne des TYPE IDs (présents dans
`WEAPON_ID_MAP`), contrairement à `scan_formula_a` (raw) qui retourne des instance handles
jamais dans `WEAPON_ID_MAP`.

```python
def scan_formula_a_ns(chunk_data: bytes) -> list[dict]:
    """Scanne les snapshots Section 1 via couche nibble-shifted.

    Champs : pi (= ns[wid_pos-1] >> 5), weapon_id (TYPE ID), byte_pos.
    Filtre les fire events parasites via ns[wid_pos-5] != 0x26.
    TYPE IDs retournés sont directement utilisables pour WEAPON_ID_MAP.
    """
```

**Usage** : `build_weapon_timeline_ns` et `map_b2_to_player` dépendent de cette fonction.

---

#### `build_weapon_timeline_ns(chunks_sorted) → tuple[dict, dict]`

Construit la timeline weapon par chunk/pi depuis la couche NS (TYPE IDs).
Utilisée pour le fallback formula_a et par `map_b2_to_player`.

```python
def build_weapon_timeline_ns(
    chunks_sorted: list[tuple[int, bytes, int, int]],  # (idx, data, start_ms, dur_ms)
) -> tuple[dict[int, dict[int, int]], dict[int, set[int]]]:
    """Timeline NS : {chunk_idx: {pi: weapon_id_int}}, {chunk_idx: {pi_avec_swap}}.

    Contrairement à build_weapon_timeline (raw), les weapon_id retournés
    sont des TYPE IDs présents dans WEAPON_ID_MAP → high/medium atteignables dans le fallback formula_a.
    """
```

> **Note** : `map_b2_to_player` et `group_events_by_pi` sont dans `weapon_parser.py` au même titre que les autres fonctions du pipeline — pas dans un module séparé.

### 5.3 Corrections de bugs intégrées

| Bug | Correction dans v2 | Test dédié |
|-----|---------------------|------------|
| **Claim double** (fire event partagé entre 2 kills) | Claim-and-remove dans `correlate_kills()` | `test_claim_and_remove_no_double_attribution` |
| **Cross-chunk manqué** | Liste globale plate triée, pas de frontière chunk dans la corrélation | `test_cross_chunk_boundary_kill` |
| **W2 retenu au lieu de W1** (Zone B) | Revoir logique `check_zone_b_swap()` : retenir W1 (arme avant swap) sauf preuve contraire | `test_zone_b_retains_w1_not_w2` |
| **5 hex manquants** | Ajoutés dans `WEAPON_ID_MAP` (Bug D) | `test_all_confirmed_weapons_in_map` |
| **MELEE_MEDALS incomplet** | `+Ninja`, `+Pancake` (Bug B) | `test_melee_medals_complete` |
| **formula_a raw vs NS** | Le fallback formula_a utilise `build_weapon_timeline_ns` (TYPE IDs) — `confidence=high`/`medium` atteignables | `test_t1_ns_confidence_high_reachable` |
| **scan_fire_events match-level** | Renommée `scan_fire_events_all()`, 0 filtre pi — dispatch b2_stream via `map_b2_to_player` (intégré dans weapon_parser.py, même pipeline pour tous les joueurs) | `test_scan_fire_events_all_match_level` |

### 5.4 Contrat de sortie du parser

Le parser retourne `list[KillAttribution]` avec les garanties :
- `weapon_id` = hex brut du film ou `None`. **Jamais** un sentinel (0/1/2).
- `reconciled_as` = `None` systématiquement (pas la responsabilité du parser).
- `confidence` ∈ `{"high", "medium", "low", "none"}` — jamais `None` Python.
- `attribution_path` ∈ `{"fire_event", "melee_event", "formula_a", "none"}` — jamais `None` Python.

  | Valeur | Description |
  |--------|-------------|
  | `fire_event` | Fire event corrélé via b2_stream (tous joueurs — POV ou non) |
  | `melee_event` | Melee event film corrélé (marqueur `0xd340`) |
  | `grenade_event` | *(FUTUR v3+)* Grenade/explosive event corrélé — `weapon_id = None` (grenade ≠ loadout weapon) |
  | `formula_a` | NS Section 1 snapshot (TYPE ID) — b2_stream non résolu pour ce joueur |
  | `none` | Aucune source film disponible |

---

## 6. Phase 3 — Service v2 (orchestration)

> Fichier : `src/data/services/weapon_extraction_service.py`

### 6.1 Méthode principale — `process_match()`

```python
async def process_match(
    self,
    match_id: str,
    gamertag: str,
    xuid: str,
    *,
    dry_run: bool = False,
    enable_reconciliation: bool = True,
    enable_sentinels: bool = True,
    log_collector: MatchLogCollector | None = None,
) -> MatchProcessingResult:
```

**Nouveaux paramètres** :
- `enable_reconciliation` : permet de désactiver la réconciliation API (tests, debug)
- `enable_sentinels` : permet de désactiver l'assignation de sentinels
- `log_collector` : collecteur de logs structurés (voir §9)

### 6.2 Flux interne réécrit

```python
async def process_match(self, match_id, gamertag, xuid, *, ...):
    log = log_collector or MatchLogCollector(match_id)
    
    # ── Étape 1 : Chargement batch SQL ──
    participants = self._load_participants(match_id)
    all_kills_by_xuid = self._load_all_kill_times(match_id, list(participants.keys()))
    log.record_step("load_data", kills_total=sum(len(v) for v in all_kills_by_xuid.values()))
    
    if not any(all_kills_by_xuid.values()):
        log.record_step("early_exit", reason="no_kills")
        return MatchProcessingResult.empty(match_id)
    
    # ── Étape 2 : Filtrage chunks ──
    all_kill_times = [t for kills in all_kills_by_xuid.values() for t in kills]
    needed_chunks = self._compute_needed_chunks(match_id, all_kill_times)
    log.record_step("chunk_filter", needed=len(needed_chunks), total=self._total_chunks)
    
    # ── Étape 3 : Téléchargement chunks (async) ──
    chunks = await self._download_needed_chunks(match_id, needed_chunks)
    log.record_step("download", chunks_downloaded=len(chunks))
    
    # ── Étape 4 : Résolution player_index (offload thread) ──
    xuid_ints = {int(x[5:-1]): x for x in participants.keys() if x.startswith("xuid(")}
    pi_map = await asyncio.to_thread(self._resolve_all_player_indices, chunks, xuid_ints)
    log.record_step("resolve_pi", resolved=len(pi_map), total=len(xuid_ints))
    
    # ── Étape 5 : Scan Phase (offload thread) ──
    scan_result = await asyncio.to_thread(
        self._scan_all_chunks, chunks, pi_map, log
    )
    # scan_result = ScanResult(fire_events_by_pi, timeline, swap_pis, timing)
    
    # ── Étape 6 : Correlation Phase (offload thread) ──
    all_attributions: list[KillAttribution] = []
    for xuid_str, kills in all_kills_by_xuid.items():
        if not kills:
            continue
        pi = pi_map.get(xuid_str)
        if pi is None:
            # Joueur non résolu → confidence=none pour tous ses kills
            all_attributions.extend(self._unresolved_kills(match_id, xuid_str, kills))
            log.record_step("unresolved_player", xuid=xuid_str, kills=len(kills))
            continue
        
        attributions = await asyncio.to_thread(
            correlate_kills,
            kills=kills,
            fire_events_by_pi=scan_result.fire_events_by_pi,
            timeline=scan_result.timeline,
            swap_pis=scan_result.swap_pis,
            timing=scan_result.timing,
            player_pi_map={xuid_str: pi},
            log_callback=log.kill_decision if log else None,
        )
        all_attributions.extend(attributions)
    
    # ── Étape 7 : Réconciliation API (optionnelle) ──
    if enable_reconciliation:
        all_attributions = reconcile_api_aggregates(
            all_attributions, self.conn, match_id,
            enable_sentinels=enable_sentinels,
            log_callback=log.reconciliation_decision,
        )
    
    # ── Étape 8 : Écriture DB ──
    if not dry_run:
        rows_inserted = await self._write_attributions(match_id, all_attributions)
        self._mark_backfill_done(match_id)
        log.record_step("write_db", rows_inserted=rows_inserted)
    
    # ── Étape 9 : Flush logs ──
    log.flush()
    
    return MatchProcessingResult(
        match_id=match_id,
        kills_total=len(all_attributions),
        kills_attributed=sum(1 for a in all_attributions if a.weapon_id is not None),
        rows_inserted=rows_inserted if not dry_run else 0,
        players_processed=len(all_kills_by_xuid),
        log_summary=log.summary(),
    )
```

### 6.3 Résolution player_index — double méthode

```python
def _resolve_all_player_indices(self, chunks, xuid_ints) -> dict[str, int]:
    """Résout pi pour tous les joueurs.
    
    Méthode 1 (rapide) : PLAYER_METADATA packet (~25 KB)
    Méthode 2 (fallback) : acurtis bitstring scan (~700 KB)
    """
    pi_map = {}
    
    # Tentative METADATA (premier chunk uniquement)
    first_chunk = next(iter(chunks.values()), None)
    if first_chunk:
        packets = index_chunk(first_chunk[0])
        metadata = extract_metadata_payload(first_chunk[0], packets)
        if metadata:
            detected = detect_pi_from_metadata(metadata, set(xuid_ints.keys()))
            for pi, xuid_int in detected.items():
                xuid_str = xuid_ints[xuid_int]
                pi_map[xuid_str] = pi
    
    # Fallback acurtis pour les non-résolus
    missing = {xi: xs for xi, xs in xuid_ints.items() if xs not in pi_map}
    if missing:
        for chunk_data, _, _ in chunks.values():
            detected = detect_player_indices(chunk_data, set(missing.keys()))
            for pi, xuid_int in detected.items():
                xuid_str = missing[xuid_int]
                pi_map[xuid_str] = pi
            missing = {xi: xs for xi, xs in missing.items() if xs not in pi_map}
            if not missing:
                break
    
    return pi_map
```

### 6.4 Scan Phase — accumulation cross-chunk

```python
@dataclass
class ScanResult:
    fire_events_by_pi: dict[int, list[dict]]      # {pi: [events triés par timestamp_ms]}
    melee_events:       list[dict]                # melee events POV (marqueur 0xd340)
    timeline_ns:        dict[int, dict[int, int]] # {chunk_idx: {pi: weapon_id_int}} TYPE IDs NS
    timeline_raw:       dict[int, dict[int, bytes]]# {chunk_idx: {pi: handle}} raw legacy
    swap_pis:           dict[int, set[int]]       # {chunk_idx: {pi avec swap}}
    timing:             list[tuple[int, int, int]] # [(chunk_idx, start_ms, duration_ms)]
    b2_to_pi:           dict[int, int]            # {b2_value: pi} résolu par map_b2_to_player

def _scan_all_chunks(self, chunks, pi_map, log) -> ScanResult:
    """Scan Phase — accumule fire events et timeline sur tous les chunks.
    
    INVARIANT : fire_events_by_pi[pi] est trié par timestamp_ms croissant.
    """
    all_raw_events: list[dict] = []
    all_melee_events: list[dict] = []
    timeline_ns: dict[int, dict[int, int]] = {}
    timeline_raw: dict[int, dict[int, bytes]] = {}
    swap_pis: dict[int, set[int]] = {}
    timing: list[tuple[int, int, int]] = []
    
    for chunk_idx, (chunk_data, start_ms, duration_ms) in sorted(chunks.items()):
        packets = index_chunk(chunk_data)
        timing.append((chunk_idx, start_ms, duration_ms))
        
        # Section 2 — Fire events match-level (1 scan par chunk, pas de filtre pi)
        raw_events = scan_fire_events_all(
            chunk_data, start_ms, duration_ms, packets=packets
        )
        all_raw_events.extend(raw_events)
        log.record_step("scan_fire", chunk=chunk_idx, events=len(raw_events))

        # Section 2 — Melee events POV (marqueur 0xd340)
        chunk_melee = scan_melee_events(
            chunk_data, start_ms, duration_ms, packets=packets
        )
        all_melee_events.extend(chunk_melee)

        # Section 1 NS — TYPE IDs pour map_b2_to_player et fallback formula_a
        fa_ns = scan_formula_a_ns(chunk_data)
        chunk_timeline_ns, chunk_swaps = _process_formula_a_for_chunk_ns(fa_ns, chunk_idx)
        timeline_ns[chunk_idx] = chunk_timeline_ns
        swap_pis[chunk_idx] = chunk_swaps

        # Section 1 raw — instance handles (legacy fallback, à terme remplacé par NS)
        chunk_timeline_raw, _ = _process_formula_a_for_chunk(
            scan_formula_a(chunk_data), chunk_idx, set(pi_map.values())
        )
        timeline_raw[chunk_idx] = chunk_timeline_raw

    # Tri global par timestamp_ms
    all_raw_events.sort(key=lambda e: e["timestamp_ms"])

    # Dispatch b2_stream → pi via NS timeline (pipeline unifié)
    b2_to_pi = map_b2_to_player(all_raw_events, timeline_ns, chunks)
    fire_events_by_pi = group_events_by_pi(all_raw_events, b2_to_pi)
    log.record_step("b2_dispatch", resolved_b2=len(b2_to_pi), total_events=len(all_raw_events))

    return ScanResult(
        fire_events_by_pi=fire_events_by_pi,
        melee_events=all_melee_events,
        timeline_ns=timeline_ns,
        timeline_raw=timeline_raw,
        swap_pis=swap_pis,
        timing=timing,
        b2_to_pi=b2_to_pi,
    )
```

### 6.5 Écriture batch

```python
async def _write_attributions(self, match_id: str, attributions: list[KillAttribution]) -> int:
    """Écriture batch via write_lock (sérialisé pour DuckDB)."""
    async with self.write_lock:
        return self.repo.insert_weapon_kill_rows_v2(
            self.conn, match_id, attributions
        )
```

---

## 7. Phase 4 — Réconciliation API découplée

> Nouveau fichier : `src/analysis/reconciliation.py`  
> **Principe** : post-traitement optionnel, désactivable sans impact sur le parser.

### 7.1 Interface publique

```python
def reconcile_api_aggregates(
    attributions: list[KillAttribution],
    conn: duckdb.DuckDBPyConnection,
    match_id: str,
    *,
    enable_sentinels: bool = True,
    log_callback: Callable | None = None,
) -> list[KillAttribution]:
    """Ajuste les attributions film vs agrégats API (grenade_kills, melee_kills).
    
    RÈGLE ABSOLUE : ne modifie JAMAIS weapon_id.
    Les sentinels sont écrits dans reconciled_as uniquement.
    
    Éligibilité pour reconciled_as :
    - confidence == "low"  → ✅ (timing suspect, signal API potentiellement plus fiable)
    - confidence == "none" → ✅ (pas de donnée film)
    - confidence == "high" → ❌ jamais
    - confidence == "medium" → ❌ jamais
    
    RÈGLE SUPPLÉMENTAIRE (correction Bug F) :
    - weapon_id != None ET weapon_id NOT IN EXCLUDED_WEAPON_IDS → ❌ jamais
      (ne pas écraser un hex réel, quelle que soit la confidence)
    """
```

### 7.2 Étapes internes

```
Step 4a — Vérification surplus weapon kills :
  Si count(film_weapon_kills) > count(API_weapon_kills) :
    Dégrader les HIGH les plus incertains (delta_ms le plus grand) → MEDIUM
    Log : "demoted {n} kills HIGH→MEDIUM (surplus vs API)"

Step 4b — Injection sentinels (CORRIGÉ) :
  deficit_grenade = API.grenade_kills - count(reconciled_as == GRENADE)
  deficit_melee = API.melee_kills - count(reconciled_as == MELEE)
  
  Candidats éligibles (triés par delta_ms décroissant) :
    - confidence ∈ {"low", "none"}
    - weapon_id IS NULL  ← NOUVEAU FILTRE (Bug F fix)
    - weapon_id NOT IN WEAPON_IDS_INT  ← déjà inconnu OK
    
  INTERDIT : reclassifier un kill avec weapon_id connu (hex réel dans WEAPON_ID_MAP)
  
  Pour chaque candidat éligible :
    attribution.reconciled_as = GRENADE_WEAPON_ID ou MELEE_WEAPON_ID
    Log : "sentinel assigned: kill at {t_ms} → reconciled_as={sentinel}"

Step 4c — Promotion MEDIUM→HIGH :
  Si count(film_weapon_kills) < count(API_weapon_kills) :
    Promouvoir les MEDIUM les plus fiables (delta_ms le plus petit) → HIGH
    Log : "promoted {n} kills MEDIUM→HIGH (deficit vs API)"
```

### 7.3 Tests réconciliation

| # | Test | Vérifie |
|---|------|---------|
| R1 | `test_reconcile_no_api_data_unchanged` | Pas de ligne match_participants → attributions inchangées |
| R2 | `test_reconcile_exact_match_unchanged` | Film = API → 0 modifications |
| R3 | `test_reconcile_surplus_demotes_high` | Surplus weapon kills → HIGH dégradé en MEDIUM (plus grand delta_ms d'abord) |
| R4 | `test_reconcile_deficit_grenade_assigns_sentinel` | Déficit grenades → reconciled_as=0 sur candidats éligibles |
| R5 | `test_reconcile_deficit_melee_assigns_sentinel` | Déficit melee → reconciled_as=1 sur candidats éligibles |
| R6 | `test_reconcile_never_overwrites_weapon_id` | weapon_id JAMAIS modifié (assertion sur valeur avant/après) |
| R7 | `test_reconcile_never_assigns_sentinel_on_high` | confidence=high → reconciled_as reste None |
| R8 | `test_reconcile_never_assigns_sentinel_on_real_hex` | weapon_id ∈ WEAPON_IDS_INT → reconciled_as reste None (Bug F fix) |
| R9 | `test_reconcile_sentinel_only_on_null_weapon` | Seuls les kills weapon_id=None reçoivent reconciled_as |
| R10 | `test_reconcile_melee_priority_over_grenade` | Un kill éligible aux deux → melee gagne (is_melee > is_grenade) |
| R11 | `test_reconcile_disabled_flag` | `enable_sentinels=False` → reconciled_as=None partout |
| R12 | `test_reconcile_log_callback_called` | Chaque décision produit un appel log_callback |

---

## 8. Phase 5 — Repository v2

> Fichier : `src/data/repositories/_weapon_kills_repo.py`

### 8.1 Nouvelle méthode d'insertion

```python
def insert_weapon_kill_rows_v2(
    self,
    conn: duckdb.DuckDBPyConnection,
    match_id: str,
    attributions: list[KillAttribution],
) -> int:
    """Insertion batch v2 avec reconciled_as et attribution_path.
    
    Idempotent : DELETE + INSERT pour (match_id).
    Quality gate : n'écrase pas si existing_good > new_good.
    """
```

### 8.2 Requêtes modifiées

Tous les `SELECT weapon_id` consommateurs doivent migrer vers :

```sql
-- AVANT (v1)
SELECT weapon_id FROM weapon_kills WHERE ...

-- APRÈS (v2) — via la vue
SELECT effective_weapon_id FROM v_weapon_kills WHERE ...

-- OU directement
SELECT COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
FROM weapon_kills WHERE ...
```

**Fichiers impactés** (à auditer) :

| Fichier | Méthode | Migration |
|---------|---------|-----------|
| `_weapon_kills_repo.py` | `load_weapon_kills_for_match()` | → `v_weapon_kills` |
| `_weapon_kills_repo.py` | `load_top_weapon_per_player()` | → `v_weapon_kills` |
| `_weapon_kills_repo.py` | `load_weapon_kills_for_player()` | → `v_weapon_kills` |
| `_weapon_kills_repo.py` | `load_weapon_kills_aggregated()` | → `v_weapon_kills` |
| Pages UI Streamlit | Tout accès weapon_id | → `effective_weapon_id` |

### 8.3 Tests repository v2

| # | Test | Vérifie |
|---|------|---------|
| DB1 | `test_insert_v2_creates_rows_with_reconciled_as` | reconciled_as persisté correctement |
| DB2 | `test_insert_v2_idempotent` | DELETE+INSERT sans doublon |
| DB3 | `test_insert_v2_quality_gate` | N'écrase pas si existing > new qualité |
| DB4 | `test_view_effective_weapon_id` | COALESCE fonctionne dans la vue |
| DB5 | `test_load_uses_effective_weapon_id` | Méthodes load retournent effective_weapon_id |
| DB6 | `test_attribution_path_persisted` | Colonne attribution_path sauvegardée |
| DB7 | `test_insert_v2_batch_performance` | ≤ 50ms pour 100 rows (guard perf) |

---

## 9. Stratégie de logging

### 9.1 Principes

1. **Structuré (JSON)** : chaque log est un dict sérialisable, pas du texte libre
2. **Hiérarchique** : match → joueur → kill → décision
3. **Optionnel** : le parser fonctionne sans logger (pas de dépendance)
4. **Exportable** : un match entier peut être re-audité sans relancer le parser
5. **Niveaux** : `DEBUG` pour chaque décision, `INFO` pour les résumés, `WARNING` pour les anomalies

### 9.2 Module — `src/analysis/_parser_logging.py`

```python
import logging
from dataclasses import dataclass, field

logger = logging.getLogger("levelup.weapon_parser")


@dataclass
class MatchLogCollector:
    """Collecteur de logs structurés pour un match."""
    match_id: str
    steps: list[dict] = field(default_factory=list)
    kill_decisions: list[dict] = field(default_factory=list)
    reconciliation_decisions: list[dict] = field(default_factory=list)
    warnings: list[dict] = field(default_factory=list)
    
    def record_step(self, step_name: str, **kwargs) -> None:
        """Enregistre une étape du pipeline."""
        entry = {"step": step_name, **kwargs}
        self.steps.append(entry)
        logger.debug("match=%s step=%s %s", self.match_id, step_name, kwargs)
    
    def kill_decision(
        self,
        kill: dict,
        attribution: "KillAttribution",
        details: dict,
    ) -> None:
        """Enregistre la décision d'attribution pour un kill."""
        entry = {
            "xuid": kill["xuid"],
            "time_ms": kill["time_ms"],
            "weapon_id": attribution.weapon_id,
            "confidence": attribution.confidence,
            "attribution_path": attribution.attribution_path,
            "delta_ms": attribution.delta_ms,
            "candidates_count": details.get("candidates_count", 0),
            "claimed_event_ts": details.get("claimed_event_ts"),
            "fallback_used": details.get("fallback_used", False),
        }
        self.kill_decisions.append(entry)
        logger.debug(
            "match=%s kill xuid=%s t=%dms → weapon=%s conf=%s path=%s delta=%s",
            self.match_id, kill["xuid"], kill["time_ms"],
            attribution.weapon_id, attribution.confidence,
            attribution.attribution_path, attribution.delta_ms,
        )
    
    def reconciliation_decision(
        self,
        action: str,  # "demote_high", "inject_sentinel", "promote_medium"
        kill_time_ms: int,
        xuid: str,
        before: dict,
        after: dict,
    ) -> None:
        """Enregistre une décision de réconciliation."""
        entry = {
            "action": action,
            "xuid": xuid,
            "time_ms": kill_time_ms,
            "before": before,
            "after": after,
        }
        self.reconciliation_decisions.append(entry)
        logger.info(
            "match=%s reconcile %s xuid=%s t=%dms: %s → %s",
            self.match_id, action, xuid, kill_time_ms, before, after,
        )
    
    def warn(self, message: str, **context) -> None:
        """Enregistre un warning."""
        entry = {"message": message, **context}
        self.warnings.append(entry)
        logger.warning("match=%s %s %s", self.match_id, message, context)
    
    def summary(self) -> dict:
        """Résumé JSON du traitement."""
        return {
            "match_id": self.match_id,
            "steps_count": len(self.steps),
            "kills_decided": len(self.kill_decisions),
            "reconciliations": len(self.reconciliation_decisions),
            "warnings": len(self.warnings),
            "confidence_distribution": self._confidence_dist(),
            "path_distribution": self._path_dist(),
        }
    
    def _confidence_dist(self) -> dict[str, int]:
        from collections import Counter
        return dict(Counter(d["confidence"] for d in self.kill_decisions))
    
    def _path_dist(self) -> dict[str, int]:
        from collections import Counter
        return dict(Counter(d["attribution_path"] for d in self.kill_decisions))
    
    def flush(self) -> None:
        """Écrit le résumé dans le logger INFO."""
        s = self.summary()
        logger.info(
            "match=%s COMPLETE kills=%d conf=%s paths=%s warnings=%d",
            self.match_id,
            s["kills_decided"],
            s["confidence_distribution"],
            s["path_distribution"],
            s["warnings"],
        )
```

### 9.3 Points de logging dans le pipeline

| Étape | Niveau | Message type | Données |
|-------|--------|-------------|---------|
| Chargement kills | DEBUG | `load_data` | `kills_total`, `players_count` |
| Filtrage chunks | DEBUG | `chunk_filter` | `needed`, `total`, `skipped` |
| Téléchargement | DEBUG | `download` | `chunks_downloaded`, `cache_hits`, `download_ms` |
| Résolution pi | INFO | `resolve_pi` | `resolved`, `total`, `method` (metadata/acurtis) |
| Scan fire events | DEBUG | `scan_fire` | Par chunk×pi : `events_found`, `dedup_removed` |
| Scan Formula A | DEBUG | `scan_formula_a` | Par chunk : `snapshots_found`, `pis_seen` |
| **Corrélation kill** | **DEBUG** | `kill_decision` | `time_ms`, `weapon_id`, `confidence`, `path`, `delta_ms`, `candidates_count` |
| Fire event claimé | DEBUG | `event_claimed` | `event_ts`, `kill_ts`, `delta_ms`, `weapon_id` |
| Fallback Formula A | DEBUG | `fallback_formula_a` | `chunk_idx`, `pi`, `weapon_in_timeline` |
| Aucune info | WARNING | `no_attribution` | `time_ms`, `xuid`, `pi` |
| Joueur non résolu | WARNING | `unresolved_player` | `xuid`, `kills_count` |
| **Réconciliation** | **INFO** | `reconcile_*` | `action`, `before`, `after`, `reason` |
| Sentinel assigné | INFO | `sentinel_assigned` | `time_ms`, `xuid`, `sentinel_type`, `reason` |
| Sentinel refusé (Bug F) | INFO | `sentinel_blocked` | `time_ms`, `xuid`, `weapon_id`, `reason="real_hex"` |
| Écriture DB | INFO | `write_db` | `rows_inserted`, `duration_ms` |
| **Résumé** | **INFO** | `COMPLETE` | `confidence_dist`, `path_dist`, `warnings_count` |

### 9.4 Configuration

```python
# Dans pyproject.toml ou .env.local
[tool.levelup.logging]
weapon_parser = "INFO"  # Défaut production
# weapon_parser = "DEBUG"  # Développement / investigation
```

### 9.5 Export pour audit

```python
# En CLI pour investiguer un match spécifique
import json

log = MatchLogCollector(match_id)
result = await service.process_match(match_id, ..., log_collector=log)

# Export JSON complet
with open(f"data/investigation/weapon_log_{match_id}.json", "w") as f:
    json.dump({
        "summary": log.summary(),
        "steps": log.steps,
        "kill_decisions": log.kill_decisions,
        "reconciliation_decisions": log.reconciliation_decisions,
        "warnings": log.warnings,
    }, f, indent=2)
```

---

## 10. Stratégie de tests

### 10.1 Organisation

```
tests/
├── test_weapon_parser_v2.py        # Parser couche pure (~80 tests)
├── test_weapon_service_v2.py       # Service orchestration (~40 tests)
├── test_weapon_reconciliation.py   # Réconciliation découplée (~15 tests)
├── test_weapon_data.py             # Constantes + résolution (~25 tests, existant étendu)
├── test_weapon_logging.py          # Logging structuré (~10 tests)
├── test_weapon_migration.py        # Migration reconciled_as (~6 tests)
└── test_weapon_parser.py           # Tests v1 conservés (régression)
```

### 10.2 Tests parser v2 — `test_weapon_parser_v2.py`

#### Groupe A — Constantes et invariants

| # | Test | Assert |
|---|------|--------|
| A1 | `test_pov_player_index_is_1` | `POV_PLAYER_INDEX == 1` |
| A2 | `test_sentinel_values_cannot_collide_with_film` | `0, 1, 2` impossibles comme uint64 film |
| A3 | `test_all_confirmed_weapons_in_map` | 36 armes dans `WEAPON_ID_MAP` (35 + 1 nouveau) |
| A4 | `test_weapon_timing_covers_all_known_weapons` | Chaque arme a un `(swap_ms, travel_max_ms)` |
| A5 | `test_melee_medals_includes_ninja_pancake` | `"Ninja" in MELEE_MEDALS`, `"Pancake" in MELEE_MEDALS` |
| A6 | `test_kill_window_ms_is_5000` | `KILL_WINDOW_MS == 5000` |

#### Groupe B — scan_fire_events_all (match-level)

| # | Test | Setup | Assert |
|---|------|-------|--------|
| B1 | `test_scan_fire_events_all_empty` | data = b"" | `[]` |
| B2 | `test_scan_fire_events_all_too_small` | data = b"\x00" * 50 | `[]` |
| B3 | `test_scan_fire_events_all_captures_all_players` | Chunk avec events marqueurs byte[1]=0x26 pour pi=1,3,5 | Tous capturés (match-level) |
| B4 | `test_scan_fire_events_all_b2_stream_present` | 1 fire event | `e["b2_stream"]` non-`None` |
| B5 | `test_scan_fire_events_all_dedup_dual_stream` | 2 entries même `fire_counter`, weapon_id BR75 | 1 event après dédup `(weapon_id, fire_counter)` |
| B6 | `test_scan_fire_events_all_multi_chunk` | 3 chunks, events répartis | Events plats triés par timestamp_ms |
| B7 | `test_scan_fire_events_all_with_packets` | Chunk + index_chunk → packets | Timestamps µs-precision |
| B8 | `test_scan_fire_events_all_returns_is_burst_end` | BR75 séquence 0-0-1 | 3e event : `is_burst_end=True` |
| B9 | `test_scan_melee_events_empty` | data = b"" | `[]` |
| B10 | `test_scan_melee_events_valid` | Chunk avec marqueur 0xd340 | 1 melee event, animation_type ∈ {5, 13} |

#### Groupe C — b2_stream dispatch : `map_b2_to_player` + `group_events_by_pi` (pipeline unifié tous joueurs)

| # | Test | Setup | Assert |
|---|------|-------|--------|
| C1 | `test_map_b2_to_player_empty` | events=[], timeline_ns={} | `{}` |
| C2 | `test_map_b2_to_player_single_player` | events avec b2=42, NS timeline pi=6 weapon BR75 | `{42: 6}` |
| C3 | `test_map_b2_to_player_majority_vote` | b2=0x3A → 9 events pi=3, 1 event pi=1 | `{0x3A: 3}` (majorité) |
| C4 | `test_map_b2_to_player_unknown_b2` | b2 absent de NS timeline | non présent dans dict résultat |
| C5 | `test_map_b2_to_player_multiple_players` | 3 joueurs, b2 distincts | 3 mappings corrects |
| C6 | `test_group_events_by_pi_empty` | events=[], b2_to_pi={} | `{}` |
| C7 | `test_group_events_by_pi_resolved` | 5 events b2 résolu → pi=2 | `{2: [5 events]}` |
| C8 | `test_group_events_by_pi_unresolved` | events b2 non résolu | non présents dans dict |
| C9 | `test_group_events_sorted_by_ts` | events non triés | Sortie triée par `timestamp_ms` |
| C10 | `test_attribution_path_values` | `attribution_path` ne peut être que `fire_event`, `melee_event`, `formula_a`, `none` |
| C11 | `test_scan_formula_a_ns_returns_type_ids` | Chunk NS avec TYPE ID BR75 | weapon_id in WEAPON_ID_MAP |
| C12 | `test_build_weapon_timeline_ns_basic` | 1 chunk, pi=3, BR75 | timeline_ns[0][3] == BR75_weapon_id |
| C13 | `test_t1_ns_confidence_high_reachable` | Kill formula_a via NS timeline (TYPE ID connu, pas de swap) | confidence="high" |

#### Groupe D — scan_formula_a

| # | Test | Setup | Assert |
|---|------|-------|--------|
| D1 | `test_formula_a_empty` | data = b"" | `[]` |
| D2 | `test_formula_a_standard_suffix` | Pattern `200002` + pi=2 + weapon 8B | 1 snapshot, pi=2 |
| D3 | `test_formula_a_multiple_pis` | 3 snapshots pi=1,2,3 | 3 résultats |
| D4 | `test_formula_a_swap_detection` | 2 snapshots même pi, armes différentes | swap_detected pour ce chunk |

#### Groupe E — build_weapon_timeline

| # | Test | Setup | Assert |
|---|------|-------|--------|
| E1 | `test_timeline_empty` | Pas de chunks | `{}`, `{}` |
| E2 | `test_timeline_single_chunk_single_pi` | 1 chunk, 1 pi, 1 weapon | timeline[0][pi] = weapon |
| E3 | `test_timeline_swap_intra_chunk` | 2 weapons même pi même chunk | swap_pis[chunk] contient pi |
| E4 | `test_timeline_multi_chunk` | 3 chunks séquentiels | Timeline correcte par chunk |
| E5 | `test_timeline_fallback_previous_chunk` | Chunk N vide pour pi, chunk N-1 rempli | Fallback fonctionne |

#### Groupe F — correlate_kills (CŒUR v2)

| # | Test | Setup | Assert |
|---|------|-------|--------|
| F1 | `test_correlate_empty_kills` | kills=[] | `[]` |
| F2 | `test_correlate_single_kill_single_event` | 1 kill, 1 fire event 2s avant | confidence=high, delta=2000 |
| F3 | `test_correlate_kill_no_event` | 1 kill, 0 fire events | confidence=none, path=none |
| F4 | `test_correlate_kill_with_formula_a_fallback` | 1 kill, 0 fire events, timeline disponible | path=formula_a |
| F5 | `test_correlate_claim_and_remove_no_double` | 2 kills 1s apart, 1 fire event | 1er kill claimsfire event, 2e → fallback |
| F6 | `test_correlate_claim_and_remove_two_events` | 2 kills, 2 fire events | Chacun claim le sien |
| F7 | `test_correlate_cross_chunk_boundary` | Kill à t=19050, event à t=18800 (chunk précédent) | Corrélation réussie (liste plate) |
| F8 | `test_correlate_window_exactly_5s` | Event à kill_t - 5000ms | Inclus dans la fenêtre |
| F9 | `test_correlate_window_exceeded` | Event à kill_t - 5001ms | Exclu de la fenêtre |
| F10 | `test_correlate_takes_last_unclaimed` | 3 events dans fenêtre | Prend le plus récent non-claimé |
| F11 | `test_correlate_confidence_high_zone_a` | delta < swap_ms | confidence=high |
| F12 | `test_correlate_confidence_medium_zone_b` | swap_ms ≤ delta ≤ travel_max | confidence=medium |
| F13 | `test_correlate_confidence_low_zone_c` | delta > travel_max | confidence=low, delayed_damage=True |
| F14 | `test_correlate_zone_b_swap_check` | Zone B + fire event post-kill avec autre arme | confidence upgradé |
| F15 | `test_correlate_zone_b_swap_retains_w1` | Zone B + swap détecté → W1 (arme avant swap) est retenue | weapon_id = W1 |
| F16 | `test_correlate_confidence_low_zone_c_formula_a` | formula_a, kill à la limite zone C (delta_ms > travel_max) | confidence=low, delayed_damage=True |
| F17 | `test_correlate_aoe_simultaneous_kills` | 2 kills même t_ms, même killer (Hammer) | Même weapon_id, même fire event (pas de double-claim car même event) |
| F18 | `test_correlate_output_length_invariant` | N kills → N attributions | `len(output) == len(kills)` |
| F19 | `test_correlate_weapon_id_never_sentinel` | Toute sortie du parser | `all(a.weapon_id not in {0, 1, 2} for a in output if a.weapon_id)` |
| F20 | `test_correlate_reconciled_as_always_none` | Toute sortie du parser | `all(a.reconciled_as is None for a in output)` |
| F21 | `test_correlate_log_callback_per_kill` | Mock log_callback | Appelé exactement len(kills) fois |
| F22 | `test_correlate_multiple_players` | 2 joueurs, kills entrelacés | Attribution indépendante par pi |
| F23 | `test_correlate_fire_event_after_kill_ignored` | Event à kill_t + 500 | Non retenu (hors fenêtre) |
| F24 | `test_correlate_melee_event_path` | Kill melee suivi d'1 melee_event dans fenêtre | `attribution_path="melee_event"`, `confidence="high"` |
| F25 | `test_correlate_non_pov_fire_event_path` | Kill non-POV, b2 résolu via map_b2_to_player | `attribution_path="fire_event"` (même valeur que POV) |
| F26 | `test_correlate_formula_a_unresolved_b2` | Kill non-POV, b2 non résolu, NS timeline disponible | `attribution_path="formula_a"`, `confidence≠"low"` si TYPE ID connu |

#### Groupe G — compute_confidence

| # | Test | Setup | Assert |
|---|------|-------|--------|
| G1 | `test_confidence_known_weapon_zone_a` | BR75, delta=300ms | "high" (300 < 650=swap_ms) |
| G2 | `test_confidence_known_weapon_zone_b` | Sidekick (swap_ms=400), delta=420ms | "medium" (420 > swap_ms=400, dans zone B) |
| G3 | `test_confidence_known_weapon_zone_c` | Cindershot, delta=5500ms | "low" (> 5000=travel_max) |
| G4 | `test_confidence_unknown_weapon_uses_default_timing` | Hex pas dans WEAPON_TIMING_BY_ID | Utilise timing par défaut (650, 2000) — confidence reste déterminée par delta_ms, pas par la présence dans le map |
| G5 | `test_confidence_delayed_damage_flag` | Zone C → `delayed_damage=True` | Correct |

#### Groupe H — KillAttribution dataclass

| # | Test | Setup | Assert |
|---|------|-------|--------|
| H1 | `test_effective_weapon_id_no_reconciled` | reconciled_as=None, weapon_id=X | effective=X |
| H2 | `test_effective_weapon_id_with_reconciled` | reconciled_as=0, weapon_id=X | effective=0 |
| H3 | `test_effective_weapon_id_both_none` | reconciled_as=None, weapon_id=None | effective=None |

### 10.3 Tests service v2 — `test_weapon_service_v2.py`

| # | Test | Type | Assert |
|---|------|------|--------|
| S1 | `test_process_match_no_kills` | In-memory DB | MatchProcessingResult.empty |
| S2 | `test_process_match_no_chunks` | Mock API 404 | Bit NO_FILM posé, 0 rows |
| S3 | `test_process_match_dry_run` | In-memory DB | 0 writes, résultat non-vide |
| S4 | `test_process_match_pov_only` | 1 joueur POV, 3 kills | 3 KillAttribution, path=fire_event |
| S5 | `test_process_match_pov_plus_t1` | POV + 1 coéquipier | Résolutions pi distinctes |
| S6 | `test_process_match_unresolved_player` | Pi introuvable pour 1 joueur | confidence=none pour ses kills, warning logged |
| S7 | `test_process_match_with_reconciliation` | enable_reconciliation=True | reconciled_as set |
| S8 | `test_process_match_without_reconciliation` | enable_reconciliation=False | reconciled_as=None partout |
| S9 | `test_process_match_batch_sql_efficiency` | Mock conn, count queries | ≤ 4 queries total (2 load + 1 delete + 1 insert) |
| S10 | `test_process_match_chunk_cache_hit` | Film déjà en cache | 0 downloads, parsing normal |
| S11 | `test_process_match_exception_no_partial_write` | Exception mid-process | DB inchangée (transactional) |
| S12 | `test_process_match_write_lock_serialized` | 2 matchs concurrent, même lock | Pas de write overlap |
| S13 | `test_process_match_log_collector` | MatchLogCollector fourni | summary() non-vide |
| S14 | `test_resolve_pi_metadata_first` | Metadata packet présent | Acurtis non appelé |
| S15 | `test_resolve_pi_fallback_acurtis` | Pas de metadata packet | Acurtis appelé |
| S16 | `test_scan_fire_events_all_single_pass` | Mock `scan_fire_events_all` | Appelé 1× par chunk (match-level, pas par joueur) |
| S17 | `test_b2_dispatch_called_once_after_scan` | Mock `map_b2_to_player` | Appelé 1× après accumulation de tous les chunks, résultat partagé par tous les joueurs |
| S18 | `test_melee_events_accumulated_cross_chunk` | 2 chunks avec melee events | `scan_result.melee_events` contient les 2 |

### 10.4 Tests réconciliation — `test_weapon_reconciliation.py`

Voir §7.3 (R1–R12).

### 10.5 Tests logging — `test_weapon_logging.py`

| # | Test | Assert |
|---|------|--------|
| L1 | `test_log_collector_empty_summary` | summary() structuré, compteurs à 0 |
| L2 | `test_log_collector_record_step` | steps list augmentée |
| L3 | `test_log_collector_kill_decision` | Entrée complète avec tous les champs |
| L4 | `test_log_collector_reconciliation_decision` | Action + before/after |
| L5 | `test_log_collector_warn` | warnings list augmentée |
| L6 | `test_log_collector_confidence_distribution` | Counter correcte |
| L7 | `test_log_collector_path_distribution` | Counter correcte |
| L8 | `test_log_collector_flush_logs_to_logger` | logging.INFO avec résumé |
| L9 | `test_log_collector_serializable_json` | `json.dumps(summary())` ne raise pas |
| L10 | `test_log_collector_optional_in_parser` | Parser fonctionne avec log_callback=None |

### 10.6 Tests migration — `test_weapon_migration.py`

Voir §4.6 (6 tests).

### 10.7 Fixtures partagées

```python
# tests/conftest.py ou tests/fixtures/weapon_fixtures.py

@pytest.fixture
def br75_weapon_id() -> int:
    return int.from_bytes(bytes.fromhex("2B1824D542C9679F"), "big")

@pytest.fixture 
def sidekick_weapon_id() -> int:
    return int.from_bytes(bytes.fromhex("F408190F42C9679F"), "big")

@pytest.fixture
def sample_kill(br75_weapon_id) -> dict:
    return {
        "match_id": "test-match-001",
        "xuid": "xuid(1234567890123456)",
        "time_ms": 45000,
        "gamertag": "TestPlayer",
        "medals_nearby": [],
        "is_melee": False,
        "is_grenade": False,
    }

@pytest.fixture
def sample_fire_event(br75_weapon_id) -> dict:
    return {
        "timestamp_ms": 44000,
        "weapon_bytes": bytes.fromhex("2B1824D542C9679F"),
        "weapon_id": br75_weapon_id,
        "fire_seq": 12,
        "player_index": 1,
    }

@pytest.fixture
def in_memory_shared_db() -> duckdb.DuckDBPyConnection:
    """DB DuckDB en mémoire avec schéma weapon_kills v2."""
    conn = duckdb.connect(":memory:")
    conn.execute("""
        CREATE TABLE weapon_kills (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            time_ms INTEGER NOT NULL,
            weapon_id UBIGINT,
            reconciled_as UBIGINT,
            delta_ms INTEGER,
            confidence VARCHAR NOT NULL DEFAULT 'none',
            swap_detected BOOLEAN NOT NULL DEFAULT FALSE,
            delayed_damage BOOLEAN NOT NULL DEFAULT FALSE,
            attribution_path VARCHAR NOT NULL DEFAULT 'unknown',
            PRIMARY KEY (match_id, xuid, time_ms)
        )
    """)
    conn.execute("""
        CREATE VIEW v_weapon_kills AS
        SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
        FROM weapon_kills
    """)
    # Tables supports (match_participants, highlight_events, medals_earned)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
            grenade_kills INTEGER DEFAULT 0,
            melee_kills INTEGER DEFAULT 0,
            kills INTEGER DEFAULT 0
        )
    """)
    conn.execute("""
        CREATE TABLE highlight_events (
            match_id VARCHAR, xuid VARCHAR, time_ms INTEGER,
            event_type VARCHAR DEFAULT 'Kill'
        )
    """)
    conn.execute("""
        CREATE TABLE medals_earned (
            match_id VARCHAR, xuid VARCHAR, time_ms INTEGER,
            medal_name VARCHAR
        )
    """)
    return conn
```

### 10.8 Matrice de couverture cible

| Module | Tests | Lignes estimées | Couverture cible |
|--------|:-----:|:---------------:|:----------------:|
| `weapon_parser.py` | ~105 | ~600 | ≥ 95 % |
| `weapon_extraction_service.py` | ~45 | ~450 | ≥ 85 % |
| `reconciliation.py` | ~15 | ~200 | ≥ 95 % |
| `_parser_logging.py` | ~10 | ~100 | ≥ 90 % |
| `_weapon_data.py` | ~25 | ~450 | ≥ 90 % |
| `_weapon_kills_repo.py` | ~10 | ~450 | ≥ 80 % |
| **Total** | **~210** | **~2250** | **≥ 90 %** |

---

## 11. Plan de migration des données

### 11.1 Stratégie

| Étape | Action | Risque | Rollback |
|-------|--------|--------|----------|
| M1 | Migration schéma (reconciled_as + vue) | Nul | `ALTER TABLE DROP COLUMN` |
| M2 | Déployer code v2 (parser + service + repo) | Moyen | Revert git, ancien code compatible |
| M3 | Backfill incrémental : `--weapons --force-weapons` sur joueurs cibles | Faible | DELETE+INSERT idempotent |
| M4 | Audit : comparer v1 vs v2 sur 20 matchs de référence | Nul | Lecture seule |
| M5 | Backfill complet : `--weapons --force-weapons --all` | Moyen (temps) | Arrêter et reprendre |

### 11.2 Script d'audit comparatif

```python
# scripts/experimental/audit_weapon_parser_v2.py
# Compare les attributions v1 (en DB) avec les résultats v2 (dry_run)
# Sortie : CSV avec colonnes (match_id, xuid, time_ms, v1_weapon, v2_weapon, v1_conf, v2_conf, changed)
```

### 11.3 Métriques de validation

| Métrique | Seuil d'acceptation |
|----------|:-------------------:|
| Taux de régression (v2 pire que v1) | < 2 % des kills |
| Taux d'amélioration (v2 meilleur que v1) | > 10 % des kills |
| Nouveaux NULL introduits | 0 |
| weapon_id écrasé par sentinel | 0 |
| Tests verts | 100 % |

---

## 12. Risques et mitigations

| # | Risque | Probabilité | Impact | Mitigation |
|---|--------|:-----------:|:------:|------------|
| R1 | Non-POV fire events insuffisants → hybrid obligatoire | Moyenne | Faible (architecture supporte les deux) | Phase 0 décisionnelle avant dev |
| R2 | `map_b2_to_player` coverage partielle (<60%) sur certains matchs | Faible | Moyen | Fallback formula_a universel ; couverture mesurée à ~21% sur test match (baseline acceptable) |
| R3 | Migration reconciled_as casse des requêtes UI | Moyenne | Moyen | Vue `v_weapon_kills` isole les consommateurs |
| R4 | Backfill v2 produit des résultats différents → utilisateur perturbé | Moyenne | Faible | Audit comparatif M4 avant backfill complet |
| R5 | Performance dégradée (logging overhead) | Faible | Faible | Logger optionnel, `DEBUG` désactivé en prod |
| R6 | Claim-and-remove change l'ordrede certaines attributions | Moyenne | Faible | Tests F5-F10 couvrent exhaustivement les cas limites |

---

## 13. Checklist de livraison

### Avant Phase 0

- [ ] Branche `analysis/weapon-parser-rewrite` créée depuis `main`
- [ ] Script d'exploration non-POV écrit et exécuté
- [ ] Décision go/no-go documentée dans `.ai/thought_log.md`

### Avant Phase 1

- [ ] Migration step créée (`add_reconciled_as.py`)
- [ ] 6 tests migration verts
- [ ] Vue `v_weapon_kills` fonctionnelle

### Avant Phase 2

- [ ] `KillAttribution` dataclass définie
- [ ] `correlate_kills()` implémentée avec claim-and-remove (fire_event / formula_a)
- [ ] `scan_fire_events_all()` implémentée (match-level, 0 filtre pi)
- [ ] `scan_melee_events()` implémentée (marqueur 0xd340)
- [ ] `scan_formula_a_ns()` implémentée (NS layer → TYPE IDs)
- [ ] `build_weapon_timeline_ns()` implémentée (TYPE IDs par chunk/pi)
- [ ] `map_b2_to_player()` + `group_events_by_pi()` intégrés dans `weapon_parser.py` (pipeline unifié, plus de module `_player_attribution.py` séparé)
- [ ] `compute_confidence()` publique (ex-`_get_confidence`)
- [ ] `check_zone_b_swap()` corrigée (W1 vs W2 re-évalué)
- [ ] 36 armes dans `WEAPON_ID_MAP` (source acurtis166 mars 2026)
- [ ] `Ninja` + `Pancake` ajoutés à `MELEE_MEDALS`
- [ ] ~90 tests parser verts (groupes A–H + C13 NS)

### Avant Phase 3

- [ ] Service v2 `process_match()` avec nouveaux paramètres
- [ ] Scan Phase cross-chunk fonctionnelle
- [ ] Résolution pi double méthode
- [ ] ~40 tests service verts

### Avant Phase 4

- [ ] `reconciliation.py` créé
- [ ] Bug F corrigé (weapon_id réel jamais écrasé par sentinel)
- [ ] ~15 tests réconciliation verts

### Avant Phase 5

- [ ] `insert_weapon_kill_rows_v2()` avec reconciled_as
- [ ] Requêtes load migrées vers `v_weapon_kills`
- [ ] ~10 tests repo verts

### Avant merge

- [ ] 180+ tests verts
- [ ] Couverture ≥ 90 %
- [ ] Audit 20 matchs comparatif OK
- [ ] Aucune régression > 2 %
- [ ] Aucun `import pandas`
- [ ] Aucun fichier > 500 lignes
- [ ] Aucune fonction > 80 lignes
- [ ] `.ai/thought_log.md` à jour
- [ ] Conventional commit(s) sur la branche
- [ ] `_parser_logging.py`, `reconciliation.py` relus
