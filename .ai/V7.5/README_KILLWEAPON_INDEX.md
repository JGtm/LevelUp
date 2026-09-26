# ★ INDEX MAÎTRE — Arme par kill (À LIRE EN PREMIER, AVANT TOUTE ACTION)

> But unique du sujet : **arme par kill = la SOURCE DE DÉGÂT du coup fatal**, 100% offline (film seul,
> zéro CE/runtime au décodage), universel (tout match/mode/titre). ⛔ JAMAIS « held-weapon au tick »
> (rejeté ~10× par l'user). Branche : `feat/weapon-attribution-v3` (worktree). Répondre en français.
>
> **Ce fichier existe pour ne PLUS reperdre des heures à re-tester des pistes mortes.** Si tu reprends
> à froid : lis §0 → §1 → §2 (impasses) AVANT de coder ou lancer quoi que ce soit.

---

## §0bis. MAJ 2026-06-13 — ORACLE CE GROUND-TRUTH : le décodage dead-state offline est DÉSALIGNÉ (À LIRE)

**Session oracle CE (film joué en direct) = reframe décisif. LIRE AVANT de reprendre le décodeur.**

- **Pipeline CE réparé** : `filmdec_deadstate_capture.lua` avait `DREC=0x40` au dump alors que l'asm écrit en stride
  `0x60` → dump désaligné = grosse cause du "flaky" historique. Corrigé `DREC=0x60` ; `cmd/tmp_dscap` aligné.
- **Validateur ground-truth NOUVEAU — `cmd/tmp_cematch`** : apparie chaque mort résolue par le jeu (CE) à sa frame
  offline exacte (fingerprint buffer-base + bit `b2c`), puis vérifie « mon décodage atteint-il le dead-state à b2c ? ».
  Remplace les proxies trompeurs (compte Mort, `tmp_dsvalid` 2.2%, match flou kill-feed).
- **RÉSULTAT SANS APPEL** : **11/11 morts CE appariées** (fingerprint trouvé dans mes payloads → buffer CE == payload
  offline, appariement valide) mais **0/11 atteintes**. Les **~413 dead-states que `tmp_dsraw` trouvait sont du
  GARBAGE** (bits Mort lus à de mauvaises positions). Les vraies morts sont à `b2c` ~6000-9500 bits (≈ fin de frame),
  jamais atteint. Cross-check `tmp_dsmatch` (kill-feed) = match au HASARD (paire 1.9% ~ random 2%).
- **⛔ LE GRADIENT `tmp_cleanframe` N'EST PAS UN PROXY VALIDE** : une frame « propre » (atteint recEnd) peut être
  faux-mais-compensée. Valider DÉSORMAIS au ground-truth `tmp_cematch` (atteindre b2c), PAS au gradient.
- **VRAIE CAUSE du blocage (instrumentée `tmp_cematch VERBOSE`)** : la frame de mort désync à `endBit=14` sur
  `rec0 type=3(delta) slot=1038`. slot 1038 **N'EST PAS lié dans le World** → `decodeDelta` met `DesyncAt=0` +
  `TypeIndex=0` (zéro-value) → le diag affiche FAUSSEMENT « ti=0 game-engine-team-mapping ». **Le vrai problème =
  binding World incomplet** (cascading-desync : la frame qui devrait binder 1038 désync aussi). = fiabilité de
  décodage GLOBALE, **pas** un composant isolé. ⇒ le diagnostic largeurs-composants du 12 juin regardait trop en aval.
- **CE confirme aussi** : victime résolue base `0xE1500000` stride `0x10002`, toutes les morts résolues = **victime
  idx0 (joueur local, POV film)**, tueurs variables (1/5/4/6) ; **GID `06f8b7d3` CONSTANT** = pas l'arme (re-confirmé).
- **Workflow biped i0-i11** (12 agents) : **i5 shield-vitality** était `INFERRED-BY-TEMPLATE` (11 bits) et FAUX ; vrai
  deser `FUN_140d50cbc` = 29-55 bits (corrigé). **i3/i9 = faux leads** : leur `descriptor+0x28` est le deser KEYFRAME,
  pas le frame-deser (wrapper distinct) — cf [[reference_filmdec_descriptor_vs_frame_deser]].
- **Prochain pas** : investiguer le binding de slot 1038 (offline) ; option CE = snapshoter le binding World complet
  (slot→archétype) pour ISOLER « binding cassé » vs « composants cassés ». Mémoire [[project_deadstate_offline_misaligned_ceproof]].

---

## §0. TL;DR — où on en est (MAJ 2026-06-12, voir §0bis pour le reframe 06-13)

- **Ce qui MARCHE, livrable** : pipeline warp `apps/go-api/cmd/tmp_offwarp` — arme par kill 100% offline,
  **96% Fiesta / 58% Team Slayer**. Validé vs vérité-terrain live. **PAS encore en prod** (reste à brancher
  dans `internal/sync/backfill_weapons.go`).
- **Pourquoi 58% en Team Slayer (PLAFOND, ne pas re-tenter de bricoler l'horloge)** : le warp corrèle 2 HORLOGES
  (dégât `0xd2` = horloge flux ; kill-feed `chunk_27` = horloge temps-jeu) avec un résidu ~1-2s. En Team Slayer
  tout le monde a BR+MA40 alternés sur ~1-2s → mauvaise rafale. **C'est un mur d'horloge, pas un manque de données.**
- **La SEULE voie pour dépasser 58%** : corrélation MÊME-HORLOGE = lire le kill (tueur+victime) dans le **flux
  FRAME** (même horloge que `0xd2`). Ça exige le **DÉCODEUR ECS bit-exact** (décision user du 2026-06-05,
  `V7.5/film_re/PLAN_FILM_ECS_DECODER.md`).
- **Le mur ECS, et le levier NEUF (2026-06-12)** : le décodeur doit être **STATEFUL** (un delta se décode contre
  l'état de la frame précédente : counts RAM-dérivés type i63 `loop2`, `recordStateParam`, résolution dead-state).
  Le filmdec actuel est stateless → c'est LA cause du « garbage ». Levier neuf : les largeurs de quantification
  (qu'on croyait nécessiter CE) sont **dérivables des constantes statiques du .exe** (`step(L)=2^(16-L)/120`,
  validé L=0→6/6/6). Voir mémoire [[reference_filmdec_quant_width_formula]].
- **Direction committée** : construire le décodeur ECS STATEFUL (offline-pur). Gradient de pilotage =
  `cmd/tmp_cleanframe` (% frames propres, 17.87% au 2026-06-12). Oracle de validation = capture CE
  `tools/ce/filmdec_deadstate_capture.lua` (faite, victime/tueur/GID résolus base 0xE1500000).

---

## §1. ORDRE DE LECTURE (ne rien sauter)

1. **CE fichier** (index + impasses).
2. **`V7.5/killweapon/KILLFEED_STATE.md`** — source de vérité consolidée (état stratégique, §0 = vérité live validée).
3. **`V7.5/killweapon/KILLFEED_DEATH_RECAP_FIELDS_RE.md`** (10 juin) — inventaire RE le plus complet : grammaire dead-state
   (`FUN_140c1dd44`) + kill-event (`FUN_14104bd08`), le mur du reach, et §7 = TOUS les scans de localisation
   négatifs. **LE doc qui liste les impasses.**
4. **`V7.5/film_re/PLAN_FILM_ECS_DECODER.md`** + **`V7.5/killweapon/PLAN_FILM_KILLFEED_V3.md`** — la décision « build ECS complet » (5 juin).
5. **`V7.5/killweapon/WALK_PORT_NOTES.md`** — port bit-exact du walk biped (i54/i59/i63), position-linking réfuté.
6. **`V7.5/killweapon/FIREARM_PER_KILL_OFFLINE_SOLVED.md`** + **`V7.5/killweapon/PLAN_SIBLING_DAMAGE_DESERS.md`** — le warp 96% (0xd2 + frères).
7. **`V7.5/killweapon/PLAN_SAMECLOCK_ATTRIBUTION.md`** — voie same-clock (avec corrections 2026-06-12 + formule largeur).
8. **`V7.5/killweapon/PLAN_WEAPON_PER_KILL_PRODUCTION.md`** — productionisation du warp (`backfill_weapons.go`).
9. **`thought_log.md`** (entrées du haut, 2026-06-12) — chronologie détaillée + corrections récentes.
10. Référence RE statique : **`V7.5/film_re/RE_EXE_GHIDRA_FINDINGS.md`**, **`RESEARCH_THEATER_RE.md`**,
    **`reference_film_chunks_structure`** (mémoire). Pour reverse externe : `V7.5/film_re/HANDOFF_FILM_RE_STATE.md`,
    `V7.5/film_re/GITHUB_RE_FINDINGS_EN.md`.
11. **`V7.5/icones/ETAT_DE_L_ART_ICONES.md`** (2026-08-08) — les ICÔNES du kill feed, extraites
    du jeu. Sans rapport avec la résolution de l'arme (elle est résolue), mais c'est là que se
    trouve la table `bitd 8646f61a` : `identifier` (murmur3) + `bitmap` + `bitmap index`, 85
    entrées, motif `killfeed_<nom>`. C'est le nommage que la chaîne sonore n'avait pas su donner
    — et le **Falcon** y figure sous son vrai nom, là où la banque Wwise disait « Pelican ».
12. Handoffs historiques (contexte, redondants) : `V7.5/killweapon/HANDOFF_WEAPON_ATTRIBUTION.md`, `V7.5/killweapon/HANDOFF_KILLFEED_VOIE_A.md`,
    `V7.5/killweapon/HANDOFF_KILLFEED_2026-06-07.md`, `V7.5/film_re/HANDOFF_FRAME_DECODER_L3.md`, `V7.5/killweapon/REPRISE_KILLWEAPON_FILM.md`.

**Mémoires à charger** : [[reference_killfeed_deadstate_fields]], [[project_kill_feed_frame_decoder]],
[[project_killfeed_damage_source_goal]], [[feedback_no_fire_events_weaponv3]], [[reference_film_chunks_structure]],
[[reference_filmdec_quant_width_formula]], [[feedback_always_offline_pure_universal]].

---

## §2. PISTES DÉJÀ FAITES & TESTÉES — VERDICTS (⛔ NE PAS RE-TENTER)

| # | Piste | Verdict | Où |
|---|---|---|---|
| 1 | **Held-weapon au tick** (arme équipée du tueur) | ⛔ **REJETÉ par l'user ~10×**. La méthode = source de dégât, pas l'arme tenue | mémoire `project_kill_feed_frame_decoder` |
| 2 | **0xd2 → victime par bit-scan** | ❌ insuffisant ; record victime-keyé sans champ propre → nécessiterait CE | `FIREARM…SOLVED` §95-106 |
| 3 | **dead-state +0x10 = arme** | ❌ **ABANDONNÉ** : CE mesure +0x10 CONSTANT (116963283 RAM ; 0x06f8b7d3 ce tour) → ne distingue pas le modèle. Le dead-state porte tueur·victime·**catégorie** (mêlée/lancé), PAS l'arme | `KILLFEED_DEATH_RECAP_FIELDS_RE` §2bis, §5 |
| 4 | **Localiser le kill-event component par scan/pattern** (victim/killer/assist) | ❌ **TOUS négatifs** : grammaire 5-bit trop permissive (29257 frames), tmp_killevtscan/killevttime/xuidcluster | `KILLFEED_DEATH_RECAP_FIELDS_RE` §7 |
| 5 | **Position-linking** (record→victime par position monde) | ❌ réfuté | `WALK_PORT_NOTES` (section position-linking) |
| 6 | **Marqueurs DamageReport frères (0xe9/0x89/0xc7…)** | ✅ devenu le **warp 96%** (= la solution actuelle) | `PLAN_SIBLING_DAMAGE_DESERS` (RÉSOLU), `FIREARM…SOLVED` |
| 7 | **Resserrer/bricoler le warp d'horloge** pour battre 58% | ❌ **PLAFOND** : 2 horloges, résidu irréductible offline. Team Slayer = 58% (re-confirmé 2026-06-12) | ce fichier §0 |
| 8 | **fire/melee/grenade events pour l'arme** | ⛔ rejeté par l'user (v1/v2 non fiable) | mémoire `feedback_no_fire_events_weaponv3` ; `FIRE_MELEE_GRENADE_EVENTS` (attaquant OK, arme NON) |
| 9 | **Décodage STATELESS du dead-state** (filmdec actuel) | ❌ **garbage** : 2.2% (bruit), même en frame propre. EnumA/EnumB ne corrèlent pas au slot, 40% de gates « absents » faux | thought_log 2026-06-12 ; tmp_dsvalid/dsraw/dsonset |
| 10 | **Sweeps globaux `recordStateParam` / `defaultReplRange`** pour caler le reach | ❌ plats (valeur PAR-RECORD/stateful, pas globale) | thought_log ; tmp_replsweep |
| 11 | **Largeurs quantif = nécessitent CE (world_dump)** | ❌ **RÉFUTÉ 2026-06-12** : dérivables du .exe statique (`step(L)=2^(16-L)/120`, validé) | [[reference_filmdec_quant_width_formula]] |
| 12 | **Priorité reach « i23 → i0/i1/i5 »** (ancien handoff) | ❌ **FAUSSE** : dead-state = i11, i23 est APRÈS ; i0/i1/i5 absents de 44% des morts. Vrai bloqueur = i63 + stateful | thought_log 2026-06-12 |
| 13 | **Voie debugger CE (live) pour l'arme** | ✅ **MARCHE** mais LIVE only (résolveur `FUN_140495abc` plante offline). Sert d'ORACLE, pas de produit | `KILLFEED_STATE` §0 |
| 14 | **Gradient `tmp_cleanframe` comme proxy d'alignement dead-state** | ❌ **INVALIDE** (2026-06-13) : frame propre ≠ bit-exact (faux-mais-compensé). Valider au ground-truth `tmp_cematch` | §0bis ; tmp_cematch |
| 15 | **Les 413 dead-states `tmp_dsraw` (slot519, EnumA=-1 40%) sont des morts réelles** | ❌ **GARBAGE prouvé CE** : 0/11 morts CE atteintes ; bits Mort lus à de mauvaises positions | §0bis ; tmp_cematch |
| 16 | **descriptor+0x28 = le frame-deser (velocity i3 / obje i9)** | ❌ c'est le deser KEYFRAME ; le frame-loop utilise un wrapper distinct (i3 FUN_140d87740, i9 FUN_14080c1f8). Appliquer le leaf TANK le décodage | [[reference_filmdec_descriptor_vs_frame_deser]] |
| 17 | **i5 shield-vitality = R(8)+3×R(1) (template body-vitality)** | ✅ **CORRIGÉ** : vrai deser FUN_140d50cbc = 29-55 bits (presence+regen+R16+4flags). i5 était sous-lu de 18-44 bits | `components_object.go` ; thought_log 06-13 |
| 18 | **Bloqueur reach = largeurs composants i23/i0/i1/i5/i6/i7/i9 (diag 12 juin)** | ⚠️ **EN AVAL du vrai bloqueur** : la mort désync AVANT, à bit 14, sur un slot NON LIÉ (1038). Binding World incomplet d'abord | §0bis ; tmp_cematch VERBOSE |

**Résumé impasses** : tout chemin « pas-de-décodeur-ECS » a été essayé et plafonne (warp 58%, scans négatifs,
dead-state stateless garbage). Le held-weapon et les fire-events sont interdits. Reste UNE voie : le décodeur ECS
stateful (offline-pur), avec les levers neufs (formule largeur statique + gradient + oracle CE).

---

## §3. PLAN COMMITTÉ — décodeur ECS STATEFUL (offline-pur, universel)

**Pourquoi stateful (la cause racine)** : un record delta n'encode que les CHANGEMENTS, décodés contre le blob
d'état de l'entité (le moteur fait `memcpy` du baseline dans `FUN_141f86b58` puis applique le delta). Les counts
sont RAM-dérivés : i63 `count2 = FUN_1409fe718(state,0x49)` = popcount d'un masque 73-bit de l'état ; idem
`recordStateParam`, résolution dead-state. **Sans maintenir l'état par entité, c'est indécidable** → d'où le garbage.

- **Phase 1 — rendre filmdec STATEFUL** : blob d'état par slot d'entité, updaté à chaque NEW/delta. Débloque i63
  (loop2), recordStateParam, la résolution. État reconstruit DU FILM (zéro CE). Réf moteur : `FUN_141f86b58`.
- **Phase 2 — reach propre** : monter `tmp_cleanframe` → ~100% (porter i63 + composants restants avec la formule
  largeur statique). Lire (tueur, victime, stream-ts). Valider vs oracle CE `*_deadstate.bin`.
- **Phase 3 — corrélation same-clock** : tueur/victime (flux) ⋈ dégât `0xd2` (flux, même horloge) → arme exacte,
  tous modes. Cible : >90% Team Slayer.

**Anchors kill dans le flux** (2 options, grammaire offline connue) : (a) dead-state i11 sur le biped victime
(`FUN_140c1dd44` : +4 victime, +8 tueur, +0x10 source) ; (b) kill-event component (`FUN_14104bd08` : tueur/victime/
**assistant**). (a) est porté ; (b) reste à localiser (le décodeur ECS le localise).

---

## §4. OUTILS (cmd/tmp_* et CE) — inventaire

**Pipeline / preuve** :
- `cmd/tmp_offwarp <matchID>` — LE pipeline warp (96%/58%). Décode 0xd2 + roster type-8 + warp + attribution.
- `cmd/tmp_cleanframe [maxChunk]` — **GRADIENT** : % frames propres (jusqu'à recEnd) + histo composant de désync.

**Diagnostic dead-state (2026-06-12)** :
- `cmd/tmp_archdump` — ordre des composants du biped #35 (dead-state = i11).
- `cmd/tmp_dspre` / `tmp_dsvalid` / `tmp_dsraw` / `tmp_dsonset` — co-présence, validité, distribution brute, onsets.
- `cmd/tmp_killeridx` — validation tueur/victime vs kill-feed (garbage en stateless).
- `cmd/tmp_dscap [prefix]` — décode la capture CE dead-state (handle→idx base 0xE1500000). **Stride 0x60** (corrigé 06-13).
- `cmd/tmp_cematch [prefix]` — **VALIDATEUR GROUND-TRUTH (06-13)** : apparie chaque mort CE résolue à sa frame offline
  (fingerprint + b2c) et vérifie si le décodage atteint le dead-state à b2c. `VERBOSE=1` dump la chaîne de records
  de la 1ère frame de mort (montre le slot non-lié). Actuellement **0/11** — cible 11/11.
- `cmd/tmp_dsmatch [maxChunk] [winMs]` — cross-check offline : EnumA/EnumB décodé vs kill-feed chunk_27 (au hasard = faux).

**Capture CE (oracle, pure-read)** — dans `tools/ce/` :
- `filmdec_dualcap_capture.lua` — dual-hook dégât+kill (rdtsc) → `<prefix>_{dmg,kill}.bin`. (97/98 validé)
- `filmdec_deadstate_capture.lua` — hook `FUN_140c1dd44`, dump victime/tueur/GID résolus + bitreader + rdtsc.
- Pilotage via MCP `cheatengine` (CE attaché à HaloInfinite.exe + bridge). Setup : [[reference_cheatengine_mcp_setup]].

**Vérité-terrain** (`tools/ce/`) : `dmgcapture_run2.bin`/`killcapture.bin` (000d5950, 97/98),
`9b191a7f_{dmg,kill}.bin` (Team Slayer), `000d5950_deadstate.bin` (dead-state résolu).

**Build** : outils filmdec (sans DuckDB) → `CGO_ENABLED=0 go run ./cmd/...`. DuckDB → `CGO_ENABLED=1` +
`export PATH=/c/msys64/ucrt64/bin:$PATH`. gofmt requis (pre-commit). Films en cache : `data/cache/film_chunks/<id>/`.

---

## §5. FAITS RE CLÉS (constantes, ne pas re-chercher)

- **0xd2 (dégât)** `FUN_14080c1f8` : en-tête 36 bits → R5 attaquant (bit 36, `slot×2`) → slot/cause → famille
  (bit ~41) + suffixe `0x42c9679f`. Roster type-8 → slot→xuid (8/8). Cf `FIREARM…SOLVED` §1-2.
- **dead-state (i11)** `FUN_140c1dd44` : Mort(+0x70 R1), anim-handle(R1+optR32), R8, **+4 victime / +8 tueur**
  (R1+optR5, **résolus** via `FUN_14049746c`+`FUN_140e958c4` = index datum-table → base 0xE1500000 stride 0x10002),
  +0xc R4, +0xe R3, **+0x10 source** (R1; R1; R32 brut = global-id, mais CONSTANT → pas le modèle d'arme).
- **kill-event** `FUN_14104bd08` : tueur/victime/assistant (R1+optR5 ×3) + R32 scalaires. Grammaire offline OK,
  **localisation dans le flux = ouverte**.
- **largeur quantif** : `step(L)=2^(16-L)/120` (C=0x3c088889@143cd9758) ; `width=min(26,bitLen(ceil(40000/(2·step))))`
  ; range défaut ±20000 (`DAT_143b8c6b8`). Setup map-load = `FUN_140be9a14`. L=0→6/6/6 (= mesure CE). Cf §0.
- **i63 biped-action** `FUN_142f26a20` : subblock(96) + count1=R(4) + count1×{R(7)+`FUN_142ef1734`[R5 tag+body]}
  + count2=`FUN_1409fe718(state,0x49)`[popcount 73-bit STATEFUL]×{R(1)[+`FUN_14076e304`]} + subblock(96).
- **structure film** : chunk_00 = registre archétypes (1.97 Mo), chunk_01 = keyframe, 02-25 = frames gameplay,
  chunk_26 = re-sync, chunk_27 = type-9 highlight events (kill-feed killer/victime/temps **94/94 offline**).
  Cf `reference_film_chunks_structure`.

---

## §6. RÈGLES (comportement)
- ⛔ JAMAIS held-weapon / fire-events pour l'arme (rejeté). Méthode = source de dégât.
- Offline-pur + universel TOUJOURS (pas de CE/runtime au décodage, pas de calibration par-map). CE = oracle only.
  Cf [[feedback_always_offline_pure_universal]].
- Worktree `feat/weapon-attribution-v3`, JAMAIS main. Demander avant tout commit. Pas d'emoji. Prévenir avant op VPS.
