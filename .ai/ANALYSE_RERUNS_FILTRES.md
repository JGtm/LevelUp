# Analyse : Rechargements intempestifs des filtres

> **Challenge** : "Chaque filtre playlist entraînait un rechargement (cascade). Normalement on n'a plus ça ?"

---

## 🔴 Problème actuel

### Comportement constaté (checkbox_filter.py)

```python
# Ligne 194-204
for opt in options:
    checked = opt in st.session_state[session_key]
    new_val = st.checkbox(
        opt,
        value=checked,
        key=f"{session_key}_cb_{opt}",
    )
    if new_val and opt not in st.session_state[session_key]:
        st.session_state[session_key] = st.session_state[session_key] | {opt}
    elif not new_val and opt in st.session_state[session_key]:
        st.session_state[session_key] = st.session_state[session_key] - {opt}
```

**❌ Pas de `st.rerun()` ici → BON !**

Mais...

```python
# Ligne 186-191 (Boutons Tout/Aucun)
if cols[0].button("✓ Tout", key=f"{session_key}_all", width="stretch"):
    st.session_state[session_key] = set(options)
    st.rerun()  # ← RERUN ICI
if cols[1].button("✗ Aucun", key=f"{session_key}_none", width="stretch"):
    st.session_state[session_key] = set()
    st.rerun()  # ← RERUN ICI
```

**✅ Les reruns sont UNIQUEMENT sur les boutons Tout/Aucun**

### Cascade : Comment ça marche ?

```python
# filters_render.py ligne 556+
# Scope après filtre playlist
scope1 = dropdown_base
if playlists_selected and len(playlists_selected) < len(playlist_values):
    scope1 = scope1.filter(pl.col("playlist_ui").is_in(playlists_selected))

# --- Modes ---
mode_values = sorted(
    {str(x).strip() for x in scope1["mode_ui"].to_list()}  # ← Calculé depuis scope1
)
modes_selected = render_hierarchical_checkbox_filter(
    label="Modes",
    options=mode_values,  # ← Options dépendent de playlists_selected
    session_key="filter_modes",
)

# Scope après filtre mode
scope2 = scope1
if modes_selected and len(modes_selected) < len(mode_values):
    scope2 = scope2.filter(pl.col("mode_ui").is_in(modes_selected))

# --- Cartes ---
map_values = sorted(
    {str(x).strip() for x in scope2["map_ui"].to_list()}  # ← Calculé depuis scope2
)
maps_selected = render_checkbox_filter(
    label="Cartes",
    options=map_values,  # ← Options dépendent de modes_selected
    session_key="filter_maps",
)
```

**Comportement actuel** :
1. Tu coches une playlist
2. `st.session_state["filter_playlists"]` est modifié
3. **Streamlit rerun automatiquement** (comportement natif des widgets)
4. Au rerun, `scope1` est recalculé avec la nouvelle sélection
5. `mode_values` change → Les modes disponibles changent
6. Pareil pour les cartes

**✅ C'est le comportement NORMAL de Streamlit**
**❌ MAIS c'est effectivement chiant si tu veux sélectionner plusieurs playlists**

---

## 💡 Analyse : Est-ce vraiment un problème ?

### Cas 1 : Tu sélectionnes UNE playlist

```
1. Clic sur "Arène classée"
2. Rerun automatique (Streamlit)
3. Modes mis à jour (uniquement ceux d'Arène)
4. Cartes mises à jour
5. Stats recalculées
```

**Impact** : 1 rerun = OK

### Cas 2 : Tu sélectionnes PLUSIEURS playlists

```
1. Clic sur "Arène classée"
2. Rerun (1)
3. Clic sur "Assassin classé"
4. Rerun (2)
5. Clic sur "BTB"
6. Rerun (3)
```

**Impact** : 3 reruns = ⚠️ **CHIANT**

### Cas 3 : Tu utilises "Tout"

```
1. Clic sur bouton "✓ Tout"
2. Rerun explicite (st.rerun())
3. Toutes les playlists cochées
4. Modes et cartes mis à jour
```

**Impact** : 1 rerun = OK

---

## 🎯 Solutions possibles

### Solution 1 : Désactiver la cascade (SIMPLE)

**Concept** : Ne pas recalculer les options de Modes/Cartes en fonction de Playlists.

```python
# Au lieu de :
scope1 = dropdown_base.filter(playlists)
mode_values = scope1["mode_ui"].unique()

# Faire :
mode_values = dropdown_base["mode_ui"].unique()  # Toutes les options
```

**Avantages** :
- ✅ Pas de recalcul à chaque sélection
- ✅ Performances améliorées
- ✅ Options stables

**Inconvénients** :
- ❌ Tu vois des modes qui n'existent pas dans les playlists sélectionnées
- ❌ Si tu sélectionnes "Arène classée", tu vois quand même "BTB : Stockpile"

**🤔 Ton avis ?** Compatible avec ton workflow ?

---

### Solution 2 : Bouton "Appliquer les filtres" (TRADITIONNEL)

**Concept** : Les changements ne sont pas appliqués immédiatement.

```python
# Utiliser un état "pending"
playlists_pending = st.session_state.get("filter_playlists_pending", set())

# Checkboxes modifient le pending (pas de rerun)
for opt in options:
    checked = opt in playlists_pending
    new_val = st.checkbox(opt, value=checked, key=f"{session_key}_cb_{opt}")
    if new_val != checked:
        if new_val:
            playlists_pending.add(opt)
        else:
            playlists_pending.discard(opt)

# Bouton pour appliquer
if st.button("🔄 Appliquer les filtres", key="apply_filters"):
    st.session_state["filter_playlists"] = playlists_pending
    st.rerun()
```

**Avantages** :
- ✅ Tu sélectionnes autant de playlists que tu veux
- ✅ 1 seul rerun quand tu cliques "Appliquer"
- ✅ UX traditionnelle (comme des formulaires web)

**Inconvénients** :
- ❌ Un clic supplémentaire
- ❌ Pas de feedback immédiat
- ❌ Plus complexe à implémenter

**🤔 Ton avis ?** Acceptable ?

---

### Solution 3 : Debounce intelligent (HYBRIDE)

**Concept** : Attendre un peu avant d'appliquer, mais pas besoin de bouton.

```python
import time

# Utiliser un timestamp
last_change = st.session_state.get("filter_last_change", 0)
current_time = time.time()

# Si un changement a eu lieu récemment
if current_time - last_change < 0.5:  # 500ms de debounce
    # Afficher un indicateur
    st.caption("⏳ Filtres en cours de modification...")
    # Ne pas recalculer les stats
else:
    # Appliquer les filtres
    # Recalculer les stats
```

**Avantages** :
- ✅ Pas de bouton
- ✅ Feedback immédiat (après 500ms)
- ✅ Tu peux sélectionner rapidement plusieurs playlists

**Inconvénients** :
- ❌ Complexe à implémenter
- ❌ Comportement "magique" (pas évident pour l'utilisateur)
- ❌ Dépend du timing

**🤔 Ton avis ?** Trop complexe ?

---

### Solution 4 : Mode "Rapide" vs "Précis" (PARAMÈTRE)

**Concept** : Laisser l'utilisateur choisir.

```python
# Dans settings ou en haut de sidebar
filter_mode = st.radio(
    "Mode filtres",
    ["Rapide (cascade désactivée)", "Précis (cascade activée)"],
    horizontal=True,
)

if filter_mode == "Rapide":
    # Pas de cascade, toutes les options affichées
    mode_values = dropdown_base["mode_ui"].unique()
else:
    # Cascade activée, options filtrées
    scope1 = dropdown_base.filter(playlists)
    mode_values = scope1["mode_ui"].unique()
```

**Avantages** :
- ✅ Flexibilité maximale
- ✅ L'utilisateur choisit son workflow
- ✅ Facile à implémenter

**Inconvénients** :
- ❌ Un choix de plus pour l'utilisateur
- ❌ Peut être confus

**🤔 Ton avis ?** Intéressant ?

---

## 🧠 Ma recommandation

### Option recommandée : **Solution 1 (Désactiver la cascade)** + **Feedback visuel**

**Pourquoi** :
1. **Simplicité** : Pas de bouton, pas de timing, pas de paramètres
2. **Performance** : Pas de recalcul à chaque sélection
3. **Cohérent** avec ton workflow : "Je laisse tout coché sauf Firefight"
4. **Compatible** avec le mode "exclude" que je t'ai proposé

**Implémentation** :

```python
# filters_render.py

# 1. Calculer TOUTES les options (pas de cascade)
playlist_values = sorted({...from dropdown_base...})
mode_values = sorted({...from dropdown_base...})  # Pas de scope1 !
map_values = sorted({...from dropdown_base...})   # Pas de scope2 !

# 2. Afficher les filtres
playlists_selected = render_checkbox_filter(...)
modes_selected = render_hierarchical_checkbox_filter(...)
maps_selected = render_checkbox_filter(...)

# 3. Appliquer les filtres AU MOMENT DU FILTRAGE des données
# (Pas au moment de l'affichage des checkboxes)
filtered_data = dropdown_base.filter(
    pl.col("playlist_ui").is_in(playlists_selected)
    & pl.col("mode_ui").is_in(modes_selected)
    & pl.col("map_ui").is_in(maps_selected)
)

# 4. Feedback visuel (optionnel)
st.caption(f"📊 {len(filtered_data)} matchs avec cette sélection")
```

**Avec feedback amélioré** :

```python
# Compter les matchs pour chaque option
def count_matches_per_playlist(data, playlists):
    counts = {}
    for p in playlists:
        count = len(data.filter(pl.col("playlist_ui") == p))
        counts[p] = count
    return counts

# Afficher avec compteurs
for opt in playlist_values:
    count = playlist_counts.get(opt, 0)
    label = f"{opt} ({count} matchs)" if count > 0 else f"{opt} (aucun)"
    st.checkbox(label, ...)
```

---

## 📊 Comparaison des solutions

| Solution | Reruns | Implémentation | UX | Workflow actuel |
|----------|--------|----------------|-----|-----------------|
| **1. Désactiver cascade** | 1 par clic | ⭐⭐⭐ Simple | ⭐⭐⭐ | ✅ Compatible |
| **2. Bouton "Appliquer"** | 1 total | ⭐⭐ Moyen | ⭐⭐ | ⚠️ Clic supplémentaire |
| **3. Debounce** | 1 après délai | ⭐ Complexe | ⭐⭐ | ⚠️ Comportement "magique" |
| **4. Mode Rapide/Précis** | Variable | ⭐⭐ Moyen | ⭐⭐ | ⚠️ Choix supplémentaire |
| **Actuel (cascade)** | 1 par clic | - | ⭐ | ❌ Chiant |

---

## 🎯 Challenge de ma logique

### Point 1 : "La cascade est utile ?"

**Mon avis** : Non, pas vraiment.

**Raison** :
- Tu as dit que tu laisses "tout coché sauf Firefight"
- Donc tu ne filtres PAS vraiment par playlist
- La cascade ne t'apporte rien
- Elle te ralentit quand tu veux exclure ponctuellement

**Contre-argument possible** :
- Si tu veux voir "uniquement Arène classée + Assassin"
- La cascade réduit les options → Moins de scrolling

**Ma réponse** :
- Avec l'option de recherche/filtrage dans les checkboxes, le scrolling n'est pas un problème
- Le gain est marginal vs la friction des reruns

### Point 2 : "Les options stables c'est mieux ?"

**Mon avis** : Oui, absolument.

**Raison** :
- Cohérent avec le mode "exclude" que je t'ai proposé
- Tu sais toujours ce qui existe (pas de surprise)
- Performances meilleures
- Pas de reruns intempestifs

**Contre-argument possible** :
- Tu vois des modes qui n'ont pas de matchs
- "BTB : Stockpile" affiché même si tu n'as sélectionné que "Arène"

**Ma réponse** :
- On peut griser les options sans matchs (feedback visuel)
- Ou afficher le compteur "(0 matchs)"
- L'utilisateur comprend que c'est normal

### Point 3 : "Un bouton 'Appliquer' c'est mieux ?"

**Mon avis** : Non, pas pour ton use case.

**Raison** :
- Tu as un workflow "consultation"
- Pas "configuration puis exécution"
- Le feedback immédiat est important
- Un bouton = friction

**Contre-argument possible** :
- Si tu as 50+ playlists
- Sélectionner 45 une par une = 45 reruns
- Bouton = 1 seul rerun

**Ma réponse** :
- Cas rare (tu utilises "Tout" puis tu décoches)
- Bouton "✓ Tout" = 1 rerun déjà
- Le gain est marginal

---

## ✅ Conclusion

**Ma recommandation finale** :

1. **Désactiver la cascade** (Solution 1)
2. **Ajouter des compteurs** pour le feedback
3. **Garder les boutons Tout/Aucun** (ils fonctionnent bien)
4. **Pas de bouton "Appliquer"** (friction inutile)

**Implémentation** : ~2-3h (simple modification de `_render_cascade_filters`)

**Impact** :
- ✅ Plus de reruns intempestifs pour sélections multiples
- ✅ Options stables (cohérent avec mode "exclude")
- ✅ Performances améliorées
- ✅ Workflow préservé

---

## 💬 Questions pour toi

1. **Es-tu d'accord** pour désactiver la cascade ?
2. **Veux-tu** des compteurs de matchs sur les options ?
3. **Préfères-tu** garder un mode "Rapide/Précis" au cas où ?
4. **Autre idée** que je n'ai pas explorée ?

**Challenge ma logique si tu n'es pas d'accord !** 💪
