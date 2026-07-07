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

- [ ] H1.1 `git fetch origin && git checkout -b hotfix/lusr-shadow-ro origin/main`.
- [ ] H1.2 Vérifier sur pièces (la topologie main ≠ branche audits) : localisation
      réelle de `runSkillRatingSteps` / `runSkillRatingStepsWithDB`
      (`engine_postsync_scoring.go`), `runLUSRV2Shadow` + `persistComputedMatchSkillV2`
      (`skill_v2_shadow.go`), `SharedAccess` (`shared_access.go`),
      `postsyncEventsBurstChunk` (valeur), tous les callers de
      `RunLUSRV2Shadow(OwnerOnly)?` (`grep -rn "RunLUSRV2Shadow" apps/go-api`).
      Consigner les chemins/lignes constatés ici.
- [ ] H1.3 Reproduire le symptôme en local AVANT fix : test d'intégration provisoire
      ou scénario existant adapté — un `SharedAccess` burst dont `read` sert un handle
      où `shared_matches_v2` n'est pas inscriptible → `UpsertState` échoue avec le
      message de prod. (Ce test devient le test pérenne en H3 — rouge ici, vert
      après H2.)

**Gate H1** : chemins consignés ; test de repro ROUGE avec l'erreur
`read-only`/non-inscriptible attendue.

### H2 — Implémentation (effort : moyen)

- [ ] H2.1 Signatures + plombage (DC-H2) : `runLUSRV2Shadow(ctx, playerDB,
      shared *SharedAccess, xuid, ownerOnlyPersist)` ; suppression du paramètre
      `sharedDB *sql.DB` ; `shadowRunContext` transporte ce qu'il faut (handle du
      burst courant passé aux étapes, pas stocké globalement).
- [ ] H2.2 Découpage Read/Write (DC-H3) : sélection sous Read (release immédiat) ;
      chunks sous `shared.Write(ctx, "lusr")` ; repos (`NewSkillV2Repo`,
      `NewSquadOffsetRepo`) construits sur le handle du burst courant ;
      `heldGroups`/watermark : sémantique INCHANGÉE (anti-gap 2026-06-07 préservé —
      les tests existants le verrouillent).
- [ ] H2.3 `runSkillRatingSteps` (DC-H4) : v1 + medal map sous Read, release, puis
      `RunLUSRV2ShadowOwnerOnly(scoringCtx, playerDB, shared, e.xuid)` ; le
      commentaire de classification est CORRIGÉ (doc inversée = anti-pattern n°9).
- [ ] H2.4 Callers migrés (DC-H2) : `lusr_full_recompute.go`, `cmd/lusr_v2_replay`,
      chemin H5 livesync de main, tests → `NewPinnedSharedAccess(db)`.
- [ ] H2.5 Grep de contrôle : plus AUCUN appel `RunLUSRV2Shadow*(..., *sql.DB, ...)` ;
      `go vet ./...` propre.

**Gate H2** : `cd apps/go-api && go build ./... && go vet ./...` verts ; test de repro
H1.3 devenu VERT ; suite `go test ./internal/sync/...` verte.

### H3 — Tests et audit des segments frères (effort : moyen)

- [ ] H3.1 Pérenniser le test de repro (nom explicite, ex.
      `TestLUSRV2Shadow_PersistsViaWriteBurst_WhenReadHandleIsReadOnly`) : lease burst
      avec `read` = handle non-inscriptible et `acquire` = handle RW → run owner-only
      traite et persiste ; assertion aussi sur l'avance du watermark.
- [ ] H3.2 Test anti-deadlock : le shadow ne demande jamais un burst Write pendant
      qu'un Read du même SharedAccess est en vol (le garde de `Write` transformerait
      le bug en erreur — vérifier qu'aucun chemin ne la déclenche, y compris quand
      `loadShadowMatches` retourne > 1 chunk).
- [ ] H3.3 Non-régression : suite shadow complète
      (`go test ./internal/sync/ -run "LUSR|Shadow"` + intégration ciblée) — les tests
      d'autonomie owner-only / anti-gap / dual-row restent verts.
- [ ] H3.4 AUDIT des autres segments lecture du post-sync sur main : pour chaque
      `shared.Read(` de `engine_postsync*.go`, vérifier sur pièces que le bloc
      n'appelle AUCUN chemin qui écrit shared (events/weapons = bursts Write, OK
      constaté ; bloc intensités/scoring L22 : vérifier où écrivent les résultats).
      Consigner le verdict par segment ICI (tableau). Toute anomalie trouvée = la
      corriger dans CE hotfix seulement si c'est le même défaut de classification ;
      sinon §Découvertes.

**Gate H3** : `go test ./...` (racine apps/go-api) vert ;
`go test -tags=integration -p 1 -timeout 900s ./...` exit 0 (OBLIGATOIRE — sync/persist
touchés ; jamais de commandes go concurrentes sur ce poste) ; tableau H3.4 rempli.

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

- (vide)
