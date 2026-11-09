# HANDOFF — Kill feed arme par kill (offline) — 2026-06-07

> Handoff pour reprise À FROID. État stratégique = `.ai/V7.5/killweapon/KILLFEED_STATE.md` (source de vérité, à lire en 1er).
> RE mécanisme = `.ai/V7.5/killweapon/KILLFEED_RE_FINDINGS.md`. Décodeur = `.ai/V7.5/film_re/HANDOFF_FRAME_DECODER_L3.md`. Interdits = mémoire
> `project_kill_feed_frame_decoder`. Ce doc = l'OPÉRATIONNEL (fichiers, build, Ghidra, étape suivante détaillée).

## TL;DR
- **But** : arme (FAMILLE suffit, ex « Gravity Hammer ») par kill, OFFLINE en Go, scalable centaines de matchs (film seul, zéro CE).
- **Acquis** : kill feed `tueur·victime·temps·team·slot` = ✅ offline (93/93). Arme-famille présente dans le film = ✅.
- **Gap unique** : attacher la famille à chaque kill.
- **Prochaine action** : Voie A étape 1 = finir le deser i63 `count1>0` → atteindre les 8 bipeds par frame.

## ⛔ NE PAS REFAIRE (sinon on retourne en arrière)
- fire-events / weaponv3 (rejeté, non fiable). dead-state `GlobalID` comme arme (réfuté 0/637 = méthode, pas arme).
- Re-poser « event+0x538 est-il offline ? » : OUI (désérialisé du flux, KILLFEED_RE_FINDINGS §4). Tranché.
- Re-craindre un « mur de largeurs runtime » sur le walk delta : NON (verdict L4 — `FUN_1406d84b4` = largeurs littérales).
- Re-dire « slot→joueur non résolu » : RÉSOLU (chunk_27 `b36` duo + `b37` team = 8 combos bijection/match).

## Environnement
- **Worktree** : `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3`
  (branche `feat/weapon-attribution-v3`). Le code filmdec + cmd y vit.
- **Build/run Go (CGO requis pour duckdb)** :
  ```
  cd <worktree>/apps/go-api
  export PATH="/c/msys64/ucrt64/bin:$PATH" && export CGO_ENABLED=1
  go run ./cmd/<outil> [args]
  ```
- **Données film 000d5950 (Fiesta, match RE de référence)** : cache RACINE
  `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/`
  = `chunk_00.bin` (registre) … `chunk_27.bin` (events) + **`world_dump.txt`** (250 entités, debugger — CRUTCH DEV
  uniquement, PAS prod). NB : le cache du worktree n'a que `chunk_header.bin` ; les outils pointent le cache RACINE en absolu.
- **Fixture match complet jgtm** (pour tests film-seul, sans world_dump) : `apps/go-api/internal/sync/testdata/jgtm_full_match/`
  (filmChunk0=header/registre, filmChunk1=keyframe type-2, filmChunk1..28=réplication, filmChunk29=events ; gitignoré, copié ce jour).

## Code filmdec (`apps/go-api/internal/analysis/filmdec/`)
- `bitreader.go` — BitReader MSB-first big-endian. `registry.go` — ParseRegistryChunk (chunk_00 → 118 archétypes).
- `traverse.go` — TraverseEntity (keyframe) + `SetRecordStateParam` + dispatch `consumeByName`. `world.go` — World slot→archétype.
- `frame_records.go` — DecodeFrameRecords (boucle type-0, auto-bind NEW), FrameConfig{HasExtraFields,IDLowBits}.
- `components_*.go` / `unit_weaponstate.go` / `components_biped_ability.go` — desers biped i0..i63.
- `default_state.go` — consumeBipedDefaultState (keyframe, 120/380 bits + résidu). `entity.go`/`entity_quant.go` — 'obje'.
- Calibration validée 000d5950 : `SetRecordStateParam(2)`, `FrameConfig{false, IDLowBits:11}`.

## Outils cmd (sondes)
- `tmp_worldreplay [chunk] [npkts]` — replay deltas 000d5950 avec world_dump ; sections COUVERTURE BIPED (clean% par slot),
  DIAGNOSTIC i63, RESTE-À-PORTER. **= le banc de test de l'étape 1.**
- `tmp_killfeed_weapons [maxChunk]` — kill feed killer→victime (93) + scan 885 littéraux id64 + timeline.
- `tmp_loadout [mode]` — 8 loadouts keyframe (mode `final`). `tmp_deathfield` — dead-state (méthode, PAS arme).
- `tmp_archlist [filtre]` — dump archétypes du registre. Sondes film-seul : `tmp_kf_offline`, `tmp_kf_bootstrap`,
  `tmp_worldreplay_jgtm`, `tmp_kf_world` (NB : DecodeFrameRecords ne marche PAS sur le keyframe type-2 — grammaire dense distincte).

## Ghidra (HaloInfinite.exe, image_base 0x140000000 ; outils mcp__ghidra__*)
- `FUN_1406730c4` — consommateur kill feed au replay : lVar11=FUN_1404969f0(event) ; [lVar11+0x538]=handle arme ;
  [lVar11+0x1f30]=high-32 famille ; id64 via FUN_140477618(.,0x1003)+FUN_1406713c4 (def+0x478=low).
- `FUN_1404969f0` — résolveur datum-handle→objet (pool TLS, générique).
- `FUN_142f26a20` — deser i63 biped-action : subblock 96b + R(4)=count1 + count1×item + count2(RAM)×… + subblock 96b.
- `FUN_141fd4814` — dispatch tag de l'item i63 : **6 branches tag 0..5** (tag≥6 = erreur). Décompilé ce jour (cf. ci-dessous).
- `FUN_140c1dd44`/`FUN_140c1dce0` — dead-state (méthode/modificateur, PAS l'arme).

## ÉTAPE 1 (prochaine action) — finir i63 `count1>0`
**Où** : `components_biped_ability.go` → `consumeBipedAction` / `consumeBipedActionLoop1Item` / `consumeBipedActionTag`.
**État** : i0..i62 bit-exacts (delta biped). i63 cas commun (count1=0,count2=0)=196 bits porté. Tags 0..5 du dispatch
`FUN_141fd4814` portés et **re-vérifiés ce jour vs décompile** (structure OK). **Mais** `count1>0` désync encore (slot 519
reste 25 % clean ; ~761 désyncs @i63 sur chunk_03).
**Hypothèse** (à confirmer) : une largeur de sous-deser de tag (FUN_143193fe0/FUN_1431bc8a8/FUN_14319572c/FUN_1431a2f10/
FUN_1407f08bc/FUN_1431a3a50) est légèrement fausse → l'item suivant lit un tag≥6 (→ false → désync). Décompiler chaque
sous-deser de tag et vérifier la largeur exacte vs le port Go.
**Banc de test** : `go run ./cmd/tmp_worldreplay 3 30` → section « COUVERTURE BIPED » : slot 519 clean% doit monter de 25 %
vers ~90 %+. Section « DIAGNOSTIC i63 » donne la distribution count1. Quand slot 519 est clean, la boucle atteint les autres
bipeds (512-518) dans la même frame.
**Après i63** : étape 2 = isoler les records de swap d'arme (WST handle@+1 ∈ `analysis.WeaponIDToName`) pour la timeline
arme-famille par slot ; étape 3 = bootstrap World depuis le keyframe du film seul (prod, remplace world_dump) ; étape 4 =
croiser chunk_27 (slot tueur via b36/b37 + temps) → famille par kill ; étape 5 = valider vs CE 12 kills + narration.

## Vérité-terrain (validation)
- CE-validé 12 kills (KILLFEED_RE_FINDINGS §4) : BR75/sniper/marteau/plasma.
- Narration user 000d5950 (film time ≈ +1.5s) : IKE→JGtm marteau @292.5s ; whiteknight→IKE épée @296.3s ;
  Akatsuki→JGtm sniper @308.2s ; JGtm→Akatsuki BR75 @329.8s ; aldus→JGtm marteau @337.7s ; LORD PEINX→aldus @343.2s ;
  aldus→LORD PEINX plasma @344.6s ; IKE→JGtm marteau @355.7s. (Kill feed killer→victime déjà confirmé matcher cette séquence.)
- `shared.medals_earned` (000d5950) = médailles arme par joueur (proxy partiel).
- pi→gamertag : pi0 whiteknight2519 · pi1 JAVIERLOLITO540 · pi2 JGtm · pi3 LORD PEINX13 · pi4 IKE ILYA · pi5 Akatsuki fire17 · pi6 aldusbroncus · pi7 VitaminA1688.

## Productionisation (cap)
`internal/analysis` (pur) → `internal/service` → `internal/api/handlers` ; table append-only ART-safe (cf. ADR 0019) ;
capability-gated (pas de slug `halo_infinite` en dur) ; chemins via PathResolver. Cible d'écriture candidate : `weapon_kills_v3`
(shadow, déjà schématisée) OU colonne `weapon_id` sur `killer_victim_pairs` (décision data à arrêter).
