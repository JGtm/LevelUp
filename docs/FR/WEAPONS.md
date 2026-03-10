# Attribution des kills par arme — Référence technique

Ce document décrit le système d'attribution des kills par arme utilisé par LevelUp
pour identifier quelle arme est responsable de chaque kill dans un match Halo Infinite,
en s'appuyant sur les données binaires des chunks filmshell récupérés via l'API SPNKr.

---

## Table des matières

1. [Vue d'ensemble](#vue-densemble)
2. [Sources de données](#sources-de-données)
3. [Algorithme d'attribution](#algorithme-dattribution)
   - [Section 2 — Événements de tir POV (§6a)](#section-2--événements-de-tir-pov-6a)
   - [Formula A — Coéquipiers T1 (§6b)](#formula-a--coéquipiers-t1-6b)
4. [Niveaux de confiance](#niveaux-de-confiance)
5. [Noms d'armes spéciaux](#noms-darmes-spéciaux)
6. [WIDs inconnus (format `?hex`)](#wids-inconnus-format-hex)
7. [Schéma de la table `weapon_kills`](#schéma-de-la-table-weapon_kills)
8. [Référence WEAPON_ID_MAP](#référence-weapon_id_map)
9. [Paramètres de timing par arme](#paramètres-de-timing-par-arme)
10. [Résoudre un WID inconnu](#résoudre-un-wid-inconnu)
11. [Ajouter un nouvel identifiant d'arme](#ajouter-un-nouvel-identifiant-darme)

---

## Vue d'ensemble

LevelUp analyse les chunks binaires de type `REPLICATION_DATA` des filmshells pour
extraire l'attribution kill-par-arme de chaque joueur. Les résultats sont stockés
dans la table `weapon_kills` de `shared_matches.duckdb`.

Deux chemins de traitement distincts sont utilisés, selon le rôle du joueur dans le film :

| Chemin | Portée | Source | Confiance |
|--------|--------|--------|-----------|
| **Section 2** (événements de tir) | Joueur POV uniquement | `scan_fire_events()` | high/medium |
| **Formula A** (snapshot) | Tous les autres joueurs (T1) | `build_weapon_timeline()` | high/medium/low |

---

## Sources de données

| Source | Fichier |
|--------|---------|
| Algorithme & parseur | `src/analysis/weapon_parser.py` |
| Map des IDs et timings | `src/analysis/_weapon_data.py` |
| Service d'orchestration | `src/data/services/weapon_extraction_service.py` |
| Écritures en base | `src/data/repositories/_weapon_kills_repo.py` |
| CLI de backfill | `scripts/backfill_data.py --weapons` |

Les chunks sont mis en cache localement dans :

```
data/investigation/chunks/<match_id[:8]>/chunk_NN.bin
```

---

## Algorithme d'attribution

### Section 2 — Événements de tir POV (§6a)

**Portée :** Le joueur unique qui a enregistré le film (POV). Ce joueur est toujours
à `player_index = 1` dans la Section 2 du filmshell — indépendamment de son index
acurtis réel.

**Étapes :**

1. Télécharger tous les chunks `REPLICATION_DATA` qui couvrent une fenêtre de kill
   (`kill_time_ms − KILL_WINDOW_MS` jusqu'à `kill_time_ms`).
2. Appeler `scan_fire_events(chunk, pi=1, ...)` sur chaque chunk. Cette fonction
   retourne une liste d'événements `{weapon_name, timestamp_ms, swap_detected,
   delayed_damage}` pour les armes dont l'ID est dans
   `WEAPON_IDS_INT ∪ COMMON_WEAPON_SUFFIX`.
3. Appeler `correlate_kills_to_weapons(kills, fire_events)` — chaque kill est
   associé à l'événement de tir précédent le plus proche, dans la fenêtre de
   timing propre à cette classe d'arme (`swap_ms`, `travel_max_ms` depuis
   `WEAPON_TIMING`).
4. **Réconciliation API** (`_reconcile_api_aggregates`) : comparer les kills HIGH
   attribués avec la valeur `match_participants.kills` de l'API Halo.
   - Si kills HIGH > kills API → dégrader les kills les moins certains (plus grand
     `delta_ms`) à MEDIUM.
   - Si kills HIGH < kills API → promouvoir des kills MEDIUM (plus petit `delta_ms`
     en premier) en HIGH jusqu'à atteindre le total API.

> **Remarque :** La Section 2 ne peut retourner que des kills pour des armes déjà
> présentes dans `WEAPON_ID_MAP`. Les WIDs inconnus n'apparaissent jamais par ce
> chemin.

---

### Formula A — Coéquipiers T1 (§6b)

**Portée :** Tous les joueurs sauf le POV. Leurs indices de joueur sont résolus via
la *méthode acurtis* (inv #26) — les XUIDs sont associés aux valeurs `player_index`
trouvées dans le premier chunk.

**Étapes :**

1. Appeler `build_weapon_timeline(chunks)` → construit un dictionnaire :
   `timeline[chunk_index][player_index] = wid (bytes)`.
   Chaque entrée enregistre l'arme tenue par chaque joueur à la *snapshot* de ce
   chunk (~19 s/chunk).
2. Pour chaque kill à l'instant `T` :
   - Trouver le chunk couvrant `T` avec `find_chunk_at_time(...)`.
   - Lire `timeline[chunk][player_index]`.
   - Se rabattre sur `timeline[chunk - 1][player_index]` si aucune mise à jour dans
     le chunk courant.
3. Mapper `wid` → nom d'arme via `WEAPON_ID_MAP` :
   - **WID connu :** `confidence = "high"` (dégradé à `"medium"` si un changement
     d'arme a été détecté dans le même chunk — `swap_pis`).
   - **WID inconnu :** stocké verbatim sous la forme `?{wid.hex()}` (16 caractères
     hex), `confidence = "low"`.
   - **Pas de snapshot trouvée :** stocké sous `"UNKNOWN"`, `confidence = "none"`.

> **Important :** Parce que Formula A travaille sur les octets bruts du WID sans
> filtrage, elle **peut** produire des entrées WID inconnues. La Section 2 ne peut
> pas.

---

## Niveaux de confiance

| Valeur | Signification |
|--------|---------------|
| `high` | Arme correspondant avec un timing précis ou snapshot confirmée (réconciliée avec l'API) |
| `medium` | Arme correspondant mais ambiguïté de swap ou de timing détectée |
| `low` | WID inconnu — octets bruts stockés en `?hex`, nom d'arme non résolu |
| `none` | Kill non attribuable (valeurs de repli : `NON TROUVE`, `UNKNOWN`) |

---

## Noms d'armes spéciaux

| Valeur | Origine | Signification |
|--------|---------|---------------|
| `MELEE` | Détection de médaille | Kill attribué à une attaque au corps à corps |
| `GRENADE` | Détection de médaille | Kill attribué à une grenade (type inconnu) |
| `NON TROUVE` | Chemin Section 2 | Événement de tir trouvé mais WID absent de `WEAPON_ID_MAP` |
| `UNKNOWN` | Chemin Formula A | Aucune snapshot d'arme trouvée au moment du kill (cas T0) |
| `?{16 caractères hex}` | Chemin Formula A | WID brut 8 octets non encore présent dans `WEAPON_ID_MAP` |

> Le *cas T0* survient quand un joueur n'est pas le POV **et** que son player_index
> n'a pas pu être résolu via la méthode acurtis (ex : joueur absent du premier chunk).

---

## WIDs inconnus (format `?hex`)

Quand Formula A rencontre un WID absent de `WEAPON_ID_MAP`, elle stocke l'identifiant
complet de 8 octets sous forme de chaîne hexadécimale préfixée par `?` :

```
?91eb16de42c9679f   ← 16 caractères hex = 8 octets
```

Propriétés :
- Le préfixe `?` est intentionnel — il distingue les WIDs non résolus des vrais noms
  d'armes et facilite leur recherche ou filtrage.
- Les 8 octets complets sont préservés pour permettre l'identification future sans
  retraiter les chunks.
- La `confidence` est fixée à `"low"` pour toutes les entrées `?hex`.

Pour lister les WIDs inconnus les plus fréquents dans la base :

```sql
SELECT weapon_name, COUNT(*) AS kills, COUNT(DISTINCT match_id) AS matchs
FROM weapon_kills
WHERE weapon_name LIKE '?%'
GROUP BY weapon_name
ORDER BY kills DESC;
```

---

## Schéma de la table `weapon_kills`

Table située dans `data/warehouse/shared_matches.duckdb`.

| Colonne | Type | Description |
|---------|------|-------------|
| `match_id` | VARCHAR | UUID du match Halo |
| `xuid` | VARCHAR | XUID du joueur (tueur) |
| `victim_xuid` | VARCHAR | XUID de la victime |
| `time_ms` | INTEGER | Horodatage du kill (ms depuis le début du match) |
| `weapon_name` | VARCHAR | Nom de l'arme, `?hex`, `MELEE`, `GRENADE`, `NON TROUVE`, ou `UNKNOWN` |
| `confidence` | VARCHAR | `high`, `medium`, `low`, ou `none` |
| `delta_ms` | INTEGER | Écart entre le dernier événement de tir et le kill (Section 2 uniquement) |
| `swap_detected` | BOOLEAN | Changement d'arme détecté au voisinage du kill |
| `delayed_damage` | BOOLEAN | Temps de vol du projectile susceptible d'avoir amplifié le delta |

Clé primaire : `(match_id, xuid, victim_xuid, time_ms)`.

---

## Référence WEAPON_ID_MAP

Définie dans `src/analysis/_weapon_data.py`. Chaque entrée associe un WID de 8
octets (issu du format binaire filmshell conçu par Andy Curtis / acurtis) à un
nom d'arme lisible.

**Structure d'un WID :**
- Octets 1–4 : identifiant spécifique à l'arme (unique par type/variante).
- Octets 5–8 : suffixe partagé `42c9679f` pour la majorité des armes standard.
  Les armes avec un suffixe différent appartiennent à des familles particulières
  (ex. variantes d'Energy Sword, Mythic Sandwich).

**Organisation des entrées :**

| Groupe | Description |
|--------|-------------|
| Armes standard | `MA40 AR`, `BR75`, `Mk51 Sidekick`, … — suffixe unique par arme |
| Famille Energy Sword | Octets 1–4 identiques (`4ff3937e`), suffixe différent par skin |
| Famille Gravity Hammer | Octets 1–4 identiques (`841ac5e5`), suffixe différent par variante |
| Grenades | Identifiées via snapshot d'inventaire Formula A (pas via événements de tir) |
| Variantes / skins | Même arme en jeu, variante cosmétique — entrée séparée |

---

## Paramètres de timing par arme

Définis dans `WEAPON_TIMING` dans `src/analysis/_weapon_data.py`.

Chaque classe d'arme possède deux paramètres utilisés par `correlate_kills_to_weapons()` :

| Paramètre | Signification |
|-----------|---------------|
| `swap_ms` | Délai physique minimum pour équiper cette arme avant un kill. Les événements de tir antérieurs à `kill_time − swap_ms` sont exclus. |
| `travel_max_ms` | Temps de vol projectile/explosion maximum. Les événements de tir postérieurs à `kill_time − travel_max_ms` sont exclus. |

Fenêtre effective : `[kill_time − swap_ms, kill_time − travel_max_ms]`.

| Classe | `swap_ms` | `travel_max_ms` | Remarque |
|--------|-----------|-----------------|----------|
| Armes de poing (Sidekick, etc.) | 400 | 300 | Swap rapide, hitscan |
| Fusils d'assaut, BR | 650 | 500 | Standard |
| Armes à faisceau / projectiles | 650 | 2000–5000 | Long vol |
| Snipers, Stalker | 900 | 300 | Équipement lent, hitscan |
| Armes lourdes (SPNKr, Hammer, Sword) | 1100 | 1400–2000 | Équipement lent, splash |
| Grenades | 950 | 1350–1650 | Cuisson + vol |

---

## Résoudre un WID inconnu

Quand un WID `?hex` est positivement identifié comme une arme précise (via dump
d'assets, recherche communautaire, ou inspection directe de chunks), la procédure
de résolution comporte **deux étapes obligatoires** :

### Étape 1 — Ajouter dans WEAPON_ID_MAP

Modifier `src/analysis/_weapon_data.py` et ajouter une nouvelle entrée dans
`WEAPON_ID_MAP` :

```python
# Exemple : résolution de ?91eb16de42c9679f en "Nom Arme"
bytes.fromhex("91eb16de42c9679f"): "Nom Arme",  # pragma: allowlist secret
```

Placer l'entrée dans le groupe approprié (standard, grenade, variante, etc.).
Si l'arme présente des caractéristiques de timing différentes du groupe par défaut,
ajouter également une entrée dans `WEAPON_TIMING`.

### Étape 2 — Migrer les lignes existantes

Créer une migration step dans `src/data/migration/steps/` pour mettre à jour les
lignes déjà présentes en base :

```python
# src/data/migration/steps/add_nom_arme_wid.py
from src.data.migration.registry import Migration, register

def apply_schema(conn):
    conn.execute("""
        UPDATE weapon_kills
        SET weapon_name = 'Nom Arme', confidence = 'high'
        WHERE weapon_name = '?91eb16de42c9679f'
    """)

register(Migration(
    name="add_nom_arme_wid",
    target_db="shared",
    description="Résolution WID 91eb16de → Nom Arme",
    apply_schema=apply_schema,
))
```

Enregistrer l'import dans `src/data/migration/steps/__init__.py` :

```python
from src.data.migration.steps import add_nom_arme_wid  # noqa: F401
```

### Étape 3 — Nouveaux matchs

Aucune action supplémentaire. Comme `WEAPON_ID_MAP` est maintenant mis à jour, tous
les futurs appels à `process_match` utiliseront automatiquement le nouveau nom d'arme
pour ce WID.

---

## Ajouter un nouvel identifiant d'arme

Identique à [Résoudre un WID inconnu](#résoudre-un-wid-inconnu), sauf que le WID
n'est jamais apparu en base auparavant (nouvelle arme introduite dans une mise à jour
de saison Halo Infinite) :

1. Ajouter le WID dans `WEAPON_ID_MAP` dans `_weapon_data.py`.
2. Ajouter les paramètres de timing dans `WEAPON_TIMING` si la classe d'arme est
   nouvelle.
3. Aucune migration step nécessaire puisque aucune ligne existante ne référence ce WID.

> **Attention :** Ne pas ajouter de WIDs de manière spéculative en se basant sur une
> similarité structurelle (ex. même famille d'octets). N'ajouter que des WIDs
> positivement identifiés via des sources d'assets fiables ou une inspection directe
> de chunks. Des entrées incorrectes causeraient des attributions erronées sur tous
> les matchs passés et futurs.
