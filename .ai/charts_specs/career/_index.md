# Page Carrière — composition

> Source Python : `src/ui/pages/career.py` sur `origin/v7/cockpit` (355 lignes).
> Fonction d'entrée : `render_career_page(db_path, xuid, db_key, waypoint_player)` (career.py:332).

## Vue d'ensemble

La page Carrière agrège **6 sections** chronologiquement organisées :
1. Rang actuel (header + gauge XP)
2. Progression vers le rang Héros (272)
3. Historique XP (timeline + snapshots)
4. LUSR/CSR (cards + timeline + filtres)
5. Top 10 meilleurs / pires matchs (2 tableaux)
6. Encounters (3 tableaux : adversaires, nemeses, victims)

Pas de filtre global de page — chaque section a éventuellement ses propres contrôles.

## Ordre d'affichage

| # | Élément | Type | Source | Spec YAML |
|---|---|---|---|---|
| 1 | Header rang | KPI cards (st.metric × 4 + image adornment) | `_render_rank_header` (career.py:107) | _composants standard, pas de YAML_ |
| 1.1 | Gauge progression rang | `go.Indicator` mode gauge+number | `create_career_progress_gauge` (career_progress_circle.py:79) | `01_rank_progress_gauge.yaml` |
| 2 | Hero metrics | KPI cards (st.metric × 4) | `_render_hero_section` (career.py:199) | _composants standard, pas de YAML_ |
| 2.1 | Gauge Héros | `go.Indicator` (identique 1.1) | `create_hero_progress_gauge` (career_progress_circle.py:~130) | `02_hero_progress_gauge.yaml` |
| 3.1 | Timeline XP | `go.Scatter` multi-traces (XP réel + estimé pré-sync + projections + autres joueurs) | `_create_xp_history_chart` (career_charts.py) | `03_xp_history_timeline.yaml` |
| 3.2 | Snapshots XP | Texte formaté dans `st.expander` (10 dernières lignes) | `_render_xp_snapshots_table` (career.py:233) | `10_xp_snapshots_table.yaml` |
| 4 | Cards LUSR | Grille 3 colonnes de cards HTML markdown (rang, tier, delta, badge type) | `_render_lusr_rank_cards` (career_lusr.py:32) | `11_lusr_rank_cards.yaml` |
| 4.1 | Timeline LUSR | `go.Scatter` lissé EWMA + marqueurs ▲/▼/◆ | `plot_lusr_timeseries` (timeseries_combat.py) | `04_lusr_timeline.yaml` |
| 5.1 | Top 10 meilleurs matchs | Tableau HTML scoreboard | `_build_top_table_html(best=True)` (career_top_matches_render.py:184) | `05_top_best_matches_table.yaml` |
| 5.2 | Top 10 pires matchs | Tableau HTML scoreboard | `_build_top_table_html(best=False)` (idem) | `06_top_worst_matches_table.yaml` |
| 6.1 | Top 10 encounters | Tableau HTML scoreboard | `build_encounters_table_html` (career_encounters_html.py) | `07_encounters_table.yaml` |
| 6.2 | Top 10 nemeses | Tableau HTML scoreboard (col gauche) | `build_antagonist_table_html(mode='nemesis')` | `08_nemeses_table.yaml` |
| 6.3 | Top 10 victims | Tableau HTML scoreboard (col droite) | `build_antagonist_table_html(mode='victim')` | `09_victims_table.yaml` |

**Note 3.2 — snapshots** : ce n'est PAS un tableau HTML mais un `st.expander` contenant 10 lignes `st.text(f"{date} | rank N: label | XP: x,xxx")`. Pour le portage, c'est plutôt un composant "log/history" qu'une table. Documenté dans le YAML 03 (timeline XP) en sous-section, pas en YAML séparé.

## Contrôles de page

| Control | Widget | Scope | Affecte |
|---|---|---|---|
| `lusr_period` | `segmented_control` | section LUSR | timeline LUSR (4.1) — filtre `start_time >= now - days(period)` + truncate granularité |
| `lusr_group_select` | `selectbox` | chart 4.1 | filtre `playlist_group` (ranked, arena, btb, tactical, social, fun, ou tous) |
| `encounters_period` | `segmented_control` | section Encounters (6.x) | filtre `since` pour les 3 tableaux d'encounters |

Aucun contrôle global de page — chaque section gère ses propres filtres.

## KPI metrics (composants UI hors charts)

Pour traçabilité. Documentés ici plutôt que dupliqués dans les YAML :

### Section 1 (rang actuel)
| Metric key | Label i18n | Format |
|---|---|---|
| `career_metric_rank` | "Rang" | `"{rank} / 272"` |
| `career_metric_xp_total` | "XP total" | `"{xp_total:,}"` |
| `career_metric_current_xp` | "XP actuel" | `"{current_xp:,}"` |
| `career_metric_next_rank_xp` | "XP prochain rang" | `"{xp_for_next:,}"` |

### Section 2 (Hero)
| Metric key | Label i18n | Format |
|---|---|---|
| `career_metric_xp_earned` | "XP gagné" | `"{xp_total:,}"` |
| `career_metric_xp_remaining` | "XP restant" | `"{xp_remaining:,}"` |
| `career_metric_xp_required` | "XP requis" | `"9,319,350"` |
| `career_metric_rank` | "Rang" | `"{rank} / 272"` |

## Cards LUSR (composants HTML hors charts)

3-6 cards selon nombre de playlists actives (ranked, arena, btb, tactical, social, fun).

Pour chaque card :
- **Header** : icône emoji (🏆 ranked, ⚔️ arena, 💥 btb, 🎯 tactical, 🎮 social, 🎉 fun) + label playlist
- **Image rang** : 90×90 PNG depuis `get_rank_image_path(rating_value)`
- **Tier label** : nom du palier (Bronze/Silver/Gold/Onyx/Champion ou équivalent CSR)
- **Badge type** : `LUSR` (cyan #00B7EB) ou `CSR` (gold #FFD700)
- **Valeur** : rating arrondi (`{r_value:.0f}`)
- **Delta** : si `rating_delta` non null → arrow ▲/▼ + `+/-N` en vert/rouge

Layout : `st.columns(3)` × N rangées. Chaque card rendue via `st.container(border=True)` + `st.markdown(unsafe_allow_html=True)`.

Côté Go/React → composant `<LusrCard>` réutilisable.

## États vides

| Section | Condition | Affichage |
|---|---|---|
| Toute la page | `career_data is None` | `st.info(t("career_no_data"))` |
| Section 1 (rang) | gauge fig is None | `st.info(t("career_gauge_generate_error"))` |
| Section 3 (XP history) | `history` vide | `st.info(t("career_computing"))` |
| Section 3 (XP history) | history_fig is None | `st.info(t("career_rank_history_no_data"))` |
| Section 4 (LUSR) | `snapshot` vide | `st.info(t("career_lusr_no_rating"))` |
| Section 4 (chart) | `history_all` vide | retour silencieux (pas de subheader affiché) |
| Section 4 (chart) | `available_groups` vide | retour silencieux |
| Section 5 (top) | aucun match | (à confirmer dans top_matches_data) |
| Section 6 (encounters) | `encountered` vide | `st.info(t("career_encounters_no_data"))` |
| Section 6 (nemeses/victims) | les 2 vides | `st.info(t("career_antagonists_no_data"))` |
| Section 6 (nemeses) | nemeses vide mais victims non | `st.info(t("career_antagonists_no_data"))` dans col gauche |

## Configs Plotly par section

| # | Config |
|---|---|
| 1.1 (gauge rang) | `PLOTLY_STATIC_CONFIG` |
| 2.1 (gauge Héros) | `PLOTLY_STATIC_CONFIG` |
| 3.1 (timeline XP) | `PLOTLY_CLEAN_CONFIG` |
| 4.1 (timeline LUSR) | `PLOTLY_CLEAN_CONFIG` |

Tableaux 5.x / 6.x : pas de Plotly, rendu HTML markdown direct.

## Pré-condition côté DataFrame / DB

La page Career consomme directement le `db_path` (DuckDB du joueur) et le `xuid`. Tables sollicitées :

| Table | Section |
|---|---|
| `career_progression` | 1, 2, 3 (history XP, rang courant) |
| `metadata.career_ranks` (référentiel) | 1, 2, 3 (labels rangs) |
| `match_skill_rank` (player DB) | 4 (history LUSR/CSR par match) |
| `shared.match_participants` (avec join match_skill_rank) | 4, 5 (top matches stats) |
| `shared.killer_victim_pairs` | 6 (encounters/nemeses/victims) |
| `shared.xuid_aliases` | 6 (gamertag des adversaires) |

Pas de filtre global de période sur la page → chaque section requête en propre selon ses contrôles.
