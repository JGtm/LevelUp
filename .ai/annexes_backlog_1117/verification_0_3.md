# Verification sur pieces (item 0.3 du plan des entrees du backlog du 2026-11-17)

Faite le 2026-11-17, en lecture seule, sur le worktree `C:/t/bl1117` (branche `chore/830-backlog-1117`, tete `3ae75c78f5`). Chaque reference de fichier, de ligne, de symbole ou de chiffre des pistes A, B, C, D, F et G a ete rouverte. Statut : « confirme » ou « ecart : ... » avec la valeur reelle. Les chemins sont relatifs a la racine du depot ; `go-api/` = `apps/go-api/`, `web/` = `apps/web/src/`.

Ce que cette verification n'a PAS pu faire est en dernier paragraphe.

## ECARTS (liste seule, classes par piste)

### Piste A
- A-1. « 31 appels directs dans 27 fichiers » (section 1) : 30 appels reels dans 26 fichiers du paquet racine ; le 31e est une mention en commentaire (`migration_provider_test.go:13`).
- A-2. A.3 est incomplet : 13 appels reels, dans 13 fonctions, ne sont ni dans la liste de A.3 ni dans celle de A.4 (liste au paragraphe Piste A).
- A-3. A.3 et A.4 se contredisent sur `setupPipelineEnv` : le helper est a rebrancher (A.3) alors que son fichier `csr_pipeline_e2e_integration_test.go` est dans les exemptions de depart de A.4.
- A-4. Huit appels `duckdb.GetOrOpen(` (5 fichiers de test) font une migration neuve de facon INDIRECTE (`pool.go:208` puis `pool.go:381`, et `pool.go:261`) : ni A.3 ni le garde-rail de A.4 (qui cherche `RunForDB(`/`RunForTitleDB(` dans les `_test.go`) ne les voient.
- A-5. D'autres constructions de schema neuf echappent au garde-rail de A.4 : `migration.ForTarget(...)` + `ApplySchema` (2 fichiers) et `migration.ExecScript` (1 fichier).
- A-6. `RunForTitleDB(` n'est appele par AUCUN test du paquet (deux occurrences, toutes deux dans des chaines/commentaires de `no_attach_on_social_test.go`).
- A-7. La cible `shared_pve` (5e gabarit de A.1) n'a aucun consommateur dans les tests du paquet : `TargetSharedPvE` n'y apparait pas.

### Piste B
- B-1. Liste fermee de B.4 incomplete : au moins 6 autres fichiers de test ecrivent, lisent ou nomment le fichier tactique `.json` (liste au paragraphe Piste B).
- B-2. Le gate B (`grep -rn '"\.json"' ...replayartifacts ... replay_purge_cron.go`) rend DEJA une ligne aujourd'hui (`internal/sync/replayartifacts/t0film_test.go:32`, sans rapport avec le fichier tactique) : il ne discrimine pas. La regle reelle est deja tenue par `archlint/no_replay_json_literal_test.go`.
- B-3. `ReplaySidecarExt = ".json.gz"` aura la MEME valeur que `ReplayArtifactExt` (`film_id.go:87`) ; le commentaire de `film_id.go:91` (« JSON brut ») et l'en-tete de `no_replay_json_literal_test.go` (lignes 5-6) deviennent faux.
- B-4. La commande de controle de `docs/COMMANDS.md` (EN ligne 338, FR ligne 350 : `grep -l '"schema_version":8' <data>/cache/replays/halo_infinite/rasters/*.json`) ne lit plus un fichier gzip ; a mettre a jour avec B.6.

### Piste C
- C-1. `zonesPures` : « 6 appels » dans `service/tactical_service_lectures.go` est faux. Cette fonction y a UN appel (ligne 69). Le paquet `service` compte 7 sites d'appel (5 de production dans 4 fichiers, 2 de test).
- C-2. C.3 : `paceMatch` est alimente, pour le joueur principal, par `domain.SquadMatchRow` (entiers : `Kills`, `Deaths`, `Assists`) via `mainPaceMatches` (`teammates_squad_per_minute.go:98-110`) ; un compte inconnu n'y est pas representable sans changer ce type. Seule la branche des coequipiers (lignes 155-164) lit des pointeurs (`r.Self.Kills`...) et les convertit en 0 par `intPtrOrZero`.
- C-3. C.7 : les lignes 81 et 139 de l'ADR 0024 disent 200 (pas 500) ; c'est bien ce qu'il faut corriger (le code dit 500 : `ep/match2team.go:111`). Pas d'ecart sur le fond, a lire dans ce sens.
- C-4. C.5 : le chemin absolu des deux lignes de `sgh_interlock.json` NAIT dans la brique `halo-infinite-map-images` (`himap/sddt.go:476`, `fmt.Errorf("module %s/%s introuvable : %w", ...)`, v1.3.1 du cache de modules) ; `ecritSidecar` ne fait que recopier `b.Degradations` (`cuisson.go:242`). Le correctif ne peut donc etre qu'une reecriture de la chaine cote LevelUp.
- C-5. C.8 : `REGISTRE_CARTES.md` n'est pas a `.ai/` mais a `.ai/archive/V7.5/cartes/REGISTRE_CARTES.md` ; le seuil « 1 000 objets » est cite aux lignes 1057 et 1246 de ce fichier (pas seulement pour Thunderhead), et aux lignes 650 et 659 de `map_fond_reglages.json` (les autres mentions de Thunderhead de ce fichier, « n'en publie pas », sont exactes).

### Piste D
- D-1. Le plan ecrit « `shadow`, `from`, `via`, `to` entrent dans les prefixes balayes » : ils sont DEJA dans `ABSOLUTE_PREFIXES` (noir et blanc absolus) ; ils manquent seulement dans `TW_PREFIXES` (couleurs nommees).
- D-2. `tools/` est a la racine du depot, pas sous `apps/web/` : le gate commun web `node tools/lint-no-hardcoded-colors.mjs` se lance depuis la racine (ou `npm run lint:colors` depuis `apps/web`).
- D-3. Comportement actuel qui change a la migration (D.1) : `_shared/ExperienceDropdown.tsx` et `_shared/ViewDropdown.tsx` n'ecoutent AUCUN Echap aujourd'hui (clic dehors seul, ecouteur permanent, non garde par `open`) ; `explorer/GamertagSearchInput.tsx` ferme sur Echap par un `onKeyDown` React de l'`<input>` (pas par un ecouteur de document).
- D-4. `ALLOWED_FILES` (`tools/lint-no-hardcoded-colors.lib.mjs:68-69`) porte deux entrees mortes (`features/home/rarity.ts`, `features/battlepass/rarity.ts`, fichiers absents).

### Piste F
- F-1. BLOQUANT POUR F.1 : retirer `hits.go` casse `internal/filmreplis` tant que la brique decodeur v1.4.1 marque `repli_distances_de_touche_desactivees` et `repli_precision_par_arme_passe_sautee` « completee par l'hote » (details au paragraphe Piste F). Le plan interdit toute montee de brique (D-6).
- F-2. `ConfigureFilmAccuracy` n'a AUCUN appelant de production : la passe est morte deux fois (capacite non exposee ET jamais configuree). Aucune preuve supplementaire n'est contraire au plan, mais la raison donnee (« ne s'executent plus depuis le 2026-09-01 ») est incomplete.
- F-3. L'inventaire est plus large que les quatre cibles nommees : `hits.go` partage `nomsDeCarteDuMatch`, `entreeDeCatalogueParNom`, `premierNomSansArbitrage` avec la passe des positions (a GARDER) ; des tests partages (`collecteurAvecCarte`) doivent survivre.
- F-4. Le gate F `grep -rn "PairWeaponHits\|WeaponHitDistancePersister" apps/go-api` rend aussi des commentaires (`ingest/weapon_accuracy_film.go:7` sera supprime ; `persist/usage_summary_persister.go:6`, a reecrire).

### Piste G
- G-1. Le profil `dev` n'a PAS la capacite `ReplayRegen` (`config/deployment.go:312`, `ReplayRegen: desktop`) : la capacite neuve `can_generate_replays` ne peut pas se deriver de celle-ci pour couvrir « application de bureau et poste de developpement ».
- G-2. L'action d'administration `replay-build/enqueue` (`admin_actions_replay_build.go:41-60`, route `/enqueue`) resout deja le manifeste du film (`resolveFilmChunkURLs`, `api/wire/registry_build_queue.go:153-163`) avec des jetons pris par `haloTokensForDrain` (ligne 159 -> `api/wire/registry_catalog_drain.go:151`) qui prend le premier jeton valide et dont l'erreur conseille de « re-capturer un token » : la decouverte 3 du plan est confirmee (ligne 171).
- G-3. `ReplayPage` : la route de rejeu repond 404 quand aucun artefact n'existe ; l'etat vide est le bloc `loadState === 'empty'` (lignes 245-249), qui n'a ni titre ni action.

---

## Piste A — paquet `internal/platform/duckdb`

| Reference du plan | Constat | Statut |
|---|---|---|
| A.1 `migration_provider_test.go` porte le `TestMain` existant | `go-api/internal/platform/duckdb/migration_provider_test.go:16`, `TestMain` = `migration.SetTitleStepsProvider(halomigrations.StepsFor)` puis `os.Exit(m.Run())`. Il ne cree ni ne retire aucun dossier ; `os.Exit` saute les `defer` : le nettoyage du gabarit devra se faire avant `os.Exit` | confirme (l'extension reste a ecrire) |
| A.1 5 cibles `shared`, `player`, `metadata`, `shared_social`, `shared_pve` | `go-api/internal/migration/registry.go:21-25` : `TargetPlayer`, `TargetShared`, `TargetSharedPvE`, `TargetSharedSocial`, `TargetMetadata` | confirme (voir A-7) |
| A.1 `migrated_template_test.go` neuf | n'existe pas | confirme |
| A.4 `no_fresh_migration_in_tests_test.go` neuf | n'existe pas | confirme |
| Section 1 « 31 appels directs dans 27 fichiers » | 30 appels `migration.RunForDB(` reels dans 26 fichiers `_test.go` du paquet racine, plus 1 commentaire (`migration_provider_test.go:13`). Les paquets voisins `profilerepo` (2 appels), `worldrepo` (1), `prestige` (4) ont leurs propres appels : paquets distincts, hors du compte | ecart A-1 |
| `RunForTitleDB(` | 0 appel dans les tests du paquet ; seulement `no_attach_on_social_test.go:260` et `:319` (chaines d'allowlist) | ecart A-6 |

### Liste fermee des appels `migration.RunForDB(` dans les `*_test.go` du paquet `internal/platform/duckdb` (30)

Fichier:ligne, helper (ou test) qui le contient, cible. Colonne « A.3/A.4 » : N = nomme en A.3, X = exemption de depart de A.4, « — » = nomme nulle part.

| # | Fichier:ligne | Helper ou test contenant l'appel | Cible | A.3/A.4 |
|---|---|---|---|---|
| 1 | `achievements_repo_test.go:22` | `openAchievementsPlayerDB` | player | — |
| 2 | `achievements_repo_test.go:37` | `openAchievementsMetadataDB` | metadata | — |
| 3 | `battlepass_cache_test.go:22` | `openBattlePassTestDB` | parametre | N |
| 4 | `career_progression_fresh_migrations_test.go:44` | `newFreshMigratedPlayerDB` | player | X |
| 5 | `career_repo_lusr_exclusion_test.go:30` | `openMigratedFile` | parametre | — |
| 6 | `catalog_repo_test.go:23` | `setupCatalogRepoDB` | metadata | N |
| 7 | `csr_pipeline_e2e_integration_test.go:69` | `setupPipelineEnv` | player | N et X |
| 8 | `csr_pipeline_e2e_integration_test.go:93` | `setupPipelineEnv` | metadata | N et X |
| 9 | `csr_thresholds_repo_test.go:22` | `openTempMetadataDBWithThresholds` | metadata | N |
| 10 | `fenetres_perimetre_helpers_test.go:130` | `newBaseNotee` | shared | N |
| 11 | `killsource_fixture_test.go:100` | `newKillSourceTestPlayerDB` | shared | N |
| 12 | `last_match_media_stores_test.go:33` | `migratedFile` | parametre | N |
| 13 | `media_loadfiles_real_topology_test.go:49` | `TestLoadMediaFiles_RealTopology_NoCrossDBSQL` (fermeture `open`) | parametre | X |
| 14 | `migrated_db_test_helper_test.go:36` | `setupPrestigeDB` | parametre | N |
| 15 | `notifications_repo_test.go:33` | `newNotifTestDB` | shared_social | N |
| 16 | `objective_events_repo_test.go:35` | `newObjectiveEventsTestPlayerDB` | shared | — |
| 17 | `objective_stats_repo_test.go:46` | `newObjectiveStatsRepoOnMem` (sous `if migrate`) | shared | N |
| 18 | `player_positions_repo_test.go:45` | `newPlayerPositionsTestPlayerDB` | shared | — |
| 19 | `pool_shared_social_recovery_concurrent_test.go:40` | `TestOpenSharedSocial_MigrationIdempotent_AfterRecovery` | shared_social | X |
| 20 | `pool_shared_social_recovery_concurrent_test.go:53` | idem | shared_social | X |
| 21 | `pool_shared_social_recovery_concurrent_test.go:58` | idem | shared_social | X |
| 22 | `prestige_telemetry_diag_repo_test.go:26` | `openTelemetryDiagPlayerDB` | player | — |
| 23 | `queries_career_lusr_latest_test.go:43` | `newPlayerFixtureDB` | player | — |
| 24 | `replay_map_repo_carte_test.go:31` | `newCarteRepo` | shared | — |
| 25 | `resumes_fixture_test.go:37` | `ouvrirSharedMigree` | shared | — |
| 26 | `session_usage_aggregate_integration_test.go:45` | `setupUsageSharedDB` | shared | — |
| 27 | `tactical_repo_cartes_test.go:224` | `TestTacticalRepo_TablesAbsentes_Capability` (migre puis `DROP TABLE` x5) | shared | — |
| 28 | `tactical_repo_coequipiers_test.go:143` | `pdbSansTable` (migre puis `DROP TABLE`) | shared | — |
| 29 | `tactical_repo_contextes_test.go:88` | `TestTacticalRepo_ContextesDeMort_TableAbsente` (migre puis `DROP TABLE`) | shared | — |
| 30 | `tactical_repo_test.go:71` | `newTacticalTestPlayerDB` | shared | N |

Les 11 helpers de A.3 existent tous, chacun contient exactement un appel (sauf `setupPipelineEnv` : 2) : `newKillSourceTestPlayerDB` `killsource_fixture_test.go:92`, `newTacticalTestPlayerDB` `tactical_repo_test.go:63`, `newBaseNotee` `fenetres_perimetre_helpers_test.go:115`, `newObjectiveStatsRepoOnMem` `objective_stats_repo_test.go:36`, `newNotifTestDB` `notifications_repo_test.go:22`, `openBattlePassTestDB` `battlepass_cache_test.go:13`, `setupPrestigeDB` `migrated_db_test_helper_test.go:27` (un homonyme existe dans `prestige/prestige_repos_test.go:29`, autre paquet), `migratedFile` `last_match_media_stores_test.go:24`, `openTempMetadataDBWithThresholds` `csr_thresholds_repo_test.go:13`, `setupCatalogRepoDB` `catalog_repo_test.go:16`, `setupPipelineEnv` `csr_pipeline_e2e_integration_test.go:52`. Confirme.

**Ce qui manque a A.3 (13 appels, 13 fonctions)** : `openAchievementsPlayerDB`, `openAchievementsMetadataDB`, `openMigratedFile`, `newObjectiveEventsTestPlayerDB`, `newPlayerPositionsTestPlayerDB`, `openTelemetryDiagPlayerDB`, `newPlayerFixtureDB`, `newCarteRepo`, `ouvrirSharedMigree`, `setupUsageSharedDB`, `TestTacticalRepo_TablesAbsentes_Capability`, `pdbSansTable`, `TestTacticalRepo_ContextesDeMort_TableAbsente`. Les trois derniers migrent puis suppriment des tables : ils passent au gabarit puis gardent leurs `DROP`.

**A.4, liste de depart** : `career_progression_fresh_migrations_test.go` (ligne 44 du fichier), `pool_shared_social_recovery_concurrent_test.go` (lignes 40, 53, 58), `csr_pipeline_e2e_integration_test.go` (lignes 69 et 93, contradiction A-3), `media_loadfiles_real_topology_test.go` (ligne 49) : les quatre fichiers existent et portent bien un appel. Il faut y AJOUTER le fichier du gabarit lui-meme (`migrated_template_test.go`, deja prevu).

### Migrations neuves indirectes (non vues par le garde-rail de A.4)
- `duckdb.GetOrOpen(` : `pool_attach_swap_integration_test.go:116`, `pool_migration_test.go:28`, `:70`, `:99`, `pool_shared_reader_integration_test.go:71`, `:125`, `:186`, `pool_swap_multititre_integration_test.go:103`. Chemin : `pool.go:139` (`GetOrOpen`) -> `pool.go:207` (`openPlayerDB`) -> `pool.go:208` (`ensurePlayerDBMigrations`, `RunForDB` a `pool.go:381`), et `RunForDB(shared_social)` a `pool.go:261`.
- `migration.ForTarget(` + `ApplySchema` : `leaderboard_world_repo_test.go:32` (`applyWorldLeaderboardMigration`), `world_player_stats_repo_test.go:21` (`applyWorldPlayerStatsMigration`) ; ces deux-la n'appliquent que quelques steps nommes, pas la cible entiere.
- `migration.ExecScript(` : `weapon_names_testutil_test.go:16` (une seule table).
- `migration.All()` (12 occurrences) n'enregistre que les steps, il ne migre rien.

## Piste B — fichiers tactiques `rasters/*.json`

| Reference du plan | Constat | Statut |
|---|---|---|
| B.2 `title.ReplaySidecarExt = ".json.gz"` dans `internal/domain/title/film_id.go` | `go-api/internal/domain/title/film_id.go:92` : `ReplaySidecarExt = ".json"` (ligne 91 : commentaire « JSON brut ») | confirme la cible ; ecart B-3 sur la valeur future |
| B.2 ecrivain `replayartifacts.EcrireSidecarRaster` | `go-api/internal/sync/replayartifacts/raster.go:308` : `json.Marshal`, `os.MkdirAll`, `atomicfile.WriteFile(context.Background(), path, raw, permSidecarRaster)`. Appelants : `raster.go:357` (`projeterRastersTactiques`) et `cmd/levelup/cmd_tactical_rasters.go:194` ; tests `raster_test.go:174`, `derivations_raster_ancien_integration_test.go:48`. C'est l'ecrivain UNIQUE | confirme |
| B.2 lecteur 1 `service/tactical_service_rasters.go` | `go-api/internal/service/tactical_service_rasters.go:68-70`, `(*tacticalRasterStore).Charger` : `os.ReadFile` sur `TacticalRasterPath`, absence = `(nil, nil)`, puis `json.Unmarshal` | confirme |
| B.2 lecteur 2 `service/replay_spawn_candidats.go` | `go-api/internal/service/replay_spawn_candidats.go:72` (appel) et `:95-96` (`lireSidecarReapparitions` : `os.ReadFile` puis `json.Unmarshal` vers `sidecarReapparitions`) | confirme |
| B.2 lecteur 3 `cmd/levelup/cmd_tactical_rasters.go` | `go-api/cmd/levelup/cmd_tactical_rasters.go:242-243` (`lireSidecarRaster` : `os.ReadFile`), appele par `sidecarAJour` (ligne 217) | confirme |
| « trois lecteurs » | aucun autre `os.ReadFile` d'un chemin `TacticalRasterPath` en production | confirme |
| B.3 purge 1 `scheduler/replay_purge_cron.go` « ne reconstruit plus le chemin a la main » | `go-api/internal/scheduler/replay_purge_cron.go:246-251` (`supprimerSidecarRaster`, appelee ligne 213) construit le chemin a la main : `filepath.Join(artifactsDir, titlePkg.SousDossierRasters, short+titlePkg.ReplaySidecarExt)` puis `os.Remove` (absence tolerante, autre erreur en `WARN`). Il n'a que `artifactsDir`, pas de `PathResolver` | confirme |
| B.3 purge 2 `replaybuild/artifact_purge.go` | `go-api/internal/replaybuild/artifact_purge.go:28` : `candidats := []string{artefact, DerivationsMarkPath(artefact), MetaPath(artefact), pr.TacticalRasterPath(titleSlug, matchID)}`, `os.Stat` + `os.Remove`, dry-run supporte | confirme |
| B.1 `cacheAEmpreinte` dans `service/catalogue_cache.go` | `go-api/internal/service/catalogue_cache.go:50` (type generique `cacheAEmpreinte[T]`), constructeur `nouveauCacheAEmpreinte` (ligne 64), `lire(chemin)` (ligne 69) : `os.Stat`, invalidation sur date ET taille, absence = erreur de `os.Stat` (`fs.ErrNotExist`), echec de lecture non garde. Meme paquet `service` que `replay_spawn_candidats.go` | confirme |
| B.2 `replayartifact.Write`, `Read`, `ReadBytes` | `go-api/internal/replayartifact/gate.go:166` `Write(ctx, outPath string, doc []byte) (written string, stored []byte, err error)` (gzip niveau 6, atomique, cree le dossier) ; `:97` `Read(path) (Content, error)` (absence = erreur enveloppant `fs.ErrNotExist`, illisible = `ERROR` + compteur) ; `:114` `ReadBytes(path) ([]byte, error)`. La porte n'a pas de repli `.json` brut (en-tete du paquet) | confirme |
| B.4 tests adaptes, liste fermee | `cmd/levelup/cmd_tactical_rasters_test.go`, `sync/replayartifacts/raster_test.go`, `scheduler/replay_purge_cron_test.go`, `service/replay_spawn_forecast_test.go`, `service/tactical_service_rasters_test.go`, test du `PathResolver` (`domain/title/film_short_id_test.go:93-100`, assertion de suffixe `rasters/000d5950.json` ; et `match_id_path_test.go:35`) existent tous | confirme ; liste incomplete (ecart B-1) |

**Ecart B-1, fichiers de test touchant le fichier tactique et absents de la liste de B.4** : `go-api/internal/sync/replayartifacts/raster_reapparitions_test.go:201` (`os.ReadFile` du sidecar) et `:264`, `raster_integration_test.go:125` et `:154`, `derivations_raster_ancien_integration_test.go:40` et `:48`, `go-api/internal/replaybuild/artifact_purge_test.go:34`, `go-api/internal/ops/seed_demo_replays_test.go:211` (attend `rasters/<garde>.json` intact apres la purge de la demo), `go-api/cmd/server/demo_paths_test.go:144` (dossier seulement), `go-api/internal/archlint/no_replay_json_literal_test.go` (en-tete lignes 5-6 ; pas de changement de code). Les assertions `os.Stat` de `raster_test.go:223-237` sont, elles, dans la liste.

**Garde-rails existants proches de B.4** : `archlint/no_artefact_bytes_outside_gate_test.go` interdit deja `os.ReadFile`/`os.Open`/`os.Stat`/`os.WriteFile` sur un chemin d'artefact hors de `internal/replayartifact/` et `internal/replaybuild/`, mais son critere est lexical (`art[ei]fact|rejeu` dans l'argument ou le nom de fonction) : les trois lecteurs du fichier tactique ne le declenchent pas (leurs identifiants disent `sidecar`, `raster`). Il faut donc un garde-rail neuf ou l'extension de celui-ci ; `archlint/no_replay_json_literal_test.go` tient deja « aucun litteral `.json`/`.json.gz` dans une fonction qui nomme `TacticalRasterPath` ou `SousDossierRasters` » (motif ligne 39).

**Gate B** : `grep -rn '"\.json"' go-api/internal/sync/replayartifacts go-api/internal/scheduler/replay_purge_cron.go` rend aujourd'hui `go-api/internal/sync/replayartifacts/t0film_test.go:32` (`strings.TrimSuffix(nom, ".json")`, rien a voir) : ecart B-2.

**B.6, documents qui citent l'extension** : `docs/COMMANDS.md:331-338` (EN) et `docs/FR/COMMANDS.md:344-350` (FR), `go-api/cmd/levelup/main.go:309-311` (aide de la commande `tactical-rasters`, cite `data/cache/replays/{slug}/rasters/`), `docs/CHANGELOG.md:25` et `docs/FR/CHANGELOG.md:25` (format 8). Confirme : COMMANDS FR et EN sont concernes.

## Piste C — petits correctifs Go

| Item | Reference du plan | Constat | Statut |
|---|---|---|---|
| C.1 | `service/tactical_service_lectures.go`, `zonesPures`, « 6 appels » | definition `go-api/internal/service/tactical_service_lectures.go:81` ; un seul appel dans ce fichier (`:69`). Appels du paquet : `replay_spawn_forecast.go:187`, `tactical_service_cellule_enrichir.go:85`, `tactical_service_lectures.go:69`, `tactical_service_reapparitions_adverses.go:65` et `:76` (production, 5), `tactical_callouts_test.go:38` et `:61` (tests, 2) | ecart C-1 (7 appels, pas 6 dans ce fichier) |
| C.1 | `zonesPuresDuCatalogue` dans `cmd/levelup/cmd_spawn_model_zones.go` | `go-api/cmd/levelup/cmd_spawn_model_zones.go:41` (appel `:34`), son commentaire se dit « seconde copie » | confirme |
| C.1 | `internal/analysis/tactical`, `domain.ZoneNommee` | `tactical.ZoneNommee` `analysis/tactical/spawn.go:72` ; `domain.ZoneNommee` `domain/tactical_raster.go:326` (le commentaire dit que le jumeau est voulu) ; le paquet `analysis/tactical` importe deja `domain` (`equipe.go`, `grid.go`, `merge.go`, `placement.go`...) : `ZonesDepuisLePort([]domain.ZoneNommee)` n'ajoute pas de dependance. Aucune autre construction de `tactical.ZoneNommee{` qu'a ces deux projections (hors un litteral de test, `tactical_callouts_test.go:89`) | confirme |
| C.2 | `internal/migration/psa_index_repro_msr_fixture_test.go:173` | `{"F1", "Q24LUSRHistory (platform/duckdb/queries_career_encounters.go)", ...` | confirme |
| C.2 | `psa_index_repro_msr_planprobe_test.go:34` | `// TENU le 2026-09-27 ... : F1 (Q24LUSRHistory, 12 000 lignes lues),` ; la fonction n'existe plus (`no_raw_rating_reads_test.go:22-23` le dit : retiree le 2026-11-17, S.2) ; le fichier `queries_career_encounters.go` existe encore | confirme |
| C.3 | `service/teammates/teammates_squad_per_minute.go`, `paceMatch` | `go-api/internal/service/teammates/teammates_squad_per_minute.go:19` : `type paceMatch struct{ kills, deaths, assists, secs int }` ; `matchRate` (ligne 46) utilise `analysis.PooledPerMinute` avec `ok = true` partout pour les comptes ; `damageRate` (ligne 59) retire deja les matchs inconnus | confirme ; ecart C-2 sur la source du joueur principal |
| C.3 | tests de caracterisation | `per_minute_caracterisation_test.go`, `per_minute_temoins_test.go` (et `per_minute_sans_duree_test.go`) existent et passent ; voir `temoins_avant.md` | confirme |
| C.4 | `scripts/git-hooks/lefthook/go-vet.sh` | 11 lignes : `pkgs=$(go list ./... ... grep -v "cmd/server")`, `CGO_ENABLED=0 go vet $pkgs 2>&1 \|\| true` ; en-tete : « non bloquant (`\|\| true`) » | confirme |
| C.4 | cibles de la CI | `.github/workflows/ci.yml:431-432` : `go vet ./internal/domain/... ./internal/analysis/... ./internal/games/weapons/...` | confirme |
| C.4 | commentaires de `lefthook.yml` | `lefthook.yml:50-52` (« go vet — informatif, non bloquant (\|\| true dans le script) ») et `:113-120` (hook `go-vet-cgo`, bloquant, existe : `scripts/git-hooks/lefthook/go-vet-cgo.sh`) | confirme |
| C.5 | `cmd/mapfond-build/cuisson.go`, `ecritSidecar` | `go-api/cmd/mapfond-build/cuisson.go:228` ; `Degradations: b.Degradations` ligne 242 ; ecriture par `mapfondsidecar.Write` | confirme ; ecart C-4 |
| C.5 | `data/titles/halo_infinite/reference/map_backgrounds/sgh_interlock.json`, deux lignes | fichier suivi par git ; lignes 44-45 : `"sddt : module any/globals introuvable : GetFileAttributesEx C:\\XboxGames\\Halo Infinite\\Content\\deploy\\any\\levels\\multi\\globals\\common-rtx-new.module: ..."` et `"eau : ..."` (meme chemin) ; ligne 46 : `"aucun maillage de navigation dans le module any"`. Sur les 339 entrees du dossier, `grep XboxGames` ne rend que ce fichier : aucun autre sidecar publie ne porte de chemin absolu | confirme |
| C.6 | `config/replay_corpus.toml`, temoin `f75e7053` | `config/replay_corpus.toml:277` (`id = "f75e7053"`, famille `koth_collines`, `Ranked:King of the Hill`, carte `Solitude - Ranked`, raison lignes 281) | confirme |
| C.6 | ligne de `.ai/REGISTRE_REPORTS.md` | `.ai/REGISTRE_REPORTS.md:268` (« Le temoin `f75e7053` ... est absent du parc local ... `config/replay_corpus.toml:277` ») | confirme |
| C.7 | ADR 0024, 500 iterations lignes 81 et 139 | `docs/adr/0024-lusr-v2-trueskill2-with-counts.md:81` « MaxIters = 200 », `:139` « en 200 iterations EP » ; code : `go-api/internal/analysis/skill_v2/ep/match2team.go:106-111` (`MaxIters: 500`) | confirme (lire comme : corriger 200 en 500, ecart C-3) |
| C.7 | limites 4, 5, 6 livrees | 4 : `analysis/skill_v2/mode_correlation.go`, `mode_correlation_matrix.go` ; 5 : `analysis/skill_v2/squad.go`, `platform/duckdb/squad_offset_repo.go` ; 6 : `sync/skill/skill_v2_quit_penalty.go`, `analysis/skill_v2/trueskill_ep.go`, `domain/skill_v2.go` | confirme (fichiers existent ; aucune lecture de logique poussee) |
| C.7 | limite 7, `applyMatchToSkillV2` | `grep applyMatchToSkillV2` dans `apps/go-api` : 0 resultat (seul l'ADR 0024:144 le cite) | confirme (symbole absent) |
| C.7 | cinq liens vers `.ai/archive/V7/LUSR%20v2/` | liens aux lignes 192, 193, 195, 196, 197, tous vers `../../.ai/V7/LUSR%20v2/...` (chemin qui n'existe pas) ; le dossier reel est `.ai/archive/V7/LUSR v2/` et contient `trueskill2.txt`, `menke.txt`, `lusr_v2_phase0_metrics.md`, `lusr_v2_phase1d_replay.md`, `lusr_v2_phase3d_replay.md` | confirme |
| C.7 | amendement daté | deux amendements existent deja (`:182` 2026-10-17, `:201` 2026-11-09) ; corps de l'ADR en francais, amendements en anglais | confirme |
| C.8 | `cmd/mapfond-build/sans_plan_test.go:63` | `go-api/cmd/mapfond-build/sans_plan_test.go:63-65` : « Live Fire n'a pas de maillage dans son module ; les deux variantes de Thunderhead n'ont pas de blob au depot. » ; la liste `fondsSansPlanAttendus` suit (lignes 66-70) | confirme |
| C.8 | `map_fond_reglages.json` | `data/titles/halo_infinite/reference/map_fond_reglages.json:650` et `:659` : « PAS DE MAILLAGE PUBLIE par l asset (mesure : le navmesh n existe qu au-dela d environ 1 000 objets) » ; autres lignes (506, 534, 560, 574, 585, 650, 659...) : « Thunderhead n en publie pas » (exact) | confirme ; ecart C-5 |
| C.8 | `REGISTRE_CARTES.md` | `.ai/archive/V7.5/cartes/REGISTRE_CARTES.md:1246` (« n'ont pas de maillage publie (sous le seuil des ~1 000 objets) »), `:1057` (regle generale), tables `:1318-1319` et `:1429-1430` (« pas de maillage ») | confirme ; ecart C-5 |

## Piste D — web

### D.1, les 12 fichiers : comment chacun ecoute Echap ou le clic dehors aujourd'hui

Chemins sous `web/features/`. « Doc » = ecouteur sur `document`, « Win » = sur `window`.

| Lot | Fichier:ligne | Echap | Clic dehors | Particularites | Statut |
|---|---|---|---|---|---|
| 1 | `media/MediaAudioConfigButton.tsx:80-86` | Doc `keydown` (`e.key === 'Escape'` ligne 82) -> `onClose()`, actif tant que le composant est monte (pas de garde `open`) | aucun | modale | confirme |
| 1 | `media/CoverFlowModal.tsx:552-561` | Win `keydown`, handler partage avec ArrowLeft/ArrowRight (Echap ligne 554) | aucun | le fichier compte 957 lignes (> 500) : « ne grossit pas » est un vrai contrainte ; `eslint-disable` de dependances ligne 559 | confirme |
| 1 | `media/MediaToolbar.tsx:55-71` (`AuthorsMultiSelect`) | Doc `keydown` (ligne 63), garde `if (!open) return` (ligne 56) | Doc `mousedown` (ligne 65), `containerRef` | | confirme |
| 2 | `asset-drawer/AssetDrawer.tsx:19-26` | Doc `keydown` (ligne 22) -> `close()`, garde `isOpen` | aucun | | confirme |
| 2 | `feedback-drawer/FeedbackDrawer.tsx:77-84` | Doc `keydown` (ligne 80) -> `close()`, garde `isOpen` | aucun | | confirme |
| 2 | `spartan-customizer/SpartanCustomizerModal.tsx:110-116` | Win `keydown` (ligne 112) -> `onClose()`, sans garde | aucun | | confirme |
| 2 | `palmares/BattlePassRewardLightbox.tsx:125-134` | Win `keydown` partage avec les fleches (Echap ligne 127) | aucun | les fleches gardent leur ecouteur | confirme |
| 2 | `ascension/campaign/StartCampaignModal.tsx:55-65` | Win `keydown` (ligne 58), garde `open`, condition `!muts.start.isPending`, `event.preventDefault()`, `onOpenChange(false)` | aucun | la condition « envoi en cours » passe dans `onClose` comme prevu | confirme |
| 3 | `_shared/ExperienceDropdown.tsx:33-39` | AUCUN | Doc `mousedown` (ligne 37), permanent (deps `[]`, pas de garde `open`) | ecart D-3 | confirme + ecart |
| 3 | `_shared/ViewDropdown.tsx:23-29` | AUCUN | Doc `mousedown` (ligne 27), permanent | ecart D-3 | confirme + ecart |
| 3 | `explorer/GamertagSearchInput.tsx:58-66` et `:101-116` | `onKeyDown` React de l'`<input>` (ligne 113, branche `else if (e.key === 'Escape')` -> `setOpen(false)`) | Doc `mousedown` (ligne 64), permanent | `open` est l'etat, `showDropdown` = `open && trimmed.length > 0 && (...)` (lignes 82-93) : « `open` = liste visible » = passer `showDropdown` au hook | confirme + ecart D-3 |
| 3 | `notifications/NotificationsBell.tsx:112-140` | Doc `keydown` (ligne 125) avec garde `open` (ligne 123), dans un effet qui gere aussi ArrowUp/ArrowDown | Doc `mousedown` (ligne 117), permanent | « les fleches gardent leur ecouteur » : l'effet reste pour les fleches | confirme |

Confirmation chiffree : le garde-rail compte exactement ces 12 + les 5 de D.2 = 17 fichiers (liste complete dans `temoins_avant.md` §3). `web/lib/hooks/useDismissable.ts` existe (120 lignes) : signature `useDismissable(open, onClose(reason), options)`, options `ref`, `outside` (`'mousedown'` par defaut, `'pointerdown'`, `null`), `ignore`, `innerLayerSelector` ; un seul ecouteur d'Echap de fenetre sert toutes les couches (la plus recente se ferme seule) ; hook deja utilise par `components/shell/_filter_pills/{Filtres,Periode,Saison,Session}Pill.tsx` et par le rejeu.

### D.2, les 5 cas particuliers

| Fichier:ligne | Constat | Statut |
|---|---|---|
| `asset-drawer/AssetDockedPanel.tsx:60-65` | `onPanelKeyDown` (gestionnaire React du volet) : Echap -> `e.stopPropagation()` puis `closeAssetPanel()` ; limite au volet ; tenu par `AssetDockedPanel.test.tsx:189` (`keyDown` sur le `searchbox`) et `:197` (`keyDown` sur `document.body`) | confirme |
| `tactical/TacticalAnalysisView.tsx:293-305` et `:326-333` | deux ecouteurs d'Echap de fenetre en pile : `useEchapEcarteLaZone` (ligne 297 `!== 'Escape'`, ignore si `event.defaultPrevented` ou cible editable, actif seulement `actif`) et `useContributionChoisie` (ligne 329, garde `choisie !== null`). Le premier doc dit que « le meme Echap referme d'abord la mort, un second ecarte la zone » | confirme |
| `tactical/TacticalPlanTueurs.tsx:151` | `cadre.addEventListener('pointerdown', surAppui, true)` : phase de capture d'un clic sur un point de la toile (distinction clic/glisser), aucune fermeture de couche | confirme (faux positif) |
| `squad/SquadFocusStrip.tsx:209-216` | `onKeyDown` d'un `<input>` de renommage : Entree valide, Echap (ligne 213) `setEditing(false)` | confirme (faux positif) |
| `squad/useSquadPresets.tsx:198-205` | meme patron (ligne 202) : `setEditingId(null)` | confirme (faux positif) |

Garde-rail : `web/features/dismissableLayers.guard.test.ts`, constante `PLAFOND_HORS_REJEU = 17` ligne 28, `REPLAY = 'features/match-replay/'` ligne 25. Le compte en fichiers est celui que D.2 remplace par une liste d'exemptions de 5 : confirme.

### D.4, les deux commentaires faux

| Fichier:ligne | Texte actuel | Statut |
|---|---|---|
| `go-api/internal/domain/replaydoc/inventory.go:96-103` | commentaire de `Stance` (96-101) : « `Kind` vaut `crouch` ..., `slide` ... ou `clamber` ... et rien d'autre : le sprint est refute comme observable par la vitesse et le saut n'est pas prouve (cf. `replay/document_stances.go`) » ; la declaration `type Stance struct` est ligne 102 ; le commentaire de `Kind` (lignes 104-106) cite `jumpDerived` (derive, schema 66) | confirme (le passage faux est 98-100) |
| `web/features/match-replay/model/replayContract.test.ts:131-133` | « `stances` : les ETATS DE MOUVEMENT ... Trois genres seulement : le sprint est refute comme observable par la vitesse et le saut n'est pas prouve (lot 5.3.5). » | confirme |

La veracite de la correction (le sprint est bien lu par le decodeur) vit dans le depot du decodeur : non verifiee ici.

### D.3, D.5, D.6

| Reference | Constat | Statut |
|---|---|---|
| D.3, 19 marqueurs inertes | liste fermee dans `.ai/annexes_backlog_1117/marqueurs_inertes.md` | confirme (19 sur 95) |
| D.5 `tools/lint-no-hardcoded-colors.lib.mjs` | `TW_PREFIXES` (lignes 36-39) : `text, bg, border, ring, ring-offset, fill, stroke, divide, placeholder, outline, decoration, accent, caret` ; `ABSOLUTE_PREFIXES` (lignes 47-51) contient deja `from, via, to, shadow` | ecart D-1 |
| D.5 `lintNoHardcodedColors.guard.test.ts` | `web/lib/accessibility/lintNoHardcodedColors.guard.test.ts` (importe la lib par `../../../../../tools/...`) | confirme |
| D.5/Section 1 « 0 occurrence a corriger pour `shadow`, `from`, `via`, `to` ; les 15 trouvees sont dans `palmares/rarity.ts` » | balayage `(shadow\|from\|via\|to)-<couleur nommee>-<teinte>` sur `features/`, `components/`, `lib/replay/` : 15 occurrences, toutes dans `features/palmares/rarity.ts` ; 0 ailleurs | confirme |
| D.6 `palmares/rarity.ts` dans la liste des fichiers autorises | `tools/lint-no-hardcoded-colors.lib.mjs:70` (`'features/palmares/rarity.ts'`) | confirme (ecart D-4 sur les lignes 68-69) |
| Gate D `npx vitest run src/features/dismissableLayers.guard.test.ts src/lib/accessibility` | les deux chemins existent ; deja verts dans la serie de `temoins_avant.md` §5 | confirme |

## Piste F — precision par film : inventaire FERME (F.0)

### Preuve qu'aucun lecteur de `match_weapon_hit_distance` n'existe

Recherches faites le 2026-11-17 dans `apps/go-api`, `apps/web/src`, `apps/desktop`, `config`, `docs`, `scripts` :
- `grep -rnE "(FROM|JOIN)\s+match_weapon_hit_distance"` (Go) : 4 resultats, tous dans la migration ou ses tests : `migration/steps_shared_weapon_hit_distance.go:145` (la definition de la vue `_latest`), `migration/steps_shared_weapon_hit_distance_test.go:61`, `:81`, `:100` (lectures de la vue par le test du DDL).
- `grep -rn "hit_distance\|HitDistance\|hitDistance"` hors Go : un seul fichier, `config/titles/halo_infinite/mappings/capabilities.toml` (commentaire, ligne 74). Rien dans `apps/web/src`, `apps/go-api/api/openapi.yaml`, `apps/desktop`, `docs`, `scripts`.
- `internal/ops/seed_demo_shared_tables_schema_test.go:37` l'ecrit deja : « ecrite a la cuisson du film, aucun lecteur applicatif ».
- Les autres mentions Go sont des commentaires, la liste de compaction (`migration/compaction_registry.go:100`), l'ordre des steps (`migration/order.go:349`) et deux listes de tables append-only (`sync/no_art_patterns_test.go:184-202`, `sync/append_only_state_guard_test.go:133-153`).
- Lecteurs de `weapon_accuracy` (autre table, a GARDER) : `platform/duckdb/weapon_accuracy_repo.go:146`, `games/halo_5/livesync/events_backfill_surfaces.go:104` et `:138`.

Confirme : aucun lecteur applicatif de `match_weapon_hit_distance`.

### Etat d'execution aujourd'hui (preuve que le chemin ne tourne pas)
- Capacite : `config/titles/halo_infinite/mappings/capabilities.toml:81` `"match.weapon.accuracy" = "not_exposed"` (Halo 5, ligne 52 : `"supported"`) ; `games/halo_infinite/adapter_data.go:198` `games.CapWeaponAccuracy: games.CapNotExposed`.
- Configuration : `(*KillSourceCollector).ConfigureFilmAccuracy` (`killcollector/collector.go:157`) n'a AUCUN appelant hors tests (`hits_capability_test.go:57`, `:112`, `emprise_cablage_partage_test.go` en commentaire) ; `archlint/no_film_reread_test.go:306-309` le consigne (« n'a aucun appelant hors tests »). Sans elle `c.filmDir == nil` et la passe se saute (`hits.go:99-103`, repli `repli_precision_par_arme_passe_sautee`).

### Fichiers a supprimer en entier
1. `go-api/internal/sync/killcollector/hits.go` (296 lignes) : `collectHits`, `buildHitsBatches`, `resolveHitDistanceFunc`, `entreeDeCarteDesTouches`, `hitsScanFailed`, `writeHits`, `publishHitsPass`, les dix constantes `metricHits*` (compteurs expvar `killsource_hits_matchs`, `_accuracy_lignes`, `_distance_lignes`, `_indices_non_resolus`, `_films_absents_disque`, `_cartes_hors_catalogue`, `_matchs_sans_nom_de_carte`, `_carte_non_cablee`, `_erreurs_scan`, `_erreurs_ecriture`). Il appelle `decfilm.ScanFilmWeaponShots`, `ScanFilmWeaponDamages`, `PairWeaponHits`, `WeaponHitPairWindowUS`, `FilmWeaponHitDistance`, `CountFilmChunks` (ce dernier n'a aucun autre appelant de production).
2. `go-api/internal/games/halo_infinite/ingest/weapon_accuracy_film.go` (130 lignes) : `MapWeaponAccuracyFilm`, `assembleFilmSurfaces`, `marshalBuckets`, types `filmAccuracyKey`, `filmAccuracyAgg` ; utilise `decfilm.WeaponHitStats`, `WeaponHitBucketCount`.
3. `go-api/internal/persist/weapon_hit_distance_persister.go` : `WeaponHitDistancePersister`, `NewWeaponHitDistancePersister`, `PersistPass`, `insertAccuracy`, `insertDistance`, `EvaluateHitsGate` (sa seule consommatrice est ce fichier).

### Declarations a retirer de fichiers qui restent
4. `go-api/internal/persist/rows.go:121-143` : `WeaponHitDistanceRow`, `WeaponHitDistanceBatch` (a GARDER : `WeaponAccuracyInsert`, lignes 108-119, ecrit par Halo 5).
5. `go-api/internal/sync/killcollector/collector.go` : champ `filmDir` (ligne 127) et son commentaire (124-126), type `FilmDirResolver` (commentaire 142-147, type ligne 148), methode `ConfigureFilmAccuracy` (150-159), et la phrase du commentaire de `mapNames`/`mapBounds` (avant les lignes 122-123, « DEUX passes les partagent depuis le lot 1.9.4 ») ; `collector_run.go:281-284` (appel `c.collectHits(...)` et son commentaire) ; `porte_de_la_base.go:15-26` (commentaire qui compte `writeHits` et `collectHits` parmi « sept sites » d'ecriture).
6. `go-api/internal/migration/steps_shared_weapon_hit_distance.go` : la table, la vue `_latest` et le step RESTENT (decision D-5) ; `WeaponHitDistanceDecoderRev` (ligne 75) et `WeaponHitsMinShots` (ligne 80) perdent leurs seuls usages hors tests (`hits.go`, mapper, `EvaluateHitsGate` et ses tests) : a trancher (regle 7 « 0 code mort », mais `steps_shared_weapon_hit_distance_test.go:42` lit la premiere et les commentaires du DDL citent les deux).
7. `go-api/internal/filmreplis/sites.go` : constante `fichierHits` (ligne 7), entree `NomDistancesDeToucheDesactivees` (lignes 17-20, deux ancres), l'ancre `premierNomSansArbitrage(ctx, noms, err)` de `NomCartePremierNomResolu` (ligne 23, les deux autres ancres de ce repli sont dans `map_identity.go` et restent), entree `NomPrecisionParArmePasseSautee` (lignes 39-42). VOIR F-1.
8. `go-api/internal/archlint/no_film_reread_test.go` : commentaire de l'allowlist (lignes 299-314) et l'UNIQUE entree `"internal/sync/killcollector/hits.go -> CountFilmChunks"` (ligne 316) de `appelsDEnveloppeAutorises` : l'allowlist est verifiee dans les deux sens (lignes 352-367), une entree morte fait echouer le test ; elle devient vide.
9. `go-api/internal/archlint/film_facade_surface_test.go` : `plafondSurfaceFacade = 182` (ligne 166), `plafondSurfaceReplay = 335` (ligne 462), `plafondsParFamilleFacade` (ligne 473) et le plafond compagnon ; les nouvelles valeurs se lisent en lancant le test apres la suppression (les symboles `decfilm` utilises seulement par les fichiers 1 et 2 ci-dessus sortent de la surface citee hors de `film/`). Valeurs non calculees ici.

### Commentaires et textes a mettre d'accord
10. `config/titles/halo_infinite/mappings/capabilities.toml:63-81` (bloc « Precision par arme : REMISEE le 2026-09-01 ... le code backend (decodeur, mapper killcollector/hits.go, persister, table match_weapon_hit_distance) est conserve ») et lignes 106-108 (« le taux est match.weapon.accuracy, et il est not_exposed ») ; `go-api/internal/games/halo_infinite/adapter_data.go:186-198` (meme paragraphe, cite `killcollector/hits.go` et `collectHits`).
11. `go-api/internal/persist/usage_summary_persister.go:6` (cite `[WeaponHitDistancePersister]`), `persist/rows.go:135-138`, `migration/steps_shared_weapon_hit_distance.go:1-60` (en-tete : producteur, lecteur futur) et `:70-75`, `migration/steps_shared_bomb_stats.go:54`, `migration/steps_shared_player_positions_appendonly.go:27` et `:63`, `migration/steps_shared_usage_summary.go:38` (mentions de la table comme « ecrite a la cuisson »).
12. `go-api/internal/ops/seed_demo_shared_tables_schema_test.go:37` : la chaine « ecrite a la cuisson du film, aucun lecteur applicatif » devient fausse (plus d'ecrivain).
13. `go-api/internal/archlint/no_recomputed_film_context_test.go:98-102` (cite `sync/killcollector/hits.go` comme « unique appelant »), `filmdec_package_vars_test.go:35-37` (cite `weapon_hits.go` du decodeur, hors perimetre), `sync/no_art_patterns_test.go:184-202` et `sync/append_only_state_guard_test.go:133-153` (listes de tables append-only qui nomment `match_weapon_hit_distance` : la table reste, les entrees restent, leurs commentaires parlent d'un ecrivain).

### Tests a supprimer ou adapter
14. A supprimer : `go-api/internal/sync/killcollector/hits_capability_test.go` (tag `integration`), `go-api/internal/games/halo_infinite/ingest/weapon_accuracy_film_test.go`, `go-api/internal/persist/weapon_hit_distance_persister_test.go`.
15. A adapter : `killcollector/hits_carte_par_nom_test.go` (quatre `TestCarteDesTouches*` aux lignes 95, 121, 134, 148 testent `entreeDeCarteDesTouches` : a supprimer ; GARDER `collecteurAvecCarte` (ligne 58), `TestJumellesDuCatalogueSontIndistinguablesParSignature` (72), et revoir `TestLesDeuxPassesPartagentLaMemeResolutionDeCarte` (158) qui compare les deux passes) ; `killcollector/carte_ordre_des_candidats_test.go:92-104` (`TestLesDeuxPassesSuiventLeMemeOrdreDeCandidats` appelle `entreeDeCarteDesTouches`, ligne 99) ; `killcollector/backfill_cout_integration_test.go:283` (`col.collectHits(...)`, mesure de cout) ; `killcollector/emprise_cablage_partage_test.go:40-43` (commentaire).
16. A GARDER : `migration/steps_shared_weapon_hit_distance_test.go` (le DDL reste).

### A garder, partage avec d'autres passes
`killcollector/map_identity.go` (`nomsDeCarteDuMatch` ligne 72-74, `entreeDeCatalogueParNom` ligne 99, `resolveMapBounds` ligne 62, `premierNomSansArbitrage` ligne 165-168) : utilises par la passe des positions ; `resolvePlayerIndices` et `ReplicationChunks` (passe des tirs) ; `replisDeLaPasse` ; `migration.WeaponShotsDecoderRev`.

### Ce qui doit rester pour Halo 5
- Table `weapon_accuracy` et son DDL ; `persist.WeaponAccuracyInsert` (`rows.go:108-119`), son transport (`persist/batch.go:83`, `persist/builder.go:79`) et son INSERT (`persist/shared_persister.go:340`).
- `games/halo_5/ingest/weapon_accuracy.go` (+ `weapon_accuracy_test.go`, tests `TestMapWeaponAccuracy_AggregatesPerPlayerWeapon` ligne 25 et `...EmptyWhenNoFiredDrops` ligne 64), `games/halo_5/livesync/events_backfill_surfaces.go:97-138` (+ test), lecteur `platform/duckdb/weapon_accuracy_repo.go` (+ test), `transfer/transfer_schema_integration_test.go:54-141`.
- `games.CapWeaponAccuracy` (`games/capabilities.go:32`, `games/adapter.go:68-136`, `domain/title/registry.go:91-98`, `:404`, `config_loader.go:46`, `capabilities_parity_*`), `config/titles/halo_5/mappings/capabilities.toml:52` (`supported`).
- La table et la vue `match_weapon_hit_distance[_latest]` (D-5) et leur step de migration.

### Ecart F-1 en detail (a trancher avant F.1)
`go-api/internal/filmreplis/filmreplis.go:61-77` : `Verifier()` exige, pour chaque repli que le registre de la brique decodeur marque `CompleteeParLHote`, au moins un site ou appelant declare dans `filmreplis/sites.go`. Le decodeur v1.4.1 (`github.com/JGtm/halo-infinite-film-decoder@v1.4.1`) declare `repli_distances_de_touche_desactivees` (`internal/facts/fallback/registre_killsource_carte.go:34`, `CompleteeParLHote: true`) et `repli_precision_par_arme_passe_sautee` (`registre_killsource_collecteur.go:58`, `CompleteeParLHote: true`). Leurs seuls sites de l'hote sont dans `hits.go`. Supprimer `hits.go` et ces entrees fait donc echouer `TestLaPartDeLHoteCompleteExactementLeRegistreDeLaBrique` (`filmreplis_test.go:10`) ; conserver les entrees fait echouer les gardes d'ancres (une ancre dans un fichier absent). Sortie possible : un changement du registre du decodeur (retrait des deux replis), donc une version de brique, contraire a D-6 et a « aucune montee de brique ». Etabli par lecture du code ; non execute (aucune suppression n'a ete faite).

## Piste G — rejeu a la demande

| Reference du plan | Constat | Statut |
|---|---|---|
| `internal/replayregen` : ce que la file accepte aujourd'hui | `go-api/internal/replayregen/` (`service.go`, `scan.go`, `count.go`). La file ne traite que des `Candidate{TitleSlug, MatchID, modTime}` produits par `scanTitle` (`scan.go:57-84`) : un candidat est un artefact DEJA rangE (`replayartifact.List`) dont le film est encore dans l'archive (`filmPresent`) ET dont l'en-tete est en retard (`artifactStale`/`isStale`, `scan.go:41-54`), hors matchs refuses par regle (`RefusedByRule`, mode non gere) ou deja refuses (`blocked`). Lancement : `Start()` (`service.go:186`), `ResumeIfPending` (192), `Pause`/`Resume`/`Rescan` ; un film a la fois (`process`, 424 ; `rebuildWithRetry`, 459, reessaie sur `ErrBusy`). `Deps` : `Rebuild(ctx, titleSlug, matchID)`, `FilmPresent`, `HeavyAllowed`, `SyncRunning`... Aucun point d'entree pour un match sans artefact | confirme |
| Branchement | `cmd/server/boot_replayregen.go:28` ; `Rebuild` = `ServiceRegistry.regenerateReplay` (`api/wire/registry_replay_regen.go:87`) qui appelle `RunReplayBuild` (`registry_replay_build.go:39`, « depuis le cache film local ») puis `deriverArtefactRange` ; routes montees seulement sous la capacite `ReplayRegen` : `api/server_desktop.go:132-136`, `handlers/desktop_replay_regen.go` (GET/start/pause/resume) | confirme |
| Verrou `film_decode.lock` | `go-api/internal/filmrun/solo.go:41` et `doc.go:14` ; `Deps.Rebuild` enveloppe `ErrBusy` (`registry_replay_regen.go:101-105`) | confirme |
| Bit « film absent » pose par le persister existant | `MBitFilmAbsent` (`killcollector/registry_flags.go:7`, `collector_run.go:114-115`, `IsFilmGoneErr` de `filmfetch`) ; relu par `platform/duckdb/analysis_probe_repo.go:91` | confirme |
| `RequirePlayerOwnership` | `go-api/internal/api/middleware/require_player_ownership.go:55` ; monte `api/server_apiv1.go:161` (`ownershipMW`). Ecriture (`IsMutatingRequest`) : profil possede ou admin ; lecture : possede ou co-membre ; transparent si l'enforcement est desactive (demo, auth non activee, mode `desktop`). Slug mal forme : 400 `invalid_player_slug` | confirme |
| `auth.RefreshHaloTokensViaStoreFirst` | `go-api/internal/platform/auth/cli_refresh.go:40` : `(ctx, store *MultiUserTokenStore, provider TokenProvider, xuid, gamertag string) (*ExchangeResult, error)` ; `(nil, nil)` seulement quand il n'y a rien a tenter (store/xuid absent, entree introuvable, pas de RT) ; echec reel = erreur | confirme |
| Decouverte 3 (`haloTokensForDrain`) | `go-api/internal/api/wire/registry_catalog_drain.go:151` : boucle sur `r.cfg.LoadPlayers(titleSlug)` et rend les jetons du PREMIER joueur dont `RefreshHaloTokensViaStoreFirst` reussit ; message d'erreur ligne 171 : « ... re-capturer un token » ; appele depuis `registry_build_queue.go:159` (file de construction durable) et `jetonsCatalogue` (ligne 175) | confirme (ecart G-2) |
| `ReplayPage` : etat vide | `web/features/match-replay/ReplayPage.tsx:245-249` : `loadState === 'empty'` rend un bloc `rounded-lg border ... px-3 py-12 text-center` avec `<p>{t.empty}</p>` (cle `empty`, `i18n/i18nContract.ts:89`) ; l'etat d'erreur (251-258) utilise `EmptyStateNotice` avec action. La page rend deja `<ReplayRegenBanner />` (ligne 240, composant de `components/replay-regen/`, sous `useDeploymentCapability('can_regenerate_replays')`) | confirme |
| `MatchHeader.replayLink.tsx`, commentaire perime | `web/features/match-view/MatchHeader.replayLink.tsx:23-29` : « ReplayLink ne rend RIEN quand l'artefact n'existe pas ... En production aucun artefact n'est produit aujourd'hui — donc aucun lien, et rien a expliquer. » (ligne 28-29) : faux, des artefacts sont produits ; le composant (`available`) est appele par `MatchHeader.card.tsx:289` avec `header.replay_available === true` | confirme |
| Ou naissent les capacites de `/bootstrap` | `go-api/internal/config/deployment.go` : champs de `DeploymentCapabilities` (ex. `ReplayRegen`, lignes 165-168) et leur derivation par `capabilitiesOf(cfg)` (lignes 275-319 ; `ReplayRegen: desktop`, ligne 312) ; copie vers le contrat dans `go-api/internal/service/bootstrap_service.go:453` (`CanRegenerateReplays: dc.ReplayRegen`) ; type publie `go-api/internal/domain/bootstrap.go:21-64` (`CapabilityMap`, etiquettes `json:"can_..."`) ; cote web `lib/query/bootstrap.ts` + `useDeploymentCapability`. Profils : `hosted`, `desktop`, `demo`, `dev` ; `ReplayRegen` n'est vraie que pour `desktop` | confirme (ecart G-1) |
| Capacite du titre `film.replay_artifact` | `config/titles/halo_infinite/mappings/capabilities.toml:222` `supported` ; `halo_5` ligne 65 `not_exposed` ; `synthetic_title_b` ligne 43 `supported` ; constante `games.CapFilmReplayArtifact` (`games/capabilities.go:44`) ; `titlesWithReplay` (`registry_replay_regen.go`) la lit par `LoadCapabilityMap` | confirme |
| `port.ReplayGeneration`, `can_generate_replays`, route `replay/generate` | n'existent nulle part (Go, web, openapi) | confirme (a creer) |
| Routes de rejeu existantes sous le meme prefixe | `go-api/internal/api/handlers/replay.go:51-66` : `GET /matches/{match_id}/replay`, `/replay/background`, `/replay/callouts`, `/replay/reapparitions`, `/replay/background.png` ; `match_view_replay_context.go:29` `GET .../replay/context` ; admin : `admin_actions_replay_build.go` (`POST /actions/replay-build/run` et `/enqueue`, refus nomme selon `replaybuild.Placement*`) | confirme |
| Cle de requete | `web/lib/query/keys.ts` existe ; `matchReplayContext(playerSlug, titleSlug, matchId)` (ligne 174) ne porte pas la langue (decouverte 1 du plan) | confirme (hors du perimetre de G) |

## Ce qui n'a pas ete verifie, et pourquoi

- Tous les chiffres tires de `data/` ou du poste (1 455 rejeux, 1,13 Go, rapport gzip 0,137, 39 fichiers par carte, 186 matchs a coup d'envoi du film, 7 matchs sans rejeu dont le film existe, temps de lecture de 4 a 6 ms, `0a247154` sur le poste, 5 405 lignes d'objectif) : la consigne interdit de toucher a `data/` ; le worktree n'a pas de jonction ; non reproductibles ici.
- Les durees de migration par cible (0,6 a 5 s), les copies de gabarit (19 a 57 ms) et l'identite de schema d'une base copiee : non mesurees (hors de 0.2/0.3).
- Tout ce qui vit dans les depots des briques (sprint lu par le decodeur, `shotStats`, `flagReturnS` au schema 125, verdict de score, `replay-equiv`) : seules les copies du cache de modules (decodeur v1.4.1, map-images v1.3.1) ont ete lues pour F-1 et C-4.
- `mapnav-fetch -dry-run` (Thunderhead sans maillage) : demande le reseau et le jeu ; non rejoue.
- F-1 : la consequence de la suppression sur `filmreplis` est etablie par lecture, sans avoir supprime ni execute quoi que ce soit.
- La mesure « au calme » du paquet DuckDB (voir `temoins_avant.md`).
