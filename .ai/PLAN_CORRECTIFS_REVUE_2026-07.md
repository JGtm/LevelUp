# PLAN_CORRECTIFS_REVUE_2026-07 — solde des findings de la revue 10 jours

> Référence findings : `.ai/REVUE_CODE_2026-07-17.md` (revue adversariale
> 24fc02f2f..HEAD — 31 confirmés, verdicts avec citations).
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la
> fois, AUCUN report d'item exécutable, statut obligatoire par item, vérifier
> sur pièces avant de coder ET avant de cocher, zéro fix hors périmètre —
> consigner en Découvertes).

## Objectif et critère de succès

Solder TOUS les findings confirmés de la revue (dysfonctionnels ET robustesse —
décision utilisateur 2026-07-17 : rien n'est exclu). Succès = chaque item
statué `[x]` / `[~]` (couvert ailleurs, référence) / `[!]` (justification
écrite) ; gates verts par lot ; baseline tests intacte ; aucune nouvelle issue
lint (`--new-from-merge-base=main`).

- **Branche** : `fix/revue-2026-07-correctifs` (depuis main local, 1 branche,
  N commits — 1 commit par lot minimum).
- **Worktree** : `c:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-revue-correctifs`.
- **Effort estimé** : lots A/E rapides, B/D moyens, C lourd, F moyen —
  ~5-6 jours effectifs au total.
- **Exécution** : agents Opus séquentiels (JAMAIS deux builds Go en parallèle —
  corruption cache Windows). Le superviseur ne code pas : git/merges/CI/journal.

## Décisions tranchées AVANT exécution (ne pas ré-arbitrer en cours de route)

- **D1 — Sémantique crons** : `ReportCronRun` reçoit l'erreur agrégée réelle du
  cycle ; un cycle partiellement échoué = échec avec cause. Appliqué aux 5 crons
  actuellement « toujours verts » (world_leaderboard, spartan_customization,
  data_health_check, catalog_refresh, asset_name_sweep).
- **D2 — Emojis Discord** : le payload Discord est du contenu produit →
  exemption datée en commentaire dans `internal/notify/discord.go` ; la sortie
  CLI (`cmd_data.go:233`) est purgée.
- **D3 — Filets squash (M2)** : réparation par INTROSPECTION du schéma (jamais
  par sentinelle), auto-guérison au boot, idempotente (doctrine convergent
  sync). Pas de cmd manuel. Fixtures de DBs aux états intermédiaires exigées.
- **D4 — Identité** : lot A = fix ponctuel season pass + garde-rail ratchet
  (aucune 5e occurrence possible pendant le chantier) ; lot B = refactor
  « sujet en paramètre explicite » (fix profond).
- **D5 — Flush notifications** : `visibilitychange`/`pagehide` + envoi
  `keepalive` (pas `beforeunload` seul).
- **D6 — Recovery par construction (F13)** : ON LE FAIT, périmètre = handle de
  lecture player-DB n'exposant que les variantes `*Recovered`.
- **D7 — Doctrine tokens vs sujet (validée utilisateur 2026-07-17)** :
  emprunter un token VALIDE du pool pour authentifier un fetch est un design
  assumé (le pool existe pour ça) — ne jamais « corriger » l'emprunt lui-même.
  Les invariants sont : (a) le SUJET de la requête (quel joueur) vient de la
  PAGE, jamais du contexte ambiant ; (b) ne JAMAIS persister des données sous
  un xuid différent de celui pour lequel l'API les a servies ; (c) le
  budget/quota se débite au PORTEUR réel du token (B1). Cas particulier des
  endpoints ownership-scoped upstream (BP/défis : fetchables uniquement pour
  soi) : viewer ≠ sujet → servir le persisté du sujet, pas de fetch-écriture.

## Gates (commandes exactes)

- Go ciblé : `cd apps/go-api && go test ./internal/<paquets touchés>/...`
- Go intégration (lots C, F7, et clôture) :
  `cd apps/go-api && go test -tags=integration -p 1 ./internal/<filtre ANCRÉ>/...`
  (jamais de filtre non ancré — incident LOT B).
- Lint : `golangci-lint run --timeout 5m --new-from-merge-base=main` (v2.12.2).
- Web : `make check-types` ; vitest ciblé puis `make test-web` (hors sandbox).
- Clôture chantier : suite complète `go test -tags=integration -p 1 ./...` +
  `scripts/check_test_baseline.sh tests` (~13 min) + skill `delivery-checklist`.
- Chaque lot : entrée `thought_log.md` AVANT commit (règle obligatoire).

---

## LOT A — Correctness rapides (Go + web)

- [x] A1 (ID1) `internal/api/wire/registry_auth.go:105` — corriger le SUJET du
      season pass (décision D7). ÉTAPE 1 (vérifiée sur pièces) : GetBattlePass/
      GetChallenges construisent l'URL `players/xuid(<HaloXUID>)/{rewardtracks,decks}`
      (provider.go:270,308,382,427) et persistent sous `sink.xuid` = pdb.XUID → endpoints
      OWNERSHIP-SCOPED (pattern « fetch live 403 → cache 24 h », mémoire
      reference_bp_challenges_staleness_auth403). ÉTAPE 2 : `forcePageIdentityXUID`
      appliqué comme ligne 53 → sujet=page ; porteur≠page → 403 upstream → fallback cache
      DB du sujet (aucune écriture croisée). Docstrings mises à jour. Tests ajoutés
      (`registry_auth_enrich_test.go` : viewer≠sujet / viewer=sujet). `go test ./internal/api/wire/...` ok.
- [x] A2 (ID4) — ratchet `registry_auth_page_identity_ratchet_test.go` (AST) : recense
      les FuncDecl du package wire appelant `enrichWithHaloTokens`, échoue si l'un n'applique
      pas `forcePageIdentityXUID` dans la même fonction ; allowlist datée (Compare,
      ExplorerCtxWithAuth = xuid explicite) + garde anti-exemption-périmée. Vert.
- [x] A3 (W1) `XboxLoginPage.tsx:70` — `fetchQuery({staleTime:0})` (défaut global 5 min
      vérifié queryClient.ts:22). La lecture repeuple `queryKeys.bootstrap` que `__root`
      observe → `hydrateFromBootstrap` resync l'appShellStore. Test vitest (cache anonyme
      préchargé, client staleTime 5 min → dashboard, pas onboarding). 9/9 vert.
- [x] A4 (M3) — nouveau champ `Migration.EnsureAdditive` appelé par le chemin DM-5
      (`recordSupersededBaseline`, avant l'INSERT). `ensureBaselinePlayerV1AdditiveColumns`
      étendu (colonnes render challenge_snapshots + `engagement_response_bins` via const
      partagée `ddlEngagementResponseBins`, sortie de applyBaselinePlayerV1Tables → source
      unique). Golden invariant intact. Test heal DM-5 (sentinelle présente, additifs
      absents → boot → présents, INSERT expected_win_prob OK). Intégration verte.
- [x] A5 (LB1) `engine_postsync_csr.go` — lecture de la vue partitionnée
      `world_csr_leaderboard_latest` (par titre/saison/playlist) au lieu du MAX(fetched_at)
      global. Test intégration (2 playlists à fetched_at distincts → les 2). Vert.
- [x] A6 (E5) — milestones_earned_repo.go Append : QueryRow→QueryRowRecovered +
      Exec→ExecRecovered (handle nu même méthode) ; 4 `.Player.Exec(`
      (career_progression_partial, career_live_repo, engagement_score_repo, fanout_repo)
      → ExecRecovered. Ratchet `player_db_recovery_routing_test.go` étendu au motif
      `.Exec(` (sanity + doc). 0 plat restant (non-test). Package duckdb vert.
- [x] A7 (ME1) `cmd/cleanup_media_index/main.go` — `escapeLikeLiteral` (\, _, %) +
      `LIKE ... ESCAPE '\'`. Test unitaire (échappement) + test DuckDB (préfixe littéral
      "Halo_5_Guardians-" ne matche plus "HaloX5YGuardians-"). Vert.
- [x] A8 (AU1) `refresh_user_xsts.go` — `refreshAccessTokenForUser` reçoit le store et
      persiste le RT roté IMMÉDIATEMENT (persistRotatedRT) après le refresh OAuth, avant
      XSTS. Test : refresh mocké rote le RT, XSTS échoue → store contient le RT roté. Vert.
- [x] A9 (AU3) `access_token_store_first.go:58` — `store.Load` err logué
      (slog.ErrorContext) avant la bascule legacy. Test : xuid unsafe → Load err → log
      ERROR présent. Vert.
- [x] A10 (W4) `DetectionsPanel.tsx:49` — `window.prompt`→null ⇒ early-return, aucune
      mutation. Tests vitest (Annuler → pas de PATCH ; validation → PATCH). Vert.
- [x] A11 (W3) `HomeRecentPlaylistsCard.tsx` — 4 clés i18n
      (`common.home.rank_unrated`/`rank_placement`/`rank_placement_progress`/`rank_label`)
      FR+EN dans common.toml (regénéré). Libellés FR préservés → tests home existants
      passent. tsc + vitest home verts.
- [x] A12 (CV4) — `cmd_data.go:233` emoji retiré ; `discord.go` commentaire d'exemption
      daté D2 (emojis = contenu du payload Discord). vet levelup+notify ok.
- [x] A13 (CV1) `cmd/engagement-calibrate/main.go` — glob via
      `PathResolver.PlayersRootDir` ; `log`→`slog` (5 sites) ; champ `Bins` mort supprimé ;
      param `ref` d'`autoVerdict` retiré + godoc corrigée. Build cgo ok.

**Gate A** : go test ciblés (wire, migration, sync, platform/auth,
platform/duckdb, cmd) ; `make check-types` ; vitest ciblé (auth, admin, home) ;
lint new-from-merge-base ; entrée thought_log ; commit `fix(revue): lot A ...`.

## LOT B — Identité en profondeur

- [x] B1 (ID3) `internal/service/career_live_fetcher.go` — clé de contexte
      `tokensOwnerXUID` (ctxkeys). POSE : `WithHaloAuth` la pose = xuid (SEUL point
      d'écriture des tokens → tout ctx portant des tokens a un porteur) ; `WithHaloXUID`
      (forcePageIdentityXUID) NE la touche PAS → sujet forcé sur la page, porteur
      préservé. CORRECTION bg : `kickoffBackgroundRefresh` re-injecte le porteur réel
      capturé du ctx requête (le factory tourne dans le bg pour le chemin home — sans
      cette carry-forward le bug persistait). Le factory clé le limiteur sur
      `ratebudget.ForXUID(TokensOwnerXUID(ctx))` (au lieu de `HaloXUID`). Vérifié sur
      pièces : Home forcé (session Y, page X → tokensOwnerXUID=Y préservé après forçage) ;
      pool sync (`pool.go` clé déjà sur `src.XUID` = porteur, inchangé) ; watcher
      (`WithHaloAuth(ctx, tokens, r.xuid)` = porteur). Test : `TestCareerFetcherFactory_
      BudgetKeyedOnTokensOwner` (page X + porteur Y → CurrentRPS(Y)=rps, CurrentRPS(X)=0),
      + `TestForcePageSubject_PreservesTokensOwner` (ctxkeys, invariant central). Gate B vert.
- [x] B2 (ID4 profond) — `GetSpartanIdentity(ctx)` SUPPRIMÉ (lisait le sujet ambiant
      `ctxkeys.HaloXUID`) ; le sujet est désormais un PARAMÈTRE explicite via
      `GetSpartanIdentityFor(ctx, xuid)`. Call-sites migrés (Grep exhaustif) :
      PROD (2) — `home_service.go` (`s.xuid` = pdb.XUID, vérifié : WithMatchesCache pose
      pdb.XUID dans les 3 constructions HomeService ; HaloXUID forcé = pdb.XUID dans les
      3 chemins → identique) ; `spartan_customization_cron.go` (`p.XUID`, interface
      `SpartanIdentityFetcher` mise à jour) ; TESTS (14) — career_live_service_test.go
      (10), career_live_e2e_scenarios_test.go (3), spartan_customization_cron_test.go
      (mock + assertion p.XUID). `ctxkeys.HaloXUID` conserve sa sémantique ownership
      (`subjectIsOwner`, garde persist tiers) — non touché. Ratchet A2
      (`TestEnrichCallersForcePageIdentity`) reste VERT (non-régression). `forcePageIdentityXUID`
      conservé (sujet BP/défis + ownership). Docs mises à jour (main.go, og_inject.go/_test.go).

**Gate B** : gofmt + go vet OK ; `go test ./internal/service/... ./internal/scheduler/...
./internal/api/wire/... ./internal/ctxkeys/...` VERT ; `go build ./...` VERT (aucun caller
orphelin) ; intégration N/A (aucun chemin persist/sync touché) ; golangci-lint
`--new-from-merge-base=main` = **0 issues** ; thought_log ; commit.

## LOT C — Filets squash M2 (le morceau lourd)

- [x] C1 — réparation par introspection (D3) :
      `migration.EnsurePlayerCSRSnapshotsAppendOnly` (nouveau, dans
      `steps_player_append_only_csr_snapshots.go` — fichier tombstone du squash,
      réhabilité) : détection par INTROSPECTION des colonnes (marqueur `id`
      absent = ancien schéma), conversion via le helper ADR 0026
      `applyAppendOnlyRebuild` (CTAS transactionnel + garde anti-perte
      rebuilt==before + recoverOrphan + idempotence). ViewSQL vide À DESSEIN :
      la vue reste possédée par playerSchemaSQL (source unique). EMPLACEMENT :
      câblé en TÊTE de `sync.EnsurePlayerSchema` (AVANT playerSchemaSQL) —
      choisi car c'est LE point que TOUT chemin d'ouverture player traverse
      (OpenPlayerDB : provider, sync, H5 livesync) et le seul qui garantisse
      l'exécution AVANT la création de l'index/vue qui bindent sur
      written_at/id ; la couche migration seule (RunForDB boot 3b) ne convient
      pas : elle ne tourne pas sur tous les chemins d'ouverture et son step
      squashé ne se rejoue plus. Tests
      `TestPlayerCSRSnapshotsAppendOnly_LegacySwap` (sanity : la vue NE binde
      PAS avant, binde après ; zéro perte 3/3 ; idempotence re-run ; pas
      d'orphelin) + `_FreshDBNoop`. Verts.
- [~] C2 — colonnes render `challenge_snapshots` : couvert par l'ensure étendu
      de A4 (`ensureBaselinePlayerV1AdditiveColumns`, chemin DM-5
      `recordSupersededBaseline` via `Migration.EnsureAdditive` — commit
      b497befc1). VÉRIFIÉ sur fixture mi-bloc : subtest C3
      `c_without_challenge_render_columns` (sentinelle présente, 4 colonnes
      render droppées → boot → INSERT challenges avec title/description/
      image_url/display_path OK, 2e boot idempotent). Vert.
- [x] C3 — fixtures états intermédiaires PROGRAMMATIQUES
      (`internal/sync/squash_convergence_test.go`, DDL historique exact tiré de
      `git show 37264462f^`) : (a) pré-05-24 ancien player_csr_snapshots
      PK(playlist_id,season_id) sans id/written_at ; (b) sentinelle DM-5 sans
      expected_win_prob ; (c) sans colonnes render ; (d) sans
      engagement_response_bins. Convergence par fixture : RunForDB(player) +
      OpenPlayerDB OK, `player_csr_snapshots_latest` liée et requêtable,
      INSERT persist LUSR (liste de colonnes du lusr_append_only_persister,
      expected_win_prob incluse) OK, INSERT challenges (liste de colonnes de
      persist_sink_challenges, render incluses) OK. IDEMPOTENCE : 2e boot sans
      échec, count player_csr_snapshots stable, pas de table orpheline
      `__appendonly`. 4/4 subtests verts (a 0.66s, b 0.80s, c 0.78s, d 0.77s).
- [x] C4 — `sync/schema.go` : `recoverPlayerSchemaBoot` — un échec de DDL/bind
      au boot player log `slog.ErrorContext` (vue + cause brute), déclenche la
      réparation C1, rejoue le script une fois ; si l'échec persiste → erreur
      EXPLICITE « intervention requise » (jamais de panic, jamais d'avalement).
      Test `TestEnsurePlayerSchema_C4_UnrepairableBindReturnsExplicitError`
      (nom de vue squatté par une table → erreur explicite). Vert.

**Gate C** : `go test -tags=integration -p 1 ./internal/migration/... ./internal/persist/...`
(filtre ancré) + tests fixtures ; thought_log ; commit.
GATE C PASSÉ 2026-07-17 : gofmt clean + `go vet` migration/sync OK ;
`go test -tags=integration -p 1 ./internal/migration/...
./internal/games/halo_infinite/migrations/... ./internal/sync/...
./internal/persist/...` = TOUT ok (migration 2.8s, halo migrations 26.2s,
sync 104.6s, persist 16.6s) ; `go test ./internal/platform/duckdb/...` ok
(31.3s) ; `golangci-lint --new-from-merge-base=main` = 0 issues.

## LOT D — Efficacité (VPS 2 vCPU / 2 Go)

- [x] D1 (E1) `internal/service/engagement_player_service.go` — memo par
      `mode_category` (champ `expectedMemo` + type `expectedInputsEntry`, lazy
      dans `loadExpectedInputs`). Correct par construction : coef+bins ne
      dépendent que de (xuid, mode_category), xuid figé sur le service (créé par
      requête) → cache stable, pas de reset explicite en début de GetTimeseries
      requis (mesurable ≤4 atteint identiquement). REPO
      (`engagement_response_bins_repo.go`) : `responseBinsTableExists` mémoise
      son scan information_schema (champ `responseBinsExists *bool` sur
      `EngagementScoreRepo`, extraction `queryResponseBinsTableExists`) → 1 scan
      par handle au lieu d'1 par appel. Décision A4/C : check CONSERVÉ (les DBs
      pas encore bootées peuvent ne pas avoir la table) mais mémoïsé — la
      « simplification » = per-call → per-handle. Test
      `TestGetTimeseries_MemoizesCoefAndBinsPerMode` (200 matchs, 2 modes) :
      **coefCalls=2, binsCalls=2 (=4 total)** vs ~600 avant. VERT.
- [x] D2 (E3) `internal/api/wire/registry_monitoring_freshness.go` — `lastMatchAt`
      (par joueur, résolution player-DB jetée + MAX+JOIN par joueur, CRÉAIT les
      DBs auth_only) SUPPRIMÉ → `lastMatchByXUID(ctx, titleSlug, xuids)` : UNE
      requête groupée `WHERE mp.xuid IN (...) GROUP BY mp.xuid` (timezone
      canonique `analysis.SQLStartTimeCanonical`) sur le shared lu via
      `duckdb.OpenReadForQuery(SharedDBPath)` (B-swap-safe, jamais OpenReadOnly
      forcé). AUCUNE player-DB résolue → aucune DB créée pour les auth_only (ils
      sont juste absents du résultat). `resolveMonitoringDBs` conservé (autre
      caller : ConvergenceBacklog). Test
      `TestLastMatchByXUID_GroupedNoPlayerDBCreation` : grouping+MAX corrects,
      xuid auth_only absent, **répertoire players NON créé**. VERT.
- [x] D3 (E4) `internal/ops/monitoring_retention.go` (nouveau) + wiring flush —
      `SweepRetention` (CapAndSweep façon notifications) sur `detection_events`
      (cap 20 000), `detection_status_events` (cap 10 000), `cron_runs` (cap
      50 000) — constantes nommées, justifiées (croissance/scan vue `_latest` <
      qq ms VPS). DELETE PURGE mono-writer (lease `KindMonitoring`, même writer
      que le flush ; COUNT sans lease, DELETE sous lease seulement si > cap) qui
      PROTÈGE le dernier événement par partition (MAX(id) par fingerprint /
      cron_name) → vues `_latest` intactes. Sweep piggyback sur
      `RunDetectionFlushLoop` (tick + shutdown), pas de cron dédié. ART : tables
      HORS `tablesProtegees` ; base globale isolée mono-writer PK BIGINT → DELETE
      sûr. Garde-rail `monitoring_store_guard_test.go` MIS À JOUR : writer
      (`monitoring_store.go`) reste zéro UPDATE/DELETE ; retention
      (`monitoring_retention.go`) = zéro UPDATE, DELETE sanctionné (daté D3).
      Test `TestMonitoringStore_SweepRetention_BoundsAndPreservesLatest` :
      bornes respectées + `detection_status_latest`/`cron_runs_latest` corrects
      après purge + idempotence. VERT.
- [x] D4 (E2) `apps/web/src/features/explorer/ExplorerPage.tsx` —
      `include_briefing: mode === 'matches'` + query gatée `enabled: mode ===
      'matches'` (4e arg de `useExplorerMatches`, non consommée en mode Joueur) ;
      input match-ID débouncé 250 ms (`MATCH_ID_DEBOUNCE_MS`) via hook partagé
      `useDebounced` extrait dans `apps/web/src/lib/hooks/useDebounced.ts`
      (migration de la copie feedback-drawer, pas de ré-implémentation — règle
      n°6). Tests `ExplorerPage.efficiency.test.tsx` : mode Joueur → query
      désactivée + include_briefing=false ; frappe rapide → seule 'abc' atteint
      la query après 250 ms (1 POST/rafale). 103 tests web VERTS.

**Gate D** : PASSÉ 2026-07-17. gofmt clean (fichiers touchés) ; `go vet`
service/ops/wire/duckdb OK ; `go test ./internal/service/... ./internal/ops/...
./internal/platform/duckdb/...` = ok (service 11.2s, ops 17.8s, duckdb 33.5s) ;
`go test ./internal/api/wire/...` = ok (0.8s) ; `go build ./...` OK ; schéma
monitoring NON touché → migration integration N/A ; `golangci-lint
--new-from-merge-base=main` = **0 issues** (goconst corrigé :
`freshnessLastMatchReadErr`) ; `make check-types` OK ; vitest explorer +
feedback-drawer = 103/103.

## LOT E — UX web restants

- [x] E1 (W2) `apps/web/src/features/match-view/MatchTugOfWarChart.tsx` —
      tooltips item des séries kill-feed/vagues rétablis. FORME CHOISIE : trigger
      GLOBAL `'item'` (option 1 du plan), résumé de bin servi par le tooltip item
      des DEUX barres (`buildBarSeries` reçoit `binTooltips` via options-object,
      `tooltip: { formatter: binTooltipFormatter(binTooltips) }`). Justif : ECharts
      n'a PAS de `trigger` par-série (vérifié : `series.tooltip` ne porte que
      `formatter`/`position`, jamais `trigger`) — l'option 2 (« garder 'axis' et
      re-router ») ne peut PAS produire de vrai tip per-item (axis agrège toute la
      colonne, deux grilles → conflit), donc option 1 est la seule forme correcte
      ET la plus simple ; elle préserve LES DEUX comportements (résumé de bin =
      contenu delta/X-Y/cumuls intact, tips per-kill/vague effectifs) et retrouve
      la structure connue-bonne d'avant la régression (24fc02f2f était déjà en
      'item' + formatters par-série). Retrait des `trigger:'item'` par-série morts
      (scatter + vagues) + de l'`axisPointer` shadow (affordance axis inutile en
      item). Test `MatchTugOfWarChart.test.tsx` (mock echarts-for-react capturant
      l'option, aucun canvas jsdom) : trigger global == 'item' ; formatter barre →
      résumé de bin (Écart +3 « Mon équipe ») ; formatter scatter → tip per-kill
      (« gamertag — 0:01 ») ; formatter vague → « ×3 » ; garde-rail : scatter/vague
      sans `tooltip.trigger`. 4/4 vert.
- [x] E2 (W5) `apps/web/src/features/notifications/NotificationsBell.tsx` —
      filet `visibilitychange` (hidden) + `pagehide` flushant `pendingReadRef` via
      keepalive (D5). Client NON dupliqué : option `keepalive` ajoutée à
      `request()` + méthode `api.postKeepalive` (mêmes URL/headers/credentials/
      erreurs que `post`) ; helper `markNotificationsReadKeepalive` dans
      `mutations.ts`. Chemin nominal open→false INCHANGÉ. Anti double-envoi :
      `flushedReadRef` — le flush vide `pendingReadRef` AVANT l'envoi et marque les
      ids flushés ; l'accumulation (dropdown ouvert, onglet re-visible sans
      fermeture) les ignore ; le nominal ne touche PAS `flushedReadRef` (un
      rollback d'erreur doit pouvoir re-tenter, dropdown fermé = accumulation
      stoppée). Tests `NotificationsBell.test.tsx` : visibilitychange hidden
      (dropdown ouvert) → 1 markRead [101,102] ; re-hidden → pas de re-flush
      (toujours 1) ; pagehide → 1 markRead [303]. 7/7 vert (5 existants + 2).

**Gate E** : PASSÉ 2026-07-17. `make check-types` (tsc -b) OK ; vitest ciblé
(match-view + notifications) = 11/11 ; `make test-web` complet = 263 fichiers,
2272 passés / 14 skippés, 0 échec ; `npx eslint` sur les 6 fichiers touchés
(MatchTugOfWarChart .tsx + .test.tsx, NotificationsBell .tsx + .test.tsx,
mutations.ts, client.ts) = 0 issue ; thought_log ; commit.

## LOT F — Duplications, dette, robustesse

- [ ] F1 (R2) — helper unique outcome int→valeur (`apps/web/src/lib/`), contrat
      ADR 0006, défaut EXPLICITE documenté ; migrer les 4 copies
      (ExplorerBriefing.logic, session-detail/_shared, TimeseriesPage.summary,
      SquadSynergiesPage) ; garde-rail grep interdisant les mappings locaux.
- [ ] F2 (R1) — helper « plus longue série » dans `internal/analysis/` ;
      migrer les 4 copies Go (briefing_streaks, highlights_tiles, synthesis,
      detectTilt) ; garde-rail.
- [ ] F3 (R4) — `formatSignedFixed` dans `lib/formatters/` ; migrer les 3
      copies strictes ; glyphe unique '−' (U+2212) ; garde-rail (3e copie).
- [ ] F4 (R5) — une seule fonction coalesce dans `internal/service` ; migrer.
- [ ] F5 (R7) — `deltaToken` exporté depuis `ExplorerBriefing.logic.ts`,
      2 copies supprimées.
- [ ] F6 (R8) `internal/analysis/campaign_exclusion.go` — `quotedIDList` +
      builder commun aux deux fonctions SQL.
- [ ] F7 (R6) — seeder synthétique : partager les listes de colonnes avec
      `persist` (constantes exportées ou helpers d'insert réutilisés) + test
      qui CASSE si les colonnes du seeder divergent de persist (protège la
      recette ADR 0026).
- [ ] F8 (ME3) `internal/platform/duckdb/media_repo_registry.go:205` —
      remplacer le mini-framework à closures par un helper direct appelé 3
      fois (bulk resolve conservé).
- [ ] F9 (CV5) `internal/api/wire/post_sync_deltas.go:280` — variadique →
      paramètre struct obligatoire (mise à jour mécanique des ~25 call-sites
      de test).
- [ ] F10 (LB2, décision D1) — `runOnceForTitle` retourne error ; RunOnce
      agrège et passe l'erreur réelle à `ReportCronRun` ; appliquer aux 5
      crons concernés ; tests cronstatus (échec partiel → failure visible).
- [ ] F11 (LB3) — graine de découverte de saison = dernière saison persistée
      dans les snapshots, constante `csrseason13-2` en simple fallback. Test.
- [ ] F12 (AU4) — provenance du token (famille de client) persistée dans
      `UserTokens` à l'acquisition ; préfixe RpsTicket déterministe ; retry
      conservé en filet AVEC log WARN. Test.
- [ ] F13 (E5 profond, décision D6) — handle de lecture player-DB n'exposant
      que les variantes `*Recovered` (fermeture par construction des 3 classes
      de trous du garde-rail grep). Migrer les lecteurs player-DB vers ce type.

**Gate F** : go test paquets touchés + intégration filtrée ancrée (persist/
seeder) ; vitest lib/features migrées ; lint ; thought_log ; commit.

## Clôture chantier

- [ ] Z1 — suite complète : `go test -tags=integration -p 1 ./...` (apps/go-api).
- [ ] Z2 — `scripts/check_test_baseline.sh tests` (baseline intacte).
- [ ] Z3 — `make check-types` + `make test-web` complets.
- [ ] Z4 — skill `delivery-checklist` ; statuer TOUS les items du plan ;
      section Découvertes soldée (traitée ou consignée) ; entrée thought_log de
      clôture.
- [ ] Z5 — superviseur : merge dans main + prévenir l'utilisateur AVANT push
      (push main = deploy prod) ; revue visuelle utilisateur au merge.

## Protocole de reprise de session

1. Lire ce plan (version du WORKTREE, qui fait foi pendant le chantier) : le
   premier item non statué du premier lot incomplet = point de reprise.
2. Lire les entrées thought_log du chantier (`[2026-07-*] LOT x — revue
   correctifs`).
3. `git -C <worktree> log --oneline -10` pour l'état des commits.
4. Un seul agent à la fois dans le worktree ; vérifier qu'aucun agent/build
   n'est vivant avant d'en lancer un autre (link.exe orphelins → kill).

## Découvertes (hors périmètre — consigner ici, NE PAS traiter)

- [LOT A] `cmd/levelup/cmd_data.go` porte des emojis PRÉEXISTANTS en sortie CLI
  (lignes 52, 95, 137, 185, 190, 293, 298, 305, 311, 351 — `✅`/`⚠️`). La revue n'a
  flaggé que la ligne 233 (seule AJOUTÉE dans la fenêtre) ; A12 l'a retirée. Les autres
  sont de la dette gelée (baseline) hors périmètre A12 — non traités. À solder si un
  chantier « purge emojis CLI » est décidé.
- [LOT A] `internal/platform/duckdb/fanout_repo.go:98` : `slog.Warn(...)` sans variante
  `*Context` (viole la convention slog structuré-avec-ctx). Préexistant, hors périmètre
  A6 (qui ne portait que la conversion recovery Exec→ExecRecovered). Non traité.
- [LOT D / D1] `LoadPlayerHistory` est appelé PAR MATCH dans le compute timeseries
  (`loadHistorySafeByMode`, avec `ExcludeMatchID` variable) → N requêtes retournant
  jusqu'à 200 lignes chacune. E1 ne visait QUE coef+bins (fonction de (xuid,
  mode_category)) ; l'historique dépend de l'exclusion du match courant, sa mémoïsation
  changerait la sémantique (percentile baseline). Hors périmètre D1 — non traité. Piste
  si besoin : charger l'historique complet par mode une fois et exclure en mémoire.
- [LOT E / E1] `MatchTugOfWarChart.tsx` porte des libellés FR EN DUR
  PRÉEXISTANTS (« Lane alliée », « Lane ennemie », « Mes kills », « Kills
  ennemis », « Vague <side> ×N », « kill(s) ») non passés par i18n — antérieurs à
  la fenêtre de revue (déjà présents en 24fc02f2f), NON flaggés par la revue
  adversariale, hors périmètre E1 (qui ne visait que le trigger des tooltips).
  Non traités. À solder si un chantier i18n match-view est décidé.
- [LOT D / D3] `data_health_runs` croît aussi sans rétention (1 ligne par audit
  data-health), mais basse fréquence et son lecteur (`LatestDataHealthJSON`) ne lit que
  la dernière ligne. Le plan D3 liste explicitement 3 tables (detection_events,
  detection_status_events, cron_runs) — `data_health_runs` NON incluse, non traitée.
  À ajouter à `monitoringRetentionSpecs` si un chantier rétention l'exige.
