# PARITY_MATRIX.md — Fiches écran et matrice de parité

> Pour chaque surface : référence Streamlit, invariants à préserver, état à externaliser, stratégie API, priorité.
> Source : PLAN_MIGRATION_FASTAPI_REACT.md § Étape critique 2
> Ce fichier est la référence fonctionnelle pendant toute la migration.
>
> **Aligné sur les sections V7 réelles** — voir [FUNCTIONAL_SPECS.md](FUNCTIONAL_SPECS.md) pour les spécifications fonctionnelles exhaustives.

---

## Matrice de parité (vue synthétique par section V7)

| Section V7 | Anciennes pages Streamlit regroupées | Invariants clés | État à externaliser | Type d'API | Priorité |
|---|---|---|---|---|---|
| Setup / Onboarding | `setup_wizard.py`, `setup_smoke_test.py` | gating, auth modes, provisioning, smoke test | `_setup_mode`, `_xbox_oauth_result`, `_smoke_*` | commandes + jobs + session auth | **MVP P0** |
| Settings [§8] | `settings.py` | écriture app_settings, hints, media, Discord, backfill | `app_settings`, `setting_*`, browser prefs | CRUD settings + actions serveur | **MVP P0** |
| **Profil** [§7] | `career.py` + `career_*` + `citations.py` | XP, rang, Hero, LUSR, top matches, encounters, commendations, médailles | presque rien hors contexte global | mix data brute + Plotly JSON + agrégats | **MVP P1** (Carrière) / **P2** (Citations) |
| **Stats** [§2] | `match_history.py` + `timeseries.py` + `_timeseries_*` + `session_compare.py` + logic/charts | lignes, tri, score, win rate hist, perf relative, CSV, séries, cumul, EWMA, sélection A/B | `compare_session_a/b`, `_last_picked_for_compare` | data brute paginée + Plotly JSON | **MVP P1** (Historique) / **P3** (Séries, Sessions) |
| **Explorer** [§5] | `explorer.py` + `explorer_*` + `match_view.py` + `match_view_*` + `last_match.py` | fuzzy search, cascade, deep links, scoreboards, tabs, rang, armes, citations, prev/next | `_pending_*`, `match_id_input`, `_last_match_nav_*` | search + payload composé + scope filtré | **MVP P1** |
| **Accueil** [§1] | `home_mission_control.py` + logic/api | highlights, battle pass, défis, dernier match, médias récents | battle pass focus, deep links stats/scope | agrégat multi-endpoints + **live API** | **P2** |
| **Escouade** [§3] | `teammates.py` + `teammates_*` | sélection mates, synergies, impact, armes, PersonalScores, multi-DB/shared | `teammates_picked_labels`, scope sessions | APIs spécifiques teammates | **P2** |
| **Synthèse** [§4] | `synthesis.py` + `objective_analysis.py` | comparatif solo/escouade, fenêtre période, awards objectifs | `synthesis_period` | data brute + figures | **P3** |
| **Médias** [§6] | `media_v2.py` + `media_v2_*` | index local, filtres, groupes, lightbox, likes localStorage | `_lb_state`, `mv2_autoplay`, `_pending_match_id` | data brute media + thumbs + prefs locales | **P2** |
| ~~Win/Loss autonome~~ | `win_loss.py` | absorbé dans Stats/Synthèse | n/a | — | **Ne pas migrer** |
| ~~Media legacy~~ | `media_tab.py` / `media_library.py` | absorbé par Médias V2 | n/a | — | **Ne pas migrer** |
| ~~Objective Analysis~~ | `objective_analysis.py` | absorbé dans Escouade (radar) + Synthèse | n/a | — | **Absorbé** (voir note ci-dessous) |

> **Note Objective Analysis** : cette ancienne page autonome (P4) n'est pas migrée comme section V7 distincte.
> Les awards objectifs (PersonalScores API) sont consommés par le radar complémentarité de l'Escouade (§3.7.2).
> Si des trends objectifs spécifiques sont nécessaires, ils trouvent place dans Synthèse.

---

## Fiches écran détaillées

> Les fiches sont regroupées par section V7. Chaque fiche conserve le détail de parité par ancienne page
> Streamlit pour faciliter l'implémentation et les tests.

### Section V7 : Setup / Onboarding + Settings (Slice 1)

#### Setup Wizard + Smoke Test

- **Référence Streamlit** : `src/ui/pages/setup_wizard.py`, `src/ui/pages/setup_smoke_test.py`, `streamlit_app.py` avant tout dispatch principal
- **Rôle métier** : bloquer l'app si la configuration est incomplète, permettre le choix du mode d'auth, provisionner le joueur, lancer une sync initiale, faire un smoke test, puis seulement ouvrir le produit
- **Sources de données / logique** : `setup_wizard_logic.py`, `setup_wizard_xbox.py`, auth provider, création du player profile, `smoke_test_logic.py`
- **État Streamlit à externaliser** : `_setup_mode`, `_xbox_oauth_result`, `_smoke_gamertag`, `_smoke_db_path`, `_setup_smoke_completed`, `_smoke_test_done`, `_smoke_test_result`
- **Règle de parité** : les mêmes gates bloquent l'accès à l'app, les mêmes étapes sont franchies, les mêmes checks d'intégrité passent ou avertissent
- **Stratégie React** : flow dédié d'onboarding avec session backend + endpoint de job status pour sync/backfill/smoke test
- **Priorité** : **P0 absolu**

---

### Settings

- **Référence Streamlit** : `src/ui/pages/settings.py`
- **Rôle métier** : piloter la langue, timezone, options de backfill, options d'affichage, media watcher/index, notifications Discord, reset index media
- **Sources de données / logique** : `AppSettings`, `patch_settings`, `_write_settings`, `MediaIndexer.reset_media_tables`, browser storage pour lang/show_hints
- **État Streamlit à externaliser** : `app_settings`, `setting_*`, `_hints_visible`, préférences navigateur associées
- **Règle de parité** : toute option modifiée doit produire le même effet serveur qu'aujourd'hui, sans dépendre d'un rerun Streamlit
- **Stratégie React** : formulaires typed + endpoints PATCH settings + actions explicites pour rescan/reset media
- **Priorité** : P0, car bootstrap et exploitation

---

### Section V7 : Profil — Carrière + Citations (Slice 2)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 7 Profil](FUNCTIONAL_SPECS.md#7-profil)

#### Carrière (Phase A — MVP P1)

- **Référence Streamlit** : `src/ui/pages/career.py`, `career_data.py`, `career_logic.py`, `career_lusr.py`, `career_top_matches_render.py`, `career_encounters_render.py`
- **Rôle métier** : progression carrière, jauges XP, progression Hero, historique XP, LUSR, top matches, encounters
- **Sources de données / logique** : `career_progression`, `mv_player_matches`, rang metadata, projections XP, logique LUSR, top matches data
- **État Streamlit à externaliser** : quasi nul hors contexte global ; attention à l'usage de `app_settings` dans certains sous-rendus
- **Règle de parité** : même rang courant, même XP total, même progression Hero, mêmes projections, même LUSR, même top matches, mêmes encounters
- **Stratégie React** : très bon premier slice ; réexposer les KPIs et figures en API sans changer les calculs Python
- **Priorité** : P1

---

#### Citations (Phase B — Post-MVP P2)

- **Référence Streamlit** : `src/ui/pages/citations.py`
- **Rôle métier** : afficher les commendations H5G et les médailles Halo Infinite sur le scope filtré
- **Sources de données / logique** : `CitationEngine`, `match_citations`, `top_medals_fn`, référentiels médailles, distribution Plotly
- **État Streamlit à externaliser** : peu de state local ; dépendance surtout au scope filtre et aux caches du moteur de citations
- **Règle de parité** : mêmes totaux de citations, mêmes médailles, mêmes deltas filtre vs complet, même grille et même distribution
- **Stratégie React** : endpoint agrégat brut + éventuellement figure Plotly JSON pour la distribution
- **Priorité** : P2

---

### Section V7 : Stats — Séries + Sessions + Historique (Slice 3)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 2 Stats](FUNCTIONAL_SPECS.md#2-stats)

#### Historique des parties (Phase A — MVP P1)

- **Référence Streamlit** : `src/ui/pages/match_history.py`, `match_table_html.py`
- **Rôle métier** : table complète des matchs filtrés, enrichie avec score, map/mode/playlist, win rate historique, performance relative, CSV export
- **Sources de données / logique** : `dff` + `df_full`, `compute_performance_series`, traductions, vectorisation Polars
- **État Streamlit à externaliser** : aucun state local critique hors filtres globaux ; export CSV à recâbler en HTTP
- **Règle de parité** : même nombre de lignes, même ordre, mêmes valeurs calculées, même tri/filtrage sémantique, même export
- **Stratégie React** : endpoint paginé/raw data + grille React riche ; très bon candidat MVP
- **Priorité** : P1

---

#### Séries temporelles (Phase B — Post-MVP P3)

- **Référence Streamlit** : `src/ui/pages/timeseries.py` + modules `_timeseries_*` + win_loss helpers réutilisés
- **Rôle métier** : lecture analytique temporelle du joueur via KPIs, KDA, cumul, forme récente, intensité, distributions, corrélations, weapon kills, personal scores
- **Sources de données / logique** : `TimeseriesService`, win_loss helpers, nombreux modules Plotly, `downsample_for_plot`
- **État Streamlit à externaliser** : dépend surtout des filtres globaux ; lecture locale de `filter_mode`
- **Règle de parité** : mêmes séries, mêmes agrégats, mêmes seuils de downsampling, mêmes annotations et mêmes onglets logiques
- **Stratégie React** : conserver les calculs et figures côté Python au début ; livrer du Plotly JSON
- **Priorité** : P3

#### Comparaison de sessions (Phase C — Post-MVP P3)

- **Référence Streamlit** : `src/ui/pages/session_compare.py`, `session_compare_logic.py`, `session_compare_charts.py`, `_session_compare_*`
- **Rôle métier** : comparer deux sessions et les replacer dans un contexte historique similaire
- **Sources de données / logique** : `cached_compute_sessions_db`, friends mapping, `build_skill_series`, `compute_historical_context`
- **État Streamlit à externaliser** : `compare_session_a`, `compare_session_b`, `_last_picked_for_compare`
- **Règle de parité** : mêmes sessions candidates, même choix par défaut, mêmes deltas et comparaisons
- **Stratégie React** : après avoir sorti proprement le modèle de sessions et les filtres URL-first
- **Priorité** : P3

---

### Section V7 : Explorer — Filtres + Match View + Last Match (Slice 4)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 5 Explorer](FUNCTIONAL_SPECS.md#5-explorer)

#### Explorer (Phase A — MVP P1)

- **Référence Streamlit** : `src/ui/pages/explorer.py`, `explorer_logic.py`, `explorer_data.py`, `explorer_results.py`
- **Rôle métier** : rechercher un joueur ou un match, filtrer des rencontres via cascade, ouvrir un détail de match, supporter les deep links
- **Sources de données / logique** : `get_all_gamertags`, `resolve_gamertag_to_xuid`, `fuzzy_search_gamertags`, `classify_experience_type`, `load_is_with_friends`
- **État Streamlit à externaliser** : `_pending_gamertag`, `_pending_match_id`, `match_id_input`, `_explorer_selected_match`, pagination locale des tables de résultats
- **Règle de parité** : mêmes résultats de recherche floue, mêmes filtres cascade, mêmes match_ids ciblés, même comportement de deep link
- **Stratégie React** : route URL-first + endpoints search / lookup / filtered results ; très bon slice après Match History
- **Priorité** : P1

---

#### Match View (Phase B — MVP P1)

- **Référence Streamlit** : `src/ui/pages/match_view.py` + famille `match_view_*.py`
- **Rôle métier** : détail complet d'un match, avec header score/rang/carte, onglets Résumé/Combat/Équipe/Médias/Citations
- **Sources de données / logique** : `match_view_logic.py`, cached loaders injectés via `MatchViewParams`, `match_view_tabs.py`, `match_view_weapon_kills.py`, scoreboards, nemesis, timeline
- **État Streamlit à externaliser** : `match_id` de route, callbacks injectés aujourd'hui via `match_view_params`, quelques flags de debug et de navigation venant des pages parentes
- **Règle de parité** : même score, même roster, mêmes médailles, mêmes events, mêmes armes, même rang, mêmes citations
- **Stratégie React** : route `/matches/:id`, payload détaillé composé au backend, figures JSON quand utile, pas de réimplémentation métier dans le front
- **Priorité** : P1

---

#### Last Match (Phase C — MVP P1.5)

- **Référence Streamlit** : `src/ui/pages/last_match.py`
- **Rôle métier** : wrapper de Match View sur le dernier match du scope courant, avec navigation précédent/suivant
- **Sources de données / logique** : `dff` filtré courant + `_resolve_nav_index()`
- **État Streamlit à externaliser** : `_last_match_nav_index`, `_last_match_nav_total`, `_last_match_nav_session_key`
- **Règle de parité** : même match courant selon le scope filtré, même logique de reset quand les filtres changent, même navigation prev/next
- **Stratégie React** : ne pas en faire une API distincte ; le considérer comme une vue dérivée de Match View + liste filtrée
- **Priorité** : P1.5, avec Match View

---

### Section V7 : Médias — Galerie + Lightbox (Slice 8)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 6 Médias](FUNCTIONAL_SPECS.md#6-médias)

#### Media V2

- **Référence Streamlit** : `src/ui/pages/media_v2.py`, `media_v2_filters.py`, `media_v2_grid.py`, `media_v2_likes.py`
- **Rôle métier** : bibliothèque locale de captures, groupement par auteur/date/carte/mode/session/experience/liked, lightbox, likes persistants, renvoi vers match
- **Sources de données / logique** : `MediaIndexer.load_media_for_ui`, table d'enrichissement media, likes navigateur, jointure avec `df_full`
- **État Streamlit à externaliser** : `_lb_state`, `mv2_autoplay`, `_pending_page`, `_pending_match_id`, états de filtres, persistence likes
- **Règle de parité** : mêmes médias, même regroupement, mêmes filtres, même navigation vers le match, même persistence des likes
- **Stratégie React** : endpoints raw media + URLs/paths de thumbnails + persistence locale navigateur ; forte refonte UI autorisée, logique métier stricte
- **Priorité** : P2

---

### Section V7 : Accueil — Home Mission Control (Slice 5)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 1 Accueil](FUNCTIONAL_SPECS.md#1-accueil-home-mission-control)

#### Home Mission Control

- **Référence Streamlit** : `src/ui/pages/home_mission_control.py`, `home_mission_control_logic.py`, `home_mission_control_api.py`, battlepass/challenges modules
- **Rôle métier** : accueil V7 composé de hero, highlights, KPIs, quick actions, résumé sessions, activité récente, battle pass, défis, dernier match, médias récents
- **Sources de données / logique** : summaries de sessions, recent matches/media, live APIs battlepass/challenges avec cache process-level, embed de Last Match
- **État Streamlit à externaliser** : `v7_current_section`, `_v7_pages`, `stats_view/session/scope/match_id` en deep link, état du navigateur battle pass, prefetch home progressions
- **Règle de parité** : mêmes highlights, même ordre des matchs récents, mêmes KPIs, même contenu battle pass/défis, mêmes CTA contextuels
- **Stratégie React** : route composée qui agrège plusieurs endpoints ; à traiter une fois Career, Match View et Media déjà exposés
- **Priorité** : P2

---

### Section V7 : Escouade — Synergies + Contributions (Slice 6)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 3 Escouade](FUNCTIONAL_SPECS.md#3-escouade)

#### Teammates

- **Référence Streamlit** : `src/ui/pages/teammates.py` + sous-modules `teammates_*`
- **Rôle métier** : analyser les performances avec 1 à 3 coéquipiers, synergies, impact, intensité, armes, radars et vues duo/trio
- **Sources de données / logique** : `TeammatesService`, shared + éventuelles DB joueurs, enrichissement perfect kills, `build_teammates_opts_map`, `base_s_ui`
- **État Streamlit à externaliser** : `teammates_picked_labels`, `_cache_warning_shown`, scope de sessions solo/escouade, `show_records`
- **Règle de parité** : mêmes coéquipiers proposés, mêmes stats par duo/trio, mêmes radars, mêmes enrichissements par armes et impact
- **Stratégie React** : slice tardif et lourd ; nécessite des contrats API spécifiques et une clarification définitive du modèle multi-joueur
- **Priorité** : P3 critique en complexité

---

### Section V7 : Synthèse — Solo vs Escouade (Slice 7)

> **Ref specs** : [FUNCTIONAL_SPECS.md § 4 Synthèse](FUNCTIONAL_SPECS.md#4-synthèse)

#### Synthesis

- **Référence Streamlit** : `src/ui/pages/synthesis.py`
- **Rôle métier** : vue d'ensemble solo vs escouade, période locale, synthèse stratégique en réemploi de briques analytiques existantes
- **Sources de données / logique** : `KPIStats`, `load_is_with_friends`, win_loss helpers, comparaison solo/squad
- **État Streamlit à externaliser** : `synthesis_period`
- **Règle de parité** : mêmes fenêtres temporelles, même découpage solo/escouade, mêmes agrégats et mêmes chartes de comparaison
- **Stratégie React** : bon écran de consolidation V7 une fois le shell web stabilisé
- **Priorité** : P3

---

#### ~~Objective Analysis~~ (absorbé)

> **Absorbé** : cette ancienne page autonome (P4) n’est pas migrée comme section V7 distincte.
> Les awards objectifs (PersonalScores API) sont consommés par le **radar complémentarité**
> de l’Escouade (§3.7.2). Les trends objectifs trouvent place dans Synthèse si nécessaire.

- **Référence Streamlit** : `src/ui/pages/objective_analysis.py`
- **Rôle métier** : valoriser les awards objectifs, le profil support/slayer, les trends objectifs et corrélations avec les kills
- **Sources de données / logique** : `personal_score_awards`, `objective_participation`, `objective_charts`, `mv_player_matches`
- **État Streamlit à externaliser** : wrapper `from_session_state` et dépendances implicites au joueur courant
- **Règle de parité** : mêmes points objectifs, même ratio, même classification support/polyvalent/slayer, mêmes breakdowns et trends
- **Stratégie React** : feature annexe ou future page d'analyse ; ne pas prioriser avant les parcours centraux
- **Priorité** : P4

---

## Corpus de référence à constituer (avant Slice 0)

> Voir aussi `SLICES.md` § Prérequis avant Slice 0 pour la structure de fichiers.

Le corpus doit être constitué **avant** d'écrire du code de migration. Il conditionne la validité de toute la stratégie de parité.

### Joueur de référence
- Un joueur avec historique stable, non en sync active pendant la migration
- DBs copiées dans `tests/fixtures/ref_player/` — jamais écrasées par une sync réelle

### Scopes à figer
- Période complète (aucun filtre)
- Période courte aux dates fixes (ex : 2025-01-01 → 2025-03-31)
- Une session solo nommée connue
- Une session escouade nommée connue
- Un filtre playlist + mode connu

### Match IDs de référence
5 à 10 matchs couvrant : victoire ranked, défaite BTB, match avec bot, match avec armes inhabituelles, match avec citations.

### Golden values à capturer depuis Streamlit maintenant
Pour chaque scope figé, noter avant de commencer :

| Valeur | Scope | Attendu |
|--------|-------|---------|
| Rang courant | — | à remplir |
| XP total | — | à remplir |
| LUSR rating | — | à remplir |
| Nombre de lignes Match History | Période complète | à remplir |
| Score match de référence | match_id X | à remplir |
| Roster taille équipe match réf. | match_id X | à remplir |
| Nombre de médailles match réf. | match_id X | à remplir |

Ces valeurs deviennent les golden values des tests `tests/parity/`.

---

## Jeu minimal de tests de parité à préparer (par section V7)

- **Profil** [§7] — Carrière : rang, XP total, progression Hero, LUSR, top matches / Citations : totaux, médailles, deltas
- **Stats** [§2] — Historique : cardinalité, ordre, tri, filtres, match_id ciblés / Séries : mêmes agrégats, onglets / Sessions : sélection A/B, deltas
- **Explorer** [§5] — Filtres : cascade, fuzzy search / Match View : score, roster, tabs, armes, rang, citations / Last Match : prev/next
- **Accueil** [§1] — Home : matchs récents, sessions, battle pass, défis
- **Escouade** [§3] — Teammates : mêmes mates, mêmes radars, mêmes stats impact, armes
- **Synthèse** [§4] — Solo/Escouade : agrégats, heatmap, fenêtres temporelles
- **Médias** [§6] — Media : cardinalité, groupements, likes, navigation vers match, lightbox
- **Setup / Settings** [§8] — mêmes gates, mêmes side effects de configuration, même résultat smoke test
