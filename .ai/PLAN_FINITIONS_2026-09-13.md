# Plan — finitions v7.5 (2026-09-13)

> Arbitré par l'utilisateur le 2026-09-13 (message « tu peux t'occuper de tout le reste »).
> Contrat : skill `plan-execution`. **Périmètre FERMÉ** : toute découverte hors périmètre est
> consignée en §Découvertes et NON traitée, sauf P0 (corruption de données, perte, faille).
> Base : feat/v75 après le commit `chore(deps)` du 13/09. Un lot = un worktree
> `LevelUp-wt-finitions-<slug>` + branche `feat/finitions-<slug>` (les branches `wt/*` ne
> déclenchent pas la CI). Fusion dans feat/v75 par le pilote, revue adversariale finale unique.
> Règle « vérifier sur pièces » : l'arbitrage date du 11/09, des lots ont pu fermer un item
> depuis — un item déjà réglé se statue `[~]` avec le commit qui l'a réglé, jamais refait.

## Décisions utilisateur (ne pas rediscuter)

- D1 : NON (l'info est déjà servie par l'API). D8 : hors périmètre (couvert par la refonte du
  décodeur, lot I). D2, D4, D5 : lots ultérieurs avec recuisson. D6, D12, D14, D16, G5, G8,
  G9, H2, H3, H5 : NON / plus tard. D13 : à planifier à part (équipement à gérer, M).
- Retenus ici : D9, D10, D15, G2, G3, G4, G6 (ADR), G7, H1, H4, retrait de la migration boot
  des jetons (échéance 2026-10-01, critère tenu en prod), statut killpos (échéance 2026-11-08,
  critère tenu), lot PSA, lot LUSR.

## Lot A — Dependabot (pilote) — FAIT
- [x] A.1 Bumps sur feat/v75, gates verts, commit `chore(deps)`, push.
- [ ] A.2 CI feat/v75 verte au niveau job.
- [ ] A.3 Fermer les PR 79-83 avec renvoi vers le commit.

## Lot B — hygiène + PSA + jetons + killpos (`feat/finitions-hygiene`, Go/config/docs)

### B.1 PSA (garde data-health + rectification du numéro d'issue)
- [ ] B.1.1 `git merge wt/psa-index-cause` (merge-tree propre vérifié le 13/09) ; déplacer
      `RAPPORT_VOLET2_INDEX_PSA.md` de la racine vers `.ai/V7.5/`. Les 5 fichiers
      `internal/migration/psa_index_repro*_test.go` restent derrière le tag `psarepro`.
- [ ] B.1.2 Rectifier `#23046` -> `#23645` partout où le texte désigne le bug « Failed to delete
      all rows from index » : `CLAUDE.md` (L107), `docs/adr/0026-*.md` (titre + L12),
      `docs/adr/0030-*.md` (L15), `docs/WEAPONS.md`, `docs/FR/WEAPONS.md`, `docs/CHANGELOG.md`
      et `docs/FR/CHANGELOG.md` (une occurrence chacun), tous les `.go` sous `apps/go-api`
      (commentaires et messages de test, ~25 fichiers — `git grep -n 23046 -- '*.go'`).
      Ne PAS toucher `.ai/archive/`, `.ai/migrations/squashed/` ni les rapports datés.
- [ ] B.1.3 `.ai/V7.5/REGISTRE_REPORTS.md` L538 : cause = amont duckdb/duckdb#23645 (ouverte,
      présente en 1.5.5), garde posée (alerte seule, 41 ms/base), condition de reprise =
      sortie d'une 1.5.6 contenant #24744, ou jauge `data_health_psa_index_desync_keys` > 0.
- [ ] B.1.4 Gate : `go test -tags=integration -p 1 ./internal/scheduler/... ./internal/migration/...`
      (exit 0 vérifié) + `go test ./internal/scheduler/...`.

### B.2 Retrait de la migration boot des jetons legacy (ADR 0023, échéance 2026-10-01)
Critère tenu : prod `auth_migration: scan terminé` rt_migrated=0 à chaque boot du 2026-06-14
au 2026-09-13 (2 RT migrés le 2026-06-13, jamais depuis) ; local idem depuis le 2026-05-29.
- [ ] B.2.1 Supprimer `internal/platform/auth/migration.go` + son test,
      `migrateLegacyAuthTokensAtBoot` + `legacyAuthSourcesReader` dans `cmd/server/main.go`,
      les helpers DuckDB de `internal/platform/duckdb/queries_auth.go` devenus orphelins,
      `auth.EnvRefreshTokenForGamertag` et tout appelant, les entrées d'allowlist de
      `internal/platform/auth/sentinel_test.go` (l'allowlist doit tomber à 0 entrée, pas être
      contournée), la référence dans `internal/ops/seed_demo_sync_meta.go`,
      `internal/sync/no_legacy_source_used_test.go` mis à jour (le garde reste, l'exception
      disparaît). Aucun import mort.
- [ ] B.2.2 `CLAUDE.md` § « Règle auth tokens » : retirer le paragraphe « Seule exception legacy
      restante » (dater le retrait 2026-09-13, critère constaté). ADR 0023 : note de clôture
      de la Phase 5 (EN-only).
- [ ] B.2.3 Gate : `go build ./...`, `go vet ./...`, `go test ./internal/platform/auth/...
      ./internal/sync/... ./cmd/server/...` + `-tags=integration -p 1 ./internal/sync/...`.

### B.3 Hygiène XS
- [ ] B.3.1 D9 — socle central d'Illusion étiqueté équipe 0 au catalogue : localiser le
      catalogue (`film/replay/mapvar/objectives.go` ou le TOML de carte), corriger en
      « neutre », test de non-régression ; si l'item est déjà réglé, `[~]` + commit.
- [ ] B.3.2 D15 — `ListMapsByTitle` (`platform/duckdb/metadata_repo_assets_list.go`) :
      dédoublonner par `map_asset_id` (le web le fait déjà par identifiant), test.
- [ ] B.3.3 G2 — `cmd/replay-corpus-gate` : un témoin du corpus sans chunks en cache fait
      ÉCHOUER le gate avec le nom du film (aujourd'hui silencieux) ; test unitaire sur le
      manifeste.
- [ ] B.3.4 G3 — `config/replay_corpus.toml` : la raison du témoin Oddball décrit un résidu
      fermé le 2026-09-08 ; réécrire la raison (ce que le témoin exerce aujourd'hui).
- [ ] B.3.5 G4 — commentaires : `domain/replaydoc/coverage_objectives.go` /
      `film/replay/coverage_bridge.go` — l'invariant `Balanced()` est vrai par construction
      (protection réelle = un seul fermoir par portage) ; `closedBy*` baissent sur les films à
      `noTrack` (raison à écrire, pas le double comptage). Doc seule.
- [ ] B.3.6 G6 — ADR 0026 : documenter `decode_pass` (colonne de `kill_positions`, passe de
      décodage, clé de la vue `_latest`). Si `seed_demo_corpus.go` porte encore un
      `UPDATE kill_positions` : allowlister avec justification datée dans
      `internal/sync/no_art_patterns_test.go` (outil de démo, base jetable) — sinon `[~]`.
- [ ] B.3.7 G7 — commentaire de `GroundWeapon.W` (`document_ground_weapon_items.go`) : espace
      de clés distinct de `Loadout.W` (rapport 6.6 découverte 1). Doc seule.
- [ ] B.3.8 H1 — `MatchMetrics` (`internal/domain/stats.go`) : supprimer si aucun usage
      (`git grep -nw MatchMetrics`), sinon `[~]` ; `cmd/mapcallouts-build/decoupe_masque.go` :
      extension `.png` en constante nommée.
- [ ] B.3.9 H4 — supprimer `cmd/weapon-sounds`, `cmd/vs-measure`, `cmd/vehicle-sprite` (git
      garde l'historique) ; retirer toute mention dans `docs/COMMANDS.md` (FR+EN) et
      `.ai/project_map.md`.
- [ ] B.3.10 killpos — `.ai/V7.5/PLAN_LOT_PONT_ET_KILLPOSITIONS.md` items 2.3, 2.4, 2.6 : statuer
      `[~]` (critère du 2026-11-08 tenu : `BuildKillPositions` appelé par
      `sync/killcollector/positions.go:347`, écriture `persist/shared_persister.go` INSERT-only,
      `kill_positions_latest` = 114 038 lignes / 1 307 matchs Infinite au 2026-09-13) ; le CLI
      `cmd/killpos-build` est supersédé par `levelup backfill-killsource`. Le garde 88 %
      (`replay_local_gate.go`) reste à la main de l'utilisateur (Notion).
- [ ] B.3.11 Gate lot B : `go build ./...`, `go vet ./...`,
      `go test $(go list ./... | grep -v /internal/himap)`, `make go-api-lint`, push, CI verte.

## Lot C — LUSR (`feat/finitions-lusr`, Go)
Source : `RAPPORT_VOLET1_LUSR_H5.md` sur `wt/lusr-h5-cause` (§5.3 G1-G4, §6). Census du
13/09 (serveur arrêté) : lignes brutes `h5_arena` dans les player DB Infinite = JGtm 913,
Madina 1 064, Chocoboflor 471, Daemon 31, chacune en LUSR + LUSR_V2 ; `_latest` n'en garde
que 2 (Madina, matchs non rejouables) ; les 4 bases halo_5 ne portent que `h5_arena`.
- [x] C.1 `SyncEngine.RecomputeLUSRCanonical` (`internal/sync/engine_backfills.go:187`) stampe
      `ctxkeys.WithTitleSlug(ctx, e.titleSlug)` avant `RecomputeLUSRCanonicalForPlayer` (miroir
      de `engine_postsync_scoring.go:162`). Couvre `registry_lusr_gaps.go` (replay admin) ;
      `service/openspartan_post_import_service.go:152` stampe aussi le titre de la base qu'il
      ouvre. Test : un ctx porteur d'un autre titre ne change pas la chaîne écrite.
- [x] C.2 Fail-loud au câblage V2 : `cmd/server/sync_v2_wiring.go`, avant L283 — si
      `p.TitleSlug != "" && p.TitleSlug != deps.TitleSlug`, erreur `ErrorContext` nommant
      profil, titre du profil, titre du cycle et l'incident 2026-06-26 ; le profil remonte
      `failed` dans le `CycleResult`. Test `TestBuildSyncEngineFactory_RefusesForeignTitleProfile`
      (3 cas du rapport). Ratchet `TestSyncV2WiringHasSingleTitleSource` (toute ligne portant
      `p.TitleSlug` porte aussi `deps.TitleSlug`).
- [x] C.3 Lecteur : `Q8LUSRHistoryPlayer` (`platform/duckdb/queries_career.go:207`) lit
      `match_skill_rank_latest` (une ligne par match et rating_type, la plus récente) au lieu de
      la table brute ; mettre à jour le commentaire (règle ART n°2). Test existant du repo
      carrière adapté : une ligne périmée d'un match rejoué n'apparaît plus.
- [x] C.3 bis (correction pilote) : `match_skill_rank_latest` partitionne par `match_id` SEUL
      avec priorité CSR > LUSR — la brancher au graphe faisait DISPARAÎTRE le point LUSR de
      tout match classé portant les deux lignes (perte de données rendues). Nouvelle vue
      `match_skill_rank_latest_by_type` (migration player `player_msr_view_latest_by_type_v1`,
      partition `(match_id, rating_type)`, `written_at DESC, id DESC`) ; `Q8LUSRHistoryPlayer`
      la lit ; ratchet `no_raw_rating_reads_test.go` élargi explicitement au suffixe ;
      test « match classé → les DEUX lignes servies ».
- [x] C.4 Invariant `invariants.CheckPlayerLUSRChains(ctx, db, allowed)` (clé
      `lusr_chain_foreign_title`, `SeverityFail`, table brute) + test de violation ; branché au
      gate d'intégration avec la liste des chaînes du titre lue depuis
      `games/halo_infinite/skillchain` (vérifier la liste SUR PIÈCES, ne pas la recopier).
- [x] C.5 Outil `cmd/purge_foreign_lusr_chain` : une base à la fois, `-dry-run` par défaut,
      `-commit` explicite, reconstruction CTAS transactionnelle sur le modèle de
      `migration/append_only_rebuild.go` (garde de cardinalité avant DROP, vue `_latest`
      recréée, CHECKPOINT), JAMAIS de DELETE. Test sur fixture (2 lignes étrangères, 3 saines).
      L'EXÉCUTION sur les 4 bases est faite par le pilote (serveur arrêté, sauvegarde préalable).
- [x] C.6 Registre L537 : cause PROUVÉE (double source de titre, profils déclarés sous deux
      titres) + gardes C.1/C.2/C.4 + purge à exécuter.
- [x] C.8 (P0, 2026-09-13) Désynchronisation d'index ART sur `match_skill_rank` de JGtm,
      constatée au dry-run de la purge : `COUNT(*) FILTER (WHERE playlist_group='h5_arena')`
      (scan) rend 1 826 lignes, `WHERE playlist_group = 'h5_arena'` (lookup par
      `idx_msr_playlist`) n'en rend que 22. Donnée INTACTE, seuls les lookups mentent —
      même famille que `personal_score_awards` le 2026-08-27 (duckdb#23645). Livré :
      `cmd/repair_msr_index` (diag scan-vs-lookup par axe indexé + `-repair` DROP/CREATE
      avec la DDL capturée dans la base + CHECKPOINT, jamais de DELETE ni d'UPDATE) ;
      `purge_foreign_lusr_chain` durci (recensement et filtre CTAS par scan forcé
      `playlist_group || ''`, contrôle pré-vol lookup=scan qui REFUSE `-commit` et nomme
      `repair_msr_index`). Les 3 autres joueurs rendent des comptes cohérents.
- [x] C.7 Gate : `go build ./...`, `go vet ./...`, `go test ./...` (hors himap),
      `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/migration/... ./internal/platform/duckdb/...`
      (exit 0), `make go-api-lint`, push, CI verte.

## Lot D — D10 rejeu web (`feat/finitions-d10`, web)
- [ ] D.1 `features/match-replay/layers/objectivesLayer.ts` : les pulses ne sont construits que
      pour les familles d'objectif (zones, drapeau tant que non retiré, crâne, bombe, VIP) —
      jamais pour les frags ni les assistances (15 648 par image sur `8bc6074f`).
- [ ] D.2 Dénominateur de couverture du calque : le pourcentage ne compte plus les actions
      hors objectif ; un calque dont 99 % des actions sont des frags n'affiche plus « 100 % ».
- [ ] D.3 Tests vitest (familles filtrées, dénominateur), tsc (cache purgé), eslint 0 erreur,
      push, CI verte. Aucune string UI nouvelle sans FR+EN.

## Lot E — clôture (pilote)
- [ ] E.1 Fusions B, C, D dans feat/v75 (`-X theirs` inutile : branches courtes), CI verte.
- [ ] E.2 Purge C.5 exécutée sur les 4 bases Infinite (sauvegarde `data/backups/` datée,
      serveur arrêté, `-dry-run` puis `-commit`, recensement avant/après), serveur relancé.
- [ ] E.3 Revue adversariale finale (contexte frais) sur `git diff e372e5d28..feat/v75` hors
      `chore(deps)` ; seuls P0/P1 sont corrigés, le reste est consigné.
- [ ] E.4 PR Dependabot fermées, Notion : item « ≥ 01/10 retrait migration boot » coché,
      tâche « Deux lots à planifier » tracée ; thought_log ; suppression des worktrees.

## Découvertes (consignées, NON traitées)
- **P0 TRAITÉ (C.8)** — désynchronisation d'index ART sur `match_skill_rank` (JGtm) :
  détectée au dry-run de la purge, outillée le 2026-09-13 (`cmd/repair_msr_index` + pré-vol
  de la purge). La FAMILLE de défaut est confirmée au-delà de `personal_score_awards` : tout
  lecteur applicatif qui interroge `match_skill_rank` par prédicat indexé a pu servir des
  lignes amputées sur cette base. À re-sonder périodiquement — l'outil en `-dry-run` est le
  détecteur. Exécution de la réparation : pilote, serveur arrêté.
- **Lot C — `migration.RebuildMatchSkillRankART`
  (`internal/migration/steps_player_rebuild_match_skill_rank.go:71`) repose
  `ADD PRIMARY KEY (match_id)`** : c'est le schéma PRÉ-append-only. Sur une player DB
  d'aujourd'hui (PK technique `id`, N lignes par match_id) ce rebuild échouerait, et s'il
  passait il détruirait l'invariant append-only et la vue `_latest`. Son unique appelant
  est `cmd/force_rebuild_art`. À arbitrer : corriger (aligner sur la DDL de
  `steps_player_match_skill_rank.go`) ou supprimer avec son CLI.
- **Lot C — `Q24LUSRHistory` (`platform/duckdb/queries_career_encounters.go`) reste en
  lecture brute** : allowlist `TestNoRawAppendOnlyReads`, justification datée du
  2026-07-10 (décision B7 : `_latest` injecterait des valeurs d'échelle CSR dans un
  pipeline purement LUSR). Le résidu `h5_arena` y survit donc jusqu'à la purge C.5 ;
  après la purge, la question redevient théorique. Non traité (décision existante).
- **Lot C — le post-import OpenSpartan ne stampe le titre que pour l'étape LUSR** : les
  étapes `recomputePerfScores` (→ `GetPerformanceChain`, title-aware depuis `5be99a2c3`)
  et suivantes reçoivent encore le ctx brut. Même famille de défaut sur une autre colonne
  (`performance_chain`), non mesurée. Périmètre C.1 = LUSR seul.

## Journal
- 2026-09-13 : plan écrit ; lot A fait (commit deps + push).
- 2026-09-13 : **lot C (LUSR) CLOS** sur `feat/finitions-lusr`, 7 commits. C.1 → C.7 tous
  `[x]`. Deux constats « sur pièces » qui corrigent le plan : (1) la vue
  `match_skill_rank_latest` partitionne PAR `match_id` seul avec priorité CSR > LUSR >
  LUSR_V2 (et non par `(match_id, rating_type)` comme l'annonçait C.3) — la bascule reste
  bonne quant au principe (ne plus lire le brut) mais FAUSSE quant à la vue choisie —
  corrigé en C.3 bis, cf. entrée ci-dessous ; (2) le filtre de purge
  devait être `IS DISTINCT FROM` et non `<>` (un `<>` nu jette les lignes à
  `playlist_group` NULL). C.2 va au-delà du fail-loud : le moteur est désormais construit
  sur `deps.TitleSlug`, donc la double source de titre n'existe plus, elle est comparée.
  La PURGE des 4 bases reste à exécuter par le pilote (E.2).
- 2026-09-13 : **C.3 bis** — correction demandée par le pilote, sur pièces. Brancher le graphe
  d'évolution sur `match_skill_rank_latest` était un CHANGEMENT DE DONNÉES RENDUES, pas une
  correction : la vue arbitre CSR > LUSR par `match_id`, alors que le graphe trace deux séries
  (`CareerChartsSection.lusrEvolution.tsx:104-112`) et calcule ses deltas par
  `(rating_type, playlist_group)` — le point LUSR de tout match classé à double ligne
  disparaissait. La règle « les matchs classés affichent le CSR » est propre à Halo 5
  (`games/halo_5/livesync/csr_match.go`), pas au titre Infinite. Vue dédiée
  `match_skill_rank_latest_by_type` : une ligne par match ET par type, la plus récente — les
  lignes `h5_arena` supersédées restent masquées, aucun type n'est arbitré contre un autre.
  Effet de bord du lot : quatre fixtures de test de `platform/duckdb` recopient la DDL de la
  vue `_latest` au lieu de passer par les migrations (piège connu du dépôt) ; la vue par type
  y a été ajoutée en miroir, avec renvoi au nom du step.
- 2026-09-13 : **C.8 (P0)** — le dry-run de la purge sur JGtm a révélé un index ART
  désynchronisé (`idx_msr_playlist` : 22 lignes annoncées pour 1 826 réelles). Sans le
  durcissement, la purge aurait recensé 22 lignes étrangères puis fait rollback sur sa garde
  de cardinalité — le garde a fonctionné, mais il fallait remonter d'un cran : la purge lit
  et filtre désormais par SCAN FORCÉ, et refuse `-commit` tant que lookup ≠ scan.
  `cmd/repair_msr_index` répare l'index (DDL capturée dans la base, jamais recopiée).
