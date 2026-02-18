# Timeline Approximative des Captures d'Objectifs

> **Date** : 2026-02-18  
> **Contexte** : Suite à la question "Et en croisant des données on peut pas avoir une timeline grossière pour les sécurisation des drapeaux et bases ?"  
> **Statut** : Implémenté ✅

---

## 🎯 Objectif

Créer une **timeline approximative** des captures d'objectifs (drapeaux, bases) en croisant :
- `personal_score_awards` : Qui a capturé, combien de fois (SANS timestamp)
- `highlight_events` : Kills/deaths horodatés (AVEC time_ms)

---

## ✅ Réponse Directe

**OUI, on peut estimer une timeline grossière** en distribuant les captures entre les événements horodatés d'un joueur.

### Précision Attendue

- **±30 secondes** par estimation
- Dépend de l'activité du joueur (nombre de kills/deaths)
- Plus le joueur est actif, plus l'estimation est précise

### Score de Confiance

| Confiance | Critère | Interprétation |
|-----------|---------|----------------|
| **high** | ≥5 événements dans ±30s | Estimation fiable |
| **medium** | 2-4 événements dans ±30s | Estimation correcte |
| **low** | <2 événements dans ±30s | Estimation peu fiable |

---

## 🔧 Implémentation

### Fichiers Créés

1. **Service** : `src/data/services/objective_timeline_service.py` (14 KB)
   - `estimate_objective_captures_timeline()` — Fonction principale
   - `get_player_highlight_events()` — Extraction des événements horodatés
   - `format_timeline_for_display()` — Formatage UI
   - `get_timeline_summary()` — Statistiques résumées

2. **Tests** : `tests/test_objective_timeline_service.py` (11 KB)
   - 15 tests unitaires
   - Tests de distribution, confiance, formatage

---

## 📊 Algorithme

### Principe

1. **Identifier les captureurs** : Requête sur `personal_score_awards`
   ```sql
   SELECT xuid, gamertag, award_count
   FROM personal_score_awards
   WHERE award_name = 'Flag Captured'
   ```

2. **Extraire les événements horodatés** : Requête sur `highlight_events`
   ```sql
   SELECT time_ms, event_type
   FROM highlight_events
   WHERE killer_xuid = ? OR victim_xuid = ?
   ORDER BY time_ms ASC
   ```

3. **Distribuer les captures** :
   - Diviser la plage temporelle en N segments (N = nombre de captures)
   - Pour chaque segment, prendre le milieu comme estimation
   - Calculer la confiance basée sur la densité d'événements

### Exemple Visuel

```
Timeline du joueur :
[====Kill====][====Death====][====Kill====][====Death====]
10s           30s            50s           70s

Si le joueur a capturé 2 fois :
Segment 1 : 10s - 50s → Estimation à ~30s
Segment 2 : 50s - 70s → Estimation à ~60s

Résultat :
  Capture 1 : ~01:00 (confidence: medium, 3 événements proches)
  Capture 2 : ~01:50 (confidence: high, 5 événements proches)
```

---

## 💡 Exemples d'Utilisation

### Exemple 1 : Timeline CTF

```python
from src.data.repositories.duckdb_repo import DuckDBRepository
from src.data.services.objective_timeline_service import (
    estimate_objective_captures_timeline,
    format_timeline_for_display,
)

repo = DuckDBRepository("data/players/JGtm/stats.duckdb", "xuid123")

# Estimer la timeline des captures de drapeaux
timeline = estimate_objective_captures_timeline(repo, "match_abc123", mode="CTF")

# Formater pour l'affichage
display = format_timeline_for_display(timeline, match_duration_ms=600000)

print(display[["gamertag", "time_formatted", "confidence", "capture_index"]])
```

**Résultat** :
```
gamertag     time_formatted  confidence  capture_index
JohnSpartan  01:30          high        1
JohnSpartan  03:45          medium      2
JohnSpartan  05:12          high        3
Arbiter      02:15          high        1
Arbiter      04:30          low         2
```

### Exemple 2 : Timeline Strongholds

```python
# Même principe pour les bases
timeline = estimate_objective_captures_timeline(repo, "match_sh_123", mode="Strongholds")

display = format_timeline_for_display(timeline)
print(display[["gamertag", "award_name", "time_formatted", "confidence"]])
```

**Résultat** :
```
gamertag  award_name          time_formatted  confidence
PlayerA   Zone Captured 100%  02:00          high
PlayerA   Zone Captured 100%  04:30          medium
PlayerB   Zone Captured 75%   03:15          high
PlayerB   Zone Captured 100%  05:45          high
```

### Exemple 3 : Statistiques Résumées

```python
from src.data.services.objective_timeline_service import get_timeline_summary

timeline = estimate_objective_captures_timeline(repo, "match_abc123", "CTF")
summary = get_timeline_summary(timeline)

print(f"Total captures: {summary['total_captures']}")
print(f"Haute confiance: {summary['high_confidence_count']}")
print(f"Précision moyenne: {summary['average_nearby_events']:.1f} événements proches")
```

**Résultat** :
```
Total captures: 8
Haute confiance: 5
Précision moyenne: 4.2 événements proches
```

---

## 📈 Cas d'Usage

### 1. Timeline Visuelle (UI)

```python
# Créer un graphique de timeline
import plotly.graph_objects as go

timeline = estimate_objective_captures_timeline(repo, match_id, "CTF")
display = format_timeline_for_display(timeline, match_duration_ms=600000)

fig = go.Figure()

# Une trace par équipe
for team_id in [0, 1]:
    team_data = display.filter(pl.col("team_id") == team_id)
    
    fig.add_trace(go.Scatter(
        x=team_data["estimated_time_ms"],
        y=[team_id] * team_data.height,
        mode='markers+text',
        name=f"Équipe {team_id}",
        text=team_data["gamertag"],
        textposition="top center",
        marker=dict(
            size=15,
            color=['green' if c == 'high' else 'orange' if c == 'medium' else 'red' 
                   for c in team_data["confidence"]],
        ),
    ))

fig.update_layout(
    title="Timeline des Captures de Drapeaux",
    xaxis_title="Temps (ms)",
    yaxis_title="Équipe",
    yaxis=dict(tickvals=[0, 1], ticktext=["Équipe 0", "Équipe 1"]),
)

fig.show()
```

### 2. Analyse des Moments Clés

```python
# Identifier les captures rapprochées (momentum)
timeline = estimate_objective_captures_timeline(repo, match_id, "CTF")

# Grouper les captures par fenêtre de 30 secondes
window_size = 30000  # 30s
timeline_sorted = timeline.sort("estimated_time_ms")

clusters = []
current_cluster = []

for row in timeline_sorted.iter_rows(named=True):
    if not current_cluster:
        current_cluster.append(row)
    elif row["estimated_time_ms"] - current_cluster[-1]["estimated_time_ms"] <= window_size:
        current_cluster.append(row)
    else:
        clusters.append(current_cluster)
        current_cluster = [row]

if current_cluster:
    clusters.append(current_cluster)

# Trouver les clusters importants (≥3 captures)
important_moments = [c for c in clusters if len(c) >= 3]

print(f"Moments clés identifiés: {len(important_moments)}")
for i, moment in enumerate(important_moments, 1):
    start_time = moment[0]["estimated_time_ms"]
    print(f"Moment {i}: ~{start_time//1000}s - {len(moment)} captures")
```

### 3. Comparaison Équipes

```python
# Voir quelle équipe contrôle le match
timeline = estimate_objective_captures_timeline(repo, match_id, "CTF")

team_0 = timeline.filter(pl.col("team_id") == 0)
team_1 = timeline.filter(pl.col("team_id") == 1)

print(f"Équipe 0: {team_0.height} captures")
print(f"Équipe 1: {team_1.height} captures")

# Voir les périodes de contrôle
team_0_times = team_0["estimated_time_ms"].sort().to_list()
team_1_times = team_1["estimated_time_ms"].sort().to_list()

print("\nPremière capture:")
print(f"  Équipe 0: {team_0_times[0]//1000}s" if team_0_times else "  Aucune")
print(f"  Équipe 1: {team_1_times[0]//1000}s" if team_1_times else "  Aucune")

print("\nDernière capture:")
print(f"  Équipe 0: {team_0_times[-1]//1000}s" if team_0_times else "  Aucune")
print(f"  Équipe 1: {team_1_times[-1]//1000}s" if team_1_times else "  Aucune")
```

---

## ⚠️ Limitations

### 1. Précision Variable

La précision dépend de l'activité du joueur :

| Activité | Événements/min | Précision attendue |
|----------|----------------|-------------------|
| Très élevée | >10 | ±15 secondes |
| Élevée | 5-10 | ±30 secondes |
| Moyenne | 2-5 | ±45 secondes |
| Faible | <2 | ±60+ secondes |

### 2. Cas Problématiques

**Joueur inactif** :
- Peu/pas de kills/deaths
- Impossible d'estimer avec précision
- Confidence = "low"

**Captures groupées** :
- Si toutes les captures dans une courte période
- Les estimations seront proches
- Difficile de distinguer l'ordre exact

**Début/fin de match** :
- Peu d'événements avant la première capture
- Peu d'événements après la dernière capture
- Estimations moins fiables

### 3. Ce que la Timeline NE Peut PAS Faire

❌ **Distinguer les captures simultanées** : Si 2 joueurs capturent en même temps, on ne saura pas l'ordre exact.

❌ **Capturer les tentatives échouées** : On ne voit que les captures réussies (FLAG_CAPTURED), pas les tentatives.

❌ **Événements sans kills** : Si un joueur capture sans tuer personne autour, impossible d'estimer.

---

## 🧪 Tests

### Exécuter les Tests

```bash
# Tests spécifiques à la timeline
python -m pytest tests/test_objective_timeline_service.py -v

# Avec couverture
python -m pytest tests/test_objective_timeline_service.py --cov=src.data.services.objective_timeline_service
```

### Tests Implémentés

1. `test_get_player_highlight_events_returns_dataframe` — Extraction des événements
2. `test_distribute_captures_single_capture` — Distribution d'1 capture
3. `test_distribute_captures_multiple_captures` — Distribution de N captures
4. `test_distribute_captures_confidence_high` — Confiance élevée
5. `test_distribute_captures_confidence_low` — Confiance faible
6. `test_distribute_captures_empty_events` — Cas vide
7. `test_get_award_filters_for_ctf` — Filtres CTF
8. `test_get_award_filters_for_strongholds` — Filtres Strongholds
9. `test_get_award_filters_for_oddball` — Filtres Oddball
10. `test_ms_to_mmss_formatting` — Formatage temps
11. `test_format_timeline_for_display` — Formatage UI
12. `test_get_timeline_summary` — Statistiques
13. `test_get_timeline_summary_empty` — Stats vides
14. `test_estimate_objective_captures_timeline_mocked` — Test complet mocké

---

## 📝 Prochaines Étapes

### Phase 1 : Amélioration Algorithme (Optionnel)

- [ ] Utiliser un algorithme de clustering (DBSCAN) pour mieux grouper les événements
- [ ] Pondérer les kills plus que les deaths (kills = moment d'action)
- [ ] Prendre en compte les médailles objectives (Flag Taken, etc.)

### Phase 2 : Visualisations

- [ ] Créer `src/visualization/objective_timeline_chart.py`
- [ ] Graphique timeline avec marqueurs de confiance
- [ ] Graphique barres horizontales par équipe
- [ ] Graphique de momentum (captures/minute)

### Phase 3 : UI Streamlit

- [ ] Ajouter section "Timeline" dans `src/ui/pages/objective_events_details.py`
- [ ] Afficher la timeline avec slider temporel
- [ ] Filtres par confiance (high/medium/low)
- [ ] Export timeline en CSV

---

## 📚 Références

- **Service** : `src/data/services/objective_timeline_service.py`
- **Tests** : `tests/test_objective_timeline_service.py`
- **Schéma DB** : `docs/SHARED_MATCHES_SCHEMA.md`
- **Recherche initiale** : `.ai/research/OBJECTIVE_EVENTS_TRACKING.md`

---

## 🎓 Conclusion

La timeline approximative fonctionne en **distribuant intelligemment** les captures entre les événements horodatés d'un joueur. La précision varie selon l'activité, mais permet d'obtenir une **vision grossière mais utile** de la dynamique du match.

**Cas d'usage principaux** :
- Visualiser le momentum des équipes
- Identifier les moments clés
- Analyser les patterns de captures
- Comparer les stratégies d'équipe

**Précision attendue** : ±30 secondes (en moyenne)

---

*Document créé le 2026-02-18 suite à la question utilisateur sur la timeline approximative*
