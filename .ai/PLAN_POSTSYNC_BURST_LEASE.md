# Plan — Étape 1 contention : post-sync en bursts paresseux (writer non tenu pendant I/O)

Statut : EN COURS (incrément 1)
Date : 2026-07-02 · Branche : fix/h5-ui-adjustments-batch (rester dessus)
Mesure de référence (étape 0, live) : `sync_v2_postsync` = 8/8 swaps, 104,1s/105s de cycle,
avg 13 017 ms, max 21 909 ms, watchdog 6/8 — détenteur UNIQUE de la fenêtre RW.
GATE DE SUCCÈS : `rw_window_max(sync_v2_postsync*) < 1500 ms` sur un cycle live + suite verte.

## Constat validé code (2 agents croisés)

- Pipeline post-sync (engine_postsync.go:59 runConditionalPostSync / :124 runPostSyncPipeline,
  ~16 étapes) reçoit un `*sql.DB` shared RW acquis UNE fois (V2 : post_sync_runner.go:94 ;
  V1 post-drain : engine.go:559 ; V1 non-batch : writer primaire engine.go:602).
- Seuls 4 points écrivent VRAIMENT dans shared :
  1. engagement `match_intensity` (engagement.go:65 → appel inline :169, write :529-535)
  2. convergeEvents (convergence.go:293-306 → ProcessHighlightEvents)
  3. processWeaponKillsInline (backfill_weapons.go:262-303 → BackfillWeaponKillsForMatch:45)
  4. alias PSA (convergence.go:131/:143/:173-197 → UpsertXUIDAlias:192)
  CITATIONS/DOMINANCE/LUSR écrivent la PLAYER DB (comeback_postsync_persist.go:55-59,
  skill_v2_shadow.go:69) — le commentaire engine_postsync.go:163 est périmé.
- Tout le reste = lectures shared ou player-only → en stationnaire (0 match), 0 write shared.
- Précédent maison du pattern : convergence_backfill_events.go (detect RO → fetch hors lease →
  write par chunk, labels étape 0 déjà posés).
- Pièges : finalizer progression lit shared RO après pipeline (engine.go:222-239) ; closure-swap
  release batch-async (engine.go:514-526) ; drain auto-deadlock si RO tenu pendant AcquireWriter
  (provider.go:52-56, rollback provider_writer.go:242-259) ; read-your-writes inter-bursts OK car
  releaseWriter rouvre le RO synchronement (provider_writer.go:296-315).

## Design : `SharedAccess` (nouveau internal/sync/shared_access.go)

```go
type SharedAccess struct { acquire SharedWriterAcquirer; read func(ctx)(*sql.DB,func(),error);
  label string; outstandingReads atomic.Int32; probed bool /* assertSharedWritable memoïsé */ }
NewPinnedSharedAccess(db *sql.DB) *SharedAccess   // handle déjà tenu, release no-op (compat V1 non-batch)
NewBurstSharedAccess(acquire, read, label) *SharedAccess
(a) Read(ctx) (*sql.DB, func(), error)             // RO (provider.Get) ; pinned → handle pinné
(a) Write(ctx, step string) (*sql.DB, func(), error) // burst RW label = label+"/"+step
```
- Write REFUSE (erreur) si outstandingReads>0 (anti auto-deadlock drain).
- assertSharedWritable (shared_rw_guard.go:34, appelé engine_postsync.go:167) migre dans Write (1er burst).
- Mode legacy (provider nil) : Read délègue au chemin Write burst (dblease engine_acquire.go:63-76).

## Séquence d'incréments COMMITTABLES (sûr → risqué)

1. **SharedAccess + tests unitaires** (pinned=passthrough ; Write-pendant-Read=erreur ; probe 1er burst).
   Aucun call-site touché. [EN COURS]
2. **Plomberie signatures** : runConditionalPostSync/runPostSyncPipeline (engine_postsync.go:59/:124),
   RunPostSyncForV2 (engine_v2bridge.go:32-39), runScoringSteps/runSkillRatingSteps
   (engine_postsync_scoring.go:14/:94), hasConvergenceBacklog (convergence.go:66) → `*SharedAccess`.
   TOUS les call-sites en NewPinnedSharedAccess = byte-identique, suite verte.
3. **Segments Read** pour les étapes pure-lecture + test « stationnaire = 0 acquire » (fake acquirer compteur).
4. **Mixtes, 1 commit chacun** (fetch hors lease → burst write, SQL INTACT) :
   a. engagement : supprimer write inline :169, accumuler (matchID,intensity), 1 burst en fin d'étape.
   b. PSA : fetch JSONs hors lease, writes player inchangés, alias bufferisés → 1 burst UpsertXUIDAlias.
   c. events : scinder ProcessHighlightEvents au seam fetch/write (engine_highlight_events.go:249),
      chunks ~25 : fetch hors lease → 1 burst write.
   d. weapons (LE PLUS RISQUÉ — mémoire films) : scinder BackfillWeaponKillsForMatch:45 — fetch film +
      timelines (parallélisme 24 conservé) hors lease, chunks ~50 → burst write sérialisé
      (DELETE-then-INSERT weapon_kills INCHANGÉ :249-252).
5. **Flip** : V2 (post_sync_runner.go:94-108 → NewBurstSharedAccess ; NewPostSyncRunner:63 gagne un
   reader RO ; wiring sync_v2_wiring.go:142/:154-170) + V1 batch post-drain (engine.go:549-585).
   V1 non-batch reste pinned (engine.go:602). Rollback env `LEVELUP_POSTSYNC_BURST=0` → pinned
   (défaut ON ; à RETIRER après validation, règle « pas de flag OFF »). Maj rca_test + test intégration
   « fenêtre bornée par étape » (assert holderSnapshot par label sync_v2_postsync/* : max<1500ms) +
   test visibilité read-after-burst. Mesure live finale.

## Tests

- Verrouillent l'existant : v2/post_sync_test.go, v2/post_sync_runner_rca_test.go (à ADAPTER :
  acquirer appelé au 1er burst, dégradation PAR ÉTAPE), v2/soak/e2e, scheduler/auto_sync_postsync_e2e,
  sync/engine_post_sync_runner*_test, trigger_engine_parity, no_art_patterns (DOIT rester vert),
  sharedprovider watchdog/holder.
- Nouveaux : unit SharedAccess ×3 ; stationnaire=0 acquire ; intégration fenêtre bornée par étape ;
  visibilité inter-bursts.

## Risques & mitigations

1. Auto-deadlock drain → guard outstandingReads + rollback drain-timeout existant.
2. Stale read inter-bursts (weapons lit events du burst précédent) → release synchrone + test visibilité.
3. Multiplication des swaps sous PostSyncParallelism (post_sync.go:104-126) → paresseux (0 en
   stationnaire), ≤4-6 bursts/joueur, monitoring swapTotal + watchdog 2s conservés.
