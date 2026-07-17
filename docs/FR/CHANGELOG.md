# Journal des modifications

> Version française du [CHANGELOG.md](../CHANGELOG.md) racine.

Toutes les modifications notables de ce projet sont documentées ici.

Le format est basé sur [Keep a Changelog](https://keepachangelog.fr/fr/1.1.0/).

## [Non publié] - 2026-06-15

> Entrée consolidée regroupant le travail livré depuis le 2026-05-02 (v7.0 pas encore publiée). Résumé par domaine, pas commit par commit.

### Ajouté (React / TypeScript)

- **Page Sessions — refonte** — refonte UX complète : graphes F/D/A par match et par minute, score de performance par tier, radar F/D/A, nuage TC/RD et engagement par match avec axes explicites et bandes de sous-palier ; drawer de comparaison A/B avec échelles partagées ; métriques en vue solo (Taux de victoire, KDR, kills/match, delta de rang) ; fenêtre de session adaptative ; les sessions d'un seul match ne sont plus listées.
- **Explorer — profils de combat & rivalités** — profil de combat en live (lecture seule) de n'importe quel joueur non suivi (rang de carrière + grade Spartan, graphes de cadence, cache court) ; métriques de dominance et rencontres issues de l'historique partagé ; export CSV ; filtres en cascade sur cinq dimensions ; barres de matchs par saison avec badge rang CSR ; recherche partielle par ID de match.
- **Explorer — briefing V2** — ajustements du briefing rétrospectif (mode Matchs) : la carte de classement est scindée **par type de rating** (CSR / LUSR), chacun affiché en progression de paliers connus (ex. `Or II → Or VI`) plus la moyenne de points par match — plus aucun cumul brut ; nouvelles cartes **Séries** (meilleure série de victoires / pire série de défaites sur tout le scope filtré) et **Moments forts** (compteurs de dominance) ; carte contexte **solo/escouade conditionnelle** (affichée uniquement quand les deux sous-groupes sont pertinents) ; les deltas `vs habituel` sont masqués quand le scope = tout l'historique (ils réapparaissent dès qu'un filtre le rétrécit), avec les dimensions du plein historique triées par taux de victoire ; en-têtes de cartes-sections unifiés ; les plages de dates incluent désormais l'année ; la dimension playlist est renommée `Par sélection`. Cela remplace l'ancien bloc `attendu vs réel` (probabilité de victoire pré-match), retiré car non fiable.
- **Compare / Face-à-face** — page dédiée avec mode miroir 3 joueurs (B vs A vs C), rang de carrière lisible + CSR all-time pour les joueurs non-locaux, stats et badges de rencontre.
- **Citations** — page dédiée avec score composite, badges LUSR/CSR et `CitationProgressRing`.
- **Ascension** — profil joueur V3 (radar, badge de style, composantes LUSR, panneau de leviers), détection de patterns comportementaux (tilt / fatigue / plateau / plafond), grille de contexte (par mode/carte/escouade), suivi de campagne avec modale de démarrage, layout 2 onglets.
- **Objectifs / Prestige** — défis + défis d'escouade (collectifs / compétitifs), arcs narratifs (création libre, presets, suppression, bonus de complétion), mode guidé/piloté par le coach, leaderboard PP.
- **Coach d'escouade** — strip d'orientation d'escouade, biais du pool de défis par axes de performance, `CoachFocusCard` (« cap du moment ») avec signal soft-négatif.
- **Classement mondial** — classement CSR mondial enrichi de stats natives par joueur, multi-saisons avec indicateur de tendance inter-saison, joueurs locaux remontés en tête.
- **Médias — lecteur HLS** — lecture des clips dans le navigateur (`hls.js`) avec sélecteur de piste audio (jeu / voix / mix complet) ; modale de réassociation manuelle `MediaMatchPicker` (fenêtre ±15 / ±60 / ±180 min, miniature de carte, lobby par équipe, badge de résultat) appelant `POST /players/{slug}/media/associate`.
- **Dashboard admin** — UI de monitoring complète (cycles de sync + sparklines de tendance, convergence, invariants d'intégrité des données, santé des tokens, attribution des appels API Halo par joueur, collecteur d'erreurs récurrentes, logs, perf).
- **Paramètres — onglet Sauvegarde** — statut des snapshots restic, déclenchement manuel, contrôle d'intégrité par base.
- **CSR / saisons** — sélecteur de saison CSR + saisons disponibles, seuils de placement dynamiques, badges classés, pastilles de saison avec repli cascade-aware.

### Ajouté (API Go)

- **CSR par match & par playlist** — `GetPlaylistCsr`, CSR par match via RankRecap, `season_id` + `is_ranked` à l'écriture, seuils de placement dynamiques par saison, référence autoritative des playlists classées, distribution automatique du CSR des coéquipiers, CLI `backfill-csr-history`.
- **LUSR v2 (TrueSkill2)** — `internal/analysis/skill_v2/` graphe de facteurs + expectation propagation, pondération par temps joué, ordre des abandons, probabilité de victoire pré-match, calibration des paliers, protections anti-volatilité ; mode shadow puis canonical, avec replay offline et outillage de ré-estimation des hyperparamètres par lot.
- **Classement CSR mondial** — scraper Halo Waypoint, enrichissement `world_player_season_stats` (append-only), agrégateur multi-tokens, cron dédié + header provider, CLI de backfill.
- **Coach advisor & coach d'escouade** — orchestration génération/acceptation des propositions (ADR 0020/0028), hook post-sync, endpoints HTTP, orientation d'escouade + biais du pool de défis.
- **Sauvegarde / restauration** — `pkg/duckdbbackup` scheduler restic générique + adaptateur LevelUp, `cmd/restore` restauration à une date, logging structuré.
- **Sync convergent** — résolution autonome des noms d'assets au primary write, filet hebdomadaire pour la traîne, cron de rafraîchissement du catalogue in-cycle, gate de déduplication cross-source, gate d'invariants de qualité des données.
- **Timeline de match / T0** — `MatchTimeline` + `ComputeT0`, vraie durée de gameplay (décompte pré-partie soustrait), câblage `CorrectEvents`/`CorrectImpactEvents`, re-normalisation TZ de `first_joined_time`/`last_leave_time`.
- **Succès (Xbox)** — CLI `sync-achievements` + `RunAchievementsOnly`, service de merge cross-DB, handler HTTP, filtre par catégorie.
- **Contrôle d'accès** — ownership joueur multi-user + middleware `RequirePlayerOwnership` (ADR 0029), verrouillage d'instance, gating « page indisponible » avec `apiErrorCode`.
- **Observabilité** — `event_id` propagé à tous les flows sync/auth/watcher, métriques de concurrence expvar, invariants d'intégrité des données, endpoints de diagnostic admin.
- **DTO briefing Explorer — classement par type & nouvelles sections** — `ExplorerBriefingRanked` refondu pour émettre un `ExplorerBriefingRankedKind` par type de rating (paliers début/fin, flags de placement, points par match) au lieu d'un cumul brut ; champ additif `KPIStats.RankDeltas []RankDelta` exposant les buckets par `RatingType` existants ; nouveaux blocs `ExplorerBriefingContextSplit` (solo/escouade), `ExplorerBriefingStreaks` et `ExplorerBriefingDominance` ; re-tri des dimensions par taux de victoire en plein historique. Le calcul `expected_win_prob` et ses champs DTO ont été retirés.

### Modifié

- **Auth** — provider SISU par défaut (MSAL retiré de l'UI) ; `MultiUserTokenStore` source unique des identifiants (ADR 0023), avec détection de refresh-token mort, bannière de reconnexion et mot de passe opt-in pour re-login SSO rapide.
- **Sûreté des écritures DuckDB** — tables critiques migrées en append-only / INSERT-only pour éviter le bug de corruption ART (`match_skill_rank`, `match_csrs`, `player_csr_snapshots`, `pve_match_stats`) ; provider de base partagée (B-swap) activé par défaut.
- **Pool de tokens** — honore `Retry-After` (429/503) avec backoff exponentiel, singleflight sur le resolver pour éviter les bursts `invalid_grant`, re-scan périodique pour ajouter des tokens à chaud sans reboot.

### Corrigé

- **Affichage gamertag** — vue de lookup source unique, noms masqués résolus via `killer_victim_pairs`.
- **Fuseau horaire** — re-normalisation de `first_joined_time` (corrige les décalages T0 + ordre des abandons).
- **LUSR v2** — désync watermark vs ligne (matchs sautés), delta ordonné par `start_time`.
- **Médias** — sélection de piste audio HLS sur Chrome, remux HEVC au scan, fallback `data/media` quand le dossier de captures est invalide.
- **Escouade** — graphes vides avec 2+ coéquipiers (intersection dédupliquée), session affichée = composition exacte.

## [7.0.1] - 2026-04-29

### Ajouté

- **Module Objectifs** — nouveau centre de défis accessible depuis la barre de navigation (`/objectifs`). Page en deux onglets :
  - **Défis** — créez des défis individuels ou d'escouade sur n'importe quelle métrique Halo (kills, KDA, précision, dégâts…) avec une fenêtre temporelle, un palier (Normal / Heroic / Legendary / Mythic) et un mode d'évaluation (seuil ou cumulatif). Les défis d'escouade fonctionnent en mode **collectif** (objectif commun partagé entre membres) ou **compétitif** (les membres se classent entre eux sur la même métrique). Toggle « Défis pilotés » pour des suggestions auto-calibrées sur votre historique.
  - **Mon parcours** — rétrospective de vos arcs narratifs (séquences d'objectifs thématiques) et progression Prestige globale.

- **Système Prestige** — points Prestige (PP) gagnés en complétant des objectifs. Quatre paliers : Normal (gris), Heroic (bleu), Legendary (violet), Mythic (or). Nouveau sous-onglet **Communauté** dans Palmarès : leaderboard PP comparant votre score à ceux de votre escouade et de vos relations, dérivé automatiquement des données partagées.

- **Centre de notifications in-app** — cloche dans la barre de navigation avec badge de non-lus. Page `/notifications` : liste paginée filtrable par catégorie ou « non lus uniquement », groupée par jour, avec sélection multiple et actions groupées (marquer lu, supprimer). Rafraîchissement automatique toutes les 60 secondes. Préférences par joueur configurables depuis les Paramètres.

## [7.0.0] - 2026-04-12

### Ajouté

- **Home V7 — défis actifs restaurés via HaloStats `/decks`** — la home Mission Control réaffiche la carte des défis actifs, avec compteurs de deck, échéance réelle, titre/description localisés, dérivation du badge Waypoint et vraie progression joueur au format `x/y`.

- **Page Synthèse V7** — la nouvelle section de premier niveau après Escouade travaille désormais sur tout l'historique par défaut avec son propre sélecteur de période, regroupe les graphes de vue d'ensemble existants, ajoute un duel chart Solo vs Escouade auto-suffisant, et améliore le bucketting long terme du graphe top-vs-total pour les historiques multi-années clairsemés.

- **Media V2 — likes persistants et groupements enrichis** — les captures et vidéos peuvent désormais être likées directement dans la grille Media V2, groupées par état aimé, session ou contexte solo/escouade, avec les vrais assets coeur fournis localement.

- **Couche de persistance des défis** — nouveau module de domaine `src/data/challenges.py`, scindé en interne entre `src/data/_challenge_catalog.py` et `src/data/_challenge_snapshots.py` :
  - `challenge_definitions` + `challenge_translations` dans `metadata.duckdb`
  - `challenge_snapshots` dans chaque `stats.duckdb` joueur
  - versionnement des définitions par `content_hash`
  - snapshots joueur append-only, dédupliqués sur changement d'état via `state_hash`

- **Catalogue multi-langue des défis** — toutes les traductions titre/description exposées par le CMS sont stockées localement, normalisées en BCP-47, avec fallback `en-US` si la langue demandée n'est pas disponible.

- **Page d'authentification et d'inscription par invitation** — nouvelle page `/register` : la création de compte est réservée aux personnes disposant d'un code d'invitation transmis par l'administrateur. Le code est validé par l'API avant toute création de compte ; un code expiré ou déjà utilisé affiche un message d'erreur explicite.

### Modifié

- **Chargement des migrations** — les steps de migration sont maintenant chargés dynamiquement ; les nouveaux modules `add_challenge_metadata` et `add_challenge_snapshots` ne dépendent plus d'une liste d'imports maintenue à la main.

- **Pipeline de rendu Media V2** — la grille médias utilise maintenant des thumbnails Streamlit natifs au lieu d'une iframe par carte, avec une lightbox partagée et une avance automatique optionnelle en fin de vidéo.

### Corrigé

- **Défis home en mode live-first et failsafe** — si `metadata.duckdb` est temporairement verrouillée par un autre process Python, la carte de défis V7 continue de s'afficher depuis l'API live et saute simplement la persistance pour ce refresh, au lieu de retourner `None`.

- **Likes médias persistants au rerun et au reload** — `data/ui_prefs.json` préserve désormais correctement `media_likes` lors des merges de préférences, répare les anciens formats stringifiés et évite le cas de double bascule sur le contrôle coeur.

### Tests

- Couverture ciblée ajoutée pour la persistance des défis et l'enrichissement Home V7 dans `tests/test_challenges_data.py`, `tests/test_home_mission_control_challenges.py` et `tests/test_home_mission_control.py`.

- Couverture ciblée ajoutée pour le rendu natif des thumbnails Media V2, la persistance des likes et le fallback des callbacks de bouton dans `tests/test_media_components_sprint4.py`, `tests/test_media_v2_grid_interactions.py` et `tests/test_ui_persistence_v64.py`.

## [6.5.0] - 2026-04-10

### Ajouté

- **Coéquipiers — heatmap d'intensité par joueur** — une nouvelle section après "Complémentarité de l'escouade" affiche une heatmap match × phase (début/milieu/fin) du profil de kills de chaque membre. Un sélecteur bascule (Tous / par joueur) change de vue sans re-requêter la DB — les données sont chargées en une seule passe. Réutilise `compute_match_intensity_profiles` + `plot_match_intensity_heatmap`.

- **Discord — notifications sync/backfill séparées** — un nouveau toggle indépendant `discord_notify_backfill` distinct de `discord_notify_sync`. Les deux cases sont maintenant affichées verticalement. `notify_operation_done` route vers le bon flag selon `operation.startswith("backfill")`. La section Backfill dans les Paramètres est déplacée en dernière position, sous un `st.subheader` avec une légende d'avertissement et un expander réduit par défaut.

- **Composant info-layer partagé** — `_render_note` extrait en fonction publique `render_info_note(key, lang)` dans `src/ui/components/info_note.py`, partagée par les pages Coéquipiers et Séries temporelles.
  - 6 nouvelles clés i18n dans `teammates.py` : `tm_no_data`, `tm_impact_caption`, `tm_weapons_chart_caption`, `tm_metrics_caption`, `tm_note_radar`, `tm_note_cadence`.
  - Légendes ajoutées : heatmap d'impact, graphique barres armes, graphiques barres métriques.
  - Notes post-graphique ajoutées après le radar de synergie et la carte de cadence.
  - Toutes les légendes conditionnelles enveloppées dans `hints_visible()` sur Coéquipiers et Séries temporelles.
  - Import `EXCLUDED_WEAPON_IDS` unifié depuis `_weapon_data.py` (remplace la constante locale `_FILM_EXCLUDED_IDS`).

### Modifié

- **Paramètres V3 — `frozen=True`, `patch_settings`, écriture atomique** — refonte interne majeure de la couche paramètres :
  - `AppSettings` est désormais `frozen=True` — plus de mutation silencieuse en mémoire.
  - `patch_settings(key, value)` remplace les appels directs à `save_settings()` dans toute la base de code.
  - `_WRITE_LOCK` garantit les écritures thread-safe ; `_PROCESS_CACHE` déduplique le contenu pour éviter les I/O inutiles.
  - Pages UI de paramètres entièrement migrées vers des callbacks `on_change` — `_auto_save_show_*` et `_build_settings_from_ui` supprimés.
  - `_render_backfill_checkboxes` extrait ; `_check_settings_consistency` rendu non-bloquant.
  - `save_settings()` conservé comme wrapper CLI uniquement.
  - `path_picker.directory_input` supporte maintenant `on_change` / `args`.
  - `__init__.py` exporte `patch_settings`.

### Corrigé

- **Paramètres — écriture atomique + cascade de récupération multiplateforme** — les paramètres sont désormais écrits atomiquement via `os.replace()` après écriture dans un fichier temporaire. Cascade de récupération : restauration de sauvegarde valide → réinitialisation usine → protection fichier vide. Prévient la corruption de `app_settings.json` en cas de crash.
  - `_atomic_write` : retry `os.replace()` jusqu'à 4 tentatives (50/100/200/500 ms) pour le verrouillage de fichiers Windows.
  - `save_settings()` : traceback complet journalisé (5 niveaux) pour tracer tout écrasement accidentel futur.

- **Paramètres — `show_records` réinitialisé silencieusement à `True`** — valeur par défaut de repli corrigée de `True` → `False` dans 3 fichiers.

- **`plot_map_outcome_timeline` supprimé** — le graphique était désactivé via `if False:` dans tous les points d'appel. La fonction et son fichier source (`_maps_outcome_timeline.py`) ont été supprimés ; les clés i18n orphelines également retirées.

- **`app_settings.json.bak` exclu du contrôle de version** — contient les mêmes données sensibles que `app_settings.json` (webhook Discord, chemins locaux) ; ajouté au `.gitignore`.

### Tests

- **Paramètres V3 + écriture atomique** — 76 nouveaux tests (31 + 45) ; couverture paramètres : 77,5 % → 87,7 %.
- `TestRenderInfoNote` — 10 nouveaux cas : `hints_visible`, listes, texte gras, paragraphes, entrée vide.

---

## [6.4.0] - 2026-04-07

### Ajouté

- **Bibliothèque médias — filtres & tri** — la page Médias dispose désormais d'un panneau de filtres complet :
  - **Propriétaire** — multiselect pour afficher/masquer les sections (Mes captures / Coéquipier(s) / Non associés)
  - **Carte** — filtre par nom de carte (provenant des données de match)
  - **Mode** — filtre par label de mode normalisé
  - **Résultat** — multiselect (Victoire / Défaite / Égalité / Non terminé) basé sur les codes de résultat
  - **Contexte** — radio pour restreindre aux matchs Solo ou en Escouade
  - **Tri** — trier par date de capture (par défaut), carte, mode, résultat ou propriétaire ; bascule croissant/décroissant
  - Les filtres type (image/vidéo) et nom de fichier sont conservés depuis l'ancien panneau
  - Les médias non associés (sans match) ne sont pas affectés par les filtres de match et apparaissent toujours dans leur propre section quand elle est sélectionnée

- **Fanout CSR coéquipiers** — lors de la synchronisation d'un match classé, les données CSR de tous les co-joueurs enregistrés sont désormais automatiquement collectées depuis le payload API `skill_json` et distribuées à chaque DB joueur. Auparavant, chaque joueur devait synchroniser son propre compte pour alimenter son historique CSR sur les matchs communs.

- **Fanout badges comeback** — les badges Remontada / Débandade / Contre-Remontada sont désormais calculés pour les co-joueurs enregistrés lors du fanout de sync, en parallèle de la distribution PSA et CSR.

### Ajouté (suite)

- **Healthcheck DB** — nouveau module (`src/utils/healthcheck_db.py`) qui vérifie l'état de toutes les bases DuckDB à chaque démarrage et après chaque déploiement :
  - Vérifie la présence des tables, des vues v6 (`v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills`) et des colonnes critiques
  - Vérifie que `metadata.duckdb` est attachable depuis `shared_matches`
  - Détecte les migrations en attente
  - Auto-répare les vues v6 manquantes/cassées via `ensure_resolution_views()`
  - Mode `--deep` : intégrité référentielle (participants orphelins, doublons)
  - CLI : `python scripts/healthcheck_db.py [--verbose] [--deep] [--player GT] [--json]`
  - Intégré dans `launcher.py` — s'exécute automatiquement après les migrations au démarrage, affiche ✅/⚠️/❌ dans la console
  - Intégré dans `deploy.sh` — smoke test post-déploiement, résultats ajoutés dans `data/logs/healthcheck_deploy.log` (persisté entre les déploiements via le volume Docker)

### Ajouté (suite)

- **Bascule "Aides à la lecture"** — une nouvelle case à cocher dans la sidebar permet d'afficher ou masquer les ~45 bandeaux d'aide contextuels présents sur chaque page. Le paramètre est persisté dans `ui_prefs.json` (clé `show_hints`) et actif par défaut. `hints_visible()` est le prédicat global utilisé dans tous les modules de page.
  - Plusieurs blocs `st.expander` convertis en `st.popover` pour une expérience plus légère (`match_view_players`, `match_view_encounters`, `career_top_matches_render`, `career_encounters_render`, `teammates_impact`)
  - `restore_hints_from_prefs()` restaure la valeur persistée depuis `ui_prefs.json` au redémarrage

- **Cases KPI carrière refondues** — la ligne de résumé au-dessus des graphes carrière est désormais une rangée compacte de 8 cases :
  - **Matchs joués**, **Durée totale**, **Frags**, **Morts**, **Assists**, **Précision**, **Durée de vie**, **Résultats**
  - Frags, Morts, Assists : valeur principale avec une sous-valeur `/min` inline en petite police
  - Code couleur (vert / or / rouge) par rapport à la moyenne all-time (seuil ±8 %)
  - Case Résultats : barre segmentée avec compteurs bruts V/D/E/DNF et légende colorée
  - `render_top_summary()` supprimé (redondant) ; `_build_kpi_cards()` extrait pour respecter la limite 80L

- **Page Win/Loss fusionnée dans Timeseries** — la page Win/Loss autonome a été absorbée dans la page Timeseries sous forme d'un nouvel onglet. La route `win_loss` est supprimée de `page_router.py`. Onglets Timeseries renommés : Résumé · Cartes & Modes · Progression · Avancé.

### Modifié

- **Image Docker** — `ffmpeg` est désormais installé dans l'image, permettant la génération de miniatures vidéo dans les déploiements conteneurisés sans étape d'installation manuelle supplémentaire.

- **Coéquipiers — panneau légende fixe** — un panneau flottant (en bas à droite, `position: fixed`) affiche la couleur de chaque membre de l'escouade tout au long de la section escouade. Il apparaît à partir de l'en-tête escouade et reste visible au scroll. Les légendes ont été supprimées de tous les graphes individuels (kills/morts, stats/min, métriques, killing sprees, HS+PK, premier événement, kills par arme) car remplacées par ce panneau. Changer de stratégie via `_PANEL_MODE` dans `teammates_legend.py` (`"fixed"` / `"sidebar"` / `"hidden"`).

  - `MetadataResolver.resolve()` accepte désormais un paramètre `lang`
  - Peuplement : `python scripts/populate_asset_translations.py` (supporte `--dry-run`, `--force`, `--types map playlist pair game_variant`)

- **Tooltips descriptions des médailles** — survoler une médaille dans la grille du dernier match ou la section Citations affiche sa description. Repli sur le nom de la médaille si la description est indisponible.

- **Records historiques de l'escouade** (page Coéquipiers) — les meilleurs scores all-time par joueur s'affichent sur la page Escouade avec des annotations rectangulaires colorées par joueur et un classement par carte. Les records sont chargés depuis l'historique complet de chaque joueur (pas seulement les sessions partagées).
  - `compute_squad_records()` — fonction d'analyse pure retournant le meilleur all-time par métrique par joueur
  - Overlays rectangles colorés par joueur + graphes records par carte
  - Intégré dans `render_trio_charts` et `render_metric_bar_charts`

- **Badge Top Killer** 🔫 — nouveau badge sur la timeline Impact pour le premier joueur de l'équipe à atteindre 10 kills. Ajouté à `_EVENT_TO_EMOJI`. Explicitement exclu de la détection des badges *Héros silencieux* et *Faux-frère*. Légende ajoutée dans l'expander sous la grille.

- **Histogramme en papillon — Premier frag / Première mort** (page Coéquipiers) — la distribution premier-frag / première-mort devient un histogramme en miroir avec des tranches de 15 secondes, des séparateurs visuels et des étiquettes de graduation à chaque bordure. Le countdown pré-match est soustrait pour afficher le temps réel de jeu ; `NULL` préservé si aucun kill/mort dans le match.

- **`playable_duration_seconds` + `real_start_time`** dans `match_registry` (migration schéma v6.3) — `playable_duration_seconds` est la durée totale du match moins le countdown ; `real_start_time` est l'horodatage UTC absolu du début du gameplay. Backfill : `python scripts/backfill_data.py --playable-duration`.

- **Fanout PSA coéquipiers** — les Personal Score Awards des membres de l'escouade sont distribués dans toutes les DBs joueur concernées lors du post-sync.

- **Histogramme win rate enrichi** — le tooltip des barres affiche désormais le total de matchs par carte ; la colonne carte utilise le nom traduit (`map_ui`).

- **Page Paramètres V2** — réorganisée en sections fixes (Général, Sync, Performance, Affichage) avec une signature de fonction interne simplifiée. Aucun changement comportemental.

- **Artefacts déploiement VPS Ionos** — configuration `packaging/nginx/`, `deploy.sh`, guide étape par étape (`docs/DEPLOY_GUIDE_ETAPES.md`) et guide VPS Ionos (`docs/DEPLOY_VPS_IONOS.md`).

### Modifié

- **`shared_matches.duckdb` → `shared_matches_v2.duckdb`** — la base de données des matchs partagés a été renommée. Helper `get_shared_matches_path()` introduit ; tous les chemins hardcodés mis à jour ; `compute_sessions.py` mis à jour.

- **Radar escouade** — seuils de l'axe complémentarité recalibrés à p90 (était p80) ; la vue all-time supprimée, le radar est filtré par session uniquement.

- **Seed incrémental LUSR** — bug de cascade corrigé : `seed_ratings` était muté avant d'être lu par `existing_states` en mode incrémental.

- **Sidebar i18n** — `_filters_cascade()` utilise `playlist_name_fr` / `map_name_fr` quand la langue de l'interface est le français.

- **`_try_attach_meta_for_views()`** — vérifie désormais `meta.asset_translations` (présente en v6) au lieu de `meta.maps` (supprimée en v6).

### Corrigé

- `v_match_full` était silencieusement créée sans i18n — `map_name_fr` était toujours `NULL` en production car `_try_attach_meta_for_views()` cherchait `meta.maps` (supprimée en v6) et basculait sur le chemin sans métadonnées.
- `map_name_fr` / `playlist_name_fr` / `pair_name_fr` propagés dans tous les chemins de données Polars via `COLUMNS_COMMON` ; mismatch entre valeurs du filtre sidebar et colonnes du DataFrame corrigé.
- Noms de cartes/modes affichés en français partout : sidebar, graphes bullet + perf, heatmaps, historique de matchs, graphes Escouade et page Records.
- Timeseries : countdown soustrait des timestamps premier-frag et première-mort dans `load_first_event_times()` ; `NULL` préservé après soustraction.
- Apostrophes échappées dans les attributs `title=` des tooltips médailles et citations (HTML brisé corrigé).
- `_build_map_id_index` lit `asset_translations` au lieu de la table `maps` supprimée.
- Miniature carte résolue via `map_id` (asset_id) indépendamment de la langue affichée.
- Radar escouade : l'intersection `shared_match_ids` ne s'effondre plus si un membre n'a pas de matchs.
- Radar escouade : `compute_participation_profile` appelé avec `ProfileOptions` (corrige `TypeError`).
- Page Records : barres hachurées fantômes supprimées ; `offsetgroup` corrigé sur les couches de données.
- Overlay Records : ligne correctement colorée, largeur exacte, repli sur `duration_seconds`.
- `playlist_name_fr` utilise le nom EN comme fallback dans `v_match_full` si la traduction FR est absente.
- Page Performance : graphe score équipe affiche bonus ou base selon le contexte.
- Stats/min escouade visibles pour tous les membres (pas seulement le joueur focal).
- Aliases armes `Mutilator` / `Mutilateur` ajoutés dans `_scoreboard_asset_urls`.
- `top_killer` ajouté à `_EVENT_TO_EMOJI` (entrée manquante causait un repli d'affichage emoji).
- Panneau légende coéquipiers : visibilité conditionnée à la section Escouade uniquement — le panneau démarre en `display:none`, apparaît quand la sentinelle `#llp-squad-start` entre dans le viewport et se masque quand la sentinelle Impact (`#llp-impact-end`) atteint le haut de l'écran.
- Barre native Streamlit masquée — header, toolbar, menu et barre de décoration supprimés via `.streamlit/config.toml` et surcharges CSS.
- Deep links Explorer : `open_match_button` utilise désormais `session_state` au lieu de `query_params` pour éviter le changement de DB lors de la navigation ; `_scroll_into_view` fait défiler la ligne correspondante via un `scrollIntoView` JS same-origin ; le deep link gamertag bénéficie aussi du scroll automatique.
- Ratio KDA/efficacité : valeur lue directement depuis le champ API `mean(ratio)` au lieu d'être recalculée en `(K + A/3) / D`, corrigeant une divergence systématique pour les joueurs avec beaucoup d'assists.
- Watcher media : guard process-level (`_PERIODIC_LOCK` / `_PERIODIC_STARTED`) déplacé avant le branchement Linux/Windows pour éviter la création d'instances `watchdog.Observer` dupliquées lors des reruns Streamlit ; mode actif (inotify vs. polling) logué au démarrage.
- Migrations sync : la DB est marquée comme migrée uniquement après le succès de `ensure_resolution_views()` (guard success-based) ; les `duckdb.connect()` nus dans `_engine_connections.py` et `launcher.py` remplacés par des context managers.
- Healthcheck : statut `'repaired'` traité comme warning (était ignoré silencieusement) ; `recompute_status()` ajouté pour recalculer le résultat global après mutation de checks individuels ; `deploy.sh` mis à jour en conséquence.
- Runner migrations : la DB `metadata` est traitée avant `shared` pour que les vues i18n v6 puissent attacher `metadata.duckdb` à la création ; un `logger.warning()` est désormais émis quand les vues basculent en mode dégradé (colonnes FR = NULL).

### Tests

- `test(ui_persistence)` : 9 nouveaux tests pour `hints_visible()` / `restore_hints_from_prefs()` ; 5 930 tests passent au total
- `test(remediation)` : tests de non-régression P0.2 (suppression code mort `browser_storage`), P0.3 (corrections commentaires localStorage), P1.2 (4 cas supplémentaires)

- `test(i18n)` : couverture `resolve_map_display_names()` + assertions colonnes `map_ui` / `mode_ui`
- `test(radar)` : cas limite `f1_vide` + régression effondrement `shared_match_ids`
- Suite mise à jour pour la signature V2 des Paramètres, la propagation i18n, le refactoring médailles/playlists
- Baseline taille mise à jour (`scripts/size_baseline.txt`)

---

## [6.2.1] - 2026-03-29

### Ajouts

- **Badges "Héros silencieux" & "Faux-frère"** (`src/analysis/impact_analysis.py`) — deux badges d'impact narratifs détectés depuis `match_participants` + `medals_earned` :
  - *Héros silencieux* : top assists sans être top kills dans l'équipe (formule B : assists ≥ seuil × bonus médailles)
  - *Faux-frère* : nombreux kills dans une équipe perdante où les coéquipiers ont sous-performé
  - Variantes single-match et multi-match ; visibles sur la timeline Impact et la page Coéquipiers
  - Légende exposée dans un expander sous la grille de badges
- **Tableau ranking d'impact** — tableau HTML (`os-impact-table`) remplace la heatmap précédente ; affiche le score d'impact classé par joueur par match avec lignes colorées
- **Graphe combiné Tirs à la tête + Frags parfaits** (page coéquipiers) — `plot_headshots_perfect_kills()` affiche un seul histogramme groupé par coéquipier au lieu de deux graphes séparés
- **Top matchs — exclusion BTB** — paramètre `career_top_exclude_btb` dans `AppSettings` ; quand activé, les matchs Big Team Battle sont exclus du classement pour une comparaison Arena/BTB équitable
- **`--btb-only` / `--arena-only`** — nouvelles options dans `scripts/backfill_data.py` pour réparer ciblé les scores d'équipe corrompus CTF/TC/KOTH/Assault
- **`monitor_uptime.sh`** — équivalent shell bash de `monitor_uptime.py` pour les environnements sans Python

### Changements

- **Renommage KDA → Efficiency** — toutes les variables agrégat locales nommées `kda` dans `src/analysis/` renommées en `efficiency` ; API publique inchangée
- **Score impact** — poids *Héros silencieux* relevé à +1.5 ; constantes manquantes (`SILENT_HERO_MEDAL_BOOST`, `FALSE_BROTHER_LOSS_PENALTY`) centralisées
- **Page Impact** — heatmap remplacée par le tableau HTML de ranking ; badges d'impact en premier, section coéquipiers en dernier
- **Page Coéquipiers** — section médailles déplacée en fin de page ; graphes individuels Tirs à la tête et Frags parfaits remplacés par le graphe combiné
- **Normalisation labels modes de jeu (phases 1 + 2)** — `resolve_display_mode()` ajouté comme résolveur unifié ; `translate_pair_name()` lui délègue ; `mode_pair_overrides` étendu avec 29 surcharges FR/EN ; filtre sidebar et tableaux de matchs utilisent les labels normalisés
- **Normalisation score top matchs** — écart de score divisé par le max du match pour équité Arena/BTB ; tri par `performance_score` (pas le `score_diff` brut)
- **`waypoint_player` propagé** dans les liens Explorer des top matchs pour les deep links joueur
- **Docker** — montage `.env.local` désormais obligatoire (suppression de `required: false`)

### Corrections

- **Scores d'équipe corrompus** — `team_score` mis à `NULL` quand contaminé par `ps_score` (CASTLE WARS, Sentry Defense, et tous les modes objectifs) ; `backfill_fix_score_inversions` étendu à Slayer, KOTH et Assault
- **Inversions de score** — inversion Bleu/Rouge corrigée pour Slayer + KOTH/Assault ; seuil comeback rendu proportionnel à la durée du match
- **Flag dominance** — `WHERE dominance_flag IS NULL` corrigé en `WHERE dominance_flag NOT IN (1, 2)` pour ne plus ignorer les lignes avec valeur par défaut `0`
- **Ordre badges comeback** — `CONTRE_REMONTADA` évalué avant `REMONTADA` (chemin mort corrigé)
- **Seuil comeback** — formule symétrique via `_resolve_threshold()` pour éviter une détection asymétrique
- **Navigation deep link** — DB joueur correctement restaurée à la navigation directe via paramètres URL `gamertag + match_id`
- **`run.sh`** — fichier `nul` parasite créé par Git Bash sous Windows supprimé au démarrage
- Badge Faux-frère : emoji 🗡️ appliqué uniformément ; label "assists" corrigé en français

### Tests

- `identify_silent_hero_multi` + `identify_false_brother_multi` : 16 nouveaux tests
- `infer_mode_category` + format inversé sidebar : couvert
- Top matchs : cas E/F/G — filtre BTB, exclusion score NULL, tri priorité badge
- `test(analysis)` : logging + couverture efficiency/cumul (renommage v6.2.1)
- **Total : suite précédente + ~25 nouveaux tests, 0 échecs**

---

## [6.2.0] - 2026-03-28

### Ajouté

- **`src/analysis/comeback_analysis.py`** — module d'analyse pure pour la détection des badges narrative depuis les kill-events highlight (`event_type='kill'`, `time_ms`). Expose `build_score_snapshot()` et `detect_comeback_badge()`. Aucun accès DB.
- **`src/data/comeback_backfill.py`** — couche service qui charge les events/participants depuis `shared_matches.duckdb`, appelle `comeback_analysis`, et écrit le résultat dans `player_match_enrichment.dominance_flag`. Même pattern que `dominance_backfill.py`.
- **`DominanceFlag.REMONTADA = 3`**, **`DominanceFlag.DEBANDADE = 4`**, **`DominanceFlag.CONTRE_REMONTADA = 5`** — nouvelles valeurs dans `src/analysis/_medal_verdicts.py`. Stockées dans la colonne `dominance_flag` TINYINT existante (aucune migration requise). Mutuellement exclusives avec DOMINATION/HUMILIATION.
- **Constantes de seuil** dans `src/analysis/_medal_verdicts.py` : `COMEBACK_DEFICIT_THRESHOLD`, `COMEBACK_COUNTER_GAP`, `COMEBACK_EARLY_CUTOFF`, `COMEBACK_COLLAPSE_CUTOFF`. Centralisées pour faciliter l'ajustement.
- **`SyncScope.comeback_badges`** + **`SyncScope.force_comeback_badges`** — activé par `--all-data`.
- **Arguments CLI `--comeback-badges` / `--force-comeback-badges`** dans `scripts/backfill/cli.py`.
- **Vue escouade unifiée** — `f2_xuid` est désormais optionnel dans `render_trio_view` ; la vue trio gère 2, 3 ou 4 joueurs. `render_single_teammate_view` supprimé.
- **Graphe combiné Frags ↑ / Morts ↓** — `plot_trio_kills_deaths()` remplace les deux graphes séparés par un graphe miroir avec axe Y symétrique.

### Modifié

- `render_trio_view` supporte 1 coéquipier (duo) sans changement de chemin de code — `f2_xuid = None` quand un seul ami est sélectionné.
- `_merge_trio_dataframes`, `render_trio_synergy_radar`, `_render_per_minute_stats`, `_render_trio_performance_charts`, `_render_trio_medals` acceptent tous `f2_name/f2_df = None`.
- `_TRIO_METRIC_SPECS` ne contient plus les entrées kills/deaths (remplacées par le graphe combiné).

### Supprimé

- `render_single_teammate_view()` et fonctions associées dans `teammates_views.py`.
- `render_comparison_charts()` (code mort) dans `teammates_charts.py`.
- Branch de routage `elif len(picked_xuids) == 1` dans `teammates.py`.
- Classes de test `TestRenderSingleTeammateView`, `TestRenderSingleMapSection`, `test_render_comparison_charts_exists`.

### Tests

- **5 178 tests, 0 échec** (4 ignorés)

---

## [6.0.0] - 2026-03-15

> ⚠️ **Extraction d'armes toujours en bêta** — la précision de l'attribution n'est pas garantie dans tous les cas (couverture estimée 70–100 % selon les matchs) ; catalogue d'armes en cours de complétion.

### Ajouté

- **Couche d'abstraction résolution IDs** — trois vues SQL dans `shared_matches.duckdb` remplaçant toutes les cascades 5-sources ad hoc :
  - `v_gamertag_lookup` — FULL OUTER JOIN `xuid_aliases` + `match_participants` avec déduplication et priorité
  - `v_match_full` — `match_registry` enrichie des métadonnées i18n (cartes, playlists, variantes de jeu)
  - `v_killer_victim_full` — paires killer/victim avec gamertags résolus
  - Helper `ensure_metadata_attached(conn)` ajouté dans `src/utils/db.py`

- **Table `weapon_labels`** dans `metadata.duckdb` (`src/data/migration/steps/add_weapon_labels.py`)
  - Schéma : `weapon_labels(weapon_id UBIGINT PK, name_en VARCHAR, name_fr VARCHAR)`
  - Résolution DB-first : `_resolve_weapon_from_db()` avec `@lru_cache` + fallback dicts Python
  - Migration automatique `add_weapon_labels` (`target_db="metadata"`) enregistrée dans le système de migrations
  - `src/ui/i18n/weapons.py` nettoyé : `get_all_weapon_ids`, `get_weapon_ids_by_faction`, `translate_weapon_name` supprimés

- **Package `src/auth/`** — nouvelle couche d'authentification remplaçant toute configuration Azure/env manuelle :
  - `LEVELUP_CLIENT_ID` intégré dans la codebase — aucune configuration Azure requise pour l'utilisateur
  - `_msal.py` : `SerializableTokenCache` persisté en DuckDB (`sync_meta`) via MSAL
  - `provider.py` : point d'entrée unique — cache process (TTL 4 h), refresh MSAL silencieux, `AuthRequiredError`, `start/complete_device_flow`
  - `_halo_exchange.py` : échange stateless `access_token → (spartan, clearance)` via spnkr.auth

- **Launcher — SSO automatique** (`launcher.py`)
  - Gamertag résolu automatiquement depuis le login Microsoft via l'API Halo (plus de saisie manuelle)
  - Nouveau flux premier lancement : Device Code → DuckDB MSAL → sync → Streamlit (zéro configuration manuelle)
  - Menu de récupération simplifié : Device Code Flow uniquement

- **Helper `resolve_medal_name`** (`src/analysis/`) — résolution du nom des médailles depuis `metadata.duckdb`, sans dicts codés en dur

- **Dernier match — navigation précédent/suivant** — boutons `◀ Précédent` / `Suivant ▶` pour naviguer entre les matchs filtrés sans rechargement

- **`populate_metadata_from_discovery.py` réécrit** pour v5.1+
  - Lit `match_registry` dans `shared_matches.duckdb` (remplace `match_stats` dépréciée)
  - DDL étendu avec colonnes i18n (`name_en`, `name_fr`, `mode_name`, `playlist_canonical_*`)
  - Logique extraite dans `scripts/_metadata_db.py` (DDL + enrichissement i18n)

- **Scoreboard — panel détail joueur inline** — un clic sur une ligne dérouler un panel inline (toggle checkbox HTML/CSS pur, sans rerun Streamlit)
  - 4 meilleures armes, 5 meilleures médailles, antagoniste principal résolu via `v_killer_victim_full`
  - Score de performance, citations et contexte de session affichés si la DB locale du joueur est disponible
  - Nouveau fichier `match_view_scoreboard_detail.py` : `load_scoreboard_player_extra_data()` + `render_scoreboard_player_detail_html()`

- **Badges Domination / Humiliation** — enrichissement du résultat basé sur la médaille Steaktacular ("À table")
  - Enum `DominanceFlag` (`NONE / DOMINATION / HUMILIATION`) + `dominance_backfill.py` persistant le flag dans `player_match_enrichment.dominance_flag`
  - Domination : l'équipe du joueur obtient Steaktacular, l'adversaire non ; Humiliation : l'inverse
  - Badge coloré dans l'en-tête du résumé du match (vert = domination, violet = humiliation)
  - Backfill : `python scripts/backfill_data.py --dominance` (ou `--force-dominance`)

- **Carrière — Top meilleurs / pires matchs** — nouvelle section "Top matchs" en bas de la page Carrière
  - Top 10 meilleurs matchs (score de performance le plus élevé) et Top 10 pires, côte à côte
  - Chaque carte : K/D/A, carte, mode/playlist, durée, score, date — avec lien direct vers Match View
  - Badge Domination/Humiliation affiché sur les cartes concernées
  - Implémenté dans `career_top_matches_data.py` + `career_top_matches_render.py`

- **Images commendations — citations d'armes** — illustrations H5G ajoutées pour les 16 commendations d'armes (UNSC, Covenant, Banished)
  - Nouvelle citation : *Maîtrise du MLRS-2 Hydra* avec image dédiée `H5G_citation_Hydra.png`
  - Correction image SPNKr : `H5G_citation_Lance-roquettes.png` → `H5G_citation_SPNKR.png`
  - `WEAPON_FUSION_MAP` : les kills MLRS-2 Hydra (variante alt) comptent désormais vers la citation Hydra de base

- **Médaille custom Vengeur (Avenger)** — détecte les kills de vengeance (tuer l'adversaire responsable de votre mort précédente)
  - ID custom `9 000 000 001` (au-delà de la plage officielle Halo)
  - Backfill global via `killer_victim_pairs` : sous-requête corrélée identifiant pour chaque kill si la victime est l'auteur de la mort précédente du joueur
  - CLI : `python scripts/backfill_data.py --avenger` (ou `--force-avenger` pour recalcul complet)
  - Noms (`medals_fr/en.json`) et descriptions (`medals_descriptions_fr/en.json`) en fichiers JSON statiques
  - `resolve_medal_description()` enrichi d'un fallback JSON quand `metadata.duckdb` ne contient pas de table `medals`
  - 18 tests (12 backfill + 6 description)

- **Étiquette Top Gun** 🔫 — badge sur la timeline d'Impact pour le premier joueur de votre équipe à atteindre 10 kills dans un match
  - Constante `TOP_GUN_KILL_THRESHOLD = 10` ; fonction `_find_top_gun_event()` scanne les `highlight_events` en ordre chronologique
  - Intégré dans le pipeline d'événements d'impact existant (aucune modification des pages UI appelantes)
  - Labels bilingues : « As de la gâchette » (FR) / « Top Gun » (EN)

### Modifié

- **Resolver gamertag** (`src/data/sync/_gamertag_resolver.py`) — cascade 5-sources remplacée par un JOIN unique sur `v_gamertag_lookup` ; `load_match_player_gamertags()` passe de 4 requêtes séquentielles à 1
- **Consommateurs `match_registry`** migrés vers `v_match_full` (asset loader, career encounters, etc.)
- **`killer_victim_repo`** et `career_encounters_data` migrés vers `v_killer_victim_full`
- **Migration i18n DuckDB** — `modes_fr/en.json` migré dans `metadata.duckdb` ; dicts JSON playlists et variantes supprimés du code source
- **`get_tokens_from_env()`** (sync) — wrapper déprécié délégant vers `src.auth` ; appelants mis à jour
- **Parser d'armes — corrélation globale** — taux de correspondance fire_event corrigé de 15 % à 95 % après correction du routage `b2_dispatch` ; logs COMPLETE compacts avec distinction sentinel/no_weapon et taux de rejet `b2_dispatch` exposé
- **Backfill `--weapons --all`** — déduplication des match_ids sur tous les joueurs, chaque film téléchargé une seule fois

### Supprimé

- **Colonne `highlight_events.gamertag`** — migration `drop_highlight_events_gamertag` ; gamertag résolu via `v_gamertag_lookup`
- **Wrapper `resolve_xuid_from_input`** — code mort supprimé
- **`get_outcome_name_fr`** et `_refdata_outcomes`** — remplacés par une résolution via metadata.duckdb
- **14 fonctions Azure/OAuth** dans `launcher.py` (−652 lignes nettes) : wizard Azure, `has_client_id`, options de récupération `config-az`/`paste-id`, variable d'environnement `SPNKR_AZURE_CLIENT_ID` non requise

### Tests

- `tests/test_resolution_views.py` — 11 tests : priorité des vues, fallback alias/participants, filtre NULL, déduplication, colonnes EN toujours peuplées, colonnes FR NULL sans metadata, idempotence, résolution gamertag killer/victim
- `tests/test_global_correlation.py` — 19 tests : **couverture 100 %** sur `_global_correlation.py` (38/38 stmts, 12/12 branches)
- `_parser_logging.py` — **couverture 100 %** (57/57 stmts, 10/10 branches)
- **4 719 tests au total, 0 échec**

---

## [5.7.0] - 2026-03-13

### Ajouté

- **Traductions FR des rangs Halo** (`src/ui/i18n/ranks.py`)
  - 17 rangs de carrière (Recruit→Recrue, General→Général, Hero→Héros…) + 6 paliers CSR (Silver→Argent, Gold→Or…)
  - Helper `translate_rank()` avec fallback sur le nom anglais original
  - Intégré dans le script de migration metadata (`migrate_metadata_to_duckdb.py`)

- **Launchers bilingues FR/EN** (`LevelUp.sh`, `LevelUp.bat`)
  - Détection automatique de la langue système (POSIX `LC_ALL`/`LANG`, Windows Registry `LocaleName`)
  - ~30 messages localisés dans chaque launcher (premier lancement, erreurs, winget, etc.)

### Modifié

- **Pandas→Polars** : suppression de 7 appels `.to_pandas()` dans les modules UI/viz
  - `participation_charts.py` (camembert, barres, empilé) : Polars natif bout en bout
  - `participation_charts_extra.py` (sunburst) : `.to_pandas()` conservé uniquement à la frontière `px.sunburst`
  - `objective_analysis.py` (3 tables assist/awards) : `st.dataframe` Polars natif
  - `duckdb_analytics.py` (tendance KDA) : `st.line_chart` avec `x=`/`y=` Polars natif

- **Miniatures de cartes CSS-only** : remplacement du script JS sandboxé (non fonctionnel) par un système hover CSS pur
  - Suppression de `_MAP_TOOLTIP_SCRIPT` dans `styles.py` (38 lignes JS)
  - Classes `.map-hover` + `.map-popup` dans `static/styles.css`
  - `match_table_html.py` et `win_loss_table_style.py` mis à jour
  - `_build_map_url_index()` amélioré : `lru_cache(maxsize=None)`, normalisation Unicode

### Supprimé

- Guard `was_pandas` dans `_performance_relative.py` : `compute_performance_series()` accepte désormais uniquement `pl.DataFrame`

### Tests

- `TestHighlightEventsSequenceIdempotent` ajouté dans `test_migrations.py` (couverture A.4)
- 45/45 tests migrations passants

---
## [5.6.0-bêta] - 2026-03-10

> ⚠️ **Bêta** — la précision de l’attribution n’est pas encore garantie dans tous les cas (couverture estimée à 70–100 % selon les matchs) ; le catalogue d’armes est en cours de complétion.

### Ajouté

- **Extraction d’armes depuis les films SPNKr** (`src/analysis/weapon_parser.py`, `src/data/services/weapon_extraction_service.py`)
  - Analyse des chunks `REPLICATION_DATA` des films de match pour identifier l’arme utilisée à chaque kill POV (player_index=1, invariant universel)
  - Corrélation kill→dernier fire event dans une fenêtre de 2 000 ms ; kills melee/grenade/véhicule détectés via médailles
  - Table `weapon_kills (match_id, xuid, weapon_id, kills)` dans `shared_matches.duckdb` + migration automatique
  - Architecture hexagonale : domaine pur (`weapon_parser.py`), service d’orchestration, port API étendu

- **Câblage sync** (`src/data/sync/_engine_weapon_kills.py`)
  - Extraction automatique au sync des nouveaux matchs via `WeaponKillsEngineMixin`
  - Contrôlé par `SyncOptions.with_weapons` et `spnkr_refresh_backfill_weapons` dans `app_settings.json` (case à cocher dans Paramètres)

- **Backfill weapon_kills** (`scripts/backfill_data.py --weapons`)
  - `--weapons [--force-weapons] [--gamertag <GT>]` via le CLI backfill unifié
  - Bit `MatchBits.WEAPON_KILLS` (1 << 21) posé sur `match_registry.backfill_completed` après traitement

- **Section armes dans Match View** (`src/ui/pages/match_view_weapon_kills.py`)
  - Onglet Résumé : tableau des kills par arme pour le joueur POV

- **Onglet Armes coéquipiers** (`src/ui/pages/teammates_weapons.py`)
  - Kills par arme pour tous les coéquipiers sur les matchs partagés

- **MSAL Device Code Flow** (`src/utils/msal_device_flow.py`, `src/ui/xbox_oauth_ui.py`)
  - Remplace le flux OAuth redirect : l’utilisateur entre un code court sur xbox.com/activate (sans URI de redirection ni `client_secret`)
  - Wrapper MSAL pur : `initiate_device_flow()`, `acquire_token_blocking()`, `DeviceCodeResult`, `DeviceFlowError`
  - Composant Streamlit : démarrage / polling / réinitialisation (intégré dans le Wizard étape 2 et Paramètres)
  - `setup_wizard_xbox.py` extrait de `setup_wizard.py` pour respecter la limite 500 lignes
  - Option `--device-code` ajoutée dans `scripts/spnkr_get_refresh_token.py` pour acquisition CLI
  - `msal>=1.28.0` ajouté comme dépendance optionnelle
  - Configuration Azure simplifiée : seul `client_id` requis (ni `client_secret`, ni `redirect_uri`)

- **Matrice d’impact** (`src/visualization/friends_impact_heatmap.py`)
  - Séparateurs verticaux (Plotly shapes) entre chaque colonne de match pour améliorer la lisibilité
  - Renommage i18n FR : « Heatmap d’Impact » → « Matrice d’Impact »

- **Documentation** (`docs/FR/CONFIGURATION.md`)
  - Guide Azure simplifié pour le Device Code Flow — étapes `client_secret` et `redirect_uri` supprimées

### Corrigé

- **Discord notifier** (`src/utils/discord_notifier.py`) — Embed allégé restauré quand tous les joueurs sont inactifs (avait été accidentellement supprimé)

### Tests

- **51 tests unitaires** (`tests/test_weapon_parser.py`, `tests/test_weapon_service.py`) : constantes, `find_frame_positions`, `build_frame_estimator`, `correlate_kills_to_weapons`, `WeaponExtractionService` (mocks, dry-run, upsert, erreurs), `WeaponKillsMixin` (upsert/conflit, bit marking)
- **28 tests** ajoutés/réécrits pour le Device Code Flow (`tests/test_msal_device_flow.py`, `tests/test_auth.py`, `tests/test_xbox_oauth.py`) : `authorization_pending`, `slow_down`, pattern sans secret, `get_spartan_tokens`, `resolve_player_identity`
- **4 041 tests, 0 failure**

### Supprimé

- **Flux OAuth redirect Xbox** — `build_xbox_auth_url()`, `generate_oauth_state()`, `exchange_code_for_refresh_token()`, `run_xbox_oauth_callback()`, `_handle_xbox_oauth_callback()` supprimés ; remplacés par le Device Code Flow
- **`client_secret` / `redirect_uri`** — Plus nécessaires pour l’obtention du token ; variables `SPNKR_AZURE_CLIENT_SECRET` et `SPNKR_AZURE_REDIRECT_URI` dépréciées
- **`scripts/backfill/backfill_weapon_kills.py`** — Script standalone supprimé (violait CLAUDE.md : le backfill doit passer par `scripts/backfill_data.py`)

---
## [5.5.0] - 2026-03-06

### Ajouté

- **Page Comparaison de sessions entièrement revue** (`src/ui/pages/session_compare.py` et modules associés)
  - Répartition des résultats : 2 donuts W/L/T/DNF par session avec taux de victoire au centre
  - Highlights : meilleur/pire match par session (ratio F/D, nom du mode)
  - Courbe F/D + précision : K/D renommé F/D, précision sur axe Y secondaire (tirets), durée de vie en hover
  - Modes joués : barres horizontales groupées par session
  - Tableau par carte : victoires/défaites par carte pour chaque session
  - Net score cumulé : coloration des marqueurs par score de performance (vert ≥70 / orange ≥45 / rouge <45) + overlay LUSR ou CSR sur axe secondaire (détecté automatiquement depuis `match_skill_rank`)
  - Profil de participation : radar opaque remplacé par barres horizontales groupées ; seuils mis à l'échelle par nombre de matchs

- **Wizard de configuration — Configuration initiale guidée** (`src/ui/pages/setup_wizard.py` + `setup_wizard_logic.py`)
  - Deux parcours : **Xbox Express** (recommandé, 2 étapes) et **Azure manuel** (avancé, 3 étapes)
  - Cards CSS personnalisées avec icônes, barre de progression animée, étapes numérotées
  - Logique séparée de l'UI (`SetupStatus`, `validate_azure_credentials()`, `validate_gamertag()`, `create_player_profile()`, `save_azure_credentials()`)
  - Guard dans `main()` : le wizard s'affiche automatiquement si credentials ou joueur manquants
  - i18n FR/EN (~49 clés) dans `src/ui/i18n/setup.py`

- **Xbox OAuth — Connexion Xbox en 1 clic** (`src/ui/xbox_oauth.py` + `xbox_oauth_ui.py`)
  - Flux complet : URL Microsoft → callback `?code=XXX&state=YYY` → échange code → refresh_token → spartan/clearance tokens → résolution gamertag+XUID → provisionnement automatique
  - `xbox_oauth.py` (436L) : logique OAuth pure sans dépendance Streamlit
  - `xbox_oauth_ui.py` (163L) : composant Streamlit intégré dans Paramètres (bouton login, statut, déconnexion)
  - Protection CSRF avec `state` aléatoire validé au retour du callback
  - i18n FR/EN dans `src/ui/i18n/pages/xbox.py`

- **Provisionnement automatique** (`src/app/player_provisioning.py`)
  - `provision_player()` : crée `data/players/{gamertag}/stats.duckdb` + table `sync_meta` + enregistre dans `db_profiles.json` — idempotent

- **État d'authentification** (`src/utils/auth.py`)
  - `AuthStatus` dataclass + `get_auth_status()`, `check_credentials()`, `write_env_local()` (écriture/mise à jour de `.env.local` en préservant les commentaires)

- **Compatibilité macOS / Linux** — `LevelUp.sh` (nouveau) : lanceur premier-lancement équivalent à `LevelUp.bat` pour macOS/Linux, écrit en POSIX sh (sans bashism — compatible macOS bash 3.2, dash, zsh). Détecte Python 3.10+ via binaires versionnnés (`python3.12` → Homebrew), chemins Homebrew Intel/Apple Silicon (`/opt/homebrew`, `/usr/local`), puis générique. Messages d'aide ciblés par distribution. `run.sh` corrigé cross-platform. `launcher.py` enrichi : candidats Python versionnnés, chemins Homebrew, `doctor` cross-platform.

- **`launcher.py setup`** — Commande d'installation interactive : détecte Python (py launcher → PATH → emplacements standard → installation via winget), crée le `.venv`, installe les dépendances (`pip install -e ".[spnkr]"`). Supporte `--update` pour mettre à jour un environnement existant.

- **`launcher.py doctor`** — Diagnostic complet de l'environnement : OS, Python, venv, versions des packages critiques vs attendues, nombre de joueurs configurés, présence de `metadata.duckdb`

- **Packaging portable** (`packaging/build_release.py`)
  - Génère un zip autonome `LevelUp-v{version}-win64-portable.zip` contenant Python Embeddable 3.12 (~15 Mo) + le projet complet
  - Premier lancement : installation automatique des dépendances via pip

- **Release GitHub Actions** (`.github/workflows/release.yml`)
  - Déclenché sur push de tag `v*.*.*`
  - Build du zip portable + publication automatique en GitHub Release

- **Mode portable `%APPDATA%`** (`src/utils/paths.py`, `auth.py`, `env.py`)
  - Données stockées dans `%APPDATA%/LevelUp/` (Windows) ou `$XDG_DATA_HOME/levelup/` (Linux) quand pas de `.venv` à la racine
  - Mode développeur : `./data/` si `.venv` existe
  - Override possible via variable d'environnement `LEVELUP_DATA`
  - `.env.local` cherché dans `DATA_DIR` en priorité, puis à la racine du repo

- **Fallback token DB** (`src/ui/profile_api_tokens.py`)
  - Fallback 3 : lecture du refresh_token depuis `sync_meta` de la DB joueur si absent des variables d'environnement

- **Documentation**
  - `docs/CONFIGURATION.md` : réécriture complète avec sommaire, guide Azure pas-à-pas avec 11 captures d'écran annotées
  - `docs/FR/CONFIGURATION.md` : version FR mise à jour
  - `docs/SYNC_GUIDE.md` : réécriture avec architecture sync v5.1, diagramme ASCII
  - `docs/FR/SYNC_GUIDE.md` : mise à jour

- **Migrations de schéma automatiques** (`src/data/migration/`) — runner versionné appliqué automatiquement au démarrage (`launcher.py → _run_migrations()`). Chaque DB (`player`, `shared`, `shared_pve`) trace les migrations dans une table `schema_migrations`. 11 migrations initiales enregistrées. Pour ajouter un changement de schéma : créer une fonction `ensure_xxx` idempotente dans `src/data/sync/migrations.py`, créer le step dans `src/data/migration/steps/` et l'importer dans `steps/__init__.py`.

### Corrigé

- **CSRF** (`streamlit_app.py`) — Correction comparaison `_xbox_state != _xbox_state` (auto-comparaison, toujours False) → `_xbox_state != _expected_state`
- **`_repo_root` indéfini** (`src/ui/profile_api_tokens.py`) — `_repo_root()` jamais importée → remplacée par `REPO_ROOT` depuis `src.utils.paths`
- **DuckDB retry élargi** (`src/data/sync/_engine_connections.py`) — `except duckdb.IOException` → `except duckdb.Error` + délai retry `0.15s → 0.5s`
- **GC mode sync** (`src/ui/_sync_duckdb_ops.py`) — `gc.collect()` + `time.sleep(0.3)` pour libérer les handles fichiers DuckDB sous Windows
- **Guard OAuth consumed** (`streamlit_app.py`) — Flag `_xbox_oauth_consumed` pour éviter le double-traitement du callback au rerun Streamlit
- **Isolation test webhook** (`tests/test_monitor_uptime.py`) — Patch `get_secret` au lieu de manipuler `os.environ` pour éviter le rechargement `.env.local`

### Tests

- **75 tests ajoutés** (1 482 lignes) couvrant l'ensemble des nouveaux modules :
  - `test_auth.py` (13 tests) : `AuthStatus`, `get_auth_status()`, `write_env_local()`
  - `test_setup_wizard_logic.py` (20+ tests) : `SetupStatus`, validations, création de profil
  - `test_xbox_oauth.py` (18 tests) : URL OAuth, échange de code, store/load token, provisionnement
  - `test_xbox_oauth_callback_e2e.py` (9 tests) : flux complet code→player, erreurs, CSRF
  - `test_setup_wizard_page.py` (15 tests) : UI mockée (MockStreamlit), modes Xbox/Azure, progression

### Supprimé

- **`scripts/_archive/`** — 89 fichiers de code mort supprimés (anciens scripts d'analyse d'armes, diagnostics, patchs i18n, utilitaires obsolètes)
- **`requirements.txt`** — Supprimé, remplacé par `pyproject.toml` (source unique de vérité pour les dépendances)
- **`setup.bat`** — Remplacé par `LevelUp.bat` (détection Python améliorée, installation via winget, utilisation de `pip install -e .`)
- **`scripts/install_dependencies.py`** — Workaround MSYS2 SSL, utilisait `requirements.txt`
- **`scripts/setup_env.ps1`**, **`scripts/setup_env.sh`**, **`scripts/activate_env.sh`** — Remplacés par `launcher.py setup`
- **`tests/test_spnkr_refactoring.py`** — Tests pour du code archivé supprimé

### Maintenance

- Rangement racine : `ACKNOWLEDGMENTS.md`, `CHANGELOG.md`, `CONTRIBUTING.md` déplacés vers `docs/`
- Scripts déplacés : `activate_env.sh`, `run_monitor_hidden.vbs` → `scripts/`
- `LevelUp.bat` remplace `setup.bat` comme point d'entrée Windows
- `Dockerfile` et `e2e-browser-optional.yml` mis à jour pour utiliser `pyproject.toml` au lieu de `requirements.txt`
- `run.sh` redirige vers `launcher.py setup` au lieu de `activate_env.sh`

## [5.4.0] - 2026-03-06

### Ajouté

- **Page Explorer — recherche et navigation unifiée dans les matchs** (`src/ui/pages/explorer.py`)
  - Remplace l'ancienne page "Match" avec 6 modules (explorer, explorer_results, explorer_enrich, explorer_data, explorer_logic, match_table_html)
  - Filtres en cascade : date, escouade, type, playlist, mode, carte
  - Recherche floue par gamertag avec suggestions et résolution XUID
  - Tableau HTML OS-style avec KDA, MMR delta, performance, liens deep-link
  - Deep linking : `?page=Explorer&gamertag=XXX` ou `&match_id=XXX`
  - Badges encounter : rival, mentor, proie
  - i18n FR/EN complet + logging structuré + 40 tests unitaires

- **Historique des rencontres — section sous le scoreboard** (`src/ui/pages/match_view_encounters.py`)
  - Tableau HTML affiché sous le scoreboard sur la page Vue Match
  - Par joueur non-ami : fréquence de rencontres, répartition allié/ennemi, win rate allié/ennemi, K/D croisé, date de dernière rencontre
  - Tri : ennemis en premier, puis alliés ; dans chaque groupe par `total_encounters DESC`
  - Badges automatiques : **Dur à cuire**, **Allié+**, **Coriace**

- **Loader SQL dédié** (`src/data/repositories/_encounter_loader.py`)
  - `load_encounter_stats()` — 3 CTEs sur `shared_matches.duckdb`

- **Logique pure testable** (`src/ui/pages/match_view_encounters_logic.py`)
  - `EncounterStats` (Pydantic v2), `Badge`, `build_friends_set()`, `compute_encounter_badges()`
  - 28 tests unitaires dans `tests/test_match_view_encounters.py`

### Refactoring & Architecture

- **Split `transformers.py` (2 095L → package)** — `src/data/sync/transformers/` avec 7 sous-modules (`_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`) + `__init__.py` ré-exportant tout ; aucun breaking change
- **Split `filters_render.py` (1 460L → 4 modules)** — `_filters_period.py`, `_filters_session.py`, `_filters_cascade.py` extraits
- **`_SyncProtocol`** (`src/data/sync/_protocol.py`) — contrat `Protocol` explicite pour les 8 mixins du `DuckDBSyncEngine` ; élimine 70+ `# type: ignore[attr-defined]`
- **`PageContext` + `MatchViewParams`** (`src/app/_page_context.py`) — types réels à la place de 5 champs `Any`
- **`SessionKeys` / `SK`** (`src/app/session_keys.py`) — 20+ clés `st.session_state` centralisées
- **`_sql_fragments.py`** (`src/data/query/_sql_fragments.py`) — source de vérité unique pour `WIN_RATE_EXPR` (dénominateur WIN+LOSS, NULLIF division) ; 7 duplications supprimées dans `analytics.py` et `trends.py`
- **Dettes techniques v4→v5 supprimées** : guard `_PERF_SCORE_AVAILABLE` (always-True), dead method `_ensure_performance_score_column()`, magic number `outcome == 4` → `Outcome.DID_NOT_FINISH`
- **Système de logs centralisé** (`src/utils/log_config.py`) — `setup_app_logging()` : logs fichiers uniquement (`data/logs/app.log` 5 Mo×3, `data/logs/sync.log` 10 Mo×5), aucune sortie console ; `setup_script_logging()` pour les scripts CLI ; `log_duration()` context manager avec seuil ms configurable. Câblé dans : lancement app, chargement joueur, sélection session, changements filtres, chargement DataFrame, KPIs, navigation match (boutons dernier match / carnage / match précédent), sync UI, backfill CLI, tailscale, RAG. `data/logs/` exclu du dépôt.
- **`.gitattributes`** — enforce `eol=lf` sur tout le dépôt ; résout les conflits pre-commit mixed-line-ending sur Windows
- **`pyproject.toml`** — `per-file-ignores` pour `scripts/*` et `launcher.py` (complexité tolérée dans les scripts utilitaires)
- **Enforcement qualité** : `scripts/check_code_size.py` (ratchet 247 violations connues), `tests/test_code_quality.py` (3 tests qualité structurelle), règles CLAUDE.md 13-17

### Corrections de bugs

- **Filtres auto-invalidation post-sync** — `_filters_db_key_{player}` remplace le booléen write-once ; les filtres se réinitialisent quand la DB change (sync, CLI, backfill, changement de profil)
- **Citations calculées post-sync** (`src/data/citations_backfill.py`) — module incrémental appelé après chaque sync ; plus de matchs sans citations
- **SyncLock câblé à l'UI** — `SyncLock(timeout=0)` protège contre les syncs concurrents inter-processus ; flush WAL DuckDB avant `end_sync_mode()`
- **Tailscale guard process-level** — `threading.Event` module-level remplace `st.session_state` ; une seule notification Discord par démarrage de processus
- **Fausse alerte Discord webhook** — skip du check si Doppler est actif ; chargement `.env.local` avant vérification
- **`_PERF_SCORE_AVAILABLE` manquant** (`src/data/sync/_performance.py`) — variable absente après le split en mixins ; guard `try/except ImportError` ajouté ; corrige `NameError` à l'exécution
- **NaN-check fragile** (`match_view.py`) — `x == x` (idiome NaN flottant) → `x is not None`
- **i18n** — 2 clés `PAIR_FR` tronquées restaurées, doublon `tm_session_trend` supprimé, 343 entrées redondantes nettoyées (399 → 56 entrées utiles)
- **Détection backfill per-player** (`detection.py`) — les 6 flags per-player (medals, personal_scores, performance_scores, accuracy, shots, enemy_mmr) vérifient les données réelles du joueur au lieu du bitmask global `backfill_completed` ; corrige un masquage entre joueurs lors du premier sync ; `_player_done_guard()` + 15 tests multi-joueur + 9 tests adaptés

---

## [5.3.0] - Non publié

### Ajouté

- **LUSR (LevelUp Skill Rank) — Système de rating TrueSkill 2 per-groupe** (`src/analysis/`)
  - `skill_rating_config.py` : constantes TrueSkill 2, tiers Bronze→Onyx I-VI, score composite 5 composantes
  - `playlist_groups.py` : 6 groupes Halo Infinite isolés (ranked 1.00, arena 0.80, tactical 0.70, btb 0.60, social 0.40, fun 0.15) avec détection par `pair_name` prefix ou `playlist_name`
  - `skill_rating.py` : algorithme complet — `PlayerState` par groupe, `compute_composite_score()`, `trueskill_update()`, `compute_enemy_strength()`, inactivité par groupe, `compute_skill_ratings_batch()` séquentiel
  - `skill_rating_calibration.py` : module de calibration des poids COMPOSITE_WEIGHTS par comparaison avec `team_mmr` API (grid search aléatoire, métrique MAE ou corrélation Pearson)
  - 68 tests unitaires couvrant l'algorithme, les groupes, l'inactivité, les tiers et la calibration

- **LUSR per-groupe : état TrueSkill indépendant par contexte**
  - `existing_states: dict[str, PlayerState]` remplace `existing_state: PlayerState` — un match ranked n'affecte plus le rating arena
  - `states.setdefault(group, PlayerState())` crée un état au premier match de chaque groupe
  - Inactivité, historique précision et σ decay sont désormais par-groupe

- **Backfill LUSR/CSR** (`scripts/backfill_data.py`, `scripts/backfill/`)
  - `--lusr` / `--force-lusr` : calcul local du LUSR depuis `shared.match_participants` (séquentiel, incrémental)
  - `--csr` / `--force-csr` : récupération CSR depuis l'API Halo pour les matchs ranked
  - `compute_lusr_for_player()` dans `strategies.py` : UPSERT dans `match_skill_rank` avec `rating_delta`, tier et tier_label
  - Table `match_skill_rank` créée automatiquement par `ensure_match_skill_rank_table()` dans `migrations.py`
  - Bits backfill : `lusr = 1 << 16` (65536), `csr = 1 << 17` (131072) dans `BACKFILL_FLAGS`

- **Flags SyncScope** : `lusr`, `force_lusr`, `csr`, `force_csr` dans `src/data/sync/scope.py`

- **Modèle de données CSR** (`src/data/sync/models.py`, `src/data/sync/transformers.py`)
  - `SkillParticipantUpdate` étendu : `pre_match_csr`, `post_match_csr`, `csr_tier`, `csr_sub_tier`
  - Extraction `RankRecap.PreMatchCsr` / `PostMatchCsr` dans `transform_all_skill_stats()`

- **Visualisation LUSR** (`src/visualization/timeseries_combat.py`)
  - `plot_lusr_timeseries()` : zones de tier semi-transparentes, bande de confiance `rating ± deviation`, tendance lissée 20 matchs

- **UI — Page Carrière et Vue Match** (`src/ui/pages/`)
  - `career.py` : cartes visuelles par groupe (image rang 90px centrée, badge LUSR/CSR, delta ▲/▼) + sélecteur de groupe (`st.selectbox`) pour le graphe d'évolution — remplace le tableau en expander et les onglets
  - `match_view.py` : onglet 🏅 Rang avec badge rang, barre de progression colorée, delta vert/rouge

- **Calibration CLI**
  - `python -m src.analysis.skill_rating_calibration --player <GT> [--n-samples 300] [--metric corr]`
  - Grid search sur le simplexe des poids (distribution Dirichlet uniforme, graine reproductible)
  - Affiche les poids optimaux prêts à copier dans `skill_rating_config.py`

- **Notifications Discord post-sync/backfill** (`src/utils/discord_notifier.py`)
  - Nouveau module failsafe — aucune dépendance externe (stdlib `urllib.request` uniquement)
  - Envoi d'un Rich Embed Discord à la fin de chaque `sync.py` et `backfill_data.py`
  - Contenu de l'embed : opération, heure début/fin, durée totale, nombre de joueurs et matchs traités
  - Par joueur : matchs synchronisés (ou traités par backfill), complétude des données (médailles + events), dernier match (carte, mode, FDA, résultat, playlist)
  - Couleur de la barre : vert ✅ (tout OK), jaune ⚠️ (données incomplètes), rouge ❌ (erreur)
  - Flag `--no-discord` sur `sync.py` et `backfill_data.py` pour désactiver ponctuellement
  - `notify_operation_done()` : entrypoint public — `disabled=True` court-circuite immédiatement
  - `fetch_last_match_info(xuid)` : SQL sur `shared_matches.duckdb` (JOIN `match_registry` + `match_participants`)
  - `count_new_matches(xuid, gamertag, since)` : compte les matchs avec `first_sync_at >= since`
  - `count_matches_missing_data(xuid)` : compte les matchs avec `medals_loaded=FALSE OR events_loaded=FALSE`

- **Configuration webhook Discord sécurisée**
  - Toggle `discord_notifications_enabled: false` dans `app_settings.json` (pas de secrets dans ce fichier)
  - URL webhook lue depuis `DISCORD_WEBHOOK_URL` dans `.env.local` (gitignored) via `_load_dotenv_if_present()`
  - Fallback rétrocompatible sur la clé `discord_webhook_url` dans `app_settings.json`
  - Section documentée dans `.env.local.example`

- **Internationalisation FR/EN complète (i18n)** (`src/ui/i18n/`)
  - Package i18n dédié avec modules spécialisés : `common.py`, `pages.py`, `widgets.py`, `viz.py`, `cli.py`
  - Fonctions : `t(key, lang=None)` (UI Streamlit), `viz_t(key, lang)` (Plotly), `discord_t(key, **kwargs)` (Discord), `ct(key, **kwargs)` (CLI/scripts)
  - Langue stockée dans `st.session_state["lang"]` (Streamlit) ou variable d'env `LEVELUP_LANG` (scripts)
  - Sélecteur de langue 🇫🇷/🇬🇧 dans la sidebar (`_render_lang_selector()` dans `src/app/sidebar.py`)
  - Trois champs dans `AppSettings` : `lang`, `discord_lang`, `cli_lang` (défaut `"fr"`)
  - `src/ui/translations.py` bilingue : `translate_playlist_name(name, lang)` et `translate_pair_name(name, lang)` — conserve le regroupement `" on Map"` et les préfixes Halo (Arena, BTB, Ranked)
  - `src/analysis/mode_categories.py` : `normalize_pair_name_to_mode_ui(pair_name, lang)` bilingue
  - `src/utils/discord_notifier.py` entièrement bilingue : `_format_player_field`, `build_embed_payload`, outcomes (🏆/💀/⚖️/🚶), KDA (`{k}K / {d}D / {a}A` vs `{k}F / {d}D / {a}A`), labels opération, footer
  - `src/visualization/distributions_outcomes.py` bilingue : traces Wins/Losses/Ties/Unfinished, buckets temporels (match/hour/day/week/month), heatmap win rate (jours EN/FR), `plot_matches_at_top_by_week` (Others/Top Rate)
  - `src/visualization/antagonist_charts.py` bilingue : `plot_duel_history` traduit Win/Loss/Tie dans l'annotation de duel
  - `src/ui/pages/win_loss.py` : tous les appels viz passent `lang=get_lang()`

### Modifié

- **Algorithme LUSR — mise à jour Elo-style (`K_ELO = 32`)** remplace la zone draw TrueSkill
  - Cause racine de la divergence : `v_draw(t > 0)` donnait des deltas positifs même sur composite=0.5, créant un drift infini quand `state.mu > INITIAL_MU` ou quand le joueur sur-fragait ses `kills_expected`
  - Nouvelle formule mu : `delta_mu = K_ELO × (composite − 0.5) × weight_factor` → ZÉRO exact à composite=0.5, indépendant de `mu_opp`
  - Sigma conserve la réduction TrueSkill évaluée à t=0 (symétrique, `mu_opp` influence `c²` uniquement)
  - Résultat : ratings stabilisés — SpartanA (Diamant V) → Platine IV BTB / Platine VI Arena / Diamant IV Ranked, SpartanB/SpartanC → Or II-IV selon mode
- **Score composite calibré sur 1765 matchs** (SpartanA, SpartanC, SpartanB — Argent → Diamant)
  - Signal cible : `individual_mmr = team_mmr × (kills_expected / ke_avg_match)`
  - Pondération par `nb_matchs × amélioration_MAE` : SpartanA 36.7%, SpartanC 40.0%, SpartanB 13.3%
  - Nouveaux poids : kills_vs_expected=31%, deaths_vs_expected=28%, damage_efficiency=23%, accuracy_delta=13%, win_factor=5%
- **Élimination du biais damage_efficiency** : `PlayerState.damage_eff_history` per-groupe — le composant damage utilise un delta vs historique personnel (comme accuracy_delta) au lieu de la valeur brute
- **Ancrage mu_opp sur `state.mu`** : `compute_enemy_strength` utilise `player_mu=state.mu` comme base d'estimation des adversaires (matchmaking met des joueurs de niveau similaire)
- **Paramètres d'inactivité réduits** : `INACTIVITY_SIGMA_PER_DAY` 3.5→1.0, `MAX_INACTIVITY_DAYS` 30→14 — évite les swings de ±200 pts après une longue pause
- **Seed sigma CSR réduit** : `PlayerState.from_csr()` démarre à `sigma=MIN_SIGMA` (60) au lieu de `INITIAL_SIGMA × 0.6` (210) — le CSR est un ancrage fort, démarrer en état stable évite la volatilité initiale

- **Page Carrière — Comparaison XP & projections multi-joueurs** (`src/ui/pages/career.py`, `career_logic.py`, `career_data.py`)
  - Pour chaque joueur possédant un refresh token, le graphe affiche désormais ses courbes côte à côte avec le joueur courant :
    - Courbe XP réelle (lignes + marqueurs, couleur distincte par joueur)
    - Courbe XP estimée pré-sync (pointillés même couleur) — interpolation linéaire sur les matchs antérieurs au premier sync
    - Projection « à ce rythme » → Héros (tirets) et projection optimiste défis + boost×2 (tirets-points)
    - Toutes les courbes secondaires masquées par défaut — cliquer dans la légende pour les afficher
  - **Niveau de précision variable** selon l’historique disponible :
    - Niveau 1 (fallback) : XP total / jours depuis le premier match connu — actif dès le première sync
    - Niveau 2 (delta réel) : delta XP inter-snapshots / jours actifs — plus précis, s’active automatiquement quand le joueur a joué entre deux syncs
  - Nouvelles clés i18n : `career_xp_other_estimated`, `career_projection_other_hero`, `career_projection_other_optimistic`
  - `_compute_fallback_xp_per_day()` ajoutée dans `career_logic.py`

- **Page Carrière — Courbe XP estimée pré-sync** (`src/ui/pages/career.py`)
  - Trace violette pointillée estimant l'XP pour les ~561 matchs joués avant le premier sync
  - XP moyen/match = `first_xp / n_matchs_pré_sync` — la courbe part de ~0 au match le plus ancien et rejoint le premier snapshot réel
  - Visuellement distincte de la courbe XP réelle (violet `#CE93D8` pointillé)

- **Page Carrière — Courbes de projection vers le rang Héros** (`src/ui/pages/career.py`)
  - **Projection standard** (orange, tirets) : extrapole depuis le rythme actif XP/jour en excluant les gaps d'inactivité > 14 jours
  - **Projection optimiste** (vert, tirets-points) : ajoute les défis hebdomadaires (950 XP/semaine = 4×50 + 3×100 + 3×150) et le défi quotidien (500 XP/jour), le tout avec boost XP ×2 — soit +4 450 XP/semaine en défis
  - Les deux courbes masquées par défaut — cliquer sur la légende pour les afficher
  - Ligne horizontale dorée au seuil Héros (9 319 350 XP)
  - Projection plafonnée à 10 ans pour éviter les graphes infinis
  - Légende déplacée en dessous du graphe (centrée)
  - 23 tests unitaires dans `tests/test_career_xp_projection.py`

### Corrigé

- **20 tests pré-existants corrigés** suite à la migration v5.1 (architecture shared)
  - Groupe A (assertions/fixtures) : `test_backfill_bitmask`, `test_backfill_detection`, `test_xuid_resolution_regression` (×2), `test_post_refactor_perf_contracts`, `test_data_services_contracts`, `test_media_components_sprint4` (×2), `test_media_improvements`, `test_legacy_free_global`
  - Groupe B (mocks v4→v5) : `test_lazy_loading` (×5 — `_get_match_source` v5.1), `test_data_contract_sessions` (réécriture fixture v5 shared + player_match_enrichment)
  - Groupe C (source + mocks) : `test_sessions_integration` (fallback DB production masqué par `__file__` patch), `test_duckdb_repository_schema_contract` (schéma `xuid/gamertag` dans shared fixture), `test_teammates_impact_tab` (×2 — mock `_ensure_shared_attached` + `_load_highlight_events`)

---

## [5.2.0] - 2026-02-20

### Ajouté

- **Filtres v5.2 — Persistance intent-based** (`src/ui/filter_state.py`)
  - `FilterPreferences` : dataclass sauvegardée en JSON par joueur
  - Modes persistés : `playlist_mode`, `mode_mode`, `map_mode` (`"exclude"` / `"include"`)
  - Listes d'exclusions : `excluded_playlists`, `excluded_modes`, `excluded_maps`
  - `_detect_filter_mode()` : heuristique 70/30 — si > 70% des options sont cochées, mode "exclude" ; sinon "include"
  - `reconcile_filter_prefs()` : auto-réconciliation lors de l'apparition de nouvelles options — nouvelles playlists/modes/cartes incluses par défaut sans reset des préférences existantes
  - 45 tests unitaires dans `tests/test_filter_state.py`

- **Filtres v5.2 — Sélecteur "Type d'expérience"** (`src/app/filters_render.py`)
  - Pré-filtre statique : "PVP non classé", "PVP classé", "PVE (Firefight)" activant le filtre `is_firefight`
  - Cascade suppression correcte : modes/cartes calculés depuis `dropdown_base` complet (avant filtre playlist)
  - Intégration des `FilterPreferences` dans le rendu des filtres cascades

- **Stats PvE / Firefight v5.2 — Base dédiée `shared_pve.duckdb`**
  - Nouvelle base `data/warehouse/shared_pve.duckdb` séparée de `shared_matches.duckdb` (évite les colonnes NULL sur 90%+ des matchs PvP)
  - Table `pve_match_stats` : stats par joueur par match Firefight — waves, boss kills, kills par type d'ennemi (Banished : Grunt, Elite, Jackal, Brute, Hunter, Skimmer ; Forerunner : Crawler, Soldier, Knight, Warden)
  - `ensure_pve_schema()` dans `src/data/sync/migrations.py` — création idempotente du schéma
  - `PVE_SCHEMA_DDL` : DDL complet + index `idx_pve_xuid` + `idx_pve_match_id`

- **Stats PvE — Modèles Python** (`src/data/sync/models.py`)
  - `PveMatchStatsRow` : dataclass avec 20 colonnes (waves, boss, ennemi par type, pve_bits)

- **Stats PvE — Transformer** (`src/data/sync/transformers.py`)
  - `extract_pve_stats(match_json)` : extraction pour tous les joueurs d'un match Firefight
  - `_find_pve_stats_dict(player)` : recherche récursive du bloc PvE (EliminationStats / PveStats / FirefightStats / détection par clés)
  - `_extract_enemy_kills_by_type(pve_dict)` : support double structure (champs directs `GruntKills` + sous-dict `EnemyKillsByType`)
  - `_is_firefight_match()` enrichie : 3 critères — `GameVariantCategory` (IDs 41, 42 validés sur JSON API réels), `UgcGameVariant.PublicName`, `Playlist.PublicName` (firefight/baptême/survive)

- **Stats PvE — Pipeline insertion** (`src/data/sync/batch_insert.py`)
  - `batch_insert_pve_stats(conn, rows)` : insertion batch avec `INSERT OR REPLACE`

- **Stats PvE — Bitmask** (`src/data/sync/constants.py`)
  - `PveBits(IntFlag)` : bitmask granulaire pour `pve_match_stats.pve_bits` — TOTAL_KILLS, BOSS_KILLS, GRUNT, ELITE, JACKAL, BRUTE, HUNTER, SKIMMER, SENTINEL, MARINE + combinaisons ALL_ENEMIES, FULL_PVE
  - `MatchBits.PVE_STATS = 1 << 20` : guard global dans `match_registry.backfill_completed` — posé pour tout match traité (Firefight ou non) pour éviter la re-détection infinie

- **Stats PvE — Sync Engine** (`src/data/sync/engine.py`)
  - `_pve_connection` : connexion lazy-init vers `shared_pve.duckdb`
  - `_pve_db_lock` : verrou asyncio dédié
  - `_get_pve_connection()` : lazy init + `ensure_pve_schema` au premier accès
  - `_try_insert_pve_stats(stats_json, match_id, shared_conn)` : extraction + insertion + pose du bit `MatchBits.PVE_STATS` — appelé dans `_process_new_match` et `_process_known_match`

- **Stats PvE — SyncScope** (`src/data/sync/scope.py`)
  - Champs `pve_stats: bool` et `force_pve_stats: bool` dans `SyncScope`
  - Registrés dans `_FORCE_MAP` et `_ALL_DATA_FIELDS`

- **Stats PvE — Détection backfill** (`scripts/backfill/detection.py`)
  - Double guard : `mr.is_firefight = TRUE AND (COALESCE(mr.backfill_completed, 0) & PVE_STATS) = 0`
  - `force_pve_stats` : ignore le guard, retourne tous les matchs Firefight
  - `MatchBits.PVE_STATS` ajouté à `compute_bits_needed_from_scope`

- **Stats PvE — CLI backfill** (`scripts/backfill/cli.py`)
  - Arguments `--pve-stats` et `--force-pve-stats`

- **Stats PvE — Orchestrateur backfill** (`scripts/backfill/orchestrator.py`)
  - `_backfill_pve_for_match()` : ouverture `shared_pve.duckdb`, `ensure_pve_schema`, `batch_insert_pve_stats`, pose du bit guard dans `match_registry`
  - Compteur `pve_stats_inserted` dans `_empty_result()`

- **Citations PvE** (`src/analysis/citations/engine.py`)
  - `load_match_pve_stats(match_id)` : lecture depuis `shared_pve.duckdb`
  - Fusion des stats PvE dans `match_stats` avant calcul des citations
  - `pve_stat` reconnu comme `mapping_type` (traité identiquement à `stat`)

- **81 nouveaux tests** :
  - `tests/test_filter_state.py` : 45 tests — `FilterPreferences`, `_detect_filter_mode()`, `reconcile_filter_prefs()`, save/load
  - `tests/test_pve_transformers.py` : 36 tests — `_is_firefight_match()`, `_extract_enemy_kills_by_type()`, `extract_pve_stats()`, schéma DuckDB, batch insert, `PveMatchStatsRow`, `PveBits`, `SyncScope.pve_stats`

- **Scoreboard "Dernier match"** (`src/ui/pages/match_view_players.py`, `src/data/repositories/_roster_loader.py`)
  - `load_match_scoreboard(match_id)` : requête DuckDB joignant `match_participants` + `xuid_aliases` + sous-requête `medals_earned` (Perfect Kill, ID 1512363953). 20 champs par joueur, trié par `(team_id, rank)`.
  - `render_match_scoreboard()` : tableau HTML par équipe avec 18 colonnes — Gamertag, Rang, Score, Frags, Morts, Assist., FDA, Folie meurtrière, Tirs à la tête, Frags parfaits, Tirs, Tirs au but, Précision, Corps à corps, Armes lourdes, Dégâts infligés, Dégâts subis, Durée de vie moy.
  - Gestion N équipes + joueurs sans `team_id` (NULL → groupe séparé en fin)
  - En-têtes couleur Okabe-Ito : bleu `#0072B2` pour l'équipe du joueur, vermillon `#D55E00` pour les adversaires
  - Ligne du joueur surlignée (cyan `#00e5ff`)
  - Résolution gamertags via `load_match_gamertags_fn` (même pipeline que l'ancien roster)
  - CSS `.os-scoreboard` / `.os-sb-*` avec wrapping colonnes (`max-width: 80px`, `word-break`)
  - Remplace la section "Joueurs" (roster) supprimée

- **Tokens per-player pour endpoints player-gated** (`src/data/sync/api_client.py`, `src/ui/profile_api_tokens.py`)
  - `SPNKR_OAUTH_REFRESH_TOKEN_<GT_NORMALISÉ>` dans `.env.local` pour chaque joueur (ex: `_SPARTANC`, `_MON_GT_2`)
  - Normalisation : `re.sub(r"[^A-Za-z0-9]", "_", gt.strip()).upper()`
  - `get_tokens_for_player(gamertag)` : async, retourne `Tokens | None` — skip + warning si absent (pas de fallback global sur endpoint restreint)
  - `get_player_token_env_key(gamertag)` : retourne la clé env normalisée
  - `profile_api_tokens.get_tokens()` enrichi : param `gamertag` optionnel — priorité token joueur > token global (fallback naturel pour endpoints publics)
  - `profile_api.py`, `get_profile_appearance()` : param `gamertag` propagé jusqu'au fetch SPNKr
  - `load_profile_api()` : dérive le gamertag depuis la DB et le passe à `get_profile_appearance()` — corrige l'adornment/career rank pour les joueurs non-propriétaire du token global

- **Sync Career Rank player-gated** (`src/data/sync/engine.py`)
  - `sync_career_rank()` utilise `get_tokens_for_player()` — skip silencieux + warning si absent
  - Persiste `spartan_id` dans `career_progression` (colonne ajoutée via migration `add_spartan_id_to_career_progression()`)
  - `CareerRankRow.spartan_id` dans `src/data/sync/models.py`

- **Spartan ID dans le hero banner** (`src/ui/styles.py`, `src/app/main_helpers.py`)
  - `get_hero_html()` : nouveau paramètre `spartan_id` — affiché dans la section career-rank sous le label de rang (`.career-rank__spartan-id`)
  - `render_profile_hero()` : charge `spartan_id` depuis `career_progression` (DB, source de vérité) et le passe au hero HTML
  - CSS `.career-rank__spartan-id` : style compact, semi-transparent, lettres espacées

- **32 nouveaux tests** (`tests/test_player_tokens.py`)

### Modifié

- **Accessibilité daltonisme — Migration palette Okabe-Ito** (`src/visualization/`)
  - 7 fichiers de visualisation mis à jour : `antagonist_charts.py`, `performance.py`, `objective_charts.py`, `participation_charts.py`, `team_dominance_timeline.py`, `match_impact_timeline.py`, `friends_impact_heatmap.py`
  - Remplacement des couples rouge/vert néon saturés (incompatibles deuteranopie et protanopie) par la palette **Okabe-Ito** (Wong 2011), référence internationale pour les graphiques accessibles
  - Correspondances principales : vert néon `#00ff00` → vert bleuté `#009E73` · rouge `#ff4444` → vermillon `#D55E00` · magenta `#ff66ff` → rose mauve `#CC79A7` · couleurs équipe `#3DFFB5`/`#FF4D6D` → bleu `#0072B2`/vermillon `#D55E00`
  - Chaque palette documentée avec les anciens hex et la justification dans un bloc de commentaires

- **`_is_firefight_match()`** — Fusion des deux définitions dupliquées en une seule fonction unifiée couvrant les 3 critères (GameVariantCategory + UgcGameVariant.PublicName + Playlist.PublicName)

### Déprécié

- **`display_name_from_xuid()` et `get_xuid_aliases()`** (`src/ui/aliases.py`) — Marquées `.. deprecated::`. Utiliser `load_match_gamertags_fn` pour le contexte match. Conservées pour scripts/migration/export.

### Supprimé

- **Section "Joueurs" (roster)** de la page Dernier match — Remplacée par le scoreboard. `render_roster_section` n'est plus appelée depuis `match_view.py`.

### Corrigé

- **Duplication `_is_firefight_match()`** — Deux définitions coexistaient dans `transformers.py`. La deuxième écrasait silencieusement la première, rendant inopérante la détection via `UgcGameVariant`. Fusion en une seule définition complète.

---

## [5.1.0] - 2026-02-17

### Ajouté

- **Module `src/data/sync/scope.py`** — Dataclass **SyncScope** centralisant les flags
  - Remplace les 30+ kwargs booléens copiés dans 6 fichiers (cli → backfill_data → orchestrator → detection → API)
  - `SyncScope.from_cli_args(args)` : construction depuis argparse
  - `SyncScope.make_all()` : factory pour `--all-data`
  - `resolve()` : implications automatiques (`all_data` → champs, `force_X` → X)
  - Propriétés : `has_any_option()`, `needs_api`, `needs_local_only`, `requested_types`
  - Registres : `_ALL_DATA_FIELDS`, `_FORCE_MAP`, `_REQUESTED_TYPE_MAP`
  - 98 tests unitaires dans `tests/test_sync_scope.py`
  - **Ajouter un nouveau type** : 1 champ dans SyncScope + 1 arg CLI + implémentation métier
- **Module `src/ui/streamlit_modern.py`** — Wrappers compatibilité Streamlit moderne
  - `fragment_if_available` : décorateur graceful-degradation pour `@st.fragment`
  - `PLOTLY_CLEAN_CONFIG` : config Plotly sans barre d'outils
  - `plotly_chart` : wrapper avec config propre par défaut
  - `HAS_FRAGMENT`, `HAS_NAVIGATION` : détection de version
- **Module `src/ui/vectorize_helpers.py`** — Remplacement vectorisé de `map_elements()`
  - `build_mapping()` : pré-calcul dict mapping sur valeurs distinctes
  - `vectorized_apply()` : apply vectorisé via `replace_strict()`
  - `safe_int_format()`, `format_score_pair()` : expressions Polars réutilisables
- **Helpers `get_shared_matches_path()`** — Fonctions centralisées dans `src/utils/paths.py`
  - `get_shared_matches_path()` : chemin absolu vers `shared_matches.duckdb`
  - `get_shared_matches_path_from_player()` : déduction depuis path joueur
- **Script `cleanup_legacy_tables.py`** — Suppression tables obsolètes
  - 9 tables supprimées : `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`, + 4 vues `mv_*`
  - Options : `--dry-run`, `--backup`, `--all`
  - Backups automatiques dans `backups/pre_cleanup/`
- **Vue matérialisée `mv_player_matches`** — Optimisation performance v5.1
  - Pré-calcul jointures match_participants + match_registry + metadata
  - Réduction parsing SQL de 170→10 lignes par requête
  - Gain performance : -70% parsing SQL
- **Cache Repository Streamlit** — `get_cached_repository_st()` avec `@st.cache_resource(ttl=3600)`
  - Connexion DB persistante entre pages UI
  - Gain : 80ms→<20ms connexion
- **Index DuckDB Performance** — 16+ index créés sur 9 tables
  - Index composites `(xuid, match_id)`, `(match_id, xuid)`
  - Index triés sur `start_time`
- **Cache schema metadata** — `_has_column()` et `_has_shared_mp_column()` cachés
  - Évite requêtes `information_schema` répétées
- **Scripts migration bannières LEGACY** — 5 scripts marqués + README.md
  - Bannière claire "HORS SERVICE POST-V5.1"
  - Documentation dans `scripts/migration/README.md`

### Modifié

- **`backfill_data.py` refactoré** — `main()` utilise `SyncScope.from_cli_args()` (−90 lignes)
  - Plus besoin de copier 30+ `args.X` deux fois pour `--all` et `--player`
- **`orchestrator.py` refactoré** — `backfill_player_data`, `backfill_all_players`, `_backfill_with_api` acceptent `scope=SyncScope`
  - Anciens kwargs conservés (marqués `LEGACY`) pour rétro-compatibilité
  - `requested_types` construit via `scope.requested_types` au lieu de 16 `if/append`
- **`detection.py` refactoré** — `find_matches_missing_data` accepte `scope=SyncScope`
  - Anciens kwargs conservés (marqués `LEGACY`) pour rétro-compatibilité
- **Bump Streamlit ≥1.37.0** — Requis pour `@st.fragment` et futures migrations `st.navigation`
- **Plotly `config={"displayModeBar": False}`** — Appliqué sur 69 `st.plotly_chart` (15 fichiers)
  - Suppression barre d'outils Plotly pour une UI plus propre
- **`@fragment_if_available`** — Décorateur appliqué sur 5 pages multi-charts
  - timeseries, session_compare, win_loss, objective_analysis, career
  - Réduit le re-render au fragment seul lors d'interactions filtre
- **`match_history.py` modernisé** — Remplacement HTML custom par `st.dataframe` + `column_config`
  - Suppression dead code : `_format_score_label`, `_fmt`, `_fmt_mmr_int`
  - Virtualisation native Streamlit pour tableaux larges
- **`st.navigation` lazy loading** — 11 page closures dans `streamlit_app.py`
  - `build_navigation()` + `render_page_selector_nav()` dans `page_router.py`
  - Fallback legacy `dispatch_page()` pour Streamlit < 1.36
  - Seules les pages visitées sont importées → -60% mémoire initiale
- **Centralisation `duckdb_read_only()`** — Context manager dans `src/utils/db.py`
  - 7 fichiers migrés (career, cache_loaders, cache_filters, media_library, multiplayer, data_loader)
  - `duckdb.connect` directs : 14 → 4 (restants : sync engine, écriture légitime)
- **Réduction `st.rerun()`** — 32 → 14 dans `src/`
  - `checkbox_filter.py` : 16 reruns → 0 via callbacks `on_click`/`on_change`
  - Trio button filters : `on_click=_apply_trio_filter`
- **Sécurisation `unsafe_allow_html`** — html.escape() sur données dynamiques
  - `kpi.py` et `performance.py` : XSS protection
  - `sidebar.py` brand : HTML → `st.header()` + `st.divider()`
- **Tests non-régression modernisation** — 30 tests dans `test_8ter_modernisation.py`
  - Couverture : staticPlot, fragments, st.navigation, duckdb_read_only, st.rerun, html.escape
- **Éradication complète `map_elements()`** — 28 occurrences remplacées dans 15 fichiers
  - Remplacement par `build_mapping()` + `replace_strict()` ou expressions Polars natives
  - Fichiers : filters.py, filters_render.py, win_loss.py, last_match.py, stats.py,
    match_view_charts.py, media_library.py, teammates_helpers.py, session_compare.py,
    session_compare_charts.py, duckdb_analytics.py, match_view.py, citations.py,
    teammates_service.py, media_indexer.py
- **Migration `xuid_aliases` → `shared_matches.duckdb`** — Source unique centralisée
  - 9 fichiers migrés pour lire depuis `shared.xuid_aliases` (13 955 rows)
  - Suppression fallbacks locaux `stats.duckdb`
  - Fichiers : `aliases.py`, `xuid.py`, `multiplayer.py`, `cache_loaders.py`, `engine.py`, `_roster_loader.py`, `sessions_backfill.py`, `sync.py`, `resolve_missing_gamertags.py`
- **`_get_match_source()`** retourne maintenant un 3-tuple `(source_sql, params, uses_mv)`
  - Permet skip jointures redondantes en mode v5.1
- **8+ fonctions cache_loaders** migrées vers `get_cached_repository_st()`
  - Suppression connexions neuves redondantes
- **Jointures metadata/MMR** skippées en mode v5.1 quand `uses_mv=True`
  - RC3/RC4 : -3 LEFT JOIN sur chemin critique

### Corrigé

- **Onglet Citations affichait 159 citations au lieu de 45** — Filtrage par `citation_mappings.enabled` réactivé
  - Le JSON `halo5_commendations_fr.json` contient 159 citations (armes, Spartan Companies, etc.)
  - Le filtrage avait été supprimé, affichant toutes les citations y compris celles sans mapping
  - Correction : les items JSON sont maintenant filtrés par les noms normalisés des citations activées via `CitationEngine.load_mappings()`
  - Fichier : `src/ui/commendations.py`

### Supprimé

- **Tables legacy player DBs** — 9 tables par joueur, données centralisées
  - `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`
  - Vues obsolètes : `mv_match_stats_with_context`, `mv_recent_matches`, `mv_team_stats`, `mv_opponent_stats`
  - 38 528 rows libérées sur 4 joueurs
- **Références SQLite runtime** — 0 `import sqlite3` dans `src/`
- **Références `metadata.db`** — Tout migré vers `metadata.duckdb`
- **Méthode dépréciée `attach_sqlite`** — Supprimée de duckdb_engine.py

### Performance

| Métrique | v5.0 | v5.1 | Gain |
|----------|------|------|------|
| Connexion DB | 80ms | <20ms | **-75%** |
| load_matches(100) | 200ms | <80ms | **-60%** |
| Première page UI | 1500ms | <800ms | **-47%** |
| Parsing SQL/requête | 170 lignes | 10 lignes | **-94%** |

---

## [5.0.0] - 2026-02-15

### Ajouté

- **Architecture shared_matches.duckdb** — Base de données partagée centralisant les matchs de tous les joueurs
  - 6 tables : `match_registry`, `match_participants`, `highlight_events`, `medals_earned`, `xuid_aliases`, séquence `highlight_events_id_seq`
  - 14 index optimisés (match_id, xuid, start_time, composites)
  - Schéma DDL complet : `scripts/migration/schema_v5.sql`
  - Documentation : `docs/SHARED_MATCHES_SCHEMA.md`
- **Migration v4 → v5** — Scripts de migration incrémentale par joueur
  - `scripts/migration/create_shared_matches_db.py` : création de la DB partagée
  - `scripts/migration/migrate_player_to_shared.py` : migration par joueur
  - Résultat : 1289 matchs migrés, 285 partagés (22.1%), 0 orphelins
- **Détection matchs partagés dans Sync Engine** — Sync allégée pour matchs déjà connus
  - `_process_known_match()` : enrichissement personnel uniquement (économie 1-2 appels API/match)
  - `_process_new_match()` : sync complète vers shared (registry + participants + events + medals)
  - `extract_all_medals()` : extraction des médailles de TOUS les joueurs du match
  - `extract_match_registry_data()` : extraction données communes du match
- **ATTACH multi-DB dans DuckDBRepository** — Lecture transparente depuis `shared_matches.duckdb`
  - `shared_db_path` auto-détecté ou configurable
  - Queries natives `shared.match_participants`, `shared.match_registry`, `shared.medals_earned`
  - Propagation dans la factory repository
- **Sous-requête `_get_match_source()`** — Abstraction permettant à toutes les pages UI de lire depuis shared sans modification
- **Optimisations API Sync v5**
  - Parallélisation appels API skill + events (`asyncio.gather`)
  - Batching des insertions DB (commit tous les 10 matchs)
  - Performance scores calculés en batch post-sync
  - Rate limit optimisé (10 req/s, parallel_matches=5)
- **Citations DuckDB-first** — Nouveau système de citations stockées par match
  - `CitationEngine` : moteur de calcul et agrégation SQL
  - Table `citation_mappings` dans `metadata.duckdb` : 14 règles (8 existantes + 6 réintégrées)
  - Table `match_citations` dans chaque `stats.duckdb` joueur
  - Backfill CLI : `--citations` / `--force-citations` dans `scripts/backfill_data.py`
  - 6 citations objectives réintégrées : Défenseur du drapeau, Je te tiens !, Sus au porteur du drapeau, Partie prenante, À la charge, Annexion forcée
  - Colonne `enabled` dans `citation_mappings` pour désactivation sans suppression
  - Support V5 (shared_matches) dans `CitationEngine` avec fallback V4
  - Documentation : `docs/CITATIONS.md`
- **Framework de test MockStreamlit** — Fixture `MockStreamlit` dans `conftest.py` pour tester les pages UI en mode headless
- **+946 tests** ajoutés (S1→S7ter) — total 2768 passed, 0 failed, 38 skipped
- **Script de nettoyage post-migration** — `scripts/cleanup_player_dbs_v5.py`
  - Supprime les tables redondantes des player DBs après migration v5 (match_stats, match_participants, highlight_events, medals_earned)
  - Mode --dry-run pour simulation sans modification
  - Backup optionnel avant nettoyage
  - Validation automatique de l'existence de shared_matches.duckdb
  - VACUUM automatique pour récupération d'espace disque (-85% de taille en moyenne)
  - Documentation : `docs/CLEANUP_V5.md`
- **Documentation** : `docs/SHARED_MATCHES_SCHEMA.md`, `docs/SYNC_OPTIMIZATIONS_V5.md`, `docs/TESTING_V5.md`, `docs/ARCHITECTURE_V5.md`, `docs/MIGRATION_V4_TO_V5.md`, `docs/CLEANUP_V5.md`

### Modifié

- **`DuckDBSyncEngine`** refactoré pour écrire dans `shared_matches.duckdb` (matchs, participants, events, médailles)
- **`DuckDBRepository`** refactoré avec ATTACH `shared_matches.duckdb` en read-only
  - `load_match_participants()` → lecture depuis `shared.match_participants`
  - `load_highlight_events()` → lecture depuis `shared.highlight_events`
  - `load_medals_for_match()` → lecture depuis `shared.medals_earned`
  - `load_matches()` → JOIN `shared.match_participants` + `shared.match_registry` + `player_match_enrichment`
- **Toutes les pages UI** utilisent `_get_match_source()` au lieu de `match_stats` directement
- **`render_h5g_commendations_section()`** utilise `CitationEngine` (agrégation SQL, ~90% plus rapide)
- **`render_citations_page()`** simplifié — ne pré-agrège plus les médailles/stats pour les citations
- **Filtrage des citations** piloté par `citation_mappings.enabled` (plus besoin du JSON d'exclusion)
- **Version `pyproject.toml`** bumpée de 3.0.0 à 5.0.0
- **Statut projet** : Development Status 4-Beta → 5-Production/Stable

### Supprimé

- **VIEWs de compatibilité v4** supprimées (`scripts/migration/remove_compat_views.py`)
- **Données dupliquées** dans les player DBs : `match_participants`, `highlight_events`, `medals_earned` centralisés dans shared
- **Shim `src/db/migrations.py`** — déprécié, supprimé en faveur de `src.data.sync.migrations`
- `CUSTOM_CITATION_RULES` dict (ancien `commendations.py`)
- `_compute_custom_citation_value()` (itérations lentes, remplacé par SQL)
- `load_h5g_commendations_tracking_rules()` (remplacé par `citation_mappings` DuckDB)
- Constantes `DEFAULT_H5G_TRACKING_ASSUMED_PATH` / `DEFAULT_H5G_TRACKING_UNMATCHED_PATH`
- Dépendance aux fichiers JSON de tracking commendations
- Logique d'exclusion JSON dans `render_h5g_commendations_section()`

### Corrigé

- **Tests flaky Windows** : `tmp_dir` → `tmp_path` pour éviter DuckDB `WinError 32` (file locking)
- **Tests lazy_loading** : mode v4 forcé pour compatibilité

### Performance

| Métrique | v4 | v5 | Gain |
|----------|----|----|------|
| Stockage (4 joueurs) | 800 MB | 250 MB | **-69%** |
| DB size par joueur | 200 MB | 30 MB | **-85%** |
| Appels API (sync 4 joueurs) | 12 000 | 3 300 | **-72%** |
| Temps sync (100 matchs) | 45 min | 12 min | **-73%** |
| Temps/match (partagé) | 16s | 0.5s | **-97%** |
| Temps/match (nouveau) | 16s | 2-3s | **-81%** |
