# Résumé : Tracking des Événements d'Objectifs

> **TL;DR** : On peut détecter QUI et QUELLE ÉQUIPE a capturé, mais PAS QUAND exactement.

---

## 🎯 Réponse Rapide

| Mode | Événements Trackés | Timestamp Précis | Équipe Identifiable |
|------|-------------------|------------------|-------------------|
| **CTF** | ✅ 6 types (Flag Captured, Flag Stolen, etc.) | ❌ Non | ✅ Oui |
| **Strongholds** | ✅ 4 types (Zone 50%, 75%, 100%, Secured) | ❌ Non | ✅ Oui |
| **Oddball** | ✅ 3 types (Ball Control, Taken, Carrier Stopped) | ❌ Non | ✅ Oui |
| **Assault** | ❌ Aucun événement | ❌ Non | ❌ Non |

---

## 📊 Données Disponibles

### Ce qu'on SAIT

```python
personal_score_awards = {
    "match_id": "abc123",
    "xuid": "123456789",
    "award_name": "Flag Captured",
    "award_count": 3,           # ← Nombre de captures
    "award_score": 900,          # ← Points gagnés (3 × 300)
}

# Jointure avec match_participants pour avoir l'équipe
match_participants = {
    "match_id": "abc123",
    "xuid": "123456789",
    "team_id": 0,                # ← ÉQUIPE !
    "gamertag": "JohnSpartan",
}
```

### Ce qu'on NE SAIT PAS

- ❌ **Timestamp précis** de chaque capture (pas de `time_ms`)
- ❌ **Ordre chronologique** des captures (si plusieurs fois)
- ❌ **Événements Assault** (non documentés dans l'API)

---

## 🔍 Exemples de Requêtes

### CTF : Qui a capturé des drapeaux ?

```sql
SELECT 
    mp.gamertag,
    mp.team_id,
    psa.award_count AS captures
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_name = 'Flag Captured'
ORDER BY psa.award_count DESC;
```

**Résultat** :
```
gamertag      team_id  captures
JohnSpartan   0        3
MasterChief   0        1
Arbiter       1        2
```

### Strongholds : Captures par équipe

```sql
SELECT 
    mp.team_id,
    COUNT(DISTINCT mp.xuid) AS joueurs,
    SUM(psa.award_count) AS total_captures
FROM personal_score_awards psa
JOIN match_participants mp USING (match_id, xuid)
WHERE psa.match_id = 'xxx'
  AND psa.award_name LIKE 'Zone Captured%'
GROUP BY mp.team_id;
```

**Résultat** :
```
team_id  joueurs  total_captures
0        4        15
1        4        12
```

---

## ⚙️ Implémentation Technique

### Fichiers Existants

| Fichier | Fonction |
|---------|----------|
| `src/data/sync/transformers.py` | ✅ `extract_personal_score_awards()` |
| `src/data/sync/transformers.py` | ✅ `categorize_personal_score()` |
| `src/data/domain/refdata.py` | ✅ `PersonalScoreNameId` enum |
| `src/ui/pages/objective_analysis.py` | ✅ Page UI existante (analyse globale) |

### Fichiers à Créer (Proposition)

| Fichier | Objectif |
|---------|----------|
| `src/data/services/objective_events_service.py` | Service pour extraire les événements par équipe/joueur |
| `src/visualization/objective_timeline.py` | Graphiques CTF/Strongholds/Oddball |
| `src/ui/pages/objective_events_details.py` | Page UI dédiée aux événements par match |

---

## 📈 Visualisations Possibles

### 1. Barres Empilées : Captures par Équipe

```
Équipe 0: ▓▓▓▓▓▓▓▓▓ (15 captures)
Équipe 1: ▓▓▓▓▓▓ (12 captures)
```

### 2. Heatmap : Contributions Individuelles

```
           Flag_Captured  Zone_100  Ball_Control
Player1         3           5          0
Player2         0           8         12
Player3         1           2          3
```

### 3. Timeline Approximative (avec corrélation kills)

```
[00:00] Match Start
[02:30] ~Flag Captured (Équipe 0, JohnSpartan)
[05:15] ~Flag Captured (Équipe 1, Arbiter)
[07:45] ~Flag Captured (Équipe 0, JohnSpartan)
[10:00] Match End
```

---

## 🚀 Plan d'Action Proposé

### Phase 1 : Service Layer (2-3h)

- [ ] Créer `objective_events_service.py`
- [ ] Fonction `get_objective_events_by_team(match_id)`
- [ ] Fonction `get_flag_captures_by_player(match_id)`
- [ ] Fonction `get_base_captures_by_player(match_id)`
- [ ] Tests unitaires

### Phase 2 : Visualisations (3-4h)

- [ ] Créer `objective_timeline.py`
- [ ] Graphique barres : Captures CTF par équipe
- [ ] Graphique barres empilées : Strongholds par joueur
- [ ] Graphique radar : Profil objectifs du joueur
- [ ] Tests visuels

### Phase 3 : UI Streamlit (2-3h)

- [ ] Créer `objective_events_details.py`
- [ ] Section CTF avec graphiques
- [ ] Section Strongholds avec graphiques
- [ ] Section Oddball avec graphiques
- [ ] Intégration dans le menu principal

### Phase 4 : Estimation Timestamps (Optionnel, 4-5h)

- [ ] Fonction `estimate_capture_times(match_id)`
- [ ] Corrélation avec `highlight_events` (kills/deaths)
- [ ] Graphique timeline approximative
- [ ] Tests de précision

---

## 📝 Références Rapides

### Événements CTF (6)

```python
FLAG_CAPTURED = 601966503      # 300 pts - CAPTURE CONFIRMÉE
FLAG_STOLEN = 3002710045       # 25 pts
FLAG_RETURNED = 22113181       # 25 pts
FLAG_TAKEN = 2387185397        # 10 pts
FLAG_CAPTURE_ASSIST = 555570945 # 100 pts
RUNNER_STOPPED = 316828380     # 25 pts
```

### Événements Strongholds (4)

```python
ZONE_CAPTURED_100 = 757037588  # 100 pts - CAPTURE COMPLÈTE
ZONE_CAPTURED_75 = 4026987576  # 75 pts
ZONE_CAPTURED_50 = 3507884073  # 50 pts
ZONE_SECURED = 709346128       # 25 pts
```

### Événements Oddball (3)

```python
BALL_CONTROL = 454168309       # 50 pts - CONTRÔLE BALLE
BALL_TAKEN = 204144695         # 10 pts
CARRIER_STOPPED = 746397417    # 25 pts
```

---

## ❓ FAQ

**Q: Peut-on savoir exactement quand un drapeau a été capturé ?**  
R: Non, les `PersonalScores` n'ont pas de timestamp. On peut seulement estimer (±30s) en corrélant avec les kills/deaths.

**Q: Pourquoi pas d'événements Assault ?**  
R: L'enum `PersonalScoreNameId` ne les documente pas. Soit ils n'existent pas, soit ils sont dans un autre endpoint.

**Q: Peut-on voir quelle base spécifique a été capturée dans Strongholds ?**  
R: Non, on sait seulement qu'une base a été capturée (sans savoir laquelle : A, B ou C).

**Q: Les données sont-elles fiables ?**  
R: Oui, elles viennent directement de l'API officielle Halo Waypoint (via SPNKr).

---

*Document de référence rapide - Voir `.ai/research/OBJECTIVE_EVENTS_TRACKING.md` pour les détails complets*
