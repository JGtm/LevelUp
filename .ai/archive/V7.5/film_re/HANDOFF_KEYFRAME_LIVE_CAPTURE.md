# HANDOFF — Cracker le deser KEYFRAME via capture live (débloque tous les sujets RE film)

> But : le keyframe (état complet initial) binde TOUTES les entités → le décodage séquentiel des
> deltas atteint tous les bipeds DENSÉMENT + fournit entité→player_index. Bloqué en statique
> (dispatch vtable indirect). Solution = **breakpoint live** pour révéler le frame-processor.

## ═══ RÉSOLU — le keyframe EST un flat NEW-loop séquentiel (CAPTURE LIVE, 2026-07-04) ═══

**Tranché par capture live** (hook CE Lua NON-bloquant sur `FUN_1406cbaa0`, `return 0` pour continuer —
`return 1` = break dans cette version de CE). Le keyframe est un **flux SÉQUENTIEL de records NEW à slots
CROISSANTS dans UN buffer partagé**. Toutes les théories antérieures (file par-entité, film-vs-.exe,
bitmask) sont RÉFUTÉES.

**Faits mesurés (ground truth)** :
- Reader = RBX à l'entrée de FUN_1406cbaa0 ; +0x08 data, +0x18 len, +0x2c bit-pos.
- record0 = NEW slot0 @bit20 ; record1 = NEW slot1 @bit45 ; **même buffer** (data ptr constant). Slots
  0,1,2,3… strictement croissants, bit-pos croissant.
- **Largeur d'un record = VARIABLE selon l'archétype** : slot0 (ti système)=7 bits, slot1=1257 bits, la
  plupart des ti=6 = 275 bits, etc. C'est ça qui bloquait le décodage offline (le port lisait un masque
  quand le gate=0 → sur-lecture).
- Le `ti` de chaque record = **R(6) lu dans le buffer au bit-pos** (juste après le header type+id).

**Décodage offline PROUVÉ** (`cmd/tmp_kfcapture` : croise `kf_capture.txt` frontières + `kf_slot0_live.bin`
buffer) : **205 entités keyframe décodées**, ti cohérents (5,6,14,17,38,42,43,45,47…), largeurs stables
par ti. Fixtures : `.ai/V7.5/dumps/kf_slot0_live.bin` + `kf_capture_sample.txt`.

**Pour les objectifs** : le keyframe INITIAL (frame 0) contient 41 entités positionnées (ti=38/43 =
object-position, positions incluses) mais **PAS les bipeds joueurs (ti=35)** — ils spawnent au début du
match (records NEW pendant la lecture). PROCHAIN : capturer un spawn biped en jeu actif (hook `biped_buf.bin`
qui dumpe le buffer au 1er NEW slot≥0x100) → décoder son default-state (position) → replay.

**Méthode de capture (reproductible)** : `open_process` + `enum_modules` (base) ; `evaluate_lua` :
`debug_setBreakpoint(base+0x6CBAA0, 1, bptExecute, cb)`, cb logge (RCX type, RDX id, RBX+0x2c bit-pos,
RBX+0x08 data) et `return 0`. Éviter les accents dans le `return` (CE renvoie en Latin-1 → erreur UTF-8 MCP).

## ═══ ÉTAT ACTUEL — le plus récent fait foi (2026-07-03, après 2 pairs .exe) ═══

**HONNÊTE : seul record0 décode. Le keyframe n'est PAS un flat NEW-record-loop.** Mon ex-conclusion
« record-loop confirmé, record0+record1 » était SUR-CONCLUE depuis record0 seul. Corrections autoritatives :

- **record0 = NEW objet slot0 ti=22** décode et matche la capture live (header + ti VALIDÉS). C'est réel.
- **Aucune largeur de record0 (0..300 bits) ne produit un record1 valide** (vérifié : E=77 est le seul
  candidat, et il donne ti=63 = garbage). Le +1 tail était une COÏNCIDENCE. Le flat NEW-loop est une IMPASSE
  au-delà de record0.
- **.exe DÉCISIF** : la table ECS a **exactement 50 archétypes objet** (ti 0..49) → R(6) ti≥50 impossible
  (garde-rail ajouté). ET `FUN_1406cd128` (delta-loop) est désactivée quand le gate keyframe `*(param_1+0x12)`
  est set → elle NE décode PAS le keyframe. Le keyframe passe par **FUN_142f2913c (baseline-emit)** qui draine
  une QUEUE par-entité (`param_1+0x1b320`), pas le paquet (cf. CAPTURE #2). Le buffer 11485 o a des runs
  `ff ff ff` = signature bitmask/skip, pas des headers de records.

**⇒ Le keyframe est vraisemblablement PRESENCE-WORD / BITMASK-DRIVEN** (mon hypothèse INITIALE, abandonnée à
tort après le 1er workflow). Encodeur miroir = **FUN_142f2e174** (vtable[0x10]) : itère la table d'entités par
blocs de 0x40 slots, mot de présence 64b `*(param_1+0x58 + (slot>>6)*8)`, skip 64 si 0.

**DEUX PISTES** :
- **(A) RE le framing bitmask** : décompiler FUN_142f2e174 (encodeur) → structure exacte du mot de présence +
  ordre d'itération → répliquer offline contre les runs ff/données de l'oracle. C'est le chemin pour cracker
  le keyframe proprement (snapshots denses périodiques de TOUS les joueurs). Un pair .exe propose de le RE.
- **(B) Fallback** : abandonner le keyframe, rester sur les deltas type-0 + inférence (`INFERRESYNC` atteint
  99/99 bipeds mais épars). Déjà fonctionnel, non-dense.

**ACQUIS RÉUTILISABLES (valides quel que soit le framing)** : largeurs quantif offline `min(26,6+L)` (câblé
`quantAxisWidth`) ; 50 archétypes objet + mapping ti→descripteur (.exe) ; garde-rail ti<50 ; physics-state
= R(32)+R(1)[+R(32)] ; corruption-guard per-composant-présent film-mode (R(1)[+R(32)], toggle
`filmComponentCorruptionCheck`) ; oracle `keyframe_buffer_live.bin`. Les sections « record-loop » ci-dessous
sont valides pour record0 mais NE généralisent PAS — lire cette section en priorité.

## ═══ REFRAMING MAJEUR — messages KIND-TAGGÉS, pas un flat object-loop (agent, 2026-07-03) ═══

**Le keyframe n'est PAS une boucle plate de records objet.** C'est un ensemble de **messages TAGGÉS PAR KIND**,
démultiplexés en buckets 0..14 (`FUN_142e33048`). Chaque kind a son propre lecteur (dispatchers
`FUN_142e2e6e4` sur `*record`, `FUN_141f86238` sur la kind-byte). Deux namespaces de records CLÉS :

| kind | record | header | table descr | payload |
|------|--------|--------|-------------|---------|
| **0x03** | **OBJET** | R(6) typeIndex **cappé < 0x32 (50)** + object-id (~13+2b) | `DAT_144e61d88 + 8` | vtable[0x60] |
| **0x01** | **CHAMP (field)** | R(7) field-index **< 0x7b (123)** + R(2) count | `DAT_144e61d88 + 0x210` | vtable[0x68] |

**LE « ti=63 » ÉTAIT UN MISREAD.** typeIndex objet est **cappé < 50** (hard gate du jeu). record0 = OBJET
slot0 ti=22 (valide, décodé, = capture live). Mais record1 lu en R(6)=**63 ≥ 50 = IMPOSSIBLE pour un objet**.
Donc record1 n'est PAS un record objet : soit (a) c'est un **record CHAMP (kind 0x01)** — R(7) index=63 est
VALIDE (<123) — que je feed à tort au lecteur objet (R(6)+table+8) → désync ; soit (b) **dérive du curseur**
dans le corps de record0 (physics-state ou masque mal porté), un motif mid-stream lu comme R(6)=63.

**⚠ CORRECTION** : mon ex-conclusion « chunk_00 incomplet pour ti≥50 » est FAUSSE. chunk_00 est COMPLET pour
les objets (0..49, cappé <50) ; ti=63 n'est pas un archétype manquant, c'est un field-index OU une dérive.

**PROCHAIN PAS (le vrai)** : résoudre le LAYOUT des messages kind-taggés dans le buffer keyframe — comment
objets (0x03) et champs (0x01) s'enchaînent (kind-byte ? buckets concaténés ? marqueur de fin de bucket ?).
Puis lire les records CHAMP avec R(7) field-index + R(2) + table `+0x210` + vtable[0x68]. Alternative si
(b) : re-auditer la largeur du corps de record0 (physics-state `FUN_142ed6c20` = R(32)+R(1)[+R(32)] ; masque ;
object-id ~13+2b). Le fait que R(6)=63 ≥ 50 est la PREUVE que ce n'est pas un record objet valide à cet offset.

**Spec ti=8 varint** (bonus agent, `FUN_142f14688`) : gate R(1) ; si 0 → R(8) (9b) ; si 1 → R(8) b0 ; si
b0≤1 → R(8) (17b) sinon R(16) → valeur 24b (25b). Utile pour porter le default-state de ti=8.

Artefacts : `FUN_142e2e6e4`+`FUN_141f86238` (dispatchers kind), `FUN_141f85fe0` (header objet R(6)),
`FUN_142e306d4` (header champ R(7)+R(2)), tables `DAT_144e61d88+8` (objets, cap 50) / `+0x210` (champs, cap 123).

## ═══ (obsolète, cf reframing ci-dessus) BLOCAGE ex-supposé chunk_00 ═══
Vérifié : parse chunk_00 = archétypes valides ti 0..49, vides ti 50+. Cohérent avec le cap objet <50 (pas un
manque). L'interprétation « registre incomplet » est remplacée par le reframing kind-taggé ci-dessus.

## ═══ KEYFRAME DÉCODE — record-loop confirmé BIT-EXACT (agent Ghidra + oracle, 2026-07-03) ═══

**Le keyframe DÉCODE.** C'est un RECORD-LOOP PLAT tout-NEW à slots CROISSANTS, avec un BIT TERMINAL par
record. Validé bit-exact contre l'oracle (`cmd/tmp_kfdecode`, tail=1) :
- **record0 = NEW slot0 ti=22 (physics-state), fin bit 77** (= prédiction agent).
- **record1 = NEW slot1 ti=63, propre, ascendant.**

**Découvertes clés (agent Ghidra a9d98…, résout la structure du default-state)** :
- **ti=22 default-state = 0 bit** : stub partagé `vtable[0x60] = 0x1408d8220` (`MOV AL,1 ; RET`, 0 bit lu).
  La MAJORITÉ des archétypes légers pointent vers ce stub → default-state=0 (le `Skip(0)` était déjà bon).
  Résolution STATIQUE `ti→descr→vtable[0x60]` : table descripteurs `DAT_144e61d88`, ti=22 → descr `0x14468e9f0`
  → vtable `0x1436fdb80` → slot 0x60 = stub. (biped ti=35 = FUN_140F44C38 spécial ; ti=5=R(6), ti=14=R(5),
  ti=37 = desers courts par-archétype, largeurs FIXES lisibles par décompile.)
- **+1 BIT TERMINAL par record NEW** (tail de FUN_1408f1aa4 après FUN_14076cb60, `SetNewRecordTailBits(1)`) :
  sans lui record0 finit bit 76 → record1 faux DELTA ; avec, record0 finit bit 77 → record1 = NEW slot1.
  CONFIRMÉ oracle. (PAS le corruption-check, qui donnait +33 avec guard=1.)

**MODÈLE CONFIRMÉ** : `[préambule 2 bits] puis répéter : header (type R(1);si0 R(2) + id R(13)+R(2)tag) +
R(6) typeIndex + default-state (vtable[0x60] : 0 si stub, deser court sinon) + gate R(1) + masque + composants
+ TAIL R(1)`. Slots CROISSANTS (all-NEW full-state), idLowBits=13, HasExtraFields=false.

**RESTE (itératif, FRAMEWORK PROUVÉ end-to-end)** : porter chaque archétype rencontré (default-state stub=0
ou deser court + composants) pour étendre la chaîne. Prochain = **ti=63** (record1, actuellement clean mais
sa largeur bloque record2). Chaque archétype porté = +1 record = +1 entité avec sa position (i0, largeur
`quantAxisWidth(L)`). Validation continue : `cmd/tmp_kfdecode` (chaîne de NEW à slots croissants, sans désync).
Descripteurs résolubles statiquement (Q2 agent) : `ti→DAT_144e61d88[ti]→vtable[0x60]`.

## ═══ PERCÉE — LES LARGEURS SONT OFFLINE (workflow type1-precision-table-re, 6 agents, 2026-07-03) ═══

**Renverse le "mur" supposé insoluble depuis des mois.** Verdict unanime (3 angles Ghidra + 2 offline +
synthèse) : les largeurs de quantification des default-states **NE VIENNENT PAS du film ni d'aucun chunk**.
Ce sont une **fonction pure fermée** :

```
axisW(L) = min(26, 6 + L)      où L = arch.Level(i) (= champ flags du registre chunk_00, DÉJÀ parsé offline)
```

Dérivée validée numériquement : `axisW = min(26, bitLen(ceil(40000/(2·step(L)))))`, `step(L)=2^(16-L)/120` →
donne exactement 6,7,8,…,14 pour L=0..8. Le half-extent 40000 = ±20000 (constante `.rdata` DAT_143b8c6b8,
lue byte-exact). L=0→6/6/6 confirmé live. **Le chunk type-1 (343Ko, ladder 0x414<<L) est REDONDANT** avec
la formule — pas besoin de le parser.

**Mécanisme (décompilé)** : au map-load `FUN_140be9890→FUN_140be9a14` calcule les largeurs via
`FUN_140be9b88(L, ranges)` (pas de lecture film). Lecteur vec3 `FUN_14076e524` : `R(1) gate ; si 0 → R(idxW)
index ; 3×R(axisW)` avec axisW LUE dans la LUT indexée par L (jamais dans le flux). Déquant :
`world = min + step·(q+0.5)`, `step=(max-min)/2^axisW`. **Résidu biped 96-bit "0x3FC,0"×4 = R(96) brut**
(3 float32 keep-baseline, largeur FIXE, trivialement offline).

**⚠ NUANCE HONNÊTE (map, pas film)** : les PLAGES min/max monde des positions ABSOLUES par-objet viennent
de la map (DAT_14462cbe0, sélectionnées par un index lu au flux). Mais pour un décodeur **bit-exact
(framing/désync)** seules les LARGEURS comptent → offline OK. Le biped mouvement (i0, L0) utilise la table
DÉFAUT (offline pur, ±20000). Les coordonnées monde absolues par-objet nécessiteront la plage par-map (une
fois par map, dérivable du build), pas le film.

**PROBES empiriques (cmd/tmp_kfdecode, contre l'oracle)** — le modèle record-loop simple est RÉFUTÉ :
- PROBE largeur : default-state à largeur CONSTANTE (Skip N) → meilleur = slots [0, 4487] garbage.
- PROBE 3 pas-constant : record k @ bit 2+k·W → meilleur W=1362 bits donne slots [0, 926, 4213] (pas
  croissants). ⇒ **records à largeur VARIABLE** (chaque archétype son default-state), pas de pas fixe ni de
  slots ascendants-par-1. Le default-state n'est PAS un `Skip` : c'est un deser STRUCTURÉ par-archétype. Pas
  de raccourci offline — il FAUT décoder le default-state de ti=22 (record0) bit-exact. C'est STEP 3, dont
  la structure exacte est en cours de RE (agent Ghidra : générique vs par-archétype, résolution ti→vtable[0x60]).

**PLAN D'IMPLÉMENTATION (4 étapes, 1 branche N commits, validé contre l'oracle à chaque pas)** :
1. **Fondation** : helper unique `quantAxisWidth(L) = min(26, 6+int(L))`, câblé partout où `6+level` est
   inline (traverse.go consumeQuantVec3, components_movement.go) + garde-rail grep (règle CLAUDE #6).
2. **Résidu biped** : compléter `consumeBipedDefaultState` — le bloc 96-bit "0x3FC,0"×4 = quat/vec3 quantifié
   L=5 (3×R(11)) + défaut i0 (L0, 3×R(6)) ; supprimer le résidu-skip.
3. **Générique non-biped** : `consumeDefaultState(arch, br)` = default-mask OR presence-mask, chaque composant
   via consumeByName + largeurs quantAxisWidth(Level(i)). Router TraverseEntity vers lui (au lieu de Skip(0)).
   ti=22 (physics-state) d'abord = débloque record0/record1 de l'oracle.
4. **Validation** : à chaque pas, `tmp_kfdecode` sur l'oracle → exiger record1 sans désync + N NEW slots
   croissants. NE PAS valider au clean-frame count (piège faux-propres, recette §8).

**FONCTIONS CLÉS** : `FUN_140be9b88` (formule largeur), `FUN_140be9c78` (step=2^(16-L)/120, C=1/120 @143cd975c),
`FUN_14076e524` (lecteur vec3 quantifié), `FUN_140be9a14` (populate map-load), `FUN_1406cfe44` (deser i0 biped),
range défaut ±20000 @143b8c6b8. Sortie brute complète : `tasks/wmhz9v48q.output`.

## ═══ FRAMING RÉSOLU (workflow keyframe-deser-re, 7 agents, 2026-07-03) ═══

**Hypothèse bitmask-driven RÉFUTÉE.** Décompilation parallèle Ghidra (FUN_1406cbaa0 dispatch,
FUN_1406cd128 boucle, FUN_142f2913c baseline-emit, FUN_1406cf008 ReadBit, FUN_142f2f73c applier) +
analyse binaire de l'oracle. Le keyframe utilise **la MÊME grammaire de records que les deltas** :

```
[préambule 2 bits]
répéter par record :
  type   = R(1) ; si 1 -> DELTA. sinon R(2) in {0=FIN,1=NEW,2=DEL,3=DELTA}
  id     : low = R(idLowBits) + idBase ; tag = R(2) en bits 30-31 ; slot = id & 0x3fffffff
  si NEW : R(6) typeIndex + default-state(vtable[0x60] archétype) + gate R(1) + masque + composants
  si DEL : R(32)
  si DELTA : masque + composants (archétype/base depuis le World)
```

**⭐ ANCRE BIT-EXACTE (validée contre l'oracle par `cmd/tmp_kfdecode`)** : démarrer au **bit 2**
(préambule) + **idLowBits=13** → record0 = **NEW slot0 id0x40000000, fin bit20**, exactement la capture
live (RCX=1 NEW, RDX=0x40000000, reader+0x2c=20). Aucun autre (skip,idLow) ne reproduit ce triplet.
`idLowBits=13` = FUN_1406d310c(DAT_144706100=0x1FFF), PAS 11. `HasExtraFields=false`. Mon sweep précédent
testait skip=2 mais avec idLow=11 seulement — d'où le garbage ti=21.

**⛔ LE VRAI MUR (identique aux deltas, le keyframe NE le contourne PAS)** : record1 désync
immédiatement. Cause prouvée : record0 est ti=**22** (physics-state-component, PAS biped 35) ; dans
`TraverseEntity`, hors biped, le default-state = `Skip(defaultStateBits=0)` → largeur de record0 FAUSSE →
record1 démarre décalé. Il faut le **default-state bit-exact par archétype non-biped** (vtable[0x60] du
descripteur). De plus, les largeurs de champs sont des **précisions runtime** (DAT_1445cc9e0/DAT_144632be0/
DAT_145121140, peuplées au map-load depuis la table **type-1** 343Ko, séquence doublante 0x414/0x828/0x1050
= step(L)=2^(16-L)/120 ; lues 0 en statique → **origine des runs FF**). C'est le mur documenté depuis des
mois ("même mur que les deltas"), désormais CONFIRMÉ pour le keyframe.

**FONCTIONS CLÉS (décompilées, image_base 0x140000000)** :
- Dispatch NEW : FUN_1406cbaa0 → **FUN_1408f1aa4** : R(6) typeIndex → descr = `*(*(param_1+0x18)+8+ti*8)` →
  `(**(descr+0x60))(descr,cnt,buf,reader,1)` = default-state → **FUN_14076cb60** = masque+composants.
- id-reader : **FUN_1406d3140**(_,reader,7,out) : idLowBits = FUN_1406d310c(largeur) ; +R(2) tag.
- ENCODE miroir (candidat VOIE 1 presence-word) : **FUN_142f2e174** (vtable[0x10]) : itère la table
  d'entités par blocs de 0x40 slots, mot de présence 64b `*(param+0x58+(slot>>6)*8)`, skip 64 si 0.
- config bit : DAT_144706104 = R(1) lu par FUN_142987460 en tête de frame ; pilote la largeur d'id.

**PROCHAINE BRIQUE (bien définie, = "décodeur universel")** :
1. Parser la table **type-1** (343Ko, doublante) → largeurs de quantification runtime par niveau L.
   Formule déjà dérivée : `reference_filmdec_quant_width_formula` (step(L)=2^(16-L)/120,
   width=min(26,bitLen(ceil(40000/(2·step))))).
2. Porter vtable[0x60] par archétype non-biped (ti=22 physics-state d'abord = débloque record0/record1).
3. Puis le record-loop décode densément (valider : record1 ne désync plus, N NEW slots croissants).
Toggle offline : `filmComponentCorruptionCheck=true` (mode film, +R(1)[+R(32) sentinel 0xbcddcba]/composant).

## ═══ CAPTURE #2 — BUFFER KEYFRAME DÉTERMINISTE DUMPÉ (2026-07-03, DÉCISIF) ═══

**Méthode** : 2e rechargement du film, BP haltant CE sur `HaloInfinite.exe+6CBAA0` (FUN_1406cbaa0),
figé au 1er record du keyframe. Base ASLR CHANGÉE → `0x7FF63AC90000` (le module+offset a tenu).

**Capturé (jeu figé, 2e lancement)** :
- RIP=`0x7FF63B35BAA0` = base+0x6CBAA0 = FUN_1406cbaa0 ✓. RCX=1 (**NEW**), RDX=0x40000000 (**slot 0**).
- Reader (arg6=RBX) @`0x23936F9EC40` : data@`0x23A33980000`, longueur `0x2CDD=`**11485 o** (IDENTIQUE
  capture #1), bit-pos +0x2c=**20**, byte-ptr +0x40=data+8. Chaîne d'appels (stack walk) : FUN_1406cbaa0
  ← FUN_1406cd128 (RSP+0) ← FUN_142f2913c baseline-emit (RSP+0x60) ← … ← **FUN_142987460** frame-proc
  (RSP+0x1C8) → confirme la même architecture 3-vues que les deltas.

**⭐ FAIT DÉCISIF — le buffer est DÉTERMINISTE** : dump des 11485 o (`write_region_to_file`) →
`.ai/V7.5/dumps/keyframe_buffer_live.bin`. **Byte-for-byte IDENTIQUE** à la capture #1 (autre lancement,
autre base ASLR) : `88 00 15 84 00 2C 54 0C 61 C9 00 0B FF FF FF FC 78 96 F6 C1 A8 1F FF FF FF E0 7F FF
FF FF C1 …`. ⇒ **le buffer est une fonction PURE du film, reproductible offline.** Ce n'est pas de l'état
runtime volatile. C'est un ORACLE bit-exact pour porter le deser keyframe.

**Décodage offline du buffer (`cmd/tmp_kfdecode`, sweep skip 0..32 × idLowBits 11/13)** : la boucle de
records DELTA ne le décode PAS (≤3 records puis désync, ex skip=2 → 1er record ti=21 flock-emitting =
garbage). ⇒ **framing du keyframe ≠ boucle type-prefix des deltas.**

**⭐ STRUCTURE DU BUFFER (lecture directe)** : longs runs `FF FF FF` (défaut/skip) ponctués de données
réelles (`78 96 F6 C1 A8 1F`, `FF FF FF FC 37`, …). Signature d'un encodage **SKIP/BITMASK-DRIVEN**
(la plupart des entités au défaut, quelques-unes portent un état), PAS un flux de records à préfixe de
type. C'est POURQUOI la delta-loop produit du garbage. En-tête = 12 o `88 00 15 84 00 2C 54 0C 61 C9 00 0B`
puis le bitstream masque+données.

**CONCLUSION CAPTURE #2** : le buffer keyframe (11485 o) est DÉTERMINISTE (crackable offline) et décodé
par le jeu via une boucle bitmask-driven (≠ delta-loop). L'oracle bit-exact est sauvegardé. La brique
manquante = RE du framing keyframe (itération bitmask / skip-run) + les default-states non-biped. 11485 o
= vraisemblablement UN sous-bloc (le type-2 fait 142695 o ≈ 12×) : capturer TOUS les buffers d'un keyframe.

## ═══ CAPTURE #1 DU KEYFRAME (2026-07-03, jeu rechargé, BP FUN_1406cbaa0) ═══

**Méthode** : user a posé un BP CE sur `HaloInfinite.exe+6CBAA0` (FUN_1406cbaa0 = dispatch record,
robuste ASLR via module+offset) puis rechargé le film → figé au 1er record du keyframe.

**Capturé (jeu figé)** :
- RCX=1 (**type=NEW**), RDX=0x40000000 (**id slot 0**) → le keyframe traite les entités depuis slot 0.
- `[RSP]`=0x1406CD334 = **DANS FUN_1406cd128** → le keyframe passe par LA MÊME boucle de records que
  les deltas (via FUN_142f2913c → FUN_1406cd128 → FUN_1406cbaa0). Pas de deser séparé.
- Reader (arg6) @0x1A671DCA560 : data@0x1A77C1E0000, byte-ptr +8, +0x2c=**20 bits** consommés au 1er
  record (préambule + type + id), longueur buffer 0x2CDD=**11485 octets**.

**⚠ NUANCE CLÉ (précisée par capture #2)** : le buffer reader (`88 00 15 84 00 2c 54 0c 61 c9 00 0b ff ff
ff fc 78 96 f6 c1 a8 1f ff ff ff e0 ...`) **NE MATCHE PAS** le payload type-2 brut du film (`a0 00 00 00
00 00 00 0b 7f ff ff ff 98 ...`, 142695 o). Mais il est DÉTERMINISTE (capture #2) ⇒ transformation
film→buffer déterministe (pas de l'état volatile). 11485 vs 142695 ⇒ le buffer = 1 sous-bloc du type-2.

## ═══ SYNTHÈSE COMPLÈTE (2026-07-03, après capture live + analyse chunk_02) ═══

**PERCÉE 1 — architecture du décodeur de réplication (RE'd, breakpoint haltant live)** :
- Frame-processor = **FUN_142987460** (capturé via `[RSP]` propre au hit de FUN_1406cd128, jeu figé).
  Traite un paquet type-0 ainsi : `R(1)` config (`DAT_144706104=FUN_1406cf008`) puis **boucle 3 VUES**
  (objet+0x228). Chaque vue = vtable[0x60] (**FUN_142f2913c** baseline-emit, draine une QUEUE
  par-entité @objet+0x1b320, SANS lire le paquet) + vtable[0x40] (**FUN_1406cd128** delta-loop, lit le
  paquet). Puis applique tous les records via vtable[0x48] (**FUN_142f2f73c** : NEW→FUN_142e32b24 bind,
  DEL→unbind, DELTA→FUN_142e32cc4 apply).
- Dispatch record universel = **FUN_1406cbaa0** (appelé par la delta-loop ET la baseline-emit).
- vtable classe décodeur @**0x1436A87E0** : +0x00 FUN_142f22f98, +0x10 FUN_142f2e174 (ENCODE snapshot :
  itère table entités stride 0xa0), +0x18 FUN_142f24a78, +0x40 FUN_1406cd128 (delta), +0x48 FUN_142f2f73c
  (applier), +0x60 FUN_142f2913c (baseline emit), +0x68 FUN_142f24fd8.
- **Table d'entités runtime** : indexée par slot (id&0x3fffffff), stride **0xa0**, typeIndex `short` @+2,
  id `u32` @+8 (source : FUN_1406cd128 `(id&0x3fffffff)*0xa0`).

**PERCÉE 2 — le binding est STATEFUL, pas un keyframe unique** :
- FUN_142f2913c dequeue des records par-entité (item+0x10=bitstream, +0x20=len, +0x28=id, +0x30=type)
  et les dispatch via FUN_1406cbaa0. La queue est alimentée DANS la delta-loop (FUN_142f29538 sous garde
  FUN_142f2b5c4). ⇒ le jeu baseline les entités nouvellement pertinentes EN CONTINU ; pas de keyframe
  unique à décoder. Décoder offline = maintenir l'état DEPUIS LE DÉBUT du film.
- Test multi-vue (`DecodeFrameViews`, `tmp_framedump mv`) : structure 3-vues RÉELLE mais vues 1/2
  quasi-vides, vue 0 désync (transients) ⇒ n'unlock PAS la densité.

**PERCÉE 3 — contenu de chunk_02 (analyse offline `cmd/tmp_kftable`)** :
- **type-1 @0 (343Ko)** = TABLE CONSTANTE (motif période 79 octets, séquence DOUBLANTE 0x414,0x828,
  0x1050,0x20a0,0x4140,0x8280…). **PAS la table d'entités** (0 biped ti=35). Candidat = table de
  précision/quantification (le doublement matche `step(L)=2^(16-L)`) — utile pour rendre le décodeur
  UNIVERSEL (largeurs actuellement hardcodées CE), à confirmer.
- **type-2 @343035 (142Ko)** = LE keyframe bitstream. Préambule `a0 00 00 00 00 00 00 0b | 7f ff ff ff
  (sentinel) | 98.. | 00 60.. | 67 ab d4 (id/hash?) | 2a 01.. | ...` puis blocs réguliers de 128 bits
  `0f ff ff ff f0 00.. ×N` (masques présence ?). Framing NON résolu (ni record-loop, ni les configs
  testées). **C'est le deser à cracker** (film-load capture).
- **type-8 (25Ko)** = PLAYER_METADATA (pi→xuid).

**CE QU'IL RESTE (2 voies, non exclusives)** :
1. **Cracker le deser type-2 (keyframe)** : breakpoint pendant le CHARGEMENT du film (pas la lecture) —
   le keyframe est traité une fois au démarrage. Poser un BP haltant sur FUN_1406cbaa0 (dispatch record
   universel) pendant le load → capturer les records du keyframe (id/type/reader) + le préambule. Le
   type-2 binde les entités INITIALES.
2. **Handler transients mid-match** : même avec le keyframe, les entités spawnées EN MATCH (projectiles,
   respawns) désync (NEW non-biped non porté). Porter les default-states/composants des archétypes
   transients (ti=0,2,5,14,37) = voie AUTONOME offline (oracle bipeds/frame).
- Interim utilisable : `tmp_traj INFERRESYNC=1` (99/99 bipeds atteints, épars) + mapping entité→joueur.

## MISE À JOUR — capture live (jeu lancé, 2026-07-03)

**Confirmé live (CE, film en lecture)** :
- Base ASLR : `HaloInfinite.exe` @ `0x7FF775050000`. `rt(FUN) = base + (FUN - 0x140000000)`.
  `rt_1406cd128 = 0x7FF77571D128` (vérifié : prologue `48 89 5C 24 10...push rbp/r12-15`).
- **vtable classe décodeur** = `0x1436A87E0` (statique) = `0x7FF7786F87E0` (runtime, lu dans R10 + à
  `[objet+0]`). Slots : +0x00 FUN_142f22f98, +0x08 FUN_14076c27c, +0x10 **FUN_142f2e174** (ENCODE
  snapshot : itère la table d'entités stride 0xa0), +0x18 FUN_142f24a78 (process 1 record →
  FUN_142f2cee0), +0x20 FUN_14076b9c8, +0x28 FUN_14076bd08, +0x30 FUN_1409c98c0, +0x38 FUN_14076b010,
  **+0x40 FUN_1406cd128 (delta loop)**, +0x48 FUN_142f2f73c (applier bind/unbind/apply), +0x50
  FUN_142f2f4a0, +0x58 FUN_140862664.
- **Objet décodeur** @runtime 0x1A76C870238 : `[+0]`=vtable, `[+0x11]`=1, **`[+0x12]`=0** (= gate delta,
  la boucle FUN_1406cd128 tourne). Pour le keyframe le gate serait ≠0 OU un autre objet/slot.
- Args au hit : RCX/RBX=objet(param_1), R8=reader(param_3), R10=vtable.

**⛔ BLOCAGE capture** : le frame-processor (appelant de la boucle) est **dispatché par trampoline JIT** —
  `[RSP]` au hit pointe une région heap zéroée (`0x1A7EC510002`), et un `set_breakpoint` LOGGING de CE a
  un `capture_stack` BUGGÉ (rend cette valeur garbage constante). Lire `[RSP]` après coup = STALE (le
  thread réutilise la stack : j'ai lu une frame `std::time_get` sans rapport, FUN_1409821c8 = FAUX
  frame-processor). **Il faut un breakpoint HALTANT** pour lire `[RSP]` proprement au hit.

**PROCHAIN PAS PRÉCIS (breakpoint haltant)** :
- Option A (MCP) : `debug_process` puis `debug_set_breakpoint_for_thread(thread, rt_1406cd128, execute)`.
  Problème : 112 threads, identifier le thread de décodage. Puis `debug_get_context` → RSP propre →
  read `[RSP]` = vrai return = frame-processor. `debug_continue`.
- Option B (CE GUI, film lancé) : poser un BP debugger sur `0x7FF77571D128`, au break lire la **call
  stack** / `[RSP]` → l'adresse de retour dans l'exe = le frame-processor. Convertir statique
  (`ret - 0x7FF775050000 + 0x140000000`) et me la donner → je décompile la branche keyframe.
- Une fois le frame-processor : y lire le dispatch par type de frame (type-2 → slot vtable keyframe) →
  décompiler ce deser → porter (préambule + boucle NEW → bind toutes entités + i9 player_index).

## Contexte RE (statique, acquis)

- **Boucle records DELTA** = `FUN_1406cd128` (base statique Ghidra 0x140000000). Module
  `replication_entity_manager_view.cpp` (string @143c99a50). Décode les records type-0 en structs
  de 0xc0 octets (param_5=array, param_6=count) ; l'applier `FUN_142f2f73c` les binde/applique.
- **GATE** : `cVar3 = *(param_1+0x12)` ; si ≠0 la boucle est SAUTÉE (return 2, 0 record). ⇒ le
  keyframe n'est PAS décodé par cette boucle = mécanisme séparé.
- **vtable classe décodeur** @1436a8800 : [+0x00 FUN_14076b9c8, +0x08 FUN_14076bd08,
  +0x10 FUN_1409c98c0, +0x18 FUN_14076b010, **+0x20 FUN_1406cd128 (delta loop)**,
  +0x28 FUN_142f2f73c (applier), +0x30 FUN_142f2f4a0, +0x38 FUN_140862664].
- **chunk_02** (keyframe) : frame type=1 @0 (343Ko, TABLE structurée LE 4-octets) + type=2 @343035
  (142Ko, bitstream, préambule `a0 00 00 00 00 00 00 0b 7f ff ff ff 98 ...`). Le préambule n'est PAS
  un record loop (sweep configs = échec).

## Procédure de capture live (jeu lancé, film en Théâtre)

**Étape 1 — révéler le frame-processor (l'appelant)** :
1. `base = getAddress("HaloInfinite.exe")` (ASLR).
2. `rt(FUN) = base + (FUN - 0x140000000)`. Ex `rt_1406cd128 = base + 0x6cd128`.
3. Breakpoint sur `rt_1406cd128`. À l'entrée, lire `[RSP]` = adresse de retour = **l'appelant**
   (le frame-processor qui dispatche par type). `static_caller = ret - base + 0x140000000`.
4. Décompiler `static_caller` dans Ghidra : y trouver le dispatch par type de frame (comparaison à
   1/2/...) et **la branche type-2 → deser keyframe**.

**Étape 2 — le deser keyframe** :
5. Breakpoint sur la fonction keyframe trouvée. Lire son arg reader (RDX/RCX) → `reader+0x40`
   (byte ptr) / `reader+0x2c` (bit pos) avant/après pour mesurer la conso du préambule + records.
6. Porter : structure du préambule (skip d'en-tête) puis boucle NEW → binder toutes entités +
   lire i9 `object-multiplayer-properties` (R(9) = player_index candidat) par biped.

**Étape 3 — attribution** :
7. Parser la frame type=8 (~25Ko, PLAYER_METADATA) → player_index→xuid (`b5>>4` + metadata,
   `.ai/PLAYER_INDEX_FIRE-EVENTS_RESOLUTION.md`). pi=0 = POV.

## Astuce si BP sur FUN_1406cd128 trop bruyant

La boucle delta est appelée à CHAQUE frame (60Hz). Pour isoler le keyframe : conditionner sur
`*(param_1+0x12)!=0` (le gate keyframe) OU breakpoint pendant le CHARGEMENT du film (le keyframe est
traité une fois au démarrage, avant les deltas). Alternative : BP sur l'applier `FUN_142f2f73c` avec
type=1 (NEW) en masse = signature du keyframe (tous NEW).

## ═══ PROCHAIN PAS RECOMMANDÉ (après capture #2 — oracle en main) ═══

On a un ORACLE bit-exact : `.ai/V7.5/dumps/keyframe_buffer_live.bin` (11485 o, déterministe). Deux voies :

**Voie K (cracker le keyframe, haute valeur = TOUS les joueurs périodiquement)** :
1. Décompiler la branche keyframe de `FUN_1406cd128` (le gate `*(param_1+0x12)` ≠0 = chemin keyframe) :
   trouver la boucle bitmask/skip-run qui itère la table d'entités et lit les présents. C'est un framing
   `[en-tête 12o][pour chaque slot : bit présence ; si présent, default-state via vtable[0x60]]`.
2. Répliquer offline contre l'oracle : `cmd/tmp_kfdecode` charge déjà le buffer. Y implémenter la boucle
   bitmask (pas la delta-loop) et valider bit-à-bit (les runs FF = slots absents, données = présents).
3. 11485 o = 1 sous-bloc → capturer les buffers SUIVANTS du même keyframe (continuer après le 1er record,
   re-BP, dumper chaque reader) pour couvrir les 142695 o du type-2.

**Voie T (autonome, sans keyframe — densité bipeds)** : porter les default-states des archétypes
transients (ti=0/2/5/14/37) via table ECS vtable[0x60] → NEW records décodent → binding → delta traverse.
Interim : `tmp_traj INFERRESYNC=1` (99/99 bipeds épars) + mapping entité→player_index (i9) → xuid (type-8).

## Outils
- CE MCP (`reference_cheatengine_mcp_setup`) : `open_process`, `set_breakpoint`, `debug_get_context`,
  `read_pointer`, `enum_modules`, **`write_region_to_file`** (dump buffer → fichier).
- Ghidra MCP : `decompile_function`, `read_memory`.
- Offline : **`cmd/tmp_kfdecode <buffer.bin> [filmDir]`** (décode l'oracle keyframe, sweep skip/idLowBits),
  `cmd/tmp_framedump kf` (dump préambule + sweep), `internal/analysis/filmdec/frame_debug.go`.
- **Oracle** : `.ai/V7.5/dumps/keyframe_buffer_live.bin` (buffer keyframe game-interne, déterministe, bit-exact).
- Une fois le deser porté : `DebugDecodeFrame` sur la frame type-2 doit donner N NEW records (≈ nb entités).
