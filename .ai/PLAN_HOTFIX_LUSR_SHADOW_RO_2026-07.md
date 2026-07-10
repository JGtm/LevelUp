# PLAN HOTFIX — LUSR v2 shadow écrit sur un attach read-only (prod, régression 2026-07-03)

> Statut : PRÊT — exécutable par une session agent autonome (Opus), AUCUN contexte
> conversationnel requis : tout est dans ce fichier. Exécution sous contrat du skill
> `plan-execution` (ordre strict, une phase à la fois, gates exacts, statuts
> [x]/[~]/[!], zéro fix hors périmètre — consigner en §Découvertes).
>
> **Branche : `hotfix/lusr-shadow-ro` créée depuis `origin/main`** (PAS depuis
> `refactor/audits-2026-07` — la branche d'audits n'est pas mergée et son arborescence
> diffère : le lot K y a déplacé le code skill vers `internal/sync/skill/` ; sur main
> tout est encore à plat dans `internal/sync/`). TOUJOURS vérifier les cibles sur
> pièces dans la branche de travail avant d'éditer (règle plan-execution n°4).
>
> **Push sur main = déploiement prod AUTOMATIQUE** : le merge/push final exige le GO
> explicite du user dans le tour courant (H5). Tout le reste est autonome.

## 1. Contexte incident (autoporteur)

- Prod (VPS `ssh lvelup`, conteneur `levelup-levelup-1`, logs hôte
  `/opt/levelup/data/logs/`) : depuis le **2026-07-03**, ~6 500 WARN/jour
  `LUSR v2 shadow: persist état échoué — watermark non avancé` avec
  `err = persist owner-only: SkillV2Repo.UpsertState(<xuid>, <group>): Invalid Input
  Error: Cannot execute statement of type "INSERT" on database "shared_matches_v2"
  which is attached in read-only mode!` — groupes Halo Infinite ET Halo 5.
- Effets mesurés (2026-07-07, post-reboot) : ~280 WARN/h ; watermark LUSR figé depuis
  4 jours (aucune ligne LUSR récente) ; boucle de retry à chaque cycle ;
  `sharedprovider: writer RW tenu au-delà du seuil` ×150/4 h ; lectures shared gatées ;
  **`GET /health` répond 503 par intermittence** (×44/4 h).
- Déclencheur : déploiement main du 2026-07-02 soir — refactor « étape 1 contention /
  bursts paresseux » (`b34724a7f` et commits liés, cf. `.ai/PLAN_POSTSYNC_BURST_LEASE.md`).

## 2. Cause racine (établie sur pièces le 2026-07-07)

`engine_postsync_scoring.go` (sur main, ~L135-146) : `runSkillRatingSteps` classe TOUT
le bloc LUSR en **segment lecture** :

```go
// Étape 1 contention : LUSR LIT shared (repos SkillV2/SquadOffset) et écrit
// la PLAYER DB (match_skill_rank, skill_v2_shadow.go:69) — segment Read.
sharedDB, releaseRead, rerr := shared.Read(ctx)
```

Ce commentaire n'est vrai que pour LUSR **v1** (`batchComputeLUSR` : lit shared, écrit
player DB). Le **v2 shadow** (`RunLUSRV2ShadowOwnerOnly` → `runLUSRV2Shadow` →
`persistComputedMatchSkillV2` → `repo.UpsertState`) écrit `player_skill_state_v2`
**côté SHARED** (cf. doc de `runLUSRV2Shadow` : « v2 … écrit uniquement dans
`player_skill_state_v2_latest` côté shared »). En mode burst (défaut prod), `Read`
sert un handle RO → tout INSERT shared échoue. Erreur de CLASSIFICATION du refactor
contention ; le shadow lui-même est sain.

Architecture cible déjà en place : les étapes post-sync qui écrivent shared
(convergence events, weapon kills — `engine_postsync.go` ~L274-288) utilisent des
**bursts `shared.Write(ctx, "<step>")` courts et chunkés**
(cf. `postsyncEventsBurstChunk`). Le fix = amener le shadow LUSR sur ce pattern.

API `SharedAccess` (`internal/sync/shared_access.go`) :
- `Read(ctx) (*sql.DB, func(), error)` — RO, ne gate personne.
- `Write(ctx, step string) (*sql.DB, func(), error)` — burst RW court, labellisé.
- **GARDE ANTI-DEADLOCK** : `Write` REFUSE si un `Read` du même SharedAccess est
  encore en vol → toujours release le Read AVANT de demander un burst Write.
- `NewPinnedSharedAccess(db)` : mode « handle déjà tenu » (Read/Write retournent le
  même handle, release no-op) — pour les callers CLI/backfill qui possèdent déjà le
  writer, et pour les tests.
- Échappatoire rollback : env `LEVELUP_POSTSYNC_BURST=0` → mode pinned historique
  (writer tenu tout le pipeline ; les écritures shared re-marchent, la contention
  revient). C'est le plan de repli post-deploy, PAS le fix.

## 3. Décisions figées (ne pas re-débattre en cours d'exécution)

- **DC-H1** : fix pérenne conforme burst-lease, PAS de revert du refactor, PAS de
  passage du bloc entier en `Write` (re-tiendrait le writer pendant le calcul v1+v2,
  annulant le gain validé : 255 ms max, 0 watchdog).
- **DC-H2** : sur main (même package `sync`), les signatures de
  `RunLUSRV2Shadow` / `RunLUSRV2ShadowOwnerOnly` passent de
  `(ctx, playerDB, sharedDB *sql.DB, xuid)` à `(ctx, playerDB *sql.DB,
  shared *SharedAccess, xuid)`. AUCUNE variante de compat conservée (le paramètre
  `*sql.DB` brut est le footgun qui a causé l'incident) ; tous les callers migrent
  dans le même commit : engine (`engine_postsync_scoring.go`),
  `lusr_full_recompute.go`, `cmd/lusr_v2_replay`, callers H5 livesync (chemin main —
  vérifier sur pièces), tests (~20 sites → `NewPinnedSharedAccess(db)`).
- **DC-H3** : découpage interne de `runLUSRV2Shadow` : sélection
  (`loadShadowMatches`) sous segment **Read, release immédiat** ; puis boucle
  `processOneShadowMatch` par **chunks sous burst `Write(ctx, "lusr")`** (constante
  nommée `postsyncLUSRBurstChunk`, même esprit/valeur que
  `postsyncEventsBurstChunk` — vérifier sa valeur sur pièces). Les lectures per-match
  (états, rosters, params) passent par le handle du burst (un writer lit aussi).
  0 match candidat → AUCUN burst (cas stationnaire dominant). La dépendance
  séquentielle des états (EP) impose de persister au fil de l'eau — le chunk borne la
  fenêtre RW, les lecteurs passent entre les chunks.
- **DC-H4** : dans `runSkillRatingSteps`, le segment Read existant reste pour v1 +
  `loadMedalExploitMapBestEffort`, et il est **release AVANT** le shadow v2 (garde
  anti-deadlock). Le shadow reçoit le `*SharedAccess`, plus un handle.
- **DC-H5** : livraison = merge `hotfix/lusr-shadow-ro` → `main` après GO user
  explicite. Pas de squash des commits de la branche (historique lisible).
- **DC-H6** : PAS de backfill préventif. Le watermark n'ayant jamais avancé, le
  backlog depuis le 03-07 re-rentre automatiquement au premier post-sync réussi de
  chaque joueur. Backfill (`lusr_v2_canonical_backfill --commit`) SEULEMENT si le
  soak H6 montre des trous résiduels (groupes `heldGroups` coincés).
- **DC-H7** : logs en français, `slog.*Context` structuré, pas d'emoji, seuils
  fichier/fonction respectés (baseline lint à ne pas accroître).

## 4. Phases

### H1 — Préparation et vérification sur pièces (effort : rapide)

- [x] H1.1 Branche `hotfix/lusr-shadow-ro` déjà créée depuis `origin/main` (HEAD
      `28146aa3a`, qui contient le merge audits) et checkoutée. Vérifié
      `git branch --show-current` = `hotfix/lusr-shadow-ro`.
- [x] H1.2 Vérifié sur pièces (topologie RÉELLE = sous-package, cf. §7 déviation) :
      - `runSkillRatingSteps` / `runSkillRatingStepsWithDB` : `internal/sync/engine_postsync_scoring.go`
        L135 / L151 (package `sync`). Caller : `engine_postsync.go:438`.
      - `runLUSRV2Shadow` + `RunLUSRV2Shadow` + `RunLUSRV2ShadowOwnerOnly` +
        `persistComputedMatchSkillV2` : `internal/sync/skill/skill_v2_shadow.go`
        (L83 / L102 / L106 / L695) — **package `skill`**, pas `sync`.
      - `SharedAccess` (+ `NewPinnedSharedAccess`, `Read`, `Write`) :
        `internal/sync/shared_access.go` (package `sync`, L42/L69/L90/L118). Méthodes
        exactement `Read(ctx)(*sql.DB,func(),error)` + `Write(ctx,string)(*sql.DB,func(),error)`
        → `*sync.SharedAccess` satisfait structurellement l'interface `skill.SharedAccessor`.
      - `postsyncEventsBurstChunk = 3` : `internal/sync/engine.go:50` (package `sync`).
        → nouvelle constante `postsyncLUSRBurstChunk = 3` déclarée côté `skill` (pas
        d'accès cross-package).
      - Callers de `RunLUSRV2Shadow(OwnerOnly)?` : engine_postsync_scoring.go:194 (fix),
        lusr_full_recompute.go:46, cmd/lusr_v2_replay:89, cmd/h5-lusr-smoke:67,
        cmd/lusr_v2_canonical_backfill:110, internal/games/halo_5/livesync/wire.go:107,
        + ~20 sites de test dans internal/sync/skill/skill_v2_shadow_test.go &
        skill_v2_capability_test.go. Re-exports : skill_reexport.go:101/117.
- [x] H1.3 Repro observé ROUGE contre l'ANCIENNE signature (handle unique) : DB fichier
      seedée (1 match 2v2 éligible), attachée READ_ONLY (`ATTACH ... (READ_ONLY); USE s`),
      passée à `RunLUSRV2Shadow(ctx, nil, roHandle, "owner")`. Résultat : `loadShadowMatches`
      + `LoadState` OK (SELECT), `persistComputedMatchSkillV2` → `UpsertState` échoue
      `Cannot execute statement of type "INSERT" on database "s" which is attached in
      read-only mode!` → `processed=0`, aucune ligne `player_skill_state_v2` écrite
      (message prod reproduit). La version pérenne VERTE (accès scindé) est livrée en
      H3.1 (cf. §7 adaptation).

**Gate H1** : chemins consignés ; test de repro ROUGE avec l'erreur
`read-only`/non-inscriptible attendue.

### H2 — Implémentation (effort : moyen)

- [x] H2.1 Signatures + plombage (DC-H2, ADAPTÉ interface seam §7) :
      `runLUSRV2Shadow(ctx, playerDB *sql.DB, shared SharedAccessor, xuid, ownerOnlyPersist)` ;
      suppression du paramètre `sharedDB *sql.DB`. Interface `SharedAccessor`
      (Read/Write) déclarée côté `skill` (`skill_v2_shared_access.go`), satisfaite
      structurellement par `*sync.SharedAccess`. `shadowRunContext.repo/squadRepo/sharedDB`
      câblés PAR CHUNK sur le handle du burst (pas globalement).
- [x] H2.2 Découpage Read/Write (DC-H3) : sélection sous `loadShadowMatchesUnderRead`
      (Read, release immédiat) ; `processShadowChunk` par chunks de
      `postsyncLUSRBurstChunk=3` sous `shared.Write(ctx, "lusr")` ; repos
      `NewSkillV2Repo`/`NewSquadOffsetRepo` construits sur le handle du burst ;
      `heldGroups`/watermark sémantique INCHANGÉE (map partagée entre chunks, ordre
      chrono ASC préservé) ; 0 match candidat → AUCUN burst. Tests anti-gap/held/dual-row
      existants restent verts (cf. gate).
- [x] H2.3 `runSkillRatingSteps` (DC-H4) : v1 + medal map sous Read (helper
      `runLUSRV1UnderRead`, defer release), PUIS `RunLUSRV2ShadowOwnerOnly(scoringCtx,
      playerDB, shared, e.xuid)` (reçoit le `*SharedAccess`), PUIS sentinelle
      (`runDualRowSentinelBestEffort`, playerDB-only). Commentaire de classification
      CORRIGÉ (le v2 écrit shared → burst Write dédié). `runSkillRatingStepsWithDB`
      SUPPRIMÉ (code mort).
- [x] H2.4 Callers migrés : `engine_postsync_scoring.go` (passe `shared`),
      `lusr_full_recompute.go` (`NewPinnedSharedAccess`), `cmd/lusr_v2_replay`,
      `cmd/h5-lusr-smoke`, `cmd/lusr_v2_canonical_backfill` (`lusync.NewPinnedSharedAccess`),
      `internal/games/halo_5/livesync/wire.go` (`syncpkg.NewPinnedSharedAccess`), ~21 sites
      de test (`newPinnedSharedAccessor`, skill-local).
- [x] H2.5 Grep de contrôle : plus AUCUN caller ne passe un `*sql.DB` brut au shadow
      (build type-check garantit la signature `SharedAccessor`) ; `go vet ./...` propre.

**Gate H2** : `cd apps/go-api && CGO_ENABLED=1 go build ./... && go vet ./...` verts (exit 0/0) ;
suite `go test -tags=cgo ./internal/sync/...` verte (sync 24.5s, skill 0.8s, v2 13.3s, tous ok).
Test de repro devenu VERT : couvert par la version pérenne H3.1 [~] (le repro a été retiré
en H1 pour garder l'arbre buildable — cf. §7 adaptation).

### H3 — Tests et audit des segments frères (effort : moyen)

- [x] H3.1 Test pérenne `TestLUSRV2Shadow_PersistsViaWriteBurst_WhenReadHandleIsReadOnly`
      (`skill_v2_shadow_burst_test.go`) : `roRwSplitAccess` sur DB FICHIER — `Read` sert
      un attach `READ_ONLY` (sélection OK), `Write` un attach RW (persist OK). Run
      owner-only → `processed=1`, `writeCalls>=1`, état owner persisté avec
      `last_match_at` non NULL (watermark avancé). VERT. Garde de régression : le handle
      Read est réellement read-only → tout retour au persist-via-Read casserait le test.
- [x] H3.2 Test anti-deadlock `TestLUSRV2Shadow_ReleasesReadBeforeWriteBurst_MultiChunk` :
      `orderTrackingAccess` échoue si un `Write` est demandé pendant qu'un `Read` est en
      vol. Sur 4 matchs (2 chunks) → `processed=4`, `writeCalls>=2`, `violation=""`. VERT.
- [x] H3.3 Non-régression : suite complète `go test -tags=cgo ./internal/sync/skill/`
      VERTE (0.85s) — owner-only / anti-gap / held-group / dual-row / canonical / h5
      title-aware tous verts. `./internal/sync/...` vert (cf. H2).
- [x] H3.4 AUDIT des segments lecture d'`engine_postsync*.go` (verdict par segment) :

      | Segment (`shared.Read`/`withSharedRead`) | Callee | Écrit shared ? | Verdict |
      |---|---|---|---|
      | pré-checks (postsync.go:81) | `hasMatchesNeedingScoreRefresh`, `hasConvergenceBacklog` | non (lectures) | OK |
      | scoring (scoring.go:22) | `runScoringStepsWithDB` (perf/engagement/sessions/bot) | non — écrit PLAYER ; intensités ACCUMULÉES puis flushées en burst `Write("engagement_intensity")` (scoring.go:37) | OK |
      | LUSR v1 (scoring.go:186) | `runLUSRV1UnderRead`→`batchComputeLUSR` | non — écrit PLAYER (`match_skill_rank`) | OK |
      | enrichment_rows (postsync.go:217) | `ensurePlayerEnrichmentRows` | non — INSERT PLAYER `player_match_enrichment` | OK |
      | events_select (260) | `selectMatchesMissingEvents` | non ; converge en burst `Write("events")` (276) | OK |
      | weapons_select (302) | `selectMatchesMissingWeapons` | non ; converge en burst `Write("weapons")` (315) | OK |
      | catalog_refresh (382) | `CatalogRefreshFromRegistry(metaDB, sharedDB)` | non — LIT `sharedDB` (match_registry), ÉCRIT `metadataDB` (playlists/maps/variants catalog) | OK (vérifié sur pièces catalog_refresh.go:60/89) |
      | citations (396) | `runPostSyncCitations` | non — écrit PLAYER (`match_citations`) | OK |
      | dominance (420) | `BackfillDominanceFlags` | non — écrit PLAYER via `PostSyncEnrichmentPersister` (commentaire L418) | OK |
      | friends (471) | `RecomputeIsWithFriendsCore` | non — écrit PLAYER (`is_with_friends`) | OK |
      | snapshot_readiness (511) | `EvaluateSnapshotReadiness` | non — écrit PLAYER (marqueur readiness) | OK |

      Vérification transverse : grep `sharedDB.ExecContext` sur les .go de prod → seuls
      écrivains shared = `csr_shared_writes` (via `runCSRSnapshotSync`, writer AUTONOME hors
      segment Read), `engagement.go:559` (`persistMatchIntensities`, appelé SOUS le burst
      `Write`), `pve.go` (pipeline PVE distinct), `backfill_registry_names`/`lusr_full_recompute`
      (CLI/recovery, writer tenu). AUCUN dans un segment Read post-sync. Le shadow LUSR v2
      était le SEUL écrivain shared mal classé « lecture » — corrigé. Aucune anomalie
      résiduelle → §Découvertes = aucune sur ce point.

**Gate H3** : VERT. `go test ./...` (racine) exit 0 (aucun échec) ;
`go test -tags=integration -p 1 -timeout 900s ./...` exit 0 ; intégration ciblée
`./internal/persist/... ./internal/sync/...` (anti-ART) exécutée et verte (persist 13.4s,
sync 80.2s, skill 1.1s, v2 13.2s) ; tableau H3.4 rempli.

### H4 — Gate final pré-livraison (effort : rapide)

- [ ] H4.1 `make go-api-lint` : 0 nouvelle erreur vs baseline.
- [ ] H4.2 Skill `delivery-checklist` passé ; entrée `thought_log.md` (obligatoire
      avant commit final).
- [ ] H4.3 Commits propres sur `hotfix/lusr-shadow-ro` (messages
      `fix(sync): ...` explicites), CI de branche verte si disponible.

**Gate H4** : tous les items H1→H4 statués ; aucun `[ ]` restant.

### H5 — Livraison (GATE USER — ne pas franchir sans GO explicite)

- [ ] H5.1 Présenter au user : diff résumé, résultats des gates, rappel « push main =
      deploy prod auto ». ATTENDRE le GO dans le tour courant.
- [ ] H5.2 Après GO : merge dans main (se placer sur main à jour, merger la branche,
      pousser — jamais de push direct branche→main sans synchroniser main local).
- [ ] H5.3 Surveiller le déploiement (conteneur redéployé, `docker ps` healthy,
      `/health` 200).

### H6 — Vérification post-deploy et soak (effort : rapide, lecture seule prod)

- [ ] H6.1 Sur le VPS (`ssh lvelup`, lecture seule,
      logs `/opt/levelup/data/logs/`) : plus AUCUNE occurrence nouvelle de
      `persist état échoué — watermark non avancé` ; compteur
      `writer RW tenu au-delà du seuil` retombé ; plus de 503 sur `/health`
      (`grep '"status":503' general.log` sur la fenêtre post-deploy).
- [ ] H6.2 Rattrapage automatique : sur 2-3 cycles, `LUSR v2 shadow terminé` montre
      `processed > 0` puis retour à 0 (backlog 03-07→deploy résorbé) ; les lignes
      LUSR récentes réapparaissent (vues `_latest`).
- [ ] H6.3 Si trous résiduels après 24 h (groupes tenus) : backfill ciblé
      `lusr_v2_canonical_backfill --commit` (DC-H6) — sinon statuer [~] « non requis ».
- [ ] H6.4 Rollback si dégradation imprévue : `LEVELUP_POSTSYNC_BURST=0` sur le
      conteneur (mode pinned) le temps du diagnostic — PRÉVENIR le user.

**Gate H6** : H6.1 + H6.2 constatés sur pièces ; sinon rollback H6.4 + réouverture H2.

### H7 — Clôture et coordination (effort : rapide)

- [ ] H7.1 Statuer tous les items ; §Découvertes rempli ; entrée finale
      `thought_log.md` (résultats prod observés).
- [ ] H7.2 Coordination branche audits : après le merge main, reporter le fix sur
      `refactor/audits-2026-07` (merge main → branche) — conflits ATTENDUS sur les
      fichiers déplacés par le lot K (`internal/sync/skill/`) ; sur la branche, le
      seam devient une petite interface déclarée côté `sync/skill` (le sous-package ne
      peut pas importer `sync` — cycle), satisfaite structurellement par
      `*sync.SharedAccess`. Si la session ne fait pas ce report : le consigner
      explicitement comme reste-à-faire dans le thought_log ET dans
      `.ai/PLAN_CLOTURE_AUDITS_2026-07.md` (section découvertes/merge).
- [ ] H7.3 Mettre à jour `.ai/PLAN_MONITORING_TRIAGE_DETECTIONS_2026-07.md` : B1
      statué [x] avec référence à ce plan et aux constats H6.

## 5. Périmètre interdit (zéro fix opportuniste)

NE PAS traiter ici (déjà consignés ailleurs) : le design du healthcheck `/health`
(I/O DB → 503 sous gate), la rotation des logs, le disque à 82 %, les démotions de
bruit (plan triage B5/B6), toute autre découverte → §Découvertes de ce fichier.

## 6. Protocole de reprise de session

Lire ce fichier (statuts + chemins consignés en H1.2) puis
`git log --oneline -10` sur `hotfix/lusr-shadow-ro`. Reprendre à la première case non
statuée de la phase la plus basse non close (phase close = items statués + gate passé).
H5 exige un GO user du tour courant — ne jamais le considérer acquis d'une session
précédente.

## 7. Découvertes hors périmètre

- **DÉVIATION MAJEURE DE TOPOLOGIE (constatée 2026-07-10, sur pièces)** : contrairement
  à l'hypothèse de l'en-tête du plan (« sur main tout est encore à plat dans
  `internal/sync/` »), la branche `hotfix/lusr-shadow-ro` a été créée depuis un
  `origin/main` qui **contient déjà le merge de la campagne audits** (commit
  `28146aa3a`, déployé prod 2026-07-10). La topologie réelle = celle de la branche
  audits : le cluster skill (dont `RunLUSRV2Shadow`) vit dans le **sous-package**
  `internal/sync/skill/` (package `skill`), ré-exporté vers `sync` via
  `skill_reexport.go`. `SharedAccess` reste dans package `sync`
  (`internal/sync/shared_access.go`). Le sous-package `skill` NE PEUT PAS importer
  `sync` (cycle sync→skill). Conséquence : DC-H2 est appliqué via l'**interface seam**
  décrit en H7.2 (petite interface `SharedAccessor` déclarée côté `skill`, satisfaite
  STRUCTURELLEMENT par `*sync.SharedAccess`), et NON par passage direct d'un
  `*sync.SharedAccess` au runner. H7.2 (report du fix sur la branche audits) devient
  **sans objet** : la branche audits est déjà mergée dans main. Voir thought_log
  2026-07-10.
- Adaptation du test de repro (H1.3) : la topologie sous-package impose que la version
  pérenne du test utilise la nouvelle API (`SharedAccessor`). Le repro ROUGE est observé
  contre l'ANCIENNE signature (handle unique RO), puis la version pérenne VERTE (accès
  scindé Read RO / Write RW) est livrée en H2/H3 — même intention, API adaptée.
