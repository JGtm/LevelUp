# Visualisation Timeline de Domination du Match

> **Date** : 2026-02-18  
> **Contexte** : Demande de frise chronologique pour visualiser qui domine le match  
> **Status** : ✅ Module créé, prêt pour intégration UI

---

## 🎯 Demande Utilisateur

> "Bon alors ok je veux une visualisation dans l'onglet match et l'onglet dernier match. Comme une frise chronologique pour savoir qui domine le match. Si c'est en mode assassin on peut traquer les frags. Je ne sais pas encore sous quelle forme on traduira ça graphiquement. Pour les captures de drapeau on traque qui a le plus de securisation/capture (en neutre victoen 5 point, en normal, 3). Et pour les bases on fait comme t'as dit. On peut éventuellement envisager d'ajouter cette frise ou representation sur un axe et afficher les frag en baton au dessus et les deces en dessous. Je te laisse etre créatif"

### Spécifications

1. **Onglets** : Match + Dernier Match
2. **Modes** :
   - **Slayer** : Traquer les frags (kills)
   - **CTF** : Traquer captures (5 points neutre, 3 normal)
   - **Strongholds** : Contrôle bases (clustering temporel)
3. **Visualisation** : Frise chronologique avec option overlay kills/deaths

---

## 💻 Module Implémenté

### Fichier

**`src/visualization/match_dominance_timeline.py`** (21 KB, 685 lignes)

### Fonctions Principales (5 types de timeline)

#### 1. Timeline Slayer

```python
create_slayer_dominance_timeline(repo, match_id, show_kills_deaths=True)
```

**Visualisation** : Area chart avec kills cumulés par équipe

```
Kills Cumulés
    ^
 50 |     ████████████  (Équipe 0 - Bleu)
 40 |    ████████  (Équipe 1 - Rouge)
 30 |
 20 |
 10 |
  0 |___________________________> Temps (s)
    0    60   120  180  240
```

**Caractéristiques** :
- Ligne continue avec fill (area chart)
- Couleurs : Équipe 0 (#0A84FF bleu), Équipe 1 (#FF453A rouge)
- Hover : Nombre de kills + timestamp
- Source : `highlight_events` (time_ms précis)

---

#### 2. Timeline CTF

```python
create_ctf_dominance_timeline(repo, match_id)
```

**Visualisation** : Line + markers avec captures progressives

```
Captures
    ^
  5 |          ★      ★  (Équipe 0)
  4 |     ★      
  3 |  ★     ★  (Équipe 1)
  2 |
  1 |
  0 |___________________________> Temps (s)
    0    60   120  180  240
```

**Caractéristiques** :
- Markers en forme d'étoile (★) pour chaque capture
- Ligne reliant les captures
- Source : `estimate_objective_captures_timeline()` (précision ±30s)
- Hover : Numéro de capture + timestamp estimé

---

#### 3. Timeline Strongholds

```python
create_strongholds_dominance_timeline(repo, match_id)
```

**Visualisation** : Step chart (paliers) avec contrôle bases

```
Bases
    ^
  3 |  ▄▄▄▄▄▄▄█████  (Équipe 0)
  2 |  █████▄▄▄
  1 |▄▄
  0 |___________________________> Temps (s)
    0    60   120  180  240
```

**Caractéristiques** :
- Shape='hv' (horizontal-vertical) pour effet paliers
- Max 3 bases (plafond Strongholds)
- Source : `estimate_team_base_control()` (clustering temporel)
- Annotation : Contrôle final estimé
- Hover : ~N bases + timestamp

---

#### 4. Timeline avec Overlay ⭐

```python
create_dominance_timeline_with_overlay(repo, match_id, mode="Slayer")
```

**Visualisation** : 2 sous-graphiques (domination + kills/deaths)

```
┌─────────────────────────────────────┐
│  Domination du Match                │
│   ████████ Équipe 0                 │
│  ████ Équipe 1                      │
│                                     │
├─────────────────────────────────────┤
│  Kills & Deaths                     │
│                                     │
│  5 ║ ║   ║  Kills Équipe 0 (bleu)  │
│  0 ─────────────────────────────    │
│ -5 ║ ║   ║  Deaths Équipe 0 (bleu) │
│    ║ ║   ║  Kills Équipe 1 (rouge) │
│                                     │
└─────────────────────────────────────┘
```

**Caractéristiques** :
- Row heights : [60%, 40%]
- Overlay kills en positif (bâtons vers le haut)
- Overlay deaths en négatif (bâtons vers le bas)
- Agrégation par fenêtres de 30 secondes
- Barmode='relative' (pas stacked)

---

#### 5. Fonction Adaptative ⭐⭐⭐

```python
create_match_dominance_timeline(
    repo, 
    match_id, 
    game_mode=None,  # Auto-détection
    show_kills_overlay=True
)
```

**Auto-détection du mode** :
1. Lit `game_variant_category` depuis `match_registry`
2. Normalise le mode (lowercase)
3. Sélectionne la visualisation appropriée :
   - "ctf" / "flag" → `create_ctf_dominance_timeline()`
   - "stronghold" / "zone" → `create_strongholds_dominance_timeline()`
   - "oddball" → Fallback Slayer (à implémenter)
   - Autre → `create_slayer_dominance_timeline()`

**Option overlay** :
- `show_kills_overlay=True` → Appelle `create_dominance_timeline_with_overlay()`
- `show_kills_overlay=False` → Timeline simple

---

## 🎨 Caractéristiques Visuelles

### Couleurs

| Équipe | Couleur | Hex | Fill Alpha |
|--------|---------|-----|------------|
| Équipe 0 | Bleu | `#0A84FF` | 0.2-0.3 |
| Équipe 1 | Rouge | `#FF453A` | 0.2-0.3 |

### Markers

| Mode | Symbole | Taille |
|------|---------|--------|
| CTF | `star` (★) | 10 |
| Strongholds | `square` (■) | 8 |

### Template

- **Theme** : `plotly_dark` (cohérent avec le reste de l'UI)
- **Height** : 400px (simple), 700px (overlay)
- **Hover mode** : `x unified` (synchronisé sur l'axe X)

---

## 🔧 Intégration UI

### Étape 1 : Import dans match_view.py

```python
from src.visualization.match_dominance_timeline import (
    create_match_dominance_timeline
)
```

### Étape 2 : Ajouter Section Timeline

```python
# Dans render_match_view() après la section performance

st.subheader("Timeline de Domination")

# Toggle pour l'overlay
show_overlay = st.checkbox(
    "Afficher kills/deaths détaillés",
    value=True,
    key=f"timeline_overlay_{match_id}"
)

# Créer la timeline
try:
    fig_timeline = create_match_dominance_timeline(
        repo=repo,
        match_id=match_id,
        game_mode=row.get("game_variant_category"),
        show_kills_overlay=show_overlay,
    )
    
    st.plotly_chart(fig_timeline, use_container_width=True)
    
except Exception as e:
    st.error(f"Erreur lors de la création de la timeline : {e}")
```

### Étape 3 : Répéter pour last_match.py

Même logique dans `render_last_match_page()`.

---

## 📊 Exemples de Résultats

### Match Slayer (20 min)

```
Timeline Simple :
- Équipe 0 : 50 kills → Progression linéaire
- Équipe 1 : 42 kills → Légèrement en retard
- Domination : Équipe 0 dès 5 minutes

Timeline avec Overlay :
- Pic de kills Équipe 0 à 8-10 minutes (rush power weapon)
- Équilibre deaths entre les deux équipes
```

### Match CTF (10 min)

```
Timeline Simple :
- Équipe 0 : 5 captures (victoire 5-3)
- Équipe 1 : 3 captures
- Première capture Équipe 0 à ~02:30
- Captures espacées de 2-3 minutes

Timeline avec Overlay :
- Corrélation entre pics de kills et captures
- Équipe 0 domine en kills pendant les captures
```

### Match Strongholds (15 min)

```
Timeline Simple :
- Équipe 0 : 3 bases contrôlées (estimation)
- Équipe 1 : 2 bases
- Contrôle final : Équipe 0 (3) - Équipe 1 (2)
- Confiance : high

Timeline avec Overlay :
- Pics de kills aux moments de contestation des bases
- Paliers dans le contrôle (step chart)
```

---

## ⚠️ Limitations

### Précision Temporelle

| Mode | Source | Précision | Confiance |
|------|--------|-----------|-----------|
| **Slayer** | highlight_events | Exacte (time_ms) | ✅ 100% |
| **CTF** | Estimation interpolée | ±30 secondes | ⚠️ Variable (high/medium/low) |
| **Strongholds** | Estimation clustering | ±30 secondes | ⚠️ Variable |

### Overlay Kills/Deaths

- **Agrégation 30s** : Perte de granularité fine
- **Barmode relative** : Peut superposer les barres si beaucoup d'activité
- **Solution** : Augmenter la fenêtre (60s) ou utiliser stacked

### Mode Oddball

- **Non implémenté** : Fallback sur Slayer
- **À faire** : Implémenter timeline contrôle de balle

---

## 🚀 Améliorations Futures

### Court Terme

1. **Implémenter Oddball** :
   ```python
   def create_oddball_dominance_timeline(repo, match_id):
       # Timeline du contrôle de balle par équipe
       pass
   ```

2. **Markers de Game Events** :
   - Power weapons spawns
   - Momentum shifts (série de kills)
   - Overtime

3. **Annotations Interactives** :
   - Clic sur un point → Voir détails joueur
   - Hover sur capture → Voir joueurs impliqués

### Moyen Terme

4. **Comparaison Multi-Matchs** :
   - Superposer timelines de plusieurs matchs
   - Pattern recognition

5. **Export** :
   - Bouton télécharger PNG
   - Bouton télécharger données CSV

6. **Personnalisation** :
   - Choix des couleurs
   - Choix de la fenêtre d'agrégation (15s, 30s, 60s)
   - Toggle fill area

---

## 📚 Références

### Code

- **Module** : `src/visualization/match_dominance_timeline.py`
- **Services** : `src/data/services/objective_timeline_service.py`
- **UI** : `src/ui/pages/match_view.py`, `src/ui/pages/last_match.py`

### Documentation

- **Timeline approximative** : `.ai/research/TIMELINE_APPROXIMATIVE_OBJECTIFS.md`
- **Agrégation bases** : `.ai/research/AGREGATION_PAR_EQUIPE.md`
- **Architecture** : `.ai/research/ARCHITECTURE_TIMELINE_DECISION.md`

---

## ✅ Status

- ✅ Module créé (`match_dominance_timeline.py`)
- ✅ 5 types de timeline implémentés
- ✅ Auto-détection du mode de jeu
- ✅ Overlay kills/deaths
- ✅ Couleurs et thème cohérents
- 📝 **Prêt pour intégration dans match_view.py et last_match.py**

---

## 💡 Exemple d'Utilisation

```python
from src.data.repositories.duckdb_repo import DuckDBRepository
from src.visualization.match_dominance_timeline import (
    create_match_dominance_timeline
)

# Setup
repo = DuckDBRepository("data/warehouse/shared_matches.duckdb", xuid=None)
match_id = "abc123-def456"

# Créer la timeline (auto-détection du mode)
fig = create_match_dominance_timeline(
    repo=repo,
    match_id=match_id,
    show_kills_overlay=True,
)

# Afficher dans Streamlit
import streamlit as st
st.plotly_chart(fig, use_container_width=True)
```

---

*Document créé le 2026-02-18 - Visualisation timeline de domination*
