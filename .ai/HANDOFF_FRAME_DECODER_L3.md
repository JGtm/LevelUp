# ÉTAT 2026-06-07 — principe PROUVÉ offline + pipeline bootstrappé sur film neuf (LIRE EN PREMIER)

**CE ABANDONNÉ pour l'arme** (cf. thought_log 2026-06-07 + mémoire `project_kill_feed_frame_decoder`) : en Theater le handle `event+0x538` est une réf sérialisée, le résolveur live `FUN_140495abc` plante dessus. L'arme se décode du flux de composants du film, en Go.

**Principe validé sans CE** sur la fixture COMPLÈTE `internal/sync/testdata/jgtm_full_match` (30 chunks, Arena 8 joueurs, copiée gitignorée dans le worktree), via `cmd/tmp_kf_offline` :
- **Kill feed killer→victime : 94/94** offline (chunk events type-3, jointure KILL@t⋈DEATH@t).
- **Arme = id GLOBAL dans le film** (high-32 `WeaponIDToName`) : ~35 familles dans keyframe (type-2) + deltas (type-0). Tranche `REPRISE_KILLWEAPON_FILM.md` : **global stable, pas handle per-match**.

**Bootstrap sur film NEUF (jgtm) — `cmd/tmp_kf_bootstrap`** :
- **Registre OK sans world_dump CE** : `ParseRegistryChunk(inflate(filmChunk0))` = le header (1.97 Mo) = l'ex-`chunk_00.bin`. 118 archétypes ; #35 biped = i0 position, i9 'obje', **i11 dead-state**, **i43-46 WST**. Le registre ne dépend PAS de CE.
- **Keyframe type-2 = `filmChunk1`** (140177 o).
- ⚠️ **Le keyframe stocke `variant_name` = high-32 (FAMILLE) SEUL, pas l'id64 contigu** : `findWeaponLits` (ancre id64-complet de `tmp_loadout`) trouve **0** sur jgtm. ⇒ la calibration loadout doit ancrer sur **high-32** (mon scan high-32 trouve bien les familles), PAS l'id64. Cohérent avec « variante exacte = def runtime, hors film ».

**DÉPENDANCE CE identifiée** : `tmp_worldreplay` (walk delta) est bootstrappé sur `cache/000d5950/world_dump.txt` (capturé au DEBUGGER, slots 512-519=bipeds). Pour un film neuf offline, le World doit être **initialisé depuis le keyframe** (traversée type-2), pas un dump debugger. C'est le prochain jalon infra.

**ROADMAP weapon-par-kill tous joueurs (build conséquent, multi-étapes)** :
1. World bootstrap depuis keyframe jgtm (8 bipeds loadouts via décode WST structuré, ancré high-32) → remplace world_dump CE.
2. Replay deltas type-0 tous bipeds (porter i63 count1>0 + archétypes non-biped ; recordStateParam) — actuellement fiable seulement slot 519.
3. Lire `DeadState` (i11) à la frame de mort de chaque victime → `GlobalID` (@comp+0x10, réf arme/source) + `EnumA/EnumB` (type-dégât).
4. Résoudre `GlobalID` → arme (global-id → entité source → 'obje' famille) OU exploiter EnumA/B.
5. **slot→player-index→xuid** (NON résolu : pas d'ancre xuid au keyframe).
6. Croiser chunk_27 (killer/victime/time 94/94) → kill feed nommé.

**BLOCAGES OUVERTS** : (a) largeurs quantifiées delta sourcées CE (axes 6/6/6) → re-dériver statiquement pour 100 % CE-free ; (b) slot→xuid ; (c) chaîne GlobalID→arme à valider empiriquement (la constante CE +0x10=116963283 était la forme RAM, PAS le champ répliqué — à vérifier sur de vrais morts) ; (d) **variante exacte = famille seule** offline.

**EMPIRIQUE 2026-06-07 PM (replay delta sur jgtm, `cmd/tmp_worldreplay_jgtm` + `cmd/tmp_kf_world`)** :
- **Slots biped = match-spécifiques** : PAS 512-519 sur jgtm (c'était le dump CE de 000d5950). Le top-8 du 1er-record varie selon idLowBits ; aucun config ne nomme d'arme en delta (namedWST=0) car en delta le WST gate=0 = « arme inchangée » (l'arme n'est (re)transmise qu'au swap/pickup, gate=1). ⇒ l'arme initiale vient du KEYFRAME, les deltas = swaps, le dead-state = réf par kill.
- **La boucle de records CALE après ~1 record** sans World complet (`records≈paquets`) : un delta sur un slot non-bindé désync → fin de paquet. Confirme qu'il faut le **World complet** (tous slots→archétype).
- **`DecodeFrameRecords` ne s'applique PAS au keyframe** (`tmp_kf_world`) : type-2 ET type-1 calent à endBit≈11-19 (0-1 record). Le keyframe a une **grammaire dense distincte** (préambule + records à default-state lourd), traversée par calibration (brute-force start, cf `tmp_loadout`), pas par la boucle type-0.

⇒ **LE morceau restant = walker complet du KEYFRAME** (décoder chaque record entité = R(6) typeIndex + default-state + composants, TOUS archétypes) pour bâtir le World slot→archétype d'un film neuf, SANS dump CE. Puis : replay deltas (World complet) → maj held-weapon (swaps gate=1) + capture dead-state (i11) aux morts → `GlobalID`/EnumA-B → croiser chunk_27. C'est de l'ingénierie de décodeur (porter les desers d'archétypes + largeurs default-state du keyframe), pas un mur de données.

Sondes neuves : `cmd/tmp_kf_offline` (kill feed + scan arme global), `cmd/tmp_kf_bootstrap` (registre+keyframe film neuf), `cmd/tmp_worldreplay_jgtm` (replay delta film neuf, découverte slots biped), `cmd/tmp_kf_world` (test keyframe→World).

---

# HANDOFF — Décodeur FRAME L3 (arme du kill feed depuis le film) — 2026-06-05

## 🎯 ÉTAT POUR REPRISE (2026-06-06) — LIRE EN PREMIER (fait autorité sur tout le texte « held@tick » plus bas, RÉTRACTÉ)

⛔ **INTERDIT** (rejeté ~10× par l'user, c'est FAUX) : « held weapon au tick », reconstruction ECS de l'arme, « timeline held-weapon ». Tout passage du HANDOFF qui en parle est **mort**. Réf ancre : mémoire `project_kill_feed_frame_decoder`.

**BUT** : reverse-engineer le **KILL FEED** du jeu — comment il affiche `tueur · arme/méthode · victime` (+ suicides : joueur·méthode ; départs/arrivées).

**MÉCANISME TROUVÉ (Ghidra, factuel)** : le widget KillFeed (`FUN_14066b5e8`) lit l'arme dans l'**ÉVÉNEMENT DE MORT = le composant `object-dead-state` de la VICTIME**. Forme lourde biped = **`FUN_140c1dd44`** (lue quand le flag de mort est posé). Elle écrit la cause de mort :
- `+4`, `+8` : **2 ENUMS** (`FUN_1407f2058` = R(1) ; si 0 → R(5)), résolus par table de tags → type-de-dégât / méthode.
- `+0x10` : **RÉFÉRENCE** = global-id R(32) résolu par `GetLocalHandleFromGlobalId` (`FUN_14080d61c`, table `DAT_144b404f0`) — **même mécanisme que l'arme du WST** → candidat fort ARME/source-de-dégât.
- (+ position de mort `+0x20`/`+0x2c`). ⇒ c'est un **enum / référence à MAPPER** (pas l'arme en clair — l'user l'avait prédit).

**ACQUIS** :
- **Events kill feed décodés** (chunk_27, `highlight_event_parser.go`) : tueur↔victime par jointure temporelle KILL@t⋈DEATH@t équipe-opposée = **93/93 zéro erreur** ; `team` (block[37], 8/8 vs match_participants) ; `slot` (block[36]) ; `time_ms`. L'arme N'EST PAS dans ce bloc 60o (prouvé). Sonde : `cmd/tmp_killfeed_decode`.
- **Grammaire `FUN_140c1dd44` portée bit-exact** dans `components_object.go` (struct `DeadState`, bloc `+0x10` corrigé).
- Décodeur FRAME : World capturé (debugger, bipeds = slots 512-519), `FrameConfig{HasExtraFields:false, IDLowBits:11}`, `recordStateParam=2`. Desers biped i0-i63 + non-biped (game-engine-team-mapping) portés.

**LE BLOCAGE (le seul, précis)** : le décodeur de **DELTAS n'est pas bit-exact AVANT i11** → le dead-state lu est **désaligné**. Cause = **largeurs runtime de composants à forme delta data-dépendante** (absentes du .exe statique). **Ce n'est NI une limite de donnée, NI du held-weapon : c'est un trou de décodeur.**

**AVANCÉE 2026-06-06 (session reprise)** — RE Ghidra + fix + pipeline de calibration :
- **Struct BitReader figé** : `+0x2c` = curseur (bits consommés), `+0x30` registre 64b MSB, `+0x38` bits du mot courant, `+0x40` ptr octets, `+0x10` fin. **Itérateur FUN_14076cb60** : deser dispatché par `vtable+0x28(desc, bitreader, record, &out, param_4)` ; **`param_4`(=recordStateParam) = `vtable[0](desc)` = RUNTIME** (jamais dans le stream). hasExtraFields=OFF. Site dispatch `14076cd19 CALL [RAX+0x28]`.
- **Fix i0 predicted-delta APPLIQUÉ** (FUN_1406cfe44, `components_movement.go`) : suppression d'un bit de contrôle FANTÔME (`precSelect`) + queue handle re-gatée par flag RUNTIME `PositionDeltaHasHandleTail` (défaut false), pas un bit du stream. BUILD+VET OK. ⇒ deltas biped « clean » 8086/25545 (vs désync précoce avant), 3161 atteignent i11.
- **MAIS spine PAS encore bit-exact** (validation dure `cmd/tmp_deltacal replay`) : variante WST (i43, après i11) = **0% match catalogue** (3451 variantes = bruit) ; 3443/29257 frames terminent proprement seulement. « clean »=faux positif. Le bug n'est PAS i0 (records i0=absolute ET i0=absent échouent aussi) → composant à forme delta data-dépendante (i15 low-freq=1010b, i20, i6/i7…). Keyframe n'aide pas (WST 167b DENSE vs 11b SPARSE ; i11 ABSENT du keyframe).
- **Dead-state vérifié Ghidra** (FUN_140c1dce0 wrapper + tête FUN_140c1dd44) : **tête jusqu'à la référence arme +0x10 = BIT-EXACTE** (handle R1+optR32 ; FUN_140c1e3f0=R8 confirmé ; 2× enum R1+optR5 ; FUN_1406d310c(10)=4 → R4 +0xc ; R3 +0xe ; bloc +0x10 : presentA bit1 → gidPresent R1 → GID R32 → +0x14 R3 → +0x18 R1+optR6). La queue (position/orientation/enums) a des under-reads APRÈS la réf arme → sans effet sur elle. ⇒ dès que le spine atterrit juste sur i11, +0x10 se lit.

**PROCHAINE BRIQUE (reprise) — UN seul item bloquant = action utilisateur (capture CE)** :
1. **Capture Cheat Engine** : `tools/ce/filmdec_delta_capture.lua` (AOB module-relatif, code cave, hook unique sur le dispatch). Lancer Halo→Theater→film **000d5950**, `startFilmdecCapture()`, lire qq s de deltas (idéalement autour d'un kill), `stopFilmdecCapture()`, `dumpFilmdecCapture([[chemin.csv]])`. → CSV (compIndex, param_4, curseur de bits par delta biped).
2. **Diff** : `go run ./cmd/tmp_deltacal ingest <csv>` → table largeur+param_4 par composant + **diff Go(replay) vs CE(capture)** → **1er écart de largeur = le deser delta à corriger**.
3. Corriger ce deser → re-`replay` → WST doit matcher ~100% → spine bit-exact → atterrissage i11 correct → réf arme +0x10 lisible → mapper (catalogue + UI/CE) → croiser events 93/93 → kill feed complet.

**Sondes** : `cmd/tmp_deltacal` (replay+ingest+diff, NOUVEAU), `cmd/tmp_deathfield` (capture dead-state aux morts), `cmd/tmp_killfeed_decode` (events), `cmd/tmp_worldreplay` (deltas+World), `cmd/tmp_i48trace` (traversée keyframe). Outil CE : `tools/ce/filmdec_delta_capture.lua`.

---

> Doc de reprise consolidé. Capture l'état après une longue session d'élimination méthodique.
> Détails grammaire/architecture : `PLAN_FILM_KILLFEED_V3.md`. RE exe : `RE_EXE_GHIDRA_FINDINGS.md`.
> Reprise antérieure : `REPRISE_KILLWEAPON_FILM.md`.

## But (modèle utilisateur, CADRÉ — ne plus le redéfinir)

L'**arme affichée par le kill feed** = une **RÉFÉRENCE** (handle / entity-id) attachée à l'**événement de kill/mort** (le DamageReport), résolue au replay via le composant ECS `'obje'` de l'entité référencée. C'est « le **tir** d'un joueur avec l'arme X rattaché à la **mort** » (formulation user). 

**Termes BANNIS** (rejetés fermement par l'utilisateur, ne JAMAIS reproposer) :
- « weapon held » / arme dans la main au tick / held-weapon resolution.
- fire-events / weaponv3 / corrélation temporelle tir↔kill (= v1/v2, jugé non fiable).

## Ce qui est FERMÉ (exhaustivement vérifié — NE PAS REFAIRE)

1. **chunk_27 (highlight events) = AUCUNE arme.** Il porte victime + tueur (les death-type20 et kill-type50 sont **adjacents** : 74/93 deaths suivis du kill correspondant) + temps + un champ à **8 valeurs distinctes** (= un player-index, le tueur). Vérifié **avant / dans / après** le death event : 0 littéral d'arme, 0 champ à >8 valeurs distinctes (donc pas d'enum d'arme ~15 valeurs). Sondes : `cmd/tmp_killfeed`, `cmd/tmp_death`, `cmd/tmp_after`.
2. **CE value-scan de l'id d'arme = FERMÉ.** L'arme du kill = `{handle, weapon-index}` résolu au RENDER (`FUN_1420ca9a0`), pas un id stocké qui change par kill. Le scan ne tombe que sur les **entités-armes statiques** (gros tableau stride **0x1448**, ~250 slots = pool d'entités monde) + les **défs statiques** (`HaloInfinite.exe+487FC58/4887A18/4DC67A0/4DC7C00`). Le Next-Scan (autre arme) ne garde rien → confirme que ce n'est pas une valeur mutable.
3. **Marqueur littéral = REJETÉ** (workflow agent, 8 sondes + groupes de CONTRÔLE) : `[killer_pi][arme][victim_pi]` = 31% vs **28% contrôle aléatoire** = bruit ; FourCC `'obje'` absent des type-0 ; new/delta au tick non-discriminant ; littéraux d'arme trop rares au tick (1/93 ≤5ms).
4. **Flux `d2`** (519 paquets type-0, weapon-id 64-bit @bit44, préambule `d2 6X X4 NN`, slot `(byte1,byte2hi)` **90%+ pur par arme**, byte3 = compteur monotone) = **update des armes-MONDE ambiantes, SANS référence au porteur**. → utile pour **RÉSOUDRE** un entity-id→arme (étape L5), inutile pour identifier l'arme du kill seul.

**Conclusion** : l'arme du kill n'existe nulle part comme valeur directe. Elle = une **référence dans le DamageReport du flux gameplay type-0**, résoluble UNIQUEMENT via le décodeur FRAME. Tous les raccourcis sont éliminés, proprement.

## La SEULE voie = décodeur FRAME-delta (L3) — grammaire confirmée (Ghidra)

- **Paquets** : type-0 = FRAME (~60fps, 1199/chunk 20s ; type-10 = moitié appairée). Certaines frames préfixées du FrameMarker byte-aligné `a0 7b 42` (marqueur de POSITION, **pas universel**).
- **Boucle records `FUN_1406cd128`** : par record → (option R(32) si flag global cVar1) ; `type` ~2 bits (1=new/2=del/3=delta/0=fin) ; `id` = R(7) (via FUN_1406d3140) = entity-index (table param_1+0x38, stride **0xa0**, porte l'archétype) ; dispatch 1→`FUN_141f86704`, 2→R(32), 3→`FUN_141f86b58` ; records poussés (stride 0xc0) → `FUN_1406cbaa0`.
- **Delta `FUN_141f86b58`** = **bonne nouvelle** : archétype (`FUN_1406cb5f0`) + memcpy état précédent (lVar4+0xb8) + `FUN_14076cb60` (itérateur composants = mask `FUN_1406d7610` + composants présents). **PAS de default-state dans le bitstream** (en RAM). ⇒ le delta = `filmdec.consumeMask` + `consumeByName`, **sans** le `R(6) typeIndex + default-state` du keyframe. **Plus simple que le keyframe** (qui butait sur le default-state non-mesuré).
- **Résolution arme (L5)** : référence/handle → `FUN_14049a384` (handle→objet) → `FUN_1405839d0(entity,'obje'=0x6f626a65)` → arme = u32 asset-id à `srcItem+0x18` (= high-32 famille, cf `analysis.WeaponIDToName`).

## Workflow lancé (en cours) : `frame-decoder-grammar` (runId w64dqomkk)

Extrait en parallèle la grammaire bit-exacte de : FUN_1406cd128 (boucle), FUN_14076cb60+FUN_1406d7610 (itérateur+mask), FUN_141f86704+FUN_141f86b58 (new+delta), FUN_1406d3140+FUN_1406cb5f0 (primitives+archétype), FUN_1406cbaa0 (dispatch) + chaîne de résolution arme. → **synthèse = spec d'implémentation L3**. Récupérer le résultat à la complétion.

## Prochaines étapes (build L3 sur `internal/analysis/filmdec`, NE PAS réinventer les desers)

1. **L3 boucle records** : parser le payload type-0 `[type 2b][id 7b]` + dispatch (selon spec workflow).
2. **Mapping entity-id → archétype** : init depuis le keyframe type-2 (entités R6 typeIndex) + maj sur records `new` (type 1).
3. **Delta** : `consumeMask` + `consumeByName` selon les composants de l'archétype → atteindre la référence d'arme / le composant pertinent.
4. **Fermer les desyncs desers** (i12/14/16 etc.) avec une vérité-terrain (regarder le replay : arme réelle par kill — value-scan CE fermé).
5. **L5** : par kill (chunk_27 : tueur+victime adjacents, temps) → DamageReport au tick → référence d'arme → résolution `d2`/'obje' → arme → `killer_victim_pairs.weapon_id`.

## Sondes throwaway de cette session (cmd/, worktree weapon-attribution-v3)

`tmp_matchinfo` (DB match 000d5950 : Cliffhanger Super Fiesta, 8 mars 2026 ; roster + pi map), `tmp_wpnscan` (scan high-32 keyframes), `tmp_pidx`/`tmp_slotxor`/`tmp_b5keyframe`/`tmp_formulaa`/`tmp_trackdist` (recherche player-index keyframe = NÉGATIF), `tmp_killfeed`/`tmp_death`/`tmp_after` (kill/death events chunk_27 = pas d'arme), `tmp_killweapon` (known-plaintext = ambigu), `tmp_frame`/`tmp_packettypes` (structure type-0), `wf_marker_*` (agent, marqueur rejeté). 

## Mapping pi→xuid (bit-vérifié, réutilisable)

pi0=2535467794760703 · pi1=2535437947245250 · **pi2=JGtm 2533274823110022** · pi3=2533274980284321 · pi4=2533274815845110 · pi5=2535444178793711 · pi6=2533274882097883 · pi7=2533274826120416. t0 (chunk_02 type-2 ts) = 4537898226 ; ts_kill = t0 + time_ms×1000.

## L3 — Grammaire CORRIGÉE (workflow frame-decoder-grammar) + code IMPLÉMENTÉ (Étapes 0-2, 2026-06-05)

**Corrections décisives vs la grammaire naïve `[type 2b][id 7b]`** (2 pièges d'un port naïf) :
- **TYPE = PREFIX CODE**, pas R(2) plat : `R(1)` ; si 1 → DELTA (1 bit) ; si 0 → `R(2)` ∈ {0=fin,1=new,2=del,3=delta}. Optimise le cas commun (delta) à 1 bit.
- **ID = table-driven**, pas R(7) fixe (le 7 = index de slot) : `low = R(idLowBits) + idBase` ; `tag = R(2)` en bits 30-31 ; `id = (tag<<30)|(low&0x3fffffff)` ; `slot = id&0x3fffffff`. `idLowBits = bit_length(entityCount)` ≈ 13 (runtime).
- **DELTA (type-3) = AUCUN default-state, AUCUN R(6), AUCUN gate** : archétype + état de base en RAM (World) ; bitstream = `consumeMask` + composants présents seulement. (NEW garde R(6)+default-state+gate = `TraverseEntity`.)
- Préfixe **R(32) par record + gardes R(8)** seulement si `hasExtraFields` (cVar1 global ; **hypothèse film offline = false**).

**Code livré (compile + vet OK)** dans `internal/analysis/filmdec/` :
- `traverse.go` : `traverseComponentLoop(br, arch, *t)` extrait de `TraverseEntity` (corps partagé NEW/delta).
- `world.go` : `World` {slot→(TypeIndex, FullID, HeldWeapon)} + `BindFull/Unbind/ArchetypeForSlot/SetHeldWeapon/HeldWeapon`.
- `frame_records.go` : `FrameConfig` (HasExtraFields/IDLowBits/IDBase/NewDefaultStateBits + `DefaultFrameConfig`) ; `readRecordType` (prefix code) ; `readRecordID` (table-driven) ; `decodeDelta` (mask + loop, sans header) ; `DecodeFrameRecords` (boucle type-0 complète + maintenance World).

**RESTE (Étapes 3-5, prochaine session)** :
3. Brancher `DecodeFrameRecords` sur un paquet **type-0 réel** (extractType pour typ==0). Valider : la boucle se termine sur type==0 proprement (DesyncAt==-1, BitPos ≈ fin de paquet).
4. **Calibrer `FrameConfig`** par SWEEP (comme `sweepI0`) : `IDLowBits ∈ {0..16}` × `HasExtraFields ∈ {false,true}` × `NewDefaultStateBits` — retenir la combo qui termine proprement ET aligne les armes connues (`knownWeapon` high-32). Départ : false/13/0.
5. **Cache + L5** : `SetHeldWeapon` quand un delta porte `weapon-state-type-info` ; sinon garder le cache ; croiser kill-feed (time_ms, killer/victim) ↔ obje(i9)=identité → `killer_victim_pairs.weapon_id`.

**Risques ouverts (par proba de blocage)** : (1) `NewDefaultStateBits` (largeur codec archétype, runtime, brute-force) ; (2) `idLowBits` (table runtime, sweep) ; (3) `hasExtraFields` (config, sweep) ; (4) le type-2 keyframe partage-t-il ce header ? (sinon init World via ancres = OPTION B) ; (5) desers non-portés (i15 tail, biped-* stubs, quat) → desync si présents avant l'arme dans un delta ; (6) `recordStateParam` biped (runtime). La STRUCTURE est sûre (décompile+asm) ; le risque résiduel = largeurs runtime + couverture desers. Spec complet : output workflow w64dqomkk.

## L3 — Vérification/correction des desers (workflow frame-deser-fix wgs1wq37o, 2026-06-05)

**Départ de boucle type-0 = BIT 0 du payload** (marker `a0 7b 42` INCLUS, ne PAS skip 24 bits ; le marker = encodage naturel des 1ers records delta-position, pas un en-tête). ⇒ `frame_records.go::DecodeFrameRecords(NewBitReader(p.data), ...)` est CORRECT. (La sonde `tmp_frame` pos:24 + header flat était fausse — ignorer.)

**Statut des desers (vérif Ghidra)** :
- **BIT-EXACTS (aucune correction)** : i1 (transl-vel), i3 (angular-vel), i6 (region-state), i7 (damage-sections), i8 (constraint), i10 (parent-state — CODE ok), i12 (scale), i13 (max-vit), i14 (dissolver), i16 (physics-flags), i17 (frame-config, modulo réserve polarité consumeGateR/FUN_1406d1024), i20/i23 (corps), **i43 held-weapon (l'arme R(32) via FUN_14080dec4 est SAINE)**.
- **À CORRIGER (code)** :
  1. **i0 predicted-delta** (components_movement.go `consumeObjectPositionDynamicPrecisionD`) — 2 bugs : present-flag manquant + le sous-cas dominant lit `3×R(8)` SIGNÉS FIXE (PAS `3×R(axisW)`) ; `bExt` inline manquant ; tail `consumePositionHandleTail` sous-spécifié (handle-resolve FUN_1408f0ac4 non compté). Grammaire i0 canonique + code corrigé : output wgs1wq37o. NB : i0 absolu (keyframe) est OK ; le bug ne touche que les deltas frame-à-frame.
  2. **i19 `consume1406d025c`** (unit_weaponstate.go) — SOURCE #1 de désync zone unit-* : largeur 8 (pas 10) → `consumeGateR(br,8)` ; 2× `consumeId2` à GATER sur flags a/b/c ; tail FUN_142f26740/FUN_140c9e4d8 ABSENT (décompiler FUN_140c9e990/FUN_140c9e738).
  3. **i15 object-low-frequency tail** (components_object_state.go) — CORE ok, 3 sous-readers de queue manquants (ef6d4/ef520/ef4c8, ≥5 bits ; décompiler FUN_142af27f8/FUN_1424d9a30/FUN_141015740).

**Param runtime #1 = `recordStateParam`** (figé 0 dans traverse.go). **C'est la cause du désync i10** (pas un bug code). Affecte i10/i19/i20/i23. **CÂBLÉ en variable** : `var recordStateParam` + `filmdec.SetRecordStateParam(v)`. **Calibrer par sweep {0,1,2,3}** : retenir la valeur qui re-synchronise i12 après i10 sur un record BIPED (typeIndex=35). Autres params : i0 axisW/IndexW (déjà 6/6/6+1, n'affecte que l'absolu), FUN_1406cb0cc (résidu tail i0), dequants (i19 f2=9, i20=10…).

**VERDICT chaîne keyframe** : la 1ère entité BIPED ne chaîne PAS encore past i10 (param recordStateParam, pas bug). Avec (a) fix i0 predicted + (b) calibration recordStateParam → franchit i10→i12. Restent bloquants ensuite : i19 `consume1406d025c` + i15 tail. **i43 (l'arme) est sain — la désync est strictement EN AMONT.**

**PROCHAINES ÉTAPES (ordre du workflow)** : (1) recordStateParam var [FAIT] ; (2) instrumenter compteur de bits + sweep recordStateParam sur un record biped → valeur qui re-sync i12 ; (3) appliquer corrections code i0/i19/i15 ; (4) décompiler les sous-blocs résiduels (tails i19/i15, FUN_1406cb0cc) ; (5) re-tester traversée i0..i46 → held-weapon i43 aligné. Corrections code complètes : output workflow **wgs1wq37o**.

## L3 — Corrections desers APPLIQUÉES + JALON traversée biped (workflow wz2dr3l5i, 2026-06-06)

**Les 3 desers corrigés sont APPLIQUÉS et COMPILENT (`BUILD+VET OK`)** :
- i0 (`components_movement.go`) : predicted-delta réécrit (bExt, present-flag double, **3×R(8) signés** dominant vs 3×R(axisW), `consumeAbsoluteWithGate` factorisé, tail handle-resolve `R(IndexW)+R(2)`, `PositionFullPrecision` var). `FUN_1406cb0cc` confirmé = **0 bit** (prédicat runtime).
- i19 (`unit_weaponstate.go`) + i15 (`components_object_state.go`) : déjà appliqués par les agents (width 8, gates, tails portés). Idempotents.
- `recordStateParam` câblé en var (`SetRecordStateParam`).

**JALON MAJEUR — la traversée ECS biped FONCTIONNE** (`cmd/tmp_bipedcal`) : avec recordStateParam=0 + default-state≈88, `TraverseEntity` sur un biped (typeIndex=35) du keyframe consomme **29 composants bit-portés** (i5 santé → i11 mort → i12 scale → i15 low-freq → i18-i29 unit-* → i30-i40 weapon-state-ammo/rounds/overheated → **i43/i45 weapon-state-type-info**) et désync proprement à **i48 (biped-desired-ability-set, NON porté, APRÈS l'arme)**. ⇒ le framework + les desers corrigés sont **fonctionnels jusqu'à l'arme**.

**RÉSIDUEL à fermer (3 pistes, par proba)** :
1. **Les bipeds brute-forcés ≠ les 8 records de LOADOUT.** Leurs i43 tombent ~194313, alors que les 16 littéraux d'armes connus sont à **195323+**. La fenêtre de brute-force attrape d'autres bipeds (bots/non-loadout). → Re-cibler : brute-forcer le biped dont i43 (handle OU variant) atterrit À 195323 == Hydra (0x767db96d).
2. **Encodage held-weapon keyframe DENSE vs deser type-1.** Rappel angle-C : dans le type-2 dense, les armes tenues sont des **littéraux u32 BRUTS à pas +196**, pas forcément la forme gate+handle+variant du deser FUN_1407f06bc (qui régit le type-1 sparse). Le i43 décode handle=0x0000ff00/variant=0x00002501 = pas une arme → soit désalignement résiduel, soit forme dense différente. **Le vrai objectif = les DELTAS type-0** (qui utilisent l'itérateur de composants, pas la forme dense) → tester là quand le World est initialisé.
3. **Désalignement résiduel** d'un composant pré-i43 (default-state approximatif au sweep, ou un deser off-by-N parmi i15/i19/i20/weapon-state-ammo). À fermer en ancrant sur un i43 == arme connue.

**PROCHAIN PAS** : (a) ancrer le brute-force sur i43==Hydra@195323 (lever handle/variant + valider l'alignement) ; (b) porter i48 (biped-desired-ability-set) pour chaîner au-delà ; (c) initialiser le World depuis le keyframe puis tester `DecodeFrameRecords` sur un paquet type-0 (le vrai chemin). Sonde : `cmd/tmp_bipedcal`.

## L3 — TOURNANT : modèle CORRIGÉ + bug polarité corrigé + calibration trouvée (2026-06-06, session parallèle)

> 5 agents parallèles (pistes A/B/C puis K1/K2) + vérif Ghidra directe. Cette section **SUPERSEDE** plusieurs conclusions ci-dessus. Lire en priorité.

### ⚠️ RÉTRACTÉ — le modèle "handle → entité-arme séparée"
La conclusion de la Piste A « le biped porte un handle (0x0000ff00) vers une entité-arme » est **FAUSSE** : c'était du **bruit de désalignement** (mauvaise calibration `defaultBits=88, rsp=0`). **Il n'existe AUCUN archétype entité-arme** : dans le registre (118 archétypes), **seul le biped #35 porte `weapon-state-type-info`**. L'**arme est encodée DIRECTEMENT dans le WST du biped** (le champ R(32) "variant-name" = high-32 famille ; le R(32) suivant = low-32 suffixe `0x42c9679f`, ensemble = id64 du catalogue `analysis.WeaponIDToName`). Pas de résolution de handle à faire pour l'attribution. (Le handle 0x010xxxxx est dérivé EN MÉMOIRE par `GetLocalHandleFromGlobalId` (`FUN_14080d61c`), ce n'est PAS un champ du film — décompose en `slot=handle>>13`, `gen=handle&0x1FFF` si jamais utile pour les deltas.)

### ✅ CALIBRATION TROUVÉE (cornerstone, reproduit 2×)
Biped #35 du keyframe, **`start=194126, defaultBits=380, recordStateParam=2`** → traversée bit-exacte qui place le WST **i45 @bit195322 = `handle=0x767db96d` = MLRS-2 Hydra** (aligné PILE sur le littéral connu), puis i46, puis désync à **i47 `biped-desired-grenade-set-component`** (non porté). C'est la **1ère arme réelle lue d'un biped via la traversée ECS** — preuve que toute la chaîne (desers i0→i44 + fix polarité) est bit-exacte à cette calibration. NB : `defaultBits` du biped ≈ **380** (PAS 88 ; le 88 de la Piste A était faux) ; `recordStateParam=2`. Sonde de repro : `cmd/tmp_i48trace` (2 calibrations comparées).

### ✅ BUG DE POLARITÉ CORRIGÉ (sur le chemin de l'arme) — vérifié Ghidra direct
`FUN_1406d1024` (w=6) lit son payload quand le **gate bit == 0** (sentinelle 0xffffffff si bit==1) — polarité **INVERSE** de `consumeGateR`/`FUN_140c50d1c` (vérifiés bit==1 : 140c50d1c, 140cec0a0, 140e82b84). L'ancien code modélisait `FUN_1406d1024` avec `consumeGateR(br,6)` (faux) DANS `consume1407f0550`, **à l'intérieur du deser d'arme i43/i45** → décalait i46+. Corrigé : nouveau helper **`consumeGate0R(br,width)`** (lit si bit==0), appliqué à `consume1407f0550` et au nouvel i48. Fichiers : `unit_weaponstate.go`.

### ✅ i48 PORTÉ — `consumeBipedDesiredAbilitySet` (FUN_1406d0ff0)
`R(3)` + `consumeGate0R(br,6)` = 4 bits (gate==1) ou 10 (gate==0). Bit-exact certain. Nouveau fichier `components_biped_ability.go` + dispatch `case "biped-desired-ability-set"[-component]` dans `traverse.go`. **BUILD+VET OK.**

### Loadouts des 8 joueurs au keyframe (16 littéraux WST, K2 — tous typeIndex=35, i43..i46)
bit195323 **Hydra** | 195519 Rushdown Hammer · 198140 **Shock Rifle** | 198335 Rushdown Hammer · 200933 **M41 SPNKr** | 201128 Heatwave · 203736 **Mangler** | 203931 Ravager · 206529 **MA40 AR** | 206725 Disruptor · 209339 **Mangler** | 209534 Skewer · 212127 **Cindershot** | 212323 Shock Rifle · 214922 **CQS48 Bulldog** | 215118 M41 SPNKr.

### K1 — pas d'énumération cheap (make-or-break tranché : NON)
Le keyframe type-2 = **1 seul paquet** (142 695 o) = cascade de records **NEW full-state**, **même boucle/vtable que type-0** (`FUN_1406cd128` décode + `FUN_142f2913c` replay, vtable `0x1436a8810` slots +0x10/+0x30 ; dispatch `FUN_1406cbaa0`). **Aucun** préfixe de longueur, **aucun** marqueur `a0 7b 42` (0 occurrence), **aucune** table d'entités. L'avance record→record = **décodage bit-exact obligatoire** du corps. Bornage = plafond de compte de records (`param_4`), sortie propre = type==end. ⇒ World = à bâtir en décodant réellement, **amorçage incrémental** : chaque NEW `BindFull(slot,typeIndex)` ; à chaque désync, `DesyncAt` pointe l'archétype à porter ensuite.

### Réserve (non bloquante) — deser WST 1 vs 2 R(32)
K2 note que `FUN_1407f06bc` ne lirait qu'**un** R(32) (variant-name=high-32) là où `consumeWeaponStateTypeInfoVariant` (components_object.go) en lit **deux**. Empiriquement la traversée reste alignée (i45→i46 propre) et l'id64 (high|low) matche le catalogue → OK pour l'attribution. À auditer si on veut un WST bit-parfait (i46 tombe ~30 bits avant le littéral Hammer attendu → longueur interne WST data-dépendante à raffiner). NON bloquant pour lire l'arme primaire.

### PROCHAINES ÉTAPES (ordonnées, fondation prouvée)
1. **Porter i47 `biped-desired-grenade-set-component`** (sibling d'i42/i48, sans doute même famille `R(3)+gate`) + **i49 `biped-control-context-component`** → faire traverser un biped jusqu'à `DesyncAt==-1` (fin de record connue). LEAD i47 (Ghidra) : string `"biped-desired-grenade-set-component"` @ **143c98e60** (adjacent à i48 @143c98ec8) ; xref DATA depuis **1411775e0** (descripteur — remonter au thunk deser `+0x28` puis au `JMP 0x1406d0xxx`, cluster des biped-desired-* : i42=FUN_1406d01fc, i48=FUN_1406d0ff0). i49 `biped-control-context-component` : même méthode (search_strings).
2. **Fixer `defaultBits` du biped #35** (≈380) : confirmer constant sur plusieurs bipeds OU déterminer d'où vient la largeur (runtime). Idem `recordStateParam=2`.
3. **Walk keyframe** : décoder l'en-tête de boucle (prefix-code type + id + R6 typeIndex), pour chaque NEW skip default-state + traverser comps + avancer → World (slot→typeIndex) + lire l'arme de chaque biped (WST i43..i46). Porter les archétypes non-biped rencontrés au fil des désyncs.
4. **Mapper biped→joueur** : via i9 `object-multiplayer-properties` ('obje', `DecodeEntityRecordQ`) ou l'id/slot du record ↔ pi→xuid (map ci-dessus).
5. **Rejouer les deltas type-0** (World amorcé) pour suivre les changements d'arme dans le temps ; croiser kill-feed (chunk_27 : tueur+victime adjacents, time_ms) → arme du tueur au tick → `killer_victim_pairs.weapon_id` (L5).

Sondes de cette session : `cmd/tmp_i48trace` (repro calibration), `cmd/tmp_kfframe` (K1 framing), `cmd/tmp_wpnentity` (K2 archétypes+littéraux). Code livré : `components_biped_ability.go` (i48), `unit_weaponstate.go` (consumeGate0R + fix consume1407f0550), `traverse.go` (dispatch i48).

### ✅✅ JALON — biped Hydra traverse INTÉGRALEMENT (DesyncAt==-1, endBit=195892)
En portant 6 composants biped supplémentaires (famille `biped-*`/`simulation-*`), le record biped #35 porteur de la Hydra (start=194126, defaultBits=380, rsp=2) **se décode de bout en bout** : 29 composants présents (i5→i62), aucun trou de bits, i45=Hydra exact. **On connaît désormais la longueur exacte d'un record** → le walk record→record est débloqué. `BUILD+VET OK`.

Composants portés (tous dans `components_biped_ability.go`, dispatch dans `traverse.go`) :
- **i47** `biped-desired-grenade-set` (FUN_140c6a638) = R(6)+R(3), 9 bits.
- **i48** `biped-desired-ability-set` (FUN_1406d0ff0) = R(3)+consumeGate0R(6).
- **i49** `biped-control-context` (FUN_14107166c) = R(2|4 selon PositionFullPrecision)+R(1).
- **i54** `biped-mobility-action` (FUN_1408f0264) = R(1)+R(1)+[flag1:consume1408f0ac4]+[**état ctx+0x9d**: gros bloc si mid-action, sinon 0]. Cas commun modélisé.
- **i56** `biped-spartan-ability-energy` (FUN_140fc1410) = R(3) plat.
- **i59** `biped-spartan-ability-non-predicted-state` (FUN_142f02994) = R(2) tag +[**tag==3**: gros bloc FUN_142f25e90 NON porté]+[rsp>1: R(3)]. Cas commun tag≠3.
- **i61** `simulation-state-playback` (FUN_142ed6d20) = R(1) gate; si 1: R(4)+R(32)+R(32).
- **i62** `biped-slide` (FUN_142f02978) = R(1) gate; si 1: FUN_14076d528(R1g0→R19+R10)+R(8)+[rsp≥1:R(8)]+R(8).

**CAVEATS value/état-gated (à porter quand un AUTRE record les déclenche)** : i54 (ctx+0x9d mid-mobility), i59 (tag==3 FUN_142f25e90 = positions/quats/dequants). Ces branches lourdes sont absentes du record Hydra mais surviendront sur d'autres bipeds → désync ponctuel pointant le bloc à porter.

### PROCHAIN PAS RÉVISÉ (le walk, maintenant débloqué)
1. **Calibrer l'en-tête de boucle keyframe** : la boucle (prefix-code type + id table-driven, cf. `frame_records.go`) précède chaque record. Le `defaultBits`(=380 ici) et `recordStateParam`(=2 ici) sont des params RUNTIME par record/archétype — déterminer s'ils sont constants par archétype (biped #35 = 380/2 ?) ou variables. Sweep `FrameConfig{IDLowBits,HasExtraFields}` pour que la boucle parte du 1er record et enchaîne (chaque NEW → `BindFull` + traverse + avance via EndBit).
2. **Walk complet** → World (slot→typeIndex) + arme de chaque biped (WST). Porter les caveats (i54/i59) + les archétypes non-biped au fil des désyncs.
3. Mapper biped→joueur (i9 'obje' = `DecodeEntityRecordQ`, déjà porté) ; rejouer deltas type-0 ; croiser kill-feed → `killer_victim_pairs.weapon_id`.

### VERDICT walk (workflow keyframe-walk-header, 2026-06-06) — corrections décisives
- **La "default-state" (~380 bits) N'EST PAS un `Skip` de largeur fixe** : c'est un **DESER unique `vtable[0x60](descripteur_archétype, …, bitreader, …)`** (vérifié dans `FUN_141f86704`) qui consomme un **nombre de bits VARIABLE / auto-délimité**. Donc `defaultBits=380` est **spécifique au record Hydra** et NE généralise PAS — le `Skip(defaultStateBits)` de `TraverseEntity` est une calibration ad-hoc, pas le vrai chemin. Pour walker il faut **porter `vtable[0x60]`**.
- **Stride d'archétype = 0xC8 (200)**, PAS 0xa0 (`FUN_1406cb5f0 *200`). Le commentaire de `world.go` (« stride 0xa0 ») est à corriger.
- **`recordStateParam` (param_4) = per-composant STATIQUE** (= `vtable[0]` du descripteur de composant, name-getter RTTI pur, p.ex. WST `FUN_141172060` = `lea rax,["weapon-state-type-info"]; ret`) → **rejouable à l'identique**, pas runtime. (La valeur scalaire 2 non prouvée au binaire — décalage prototype Ghidra — mais provenance statique tranchée.)
- **Walk FAISABLE.** Seul vrai blocage = porter `vtable[0x60]` (deser default-state), piloté par les tables de largeur-par-channel `DAT_1451f98xx` + lecteurs `FUN_1406d3140`/`FUN_1406d310c`.

**Séquence de walk cible** (par record) : `R(6) typeIndex` → **deser default-state `vtable[0x60]`** (avance le reader, largeur variable) → `mask` (`FUN_1406d7610` : gate R(1) ; si 1 → `R(3) count` + count×`R(6)` ; sinon R(64)) → composants auto-délimités. **Validation** : enchaîner depuis l'endBit Hydra (195892) → le record suivant doit donner un `R(6) typeIndex` propre (verdict suggère =16 change-scene-component).

**2 pistes lancées (workflow keyframe-loadout-and-defstate)** : **A** = loadout des 8 joueurs (mapping biped→joueur via i9 'obje', sans le deser default-state, par ancrage brute-force) ; **B** = résoudre + décompiler + porter `vtable[0x60]` biped #35 (valider 380 bits sur le record Hydra).

### RÉSULTATS (workflow keyframe-loadout-and-defstate, 2026-06-06)

**Track A — LOADOUTS DES 8 JOUEURS = CERTAINS (bit-exacts), en ordre de record keyframe** :
| # | primaire | secondaire | bits (prim/sec) |
|---|---|---|---|
| 0 | MLRS-2 Hydra | Rushdown Hammer | 195322/195518 |
| 1 | Shock Rifle | Rushdown Hammer | 198139/198334 |
| 2 | M41 SPNKr | Heatwave | 200932/201127 |
| 3 | Mangler | Ravager | 203735/203930 |
| 4 | MA40 AR | Disruptor | 206528/206724 |
| 5 | Mangler | Skewer | 209338/209533 |
| 6 | Cindershot | Shock Rifle | 212126/212322 |
| 7 | CQS48 Bulldog | M41 SPNKr | 214921/215117 |
Chaque WST = `gate=1`@litBit-1, `handle`=high32 (famille), `variant`=low32 (suffixe `0x42c9679f`, sauf Hammer `0xd8d07ca1` = équipement mêlée légitime). Sonde : `cmd/tmp_loadout` (mode `anat`/`final`). **Mapping record→joueur NON résolu** : aucun xuid dans chunk_02 ; le 'obje' i9 (LocalID/VariantName) n'encode PAS le joueur (incohérent inter-records = handle d'objet MP runtime, pas clé réseau). Roster DB (réf) : team0 LORD PEINX13(pi3) whiteknight2519(pi0) JGtm(pi2) VitaminA1688(pi7) ; team1 aldusbroncus(pi6) JAVIERLOLITO540(pi1) IKE ILYA(pi4) Akatsuki fire17(pi5). Ordre roster created_at : pi3,pi4,pi6,pi1,pi0,pi5,pi2,pi7. **Hypothèse à valider** : record#i = player-slot#i.

**Track B — `vtable[0x60]` = le deser default-state, CONFIRMÉ en assembleur** (`FUN_141f86704`@141f868c2 : `CALL [RAX+0x60](descripteur, size1, dst, bitreader, 1)`). Dispatcher `FUN_1406cd128` lit channel-type 2 bits ; type1→NEW(FUN_141f86704), type3→delta(FUN_141f86b58, memcpy default-state depuis entry+0xb8 taille entry+0xc0). **MAIS l'adresse concrète du deser pour typeIndex=35 N'EST PAS résoluble statiquement** : la table `param_1+0x18` est construite au runtime (relocations non tracées par Ghidra ; pas de globale). La vtable de DÉFINITION du tag "biped" (`143737138`, renvoie 35) n'est PAS la vtable du descripteur de réplication. **Largeurs default-state VARIABLES confirmées** (mesuré : 155/380/382/224/157/44/6 selon le record ; 380 = spécifique Hydra). Port draft `cmd/tmp_defstateport` valide que Skip(380) consomme exactement la default-state Hydra (gate@194512), mais ne généralise pas. **CHEMIN DE DÉBLOCAGE = DEBUGGER LIVE** : breakpoint à `141f868c2`, lire `R14`(descripteur)→`[R14]`(vtable)→`[vtable+0x60]` quand `typeIndex==35` ; alt : breakpoint au constructeur du contexte (vtable `1436a8800`, `FUN_1406cd128` à slot +0x20) et lire `this+0x18`. ⚠️ ESCALADE OPÉRATIONNELLE : nécessite le jeu lancé + Ghidra debugger (bridge :8099 absent actuellement).

### ✅✅✅ MUR FRANCHI (debugger live Cheat Engine, 2026-06-06)
Capture dynamique réussie (offline, sans AC, compte dédié). Script CE Lua : breakpoint **int3** (software, tous threads — le hardware BP ne se déclenchait pas, mauvais thread) à `HaloInfinite.exe+1F867F8` (= `141f867f8`), `debugger_onBreakpoint` filtrant `RCX==35`, lecture `[[R14]+0x60]`. Résultat (8 hits biped, valeur stable) :
- **Deser default-state biped #35 = `HaloInfinite.exe+0xF44C38` = `FUN_140F44C38`** (vtable[0x60]).
- vtable biped (runtime ce session) = 0x7FF656BA7178.
- Confirmé : ~282 records dans le keyframe (typeIndex vus : 2,4,5,6,9,13,14,15,17,18,19,21,22,25,26,27,29,34,**35**,37,38,41,42,43,45,47) ; biped = bien typeIndex 35 au runtime.

**`FUN_140F44C38` décompilé** (statique). Lit une séquence de champs gouvernée par une version `uVar10` (= R(1) gate ? R(8) : 13) : `player-representation-name` R(32) gaté, `FUN_1407f2058` R(1)[+R(5)], gates R(1)[+R(6)], `FUN_14080d69c` R(1)[+R(32)], branche quat `FUN_14076e494` R(16)+`FUN_1407f2058`, `FUN_14076dc04` R(19), flags selon uVar10 (>5, >=0xc). **`vtable[0x88]` = 0 bit** (le bitreader RSI n'est PAS passé — vérifié au désassemblage de FUN_141f86704 @868e2). **ÉNIGME à trancher empiriquement** : le comptage naïf de FUN_140F44C38 donne ~164 bits max alors que le `Skip(380)` calibré aligne le record Hydra → soit un sous-lecteur lit plus (FUN_14076dc04/quat largeurs à revérifier), soit un résidu. À MESURER : porter FUN_140F44C38, l'exécuter sur le record Hydra depuis bit194132, voir s'il atteint le gate@194512 (=380 bits). PROCHAIN : port + mesure + intégration dans TraverseEntity (remplacer `Skip(defaultStateBits)` par le vrai deser quand typeIndex==35) → puis walk record→record.

### ⚠️ RÉSULTAT DU PORT (2026-06-06) — FUN_140F44C38 = 32 bits seulement (en-tête identité, PAS toute la default-state)
Port bit-exact livré (`filmdec/default_state.go` : `consumeBipedDefaultState`). Grammaire confirmée (gouvernée par `uVar10 = R(1)?R(8):13` ; sur Hydra uVar10=13) : `R(1)g0`, `[g0:R(8)]`, `R(1)gRep`, `[gRep:R(32) player-representation-name (FUN_14080dec4)]`, `[uVar10>10: FUN_1407f2058 R(1)[+R(5)]]`, `R(1)gC6`, `[gC6:R(6)]`, `R(1)`, `FUN_14080d69c R(1)[+R(32)]`, `[état-dst: quat FUN_14076e494 (≈1-2 bits, le 0x10=offset table PAS largeur) + FUN_1407f2058]`, `FUN_14076dc04 R(19)`, `[uVar10>5: R(1)]`, `[uVar10>=12: FUN_14080d69c R(1)[+R(32)]]`. Fonctions sur le buffer dst (param_3) = 0 bit.
- **MESURE : FUN_140F44C38 consomme 32 bits sur Hydra** (arrêt @194164), PAS 380. C'est l'en-tête d'IDENTITÉ de l'entité, pas l'état complet.
- **RÉSIDU = 348 bits [194164, 194512]** : bloc structuré (motif `count×R(n)` = boucle) entre l'identité et le mask (dense `0x6940e217d79257a0` popcount=29 @194513). PAS dans vtable[0x88] (0 bit, RSI non passé @868e2), PAS dans le tail de FUN_141f86704 (≤2×R(1)+memset). **Hypothèse : composants TOUJOURS-RÉPLIQUÉS lus avant le mask** (probablement par `FUN_14076cb60` en préambule, ou une boucle dans le contexte de FUN_141f86704). Grammaire à identifier = LA frontière restante du walk.
- Intégration : `TraverseEntity` route typeIndex==35 → `consumeBipedDefaultState` (32 bits réels) + résidu sauté explicitement jusqu'à defaultStateBits. Record Hydra : `DesyncAt=-1`, WST i45=Hydra exact. `BUILD/VET/TEST OK`. Fichiers : `default_state.go` (créé), `traverse.go` + `probe_export.go` (modifiés).
- **PROCHAIN** : cracker les 348 bits (décompile FULL de `FUN_141f86704` + `FUN_14076cb60` ; tester l'hypothèse composants-toujours-répliqués i0/i1/i3/i4… déjà portés). Une fois le résidu lu bit-exact (variable par record) → `Skip` éliminé → **walk record→record généralisable**.

### RÉSULTAT crack-résidu (2026-06-06) — 120/380 bits, le reste = mur PrecisionDescriptor
- **Hypothèse "composants always-on" INFIRMÉE** : `FUN_14076cb60` lit le **mask EN PREMIER** (`FUN_1406d7610`), puis itère ; rien n'est lu entre la default-state et le mask. Le `consumeMask` Go est bit-exact (mask Hydra = `0x6940e217d79257a0` popcount=29 @194514).
- **Les bits manquants sont DANS `FUN_140F44C38`** : le port omettait **`FUN_14080cfe8`** (bloc object-multiplayer-properties du default-state ; son param_2 EST le bitreader, prouvé asm @140f44d0e). Porté = `consumeMultiplayerPropertiesBlock` (`R(9)+R(32)+gates+R(5)+R(3)count+boucle(R(5)+opt32)+tail`). Couverture **32 → 120 bits**.
- **RÉSIDU = 260 bits non portés**, prouvés venir de `FUN_140F44C38` (seul à recevoir le bitreader). Ce sont des **chemins-feuille à largeurs chargées au map-load** (`DAT_1445cc9e0` largeurs d'axe, `DAT_144632be0`, `DAT_145121140`) — **lues 0 en statique = LE MÊME MUR que `PrecisionDescriptor`/`TraversalPrecision`/i0** (déjà mesuré au Cheat Engine : axes 6/6/6, index 1). Sur le record : bloc régulier 96 bits (`0x3FC,0`×4 @194256 = vec3/quat quantifié) + mots denses. **À résoudre en sourçant ces largeurs (comme i0) puis en portant le bloc vec3/quat.**
- Intégration : `consumeBipedDefaultState` câble identité(32)+mpp(88) = 120 bits réels + résidu(260) sauté. Record Hydra : `DesyncAt=-1`, WST=Hydra exact. `BUILD/VET/TEST OK`. Fichiers : `default_state.go` (consumeMultiplayerPropertiesBlock+helpers), `traverse.go`, `probe_export.go` (+ sondes `cmd/tmp_residual`, `cmd/tmp_mpp`). NON committé.

### 📋 SCOPE RESTANT (bilan lucide pour décider la suite)
Le **vrai but** (arme par kill → `killer_victim_pairs.weapon_id`) exige, dans l'ordre :
1. **Default-state biped complet** (260 bits restants = bloc quantifié à largeurs runtime ; largeurs position connues 6/6/6, à appliquer + porter le vec3/quat). Tractable.
2. **Walk keyframe complet** = décoder TOUS les ~25 archétypes présents (typeIndex vus : 2,4,5,6,9,13,14,15,17,18,19,21,22,25,26,27,29,34,35,37,38,41,42,43,45,47). CHAQUE archétype a son propre deser default-state (vtable[0x60], adresse runtime — la diag debugger a loggé le `R14`/descripteur de chacun, mais pas leur deser), souvent avec le même mur de largeurs quantifiées. **Gros chantier multi-archétypes.**
3. **World** (slot→typeIndex) construit par le walk. ALTERNATIVE : dumper la table d'entités via le debugger après décodage keyframe (raccourci).
4. **Rejeu des deltas type-0** (mask+composants, sans default-state — plus simple, mais quantif runtime sur i0 etc.).
5. **Mapping record→joueur** : NON résolu (pas d'ancre xuid au keyframe ; 'obje' i9 mask-gated n'encode pas le joueur — mais le bloc mpp du DEFAULT-STATE, `FUN_14080cfe8`, lit un R(32) variant + R(9) : à inspecter, pourrait porter l'identité). 
6. **Croisement kill-feed** (chunk_27 : tueur+victime+time).

### ✅✅✅✅ WORLD CAPTURÉ (debugger live, raccourci validé — 2026-06-06)
Le dump de la table d'entités live échouait (numérotation moteur ≠ film, table pré-allouée). **Solution = 2 breakpoints appairés sur le chemin de décodage FILM** (le seul qui donne les typeIndex du film) : à l'entrée de `FUN_141f86704` (`RDX = id`, param_2), juste après le R(6) (`RCX = typeIndex`). Script CE Lua `debugger_onBreakpoint` filtrant `RIP`, pairant `id→typeIndex` jusqu'à 250 entités. **RÉSULTAT** : World complet (250 entités) sauvé dans `data/cache/film_chunks/000d5950/world_dump.txt`. **Les 8 bipeds (typeIndex 35) = slots 512..519** (bloc contigu = les 8 joueurs). Autres types présents : 6 (×48), 5, 17, 14, 38 (×beaucoup = armes/objets monde), 9/47 alternés, 42, 43, etc. **Plus aucune session debugger nécessaire** (on a FUN_140F44C38 + le World).

### PROCHAIN PAS = REJEU DES DELTAS (débloqué)
Avec le World (slot→typeIndex), on saute tout le décodage des default-states du keyframe. Les **deltas type-0** = mask + composants (PAS de default-state → pas le mur des 380 bits). Plan : (1) init `World.BindFull(slot,typeIndex)` depuis le dump ; (2) extraire les paquets type-0 d'un chunk gameplay ; (3) `DecodeFrameRecords(br, world, cfg)` — calibrer `FrameConfig.IDLowBits` pour que les slots décodés matchent le World (512-519 = bipeds) ; (4) pour les slots biped, suivre le composant `weapon-state-type-info` (i43-46) dans les deltas → arme dans le temps ; (5) croiser kill-feed (chunk_27 : tueur pi + time) → arme au tick du kill → `killer_victim_pairs.weapon_id`. Mapping joueur : slots 512-519 = player-slots 0-7 (à confirmer via roster ordre / player-representation-name).

### ✅ REJEU DES DELTAS — CALIBRATION TROUVÉE + traversée biped OK ; mais l'arme N'EST PAS dans les deltas de mouvement (2026-06-06, sonde `cmd/tmp_worldreplay`)
**CALIBRATION `FrameConfig` (reproduite sur 1199 paquets) : `HasExtraFields=false, IDLowBits=11, recordStateParam=2`.** Trouvée par la métrique « slot du 1er record ∈ World » (le scoring « cleanEnd » du keyframe était trompeur — il favorisait un combo qui s'arrête vite). Avec idLowBits=11 le 1er record de chaque frame (un delta, bit0=1) tombe sur le **biped slot 519** dans **1080/1199 paquets** (1097/1199 dans le World). Tout autre idLowBits rate les bipeds (8→slot64, 9→slot129, 13+→slots ≥2078 hors-World). Le départ reste **bit 0** (marker `a0 7b 42` inclus = encodage naturel `delta + id 519`).

**FIX décodeur (appliqué, `traverse.go`)** : le case `consumeByName` matchait `object-angular-velocity-dynamic-precision-component`, mais le biped #35 nomme i3 **`object-angular-velocity-component`** (sans `-dynamic-precision-`). Alias ajouté (même deser). Avant : désync à i3. Après : la traversée delta biped enchaîne **i0→i62**.

**JALON — la traversée delta biped est bit-exacte jusqu'à l'arme** : sur 58 deltas biped (slot 519), **56 désync PILE à i63 `biped-action-component`** (le DERNIER composant, non porté) ; 2 seulement plus tôt. Désyncs concentrés sur le dernier composant = preuve que les largeurs i0→i62 (incl. les chemins delta de i0 position/i1/i3, i9 'obje', les weapon-state-ammo/rounds/overheated, les WST i43-46, les biped-*) sont **correctes en mode delta**. `i63` a une largeur **variable** non fermable sans Ghidra (hook `SetUnportedStubWidth(name,w)` ajouté pour la sonder ; le brute-force par chaînage ne donne pas de pic net → largeur data-dependent).

**⚠️ RÉSULTAT CLÉ — l'arme n'est PAS ré-émise dans les deltas de mouvement** : sur **346 WST i43-46** rencontrés dans les deltas biped (148 avec gate interne=1) et sur **414 WST i43/i45 gate=1** scannés sur le chunk entier, **0** ne contient un handle/variant d'arme du catalogue — à **AUCUN offset interne** [start, +260]. Or au keyframe le **même deser** lit `handle@+1 = 0x767db96d = MLRS-2 Hydra` exact (régression vérifiée, `cmd/tmp_i48trace` toujours `DesyncAt=-1` endBit=195892). Donc le deser WST est bon ; en mode **delta**, un WST `gate=1` signale un changement d'**état** du slot (ammo / rounds-inventory / overheat / cooldown / chambrage) mais **ne ré-encode PAS le variant-name** (= l'identité de l'arme, donnée quasi-statique en RAM, transmise au keyframe + au swap uniquement). C'est le comportement ECS attendu (delta = ce qui change frame-à-frame).

**IMPLICATION pour l'attribution arme-dans-le-temps** : NE PAS chercher le variant-name dans les deltas de mouvement. L'arme courante d'un biped = son **loadout keyframe** (les 8 connus bit-exact : Hydra, Shock Rifle, M41 SPNKr, Mangler, MA40 AR, Mangler, Cindershot, Bulldog en primaire) **propagé dans le temps**, mis à jour SEULEMENT aux **records de swap/pickup** (un WST portant réellement le variant — à isoler : ce sont des deltas/NEW spécifiques RARES, pas les 414 deltas d'état observés ; piste = filtrer les WST dont `handle@+1` matche le catalogue, ou les records `NEW` d'entité-arme au pickup).

**RESTE à porter pour walker le record loop complet** (et atteindre les 8 bipeds 512-519, pas seulement 519) : (a) **i63 `biped-action-component`** (largeur variable — Ghidra requis) ; (b) les **archétypes non-biped** rencontrés dans le loop (typeIdx 5 `player-engine-loadout-index`, 6, 12 `managed-navpoint-flags`, 38, 0 `game-engine-team-mapping`…) — chacun a ses propres desers de composants. Tant qu'ils désync, le loop s'arrête au 1er biped (519) de chaque frame.

**Sondes** : `cmd/tmp_worldreplay` (parse world_dump → World ; sweep calibration ; dump/anatomie delta biped ; histogramme désyncs ; chasse WST gate=1 + scan offset arme ; brute-force i63 ; bipeds distincts). `cmd/tmp_archdump` (noms ordonnés des 64 composants du biped #35). Code lib modifié : `traverse.go` (alias i3 + `SetUnportedStubWidth`/`unportedStubWidth`). BUILD/VET/TEST filmdec OK ; keyframe Hydra non régressé.

## ⛔ CORRECTION + RÉ-ANCRAGE (2026-06-06, user) — lire AVANT de continuer

Deux conclusions ci-dessus sont **FAUSSES et RÉTRACTÉES** (dérives récurrentes, cf mémoire `project_kill_feed_frame_decoder`) :

1. **« l'arme n'est PAS dans les deltas » = FAUX.** Un autre agent a trouvé **247 littéraux d'armes complets** (high32|low32 catalogués, 100% précédés d'un gate WST) dans le flux gameplay type-0, avec une arme qui **CHANGE dans le temps** (Cindershot→M41 SPNKr→Skewer→S7 Sniper). Le held-weapon de tous les joueurs **EST** dans les deltas (events de swap/pickup, sporadiques 0.01-0.04/paquet). Le "0 arme" précédent = on ne lisait que le biped 519 + décode WST delta config-gated. **Ne PAS reconclure que l'arme n'est pas là.**
2. **« limite de données per-client / seulement joueurs proches » = FAUX.** Le film réplique **TOUS les joueurs**. Le "1 biped/paquet" = **trou de notre décodeur**, jamais une fatalité de la donnée.

**LE BUT (ancré)** : RE du **KILL FEED** = tracker d'événements — KILL (tueur · arme/**méthode** · victime), SUICIDE (joueur · méthode de mort), DÉPART/ARRIVÉE. PAS "l'arme au kill" isolée. Réf : `PLAN_FILM_KILLFEED_V3.md`. Arme **reconstruite** comme le moteur : hitscan → held du tueur **au tick** (état ECS reconstruit) ; projectile → entité projectile létale @position-de-mort. Events = chunk_27 (type-3, killer/victim/time/méthode).

**i63 `biped-action-component` PORTÉ (cas commun)** : `FUN_142f26a20` = `FUN_142f21b10`(R(32)×3=96b début) + `R(4)` count1 + count1×[R(7)+dispatch value-gated `FUN_141fd4814`(R(5)tag) NON porté] + count2×[R(1)+optR(2)] (count2 = popcount RAM, ~toujours 0) + `FUN_142f21b10`(96b fin). **Cas commun (count1=0,count2=0) = 196 bits.** Débloque 22% des deltas biped (519). `consumeBipedAction` dans `components_biped_ability.go`, dispatch `case "biped-action-component"`. count1>0 (dispatch value-gated config-gated) reste non porté → désync propre. BUILD/VET/TEST OK.

**ÉTAT vs PLAN** : L1-L3 ✅ (décodeur + World + IDLowBits=11). **L4 = timeline held-weapon par joueur** (RESTE) : finir le décodeur pour atteindre TOUS les bipeds (porter i50/i52 + i63 count1>0 + archétypes non-biped du loop : typeIdx 0/5/6/12/14/38…) + WST delta bit-exact (largeur config-gated, mesurable comme i0=6/6/6). Le held-weapon est là (247 littéraux), il faut juste finir le décodeur pour l'attribuer slot+tick. **L5** = résolution kill (held@tick hitscan / projectile@death-pos) + croisement events chunk_27. NON committé jusqu'ici.

---

## CAMPAGNE PORTAGE COMPOSANTS (workflow ultracode, 2026-06-30) — but POSITIONS joueurs

> Objectif user : reconstruire les **déplacements 2D des 8 joueurs** sur la carte. Reprise du frame-decoder
> (cf. ligne 310 : « 1 biped/frame = TROU du décodeur, le film réplique TOUS les joueurs »). Branche
> `feat/filmdec-continuation`. **42 composants portés cette session** (6 player-engine à la main + 2 lots
> workflow de 19 et 17). Commits : `7bfb5e221`, `dbbbaf7c4`, `83f1e9703`.

### MÉTHODE — RE parallèle via workflow + Ghidra MCP (efficace, reproductible)
`Workflow` (ultracode) lance **N agents en parallèle, 1 par composant non porté**. Chaque agent, via le MCP
Ghidra (déjà connecté, HaloInfinite.exe image_base 0x140000000), suit la chaîne canonique et retourne une
grammaire de bits structurée (+ snippet Go). **Recette de résolution d'un deser de composant** :
1. `search_strings "<nom-component>"` → adresse S de la chaîne.
2. `get_xrefs_to S` → name-getter N (stub `lea rax,[rip+...]; ret` → S).
3. `get_xrefs_to N` → V = **vtable[0]** du descripteur.
4. `read_memory(V + 0x28, 8)` little-endian → adresse D du **deser** (le dispatch `vtable+0x28`, cf §L3).
5. `decompile_function D` ; suivre les helpers récursivement, sommer les bits.
**Helpers de lecture (bit-cost) confirmés** (param BitReader : +0x2c curseur, +0x30 registre 64b MSB,
+0x38 bits, +0x40 ptr) : `FUN_1406cf008`=R(1) ; `FUN_1406d84b4(...,W,...)`=R(W) (W = 1er arg pile [RSP+0x20]) ;
`FUN_1407f0354`=R(5) ; `FUN_140ebf854`=R(5) ; `FUN_14080dec4`/`FUN_14080d6f0`=R(32) ; `FUN_1406d676c(...,0x60)`=R(96) ;
`FUN_14076d528(...,mag,scale)`=R(1) present[+R(mag)+R(scale)]. **Bloc inline if/else (boucle d'octets +
`*(br+0x2c)+=N`) = R(N).** Pièges : `FUN_14076f91c`/`FUN_1406cb0cc`/`FUN_1405d5dbc`/lookups = **0 bit**.
**Anti-rate-limit** : 20 agents simultanés saturent le serveur → **lots de 4** (`parallel` par chunk de 4).
Script réutilisable : `…/workflows/scripts/filmdec-port-component-desers-wf_*.js` (éditer la liste `COMPONENTS`
+ relancer via `{scriptPath}`).

### HIGH vs LOW — la frontière critique (largeurs runtime)
- **HIGH (largeur fixe, immédiat)** : portés direct. Ex : player-engine-loadout 8×R(8), player-fade-properties
  6×R(12), game-engine-round-timer R(16+16+5), music-state (R1+R32+R33+R33+gates), managed-player-color-override
  8×R(8), statborg-entry-index R(32+8), physics-state R(32)[+R32], biped-emp-timer R(8)…
- **LOW (largeur DATA-DEPENDENT runtime)** : un sous-bloc **vec3 quantifié** (`FUN_14076e524`→`FUN_140cc5128`)
  lit 3 axes de largeurs venant du **precision descriptor runtime** (tables `DAT_1445cc9e0`/`DAT_1445ccbe0`
  + index `DAT_144632be0`, **toutes à 0 en statique** = remplies au runtime). Formule connue (mémoire
  `reference_filmdec_quant_width_formula`) : `step(L)=2^(16-L)/120`, `L=0→6/6/6`. **MAIS le niveau L diffère
  par composant** → un port à 6/6/6 risque un bit-error → corruption SILENCIEUSE en aval. **RÈGLE D'OR** :
  pour un LOW, porter le **préfixe sûr** (gates R(1)/champs fixes) et **`return ...,false` (desync PROPRE)**
  dès la branche data-dependent ; JAMAIS de bit-guess. Le gate de défaut (`precHigh==1` → vecteur défaut,
  0 bit) avance le cas dominant pour certains (player-primary-respawn-object/desired-location : gate==0=absent
  dominant → avance ; gate==1 → desync). Vec3 purs (tacmap-poiicon/offset, flock-destination, biped-posture-
  physics polymorphe) = **SKIP** (pas de préfixe sûr).

### VÉRIFICATION (anti-régression bit-exact)
`cmd/tmp_desynccomp` (World=`world_dump.txt`, 250 entités, bipeds 512-519) décode tous les type-0, et pour le
**1er record qui désync par frame** loggue `(typeIdx, i<N> composant non-porté)`. Après chaque lot : un
composant porté **disparaît** des bloqueurs et la boucle **avance** au composant suivant (i0→i1→…). Un port
FAUX se trahirait par un **nouveau bloqueur aberrant en aval** (corruption) ou une **hausse** du desync.
Observé : desync 44%→42%, bloqueurs avancés proprement (player i1→i17, game-engine→grace-period, etc.).

### ⛔ TROUVAILLE STRATÉGIQUE — le grind composants N'ATTEINT PAS les positions seul
- **Le dominant = slots dynamiques NON-LIÉS** (typeIdx=0, ~10900 desyncs ; 1714 slots distincts ; entités
  transitoires projectiles/effets absentes des dumps CE). Au **record #1** (juste après le biped POV 519 =
  record #0) tombe souvent un DELTA sur slot non-lié → désync → les bipeds 512-518 (records suivants) jamais atteints.
- **Les composants ne sont PAS le goulot** : les porter avance les chaînes par-archétype mais ne touche pas
  le cascade des non-liés. Mesuré : desync 44%→42% seulement ; `tmp_resync` (World persistant + calibration
  default-state + 36 composants) = 43% frames complètes, bipeds 512-518 toujours ~0 i0.
- **Pourquoi 512-518 ~0 i0 même décodés** : ils émettent surtout du **keep-baseline** (position tenue, cf
  §10.4 HANDOFF_MAP_GEOMETRY) dans les frames SIMPLES ; leurs frames de **mouvement** (i0 deltas denses) sont
  les frames COMPLEXES qui désync sur les non-liés. ⇒ atteindre leur mouvement = **décoder les frames complexes**
  = **lier TOUS les slots** = besoin d'un **World COMPLET**.
- **Le cascade des non-liés se brise uniquement par un World complet** : un slot X non-lié l'est car son
  record NEW (au spawn) était dans une frame qui a désync avant de l'atteindre (récursif). Pas de point d'entrée
  propre sans World complet.

### VOIE POUR LES POSITIONS = World complet (2 options)
- **(A) Walk keyframe OFFLINE** → reconstruit le World (le keyframe a 1 NEW/entité). **BLOQUÉ AU BOOTSTRAP** :
  `cmd/tmp_kfwalk` (header type+id + calibration default-state par brute-force d + lookahead) rend **0 record**
  depuis bit 0 — le keyframe ne démarre PAS proprement au header type+id à bit 0 (structure dense distincte,
  records bipeds à ~194126, préambule devant ; cf ligne 29 « DecodeFrameRecords ne s'applique pas au keyframe »).
  + exige TOUS les composants (dont vec3 runtime). 2 sous-problèmes durs.
- **(B) Capture debugger** (méthode qui a produit `world_dump.txt`/250 et `world_dump_full.txt`/2229, cf
  ligne 285) : breakpoint apparié sur le chemin de décodage FILM, logge `id→typeIndex` au fil. Une capture
  **exhaustive** (tout le film, pas un snapshot) lie les transitoires. **Rapide mais jeu requis.**

### OUTILS de cette campagne
`cmd/tmp_desynccomp` (1er bloqueur par frame — le pilote du grind), `cmd/tmp_archcomps` (dump Components[]
par archétype), `cmd/tmp_kfwalk` (walk keyframe calibré — bootstrap bloqué), `cmd/tmp_resync` (World persistant
+ calibration NEW), `cmd/tmp_newdiag` (histogramme NEW/desync). Probes untracked. Dispatch des 42 composants
dans `internal/analysis/filmdec/traverse.go` (cases groupées « lot 1 / lot 2 », commentées `// ti=<arch> i<idx> (FUN_...)`).

### PROCHAINS LOTS (si grind continué — `tmp_desynccomp` top, hors vec3 runtime)
player-engine i15-i26 (last-betrayer, vehicle-entrance-ban, control-aiming, active-in-game, malleable-props,
representation, aim-assist, power-frame-points, desired-frame-config, supply-lines, allowed-to-quit) ;
managed-player i5-i9 (active-mission-name, show-active-mission, campaign-progress, current-season, custom-input) ;
game-engine i7-i26 (grace-period, round-condition-flags, alliance, soft-ceilings, campaign-timer, matchflow,
forge-engine ×8…) ; flock i12-i17 (forced-respawn fait, relevancy fait ; remembered-danger, position, current-dest) ;
tacmap i3-i16 (mapscale, settingstag, etc.). **Les vec3 runtime (poi/destination/posture) restent SKIP jusqu'à
résolution du precision descriptor (voie B, ou capture des largeurs comme i0=6/6/6 validée au gradient `tmp_cleanframe`).**

---

## PISTES BANCARISÉES (2026-06-30, leads user + investigation Bond/précision)

### ⭐⭐⭐ PISTE 1 — ✅ VALIDÉE + GÉNÉRALISÉE (2026-06-30) — précision quantif = `flags` registre + formule
**RÉSOLU.** Commits `c61cb8054` (validation) + `e803d32b5` (data-driven). L'intuition user (« la précision est
dans le film, auto-descriptive ») est CONFIRMÉE : **la largeur de quantif vec3 = `6 + flags` (flags = u32 @ slot+4
du registre = LEVEL de précision PAR-SLOT)**. 100% offline, sans Cheat Engine.
- **Validé empiriquement** (`tmp_desynccomp`, bloqueur avance proprement, 0 aberration aval, desync 12326→12221) :
  crew-order (arch14 i0, L0=6/6/6), player-desired-respawn-location (arch5 i12, L1=7/7/7+R19), flock-destination
  (arch21 i2-i11, **L1=7 ET L2=8 selon le SLOT** → data-driven indispensable), tacmap-poiiconoffset (arch30 i1),
  tacmap-poiicon (arch30 i0, vec3 au milieu d'un bloc).
- **Implémenté** : `Registry.Archetype.Flags[]` (parse u32 @ slot+4), `Archetype.Level(i)`, `consumeByName(...,level)`,
  helper `consumeQuantVec3(br, axisW=6+level, indexW=1)` (= FUN_14076e524 : precHigh gate + index gate + 3×axis, sans emit).
- **Reste vec3** : nav-cutscene-flag (arch15 i0, vec3 dans bloc), biped-posture-physics (arch35 i55, union polymorphe).
  Composants à **handle var-width** (respawn-seat i7, primary-respawn-object i10, crew-marked-objects i1) = table de
  précision DIFFÉRENTE (DAT_1451f98d0 handle, pas axis) → sous-problème distinct (le flags ne donne pas leur largeur).
- **MAIS** : ne débloque PAS encore les positions seul — le dominant reste le **cascade des slots non-liés** (11088),
  qui exige le World complet. La PISTE 1 est la BRIQUE PRÉCISION du walk keyframe ; reste le **bootstrap keyframe**
  (`tmp_kfwalk` toujours 0 record depuis bit 0, même avec vec3) OU une capture debugger.

#### (archive de la piste avant résolution)
Déclenchée par l'intuition user (« la structure/précision est dans le film, auto-descriptive », cf philosophie
dend bond-reader). **Investigation Ghidra + registre** :
- `FUN_140be9a14` peuple la précision depuis une **table de descripteurs** (`DAT_144976b60+0x7ac`, count@+0x7bc,
  stride 0xdc, **3 valeurs d'axe min/max @+0x44/+0x4c/+0x54** par descripteur). `DAT_144632be0`(IndexW)=1 si count>1.
- `FUN_140be9b88` calcule la largeur d'axe : **`width = min(26=0x1a, bitLen(ceil(axisRange / (2·step(level)))))`**
  — EXACTEMENT la formule mémoire `reference_filmdec_quant_width_formula`. `step(level)=FUN_140be9c78(level)`,
  `axisRange = max-min` du descripteur.
- **`flags` du registre (slot `[u32 kind][u32 flags][name]`, on lit le nom, on IGNORE kind/flags)** : kind=TOUJOURS 0 ;
  **flags ∈ {0,1,2,3,4} = très probablement le LEVEL de précision par composant**. i0 `object-position-dynamic-precision`
  a **flags=0 → level 0 → 6/6/6** = exactement la valeur capturée au CE (validé). Formule → L0=6,L1=7,L2=8,L3=9,L4=10.
- **MANQUE** : l'`axisRange` par composant (table descripteurs runtime, peuplée depuis la **map data**, PAS le
  registre — trailing 9600o du registre = ZÉRO). Pour les vec3 **position-monde**, range = boîte de réplication
  monde → **width = 6+flags**. Pour vec3 « locales » (offsets), range différent → largeur différente.
- **TEST À FAIRE** : porter une vec3 position-monde (ex `player-desired-respawn-location` arch5 i12, ou crew-order
  vec3 flags=0→6/6/6) avec `width=6+flags, IndexW=1`, structure FUN_14076e524 (precHigh gate + index gate + 3×axis),
  valider au `tmp_desynccomp` (bloqueur avance proprement = OK ; desync aberrant aval = range≠monde, revert).
  **Si OK = débloque tous les vec3 quantifiés → walk keyframe → World complet → positions.** Outil : `cmd/tmp_regschema`
  (dump kind/flags/name par archétype). NB : le bootstrap du walk keyframe reste à résoudre EN PLUS.

### ⭐ PISTE 2 — décodage direction (aim/vélocité) = cubemap `FUN_1406d8288` (lead user GitHub)
Un dev (thread Halo film) a décodé le **vecteur d'aim** : c'est un **cubemap 6 faces** (+X+Y+Z-X-Y-Z), `FACE_SIZE=0xAAA8000`,
30 bits, `face,coord = divmod(encoded, FACE_SIZE)` puis projection inverse cubemap → XYZ unit. Il bloque sur la **2e
coordonnée** (met 0 → erreur 45° aux coutures).
- **C'est la sémantique de `FUN_1406d8288`** (déjà décompilé : `iVar2=param_1/DAT_1447084d0[p]; reste; iVar3=reste/DAT_1447084d4[p]`)
  = divmod à 2 niveaux → face + 2 coords. **`FACE_SIZE 0xAAA8000` = `DAT_1447084d0[p]` pour la précision 30-bit.**
- **On a une longueur d'avance** : le **2e divmod (`DAT_1447084d4[p]`) = la 2e coordonnée** qu'il n'a pas trouvée →
  on peut décoder le vecteur COMPLET (sans erreur 45°) en portant `FUN_1406d8288` (tables d0/d4 + constantes
  DAT_143cd8374 etc.). Utilisé par : i1 vélocité-direction (R(19) packé), i17 player-control-aiming (R(19)), aim des
  fire events (30 bits). **Direct pour le but kill-feed/arme** (labelliser fire events) ; pour POSITIONS, complète la
  vélocité-direction (piste dead-reckoning, bémol vélocité sparse non-POV).
- **USAGE REPLAY 2D (user)** : l'aim = la **DIRECTION DE REGARD** du joueur → **flèche/cône de visée par marqueur
  joueur** sur la carte (position + où il regarde + events = replay riche). **`i17 player-control-aiming` est DÉJÀ
  porté (R(19), lot 3)** = la valeur d'aim est déjà consommée du flux ; reste à la DÉCODER via le cubemap
  `FUN_1406d8288` (face = `divmod(v, DAT_1447084d0[p])`, 2e coord via `DAT_1447084d4[p]`) → vecteur XYZ → projeter
  en cap 2D. Même limite de couverture que les positions (dense POV, sparse non-POV tant que le World incomplet).
