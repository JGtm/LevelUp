# Agrégation par Équipe : Contrôle de Bases

> **Date** : 2026-02-18  
> **Contexte** : Challenge de l'agrégation multi-joueurs pour le contrôle de bases  
> **Question** : "Combien de base l'équipe maîtrise sur cette timeline. Car une base peut être capturée par 1 ou 4 joueurs en même temps"

---

## 🎯 Problème

### Challenge Identifié

**Exemple Strongholds** :

```
Équipe 0 :
  02:30 - Joueur A capture Base ? (Zone 100%)
  02:32 - Joueur B capture Base ? (Zone 100%)
  02:35 - Joueur C capture Base ? (Zone 100%)

Questions :
1. Est-ce 3 bases différentes ?
2. Ou A et B ont capturé la même base, et C une autre (= 2 bases) ?
3. Ou les 3 ont capturé la même base (= 1 base) ?
```

### Limitation API

❌ **L'API ne fournit pas l'ID de la base**

```json
// Ce qu'on a :
{
  "PersonalScoreNameId": 757037588,  // Zone Captured 100%
  "Count": 2
}

// Ce qu'on voudrait (mais n'existe pas) :
{
  "PersonalScoreNameId": 757037588,
  "Count": 2,
  "BaseId": "A"  // ❌ N'existe pas
}
```

**Conséquence** : On doit **estimer** le nombre de bases uniques capturées.

---

## 💡 Solution Implémentée

### Approche : Clustering Temporel

**Hypothèse** : Les captures de la même base arrivent dans une fenêtre temporelle courte (≤15 secondes).

#### Algorithme

1. **Grouper les captures par équipe**
2. **Trier par temps estimé**
3. **Détecter les clusters temporels** :
   - Si 2 captures espacées de ≤15s → Même base (probablement)
   - Si 2 captures espacées de >15s → Bases différentes
4. **Compter les clusters** = Nombre de bases uniques

#### Exemple Visuel

```
Timeline Équipe 0 :
[Cap1: 10s] [Cap2: 12s] .... [Cap3: 50s] [Cap4: 52s] .... [Cap5: 90s]
│─────────────────────│       │─────────────────────│       │────────│
   Cluster 1 (Base A)            Cluster 2 (Base B)        Cluster 3 (Base C)

Résultat : 3 bases capturées
```

---

## 🔧 Implémentation

### Fonctions Créées

#### 1. `estimate_team_base_control()`

```python
from src.data.services.objective_timeline_service import estimate_team_base_control

timeline = estimate_objective_captures_timeline(repo, match_id, "Strongholds")
control = estimate_team_base_control(timeline, mode="Strongholds", window_ms=15000)

print(control)
```

**Résultat** :
```
team_id  total_captures  estimated_unique_bases  confidence  method
0        8               3                       high        temporal_clustering
1        6               2                       medium      temporal_clustering
```

**Colonnes** :
- `team_id` : ID de l'équipe (0 ou 1)
- `total_captures` : Nombre de captures individuelles
- `estimated_unique_bases` : Estimation du nombre de bases **uniques**
- `confidence` : "high", "medium", "low"
- `method` : "temporal_clustering"

#### 2. `get_base_control_summary()`

```python
from src.data.services.objective_timeline_service import get_base_control_summary

timeline = estimate_objective_captures_timeline(repo, match_id, "Strongholds")
summary = get_base_control_summary(timeline, "Strongholds")

print(summary)
```

**Résultat** :
```python
{
    "team_0_bases": 3,
    "team_1_bases": 2,
    "team_0_confidence": "high",
    "team_1_confidence": "medium",
    "dominant_team": 0,  # Équipe 0 domine
    "total_unique_bases": 5,  # Approximation
}
```

---

## 📊 Calcul de Confiance

### Score de Confiance

Le score de confiance indique la fiabilité de l'estimation.

| Confiance | Critère | Signification |
|-----------|---------|---------------|
| **high** | Avg cluster size ≥2 + 50%+ multi-captures | Captures bien groupées → Estimation fiable |
| **medium** | Avg cluster size ≥1.5 ou 30%+ multi-captures | Groupement partiel → Estimation correcte |
| **low** | Avg cluster size <1.5 + <30% multi-captures | Captures isolées → Estimation peu fiable |

### Exemples

**Confiance HAUTE** :
```
Équipe avec 6 captures en 3 clusters de 2 :
[10s, 12s] [50s, 52s] [90s, 92s]
→ 3 bases estimées, confiance HIGH
```

**Confiance BASSE** :
```
Équipe avec 5 captures toutes isolées :
[10s] [30s] [50s] [70s] [90s]
→ 5 bases estimées, confiance LOW (probablement surestimé)
```

---

## ⚙️ Paramétrage

### Fenêtre Temporelle (`window_ms`)

**Défaut** : 15 000 ms (15 secondes)

**Justification** :
- En Strongholds, capturer une base prend ~10-20 secondes
- Si 2 joueurs capturent ensemble, ils arrivent à ±5-10s d'écart
- Fenêtre de 15s capture la majorité des cas

**Ajustable** :
```python
# Fenêtre stricte (captures très proches)
control = estimate_team_base_control(timeline, window_ms=10000)

# Fenêtre large (captures plus espacées)
control = estimate_team_base_control(timeline, window_ms=20000)
```

### Limites par Mode

Le nombre de bases est plafonné selon le mode :

| Mode | Limite | Justification |
|------|--------|---------------|
| **Strongholds** | 3 | 3 bases max sur la carte |
| **CTF** | 1 | 1 drapeau capturé à la fois |
| **Oddball** | 1 | 1 balle unique |

**Exemple** :
```python
# Si 5 captures détectées en Strongholds
estimated_bases = min(5, 3)  # → Plafonné à 3
```

---

## 🎨 Utilisation UI

### Exemple Streamlit

```python
import streamlit as st
from src.data.services.objective_timeline_service import (
    estimate_objective_captures_timeline,
    get_base_control_summary,
)

# Calculer timeline
timeline = estimate_objective_captures_timeline(repo, match_id, "Strongholds")

# Résumé du contrôle
summary = get_base_control_summary(timeline, "Strongholds")

# Affichage
col1, col2 = st.columns(2)

with col1:
    st.metric(
        "Équipe 0 - Bases contrôlées",
        summary["team_0_bases"],
        delta="Dominante" if summary["dominant_team"] == 0 else None,
    )
    st.caption(f"Confiance : {summary['team_0_confidence']}")

with col2:
    st.metric(
        "Équipe 1 - Bases contrôlées",
        summary["team_1_bases"],
        delta="Dominante" if summary["dominant_team"] == 1 else None,
    )
    st.caption(f"Confiance : {summary['team_1_confidence']}")

# Détails
with st.expander("Détails de l'estimation"):
    control = estimate_team_base_control(timeline, "Strongholds")
    st.dataframe(control)
    st.caption(
        "Méthode : Clustering temporel (fenêtre 15s). "
        "Les captures proches sont considérées comme la même base."
    )
```

**Résultat visuel** :
```
┌─────────────────────────┐  ┌─────────────────────────┐
│ Équipe 0 - Bases        │  │ Équipe 1 - Bases        │
│                         │  │                         │
│        3 ▲ Dominante    │  │        2                │
│ Confiance : high        │  │ Confiance : medium      │
└─────────────────────────┘  └─────────────────────────┘
```

---

## ⚠️ Limitations

### Ce qui Fonctionne

✅ **Strongholds standard** : 3 bases, captures espacées  
✅ **Multi-captures** : Plusieurs joueurs sur même base  
✅ **Strongholds rapide** : Captures successives  

### Ce qui Fonctionne Moins Bien

❌ **Captures simultanées de 2 bases** : Si 2 bases capturées en <15s par équipes différentes  
❌ **Captures successives rapides** : Base A puis B en <15s = comptées comme 1  
❌ **Mode Custom** : Nombre de bases inconnu  

### Cas Problématiques

**Problème 1** : Rush simultané
```
10s - Joueur A capture Base A
12s - Joueur B capture Base B
→ Détecté comme 1 base (faux)
```

**Problème 2** : Re-captures
```
10s - Base A capturée
50s - Base A perdue et recapturée
→ Détecté comme 2 bases (correct si on compte les événements)
```

### Acceptation des Limitations

✅ **Pour visualisation/tendances** : Suffisant  
✅ **Pour statistiques globales** : Acceptable  
❌ **Pour rejeu précis** : Insuffisant  

---

## 🧪 Tests

### Tests Implémentés (11 nouveaux tests)

1. `test_detect_temporal_clusters_single_cluster` — Cluster unique
2. `test_detect_temporal_clusters_multiple_clusters` — Plusieurs clusters
3. `test_detect_temporal_clusters_empty` — Cas vide
4. `test_calculate_cluster_confidence_high` — Confiance haute
5. `test_calculate_cluster_confidence_low` — Confiance basse
6. `test_estimate_team_base_control_strongholds` — Estimation Strongholds
7. `test_estimate_team_base_control_respects_mode_limit` — Plafond 3 bases
8. `test_estimate_team_base_control_empty` — Timeline vide
9. `test_get_base_control_summary` — Résumé complet
10. `test_get_base_control_summary_tie` — Égalité
11. `test_get_base_control_summary_empty` — Résumé vide

### Exécuter les Tests

```bash
# Tests agrégation uniquement
python -m pytest tests/test_objective_timeline_service.py::test_estimate_team_base_control -v

# Tous les tests timeline (26 tests)
python -m pytest tests/test_objective_timeline_service.py -v
```

---

## 📚 Références

- **Code** : `src/data/services/objective_timeline_service.py` (lignes 405-600)
- **Tests** : `tests/test_objective_timeline_service.py` (lignes 342-540)
- **Architecture** : `.ai/research/ARCHITECTURE_TIMELINE_DECISION.md`

---

## 🎓 Conclusion

### Résumé

✅ **Problème résolu** : Estimation du nombre de bases contrôlées par équipe  
✅ **Approche** : Clustering temporel avec fenêtre 15s  
✅ **Confiance** : Score high/medium/low basé sur groupement  
✅ **Limitations** : Acceptables pour visualisation/stats  

### Utilisation

**Calcul dynamique recommandé** :
- Pas de stockage DB
- Cache Streamlit pour performance
- Flexibilité algorithme

**Pour affichage UI** :
```python
summary = get_base_control_summary(timeline, "Strongholds")
st.metric("Bases contrôlées", summary["team_0_bases"])
```

---

*Document créé le 2026-02-18 - Solution au challenge d'agrégation par équipe*
