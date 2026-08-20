# FUNCTIONAL_SPECS.md — Spécifications fonctionnelles V7

> **Ce document est le référentiel exhaustif de ce que chaque section V7 affiche, calcule et expose à l'utilisateur.**
> Il décrit le *quoi* (comportement visible) et non le *comment* (implémentation Streamlit).
>
> Chaque section se termine par un **checkpoint de lecture** (🔒). L'agent doit confirmer avoir lu
> chaque checkpoint avant de considérer la spec comme lue.

---

## Table des matières

| # | Section | Ancre |
|---|---------|-------|
| 0 | [Shell global & navigation](#0-shell-global--navigation) | `§0` |
| 1 | [Accueil (Home Mission Control)](#1-accueil-home-mission-control) | `§1` |
| 2 | [Stats](#2-stats) | `§2` |
| 3 | [Escouade](#3-escouade) | `§3` |
| 4 | [Synthèse](#4-synthèse) | `§4` |
| 5 | [Explorer](#5-explorer) | `§5` |
| 6 | [Médias](#6-médias) | `§6` |
| 7 | [Profil](#7-profil) | `§7` |
| 8 | [Settings](#8-settings) | `§8` |

---

## 0. Shell global & navigation

### 0.1 Architecture de la page

```
┌─────────────────────────────────────────────────────────┐
│ HEADER L1 (permanent — toutes les sections)             │
│  Logo "LevelUp" │ Tabs nav │ Player selector │ ⟳ │ ⚙   │
├─────────────────────────────────────────────────────────┤
│ HEADER L2 (contextuel — Stats & Escouade uniquement)    │
│  Titre section │ Mode Période/Sessions │ Scope │ ◀ ▶   │
│  Expander : filtres cascade (playlists, modes, maps)    │
│  Filter chips (résumé des filtres actifs)               │
├─────────────────────────────────────────────────────────┤
│ KPI BAR (secondaire — avant le contenu des sections)    │
│  Parties │ Durée │ KD │ Précision │ Win rate            │
├─────────────────────────────────────────────────────────┤
│ CONTENU SECTION (dispatch par section active)           │
└─────────────────────────────────────────────────────────┘
```

### 0.2 Header L1 — Navigation permanente

**Visible** : toujours, sur toutes les sections.

| Zone | Contenu | Interaction |
|------|---------|-------------|
| **Logo** | Texte "LevelUp" | Lien vers Accueil (`/`) |
| **Tabs** | 7 onglets : Accueil, Stats, Escouade, Synthèse, Explorer, Médias, Profil | Click → navigation URL (`/stats`, `/squad`, etc.) ; l'onglet actif a un style distinct |
| **Player selector** | Gamertag courant (texte si mono-joueur, dropdown si multi) | Click dropdown → change joueur via `?player=X` → rerun complet |
| **Sync indicator** | Point coloré (état sync) | Visuel uniquement |
| **Settings icon** | Icône ⚙ | Lien vers `/settings` |

**Routing URL** :

| Section | URL |
|---------|-----|
| Accueil | `/` |
| Stats | `/stats` |
| Escouade | `/squad` |
| Synthèse | `/synthesis` |
| Explorer | `/explorer` |
| Médias | `/media` |
| Profil | `/profile` |
| Settings | `/settings` |

**Deep links supportés** (query params) :
- `?player=Gamertag` → switch joueur
- `?page=X` → mapping legacy vers section V7
- `?match_id=Y` → ouvre l'Explorer sur ce match
- `?stats_view=timeseries|session_compare|match_history` → sous-vue Stats
- `?session=label` → scope session
- `?scope=squad|solo` → mode escouade/solo

### 0.3 Header L2 — Contexte & filtres (Stats et Escouade uniquement)

**Visible** : uniquement sur les sections `stats` et `squad`.

**Composants du L2** :

| Composant | Détail |
|-----------|--------|
| **Titre section** | "Stats" ou "Escouade" |
| **Caption dynamique** | "Mode Période / Toutes les sessions actives" (adapté au contexte) |
| **Segmented control Mode** | `Période` / `Sessions` — bascule le mode de filtrage global |
| **Scope selectbox** | Apparaît si mode = "Sessions" — liste les sessions détectées (solo/squad, triées par date desc) |
| **Boutons ◀ / ▶** | Navigation vers session précédente / dernière — désactivés si mode ≠ Sessions |
| **Expander filtres** | En mode "Période" : date range picker. En mode "Sessions" : filtres cascade (playlists, modes, maps) avec mode include/exclude |
| **Filter chips** | Résumé visuel des filtres actifs (max 3 items affichés + "+N" si davantage). Si aucun filtre : "Aucun filtre appliqué" |
| **Bouton Reset** | Supprime tous les filtres → rerun |

**Filtres cascade disponibles** :

| Dimension | Mode | Valeurs |
|-----------|------|---------|
| Playlists | include / exclude | Noms playlist distincts du scope |
| Modes | include / exclude | Noms de mode distincts du scope |
| Cartes | include / exclude | Noms de carte distincts du scope |

### 0.4 KPI Bar — Métriques globales

**Visible** : sur toutes les sections sauf Settings.

**5 métriques affichées** :

| Métrique | Format | Calcul |
|----------|--------|--------|
| Parties | entier | `count(match_id)` sur scope filtré |
| Durée | `Xj Yh Zm` | `sum(total_play_seconds)` formaté |
| KD | `X.XX` | `sum(kills) / sum(deaths)` global |
| Précision | `X%` | `mean(accuracy)` |
| Win rate | `X%` | `count(outcome=WIN) / count(*)` × 100 |

### 0.5 Contexte global (`PageContext`)

Objet central partagé entre toutes les sections. Contient :
- `df` : DataFrame Polars de **tous** les matchs du joueur
- `dff` : DataFrame Polars **filtré** (après application des filtres L2)
- `base` : Clone de `df` avant filtres (donnée de référence)
- `db_path`, `xuid` : Identité joueur + chemin DB
- `settings` : Configuration application (`AppSettings`)
- `waypoint_player` : Gamertag pour URLs Halo Waypoint
- `me_name` : Gamertag résolu du joueur principal
- `picked_session_labels` : Sessions sélectionnées (ou `None` si mode Période)
- `base_s_ui` : DataFrame des sessions groupées
- `match_view_params` : Paramètres pour les sous-vues match (callbacks, loaders)

> 🔒 **CHECKPOINT §0** — L'agent confirme avoir lu : architecture shell, L1 (7 tabs + player + sync + settings), L2 (mode/scope/filters/chips/reset), KPI bar (5 métriques), routing URL (8 sections), deep links (6 query params), PageContext (13 champs).

---

## 1. Accueil (Home Mission Control)

### 1.1 Objectif

Tableau de bord de synthèse — point d'entrée de l'application. Affiche l'état courant du joueur en un coup d'œil : forme récente, progression Battle Pass, défis actifs, dernière session escouade, activité récente et derniers médias.

### 1.2 Sources de données

| Source | Type | Contenu |
|--------|------|---------|
| `ctx.df` (Polars) | Local | Tous les matchs du joueur |
| `ctx.base_s_ui` (Polars) | Local | Sessions groupées (label, start_time, is_with_friends) |
| Battle Pass API | Live (SPNKr) | Opérations, reward tracks, rank, XP, tiers, images |
| Challenges API | Live (SPNKr) | Décks actifs, progression, XP disponible, badges |
| `media_files` table | DB joueur | Médias indexés (clips, captures) |
| `player_match_enrichment` | DB joueur | performance_score, session_id, is_with_friends |

### 1.3 Layout complet (ordre de rendu)

```
┌─────────────────────────────────────────────────────────────┐
│ ROW 0  │ Hero Card (70%)          │ Signaux Card (30%)      │
├─────────────────────────────────────────────────────────────┤
│ KPI BAR (global)                                            │
├─────────────────────────────────────────────────────────────┤
│ QUICK ACTIONS (4 cartes : Stats · Escouade · Explorer · Médias) │
├──────────────────┬──────────────────┬───────────────────────┤
│ ROW 1            │                  │                       │
│ Forme récente    │ Session Squad    │ Activité récente      │
│ (brief+stats)    │ (label+KPI+CTA)  │ (timeline 4 matchs)  │
├──────────────────┴──────────────────┴───────────────────────┤
│ ROW 2 (2 colonnes)                                          │
│ Battle Pass Card          │ Challenges Card                 │
├─────────────────────────────────────────────────────────────┤
│ ROW 3 : Dernier Match (scoreboard complet)                  │
├─────────────────────────────────────────────────────────────┤
│ ROW 4 : Médias Récents (3 items max)                        │
└─────────────────────────────────────────────────────────────┘
```

### 1.4 Composants détaillés

#### 1.4.1 Hero Card

| Élément | Contenu | Condition |
|---------|---------|-----------|
| Kicker | "Mission Control" (texte gris) | Toujours |
| Titre | Gamertag en H2 bold | Toujours |
| Sous-titre | Clé i18n `v7_home_hero_title` | Toujours |
| Brief dernier match | `"{outcome} · {map_name}"` (ex: "Victoire · Dereliction") | Si matchs récents ≥ 1 |
| Chips contexte | Chip 1: `"Dernier match: {outcome} {map}"` · Chip 2: `"Solo: {session_label}"` · Chip 3: `"Squad: {session_label}"` | Conditionnels (si données) |
| Stats ligne | `"KD {ratio}"` · `"ACC {accuracy}%"` · `"WR {win_rate}%"` · `"{total} parties"` | Si trend_snapshot existe |
| Trend ligne | `"KD {±delta:.2f} · ACC {±delta:.0f}% · WR {±delta:.0f}%"` (fenêtre 5 vs 5 matchs précédents) | Si calculable, sinon "N/A" |
| **Empty state** | "Aucune donnée disponible" | Si 0 matchs récents |

#### 1.4.2 Signaux Card (max 3 items)

| # | Signal | Titre i18n | Valeur | Détail |
|---|--------|-----------|--------|--------|
| 1 | Peak KD | `v7_home_highlight_peak` | `KD {meilleur_ratio}` (8 derniers matchs) | `{map} · {mode} · {date_heure_paris}` |
| 2 | Trend | `v7_home_highlight_trend` | `KD {±delta:.2f}` | `ACC {±delta:.0f}% · WR {±delta:.0f}%` |
| 3 | Squad meta | `v7_home_highlight_squad` | `{match_count} parties` | `{session_label} · WR {win_rate}%` |
| Fallback | Volume | `v7_home_highlight_volume` | `{count} parties` | `KD {ratio} · {durée_dhm}` |

Si < 3 signaux calculables, le fallback "Volume" complète la grille.

#### 1.4.3 Actions Rapides (4 cartes)

| Carte | Titre i18n | Description dynamique | CTA | Navigation cible |
|-------|-----------|----------------------|-----|------------------|
| **Stats** | `v7_nav_stats` | Si solo_summary: `"{label} · {count} parties"` ; sinon hint | `v7_home_open_section` | section=stats, view=timeseries, session=solo |
| **Escouade** | `page_teammates` | Si squad_summary: `"{label} · WR {wr}%"` ; sinon hint | `v7_home_open_section` | section=squad, session=squad |
| **Explorer** | `page_explorer` | Si matchs: titre du 1er match ; sinon hint | `v7_home_open_match` | section=explorer, match_id=1er match |
| **Médias** | `page_media` | Hint constante | `v7_home_open_section` | section=media |

#### 1.4.4 Forme Récente

- Brief du dernier match : `"{outcome} · {map} · {mode}"`
- Trend ligne : `"KD {±delta:.2f} · ACC {±delta:.0f}% · WR {±delta:.0f}%"`
- 3 stats en bloc : KD (X.XX), ACC (X%), WR (X%)
- **Empty state** : "Aucune donnée" si 0 matchs

#### 1.4.5 Session Squad Card

- **Si squad_summary existe** :
  - Label session : `"{label} · {HH:MM UTC}"`
  - 5 stats : parties, durée (dhm), KD, ACC%, WR%
  - Bouton CTA → navigate vers section squad avec session
- **Si absent** : texte `v7_home_no_recent_squad` + CTA vers squad sans session

#### 1.4.6 Activité Récente (Timeline)

4 derniers matchs (triés start_time DESC). Chaque item :
- **Pill outcome** : {VICTOIRE|DÉFAITE|ÉGALITÉ|DNF} + couleur (vert/rouge/gris/orange)
- **Titre** : `"{outcome} · {map}"`
- **Détail** : `"{mode} · KD {ratio:.2f} · {accuracy}%"`
- **Interaction** : Click → ouvre l'Explorer sur ce match
- **Empty state** : `v7_home_no_recent_activity`

#### 1.4.7 Battle Pass Card

| Sous-section | Contenu |
|--------------|---------|
| **Track hero** | Image de l'opération en cours (PNG base64, API CMS) |
| **Header** | Nom du track + badge "Premium" ou "Free" |
| **Rang courant** | `"Niv. {op_rank}"` |
| **Barre XP** | Si max atteint : `[████] MAX`. Sinon : `[███░░░] {current}/{total} XP` |
| **Browser de paliers** | 3–4 paliers autour du focus (index persisté session) ; chaque palier montre ses récompenses Free + Premium avec images/icônes |
| **Navigation ◀ / ▶** | Boutons Prev/Next pour scroller les paliers. Centre : `"Palier {n} / Max {max}"` |
| **Empty state** | "API indisponible" si tokens absents |

**Données live** : 7+ appels API (career rank, operations, track metadata CMS, item definitions ×N, reward images ×N, operation artwork).

#### 1.4.8 Challenges Card

| Sous-section | Contenu |
|--------------|---------|
| **Header** | Badge PNG du défi + nom + description |
| **Stats** | `"{completed}/{total} Complétés"` · `"{progress}/{target} En cours"` · `"+{xp:,} XP"` |
| **Expiration** | `"Expire le {date_courte} UTC"` |
| **Empty states** | API indisponible : "API indisponible pour le moment" · 0 défis : "Aucun défi actuellement" |

#### 1.4.9 Dernier Match (Row 3)

Affiche la vue complète du dernier match via `render_last_match_page()` — scoreboard, stats, détails complets. Voir [§5 Explorer — Match View](#54-match-view-détail-dun-match) pour la spec du rendu match.

#### 1.4.10 Médias Récents (Row 4)

- 3 derniers médias (clips/captures)
- Chaque item : nom fichier, date, match_id associé
- CTA en bas : bouton navigation → section Médias
- **Empty state** : `v7_home_no_recent_media`

> 🔒 **CHECKPOINT §1** — L'agent confirme avoir lu : Hero Card (7 éléments), Signaux (3+fallback), Quick Actions (4 cartes avec nav cibles), Forme récente, Session Squad (avec/sans données), Timeline (4 matchs, pills outcome), Battle Pass (track+rank+XP+browser paliers+nav), Challenges (badge+stats+expiry+2 empty states), Dernier Match (délégation complète), Médias Récents (3 items+CTA). Total : 10 composants, 6 appels API live, 4 empty states nommés.

---

## 2. Stats

### 2.1 Objectif

Section analytique du joueur. Regroupe 3 sous-vues accessibles via segmented control : **Séries** (évolutions temporelles), **Sessions** (comparaison de sessions), **Historique** (tableau des matchs).

### 2.2 Navigation interne

**Segmented control** en haut de la section :
- Options : `timeseries` | `session_compare` | `match_history`
- Labels i18n : "Séries" | "Comparaison sessions" | "Historique"
- State key : `SK.V7_STATS_VIEW`

Le L2 et la KPI Bar sont affichés **avant** le contenu de la sous-vue.

### 2.3 Sous-vue "Séries" (Timeseries)

**5 onglets (tabs)** :

#### Onglet 1 : "Résumé"

| Composant | Type | Détail |
|-----------|------|--------|
| **KPIs secondaires** | Métriques | Parties, Durée totale, KD moyen, Précision moyenne, Win rate |
| **Score de forme** | Plotly line chart | Rolling form_score (14 matchs) avec shaded area. KPI latéral : form_score moyen ± delta vs baseline. Si ≤ seuil matchs : points intra-match (10 buckets par match) |
| **K/D/A** | Plotly line chart | 3 traces (kills, deaths, assists) + shaded confiance. Downsamplé si > 200 points. Distribution KDA en histogramme avec KDE |
| **Résultats temporels** | Plotly chart | Série V/D/É colorée par match |
| **Séries meurtrières** | Plotly chart | Max killing spree, win streaks |
| **Armes** | Plotly barres horizontales | Top armes par kills (agrégé) + grenades/mêlée. Résolution labels FR/EN. Exclu weapon IDs blacklistés. **Condition** : db_path + xuid requis |

#### Onglet 2 : "Cartes et Modes"

| Composant | Type | Détail |
|-----------|------|--------|
| **Répartition par carte** | Charts | Breakdown V/D par carte (top 20, trié win rate desc) |
| **Win rate par carte vs historique** | Bullet chart | Barres session vs historique global |
| **Performance par mode** | Charts | Performance relative par mode de jeu |

#### Onglet 3 : "Distributions"

**Histogrammes (6 graphes en 3 lignes de 2)** :

| Ligne | Graphe 1 | Graphe 2 |
|-------|----------|----------|
| 1 | Accuracy (KDE, cyan) | Kills (KDE, vert) |
| 2 | Durée de vie moy (KDE, amber) | Performance score (KDE, violet) |
| 3 | Score/minute (si dispo) | Win rate glissant 14-match |

**Scatter plots (corrélations, 3 lignes de 2 + 1 pleine largeur)** :

| Scatter | Axes | Trendline |
|---------|------|-----------|
| Durée de vie vs Kills | X=avg_life, Y=kills | Oui |
| Précision vs K/D | X=accuracy, Y=ratio | Oui |
| Durée de vie vs Morts | X=avg_life, Y=deaths | Oui |
| Kills vs Morts | X=kills, Y=deaths | Oui |
| **Team MMR vs Enemy MMR** | Pleine largeur | Oui |

- Points colorés par outcome (WIN vert, LOSS rouge, TIE gris, DNF orange)
- Min 6 points requis, sinon message info
- Config Plotly : `PLOTLY_CLEAN_CONFIG`

#### Onglet 4 : "Avancé"

| Composant | Condition | Contenu |
|-----------|-----------|---------|
| **Premier événement** | db_path + xuid + ≥3 matchs | Distribution timing du 1er frag / 1ère mort |
| **Performance** | Toujours | Courbe performance score dans le temps vs historique (IC) |
| **Assists** | Toujours | Courbe assists/match |
| **Stats/minute** | Toujours | KPM, kills/min |
| **Durée de vie** | Toujours | Moyenne secondes + distribution |
| **Folie meurtrière** | db_path + xuid | Max spree, headshots %, précision tirs parfaits |
| **Tirs** | Si colonnes shots_fired/hit | Graphe précision tirs |
| **Dégâts** | Si colonnes damage_dealt/taken | Graphe dégâts infligés/reçus |
| **Rang/Score** | Si colonne rank ou personal_score | Graphe évolution rang/score |

#### Onglet 5 : "Progression"

| Composant | Contenu |
|-----------|---------|
| **Net Score Cumulé** | NPH (net score par heure) par match, coloré par outcome |
| **K/D cumulé + IC 95%** | Bandes de confiance autour du cumul |
| **EWMA K/D + Trendline** | EWMA α=0.20 du ratio + régression linéaire. Stats affichées (slope, R², pente) |
| **Trendline seule** | Si ≥ 10 matchs |
| **Heatmap d'intensité** | Y=match, X=10 buckets temporels, couleur=densité kills. Filtre segmenté : Tous/Victoires/Défaites. **Condition** : db_path + xuid + ≥3 matchs |
| **Progression de rang** | CSR/LUSR dans le temps (si données) |
| **Heatmap Win/Loss** | Matrice cartes × résultats |
| **Top par semaine** | Meilleurs stats agrégées par semaine |

> 🔒 **CHECKPOINT §2a** — Sous-vue Séries : 5 onglets, 6+ histogrammes, 5 scatter plots, 8+ graphes avancés, heatmap intensité avec filtre outcome, progression rang. Total composants : ~25 graphes/tableaux.

### 2.4 Sous-vue "Comparaison de Sessions"

#### 2.4.1 Sélecteurs

- 2 selectbox (Session A / Session B) triés par dernière activité
- Labels : `"[Classé] {label}"` si ranked, sinon `"{label}"`
- Défaut Session B : session active (ou première)
- Défaut Session A : session antérieure la plus similaire (`find_best_matching_previous_session()` — même catégorie + même statut ami)
- **Validation** : min 2 sessions, sinon warning `sc_need_two_sessions`

#### 2.4.2 En-tête temporel

2 colonnes côte à côte :
- Session A (rouge #E74C3C) : jour semaine + date + nombre de parties
- Session B (bleu #3498DB) : idem

#### 2.4.3 Cartes performance score

Grandes cartes côte à côte avec score de performance, badges comparatifs, delta.

#### 2.4.4 Donuts résultats

2 camemberts (A et B) : répartition V/D/É/DNF.
- Symboles : ▲ Victoire (vert), ▼ Défaite (rouge), ■ Égalité (gris), ⎕ DNF (noir)
- Centre : `{wins}/{total}`
- Hover : count absolu

#### 2.4.5 Match highlights

- Meilleur match : `{K}/{D} · F/D {ratio} — {mode}`
- Pire match : idem (si > 1 match)

#### 2.4.6 Métriques condensées (tableau)

| Métrique | Session A | Session B | Couleur |
|----------|-----------|-----------|---------|
| Parties | count | count | Vert = plus |
| Win rate | % | % | Vert = meilleur |
| Efficacité | ratio | ratio | Vert = meilleur |
| Durée vie | MM:SS | MM:SS | Vert = plus long |
| Total Kills | count | count | Vert = plus |
| Total Morts | count | count | Vert = **moins** |
| Assists | count | count | Vert = plus |

#### 2.4.7 Comparaison MMR

| Métrique | Session A | Session B |
|----------|-----------|-----------|
| Team MMR moyen | `{:.1f}` | `{:.1f}` |
| Enemy MMR moyen | `{:.1f}` | `{:.1f}` |
| Gap MMR moyen | `{:+.1f}` | `{:+.1f}` |

#### 2.4.8 Radar comparatif

- 3 axes : K/D (normalisé 0–100, ratio 2.0 = 100), Win rate (%), Accuracy (%)
- Traces : Moyenne historique (pointillé violet, si ≥1 session), Session A (rouge), Session B (bleu)
- Config : `PLOTLY_STATIC_CONFIG`

#### 2.4.9 Barres métriques groupées

Barres horizontales groupées : Kills/match, Morts/match, Ratio F/D, (optionnel) Win rate (Y2).
Couleurs : Rouge (A) vs Bleu (B).

#### 2.4.10 Net Score Cumulé

Deux courbes (A vs B). Overlay optionnel : skill rating (CSR/LUSR), performance_score en dashed.

#### 2.4.11 K/D Progression par partie

X = index match, Y = ratio F/D. Points A (rouge) + B (bleu). Optionnel Y2 : Accuracy (%).

#### 2.4.12 Répartition des modes

Barres horizontales groupées : Y = modes triés par total, X = nb parties. A (rouge) vs B (bleu).

#### 2.4.13 Win/Loss par cartes (tableau HTML)

| Carte | Session A (▲V ▼D ■É) | Session B (▲V ▼D ■É) | Total |

Symboles colorés. Thumbnail carte en hover.

#### 2.4.14 Tendance de participation

Radar ou barres 6 axes : Objectifs, Combat, Support, Score, Impact, Survie. Normes % par session. Moyenne historique en fond.

#### 2.4.15 Historique de parties (2 onglets)

Un tab par session (A / B). Tableau par session avec colonnes :

| Colonne | Format |
|---------|--------|
| Heure | jour semaine + date courte |
| Mode | Nom localisé FR |
| Carte | Nom + symbole V/D coloré |
| K / D / A | Entiers |
| Résultat | V/D/É/DNF coloré |
| Perf | Score gradienté |
| Team/Enemy MMR | `{:.0f} / {:.0f}` |

Tri : chronologique desc.

> 🔒 **CHECKPOINT §2b** — Sous-vue Sessions : sélecteurs (2, matching auto), en-tête temporel, cartes perf, 2 donuts, highlights, 7 métriques comparées, MMR (3 lignes), radar 3-axes, barres groupées, net score cumulé, K/D progression, modes, tableau carte, participation 6-axes, 2 tabs historique. Total : 15 composants.

### 2.5 Sous-vue "Historique des Parties"

Tableau principal des matchs avec colonnes :

| Colonne | Source | Format |
|---------|--------|--------|
| Lien | `match_url` | URL Waypoint |
| Date/Heure | `start_time` | JJ mmm. YYYY HH:MM (FR) |
| Carte | `map_name` | Nom traduit |
| Playlist | `playlist_fr` | Nom playlist localisé |
| Mode | `mode_ui` / `pair_name_fr` | Mode localisé |
| Résultat | `outcome_label` | V/D/É/DNF coloré |
| Score | — | `{my_team_score} - {enemy_team_score}` |
| Team MMR | `team_mmr` | `{:.0f}` |
| Enemy MMR | `enemy_mmr` | `{:.0f}` |
| Delta MMR | — | `{team_mmr - enemy_mmr}` |
| K/D/A | `kda` | Ratio formaté |
| Kills | `kills` | Entier |
| Deaths | `deaths` | Entier |
| Max Streak | `max_killing_spree` | Entier |
| Headshots | `headshot_kills` | Entier |
| Durée vie | `average_life` | MM:SS |
| Assists | `assists` | Entier |

**Empty state** : `no_matches` si DataFrame vide.

> 🔒 **CHECKPOINT §2c** — Sous-vue Historique : 17 colonnes, empty state. Sous-total section Stats : 3 sous-vues, ~40+ composants graphiques.

---

## 3. Escouade

### 3.1 Objectif

Analyse des coéquipiers et synergies d'équipe. Permet de sélectionner jusqu'à 3 coéquipiers et de comparer les performances en escouade (matchs communs same-team).

### 3.2 Architecture modulaire

13 sous-modules spécialisés orchestrés par `render_teammates_page()`.

### 3.3 Layout global

```
┌────────────────────────────────────────────────────────┐
│ Section "Mes stats" (KPIs perso)                       │
├────────────────────────────────────────────────────────┤
│ Section "Escouade" (multiselect coéquipiers, max 3)    │
├────────────────────────────────────────────────────────┤
│ En-tête escouade (KPI par membre)                      │
├────────────────────────────────────────────────────────┤
│ TAB 1 : Synergies     │ TAB 2 : Contributions          │
│   - Maps charts        │   - Stats/min                  │
│   - Form score         │   - Radar synergy              │
│   - Timeline perf      │   - Heatmap intensité          │
│   - Impact ranking     │   - 5 graphes métriques        │
│                        │   - Armes de kill              │
│                        │   - Butterfly premier frag     │
│                        │   - Médailles                  │
├────────────────────────┴───────────────────────────────┤
│ Tableau historique (12 colonnes, top 250)               │
├────────────────────────────────────────────────────────┤
│ [PANNEAU LÉGENDE] (position fixe, côté droit)          │
└────────────────────────────────────────────────────────┘
```

### 3.4 Sélecteur de coéquipiers

- **Widget** : `st.multiselect` (max 3 sélections)
- **Options** : Gamertags distincts depuis `shared.match_participants`, agrégés par xuid
- **Labels** : `"{gamertag} 🔵"` avec pastille couleur attribuée
- **Palette** : Okabe-Ito (daltonien-friendly)
- **Si 0 sélectionné** : message invite `tm_select_teammate`

### 3.5 Panneau Légende (position fixe)

- Panneau **position:fixed** côté droit (z-index:999)
- Points colorés (palette Okabe-Ito) + noms joueurs
- Visible uniquement entre les sections "Escouade" et "Impact" (contrôle via sentinelles DOM + JS)
- Max 150px largeur
- Modes : "fixed" (défaut) | "sidebar" (test) | "hidden" (debug)

### 3.6 Onglet "Synergies"

#### 3.6.1 Charts par carte

| Composant | Détail |
|-----------|--------|
| **Lollipop par carte** | Top 20 cartes triées par win rate desc. Barres horizontales |
| **Bullet win rate vs historique** | Session (bleu) vs historique global (gris) par carte |
| **Perf vs historique** | Performance_score session vs historique par carte |
| **Heatmap escouade** | Axes : Joueurs × Cartes. Cellules : performance_score colorée. Min 2 joueurs + 2 matchs |

#### 3.6.2 Historique de forme (rolling)

- Historique complet individuel (rolling 14 vs 90 matchs) filtré aux matchs de l'escouade
- Mode détail (≤ 20 matchs) : Ajoute buckets intra-match pour le joueur principal
- Une courbe par joueur, couleurs préservées de la palette

#### 3.6.3 Timeline d'évolution escouade

- Courbes lisses performance_score par joueur + MMR équipes
- X = matchs chrono, Y = performance_score
- Traces additionnelles : team_mmr vs enemy_mmr

#### 3.6.4 Impact (heatmap + ranking)

**Heatmap impact événements** :
- Lignes : joueurs. Colonnes : matchs (background outcome colorée)
- Cellules : emojis événements (⚡ First blood, 🎯 Clutch finisher, 💀 Last casualty, 🐌 Last group kill, 🪦 First group death, 🛡️ Silent hero, 🗡️ False brother, 💥 Top killer)

**Tableau ranking** :
- Colonnes : Joueur | Matchs #1-N | Badges (counts) | Score | Résultat
- MVP → 🏆 (vert), LVP → 🍌 (rouge), Passager clandestin (gris)
- Scoring : silent_hero=+30, false_brother=+40, top_killer=+50, etc.

### 3.7 Onglet "Contributions"

#### 3.7.1 Stats/min (barres groupées)

- Métriques : kills/min, deaths/min (sous axe, teinte négative), assists/min
- 1 groupe par joueur (jusqu'à 4)
- Records optionnels en traces fantômes hachurées

#### 3.7.2 Radar complémentarité (6 axes)

- Axes : Objectifs, Combat, Support, Score, Impact, Survie
- Données : PersonalScores API (Personal Score Awards)
- Remplissage : activé pour duo, désactivé pour trio

#### 3.7.3 Heatmap intensité kills

- Segmented control : toggle par joueur (All | Joueur 1 | 2 | 3)
- X = 10 phases intra-match. Y = matchs. Cellules = intensité kills
- Min requis : ≥ 3 matchs + ≥ 2 joueurs

#### 3.7.4 Métriques trio (5 graphes séquentiels)

| Métrique | Axe Y | Format |
|----------|-------|--------|
| K/D/A groupé | count | Barres + légende |
| Assists | assists | Courbe lisse |
| K/D Ratio | ratio | `.3f` |
| Accuracy | % | `.2f%` |
| Avg Life | secondes | `.1f` |
| Performance | score | `.1f` |

Couleurs : `colors_by_name` (Okabe-Ito). Smooth : rolling 10-match optionnel.

#### 3.7.5 Armes de kill (barres horizontales)

- Top-12 armes + grenades/mêlée
- 1 barre par joueur côte à côte
- Labels kills au-dessus (blanc, gras)
- Bandes alternées par arme

#### 3.7.6 Butterfly premier frag/mort

- Histogramme symétrique : frags ↑ positif (haut), morts ↓ négatif (bas)
- X = tranches 15 secondes
- Y = nombre de matchs
- Emojis : ▲ Premier frag | ▼ Première mort

#### 3.7.7 Médailles escouade

- 1 expander par joueur (expanded=True)
- Grille 6 colonnes par rangée
- Top 12 médailles par joueur sur matchs escouade

### 3.8 Tableau historique

12 colonnes, top 250 matchs récents :

| # | Colonne | Format | Couleur |
|---|---------|--------|---------|
| 1 | 🔗 App | Lien interne | — |
| 2 | Waypoint | Lien externe | — |
| 3 | Date | Jour HH:MM | — |
| 4 | Carte | Nom traduit + emoji | Carte-spécifique |
| 5 | Playlist | Nom traduit | — |
| 6 | Mode | Normalisé | — |
| 7 | Résultat | V/D/É/DNF | 🟢/🔴/🟣 |
| 8 | Win rate hist | % sur tout l'historique | 🟢 ≥55% / 🔴 ≤45% |
| 9 | Score | `{my} - {enemy}` | — |
| 10 | Team MMR | Entier | — |
| 11 | Enemy MMR | Entier | — |
| 12 | Δ MMR | ±N | 🟢 >0 / 🔴 <0 |

### 3.9 Empty states

| Condition | Message |
|-----------|---------|
| 0 matchs après filtre | `no_matches` |
| Session solo sélectionnée | `tm_solo_session_info` |
| Cache lent | `tm_loading_slow` |
| Aucun match trio commun | `tm_no_trio_matches` |
| Aucune médaille partagée | `tm_no_shared_medals` |
| < 3 matchs (heatmap) | Section non affichée |
| < 2 joueurs (heatmap escouade) | Section non affichée |

> 🔒 **CHECKPOINT §3** — L'agent confirme avoir lu : sélecteur multi (max 3), panneau légende fixe, 2 onglets (Synergies : 4 charts carte + form score + timeline + impact 8 emojis + ranking ; Contributions : stats/min + radar 6 axes + heatmap intensité + 5/6 métriques + armes top-12 + butterfly + médailles), tableau 12 colonnes (250 rows), 7 empty states. Données : shared.match_participants + highlight_events + PersonalScores API.

---

## 4. Synthèse

### 4.1 Objectif

Vue stratégique comparant les performances en **Solo** vs **Escouade**, avec des visualisations agrégées de l'activité (heatmap temporelle, top hebdo, breakdown carte/mode).

### 4.2 Layout

```
┌──────────────────────────────────────────┐
│ Sélecteur de période (segmenté)          │
├──────────────────────────────────────────┤
│ Répartition par Carte et Mode            │
├──────────────────────────────────────────┤
│ Heatmap Temporelle (jour × heure)        │
├──────────────────────────────────────────┤
│ Top par Semaine                          │
├──────────────────────────────────────────┤
│ Comparaison Solo vs Escouade (bipolaire) │
└──────────────────────────────────────────┘
```

### 4.3 Sélecteur de période

- **Type** : segmented_control
- **Options** : `all` | `2y` | `1y` | `1m` | `1w`
- **Labels i18n** : `encounters_period_{key}` (Tous, 2 ans, 1 an, 1 mois, 1 semaine)
- **Défaut** : `"all"`
- **Logique** : Filtre le DataFrame selon le nombre de jours (`{2y: 730, 1y: 365, 1m: 30, 1w: 7}`)

### 4.4 Répartition par Carte et Mode

- Breakdown par carte + breakdown par mode (graphiques délégués au module `win_loss.py`)

### 4.5 Heatmap Temporelle

- Heatmap jours × heures de l'activité (matchs joués)
- Source : `start_time` (dates)

### 4.6 Top par Semaine

- Classement des meilleures semaines en KDA, win rate

### 4.7 Comparaison Solo vs Escouade

**Graphique bipolaire** (barres horizontales divergentes)

- **Solo** ← gauche (Cyan), **Escouade** → droite (Vert)
- **Ligne zéro** : Slate opacity 0.8

**6 métriques comparées** (ordre reverse dans le graphe) :

| # | Métrique | Format | i18n |
|---|----------|--------|------|
| 1 | K/D | `{:.2f}` | — |
| 2 | Win Rate | `{:.1f}%` | `col_win_rate` |
| 3 | Accuracy | `{:.1f}%` | `col_accuracy` |
| 4 | KPM (kills/min) | `{:.2f}` | `col_kpm` |
| 5 | Avg Life | `{:.0f}s` | `col_avg_life` |
| 6 | Perf Score | `{:.1f}` | `sc_performance_score` |

**Données** :
- Split via `is_with_friends` (True = escouade, False = solo)
- Source : `player_match_enrichment` table

**Layout Plotly** : bargroupé overlay, hauteur dynamique `max(320, 70 × len(metrics))`

**Caption** : `syn_sample_split` avec `solo={count}, squad={count}`

**Empty states** :
- `dff.is_empty()` → `no_matches`
- `is_with_friends` absent → `syn_no_data`
- Solo ou Squad vide → `syn_no_data`
- Métriques vides → `syn_no_data`

> 🔒 **CHECKPOINT §4** — L'agent confirme avoir lu : sélecteur période (5 options), breakdown carte/mode, heatmap temporelle, top semaine, bipolaire Solo/Escouade (6 métriques, couleurs Cyan/Vert, 4 empty states). Source clé : `is_with_friends`.

---

## 5. Explorer

### 5.1 Objectif

Recherche et exploration détaillée des matchs. Deux modes d'accès : **par filtres** (cascade date/type/playlist/mode/carte) ou **par joueur** (recherche fuzzy gamertag). Affiche ensuite soit la liste des résultats, soit la vue détaillée d'un match unique.

### 5.2 Layout

```
┌─────────────────────────────────────────────────┐
│ SECTION 1 : Filtres en Cascade                  │
│   Ligne 1 : Date + Escouade                     │
│   Ligne 2 : Type · Playlist · Mode · Carte      │
│   Ligne 3 : Sélecteur de match                  │
├─────────────────────────────────────────────────┤
│ DIVIDER                                         │
├─────────────────────────────────────────────────┤
│ SECTION 2 : Recherche Joueur (typeahead + badges)│
├─────────────────────────────────────────────────┤
│ [Bouton Recherche]                              │
├─────────────────────────────────────────────────┤
│ SECTION 3 :                                     │
│   Variante A : Tableau matchs filtrés (paginé)  │
│   Variante B : Résultats par joueur (encounters) │
│   Variante C : Vue détail match unique          │
└─────────────────────────────────────────────────┘
```

### 5.3 Filtres en cascade

#### 5.3.1 Ligne 1 : Date + Escouade

| Champ | Type | Détail |
|-------|------|--------|
| **Date** | Date input DD/MM/YYYY | Min/Max = dates extrêmes du joueur. Défaut = dernière date. Si aucun match à la date : trouve la date la plus proche et affiche info |
| **Mode escouade** | Selectbox | Options : Tous / Solo / Escouade. Filtre via `is_with_friends` |

#### 5.3.2 Ligne 2 : Cascade 4 dimensions

| # | Dimension | Options | Logique cascade |
|---|-----------|---------|-----------------|
| 1 | **Type d'expérience** | PvE (firefight), Ranked, Unranked | Classification automatique depuis playlist |
| 2 | **Playlist** | Valeurs distinctes, filtrées par type si sélectionné | Noms traduits FR |
| 3 | **Mode** | Valeurs distinctes, filtrées par playlist si sélectionné | `normalize_mode_label()` |
| 4 | **Carte** | Valeurs distinctes, filtrées par mode si sélectionné | Noms traduits |

Chaque filtre cascade restreint les options du suivant.

#### 5.3.3 Sélecteur de match

- Selectbox : Match ID + datetime formaté
- Retourne `selected_mid` (str ou None)

### 5.4 Recherche Joueur

| Composant | Détail |
|-----------|--------|
| **Input gamertag** | Typeahead avec fuzzy search (`difflib.get_close_matches()` + substring + leetspeak). Top 8 suggestions, cutoff 0.4 |
| **Badges encounter** | Si gamertag choisi : badges de stats encounter (Dur à cuire, Allié+, Coriace) |
| **Bouton Recherche** | `btn_search`, width="stretch", type="primary" |

### 5.5 Résultats — Variante A (filtres)

**Tableau paginé** (si aucun match spécifique sélectionné) :

- **Page size** : [50, 100, 250] ou total si < 50
- **Navigation pages** : input numérique 1-based
- **Caption** : `"Affichage {start}–{end} sur {total}"`

**Colonnes** : Match ID (lien Waypoint), Date, Playlist (FR), Mode (FR), Map, Score (`{my} vs {enemy}`), Outcome (coloré), K/D/A, Accuracy, Performance, MMR Team/Enemy/Delta

### 5.6 Résultats — Variante B (par joueur)

1. Charge matchs communs (joueur principal vs cible)
2. Affiche bilan encounter : `"{gamertag} — {total} matchs (A:{alliés} | E:{ennemis})"` + badges
3. Sépare alliés / adversaires
4. 2 tableaux paginés (section alliés vert, section ennemis rouge)

**Badges encounter** :
- **Dur à cuire** : deaths/kills > 2.0
- **Allié+** : winrate_as_ally ≥ 65%
- **Coriace** : winrate_vs_enemy ≤ 35%

### 5.7 Match View — Détail d'un match unique

Quand un match spécifique est sélectionné (via filtres, recherche ou deep link), affiche la vue détaillée complète.

#### 5.7.1 KPI Header (4 colonnes)

| Zone | Contenu |
|------|---------|
| Date | Format FR complet |
| Score | `{my_team} vs {enemy}` coloré par outcome. Badge optionnel : dominance/humiliation/remontada/débandade/contre-remontada |
| Playlist | FR traduit |
| Mode & Carte | `"{mode} sur {map}"` |

**Badges spéciaux de match** :

| Code | Label | Couleur |
|------|-------|---------|
| 1 | Domination | Vert foncé |
| 2 | Humiliation | Violet |
| 3 | Remontada | Bleu |
| 4 | Débandade | Rouge |
| 5 | Contre-remontada | Vert-canard |

#### 5.7.2 Miniature + Performance + Rang (3 colonnes)

| Zone | Contenu |
|------|---------|
| **Thumbnail carte** | Image de la carte ou fallback info |
| **Performance Score** | Gros numéro 0–100, coloré (≥80 vert, ≥60 cyan, ≥40 amber, ≥20 orange, <20 rouge) |
| **Rang** | HTML du rang courant (CSR/LUSR) |

#### 5.7.3 Onglet "Résumé"

| Composant | Type | Détail |
|-----------|------|--------|
| **Expected vs Actual** | Barres | K/D/A réels vs attendus (basé historique) |
| **Weapon Kills** | Camembert 70% + tableau 30% | Kills par arme (filmshell). Fusion variantes. + grenades/mêlée API |
| **Participation** | Radar 6 axes | Objectifs, Combat, Support, Score, Impact, Survie (PersonalScores API) |

#### 5.7.4 Onglet "Combat"

| Composant | Type | Détail |
|-----------|------|--------|
| **Impact & Timeline** | Badges + timeline | First Blood, Shutdown, Spree Ender. Timeline framerate des highlight_events |
| **Dominance d'équipe** | Tug-of-War | Barres dominance + kill feed, granularité 30s |
| **Cadence** | Histogramme stacked | Kill rate par bucket (segmenté 15/30/60s). Mon équipe vs adverse |
| **Antagoniste** | Barres stacked | Némésis + souffre-douleur + interactions croisées |
| **Timeline K/D** | Multi-lignes | Frags cumulés par joueur vs temps |

#### 5.7.5 Onglet "Équipe"

##### Scoreboard (19 colonnes)

| Colonne | Source |
|---------|--------|
| Joueur | gamertag |
| Rang | rank |
| Score | score |
| Kills / Deaths / Assists | kills, deaths, assists |
| K/D/A ratio | calculé |
| Arme de prédilection | top_weapon_id |
| Killing Spree max | max_killing_spree |
| Headshots | headshot_kills |
| Perfect Kills | perfect_kills |
| Tirs tirés / touchés | shots_fired, shots_hit |
| Précision | accuracy |
| Mêlée | melee_kills |
| Power Weapon | power_weapon_kills |
| Dégâts infligés / reçus | damage_dealt, damage_taken |
| Durée de vie moy | avg_life_seconds |

**Highlights** :
- Min/Max par colonne → classes CSS `os-sb-td--best` (vert), `os-sb-td--worst` (orange)
- Colonnes inversées (deaths, damage_taken) : moins = mieux
- **MVP** : joueur humain max(best_count) → badge MVP
- **LVP** : joueur humain max(worst_count) → badge LVP
- Détails expandibles par joueur (click pour extra stats)

##### Historique des Rencontres

| Colonne | Format |
|---------|--------|
| Joueur | gamertag + ordinal badge |
| Rôle | Allié/Ennemi (coloré) |
| Rencontres | total count |
| WR Allié | winrate as ally (%) |
| WR Ennemi | winrate vs enemy (%) |
| K/D croisé | kills_dealt/deaths_suffered |
| Dernière rencontre | date relative ("il y a 2 jours") |

Badges par joueur : 🟢 Allié+ (WR allié ≥65%, ≥2 matchs), 🟡 Coriace (WR ennemi ≤35%, ≥3 matchs), 🔴 Dur à cuire (deaths/kills >2.0, ≥3 deaths).

#### 5.7.6 Onglet "Citations & Médailles"

**Citations** :
- Grille centrée 8 colonnes max
- Chaque citation qui a progressé dans ce match : image circulaire avec ring progress (doré si maître, cyan sinon), nom ellipsisé, niveau, compteur + delta vert
- Logique : si newly mastered → afficher, si déjà maître avant → skip

**Médailles** :
- Grille des médailles obtenues dans ce match
- Images, noms, counts

### 5.8 Deep links

- `?match_id=X` → ouvre directement la vue match (scroll automatique)
- `?player=X` → initialise la recherche joueur (pending gamertag)
- Scroll via micro-composant HTML invisible `scrollIntoView({behavior:'instant'})`

### 5.9 Empty states

| Condition | Message |
|-----------|---------|
| Aucun match à la date | Info + fallback date la plus proche |
| 0 résultats filtrés | `exp_select_match_hint` |
| Aucun encounter joueur | Section non affichée |

> 🔒 **CHECKPOINT §5** — L'agent confirme avoir lu : filtres cascade (date + escouade + 4 dimensions cascade), recherche joueur (fuzzy top 8 + badges), 3 variantes résultats (tableau paginé / par joueur / match view), Match View : KPI header (5 badges spéciaux), thumbnail+perf+rang, 4 onglets (Résumé : expected vs actual + armes camembert + radar 6 axes ; Combat : impact timeline + dominance tug-of-war + cadence 15/30/60s + antagoniste + K/D timeline ; Équipe : scoreboard 19 colonnes MVP/LVP + encounters ; Citations : progression rings + médailles). Deep links : match_id + player.

---

## 6. Médias

### 6.1 Objectif

Galerie des captures et clips vidéo du joueur. Indexation automatique depuis le disque local, avec filtrage multicritère, groupement, tri, système de likes et lightbox intégrée.

### 6.2 Pipeline d'enrichissement des données

1. `MediaIndexer.load_media_for_ui()` → DF brut (file_path, file_name, kind, thumbnail_path, capture_end_utc)
2. `_enrich_with_match_data()` → JOIN sur match_id (ajoute map_ui, mode_ui, outcome)
3. `_enrich_from_enrichment_table()` → SELECT player_match_enrichment (ajoute session_label, is_with_friends)
4. `_enrich_with_likes()` → Booléen depuis browser localStorage
5. `_normalize_mode_ui()` → normalize_mode_label() si activé dans settings

### 6.3 Toolbar de filtres

8 contrôles en ligne horizontale compacte :

| Champ | Type | Options | Défaut |
|-------|------|---------|--------|
| **Auteur** | Selectbox | "Tous", "Mine", "Coéquipier → {gt}" | "Tous" |
| **Carte** | Selectbox | Valeurs distinctes map_ui | "Tous" |
| **Mode** | Selectbox | Valeurs distinctes mode_ui | "Tous" |
| — | Séparateur | `│` | — |
| **Grouper par** | Selectbox | Aucun, Auteur, Date, Session, Expérience, Carte, Mode, Aimé | "Aucun" |
| **Trier par** | Selectbox | Date capture, Carte, Mode | "Date capture" |
| **Ordre** | Selectbox | Décroissant, Ascendant | "Décroissant" |
| **Autoplay** | Implicite (toggle session) | — | True |

### 6.4 Modes de groupement

#### Mode "Auteur"
- Section "🎬 Mes captures" (section == "mine")
- Section "🎬 Captures de {gamertag}" — par coéquipier
- Section "⚠️ Captures sans correspondance" — si show_all=True

#### Mode autre (Date / Carte / Mode / Session / Expérience / Aimé)
- Colonne de groupe construite dynamiquement
- Chaque groupe = section avec titre formaté + grille de médias

**Formatage des titres de groupe** :

| Clé | Format |
|-----|--------|
| **date** | Format complet du jour ("17 avril 2026") |
| **session** | Label session ("Session 1 - 2026-04-17") |
| **experience** | "Solo" / "Escouade" |
| **liked** | i18n liked_yes / liked_no |
| Autres | Valeur brute |

### 6.5 Grille de médias

- Grille responsive de thumbnails
- Chaque item : thumbnail + nom fichier + date + match_id
- Click → ouvre lightbox

### 6.6 Lightbox

| Élément | Détail |
|---------|--------|
| **Dialog Streamlit** | Plein écran |
| **Navigation** | Boutons ◀ / ▶ (prev/next). Autoplay vidéos (avance à fin de vidéo si activé) |
| **En-tête** | `"{map} · {mode} · {date_long} {idx}/{total}"` |
| **Pagination** | "1/10" si multiples |
| **Actions** | Navigation match (→ Explorer ce match), Fermeture |

### 6.7 Système de likes

- Bouton Like (icône SVG data-URI)
- Persisté en **browser localStorage** (pas en DB)
- Fragment isolé (`@fragment_if_available`)
- Permet de filtrer par "Aimé" dans le groupement

### 6.8 Condition de désactivation

- Si `settings.media_enabled == False` → affiche `media_disabled` et arrête le rendu

### 6.9 Empty states

| Condition | Message |
|-----------|---------|
| Media désactivé | `media_disabled` |
| Aucun média indexé | `media_no_indexed` |
| Aucun résultat après filtre | `no_data_filter` |
| Fichier manquant sur disque | `media_file_missing` |

> 🔒 **CHECKPOINT §6** — L'agent confirme avoir lu : pipeline 5 étapes d'enrichissement, toolbar 8 contrôles, 2 modes groupement (auteur / autre avec 7 clés), grille thumbnails, lightbox (nav ◀▶ + autoplay + pagination + nav match), likes (localStorage + fragment), condition désactivation, 4 empty states. Sources : disque local + media_files table + match enrichment + browser storage.

---

## 7. Profil

### 7.1 Objectif

Section bi-face : **Carrière** (progression rang, XP, LUSR/CSR, rencontres) et **Citations** (commendations + médailles). Navigation via segmented control.

### 7.2 Navigation interne

- **Segmented control** : `career` | `citations`
- **Labels** : "Carrière" | "Citations"
- **State key** : `SK.V7_PROFILE_VIEW`

### 7.3 Vue Carrière

#### 7.3.1 Rang & Progression XP (3 colonnes)

| Zone | Contenu |
|------|---------|
| **Icône rang** | Adornement personnalisé ou icône de rang par défaut (base64). Caption : "Données du JJ/MM/YYYY HH:MM" |
| **Métriques texte** | Rang : `"{rank_number} / 272"`. XP Total : `"{xp:,}"`. XP courant : `"{current_xp:,}"` (si pas max). XP rang suivant : `"{xp_for_next:,}"` (si pas max). Si max : "RANG MAX" |
| **Gauge Plotly** | Jauge de progression vers le rang suivant. Si rang max : gauge pleine |

#### 7.3.2 Progression vers Héros (2 colonnes)

| Zone | Contenu |
|------|---------|
| **Métriques** | XP gagné, XP restant, XP requis total (constante 19 050 000), Rang actuel / max |
| **Gauge Héros** | Jauge de progression vers le statut Héros |

#### 7.3.3 Historique XP & Projections

- **Graphe XP** : Courbe XP dans le temps avec :
  - Courbe estimée (si dates pré-sync disponibles)
  - 2 projections Héros : optimiste (rythme actuel) + pessimiste (moyenne depuis 1er match)
- **Tableau snapshots** : Expander (collapsed). 10 derniers snapshots. Format : `"{date} | Rang {n}: {label} | XP: {total:,}"`
- **Calcul rythme** : `_compute_active_xp_per_day()` avec fallback `_compute_fallback_xp_per_day()`

#### 7.3.4 LUSR / CSR

##### Cartes de rang (3 par ligne)

Tri fixe : ranked, arena, btb, tactical, social, fun, puis autres.

Chaque carte :
- **Header** : `"{emoji} {playlist_group_label}"` |
- **Image** : Tier correspondant (90px², base64) |
- **Tier label** : ex "Gold 3" |
- **Badge** : fond bleu `#00B7EB` = "LUSR", fond or `#FFD700` = "CSR" + valeur `{rating:.0f}` |
- **Delta** : `▲ +{delta}` (vert `#00C853`), `▼ -{delta}` (rouge `#FF5252`), `= 0` (gris) |

##### Graphe d'évolution LUSR/CSR

**Filtre période** : segmented `all|2y|1y|1m|1w` (identique à Synthèse)
**Filtre groupe playlist** : selectbox ("Tous les groupes" ou groupe spécifique)
**Granularité temporelle** : auto-adaptée (1w → sans troncature, 1m → par jour, ≥1y → par semaine)
**Chart Plotly** : courbes par playlist group

#### 7.3.5 Rencontres (Antagonistes & Victimes)

**Filtre période** : segmented (identique au LUSR)

3 sous-sections :

| Section | Source | Contenu |
|---------|--------|---------|
| **Top 10 Rencontrés** | `_load_top_encountered()` | Tableau HTML. Popover légende badges. Empty : `career_encounters_no_data` |
| **Top 10 Némésis** | `_load_top_nemeses()` | Tableau HTML mode="nemesis". Empty : `career_antagonists_no_data` |
| **Top 10 Victimes** | `_load_top_victims()` | Tableau HTML mode="victim". Empty : `career_antagonists_no_data` |

Layout : Némésis et Victimes côte à côte (2 colonnes).

### 7.4 Vue Citations

#### 7.4.1 Commendations Halo 5

- Agrégation via `CitationEngine.aggregate_for_display()`
- **3 métriques** : Citations obtenues (`{total:,}`), Matchs analysés (len(dff))
- **Grille détaillée** : `render_h5g_commendations_section()` — progression par commendation avec rings, niveaux, compteurs

#### 7.4.2 Médailles Halo Infinite

- Chargement via callback `top_medals_fn()`
- **3 métriques** : Médailles distinctes, Médailles totales (`{sum:,}`), Matchs analysés
- **Hint optionnel** : si indices UI visibles
- **Distribution** : Barplot Plotly top 25 médailles. Config `PLOTLY_STATIC_CONFIG`
- **Grille médailles** : 8 colonnes par rangée. Deltas si filtré. Descriptions chargées depuis `load_medal_description_map(lang)`

> 🔒 **CHECKPOINT §7** — L'agent confirme avoir lu : 2 vues (career / citations). Career : rang + XP (gauge + métriques + max), Héros (gauge + projection 19.05M XP), historique XP (graphe + 2 projections + snapshots), LUSR/CSR (cartes n=6 triées + graphe évolution avec filtre période + filtre groupe), rencontres (top 10 ×3 avec filtre période). Citations : commendations H5 (agrégation + rings + niveaux), médailles HI (distribution barplot + grille 8 colonnes + deltas + descriptions).

---

## 8. Settings

### 8.1 Objectif

Configuration de l'application : langue, affichage, médias, notifications Discord, options de backfill.

### 8.2 Layout

```
┌──────────────────────────────────────────────────────────┐
│ Titre + [Sauvegarder] [Recharger]                        │
├──────────────────────────────────────────────────────────┤
│ § Langue & Région                                        │
├──────────────────────────────────────────────────────────┤
│ § Affichage                                              │
├──────────────────────────────────────────────────────────┤
│ § Médias                                                 │
├──────────────────────────────────────────────────────────┤
│ § Discord Notifications                                  │
├──────────────────────────────────────────────────────────┤
│ § Backfill (Données manquantes)                          │
├──────────────────────────────────────────────────────────┤
│ Avertissements cohérence (en bas)                        │
└──────────────────────────────────────────────────────────┘
```

### 8.3 Section Langue & Région

| Champ | Type | Options | Défaut | Persistance |
|-------|------|---------|--------|-------------|
| **Langue** | Selectbox | Français, English | Depuis settings disque | Disque + browser + i18n reload + rerun |
| **Fuseau horaire** | Selectbox | Liste curatée (`CURATED_TZ_LIST`) | Europe/Paris | Disque |

### 8.4 Section Affichage

| Champ | Type | Défaut | Effet |
|-------|------|--------|-------|
| **Normaliser labels modes** | Toggle | True | Affecte rendu `mode_ui` partout |
| **Afficher indices UI** | Toggle | Depuis browser | Persiste session + browser |
| **Afficher Records** | Toggle | False | Traces records dans graphes |
| **Career exclure BTB** | Toggle | — | Exclut BTB des top matches career |
| **Sync efface caches** | Toggle | False | Force cache clear à chaque sync |

### 8.5 Section Médias

| Champ | Type | Détail |
|-------|------|--------|
| **Dossier captures** | Directory input | Chemin local. Placeholder : "Ex: D:/Captures" |
| **Tolérance match** | Slider 0–30 min | Step 1. Fenêtre d'association média↔match |
| **Watcher actif** | Toggle | Active le monitoring fichiers en temps réel |
| **Débounce watcher** | Slider 1–60s | Désactivé si watcher off |
| **Réinitialiser index** | Bouton | Appelle `MediaIndexer.reset_media_tables()`. Success/Error feedback |

### 8.6 Section Discord Notifications

| Champ | Type | Détail |
|-------|------|--------|
| **Master toggle** | Toggle | Active/désactive tout le bloc Discord |
| **Webhook URL** | Text input | Désactivé si toggle off. Détection auto env var `DISCORD_WEBHOOK_URL` (prioritaire, invisible) |
| **Langue Discord** | Selectbox | fr / en. Désactivé si toggle off |
| **Types notifications** | 4 checkboxes | Sync, Backfill, Nouvelle version, Nouveau média. Tous désactivés si toggle off |

### 8.7 Section Backfill

- **Avertissement** : Caption `settings_backfill_warning`
- **Expander** (collapsed=False)

| Champ | Type | Détail |
|-------|------|--------|
| **Master toggle** | Toggle | Active/désactive tous les sous-types |
| **Grille 3×3** | 8 checkboxes | Médailles, Compétence, Alias, Personal Scores, Performance Scores (on), LUSR (on), Événements, Armes. Tous désactivés si master off |

### 8.8 Avertissements de cohérence (footer)

| Check | Condition | Warning |
|-------|-----------|---------|
| Discord sans webhook | notifications=on ET URL vide ET pas d'env var | `warn_discord_no_webhook` |
| Médias sans dossier | media=on ET captures_base_dir vide | `warn_media_no_dir` |

### 8.9 Boutons header

- **Sauvegarder** : Double-write disque + success feedback + rerun
- **Recharger** : Recharge depuis disque → session_state + rerun

> 🔒 **CHECKPOINT §8** — L'agent confirme avoir lu : 5 sections (Langue : 2 champs, Affichage : 5 toggles, Médias : 5 champs + bouton reset, Discord : master toggle + URL + langue + 4 types notifications, Backfill : master toggle + 8 checkboxes 3×3), 2 avertissements cohérence, double-persist (disque + browser).

---

## Annexe A — Récapitulatif des checkpoints

| § | ID Checkpoint | Composants clés à confirmer |
|---|---------------|----------------------------|
| 0 | §0 | Shell L1 (7 tabs), L2 (filtres/chips), KPI bar (5 métriques), routing (8 URLs), deep links (6 params), PageContext (13 champs) |
| 1 | §1 | 10 composants Accueil, 6 appels API live, 4 empty states |
| 2a | §2a | Séries : 5 onglets, ~25 graphes |
| 2b | §2b | Sessions : 15 composants, radar 3-axes, 2 tabs historique |
| 2c | §2c | Historique : 17 colonnes |
| 3 | §3 | Escouade : 2 onglets (Synergies + Contributions), 12 colonnes historique, 7 empty states |
| 4 | §4 | Synthèse : sélecteur période, bipolaire 6 métriques, 4 empty states |
| 5 | §5 | Explorer : cascade 4 dim, fuzzy search, 3 variantes résultats, Match View 4 onglets (19 col scoreboard) |
| 6 | §6 | Médias : pipeline 5 étapes, toolbar 8 contrôles, lightbox, likes localStorage, 4 empty states |
| 7 | §7 | Profil : Career (rang+Héros+XP+LUSR+encounters) + Citations (commendations+médailles barplot+grille) |
| 8 | §8 | Settings : 5 sections, 2 avertissements, double-persist |

> 🔒 **CHECKPOINT FINAL** — L'agent confirme avoir lu **l'intégralité** du document FUNCTIONAL_SPECS.md :
> 9 sections (§0–§8), 10 checkpoints intermédiaires, 8 sections V7 spécifiées.
> Total approximatif : ~80 composants graphiques, ~100 champs/contrôles, ~30 empty states, 6+ appels API live.
