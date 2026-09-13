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
  décodeur, lot I). D2, D4, D5 : lots ultérieurs avec recuisson. D3, D6, D11, D14, D16, G5, G8,
  G9, H2, H3, H5 : NON / plus tard. D12 et D13 : CORRIGER (lot F, second message du 13/09).
- Retenus ici : D9, D10, D15, G2, G3, G4, G6 (ADR), G7, H1, H4, retrait de la migration boot
  des jetons (échéance 2026-10-01, critère tenu en prod), statut killpos (échéance 2026-11-08,
  critère tenu), lot PSA, lot LUSR.

## Lot A — Dependabot (pilote) — FAIT
- [x] A.1 Bumps sur feat/v75, gates verts, commit `chore(deps)`, push.
- [x] A.2 CI feat/v75 verte au niveau job (run 34752283478, 8 jobs verts, E2E skipped).
- [x] A.3 PR 79-83 fermées avec renvoi vers e63d89bc7 (13/09).

## Lot B — hygiène + PSA + jetons + killpos (`feat/finitions-hygiene`, Go/config/docs)

### B.1 PSA (garde data-health + rectification du numéro d'issue)
- [x] B.1.1 `git merge wt/psa-index-cause` (merge-tree propre vérifié le 13/09) ; déplacer
      `RAPPORT_VOLET2_INDEX_PSA.md` de la racine vers `.ai/V7.5/`. Les 5 fichiers
      `internal/migration/psa_index_repro*_test.go` restent derrière le tag `psarepro`.
- [x] B.1.2 Rectifier `#23046` -> `#23645` partout où le texte désigne le bug « Failed to delete
      all rows from index » : `CLAUDE.md` (L107), `docs/adr/0026-*.md` (titre + L12),
      `docs/adr/0030-*.md` (L15), `docs/WEAPONS.md`, `docs/FR/WEAPONS.md`, `docs/CHANGELOG.md`
      et `docs/FR/CHANGELOG.md` (une occurrence chacun), tous les `.go` sous `apps/go-api`
      (commentaires et messages de test, ~25 fichiers — `git grep -n 23046 -- '*.go'`).
      Ne PAS toucher `.ai/archive/`, `.ai/migrations/squashed/` ni les rapports datés.
- [x] B.1.3 `.ai/V7.5/REGISTRE_REPORTS.md` L538 : cause = amont duckdb/duckdb#23645 (ouverte,
      présente en 1.5.5), garde posée (alerte seule, 41 ms/base), condition de reprise =
      sortie d'une 1.5.6 contenant #24744, ou jauge `data_health_psa_index_desync_keys` > 0.
- [x] B.1.4 Gate : `go test -tags=integration -p 1 ./internal/scheduler/... ./internal/migration/...`
      (exit 0 vérifié) + `go test ./internal/scheduler/...`.

### B.2 Retrait de la migration boot des jetons legacy (ADR 0023, échéance 2026-10-01)
Critère tenu : prod `auth_migration: scan terminé` rt_migrated=0 à chaque boot du 2026-06-14
au 2026-09-13 (2 RT migrés le 2026-06-13, jamais depuis) ; local idem depuis le 2026-05-29.
- [x] B.2.1 Supprimer `internal/platform/auth/migration.go` + son test,
      `migrateLegacyAuthTokensAtBoot` + `legacyAuthSourcesReader` dans `cmd/server/main.go`,
      les helpers DuckDB de `internal/platform/duckdb/queries_auth.go` devenus orphelins,
      `auth.EnvRefreshTokenForGamertag` et tout appelant, les entrées d'allowlist de
      `internal/platform/auth/sentinel_test.go` (l'allowlist doit tomber à 0 entrée, pas être
      contournée), la référence dans `internal/ops/seed_demo_sync_meta.go`,
      `internal/sync/no_legacy_source_used_test.go` mis à jour (le garde reste, l'exception
      disparaît). Aucun import mort.
- [x] B.2.2 `CLAUDE.md` § « Règle auth tokens » : retirer le paragraphe « Seule exception legacy
      restante » (dater le retrait 2026-09-13, critère constaté). ADR 0023 : note de clôture
      de la Phase 5 (EN-only).
- [x] B.2.3 Gate : `go build ./...`, `go vet ./...`, `go test ./internal/platform/auth/...
      ./internal/sync/... ./cmd/server/...` + `-tags=integration -p 1 ./internal/sync/...`.

### B.3 Hygiène XS
- [x] B.3.1 D9 — socle central d'Illusion étiqueté équipe 0 au catalogue : localiser le
      catalogue (`film/replay/mapvar/objectives.go` ou le TOML de carte), corriger en
      « neutre », test de non-régression ; si l'item est déjà réglé, `[~]` + commit.
- [x] B.3.2 D15 — `ListMapsByTitle` (`platform/duckdb/metadata_repo_assets_list.go`) :
      dédoublonner par `map_asset_id` (le web le fait déjà par identifiant), test.
- [~] B.3.3 G2 — `cmd/replay-corpus-gate` : un témoin du corpus sans chunks en cache fait
      ÉCHOUER le gate avec le nom du film (aujourd'hui silencieux) ; test unitaire sur le
      manifeste.
- [x] B.3.4 G3 — `config/replay_corpus.toml` : la raison du témoin Oddball décrit un résidu
      fermé le 2026-09-08 ; réécrire la raison (ce que le témoin exerce aujourd'hui).
- [x] B.3.5 G4 — commentaires : `domain/replaydoc/coverage_objectives.go` /
      `film/replay/coverage_bridge.go` — l'invariant `Balanced()` est vrai par construction
      (protection réelle = un seul fermoir par portage) ; `closedBy*` baissent sur les films à
      `noTrack` (raison à écrire, pas le double comptage). Doc seule.
- [x] B.3.6 G6 — ADR 0026 : documenter `decode_pass` (colonne de `kill_positions`, passe de
      décodage, clé de la vue `_latest`). Si `seed_demo_corpus.go` porte encore un
      `UPDATE kill_positions` : le CORRIGER (décision utilisateur 13/09 : « pas de risques », pas
      d'allowlist). FAIT : UPDATE set-based interpolé -> N UPDATE ligne à ligne à valeurs liées
      (forme prescrite par le ratchet ; INSERT-only refusé sur un chemin d'anonymisation), test
      d'intégration ajouté (commit 044751026).
- [x] B.3.7 G7 — commentaire de `GroundWeapon.W` (`document_ground_weapon_items.go`) : espace
      de clés distinct de `Loadout.W` (rapport 6.6 découverte 1). Doc seule.
- [x] B.3.8 H1 — `MatchMetrics` (`internal/domain/stats.go`) : supprimer si aucun usage
      (`git grep -nw MatchMetrics`), sinon `[~]` ; `cmd/mapcallouts-build/decoupe_masque.go` :
      extension `.png` en constante nommée.
- [x] B.3.9 H4 — supprimer `cmd/weapon-sounds`, `cmd/vs-measure`, `cmd/vehicle-sprite` (git
      garde l'historique) ; retirer toute mention dans `docs/COMMANDS.md` (FR+EN) et
      `.ai/project_map.md`.
- [x] B.3.10 killpos — `.ai/V7.5/PLAN_LOT_PONT_ET_KILLPOSITIONS.md` items 2.3, 2.4, 2.6 : statuer
      `[~]` (critère du 2026-11-08 tenu : `BuildKillPositions` appelé par
      `sync/killcollector/positions.go:347`, écriture `persist/shared_persister.go` INSERT-only,
      `kill_positions_latest` = 114 038 lignes / 1 307 matchs Infinite au 2026-09-13) ; le CLI
      `cmd/killpos-build` est supersédé par `levelup backfill-killsource`. Le garde 88 %
      (`replay_local_gate.go`) reste à la main de l'utilisateur (Notion).
- [x] B.3.11 Gate lot B : `go build ./...`, `go vet ./...`,
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
- [x] C.9 (2026-09-13, après l'exécution d'E.2) Le swap de la purge ne restaurait qu'UNE
      vue : l'outil capturait le NOM `match_skill_rank_latest`, alors que C.3 bis en a ajouté
      une seconde (`match_skill_rank_latest_by_type`). Corrigé : capture de TOUTES les vues
      non internes dont le SQL référence `match_skill_rank` (filtre sur le SQL, jamais sur un
      nom), DROP + recréation de chacune avec résolution des dépendances, et garde de
      cardinalité sur les vues DANS la transaction (rollback si une manque). Mesure DuckDB
      consignée : `DROP TABLE` ne supprime PAS une vue dépendante — elle survit et se re-lie
      à la table renommée, ce qui MASQUE le défaut en bout de chaîne ; le garde porte donc sur
      la CAPTURE. `repair_msr_index` : test prouvant que DROP/CREATE INDEX ne touche aucune
      vue, plus un drapeau `-ensure-views` qui repose les vues avec la DDL des migrations
      (`halomigrations.EnsureMatchSkillRankViews`, CREATE OR REPLACE, aucune donnée touchée)
      — seule voie sûre quand le step est déjà inscrit au ledger.
- [x] C.7 Gate : `go build ./...`, `go vet ./...`, `go test ./...` (hors himap),
      `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/migration/... ./internal/platform/duckdb/...`
      (exit 0), `make go-api-lint`, push, CI verte.

## Lot D — D10 rejeu web (`feat/finitions-d10`, web)
- [x] D.1 `features/match-replay/layers/objectivesLayer.ts` : les pulses ne sont construits que
      pour les familles d'objectif (zones, drapeau tant que non retiré, crâne, bombe, VIP) —
      jamais pour les frags ni les assistances (15 648 par image sur `8bc6074f`).
      Tri par FAMILLE du nom de stat (`model/objectiveFamilies.ts`, dérivé des `ObjectiveType*`
      de `objectiveevents/extract.go`), pas par liste de statistiques. Parc du 13/09 (schéma 54,
      recuit depuis l'audit) : `8bc6074f` 119 -> 0 pulses par image (drapeau déjà retiré),
      `32d9a94f` 148 -> 55.
- [x] D.2 Dénominateur de couverture du calque : le pourcentage ne compte plus les actions
      hors objectif ; un calque dont 99 % des actions sont des frags n'affiche plus « 100 % ».
      SUR PIÈCES, le pourcentage n'est PAS côté web : `coverage.objectives` n'a aucun lecteur
      dans `apps/web` (grep `.coverage` complet) — il est produit par `buildObjectiveActions`
      (`internal/games/halo_infinite/film/replay/objectives.go`) et par les deux comptes de
      `replaybuild.identifiedEvents`. Les trois sont restreints aux familles d'objectif ;
      la PUBLICATION (`doc.Objectives`) ne perd rien (doctrine R1). Parc du 13/09 : `8bc6074f`
      218 -> 99 disponibles, `32d9a94f` 148 -> 55. Aucun champ ne bouge, `SchemaVersion`
      inchangé : les artefacts déjà cuits gardent l'ancien dénominateur jusqu'à recuisson.
- [x] D.3 Tests vitest (familles filtrées, dénominateur), tsc (cache purgé), eslint 0 erreur,
      push, CI verte. Aucune string UI nouvelle sans FR+EN.
      Gates du 13/09 : `tsc --noEmit` 0 ; `eslint .` 0 erreur / 31 avertissements préexistants ;
      `vitest run src/features/match-replay` 2 727 tests ; `vitest run` complet 7 389 tests ;
      `knip-ratchet` 0/0/0. Le dénominateur étant côté Go, gates Go ajoutés : `go build ./...`,
      `go vet` (3 paquets), `go test ./...` exit 0, `golangci-lint --new-from-merge-base` 0 issue.
      Aucune string UI ajoutée (le module de familles n'en porte aucune).

## Lot F — équipement : D12 mur et D13 champ de réparation (`feat/finitions-equipement`, Go, INSTRUCTION D'ABORD)
Décision utilisateur (13/09, second et troisième messages) : « jamais un équipement ne doit avoir
un événement sur une règle arbitraire ; le film dit s'il est utilisé ou déployé » et « on décode
mal l'événement d'apparition ; les développeurs du Theater ont un signal fiable ». Relu sur
pièces : ce qui est PROUVÉ est seulement que (a) la lecture en TÊTE de liste du type 103
`EquipmentSpawnedObject` (rapport R5) et (b) les créations `ti=37` dont le GlobalID est au
manifeste ne séparent pas déploiement et lâcher à la mort. Trois choses n'ont JAMAIS été faites :
1. marcher le 103 dans la LISTE COMPLÈTE (le marcheur de R7 existe) et RÉSOUDRE ses deux
   références objet (R5 §3.1 : ref0 et ref1 ≈ objets sur 13 bits, jamais résolues en handles
   d'entité) — si ref1 est l'équipement source et ref0 l'objet engendré, le 103 EST le fait
   « déployé » ;
2. le film voit bien plus d'apparitions que l'artefact n'en publie (6 651 ancres pour 295
   poses publiées sur `000d5950`) : les GlobalID non mappés des objets créés peuvent porter la
   forme DÉPLOYÉE du capteur, du traqueur et du champ de réparation (mesure E0 : 0 `spent` sur
   202 couvert par une pose connue — la pièce engendrée n'est peut-être simplement pas au
   manifeste) ;
3. les 90 têtes 103 « vers un lâcher » de R5 sont un appariement TEMPS SEUL à ±1,2 s sans
   référence résolue : l'affirmation « le 103 tire aussi à la mort » n'est pas établie.
- [x] F.0 Instruction (mesure, aucun code de production) : sur les 8 cas de D12 (identifiants
      dans l'audit §12 et le rapport de dette vague 5) et sur 3 films à oracle Theater connu
      (`000d5950`, `1cd3848a`, `215e7022`), marcher la liste complète, isoler chaque 103, résoudre
      ref0/ref1 contre les records de création `ti=37` et les handles d'unité ; recenser les
      GlobalID des objets créés dans les 2 s d'un `spent` de capteur/traqueur/champ. Verdict
      attendu par question : le 103 tire-t-il à la mort avec référence résolue (oui/non, compte) ;
      le déploiement d'un capteur/champ engendre-t-il un objet (GlobalID, compte) ;
      le 8/8 de D12 est-il séparé par le 103 (oui/non).
      **RENDU le 2026-09-13 : `.ai/V7.5/RAPPORT_F0_DEPLOIEMENT_103_2026-09-13.md`** (25 films,
      5 761 poses appariées, 931 occurrences du 103, instruments `filmdec/f0_103_*`).
      Verdicts : (1) `ref1` du 103 EST l'objet engendré — index 13 bits BASE 512 + génération,
      93,6 % de résolution contre 2,2 % au témoin de hasard, dt médian +49 ms, objets désignés
      `0x528fce46` ×227 et `0x686b40c9` ×3, c'est-à-dire les DEUX panneaux de mur du manifeste ;
      `ref0` désigne un `ti=37` de longue durée non identifié ; `ref2` est absente (3/931).
      (2) **Le 103 ne tire PAS à la mort** : 4 poses désignées sur 4 853 `dropped`, et 216 des
      217 poses désignées du parc sont des panneaux — l'affirmation « 90 têtes vers un dropped »
      de R5 §3.2 (appariement en TEMPS SEUL) est RÉFUTÉE. (3) **Mais il n'existe que pour la
      famille qui engendre une pièce** : 216/216 poses de panneau désignées, 0 sur les 91 poses
      `deployed` d'un déployable PORTÉ (mur 34, capteur 48, écran 4, traqueur 3, champ 2).
      (4) **Les 15 cas de D12 du parc recuit ne se tranchent donc pas sur le 103** — silence
      total pour l'appareil porté (0/31 `deployed`, 0/145 `dropped`) — ni par la voie INDIRECTE
      (une pièce voisine à ±5 s : 14,7 % sur les `deployed` contre 21,8 % sur les `dropped`).
      (5) D13 : aucune pièce engendrée pour le capteur, le traqueur ni l'écran (leurs fenêtres
      ne montrent que l'objet PORTÉ lui-même), et **le champ de réparation ne porte qu'UNE
      consommation exploitable dans les 76 artefacts du parc** — non mesurable. Témoin positif
      passé : le mur rend `0x528fce46` à ×20,3 d'enrichissement.
- [x] F.1 D12 — BRANCHE DE REPLI appliquée (F.0 a dit NON : le 103 est muet pour tout appareil
      porté). `equipmentOrigin` ne pose plus qu'une question TEMPORELLE : la clause de distance
      est retirée, le fait temporel reste. `originDropMaxDist` **survit au paquet** — trois
      autres chaînes s'en servent pour leur propre question (armes au sol, drapeau, crâne) —
      avec un commentaire qui dit pourquoi elle ne classe plus une pose d'équipement.
      `equipment_origin_test.go` : le cas « au bon instant mais trop loin » rend désormais
      `dropped`, un contrôle « loin ET après la fenêtre » borne la portée du changement, et la
      MUTATION est vérifiée (remettre la clause fait échouer le test sur exactement ce cas).
      **Mesure avant/après** (`f1_origine_{mesure,verdict}_research_test.go`, 25 films demandés,
      **21 mesurés**, 5 363 poses) : `deployed` 472 -> 450 (**-22**), `dropped` 4 509 -> 4 531
      (**+22**), `unknown` inchangé. **22 poses changent, toutes dans le même sens**, toutes à
      19,9-171,7 ms de la fin de vie et à 1,51-2,70 m : mur 7, grenade à fragmentation 10,
      grappin 2, propulseur 1, traqueur 1, grenade spike 1.
      **ÉCART AVEC L'ATTENDU, DIT : 22 et non 15.** Les 15 cas du §3.1 de F.0 étaient repérés
      par « t0 == dernière frame du poseur », un proxy à la granularité de 100 ms ; le critère
      RÉEL est la fenêtre de 200 ms, qui attrape 8 poses de plus (44 à 172 ms de la fin de vie,
      toutes sur `4f77afc1`). 14 des 15 sont bien dans les 22 ; le quinzième (`0797ce72`) est
      sur un film EXCLU de la mesure. Aucune famille hors de la liste du §3.1 n'est touchée,
      sauf une grenade spike lâchée au même instant que deux grenades du même poseur.
      **4 films hors mesure, dits** : `000d5950` et `215e7022` n'ont pas d'artefact local ;
      `0797ce72` et `c88ec007` (Aquarius) — aucune carte du catalogue ne reproduit leurs repères
      publiés (écart médian 29,2 et 29,9 m), cf. Découverte D-F5.
- [!] F.2 D13 — **NON TRAITÉ, et la mesure est la raison.** F.0 §4 : aucune pièce engendrée
      n'est identifiable pour le capteur (28 consommations), le traqueur (6) ni l'écran
      occultant (8) — leurs fenêtres ne montrent que l'objet PORTÉ lui-même (`0x4396db42` à
      ×14,0, `0x4744d742` à ×17,2), c'est-à-dire le même GlobalID des deux côtés ; et le CHAMP
      DE RÉPARATION ne porte qu'**UNE** consommation exploitable dans les 76 artefacts du parc,
      ce qui le rend non mesurable. Le témoin POSITIF passe (le mur rend `0x528fce46` à ×20,3),
      donc le négatif est ancré. **Rien n'est ajouté au manifeste ni à
      `usageFamiliesWithSpawnedPiece`** : y inscrire une famille sans pièce ferait relire son
      « utilisé » sur `DeployedByFamily`, le défaut exact que `us6` a corrigé le 2026-09-10.
      Reprise : un parc portant plusieurs dizaines de consommations de champ de réparation.
- [x] F.3 **`SchemaVersion` NON bumpé (reste 54)** : seule la CLASSIFICATION change, aucun champ
      ne bouge — les artefacts déjà cuits gardent leur ancienne origine jusqu'à recuisson, ce qui
      est la même règle que D.2 du lot D. Gates joués :
      `go build ./...` 0 · `go vet ./internal/games/halo_infinite/film/...` 0 ·
      `go test` sur `film/replay` et `film/filmdec` 0 ·
      `golangci-lint --new-from-merge-base=origin/feat/v75 ./internal/games/halo_infinite/film/...`
      **0 issue** · gofmt propre.
      **`replay-corpus-gate --reference=base --base origin/feat/v75` en RACINE JETABLE**
      (`--work-root <worktree>/.f3-work`, `--parc-root` = dépôt principal en LECTURE ; le verrou
      de décodage reste celui du parc, c'est sa raison d'être — un verrou local laisserait deux
      décodeurs se marcher dessus). 12 témoins, 24 cuissons. **Sortie 1, et c'est ATTENDU :
      les 4 témoins « PERTE » ne perdent que des compteurs `deployed`, avec autant de GAINS que
      de pertes (3/3, 4/4, 2/2, 2/2)** — c'est-à-dire la reclassification elle-même, comptée
      comme une baisse par un gate qui ne sait pas qu'elle est voulue. Détail :
      `d9781168` `placements.deployed` 16 -> 14 (grenade frag 13 -> 12, spike 2 -> 1) ;
      `51ebbc0f` 17 -> 13 (plasma 2 -> absent, frag 9 -> 8, répulseur 5 -> 4) ;
      `084a804d` 53 -> 51 (frag 23 -> 21) ; `0797ce72` 34 -> 33 (**mur 21 -> 20**).
      **Ce dernier CORROBORE la mesure F.1** : `0797ce72` est le film que l'instrument avait dû
      exclure faute de carte — le gate, qui prend la carte de la base du match, y voit bien le
      cas de mur basculer. Les 15 cas de F.0 §3.1 sont donc tous couverts.
      Aucun autre axe du gate (identité, calques, faits) ne bouge : **0 perte hors
      `coverage.placements.*`**.
- [!] F.4 Recuisson du parc : **NON LANCÉE** — décision de l'utilisateur (22 min de coupure).
      Tant qu'elle n'a pas tourné, les artefacts du parc gardent l'ancienne origine ; la vue
      match la recalcule à la volée, Sessions/Solo/Escouade non.

## Décisions complémentaires (13/09, second message)
- D3 : IGNORÉ (assez de cartes dessinées). D11 : rien (l'API sert la donnée). D16 : refonte
  du décodeur. G5 : plus tard. G6 : CORRIGER l'UPDATE (pas d'allowlist) — transmis au lot B.

## Lot E — clôture (pilote)
- [x] E.1 Fusions dans feat/v75 : B `279b5c835`+`1f4e55ead`, D `56e425c8e`, C `52828bc05`+`a16e78c28` (C.8/C.9), F `0f96d42e2` — CI verte au niveau job après chaque fusion.
- [x] E.2 (13/09) index `idx_msr_playlist` de JGtm réparé (repair_msr_index, 35 502 lignes intactes) ; purge commitée sur les 4 bases : 1 826 / 2 128 / 942 / 62 lignes `h5_arena` retirées, 0 après, 3 index reposés ; sauvegarde `data/backups/2026-09-13_purge_lusr_h5_arena/` ; serveur relancé (200).
- [x] E.3 Revue adversariale finale (contexte frais, `.ai/V7.5/REVUE_FINITIONS_2026-09-13.md`) : 1 P0 (R1 : vue `match_skill_rank_latest_by_type` absente d'une player DB créée par `EnsurePlayerSchema` seul — page Carrière Halo 5 et onboarding en Catalog Error) CORRIGÉ dans `sync/schema.go` + garde `schema_msr_views_test.go` ; 0 P1 ; 14 P2 : R2-R5 (numéros d'issue et phrases d'ADR inversés par le lot lui-même) corrigés comme défauts du lot, R6-R11, R14, R15 consignés ci-dessous, R12/R13 = clôture.
- [x] E.4 PR 79-83 fermées ; worktrees et branches des lots supprimés ; thought_log ; Notion NON touché (carnet utilisateur : l'item « ≥ 01/10 retrait migration boot » est à cocher par l'utilisateur, fait le 13/09).

## Découvertes (consignées, NON traitées)

### Lot B (2026-09-13)

- **D-B1 — Le ratchet anti-ART est aveugle aux noms de table INTERPOLÉS.**
  `internal/sync/no_art_patterns_test.go` scanne des LITTÉRAUX. `internal/ops/seed_demo_corpus.go`
  construisait ses écritures par `fmt.Sprintf("UPDATE %s …")` : aucun littéral `UPDATE weapon_kills`
  n'existait dans la source, donc ni `TestNoBulkMultiRowUpdateOnCriticalTables` (qui couvre pourtant
  `weapon_kills`, `medals_earned`, `killer_victim_pairs` et `match_participants`, tous présents dans
  cette boucle) ni le scan principal ne pouvaient le voir. Le cas précis est corrigé en B.3.6, mais
  **le trou du garde-rail reste** : toute écriture à table interpolée échappe encore aux deux tests.
  Traitement possible : détecter la FORME `UPDATE %s` / `UPDATE " + table` en plus des littéraux, ou
  une analyse AST. NON TRAITÉ.
- **D-B2 — 8 socles `flag_spawn` d'équipe portent `team_index = -1` au catalogue d'objectifs**
  (Cliffside, Highpower Heavies, Solitude, Solitude - Ranked, plus 4 entrées sans `public_name`).
  C'est le défaut SYMÉTRIQUE de D9 : là où Illusion déclarait neutre un socle d'équipe, ces cartes
  déclarent « sans équipe » des socles qui en ont une. `flag_neutral.go` triant sur
  `Team == TeamNeutral`, ces socles tombent dans le panier NEUTRE et pourraient faire basculer à tort
  un film en variante « drapeau neutre » (il faut `neutralBirths >= 3` et strictement plus que les
  naissances d'équipe — donc pas automatique, mais possible). La correction de B.3.1 ne les touche
  pas : elle porte sur le label, et ces socles n'en ont pas. Traitement propre : porter la neutralité
  en champ EXPLICITE de `FlagSpawn` plutôt que de la surcharger sur `Team`. NON TRAITÉ.
- **D-B3 — `cmd/weapon-sounds` et `cmd/vehicle-sprite` ne devaient PAS être supprimés**, contrairement
  à ce que l'arbitrage H4 indiquait — et l'annexe G9 de l'audit du 2026-09-05, sa propre source, le
  disait déjà (constats P1-2 et P1-4). Ce sont les seuls producteurs de 197 fichiers d'assets
  VERSIONNÉS servis en production (20 sprites de véhicules lus par `useReplayVehicles.ts`,
  177 sons sous `static/sounds/halo_infinite/`). Seul `vs-measure` était jetable et a été supprimé.
  Traité dans B.3.9 et statué au registre ; consigné ici parce que l'écart vient de l'arbitrage.
- **D-B4 — Deux `//nolint` ne suppriment probablement rien.** `golangci-lint` avertit :
  « Found unknown linters in //nolint directives: 2026-09-07), gosec — limit/placeholders
  maîtrisés, plr0913 — clé canonique d'issue (d5, plr0913 — coordinator function ». Les
  directives de `internal/service/match_view_builders_summary.go` et
  `match_view_builders_team.go` (commit `898bb3084`, hors de ce lot) mettent une justification
  en PROSE là où le linter attend des noms de linters, et il parse donc la phrase comme tels.
  Conséquence : la suppression demandée n'a probablement pas lieu. Forme correcte :
  `//nolint:gosec,plr0913 // justification`. Le lint sort tout de même en 0 issue (dette gelée).
  NON TRAITÉ.

- **Lot D** — `features/match-replay/model/objectiveMark.ts` garde sa table `EVENT_STATS`
  (statistiques nommées une à une : `zone_captures`, `zone_secures`, `bomb_detonations`). Elle
  répond à une autre question que le prédicat de famille (quelle MARQUE de fiche pour quel geste,
  pas « est-ce un objectif ») et n'a pas été migrée.
- **Lot D** — `apps/go-api/internal/replaybuild/matchfacts.go` passe de 501 à 504 lignes : il
  était déjà au-dessus du seuil de 500 avant ce lot, et l'extraction serait un refactor hors
  périmètre.
- **Lot D** — les artefacts DÉJÀ CUITS gardent l'ancien dénominateur de `coverage.objectives` :
  le correctif D.2 ne se voit qu'à la recuisson. Aucun champ du document ne bouge,
  `SchemaVersion` reste à 54 — la recuisson est donc une décision de fraîcheur, pas de contrat.

### Lot F.0 (2026-09-13)

- **D-F1 — 7 poses de PANNEAU de mur du parc sont classées `dropped` (3) ou `unknown` (4)**,
  alors qu'un panneau n'existe qu'une fois déployé et ne peut donc pas être lâché à la mort.
  C'est le défaut SYMÉTRIQUE de D12 — même cause (`equipmentOrigin` ne pose qu'une question
  temporelle), sens inverse. 7 sur 216 panneaux. À traiter AVEC D12, pas à part. NON TRAITÉ.
- **D-F2 — `ref0` du type 103 désigne un `ti=37` que les images-clés voient (737 sur 739
  recensées) et que les paquets delta ne créent JAMAIS** (2,8 % de résolution). Ce n'est pas
  l'appareil porté. Si c'est l'entité qui ENGENDRE la pièce, elle ouvrirait l'identification de
  l'équipement SOURCE d'un déploiement — sujet du lot I (refonte du décodeur). NON TRAITÉ.
- **D-F3 — la sélectivité de l'en-tête NEW `ti=37` s'effondre sur les films BTB** : 2 263
  records acceptés hors manifeste sur 3 185 (71 %) sur `4f77afc1` et 927 sur 1 605 (58 %) sur
  `5676a9ba`, contre ~25 % sur un film d'arène. Aucune mesure du dépôt ne borne aujourd'hui ce
  taux pour l'équipement (la borne connue vaut pour les ARMES au sol). NON TRAITÉ.
- **D-F5 — DEUX films d'Aquarius ne se raccrochent à aucune carte du catalogue.** `0797ce72`
  et `c88ec007` portent le découpage d'i0 `[13 12 11]`, celui d'`aquarius` et de lui seul, mais
  aucune entrée de cette classe ne reproduit les repères que LEURS PROPRES ARTEFACTS publient :
  écart médian de **29,2 m** et **29,9 m** par piste, quand les 21 autres films tiennent sous
  0,20 m. Soit les bornes `aquarius` du catalogue ont changé depuis la cuisson de ces deux
  artefacts, soit ces artefacts sont antérieurs à une correction de bornes. Les deux films sont
  sortis de la mesure F.1 plutôt que mesurés en mètres faux. NON TRAITÉ.
- **D-F4 — `0x412000aa` est désigné une fois par un `ref1` de 103** sur `9e8fb31b`, à
  +19 386 ms — très probablement une collision de clé `(slot, génération)` rebouclée, mais
  l'identifiant est hors manifeste et n'a pas été instruit. NON TRAITÉ.


- **C.9 — diagnostic pour le pilote** : sur les 4 bases purgées, le recensement d'après ne
  compte qu'UNE vue `match_skill_rank_latest%`. Or il est MESURÉ qu'une vue dépendante survit
  au `DROP TABLE` du swap et se re-lie à la table renommée : si
  `match_skill_rank_latest_by_type` avait existé, elle serait encore là et le compte serait 2.
  Le compte de 1 dit donc que la vue n'existait pas — la migration
  `player_msr_view_latest_by_type_v1` (née en C.3 bis) n'avait pas encore été jouée sur ces
  bases. Rien n'a été perdu ; elle sera créée au prochain boot. Vérification directe :
  `SELECT name FROM schema_migrations WHERE name = 'player_msr_view_latest_by_type_v1'`
  (absente = confirmé). Si elle y figurait, `repair_msr_index -ensure-views` la repose.
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
- 2026-09-13 : lot B.1 clos (`feat/finitions-hygiene`) — merge `wt/psa-index-cause` (garde data-health PSA, 5 reproducteurs derrière `psarepro`), rapport déplacé en `.ai/V7.5/RAPPORT_VOLET2_INDEX_PSA_2026-08-28.md`, `#23046` -> `#23645` sur 133 fichiers Go + 6 docs + CLAUDE.md (196 occurrences Go), registre L538 réécrit (cause amont prouvée, garde alerte-seule 41 ms/base, condition de reprise = 1.5.6 avec #24744 ou jauge > 0). Gates : `go build ./...` exit 0, `go test ./internal/archlint/... ./internal/scheduler/...` exit 0, `go test -tags=integration -p 1 ./internal/scheduler/... ./internal/migration/...` exit 0.
- 2026-09-13 : lot B.2 clos — migration one-shot des jetons legacy RETIREE (ADR 0023 Phase 5 close, en avance sur l'echeance 2026-10-01, critere tenu). Supprimes : `auth/migration.go` + test, `migrateLegacyAuthTokensAtBoot`/`legacyAuthSourcesReader` + `cmd/server/migration_boot_test.go`, `platform/duckdb/queries_auth.go` + son test d'integration. Allowlists sentinel 2 et 3 a 0 entree (ratchets anti-resurrection) ; guard 1 garde la seule entree `capturecli.go` (stdin, pas d'environnement). `no_legacy_source_used_test.go` inchange : il n'a jamais porte d'exception. Docs : CLAUDE.md, ADR 0023 (section « Cloture de la Phase 5 »), `ops/seed_demo_sync_meta.go`, `groupstore/migrate.go` (reference morte). Gates : `go build ./...` 0, `go vet ./...` 0, `go test ./internal/platform/auth/... ./internal/sync/... ./cmd/server/... ./internal/ops/... ./internal/platform/duckdb/...` 0, `go test -tags=integration -p 1 ./internal/sync/...` 0.
- 2026-09-13 : lot B.3 clos. **B.3.1 (D9)** corrige a la SOURCE : le socle central d'Illusion est neutre par son LABEL (`ctf_neutral_include`), pas par son `team_index` qui vaut 0 — recensement du catalogue : sur 63 socles neutres le label est juste 63 fois, le team_index 62. `mapvar.Objective.IsCTFNeutral` + `PointObjective.Neutral` (la projection laissait tomber `Labels`) + les deux lecteurs (`replaybuild.flagSpawnTeam`, `BuildMapObjectives`) ; 3 tests, mutation verifiee (le retrait du correctif fait bien echouer le test). **B.3.2 (D15)** dedup par `map_asset_id`, requete enveloppante pour garder le tri d'affichage, test des homonymes. **B.3.3 (G2)** `[~]` : deja livre par `128ae9d15` (CORPUS-R1 C3) — `verifierCouverture` (report.go:149) fait sortir le gate en 2 en nommant chaque temoin absent et sa cause, appele depuis main.go:319, 4 tests qui ne touchent pas le cache reel. **B.3.4 (G3)** raison du temoin Oddball reecrite (residu ferme au lot 6.2 le 2026-09-10 — defaut REFUTE, pas corrige). **B.3.5 (G4)**, **B.3.7 (G7)** doc seule. **B.3.6 (G6)** ADR 0026 : `decode_pass` documente comme 4e mecanisme (le seul qui sait RETRACTER) + les 3 pieges ; l'`UPDATE kill_positions` de la demo EXISTAIT (invisible au grep : nom de table interpole) et a ete corrige en N UPDATE row-by-row a valeurs liees — PAS en INSERT-only, qui aurait laisse les xuid REELS dans une base publiee ; test d'anonymisation ajoute (la fonction n'en avait aucun). **B.3.8 (H1)** `MatchMetrics` supprime ; le `.png` de `decoupe_masque.go` est `[~]` (deja porte par `PathResolver.MapBackgroundPath`). **B.3.9 (H4)** SEUL `vs-measure` supprime : `vehicle-sprite` et `weapon-sounds` produisent 197 assets versionnes servis en production (cf. Decouverte D-B3). **B.3.10** echeance killpos soldee par anticipation. Gates : build 0, vet 0, `go test` hors himap 0 (170 paquets), `-tags=integration -p 1` sur sync/persist/migration/duckdb/ops/scheduler 0 (19 paquets), golangci-lint `--new-from-merge-base=origin/main` **0 issue**, gofmt propre.
- 2026-09-13 : B.3.11 — branche `feat/finitions-hygiene` poussee (5 commits + le merge PSA). PIEGE RENCONTRE, a savoir pour les lots C et D : dans un worktree FRAIS, le hook de pre-push `knip-ratchet` rend 197 exports / 168 types morts contre un plafond de 0 et BLOQUE le push — c'est un FAUX POSITIF, knip ne sait pas resoudre les imports sans `node_modules`. Remede : `cd apps/web && npm ci` dans le worktree, apres quoi le ratchet rend 0/0/0. Ne PAS relever le plafond, ne PAS passer `--no-verify`.
- 2026-09-13 : **CI VERTE sur `feat/finitions-hygiene` (`b566823cb`, run 34756564137)** — 8 jobs verts, E2E React skippe par conception, plus Deploy Pre-Check et gitleaks verts. Un incident en chemin, repare : le job « Go Coverage + Baseline non-regression » a rougi deux fois, cause MIENNE et pas un flake. La suite Go etait verte (`go test` exit=0) ; c'est le controle de PRESENCE de `check_test_baseline.sh` qui echouait — B.2 a supprime 23 tests avec le code de la migration des jetons, or `.ai/baselines/tests_pre_migration.jsonl` est un CUMUL qui continuait de les exiger. Retrait chirurgical des 23 entrees (120 lignes JSONL), verifie par difference des paires (Package, Test), jamais une re-capture complete. **LECON POUR LES LOTS C ET D : tout test supprime doit l'etre AUSSI de la baseline, dans le meme commit — sinon la CI rougit sur le job le plus long (~23 min de boucle de retour).**
- 2026-09-13 : lot D (D10) clos sur `feat/finitions-d10`, 3 commits. D.1 et D.2 reposent sur le
  même fait — `doc.objectives` porte `kills` et `assists` (ancre d'identité du balayage et
  contrôle croisé), et le seul discriminant publié est le NOM de la statistique. Liste blanche de
  FAMILLES des deux côtés (`model/objectiveFamilies.ts`, `objectiveevents/families.go`), dérivée
  des `ObjectiveType*`, avec garde-rail Go `TestStatsNommeesPortentLeurFamille`. D.2 s'est révélé
  Go et non web (aucun lecteur de `coverage.objectives` dans `apps/web`) : correctif dans
  `buildObjectiveActions` + les deux comptes de `replaybuild.identifiedEvents`, publication
  inchangée (doctrine R1). Mesures : `8bc6074f` 119 -> 0 pulses/image et 218 -> 99 disponibles,
  `32d9a94f` 148 -> 55 des deux côtés.
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
- 2026-09-13 : **F.0 rendu** (`feat/finitions-equipement`, commit `a88504d97`, push, CI). Instruction
  pure : aucun code de production, aucune DuckDB, aucune écriture sous `data/`. Cinq instruments sous
  gardes d'environnement (`filmdec/f0_103_*`) + un changement ADDITIF au marcheur de R7 (`r7Ev.Refs`
  porte les trois références et leur génération ; marche inchangée bit pour bit, contrôle 97,9 % de
  fins propres). Corpus : les 10 films imposés + 15 choisis par le recensement du parc d'artefacts
  (`TestF0CorpusSpent`), parce que les 10 imposés ne portaient AUCUNE consommation de charge de champ
  de réparation. Rapport : `.ai/V7.5/RAPPORT_F0_DEPLOIEMENT_103_2026-09-13.md`. Gates : `go build
  ./...` 0, `go vet ./internal/games/halo_infinite/film/...` 0, `go test` sur les 8 paquets `film/*`
  0, `golangci-lint --new-from-merge-base=origin/feat/v75` **0 issue**. Piège rencontré, déjà connu du
  lot B : `knip-ratchet` bloque le push d'un worktree frais (197/168 contre un plafond de 0) faute de
  `node_modules` — `npm ci` dans le worktree, puis 0/0/0. `.golangci-cache*/` ajouté au `.gitignore`
  (même raison que `.gocache*/` : cache GLOBAL par défaut, à isoler par worktree).

### Revue finale E.3 (2026-09-13) — P2 consignés, NON traités
- R6 — listes de familles d'objectif Go (`objectiveevents/families.go`) et TS (`model/objectiveFamilies.ts`) indépendantes, aucun test de parité ; le TSDoc prétend le contraire. Une 7e famille rendrait le Go rouge et le TS silencieusement muet.
- R7 — `infiniteLUSRChains` (gate d'intégration I14) recopiée, sans test d'exhaustivité contre `skillchain/classify.go`.
- R8 — `purge_test.go` ne prouve pas la restauration du DEFAUT `written_at` (mutation verte).
- R9 — précondition de disjonction DemoXUID / SourceXUID de `applyUniversalAnonymization` non assertée ni testée.
- R10 — `cmd/repair_msr_index/main.go` bâtit le chemin player à la main (`halo_infinite` et 4 gamertags en dur) au lieu de `PathResolver` ; 2e copie du littéral (1re : `repair_psa_index`).
- R11 — dette de taille non consignée : `equipment_placements.go` 594 -> 628 L ; `buildSyncEngineFactoryParityComplete` 85 -> 110 L.
- R14 — `ListMapsByTitle` : `COALESCE(name_canonical,'')` fait passer les cartes sans nom canonique en tête du tri (le commentaire dit « tri inchangé »).
- R15 — deux formulations imprécises : référence équipement §1 (« de la dernière position » -> « de la fin de vie ») ; godoc `originDropMaxDist` (le crâne n'est qu'un test).

## Lot G — fiabilité (`feat/finitions-fiabilite`, Go) — arbitré le 13/09 soir (points 1 à 5 des recos)
Contrat `plan-execution`, périmètre FERMÉ, découvertes consignées non traitées sauf P0.
- [x] G.1 Supprimer `migration.RebuildMatchSkillRankART` (`internal/migration/steps_player_rebuild_match_skill_rank.go`) et son unique appelant `cmd/force_rebuild_art`, avec tests, imports, mentions docs (`docs/COMMANDS.md` FR+EN, `.ai/project_map.md`) et entrées de baseline de tests retirées dans le MÊME commit. Si un autre appelant existe, statuer `[!]` avec preuve.
- [ ] G.2 `internal/sync/no_art_patterns_test.go` : détecter aussi les écritures à nom de table INTERPOLÉ (`fmt.Sprintf("UPDATE %s`, `"UPDATE " + table`, `DELETE FROM %s`, `INSERT INTO %s … ON CONFLICT`) sur les tables protégées ; le cas de `seed_demo_corpus.go` (forme ligne à ligne à valeurs liées) doit rester VERT ; un cas témoin rouge (fixture de test) prouve la morsure ; aucune allowlist agrandie.
- [ ] G.3 Sonde data-health « index désynchronisé » étendue à `match_skill_rank` des player DB (`internal/scheduler/`), calquée sur `data_health_psa_index.go` (alerte seule, `OpenReadForQuery`, échantillon borné, jauge gelée si non mesuré, entrée dans `WarningsTotal`, title-agnostic, coût mesuré). Réutiliser la règle de comparaison de `cmd/repair_msr_index/diag.go` plutôt que la recopier (extraire un helper partagé si besoin, avec garde-rail ≤ 2 copies). Le message d'alerte nomme `repair_msr_index -repair`.
- [ ] G.4 `internal/service/openspartan_post_import_service.go` : le titre de la base est stampé UNE fois à l'entrée du post-import, pour TOUTES les étapes (LUSR, `recomputePerfScores` → `GetPerformanceChain`, suivantes) ; test : un ctx entrant portant un autre titre ne change pas la chaîne de performance écrite. Mesure : requête lecture seule sur les 4 player DB Infinite (serveur ARRÊTÉ par le pilote, pas par l'agent) pour compter les `performance_chain` étrangères — l'agent livre la requête, le pilote l'exécute.
- [ ] G.5 Tests de parité : (a) Go ↔ TS des familles d'objectif (`objectiveevents/families.go` ↔ `features/match-replay/model/objectiveFamilies.ts`, ratchet Go qui lit le fichier TS, modèle `archlint/*_test.go`) ; (b) `infiniteLUSRChains` du gate d'intégration dérivée de `games/halo_infinite/skillchain` (export d'une liste ou test d'exhaustivité contre `ClassifyLUSRChain`), plus de copie manuelle.
- [ ] G.6 Gates : build, vet, `go test` hors himap, `-tags=integration -p 1` sur sync/persist/migration/duckdb/scheduler/service, lint, baseline de tests, push, CI verte au niveau job.

## Lot H — rejeu (`feat/finitions-rejeu`, Go + config) — points 6 à 8 des recos
- [ ] H.1 D-B2 : neutralité des socles de drapeau en champ EXPLICITE de `FlagSpawn` (plus surchargée sur `Team`), posée depuis le label (`IsCTFNeutral`) ; les 8 socles à `team_index = -1` (Cliffside, Highpower Heavies, Solitude, Solitude - Ranked + 4 sans nom) ne tombent plus dans le panier neutre ; `flag_neutral.go` lit le champ ; tests (dont un film qui basculait à tort en variante neutre si mesurable, sinon test unitaire du tri).
- [ ] H.2 D-F1 : une pose d'une PIÈCE ENGENDRÉE (objets `kind = "deployed"` au manifeste, ex. panneaux de mur) est TOUJOURS `deployed`, jamais `dropped`/`unknown` (`equipment_placements.go`) ; test ; mesure avant/après en racine jetable (attendu : 7 panneaux basculent, rien d'autre).
- [ ] H.3 D-F5 : dater les artefacts `0797ce72` et `c88ec007` (Aquarius) et la dernière modification des bornes `aquarius` du catalogue ; conclure lequel est périmé ; si ce sont les artefacts, les recuire TOUS LES DEUX SEULEMENT (autorisation utilisateur donnée le 13/09 pour ces deux films, jamais le parc) via la commande de cuisson canonique, serveur ARRÊTÉ par le pilote ; vérifier l'écart aux repères < 0,20 m après.
- [ ] H.4 Gates : build, vet, `go test` replay + filmdec + replaybuild, `replay-corpus-gate` en racine jetable, lint, push, CI verte.

## Lot I — clôture (pilote)
- [ ] I.1 Fusions G et H, CI verte ; revue adversariale bornée (P0/P1 seuls) ; thought_log ; worktrees supprimés.
