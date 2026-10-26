# HANDOFF — Kill feed / source de dégât par kill — 2026-06-07

> Reprise à froid. LIS DANS L'ORDRE : ce fichier → `KILLFEED_STATE.md` **§0** (vérité validée) →
> `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` (avancée walk + **PISTE POSITION-LINKING** en bas) → `.ai/V7.5/killweapon/DBG_CAPTURE_dmg_record.md`
> (capture live + runbook debugger). Branche : `feat/weapon-attribution-v3`
> (worktree `.claude/worktrees/weapon-attribution-v3`). Répondre en français.

## ⭐ MEILLEURE PISTE ACTUELLE (2026-06-07) — LIER RECORD→VICTIME PAR POSITION (≠ walk complet, ≠ attaquant)
Née de la question user « dégâts à la victime ». Le record de dégât n'a pas d'ID victime, MAIS a la **position d'impact** ;
et la **position joueur (biped i0)** se lit AVANT la désync. Donc : par kill (victime=slot via chunk_27, temps T) →
position victime à T ⋈ record dont impact-pos ≈ victime-pos près de T → coup létal → ARME. Précision temps = ms.
**BUILD (scopé, détaillé dans WALK_PORT_NOTES.md §"NOUVELLE PISTE") :**
1. CÔTÉ VICTIME : modifier `filmdec.consumeObjectPositionDynamicPrecisionD` (i0) pour RETOURNER la position +
   l'exposer dans `EntityTrace` ; décoder par frame chaque biped (slot via id record) → position par slot/joueur.
2. CÔTÉ IMPACT : porter le deser COMPLET `FUN_14080c1f8` (décompile entière dispo) jusqu'à la position d'impact
   (param_5=1 → `+0x27c` via FUN_140c9e4d8 ; param_5=0 → `+0x2a0` via FUN_1406cd5b8). Étendre `cmd/tmp_dmgscan`.
3. MATCH : par kill, victime-pos(T) ⋈ impact-pos(record près de T) → arme. Valider narration (BR75 JGtm→Akatsuki ;
   marteau IKE→JGtm). C'est l'effort substantiel à mener ; **commencer par le CÔTÉ VICTIME** (plus court, i0 déjà consommé).
> NB : la Voie A "walk complet 8 bipeds" reste possible mais a une LIMITE DURE (i63 count2 = popcount RAM hors flux,
> cf WALK_PORT_NOTES) ; le position-linking la contourne (n'a besoin que de i0, lu avant désync).

## 1. BUT (figé)
La **SOURCE DE DÉGÂTS qui a causé la mort, par kill, FIABLE, tout type** (arme/grenade/mêlée/terrain), OFFLINE
depuis le film, scalable. **Famille suffit** ; un **id brut suffit** (l'user fera la lookup). Cf mémoire
`project_killfeed_damage_source_goal`. INTERDITS (mémoires) : fire-events/weaponv3 (corrélation timing), held-weapon
« au tick » non fiable, dead-state-GID. La Voie A = arme équipée **EXACTE** via ECS (≠ corrélation).

## 2. CE QUI EST ACQUIS (prouvé cette session — ne pas refaire)
- **Mécanisme RE'd** : kill feed lit `report+0x1f30` (famille), écrit au replay par `FUN_1407e00ac` (apply dégât) qui
  LIT le descripteur (pas de re-sim). Désérialiseur du record dégât = `FUN_14080c1f8` ; lecteur générique = `FUN_14080AADE`.
- **CAPTURE LIVE DEBUGGER réussie** (Ghidra dbgeng via ghidra-mcp, fix `go_blocking`) : a donné le framing. Détail +
  runbook reproductible dans `.ai/V7.5/killweapon/DBG_CAPTURE_dmg_record.md`.
- **Records de dégât = paquets type-0 dont `payload[0]==0xd2`**. **519 records** sur 000d5950 (13.8→481.4s).
  **Famille d'arme = global-id `+0x0c`** (`R(5)+R(32 BE)`, le R32 = high-32 = clé `analysis.WeaponIDToName` ; low-32
  `0x42c9679f` contigu → id64). Décode bit-exact validé vs capture (Disruptor). Probes : `cmd/tmp_dmgdecode`,
  `cmd/tmp_dmgscan`, `cmd/tmp_bufmatch`. Décode : payload type-0, **startBit logique=36** (entête 8 o consommée),
  `filmdec.BitReader.Skip(36)`, MSB-first BE.
- **VERDICT source-par-kill via record = NÉGATIF** : le record ne porte NI tueur NI victime (attaquant=4 valeurs
  catégorielles ; victime gate=0 dans 519/519). Appariement seulement temporel → non fiable Fiesta → **narration 0/6**
  (faux positif Stalker sur frag marteau). chunk_27 ne porte pas la méthode (mêlée/grenade non classables ; médailles
  b59 partielles 44/93). Probes : `cmd/tmp_dmgattacker`, `cmd/tmp_killmethod`, `cmd/tmp_killsource`. ⇒ data d'arme OK,
  mais pas la source-par-kill. **NE PAS productioniser l'appariement temporel** (faux positifs prouvés).

## 3. DÉCISION USER = VOIE A (à exécuter)
Lire l'**arme équipée du tueur** au moment du kill (le tueur étant connu), cross-checkée par les 519 records.
Étapes :
1. **Finir le walk biped** (le « long pole ») : porter les branches amont non portées qui désyncent le walk —
   **i54** (ctx+0x9d mid-mobility), **i59** (tag==3 = `FUN_142f25e90`), **i63** (count1>0). Réf décodeur :
   `internal/analysis/filmdec/traverse.go` (dispatch `consumeByName`), `components_biped_ability.go` (i63),
   + `../film_re/HANDOFF_FRAME_DECODER_L3.md` (caveats des branches non portées). **Besoin de Ghidra statique** pour décompiler
   ces fonctions (FUN_142f25e90 + les handlers i54/i63 count1>0) et porter les largeurs bit-exact.
   État walk actuel (sonde `cmd/tmp_worldreplay`) : slot 512 ~98% clean, 515 ~86%, **519 ~29%**. Cible : **8 bipeds/frame**.
2. **Lire WST i43** (= `variant_name` = famille) du **slot tueur** au temps du kill. Slot tueur via chunk_27
   **b36 (duo) + b37 (team)** = bijection per-match (déjà acquis, cf `cmd/tmp_killfeed_weapons`). Loadout initial =
   keyframe (`cmd/tmp_loadout`, 8 records biped #35) + swaps (records WST entre keyframes).
3. **Croiser** : par kill (chunk_27 : tueur slot + temps, 93/93) → arme équipée à t → `tueur·arme·victime`.
   **Cross-check** par les 519 records (la famille équipée doit matcher un record proche en temps).
4. **Valider narration** : marteau IKE ILYA→JGtm (mêlée) ; BR75 JGtm→Akatsuki fire17 (arme). pi→xuid :
   pi2=2533274823110022 JGtm, pi4=2533274815845110 IKE ILYA, pi5=2535444178793711 Akatsuki fire17.
5. **Mêlée/grenade/terrain** : PAS de voie offline fiable identifiée (record absent pour mêlée — Hammer 0/519 ;
   chunk_27 muet). Pour la mêlée : l'arme équipée au moment du frag marteau = le marteau (si dégainé) → le walk
   pourrait la capter ; sinon à re-statuer après le walk. Grenade/terrain = ouvert.

## 4. GHIDRA — OPÉRATIONNEL (résolu 2026-06-07)
`decompile_function` remarche après **redémarrage de Ghidra + plugin MCP** par l'user. (Le bug de reconnexion
mid-session = le pont tente un socket Unix `AF_UNIX` inexistant sur Python Windows → `Schema fetch failed`. Contourné
par un démarrage propre ; si ça recasse en cours de session, redémarrer Ghidra/le plugin, ou nouvelle session Claude.)
Setup : `reference_ghidra_mcp_setup` (HaloInfinite.exe image_base 0x140000000, port 8089).
**AVANCÉE PORT WALK : voir `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`** — i59 (`FUN_142f25e90`) décompilé + structuré (tags 0-6),
helper vec3 `FUN_142f26e9c`→`FUN_14076d528` identifié (= probablement `ReadQuantizedVec3` déjà porté). Restent à
décompiler : helpers `FUN_14076dc04`/`FUN_1408f0ac4`/`FUN_1407f08bc`/`FUN_14076e494` (largeurs) + localiser i54/i63.

## 5. DEBUGGER (secours, opérationnel)
Serveur sur 8099 (peut être relancé). Runbook reproductible complet dans `.ai/V7.5/killweapon/DBG_CAPTURE_dmg_record.md` (attach par
PID, sync_modules base 0x140000000, bp, **`go_blocking` en background** — PAS `go` standard qui gèle, puis lire
registres/mémoire/pile au hit, détacher). Patchs additifs au bridge : `server.py` (endpoint `/debugger/go_blocking`) +
`tracing.py` (handler x64). Usage futur : capter `FUN_1407e00ac` (l'apply, a attaquant `+0x538` + victime en RAM) →
ground-truth par kill pour valider la Voie A (non scalable, validation seulement). NB Theater : les events se
désérialisent au **CHARGEMENT** du film, pas en lecture.

## 6. OUTILLAGE / CONVENTIONS
- Décode film = **Go pur** (pas de CGO) ; duckdb (gamertags) = CGO + `export PATH="/c/msys64/ucrt64/bin:$PATH"` + `CGO_ENABLED=1`.
- Package décode : `apps/go-api/internal/analysis/filmdec` (`BitReader` MSB-first BE, `ParseRegistryChunk`,
  `TraverseEntity`, `DecodeFrameRecords`, `SetRecordStateParam`). Calibration 000d5950 : `SetRecordStateParam(2)`,
  `FrameConfig{HasExtraFields:false, IDLowBits:11}`.
- Film : `data/cache/film_chunks/000d5950/chunk_00..27.bin` (zlib ; header paquet 16 o `[Type u16 LE][b2][b3][Size u32 LE][ts u64 LE]` ; t0Us=4537898226). type-0=frame, type-2=keyframe(chunk_02), type-3=highlight(chunk_27).
- `analysis.WeaponIDToName` : `map[uint64]string` id64(high|low)→nom ; **clés high-32 = familles**. Hammer high32=0x841ac5e5.
- Probes untracked (à ne PAS committer sans accord user) : `cmd/tmp_{dmgdecode,dmgscan,dmgattacker,killmethod,killsource,bufmatch,worldreplay,loadout,killfeed_weapons,wpnentity,...}`.

## 7. DOCS (source de vérité)
- `KILLFEED_STATE.md` **§0** = vérité validée consolidée (LIRE EN PREMIER ; supersede l'historique en cas de conflit).
- `.ai/V7.5/killweapon/DBG_CAPTURE_dmg_record.md` = capture live brute + runbook debugger.
- `.ai/thought_log.md` = journal (entrées 2026-06-07 : capture, décode, verdict, décision Voie A).
- Mémoires clés : `project_killfeed_damage_source_goal`, `project_kill_feed_frame_decoder`, `reference_ghidra_mcp_setup`,
  `reference_killfeed_deadstate_fields`, `feedback_no_fire_events_weaponv3`.

## 8. PROCHAINE ACTION IMMÉDIATE (nouvelle session)
1. Reconnecter Ghidra (`list_instances` → `connect_instance` ; si échec, c'est le quirk → c'est déjà une session fraîche
   donc le pont devrait être OK au démarrage). Tester `decompile_function 142f25e90`.
2. Décompiler `FUN_142f25e90` (i59 tag==3) + le handler i54 (ctx+0x9d) + la branche i63 count1>0 → porter les largeurs
   bit-exact dans filmdec → relancer `cmd/tmp_worldreplay`, viser slot 519 clean ~90 % / 8 bipeds.
3. Câbler la lecture arme-équipée par kill + valider narration.
