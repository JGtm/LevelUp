# Tracking des Événements d'Objectifs via SPNKr API

> **Date** : 2026-02-18  
> **Contexte** : Question utilisateur sur la capacité de tracker les captures de drapeaux, bases, etc.  
> **Statut** : Recherche terminée ✅

---

## 🎯 Question Posée

> "Est-ce qu'avec grunt api ou spnkr api on peut déterminer à quel moment notre équipe ou l'équipe adverse ont capturé le drapeau ? Et sur les bases quand elles sont capturées ? Même type de question pour le mode assaut et oddball"

---

## 📊 Réponse Courte

**OUI** pour CTF, Strongholds et Oddball ✅  
**NON** pour Assault ❌  
**PARTIEL** pour les timestamps ⚠️

### Points Clés

| Capacité | Disponibilité | Notes |
|----------|--------------|-------|
| **Quel joueur a capturé?** | ✅ **OUI** | Via XUID + gamertag |
| **Quelle équipe?** | ✅ **OUI** | Via team_id dans match_participants |
| **Combien de fois?** | ✅ **OUI** | Via award_count |
| **Timestamp précis?** | ❌ **NON** | PersonalScores sont agrégés sans time_ms |
| **Mode Assault?** | ❌ **NON** | Aucun événement spécifique documenté |

---

## 🔍 Architecture des Données

### Source API

Les événements d'objectifs sont extraits depuis :

```
API Halo Waypoint
└── Match Stats JSON
    └── Players[]
        └── PlayerTeamStats[]
            └── Stats
                └── CoreStats
                    └── PersonalScores[] ← ICI
                        ├── PersonalScoreNameId (int)
                        ├── Count (int)
                        └── TotalPersonalScoreAwarded (int)
```

### Implémentation Actuelle

**Module** : `src/data/sync/transformers.py`

```python
def extract_personal_score_awards(match_json: dict, xuid: str) -> list[dict]:
    """Extrait les PersonalScores depuis le JSON du match.
    
    Retourne une liste de dictionnaires avec:
    - name_id: PersonalScoreNameId (int)
    - count: Nombre d'occurrences
    - total_score: Points gagnés
    """
```

**Catégorisation** :

```python
def categorize_personal_score(name_id: int) -> str:
    """Classe un PersonalScore en catégorie.
    
    Returns:
        "kill", "assist", "objective", "vehicle", "penalty", "other"
    """
```

**Stockage** : Table `personal_score_awards` (par joueur, dans stats.duckdb)

| Colonne | Type | Description |
|---------|------|-------------|
| `match_id` | VARCHAR | FK → match_stats |
| `xuid` | VARCHAR | Joueur |
| `award_name` | VARCHAR | "Flag Captured", etc. |
| `award_category` | VARCHAR | "objective", "kill", etc. |
| `award_count` | INTEGER | Nombre de fois obtenu |
| `award_score` | INTEGER | Points gagnés |

---

## 📋 Événements Disponibles par Mode

### 🏴 CTF (Capture de Drapeau) — 6 événements

| Événement | PersonalScoreNameId | Points | Catégorie | Description |
|-----------|---------------------|--------|-----------|-------------|
| **FLAG_CAPTURED** | 601966503 | 300 | objective | ✅ **Capture confirmée** |
| FLAG_STOLEN | 3002710045 | 25 | objective | Drapeau ennemi volé |
| FLAG_RETURNED | 22113181 | 25 | objective | Drapeau allié ramené |
| FLAG_TAKEN | 2387185397 | 10 | objective | Drapeau pris (initial) |
| FLAG_CAPTURE_ASSIST | 555570945 | 100 | assist | Assistance à la capture |
| RUNNER_STOPPED | 316828380 | 25 | objective | Porteur ennemi éliminé |

**Exemple de requête** :

```sql
-- Qui a capturé des drapeaux dans ce match ?
SELECT 
    mp.gamertag,
    mp.team_id,
    psa.award_count AS captures,
    psa.award_score AS points
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_name = 'Flag Captured'
ORDER BY psa.award_count DESC;
```

### 🏰 Strongholds (Bastions) — 4 événements

| Événement | PersonalScoreNameId | Points | Catégorie | Description |
|-----------|---------------------|--------|-----------|-------------|
| **ZONE_CAPTURED_100** | 757037588 | 100 | objective | ✅ **Base capturée à 100%** |
| ZONE_CAPTURED_75 | 4026987576 | 75 | objective | Base capturée à 75% |
| ZONE_CAPTURED_50 | 3507884073 | 50 | objective | Base capturée à 50% |
| ZONE_SECURED | 709346128 | 25 | objective | Zone sécurisée |

**Notes** :
- Les pourcentages représentent le niveau de contrôle nécessaire pour déclencher l'événement
- Une capture complète génère probablement plusieurs événements (50% → 75% → 100%)

**Exemple de requête** :

```sql
-- Captures de bases par équipe
SELECT 
    mp.team_id,
    COUNT(DISTINCT mp.xuid) AS joueurs_ayant_capturé,
    SUM(psa.award_count) AS total_captures
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_name IN ('Zone Captured 100%', 'Zone Captured 75%', 'Zone Captured 50%')
GROUP BY mp.team_id;
```

### 🏈 Oddball (Balle) — 3 événements

| Événement | PersonalScoreNameId | Points | Catégorie | Description |
|-----------|---------------------|--------|-----------|-------------|
| **BALL_CONTROL** | 454168309 | 50 | objective | ✅ **Contrôle de la balle** |
| BALL_TAKEN | 204144695 | 10 | objective | Balle ramassée |
| CARRIER_STOPPED | 746397417 | 25 | objective | Porteur ennemi éliminé |

**Notes** :
- `BALL_CONTROL` est probablement donné toutes les X secondes de possession
- `award_count` pour BALL_CONTROL pourrait indiquer la durée relative de possession

**Exemple de requête** :

```sql
-- Temps de possession par joueur (estimé via BALL_CONTROL count)
SELECT 
    mp.gamertag,
    mp.team_id,
    psa.award_count AS ball_control_events,
    psa.award_score AS points
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_name = 'Ball Control'
ORDER BY psa.award_count DESC;
```

### ⚔️ Assault (Assaut) — ❌ Aucun événement

**Statut** : L'enum `PersonalScoreNameId` ne contient **AUCUN** événement spécifique au mode Assault.

**Hypothèses** :
1. Le mode Assault pourrait utiliser des événements génériques (kills, zone control)
2. Les données pourraient être dans un autre endpoint non exploité
3. Le mode pourrait ne pas avoir d'événements de score personnalisés

**Workaround possible** :
- Analyser `match_registry.team_0_score` et `match_registry.team_1_score` pour déduire les rounds gagnés
- Utiliser les kills/deaths généraux pour inférer les succès d'attaque/défense

---

## ⏱️ Problème des Timestamps

### Limitation Actuelle

**PersonalScores n'ont PAS de timestamp individuel** ⚠️

Les données fournies par l'API sont :
```json
{
  "PersonalScores": [
    {
      "PersonalScoreNameId": 601966503,
      "Count": 2,
      "TotalPersonalScoreAwarded": 600
    }
  ]
}
```

**Manquant** : Pas de champ `Time`, `Timestamp` ou `EventTime`

### Alternative : highlight_events

La table `highlight_events` (dans shared_matches.duckdb) contient des timestamps précis :

```sql
CREATE TABLE highlight_events (
    id INTEGER PRIMARY KEY,
    match_id VARCHAR NOT NULL,
    event_type VARCHAR,     -- "kill", "death", "medal"
    time_ms INTEGER,        -- ← TIMESTAMP PRÉCIS EN MS
    killer_xuid VARCHAR,
    victim_xuid VARCHAR,
    ...
);
```

**MAIS** : `highlight_events` ne contient que :
- Kills
- Deaths
- Medals

**DONC** : Les captures de drapeaux/bases ne sont **PAS** dans highlight_events.

### Solution Partielle : Corréler avec Kills

Si un joueur capture le drapeau, on pourrait :
1. Trouver ses kills/deaths dans `highlight_events` autour de la capture
2. Estimer une fenêtre temporelle approximative

**Exemple** :

```sql
-- Fenêtre temporelle probable d'une capture de drapeau
WITH flag_capturers AS (
  SELECT mp.xuid, mp.gamertag
  FROM personal_score_awards psa
  JOIN match_participants mp USING (match_id, xuid)
  WHERE psa.match_id = 'xxx'
    AND psa.award_name = 'Flag Captured'
),
related_events AS (
  SELECT 
    he.time_ms,
    he.event_type,
    he.killer_xuid,
    he.victim_xuid
  FROM highlight_events he
  WHERE he.match_id = 'xxx'
    AND (he.killer_xuid IN (SELECT xuid FROM flag_capturers)
         OR he.victim_xuid IN (SELECT xuid FROM flag_capturers))
)
SELECT * FROM related_events
ORDER BY time_ms;
```

---

## 🔗 Relation Équipe

### Comment déterminer quelle équipe a capturé ?

**Via `match_participants`** :

```sql
SELECT 
    psa.award_name,
    psa.award_count,
    mp.team_id,              -- ← 0 ou 1
    mp.gamertag,
    mp.outcome               -- 'win', 'loss', 'tie'
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_category = 'objective';
```

**Agrégation par équipe** :

```sql
-- Captures de drapeaux par équipe
SELECT 
    mp.team_id,
    SUM(psa.award_count) AS total_flags_captured
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_name = 'Flag Captured'
GROUP BY mp.team_id;
```

---

## 📈 Cas d'Usage Possibles

### 1. Timeline des captures (approximative)

**Approche** :
1. Lister tous les joueurs ayant capturé (`FLAG_CAPTURED`, `ZONE_CAPTURED_100`, etc.)
2. Pour chaque joueur, extraire ses kills/deaths depuis `highlight_events`
3. Interpoler les moments probables de capture entre deux événements

**Précision** : ±30 secondes (estimation)

### 2. Heatmap des contributions aux objectifs

**Par joueur** :
```python
# Pseudo-code
df = get_personal_score_awards(match_id, category='objective')
heatmap = df.groupby(['gamertag', 'award_name']).agg({'award_count': 'sum'})
plot_heatmap(heatmap)
```

**Résultat** :
```
                   Flag_Captured  Zone_Secured  Ball_Control
Player1                     3           5             0
Player2                     0           8            12
Player3                     1           2             3
```

### 3. Comparaison équipes

**Objectifs complétés par équipe** :
```sql
SELECT 
    mp.team_id,
    SUM(CASE WHEN psa.award_name = 'Flag Captured' THEN psa.award_count ELSE 0 END) AS flags,
    SUM(CASE WHEN psa.award_name LIKE 'Zone Captured%' THEN psa.award_count ELSE 0 END) AS zones,
    SUM(CASE WHEN psa.award_name = 'Ball Control' THEN psa.award_count ELSE 0 END) AS ball_controls
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
GROUP BY mp.team_id;
```

### 4. Joueur MVP Objectifs

**Top contributeur** :
```sql
SELECT 
    mp.gamertag,
    mp.team_id,
    SUM(psa.award_score) AS total_objective_score,
    SUM(psa.award_count) AS total_actions
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_category = 'objective'
GROUP BY mp.xuid, mp.gamertag, mp.team_id
ORDER BY total_objective_score DESC
LIMIT 1;
```

---

## 🚀 Implémentation Proposée

### Phase 1 : Extraction des Données (✅ Déjà fait)

- [x] Fonction `extract_personal_score_awards()` existe
- [x] Fonction `categorize_personal_score()` existe
- [x] Table `personal_score_awards` existe (dans stats.duckdb)

### Phase 2 : Service Layer (À faire)

**Nouveau fichier** : `src/data/services/objective_events_service.py`

```python
"""Service pour extraire et analyser les événements d'objectifs."""

import polars as pl
from src.data.repositories.duckdb_repo import DuckDBRepository

def get_objective_events_by_team(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne les événements d'objectifs agrégés par équipe.
    
    Returns:
        DataFrame avec colonnes:
        - team_id
        - award_name
        - total_count
        - total_score
    """
    query = """
    SELECT 
        mp.team_id,
        psa.award_name,
        SUM(psa.award_count) AS total_count,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp USING (match_id, xuid)
    WHERE psa.match_id = ?
      AND psa.award_category = 'objective'
    GROUP BY mp.team_id, psa.award_name
    ORDER BY mp.team_id, total_score DESC
    """
    return repo.query_df(query, [match_id])

def get_flag_captures_timeline(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Timeline approximative des captures de drapeaux.
    
    Corrèle les captures avec les kills/deaths pour estimer les moments.
    
    Returns:
        DataFrame avec colonnes:
        - gamertag
        - team_id
        - flag_captures
        - approx_time_ms (liste des timestamps estimés)
    """
    # TODO: Implémenter corrélation avec highlight_events
    pass

def get_base_captures_by_player(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Contributions individuelles aux captures de bases (Strongholds).
    
    Returns:
        DataFrame avec colonnes:
        - gamertag
        - team_id
        - zone_50_count
        - zone_75_count
        - zone_100_count
        - total_score
    """
    query = """
    SELECT 
        mp.gamertag,
        mp.team_id,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 50%' THEN psa.award_count ELSE 0 END) AS zone_50_count,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 75%' THEN psa.award_count ELSE 0 END) AS zone_75_count,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 100%' THEN psa.award_count ELSE 0 END) AS zone_100_count,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp USING (match_id, xuid)
    WHERE psa.match_id = ?
      AND psa.award_name LIKE 'Zone Captured%'
    GROUP BY mp.xuid, mp.gamertag, mp.team_id
    ORDER BY total_score DESC
    """
    return repo.query_df(query, [match_id])
```

### Phase 3 : Visualisations (À faire)

**Nouveau fichier** : `src/visualization/objective_timeline.py`

```python
"""Graphiques pour les événements d'objectifs."""

import plotly.graph_objects as go
import polars as pl

def plot_flag_captures_by_team(df: pl.DataFrame) -> go.Figure:
    """Barres empilées : Captures de drapeaux par équipe."""
    fig = go.Figure()
    
    for team in [0, 1]:
        team_df = df.filter(pl.col("team_id") == team)
        fig.add_trace(go.Bar(
            name=f"Équipe {team}",
            x=team_df["award_name"],
            y=team_df["total_count"],
        ))
    
    fig.update_layout(
        title="Captures de Drapeaux par Équipe",
        xaxis_title="Type d'Événement",
        yaxis_title="Nombre",
        barmode='group',
    )
    return fig

def plot_base_capture_contributions(df: pl.DataFrame) -> go.Figure:
    """Barres empilées : Contributions aux captures de bases (Strongholds)."""
    # Top 10 joueurs
    top_players = df.head(10)
    
    fig = go.Figure()
    
    fig.add_trace(go.Bar(
        name="50%",
        x=top_players["gamertag"],
        y=top_players["zone_50_count"],
    ))
    fig.add_trace(go.Bar(
        name="75%",
        x=top_players["gamertag"],
        y=top_players["zone_75_count"],
    ))
    fig.add_trace(go.Bar(
        name="100%",
        x=top_players["gamertag"],
        y=top_players["zone_100_count"],
    ))
    
    fig.update_layout(
        title="Contributions aux Captures de Bases",
        xaxis_title="Joueur",
        yaxis_title="Nombre de Captures",
        barmode='stack',
    )
    return fig
```

### Phase 4 : UI Streamlit (À faire)

**Nouveau fichier** : `src/ui/pages/objective_events.py`

```python
"""Page d'analyse des événements d'objectifs."""

import streamlit as st
from src.data.services.objective_events_service import (
    get_objective_events_by_team,
    get_base_captures_by_player,
)
from src.visualization.objective_timeline import (
    plot_flag_captures_by_team,
    plot_base_capture_contributions,
)

def render_objective_events_page(repo, match_id: str):
    st.title("⚔️ Événements d'Objectifs")
    
    # Section CTF
    st.header("🏴 Capture de Drapeau")
    flag_events = get_objective_events_by_team(repo, match_id)
    flag_events_filtered = flag_events.filter(
        pl.col("award_name").str.contains("Flag")
    )
    if not flag_events_filtered.is_empty():
        st.plotly_chart(plot_flag_captures_by_team(flag_events_filtered))
    else:
        st.info("Aucun événement CTF dans ce match.")
    
    # Section Strongholds
    st.header("🏰 Bastions")
    base_captures = get_base_captures_by_player(repo, match_id)
    if not base_captures.is_empty():
        st.plotly_chart(plot_base_capture_contributions(base_captures))
    else:
        st.info("Aucun événement Strongholds dans ce match.")
    
    # Section Oddball
    st.header("🏈 Oddball")
    # TODO: Implémenter
    st.info("Visualisation Oddball en cours de développement.")
```

---

## 📝 Résumé Exécutif

### ✅ Ce qu'on PEUT faire

1. **Identifier les captures** : Oui, via `personal_score_awards`
2. **Compter les captures** : Oui, via `award_count`
3. **Identifier l'équipe** : Oui, via `match_participants.team_id`
4. **Calculer les points** : Oui, via `award_score`
5. **Lister les contributeurs** : Oui, avec jointures

### ❌ Ce qu'on NE PEUT PAS faire

1. **Timestamp précis des captures** : Non, PersonalScores n'ont pas de `time_ms`
2. **Mode Assault** : Non, aucun événement spécifique documenté
3. **Timeline exacte** : Non, seulement estimation via corrélation avec kills

### ⚠️ Ce qu'on PEUT ESTIMER

1. **Timeline approximative** : Corrélation avec `highlight_events` (précision ±30s)
2. **Ordre des captures** : Si on suppose que l'ordre API reflète l'ordre temporel (non garanti)

---

## 🎓 Recommandations

### Court Terme (Sprint actuel)

1. ✅ **Documenter les capacités actuelles** (ce document)
2. 🔧 **Implémenter les services d'extraction** (Phase 2)
3. 📊 **Créer les visualisations basiques** (Phase 3)
4. 🖥️ **Ajouter une page UI** (Phase 4)

### Moyen Terme (Sprints futurs)

5. 🔍 **Investiguer Grunt API** : Vérifier si Grunt expose des endpoints avec timestamps
6. 🧪 **Tester corrélation avec highlight_events** : Valider la précision de l'estimation temporelle
7. 📡 **Explorer endpoints non documentés** : Chercher des données Assault
8. 📊 **Enrichir les visualisations** : Timeline approximative, heatmaps, etc.

### Long Terme (R&D)

9. 🔬 **Reverse-engineer film chunks** : Les chunks type 2 contiennent peut-être des positions/événements objectifs
10. 🤖 **Contribuer à SPNKr** : Proposer des PRs pour exposer plus de données
11. 📈 **ML pour estimation temporelle** : Prédire les moments de capture via patterns de kills

---

## 📚 Références

- **SPNKr Refdata** : `spnkr.models.refdata.PersonalScoreNameId`
- **Implémentation LevelUp** : `src/data/sync/transformers.py`
- **Schéma DB** : `docs/SHARED_MATCHES_SCHEMA.md`
- **API Source** : Blog Den Delimarsky - [Extracting Match Stats From Halo Infinite Film Files](https://den.dev/blog/extracting-stats-film-files-halo-infinite)

---

*Document généré le 2026-02-18 suite à la question utilisateur sur le tracking des événements d'objectifs*
