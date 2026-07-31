# KILL FEED — DOC CONSOLIDÉ (source de vérité unique) — 2026-06-07

> ⭐⭐ **MAJ 2026-06-12 — LE « VERDICT NÉGATIF source-par-kill » CI-DESSOUS NE VAUT QUE POUR L'OFFLINE.** La voie
> debugger ground-truth (CE-MCP au replay) a été exécutée et MARCHE : **arme-à-feu-par-kill obtenue LIVE** via
> `FUN_1407e00ac` (apply dégât) — attaquant `param_3+0x0c`, victime `*param_2`, arme `param_3+0x10|0x14` ; marqueur
> finishing-blow = `FUN_1406730c4`. Validé 000d5950 : 98 kills, totaux/tueur exacts, 17 familles = décode offline
> (Disruptor 0x84BD29ED…). Mêlée/grenade = déjà offline (cf FIRE_MELEE_GRENADE §2/§3). QUI+assist+% par kill =
> kill-event component `FUN_14104bd08` (+4 tueur, +8 %dmg, +0x10 assistant, +0x14 %assist). Détail RE + offsets :
> mémoires `reference_killfeed_deadstate_fields` (§MÉCANISME GROUND-TRUTH) + `reference_cheatengine_mcp_setup`.
> Portée : LIVE non scalable (1 replay/match) ; le mur biped→joueur reste pour le SCALABLE offline.

> ⭐ **TOURNANT 2026-06-07 — LIRE `.ai/V7.5/killweapon/FIRE_MELEE_GRENADE_EVENTS.md` D'ABORD.** La conclusion « pas
> d'attaquant offline » de ce doc vaut UNIQUEMENT pour les RECORDS DE DÉGÂT (`FUN_14080c1f8`, 519 records,
> ni tueur ni victime). Les EVENTS de tir/mêlée/grenade, eux, **portent le PLAYER INDEX de l'auteur** →
> la voie « source par kill » est ROUVERTE. Grenade ✅ décodée (70 events, player index 0-7). Mêlée ⚠️
> calibrée (marqueur 0x534/0x535, type 0x47 ; player-index offset TBD). Fire ❌ à décoder (heuristique
> user : co-localisé avec melee/grenade). Tout le reste de ce §0 (records de dégât) reste valide comme
> data d'arme + cross-check, mais N'EST PLUS le chemin principal de l'attribution par kill.

> Consolide : `KILLFEED_RE_FINDINGS.md` (RE mécanisme), `../film_re/HANDOFF_FRAME_DECODER_L3.md` (décodeur),
> `PLAN_FILM_KILLFEED_V3.md` + `../film_re/PLAN_FILM_ECS_DECODER.md` (archi), `../film_re/HANDOFF_FILM_RE_STATE.md` (handoff externe),
> et les entrées `../../thought_log.md` 2026-06-02→07. À RELIRE EN PREMIER avant toute action kill-feed.

## 0ZÉRO. R1 — MAKE-OR-BREAK TRANCHÉ (2026-06-07, décompile bridge HTTP) : LA SOURCE DE DÉGÂT EST **SÉRIALISÉE DANS LE FILM**
> Mission R1 : « l'objet `param_3` (descripteur de dégât que lit `FUN_1407e00ac`) est-il désérialisé du film
> ou construit live ? ». **VERDICT : SÉRIALISÉE DANS LE FILM — OUI.** Cela **INVALIDE la "CONCLUSION DURE"
> du §7quater** (« aucun champ kill→arme prêt-à-lire, association faite au replay par pipeline live »). FAUX.
> Établi par décompile de la **paire deser/apply** enregistrée dans la table de dispatch + lecture des offsets.

**PREUVE (adresses Ghidra, image_base 0x140000000) :**
1. `FUN_14080a18c(p1,p2,param_3,p4,param_5)` → appelle `FUN_1407e00ac(param_2,param_3,param_5)` (gate `FUN_14080a138`).
   `FUN_14080a18c` n'a **AUCUN appelant direct** : référencé seulement par 2 pointeurs DATA = slots de table de dispatch :
   - Table 1 (RVA, structs **12 o** `[deser_fn][apply_fn][disc]`) : slot apply `1453fd314`, slot deser `1453fd530`.
   - Table 2 (array de **ptr 64-bit**, vtable) : apply `143d0ad18`, deser `143d0ad08` (**slots adjacents, 0x10**).
2. **LE DESERIALISEUR DE `param_3` = `FUN_14080c1f8`** (l'autre moitié de la paire, même type d'event).
   Signature `FUN_14080c1f8(p1,p2,byte* param_3,longlong param_4,char param_5)` : `param_3` = le descripteur
   (`memset(param_3,0,0x328)` = les ~0x320 o du §7quater), `param_4` = **l'état du bit-reader film**
   (`+0x2c` bitpos, `+0x30` registre 64-bit MSB-first, `+0x38` count, `+0x40` ptr octets ; primitives
   `FUN_1406cf008`=read-1-bit, `FUN_1406d6c7c`=read-N, `FUN_1406d00ec`, `FUN_14080dec4`=R(32) big-endian — **famille
   bit-reader film** identique à FUN_140cec0a0/FUN_14080d6f0).
3. **LES CHAMPS QUE LIT `FUN_1407e00ac` SONT ÉCRITS PAR CE DESERIALISEUR depuis le flux :**
   - `*(param_3+8)` = **slot/cause de dégât** ← `*(param_3+8) = FUN_1406d00ec(param_4)` (lu du bitstream).
     `FUN_1407e00ac` le lit lignes 145/151/152/210/336 (`inv[+0x48 + slot*4]`, `unité+0x764`, etc.).
   - `*(param_3+0x14)` = **variant_name R(32) = FAMILLE D'ARME** ← `FUN_14080dec4(param_4,"variant_name",param_3+0x14)`
     (LE MÊME reader `variant_name` que le WST i43 et le NEW record d'entité-arme — clé `analysis.WeaponIDToName`).
   - `*(param_3+0xc)` = global-id ← `FUN_1407f2034(param_4)` ; `*(param_3+0x10)` ← `FUN_14080d69c(param_4,…)`.
   - `*(param_3+0x18)` = **datum-id tag `'weap'` (LA SOURCE)** ← `FUN_14080cd58(param_3+0x10, *(param_3+0x14))`
     (=`GetLocalHandleFromGlobalId`). `FUN_1407e00ac` le lit `FUN_140583b80(param_3+0x18, 0x77656170='weap')`
     (l.198/204) + validation `local_f420+0x2c == *(param_3+0x18)` (l.214). FourCC `0x77656170` = `'weap'`.
   - Le descripteur porte AUSSI : hit-sections `param_3+0x40` (stride 0xc, count `+0x34`) et `param_3+0x110`
     (stride 0x18, count `+0xf8`), positions `param_3+0x2a0`, etc. — **tous lus du bitstream** (boucles bit-read).

**CONSÉQUENCE DIRECTE (corrige §7quater "CONCLUSION DURE" + §7ter (A)) :**
- Le film **SÉRIALISE un record de dégât complet par event de dégât**, contenant la **SOURCE** (`'weap' @+0x18`,
  famille `variant_name @+0x14`, global-id `@+0xc`) ET le **slot/cause** (`@+0x8`). `FUN_1407e00ac` ne *reconstruit*
  PAS la source : il **résout** un id déjà désérialisé (`+0x18`) et le valide contre l'inventaire RAM du tueur
  (le `+0x48[slot]`/`+0x764` RAM sert de *cohérence/cross-check*, pas de source primaire).
- L'erreur historique (§7ter A : « aucun bitreader n'écrit le champ d'attribution ; le champ est reconstruit au
  replay ») venait d'avoir cherché `report+0x1f30` sur le **Participant record** (`FUN_1406cfb28`, 0x1F40 o, =
  PlayerHandle/+0x538). Mais le descripteur de DÉGÂT est une **AUTRE structure** (0x328 o) avec son **propre
  désérialiseur `FUN_14080c1f8`** dans la table de dispatch. C'est cette structure (param_3), pas le Participant,
  qui porte l'arme par kill. `+0x1f30` n'est qu'une copie runtime ÉCRITE depuis `param_3+0x18` (l.256 :
  `*(lVar16+0x1f30)=uVar24`), pas la source.

**⇒ POUR L'APP (offline, méthode FIDÈLE et DIRECTE) :** décoder le record désérialisé par `FUN_14080c1f8` par event
de dégât → lire `+0x14` (famille `variant_name`) et `+0x18` (datum `'weap'`) + `+0x8` (slot/cause). C'est la
**source de dégât PAR KILL, littéralement dans le film** — couvre arme/grenade/mêlée/terrain via le `'weap'`/cause,
sans suivre l'inventaire ni les fire-events. Reste à : (a) localiser ce record dans le flux (quel chunk/event-type
le porte ; la table de dispatch 1453fd2f8/143d0ad18 = la classe d'event) ; (b) câbler le walk pour atteindre
`FUN_14080c1f8` ; (c) valider vs CE 12 kills. **C'EST LA VOIE B du §5, désormais PROUVÉE viable** (record kill-event
localisé conceptuellement = le record désérialisé par FUN_14080c1f8). NB : `+0x14` = high-32 (famille) ; variante
exacte (low-32) non dans ce champ — la famille suffit (but figé).

## 0quinquies. E1 — PREUVE SUR LE FILM (Go, 2026-06-07) : proxy temporel + held-weapon + dead-state RÉFUTÉS empiriquement
> Mission E1 : prouver sur le film (000d5950), par kill, tueur/victime/temps + source-id BRUT + méthode dead-state.
> Probes : `apps/go-api/cmd/tmp_dmgsource/main.go` (tableau par kill) + `apps/go-api/cmd/tmp_dmgrecord/main.go`
> (signature du record de dégât). VERDICT : source par kill FIABLE = **NON** via les canaux offline actuels.

**CHIFFRES (93 kills, 000d5950) :**
- Kill feed chunk_27 = 93/93 (tueur/victime/temps). `medalType`=0 sur 93/93 ⇒ pas de méthode dans le kill event.
- **Proxy littéral d'arme id64 (885 horodatés)** : présent fenêtre [-3s,+200ms] = 87/93 (94%), MAIS non-ambigu
  (1 seule famille) = 24% seulement. **Kills narrés = 0/10 matchés.** Marteau IKE→JGtm (115.5/292.5/355.7/375.1s) :
  AUCUN Gravity/Rushdown Hammer (0x841ac5e5) en ±1.5s. BR75 JGtm→Akatsuki (112.9/329.8s) : aucun BR75. Le proxy
  capte les armes de TOUS les joueurs du Fiesta, pas celle du tueur ⇒ **réfuté empiriquement**.
- **Record de dégât FUN_14080c1f8 : 0 signature littérale dans le flux delta.** `'weap'`(0x77656170) = 0 occ dans
  chunks 02-26 (byte ET bit-aligné) ; 19 `'weap'` + 319 `'obje'` UNIQUEMENT dans chunk_00 (schéma ECS). ⇒ le
  datum-id `'weap'`@+0x18 est RÉSOLU au replay (GetLocalHandleFromGlobalId), PAS sérialisé comme FourCC par kill.
  Confirme R1 (le record porte un id résolu, pas un littéral) + ferme la voie "scan de signature".
- high32 famille dans le flux = 1159 occ (885 id64-complets + 274 high32-isolés), dispersés, ≠ 93 kills.
- Held-weapon ECS (tmp_killfeed_weapons) = 0/93 (walk biped désync, slot→xuid identifie 2/8 slots).
- Dead-state clean apparié ±400ms = 46/93 ; EnumA/EnumB/Val0c = bruit (walk désync, cf R3/§10.4).

**⇒ COUVERTURE par type : arme NON, grenade NON, mêlée NON, terrain NON (offline, canaux actuels).** Le GAP reste
le MÊME (R1/R2) : localiser le record FUN_14080c1f8 dans le flux (event-type + chunk via table 1453fd2f8/143d0ad..)
puis câbler le walk vers le deser. **Proxy temporel + held-weapon ECS + scan signature = empiriquement clos, ne pas
y réinvestir.** Le walk biped bit-exact (P1) reste le pré-requis transverse (débloque held-weapon ET dead-state).

## 0septies. E2 — LOCALISER LE RECORD DANS LE FILM (Go, 2026-06-07) : 4 voies offline closes empiriquement, 0/93
> Mission E2 : localiser les events de dégât dans le film 000d5950, décoder par kill (tueur/victime/temps +
> famille @+0x14 + slot @+0x08 via miroir FUN_14080c1f8), croiser chunk_27, valider les narrés. Probe :
> `apps/go-api/cmd/tmp_dmgrecord/main.go` (phases A→E, Go pur CGO_ENABLED=0). VERDICT : **record NON localisé**
> (taux source-par-kill décodée = **0/93** ; narrés OK = **0/2**). GAP identique à R1/R2/E1.

**CARTO CONTENEUR (Phase A/B) — 9 packet-types :** 0(frame ×30418), 1(registre 343Ko), 2(keyframe ~140Ko),
6(4o), 8(roster 25Ko), 9(714Ko ×1), **10(10o, ×30418 = APPAIRÉ 1:1 au type-0)**, 12(4o), 41(27o ×1). Le type-10
par-frame (seul candidat de packet d'event périodique non encore examiné) = **données de CAMÉRA SPECTATEUR**
(payloads = floats : `40 8d 0e..`, `81 02 34..` ; 171 distincts seulement ; b2/b3=0 sur 98.9%). **Éliminé.**
(chunk_01 = header spécial : ts wraparound, b2/b3 = octets de données.)

**SIGNATURE (Phase C) :** `'weap'`(0x77656170) = **0** hors chunk_00 (schéma ECS), toutes orientations byte. Le
high32 famille (variant_name) n'apparaît **QUE dans type-0** (381 hits autour des kills ; 0 dans tout autre
packet-type). ⇒ reconfirme R1/L2 : le record porte un id RÉSOLU non-littéral ; pas de packet de dégât signable.

**TAIL TYPE-0 (Phase D) :** décode chaque type-0 jusqu'au recEnd (boucle d'entités) → 18% clean, **tail moyen
65 bits**, high32 famille des frames clean = 63 dans le tail vs 27 avant recEnd (le tail concentre des armes).
MAIS les frames de KILL décodent ~toutes en **desync** (le walk biped casse précisément quand un biped meurt) ⇒
tail des frames de kill **inaccessible**. Le tail existe et porte des armes mais on ne peut pas l'atteindre aux kills.

**CONCENTRATION STRICTE (Phase E) :** frame type-0 du kill ±100ms → 25/93 kills ont ≥1 famille, 12/93 unique.
Narrés : **1 seule** occurrence correcte (Gravity Hammer 0x841ac5e5 @292490ms IKE→JGtm = coïncidence d'un
spawn d'entité-arme), **0 BR75** sur les 2 kills JGtm→Akatsuki. ⇒ le high32 capté = NEW records d'entités-armes
(pickup/spawn), PAS un record de dégât par kill.

**RE COMPLÉMENTAIRE (HTTP :8089, reconfirme L1/L2) :** `FUN_14054d014` (init) construit `DAT_144e61d88` : memset
`+0x8` (0x200 = registre ECS, 50 entrées, count `+0x208`=0x32) + memset `+0x210` (0x400 = registre EVENTS). Builder
`FUN_140e453b4` : ECS via `FUN_140e45fc4(reg,type,&cls)` (`reg+8+type*8`) ; events via affectations directes
`reg+0x210..+0x5e0`. **Damage = `reg+0x268` = event-type 11** (vtable 143d0ac08, deser @+0x100 = FUN_14080c1f8).
deser/apply = DATA-only (vtable+.pdata) ; `FUN_140620564` (dispatcher events) = .pdata-only. **Aucun reader plat
`tbl[type].deser` par xref statique** (= verdict L2 reconfirmé).

**⇒ E2 : 4 voies offline CLOSES** (type-10 caméra ; signature id-résolu ; tail bloqué par désync biped ; proxy/
concentration = armes-spawn). 2 seuls leviers restants : **(1) walk biped bit-exact** (linchpin : débloque le tail
type-0 ET held-weapon ET dead-state — c'est le pré-requis de TOUTES les voies) ; **(2) debugger** (breakpoint
FUN_14080c1f8 au replay Theater → packet-type + offset de bit réels). Ne plus réinvestir type-10/signature/proxy.

## 0sexies. L2 — FRAMING DU FLUX D'EVENTS : le dispatch est SCHÉMA-RÉCURSIF, pas index-par-type (2026-06-07, décompile bridge HTTP :8089)
> Mission L2 : trouver le reader du flux qui lit un event-type, indexe la table de dispatch (1453fd2f8/143d0ad..)
> et appelle le deser (FUN_14080c1f8 / FUN_14080cfe8) ; établir le framing (packet-type, encodage event-type,
> init bitreader). VERDICT : **il n'existe PAS de reader « lit-event-type → table[id].deser »**. Le décodage
> film est un système de **réflexion auto-descriptif** (registre de structs sérialisables, navigation par schéma),
> pas un dispatch plat par event-id. Tout établi par décompile + lecture mémoire des tables.

**(L2-a) LA STRUCTURE DE TABLE EXACTE (lue en mémoire, confirmée) :** chaque entrée = struct **12 o** de
**3 RVA u32** (relatifs image_base 0x140000000) `[serialize_rva][deserialize_rva][typedesc_rva]`. Lues :
- damage @`1453fd530` : ser=`FUN_142f193e4` (writer, écrit `variant_name`@+0x14 via `FUN_1407edaf4`),
  deser=`FUN_14080c1f8` ✓, typedesc=`0x143ee5020`.
- NEW @`1453fd584` : ser=`FUN_142f1?`, deser=`FUN_14080cfe8` ✓, typedesc=`0x143ee5090`.
- record-loop FRAME @`1453f0df8` : fn=`FUN_1406cd128` ✓ (!), typedesc=`0x143ecd3a8`.
- read-1-bit @`1453f0e4c` : fn=`FUN_1406cf008` ✓ (une PRIMITIVE est aussi une entrée).
⇒ La table N'EST PAS « les classes d'event » : c'est un registre EXHAUSTIF de TOUT ce qui est sérialisable
(primitives, record-loop, NEW, damage…). Le `typedesc` (3e RVA, zone `143ee.../143ecd...`) = **schéma de champs
bit-packé** (triplets offset/width + sous-tables `[ser][deser][disc]` imbriquées), pas un type-id numérique ni
une string de nom. (vtable ptr64 jumelle `143d0ad00` = même paire, slots {writer@+0, deser@+8, …, apply@+0x18}.)

**(L2-b) PERSONNE N'INDEXE CES TABLES PAR INSTRUCTION (le verrou) :** `search_instructions` operand `1453fd`,
`1453ea`, `143d0ad`, `143ee3` = **0 match** ; LEA/ptr64 vers la base ou les slots = **0** ; byte-search du
ptr64 `0x143d0ad00` = **0**. La table s'étend contiguë de `<0x1453a0000` à `0x1453fd…` (DES MILLIERS d'entrées
12 o). Elle n'est jamais chargée par adresse littérale ⇒ accédée via un **pointeur runtime** (registre construit
au boot) et/ou navigation par schéma (chaque struct pointe le typedesc de ses sous-champs, qui pointe leur deser).
**Conséquence : il n'y a pas une fonction « reader » unique qui fait `tbl[event_type].deser(...)`** à trouver par
xref. C'est le « GAP UNIQUE » du §0 reformulé correctement : le record n'a pas d'event-type top-level localisable.

**(L2-c) PREUVE QUE TOUS LES « DISPATCHERS » SONT TABLE-ONLY (xrefs) :** `FUN_14080c1f8`(deser damage),
`FUN_14080a18c`(apply damage), `FUN_1406cd128`(record-loop FRAME), `FUN_140620564`(dispatcher events 0x02-0x3c),
`FUN_1407f0c68`(ctor NEW) → **chacun référencé UNIQUEMENT par DATA** (slots de table), 0 caller-instruction.
Le NEW deser `FUN_14080cfe8` a 6 callers DIRECTS (`FUN_1407f2224`, `FUN_1408efb58`, `FUN_1408f0b48`,
`FUN_140f44c38`, `FUN_140fe7630`, `FUN_1410a5a74`) — mais **eux-mêmes sont tous table-only en amont** : ce sont
des deser de champ COMPOSITE qui inlinent le NEW comme sous-record. La chaîne ne remonte jamais à un reader plat.

**(L2-d) LE DAMAGE RECORD N'EST PAS DANS LE FLUX D'ENTITÉS TYPE-0 (prouvé négatif) :** le record-loop FRAME
`FUN_1406cd128` (= type-0, déjà porté Go `frame_records.go`, bitreader=param_3) traite NEW/DEL/DELTA d'**entités
ECS** uniquement. Ses callees (dont delta `FUN_141f86b58`, `FUN_141f86704`, `FUN_142a7782c`, `FUN_142f29538`,
`FUN_142f2b5c4`) **n'atteignent PAS** `FUN_14080c1f8`/`FUN_14080a18c`/`FUN_1407e00ac` (vérifié profondeur 1-2).
Le delta n'atteint pas le damage deser. Le typedesc damage `0x143ee5020` n'est référencé QUE par son propre slot
(`1453fd538`) ⇒ le damage record N'EST PAS un sous-champ d'un type parent connu : c'est une **racine de message**.

**(L2-e) FRAMING ACQUIS (la partie solide) :** packet header 16 o `[Type u16 LE][b2][b3][Size u32 LE][ts u64 LE]`
(cf `tmp_dmgrecord`/`tmp_loadout`). Bitreader film = struct `param_4`/`param_3` : `+0x40`=ptr octets courant,
`+0x10`=ptr fin, `+0x30`=registre 64-bit MSB-first, `+0x38`=bitcount, `+0x28/+0x2c`=positions, `+0x18`=limite
(validée `*(+0x18)*8 >= +0x2c`). Sortie observable = bitstream big-endian MSB-first (cf `filmdec/doc.go`). Type-0
FRAME = record-loop d'entités (`FUN_1406cd128`, prefix-code R(1)/R(2) + id table-driven). Type-2 keyframe = 8 NEW
bipeds (snapshot statique, `tmp_loadout`). Type-3 highlight (chunk_27) = kill feed décodé structurellement.
`FUN_140620564` (events 0x02-0x3c : spawn/pickup/objectif/équipement) consomme une struct event DÉJÀ décodée
(`type=*(param_2+8)`, payload `+0x10+`) — c'est un **apply**, pas le decode ; il ne porte pas le damage.

**⇒ RÉSULTAT L2 (réoriente P-locate) :** la cible « event-type + chunk porteur » telle que formulée **n'existe pas**
(pas de type-id plat pour le damage ; pas de reader index-par-type). Le damage record est une racine sérialisée
décodée par schéma (`FUN_14080c1f8`) hors du flux d'entités type-0. Les 3 voies pour la localiser réellement, par
ordre de ROI :
1. **DEBUGGER (recommandé, le seul déterministe)** : breakpoint sur `FUN_14080c1f8` pendant un replay Theater du
   film 000d5950 ; lire la pile + `param_4` (bitreader) au hit → donne le packet-type porteur, l'offset de bit de
   départ, et le caller réel (le dispatcher générique). C'est le fallback P5 du §0 — désormais **la voie n°1** car
   le RE statique est structurellement bloqué (table jamais chargée par instruction).
2. **TROUVER LE REGISTRE RUNTIME** : la table 12 o est copiée/indexée au boot par un initialiseur (zone `14011e…`
   fait des memset RVA via `LEA R8,[0x140000000]` — c'est de l'init de globales, PAS le dispatcher ; à élargir).
   Identifier la globale qui stocke la base de table puis ses lecteurs. Lourd, incertain.
3. **WALK SCHÉMA OFFLINE** : porter le décodage par typedesc (lire `0x143ee5020` comme schéma de champs et marcher
   le record damage quand on est positionné dessus) — mais sans (1) on ne sait pas OÙ se positionner dans le flux.

**NB cap-décisif :** la Voie A (held-weapon WST i43, §7quater) NE dépend PAS de localiser ce record et reste la
voie de build la plus avancée pour la SOURCE-au-tir. Le damage-record (Voie B) exige (1) le debugger pour franchir
le verrou de framing. Ne plus chercher le reader plat par xref statique (clos).

## 0quater. R2 — CONFIRMATION CONVERGENTE de R1 + 2 pistes ÉLIMINÉES (2026-06-07, décompile bridge HTTP + sonde Go)
> Mission R2 (indépendante de R1) : « quel record/event sérialisé du film porte (victime + source de dégât) par
> kill ? ». Résultat : **converge sur R1** (`FUN_14080c1f8` = le record). Et **élimine empiriquement** les 2 autres
> candidats (dispatcher d'events film ; dead-state +0x10). Établi par décompile complète + run `tmp_deathfield`.

**(R2-a) CANDIDAT RETENU = celui de R1, recoupé par un 2e chemin.** `FUN_14080c1f8` (deser descripteur de dégât
0x328 o) est référencé DATA à `1453fd530` ; `FUN_14080cfe8` (NEW record d'entité, **prouvé désérialisé du film**,
cf §0bis/§7ter B) est référencé DATA à `1453fd584`. **Même table `1453fd5xx`** (Δ0x54 ≈ 7 entrées de 12 o) ⇒ le
descripteur de dégât est dans le MÊME registre de désérialiseurs film que les NEW records prouvés présents dans le
flux. Champs (re-décompilés, identiques à R1) : `+8`=slot/cause (`FUN_1406d00ec`), `+0x10`=handle (R1;R32),
**`+0x14`=variant_name=FAMILLE (`FUN_14080dec4`)**, `+0x18`='weap' résolu (`FUN_14080cd58`), `+0x34..+0xf8`=array
hit-sections (stride 0xc), positions `+0x28/+0x30`. `param_5` (5e arg) = quantize vs R(32) brut.

**(R2-b) ÉLIMINÉ : le dispatcher d'events film `FUN_140620564` (switch event-type 0x02..0x3c) ne porte AUCUN
champ kill→arme.** Tous les cases décompilés et mappés. Aucun ne lit (victime+source d'arme). Les plus proches :
- case **0x30** (`FUN_14316c8d8`) = event "dégât/hit appliqué" : lit `param_2+0x14/+0x20/+0x2c` (3 positions),
  `+0x38/+0x3c/+0x40`, un scalaire **clampé** (`FUN_140475500`=max-puis-min), un modifier ; met biped `+0x500=3/4`
  (état) et `+0x538`(attaquant) — **PAS d'id d'arme/source**.
- case **0x28** (`FUN_14275d8dc`, défaut) = transfert d'état "death blow" INTERNE du biped (`+0xdf8..+0xe18` →
  `+0x1f20..+0x1f38`) — RAM-only, pas une source désérialisée.
- cases 0x21 (spawn/respawn), 0x22 (toggle entité), 0x23/0x24 (objectif/ability), 0x2d (équipement) = hors-sujet.
⇒ ce dispatcher = events de **réplication d'état/gameplay**, pas le record de dégât (qui est dans l'AUTRE table
`1453fd5xx` / `143d0ad..`, celle de R1). Confirme que le record de dégât n'est PAS un case de ce switch.

**(R2-c) ÉLIMINÉ EMPIRIQUEMENT : le dead-state `+0x10` (R32 brut) N'EST PAS la source** (réfute la piste B de
`KILLFEED_RE_FINDINGS §3`). Le deser dead-state `FUN_140c1dd44` lit bien un R(32) brut (`FUN_14080d6f0`) à `+0x10`,
résolu via le tag-registry global `DAT_144eae7b8`/`GetLocalHandleFromGlobalId`. MAIS run `tmp_deathfield` (667
dead-states Mort==true, match 000d5950) : `+0x10` = `0xFFFFFFFF` (absent) sur **557/667** ; sur les 110 présents,
**110 valeurs distinctes** (ratio distinct/obs = **0.99-1.00** = bruit), **0 match catalogue armes** (high32 ni
low32), et au croisement temps avec les 93 vraies morts le GID est absent ou hors-catalogue. Cause probable : le
résidu data-dépendant (position-vec `FUN_14076dc04` / orientation runtime-width) décale la lecture. ⇒ **le dead-state
porte tueur(+8)/victime(+4) + EnumA/EnumB (méthode de mort, vocab borné 33 val.) mais PAS l'arme/source.**

**⇒ VERDICT R2 (= R1, double-prouvé) : la SEULE source de dégât par kill sérialisée et fiable dans le film = le
descripteur de dégât `FUN_14080c1f8` (`variant_name @+0x14` = famille ; `'weap' @+0x18` ; slot/cause @+8). Couvre
arme/grenade/mêlée/terrain via le `'weap'`/cause.** RESTE LE MÊME GAP QUE R1 (le seul) : **localiser ce record dans
le flux** (quel chunk + quel event-type le porte) puis câbler le walk pour atteindre `FUN_14080c1f8` et valider vs
CE 12 kills. Les 2 autres pistes (dispatcher 0x02-0x3c ; dead-state +0x10) sont **closes** — ne pas y revenir.

## 0. VÉRITÉ VALIDÉE (2026-06-07) — LIRE EN PREMIER ; supersede tout conflit dans les sections historiques
> Établi par : décode film (filmdec) + RE Ghidra (décompile `FUN_1407e00ac`, `FUN_1407f0c68`, dispatch table)
> + workflow w5655pfne (6 agents). C'est la seule formulation à jour. Les §3/§4/§7ter historiques contiennent
> des étapes intermédiaires (certaines fausses, corrigées ci-dessous) — en cas de doute, CE §0 fait foi.

**FAITS VALIDÉS :**
1. **Kill feed offline** : `tueur · victime · temps` décodés du film (chunk_27), **93/93** (000d5950), **94/94** (jgtm).
   `slot→joueur` via chunk_27 `b36`(duo)+`b37`(team) = bijection per-match, scalable, sans world_dump.
2. **La famille d'arme EST dans le film.** Chaque **entité-arme** a un **NEW record désérialisé** portant
   `variant-name` = **FAMILLE** (clé `analysis.WeaponIDToName`) + son **local-handle**. Présente aussi comme
   8 loadouts keyframe + 885 littéraux id64 dans le flux. ⇒ **PAS de table-module runtime requise** pour la famille.
3. **LA SOURCE DE DÉGÂT EST SÉRIALISÉE DANS LE FILM, PAR ÉVÉNEMENT DE DÉGÂT** (R1, décompile — cf §0ZÉRO).
   Un **record de dégât dédié** (struct 0x328 o) est désérialisé par **`FUN_14080c1f8`** (vrai bitreader-film,
   même table de dispatch `1453fd5xx` que le NEW record d'entité `FUN_14080cfe8` déjà prouvé dans le film).
   Champs (offsets param_3) : **`+0x14` = `variant_name` R(32) = FAMILLE/SOURCE** (clé `analysis.WeaponIDToName`) ;
   `+0x18` = datum `'weap'` (source résolue) ; **`+0x08` = slot/cause** (arme/grenade/mêlée/terrain). `FUN_1407e00ac`
   ne *reconstruit* pas : il **lit** ce record désérialisé. ⇒ source par kill = read direct, **tous types**.
4. **MÉTHODE RETENUE (corrigée 2026-06-07)** : lire le **record de dégât** (`FUN_14080c1f8`) par event → `+0x14`
   (famille brute) + `+0x08` (slot/cause). C'est la **Voie B** (record kill-event), désormais PROUVÉE viable et
   SUPÉRIEURE à l'arme-équipée-ECS (couvre grenade/mêlée/terrain, pas seulement le tir). L'arme-équipée-ECS
   (Voie A) devient un fallback/cross-check.

**GAP RÉSOLU PAR CAPTURE LIVE (2026-06-07) — le record EST localisé : paquets type-0.**
> Capture debugger réussie (Ghidra dbgeng via ghidra-mcp, `go_blocking` avec WaitForEvent ; bp sur `FUN_14080c1f8`
> pendant le CHARGEMENT du film 000d5950 en Theater — les events se désérialisent au LOAD, pas en lecture).
> Détail complet : **`.ai/V7.5/killweapon/DBG_CAPTURE_dmg_record.md`** (registres, bitreader, message 214 o, pile). À NE PAS PERDRE.
- **Caller (lecteur générique) = `FUN_14080AADE`** (puis `FUN_14076A24E`) : lit l'entête 8 o du message + init le
  bitreader + appelle `FUN_14080c1f8`.
- **Bitreader (R9)** : buffer 214 o, à l'entrée du deser byteptr=+8, bitpos=24, registre `0x0004384BD2000000`, count=24
  (l'entête 8 o est déjà consommée ; le deser lit la suite, MSB-first même famille que `FUN_140cec0a0`).
- **MATCH OFFLINE PROUVÉ (`tmp_bufmatch`)** : le message capturé (214 o) se retrouve **VERBATIM** dans le chunk
  INFLATÉ `chunk_02 @0x9cbb2` = **packet-type=0, payload_off=0**. Le suffixe variant d'arme `0x42c9679f` est dans le
  message (octet ~8, bit-packé). ⇒ **les records de dégât sont des paquets type-0** (payload = le message).
- **CORRIGE l'analyse statique antérieure** (R1/R2/E1 disaient « pas type-0, pas signable, structurellement bloqué »).
  C'était FAUX : la capture live tranche → **type-0, verbatim, décodable offline**. La RE statique avait calé car la
  table de dispatch est runtime ; la capture a court-circuité ça.
- **DÉCODE OFFLINE RÉUSSI (workflow `wczwrzy3w`, 2026-06-07)** : preuves `cmd/tmp_dmgdecode` + `cmd/tmp_dmgscan`.
  - Message capturé décodé **bit-exact** → **Disruptor** (id64 `0x84bd29ed42c9679f`). = 1er record du flux (chunk_02, t=13.8s).
  - **CORRECTION field-map** : l'arme est portée par le **global-id `+0x0c`** (`R(5)+R(32 BE)` ; le R32 = high-32 FAMILLE,
    low-32 `0x42c9679f` contigu → id64), **PAS** `variant_name +0x14`. `+0x08` = slot/cause (consumeId2 : R1 puis R2 si 0).
    Décode : payload type-0, **startBit logique = 36** (8 o d'entête consommés par `FUN_14080AADE`), bitreader MSB-first BE (`filmdec.BitReader.Skip(36)`).
  - **DISCRIMINANT O(1)** : paquet type-0 dont **`payload[0]==0xd2`** = record de dégât-arme. **519 paquets `0xd2` == 519 records décodés stricts** (recoupement exact, 0 hors-0xd2). (NB : 30418 type-0 total ; 26548 commencent par `0xa0` = frames d'état d'entités.)
  - **519 records dégât-arme** sur 000d5950, couverture **13.8s→481.4s** (tout le match), **17 familles, 0 aberrante** :
    Needler 76, MA40 AR 67, Disruptor 59, Stalker 53, S7 Sniper 50, BR75 40, SPNKr 34, Ravager 25, Shock 25, Pulse Carbine 21, Mangler 20, Sidekick 13, Heatwave 9, Bulldog 8, Cindershot 7, Skewer 6, Hydra 6. `+0x08` slot/cause = 4 valeurs corrélées à la famille (type de projectile/slot).

⇒ **LA SOURCE-ARME EST DÉCODÉE OFFLINE, DÉTERMINISTE, VALIDÉE.** Reste 2 verrous pour la SOURCE-PAR-KILL :
1. **Record→kill : le message ne porte NI attaquant NI victime** (entête 8 o = enveloppe + compteur de séquence ;
   victime `+0x10` gate=0 dans 519/519). Un record = 1 **tick de dégât** (souvent non-létal), pas un kill. Appariement
   actuel via **ts du paquet** (header +0x08 u64 LE) ⋈ chunk_27 = **33/93 kills** à ±300ms (ambigu, plusieurs joueurs tirent).
   **Piste pour lever l'ambiguïté** : le **global-id `+0x0c`** (R5+R32) pourrait encoder l'**entité-arme attaquante**
   (→ joueur) — à RE/cross-référencer (+ le préfixe R5). Sinon : dernier/densest tick de même famille avant chaque death.
2. **Mêlée/grenade/terrain : 0 record d'arme** (Gravity Hammer = 0/519). Le kill marteau IKE→JGtm N'EST PAS attribuable
   par cette voie → path séparé via la **catégorie de mort du dead-state / chunk_27** (qui distingue déjà mêlée).

### ⇒ SOURCE-PAR-KILL : VERDICT NÉGATIF (workflow `w5v591zoz`, 2026-06-07) — la voie record-de-dégât ne suffit pas
> Probes : `cmd/tmp_dmgattacker` (A1), `cmd/tmp_killmethod` (M1), `cmd/tmp_killsource` (table).
- **(A1) Le record ne porte PAS l'attaquant.** `+0x0c` global-id = R5(5b)+R32 ; le R5 n'a que **4 valeurs** {1,3,17,19}
  (2 bits utiles) = attributs catégoriels (type projectile/partie touchée), **pas** l'identité du tireur. Victime `+0x10`
  gate=0 dans 519/519. ⇒ seul lien record→kill = **proximité temporelle** (ts paquet ⋈ chunk_27) = **NON fiable** (Fiesta).
- **(M1) chunk_27 ne classe pas la méthode.** Le bloc 60 o KILL/DEATH n'a aucun champ méthode (les 4 morts marteau de
  JGtm sont bit-identiques à ses morts à l'arme). Médailles b59 = partiel (44/93) + décode non-bijectif. ⇒ mêlée 2/93, grenade/terrain 0.
- **VALIDATION NARRATION = 0/6.** BR75 JGtm→Akatsuki : non-résolu (noyé dans le bruit). Marteau IKE→JGtm : 3 non-résolus
  + 1 **FAUX POSITIF** « Stalker Rifle » (tick d'un AUTRE joueur dans la fenêtre). Couverture fiable réelle = **0**.
- **CAUSE RACINE** : le record de dégât (519 ticks, famille EXACTE) ne contient ni tueur ni victime → l'appariement
  temporel attribue l'arme du mauvais joueur en Fiesta. **NE PAS productioniser cet appariement** (faux positifs prouvés).

### ⇒ ACQUIS RÉEL vs CE QUI RESTE
- **ACQUIS** : les **familles d'armes + temps** sont décodées offline, déterministe (519 records, validé bit-exact).
  = data d'usage d'arme / timeline de dégâts exploitable. Mais **PAS** la source-par-kill.
- **VOIE FIABLE RESTANTE pour la source-PAR-KILL** = **Voie A (arme équipée du tueur)** : finir le **walk biped**
  (port i54/i59/i63 → 8 bipeds/frame) → lire le WST i43 (famille) du **slot tueur** (connu via chunk_27 b36/b37) au temps
  du kill. Les 519 records servent alors de **cross-check** (la famille équipée doit matcher un record proche). C'est la
  méthode que l'user avait choisie (Option 1). Le détour record-de-dégât a CONFIRMÉ que la donnée arme est dans le film,
  mais pas le lien par kill — qui repasse par le walk biped.
- Alternative RE/validation (pas prod) : capture debugger au **`FUN_1407e00ac`** (l'apply) qui a attaquant(+0x538)/victime
  en RAM → ground-truth par kill pour 000d5950. Non scalable (replay in-game par film) → validation seulement.

### ⇒ DÉCISION USER (2026-06-07) : **VOIE A — finir le walk biped**
On finit le walk biped (port i54/i59/i63 → 8 bipeds/frame) → lire l'arme équipée (WST i43, famille) du **slot tueur**
(connu via chunk_27 b36/b37) au temps du kill → cross-check par les 519 records de dégât. Le **debugger Ghidra reste
en secours** (capture d'event en jeu au besoin — `go_blocking` + bp opérationnels, cf `.ai/V7.5/killweapon/DBG_CAPTURE_dmg_record.md`).
État walk (réf §7bis/§6bis) : slot 512 ~98% clean, 515 ~86%, **519 ~29%** ; résidu = branches amont **i54** (ctx+0x9d
mid-mobility), **i59** (tag==3 `FUN_142f25e90`), **i63** (count1>0) NON portées. C'est le « long pole » à franchir.

**ERREURS INTERMÉDIAIRES CORRIGÉES (ne pas y revenir) :**
- (§3 hist.) « `event+0x538` désérialisé = l'arme » → **FAUX** : `+0x538` = PlayerHandle (Participant record), pas l'arme.
- (§7ter / mon §0 v1) « le film ne stocke PAS la source, le jeu lit l'arme équipée au replay » → **FAUX/INCOMPLET** :
  on regardait le **mauvais record** (Participant `FUN_1406cfb28`, 0x1F40 o, = handles joueur). La source est dans un
  **AUTRE record** (Dégât `FUN_14080c1f8`, 0x328 o), bien sérialisé. La source EST dans le film (R1 corrige tout ça).
- dead-state : **PAS** d'enum death-type sérialisé (R3 réfute l'ancienne lecture EnumA/EnumB ; `+0x04`=victime,
  `+0x08`=tueur, pas une méthode). Le type (mêlée/grenade/tir) vient du record de dégât (`+0x08` slot + `+0x18` source).

**PLAN BUILD (linéaire, recalé R1 — Voie B = record de dégât) :**
- [FAIT] Kill feed 93/93 + slot→joueur. [FAIT] R1 : source SÉRIALISÉE prouvée (record `FUN_14080c1f8`, famille @+0x14).
- [EN COURS] **P-locate** (GAP unique) : localiser le record de dégât dans le flux. (1) lire le **discriminant/event-type**
  de l'entrée dégât dans la table de dispatch (`1453fd2f8` structs 12 o `[deser][apply][disc]` ; deser=`FUN_14080c1f8`) ;
  (2) trouver le **reader du flux d'events** qui itère le stream, lit un type, indexe la table, appelle le deser ;
  (3) en déduire packet-type + chunk porteur. (Fallback P5 : breakpoint debugger sur `FUN_14080c1f8` au replay Theater.)
- [À FAIRE] **P-decode** : miroir Go de `FUN_14080c1f8` → lire `+0x08` (slot/cause), `+0x14` (famille R32), `+0x18`
  (source brute), + handles attaquant/victime du record (mapper TOUS les champs du 0x328).
- [À FAIRE] **P-cross** : associer record↔kill (handle victime/attaquant OU ordre temporel) vs chunk_27 → `X·source·Y`.
- [À FAIRE] **P-valid** : vs CE 12 kills + narration (marteau IKE→JGtm, BR75 JGtm→Akatsuki…) — critère ≥10/12, ≥1 mêlée +1 distance.
- [À FAIRE] **P-prod** : analysis→service→handler, append-only ART-safe, capability-gated, multi-titre PathResolver.
- NB : le **walk biped bit-exact** (port i54/i59/i63) reste utile (fallback arme-équipée + dead-state) mais n'est
  PLUS le chemin principal — P-locate/P-decode du record de dégât ne dépend pas du walk biped.

## 0bis. MISSION A2 (2026-06-07) — FORMAT DU HANDLE & RÈGLE DE JOINTURE (décompile Ghidra)
> Établi par décompile de `FUN_1407e00ac` (handler dégât), `FUN_140477aa0`/`FUN_140471c88`/`FUN_140498800`
> (résolveurs), `FUN_140496e58` (entité tueur), `FUN_14080cfe8`/`FUN_14080d61c` (NEW record arme).
> RÉSULTAT PARTIEL : encodage handle + 2 espaces + règle de jointure = PROUVÉS. Reader WST i43 exact = NON
> décompilé (serveur MCP ghidra tombé en cours de mission) → 1 confirmation restante avant build.

**ENCODAGE DU HANDLE WORLD (datum-handle uint32) — prouvé identique dans 4 résolveurs :**
- `index = (handle >> 1) & 0x7FFF` (**15 bits**, bits 1..15).
- `table_selector = handle & 1` (**bit 0**) → choisit 1 des 2 tables (`TLS+0x4acc0[selector]`, ou fallback `DAT_144eb5b08[selector]`).
- `generation = handle >> 16` (**16 bits hauts**) → comparé à `slot.generation` (`*(short*)slot == handle>>0x10`).
- Slot world : `base=table[+0x78]`, `stride = table[+0x40]` (générique `FUN_140477aa0`) ou **0x18** (accès direct `FUN_140471c88`/`FUN_140498800`/`FUN_140496e58`) ; payload entité = `*(slot+8)` (ou `slot+0x10` pour les accès directs stride-0x18) ; `slot[+6]==0x02` = type entité vivante (biped) ; `slot[+3]` = capability-bits (le param_2=4 de `FUN_140477aa0` teste le **bit 2**).

**DEUX ESPACES DE HANDLE DISTINCTS (le point décisif) :**
1. **HANDLE WORLD (instance)** — table `TLS+0x4acc0[handle&1]`, décodage ci-dessus. C'est ce qu'on lit dans :
   - l'**inventaire du tueur** : `*(uint*)(lVar14 + 0x48 + slot*4)` où `lVar14 = FUN_140496e58(entité-tueur)`, `slot = *(int*)(param_3+8)` (slot/cause du dégât). Résolu par `FUN_140477aa0(&h,4)` → entité-arme instance.
   - le handle d'arme final `uVar10` repassé à `FUN_140498800(uVar10,4)`.
2. **HANDLE TAG (définition)** — table `DAT_144eae7b8` via `GetLocalHandleFromGlobalId` (`FUN_14080d61c`, string "GetLocalHandleFromGlobalId could not find local **tag** handle"). C'est `param_1[5]`(+0x14) du NEW record d'entité-arme. ⇒ **PAS le même espace que le handle world**. Le NEW record porte aussi `param_1[3]`(+0xC)=global-id (big-endian) et `param_1[4]`(+0x10)=**variant-name=FAMILLE**.

**CONSÉQUENCE — la jointure N'EST PAS `inventaire[slot] == param_1[5]`** (world-instance vs tag-def, espaces différents).
Le NEW record d'entité-arme ne contient PAS son propre handle world-instance ; il porte (global-id, famille, tag-handle).
⇒ Le lien biped→entité-arme passe par un **handle world-instance**, qu'il faut retrouver côté biped (WST i43 candidat).

**RÈGLE DE JOINTURE CIBLE (à confirmer par le reader WST i43) :**
- SI le WST i43 du biped contient un **handle world-instance** (même encodage `(h>>1)&0x7FFF | bit0-table | gen<<16`), ALORS
  jointure = `resolve_world(WST_i43)` == l'entité-arme instance, dont le `'obje'` (record NEW) porte `variant-name`=famille.
  Pas de transformation (pas de `>>n`/`&mask` au-delà du décodage standard ci-dessus) : **handle == handle** dans l'espace world.
- Le pont NEW-record↔instance-world se fait par **global-id** (`param_1[3]`) : l'instance world est créée et indexée par le
  global-id ; le record porte la famille. Donc map offline = `world_handle → global_id → (NEW record).variant_name`.
- Côté handler le jeu confirme le bon couplage par un check de **définition** (pas de handle) :
  `FUN_140477aa0(inv[slot],4)+0x2c == *(int*)(param_3+0x18)` (datum-id de tag `'weap'`). Et plus bas `FUN_140498800(uVar10)+0x2c` idem.
  ⇒ `entité-arme-instance +0x2c = datum-id de définition 'weap'` (le tag de l'arme) — c'est le pont instance→définition.

**CE QUI RESTE (1 décompile) :** lire le reader du composant WST biped (i43..i46) pour trancher : i43 = handle world-instance
(→ règle ci-dessus) OU index d'inventaire (→ `inventaire[i43]` puis resolve world) OU id64 inline. La structure de
`FUN_14080cfe8` (NEW record) montre que les sous-champs après `param_1[5]` sont des arrays de slots indexés (`param_1[0xb]`
= count ≤5, boucle `param_1[0xd + i*2]`) — cohérent avec un mini-inventaire embarqué. À RE au prochain accès ghidra.

## 1. BUT (figé + reformulé fermement 2026-06-07 — cf mémoire project_killfeed_damage_source_goal)
**La SOURCE DE DÉGÂTS qui a causé la mort, par kill, FIABLE, sur n'importe quel match : `X tue Y avec source Z`.**
- **Z = TOUS les types** : arme à feu, **grenade**, **mêlée**, **objet de terrain/environnement**. NE PAS réduire
  à « l'arme équipée/dégainée du tueur » (trop étroit, rate grenade/mêlée/terrain).
- **FIABILITÉ > LISIBILITÉ** : un **id brut** par kill suffit ; l'utilisateur fera la lookup id→nom par observation.
- OFFLINE en Go, depuis le film, scalable (zéro CE, zéro world_dump debugger).
- **Make-or-break** : la source de dégât est-elle ENREGISTRÉE par kill dans le film ? Logiquement OUI (le replay
  affiche mêlée/grenade/terrain correctement) → la TROUVER, pas la reconstruire.

## 2. ⛔ INTERDIT (rejeté par l'utilisateur — ne JAMAIS y revenir)
- **fire-events / weaponv3** (corrélation tir↔kill, v1/v2 ~85-90 %, jugé non fiable). C'est la voie REJETÉE.
- **dead-state `GlobalID` comme arme** : testé 0/637, c'est la MÉTHODE (mêlée/headshot), pas l'arme.
- Présenter la narration de l'utilisateur comme un résultat décodeur.

## 3. CE QU'ON SAIT (réconcilié — ne plus re-dériver)
- **Conteneur** : film = chunks zlib ; paquets `[Type u16][b2][b3][Size u32][ts µs u64]`. type-0=FRAME(~60fps),
  type-1=header/registre, type-2=keyframe(~20s), type-3=highlight events (footer), type-8=roster.
- **chunk_00 (type-1, 1.97 Mo)** = registre ECS : 118 archétypes, listes ordonnées de composants. biped=#35.
- **Mécanisme arme (RE Ghidra + CE-validé 12/12)** : l'event kill (`FUN_1406730c4`) porte `event+0x04`=victime,
  `event+0x08`=tueur, `event+0x538`=handle entité-arme, `event+0x1f30`=high-32 (FAMILLE). id64=high|low
  (low=`def+0x478` runtime). **Famille = high-32 = `variant_name` du `'obje'` de l'entité-arme = `rec.VariantName`**.
  `event+0x538` est **désérialisé du flux du film** → présent offline (KILLFEED_RE_FINDINGS §4).
- **Pas de mur de largeurs runtime** pour le walk delta (verdict L4, 2026-06-06) : `FUN_1406d84b4` = largeurs
  LITTÉRALES ; tables suspectées = bornes/dequant (0 bit). **Aucune capture CE requise** pour le walk delta.
- **L'arme N'EST PAS** : dans le bloc 60o du chunk_27 (prouvé) ; ni dans les deltas de MOUVEMENT du biped
  (les WST gate=1 portent de l'ÉTAT, pas le variant-name à aucun offset). Elle EST : au **keyframe (loadout)**
  + aux **records de swap/pickup rares** (WST dont handle@+1 ∈ catalogue) + comme **885 littéraux id64** dans le flux.

## 4. ✅ FAIT (offline, vérifié)
- **Kill feed killer→victime+temps+team+slot** : chunk_27 décodé intégralement. Couple = jointure KILL@t⋈DEATH@t
  équipe-opposée = **93/93 zéro erreur**. `b37`=team (8/8), `b36`=duo → **player-slot per-match (8 combos,
  bijection stable)** = le mapping slot→joueur, OFFLINE et SCALABLE (pas de world_dump). `b59`=medal_type (méthode proxy).
- **8 loadouts keyframe** décodés bit-exact (Hydra, Shock Rifle, SPNKr, Mangler, AR, Mangler, Cindershot, Bulldog…).
- **885 littéraux d'arme id64** décodés du flux type-0 (variante exacte présente).
- **Spine biped delta** : i0..i62 bit-exacts ; après port i63(tag0-5)/i50/i52/i53 + team-mapping : slot 512 98 % clean,
  slot 515 86 %, slot 519 25 % (résidu = i63 count1>0). Réf : `tmp_worldreplay`, `tmp_killfeed_weapons`, `tmp_loadout`.

## 5. ⏳ LE GAP — attacher la FAMILLE à chaque kill, offline
Deux voies de décode (mêmes données au fond : l'arme = l'entité-arme que l'event référence = l'arme courante du tueur) :
- **Voie A — held-weapon ECS (loadout keyframe + swaps)** : maintenir l'arme courante par slot (loadout + records de
  swap), lire au temps du kill pour le slot tueur (connu via chunk_27 b36/b37). Décodage ECS PROPRE (≠ fire-events).
  Reste : (a) i63 count1>0 (1 largeur de tag) ; (b) **isoler les records de swap** (WST handle@+1 ∈ catalogue) ;
  (c) atteindre les 8 bipeds = porter les **default-states NEW non-biped** (multi-archétype) OU bootstrap World film-seul.
- **Voie B — record kill-event** : localiser+décoder le record de l'event kill (event+0x1f30 high-32 famille direct).
  Plus « principielle » (la structure du jeu), mais le record n'est pas encore localisé dans le flux.
- **Note honnête** : A et B convergent (l'event référence l'arme tenue). A est le plus avancé. Le « held weapon »
  interdit visait la MÉTHODE fire-events non fiable ; ici c'est l'arme ECS désérialisée (fiable), = ce que l'event pointe.

## 6. PLAN (ordonné, effort croissant) — Voie A par défaut
1. **i63 count1>0** : raffiner la largeur de la branche tag fautive (`FUN_141fd4814`) → slot 519 clean → 8 bipeds atteints.
2. **Isoler les records de swap d'arme** : filtrer les WST/records dont handle@+1 ∈ `analysis.WeaponIDToName` → timeline
   arme-famille par slot (loadout init + swaps).
3. **Bootstrap World depuis le film seul** (prod) : keyframe → slot→archétype (porter default-states NEW par archétype),
   ou ancrage minimal bipeds. Élimine le world_dump debugger. Scalable centaines de matchs.
4. **Croiser** : par kill (chunk_27 : tueur slot + temps) → arme-famille tenue à t → `tueur · arme · victime`.
5. **Valider** vs CE 12 kills + narration (IKE→JGtm marteau, JGtm→Akatsuki BR75, …) + medals_earned (armes).
6. **Productioniser** : `internal/analysis` (pur) → `internal/service` → handler ; table append-only ART-safe ;
   capability-gated (pas slug en dur) ; multi-titres via PathResolver.

## 7. JOURNAL DÉCISIONS
- **2026-06-07** — DÉCISION USER (fork tranché) : méthode = **arme équipée exacte (ECS)** (Option 1). On
  reproduit le mécanisme du jeu : suivre l'arme équipée (définition) du tueur via l'ECS (loadout + swaps)
  et la lire au kill (tueur+temps via chunk_27). EXACT (résolution par slot d'inventaire, comme le jeu),
  ≠ corrélation fire-events 85-90 % (rejetée). Workflow w5655pfne (6 agents) confirme : pas de champ
  kill→arme stocké ; film désérialise PlayerHandle/BotHandle, PAS l'arme ; famille obtenue via les NEW
  records d'entités-armes (variant-name) → PAS de table-module requise pour la famille.
- **2026-06-07** — BUILD v1 — CONSTAT : `tmp_killfeed_weapons` → kill feed 93/93 OK, mais timeline
  arme par slot biped = **0 evt** (le scan littéral id64 dans la FENÊTRE du record biped ne trouve rien :
  les 885 littéraux sont dans des records d'ENTITÉS-ARMES séparés, pas dans le delta biped). ⇒ LINCHPIN
  confirmé (cohérent enquêteur B) : lier biped→entité-arme par **local-handle** (WST i43 du biped ↔
  `param_1[5]` du NEW record de l'entité-arme, qui porte `param_1[4]`=famille). `tmp_wsthandle` a échoué
  car il testait `handle>>13`→slot World ; le bon match = **handle == handle** (brut). PROCHAIN PAS BUILD :
  (P-link) décoder les NEW records d'entités-armes → map(local-handle→famille) ; lire WST i43 biped →
  local-handle ; jointure → arme équipée par biped ; propager loadout+swaps ; lire au kill.
- **2026-06-07** — BUT FIGÉ : famille suffit ; offline Go ; prod centaines de matchs.
- 2026-06-07 — CORRECTIONS de mes erreurs : (a) PAS de mur largeurs runtime (verdict L4) ; (b) slot→joueur RÉSOLU
  offline via chunk_27 b36/b37 ; (c) dead-state GID réfuté comme arme (0/637) ; (d) dérive held-weapon mal cadrée → recadrée Voie A.
- 2026-06-06 — chunk_27 60o décodé (team/slot/medal) ; spine delta i0..i62 + i63 tag0-5/i50/i52/i53 + team-mapping portés.
- 2026-06-06 — mécanisme arme cracké (event+0x1f30|def+0x478) + CE-validé 12 kills.
- 2026-06-05 — abandon weaponv3 fire-events (gain mineur/non fiable) → vraie v3 = reconstruction ECS.

## 7ter. VERDICT RE RÉCONCILIÉ (2026-06-07, workflow 4 enquêteurs A/B/C/D) — corrige l'ancien §7ter
> L'ancien §7ter (« le jeu RECALCULE, le film ne stocke PAS ») était à moitié faux. Verdict corrigé,
> étayé Ghidra par 4 enquêteurs indépendants (B et D conclus rigoureux ; A conclu ; C partiel).

**(D) `FUN_1407e00ac` n'est PAS une re-simulation physique.** La ref `'weap'` (`param_3+0x18`) est LUE 6×,
jamais écrite (`1407e0359/04bd/03bd/064b` + 2 LEA ; aucun store vers R14+0x18). La famille `[+0x1f30]`
(`MOV @1407e058b`) est écrite AVANT les seuls `SQRTPS @1407e0951/0972` → la distance ne CHOISIT pas l'arme
(elle sert à valider la portée + placer les hit-markers, effet dérivé). C'est une **RÉSOLUTION de handle**
(id→entité-arme→`'obje'`→variant_name/famille), pas un calcul géométrique. ⇒ « RECALCULE » du sens
« re-sim physique » = FAUX.

**(B) La FAMILLE EST dans le film, littéralement.** Le record NEW d'entité (`FUN_1407f0c68→FUN_14080cfe8`)
désérialise du bit-reader : `param_1[3]`(+0xC)=global-id ; `param_1[4]`(+0x10)=**variant-name=FAMILLE**
(`FUN_14080dec4`, clé `analysis.WeaponIDToName`) ; `param_1[5]`(+0x14)=handle LOCAL (`GetLocalHandleFromGlobalId`).
⇒ Chaque entité-arme porte sa famille, désérialisée, **sans table runtime**. (anti-dérive #1 confirmé.)

**(A) Le CHAMP d'attribution `report+0x1f30` n'est PAS un champ prêt-à-lire.** `FUN_140a203d4` (constructeur)
reset `+0x538/+0x53c/+0x1f30/+0x1f38` sur le MÊME `param_1` ⇒ c'est UNE structure (« Participant record »).
Le déserialiseur film `FUN_1406cfb28` écrit `+0x538`=PlayerHandle et `+0x53c`=BotHandle (désérialisés),
mais **JAMAIS `+0x1f30`** (=ParticipantUnitHandle, setté runtime). Recherche exhaustive : **aucun bitreader
n'écrit `+0x1f30`**. ⇒ le champ que le kill feed lit est reconstruit au replay (par résolution, cf. D).
Forme de la ref arme dans le descripteur : `param_3+0x18` = **datum-id de DÉFINITION (tag)**, résolu
`FUN_140583b80/1405839d0` (table tags `DAT_14494a908+0x78`, stride 0x34, FourCC `'weap'`→`'obje'`).

**(C) Pas de champ « arme reçue » prêt-à-lire côté victime.** Le death-recap `KillerWeapon`
(getter `FUN_14348977c`) lit un ViewModel RAM (`+0x1f8`) rempli à la mort = runtime, pas un record film.

## 7quater. CIBLE DE LECTURE OFFLINE (A3, 2026-06-07) — RÉSOLU : lire le WST i43 du biped tueur
> Mission A3 : confirmer PRÉCISÉMENT quel état du biped lire offline pour l'arme équipée. Établi par
> décompile HTTP-bridge (`http://127.0.0.1:8089`, le bridge UDS Windows ne se connecte pas) de
> `FUN_1407e00ac`, `FUN_140495abc`, `FUN_1408b555c`, `FUN_1407f06bc`, `FUN_14080dec4`, `FUN_140498500/898`.

**Côté JEU (replay), chaîne exacte de l'arme attribuée (asm `FUN_1407e00ac`) :**
1. `FUN_1408b555c` lit `*(rapport+0x538)` = **PlayerHandle TUEUR** → `local_res8`.
2. `lVar14 = FUN_140495abc(local_res8)` = **résolveur datum-handle générique** (handle = `gen<<16|idx<<1|parité` ;
   table indexée par idx16, check génération) → **entité unité/biped TUEUR** (`local_f3d8`).
   Checks : `*(lVar14+0x1c)!=0` et `*(lVar14+0x20)==uVar9` (c'est bien l'unité visée).
3. `1407e0325: MOV EDI,[RDI + RAX*0x4 + 0x48]` ⇒ `uVar9 = *(uint*)(lVar14 + 0x48 + [param_3+8]*4)` =
   **slot d'inventaire RAM du biped tueur**, RAX=`*(param_3+8)`=index slot/cause-de-dégât. Chaque slot = un
   **handle d'entité-arme**.
4. `local_f420 = FUN_140477aa0(local_res8, 4)` (résout le handle avec type-mask 4 = entité-arme) puis validation
   `*(local_f420+0x2c) == *(param_3+0x18)` (la def `'weap'` du descripteur). La famille écrite `+0x1f30` = ce handle.

**Le tableau d'inventaire `biped+0x48` n'EST PAS sérialisé dans le film.** C'est un offset de la struct
RAM de l'unité VIVANTE (handles runtime d'entités-armes), peuplé en mémoire, jamais écrit par un bit-reader.
La variante équivalente `unité+0x764` (indexée par le slot équipé `unité+0x38c`, cf. `FUN_140498500/898`) est
aussi RAM-only. ⇒ **on ne peut PAS lire `+0x48`/`+0x764` offline directement.**

**MAIS le composant ECS biped i43..46 = `weapon-state-type-info` (= HELD WEAPON, 4 slots) EST sérialisé**
et porte EXACTEMENT la donnée d'arme. Déserialiseur `FUN_1407f06bc` (registre chunk_00 biped #35) :
```
lVar14 = recState[0x10] + 0x7f0 + (*(param_1+8) * 0x90)   // *(param_1+8)=slot d'arme, stride 0x90, 4 slots
gate = FUN_14080d69c(...)                                  // présence
if !gate: *(lVar14+8) = 0xffffffff                         // slot vide
else:
  FUN_14080dec4(param_2,"variant-name",lVar14+4)           // <-- R(32) variant-name = FAMILLE (high-32)
  *(lVar14+8) = FUN_14080d61c(lVar14, *(lVar14+4))         // = local-handle de l'entité-arme (GetLocalHandleFromGlobalId)
  ... (R12 ammo + dequants)
```
⇒ Le WST i43 porte LITTÉRALEMENT : (a) `+4` = **variant-name R(32) = la famille** (clé du high-32 de
`analysis.WeaponIDToName` ; ex `841ac5e5` = Gravity Hammer, partagé par toutes ses variantes) ; (b) `+8` =
le **local-handle de la même entité-arme** que le jeu résoudrait en suivant `+0x48`/`+0x764`. C'est la
convergence exacte : `+0x48[slot] → entité-arme → 'obje' → variant_name`  ≡  `WST i43 variant-name`.

**⇒ CIBLE OFFLINE RECOMMANDÉE = composant ECS `weapon-state-type-info` (biped #35, i43..46), champ
`variant-name` (R(32), lu par `ConsumeWeaponStateTypeInfo` → `HeldWeapon.VariantName`).** C'est DÉJÀ
implémenté (`filmdec/unit_weaponstate.go:690`, capturé par `traverse.go:337` dans `EntityTrace.HeldWeapon`).
Pas besoin de suivre les entités-armes ni leur owner : le biped expose sa propre arme équipée par slot.

**PIÈGES :**
- **Slot équipé ≠ premier slot.** Le jeu lit le slot `*(param_3+8)` (cause de dégât) ; le walk actuel capture
  le PREMIER WST présent (`traverse.go:337`). Pour un tueur portant 2+ armes équipées (4 slots WST), il faut
  lire le slot correspondant à l'arme active, pas systématiquement i43. Sélection du slot actif = `unité+0x38c`
  côté RAM (non sérialisé) → offline, prendre le slot WST présent le plus récent / le seul présent, ou croiser
  avec le `desired-weapon-set` i42 (`FUN_1406d01fc`). À valider sur les 12 kills CE (cas multi-armes).
- **WST = high-32 (famille) seulement.** Variante exacte (gravity vs antigrav) = low-32, absent du WST
  (le `+0x478` def runtime n'est pas sérialisé là). La famille suffit (but figé). 
- **Atteindre i43 exige le walk biped propre** : il faut consommer i18..i42 bit-exact (déjà porté) AVANT i43.
  Le résidu i63 (P1) est APRÈS l'arme (n'affecte pas la lecture du WST). Mêlée/équipement : la cause de dégât
  (`param_3+8`) peut pointer un slot équipement/grenade (i22 grenade-counts, i26 unit-equipment), pas une arme
  WST — gérer ces cas via la catégorie dead-state (mêlée/lancé) plutôt que le WST.

### ⇒ FIL OUVERT TRANCHÉ (2026-06-07, décompile `FUN_1407e00ac` + dispatch table)
**Décompile complète `FUN_1407e00ac`** : l'arme = `param_3+0x18` (datum-id de tag `'weap'`, résolu
`FUN_140583b80(param_3+0x18,0x77656170)` / `FUN_1405839d0`) VALIDÉE contre l'arme d'inventaire du tueur :
`uVar9 = *(uint*)(lVar14 + 0x48 + [param_3+8]*4)` puis check `*(int*)(local_f420+0x2c) == *(int*)(param_3+0x18)`
(`lVar14`=entité tueur `FUN_140495abc`, `param_3+8`=index slot/cause). La famille `[+0x1f30]=uVar24` = le
handle de cette entité-arme. ⇒ **le jeu attribue = l'arme ÉQUIPÉE du tueur dans le slot ayant causé le dégât**,
sa définition matchant le `'weap'` du descripteur.
**`param_3` = descripteur de dégât LIVE riche** (~0x320 o : hit-sections `+0xf8/+0x100` stride 0x18,
positions `+0x2a0`, direction `+0x318`, géométrie `SQRTPS`). Dispatché via table de handlers
(`1453fd2f8` / `143d0ad..`), PAS via le dispatcher d'events-film `FUN_140620564` (spawn/pickup/mode 0x02-0x3c)
→ c'est le **pipeline de dégât live** (rejoué depuis les états répliqués + fire-events), pas un record compact sérialisé.

**CONCLUSION DURE** : il n'existe **AUCUN champ `kill→arme` prêt-à-lire** dans le film (ni chunk_27, ni death-recap
ViewModel, ni event de dégât compact). La DONNÉE d'arme (définition `'weap'` = famille) EST partout dans le film
(loadouts keyframe, NEW records d'entités-armes `param_1[4]`, fire-events 64-bit, 885 littéraux). Mais
l'ASSOCIATION par kill est faite au replay par le pipeline de dégât (descripteur → inventaire tueur → entité-arme).

⇒ **La seule méthode offline FIDÈLE = répliquer le mécanisme du jeu** : suivre l'**arme équipée (définition) du
tueur** dans l'ECS (loadout + swaps) et la lire au kill (tueur+temps via chunk_27). C'est EXACT (résolution par
slot d'inventaire, comme le jeu), ≠ la corrélation fire-events 85-90 % (rejetée). Mécaniquement = « arme active du
tueur », que l'user a interdit ~10× sur la base de l'ANCIENNE méthode non fiable. **FORK PRODUIT à trancher par
l'user** (cf. réponse 2026-06-07) : (1) accepter cette méthode ECS exacte, ou (2) tenter la repro lourde du
pipeline projectile (positions — jugé absurde par user), ou (3) accepter la famille au niveau loadout/fenêtre.

**Rappel direction user (mémoire `project_kill_feed_frame_decoder`)** : INTERDIT « held weapon au tick » ET
« reconstruction ECS ». La voie autorisée = RE le MÉCANISME (fait) → trouver la DONNÉE de l'arme DANS LE FILM
(event de dégât sérialisé). La « Voie A held-weapon/ECS » du §5 reste donc subordonnée à l'échec de ce fil.

## 6bis. PLAN ENGAGÉ (2026-06-07) — finir le décodeur ECS jusqu'à la formule complète
> Décision user : s'engager sur le décodeur ECS structuré (la SEULE voie offline fiable ; le scan-par-ancre = v3
> rejeté). Documenter avancée/statut/résultats régulièrement (ce doc = registre vivant + thought_log).
> Le walk structuré FRANCHIT le mur delta-compression (suit chaque entité frame-à-frame) ; le scan ne le pouvait pas.

| Phase | Objectif | Critère de succès | Statut |
|---|---|---|---|
| **P1** | Walk biped complet : atteindre les 8 bipeds/frame (porter i54/i59 branches lourdes amont + résidu i63) | slot 519 clean ~90 %+ ; 8 bipeds atteints/frame | EN COURS |
| **P2** | Timeline arme active par joueur (loadout keyframe + swaps via WST) sur les 8 bipeds | arme active par joueur dans le temps, validée | À FAIRE |
| **P3** | Archétypes projectile/arme : décoder position + `'obje'` famille des entités-armes/projectiles | familles + trajectoires projectiles décodées | À FAIRE |
| **P4** | Positions attribuées par-entité via le walk (remplace le nuage keyframe non-attribué) | position par joueur/projectile au tick | À FAIRE |
| **P5** | Attribution source-de-dégât : hitscan=arme active ; projectile=projectile à la position de mort ; mêlée=arme mêlée+catégorie | famille par kill | À FAIRE |
| **P6** | Croiser chunk_27 + valider (CE 12 kills + narration) + productioniser (analysis→service→handler, append-only, capability) | kill feed nommé en prod | À FAIRE |

**Suivi détaillé par phase ci-dessous (mis à jour à chaque avancée).**

## 7bis. AVANCÉE étape 1 (2026-06-07) — i63 dispatch corrigé + cause résiduelle identifiée
- **BUG TROUVÉ + CORRIGÉ** : le dispatch de tags i63 (`FUN_141fd4814` puis `FUN_142ef01c4`) gère
  **12 tags (0–11)**, pas 6. Le port Go désyncait sur tags 6–11 (valides). Portés bit-exact (vérif Ghidra) :
  tag6=gate8+R32+R16 ; tag7=gate8+R3+R15 ; tag8=`FUN_1432026f4`+R4+R1 ; tag9/10=`FUN_1432026f4`
  (=R1[+R19]+R32+R4) ; tag11=gate8+R19+R2. tag≥12 = vrai chemin d'erreur (`FUN_142ef0538`).
  Fichier : `components_biped_ability.go`. Keyframe Hydra non régressé.
- **Résultat** : slot 519 clean 25 %→29 % (256→291). Amélioration réelle mais partielle.
- **CAUSE RÉSIDUELLE (sonde `tmp_i63tags`)** : ~49 % des records qui désync ont leur 1er tag déjà ≥12 =
  **désalignement EN AMONT d'i63**, PAS un tag i63. Suspect = branches lourdes data-dépendantes d'un
  composant amont (**i54** ctx+0x9d mid-mobility, **i59** tag==3 `FUN_142f25e90`) signalées non portées
  dans le HANDOFF. ⇒ pour finir le walk slot-519, porter ces branches (RE multi-fonctions). Note : ces
  branches n'affectent PAS la lecture du WST i43-46 (arme), qui est AVANT.

## 9. NOTES DE CÔTÉ — stats potentielles pour l'app (user 2026-06-07)
- **i63 `biped-action-component` = 12 sous-types d'action (tags 0–11)**. tag 0 = dominant (idle/commun) ;
  1/2/8/10 fréquents ; 6/7/11 plus rares. Chaque tag = un type d'action joueur répliqué (capacité,
  recharge, swap, mobilité…). Largeurs bit connues (portées) ; **sémantique exacte par tag = à RE** (noms
  de champs). POTENTIEL : mine de stats gameplay (actions/capacités par joueur dans le temps) — À REVISITER
  après l'arme. NE PAS bloquer l'arme là-dessus.
- ⚠️ CORRIGÉ par R3 (2026-06-07) : **EnumA/EnumB ne sont PAS la méthode de mort**. Ce sont des datum-handles
  VICTIME(+0x04)/TUEUR(+0x08) (cf §10 ci-dessous). La « méthode » réfléchie du jeu (DamageType / killbreakdown)
  n'est PAS un champ propre du record dead-state film.

## 8. ANTI-DÉRIVE (auto-discipline — relire avant d'agir)
1. L'arme EST dans le film (885 littéraux + loadouts) — ne JAMAIS reconclure « pas là ».
2. NE PAS repartir sur fire-events / dead-state-GID.
3. slot→joueur = ACQUIS (chunk_27 b36/b37) ; largeurs runtime = PAS un mur (verdict L4) — ne pas re-présenter comme bloquants.
4. Relire CE doc + `thought_log` entrées kill-feed AVANT d'agir. Ne pas re-dériver l'acquis.
5. Tenir CE doc à jour à chaque avancée (c'est le registre vivant).
6. (R3) NE PAS re-traiter EnumA/EnumB comme un death-type/méthode — ce sont victime/tueur (datum-handles). §10.

## 10. R3 — DEAD-STATE : ENUM DU TYPE DE MORT (RE Ghidra complet, 2026-06-07)
> Mission R3 : décoder à fond le canal dead-state qui classerait melee/grenade/terrain.
> Établi par décompile HTTP-bridge (127.0.0.1:8089) de FUN_140c1dce0 (i11 wrapper), FUN_140c1dd44
> (heavy form), FUN_1407f2058/FUN_14049746c/FUN_140e958c4 (lecteurs EnumA/B), FUN_1406d84b4 +
> DAT_143cd8454/84b8 (champs +0x40/+0x44), FUN_14034f080 (enum DamageType réfléchi), FUN_14080d61c
> (ref +0x10). + run empirique `tmp_deathfield 26` (match 000d5950, 667 captures clean Mort==true).

### 10.1 STRUCTURE EXACTE du record dead-state désérialisé (param_1 = comp+0x74 ; i11 wrapper écrit comp+0x70=mort, comp+0xc4=R(1) si #35)
| off | lecteur Ghidra | largeur | sens (R3) |
|---|---|---|---|
| +0x00 | FUN_14080d69c | R(1);if1→R(32) | anim/handle |
| +0x04 | FUN_1407f2058→FUN_14049746c→FUN_140e958c4 | R(1)pres;if0→R(5)→datum-handle | **VICTIME (datum-handle world)** |
| +0x08 | idem | idem | **TUEUR (datum-handle world)** |
| +0x0c | inline (FUN_1406d310c(10)→largeur 4) | R(4) | petit champ (0..15) |
| +0x0e | inline | R(3) | petit champ (0..7) |
| +0x10 | ref : R(1)+R(1);if1→R(32) puis GetLocalHandleFromGlobalId (DAT_144eae7b8/DAT_144b404f0) | R(32) | ref tag damage-effect — **RÉFUTÉ arme (0/637, constante offline)** |
| +0x14 | FUN_140c1e31c | R(3) | sous-champ (présent si ref) |
| +0x18 | FUN_1406d1024 | R(1);if0→R(6) | sous-champ (présent si ref) |
| +0x38 | inline | R(4) | sous-champ |
| +0x3c | FUN_1407f1f24 | R(4) | sous-champ |
| +0x1e | FUN_1407f1e4c | R(1);if0→R(10) | sous-champ |
| +0x20..28 | FUN_14076dc04 (gated) | quant vec | **position de mort** |
| +0x40/+0x44 | FUN_1406d84b4(...,5) | **R(5)+R(5)=10 bits** déquant float | scalaires quantifiés (angle/force) — **PAS des enums, PAS 0-bit** (corrige note Go) |
| +0x4c | FUN_14080d69c | R(1);if1→R(32) | handle |

### 10.2 EnumA/EnumB = VICTIME/TUEUR, pas un death-type (preuve)
- Code : `iVar5 = FUN_140e958c4(local_res8, R5_index)` ; FUN_140e958c4 = packer de datum-handle world
  `(idx>>16<<15 | idx&0xffff)*2 | tls` (immediates {32,16,15}) = LE MÊME que le résolveur de local-handle
  d'entité-arme du WST (FUN_1407f06bc) et le tag-de-joueur. ⇒ +0x04/+0x08 référencent 2 entités du monde.
- Live-CE (RE_FINDINGS §1) : +0x04 = player-index VICTIME, +0x08 = player-index TUEUR, validé contre la narration.
- Le décodeur Go capture le **R(5) BRUT** (l'index 0..31), pas le handle résolu. C'est l'index-monde victime/tueur.

### 10.3 L'enum DamageType existe — mais c'est RUNTIME, pas dans le record film
- Enum réfléchi `DamageType` (registrar FUN_14034f080, strings @143c7e9d0..) : **None=1, Kinetic=2, Plasma=3,
  Hardlight=4, Shock=5, Power=6, Melee** (=0/7). + `DamageHitType_{Neutral/Friendly/Enemy}{Critical/Headshot/Hit}`.
- Catégories death-recap UI `killbreakdown_{weapon,grenade,melee,other}` (@143809ff8..) — labels PGCR localisés.
- AUCUN de ces enums n'est un champ sérialisé du record dead-state. La classe de dégât (et donc melee/grenade/
  terrain) est résolue au REPLAY depuis le DamageReport (param_3, FUN_1407e00ac), pas stockée par-kill dans le film.

### 10.4 EMPIRIQUE (tmp_deathfield 26) — les champs offline sont du BRUIT (walk désync)
- 18 dead-states appariés à des morts connues (±200ms, slot 519) : EnumA/EnumB balaient 0..28 SANS structure
  (un death-type aurait ≤7 valeurs ; victime/tueur ≤8). EnumA constant sur seulement 73/156 groupes-de-mort.
- GID +0x10 : ratio distinct/obs = 0.99 (bruit pur ; 0 hit catalogue). v0c/v0e dispersés 0..12/0..7.
- CAUSE : le walk biped désaligne le flux avant/pendant le dead-state (slot 519 ~29% clean ; §7bis i63+i54/i59).
  Ce N'EST PAS un défaut de structure du dead-state — c'est l'amont du walk. La structure (10.1) est juste ;
  les VALEURS offline ne le seront qu'avec un walk biped bit-exact (P1).

### 10.5 VERDICT R3 (classification du type de kill via dead-state)
**Le dead-state seul NE permet PAS, de façon fiable, de classer melee/grenade/tir/terrain — pour 2 raisons :**
1. **STRUCTUREL** : le record film ne porte AUCUN enum de death-type/damage-category prêt-à-lire. Les seuls
   « enums » (EnumA/EnumB) sont victime/tueur. Les petits champs +0xc/+0xe/+0x14/+0x18/+0x38/+0x3c restent
   non-mappés à une catégorie (le live-CE suggère que +0xc/+0xe combinés encodent melee=0x40000 vs lancé=0x10001
   dans la struct RAM, mais : (a) non confirmé sur le record sérialisé, (b) le CE a prouvé marteau≡marteau-antigrav
   et BR75≡sniper → le dead-state NE distingue PAS l'arme, au mieux la grosse catégorie). La granularité réelle du
   jeu (DamageType, killbreakdown weapon/grenade/melee/other) vient du DamageReport au replay, pas du film.
2. **EMPIRIQUE** : tant que le walk biped n'est pas bit-exact (P1 en cours), les champs dead-state captés offline
   sont du bruit (0% exploitable sur 000d5950).

**COUVERTURE des types via dead-state : NON fiable.** Au mieux, si P1 livre un walk propre ET si +0xc/+0xe
encodent bien melee-vs-lancé, on obtiendrait une catégorie GROSSIÈRE (mêlée vs distance), insuffisante pour
séparer grenade/tir/terrain et incapable de nommer l'arme. ⇒ le dead-state n'est PAS le canal du « type » fiable.
La voie qui reste cohérente avec le BUT (source de dégât par kill, tous types) = répliquer le DamageReport offline
(arme équipée WST i43 pour le tir, cf §7quater) ; le dead-state n'apporte qu'un modificateur secondaire éventuel
(headshot/perfect via DamageHitType), pas la source. À NE PAS sur-investir.

## ⚠️ RÉCONCILIATION GLYPHE vs KILL FEED (2026-06-07) — lever la confusion "on pouvait récupérer le glyphe"
Le **GLYPHE / l'arme (l'icône)** EST récupérable offline — c'est la FAMILLE d'arme, et on l'a : 519 records décodés
bit-exact (`+0x14`). Et les **KILLS** (tueur·victime·temps) sont récupérables (chunk_27, 93/93). On a donc les DEUX
moitiés du kill feed séparément.
**Ce qui N'EST PAS récupérable offline = LA JOINTURE** glyphe↔kill (« CE kill a CE glyphe »). Le jeu fait cette
jointure AU REPLAY, en RAM : l'`apply` (`FUN_1407e00ac`) écrit la famille sur le **record participant de la VICTIME**
(`report+0x1f30`) au moment du dégât létal, et le kill feed la lit. Or `+0x1f30` n'est **PAS sérialisé**
(`FUN_1406cfb28` ne désérialise que PlayerHandle/BotHandle). Le record de dégât n'a pas la victime ; le kill event
n'a pas l'arme ; rien ne les relie dans le flux.
⇒ **On peut produire la TIMELINE des armes (glyphes) ET la liste des kills, mais pas le kill feed labellisé
(X tue Y avec glyphe Z).** La jointure est l'opération runtime, absente du fichier. (Mon imprécision passée :
« on peut récupérer le glyphe » = VRAI pour les glyphes eux-mêmes ; je n'avais pas dit que la JOINTURE était le manquant.)
