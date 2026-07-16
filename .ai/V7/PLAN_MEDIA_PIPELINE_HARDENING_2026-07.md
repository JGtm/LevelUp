# PLAN — Durcissement pipeline média (remux / audio HLS / lightbox)

> Date : 2026-07-16 · Branche : `fix/media-pipeline-hardening` (worktree
> `.claude/worktrees/media-pipeline-hardening`, base origin/main 15d8e3902)
> Origine : audit pipeline média 2026-07-16 (11 trouvailles, toutes validées par
> l'utilisateur). Exécution sous contrat skill `plan-execution` — ordre strict,
> zéro report d'item exécutable, statuts [x]/[~]/[!] obligatoires à la clôture.
> Supervision : agent principal (git/commits/gates) ; implémentation : agents Opus.

## Objectif / critère de succès

Les 11 trouvailles de l'audit sont traitées (code ou justification écrite),
`go test ./...` + `go vet` verts côté Go, typecheck+lint+vitest verts côté web,
aucun garde-rail affaibli. Pas de merge dans main ici (revue utilisateur ensuite).

## Découvertes hors périmètre

(consigner ici, ne PAS traiter)

- Branche parquée `feat/media-keep-source-toggle` (0 commit) : intention probable
  de rendre optionnelle la suppression du source — recouvre la trouvaille #1 côté
  produit, à décider hors chantier.
- (LOT A) Provisionnement de `transcode_started_at` : la colonne est ajoutée par
  `ensureMediaTables` (appelée par `IndexMedia`), qui tourne TOUJOURS avant le
  balayage post-sync (vérifié : 3 sites `triggerHLSSweep` précédés d'`IndexMedia`).
  Le CLI `backfill-media-hls`, lui, n'appelle ni migration ni `ensureMediaTables` :
  il suppose (comme déjà pour `hls_path`) que le serveur a indexé au moins une fois
  depuis le déploiement. Contrat identique à l'existant, pas de régression
  introduite ; si on veut rendre le CLI auto-suffisant, exposer un `EnsureSchema`
  serait le geste — hors périmètre LOT A.

---

## LOT A — Concurrence transcodage + décisions persistées (trouvailles #1, #2, #10)

Fichiers : `apps/go-api/internal/ops/media_hls.go`, `media_hls_sweep.go`,
`apps/go-api/internal/service/media_service_upload.go`, schéma `media_files`
(shared_social) + tests associés. Chercher le mécanisme de migration existant du
schéma media_files (grep `ALTER TABLE media_files` / création table) et s'y
conformer.

- [x] A1. Nouveau statut persisté `TranscodeDirect = "direct"` : quand
      `DetectHLSNeeded` retourne false (upload OU sweep), marquer la ligne
      `transcode_status='direct'` — plus jamais re-probée ensuite.
      → constante dans `media_hls.go` ; marquage dans `launchHLSTranscoding`
      (upload) et `processHLSCandidate` (sweep, hors dry-run).
- [x] A2. Colonne `transcode_started_at TIMESTAMPTZ` posée avec
      `transcode_status='processing'` (upload ET sweep). → colonne ajoutée à la
      boucle ALTER idempotente de `ensureMediaTables` (media_store.go), pattern
      identique à `hls_path`/`transcode_status` (colonnes HLS sœurs) ; nouvelle
      fonction `MarkTranscodeProcessing` appelée par l'upload ET le sweep AVANT
      `RunHLSTranscode`. [Revue superviseur 2026-07-17] durcie en COMPARE-AND-SET :
      l'UPDATE ne s'applique que si la ligne n'est pas déjà 'processing' frais
      (même prédicat que la sélection, fragment SQL partagé
      `transcodeNotFreshProcessingSQL`) ; retour `(acquired bool, err)` via
      RowsAffected ; les DEUX callers sautent le transcodage (log Info) sur
      acquired=false. Ferme la fenêtre sélection→marquage du sweep (un upload qui
      marquait entre les deux était écrasé par l'UPDATE inconditionnel → double
      ffmpeg).
- [x] A3. Sélection du sweep (`selectPendingHLSCandidates`) : exclut
      `transcode_status IN ('direct','failed')` ET les 'processing' FRAIS ;
      'processing' périmé (`transcode_started_at` > 2 h) ou sans horodatage
      (orphelin legacy) redevient éligible ; `transcode_status` NULL reste
      éligible. Constante `transcodeStaleAfter = 2 * time.Hour` (déplacée dans
      media_hls.go, source unique partagée avec le CAS — zéro copie du littéral).
      Seuil comparé Go-side (`time.Now().UTC().Add(-transcodeStaleAfter)`) ;
      prédicat 'processing' = fragment partagé `transcodeNotFreshProcessingSQL`.
- [x] A4. 'failed' : plus de retry automatique. `cmd/backfill-media-hls` gagne
      `--retry-failed` → `ops.ResetFailedTranscodes` (failed → NULL, scope --slug),
      ignoré en dry-run. Aide du flag documentée + en-tête du fichier.
- [x] A5. Les 2 CHECKPOINT avalés (`MarkTranscodeStatus`, `finalizeMediaHLS`) →
      helper `checkpointBestEffort` (log WARN slog module "media"). Centralisé
      (aussi utilisé par `MarkTranscodeProcessing`/`ResetFailedTranscodes` : 1
      seule occurrence du littéral CHECKPOINT, pas de 3e copie).
- [x] A6. Tests : `TestSelectPendingHLSCandidates` réécrit (exclusions + ré-
      éligibilité) ; `TestMarkTranscodeProcessing_CompareAndSet` (1re acquisition
      true + status/horodatage, 2e refusée sur 'processing' frais + ligne
      inchangée, ré-acquisition après vieillissement artificiel > 2 h) ;
      `TestEnsurePendingHLS_DirectMarkingPersists` (2e sweep ne re-probe pas,
      ffmpeg) ; assertion `transcode_started_at` non-NULL dans
      `TestEnsurePendingHLS_TranscodesScannedVideo` (marquage processing du
      sweep) ; `TestResetFailedTranscodes` ; schéma `transcode_started_at`.

Gate A : `cd apps/go-api && go vet ./... && go test ./internal/ops/... ./internal/service/... ./internal/media/...`
puis `go test -tags=integration -p 1 ./internal/ops/...` (code de sortie vérifié,
filtre ancré `^--- FAIL:`). Aucun test skippé sans justification.

> Clôture LOT A [2026-07-17] : les 3 commandes de gate exécutées, code de sortie 0
> vérifié pour chacune (vet, tests ops/service/media, tests intégration ops -p 1 —
> aucune ligne `^--- FAIL:`). ffmpeg/ffprobe présents → tests HLS gated RÉELLEMENT
> exécutés (aucun skip). Garde-rails anti-ART (`internal/sync`) re-vérifiés verts :
> `media_files` n'est ni protégée ni critique, mes UPDATE mono-ligne (`?`) ne
> déclenchent aucun motif — allowlist inchangée. Pas de commit (superviseur).
>
> Revue superviseur [2026-07-17], 2 corrections intégrées puis gate REJOUÉ (3× exit
> 0, 0 `^--- FAIL:`) : (1) `MarkTranscodeProcessing` → compare-and-set (voir A2) —
> le TOCTOU sélection-de-sweep→marquage est fermé ; (2) numéros de trouvaille
> supprimés des commentaires code/tests (descriptions concrètes à la place — les
> numéros ne survivent pas au contexte de l'audit) ; ils ne subsistent que dans ce
> plan, qui les définit.

## LOT B — Lecteur lightbox (trouvailles #3, #4)

Fichiers : `apps/web/src/features/media/CoverFlowModal.tsx` + tests
(`CoverFlowModal.hls.test.tsx`, `CoverFlowModal.test.tsx`).
Pré-requis : `npm install` dans `apps/web` du worktree (node_modules absent).

- [x] B1. Désync toggles/mute : au recentrage d'un clip (isCenter redevient
      true), l'état des DEUX interrupteurs est réappliqué (muted si les deux
      OFF, rendition correspondante sinon). Décision produit TRANCHÉE :
      l'état choisi persiste (pas de reset à ON/ON). Attention à l'ordre des
      effets React (l'effet parent [currentItem] de CoverFlowModal force
      muted=false APRÈS les effets de l'enfant) : le parent ne doit plus
      démuter un clip dont les deux toggles sont OFF — mécanisme au choix de
      l'implémentation (ex. dataset sur l'élément video consulté par le parent),
      mais SANS état global ni prop drilling nouveau.
      → MÉCANISME RETENU : marqueur DOM `video.dataset.audioOff='1'` posé par
      l'effet enfant des toggles quand les deux sont OFF (retiré via `delete`
      sinon) ; l'effet parent [currentItem] ne démute (`vid.muted=false`) QUE si
      `vid.dataset.audioOff !== '1'`. Résiste à l'ordre parent-après-enfant :
      l'enfant écrit le marqueur pendant son effet (qui court AVANT le parent
      dans le même commit), le parent le LIT et s'abstient — réappliquer `.muted`
      côté enfant seul ne suffirait pas (le parent le démuterait derrière).
      `isCenter` ajouté aux deps de l'effet enfant pour la réapplication au
      recentrage. Zéro état global, zéro prop nouvelle (dataset local au node).
      Commentaire mensonger « ClipPlayer remonte par clip » corrigé aux 2 sites
      (déclaration gameOn/voiceOn + effet d'application) : l'instance PERSISTE
      dans la fenêtre ±2 (key sur la div de slot, pas sur ClipPlayer).
- [x] B2. Chargement voisins : les instances hls.js sont créées avec
      `autoStartLoad: false` ; `startLoad()` uniquement quand le clip est
      centré, `stopLoad()` quand il quitte le centre. Décision TRANCHÉE :
      chargement centre uniquement (pas de préchargement ±1).
      → `new Hls({ enableWorker: true, autoStartLoad: false })` ; nouvel effet
      dédié `[isHls, isCenter]` : `hlsRef.current.startLoad()` si isCenter,
      `stopLoad()` sinon. `loadSource()` INCHANGÉ dans l'effet d'attache (charge
      le manifest → AUDIO_TRACKS_UPDATED peuple encore le sélecteur des voisins ;
      seuls les segments sont gatés). Effet start/stop déclaré APRÈS l'effet
      d'attache → hlsRef.current déjà posé au montage. Repli natif Safari et
      priorité hls.js (quirk Chrome canPlayType "maybe") intacts.
- [x] B3. Tests vitest : (a) scénario both-OFF → navigation ailleurs → retour →
      vidéo toujours muette et toggles OFF ; (b) startLoad appelé pour le clip
      centré seulement (étendre le mock hls.js existant).
      → Mock hls.js étendu (`startLoad`/`stopLoad` = vi.fn, + interface). (a)
      nouveau describe « persistance du mute deux OFF au recentrage » : 2 clips
      HLS, A both-OFF (assert muted + dataset.audioOff='1'), navigation
      A→B→A via flèches (fake timers pour libérer animatingRef sur ANIM_MS),
      re-assertion muted=true + toggles aria-pressed=false. (b) describe
      « chargement HLS réservé au clip centré » : instances[0]=A startLoad et
      pas stopLoad, instances[1]=B jamais startLoad + stopLoad ; après navigation
      A→B, instances[0].stopLoad et instances[1].startLoad. (c) les 4 combos
      toggles existants restent verts (36/36 sur les 2 fichiers média).

Gate B : depuis `apps/web` : purge `node_modules\.tmp` puis
`npm run typecheck && npm run lint && npm run test` (vitest hors sandbox,
cf. mémoire). Zéro nouvelle string UI attendue (sinon i18n FR+EN).

> Clôture LOT B [2026-07-17] : 4 commandes de gate exécutées, codes de sortie
> vérifiés — purge .tmp (fait), `npm run typecheck` exit 0, `npm run lint` exit 0
> (68 warnings baseline pré-existants, 0 erreur ; les 4 warnings sur
> CoverFlowModal.tsx — set-state-in-effect ligne pendingPageAdvance + 2 deps
> `navigate` manquantes sur keydown/autoChain — sont antérieurs et intentionnels,
> mes effets n'en ajoutent aucun), `npm run test` exit 0 (2239 passed, 14 skipped
> pré-existants, 0 failed). Aucune string UI ajoutée (le marqueur data-audio-off
> n'est pas un label). Aucune couleur hex/classe Tailwind couleur introduite.
> Pas de commit (superviseur).

## LOT C — Durcissements ffmpeg / serving (trouvailles #5, #6, #7, #8, #9, #11)

Fichiers : `apps/go-api/internal/media/hls.go`, `hls_audio_analyze.go`,
`remux.go`, `apps/go-api/internal/api/handlers/media_serve.go`,
`apps/go-api/internal/ops/media_hls.go` (VerifyHLSPlayable caller) + tests.

- [ ] C1 (#5a). Mono-piste : `singleAudioRendition` passe par
      `aacUniformAction` (copy si déjà AAC, sinon réencode AAC) — supprime le
      cas Opus copié inaudible en HLS natif Safari. Supprimer `planAudio` si
      plus aucun caller (règle 0 code mort) ; adapter les tests existants.
- [ ] C2 (#5b). Vidéo HEVC copiée : ajouter `-tag:v hvc1` dans `buildHLSArgs`
      quand action=copy ET codec source hevc/h265 (nécessite de propager le
      codec source dans le plan). Test sur les args générés.
- [ ] C3 (#5c). Commentaire d'en-tête hls.go : acter que la cible navigateur
      est Chrome/Firefox/Edge via hls.js, Safari natif = best-effort (sélecteur
      de pistes absent en natif).
- [ ] C4 (#6). Clipping : chaque sortie amix (voices multi-composantes, full
      historique) est suivie d'un `alimiter` dans le filter_complex (limite
      nommée en constante, ~0.98). Tests des chaînes de filtre générées
      (amixFilter/componentRenditions/fullMixRenditions).
- [ ] C5 (#7). `audioEnvelope` : borner la lecture à 600 s (constante nommée,
      commentaire : la corrélation d'enveloppe converge bien avant ; évite de
      bufferiser des heures de PCM en RAM sur le VPS 2 Go).
- [ ] C6 (#8). `serveRemuxedWebM` : pre-flight (probe + chooseAudioMap) AVANT
      d'écrire tout header → 415 si codecs incompatibles, 502/500 si probe
      échoue ; streaming ensuite inchangé. Nécessite de scinder
      `StreamRemuxAsWebM` (plan pur exposé + exécution) — pas de duplication de
      la logique de probe. Test handler httptest (415 sur codec incompatible).
- [ ] C7 (#11). `VerifyHLSPlayable` : vérifier AUSSI que le nombre de pistes
      audio du master == attendu (paramètre depuis `HLSResult.AudioTracks`)
      avant la suppression du source. Adapter le caller RunHLSTranscode + tests.
- [ ] C8 (#9). Collision de stem `hls/{stem}` (clip.mkv + clip.mp4 coexistants) :
      documenter la limite en commentaire de `HLSPathsFor`. Statut [~] accepté
      (pas de correctif structurel dans ce chantier — le renommage impacterait
      les arbres existants).

Gate C : `cd apps/go-api && go vet ./... && go test ./internal/media/... ./internal/api/handlers/... ./internal/ops/...`
+ si ffmpeg présent dans le PATH, vérifier que les tests d'intégration media ne
sont PAS skippés (sinon les lancer explicitement).

---

## Clôture chantier

- [ ] Relecture diff complet par le superviseur (altitude : cohérence inter-lots).
- [ ] `go test ./...` complet depuis apps/go-api (pas seulement les packages touchés).
- [ ] Entrée `.ai/thought_log.md` (date, titre, décisions, résultats, suite).
- [ ] Commits par lot sur `fix/media-pipeline-hardening`, push, PR SANS merge
      (revue visuelle utilisateur au merge — push main = deploy prod).

## Protocole de reprise

Avancement = cases de ce fichier + `git log --oneline` de la branche. Reprendre
au premier item non coché du lot en cours. Un lot est clos quand son gate est
passé (code de sortie 0 vérifié) ET ses items statués.
