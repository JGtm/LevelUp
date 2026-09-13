# Revue adversariale finale — finitions v7.5 (E.3)

- **Diff relu** : `git diff e63d89bc7..0f96d42e2` (232 fichiers, 39 commits), worktree de
  LECTURE `LevelUp-wt-finitions-review`, HEAD détaché sur `0f96d42e2`.
- **Contexte frais** : aucune participation à l'écriture des lots. Aucune correction apportée.
- **Hors périmètre** : `e63d89bc7` (bumps Dependabot).
- **Décisions utilisateur respectées sans rediscussion** : périmètre fermé, D12/D13 « le film
  dit », G6 = corriger l'UPDATE, H4 = ne pas supprimer les producteurs d'assets.

## 1. Constats

### R1 — P0 — la vue `match_skill_rank_latest_by_type` n'est créée que par la migration player

- **Où** : lecteur `internal/platform/duckdb/queries_career.go:233` ; seul créateur
  `internal/games/halo_infinite/migrations/steps_player_match_skill_rank.go:318-343` ;
  créateur MANQUANT `internal/sync/schema.go:113-122` (qui pose pourtant la vue sœur
  `match_skill_rank_latest`) ; boucle de migration player du boot limitée à
  `title.DefaultSlug` : `cmd/server/main.go:376` + `:486-509`.
- **Reproduction** (exécutée dans ce worktree, fichier de sonde créé puis supprimé) : ouvrir
  une player DB neuve par `sync.OpenPlayerDB` seul — c'est-à-dire `EnsurePlayerSchema` sans
  `migration.RunForDB(TargetPlayer)` — puis exécuter `duckdb.Q8LUSRHistoryPlayer`.
  Sortie obtenue :

  ```
  vue match_skill_rank_latest            present=1
  vue match_skill_rank_latest_by_type    present=0
  Q8LUSRHistoryPlayer ECHOUE : Catalog Error: Table with name
      match_skill_rank_latest_by_type does not exist!
      Did you mean "match_skill_rank_latest"?
      LINE 10: FROM match_skill_rank_latest_by_type msr
  ```

- **Impact utilisateur** : **toute la page Carrière tombe en erreur**, pas seulement le graphe :
  l'erreur remonte jusqu'à `internal/service/career_service.go:210-213`, qui renvoie
  `domain.CareerPageResponse{}` et l'erreur. Trois populations exposées, toutes atteignables :
  1. **Halo 5** — la boucle de migration player du boot ne parcourt que `title.DefaultSlug`
     (`main.go:376`) ; les player DB `halo_5` ne sont provisionnées que par les CLI hors ligne
     `cmd/h5-enrich` / `cmd/h5-lusr-backfill`. Or l'historique LUSR Halo 5 passe explicitement
     par ce repo : `internal/games/halo_5/adapter_data.go:305-310` — « il est donc lu via le
     FALLBACK repo.GetLUSRHistory côté service ».
  2. **Joueur onboardé entre deux boots** — `main.go:496-498` le dit lui-même : « la création
     de la DB appartient au chemin sync/onboarding, pas au boot ». Ce chemin passe par
     `OpenPlayerDB` → `EnsurePlayerSchema` seul.
  3. **Migration player échouée** — `main.go:503-506` la traite en WARN non fatal (DB
     verrouillée au boot) ; la page reste cassée pour tout le cycle de vie du process.

  Avant C.3 / C.3 bis, `Q8LUSRHistoryPlayer` lisait la table brute, qui existe toujours : la
  panne est **introduite par ce diff**.
- **Correction proposée** : ajouter la vue dans `EnsurePlayerSchema` (`internal/sync/schema.go`,
  juste après le `CREATE OR REPLACE VIEW match_skill_rank_latest` des lignes 113-122) — c'est
  la convention déjà en place pour la vue sœur, et `steps_player_match_skill_rank.go:103` nomme
  `sync/schema.go` comme autorité ; à défaut, étendre la boucle `main.go:486-509` à tous les
  titres du registre.

### Constats P2

| id | fichier:ligne | reproduction | impact | correction |
|---|---|---|---|---|
| R2 | `apps/go-api/cmd/purge_foreign_lusr_chain/main.go:15` | `git grep -n 23046 -- '*.go'` → **1 seule occurrence**, et elle est dans un fichier NEUF de ce diff | L'item B.1.2 est coché `[x]` en affirmant « tous les `.go` sous `apps/go-api` ». Doctrine ART incohérente pour le prochain lecteur. | `#23046` → `#23645` (commentaire seul). |
| R3 | `apps/go-api/internal/scheduler/data_health_psa_index.go:11-15` | Lire les 5 lignes : « L'issue upstream qui décrit EXACTEMENT ce symptôme est duckdb/duckdb#23645 […] **Ce n'est PAS #23645**, que CLAUDE.md cite et qui porte sur une corruption de tas en 1.5.0 » | Phrase auto-contradictoire, produite par la substitution en bloc de B.1.2 sur un texte qui OPPOSAIT les deux issues. La distinction établie par le rapport volet 2 n'est plus reconstructible. | Restaurer la seconde occurrence en `#23046` (c'est bien elle, la corruption de tas). |
| R4 | `docs/adr/0026-append-only-art-eradication.md:12` | `git grep -n "corrompt le heap" docs/adr/0026*` → « l'ART corrompt le heap (bug amont #23645) », suivi du bloc `Failed to delete all rows from index` | Même substitution en bloc : #23645 est l'erreur « Failed to delete… », la corruption de tas est #23046 (`.ai/V7.5/RAPPORT_VOLET2_INDEX_PSA_2026-08-28.md:145`). L'ADR d'autorité attribue le mauvais symptôme à la bonne issue. | Reformuler : « l'ART perd la cohérence de son index (bug amont #23645) ». |
| R5 | `docs/adr/0023-auth-tokens-single-source.md:163-164` et `:180-181` | Lire : « Il ne reste de `queries_auth.go` que `ReadOAuthRefreshToken`, pour la seule migration boot » — le fichier est SUPPRIMÉ par ce diff ; et « allowlist des lecteurs d'env var réduite de ~30 entrées à 1 (la migration boot) » — l'entrée restante est `capturecli.go` | Deux phrases au présent devenues fausses, dans la section que le même diff a par ailleurs mise au passé (« LEVÉE le 2026-09-13 ») : anti-patron « doc inversée » à trois lignes d'écart. | Passer ces deux phrases au passé, ou renvoyer à la section « Clôture de la Phase 5 ». |
| R6 | `apps/web/src/features/match-replay/model/objectiveFamilies.ts:18-19` × `apps/go-api/internal/analysis/objectiveevents/families.go:34-41` | Les deux listes sont indépendantes ; aucun test ne les compare. `TestStatsNommeesPortentLeurFamille` (`families_test.go:15`) ne lit QUE les tables Go ; `objectiveFamilies.test.ts:54` fige la liste TS par un littéral | Le TSDoc affirme que la liste TS est « stable et fermée côté serveur (garde-rail Go : `objectiveevents.TestStatsNommeesPortentLeurFamille`) » — c'est faux. Ajouter une 7e famille d'objectif au décodeur rendra le Go rouge (bien), le TS restera vert, et **le calque cessera silencieusement de dessiner les pulses de cette famille**. | Un ratchet Go qui lit `objectiveFamilies.ts` et le compare aux `ObjectiveType*` (modèle : les ratchets `archlint/*_test.go` qui scannent des sources), ou au minimum corriger le TSDoc. |
| R7 | `apps/go-api/internal/sync/invariants_gate_integration_test.go:329-340` | Le commentaire promet « une chaîne ajoutée au titre sans l'être ici rend le gate rouge, jamais silencieux ». Or `TestSkillChainLiterals_NoDrift` (`skill_chain_crosscheck_test.go:20-37`) ne vérifie que 5 `pair_name` fixes et les 4 chaînes existantes : ajouter une 5e chaîne à `ClassifyLUSRChain` ne le fait pas rougir, et `infiniteLUSRChains` n'a aucun test d'exhaustivité | La liste injectée à `CheckPlayerLUSRChains` peut dériver sans alarme ; l'invariant I14 classerait alors des lignes LÉGITIMES comme « chaîne étrangère ». Aucun impact aujourd'hui (l'invariant n'est branché qu'au gate d'intégration). | Exporter la liste des chaînes depuis `games/halo_infinite/skillchain` et l'injecter au lieu de la recopier ; ou un test d'exhaustivité. |
| R8 | `apps/go-api/cmd/purge_foreign_lusr_chain/purge_test.go:109-152` | Mutation : supprimer `purge.go:153` (`ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT …`) → `go test ./cmd/purge_foreign_lusr_chain/` reste VERT (le seul INSERT du test, L144-146, ne lit jamais `written_at`) | Le test prouve la restauration de la PK, de la séquence et des index, **pas** celle du DEFAUT de `written_at`. Si ce DEFAUT disparaissait, les lignes écrites après une purge porteraient `written_at` NULL — et `ORDER BY written_at DESC` (NULLS FIRST en DESC sous DuckDB) les ferait **gagner** l'arbitrage des deux vues `_latest`. Le code est correct ; c'est la garde qui manque. | Asserter `written_at IS NOT NULL` après l'INSERT du test. |
| R9 | `apps/go-api/internal/ops/seed_demo_corpus.go:346-351` et `seed_demo_anonymize_integration_test.go:80-83` | Le commentaire pose une PRÉCONDITION (« aucun `DemoXUID` égal au `SourceXUID` d'une autre entrée ») et la déclare tenue « PAR CONSTRUCTION ». Le test d'intégration ne l'exerce pas : son roster (`reelA`/`reelB` vs `00…00`/`00…01`) est disjoint d'office. Aucun code ne vérifie la disjonction ; le filtre source (`seed_demo_corpus.go:268`) accepte `^[0-9]{15,16}$`, donc n'exclut pas formellement un xuid à zéros | Si la disjonction se rompait, la passe séquentielle re-remapperait des lignes déjà réécrites → appariement xuid/gamertag faux dans un jeu de données PUBLIÉ. Risque nul aujourd'hui, non gardé demain ; le commentaire délègue à « un futur générateur ». | Assertion de disjonction en tête de `applyUniversalAnonymization` (échec bruyant) + un cas de test qui la déclenche. |
| R10 | `apps/go-api/cmd/repair_msr_index/main.go:56` et `:122` | `grep -n 'filepath.Join(dataRoot, "titles"' apps/go-api/cmd/repair_msr_index/main.go` → L122 reconstruit `titles/{slug}/players/{gt}/stats.duckdb` à la main, avec `titleSlug = "halo_infinite"` en dur (L56) et 4 gamertags en dur (L62), alors que `title.PathResolver.PlayerDBPath` existe et que le fichier importe déjà `halomigrations`. Le ratchet `archlint/no_data_path_join_test.go` ne parcourt que `internal/` | Outil NEUF hors de la source unique des chemins (règle CLAUDE.md « tout via PathResolver ») et mono-titre en dur. Copie n°2 du même littéral (n°1 : `cmd/repair_psa_index/main.go:108`) — à la 3e, la règle des ≤ 2 copies impose un helper. Échec bruyant (`os.Stat`), aucune perte de données. | `titlePkg.NewPathResolver(repoRoot).PlayerDBPath(slug, gt)` et un drapeau `-title`. |
| R11 | `apps/go-api/internal/games/halo_infinite/film/replay/equipment_placements.go` (594 → 628 L) et `apps/go-api/cmd/server/sync_v2_wiring.go:280` (`buildSyncEngineFactoryParityComplete`, 85 → 110 L) | `git show e63d89bc7:<fichier> \| wc -l` vs `wc -l <fichier>` ; pour la fonction, bornes `func`…`}` sur les deux révisions | CLAUDE.md règle 5 : dette gelée, « ne pas l'accroître ». Les Découvertes du plan consignent la croissance de `matchfacts.go` (501 → 504) mais pas ces deux-là, 10× plus grosses. Le lint reste vert (aucun linter de longueur de FICHIER ; `funlen` ne rouvre pas un constat dont la ligne de `func` n'a pas changé sous `--new-from-merge-base`) : la dette croît sans trace. | Consigner les deux en Découvertes, ou extraire le bloc de garde de `buildSyncEngineFactoryParityComplete`. |
| R12 | `.ai/PLAN_FINITIONS_2026-09-13.md:295` | `grep -n "^- \[ \]" .ai/PLAN_FINITIONS_2026-09-13.md` → E.1, E.3, E.4 | E.1 (« Fusions B, C, D dans feat/v75, CI verte ») est **fait** — les merges sont dans le diff relu (`1f4e55ead`, `56e425c8e`, `52828bc05`, `a16e78c28`, `0f96d42e2`) — mais reste `[ ]`. Règle 3 de `plan-execution` : aucun item sans statut à la clôture. | Statuer E.1 `[x]` avec les SHA de merge. |
| R13 | `.ai/thought_log.md` (aucune entrée E.2) vs `.ai/PLAN_FINITIONS_2026-09-13.md:296` | `git diff e63d89bc7..0f96d42e2 -- .ai/thought_log.md \| grep "^+## "` → 9 entrées, aucune pour E.2. Seule trace de l'exécution (réparation d'index + purge **commitée sur 4 player DB réelles**, 1 826 / 2 128 / 942 / 62 lignes retirées, sauvegarde `data/backups/2026-09-13_purge_lusr_h5_arena/`) : la ligne de plan et le commit `be118eaf0` (docs seuls) | Une opération d'écriture sur des données réelles est la plus susceptible d'être rejouée ou auditée ; la règle THOUGHT LOG de CLAUDE.md est explicite (« avant tout commit »). | Une entrée datée E.2 : commandes exactes, comptes avant/après, emplacement de la sauvegarde. |
| R14 | `apps/go-api/internal/platform/duckdb/metadata_repo_assets_list.go:30-41` (commentaire) et `:63-64` (SQL) | Comparer l'ORDER BY avant/après : `ORDER BY m.name_canonical, at_en.name` → `ORDER BY name_canonical, name_en` où `name_canonical = COALESCE(m.name_canonical, '')`. Sous DuckDB, `ASC` trie NULLS LAST ; `COALESCE(…,'')` les transforme en `''`, donc **premiers** | Le commentaire affirme « Le tri d'affichage reste celui d'avant ». Une carte sans `name_canonical` (le filtre `COALESCE(…) NOT LIKE '% - %'` ne l'exclut pas) passe de la fin au début du tiroir des cartes. Cosmétique, mais l'affirmation est fausse. | `ORDER BY NULLIF(name_canonical, '') NULLS LAST, name_en`, ou corriger le commentaire. |
| R15 | `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md:21` et `apps/go-api/internal/games/halo_infinite/film/replay/equipment_placements.go:306` | (a) la ligne de tableau réécrite dit « Classé `dropped` à < 200 ms **de la dernière position** du porteur » : la distance ayant été retirée, le complément correct est « de la fin de vie ». (b) le godoc dit que `originDropMaxDist` survit pour « les ARMES AU SOL, le DRAPEAU et **le CRÂNE** » ; `grep -rn originDropMaxDist` ne rend, hors tests, que les armes au sol (`ground_weapon_bounds.go:160`, `ground_weapon_rules.go:374`, `document_ground_weapon_items.go:53`) et le drapeau (`flag_objects.go:356,417`, `flag_carries_lives.go:331`) — le crâne n'apparaît que dans `oddball_portage_d6_test.go:50`, un fichier de test | Le lot F a précisément pour objet de n'affirmer que ce que la mesure porte ; ces deux imprécisions relâchent ce standard dans les deux documents de référence de l'équipement. | Corriger les deux formulations (« de la fin de vie du porteur » ; « deux autres chaînes de production, plus un contrôle Oddball en test »). |

**Aucun P1.** Les violations de règle écrite trouvées relèvent de la doctrine documentaire
(R2-R5, R15), de dette non consignée (R11), ou de garde-rails manquants sur du code correct
(R6-R9) : aucune ne change un résultat servi aujourd'hui. Le seul défaut qui casse un chemin
utilisateur est R1.

## 2. Ce qui a été vérifié SUR PIÈCES et tient

### Axe 1 — anti-ART (ADR 0019/0026/0030)

- `cmd/purge_foreign_lusr_chain/purge.go` : **aucun DELETE, aucun UPDATE**. Swap CTAS en
  transaction unique ; garde de cardinalité sur les LIGNES **avant** le DROP (L137-141) ET sur
  les VUES **dans** la transaction (`assertViewsRestored`, L164) ; `IS DISTINCT FROM` (L128) —
  la fixture le prouve, la ligne `playlist_group NULL` survit (`purge_test.go:133-137`) ; scan
  forcé `playlist_group || ''` partout (L61, L79, L128) ; `CHECKPOINT` après commit (L173).
- DDL reposée **identique** à `steps_player_match_skill_rank.go` : PK `(id)`,
  `CREATE SEQUENCE IF NOT EXISTS msr_seq` (valeur courante préservée), DEFAUT
  `nextval('msr_seq')`, DEFAUT `written_at` UTC, puis les index CAPTURÉS dans la base
  (`duckdb_indexes()`), jamais recopiés. La fixture du test monte le schéma par
  `migration.RunForDB(TargetPlayer)` : la comparaison se fait contre les migrations réelles.
- C.9 : `captureDependentViews` filtre **sur le SQL** (`lower(sql) LIKE '%match_skill_rank%'`),
  jamais sur un nom ; `TestPurge_KeepsAllDependentViews` et
  `TestCaptureDependentViews_SeesEveryViewOnTheTable` couvrent le cas à DEUX vues.
- `cmd/repair_msr_index` : DDL d'index capturée dans la base (`diag.go:217-234`) ; refus
  explicite si l'index est inconnu (`diag.go:247-250`, `TestRepairIndexes_RefusesUnknownIndex`) ;
  aucune donnée touchée (recomptage `main.go:190-197`, `TestRepairIndexes_LeavesDataUntouched`) ;
  `-ensure-views` rejoue `halomigrations.EnsureMatchSkillRankViews` (`main.go:276-293`,
  `TestEnsureReadViews_RestoresADroppedView`) ; `TestRepairIndexes_LeavesViewsUntouched`.
- `seed_demo_corpus.go` : la forme livrée (`UPDATE <t> SET col = ? [, gt = ?] WHERE col = ?`)
  est exactement celle que le ratchet reconnaît comme sûre — `reUpdateRawSQL` exige au moins un
  `?` dans le littéral (`no_art_patterns_test.go:464-473`, cas témoin L544-552). Le `break` sur
  table absente sort bien de la table courante, pas du roster (vérifié ligne à ligne). Le test
  d'intégration couvre les 3 paires de `match_kill_events` (dont l'assistant), le compte de
  lignes inchangé et le bot hors roster.
- `no_art_patterns_test.go` : **non affaibli** — le seul changement est textuel (`#23046` →
  `#23645` dans deux commentaires). `tablesProtegees`, `criticalMatchTables`, `patternsAtRisk`
  et les deux allowlists sont inchangés.
- `no_raw_rating_reads_test.go` : allowlist **5 → 4** (`queries_career.go` retiré, justification
  datée) ; motif élargi à `_latest(?:_by_type)?`. Vérifié qu'avant l'élargissement le motif ne
  matchait PAS `_latest_by_type` du tout (la frontière de mot échouait devant `_`) : la lecture
  serait passée « par accident ». Le test reste mordant (`len(m[2]) == 0` = lecture brute).

### Axe 2 — LUSR

- C.1 : `engine_backfills.go:211` stampe `e.titleSlug`, qui est bien le titre des bases ouvertes
  juste au-dessus (`e.playerDBPath`, `e.acquireSharedWriter`).
  `openspartan_post_import_service.go:107` stampe `opts.TitleSlug`, qui est le titre de
  `config.PlayerDBPath` / `SharedDBPath` (L86-87). **Le titre stampé est celui de la BASE.**
- C.2 : moteur construit sur `deps.TitleSlug` (`sync_v2_wiring.go:308`) ; `p.TitleSlug == ""`
  accepté ; profil étranger refusé avec `ErrorContext` nommant les 3 valeurs. **Le refus ne
  casse pas le cycle des autres** : `post_sync.go:110-125` est best-effort par joueur
  (`return nil //nolint:nilerr`), l'erreur atterrit dans `PerPlayer[slug].Err` puis
  `mergePostSyncOutcome` (`cycle.go:275-278`). Ratchet `TestSyncV2WiringHasSingleTitleSource`
  vérifié mordant, y compris sur une expression multi-lignes.
- C.3 bis : migration `player_msr_view_latest_by_type_v1` idempotente (`CREATE OR REPLACE
  VIEW`), gardée par `ColumnExists(id)`, partition `(match_id, rating_type)` ordre
  `written_at DESC, id DESC` ; ordonnée **après** `player_msr_view_priority_csr_v1` dans
  `migration/order.go:96-97`. `queries_career_lusr_latest_test.go` monte sa fixture **sur
  fichier par les migrations réelles** et prouve les deux contrats (ligne périmée masquée ;
  match classé → les DEUX séries). Les 4 fixtures qui recopient la DDL
  (`patterns_repo_db_test.go` ×2, `player_repos_test.go`, `repos_extra_test.go`) ajoutent la vue
  en miroir avec renvoi au nom du step — piège connu, consigné, non aggravé.
- C.4 : `CheckPlayerLUSRChains` lit la table BRUTE **volontairement** (une chaîne fausse survit
  sous la ligne gagnante de `_latest`) ; `allowedChains` vide = erreur, jamais un succès
  silencieux ; branché au gate d'intégration (`invariants_gate_integration_test.go:180`). Les
  4 chaînes sont bien celles de `skillchain/classify.go:21-24`, lues sur pièces — sous la
  réserve R7 sur l'absence de ratchet d'exhaustivité.

### Axe 3 — retrait de la migration des jetons (B.2)

- `git grep SPNKR_OAUTH_REFRESH_TOKEN -- apps docs CLAUDE.md` : plus **aucun** lecteur de
  production hors `capturecli.go` (parse de **stdin**, pas d'environnement, allowlisté).
- `git grep oauth_refresh_token / msal_token_cache -- apps` : plus aucun lecteur ; ne restent
  que le champ JSON du store, les garde-rails et le guard d'anonymisation démo.
- `sentinel_test.go` : allowlists des guards 2 et 3 **vides**, guards conservés en ratchet
  anti-résurrection, motifs inchangés (aucun affaiblissement). Guard 1 : 1 entrée.
- `no_legacy_source_used_test.go` **non modifié** — il n'a jamais porté d'exception (le plan dit
  « mis à jour », le journal rectifie ; l'état du code est le bon).
- CLAUDE.md et ADR 0023 disent bien « aucune exception legacy » (sous réserve R5).
- **Baseline** : les entrées retirées sont **exactement 23**, toutes
  `TestMigrateLegacyTokens_*` / `TestEnvRefreshTokenForGamertag_*` (`internal/platform/auth`, 17)
  et `TestMigrateLegacyAuthTokensAtBoot_*` (`cmd/server`, 6) — vérifié par différence des paires
  `(Package, Test)` du JSONL. Rien d'autre n'a bougé. `TestReadOAuthRefreshToken`
  (`queries_auth_test.go`, supprimé aussi) n'était pas dans la baseline : aucune entrée
  manquante. `scripts/check_test_baseline.sh` documente le retrait partiel.

### Axe 4 — PSA (B.1)

- `OpenReadForQuery` (`data_health_psa_index.go:213`), jamais `OpenReadOnly` forcé.
- **Alerte seule** : aucune réparation automatique ; `slog.ErrorContext` nomme la remédiation
  manuelle. Jauge **gelée** si `PSAIndexPlayersUnmeasured > 0`
  (`publishPSAIndexGaugeIfComplete`).
- `PSAIndexDesyncKeys` entre bien dans `WarningsTotal` (`data_health_check.go:209-210`).
- Title-agnostic : boucle sur les titres du registre, prédicat = présence de la table ; chemins
  par `PathResolver` (`PlayersRootDir`, `PlayerDBPath`) ; aucun `slug == "…"`.
- Rectification `#23046 → #23645` : `.ai/archive/`, `.ai/migrations/squashed/`, les rapports
  datés et le thought_log sont **intacts** (`git grep 23046`) — seuls R2/R3/R4 dérogent, dans
  l'autre sens.
- Les 5 reproducteurs sont bien derrière `//go:build psarepro`, comme le test de coût.

### Axe 5 — hygiène (B.3)

- **D9** : la neutralité se lit sur le LABEL (`mapvar.Objective.IsCTFNeutral`),
  `PointObjective.Neutral` propage, les deux lecteurs (`replaybuild.flagSpawnTeam`,
  `BuildMapObjectives`) l'utilisent. Réponse à la question posée : **oui**, un socle d'équipe
  portant un label neutre erroné passerait neutre — c'est le choix explicite (recensement : le
  label est juste 63/63, le `team_index` 62/63), et le défaut symétrique est consigné en
  Découverte D-B2. 3 tests, mutation annoncée.
- **D15** : `DISTINCT ON (map_asset_id)` + requête enveloppante ; la locale est intacte
  (`at_fr` inchangé) ; le tri d'affichage est conservé **sauf** le cas NULL de R14.
  `TestListMapsByTitle_DedupeParAssetIDPasParNom` couvre les homonymes.
- **G2 `[~]`** : vérifié sur pièces — `verifierCouverture`
  (`cmd/replay-corpus-gate/report.go:149-166`) renvoie une erreur nommant chaque témoin absent
  et sa cause, et `finaliser` (`main.go:319-321`) rend **2** dans ce cas. La référence
  `128ae9d15` est la bonne.
- **H1** : `git grep -nw MatchMetrics` → plus aucune occurrence Go ; ne restent que des documents.
- **H4** : seul `cmd/vs-measure` supprimé ; `cmd/weapon-sounds` et `cmd/vehicle-sprite`
  conservés (décision utilisateur). `grep vs-measure` sur `Makefile`, `docs/COMMANDS.md`,
  `docs/FR/COMMANDS.md` et `.ai/project_map.md` → **aucune mention orpheline** (ne subsistent
  que des audits datés).
- **killpos** : les trois faits du statut `[~]` sont exacts — `replay.BuildKillPositions` est
  bien appelé depuis `sync/killcollector/positions.go:347`, l'écriture passe par
  `persist.AddKillPositions` (`persist/builder.go:95`) et `KillPositionPersister` (INSERT pur),
  et `cmd/killpos-build` n'existe pas dans l'arbre.
- **G3 / G4 / G7** : documentation seule, exacte.

### Axe 6 — D10

- `objectiveFamilies.ts` et `objectiveevents/families.go` dérivent bien des `ObjectiveType*`
  (`extract.go:22-27` : flag, zone, hill, skull, vip, bomb — les six, dans le même ordre).
  Garde-rail Go `TestStatsNommeesPortentLeurFamille` mordant côté Go (réserve R6 pour la
  parité TS).
- `flagPulsesRetired` **intact** ; `dropFlags` teste désormais `family === 'flag'`, strictement
  équivalent à l'ancien `startsWith('flag_')`.
- **`Balanced()` tient**, démontré branche par branche : `Available = famille(named)` dans les
  trois chemins (refus roster, fil des morts illisible, chemin nominal), et la somme des
  compteurs vaut exactement `famille(named)` puisque `unnamed = famille(named) − famille(out)`
  et que `IdentifyNamedEventsByRound` (`slotidentity.go:167-180`) ne duplique jamais un
  événement — `unnamed` ne peut donc pas devenir négatif.
- `SchemaVersion` inchangé (54) — aucun champ du document ne bouge.
- **Aucune stat d'objectif exclue par erreur** : `TestStatsNommeesPortentLeurFamille` parcourt
  tout `namedStatSlots` et n'admet que `kills` et `assists` sans préfixe ; toute autre
  statistique sans famille rougirait. Vérifié aussi qu'aucun lecteur de `coverage` n'existe côté
  web (`grep -rn coverage apps/web/src`) — la conclusion « D.2 est Go, pas web » est exacte.
- Tests de mutation présents et explicités des deux côtés (`objectives_test.go:290`,
  `matchfacts_familles_test.go:42`, `objectivesLayer.test.ts:331-390`).

### Axe 7 — F.1

- `equipmentOrigin` ne pose plus qu'une question temporelle (clause de distance retirée,
  `equipment_placements.go:334-337` supprimé). `equipLife.x/y/z` n'est pas devenu mort : lu par
  `gwPadsClass` (`ground_weapon_rules.go:374`).
- **Mutation vérifiée par lecture** : restaurer
  `if dist3(...) >= originDropMaxDist { return OriginDeployed }` fait échouer exactement le cas
  `{"au bon instant, a 5 m" → OriginDropped}` (5 m ≥ 1,5) et laisse intact le contrôle de portée
  `{"loin ET apres la fenetre"}` (la garde temporelle tire d'abord).
- **Mesure rejouable** : `f1_origine_mesure_research_test.go` et
  `f1_origine_verdict_research_test.go` sont gardés par environnement (`t.Skipf` sinon) et
  réimplémentent la règle AVANT (L71) pour comparer — la mesure « 22 poses basculent » est
  rejouable, et les 4 films hors mesure sont nommés avec leur cause.
- Les 6 instruments F.0 (`filmdec/f0_103_*`) sont tous gardés par environnement : `go test ./...`
  ne lit aucun film.
- **R5 amendé par note datée seulement** : le corps du rapport du 2026-09-03 n'est pas réécrit ;
  la note dit ce qui est réfuté et ce qui tient.
- Référence équipement §1 et §6 mises à jour (sous réserve R15).
- Réponse à la question posée : `originDropMaxDist` survit pour **deux** chaînes de production
  (armes au sol, drapeau) et un contrôle Oddball en test — cohérent avec « aucune règle
  arbitraire », chacune posant sa propre question (le LIEU de la fin de vie y est le fait même).

### Axe 8 — transverse

- **Aucun emoji** dans un fichier versionné touché (balayage Unicode sur les 232 fichiers).
- **`slog` structuré** partout dans le code de service ; les `fmt.Printf` sont confinés aux
  `cmd/` de diagnostic (convention du dépôt, exemption `.golangci.yml` `path-except`). Aucune
  erreur avalée introduite : les deux `_ =` du diff sont un `tx.Rollback()` en defer et un
  `db.Close()` en cleanup de test.
- **Aucun `filepath.Join(..., "data", ...)`** introduit ; le seul chemin bâti à la main est R10.
- **Aucune string UI ajoutée** — donc aucune parité FR/EN à vérifier (`git diff` sur `i18n.ts` :
  vide).
- `docs/COMMANDS.md` **et** `docs/FR/COMMANDS.md` : non touchés par ce diff, et cohérents.
- `.ai/thought_log.md` : **une entrée par lot** (B, C ×4, D, F.0, F.1) — réserve R13 pour E.2.
- `funlen` : `purgeForeignChain` (86 L) est dans `cmd/purge_foreign_lusr_chain`, couvert par
  l'exemption datée `.golangci.yml` des CLI hors des six binaires de production — conforme.

## 3. Gates rejoués dans ce worktree

Recette CGO msys64 ucrt64, une commande `go` à la fois, `./internal/himap/...` exclu.

| gate | sortie |
|---|---|
| `go build ./...` | **0** |
| `go vet ./...` | **0** |
| `go test` sur 10 chemins ciblés (purge, repair, objectiveevents, auth, scheduler, duckdb, sync, ops, replaybuild, cmd/server) | **0** — 27 paquets `ok` |
| `go test $(go list ./... \| grep -v /internal/himap)` | **0** — 172 paquets `ok`, 0 `FAIL` |
| `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/migration/... ./internal/platform/duckdb/... ./internal/ops/... ./internal/scheduler/...` | **0** — 22 paquets `ok` (`internal/sync` 167,8 s ; `platform/duckdb` 159,3 s) |
| Sonde de revue (fichier temporaire, créé puis **supprimé** — arbre laissé propre) | **reproduit R1** : `Catalog Error: Table with name match_skill_rank_latest_by_type does not exist!` |

**Non rejoué, et dit** : les gates web (`tsc --noEmit`, `eslint .`, `vitest`, `knip-ratchet`) —
ce worktree de lecture n'a pas de `node_modules`, et `npm ci` y serait une écriture lourde hors
mandat. `make go-api-lint` n'a pas été rejoué non plus (cache golangci global à isoler) ; les
constats de seuil (R11) ont été établis par mesure directe des fichiers et des fonctions.

## 4. Conclusion

**1 P0, 0 P1, 14 P2.** Le corps du travail est solide : les invariants anti-ART sont tenus par
construction ET par test (garde de cardinalité lignes et vues, `IS DISTINCT FROM`, scan forcé,
DDL capturée plutôt que recopiée), la double source de titre est structurellement fermée et pas
seulement comparée, le retrait des jetons legacy est complet avec ses garde-rails durcis à
0 entrée, et les trois lots de mesure (D10, F.0, F.1) rendent des chiffres rejouables avec leurs
exclusions nommées.

Le seul défaut bloquant est R1, et il est de la famille même que ce lot traquait : une vue de
lecture neuve posée dans UN seul des deux endroits qui créent le schéma player. Il ne touche pas
les 4 bases `halo_infinite` suivies (migrées au boot, serveur redémarré en E.2), mais il casse la
page Carrière de Halo 5 et celle de tout joueur onboardé entre deux boots.
