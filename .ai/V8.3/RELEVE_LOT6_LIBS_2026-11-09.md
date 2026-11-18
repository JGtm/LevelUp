# Relevé du lot 6.1 — tout ce qui nomme `libs/<brique>` dans LevelUp (2026-11-09)

Annexe du plan `.ai/PLAN_FIN_DECOUPLAGE_BRIQUE_FILM_2026-11-02.md`, item 6.1. Relevé en lecture seule sur
`feat/film-lots-5-6` (HEAD `92a413bd95`), par `git grep` (fichiers suivis). Les quatre modules : `libs/gamefiles`,
`libs/himap`, `libs/haloapi`, `libs/film`, chemin `github.com/JGtm/LevelUp/libs/<brique>`, reliés par le `go.work` racine.

## Totaux

| # | Catégorie | Total |
|---|---|---|
| 1 | Build et outillage | 20 fichiers (dont 2 baselines JSONL, 1 266 lignes), ~330 lignes hors baselines |
| 2 | Imports Go de `apps/go-api` | 565 fichiers importeurs (300 prod, 265 test), 606 liens fichier x module |
| 3 | Imports croisés entre briques | `film` -> `gamefiles`/`himap`, `himap` -> `gamefiles` ; `gamefiles` et `haloapi` n'importent aucune brique |
| 4 | Ratchets et tests d'architecture | `archlint` : 73 fichiers sur 210 citent `libs/` (49 en code), 90 avec les helpers ; ~20 tests hors `archlint` |
| 5 | Références hors Go | docs EN 187 lignes, FR 69, `CLAUDE.md` 14, `NOTICE` 2, `.ai/*.md` racine 392, web 4 fichiers |
| 6 | Chemins codés hors imports | ~35 lignes dans 22 fichiers sous `apps/go-api` hors `archlint`, plus 2 helpers (28 appels) |
| 7 | Tags | 4 tags annotés `libs/<brique>/v0.1.0`, tous sur `origin` |

## 1. Build et outillage

| Fichier | Lignes | Nature | Détail |
|---|---|---|---|
| `go.work` | 5-10 | `use` | `./libs/film`, `./libs/gamefiles`, `./libs/haloapi`, `./libs/himap` |
| `apps/go-api/go.mod` | — | ABSENCE | aucun `require` ni `replace` des 4 modules : résolution par `go.work` seul |
| `libs/film/go.mod` | 1, 6, 7 | module, `require` | `gamefiles v0.1.0`, `himap v0.1.0` ; `go.sum` 1-4 |
| `libs/himap/go.mod` | 1, 5 | module, `require` | `gamefiles v0.1.0` ; `go.sum` 1-2 |
| `libs/gamefiles/go.mod`, `libs/haloapi/go.mod` | 1 | module | |
| `Makefile` | 33, 214-215, 243, 257-266, 357, 363, 434-437 | commandes | `libs-test` (boucle `libs/*/go.mod` + autonomie `grep levelup/go-api`), `go-api-test`, `go-api-test-gamefiles`, `go-api-lint`, `gate-push` ; commentaires 206, 227-237, 251-252, 303, 325, 341, 419 |
| `apps/go-api/Makefile` | 83-85 | commande | cible `tidy` : `../../libs/*/`, `grep '^github.com/JGtm/LevelUp/libs/'` |
| `apps/go-api/.air.toml` | 34-37 | chemin | `include_dir` des 4 briques, `exclude_regex '[/\\]libs[/\\].*(testdata\|research)'` |
| `apps/go-api/.golangci.yml` | 222, 233 | motif | `libs/film/(replay\|internal/(facts\|source\|grammar\|profile))/`, `libs/haloapi/halo5/(dto\|dto_carnage\|events_dto)\.go$` |
| `Dockerfile` | 101-106, 131, 76-80 | commande | `COPY go.work go.work.sum`, 4 x `COPY libs/<b>/go.mod go.sum`, `COPY libs/ ./libs/` ; `GOWORK=off` du stage `pack-builder` |
| `.github/workflows/ci.yml` | 153-155, 175-177, 167-169 | motifs | filtres `go` (`libs/**`, `go.work*`), `autre` (négations), `croisee` (`libs/film/replay/**`, `internal/grammar/**`, `killicon/data/**`) |
| `.github/workflows/ci.yml` | 489-586, 532, 641, 705-710, 881-912, 978-1003, 1063, 1291-1292, 1321-1343, + 35 lignes `cache-dependency-path` | commandes | job `go-build` (5 pas `working-directory: libs/<b>`, 4 avec `grep levelup/go-api`), tests film/gamefiles, 4 lint, 4 govulncheck, matrice couverture `[..., film, haloapi]`, baselines `baseline_libs_*.jsonl`, `--lots-du-module libs/film` |
| `.github/workflows/release.yml` 58-61, 199-202 ; `shared-social-gate.yml` 90-93 | chemin | `libs/*/go.sum` |
| `.github/ci/go-shards.txt` | 68, 76 | directive | `module libs/film`, `module libs/haloapi` |
| `.github/dependabot.yml` | 70, 83, 96, 109 | chemin | `directory: "/libs/<b>"` x4 |
| `lefthook.yml` 48, 125 ; `scripts/git-hooks/lefthook/go-vet-cgo.sh` 27 | glob, boucle | `libs/**/*.go`, `for mod in libs/*/go.mod` |
| `scripts/gate-push.ps1` 135-150 ; `scripts/check_test_baseline.sh` 430, 439, 648-649, 672, 681-683, 842-853 | variables | `libs\film`, `BASELINE_FILM`, `FILM_LANE_MODULE='libs/film'`, `FILM_LANE_PATTERN='./...'` |
| `scripts/ci_go_shard.sh` 77-86 | dépendance | lit le chemin de module dans `$mod/go.mod`, `go list` depuis `$REPO_ROOT/$mod` |
| `.gitignore` 171-172, 367-370 ; `.gitleaks.toml` 177 | motifs | `baseline_current_libs_film.jsonl` ; `libs/gamefiles/mapvar/testdata/*.mvar` + 3 exceptions ; allowlist `libs/haloapi/infinite/testdata/leaderboard_sample.html` |
| `.ai/baselines/tests_libs_film.jsonl` (1 050 l.), `tests_libs_haloapi.jsonl` (216 l.) | champ `Package` | chemins de module complets |
| `apps/desktop/THIRD_PARTY_NOTICES.md` 78 | prose | « `libs/*/go.mod` in the source repository » |
| `GOWORK=off` | `Makefile:277`, `ci.yml:764`, `src-tauri/tests/common/mod.rs:41` | | faux sidecar |

## 2. Imports Go dans `apps/go-api`

| Module | Fichiers | dont tests | Paquets importés (fichiers par paquet) |
|---|---|---|---|
| `libs/film` | 410 | 203 | `replay` 239, `decfilm` 139, `filmcache` 72, `finalise` 31, `types` 17, `memguard` 16, `damagetag` 10, `objectiveevent` 10, `playerposition` 9, `filmshell` 7, `killicon` 6, `filmobs` 6, `equipenergy` 3, `medalname` 2, `highlightevent` 2, `identity` 1 |
| `libs/gamefiles` | 68 | 19 | `himodule` 35, `mapvar` 31, `hinavmesh` 2, `ooz` 1 (`cmd/weapon-icons-build/hmod.go`) |
| `libs/haloapi` | 74 | 32 | `infinite` 33, `halo5` 18, `xblauth` 11, racine 10, `haloapitest` 8, `xboxlive` 4 |
| `libs/himap` | 54 | 31 | `himap` 54 |

Répertoires les plus importateurs : `internal/sync` 120, `internal/games` 62, `internal/replaybuild` 52, `internal/service` 44,
`cmd/weapon-sounds` 33, `internal/gamefilescorpus` 27. Auto-référence au chemin de module dans `libs/` : `film` 1 386 fichiers
(2 468 lignes), `haloapi` 27, `gamefiles` 6, `himap` 1.

## 3. Imports croisés entre briques

| De -> vers | Paquets | Prod | Test |
|---|---|---|---|
| `film` -> `gamefiles` | `mapvar` | 6 (`replay/{map_objectives,map_weapon_pads,map_weapon_pads_catalog,objectives_catalog,socles_de_drapeau,zone_attribution}.go`) | 21 |
| `film` -> `himap` | `himap` | 0 | 2 (`internal/profile/loi_largeurs_himap_test.go`, `research/cmd_fermeture/bis2_regions_research_test.go`) |
| `himap` -> `gamefiles` | `himodule` 10 / `hinavmesh` 3 / `mapvar` 2 | 15 | 32 |

Chemin de module en chaîne hors imports : `libs/film/revision/cadre_test.go:21, 29, 64-68` ; `libs/film/revision/ordre_imports_test.go:362` ;
`libs/haloapi/doc.go:13`.

## 4. Ratchets et tests d'architecture

### 4a. `apps/go-api/internal/archlint`

| Test ou fichier | Usage du chemin |
|---|---|
| `briques_imports_test.go` (`TestBriquesImports`, `TestCheminImport`, `TestBriquesImports_TableCoherente`) | table `briques`, `prefixeBriques = "github.com/JGtm/LevelUp/libs/"`, `cheminImport()`, `repertoireDuDepot()` ; `WalkDir` + `parser.ParseFile` de chaque brique |
| `briques_garde_test.go` (`TestChaqueDossierGoDeLibsEstDansLaTable`) | walk de `<racine>/libs` ; `Fatalf` si moins de 5 dossiers |
| `go_source_walk_test.go` | `racinesEspaceDeTravail` (l. 112), `workspaceRoot()`, `surDisque()`, `cheminDeGardeRail()`, `marcherAPIEtBriqueFilm()` |
| `gamefiles_tag_test.go` (4 tests) | balaye `racinesEspaceDeTravail` ; allowlist `libs/himap/himap/deploy_root_test.go` ; lit le `Makefile` racine |
| `dockerfile_go_version_test.go` (`TestGoVersionIdentiqueDansLEspaceDeTravail`, `TestModulesLibsSurveillesParDependabot`) | `Glob(libs/*/go.mod)`, lit `go.work` et `dependabot.yml` ; `Fatalf` sur glob vide |
| `ci_web_reads_go_test.go` | regex `(?:go-api\|\blibs)/…` sur les tests web, couverture par `croisee:` |
| `ci_go_shards_test.go` | `module libs/film` (340-362), `TestCIGoShardsHeavyTestsExist` lit `libs/film/internal/grammar` ; plafond `libs/film/replay/document_chronicle.go: 3720` (l. 324) |
| `film_lane_duckdb_free_test.go` | `go list -test -deps` avec `cmd.Dir = surDisque(racineBriqueFilm)` ; lit `check_test_baseline.sh` |
| `film_racine_depot_unique_test.go` | littéral `"go.work"` / `func racineDepot(` sous `libs/film` ; seul `internal/depottest` autorisé |
| `copies_identiques_test.go` | 3 paires octet à octet `platform/{atomicfile,filelock}`, `analysis/digest` <-> `libs/film/internal/*` |
| `film_layers_deps_test.go` (+ helpers) | table `"libs/film/<paquet>": couche` (~40) ; `racineBriqueFilm = "libs/film"` (l. 28), `importBriqueFilm` (l. 31) |
| `film_facade_surface_test.go` | `cheminDeLaFacade = racineBriqueFilm+"/decfilm/decfilm.go"` ; plafond 166 ; compagnon `replay.X` hors `/film/` |
| `film_file_size_test.go` (34 l.), `film_function_length_test.go` (9), `film_exported_vars_test.go` (16) | clés `libs/film/...`, `libs/himap/himap/...` |
| `no_title_package_in_analysis_test.go` | allowlist `github.com/JGtm/LevelUp/libs/film/{types,filmshell,replay,equipenergy,decfilm,...}` (353-380) |
| `film_tri_total`, `no_unregistered_fallback`, `no_raw_film_bytes_outside_source` | racines `"libs/film"` (+ `internal/source` 222, `filmshell` 225, `research/reapparition` 235) |
| `childproc_single_spawn_test.go:27`, `no_discarded_remove_test.go:32` | racines `{"internal","libs/film","libs/gamefiles","libs/haloapi","libs/himap"}` |
| `pas_de_lecture_sous_ai`, `no_accidental_goos_suffix` (54-60), `no_hardcoded_film_cache_dirs` (50-73) | allowlists par chemin `libs/...` |
| `no_repo_root_walk_test.go` (43, 100, 118, 202) | walk `{"cmd","internal",racineBriqueFilm}` ; allowlist `libs/film/internal/grammar/map_bounds_test.go` |
| `deploy_root_copie_unique_test.go` (32, 83-92) | source canonique `libs/himap/himap/deploy_root.go` |
| `haloapi_sans_client_http_test.go` (`racineBriqueHaloapi`, l. 50), `haloapi_{doer_garde,fermeture_domaine,masque_unique,parametres,sans_etat}_test.go`, `enregistreur_http_unique_test.go` | walk `libs/haloapi` ; `haloapitest/` exigé |
| `film_sans_hote_test.go`, `film_sans_exploitation_test.go` | allowlists `revision/cadre.go`, `revision/porte.go` |
| ~21 autres tests citant `libs/film/...` | `film_pont_identite`, `film_no_slog_in_layers`, `keyframe_walk_proof`, `no_film_reread`, `no_recomputed_film_context`, `no_french_label_literal`, `no_life_cause_divergence`, `no_map_variant_suffix_literal`, `no_raw_film_bytes_extraction`, `no_raw_*_literal`, `filmdec_package_vars`, `fallback_versement`, `i18n_langue_suffixe`, `pose_de_carte_unique`, `record36_grammaire`, `research_tag`, `server_stdout_internal`, `tactical_pure`, `no_stale_fallback_target`, `no_unregistered_fallback_sites`, `no_halowaypoint_literal` |
| 17 fichiers via helpers seulement | `aides_uniques_film`, `film_consultations_comptees`, `film_faits_sans_octets`, `film_garde_de_tampon`, `film_handle_lecture`, `film_lecture`, `film_localisateur_unique`, `killsource_*`, `no_ad_hoc_source_fingerprint`, `no_mojibake`, `todo_expiry`, `unarmed_*` |

Fréquence des helpers : `racineBriqueFilm` 45 fichiers, `surDisque` 31, `cheminDeGardeRail` 24, `workspaceRoot` 21,
`racineBriqueHaloapi` 8, `racinesEspaceDeTravail` 7, `marcherAPIEtBriqueFilm` 6.

### 4b. Hors `archlint`

| Fichier | Usage |
|---|---|
| `cmd/server/licence_gplv3_test.go` (`TestServeurNeLinkePasDeCodeGPL` + 3) | `libsPrefix` (l. 42), `paquetsInterdits = {gamefiles/ooz}`, `dossierDuPaquet()` (l. 121) lit les sources sous `<depot>/libs/...` |
| `internal/replayverite/dependances_test.go:36` | refuse tout import de `libs/film` dans le banc |
| `cmd/anonymise-gamertags/{registre,binaires_couverts,xuid_outil,anonymiser}_test.go` | `racineDuDepot`, `libs/film/testdata/IDENTITES.tsv`, balayage de `libs/film` |
| `internal/filmcorpus/{catalogue_titre,document_shape_jumeaux}_test.go` | goldens `libs/film/replay/testdata/...` |
| `internal/games/halo_infinite/filmprofile/conformite_{empreintes,table21}_test.go` | lisent `libs/film/internal/profile/{registre_empreintes,profile_table}.go` comme texte |
| `internal/games/halo_infinite/haloadapter/aides_test.go:32` | fixtures `libs/haloapi` |
| `internal/replaybuild/*_test.go` (5), `internal/sync/killcollector/cle_inconnue_test.go:124` | `filepath.Join(racine,"libs","film",...,"testdata",...)` |
| `internal/filmreplis/filmreplis.go:28` (PRODUCTION) | `racineDeLaBrique = "libs/film/"` préfixe des `Site.Fichier` |
| `internal/testutil/brique_film_test.go:11` | message `libs/film/go.mod introuvable` |

## 5. Références hors Go

| Fichier | Lignes | Structurant |
|---|---|---|
| `docs/COMMANDS.md` (45) / `docs/FR/COMMANDS.md` (44) | § « Go modules under `libs/` » 1254-1362 ; 1042, 1126, 1251, 1295 |
| `docs/adr/0041` (45), `0043` (45), `0042` (12), `0034` (10), `0045` (2) | D1-D4 chemins et `go.work` ; tableau des dépôts cibles ; amendements |
| `docs/ARCHITECTURE_V6.md` (15) + FR (15) | arborescence `go.work use ...` (273), rôle des briques (275-282) |
| `CLAUDE.md` (14) | tableau des briques (16-17), commandes (222-225), tag `gamefiles` (247-251), ADR (412-428) |
| `docs/WEAPONS.md` (4) + FR, `RUNBOOK_FILM_PROFILES.md` (3), `CHANGELOG` (3+3), `SYNC_GUIDE` (2+2), `ACKNOWLEDGMENTS` (1+1) | chemins de fichiers |
| `NOTICE` (2) | 78-80 : fixtures `libs/gamefiles/mapvar/testdata/...`, `libs/himap/himap/testdata/carte_validee_v1.png` |
| `.ai/` racine | plan 285, `thought_log` 78, `REGISTRE_REPORTS` 17, `HANDOFF_PUBLICATION_BRIQUES` 6, `project_map` 3, `BACKLOG` 3 |
| `apps/web/src/features/match-replay/layers/deltaLayersContract.guard.test.ts` 40-41, `replaySchemaStatusLogic.test.ts` 131, `vehicleCycleTime.test.ts` 65, `sound/replaySoundAssets.guard.test.ts` 102 | lisent des sources de `libs/film/{replay,internal/grammar,killicon/data}` |
| `config/titles/halo_infinite/mappings/vehicle_weapons.toml` 137, 144 | commentaire |
| `README.md`, `LICENSE*`, `CITATION.cff`, `.claude/skills/**`, `FOUNDATIONS_GUIDE` | aucune mention |

## 6. Chemins codés hors imports

| Fichier:ligne | Chemin |
|---|---|
| `internal/testfixtures/cartes.go:16, 19, 23` | `MapvarFixturesDir`, `NavmeshFixturesDir`, `CarteValideeFixturesDir` (~13 utilisateurs, dont `cmd/mapcallouts-build/main.go`, `cmd/mapobj-build/*_test.go`, `internal/sync/replayartifacts/mvar_rattrapage.go`) |
| `internal/testutil/brique_film.go:11` | `CheminBriqueFilm()` : 21 appels dans 17 fichiers (`replaybuild` 4, `sync` 4, `killcollector` 4, `filmarchive`, `replaydiff`, `replayverite`, `testutil` 2) |
| `internal/testutil/replay_chronicle.go:55` | `libs/film/replay/document_chronicle.go` (`ReplayChroniclePath`) |
| `cmd/mapcallouts-build/main.go:84, 134` | cache par défaut EN ÉCRITURE sous `libs/gamefiles/mapvar/testdata` |
| `cmd/refresh_golden_fixture/main.go:35-36` | écrit `libs/film/internal/grammar/testdata/v41_*` |
| `cmd/replay-equiv/identites.go:110`, `main.go:201` | `libs/film/testdata/IDENTITES.tsv`, `libs/film/replay/testdata/equivalence` |
| `internal/filmreplis/filmreplis.go:28` | production, `"libs/film/"` |
| `internal/api/handlers/replay_test.go:164`, `internal/ops/seed_demo_synthetic_sources_test.go:41` | libellés citant `libs/film/damagetag/data/labels.tsv` |
| `libs/**` | aucun `"libs/` codé ; `libs/film/internal/depottest/depottest.go:29` remonte jusqu'au premier `go.work` |

## 7. Tags

| Tag | Commit | Mentions |
|---|---|---|
| `libs/film/v0.1.0` | `6c3fa49063` | `libs/film/README.md:10, 57`, ADR 0043:250 |
| `libs/gamefiles/v0.1.0` | `e2c449b38c` | `libs/gamefiles/README.md:9, 43` |
| `libs/haloapi/v0.1.0` | `e2c449b38c` | `libs/haloapi/README.md:10, 44` |
| `libs/himap/v0.1.0` | `5c37ca56a3` | `libs/himap/README.md:9, 43` |

## Points durs

1. **`apps/go-api/go.mod` sans `require`** : 4 `require` + `go.sum` à ajouter ; 565 fichiers d'imports à réécrire (gofmt
   redistribue les blocs) ; `libs/film` et `libs/himap` portent leurs propres `require` à l'ancien chemin.
2. **Ratchets qui marchent sur le disque de `libs/`** (90 fichiers d'`archlint`, `TestServeurNeLinkePasDeCodeGPL`, plafonds par
   chemin, `copies_identiques`, lots CI) : sans `libs/` dans le dépôt, ils n'ont plus de source à lire ; lecture dans
   `GOMODCACHE` ou déplacement vers les dépôts dédiés à décider test par test.
3. **Cadre historique de `libs/film/revision`** : chemin courant lu par `reflect ... PkgPath()` (pas à écrire) ; constantes
   `briqueFuture` (`cadre_test.go:21`) et `brique` (`ordre_imports_test.go:362`) à mettre à jour ; `casLitteralHorsDeLaBrique`
   doit rester « hors brique » ; risque résiduel : l'ordre gelé des blocs d'imports (`ordresFiges`) peut bouger sous un
   nouveau préfixe -> à PROUVER par rejeu des cinq empreintes.
4. **Fixtures lues par chemin depuis la racine** : `testfixtures` (3 constantes), `testutil.CheminBriqueFilm` (21 appels), 11 tests
   `filepath.Join(racine,"libs",...)`, 2 outils qui ÉCRIVENT sous `libs/`, 4 tests web qui lisent `libs/film/replay`,
   `filmreplis.go:28` en production.
5. **Couplage inverse : `libs/` lit le dépôt hôte** : `depottest.Racine()` (32 fichiers de `libs/film`, `t.Skip` sans `go.work`
   = perte de couverture silencieuse) ; 70 fichiers non-research de `libs/` lisent `data/titles`, `config/titles` ou `static/`
   (`map_quant_bounds.json` 36 usages, `map_weapon_pads.json` 16, `map_objectives.json` 9) ; 8 fichiers citent `apps/web`,
   7 citent `apps/go-api` ; 27 mentionnent `.ai`.
6. **Baselines et lots CI** : 1 050 + 216 lignes JSONL au chemin de module complet ; `FILM_LANE_MODULE` ; matrice couverture,
   `go-shards.txt`, 5 pas `go-build`, 4 lint, 4 govulncheck, 2 affichages de couverture.
7. **`go.work` supprimé** : Dockerfile (6 `COPY`), boucles `libs/*/go.mod` (`libs-test` échouerait sur `cd` avec `set -e`),
   `.air.toml`, `lefthook.yml`, `dependabot.yml`, 47 lignes `go.sum` de caches CI, `dockerfile_go_version_test.go` (`Fatalf`).
8. **Config golangci partagée** : `apps/go-api/.golangci.yml` passé aux briques par `--config` ; ses motifs `libs/film/...`,
   `libs/haloapi/halo5/...` dépendent du chemin relatif ; chaque dépôt dédié aura sa config.
9. **Tags et chemin déprécié** (ADR 0043 D-5, plan D-9) : `v0.1.1` « Deprecated: moved to ... » sur l'ancien chemin, `v0.1.0` dans
   chaque nouveau dépôt ; chemins et URLs `github.com/JGtm/LevelUp` à réécrire dans `go.mod`, README, NOTICE, `doc.go`.
10. **Doc du dépôt** : `CLAUDE.md`, ADR 0041-0043 (amendements datés), `COMMANDS` et `ARCHITECTURE_V6` EN/FR décrivent `libs/`
    comme actuel.
