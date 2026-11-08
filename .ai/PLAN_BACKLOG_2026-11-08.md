# Plan — Backlog exécutable du 2026-11-08

**Objectif.** Traiter les entrées du backlog faisables maintenant, clore sur preuve celles qui sont déjà
réglées ou sans objet, et mesurer sans y toucher l'état des défauts du rejeu.

**Critère de succès.** Chaque entrée visée a un statut prouvé ; aucun correctif de rejeu ; `make gate-push` vert ;
`.ai/BACKLOG.md` ne contient plus ce qui est fait et la section « Récemment complété » le référence.

**Hors périmètre.** Application de bureau, dépôts dédiés, publication du décodeur (chantiers en cours). Fins de partie
multi-équipes, stats Élimination et Infection (bloquées faute de donnée).

## Contrat d'exécution

- Skill `plan-execution` (10 règles) ; `delivery-checklist` à chaque clôture d'étape. Ce fichier fait foi pour
  l'avancement. Ordre strict E0 à E8, une étape close avant la suivante. Statuts : `[x]`, `[~]` (référence), `[!]`
  (justification au journal). Aucune case vide à la clôture. Une entrée `.ai/thought_log.md` par étape close.
- Aucun fix hors périmètre : section « Découvertes ». Commentaires de code sans citation `.ai/` ni histoire (règle 17).
- Rouvrir chaque fichier et chaque ligne avant de coder : les numéros de ligne valent pour `origin/release/v8.1.0`
  (`ac8d6c3da6`, 2026-11-08).
- Agents d'exécution : un worktree, jamais `git add -A`, jamais de commit (le pilote relit sur pièces puis commite).

**Branche et lieu.** `chore/backlog-2026-11-08`, depuis `origin/release/v8.1.0`, worktree `../LevelUp-wt-backlog`.
Le go de ce plan vaut pour les commits sur cette branche. Pousser, ouvrir la PR et fusionner : demander.

**Reprise de session.** Relire le contrat, le journal (fin de fichier), `git log --oneline -10` du worktree, reprendre à
la première case vide.

**Environnement des gates Go (PowerShell).**
`$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"; $env:GOCACHE = "$env:LOCALAPPDATA\go-build-backlog1108"`

| Étape | Exécutant | Pourquoi | Effort |
|---|---|---|---|
| E0 préparation | Pilote | Git | rapide |
| E1 micro-tâches | **Haiku** | Remplacements fermés, contrôlés sur pièces | rapide |
| E2 multi-titre | **Sonnet** | Défaut | moyen |
| E3 LUSR | **Sonnet** | Défaut ; tests et relecture ciblée persist/sync | moyen |
| E4 citations | **Sonnet** | Défaut ; 4.2 est une vraie réécriture du seed | moyen |
| E5 rafraîchissement OAuth | **Sonnet** + relecteur Sonnet | Chemin des jetons : deux relectures (ADR 0023), celle-ci et E7.1 | moyen |
| E6 rejeu, mesure seule | **Opus** | Demande de l'utilisateur : sujet sensible | moyen |
| E7 revue, gates finaux | **Opus** (1 relecteur, 1 ronde) | Demande de l'utilisateur ; ronde 2 seulement s'il reste un P0 | moyen |
| E8 nettoyage du backlog | **Haiku**, contrôlé par le pilote | Édition fermée par la table 8.1 | rapide |

## 0. Constats sur pièces (2026-11-08, relus par une revue indépendante)

### Clos sans code (retirés du backlog à E8)

| Entrée | Verdict | Preuve |
|---|---|---|
| Deux `continue` silencieux de `weapon_resolver.go` | Déjà fait | Les quatre lectures de ligne journalisent avant `continue` (:114-117, :120-123, :230-233, :318-321) |
| Démo sans `match_bomb_stats` | Déjà fait | `ops/seed_demo_shared_tables.go:62` (`tableMatchBombStats`) ; test `seed_demo_film_tables_integration_test.go:92` |
| `cmd/levelup` n'installe pas les étapes de migration par titre | Déjà fait | `wireStartupSeams` appelle `titleseams.RegisterAll`, dont la première instruction pose le fournisseur (`games/titleseams/titleseams.go:41`) ; ratchet `archlint/titleseams_wired_test.go` |
| Halo 5 : retirer le 3e maillon du COALESCE de nom | Sans objet | Maillon disparu à la réécriture i18n ; `weapon_label_names` reste le repli des identifiants sans `weapon_key` (`weapon_resolver.go:14-20`). Rien à retirer |
| Fixture `psa_index_repro_msr_fixture_test.go` | Volontaire | Reproduction d'incident (tag `psarepro`) : elle fige les anciennes requêtes à dessein |
| Lecture du précédent LUSR (~6 ms par match) | Non retenu | De l'ordre d'une minute sur un rejeu complet de 10 000 matchs, rare ; aucune gêne mesurée |
| Filtre de chemins de la CI | Accepté | Les chemins sensibles comptent pour les deux piles (`ci.yml:161-164`) ; le reste est rattrapé par la CI complète de la branche de release |
| `RunForDB` dans 3 outils | Non retenu | Outils propres à Infinite (`populate-playlists-catalog`, `seed-assists-model`, `snapshot-world-leaderboard`) ; aucun effet observable |
| Passes `--csr` / `--shared-csr` | Garde en place | Tout titre sans la capability est refusé (`cmd_backfill_csr.go:27-42`), Halo 5 ne la déclare pas ; le commentaire pose la condition de routage. À traiter si un 2e titre réel la déclare |
| Première vérification de participation | Non retenu | Aucune lenteur mesurée ; le cache sert les suivantes |
| Coût de `/tactical/maps` | Non retenu | Section `tactical_maps_fonds`, `logs/general.log` local du 2026-10-26 : 6 mesures de 99 à 1 128 ms, médiane 272 ms. Pas de lot sans gêne constatée |
| Banc Cliffhanger sous 64,7 % | Sans objet ici | Banc derrière le tag `research` (`libs/himap/himap/sddt_research_test.go:1`), hors du corpus `gamefiles`. L'écart reste suivi par `.ai/REGISTRE_REPORTS.md:38`. Le reste de l'entrée corpus : E1.5 |

### Piste d'origine invalidée, traitée autrement (E5)

**Rafraîchissement OAuth, « essayer d'abord la famille enregistrée ».** Le magasin enregistre `azure` pour JGtm et huit
autres comptes, vide pour trois, `xbox_native` pour aucun. Or Azure refuse et le client natif réussit (1 490 lignes au
journal général local, 5 le 2026-11-08). La famille enregistrée vient du ticket XBL, pas du client qui a rafraîchi
(`multi_user_token_store.go:304-312`, `refresh_user_xsts.go:74-95`) : la suivre garderait Azure en premier. E5 retient
donc le client qui a réellement réussi, déjà rendu par `ExchangeRefreshTokenWithRotation`, en mémoire du processus.

### Couvert ailleurs (l'entrée reste jusqu'à la fusion)

`filepath.Join(depot, "data", ...)` des instruments `research` : couvert par l'item 1.4 du plan de la brique film, sur
`refactor/brique-film-publication`, non fusionnée. Sur la release, 9 fichiers restent concernés.

## 1. Décisions

Fermes :
- **F-1.** Aucune modification du code de rejeu ni des artefacts dans ce plan, aucune recuisson ni lot d'artefacts.
- **F-2.** E6 s'exécute sur `origin/release/v8.1.0` : la branche `fix/rejeu-vies-equipement`, non fusionnée, peut
  déplacer ces mesures ; le constat porte sa date.
- **F-3.** Un match exclu disparaît du graphe de carrière.
- **F-4.** Échec de la réconciliation des citations au démarrage (E4.2) : `slog.ErrorContext` puis démarrage poursuivi.
  Contrairement au registre d'armes, une citation manquante ne casse aucune lecture.

- **F-5.** Le client qui a réussi le rafraîchissement se retient en mémoire du processus, jamais dans les fichiers de
  jetons : aucun changement de format sur le chemin de rotation, et au pire un appel perdu après un redémarrage.

Validées par l'utilisateur le 2026-11-08 :
- **D-1. Barre de filtres de la page Citations.** Aujourd'hui sans effet (`handlers/citations.go`, seule `category` est
  lue). **Décision : la retirer.** Une citation est un total de carrière : filtrée, elle change de sens. La
  brancher reste possible (`match_citations_latest` est par match) mais coûte plus.
- **D-2. Citations ajoutées après l'installation.** **Décision : les réconcilier à chaque démarrage**, comme le
  registre d'armes. L'application de bureau n'a pas de ligne de commande : sans cela, une mise à jour n'apporte jamais
  une nouvelle citation.
- **D-3. Discord par titre et catégories réglables.** **Décision : retirer les deux du backlog.** Rien ne les
  demande : Discord est coupé sur ce poste, le relais du coach est optionnel et désactivé par défaut, sans
  interrupteur web, et aucune deuxième liste de catégories n'existe. Le vrai manque, l'absence d'interrupteurs web, reste
  au backlog (8.4).

## 2. Étapes

### E0 — Préparation (pilote)

- [x] 0.1 `git fetch origin` ; `git worktree add ../LevelUp-wt-backlog -b chore/backlog-2026-11-08 origin/release/v8.1.0`.
- [x] 0.2 Dans le worktree, depuis `apps/go-api` : `go build ./...` vert ; depuis `apps/web` : dépendances installées.
- [x] 0.3 D-1 à D-3 inscrites au journal avec la réponse de l'utilisateur.

Gate : `git -C ../LevelUp-wt-backlog branch --show-current` rend `chore/backlog-2026-11-08` ; build vert.

### E1 — Micro-tâches (Haiku, un agent)

- [x] 1.1 `apps/go-api/cmd/levelup/cmd_reset_bitmasks.go` : importer `go_sync "levelup/go-api/internal/sync"`. Supprimer
  les constantes des lignes 196-202. Remplacer **à chaque site** : `skillBitsForReset` par `go_sync.PBitSkill` ;
  `backfillFlagSkillForReset` par `go_sync.BackfillFlags[go_sync.BackfillTypeSkill]` (aucune variable intermédiaire) ;
  `mBitPVEStatsForReset` par `matchflags.MBitPVEStats`. Aucune valeur ne change (`backfill_flags.go:27-30, 59, 135`).
- [x] 1.2 `apps/web/src/features/setup/StepDeviceCode.tsx`, effet de récupération (:130-139) : importer
  `isDeviceFlowSucceeded` depuis `./queries` et sortir de l'effet si `isDeviceFlowSucceeded(status?.status)`, en ajoutant
  `status?.status` aux dépendances. Motif : `XboxLoginPage.tsx:239`. Un retour de focus peut refaire la requête après le
  succès et recevoir un 404. Test dans `StepDeviceCode.test.tsx` : tentative autorisée puis `attempt_not_found`, aucun
  nouveau `start`, rouge avant le correctif.
- [x] 1.3 `apps/go-api/internal/service/citations_service.go:29-33` : remplacer la dégradation muette par
  `return nil, fmt.Errorf("citations: chargement des règles: %w", err)`, retirer le commentaire, ajouter l'import `fmt`.
  Dans `citations_service_test.go:97`, inverser `TestCitationsService_GetCitationsPage_MappingsError_Graceful` : une
  erreur remonte, test renommé `..._MappingsError_Remonte`. Ne pas toucher `GetCommendationsPage`. La page affiche son
  état d'erreur avec « réessayer ».
- [x] 1.4 `apps/web/src/features/citations/CitationCard.tsx:48` : remplacer `truncate` par `line-clamp-2` sur le nom de
  la tuile (motif de `asset-drawer/AssetCard.tsx:74`). Aucune couleur touchée.
- [x] 1.5 Corpus `gamefiles` (exécuté le 2026-11-08 : tout vert sauf `TestRenduForgeVagabond`, qui exige
  `.ai/re_dump/mapvar`, absent de tout checkout). Dans `apps/go-api/internal/gamefilescorpus/` :
  `git mv rendu_forge_gamefiles_test.go rendu_forge_research_test.go` et
  `git mv depot_variantes_gamefiles_test.go depot_variantes_research_test.go` ; en-tête des deux :
  `//go:build gamefiles && research`. Dans `archlint/gamefiles_tag_test.go`, `corpusGamefilesPlancher` passe de 62 à 60,
  avec la raison au commentaire (deux instruments reclassés).
- [x] 1.6 `apps/go-api/cmd/h5-metadata-fetch/main.go`, commentaire d'en-tête : ajouter que cet outil et la table
  `weapon_labels` sont les seuls porteurs des icônes d'armes de Halo 5, donc à conserver.

Gate :
`cd apps/go-api; go vet ./cmd/levelup/ ./internal/service/; go vet -tags=gamefiles,research ./internal/gamefilescorpus/; go vet -tags=gamefiles ./internal/gamefilescorpus/; go test ./cmd/levelup/ ./internal/service/ -run "Citation"; go test ./internal/archlint/ -run "Gamefiles|Research"`
`cd apps/web; npx tsc -b; npx vitest run src/features/citations src/features/setup`
`git grep -n -E "pBit(TeamMMR|EnemyMMR|KillsExp|DeathsExp)|ForReset" -- apps/go-api/cmd/levelup/cmd_reset_bitmasks.go` : 0 ligne.

### E2 — Multi-titre (Sonnet, un agent)

- [x] 2.1 `apps/go-api/internal/scheduler/auto_sync_engine.go:55-57` (`BuildEngine`) : passer
  `s.cfg.SharedProviderForTitle(titleSlug)` à `WithSharedProvider`, garde nil conservée. Titre par défaut inchangé par
  construction (`config/shared_provider_title.go`). Test unitaire : titre par défaut = `cfg.SharedProvider` ; autre titre
  = jamais `cfg.SharedProvider`.
- [x] 2.2 `apps/go-api/internal/api/handlers/sync_handler.go:202-204` (`newEngineFor`) : même remplacement, même test.
- [x] 2.3 `apps/go-api/internal/api/handlers/settings_sessions_recalc.go` : écarter les profils `p.AuthOnly` de la
  boucle de `handlePostRecalculateSessions` (prédicat de `api/wire/notifications_boot.go:90`) ; ils ne sont ni comptés
  ni ouverts, donc leur base n'est plus créée. Test dans `settings_sessions_recalc_job_test.go`.
- [x] 2.4 `apps/go-api/internal/archlint/no_slug_comparison_test.go` : balayer aussi `cmd/` (chemins relatifs préfixés
  `cmd/`). `cmd/levelup/cmd_title.go:42`, seule occurrence attendue, entre dans l'allowlist avec sa raison et la date du
  2026-11-08.

Gate :
`cd apps/go-api; go test ./internal/scheduler/ ./internal/api/handlers/ ./internal/config/ ./internal/archlint/ ./cmd/levelup/`
`go test -tags=integration ./internal/sync/ ./internal/scheduler/ ./internal/api/wire/` (un changement de construction
de moteur peut ne casser que sous ce tag).

### E3 — LUSR (Sonnet, un agent)

- [x] 3.1 Graphe de carrière, `apps/go-api/internal/platform/duckdb/career_repo_lusr.go` : écarter les matchs exclus
  **avant** la phase B, `assembleLUSRResults` et `computeLUSRRatingDeltas` (l'écart se calcule entre matchs retenus). Lire
  l'exclusion par `LoadPlayerMatchEnrichments` (`shared_query_helpers.go`) sur les identifiants de la phase A. Si plus
  aucune ligne après filtre : retour anticipé `nil, nil` (`Placeholders(0)` rend un `IN ()` invalide). Échec de lecture :
  erreur remontée. Test sur base réelle : 3 matchs notés, 1 exclu, 2 points, écart entre les 2 restants.
- [x] 3.2 `apps/go-api/internal/progression/profile/queries.go:~396` : lire d'abord les identifiants exclus
  (`SELECT match_id FROM player_match_enrichment_latest WHERE COALESCE(is_excluded, FALSE)`, modèle
  `sync/skill/exclusion_filter.go:27-30`), puis les passer en paramètres constants `AND match_id NOT IN (?, ...)` dans
  `ranked` ; aucune clause si la liste est vide. Pas de sous-requête (ADR 0036 I2). Test : une composante d'un match exclu
  ne pèse plus.
- [x] 3.3 Écriture tenue : `apps/go-api/internal/sync/skill/skill_v2_options.go`, ajouter une `ShadowOption`
  `WithErrorOnHeldWrites` ; à la fin de `runLUSRV2Shadow` (`skill_v2_shadow.go`, après `logShadowBurstDone` et son test
  d'erreur), si l'option est posée et `s.skippedWriteFailed > 0`, rendre `s.processed` et une erreur enveloppant un
  sentinel exporté `ErrCanonicalWriteHeld`. `RecomputeLUSRCanonicalWithAccess` (`internal/sync/lusr_full_recompute.go`)
  pose l'option. Le post-sync ne la pose pas : son retry au cycle suivant reste identique. Appelants qui recevront
  désormais l'erreur, à relire : `MatchExclusionService` (`:79-83`, déclenche `RecomputePending`),
  `openspartan_post_import_service.go:175`, `engine_backfills.go:191`, `cmd/h5-lusr-backfill`, `cmd/recompute_perfnote`.
  Test dans un **nouveau** fichier `skill_v2_shadow_held_writes_test.go` (`skill_v2_shadow_test.go` dépasse le seuil),
  injection d'échec calquée sur le test existant de la rétention : avec l'option erreur, sans l'option `nil`.

Gate :
`cd apps/go-api; go test ./internal/platform/duckdb/ ./internal/progression/profile/ ./internal/service/ ./internal/archlint/`
`go test -tags=integration ./internal/sync/...`

### E4 — Citations (Sonnet, un agent)

- [x] 4.1 Selon D-1, retirer la barre. Dans `apps/web/` : `src/features/citations/UnifiedCitationsPage.tsx`
  (`useLocalFilterBar`, `bar`, `committedHash`), `CitationsView.tsx` (prop `filterBar`), `queries.ts` (`request`,
  `filterHash`), `src/lib/query/keys.ts` (`queryKeys.citations`) et ses tests `keys.locale.test.ts:228`,
  `keys.title-slug.guard.test.ts:105`, `src/features/home/HomeCitationsNearCompletion.tsx:60-64` et son test,
  `src/lib/api/types.ts` (`CitationsQueryRequest`), `src/lib/citations/{types.ts:53,normalize.ts:70,92,normalize.test.ts}`
  (`hasFilters`), commentaire de `src/features/_shared/useLocalFilterBar.tsx:8`, clés `citations.filters.*` de
  `src/lib/i18n/manifests/citations.toml` puis `node scripts/build_i18n_manifests.mjs`, tests 3a et 3b de
  `e2e/filter-stores-split.spec.ts:138-182`. Ne pas toucher la source Halo 5 ni `useLocalFilterBar`. Si D-1 change :
  `[!]` et retour au pilote.
- [x] 4.2 Selon D-2. Dans `apps/go-api/internal/ops/` : extraire de `SeedCitationMappings` un cœur
  `seedCitationMappingsOn(ctx, db)` sans `sql.Open`, exposé par `ReconcileCitationMappings(ctx, db)`. Aujourd'hui le seed
  fait un `UPDATE` inconditionnel (`seed.go:416-454`) et `UpsertAssetTexts` idem : le cœur lit les 13 colonnes et les textes
  existants, compare, et **n'écrit que ce qui diffère** (index ART). `SeedResult` gagne un compteur `Updated`.
  `cmd/levelup` garde `SeedCitationMappings`. Appel dans `apps/go-api/cmd/server/boot_migrations.go`, `migrateMetadataDB`,
  après `ReconcileNameLabels`, même handle, titre par défaut seulement, politique F-4. Tests d'intégration : nouvelle
  citation insérée sur une base existante ; deuxième passage `Inserted == 0` et `Updated == 0` ; erreur de réconciliation
  n'empêche pas `migrateMetadataDB` de réussir.

Gate :
`cd apps/go-api; go test -tags=integration ./internal/ops/ ./cmd/server/ ./internal/archlint/`
`cd apps/web; npx tsc -b; npx eslint src/features/citations src/features/home src/lib/citations src/lib/query; npx vitest run src/features/citations src/features/home src/lib/citations src/lib/query src/lib/i18n`
`git grep -n -E "citations\.filters|hasFilters" -- apps/web/src apps/web/e2e` : 0 ligne, fichiers générés compris.

### E5 — Rafraîchissement OAuth (Sonnet, un agent, puis un relecteur Sonnet)

Résultat visible : plus d'appel refusé ni de ligne de journal à chaque rafraîchissement d'un compte du client natif.
Comportement inchangé pour les comptes Azure et au premier rafraîchissement après un démarrage.

- [x] 5.1 Brique, `libs/haloapi/xblauth/oauth_refresh.go` : ajouter à `OAuth` le champ `NatifDabord bool`. Quand il est vrai,
  `RefreshWithRotation` essaie `postMSATokenExchange` en premier. Succès : `Famille = FamilleXboxNative`. Échec de classe
  `AuthErrorRevoked` ou Config : dérouler le chemin Azure actuel, sans nouveau repli natif, et propager son erreur. **Échec
  transitoire (réseau, 429, 5xx) : le rendre tel quel, sans essayer Azure**, dont le refus serait lu à tort comme un jeton révoqué.
  Faux : code actuel strictement inchangé. Tests `httptest` dans `oauth_refresh_test.go` : ordre natif puis arrêt ; natif
  révoqué puis Azure ; natif en 500 sans appel Azure ; champ à faux, ordre Azure puis natif.
- [x] 5.2 Application, nouveau fichier `apps/go-api/internal/platform/auth/refresh_client_memory.go` : table
  xuid -> dernier client ayant réussi, protégée par un verrou, plus un porteur de contexte
  `WithRefreshXUID(ctx, xuid)`. `ExchangeRefreshTokenWithRotation` (`xbl_oauth.go`) lit le xuid du contexte, pose
  `NatifDabord` si le dernier client connu est `TokenFamilyXboxNative`, et retient `out.Famille` après un succès. xuid
  absent : aucun changement. La brique reste sans état (ADR 0042).
- [x] 5.3 Poser `WithRefreshXUID` aux sites de production : `refresh_user_xsts.go:99`, `refresh_loop.go:182`,
  `sisu_provider.go:188` via l'appelant `pool/resolver.go:384`. Aucun jeton dans un journal ni dans la table.
- [x] 5.4 Test applicatif : deux rafraîchissements successifs du même xuid sur un faux serveur, le second ne sollicite
  pas Azure ; un redémarrage (table vide) reprend l'ordre actuel ; un RT redevenu Azure bascule et la table suit.
- [x] 5.5 Relecteur Sonnet (`adversarial-review`, axe rotation et classification des erreurs, ADR 0023) ; constats statués.

Gate :
`cd libs/haloapi; go test ./xblauth/`
`cd apps/go-api; go test ./internal/platform/auth/... ./internal/archlint/`
Après fusion seulement, à contrôler sur le poste : `invalid_grant côté Azure` ne se répète plus pour un même compte.

### E6 — Rejeu : mesure seule, aucune modification (F-1) (Opus, un agent)

Toute sortie hors du worktree, dans le scratchpad de la session. `<PRINCIPAL>` = checkout portant `data/cache`. Si un
défaut est encore présent, Opus peut en lire la cause dans le code, sans rien modifier, et la consigne au journal.

**6a**, lecture seule de `<PRINCIPAL>/data/cache/replays/halo_infinite/*.json` hors `*.derived.json`, un fichier à la fois :
- [ ] 6a.1 Par film : nombre de `(.tracks // [])[] | select(.xuid == null or .xuid == "")` lorsque `(.roster // [])` ne
  déclare aucune entrée sans `xuid`. Un `xuid` absent ne suffit pas : les bots n'en ont pas. Relevé pilote non filtré
  du 2026-11-08 : 2 463 pistes sur 491 rejeux, dans les trois plus gros cas des films à bots déclarés ; ce chiffre ne
  mesure donc pas les vies sans nom.
- [ ] 6a.2 Films dont `coverage.fallbacks` contient `repli_chassis_vehicule_marqueur_neutre`, somme des `hits`.

**6b** dans le worktree, depuis `libs/film` :
`$env:L1_DEPOT = "<PRINCIPAL>"; go test -tags research -count=1 -v -run '^<Test>$' ./replay/`. Une exécution par
instrument, aucune sonde nouvelle (ne compile pas : `[!]`). **Validité : la sortie doit citer le film cible ou plus de 8
films du parc. Sinon « non mesurable », jamais « réglé »** : sans dépôt, ces tests passent sur leurs seules fixtures.
- [ ] 6b.1 Vies sans nom : 5da6fd30 relu par `jq` (6 vies au 2026-11-08). 72c48587 et 7aa132d1 sans artefact local :
  `[!]` non mesurables, pas de cuisson.
- [ ] 6b.2 Second corps à 83 m (4f77afc1) : `death_context_lieu_unique_research_test.go`.
- [ ] 6b.3 Entame sans position (`kill_positions_e81bc_mesure_research_test.go`, `KP_KILLS` et `KP_BASE`) et parent non
  véhicule, 5676a9ba (`kill_positions_regles_research_test.go`, `KP_KILLS`) : ces instruments exigent des exports d'une
  **copie à froid** de la base partagée. Le pilote demande à l'utilisateur d'arrêter le serveur ; copie sous le scratchpad, exports
  TSV par `cmd/diag_q` avec le format donné en tête de chaque instrument. Serveur non arrêté : `[!]` non mesuré.

Sortie : une ligne par défaut, « réglé » / « encore présent, sur N films, visible comme ... » / « non mesurable ». Point
d'étape en termes visibles. Tout correctif éventuel attend l'accord de l'utilisateur, dans un autre plan.

Gate : tableau rempli au journal ; `git -C ../LevelUp-wt-backlog status --short` n'affiche aucun fichier issu de E6.

### E7 — Relecture adversariale finale et gates finaux

- [x] 7.1 Skill `adversarial-review` sur tout le diff de la branche, **un relecteur Opus**, axes persist/sync, exclusions,
  seed au démarrage, chemin des jetons. Une ronde ; ronde 2 seulement s'il reste un P0. Constats statués au journal.
- [ ] 7.2 `delivery-checklist`.
- [ ] 7.3 `make generate-types` puis aucun écart non voulu ; `make check-types` ; `make test-web` ; `make gate-push` ;
  `make go-api-test-gamefiles` (Halo installé, environ 10 min).
- [ ] 7.4 Contrôle visuel de l'utilisateur, à proposer : tuiles de la page Citations (noms sur deux lignes, page sans
  barre de filtres) et exclusion d'un match (le point disparaît du graphe de carrière).

Gate : `make gate-push` et le corpus verts (sorties au journal).

### E8 — Nettoyage et mise en référence du backlog (Haiku, contrôlé par le pilote)

Dans `.ai/BACKLOG.md`, sur la branche du plan. La branche film modifie aussi la ligne 68 de ce fichier : le pilote
résout ce conflit au rebase. Les deux lignes non commitées du dossier principal (entrée « Décodeur de film ») ne sont pas
reprises ici.

- [x] 8.1 Section « [dette/backlog-2026-10-16] », puce par puce, selon le statut final de son item (un `[!]` garde sa puce
  avec sa raison) :

| Groupe | Puce | Action si close |
|---|---|---|
| LUSR | Match exclu dans le graphe | Retirer (3.1, 3.2) |
| LUSR | `skippedWriteFailed` | Retirer (3.3) |
| LUSR | Lecture du précédent | Retirer (§0) |
| LUSR | Fixture `psa_index_repro` | Retirer (§0) |
| Multi-titre | Provider d'Infinite | Retirer (2.1, 2.2) |
| Multi-titre | `--csr` / `--shared-csr` | Retirer (§0) |
| Multi-titre | Ratchet sans `cmd/` | Retirer (2.4) |
| Multi-titre | `cmd/levelup` et étapes par titre | Retirer (§0) |
| Multi-titre | `RunForDB` | Retirer (§0) |
| Multi-titre | Sessions et `auth_only` | Retirer (2.3) |
| CI | Filtre de chemins | Retirer (§0) |
| CI | Corpus `gamefiles` | Retirer (1.5, §0) |
| Citations | `LoadCitationMappings` | Retirer (1.3) |
| Citations | Barre de filtres | Retirer (4.1) |
| Citations | Noms tronqués | Retirer (1.4) |
| Citations | Seed jamais rejoué | Retirer (4.2) |
| Divers | `StepDeviceCode` | Retirer (1.2) |
| Divers | Première vérification de participation | Retirer (§0) |
| Divers | `cmd_reset_bitmasks` | Retirer (1.1) |
| Divers | `continue` de `weapon_resolver` | Retirer (§0) |
| Divers | `match_bomb_stats` | Retirer (§0) |
| Divers | `/tactical/maps` | Retirer (§0) |
| Divers | Instruments `research` | Retirer (branche film fusionnée dans la release, 5842e16cad) |

- [x] 8.2 Entrée « [ops/h5] weapon_labels » : la supprimer (rien à retirer, conservation portée par 1.6).
- [x] 8.3 Entrée « [auth] Rafraîchissement OAuth » : la supprimer si E5 est close ; marquer traitée la ligne D3-L1-f de
  `.ai/REGISTRE_REPORTS.md`, en renvoyant à ce plan.
- [x] 8.4 Selon D-3, retirer « overlay Discord par titre » et « exposition des catégories forwardées » de la section
  Coach, garder le funnel, et ajouter : « Réglages Discord sans interrupteur web : coach, rejeu, reconnexion, disque ».
- [ ] 8.5 (Pilote, pas Haiku) Entrées rejeu : y écrire le résultat daté d'E6 ; ne retirer que ce qui est mesuré réglé.
- [x] 8.6 « Récemment complété » : une ligne datée (branche, plan, faits, clos sur preuve, renvoi au §0).
- [ ] 8.7 `.ai/thought_log.md` : entrée finale. Plan déplacé sous `.ai/V8/` une fois la PR fusionnée.
- [ ] 8.8 `powershell -File scripts/disk-hygiene.ps1 -Lot backlog1108 -Apply`, worktree retiré après fusion.

Gate : `git grep -n -E "skippedWriteFailed|psa_index_repro|--shared-csr|no_slug_comparison|RunForDB|filtre de chemins|LoadCitationMappings|StepDeviceCode|cmd_reset_bitmasks|weapon_resolver|match_bomb_stats|tactical/maps" -- .ai/BACKLOG.md | grep -v ":| 20"` ne rend que
les puces gardées en `[!]` ; relecture du diff par le pilote.

## 3. Découvertes

- Pilote, 2026-11-08 : `newEngineFor` de `sync_handler.go` n'est emprunté que si `engineBuilder` n'est pas câblé
  (commentaire `:207`) ; s'il est mort en production, la règle 7 demande son retrait. Non traité.
- Pilote, 2026-11-08 : seul le relais `replay` lit la configuration Discord par titre. Aucun réglage web pour
  `discord_notify_coach`, `discord_notify_replay`, `discord_notify_reauth`, `discord_notify_disk`. Reversé au backlog (8.4).
- Revue, 2026-11-08 : `GetCommendationsPage` (`citations_service.go:75-79`) a la même dégradation muette que 1.3.
  Non traité.
- Revue, 2026-11-08 : `cmd/server/sync_v2_wiring.go:167, 181-182, 318` passent `deps.Cfg.SharedProvider` au moteur de
  `deps.TitleSlug` ; ce pipeline est mono-titre (Infinite). Un garde-rail contre le `WithSharedProvider` nu resterait à
  poser le jour où il cessera de l'être. Non traité.

- E1 (Haiku), 2026-11-08 : `previewReset` de `cmd_reset_bitmasks.go` ignore ses erreurs de lecture (`_ =`, lignes 75, 84, 96, 107), hors périmètre. Non traité.

- E3 (Sonnet), 2026-11-08 : `loadLUSRComponentsBreakdown` avale l'erreur de sa requête principale (dégradation annoncée) ; doc-comment de `GetLUSRHistory` placé au-dessus du type `lusrPlayerRow` ; `LUSRRecomputed` reste à 0 sur erreur partielle à l'import OpenSpartan ; `cmd/recompute_perfnote` s'arrête à la première erreur. Non traités.

- E4 (Sonnet), 2026-11-08 : clés de manifest déjà inutilisées (`citations.section.distribution`, `section.medals_summary`, `distribution.*`, `commendations.*`, `medals.*`, `kpi.*`) ; `POST /pages/citations` accepte encore un `category` qu'aucun client n'envoie. Non traités.

- E5 (Sonnet), 2026-11-08 : `RefreshHaloTokensViaStoreFirst` (`registry_catalog_drain.go`, `registry_appearance_diag.go`) garde l'ordre Azure d'abord, chemins rares. `ResolveMSAccessTokenStoreFirst` pose le xuid aussi pour ses appelants CLI, sans effet observable. Non traités.

- Revue Opus, 2026-11-08 : `cmd/diag_perfsim/load.go` garde une copie déclarée de `LoadExcludedMatchIDs` (2e copie, seuil de la règle 6 non atteint) ; cette lecture passe par `SQLDb()` sans reprise sur handle invalidé. Non traités.

## 4. Journal

- 2026-11-08 : plan rédigé, sources vérifiées sur `origin/release/v8.1.0`. Revue indépendante (Sonnet, une ronde) : 3 P0,
  11 P1 intégrés, P0 et points clés rejoués sur pièces par le pilote.
- 2026-11-08 : réponses de l'utilisateur : D-1 retirer la barre, D-2 réconcilier au démarrage, D-3 retirer les deux réglages.
  Demandes : traiter aussi le rafraîchissement OAuth (E5 ajoutée), relecture finale par Opus, rejeu confié à Opus.
  En attente du go.
- 2026-11-08 E1 close : six items faits, gate vert rejoué par l'exécutant, diff relu par le pilote. Relance du flux prouvée rouge (4 démarrages) puis verte (1).
- 2026-11-08 E2 close : quatre items faits, tests rouges puis verts, gate vert (intégration sous `-p 1`), diff relu par le pilote. Accesseur `SharedProviderForTest` ajouté pour prouver quel provider sert le moteur.
- 2026-11-08 E3 close : trois items faits, tests rouges puis verts, gate vert (intégration `./internal/sync/...` sous `-p 1`), diff relu par le pilote. Chemin réel de l'exclusion : `MatchRecomputer.recompute`, l'échec alimente `RecomputePending`.
- 2026-11-08 E4 close : barre de filtres retirée, réconciliation des citations au démarrage (écrit seulement ce qui diffère, échec non bloquant), tests rouges puis verts par mutation, gate vert ; e2e non lancé. Pilote : `docs/COMMENDATIONS.md` aligné (guide EN seul) ; une édition manuelle de `citation_mappings.enabled` est désormais écrasée au démarrage, documenté.
- 2026-11-08 coordination `levelup-cf` : fusion en 2e position, vers 21 h, rebase à ce tour. E6 attend la fin de la recuisson en cours (vers 20 h 30) et mesure donc les artefacts recuits : F-2 s'interprète ainsi.
- 2026-11-08 E5 close : rafraîchissement OAuth par le dernier client gagnant, en mémoire. Relecture Sonnet : 1 constat recevable (P1), 11 conditions tenues. Compte reconnecté en web + natif en erreur de configuration = échec en boucle ; corrigé : seul un échec transitoire du natif est rendu sans Azure, test rouge puis vert. Gate vert.
- 2026-11-08 ordre : E6 dépend de la fin de la recuisson (dépendance externe, `levelup-cf`) ; E7.1, sans lien avec elle, passe devant E6.
- 2026-11-08 E7.1 close : relecture Opus sur toute la branche, 4 constats recevables, tous corrigés avec test rouge puis vert (cache du graphe après exclusion, copie de la lecture des exclus, plancher daté, chemins d'échec), 25 conditions tenues. Aucun P0 : pas de ronde 2.
- 2026-11-08 E8 : section dette, entrées `weapon_labels` et OAuth retirées, section Coach réduite, référence ajoutée, registre et CHANGELOG 8.1.0 (EN et FR) à jour. Reste 8.5, 8.7, 8.8 après E6.
- 2026-11-08 CI de branche après rebase sur `4726e9bbe4` : 2 tests rouges d'intégration, E3 croisait la révision des KPI (#158), qui tolère un recalcul partiel. Une écriture tenue (`ErrCanonicalWriteHeld`) est classée recalcul partiel par `MatchRecomputer` : exclusion non refusée, `RecomputePending` vrai, révision non rangée.
