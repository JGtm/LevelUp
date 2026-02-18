# Architecture Timeline : Calcul Dynamique vs Stockage BD

> **Date** : 2026-02-18  
> **Contexte** : Question sur l'architecture de la timeline approximative  
> **Question** : "Et si on fait un graphique, on calcule tout ça dynamiquement ? Ou vaut mieux stocker cette timeline en bdd ?"

---

## 🎯 Questions Posées

### Question 1 : Calcul Dynamique vs Stockage

**Calcul dynamique** : Calculer la timeline à chaque fois qu'on en a besoin  
**Stockage BD** : Pré-calculer et stocker les timestamps estimés

### Question 2 : Agrégation par Base

**Challenge** : "Je crois que le challenge c'est pour savoir combien de joueur sécurisent une base pour déterminer combien de base l'équipe maîtrise sur cette timeline. Car une base peut être capturée par 1 ou 4 joueurs en même temps"

**Problème concret** :
```
Joueur A (Équipe 0) capture Base 1 à ~02:30
Joueur B (Équipe 0) capture Base 1 à ~02:32
Joueur C (Équipe 0) capture Base 2 à ~02:35

Question : À 02:35, combien de bases l'équipe contrôle ?
Réponse attendue : 2 bases (Base 1 et Base 2)
Réponse naïve (fausse) : 3 captures
```

---

## 📊 Analyse : Calcul Dynamique vs Stockage

### Option A : Calcul Dynamique ⭐ RECOMMANDÉ

#### Avantages

✅ **Pas de duplication de données**
- Les données sources (highlight_events, personal_score_awards) sont déjà en DB
- Évite la redondance

✅ **Toujours à jour**
- Si on recalcule les scores (backfill), la timeline suit automatiquement
- Pas de risque de désynchronisation

✅ **Pas de migration DB**
- Pas de nouvelle table à créer
- Pas de maintenance de schéma

✅ **Flexibilité de l'algorithme**
- On peut améliorer l'algorithme sans migrer des données
- Facile d'ajouter des variantes (confiance stricte, pondération kills, etc.)

✅ **Coût acceptable**
- Quelques centaines d'événements max par match
- Calcul en mémoire avec Polars (très rapide)
- Cache possible si nécessaire (session Streamlit)

#### Inconvénients

❌ **Recalcul à chaque affichage**
- Mais mitigé par le cache Streamlit (@st.cache_data)

❌ **Pas de requêtes SQL directes**
- Mais on peut créer des vues matérialisées si nécessaire

#### Code Actuel

```python
# Déjà implémenté - calcul à la demande
timeline = estimate_objective_captures_timeline(repo, match_id, "CTF")
```

#### Performance Estimée

| Match Type | Événements | Calcul Time | Acceptable ? |
|------------|-----------|-------------|--------------|
| Court (5min) | ~100-200 | <50ms | ✅ Oui |
| Normal (10min) | ~300-500 | <100ms | ✅ Oui |
| Long (15min) | ~500-800 | <200ms | ✅ Oui |

---

### Option B : Stockage en BD

#### Structure Proposée

```sql
CREATE TABLE objective_timeline_cache (
    id INTEGER PRIMARY KEY,
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    team_id INTEGER NOT NULL,
    award_name VARCHAR NOT NULL,
    capture_index INTEGER NOT NULL,
    estimated_time_ms INTEGER NOT NULL,
    confidence VARCHAR NOT NULL,  -- 'high', 'medium', 'low'
    nearby_events_count INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(match_id, xuid, award_name, capture_index)
);

CREATE INDEX idx_timeline_match ON objective_timeline_cache(match_id);
CREATE INDEX idx_timeline_team ON objective_timeline_cache(match_id, team_id);
```

#### Avantages

✅ **Performance maximale**
- Pas de recalcul
- Requêtes SQL directes

✅ **Enrichissement possible**
- Validation manuelle des timestamps
- Corrections utilisateur
- Notes/commentaires

✅ **Historique**
- Audit trail des changements d'algorithme
- Comparaison versions

#### Inconvénients

❌ **Complexité accrue**
- Nouvelle table à maintenir
- Migrations DB
- Gestion de la synchronisation

❌ **Risque de désynchronisation**
- Si backfill sans recalcul timeline
- Si changement d'algorithme

❌ **Duplication de données**
- Redondance avec les sources

❌ **Effort d'implémentation**
- ~2-3 jours de travail
- Tests de migration
- Gestion des cas limites

#### Quand Considérer le Stockage ?

**Critères pour basculer vers stockage** :
1. **Volume** : >1000 matchs analysés fréquemment
2. **Performance** : Calcul >500ms (ressenti utilisateur)
3. **Enrichissement** : Besoin de validation manuelle
4. **Analytics** : Requêtes SQL complexes sur timeline
5. **API externe** : Exposition de la timeline via REST API

---

## 🏆 Recommandation Finale

### ✅ CALCUL DYNAMIQUE

**Justification** :

1. **Volume actuel acceptable** : Quelques dizaines/centaines de matchs
2. **Performance suffisante** : <200ms pour 99% des cas
3. **Simplicité** : Pas de maintenance DB additionnelle
4. **Flexibilité** : Algorithme facilement améliorable
5. **Pas de désynchronisation** : Toujours cohérent avec les sources

**Avec optimisation** :
```python
# Cache Streamlit pour éviter recalcul
@st.cache_data(ttl=3600)  # 1 heure
def get_cached_timeline(match_id: str, mode: str) -> pl.DataFrame:
    return estimate_objective_captures_timeline(repo, match_id, mode)
```

### Migration Future

**Si besoin évolue** (volume x10, perf insuffisante) :
1. Créer table `objective_timeline_cache`
2. Trigger/job de pré-calcul après sync
3. Fallback dynamique si cache absent
4. Migration progressive

---

## 🔧 Solution au Challenge d'Agrégation

### Problème : Compter les Bases Contrôlées

**Exemple Strongholds** :

```
Timeline individuelle :
  02:30 - Joueur A capture Base A (Zone 100%)
  02:32 - Joueur B capture Base A (Zone 100%)  ← Même base !
  02:35 - Joueur C capture Base B (Zone 100%)

Question : Combien de bases contrôlées à 02:35 ?
Réponse : 2 bases (A et B), pas 3 captures
```

**Problème** : On ne sait pas quelle base précise (A, B ou C) a été capturée.

### Limitation API

❌ **PersonalScores ne contient pas l'ID de la base**

```python
# Ce qu'on a :
{
  "PersonalScoreNameId": 757037588,  # Zone Captured 100%
  "Count": 2,
  "TotalPersonalScoreAwarded": 200
}

# Ce qu'on voudrait :
{
  "PersonalScoreNameId": 757037588,
  "Count": 2,
  "BaseId": "A",  # ❌ N'existe pas
  "TotalPersonalScoreAwarded": 200
}
```

### Solutions Possibles

#### Solution 1 : Fenêtre Temporelle (Approximative) ⭐

**Principe** : Grouper les captures proches temporellement comme étant la même base.

```python
def estimate_base_control_timeline(
    timeline: pl.DataFrame,
    window_ms: int = 10000,  # 10 secondes
) -> pl.DataFrame:
    """Estime le nombre de bases contrôlées en groupant par fenêtre temporelle.
    
    Hypothèse : Les captures de la même base arrivent dans une fenêtre de 10s.
    """
    # Trier par équipe et temps
    sorted_timeline = timeline.sort(["team_id", "estimated_time_ms"])
    
    # Grouper par équipe
    base_control = []
    
    for team_id in sorted_timeline["team_id"].unique():
        team_data = sorted_timeline.filter(pl.col("team_id") == team_id)
        
        # Détecter les clusters temporels (= probablement la même base)
        clusters = _detect_temporal_clusters(
            team_data["estimated_time_ms"].to_list(),
            window_ms
        )
        
        # Compter bases uniques = nombre de clusters
        unique_bases = len(clusters)
        
        base_control.append({
            "team_id": team_id,
            "estimated_bases_controlled": unique_bases,
            "total_captures": team_data.height,
        })
    
    return pl.DataFrame(base_control)
```

**Avantages** :
- ✅ Fonctionne avec les données disponibles
- ✅ Raisonnable pour Strongholds (3 bases max)
- ✅ Donne une estimation utile

**Limitations** :
- ❌ Approximatif (2 bases capturées en <10s = comptées comme 1)
- ❌ Ne fonctionne pas si captures simultanées espacées

#### Solution 2 : Heuristique par Nombre de Joueurs

**Principe** : Si 3+ joueurs capturent en même temps, c'est probablement la même base.

```python
def infer_base_count_from_simultaneity(
    timeline: pl.DataFrame,
    window_ms: int = 15000,  # 15 secondes
) -> dict:
    """Infère le nombre de bases par simultanéité des captures."""
    
    # Pour chaque capture, compter combien de coéquipiers dans ±15s
    timeline = timeline.with_columns([
        pl.col("estimated_time_ms").alias("time")
    ])
    
    clusters = []
    for row in timeline.iter_rows(named=True):
        # Compter les captures proches
        nearby = timeline.filter(
            (pl.col("team_id") == row["team_id"]) &
            (pl.col("estimated_time_ms") >= row["time"] - window_ms) &
            (pl.col("estimated_time_ms") <= row["time"] + window_ms)
        ).height
        
        clusters.append({
            "time": row["time"],
            "nearby_count": nearby,
            "likely_same_base": nearby >= 2,  # 2+ joueurs = même base
        })
    
    # Compter les bases uniques = clusters distincts
    return clusters
```

**Avantages** :
- ✅ Utilise la logique du jeu (multi-capture = même base)
- ✅ Plus précis que fenêtre simple

**Limitations** :
- ❌ Suppose que captures simultanées = même base (pas toujours vrai)
- ❌ Complexe à valider

#### Solution 3 : Borne Supérieure (Conservative) ⭐

**Principe** : Donner le nombre **maximum** de bases contrôlées (borne sup).

```python
def get_max_bases_controlled(
    timeline: pl.DataFrame,
    mode: str,
) -> dict:
    """Calcule la borne supérieure du nombre de bases contrôlées.
    
    Pour Strongholds : Max 3 bases
    Pour CTF : Max 1 drapeau à la fois
    """
    max_bases = {
        "Strongholds": 3,
        "CTF": 1,  # 1 drapeau capturé à la fois
        "Oddball": 1,  # 1 balle
    }
    
    result = {}
    for team_id in timeline["team_id"].unique():
        team_captures = timeline.filter(pl.col("team_id") == team_id).height
        
        # Borne sup = min(captures, max_bases_mode)
        result[team_id] = min(team_captures, max_bases.get(mode, team_captures))
    
    return result
```

**Avantages** :
- ✅ Simple et sûr
- ✅ Borne mathématique garantie
- ✅ Pas de faux positifs

**Limitations** :
- ❌ Peut surestimer (si 5 captures de la même base)
- ❌ Pas d'évolution temporelle

---

## 💡 Recommandation d'Implémentation

### Approche Hybride ⭐

**Combiner plusieurs méthodes** :

```python
def estimate_team_base_control(
    timeline: pl.DataFrame,
    mode: str = "Strongholds",
) -> pl.DataFrame:
    """Estime le contrôle de bases par équipe avec plusieurs méthodes.
    
    Returns:
        DataFrame avec :
        - team_id
        - method_cluster (fenêtre temporelle)
        - method_simultaneity (heuristique joueurs)
        - method_upper_bound (borne sup)
        - recommended_estimate (meilleure estimation)
    """
    # Méthode 1 : Clustering temporel
    cluster_estimate = _cluster_method(timeline)
    
    # Méthode 2 : Simultanéité
    simul_estimate = _simultaneity_method(timeline)
    
    # Méthode 3 : Borne supérieure
    upper_bound = _upper_bound_method(timeline, mode)
    
    # Recommandation : Prendre la médiane des 3
    recommended = {
        team_id: int(statistics.median([
            cluster_estimate[team_id],
            simul_estimate[team_id],
            upper_bound[team_id],
        ]))
        for team_id in timeline["team_id"].unique()
    }
    
    return pl.DataFrame({
        "team_id": list(recommended.keys()),
        "method_cluster": [cluster_estimate[t] for t in recommended.keys()],
        "method_simultaneity": [simul_estimate[t] for t in recommended.keys()],
        "method_upper_bound": [upper_bound[t] for t in recommended.keys()],
        "recommended_estimate": list(recommended.values()),
    })
```

**Affichage UI** :

```python
# Dans Streamlit
st.metric("Bases contrôlées (estimé)", recommended_estimate)
with st.expander("Détails de l'estimation"):
    st.write(f"Méthode clustering : {cluster}")
    st.write(f"Méthode simultanéité : {simul}")
    st.write(f"Borne supérieure : {upper}")
    st.caption("L'estimation recommandée est la médiane des 3 méthodes")
```

---

## 🎓 Conclusion

### Décisions Architecturales

| Aspect | Décision | Justification |
|--------|----------|---------------|
| **Calcul vs Stockage** | ✅ Calcul dynamique | Performance acceptable, simplicité, flexibilité |
| **Cache** | ✅ Cache Streamlit (1h) | Optimisation sans complexité DB |
| **Agrégation bases** | ✅ Approche hybride | Combine 3 méthodes pour meilleure estimation |
| **Affichage** | ✅ Estimation + détails | Transparent sur l'incertitude |

### Implémentations Prioritaires

1. **Court terme** (Maintenant) :
   - Garder calcul dynamique
   - Ajouter cache Streamlit
   - Implémenter méthode clustering temporel

2. **Moyen terme** (Si besoin) :
   - Ajouter méthodes simultanéité + borne sup
   - Affichage multi-méthodes dans UI

3. **Long terme** (Si volume x10) :
   - Considérer table `objective_timeline_cache`
   - Jobs de pré-calcul
   - API REST

### Limitations Acceptées

❌ **Impossible de savoir quelle base précise** (A, B ou C)
- L'API ne fournit pas cette information
- Toute estimation sera approximative

✅ **Mais on peut estimer le NOMBRE de bases**
- Avec confiance raisonnable
- Utile pour visualisation momentum

---

## 📚 Références

- **Service actuel** : `src/data/services/objective_timeline_service.py`
- **Tests** : `tests/test_objective_timeline_service.py`
- **Documentation timeline** : `.ai/research/TIMELINE_APPROXIMATIVE_OBJECTIFS.md`

---

*Document créé le 2026-02-18 suite à la question architecture timeline*
