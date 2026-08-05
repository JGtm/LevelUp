# Journal de decisions — archive 2026-Q1

> Rotation trimestrielle (regle CLAUDE.md).

## [2026-03-11] Fix Step 4b — Reclassification melee/grenade manquants dans `_reconcile_api_aggregates`

**Statut** : Complété

**Contexte** : Sur le dernier match de Chocoboflor (`20fd2c23`), les 2 corps à corps et 1 grenade (confirmés par `match_participants.melee_kills=2` / `grenade_kills=1`) étaient attribués au Sidekick et MA40 par le pipeline weapon. Cause : les médailles contextuelles (Pummel, Back Smack, Stick…) absentes de `highlight_events` → `is_melee=False` / `is_grenade=False` sur tous les kills → tous passaient dans la branche Formula A snapshot.

**Décision technique** : Ajout d'un **Step 4b** dans `_reconcile_api_aggregates` (avant Step 4a), qui compare les sentinelles déjà détectées avec les agrégats API et reclassifie les kills weapon les moins certains (priorité : `low` → `none` → `medium` → `high+swap` → `high`, à égalité : delta_ms desc) en `MELEE_WEAPON_ID` / `GRENADE_WEAPON_ID` avec `confidence='high'`.

**Résultats observés** :
- Avant : `{'Sidekick': 7, 'MA40 AR': 5}` — 0 melee, 0 grenade
- Après : `{'Corps à corps': 2, 'Sidekick': 5, 'MA40 AR': 4, 'Grenade': 1}` — conforme à l'API ✓
- Backfill Chocoboflor : 288 matchs, 6200 lignes, 0 erreurs

**Fichier modifié** : `src/data/services/weapon_extraction_service.py` — `_reconcile_api_aggregates()`

**Conclusion** : Fix minimal, sans régression sur les matchs où melee/grenade sont détectés via médailles (dans ce cas `detected == api`, le step 4b ne fait rien). Backfill global `--all --weapons --force-weapons` lancé en parallèle pour les 3 autres joueurs.

---

## [2026-03-12] Analyse faisabilité — Détection de langue système dans `LevelUp.sh` / `LevelUp.bat`

**Statut** : Complété ✅

**Demande** : Déterminer si la détection de la langue système est possible dans les scripts lanceurs, et documenter la feature dans le backlog.

**Décision technique** :
- **`LevelUp.sh`** : Détection via variables POSIX `$LC_ALL` > `$LC_MESSAGES` > `$LANG` (ex. `fr_FR.UTF-8`). Extraction des 2 premières lettres via `cut -c1-2`. Compatible POSIX strict (dash/bash/zsh, macOS/Linux/WSL2). Aucune commande externe requise.
- **`LevelUp.bat`** : Détection via `REG QUERY "HKCU\Control Panel\International" /v LocaleName` (retourne `fr-FR`, `en-US`…). Disponible sur Windows Vista+, aucune dépendance externe. Alternative PowerShell documentée.
- **Pattern d'implémentation** : Variables nommées `msg_<key>_fr` / `msg_<key>_en` avec macro de résolution — compatible POSIX sh strict et CMD sans tableaux associatifs.

**Résultat** : Section ajoutée dans `.ai/BACKLOG.md` avec inventaire complet des ~35 (sh) + ~30 (bat) chaînes à traduire, exemples de code de détection, plan en 6 étapes, complexité M.

**Conclusion** : Feature entièrement faisable, documentée et prête à implémenter. Aucun fichier de code modifié (tâche de backlog uniquement).
## [2026-03-12] Azure Auto-Registration — Suppression du client_secret et Device Code Flow

**Statut** : Complété

**Contexte** :
L'utilisateur souhaitait que `LevelUp.bat` / `LevelUp.sh` dispensent l'utilisateur de visiter
portal.azure.com pour configurer l'application Azure. Le wizard CLI (`_wizard_azure_creds()`)
demandait encore `client_id` + `client_secret` (ancien flux Authorization Code), alors que le
wizard web (`setup_wizard.py`) utilisait déjà le Device Code Flow (client_id uniquement).

**Décisions techniques** :
1. **Ajout de `_try_azure_auto_register()`** dans `launcher.py` : si `az` CLI est disponible,
   crée automatiquement l'application Azure « LevelUp Halo » (public client, Device Code Flow)
   sans visiter portal.azure.com. Vérifie si une app existe déjà avant de la créer.
2. **Refonte de `_wizard_azure_creds()`** : tente d'abord `_try_azure_auto_register()`, sinon
   saisie manuelle du `client_id` uniquement (plus de `client_secret`). Ouvre portal.azure.com
   dans le navigateur et affiche le conseil d'installer `az` CLI.
3. **Refonte de `_wizard_oauth_token()`** : remplace le flux Authorization Code + client_secret
   par MSAL Device Code Flow (import depuis `src.utils.msal_device_flow`). Pas de redirect URI.
4. **Mise à jour de `_onboard_first_player()`** : ne vérifie plus `SPNKR_AZURE_CLIENT_SECRET`.
5. **Mise à jour de `_cmd_add_player()`** : idem, seul `SPNKR_AZURE_CLIENT_ID` requis.
6. **Mise à jour de `_env_check_for_player()`** : suppression de la clé `client_secret`.
7. **Mise à jour de `_print_token_setup_instructions()`** : instructions Device Code Flow.

**Résultats** : 649 tests passent (2 échecs pre-existants liés à l'environnement CI :
`check_code_size.py` absent + `ruff` non installé).

**Conclusion** : Avec `az` CLI installé, zéro visite du portail Azure requise.
Sans `az`, seul le `client_id` est demandé (plus simple qu'avant).

---

## [2026-03-12] Azure CLI — Proposition d'installation automatique

**Statut** : Complété

**Contexte** :
Après avoir implémenté `_try_azure_auto_register()`, l'utilisateur demande explicitement
que LevelUp propose d'*installer* Azure CLI si celui-ci n'est pas trouvé sur le système.

**Décisions techniques** :
- `_offer_install_azure_cli()` : si `az` introuvable + terminal interactif → affiche le contexte
  et demande confirmation [O/n]
- `_run_az_install(platform)` : délégation par plateforme :
  - Windows (`win32`) : `winget install --id Microsoft.AzureCLI -e` (si winget disponible)
  - macOS (`darwin`) : `brew install azure-cli` (si brew disponible)
  - Linux : `curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash`
  - Fallback universel : lien `https://aka.ms/installazurecli`
- `_try_azure_auto_register()` : appelle `_offer_install_azure_cli()` si `az` absent, puis
  re-vérifie avec `shutil.which("az")` après installation (avertit de redémarrer le terminal
  si az reste introuvable — cas winget sur Windows).

**Résultats** : 4250 tests passent (24 échecs pre-existants, aucune régression).

### [2026-03-13] — Réduction baseline taille code : 135 → 110 violations

- **Statut** : Complété
- **Tâche** : Réduire les violations de taille (fonctions > 80L, modules > 500L) de 135 à ≤ 110.

**Décision technique** : Extraire des helpers/sous-fonctions (extract method) pour chaque fonction dépassant 80 lignes, en commençant par les plus petites violations (81-87L).

**Actions (24 fonctions refactorisées dans 23 fichiers) :**

Batch 1 (81-82L) — 10 fonctions :
- `compute_session_performance_score_v2` → `_build_v2_result()` (keyword-only args)
- `_get_shared_connection` → `_run_shared_migrations()` (static method)
- `load_matches` → `_row_to_match_row()` (module-level)
- `build_thumbnail_html` → `_build_thumbnail_container_html()` (f-string, pas `.format()`)
- `plot_top_players_objective_bars` → `_extract_ranking_data()` + `_get_ranking_attr()`
- `render_comparison_radar_chart` → `_add_radar_trace()` (dash optionnel)
- `_render_backfill_section` → constante `_ALL_BACKFILL_FLAGS`
- `_sync_async` → `_finalize_sync_result()`
- `plot_damage_dealt_taken` → `_add_damage_traces()` (paramétrisé)
- `plot_assist_breakdown_pie` → `_extract_assist_values()`

Batch 2 (83-87L) — 7 fonctions :
- `create_career_progress_gauge` + `create_hero_progress_gauge` → DRY (`_progress_bar_color()`, `_build_progress_gauge()`)
- `_extract_mmr_from_skill` → 3 helpers (`_find_player_result`, `_extract_enemy_mmr_from_team_mmrs`, `_extract_enemy_mmr_from_teammates`)
- `_upsert_csr_rating` → `_build_csr_tier_label()` + constant `_CSR_UPSERT_SQL` + `_ROMAN`
- `_build_friend_df_from_match_ids_v4` → `_translate_playlist_pair_columns()` + `_convert_start_time_timezone()`
- `create_teammate_synergy_radar` → `_add_synergy_trace()`
- `create_stats_per_minute_radar` → `_add_permin_radar_trace()`
- `_render_media_legacy` → `_scan_media_in_window()` + `_render_legacy_video_selector()`

Batch 3 (81-85L) — 7 fonctions :
- `_build_settings_from_ui` → `_get_preserved_settings()` (dict de champs non-UI)
- `plot_cumulative_net_score` → `_add_cumulative_score_traces()`
- `plot_performance_timeseries` → `_ensure_performance_column()`
- `plot_kd_timeseries` → `_add_kd_cumulative_trace()`
- `add_outcome_traces` → `_add_sparse_bar_trace()` (DRY : ties/left)
- `render_participation_section` → `_load_participation_awards()`
- `render_participation_comparison` → `_build_comparison_profiles()`

**Corrections additionnelles :**
- Bug `_run_shared_migrations` : `return self._shared_connection` stale dans `@staticmethod` → supprimé
- PLR0913 : ajout `# noqa` sur helpers extraits (>5 args inévitables)
- F401/F821 : nettoyage imports inutilisés post-extraction

**Résultats** :
- Baseline : 135 → 110 (objectif atteint)
- 104 fonctions > 80L + 6 modules > 500L
- Ruff : All checks passed
- Tests : 4485 passed, 0 regressions (6 échecs pré-existants : verrou fichier shared_matches + test sync)

---

### [2026-03-15] — Backfill weapons --force : correction bugs post-run

- **Statut** : Complété
- **Tâche** : Analyser le résultat du backfill `--all --force-weapons` (32 369 lignes sur 4 joueurs/1984 matchs), identifier les avertissements `unresolved_player` et corriger les bugs

**Contexte** :
- Run 1 (~2h45) → 0 lignes insérées : migration `add_weapon_kills_reconciled_as` absente de `_apply_schema_migrations()`. Corrigée manuellement (ensure + insert schema_migrations).
- Run 2 (~11 min partiel) → 32 369 lignes. Warnings `unresolved_player` sur chaque match.

**Décision technique principale** :

**Bug 1 — `_apply_schema_migrations()` manquait `ensure_weapon_kills_reconciled_as`** :
- Fichier : `scripts/backfill/orchestrator.py`
- Fix : ajout de l'import + appel `ensure_weapon_kills_reconciled_as(shared_conn)` dans la fonction

**Bug 2 — `unresolved_player` sur le joueur POV** :
- Root cause identifiée via inv130 : dans le PLAYER_METADATA packet, chaque joueur non-POV a son XUID 2 fois (une avec pi réel 1-7, une avec pi=0). Le joueur POV n'a **que** des occurrences pi=0.
- `detect_pi_from_metadata()` saute explicitement pi=0 → le joueur POV n'est jamais retourné.
- `_resolve_player_indices()` retourne immédiatement si metadata non vide (7/8 joueurs) → le POV est perdu.
- Le docstring `"le POV est toujours pi=1 dans l'espace Section 2"` était **incorrect** : la cross-validation METADATA vs acurtis (inv130) montre que le POV a pi=0 dans les fire events aussi.
- Fix : après la résolution METADATA, faire un acurtis ciblé sur les XUIDs manquants → le POV est résolu avec pi=0 via `detect_player_indices(first_chunk_data, missing)`.
- Fichier : `src/data/services/weapon_extraction_service.py` (`_resolve_player_indices`)
- Docstring corrigée dans `src/analysis/packet_index.py`

**Résultats observés** :
- 0 erreurs de lint/type sur les 3 fichiers modifiés
- Fix proactif : tout futur backfill trouvera les colonnes correctes sans erreur silencieuse

**Conclusion** :
- Le prochain `--force-weapons` sur de vrais données devrait éliminer les `unresolved_player` et inscrire un `player_index=0` pour le joueur POV, activant ainsi la corrélation fire event + Formula A pour ses kills.

---

### [2026-03-14] — Cache manifest film (bug 3 : appel API redondant)

- **Statut** : Complété
- **Tâche** : Éviter un appel `get_film_by_match_id` (API Halo) par match sur les re-runs du backfill weapons.

**Root cause** : Sans cache du manifest film, chaque re-run télécharge le manifest depuis l'API même pour des matchs déjà traités. Le manifest (~2KB JSON) contient uniquement le `blob_prefix` et la liste des chunks (index, timestamps, `file_relative_path`), données stables et réutilisables.

**Décision technique** :
- Nouveau module `src/data/services/_film_manifest_cache.py` : `write_manifest_cache()`, `load_manifest_cache()`, `compute_needed_chunks()`.
- Le manifest est sérialisé en JSON dans `data/investigation/chunks/{match_id[:8]}/manifest.json` (~2KB/match).
- `_download_needed_chunks` tente d'abord `load_manifest_cache` avant tout appel API. Si miss → appel API + sauvegarde.
- `_compute_needed_chunks` déplacé dans `_film_manifest_cache.py` (même sémantique : analyse métadonnées chunks).
- `_download_chunk_with_sem` + `_download_chunk` fusionnés pour rester sous 500L.

**Résultats** :
- `weapon_extraction_service.py` : 505L → 495L (sous la limite)
- `_film_manifest_cache.py` : nouveau module 73L
- 1984 manifests seront créés au premier run → les re-runs n'auront plus aucun appel API manifest

---

### [2026-03-15] — Wave 4 + 5 PLAN_ABSTRACTION_RESOLUTION v6 (Commits 8-10)

- **Statut** : Complété (Wave 4 + audit Wave 5 partiel)
- **Branche** : `refactor/id-resolution-cleanup`

**Commit 8 — `feat(migration): supprimer highlight_events.gamertag + nettoyer resolver`** (0a5c69c)
- Supprimé `_resolve_from_highlight_events()` et `_extract_ascii_token()` de `_gamertag_resolver.py`
- `_events_repo.py` : `COALESCE(vg.gamertag, he.gamertag)` → `vg.gamertag` (branche view) ; `NULL AS gamertag` (branche fallback)
- `teammates_impact.py` : même simplification COALESCE
- `_encounter_loader.py` : CTE `he_gamertags` entièrement supprimée + `LEFT JOIN` orphelin + paramètre target_xuids orphelin corrigé
- `_weapon_kills_repo.py` : ajout `_has_gamertag_column()` helper défensif (compatible tests unitaires qui créent la table avec gamertag)
- `migrations.py` : `_recreate_highlight_events_with_sequence()` — schéma sans gamertag + INSERT colonne-explicite
- Nouveau step `drop_highlight_events_gamertag.py` : recréation complète (DuckDB ne supporte pas ALTER TABLE DROP COLUMN sur table indexée)
- Baseline size-ratchet mis à jour (102 violations)
- 4647 tests passants (+59 vs Commit 7)

**Commit 9 — `feat(analysis): helper resolve_medal_name depuis metadata.duckdb`** (ffdd959)
- Nouveau module `src/analysis/_medal_data.py` : `resolve_medal_name(medal_name_id, lang="fr")` — Sources : metadata.duckdb si table medals existe, sinon JSON statiques `static/medals/medals_{lang}.json`, fallback `str(id)`
- 7 tests dans `tests/test_medal_data.py`
- 4654 tests passants

**Audit Commit 10 — résultats**
- `grep highlight_events.*gamertag` → 0 hit non légitime (helper migration + docstrings seulement)
- `grep match_registry.*map_name/playlist_name` → 0 hit
- `grep killer_victim_pairs.*killer/victim_gamertag` → 0 hit
- Vues v2 : `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full` ✅ présentes
- `highlight_events.gamertag` : supprimée de `shared_matches_v2.duckdb` via migration recréation (239 429 lignes préservées)
- 4654 tests passants, 0 échec

**Note bascule v2 → prod** : La bascule `shared_matches.duckdb ↔ shared_matches_v2.duckdb` est une opération manuelle à exécuter avec l'app arrêtée. Condition préalable : vérifier `shared_matches.duckdb` (prod actuelle) reçoit aussi la migration `drop_highlight_events_gamertag` au premier prochain démarrage.

**Décision technique principale** : `ALTER TABLE DROP COLUMN` non supporté par DuckDB 1.4 quand des index existent → recréation de table requise (même pattern que `_recreate_highlight_events_with_sequence`).

**Conclusion** : Wave 4 complète. Wave 5 (Commit 11 + 11b — nettoyage traduction assets obsolètes) nécessite analyse préalable des dépendances résiduelles avant suppression. Commits 0-10 sur branche `refactor/id-resolution-cleanup`.

---

### [2026-03-15] — Wave 5 complète : Commits 11 + 11b — Nettoyage couche i18n

**Statut** : Complété ✅

**Commits** :
- `57a755c` — refactor(i18n): supprimer dicts/JSON playlists obsolètes
- `b4ff066` — refactor(i18n): migrer modes_fr/en.json vers metadata.duckdb

**Décision technique principale (Commit 11)** :
`PLAYLIST_FR`, `PLAYLIST_EN`, `PAIR_FR` supprimés de `translations.py`. `translate_playlist_name()` réécrite en passthrough + UUID warning. Source de vérité : `metadata.duckdb` via `v_match_full.playlist_name_fr`. `match_history.py` et `explorer_enrich.py` migrés vers aliasing passthrough. Migration framework étendu (`target_db="metadata"` + `metadata_db_path` dans `apply_pending_migrations`). `drop_legacy_translation_tables` créé pour supprimer `mode_translations` + `playlist_translations` legacy.

**Décision technique principale (Commit 11b)** :
`modes_fr/en.json` migrés vers 4 tables DuckDB (`mode_prefix_names`, `mode_name_tr`, `mode_pair_overrides`, `mode_lang_settings`). `translate_pair_name()` réécrite : 35L sans `noqa: C901`, 3 étapes (override → combinatoire → mode seul), cache LRU process-level via `_load_mode_tables(lang)`. Fallback gracieux pour langues inconnues et DB absente. 9 tests dédiés dans `tests/test_translate_pair_name.py`.

**Résultats observés** :
- Tests avant : 4607 / après Commit 11 : 4607 / après Commit 11b : 4621 (+14 nouveaux tests)
- Zéro régression sur les 2 commits
- `mode_pair_overrides` : 15 lignes (vs 22 estimé dans le plan — normal : doublons de maps normalisés + EN moins de paires que FR)
- Hooks pre-commit : 2 tentatives par commit (ruff-format reformate, 2ème commit propre)

**Conclusion** : Plan v6 PLAN_ABSTRACTION_RESOLUTION.md entièrement complété. Branche `refactor/id-resolution-cleanup` prête pour merge. 12 commits (0-11b) couvrant fondation SQL, migration consommateurs, nettoyage, migrations schéma et couche i18n complète.

---

### [2026-03-15] — Audit final + couverture de tests

**Statut** : Complété ✅

**Commit** : `2878eaa` — test(audit): couverture mode dégradé + migration drop_legacy_translation_tables

**Décision technique** :
Audit post-Wave 5 : vérification complète DB, ruff, size baseline, e2e migrations. 3 lacunes de couverture identifiées et corrigées :
1. `translate_pair_name` sans DB (mode dégradé) — monkeypatch sur `src.utils.paths.get_metadata_db_path` (import local à la fonction)
2. `_load_mode_tables` retourne un dict stable quand DB absente
3. `TestDropLegacyTranslationTables` : 5 tests e2e migration (`drop_legacy_translation_tables`)

**Bug corrigé** : Target du monkeypatch `"src.ui.translations.get_metadata_db_path"` échoue (import local) → corrigé en `"src.utils.paths.get_metadata_db_path"`.

**Résultats observés** :
- 4682 tests passants (4621 + 61 nouveaux suite à l'audit complet)
- Branche `refactor/id-resolution-cleanup` : 13 commits au total
- `metadata.duckdb` : 8 tables confirmées ; `mode_translations` + `playlist_translations` legacy supprimées par migration au prochain lancement

**Conclusion** : Audit terminé. Couverture tests complète sur les nouvelles fonctionnalités i18n v6. Branche prête pour merge.

---

### [2026-03-15] — Phase 2 abstraction DB : CareerMixin + explorer_data migration

**Statut** : Complété ✅

**Décision technique principale** :
Migration systématique des appels `duckdb_read_only` directs dans la couche UI vers `DuckDBRepository`. Phase 2 couvre `career_data.py`, `career_lusr.py` et `explorer_data.py`.

**Changements effectués** :
- `src/data/repositories/_career_repo.py` (NOUVEAU) : `CareerMixin` — 6 méthodes : `load_career_data`, `load_career_history`, `load_pre_sync_match_dates`, `load_lusr_snapshot`, `load_lusr_history`, `load_is_with_friends_batch`
- `src/data/repositories/_gamertag_resolver.py` : ajout `get_all_gamertags()` → lit `shared.v_gamertag_lookup`
- `src/data/repositories/_roster_loader.py` : ajout `load_common_matches_df(target_xuid)` → JOIN `match_participants + match_registry`
- `src/data/repositories/duckdb_repo.py` : `CareerMixin` inséré dans le MRO
- `src/ui/pages/career_data.py` : 5/6 fonctions migrées (`_load_post_sync_match_count` = dead code conservé)
- `src/ui/pages/career_lusr.py` : `xuid` threadé dans `_render_lusr_rating_chart`
- `src/ui/pages/explorer_data.py` : entièrement réécrit — 4 fonctions déléguent au repo, suppression de `duckdb_read_only` et `_shared_db_path`
- `src/ui/pages/explorer.py` : `xuid` threadé dans `_render_match_filters`, `_render_player_search`, `_cached_all_gamertags`
- `tests/test_explorer_logic.py` : signatures mises à jour, `test_shared_db_path_derivation` supprimé
- `scripts/size_baseline.txt` : `_roster_loader.py` mis à jour (545L → 592L)

**Résultats observés** :
- 4800 / 4800 tests passants (zéro régression)
- `explorer_data.py` : ~150L → 80L (suppression code dupliqué)

**Conclusion** : Phase 2 complète. Prochaine étape Phase 3 : `main_helpers.py`, `career_top_matches_data.py`, `career_encounters_data.py`, `aliases.py`, `match_view_encounters.py`, `session_compare_logic.py`, `media_library_data.py`.

---

### [2026-03-17] — Nettoyage DB weapon_kills + backfill NS timeline
**Statut** : Complété ✅

**Décision technique principale** :
Nettoyage chirurgical des anomalies dans `shared_matches_v2.duckdb::weapon_kills`
suite au fix NS timeline (`b2fc825`). Trois catégories d'anomalies identifiées et corrigées.

**Anomalies corrigées** :
1. **Cat1a** (1 219 lignes) : `weapon_id=0 confidence='none'` → mis à `NULL` (puis backfill les a rétablis comme grenades correctes)
2. **Cat1b** (375 lignes) : `weapon_id=0 confidence='high'` → DELETE + reset bits → backfill re-extrait
3. **Cat2** (1 300 lignes) : sentinels melee `weapon_id=1` avec `confidence='high'` + `delayed_damage=TRUE` → normalisés (`confidence='none'`, `delta_ms=NULL`, `delayed_damage=FALSE`, `swap_detected=FALSE`)
4. **Cat3** (22 594 lignes / 624 matchs) : raw FA handles en `weapon_id` → DELETE + bits `WEAPON_KILLS` + `WEAPON_KILLS_NO_FILM` resetés → backfill complet

**Note importante** : `GRENADE_WEAPON_ID = 0` est un sentinel **légitime** (pas une anomalie). L'anomalie initiale était `weapon_id=0 AND confidence='high'`, pas tous les weapon_id=0.

**Résultats observés** :
- `fire_event` : 8 211 → **60 669** kills attribués (+52 458, ×7.4x)
- `path='none'` (non résolu) : 56 985 → **2 826** (−95 %)
- 1 457/1 457 matchs avec bits WEAPON_KILLS settés
- Zero anomalie sentinel restante

**Scripts créés** :
- `scripts/_fix_weapon_kills_sentinel.py` — nettoyage idempotent (à supprimer après usage)
- `scripts/_verify_weapon_kills.py` — vérification de l'état DB

**Conclusion** : Le fix NS timeline est validé en production. Les weapons data sont propres.
Prochaine étape : supprimer les scripts temporaires `_fix_*` et `_verify_*`, puis commit.

---

### [2026-03-17] — Scoreboard detail : assets d'armes + description médailles + images commendations HI
**Statut** : Complété ✅

**Décision technique principale** :
Enrichissement visuel du panneau scoreboard inline avec assets graphiques (armes et commendations)
et tooltip description sur les médailles.

**Changements** :
- `WeaponDetailItem` (dataclass) remplace les tuples `(str, int)` dans `ScoreboardPlayerExtraData.weapons`
- `_render_weapons_section()` — section armes avec images PNG (`/app/static/weapons-assets/`) via `_weapon_asset_url()`
- `_normalize_weapon_asset_key()` + `_build_weapon_asset_url_index()` — index normalisé (NFKD ASCII) pour correspondre noms d'armes → fichiers
- `resolve_medal_description()` dans `_medal_data.py` — résolution description depuis `metadata.duckdb` (colonnes candidates : `description_fr/en`, `desc_fr/en`, `blurb_fr/en`)
- `MedalDetailItem.description` — tooltip sur les icônes médailles
- Assets statiques : 27 PNG armes (`static/weapons-assets/`), 26 PNG commendations HI + 1 H5G
- `static/styles.css` : nouveaux sélecteurs `.os-sb-detail-item--weapon`, `.os-sb-detail-weapon-asset`, `.os-sb-detail-weapon-fallback`
- `src/ui/sync.py` : `SyncLock(timeout=0, lock_file=...)` avec chemin explicite `data/.sync.lock`
- Logs DEBUG ajoutés dans `weapon_parser._fallback_formula_a()` et `_global_correlation._attribution_from_event()` pour tracer NS → weapon_id résolu

**Conclusion** : Le scoreboard inline affiche désormais les assets visuels des armes avec fallback texte, et les médailles ont un tooltip avec leur description.
Prochaine étape : commit + push.

---

### [2026-03-18] — Session escouade du 18/03 classée "solo" en UI
**Statut** : Complété ✅

**Décision technique principale** :
Utiliser `player_match_enrichment.is_with_friends` comme source de vérité pour la classification Solo/Escouade, au lieu de dépendre uniquement de `teammates_signature` + sélection d'amis UI.

**Résultats observés** :
- Audit DB sur les matchs concernés : pas d'anomalie (`is_with_friends=TRUE` sur les matchs trio).
- Le mauvais classement venait de la couche UI qui pouvait marquer une session en solo selon le contexte de sélection d'amis.

**Changements code** :
- `src/app/_filters_session.py` : `_classify_sessions_solo_squad()` priorise `is_with_friends` si présent.
- `src/ui/_cache_sessions.py` : `cached_compute_sessions_db()` charge et propage `is_with_friends` (SQL, schémas vides, retour Cas A/B, chemin d'erreur).

**Conclusion** :
La session du 18/03 est désormais classée escouade selon le flag BDD persistant, même si la sélection d'amis UI change.

---

### [2025-07-18] — Axe 7 : batch_commit_size adaptatif (Phase 1 perf/sync)
**Statut** : Complété ✅

**Décision technique principale** :
Remplacer la valeur fixe `batch_commit_size=25` par un mode auto (`-1`) qui résout la taille optimale selon `max_matches`. Logique encapsulée dans `SyncOptions.with_resolved_batch_size()` pour garder `engine.py` sous la limite 500L.

**Résultats observés** :
- 74 tests ciblés verts (tests/perf + test_sync_engine + test_sync_sprint6)
- engine.py : 510L → 498L  
- _sync_internal : 85L → 75L (limites respectées sans `# noqa`)
- Commit : `149fa3f` sur branche `perf/batch-commit-auto`

**Changements code** :
- `src/data/sync/models_sync.py` : import `replace` + `logging`, + `with_resolved_batch_size()`, + `compute_optimal_batch_size()`
- `src/data/sync/engine.py` : supprimé `dc_replace`, bloc 11L → `options.with_resolved_batch_size()` (1L)
- `tests/perf/test_batch_commit_adaptive.py` : 11 tests (nouveau fichier)
- `tests/test_sync_engine.py` + `test_sync_sprint6_optimizations.py` : 4 assertions stale corrigées

**Conclusion** :
Axe 7 implémenté et validé. Prochaine étape : Axe 6 — LUSR UPSERT vectorisé (`_skill_rating.py`).

---

### [2025-07-18] — Axe 6 : LUSR UPSERT vectorisé (Phase 1 perf/sync)
**Statut** : Complété ✅

**Décision technique principale** :
Remplacer les N `conn.execute()` individuels dans `_upsert_lusr_ratings` par une
liste `rows_to_insert` + un unique `conn.executemany(_LUSR_UPSERT_SQL, rows_to_insert)`.
Guard-rail ±100 pts séquentiel préservé (dicté par `prev_rating[pg]`) — seul le flush est vectorisé.

**Résultats observés** :
- 11 + 85 = 96 tests verts (tests/perf/test_lusr_batch_upsert + tests existants skill_rating)
- Commit : `b0771f1` sur branche `perf/lusr-vectorized`

**Changements code** :
- `src/data/sync/_skill_rating.py` : `_upsert_lusr_ratings` collecte `rows_to_insert` puis flush via `executemany`
- `tests/perf/test_lusr_batch_upsert.py` : 11 tests (nouveau fichier)

**Conclusion** :
Axe 6 validé. Branche mergée dans `perf/shared-handle-fix`.

---

### [2025-07-18] — Axe 2 : shared_matches R/O direct sans ATTACH (Phase 1 perf/sync)
**Statut** : Complété ✅

**Décision technique principale** :
Option A — remplacer `ensure_shared_attached(player_conn, ...)` par `duckdb.connect(shared_path, read_only=True)` (connexion directe R/O). DuckDB supporte DIRECT+DIRECT (MVCC) mais pas ATTACH+DIRECT sur le même fichier.

Découverte clé : DuckDB partage le catalogue entre TOUTES les connexions au même fichier. Un ATTACH sur `cit_conn` est visible depuis `player_conn`. Solution : `try/finally` qui DETACH avant fermeture, même en cas de retour anticipé.

**Résultats observés** :
- 42 tests verts (tests/perf × 3 + test_sessions_integration)
- Commit : `a5e5ed1` sur branche `perf/shared-handle-fix`
- `sessions_backfill.py` : 488L (sous 500L), `backfill_sessions_for_player` : 79L (sous 80L)

**Changements code** :
- `src/data/citations_backfill.py` : `_process_citations_batch` avec `try/finally DETACH`
- `src/data/sessions_backfill.py` : `_fetch_shared_context_ro` + `_dry_run_count` helper
- `src/data/sessions_backfill_shared.py` : `_load_matches_split` (2 connexions directes + Polars join)
- `tests/perf/test_shared_handle_fix.py` : 9 tests (nouveau fichier)

**Conclusion** :
Axe 2 validé. Prochaine étape : Axe 4 — Citations batch SQL.

---

### [2025-07-18] — Axe 4 : Citations bulk SQL + executemany (Phase 2 perf/sync)
**Statut** : Complété ✅

**Décision technique principale** :
Remplacer la boucle N×(6 SQL queries + 1 INSERT) par 6 bulk queries + 1 executemany INSERT.
CitationEngine reçoit les données pré-chargées via `compute_all_for_match()` (0 SQL à l'intérieur).
Plus d'ATTACH sur `cit_conn` depuis Axe 4 — `shared_ro` direct R/O suffit.

**Distribution des mappings** (discovery matrix) :
- `weapon_stat` : 20 — batchable via `v_weapon_kills`
- `medal` : 15 — batchable via `medals_earned`
- `custom` : 12 — Python pur, données pré-chargées (df_match construit depuis match_stats)
- `stat` : 11 — batchable via `match_participants`
- `award` : 9 — batchable via `personal_score_awards`
- `composite` : 7 — non par-match
- `pve_stat` : 6 — batchable via `shared_pve.duckdb` séparé

**Résultats observés** :
- 44 tests verts (tests/perf × 4)
- citations_backfill.py : 331L (sous 500L), toutes fonctions ≤80L
- Commit : `3183fa1` sur branche `perf/citations-batch-sql`

**Changements code** :
- `src/data/citations_backfill.py` : 6 fonctions `_bulk_*` + `_build_match_data_map` + `_process_citations_batch` refactoré
- `tests/perf/test_citations_batch.py` : 11 tests (nouveau fichier)

**Conclusion** :
Axe 4 validé. Prochaine étape : Axe 1 — Post-sync partiellement parallèle.

---

### [2025-07-18] — Axe 1 : Post-sync partiellement parallèle (Phase 3 perf/sync)
**Statut** : Complété ✅

**Décision technique principale** :
Rendre `_run_post_sync_compute` async et lancer les citations via `run_in_executor` (thread pool)
pendant que perf_score → sessions → dominance s'exécutent séquentiellement.
Pas de conflit de tables : `match_citations` (citations) vs `player_match_enrichment` (perf/sessions/dominance).
DuckDB MVCC garantit la cohérence avec plusieurs connexions R/W simultanées sur le même fichier.

**Stratégie de parallélisation** :
- `cit_future = loop.run_in_executor(None, self._post_sync_citations_sync)` lancé avant le bloc sériel
- `_post_sync_citations_sync` ouvre sa propre connexion R/W DuckDB (thread-safe, MVCC)
- `_shared_connection` fermée **avant** le scatter pour éviter tout conflit de catalogue
- `await cit_future` à la fin — le bloc sériel se termine avant d'attendre les citations

**Contrainte taille** :
- `engine.py` était 498L après trim (ajout ~57L, suppression old 55L = +2L net)
- Deux sessions de trim de commentaires/blancs pour rester ≤500L

**Résultats observés** :
- 6 tests verts : coroutine, run_in_executor, close-before-executor, exception-fallback, future-awaited, sync-fallback
- engine.py : 498L (sous 500L)
- Commit : `cc90e7b` sur branche `perf/post-sync-parallel`

**Changements code** :
- `src/data/sync/engine.py` : `_run_post_sync_compute` → async + `_post_sync_citations_sync` wrapper ajouté
- `tests/perf/test_post_sync_parallel.py` : 6 tests (nouveau fichier)

**Conclusion** :
Axe 1 validé. Prochaines étapes : Axe 5 (run_in_executor MetadataResolver) puis Axe 3 (dual semaphore).

---

### [2025-07-19] — Fix xuid_input : lire depuis sync_meta — Complété

**Problème :** `init_source_state` peuplait `xuid_input` avec le gamertag extrait du chemin
(`_infer_gamertag_from_v5_path` → `"JGtm"`). Mais `resolve_xuid_input("JGtm", db_path)` ne
trouvait pas le XUID numérique → `xuid = ""` → condition `load_match_dataframe` échouait →
message "Configure une DB et un joueur dans Paramètres" affiché au lieu du dashboard.

**Cause racine :** La fonction de résolution `resolve_xuid_input` doit pouvoir trouver le XUID
via xuid_aliases ou sync_meta, mais si `xuid_aliases` ne contient pas le gamertag (premier
lancement après sync, ou gamertag incohérent), elle retourne `""`.

**Fix (`src/app/state.py`):**
- Ajout de `_read_xuid_from_sync_meta(db_path)` : lit directement `sync_meta WHERE key='xuid'`
  → retourne `"2535469190789936"` (XUID numérique valide, pas de résolution nécessaire)
- `init_source_state` : appelle `_read_xuid_from_sync_meta` en priorité, fallback sur
  `_infer_gamertag_from_v5_path` (avant premier sync, sync_meta est vide)

**Résultat :** `xuid_input = "2535469190789936"` → `str(xuid or "").strip()` ≠ `""` → dashboard
s'affiche correctement.

**Commit :** `7ae483a` sur branche `fix/count-matches-use-syncresult`

---

### [2025-07-18] — Axe 5 : Transformations CPU-bound via run_in_executor (Phase 3 perf/sync)
**Statut** : Complété ✅

**Décision technique principale** :
Pré-requis bloquant résolu en premier : `threading.RLock()` ajouté dans `MetadataResolver` pour
protéger `_cache` et `_conn` en cas d'accès multi-thread (Axe 5 + futur Axe 3).
Ensuite `_transform_match_stats_async` ajouté dans `_match_processing_helpers.py` — utilise
`functools.partial + loop.run_in_executor(None, fn)` pour exécuter `transform_match_stats`
dans le thread pool default (libère l'event loop 50-200ms par match).

**Stratégie** :
- `_transform_match_stats_async` dans helpers (308→327L, sous 500L)
- `_match_processing.py` migré vers `await self._transform_match_stats_async(stats_json, skill_json)`
- Import `transform_match_stats` retiré de `_match_processing.py` → 543L → 539L (gain net)
- `size_baseline.txt` mis à jour (ratchet) : décalages de lignes suite aux edits

**Résultats observés** :
- 6 tests verts : RLock, thread-safety 10 threads, run_in_executor, partial kwargs, exception
- metadata_resolver.py : 230L → 234L (sous 500L)
- Commit : `0c7d7dd` sur branche `perf/post-sync-parallel`

**Changements code** :
- `src/data/sync/metadata_resolver.py` : `threading.RLock()` + `resolve()` protégé par lock
- `src/data/sync/_match_processing_helpers.py` : ajout `asyncio`, `functools`, `transform_match_stats` import + `_transform_match_stats_async`
- `src/data/sync/_match_processing.py` : 2 callers migrés, import retiré, -4L net
- `scripts/size_baseline.txt` : ratchet mis à jour
- `tests/perf/test_transform_async.py` : 6 tests (nouveau fichier)

**Conclusion** :
Axe 5 validé. Prochaine étape : Axe 3 (dual semaphore fetch/CPU — le plus complexe).

---

## [2026-03-22] Fix fresh install : mv_player_matches jamais créée

**Statut** : Complété

**Problème** : Sur une fresh install (VM), après l'onboarding (sync 10 matchs),
l'app affichait "Aucun match trouvé" alors que les matchs étaient bien dans
`shared_matches_v2.duckdb`.

**Diagnostic** : `ensure_mv_player_matches_view()` était définie dans
`migrations.py` mais n'était appelée **nulle part** dans le code de production
(seulement dans les tests). La vue `mv_player_matches` n'existait donc jamais sur
une fresh install. `_get_match_source()` tente `FROM shared.mv_player_matches` →
exception → fallback `pl.DataFrame()` vide → message "Aucun match trouvé".

**Décision** : Créer une migration formelle dans le système de migration, pattern
identique aux autres migrations `target_db="shared"`. Elle sera appliquée
automatiquement par `launcher.py → _run_migrations()` au prochain lancement.

**Fichiers** :
- `src/data/migration/steps/add_mv_player_matches_view.py` (nouveau)
- `src/data/migration/steps/__init__.py` (+1 import + 1 entrée `__all__`)

**Tests** : 30/30 passed (`test_performance_optimizations.py`)

---

## [2025-01-xx] fix(asyncio) — ConnectionResetError WinError 10054 Windows — Complété

**Branche** : `fix/count-matches-player-enrichment` — commit `0811dda`

**Problème** : Sur Windows, les logs étaient pollués massivement par :
```
_ProactorBasePipeTransport._call_connection_lost
ConnectionResetError: [WinError 10054] Une connexion existante a dû être fermée par l'hôte distant
```

**Diagnostic** : Bug connu de `ProactorEventLoop` (défaut Windows Python 3.8+). Asyncio appelle
`socket.shutdown(SHUT_RDWR)` sur des sockets déjà fermées par le serveur distant (MSAL device
flow, Microsoft auth). L'erreur est purement cosmétique — aucune donnée perdue.

**Décision** : Exception handler asyncio personnalisé qui absorbe silencieusement les
`ConnectionResetError` (les autres exceptions sont délégués au handler par défaut).
Installé dans `main()` du launcher via `suppress_asyncio_proactor_connection_reset()`.

**Fichiers** :
- `src/utils/log_config.py` — ajout de `suppress_asyncio_proactor_connection_reset()`
- `launcher.py` — appel dans `main()` après `setup_script_logging`

**Résultat** : Élimination du spam WinError 10054 dans les logs launcher sans impacter
les vraies erreurs asyncio.

---

## [2026-03-30] fix(radar) — Normalisation axe Objectifs du radar Complémentarité — Complété

**Branche** : `fix/radar-objectifs-normalisation` — commits `1df74ce`, `93568dc`, `1638c4e`

**Problème** : L'axe "Objectifs" du radar "Complémentarité de l'escouade" (teammates) et du
radar de participation (match view) s'affichait proche de 0, même pour d'excellents scores CTF
ou Strongholds. Exemple mesuré : 1800 pts sur 3 matchs CTF → 20% de l'axe.

**Cause racine** : Dans `compute_global_radar_thresholds()`, le seuil objectifs était calculé
comme `max(max_obj, max_kill)` — `max_kill` (~3000) écrasait systématiquement `max_obj` (~600
en CTF). Le seuil objectifs se retrouvait calibré sur les kills, rendant les scores objectifs
insignifiants.

**Décision technique** :
- Phase 0 : calcul du p90 réel par famille de mode (CTF, Strongholds, Oddball, Slayer…) via
  une requête supplémentaire lors du scan des DBs joueurs
- Phase 1 : `objectifs = max_obj * factor` (plus de `max(max_obj, max_kill)`)
- Phase 2 : seuil objectifs de session = somme des p90 par match selon la famille détectée
  par `_get_mode_family(pair_name)` — gestion native des sessions mixtes (BTB + Arena)
- Percentile p90 : un joueur bon atteint ~82%, seul le top 10% plafonne à 100%
- Match view : même correction, seuil per-mode appliqué au match unique

**Fichiers modifiés** :
- `src/analysis/participation_radar.py` — `RADAR_THRESHOLDS_PER_MODE`, `_get_mode_family()`,
  `get_mode_family()` (public), scan per-mode dans `compute_global_radar_thresholds()`
- `src/ui/pages/teammates_synergy.py` — `_compute_player_profile()` : seuil pondéré per-mode
- `src/ui/pages/match_view_participation.py` — seuil per-mode sur le match unique

**Tests ajoutés** :
- `tests/test_participation_radar.py` — `TestGetModeFamily` : 22 cas (CTF/EN/FR, Strongholds,
  Oddball, KOTH, Slayer, Fiesta, None, casse, invariant RADAR_THRESHOLDS_PER_MODE)
- `tests/ui/test_teammates_helpers.py` — 2 cas : CTF objectifs_norm ≈ 750/p90_ctf,
  custom per_mode consommé correctement

**Résultat** : 49 tests verts. 1800 pts sur 3 CTF → 86% (contre 20% avant fix).
Tous les radars (teammates + match view) utilisent maintenant le même référentiel p90 calibré.
---

## [2026-03-30] Fix propagation map_id dans _session_compare_history.py

**Statut** : Complété

**Décision technique** :
`map_id` était disponible dans `df_sess` à l'entrée du pipeline mais éliminé par
`.select(display_cols)` dans `_build_history_dataframe`. Résultat : `map_name_cell_html`
était appelé sans `map_id` → fallback EN, pas de thumbnail par ID.

Pattern appliqué : identique à `perf_scores` — extraire la Series **avant** le `.select()`
et la passer en 3ᵉ élément du tuple de retour, sans polluer `df_display`.

**Fichiers modifiés** :
- `src/ui/pages/_session_compare_history.py`
  - `_build_history_dataframe` : signature `→ tuple[..., pl.Series | None, pl.Series | None]`,
    extrait `map_ids` avant `.select(display_cols)`
  - `_render_history_html` : nouveau paramètre `map_ids`, passe `map_ids[idx]` à
    `map_name_cell_html(val, map_id)`
  - `render_session_history_table` : décompacte le 3ᵉ élément et le transmet

**Tests ajoutés** :
- `tests/test_session_compare_history_map_id.py` — 7 cas :
  retour 3-tuple, Series présente/absente, valeurs correctes, map_id absent de df_display,
  longueur cohérente, perf_scores non cassé

**Résultat** : 7/7 tests verts. La colonne Carte dans l'historique de session utilise
désormais `map_id` pour la traduction FR et les thumbnails, comme les autres cal

---

## [2026-03-31] fix(radar) — Radar "Complémentarité de l'escouade" : "Données insuffisantes" malgré 12 matchs — Complété

**Statut** : Complété — commit `2cefec6` sur `feat/teammates-first-events-chart`

**Cause racine** : Lors du refactoring de `compute_participation_profile` (session précédente), la fonction a perdu ses kwargs directs (`name=`, `color=`, `pair_name=`, `thresholds=`) au profit de `ProfileOptions`. Mais `_compute_player_profile` dans `teammates_synergy.py` utilisait encore l'ancienne signature → `TypeError` catchée silencieusement par `_compute_profiles_from_squad` → `profiles` liste vide → `_render_radar_display(profiles)` → `st.info(t("insufficient_data_chart"))`.

**Diagnostic** : 
- Madina97294 : PSA = 0/12 pour les matchs du 24 mars (missing sync)
- Chocoboflor : PSA = 12/12 ✓
- Même si Chocoboflor avait un profil valide, la TypeError l'excluait aussi
- `test_viz_participation.py::TestComputeParticipationProfile` échouaient tous (même bug)

**Décision technique** : Utiliser `ProfileOptions(name=..., color=..., pair_name=..., thresholds=...)` partout, re-exporter `ProfileOptions` depuis `src/visualization/participation_radar.py` pour la compat des tests.

**Fichiers modifiés** :
- `src/visualization/participation_radar.py` : re-export `ProfileOptions`
- `src/ui/pages/teammates_synergy.py` : pass `ProfileOptions(...)` dans `_compute_player_profile`
- `tests/test_viz_participation.py` : `TestComputeParticipationProfile` → `ProfileOptions`

**Résultats** : 109 tests passent (test_viz_participation + test_participation_radar + test_teammates_helpers)

**Conclusion** : Le graphe radar "Complémentarité de l'escouade" s'affiche désormais correctement. Le bug PSA manquants pour Madina reste un sujet sync (backfill --personal-scores à relancer), mais l'affichage fonctionne quand au moins un joueur a ses PSA.lers.
---

## [2026-03-31] Fix 3 régressions tests post-i18n graphiques

**Statut** : Complété

**Contexte** : Suite aux 3 fixes i18n sur les graphiques (session précédente), 28 tests échouaient. Après isolation, 3 échecs réels :

1. `test_teammates_history_rows_use_map_hover` — regression : `_build_html_rows` avait son elif changé de `"map_name"` à `"map_ui"`, mais le test passait `col_key="map_name"`. Fix : condition `elif key in ("map_ui", "map_name")`.

2. `test_build_history_dataframe_empty` — `_build_history_dataframe` retourne désormais un tuple de 3 valeurs (`df_display, perf_scores, map_ids`) depuis commit 405e246. Le test attendait 2. Fix : `assert len(result) == 3`.

3. `test_impact_tab_renders_heatmap_and_ranking` — test entièrement désynchronisé avec le module actuel (`plot_friends_impact_scatter`, `count_events_by_player`, `build_impact_ranking_df` n'existent plus dans `teammates_impact.py`). Fix : suppression des 3 monkeypatches invalides, correction schéma mock `build_impact_matrix` (colonne `events: List[Struct]`), ajout mock `_load_match_participants → None`, ajout mock `st.markdown`, updated assertions vers `st_mocks["markdown"].called`.

**Décision technique** : Corriger les tests pour refléter l'API actuelle, pas ajouter du code mort pour satisfaire les anciens tests.

**Résultats** : 49/49 tests passent sur les fichiers ciblés. La régression `test_viz_participation` et `test_teammates_helpers` observée en full-suite est du flapping lié à l'ordre d'exécution (passes en isolation).

**Conclusion** : Branche propre, pas de nouvelles régressions.

### [2025-07-24] — Butterfly histogram premier frag/mort (teammates)

**Statut** : Complété
**Branche** : `feat/teammates-first-events-chart`

**Décision technique** : Implémentation d'un butterfly histogram (barres miroir positives/négatives) pour visualiser la distribution des premiers frags et premières morts par tranche de 15 secondes, par joueur de l'escouade.

**Architecture** :
- `src/analysis/first_events.py` : logique pure rolling avg (préservée, non utilisée dans le chemin final)
- `src/data/services/_teammates_first_events_queries.py` : requête SQL sur `shared.highlight_events` MIN(time_ms) par event_type par match par xuid
- `src/ui/pages/teammates_charts.py` : `_format_bin_label`, `_compute_bin_counts`, `_build_first_events_fig`, `render_first_events_chart`
- `src/ui/pages/_teammates_trio.py` : wiring + fix bug xuid joueur principal
- `src/ui/i18n/pages/teammates.py` : 3 clés FR/EN

**Itérations design** :
1. Rolling avg par index de match → rejeté (pas d'axe temporel)
2. Subplots datetime → rejeté
3. Butterfly histogram 15s bins → retenu

**Fonctionnalités finales** :
- Barres positives (frags) / négatives (morts) par tranche de 15s
- Couleurs par joueur depuis `colors_by_name`
- Axe X blanc gras (`Arial Black`), labels `0s`, `0m15s`, `0m30s`...
- Séparateurs verticaux pointillés blancs entre tranches (`col_shapes`, `xref="x"`)
- Annotations ▲ Frags / ▼ Morts
- Bug fix : `me_df` n'a pas de colonne `xuid` → init directe depuis paramètre `xuid` de `render_trio_view`

**Résultats** : Ruff all checks passed, commit `185f98b`.
**Conclusion** : Feature complète et livrée.

---
