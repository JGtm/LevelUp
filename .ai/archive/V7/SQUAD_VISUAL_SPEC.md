# Squad — Specification visuelle des graphes et tableaux

> Source de verite pour reproduire **quasi-exactement** chaque chart/tableau Python `v7/cockpit` en Plotly + React.
>
> Mode operatoire pour remplir ce document :
> 1. Lancer l'app Streamlit locale sur la branche `v7/cockpit` avec un dataset de demo riche (≥ 200 matchs avec ≥ 3 coequipiers frequents).
> 2. Pour chaque section ci-dessous, naviguer sur la page Coequipiers, faire un screenshot.
> 3. Extraire le JSON Plotly via Streamlit (clic-droit sur le chart -> "Save as JSON" si dispo, ou via `fig.to_json()` dans un script de debug temporaire qui patche les `st.plotly_chart`).
> 4. Coller dans la fiche : screenshot path + extrait JSON pertinent (data + layout) + notes.
>
> Date de production attendue : Phase 0bis (1.5 jour).

---

## Table des matieres

- [En-tetes](#en-tetes)
  - [§2.1 KPI personnels](#§21-kpi-personnels-8-cartes--tendance)
  - [§2.2 Score equipe + scores individuels](#§22-score-equipe--scores-individuels)
- [Onglet Synergies](#onglet-synergies)
  - [§3.1 Lollipop W/L par carte](#§31-lollipop-wl-par-carte)
  - [§3.2 Bullet winrate session vs historique](#§32-bullet-winrate-session-vs-historique)
  - [§3.3 Perf vs historique par carte](#§33-perf-vs-historique-par-carte)
  - [§3.4 Heatmap escouade joueur x carte](#§34-heatmap-escouade-joueur-x-carte)
  - [§3.5 Timeline performance multi-joueurs](#§35-timeline-performance-multi-joueurs)
  - [§3.6 Form Score lisse (LOWESS)](#§36-form-score-lisse-lowess)
  - [§3.7 Impact 8 roles (heatmap + ranking)](#§37-impact-8-roles)
  - [§3.8 Tableau historique escouade](#§38-tableau-historique-escouade)
  - [§3.9 Cadence trio](#§39-cadence-trio)
- [Onglet Contributions](#onglet-contributions)
  - [§4.1 Stats par minute groupees](#§41-stats-par-minute-groupees)
  - [§4.2 Radar 6 axes normalises par mode](#§42-radar-6-axes)
  - [§4.3 6 charts performance trio dedies](#§43-6-charts-performance-trio)
  - [§4.4 Killing Spree + HS/PK enrichis](#§44-killing-spree--hspk-enrichis)
  - [§4.5 Heatmap intensite (match x 10 buckets)](#§45-heatmap-intensite)
  - [§4.6 First Events](#§46-first-events)
  - [§4.7 Tableau armes](#§47-tableau-armes)
  - [§4.8 Barplot armes top 12 grouped](#§48-barplot-armes-top-12)
  - [§4.9 Galerie medailles](#§49-galerie-medailles)

---

## Format de fiche

Chaque fiche suit cette structure :

```
### Titre

- Reference Python : <fichier>:<ligne fonction>
- Screenshot : `docs/screenshots/squad/<nom>.png`
- Type Plotly : <type principal + traces composantes>
- Composition : <si compose>

#### Encoding
- xAxis : type (`category|value|date`), valeur, formatter, range
- yAxis : idem
- series[]
  - trace 0 : type, x, y, marker.color, name, customdata, hovertemplate
  - trace 1 : ...

#### Layout
- title : ...
- legend : ...
- annotations : ...
- shapes : ...
- barmode : ...
- hovermode : ...
- margin : ...

#### Tooltip
- hovertemplate exact

#### Comportements
- empty state : ...
- loading state : ...
- error state : ...
- interactivite : zoom / brush / segmented_control / ...

#### JSON Plotly extrait (data + layout)
```json
<coller ici>
```

#### Notes de portage
- Differences attendues entre Python et React.
- Cas limites a tester.
```

---

## En-tetes

### §2.1 KPI personnels (8 cartes + tendance)

- Reference Python : `src/app/kpis_render.py::render_kpis_section`, `_build_kpi_cards`
- Screenshot : `docs/screenshots/squad/kpi-strip.png` **TODO**
- Type : composant React custom, **PAS un chart Plotly** — 8 cartes + 1 barre empilee
- Composition : `<KpiStrip>` avec 8 `<KpiCard>` (label, main, sub, trend) et 1 `<OutcomeBar>` (W/L/T/DNF stacked)

#### Encoding
TODO Phase 0bis : extraire la liste `_build_kpi_cards()` exacte avec valeurs/sous-valeurs/threshold trend (8 %).

#### Trend visuel
- `'above'` : fleche `▲` couleur `--score-good`
- `'below'` : fleche `▼` couleur `--score-poor`
- `'near'` ou `'none'` : pas de fleche

#### Comportements
- Empty : afficher cartes avec `'-'` mais pas de placeholder gris.
- Loading : skeleton avec dimensions identiques.

#### Notes de portage
- Pas un chart Plotly — implementer en React pur via shadcn `<Card>`.
- Trend calcule cote backend (endpoint `/players/{slug}/kpi-stats?scope=current`) ; `reference_kpis` charge avec `scope=alltime`.

---

### §2.2 Score equipe + scores individuels

- Reference Python : `src/ui/components/performance.py::render_squad_session_header` (ligne 215)
- Screenshot : `docs/screenshots/squad/squad-score-header.png` **TODO**
- Type : composant React custom, **PAS un chart Plotly** — N+1 cartes
- Composition : `<SquadScoreHeader>` avec 1 `<TeamScoreCard>` + N `<PlayerScoreCard compact>`

#### Layout
`st.columns(N+1)` : team carte premiere, puis 1 carte par joueur dans l'ordre `[me, f1, f2, f3]`.

#### Carte equipe (`_render_compact_team_card`)
- Label : `t('squad_score_header')` ("Score d'equipe")
- Score : `final` (entier 0-100)
- Status : grade lettre (font 1.6rem, bold, letter-spacing 0.05em)
- Detail bonus : `t('squad_score_bonus', base, bonus)` si `final > base_avg`, sinon `t('squad_score_base_only', base)`

#### Carte joueur (`render_performance_score_card` compact)
- Label : nom joueur
- Score : `score` (entier 0-100)
- Status : `get_score_label(score)` (excellent/good/average/poor/bad)
- Couleur : `get_score_class(score)` -> `text-excellent` etc.
- Badge : `▲` (text-positive) si `player_score > avg_score`, `▼` (text-negative) si `<`, sinon rien

#### Comportements
- Si `len(dff) < 2` : ne pas afficher.
- Si moins d'1 coequipier selectionne : ne pas afficher (deja ailleurs sur la page).

#### Notes de portage
- Pas un chart — implementer en React pur.
- Mapping `SCORE_THRESHOLDS` -> tokens dans `SQUAD_DESIGN_TOKENS.md` §4.

---

## Onglet Synergies

### §3.1 Lollipop W/L par carte

- Reference Python : `src/visualization/map_charts.py::plot_map_winrate_lollipop` (a localiser)
- Screenshot : `docs/screenshots/squad/3-1-lollipop-wl.png` **TODO**
- Type Plotly : composition `bar` (barre fine) + `scatter` (point a l'extremite)
- Composition : 2 traces par carte (Wins en vert, Losses en rouge) ou 1 trace stack ?
  - **A confirmer Phase 0bis** : extraire le JSON Plotly exact.

#### Encoding
- xAxis : `value` (count matches, type=`linear`)
- yAxis : `category` (cartes, ordre chronologique d'apparition session)
- traces : TODO Phase 0bis

#### Layout
- TODO Phase 0bis (margins, height, barmode si stack).

#### Comportements
- Empty (escouade < 2 cartes) : `st.info('tm_not_enough_matches')`.

#### JSON Plotly extrait
```json
TODO Phase 0bis
```

---

### §3.2 Bullet winrate session vs historique

- Reference Python : `src/visualization/map_charts.py::plot_map_winrate_bullet`
- Screenshot : `docs/screenshots/squad/3-2-bullet-winrate.png` **TODO**
- Type Plotly : `bar` 3 traces empilees (WR session, WR historique escouade, ligne joueur solo)

#### Encoding
TODO Phase 0bis.

---

### §3.3 Perf vs historique par carte

- Reference Python : `src/visualization/map_charts.py::plot_map_perf_vs_history`
- Screenshot : `docs/screenshots/squad/3-3-perf-vs-history.png` **TODO**
- Type Plotly : `bar` horizontales delta (`Δ performance_score`)

#### Encoding
TODO Phase 0bis.

---

### §3.4 Heatmap escouade joueur x carte

- Reference Python : `src/visualization/squad_map_heatmap.py::plot_squad_map_heatmap`
- Screenshot : `docs/screenshots/squad/3-4-heatmap-squad.png` **TODO**
- Type Plotly : `heatmap`

#### Encoding
- xAxis : `category` (cartes, top 20 ordre chrono)
- yAxis : `category` (joueurs : `[me, f1, f2, f3]`)
- z : matrice perf_score normalisee
- colorscale : voir `SQUAD_DESIGN_TOKENS.md` §3.1

#### Comportements
- Empty (< 2 joueurs) : ne rien afficher.

---

### §3.5 Timeline performance multi-joueurs

- Reference Python : `src/visualization/squad_performance_timeline.py::plot_squad_performance_timeline`
- Screenshot : `docs/screenshots/squad/3-5-timeline-perf.png` **TODO**
- Type Plotly : `scatter` mode `lines+markers`, 1 trace par joueur

#### Encoding
- xAxis : `date` (chronologique)
- yAxis : `value` (`performance_score` 0-100)
- traces : 1 par joueur, couleur = `attributePlayerColor(xuid)`, marker symbol = `outcomeMarkerSymbol(outcome)` par point

#### Comportements
- Empty : ne pas afficher si `len(series) < 2`.

---

### §3.6 Form Score lisse (LOWESS)

- Reference Python : `src/visualization/_form_score.py::plot_form_score_history`
- Screenshot : `docs/screenshots/squad/3-6-form-score.png` **TODO**
- Type Plotly : `scatter` mode `lines`, 2 traces (escouade lissee, joueur principal lisse)

#### Encoding
- LOWESS alpha : TODO Phase 0bis (probablement 0.3 vu le `_performance_form.DETAIL_THRESHOLD`)
- Bande de confiance ? TODO Phase 0bis

---

### §3.7 Impact 8 roles

- Reference Python : `src/ui/pages/teammates_impact.py::render_impact_taquinerie`
- Screenshot : `docs/screenshots/squad/3-7-impact-heatmap.png` + `3-7-impact-ranking.png` **TODO**
- Type :
  1. Heatmap roles x joueurs (`heatmap` Plotly avec `text` emojis sur cellules + fond outcome)
  2. Tableau ranking 8 colonnes (HTML custom, pas Plotly)

#### Heatmap encoding
- xAxis : `category` (matchs, ordre chrono)
- yAxis : `category` (joueurs)
- text : emojis concatenes par cellule (`_pivot_matrix_cells`)
- fond cellule : couleur outcome via `_OUTCOME_BG`

#### Tableau ranking
- 8 colonnes (1 par role) + colonne joueur en premiere position
- Cellules : compteur d'occurrence + gradient Okabe-Ito selon score
- **Roles inverses** (`_IMPACT_INVERTED`) : gradient inverse
- Source scores : `SCORE_SILENT_HERO`, `SCORE_FALSE_BROTHER`, `SCORE_TOP_KILLER` + comptes purs pour les autres

TODO Phase 0bis : extraire les seuils min/max du gradient.

#### Toggle viz
- `tmi_viz_heatmap` (defaut) / `tmi_viz_scatter` (alternative points/symboles)

---

### §3.8 Tableau historique escouade

- Reference Python : `src/ui/pages/teammates_helpers.py::render_friends_history_table`
- Screenshot : `docs/screenshots/squad/3-8-history-table.png` **TODO**
- Type : tableau HTML custom (classe `os-table`)

#### Colonnes
| # | Header | Source | Format | Alignement |
|---|--------|--------|--------|------------|
| 1 | Carte | `map_ui` | label localise + miniature | left |
| 2 | Mode | `pair_name` | `normalize_mode_label` | left |
| 3 | Playlist | `playlist_name` | `translate_playlist_name(lang)` | left |
| 4 | Date | `start_time` | `formatDate(date, locale, 'datetime')` | left |
| 5 | Resultat | `outcome` | badge color outcome + emoji | center |
| 6 | Waypoint | `match_id` | lien externe icone | center |

#### Tri par defaut
`start_time DESC` (matchs recents en haut).

#### Pagination
TODO Phase 0bis : Streamlit ne pagine pas, donc ? Probablement scroll vertical avec hauteur max. A decider en React (virtualisation `react-virtual` si > 100 lignes ?).

---

### §3.9 Cadence trio

- Reference Python : `src/visualization/squad_cadence_chart.py::plot_squad_cadence_profiles`
- Screenshot : `docs/screenshots/squad/3-9-cadence.png` **TODO**
- Type Plotly : `bar` empilees par phase 60 s

#### Encoding
- xAxis : `category` (phases : "0-60s", "60-120s", "120-180s", ...)
- yAxis : `value` (kills moyens normalisees)
- traces : 1 par joueur, couleur = `attributePlayerColor`

#### Note post-graphe
`tm_note_cadence` ("Des pics synchronises -> push coordonne, etc.")

---

## Onglet Contributions

### §4.1 Stats par minute groupees

- Reference Python : `src/ui/pages/_teammates_trio_helpers.py::_render_per_minute_stats`
- Screenshot : `docs/screenshots/squad/4-1-per-minute.png` **TODO**
- Type Plotly : `bar` `barmode=group`, 3 categories (frags/min, deaths/min, assists/min) x N joueurs

#### Encoding
- xAxis : `category` (3 metriques)
- yAxis : `value` ; **deaths affiches en negatif** (vers le bas) avec couleur `negativeColor(playerColor)`
- Labels textes : valeurs absolues (`textposition='auto'`)

#### Layout
- `barmode = 'group'`
- `height = 350`
- `margin = {l: 40, r: 20, t: 30, b: 80}`
- `showlegend = false` (legende dans panneau lateral)
- **Axe zero** : `yaxis.zeroline = true`, `zerolinecolor = 'rgba(255,255,255,0.75)'`, `zerolinewidth = 2`

#### Records overlay
Si `pm_records` fourni : ajouter traces fantomes hachurees (`applyRecordsOverlay`).

---

### §4.2 Radar 6 axes

- Reference Python : `src/ui/components/radar_chart.py::create_participation_profile_radar`
- Screenshot : `docs/screenshots/squad/4-2-radar.png` **TODO**
- Type Plotly : `scatterpolar` `fill='toself'`

#### Axes (6)
| # | Cle | Label FR | Label EN | Source |
|---|-----|----------|----------|--------|
| 1 | combat | Combat | Combat | `kill_score` normalise |
| 2 | survie | Survie | Survival | `1 / death_rate` |
| 3 | soutien | Soutien | Support | `assist_score` |
| 4 | score | Score | Score | `kill_score + assist_score` |
| 5 | objectifs | Objectifs | Objectives | `objective_score` (variable selon mode) |
| 6 | impact | Impact | Impact | `points_per_minute` |

#### Normalisation
- `RADAR_THRESHOLDS_PER_MODE` (Slayer / CTF / Strongholds / Oddball / Custom)
- Scaling par `n_matches` pour les axes absolus (combat, support, score)
- Pour `objectifs` : different selon `is_objective_mode_from_pair_name`

TODO Phase 0bis : extraire les `RADAR_THRESHOLDS` exacts.

#### Note post-graphe
`tm_note_radar`.

---

### §4.3 6 charts performance trio dedies

Liste : Frags/Morts combine, Assists, KDA Ratio, Accuracy, Avg Life, Performance Score.

Pour **chaque** des 6 charts :

#### Specs communes
- Type Plotly : `bar` `barmode=group`, 1 trace par joueur, x=match_id (categoriel chrono)
- Couleurs : `attributePlayerColor(xuid)`
- Records overlay si `show_records`
- Legende masquee (`hideLegend`)

#### Specs individuelles
| Chart | y_metric | y_format | y_suffix | y_title |
|-------|----------|----------|----------|---------|
| Frags/Morts | combine kills (positif) + deaths (negatif) | `.0f` | — | `tm_kills_deaths` |
| Assists | assists | `.0f` | — | `tm_assists` |
| KDA | ratio | `.3f` | — | `tm_kda` |
| Accuracy | accuracy | `.2f` | `%` | `tm_accuracy` |
| Avg Life | average_life_seconds | `.1f` | s | `tm_avg_life` |
| Performance | performance | `.1f` | — | `tm_performance` |

TODO Phase 0bis : screenshot + JSON pour chacun.

---

### §4.4 Killing Spree + HS/PK enrichis

#### Killing Spree (max)
- Reference Python : `plot_friend_metric` avec `metric_col='max_killing_spree'`, `smooth_window=10`
- Type Plotly : `bar` + `scatter` mode `lines` (smooth)
- Smoothing : moyenne mobile fenetre 10 matchs (TODO Phase 0bis : confirmer si MA simple ou EMA)

#### HS + PK stacked
- Reference Python : `src/visualization/teammates_hs_pk.py::plot_hs_pk_stacked`
- Type Plotly : `bar` `stack=true` 3 traces (Headshot, Perfect, Other)
- Records overlay si `hspk_records`

---

### §4.5 Heatmap intensite

- Reference Python : `src/visualization/match_intensity_heatmap.py::plot_match_intensity_heatmap`
- Screenshot : `docs/screenshots/squad/4-5-intensity.png` **TODO**
- Type Plotly : `heatmap`

#### Encoding
- xAxis : `category` (10 buckets de phase, 0-10%, 10-20%, ...)
- yAxis : `category` (matchs, label = carte si dispo sinon date via `prepareTimeAxis`)
- z : profil de kills normalise (0-1)
- colorscale : `SQUAD_DESIGN_TOKENS.md` §3.3

#### UI
- `SegmentedControl` au-dessus : `[Tous, joueur1, joueur2, joueur3]`
- Si `Tous` : agrege l'escouade
- Sinon : filtre `events_df` par `xuid` du joueur selectionne

#### Comportements
- Empty (< 3 matchs) : ne pas afficher.
- Empty profile (< 2 lignes apres calcul) : ne pas afficher.

---

### §4.6 First Events

- Reference Python : `src/ui/pages/teammates_charts.py::render_first_events_chart`
- Screenshot : `docs/screenshots/squad/4-6-first-events.png` **TODO**
- Type Plotly : `scatter` mode `markers`, X = position relative dans match (0-1 ou 0-100%), Y = match (categoriel)

TODO Phase 0bis : extraire JSON.

---

### §4.7 Tableau armes

- Reference Python : `src/ui/pages/teammates_weapons.py::render_weapon_kills_table` + variant multi-joueurs
- Screenshot : `docs/screenshots/squad/4-7-weapons-table.png` **TODO**
- Type : tableau HTML custom (classe `os-table`)

#### Colonnes
| # | Header | Source | Format | Alignement |
|---|--------|--------|--------|------------|
| 1 | Arme | `weapon_id` | `resolveWeaponDisplay(wid, lang)` | left |
| 2 | Faction | mapping faction par weapon_id | text ou "—" | left |
| 3 | Me | total_kills xuid_me | `formatNumber` | right |
| 4 | F1 | total_kills xuid_f1 | `formatNumber` | right |
| 5 | F2 | total_kills xuid_f2 (si present) | `formatNumber` | right |
| 6 | F3 | total_kills xuid_f3 (si present) | `formatNumber` | right |
| 7 | Total | sum | `formatNumber` | right |

#### Filtre
- Slider min kills : range `[0, max(total_kills)]`, step `1`, default `0`
- TODO Phase 0bis : confirmer `_TOP_N_WEAPONS = 12` dans le code Python (vu une fois)

#### Tri
- Defaut : `Total` desc
- Tris autorises : toutes colonnes numeriques, alphabetique sur Arme

#### Reinjection grenade/melee
- Filtrer `_FILM_EXCLUDED_IDS`
- Lire `match_participants.{grenade_kills, melee_kills}`
- Cap `remainder = api_total - film_kills`
- Ajouter lignes synthetiques `_GRENADE_WEAPON_ID`, `_MELEE_WEAPON_ID`

---

### §4.8 Barplot armes top 12

- Reference Python : `render_weapon_kills_bar_chart`
- Screenshot : `docs/screenshots/squad/4-8-weapons-bars.png` **TODO**
- Type Plotly : `bar` `barmode=group`, top 12 armes x N joueurs

#### Encoding
- xAxis : `category` (armes, top 12 par kills total)
- yAxis : `value` (kills)
- traces : 1 par joueur, couleur = `attributePlayerColor`

---

### §4.9 Galerie medailles

- Reference Python : `src/ui/pages/_teammates_trio_helpers.py::_render_trio_medals` -> `src/ui/medals.py::render_medals_grid`
- Screenshot : `docs/screenshots/squad/4-9-medals-gallery.png` **TODO**
- Type : composant React custom — grille de cartes match

#### Layout grille
- TODO Phase 0bis : decider nombre de colonnes (Streamlit utilise probablement 2-3 colonnes responsive)
- Top 20 matchs partages

#### Carte match
- Image carte (miniature centree avec gradient overlay)
- Date locale (top right)
- Liste joueurs escouade (top left, badges avatars)
- Icones medailles principales (centre bas, taille TODO)
- Badge outcome (W/L/T) avec couleur
- Lien Waypoint (icone in coin)

#### Comportements
- Empty : `t('tm_no_shared_medals')`.

---

## Conventions transverses (a appliquer dans toutes les fiches)

### Plotly config
- `responsive: true`
- `displayModeBar: false` (pas de toolbar)
- `displaylogo: false`
- `modeBarButtonsToRemove: ['lasso2d', 'select2d']`

### Layout commun
- `paper_bgcolor: 'rgba(0,0,0,0)'`
- `plot_bgcolor: 'rgba(0,0,0,0)'`
- Police : font CSS variable theme Halo
- `hovermode: 'closest'` (defaut) ou `'x unified'` (timelines)

### Empty state generique
Si donnees insuffisantes : afficher `<EmptyStateNotice>` (existant dans le code Go) avec titre + description i18n.

### Animation
- Plotly : `transition.duration = 0` (pas d'animation par defaut, evite flicker entre updates).

---

## Checklist de remplissage Phase 0bis

- [ ] App Streamlit `v7/cockpit` lancee localement avec dataset demo (≥ 200 matchs, ≥ 3 amis frequents).
- [ ] 22 screenshots produits dans `docs/screenshots/squad/`.
- [ ] 22 fiches completees (encoding + layout + tooltip + comportements + JSON Plotly).
- [ ] Conventions transverses confirmees (config, layout, empty, animation).
- [ ] Cas limite documentes (empty, single-player, multi-page session).
- [ ] Validation croisee avec l'audit `AUDIT_TEAMMATES_V7_COCKPIT.md` (pas d'oubli).
