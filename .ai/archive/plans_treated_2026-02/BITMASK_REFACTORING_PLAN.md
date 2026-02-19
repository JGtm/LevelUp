# Plan de Refactoring : Bitmask Granulaire par Joueur

**Date** : 2026-02-19  
**Statut** : À valider  
**Priorité** : Haute (optimisation coûts API)

---

## 1. Inventaire Complet des Données

### 1.1 Tables Shared (`shared_matches.duckdb`)

#### match_registry (par match)
```
match_id, start_time, end_time, duration_seconds
playlist_id, playlist_name, map_id, map_name
pair_id, pair_name, game_variant_id, game_variant_name
mode_category, is_ranked, is_firefight
team_0_score, team_1_score
backfill_completed (bitmask actuel)
participants_loaded, events_loaded, medals_loaded (flags booléens)
first_sync_by, first_sync_at, last_updated_at, player_count
```

#### match_participants (par joueur × match)
```
match_id, xuid, gamertag, team_id, outcome, rank, score
kills, deaths, assists, kda
shots_fired, shots_hit, accuracy
damage_dealt, damage_taken
avg_life_seconds, time_played_seconds
headshot_kills, max_killing_spree
grenade_kills, melee_kills, power_weapon_kills
personal_score
team_mmr, enemy_mmr
kills_expected, kills_stddev
deaths_expected, deaths_stddev
assists_expected, assists_stddev
```

#### medals_earned (par joueur × match)
```
match_id, xuid, medal_name_id, count
```

#### highlight_events (par match)
```
id, match_id, event_type, time_ms, xuid, gamertag, type_hint, raw_json
```

#### killer_victim_pairs (par match)
```
match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag
kill_count, time_ms, is_validated
```

#### xuid_aliases (global)
```
xuid, gamertag, last_seen, source
```

### 1.2 Tables Player DB (`stats.duckdb` par joueur)

#### player_match_enrichment (par match, joueur courant)
```
match_id, performance_score, session_id, session_label
is_with_friends, teammates_signature, known_teammates_count, friends_xuids
```

#### personal_score_awards (par match, joueur courant)
```
match_id, xuid, award_name, award_category, award_count, award_score
```

#### match_citations (par match)
```
match_id, citation_name_norm, value
```

#### backfill_status (EXISTE DÉJÀ !)
```
match_id
attempted_medals, attempted_events, attempted_skill
attempted_personal_scores, attempted_performance_scores
attempted_aliases, attempted_accuracy, attempted_shots
attempted_enemy_mmr, attempted_assets, attempted_participants
attempted_participants_scores, attempted_participants_kda
attempted_participants_shots, attempted_participants_damage
last_attempt_at
```

### 1.3 Sources API

| API | Données retournées | Granularité |
|-----|-------------------|-------------|
| `get_match_stats` | Stats de base (kills, deaths, assists, score...) | Tous joueurs |
| `get_skill_stats` | MMR + expected/stddev | Tous joueurs |
| `get_match_overview` | Highlight events | Par match |
| `get_career_rank` | Career progression | Joueur courant |

---

## 2. Problème Actuel

### 2.1 Architecture existante (problématique)

```
match_registry.backfill_completed (INTEGER bitmask)
├── Bit 0 (1)    : medals         ← PAR JOUEUR mais bitmask par match !
├── Bit 1 (2)    : events         ← OK (par match)
├── Bit 2 (4)    : skill          ← PAR JOUEUR + trop générique (8 colonnes)
├── Bit 3 (8)    : personal_scores← PAR JOUEUR  
├── Bit 4 (16)   : performance    ← PAR JOUEUR
├── Bit 5 (32)   : accuracy       ← PAR JOUEUR
├── Bit 6 (64)   : shots          ← PAR JOUEUR
├── Bit 7 (128)  : assets         ← OK (par match)
├── Bit 8 (256)  : aliases        ← OK (par match)
├── Bit 9 (512)  : participants   ← OK (par match)
└── ...
```

**Table backfill_status existe déjà** mais :
- Dans player DB (pas shared)
- Flags booléens `attempted_*` (pas de valeur réelle)
- Ne track pas les colonnes individuelles

### 2.2 Problèmes identifiés

| Problème | Impact |
|----------|--------|
| Bitmask dans `match_registry` | Granularité par **match**, pas par **joueur** |
| "skill" trop générique | Regroupe 8 colonnes (mmr, expected, stddev) |
| Données joueur vs match | Medals, skill sont par joueur mais bitmask par match |
| `--force-skill` seule option | Oblige à tout recharger si 1 colonne manque |
| Colonnes oubliées | headshot_kills, max_killing_spree, kda, time_played non trackés |
| backfill_status local | Pas accessible multi-joueurs |

### 2.3 Scénario problématique

1. JGtm sync le match `04f7d9d5`
2. API Skill appelée → données de tous les joueurs retournées
3. **Bug corrigé** : maintenant tous les joueurs reçoivent les données
4. Bitmask `skill=1` dans `match_registry`
5. Madina97294 sync plus tard → match existe, bitmask OK → **skip**
6. Mais si Madina avait des données NULL → pas de backfill possible sans `--force-skill`

### 2.4 Coût actuel d'un backfill skill

```
--skill --force-skill pour 1 joueur, 1000 matchs :
├── 1000 appels API Skill (rate limited)
├── 8000 UPDATEs (8 colonnes × 1000)
├── Temps : ~15-30 minutes
└── Données inutiles : 95% déjà présentes
```

---

## 3. Solution Proposée

### 3.1 Nouveau schéma : bitmask dans `match_participants`

```sql
-- Migration
ALTER TABLE match_participants ADD COLUMN backfill_bits INTEGER DEFAULT 0;

-- Index pour la détection rapide
CREATE INDEX idx_mp_backfill ON match_participants(xuid, backfill_bits);
```

### 3.2 Définition des bits (par joueur) — COMPLET

| Bit | Valeur | Nom technique | Colonnes concernées | Source API |
|-----|--------|---------------|---------------------|------------|
| 0 | 1 | `BIT_TEAM_MMR` | team_mmr | get_skill_stats |
| 1 | 2 | `BIT_ENEMY_MMR` | enemy_mmr | get_skill_stats |
| 2 | 4 | `BIT_KILLS_EXP` | kills_expected, kills_stddev | get_skill_stats |
| 3 | 8 | `BIT_DEATHS_EXP` | deaths_expected, deaths_stddev | get_skill_stats |
| 4 | 16 | `BIT_ASSISTS_EXP` | assists_expected, assists_stddev | get_skill_stats |
| 5 | 32 | `BIT_ACCURACY` | accuracy | get_match_stats |
| 6 | 64 | `BIT_SHOTS` | shots_fired, shots_hit | get_match_stats |
| 7 | 128 | `BIT_DAMAGE` | damage_dealt, damage_taken | get_match_stats |
| 8 | 256 | `BIT_AVG_LIFE` | avg_life_seconds | get_match_stats |
| 9 | 512 | `BIT_MEDALS` | (medals_earned pour xuid) | get_match_stats |
| 10 | 1024 | `BIT_GRENADE_KILLS` | grenade_kills | get_match_stats |
| 11 | 2048 | `BIT_MELEE_KILLS` | melee_kills | get_match_stats |
| 12 | 4096 | `BIT_POWER_WEAPON` | power_weapon_kills | get_match_stats |
| 13 | 8192 | `BIT_PERSONAL_SCORE` | personal_score | get_match_stats |
| 14 | 16384 | `BIT_HEADSHOT_KILLS` | headshot_kills | get_match_stats |
| 15 | 32768 | `BIT_MAX_SPREE` | max_killing_spree | get_match_stats |
| 16 | 65536 | `BIT_KDA` | kda (calculé) | calculé |
| 17 | 131072 | `BIT_TIME_PLAYED` | time_played_seconds | get_match_stats |
| 18 | 262144 | `BIT_KILLER_VICTIM` | (killer_victim_pairs) | get_match_stats |

**Réservés pour le futur** : bits 19-31

### 3.3 Bitmask match_registry (données par match uniquement)

Le bitmask dans `match_registry` reste UNIQUEMENT pour les données par match :

| Bit | Valeur | Donnée | Source |
|-----|--------|--------|--------|
| 0 | 1 | highlight_events | get_match_overview |
| 1 | 2 | assets (map_name, playlist_name...) | get_match_stats |
| 2 | 4 | aliases (extraction gamertags) | get_match_stats |
| 3 | 8 | killer_victim_pairs_loaded | get_match_stats |

> **Clarification KILLER_VICTIM** : `BIT_KILLER_VICTIM` (bit 18, valeur 262144) dans `ParticipantBits`
> indique que **ce joueur spécifique** a des entrées dans `killer_victim_pairs` pour ce match.
> Le bit 3 de `match_registry` indique que **le chargement global** des paires a été effectué
> pour ce match. Les deux coexistent : le bit match dit « on a tenté le chargement global »,
> le bit participant dit « ce joueur est présent dans les paires ».

### 3.4 Données Player DB (tracking séparé)

Ces données restent trackées dans `player_match_enrichment` ou table dédiée :

| Donnée | Table | Tracking |
|--------|-------|----------|
| performance_score | player_match_enrichment | Colonne NOT NULL |
| session_id | player_match_enrichment | Colonne NOT NULL |
| personal_score_awards | personal_score_awards | EXISTS check |
| match_citations | match_citations | EXISTS check |

### 3.5 Groupes logiques (raccourcis CLI)

| Groupe | Bits inclus | Colonnes | API nécessaire |
|--------|-------------|----------|----------------|
| `--mmr` | 0, 1 | team_mmr, enemy_mmr | get_skill_stats |
| `--expected` | 2, 3, 4 | kills/deaths/assists expected+stddev | get_skill_stats |
| `--skill` | 0-4 | Tous MMR + expected | get_skill_stats |
| `--combat` | 5, 6, 7 | accuracy, shots, damage | get_match_stats |
| `--kills-detail` | 10, 11, 12, 14 | grenade, melee, power_weapon, headshot | get_match_stats |
| `--core-stats` | 5-17 | Toutes stats de combat | get_match_stats |

---

## 4. Nouvelles Options CLI

### 4.1 Options individuelles

```bash
# MMR
--team-mmr              # Backfill si team_mmr IS NULL
--enemy-mmr             # Backfill si enemy_mmr IS NULL
--mmr                   # = --team-mmr --enemy-mmr

# Expected values
--kills-expected        # Backfill si kills_expected IS NULL
--deaths-expected       # Backfill si deaths_expected IS NULL  
--assists-expected      # Backfill si assists_expected IS NULL
--expected              # = --kills-expected --deaths-expected --assists-expected

# Groupe skill (rétrocompatibilité)
--skill                 # = --mmr --expected (comportement actuel)

# Combat stats
--accuracy              # Backfill si accuracy IS NULL
--shots                 # Backfill si shots_fired/shots_hit IS NULL
--damage                # Backfill si damage_dealt/damage_taken IS NULL
--combat                # = --accuracy --shots --damage

# Kills détail
--grenade-kills         # Backfill si grenade_kills IS NULL
--melee-kills           # Backfill si melee_kills IS NULL
--power-weapon-kills    # Backfill si power_weapon_kills IS NULL
--headshot-kills        # Backfill si headshot_kills IS NULL
--max-spree             # Backfill si max_killing_spree IS NULL
--kills-detail          # = tous les kills-*

# Stats supplémentaires
--avg-life              # Backfill si avg_life_seconds IS NULL
--kda                   # Backfill si kda IS NULL (calculé)
--time-played           # Backfill si time_played_seconds IS NULL
--killer-victim         # Backfill killer_victim_pairs pour ce match
--core-stats            # = accuracy, shots, damage, avg-life, kills-detail, kda, time-played

# Medals (déjà existant)
--medals                # Backfill médailles manquantes
```

### 4.2 Options force (toutes)

```bash
# MMR
--force-team-mmr
--force-enemy-mmr
--force-mmr             # = --force-team-mmr --force-enemy-mmr

# Expected
--force-kills-expected
--force-deaths-expected
--force-assists-expected
--force-expected        # = tous les force-*-expected
--force-skill           # = --force-mmr --force-expected

# Combat
--force-accuracy
--force-shots
--force-damage
--force-combat          # = tous les force combat

# Kills détail
--force-grenade-kills
--force-melee-kills
--force-power-weapon-kills
--force-headshot-kills
--force-max-spree
--force-kills-detail    # = tous les force kills

# Stats supplémentaires
--force-avg-life
--force-kda
--force-time-played
--force-killer-victim
--force-core-stats      # = tous les force core-stats

# Medals
--force-medals
```

### 4.3 Exemple d'usage

```bash
# Cas 1 : Seulement les kills_expected manquants
python scripts/backfill_data.py --player JGtm --kills-expected

# Cas 2 : Tout le MMR manquant pour tous les joueurs
python scripts/backfill_data.py --all --mmr

# Cas 3 : Forcer rechargement accuracy (données corrompues)
python scripts/backfill_data.py --player Madina97294 --force-accuracy

# Cas 4 : Rétrocompatibilité (ancien comportement)
python scripts/backfill_data.py --player JGtm --skill
```

---

## 5. Modifications par Fichier

### 5.0 Sync Engine — Adaptation principale

Le script de sync doit être adapté pour :
1. Mettre à jour `match_participants.backfill_bits` au lieu de `match_registry.backfill_completed` pour les données par joueur
2. Calculer les bits individuels lors de l'insertion des participants
3. Gérer la transition pendant la période de migration

#### 5.0.1 État actuel (`src/data/sync/engine.py`)

```python
# Définition actuelle (migrations.py)
BACKFILL_FLAGS = {
    "medals": 1 << 0,           # 1
    "events": 1 << 1,           # 2
    "skill": 1 << 2,            # 4  ← Trop grossier (8 colonnes)
    "personal_scores": 1 << 3,  # 8
    "performance_scores": 1 << 4, # 16
    "accuracy": 1 << 5,         # 32
    "shots": 1 << 6,            # 64
    "enemy_mmr": 1 << 7,        # 128
    "assets": 1 << 8,           # 256
    "participants": 1 << 9,     # 512
    "participants_scores": 1 << 10,   # 1024
    "participants_kda": 1 << 11,      # 2048
    "participants_shots": 1 << 12,    # 4096
    "participants_damage": 1 << 13,   # 8192
    "aliases": 1 << 14,         # 16384
    "participants_avg_life": 1 << 15, # 32768
}

# Usage actuel
def _compute_backfill_mask(self, options: SyncOptions) -> int:
    bf_mask = 0
    bf_mask |= BACKFILL_FLAGS["medals"]
    # ... etc
    return bf_mask

# Application au match_registry (problématique)
shared_conn.execute(
    "UPDATE match_registry "
    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
    "WHERE match_id = ?",
    (bf_mask, match_id),
)
```

#### 5.0.2 Nouvelle logique

```python
# Nouveau fichier : src/data/sync/constants.py
class MatchBits:
    """Bitmask pour match_registry.backfill_completed (données par match).

    ⚠️ IMPORTANT — Compatibilité avec BACKFILL_FLAGS existants :
    La colonne backfill_completed contient déjà des données avec l'ancien
    BACKFILL_FLAGS (medals=1, events=2, skill=4, assets=256, aliases=16384...).
    Les valeurs MatchBits doivent être choisies pour NE PAS entrer en collision
    avec ces anciens bits, afin de coexister pendant la période de transition.

    Anciens bits occupés (BACKFILL_FLAGS) :
        medals=1, events=2, skill=4, personal_scores=8, performance=16,
        accuracy=32, shots=64, enemy_mmr=128, assets=256, participants=512,
        participants_scores=1024, participants_kda=2048, participants_shots=4096,
        participants_damage=8192, aliases=16384, participants_avg_life=32768

    → Premiers bits libres disponibles : à partir de 1 << 16 (65536)

    Stratégie retenue : utiliser les bits hauts (≥ 16) pour les nouvelles
    données match afin d'éviter toute collision avec l'existant.
    """
    EVENTS = 1 << 16           # 65536  — highlight_events (remplace bit 2 legacy)
    ASSETS = 1 << 17           # 131072 — map_name, playlist_name résolu
    ALIASES = 1 << 18          # 262144 — xuid_aliases extraits
    KILLER_VICTIM_LOADED = 1 << 19  # 524288 — killer_victim_pairs chargés

class ParticipantBits:
    """Bitmask pour match_participants.backfill_bits (données par joueur)."""
    TEAM_MMR = 1 << 0         # 1
    ENEMY_MMR = 1 << 1        # 2
    # ... (complet comme défini dans la section 3.2)

# Nouvelle fonction dans engine.py
def _compute_participant_bits(self, participant_data: dict) -> int:
    """Calcule le bitmask pour un participant basé sur les données présentes."""
    bits = 0
    if participant_data.get("team_mmr") is not None:
        bits |= ParticipantBits.TEAM_MMR
    if participant_data.get("enemy_mmr") is not None:
        bits |= ParticipantBits.ENEMY_MMR
    if participant_data.get("kills_expected") is not None:
        bits |= ParticipantBits.KILLS_EXP
    if participant_data.get("deaths_expected") is not None:
        bits |= ParticipantBits.DEATHS_EXP
    if participant_data.get("assists_expected") is not None:
        bits |= ParticipantBits.ASSISTS_EXP
    if participant_data.get("accuracy") is not None:
        bits |= ParticipantBits.ACCURACY
    if participant_data.get("shots_fired") is not None:
        bits |= ParticipantBits.SHOTS
    if participant_data.get("damage_dealt") is not None:
        bits |= ParticipantBits.DAMAGE
    if participant_data.get("avg_life_seconds") is not None:
        bits |= ParticipantBits.AVG_LIFE
    # ... etc pour tous les bits
    return bits

def _compute_match_bits(self, options: SyncOptions) -> int:
    """Calcule le bitmask pour un match (données globales)."""
    bits = 0
    if options.with_highlight_events:
        bits |= MatchBits.EVENTS
    if options.with_assets:
        bits |= MatchBits.ASSETS
    if options.with_aliases:
        bits |= MatchBits.ALIASES
    return bits
```

#### 5.0.3 Insertion des participants avec bitmask

```python
# Dans _insert_shared_participants()
def _insert_shared_participants(
    self,
    shared_conn: duckdb.DuckDBPyConnection,
    participants: list[dict],
    match_id: str,
) -> int:
    """Insère les participants avec leur bitmask calculé."""
    inserted = 0
    for p in participants:
        # Calculer le bitmask pour ce participant
        backfill_bits = self._compute_participant_bits(p)
        
        shared_conn.execute(
            """INSERT OR REPLACE INTO match_participants (
                match_id, xuid, gamertag, team_id, outcome, rank, score,
                kills, deaths, assists, kda,
                shots_fired, shots_hit, accuracy,
                damage_dealt, damage_taken,
                avg_life_seconds, time_played_seconds,
                headshot_kills, max_killing_spree,
                grenade_kills, melee_kills, power_weapon_kills,
                personal_score,
                team_mmr, enemy_mmr,
                kills_expected, kills_stddev,
                deaths_expected, deaths_stddev,
                assists_expected, assists_stddev,
                backfill_bits  -- NOUVEAU
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                match_id, p["xuid"], p["gamertag"], ...
                backfill_bits,  # Valeur calculée
            ),
        )
        inserted += 1
    return inserted
```

#### 5.0.4 Mise à jour des bits après backfill skill

```python
# Dans le backfill (après appel API skill)
def _update_participant_skill_bits(
    self,
    shared_conn: duckdb.DuckDBPyConnection,
    match_id: str,
    xuid: str,
    skill_data: dict,
) -> None:
    """Met à jour les bits skill après backfill."""
    bits_to_set = 0
    if skill_data.get("team_mmr") is not None:
        bits_to_set |= ParticipantBits.TEAM_MMR
    if skill_data.get("enemy_mmr") is not None:
        bits_to_set |= ParticipantBits.ENEMY_MMR
    if skill_data.get("kills_expected") is not None:
        bits_to_set |= ParticipantBits.KILLS_EXP
    # ... etc
    
    shared_conn.execute(
        """UPDATE match_participants 
           SET backfill_bits = COALESCE(backfill_bits, 0) | ?
           WHERE match_id = ? AND xuid = ?""",
        (bits_to_set, match_id, xuid),
    )
```

#### 5.0.5 Gestion des medals par joueur

```python
# Après insertion des médailles pour un joueur
def _mark_medals_loaded(
    self,
    shared_conn: duckdb.DuckDBPyConnection,
    match_id: str,
    xuid: str,
) -> None:
    """Marque le bit MEDALS pour ce participant."""
    shared_conn.execute(
        """UPDATE match_participants 
           SET backfill_bits = COALESCE(backfill_bits, 0) | ?
           WHERE match_id = ? AND xuid = ?""",
        (ParticipantBits.MEDALS, match_id, xuid),
    )
```

#### 5.0.6 Rétrocompatibilité pendant la transition

```python
# Pendant la période de migration, maintenir les deux systèmes
def _update_backfill_tracking(
    self,
    shared_conn: duckdb.DuckDBPyConnection,
    match_id: str,
    xuid: str | None,
    participant_bits: int,
    match_bits: int,
) -> None:
    """Met à jour le tracking backfill (nouveau + legacy)."""
    # Nouveau système : match_participants.backfill_bits
    if xuid and participant_bits:
        shared_conn.execute(
            """UPDATE match_participants 
               SET backfill_bits = COALESCE(backfill_bits, 0) | ?
               WHERE match_id = ? AND xuid = ?""",
            (participant_bits, match_id, xuid),
        )
    
    # Nouveau système : match_registry.backfill_completed (pour données match)
    if match_bits:
        shared_conn.execute(
            """UPDATE match_registry 
               SET backfill_completed = COALESCE(backfill_completed, 0) | ?
               WHERE match_id = ?""",
            (match_bits, match_id),
        )
    
    # LEGACY : Continuer à écrire dans l'ancien format (sera supprimé en v6)
    # Ceci permet aux anciennes versions de backfill/sync de fonctionner
    legacy_bits = self._convert_to_legacy_bits(participant_bits, match_bits)
    if legacy_bits:
        shared_conn.execute(
            """UPDATE match_registry
               SET backfill_completed = COALESCE(backfill_completed, 0) | ?
               WHERE match_id = ?""",
            (legacy_bits, match_id),
        )


def _convert_to_legacy_bits(self, participant_bits: int, match_bits: int) -> int:
    """Convertit les nouveaux bits vers l'ancien format BACKFILL_FLAGS (legacy).

    Utilisé pendant la période de transition pour maintenir la compatibilité
    avec les anciens scripts qui lisent match_registry.backfill_completed.

    Correspondance :
        ParticipantBits → BACKFILL_FLAGS
        TEAM_MMR | ENEMY_MMR | KILLS_EXP | DEATHS_EXP | ASSISTS_EXP → "skill" (4)
        MEDALS → "medals" (1)
        ACCURACY → "accuracy" (32)
        SHOTS → "shots" (64)
        MatchBits → BACKFILL_FLAGS
        EVENTS → "events" (2)
        ASSETS → "assets" (256)
        ALIASES → "aliases" (16384)
    """
    from src.data.sync.migrations import BACKFILL_FLAGS

    legacy = 0
    # Participant bits → legacy match_registry bits
    skill_bits = ParticipantBits.TEAM_MMR | ParticipantBits.ENEMY_MMR | ParticipantBits.KILLS_EXP
    if participant_bits & skill_bits:
        legacy |= BACKFILL_FLAGS.get("skill", 0)
    if participant_bits & ParticipantBits.MEDALS:
        legacy |= BACKFILL_FLAGS.get("medals", 0)
    if participant_bits & ParticipantBits.ACCURACY:
        legacy |= BACKFILL_FLAGS.get("accuracy", 0)
    if participant_bits & ParticipantBits.SHOTS:
        legacy |= BACKFILL_FLAGS.get("shots", 0)
    # Match bits → legacy
    if match_bits & MatchBits.EVENTS:
        legacy |= BACKFILL_FLAGS.get("events", 0)
    if match_bits & MatchBits.ASSETS:
        legacy |= BACKFILL_FLAGS.get("assets", 0)
    if match_bits & MatchBits.ALIASES:
        legacy |= BACKFILL_FLAGS.get("aliases", 0)
    return legacy
```

### 5.1 Schéma DuckDB

| Fichier | Modification |
|---------|--------------|
| `src/data/sync/engine.py` | Migration ALTER TABLE |
| `src/data/sync/migrations.py` | Fonction `ensure_match_participants_backfill_bits()` |
| `src/data/sync/batch_insert.py` | Ajouter `backfill_bits` dans l'INSERT participants |
| `docs/SHARED_MATCHES_SCHEMA.md` | Documentation schéma mis à jour |

Migration à ajouter dans `migrations.py` :

```python
def ensure_match_participants_backfill_bits(
    shared_conn: duckdb.DuckDBPyConnection,
) -> None:
    """Ajoute backfill_bits à match_participants si absent."""
    _add_column_if_missing(
        shared_conn, "match_participants", "backfill_bits", "INTEGER DEFAULT 0"
    )
    # Index pour la détection rapide des participants avec données manquantes
    try:
        shared_conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_mp_backfill "
            "ON match_participants(xuid, backfill_bits)"
        )
    except Exception as e:
        logger.debug(f"Index idx_mp_backfill déjà existant ou erreur: {e}")
```

### 5.2 Constantes et SyncScope

| Fichier | Modification |
|---------|--------------|
| `src/data/sync/scope.py` | Nouveaux champs granulaires |
| `src/data/sync/constants.py` (nouveau) | Définition des bits |

**Nouveaux champs à ajouter dans `SyncScope`** (et dans `_ALL_DATA_FIELDS`, `_FORCE_MAP`, `_REQUESTED_TYPE_MAP`) :

```python
# Types de données granulaires (ajouts)
# Note : enemy_mmr EXISTE DÉJÀ dans SyncScope (legacy). Pour éviter le conflit,
# on garde enemy_mmr pour la compatibilité et on ajoute team_mmr comme nouveau champ.
# Le groupe --mmr active les deux.
team_mmr: bool = False
# enemy_mmr conservé tel quel (déjà présent, trackait enemy_mmr + team_mmr ensemble)
kills_expected: bool = False
deaths_expected: bool = False
assists_expected: bool = False
avg_life: bool = False
damage: bool = False
grenade_kills: bool = False
melee_kills: bool = False
power_weapon_kills: bool = False
headshot_kills: bool = False
max_spree: bool = False
kda_recalc: bool = False
time_played: bool = False

# Groupes (alias dans scope ou dans CLI seulement)
mmr: bool = False              # = team_mmr + enemy_mmr (enemy_mmr déjà existant)
expected: bool = False         # = kills_expected + deaths_expected + assists_expected
combat: bool = False           # = accuracy + shots + damage
kills_detail: bool = False     # = grenade_kills + melee_kills + power_weapon_kills + headshot_kills
core_stats: bool = False       # = combat + avg_life + kills_detail + kda + time_played

# Flags force correspondants
force_team_mmr: bool = False
force_enemy_mmr_new: bool = False  # à ne pas confondre avec force_enemy_mmr legacy
force_kills_expected: bool = False
force_deaths_expected: bool = False
force_assists_expected: bool = False
force_avg_life: bool = False
force_damage: bool = False
force_grenade_kills: bool = False
force_melee_kills: bool = False
force_power_weapon_kills: bool = False
force_headshot_kills: bool = False
force_max_spree: bool = False
force_kda_recalc: bool = False
force_time_played: bool = False
force_mmr: bool = False
force_expected: bool = False
force_combat: bool = False
force_kills_detail: bool = False
force_core_stats: bool = False
```

**Résolution des groupes dans `resolve()`** (à ajouter) :

> **Ordre critique** : les groupes imbriqués (`core_stats` → `combat` → champs individuels)
> doivent être résolus **de l'extérieur vers l'intérieur** dans `resolve()`, pas en plusieurs
> passes. L'ordre dans la méthode doit donc être : `core_stats`, puis `combat`, `kills_detail`,
> puis `mmr`, `expected`, enfin `_FORCE_MAP`.

```python
def resolve(self) -> None:
    # ... code existant (all_data) ...
    # ── 1. Groupes de haut niveau (résoudre en premier) ──
    if self.core_stats:
        self.combat = True
        self.avg_life = True
        self.kills_detail = True
        self.kda_recalc = True
        self.time_played = True
    # ── 2. Groupes intermédiaires ──
    if self.combat:
        self.accuracy = True
        self.shots = True
        self.damage = True
    if self.kills_detail:
        self.grenade_kills = True
        self.melee_kills = True
        self.power_weapon_kills = True
        self.headshot_kills = True
    # ── 3. Groupes skill ──
    if self.mmr:
        self.team_mmr = True
        self.enemy_mmr = True  # enemy_mmr existant
    if self.expected:
        self.kills_expected = True
        self.deaths_expected = True
        self.assists_expected = True
    # ── 4. _FORCE_MAP (code existant, doit rester en dernier) ──
    for force_field, data_field in _FORCE_MAP.items():
        if getattr(self, force_field, False) and not getattr(self, data_field, False):
            setattr(self, data_field, True)
```

```python
# src/data/sync/constants.py
class ParticipantBits:
    """Bitmask pour match_participants.backfill_bits."""
    TEAM_MMR = 1 << 0       # 1
    ENEMY_MMR = 1 << 1      # 2
    KILLS_EXP = 1 << 2      # 4
    DEATHS_EXP = 1 << 3     # 8
    ASSISTS_EXP = 1 << 4    # 16
    ACCURACY = 1 << 5       # 32
    SHOTS = 1 << 6          # 64
    DAMAGE = 1 << 7         # 128
    AVG_LIFE = 1 << 8       # 256
    MEDALS = 1 << 9         # 512
    GRENADE_KILLS = 1 << 10 # 1024
    MELEE_KILLS = 1 << 11   # 2048
    POWER_WEAPON = 1 << 12  # 4096
    PERSONAL_SCORE = 1 << 13# 8192
    HEADSHOT_KILLS = 1 << 14# 16384
    MAX_SPREE = 1 << 15     # 32768
    KDA = 1 << 16           # 65536
    TIME_PLAYED = 1 << 17   # 131072
    KILLER_VICTIM = 1 << 18 # 262144
    
    # Groupes
    MMR = TEAM_MMR | ENEMY_MMR
    EXPECTED = KILLS_EXP | DEATHS_EXP | ASSISTS_EXP
    SKILL = MMR | EXPECTED
    COMBAT = ACCURACY | SHOTS | DAMAGE
    KILLS_DETAIL = GRENADE_KILLS | MELEE_KILLS | POWER_WEAPON | HEADSHOT_KILLS
    CORE_STATS = ACCURACY | SHOTS | DAMAGE | AVG_LIFE | GRENADE_KILLS | MELEE_KILLS | \
                 POWER_WEAPON | PERSONAL_SCORE | HEADSHOT_KILLS | MAX_SPREE | KDA | TIME_PLAYED
```

### 5.3 Détection

| Fichier | Modification |
|---------|--------------|
| `scripts/backfill/detection.py` | Nouvelle logique par bit |

```python
def _find_matches_missing_participant_data(
    shared_conn: Any,
    xuid: str,
    *,
    bits_required: int,  # Masque des bits requis
    force: bool = False,
    max_matches: int | None = None,
) -> list[str]:
    """Trouve les matchs où ce joueur a des bits manquants."""
    if force:
        # Ignorer le bitmask, retourner tous les matchs
        condition = "1=1"
    else:
        # IMPORTANT : COALESCE obligatoire — backfill_bits est NULL pour les
        # enregistrements antérieurs à la migration (ajout de la colonne).
        # Sans COALESCE, NULL & bits_required = NULL != bits_required → toujours True,
        # ce qui forcerait un backfill pour TOUS les anciens matchs.
        condition = f"(COALESCE(mp.backfill_bits, 0) & {bits_required}) != {bits_required}"

    query = f"""
        SELECT mp.match_id
        FROM match_participants mp
        JOIN match_registry mr ON mr.match_id = mp.match_id
        WHERE mp.xuid = ? AND {condition}
        ORDER BY mr.start_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        return [row[0] for row in shared_conn.execute(query, [xuid]).fetchall()]
    except Exception as e:
        logger.error(f"Erreur détection bits manquants (xuid={xuid}): {e}")
        return []


def _find_matches_missing_match_data(
    shared_conn: Any,
    *,
    bits_required: int,
    force: bool = False,
    max_matches: int | None = None,
) -> list[str]:
    """Trouve les matchs où des données au niveau match sont manquantes.

    Utilisé pour les données par match (events, assets, aliases).
    Pas de filtre xuid — les données match sont globales.
    """
    if force:
        condition = "1=1"
    else:
        condition = f"(COALESCE(mr.backfill_completed, 0) & {bits_required}) != {bits_required}"

    query = f"""
        SELECT mr.match_id
        FROM match_registry mr
        WHERE {condition}
        ORDER BY mr.start_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        return [row[0] for row in shared_conn.execute(query).fetchall()]
    except Exception as e:
        logger.error(f"Erreur détection bits match manquants: {e}")
        return []
```

### 5.4 CLI

| Fichier | Modification |
|---------|--------------|
| `scripts/backfill/cli.py` | Nouveaux arguments |

**Nouveaux arguments à ajouter dans `create_argument_parser()`** :

```python
# ── Granulaire MMR ──
# Note : --enemy-mmr EXISTE DÉJÀ en legacy (= backfill enemy_mmr colonne).
# --team-mmr est le nouveau champ granulaire. --mmr active les deux.
parser.add_argument("--team-mmr", action="store_true", help="Backfill team_mmr si NULL")
# --enemy-mmr déjà présent en legacy, conservé tel quel
parser.add_argument("--mmr", action="store_true", help="= --team-mmr + --enemy-mmr")
parser.add_argument("--force-team-mmr", action="store_true")
# --force-enemy-mmr déjà présent en legacy
parser.add_argument("--force-mmr", action="store_true")

# ── Granulaire Expected ──
parser.add_argument("--kills-expected", action="store_true")
parser.add_argument("--deaths-expected", action="store_true")
parser.add_argument("--assists-expected", action="store_true")
parser.add_argument("--expected", action="store_true", help="= kills/deaths/assists-expected")
parser.add_argument("--force-kills-expected", action="store_true")
parser.add_argument("--force-deaths-expected", action="store_true")
parser.add_argument("--force-assists-expected", action="store_true")
parser.add_argument("--force-expected", action="store_true")

# ── Groupe Skill (rétrocompatibilité) ──
# --skill et --force-skill déjà présents

# ── Combat granulaire ──
parser.add_argument("--damage", action="store_true", help="Backfill damage_dealt/damage_taken si NULL")
parser.add_argument("--avg-life", action="store_true", help="Backfill avg_life_seconds si NULL")
parser.add_argument("--combat", action="store_true", help="= --accuracy + --shots + --damage")
parser.add_argument("--force-damage", action="store_true")
parser.add_argument("--force-avg-life", action="store_true")
parser.add_argument("--force-combat", action="store_true")

# ── Kills détail ──
parser.add_argument("--grenade-kills", action="store_true")
parser.add_argument("--melee-kills", action="store_true")
parser.add_argument("--power-weapon-kills", action="store_true")
parser.add_argument("--headshot-kills", action="store_true")
parser.add_argument("--max-spree", action="store_true")
parser.add_argument("--kills-detail", action="store_true", help="= tous les kills-*")
parser.add_argument("--force-grenade-kills", action="store_true")
parser.add_argument("--force-melee-kills", action="store_true")
parser.add_argument("--force-power-weapon-kills", action="store_true")
parser.add_argument("--force-headshot-kills", action="store_true")
parser.add_argument("--force-max-spree", action="store_true")
parser.add_argument("--force-kills-detail", action="store_true")

# ── Divers ──
parser.add_argument("--kda-recalc", action="store_true", help="Recalcule kda si NULL")
parser.add_argument("--time-played", action="store_true")
parser.add_argument("--force-kda-recalc", action="store_true")
parser.add_argument("--force-time-played", action="store_true")

# ── Groupe Core Stats ──
parser.add_argument("--core-stats", action="store_true",
    help="= accuracy, shots, damage, avg-life, kills-detail, kda, time-played")
parser.add_argument("--force-core-stats", action="store_true")
```

### 5.5 Orchestrateur

| Fichier | Modification |
|---------|--------------|
| `scripts/backfill/orchestrator.py` | Logique de backfill par bit |
| `scripts/backfill/core.py` | Mettre à jour `backfill_bits` après insertion |
| `scripts/backfill/strategies.py` | Mettre à jour bit KILLER_VICTIM après traitement |

**Modifications dans `orchestrator.py`** :

```python
async def _process_match_backfill(
    match_id: str,
    xuid: str,
    scope: SyncScope,
    shared_conn: Any,
    player_conn: Any,
    api: Any,
) -> None:
    """Traite le backfill d'un match avec mise à jour des bits."""
    participant_bits_added = 0
    match_bits_added = 0

    # ── Skill / MMR ──
    if scope.skill or scope.mmr or scope.team_mmr or scope.expected or scope.kills_expected:
        # Calculer le masque des bits skill requis
        skill_bits_needed = _compute_skill_bits_needed(scope)
        if scope.force_skill or (COALESCE(current_bits) & skill_bits_needed) != skill_bits_needed:
            skill_data = await api.get_skill_stats(match_id)
            if skill_data:
                _write_skill_to_shared(shared_conn, match_id, skill_data)
                participant_bits_added |= _compute_participant_bits(skill_data.get(xuid, {}))

    # ── Stats de combat (get_match_stats) ──
    combat_bits_needed = _compute_combat_bits_needed(scope)
    if combat_bits_needed and (COALESCE(current_bits) & combat_bits_needed) != combat_bits_needed:
        # get_match_stats retourne déjà les données → pas de re-appel si déjà fait
        # pour ce match lors de ce run
        ...

    # ── Mise à jour finale des bits ──
    if participant_bits_added:
        shared_conn.execute(
            """UPDATE match_participants
               SET backfill_bits = COALESCE(backfill_bits, 0) | ?
               WHERE match_id = ? AND xuid = ?""",
            (participant_bits_added, match_id, xuid),
        )
    if match_bits_added:
        shared_conn.execute(
            """UPDATE match_registry
               SET backfill_completed = COALESCE(backfill_completed, 0) | ?
               WHERE match_id = ?""",
            (match_bits_added, match_id),
        )
```

**Helpers à définir dans `orchestrator.py`** (référencés dans le code ci-dessus) :

```python
def _compute_skill_bits_needed(scope: SyncScope) -> int:
    """Calcule le masque des bits skill requis selon le scope."""
    from src.data.sync.constants import ParticipantBits
    bits = 0
    if scope.skill or scope.mmr or scope.team_mmr:
        bits |= ParticipantBits.TEAM_MMR | ParticipantBits.ENEMY_MMR
    if scope.skill or scope.expected or scope.kills_expected:
        bits |= ParticipantBits.KILLS_EXP
    if scope.skill or scope.expected or scope.deaths_expected:
        bits |= ParticipantBits.DEATHS_EXP
    if scope.skill or scope.expected or scope.assists_expected:
        bits |= ParticipantBits.ASSISTS_EXP
    return bits

def _compute_combat_bits_needed(scope: SyncScope) -> int:
    """Calcule le masque des bits combat requis selon le scope."""
    from src.data.sync.constants import ParticipantBits
    bits = 0
    if scope.accuracy or scope.combat or scope.core_stats:
        bits |= ParticipantBits.ACCURACY
    if scope.shots or scope.combat or scope.core_stats:
        bits |= ParticipantBits.SHOTS
    if scope.damage or scope.combat or scope.core_stats:
        bits |= ParticipantBits.DAMAGE
    if scope.avg_life or scope.core_stats:
        bits |= ParticipantBits.AVG_LIFE
    if scope.grenade_kills or scope.kills_detail or scope.core_stats:
        bits |= ParticipantBits.GRENADE_KILLS
    if scope.melee_kills or scope.kills_detail or scope.core_stats:
        bits |= ParticipantBits.MELEE_KILLS
    if scope.power_weapon_kills or scope.kills_detail or scope.core_stats:
        bits |= ParticipantBits.POWER_WEAPON
    if scope.headshot_kills or scope.kills_detail or scope.core_stats:
        bits |= ParticipantBits.HEADSHOT_KILLS
    if scope.max_spree:
        bits |= ParticipantBits.MAX_SPREE
    if scope.kda_recalc or scope.core_stats:
        bits |= ParticipantBits.KDA
    if scope.time_played or scope.core_stats:
        bits |= ParticipantBits.TIME_PLAYED
    return bits
```

> **Note importante** : `get_match_stats` retourne les données pour **tous les joueurs** du match.
> Lors d'un appel pour un joueur, optimiser en mettant à jour `backfill_bits` pour **tous les
> participants** présents dans la réponse, pas uniquement pour le joueur courant.

---

## 6. Plan d'Implémentation

### Phase 1 : Infrastructure (1-2h)

- [ ] Créer `src/data/sync/constants.py` avec `ParticipantBits` et `MatchBits`
- [ ] Ajouter migration pour `backfill_bits` dans `match_participants`
- [ ] Mettre à jour `SyncScope` avec les nouveaux champs granulaires
- [ ] Documenter le nouveau schéma dans `docs/SHARED_MATCHES_SCHEMA.md`

### Phase 2 : Sync Engine (2-3h)

- [ ] Refactorer `_compute_backfill_mask()` → `_compute_participant_bits()` + `_compute_match_bits()`
- [ ] Modifier `_insert_shared_participants()` pour calculer et insérer `backfill_bits`
- [ ] Ajouter `_update_participant_skill_bits()` pour le backfill skill
- [ ] Ajouter `_mark_medals_loaded()` pour le tracking medals par joueur
- [ ] Implémenter `_update_backfill_tracking()` pour rétrocompatibilité
- [ ] Migrer les appels existants vers le nouveau système
- [ ] Tests unitaires pour les nouvelles fonctions

### Phase 3 : Détection Backfill (1-2h)

- [ ] Créer `_find_matches_missing_participant_data()` 
- [ ] Adapter `find_matches_missing_data()` pour router correctement (bits requis)
- [ ] Tests unitaires de détection

### Phase 4 : CLI (1h)

- [ ] Ajouter les arguments individuels dans `cli.py`
- [ ] Ajouter les arguments `--force-*`
- [ ] Valider la rétrocompatibilité `--skill`

### Phase 5 : Orchestrateur Backfill (2-3h)

- [ ] Refactorer `_process_match_backfill()` pour traiter par bits
- [ ] Optimiser : ne pas appeler l'API si données déjà en base
- [ ] Mettre à jour `backfill_bits` après traitement réussi
- [ ] Tests d'intégration

### Phase 6 : Migration données existantes (1h)

- [ ] Script pour calculer `backfill_bits` depuis colonnes existantes
- [ ] Script pour le bit medals (vérifier medals_earned)
- [ ] Script pour le bit killer_victim (vérifier killer_victim_pairs)
- [ ] Valider sur données réelles
- [ ] Documenter la migration

### Phase 7 : Nettoyage et dépréciation (30min)

- [ ] Marquer `backfill_status` (player DB) comme déprécié
- [ ] Documenter les flags legacy dans `match_registry.backfill_completed`
- [ ] Mettre à jour la documentation
- [ ] Mettre à jour `.ai/` files

---

## 7. Migration des Données Existantes

### 7.1 Calcul initial des bits

```sql
-- Calculer backfill_bits depuis les colonnes existantes
UPDATE match_participants SET backfill_bits = 
    CASE WHEN team_mmr IS NOT NULL THEN 1 ELSE 0 END |
    CASE WHEN enemy_mmr IS NOT NULL THEN 2 ELSE 0 END |
    CASE WHEN kills_expected IS NOT NULL THEN 4 ELSE 0 END |
    CASE WHEN deaths_expected IS NOT NULL THEN 8 ELSE 0 END |
    CASE WHEN assists_expected IS NOT NULL THEN 16 ELSE 0 END |
    CASE WHEN accuracy IS NOT NULL THEN 32 ELSE 0 END |
    CASE WHEN shots_fired IS NOT NULL AND shots_hit IS NOT NULL THEN 64 ELSE 0 END |
    CASE WHEN damage_dealt IS NOT NULL AND damage_taken IS NOT NULL THEN 128 ELSE 0 END |
    CASE WHEN avg_life_seconds IS NOT NULL THEN 256 ELSE 0 END |
    -- Bit 9 (medals) géré séparément
    CASE WHEN grenade_kills IS NOT NULL THEN 1024 ELSE 0 END |
    CASE WHEN melee_kills IS NOT NULL THEN 2048 ELSE 0 END |
    CASE WHEN power_weapon_kills IS NOT NULL THEN 4096 ELSE 0 END |
    CASE WHEN personal_score IS NOT NULL THEN 8192 ELSE 0 END |
    CASE WHEN headshot_kills IS NOT NULL THEN 16384 ELSE 0 END |
    CASE WHEN max_killing_spree IS NOT NULL THEN 32768 ELSE 0 END |
    CASE WHEN kda IS NOT NULL THEN 65536 ELSE 0 END |
    CASE WHEN time_played_seconds IS NOT NULL THEN 131072 ELSE 0 END;
    -- Bit 18 (killer_victim) géré séparément
```

### 7.2 Bit medals (cas spécial)

```sql
-- Medals : vérifier existence dans medals_earned
UPDATE match_participants mp SET backfill_bits = backfill_bits | 512
WHERE EXISTS (
    SELECT 1 FROM medals_earned me 
    WHERE me.match_id = mp.match_id AND me.xuid = mp.xuid
);
```

### 7.3 Bit killer_victim (cas spécial)

```sql
-- Killer/victim : vérifier existence dans killer_victim_pairs
UPDATE match_participants mp SET backfill_bits = backfill_bits | 262144
WHERE EXISTS (
    SELECT 1 FROM killer_victim_pairs kvp 
    WHERE kvp.match_id = mp.match_id 
      AND (kvp.killer_xuid = mp.xuid OR kvp.victim_xuid = mp.xuid)
);
```

### 7.4 Gestion de la table `backfill_status` existante

**Situation actuelle** : La table `backfill_status` existe dans chaque player DB (`stats.duckdb`) avec des flags `attempted_*`.

| Champ | Usage actuel |
|-------|--------------|
| `attempted_medals` | Booléen, médailles tentées |
| `attempted_events` | Booléen, events tentés |
| `attempted_skill` | Booléen, skill tenté |
| `attempted_personal_scores` | Booléen, personal scores tentés |
| `attempted_performance_scores` | Booléen, performance tentée |
| `attempted_aliases` | Booléen, aliases tentés |
| `attempted_accuracy` | Booléen, accuracy tentée |
| `attempted_shots` | Booléen, shots tentés |
| `attempted_enemy_mmr` | Booléen, enemy MMR tenté |
| `attempted_assets` | Booléen, assets tentés |
| `attempted_participants` | Booléen, participants tentés |
| `attempted_participants_scores` | Booléen, scores part. tentés |
| `attempted_participants_kda` | Booléen, KDA part. tenté |
| `attempted_participants_shots` | Booléen, shots part. tentés |
| `attempted_participants_damage` | Booléen, damage part. tenté |

**Décision** : **DÉPRÉCIER** cette table au profit du nouveau système.

**Raisons** :
1. Tracking "attempted" vs "réussi" — le nouveau bitmask track les données réellement présentes
2. Localisation dans player DB — pas accessible multi-joueurs
3. Redondance — le bitmask dans `match_participants` suffit
4. Booléens — pas de granularité fine

**Migration** :
1. Phase 1-5 : Nouveau système fonctionne en parallèle
2. Phase 6 : Marquer `backfill_status` comme déprécié
3. Phase 7 (future) : Supprimer après 2 releases stables

**Rétrocompatibilité** :
```python
# Pendant la période de transition, écrire dans les deux
if self._legacy_backfill_status_exists():
    self._update_legacy_backfill_status(match_id, attempted_skill=True)
# Toujours écrire dans le nouveau
self._update_participant_backfill_bits(match_id, xuid, ParticipantBits.SKILL)
```

---

## 8. Estimation des Gains

### Avant (scénario actuel)

```
Madina97294 avec 36 matchs sans kills_expected :
├── Appel --skill : 36 appels API
├── Chaque appel traite 8 colonnes × 8 joueurs
├── Total UPDATEs : 36 × 8 × 8 = 2304
└── Temps : ~3-5 minutes
```

### Après (nouveau système)

```
Madina97294 avec 36 matchs sans kills_expected :
├── Appel --kills-expected : 36 appels API (même chose)
├── MAIS : détection précise, pas de faux positifs
├── UPDATEs ciblés : 36 × 2 (kills_expected + stddev)
└── Temps : ~1-2 minutes
```

**Gain principal** : éviter les appels API inutiles quand seule une colonne manque.

---

## 9. Risques et Mitigations

| Risque | Mitigation |
|--------|------------|
| Régression sur --skill | Garder le comportement identique (groupe de bits) |
| Migration lente | Exécuter en batch, indexer avant |
| Complexité CLI | Documenter clairement, exemples dans --help |
| Incohérence bitmask | Recalculer depuis colonnes si doute (section 7.1) |
| `backfill_bits IS NULL` | Toujours utiliser `COALESCE(backfill_bits, 0)` dans les requêtes |
| Rollback migration échouée | Avant migration : `CREATE TABLE match_participants_backup AS SELECT * FROM match_participants` — restaurer avec `INSERT OR REPLACE INTO match_participants SELECT * FROM match_participants_backup` |
| Conflit connexions DuckDB | shared_matches.duckdb n'accepte qu'une connexion write simultanée — s'assurer que le sync et le backfill ne tournent pas en parallèle |
| Confusion KILLER_VICTIM bits | Bit 18 (participants) ≠ bit 3 match_registry — voir section 3.3 pour clarification |

---

## 10. Fichiers à Modifier

```
src/data/sync/
├── constants.py        # (NOUVEAU) ParticipantBits, MatchBits
├── scope.py            # Champs granulaires SyncScope (30+ nouveaux champs)
├── engine.py           # _compute_participant_bits(), _update_backfill_tracking()
├── migrations.py       # ensure_match_participants_backfill_bits()
└── batch_insert.py     # Ajouter backfill_bits dans INSERT match_participants

scripts/backfill/
├── cli.py              # Nouveaux arguments (--kills-expected, --mmr, etc.)
├── detection.py        # _find_matches_missing_participant_data() + _find_matches_missing_match_data()
├── orchestrator.py     # Traitement par bit, mise à jour backfill_bits multi-participants
├── core.py             # Mettre à jour backfill_bits après insert medals/events
├── strategies.py       # Mettre à jour bit KILLER_VICTIM après traitement
└── migrate_bits.py     # (NOUVEAU) Script de migration des données existantes

docs/
├── SHARED_MATCHES_SCHEMA.md  # Schéma mis à jour (backfill_bits dans match_participants)
├── COMMANDS.md               # Nouvelles options CLI
└── SYNC_GUIDE.md             # Guide sync mis à jour

.ai/
├── thought_log.md      # Décision architecturale
└── data_lineage.md     # Mise à jour flux
```

### Structure de `migrate_bits.py` (NOUVEAU)

```python
"""Migration : calcul initial de backfill_bits depuis les colonnes existantes.

Usage :
    python scripts/backfill/migrate_bits.py
    python scripts/backfill/migrate_bits.py --dry-run
    python scripts/backfill/migrate_bits.py --batch-size 5000
"""
import argparse
import duckdb
from pathlib import Path

SHARED_PATH = Path("data/warehouse/shared_matches.duckdb")

def migrate_backfill_bits(conn: duckdb.DuckDBPyConnection, dry_run: bool = False) -> None:
    """Calcule et remplit backfill_bits depuis les colonnes existantes."""
    # 1. Ajouter la colonne si absente
    # 2. Calculer bits depuis colonnes NULL/NOT NULL (voir SQL section 7.1)
    # 3. Appliquer bit MEDALS via sous-requête medals_earned (section 7.2)
    # 4. Appliquer bit KILLER_VICTIM via sous-requête killer_victim_pairs (section 7.3)
    # 5. Rapport final : distribution des bits

def main() -> None:
    parser = argparse.ArgumentParser(description="Migration backfill_bits")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--batch-size", type=int, default=10_000)
    args = parser.parse_args()
    conn = duckdb.connect(str(SHARED_PATH))
    migrate_backfill_bits(conn, dry_run=args.dry_run)

if __name__ == "__main__":
    main()
```

---

## 11. Validation

- [ ] Tests unitaires pour chaque bit
- [ ] Tests `_compute_participant_bits()` avec différentes combinaisons
- [ ] Test rétrocompatibilité `--skill`
- [ ] Test migration sur backup de données
- [ ] Test sync nouveau match → bits corrects
- [ ] Test sync match existant → bits mis à jour
- [ ] Test backfill skill → bits mis à jour
- [ ] Benchmark performance avant/après
- [ ] Documentation utilisateur

---

**Estimation totale** : 8-12 heures de travail

**Phases critiques** :
1. Phase 1 (Infrastructure) — fondation, doit être parfaite
2. Phase 2 (Sync Engine) — changement majeur, risque de régression
3. Phase 6 (Migration) — données existantes, ne pas perdre l'historique

**Prêt à implémenter ?**
