# Solution concrète : Filtres basés sur l'exclusion

> **Basé sur le workflow réel** : "Tout sauf Firefight" (et éventuellement d'autres exclusions)

---

## 🎯 Workflow utilisateur confirmé

### Cas d'usage principal (90%)
```
1. Ouvre l'app
2. Sélectionne une période OU une session
3. Garde tout coché (playlists, modes, cartes)
4. SAUF Firefight (décoché par défaut)
5. Regarde les stats PvP
```

### Cas d'usage secondaire (9%)
```
1. Veut exclure une playlist/mode spécifique
2. Décoche l'élément temporairement
3. Regarde les stats filtrées
```

### Cas d'usage rare (1%)
```
1. Veut regarder UNIQUEMENT PvE (Firefight)
2. Décoche tout sauf Firefight
3. Regarde les stats PvE
```

**Conclusion** : Philosophie = **EXCLUSION** ("tout sauf X") pas INCLUSION ("seulement Y")

---

## 🐛 Problème actuel

### Ce qui est sauvegardé
```json
{
  "playlists_selected": ["Partie rapide", "Arène classée", "Assassin classé"]
}
```

### Problème quand la période change
```
Nouvelle période → Nouvelle playlist "BTB" disponible
→ BTB n'est pas dans playlists_selected
→ BTB est décoché automatiquement ❌

Mais l'utilisateur voulait "tout sauf Firefight", pas "ces 3 playlists exactes"
```

---

## ✅ Solution : Persistance basée sur l'exclusion

### 1. Nouveau format de sauvegarde

```python
@dataclass
class FilterPreferences:
    # ... (autres champs)
    
    # NOUVEAU : Mode de filtrage
    playlists_mode: str | None = None  # "include" ou "exclude"
    modes_mode: str | None = None
    maps_mode: str | None = None
    
    # Listes peuvent être inclusions OU exclusions selon le mode
    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None
```

### 2. Exemple de sauvegarde (cas normal)

```json
{
  "playlists_mode": "exclude",
  "playlists_selected": ["Firefight: Gruntpocalypse", "Firefight: Heroic"],
  "modes_mode": "exclude",
  "modes_selected": [],
  "maps_mode": "exclude",
  "maps_selected": []
}
```

**Signification** : "Tout sauf les playlists Firefight"

### 3. Exemple de sauvegarde (cas PvE uniquement)

```json
{
  "playlists_mode": "include",
  "playlists_selected": ["Firefight: Gruntpocalypse", "Firefight: Heroic"],
  "modes_mode": "include",
  "modes_selected": [],
  "maps_mode": "include",
  "maps_selected": []
}
```

**Signification** : "Uniquement les playlists Firefight" (mode rare)

---

## 🔧 Implémentation

### Étape 1 : Modifier FilterPreferences

```python
# src/ui/filter_state.py

@dataclass
class FilterPreferences:
    """Préférences de filtres pour un joueur."""
    
    # Mode de filtre ("Période" ou "Sessions")
    filter_mode: str | None = None
    
    # Mode Période
    start_date: str | None = None
    end_date: str | None = None
    
    # Mode Sessions
    gap_minutes: int | None = None
    picked_session_label: str | None = None
    
    # NOUVEAU : Modes de filtrage (include/exclude)
    playlists_mode: str | None = "exclude"  # Défaut = exclusion
    modes_mode: str | None = "exclude"
    maps_mode: str | None = "exclude"
    
    # Filtres (interprétés selon le mode)
    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None
    
    def to_dict(self) -> dict[str, Any]:
        """Convertit en dictionnaire pour sérialisation JSON."""
        return {k: v for k, v in asdict(self).items() if v is not None}
    
    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> FilterPreferences:
        """Crée depuis un dictionnaire (désérialisation JSON)."""
        # Rétrocompatibilité : si pas de mode, assume "include" (ancien comportement)
        if "playlists_selected" in data and "playlists_mode" not in data:
            data["playlists_mode"] = "include"
        if "modes_selected" in data and "modes_mode" not in data:
            data["modes_mode"] = "include"
        if "maps_selected" in data and "maps_mode" not in data:
            data["maps_mode"] = "include"
        
        return cls(**{k: v for k, v in data.items() if k in cls.__dataclass_fields__})
```

### Étape 2 : Modifier save_filter_preferences

```python
# src/ui/filter_state.py

def save_filter_preferences(
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
) -> None:
    """Sauvegarde les préférences de filtres pour un joueur."""
    if preferences is None:
        preferences = FilterPreferences()
        
        # Mode de filtre
        filter_mode = st.session_state.get("filter_mode")
        if filter_mode in ("Période", "Sessions"):
            preferences.filter_mode = filter_mode
        
        # ... (dates, session, etc.)
        
        # NOUVEAU : Détection du mode (include/exclude)
        playlists = st.session_state.get("filter_playlists")
        if isinstance(playlists, (set, list)):
            playlists_set = set(playlists)
            all_playlists = st.session_state.get("_all_playlists_available", set())
            
            # Si presque tout est sélectionné, c'est probablement de l'exclusion
            if len(playlists_set) > len(all_playlists) * 0.7:
                preferences.playlists_mode = "exclude"
                preferences.playlists_selected = sorted(all_playlists - playlists_set)
            else:
                preferences.playlists_mode = "include"
                preferences.playlists_selected = sorted(playlists_set)
        
        # Idem pour modes et maps...
    
    # Sauvegarder dans le fichier
    player_key = _get_player_key(xuid, db_path)
    file_path = _get_filter_file_path(player_key)
    
    try:
        with open(file_path, "w", encoding="utf-8") as f:
            json.dump(preferences.to_dict(), f, indent=2, ensure_ascii=False)
    except Exception as e:
        st.warning(f"Impossible de sauvegarder les préférences de filtres: {e}")
```

### Étape 3 : Modifier apply_filter_preferences

```python
# src/ui/filter_state.py

def apply_filter_preferences(
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
) -> None:
    """Applique les préférences de filtres dans session_state."""
    if preferences is None:
        preferences = load_filter_preferences(xuid, db_path)
        if preferences is None:
            return
    
    # ... (mode, dates, session)
    
    # NOUVEAU : Application selon le mode (include/exclude)
    if preferences.playlists_selected is not None:
        all_playlists = st.session_state.get("_all_playlists_available", set())
        
        if preferences.playlists_mode == "exclude":
            # Mode exclusion : tout sauf les exclus
            excluded = set(preferences.playlists_selected)
            st.session_state["filter_playlists"] = all_playlists - excluded
        else:
            # Mode inclusion : uniquement les inclus (ancien comportement)
            st.session_state["filter_playlists"] = set(preferences.playlists_selected)
    
    # Idem pour modes et maps...
```

### Étape 4 : Modifier render_checkbox_filter (optionnel)

```python
# src/ui/components/checkbox_filter.py

def render_checkbox_filter(
    *,
    label: str,
    options: list[str],
    session_key: str,
    default_unchecked: set[str] | None = None,
    show_select_buttons: bool = True,
    expanded: bool = False,
    show_mode_toggle: bool = False,  # NOUVEAU
) -> set[str]:
    """Affiche un expander avec checkboxes."""
    
    # ... (code existant)
    
    with st.expander(title, expanded=expanded):
        # NOUVEAU : Toggle include/exclude (optionnel, pour power users)
        if show_mode_toggle:
            mode_key = f"{session_key}_mode"
            if mode_key not in st.session_state:
                st.session_state[mode_key] = "exclude"
            
            mode = st.radio(
                "Mode",
                ["Tout sauf", "Uniquement"],
                key=mode_key,
                horizontal=True,
                label_visibility="collapsed",
            )
            
            if mode == "Tout sauf":
                st.caption("Cochez les éléments à EXCLURE")
            else:
                st.caption("Cochez les éléments à INCLURE")
        
        # Boutons Tout / Aucun
        if show_select_buttons and len(options) > 1:
            cols = st.columns(2)
            if cols[0].button("✓ Tout", key=f"{session_key}_all", width="stretch"):
                st.session_state[session_key] = set(options)
                st.rerun()
            if cols[1].button("✗ Aucun", key=f"{session_key}_none", width="stretch"):
                st.session_state[session_key] = set()
                st.rerun()
        
        # Checkboxes...
```

---

## 🎨 UX proposée

### Interface par défaut (mode simple)

```
┌─ Playlists (11/12) ────────────────────────┐
│ ☑ Partie rapide                            │
│ ☑ Arène classée                            │
│ ☑ Assassin classé                          │
│ ☑ BTB                                      │
│ ☐ Firefight: Gruntpocalypse                │  ← Décoché (exclusion)
│ ☐ Firefight: Heroic                        │  ← Décoché (exclusion)
│ ☑ Action Sack                              │
│ ...                                        │
│                                            │
│ 💡 Mode exclusion : Firefight exclu        │  ← Feedback
└────────────────────────────────────────────┘
```

### Interface avancée (avec toggle, optionnel)

```
┌─ Playlists (11/12) ────────────────────────┐
│ Mode : ⦿ Tout sauf  ○ Uniquement           │  ← Toggle
│ Cochez les éléments à EXCLURE              │  ← Aide contextuelle
│                                            │
│ ☐ Partie rapide                            │
│ ☐ Arène classée                            │
│ ☑ Firefight: Gruntpocalypse                │  ← Coché = exclu
│ ☑ Firefight: Heroic                        │  ← Coché = exclu
│ ...                                        │
└────────────────────────────────────────────┘
```

---

## 📊 Avantages

### 1. Résout le problème de désorientation

**Avant** :
```
Période 1 : Playlists = [A, B, C] → Tout coché sauf Firefight
Période 2 : Playlists = [A, B, C, D, E] → D et E décochés ❌
```

**Après** :
```
Préférence = "exclude: [Firefight]"
Période 1 : Playlists = [A, B, C] → Tout coché sauf Firefight ✅
Période 2 : Playlists = [A, B, C, D, E] → Tout coché sauf Firefight ✅
```

### 2. Supporte les deux philosophies

- **Mode exclusion** (défaut) : "Tout sauf Firefight"
- **Mode inclusion** (rare) : "Uniquement Firefight"

### 3. Rétrocompatible

Les anciens JSON sont interprétés comme mode "include" (comportement actuel).

### 4. Pas de listing statique obligatoire

On peut garder le listing dynamique, l'exclusion fonctionne avec.

---

## 🚀 Plan d'implémentation

### Phase 1 : Backend (1 jour)

1. ✅ Modifier `FilterPreferences` (ajouter `*_mode`)
2. ✅ Modifier `save_filter_preferences` (détecter mode)
3. ✅ Modifier `apply_filter_preferences` (appliquer mode)
4. ✅ Tests unitaires

### Phase 2 : UI simple (1 jour)

1. ✅ Ajouter feedback visuel ("Mode exclusion : X exclu")
2. ✅ Tests manuels
3. ✅ Migration des JSON existants

### Phase 3 : UI avancée (optionnel, 1 jour)

1. ⚠️ Ajouter toggle "Tout sauf" / "Uniquement"
2. ⚠️ Tests utilisateur
3. ⚠️ Documentation

**Total : 2-3 jours** (Phase 1+2 obligatoires, Phase 3 optionnelle)

---

## ❓ Questions pour validation

1. **UX simple ou avancée ?**
   - Simple = Pas de toggle, juste exclusion automatique
   - Avancée = Toggle visible pour basculer

2. **Migration automatique ?**
   - Détecter si ancien JSON et convertir automatiquement ?
   - Ou garder mode "include" pour anciens JSON ?

3. **Feedback visuel ?**
   - Afficher "Mode exclusion : Firefight exclu" ?
   - Ou rester discret ?

4. **Tester rapidement ?**
   - Je peux implémenter Phase 1 en 2h
   - Tu pourras tester et valider l'approche

---

## 🎯 Ma recommandation finale

**Implémenter Phase 1 + 2 (UX simple)**

**Pourquoi** :
- ✅ Résout ton problème (persistance de "tout sauf Firefight")
- ✅ Pas de changement UX visible (juste ça marche mieux)
- ✅ Effort minimal (2 jours)
- ✅ Rétrocompatible
- ✅ Extensible (Phase 3 plus tard si besoin)

**Tu valides ?** Je commence l'implémentation maintenant ? 🚀
