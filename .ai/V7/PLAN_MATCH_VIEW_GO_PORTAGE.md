# Plan de portage de la Match View — Python v7/cockpit -> Go + Next.js

> Audit comparatif rigoureux et plan de portage par phases.
> Branche source : `v7/cockpit` (Streamlit/Python).
> Branche cible : `feat/multi-title-adapters-and-mappings` (Go + Next.js + recharts).
> Date d'audit : 2026-04-26 (audit) + faisabilité 2026-04-26.
>
> **OBJECTIF FONDAMENTAL** : la page Go résultante doit être **quasi ISO** avec la version v7/cockpit. Chaque visualisation, KPI, badge, tooltip et interaction listés dans ce plan doit avoir son équivalent fonctionnel à la fin du portage. Toute simplification / dégradation doit être documentée et justifiée explicitement (cf. blockers § 5).

> **Note d'amendement — 2026-04-27** : ce plan est **partiellement supersedé** par
> [`PLAN_META_FOUNDATIONS_GO.md`](./PLAN_META_FOUNDATIONS_GO.md). Avant toute
> implémentation à partir de ce plan, consulter le méta-plan pour les fondations
> communes : helpers `analysis/{temporal,breakdown,narrative}`, type partagé
> `canonical.PlayerMatchRow`, stack chart **ECharts** (Plotly et Recharts retirés),
> manifest i18n centralisé. La pile chart cible n'est plus `recharts` mais ECharts.
> Réécriture complète prévue en **Phase 1** du méta-plan (Squad + MatchView pilotes).

### Statut des sections de ce plan vis-à-vis du méta-plan

| Section / Phase | Statut | Action |
|---|---|---|
| Phase A — Header dominance + map thumbnail | À refactorer | Badge dominance via `analysis/narrative.ResolveDominanceBadge` ; map thumbnail via `TitleSemanticAdapter`. |
| Phase B — KPIs MMR + scoreboard | À conserver | Spécifique Match View. |
| Phase C — Tug-of-war + cadence | À refactorer | Wrappers `<BarStacked>` et `<Cadence>` ECharts (méta-plan § 5.2.2). |
| Phase D — Radar Participation 6 axes | À refactorer | `analysis/narrative.ComputeParticipationProfile` + `<Radar>` ECharts. |
| Phase E — Donut weapons + Spree HS | À refactorer | `<Donut>` + `<BarStacked>` ECharts. |
| Phase F — Encounters + badges | À refactorer | `LoadPlayerMatches` + `analysis/narrative.ComputeEncounterBadges`. |
| Phase G — Citations tab | À refactorer | Voir `PLAN_CITATIONS_GO_PORTAGE.md` (lui aussi amendé). |
| Phase H — Kill feed + cadence + impact timeline | À refactorer | `<TimeseriesLine>` + `<Cadence>` ECharts ; impact via `analysis/narrative.IdentifyImpactRoles` (8 rôles). |
| Phase I — Charts cleanup (stubs `PlotlyFigurePayload`) | Obsolète | Supersédé par méta-plan § 5.2.1 (`ChartSeries[T]`). |
| Phase J — `personal_score_awards` lecture Go | À conserver | **Prérequis bloquant Phase 1 méta-plan** — à boucler en Phase 0. |
| Phase K — Cleanup final | À conserver | Spécifique Match View. |

---

## 0. Synthèse exécutive

La match view actuelle en Go expose **5 onglets** alignés structurellement avec la version Python v7/cockpit, mais le contenu de chaque onglet est extrêmement appauvri : sur les **~30 visualisations distinctes** identifiées dans la version Python, la version Go n'en implémente correctement qu'**environ 8** (certaines sous forme dégradée). Plusieurs sections backend sont câblées en dur sur des listes vides (`citations_tab.commendations`, `citations_tab.medals`, `team_tab.roster`, `combat_tab.nemesis_duels`).

**Écart par onglet** :

| Onglet      | Viz Python | Viz Go (présent) | Viz Go (manquant) |
|-------------|-----------:|-----------------:|------------------:|
| Header      | 7 blocs    | 5 blocs          | 2 (map thumbnail, bloc Performance grand format, bloc Rang visuel complet — partiellement présent) |
| Summary     | 10         | 5 KPI + 2 listes | 4 charts majeurs (F/D/A grouped, Spree/HS/Perfect, Donut weapons, Radar Participation 6 axes) + tableau weapons + KPI MMR |
| Combat      | 9          | 4 (dont 2 charts simplistes) | Tug-of-war stacked, kill feed avec streaks, cadence histogram + MA, cards Nemesis/Bully visuelles, stacked bars KV, K/D timeline tous joueurs, annotations d'impact sur la timeline |
| Team        | 5          | 4 (dont scoreboard partiel) | Tableau Encounters historique, expansion détail pour TOUS joueurs, Expected dans détail, Antagonist dans détail, popover légende |
| Citations   | 2          | 2 listes vides   | Backend complet + anneaux progression CSS + grille médailles avec icônes |
| Media       | 1 grille   | 1 grille basique | Sections mine/teammate + lecteur vidéo intégré |

**Conclusion** : la match view Go est un MVP très sommaire. Le portage doit aboutir à une page **quasi ISO** avec v7/cockpit. L'audit de faisabilité (§ 6) montre que **33/35 viz (94%) sont implémentables sur l'architecture actuelle** (12 ne demandent même aucun travail backend). Restent **2 blockers** documentés en § 8 :

- 🔴 **#16 Radar Participation** : `personal_score_awards` jamais lue côté Go — port complet à faire (Phase J, ~1 semaine)
- 🔴 **#28 Panneau détail joueur** : architecture Go mono-player-DB → Local Performance/SkillRank des autres joueurs inaccessibles. **Workaround retenu** : dégrader gracieusement (les autres joueurs voient Armes+Médailles+Expected+Antagonist mais pas Local — chantier multi-player-DB hors scope)

Le plan est ordonné en **11 phases (A→K)** par effort et risque croissants : Phase A produit un gain visible immédiat sans toucher au backend ; Phase J (Radar) est gardée pour la fin. Effort total estimé **~35-45 jours-homme backend** + travail front en parallèle.

---

## 1. Cartographie source (v7/cockpit)

### 1.1 Fichiers Python (17 fichiers, ~150 KB)

```
src/ui/pages/match_view.py                       (entry, 426 L)
  +- src/ui/pages/match_view_helpers.py          (KPI cards + media section)
  +- src/ui/pages/match_view_logic.py            (resolve_outcome, enrich_pm)
  +- src/ui/pages/match_view_rank.py             (header rank LUSR/CSR)
  +- src/ui/pages/match_view_tabs.py             (orchestrateur des 5 onglets)
        |
        +- match_view_charts.py                  (Summary — Expected vs Actual + Spree/HS/Perfect)
        +- match_view_weapon_kills.py            (Summary — Donut + tableau)
        +- match_view_participation.py           (Summary — Radar 6 axes)
        |
        +- match_view_players.py                 (Combat — impact badges + impact timeline)
        +- match_view_players_data.py            (helpers chargement)
        +- match_view_players_nemesis.py         (Combat — Nemesis cards + KV stacked)
        +- match_view_players_timeline.py        (Combat — dominance + cadence + KD timeline)
        |
        +- match_view_scoreboard.py              (Team — scoreboard table)
        +- match_view_scoreboard_detail.py       (Team — panneau détail dépliable)
        +- match_view_encounters.py              (Team — tableau historique encounters)
        +- match_view_encounters_logic.py        (badges Allié+/Coriace/Tough nut)
        |
        +- match_view_citations.py               (Citations + médailles)
```

### 1.2 Pipeline de données

- DB centrales lues : `shared_matches_v2.duckdb` (`match_participants`, `medals_earned`, `killer_victim_pairs`, `highlight_events`, `v_weapon_kills`)
- DB par joueur lue : `data/players/<gt>/stats.duckdb` (`player_match_enrichment`, `personal_score_awards`, `match_skill_rank`, `match_citations`, `media_files`, `media_match_associations`)
- Référentiels : `metadata.duckdb` (`weapon_labels`, `mode_pair_overrides`, `mode_name_tr`)
- Définitions citations : YAML/JSON externes via `load_citation_definitions()`
- Assets : icônes médailles, rangs, armes (locales encodées en data-URI)

---

## 2. Cartographie cible (Go + Next.js)

### 2.1 Frontend (5 fichiers, 1081 lignes)

```
apps/web/src/features/match-view/
  +- MatchViewPage.tsx        (579 L — orchestrateur tabs + recharts inline)
  +- MatchScoreboard.tsx      (179 L — tableau scoreboard avec MVP/LVP + expansion)
  +- MatchStatCards.tsx       (182 L — Expected cards + Rank badge + KD nemesis)
  +- PlayerDetailPanel.tsx    (114 L — détail joueur sous expansion scoreboard)
  +- queries.ts               (27 L — useMatchView + useMatchNeighbors)
```

Routing TanStack Router : `apps/web/src/routes/players/$playerSlug/matches/$matchId.tsx`.

### 2.2 Backend Go

```
apps/go-api/internal/api/handlers/match_view.go             (handlers HTTP)
apps/go-api/internal/service/match_view_service.go          (orchestrateur 11 sous-requêtes errgroup)
apps/go-api/internal/platform/duckdb/match_view_repo.go     (impl repository DuckDB)
apps/go-api/internal/domain/match_view.go                   (DTO MatchViewResponse)
apps/go-api/internal/analysis/                              (ComputeTugOfWar, ComputeSingleMatchImpact, ComputeKDTimeline, ComputeCombatYield)
```

Endpoints :
- `GET /api/v1/players/{slug}/matches/{match_id}` -> `MatchViewResponse` (payload monolithique)
- `GET /api/v1/players/{slug}/matches/{match_id}/neighbors` -> `MatchNeighbors`

### 2.3 Stack chart

**recharts** uniquement (`LineChart`, `AreaChart`, `XAxis`, `YAxis`, `Tooltip`, `CartesianGrid`, `ResponsiveContainer`, `Line`, `Area`).
Pas de wrapper chart partagé sous `apps/web/src/components/charts/`. Pas de radar, pas de pie, pas de stacked bars horizontales utilisés ailleurs dans la codebase Match View — à introduire.

---

## 3. Audit comparatif détaillé section par section

Chaque section liste les visualisations Python attendues, ce qui existe Go, ce qui manque, et la dette de portage (data + calcul + viz).

### 3.1 Header de match

| # | Élément Python | État Go | Manquant Go |
|---|----------------|---------|-------------|
| H1 | Carte KPI Date (`format_date_fr`) | OK (texte `start_time_label`) | Layout dédié |
| H2 | Carte KPI Score colorée + badge Domination/Humiliation/Remontada/etc. | Partiel (label seul) | Badge dominance (5 flags), couleur outcome |
| H3 | Carte KPI Playlist | OK (badge) | - |
| H4 | Carte KPI "Mode sur Carte" | Partiel (champs séparés) | Concaténation locale FR/EN |
| H5 | Map thumbnail (resolve via `map_thumb_path`) | **MANQUE** | Asset map + endpoint |
| H6 | Bloc Performance grand format (font 4.2em) | Partiel (texte) | Mise en forme et couleur (`get_score_class`) |
| H7 | Bloc Rang LUSR/CSR : image rang 110x110 + tier_label + barre sub-tier + delta gagné/perdu (vert/rouge) + note bot 💪/⚠️ | Partiel (badge texte) | Image rang, barre progression, delta visuel, note bot |

**Sources Go déjà disponibles** : `MatchViewHeader`, `MatchViewRank`. **À ajouter** : `dominance_flag`, `map_thumbnail_url`, données rang complètes (`tier_image_path`, `sub_tier_start`, `tier_size`, `rating_delta`, `had_bot_teammate`).

### 3.2 Onglet Summary

| # | Visualisation Python | État Go | Action portage |
|---|----------------------|---------|----------------|
| S1 | KPI MMR Équipe vs Ennemis avec delta coloré | **MANQUE** | Ajouter dans `MatchSummaryKpis` : `team_mmr`, `enemy_mmr`. Card front avec couleur conditionnelle. |
| S2 | KPI Kills réel vs attendu ±delta (`os_card`) | Partiel (`StatExpectedCard`) | OK — vérifier l'écart visuel (police, layout) |
| S3 | KPI Deaths réel vs attendu ±delta (mode "inverse" : delta négatif = vert) | Partiel | Vérifier inversion couleur en mode `lowerIsBetter` |
| S4 | KPI Average Life (carte dédiée `format_mmss`) | Présent (KPI Vie moy.) | Vérifier format MM:SS |
| S5 | **Bar chart F/D/A grouped : Réel vs Attendu (pattern hachuré) vs Moyenne historique catégorie (pattern pointillé)** + annotation badge ratio K/D/A | **MANQUE** | Chart majeur. Recharts `BarChart` avec 3 séries groupées. Pattern fill via SVG defs (recharts ne supporte pas nativement -> custom shape). Backend doit renvoyer `expected_kills/deaths/assists` (déjà dans `MatchExpectedStats`) + `hist_avg_kills/deaths/assists` + `hist_match_count` (champs présents mais `HasHistAvg: false` câblé en dur). |
| S6 | **Bar chart Killing Spree / Headshots / Perfect Kills : Réel vs Moyenne historique** | **MANQUE** | Recharts `BarChart` 2 séries. Backend doit fournir `max_killing_spree`, `headshot_kills`, `perfect_kills` (déjà dans KPIs) + leurs moyennes historiques + `perfect_kills` du match (medal_name_id 1512363953). |
| S7 | **Donut camembert kills par arme (top 8 cyclé)** | **MANQUE** | Recharts `PieChart` avec `innerRadius`. Backend `combat_tab.weapon_kills[]` déjà exposé, à enrichir : appliquer `WEAPON_FUSION_MAP_ID` côté Go + cap melee/grenade par `remainder = api_total - film_kills`. |
| S8 | Tableau Arme/Frags style scoreboard | **MANQUE** | Tableau HTML simple à côté du donut, alimenté par la même source S7. |
| S9 | **Radar Participation 6 axes (Objectifs / Combat / Support / Score / Impact / Survie)** normalisé 0-100% | **MANQUE** | Chart majeur. Recharts `RadarChart`. Backend doit exposer un nouvel objet `participation: {objectifs_norm, combat_norm, support_norm, score_norm, impact_norm, survie_norm, raw_values, mode_family, is_objective_mode}`. Logique métier complexe à porter (`compute_participation_profile`, `_get_mode_family`, seuils par mode). |
| S10 | Légende textuelle 6 axes (panneau latéral conditionnel `hints_visible()`) | **MANQUE** | Composant React avec toggle (state local ou setting utilisateur). |

### 3.3 Onglet Combat

| # | Visualisation Python | État Go | Action portage |
|---|----------------------|---------|----------------|
| C1 | **Badges d'impact** (premier sang, finisseur, touriste, première victime, top gun, héros silencieux, faux frère, top killer, dernière victime) avec icônes emoji + nom joueur lié + `extra_label` ou timestamp M:SS | Partiel (`impact_badges` liste basique key/label) | Enrichir `MatchImpactBadge` côté Go : ajouter `icon`, `display_name`, `time_ms`, `is_me`, `extra_label`. Migrer la logique `compute_single_match_impact` (`silent_hero`, `false_brother`, `top_killer`, `top_gun`) si elle n'est pas encore dans `analysis.ComputeSingleMatchImpact`. |
| C2 | **Timeline kills/deaths cumulés du joueur principal** avec annotations d'impact (flèches vers events) | Présent partiel (recharts `LineChart` 2 lignes) | Ajouter annotations recharts (`<ReferenceDot>` ou customLabel). Couleurs Okabe-Ito #0072B2 (kills) / #D55E00 (deaths). Position annotation ajustée si proche temporellement (logique anti-collision `PROXIMITY_THRESHOLD_MS = 30000`). |
| C3 | **Tug-of-war dominance d'équipe stacked bars (buckets 30s)** avec annotations cumul highlightées + ligne de parité 50% | Partiel (recharts `AreaChart` 1 série `net_kills`) | Refonte complète. Recharts `BarChart` empilé horizontal par bucket. 2 séries `my_share` / `enemy_share`. Annotations cumul (au-dessus + en dessous). Backend doit renvoyer buckets bruts `(t_start, t_end, my_kills, enemy_kills, cumul_my, cumul_enemy)` et pas seulement `net_kills`. |
| C4 | **Kill feed individuel + séries (KillStreak ≥3, gap ≤60s)** : 2 panneaux markers + lignes pour streaks + annotations gamertag×N | **MANQUE** | Sub-plot du C3. Détection des streaks : `detect_streaks(min_kills=3, gap_s=60)`. Recharts ne supporte pas nativement les sub-plots ; implémenter 2 `<ResponsiveContainer>` synchronisés sur la même échelle X. |
| C5 | **Histogramme cadence bicolore (15/30/60s) + moyennes glissantes (window=3)** + annotation pic + segmented control | **MANQUE** | Recharts `ComposedChart` : 2 séries Bar + 2 séries Line (MA). `<Tabs>` ou `<ToggleGroup>` pour 15/30/60s. Backend doit exposer `combat_tab.cadence_buckets` et calcul MA. |
| C6 | **Cards Nemesis + Victime/Bully** colorées (vert/rouge/violet/slate selon ratio) avec `≈` si estimated | Liste textuelle simple | Composant card visuel à créer. Backend doit retourner `nemesis_card` + `bully_card` avec `gamertag`, `killed_me_certain`, `killed_me_estimated`, `i_killed_certain`, `i_killed_estimated`, `cmp_color`. Logique `compute_personal_antagonists` (hybride 2-passes) à porter. |
| C7 | Caption Debug antagonistes (validé/non + counts) | **MANQUE** | Optionnel — mode debug query param `?debug_antagonists=1`. |
| C8 | **Stacked bars Killer→Victime** (1 ligne par tueur, segments par victime, palette Okabe-Ito 12 couleurs) | **MANQUE** | Recharts `BarChart` horizontal (`layout="vertical"`) empilé. Backend doit exposer `killer_victim_matrix: [{killer_xuid, killer_gamertag, killer_rank, victims: [{victim_gamertag, count}]}]`. |
| C9 | **Timeline K/D différentiel cumulé (+1 kill / -1 death) tous joueurs** avec hline 0 et joueur principal mis en avant | **MANQUE** | Recharts `LineChart` multi-séries (1 série par joueur). Joueur principal avec `strokeWidth=4.5`, autres avec `strokeWidth=2.8 opacity=0.65`. Backend doit exposer `kd_differential_timeline: [{xuid, gamertag, points: [{time_ms, kd_score}]}]`. Couleurs via palette Okabe-Ito 12 couleurs côté front. |

### 3.4 Onglet Team

| # | Visualisation Python | État Go | Action portage |
|---|----------------------|---------|----------------|
| T1 | Scoreboard par équipe avec highlight best/worst, MVP/LVP, ligne "moi" | OK (`MatchScoreboard.tsx`) | Vérifier les 19 colonnes Python vs 16 Go (manque potentiellement `melee_kills`, `power_weapon_kills`, `top_weapon_id` selon les versions) |
| T2 | Toggle expansion CSS pure (input checkbox + label) | OK (état React) | - |
| T3 | **Panneau détail joueur** pour TOUS les joueurs (pas que `is_me`) | Partiel (`PlayerDetailPanel.tsx` mais sections Armes/Médailles/Citations limitées à `is_me`) | Étendre. Backend doit renvoyer pour CHAQUE joueur du scoreboard : `weapon_kills`, `medals`, `citations_progress`, `expected_delta`, `nemesis`, `bully`, `performance_score_local`, `skill_rank`, `bot_note`, `profile_link`. |
| T3a | Section Armes joueur (icônes + count) | Présent pour `is_me` | Étendre tous joueurs |
| T3b | Section Médailles + Citations joueur (icônes circulaires + ×count) | Présent pour `is_me` | Étendre tous joueurs |
| T3c | **Section Expected K/D/A delta** dans le détail (label : exp vs actual ↑+delta coloré) | **MANQUE** | Ajouter au DTO joueur |
| T3d | **Section Antagonist** dans le détail (Nemesis + Bully) | **MANQUE** | Ajouter au DTO joueur |
| T3e | Section Local (Performance + Skill rank + bot note) | Partiel | Vérifier présence skill rank et bot note |
| T3f | Footer profile link (badge "DB joueur" / "Shared seulement" + lien Explorer) | **MANQUE** | Backend doit renvoyer `has_player_db` + `profile_url` |
| T4 | **Tableau Encounters historique** avec badges Allié+/Coriace/Tough nut, WR coloré, K/D croisé coloré, dernière rencontre relative, ordinal "1ère/2e/3e" | Liste textuelle simple (`is_ally`, `count_together`) | Refonte complète. Backend doit exposer `team_tab.encounters[]` enrichi : `{xuid, gamertag, total_encounters, ally_count, enemy_count, winrate_as_ally, winrate_vs_enemy, kills_dealt, deaths_suffered, last_seen, badges: ["allie_plus", "coriace", "tough_nut"]}`. Logique `compute_encounter_badges`, ghost player filter, friends filter à porter. |
| T5 | Popover Légende badges encounters | **MANQUE** | Composant `<Popover>` shadcn |

### 3.5 Onglet Citations & Médailles

| # | Visualisation Python | État Go | Action portage |
|---|----------------------|---------|----------------|
| CT1 | Anneaux progression circulaires CSS (image arrière-plan + ratio + état master or) avec compteur delta `+N` | **Backend vide** | Backend `citations_tab.commendations` toujours `[]`. À implémenter : appel à `CitationEngine` équivalent côté Go ou agrégation directe `match_citations` table par-joueur. Front : composant `<CitationRing>` avec SVG `<circle>` + image. |
| CT2 | Grille médailles 8 colonnes centrée avec icônes + tooltips description | **Backend vide** | Backend `citations_tab.medals` toujours `[]`. À implémenter : reprendre `summary_tab.medals` (déjà rempli) + enrichir avec descriptions FR/EN via `metadata.duckdb`. Front : grille existante à adapter. |

### 3.6 Onglet Media

| # | Visualisation Python | État Go | Action portage |
|---|----------------------|---------|----------------|
| M1 | Section "mes captures" (mine) | Partiel (mélange tout) | Backend doit séparer `mine[]` vs `teammate[]` (groupé par gamertag). Champ `section` déjà présent dans Python. |
| M2 | Section "captures de coéquipier <gt>" | **MANQUE** | Idem M1 |
| M3 | Grille images 4 cols (thumbnail prioritaire) | OK (grid responsive) | - |
| M4 | Sélecteur vidéo intégré (`st.selectbox` + `st.video`) | **MANQUE** (file_name texte uniquement) | Composant `<video>` HTML5 ou lecteur custom + dropdown sélection |
| M5 | Caption fenêtre temporelle si fallback legacy | N/A | Pas pertinent côté Go (pas de scan disque, tout passe par DB indexée) |

---

## 4. Dette de portage backend Go (DTO)

### 4.1 Champs à ajouter dans `MatchViewResponse`

```go
type MatchViewHeader struct {
    // existant...
    DominanceFlag       *int    `json:"dominance_flag,omitempty"`     // 1-5 (Domination, Humiliation, Remontada, Débandade, Contre-remontada)
    MapThumbnailURL     string  `json:"map_thumbnail_url,omitempty"`
}

type MatchViewRank struct {
    // existant : RatingType, TierLabel, NumericValue, DeltaValue
    TierImagePath   string  `json:"tier_image_path,omitempty"`     // ou data URL
    SubTierStart    *float64 `json:"sub_tier_start,omitempty"`
    TierSize        *float64 `json:"tier_size,omitempty"`
    HadBotTeammate  bool    `json:"had_bot_teammate"`
    BotOutcome      *int    `json:"bot_outcome,omitempty"`
}

type MatchSummaryKpis struct {
    // existant...
    TeamMMR    *float64 `json:"team_mmr,omitempty"`
    EnemyMMR   *float64 `json:"enemy_mmr,omitempty"`
}

type MatchExpectedStats struct {
    // existant : HasExpectedData, ExpectedKills, ExpectedDeaths, ExpectedAssists
    HasHistAvg          bool    `json:"has_hist_avg"`              // câblé true au lieu de false
    HistAvgKills        float64 `json:"hist_avg_kills"`
    HistAvgDeaths       float64 `json:"hist_avg_deaths"`
    HistAvgAssists      float64 `json:"hist_avg_assists"`
    HistAvgKilling      float64 `json:"hist_avg_killing_spree"`
    HistAvgHeadshots    float64 `json:"hist_avg_headshot_kills"`
    HistAvgPerfectKills float64 `json:"hist_avg_perfect_kills"`
    HistMatchCount      int     `json:"hist_match_count"`
    HistModeCategory    string  `json:"hist_mode_category"`         // Assassin, Fiesta, BTB, Ranked, Firefight, Other
    PerfectKillsMatch   int     `json:"perfect_kills_match"`
}

type MatchParticipationRadar struct {
    ObjectifsNorm  float64            `json:"objectifs_norm"`
    CombatNorm     float64            `json:"combat_norm"`
    SupportNorm    float64            `json:"support_norm"`
    ScoreNorm      float64            `json:"score_norm"`
    ImpactNorm     float64            `json:"impact_norm"`
    SurvieNorm     float64            `json:"survie_norm"`
    RawValues      map[string]float64 `json:"raw_values"`
    ModeFamily     string             `json:"mode_family"`
    IsObjectiveMode bool              `json:"is_objective_mode"`
    HasData        bool               `json:"has_data"`
}

type MatchSummaryTab struct {
    // existant + :
    Participation *MatchParticipationRadar `json:"participation,omitempty"`
}

type MatchImpactBadge struct {
    Key         string  `json:"key"`           // first_blood, clutch_finisher, last_casualty, etc.
    Icon        string  `json:"icon"`          // emoji
    Label       string  `json:"label"`         // i18n FR/EN
    DisplayName string  `json:"display_name"`  // gamertag
    XUID        string  `json:"xuid"`
    TimeMs      int64   `json:"time_ms"`       // -1 si stats-only
    IsMe        bool    `json:"is_me"`
    ExtraLabel  string  `json:"extra_label,omitempty"`  // pour stats-only ("3 assists. · 1 mort")
}

type MatchTugOfWarBucket struct {
    BinStart     int     `json:"bin_start"`        // secondes
    BinEnd       int     `json:"bin_end"`
    MyKills      int     `json:"my_kills"`
    EnemyKills   int     `json:"enemy_kills"`
    MyShare      float64 `json:"my_share"`         // 0-100
    CumulMy      int     `json:"cumul_my"`
    CumulEnemy   int     `json:"cumul_enemy"`
}

type MatchKillStreak struct {
    XUID         string   `json:"xuid"`
    Gamertag     string   `json:"gamertag"`
    TeamID       int      `json:"team_id"`
    KillTimesMs  []int64  `json:"kill_times_ms"`
    KillsCount   int      `json:"kills_count"`
}

type MatchKillFeedItem struct {
    TimeMs   int64  `json:"time_ms"`
    XUID     string `json:"xuid"`
    Gamertag string `json:"gamertag"`
    TeamID   int    `json:"team_id"`
}

type MatchCadenceBucket struct {
    BinStart     int     `json:"bin_start"`
    BinEnd       int     `json:"bin_end"`
    MyKills      int     `json:"my_kills"`
    EnemyKills   int     `json:"enemy_kills"`
    MAMy         float64 `json:"ma_my"`         // moyenne glissante window=3
    MAEnemy      float64 `json:"ma_enemy"`
}

type MatchAntagonistCard struct {
    XUID                  string  `json:"xuid"`
    Gamertag              string  `json:"gamertag"`
    KilledMeCertain       int     `json:"killed_me_certain"`
    KilledMeEstimated     int     `json:"killed_me_estimated"`
    IKilledThemCertain    int     `json:"i_killed_them_certain"`
    IKilledThemEstimated  int     `json:"i_killed_them_estimated"`
    HasEstimated          bool    `json:"has_estimated"`
    CmpColor              string  `json:"cmp_color"`              // "red"|"green"|"violet"|"slate"
}

type MatchKillerVictimRow struct {
    KillerXUID     string `json:"killer_xuid"`
    KillerGamertag string `json:"killer_gamertag"`
    KillerRank     int    `json:"killer_rank"`
    Victims        []struct {
        VictimXUID     string `json:"victim_xuid"`
        VictimGamertag string `json:"victim_gamertag"`
        Count          int    `json:"count"`
    } `json:"victims"`
}

type MatchKDPlayerSeries struct {
    XUID     string `json:"xuid"`
    Gamertag string `json:"gamertag"`
    IsMe     bool   `json:"is_me"`
    Points   []struct {
        TimeMs  int64 `json:"time_ms"`
        KDScore int   `json:"kd_score"`        // cumulé +1/-1
    } `json:"points"`
}

type MatchCombatTab struct {
    // existant :
    WeaponKills      []MatchWeaponKill
    HighlightEvents  []MatchHighlightEvent
    TugOfWar         []MatchTugOfWarBucket   // refonte (était []MatchTugOfWarBin)
    ImpactBadges     []MatchImpactBadge      // enrichi
    KDTimeline       []MatchKDTimelinePoint  // joueur principal (existant, OK)
    NemesisDuels     []MatchNemesisRow       // déjà existant mais vide

    // ajouts :
    KillStreaks         []MatchKillStreak
    KillFeed            []MatchKillFeedItem
    CadenceBuckets15s   []MatchCadenceBucket
    CadenceBuckets30s   []MatchCadenceBucket
    CadenceBuckets60s   []MatchCadenceBucket
    NemesisCard         *MatchAntagonistCard `json:"nemesis_card,omitempty"`
    BullyCard           *MatchAntagonistCard `json:"bully_card,omitempty"`
    KillerVictimMatrix  []MatchKillerVictimRow
    KDDifferentialAll   []MatchKDPlayerSeries
}

type MatchScoreboardRow struct {
    // existant : 30+ champs.
    // ajouts pour le panneau détail :
    DetailWeaponKills    []MatchWeaponKill        `json:"detail_weapon_kills,omitempty"`
    DetailMedals         []MatchMedal             `json:"detail_medals,omitempty"`
    DetailCitations      []MatchCitation          `json:"detail_citations,omitempty"`
    DetailExpected       *MatchExpectedDelta      `json:"detail_expected,omitempty"`
    DetailNemesis        *MatchAntagonistCard     `json:"detail_nemesis,omitempty"`
    DetailBully          *MatchAntagonistCard     `json:"detail_bully,omitempty"`
    DetailLocalScore     *float64                 `json:"detail_local_score,omitempty"`
    DetailLocalRank      *MatchViewRank           `json:"detail_local_rank,omitempty"`
    DetailHasPlayerDB    bool                     `json:"detail_has_player_db"`
    DetailProfileURL     string                   `json:"detail_profile_url,omitempty"`
}

type MatchEncounterRowV2 struct {
    XUID              string   `json:"xuid"`
    Gamertag          string   `json:"gamertag"`
    IsAlly            bool     `json:"is_ally"`
    TotalEncounters   int      `json:"total_encounters"`
    AllyCount         int      `json:"ally_count"`
    EnemyCount        int      `json:"enemy_count"`
    WinrateAsAlly     float64  `json:"winrate_as_ally"`
    WinrateVsEnemy    float64  `json:"winrate_vs_enemy"`
    KillsDealt        int      `json:"kills_dealt"`
    DeathsSuffered    int      `json:"deaths_suffered"`
    LastSeen          *string  `json:"last_seen,omitempty"`        // ISO date
    Badges            []string `json:"badges"`                     // "allie_plus", "coriace", "tough_nut"
}

type MatchMediaItem struct {
    // existant : FileID, FileName, FilePath, ThumbnailURL, DurationSeconds, CaptureTime, Liked
    Section   string `json:"section"`              // "mine" | "teammate"
    Gamertag  string `json:"gamertag,omitempty"`
    Kind      string `json:"kind"`                 // "image" | "video"
}
```

### 4.2 Logique métier à porter dans `apps/go-api/internal/analysis/`

| Fonction Python | Cible Go | Notes |
|----------------|----------|-------|
| `compute_single_match_impact` (badges) | `analysis/match_impact.go` (existe partiellement) | Vérifier que les 9 types de badges sont implémentés (first_blood, clutch_finisher, last_casualty, last_group_kill, first_group_death, top_gun, top_killer, silent_hero, false_brother). Si manquants, ajouter. |
| `compute_dominance_buckets(bucket_s=30)` | `analysis/tug_of_war.go` | Refonte : retourner buckets bruts (pas seulement net_kills) |
| `detect_streaks(min_kills=3, gap_s=60)` | nouveau `analysis/kill_streaks.go` | Algorithmie chronologique avec reset sur death ou gap |
| `prepare_kill_feed` | nouveau `analysis/kill_feed.go` | Filtrer kills hors streaks |
| `compute_cadence_buckets(bucket_s)` + MA window=3 | nouveau `analysis/cadence.go` | 3 granularités (15s, 30s, 60s) |
| `compute_personal_antagonists` (hybride 2-passes A+B, tolerance ±5ms) | nouveau `analysis/antagonists.go` | Logique non triviale : pass1 certain, pass2 estimé via _choose_best |
| `compute_killer_victim_pairs` (fallback) | nouveau `analysis/killer_victim.go` | Bisect kill↔death à ±5 ms |
| `plot_all_players_frags_timeline` | équivalent côté analysis pour préparer `MatchKDPlayerSeries[]` | +1 par kill, -1 par death cumulé |
| `compute_participation_profile` + `_get_mode_family` + `_is_objective_mode_from_pair_name` | nouveau `analysis/participation.go` | Charger PSA depuis player DB ; calculer 6 axes ; normaliser via seuils par mode |
| `compute_global_radar_thresholds` | nouveau `analysis/participation_thresholds.go` | Cache process. Scan cross-DB des max par catégorie |
| `compute_mode_category_averages` | `analysis/mode_categories.go` (vérifier) | Retourne moyennes K/D/A/spree/HS/perfect filtrées par catégorie custom |
| `compute_encounter_badges` (Allié+, Coriace, Tough nut) + `filter_encounter_xuids` (ghost + friends) | nouveau `analysis/encounters.go` | Avec dataclass `EncounterStats` |
| `WEAPON_FUSION_MAP_ID` + résolution multi-source | extension de `analysis/weapons.go` | Sentinels weapon_id : melee=1, grenade=0, vehicle=2 |

### 4.3 Constantes magiques à centraliser

```go
const (
    PerfectKillMedalID       = uint64(1512363953)
    MeleeWeaponID            = uint64(1)
    GrenadeWeaponID          = uint64(0)
    VehicleWeaponID          = uint64(2)
    StreakMinKills           = 3
    StreakGapSeconds         = 60
    DominanceBucketSeconds   = 30
    CadenceMAWindow          = 3
    AntagonistToleranceMs    = 5
    ImpactProximityMs        = 30000
    TopGunKillThreshold      = 10
    EncounterAllyPlusWR      = 0.65
    EncounterAllyPlusMin     = 2
    EncounterCoriaceWR       = 0.35
    EncounterCoriaceMin      = 3
    EncounterToughNutDeaths  = 3
    EncounterToughNutRatio   = 2.0
    SurvieDeathsPerMinRef    = 2.0
    SurvieAvgLifeRefSeconds  = 90.0
)
```

### 4.4 Interface `port.MatchViewRepository` — à étendre avant l'implémentation

Règle impérative : **déclarer dans l'interface avant d'implémenter dans le repo concret.**

```
1. Déclarer la méthode dans internal/port/match_view.go (ou repository.go selon structure)
2. Implémenter dans internal/platform/duckdb/match_view_repo.go
3. Câbler dans internal/service/match_view_service.go via injection
   → jamais d'accès au repo concret depuis le handler
```

**Nouvelles méthodes à ajouter par phase** :

| Phase | Méthode interface |
|-------|------------------|
| C | `GetMatchSkillRankWithCareer(ctx, matchID, xuid string) (*MatchViewRank, error)` |
| C | `GetMatchCitations(ctx, matchID, xuid string) ([]MatchCitation, error)` |
| C | `GetMatchMediaItems(ctx, matchID string) ([]MatchMediaItem, error)` |
| G | `GetPlayerMatchHistoryAvg(ctx, xuid, modeCategory string) (*MatchHistoryAvg, error)` |
| I | `GetMatchWeaponKillsBatch(ctx, matchID string, xuids []string) (map[string][]MatchWeaponKill, error)` |
| I | `GetMatchMedalsBatch(ctx, matchID string, xuids []string) (map[string][]MatchMedal, error)` |
| J | `GetMatchPSA(ctx, matchID, xuid string) ([]PersonalScoreAward, error)` |

---

## 5. Dette de portage frontend (Next.js + recharts)

### 5.1 Composants chart à créer

| Composant | Lib | Usage |
|-----------|-----|-------|
| `<RadarChart6>` | recharts `RadarChart` | Radar Participation |
| `<GroupedBarChart>` | recharts `BarChart` (multi-Bar) + custom shapes pour patterns hachuré/pointillé | F/D/A vs attendu vs hist + Spree/HS/Perfect |
| `<DonutChart>` | recharts `PieChart` `innerRadius=35%` | Kills par arme |
| `<TugOfWarStacked>` | recharts `BarChart` empilé + annotations custom | Dominance d'équipe |
| `<KillFeedTimeline>` | 2 `<ResponsiveContainer>` synchronisés (ScatterChart) | Kill feed + streaks |
| `<CadenceHistogram>` | recharts `ComposedChart` (Bar + Line) | Cadence + MA |
| `<KillerVictimMatrix>` | recharts `BarChart` horizontal empilé | KV stacked |
| `<KDDifferentialMultiLine>` | recharts `LineChart` multi-séries | K/D différentiel tous joueurs |
| `<MatchImpactTimelineAnnotated>` | recharts `LineChart` + `<ReferenceDot>` + custom labels | K/D cumulé joueur + annotations |
| `<RankProgressBar>` | SVG/CSS pure | Bloc rang header |
| `<CitationRing>` | SVG `<circle>` | Anneau progression citation |

### 5.2 Composants UI à créer

| Composant | Notes |
|-----------|-------|
| `<MapThumbnail src={url} />` | Asset map header |
| `<DominanceBadge flag={1..5} />` | Badges Domination/Humiliation/Remontada/Débandade/Contre-remontada |
| `<NemesisCard />` / `<BullyCard />` | Cards avec couleur conditionnelle (rouge/vert/violet/slate) |
| `<EncounterBadgeChip type="allie_plus|coriace|tough_nut" />` | Badges colorés avec tooltip |
| `<ImpactBadgesGrid badges={...} />` | Flexbox wrap des 9 types de badges |
| `<MediaPlayer kind="image|video" />` | `<video controls>` pour vidéos |
| `<CitationRingGrid citations={...} />` | Grille 8 colonnes |
| `<MedalsGrid medals={...} cols={8} center />` | Grille médailles |
| `<EncountersTable rows={...} />` | Tableau dédié (≠ scoreboard) |
| `<HintsToggleSection>` | Wrapper conditionnel pour textes d'aide |

### 5.3 Internationalisation

Tous les libellés `t("mv_*")`, `viz_t("*")`, `t("mvc_*")`, `t("mvp_*")`, `t("radar_desc_*")`, `t("mv_weapon_kills_*")`, `t("col_*")` doivent être migrés dans `apps/web/src/lib/i18n/` (FR/EN). À cataloguer dans une étape dédiée (probablement >150 clés).

### 5.4 Couleurs (règle 20 CLAUDE.md)

Toutes les couleurs sémantiques doivent passer par `tokenCssVar(token)` ou `resolveToken(token)` ou `getSeriesColors`. Migrer la palette Okabe-Ito (`#0072B2` bleu, `#D55E00` vermillon, `#009E73` vert, `#E69F00` orange, `#56B4E9`, `#CC79A7`, `#F0E442`, etc.) dans `apps/web/src/lib/accessibility/palettes/` si pas déjà fait. Vérifier que `getSeriesColors(12, palette="okabe_ito")` est disponible.

### 5.5 Query keys et labels de stats

**Query keys** : les 2 endpoints existants couvrent tout le scope du portage — aucune nouvelle query key n'est nécessaire dans `apps/web/src/lib/query/keys.ts` pour les phases A–K. Vérifier que `matchView` et `matchNeighbors` sont bien déclarées dans ce fichier avant de commencer.

**Labels de stats** : tout libellé de colonne ou métrique doit passer par `useFieldLabel(fieldKey)` et `useOutcomeLabel(outcome)` — jamais de chaînes hardcodées. En particulier :
- Colonnes scoreboard : `kills`, `deaths`, `assists`, `kda`, `accuracy`, `personal_score`, `damage_dealt`, `damage_taken`
- Axes radar : `objectifs_norm`, `combat_norm`, `support_norm`, `score_norm`, `impact_norm`, `survie_norm` — à enregistrer dans `fields.toml` si absents (Phase J)
- Outcomes (`OutcomeWin`, `OutcomeLoss`, `OutcomeTie`, `OutcomeDNF`) via `useOutcomeLabel`

### 5.6 Conformité architecture multi-titres

Toutes les nouvelles données backend doivent passer par l'abstraction multi-titres.

**TitleDataAdapter / types canoniques** :
- Toute lecture de données de match depuis le service doit passer par `TitleDataAdapter.Load*()` retournant des types `canonical.*` — jamais de colonnes DuckDB title-specific dans `match_view_service.go`
- Les nouveaux champs de scoring (MMR, participation_radar, dominance_flag) doivent être retournés en types `canonical.MatchDetail` ou `canonical.MatchParticipant`, pas en structs Halo-spécifiques

**HasCapability() — branchements obligatoires** :

| Feature | Capability à vérifier |
|---------|----------------------|
| Radar Participation 6 axes (#16) | `CapRanked` ou cap dédiée `CapPersonalScoreAwards` si créée |
| Onglet Media mine/teammate (#33) | `CapMedia` |
| Anneaux citations (#31) | `CapCareer` |
| Image rang + sub-tier (#7) | `CapRanked` |

Si la capability est absente : retourner `ErrCapabilityNotSupported` (voir § 8.2) — jamais de `nil` silencieux ni de panic.

**TOML mappings** : tout nouveau `FieldKey` canonique doit être déclaré dans `config/titles/halo_infinite/mappings/fields.toml`. Champs à ajouter par phase :

| Phase | FieldKey à ajouter dans fields.toml |
|-------|--------------------------------------|
| B | `team_mmr`, `enemy_mmr`, `dominance_flag` |
| C | `tier_image_path`, `sub_tier_start`, `tier_size` |
| J | `objectifs_norm`, `combat_norm`, `support_norm`, `score_norm`, `impact_norm`, `survie_norm` |

**PathResolver** : tout chemin vers un asset (image map, image rang) doit passer par `PathResolver` — aucun `filepath.Join` direct sur `data/` ni URL construite en dur dans le service ou le repo.

### 5.7 Spécifications de rendu par chart (recharts)

> Audit direct du code Python v7/cockpit (Plotly). Mapping vers recharts équivalents.
> Format temps commun : ms → `"M:SS"` = `Math.floor(ms/60000) + ":" + String(Math.floor((ms%60000)/1000)).padStart(2,"0")`.
> Couleurs : toujours via `getSeriesColors(n)` ou `resolveToken(token)` — jamais de hex inline.

| Chart | Hauteur | Axe X | Axe Y | État vide |
|-------|--------:|-------|-------|-----------|
| S5 F/D/A grouped | 360px | Labels texte K/D/A | entiers, min=0 | masquer si `!hasExpectedData` |
| S6 Spree/HS/Perfect | 260px | Labels texte 3 catégories | entiers, min=0 | masquer si `!hasSpreeOrHs` |
| S7 Donut weapon kills | 320px | — | — | `<EmptyStateNotice>` si `[]` |
| S9 Radar Participation | 380px | 6 axes angulaires i18n | 0–1, ticks "25%"…"100%" | `<EmptyStateNotice label={t("mv_radar_no_psa")}>` si `!has_data` |
| C2 KD cumulé + annotations | 340px | ms → M:SS, tick / 60s | entiers, min=0 | masquer si kills+deaths vides |
| C3+C4 Tug-of-war + kill feed | 420px total (68%+32%) | secondes → M:SS, partagé | 0–100% (haut) / markers (bas) | `<EmptyStateNotice>` si `[]` |
| C5 Cadence histogram | 320px | secondes → M:SS | kills entiers, min=0 | masquer si total kills < 3 |
| C8 KV stacked horiz | `max(80, 80+24×n)` px | kill count | gamertag (categ.) | `<EmptyStateNotice>` si `[]` |
| C9 KD diff. multi-joueurs | 380px | ms → M:SS | score KD (négatif possible) | masquer si `[]` |

**Détails séries et tooltips** :

**S5 F/D/A** — 3 `<Bar>` par catégorie, `barCategoryGap="20%"`. Réel : opacité 1. Attendu : opacité 0.6, SVG pattern hachuré `stroke-dasharray="/ 3"`. Hist (si `hist_match_count ≥ 10`) : opacité 0.35, SVG pattern pointillé. Tooltip : `"K : réel={v} / attendu={exp:.1f} / hist={hist:.1f}"`. Annotation fixe : badge amber en `position:absolute top-2 right-2` avec ratio `(K+A)/D`. Légende : `<Legend verticalAlign="bottom">`.

**S6 Spree/HS/Perfect** — 2 `<Bar>` (réel / hist). Couleurs : `getSeriesColors(3)` = [violet, cyan, vert]. Hist conditionnel ≥ 10 matchs, pattern pointillé. Tooltip : `"Réel: {v} / Hist: {avg:.1f}"`.

**S7 Donut** — `<PieChart>` + `<Pie innerRadius="35%" outerRadius="80%">`. Top 8 armes post-fusion. Couleurs : `getSeriesColors(8)`. Tableau adjacent : 2 colonnes (weapon_label, kill_count), tri décroissant. Tooltip : `"{label}: {count} kills ({pct:.1f}%)"`.

**S9 Radar** — `<RadarChart>` fermé (valeurs + [valeurs[0]]). `<PolarRadiusAxis domain={[0,1]} tickCount={4} tickFormatter={v => v*100+"%"}>`. Fill `resolveToken("perf-tier-2")` opacité 0.25. Grille : `gridColor="rgba(255,255,255,0.12)"`. Tooltip : `"{axe}: {raw} → {pct:.0f}%"`.

**C2 KD cumulé joueur** — Kills `#0072B2` (Okabe bleu) `strokeWidth=2.5`. Deaths `#D55E00` (Okabe vermillion) `strokeDasharray="5 3"`. `<ReferenceDot>` par badge impact, couleur `#E69F00`, label offset vertical `[-40,-90,-140]` si `|Δt| < 30000ms`. Tooltip : `"M:SS — Kills: {k} / Deaths: {d}"`.

**C3+C4 Tug-of-war** — 2 `<ResponsiveContainer>` synchronisés via state `xDomain`. Panneau 1 : `<BarChart>` stackId `"tw"`, Mon équipe `rgba(0,114,178,0.85)`, Ennemi `rgba(213,94,0,0.85)`, `<ReferenceLine y={50} strokeDasharray="4 2">`. `<LabelList>` pour `cumul_my`/`cumul_enemy`. Panneau 2 : `<ScatterChart>` markers diamond 6px. Streaks : `<ReferenceArea>` avec annotation `"{gt} ×{n}"`.

**C5 Cadence** — `<ComposedChart>`. Barres mon équipe `rgba(0,114,178,0.5)` bordure `#0072B2`. Barres ennemi `rgba(213,94,0,0.5)` bordure `#D55E00`. MA : `strokeWidth=3`, outline blanc 5px en fond (trace fantôme). `<ToggleGroup value={bucket}>` 15s/30s/60s (state local). `<ReferenceLine x={peakBin}>` + label pic. Tooltip : `"Bin M:SS — Mon équipe: {my} kills (MA: {ma:.1f}) / Ennemi: {enemy} kills"`.

**C8 KV matrix** — `<BarChart layout="vertical" stackId="kv">`. `yAxis type="category"` trié par rank puis kills desc, `reversed={true}`. 1 `<Bar>` par victime unique, `getSeriesColors(n_victims)`. Margin left 140px. Légende verticale droite. Tooltip : `"{tueur} → {victime} : {count} kills"`.

**C9 KD diff.** — 1 `<Line>` par joueur, `strokeWidth={isMe ? 4.5 : 2.8}`, `opacity={isMe ? 1 : 0.65}`. `<ReferenceLine y={0}>`. Début forcé à (0,0). Hover unifié : `"{gamertag}: {kd_score}"`. Ordre traces : joueur principal d'abord.

### 5.8 Spécifications des tableaux (colonnes, couleurs, tri)

> **Source de vérité colonnes** : `src/ui/pages/match_view_scoreboard.py` v7/cockpit (`_get_scoreboard_cols()`).
> Source seuils Go : `instances.ts`.

#### Scoreboard (21 colonnes = 19 Python + 2 ajouts Go)

> **Colonnes inversées Python** : seulement `deaths` et `damage_taken` (pas `damage_per_kill`).
> **MVP/LVP Python** : basé sur count de cellules best/worst, bots exclus — plus juste que max kills seul.

| # | Champ | Label FR | Format | Inversé | Source |
|---|-------|----------|--------|:---:|--------|
| 1 | gamertag | Joueur | texte + badges | — | Python |
| 2 | rank | Rang | entier (1, 2…) | — | Python |
| 3 | score | Score | entier | — | Python |
| 4 | kills | K | entier | non | Python |
| 5 | deaths | D | entier | **oui** | Python |
| 6 | assists | A | entier | non | Python |
| 7 | kda | KDA | 2 déc. | non | Python |
| 8 | top_weapon_label | Arme | texte (résolu) | — | Python |
| 9 | max_killing_spree | Spree | entier | non | Python |
| 10 | headshot_kills | HS | entier | non | Python |
| 11 | perfect_kills | Perf | entier | non | Python |
| 12 | shots_fired | Tirs | entier | non | Python |
| 13 | shots_hit | Touchés | entier | non | Python |
| 14 | accuracy | Précision | `"{v:.1f} %"` | non | Python (champ API direct) |
| 15 | melee_kills | CàC | entier | non | Python |
| 16 | power_weapon_kills | PW | entier | non | Python |
| 17 | damage_dealt | Dmg+ | entier | non | Python |
| 18 | damage_taken | Dmg- | entier | **oui** | Python |
| 19 | avg_life_seconds | Vie moy | M:SS | — | Python |
| 20 | offensive_conversion | Rendement | `"{v*100:.0f}%"` | non | **Ajout Go** |
| 21 | defensive_resistance | Résistance | `"{v*100:.0f}%"` | non | **Ajout Go** |

- **Highlight best/worst** : `bg-success/40 text-success font-semibold` (best) / `bg-destructive/40 text-destructive` (worst). Condition : `min ≠ max` et colonne non dans `{gamertag, rank, top_weapon_label}`.
- **MVP** : count de cellules "best" le plus élevé (bots exclus), tie-breaker `rank`. **LVP** : count "worst" le plus élevé.
- **Joueur courant** (`is_me`) : row `bg-info/10`.
- **Tri** : aucun contrôle interactif — groupé par `team_side`, ordre backend.
- **Expansion** : clic row → `<PlayerDetailPanel>` en `<tr><td colSpan={n+3}>`. Indicateur chevron ▸/▾. **Une seule row expandée** (state `expandedXuid: string | null`, toggle null/xuid).

#### Encounters (match view — nouveau composant `<EncountersTable>`)

Distinct du composant career encounters. Construit à partir des données `team_tab.encounters[]`.

| # | Champ | Label FR | Label EN | Format | Couleur |
|---|-------|----------|----------|--------|---------|
| 1 | gamertag | Joueur | Player | texte | — |
| 2 | total_encounters | Matchs | Matches | entier | — |
| 3 | ally_count + enemy_count | Allié / Ennemi | Ally / Enemy | "{ally}A / {enemy}E" | — |
| 4 | winrate_as_ally | WR allié | Win% (ally) | pct | `≥65%`=`text-success` / `≤35%`=`text-destructive` |
| 5 | winrate_vs_enemy | WR vs ennemi | Win% (vs) | pct | `≥65%`=`text-success` / `≤35%`=`text-destructive` |
| 6 | kills_dealt / deaths_suffered | K/D croisé | K/D | "{k}/{d}" | `k/d ≥ 1.5`=`text-success` / `k/d ≤ 0.67`=`text-destructive` |
| 7 | badges | Badges | Badges | chips `<EncounterBadgeChip>` | voir thresholds |
| 8 | last_seen | Dernière rencontre | Last seen | `toLocaleDateString('fr-FR')` | `text-muted-foreground text-xs` |

- **Tri par défaut** : `total_encounters DESC`.
- **Badge thresholds** :
  - Allié+ : `winrate_as_ally ≥ 0.65 AND ally_count ≥ 2` → badge vert
  - Coriace : `winrate_vs_enemy ≤ 0.35 AND enemy_count ≥ 3` → badge rouge
  - Tough nut : `deaths_suffered ≥ 3 AND kills_dealt/deaths_suffered ≥ 2.0` → badge violet
- **Pagination** : aucune — afficher toutes les rencontres du match.
- **Popover légende** : `<Popover>` shadcn déclenchable depuis icône `ℹ`.

#### Weapons table (inline avec donut S7)

| Champ | Label FR | Format |
|-------|----------|--------|
| weapon_label | Arme | texte |
| kill_count | Frags | entier |

- 2 colonnes, tri décroissant par kill_count, pas de pagination.

#### Détail joueur — `<PlayerDetailPanel>` (expander)

**Architecture React** :
```typescript
// MatchScoreboard.tsx
const [expandedXuid, setExpandedXuid] = useState<string | null>(null)
// Click row → toggle : null si déjà ouvert, xuid sinon
// Une seule row ouverte à la fois (auto-fermeture)
// Données : monolithique — pas d'appel API séparé, tout dans MatchViewResponse
```

**Données dans le payload** : `team_tab.scoreboard[].detail_*` — chargé en batch côté Go :
- `GetMatchWeaponKillsBatch(matchID, xuids[])` → 1 requête pour N joueurs
- `GetMatchMedalsBatch(matchID, xuids[])` → 1 requête pour N joueurs
- `GetMatchExpectedStatsBatch(matchID, xuids[])` → depuis `match_participants` déjà chargé
- Antagonist : `KVPairsReparametrized(matchID, xuid)` par joueur (acceptable car petit volume)

**Sections dans l'ordre, avec disponibilité** :

| # | Section | Contenu | Tous joueurs ? |
|---|---------|---------|:-:|
| 1 | Statistiques | kills, deaths, assists, kda, damage_dealt, damage_taken, accuracy, shots_fired, shots_hit, avg_life_seconds, suicides, betrayals (grid 2-3 col) — accuracy = champ API direct | Oui (shared DB) |
| 2 | Armes | top 5 weapons avec kill count (icône + label) | Oui (shared DB) |
| 3 | Médailles | grille médailles avec ×count | Oui (shared DB) |
| 4 | Expected K/D/A | `"K exp: {exp:.1f} → réel: {actual} (+{delta})"`, couleur `divergent-pos`/`neg` | Oui (shared DB) |
| 5 | Antagonist | `<NemesisCard>` + `<BullyCard>` (KV pairs reparamétrés avec `myXUID=row.xuid`) | Oui (shared DB) |
| 6 | Local (Performance + Rank) | Performance score + Skill rank + bot note | **`is_me` uniquement** → `<EmptyStateNotice>` + lien Explorer pour autres |
| 7 | Citations | Anneaux progression (Phase C) | **`is_me` uniquement** → `<EmptyStateNotice>` pour autres |
| 8 | Footer | badge "DB joueur" (vert) ou "Données partagées" (gris) + lien `?Explorer&gamertag=` | Toujours |

**CSS insertion** :
```tsx
{isExpanded && (
  <tr key={`${row.xuid}-detail`}>
    <td colSpan={cols.length + 3} className="p-0">
      <PlayerDetailPanel row={row} />
    </td>
  </tr>
)}
```
Fond : `bg-[#151a1f]/80 border border-border rounded-b`.

---

## 6. Audit de faisabilité — état actuel du backend Go

> Croisement plan / état réel du repo Go (audit 2026-04-26).
> Permet d'ordonner les phases selon la difficulté réelle, pas seulement l'ordre logique des onglets.

### 6.1 Légende des catégories

| Symbole | Catégorie | Signification |
|:-:|----|----|
| 🟢 | READY | Tout est en place côté Go (table + repo + analysis). Reste à câbler la viz front. |
| 🟡 | DATA-OK | Données accessibles, mais fonction analysis Go à écrire (port depuis Python). |
| 🟠 | REPO-MISSING | Table existe et est synchronisée, mais aucun repository Go ne la lit. SQL trivial à ajouter. |
| 🔴 | DATA-MISSING | Table non lue/non synchronisée. Blocker majeur — sync ou architecture à changer. |
| ⚪ | FRONT-ONLY | Composant UI pur, pas de data backend. |

### 6.2 Classification des 35 visualisations

| # | Onglet | Viz | Cat. | Notes |
|--:|--------|-----|:--:|------|
| 1 | Header | KPI Date | 🟢 | `header.start_time_label` formaté |
| 2 | Header | KPI Score + badge dominance | 🟡 | Score OK ; dominance_flag : 5 règles à calculer (Domination/Humiliation/Remontada/Débandade/Contre-remontada) |
| 3 | Header | KPI Playlist | 🟢 | `header.playlist_label` |
| 4 | Header | KPI Mode/Map | 🟢 | `header.map_ui` + `mode_ui` |
| 5 | Header | Map thumbnail | 🟡 | `map_id` exposé, endpoint asset existe — câbler URL helper |
| 6 | Header | Bloc Performance | 🟢 | `perf_display` + `perf_color` calculés |
| 7 | Header | Bloc Rang LUSR/CSR (image+barre+delta+bot note) | 🟠 | Q22 OK pour valeurs, manque LEFT JOIN `metadata.career_ranks` pour image/sub_tier_start/tier_size + remplissage `had_bot_teammate` depuis scoreboard |
| 8 | Summary | KPI MMR Équipe vs Ennemis | 🟡 | `team_mmr`/`enemy_mmr` LUS par Q12 mais perdus avant `MatchSummaryKpis` — fix de 5 minutes |
| 9 | Summary | KPI Kills vs attendu | 🟢 | `expected_stats.expected_kills` OK |
| 10 | Summary | KPI Deaths vs attendu (inverse) | 🟢 | `expected_stats.expected_deaths` OK ; inversion couleur front |
| 11 | Summary | KPI Average Life | 🟢 | `kpis.average_life` formaté MM:SS |
| 12 | Summary | Bar chart F/D/A Réel/Attendu/Hist | 🟡 | `ComputeModeCategoryAverages` existe en Go ; `HasHistAvg` câblé en dur `false` (`match_view_service.go:378`) — câblage trivial |
| 13 | Summary | Bar chart Spree/HS/Perfect Réel/Hist | 🟡 | Réel OK ; étendre `match_history_avg.go` pour spree/HS/perfect |
| 14 | Summary | Donut weapon kills | 🟡 | `Q16WeaponKills` OK ; appliquer `WEAPON_FUSION_MAP_ID` + cap remainder |
| 15 | Summary | Tableau Arme/Frags | 🟢 | Réutilise S14, table HTML simple |
| 16 | Summary | **Radar Participation 6 axes** | 🔴 | `personal_score_awards` JAMAIS lue côté Go ; aucune fonction `analysis/participation_profile.go` ; aucun cache global thresholds |
| 17 | Summary | Légende textuelle 6 axes | ⚪ | i18n |
| 18 | Combat | Badges d'impact 9 types | 🟡 | `ComputeSingleMatchImpact` ne fait que 3 badges (first_blood, finisher, tourist, first_victim) ; manquent clutch_finisher, last_casualty, last_group_kill, first_group_death, top_gun, top_killer, silent_hero, false_brother |
| 19 | Combat | Timeline K/D cumulé joueur + annotations impact | 🟡 | `ComputeKDTimeline` OK ; annotations = front (réutiliser badges enrichis) |
| 20 | Combat | Tug-of-war dominance équipe stacked + annotations cumul | 🟡 | `ComputeTugOfWar` retourne `Delta` agrégé ; refondre pour exposer `MyKills/EnemyKills/CumulMy/CumulEnemy` par bin |
| 21 | Combat | Kill feed individuel + KillStreak | 🟡 | KV pairs OK ; créer `analysis/kill_streaks.go` (min=3, gap=60s) + `analysis/kill_feed.go` |
| 22 | Combat | Histogramme cadence bicolore + MA + segmented control | 🟡 | KV pairs OK ; créer `analysis/cadence.go` (3 granularités 15/30/60s + MA window=3) |
| 23 | Combat | Cards Nemesis + Bully colorées | 🟡 | `ComputeAntagonistCounts` agrégat simple existe ; porter `compute_personal_antagonists` hybride 2-passes (`analysis/kill_attribution.go` peut servir de base) |
| 24 | Combat | Caption Debug antagonistes | ⚪ | Toggle debug |
| 25 | Combat | Stacked bars Killer→Victime | 🟢 | KV pairs (Q20) OK ; agrégation triviale |
| 26 | Combat | Timeline K/D différentiel cumulé tous joueurs | 🟡 | Généraliser `ComputeKDTimeline` à N xuid |
| 27 | Team | Scoreboard table + MVP/LVP/best/worst | 🟢 | Q12 25 colonnes OK ; MVP/LVP = ranking front |
| 28 | Team | Panneau détail joueur (Armes/Médailles/Citations/Expected/Antagonist/Local/Footer) | 🔴 | Sections Weapons/Medals/Expected/Antagonist faisables par-joueur via shared. **Local Performance + SkillRank des AUTRES joueurs** = blocker architectural (1 player DB ouverte à la fois). Workaround : dégrader gracieusement |
| 29 | Team | Tableau Encounters historique + badges | 🟡 | Q23 actuel ne renvoie que `count_together`+`is_ally` ; enrichir + créer `analysis/encounters.go` (badges Allié+/Coriace/Tough nut + filter ghost/friends) |
| 30 | Team | Popover Légende badges | ⚪ | shadcn Popover |
| 31 | Citations | Anneaux progression citations | 🟠 | `match_citations` lue par `HomeRepo` mais pas par MatchViewRepo ; service câble `[]` en dur (`match_view_service.go:196`) |
| 32 | Citations | Grille médailles 8 cols + tooltips | 🟢 | `summary_tab.medals` déjà rempli + descriptions via `metadata.citation_mappings` |
| 33 | Media | Sections mine / teammate | 🟠 | Q24 filtre `player_slug=?` → ne charge que les médias du joueur ; retirer filtre + classifier mine/teammate via `match_participants` |
| 34 | Media | Grille images 4 cols | 🟢 | Composant existant |
| 35 | Media | Lecteur vidéo intégré | ⚪ | `<video controls>` HTML5 |

### 6.3 Récapitulatif

| Catégorie | Nb | % |
|-----------|--:|--:|
| 🟢 READY | 12 | 34 % |
| 🟡 DATA-OK | 14 | 40 % |
| 🟠 REPO-MISSING | 3 | 9 % |
| 🔴 DATA-MISSING | 2 | 6 % |
| ⚪ FRONT-ONLY | 4 | 11 % |

**33/35 viz (94%) sont faisables sur l'architecture Go actuelle**, dont 12 sans aucun travail backend. Les 2 blockers réels sont documentés en § 7.

---

## 7. Plan de portage par phases (réajusté selon faisabilité)

> Les phases sont ordonnées par effort croissant et risque croissant.
> Chaque phase est livrable indépendamment et apporte un gain visible immédiat.

### Phase A — Câblage front des READY (1-2 jours, **gain visible immédiat**)

**Aucune modification backend.** Câbler côté front uniquement les 12 viz 🟢 READY :
- KPIs Date (#1), Playlist (#3), Mode/Map (#4), Performance (#6), Kills/Deaths/Assists/KDA/Avg Life (#9, #10, #11)
- Tableau Arme/Frags (#15) — réutilise donut data
- Stacked bars Killer→Victime (#25) — agrégation côté front
- Scoreboard MVP/LVP/best/worst (#27)
- Grille médailles dans onglet Citations (#32) — réutilise `summary_tab.medals`
- Grille images media (#34)
- Légendes/popovers (#17, #24, #30)
- Lecteur vidéo HTML5 (#35)

**Livrable Phase A** : 12 viz visibles, page partiellement enrichie.

**Tests Phase A** : typecheck `npm run typecheck` sans erreur ; vérifier que les 12 viz s'affichent sur un match réel sans console error.

### Phase B — Enrichissements DTO triviaux (2-3 jours backend)

Viz 🟡 DATA-OK low-effort (données déjà chargées, juste à propager) :
- **#2 Dominance flag** : ajouter calcul des 5 règles dans `service/match_view_service.go` à partir du scoreboard
- **#5 Map thumbnail URL** : helper qui construit l'URL `/assets/maps/{title}/{map_id}/image` dans `MatchViewHeader`
- **#8 MMR équipe/ennemi** : propager `s.TeamMMR/EnemyMMR` lus dans `Q12MatchScoreboard` vers `MatchSummaryKpis`
- **#26 KD différentiel multi-joueurs** : généraliser `ComputeKDTimeline` (boucle xuid)

**Livrable Phase B** : Header complet (avec dominance + map thumbnail) + KPI MMR + Combat enrichi (KV stacked + KD multi).

**Tests Phase B** : tests unitaires pour les 5 règles `dominance_flag` dans `analysis/match_impact_test.go` ; test httptest handler vérifie `team_mmr`/`enemy_mmr` non nuls sur fixture scoreboard ; `go vet ./...` sans warning.

### Phase C — Repo-missing (2-3 jours, SQL pur)

3 viz 🟠 :
- **#7 Bloc Rang complet** : étendre `Q22MatchSkillRank` avec LEFT JOIN `metadata.career_ranks` (image_path, sub_tier_start, tier_size) + remplir `had_bot_teammate`/`bot_outcome` depuis scoreboard
- **#31 Anneaux citations** : ajouter `MatchViewRepo.GetMatchCitations(matchID)` (cf. `HomeRepo.LoadMatchCitations` pour template) + remplacer `[]` ligne 196 du service
- **#33 Media mine/teammate** : élargir `Q24MatchMedia` (retirer filtre `player_slug`, JOIN `match_participants`) + ajouter champ `Section` au DTO

**Livrable Phase C** : Header rang visuellement complet + onglet Citations rempli + Media sectionné.

**Tests Phase C** : test `platform/duckdb` avec DuckDB `:memory:` pour chaque nouvelle méthode repo (`GetMatchSkillRankWithCareer`, `GetMatchCitations`, `GetMatchMediaItems`) ; test service avec mock `port.MatchViewRepository` vérifiant que les sections ne sont plus `[]` ; `go test ./internal/... -race`.

### Phase D — Refonte tug-of-war + impact badges 9 types (3-4 jours)

- **#20 Refonte `ComputeTugOfWar`** : exposer `MyKills/EnemyKills/CumulMy/CumulEnemy` par bin + DTO `MatchTugOfWarBucket`
- **#18 Étendre `ComputeSingleMatchImpact`** : porter les 5 badges manquants (clutch_finisher, last_casualty, last_group_kill, first_group_death, top_gun, top_killer, silent_hero, false_brother) + enrichir DTO `MatchImpactBadge`
- **#19 Annotations sur la timeline existante** : composant front réutilisant les badges enrichis (anti-collision verticale)

**Livrable Phase D** : tug-of-war visuellement aligné Python + 9 badges + timeline annotée.

**Tests Phase D** : tests unitaires `analysis/tug_of_war_test.go` sur fixture kill sequence (vérifier `CumulMy`/`CumulEnemy` par bin) ; `analysis/match_impact_test.go` couvre les 9 types de badges (dont cas `top_gun`, `silent_hero = top_gun exclusif`).

### Phase E — Nouveaux modules analysis Combat (4-5 jours)

- **#21 `analysis/kill_streaks.go`** : `detect_streaks(min_kills=3, gap_s=60)` avec reset sur death/gap
- **#21 `analysis/kill_feed.go`** : filtrer kills hors streaks
- **#22 `analysis/cadence.go`** : `compute_cadence_buckets` × 3 granularités (15s, 30s, 60s) + MA window=3
- DTOs `MatchKillStreak`, `MatchKillFeedItem`, `MatchCadenceBucket` × 3

**Livrable Phase E** : kill feed avec streaks + cadence histogram fonctionnels.

**Tests Phase E** : `analysis/kill_streaks_test.go` — cas streak interrompu par mort, cas gap >60s, cas min_kills=3 exactement ; `analysis/cadence_test.go` — vérifier MA window=3 sur séquence connue ; `go test ./internal/analysis/... -race`.

### Phase F — Antagonists hybride 2-passes (3-4 jours)

- **#23 `analysis/antagonists.go`** : port complet de `compute_personal_antagonists` (pass1 certain ±5ms, pass2 estimé via `_choose_best`, validation officielle vs scoreboard)
- DTO `MatchAntagonistCard` (cmp_color, has_estimated, certain/estimated counts)

**Livrable Phase F** : cards Nemesis/Bully visuellement et logiquement alignées Python.

**Tests Phase F** : `analysis/antagonists_test.go` — cas nemesis = bully (match 1v1), cas kill estimé vs certain, cas tie-breaker XUID, tolérance ±5ms exacte ; vérifier `CmpColor` produit "red"/"green"/"violet"/"slate" selon ratio.

### Phase G — Historiques pour summary charts (2-3 jours)

- **#12, #13** : étendre `analysis/match_history_avg.go` pour spree/HS/perfect ; ajouter `LoadPlayerMatchHistory` au repo ; câbler `HasHistAvg=true`
- Backend : appliquer `WEAPON_FUSION_MAP_ID` + cap remainder dans `Q16WeaponKills` (#14)
- Frontend : `<GroupedBarChart>` × 2 (avec patterns SVG hachuré/pointillé) + `<DonutChart>`

**Livrable Phase G** : Summary tab visuellement complet sauf Radar.

**Tests Phase G** : `platform/duckdb` `:memory:` pour `GetPlayerMatchHistoryAvg` (vérifier `HistMatchCount` > 0 sur fixture) ; test service mock vérifiant `HasHistAvg=true` ; `analysis/weapons_test.go` — vérifier fusion `WEAPON_FUSION_MAP_ID` + cap remainder non négatif.

### Phase H — Encounters badges (3-4 jours)

- **#29** : enrichir Q23 (winrate_as_ally, winrate_vs_enemy, kills_dealt, deaths_suffered, last_seen)
- Créer `analysis/encounters.go` : `compute_encounter_badges` (Allié+, Coriace, Tough nut) + `filter_encounter_xuids` (ghost + friends)
- Helpers FR/EN : ordinal + relative_date
- Frontend : `<EncountersTable>` riche + `<EncounterBadgeChip>`

**Livrable Phase H** : Team tab avec tableau Encounters visuellement aligné Python.

**Tests Phase H** : `analysis/encounters_test.go` — cas badge Allié+ (WR ≥ 0.65, count ≥ 2), cas Coriace (WR ≤ 0.35, count ≥ 3), cas Tough nut (deaths ≥ 3 et ratio ≥ 2.0), cas ghost filter (0 rencontre hors ce match) ; test service mock vérifiant `Badges` non vide.

### Phase I — Panneau détail joueur étendu (4-5 jours, workaround #28)

**Sections faisables sans toucher l'archi (charger depuis shared DB)** :
- Section Armes par-joueur : boucle sur scoreboard, appel `repo.GetMatchWeaponKills(matchID, xuid)` pour chaque
- Section Médailles par-joueur : idem `GetMatchMedals`
- Section Expected K/D/A delta par-joueur : idem `GetMatchExpectedStats`
- Section Antagonist par-joueur : reparamétrer KV pairs avec autre myXUID

**Sections dégradées (architectural blocker)** :
- Section Local Performance + Skill rank + bot note : visible uniquement pour `is_me` (les autres joueurs : badge "Données partagées seulement" + lien profil Explorer)

Risque N+1 sur scoreboard détail : implémenter préchargement batch `WHERE xuid IN (...)` pour weapons/medals/expected.

**Livrable Phase I** : Team tab avec panneau détail riche pour tous joueurs (Local section dégradée pour autres, voir blockers § 8).

**Tests Phase I** : test service mock — vérifier que `DetailHasPlayerDB=false` produit `ErrCapabilityNotSupported` (pas de panic, pas de 500) ; test N+1 — vérifier que le batch `GetMatchWeaponKillsBatch` émet 1 requête SQL pour N joueurs ; typecheck frontend sur `PlayerDetailPanel`.

### Phase J — Radar Participation 6 axes (6-8 jours, **le plus risqué**)

Le plus complexe — à garder pour la fin :

1. **Backend repo** : créer `MatchViewRepo.GetMatchPSA(matchID, xuid)` (première lecture de `personal_score_awards` côté Go)
2. **Backend analysis** :
   - `analysis/participation_profile.go` : port complet de `compute_participation_profile` (10 familles regex + 12 patterns objectif + axe Survie composite)
   - `analysis/participation_thresholds.go` : `compute_global_radar_thresholds` (cache process cross-DB)
   - Helpers `_get_mode_family`, `_is_objective_mode_from_pair_name`
3. **Backend DTO** : `MatchParticipationRadar` exposé dans `MatchSummaryTab`
4. **Frontend** : `<RadarChart6>` recharts + légende textuelle conditionnelle (`hints_visible()`)

**Fallback gracieux** (cf. § 12.3) : si PSA absent pour le joueur, basculer sur `create_participation_radar` 5 axes (calcul direct depuis match_participants, sans seuils dynamiques).

**Livrable Phase J** : Match View ISO avec v7/cockpit (sauf workaround #28 documenté).

**Tests Phase J** : `analysis/participation_profile_test.go` — cas mode objectif (Oddball) vs Slayer, cas durée=0, cas deaths=0 (avg_life infinie), cas mode inconnu (`other`) → axes à 0 gracieusement ; vérifier valeurs axes cohérentes Python (diff < 2%) sur 3 matchs de référence annotés ; test fallback radar 5 axes si PSA absent.

### Phase K — Polish & vérification ISO (2-3 jours)

1. i18n FR/EN complet (toutes clés `mv_*`, `mvc_*`, `mvp_*`, `viz_t(*)`, `radar_desc_*` — ~150 clés)
2. Conformité règle 20 : couleurs via `tokenCssVar` / `getSeriesColors` exclusivement
3. **Comparaison ISO** : screenshots Python vs Go pour 5 matchs représentatifs (Slayer Ranked, BTB Open, Firefight, Oddball objectif, FFA Fiesta) — différence visuelle attendue zéro
4. Tests E2E : navigation prev/next, abandon, bot teammate, firefight, FFA, mode 1v1
5. Performance : TTFB < 500 ms, pas de N+1 dans les profile traces

**Tests Phase K** : `go test ./...` passe sans erreur ; `go vet ./...` sans warning ; `npm run typecheck && npm run lint` sans erreur ; `grep -r 'fmt\.Println\|log\.Printf' apps/go-api/internal/` retourne vide.

---

### 7.1 Estimation totale

| Phase | Effort | Cumulé |
|-------|--------|--------|
| A — Câblage READY | 1-2 j | 2 j |
| B — DTO triviaux | 2-3 j | 5 j |
| C — Repo-missing | 2-3 j | 8 j |
| D — Tug-of-war + 9 badges | 3-4 j | 12 j |
| E — Streaks + cadence | 4-5 j | 17 j |
| F — Antagonists 2-passes | 3-4 j | 21 j |
| G — Historiques summary | 2-3 j | 24 j |
| H — Encounters badges | 3-4 j | 28 j |
| I — Détail joueur étendu | 4-5 j | 33 j |
| J — Radar Participation | 6-8 j | 41 j |
| K — Polish ISO | 2-3 j | 44 j |

**Total estimé : ~35-45 jours-homme backend** + travail front en parallèle (recharts custom shapes, components SVG, i18n).

---

## 8. Blockers et workarounds

### 8.1 🔴 Blocker #16 — Radar Participation 6 axes

**Diagnostic** :
- Table `personal_score_awards` (player DB) JAMAIS lue côté Go (un seul `COUNT(*)` dans `post_sync_deltas.go` pour télémétrie sync)
- Aucune fonction `analysis/participation_profile.go` PSA-based
- La fonction `ComputeParticipationProfile` qui existe en Go est le radar squad-level (kills/deaths) — **ce n'est PAS le même radar**
- Aucun cache `compute_global_radar_thresholds` cross-DB

**Plan d'attaque** : Phase J (la plus lourde, ~1 semaine).

**Fallback gracieux documenté** : si PSA absente pour un joueur (DB partielle, sync incomplète), basculer sur le radar 5 axes (`create_participation_radar`, cf. § 10.3) calculé directement depuis `match_participants` — pas d'ISO sur ce cas dégradé mais évite l'écran vide.

### 8.2 🔴 Blocker #28 — Panneau détail joueur (sections Local pour les AUTRES joueurs)

**Diagnostic architectural** :
- L'architecture Go actuelle ouvre **1 player DB à la fois** via le pool (`apps/go-api/internal/platform/duckdb/pool.go`) : celle du joueur courant
- Les tables `player_match_enrichment`, `match_skill_rank`, `personal_score_awards`, `match_citations` des AUTRES joueurs sont **physiquement inaccessibles** depuis l'instance courante
- Affichage Performance Score / Skill Rank / Citations pour les autres joueurs du scoreboard nécessiterait un pool multi-player-DB

**Workaround retenu (Phase I)** :
- Sections **calculables depuis shared DB** rendues pour tous : Armes (Q16), Médailles (Q14), Expected (Q26), Antagonist (KV pairs reparamétrés)
- Sections **player-DB-only** pour les autres joueurs : le service retourne `games.ErrCapabilityNotSupported` (erreur typée, pas une erreur 500) pour les champs `detail_local_score`, `detail_local_rank`, `detail_citations` ; le frontend affiche un composant `<EmptyStateNotice>` avec lien `?Explorer&gamertag=...` — jamais de panic ni de nil pointer déréférencé

**Conséquence ISO** : pour les autres joueurs du scoreboard, le panneau détail aura ~80% du contenu Python (toutes les sections sauf Local et Citations détaillées). **Document explicite dans la PR**.

**Évolution future** (hors scope match view) : architecture multi-player-DB qui permettrait l'accès aux player DBs des coéquipiers — chantier transverse à planifier séparément.

---

## 9. Points d'attention et risques techniques

### 9.1 Logique non triviale à porter avec soin

- **`compute_personal_antagonists` (hybride 2-passes)** : algorithme à fenêtre temporelle ±5ms avec arbitrage rang officiel + tie-breaker XUID. Test : match 1v1 où nemesis = bully.
- **`compute_participation_profile`** : 10 familles de mode regex + 12 patterns objectif + axe Survie composite + seuils dynamiques par mode + cache cross-DB. Edge cases : durée=0, deaths=0, avg_life=0, mode inconnu (`other`).
- **`detect_streaks`** : reset sur death OU gap >60s. Test : streak interrompu par mort puis reprise.
- **`compute_single_match_impact`** : 9 types de badges avec règles différentes (top_killer requiert ≥2 joueurs équipe, silent_hero exclut top_killer, etc.). Anti-collision verticale des annotations à porter (même temps -> 3 niveaux).
- **`WEAPON_FUSION_MAP_ID`** : variantes (Diminisher of Hope, Rushdown Hammer -> Gravity Hammer, etc.). Liste à recopier ou centraliser dans `metadata.duckdb`.
- **Cap melee/grenade par remainder** : `remainder = api_total - film_kills` puis `melee_net = min(melee, remainder)`, `grenade_net = min(grenade, remainder - melee_net)`.

### 9.2 Performance

- Endpoint actuel : 11 sous-requêtes en `errgroup`. Enrichissement final : ~20 sous-requêtes
- Risque N+1 sur scoreboard détail (Phase I) : implémenter préchargement batch `WHERE xuid IN (...)` pour weapons/medals/expected par-joueur
- Cache process pour les seuils Radar globaux (équivalent `_global_thresholds_cache` Python) — Phase J

### 9.3 Compatibilité multi-titres

- Service Go utilise déjà `WithDataAdapter(games.TitleDataAdapter)` (Phase C+ multi-titres). Tous les nouveaux champs (MMR, dominance_flag, participation_radar) doivent passer par l'abstraction `canonical.MatchDetail` ou être annotés `halo-infinite-only`.
- Scoreboard frontend utilise `useFieldMappings()` — étendre aux nouvelles colonnes du panneau détail.

### 9.4 Couleurs et accessibilité

- Backend : couleurs hex hard-codées tolérées (sortent en JSON, consommées par le front qui résout via tokens)
- Frontend (règle 20 CLAUDE.md) : aucune couleur hex dans `apps/web/src/features/match-view/` — tout via `tokenCssVar(token)` / `resolveToken(token)` / `getSeriesColors(n, palette)`
- Migrer la palette Okabe-Ito 12 couleurs dans `apps/web/src/lib/accessibility/palettes/` si pas déjà disponible

### 9.5 Logging (règle slog obligatoire)

Tout le code Go produit dans ce portage doit respecter les patterns slog du projet :

```go
// Opérations significatives (début de chaque phase service)
slog.InfoContext(ctx, "match view: loading participation profile",
    "match_id", matchID, "player", xuid, "titleSlug", titleSlug)

// Durées (notamment pour les algos non-triviaux : antagonists 2-passes, participation thresholds)
slog.DebugContext(ctx, "match view: participation thresholds computed",
    "duration", time.Since(start).String(), "match_id", matchID)

// Toute erreur non-triviale
slog.ErrorContext(ctx, "match view: failed to load PSA",
    "err", err, "match_id", matchID, "player", xuid)
```

Clés structurées standard : `"err"`, `"match_id"`, `"player"`, `"titleSlug"`, `"duration"`.

**Interdit** : `fmt.Println`, `log.Printf`, `log.Println` dans tout fichier `internal/` — vérifier avec `grep -r 'fmt\.Println\|log\.Printf\|log\.Println' apps/go-api/internal/` avant chaque PR.

### 9.6 Modularité (seuils 500L / 80L / 5-args)

Les seuils du projet s'appliquent sans exception à tout code créé dans ce portage :

| Seuil | Valeur | Critique pour |
|-------|--------|--------------|
| Lignes par fichier | 500 L | `analysis/participation_profile.go` — la fonction Python source fait ~400 lignes : découper en `_profile_axes.go` + `_profile_thresholds.go` si nécessaire |
| Lignes par fonction | 80 L | `compute_participation_profile` et `compute_personal_antagonists` sont à risque — extraire sous-fonctions nommées |
| Arguments par fonction | 5 max | Si plus de 5 args → passer par une struct `ParticipationInput` ou `AntagonistInput` |

**Exemples de découpe préventive** pour `analysis/participation_profile.go` :
```
compute_participation_profile(ctx, input ParticipationInput) ParticipationRadar  // orchestre
_computeObjectifsAxis(psa []PSA, modeFamily string) float64                      // < 30L
_computeSurvieAxis(avgLifeS, durationS float64, deathsPerMin float64) float64    // < 20L
_normalizeAxis(raw, threshold float64) float64                                   // < 10L
_getModeFamily(pairName string) string                                           // < 40L
```

---

## 10. Critères d'acceptation ISO

Le portage est considéré complet quand la match view Go est **quasi ISO** avec la version Python v7/cockpit :

### 10.1 Conformité visuelle (test screenshot pour 5 matchs représentatifs)

Pour chaque match — Slayer Ranked, BTB Open, Firefight, Oddball objectif, FFA Fiesta — chacun des 35 éléments du § 6.2 doit être présent à l'emplacement attendu, avec le bon type de visualisation, le bon nombre de séries, les bonnes couleurs (via tokens), les bons libellés FR/EN.

**Tolérance** :
- Différence de pixel exacte non requise (différences de moteur de rendu Plotly vs recharts acceptées)
- Disposition globale identique (header → 5 onglets → mêmes sections dans le même ordre)
- Aucun élément manquant ni ajouté sans justification

### 10.2 Conformité fonctionnelle

1. KPI numériques (MMR, expected, hist_avg, performance, perfect_kills) : valeurs **exactement identiques** entre Python et Go (tolérance arrondi 0.01)
2. Badges d'impact correctement attribués (test de référence sur match annoté manuellement) : 9 types implémentés
3. Radar Participation : 6 axes normalisés, valeurs cohérentes Python (différence <2% sur les axes)
4. Tableau Encounters : badges Allié+/Coriace/Tough nut sur cas de test connus
5. Onglet Citations : anneaux progression + grille médailles
6. Onglet Media : sections mine/teammate + lecteur vidéo
7. Panneau détail joueur : Armes + Médailles + Expected + Antagonist visibles pour TOUS joueurs ; Local + Citations détaillées limitées à `is_me` (workaround documenté § 8.2)

### 10.3 Conformité technique

8. Aucune couleur hex hard-codée dans `apps/web/src/features/match-view/` (sauf exceptions justifiées par commentaire — règle 20)
9. Tous les libellés sont traduits FR/EN
10. TTFB < 500 ms sur match local, pas de N+1 visible dans le profile

### 10.4 Non-objectifs explicites (hors scope ISO)

- Les 9 fonctions de viz latentes (cf. § 10) restent inactives — pas de réintroduction sans validation utilisateur
- Local Performance / Skill rank / Citations détaillées des AUTRES joueurs (workaround Phase I documenté)
- Animations Plotly avancées (transitions, légendes interactives sophistiquées) non garanties à l'identique sous recharts

---

## 11. Annexes

### 11.1 Liste exhaustive des 30+ visualisations Python (référence)

| # | Onglet | Nom | Type | Présent Go ? |
|---|--------|-----|------|:---:|
| 1 | Header | Carte KPI Date | Card text | partiel |
| 2 | Header | Carte KPI Score + badge dominance | Card text | partiel |
| 3 | Header | Carte KPI Playlist | Badge | OK |
| 4 | Header | Carte KPI Mode/Map | Card text | partiel |
| 5 | Header | Map thumbnail | Image | NON |
| 6 | Header | Bloc Performance (font 4.2em) | KPI | partiel |
| 7 | Header | Bloc Rang LUSR/CSR (image+barre+delta+bot note) | Composite | partiel |
| 8 | Summary | KPI MMR Équipe vs Ennemis | Card | NON |
| 9 | Summary | KPI Kills réel vs attendu | Card | OK |
| 10 | Summary | KPI Deaths réel vs attendu (inverse) | Card | OK |
| 11 | Summary | KPI Average Life | Card | OK |
| 12 | Summary | Bar chart F/D/A Réel/Attendu/Hist | Plotly grouped bar | NON |
| 13 | Summary | Bar chart Spree/HS/Perfect Réel/Hist | Plotly grouped bar | NON |
| 14 | Summary | Donut weapon kills | Plotly pie | NON |
| 15 | Summary | Tableau Arme/Frags | HTML table | NON |
| 16 | Summary | Radar Participation 6 axes | Plotly polar | NON |
| 17 | Summary | Légende textuelle 6 axes | Markdown | NON |
| 18 | Combat | Badges d'impact 9 types | HTML cards | partiel |
| 19 | Combat | Timeline K/D cumulé joueur + annotations impact | Plotly line + flèches | partiel |
| 20 | Combat | Tug-of-war dominance équipe stacked + annotations cumul | Plotly stacked bar | partiel |
| 21 | Combat | Kill feed individuel + KillStreak | Plotly scatter+lines (2 panneaux) | NON |
| 22 | Combat | Histogramme cadence bicolore + MA + segmented control | Plotly bar+line | NON |
| 23 | Combat | Cards Nemesis + Bully colorées | HTML cards | NON (liste textuelle) |
| 24 | Combat | Caption Debug antagonistes | Caption | NON |
| 25 | Combat | Stacked bars Killer→Victime | Plotly horiz stacked bar | NON |
| 26 | Combat | Timeline K/D différentiel cumulé tous joueurs | Plotly multi-line | NON |
| 27 | Team | Scoreboard table + MVP/LVP/best/worst | HTML table | OK |
| 28 | Team | Panneau détail joueur (Armes/Médailles/Citations/Expected/Antagonist/Local/Footer) | HTML accordion | partiel (`is_me` seulement) |
| 29 | Team | Tableau Encounters historique + badges | HTML table | NON (liste basique) |
| 30 | Team | Popover Légende badges | Popover | NON |
| 31 | Citations | Anneaux progression circulaires | CSS SVG | NON (vide) |
| 32 | Citations | Grille médailles 8 cols + tooltips | HTML grid | NON (vide) |
| 33 | Media | Sections mine / teammate | Streamlit groups | NON (mélangé) |
| 34 | Media | Grille images 4 cols | Streamlit columns | OK |
| 35 | Media | Lecteur vidéo intégré | st.video | NON |

### 11.2 Références fichiers Python (à consulter pendant le portage)

- `src/ui/pages/match_view.py:213` — `render_match_view` (entry)
- `src/ui/pages/match_view_charts.py:60` — `render_expected_vs_actual` + `_render_spree_headshots:258`
- `src/ui/pages/match_view_weapon_kills.py:165` — `render_weapon_kills_section`
- `src/ui/pages/match_view_participation.py:45` — `render_participation_section`
- `src/analysis/participation_radar.py:303` — `compute_participation_profile`
- `src/ui/pages/match_view_players.py:245` — `render_match_impact_section`
- `src/visualization/_match_impact_events.py` — `compute_single_match_impact`
- `src/ui/pages/match_view_players_timeline.py:37` — `render_team_dominance_section` + `:138 render_match_cadence_section` + `:228 render_kd_timeline_section`
- `src/ui/pages/match_view_players_nemesis.py:242` — `render_nemesis_section`
- `src/ui/pages/match_view_scoreboard.py:340` — `render_match_scoreboard`
- `src/ui/pages/match_view_scoreboard_detail.py:75` — `load_scoreboard_player_extra_data` + `:155 render_scoreboard_player_detail_html`
- `src/ui/pages/match_view_encounters.py:227` — `render_encounter_section`
- `src/ui/pages/match_view_encounters_logic.py` — `compute_encounter_badges` + `filter_encounter_xuids`
- `src/ui/pages/match_view_citations.py:16` — `render_match_citations_section` + `:157 render_medals_tab`
- `src/ui/pages/match_view_helpers.py:273` — `render_media_section`
- `src/ui/pages/match_view_rank.py:21` — `_build_match_rank_html`

### 11.3 Références fichiers Go actuels

- `apps/go-api/internal/api/handlers/match_view.go:27` — `GetMatchView`
- `apps/go-api/internal/service/match_view_service.go:58` — `GetMatchView`
- `apps/go-api/internal/platform/duckdb/match_view_repo.go` — implémentation repo
- `apps/go-api/internal/domain/match_view.go:10` — DTO
- `apps/web/src/features/match-view/MatchViewPage.tsx` — page principale
- `apps/web/src/features/match-view/MatchScoreboard.tsx` — scoreboard
- `apps/web/src/features/match-view/PlayerDetailPanel.tsx` — détail joueur
- `apps/web/src/features/match-view/queries.ts` — hooks TanStack Query
- `apps/web/src/routes/players/$playerSlug/matches/$matchId.tsx` — route

---

## 12. Fonctions de viz latentes — jugées non pertinentes pour le portage initial

> Source : croisement `.ai/CHARTS_AND_TABLES.md` (généré 2026-04-01) vs code réel v7/cockpit audité le 2026-04-26.
>
> Ces fonctions existent dans `src/visualization/` mais ont été **délibérément retirées** de la match view lors de la simplification v7/cockpit. Elles sont jugées **hors périmètre des phases 1 à 9**. Une réévaluation est possible en **phase finale (post-Phase 9)** uniquement si les retours utilisateurs le justifient. Ne pas les porter par défaut.

### 12.1 Nemesis / antagonist — fonctions retirées (`src/visualization/_antagonist_duels.py`, `_antagonist_kv.py`)

**Statut : hors périmètre. Réévaluation possible post-Phase 9.**

| Fonction | Ce qu'elle fait |
|----------|----------------|
| `plot_top_antagonists_bars` | Barres doubles horizontales top 5 adversaires (j'ai tué N fois / m'a tué N fois) |
| `plot_duel_history` | Historique chronologique win/loss du duel 1v1 + K/D cumulé vs l'adversaire |
| `plot_nemesis_victim_summary` | Tableaux résumé multi-adversaires |
| `create_kd_indicator` | `go.Indicator delta` K/D ratio vs cet adversaire, delta vs moyenne générale |
| `plot_killer_victim_heatmap` | Heatmap 2D tueurs × victimes (alternative à la stacked bar retenue en Phase 5) |

### 12.2 Participation — fonctions retirées (`src/visualization/participation_charts.py`)

**Statut : hors périmètre. Réévaluation possible post-Phase 9.**

| Fonction | Ce qu'elle fait |
|----------|----------------|
| `plot_participation_pie` | Donut pie répartition du score par catégorie (kills, assists, obj, véhicules, pénalités) |
| `plot_participation_bars` | Barres horizontales empilées par catégorie |
| `plot_participation_by_match` | Barres empilées chronologiques multi-matchs — non pertinent dans une vue match unique |

### 12.3 Radar participation 5 axes — version simplifiée (`create_participation_radar`)

**Statut : hors périmètre. Exception possible si PSA indisponible.**

Fonction distincte de la cible du portage. La cible est `create_participation_profile_radar` (6 axes, Phase 4).

| Fonction | Axes | Dépendance |
|----------|------|------------|
| `create_participation_radar` (inactive) | kill%, assists%, obj%, véhicules%, pénalités% | Aucune — calcul direct depuis match_participants |
| `create_participation_profile_radar` (active, cible) | Objectifs, Combat, Support, Score, Impact, Survie normalisés | `personal_score_awards` + seuils par mode |

La version 5 axes pourrait servir de **dégradé gracieux** uniquement si `personal_score_awards` est absent pour un joueur — à documenter comme cas limite lors de la Phase 4, mais ne pas la porter par défaut.

### 12.4 Kills par arme — barres horizontales (`plot_top_weapons`)

**Statut : hors périmètre. Réévaluation possible post-Phase 9.**

Alternative au donut (Phase 3). La barre horizontale est plus lisible en accessibilité daltonisme. En v7/cockpit, le donut + tableau HTML ont été retenus. Ne pas porter les deux par défaut.
