# Plan d'implémentation : Stats PvE (Firefight) en BDD

> **Date** : 2026-02-18
> **Statut** : 📋 En attente de Phase 1 (capture JSON)
> **Auteur** : GitHub Copilot
> **Version** : 1.1 (ajout section bitmask/backfill v5.2 — 2026-02-19)

---

## Table des matières

1. [Contexte et objectifs](#contexte-et-objectifs)
2. [Prérequis : Capture JSON Firefight](#prérequis--capture-json-firefight)
3. [Phase 1 : Recherche API](#phase-1--recherche-api)
4. [Phase 2 : Schéma BDD](#phase-2--schéma-bdd)
5. [Phase 3 : Modèles Python](#phase-3--modèles-python)
6. [Phase 4 : Transformer](#phase-4--transformer)
7. [Phase 5 : Pipeline Sync/Batch Insert](#phase-5--pipeline-syncbatch-insert)
8. [Phase 5b : Architecture Bitmask PvE (v5.2)](#phase-5b--architecture-bitmask-pve-v52)
9. [Phase 6 : Backfill](#phase-6--backfill)
10. [Phase 7 : Citations Covenant](#phase-7--citations-covenant)
11. [Phase 8 : Tests](#phase-8--tests)
12. [Résumé des fichiers](#résumé-des-fichiers)

---

## Contexte et objectifs

### Problème actuel

L'API Halo Infinite retourne des stats spécifiques aux matchs Firefight dans `PlayerTeamStats[].Stats`, mais le code actuel extrait uniquement les **CoreStats** (Kills, Deaths, Assists, etc.). Les stats PvE spécifiques sont ignorées :

- Stats de waves/rounds
- Kills par type d'ennemi (Grunts, Elites, Jackals, Hunters, Brutes, etc.)
- Boss kills, Mythic boss kills

### Objectifs

| Objectif | Description |
|----------|-------------|
| **Stocker les stats PvE** | Waves complétées, boss kills, etc. pour analyse future |
| **Stocker les kills par type d'ennemi** | Pour les citations Covenant (Tueur de Grunts, Tueur d'Élites, etc.) |
| **Respecter l'architecture v5** | Données partagées dans `shared_matches.duckdb` |
| **Intégrer au système de citations** | Nouveau `mapping_type = 'pve_stat'` |

### Ce qui existe déjà

Le transformer actuel (`src/data/sync/transformers.py`) utilise `_find_core_stats_dict()` pour parcourir récursivement `PlayerTeamStats` et extraire les CoreStats. Cette logique sera réutilisée pour les stats PvE.

---

## Prérequis : Capture JSON Firefight

⚠️ **BLOQUANT** : Avant toute implémentation, il faut capturer un JSON brut de match Firefight pour documenter la structure exacte des champs API.

### Option A : Script de capture dédié

Créer `scripts/capture_firefight_json.py` :

```python
#!/usr/bin/env python3
"""Capture un JSON brut de match Firefight pour analyse de structure.

Usage:
    python scripts/capture_firefight_json.py --gamertag MonGT --match-id <UUID>
    python scripts/capture_firefight_json.py --gamertag MonGT --last-firefight
"""

import argparse
import asyncio
import json
import os
from datetime import datetime

from src.data.sync.api_client import get_api_client
from src.utils.auth import load_credentials


async def capture_match_json(gamertag: str, match_id: str | None, last_firefight: bool):
    """Capture et sauvegarde le JSON brut d'un match."""
    creds = load_credentials()
    async with get_api_client(creds) as client:
        # Si --last-firefight, trouver le dernier match Firefight
        if last_firefight:
            xuid = await client.resolve_gamertag(gamertag)
            history = await client.get_match_history(xuid, count=50)
            
            for match in history:
                # Détecter Firefight par playlist_name ou mode_category
                if "firefight" in (match.playlist_name or "").lower() or \
                   "baptême" in (match.playlist_name or "").lower():
                    match_id = match.match_id
                    print(f"Match Firefight trouvé: {match_id}")
                    break
            
            if not match_id:
                print("Aucun match Firefight trouvé dans les 50 derniers matchs.")
                return
        
        # Récupérer le JSON complet
        match_json = await client.get_match_stats_raw(match_id)
        
        # Sauvegarder
        output_dir = ".ai/research/samples"
        os.makedirs(output_dir, exist_ok=True)
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"{output_dir}/firefight_match_{timestamp}.json"
        
        with open(filename, "w", encoding="utf-8") as f:
            json.dump(match_json, f, indent=2, ensure_ascii=False)
        
        print(f"JSON sauvegardé dans: {filename}")
        print(f"Taille: {os.path.getsize(filename) / 1024:.1f} KB")


def main():
    parser = argparse.ArgumentParser(description="Capture JSON Firefight")
    parser.add_argument("--gamertag", required=True, help="Gamertag du joueur")
    parser.add_argument("--match-id", help="ID du match spécifique")
    parser.add_argument("--last-firefight", action="store_true", 
                       help="Capturer le dernier match Firefight")
    
    args = parser.parse_args()
    
    if not args.match_id and not args.last_firefight:
        parser.error("Spécifiez --match-id ou --last-firefight")
    
    asyncio.run(capture_match_json(args.gamertag, args.match_id, args.last_firefight))


if __name__ == "__main__":
    main()
```

### Option B : Mode debug dans sync existant

Modifier temporairement `src/data/sync/engine.py` pour sauvegarder le JSON brut lors d'un sync :

```python
# Dans _process_match() après récupération du JSON
if self._is_firefight_match(match_json):
    import json
    from pathlib import Path
    debug_dir = Path(".ai/research/samples")
    debug_dir.mkdir(parents=True, exist_ok=True)
    with open(debug_dir / f"firefight_{match_id}.json", "w") as f:
        json.dump(match_json, f, indent=2)
```

### Option C : Utiliser l'API directement via curl/Postman

```bash
# Avec les tokens d'authentification
curl -H "x-343-authorization-spartan: $SPARTAN_TOKEN" \
     -H "343-clearance: $CLEARANCE_TOKEN" \
     "https://halostats.svc.halowaypoint.com/hi/matches/{MATCH_ID}/stats" \
     > firefight_match.json
```

### Structure JSON attendue (hypothétique)

Basé sur la documentation Halo Infinite et les patterns observés dans CoreStats :

```json
{
  "MatchId": "abc123...",
  "MatchInfo": {
    "StartTime": "2026-02-18T12:00:00Z",
    "GameVariantCategory": 24,  // Probable ID pour Firefight
    "Playlist": {
      "AssetId": "...",
      "PublicName": "Firefight: Kilo Five"
    }
  },
  "Players": [
    {
      "PlayerId": "xuid(1234567890)",
      "PlayerGamertag": "MonGT",
      "PlayerTeamStats": [
        {
          "TeamId": 0,
          "Stats": {
            "CoreStats": {
              "Kills": 150,
              "Deaths": 5,
              "Assists": 30
              // ... stats communes
            },
            "EliminationStats": {  // <-- STRUCTURE À VALIDER
              "WavesCompleted": 12,
              "MaxWaveReached": 12,
              "TotalEnemyKills": 150,
              "BossKills": 8,
              "MythicBossKills": 2,
              "EnemyKillsByType": {
                "Grunt": 45,
                "Elite": 25,
                "Jackal": 30,
                "Brute": 20,
                "Hunter": 10,
                "Skimmer": 15,
                "Crawler": 3,
                "Soldier": 2
              }
            }
          }
        }
      ]
    }
  ]
}
```

⚠️ **IMPORTANT** : Cette structure est **hypothétique**. Les noms exacts des champs (`EliminationStats`, `EnemyKillsByType`, etc.) doivent être validés avec un JSON réel.

---

## Phase 1 : Recherche API

### Objectif

Documenter la structure exacte des stats PvE dans l'API Halo Infinite.

### Actions

1. **Capturer un JSON brut** de match Firefight (voir section Prérequis)
2. **Analyser la structure** `PlayerTeamStats[].Stats` pour identifier :
   - Nom du bloc PvE (`EliminationStats`, `PveStats`, ou autre)
   - Sous-structures de kills par type d'ennemi
   - Informations de waves/rounds
3. **Documenter** dans `.ai/research/PVE_STATS_API_STRUCTURE.md`

### Template de documentation

Créer `.ai/research/PVE_STATS_API_STRUCTURE.md` après analyse :

```markdown
# Structure API Stats PvE (Firefight)

> Date d'analyse : YYYY-MM-DD
> Match ID analysé : {MATCH_ID}
> Version API : Halo Infinite Season X

## Chemins JSON

### Bloc principal PvE
- **Chemin** : `Players[].PlayerTeamStats[].Stats.{NOM_BLOC}`
- **Nom du bloc** : `{NOM_EXACT}` (ex: EliminationStats, PveStats)

### Champs disponibles

| Champ API | Type | Description | Exemple |
|-----------|------|-------------|---------|
| WavesCompleted | int | Nombre de waves terminées | 12 |
| ... | ... | ... | ... |

### Kills par type d'ennemi

| Type ennemi | Champ API | Valeur exemple |
|-------------|-----------|----------------|
| Grunt | GruntKills | 45 |
| Elite | EliteKills | 25 |
| ... | ... | ... |

## JSON complet (extrait)

```json
{
  "Stats": {
    "{NOM_BLOC}": {
      // Coller ici le bloc réel
    }
  }
}
```
```

### Livrables Phase 1

- [ ] JSON brut sauvegardé dans `.ai/research/samples/`
- [ ] Documentation `.ai/research/PVE_STATS_API_STRUCTURE.md`
- [ ] Liste des champs exploitables validée

---

## Phase 2 : Schéma BDD

### Nouvelle table `pve_match_stats` dans `shared_pve.duckdb`

> ⚠️ **Correction v1.1** : La table va dans `shared_pve.duckdb`, PAS dans `shared_matches.duckdb`.
> Ne pas ajouter `pve_stats_loaded BOOLEAN` à `match_registry` — utiliser `BACKFILL_FLAGS["pve_stats"] = 65536` (voir Phase 5b).

```sql
-- src/data/sync/migrations.py — dans PVE_SCHEMA_DDL (nouveau bloc, shared_pve.duckdb)

CREATE TABLE IF NOT EXISTS pve_match_stats (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,

    -- Stats globales PvE
    waves_completed    INTEGER,
    max_wave_reached   INTEGER,
    boss_kills         INTEGER,
    mythic_boss_kills  INTEGER,
    total_enemy_kills  INTEGER,

    -- Kills par type d'ennemi (Banished)
    grunt_kills    INTEGER DEFAULT 0,
    elite_kills    INTEGER DEFAULT 0,
    jackal_kills   INTEGER DEFAULT 0,
    brute_kills    INTEGER DEFAULT 0,
    hunter_kills   INTEGER DEFAULT 0,
    skimmer_kills  INTEGER DEFAULT 0,

    -- Forerunners (si disponible dans l'API)
    crawler_kills  INTEGER DEFAULT 0,
    soldier_kills  INTEGER DEFAULT 0,
    knight_kills   INTEGER DEFAULT 0,
    warden_kills   INTEGER DEFAULT 0,

    -- Bitmask granulaire (v5.2) — quels champs ont été récupérés
    pve_bits       INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (match_id, xuid)
);

CREATE INDEX IF NOT EXISTS idx_pve_xuid     ON pve_match_stats(xuid);
CREATE INDEX IF NOT EXISTS idx_pve_match_id ON pve_match_stats(match_id);
```

### Justification de la BDD dédiée

| Critère | `shared_pve.duckdb` ✓ | Dans `shared_matches.duckdb` ✗ |
|---------|----------------------|-------------------------------|
| **Pertinence** | 100% des lignes sont PvE | ~10% des matchs seulement |
| **Performance** | Petite DB, scans rapides | Colonne NULL sur 90% des matchs |
| **Cohérence** | Schéma dédié évolutif | Pollue le schéma partagé |
| **Coéquipiers** | Stats accessibles (même xuid) | Idem |

---

## Phase 3 : Modèles Python

### Fichier : `src/data/sync/models.py`

Ajouter après `MatchParticipantRow` :

```python
@dataclass
class PveMatchStatsRow:
    """Ligne pour la table pve_match_stats (Firefight/PvE).
    
    Stocke les stats spécifiques aux matchs PvE pour chaque joueur.
    Les colonnes exactes dépendent de la structure API validée en Phase 1.
    """
    
    match_id: str
    xuid: str
    
    # Stats globales PvE
    waves_completed: int | None = None
    max_wave_reached: int | None = None
    boss_kills: int | None = None
    mythic_boss_kills: int | None = None
    total_enemy_kills: int | None = None
    
    # Kills par type d'ennemi - Banished
    grunt_kills: int = 0
    elite_kills: int = 0
    jackal_kills: int = 0
    brute_kills: int = 0
    hunter_kills: int = 0
    skimmer_kills: int = 0
    
    # Kills par type d'ennemi - Forerunners
    crawler_kills: int = 0
    soldier_kills: int = 0
    knight_kills: int = 0
    warden_kills: int = 0

    # Bitmask granulaire (v5.2) — posé par _update_match_pve_bits() après insertion
    pve_bits: int = 0
```

⚠️ **Note** : Les noms de champs seront ajustés après validation de la structure API en Phase 1.

---

## Phase 4 : Transformer

### Fichier : `src/data/sync/transformers.py`

#### Fonction de recherche du bloc PvE

```python
def _find_pve_stats_dict(player_obj: dict[str, Any]) -> dict[str, Any] | None:
    """Trouve le dictionnaire contenant les stats PvE/Elimination.
    
    Parcourt récursivement PlayerTeamStats pour trouver le bloc de stats PvE.
    
    Args:
        player_obj: Objet joueur du JSON API.
    
    Returns:
        Dict des stats PvE ou None si non trouvé.
    """
    # Clés cibles (à ajuster après Phase 1)
    # TODO: Mettre à jour avec les vrais noms de champs API
    targets = {"WavesCompleted", "BossKills", "TotalEnemyKills"}
    
    def find_pve(x: Any) -> dict[str, Any] | None:
        if isinstance(x, dict):
            # Chemin direct connu (à valider)
            for key in ("EliminationStats", "PveStats", "FirefightStats"):
                if key in x:
                    return x.get(key)
            
            # Détection par clés PvE
            if any(k in x for k in targets):
                return x
            
            # Recherche récursive
            for v in x.values():
                r = find_pve(v)
                if r is not None:
                    return r
        elif isinstance(x, list):
            for v in x:
                r = find_pve(v)
                if r is not None:
                    return r
        return None
    
    return find_pve(player_obj.get("PlayerTeamStats"))


def _extract_enemy_kills_by_type(pve_dict: dict[str, Any]) -> dict[str, int]:
    """Extrait les kills par type d'ennemi.
    
    Gère plusieurs structures possibles :
    - Champs directs : GruntKills, EliteKills, etc.
    - Sous-dict : EnemyKillsByType.Grunt, etc.
    
    Args:
        pve_dict: Dictionnaire des stats PvE.
    
    Returns:
        Dict {enemy_type: kill_count}.
    """
    result: dict[str, int] = {}
    
    # Structure 1 : Champs directs (GruntKills, EliteKills, etc.)
    direct_mappings = {
        "grunt": ["GruntKills", "Grunts"],
        "elite": ["EliteKills", "Elites"],
        "jackal": ["JackalKills", "Jackals"],
        "brute": ["BruteKills", "Brutes"],
        "hunter": ["HunterKills", "Hunters"],
        "skimmer": ["SkimmerKills", "Skimmers"],
        "crawler": ["CrawlerKills", "Crawlers"],
        "soldier": ["SoldierKills", "Soldiers"],
        "knight": ["KnightKills", "Knights"],
        "warden": ["WardenKills", "Wardens"],
    }
    
    for enemy_type, api_keys in direct_mappings.items():
        for key in api_keys:
            val = pve_dict.get(key)
            if val is not None:
                result[enemy_type] = _safe_int(val) or 0
                break
    
    # Structure 2 : Sous-dictionnaire EnemyKillsByType
    by_type = pve_dict.get("EnemyKillsByType") or pve_dict.get("KillsByEnemyType")
    if isinstance(by_type, dict):
        for enemy_type in direct_mappings:
            for key in [enemy_type.capitalize(), enemy_type.upper(), enemy_type]:
                val = by_type.get(key)
                if val is not None:
                    result[enemy_type] = _safe_int(val) or 0
                    break
    
    return result
```

#### `_is_firefight_match()` — détection du mode

```python
# Catégories connues pour Firefight dans GameVariantCategory
# (à valider avec le JSON capturé en Phase 1)
_FIREFIGHT_CATEGORY_IDS: frozenset[int] = frozenset({9, 24})  # Hypothétique

def _is_firefight_match(match_info: dict[str, Any]) -> bool:
    """Retourne True si le match est un mode Firefight/PvE.

    Utilise GameVariantCategory en priorité (plus fiable que playlist_name),
    puis playlist_name en fallback (cas des matchs custom sans catégorie).
    Doit être cohérent avec la condition FIREFIGHT_CONDITION de detection.py.
    """
    # Priorité 1 : GameVariantCategory (ID numérique stable entre saisons)
    category = match_info.get("GameVariantCategory")
    if isinstance(category, int) and category in _FIREFIGHT_CATEGORY_IDS:
        return True

    # Priorité 2 : playlist_name (fallback texte)
    playlist_name = (match_info.get("Playlist") or {}).get("PublicName", "") or ""
    name_lower = playlist_name.lower()
    return (
        "firefight" in name_lower
        or "baptême" in name_lower
        or "survive" in name_lower
    )
```

> **Note** : `_FIREFIGHT_CATEGORY_IDS` doit être validé avec le JSON capturé (Phase 1).
> `GameVariantCategory` est plus fiable que `playlist_name` pour les cas edge (UUID bruts).
> Cette fonction doit rester **cohérente** avec `FIREFIGHT_CONDITION` dans `detection.py`.

#### Fonction principale d'extraction

```python
def extract_pve_stats(match_json: dict[str, Any]) -> list[PveMatchStatsRow]:
    """Extrait les stats PvE de tous les joueurs d'un match Firefight.

    Ne retourne des données que si le match est identifié comme Firefight.
    Extrait les stats pour TOUS les joueurs du match (pas seulement le joueur tracké).

    Args:
        match_json: JSON brut du match (MatchStats).

    Returns:
        Liste de PveMatchStatsRow pour chaque joueur, vide si pas un match PvE.
    """
    # Vérifier si c'est un match Firefight
    match_info = match_json.get("MatchInfo", {})
    if not _is_firefight_match(match_info):
        return []
    
    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str):
        return []
    
    players = match_json.get("Players", [])
    if not isinstance(players, list):
        return []
    
    rows: list[PveMatchStatsRow] = []
    
    for player in players:
        if not isinstance(player, dict):
            continue
        
        # Extraire le XUID
        pid = player.get("PlayerId")
        xuid = None
        if isinstance(pid, str):
            m = XUID_RE.search(pid)
            if m:
                xuid = m.group(1)
        
        if not xuid:
            continue
        
        # Trouver le bloc de stats PvE
        pve_dict = _find_pve_stats_dict(player)
        if pve_dict is None:
            # Pas de stats PvE pour ce joueur (bot ou erreur)
            continue
        
        # Extraire les kills par type d'ennemi
        enemy_kills = _extract_enemy_kills_by_type(pve_dict)
        
        rows.append(PveMatchStatsRow(
            match_id=match_id,
            xuid=xuid,
            waves_completed=_safe_int(pve_dict.get("WavesCompleted")),
            max_wave_reached=_safe_int(pve_dict.get("MaxWaveReached")),
            boss_kills=_safe_int(pve_dict.get("BossKills")),
            mythic_boss_kills=_safe_int(pve_dict.get("MythicBossKills")),
            total_enemy_kills=_safe_int(pve_dict.get("TotalEnemyKills")),
            grunt_kills=enemy_kills.get("grunt", 0),
            elite_kills=enemy_kills.get("elite", 0),
            jackal_kills=enemy_kills.get("jackal", 0),
            brute_kills=enemy_kills.get("brute", 0),
            hunter_kills=enemy_kills.get("hunter", 0),
            skimmer_kills=enemy_kills.get("skimmer", 0),
            crawler_kills=enemy_kills.get("crawler", 0),
            soldier_kills=enemy_kills.get("soldier", 0),
            knight_kills=enemy_kills.get("knight", 0),
            warden_kills=enemy_kills.get("warden", 0),
        ))
    
    return rows
```

---

## Phase 5 : Pipeline Sync/Batch Insert

### Fichier : `src/data/sync/batch_insert.py`

```python
def batch_insert_pve_stats(
    conn: duckdb.DuckDBPyConnection,
    rows: list[PveMatchStatsRow],
) -> int:
    """Insère les stats PvE en batch dans shared_pve.duckdb.
    
    Utilise INSERT OR REPLACE pour gérer les re-syncs.
    
    Args:
        conn: Connexion DuckDB vers shared_matches.duckdb.
        rows: Liste de PveMatchStatsRow à insérer.
    
    Returns:
        Nombre de lignes insérées.
    """
    if not rows:
        return 0
    
    values = [
        (
            r.match_id, r.xuid, r.waves_completed, r.max_wave_reached,
            r.boss_kills, r.mythic_boss_kills, r.total_enemy_kills,
            r.grunt_kills, r.elite_kills, r.jackal_kills, r.brute_kills,
            r.hunter_kills, r.skimmer_kills, r.crawler_kills, r.soldier_kills,
            r.knight_kills, r.warden_kills,
        )
        for r in rows
    ]
    
    conn.executemany("""
        INSERT OR REPLACE INTO pve_match_stats (
            match_id, xuid, waves_completed, max_wave_reached,
            boss_kills, mythic_boss_kills, total_enemy_kills,
            grunt_kills, elite_kills, jackal_kills, brute_kills,
            hunter_kills, skimmer_kills, crawler_kills, soldier_kills,
            knight_kills, warden_kills
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, values)
    
    return len(rows)
```

### Fichier : `src/data/sync/engine.py`

#### Connexion dédiée `shared_pve.duckdb` (nouveau attribut)

`engine.py` suit le pattern lazy-init avec attribut `self._shared_connection`.
Ajouter un troisième attribut `self._pve_connection` :

```python
class DuckDBSyncEngine:
    def __init__(self, ...):
        # ... existants ...
        self._pve_db_path: Path | None = _PROJECT_ROOT / "data" / "warehouse" / "shared_pve.duckdb"
        self._pve_connection: duckdb.DuckDBPyConnection | None = None
        self._pve_db_lock = asyncio.Lock()

    def _get_pve_connection(self) -> duckdb.DuckDBPyConnection:
        """Lazy init de shared_pve.duckdb + création du schéma si absent."""
        if self._pve_connection is not None:
            return self._pve_connection
        self._pve_connection = duckdb.connect(str(self._pve_db_path), read_only=False)
        # Initialiser le schéma PvE (idempotent)
        self._pve_connection.execute(PVE_SCHEMA_DDL)  # défini dans migrations.py
        return self._pve_connection

    def close(self):
        # ... existant ...
        if self._pve_connection:
            self._pve_connection.close()
            self._pve_connection = None
```

#### Intégration dans `_process_new_match()` / `_process_known_match()`

```python
# Dans _process_new_match() ou _process_known_match(), après l'extraction principale :

# Extraction stats PvE pour les matchs Firefight
pve_rows = extract_pve_stats(match_json)
pve_conn = self._get_pve_connection()
if pve_rows:
    async with self._pve_db_lock:
        batch_insert_pve_stats(pve_conn, pve_rows)
        self._update_match_pve_bits(pve_conn, match_id)
        pve_conn.commit()
else:
    # Match non-Firefight : poser le bit guard quand même pour éviter re-détection
    pass  # Le bit 65536 sera posé par _update_match_pve_bits ci-dessous

# Toujours poser le bit guard (même si pas de données PvE)
shared_conn = self._get_shared_connection()
if shared_conn:
    shared_conn.execute(
        "UPDATE match_registry "
        "SET backfill_completed = COALESCE(backfill_completed, 0) | 65536 "
        "WHERE match_id = ?",
        (match_id,),
    )
```

---

## Phase 5b : Architecture Bitmask PvE (v5.2)

> **Contexte** : Le système bitmask v5.2 a été implémenté en session 2026-02-19
> (`ParticipantBits` dans `src/data/sync/constants.py`, `backfill_bits` dans
> `match_participants`). Les stats PvE doivent s'intégrer à cette architecture
> pour éviter les re-détections infinies et permettre la vérification de cohérence.

### Architecture BDD dédiée

Les données PvE vivront dans une **base dédiée** `data/warehouse/shared_pve.duckdb`
(séparation des données PvP et PvE) :

```
data/warehouse/
├── shared_matches.duckdb   ← PvP : match_participants, medals, events, etc.
├── shared_pve.duckdb       ← PvE : pve_match_stats, pve_backfill_registry
└── metadata.duckdb         ← Référentiels
```

> **Pourquoi une DB séparée ?** Les stats PvE (waves, kills par ennemi) n'ont
> aucun sens pour les matchs PvP. Séparer évite de polluer `shared_matches.duckdb`
> avec des colonnes NULL pour 90%+ des matchs.

### Deux niveaux de bitmask PvE

#### Niveau 1 — Guard global dans `match_registry` (shared_matches.duckdb)

Ajouter un bit `pve_stats` dans `BACKFILL_FLAGS` (`src/data/sync/migrations.py`) :

```python
# src/data/sync/migrations.py
BACKFILL_FLAGS = {
    # ... existants (medals=1 ... participants_avg_life=32768) ...
    "pve_stats": 65536,   # bit 16 — PvE stats tenté pour ce match
}
```

Ce bit sert de **guard rapide** dans la détection :
- `0` → stats PvE jamais tentées pour ce match (à backfiller)
- `1` → stats PvE déjà tentées (même si résultat vide — match non-Firefight)

Cela évite une re-détection infinie pour les matchs PvP où `pve_match_stats`
sera naturellement vide.

#### Niveau 2 — Bitmask granulaire dans `pve_match_stats` (shared_pve.duckdb)

Ajouter `pve_bits INTEGER DEFAULT 0` à la table `pve_match_stats` :

```sql
CREATE TABLE IF NOT EXISTS pve_match_stats (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,

    -- Stats globales PvE
    waves_completed    INTEGER,
    max_wave_reached   INTEGER,
    boss_kills         INTEGER,
    mythic_boss_kills  INTEGER,
    total_enemy_kills  INTEGER,

    -- Kills par type d'ennemi (Banished)
    grunt_kills    INTEGER DEFAULT 0,
    elite_kills    INTEGER DEFAULT 0,
    jackal_kills   INTEGER DEFAULT 0,
    brute_kills    INTEGER DEFAULT 0,
    hunter_kills   INTEGER DEFAULT 0,
    skimmer_kills  INTEGER DEFAULT 0,

    -- Forerunners (si disponible)
    crawler_kills  INTEGER DEFAULT 0,
    soldier_kills  INTEGER DEFAULT 0,
    knight_kills   INTEGER DEFAULT 0,
    warden_kills   INTEGER DEFAULT 0,

    -- Bitmask granulaire (v5.2) — quels champs ont été récupérés
    pve_bits       INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid)
);
```

**Note** : Supprimer `pve_stats_loaded BOOLEAN` de `match_registry` — redondant
avec `BACKFILL_FLAGS["pve_stats"]`.

#### `PveBits` enum dans `src/data/sync/constants.py`

```python
class PveBits(IntFlag):
    """Bitmask granulaire pour les champs pve_match_stats.

    Stocké dans pve_match_stats.pve_bits (shared_pve.duckdb).
    Permet de savoir précisément quels champs ont été récupérés par l'API.
    """
    WAVES        = 1        # waves_completed + max_wave_reached
    BOSS_KILLS   = 2        # boss_kills + mythic_boss_kills
    TOTAL_KILLS  = 4        # total_enemy_kills
    GRUNT        = 8        # grunt_kills
    ELITE        = 16       # elite_kills
    JACKAL       = 32       # jackal_kills
    BRUTE        = 64       # brute_kills
    HUNTER       = 128      # hunter_kills
    SKIMMER      = 256      # skimmer_kills
    CRAWLER      = 512      # crawler_kills (Forerunner)
    SOLDIER      = 1024     # soldier_kills (Forerunner)
    KNIGHT       = 2048     # knight_kills  (Forerunner)
    WARDEN       = 4096     # warden_kills  (Forerunner)

    # Combinaisons utiles
    BANISHED_FULL  = GRUNT | ELITE | JACKAL | BRUTE | HUNTER | SKIMMER
    FORERUNNER_ANY = CRAWLER | SOLDIER | KNIGHT | WARDEN
    FULL_PVE       = WAVES | BOSS_KILLS | TOTAL_KILLS | BANISHED_FULL
```

### Mise à jour `pve_bits` après insertion

Dans `engine.py`, après chaque `batch_insert_pve_stats()`, mettre à jour
`pve_bits` via un UPDATE vectorisé (même pattern que `_update_match_participant_bits()`) :

```python
def _update_match_pve_bits(self, pve_conn: duckdb.DuckDBPyConnection, match_id: str) -> None:
    """Met à jour pve_bits pour tous les participants d'un match PvE (vectorisé)."""
    from src.data.sync.constants import PveBits as PB
    try:
        pve_conn.execute(
            """UPDATE pve_match_stats SET pve_bits = COALESCE(pve_bits, 0) | (
                CASE WHEN waves_completed IS NOT NULL THEN ? ELSE 0 END |
                CASE WHEN boss_kills IS NOT NULL THEN ? ELSE 0 END |
                CASE WHEN total_enemy_kills IS NOT NULL THEN ? ELSE 0 END |
                CASE WHEN grunt_kills  > 0 THEN ? ELSE 0 END |
                CASE WHEN elite_kills  > 0 THEN ? ELSE 0 END |
                CASE WHEN jackal_kills > 0 THEN ? ELSE 0 END |
                CASE WHEN brute_kills  > 0 THEN ? ELSE 0 END |
                CASE WHEN hunter_kills > 0 THEN ? ELSE 0 END |
                CASE WHEN skimmer_kills > 0 THEN ? ELSE 0 END |
                CASE WHEN crawler_kills > 0 THEN ? ELSE 0 END |
                CASE WHEN soldier_kills > 0 THEN ? ELSE 0 END |
                CASE WHEN knight_kills > 0 THEN ? ELSE 0 END |
                CASE WHEN warden_kills > 0 THEN ? ELSE 0 END
            ) WHERE match_id = ?""",
            [PB.WAVES, PB.BOSS_KILLS, PB.TOTAL_KILLS,
             PB.GRUNT, PB.ELITE, PB.JACKAL, PB.BRUTE,
             PB.HUNTER, PB.SKIMMER, PB.CRAWLER, PB.SOLDIER,
             PB.KNIGHT, PB.WARDEN, match_id],
        )
        # Mettre à jour le guard global dans shared_matches
        self._shared_conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | 65536 "
            "WHERE match_id = ?",
            (match_id,),
        )
    except Exception as e:
        logger.debug(f"Mise à jour pve_bits match {match_id}: {e}")
```

> **Important** : Le bit `65536` (`pve_stats`) doit être posé dans
> `backfill_completed` **même pour les matchs non-Firefight** (lors du sync,
> après avoir vérifié qu'il ne s'agit pas d'un match PvE). Cela évite de
> re-tenter la détection à chaque backfill.

### Détection Firefight : identifier le mode sans `is_firefight`

`match_registry` n'a **pas** de colonne `is_firefight`. La détection utilise
`playlist_name` :

```python
# Dans detection.py, condition pour les matchs Firefight manquants
FIREFIGHT_CONDITION = """
    (
        LOWER(mr.playlist_name) LIKE '%firefight%'
        OR LOWER(mr.playlist_name) LIKE '%baptême%'
        OR LOWER(mr.playlist_name) LIKE '%survive%'
    )
"""
```

> **Attention** : Certains matchs Firefight ont un UUID brut comme `playlist_name`
> (ex : `dc4929de-216c-43bc-b207-1702253f4576`) — ces cas nécessitent que
> `assets` soit d'abord backfillé pour résoudre le `playlist_name`.

#### Pattern de détection double guard (comme le skill)

```python
# Dans detection.py — pve_stats
if pve_stats:
    if force_pve_stats:
        conditions.append(FIREFIGHT_CONDITION)
    else:
        # Double guard :
        # 1. Match Firefight (par playlist_name)
        # 2. Guard backfill_completed : pve_stats bit (65536) jamais tenté
        conditions.append(
            FIREFIGHT_CONDITION
            + " AND (COALESCE(mr.backfill_completed, 0) & 65536) = 0"
        )
```

> **Pourquoi le double guard est indispensable ici ?**
> Contrairement aux stats PvP, un match Firefight avec `pve_bits = 0` est
> **toujours suspect** (les Firefight ont toujours des waves et des kills
> d'ennemis). Mais le guard `backfill_completed & 65536` évite de re-tenter
> si on sait que l'API n'a pas retourné ces données (bug API, match corrompu).

### `migrate_bits` PvE

Créer `scripts/backfill/migrate_pve_bits.py` sur le modèle de `migrate_bits.py` :

```bash
# Migration initiale des pve_bits depuis les colonnes existantes
python scripts/backfill/migrate_pve_bits.py

# Vérification de cohérence
python scripts/backfill/migrate_pve_bits.py --verify
```

La logique `verify_pve_bits_coherence()` vérifie que chaque colonne `NOT NULL`
a son bit correspondant posé dans `pve_bits` :

```python
checks = [
    ("WAVES",       PveBits.WAVES,      "waves_completed IS NOT NULL"),
    ("BOSS_KILLS",  PveBits.BOSS_KILLS, "boss_kills IS NOT NULL"),
    ("GRUNT",       PveBits.GRUNT,      "grunt_kills > 0"),
    # ...
]
```

---

## Phase 6 : Backfill

### Fichier : `src/data/sync/scope.py`

Ajouter les nouveaux flags :

```python
@dataclass
class SyncScope:
    """Flags de synchronisation et backfill."""
    
    # ... champs existants ...
    
    # PvE Stats
    pve_stats: bool = False
    force_pve_stats: bool = False
    
    # Registres à mettre à jour
    _ALL_DATA_FIELDS: ClassVar[tuple[str, ...]] = (
        # ... existants ...
        "pve_stats",
    )

    _FORCE_MAP: ClassVar[dict[str, str]] = {
        # ... existants ...
        "force_pve_stats": "pve_stats",
    }

    # ⚠️ Ne pas oublier _REQUESTED_TYPE_MAP — utilisé par la propriété
    # `requested_types` pour le routage bitmask backfill_completed.
    # Sans cette entrée, `scope.requested_types` n'inclura pas "pve_stats"
    # et le bit 65536 ne sera pas posé automatiquement.
    _REQUESTED_TYPE_MAP: ClassVar[dict[str, str]] = {
        # ... existants ...
        "pve_stats": "pve_stats",
    }
```

### Fichier : `scripts/backfill/cli.py`

Ajouter les arguments :

```python
parser.add_argument(
    "--pve-stats",
    action="store_true",
    help="Backfill stats PvE (Firefight) pour les matchs manquants"
)
parser.add_argument(
    "--force-pve-stats",
    action="store_true",
    help="Forcer le recalcul des stats PvE même si déjà présentes"
)
```

### Logique de détection des matchs à backfiller

> ⚠️ Ne PAS utiliser `pve_stats_loaded BOOLEAN` ni `is_firefight` — ces colonnes
> n'existent pas. Utiliser `backfill_completed & 65536` (bit `pve_stats`) et
> `playlist_name LIKE '%firefight%'` (voir Phase 5b).

```python
def find_matches_missing_pve_stats(
    shared_conn: duckdb.DuckDBPyConnection,
    *,
    force: bool = False,
    max_matches: int | None = None,
) -> list[str]:
    """Trouve les matchs Firefight sans stats PvE (via shared_matches.duckdb).

    Utilise le double guard bitmask (Phase 5b) :
    - Filtre Firefight par playlist_name
    - Guard backfill_completed & 65536 pour éviter la re-détection

    Args:
        shared_conn: Connexion à shared_matches.duckdb.
        force: Si True, retourne tous les matchs Firefight (ignore le guard).
        max_matches: Limite de résultats.

    Returns:
        Liste des match_id à backfiller.
    """
    firefight_cond = """(
        LOWER(mr.playlist_name) LIKE '%firefight%'
        OR LOWER(mr.playlist_name) LIKE '%baptême%'
        OR LOWER(mr.playlist_name) LIKE '%survive%'
    )"""
    if force:
        where = firefight_cond
    else:
        where = firefight_cond + " AND (COALESCE(mr.backfill_completed, 0) & 65536) = 0"

    query = f"""
        SELECT mr.match_id
        FROM match_registry mr
        WHERE {where}
        ORDER BY mr.start_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        return [row[0] for row in shared_conn.execute(query).fetchall()]
    except Exception as e:
        logger.error(f"Erreur détection matchs PvE: {e}")
        return []
```

#### Intégration dans `detection.py` / `_find_matches_in_shared_all()`

Ajouter dans `_find_matches_in_shared_all()` le cas `pve_stats`, **après** la
section `participants` (en bas de la liste des conditions) :

```python
# PvE stats — voir Phase 5b pour l'explication du double guard
if pve_stats:
    if force_pve_stats:
        conditions.append(
            "(LOWER(mr.playlist_name) LIKE '%firefight%' "
            " OR LOWER(mr.playlist_name) LIKE '%baptême%' "
            " OR LOWER(mr.playlist_name) LIKE '%survive%')"
        )
    else:
        conditions.append(
            "(LOWER(mr.playlist_name) LIKE '%firefight%' "
            " OR LOWER(mr.playlist_name) LIKE '%baptême%' "
            " OR LOWER(mr.playlist_name) LIKE '%survive%')"
            " AND (COALESCE(mr.backfill_completed, 0) & 65536) = 0"
        )
```

#### Orchestrateur : connexion à `shared_pve.duckdb`

Les fonctions backfill PvE reçoivent **deux connexions** :
- `shared_conn` → `shared_matches.duckdb` (pour `match_registry` et mise à jour du guard)
- `pve_conn` → `shared_pve.duckdb` (pour `pve_match_stats`)

```python
# Dans orchestrator.py — backfill PvE
async def _backfill_pve_for_match(
    match_id: str,
    shared_conn: duckdb.DuckDBPyConnection,
    pve_conn: duckdb.DuckDBPyConnection,
    scope: SyncScope,
    api_client: Any,
) -> int:
    """Backfille les stats PvE d'un match Firefight.

    1. Récupère le JSON du match via l'API
    2. Extrait les stats PvE (extract_pve_stats)
    3. Insère dans pve_match_stats (pve_conn)
    4. Met à jour pve_bits (vectorisé)
    5. Pose le bit 65536 dans match_registry.backfill_completed (shared_conn)
    """
    # ... implémentation ...
```

---

## Phase 7 : Citations Covenant

### Fichier : `scripts/create_citation_mappings_table.py`

Ajouter les nouvelles citations :

```python
COVENANT_CITATIONS = [
    # (citation_name_norm, citation_name_display, mapping_type, stat_name, confidence)
    ("tueur_de_grunts", "Tueur de Grunts", "pve_stat", "grunt_kills", "high"),
    ("tueur_d_elites", "Tueur d'Élites", "pve_stat", "elite_kills", "high"),
    ("tueur_de_jackals", "Tueur de Rapaces", "pve_stat", "jackal_kills", "high"),
    ("tueur_de_brutes", "Tueur de Brutes", "pve_stat", "brute_kills", "high"),
    ("tueur_de_hunters", "Tueur de Chasseurs", "pve_stat", "hunter_kills", "high"),
    ("destructeur_de_boss", "Destructeur de boss", "pve_stat", "boss_kills", "high"),
    ("legende_firefight", "Légende Firefight", "pve_stat", "waves_completed", "medium"),
]
```

### Fichier : `src/analysis/citations/engine.py`

Ajouter le support du nouveau `mapping_type` dans `compute_citation_for_match()` :

```python
def compute_citation_for_match(
    self,
    mapping: dict[str, Any],
    *,
    match_medals: dict[int, int] | None = None,
    match_stats: dict[str, Any] | None = None,
    match_awards: dict[str, int] | None = None,
    match_pve_stats: dict[str, int] | None = None,  # NOUVEAU
    df_match: pl.DataFrame | None = None,
) -> int:
    """Calcule la valeur d'une citation pour un match."""
    mtype = mapping.get("mapping_type", "")
    
    # ... types existants (medal, stat, award, custom) ...
    
    if mtype == "pve_stat":
        stat_name = mapping.get("stat_name", "")
        if stat_name and match_pve_stats:
            return match_pve_stats.get(stat_name, 0)
        return 0
    
    return 0
```

### Chargement des stats PvE pour le calcul

Dans `aggregate_for_display()`, charger les stats PvE depuis `shared_pve.duckdb` :

```python
def _load_pve_stats_for_matches(self, match_ids: list[str]) -> dict[str, dict[str, int]]:
    """Charge les stats PvE pour une liste de matchs depuis shared_pve.duckdb.

    Returns:
        Dict {match_id: {grunt_kills: X, elite_kills: Y, ...}}
    """
    if not match_ids:
        return {}

    pve_path = _PROJECT_ROOT / "data" / "warehouse" / "shared_pve.duckdb"
    if not pve_path.exists():
        return {}
    conn = duckdb.connect(str(pve_path), read_only=True)
    try:
        placeholders = ",".join(["?"] * len(match_ids))
        rows = conn.execute(f"""
            SELECT match_id, grunt_kills, elite_kills, jackal_kills, brute_kills,
                   hunter_kills, skimmer_kills, waves_completed, boss_kills
            FROM pve_match_stats
            WHERE match_id IN ({placeholders}) AND xuid = ?
        """, (*match_ids, self._xuid)).fetchall()
        
        result = {}
        for row in rows:
            result[row[0]] = {
                "grunt_kills": row[1] or 0,
                "elite_kills": row[2] or 0,
                "jackal_kills": row[3] or 0,
                "brute_kills": row[4] or 0,
                "hunter_kills": row[5] or 0,
                "skimmer_kills": row[6] or 0,
                "waves_completed": row[7] or 0,
                "boss_kills": row[8] or 0,
            }
        return result
    finally:
        conn.close()
```

---

## Phase 8 : Tests

### Tests unitaires : `tests/test_pve_transformers.py`

```python
"""Tests pour l'extraction des stats PvE."""

import pytest

from src.data.sync.transformers import extract_pve_stats, _find_pve_stats_dict


class TestFindPveStatsDict:
    """Tests pour _find_pve_stats_dict."""
    
    def test_returns_none_for_pvp_match(self):
        """Un match PvP sans stats PvE retourne None."""
        player = {
            "PlayerTeamStats": [
                {"Stats": {"CoreStats": {"Kills": 10, "Deaths": 5}}}
            ]
        }
        assert _find_pve_stats_dict(player) is None
    
    def test_finds_elimination_stats(self):
        """Trouve le bloc EliminationStats."""
        player = {
            "PlayerTeamStats": [
                {
                    "Stats": {
                        "CoreStats": {"Kills": 100},
                        "EliminationStats": {
                            "WavesCompleted": 12,
                            "BossKills": 5
                        }
                    }
                }
            ]
        }
        result = _find_pve_stats_dict(player)
        assert result is not None
        assert result.get("WavesCompleted") == 12


class TestExtractPveStats:
    """Tests pour extract_pve_stats."""
    
    def test_returns_empty_for_non_firefight(self):
        """Retourne liste vide si pas un match Firefight."""
        match = {
            "MatchId": "abc123",
            "MatchInfo": {"Playlist": {"PublicName": "Slayer"}},
            "Players": []
        }
        assert extract_pve_stats(match) == []
    
    def test_extracts_all_players(self):
        """Extrait les stats pour tous les joueurs."""
        match = {
            "MatchId": "abc123",
            "MatchInfo": {"Playlist": {"PublicName": "Firefight: Kilo Five"}},
            "Players": [
                {
                    "PlayerId": "xuid(111)",
                    "PlayerTeamStats": [{"Stats": {"EliminationStats": {"GruntKills": 50}}}]
                },
                {
                    "PlayerId": "xuid(222)",
                    "PlayerTeamStats": [{"Stats": {"EliminationStats": {"GruntKills": 30}}}]
                }
            ]
        }
        rows = extract_pve_stats(match)
        assert len(rows) == 2
        assert rows[0].grunt_kills == 50
        assert rows[1].grunt_kills == 30
```

### Tests intégration : `tests/integration/test_pve_sync.py`

```python
"""Tests d'intégration pour le pipeline PvE."""

import pytest
import duckdb

from src.data.sync.batch_insert import batch_insert_pve_stats
from src.data.sync.models import PveMatchStatsRow


@pytest.fixture
def shared_db(tmp_path):
    """Crée une DB PvE temporaire avec le schéma PvE."""
    db_path = tmp_path / "shared_pve.duckdb"
    conn = duckdb.connect(str(db_path))
    conn.execute("""
        CREATE TABLE pve_match_stats (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            grunt_kills INTEGER DEFAULT 0,
            elite_kills INTEGER DEFAULT 0,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    conn.close()
    return db_path


def test_batch_insert_pve_stats(shared_db):
    """Test insertion batch des stats PvE."""
    rows = [
        PveMatchStatsRow(match_id="m1", xuid="x1", grunt_kills=50, elite_kills=25),
        PveMatchStatsRow(match_id="m1", xuid="x2", grunt_kills=30, elite_kills=15),
    ]
    
    conn = duckdb.connect(str(shared_db))
    count = batch_insert_pve_stats(conn, rows)
    conn.close()
    
    assert count == 2
    
    # Vérifier l'insertion
    conn = duckdb.connect(str(shared_db), read_only=True)
    result = conn.execute("SELECT COUNT(*) FROM pve_match_stats").fetchone()[0]
    conn.close()
    
    assert result == 2
```

---

## Résumé des fichiers

### Fichiers à modifier

| Fichier | Modifications |
|---------|---------------|
| `src/data/sync/models.py` | Ajouter `PveMatchStatsRow` (avec champ `pve_bits`) |
| `src/data/sync/transformers.py` | Ajouter `_find_pve_stats_dict()`, `_extract_enemy_kills_by_type()`, `extract_pve_stats()` |
| `src/data/sync/batch_insert.py` | Ajouter `batch_insert_pve_stats()` |
| `src/data/sync/constants.py` | Ajouter `PveBits` enum (après `ParticipantBits`, `MatchBits`) |
| `src/data/sync/migrations.py` | DDL `pve_match_stats` dans **`shared_pve.duckdb`** ; bit `pve_stats=65536` dans `BACKFILL_FLAGS` |
| `src/data/sync/engine.py` | Intégrer extraction PvE + `_update_match_pve_bits()` ; poser bit 65536 dans `backfill_completed` |
| `src/data/sync/scope.py` | Ajouter champs `pve_stats`, `force_pve_stats` dans `SyncScope` |
| `scripts/backfill/cli.py` | Ajouter `--pve-stats`, `--force-pve-stats` |
| `scripts/backfill/detection.py` | Ajouter condition `pve_stats` avec double guard dans `_find_matches_in_shared_all()` |
| `scripts/backfill/orchestrator.py` | Ajouter `_backfill_pve_for_match()` avec connexion `pve_conn` |
| `scripts/create_citation_mappings_table.py` | Ajouter citations Covenant |
| `src/analysis/citations/engine.py` | Supporter `pve_stat` mapping_type |
| `docs/SHARED_MATCHES_SCHEMA.md` | Documenter `shared_pve.duckdb` + `pve_match_stats` |

### Fichiers à créer

| Fichier | Description |
|---------|-------------|
| `scripts/backfill/migrate_pve_bits.py` | Migration initiale `pve_bits` + `--verify` (cohérence) |
| `.ai/research/PVE_STATS_API_STRUCTURE.md` | Documentation structure API (après Phase 1) |
| `.ai/research/samples/*.json` | Samples JSON Firefight capturés |
| `tests/test_pve_transformers.py` | Tests unitaires extraction PvE |
| `tests/integration/test_pve_sync.py` | Tests intégration pipeline PvE |

### Points d'attention v5.2

| Sujet | Règle |
|-------|-------|
| **BDD séparée** | `shared_pve.duckdb` — ne pas polluer `shared_matches.duckdb` |
| **Pas de colonne booléenne** | Utiliser `BACKFILL_FLAGS["pve_stats"] = 65536` au lieu de `pve_stats_loaded BOOLEAN` |
| **Détection Firefight** | `playlist_name LIKE '%firefight%'` — pas de colonne `is_firefight` |
| **Guard double** | `playlist + (backfill_completed & 65536) = 0` — même pattern que skill |
| **UUID playlist** | Certains matchs Firefight ont UUID brut → backfiller `assets` d'abord |
| **Modes non-ranked** | Firefight = toujours non-ranked → `team_mmr = NULL` normal, ne pas utiliser `backfill_bits & TEAM_MMR` pour détecter les Firefight |
| **`migrate_pve_bits.py`** | À lancer UNE SEULE FOIS après la première insertion de données PvE |

---

## Checklist d'implémentation

- [ ] **Phase 1** : Capturer JSON Firefight et documenter structure API
- [ ] **Phase 2** : Créer schéma SQL (`shared_pve.duckdb` + `pve_bits` + **pas** de `pve_stats_loaded BOOLEAN`)
- [ ] **Phase 3** : Ajouter `PveMatchStatsRow` dans models.py
- [ ] **Phase 4** : Implémenter transformer `extract_pve_stats()`
- [ ] **Phase 5** : Implémenter `batch_insert_pve_stats()` et intégrer dans engine
- [ ] **Phase 5b** : Ajouter `PveBits` dans `constants.py`, bit `pve_stats=65536` dans `BACKFILL_FLAGS`, `_update_match_pve_bits()` dans engine
- [ ] **Phase 6** : Ajouter flags SyncScope + CLI + `find_matches_missing_pve_stats()` + intégration `detection.py`
- [ ] **Phase 6b** : Créer `scripts/backfill/migrate_pve_bits.py` (migration initiale + `--verify`)
- [ ] **Phase 7** : Ajouter citations Covenant et support `pve_stat`
- [ ] **Phase 8** : Écrire et valider les tests
- [ ] **Documentation** : Mettre à jour SHARED_MATCHES_SCHEMA.md + ajouter `shared_pve.duckdb`
- [ ] **CHANGELOG** : Ajouter entrée pour la feature

---

## Estimation effort

| Phase | Effort | Dépendances |
|-------|--------|-------------|
| Phase 1 (Recherche API) | 0.5h | Aucune |
| Phase 2 (Schéma BDD) | 0.5h | Phase 1 |
| Phase 3 (Modèles) | 0.25h | Phase 1 |
| Phase 4 (Transformer) | 1h | Phase 1, 3 |
| Phase 5 (Pipeline) | 1h | Phase 2, 4 |
| Phase 5b (Bitmask PvE) | 0.75h | Phase 2, constants.py existant |
| Phase 6 (Backfill + detection) | 0.75h | Phase 5, 5b |
| Phase 7 (Citations) | 0.5h | Phase 5 |
| Phase 8 (Tests) | 1h | Phase 4, 5, 5b |
| **Total** | **~6.25h** | |

---

## Risques et mitigations

| Risque | Impact | Mitigation |
|--------|--------|------------|
| Structure API différente de l'attendu | Élevé | Phase 1 bloquante, validation avant implémentation |
| Pas de stats par type d'ennemi dans l'API | Moyen | Peut-être disponible via médailles ou awards à la place |
| Matchs Firefight historiques sans re-fetch possible | Moyen | Rate limit API à respecter, prioriser matchs récents |
| Changement d'API lors d'une mise à jour Halo | Faible | Extraction récursive flexible, logging des erreurs |
| `GameVariantCategory` inconnu pour Firefight | Moyen | Valider l'ID en Phase 1 ; `_is_firefight_match()` a un fallback `playlist_name` |
| Incohérence détection `_is_firefight_match()` vs `FIREFIGHT_CONDITION` | Élevé | Les deux doivent utiliser les mêmes critères — tester avec un Firefight + un match social |
| Concurrence sur `shared_pve.duckdb` | Faible | Utiliser `self._pve_db_lock` (même pattern que `_db_lock` et `_shared_db_lock`) |
