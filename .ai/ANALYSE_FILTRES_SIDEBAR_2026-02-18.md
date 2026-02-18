# Analyse approfondie : Architecture des filtres sidebar

> **Date** : 2026-02-18  
> **Demande** : Étude approfondie de la fonctionnalité de sélection et mémorisation des filtres dans la sidebar  
> **Contexte** : Bugs récurrents, instabilités, expérience utilisateur dégradée  
> **Objectif** : Analyser les problèmes de conception et proposer une meilleure architecture  
> **Contrainte** : NE PAS toucher au code pour l'instant

---

## Executive Summary

L'architecture actuelle des filtres sidebar souffre de **problèmes de conception fondamentaux** qui causent des bugs récurrents et une instabilité chronique :

1. **❌ Listing dynamique incohérent** : Les options disponibles changent selon la période/session sélectionnée, créant des décochages inattendus
2. **❌ État global non scopé** : Les widgets Streamlit partagent des clés globales entre tous les joueurs
3. **❌ Persistance automatique dangereuse** : Sauvegarde systématique d'états corrompus/incohérents
4. **❌ Cascade complexe et fragile** : Filtres interdépendants (Playlist → Mode → Carte) difficiles à maintenir
5. **❌ Nettoyage incomplet au changement de joueur** : Réutilisation d'états obsolètes

**Recommandation principale** : Repenser l'architecture avec une **source de vérité unique** (les matchs du joueur) et des **options statiques** plutôt que dynamiques.

---

## 1. Diagnostic des problèmes actuels

### 1.1 Problème #1 : Listing dynamique incohérent (Critique ⚠️)

**Tu as identifié le problème clé** dans ta demande : *"On ne liste que les playlists/modes/cartes sur une période/session, et du coup si on change de période on rajoute des éléments potentiellement qui seront décochés"*.

#### Comportement actuel

```python
# filters_render.py, ligne 493+
def _render_cascade_filters(...):
    # Calcul des options disponibles SELON le filtre actuel
    if filter_mode == "Période":
        dropdown_base = dropdown_base.filter(dates)
    else:
        dropdown_base = dropdown_base.filter(session_match_ids)
    
    # Extraction des valeurs uniques DANS le scope filtré
    playlist_values = sorted({
        str(x).strip() 
        for x in dropdown_base["playlist_ui"].drop_nulls().to_list()
    })
```

#### Scénario problématique

1. **État initial** : Mode "Sessions", dernière session sélectionnée
   - 3 matchs joués → Playlists : `["Partie rapide", "Arène classée"]`
   - Utilisateur coche les deux ✅

2. **Changement de période** : Passage en mode "Période", toutes les dates
   - 150 matchs joués → Playlists : `["Partie rapide", "Arène classée", "Assassin classé", "BTB", "Firefight"]`
   - Les nouvelles playlists apparaissent **décochées par défaut** ❌

3. **Retour à la session** : Retour en mode "Sessions"
   - Playlists : retour à `["Partie rapide", "Arène classée"]`
   - Mais `checkbox_filter.py` nettoie les valeurs obsolètes (ligne 167) :
     ```python
     current_selection = current_selection & set(options)
     ```
   - Si l'utilisateur avait coché "BTB" pendant la période étendue, cette sélection est **silencieusement supprimée**

#### Conséquences

- **Désorientation** : L'utilisateur ne comprend pas pourquoi des options apparaissent/disparaissent
- **Perte de sélection** : Les choix sont effacés lors de changements de contexte
- **Effet d'accumulation** : Chaque aller-retour dégrade progressivement l'état
- **Comportement imprévisible** : Impossible de mémoriser une préférence stable

---

### 1.2 Problème #2 : Clés de widgets non scopées par joueur (Critique ⚠️)

#### Architecture actuelle

```python
# checkbox_filter.py, ligne 133+
def render_checkbox_filter(
    *,
    session_key: str,  # Ex: "filter_playlists" (GLOBAL)
    ...
):
    # Génération des clés de widgets
    for opt in options:
        st.checkbox(
            opt,
            value=checked,
            key=f"{session_key}_cb_{opt}",  # Ex: "filter_playlists_cb_Arène_classée"
        )
```

**Problème** : Ces clés sont **globales** (partagées entre tous les joueurs).

#### Scénario de corruption

1. **Joueur A** : Sélectionne `["Partie rapide", "Arène classée"]`
   - `session_state["filter_playlists_cb_Partie_rapide"]` = True
   - `session_state["filter_playlists_cb_Arène_classée"]` = True

2. **Changement vers Joueur B**
   - `streamlit_app.py` supprime `filter_playlists` (les données)
   - Mais **NE SUPPRIME PAS** les clés des widgets (`*_cb_*`)
   - Applique les préférences de B : `filter_playlists = ["Assassin classé"]`

3. **Rendu pour Joueur B**
   - Streamlit réutilise les clés widgets existantes
   - Affiche "Partie rapide" cochée (état résiduel de A) alors que B ne l'a pas sélectionnée
   - Utilisateur clique pour "corriger" → Modifie involontairement la sélection de B

4. **Sauvegarde automatique** (ligne 206 de `filters_render.py`)
   - Enregistre l'état corrompu dans `player_B.json`
   - La préférence de B est **écrasée** par un état mixte A+B

5. **Retour à Joueur A**
   - Les widgets sont maintenant dans un état incohérent (mixe A/B)
   - L'utilisateur voit "encore plus de filtres désélectionnés"

#### Fichier concerné : `streamlit_app.py`

```python
# Ligne ~150 (bloc changement de joueur)
filter_keys_to_clear = [
    "filter_mode",
    "start_date_cal",
    "end_date_cal",
    "picked_session_label",
    "picked_sessions",
    "filter_playlists",   # ✅ Données supprimées
    "filter_modes",       # ✅ Données supprimées
    "filter_maps",        # ✅ Données supprimées
]
# ❌ Manque : filter_playlists_cb_*, filter_playlists_version, etc.
# ❌ Manque : gap_minutes, _latest_session_label, min_matches_maps, etc.
```

---

### 1.3 Problème #3 : Cascade complexe et fragile

#### Architecture en cascade

```
Période/Session
    ↓ filtre
Playlists disponibles
    ↓ filtre
Modes disponibles
    ↓ filtre
Cartes disponibles
```

#### Code (`filters_render.py`, ligne 556+)

```python
# Scope après filtre playlist
scope1 = dropdown_base
if playlists_selected and len(playlists_selected) < len(playlist_values):
    scope1 = scope1.filter(pl.col("playlist_ui").fill_null("").is_in(playlists_selected))

# Scope après filtre mode
scope2 = scope1
if modes_selected and len(modes_selected) < len(mode_values):
    scope2 = scope2.filter(pl.col("mode_ui").fill_null("").is_in(modes_selected))
```

#### Problèmes

1. **Dépendances circulaires** :
   - Les options de Modes dépendent de Playlists
   - Les options de Cartes dépendent de Modes
   - Mais l'utilisateur peut modifier dans n'importe quel ordre

2. **Refiltrages multiples** :
   - Chaque changement déclenche un recalcul de toute la cascade
   - Performance dégradée sur gros datasets

3. **États intermédiaires incohérents** :
   - Entre la sélection et le rerun, l'état peut être invalide
   - Ex : Mode "Assassin" sélectionné, mais toutes les playlists Assassin décochées

4. **Complexité de maintenance** :
   - 3 fonctions différentes pour render (playlists, modes hiérarchiques, cartes)
   - Logique dispersée entre `filters.py`, `filters_render.py`, `checkbox_filter.py`

---

### 1.4 Problème #4 : Persistance automatique dangereuse

#### Code actuel (`filters_render.py`, ligne 204+)

```python
# Sauvegarder automatiquement les filtres si le joueur n'a pas changé
if last_saved_key not in st.session_state or st.session_state[last_saved_key] == player_key:
    try:
        save_filter_preferences(xuid, db_path)  # ⚠️ Sauvegarde TOUT depuis session_state
        st.session_state[last_saved_key] = player_key
    except Exception:
        pass
```

#### Problèmes

1. **Aucune validation** : Sauvegarde même si l'état est corrompu
   - Sélection vide alors que le joueur avait 10 playlists
   - Options invalides (playlists supprimées)
   - États intermédiaires (mid-rerun)

2. **Trop fréquent** : À chaque rendu (plusieurs fois par seconde si reruns)
   - Risque d'écraser des préférences valides
   - Performances I/O dégradées

3. **Pas de confirmation** : L'utilisateur ne sait pas quand ses préférences sont sauvegardées
   - Pas de feedback visuel
   - Pas de bouton "Sauvegarder" explicite

---

### 1.5 Problème #5 : Initialisation par défaut incohérente

#### Code (`filters_render.py`, ligne 227+)

```python
def _apply_default_last_session(...):
    """Applique par défaut la dernière session du joueur."""
    # Calcul des sessions
    base_s = cached_compute_sessions_db(...)
    options = _session_labels_ordered_by_last_match(base_s)
    last_label = options[0] if options else "(toutes)"
    
    # Force le mode "Sessions"
    st.session_state["filter_mode"] = "Sessions"
    st.session_state["picked_session_label"] = last_label
    st.session_state["min_matches_maps"] = 1  # ⚠️ Effet de bord
```

#### Problèmes

1. **Force un comportement** : Toujours "dernière session" même si l'utilisateur préfère "Période"
2. **Effet de bord caché** : Modifie `min_matches_maps` sans relation apparente
3. **Performance** : Calcul sessions coûteux même si non utilisé

---

## 2. Causes racines

### 2.1 Conception : Source de vérité mal définie

**Problème fondamental** : Le système a **deux sources de vérité concurrentes** :

1. **Les matchs du joueur** (dans la DB) → Vérité absolue
2. **Les options affichées** (dans les dropdowns) → Vérité relative (change selon le contexte)

Quand on filtre par période/session, on calcule les options disponibles **à partir du subset filtré**, créant une **boucle de rétroaction** :

```
Filtres période/session
    ↓ limite
Options disponibles
    ↓ utilisateur sélectionne
Filtres playlists/modes/cartes
    ↓ applique
Dataset final
```

**Conséquence** : L'utilisateur ne peut pas exprimer une préférence **stable** (ex: "Je veux toujours Arène classée") car les options changent dynamiquement.

### 2.2 Architecture : État fragmenté entre 4 modules

| Module | Responsabilité | Problème |
|--------|----------------|----------|
| `filters.py` | Logique métier (dates, sessions, amis) | 597 lignes, mélange UI et logique |
| `filters_render.py` | Rendu sidebar + orchestration | 777 lignes, fait trop de choses |
| `filter_state.py` | Persistance JSON | Lit/écrit depuis `session_state` global |
| `checkbox_filter.py` | Composants UI | Génère des clés globales non scopées |

**Aucun module n'a la vision complète** de l'état des filtres. Chaque module modifie `session_state` directement, créant des **couplages cachés**.

### 2.3 Données : Persistance naïve

**Format JSON** (`.streamlit/filter_preferences/player_{gamertag}.json`) :

```json
{
  "filter_mode": "Période",
  "playlists_selected": ["Partie rapide", "Arène classée"],
  "modes_selected": ["Assassin", "Fiesta"],
  "maps_selected": ["Recharge", "Aquarius"]
}
```

**Problème** : Les valeurs sont des **labels UI traduits** (ex: "Partie rapide"), pas des **identifiants stables** (ex: `edfef3ac-9cbe-4fa2-b949-8f29deafd483`).

**Conséquences** :
- Si la traduction change, les préférences sont perdues
- Si une playlist est renommée, les préférences sont invalides
- Impossible de valider si une playlist existe encore

---

## 3. Proposition d'architecture cible

### 3.1 Principes directeurs

1. **✅ Source de vérité unique** : Les matchs du joueur dans la DB
2. **✅ Options statiques** : Toutes les playlists/modes/cartes du jeu (pas seulement celles jouées)
3. **✅ Identifiants stables** : UUIDs au lieu de labels UI
4. **✅ État immutable** : Objet `FilterState` en mémoire, pas de mutations directes
5. **✅ Persistance explicite** : Bouton "Sauvegarder" au lieu de sauvegarde automatique
6. **✅ Scopage par joueur** : Clés `session_state` préfixées par `player_key`

### 3.2 Architecture proposée (3 couches)

```
┌──────────────────────────────────────────────────────────┐
│                    COUCHE PRÉSENTATION                    │
│  filters_sidebar.py (Rendu Streamlit)                    │
│  - Widgets scopés par joueur                              │
│  - Lecture seule de FilterState                           │
│  - Événements → Dispatcher                                │
└──────────────────┬───────────────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────────────┐
│                    COUCHE MÉTIER                          │
│  filter_manager.py (Orchestration)                       │
│  - FilterState (dataclass immutable)                      │
│  - FilterDispatcher (événements)                          │
│  - Validation & calcul options disponibles                │
└──────────────────┬───────────────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────────────┐
│                    COUCHE DONNÉES                         │
│  filter_repository.py (Persistance)                      │
│  - Lecture/écriture JSON par joueur                       │
│  - Validation des préférences                             │
│  filter_options_provider.py (Référentiel)                │
│  - Options disponibles (depuis metadata.duckdb)           │
└──────────────────────────────────────────────────────────┘
```

### 3.3 Dataclasses proposées

```python
from dataclasses import dataclass, field
from datetime import date
from enum import Enum

class FilterMode(Enum):
    PERIOD = "period"
    SESSIONS = "sessions"

@dataclass(frozen=True)  # Immutable
class FilterOptions:
    """Options disponibles pour les filtres (source de vérité)."""
    playlists: dict[str, str]  # {uuid: label_fr}
    modes: dict[str, str]       # {uuid: label_fr}
    maps: dict[str, str]        # {uuid: label_fr}
    
    @classmethod
    def from_metadata(cls, db_path: str) -> "FilterOptions":
        """Charge depuis metadata.duckdb (toutes les options du jeu)."""
        ...

@dataclass(frozen=True)  # Immutable
class FilterState:
    """État complet des filtres (source de vérité en mémoire)."""
    mode: FilterMode
    
    # Période
    start_date: date | None = None
    end_date: date | None = None
    
    # Sessions
    gap_minutes: int = 120
    session_label: str | None = None
    
    # Sélections (UUIDs, pas labels UI)
    playlists_selected: frozenset[str] = field(default_factory=frozenset)
    modes_selected: frozenset[str] = field(default_factory=frozenset)
    maps_selected: frozenset[str] = field(default_factory=frozenset)
    
    def with_playlists(self, playlists: set[str]) -> "FilterState":
        """Retourne une nouvelle instance avec playlists modifiées."""
        return dataclass_replace(self, playlists_selected=frozenset(playlists))
    
    def apply_to_dataframe(self, df: pl.DataFrame, options: FilterOptions) -> pl.DataFrame:
        """Applique les filtres au DataFrame."""
        ...

@dataclass
class FilterPreferences:
    """Préférences persistées (format JSON)."""
    mode: str  # "period" ou "sessions"
    start_date: str | None = None  # ISO format
    end_date: str | None = None
    gap_minutes: int | None = None
    session_label: str | None = None
    playlists: list[str] = field(default_factory=list)  # UUIDs
    modes: list[str] = field(default_factory=list)      # UUIDs
    maps: list[str] = field(default_factory=list)       # UUIDs
    
    def to_filter_state(self, options: FilterOptions) -> FilterState:
        """Convertit en FilterState (validation des UUIDs)."""
        ...
```

### 3.4 Gestion des événements (Dispatcher)

```python
class FilterEvent(Enum):
    MODE_CHANGED = "mode_changed"
    PERIOD_CHANGED = "period_changed"
    SESSION_CHANGED = "session_changed"
    PLAYLISTS_CHANGED = "playlists_changed"
    MODES_CHANGED = "modes_changed"
    MAPS_CHANGED = "maps_changed"
    SAVE_REQUESTED = "save_requested"
    RESET_REQUESTED = "reset_requested"

class FilterDispatcher:
    """Gestionnaire d'événements pour les filtres."""
    
    def dispatch(self, event: FilterEvent, payload: dict) -> FilterState:
        """Traite un événement et retourne le nouvel état."""
        current_state = self._get_current_state()
        
        match event:
            case FilterEvent.MODE_CHANGED:
                return current_state.with_mode(payload["mode"])
            case FilterEvent.PLAYLISTS_CHANGED:
                return current_state.with_playlists(payload["playlists"])
            case FilterEvent.SAVE_REQUESTED:
                self._save_to_repository(current_state)
                return current_state
            ...
        
        return current_state
```

---

## 4. Solution recommandée : Options statiques

### 4.1 Concept

**Au lieu de** : *"Lister uniquement les playlists jouées dans la période sélectionnée"*  
**Proposer** : *"Lister toutes les playlists du jeu, griser celles non jouées"*

### 4.2 Avantages

1. **✅ Stabilité** : Les options ne changent jamais, l'utilisateur peut mémoriser ses préférences
2. **✅ Prévisibilité** : L'utilisateur sait toujours ce qui est disponible
3. **✅ Feedback visuel** : Les options grisées indiquent "pas de matchs dans cette période"
4. **✅ Pas de perte de sélection** : Les choix sont conservés même si temporairement indisponibles

### 4.3 Implémentation

```python
def render_static_playlist_filter(
    *,
    all_playlists: dict[str, str],  # Toutes les playlists du jeu
    played_playlists: set[str],     # Playlists jouées dans la période
    selected: set[str],             # Playlists sélectionnées
    player_key: str,                # Scopage
) -> set[str]:
    """Rend un filtre avec options statiques."""
    
    with st.expander("Playlists", expanded=False):
        for uuid, label in sorted(all_playlists.items(), key=lambda x: x[1]):
            is_available = uuid in played_playlists
            is_selected = uuid in selected
            
            # Griser si non disponible
            disabled = not is_available
            help_text = None if is_available else "Aucun match dans cette période"
            
            new_val = st.checkbox(
                label,
                value=is_selected,
                key=f"filter_playlist_{player_key}_{uuid}",
                disabled=disabled,
                help=help_text,
            )
            
            if new_val != is_selected:
                if new_val:
                    selected.add(uuid)
                else:
                    selected.discard(uuid)
    
    return selected
```

### 4.4 Exemple visuel

```
┌─ Playlists ────────────────────────────────┐
│ ☑ Partie rapide                            │  ← Disponible, sélectionnée
│ ☑ Arène classée                            │  ← Disponible, sélectionnée
│ ☐ Assassin classé                          │  ← Disponible, non sélectionnée
│ ☐ BTB (i)                                  │  ← NON disponible (grisé)
│     "Aucun match dans cette période"       │
│ ☐ Firefight (i)                            │  ← NON disponible (grisé)
│     "Aucun match dans cette période"       │
└────────────────────────────────────────────┘
```

---

## 5. Solution alternative : Mode avancé / Mode simple

Si vous souhaitez conserver le listing dynamique dans certains cas :

### 5.1 Mode Simple (par défaut)

- Options statiques (toutes les playlists du jeu)
- Idéal pour les utilisateurs occasionnels
- Pas de désorientation

### 5.2 Mode Avancé (opt-in)

- Options dynamiques (uniquement celles jouées)
- Pour les power users qui veulent filtrer rapidement
- Avec un warning : "Les options changent selon votre sélection"

### 5.3 Implémentation

```python
# Dans settings
advanced_filters_mode = st.checkbox(
    "Mode avancé : Lister uniquement les options jouées",
    value=False,
    help="⚠️ Les options changeront selon votre période/session",
)

if advanced_filters_mode:
    # Listing dynamique (comportement actuel)
    playlists = get_playlists_from_filtered_data(...)
else:
    # Listing statique (recommandé)
    playlists = get_all_playlists_from_metadata(...)
```

---

## 6. Plan de migration (sans casser l'existant)

### Phase 1 : Nettoyage immédiat (1-2j)

1. **Fixer le nettoyage au changement de joueur**
   - Ajouter toutes les clés manquantes (`gap_minutes`, `min_matches_maps`, etc.)
   - Supprimer toutes les clés widgets (`filter_*_cb_*`, `filter_*_version`, etc.)
   - Centraliser dans `filter_state.py` : `FILTER_KEYS_TO_CLEAR`

2. **Tests de non-régression**
   - Scénario A → B → A (isolation)
   - Vérifier que les JSON ne sont pas corrompus

### Phase 2 : Options statiques (2-3j)

1. **Créer `filter_options_provider.py`**
   - Charger toutes les playlists/modes/cartes depuis `metadata.duckdb`
   - Cache Streamlit (`@st.cache_resource`)

2. **Modifier `checkbox_filter.py`**
   - Ajouter paramètre `available_options: set[str]`
   - Griser les options non disponibles

3. **Adapter `filters_render.py`**
   - Calculer `played_playlists` depuis le DataFrame
   - Passer `all_playlists` et `played_playlists` aux checkboxes

### Phase 3 : Persistance par UUID (3-4j)

1. **Créer mapping `label_fr → UUID`**
   - Depuis `metadata.duckdb`
   - Gérer les traductions

2. **Migration des JSON existants**
   - Script de conversion `label_fr → UUID`
   - Backup avant migration

3. **Adapter `filter_state.py`**
   - Lire/écrire des UUIDs au lieu de labels
   - Validation des UUIDs au chargement

### Phase 4 : Refactoring architecture (1-2 semaines)

1. **Créer `filter_manager.py`**
   - `FilterState` (dataclass immutable)
   - `FilterDispatcher` (événements)
   - Logique métier centralisée

2. **Simplifier `filters_render.py`**
   - Devient un pur composant UI
   - Lecture seule de `FilterState`
   - Dispatch événements

3. **Scopage par joueur**
   - Clés `session_state` préfixées
   - Plus de conflits entre joueurs

---

## 7. Comparaison des approches

| Critère | Architecture actuelle | Option 1 : Nettoyage seul | Option 2 : Options statiques | Option 3 : Refactoring complet |
|---------|----------------------|---------------------------|------------------------------|-------------------------------|
| **Complexité** | Élevée (4 modules) | Moyenne | Moyenne | Faible (3 couches) |
| **Stabilité** | ❌ Bugs fréquents | ⚠️ Réduit les bugs | ✅ Stable | ✅ Très stable |
| **UX** | ❌ Désorientation | ⚠️ Idem | ✅ Prévisible | ✅ Excellente |
| **Performance** | ⚠️ Refiltrages multiples | ⚠️ Idem | ✅ Bon | ✅ Excellent |
| **Maintenabilité** | ❌ Difficile | ⚠️ Fragile | ✅ Bonne | ✅ Excellente |
| **Effort** | - | 1-2j | 5-7j | 2-3 semaines |
| **Risque** | - | Faible | Moyen | Moyen |

---

## 8. Recommandations finales

### 8.1 Court terme (cette semaine)

**👉 Phase 1 : Nettoyage immédiat**

- Fixer le changement de joueur (nettoyage complet)
- Centraliser les clés dans `filter_state.py`
- Tests A → B → A

**Rationale** : Résout 80% des bugs avec un effort minimal.

### 8.2 Moyen terme (2-3 semaines)

**👉 Phase 2 + 3 : Options statiques + UUIDs**

- Listing statique avec options grisées
- Persistance par UUID pour robustesse
- Meilleure UX

**Rationale** : Résout le problème de conception fondamental (listing dynamique).

### 8.3 Long terme (1-2 mois)

**👉 Phase 4 : Refactoring architecture**

- 3 couches (Présentation / Métier / Données)
- État immutable (`FilterState`)
- Dispatcher d'événements

**Rationale** : Maintenabilité à long terme, facilite l'ajout de nouvelles fonctionnalités.

---

## 9. Questions pour validation

Avant de coder, j'ai besoin de clarifier :

1. **UX souhaitée** : Préférez-vous les options statiques (toutes les playlists) ou dynamiques (uniquement celles jouées) ?

2. **Priorités** : Quels bugs sont les plus critiques ?
   - Corruption au changement de joueur ?
   - Désorientation lors du changement de période ?
   - Performance ?

3. **Migration** : Acceptez-vous une migration des JSON existants (labels → UUIDs) ?

4. **Effort** : Préférez-vous :
   - Fix rapide (Phase 1 seule) ?
   - Fix complet (Phases 1-3) ?
   - Refactoring profond (Phases 1-4) ?

5. **Backward compatibility** : Faut-il supporter l'ancien format JSON pendant une période de transition ?

---

## 10. Prochaines étapes

Après validation de cette analyse :

1. ✅ Créer des tickets détaillés pour chaque phase
2. ✅ Définir les critères de succès et les tests
3. ✅ Commencer l'implémentation en mode itératif (petits PRs)
4. ✅ Documentation utilisateur (changelog, guide migration)

---

**⏸️ En attente de vos retours avant de toucher au code.**
