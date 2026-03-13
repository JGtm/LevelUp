# Analyse des bugs et corrections — 13 mars 2026

> Document généré après exploration approfondie du codebase.
> Chaque bug est annoté : ✅ = compris, ❓ = besoin de clarification.

---

## Table des matières

| # | Bug | Sévérité | Catégorie |
|---|-----|----------|-----------|
| 1 | [Ravageur/Ravager Rebound – ID arme](#1-ravageurravager-rebound) | 🟡 Moyen | Données |
| 2 | ["Parties sélectionnées" → "Matchs joués"](#2-parties-sélectionnées--matchs-joués) | 🟢 Bas | Traduction |
| 3 | [LUSR -435 pts, anomalie de garde-fou](#3-lusr--435-pts-anomalie-garde-fou) | 🔴 Critique | Calcul |
| 4 | [Annotations impact : équipe ennemie affichée](#4-annotations-impact--équipe-ennemie-affichée) | 🔴 Haut | Logique |
| 5 | [Matrice d'Impact : ordre des matchs inversé](#5-matrice-dimpact--ordre-des-matchs) | 🟡 Moyen | UI |
| 6 | [Escouade : graphe perf valeurs >80](#6-escouade--graphe-perf-valeurs-80) | 🟡 Moyen | Données/UI |
| 7 | [F/D/A ratio moyen assassin = ligne plate](#7-fda-ratio-moyen-assassin--ligne-plate) | 🟡 Moyen | Calcul |
| 8 | [Médaille "À table" – feature](#8-médaille-à-table) | 🟢 Feature | Nouveau |
| 9 | [Clic gamertag → ouvre Explorer + matchs session](#9-clic-gamertag--explorer) | 🟡 Moyen | Navigation |
| 10 | [Durée totale incohérente](#10-durée-totale-incohérente) | 🟡 Moyen | Calcul |
| 11 | [Némésis / Souffre-douleur formulation](#11-némésis--souffre-douleur-formulation) | 🟢 Bas | Traduction |
| 12 | [Anciens graphes JGTM/chocoboflor (cache ?)](#12-anciens-graphes-cache) | 🟡 Moyen | Cache |
| 13 | [Opacité barres escouade + distinction morts](#13-opacité-barres-escouade) | 🟡 Moyen | UI |
| 14 | [Tooltips : retirer la date sur graphes escouade](#14-tooltips-date-graphes) | 🟢 Bas | UI |
| 15 | [Matrice d'Impact pas exacte (finisseur manquant)](#15-matrice-dimpact-données-fausses) | 🔴 Haut | Données |
| 16 | [Adornment rang pas à jour](#16-adornment-rang) | 🟡 Moyen | Sync |
| 17 | [Exclure bots du MVP/LVP scoreboard](#17-bots-mvplvp) | 🟡 Moyen | Logique |
| 18 | [Fadetonull introuvable dans Explorer](#18-fadetonull-introuvable) | 🟡 Moyen | Recherche |
| 19 | [Explorer : clic Rechercher reste sur même match](#19-explorer-rechercher-bloqué) | 🟡 Moyen | UI/State |
| 20 | [Sessions : dropdowns absents, inutilisable](#20-sessions-dropdowns-absents) | 🔴 Haut | UI |
| 21 | [Net score cumulé : retirer LUSR](#21-net-score-cumulé--lusr) | 🟢 Bas | UI |
| 22 | [Stats par carte : sens du tableau](#22-stats-par-carte) | 🟢 Bas | Documentation |
| 23 | [Association média : mauvais fuseau horaire](#23-association-média-timezone) | 🔴 Haut | Timezone |
| 24 | [Navigation clic gamertag → switch de DB joueur](#24-navigation-switch-db) | 🔴 Haut | Architecture |
| 25 | [Victoires/Défaites par mode incomplet](#25-victoiresdéfaites-par-mode) | 🟡 Moyen | Filtrage |
| 26 | [Fuseau horaire global — centralisation](#26-fuseau-horaire-global) | 🔴 Haut | Architecture |
| 27 | [Tableau "Par période" : utilité ?](#27-tableau-par-période) | ❓ Clarification | UI |
| 28 | [Escouade : étiquettes axe X → numéro match + carte](#28-escouade--étiquettes-axe-x) | 🟢 Bas | UI |

---

## Détail par bug

---

### 1. Ravageur/Ravager Rebound

**✅ Compréhension** : L'ID hex `bbdb215842c9679f` doit être ajouté au mapping des armes pour identifier le "Ravager Rebound" (tir rebond du Ravageur).

**Analyse** :
- L'ID Ravager standard connu est `C30D87C742C9679F` (dans `weapon_parser_how_it_works_en.md`)
- L'ID `bbdb215842c9679f` **n'existe pas** dans le codebase actuellement
- Le suffixe `42C9679F` est le même que la plupart des armes base → ID légitime
- Le Ravager Rebound est une variante de tir (le projectile rebondit au sol)

**Solution** :
Ajouter l'entrée dans le dictionnaire de mapping des armes (`src/analysis/_weapon_data.py`) :
```python
bytes.fromhex("bbdb215842c9679f"): "Ravager Rebound",
```
Et ajouter la traduction FR dans `_WEAPON_TRANSLATIONS` :
```python
"Ravager Rebound": "Ravageur (Rebond)",
```

**❓ Question** : Confirme-tu que cet ID hex est correct (observé en jeu) ? Tu l'as obtenu comment ?
**❓ Réponse** : Non confirmé à cet instant. Ne pas traiter

---

### 2. "Parties sélectionnées" → "Matchs joués"

**✅ Compréhension** : Le label KPI en haut de l'app dit "Parties sélectionnées" mais devrait dire "Matchs joués".

**Analyse** : 
- Fichier : `src/ui/i18n/widgets.py` ligne 294
- Clé : `kpi_selected_matches`
- Utilisé dans `src/ui/components/kpi.py` ligne 76

**Solution** :
```python
# Avant
"kpi_selected_matches": {"fr": "Parties sélectionnées", "en": "Selected matches"},
# Après
"kpi_selected_matches": {"fr": "Matchs joués", "en": "Matches played"},
```

**Root cause** : Simple erreur de libellé initial.

---

### 3. LUSR -435 pts, anomalie garde-fou

**✅ Compréhension** : Madina a perdu 435 points LUSR sur un seul match (passé de Diamant à Argent), alors qu'un garde-fou `_LUSR_MAX_DELTA = 100.0` existe.

**Analyse** :
- Fichier : `scripts/backfill/strategies.py` lignes 470-485
- Le garde-fou est bien codé : il cap à ±100 par match
- Le code **recalcule** `rating_value = prev_rating[pg] + delta` après cappage

**Hypothèses de cause racine (3 pistes)** :

1. **Données écrites AVANT l'ajout du guard-rail** : Si le LUSR a été calculé avant que le guard-rail ±100 soit ajouté au code, les anciennes valeurs sont dans la DB sans cappage. Solution : recalculer depuis zéro avec `--force-lusr`.

2. **Changement de playlist_group** : Le guard-rail raisonne par `playlist_group`. Si le système de classification a changé (ex: un match classé "ranked" puis "social"), le `prev_rating[pg]` pourrait être absent et le delta serait `None` (pas de cappage). **Mais ça ne donne pas -435.**

3. **Bug dans l'ordre chronologique** : Si les matchs ne sont pas triés correctement par `start_time`, le rating séquentiel pourrait sauter d'un match récent à un ancien, créant un delta artificiel géant.

**Solution proposée** :
1. **Immédiat** : Recalculer le LUSR de Madina avec `--force-lusr` pour appliquer le garde-fou
2. **Structurel** : Ajouter un second garde-fou au niveau de l'**écriture en DB** (pas seulement dans le calcul) :
   - Vérifier le delta par rapport au dernier rating stocké en base
   - Logger un WARNING si le delta dépasse ±100 _même après cappage_ (signe d'anomalie)
3. **Rebond post-anomalie** : Ajouter un mécanisme de "remembering true skill" — si un joueur performe très bien après une chute anormale, accélérer la remontée (multiplicateur de delta positif temporaire). Ceci est une feature avancée.

**❓ Question** : Est-ce que Madina a été resync récemment ? Le bug est-il reproductible ou c'est un cas unique ?
**❓ Réponse** : Oui, sync déjà fais plusieurs fois même après la mise en place du guard-rail et en ayant joué sur le même type de partie et de playlist, non reproductible, deux patchs avaient déjà été tentés.

---

### 4. Annotations impact : équipe ennemie affichée

**✅ Compréhension** : Sur la page Dernier Match, les annotations d'impact (Tourist, First Blood, First Victim, Finisseur) considèrent TOUS les joueurs du match au lieu de filtrer l'équipe alliée uniquement.

**Analyse** :
- Fichier : `src/visualization/_match_impact_events.py`, fonction `compute_single_match_impact()`
- Les `highlight_events` contiennent les kills/deaths de TOUS les joueurs (alliés + ennemis)
- **Aucun filtre d'équipe** n'est appliqué
- Le "first blood" est le premier kill du match, tous joueurs confondus → devrait être le premier kill de l'équipe alliée

**Solution** :
1. **Passer le `team_id` du joueur principal** à `compute_single_match_impact()`
2. **Filtrer les events** : ne garder que ceux dont le `xuid` appartient à la même équipe
3. **Données nécessaires** : Chaque highlight_event doit avoir un champ `team_id` (ou le récupérer depuis `match_participants`)
4. Si `highlight_events` n'a pas de `team_id` : faire un JOIN avec `match_participants` pour obtenir l'équipe de chaque xuid, puis filtrer

**Root cause** : La fonction a été conçue pour analyser "le match entier" alors que l'intent UI est "notre équipe seulement".

---

### 5. Matrice d'Impact : ordre des matchs

**✅ Compréhension** : Dans la Matrice d'Impact escouade, le premier match devrait être à gauche et le dernier à droite. L'ordre semble inversé.

**Analyse** :
- Fichier : `src/ui/pages/teammates_impact.py` ligne ~248
- Code suspect : `sorted_match_ids = sorted({...})` → tri **alphabétique** des match_ids
- Les match IDs sont des UUID/hex : le tri alphabétique ≠ tri chronologique

**Solution** :
Remplacer le tri alphabétique par un tri chronologique :
```python
# Au lieu de sorted({...})
# Utiliser l'ordre d'origine du DataFrame (déjà trié par date)
sorted_match_ids = list(dict.fromkeys(df["match_id"].to_list()))
```
Ou trier explicitement par `start_time`.

**Root cause** : `sorted()` sur des UUID n'a aucun sens chronologique.

---

### 6. Escouade : graphe perf valeurs >80

**✅ Compréhension** : Le graphe de performance escouade affiche des valeurs au-dessus de 80 pour la session du 12 mars, ce qui semble anormal.

**Analyse** :
- Le `performance_score` est borné [0, 100] dans `src/analysis/_performance_relative.py` (clamp + bonus capé)
- Le graphe a un Y-axis hardcodé `range=[0, 100]`

**Hypothèses** :

1. **Les valeurs sont correctes** : Un score >80 est possible dans le système de calcul (top performance relative). Ce n'est peut-être pas un bug mais une surprise → vérifier les données brutes.

2. **Confusion avec une autre métrique** : Le graphe pourrait afficher une métrique différente de `performance_score` (ex: KDA ratio × 10, ou un score raw non normalisé).

3. **Données de coéquipiers mal chargées** : Les stats des autres joueurs pourraient venir du mauvais match ou du mauvais joueur.

**❓ Question** : Peux-tu me donner un exemple concret ? (quel joueur, quel match, quelle valeur affichée vs attendue ?) Ça me permettra de cibler la source exacte.
**❓ Réponse** : Chocoboflor : 12/03/2026 (heure non locale) 22:16	Live Fire	Partie rapide	Arène : Assassin en équipe	Victoire	50 - 36, performance :	71. Dans le graphe indique 87

---

### 7. F/D/A ratio moyen assassin = ligne plate

**✅ Compréhension** : Dans le graphe F/D/A du dernier match, la ligne de ratio moyen "Assassin" (assists) est une ligne horizontale identique pour tous les joueurs.

**Analyse** :
- Fichier : `src/ui/pages/match_view_charts.py` ligne ~218
- La ligne de référence est calculée **une seule fois** comme moyenne globale historique (`hist_ratio`), puis répétée `[hist_ratio] * len(labels)` → même valeur pour tous = ligne droite horizontale
- C'est le **comportement voulu** d'une "ligne de référence historique"

**Mais le vrai problème pourrait être** :
- Mauvais nom : ce n'est pas "assassin" (assists) mais "ratio F/D/A historique moyen"
- Bug de calcul potentiel dans `stats.py` ligne 180 : `(total_kills + total_assists / 2.0)` a un problème d'**ordre des opérateurs** — la division est évaluée avant l'addition. Résultat : `K + (A/2)` au lieu de `(K + A) / 2`.

**❓ Question** : C'est quoi exactement "ratio moyen assassin" dans ce contexte ? 
- Si c'est la **ligne de référence historique** → c'est normal qu'elle soit plate (c'est une moyenne globale)
- Si c'est le **ratio d'assists par joueur par match** → il y a un bug d'affichage
**❓ Réponse** : "Ratio moy. Assassin" est le nom de la légende sur le graphe. Courbe à retirer, après réflexion. 

Dans tous les cas, le bug de priorité opérateur dans `stats.py:180` doit être corrigé.

---

### 8. Médaille "À table" — feature

**✅ Compréhension** : Détecter quand l'équipe ennemie obtient la médaille "À table" pour identifier victoire écrasante / défaite humiliante.

**Analyse** :
- La médaille existe dans `static/medals/medals_fr.json` (ID `1169390319`)
- Stockée dans `shared_matches.medals_earned`
- Aucune logique de détection actuellement

**Solution (conception)** :
1. **Requête** : Pour un match, vérifier si au moins un joueur ennemi a la médaille "À table" (`medal_name_id = 1169390319`)
2. **Interprétation** :
   - Si nos ennemis l'ont → **victoire écrasante** pour nous
   - Si nos alliés l'ont → **défaite humiliante** (peu probable si on perd, plutôt victoire écrasante)
3. **Affichage** : Badge visuel sur le match (icône 🍽️ ou similaire)

**❓ Question** : Tu confirmes que "À table" est donnée à toute l'équipe quand un des joueurs atteint le critère ? Ou seulement au joueur individuel ? Ça change la requête.
**❓ Réponse** : Oui mais à vérifier non plus.

**Remarque** : C'est une **feature nouvelle**, pas un bugfix. À planifier séparément.

---

### 9. Clic gamertag → ouvre Explorer + matchs session

**✅ Compréhension** : Quand on clique sur un gamertag (ex: fadetonull), l'Explorer s'ouvre et liste tous les matchs de la soirée (session) au lieu de filtrer sur les matchs en commun avec ce joueur.

**Analyse** :
- Le clic gamertag génère un lien vers Explorer avec `?gamertag=XXX`
- `explorer.py` reçoit le gamertag, lance `render_player_results()` 
- Celui-ci charge `load_common_matches()` qui cherche les matchs en commun

**Hypothèses** :
1. `load_common_matches()` fonctionne mais le **rendu affiche quand même tous les matchs** en plus
2. Le DataFrame passé (`df`) est celui de toute la session, pas le résultat filtré
3. Le problème pourrait être dans `explorer_results.py` : le tableau principal montre `df` (session complète) et pas le sous-ensemble commun

**Solution** : Vérifier que `render_player_results()` utilise **uniquement** les matchs en commun pour le rendu, pas le `df` de la session.
**❓ Précision** : Bug arrivé en cliquant sur le gamertag de Fadetonull, pas réussi à reproduire le bug.

---

### 10. Durée totale incohérente

**✅ Compréhension** : La durée totale affichée en haut de l'app montre 1h37 alors que la session réelle était de ~2-3 heures.

**Analyse** :
- Fichier : `src/app/kpis_render.py` ligne 35
- La durée affichée est `compute_total_play_seconds(dff)` = **somme des `time_played_seconds`** de chaque match
- C'est le **temps de jeu effectif** (somme des durées de match), PAS la durée "mur" de la session

**Root cause** : La somme des durées de matchs exclut :
- Les temps de chargement entre matchs
- Les pauses lobby
- Les temps de recherche de partie

**Il existe** `compute_session_span_seconds()` dans `src/app/helpers.py` (lignes 113-153) qui calcule la vraie durée "mur" (premier match → fin dernier match). **Mais elle n'est pas utilisée pour le KPI.**

**Solution** :
Deux options selon l'intent :
1. **Afficher les deux** : "Temps de jeu : 1h37" + "Session : 2h45" (span)
2. **Remplacer** par la durée session (span) car c'est ce que l'utilisateur perçoit

**❓ Question** : Tu veux afficher le temps de jeu effectif, la durée totale de session, ou les deux ?
**❓ Réponse** : La durée totale de session (sans changer le label attaché)

---

### 11. Némésis / Souffre-douleur formulation

**✅ Compréhension** : Les cards "Némésis" et "Souffre-douleur" affichent "X morts" et "Tué Y fois" → pas clair en français.

**Analyse** :
- Fichier : `src/ui/i18n/pages/match_view.py` lignes 236-237
- Textes actuels :
  - `"mv_deaths_count"` → `"{prefix}{n} morts"` (combien de fois IL m'a tué)
  - `"mv_killed_count"` → `"{prefix}Tué {n} fois"` (combien de fois JE l'ai tué)
- Rendu dans `match_view_players_nemesis.py` lignes 299-300
- L'ambiguïté : "3 morts" → morts de qui ? "Tué 2 fois" → tué par qui ?

**Solution** :
Reformuler pour lever l'ambiguïté :
```python
# Némésis (celui qui m'a tué le plus)
"mv_deaths_count": {"fr": "{prefix}M'a tué {n} fois", "en": "{prefix}Killed me {n} times"},
"mv_killed_count": {"fr": "{prefix}Je l'ai tué {n} fois", "en": "{prefix}I killed them {n} times"},
```
Ou version plus courte :
```python
"mv_deaths_count": {"fr": "{prefix}{n}× tué par lui", "en": "{prefix}Killed by them {n}×"},
"mv_killed_count": {"fr": "{prefix}{n}× éliminé", "en": "{prefix}Eliminated {n}×"},
```

**❓ Question** : Quelle formulation préfères-tu ? La première est plus explicite, la seconde plus compacte.
**❓ Réponse** : On va garder quelque chose pour le nemesis, une seule ligne "T'a victimisé X fois" (eliminated en anglais) et pour souffre douleur "Tu l'as persécuté x fois" pour ce dernier on pourrait utiliser bully en anglais 

---

### 12. Anciens graphes cache (JGTM / chocoboflor)

**✅ Compréhension** : Pour les joueurs JGTM et chocoboflor, les anciens graphes s'affichent sur Séries → Progression alors qu'ils devraient avoir les nouvelles données.

**Analyse** :
- Cache Streamlit : `@st.cache_data`, `@st.cache_resource(ttl=3600)` dans `src/ui/cache_loaders.py`
- `@lru_cache(maxsize=8)` dans `src/utils/profiles.py`
- Les caches sont keyed par gamertag + paramètres de filtre

**Hypothèses** :
1. **Cache Streamlit** : `@st.cache_data` persiste entre les reruns et n'est invalidé que par changement de paramètres ou TTL expiration. Si les données ont changé en DB mais les params de filtre sont identiques → cache stale
2. **Cache navigateur** : Streamlit utilise du cache côté client aussi
3. **Vues matérialisées stales** : Les `mv_*` dans `stats.duckdb` ne sont peut-être pas recalculées après un nouveau sync

**Solution** :
1. **Vérifier les vues matérialisées** de ces joueurs : est-ce que `mv_player_matches` est à jour ?
2. **Forcer un refresh** : Ajouter un bouton "Rafraîchir les données" ou cliquer sur "Clear cache" dans Streamlit (menu → Clear cache)
3. **Root cause** : Si les vues mat ne sont pas recalculées automatiquement après sync, c'est le vrai problème → ajouter `REFRESH MATERIALIZED VIEW` dans le pipeline post-sync

**❓ Question** : Ça concerne quel type de graphe exactement sur "Séries → Progression" ? Et est-ce que ces joueurs ont été resync récemment ?
**❓ Réponse** : Tous les graphes, oui plusieurs sync hier même, problème qui persiste pour Chocoboflor

---

### 13. Opacité barres escouade + distinction morts

**✅ Compréhension** : 
1. Les barres sur la page escouade ont une opacité trop faible (`0.32`), nuit à la lisibilité
2. Les barres de kills et de morts ont la même forme → confusion "qui a la plus haute = qui est le meilleur" alors que pour les morts c'est l'inverse

**Analyse** :
- Fichier : `src/visualization/trio.py` ligne 113
- Opacité actuelle : `opacity=0.32` → quasi-transparent
- Pas de distinction visuelle entre barres de kills (bien) et barres de morts (mal)

**Solution** :
1. **Opacité** : Augmenter à `0.75-0.85` pour une lecture confortable
2. **Distinction kills vs morts** :
   - Option A : **Couleurs inversées** — kills en vert/bleu, morts en rouge/orange
   - Option B : **Hachures** — barres de morts avec pattern rayé (hatch)
   - Option C : **Icône + orientation** — barres de morts orientées vers le bas (negative y-axis)
   - Option D : **Bordure** — barres de morts avec bordure rouge distincte

**❓ Question** : Quelle distinction préfères-tu ? L'option C (orientation inversée) est la plus intuitive (haut = bon, bas = mauvais) mais change le layout. L'option A (couleurs) est la plus simple.
**❓ Réponse** : Hachurée légèrement en rouge.

---

### 14. Tooltips : retirer la date sur graphes escouade

**✅ Compréhension** : Les tooltips hover sur les graphes (au minimum escouade, potentiellement partout) affichent la date, ce qui encombre la lecture.

**Analyse** :
- Les tooltips sont générés par Plotly via `hovertemplate` ou `hoverinfo`
- Chaque chart peut avoir son propre template de hover

**Solution** :
1. **Audit** de tous les graphes de la page escouade : identifier ceux avec date dans le tooltip
2. Pour chaque graphe, modifier le `hovertemplate` pour exclure la date
3. Ne garder que les valeurs pertinentes (nom joueur, valeur statistique)

**Root cause** : Pas de convention centralisée pour les tooltips → chaque chart est configuré indépendamment.

**Proposition** : Créer un `HOVER_TEMPLATE_NO_DATE` centralisé, réutilisable.

---

### 15. Matrice d'Impact données fausses (finisseur manquant)

**✅ Compréhension** : Sur la dernière partie, Flo était finisseur (vérifié en direct) mais la Matrice d'Impact ne le montre pas.

**Analyse** :
- Ce bug est probablement **lié au bug #4** : la détection des événements d'impact ne filtre pas par équipe
- Si le "finisseur" réel est déterminé sur TOUS les joueurs (alliés + ennemis), un ennemi avec un kill tardif pourrait être identifié comme finisseur à la place de Flo
- Ou le `outcome` passé à `compute_single_match_impact()` pourrait être incorrect pour ce match

**Solution** : Même fix que le bug #4 — filtrer par équipe alliée. Avec ce correctif, les événements d'impact seront correctement attribués.

**Root cause** : Identique au bug #4 (pas de filtre d'équipe dans les événements d'impact).

**Précision** : Non la cause ne se situe pas sur le filtre par équipe, le finisseur est celui qui fait le frag de la victoire, il ne peut y en avoir qu'un seul

---

### 16. Adornment rang pas à jour

**✅ Compréhension** : L'icône de rang (adornment) d'un joueur (ex: Madina) n'est pas mise à jour avec le rang actuel.

**Analyse** :
- L'adornment est chargé depuis `career_progression.adornment_path` en DB
- Mis à jour uniquement lors du sync via `resolve_adornment_url()` dans `src/data/sync/_career_rank_api.py`
- Si le joueur n'a pas été synchro récemment, l'adornment affiche l'ancien rang

**Solution** :
1. **Immédiat** : Re-synchroniser le joueur (`python scripts/sync.py --delta --gamertag Madina`)
2. **Structurel** : Ajouter un check de fraîcheur : si le dernier sync date de plus de X heures, afficher un badge "données potentiellement obsolètes"
3. **Optionnel** : Forcer le refresh de l'adornment indépendamment du sync complet (API call léger)

**Root cause** : L'adornment est un snapshot du dernier sync, pas un élément dynamique.

---

### 17. Exclure bots du MVP/LVP scoreboard

**✅ Compréhension** : Les bots peuvent être élus MVP ou LVP sur le scoreboard du dernier match.

**Analyse** :
- Fichier : `src/ui/pages/match_view_scoreboard.py` lignes 187-245
- Fonction `_compute_mvp_lvp()` boucle sur **tous** les joueurs sans filtrer les bots
- Les bots ont des xuids commençant par `bid(` ou `BID(`

**Solution** :
Filtrer les bots **avant** l'appel à `_compute_mvp_lvp()` :
```python
from src.config import get_bot_name

human_players = [p for p in players if get_bot_name(str(p.get("xuid", ""))) is None]
mvp_xuid, lvp_xuid = _compute_mvp_lvp(human_players, extremes)
```

**Root cause** : Oubli de filtre lors de l'implémentation MVP/LVP.

---

### 18. Fadetonull introuvable dans Explorer

**✅ Compréhension** : Le gamertag "fadetonull" (ou "fadet0null" avec un zéro) est introuvable via la recherche Explorer.

**Analyse** :
- La recherche utilise `fuzzy_search_gamertags()` avec `difflib` (cutoff=0.4) + recherche substring
- Problème possible : le gamertag contient un "0" (zéro) mais l'utilisateur tape un "o" (lettre) ou inversement

**Hypothèses** :
1. **Le joueur n'est pas dans `xuid_aliases`** : S'il n'a jamais été dans un match avec un joueur synchro, son gamertag n'est pas indexé
2. **Confusion o/0** : L'algorithme fuzzy n'a pas assez de tolérance pour `o` ↔ `0`
3. **Le joueur n'est pas dans les matchs chargés** : Si les données ne sont pas synchro

**Solution** :
1. Vérifier si le gamertag existe dans `xuid_aliases` (DB partagée)
2. Si non → le joueur n'a simplement jamais croisé un profil synchro
3. Si oui → améliorer la recherche fuzzy pour traiter les substitutions `o`↔`0`, `l`↔`1`, `s`↔`5` (leetspeak)

**❓ Question** : C'est "fadetonull" ou "fadet0null" le gamertag exact ? Tu l'as vu dans un match ou c'est un joueur qui n'a jamais joué avec vous ?
**❓ Réponse** : Vu plusieurs fois lors de la session du 12, pas réussi à déterminer si c'est un 0 ou un O dans son noms

---

### 19. Explorer : clic Rechercher reste sur même match

**✅ Compréhension** : Quand on sélectionne une date dans Explorer et qu'on clique Rechercher (sans rien d'autre), ça reste sur le même match.

**Analyse** :
- Fichier : `src/ui/pages/explorer.py`
- Le selectbox des matchs (`key="_exp_match_pick"`) persiste sa valeur dans `session_state`
- Quand la date change, le DataFrame filtré change mais le **selectbox tente de restaurer l'ancienne valeur**
- Si l'ancien match n'est pas dans la nouvelle date → comportement indéfini

**Solution** :
Reset le selectbox quand la date change :
```python
# Quand la date change, supprimer la valeur persistée du selectbox
if date_changed:
    st.session_state.pop("_exp_match_pick", None)
```
Ou utiliser un `key` dynamique basé sur la date pour forcer le reset.

**Root cause** : Persistance Streamlit du `session_state` pour le selectbox → stale quand le filtre amont change.
**Précision** : Nouvelle recherche n'affiche effectivement pas de nouvelle liste de matchs quand on entre des critères de recherche et clique sur recherche.

---

### 20. Sessions : dropdowns absents, inutilisable

**✅ Compréhension** : Sur la page de comparaison de sessions, un message dit de "choisir deux sessions" mais les dropdowns pour les choisir ne s'affichent pas. La fonctionnalité est inutilisable.

**Analyse** :
- Fichier : `src/ui/pages/session_compare.py` lignes 143-180 et 506-515
- Les dropdowns ne s'affichent que si `session_labels` a ≥2 éléments
- Si la liste est vide ou a 1 seul élément → message d'avertissement sans dropdown

**Hypothèses** :
1. **Données de sessions absentes** : `sessions` table dans `stats.duckdb` pourrait être vide (pas calculée)
2. **Filtre trop restrictif** : Le filtrage par mode/amis élimine trop de sessions
3. **Bug de branche** : Le fix du "fallback par défaut" n'est peut-être pas mergé dans main
4. **`all_sessions_df` vide** après filtrage

**Solution** :
1. Vérifier que les sessions sont bien calculées (`sessions` table peuplée)
2. Si les sessions existent mais le filtre les élimine → afficher les dropdowns **avec toutes les sessions** et filtrer après
3. Vérifier le merge du correctif de fallback session

**❓ Question** : Est-ce que la page Sessions fonctionnait avant ? C'est un nouveau bug ou ça n'a jamais marché ?
**❓ Réponse** : Probablement dû au changement qui affiche le message de warning en question, fonctionnait avant mais pas sûr

---

### 21. Net score cumulé : retirer LUSR

**✅ Compréhension** : Sur le graphe "Net score cumulé sur sessions" dans la comparaison de sessions, le LUSR est affiché en overlay et ne devrait pas l'être.

**Analyse** :
- Fichier : `src/visualization/_perf_session.py` lignes 195-211
- La fonction `_add_rank_trace()` ajoute le LUSR/CSR comme trace secondaire (axe Y2)
- Appelé dans `session_compare.py` avec `rank_a` et `rank_b` non-null

**Solution** :
Passer `rank_a=None, rank_b=None` dans l'appel à `plot_cumulative_comparison()` dans le contexte "net score cumulé". Ou ajouter un paramètre `show_rank=False` à la fonction.

**Root cause** : Le LUSR est ajouté automatiquement quand les données existent, sans option pour le désactiver.

---

### 22. Stats par carte : sens du tableau

**✅ Compréhension** : Le tableau "Stats par carte" dans Victoires/Défaites n'est pas clair → à quoi sert-il ?

**Analyse** :
- Fichier : `src/ui/pages/win_loss.py` lignes 307+
- Ce tableau montre les **statistiques agrégées par carte** (map) :
  - Nom de la carte
  - Playlists jouées dessus
  - Modes joués dessus
  - Nombre de matchs
  - Win rate, loss rate, ratio global
  - Accuracy moyenne, performance moyenne

**C'est** : "Sur quelle carte tu performes le mieux/le pire"

**Solution** :
1. Ajouter une en-tête explicative : "Performance par carte — quelles maps te conviennent le mieux"
2. Ajouter un tri visuel (couleur ou barre) par win_rate pour rendre la lecture intuitive
3. Envisager un lien vers la page Explorer filtrée par carte

**Précision** : a Supprimer

---

### 23. Association média : mauvais fuseau horaire

**✅ Compréhension** : Les fichiers médias ne sont pas associés au bon match à cause d'un décalage de fuseau horaire.

**Analyse** :
- Fichier : `src/data/media_indexer_matchers.py`
- Les timestamps de match sont en **UTC naïf** (pas de timezone info)
- Les timestamps de capture média sont probablement en **heure locale** (la caméra/logiciel d'enregistrement utilise l'heure système)
- La comparaison directe UTC vs local crée un décalage de ±1-2h (CET/CEST vs UTC)

**Solution** :
Ce bug est lié au bug #26 (timezone global). Fix spécifique :
1. Lors de l'association média, convertir `capture_start_utc` en UTC réel (si c'est du local déguisé en UTC)
2. Ou convertir le `start_time` du match en timezone locale avant la comparaison
3. Utiliser `get_tz()` de `src/ui/tz.py` comme source de vérité

**Root cause** : Mélange de timestamps UTC et local sans conversion.

---

### 28. Escouade : étiquettes axe X → numéro match + carte

**✅ Compréhension** : Sur les graphes de la page escouade, l'axe X affiche la date des matchs (`dd/mm`). Il faudrait plutôt afficher le numéro du match dans la session (`#1` = le plus ancien, `#2`…) et, si Plotly le supporte, le nom de la carte en dessous séparé par un saut de ligne `<br>`.

**Analyse** :
- Trois fonctions de graphe sont concernées, toutes dans `src/visualization/` :
  - `plot_trio_metric()` dans `trio.py` ligne ~113 : `ticktext = aligned["start_time"].dt.strftime("%d/%m").fill_null("").to_list()`
  - `plot_multi_metric_bars_by_match()` dans `match_bars.py` ligne ~253 : idem avec `FMT_SHORT_DATETIME_FR`
  - `plot_metric_bars_by_match()` dans `match_bars.py` ligne ~80 : idem avec `FMT_TICK_DATETIME`
- Ces trois fonctions sont **partagées** (pas exclusivement escouade) → on ne peut pas changer leur comportement par défaut sans impacter d'autres contextes
- La colonne `map_name` est **disponible** dans le DataFrame `sub` passé à `teammates_charts.py` (confirmé via `streamlit_bridge.py` lignes 177 et 325)
- Plotly supporte `<br>` dans `ticktext` → `"#1<br>Aquarius"` affiche le numéro sur une ligne et la carte sur la suivante
- La **matrice d'impact** (`friends_impact_heatmap.py`) utilise déjà `[f"#{i + 1}" for i in range(n_matches)]` — modèle à suivre

**Solution** :
1. **Ajouter un paramètre optionnel** `match_labels: list[str] | None = None` aux trois fonctions de visualisation
2. **Si fourni**, utiliser `match_labels` comme `ticktext` à la place des dates
3. **Si absent** (None), conserver le comportement date actuel → compatibilité rétrograde garantie
4. **Dans `teammates_charts.py`**, construire les labels en amont depuis le DataFrame `sub` :
   ```python
   map_names = sub["map_name"].fill_null("?").to_list()
   match_labels = [f"#{i + 1}<br>{m}" for i, m in enumerate(map_names)]
   ```
5. **Matrice d'impact** (`friends_impact_heatmap.py`) : déjà en `#N`, ne pas modifier — mais enrichir le `hovertemplate` avec le nom de carte (colonnes trop étroites pour `<br>` en ticktext)

**Fichiers à modifier** :
- `src/visualization/trio.py` — `plot_trio_metric()`
- `src/visualization/match_bars.py` — `plot_multi_metric_bars_by_match()` + `plot_metric_bars_by_match()`
- `src/ui/pages/teammates_charts.py` — construction de `match_labels` + passage aux fonctions
- `src/visualization/friends_impact_heatmap.py` — hover uniquement (no ticktext change)

**Root cause** : Les fonctions de visualisation construisent leurs étiquettes X en interne depuis `start_time`, sans exposer de paramètre pour les surcharger.

---

### 24. Navigation clic gamertag → switch de DB joueur

**✅ Compréhension** : Quand on clique sur un gamertag dans Explorer (ex: un coéquipier), la page s'ouvre sur la base de données de Madina (joueur par défaut) au lieu de rester sur le contexte du joueur sélectionné.

**Analyse** :
- `waypoint_player` dans `session_state` détermine le joueur actif
- `db_path` est construit à partir de `waypoint_player`
- Quand on clique un gamertag, on ne veut PAS changer de joueur principal — on veut voir les stats de CE gamertag dans NOS matchs

**Le problème** :
- L'Explorer devrait montrer les matchs **en commun** avec le gamertag cliqué, depuis la perspective du joueur principal
- Le clic ne devrait pas modifier `waypoint_player`
- Le contexte joueur devrait être **préservé en session** quand on navigue entre pages

**Solution** :
1. **Ne pas modifier `waypoint_player`** quand on clique un gamertag dans Explorer → c'est une "vue" temporaire, pas un changement de joueur
2. Stocker le "target" dans un `session_state` dédié (`explorer_target_gamertag`) séparé du joueur principal
3. S'assurer que `db_path` reste celui du joueur principal lors de la navigation entre pages

**Root cause** : Confusion entre "changer de joueur principal" et "voir les stats d'un autre joueur dans mes données".

---

### 25. Victoires/Défaites par mode incomplet

**✅ Compréhension** : Le graphe "par mode" ne montre que Assassin alors que la session contenait aussi Base et Drapeau (Capture the Flag).

**Analyse** :
- Fichier : `src/ui/pages/win_loss.py` lignes 126-150
- Le graphe reçoit `dff` (DataFrame filtré) en entrée
- Si le filtrage en amont ne retient qu'un seul mode → le graphe n'affiche que ce mode
- Le filtrage est fait par la sidebar/session selector

**Hypothèses** :
1. **Filtre session trop restrictif** : Le `picked_session_labels` ne retient que les matchs d'un seul mode
2. **`mode_ui` mal peuplé** : Certains matchs n'ont pas le bon mode dans `mode_ui`
3. **Bug de colonne** : La colonne utilisée pour grouper par mode n'existe pas ou est null pour certains matchs

**Solution** :
1. Vérifier les données : pour la session du 12 mars, combien de matchs et quels modes sont dans le DataFrame AVANT filtrage
2. Si les modes sont corrects en base mais filtrés en UI → corriger le filtre
3. Ajouter un log de debug pour afficher les modes présents dans `dff` avant le rendu

**❓ Question** : Est-ce que c'est spécifique à la page escouade (filtrage par session escouade) ou à la page Victoires/Défaites générale ?
**❓ Réponse** : Pour le moment le problème se limite à la page Escouade

---

### 26. Fuseau horaire global — centralisation

**✅ Compréhension** : Problème récurrent de fuseau horaire sur toute l'app. La lecture du TZ est centralisée (`src/ui/tz.py`) mais l'application n'est pas systématique.

**Analyse** :
- `get_tz()` et `get_tz_name()` dans `src/ui/tz.py` sont la source de vérité
- **Mais** l'application de la conversion TZ est dispersée :
  - `explorer_logic.py` : compare des dates locales sans conversion
  - `media_indexer_matchers.py` : pas de conversion
  - Les hover Plotly : formatting dates sans TZ
  - Les filtres par date dans la sidebar : comparent `date` (local ?) vs `start_time` (UTC)

**Solution (architecture, root cause)** :
1. **Créer un helper centralisé** dans `src/ui/tz.py` :
   ```python
   def utc_to_local(dt: datetime) -> datetime:
       """Convertit un datetime UTC en timezone utilisateur."""
       return dt.replace(tzinfo=timezone.utc).astimezone(get_tz())
   
   def local_to_utc(dt: datetime) -> datetime:
       """Convertit un datetime local en UTC."""
       return dt.replace(tzinfo=get_tz()).astimezone(timezone.utc)
   ```
2. **Appliquer systématiquement** `utc_to_local()` à chaque endroit où un timestamp est affiché ou comparé
3. **Audit complet** : rechercher tous les `start_time`, `format_datetime`, `.strftime` et vérifier le TZ
4. **Convention** : documenter que "en DB = UTC naïf, en affichage = TZ utilisateur"

**Root cause** : Pas de convention systématique "stocker UTC / afficher local".

---

### 27. Tableau "Par période" : utilité ?

**❓ Clarification** : Le tableau "Par période" dans Victoires/Défaites — tu ne vois pas à quoi ça sert, ou tu trouves que ça devrait montrer autre chose ?

**Ce qu'il fait actuellement** :
- Buckets temporels (ex: semaine, mois)
- Win%, loss%, ratio, K/D/A par période
- Permet de voir la progression dans le temps

**Question** : Veux-tu le supprimer, le reformater, ou le déplacer ?
**Précision** : a Supprimer

---

## Priorisation suggérée

### 🔴 Critiques (à faire en premier)
1. **#3** — LUSR guard-rail (vérifier + recalculer si besoin)
2. **#4 + #15** — Filtre équipe dans les événements d'impact
3. **#20** — Sessions dropdowns inutilisables
4. **#26** — Centralisation timezone (root cause de #23, #10 partiellement)
5. **#24** — Navigation / switch DB joueur

### 🟡 Importants
6. **#5** — Ordre matrice d'impact
7. **#17** — Bots MVP/LVP
8. **#19** — Explorer selectbox persistant
9. **#13** — Opacité barres + distinction kills/morts
10. **#10** — Durée totale (à clarifier)
11. **#25** — Modes incomplets dans V/D
12. **#18** — Recherche fadetonull (leetspeak)
13. **#12** — Cache stale

### 🟢 Rapides
14. **#2** — Renommer "Parties sélectionnées"
15. **#11** — Reformulation Némésis
16. **#14** — Tooltips sans date
17. **#21** — Retirer LUSR du net score
18. **#1** — Ajouter Ravager Rebound
19. **#16** — Adornment (resync)
20. **#22** — Clarifier Stats par carte

### 📋 Feature / Clarification
21. **#8** — Médaille "À table"
22. **#6** — Valeurs perf >80 (à investiguer)
23. **#27** — Tableau Par période
24. **#7** — Ratio moyen F/D/A (à clarifier)

### 🟢 Rapides (suite)
25. **#28** — Étiquettes axe X escouade (#N + carte)
