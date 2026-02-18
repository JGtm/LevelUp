# Recommandations : Redesign des filtres sidebar

> **Pour l'utilisateur JGtm** - Conseils concrets pour décider de la suite

---

## 🎯 TL;DR

**Tu as raison** : Le problème vient du **listing dynamique** (on liste uniquement ce qui est disponible dans la période/session sélectionnée).

**Solution recommandée** : **Options statiques** (toutes les playlists du jeu) avec **options grisées** si pas de matchs.

**Effort** : 5-7 jours (incluant fix du changement de joueur)

**Alternative rapide** : Juste fixer le changement de joueur (1-2 jours) mais ne résout pas le problème de fond.

---

## 📋 Menu des options

### Option A : Fix minimal (1-2 jours) ⚡

**Quoi** :
- Fixer le nettoyage au changement de joueur (clés widgets manquantes)
- Centraliser les clés à nettoyer dans `filter_state.py`
- Tests A → B → A

**Avantages** :
- ✅ Résout la corruption entre joueurs
- ✅ Effort minimal
- ✅ Peu de risque

**Inconvénients** :
- ❌ Ne résout pas le problème de désorientation (listing dynamique)
- ❌ L'utilisateur subira toujours les options qui apparaissent/disparaissent
- ❌ Solution temporaire, pas pérenne

**Recommandation** : ⚠️ **Seulement si budget temps très limité**

---

### Option B : Options statiques (5-7 jours) ⭐ RECOMMANDÉ

**Quoi** :
- Phase 1 : Fix changement de joueur (1-2j)
- Phase 2 : Options statiques depuis `metadata.duckdb` (2-3j)
- Phase 3 : Persistance par UUID (optionnel, 2j)

**Avantages** :
- ✅ Résout TOUS les problèmes de conception
- ✅ UX prévisible et stable
- ✅ Maintenabilité à long terme
- ✅ Performance améliorée

**Inconvénients** :
- ⚠️ Effort moyen (1 semaine)
- ⚠️ Besoin de migrer les JSON existants (si Phase 3)

**Recommandation** : ⭐ **Meilleur ROI**

**Exemple d'interface** :

```
┌─ Playlists ────────────────────────────────────────┐
│ ☑ Partie rapide (145 matchs)                      │
│ ☑ Arène classée (87 matchs)                       │
│ ☐ Assassin classé (23 matchs)                     │
│ ☐ BTB (i)                                          │
│     "Aucun match dans cette période"               │
│ ☐ Firefight (i)                                    │
│     "Aucun match dans cette période"               │
│                                                    │
│ 📊 232 matchs disponibles avec cette sélection    │
└────────────────────────────────────────────────────┘
```

---

### Option C : Refactoring complet (2-3 semaines) 🚀

**Quoi** :
- Phase 1-3 (voir Option B)
- Phase 4 : Nouvelle architecture (3 couches, état immutable, dispatcher)

**Avantages** :
- ✅ Tous les avantages de Option B
- ✅ Code maintenable et extensible
- ✅ Facilite les évolutions futures
- ✅ Séparation claire des responsabilités

**Inconvénients** :
- ⚠️ Effort important (2-3 semaines)
- ⚠️ Risque de régression si mal testé
- ⚠️ Nécessite refactoring de plusieurs modules

**Recommandation** : 🚀 **Seulement si vous avez le temps et voulez industrialiser**

---

## 🤔 Comment choisir ?

### Vous êtes pressé / prototype

→ **Option A** (fix minimal)

Critères :
- Besoin d'une solution sous 2 jours
- Application en phase de prototype
- Budget temps très limité

**Mais attention** : Vous devrez revenir sur le problème plus tard.

---

### Vous voulez une vraie solution

→ **Option B** (options statiques) ⭐

Critères :
- Acceptez 1 semaine d'effort
- Voulez une UX stable et prévisible
- Voulez résoudre le problème de fond

**C'est le meilleur compromis** effort/bénéfice.

---

### Vous voulez industrialiser

→ **Option C** (refactoring complet)

Critères :
- Avez 2-3 semaines devant vous
- Voulez un code maintenable à long terme
- Prévoyez d'ajouter des fonctionnalités complexes

**Investissement à long terme**, utile si le projet grandit.

---

## 🎨 Design UX : Options statiques vs dynamiques

### Pourquoi les options statiques sont meilleures

#### 1. Principe de **Consistance** (Nielsen)

> "Les utilisateurs ne devraient pas avoir à se demander si différents mots, situations ou actions signifient la même chose."

**Problème actuel** : Les mêmes actions (changer de période) produisent des résultats différents (options qui changent).

**Solution** : Les options restent **toujours les mêmes**, l'utilisateur sait **où cliquer**.

#### 2. Principe de **Visibilité de l'état** (Norman)

> "Le système doit toujours tenir l'utilisateur informé de ce qui se passe."

**Problème actuel** : Quand une playlist disparaît, l'utilisateur ne sait pas si :
- Elle n'existe plus ?
- Il n'a pas joué de matchs dessus ?
- C'est un bug ?

**Solution** : Toutes les playlists affichées, les **grisées** = "pas de matchs dans cette période".

#### 3. Principe de **Contrôle utilisateur**

> "Les utilisateurs doivent avoir le contrôle et la liberté."

**Problème actuel** : L'utilisateur ne peut pas dire "Je veux toujours voir Arène classée" car l'option peut disparaître.

**Solution** : L'utilisateur peut **garder ses préférences** même si temporairement indisponibles.

---

### Exemple concret : Spotify

Spotify utilise les **options statiques** pour les filtres :

```
Tous vos playlists sont toujours visibles
│
├─ Rock (125 titres)     ← Disponible
├─ Jazz (87 titres)      ← Disponible
├─ Classique (0 titres)  ← GRISÉ, mais visible
└─ Pop (234 titres)      ← Disponible
```

Avantage : Vous savez toujours où chercher, même si une playlist est vide.

---

## 💡 Conseils de mise en œuvre

### Si vous choisissez Option B (recommandé)

#### Étape 1 : Fix du changement de joueur (Jour 1)

```python
# streamlit_app.py

# Centraliser les clés à nettoyer
from src.ui.filter_state import get_all_filter_keys_to_clear

# Au changement de joueur
def handle_player_change(old_xuid, old_db_path, new_xuid, new_db_path):
    # 1. Sauvegarder l'ancien joueur
    save_filter_preferences(old_xuid, old_db_path)
    
    # 2. Nettoyage EXHAUSTIF
    keys_to_clear = get_all_filter_keys_to_clear(st.session_state)
    for key in keys_to_clear:
        del st.session_state[key]
    
    # 3. Charger le nouveau joueur
    apply_filter_preferences(new_xuid, new_db_path)
    
    st.rerun()
```

#### Étape 2 : Provider d'options statiques (Jour 2-3)

```python
# src/app/filter_options_provider.py

@st.cache_resource
def get_all_playlists() -> dict[str, str]:
    """Retourne TOUTES les playlists du jeu depuis metadata.duckdb.
    
    Returns:
        {uuid: label_fr}
    """
    conn = duckdb.connect("data/warehouse/metadata.duckdb", read_only=True)
    result = conn.execute("""
        SELECT 
            asset_id AS uuid,
            name AS label
        FROM playlists
        ORDER BY name
    """).pl()
    conn.close()
    
    return {row["uuid"]: row["label"] for row in result.iter_rows(named=True)}
```

#### Étape 3 : Adapter les checkboxes (Jour 4-5)

```python
# src/ui/components/checkbox_filter.py

def render_static_checkbox_filter(
    *,
    label: str,
    all_options: dict[str, str],      # NOUVEAU : toutes les options du jeu
    available_options: set[str],      # NOUVEAU : options jouées par le joueur
    selected: set[str],               # Sélection actuelle
    session_key: str,
) -> set[str]:
    """Rend un filtre avec options statiques."""
    
    with st.expander(label, expanded=False):
        for uuid, label_text in sorted(all_options.items(), key=lambda x: x[1]):
            is_available = uuid in available_options
            is_selected = uuid in selected
            
            # Compter les matchs disponibles avec cette option
            if is_available:
                match_count = count_matches_with_option(uuid)
                label_with_count = f"{label_text} ({match_count} matchs)"
            else:
                label_with_count = label_text
            
            # Griser si non disponible
            new_val = st.checkbox(
                label_with_count,
                value=is_selected,
                key=f"{session_key}_{uuid}",
                disabled=not is_available,
                help=None if is_available else "Aucun match dans cette période",
            )
            
            if new_val != is_selected:
                if new_val:
                    selected.add(uuid)
                else:
                    selected.discard(uuid)
    
    return selected
```

#### Étape 4 : Tests (Jour 6-7)

```python
# tests/test_static_filters.py

def test_static_filters_stability():
    """Les options ne changent jamais."""
    all_playlists = get_all_playlists()
    
    # Simuler changement de période
    options_period1 = render_filter(period="2024-01")
    options_period2 = render_filter(period="2024-02")
    
    # Les options disponibles sont les mêmes
    assert set(options_period1.keys()) == set(options_period2.keys())
    assert set(options_period1.keys()) == set(all_playlists.keys())

def test_grayed_options():
    """Les options non disponibles sont grisées."""
    all_playlists = {"uuid1": "P1", "uuid2": "P2"}
    played_playlists = {"uuid1"}  # Seulement P1 jouée
    
    checkboxes = render_static_checkbox_filter(
        all_options=all_playlists,
        available_options=played_playlists,
        selected=set(),
        session_key="test",
    )
    
    # P2 est affiché mais disabled
    assert st.session_state["test_uuid2_disabled"] == True

def test_selection_persists_when_unavailable():
    """La sélection est conservée même si temporairement indisponible."""
    selected = {"uuid2"}  # P2 sélectionnée
    played_playlists = {"uuid1"}  # Mais seulement P1 jouée actuellement
    
    checkboxes = render_static_checkbox_filter(
        all_options=all_playlists,
        available_options=played_playlists,
        selected=selected,
        session_key="test",
    )
    
    # P2 reste sélectionnée (même si grisée)
    assert "uuid2" in selected
```

---

## 🔍 Points d'attention

### 1. Migration des préférences existantes

Si vous choisissez la persistance par UUID (Phase 3), vous devrez migrer les JSON existants :

```python
# scripts/migrate_filter_preferences.py

def migrate_labels_to_uuids():
    """Convertit les labels FR en UUIDs dans les JSON existants."""
    filters_dir = Path(".streamlit/filter_preferences")
    label_to_uuid = build_label_to_uuid_mapping()  # Depuis metadata.duckdb
    
    for json_file in filters_dir.glob("*.json"):
        # Backup
        shutil.copy(json_file, json_file.with_suffix(".json.bak"))
        
        # Conversion
        prefs = load_json(json_file)
        prefs["playlists"] = [
            label_to_uuid.get(label, label) 
            for label in prefs.get("playlists_selected", [])
        ]
        save_json(json_file, prefs)
```

### 2. Feedback utilisateur

Affichez un compteur en temps réel :

```python
# Après application des filtres
match_count = len(filtered_df)
st.info(f"📊 {match_count} matchs disponibles avec cette sélection")
```

### 3. Options avancées (optionnel)

Pour les power users, proposez un mode avancé :

```python
# Dans settings
advanced_mode = st.checkbox("Mode avancé : Masquer les options indisponibles")

if advanced_mode:
    # Listing dynamique (comportement actuel)
    options = get_played_playlists(...)
else:
    # Listing statique (recommandé)
    options = get_all_playlists()
```

---

## ✅ Critères de succès

Après implémentation, vous devez pouvoir valider :

1. **✅ Isolation entre joueurs**
   - Scénario : A → B → A
   - Résultat : Les filtres de A sont **exactement** ceux laissés initialement

2. **✅ Stabilité des options**
   - Scénario : Changer de période/session
   - Résultat : Les options affichées restent **identiques**

3. **✅ Feedback visuel**
   - Scénario : Sélectionner une playlist sans matchs
   - Résultat : Option **grisée** avec tooltip explicatif

4. **✅ Performances**
   - Temps de rendu < 200ms
   - Pas de reruns intempestifs

5. **✅ Tests**
   - Coverage ≥ 80% sur les modules de filtres
   - Tests E2E pour les scénarios critiques

---

## 📞 Prochaines étapes

1. **Décidez** quelle option vous convient (A, B ou C)

2. **Clarifiez** les questions ouvertes :
   - Acceptez-vous une migration des JSON (labels → UUIDs) ?
   - Voulez-vous un mode avancé (opt-in listing dynamique) ?
   - Préférez-vous sauvegarde automatique ou bouton explicite ?

3. **Validez** l'interface proposée (mockup)
   - Options grisées OK ?
   - Compteur de matchs OK ?
   - Tooltip "Aucun match" OK ?

4. **Je commence** l'implémentation avec votre accord

---

## 🆘 Besoin d'aide pour décider ?

**Questions à vous poser** :

- Quel est le bug le plus gênant au quotidien ?
  - Corruption entre joueurs → Option A suffit
  - Désorientation avec les options qui changent → Option B minimum

- Combien de temps pouvez-vous consacrer à cette refonte ?
  - 1-2 jours → Option A
  - 1 semaine → Option B ⭐
  - 2-3 semaines → Option C

- Le projet va-t-il grandir significativement ?
  - Non → Option A ou B
  - Oui → Option C recommandé

**Mon avis personnel** : Option B (options statiques) est le **sweet spot**.

- Résout vraiment le problème
- Effort raisonnable (1 semaine)
- UX bien meilleure
- Code maintenable

---

**🎯 En attente de votre décision pour passer à l'implémentation !**
