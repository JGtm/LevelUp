# Audit : données envoyées par l'API vs données lues par le web (2026-10-21)

Pièce du dossier du plan « performance des pages » (résumés par match, fenêtre de l'Accueil,
sur-chargement). Audit statique en lecture seule (agent Explore, Sonnet), aucun appel HTTP :
**aucune taille en octets n'est mesurée ici**. Champs « jamais lus » = nom JSON absent de
`apps/web/src` (hors `generated.ts`, `lib/api/types.ts`, tests, fixtures) : la liste est sûre mais
non exhaustive (un nom générique peut masquer un champ mort). À RE-VÉRIFIER sur pièces avant tout
retrait (doctrine plan-execution règle 4).

Mesures prod associées (levelup-76, 2026-10-20, JGtm, iPhone émulé) : Accueil 28,6 s puis 20 s,
2,9 Mo de JSON décodé (292 Ko transférés), `recent_matches` = 1 182 éléments (2,94 Mo), 158 000
nœuds DOM (156 000 dans le carrousel), 161 Mo de tas JS ; le rechargement « analyse en cours »
(`homeHasAnalysisPending`) regarde toute la liste et relance la requête toutes les 60 s.
Journal prod du 20/10 : Accueil 15,2 s en moyenne (16 requêtes), vue match 14,6 s (5).

## Par page

| Page | Liste non bornée | Affiché | Gain estimé |
|---|---|---|---|
| Accueil `GET /pages/home` | `recent_matches` = tout l'historique (`home_service.go` : `BuildRecentMatchesWithFavoritesFromCanonical(rows, len(rows), …)`), ~55 champs par tuile, 4 enrichissements par lot sur toute la liste | ~5 tuiles visibles ; le carrousel (`components/ui/carousel.tsx`, scroll-snap) monte TOUTES les `MatchCard` | **fort** |
| Explorer `matches-query` / `player-query` | le web demande `page_size: 10000` (`ExplorerPage.tsx`), 38 champs par ligne, tri client délibéré | 20 lignes par page | **fort** (gros comptes) |
| Vue match + rejeu | vue match bornée au match ; MAIS elle télécharge l'artefact de rejeu complet (1,5-2,7 Mio d'après `lib/replay/queries.ts`) pour 3-4 graphes (score, usages, armes) | quelques calques | **fort** |
| Rejeu `GET …/replay` | bloc `coverage` : 500 noms JSON dont 333 jamais lus (6 sous-clés lues) ; ~50 autres noms morts hors coverage | — | moyen à fort |
| Relations `POST /pages/palmares/relations` | `relations[]` = tous les joueurs croisés >= 2 matchs, sans LIMIT (`queries_career_encounters.go`) | 25 lignes par page | moyen à fort |
| Timeseries | `match_rows`, `intensity_rows*` sur le périmètre filtré (petit par défaut : dernière session) ; `solo_session_perf` sur tout l'historique solo ; blocs calculés jamais lus : `cumul_tab`, `intensity_tab`, `outcomes_over_time`, `kill_types` (ce dernier sert en interne à `FragDistribution`) | — | moyen |
| Escouade `POST /pages/teammates` | `match_history` du périmètre ; `session_labels.solo` jamais lu | 20 par page | moyen |
| Carrière | `xp_history`, `friends_xp_history` complets (Q7 sans LIMIT), `lusr.checkpoints` complets | un point par jour | faible à moyen |
| Sessions | `GET /pages/sessions` sans appelant web (clients externes non vérifiés) ; détail : `available_sessions` non borné, historique rechargé à chaque appel ; `compare_metrics` calculé jamais lu | — | faible |
| Tactique | `voisines[]` (rasters des autres questions, préchauffage voulu) ; `centre_x/y` dérivables, répétés par cellule | — | faible à moyen |
| Comparaison | bornée | — | faible |
| `POST /filters/resolve` (toutes les pages Stats) | `session_options.all_sessions[]` sur tout l'historique ; `solo_labels`/`squad_labels` doublons jamais lus | — | faible à moyen |

## Reconstructions serveur sans usage (Accueil)

- `buildFavoriteMatchListCanonical` reconstruit toutes les tuiles pour garder les favoris.
- La liste favoris est enrichie une seconde fois (médailles, citations, commendations, frags assistés).
- `SanitizeFloatsForJSON` (parcours réflexif), `json.Marshal` puis SHA256 (ETag) sur tout le corps.
- `recent_media` chargé (`LoadRecentMedia`) mais lu nulle part : `RecentMediaRail` a son propre hook.
- `HomeMatchesCache` : `Get` ignoré, `Set(nil, nil)` — code mort.
- Navigation : `goToMatch` écrit les ids de TOUTE la liste dans `sessionStorage` à chaque clic.
- Un changement de favori invalide la requête de l'Accueil et la recharge en entier.

## Champs jamais lus (échantillon à re-vérifier)

- Accueil : `skill_progress_pct`, `skill_points_in_tier`, `skill_playlist_group` (tuile), `hero.kpis.global_ratio`, `hero.trend.*`, `session.global_ratio`, `rank_number`, `backdrop_image_url`, `recent_media`.
- Vue match : `combat_tab.kd_timeline`, `.nemesis_duels`, `.impact_roles`, `personal_result`, `header.waypoint_url`, `net_kills`, `hist_match_count`, `hist_mode_category`, `kills_stddev`, `deaths_stddev`, `top_weapon_id`, `is_mvp`, `is_lvp`, `victim_team_id`, `kills_as_flag_carrier`.
- Carrière : `current_season` (`resolveCurrentSeason` calculé pour rien), `summary.rank_number`, `.rank_tier`, `lusr.current_*`, `.trend_label`.
- Relations : `overview.distinct_players`, `.allies_count`, `.rivals_count`, `.core_count`, `enemy_wins`, `teammate_wins`, `first_seen_at`, `rivalries[].rolling_*`, `.frag_gap`.
- Explorer : `available_ranked_contexts`, `csr_tier`, `csr_sub_tier`, `match_url`, `player_outcome`, `total_medals`, `unique_medals` ; `include_export_hint: true` toujours envoyé.

## Non vérifié

Tailles réelles en octets ; part du `coverage` dans l'artefact de rejeu ; fréquence du périmètre
« tout » en Timeseries / Escouade ; clients non web de `GET /pages/sessions` ; source LUSR active
de la Carrière ; nombre de voisines Tactique ; coût serveur par étape (sections `home_*` et
`mv_<source>` livrées en 8.0.1, à relever en prod).
