# Inventaire exhaustif des graphiques et tableaux — LevelUp

> Généré le 2026-04-01. À mettre à jour lors de l'ajout/suppression de visuels.

## Chiffres clés

| Métrique | Valeur |
|----------|--------|
| Instances `st.plotly_chart` dans les pages | **~96** |
| Fonctions `plot_*/create_*` dans `src/visualization/` | **~80** |
| Fonctions `create_*` dans `src/ui/components/` | **9** |
| Pages Streamlit avec rendu graphique | **15** |
| Tableaux HTML personnalisés | **10** |
| Sections documentées (visuels distincts) | **~80** |
| Graphiques désactivés (`if False:`) | **2** (`plot_map_outcome_timeline` §2.6, `_render_ratio_by_map_section` win_loss) |

---

## Thème visuel commun

Tous les graphiques partagent :
- **Thème** : `plotly_dark` + fond `rgb(29, 35, 40)` (couleur Waypoint Halo)
- **Palette principale** (Halo + Okabe-Ito) :
  - `cyan` `#33D6FF` — kills
  - `red` `#FF4B4B` — deaths/losses
  - `green` `#00DC82` — victoires / performance positive
  - `amber` `#FFB703` — spree / avertissement
  - `violet` `#8B5CF6` — assists / neutre
  - Okabe-Ito accessibilité daltonisme : `#0072B2`, `#D55E00`, `#E69F00`, `#009E73`, `#56B4E9`, `#CC79A7`
- **Hover mode** : `x unified` (axe X partagé)
- **Légende** : horizontale en bas (`orientation: h, y: -0.22`)
- **Grille** : `rgba(255,255,255,0.07)` sur X et Y
- **Annotations extrêmes** : max doré `#FFD700` sur les axes ratio/performance

---

## 1. Page CAREER (Carrière / Rang)

**Fichiers** : `src/ui/pages/career.py`, `career_charts.py`, `career_lusr.py`, `career_top_matches_render.py`

### 1.1 Gauge : Progression vers le prochain rang

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` mode `gauge+number` |
| Valeur affichée | % de progression XP (0–100) |
| Titre | Nom du rang actuel (FR) |
| Sous-titre | `{current_xp} / {xp_for_next_rank} XP` |
| Plage | 0–100 % (4 zones couleur : rouge 0-25, orange 25-50, cyan 50-75, vert 75-100) |
| Barre | Couleur dynamique selon % (rouge → orange → cyan → vert) |
| Seuil | Trait blanc épais à la valeur courante |
| Hauteur | 280 px |
| **Source** | `src/ui/components/career_progress_circle.py::create_career_progress_gauge` |

### 1.2 Gauge : Progression vers le rang Héros (272)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` mode `gauge+number` |
| Valeur affichée | % de progression vers le rang 272 (XP total = 9 319 350) |
| Titre | "Héros" |
| Sous-titre | `{xp_cumulé} / 9 319 350 XP` |
| Barre | Même palette que §1.1 |
| **Source** | `src/ui/components/career_progress_circle.py::create_hero_progress_gauge` |

### 1.3 Timeline XP / Progression des rangs

| Propriété | Valeur |
|-----------|--------|
| Type | Multi-traces ligne + marqueurs (`go.Scatter`) |
| Axe X | Dates (chronologique) |
| Axe Y | XP cumulée |
| Trace 1 | XP réel — ligne cyan solide |
| Trace 2 | XP estimé pré-sync — violet pointillés |
| Trace 3 | Autres joueurs (multiples) — 6 couleurs distinctes, légende désactivée |
| Trace 4 | Projection Héros — orange tirets |
| Trace 5 | Projection optimiste — vert tirets-points |
| Ligne horizontale | Seuil XP Héros (9 319 350) |
| Hover | Date, XP, rang |
| **Source** | `src/ui/pages/career_charts.py` |

### 1.4 Timeline LUSR (Skill Rating Unique)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` ligne lissée |
| Axe X | Numéro de match (incrémental) |
| Axe Y | Score LUSR (elo TrueSkill) |
| Trace | Ligne bleue avec lissage EWMA |
| Marqueurs résultats | ▲ vert = Victoire, ▼ rouge = Défaite, ◆ gris = Autre (axe Y secondaire à 3 %) |
| **Source** | `src/ui/pages/career_lusr.py`, `src/visualization/_timeseries_progression.py::plot_lusr_timeseries` |

### 1.5 Tableau HTML : Top 10 meilleurs matchs

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML généré (`st.markdown` + CSS) |
| Colonnes | Rang, Date, Carte/Mode, K/D/A, Score perso, Durée, Résultat, K/D ratio, Badges |
| Badges | DOMINATION (vert), HUMILIATION (violet), REMONTADA (bleu), DÉBÂCLE (orange), CONTRE-REMONTADA (sarcelle) |
| Tri | Score personnel DESC |
| Lien | Clic sur match_id → Explorer |
| **Source** | `src/ui/pages/career_top_matches_render.py` |

### 1.6 Tableau HTML : Top 10 pires matchs

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML identique §1.5 |
| Colonnes | Idem |
| Tri | Score personnel ASC |

### 1.7 Tableau HTML : Encounters (adversaires récurrents)

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML généré (`st.markdown` unsafe) |
| Colonnes | Gamertag, Côté (allié/ennemi), N° rencontre, Win Rate %, K/D, Badges |
| Badges | Allié plus (+), Coriace, Noix dure — `os-sb-td--best` / `encounter-badge--amber` / `os-sb-td--worst` |
| Filtres | Période configurable (7j, 30j, 90j, Tout) |
| Légende badges | `build_badge_legend_html()` inline |
| **Source** | `src/ui/pages/career_encounters_render.py::render_encounters_section` |

---

## 2. Page WIN LOSS (Victoires & Défaites)

**Fichiers** : `src/ui/pages/win_loss.py`, `src/visualization/distributions_outcomes.py`, `src/visualization/maps_outcome.py`

### 2.1 Résultats dans le temps

| Propriété | Valeur |
|-----------|--------|
| Type | Barres relatives `barmode="relative"` (`go.Bar`) |
| Axe X | Buckets temporels auto (semaine / jour / heure selon nb matchs) |
| Axe Y | Nombre de matchs (positif = Victoires, négatif = Défaites) |
| Trace Victoires | `#00DC82` vert — barres positives |
| Trace Défaites | `#FF4B4B` rouge — barres négatives, `customdata` pour hover valeur absolue |
| Trace Égalités | `#8B5CF6` violet — si count > 0 |
| Trace DNF | `#8B5CF6` violet — si count > 0 |
| Hover | Bucket, compte par outcome |
| **Source** | `distributions_outcomes.py::plot_outcomes_over_time` |

### 2.2 Résultats par carte (ou mode)

| Propriété | Valeur |
|-----------|--------|
| Type | Barres empilées `barmode="stack"` horizontales |
| Axe Y | Nom des cartes / modes (top 20 max) |
| Axe X | Nombre de matchs |
| Traces | Victoires (vert), Défaites (rouge), Égalités (violet), DNF (violet) |
| Tri | Par total de matchs DESC (configurable : `total`, `wins`, `losses`) |
| Filtre | `min_matches=1` (configurable) |
| **Source** | `distributions_outcomes.py::plot_stacked_outcomes_by_category` |

### 2.3 Heatmap Win Rate jour × heure

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Heatmap` 2D |
| Axe X | Heures (00h–23h) |
| Axe Y | Jours de semaine (Lun–Dim), inversé (Lun en haut) |
| Couleur | Gradient rouge (0 %) → ambre (50 %) → vert (100 %) |
| Valeur Z | Win rate % par cellule |
| Texte | Nombre de matchs dans chaque cellule |
| Hover | Jour Heure, win rate %, nb matchs |
| Colorbar | `Win Rate` avec format `.0%` |
| Seuil minimum | Cellules < `min_matches=2` masquées (NaN) |
| **Source** | `distributions_outcomes.py::plot_win_ratio_heatmap` |

### 2.4 Matchs "Top" par semaine (Ranked)

| Propriété | Valeur |
|-----------|--------|
| Type | Barres groupées `go.Bar` + ligne `go.Scatter` |
| Axe X | Semaines (dates regroupées) |
| Axe Y | Nombre de matchs |
| Barre 1 | Matchs Top N (rang ≤ `top_n_ranks=1`) — couleur violet |
| Barre 2 | Total matchs — couleur cyan |
| Ligne | Ratio Top/Total lissé |
| Hover | Semaine, total, top count |
| **Source** | `distributions_outcomes.py::plot_matches_at_top_by_week` |

### 2.5 Lollipop chart : Win rate par carte

| Propriété | Valeur |
|-----------|--------|
| Type | Tiges `go.Scatter line` + cercles `go.Scatter markers` |
| Axe X | Win rate (0.0–1.0 → affiché en %) |
| Axe Y | Noms des cartes |
| Couleur cercle | Vert si ≥ 50 %, rouge sinon (ou gamme performance si `color_by_perf`) |
| Taille cercle | Proportionnelle au nombre de matchs (min 12, max 24 px) |
| Texte cercle | Win rate % ou "V"/"D" si 1 seul match |
| Tige | Ligne grise partant de x=0 |
| Hover | Carte, win rate, nb matchs |
| **Source** | `maps_outcome.py::plot_map_lollipop` |

### 2.6 Timeline des résultats par carte ⚠️ DÉSACTIVÉ

> **⚠️ Ce graphique est actuellement désactivé** (`if False:` dans `win_loss.py` et `teammates_map_charts.py`). Le code existe mais n'est pas rendu en production.

| Propriété | Valeur |
|-----------|--------|
| Type | Scatter plot (marqueurs) par carte × match index |
| Axe X | Index temporel du match sur la carte (0 = plus ancien) |
| Axe Y | Nom de la carte |
| Marqueurs petits | Matchs historiques (opacité 0.4) |
| Marqueurs grands | Matchs de la session courante (opacité 1.0 + bordure blanche 2px) |
| Couleur par outcome | Vert = Victoire, Rouge = Défaite, Violet = Autre |
| **Source** | `maps_outcome.py::plot_map_outcome_timeline` |

### 2.7 Bullet chart : Win rate vs historique par carte

| Propriété | Valeur |
|-----------|--------|
| Type | Barres horizontales `go.Scatter` (tiges) + `go.Scatter` (point session) + `go.Bar` (plage historique) |
| Axe Y | Noms des cartes |
| Axe X | Win rate (%) |
| Barre fond | Plage historique (gris clair, 30–70 % ou IQR) |
| Point session | Couleur verte/rouge selon ±5 % vs moyenne |
| Ligne médiane | Pointillés blancs à la moyenne historique |
| Delta | Badge `+X%` ou `-X%` affiché sur le point |
| **Source** | `maps_outcome.py::plot_map_winrate_bullet` |

### 2.8 Scatter : Performance session vs historique par carte

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` markers |
| Axe X | Cartes (catégorique) |
| Axe Y | Score de performance moyen |
| Point session | Grand (15px), couleur selon seuil performance |
| Point historique | Petit (8px), gris opacité 0.5 |
| Ligne | Connexion session ↔ historique |
| **Source** | `maps_outcome.py::plot_map_perf_vs_history` |

### 2.9 Séries de victoires / défaites (Streak chart)

| Propriété | Valeur |
|-----------|--------|
| Type | Barres `go.Bar` + ligne tendance |
| Axe X | Index de match (chronologique) |
| Axe Y | Longueur de la série (positif = victoires, négatif = défaites) |
| Barres montantes | Série de victoires (vert blur) |
| Barres descendantes | Série de défaites (rouge blur) |
| Ligne | Tendance lissée (EWMA sur les valeurs streak) |
| **Source** | `src/visualization/timeseries_combat.py::plot_streak_chart` |

### 2.10 Barres : Score personnel par match

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` + `go.Scatter` ligne lissée |
| Axe X | Numéro de match (`#N<br>Carte`) — catégorique, angle -45° |
| Axe Y | Valeur de la métrique (score personnel) |
| Barre | Couleur configurable (cyan par défaut), opacity 0.70 |
| Ligne lissée | `MA(10)` vert, fenêtre configurable |
| Hover | `hover_label=valeur` par barre |
| Hauteur | 320 px |
| **Source** | `src/visualization/match_bars.py::plot_metric_bars_by_match` |

### 2.11 Barres horizontales : Ratio K/D et Précision par carte

> Deux instances côte à côte dans la section « Rapport kills/défaites par carte ».

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales |
| Axe Y | Noms des cartes (`map_name`) |
| Axe X | Valeur de la métrique (`ratio_global` ou `accuracy_avg`) |
| Couleur barres | Cyan fixe, ou palette dynamique performance si `color_by_sign=True` |
| Hover | `map_name, value, matches` |
| Instance 1 | `metric="ratio_global"` — ratio K/D par carte |
| Instance 2 | `metric="accuracy_avg"` — précision moyenne par carte |
| **Source** | `src/visualization/maps.py::plot_map_comparison` |

---

## 3. Page TIMESERIES (Séries Temporelles)

**Fichiers** : `src/ui/pages/timeseries.py`, `_timeseries_distributions.py`, `_timeseries_weapons.py`, `timeseries_skill_rank.py`
**Onglets** : KDA | Cumul | Distribution | Avancé

### 3.1 Timeline K/D/A

| Propriété | Valeur |
|-----------|--------|
| Type | `make_subplots(secondary_y=True)` — Barres + ligne |
| Axe X | Numéro de match (`#N<br>NomCarte`) — catégorique, angle -45° |
| Axe Y gauche | Kills/Deaths (count) |
| Axe Y droit | Ratio K/D |
| Barre Kills | Cyan `#33D6FF`, largeur 0.42, opacity 0.85 |
| Barre Deaths | Rouge `#FF4B4B`, largeur 0.42, opacity 0.6 |
| Ligne Ratio | Vert, traçage secondaire Y |
| Hover | `kills=%{customdata[0]}, deaths=%{customdata[1]}, assists=%{customdata[2]}, accuracy=%{customdata[3]}, ratio=%{customdata[4]}` |
| Annotation | Max ratio → badge doré `#FFD700` |
| **Source** | `src/visualization/timeseries.py::plot_timeseries` |

### 3.2 Timeline Assistances

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` + `go.Scatter` ligne lissée |
| Axe X | Dates (format `FMT_TICK_DATETIME`) |
| Axe Y | Nombre d'assistances |
| Barre | Violet `#8B5CF6`, opacity 0.85 |
| Ligne lissée | `MA(10)` vert `#00DC82` |
| Hover | kills, deaths, assists, accuracy, ratio, carte, playlist, match_id |
| **Source** | `src/visualization/timeseries.py::plot_assists_timeseries` |

### 3.3 Stats par minute (KPM / DPM / APM)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` × 3 + `go.Scatter` × 3 (lignes MA) |
| Axe X | Numéro de match (catégorique) |
| Axe Y | Stats par minute (symétrique : DPM négatif = sous l'axe X) |
| Barre KPM | Cyan, valeurs positives |
| Barre DPM | Rouge opacity 0.4, valeurs **négatives** (hover affiche valeur absolue) |
| Barre APM | Violet opacity 0.7, valeurs positives |
| Lignes MA | Cyan (KPM), rouge pointillés (DPM abs), violet pointillés (APM) — fenêtre 10 |
| Ticks Y | Symétrisés en valeur absolue (helper `build_symmetric_abs_ticks`) |
| Hover DPM | Valeur absolue via `customdata[5]` |
| **Source** | `src/visualization/timeseries.py::plot_per_minute_timeseries` |

### 3.4 Distribution KDA (FDA / KDE)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter fill="tozeroy"` (KDE) + `go.Scatter` (rug plot) |
| Axe X | Valeur KDA ratio |
| Axe Y | Densité de probabilité |
| Courbe KDE | Cyan, remplissage `rgba(53,208,255,0.18)` — règle de Silverman |
| Rug plot | Points `line-ns-open` blanc 10px — 1 point par match |
| Ligne verticale 0 | Pointillés blancs 35% opacité |
| Médiane | Ligne tirets orange `#ffaa00` + annotation valeur |
| Hover KDE | Valeur KDA, densité |
| **Source** | `src/visualization/distributions.py::plot_kda_distribution` |

### 3.5 K/D cumulé avec CI 90 %

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter fill="toself"` (ruban CI) + `go.Scatter` markers + ligne |
| Axe X | Dates des matchs |
| Axe Y | K/D ratio |
| Ruban CI 90 % | `rgba(86,180,233,0.18)` bleu clair — se rétrécit avec le nombre de matchs |
| Points match | Gris `circle-open` opacity 0.5 — K/D par match individuel |
| Courbe cumulée | Bleue `#0072B2` width 3px + markers 7px |
| Ligne cible | Tirets à K/D=1.0 |
| Marqueurs résultats | ▲ vert / ▼ rouge / ◆ gris (overlay Y secondaire à 3 %) |
| **Source** | `src/visualization/_perf_progression.py::plot_cumulative_kd_with_ci` |

### 3.6 Net Score par heure cumulée

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` ligne + ruban IC |
| Axe X | Temps cumulé de jeu (heures) |
| Axe Y | Score personnel net par heure |
| Ruban | IC 90 % bleu clair |
| Marqueurs résultats | ▲ vert / ▼ rouge (overlay) |
| **Source** | `src/visualization/_perf_progression.py::plot_net_score_per_hour` |

### 3.7 EWMA K/D + tendance linéaire

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` ligne + `go.Scatter` régression |
| Axe X | Dates |
| Axe Y | EWMA K/D (alpha=0.2) |
| Courbe EWMA | Ligne lisse bleue |
| Ligne régression | Tirets orange (si R² significatif) + annotation R², p-value |
| Option | Affichage du R² et p-value en légende |
| **Source** | `src/visualization/_perf_progression.py::plot_ewma_kd` |

### 3.8 Score de performance par match

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` (couleur dynamique) + `go.Scatter` MA lissée |
| Axe X | Numéro de match (catégorique) |
| Axe Y | Score de performance (0–100, relatif à l'historique) |
| Couleur barres | Dynamique : vert (excellent), cyan (bien), ambre (moyen), orange (en dessous), rouge (mauvais) |
| MA lissée | Fenêtre 10, violet |
| Hover | `performance={y:.1f}, date={customdata[0]}` |
| **Source** | `src/visualization/_timeseries_progression.py::plot_performance_timeseries` |

### 3.9 Rang + Score personnel (dual-axis)

| Propriété | Valeur |
|-----------|--------|
| Type | `make_subplots(secondary_y=True)` — ligne + barres |
| Axe X | Numéro de match |
| Axe Y gauche | Rang (career rank, inversé) |
| Axe Y droit | Score personnel par match |
| Ligne rang | Bleue |
| Barres score | Couleur ambre |
| **Source** | `src/visualization/_timeseries_progression.py::plot_rank_score` |

### 3.10 Distribution : Premier event (kill/death)

| Propriété | Valeur |
|-----------|--------|
| Type | Double histogramme `go.Bar` |
| Axe X | Temps en secondes depuis le début du match |
| Axe Y | Fréquence |
| Histogramme 1 | Premier kill — cyan |
| Histogramme 2 | Première mort — rouge |
| Hover | Temps formaté MM:SS, fréquence |
| **Source** | `src/visualization/_distributions_advanced.py::plot_first_event_distribution` |

### 3.11 Corrélation entre stats (scatter)

> Section générique — voir §3.18 pour les 5 paires concrètes affichées dans l'onglet Corrélations.

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` markers + ligne régression |
| Axe X | Métrique A (configurable) |
| Axe Y | Métrique B (configurable) |
| Couleur | Outcome (vert=W, rouge=L, gris=autre) |
| Ligne | Régression linéaire simple + R² |
| **Source** | `src/visualization/distributions.py::plot_correlation_scatter` |

### 3.12 Durée de vie moyenne par match

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` + `go.Scatter` MA lissée |
| Axe X | Numéro de match (catégorique) |
| Axe Y | Secondes (label formaté MM:SS) |
| Barre | Vert `#00DC82`, opacity 0.85 |
| MA lissée | Cyan, fenêtre 10 |
| Hover | `deaths, time_played_seconds, match_id` |
| **Source** | `src/visualization/timeseries_combat.py::plot_average_life` |

### 3.13 Spree + Headshots + Perfect Kills

| Propriété | Valeur |
|-----------|--------|
| Type | `make_subplots(secondary_y=True)` — 3 barres groupées + ligne précision |
| Axe X | Numéro de match |
| Axe Y gauche | Count (Spree, Headshots, Perfect Kills) |
| Axe Y droit | Précision % |
| Barre Spree | Ambre `#FFB703`, largeur 0.42 |
| Barre Headshots | Rouge opacity 0.70, largeur 0.42 |
| Barre Perfect Kills | Vert opacity 0.65, largeur 0.28 |
| Ligne précision | Ligne secondaire sur axe Y droit |
| `barmode` | `group` |
| **Source** | `src/visualization/timeseries_combat.py::plot_spree_headshots_accuracy` |

### 3.14 Dégâts infligés vs subis

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` × 2 (groupées) |
| Axe X | Numéro de match |
| Axe Y | Dégâts |
| Barre Infligés | Vert opacity 0.85 |
| Barre Subis | Rouge opacity 0.7 |
| **Source** | `src/visualization/timeseries_combat.py::plot_damage_dealt_taken` |

### 3.15 Tirs / Précision

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` × 2 + `go.Scatter` ligne |
| Axe X | Numéro de match |
| Axe Y gauche | Tirs tirés / Tirs touchés (count) |
| Axe Y droit | Précision % |
| **Source** | `src/visualization/timeseries_combat.py::plot_shots_accuracy` |

### 3.16 Tableau : Top armes par kills

| Propriété | Valeur |
|-----------|--------|
| Type | Barres horizontales `go.Bar` |
| Axe Y | Noms d'armes (top 10) |
| Axe X | Total kills |
| Couleur | Cyan avec dégradé |
| Annotations | Headshot rate % à droite de chaque barre |
| Hover | Weapon name, kills, headshot_rate, accuracy |
| **Source** | `src/visualization/distributions.py::plot_top_weapons` |

### 3.17 Onglet Distribution : 6 histogrammes statistiques

> Layout : 3 lignes × 2 colonnes. Chaque histogramme : `go.Bar` (bins) + courbe KDE (`go.Scatter fill="tozeroy"`, Silverman). Affiché uniquement si ≥ 6 valeurs non nulles.

| Position | Métrique | Axe X | Couleur |
|----------|----------|--------|---------|
| Ligne 1 col 1 | Précision (%) | `accuracy` | Cyan `#33D6FF` |
| Ligne 1 col 2 | Kills par match | `kills` | Vert `#00DC82` |
| Ligne 2 col 1 | Durée de vie moy. (s) | `avg_life_seconds` | Ambre `#FFB703` |
| Ligne 2 col 2 | Score de performance | `performance_score` | Violet `#8B5CF6` |
| Ligne 3 col 1 | Score par minute | calculé via `TimeseriesService.compute_score_per_minute` | Ambre `#FFB703` |
| Ligne 3 col 2 | Win rate glissant (%) | calculé via `TimeseriesService.compute_rolling_win_rate` | Vert `#00DC82` |

**Source** : `src/ui/pages/_timeseries_distributions.py::render_distributions`

### 3.18 Onglet Corrélations : 5 scatter corrélation

> Chaque scatter : `plot_correlation_scatter` — couleur = outcome (vert=W, rouge=L, gris=autre), trendline si ≥ 6 points. Layout : 2 lignes × 2 colonnes + 1 pleine largeur.

| Axe X | Axe Y | Titre |
|--------|--------|-------|
| Durée de vie (s) | Kills | ts_lifespan_vs_kills |
| Précision (%) | KDA ratio | ts_accuracy_vs_kda |
| Durée de vie (s) | Morts | ts_lifespan_vs_deaths |
| Kills | Morts | ts_kills_vs_deaths |
| MMR équipe | MMR ennemi | ts_mmr_team_vs_enemy (pleine largeur) |

**Source** : `src/ui/pages/_timeseries_distributions.py::render_correlations`

### 3.19 Régression K/D — indicateurs compacts (onglet Avancé)

> Affiché sous §3.7 EWMA K/D, dans l'onglet Avancé. Conditionné à `cumul.has_enough_for_trend`.

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` × 3 (pente K/D, pente win rate, R²) |
| Hauteur | 160 px (compact) |
| Indicateur 1 | Pente K/D (slope) — delta positif = amélioration |
| Indicateur 2 | Pente win rate (%) |
| Indicateur 3 | R² — annotated ⚠ si < 0.3 (« tendance non significative ») |
| Hover | N/A (indicateurs) |
| **Source** | `src/visualization/_perf_session.py::plot_regression_trend` |

### 3.20 Barres : Kills par arme sur la période filtrée (Timeseries)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales |
| Axe Y | Noms d'armes (FR/EN via `resolve_weapon_display`) |
| Axe X | Total kills |
| Couleur | Cyan (palette Halo) |
| Données | Agrégées depuis `weapon_kills` (film) + grenade/mêlée API (post-correction) |
| Annotations | Headshot rate % à droite de chaque barre |
| Conditionnel | Affiché uniquement si données weapon_kills disponibles |
| **Source** | `src/ui/pages/_timeseries_weapons.py::render_weapon_kills_chart` |

---

## 4. Page TEAMMATES (Coéquipiers & Synergies)

**Fichiers** : `src/ui/pages/teammates.py`, `teammates_charts.py`, `teammates_impact.py`, `teammates_synergy.py`, `teammates_map_charts.py`, `teammates_weapons.py`, `_teammates_trio.py`

### 4.1 Timeline stat par coéquipier (vue 1 ami)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` + `go.Scatter` MA lissée |
| Axe X | Numéro de match commun (catégorique) |
| Axe Y | Valeur de la stat choisie (kills, assists, ratio, etc.) |
| Barre | Couleur par ami (palette Halo) |
| MA | Fenêtre 10, vert |
| **Source** | `src/ui/pages/teammates_charts.py` |

### 4.2 Graphique Headshots + Perfect Kills empilés (escouade)

| Propriété | Valeur |
|-----------|--------|
| Type | Barres empilées `go.Bar` par joueur |
| Axe X | Numéro de match |
| Axe Y | Count (Headshots + Perfect Kills) |
| Barres | 1 couleur par joueur (self, f1, f2, f3) |
| `barmode` | `stack` |
| **Source** | `src/visualization/teammates_hs_pk.py::plot_hs_pk_stacked` |

### 4.3 Trio Metric Chart (multi-joueurs)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` groupées par joueur ou `go.Scatter` (configurable) |
| Axe X | Numéro de match commun |
| Axe Y | Métrique choisie (kills, deaths, ratio, assists, accuracy, avg_life, perf) |
| Traces | Self + Friend1 + Friend2/3 (3–4 joueurs) |
| Couleurs | 1 par joueur (identiques dans tous les graphes trio) |
| `barmode` | `group` |
| **Source** | `src/visualization/trio.py::plot_trio_metric` |

### 4.4 Trio Kills/Deaths (vue séparée)

| Propriété | Valeur |
|-----------|--------|
| Type | Barres groupées doubles |
| Axe X | Numéro de match |
| Axe Y | Kills (positif) / Deaths (négatif) par joueur |
| **Source** | `src/visualization/trio.py::plot_trio_kills_deaths` |

### 4.5 Radar Synergie (A vs B)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatterpolar` |
| Axes radiaux | K/D Ratio (0-100), Win Rate (%), Accuracy (%), Kills/match, Assists/match |
| Trace Self | Coral/rouge, remplissage `toself` opacity 0.2 |
| Trace Ami | Bleue, remplissage `toself` opacity 0.2 |
| Trace Historique | Violet pointillés (optionnel) |
| Hover | Valeur par axe |
| **Source** | `src/ui/components/_radar_teammates.py::create_teammate_synergy_radar` |

### 4.6 Radar Tendance session (escouade)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatterpolar` |
| Axes radiaux | KDA, Win Rate, Perf Score, Accuracy — normalisés 0-100 |
| Traces | Escouade courante (vert) vs Historique (violet, pointillés) |
| **Source** | `src/ui/components/_radar_teammates.py::create_session_trend_radar` |

### 4.7 Heatmap d'impact coéquipiers (timeline matchs)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Heatmap` 2D (matchs × joueurs) + scatter events |
| Axe X | Index des matchs (chronologique, max 50) |
| Axe Y | Gamertags des coéquipiers (+ ligne "Résultat" du match) |
| Valeur Z | Encoding binaire de présence d'events (OR) |
| Couleurs fond | Vert=Victoire, Rouge=Défaite, Violet=Autre (rectangles de fond) |
| Events superposés | Emojis par type d'event : ⚡ Premier sang, 🎯 Finisseur, 💀 Boulet, 🐌 Touriste, 🪦 1ère victime, 🛡️ Héros silencieux, 🗡️ Faux-frère |
| Hover events | Type d'event, match_id, gamertag |
| **Source** | `src/visualization/friends_impact_heatmap.py::plot_friends_impact_heatmap` / `friends_impact_scatter.py::plot_friends_impact_scatter` |

### 4.8 Heatmap : Win rate ami × carte

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Heatmap` 2D |
| Axe X | Gamertags (amis) |
| Axe Y | Noms de cartes |
| Valeur Z | Win rate % (0.0–1.0) |
| Gradient | Rouge (0 %) → ambre (50 %) → vert (100 %) |
| Hover | Ami, Carte, Win rate, Nb matchs |
| **Source** | `src/visualization/_heatmap_squad.py::plot_squad_map_heatmap` |

### 4.9 Tableau récapitulatif coéquipiers (`st.dataframe`)

| Propriété | Valeur |
|-----------|--------|
| Type | `st.dataframe` Streamlit avec style |
| Colonnes | Gamertag, Matchs communs, Kills, Morts, Ratio, Win Rate %, Avg Life, Profil joueur |
| Tri | Par matchs communs DESC |
| **Source** | `src/ui/pages/teammates.py` |

### 4.10 Timeline de performance d'escouade par session

| Propriété | Valeur |
|-----------|--------|
| Type | `make_subplots(secondary_y=True)` — barres + 2 lignes |
| Axe X | Sessions (labels `bucket_label`, index numérique, tick tous les ~12 sessions) |
| Axe Y gauche | Performance d'escouade 0–100 |
| Axe Y droit | MMR moyen de l'équipe (si données disponibles) |
| Barres | Score de perf par session — couleur dynamique : vert (≥ excellent) → cyan → ambre → orange → rouge (< seuil) |
| Barre hover | `squad_perf`, `match_count`, `wins`, `losses` (si disponibles) |
| Ligne Win Rate | Tier secondaire Y gauche, vert `#10B981` pointillés, marqueurs diamond |
| Ligne MMR équipe | Axe Y droit, violet `#8B5CF6`, marqueurs 5px |
| `hovermode` | `x unified` |
| **Source** | `src/visualization/_squad_timeline.py::plot_squad_performance_timeline` (appelé depuis `teammates_map_charts.py`) |

### 4.11 Barres multi-métriques multi-joueurs par match (escouade)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` groupées par joueur (`ChartData` via `MatchSeries`) |
| Axe X | Positions entières normalisées 0..N-1 (labels `#N<br>Carte`) |
| Axe Y | Métrique choisie (kills, deaths, assists, accuracy, avg_life, perf, ratio, etc.) |
| Traces | 2–4 joueurs, 1 couleur par joueur fixes dans toute la page |
| Records | Lignes horizontales records historiques (global + par carte si `per_map_records` présent) |
| Downsampling | `ChartData.downsample(max_points=200)` — protection performance |
| `barmode` | `group` |
| **Source** | `src/visualization/match_bars.py::plot_multi_metric_bars_by_match` (via callback `plot_multi_metric_bars_fn`) |

### 4.12 Stats par minute escouade (barres catégorielles)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` catégorielles par métrique par minute (`ChartData(barmode="categorical")`) |
| Axe X | Noms de métriques catégoriels (KPM, DPM, APM, etc.) — pas de positions entières |
| Axe Y | Valeur par minute |
| Traces | 1 par joueur (`offsetgroup` par joueur) |
| Ghost bars | Records historiques affichés comme barres fantômes derrière (`_add_categorical_record_bars`) |
| **Source** | `src/ui/pages/_teammates_trio_helpers.py::_render_per_minute_stats` |

### 4.13 Radar Profil Participation escouade (6 axes)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatterpolar` |
| Axes radiaux | Objectifs, Combat, Support, Score, Impact, Survie (normalisés 0–1.1) |
| Traces | 1 par joueur de l'escouade, couleurs Halo distinctes |
| Remplissage | `toself`, opacity configurable (défaut 1.0) |
| Hover | Valeur normalisée + valeur brute en tooltip |
| **Source** | `src/ui/components/_radar_participation.py::create_participation_profile_radar` (depuis `teammates_synergy.py`) |

### 4.14 Barres : Armes de kill multi-joueurs (escouade)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales groupées par joueur |
| Axe Y | Noms d'armes (FR/EN via `_resolve_weapon_name`) |
| Axe X | Total kills par arme |
| Traces | 1 par joueur de l'escouade (couleurs Halo fixes par joueur) |
| `barmode` | `group` |
| Titre | `t("tm_weapon_kills_chart")` |
| **Source** | `src/ui/pages/teammates_weapons.py::render_weapon_kills_bar_chart` (depuis `_teammates_trio.py` et `teammates_views.py`) |

### 4.15 Timeline : Premier frag et première mort (escouade)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` multiligne |
| Axe X | Index de match commun (chronologique) |
| Axe Y | Temps en secondes (premier frag / première mort) |
| Traces | 2 par joueur : premier frag (solide) + première mort (pointillés) |
| Couleurs | 1 par joueur (identiques au reste de la vue escouade) |
| **Source** | `src/ui/pages/teammates_charts.py::render_first_events_chart` (depuis `_teammates_trio.py`) |

---

## 5. Page MATCH VIEW (Détail d'un match)

**Fichiers** : `src/ui/pages/match_view*.py`, `src/visualization/team_dominance_timeline.py`, `src/visualization/match_impact_timeline.py`, `src/visualization/_antagonist_*.py`

### 5.0 Expected vs Actual (Réel vs Attendu)

> Section toujours affichée en haut du Match View pour résumer la performance du joueur relativement à son historique.

**Partie 1 — KPI cards (indicateurs texte)** : 3 métriques (Kills, Deaths, Assists) affichées via `os_card()` — valeur réelle / valeur attendue (CSR/LUSR) + delta coloré.

**Partie 2 — Graphique barres groupées** :

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` × 3 traces + axe Y secondaire ratio |
| Axe X | Métriques : Kills, Deaths, Assists |
| Axe Y gauche | Valeur (count) |
| Axe Y droit | Ratio kills/deaths (`go.Bar` séparé) |
| Trace 1 "Réel" | Barres pleines — cyan kills, rouge deaths, violet assists |
| Trace 2 "Attendu" | Barres hachurées pattern `"/"` opacity 0.55 (null si pas de données CSR) |
| Trace 3 "Historique mode" | Barres `"x"` pattern vert/rouge/cyan opacity 0.45 — **affichée uniquement si** `match_count ≥ 10` pour le mode_category du match (`extract_mode_category`) |
| Hover | valeur réelle, valeur attendue, moyenne historique |
| `barmode` | `group` |
| **Source** | `src/ui/pages/match_view_charts.py::render_expected_vs_actual` |

### 5.1 Timeline dominance d'équipe (Tug-of-War)

| Propriété | Valeur |
|-----------|--------|
| Type | `make_subplots` 2 lignes — barres (haut) + scatter kill feed (bas) |
| Sous-graphe 1 axe X | Temps du match (secondes) |
| Sous-graphe 1 axe Y | Frags par tranche de 30s (tug-of-war) |
| Barre Mon équipe | `#009E73` vert Okabe-Ito, au dessus de 0 |
| Barre Équipe ennemie | `#D55E00` vermillon, au dessous de 0 |
| Sous-graphe 2 | Kill feed : scatter émojis par joueur (mon équipe au-dessus, ennemi en dessous) |
| Annotations séries | Séries ≥3 kills annotées avec gamertag + count (arrière-plan coloré) |
| Hover | Tranche temporelle, kills équipe, kills ennemis |
| **Source** | `src/visualization/team_dominance_timeline.py::plot_dominance_chart` |

### 5.2 Timeline kills/deaths cumulés (joueur principal)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` `lines+markers` × 2 |
| Axe X | Temps du match (ms) |
| Axe Y | Kills / Deaths cumulés |
| Courbe Kills | Bleu `#0072B2` Okabe-Ito, largeur 2.5px |
| Courbe Deaths | Vermillon `#D55E00` Okabe-Ito, pointillés, largeur 2.5px |
| Annotations impact | ⚡ Premier sang, 🎯 Finisseur, 🐌 Touriste, 🪦 1ère victime — avec décalage vertical anti-superposition |
| Couleur annotations | Bleu (events "moi"), orange Okabe (`#E69F00`) (events autres) |
| Hover | Temps formaté MM:SS, cumul |
| **Source** | `src/visualization/match_impact_timeline.py::plot_match_kill_death_timeline` |

### 5.3 Timeline tous joueurs (frags cumulés)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` multiligne |
| Axe X | Temps du match (ms) |
| Axe Y | Kills cumulés par joueur |
| Traces | 1 ligne par joueur (jusqu'à 8+), couleurs d'équipe |
| Joueur principal | Ligne épaissie + marqueurs |
| Légende | Gamertags |
| **Source** | `src/visualization/match_impact_timeline.py::plot_all_players_frags_timeline` |

### 5.4 Radar de participation (contribution au score)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatterpolar` |
| Axes radiaux | % kills, % assists, % objectifs, % véhicules, % pénalités (inversé) |
| Trace | Self (remplissage cyan opacity 0.3) |
| **Source** | `src/ui/components/_radar_participation.py::create_participation_radar` |

### 5.5 Pie : Répartition des kills par arme

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Pie` (donut, hole=0.35) |
| Tranches | Armes (nom affiché FR/EN) — max 8 armes + "Autres" |
| Valeurs | Kills par arme |
| Couleurs | Palette 8 couleurs distinctes Halo |
| Texte | `percent+value` |
| Hover | Arme, kills, % |
| **Source** | `src/ui/pages/match_view_weapon_kills.py` |

### 5.6 Barres : Kills par arme (vue alternative)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales |
| Axe Y | Noms d'armes |
| Axe X | Kills |
| Couleur | Cyan |
| **Source** | `src/visualization/distributions.py::plot_top_weapons` (appelé dans match_view) |

### 5.7 Heatmap Killer/Victim

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Heatmap` 2D |
| Axe X | Victimes (gamertags) |
| Axe Y | Tueurs (gamertags) |
| Valeur Z | Nombre de kills killer→victim |
| Gradient | 0 (transparent) → max (rouge) |
| Hover | Tueur, victime, count |
| **Source** | `src/visualization/_antagonist_kv.py::plot_killer_victim_heatmap` |

### 5.8 Barres K/D empilées par joueur

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` empilées ou groupées |
| Axe X | Gamertags (par équipe) |
| Axe Y | Kills (positif) / Deaths (négatif) |
| Couleurs | Vert kills / Rouge deaths |
| **Source** | `src/visualization/_antagonist_kv.py::plot_killer_victim_stacked_bars` |

### 5.9 Barres Top antagonistes

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales × 2 (kills et deaths) |
| Axe Y | Gamertags (top 5 antagonistes) |
| Axe X | Count (kills par le joueur / deaths causées) |
| Trace Kills | Vert — "j'ai tué X fois" |
| Trace Deaths | Rouge — "X m'a tué Y fois" |
| **Source** | `src/visualization/_antagonist_duels.py::plot_top_antagonists_bars` |

### 5.10 Historique duel (1v1 vs nemesis)

| Propriété | Valeur |
|-----------|--------|
| Type | Barres alternées + tableau synthèse |
| Axe X | Chronologie des matchs communs |
| Axe Y | Count kills/deaths dans le duel |
| Barres | Bleu = win duel, rouge = loss duel |
| Résumé | KD ratio cumulé vs cet adversaire |
| **Source** | `src/visualization/_antagonist_duels.py::plot_duel_history` |

### 5.11 Résumé Nemesis / Victime (indicateurs)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` delta |
| Valeur | K/D ratio vs cet adversaire |
| Delta | vs moyenne tous adversaires |
| **Source** | `src/visualization/_antagonist_duels.py::create_kd_indicator` + `plot_nemesis_victim_summary` |

### 5.12 Tableau HTML : Scoreboard

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML personnalisé (`st.markdown` unsafe) |
| Colonnes | Joueur, Rang, Score, K, D, A, KDA, Arme principale, Spree, Headshots, Perfect Kills, Tirs, Prec. %, Mêlée, Arme lourde, DMG infligés, DMG subis, Durée de vie moy. |
| Highlighting | Meilleur = vert, pire = rouge (par colonne numérique) |
| Exceptions | Morts et DMG subis = inversé (moins = mieux) |
| Non comparé | Joueur, Rang, Arme principale |
| MVP/LVP | Badge coloré sur la ligne |
| Expand | Clic → détail joueur (stats supplémentaires) |
| **Source** | `src/ui/pages/match_view_scoreboard.py` |

### 5.13 Pie / Barres : Participation au score

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Pie` (donut) |
| Tranches | Catégories : kills, assists, objectifs, véhicules, autres |
| Couleurs | Par catégorie (CATEGORY_COLORS : cyan, violet, orange, ambre, rouge) |
| Texte | `percent+value` points + % |
| Pénalités | Filtrées du pie (affichées séparément) |
| **Source** | `src/visualization/participation_charts.py::plot_participation_pie` |

### 5.14 Barres de participation par catégorie

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales empilées |
| Axe Y | Catégories de score |
| Axe X | Points |
| Couleurs | Par catégorie |
| **Source** | `src/visualization/participation_charts.py::plot_participation_bars` |

### 5.15 Participation par match (timeline)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` × N par catégorie (empilées) |
| Axe X | Numéro de match |
| Axe Y | Points par catégorie |
| **Source** | `src/visualization/participation_charts.py::plot_participation_by_match` |

### 5.16 Radar Profil Participation match (6 axes)

> Affiché dans l'onglet « Participation » du Match View. Supporte 1 joueur (vue solo) et N joueurs (vue comparaison avec l'équipe).

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatterpolar` |
| Axes radiaux | Objectifs, Combat, Support, Score, Impact, Survie — normalisés 0–1.1 |
| Vue solo | 1 trace (joueur principal) — remplissage opaque |
| Vue comparaison | N traces (joueurs de l'équipe) — remplissage semi-transparent |
| Hover | Valeur normalisée + valeur brute en tooltip |
| **Source** | `src/ui/components/_radar_participation.py::create_participation_profile_radar` (depuis `match_view_participation.py`) |

### 5.17 Bloc HTML : Rang CSR / LUSR du match

| Propriété | Valeur |
|-----------|--------|
| Type | Bloc HTML généré (`st.markdown` unsafe) |
| Contenu | Rang CSR (tier + sous-rang + points) ou LUSR après le match |
| Delta | +/- points vs match précédent, coloré vert/rouge |
| Contexte bot | Si `had_bot_teammate=True`, note contextuelle sous le delta |
| Conditionnel | Affiché uniquement si données CSR/LUSR disponibles pour ce match |
| **Source** | `src/ui/pages/match_view_rank.py::_build_match_rank_html` |

### 5.18 Tableau HTML : Encounters du match

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML généré (`st.markdown` unsafe) |
| Colonnes | Gamertag, Côté (allié/ennemi), N° rencontre, Win Rate %, Kills/Deaths |
| Badges | Allié plus (vert), Coriace (ambre), Noix dure (rouge) — calculés via `compute_encounter_badges` |
| Badge ordinal | `ème rencontre` grisé en sur-titre |
| Légende badges | `build_badge_legend_html()` inline sous le tableau |
| **Source** | `src/ui/pages/match_view_encounters.py::render_encounter_section` |

### 5.19 Panneau HTML : Détail joueur (scoreboard expandable)

> Déclenché par clic « Expand » sur une ligne du scoreboard. Panneau HTML collapsible construit dynamiquement.

| Propriété | Valeur |
|-----------|--------|
| Type | Bloc HTML généré (`st.markdown` unsafe) |
| Section Armes | Top armes par kills (nom arme + count + headshot %) |
| Section Médailles | Grille icônes médailles avec count |
| Section Citations | Liste citations gagnées dans ce match |
| Section Attendu | Comparatif réel vs. attendu (CSR / historique) |
| Section Antagoniste | Kills/deaths vs. adversaire principal |
| Badge joueur | 🎮 Données complètes vs 🔗 Données partagées seulement |
| **Source** | `src/ui/pages/match_view_scoreboard_detail.py::render_scoreboard_player_detail_html` |

---

## 6. Page SESSION COMPARE (Comparaison de sessions)

**Fichiers** : `src/ui/pages/session_compare*.py`, `_session_compare_viz.py`, `_session_compare_history.py`, `_session_compare_extra.py`

### 6.1 Radar comparaison A vs B

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatterpolar` |
| Axes radiaux | KDA (0-100), Win Rate (%), Accuracy (%), Kills/match, Assists/match — normalisés |
| Trace A | Coral `rgba(255,99,99,0.2)` remplissage, ligne pleine |
| Trace B | Bleu `rgba(99,99,255,0.2)` remplissage, ligne pleine |
| Trace Historique | Violet pointillés (optionnel) |
| Legend | "Session A", "Session B", "Historique" |
| **Source** | `src/ui/components/_radar_teammates.py::create_session_trend_radar` (réutilisé) |

### 6.2 Barres groupées de comparaison

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` groupées (dual-axis) |
| Axe X | Métriques (Kills/match, Deaths/match, KDA, Win Rate) |
| Axe Y gauche | Valeurs absolues |
| Axe Y droit | Win Rate % |
| Trace A | Coral |
| Trace B | Bleu |
| Trace Historique | Pattern hachuré violet (optionnel) |
| **Source** | `src/ui/pages/_session_compare_viz.py` |

### 6.3 Donut outcomes par session

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Pie` (donut, 2 figures côte à côte) |
| Tranches | Victoires (vert), Défaites (rouge), Égalités (violet), DNF (gris) |
| Valeurs | Count par outcome |
| Légende | Session A / Session B |
| **Source** | `src/visualization/distributions_outcomes.py` (appelé ×2) |

### 6.4 K/D temporel (A vs B)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` ligne × 2 |
| Axe X | Index match dans la session (1…N) |
| Axe Y | K/D ratio par match |
| Trace A | Coral, ligne pleine |
| Trace B | Bleu, ligne pointillée |
| **Source** | `src/ui/pages/session_compare_charts.py` |

### 6.5 Comparaison cumulée A vs B

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` aire × 2 |
| Axe X | Index match |
| Axe Y | K/D cumulé |
| Trace A | Remplissage coral opacity 0.3 |
| Trace B | Remplissage bleu opacity 0.3 |
| **Source** | `src/visualization/_perf_session.py::plot_cumulative_comparison` |

### 6.6 Indicateurs cumulés (métrique)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` delta × 4 |
| Métriques | Kills, Deaths, KDA, Win Rate |
| Delta | Session B − Session A |
| **Source** | `src/visualization/_perf_session.py::create_cumulative_metrics_indicator` |

### 6.7 Tendance de session (timeline performance)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` ligne |
| Axe X | Index match |
| Axe Y | Performance score (0-100) |
| Trace | Ligne colorée selon tendance |
| **Source** | `src/visualization/_perf_session.py::plot_session_trend` |

### 6.8 Tableau : Historique de session

| Propriété | Valeur |
|-----------|--------|
| Type | `st.dataframe` Streamlit |
| Colonnes | #, Date, Mode, Carte, K, D, A, Score, Résultat (emoji), Durée |
| Tri | Chronologique |
| **Source** | `src/ui/pages/_session_compare_history.py` |

### 6.9 Barres horizontales : Modes de jeu par session

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales groupées |
| Orientation | Horizontale (`orientation="h"`) |
| Axe Y | Noms des modes (extraits de `pair_name` via `_extract_mode`) |
| Axe X | Nombre de parties dans ce mode |
| Trace A | Coral `#E74C3C` — Session A |
| Trace B | Bleu `#3498DB` — Session B |
| `barmode` | `group` |
| Hauteur | Dynamique : max(180, N_modes × 48 px) |
| Note | Affiché uniquement si colonne `pair_name` disponible |
| **Source** | `src/ui/pages/_session_compare_extra.py::render_modes_breakdown` |

### 6.10 Barres horizontales : Profil de participation A vs B (6 axes)

> Remplace le radar superposé, plus lisible. Affiché conditionnel à la disponibilité de `personal_score_awards`.

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales groupées |
| Orientation | Horizontale (`orientation="h"`) |
| Axe Y | 6 axes : Objectifs, Combat, Support, Score, Impact, Survie |
| Axe X | Valeur normalisée (0–110 %, affiché en %) |
| Trace A | Coral `#E74C3C` — Session A |
| Trace B | Bleu `#3498DB` — Session B |
| `barmode` | `group` |
| Hauteur | 300 px |
| Conditionnel | `repo.has_personal_score_awards()` = True |
| **Source** | `src/ui/pages/session_compare_charts.py::render_participation_trend_section` → `_session_compare_viz.py::_build_participation_bar_chart` |

---

## 7. Page OBJECTIVE ANALYSIS (Analyse des objectifs)

**Fichiers** : `src/ui/pages/objective_analysis.py`, `src/visualization/objective_charts*.py`

### 7.1 Scatter : Score objectifs vs Kills

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Scatter` markers + ligne de régression |
| Axe X | Kills par match |
| Axe Y | Score objectifs (`categories: objective, mode`) par match |
| Marqueurs | Taille 12px, couleur orange objectifs, opacity 0.7 |
| Hover | Carte, date, match_id |
| Ligne tendance | Régression linéaire simple (couleur ambre) |
| **Source** | `src/visualization/objective_charts.py::plot_objective_vs_kills_scatter` |

### 7.2 Barres : Breakdown objectives par catégorie

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales empilées |
| Axe Y | Catégories de score (kills, assists, objectifs, véhicules, etc.) |
| Axe X | Points totaux |
| Couleurs | Par catégorie (OBJECTIVE_COLORS) |
| **Source** | `src/visualization/objective_charts.py::plot_objective_breakdown_bars` |

### 7.3 Gauge : Ratio objectifs

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` gauge |
| Valeur | % du score total venant des objectifs |
| Plage | 0–100 % |
| Seuils couleur | Rouge < 20 %, orange 20-40 %, cyan 40-60 %, vert > 60 % |
| Étiquette | Profil = Slayer / Polyvalent / Support |
| **Source** | `src/visualization/objective_charts.py::plot_objective_ratio_gauge` |

### 7.4 Barres : Top joueurs par score objectifs

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` horizontales |
| Axe Y | Gamertags |
| Axe X | Score objectif total |
| Annotations | Matches count sur chaque barre |
| **Source** | `src/visualization/objective_charts.py::plot_top_players_objective_bars` |

### 7.5 Timeline : Score objectifs dans le temps

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` + `go.Scatter` MA |
| Axe X | Numéro de match |
| Axe Y | Score objectifs par match |
| MA | Fenêtre 10 |
| **Source** | `src/visualization/objective_charts_extra.py::plot_objective_trend_over_time` |

### 7.6 Pie : Decomposition assists

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Pie` (donut) |
| Tranches | Types d'assists objectives (EMP, revive, objectif, etc.) |
| **Source** | `src/visualization/objective_charts_extra.py::plot_assist_breakdown_pie` |

### 7.7 Sunburst : Participation hiérarchique

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Sunburst` |
| Niveau 1 | Catégorie principale (kills, assists, obj) |
| Niveau 2 | Sous-catégorie (type d'award) |
| Valeur | Points |
| **Source** | `src/visualization/participation_charts_extra.py::plot_participation_sunburst` |

---

## 8. Page CITATIONS (Médailles & Citations)

**Fichiers** : `src/ui/pages/citations.py`, `src/visualization/distributions.py`

### 8.1 Barres : Distribution des médailles

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Bar` ou `go.Bar` horizontales |
| Axe X/Y | Noms des médailles |
| Axe Y/X | Count |
| Couleurs | Palette 8 couleurs Halo par type (kill, multi, objectif, etc.) |
| Hover | Médaille, count, description |
| **Source** | `src/visualization/distributions.py::plot_medals_distribution` |

### 8.2 Grille HTML médailles

| Propriété | Valeur |
|-----------|--------|
| Type | Grille HTML responsive (`st.markdown`) |
| Colonnes | Icône médaille, Nom, Count |
| Layout | 4–6 colonnes selon viewport |
| **Source** | `src/ui/pages/citations.py` |

---

## 9. Page EXPLORER (Moteur de recherche)

**Fichiers** : `src/ui/pages/explorer.py`, `explorer_results.py`, `match_table_html.py`
(note : refs Python périmées post-migration ; stack actuelle = `apps/web/src/features/explorer/`).

### 9.0 Bandeau de briefing (mode Matchs) — 2026-07

| Propriété | Valeur |
|-----------|--------|
| Type | Bandeau au-dessus du tableau : rangée socle 4 KPI + frise + modules conditionnels |
| Socle | Matchs+période, Taux de victoire+bilan, FDA agrégat, Perf. moyenne (deltas vs baseline) |
| Frise | `OutcomeSequenceTape` (résultats du scope, 60 max) |
| Modules | Dimensions (carte/mode/playlist top·flop + note palier), Tendance (sparkline `TimeseriesLineChart`), Pronostic (attendu vs réel + Δ classement, gaté capability ranked) |
| **Source** | `apps/web/src/features/explorer/ExplorerBriefingStrip.tsx` + `ExplorerBriefingModules.tsx` ; backend `match_history_service_briefing.go` (opt-in `include_briefing`) |

### 9.1 Tableau HTML : Résultats de recherche matchs

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML généré (`st.markdown` unsafe) |
| Colonnes | #, Date, Mode, Carte, K, D, A, Score, Résultat, Durée |
| Couleur ligne | Vert = Victoire, Rouge = Défaite, Bleu = Égalité, Gris = DNF |
| Lien | Clic → page Match View pour ce match |
| Tri | Chronologique inversé (plus récent en haut) |
| **Source** | `src/ui/pages/match_table_html.py` |

---

## 10. Page MEDIA LIBRARY (Bibliothèque médias)

**Fichiers** : `src/ui/pages/media_library*.py`

### 10.1 Heatmap temporelle des médias

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Heatmap` 2D (optionnel, si données suffisantes) |
| Axe X | Dates (chronologique) |
| Axe Y | Heure de la journée (0–23h) |
| Valeur Z | Count médias dans ce créneau |
| Gradient | 0 (transparent) → max (cyan) |
| **Source** | `src/ui/pages/media_library_temporal.py` |

---

## 13. Page MATCH HISTORY (Historique des matchs)

**Fichiers** : `src/ui/pages/match_history.py`, `match_table_html.py`

> `render_match_table_html` est un module **partagé** entre la page Match History et la page Explorer (§9.1). La table est identique dans les deux contextes.

### 13.1 Tableau HTML : Historique de matchs

| Propriété | Valeur |
|-----------|--------|
| Type | Tableau HTML généré (`st.markdown` unsafe) |
| Colonnes | #, Date, Mode, Carte, K, D, A, Score, Résultat (emoji), Durée |
| Couleur ligne | Vert = Victoire, Rouge = Défaite, Bleu = Égalité, Gris = DNF |
| Lien | Clic → page Match View pour ce match |
| Tri | Chronologique inversé (plus récent en haut) |
| Export | Bouton téléchargement CSV (`_render_csv_download`) |
| **Source** | `src/ui/pages/match_table_html.py::render_match_table_html` (partagé avec §9.1 Explorer) |

---

## 11. Composants réutilisables transversaux

### 11.1 Radar générique (`src/ui/components/radar_chart.py`)

| Fonction | Axes | Usage |
|----------|------|-------|
| `create_radar_chart` | Configurable (N axes, labels custom) | Base réutilisable |
| `create_stats_per_minute_radar` | KPM, DPM, APM — normalisés 0-100 | Vue per-minute compacte |
| `create_performance_radar` | Performance, KDA, Accuracy, Win Rate, Avg Life | Vue synthèse carrière |

### 11.2 Radar participation profil (`src/ui/components/_radar_participation.py`)

| Fonction | Axes | Usage |
|----------|------|-------|
| `create_participation_radar` | % kills, assists, obj, véhicules, pénalités | Match View rapide |
| `create_participation_profile_radar` | Gamertags × métriques | Comparaison joueurs |

### 11.3 Indicateur de performance (`src/visualization/participation_charts_extra.py`)

| Propriété | Valeur |
|-----------|--------|
| Type | `go.Indicator` delta |
| Valeur | Score total session |
| Delta | vs baseline historique |
| **Source** | `create_participation_indicator` |

---

## 12. Index par type de visualisation

### Graphiques temporels (ligne/barre par match)
| # | Nom | Page |
|---|-----|------|
| 3.1 | K/D/A timeline | Timeseries |
| 3.2 | Assists timeline | Timeseries |
| 3.3 | Stats/min timeline | Timeseries |
| 3.8 | Performance score | Timeseries |
| 3.9 | Rang + score | Timeseries |
| 2.10 | Score personnel par match | Win Loss |
| 4.11 | Stats multi-joueurs par match | Teammates |
| 4.12 | Stats/min escouade (catégoriel) | Teammates |
| 4.15 | Premier frag/mort escouade | Teammates |
| 1.3 | XP / Rang | Carrière |
| 1.4 | LUSR progression | Carrière |
| 2.9 | Séries V/D | Win Loss |
| 7.5 | Objectifs timeline | Objectives |

### Heatmaps 2D
| # | Nom | Axes | Page |
|---|-----|------|------|
| 2.3 | Win Rate jour×heure | 24h × 7j | Win Loss |
| 4.7 | Impact matchs×joueurs | matchs × gamertags | Teammates |
| 4.8 | Win rate ami×carte | amis × cartes | Teammates |
| 5.7 | Killer/Victim | killers × victims | Match View |
| 10.1 | Médias temporels | dates × heures | Media Library |

### Radar / Polaire
| # | Nom | Axes | Page |
|---|-----|------|------|
| 5.4 | Participation radar 4 axes | kills%, assists%, obj%, véhicules% | Match View |
| 5.16 | Profil participation 6 axes | Objectifs, Combat, Support, Score, Impact, Survie | Match View |
| 4.5 | Synergie coéquipiers | KDA, Win%, Accuracy, K/match, A/match | Teammates |
| 4.6 | Tendance session escouade | KDA, Win%, Perf, Accuracy | Teammates |
| 4.13 | Profil participation escouade 6 axes | Objectifs, Combat, Support, Score, Impact, Survie | Teammates |
| 6.1 | Comparaison sessions A vs B | idem | Session Compare |

### Distributions (KDE, histogramme, scatter)
| # | Nom | Page |
|---|-----|------|
| 3.4 | Distribution KDA (KDE + rug) | Timeseries |
| 3.10 | Premier event kill/death | Timeseries |
| 3.11 | Corrélation stats | Timeseries |
| 7.1 | Score obj vs Kills | Objectives |
| 4.7 | Impact scatter events | Teammates |

### Gauge / Indicateurs
| # | Nom | Page |
|---|-----|------|
| 1.1 | Progression rang XP | Carrière |
| 1.2 | Progression rang Héros | Carrière |
| 3.19 | Régression K/D (pente + R²) | Timeseries |
| 7.3 | Ratio objectifs | Objectives |
| 5.11 | K/D vs nemesis | Match View |
| 6.6 | Delta métriques A vs B | Session Compare |

### Barres par carte / mode
| # | Nom | Page |
|---|-----|------|
| 2.11 | Ratio K/D + Précision par carte | Win Loss |
| 2.5 | Win rate par carte (lollipop) | Win Loss |
| 2.7 | Win rate vs historique (bullet) | Win Loss |
| 2.2 | Résultats par carte (empilées) | Win Loss |
| 3.20 | Kills par arme (période filtrée) | Timeseries |
| 4.14 | Armes kill multi-joueurs | Teammates |
| 6.10 | Profil participation A vs B (6 axes) | Session Compare |

### Pie / Donut / Sunburst
| # | Nom | Page |
|---|-----|------|
| 5.5 | Kills par arme | Match View |
| 5.13 | Score par catégorie (pie) | Match View |
| 6.3 | Outcomes par session (donut) | Session Compare |
| 7.6 | Assists breakdown (donut) | Objectives |
| 7.7 | Participation sunburst | Objectives |
| 8.1 | Médailles distribution | Citations |

### Tableaux HTML personnalisés
| # | Nom | Page |
|---|-----|------|
| 5.12 | Scoreboard match | Match View |
| 5.17 | Rang CSR/LUSR match | Match View |
| 5.18 | Encounters du match | Match View |
| 5.19 | Détail joueur (expand scoreboard) | Match View |
| 1.5 | Top 10 meilleurs matchs | Carrière |
| 1.6 | Top 10 pires matchs | Carrière |
| 1.7 | Encounters adversaires récurrents | Carrière |
| 9.1 | Résultats Explorer | Explorer |
| 13.1 | Historique de matchs | Match History |
| 8.2 | Grille médailles | Citations |

---

## 14. Page SYNTHÈSE (Analyse du scope courant)

> Route : `/players/$playerSlug/synthesis`  
> Handler Go : `POST /api/v1/players/{player_slug}/pages/synthesis` → `SynthesisHandler.GetSynthesisPage`  
> Composant principal : `apps/web/src/features/synthesis/SynthesisPage.tsx`

### Vue d'ensemble des blocs

| Bloc | Composant | Description |
|------|-----------|-------------|
| Scope bar (Bloc 0) | `ScopeBar` | Barre de contexte : période, matchs, filtres actifs/ignorés |
| Vue d'ensemble (Bloc 1 / D4) | `SynthesisOverviewSection` | Cartes KPI cumulatifs : victoires, défaites, kills, K/D, win rate, meilleur match, série |
| KPIs Solo / Escouade (Bloc 2) | `KPISection` ×2 | Grilles 2×4 cartes (win rate, K/D, matchs, perf. score) |
| Graphique bipolaire (Bloc 3) | `buildBipolaireChart` / `PlotlyChart` | Barres horizontales Solo ← → Escouade |
| Comparaison détaillée (Bloc 4) | `MetricRow` | Tableau HTML des métriques comparatives |
| Highlights (Bloc 5 / D5) | `SynthesisHighlightsSection` | Meilleurs et pires matchs par kills/KDA/morts |
| Heatmap activité (Bloc 6) | `buildHeatmapChart` / `PlotlyChart` | Heatmap 7×24 (jour × heure) |
| Top semaines (Bloc 7) | `TopWeekRow` | Tableau HTML des meilleures semaines |
| Relations (Bloc 8 / D6) | `SynthesisRelationsPreview` | Coéquipiers top + adversaires récurrents |
| Breakdowns (Bloc 9 / D7) | `SynthesisBreakdownsSection` | Tables carte et mode (matchs + win%) |

---

### 14.1 Barre de scope (`ScopeBar`)

**Type** : Composant React HTML (pas de Plotly)  
**Source données** : `SynthesisScope` (champ `scope` de la réponse API)

**Champs affichés** :
- Période (label traduit via `PERIOD_OPTIONS`)
- Nombre de matchs (`scope.match_count`)
- Filtres appliqués (`scope.filters_applied`) si non vide
- Filtres ignorés (`scope.filters_ignored`) avec icône ⚠ en `text-amber-500`

**data-testid** : `scope-bar`

---

### 14.2 Vue d'ensemble (D4 — `SynthesisOverviewSection`)

**Type** : Grille de cartes KPI (pas de Plotly)  
**Source données** : `SynthesisOverview` (champ `overview`)

**Cartes** (jusqu'à 7) :
| Label | Champ source |
|-------|-------------|
| Victoires | `overview.total_wins` |
| Défaites | `overview.total_losses` |
| Kills | `overview.total_kills` |
| K/D moyen | `total_kills / total_deaths` (calculé) |
| Win Rate | `overview.win_rate * 100` en % |
| Meilleur match | `overview.best_kills_match` (si non null) |
| Série max | `overview.longest_win_streak` (si > 1) |

---

### 14.3 Graphique bipolaire Solo ← → Escouade

**Type** : `go.Bar` horizontal, `barmode="overlay"` (Plotly)  
**Source données** : `data.comparison_metrics` → `buildBipolaireChart(metrics)`

**Structure** :
- **Axe Y** : labels de métriques (K/D, Win Rate, Accuracy, K/min, Avg Life, Perf Score) — inversé (`autorange='reversed'`)
- **Axe X** : normalisé −100 à +100, ticks masqués, ligne verticale `x=0` (axis shape)
- **Bars Solo** : orientation `h`, x négatif (cyan `#33D6FF`/`#06B6D4`), `textposition='outside'`
- **Bars Escouade** : orientation `h`, x positif (vert `#22C55E`/`#00DC82`), `textposition='outside'`
- **Hauteur** : `max(320, 70 × nb_metrics)` px
- **Transparence** : `plot_bgcolor` et `paper_bgcolor` = `rgba(0,0,0,0)` (thème sombre)

**Affichage conditionnel** : si `comparison_metrics.length === 0`, affiche une carte vide "Pas assez de données".

---

### 14.4 Heatmap activité jour × heure

**Type** : `go.Heatmap`, colorscale `Blues` (Plotly)  
**Source données** : `data.heatmap_data` → `buildHeatmapChart(cells: HeatmapCell[])`  
**Dimensions** : matrice 7 (jours) × 24 (heures), chaque cellule = nombre de matchs  
**Labels** : axe X = heures `0h`…`23h` ; axe Y = `['Lun','Mar','Mer','Jeu','Ven','Sam','Dim']`  
**Hauteur** : 220 px  
**Note** : différent du §2.3 (Win Rate heatmap, colorscale `RdYlGn`) — ici c'est la **fréquence** de jeu, pas les résultats

**Affichage conditionnel** : si `heatmap_data.length === 0`, affiche `EmptyStateNotice`.

---

### 14.5 Table Top semaines

**Type** : Tableau HTML (pas de Plotly)  
**Source données** : `data.top_weeks: TopWeekItem[]`

**Colonnes** :
| # | Semaine | Matchs | Win% | K/D |
|---|---------|--------|------|-----|
| rang | `week_label` (format `YYYY-Www`) | `match_count` | `win_rate * 100` % | `kd_ratio` |

**Affichage conditionnel** : si vide, affiche `EmptyStateNotice`.

---

### 14.6 Highlights D5 (`SynthesisHighlightsSection`)

**Type** : Composant React HTML (pas de Plotly)  
**Source données** : `data.highlights_preview: SynthesisHighlightsPreview`  
**Fichier** : `apps/web/src/features/synthesis/SynthesisHighlightsSection.tsx`

**Sections** :
- **Meilleur match** : `top_by_kills` (fallback `top_by_kda`) — lien vers `/players/${playerSlug}/matches/${match_id}`
- **Pire match** : `worst_by_deaths`
- Badge outcome (victoire/défaite/nul), K/D inline, perf score

---

### 14.7 Relations / Rivalités D6 (`SynthesisRelationsPreview`)

**Type** : Listes HTML (pas de Plotly)  
**Source données** : `data.rivalries_preview: SynthesisRivalriesPreview`  
**Fichier** : `apps/web/src/features/synthesis/SynthesisRelationsPreview.tsx`  
**Affichage conditionnel** : affiché seulement si `top_teammates.length > 0 || top_enemies.length > 0`

---

### 14.8 Répartition carte / mode D7 (`SynthesisBreakdownsSection`)

**Type** : Tables HTML en grille 1–2 colonnes (pas de Plotly)  
**Source données** : `data.breakdowns: SynthesisBreakdowns`

**Tables** :
- Par carte : `breakdowns.top_maps` → colonnes Carte / Matchs / Win%
- Par mode : `breakdowns.top_modes` → colonnes Mode / Matchs / Win%  
**Affichage conditionnel** : `return null` si les deux listes sont vides

---

## Annexe : Conventions de nommage des clés i18n

Les labels affichés dans les graphiques sont tous gérés via `src/ui/i18n/viz.py::viz_t(key, lang)`.
Exemples de clés utilisées :

| Clé | FR | EN |
|-----|----|----|
| `trace_kills` | "Kills" | "Kills" |
| `trace_deaths` | "Morts" | "Deaths" |
| `trace_ratio` | "Ratio K/D" | "K/D Ratio" |
| `trace_assists` | "Assistances" | "Assists" |
| `trace_wins` | "Victoires" | "Wins" |
| `trace_losses` | "Défaites" | "Losses" |
| `trace_ties` | "Égalités" | "Ties" |
| `trace_performance` | "Performance" | "Performance" |
| `trace_avg_smoothed` | "Moy. lissée (×10)" | "Smoothed Avg (×10)" |
| `axis_match_number` | "N° match" | "Match #" |
| `axis_kills_deaths` | "Kills / Morts" | "Kills / Deaths" |
| `axis_ratio` | "Ratio K/D" | "K/D Ratio" |
| `axis_per_minute` | "Stats / min" | "Stats / min" |
| `hover_win_rate` | "Win Rate" | "Win Rate" |
| `label_win` | "Victoire" | "Win" |
| `label_loss` | "Défaite" | "Loss" |
