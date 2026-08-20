# Plan v7 — Cockpit analytique

## TL;DR
Plan définitif de la refonte : on ne se contente plus de déplacer la sidebar, on repense l'app comme un cockpit analytique orienté intentions utilisateur. La navigation globale devient minimale, les sections `Stats` et `Escouade` deviennent de vrais hubs d'analyse, l'`Accueil` devient un centre de pilotage, et l'existant est réutilisé quand il reste pertinent, mais peut être recomposé ou remplacé par de nouvelles vues si cela améliore réellement la lecture et la prise de décision.

---

## Positionnement

Ce plan ne vise pas seulement :
- à conserver les mêmes fonctionnalités sans sidebar
- ni à réorganiser mécaniquement les pages existantes

Elle vise à produire une expérience plus cohérente :
- moins de friction de navigation
- plus de lecture contextuelle
- plus de hubs et moins de pages techniques
- même profondeur fonctionnelle que l'existant, mais exposée autrement

---

## Architecture cible

```
[🟣 Logo = clic → Accueil]

┌──────────────────────────────────────────────────────────────────────────────────────┐
│  Accueil   Stats   Escouade   Explorer   Médias   Profil      Sync   NS·Gamertag ▼  │  ← Shell global minimal
└──────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────┐
│  Scope : Session solo #12   ◀ Précédente   ⚡ Dernière   [Slayer ×] [Ranked ×]   Filtres   Réinitialiser  │  ← Bandeau de contexte (Stats / Escouade)
└──────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────┐
│  42 matchs · 38h    KD 1.84    Précision 47%    V 58% · D 32%                        │  ← Résumé KPI compact
└──────────────────────────────────────────────────────────────────────────────────────┘

         [workspace analytique : onglets, blocs, drill-down, vues inline]
```

**Couches d'interface :**

| Couche | Rôle | Présence |
|---|---|---|
| Shell global | Navigation, joueur actif, sync | Toutes sections |
| Bandeau de contexte | Scope d'analyse, filtres, navigation contextuelle | Stats / Escouade |
| Workspace analytique | Contenu, onglets, drill-down, insights | Toutes sections |

---

## Principes directeurs

1. **Une section = un domaine d'analyse**, pas un simple regroupement de pages héritées.
2. **Le shell n'explique pas**, il oriente ; l'analyse vit dans le contenu.
3. **Le drill-down doit être inline autant que possible** — éviter les ruptures de navigation.
4. **Tout ce qui existe n'a pas vocation à survivre tel quel** — on garde ce qui est bon, on recompose le reste.
5. **Créer une nouvelle vue est légitime** si elle répond mieux à une question utilisateur que trois graphes hérités.
6. **Aucune perte d'information** — la v7 peut changer la forme, pas la capacité d'exploration.
7. **Une métrique ou un graphe doit justifier sa place** — pas de conservation patrimoniale par défaut.

---

## Réponse explicite sur la stratégie de reprise de l'existant

**Non, je ne propose pas seulement une réorganisation cosmétique de l'existant.**

Je propose une hiérarchie de décision en trois niveaux :

1. **Réutiliser tel quel** quand un écran ou un graphe est déjà juste, lisible et bien compris.
2. **Recomposer** quand plusieurs éléments existants répondent à une même intention, mais sont aujourd'hui dispersés.
3. **Créer une nouvelle vue ou un nouveau graphe** quand l'existant est trop technique, trop fragmenté, ou ne raconte pas bien l'information.

**Règle de décision :**

- Si un composant existant répond déjà clairement à une question utilisateur → on le garde.
- Si la question utilisateur existe, mais que la réponse est éclatée sur plusieurs pages → on recompose.
- Si la question utilisateur n'a pas aujourd'hui de réponse lisible → on crée une nouvelle vue.

**Donc oui :** cette V7 alternative assume la création de nouvelles vues et potentiellement de nouveaux graphes, mais **seulement** lorsqu'ils améliorent la compréhension, pas pour “faire du neuf”.

---

## Mapping conceptuel — questions utilisateur

| Section | Question principale |
|---|---|
| Accueil | Qu'est-ce qui mérite mon attention maintenant ? |
| Stats | Comment je performe et est-ce que je progresse ? |
| Escouade | Avec qui je joue bien, et que vaut mon jeu en escouade ? |
| Explorer | Comment retrouver ou investiguer un match / contexte précis ? |
| Médias | Quels médias ai-je, à quoi sont-ils liés, et comment les exploiter ? |
| Profil | Quel est mon état de carrière, mes citations et mes réglages ? |

---

## Blueprint écran par écran

### Accueil — Mission Control

**Objectif :** transformer `last_match` en centre de pilotage, pas en simple page de détail.

**Blocs recommandés :**
- Hero du dernier match
- Résumé de la dernière session solo
- Résumé de la dernière session escouade
- Faits saillants récents
- Raccourcis vers `Stats`, `Escouade`, `Explorer`
- Derniers médias associés

**Ce qui peut être réutilisé :**
- `render_match_view()` pour le bloc dernier match
- la logique de navigation ◀▶ du dernier match
- les loaders déjà utilisés pour les matchs, médias et sessions

**Ce qui mérite d'être créé :**
- un bloc `faits saillants` synthétique
- un résumé de session plus narratif
- des cartes de raccourci orientées action

**Nouvelles vues/graphes potentiels :**
- mini timeline des 5 derniers matchs
- carte `variation récente` (KD, précision, WR par rapport à la session précédente)
- bloc `À revoir` : match marquant, baisse notable, session escouade récente

### Stats — Laboratoire personnel

**Objectif :** répondre à la progression personnelle, aux patterns de performance et à l'historique.

**Onglets recommandés :**
- Vue d'ensemble
- Performance
- Historique
- Sessions
- Matchs

**Vue d'ensemble**
- Résumé KPI compact
- 2 à 4 graphes majeurs maximum
- focus sur lisibilité et tendances

**Performance**
- timeseries utiles
- armes ou styles dominants si pertinent
- `render_performance_info` replacé ici

**Historique**
- tables ou graphes historiques
- transitions vers drill-down match

**Sessions**
- sessions récentes
- comparaison légère entre sessions

**Matchs**
- liste / table des matchs filtrés
- drill-down inline de `render_match_view()`

**Ce qui peut être réutilisé :**
- `timeseries.py` (fusionné avec `win_loss.py` en v6.4 — 5 onglets : Résumé, Cartes & Modes, Distributions, Progression, Avancé)
- `match_history.py`
- `session_compare.py`

**Ce qui mérite d'être recomposé :**
- un onglet `Vue d'ensemble` qui n'existe pas aujourd'hui comme tel
- une sélection plus stricte des graphes réellement utiles

**Nouvelles vues/graphes potentiels :**
- “forme récente” sur les 10 à 20 derniers matchs
- “stabilité” par session
- “distribution de performance” plutôt qu'une seule tendance moyenne

### Escouade — Analyse relationnelle

**Objectif :** sortir de la simple page coéquipiers et assumer une vraie analyse de synergie.

**Onglets recommandés :**
- Groupe
- Synergies
- Mes stats avec l'escouade
- Sessions d'escouade

**Groupe**
- vue coéquipiers existante
- picks fréquents
- lecture des combinaisons habituelles

**Synergies**
- complémentarités
- points forts / faibles selon la composition
- matrices ou radars si elles sont lisibles

**Mes stats avec l'escouade**
- graphes personnels restreints au scope squad
- même grammaire que `Stats`, mais autre contexte

**Sessions d'escouade**
- chronologie des sessions squad
- synthèse par roster

**Ce qui peut être réutilisé :**
- `teammates.py`
- `teammates_views.py`
- `teammates_synergy.py`
- `teammates_impact.py`

**Ce qui mérite d'être créé :**
- une vraie vue `Sessions d'escouade`
- une synthèse roster/session lisible en un coup d'œil

**Nouvelles vues/graphes potentiels :**
- matrice “avec qui ça gagne / perd” simplifiée
- timeline des sessions squad
- bloc “ma perf avec eux vs ma perf globale”

### Explorer — Section d'enquête

**Objectif :** transformer Explorer en outil d'investigation, pas juste en page de recherche.

**Sous-zones recommandées :**
- Recherche / critères
- Résultats
- Détail inline

**Comportements attendus :**
- recherche par match, map, mode, période, playlist
- facettes lisibles
- drill-down instantané vers détail match
- possibilité de partir d'un résultat vers Stats ou Médias si pertinent

**Ce qui peut être réutilisé :**
- `explorer.py`
- les logiques de chargement et de filtres déjà stables

**Ce qui mérite d'être amélioré :**
- meilleure hiérarchie entre filtres, résultats et détail
- résultats plus lisibles en mode tableau ou cartes légères

### Médias — Bibliothèque exploitée

**Objectif :** garder une section simple, mais mieux reliée au reste du produit.

**Blocs recommandés :**
- bibliothèque médias
- filtres propres médias
- associations média ↔ match
- aperçu rapide et lien vers détail match

**Ce qui peut être réutilisé :**
- `media_library.py`
- la logique d'indexation et d'association existante

**Ce qui mérite d'être créé si besoin :**
- vue “médias récents liés aux meilleurs / pires matchs”
- raccourci depuis Accueil vers les derniers médias pertinents

### Profil — Capitalisation et réglages

**Objectif :** conserver un espace calme, moins fréquent, orienté état et configuration.

**Onglets recommandés :**
- Carrière
- Citations
- Paramètres

**Paramètres**
- langue
- `Mon escouade`
- préférences UI persistées
- comportements système légers si utile

**Ce qui peut être réutilisé :**
- `career.py`
- `citations.py`
- `settings.py`

---

## Phases de mise en œuvre

### Phase A — Fondations shell & routing

1. Shell global minimal
2. Navigation primaire 6 sections
3. Compatibilité complète avec les deep links existants
4. Sidebar masquée mais non supprimée
5. Polish visuel léger des charts : typographie, fonds, gridlines, légendes, hover labels, sans changer la structure des graphes

### Phase B — Contextes analytiques

5. Deux `PageContext` indépendants
6. Bandeau de contexte pour `Stats` et `Escouade`
7. Résumé KPI compact
8. Réinitialisation claire des filtres

### Phase C — Hubs d'analyse

9. `Accueil` devient Mission Control
10. `Stats` devient hub personnel
11. `Escouade` devient hub relationnel
12. `Profil` devient hub de capitalisation

### Phase D — Vues nouvelles (liste fermée)

13. **D1 — Forme récente** (`Stats > Vue d'ensemble`) — sparklines 2 métriques sur 10–20 derniers matchs — données existantes dans `ctx_stats` — source : `src/analysis/_performance_form.py` (`avg_14`, `avg_90`, `form_score`)
14. **D2 — Stabilité par session** (`Stats > Sessions`) — bandes min/médiane/max par session — agrégats session sur `ctx_stats`
15. **D3 — Delta perf escouade vs global** (`Escouade > Mes stats`) — cartes KD/précision/V-D comparatives entre `ctx_squad` et agrégat global
16. **D4 — Résumé session solo/escouade** (`Accueil > Blocs A2/A3`) — volume + KPI + variation contextuelle — données `ctx_stats` / `ctx_squad` session courante
17. **D5 — Vue d'ensemble Stats** (`Stats > onglet Vue d'ensemble`) — assemblage de D1 + blocs existants, premier onglet de `stats_hub`

> **Règle :** aucune vue hors D1–D5 ne peut être créée en Phase D. Toute demande supplémentaire est différée en v8.

### Phase E — Suppression réelle de la sidebar

18. Retrait définitif seulement après parité fonctionnelle et validation sur sous-domaine

---

## Fichiers potentiellement impactés

| Fichier | Action probable |
|---|---|
| `streamlit_app.py` | Repenser le shell principal, les contextes et le dispatch global |
| `src/app/page_router.py` | Routing 6 sections + compatibilité legacy |
| `src/app/session_keys.py` | Clés de contexte par domaine et section active |
| `src/app/filters_render.py` | Découplage total des filtres du pattern sidebar |
| `src/ui/filter_state.py` | Persistance enrichie des contextes et de `Mon escouade` |
| `src/ui/pages/last_match.py` | Réutilisation partielle pour Accueil Mission Control |
| `src/ui/pages/stats_hub.py` | Nouveau hub |
| `src/ui/pages/squad_hub.py` | Nouveau hub |
| `src/ui/pages/profile_hub.py` | Nouveau hub |
| `src/ui/pages/home_mission_control.py` | Nouvelle page potentielle si `last_match.py` devient trop contraint |
| `src/ui/layout/header_l1.py` | Shell global |
| `src/ui/layout/header_l2.py` | Bandeau de contexte |
| `src/ui/layout/kpi_bar.py` | Résumé KPI |
| `src/ui/layout/filter_chips.py` | Filtres visibles |

---

## Décisions actées

- Ce plan est le plan définitif v7 — pas une variante alternative
- Nouvelles vues autorisées uniquement via la liste fermée D1–D5 (Phase D) — tout ajout hors liste est différé en v8
- L'objectif n'est pas de préserver la géographie actuelle des pages, mais la profondeur d'information
- `Accueil` devient un espace de pilotage, pas une simple page `Dernier match`
- Le rendu des charts est revu à la marge en Phase A : police, couleurs de fond, gridlines, légendes, hover labels et marges peuvent être harmonisés sans refonte des graphes eux-mêmes
- Accueil Mission Control : chargement progressif Option B — A1 (hero) + A5 (raccourcis) synchrones, A2 (session solo) + A3 (session escouade) en `@st.fragment`
- `Stats` et `Escouade` deviennent de vrais hubs, chacun avec sa logique propre
- Onglet `Synergies` = réorganisation d'existants (`render_synergy_radar`, `render_impact_taquinerie`, `render_squad_cadence_section`) — pas de création
- Onglet `Sessions d'escouade` = réorganisation d'existants (`render_squad_timeline`) + 1 seul nouveau composant (Q11 cartes de session sur `compute_squad_records()`)
- G5 Faits saillants = variation session N vs médiane 5 dernières sessions de même contexte (playlist + mode + solo/squad) — session éligible si ≥ 5 matchs
- Le drill-down inline est privilégié partout où il évite une rupture de navigation
- Les vues héritées sont des briques, pas des contraintes structurelles
- La qualité de lecture prime sur la fidélité littérale à l'existant

---

## Critère de succès

La v7 est réussie si :

- un utilisateur retrouve toutes les informations critiques déjà disponibles aujourd'hui
- il comprend plus vite où regarder selon sa question
- il navigue moins entre pages techniques
- il garde un sentiment de contrôle sur ses filtres, son scope et son joueur actif
- l'app paraît plus cohérente, plus moderne et plus agréable sans perdre en densité analytique

---

## Wireframes textuels détaillés — prêts à coder

> Convention de lecture :
> - `Bloc` = unité UI autonome
> - `Source` = existant / recomposé / nouveau
> - `Action` = ce que l'utilisateur peut faire
> - `Sortie` = effet attendu dans l'UI

### Accueil — Mission Control

```text
[L1 Shell global]

[Bloc A1 — Hero dernier match]
Map · mode · playlist · résultat · score · heure
CTA : Voir le détail
CTA : Match précédent / suivant

[Bloc A2 — Session solo récente]    [Bloc A3 — Session escouade récente]
Volume session                      Volume session
KD / Précision / V-D                KD / Précision / V-D
1 insight clé                       1 insight clé
CTA : Ouvrir Stats                  CTA : Ouvrir Escouade

[Bloc A4 — Faits saillants récents]
- meilleure perf récente
- baisse notable
- session escouade marquante

[Bloc A5 — Raccourcis]
Stats | Escouade | Explorer | Médias

[Bloc A6 — Derniers médias liés]
3 à 6 items max avec miniature, date, lien match
```

> **Stratégie de chargement (Option B)** : A1 (hero) + A5 (raccourcis) rendus de manière synchrone — le joueur voit le match immédiatement. A2 (session solo) + A3 (session escouade) rendus en `@st.fragment` — ils apparaissent quelques ms après sans bloquer le premier affichage. A4 suit les fragments. Les zones A2/A3/A4 réservent leur espace via un placeholder skeleton avant chargement — height fixe identique au contenu final pour éviter tout layout shift au-dessus de la ligne de flottaison.

#### Spécification des blocs

**Bloc A1 — Hero dernier match**

| Champ | Valeur |
|---|---|
| Source | Existant recomposé |
| Entrées | `render_match_view()` + données du dernier match + navigation déjà existante |
| Action | Ouvrir le détail du match, naviguer sur les derniers matchs |
| Sortie | Drill-down inline ou expander détaillé sous le hero |

**Bloc A2 — Session solo récente**

| Champ | Valeur |
|---|---|
| Source | Nouveau bloc, données existantes |
| Entrées | `ctx_stats`, résumé session courante ou dernière session solo |
| Action | Aller vers `Stats` avec le bon contexte |
| Sortie | Navigation vers `Stats > Vue d'ensemble` |

**Bloc A3 — Session escouade récente**

| Champ | Valeur |
|---|---|
| Source | Nouveau bloc, données existantes |
| Entrées | `ctx_squad`, résumé session escouade |
| Action | Aller vers `Escouade` avec le bon contexte |
| Sortie | Navigation vers `Escouade > Groupe` ou `Sessions d'escouade` |

**Bloc A4 — Faits saillants récents**

| Champ | Valeur |
|---|---|
| Source | Nouveau |
| Entrées | Variation session N vs médiane des 5 dernières sessions de même contexte (même playlist + mode + composition solo/squad) — éligible si session ≥ 5 matchs |
| Action | Cliquer un signal pour ouvrir le bon hub avec le contexte filtré |
| Sortie | 3 signaux maximum : delta KD · delta V% · delta Précision — affichés comme cartes delta cliquables |

**Bloc A5 — Raccourcis**

| Champ | Valeur |
|---|---|
| Source | Nouveau |
| Entrées | Liens internes |
| Action | Accès direct aux parcours principaux |
| Sortie | Navigation sectionnelle |

**Bloc A6 — Derniers médias liés**

| Champ | Valeur |
|---|---|
| Source | Existant recomposé |
| Entrées | `media_library` + associations médias/matchs |
| Action | Ouvrir un média ou le match lié |
| Sortie | Ouverture média / navigation vers détail match |

### Stats — Hub personnel

```text
[L1 Shell global]
[L2 Contexte Stats : scope · filtres · dernière / précédente · réinitialiser]
[Résumé KPI compact]

[Tabs]
Vue d'ensemble | Performance | Historique | Sessions | Matchs

[Vue d'ensemble]
[S1 — Forme récente]
[S2 — Répartition performance]
[S3 — Contextes dominants]
[S4 — Matchs marquants]

[Performance]
[S5 — Séries temporelles principales]
[S6 — Performance info]
[S7 — Armes / styles si retenus]

[Historique]
[S8 — Historique filtré]
[S9 — Win/Loss et tendances longues]

[Sessions]
[S10 — Liste sessions]
[S11 — Comparaison légère]
[S12 — Stabilité par session]

[Matchs]
[S13 — Table matchs]
[S14 — Détail match inline]
```

#### Spécification par onglet

**Vue d'ensemble**

| Bloc | Source | Rôle |
|---|---|---|
| S1 — Forme récente | Nouveau | Lire rapidement la dynamique courte |
| S2 — Répartition performance | Nouveau ou recomposé | Montrer la dispersion, pas seulement la moyenne |
| S3 — Contextes dominants | Recomposé | Révéler les contextes les plus fréquents / rentables |
| S4 — Matchs marquants | Nouveau bloc de synthèse | Pointer vers les extrêmes récents |

**Performance**

| Bloc | Source | Rôle |
|---|---|---|
| S5 — Séries temporelles principales | Existant | Observer l'évolution dans le temps |
| S6 — Performance info | Existant déplacé | Expliquer la métrique et ses limites |
| S7 — Armes / styles | Existant ou optionnel | Approfondir si la lecture reste claire |

**Historique**

| Bloc | Source | Rôle |
|---|---|---|
| S8 — Historique filtré | Existant recomposé | Lire la suite des matchs dans un contexte stable |
| S9 — Win/Loss et tendances longues | Existant recomposé — onglet `ts_tab_maps` de `timeseries.py` (fusion v6.4) | Conserver la lecture historique sans disperser la navigation |

**Sessions**

| Bloc | Source | Rôle |
|---|---|---|
| S10 — Liste sessions | Existant recomposé | Accéder vite à une session |
| S11 — Comparaison légère | Existant recomposé | Comparer 2 à 3 sessions max |
| S12 — Stabilité par session | Nouveau | Lire la variabilité de performance |

**Matchs**

| Bloc | Source | Rôle |
|---|---|---|
| S13 — Table matchs | Existant recomposé | Zone opérationnelle de recherche et tri |
| S14 — Détail match inline | Existant | Éviter la rupture de navigation |

### Escouade — Hub relationnel

```text
[L1 Shell global]
[L2 Contexte Escouade : scope squad · filtres · dernière / précédente · réinitialiser]
[Résumé KPI compact]

[Tabs]
Groupe | Synergies | Mes stats avec l'escouade | Sessions d'escouade

[Groupe]
[Q1 — Coéquipiers fréquents]
[Q2 — Compositions habituelles]
[Q3 — Résultats par roster]

[Synergies]
[Q4 — Matrice simple duo / trio]
[Q5 — Complémentarités]
[Q6 — Forces / faiblesses en escouade]

[Mes stats avec l'escouade]
[Q7 — Perf perso en scope squad]
[Q8 — Delta vs global]
[Q9 — Matchs squad marquants]

[Sessions d'escouade]
[Q10 — Timeline sessions squad]
[Q11 — Cartes de session]
[Q12 — Détail session / roster]
```

#### Spécification par onglet

**Groupe**

| Bloc | Source | Rôle |
|---|---|---|
| Q1 — Coéquipiers fréquents | Existant | Lire les partenaires principaux |
| Q2 — Compositions habituelles | Nouveau bloc de synthèse | Voir les combinaisons récurrentes |
| Q3 — Résultats par roster | Nouveau ou recomposé | Répondre à l'efficacité du groupe |

**Synergies**

| Bloc | Source | Rôle |
|---|---|---|
| Q4 — Matrice simple duo / trio | Existant réorganisé (`render_impact_taquinerie` — heatmap + badges) | Heatmap impact et badges d'events déjà construits |
| Q5 — Complémentarités | Existant réorganisé (`render_synergy_radar`, `render_trio_synergy_radar`) | Radars 6 axes (Objectifs/Combat/Support/Score/Impact/Survie) déjà implémentés |
| Q6 — Forces / faiblesses | Existant réorganisé (`render_squad_cadence_section`) | Profil de tempo synchronisé (kills/phase par joueur) déjà présent |

**Mes stats avec l'escouade**

| Bloc | Source | Rôle |
|---|---|---|
| Q7 — Perf perso en scope squad | Nouveau recomposé à partir d'existant | Rejouer la logique de `Stats` dans le contexte squad |
| Q8 — Delta vs global | Nouveau | Répondre à la vraie question comparative |
| Q9 — Matchs squad marquants | Nouveau bloc de synthèse | Pointer vers les matchs les plus révélateurs |

**Sessions d'escouade**

| Bloc | Source | Rôle |
|---|---|---|
| Q10 — Timeline sessions squad | Existant réorganisé (`render_squad_timeline`, `plot_squad_performance_timeline`) | Vue principale déjà construite — barres session + Win Rate + MMR équipe |
| Q11 — Cartes de session | Nouveau léger (sur `compute_squad_records()` existant) | Résumé par session : volume + roster dominant + issue — seul vrai nouveau composant |
| Q12 — Détail session / roster | Existant réorganisé (`_detect_trio_session`, `render_trio_view`) | Logique de détection session déjà présente |

### Explorer — Hub d'investigation

```text
[L1 Shell global]

[E1 — Barre de recherche et facettes]
match id | map | mode | playlist | période | issues | autres filtres

[E2 — Résultats]
table ou cartes compactes triables

[E3 — Détail inline]
match sélectionné

[E4 — Actions contextuelles]
ouvrir dans Stats | ouvrir dans Médias | copier match id
```

#### Spécification des blocs

| Bloc | Source | Rôle |
|---|---|---|
| E1 — Recherche et facettes | Existant amélioré | Réduire vite l'espace de recherche |
| E2 — Résultats | Existant recomposé | Zone centrale d'exploitation |
| E3 — Détail inline | Existant | Voir sans changer de page |
| E4 — Actions contextuelles | Nouveau | Rebondir vers d'autres hubs |

### Médias — Hub simple connecté

```text
[L1 Shell global]

[M1 — Filtres médias]
[M2 — Grille / liste médias]
[M3 — Métadonnées match liées]
[M4 — Preview + action ouvrir match]
```

#### Spécification des blocs

| Bloc | Source | Rôle |
|---|---|---|
| M1 — Filtres médias | Existant | Restreindre la bibliothèque |
| M2 — Grille / liste | Existant | Lecture principale |
| M3 — Métadonnées liées | Existant recomposé | Raccorder média et match |
| M4 — Preview + action | Existant amélioré | Ouvrir l'élément utile rapidement |

### Profil — Hub calme

```text
[L1 Shell global]

[Tabs]
Carrière | Citations | Paramètres

[Carrière]
[P1 — Progression carrière]
[P2 — Tops / pires matchs]

[Citations]
[P3 — Citations débloquées]
[P4 — Progression citations]

[Paramètres]
[P5 — Langue]
[P6 — Mon escouade]
[P7 — Préférences UI]
```

#### Spécification des blocs

| Bloc | Source | Rôle |
|---|---|---|
| P1 — Progression carrière | Existant | Lire la progression globale |
| P2 — Tops / pires matchs | Existant | Ajouter de la mémoire longue |
| P3 — Citations débloquées | Existant | État actuel |
| P4 — Progression citations | Existant ou recomposé | Vision cumulée |
| P5 — Langue | Existant déplacé | Paramètre global |
| P6 — Mon escouade | Nouveau | Configurer la logique squad |
| P7 — Préférences UI | Existant / futur | Persistance utilisateur |

---

## Nouveaux graphes réellement justifiés — priorisation

### Priorité haute

#### G1 — Forme récente

| Champ | Valeur |
|---|---|
| Section | `Stats > Vue d'ensemble` |
| Question | Suis-je dans une bonne ou mauvaise dynamique récente ? |
| Type | Sparkline multi-métriques ou mini courbes synchronisées |
| Source de données | Matchs récents filtrés du contexte `ctx_stats` |
| Pourquoi justifié | L'existant montre des tendances, mais pas une lecture courte et immédiate de la dynamique récente |
| Risque | Surcharge si plus de 3 métriques |
| Recommandation | 10 à 20 derniers matchs, 2 métriques max visibles, 1 variation textuelle |

#### G2 — Stabilité par session

| Champ | Valeur |
|---|---|
| Section | `Stats > Sessions` |
| Question | Est-ce que mes sessions sont homogènes ou très volatiles ? |
| Type | Boxplot simple, ou bandes min/médiane/max par session |
| Source de données | Agrégats session sur `ctx_stats` |
| Pourquoi justifié | La moyenne cache la dispersion ; cette vue apporte une information nouvelle et utile |
| Risque | Illisible si trop de sessions affichées |
| Recommandation | Limiter aux 8 à 12 dernières sessions |

#### G3 — Perf avec l'escouade vs globale

| Champ | Valeur |
|---|---|
| Section | `Escouade > Mes stats avec l'escouade` |
| Question | Est-ce que je joue mieux, moins bien ou pareil avec mon escouade ? |
| Type | Cartes delta ou small multiples comparatifs |
| Source de données | `ctx_squad` vs agrégat global / `ctx_stats` |
| Pourquoi justifié | C'est la question centrale de la section, absente aujourd'hui sous forme claire |
| Risque | Mauvaise lecture si trop de métriques simultanées |
| Recommandation | KD, précision, V/D, volume uniquement |

#### G4 — Timeline des sessions d'escouade

| Champ | Valeur |
|---|---|
| Section | `Escouade > Sessions d'escouade` |
| Question | Quand joue-t-on ensemble, avec quels rosters et quels résultats ? |
| Type | Timeline horizontale ou liste chronologique enrichie |
| Source de données | Sessions squad + roster configuré |
| Pourquoi justifié | Donne une lecture narrative inexistante aujourd'hui |
| Risque | Peut devenir une simple liste peu utile si non enrichie |
| Recommandation | Chaque nœud doit afficher volume, roster dominant, issue, accès détail |

#### G5 — Faits saillants récents (variation contextuelle)

| Champ | Valeur |
|---|---|
| Section | `Accueil > Bloc A4` |
| Question | Suis-je en progression ou en recul par rapport à mes sessions comparables ? |
| Type | Cartes delta — pas un graphe classique |
| Source de données | Session N vs médiane des 5 dernières sessions de même contexte (playlist + mode + composition solo/squad) |
| Éligibilité | Session ≥ 5 matchs ; si aucune session comparable disponible, bloc A4 masqué |
| Pourquoi justifié | La comparaison contextuelle évite le bruit d'une baseline inadéquate (ex. comparer Ranked Arena à du BTB) |
| Risque | Bruit si les sessions sont très courtes — éliminé par le garde-fou volume |
| Recommandation | 3 signaux max : delta KD · delta V% · delta Précision · tous cliquables vers Stats avec filtre contextuel prêt |

### Priorité moyenne

#### G6 — Distribution de performance

| Champ | Valeur |
|---|---|
| Section | `Stats > Performance` ou `Vue d'ensemble` |
| Question | Ma performance typique est-elle stable, asymétrique ou portée par quelques pics ? |
| Type | Histogramme ou densité simple |
| Pourquoi justifié | Apporte une lecture complémentaire à la moyenne |
| Recommandation | À n'ajouter que si la lecture reste immédiatement compréhensible |

#### G7 — Matrice simple avec qui ça gagne / perd

| Champ | Valeur |
|---|---|
| Section | `Escouade > Synergies` |
| Question | Quelles combinaisons semblent les plus efficaces ? |
| Type | Matrice compacte duo / trio |
| Pourquoi justifié | Bonne valeur analytique si simplifiée fortement |
| Recommandation | Trier par volume minimal, limiter les rosters rares |

#### G8 — Variation récente vs session précédente

| Champ | Valeur |
|---|---|
| Section | `Accueil` |
| Question | Suis-je en progression ou en recul immédiat ? |
| Type | Cartes delta |
| Pourquoi justifié | Très bon bloc d'atterrissage si le signal reste compact |
| Recommandation | Rester en bloc synthétique, pas en graphe lourd |

### Priorité basse ou non retenue au lancement

| Graphe | Décision | Raison |
|---|---|---|
| Nouveaux graphes d'armes complexes | Reporter | La profondeur arme existe déjà et n'est pas prioritaire pour le shell V7 |
| Radars supplémentaires | Éviter | Souvent séduisants mais peu lisibles |
| Graphes composites tout-en-un | Éviter | Forte dette cognitive, faible valeur produit |

### Recommandation de lancement

Lancer la V7 alternative avec **5 nouveautés fortes maximum** :

1. `G1 — Forme récente`
2. `G2 — Stabilité par session`
3. `G3 — Perf avec l'escouade vs globale`
4. `G4 — Timeline des sessions d'escouade`
5. `G5 — Faits saillants récents`

Cette sélection suffit pour donner une vraie personnalité produit sans disperser l'effort ni surcharger les hubs.
