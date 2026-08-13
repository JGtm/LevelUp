# Thought log — REPLAY 2D (dédié)

> Créé 2026-07-11 sur demande explicite de l'utilisateur : vérifier À FOND, empiriquement,
> l'état réel du décodage (offline vs CheatEngine) avant de proposer un plan. Consigner
> TOUTES les trouvailles ici. Ne rien affirmer qui ne soit vérifié sur pièces.
> Convention : **[VÉRIFIÉ]** = testé/lu sur pièces cette session · **[À VÉRIFIER]** = à
> confirmer · **[DOC]** = affirmé par la doc RE (à re-tester si décisif).

## Question centrale

Peut-on décoder les positions / trajectoires des joueurs **100% offline** depuis les chunks
de film téléchargés (sans aucune entrée CheatEngine : world_dump.txt, allbipeds_capture,
*_live.bin, prec_*.bin) ? Et à quelle couverture / qualité (keyframe niveau-match vs dense
par-joueur) ? L'utilisateur pense que le dense-offline « est déjà fait » — trancher.

## Ce qui est déjà établi (cette session, sur pièces)

- [VÉRIFIÉ] `cmd/tmp_kfmatch` (ce que j'ai utilisé pour la démo) prend en entrée
  `allbipeds_capture_sample.txt` (2000 lignes) — son en-tête dit « décode la CAPTURE LIVE
  des deltas biped … hook CE non-bloquant sur FUN_1406caad8 ». => la démo dense = CE.
- [VÉRIFIÉ] Le film 000d5950 = 28 chunks (0-27), ~508 s (8,5 min), `start_ms` réels par
  chunk (~20 s). En local on n'a que chunks 0-8 (~160 s). (manifest lu.)
- [VÉRIFIÉ] `internal/analysis/positions.DecodeKeyframePositions` = décode le motif comb
  dans les chunks TYPE_2 — n'utilise QUE les chunks (pas de CE). Match-level, StartMS réel.
  (positions.go lu.) Couverture/densité réelle = À MESURER.
- [DOC] `HANDOFF_ALL_PLAYERS_TRAJECTORIES.md` : section « ## Capture tous-joueurs (CE) » ;
  point 5 « 100% offline (sans CE) » listé comme RESTE À FAIRE.
- [DOC] thought_log 2026-07-10i : « décodage offline PLAFONNE au plancher de bruit (~5.7%)
  … mur = largeurs runtime des composants biped … POINT CE ».
- [VÉRIFIÉ] `cmd/tmp_worldreplay` décode des frames type-0 depuis un chunk MAIS charge
  `world_dump.txt` (dump debugger/CE, 250 entités) comme World de départ. => CE-dépendant.

## À vérifier empiriquement (en cours)

1. [ ] Inventaire EXHAUSTIF des outils qui décodent positions/trajectoires : entrées
   exactes → offline-pur vs CE-dépendant. Existe-t-il UN outil dense par-joueur SANS CE ?
2. [ ] Mesurer `DecodeKeyframePositions` sur 000d5950 (offline) : nb positions, par-chunk,
   couverture. = le socle offline réel.
3. [ ] Tester le frame-decoder depuis un chunk gameplay SANS world_dump (World construit
   depuis les records NEW du chunk) : couverture biped réelle. = le vrai test dense-offline.
4. [ ] Statut doc + git : un dense-offline (sans CE) a-t-il jamais été atteint ? Le mur
   « largeurs » est-il dérivable des FICHIERS (chunk_00 / module / formule step(L)) ?
5. [ ] Map de 000d5950 : map_id → module, extractible offline ?

## Trouvailles (remplies au fil de la vérif)

### T1 — [VÉRIFIÉ] Keyframe offline MARCHE (mesuré)
`go test ./internal/analysis/positions -v` → `TestDecodeKeyframePositions_Ref` PASS :
**76 positions** décodées sur 000d5950, chunks SEULS (aucun CE), bornes x[-6.3,34.8]
y[-24,24.5] z[-2.8,2]. => la voie offline-pure keyframe existe et fonctionne. Niveau-match
(pas d'attribution par-joueur), et repère float32 (échelle ~[-35,35]) DIFFÉRENT du repère
quantifié de la capture CE dense (~[448,581]) — 2 systèmes de coordonnées à réconcilier.
Densité réelle sur tout le film (toutes keyframes) = À MESURER (le test = sous-ensemble réf).

### T3 — [VÉRIFIÉ] Frame-decoder sur chunk : bipeds 100% clean, MAIS seul le POV est dense
`tmp_worldreplay 3 20` sur chunk_03 (World chargé depuis `world_dump.txt` = dump CE) :
- World = 250 entités, **bipeds 512-519 présents 8/8**.
- **COUVERTURE BIPED (tout le chunk, 1800 records)** : slot 512=42 clean 100%, 514=1 100%,
  515=29 100%, 516=2 100%, **519=1014 clean 100%**. => les bipeds qui apparaissent décodent
  **sans désync**. Le décodage des deltas biped N'EST PAS le mur.
- **MAIS distribution ULTRA-déséquilibrée** : slot 519 (le POV enregistreur) = 1014 records
  denses ; les 6-7 autres = 1-42 records épars. => depuis les **frames type-0 du chunk**, on
  a **1 joueur dense (POV) + les autres épars**. C'est l'ancien plafond « 2 joueurs ».
- Désyncs = 516/1800 (29%) mais **sur des entités NON-joueur** : typeIdx=0 i0
  `game-engine-team-mapping-component` (446), kill-volumes, navpoints… PAS les bipeds.

**CORRECTION de mon affirmation antérieure** : le « ~5,7% » venait du thought_log 07-10i qui
parlait du **dead-state (walk ECS kill-weapon)**, un AUTRE objectif. Pour les TRAJECTOIRES,
les bipeds décodent 100% clean. Le vrai obstacle all-8 dense-offline = la **densité des
proxies** (non-POV) dans les frames du chunk : le POV émet des positions denses, les proxies
émettent surtout du keep-baseline/prédiction (peu de positions absolues) — c'est ce que la
capture CE contournait en hookant le processeur COMMUN (voit les 12 bipeds denses).

Reste à trancher empiriquement : (a) le World est-il constructible offline depuis les records
NEW (au lieu de world_dump CE) ? (b) les positions denses des proxies sont-elles récupérables
offline (décoder le keep-baseline/prédiction) ou seulement via CE ?

### T4 — [VÉRIFIÉ agent doc/git, cité] Verdict statut offline vs CE
Sources : handoffs + git (186 commits `feat/filmdec-continuation`), citations file:line vérifiées.

1. **Keyframe offline (comb, `positions.DecodeKeyframePositions`)** : PUR offline, MAIS coarse
   (~2.9 pts/keyframe, 76 total / 26 keyframes sur 000d5950), niveau-match, ET **ne place PAS
   les 8 bipeds** (comb-273 = (0,0,0) pour eux) → HANDOFF_MAP_GEOMETRY:316-319, positions.go:14.
   => **CORRECTION de mon T1** : les 76 positions ne sont pas les joueurs. Le keyframe offline
   n'est PAS une source de replay joueur exploitable.
2. **Dense par-joueur depuis chunks SEULS** : NON. Seul le POV (519, 181 pts) est dense, et il
   utilisait le `world_dump` CE pour le binding slot→archétype. « **Décodage dense exige le
   world_dump CE** ; bootstrap World OFFLINE depuis keyframe NE marche PAS (`tmp_kf_world` : 0
   bipède sur catalyst) » → HANDOFF_MAP_GEOMETRY:209-211,228-229.
3. **Le « all-11 joueurs » (04/07)** = **capture CE live** des deltas (`allbipeds_capture`), PAS
   les chunks → thought_log:206-217. Le décodeur offline de CE flux atteint 7808/8000, mais
   l'entrée est CE.
4. **Rôles du CheatEngine** : (a) ORACLE build-constant réutilisable (table type→deser ECS ;
   buffer keyframe déterministe `keyframe_buffer_live.bin`) ; (b) **capture PAR-MATCH** = le
   World (slot→typeIndex, match-spécifique) = LA béquille ; (c) validation des largeurs.
5. **[IMPORTANT] Le mur des LARGEURS est RÉSOLU OFFLINE** : `axisW(L)=min(26,6+L)`, L = flags du
   **registre chunk_00** (déjà parsé offline). « LES LARGEURS SONT OFFLINE, fonction pure
   fermée » → HANDOFF_KEYFRAME_LIVE_CAPTURE:131-156, RECETTE:51-52. Reste seulement les PLAGES
   monde absolues par-objet = 1 valeur par-map, dérivable du build (pas du film, pas du CE).
6. **Carte** : résolue offline depuis `.module` (Kraken→walker→carte 2D). MAIS `map_id` DB
   (GUID UGC) ≠ GUID module → résolution par **codename** de dossier. **Cliffhanger (000d5950)
   PAS présente** (map saisonnière, lazy-download) → la charger 1× en jeu pour qu'elle
   apparaisse dans `deploy/ds/levels/multi/<codename>/` → HANDOFF_MAP_GEOMETRY:234-240.
7. **Le vrai jalon offline restant** = le **walk keyframe type-2 structuré** (binderait tous les
   bipeds densément offline) = « **BLOQUÉ AU BOOTSTRAP** » (0 record depuis bit 0) →
   HANDOFF_FRAME_DECODER_L3:383-387.

**VERDICT agent** : la croyance « dense offline déjà fait » = **CONFLATION**. Vrais acquis
offline = formule largeurs + géométrie carte + nuage keyframe coarse (non-joueur). Dense
par-joueur = CE. Pur-offline dense = RESTE À FAIRE (bloqué au bootstrap du walk keyframe).

### T5 — [VÉRIFIÉ empiriquement] Le bootstrap World offline = 0 bipède (LE mur)
- **Les 28 chunks (0-27) sont PRÉSENTS** localement (`ls chunk_*.bin` = 28). Mon « seulement
  0-8 » était un `ls | head` tronqué. **AUCUN téléchargement nécessaire** — film complet cache.
- `tmp_kf_world <dir>` (bootstrap World OFFLINE, sans world_dump CE) :
  - type-2 keyframe (138 Ko) : idLow 8-16 → **records=1, bound=0, bipeds=0**, désync à bit 11-19.
  - type-1 snapshot (343 Ko) : idLow 8-16 → **records=0** (rien).
  => **Confirmé : le World ne se construit PAS offline depuis les chunks.** 0 bipède. Le
  `DecodeFrameRecords` désync dès le 1er record du keyframe/snapshot (framing non résolu).
- `tmp_kfpos` (rendement keyframe offline, 26 keyframes) : **76 positions total, 2.9/keyframe,
  niveau-match**, et ne place pas les 8 joueurs (valeurs (−6.3,0,0)/(0,−15,0) = artefacts).
  => inexploitable comme source de replay joueur.

**Synthèse des mesures (T1+T3+T5) :** les deltas biped décodent 100% clean SI le World est
connu (T3), MAIS le World ne se bootstrappe pas offline (T5, 0 bipède) → il venait du
`world_dump` CE. Le keyframe comb offline (T1/kfpos) est coarse et non-joueur. Donc
**aucune source de positions JOUEUR n'est disponible en pur-offline aujourd'hui.**

### T6 — [VÉRIFIÉ agent inventaire, cité] AUCUN outil dense par-joueur sans CE (exhaustif)
Audit ~90 outils. **Tous** ceux qui sortent des trajectoires denses par-slot lisent un dump CE :
`world_dump*.txt` (tmp_traj:87, tmp_deadreckon:74, tmp_bipedharvest:78, tmp_rawpos:71,
tmp_mapdense:62, tmp_victimpos:84, tmp_poslink:216, +~35 diags) OU `allbipeds_capture.txt`
(tmp_kfmatch:92). **La prod `cmd/replay-build` → `replay/build.go:34` lit la capture CE** =
exactement la démo. Plafond offline = keyframe niveau-match (`positions`, table prod
`steps_shared_player_positions.go:11` : « v1 = MATCH-LEVEL, delta-compression bloque l'index »)
OU clusters sans identité (`tmp_bittraj`).
**Cœur du mur, cité** : `frame_records.go:667-674` — le décodeur PEUT binder depuis les records
NEW du chunk (`:675 w.BindFull`) mais c'est cassé offline : « opportunistic binding REGRESSED
clean frames 18060→6718 … a NEW reached after a false-clean is bit-misaligned … binds the slot
to a WRONG archetype ». => le `world_dump` CE est injecté PRÉCISÉMENT pour ça. Oracle offline
dispo : `keyframe_buffer_live.bin` (fonction pure du film) = référence bit-exacte pour porter le
deser SANS CE. Convergence TOTALE des 4 mesures (T2/T3/T5/T6 + doc T4).

## VERDICT (vérifié)

- La croyance « le dense-offline est déjà fait » = **FAUX, mais compréhensible (conflation)**.
  Ce qui est réellement offline : (1) formule des **largeurs** (`axisW=min(26,6+L)` depuis
  chunk_00), (2) **géométrie carte** depuis `.module`, (3) nuage keyframe **coarse non-joueur**.
- Le **replay dense par-joueur** que j'ai montré = **capture CheatEngine** (World par-match +
  flux deltas). CE écarté → cette donnée n'est pas rejouable en prod.
- Le **VRAI mur offline** = le **bootstrap du World** (framing du keyframe type-2 / snapshot
  type-1 pour énumérer (id, typeIndex) et binder les entités). Empiriquement 0 bipède
  aujourd'hui. Une fois ce mur tombé : largeurs OK + deltas biped clean (T3) ⇒ dense all-8
  offline devrait suivre. C'est un objectif RE **bien défini** (pas un flou), mais **non
  résolu** et incertain.
- Carte : offline OK, mais **Cliffhanger (000d5950) pas téléchargée** dans l'install (map
  saisonnière). Pour une démo carte : soit charger la map 1× en jeu, soit démo sur une map
  présente (catalyst).

Conséquence produit : **il n'y a pas de replay 2D pur-offline livrable aujourd'hui** — la
couche visu est prête, mais aucune SOURCE de positions joueur offline n'existe encore. Le
déblocage = crack du bootstrap World (RE).

## Verdict (après vérif)

(à remplir)

## [2026-07-11] JALON — Oracle CE capturé (le match le plus récent)

Session CE live (user a lancé Halo Theater + film du match le plus récent ; pont MCP
`ce_mcp_bridge.lua`). Capture via `tools/ce/filmdec_delta_capture.lua` (AOB code-cave sur le
site de dispatch de composants `0x76CD11` / FUN_14076cb60 ; aucune adresse en dur). Notes
opé : le pont MCP timeoute sur les calls LOURDES (scan AOB, dump) mais l'opé se termine
côté CE (injection + dump aboutis malgré l'erreur affichée) ; jeu vivant PID 30636 tout du
long ; injection code-cave = OK sur ce build/compte (pas de crash anti-cheat cette fois).

**Artefacts (offline, dans `.ai/re_dump/`)** :
- `ce_capture_delta.csv` — **807 855 records**, 20,6 Mo. Format `eid,typeIndex,compIndex,
  param4,bitCursor,skipCount`. Distribution ti : **35=503705 (biped, DENSE tous joueurs)**,
  37=123876, 42=80915, 41=42475, 43=28521, 4=20350, … = riche en archétypes.
- `ce_world_dump.txt` — reconstruit (`tmp_mkworld`) : **1754 entités, 74 bipeds (ti=35)**,
  0 slot ambigu. = LE World-bootstrap que l'offline ne produisait pas (tmp_kf_world=0).
- `ce_prec_ranges_14462cbe0.bin` (18432 o, plages monde par-index, idx0=bornes map),
  `ce_prec_widths_1445cc9e0.bin` (4096 o, largeurs par-axe). Tables de précision map-load.

Exemple largeur vérité-terrain : record biped, compIndex 0,1,5,25 → bitCursor 312,359,390,419
⇒ **i0 (position) = 47 bits** (3×AxisW15 + gate). Le CSV donne la largeur de CHAQUE (ti,comp).

**Généralisable ?** (question user) : le World capturé = ce match/map seulement (slots
runtime). MAIS ce qu'on en CRACKE = universel : framing des records NEW (constante build),
largeurs (formule `6+L` depuis chunk_00), desers par-archétype (table build finie). Une fois
l'offline validé bit-à-bit contre cet oracle → reconstruit le World depuis les chunks de tout
match, tous modes/cartes, zéro CE. Le CE = investissement UNIQUE. Nuances : desers d'archétypes
exotiques = petit follow-up ; plages absolues per-map dérivables 1×/map (pas CE, et inutiles au
rendu auto-fit).

**Prochain pas (crack, solo offline)** : (1) `tmp_compwidth`/`tmp_deltacal ingest ce_capture_
delta.csv` = largeurs vérité-terrain par archétype ; (2) obtenir les chunks du match capturé
(matchId à résoudre + `fetch_film_chunks` si absent) ; (3) décoder offline + diff vs oracle →
1er écart = le deser à corriger ; (4) itérer jusqu'à `tmp_kf_world` 0→≥8 bipeds (gate). Reste
NON garanti (le mur), mais désormais outillé d'un oracle riche.

## [2026-07-11] LEAD structurel — le keyframe type-2 = TABLE éparse, pas record-loop
`tmp_kfframe` sur 000d5950, payload type-2 (142695 o) : démarre par `a0 00 00 00` (= **0xA0=160**,
le stride/count de la table d'entités runtime, cf recipe §2), premières entrées pleines
(`67 ab d4 2a…`), puis motif `0f ff ff ff f0 00 00 00 00 00` qui RÉPÈTE. **62% des octets =
`0xff`** (slots vides sentinelle `0xffffffff`), 18% = `0x00`. ⇒ HYPOTHÈSE FORTE : le keyframe
type-2 est une **TABLE d'entités éparse** (en-tête + slots stride ~0xa0, vide=`0xff`,
plein=id/typeIndex), PAS une boucle de records NEW/DELTA à plat. C'est pourquoi `tmp_kf_world`
(qui record-loope depuis bit 0) désync à bit ~11-19 immédiatement. VOIE de crack reframée :
**PARSER la table** (entrées non-`ff` = présentes → id+typeIndex = le World), valider contre
l'oracle `ce_world_dump.txt` (1754 entités). Le deser exact = Ghidra (vtable indirecte,
handoff FRAME_DECODER_L3 : record loop = replication_entity_manager_view ; à décompiler).
Type-1 snapshot (343019 o) = piste parallèle (roster initial). NB : à confirmer sur pièces
(hypothèse depuis le dump d'octets, pas encore le format d'entrée exact).

## [2026-07-11] 🎯 PERCÉE — bootstrap World OFFLINE cracké (cas de base), workflow ultracode

Workflow multi-agents `crack-keyframe-world-bootstrap` (12 agents, 1.24M tokens, 205 tool calls,
Ghidra live sur HaloInfinite 311104 fn). **Le mur "BLOQUÉ AU BOOTSTRAP" est franchi pour le
keyframe de début de match.**

**FORMAT cracké (le keyframe type-2 = bitstream, PAS une table ni une record-loop delta)** :
- 1 bit de préfixe (=1) à bit 0, puis enregistrements par entité en **ordre de slot CROISSANT**,
  un par entité PRÉSENTE (slots absents OMIS). Enregistrement = `[id:u32 MSB][typeIndex][état
  largeur variable]`. id = `gen<<30 | slot` = `0x40000000|slot` (gen==1 au spawn).
- Le fameux `a0 00 00 00` = `[préfixe 1][7 bits de tête de l'id 0x40000000]` — PAS un stride 0xA0.
- Sentinelle fin = id==0xFFFFFFFF (les 62% d'octets 0xff = padding de capacité en queue).

**VALIDÉ premier-main (`cmd/tmp_kftable`, vérifié moi-même + réimplém indépendante PowerShell)** :
sur 000d5950 chunk_02 type-2, OFFLINE zéro CE : **249/250 entités = MATCH, 0 MISMATCH,
8/8 bipeds (slots 512-519 = les 8 joueurs)**. 1 manquant (slot 1024) = spawn tardif hors snapshot.
+45 entités réelles au-delà de la capture CE (continuation propre). = SUR-ENSEMBLE de l'oracle.

**CAVEATS honnêtes (vérif adversariale = verdict PARTIEL) — à durcir avant "universel"** :
- Le ti est lu en 32-bit par le parseur ; la RE (décompile FUN_141f86704, la fonction MÊME de
  capture oracle) dit l'en-tête réel = `[id:32][champ:26=0][ti:R(6)]` = 64 bits. Le 32-bit
  "tombe juste" SEULEMENT parce que le champ 26-bit est nul au spawn. À corriger.
- `gen==1` codé en dur → valable seulement au début de match (slots datum non recyclés). Un
  keyframe mid-match (gen≥2) serait rejeté. À retirer.
- Le walk est HEURISTIQUE (auto-calibration de largeur + scanNext min-slot), pas la table
  bit-exacte `stateBits(ti)` (vtable[0x60] du descripteur d'archétype). 0 mismatch ici mais non
  prouvé robuste. Q1 ouverte : largeurs d'état par archétype (biped bit-exact ; autres empiriques).
- Testé sur 1 seul film/keyframe. PAS encore validé sur le grand oracle (1754 ent) ni un 2e film.

**Prochaines actions (durcissement → universel)** : (1) décompiler le VRAI consommateur type-2
(FUN_142987460 → branche keyframe *(mgr+0x12) → appelant de FUN_141f86704) + nommer le champ 26-bit
(gen/flags/présence) ; (2) parser l'en-tête réel `[id:32][26][ti:6]`, retirer gen==1, valider sur
(a) keyframe mid-match, (b) 2e film, (c) le monde 1754-ent `ce_world_dump` ; (3) remplacer scanNext
par `stateBits(ti)` (descripteurs vtable[0x60]) pour un walk déterministe. Fichier : `cmd/tmp_kftable`.

**CE QUE ÇA DÉBLOQUE** : World offline (ci-dessus) + deltas biped clean une fois World connu (T3) +
largeurs offline (formule) ⇒ le pipeline **dense-par-joueur 100% offline, universel, zéro CE** est
désormais À PORTÉE. La visu (faite) se rebranchera dessus (remplacer la capture CE dans replay-build).

## [2026-07-11] Workflow 2 (durcissement + positions) — SPLIT honnête : World SOLIDE, positions NON

Workflow `harden-offline-world-and-positions` (8 agents, 1.27M tokens, Ghidra live). La vérif
adversariale + la validation indépendante ont **attrapé une sur-affirmation** de l'agent d'implém.

**SOLIDE (validé sur oracle réel)** :
- En-tête keyframe type-2 confirmé bit-exact : `[id:32][field:26][ti:6]` = 64 bits (ti lu à q+58 ;
  le fait que 249/249 matchent à q+58 PROUVE le champ 26-bit — le vieux lecteur ti-32bit
  "tombait juste" car field26==0 au spawn). Résolution clé : keyframe type-2 (plat, sans opcodes)
  ≠ delta type-0 (opcodes, déjà porté frame_records.go) = 2 formats distincts (ne pas mélanger).
- `cmd/tmp_kfworldpos` (World durci) : 000d5950 = **249/250 MATCH, 0 mismatch, 8/8 bipeds
  512-519**, reproduit indépendamment. 0 modif du package filmdec. Mid-match (chunk_04/06) et 2e
  film 7344d24f PARSENT structurellement (8 bipeds ti=35).

**NON RÉSOLU / caveats (la vérif a raison)** :
- **Positions des 8 bipeds = ARTEFACT, PAS validées.** L'agent d'implém a annoncé "8 positions
  décodées, plausibles" ; débunké 3 fois : (1) AUCUNE vérité-terrain XYZ n'existe (world_dump =
  slot:ti seulement ; allbipeds_capture = bitstream brut sans XYZ + slots d'un AUTRE match) → la
  "comparaison de forme" était impossible/tautologique ; (2) l'offset i0 dans le default-state
  biped n'est PAS localisé bit-exact (traverse.go:788-800 : ~260/380 bits non résolus, largeurs
  runtime DAT_* nulles en statique) → l'outil a DEVINÉ l'offset par scan [120,340] maximisant
  "distinct" (raisonnement circulaire) ; (3) signatures d'artefact : offset DÉRIVE +182(spawn)→
  +177(mid) ; X s'EFFONDRE sur 1 valeur pour tous les bipeds en mid-match (impossible).
- Durcissement gen≥2 / field26≠0 : présent structurellement mais **jamais exercé** (toute la
  donnée = gen1/field26=0). Le walker garde même `field26==0` en dur (filtre ti à q+32). Non
  prouvé empiriquement.

**Prochaines actions (pour de VRAIES positions offline)** — 2 pistes :
- PISTE A (recommandée, sans nouveau CE) : brancher le World OFFLINE (tmp_kfworldpos) dans le
  décodeur de DELTAS (type-0), dont le deser i0 EST calibré (T3 : bipeds clean ; trajectoires
  lisses de la démo). Donne le POV dense + les autres épars (proxy keep-baseline = gap séparé).
- PISTE B (all-8 au keyframe) : (1) capturer un ORACLE XYZ (CE breakpoint FUN_1406cfe44 →
  slot→XYZ des 8 joueurs au keyframe) ; (2) remplacer le scan d'offset par le VRAI deser
  (TraverseEntity + posCaptureHook) validé contre l'oracle XYZ ; (3) résoudre les ~260 bits
  résiduels du default-state biped (FUN_140F44C38 vtable[0x60] + largeurs runtime via CE map-load).

BILAN : le **mur du bootstrap World est bien tombé** (qui = les 8 joueurs, offline, validé). Le
**où** (positions) reste à cracker proprement — surtout ne PAS re-présenter les positions actuelles
comme des données de replay (infalsifiables par construction). L'adversarial a fait son job.

## [2026-07-11] PISTE A — bootstrap World OFFLINE branché dans le décodeur de deltas = CE RETIRÉ (prouvé)

Workflow `wire-offline-world-into-delta-decoder` (4 agents, 425k tokens). Objectif : retirer la
dépendance au `world_dump` CE qui amorçait le décodeur de deltas (type-0), en la remplaçant par un
World reconstruit 100 % offline depuis le keyframe du film.

**LIVRÉ (code)** :
- `internal/analysis/filmdec/keyframe_world.go` (NOUVEAU, exporté) : `WalkKeyframeWorld([]byte)
  []KeyframeRec` (promotion FIDÈLE de la walkOffline durcie de tmp_kfworldpos) + `WorldFromKeyframe
  (reg, pay) *World` (NewWorld + BindFull((gen<<30)|slot, ti)). tmp_kfworldpos refactorisé pour
  l'utiliser (copie locale supprimée ; net = 2 copies avec tmp_kftable, conforme). build+vet OK.
- `cmd/tmp_offlinereplay/main.go` (NOUVEAU) : décode chunk_03 (1199 paquets type-0) DEUX fois —
  World offline (chunk_02 keyframe) vs World CE (world_dump) — et diffe.

**PROUVÉ (chiffres réels, film 000d5950)** :
- World offline (chunk_02) : 294 slots liés. Vs oracle CE : 249 MATCH ti, **0 conflit de ti**, 1
  CE-only (slot 1024 = le 250e que le walker rate), 45 offline-only (extras). Bipeds 512-519 = 8/8.
- Décode deltas : **1195/1199 paquets record-stream STRICTEMENT identiques** offline vs CE. Les 4
  divergents s'expliquent TOUS par la différence de binding (le 1 manquant + 3 des 45 extras),
  JAMAIS par la logique de décode. **Les 90 positions i0 bipeds décodées = 100 % identiques**.
- `ceInputsRequired = aucun` : le chemin offline ne lit que chunk_00/02/03 du film (verdict binding
  adversarial = CONFIRMED : keyframe_world.go n'a AUCUN import, incapable d'ouvrir un fichier).
- Non-régression : tmp_kfworldpos toujours 249/250 + 8/8 + 0 mismatch après refactor.

**CAVEATS HONNÊTES (verdicts PARTIAL)** :
- « traces identiques » est surénoncé : 4/1199 divergent (binding, pas décode). Réel = équivalence
  bit-exacte sur tout slot lié à l'identique.
- **Le décode delta n'est PAS propre dans l'absolu** : le chemin i0 quantifié sort des positions
  ABERRANTES pour certains bipeds (ex slot 512 ~1e28). C'est IDENTIQUE offline et CE (le pont ne
  dégrade rien) — mais ça veut dire qu'on n'a PAS encore des trajectoires propres pour les 8. Les
  90 i0 décodées sont ÉPARSES (les bipeds n'émettent pas i0 à chaque paquet). C'est le résidu T3 :
  le deser i0 delta a un souci de qualité/densité.
- Régression portabilité flaguée `kfArchMax=50` : vérifiée = FAUSSE alarme (50 = objectArchetypeCount,
  la BONNE borne per spec ; 118 dynamique était la borne morte). Commentaire enrichi (cross-titre =
  re-sourcer). Pas de changement comportemental.

**BILAN** : le **bootstrap est offline** (CE retiré, prouvé bit-exact) — c'est la fondation
nécessaire et la clôture de « le décodeur était déjà fait » (il était CE-amorcé). MAIS le **OÙ propre
et dense** reste ouvert : le deser i0 delta produit des positions éparses + partiellement aberrantes.
Prochain vrai crack = qualité/densité du décode de position (pas le bootstrap, désormais acquis).

## [2026-07-11] ORACLE DE POSITION CAPTURÉ (CE) — vérité-terrain falsifiable enfin obtenue

Deser i0 décompilé (`FUN_1406cfe44`) : 2 bits de contrôle → (0,0) ABSOLU (consumeAbsoluteWithGate),
(0,1) DELTA vs position précédente (`*param_4`), (1,·) reuse. La position reconstruite est écrite
dans l'objet entité : `obj = *(param_3+0x10)`, `[obj+4]=x [obj+8]=y [obj+0xC]=z` (floats). 2ᵉ vec3
en `obj+0x2a4` (vélocité/orient). ⇒ le décodeur DOIT maintenir la position précédente PAR entité et
accumuler les deltas sur un seed absolu — un delta sans seed correct explose (= le ~1e28 de piste A).

Outil : `tools/ce/filmdec_pos_capture.lua` (adapté du filmdec_delta_capture prouvé). Hook au SITE DE
DISPATCH unique `14076cd11` (AOB `44 89 6C 24 20 48 8B CB FF 50 28`, unique dans le module) : filtre
ASM typeIndex==35 & compIndex==0, garde null anti-crash, empile `(eid, curseur, x, y, z)`. eid = full
id → slot = eid & 0x3FFFFFFF.

RÉSULTAT : `.ai/re_dump/ce_pos_oracle.csv` = 104 860 records. **Offset CONFIRMÉ** par la trajectoire
du slot 525 : `(23.47,8.88)→(23.44,8.95)→(23.40,9.02)→(23.36,9.12)→(23.30,9.22)`, Z=−2.46 constant
= joueur marchant en ligne, lisse, sol plat. Vraies positions. Repère PARTAGÉ (8 joueurs étalés, pas
par-entité) mais PETIT (X~[-6..39], Y~[-22..25], Z~[-3..2]) ⇒ mon range offline (QuantRangeCliffhanger
[-973..179]) est FAUX ; l'oracle DÉFINIT les vraies valeurs, à recaler.

CAVEAT film : les slots vus (578-588 au début, 516-529 mid-match) ≠ 512-519 du world_dump 000d5950.
Le film chargé n'est probablement PAS 000d5950 → pour valider offline il faut les chunks de CE film
(à confirmer : id du match ; sinon DL). Hook laissé injecté (null-guardé) pour recapturer au besoin.

## [2026-07-11] Fix i0 accumulation — aberrant ÉLIMINÉ + coeur delta correct ; range ABSOLUE + mur i63 restent

Workflow implement+verify (4 agents). Fix appliqué dans filmdec (world.go Pos/PosValid + SetPos/PosOf/
ClearPos ; position_capture.go seedAbsolute/applyDelta/keepBaseline ; components_movement.go : keep=reuse
(plus de raw 96 bits), abs=seed, delta=prev+delta ; frame_records.go setAccumSlot).

PROUVÉ (vérif CONFIRMED) :
- Aberrant ~1e28 ÉLIMINÉ : 0/276 émissions >1e6, max|comp|=1036 borné. Le keep-baseline n'émet plus les
  96 bits bruts comme coordonnée. RÉSOLU.
- Bit-consumption STRICTEMENT préservée (audit bit-par-bit ; seul emitPos change) → 1195/1199 toujours,
  tmp_kfworldpos 249/250, go test filmdec vert. Zéro desync introduit.
- Coeur delta correct : deltas offline 100% sur la grille 0.0138 ; accumulation prev+delta fonctionne.
- Accumulation vit DANS filmdec (pas l'outil jetable).

RÉFUTÉ / reste cassé (vérif REFUTED/PARTIAL — honnête) :
- **Reconstruction spatiale FAUSSE** : ~53% des events i0 bipeds prennent le chemin ABSOLU qui déquantifie
  sur QuantRangeCliffhanger [-974..179] × AxisW=6 → seeds hors de la boîte oracle [-6..36] + téléportent
  (z σ 100-186, pas max 1567). L'échelle offline est ~25-50× trop grande (spans 10^3 vs oracle 10^1).
  z NON plat. → **calibration de la range ABSOLUE non résolue**.
- Le quantum 0.0138 est INJECTÉ (SetDeltaQuantum depuis l'oracle) côté delta, pas dérivé du décodage →
  la grille reconstruite complète est 0.069/0.041/0.028, pas 0.0138.
- ClearPos = CODE MORT (0 caller) ; le tag gen (FullID bits 30-31) est stocké mais jamais comparé ; le
  commentaire world.go:10-11 décrit une garde inexistante = DOC INVERSÉE. À corriger (règles projet).
- Mur de desync i63 (biped-action) : le décode complet desync tôt → ~1 record biped/paquet (épars).

HYPOTHÈSE FORTE pour la suite (lie les 2 murs) : l'AxisW du chemin ABSOLU vaut peut-être 14 (mesuré CE
3×14 sur predFlag==1), pas 6. Lire 6 au lieu de 14 (a) fausse l'échelle ET (b) désaligne le bitstream de
24 bits après chaque i0 absolu → pourrait CAUSER le mur i63. Fixer l'AxisW absolu (6→14, quantum centré
0.0138) pourrait cracker les DEUX murs (échelle + densité) d'un coup. À tester (sweep AxisW absolu +
mesurer si le décode franchit i63).

BILAN : la garbage est morte, le coeur delta est bon, mais les trajectoires propres+denses ne sont PAS
encore là — reste (A) calibrer la range/AxisW ABSOLU contre l'oracle, (B) franchir le mur i63. Ne PAS
présenter les trajectoires actuelles comme du replay (elles téléportent).

## [2026-07-11] Calibration chemin absolu i0 — ÉCHEC honnête + cause racine + HANDOFF

Workflow calibrate-absolute (3 agents). But : trouver AxisW/range du chemin absolu i0 qui met les
seeds dans la boîte oracle. RÉSULTAT : aucun combo ne marche. Verdicts adversariaux REFUTED/PARTIAL.

ÉTABLI (négatif utile) :
- Le chemin absolu lisait AxisW=6 alors que doc/CE disent 3×14=47 bits (contradiction dans le code).
  Knob `SetAbsoluteAxisW` ajouté (défaut legacy=6, préservé).
- Sweep {AxisW 6/10/12/14 × déquant centré/Cliffhanger} : AUCUN ne satisfait {seeds in-box ∧ z-plat ∧
  échelle 42/53/11}. centre/6 = dégénéré (collapse à ~0, blob 1.4×2.2×2.1 collé à l'origine, mean X≈-0.3
  vs oracle 19.7). centre/12 approche X/Y (42/49) mais Z=55 + 1/8 in-box. Cliffhanger = 10^3 (confirmé faux).
  L'échelle bascule de trop-grand (range) à trop-petit (centré) — jamais juste.
- **Le +182 offset i0 (du workflow keyframe débunké) est un FAUX POSITIF** : q14 bruts X=588-4195, or in-box
  exigerait q14 X∈[7757,10801]. Le gate-consensus ne pointe pas le vrai i0.
- **Le mur de desync i63 ne recule PAS** avec l'AxisW (7968/495 clean/desync STABLE sur les 8 combos). La
  position biped est répliquée en DELTA (absolu rare : 32 émissions/8 chunks). Le top desync =
  `game-engine-team-mapping-component` (ti=0, ×492), SANS rapport avec i0. → l'AxisW absolu n'est le levier
  NI de l'échelle NI du mur. Hypothèse AxisW=14→casse-le-mur : RÉFUTÉE.
- Quantum 0.0138 = confirmé par l'oracle (grille) mais RE-INJECTÉ côté décode (tautologie tant que le vrai
  offset/frame n'est pas trouvé).

CLEANUP fait (règles projet) : `World.ClearPos` (0 caller) + `slotState.FullID` (write-only) SUPPRIMÉS ;
doc inversée world.go:10-11 corrigée. Vérif CONFIRMED : pas de régression tmp_kfworldpos (249/250),
tests filmdec verts, pas de code mort neuf.

=== HANDOFF — crack restant (session dédiée) ===
OBJECTIF : trajectoires bipeds propres+denses OFFLINE reproduisant `ce_pos_oracle.csv`.
CE QUI EST FAIT : bootstrap World offline (WalkKeyframeWorld/WorldFromKeyframe, 249/250) ; CE retiré du
décodeur deltas (bit-exact) ; oracle capturé (46790 rec, quantum 0.0138, boîte X[-6..36]/Y[-25..27]/Z[-4..7]) ;
garbage ~1e28 tuée ; accumulation delta correcte (filmdec world.go Pos/PosValid + seedAbsolute/applyDelta/
keepBaseline). Knobs sweepables : SetAbsoluteAxisW/SetAbsDequantMode/SetDeltaQuantum + outils tmp_i0score/
tmp_absscore.
CE QUI BLOQUE (2 murs, plus profonds que l'AxisW) :
  (M1) TROUVER LE VRAI FRAME/OFFSET i0. Le +182 est faux. Piste : la position est DELTA-répliquée ; le seed
       absolu est rare et son offset/échelle exacts sont introuvables via gate-consensus. Il faut aligner le
       décode au bitstream via l'ORACLE (curseur bitCursor + valeurs) record-par-record, PAS deviner l'offset.
       Idée : utiliser l'oracle (curseur→xyz) pour localiser où le décode DOIT produire chaque xyz, et
       rétro-déduire le frame/offset + la forme de déquant. L'oracle n'est PAS quantifié à 0.0138 (résidus
       continus) → le vrai déquant est peut-être continu, pas une grille stricte.
  (M2) MUR DESYNC i63 = game-engine-team-mapping-component (ti=0). Non-biped mais bloque le walk dense. À RE
       séparément (son schéma de composants n'est pas porté). Sans lui, ~1 record biped/paquet (épars).
NOTE ORACLE : slots oracle (datum vivant 528-562) ≠ slots offline (film 512-519) → validation par FORME, pas
par slot id. Aucune comparaison per-entité décode-vs-oracle n'est encore établie.

BILAN HONNÊTE : énorme progrès d'infrastructure, mais le OÙ propre+dense n'est PAS atteint. C'est un chantier
RE profond (M1+M2), pas un quick fix — plusieurs attaques ciblées (keyframe default-state, delta accumulation,
calibration absolue) n'ont pas cracké. Ne PAS présenter les trajectoires actuelles comme du replay.

## [2026-07-11] TEST DÉCISIF float32-vs-oracle — représentation TRANCHÉE : HYBRIDE (ancres float32 + bulk quantifié)

Déclencheur : capture d'écran du viewer tiers "FilmShell" (dend, projet privé) montrant des positions
float32 par joueur/tick → hypothèse « positions = float32 bruts, facile à repérer ». Workflow
oracle-guided-float32-offset (3 agents) : scanner le film pour un triplet float32 == valeur oracle.

RÉSULTAT (falsifiable, prouvé) :
- **45 valeurs oracle distinctes SONT des float32 LITTLE-ENDIAN dans le film** (560 occurrences, résidus
  ~0.002-0.003, significativité par étroitesse des résidus). Ce sont des ANCRES (keyframe/objets statiques)
  = ce que FilmShell/positions.go voient. Storage = LE (le "41 89 19 5e" de FilmShell = notation hex, pas
  l'ordre mémoire).
- **MAIS recall = 45/23999 = 0.19%** : le GROS des positions par-tick n'est PAS un float32 brut. Offset non
  constant (bit-packé). Extraction float32 aveugle (même LE + bonne boîte oracle) = **CHAOS** (pas moyen
  2.34 vs oracle 0.043, 3302 téléports). Les anciens PNG (bitattrib=toile, fix=vide) = ce chaos.
- **Modèle RÉEL = HYBRIDE** : ancres float32 LE + **BULK QUANTIFIÉ** reconstruit par le jeu (= l'oracle).
  calib.txt : `object-position-component=45 bits` (≈15/axe), `object-position-dynamic-precision-component=47`.
  Les trajectoires denses viennent du DÉCODEUR QUANTIFIÉ, PAS d'un scan float32.

CONSÉQUENCE : mon approche ECS quantifiée était le BON niveau pour le dense — mais avec la MAUVAISE largeur.
J'ai utilisé AxisW=**6** et le sweep de calibration a testé {6,10,12,14} en **RATANT 15**. Or calib.txt +
HANDOFF disent **W=15** (largeur modale). Le quantum oracle 0.0138 × 2^15 ≈ 452 = range plausible par axe.
→ LEVIER CONCRET UNTRIED : décodeur i0 à **AxisW=15** + range calibrée sur l'oracle (quantum 0.0138).

MAIS : même W=15 → positions ÉPARSES (mur de desync i63 = game-engine-team-mapping-component, schéma non
porté, ~1 record biped/paquet). Le DENSE exige AUSSI de franchir ce mur. Donc 2 leviers restants :
(A) W=15 + range (scale, untried, prometteur) ; (B) desync i63 (densité, RE séparé).

Outils ajoutés : cmd/tmp_oraclematch (match de valeur), cmd/tmp_rawpos_oracle (scan petit-repère).
Caveat : l'oracle ce_pos_oracle.csv vient de ma capture CE (lua obj+4 = reconstruction jeu) ; la colonne
bitCursor est l'offset film, mais les valeurs sont la vérité runtime, pas un décode film circulaire.

## [2026-07-12] W=15 RÉFUTÉ définitivement — MAIS le mur est ENFIN localisé précisément

Workflow quantized-i0-width-sweep (3 agents). Sweep AxisW base {6,13,14,15,16} sur le chemin i0 DOMINANT
(TraversalPrecision, delta+absolu, pas juste l'override absolu). 3 preuves convergentes que la largeur
N'EST PAS le levier :
- **Mur INVARIANT à W** : clean/desync bipeds = 86019/66106 et histo DesyncAt BYTE-IDENTIQUES pour tout W.
- **Positions jamais à l'échelle** : W=6 = blob écrasé 2-3u (inBox 100% dégénéré), W≥13 = explosion 156→1776u.
- **FIT DIRECT (juge fiable) accablant** : le q brut du chemin absolu SATURE toute la largeur du champ
  (plein 14/15/16 bits) au lieu d'être borné à ~3000 crans (~12 bits utiles pour 42u à 0.0138). step_implique
  0.0005-0.005, JAMAIS 0.0138. → le q lu N'EST PAS la position, c'est du bruit de désync. Aucun (Min,step,W)
  ne reproduit l'oracle.

**LE MUR EST ENFIN NOMMÉ** (registre biped ti=35, avant i63) — 3 composants qui font désync AVANT que le
walk atteigne la vraie donnée dense du record biped :
- **i55 = biped-posture-physics-component** (NON PORTÉ → default case ported=false)
- **i57 = biped-spartan-ability-component** (NON PORTÉ)
- **i60 = simulation-state-component** (porté mais simStateComplete=false → désync propre)
Leur largeur ne dépend pas de i0. Tant que ces 3 schémas ne sont pas RE'd, le record biped désync et la
position dense n'est pas lisible via le walk ECS.

BILAN DÉFINITIF (session) : leviers exhaustés (scan float32=chaos hors ancres ; walk ECS quantifié W 6-16=
q bruit, mur invariant ; range absolue=échec). Représentation TRANCHÉE = hybride (ancres float32 LE rares +
bulk quantifié reconstruit par le jeu). « On le faisait avant » = jamais atteint offline : le dense venait
TOUJOURS de CE (qui lit la reconstruction du jeu, court-circuitant tout ça). Les ~18 PNG antérieurs = chaos/vide.

HANDOFF PRÉCIS pour le dense offline (chantier RE dédié) :
1. RE + porter les schémas de i55 (biped-posture-physics), i57 (biped-spartan-ability), i60 (simulation-state)
   — probablement via captures CE de leurs largeurs runtime (comme la position). C'est LE déblocage du walk.
2. Une fois le walk biped propre au-delà de i60/i63, relire i0 à son vrai offset et re-fitter la déquant
   contre l'oracle (le fit direct sera alors interprétable, q borné ~12 bits).
3. Alternative pragmatique si (1) trop lourd : replay COARSE via les ancres float32 (keyframe) + interpolation.

## [2026-07-12] Corrections utilisateur (2/3 justes) + refutation dense-float32

Workflow bitpacked-float32-in-biped-records (cadrage utilisateur : position = float32 LE bit-packé).

CE QUE L'UTILISATEUR A CORRIGÉ (j'avais tort) :
1. **ENDIANNESS = LE, pas BE.** `readRawVec3` (position_capture.go:236) lisait MSB-first (BE) → garbage.
   Preuve : tmp_oraclematch BE=0 match / LE=560 matches 3-axes (p~1e-10). C'était bien ma cause d'erreur.
2. **Composants RE'd jusqu'à ~i80 (pas "i55/i57/i60 non portés" — mon handoff précédent était FAUX).**
   Ils SONT portés bit-exact. i60=simulation-state a un FLAG `simStateComplete=false` (traverse.go:926,
   SetSimStateComplete) qui fait un DÉSYNC PROPRE VOULU (le handle-tail dépend de globals runtime + un
   prédicat float non résoluble offline). i56=spartan-ability-energy, i59=spartan-ability-non-predicted,
   i61=simulation-state-playback. Activer le flag ne recule PAS le desync (36878→36575 clean, pire).
   Et i0 (position) est décodé AVANT i60 → le desync i60 ne bloque PAS la position.

CE QUI RESTE REFUTÉ (hypothèse dense-float32) :
- AUCUN offset constant ne donne les 8 positions biped en float32 LE (keyframe ni gameplay). Les candidats
  8/8 in-box sont dégénérés (2 axes ≈0) ou une constante répétée. Le keyframe biped i0 est QUANTIFIÉ.
- Le float32 LE = ~45 ANCRES sparse (seeds/keep-baseline absolus, slots 529-543, matchés oracle tol 0.01,
  recall 0.19%, médiane 0 float32/frame en gameplay). Les "tracks" float32 = valeurs statiques répétées
  (baseline échoée, pas moyen 0.000), PAS du mouvement. PNG = quasi noir + segments épars.

MODÈLE QUI TIENT : sparse float32 LE SEEDS (anchors, corrects, LE) + DENSE = deltas QUANTIFIÉS accumulés
sur le seed. Le seed est maintenant solide (float32 LE). Le VRAI blocage = décoder les deltas quantifiés :
i0 EST atteint (avant i60) mais son q lu SATURE le champ = bruit ; aucun width {6..16}/range ne reproduit
l'oracle. Piste ouverte non testée : i0 est peut-être PRE-transform (relatif à un parent/origine) alors que
l'oracle obj+4 est POST-transform monde → décoder i0 correctement ne matcherait toujours pas sans la
transform (les ancres float32 statiques matchent car transform=identité pour les props immobiles).

BILAN : leviers activables exhaustés (float32 scan, width 6-16, range, LE seeds). Le dense reste NON craqué ;
c'est un vrai résidu RE (structure exacte du deser i0 dynamic-precision + éventuelle transform monde), pas
un simple paramètre. Le dense a TOUJOURS été fourni par CE (qui lit obj+4 post-transform).

## [2026-07-12] Port default-state biped — grammaire CARTOGRAPHIÉE (Ghidra), i0 pas encore localisable (variant block + widths runtime)

Workflow port-biped-defaultstate (Decompile→Port→Verify). Range CE confirmée comme scale-fix mais keyframe
REFUTED (aucun offset ne donne 8 spawns distincts non-dégénérés in-box ; les distinct=8 sont dégénérés :
+113 gradient X monotone Y/Z constants = compteur de payload ; +182 Z constant 7/8 ; +304 hors-boîte).

GRAMMAIRE FUN_140f44c38 (default-state biped) MAPPÉE bit par bit (Ghidra, adresses+largeurs) :
1.[1b]FUN_1406cf008 gate 2.si set [8b]version→uVar10 (défaut 0xd) 3.[1b]gate name 4.si !gate [32b]name-hash
(FUN_14080dec4) 5.si uVar10>10 ECS_ReadEntityRefIndex5[1b+5b cond]@1407f2058 6.**FUN_14080cfe8(stream) =
BLOC VARIANT/CUSTOMISATION VARIABLE (non fermé)** 7.[1b]+si set[6b] 8.[1b]flag 9.FUN_14080d69c[1b+32b cond]
@14080d69c 10-11.lookups[0b] 12.si index!=-1: FUN_14076e494=vec3 absolu (consumeAbsoluteWithGate) + EntityRef
13.**FUN_14076dc04=[19b]** scalaire (facing) @14076dc04 14.si uVar10>5 [1b] 15.si uVar10>=0xc FUN_14080d69c.

VERDICT i0 : la position movement per-frame est un serializer SÉPARÉ **APRÈS** FUN_140f44c38 (le vec3 de
FUN_14076e494 est GATÉ derrière un lookup d'index = ancre/spawn conditionnel, pas la position toujours-présente).

DEUX RÉSIDUS PRÉCIS qui bloquent la localisation de i0 :
- **(R1) FUN_14080cfe8 (bloc variant/apparence Spartan) = longueur VARIABLE non fermée** : ~32b + gate+32b +
  1b+18b + 2b + 5b + [3b compteur uVar7∈0..4]×(5b + FUN_14080d69c[1..33]) + cond ; sous-fns FUN_141fd72c0/
  FUN_14080d524/FUN_14080d4d0 non tracées. Tant que ce total n'est pas fermé, l'offset cumulé jusqu'à i0 est faux.
- **(R2) largeurs RUNTIME du vec (FUN_14076e524)** : sélecteur DAT_144632be0 + 3 largeurs d'axe (tables
  DAT_1445ccbe0/DAT_1445cc9e0) = globaux runtime, pas immédiats → à lire via CE (on a ce_prec_widths mais
  l'indexation exacte reste à figer).

PATTERN : RE bit-exact du record biped = multi-couche ; chaque couche crackée en révèle une autre. Vrais acquis
cette session : range CE (le scale, vrai bug), grammaire default-state mappée, endianness LE, flags/i80. Le
décode dense complet = fermer R1 (variant block, bounded : ~3 sous-fns) + R2 (widths runtime via CE) → puis i0
localisé + dequant connu (range CE + width 13) → positions. C'est un chantier RE dédié, pas 1 workflow.

## [2026-07-12] R1 FERMÉ bit-exact — mais 3 nouveaux résidus révélés (l'oignon confirmé)

Workflow close-variant-block (option 1 choisie par l'user). R1 (FUN_14080cfe8, bloc apparence Spartan)
FERMÉ bit-exact, 0 bit ouvert : séquence complète 15 champs mappée (FUN_141fd72c0=9b, FUN_14080d6f0=32b,
FUN_14080d524=1|14b, FUN_14080d4d0=1|19..32b, FUN_14080d69c=1|33b, ECS_ReadEntityRefIndex5=1|6b,
FUN_140cec0a0=1|9b, FUN_1406d84b4=14b). TOTAL(R1) = 56..381 bits (formule data-driven fermée). Grammaire
default-state COMPLÈTE portée (default_state.go étendu). Mesuré R1=152b (slot512)..184b (slot519).

MAIS keyframe REFUTED : offset i0 CALCULÉ (endBit 193691..213313, varie 224/230/256 par record = normal),
mais à cet endBit 5/8 lisent precHigh=1 (vecteur défaut=0,0,0), 2/8 idxSel=1 (off-map), 0/8 in-box. Donc
l'endBit ne pointe PAS sur le vrai gate i0 → la grammaire n'est PAS ENCORE bit-exacte.

3 RÉSIDUS RÉVÉLÉS (honnêtes) qui décalent l'endBit :
- (r-a) largeurs d'axe du movement i0 (FUN_140cc5128 / DAT_1445cc9e0) = RUNTIME (lues 0 en statique) → CE.
  [mais width=13 déjà déduit du quantum oracle → probablement pas le vrai blocage]
- (r-b) gate mediaFrame (bipedMediaFramePresent, défaut false) = STRUCTUREL, à trancher via Ghidra.
- (r-c) tail uVar10>=12 modélisé R(1) seul (calibré live à 166 bits) = STRUCTUREL, à fermer via Ghidra.
r-b et r-c décalent le curseur avant i0 → i0 mal localisé. Bornés (2 fns Ghidra) mais = encore une couche.

PATTERN CONFIRMÉ (3+ couches) : bootstrap→range→default-state→R1→(r-b,r-c,r-a). Chaque couche crackée en
révèle une autre. Ce n'est PLUS du chaos (grammaire ~95% mappée, principielle) mais c'est un CHANTIER RE
MULTI-SESSION, pas 1-2 workflows. Vrais acquis solides et bankables. Le décode dense complet = fermer r-b/r-c
(Ghidra) + r-a (CE) → i0 localisé → positions (range CE + width 13 déjà connus). À reprendre au propre.

## [2026-07-12] Movement serializer porté (has-comp+masque) — offset grammaire-based, mais 0/8 + 2 couches de plus

Workflow decompile-movement (option 2). Découverte précise : le port lisait i0 trop tôt car il SAUTAIT le
prologue du record-NEW. Chaîne réelle (FUN_1408f1aa4 dispatcher) : vtable[0x60]=rep(FUN_140f44c38) →
vtable[0xa0]=default-mask{i0} (RAM,0b) → **[P2] R(1) has-components** (FUN_1406cf008) → **FUN_14076cb60**
(boucle composants) qui lit **[M1] le MASQUE de présence FUN_1406d7610** (R(1) gate ; full R(64)=65b ;
sparse R(3)count+count×R(6)) → puis i0 (index 0, forcé par le default-mask). Donc offset(i0)=fin(rep)+1+bits_masque.
Fix appliqué (default_state.go : consumePresenceMask + has-comp gate + BipedMovementI0Bit). r-b confirmé =
gate d'ÉTAT DST (0 bit au keyframe), déjà modélisé. CE a donné r-a : idxW=DAT_144632be0=1, biais=DAT_143cd84b0=0.5,
widths=DAT_1445cc9e0=6+niveau (lus direct en RAM).

RÉSULTAT : offset i0 CALCULÉ (513/519 masque-sparse 4b → i0=endBit+5 ; les autres full 65b → endBit+66).
MAIS 0/8 in-box (X jusqu'à 68, Z[-76..+39] vs [-4,7]), gates i0 NON uniformes (precHigh {1,0,0,0,1,0,1,1}) =
un champ de spawns aurait gates uniformes (0,0,0). Balayage exhaustif : max 1/8 in-box, aucun offset consistant.

DEUX NOUVELLES COUCHES (l'oignon, 5+ profond) :
- **(r-d) la REP FUN_140f44c38 SOUS-LIT ~8 bits** : k-sweep montre que has-comp devient uniforme 8/8=1 + masque
  uniformément full SEULEMENT à endBit+8 (proba fortuite ~1/2^16 → structurel). = les ~260 bits de feuilles à
  largeur RUNTIME (DAT_* lus 0 en statique) mal dimensionnés. Non sourçable statiquement → CE per-map.
- **(r-e) modèle i0 au spawn incertain** : même à endBit+74, 0/8 ; precHigh non uniforme → soit range/aw/déquant
  pas l'encodage réel au spawn, soit i0-spawn n'est pas absolu (chemin keep-baseline/predicted, vtable[0x28]
  FUN_1406cfe44 vs vtable[0x30] FUN_14076e29c).

VERDICT PROCESSUS : la RE bit-exact du biped ne converge PAS en un nombre borné de workflows (couches :
rep→R1→prologue→masque→r-d rep-underread→r-e i0-model). Le pur-offline est bloqué sur des LARGEURS RUNTIME
(DAT_*=0 statique) → nécessite une calibration CE PAR MAP (pas par match), + résoudre le modèle i0-spawn.
Chantier RE multi-semaine, pas multi-workflow. Acquis solides bankés (grammaire ~97%, deser pinné, offset
grammaire-based). STOP recommandé.

## [2026-07-12] Delta accumulation gameplay — le VRAI blocage isolé : le DESYNC (pas l'encodage position)

Workflow gameplay-delta-accumulation. Le modèle delta est BON (consumePredictedDelta : Delta8=3×signed8×0.0138
ou DeltaAxis=3×R(6)×0.0138). Mais les VALEURS sont du BRUIT, et la cause est enfin nette :

**LE DÉCODEUR SÉQUENTIEL DÉSYNC APRÈS ~1 BIPED/FRAME.** L'oracle a ~8 biped-deltas/frame ; l'offline en
atteint 1 (slot 519 POV, 532 pts fragmentés en 158 sous-traj) et 7/8 bipeds (516/517/518 = 0 sample) ne
sont JAMAIS atteints. Les rares deltas lus viennent de records DÉJÀ bit-désalignés → payload aléatoire →
distribution inversée vs oracle (offline >12q=84% vs oracle 0.45% ; 1-4q offline 1.5-19% vs oracle 97.8%).
Le "sur-grille 100%" offline est TAUTOLOGIQUE (entier×quantum) — pas une preuve.

DIAGNOSTIC DÉFINITIF (confirmé sous tous les angles cette session) : le blocage N'EST PAS l'encodage de la
position (deltas, compris) NI la range (trouvée) NI le quantum (0.0138). C'est **l'ALIGNEMENT DU FLUX** : le
walk perd sa place en décodant les composants intermédiaires (i55 biped-posture-physics, i57 biped-spartan-
ability, i60 simulation-state à predicate runtime, etc.) → désync avant d'atteindre les autres bipeds.

+ SOUPÇON PLUS PROFOND (à re-vérifier) : le film est du POV du client enregistreur. Il a de la donnée DENSE
pour ses ~2 entités possédées (POV+bot) et SPARSE pour les 6 autres (proxies = keep-baseline/prédiction). CE
obtient les 8 denses car il lit la reconstruction du jeu (obj+4, positions PRÉDITES). Donc le dense-pour-les-8
pourrait ne PAS être dans le film (les proxies sont prédits par le moteur) — cf mémoire "seulement 2 joueurs =
réplication par ownership". Si vrai, le pur-offline dense-8-joueurs est FONDAMENTALEMENT limité (la donnée
n'y est pas), pas juste un problème de décodage.

CONCLUSION : le OÙ dense offline se réduit à 2 murs — (1) l'alignement du flux (désync sur composants non
portés / à state runtime) ; (2) possiblement l'ownership-sparsity (proxies non denses dans le film). CE
court-circuite les deux (lit la reconstruction jeu). Chantier RE profond, possiblement partiellement
impossible offline. Le modèle delta+range+quantum est acquis ; il ne sert qu'une fois l'alignement fiabilisé.

## [2026-07-12] Modèle user VALIDÉ (données denses+parallèles dans le film) + BUG de slot-space trouvé

Workflow parallel-idanchor (modèle user : parallèle, pas séquentiel). RÉSULTATS qui changent la donne :
- **VALIDÉ : la donnée position multi-joueurs EST dans le film, DENSE et PARALLÈLE.** L'oracle montre 32 slots
  × milliers de lectures i0 chacun (536:3681, 545:3719, 552:2794...), trajectoires lisses. PAS d'ownership-
  sparsity. Dans un même paquet, plusieurs bipeds à des bitCursors distincts = records parallèles. L'utilisateur
  a raison sur toute la ligne (parallèle, tout dans le film, pas de runtime/RAM).
- **L'ancrage-id parallèle marche** (TryDeltaAt bit-scan trouve chaque record biped indépendamment par id, zéro
  séquentiel). i0 = juste après en-tête court [type R(1)][id R(11)+tag R(2)][mask] ; offset(id→i0)=14+largeurMask
  (déterministe en décodant le mask, mode 48).
- **BUG MAJEUR TROUVÉ (mon erreur)** : en gameplay (frame type-0), les bipeds sont aux slots **528+** (frame-space),
  PAS 512-519 (= keyframe/world-space). ce_capture_delta.csv prouve ti=35 @ 528+. Je ciblais 512-519 en frame →
  desync 'pour rien'. tmp_bipedresync/mes décodes gameplay ciblaient les mauvais slots.

RESTE = CALIBRATION (pas l'oignon) : (1) cibler slots 528+ ; (2) caler le déser i0 (le précédent trouvait 80%
'absolu' = faux positifs, 0/3063 match) — vrai chemin (delta vs abs) + range CEBiped + offset 14+mask ; (3)
confirmation anti-faux-positif (clean + in-box + continuité) pour monter la couverture par-paquet vers 5-8/paquet.
Outil : cmd/tmp_parallelpos. Workflow correctif lancé (bons slots + calibration).

## [2026-07-25] PIERRE DE ROSETTE (idée utilisateur) — pont mémoire↔fichier ÉTABLI

Idée utilisateur : « on a un dump CE de la lecture en mémoire ; n'y a-t-il pas une pierre de Rosette pour
comprendre comment ce dump est transcrit dans le fichier film ? » → OUI, et ça marche.

OUTIL : `tools/ce/filmdec_pos_rosetta.lua` (hook au site de dispatch prouvé 14076cd11, filtre ASM
typeIndex==35 & compIndex==0). Capture par record 40 o : eid, bitCursor (*(reader+0x2c) = bits depuis le
début du PAYLOAD), x,y,z (obj+4/8/C = position reconstruite), **+ 16 octets bruts lus depuis
[bitreader+0x40] = pointeur d'octet courant DANS le buffer film inflaté** = la signature.

CAPTURE : `.ai/re_dump/ce_pos_rosetta.csv` = **55 100 records**, ~20 slots (516:5884, 525:5289, 517:5209,
529:4002, 512:3853, 528:3727…), positions non-nulles quasi partout. Film = 000d5950 (8 mars 2026,
Cliffhanger, Super Fiesta) — confirmé par le matching lui-même.

**VALIDÉ SUR PIÈCES** : les signatures se retrouvent LITTÉRALEMENT dans les chunks téléchargés (inflatés) :
  slot 513 (-4.3215,17.1076,-1.2180) → chunk01 octet 558202
  slot 516 (-3.7410,16.7046,-1.2180) → chunk01 octet 559462
  slot 517 (-3.6581,16.1072,-1.2180) → chunk01 octet 559984
  slot 518 (-4.0589,15.5375,-1.2180) → chunk01 octet 560434
⇒ pont **(octet du film) ↔ (slot joueur) ↔ (position exacte)** établi. Le lecteur prefetch par qword donc la
localisation est à ±8 octets = fenêtre de 64 bits (vs millions auparavant).

POURQUOI C'EST DÉCISIF : le seul verrou restant était le SEED/ATTRIBUTION (quel cluster de bits = quel
joueur ; la moitié des slots s'amorçaient sur un faux cluster). La Rosette le tranche par construction :
plus de devinette, on SAIT. Elle donne aussi le test le plus direct possible du déser (offsets connus +
valeurs connues → si le décode offline ne rend pas la bonne valeur, l'écart isole le paramètre fautif).

### RÉSULTAT DE L'EXPLOITATION — LE FORMAT EST CRAQUÉ (2026-07-25)

Verdicts : alignement **CONFIRMED**, décodeur **PARTIAL** (X/Y exacts, Z à corriger — voir plus bas).

**1. ALIGNEMENT — formule fermée, 0 exception**
- 54 798/54 799 signatures localisées (**100 %**), 54 785 uniques, en 1,2 s (index numpy prefixe-8o).
- `payload_start = sig_byte − 8·ceil(bitCursor/64)` — EXACT sur les 64 classes de `cursor%64`
  (le pointeur [bitreader+0x40] désigne le qword SUIVANT le mot courant).
- `bitCursor` = offset EN BITS depuis le 1er octet du payload, et pointe l'ENTRÉE de i0.
- 100 % des hits dans des paquets **type-0**. Chunks couverts : 01..08 (90 s de film).
- ⚠ Conteneur : chunks 01 et 27 sont zlib, **02..26 sont BRUTS** (non compressés).

**2. STRUCTURE DU RECORD BIPED — découverte, vérifiée 100 %**
```
[1]  préfixe type = 1 (DELTA)
[13] idLow = SLOT
[2]  tag = eid>>30
[1]  gate (=0 partout ; ABSENT du port Go decodeDelta)
[1]  maskSel (=0)
[3]  maskCount (2..7)
[6 × maskCount] indices de composants, croissants
puis les composants dans l'ordre ; i0 démarre IMMÉDIATEMENT après le masque
```
**LOI : gap(id → i0) = 21 + 6·maskCount** — vérifiée sur 100 % des records DELTA (les 25 exceptions
sont les NEW = 1re apparition d'un slot). Écarts observés : 33/39/45/51/57/63 = 21+6k uniquement.
⇒ **L'ATTRIBUTION EST RÉSOLUE** : le slot est lu en clair 21+6·maskCount bits avant la position.
Plus aucune heuristique de seed / corrélation de trajectoire. Zéro chevauchement entre records
(min écart fin_i0 → en-tête suivant = +13 bits sur 40 900 paires).
Largeurs de composants biped déduites (système de longueurs cohérent) : i0=44, c1=31, c5=29,
c21=25, c25=13, c32=9.

**3. DÉSER i0 — le paramètre qui était faux depuis le début**
Chemin pris par **100 %** des records : `bUsePred=0, bDelta=0, precHigh=0, idxSel=0, index R(1)=0`,
puis **3 × R(13)** ⇒ **i0 = 44 bits** (skip 5 + 39). Aucun predicted-delta, aucun keep-baseline.
⇒ **La largeur d'axe est 13** — ni 6 (TraversalPrecision actuel) ni 14 (commentaire
components_movement.go:207 « 3x14, i0=47 » = FAUX pour ce chemin). Sweep (skip 0..8)×(w 6..20) :
skip=5/w=13 donne |r| = 1.0000 sur les 3 axes ; skip=4/w=14 donne 0 % de match (erreur médiane 30 u).
Déquant confirmée : `v = min + step·(q+0.5)`, `step=(max−min)/2^w`, range QuantRangeCEBiped.
Reproduction vs vérité (54 758 paires) : X médiane 3e-5, Y 3e-5 ⇒ **exacts**.

**4. CORRECTION Z (relevée par le vérificateur — à appliquer)**
Z a un résidu **systématiquement bimodal ±0,0042 = ±S/4**, et `z_truth` tombe à 100 % sur le réseau
**S/2** (0 % sur S) ⇒ **Z est en width 14, pas 13**. Cohérent avec le quantum Z mesuré sur l'oracle
(0,0083 vs 0,0138 en X/Y ; 137,55/2^14 = 0,0084). ⇒ **AxisW = {13, 13, 14}**.
Leçon de méthode : la bonne métrique pour un champ quantifié est l'**égalité exacte de l'indice de
quantum**, pas une tolérance en unités monde (le seuil 0,05 u laissait passer un axe faux à 100 %).

**5. CORRECTIFS GO À APPLIQUER**
- `TraversalPrecision.AxisW = {13,13,14}` (traverse.go:67 vaut {6,6,6}) ou SetAbsoluteAxisW.
- `decodeDelta()` doit consommer **1 bit de gate** (toujours 0) entre l'id et consumeMask().
- Corriger le commentaire components_movement.go:207-208 (doc inversée).

**ARTEFACT** : `.ai/re_dump/rosetta_groundtruth.csv` — 54 785 lignes, 24 colonnes (chunk, packetIndex,
packetType, payloadStart, size, ts, payloadBitOffset, recordHeaderBit, gapBits, maskCount, maskIndices,
slot, eid, x/y/z_capture, x/y/z_decoded, x/y/z_truth, sigByteInChunk, residual). 27 slots, 7 714 paquets.
NB : `x_capture` = état AVANT le i0 de la ligne (hook pré-call) ⇒ la valeur PRODUITE par ce i0 est
`x_truth` (= capture du record suivant du même slot). 11,22 % de captures dupliquées (le moteur décode
certains paquets 2×) — 48 639 positions distinctes après dédup.

**LIMITES HONNÊTES**
- Mesuré sur UN film (000d5950) / UNE map ⇒ la range de déquant est per-film, à lire depuis la table
  du film pour un décodeur universel.
- Seul le chemin ABSOLU est validé (100 % des records ici) ; predicted-delta / keep-baseline / precHigh=1
  / idxSel=1 / handle-tail non exercés ⇒ ne pas appliquer AxisW=13 globalement sans garde.
- Ambiguïté [gate 1b + count 3b] vs [count 4b] non tranchée par les données (Ghidra dit R(3)).
- Largeurs des composants des archétypes NON-bipeds inconnues ⇒ rejouer un paquet COMPLET pas encore
  possible (mais inutile pour les positions joueurs : on saute de record biped en record biped).

Workflow d'exploitation lancé : index chunks → localisation signatures → alignement paquet
(packet_start ≈ sig_byte − cursor/8, vérifié contre les vrais en-têtes) → table de vérité
`.ai/re_dump/rosetta_groundtruth.csv` (chunk, paquet, bitOffset, slot, xyz) → mesure de l'écart id→i0
(constant ou fonction du masque = la règle d'attribution) → validation du déser aux offsets connus.

## ÉTAT (2026-07-11, fin de session) — récap rapide
- Visu replay : FAITE (prototype offline, non commité).
- Oracle CE : capturé (807k records + world_dump + tables) — dans `.ai/re_dump/`.
- **P1 bootstrap World offline : cracké CAS DE BASE** (`tmp_kftable` = 8/8 bipeds + 249/250 ent,
  zéro CE, vérifié). Reste : DURCIR (en-tête réel `[id:32][26][ti:6]`, retirer gen==1, stateBits,
  valider sur oracle 1754-ent + 2e film) → universel.
- **Workflow 2 lancé** (ultracode) : durcissement + WIRING World-offline → décodeur de deltas →
  trajectoires denses par-joueur (le payoff : prouver le pipeline replay 100% offline end-to-end).
- Ensuite : rebrancher `cmd/replay-build` sur la source offline (retirer la capture CE) → la visu
  faite tourne sur données offline universelles.

## Plan proposé (offline-pur — la seule voie ; passé plan-review)

Objectif : replay 2D **100% offline, universel** = positions joueurs depuis les chunks
téléchargés (présents), sans CE. La couche visu (canvas/endpoint/pipeline) est PRÊTE et
offline ; il manque UNE chose : la **source de positions joueur offline**.

Mur unique identifié (vérifié) = **le framing des records du keyframe type-2 / snapshot
type-1** (énumérer (id, typeIndex) pour binder le World + lire les positions full-state).
Tout le reste est acquis offline : largeurs (formule chunk_00), deltas biped clean une fois
le World connu (T3), géométrie carte (.module).

- **Phase 0 — Oracle (déjà acquis)** : largeurs offline + buffer keyframe déterministe
  (`keyframe_buffer_live.bin`, fonction pure du film) = oracle bit-exact pour valider le port
  SANS CE. C'est ce qui rend Phase 1 falsifiable offline.
- **Phase 1 — CRACK du framing keyframe type-2 (make-or-break)** : RE de la boucle de records
  du keyframe (Ghidra sur le deser keyframe + données film + oracle). Cible : énumérer les
  entités → binder le World offline → lire les positions full-state des **8 bipeds** à chaque
  keyframe (26 keyframes / 508 s). GATE : `tmp_kf_world` passe de **0 → ≥8 bipeds bindés** et
  positions des 8 joueurs non-nulles/plausibles à ≥1 keyframe (vs oracle).
  ⇒ Livrable si Phase 1 OK : **replay 2D offline coarse mais COMPLET** (8 joueurs, ~20s de
  résolution, temps réel via `start_ms`) + carte. Universel, zéro CE.
- **Phase 2 — Densification (deltas)** : World bootstrappé offline → chaîner les deltas type-0
  (décodent déjà clean, T3) pour densifier entre keyframes → trajectoires lisses all-8. GATE :
  couverture dense des 8 slots (pas que le POV) sans CE.
- **Phase 3 — Rebrancher la visu** : `cmd/replay-build` re-ciblé sur la source OFFLINE
  (Phase 1/2) au lieu de la capture CE ; artefact par-match généré offline. Endpoint + front
  déjà prêts (adapter le schéma si coarse/keyframe : points horodatés vs tracks denses).
- **Phase 4 — Carte** : résoudre le codename de la map du match ; Cliffhanger (000d5950) à
  charger 1× en jeu OU démo sur une map présente (catalyst) ; extraire géométrie + overlay.

RISQUE : Phase 1 EST le mur documenté « BLOQUÉ AU BOOTSTRAP ». Incertain — mais bien défini,
oracle dispo, et c'est la SEULE voie vers un replay offline. Reco : attaquer Phase 1 en effort
BORNÉ (RE ciblée, éventuellement workflow multi-agents Ghidra) ; si ça cède → replay offline
coarse livrable ; sinon → la visu (prête) reste parquée jusqu'au crack, et on ne livre RIEN de
CE. La démo CE actuelle = à considérer comme un PROTOTYPE de la visu, pas un livrable.

### Corrections plan-review (contraignantes)
- **Effort/risque par phase** : P1 = LOURD + INCERTAIN (le mur RE) ; P2 = moyen (deltas déjà
  clean) ; P3 = léger (réutilise la visu faite) ; P4 = léger-moyen (géométrie déjà résolue).
- **Borne P1 (anti-puits-sans-fond)** : cible précise = le blocage `frame_records.go:667-674`
  (détection false-clean → un NEW mal aligné binde le mauvais archétype). Approche : porter le
  deser du keyframe type-2 validé BIT-EXACT contre l'oracle `keyframe_buffer_live.bin` (déjà
  déterministe/offline) + Ghidra sur le deser. **Sortie explicite si bloqué** : si après
  l'effort borné `tmp_kf_world` ne passe pas 0→≥8 bipeds bindés, statuer BLOQUÉ, parquer la
  visu (flag OFF documenté avec critère de reprise), ne RIEN livrer de CE. Décision d'investir
  = go/no-go utilisateur (ressource), pas un choix technique à lui déléguer.
- **Archi** : le travail P1/P2 reste dans `internal/analysis/filmdec` (couche analysis) ; P3
  re-cible `cmd/replay-build` sur la source offline (l'endpoint/front/pipeline sont faits et
  arch-conformes). Aucun nouveau couplage handler/DB. Validation = oracle bit-exact + gates
  `tmp_kf_world`/couverture (pas de tests unitaires classiques sur du RE de framing).
- **Repère produit tranché** : offline-pur only (CE écarté) = décidé user. Reste go/no-go RE.

## [2026-07-11] Calibration chemin absolu i0 contre l'oracle CE + cleanup ClearPos/gen — COMPLÉTÉ (verdict négatif honnête)

**Décision technique** : ajout de 2 knobs additifs dans filmdec (defaults préservant le comportement
legacy) — `SetAbsoluteAxisW(w)` (override largeur d'axe des CHEMINS ABSOLUS uniquement :
consumeAbsoluteWithGate + predFlag==1, 0=pd.AxisW) et `SetAbsDequantMode` (AbsDequantRange legacy vs
AbsDequantCenteredQuantum `(q-2^(w-1))·DeltaQuantum`). Scorer `cmd/tmp_absscore` : sweep
{AxisW∈6,10,12,14}×{centré,range} jugé sur la boîte oracle (X[-6..36] Y[-25..27] Z[-4..7], quantum
0.0138 CONFIRMÉ par histogramme oracle = diffs 0.013/0.027/0.041).

**Résultats (chunks 3-10)** : AUCUN combo ne satisfait {seeds in-box ∧ z-plat ∧ échelle 42/53/11}.
- Mode range (Cliffhanger) = échelle 10^3 partout (0/8 in-box) — CONFIRMÉ faux.
- Centré = seule forme à l'échelle 10^1. Mais : centre/6 = in-box+z-plat MAIS collapsé (spans
  0.2/0.1/0.8, ~0, pas 42/53/11) ; centre/12 = spans X/Y ~42/49 (bon !) MAIS Z=55 (pas plat), 3/34
  in-box ; centre/14 explose (spans 190+).
- **Cause racine** : l'offset i0 keyframe +182 (gate-consensus) est un FAUX POSITIF — q14 bruts X =
  588-4195, or une position in-box exige q14 X∈[7757,10801] (centre 8192). Le vrai bit-offset de l'i0
  absolu n'est pas établi ; il faut aligner sur `bitCursor` de l'oracle (hors périmètre).
- **Mur** : clean/desync stable 7968/495 sur TOUS les combos ; top desync = game-engine-team-mapping
  (ti=0), sans rapport avec la largeur i0 biped. Changer l'AxisW absolu NE FAIT PAS reculer le mur.
- replayAbs (predFlag==1/fallback réellement émis) = 32 sur chunks 3-10 : le chemin est exercé mais
  rare (position biped dominée par le DELTA, pas l'absolu).

**Conclusion** : la piste centered-quantum est la bonne DIRECTION (tue l'aberration 10^3) et quantum
0.0138 est confirmé, mais la calibration échoue car le bit-offset/alignement de traversée de l'i0
absolu n'est pas résolu (le juge suivant = alignement bitCursor oracle). Defaults filmdec inchangés
(zéro régression : tmp_kfworldpos 249/250+8/8, tests filmdec verts).

**Cleanup (Volet 2)** : `World.ClearPos` (jamais appelée) + champ `slotState.FullID` (write-only,
jamais lu) SUPPRIMÉS + commentaire world.go corrigé (décrivait une garde de génération inexistante).
Câblage rejeté : nécessiterait de threader le gen dans decodeDelta + définir la sémantique de recyclage
+ mesurer si des deltas cross-gen existent (inconnu) → garde à moitié risquant de dropper des deltas
valides. BindFull/BindSoft réécrivent déjà tout le slotState (PosValid=false) sur rebind = seule voie
de changement d'occupant dans un flux bien formé. Sorties : scratchpad/abs_sweep.txt + best_traj.txt.

## [2026-07-25] DÉCODEUR 100 % OFFLINE — LES TRAJECTOIRES SONT DÉCODÉES (objectif atteint)

Outil : `apps/go-api/cmd/tmp_offlinedec/main.go` (Go pur, CGO_ENABLED=0). Entrée = UNIQUEMENT les chunks
du film. Vérifié non-circulaire : le scanner ne référence aucun CSV/artefact CE (grep), la bande biped
vient de `filmdec.WalkKeyframeWorld` sur le paquet type-2 (TI==35).

**RÉSULTATS (verdict accuracy-density = CONFIRMED, recalculés indépendamment)**
- OFFSETS (clé chunk+packetIndex+bitOffset, au BIT près) : recall **99,95 %** (48 614/48 639),
  précision **99,9938 %** en fenêtre couverte (3 « faux » qui sont en fait de vrais records).
- POSITIONS au **QUANTUM EXACT** (indice entier, pas de tolérance) : X **100 %**, Y 99,9979 %,
  Z 99,9959 % ; les 2 écarts s'expliquent (la vérité CE échantillonne au tick, décalage de 2 paquets —
  la position est retrouvée au bit près 2 paquets plus loin) ⇒ **100 % effectif**.
- **Z=14 CONFIRMÉ empiriquement** : en w13 le résidu est bimodal ±0,0042 = ±step14/2 sur 100 % des records
  (0 exact) ; en w14 : 47 974 résidus nuls, aucun mode secondaire. `AxisW = {13,13,14}`.
- DENSITÉ : 6,28 bipeds/paquet offline vs 7,10 vérité — identique à 0,006 % près dans la fenêtre commune
  (l'écart vient des 6 146 doublons de la vérité + de la fin du chunk 08 non couverte par CE).
- TRAJECTOIRES : pas moyen **0,0424 u** (attendu 0,04), p99 0,139, **ZÉRO téléport** sur le périmètre validé.
- FILM ENTIER (chunks 01..26) : **171 116 positions, 98 slots, 30 418 paquets, en 10,2 s**.
- Détection des bipeds : `WalkKeyframeWorld` par chunk (1 keyframe/chunk) + union avec le keyframe suivant
  + `fillBand` (comble la bande contiguë) → recall 98,88 % → 99,95 % (cas réel : slot 524 créé ET détruit
  à l'intérieur du chunk 04, invisible dans les deux keyframes).

**ARTEFACTS** : `.ai/re_dump/offline_trajectories.csv` (171 116 points : slot,chunk,packetIndex,ts,x,y,z)
et `.ai/re_dump/offline_trajectories.png` — **la géométrie de Cliffhanger apparaît** (couloirs, salles,
passages dessinés par les déplacements). Preuve visuelle du décodage.

**NUANCES HONNÊTES (verdict offline-purity = PARTIAL) — à traiter**
1. **La range de déquant reste une constante CE** : `QuantRangeCEBiped` (quantize.go:38) est un dump de
   `DAT_14462cbe0[0]` codé en dur pour la map 000d5950. Formulation exacte : « zéro CE pour la DÉTECTION
   des records ; 1 constante de calibration CE pour la DÉQUANTIFICATION ». Pour l'universalité il faut
   **lire la table de ranges DEPUIS LE FILM** (identifié, borné, non fait).
2. **Sélection de modèle sur données étiquetées** : skip=5 / w=13 choisis en maximisant la corrélation
   avec la vérité CE ; la grammaire d'en-tête dérivée en remontant depuis le curseur CE. Légitime en RE,
   mais les règles sont apprises sur ce jeu → pas de held-out (chunks 01..08 évalués = ceux couverts).
3. **Portée** : 28,4 % du volume livré est confronté à une vérité (48 617/171 116). Les chunks 09..26 et
   71 des 98 slots ne sont validés par rien. Démontré = « reproduit les 90 s couvertes par la capture ».
4. `fillBand` est une heuristique (comble les trous de la bande), calibrée sur le cas du slot 524.
5. Anomalie non diagnostiquée : slot 549 / chunk 11, positions identiques aux paquets 426/428 et 440/442
   avec une excursion de 2,68 u — signature d'un slot partagé ou d'un re-décodage. Zone sans vérité.

**PROCHAINE ÉTAPE POUR L'UNIVERSALITÉ** : extraire la table de ranges depuis le film (remplacer la
constante CE) → le décodeur devient valide sur tous les films/maps. Puis brancher
`offline_trajectories.csv` sur la visu replay 2D (le pipeline `internal/analysis/replay` existe déjà).

## [2026-07-25] GÉOMÉTRIE DE LA MAP — localisée (question utilisateur : « stockée ailleurs ? » → OUI)

Cliffhanger est une map **FORGE** (nom complet : « Super Fiesta:Slayer on Cliffhanger - Forge »), donc
absente de `deploy/ds/levels/multi/` (vérifié : aucun dossier `cliffhanger`, aucun fichier au GUID sur les
2 installs ni dans AppData).

**TROUVÉE via le cache disque du jeu** : `<install>/disk_cache/webcache/` (3072 réponses HTTP cachées,
écrites pendant la lecture du film). 3 fichiers contiennent le GUID de la map et donnent l'URL exacte :
```
https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/map/
  5324364b-39a8-4f93-96a6-b80a1f18ce8a /   ← assetId (= MapID en DB)
  5a78537e-53d5-44e2-bdbd-d2088038523c /   ← versionId (= MapVersionID en DB)
    map.mvar            (93 795 o)  ← objets Forge placés
    ridgeline.mvar      (84 794 o)
    images/hero.jpg, screenshot1.jpg, thumbnail.jpg
```
Métadonnées : auteur `vissager`, description « Haute altitude. Hautement classifié ».
**ACCÈS PUBLIC** : HTTP 200 sans authentification (Cache-Control public). Téléchargés dans
`.ai/re_dump/mapvar/cliffhanger_{map,ridgeline}.mvar`.

**RÉVÉLATION MAJEURE : Cliffhanger est bâtie sur le canevas RIDGELINE**, et `ridgeline` EST installée en
local (`deploy/ds/levels/multi/ridgeline/ridgeline-rtx-new.module`, 8,4 Mo). Donc :
  terrain de base = .module LOCAL   +   objets Forge = map.mvar TÉLÉCHARGEABLE
⇒ la géométrie complète est atteignable, et le schéma se généralise : pour toute map Forge, l'URL se
construit depuis MapID + MapVersionID déjà stockés dans `match_registry`.

FORMAT `.mvar` : en-tête `e0 dc 05 2a d7 01 0a 07 10 …` ; tags de type protobuf (`0a`, `2a`, `ca 15`…) ;
truffé de float32 LE plausibles (16.312, 6.41, -0.92, 81.0, 23.5, 1.1, 0.35…). Scan naïf : 3325 triplets
dans l'enveloppe de la map. Enveloppe réelle des trajectoires décodées (référence de recoupement) :
X[-6.7, 43.8] Y[-25.0, 30.3] Z[-4.2, 7.1]. Parsing structuré = tâche suivante.

## [2026-07-25] map.mvar PARSÉ — 453 objets Forge, fond de carte obtenu

**FORMAT : Microsoft Bond CompactBinary v2** (PAS du protobuf — la ressemblance tag+varint était fortuite).
Preuve auto-validante : l'en-tête `e0 dc 05` est un varint LEB128 = 93792 = taille-3 ⇒ le fichier entier est
UN struct à préfixe de longueur ; un parseur Bond CBv2 générique (sans schéma) consomme **exactement**
93795/93795 octets (map.mvar) et 84794/84794 (ridgeline.mvar), ~2000 longueurs imbriquées cohérentes,
zéro overrun, zéro octet résiduel.
Encodage des tags (ce qui trompait le scan naïf) :
```
tag_byte = (bond_type & 0x1F) | ((field_id & 0x07) << 5)
(field_id & 7) == 6 -> +1 octet  = field_id (uint8)
(field_id & 7) == 7 -> +2 octets = field_id (uint16 LE)
```
Types vus : 0 STOP, 2 BOOL, 3 UINT8, 5 UINT32, 7 FLOAT(4o LE), 10 STRUCT, 11 LIST, 16 INT32 (zigzag).

**SCHÉMA D'UNE ENTRÉE OBJET** (top.#3[i]) :
```
#2  STRUCT { #0 INT32 }         type_id (hash de tag)
#3  STRUCT { #0,#1,#2 FLOAT }   POSITION (x,y,z) monde
#4  STRUCT { #0,#1,#2 FLOAT }   UP (unitaire)
#5  STRUCT { #0,#1,#2 FLOAT }   FORWARD (unitaire) -> yaw dérivable
#7  UINT8                       flags
#8  STRUCT                      property bag (entiers : équipe/variante/budget)
```
**453 objets** (map.mvar) / 443 (ridgeline.mvar), 45 type_id distincts.

**VALIDATION CROISÉE (4 preuves indépendantes)**
(a) HOMOGÉNÉITÉ : 453/453 et 443/443 objets ont exactement la même signature de champs (un décalage de
    parsing éclaterait les signatures).
(b) ORTHONORMALITÉ : #4 et #5 unitaires à 1e-3, produit scalaire ~3e-5 ⇒ vraie base de rotation.
(c) RECOUPEMENT INTER-FICHIERS : 438 positions identiques au millimètre entre les deux .mvar.
(d) RECOUPEMENT AVEC LES TRAJECTOIRES (le plus fort) :
    - densité (grille 3 m) : **Spearman 0.578** vs baseline aléatoire même-boîte **0.030** ⇒ ×19.
      La baseline ayant la MÊME boîte, l'argument « même boîte » ne peut pas expliquer le signal.
    - test VERTICAL (« on marche SUR les objets ») : dz = z_joueur − z_objet le plus proche en XY (n=16199) :
      **51,0 %** à |dz| < 0,25 m, contre **12,6 %** (±5,1) pour un contrôle XY-permuté ⇒ **×4,0**, pic centré
      sur dz=0. Les origines Forge sont au niveau de la surface d'appui.
    - 1 seul point sur 171 116 hors géométrie.

**LIMITE DU FORMAT (pas du parsing)** : **aucune ÉCHELLE d'objet** — vérifié exhaustivement sur les 3562
floats du fichier. La taille géométrique vient du modèle référencé par `type_id`, qui vit dans les `.module`.
⇒ le fond de carte est une carte de CENTRES orientés + nappe d'altitude, pas un plan de sols exact.

**BORNES DE QUANTIFICATION : ABSENTES du .mvar** — certitude (fichier intégralement parsé, ses 3562 floats
énumérés) : aucun ne s'approche des 6 bornes ni de leurs dérivées (extents 113.21/113.82/137.55). Logique :
le .mvar décrit la VARIANTE Forge, pas le volume du scénario. Piste `ridgeline-rtx-new.module` explorée mais
NON CONCLUANTE (valeurs proches mais non alignées, non bit-exactes 70-180 ULP, voisinage = bruit Kraken) —
faux positifs probables ; il faudrait décompresser proprement.

ARTEFACTS : `.ai/re_dump/map_objects.csv` (453 lignes : idx,type_id,x,y,z,up_*,fwd_*,yaw_deg,flags,offset)
et `.ai/re_dump/map_geometry.png` (2 panneaux : objets seuls / objets + 171k positions joueurs).

CAVEAT PROCESSUS : la phase de VÉRIFICATION adversariale de ce workflow a planté (sortie structurée
invalide, 5 essais) ⇒ les chiffres ci-dessus sont ceux de l'agent de parsing, avec ses propres contrôles
(baselines et tests de permutation, qui sont solides), mais SANS revue adversariale indépendante.

## [2026-07-25] Overlay canevas Ridgeline — NÉGATIF assumé + BUG ooz corrigé + alerte saturation corrigée

**1. BUG DE PRODUCTION TROUVÉ ET CORRIGÉ : `internal/ooz/ooz.go`**
Crash `0xC0000005` reproductible (ridgeline, entrée #1082, rawLen=0x100000, faute à dst+0xFFFEC). Les
copieurs SSE d'ooz (Kraken/Mermaid) écrivent par blocs de 16/32 o et débordent au-delà de la dernière
position décodée ; sans la marge SAFE_SPACE d'Oodle, le process meurt quand la taille décompressée tombe
près d'une limite de page. Fix : `dst := make([]byte, rawLen+128)` puis retour `dst[:rawLen:rawLen]`.
**Catalyst passait « par chance »** (layout de heap différent) : le bug était LATENT SUR TOUTES LES MAPS.
Non-régression : catalyst re-extrait = 277 resources / 523 meshes, exactement les chiffres du HANDOFF §9.4 ;
tmp_kfworldpos toujours 249/250 + 8/8.

**2. MA PRÉMISSE « SATURATION DE MASSE » ÉTAIT FAUSSE** (diagnostic, pas effet)
Comptage réel des points collés aux bornes (≤0,005) : **9 sur 171 116 = 0,005 %** (X_min=1, Z_min=1…), pas
un effondrement au bucket 0. Ce sont des aberrants isolés, sur 5 slots. Chunks 01-08 (validés) : 3/50 516 ;
chunks 09-26 (non validés) : 6/120 600 ⇒ **la saturation ne discrimine pas les deux populations**, ce n'est
donc pas un marqueur de qualité de décodage. MAIS l'effet était réel : retirer ces 9 points fait passer le
span Z de **91,4 → 11,3 wu** (et span X 84,9 → 50,8). Enveloppe PROPRE : X[-7,016 ; 43,772]
Y[-25,144 ; 30,349] Z[-4,198 ; 7,068]. Le span Z de 11,3 identifie Z comme vertical (même convention que
le module). CSV : `.ai/re_dump/offline_trajectories_clean.csv` (171 107 pts, 100 slots).

**3. OVERLAY RIDGELINE : NÉGATIF — et la métrique historique est disqualifiée**
Chaîne OK (354 resources, 775 meshes, aucune dérive de format), mais :
- métrique « point dans l'empreinte XY » (celle des **84 % de catalyst**) : identité 91,8 % MAIS le contrôle
  négatif (grille uniforme, même bbox) donne déjà **71,3 %**, et le balayage de translation culmine à
  **99,5 % en (dx=-20)**, PAS à l'identité ⇒ métrique **sans valeur discriminante** ;
- méthode « 8 orientations normalisées » de `tmp_mapoverlay` : **100 %** — tautologique par construction
  (on renormalise les deux nuages sur leur propre bbox) ⇒ **LES 84 % DE CATALYST DOIVENT ÊTRE RELUS AVEC
  CETTE RÉSERVE** (validation historique probablement vide) ;
- métrique DISCRIMINANTE « joueur posé sur une surface » (|dZ| ≤ 1 wu du sommet de mesh le plus proche) :
  identité = **10,3 %**. Écart vertical médian : les joueurs sont ~17 wu SOUS la géométrie du canevas.
  Meilleure translation rigide 51,6 % (dx=-25, dz=+17,5), meilleure similitude 47,5 % — **paramètres ajustés
  pour maximiser le score = PLACAGE**, ni stable ni dérivable. Aucune transformation justifiable établie.

**CONCLUSION DE FOND (confirmée par l'observation de l'utilisateur sur le PNG)** : superposer les trajectoires
au terrain du CANEVAS est la mauvaise cible. Sur une map Forge, les joueurs marchent sur les **objets posés**
(le .mvar), pas sur le terrain d'origine. Et c'est cohérent avec les mesures : les objets Forge corrèlent
vraiment aux trajectoires (Spearman 0,578 vs 0,030 ; test vertical 51 % vs 12,6 %), le canevas non (10,3 %).

**PROCHAINE ÉTAPE** : donner leurs FORMES aux objets. Le .mvar n'a pas l'échelle, mais les 45 `type_id`
couvrent 453 objets (95× le même type, 63×, 42×, 30×, 22×…) = blocs Forge standard. Résoudre
`type_id` → objet → dimensions (hash inverse des tags des .module / palette Forge / inférence par les
trajectoires), puis dessiner des rectangles orientés (position + base orthonormée déjà validées).

## [2026-07-25] Objets Forge résolus (45/45) + emprises mesurées — puis RECADRAGE sur le replay 2D

**RÉSOLUTION type_id : 45/45.** Découverte : le `type_id` du .mvar **n'est PAS un hash** — c'est
DIRECTEMENT l'identifiant global du tag, stocké à l'offset **+0x28** de l'entrée de module (base 0x48,
stride 0x58). Les 45 sont tous du groupe **`food`** (forge object definition). Chaîne complète :
`food → bloc|scen|mach|weap|vehi (parfois via foki) → hlmt → mode → bloc 84 o "compression info"`.
Répartition : 26 bloc, 9 mach, 8 scen, 2 weap, 1 vehi.

PIÈGES DOCUMENTÉS (2 h perdues) :
- champ tag-reference = 0x1C octets ; **l'id exploitable est à +0x08, PAS à +0x10** (l'ordre
  « assetId puis globalId » d'AusarDocs ne s'applique pas à ce build). Avec +0x10 : 0/27 refs résolues ;
  avec +0x08 : 45/45.
- `himodule.Open` échoue sur ces gros modules : (a) `os.ReadFile` du fichier entier (2 Go), (b) `dataBase`
  faux (padding de fin). Lecteur ReadAt écrit dans `cmd/tmp_forgedim/reader.go` : table des ressources =
  fin des entrées **+8 octets** (le +8 manque dans himodule), `dataBase = align(fin table blocs, 0x1000)`.
- bbox depuis le tag `mode` : bloc 84 o, `flags u16 @+0x00` doit valoir **3**, puis 3 RealBounds (min,max)
  PAR AXE à +0x04. **Le filtre flags==3 est indispensable** (sinon on capte des blocs voisins → dimensions
  fausses). Indexation des 88 modules any/+ds/ (headers seuls) : <1 s, ~150 Mo RAM ; table des 45 types : 0,9 s.
- Les tags n'ont **aucun nom lisible** (`fileNameSize = 0`, chemins strippés du build release) ⇒ objets
  identifiés par ids, pas par nom.

**EMPRISES** : 382/453 objets (84,3 %) avec bbox mesurée ; 71 sans = modèles vides (volumes invisibles :
spawns, zones de blocage) — pas des échecs. Rendu en polygones orientés (8 coins de bbox transformés par la
base orthonormée du .mvar, enveloppe convexe projetée).

**VALIDATION DISCRIMINANTE (la bonne métrique cette fois)** :
- joueurs à |dz| < 0,25 m au-dessus d'une emprise réelle : **2,94 % réel vs 0,79 % contrôle** (permutation
  XY) = **3,74×** ; contrôles plus sévères (rotation rigide 0,52 %, translation ±3/±6 m 0,70 %) ⇒ 4 à 6×.
- métrique conditionnelle (sachant qu'on survole une emprise, est-on à la bonne altitude ?) :
  **42,0 % vs 12,7 %** = 3,32×.
- **PREUVE LA PLUS FORTE** — balayage d'offset vertical (tolérance 0,10 m) : offset 0,00 → **6,7×** ;
  offset ±0,20 à ±1,00 m → ratio **0,24 à 0,43, SOUS le contrôle**. Un artefact « point dans la boîte »
  donnerait un plateau large ; ici **pic net de ~15 cm** ⇒ vraie coïncidence physique.

**ZONES NON EXPLORÉES (question utilisateur) — chiffré** : cellules à objet sans trajectoire = **33 m²**
(21 % des cellules à objet) ; **85 % d'entre elles à ≤ 2 cellules du nuage parcouru** (bordures, décor
collé aux murs) ; aucune en hauteur ; 17 objets/453 (4 %) réellement isolés (décor périphérique, spawns
non utilisés). L'impression de vide vient du **hors-map** : 2 257 m² (61 % du rectangle englobant) n'ont
NI objet NI trajectoire.

**RECADRAGE (demande utilisateur : l'objectif est un REPLAY 2D, Z = indication d'étage, non critique)** :
les 453 objets sont de **petits props** (surface cumulée 93,7 m², moyenne 0,25 m²/objet) = armes, spawns,
décor — PAS les sols/murs. Ce sont **les trajectoires qui dessinent la map**. C'est suffisant pour le
replay 2D : les objets serviront de repères contextuels. **Arrêt du RE de géométrie ici.**

PROCHAINE ÉTAPE (ligne d'arrivée) : brancher `offline_trajectories_clean.csv` sur la visu 2D existante —
débrancher la capture CE de `cmd/replay-build`, ajouter la géométrie en champ optionnel (Étape B prévue
par le package), Z → tranches d'étages, et mapper `T` sur les `ts` de paquets pour la vitesse de lecture.

## [2026-07-25] Replay 2D branché sur le film : capture CE débranchée, timeline réelle, géométrie optionnelle

**STATUT : Complété (backend).**

**1. DÉCODEUR PROMU** — la grammaire du record biped quitte `cmd/tmp_offlinedec` (jetable) pour
`internal/analysis/filmdec` :
- `film_packets.go` : `ReadFilmChunk` (zlib), `WalkPackets`, `FilmPacket{Type, Start, Size, TimestampUS}`,
  `CountFilmChunks`. Met fin à la Nième copie du couple `inflate`/`walkPackets` des outils `tmp_*`.
- `offline_biped.go` : `ScanFilmBipedPositions(dir, opt)`, `ScanBipedRecords` (PUR, testable),
  `DequantBipedAxis`, `BipedTypeIndex`, `BipedPosition{Slot, Chunk, PacketIndex, TimestampUS, X,Y,Z}`.
- `offline_filters.go` : `DropIsolated` + `DropTeleports` (voir point 4).
- `cmd/tmp_offlinedec` réduit à une enveloppe d'export CSV (zéro duplication de grammaire).

**2. CAPTURE CE DÉBRANCHÉE** — `cmd/replay-build <matchId>` lit désormais les SEULS chunks du film.
`ParseCapture`, `CaptureRecord`, `BuildDocument`, `decodeSamples`, `reconstruct`, `DefaultAxisW`
supprimés de `internal/analysis/replay` (code mort après bascule).
Commande : `CGO_ENABLED=0 LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-build --geometry <repo>/.ai/re_dump 000d5950`

**3. TIMELINE RÉELLE** — l'axe `T` reste un index de frame (compat client) mais sur une grille UNIFORME
rééchantillonnée depuis les `ts` de paquets (µs), origine = premier paquet. Deux champs optionnels le
mettent à l'échelle : `frameIntervalMs` (100) et `durationMs`. Résultat : 4985 frames × 100 ms =
**498,5 s = 8 min 18 s**, cohérent avec la durée du match. Le rejeu « court » venait bien d'un axe
= index de record.
**À FAIRE CÔTÉ WEB** : `ReplayCanvas.DEFAULT_SPEED = 60` doit devenir `1000 / doc.frameIntervalMs`
(sinon lecture 6× trop rapide).

**4. DEUX FILTRES ANTI-FAUX-POSITIFS (découverte de cette étape)** — le scan bit à bit produit
~7 aberrations sur 171 849 qui, seules, DOUBLENT l'enveloppe (X[-37,5;49,1] au lieu de X[-7,0;43,8]).
Ce sont les 9 points « collés aux bornes » du journal précédent, mais leur vraie signature n'est pas la
saturation : ce sont des points UNIQUES séparés de 66 à 320 s du reste de leur slot.
- `DropIsolated` (défaut 15 s) : écarte un échantillon dont l'écart au voisin le plus proche du même slot
  dépasse le seuil. C'est lui qui restaure l'enveloppe propre.
- `DropTeleports` (défaut 100 m/s) : écarte les sauts intra-vie (16 points).
- `DropSaturated` (bucket 0 ou 2^w-1) : 2 points.
Enveloppe obtenue **X[-7,016;43,786] Y[-25,144;30,349] Z[-4,198;7,077]** = celle de
`offline_trajectories_clean.csv` au centimètre près, avec 99 slots (un de plus : la bande de slots
couvre un chunk de plus).

**5. DOCUMENT ÉTENDU, SchemaVersion INCHANGÉ (=1)** — tous les ajouts sont `omitempty` :
`frameIntervalMs`, `durationMs`, `geometry[]` (props Forge : typeId, x, y, z, dx, dy, yaw),
`geometryBounds`, `bounds.minZ/maxZ`, `point.z`, `track.startFrame/endFrame`.
Piège assumé et documenté : `omitempty` sur un float32 omet un zéro exact ; les coordonnées viennent
d'une déquantification à mi-bucket, un zéro exact est hors d'atteinte.

**6. ARTEFACT 000d5950** : 99 tracks / 29 221 points / 382 objets de fond / 4985 frames / 498,5 s /
**1,20 Mo** de JSON. Max **8 entités simultanées** (= les 8 joueurs), moyenne 5,61.
Compromis de taille disponible sans changer de code : `--interval 200` → 624 Ko, `--interval 250` → 508 Ko.

**LIMITE ASSUMÉE** : 99 tracks = 99 VIES, pas 8 joueurs. Le regroupement vie→joueur n'est PAS fiable par
continuité : les slots sont alloués séquentiellement (pas de structure par joueur), et une nouvelle vie
démarre à un point d'apparition sans lien spatial avec la mort précédente. Avec 8 joueurs qui meurent en
se chevauchant, l'appariement « mort → apparition suivante » est ambigu. Étape D à traiter avec une vraie
source d'identité (keyframe player→biped), pas par heuristique.

TESTS : `go build`/`go vet` OK ; `go test ./internal/analysis/replay/` + `./internal/analysis/filmdec/` OK
(nouveaux tests : timeline, décimation, bornes, géométrie, grammaire biped, filtres) ;
non-régression `tmp_kfworldpos` **249/250 + 8/8** inchangée.

---

## [2026-07-25] Rejeu 2D — front exploitable : fond de carte, temps réel, étages

**Statut** : Complété (front). Branche `feat/filmdec-continuation`, aucun commit.

**DÉCISION TECHNIQUE** — le composant canvas consomme les champs optionnels ajoutés par le backend ;
la logique reste PURE (`replayLogic.ts`) et le dessin est extrait dans un module dédié
(`replayDraw.ts`), le composant ne garde que l'état React et les contrôles.

1. **TEMPS RÉEL** — `DEFAULT_SPEED = 60` (frames/s constant) est remplacé par
   `framesPerSecond(doc) = 1000 / frameIntervalMs` (= 10 pour un pas de 100 ms) ; les boutons de
   vitesse deviennent des multiplicateurs `0,5× / 1× / 2× / 4×` de cette base, et un chronomètre
   `m:ss / m:ss` affiche le temps de match (`durationMs`). Le rejeu dure donc 8 min 18 s à 1×
   au lieu de ~83 s. La traînée est exprimée en temps réel (8 s) et non en nombre de frames.
2. **FENÊTRE DE VIE** — cause réelle du « les joueurs bougent peu » : `positionAt` maintient la
   dernière position, donc les 99 vies restaient FIGÉES à l'écran après leur mort. Les entités
   sont maintenant masquées hors de `[startFrame, endFrame]` → max 8 points mobiles (vérifié sur
   l'artefact : 8, moyenne 5,6), ce qui rend le mouvement lisible.
3. **FOND DE CARTE** — `geometry[]` dessiné SOUS les trajectoires : rectangle orienté par `yaw` sur
   l'emprise `dx/dy`, ou carré de 2,5 px quand l'emprise projetée est sous le seuil de lisibilité
   (277 rectangles + 105 points à 6,7 px/m). Cadrage sur l'UNION `bounds` + `geometryBounds`
   (`sceneBounds`), sinon les props débordant de la zone parcourue seraient rognés.
4. **ÉTAGES (non critique)** — opacité du décor croissante avec l'altitude (0,14 → 0,38) +
   filtre de tranche Bas/Milieu/Haut (3 bandes sur `bounds.minZ/maxZ`) qui estompe les entités
   hors tranche au lieu de les cacher. Contrôle masqué si le dénivelé est < 1 m.
5. **CADRAGE** — `fitWidth` : la carte est quasi carrée, le canvas large ; à hauteur fixée la scène
   n'occupait que la moitié de la largeur. Le canvas prend désormais la largeur du ratio de scène,
   centré.

**COULEURS** — zéro hex ajouté : séries via `getSeriesColors` (8 tokens `chart-series-*`), fond de
carte via `resolveToken('divergent-neutral')` (token neutre), opacités via `globalAlpha` (pas de
manipulation de valeur couleur). Le repli `#888888` du canvas (marqueur `color-allow`) a disparu.

**RÉSULTATS** — `npm run typecheck` OK ; `npm run test:run` : 191 fichiers, 1743 tests OK
(34 dans match-replay, dont 14 nouveaux) ; `eslint` OK ; `tools/lint-no-hardcoded-colors.mjs` :
0 violation. Vérif visuelle en navigateur IMPOSSIBLE ici (l'API qui tourne sur :8000 vient d'une
autre branche, sans l'endpoint replay ; build Go hors périmètre) — remplacée par un rendu PNG hors
ligne des mêmes données/projections.

**PROCHAINE ÉTAPE** — Étape D (identité slot→joueur) pour colorer par équipe et nommer les traces ;
le fond de carte reste un semis de petits props (le décor Forge ne contient ni sols ni murs).

## [2026-07-25] REPLAY 2D BRANCHÉ SUR L'OFFLINE — la boucle est bouclée

**BACKEND** (verdict vérif : CONFIRMED sur l'essentiel)
- Décodeur PROMU de `cmd/tmp_offlinedec` vers `internal/analysis/filmdec` : `film_packets.go` (101 L :
  ReadFilmChunk/WalkPackets/FilmPacket — met fin à la ~40e copie du couple inflate/walkPackets des outils
  tmp_*), `offline_biped.go` (283 L : ScanFilmBipedPositions/ScanBipedRecords/DequantBipedAxis),
  `offline_filters.go` (111 L, purs) + tests. `cmd/tmp_offlinedec` réduit à 53 L.
- **CAPTURE CE DÉBRANCHÉE** de `cmd/replay-build` : lit les chunks du film via `replay.BuildFromFilm`.
  Code mort supprimé (ParseCapture, CaptureRecord, Options.AxisW, DefaultAxisW, BuildDocument,
  decodeSamples, reconstruct + leurs tests). Grep : 0 occurrence restante.
- Document étendu, **SchemaVersion INCHANGÉ = 1**, tout en omitempty : frameIntervalMs, durationMs,
  geometry ([]MapObject), geometryBounds, Bounds.minZ/maxZ, Point.z, Track.startFrame/endFrame.
- **TIMELINE RÉELLE** : grille uniforme rééchantillonnée depuis les ts de paquets ; 4985 frames × 100 ms
  = 498,5 s = **8 min 18 s** (conforme aux ~8,5 min du match). Le « rejeu trop court » venait bien d'un axe
  = index de record.
- Artefact : `data/cache/replays/halo_infinite/000d5950.json` — 99 vies, 29 221 points, 382 objets,
  1,20 Mo (≈250-300 Ko gzip ; `--interval 250` → 508 Ko si besoin). Max 8 entités simultanées.
- **DÉCOUVERTE** : ~7 aberrations sur 171 849 positions DOUBLENT à elles seules l'enveloppe. Leur vraie
  signature n'est PAS la saturation (mon diagnostic précédent) : ce sont des points UNIQUES séparés de
  66 à 320 s du reste de leur slot (ex. slot 513 vit au chunk 3, son aberration est au chunk 13). Un filtre
  de vitesse seul échoue (dt énorme → vitesse apparente faible). D'où 3 filtres purs et testés :
  DropIsolated (>15 s du voisin le plus proche), DropTeleports (>100 m/s intra-vie), DropSaturated.
- Validation croisée : enveloppe identique au centimètre à la référence.

**FRONTEND**
- Logique séparée du composant : `replayLogic.ts` (262 L, pur) + `replayDraw.ts` (148 L, nouveau).
- Fond de carte sous les trajectoires : 277 rectangles orientés + 105 points = 382 objets, 0 hors canvas.
- Étages : opacité du décor croissante avec l'altitude + filtre 3 tranches (Tous/Bas/Milieu/Haut), les
  entités hors tranche sont estompées et non masquées. Groupe masqué si (maxZ−minZ) ≤ 1 m.
- **Vitesse corrigée** : `DEFAULT_SPEED = 60` → `framesPerSecond(doc) = 1000/frameIntervalMs` (10 f/s).
  Avant : ~83 s de lecture (6× trop rapide). Maintenant 8 min 18 s à 1×.
- Règle color-tokens respectée (garde-rail `lint-no-hardcoded-colors` : 0 violation) ; nuances via
  `globalAlpha` uniquement. i18n FR/EN. tsc 0 erreur ; 1743 tests verts (34 sur la feature, 14 nouveaux).

**LIMITES HONNÊTES (relevées par la vérif, verdict PARTIAL)**
1. **« Zéro CE » est SURESTIMÉ** : `filmdec.QuantRangeCEBiped` (quantize.go:32-38) reste une constante
   **capturée via Cheat Engine** (DAT_14462cbe0[0], 2026-07-11) codée en dur pour cette map. Aucun fichier
   CE n'est lu à l'exécution, mais **l'échelle/offset monde absolu du décodeur est d'origine CE**.
   L'en-tête de `cmd/replay-build` doit être nuancé. C'est LA dette restante pour l'universalité.
2. `--geometry` pointe par défaut `.ai/re_dump` : un artefact de production lit des CSV d'un dossier de
   documentation/RE, hors PathResolver et hors `data/`. À traiter avant productionisation.
3. Justification omitempty inexacte dans document.go (« un zéro exact est hors d'atteinte ») : faux, round2
   arrondit au centimètre — l'artefact contient 14 points à z exactement 0. Conséquence nulle (absent lu
   comme 0), mais le commentaire est à corriger.

**RETOUR UTILISATEUR (validation humaine)** : « je connais la map et les déplacements me semblent hyper
cohérents et plausibles » — validation la plus forte obtenue. Et : « les vraies structures de la map ne sont
pas là » — EXACT : les 453 objets du .mvar sont des props (0,25 m² en moyenne). Les éléments structurels
(sols, murs, rampes) appartiennent au niveau de base **Ridgeline**, dont l'alignement a échoué (10,3 % au
critère « posé sur une surface » à l'identité ; le meilleur ajustement 51,6 % est un placage). C'est le gap
restant côté carte : trouver la transformation correcte entre le repère du .module et le repère monde du jeu.

---

## [2026-07-25] Décodeur cubemap (direction/aim) porté et VALIDÉ + cap de visée dans l'artefact — Complété

**Décision technique.** Port bit-exact de `FUN_1406d8288` dans `filmdec/aim_vector.go`
(`DecodeAimVector` / `DecodeAimVectorChecked`), avec les DEUX divmods (la 2e coordonnée que
la référence communautaire n'a pas) et les tables FIGÉES : `faceSize(p)=floor(2^p/6)`,
`gridSize(p)=floor(sqrt(2^p/6))-1`, p ∈ [6,30]. L'ancien port (`components_cubemap.go`)
posait `faceSize=gridSize²` et `gridSize=max g / 6g²≤2^W` : **les deux colonnes étaient
fausses** (p=19 → 295 au lieu de 294 ; p=30 → 178 944 129 au lieu de 178 956 970).
`cubemapDecode/Encode/GridSize` supprimés, `DecodeDynPrecDir` délègue — source unique.

**Découverte de flux (mesurée, pas supposée).** Le composant i0 consomme **2 bits de plus**
que les 45 modélisés par `ScanBipedRecords` (queue handle : handleSel + regionPresent, nuls
sur le chemin dominant) — cohérent avec le « total i0 = 47 » de la capture CE. Sans cette
correction, la direction lue était du bruit (73 % de face +X, aucune face négative). Trouvé
par balayage d'offset (`cmd/tmp_aimsweep`), juge = écart médian au cap de déplacement :
offset 4 / projection **xy** → 4,0° ; tous les autres offsets 49–137°.

**Résultats observés (film 000d5950, 171 826 positions).**
- i1 vélocité (cubemap 19 bits) : 145 578 échantillons ; normes **1,000000** ; faces
  23,8/23,5/3,7/22,9/22,2/4,0 % (signature physique) ; **0** face ≥ 6 ; max iu = max iv =
  **292 = N−2** et 0 violation → le stride N=294 est confirmé par les données seules.
- JUGE : cap décodé vs direction de déplacement — vélocité **médiane 4,0°** (99,8 % < 30°),
  visée i21 **médiane 11,1°** (79,3 % < 30°) ; CONTRÔLES : 90,0° / 81,0° / 86,8°.
- Test indépendant du repère : vitesse mesurée / magnitude i1 décodée = **médiane 1,022**
  (n = 130 597).
- Gain de la 2e coordonnée sur données réelles : écart 3D médian 1,16°, p90 26,5°,
  **max 44,9°** (les 45° annoncés aux coutures) ; en test unitaire 0,115° vs 20,1° d'erreur
  moyenne (facteur 175).

**Le cap de visée n'est PAS du cubemap.** i2 object-forward-and-up est inexploitable dans ce
film (0 échantillon). La visée dense vient de **i21 unit-desired-aiming-vector** :
`R(1) flag0 + R(12) cap + R(11) élévation` (offset du champ mesuré par `cmd/tmp_aimsweep2`,
concentration circulaire R = 0,84 vs 0,03 pour le bruit). Convention mesurée :
`yaw = 360·(q+0,5)/4096` **est directement** atan2(Y,X) (offset −0,16°).

**Artefact.** `Point.H` (`h,omitempty`, degrés) ajouté, **SchemaVersion inchangé (1)** ;
`BuildFromFilm` force `CaptureDirs`. 000d5950 régénéré : 99 tracks / 29 221 points
(inchangés), **12 739 points avec cap (43,6 %)**, 98/99 tracks. Piège omitempty traité :
un cap arrondi à 0 est publié comme 360.

**Vérifications.** `go vet` OK ; `go test ./internal/analysis/...` vert ; 7 tests ajoutés
(dont l'échantillon du blog, écart 10,13° documenté et non forcé, et la non-régression
`CaptureDirs=false` → positions identiques) ; `tmp_kfworldpos` inchangé (249/250, 8/8).

**Limites.** Couverture visée 44 % (la capture s'arrête au premier composant non modélisé
entre i0 et i21) ; l'élévation (pitch 11 bits) est capturée mais NON validée (aucun oracle
vertical) et non publiée ; l'aim 30 bits des fire-events reste à extraire (le décodeur est
prêt pour p=30).

**Prochaine étape.** Étendre la couverture de la visée en portant les desers des composants
intermédiaires (i4, i5, i25…), puis brancher le cap sur le rendu web (flèche d'orientation).

## [2026-07-25] VECTEUR DE VISÉE — cubemap décodé, le blog ET notre port corrigés

**GRAMMAIRE EXACTE** (FUN_1406d8288, triangulée par 3 fonctions sœurs dont l'ENCODEUR FUN_1407eaf1c —
c'est l'inverse qui donne la certitude sur l'ordre des coordonnées et le stride) :
```
face = code / faceSize[p]           rem = code % faceSize[p]
iu   = rem / gridSize[p]            iv  = rem % gridSize[p]     <-- LA 2e COORDONNÉE
step = 2/(N-1) ; a = iu*step-1+step/2 (snap 0 si iu*2==N-2) ; idem b avec iv
face 0..5 -> (+1,a,b) (a,+1,b) (a,b,+1) (-1,a,b) (a,-1,b) (a,b,-1) ; puis normalisation L2
```
**AUCUNE négation de signe** — la référence communautaire en applique ((1,-v,-u)…), c'est FAUX ; invisible
chez elle car son v valait toujours 0.

**LES DEUX TABLES SONT DÉRIVABLES — zéro dépendance runtime, zéro CE** :
- `faceSize(W) = floor(2^W / 6)` — lue octet à octet dans le .exe, vérifiée sur les 25 entrées p=6..30.
- `gridSize(W) = floor(sqrt(2^W / 6)) - 1` — loi trouvée dans l'initialiseur FUN_14038bc40 (25 blocs
  déroulés, chacun `(int)sqrtf(K_p) - 1`). Domaine légal p ∈ [6,30].
- **`0xAAA8000` (blog) EST FAUX** : 6 × 0xAAA8000 = 2^30 − 65536. La vraie valeur est **0x0AAAAAAA**
  = 178 956 970. Confirmé par lecture directe de la table @0x1447084D0.
- **BUG DANS NOTRE PROPRE PORT** (`components_cubemap.go`) : posait faceSize = gridSize² et un gridSize
  faux → ~25° d'erreur. Corrigé.

**DÉCOUVERTE COLLATÉRALE IMPORTANTE** : i0 consomme **2 bits de plus** que les 45 modélisés (queue handle :
handleSel + regionPresent, nuls sur le chemin dominant) — cohérent avec le « total i0 = 47 » de la capture CE.
Sans cette correction la direction lue était du bruit (73 % de face +X, aucune face négative). Trouvée par
balayage d'offset (juge = écart médian au cap de déplacement) : offset 4 → 4,0°, tous les autres 49-137°.

**VALIDATION (juge : écart angulaire vs direction de déplacement, même slot/instant)**
- i1 vélocité-direction : n=25 128, **médiane 4,0°**, <30° = 99,8 %
- i21 unit-desired-aiming-vector : n=13 131, **médiane 11,1°**, <30° = 79,3 %
- CONTRÔLES (plats, comme attendu) : autre échantillon 90,0° ; même slot +500 échantillons 81,0° ;
  visée d'un autre échantillon 86,8°
- **TEST LE PLUS FORT (indépendant du repère)** : vitesse par différence finie des positions ÷ magnitude
  log/exp décodée de i1 → n=130 597, médiane **1,022** (1,0 attendu), p10 0,841 p90 1,260. Valide d'un coup
  les faces, LES DEUX coordonnées, les signes, l'ordre des axes et la déquantification.
- Faces : 23,8/23,5/3,7/22,9/22,2/4,0 % — les 4 horizontales équilibrées, ±Z rares = signature physique
  d'un déplacement de Spartan. Aucun agrégat de couture.
- **Stride confirmé PAR LES DONNÉES** : max iu = max iv = 292 = N−2, jamais 293 ⇒ N=294 sans oracle.
- **GAIN vs la version communautaire** : en 3D pur (test unitaire p=19, 5000 directions) erreur moyenne
  **0,115° contre 20,139°** = facteur 175. Sur les 145 578 vecteurs réels : écart médian 1,16°, p90 26,5°,
  **max 44,90°** — les 45° annoncés aux coutures sont bien constatés.

**INTÉGRATION** : `Point.H` (cap en degrés) optionnel, SchemaVersion inchangé. Artefact régénéré :
12 739 points portant un cap (43,6 %), 98/99 tracks.

**CRITIQUES RETENUES (verdict PARTIAL, à traiter)**
1. « normes = 1,000000 » et « norme avant normalisation ∈ [1 ; 1,732] » sont des **identités arithmétiques**
   du décodeur, vraies même sur du bruit pur ⇒ ne valident RIEN, à retirer des preuves.
2. Le cliquet « face ≥ 6 » n'a **aucun pouvoir discriminant** (espérance 0,56 sous bruit pur à p=19) ; le
   commentaire d'aim_vector.go qui l'affirme est faux. CORRECTIF : déplacer le cliquet sur
   `iu > N-2 || iv > N-2` (pouvoir mesuré : 1,75 % par échantillon sous bruit).
3. `Point.H` (le champ qui part en production) vient de i21, **le moins validé des deux** (11,1° vs 4,0°).

## [2026-07-25] Catalyst — 3 hypothèses RÉFUTÉES (résultat négatif rigoureux) + morts/joueurs VALIDÉS

### A. MORTS ET ATTRIBUTION DES JOUEURS — validés contre une source indépendante
Idée utilisateur : « le timestamp de mort est dans la data, donc c'est corrélable ». Exact.
`killer_victim_pairs` (DB) = 93 morts horodatées avec victime nommée pour 000d5950.
Corrélation avec les fins de vie décodées (dernière position transmise d'un slot) :
- **92/93 appariées (99 %)**, décalage film↔match = **−3,70 s**, écart **médian 22 ms** (moyenne 44, p90 60).
- Témoin (93 morts tirées au hasard sur la même durée) : 32/93, écart médian 512 ms.
- Pic de décalage FRANC (−8 s : 36 · −6 s : 40 · **−5 s : 92** · −4 s : 92 · −2 s : 36) ⇒ pas du hasard.
⇒ **Les fins de vie SONT les morts** (mesure, plus inférence). Attribution : 92/99 vies nommées,
**8 joueurs distincts, 4 contre 4**. Contrôle sur `match_participants` : morts par joueur
14/14/13/13/11/10/9/9 (base) vs 14/13/13/13/11/10/9/9 (replay) ⇒ **7/8 exacts**, l'unique écart = la
seule mort non appariée. **Ceci résout « l'Étape D » (attribution slot→joueur)** notée non faite.
Artefact : `.ai/re_dump/life_attribution.json`.

### B. LA PLAGE N'EST PAS LE PROBLÈME — c'est le DÉCOUPAGE DE BITS (preuve indépendante de toute plage)
Sur Catalyst, décodé avec le layout de Cliffhanger : pas médian **38,4 wu** (contre 0,0417 sur Cliffhanger,
**×920**), 94,8 % des pas > 35 wu/s, 73 % des points rejetés par les filtres.
**PREUVE DÉCISIVE, en QUANTA BRUTS** (calculés AVANT déquantification, donc insensibles à la plage) :
`|dqx| médian = 1` (X parfaitement continu ⇒ en-tête, slot, offset i0 et les 13 premiers bits sont CORRECTS
sur Catalyst) mais `|dqy| = 2048 = 2^11` et `|dqz| = 4096 = 2^12` — **puissances de 2 EXACTES**, pas du bruit
(bruit uniforme attendu : 2731 et 5461). Signature d'un bit qui bascule ⇒ champs Y/Z lus **décalés**.
Or une plage fausse est une transformation **affine par axe** : elle multiplie les mètres et ne touche
**jamais** |dq|. ⇒ le ×920 ne peut PAS venir de la plage.
Re-découpage des 27 bits Y+Z (sans re-décoder) : (skip 3, wY 13, wZ 11) donne |dY|=|dZ|=1, **facteur 2000**
mieux que le layout livré. Contrôle Cliffhanger : le layout livré n'y est PAS battu ⇒ correct là-bas.
⇒ **La dépendance per-map est PLUS PROFONDE que la plage : le layout de bits d'i0 en fait partie.**

### C. MA MÉTHODE « RÉSOUDRE LA PLAGE PAR LA GÉOMÉTRIE » EST RÉFUTÉE (testée sur le cas connu D'ABORD)
Excellent réflexe de l'agent : avant d'ajuster Catalyst, faire tourner le même critère sur Cliffhanger dont
la plage CE est connue. Résultat : critère à la **vraie** plage = 0,100 ; **permutation XY = 0,226**
(2,3× MIEUX que la vérité) ; offset Z +20 wu = **0,309**. Sur 12 combinaisons d'axes, l'identité arrive
**8e sur 12**. ⇒ le critère est **systématiquement maximal À CÔTÉ de la vérité** : ce n'est pas un oracle.
L'ajustement à 6 paramètres sur Cliffhanger donne fit 0,996 / held-out 0,988 **mais à 44,7 wu RMS de la
vérité** — le held-out temporel est inopérant, la dégénérescence étant SPATIALE. Chiffre Catalyst rejeté.

### D. LA GÉOMÉTRIE EXTRAITE N'A PAS DE PLACEMENT MONDE (cause racine de l'échec d'overlay)
`Position@0x38` est **LOCAL au resource** : médianes des centres de mesh (<60 wu) = X 0,01 / Y 0,01 / Z 0,09,
interquartile ~1 wu ⇒ **la moitié des meshes est centrée à moins d'un mètre de l'origine**. Vérifié aussi sur
Ridgeline (775 meshes, mêmes médianes) ⇒ ce n'est pas propre à Catalyst, c'est une **limite de la chaîne
d'extraction** : la transformation de placement d'instance MANQUE. Voilà pourquoi l'overlay ne collait pas —
ce n'était pas un problème de repère, il n'y a tout simplement pas de placement.
Catalyst : 277 resources / 523 meshes (conforme au handoff). bbox globale X 1602 / Y 1933 / Z 355 wu ;
meshes <60 wu : X 306 / Y 303 / Z 130.
Et la plage ne se déduit d'aucune bbox : spans Cliffhanger 113,21281 / 113,819536 / 137,55112 — ni ronds,
ni puissances de 2 (113,21281/(1/64) = 7245,6) ⇒ **flottants bruts, à LIRE, pas à deviner**.

### CE QUI EST QUAND MÊME ACQUIS
- span_X(Catalyst) ≈ **91 ± 3 wu**, span_Y ≈ **82 ± 4 wu** (calibration physique du déplacement contre une
  carte de référence de plage connue ; même cadence vérifiée, dt médian 16 686 µs sur les deux films).
- Occupation des quanta : Catalyst 84,2 / 89,6 / 94,2 % contre Cliffhanger 44,9 / 48,8 / 8,2 % ⇒ arène
  ~2× plus large à plage égale.
- `BipedPosition.Q [3]uint32` (quanta bruts) exporté ⇒ on peut re-déquantifier ET re-découper sans re-décoder.

### RÉSERVE DU VÉRIFICATEUR (verdict PARTIAL) — À NE PAS REPERDRE
Le rapport affirme « sur Cliffhanger aucun bit constant entre 18 et 44 » : **FAUX**, les bits 31 et 32 sont
constants à 99,998 % (3 exceptions / 171 849) — c'est le motif même qualifié de « marqueur » sur Catalyst.
⇒ **le découpage de Catalyst n'est PAS résolu** : une lecture rivale (X/Y/Z = 15 bits contigus depuis le
bit 5, autorisée par le triplet (15,15,15) de ce_prec_widths) reproduit les mêmes med|dq|. Ce qui est établi :
le layout livré est FAUX sur Catalyst ; ce qui ne l'est pas : quel est le bon.

### PROCHAINE ÉTAPE (périmètre resserré)
Le blocage n'est plus « la plage » mais : (1) déterminer le layout de bits d'i0 par carte — piste : la table
`ce_prec_widths` (triplets de largeurs) indexée par un niveau de précision propre à la carte ; (2) trouver la
transformation de placement d'instance dans le module (sans elle, aucune géométrie exploitable).

## [2026-07-25] Découpage de bits d'i0 — RÉSOLU par carte, lu dans le film — Complété

### Décision technique
Le registre chunk_00 est **bit-à-bit identique** entre Cliffhanger et Catalyst (118 archétypes,
1067 slots, FNV(noms+flags) = a413610cd08e4355 des deux côtés) : l'hypothèse « les largeurs
d'axe viennent des flags du registre » est **réfutée par mesure**. Les largeurs sont une
constante PAR CARTE (bornes du BSP du tag scenario, niveau 16 câblé au site d'appel).

Levier trouvé : les largeurs se **lisent dans le bitstream** par le TAUX DE BASCULE par
position de bit entre enregistrements consécutifs d'un même slot. Un champ quantifié continu
donne une dent de scie (~50 % au LSB, doublement du MSB vers le LSB, effondrement au MSB du
champ suivant). Les trois frontières se lisent directement, sans a priori de largeur.

### Résultats
- Cliffhanger : frontières [18 31 45] → gate 5 + **13/13/14** (= la valeur de référence, retrouvée
  sans ajustement). Catalyst : [20 35 50] → gate 5 + **15/15/15** (i0 = 50 bits, pas 45).
- Départage des lectures rivales (med|dq| ne sait pas le faire) : (a) la rivale « 13/13/14 +
  skip 3 » prédit des frontières en 21/34/45, mesuré 0/4 ; la lecture contiguë prédit 20/35/50,
  mesuré 3/3 avec 5 ordres de grandeur d'écart (bit 19 : 142 567 bascules → bit 20 : 0) ;
  (b) **chaînage** : la structure connue qui suit i0 (2 bits de queue + i1 = [1][1][19 dir][10
  scale], largeurs venues de l'asm FUN_14076d4d0) se superpose point par point entre les deux
  cartes une fois décalée de 5 bits, effondrement compris à +23 = fin du champ direction.
- Vérification croisée : **12 cartes × 4 films = 48 films**, zéro variance intra-carte
  (Streets 12/12/12, Bazaar 17/17/16, Aquarius 13/12/11, Illusion 18/18/17, Behemoth 17/17/15,
  Recharge 18/18/15, Live Fire 13/12/11, Prism 14/13/15, Forest 13/13/13, Chasm 18/18/17).
- Catalyst décodé : 293 837 positions, **0 téléport**, |dq| médian X=2 Y=2 Z=0 (contre
  1 / 2048 / 4096 avant), pas médian 3D ≈ 0,044 wu — même signature physique que Cliffhanger.

### Livré
`filmdec.DetectI0Layout` (i0_layout.go + test) lit le découpage dans le film ; la constante
`bipedAxisWidths = 13/13/14` est **supprimée**. Non-régression : décodage Cliffhanger identique
(sous-ensemble strict du CSV de référence, 0 ligne nouvelle → corrélation des morts 92/93
préservée), tmp_kfworldpos 249/250, `go test ./internal/analysis/...` vert.
Rapport détaillé + schéma : scratchpad `bit_layout.txt`.

### Prochaine étape
Les **bornes** de déquantification (min/max par axe) restent hors ligne indisponibles (tag
scenario, bloc structure-BSP scnr+0x7ac / AABB +0x44). Sans elles, hors 000d5950 seuls les
quanta bruts et les distances relatives sont exacts ; le quantum est encadré à un facteur 2
près (]1/120 ; 1/60] wu). Chemin : parsing du tag scenario, ou une lecture CE par carte figée
dans un JSON versionné.

## [2026-07-26] Bornes monde par carte : la loi de quantification confrontée aux modules du jeu — Complété

### Décision technique
Les bornes de déquantification (AABB du BSP) ne se lisent que dans les `.module` de la carte.
Nouveau paquet `internal/himap` : il localise `world bounds x/y/z` du tag `sbsp` par
MATCHING PAR RANG entre le plugin `sbsp.xml` et la struct-table du tag (aucun offset en dur :
les offsets bougent entre versions de Halo), et n'accepte la déduction que si tous les écarts
entre tag-blocks jusqu'à l'encadrement sont conservés. Deux formes d'en-tête de tag gérées
(`hs+ds == len`, et `hs+ds+rs == len` pour `ctf_bazaar`).

### Résultats observés
- **TEST DÉCISIF 13/13.** `W = min(26, ceilLog2(ceil(60*extent)))` appliquée aux bornes du
  module redonne EXACTEMENT les largeurs mesurées dans les films (DetectI0Layout, profil de
  bascule) sur les 13 cartes dont le module est déductible du seul nom, sur les 3 axes :
  Aquarius 13/12/11, Bazaar 17/17/16, Behemoth 17/17/15, Breaker 13/13/12, Catalyst 15/15/15,
  Chasm 18/18/17, Forbidden 13/12/14, Forest 13/13/13, Fragmentation 17/17/15,
  Highpower 18/19/17, Illusion 18/18/17, Launch Site 17/17/15, Streets 12/12/12.
  Sources totalement disjointes (bitstream du film vs archive de la carte).
- **Cliffhanger = module `ridgeline`**, confirmé indépendamment : ses bornes reproduisent la
  capture live `DAT_14462cbe0[0]` à 2,4e-4..6,9e-4 près, avec un biais systématique vers
  l'extérieur (le runtime dilate l'AABB d'un epsilon). 14e carte concordante.
- **TÉMOIN : 10/364 = 2,75 %** d'appariements croisés réussis (bornes d'une autre carte).
  Probabilité de 13/13 par hasard <= 1,2e-15 même avec le taux le plus permissif observé.
  Second témoin, sur les coordonnées : bornes étrangères -> vitesse médiane 0,39 à 202 u/s
  au lieu de 2,5, quantum hors de la bande ]1/120 ; 1/60].
- **GATE MESURÉ, plus postulé** : `gate = frontière0 - W0(module)` = 5 sur les 14 cartes,
  donc N = 1 (index de région d'1 bit, 2 régions). Aucune carte à N != 1.
  Réserve écrite : 6 cartes sur 14 n'ont qu'un sbsp dans leur module, donc l'opérande de
  ceilLog2 n'est pas le compte de sbsp du module — non tranché offline.
- **Coordonnées monde justes sur 14 cartes.** Trois invariants physiques vérifiés :
  quantum dans ]1/120 ; 1/60] (42/42), vitesse médiane 2,27-2,64 u/s sur les 14 cartes,
  pas médian 0,038-0,088 u.

### Correction de production
`QuantRangeCEBiped` (bornes de Cliffhanger) n'est plus appliquée à toutes les cartes.
Catalogue versionné `data/titles/{slug}/reference/map_quant_bounds.json` (14 cartes) produit
par `cmd/mapquant-build` depuis les modules ; `PathResolver.MapQuantBoundsPath`.
`ScanFilmOptions.WorldRange` obligatoire (sinon `ErrUnknownMapBounds`), `BipedPosition.HasWorld`,
`replay.Options.WorldRange` obligatoire, `replay-build --map <carte>`. Live Fire / Recharge /
Prism restent ABSENTES du catalogue (module non établi) : refus propre, pas de valeur fausse.
Le filtre téléport en m/s est de fait recalibré (il opérait sur une échelle fausse hors
Cliffhanger : facteur 0,38 sur Catalyst).

### Modèle 6+L
`quantAxisWidth(L) = min(26, 6+L)` est la table PAR DÉFAUT (boîte monde ~40000), pas la table
par région : vérifié terme à terme, la forme fermée est exacte pour L=0..24 et pour les deux
lectures de la boîte (+-19968 / +-20000). `TestQuantAxisWidthFormula` était TAUTOLOGIQUE
(6+l == 6+l) : il est réécrit contre la formule complète. Nouveau
`TestRegionAxisWidthIsNotDefaultTable` ancre la table par région sur 4 cartes. Doc et
garde-rail portent désormais le périmètre et deux réserves écrites (source de L pour les
composants non-position).

### Non-régression
Cliffhanger 171 826 positions (identique), tmp_kfworldpos 249/250 MISMATCH=0 bipeds 8/8,
artefact rejeu 99 tracks / 29 221 points, `go test ./internal/analysis/... ./internal/domain/...`
vert, `go build ./...` vert.

### Prochaine étape
Résoudre le module de Live Fire / Recharge / Prism (lire le tag `levl`/`scnr` des variantes ;
`sgh_interlock` ne contient aucun sbsp). Puis : provenance de la 2e région de compression,
chemin `useDefault=1`, faux positifs résiduels du balayage.
Rapport détaillé : scratchpad `bornes_monde.md`.

## [2026-07-26] Événements de tir décodés hors ligne et portés au rejeu 2D — Complété

### Décision technique
Le « fire event » de la communauté et le « record de dégât 0xd2 » du projet sont **le même
record** : l'event de type **105** du paquet film de type 0. `payload[0] >> 1` donne le type
en O(1) — le balayage bit-à-bit par « marqueur 11 bits » est mort (ce marqueur n'était que
l'en-tête du record). Décodage porté dans `filmdec.ScanFilmFireEvents` : tireur (bit 36,
5 bits, valeur = slot×2), arme 64 bits (bits 44 et 76), 5 drapeaux (108..112), et la
**visée cubemap 30 bits au bit 113 uniquement sur le chemin « record vide »**. Hors de ce
chemin le champ existe mais après des boucles de largeur runtime : on ne le lit PAS.

Pont tireur → biped, **film-seul, sans base de données** : la visée du record est la MÊME
grandeur que le cap de visée du biped (composant i21, 12 bits) au même instant. Deux champs,
deux largeurs, deux composants — l'égalité n'a aucune raison d'exister si le décodage est
faux. On ne retient que les désignations non ambiguës (meilleur < 2°, deuxième > 20°), on
vote slot → index de joueur, puis les autres events de l'index héritent de l'origine.

### Résultats observés (deux films, 000d5950 Cliffhanger et 01e1f945 Catalyst)
- Reproduction exacte de la spec Ghidra : 519 records longs / 313 courts sur 000d5950,
  distribution de l'attaquant 49/71/81/58/82/27/59/92 — chiffre pour chiffre.
- **TÉMOIN 1 (tir mortel → victime)** : angle médian **2,7°** (3D) et 1,9° (2D) sur les tirs
  mortels à visée décodée, contre **56,4°** vers un joueur vivant tiré au hasard.
  Catalyst : 5,6° contre 105,7°. Échantillons petits (7 et 10) car la visée n'est lisible
  que sur ~19 % des records.
- **TÉMOIN 1 bis (n large)** : angle au plus proche ENNEMI, médiane **7,6°** (Cliffhanger,
  n=78) et 15,1° (Catalyst, n=292) ; témoins : coéquipier 42,7 / 72,4°, **visée isotrope
  aléatoire 65,0 / 74,2°** (4 et 3 % sous 15°, contre 64 et 49 %).
- **TÉMOIN 2** : l'attribution par corrélation des morts (fins de vie × killer_victim_pairs,
  90/93 appariées à 33 ms, témoin aléatoire 10/93) est confirmée par deux méthodes
  disjointes — géométrie **8/8 index**, et « un joueur mort ne tire pas » (dates de la base
  seules) **1,0 % de violations** contre 9,7 % attendus sur une case fausse.
  Le pont film-seul redonne le même tireur sur **149/149** events (Cliffhanger) et
  897/1066 (Catalyst) ; 0 slot à votes mixtes sur Cliffhanger, 2/39 sur Catalyst.
- **TÉMOIN 3 : PARTIEL, dit tel quel.** Catalyst : corrélation de Pearson **0,85** avec
  shots_fired (0,95 avec shots_hit), ratio events/shots_fired 0,82 médian. Cliffhanger :
  **0,24 seulement** (ratio 0,24, étendue 0,10-0,41). Le film de 000d5950 porte 3× moins
  d'events 105 que celui de Catalyst à volume de paquets comparable (832 vs 2535) alors
  qu'il porte 4× plus d'events de type 97 (825 vs 187) : une partie du trafic de dégât y
  passe visiblement par un autre type. NON RÉSOLU.

### Livré
- `filmdec/fire_events.go` + test d'ancrage des offsets de bits (`fire_events_test.go`).
- `replay/shots.go` (pont film-seul) + `replay/shots_test.go` (rattachement et **refus**
  en cas d'ambiguïté) ; `ReplayDocument.Shots` (omitempty, SchemaVersion inchangé) ;
  `BuildFromPositions` prend désormais les events ; `replay-build` journalise les tirs.
- Artefacts régénérés : 000d5950 = 99 tracks / 147 tirs (44 avec direction),
  01e1f945 = 113 tracks / 1243 tirs (245 avec direction). Démo `replay_demo.html` :
  bascule « Tirs », trait bref rémanent 0,6 s dans la direction visée, éclat pointillé
  quand la direction n'est pas lisible.

### Ce qui n'est PAS acquis
La **VICTIME** n'est pas décodée (liste des cibles, largeur runtime) : le trait part du
tireur, il ne relie pas deux joueurs. Il n'y a pas de notion touché/raté à afficher — le
record n'existe que si un dégât est appliqué, donc **tous** les tirs publiés ont touché.
Couverture du rattachement : 30 % (Cliffhanger) à 57 % (Catalyst) ; le reste est OMIS.

### Prochaine étape
Résoudre la largeur de `FUN_1406D3140` (table runtime `DAT_1451F98D0`) pour lire la visée
des ~80 % de records restants et atteindre la liste des cibles (la victime). Explorer le
type d'event 97, sur-représenté dans 000d5950, pour expliquer l'écart du témoin 3.
Rapport détaillé : scratchpad `tirs.md`.

## [2026-07-26] Objectifs de carte depuis les variantes UGC (.mvar) — Complété (hors téléchargement)

### Décision technique principale
Les objets d'objectif ne sont ni dans le scénario ni identifiables par `type_id` : ils sont
désignés par des **labels de mode de jeu** stockés dans le sac de propriétés #8 de chaque
objet du `.mvar` (`[8][0].0[9]`), sous forme de hash int32, avec l'index d'équipe en
`[8][0].0[8]`. La fonction de hachage a été craquée : **murmur3 x86_32, seed 0**, sur le nom
snake_case — ancre vérifiable `LabelHash("stockpile_socket") == 2110778921`, valeur lue telle
quelle dans `ctf_breaker.mvar`.

Nouveau package `internal/analysis/replay/mapvar` (lecteur Bond CompactBinary v2 générique +
grammaire métier + table de labels + classification en rôles). Nouveau CLI
`cmd/mapobj-build` (auth ADR 0023 → Discovery UGC → blob → parse → JSON figé via
`PathResolver.MapObjectivesPath`, ajouté à `registry.go`). Le rejeu reste 100 % hors ligne.

### Résultats observés
- Parse complet, zéro octet résiduel, sur les 3 `.mvar` en dépôt (439 / 453 / 443 objets).
- 22 labels résolus ; `flag_spawn`, `flag_delivery`, `stockpile_socket`, `stockpile_navpoint`,
  `strongholds_zone`, `extraction_zone`, `oddball_spawn` portent un rôle. Les `*_include` /
  `*_exclude` sont des filtres de mode, PAS des rôles.
- Catalogue figé : Breaker 26 objectifs, Cliffhanger 14 (dont le drapeau NEUTRE, `team_index=-1`).
- **La prémisse « Breaker est symétrique » est FAUSSE** (mesuré : ≤1,6 % d'appariement sur les
  3 transformations, aucun pic dans la recherche de centre par vote). Le témoin de symétrie
  demandé est inapplicable sur les cartes disponibles.
- **Témoin de remplacement — co-localisation** : `flag_delivery` et `flag_spawn`, portés par
  des `type_id` différents, coïncident à 0,0039–0,0148 m. Sur Breaker, ce sont **les 2 seules
  paires sous 0,02 m parmi 96 141 paires**. Témoin négatif : 38,51 m de médiane entre objets
  au hasard.
- **Témoin séquence** : les 10 `stockpile_socket` reproduisent 10/10 la séquence bleu/rouge de
  la table de noms (p = 1/252) → établit team 1 = bleu, team 0 = rouge.
- **Repère validé** : 100 % des objets `.mvar` et 100 % des positions joueur dans les bornes
  BSP ; distance joueur → objet le plus proche, médiane 1,40 m (16,31 m en croisant avec une
  autre carte = témoin négatif). Aucune transformation manquante.

### Échec assumé
Téléchargement UGC impossible : `AADSTS7000215` (secret client Azure invalide pour l'app
`39829f7a`) + refresh tokens marqués `revoked` le 2026-07-25T23:23Z, **avant** toute
tentative. Aucune re-capture, aucun contournement. Le catalogue ne couvre que les 2 cartes
déjà acquises.

### Conclusion / prochaine étape
Renouveler `SPNKR_AZURE_CLIENT_SECRET` dans `.env.local`, puis
`go run ./cmd/mapobj-build --player <GT> --all` pour figer les ~180 cartes. Ensuite câbler le
service de rejeu sur `map_objectives.json`. Non vérifié faute de fichier : la concordance avec
les 816 callouts (absents de ce dépôt). Rapport détaillé : scratchpad `objectifs_ugc.md`.

## [2026-07-26] Récupération UGC sans jeton, callouts FR au POC, apparitions corrigées — Complété

### Décision technique principale
La récupération des `.mvar` ne passe PAS par `discovery-infiniteugc` (401, jeton Spartan requis)
mais par la **page publique Waypoint** (`www.halowaypoint.com/halo-infinite/ugc/maps/{assetId}`,
JSON complet dans `__NEXT_DATA__`) puis le **blob anonyme** `blobs-infiniteugc`. Zéro
authentification, donc zéro couplage à la santé des jetons — ce qui a permis d'aboutir alors
que le secret client Azure est invalide (`AADSTS7000215`) et 3 jetons révoqués.

### Résultats observés
- **120 cartes / 123** récupérées (3 échecs : 2 assets supprimés, 1 nom de fichier invalide).
- **34 cartes parsées, 597 objectifs**, 7 rôles : `flag_spawn` 162, `extraction_zone` 161,
  `flag_delivery` 90, `strongholds_zone` 63, `stockpile_socket` 53, `oddball_spawn` 50,
  `stockpile_navpoint` 18. Catalogue : `data/titles/halo_infinite/reference/map_objectives.json`.
- **Cliffhanger = ridgeline confirmé une 4e fois** : l'asset officiel liste `ridgeline.mvar`.
- **Callouts FR câblés au POC** (28 sur ridgeline), affectation en distance 3D — en 2D les
  versions haute et basse d'une même zone se confondent.
- **Apparitions corrigées** : 4 vies sur 8 démarraient au premier pas, jusqu'à 35 s après
  l'apparition réelle (cas du joueur inactif). Report arrière de la première position, tracé en
  cercle évidé tant qu'il est déduit. Correctif d'affichage : la vraie correction passera par
  les 26 images-clés (une / 20 s), qui portent slot + génération + position.

### Réserves écrites
- **Aucun rôle VIP** dans les 597 objectifs. Hypothèse de l'utilisateur, cohérente : le VIP est
  un joueur DÉSIGNÉ à l'exécution (cf. `ApplyVIPPlayerFX`), donc côté composants du film, pas
  côté placement statique. Non vérifié.
- `.ai/re_dump/mapvar/` (77 Mo) volontairement ignoré : re-téléchargeable sans jeton, le
  catalogue extrait fait foi.
- Doc inversée à corriger : `internal/platform/halo/discovery_client.go:28` affirme que l'API
  ne nécessite pas d'auth Spartan — 401 mesuré.

### Conclusion / prochaine étape
Le décor structurel (`sbsp` `instanced geometry instances` @0x1A4, stride 0x140) reste le plus
gros manque visuel : on dessine 382 accessoires de 0,25 m² couvrant 3,4 % de la carte.

## [2026-07-26] Structure de carte depuis le sbsp — Complété, verdict PARTIAL

### Décision technique principale
Le décor ne vient PAS de la variante Forge (382 accessoires, 0,25 m² médian, 3,4 % de couverture)
mais du bloc `instanced geometry instances` du tag `sbsp` (@0x1A4, stride 0x140). L'AABB monde
(@0x7C) est lue **telle quelle** — aucune transformation appliquée au livrable, la transformation
vecteur-ligne ne sert qu'à l'oracle.

**L'agent a contredit ma consigne, avec raison** : j'imposais `deploy/ds/`, or le build serveur
dédié est dépouillé du rendu (bloc cible vide, il ne garde que `instanced physics instances`
@0x1E4, 3928 entrées, pas 0xC0 et non 0xB0 comme le plugin). `deploy/pc/` donne 10 357 entrées au
pas 0x140 exact, et ma raison de l'éviter (compagnon `_hd1`) était levée par son propre correctif.

### Résultats observés
- 382 -> **10 223 emprises** ; aire médiane 0,25 -> 3,28 m² ; couverture de la zone de jeu
  3,4 % -> **100 %**. Anti-tautologie : en plafonnant à 50 m², 85 % reste couvert.
- **Témoin de surface** (|dz| au-dessus de la surface la plus proche sous le joueur, n=100 375) :
  réel 35,1 % à 1 cm / 80,6 % à 5 cm ; rotations et translations 5-10 % / 12-39 % ; hasard mesuré
  3,7 % / 11,9 %. Témoin LE PLUS DUR (altitudes permutées, emprises 2D conservées) : 19,4 / 37,5 %,
  soit un facteur **x1,8 seulement** — c'est ce chiffre qui doit être cité, pas le x9,5.
- himodule : 2 correctifs. `any/catalyst` 4,0 % -> 99,9 % d'entrées extraites ; `pc/ridgeline`
  0 (erreur) -> 98,8 %. La règle prescrite ne couvrait que `any/` : `ds/` et `pc/` ont
  dataSize == 0 et exigeaient une seconde branche.

### Réserves écrites (verdict adversarial PARTIAL)
- **Revendication fausse corrigée** : « 59 022/59 022 respectent l'encadrement sphère/AABB » est
  FAUX. 100 % = borne basse SEULE ; l'encadrement complet tient à 70,1-95,0 % contre 4,7-16,5 %
  pour le témoin. La comparaison n'était pas appariée (deux bornes pour le témoin, une pour le réel).
- Le contrôle « convention transposée 36,0 vs 37,9 % » n'existe pas dans le code livré : le
  paramètre `transposed` est appelé exclusivement avec `false`. Chiffre non reproductible.
- Les mesures himodule avant/après n'ont **aucune sortie brute archivée**, contrairement au témoin
  et à l'oracle qui ont leurs transcripts.
- **Ne se généralise pas** : couverture ridgeline 100 %, sgh_streets 100 %, mais catalyst 49 %,
  forest 41,5 %, ctf_aquarius 40,3 % — la structure y vit dans le maillage de rendu NON instancié.
  Acquis pour la carte du POC 1, pas pour celle du POC 2.
- Le lien instance->maillage reste non résolu : `meshRef` @0x3C est décodé (fourCC `rtgo`, id de tag
  @+8) mais `(meshRef, meshIndex)` n'identifie pas complètement la géométrie (49 à 87 % d'accord
  sur des paires de même orientation). On publie donc des boîtes, pas des maillages — d'où un
  rendu entièrement aligné sur les axes, sans courbe.

### Conclusion / prochaine étape
Les deux correctifs de chaînage de l'état de jeu (`weapon-state-ammo` polarité inversée,
`biped-spartan-ability-energy` R(3)+7 bits par charge) : ils faussent ce qu'on décode déjà.
Puis santé/bouclier, dont les désérialiseurs sont déjà bit-exacts — seules les valeurs sont jetées.

## [2026-07-26] L0 — deux correctifs de chaînage du record bipède, mesurés et attribués — Complété

### Décision technique principale
Les deux bugs de consommation annoncés sont réels, **re-vérifiés au désassemblage** avant
application (le bug n°2 était précisément une fausse certitude écrite dans notre code) :
- `weapon-state-ammo` (`FUN_140ea1018`) : `140ea10a9 JZ 140ea1174` — le jeu lit les 12 bits
  quand la 2e porte vaut **0**. Notre port lisait quand elle valait 1. Écart de ±12 bits à
  chaque occurrence, jamais nul.
- `biped-spartan-ability-energy` i56 (`FUN_140fc1410`) : `R(3)` masque **puis 7 bits par
  charge armée** (bloc froid externalisé `0x14246a410`, que Ghidra masque avec
  `Removing unreachable block`). Le commentaire « Total: 3 bits, CONFIRMED bit-exact » était
  FAUX ; il est remplacé par un avertissement explicite « le décompile ment ici ».

### Résultats observés
- **Le compteur de référence 86019/66106 n'est PAS reproductible** (une seule occurrence au
  dépôt, entrée du 2026-07-12 ; le décodeur a changé de génération, 152 125 records alors vs
  24 469 aujourd'hui sur le même film). Compteur **de même nature** reconstruit :
  `cmd/tmp_l0witness` (jetable) — clean/desync bipèdes en décodage séquentiel + histogramme
  du composant de desync + **atteignabilité des sites corrigés**.
- **Témoin POSITIF, pas neutre** : ti=35 clean **10 895 -> 23 485 (+115,6 %)** ; slots
  512-519 desync **6 374 -> 876 (−86,3 %)** ; desync dominant `i00 game-engine-team-mapping`
  **6 366 -> 792**.
- **Atteignabilité mesurée** (sans elle un compteur immobile serait ininterprétable) :
  `weapon-state-ammo` atteint 6 997 -> 15 057 fois ; gate2 relue DIRECTEMENT dans le tampon
  (donc indépendante de la polarité du déser, anti-tautologie) : gate2==0 dans 6 276/10 974
  cas lisibles (57 %), donc distribution franchement mixte.
- **ABLATION — le résultat est contre-intuitif** : i56 seul = 23 165 clean (96 % du gain) ;
  **ammo seul = 10 878, soit NEUTRE voire −0,16 %** ; les deux = 23 485. L'ammo ne paie
  qu'en présence du correctif i56 (+320 clean, −304 desync vs i56 seul). La spec disait
  « préalable, pas amélioration » — la mesure nuance : préalable dont l'effet n'est
  observable qu'en aval du déblocage i56.

### Dette traitée
- `ConsumeWeaponStateTypeInfo` **supprimée** (version fausse : il lui manquait le `R(32)` de
  `FUN_14080d6f0`, sous-lecture de 32 bits ; zéro appelant, zéro test), avec son type
  `HeldWeapon`, sa queue `consumeHeldWeaponTail` et `consume1407f2494` — tous doublons de la
  version canonique `consumeWeaponStateTypeInfoVariant` (components_object.go).
- **Trois** docs inversées corrigées sur i5 `object-shield-vitality` (deux dans traverse.go,
  une découverte dans components_object.go qui contredisait le code écrit sous elle) :
  `FUN_140d50cbc` n'utilise que des largeurs littérales 8/16/12 et des bornes `.rdata`
  (`DAT_143cd893c` = 4.0f lu en mémoire). Aucune précision runtime. L'ordre non séquentiel
  des 4 drapeaux (+0x66/+0x67/+0x69/+0x68) est désormais documenté pour qu'on ne le
  « corrige » pas.

### Non-régression
Positions **171 842 / 99 slots — CSV identique au bit près** (MD5 52acb8cb) ; artefact de
rejeu **identique au bit près** (MD5 876b924f : 99 tracks, 29 221 points, 147 tirs, 382
géométrie, 10 223 emprises) ; `tmp_kfworldpos` sortie strictement identique (249/250,
MISMATCH 0, 8/8) ; `go build ./...`, `go vet`, `go test ./internal/analysis/...` verts.
Les positions ne bougent pas parce que `ScanFilmBipedPositions` ancre chaque record par son
en-tête et lit i0 sans jamais parcourir la boucle jusqu'à i30/i56 — **mesuré, pas argumenté**.

### Ce qui a bougé, et pourquoi (assumé)
Les dead-state lus par le **walk séquentiel** passent de 55 à 157 signatures distinctes
(+103 gagnées, −1 perdue) : le walk atteint 2,2x plus de records bipèdes, il lit donc plus
d'i11. C'est le cas prévu « un correctif de chaînage peut légitimement changer des données
en aval ». La corrélation validée **92/93 à 22 ms** ne passe PAS par ce chemin : elle vient
des fins de vie des trajectoires, dont le CSV est byte-identique.
**Non refait, dit franchement** : la corrélation contre `killer_victim_pairs` en base n'a pas
été rejouée (l'outil qui avait produit `life_attribution.json` n'existe plus au dépôt).
L'identité au bit près de son entrée est l'argument, pas un 92/93 réaffiché.

### Découverte notée, NON traitée (hors périmètre)
`consume1407f0550` (unit_weaponstate.go, encore utilisé par components_object_state.go:150)
et `consumeWeaponAttachmentBlock` (components_object.go) portent tous deux `FUN_1407f0550`
avec des grammaires DIVERGENTES (déquant W=12 vs W=8 ; compteur `R(32)` vs `R(6)+1`). L'un
des deux est faux — à arbitrer avant de publier des valeurs déquantifiées de cette famille.

### Conclusion / prochaine étape
Lot L1 (santé i4 / bouclier i5) : les deux désérialiseurs sont bit-exacts ET vérifiés, et le
walk atteint 2,2x plus de records. Reste `DequantEndpoint` (`FUN_1406d84b4`, distinct de
celui de `quantize.go`) + son test d'extrémités, dont le détecteur est
`q=127 (santé) -> 0.0 exactement`. Rapport détaillé + schéma : scratchpad `etat_correctifs.md`.

## [2026-07-26] Fil des éliminations, score en direct, et écart POC / production — Complété

### Décision technique principale
Trois items du bloc Notion « Replay 2D » sont livrables SANS décoder le film : le fil de tués,
les médailles et le score viennent de la base (`killer_victim_pairs`, `highlight_events`), pas
du fichier de film. C'est ce qui a permis d'avancer pendant que Go était occupé.

### Résultats observés
- **Les médailles sont DÉJÀ datées** : `highlight_events` porte 44 événements `medal` horodatés
  avec leur nom (*No Scope*, *Odin's Raven*, *Yard Sale*…). Correction d'une affirmation
  antérieure : j'avais annoncé qu'il faudrait « les dater par corrélation, un chantier en soi ».
  Faux — aucune corrélation n'est nécessaire. Elles ne sont pas dans le FILM (0 composant), mais
  elles sont en base avec leur instant.
- **Calage** : décalage de −3,7 s entre l'horloge des événements et celle du film ; écart médian
  tir mortel → fin de vie = **22 ms**, retrouvé ici par un chemin indépendant de la corrélation
  qui avait nommé les joueurs. 42 médailles sur 44 se rattachent à un tué.
- **Score en direct** : calculé en cumulant les tués par équipe, il converge sur **43 – 50**,
  exactement le score officiel de `match_registry`. Témoin fort : s'il tombe juste à la fin, le
  calage et l'attribution d'équipe sont bons sur toute la durée.
- Vitesse ×2 : existait déjà (1×/2×/4×/8×).

### Réserve : mise en page
Première version du fil en surimpression sur la carte, puis sorti en colonne à droite à la
demande de l'utilisateur. **Bug introduit et corrigé** : avec `align-items: stretch`, la hauteur
de la ligne vaut le MAXIMUM des colonnes — un fil qui s'allonge étirait donc la carte. Correctif :
contenu de la colonne en `position: absolute`, elle n'a plus de hauteur propre et ne peut plus
tirer la ligne. Le fil garde toutes les entrées et défile ; le retour en haut ne se déclenche que
si l'utilisateur y était déjà (sinon consulter un frag ancien devient impossible en lecture).

### ÉCART POC → PRODUCTION, chiffré (question explicite de l'utilisateur)
Le document produit par `cmd/replay-build` contient DÉJÀ : `tracks` (99), `shots` (281),
`structure` (10 223), `structureBounds`, `bounds`, `frameIntervalMs`, `durationMs`.
Quatre calques n'existent QUE comme injection manuelle dans la démo :
  1. **callouts** (28) — le CSV vit dans le scratchpad ; il manque un référentiel versionné par
     carte, sur le modèle de `map_objectives.json` et `map_quant_bounds.json`.
  2. **objectifs** (14) — le catalogue `map_objectives.json` EST versionné ; il manque seulement
     le câblage vers le document.
  3. **fil + médailles** (95) — vient de la base ; relève de la couche API, pas du document.
  4. **noms et équipes** — issus de `life_attribution.json`, non câblés.
Côté web, `features/match-replay/` (ReplayCanvas.tsx, replayDraw.ts, replayLogic.ts) est
ANTÉRIEUR à la structure et aux tirs : le rendu de la démo (canvas vanille) ne se transpose pas
tel quel. La réponse honnête à « le code de l'artefact suffit-il ? » est NON pour le rendu,
OUI pour le pipeline de données à 2 calques sur 6.

### Conclusion / prochaine étape
L'implémentation de l'état de jeu tourne (correctifs de chaînage rendus, extraction des valeurs
en cours). Commit différé volontairement : l'arbre de travail contient ses fichiers en cours
d'écriture (`capture.go`, `vitality.go`, `quantize_endpoint.go`), dont trois ne passent pas encore
`gofmt`. Commiter maintenant figerait un état intermédiaire.

---

## [2026-07-26] Lot L1/L2/L9 — ouvrir la boîte : santé, bouclier, réapparition, horloge

**Statut** : Complété (bouclier livré au rendu ; santé, réapparition et horloge décodés mais
NON rendus, témoins insuffisants — dit explicitement dans le POC).

### Décision technique principale
`DequantEndpoint` (`quantize_endpoint.go`) porte `FUN_1406d84b4`, **distinct** du déquantificateur
de `quantize.go`. Les 113 instructions `0x1406d84b4..0x1406d8647` ont été relues au désassemblage
pendant cette session — pas reprises du handoff, précisément parce que le paquet a déjà porté un
« CONFIRMED bit-exact » faux. La formule du handoff est **confirmée**, y compris la règle du point
milieu en double (`143cd8910 = 0.5d`) qui rend `santé q=127 -> 0.0` EXACT : sans elle la branche
`endpointExact` donne 5,96·10⁻⁸ (mesuré par `TestDequantEndpointMidpointNeedsTheExclRule`). C'est
le détecteur exigé par la spec, et il passe.

Extraction de valeur sans casser le contrat « sauteur de bits » : chaque `decodeXxx(br) T`
(`vitality.go`) détient la grammaire, chaque `consumeXxx` s'y réduit. Une couche `capture.go`
intercepte les 4 composants porteurs de valeur avant le dispatch et les remonte via
`CompResult.Payload`. L'étiquette apparaît dans deux `switch` : garde-rail
`TestCaptureConsumesSameBitsAsDispatch` (500 tirages aléatoires × 4 composants) — une divergence de
bits entre les deux fait échouer le test.

### Résultats observés
- **BOUCLIER i5.** RECTIFICATION (audit adversarial du 2026-07-26) : le rapport de **13,0**
  écrit ici était FAUX — il comptait les échantillons porteurs de BOUCLIER au numérateur et ceux
  porteurs de SANTÉ au dénominateur, mesurant donc la co-présence de deux composants, pas le
  bouclier. Sur la bonne population : P(bouclier nul | 500 ms avant une mort connue) = **50,49 %**
  (924/1 830) contre **38,18 %** (6 939/18 176) chez un vivant à plus de 5 s d'une mort, soit un
  rapport de **1,32x — FAIBLE, publié tel quel**. Cause mesurée : le film ne réplique le bouclier
  que lorsqu'il CHANGE, donc une mesure de bouclier est déjà une mesure de combat et le bruit de
  fond est haut. Les instants de mort viennent des fins de vie des trajectoires, pas du bouclier.
  Ce qui porte le rendu est le **témoin de forme** : **27 404 / 27 404** quanta dans [0, 64] =
  exactement la plage d'un bouclier standard, contre 25,4 % attendus d'un champ uniforme.
  RÉSERVE : le `p < 10⁻⁴` par permutation est une PSEUDO-RÉPLICATION — il rebat les étiquettes sur
  des échantillons individuels alors que l'unité d'échantillonnage est la VIE (20 006 mesures pour
  99 vies sur 8 slots, fortement autocorrélées). Ce p est surestimé de plusieurs ordres de grandeur
  et ne doit pas être cité.
- **SANTÉ i4 — le témoin DEMANDÉ est inconstructible.** `q=0` n'apparaît JAMAIS (0/113 et 0/789) :
  le rapport est 0/0. Le décodage est pourtant crédible (974/974 quanta dans la moitié positive
  contre 49,6 % au hasard ; médiane 0,79 vivant -> 0,55 avant la mort, p < 10⁻⁴). Mais couverture
  **0,6 %** -> non rendu.
- **RÉAPPARITION ti=5 i1 — témoin faible.** 17 occurrences, 13 démarrages ; écart médian au décès
  connu 801 ms contre **1 403 ms** pour des morts tirées au hasard. 1,75× sur 13 points : insuffisant.
- **HORLOGE ti=0 i5 — témoin en échec.** Pente 0,977 / 11,16 s·s⁻¹, R² = 0,0001 / 0,016, valeurs
  jusqu'à 9 415 s pour un match de **496 s** (`duration_seconds`, lecture seule). Non rendue.
- **Recharge du bouclier — témoin en échec** : 4 suites croissantes réelles contre 7 pour le même
  échantillon dont l'ordre a été mélangé DANS chaque slot. Aucune animation de régénération.

### Effet aval MESURÉ par ablation (i3/i4/i5 retirés du balayage)
Positions **inchangées** (29 221 points, 99 tracks). Couverture du cap de visée **43,7 % -> 52,6 %**
(+15 207), qualité **inchangée** : écart médian cap↔déplacement 16,3° -> 16,4° (témoin réapparié au
hasard 77,2° / 79,2°). Conséquence : tirs rattachés **147 -> 281**. Si les nouveaux caps étaient du
bruit, la médiane aurait dérivé vers le témoin — elle n'a pas bougé de 0,1°.

### Découvertes notées, NON traitées (hors périmètre)
1. `replay.voteSlotOwners` (shots.go) : le départage `n == bestN && idx < best` avec `best`
   initialisé à −1 ne se déclenche jamais — en cas d'égalité le gagnant dépend de l'ordre
   d'itération d'une map Go. Non reproduit sur ce film (3 builds, MD5 identique), mais latent.
2. `TestContractRoutesDocumented` échoue AVANT ce lot (2 routes chi non documentées :
   objective-events, positions). Sans rapport avec filmdec/replay.
3. Le commentaire historique « i5 coûte 25/27/39/51 bits » était faux de 4 bits (les quatre R(1)
   de queue) ; le CODE les a toujours lus. Corrigé, et figé par test.

### Conclusion / prochaine étape
Le bouclier est la première valeur d'état de jeu affichée, et la seule qui ait passé la barre.
Prochaine étape crédible : L3 (munitions) — dont le témoin comptable
`delta(chargeur) + delta(réserve) == 0` est plus fort que tout ce qui a été mesuré ici.

## [2026-07-26] Colonnes d'équipe, armes nommées, et rectification du témoin bouclier — Complété

### Décision technique principale
Le POC gagne deux colonnes d'équipe (bouclier, vie, dernière arme, K/D) alimentées par les champs
`sh`/`hp` que le lot précédent a fait remonter, plus le nommage des armes emprunté à la branche
`feat/filmdec-killweapon` (`.ai/re_dump/nm_tag_to_weapon.txt`).

### Résultats observés
- **Les identifiants d'arme de nos tirs correspondent directement à la table de la branche
  killweapon** : les 32 bits de poids fort (`2b1824d5`, `48c19d2d`, `0a1992bc`, `9d6aaed2`…) sont
  les variantes qu'elle liste. 52 variantes nommées, **65 tirs sur 147 nommés (44 %)** dans la démo.
  Aucun travail de RE à refaire.
- **Les assistances par tué sont DÉJÀ décodées sur cette branche** : grammaire
  `killer(E5) victim(E5) R32 R1 assist(E5) R32`, validée **31 décodées contre 30 en API**, avec
  recoupement nominatif (JGtm 1=1, RaiiZeNBack 3=3). En base, seuls les AGRÉGATS existent (équipe 0 :
  6 assistances, équipe 1 : 11) — pas de datation par tué. Le fil de tués attendra donc la fusion
  de cette branche plutôt qu'une réimplémentation.
- 4 620 points portent bouclier et vie dans l'artefact.

### RECTIFICATION issue de l'audit adversarial
Le rapport **13,0x** du bouclier, publié dans l'entrée précédente, était FAUX (numérateur =
échantillons de bouclier, dénominateur = échantillons de santé : il mesurait la co-présence de deux
composants). Le bon rapport est **1,32x** (50,49 % contre 38,18 %), FAIBLE et publié tel quel.
L'agent l'avait rectifié dans son rapport mais **le chiffre rétracté subsistait dans deux endroits** :
un commentaire de `internal/analysis/replay/build.go` et l'entrée de ce journal. Les deux sont
corrigés. Réserve supplémentaire notée : le `p < 10⁻⁴` par permutation est une PSEUDO-RÉPLICATION
(échantillons individuels alors que l'unité est la vie) et ne doit pas être cité.

### Réserves écrites
- L'arme affichée est la **DERNIÈRE UTILISÉE**, pas l'arme en main : le composant d'arme portée
  n'est pas décodé (lot L4, non fait). L'intitulé de la colonne le dit.
- Le compteur de réapparition (témoin 1,75x sur 13 observations) et l'horloge (R2 = 1e-4, aucun
  recoupement avec `duration_seconds`) ont ÉCHOUÉ leur témoin et ne sont **pas rendus**.
- Découverte notée, non traitée (règle « zéro fix hors périmètre ») : `consume1407f0550` et
  `consumeWeaponAttachmentBlock` portent la même fonction FUN_1407f0550 avec des grammaires
  DIVERGENTES (W=12 vs W=8, R(32) vs R(6)+1). L'un des deux est faux — à arbitrer avant de publier
  des valeurs de cette famille.

### Conclusion / prochaine étape
Maillages réels (formule `v_monde = position + bx·forward + by·left + bz·up`, bornes du tag `rtgo`) :
c'est ce qui décidera des critères d'acceptation de l'utilisateur — le fer à cheval en anneau et la
zone sud reliée par deux ponts.

## [2026-07-26] Loadout au keyframe joint au joueur — témoin croisé passé sur le film Fiesta

### Décision technique principale
La chaîne de composants NE mène PAS au loadout au keyframe, et c'est mesuré, pas supposé :
les ancres de `WalkKeyframeWorld` sont exactes (écart 0 avec `biped_record_offsets.txt`), la
largeur RÉELLE d'un record bipède vaut 2785-2816 bits, et la traversée complète (default-state
+ porte has-components + masque + boucle) n'en consomme que 229-1588 avant de désync à
`i57 biped-spartan-ability`. Les 54 records « propres » sont des FAUX-PROPRES : leur masque lu
vaut `gate=0,count=0` (vide). Preuve sémantique : `i43 weapon-state-type-info` n'est « présent »
que dans 15/184 records alors que 8 joueurs portent 2 armes à chaque keyframe.
Le loadout est donc obtenu par une autre voie, VÉRIFIABLE : **ancrage de l'identifiant de
famille (high-32 du weapon-id) dans la charge utile du keyframe, puis attribution au record
qui le contient** via les bornes déjà validées (249/250 entités, 8/8 bipèdes). C'est exactement
le verrou que `origin/feat/filmdec-killweapon` déclarait non résolu (« on a les 8 loadouts mais
pas QUI a quoi ») : la borne donne le slot, le slot donne le joueur.

### Résultats observés (000d5950 Cliffhanger, Fiesta)
- **Ancrage** : 911 occurrences de famille sur 29 997 624 bits contre **0,52 attendue par pur
  hasard** (74 familles / 2^32) — ~1750x. Réparties ti=35 bipède **495**, ti=42 arme au sol 397,
  divers 19 : sémantiquement juste, et non imposé par la méthode.
- **Grammaire** : 2 emplacements d'arme par record, 1er id à +1950 (médiane ; 1664-2029), alias
  du même canon à **+97** (médiane et quasi-totalité), 2e emplacement à **+203** (195-205).
  Après dédoublonnage des alias : 2 armes dans 90 records, 3 dans 55, 4 dans 5.
- **Décodage** : 150 loadouts / 184 records bipèdes, **80 slots**, **26/26 keyframes**.
- **Jointure** : 41/150 loadouts nommés, **8/8 joueurs**, 18/26 keyframes. Le verrou est le pont
  slot→joueur (26/99 slots ont un propriétaire voté), PAS le décodage d'arme (80/99 slots).
- **TÉMOIN CROISÉ POSITIF** (loadout keyframe type-2 vs arme des events de tir type 105, deux
  sources sans contact) : **233/237 = 98,3 %**.
- **TROIS TÉMOINS NÉGATIFS DISJOINTS** : autre slot vivant au même instant **7,7 %** (104/1353) ;
  permutation cyclique des joueurs **7,2 %** (31/433) ; **rotation d'un cran de « qui porte quoi »
  au sein du MÊME keyframe 7,2 %** (17/237). Le troisième est le décisif : mêmes armes, mêmes
  instants, mêmes tirs — seule la jointure record→slot est cassée, et l'accord s'effondre.
- **Discriminance** (sans elle 98 % ne prouverait rien) : 80 combinaisons distinctes sur 150,
  entropie 6,08 bits, 23 armes distinctes, paire la plus fréquente 7/150.
- **Sensibilité au pont** : relâcher les seuils gagne de la couverture ET perd de l'accord
  (2°/20° -> 21 slots, 98,3 % ; 5°/10° -> 24, 93,5 % ; 10°/5° -> 26, 90,3 %).
- **4 désaccords** : tous un tir 3-17 s APRÈS la dernière image-clé de la vie, sans image-clé
  suivante (mort avant). 3 Ravager, 1 M41 SPNKr = armes de puissance ramassées. Cohérent mais
  INVÉRIFIABLE : comptés comme désaccords.

### Contre-épreuve qui NE valide rien, dit tel quel
Catalyst (01e1f945, Slayer standard) : positif 99,0 % MAIS négatifs **78,0 / 81,3 / 76,4 %**.
Raison mesurée : **120 loadouts sur 168 sont identiques** (MA40 AR + Mk51 Sidekick), 16
combinaisons en tout. Le témoin n'y est PAS discriminant ; le 99 % confirme la cohérence du
décodage d'arme, rien de plus. La validation de la jointure repose ENTIÈREMENT sur Cliffhanger.

### Avant / après les correctifs de chaînage — mesure honnête
Arbre `ed1bcc153` extrait par `git archive`, même sonde recompilée dessus : **strictement
identique** (184/54/133/127, désync i57=121, i55=9). Ce n'est pas un correctif sans effet : la
métrique est indicée par NUMÉRO de composant, donc aveugle à un décalage de bits quand le
composant qui échoue est le suivant immédiat. Les sites corrigés SONT traversés sur ce chemin
(i56 : 123 fois ; weapon-state-ammo : 108 fois) mais en aval d'un masque déjà faux. Le gain
10 895 -> 23 485 du lot précédent portait sur le parcours séquentiel des deltas, autre chemin.

### Changements d'arme — chiffre à ne pas maquiller
33 paires de keyframes consécutifs du même joueur, 15 loadouts différents, mais **13 sont des
réapparitions** (nouveau tirage Fiesta) et seulement **2 sont des ramassages à slot constant**.
8/15 corroborés par un tir de l'arme nouvelle dans l'intervalle ; **0/2 pour les vrais
ramassages** — les deux armes nouvelles sont un marteau et un lance-roquettes, et le record 105
n'existe que si un dégât est appliqué. Absence de témoin, pas témoin d'absence.

### Non livré (le §6 du document killweapon promettait davantage)
Grenades (i47) et capacité (i48) NON décodées : aucune table d'ancrage d'ids de capacité, et
rien ne prouve que les 3 ids de grenade du catalogue viennent d'i47. Munitions non décodées.
Instant exact d'un échange d'arme : hors de portée (pas de trace sans dégât).

### Non-régression
Artefact de rejeu `data/cache/replays/halo_infinite/000d5950.json` **MD5 identique**
(`4728ffa2bc55b73de77080722d5fb67e` : 99 trajectoires, 29 221 points, 281 tirs, 382 géométrie,
10 223 emprises). `go build`, `go vet ./internal/analysis/...`, `go test ./internal/analysis/...`
verts ; `cmd/tmp_kfworldpos` inchangé (249/250, 8/8). Seule modification hors sonde jetable :
ajout de `TraverseKeyframeBipedAt` dans `filmdec/probe_export.go` — fonction NOUVELLE, appelée
par la seule sonde, qui recompose des chemins existants sans en modifier aucun.

### Conclusion / prochaine étape
Intégrable : arme primaire + secondaire par joueur et par image-clé (~20 s de résolution), sur
les 41 loadouts nommés ; 21 des 22 armes nommées ont un visuel dans `static/weapons-assets`
(manque Cindershot ; `Carabine.png` = Pulse Carbine à confirmer à l'œil). Prochaine étape la
plus rentable : élargir le pont slot→joueur (26/99), qui plafonne à lui seul la couverture.
Rapport + schéma : scratchpad `loadout.md`, `loadout-schema.html`, export
`loadouts_000d5950.csv` ; sonde `apps/go-api/cmd/tmp_kfload`.

---

## [2026-07-26] Armes portées en PRODUCTION : ReplayDocument.Loadouts + visuels dans le rejeu

**Statut : Complété.**

### Décision technique principale
Le décodage du loadout passe de la sonde jetable au chemin de production, sans rien changer à
la méthode : `filmdec.ScanFilmKeyframeLoadouts` balaye la charge utile du keyframe à la
recherche d'identifiants de FAMILLE (high-32) et attribue chaque occurrence au record qui la
contient (bornes de `WalkKeyframeWorld`, ti=35). La voie « chaîne de composants jusqu'à i43 »
reste abandonnée, pour les raisons déjà mesurées (masque de présence faux à cette position).

Deux choix qui comptent :
1. **Catalogue de production SEUL** (`weaponv3.KnownWeaponHigh32`, 35 familles / 30 noms), pas
   la table texte `nm_tag_to_weapon.txt` (74 familles) de la sonde. Mesure : résultat
   **strictement identique** — 150 loadouts, 80 slots, 24 images-clés, 22 armes, positif 98,3 %,
   négatifs 7,7 % / 7,2 %. Et **meilleur** sur un point : 2 armes dans les 150 records
   (contre 2/3/4 avec le catalogue élargi) — les 39 familles supplémentaires n'ajoutaient que
   des doublons d'alias. Aucune dépendance à un fichier hors production.
2. **Le document publie des IDENTIFIANTS, pas des libellés** : `Loadout.W` = famille high-32 en
   hexadécimal 8 chiffres (`"0x0D20C469"`), alias repliés sur un identifiant par canon. Même
   philosophie que `Shot.Weapon`. Le piège `omitempty` est traité : `T`, `Slot` et `W` n'en ont
   PAS (frame 0, slot 0 et liste vide seraient signifiants) ; seul le conteneur `Loadouts` est
   `omitempty`. **SchemaVersion inchangé (1)** : ajout de champ optionnel.

`Options.Loadouts` porte l'entrée décodée plutôt qu'un 6e argument à `BuildFromPositions`
(seuil de 5 paramètres).

### Résultats observés
- Artefact `000d5950.json` régénéré : **150 loadouts, 80 slots, 24 instants**. Le document
  **privé du seul champ `loadouts` est octet pour octet l'ancien** (MD5
  `4728ffa2bc55b73de77080722d5fb67e`, vérifié par troncature) : zéro régression.
- Témoin croisé rejoué sur le code de production (`cmd/tmp_loadprod`) : positif **233/237 =
  98,3 %** ; négatif « autre slot vivant » 104/1353 = 7,7 % ; négatif « permutation joueur »
  31/433 = 7,2 % ; négatif décisif « rotation d'un cran de l'appariement record→slot »
  **17/237 = 7,2 %**. Entropie des combinaisons 5,83 bits, 71 combinaisons pour 150 loadouts.
- Visuels : sur les **22 armes observées, 21 ont un PNG** dans `static/weapons-assets/`.
  Seule la **Cindershot** n'en a aucun (ni Cremator ni Mutilator ne la représentent) → repli
  **texte encadré**, jamais un visuel approchant. Correspondance nom→fichier EXPLICITE dans
  `cmd/tmp_wicons` ; deux lectures d'image ont tranché les noms francisés ambigus :
  `Carabine.png` = Pulse Carbine (fourche avant caractéristique), `Plasma.png` = Plasma Pistol.
- Page de démonstration : PNG ré-échantillonnés à 34 px de haut (moyenne de boîte en alpha
  prémultiplié), intégrés en `data:` URI. **1 664 636 → 1 821 052 octets** (+156 Ko).
- Couverture d'affichage, dite telle quelle : **93,8 %** des instants « joueur vivant » ont un
  inventaire affichable, mais seulement **13,4 %** ont une arme EN MAIN désignée — la
  désignation exige un tir de la vie en cours (147 tirs rattachés seulement). Le reste affiche
  les deux armes à encre égale et le libellé « portées ». Rien n'est inventé pour combler.

### Découverte notée, NON traitée (hors périmètre)
La table `weaponNames` embarquée dans la page divergeait du catalogue Go : elle nommait
« Mêlée » les familles `0a1992bc` (S7 Sniper) et `2ac9c2ff` (Heatwave), et ne nommait que
65 tirs sur 147. Elle a été **remplacée** par les noms résolus en amont depuis
`weaponv3` (147/147 nommés) — c'était nécessaire pour comparer arme-du-tir et arme-portée.
En revanche, la barre de bouclier des colonnes d'équipe applique `Math.round(sh*100)` à une
valeur déjà en pourcentage (0..100) et rend donc `width:6900%` : **bug PRÉEXISTANT**, non
touché, à corriger dans le lot qui possède ce code.

### Conclusion / prochaine étape
Livré : `ReplayDocument.Loadouts` + rendu « arme en main » avec visuel dans les colonnes
d'équipe (thèmes clair et sombre, visuels inversés en sombre). Le plafond n'est plus le
décodage des armes mais la **désignation de la main** : elle dépend du nombre de tirs
rattachés (147 sur 519 events). Élargir ce rattachement est l'étape la plus rentable.
Fichiers : `internal/analysis/filmdec/keyframe_loadout.go`,
`internal/analysis/replay/loadouts.go`, sondes `cmd/tmp_loadprod` (témoin),
`cmd/tmp_wicons` + `cmd/tmp_famjson` (génération des visuels et de la table de noms).

## [2026-07-26] Arme portée : loadout au keyframe joint au binding joueur — Complété

### Décision technique principale
Le loadout n'est PAS lu par la chaîne de composants (elle désynchronise avant i43) mais par un
BALAYAGE de la charge utile du keyframe à la recherche des 32 bits de FAMILLE d'arme, chaque
occurrence étant attribuée au record qui la CONTIENT (bornes de `WalkKeyframeWorld`).
L'apport par rapport à la sonde de la branche killweapon n'est pas le balayage — ils l'avaient —
mais de NE PAS chercher un champ slot à un offset inconnu : la borne de record donne le slot.

### Résultats observés
- 150 loadouts décodés sur 184 records bipèdes ; 2 armes dans 90 records, 3 dans 55.
- Emplacements stables : 1re arme à ~+1950 bits du début de record, alias du même canon à +97,
  2e emplacement à +203 suivi de son alias.
- **TÉMOIN CROISÉ 233/237 = 98,3 %** (l'arme d'un tir appartient au loadout du MÊME slot).
  Trois témoins négatifs DISJOINTS : autre slot vivant 7,7 % · permutation de joueur 7,2 % ·
  **rotation d'un cran de l'appariement record→slot 7,2 %**.
- Non-circularité VÉRIFIÉE dans le code : le pont tir→slot n'utilise que `AimHeadingDeg()` et
  `PlayerIndex`, JAMAIS `WeaponID`. L'arme n'entre à aucun moment dans la jointure.
- Visuels : 21 des 22 armes observées ont leur PNG (correspondance écrite à la main, arme par
  arme). **Cindershot** n'en a aucun → repli TEXTE encadré, jamais un visuel approchant.

### Ce que l'audit adversarial a rattrapé, et ce que j'en ai fait
Le vérificateur a établi que le **témoin C — celui présenté comme décisif — n'avait AUCUNE trace
archivée**, alors que son chiffre était écrit en dur dans `loadouts.go`, `document.go` et ce
journal. Je l'ai RELANCÉ plutôt que de le croire ou de l'effacer :
```
rotation 0 (réel) : 233/237 = 98,3 %
rotation 1        :  17/237 =  7,2 %   <- le chiffre publié, CONFIRMÉ
rotation 2        :  31/237 = 13,1 %
rotation 3        :   3/237 =  1,3 %
rotation 4        :  27/237 = 11,4 %
```
Transcript versionné : `.ai/re_dump/loadout_temoin_rotation.txt`. Ce témoin est le plus fort du
lot : il ne change NI le décodage des armes, NI les slots, NI les tirs — il ne casse que
l'appariement record→slot. C'est donc bien la jointure qui porte le 98,3 %.

### Réserves écrites
- **Le film Catalyst NE VALIDE RIEN** : positif 99,0 % mais négatifs à 76-81 %, parce que 120
  loadouts sur 168 y sont IDENTIQUES (MA40 + Sidekick). Quand tout le monde porte la même chose,
  99 % ne distingue personne. La validation repose ENTIÈREMENT sur Cliffhanger (Fiesta, 22 armes,
  entropie 5,83 bits).
- **Le verrou n'est pas le loadout mais l'attribution** : 80 slots portent un loadout, seuls 26
  sur 99 ont un propriétaire voté par la règle stricte. 41 loadouts nommés sur 150 décodés.
  Élargir le pont slot→joueur est le prochain gain le plus rentable.
- Grenades et capacité NON livrées : le §6 du document killweapon promettait plus que ceci.
- Défaut de pairage relevé par l'audit : le témoin négatif A itère sur 281 tirs quand le positif
  en garde 237. La phrase « les mêmes tirs » était fausse pour A ; B et C sont correctement appariés.
- Les chiffres Catalyst du §4.2 n'ont pas de transcript archivé — à ne pas citer.

### Conclusion / prochaine étape
Arbitrer le bit `maskSel` : `consumeMask` lit 20 bits d'en-tête là où `offline_biped` en lit 21,
grammaire validée sur 171 000 positions. Un bit AVANT les index de composants — candidat sérieux
pour la désynchronisation résiduelle du walk séquentiel.

## [2026-07-26] ARBITRAGE `maskSel` — le bit n'existe pas ; le désaccord porte sur `IDLowBits`
Statut : **Complété** (verdict tranché sur pièces ; une question DIFFÉRENTE reste ouverte).

**Décision technique.** Deux lecteurs divergeaient d'un bit d'en-tête : `consumeMask`
(traverse.go, walk séquentiel) contre `bipedHeaderBits = 21` (offline_biped.go, documenté
« gate + maskSel + maskCount »). Arbitrage : `FUN_1406d7610` décompilé RENVOIE lui-même sa
longueur (`4`, `6*count+4`, `0x41`) — il n'y a **aucun** bit `maskSel`. La chaîne d'appel
(`FUN_1406cd128` → `FUN_1406d3140` → `FUN_141f86b58` → `FUN_14076cb60`) ne lit rien entre
l'id et le masque ; le garde `R(8)` du mode film n'existe que pour NEW/DEL, jamais DELTA.
Grammaire réelle : `[1 préfixe][idLow][2 tag][1 porte][3 count][6×count]` = `7 + idLow`.
Il n'existe donc pas de lecture « 20 » : le 21 se décompose `1+14+2+1+3`, le bit « en trop »
est le 14e bit du CHAMP ID. `idLow` sort de `FUN_1406d310c(DAT_1451f98d4[14])`, peuplée au
chargement de carte : c'est une valeur de RUNTIME, variable d'un film à l'autre.

**Résultats observés.** (a) Vérité terrain live `ce_capture_delta.csv` (807 855 lignes,
curseur bit par composant) : en-tête de record DELTA = **21 bits sur 93,89 %** de 166 800
paires, facteur « 6 bits par index » confirmé séparément pour count=1..7, amorce de paquet
= 2 bits (1er record à 23+6·count, 48 % de 36 738 segments), branche DENSE = 82 = 68+14 par
un chemin disjoint. (b) Hypothèse `maskSel` mesurée (patch temporaire, reverté) sur
`tmp_l0witness` : clean 23 622 → **20 458**, désync 876 → **4 027** (×4,6), trames propres
53,90 % → 45,96 %. Par archétype : ti=35 23 485→20 351, ti=5 103→92, ti=11 17→5, ti=37
72→71, ti=42 155→152, ti=41 195→160 — **aucun archétype ne s'améliore**. Hypothèse réfutée
deux fois. (c) Le résidu de 876 n'est pas un problème de masque : 792/876 tombent sur
`i00 game-engine-team-mapping-component`, donc sur des slots 512..519 liés à un archétype
non-bipède (aliasing de slot) ; ti=35 seul est à 99,93 %.

**Piège évité.** La capture live NE vient PAS de `000d5950` (i0 = 47 bits = porte+3×15 vs
13/13/14 sur Cliffhanger). `ce_world_dump.txt` n'est donc pas le World de ce film : tout
balayage `idLow` semé dessus est non concluant et a été écarté, pas publié comme preuve.

**Livré.** Correction documentaire uniquement (aucune ligne exécutable modifiée) :
`offline_biped.go` (grammaire sans `maskSel`, décomposition du 21, fenêtrage 13 bits
requalifié en MOTIF de reconnaissance) et `frame_records.go` (`IDLowBits` = valeur runtime
variable, pas un défaut 13). Non-régression : suite `filmdec` ok, `tmp_l0witness` identique
(23 622/876 · ti=35 23 485/17 · 3 000 dead-state / 741 morts / 157 signatures), positions
+ vie + bouclier 171 842/974/27 405 avec MD5 `03ddba826ee7302bc19ba50b08a37a32` identique
avant et après le patch. Outils : `cmd/tmp_hdrtruth`, `tmp_maskarb`, `tmp_l0arb`,
`tmp_tagprobe`, `tmp_nrsig`.

**Prochaine étape.** La mesure a exhumé une question DIFFÉRENTE et non résolue : sur
`000d5950`, walk (idLow=11, préambule 18) et scanner offline (préambule 21) sont à **3 bits**
d'écart et leurs curseurs i0 ne coïncident **jamais** (0/11 834 ; 2 132 à exactement −3).
Note : `512..519 = (4096..4159)>>3`, soit exactement le facteur qu'introduirait une lecture
11 bits d'un champ de 14. Le discriminant local par le tag (`tmp_tagprobe`) n'est pas
concluant (84–90 % pour toutes les largeurs) — dit tel quel. Le déblocage grenades/capacités
(i22, i47, i48, i56, i57) passe par la chaîne de composants du walk : il faut d'abord
**résoudre `idLow` pour `000d5950`** (capture de `DAT_1451f98d0/d4[14]` en vivant, ou capture
`filmdec_delta_capture` sur un film dont on possède les chunks). Ne PAS retoucher
`consumeMask` : la question y est close.

## [2026-07-26] Grenades et capacités — récolte tentée, TÉMOIN ÉCHOUÉ, RIEN INTÉGRÉ

**Statut : Complété (résultat négatif, documenté).**

**Décision technique.** Mesurer avant d'intégrer, et mesurer avec un étalon. L'atteignabilité
de `i22 unit-grenade-counts`, `i47 biped-desired-grenade-set`, `i48 biped-desired-ability-set`,
`i56 biped-spartan-ability-energy` est de **100 %** sur le walk séquentiel (9 425 / 11 483 /
11 480 / 23 169 occurrences atteintes, `cmd/tmp_grenreach`). `i57 biped-spartan-ability` n'est
présent que **9 fois sur 23 502 records** et n'est pas porté : donnée absente, pas décodage
manquant. L'atteignabilité n'était donc pas le blocage.

**Résultats observés.** Le CONTENU de ces composants est du bruit, établi par trois voies
indépendantes.
1. *Critère absolu* (aucune référence requise) : **91,19 %** des valeurs `R(8)` de i22 sont
   `> 4`, donc impossibles pour un compte de grenades (plafond Halo Infinite : 2 par type,
   4 avec Grenadier) ; 18,22 % des records annoncent plus de 4 types.
2. *Témoin croisé par les lancers* (source indépendante : marqueur `0x4c0c00`, 70 events,
   `player_index` 0-7 fiable) : **0 slot décroît sur 39 lancers / 70**, 1 seul sur les 31
   autres — et toujours le **même** slot (519) quel que soit le lanceur. Aucune bijection
   `player_index → slot`. **Témoin négatif inexploitable faute de témoin positif.**
3. *Continuité temporelle avec CONTRÔLE POSITIF et ALÉA* : i4 vie et i5 bouclier (validés)
   donnent un ratio signal/aléa+37bits de 2,20 et 2,28 ; i22/i47/i48/i56 donnent 1,04 / 0,95 /
   0,97 / 0,97 — **indiscernables de l'aléa**. L'étalon +37 bits était indispensable : le flux
   local est riche en zéros et gonfle la stabilité de n'importe quelle lecture (28,44 % pour
   i22, 27,39 % pour son aléa).

**Diagnostic.** *False-clean* : 99,93 % de records « clean » avec un contenu faux. L'échelle
de continuité composant par composant (`cmd/tmp_grenladder`) montre que **la chaîne est
alignée jusqu'à i6 (ratios 1,06 → 2,66) et dérive à partir de i7** `object-damage-sections`
(ratio 0,34, largeur moyenne 280 bits) ; au-delà, tout est à ~1,0. Tout ce qui est en
production (positions, vie, bouclier) vit AVANT la rupture ; tout ce que la mission visait vit
APRÈS. Le test causal stratifié (`cmd/tmp_grencause`, 56 strates sur 7 suspects amont) ne
trouve **aucune strate qui récupère du signal** (ratios 0,92 à 1,11) : la dérive est
systémique, pas imputable à un composant fautif isolé. Corroboration indépendante déjà dans le
dépôt : l'en-tête de `keyframe_loadout.go` documente le REFUS de la même chaîne sur le chemin
keyframe (« le masque lu n'est pas le vrai masque »), ce qui explique le décodage du loadout
par balayage — et pourquoi il ne donne « ni les grenades, ni la capacité d'armure ».

**Réserve i48, dite franchement.** Même correctement décodé, l'index `R(6)` resterait un
ENTIER NU : c'est un index dans la palette d'équipement DU MATCH, aucune énumération des
capacités spartan n'existe dans l'exécutable ni dans le catalogue Go (`grep` sur
`internal/analysis/` : une seule occurrence de « grapple », et c'est un gamertag de bot).
« Capacité #37 » ne répond pas à la demande « grappin / propulsion / mur de protection /
surbouclier / camouflage actif ».

**Livré : RIEN.** Aucune modification du `ReplayDocument` (SchemaVersion inchangé, aucun champ
`omitempty` ajouté), aucune modification de `replay_demo.html`. Une grenade au mauvais compte
est pire qu'aucune grenade. Non-régression : aucune ligne exécutable modifiée dans `internal/`
(diff `filmdec` = commentaires uniquement, vérifié par filtrage) ; `tmp_l0witness 3 26` rejoué
→ 23 622 / 876 et ti=35 23 485 / 17, identique à `witness_after.txt`.

**Outils** (`cmd/tmp_*`, lecture seule, hors chemin de production) : `tmp_grenreach`,
`tmp_grenval`, `tmp_grencont`, `tmp_grenladder`, `tmp_grencause`. Sorties archivées :
`scratchpad/gren_reach.txt`, `gren_val_witness.txt`, `gren_continuity.txt`, `gren_ladder.txt`,
`gren_causal.txt`. Rapport : `scratchpad/grenades_capacites.md` (avec schéma).

**Prochaine étape.** Ne PAS retoucher les désérialiseurs i22/i47/i48/i56 : ils sont
EXE-vérifiés et consomment le bon nombre de bits, ils sont appelés à la mauvaise position.
Réaligner la chaîne en aval de i6, ce qui suppose d'abord de **résoudre `idLow` pour
`000d5950`** — exactement la question laissée ouverte par l'arbitrage `maskSel`. Critère de
réussite déjà écrit et objectif : ratio signal/aléa de `tmp_grenladder` > 1,5 sur
i22/i47/i48/i56, et témoin négatif à « 1 slot décroît » sur les 70 lancers avec bijection
stable `player_index → slot`.

## [2026-07-26] Bit maskSel : HYPOTHÈSE RÉFUTÉE — et ce que la réfutation a révélé

### Décision technique principale
J'avais placé cet arbitrage en TÊTE de file, devant la structure en maillages, sur le pari qu'un
bit manquant expliquait la désynchronisation résiduelle du walk séquentiel. **Le pari est perdu :
le bit n'existe pas.** Deux méthodes indépendantes le prouvent.

**Par le désassemblage** : `FUN_1406d7610` RETOURNE elle-même sa longueur en bits — `4` /
`6*count+4` / `65` selon la branche. C'est de l'arithmétique écrite dans le code, pas une
inférence. Aucune place pour un cinquième bit. `consumeMask` (traverse.go:1071) est un portage
FIDÈLE. La chaîne d'appel a été remontée pour exclure « le bit est en amont » : le garde R(8) du
mode film n'existe que pour NEW et DEL, jamais pour DELTA.

**Par la mesure** : vérité terrain live (`ce_capture_delta.csv`, 807 855 lignes de curseurs de
bits réels). En-tête entre records = **21 bits sur 93,89 %** des 177 660 paires. Confirmé
SÉPARÉMENT pour chaque valeur de `count` (sept confirmations). Et la branche DENSE, chemin
totalement disjoint du compteur et des index, donne 82 = 68 + 14 — même réponse.

### LE VRAI SENS DU 21, et une découverte qui compte
`21 = 1 préfixe + idLow(14) + 2 tag + 1 gate + 3 count` — PAS `1+13+2+1+1+3`.
Le bit « en trop » est le **14e bit du CHAMP ID**. Et `idLow` sort de `FUN_1406d310c` sur une table
peuplée AU CHARGEMENT DE CARTE : **il diffère d'un film à l'autre** (11 sur 000d5950, 14 sur le
film de la capture live). Le fenêtrage 13 bits d'`offline_biped.go` est un **motif de
reconnaissance**, pas le champ id. C'est une propriété d'universalité du décodeur qu'on ignorait.

### Effet mesuré de l'hypothèse fausse (appliquée puis revertée)
  slots joueur clean 23 622 -> 20 458 (-13,4 %) ; desync 876 -> 4 027 (x4,6)
  ti=35 clean 23 485 -> 20 351 (-13,3 %) ; ti=11 objectif -70,6 % de clean
  AUCUN archétype ne s'améliore. Le conditionner au bipède aurait été tout aussi faux.

### CE QUE LA RÉFUTATION A RÉVÉLÉ (le vrai gain de ce lot)
1. **Le résidu de 876 n'est pas un problème de masque** : 792 des 876 tombent sur
   `i00 game-engine-team-mapping-component`, donc sur des records dont le slot 512..519 est lié à
   un archétype qui N'EST PAS le bipède — de l'**aliasing de slot**. L'archétype ti=35 lui-même est
   déjà à **23 485 clean / 17 desync = 99,93 %**. *(RÉSERVE : le vérificateur juge cette explication
   de remplacement insuffisamment étayée — à confirmer avant de s'en servir.)*
2. **Grenades et capacités : atteignables à 100 %, mais le CONTENU est du bruit.** Distinction
   capitale : *décodé au sens où les bits sont lus* n'est pas *décodé au sens où ils sont justes*.
   Réfuté par un critère ABSOLU qui ne demande aucune statistique de référence — Halo plafonne à
   2 grenades par type (4 avec Grenadier), 4 types au plus :
     `i22` count R(3) > 4 : 18,22 % · **valeurs R(8) > 4 : 91,19 %** (plage 0..255 pleine).
   Extrait brut : `c=[255 148 52 4 177 104 193]`. Ce ne sont pas des grenades.
   Pour `i56`, les 72,77 % de charges « pleines » sont un **artefact arithmétique** : le déser
   écrit 0x7F SANS LIRE quand le bit de masque est à 0. Sur les lectures réelles, 2,4 % donnent 127
   contre 0,78 % attendus d'un tirage uniforme, et la distribution est PLATE sur 0..127 — aucune
   rampe de recharge. Rien n'est rendu.

### Conclusion / prochaine étape
La correction retenue est **documentaire uniquement** (aucune ligne exécutable modifiée) : la
grammaire est réécrite sans `maskSel`, et `IDLowBits` est documenté comme valeur de RUNTIME
variable d'un film à l'autre. La structure en maillages reprend sa place en tête de file — elle
porte les critères d'acceptation de l'utilisateur.

---

## [2026-07-26] Lancers de grenade décodés — et deux affirmations de ce journal corrigées

**Statut** : Complété (décodeur + intégration document) / En cours (rendu POC)

### Décision technique principale

Les grenades ne se lisent pas dans la chaîne de composants (`i22`/`i47`), elles se lisent par
**balayage d'un marqueur** dans les paquets delta — exactement le remède déjà employé pour
`keyframe_loadout.go` quand le masque de présence s'est révélé faux.

    [marqueur 24 b = 0x4C0C00] [identifiant 32 b] [47 b] [index joueur 5 b]

Nouveau `internal/analysis/filmdec/grenade_events.go` (`ScanFilmGrenadeThrows`) +
`internal/analysis/replay/grenades.go`. La sélectivité ne vient PAS du marqueur seul mais de la
liste blanche des 4 identifiants de grenade. `uniqueSlotFor` a été factorisé pour que tirs et
grenades partagent le MÊME pont slot->joueur plutôt que d'en diverger.

### Résultats observés (film 000d5950, Super Fiesta, Cliffhanger)

  70 lancers validés — le compte exact annoncé par .ai/FIRE_MELEE_GRENADE_EVENTS.md §8
  types : Spike 22, Fragmentation 17, Shock 17, Plasma 14
  TÉMOIN : 70/70 index joueur dans 0..7, alors que le champ fait 5 bits (0..31)
           attendu par pur hasard : 25 % -> 0,25^70 ~ 5e-43
  couverture : 8 joueurs distincts sur 8, répartition inégale (3 à 20 lancers par joueur)
               -> ce n'est donc pas un champ de bourrage systématiquement nul
  publiés dans le document : 27/70 — plafonné par le pont slot->joueur (26 slots sur 99)

**Ce qui NE serait PAS un témoin** : « 14/14 des slots à grenade sont aussi des slots tireurs ».
Le rattachement passe par le pont voté sur les tirs, donc l'intersection est vraie **par
construction**. Noté ici pour qu'on ne le cite jamais comme preuve.

### CORRECTION 1 — « le contenu est du bruit » est une formulation fautive

L'entrée précédente écrit « le CONTENU est du bruit » à propos d'`i22`/`i47`/`i48`/`i56`.
**Le film ne contient pas de bruit** (correction utilisateur) : il contient des champs que nous
ne savons pas situer. Dire « bruit » attribue le défaut au flux et **ferme la question** ; dire
« notre curseur dérive en amont » l'attribue au décodeur et la garde ouverte. Ce n'est pas de la
rhétorique : c'est la seconde formulation qui a fait chercher la voie du marqueur, et cette voie
rend 70 lancers avec 8/8 joueurs là où la première déclarait le sujet sans espoir.

### CORRECTION 2 — le témoin « catégorique » du Slayer n'existe pas

`.ai/GRAMMAIRE_RECORD_FILM.md` affirmait qu'en Slayer `i48` devait être « vide de bout en bout »
et `i47` quasi constant, et présentait cela comme le témoin le plus fort disponible.
**FAUX** (correction utilisateur) : les créateurs de carte posent au sol des grenades de tous
types ET des capacités Spartan, ramassables en cours de partie. Ces règles ne contraignent que
l'**instant du spawn**. Un décodeur qui ne sait pas situer les spawns ne peut pas s'en servir.
Seule borne restée utilisable sans rien savoir du temps : la quantité (<= 2 types, <= 2 unités).
Leçon : vérifier les mécaniques qui assouplissent une règle de mode AVANT de la promouvoir
témoin décisif.

### Lot précédent, non consigné jusqu'ici

- **Découpage des zones de callout sur le sol praticable** : 5 382 -> 4 250 m2, **-21 %** à
  rétention égale, contre **9,1 % +- 0,3** pour des emprises tirées au hasard (~40 sigma).
  Piège désamorcé par l'agent lui-même : *la rétention des positions, prise seule, NE DISCRIMINE
  RIEN* (98,2 % au hasard contre 98,17 % en réel). C'est la réduction de surface qui porte tout.
- **Résultat négatif à garder** : Fer à cheval passe de 65,6 à 65,3 m2, **-0,5 %**. Le découpage
  retire le surplomb sur le vide, il ne corrige PAS un marqueur trop généreux posé sur du sol
  plein. **Le critère d'acceptation de l'utilisateur (anneau en donut, zone sud reliée par deux
  ponts) n'est donc PAS satisfait par ce lot** — il reste porté par la chaîne rtgo.
- **28 dispositifs** trouvés par récurrence spatiale (1 canon humain + 27 points d'apparition).
- **Assets** : 4 grenades converties depuis les captures in-game. Diagnostic mesuré des captures :
  ni cisaillement ni rotation (pics d'arêtes H et V à +1 deg tous les deux), mais un **dédoublement
  chromatique** (R décalé de (-2,-1), B de (+2,+1) — antisymétrie parfaite sur les 4 icônes).
  Le réalignement des canaux a été **mesuré puis écarté** : +2,5 % et +5,2 % de netteté sur deux
  icônes, mais -1,7 % et -0,5 % sur les deux autres, et le masque produit étant monochrome les
  franges disparaissent de toute façon. Ce qui a réellement corrigé le rendu : le **découpage sur
  la composante connexe**, qui retire le fragment de l'icône voisine (4, 3, 0 et 2 composantes
  parasites rejetées selon l'icône) et la traînée horizontale du bas.

### Conclusion / prochaine étape

Rendu des lancers dans le POC, puis reprise du plafond slot->joueur (26/99) qui borne à la fois
les grenades publiées (27/70) et le nommage des loadouts (41/150). La structure en maillages
reste en tête de file : elle seule porte les critères d'acceptation de l'utilisateur.

---

## [2026-07-26] Les grenades des keyframes sont des PROJECTILES EN VOL, pas des inventaires

**Statut** : Complété (mesure) / ouvre deux chantiers

### Point de départ : une intuition utilisateur, et une de mes affirmations fausse

L'utilisateur : « t'as rien trouvé en te basant dans la section du joueur, du loadout d'arme, un
truc approchant où se trouvent les id des grenades ? Sachant que c'est byte shifté ou avec un truc
big endian ». Il rappelait aussi, à juste titre, que la recette d'`i22` EST décodée.

Deux corrections à mon compte :
1. **`i22` est bien décodé et notre portage est bit-exact.** Désassemblage refait ce jour :
   `FUN_140f0de1c` fait `count = FUN_1424d0f48(reader)` — un `R(3)` BRUT, sans `+1` — puis
   `count × R(8)`. Le problème n'a jamais été la recette, c'est **l'adresse** : on l'exécute au
   mauvais endroit du flux. Dire « i22 n'est pas décodé » était faux.
2. **L'en-tête de `keyframe_loadout.go` affirme « CE QUE CE DÉCODEUR NE DONNE PAS : ni les
   grenades ».** C'est faux : il ne les donne pas parce qu'elles ne sont pas dans son catalogue.

### Mesure 1 — l'encodage des grenades est IDENTIQUE à celui des armes

Balayage des 4 identifiants de grenade sur tout le film, dans trois ordres d'octets :

      variante        DELTA (type 0)   KEYFRAME (type 2)
      brut 32 b MSB        70                12
      octets inverses       0                 0
      mots 16 inverses      0                 0
      64 b + suffixe 0x42C9679F : 70          12    <-- STRICTEMENT EGAL au compte 32 b

Attendu par hasard pour un motif de 32 bits : **0,012** occurrence sur les delta, **0,007** sur les
keyframes. Donc chaque identifiant de grenade est TOUJOURS suivi de `0x42C9679F`, le suffixe qui
marque une « vraie arme » dans le chemin loadout. Pas de permutation d'octets : la lecture brute
MSB-first est la bonne (l'hypothèse big-endian de l'utilisateur est écartée par la mesure, mais
son intuition sur l'ENDROIT était juste).

### Mesure 2 — ce que sont réellement ces 12 occurrences

12 pour 26 keyframes et 8 joueurs, c'est **beaucoup trop peu** pour un inventaire (on attendrait
200 à 400). Confrontation aux 70 lancers déjà décodés, sur l'horloge du film :

      appariement (MEME TYPE et moins de 4 s après un lancer) : 10/12
      témoin (les mêmes lancers décalés de 7 s)               :  2/12

Écarts observés : +0,03 · +0,09 · +0,34 · +0,36 · +0,36 · +1,66 · +1,83 · +2,36 · +3,10 s.

**Ce sont les grenades EN VOL**, saisies comme entités du monde à l'instant du keyframe. Une
grenade vole 1 à 3 s ; sur 70 lancers en ~500 s et un keyframe toutes les 20 s, l'espérance est de
l'ordre de 7 — on en observe 12, le bon ordre de grandeur.

### Ce que ça tranche

1. **L'inventaire de grenades n'est PAS dans le keyframe** par la voie des familles d'arme. La
   piste est fermée, proprement.
2. **Un projectile lancé EST une entité répliquée** — l'hypothèse de l'utilisateur sur les
   trajectoires est confirmée par une voie qu'il n'avait pas envisagée. Les archétypes porteurs
   sont `ti=41` et `ti=20` (largeurs de record crédibles : 1 261 à 7 643 bits).
3. **Réserve à ne pas masquer** : 6 des 12 occurrences sont attribuées à `ti=35` par le
   délimiteur, mais dans des « records » de **76 000 à 83 000 bits** — alors qu'un record de
   bipède en fait ~2 800. `WalkKeyframeWorld` a perdu le fil dans cette région : ces six-là sont
   **orphelines**, pas bipèdes. Ne pas les compter comme de l'inventaire joueur.

### Conclusion / prochaine étape

L'inventaire reste à trouver ailleurs (le corps des records, où il reste au moins une faute après
la correction de l'amorce de paquet). En revanche les archétypes `ti=41`/`ti=20` sont des candidats
sérieux au statut de projectile, et le chantier trajectoires en cours doit être orienté dessus.

---

## [2026-07-26] Trois chantiers rendus : projectiles résolus, inventaire partiel, donut réfuté

**Statut** : Complété (projectiles) / Partiel (inventaire) / Non atteint (géométrie)

### PROJECTILES — résolu, livré (commit 076773914)

L'archétype **ti=41** est le projectile, et le registre du film le NOMME : ses composants
s'appellent `projectile-at-rest-state`, `projectile-tether-state`, `projectile-command_tick`,
`projectile-deceleration-disabled-state`. Nos notes le classaient « divers, mal caractérisé ».

    580 trajectoires · 13 544 positions · 23,4 points par vol · durées 0,27 à 2,45 s
    TÉMOIN appariement lancer -> naissance (+-200 ms) : 65/70 = 92,9 %
    CONTRÔLE lancers décalés en bloc : +3s 13 · +7s 12 · +13s 11 · -5s 11

Trois correctifs, chacun nécessaire :
1. `object-position-component` lisait `R(2)` d'index puis 13/13/13 ; le vrai découpage est
   `R(1)` puis 13/13/**14**. **Le total est identique (45 bits)** : aucun test de longueur ni de
   désynchronisation ne pouvait le voir. Résout la contradiction « i0 45 vs 47 » — ce ne sont
   pas deux mesures du même champ, ce sont deux archétypes (porte de 3 bits contre 5).
2. Aucun filtre sur le tag : c'est la GÉNÉRATION du handle, ses 4 valeurs sont légitimes, et
   16 des 70 trajectoires sont intégralement en tag=0.
3. Découpage en vies sur un trou de 250 ms. Sans lui : des « trajectoires » de 300 à 460 s.
   Effet mesuré : 71,4 % -> 92,9 % d'appariement.

**Pas d'impact** : il n'existe aucun événement de détonation dans le film. On lit la DERNIÈRE
POSITION RÉPLIQUÉE. Écrire « dernière position connue », jamais « impact ».

### AMORCE DE PAQUET — corrigée (commit 0ef5cc023)

2 bits avant le premier record de chaque paquet, jamais consommés. Masques sains 14,65 % ->
84,81 % (vérité 99,86 %, hasard mesuré 10,67 %). Notre décodeur faisait **moins bien que le
hasard**. Nécessaire mais PAS suffisant : i22 lit encore 92,46 % de comptes impossibles.

### INVENTAIRE — partiel, et la cause résiduelle est identifiée

`i22` = `R(3)` compteur (doit valoir 4) + 4 × `R(8)`, `i47` = 9 bits. Les deux désérialiseurs
sont bit-exact (re-vérifiés au désassemblage : `FUN_140f0de1c`, compteur `R(3)` BRUT sans +1).
**Le problème n'a jamais été la recette, c'est l'adresse.** Cause dominante restante : `i0`
biped, 47 bits réels contre 23 lus.

**PIÈGE VÉRIFIÉ ET CONSIGNÉ** (commit 4a1191717) : mettre 13/13/14 dans
`TraversalPrecision.AxisW` ne corrige rien et DÉGRADE (i22 de 90,02 % à 92,83 % d'impossibles).
`AxisW` est la largeur des DELTAS ; les 13/13/14 sont celles des ABSOLUS, portées par
`absAxisW`. Le correctif doit distinguer les deux le long de chaque branche de `FUN_1406cfe44`.

Identifiants de grenade : ce sont des tags `proj` **décalés d'un bit** (`0x580B8831 << 1 =
0xB0171062`, 4/4). Le champ commence à +23, mais le bit à +23 vaut 0 dans les 1 416 marqueurs :
les deux lectures sont équivalentes ici. **L'index joueur, lui, NE se décale PAS** — mesuré
0/70 dans 0..7 à +102 contre 70/70 à +103. Propager le décalage aurait cassé un décodeur juste.

Capacités Spartan : 12 tags `eqip` sur 108 apparaissent dans le film (contrôle : 5 000 ids au
hasard -> 36 occurrences totales, rapport ~11 000x). Les 8 identifiants d'équipement sont non
nuls dans **3/3 films Super Fiesta et 0/24 ailleurs**. Seul le **Mur déployable** est nommé avec
certitude ; les 3 autres sont individualisés mais **pas nommables hors ligne** (0 occurrence des
noms dans les .module). Il faudra une capture contrôlée.

### GÉOMÉTRIE — le donut n'est PAS atteint, et le résultat qui l'annonçait est réfuté

Un agent a annoncé l'anneau (34,4 % de vide, trou centré à 0,48 m du centroïde). Trois attaques
indépendantes ont cassé la règle « sol présent + 2 m de dégagement » sur laquelle il reposait :

- elle ne reconnaît que **36,6 %** des positions de joueurs DEBOUT, contre **38,8 %** pour un
  contrôle aux coordonnées permutées — **le hasard fait mieux que la donnée** ;
- un simple couloir (plein à 100 % sous la règle du sol) se creuse à **49,1 %** sous cette
  règle ; le fer à cheval est **14e sur 16 zones** ;
- sur 337 placements du même polygone ailleurs sur la carte, **96,4 %** ont autant ou plus de
  vide.

**Ce qui est vrai et indépendant de toute reconstruction** : le disque de rayon 1 m centré sur
(19,12 ; 11,27) est déserté par les joueurs d'un **facteur 64**. L'obstacle existe ; c'est notre
reconstruction qui ne le voit pas.

Acquis réel : les emprises **orientées** (-47,4 % de surface, 10 357 calculées, visibles dans le
POC par un bouton). Mais elles ne creusent RIEN : sur le fer à cheval, 0,00 m² avant comme
après — les neuf instances qui en bouchent le centre sont à rotation nulle et échelle unité,
donc leur boîte orientée EST leur boîte alignée. L'anneau vit dans les triangles.

Deux découvertes de fond : le champ `scale` dit « vestigial » est un VRAI facteur d'échelle
(98,9 % de cohérence avec, 33,8 % sans) ; et la collision n'est pas dans le build serveur mais
dans `any/`, dont **195 modèles sur 552 vivent dans des modules GLOBAUX partagés** — toute
chaîne « une carte = un module » est structurellement incomplète.

### Conclusion / prochaine étape

Géométrie en priorité 1 : chaîner maillage -> vertex buffer (champ @0x88 du record 0x90),
déquantifier avec les bornes @+0x44 remises à l'échelle. Le témoin à reproduire est le facteur
64 de désertion, qui ne dépend d'aucune de nos reconstructions.

## [2026-07-26] Capture CE du dispatch : l'inventaire est dans les records DENSES — Complété

### Décision technique

Crocheter le dispatch des composants (`0x14076CD11`, juste avant `call [rax+28]`) et
journaliser, pour CHAQUE composant désérialisé : entité, archétype, index du composant,
curseur de bits AVANT lecture, et 16 octets bruts du tampon film. Le crochet a été posé
directement par MCP Cheat Engine sur le process vivant, pas par script manuel.

Vérification préalable décisive : `0x7FF7A655CD11 - 0x7FF7A5DF0000 = 0x76CD11`, soit
`0x14076CD11` en base Ghidra — le build correspond exactement à l'import, donc toute la
table des désérialiseurs i0..i63 est valide sur ce process.

PIÈGE ÉVITÉ, à retenir : **CE lit les tailles d'`alloc()` en DÉCIMAL et les immédiats
d'instruction en HEXADÉCIMAL.** Un `alloc(buf,5000000)` avec une borne `cmp rax,200000`
donne un tampon de 5 Mo pour une borne de 2 097 152 records = 80 Mo. L'allocation
s'arrêtait à `0x7FF7A5DEBF38`, juste sous la base du module `0x7FF7A5DF0000` : au-delà de
~125 000 records la capture aurait écrasé l'image du jeu. Détecté avant lancement par
l'écart `ffcCave -> ffcCnt` valant `0x3E8` = 1000 décimal.

### Résultats observés (200 000 composants, 36 651 records de bipède, Cliffhanger 24/07)

LARGEURS MESURÉES — plus aucune n'est portée depuis Ghidra, elles sont mesurées par
différence de curseurs :

    i0  47 bits (100 %)   i1  31 bits (99 %)   i21 25 bits (100 %)   i25 10 bits (100 %)
    i5  29 bits (100 %)   i22 35 bits (100 %)  i47  9 bits (100 %)   i56 10 bits (100 %)
    i48 10 bits (93 %) / 4 bits   i42 7 bits (68 %) / 5 bits
    i43-i44 15 bits (44 %) OU ~195-204 bits (descripteur d'arme complet)

`i0` = 47 bits sans variance : la contradiction historique « i0 45 ou 47 » est tranchée.
`i22` = 35 bits = `R(3) + 4xR(8)` et `i47` = 9 bits = `R(6)+R(3)` : **nos grammaires
d'inventaire étaient justes depuis le début.** Les heures passées à soupçonner les largeurs
(permutation i25/i26/i27, i0 à 45 vs 47) portaient sur des champs corrects — ce qui explique
pourquoi les deux corrections mesurées avaient donné « juste, mais sans effet ».

TAILLES DE MASQUE — mode à 4 composants (48,82 %), 99,877 % à 7 ou moins (branche éparse,
compteur de 3 bits), et **0,123 % au-delà** (12 à 29 composants = branche dense `R(64)`).

LOCUS DE L'INVENTAIRE — la mesure centrale, avec ses contrôles négatifs :

    i43 arme tenue        45 records, 43 denses  95,56 %   x777
    i47 grenades equipees 45 records, 42 denses  93,33 %   x759
    i22 comptes grenades  51 records, 43 denses  84,31 %   x686
    i42 arme desiree     106 records, 43 denses  40,57 %   x330
    i48 capacite          47 records, 15 denses  31,91 %   x259
    i0  position (TEMOIN)  35 491 records         0,07 %   x0,6
    i25 tick    (TEMOIN)   36 475 records         0,11 %   x0,9

Les deux témoins sont ce qui rend la mesure concluante : les composants COMMUNS restent à la
référence (0,123 %) ou en dessous, pendant que les composants d'inventaire sont enrichis de
259 à 777 fois. Cohérence de détail : `i56` (énergie de capacité) n'est dense qu'à 8,57 %,
parce qu'une énergie se vide en continu et se retransmet en épars, alors que la COMPOSITION
du loadout ne change qu'au spawn.

### Conclusion / prochaine étape

L'inventaire n'est pas dans le flux courant : il est dans ~45 records denses sur 36 651,
émis au spawn. Notre décodeur prend la branche dense de `consumeMask` (traverse.go:1171)
dans ~12 % des cas au lieu de 0,12 % — cent fois trop — et chaque lecture dense parasite
pose une trentaine de composants fantômes qui désynchronisent la suite du paquet. C'est LA
faute unique, et elle est dans le bit de porte, pas dans les grammaires.

Prochaine étape : confronter notre décodeur à cette table sur le MÊME film (la signature de
16 octets bruts identifie le film parmi les 948 du cache, sans base de données), et corriger
la lecture de la porte.

PERTE À NOTER : le process a été fermé avant le vidage final ; ~760 000 records perdus. Le
vidage à mi-parcours (200 000) avait sauvé tous les résultats ci-dessus. Règle pour la
suite : **vider le tampon sur disque PENDANT la lecture, à intervalles réguliers.**

### Confirmation sur FILM ENTIER (975 250 composants, 159 772 records de bipède)

La seconde capture couvre le film complet et confirme le résultat sur 4x plus de données.
Référence des records denses : 0,145 % (231/159 772).

    i43 arme tenue        208 records, 194 denses  93,27 %  x643
    i47 grenades equipees 215 records, 182 denses  84,65 %  x584
    i22 comptes grenades  256 records, 191 denses  74,61 %  x515
    i42 arme desiree      447 records, 198 denses  44,30 %  x306
    i48 capacite          218 records,  95 denses  43,58 %  x301
    i0  position (TEMOIN) 154 158 records           0,08 %  x0,55
    i25 tick    (TEMOIN)  158 804 records           0,13 %  x0,90

CAPACITÉ SPARTAN — l'état actif est TROUVÉ. `i57` (biped-spartan-ability) apparaît 990 fois
avec une largeur de **2 bits dans 94 % des cas** : c'est un état marche/arrêt qui bascule
990 fois dans la partie, et il n'est PAS concentré dans les records de spawn (0,62 % de
présence contre 0,13 % pour l'inventaire). Couplé à `i48` qui dit QUELLE capacité est
équipée (dans les records denses), on a le couple complet demandé : i48 = laquelle,
i57 = quand elle est active. `i59` (état non prédit) l'accompagne à 1 048 occurrences.

COMPOSANTS JAMAIS ÉMIS sur ce film : `i60` et `i61` (simulation-state / playback), `i62`
(glissade), `i63` (action). À ne pas confondre avec « non décodés » — le moteur ne les
dispatche pas du tout. Toute énergie dépensée à porter leur grammaire serait perdue.

APPARIEMENT DU FILM — ÉCHEC, et la cause est identifiée. Les signatures de 16 octets ne
départagent aucun film (14/40 sur le meilleur, 13/40 sur le second, étalé sur des dizaines).
Deux causes : (a) le filtre d'entropie à 8 octets distincts est trop laxiste et laisse passer
des fenêtres de bourrage (`fffffffe07fffffffc17`, `0000000000000700...`) qu'on retrouve
partout ; (b) surtout, si le film était en cache un film décrocherait 40/40 — aucun ne le
fait, donc **le film du 24 juillet n'est pas téléchargé**. Le match est en base, les chunks
non. L'ancrage record-contre-record attend ce téléchargement.

### PIERRE DE ROSETTE POSÉE — le film est identifié et la capture y est ancrée

Le film rejoué est `9e8fb31b-ea96-4848-a3b0-03117171d01e` (Cliffhanger, Slayer:Arena Super
Fiesta, 2026-07-24 19:21:32). **59 signatures sur 60 le désignent**, alors que le MÊME filtre
durci appliqué aux 948 films du cache en trouvait **zéro**. La discrimination est totale.

CHAÎNE DE RÉCUPÉRATION, pour la refaire :

1. `cmd/tmp_findmatch` — l'identifiant COMPLET du match par carte + date. Les manifestes sont
   nommés par le PRÉFIXE (8 hex) et rien dans le cache ne porte l'identifiant entier. L'outil
   travaille sur une COPIE de la base : le serveur peut la tenir en écriture (ADR 0013/0016)
   et on ne prend aucun risque de verrou.
2. `cmd/tmp_filmmanifest` — un manifeste FRAIS via `discovery-infiniteugc.../spectate`. Les URL
   de blob sont pré-signées et expirent : sur les 62 chunks manquants du cache, 62 rendaient
   404. Aucun chemin Go ne savait rafraîchir un manifeste avant cet outil.
3. `cmd/fetch_film_chunks` — 22 chunks récupérés (18 Mo).

AUTH — ce qui a été appris. Les entrées legacy `.env.local` sont **vides** (0 caractère) :
la migration ADR 0023 les a vidées. Le chemin canonique
`auth.RefreshHaloTokensViaStoreFirst` sur le store est le seul vivant. Sur les 9 jetons du
store, **4 rendent AADSTS70000** (« token issued for a different client id » — vieille app,
exactement le cas que l'ADR interdit de « réparer » par re-capture) et **5 sont sains**. Les
jetons sains portent les xuid 2535405528935279, 2535409018618248, 2535413181053876,
2535430985184703, 2535460062932944. Ceux des joueurs principaux (JGtm 2533274823110022,
Madina 2533274858283686, Chocoboflor 2535469190789936) sont morts et devront être renouvelés
par SSO — PAS par re-capture de token.

PIÈGE consigné : `loadEnvLocal` (cmd/refresh_golden_fixture) rend la main après le PREMIER
fichier lisible de sa liste ; un `.env.local` vide en amont masquerait celui de la racine.
Ce n'était pas la cause ici (les valeurs sont vides à la source) mais le piège est réel.

PROCHAINE ÉTAPE, désormais débloquée : faire tourner notre décodeur sur `9e8fb31b` et
comparer, record par record à l'offset exact, la liste des composants qu'il croit présents
avec celle que le moteur a réellement désérialisée.

### [2026-07-27] i0 corrigé (23 -> 47 bits) — MAIS ce n'était PAS la cause. Le vrai défaut est en amont

CE QUI A ÉTÉ CORRIGÉ, et qui est juste sur le fond. Le chemin ABSOLU d'i0 consommait 23 bits
au lieu de 47. Deux défauts additifs, tous deux vérifiés sur pièces :

  - `absAxisW` (position_capture.go) retombait sur `pd.AxisW` = 6/6/6, la largeur du chemin
    DELTA. Le chemin absolu lit la table de région, 13/13/14 — les mêmes largeurs que le
    chemin world-object, qui les porte correctement depuis toujours. -22 bits.
  - Le champ « fini » de 2 bits (FUN_14076e304, sous un prédicat à 0 bit donc lu
    systématiquement) n'était jamais lu par le chemin dynamic-precision, alors que le chemin
    world-object le lit. -2 bits.

  23 + 22 + 2 = 47, exactement la vérité CE (47 bits, une seule valeur, 100 % de 154 158
  dispatches). La double implémentation des deux chemins est ce qui les avait laissés diverger.

  Ceci explique aussi pourquoi l'essai précédent (régler `TraversalPrecision.AxisW` à 13/13/14)
  DÉGRADAIT : il changeait aussi la largeur du delta, le chemin dominant.

CE QUE LA MESURE DIT, ET QUI RÉFUTE LE DIAGNOSTIC. Aucun effet :

    i22 présence   avant 11,971 %   après 11,979 %   (vérité 0,19 %)
    records bipède avant 1 771      après 1 771

  Test qui tranche, gratuit : `I22_CALIBSKIP=1` force i0 au total CE exact (47/101) quel que
  soit le chemin — un ORACLE, pas une hypothèse. Résultat : i22 = 12,099 %, bipèdes = 1 777.
  **La largeur d'i0 n'est pas la cause du facteur 63.** Le correctif est conservé parce qu'il
  est juste au regard du désassemblage ET de la mesure CE, pas parce qu'il améliore un chiffre.

LE VRAI DÉFAUT, visible dans le même relevé et que je regardais sans le voir :

    nous décodons  1 777 records de bipède sur 58 335   (3,0 %)
    la vérité CE :  159 772 sur 315 892                  (50,6 %)

  Nous trouvons environ UN record de bipède SUR QUATRE-VINGT-DIX. Le problème n'est ni les
  largeurs de composants ni le masque : **nous ne trouvons pas les records de bipède**. La
  faute est en amont — identification des records, ou résolution slot -> archétype. Toutes
  les corrections de grammaire portaient sur les 3 % qu'on trouve en ignorant les 97 % ratés.

PROCHAINE ÉTAPE : mesurer la distribution des archétypes de NOTRE décodeur contre celle de la
capture CE (ti=35 50,6 % · ti=37 8,3 % · ti=42 3,2 % · ti=41 1,5 %…). L'écart dira si on
attribue le mauvais archétype ou si on rate les records eux-mêmes.

### Le goulot est TROUVÉ : on décroche après ~2 records par paquet

Chaîne de mesures, chacune éliminant une hypothèse :

1. **Le World d'abord.** Un record DELTA ne porte pas son `typeIndex` — l'archétype vient de la
   table slot -> archétype. Mesuré : `WalkKeyframeWorld` (bootstrap hors ligne) lie **123**
   entités ; la vérité CE en compte **2 051**. Il en rate 94 %. Il ne parse pas le keyframe, il
   CHERCHE des ancres valides et sort de la boucle à la première invalide.
   Conséquence mesurée sur le film ancré `9e8fb31b` : **ZÉRO** record de bipède décodé.

2. **World réparé par oracle.** `cmd/tmp_cecapture -worlddump` fabrique la table depuis la
   capture CE (archétype modal par entité ; 2 051 entités, **0 ambiguë**). Rechargée :
   **0 -> 1 325** records de bipède. Nécessaire, donc. Mais la vérité en compte **159 772**.

3. **Le vrai goulot, visible dans le même relevé :**

        nous : 47 053 records sur 23 132 frames  =  2,0 records/frame
        CE   : 315 892 records sur 23 132 frames = 13,7 records/frame

   **On décode les ~2 premiers records de chaque paquet et on perd le reste.** C'est la
   signature d'un décrochage précoce et systématique, pas d'une erreur de grammaire.

CE QUE ÇA EXPLIQUE RÉTROSPECTIVEMENT. Toutes les corrections de composants de ces dernières
sessions ne portaient que sur les 2 records qu'on atteignait déjà — d'où le motif répété
« correction juste, aucun effet », y compris pour i0 aujourd'hui (23 -> 47 bits, verifié à
l'oracle : zéro effet). On réparait la lecture de 15 % du flux en ignorant les 85 % perdus.

PROCHAINE ÉTAPE. Instrumenter la boucle de records : pour chaque paquet, à quel rang et sur
quel motif on s'arrête (type de record invalide ? id hors bande ? masque refusé ? fin de
payload atteinte trop tôt ?). La capture CE donne le témoin exact — pour un paquet donné, elle
dit combien de records le moteur y a lus et lesquels.

CE QUE LE DUMP ORACLE N'EST PAS : il débloque le film capturé, il ne rend PAS le rejeu possible
sur les 948 autres films. Pour ceux-là il faudra réparer `WalkKeyframeWorld` — et ce dump est
justement le témoin qui dira quels slots il rate.

### [2026-07-27] Le JUGE de position existe — et il corrige un « critère absolu » du projet

DOCTRINE REPRISE DU CHANTIER ARMES (`feat/filmdec-killweapon`, handoff §0). Ils ont buté sur
NOTRE goulot — la marche séquentielle décroche tôt dans le paquet, et leur « localisateur
slot-123 » est le même 123 que `WalkKeyframeWorld` nous rend — et l'ont CONTOURNÉ sur une idée
de l'utilisateur : « le fichier ne se lit pas en continu ». Balayer toutes les positions de bit,
ne retenir que ce qui passe plusieurs tests simultanés. Plafond cassé 88,7 % -> 97,6 %.

CE QUI MANQUAIT POUR L'APPLIQUER ICI : un juge. Un scanner sans juge produit des candidats
plausibles et rien ne dit lesquels sont vrais — le piège qui a coûté cher sur la géométrie.

LE JUGE EST CONSTRUIT (`cmd/tmp_comptruth`). La capture CE porte, pour chaque composant
désérialisé, 16 octets bruts lus depuis le pointeur d'octet du bitreader. Ces octets existent à
l'identique dans le film. Résultat sur i22 / ti=35 :

    256 lectures capturées · 249 à signature exploitable
    249 LOCALISÉES SANS AMBIGUÏTÉ · 0 introuvable · 0 ambiguë

On connaît donc, à l'octet près, où le moteur a lu chacune des 249 occurrences. Précision et
rappel deviennent MESURÉS, plus estimés. Fichier de référence : `i22_positions.tsv`.

MESURES DU SCAN DIRECT (`cmd/tmp_i22scan`), et trois corrections de mes propres prévisions :

1. **Mon estimation de faux positifs était fausse.** J'annonçais 0,1 attendu ; le scan rend
   2 087 candidats. Cause : le modèle supposait des bits équiprobables, or le flux est très
   riche en octets nuls (candidats du type `[0 0 0 2]`, `[2 0 0 0]`). Un calcul de hasard sur
   un flux non uniforme ne vaut rien — à ne pas refaire.

2. **Il n'y a PAS de décalage constant** entre le pointeur d'octet et le début des données.
   Amas à -26/-28 (50 cas) et -58/-61 (23 cas), rien de dominant. La conversion repère-octet ->
   repère-bit ne se pose donc pas par une constante.

3. **Le rappel sature à 142/249 (57 %)** et la fenêtre n'y est pour rien : 105 à 48 bits, 142 à
   96, 142 à 192, 143 à 384. En revanche l'unicité est excellente — quand le filtre accepte,
   il accepte UNE seule position (141 sur 142).

CE QUE ÇA ÉTABLIT, ET QUI CORRIGE LE PROJET. `GRAMMAIRE_RECORD_FILM.md §6` pose comme
**CRITÈRE ABSOLU** : au plus deux types de grenade portés, deux unités de chacun. La mesure dit
que **43 % des lectures d'i22 réellement effectuées par le moteur ne le respectent pas** (ou
que leur décalage sort de toute fenêtre raisonnable, ce que la saturation rend improbable).

Contrôle qui verrouille l'interprétation : en relâchant à la seule contrainte structurelle
(compteur == 4, établie par les 35 bits fixes de la capture), les 249 positions ont un parse
valide mais **ZÉRO n'en a un seul** — normal, un champ de 3 bits passe une fois sur huit. Ce
sont donc bien les bornes de JEU qui font la sélectivité, et le mode Super Fiesta ne les
invalide pas globalement : elles sont justes pour 57 % des lectures, fausses pour le reste.

PROCHAINE ÉTAPE : ne plus traiter la borne comme un absolu. Deux voies mesurables —
(a) établir la vraie distribution des valeurs d'i22 en capturant la VALEUR désérialisée (le
crochet actuel ne capture que le curseur et la signature ; ajouter la lecture du composant
donnerait la distribution exacte) ; (b) chercher un second test indépendant (position relative
à un i25, qui est présent dans 99,5 % des records) pour remplacer la sélectivité perdue.

### CORRECTION IMMÉDIATE + L'INVENTAIRE EST DÉCODÉ (249/249)

**J'AI ÉCRIT UNE HEURE PLUS TÔT que « 43 % des lectures d'i22 violent le critère absolu du
projet ». C'EST FAUX.** Le critère est parfaitement respecté ; c'était mon filtre qui était
mauvais. La ligne fautive, dans mon propre scanner :

    if nz == 0 { return v, false }   // « aucun type porté : sans valeur ici »

Elle jetait 107 des 249 lectures réelles. Un joueur qui ne porte AUCUNE grenade est un état
parfaitement normal — juste après un lancer, et très fréquent en Fiesta.

LA VOIE DIRECTE, établie et exacte. Le curseur de bits capturé EST la position absolue dans le
payload du paquet, décalage NUL. Balayage de l'amorce sur 0..8 : seul `+0` produit le moindre
parse valide, et il en produit 249 sur 249. Ce n'est pas un ajustement, c'est une identité.

    position_exacte = paquet.Start*8 + curseur_moteur

VALEURS RÉELLES relues aux 249 positions exactes, sans aucun filtre :

    compteur R(3)             : 4 dans 100 % des cas  (conforme aux 35 bits fixes mesurés)
    valeur maximale observée   : 2
    valeurs vues               : 0 (826x), 2 (122x), 1 (48x) — jamais rien d'autre
    types non nuls par lecture : 0 -> 107 · 1 -> 114 · 2 -> 28

**Les bornes du jeu sont donc EXACTES** : jamais plus de 2 unités, jamais plus de 2 types,
compteur toujours 4. Le §6 de GRAMMAIRE_RECORD_FILM n'a pas à être corrigé — c'est mon entrée
précédente qui l'accusait à tort, et elle est annulée par celle-ci.

DEUX VOIES, ET LEUR PORTÉE RESPECTIVE :

    voie curseur (capture CE)  249/249 · positions et valeurs EXACTES · film capturé seulement
    scan direct (sans capture) 249/249 de rappel · 5 085 candidats, soit 20,4 par vraie lecture

Le scan a donc un rappel PARFAIT et une précision insuffisante. C'est exactement la situation du
chantier armes avant leurs quatre tests simultanés — il faut des tests supplémentaires, pas une
autre méthode. Pistes mesurables, par ordre de force attendue :
  (a) co-occurrence i22 / i47 : ils partagent 84 % / 93 % des records denses, donc un i22 vrai a
      un i47 valide à courte distance ;
  (b) cohérence par joueur : les comptes évoluent de +-1 (lancer, ramassage), une suite de
      valeurs incohérente dénonce un faux ;
  (c) appartenance à un record dont le masque (<= 7 indices de 6 bits) contient bien 22.

PROCHAINE ÉTAPE : porter la voie curseur en oracle de validation, et instruire (a) puis (b) sur
le scan pour le rendre utilisable sans capture — c'est-à-dire sur les 948 films.

### [2026-07-27] i47 CRAQUÉ — le loadout de grenades est lu. Et i57 donne l'état actif.

MÉTHODE, désormais générale : `cmd/tmp_comptruth -comp N -bits W` localise chaque lecture par
la signature CE, puis lit la valeur à `paquet.Start*8 + curseur`. L'identité (décalage NUL) a
été établie sur i22 par balayage — seul +0 rend un parse valide, 249/249 — et elle vaut pour
TOUT composant.

#### i47 (biped-desired-grenade-set) — 190 lectures, 12 valeurs distinctes SEULEMENT

    i47 = [6 bits : masque des types possédés][3 bits : type SÉLECTIONNÉ]

    0x000  masque 000000  aucun type          selection 0
    0x009  masque 000001  type 1              selection 1
    0x012  masque 000010  type 2              selection 2
    0x023  masque 000100  type 3              selection 3
    0x044  masque 001000  type 4              selection 4
    0x019  masque 000011  types 1 et 2        selection 1
    0x02B  masque 000101  types 1 et 3        selection 3
    0x033  masque 000110  types 2 et 3        selection 3
    0x052  masque 001010  types 2 et 4        selection 2
    0x054  masque 001010  types 2 et 4        selection 4
    0x063  masque 001100  types 3 et 4        selection 3
    0x064  masque 001100  types 3 et 4        selection 4

TROIS VÉRIFICATIONS INDÉPENDANTES, et c'est ce qui fait tenir la lecture :
  1. **Le type sélectionné appartient TOUJOURS au masque** — 12 fois sur 12, sans exception.
     Une décomposition fausse produirait des sélections hors masque.
  2. **Les 2 bits de poids fort du masque sont toujours nuls** — 4 types de grenades pour un
     champ de 6 bits, exactement ce qu'on attend.
  3. **Recoupement avec i22** : 50,5 % des i47 ont un masque VIDE, i22 a 43 % de comptes tous
     nuls. Deux composants indépendants racontent le même fait (joueur sans grenade).

C'est la confirmation directe de l'intuition de l'utilisateur : « un seul type de grenade actif
à l'instant T ». Le masque dit ce qu'on POSSÈDE, les 3 bits disent ce qu'on a EN MAIN.

#### i57 (biped-spartan-ability) — 397 lectures, l'ÉTAT ACTIF

Taux de bits à 1 par position : **bit 0 = 48 %, bit 1 = 4 %**. Le premier bit est donc
l'interrupteur marche/arrêt (capacité active ~la moitié du temps), le second un drapeau rare.
Valeurs : 0 (49,4 %), 2 (47,1 %), 1 (3,0 %), 3 (0,5 %).

#### i48 / i42 / i56 — OBSERVÉ, PAS ENCORE DÉCODÉ (à ne pas publier en l'état)

    i48  203 lectures, 18 valeurs. 0x293/0x294/0x295/0x296 CONSÉCUTIVES (44 % à elles quatre),
         idem 0x340/0x344 et 0x362/0x365. Ressemble à [identifiant][petit index] — MAIS quatre
         valeurs consécutives peuvent aussi être quatre capacités voisines dans une table.
         NE PAS TRANCHER sans un second témoin.
    i42  385 lectures, 18 valeurs, TOUTES IMPAIRES (97, 99, 29, 17, 19, 61, 117, 51, 49, 81).
         Le bit de poids faible est un drapeau, pas de la donnée.
    i56  202 lectures, deux régimes nets : ~68-77 et ~544-576. Possiblement deux capacités à
         échelles d'énergie différentes.

### LA RECETTE — tableau complet des composants du bipède, mesuré en une passe

`cmd/tmp_compsweep` publie, pour chaque composant : lectures, largeur mesurée, valeurs
distinctes, bits morts. Une seule passe sur le film (index de hachage sur les 8 premiers octets
de signature — chercher composant par composant aurait coûté des dizaines de Go de balayage).
Sortie complète : `.ai/re_dump/ce_recette_composants.txt`.

TROIS ENSEIGNEMENTS DE FOND :

1. **i42 n'a que 7 valeurs distinctes sur 447 lectures.** C'est un SÉLECTEUR D'EMPLACEMENT, pas
   un identifiant d'arme. Directement énumérable pour primaire / secondaire.

2. **Les bits morts bornent les vrais champs, sans rien supposer.** i22 déclare 35 bits dont
   **26 toujours nuls** — cohérent avec des comptes valant seulement 0, 1 ou 2 (verifie par
   ailleurs). i5 déclare 29 bits dont 14 morts, i23 19 dont 15, i24 12 dont 10. On peut
   resserrer les grammaires sur mesure au lieu de porter des largeurs de Ghidra.

3. **i43/i44 sont les SEULS champs vraiment variables** : 11 et 10 largeurs distinctes contre
   1 ou 2 partout ailleurs. C'est là qu'est l'arme tenue, et sa forme longue (~200 bits) porte
   le descripteur complet. Tout le reste de la table est a largeur fixe ou binaire.

ÉNUMÉRABLES TOUT DE SUITE (peu de valeurs, largeur stable) :
    i57 capacite ACTIVE      990 lectures · 2 bits · 2 valeurs   -> cadre dore / effet verre
    i54 mobilite            1347 lectures · 2 bits · 4 valeurs   -> accroupi / sprint / glissade
    i42 arme desiree         447 lectures · 7 bits · 7 valeurs   -> primaire / secondaire
    i48 capacite equipee     218 lectures · 10 bits · 13 valeurs
    i47 grenades             215 lectures · 9 bits · 8 valeurs   -> CRAQUE (masque + selection)
    i35/i32 surchauffe      4948/3146 · 9 bits · 4 et 3 valeurs
    i22 comptes grenades     256 lectures · 35 bits · 10 valeurs -> CRAQUE

RÉSERVE À GARDER EN TÊTE : tout ceci est mesuré sur UN film (9e8fb31b, Cliffhanger Super
Fiesta). Les largeurs sont des valeurs de RUNTIME sur certains champs (cf. IDLowBits) ; avant
de figer une grammaire, refaire la mesure sur un second film d'un autre mode.

### RÉFUTÉ — un catalogue ne rend PAS sélectif un champ étroit

L'utilisateur signale que le chantier armes a trouvé « la solution universelle ». Lecture faite
(RE_LOG 7ter.73) : **« la marche est un SOUS-ENSEMBLE POSITIONNEL TOTAL du scan », 346/346 sur
quatre films.** Tout ce que la marche séquentielle trouve, le scan le trouve AU MÊME BIT. Et le
mécanisme de leur sélectivité y est nommé : « un enregistrement de la marche porte forcément un
tag DU CATALOGUE, sinon le scan ne l'aurait pas produit au même bit ».

J'AI TRANSPOSÉ, ET LA MESURE RÉFUTE LA TRANSPOSITION. `cmd/tmp_catscan`, i47 :

    catalogue   25 valeurs sur 512 possibles (bati sur les positions EXACTES)
    candidats   3 202 042  pour 179 lectures vraies
    rappel      100,0 %      precision  0,01 %
    faux positifs MESURES (comptes, pas calcules) : 7,3e-02 par position de bit

LA CAUSE EST ARITHMÉTIQUE, et j'aurais dû la voir avant de coder :

    chantier armes : catalogue de 468 valeurs sur un champ de 32 bits  -> 1 sur 9 000 000
    moi            : catalogue de  25 valeurs sur un champ de  9 bits  -> 1 sur 20

**La sélectivité vient de la LARGEUR du champ, pas de la qualité du catalogue.** Sur 43 millions
de positions, un champ de 9 bits rend deux millions de candidats quoi qu'on fasse. Aucun soin
apporté à la liste ne peut compenser. L'invariant structurel d'i47 (sélection dans le masque)
ne rattrape rien non plus : il est déjà satisfait par les valeurs du catalogue.

CE QUI RESTE VALIDE DE LEUR DOCTRINE, et c'est l'essentiel : abandonner la marche séquentielle.
Leur 346/346 l'établit — la marche ne sert qu'à la précision, jamais à la couverture.

LA VOIE QUI RESTE : combiner. i22, i47, i42, i48 coexistent dans le même record de spawn
(co-occurrence mesurée : i43 93 %, i47 85 %, i22 75 %, i42 44 %, i48 44 %), les composants sont
dispatchés par index CROISSANT et leurs largeurs sont mesurées — donc les distances entre eux
sont prédictibles. Quatre champs étroits à distances imposées forment un champ large.
Sélectivité attendue, à MESURER et non à calculer : i47 (1/20) x i42 (1/18) x i48 (1/79) plus
les bornes d'i22.

RÉSERVE MÉTHODOLOGIQUE À NE PAS OUBLIER : c'est la deuxième fois aujourd'hui qu'un calcul de
probabilité me trompe. Le taux de faux positifs de cet outil est COMPTÉ sur le flux réel, pas
déduit d'un modèle de bits équiprobables.

### La distance i22 -> i47 VALIDE le modèle, mais ne suffit pas comme test

`cmd/tmp_codist` mesure l'écart de curseur entre deux composants d'un même record. Sur i22 -> i47 :

    96 records portent les deux · 37 portent i22 seul · ZERO porte i47 seul
    22 ecarts distincts, dominante +45 bits (27,1 %), quatre premiers = 60 %
    54 masques distincts parmi ces records

DEUX ACQUIS.

1. **L'écart dominant se décompose EXACTEMENT.** i22 fait 35 bits, i25 en fait 10 : 35 + 10 = 45.
   Le cas majoritaire est donc « i22, puis i25, puis i47 ». Le modèle « distance = somme des
   largeurs des composants intermédiaires PRÉSENTS » est confirmé au bit près, sur des largeurs
   mesurées indépendamment. C'est une validation croisée, pas un ajustement.

2. **i47 n'apparaît JAMAIS sans i22** — 96 fois avec, 0 fois sans, sur les records où l'un des
   deux est présent. Invariant structurel fort et gratuit : tout candidat i47 sans i22 en amont
   à une distance plausible est un faux.

CE QUE ÇA NE DONNE PAS. 22 écarts distincts dont les quatre premiers ne couvrent que 60 % : un
test à distance FIXE ne selectionne pas assez. Estimation (à mesurer, pas à croire) : le scan
i22 rend 5 085 candidats pour 249 vraies ; ajouter « i47 valide à l'un des 4 écarts dominants »
laisserait ~990 candidats pour ~150 vraies, soit 6,6 par vraie au lieu de 20,4. Mieux, pas assez.

LA VOIE : énumérer les MASQUES plutôt que les distances. Ils sont 54 sur ces records, mais la
grammaire les borne — compteur de 3 bits, donc au plus 7 composants — et les largeurs sont
toutes mesurées. Dérouler un record depuis un candidat i22 en essayant les masques plausibles
donne un test bien plus fort que n'importe quelle distance isolée.

### RÉFUTÉ AUSSI — la multiplicité de position ne discrimine RIEN sur l'inventaire

Le rapport d'extraction désigne la multiplicité de position comme « le discriminant le plus
rentable, et il est gratuit » (chantier armes, fc_self.go) : un vrai record tombe à un bit
quelconque, un faux se répète AU MÊME BIT d'un paquet à l'autre (leurs faux : bits 746, 674,
720, 77, répétés sur 9 à 13 paquets).

MESURE SUR i22 (`cmd/tmp_multiplicite`) — les deux distributions sont IDENTIQUES :

    candidats VRAIS a multiplicite 1 :  104 /  236 = 44,1 %
    candidats FAUX  a multiplicite 1 : 2194 / 4854 = 45,2 %

    seuil s :   1     2     3     4     5     7     9    12
    cand/vrai: 22,1  23,7  25,5  25,3  25,8  25,1  24,5  24,8

Le rapport candidats/vrai ne descend JAMAIS sous 22, quel que soit le seuil. Le discriminant ne
porte aucune information ici.

LA CAUSE, une fois posée : leurs faux positifs viennent d'un motif de 32 bits revenant à offsets
FIXES (en-têtes de paquet, bourrage, champs de longueur) — d'où la concentration. Les miens
viennent de la DENSITÉ d'un filtre trop lâche sur 43 millions de positions : ils sont répartis
partout, donc leur multiplicité suit la même loi que celle des vrais.

BILAN DES TRANSPOSITIONS : deux tentées, deux réfutées, chacune avec sa cause mesurée.
  1. catalogue de valeurs -> inopérant sur un champ de 9 bits (rapport alphabet/catalogue)
  2. multiplicité de position -> distributions identiques, zéro information

CE QUI RESTE DEBOUT, et que j'avais mal lu au départ : LE GABARIT RIGIDE. Le scan du chantier
armes ne teste pas des champs isolés — il plaque 58 bits de structure complète à chaque
position, avec **quatre bits de porte à VALEUR IMPOSÉE** (tête `11`, deux gates d'enum à `0`).
Ces bits de porte sont de la contrainte gratuite qu'aucun catalogue ni aucune statistique ne
fournit, et leur sémantique est LUE dans la grammaire du désérialiseur, pas devinée.

Leur validation de ce gabarit est exemplaire et transposable : décaler le bloc de `delta` bits
et mesurer le TAUX D'APPARIEMENT (0 à 4,3 % hors delta 0, contre 80,7 à 92,0 % à delta 0). Le
COMPTE de candidats, lui, ne tranche pas — il produit des échos à +-6 bits.

PROCHAINE ÉTAPE : bâtir le gabarit d'un record portant i22 — en-tête de record, masque, portes
et largeurs des composants du masque — et le valider par le même balayage de décalage.

### LE GABARIT RIGIDE MARCHE — precision 4,9 % -> 96,7 %

Apres deux transpositions refutees (catalogue de valeurs, multiplicite de position), la
troisieme tient : plaquer la STRUCTURE COMPLETE d'un record delta a chaque position, au lieu de
tester des champs isoles.

    approche                    candidats    vrais   precision
    bornes du jeu seules            5 085      249       4,9 %
    catalogue de valeurs        3 202 042      179       0,01 %
    GABARIT RIGIDE                     61       59      96,7 %

**1,03 candidat par lecture vraie.** Le rappel tombe a 23,7 % (59/249) parce que le gabarit
exige que tous les composants intermediaires aient une largeur connue — reparable en completant
la table, alors que la precision etait le vrai verrou.

LA CONTRAINTE GRATUITE QUE JE N'EXPLOITAIS PAS : les index du masque sont STRICTEMENT
CROISSANTS (les composants sont dispatches par index croissant). Quatre valeurs de 6 bits au
hasard n'ont qu'une chance sur 4! = 24 d'etre croissantes — 4,6 bits de contrainte, gratuits.
C'est exactement la nature des bits de porte imposes qui fait marcher leur gabarit.

CALIBRATION GRATUITE EN PRIME : `idLow` se determine tout seul. Seules les valeurs 10 et 14
produisent des candidats (61 et 63, 59 apparies chacune) ; 9, 11, 12, 13 rendent 1 ou 2
candidats et ZERO appariement. Et 10 et 14 sont la meme solution — idLow=10 decale de +4 EST
idLow=14, ce que le balayage confirme.

VALIDATION PAR BALAYAGE — HONNETE, ELLE NE TRANCHE PAS AUSSI NET QUE CHEZ EUX :

    decalage   -3    -2    -1    +0    +1    +2    +3    +4    +6
    apparies   21     0     0    59     0     0     0    59    33
    taux      91 %   0 %   0 %  97 %   0 %   0 %   0 %  94 %  97 %

Le decalage 0 a le MAXIMUM d'apparies (59) et les voisins immediats s'effondrent a 0 %. Mais
-8..-3 et +5..+8 gardent 85 a 97 % de taux avec 21 a 36 apparies : ce sont des alignements
partiels qui retrouvent une PARTIE des memes records. CAUSE : mon en-tete laisse idLow et le tag
NON CONTRAINTS, alors que leur gabarit verrouille la tete `11` et deux portes d'enumeration a 0.
Il manque donc des bits imposes en amont du masque.

PROCHAINE ETAPE : (a) completer la table des largeurs pour remonter le rappel ; (b) contraindre
l'en-tete de record (le tag R(2), la plage de l'identifiant) pour eteindre les alignements
partiels ; (c) generaliser le gabarit a i47/i48/i42 — la mecanique est la meme, seul le
composant vise change.

### Gabarit : la PRÉCISION est résolue, le RAPPEL ne l'est pas — état exact

    approche                    candidats    vrais   precision   rappel
    bornes du jeu seules            5 085      249       4,9 %    100 %
    catalogue de valeurs        3 202 042      179       0,01 %    72 %
    GABARIT (branche eparse)           61       59      96,7 %    23,7 %
    GABARIT (+ branche dense)          62       59      95,2 %    23,7 %

TROIS TENTATIVES POUR REMONTER LE RAPPEL, TOUTES SANS EFFET, et c'est utile de le savoir :

  1. **Compléter la table des largeurs** (i6=358, i9=334, i15=72, i17=52, tous d'index < 22
     donc sur le chemin d'i22). Aucun changement : 61 candidats, 59 apparies.
  2. **Ajouter la branche DENSE du masque.** Motivation solide — 75 % des lectures d'i22 vivent
     dans des records a masque dense, et un masque de plus de 7 composants NE PEUT PAS etre
     epars (compteur de 3 bits). Resultat : +1 candidat, +0 appariement.
  3. **Corriger l'ordre des bits du masque dense** (le composant i au bit p+i, et non un entier
     64 bits teste par 1<<i, qui mettait le composant 0 sur le DERNIER bit lu). Meme resultat.

CE QUE ÇA DÉSIGNE. Le gabarit doit traverser tous les composants precedant i22 en additionnant
leurs largeurs. Dans un record DENSE il y en a beaucoup, et il suffit qu'UN SEUL emprunte une
largeur minoritaire pour que la somme soit fausse et le gabarit echoue. Or la table ne retient
que la largeur DOMINANTE, et plusieurs composants du chemin sont a largeurs multiples :
i11 (6 largeurs), i17 (5), i15 (2), i18 (2), i23 (2), i24 (2), i26 (2), i28 (3), i30 (3).

PISTE MESURABLE POUR LA SUITE : au lieu d'une largeur par composant, essayer TOUTES les largeurs
observees de chaque composant du chemin (produit cartesien borne — au plus quelques dizaines de
combinaisons pour un masque typique) et retenir celle qui fait tomber i22 sur des valeurs
valides. Le gabarit devient une petite recherche au lieu d'un calcul unique, et sa precision
actuelle (96,7 %) donne la marge pour se le permettre.

À NE PAS PERDRE DE VUE : la precision etait le verrou, et elle est levee — 1,03 candidat par
lecture vraie contre 20,4. Le rappel se rattrape en elargissant le gabarit ; l'inverse (gagner
en precision) n'avait pas de solution connue avant aujourd'hui.

### Le rappel est plafonné parce que le gabarit ne teste QUE les records DELTA

QUATRE TENTATIVES POUR REMONTER LE RAPPEL, TOUTES MARGINALES :

    tentative                                      candidats  apparies  rappel
    gabarit epars, largeur dominante                      61        59   23,7 %
    + table des largeurs completee (i6,i9,i15,i17)        61        59   23,7 %
    + branche DENSE du masque                             62        59   23,7 %
    + ordre des bits du masque dense corrige              62        59   23,7 %
    + recherche sur TOUTES les largeurs observees         65        60   24,1 %

Aucune de ces causes n'etait la bonne. L'explication etait dans mes propres mesures du matin,
et je ne l'avais pas reliee :

**Les gros records portent `param_4 = 3` — 24 fois chez eux, ZERO sur 36 606 deltas.** J'en
avais conclu qu'ils sont D'UNE NATURE DIFFERENTE, vraisemblablement des records **NEW** (type 1,
nouvelle entite, etat complet transmis) et non des deltas enrichis.

**Or mon gabarit commence par `prefixe == 1`, c'est-a-dire DELTA.** Il rejette donc TOUS les NEW
au premier bit. Les 75 % de lectures d'i22 que je rate sont exactement les records de spawn,
qui sont des NEW. Tout concorde : nouvelle entite -> etat complet -> inventaire inclus.

CE QUE ÇA IMPLIQUE POUR LA SUITE. Le gabarit NEW a un en-tete different (frame_records.go) :

    prefixe R(1) == 0 puis R(2) == 1        -> NEW              [3 bits imposes, mieux que DELTA]
    [si hasExtraFields && R(1) : R(8)]
    R(6) typeIndex                          == 35 pour un bipede [6 bits imposes !]
    default-state                           largeur de RUNTIME   <- LE VERROU
    gate + masque + composants

Deux bonnes nouvelles et un verrou. Le typeIndex sur 6 bits vaut 35 pour un bipede : c'est SIX
bits imposes, davantage que tout ce dont dispose le gabarit DELTA. Et le prefixe NEW en impose
trois au lieu d'un. Mais le default-state a une largeur de RUNTIME non sourcee statiquement —
c'est elle qu'il faudra calibrer, et la capture CE la donne : l'ecart entre le curseur du
premier composant d'un record NEW et le debut du record EST cette largeur.

PROCHAINE ETAPE, precise et mesurable : extraire de la capture les records dont le masque
depasse 7 composants, mesurer la distance entre le debut du record et le premier composant, et
en deduire la largeur du default-state. Puis ajouter la branche NEW au gabarit.

### GABARIT LOCAL — l'utilisateur corrige l'approche, et le compromis s'inverse

CORRECTION DE L'UTILISATEUR : « le truc qui marchait bien sur l'autre worktree, c'est qu'on
n'avait pas besoin de tout parcourir pour atteindre le composant désiré ».

IL A RAISON, et la relecture de leur gabarit de 58 bits le confirme : `mort`, `porte du tag`,
`tag`, `porte victime`, `victime`, `porte tueur`, `tueur`, `catégorie` sont TOUS des champs
INTERNES au composant i11. Ils ne touchent ni l'en-tête du record ni le masque. Mon gabarit
partait du début du record et traversait tout — d'où sa fragilité et son rappel de 24 %.

MESURE DU GABARIT LOCAL (`cmd/tmp_gablocal`), ancré sur i22 et prolongé vers l'avant :

    population                 candidats   vrais   precision   rappel
    avec prolongement i47           4 178     231       5,5 %    97,9 %
    sans prolongement                 907       5       0,6 %     2,1 %

LE COMPROMIS S'INVERSE : rappel 97,9 % contre 24 %, mais precision 5,5 % contre 96,7 %. Le
prolongement CONCENTRE bien les vrais (5,5 % contre 0,6 % pour les candidats nus, facteur 9),
mais il accepte trop.

CAUSE, chiffrée : en essayant toutes les largeurs de maillon, je teste i47 a une vingtaine
d'offsets. Or i47 n'accepte que 33 valeurs sur 512 — une sur 15,5 — donc a vingt essais la
probabilite qu'aucun ne passe est faible. Le test se dilue dans ses propres tentatives.

LA LIMITE DE FOND, qu'il faut nommer : leur selectivite vient d'un CHAMP LARGE ET CATALOGUE (tag
de 32 bits, 468 valeurs admises, 1 accident sur 9 000 000). Autour d'i22 je n'ai que des champs
de 2 a 10 bits, dont aucun ne peut fournir cet ordre de grandeur — c'est la meme limite
arithmetique que celle qui a fait echouer le scan par catalogue.

SAUF UN, ET C'EST LA PROCHAINE ETAPE : i43 (weapon-state-type-info, arme tenue) a une forme
LONGUE de ~195 a 204 bits, et il est present dans 93 % des records denses portant i22 — le taux
de co-occurrence le plus eleve de toute la table. C'est le seul champ large du voisinage, donc
le seul equivalent possible de leur tag 32 bits. S'il contient un identifiant d'arme
catalogable, l'ancrage doit se faire SUR LUI, et i22 se lit ensuite par distance.

REMARQUE DE METHODE : deux gabarits, deux compromis opposes (96,7 % de precision pour 24 % de
rappel · 5,5 % pour 97,9 %). Leur UNION n'est pas la solution — c'est le signe qu'il manque un
test, pas qu'il faut choisir entre les deux.

### [2026-07-27] Le sol reconstruit est ENFIN dans le POC — et deux erreurs de ma part corrigées

L'utilisateur signale qu'il ne voit ni la carte affinée ni l'inventaire dans le POC. Vérification
faite, IL AVAIT RAISON SUR LES DEUX, et j'ai commis deux erreurs en cherchant pourquoi.

**ERREUR 1 — j'annonçais « structure livrée » pour des BOÎTES.** Le POC affichait 9 630 boîtes
englobantes produites par `cmd/mapstruct-build`, dont l'en-tête dit lui-même : « le lien
instance -> maillage n'étant pas résolu, on ne publie PAS de forme, seulement des boîtes
englobantes ». Ce n'est pas la carte. Mon item de suivi était trompeur.

**ERREUR 2 — j'ai conclu que la carte concernait un AUTRE terrain.** Tous les artefacts de
géométrie sont nommés `*_ridgeline.*` et le POC affiche `Cliffhanger` : j'en ai déduit deux
cartes différentes et je l'ai annoncé. **C'était faux.** Le document porte les deux noms —
`"map":"Cliffhanger"` ET `"mapFolder":"ridgeline"` — c'est la MÊME carte, nom affiché contre nom
de module. Le contrôle qui tranchait en deux secondes : les callouts du POC contiennent « Fer à
cheval » et « Pont », les zones mêmes de l'étude.

**CE QUI EST FAIT.** Le sol reconstruit est injecté :

    source      sc/layers_marchable.npy — 738 757 cellules praticables (28,9 M triangles)
    decoupe     28 zones de callout -> 431 968 cellules (58,5 % du masque)
    recadrage   zone de jeu, 550x610 px a 10 cm  ->  PNG de 12,8 Ko
    reperage    coin haut-gauche (X=-9, Y=+33), coin bas-droit (X=+46, Y=-28)

Les boîtes englobantes ne se dessinent plus quand le sol est actif : les superposer masquerait
ce qu'on vient de tracer, et elles ne sont qu'un substitut.

**CE QUI N'EST TOUJOURS PAS DANS LE POC, et c'est dit sans détour** : l'inventaire équipé. `i47`
et `i48` sont décodés, mais uniquement via la capture Cheat Engine — le scan hors ligne plafonne
soit en précision (5,5 %) soit en rappel (24 %) selon le gabarit. Rien de publiable.

**DOCUMENT CRÉÉ POUR QUE ÇA NE SE REPRODUISE PAS** : `.ai/ETAT_DU_POC.md`. Il liste calque par
calque ce qui est affiché, d'où viennent les données, ce qui est absent et pourquoi. Il consigne
les deux pièges de vocabulaire (« structure » = boîtes ; « grenades » = lancers, pas inventaire)
et le piège de nom de carte. Règle posée : ne jamais écrire « X est dans le POC » sans avoir
ouvert le fichier et cherché la FONCTION DE DESSIN — une donnée embarquée mais jamais tracée est
exactement ce qui s'est produit avec `spoly`.

### [2026-07-27] La carte validée est DANS le POC — et le calage est reproductible

CE QUI A ÉTÉ FAIT DE TRAVERS AVANT D'Y ARRIVER, pour ne pas le refaire. J'ai voulu
REFABRIQUER la carte depuis les données brutes au lieu de reprendre l'image que l'utilisateur
avait déjà validée. Deux rendus successifs, tous deux mauvais :

  1er essai : fusion de TOUTES les altitudes en un aplat unique -> bouillie illisible ;
  2e essai  : face supérieure par cellule + relief en niveaux de gris -> toujours pas la bonne.

La bonne image existait depuis le 26/07 dans l'artefact validé et dans mon propre dossier
(`structure_callouts.png`, 618 987 octets). **La reprendre était la première chose à faire.**

LE CALAGE, ET POURQUOI IL EST FIABLE. Le script qui a produit l'image est introuvable, donc son
emprise monde n'est pas écrite quelque part. Deux méthodes essayées :

  (a) apparier la boîte des pixels opaques à celle des callouts -> ÉCHEC, 4,93 % d'écart entre
      l'échelle X et l'échelle Y, signe que le rendu porte des marges non symétriques ;
  (b) **ajuster sur les TRAJECTOIRES des joueurs** — elles doivent tomber sur le sol. Recherche
      sur trois paramètres (échelle unique, X0, Y1) en maximisant la part de positions sur
      pixel opaque.

Résultat de (b) : **95,6 % des 29 217 positions de joueur tombent sur le sol.**

    echelle  0,0920 m/px
    X0       -43,5      Y1  61,0
    emprise  X -43,5 .. 74,4   Y -58,0 .. 61,0
    recadrage POC : X -9..46, Y -28..33  ->  598x663 px, 539 Ko

C'est un critère OBJECTIF et non circulaire : les trajectoires viennent du film, l'image vient
de la géométrie du BSP, les deux sources sont indépendantes.

FICHIERS CONSERVÉS AU DÉPÔT (le scratchpad est temporaire) :
    .ai/re_dump/carte_validee_v1.png         l'image validée, pleine (1282x1293)
    .ai/re_dump/carte_validee_recadree.png   recadrée pour le POC (598x663)

STATUT : validé par l'utilisateur pour le POC/V1. Des passes plus précises viendront plus tard.

### [2026-07-27] L'ARME EN MAIN EST DÉCODÉE ET NOMMÉE — et c'est l'ancre large qui manquait

CE QUE JE CHERCHAIS. Le gabarit rigide plafonnait : 96,7 % de précision pour 24 % de rappel, ou
l'inverse en ancrage local. Il manquait un CHAMP LARGE, la sélectivité venant de la largeur du
champ et non de la qualité du catalogue. Seul candidat du voisinage : `i43`
(weapon-state-type-info), forme longue de ~200 bits, co-occurrence 93 % avec i22.

LE TEST. Le chantier armes a établi que les identifiants d'arme du film portent tous le suffixe
`0x42C9679F` sur leurs 32 bits bas. Recherche de cette constante autour des 192 positions
exactes d'i43 (localisées par signature) :

    suffixe sur les positions REELLES : 93 / 192 = 48,4 %
    CONTROLE, positions decalees      :  0 / 192 =  0,0 %
    offset dominant                   : +33 bits, 85 fois sur 93 = 91,4 %

**Zéro faux positif au contrôle.** L'identifiant 64 bits est donc à place FIXE : 32 bits hauts
à +1, suffixe à +33. Et les 48,4 % recoupent la répartition des largeurs mesurée le matin même
— la forme courte (15 bits, 44 % des lectures) ne porte pas d'identifiant, la forme longue si.
Deux mesures indépendantes qui se referment.

LES VALEURS — 19 identifiants distincts sur 85 lectures, et **16 sur 16 des plus fréquents sont
nommés** par `weapon_families.json`, table bâtie par un chemin de décodage TOTALEMENT INDÉPENDANT
(balayage des keyframes, pas le dispatch des composants) :

    0xF408190F 11x Mk51 Sidekick   0xC3542946 8x Plasma Pistol   0x0A1992BC 7x S7 Sniper
    0x71AB0A2C  6x M41 SPNKr       0x0D20C469 6x Skewer          0x48C19D2D 6x MA40 AR
    0xB533957E  5x Needler         0xC30D87C7 5x Ravager         0x30484EA6 4x Pulse Carbine
    0x9387A8B9  4x Shock Rifle     0x2B1824D5 4x BR75            0x767DB96D 4x MLRS-2 Hydra
    0x230447B1  3x Cindershot      0x80977BA5 2x Mangler         0xDAF193C7 2x Stalker Rifle
    0xA0955E9E  2x Sentinel Beam

La dispersion est exactement celle d'un Super Fiesta : tout l'arsenal, quelques occurrences
chacun. Ce n'est pas un ajustement — c'est une validation croisée entre deux décodages qui ne
partagent aucune étape.

CE QUE ÇA DÉBLOQUE, DANS L'ORDRE :

1. **L'arme en main pour le rejeu** (ta consigne : arme à gauche, négatif, primaire soulignée).
   Elle est lue et nommée, il reste à la câbler.
2. **L'ancre large pour l'inventaire.** Un motif de 32 bits à valeur imposée, du même ordre de
   sélectivité que celui qui fait marcher le scan du chantier armes — et `i43` co-occurre avec
   `i22` dans 93 % des records de spawn. Scanner le suffixe donne donc les records d'inventaire
   sans dépendre de la capture.

PROCHAINE ÉTAPE : scanner le film entier pour le suffixe `0x42C9679F`, mesurer précision et
rappel contre les 192 positions connues d'i43, puis remonter d'i43 vers i22/i47/i48 par les
distances mesurées.

### [2026-07-27] Le LOADOUT D'ARMES est structuré : deux emplacements + un sélecteur

QUESTION DE L'UTILISATEUR : quelle est la corrélation entre l'arme en main et celles du loadout ?

CO-OCCURRENCE MESURÉE (`cmd/tmp_codist`) :

    i43 <-> i44 : 77 ensemble · 12 i43 seul · 9 i44 seul   -> DEUX EMPLACEMENTS APPARIES
    i43 <-> i45 : 0 ensemble                                -> i45 n est PAS un 3e emplacement
    i43 <-> i46 : 0 ensemble
    i42 <-> i43 : 89 ensemble · i43 JAMAIS seul (0 sur 89)  -> le selecteur accompagne toujours

CONTENU DES PAIRES (`cmd/tmp_weaponpair`), 65 records portant deux identifiants lisibles :

    ZERO record ou les deux emplacements portent LA MEME arme.

C'est le test qui pouvait réfuter le modèle, et il ne le réfute pas. Échantillon :

    i42=99  Hydra + Sidekick      i42=97  Mangler + BR75       i42=99  Skewer + Ravager
    i42=97  BR75 + MA40 AR        i42=99  Shock Rifle + Cinder i42=97  Needler + Cindershot

Paires plausibles de Fiesta : une lourde et une légère, ou deux portées différentes.

LE SÉLECTEUR SE PARTITIONNE EN DEUX :

    i42 =  99   30 records        i42 = 117   4 records
    i42 =  97   29 records        i42 =  65 / 81   1 chacun

**30 contre 29**, et `99 − 97 = 2` — un seul bit d'écart. C'est la forme attendue d'un sélecteur
à deux emplacements : le bit de valeur 2 dit lequel des deux est en main.

CE QUI EST ÉTABLI : le loadout = i43 + i44 (les deux armes portées) ; l'arme en main = celle des
deux que i42 désigne. La corrélation n'est pas statistique mais STRUCTURELLE — i43 n'apparaît
jamais sans i42, et les deux emplacements ne portent jamais la même arme.

CE QUI NE L'EST PAS ENCORE : quel bit de i42 sélectionne, et dans quel sens. Test à deux issues,
tranchable en confrontant aux événements de tir, qui portent déjà l'arme utilisée. À faire.

RÉSERVE : i45 et i46 existent (9 et 0 lectures solitaires) mais ne co-occurrent JAMAIS avec i43.
Ce ne sont donc pas des emplacements supplémentaires du même porteur — leur nature reste ouverte.

## [2026-07-27] Handoff inventaire — verite terrain Theater, horloge disculpee

**Statut** : Complete (handoff, pas de nouveau decodage)

**Contexte** : l'utilisateur a releve en Theater, pour les 8 joueurs, les grenades, la capacite
et son compteur, l'arme en main et les munitions. Il a precise que son releve est fait a 00:25
(debut REEL du match) et non a 00:03 (debut du film) comme je l'avais demande.

**Decision technique** : consigner ce releve dans un document dedie plutot que dans le journal —
c'est la seule source non circulaire pour nommer les capacites (les libelles n'existent pas dans
les fichiers du jeu en build release : 1 chaine lisible sur 11 tags eqip, 0 libelle sur 238
tables uslg).

**Resultat observe, et correction d'une conclusion fausse** : j'avais d'abord conclu que
l'horloge du decodeur etait fausse, en constatant un premier tir a 66,0 s alors que
l'utilisateur annonce les premieres morts vers 30-40 s. Verification faite, le FIL DES MORTS du
POC produit sa premiere entree a **31,6 s** — dans la fenetre annoncee. Ce calque est decode par
un chemin independant (paquets de type 3). L'horloge est donc BONNE et le film est le bon
(deja etabli par ailleurs : 59 signatures sur 60).

Le defaut reel est isole et beaucoup plus etroit : le calque des TIRS ne produit rien avant
66,0 s alors que des morts sont journalisees des 31,6 s. Il manque 34 secondes de tirs en debut
de film — un trou de rappel dans la detection, pas un decalage temporel. Meme famille de
symptome que le rappel de 23,7 % du gabarit rigide.

**Conclusion / prochaine etape** : la confrontation avec la verite terrain est possible
immediatement, a l'image 250 (= 00:25). Ma tentative de fin de session comparait mes etats a
l'image 34 (avant le debut du match) au releve fait a 00:25 : elle ne mesurait rien, ses
conclusions sont ecartees. Ordre de reprise ecrit dans .ai/HANDOFF_INVENTAIRE_2026-07-27.md.

## [2026-07-27] Loadout craque — et la cause racine des contradictions : le POC melangeait DEUX FILMS

**Statut** : Complete

**Decision technique** : confronter les composants d'inventaire a la verite terrain relevee en
Theater par l'utilisateur, avec refutation adverse systematique (3 avocats du diable par
conclusion, dont un charge de verifier l'instant de lecture).

**Resultat majeur, non anticipe** : le document de rejeu melange deux films. Ses blocs tracks,
loadouts et roster viennent de 000d5950 (8 mars) — c'est ce que le POC declare lui-meme dans son
champ match, jamais verifie. Son bloc inv vient de 9e8fb31b (24 juillet), le film de la capture
Cheat Engine. La verite terrain decrit 000d5950 (armes en main 8/8 contre 1/8). Toute
confrontation faite jusqu'ici comparait donc DEUX MATCHS DIFFERENTS, rosters disjoints a 7
joueurs sur 8. Aucune des trois explications avancees successivement (horloge fausse, dotation de
pre-match, grammaire erronee) n'etait la bonne.

**Quatre tables etablies malgre cela** :
- grenades : compteur 0=Fragmentation 1=Plasma 2=Dynamo 3=Spike, par appariement des lancers aux
  decrements unitaires, 35/35, interne a 9e8fb31b, sans verite terrain. Nuls a zero sur 200 000.
- capacites : index 3=mur portatif 4=grappin 5=propulseur 6=capteur, lu dans le record de biped
  des images-cles de 000d5950. Les deux triplets de la verite terrain sortent groupes.
- selecteur i42 : sel=0 emplacement 0, sel=1 emplacement 1, sel=2 aucune arme. Sens etabli par
  oracle interne (l'emplacement dont les munitions bougent), 94,7 % et 95,4 %.
- munitions : i30/i33 chargeur (10 bits [0][8][1]), i31/i34 reserve (11 bits), table de 16 armes
  reproduisant la verite terrain 8/8, reserve multiple entier du chargeur 16/16.

**Gain de methode** : la localisation des lectures devient positionnelle
(offset = paquet.Start + 8*floor(curseur/64) + 8), 98,4 % de rappel, ZERO faux positif mesure sur
650 641 positions. Le filtre d'entropie devient inutile.

**Conclusion / prochaine etape** : recette consignee dans .ai/RECETTE_LOADOUT_2026-07-27.md.
Regle nouvelle : tout document de rejeu doit porter l'identifiant du film de CHAQUE bloc, et
aucune confrontation ne se fait sans verifier que les deux cotes portent le meme. Le POC est a
reconstruire sur un seul film.

## [2026-07-27] Jauge d'energie craquee — le bit de tete d'i30/i33 est un aiguillage semantique

**Statut** : Complete

**Origine** : remarque de l'utilisateur, non sollicitee — « le Ravageur a une arme a batterie, tout
comme le pistolet a plasma, qui se chargent tous les deux ». Le chantier venait de laisser 1363
lectures sur 2813 « non decodees », en concluant sur la foi de 2 lectures qu'elles ne concernaient
aucune arme a chargeur.

**Decision technique** : recroiser CHAQUE lecture d'i30/i33 avec l'arme de l'emplacement
correspondant (i43 gouverne i30, i44 gouverne i33), sur toutes les entites du film et non sur les
seuls slots initiaux, puis tabuler le bit de tete par arme.

**Resultat** : quatre armes se detachent a 99,8-100 % de prefixe 1 — Ravager (597 lectures),
Plasma Pistol (423), Pulse Carbine (112), Stalker Rifle (38). Ce sont exactement les armes a
charge. Toutes les autres sont majoritairement en prefixe 0. Le bit de tete n'est donc pas un bit
de cadrage mais un AIGUILLAGE : le meme champ porte le chargeur ou la jauge d'energie selon l'arme.

Cela corrige l'affirmation du chantier : ces quatre armes emettent bien des composants, dans
l'autre branche. L'agent n'en avait vu que 2 la ou il y en a 597.

**Deuxieme correction, meme origine** : le chantier ecrivait « palette de EXACTEMENT 4 valeurs »
pour les capacites. L'utilisateur a fait remarquer qu'il en existe plus. Verifie : onze capacites
distinctes dans static/abilities-assets/halo_infinite/. La mesure disait seulement « 4 valeurs
DANS CE MATCH ». Les index mesures (3,4,5,6) sont consecutifs, donc il en existe en dessous et
au-dessus. L'ordre n'est pas alphabetique (ThreatSensor precede Thruster alphabetiquement, on
mesure l'inverse).

**Risque nouveau, a traiter avant industrialisation** : le ThreatSeeker est une variante de mode
Infection introduite APRES la sortie du jeu (information utilisateur). Une enumeration qui grandit
au fil des mises a jour n'a pas de raison d'etre stable : un index lu dans un film ancien peut ne
pas designer la meme capacite qu'aujourd'hui. La table des capacites n'est validee que sur un film
de juillet 2026.

**Reste ouvert** : la valeur portee par la branche prefixe 1 (pourcentage ? fraction ?) n'est pas
decodee, seul l'aiguillage l'est. Et le marteau et l'epee n'emettent NI prefixe 0 NI prefixe 1 :
leur jauge, qui baisse pourtant a chaque coup porte, est ailleurs.

**Lecon de methode** : deux affirmations trop larges ont ete produites le meme jour par des agents
(« aucun composant de munitions », « exactement 4 valeurs »), toutes deux corrigees par une
remarque de l'utilisateur portant sur le jeu lui-meme. Une mesure faite sur un seul match ne borne
pas le jeu. Formuler desormais « N valeurs dans ce match » et jamais « la palette compte N ».

## [2026-07-27] Correction — la stabilite de l'enumeration des capacites n'est pas un risque

**Statut** : Complete (correction de la fiche du jour)

**Ce que j'avais ecrit** : « une enumeration qui grandit au fil des mises a jour n'a aucune raison
d'etre stable ; un index lu dans un film ancien peut ne pas designer la meme capacite ».

**Objection de l'utilisateur** : ajouter une valeur a une enumeration ne deplace pas les autres,
comme ajouter un produit a un catalogue.

**Argument supplementaire qui la renforce** : le jeu lui-meme depend de cette stabilite. Theater
doit rejouer un film enregistre avant une mise a jour ; une renumerotation ferait afficher la
mauvaise capacite sur tous les rejeux anterieurs. La compatibilite du format de rejeu impose donc
l'append-only.

**Correction retenue** : la limite reelle de la table n'est pas l'instabilite mais
l'INCOMPLETUDE — quatre index sur onze capacites connues. Ce n'est pas un risque de derive, c'est
un trou de couverture. Controle peu couteux a garder : un meme index designant deux capacites sur
deux films d'epoques differentes refuterait l'append-only.

**Lecon** : j'ai transforme une incertitude (« je n'ai pas verifie ») en risque (« ca peut
deriver »), ce qui n'est pas la meme chose et conduit a sur-dimensionner la parade. Qualifier
l'incertitude avant de la traiter comme un danger.

## [2026-07-27] Le binaire livre la recette du loadout — et corrige trois erreurs publiees

**Statut** : Complete

**Decision technique** : demander au binaire non pas la GRAMMAIRE (deja mesuree empiriquement)
mais les CONSOMMATEURS des valeurs — ce qui relit un champ porte son identite. Question posee par
l'utilisateur : « avec Ghidra on a rien qui nous donne la recette de comment lire le loadout ? ».

**Resultat principal — la carte memoire est lue, instruction par instruction** :
tampon = *(void**)(enregistrement + 0x10). Emplacements d'arme a la base 0x7F0, pas 0x90, quatre
entrees. Fermeture arithmetique independante : 0x7F0 + 4*0x90 = 0xA30, exactement le decalage
d'i47 lu ailleurs. Champs : identite +0x00, chargeur u16 +0x7E, reserve u16 +0x80, drapeaux +0x82,
jauge de charge f32 +0x84, surchauffe f32 +0x88.

**Deux points ouverts fermes** :
- i30/i33 est une UNION a deux branches, pas un champ a bit de cadrage. b0=R(1) puis chargeur
  R(8) ; b1=R(1) puis jauge dequant(R(12), 0.0f, 1.0f). L'invariant « bit0=0 implique bit9=1 »
  mesure hier sur 1450/1450 EST cette structure.
- La jauge d'energie des armes a charge n'est ni un pourcentage ni un compte d'unites : c'est une
  FRACTION dans [0,1] sur 12 bits, 4096 niveaux. Question ouverte hier, close aujourd'hui.

**Confirmation croisee des grenades** : la table grenade_types (base 0x1443E2AB0) donne
id1=frag, id2=plasma, id3=lightning (Dynamo), id4=spike, l'ordre d'i22 etant typeId-1. IDENTIQUE
a la table obtenue empiriquement par appariement des lancers (35/35, voie sans aucun rapport avec
le binaire). Deux chaines totalement independantes, 4 rangs sur 4. La question des grenades est
close. Piege releve : l'enumeration d'equipement (FUN_140157770) INTERVERTIT Dynamo et Plasma
(FragGrenade=9, DynamoGrenade=10, PlasmaGrenade=11) — ne jamais s'en servir pour i22.

**Capacites : le mecanisme est lu, le nommage est ferme par la mesure**. 0xA34 = compteur de
rotation (FUN_140A225CC fait (x+1)%7 par division magique), PAS une identite. 0xA35 = l'identite,
et c'est un RANG DE PALETTE resolu a l'execution par parcours du bloc de tags sofd. La palette vit
dans les fichiers de tags du jeu, pas dans l'executable : inutile de continuer a chercher
l'enumeration dans Ghidra. Ce n'est pas un abandon, c'est une mesure.

**Compteur d'utilisations trouve** : 7 bits par emplacement arme en 0x12EA+s, 0x7F = plein par
defaut. Le consommateur montre deux encodages, dont un discret ou le quartet de poids fort porte
les charges entieres. Le « 5 utilisations » de la verite terrain EST ce quartet.

**TROIS ERREURS PUBLIEES, corrigees** :
a) La premisse de tout le chantier — « les libelles n'existent pas en build release » — etait
   FAUSSE deux fois. Les noms sont dans l'EXE (vtable+0x08) ET dans le film lui-meme (registre ECS
   du chunk_00), ce dernier deja exploite par le depot depuis le 2026-07-01 via
   internal/analysis/filmdec/registry.go. Une partie du travail a re-derive une table existante.
b) i57 est a 0x12E4, pas 0x12E7 (defaut factuel publie).
c) Vie et bouclier n'etaient pas « non tranches » : i4 = object-body-vitality, i5 =
   object-shield-vitality, deja decodes dans vitality.go. Une passe anterieure avait prononce
   « REFUTE » sur la bonne reponse, en ecartant un champ sur une intuition semantique alors que
   son nom etait a un fichier de distance.

**Aveu de methode** : le controle interne annonce comme passe dans le premier chantier (« les
trois grappins partagent la meme valeur ») n'a tourne dans AUCUNE des quatre passes, et pour les
capacites il ne POUVAIT pas tourner — verite terrain sur 000d5950, mesures d'i48 sur 9e8fb31b.
Les predictions chiffrees qui en decoulaient auraient echoue quelle que soit la justesse de la
grammaire. Un successeur qui les suivrait jetterait une grammaire correcte.

**Prochaine mesure, la plus rentable** : la table des capacites du depot lit 3 bits ; le binaire
dit 6 bits precedes d'une porte. Elargir la fenetre a la meme position et verifier que les trois
bits ajoutes sont nuls et la porte a 0. Si cela passe, la table du depot EST la table des rangs
sofd, i48 devient nommable sur le film de la verite terrain, et le probleme des deux films est
contourne sans nouvelle capture.

**Conclusion** : recette consignee dans .ai/RECETTE_LOADOUT_2026-07-27.md, sections 7 a 12.

## [2026-07-27] Methode de retro-ingenierie consignee — sept regles et six facons de se tromper

**Statut** : Complete

**Origine** : demande explicite de l'utilisateur — « documente et journalise bien ca, surtout la
methode ». Les tables produites valent pour deux films ; la methode se transporte.

**Decision technique** : ecrire .ai/METHODE_RETRO_INGENIERIE_FILM.md en adossant CHAQUE regle a
une erreur reellement commise dans ce chantier, avec le controle bon marche qui l'aurait evitee.
Un document de methode fait de principes generaux ne change pas le comportement ; un document ou
chaque regle porte la trace de sa violation, si.

**Ce qui produit des resultats — les sept regles retenues** :
1. Suivre le CONSOMMATEUR, pas le producteur. La grammaire est chez celui qui ecrit le champ,
   l'identite chez celui qui le relit. Deux jours de correlation ont donne du bruit (53 % contre
   25 % de hasard) ; la table grenade_types a repondu en une passe.
2. Deux chaines independantes valent une preuve, une seule vaut une hypothese. Test
   d'independance : si l'une echoue, l'autre tombe-t-elle ?
3. Un oracle interne bat une verite terrain. Le sens du selecteur a ete etabli par « quel
   emplacement voit ses munitions bouger » — seule conclusion a avoir survecu intacte a la
   decouverte des deux films.
4. Mesurer les faux positifs sur le flux reel, jamais les calculer (un calcul uniforme s'est
   trompe d'un facteur 20 000).
5. La fermeture arithmetique comme verification gratuite : 0x7F0 + 4*0x90 = 0xA30, trois mesures
   independantes qui se ferment sans ajustement.
6. Le desassemblage fait foi, pas le decompile — Ghidra supprime silencieusement des blocs.
7. Attaquer par lentilles distinctes. Sur les grenades, la lentille « hasard » n'a rien casse et
   la lentille « controle interne » a tout casse.

**Les six facons de se tromper, toutes observees dans ce chantier** :
A. Batir sur une premisse jamais retestee — « les libelles n'existent pas » etait faux deux fois,
   et la table existait deja dans le depot depuis le 2026-07-01. Controle : chercher dans le depot
   avant d'attaquer un probleme « impossible ». Une premisse NEGATIVE se re-teste a chaque reprise.
B. Annoncer un controle sans l'executer. Controle : exiger la sortie chiffree ligne par ligne ;
   une formulation qualitative est le symptome d'un controle non execute.
C. Generaliser depuis un seul echantillon — « la palette compte exactement 4 valeurs » (il y en a
   au moins 11), « ces armes n'ont aucun composant » (2 lectures vues, 597 existantes). Regle
   d'ecriture : « N valeurs observees DANS CE MATCH », jamais « la palette compte N ».
D. Ne pas verifier que deux sources parlent du meme sujet. Signal d'alerte a retenir : quand TROIS
   explications successives echouent sur la meme contradiction, ce n'est pas l'explication qui
   manque, c'est une premisse partagee par les trois qui est fausse.
E. Resoudre une contradiction par une echappatoire non testee. Une contradiction ouvre au moins
   deux branches : les enumerer avant d'en choisir une.
F. Confondre incertitude et risque. « Je n'ai pas verifie » n'est pas « ca peut deriver ». Une
   incertitude sans mecanisme de defaillance identifie est une verification a planifier, pas un
   danger a parer.

**Recettes techniques consignees** : nommage d'un composant (vtable+0x08, ou registre ECS du
chunk_00 deja implemente dans registry.go), localisation positionnelle d'une lecture
(paquet.Start + 8*floor(curseur/64) + 8, 98,4 % de rappel, 0 faux positif sur 650 641 positions),
position exacte et largeur d'un composant, lecture d'une structure depuis un enregistrement,
reconnaissance d'une union dans une grammaire.

**Conclusion** : protocole en sept lignes en fin de document, plus une regle d'ecriture qui vaut
les six autres — ne jamais ecrire comme propriete du jeu ce qui n'a ete mesure que sur un match.

## [2026-07-27] Vehicules : l'archetype 40 est identifie, l'arme du vehicule ne passe PAS par le chemin des armes de joueur

**Statut** : Complete (reconnaissance ; rien de decode encore)

**Origine** : question de l'utilisateur — un kill a la Gungoose sur Launch Site, un Ghost sur High
Ground, le chunk_00 ne devrait-il pas les mentionner ?

**Application directe de la regle A de la methode** (chercher dans le depot avant d'attaquer) :
le depot possede deja tmp_archlist, tmp_archdump et tmp_archcomps. Reponse obtenue en trois
commandes, sans rien ecrire de neuf.

**Resultat 1 — le registre est un schema statique.** Il est bit-a-bit identique d'un film a
l'autre (empreinte FNV des 1067 entrees = a413610cd08e4355 sur Cliffhanger comme sur Catalyst). Il
ne dira donc JAMAIS « ce match avait une Gungoose ». Savoir quels vehicules etaient dans un match
se lit dans le FLUX, pas dans le registre.

**Resultat 2 — l'archetype 40 est le vehicule**, 48 composants. Il partage la charpente du biped :
position i0, velocite i1, orientation i2, vie i4, bouclier i5, dead-state i11, plus les composants
unit-*. Consequence immediate et sous-estimee : **la position, la vitesse, l'orientation, la vie
et le bouclier d'un vehicule se lisent DEJA avec le code existant**. Un vehicule peut etre dessine
sur la carte du rejeu sans aucun decodage supplementaire.
Partie propre : vehicle-auto-turret-triggers i30, -aiming-vector i31, transformed-or-desired-open
i32, vehicle-type-state i33, vehicle-type-physics i34, auto-turret-target i35, sentry-state i36,
emp-timer i37.

**Resultat 3, negatif et utile — weapon-state-type-info n'existe QUE dans l'archetype 35**, en
quatre exemplaires (i43 a i46). L'archetype 40 n'en porte aucun. L'identifiant d'arme d'une
Gungoose n'est donc PAS lisible par la recette craquee pour les armes de joueur. Porte fermee par
la mesure, ce qui evite d'y perdre du temps.

**ANOMALIE A TRANCHER** : RECETTE_DECODAGE_FILM_CHUNKS.md annonce 174 archetypes valides,
tmp_archlist en compte 118 sur le meme registre. L'un des deux chiffres est faux. Si c'est 174, il
existe 56 archetypes jamais listes et l'un d'eux peut porter ce qu'on cherche. A trancher par
lecture directe du registre, pas par arbitrage entre deux documents.

**Prochaines mesures, dans l'ordre** : (1) compter les entites d'archetype 40 dans un film et lire
leur vehicle-type-state, ce qui donne la liste des vehicules du match ; (2) trouver ce qui CONSOMME
vehicle-type-state — regle n1 de la methode, l'identite est chez celui qui relit le champ.

**Verite terrain gratuite** : la Gungoose de Launch Site et le Ghost de High Ground, deux
vehicules nommes sur deux films identifies, sans cout de releve supplementaire.

**Conclusion** : consigne dans .ai/VEHICULES_ARCHETYPE_40.md. Chantier bien place — terrain balise,
moitie des composants deja decodables, methode ecrite, verite terrain fournie. Ce qui manque n'est
pas de la comprehension mais du temps de machine.

## [2026-07-27] Etat de l'art du chantier voisin integre — et la Gungoose etait deja resolue

**Statut** : Complete

**Origine** : demande de l'utilisateur — mettre les recherches de l'autre worktree dans l'etat de
l'art et referencer les docs.

**Ce que la lecture a revele, et c'est une erreur de ma part** : la question des armes de vehicule
que je venais de declarer ouverte est RESOLUE dans filmdec-killweapon depuis des semaines, et la
Gungoose est leur cas d'ancrage nomme. Regle R-VEHICULE : un weap est un armement de vehicule si
un vehi le reference ou s'il pend a vcdd -> sofd -> sofa -> uwfa -> weap. 62 weap sur 194, dont
16 par la chaine vcdd, et l'ancre Gungoose passe EXCLUSIVEMENT par cette chaine. Disjonction
totale d'avec le catalogue des armes de joueur : 0/62, attendu 10,2, p = 1,1e-06. La distinction
tourelle/fixe est etablie par deux signaux independants ; le Ghost est « fixe ».

J'avais ecrit VEHICULES_ARCHETYPE_40.md comme si le nommage etait a faire. Document corrige.
C'est exactement leur erreur E8 (« une reponse deja ecrite dans le journal, re-cherchee faute de
l'avoir grepe ») que je viens de reproduire, alors qu'elle est consignee dans leur fiche.

**Convergence remarquable sur les grenades** : leur table (0 FRAG, 1 PLASMA, 2 DYNAMO, 3 POINTES)
est IDENTIQUE a la mienne, etablie par une troisieme voie encore differente — l'adressage de la
table tagref (periode d'entree 0xd0, block de 832 octets = 4 x 0xd0). Trois chaines totalement
independantes, une seule table. La question etait deja close ; elle l'est trois fois.

**LA CONNEXION QUE NI EUX NI MOI N'AVIONS FAITE** : leur chaine d'armement de vehicule traverse le
groupe de tags sofd. De mon cote, l'identite de capacite spartan se resout par un parcours du bloc
sofd (FUN_1407E7648). Deux chantiers ont bute sur la MEME structure sans le savoir. Si sofd est la
table d'equipement et d'armement d'un match, le rang de capacite et l'armement de vehicule se
lisent au meme endroit — et le nommage des capacites, que j'avais declare ferme cote executable,
pourrait s'ouvrir par le chemin de tags qu'ils parcourent deja. C'est la piste la plus prometteuse
pour combler ma table (4 index sur 11 capacites connues).

**Trois de leurs regles de methode adoptees**, absentes de la mienne :
- une mesure de concordance ne peut pas servir a la fois de SCORE et de FILTRE ;
- une liste de coupables lue sur une marche morte mesure OU on meurt, pas ce qui casse ;
- avant d'ouvrir une piste, GREPER le journal (leur E8, que je viens de refaire).

Et leur meta-patron le plus couteux me concerne directement : « deux referentiels qui decrivent le
meme objet, jamais mis en regard » — chez eux une capture nommant des decalages de structure et
une grammaire parlant de positions de bit, six semaines perdues. Chez moi, une capture decrivant
un film et un POC en decrivant un autre, deux jours perdus. Meme patron.

**Leur protocole de correction de la verite terrain**, a reprendre tel quel : le decodeur ne
corrige le terrain que si le support est independant, deja concordant ailleurs, et la contradiction
soulevee par nous. Sinon c'est le decodeur qui a tort. « Si la verite terrain cesse de pouvoir
falsifier, plus rien ne le peut. »

**Conclusion** : index de renvoi ecrit dans .ai/ETAT_DE_L_ART_CHANTIER_VOISIN.md, avec la liste de
leurs dix documents et quand les lire. Section finale « ce qu'ils n'ont pas et que j'ai » pour que
l'echange aille dans les deux sens. Regle nouvelle : avant d'ouvrir une piste sur le format de
film, chercher dans cet index.

## [2026-07-27] LE BLOC INV EST REFAIT SUR LE BON FILM — et le controle terrain a enfin tourne

**Statut** : Complete

**Origine** : le POC melangeait deux films. Son bloc `inv` venait de 9e8fb31b (capture Cheat
Engine) alors que tracks / loadouts / roster viennent de 000d5950, le film de la verite terrain.

**Decision technique** : refaire l inventaire par ANCRAGE DANS LES RECORDS DE BIPED des
images-cles type 2, entierement hors ligne, sans capture. Quatre regles d ancrage, toutes
enoncees AVANT toute confrontation au terrain :
- R1 capacite : ancre 28 bits 0x8CAC57A + motif 20 bits, index sur 3 bits, ancre exigee UNIQUE.
- R2 grenades : premier motif i22 (R(3)=4 puis 4xR(8)) situe APRES l ancre capacite.
- R3 armes : familles connues dans l ordre des bits (voie deja validee 150 records).
- R4 munitions : le bloc i30..i42 se termine EXACTEMENT sur le bit de porte d i43, a E0-1 ; on
  n accepte qu un parse qui ATTERRIT au bit pres. Critere de LARGEUR, pas de contenu.

**Le format s est referme, et c est la vraie decouverte** : les quatre emplacements d arme de la
carte memoire (0x7F0 + s*0x90) sont serialises d affilee — pour chaque emplacement, union
chargeur/jauge, puis reserve R(11), puis 9 bits (2 drapeaux + 7 surchauffe) — puis i42, puis la
porte d i43. Verifie au bit pres sur le slot 514 : 48+10=58, +11=69, +9=78 = debut exact de la
2e paire de munitions, lue 8/8 Heatwave. Les emplacements 2 et 3 sont vides (44 bits nuls) : c est
cette contrainte STRUCTURELLE, plus l exclusion de la largeur 22 (§7 n enumere que 2/10/14) et
l invariant « union vide implique reserve nulle », qui ramenent 150 parses ambigus a 99 uniques.

**LE CONTROLE TERRAIN, image 163, huit joueurs, chiffres bruts** :
- grenade portee : 8/8
- capacite : 8/8
- arme du releve presente dans la paire decodee : 8/8
- chargeur/reserve : 7/7 (le 8e est le marteau, correctement « aucune »)
- emplacement degaine : **0/8 — ECHEC MESURE**
- groupes : les trois grappins partagent 4, les trois propulseurs 5, le capteur 6, le mur 3 ;
  quatre index mutuellement distincts.

**L echec est explique, pas masque** : a l image 163 (16,4 s) i42 vaut 2 = « aucune arme
degainee » pour les huit — le match n a pas commence, les armes sont rangees. Sur tout le film le
selecteur vaut 0 dans 70 cas, 1 dans 70, 2 dans 10 seulement : le champ est bien vivant. Temoin
interne (aucune verite terrain) : la part de chargeurs ENTAMES est 31,9 % sur l emplacement
designe contre 14,9 % sur l autre en regime sel=0, et 20,4 % contre 12,5 % en regime sel=1. Le
sens va dans la direction annoncee, avec un rapport de 2,1 et 1,6 — coherent, pas concluant.

**Ce que je n ai PAS reussi, et je le dis** : le COMPTEUR D UTILISATIONS de capacite n est pas
localise. 6 ancres x 6001 offsets = 36 006 positions testees, ZERO ne reproduit les huit valeurs
du releve (5,5,5,4,5,1,5,5). Reserve supplementaire : dans ce releve les utilisations sont une
fonction de la capacite (grappin et propulseur 5, capteur 4, mur 1), donc meme un succes n aurait
apporte que peu d information independante.

**Reproduction independante de la recette** : persistance de l index de capacite par vie —
32 vies a index constant sur au moins deux images-cles contre 5 changeantes. Chiffres IDENTIQUES
a ceux de RECETTE_LOADOUT_2026-07-27 §2, obtenus ici par un code ecrit sans les relire.

**Assistances** : killsource execute sur 000d5950 depuis une COPIE de travail du worktree voisin
placee dans le scratchpad (le voisin n a jamais ete ecrit). 93 morts, porte ligne-par-ligne
AUTORISEE, marge de bijection 36. Assistant : 17 nommes, 76 absents MESURES, 0 inconnu. Part de
degats du tueur mesuree 93/93, celle de l assistant 17/93. Un main neuf a ete necessaire : la
sous-commande `json` de cmd/killsource n emet ni l assistant ni les parts de degats.

**Conclusion** : deux blocs livres en JSON dans le scratchpad, chacun portant son film. Le
cablage du POC appartient a un autre agent. Prochaine mesure la plus rentable : le compteur
d utilisations, et la reduction des 51 parses ambigus du bloc munitions.


## [2026-07-27] Rejeu 2D - l'assistant dans le fil des morts (POC replay_demo.html)

**Statut** : Complete pour l'affichage ; un point de mise en page reste ouvert et il est
attribue au chantier voisin (colonnes d'equipe), pas a celui-ci.

**Decision technique** : le fil des eliminations vient de la BASE (killer_victim_pairs,
recalage historique -3,7 s) tandis que l'assistance vient du FILM (killsource.Decode). Deux
horloges : le recollage a ete MESURE, jamais suppose. Le decalage est estime sur les SEULS
couples (tueur, victime) uniques des deux cotes - 5 couples - donc sans se servir du temps
pour choisir : mediane 3,684 s, dispersion des ecarts au median 12 ms au pire. L'appariement
complet qui en decoule est un pour un, glouton sur l'ecart croissant, couple identique
obligatoire.

**Resultats mesures** :
- 93 morts de feed appariees sur 93 ; 93 morts killsource consommees sur 93 ; 0 orpheline.
- ecart residuel : mediane 21 ms, p90 52 ms, maximum 65 ms.
- ambiguite : 0 entree dont le deuxieme candidat soit a moins d'une seconde.
- etats injectes : 17 assistants nommes, 76 « sans assistant » MESURES, 0 inconnu.
- les 2 entrees de feed sans victime (medailles orphelines) ne recoivent aucun etat : ce ne
  sont pas des morts.

**Le point qui compte** : « inconnu » et « pas d'assistant » sont deux etats DISTINCTS dans la
donnee (`as.e` vaut nomme / aucun / inconnu) et dans le rendu (« + Nom », « - sans assistant »,
« ? assistant inconnu » souligne en pointille - le pointille est deja, dans cette page, la
marque du non-mesure). Le repli du code est « inconnu » : une mort DEPOURVUE de champ s'affiche
inconnue, jamais « pas d'assistant ». Verifie en executant la fonction du POC sur une entree
fabriquee sans donnee.

**Controle non demande qui pouvait echouer** : les 17 assistants nommes appartiennent tous a
l'equipe du tueur (17/17), croisement roster x killsource qui n'entre dans aucune etape de
l'appariement.

**Parts de degats** : affichees telles que lues, chacune collee a son proprietaire (celle de
l'assistant suit son nom, celle du tueur est nommee). Tueur mesure 93/93, assistant 17/17
lorsqu'il existe. Reserve portee dans l'infobulle ET dans la note de bas de page : elles ne sont
pas bornees a 100 % (un couple mesure ici somme a 129, un tueur solo a 119) et le chemin de
donnees vers KillerPercentageDamageDone / AssistantPercentageDamageDone n'est PAS demontre.

**Ce qui n'a pas ete fait, et pourquoi** : le panneau « Eliminations » est aujourd'hui ecrase a
0 px de hauteur. Ce n'est pas l'assistance : la ligne d'assistance vit DANS la liste defilante.
Mesure a l'identique sur trois versions du meme fichier, meme fenetre : colonnes d'equipe
317,5 px -> fil 72,5 px (version avant le bloc inv) ; 385 px -> 5 px (inv v1) ; 523 px -> 0 px
(inv enrichi du 27/07). La bande laterale fait 452 px, imposee par la carte a dessein
(commentaire CSS du POC : le fil ne doit jamais etirer la carte). Rendre le fil visible demande
de raccourcir les colonnes d'equipe : c'est le perimetre du chantier inventaire, pas celui-ci,
et il etait en cours d'ecriture pendant cette intervention. Aucun debordement horizontal en
revanche (scrollWidth = clientWidth = 368 px).

**Conclusion / prochaine etape** : l'assistant est cable, mesure et trace (bloc `assistMeta`
portant film, source, recalage, denominateurs et reserve). Prochaine etape utile : redonner de
la hauteur au fil en compactant la ligne d'inventaire des colonnes d'equipe.

## [2026-07-27] POC rebati sur UN SEUL film — le controle terrain passe enfin, et la jauge de melee est resolue

**Statut** : Complete (deux defauts graves traites, quatre moyens ouverts)

**Decision technique** : refaire le bloc d'inventaire du POC sur 000d5950 seul (le film de la
verite terrain), cabler les tables craquees, et brancher les assistances de killsource. Chaque
bloc du document porte desormais l'identifiant de son film.

**LE CONTROLE TERRAIN A ENFIN TOURNE, ET IL PASSE.** Image-cle 163, meme horloge et meme origine
que les pistes du POC :
  grenade portee ........................ 8 / 8
  capacite (nom) ........................ 8 / 8
  arme du releve presente dans la paire . 8 / 8
  chargeur / reserve .................... 7 / 7  (le 8e est le marteau : « aucune », jamais 0)
  emplacement degaine ................... 0 / 8  ECHEC MESURE (i42 vaut 2 a cette image)
  compteur d'utilisations ............... 0 / 8  CHAMP NON LOCALISE (36 006 positions testees)

LES TROIS GROUPES SORTENT GROUPES : les trois Dynamo au rang 2, les trois grappins a l'index 4,
les trois propulseurs a l'index 5, le capteur seul a 6, le mur seul a 3. Quatre index mutuellement
distincts. C'est le controle annonce comme passe le 2026-07-26 alors qu'il n'avait jamais tourne ;
il tourne maintenant, sur un seul film des deux cotes.

**LA JAUGE DU MARTEAU ET DE L'EPEE EST RESOLUE — et ma fiche disait le contraire.** J'avais ecrit
« le marteau et l'epee n'apparaissent dans aucune des deux branches, leur charge est ailleurs ».
Faux : mesure refaite sur 000d5950 en croisant chaque lecture avec l'arme de l'emplacement,
37 jauges — Gravity Hammer 17, Stalker Rifle 9, Energy Sword 5, Ravager 3, Pulse Carbine 1,
Sentinel Beam 1, M41 SPNKr 1. Le marteau et l'epee totalisent 22 des 37. Ils utilisent la branche
prefixe 1 comme les armes a batterie.

L'erreur venait d'avoir mesure sur 9e8fb31b, ou ces armes sont rares et ou le suivi d'arme courante
decalait. L'utilisateur l'avait signale des la veille (« le marteau et les epees ont des reserves
d'energie, ca baisse a chaque coup porte ») ; je l'avais consigne comme point ouvert au lieu de le
tester. Le « charge 100 % » du releve Theater est donc une VALEUR DE JAUGE, pas une absence.
Reserve : la lecture unique sur M41 SPNKr est probablement un faux positif — une jauge sur une
lecture ne fait pas une categorie.

**DEFAUT GRAVE DE MISE EN PAGE, CORRIGE.** Le fil des eliminations etait ecrase a 0 px : .teams
etait en `flex: 0 0 auto` (incompressible) pendant que la carte du fil etait le seul element
flexible, donc elle absorbait seule toute la pression. Mesure de la degradation : 317 px de
colonnes -> 72 px de fil, puis 385 -> 5, puis 523 -> 0. Chaque ligne ajoutee a la fiche joueur
etait prise au fil, silencieusement. Correction : colonnes compressibles avec defilement propre,
plancher de 168 px sur le fil. Sans cette correction tout le livrable « assistances » etait
invisible.

**ASSISTANCES** : 93 morts, 17 assistants nommes, 76 « connu et absent » mesures, 0 inconnu. Les
trois etats restent distincts dans le JSON — Known=false n'est jamais rendu « pas d'assistant ».
Part de degats du tueur mesuree 93/93, de l'assistant 17/93. Les deux ne sont pas bornees a 100
(un couple somme a 129). Reserve portee : le chemin de donnees vers KillerPercentageDamageDone
n'est pas demontre.

**killsource N'EST PAS COMPILABLE depuis ce worktree**, et c'est mesure : les deux branches n'ont
AUCUN ancetre commun (killweapon est orpheline, 32 commits), et le paquet filmdec a diverge —
cinq symboles absents cote continuation. Route retenue : executer killsource depuis SON worktree
contre les chunks du depot principal, et n'echanger qu'un JSON.

**RESTE OUVERT** : quatre defauts moyens (denominateur flatteur sur la couverture munitions,
tuile annoncant 184 etats dont 33 vides, pointille servant a la fois « non mesure » et « mesure
absente », index de tracabilite deja perime), et l'emplacement degaine qui echoue a l'image 163
parce que i42 y vaut « aucune arme degainee » — a relire a une image posterieure au coup d'envoi.

**Lecon** : deux affirmations fausses de ma fiche ont ete corrigees en 24 h, toutes deux nees
d'une mesure faite sur UN film puis enoncee comme propriete du jeu. C'est exactement l'erreur C de
METHODE_RETRO_INGENIERIE_FILM.md, ecrite hier. La regle existait, elle n'a pas suffi.

## [2026-07-28] Deux jeux de donnees pour le POC : l arme de chaque mort, et les objets au sol

**Statut** : arme de kill COMPLETE ; objets au sol PARTIEL (le discriminant prescrit est non
evaluable, et c est mesure).

**Decision technique** : killsource n est toujours pas compilable depuis ce worktree ; il a ete
execute depuis la COPIE de travail deja presente dans le scratchpad (`kw_go`, verifiee
byte-identique au worktree voisin par `diff -rq` sur `killsource` ET `filmdec` : 0 difference).
Le worktree voisin n a recu aucune ecriture. Un main neuf (`cmd/armeskill`) a ete necessaire :
ni `cmd/killsource json` ni `cmd/assists` n emettent la CATEGORIE de degat.

### 1. Arme de chaque mort — `ARMES_DE_KILL_000d5950.json`
93 morts, porte ligne-par-ligne AUTORISEE, marge de bijection 36, sante NOMINAL. Par mort :
instant, tueur, victime, TAG BRUT + libelle, classe, categorie de degat, statut, assistant
(3 etats), parts de degats, divergence, provenance.
- armes nommees 87 / 93 ; repli « Autres » 6 ; 27 tags distincts ; statut SOUS_RESERVE 87.
- classes : ARME 87, MELEE 3, GRENADE 2, OBJET_EXPLOSIF 1.
- categories de degat : None 73, Headshot 8, AttachedDamage 7, HeadshotMultiplier 4,
  SilentMelee 1.
- assistant : 17 nommes, 76 « connu et absent » MESURES, 0 inconnu.
- QUATRE tags distincts sortent « Gravity Hammer » (877da252, 69fd30b9, ebcbab64, 45975ea4) :
  la raison de stocker le tag A COTE du libelle et jamais a sa place.

### 2. Objets au sol — `OBJETS_AU_SOL_000d5950.json`, et TROIS ROUTES REFUTEES
**Le discriminant prescrit (recurrence spatiale) est NON EVALUABLE : son entree n existe pas.**
La position des entites ti=42 / ti=37 n est pas decodable aujourd hui.
- Voie DELTA (detecteur d en-tete generique pilote par le jeu de slots ti=42) : 5 echantillons
  sur 178 slots declares, contre **1 006 sur un jeu de slots FANTOME de meme cardinalite**.
  Signal SOUS le bruit. Coherent avec la grammaire : un objet pose n emet aucun delta d i0,
  porte il est parente (i10).
- Voie KEYFRAME a offset fixe : **0 concordance a moins de 2 m sur 1 337 offsets x 137 records
  = 183 169 essais**. Temoin positif = les records ti=35, dont la position a l instant du
  keyframe est deja rendue par `ScanFilmBipedPositions` (oracle INTERNE). Le temoin echoue,
  donc rien sur ti=42 n aurait ete publiable. Cause : la largeur du default-state ET celle du
  masque sont variables.
- Voie CARTE (.mvar cliffhanger_ridgeline, deja parse par le depot) : 443 objets, mais la
  resolution deja faite (`forge_object_types.csv`) n attribue que **5 objets au groupe `weap`**.
  Les socles d armes d une arene stock ne sont pas dans la variante Forge.
- Secours semantique `equipment-creator` : INAPPLICABLE aux armes — le composant est i23 de
  ti=37 et il est **ABSENT des 21 composants de ti=42** (registre du film).
- Levee du blocage : porter le default-state de ti=42 (0x1407F0C68) et ti=37 (0x1407F105C).

**CE QUI EST QUAND MEME LIVRE** : 456 vies (178 ti=42, 278 ti=37) avec archetype, slot,
generation, apparition, derniere vue, nombre de keyframes ; et le TYPE quand il est resoluble —
**196 des 269 records ti=42 (72,9 %) portent un identifiant de famille d arme**, 169 vies
nommees. Aucun record ti=37 n en porte, ce qui est coherent (l equipement n est pas une arme).

**ORIGINE DE LA POPULATION ti=42, mesuree par trois voies concordantes** : ratio 1,914 vie par
mort (178 / 93) ; correlation de Pearson **0,821** entre nouvelles vies et morts sur 25 fenetres
de 20 s ; **67 paires de slots CONSECUTIFS** apparues au meme keyframe hors premier keyframe
(138 vies sur 178, 77,5 %). Un joueur qui meurt lache ses deux armes au meme instant et le
moteur leur donne deux slots consecutifs. Il ne reste pas la place d une population de carte
substantielle.

**PISTE MESUREE, NON CLOSE** : 7 vies ti=42 seulement persistent sur >= 3 keyframes, et ce sont
EXACTEMENT des vies sans libelle (71 records, 0 famille) alors que les 198 autres records en
portent 196. Controle interne execute pour eliminer l explication banale : les records
persistants sont **plus LARGES** (mediane 1 828 bits contre 1 259), pas plus courts — ils ont la
place et ne portent rien. Quatre d entre eux existent des le premier keyframe. Ces 7 entites
POURRAIENT etre les supports poses par la carte ; sans position ni libelle, rien ne le
distingue d un objet de decor. **Le fait que 7 tombe dans la fourchette 6-12 annoncee comme
temoin du discriminant spatial n est PAS ce temoin** (grappes de positions contre entites) :
les rapprocher fabriquerait une validation.

**Substitut essaye et refute** : la recurrence TEMPORELLE d une meme famille d arme suit l usage
de l arme dans le match, pas un cycle (Gravity Hammer 19 apparitions pour 22 morts au marteau,
Sentinel Beam 13 pour 11). Le cycle de reapparition reste NON MESURABLE.

**Conclusion / prochaine etape** : les deux JSON sont dans le scratchpad, chacun portant son
identifiant de film ; le cablage du POC appartient a un autre agent. La mesure la plus rentable
ensuite est le portage du default-state de ti=42 — elle debloque d un coup la position des
objets au sol, le discriminant spatial, et `equipment-creator` pour ti=37.

---

## [2026-07-28] POC — rendu des armes portees : ordre pilote par le selecteur, negatif, primaire, echange anime

**Statut** : Complete (item 20.1 de la liste Notion).

**Decision technique principale** — separer deux proprietes que le POC confondait. *Primaire /
secondaire* est une POSITION dans l enregistrement (emplacement 0 / 1) ; *en main* est le
SELECTEUR d emplacement du record d image-cle. Le POC designait la main par le DERNIER TIR, une
inference. Elle est remplacee par la lecture du selecteur : l arme degainee passe a GAUCHE, la
primaire est SOULIGNEE EN VERT a la place ou elle se trouve, et un echange entre les deux
emplacements est ANIME. Le dernier tir n est pas supprime : il reste dans l infobulle de la
vignette, comme second temoignage venu d une autre source (les evenements de tir).

**Motif, mesure sur le rendu lui-meme** (4 985 images redessinees, comptage des fiches
produites ; denominateur unique = les **29 279** fiches-joueur-par-image qui affichent
effectivement les deux armes, sur 39 880 possibles) :

| critere | fiches | part |
|---|---|---|
| main designee par le SELECTEUR | 20 069 | 68,5 % |
| main designee par le DERNIER TIR (ancien critere) | 4 085 | 14,0 % |
| les deux disponibles | 3 737 | dont 2 742 d accord et 995 de desaccord |
| primaire affichee A DROITE (ordre inverse) | 10 879 | 37,2 % |

Les deux sources ne coincident donc pas toujours, et le POC ne le cache pas. Ce qui fait
preferer le selecteur n est pas un arbitrage entre elles, c est qu il est LU la ou l autre est
DEVINE — et son sens est etabli par un oracle interne au film (l emplacement dont les munitions
bougent, 94,7 % / 95,4 %, recette du 2026-07-27 §3).

**Correction d une reserve mal formulee dans le POC**. Le document ecrivait que le selecteur
n est « pas concluant » parce que sa confrontation au releve Theater echoue 0/8. C est un test
**vide**, pas un desaccord : l unique image relevee (163) est une image ou les huit joueurs ont
encore les armes rangees, le match n ayant pas commence. La formulation a ete reecrite.

**Mesures d appui sur le bloc `inv` de 000d5950** : 184 etats, 150 portant les deux armes ;
selecteur lu 140 fois (70 sel=0, 70 sel=1), 10 « aucune arme degainee », 34 non lu ; **20
echanges** 0<->1 entre deux images-cles consecutives d un meme slot. Appariement `inv` /
`loadouts` : 150 cles (image, slot) communes, **150/150** memes noms au meme rang — d ou le
garde-fou `e.f === lo[0]` avant tout usage du selecteur.

**Points de mise en oeuvre** :
- L animation ne peut pas etre une animation CSS ordinaire : la carte d equipe est reconstruite
  par `innerHTML` a chaque image (~60 fois par seconde), donc elle redemarrerait sans jamais
  bouger. Elle recoit un `animation-delay` NEGATIF egal a son avancement, calcule sur l horloge
  du rejeu — elle reprend ou elle en etait a chaque reconstruction.
- Premiere version refutee par la mesure : chaque vignette translatee de sa PROPRE largeur
  donnait un echange de 59 px contre 10 px (largeurs de 20 a 56 px). Corrige en translatant
  chacune de la largeur de L AUTRE, precalculee une fois depuis le rapport naturel des images.
  Fermeture verifiee sur deux joueurs : boites 55 / 37 -> parcours 41 / 59, et 23 / 51 -> 55 / 27.
- `prefers-reduced-motion` ne coupait que les transitions ; les animations passaient au travers.
  La regle couvre maintenant aussi `animation-duration`, ce qui neutralise l echange ET
  l apparition de la grenade. Verifie : transformation a l etat final, aucun mouvement.
- Le lancer de grenade rend la grenade active : elle prend la premiere place et son nom
  s affiche. Le lancer est date et attribue (le film ecrit son auteur) ; que la grenade soit
  encore en main pendant la remanence de 1,4 s est une **convention d affichage**, ecrite comme
  telle dans l infobulle. 343 fiches-images concernees.

**Ce qui n a pas ete fait** : rien de la demande. Aucun decodage nouveau n a ete tente (mission
d affichage, interdiction de compiler du Go respectee — aucun `go build/run/test`). Le worktree
`filmdec-killweapon` n a recu aucune ecriture.

**Prochaine etape** : l ordre et la couleur reposent sur le selecteur ; la mesure qui le
fermerait est un relevé Theater fait sur une image OU LE MATCH A COMMENCE — l unique relevé
existant porte sur une image ou les huit joueurs ont les armes rangees, et ne peut donc rien
departager.

---

## [2026-07-28] Rejeu 2D — l arme du degat fatal a la place de la croix dans le fil des eliminations

**Statut** : Complete (POC du bac a sable uniquement ; aucun fichier du depot autre que ce journal).

**Decision technique**. Le symbole `†` entre le tueur et la victime ne portait aucune
information : une ligne du fil EST une mort. Il est remplace par l ARME DU DEGAT FATAL, issue du
decodage killsource du film 000d5950 (`ARMES_DE_KILL_000d5950.json`, produit par l agent
d extraction). Le fil vient de la BASE, l arme vient du FILM : deux horloges, donc un
appariement explicite sur la cle (horodatage killsource, tueur, victime) — **93 morts du fil sur
93 appariees, 0 cle dupliquee d aucun des deux cotes**, donc une bijection et aucun choix
arbitraire. L appariement est fait une fois a la construction du bloc de donnees, pas au rendu.

**Trois rendus, jamais confondus** (meme discipline que les trois etats de l assistance) :
visuel en negatif pour une arme nommee dont la table `wicons` a le dessin ; libelle encadre plein
pour une arme nommee sans visuel ; **pointille** pour une arme non nommee (le decodeur ne rend que
sa NATURE : melee, grenade, objet explosif), pour une mort sans arme, et pour une arme non lue.
Jamais de visuel par defaut : un visuel arbitraire affirmerait l arme qu on ignore. Le TAG BRUT
figure dans l infobulle des trois rendus — c est la quantite mesuree, le libelle n en est qu une
lecture, et il permettra de completer la table sans redecoder le film.

**Resultats mesures sur ce film** (jamais enonces comme propriete du jeu) : 87 morts sur 93
recoivent un visuel, 0 arme nommee sans visuel dans le depot, 6 morts sans libelle (repli
« Autres » : 3 melees, 2 grenades, 1 objet explosif), 2 entrees du fil sont des medailles seules
et n ont donc ni croix ni arme. Les branches « mort sans arme » et « arme non lue » sont ecrites
et rendues (verifiees en les forcant), mais **0 cas mesure sur ce film** — c est dit, pas suppose.
87 des 93 armes sont SOUS_RESERVE : la classe est sure, le nom propre ne l est pas (quatre tags
distincts rendent tous « Gravity Hammer » sur ce seul match) ; la reserve est en infobulle.

**Ce qui n a pas ete touche** : le liseré d equipe a gauche et le fond bleute des morts assistees,
qui portent deux autres informations. La table `wicons` (22 visuels) est reutilisee telle quelle,
aucun second jeu d images n a ete cree.

**Decouverte hors perimetre, NON traitee, et bloquante pour la lisibilite du fil**. Les cartes du
fil se font ECRASER par le conteneur : `.feed` est une colonne flex de hauteur contrainte et
`.feed .e` porte `overflow: hidden`, ce qui annule sa taille minimale automatique — les cartes
retrecissent donc sous leur contenu au lieu de faire defiler le fil. Mesure sur la colonne de
226 px : 2 entrees -> naturel 226 px (intact) ; 5 -> 230 px demandes, 226 rendus ; 7 -> 312
demandes, 226 rendus ; 11 -> 509 demandes, 226 rendus (cartes a 18-24 px) ; 95 -> 5 033 demandes,
1 232 rendus (cartes a 10 px). **Le defaut est ANTERIEUR a cette retouche** : controle execute en
remettant la croix a la place de l arme dans le DOM, les cartes restent a 10 px. Remede d une
seule declaration (`flex-shrink: 0` sur `.feed .e`), non applique : hors perimetre, et il change
la densite verticale d une page qui porte des semaines de travail visuel.

**Prochaine etape** : trancher ce point de mise en page, puis completer les 6 armes du repli
« Autres » — leurs tags bruts sont deja stockes dans le bloc de donnees, aucun redecodage requis.

---

## [2026-07-28] POC de rejeu — effets de tir adaptes a la famille d arme, et eclair oriente vers la victime

**Statut** : Complete (retouche du POC uniquement ; aucun code Go touche, aucune compilation)

**Demande** : un eclair de bouche dirige vers la victime, et un effet ADAPTE a l arme —
plasma, forerunner, roquette, marteau, epee, needler.

**Ce qui a rendu la chose possible, et qui n avait pas ete remarque** : les evenements de tir
du POC portent DEJA le libelle de l arme. `shots` est un tuple de 7 champs, et `sh[6]` est le
nom (`sh[5]` etant l identifiant filmshell) : **147 tirs sur 147 en portent un, 13 libelles
distincts sur ce film**. La mission n a donc rien eu a decoder — la donnee etait dans le bloc,
non tracee. C est exactement le cas de figure que `ETAT_DU_POC.md` decrit pour `spoly` :
« une donnee embarquee mais jamais tracee ».

**Decision principale — separer ce qui est mesure de ce qui est un parti pris**, et l ecrire
dans le fichier :
- MESURE : le libelle de l arme (tirs `sh[6]` 147/147 ; morts `feed[].w.n` 87/93, et pour les
  6 restantes la NATURE `w.cl` : 3 MELEE, 2 GRENADE, 1 OBJET_EXPLOSIF).
- PARTI PRIS : le regroupement en 7 familles et l aspect de chaque effet. La table range les
  **22 libelles de `wicons`, ni plus ni moins** (controle execute : 22/22 ranges, 0 entree de
  la table absente de `wicons`, 0 libelle de `wicons` non range). **18 de ces 22 apparaissent
  dans ce film** ; Cindershot, Plasma Pistol, Pulse Carbine et VK78 Commando sont ranges sans
  avoir ete exerces une seule fois — c est dit, pas tu.
- Repli `sobre` pour un libelle inconnu : couleur d equipe seulement, aucune forme qui suggere
  une nature. **0 cas sur ce film** ; la branche existe pour les autres.

**Repartition, mesuree sur CE film et sur la table reellement presente dans le fichier**
(relue dans le HTML, pas retapee) : tirs 147 — balistique 88, choc 17, aiguilles 13, plasma 12,
explosif 10, dur-lumiere 7. Morts 93 — melee 29, balistique 19, explosif 17, dur-lumiere 12,
choc 7, aiguilles 6, plasma 3.

**L eclair vers la victime ne pouvait PAS venir des tirs, et il ne vient pas de la.** Le bloc
TIRS du POC dit depuis toujours que la victime n est pas decodee ; c est toujours vrai. Le
calque des tirs garde donc l axe de VISEE (44 tirs sur 147) et l anneau pointille pour les 103
autres. C est le FIL DES MORTS qui nomme le tueur ET la victime : leurs deux positions se
relisent dans les trajectoires a l instant de la mort. Nouveau calque `drawKillFx`, bouton
dedie « Effets d arme ».

**Couverture mesuree, et publiee par le code qui dessine plutot que recopiee dans la phrase** :
93 morts, **89** avec le couple complet tueur+victime, **4** sans — celles-la ne recoivent
aucune direction. Les six nombres de la note du document sont ecrits par le JS depuis `KFX`,
`FEED` et `SHOTS` ; ils ne peuvent pas deriver du rendu. Verification a l ecran : 93 / 89 / 4
et 147 / 44 / 103, identiques aux mesures faites en Python par une seconde implementation.

**Controle interne — il ne valide aucun NOM, il valide la GEOMETRIE dont l effet se sert.**
La distance tueur-victime a l instant de la mort, mesuree sur ces 89 couples, separe la melee
du tir a distance **sans aucune table exterieure** : Energy Sword n=4 mediane 0,4 m (max 0,8) ;
nature MELEE n=3 mediane 0,5 m (max 0,5) ; Gravity Hammer n=22 mediane 2,7 m (il frappe en
zone) ; Needler n=6 mediane 3,8 m ; Sentinel Beam n=11 mediane 6,2 m ; S7 Sniper n=5 mediane
23,0 m ; Shock Rifle n=1 a 23,2 m. **L ordre obtenu est celui des portees du jeu, et il sort
des trajectoires seules — une source qui ignore tout de la table des armes.** C est ce controle
qui fixe le seuil de 8 m sous lequel la melee trace un arc entre les deux joueurs.

**Ecart assume par rapport a la consigne** : le Disruptor y etait range avec les armes
forerunner. Il a sa propre famille `choc` avec le Shock Rifle — un arc brise ne se lit pas
comme un rai continu. Remettre ces deux libelles sur `lumiere` suffit a revenir a la consigne,
et le commentaire du fichier le dit.

**Deux honnetetes de rendu qui ont demande du code** :
1. `cible` — un drapeau qui distingue une extremite REELLE (la victime) d une longueur de trace
   conventionnelle (62 px sur un tir). Le halo d impact de l explosif, la detonation du Needler
   et l arc de melee ne se jouent QUE si l extremite est reelle ; le rai de dur-lumiere s efface
   sur son dernier tiers sinon, parce qu un bord franc designerait un point qui n existe pas.
2. Aucun `Math.random` : les formes irregulieres (ondulation du plasma, arc brise du choc)
   sortent d un germe stable tire de l evenement. Revenir en arriere dans le rejeu redonne
   exactement la meme image.

**Mouvement reduit** : la feuille de style couvrait deja le DOM, mais le canvas est dessine en
JS et aucune regle CSS ne l atteint. La preference est donc lue en JS : sous `reduce`, les
effets ne sont plus animes du tout — losange a l origine, trait fin vers la cible, petit carre
ouvert a l extremite seulement si elle est reelle, **geometrie et opacite constantes** du debut
a la fin de la remanence. Verifie a l ecran aux ages 0, 5 et 11 : image identique.

**Une mesure a corrige un premier reglage.** La famille balistique decroissait au cube du temps
restant, conformement au « eclair court et sec » demande : a l ecran elle disparaissait des la
deuxieme image du rejeu, donc etait invisible aux vitesses 4x et 8x — et c est la famille la
plus nombreuse (88 tirs, 19 morts). L eclat garde une decroissance seche (carre) mais le TRAIT
vers la victime decroit en puissance 1,5 : il ne porte pas la meme chose, il porte QUI a tire
sur QUI.

**Ce qui n a pas ete fait, et pourquoi** : rien n a ete tente pour donner une victime aux
evenements de tir. Elle n est pas dans le film, la note du POC le dit depuis des semaines, et
apparier un tir a une mort proche (mesure : 42 tirs sur 147 apparies a +-0,5 s, 57 a +-1 s)
aurait fabrique des directions invraisemblables sur les tirs non mortels — le film n enregistre
que les tirs qui infligent des degats, pas seulement les tirs qui tuent.

**Prochaine etape** : les 4 libelles de `wicons` jamais exerces (Cindershot, Plasma Pistol,
Pulse Carbine, VK78 Commando) restent des familles non verifiees a l ecran ; un film qui les
porte les exercerait. Et le defaut de mise en page du fil signale la veille (`flex-shrink` sur
`.feed .e`) n a toujours pas ete tranche — il reste hors perimetre.

## [2026-07-28] Palette de capacites craquee par le groupe sofd — et elle contredit deux de mes quatre index

**Statut** : Complete

**Decision technique** : exploiter le croisement entre les deux chantiers. Le worktree voisin
resout l armement de vehicule par la chaine vcdd -> sofd -> sofa -> uwfa -> weap ; de mon cote
l identite de capacite est un rang resolu par parcours du bloc sofd (FUN_1407E7648). Meme
structure, abordee par deux bouts opposes.

**Resultat** : la palette est craquee. 27 entrees, nommees par murmur3 des identifiants de chaine
et recoupees avec l enumeration d equipement de l executable. Les rangs utiles :
  1 = ability_location_sensor (detecteur de menaces)   2 = ability_deployable_wall (mur)
  4 = ability_grapple_hook (grappin)                    5 = ability_evade (propulseur)
  6 = ability_knockback (repulseur)                     8 = active_camo
  9 = powerup_overshield                                11 = quantum_translocator
  12 = threat_seeker                                    23 = repair_field
Les rangs 0 et 7 sont la course et la melee, de categorie nulle : ce ne sont pas des capacites.

**LE CONTROLE ECHOUE 2 SUR 4, ET C EST LE RESULTAT LE PLUS INSTRUCTIF DE LA JOURNEE.**
  index 4 = grappin      -> REPRODUIT
  index 5 = propulseur   -> REPRODUIT
  index 3 = mur portatif -> CONTREDIT (le mur est au rang 2 ; le rang 3 n est pas une capacite)
  index 6 = capteur      -> CONTREDIT (le rang 6 est le repulseur ; le capteur est au rang 1)

Le detail qui donne sa valeur au resultat : les deux index REPRODUITS sont exactement ceux que ma
recette adossait a des TRIPLETS du releve Theater (3 grappins slots 512/513/516, 3 propulseurs
slots 514/518/519, 44 records chacun). Les deux CONTREDITS sont exactement ceux adosses a une
observation UNIQUE (« seul mur du releve », « seul capteur »). La table sofd valide donc la partie
du releve qui etait robuste et invalide celle qui reposait sur un seul temoignage.

Trois branches restent ouvertes, non tranchees faute de pouvoir relire le film :
  B1 le champ de 3 bits n est pas le rang de palette — ma propre fiche disait « 6 bits precedes
     d une porte, mesure decisive a faire en priorite », et cette mesure n a jamais tourne ;
  B2 le releve a confondu repulseur et capteur de menace (deux boitiers tenus en main) ;
  B3 le film n emploie pas ce sofd — argument mesure contre : c est le seul des 12 sofd du glpa ou
     le grappin soit au rang 4 et le propulseur au rang 5.

**LA PALETTE N EST PAS GLOBALE — et j avais conclu le contraire la veille.** Mesure : sur 46
equipements presents dans au moins deux des 12 sofd, 26 gardent leur rang et 20 en changent. Le
grappin est au rang 4 dans trois configurations et au rang 8 dans une quatrieme. Le sofd employe
est choisi a l execution (composant a unite+0x268). J avais ecrit le 2026-07-27 que la stabilite
etait « probablement acquise » par append-only ; c est vrai a l interieur d une famille, faux
entre familles. Ce qui sauve l exploitation : les trois sofd de la famille A ont un prefixe
rigoureusement identique sur les rangs 0 a 9, et c est la seule famille compatible avec un jeu de
capacites de joueur.

**Consequence operationnelle** : une table de decodage doit etre indexee par le sofd du match.
Determiner QUEL sofd s applique a un film donne est la question ouverte qui reste.

---

## [2026-07-28] Armes de kill au fil, objets au sol en echec, et quatre defauts graves de mise en page

**Statut** : Complete

**Armes de kill** : 93 morts sur 93, 87 nommees, 6 en repli « Autres », 27 tags bruts distincts.
Le tag brut est stocke a cote du libelle — et un detail justifie cette regle a lui seul : QUATRE
tags distincts sortent tous « Gravity Hammer ». Stocker la resolution a la place du brut aurait
fige a jamais ce qu on ignorait ce jour-la.

**Objets au sol : ECHEC, proprement documente.** Le discriminant de recurrence spatiale n a pas pu
etre evalue parce que son entree n existe pas : la POSITION des entites ti=42 et ti=37 n est pas
decodable aujourd hui. Trois routes essayees, trois refutees sur piece. La plus parlante : sur la
voie delta, le detecteur alimente avec les 178 slots ti=42 rend 5 echantillons contre 1 006 sur un
jeu de slots FANTOME de meme cardinalite — le signal est SOUS le bruit. Coherent avec la
grammaire : un objet pose n emet aucun delta de position, et porte il est parente.

**QUATRE DEFAUTS GRAVES DE MISE EN PAGE, tous mesures dans le navigateur.** C est la premiere fois
qu une verification visuelle a plusieurs instants tourne vraiment ; tous les defauts d affichage
precedents avaient ete trouves par l utilisateur.
  - la carte rendue trois fois trop petite (reduction 3,76 a 1920) ;
  - la carte RETRECISSANT quand la fenetre s elargit ;
  - le fil ecrase a 10 px par carte sur 89 % du match ;
  - le chronometre et le score se chevauchant des 1450 px.

**Cause racine, et c est mon erreur** : .wrap plafonnait la page a 1180 px pendant que je
dimensionnais la bande laterale en vw, donc sur la FENETRE. A 1920, la bande prenait 700 px d un
conteneur de 1180. Correction : plafond a min(1760px, 97vw), bande en pourcentage du conteneur.
Mesures apres : carte 428 -> 1054 px, reduction 3,76 -> 1,52, fil 10 -> 660 px, chevauchement
+38,6 px -> degagement de 757 px, troncatures 324 -> 0.

**LE MEME DEFAUT EST REVENU QUATRE FOIS SOUS QUATRE FORMES** : fil ecrase, equipes ecrasees a
41 px, fiches joueur coupees (175 colonnes sur 200), entrees du fil compressees a 15,6 px pour
63,5 px naturels. Cause unique : dans un conteneur flex les enfants ont flex-shrink: 1 par defaut,
donc ils se COMPRESSENT au lieu de deborder — et un conteneur qui ne deborde pas ne defile jamais.
Les deux regles qui le ferment sont consignees dans .ai/CAHIER_DES_CHARGES_POC.md : tout enfant
d un conteneur destine a defiler porte flex: 0 0 auto, et deux blocs voisins ne doivent jamais
pouvoir prendre la hauteur l un de l autre. Un plancher min-height ne suffit pas, il deplace le
probleme.

---

## [2026-07-28] Cahier des charges d affichage consigne

**Statut** : Complete

**Origine** : demande explicite de l utilisateur — « ca fait partie du cahier des charges donc
note le dans l etat de l art ou un doc ».

**Livrable** : .ai/CAHIER_DES_CHARGES_POC.md. Il rassemble les decisions d interface avec leur
raison, pour qu elles ne soient ni redecouvertes ni defaites a la passe suivante.

**La regle qui gouverne le reste, et qui est nouvelle** : ce qui decrit la CONFIANCE DU DECODEUR
n a pas sa place dans l interface. L ecran montre l etat du joueur, pas l etat de notre
connaissance. Consequences appliquees ce jour : vie en vert plein et bouclier en bleu plein sans
distinction mesure/suppose ; primaire et secondaire retires de l affichage (cette notion vient du
format, pas du jeu) ; le souligne vert designe desormais l objet TENU et rien d autre.
Une exception, et une seule : une LACUNE se signale toujours. « On ne sait pas » et « on a mesure
qu il n y a rien » sont deux etats differents.

**Autres decisions consignees** : rangee d objets unique (armes a gauche, grenades et capacite a
droite), sans libelle — le nom tronquait 324 fois sur 100 images et repetait la vignette ; fond
transparent sous les vignettes ; grenades et capacites a la meme hauteur que les armes ; quantite
en « x2 » ; mort en trois etats (clignotement, rouge tenu, eclat vert a la reapparition) ; effets
de tir par famille d arme, la melee n ayant PAS d eclair de bouche puisqu un coup de marteau n est
pas un tir.

**Piege technique consigne** : la carte d equipe etant reconstruite par innerHTML a chaque image,
une animation declaree normalement redemarre en boucle et reste figee sur sa premiere image — une
fiche est restee vert plein au lieu de clignoter. La parade est un delai negatif calcule sur l age
reel de l evenement. Le piege etait deja documente pour l echange d armes ; je l ai refait.

## [2026-07-28] Priorites arretees et fichier de suivi cree

**Statut** : Complete

**Decision** : ouvrir .ai/SUIVI_REPLAY_2D.md, tenu a jour au fil des sessions, pour repondre a la
question « ou en est-on ? » sans avoir a relire trois documents. Il se partage le terrain avec
RECETTE_LOADOUT (ce qui est decode), CAHIER_DES_CHARGES_POC (ce que l'ecran doit montrer) et
ETAT_DE_L_ART_CHANTIER_VOISIN (ce que le worktree voisin a resolu).

**Priorites arretees avec l'utilisateur** :
  1 medailles en images, compteur de reapparition, reprise des visuels d'armes ;
  2 hors Notion — trou de tirs, refutation de la structure, productionisation (declare IMPORTANT) ;
  3 etats actifs des capacites, etat vivant des objectifs, dispositifs de carte ;
  4 decoupage des zones sur le decor — priorite MOYENNE, avec une contrainte neuve : la solution
    doit valoir pour les 30 cartes, pas seulement pour Cliffhanger ;
  ecarte : les vehicules.

**Notion** : barres avec leurs chiffres — rendu des armes (20.1), noms des joueurs (23), et
eclairs de bouche (4) en partiel. Sur 20.1 j'ai consigne DEUX ECARTS a la demande d'origine,
decides en cours de route et valides depuis : le fond noir des vignettes retire (il decoupait un
rectangle sombre dans la carte), et primaire/secondaire retire de l'affichage (notion issue du
format, pas du jeu). Sur 4, ce qui ne se fera pas est ecrit : diriger l'eclair sur les TIRS est
impossible, les 147 evenements de tir ne portent aucune victime.

**Defaut d'affichage releve par l'utilisateur, consigne comme ouvert** : a 1:06, LORD PEINX13 est
sur le slot 520 et son dernier etat lu date de l'image 563 — dix secondes plus tot. L'etat dit
2 Fragmentation et 2 Plasma avec i47 non localise, donc selection indeterminee. Les deux « ? »
sont honnetes mais l'ecran ne dit pas que la lecture a dix secondes d'age, d'ou l'impression d'un
defaut. A traiter : montrer la fraicheur d'un etat d'inventaire, comme le rejeu le fait deja pour
le bouclier.

## [2026-07-28] Compteur de reapparition et age d une lecture d inventaire

**Statut** : Complete

**Ce qui est livre** : deux retouches ciblees du POC (scratchpad `replay_demo.html`), aucune
regeneration, aucun Go compile.

**1. Compteur de reapparition.** La fiche d un joueur mort dit desormais dans combien de temps il
revient, avec un nombre LU dans le film et non deduit d une constante : la reapparition se lit sur
l image de depart de la vie suivante du meme joueur — la donnee qui allume deja l eclat vert du
retour. Une barre montre l avancement depuis la mort, datee par le fil.

**Distribution mesuree, publiee plutot que remplacee par une constante** : 90 episodes de mort sur
000d5950, 82 se terminent par un retour lisible. Mediane 8,0 s, quartiles 7,9 et 8,0 s, 66 sur 82
(80,5 %) a 7,9 ou 8,0 s, 77 sur 82 (93,9 %) a moins d une seconde de la mediane. Palier net, mais
mesure sur UN match : ce n est pas une constante du jeu.

**Deux reserves portees a l ecran et dans le code** : (a) le nombre est une BORNE HAUTE — le film
est code en delta, un joueur qui reapparait sans bouger n emet rien, et les 4 episodes au-dela de
9 s (9,6 / 17,8 / 24,0 / 35,7 s) sont exactement ce cas ; (b) l unique episode de 5,6 s tombe la ou
deux vies de LORD PEINX13 se chevauchent (slots 594 et 595 se recouvrent sur 4251-4408), une
attribution vraisemblablement fautive, signalee et non corrigee.

**La lacune se declare** : 8 episodes n ont aucun retour lisible et affichent « retour ? », jamais
un delai devine. Ils commencent tous dans les 36 dernieres secondes ; un seul est trop tardif pour
qu un retour tienne dans le film, et pour les sept autres cinq pistes commencent dans la meme
fenetre SANS etre attribuees a un joueur.

**2. L age d une lecture d inventaire** — le defaut releve par l utilisateur a 1:06 est reproduit
puis repare. Le cas exact se rejoue : LORD PEINX13, slot 520, image-cle 563, image courante 660,
age 9,7 s. Une lecture recente est franche, une lecture ancienne s estompe (variable CSS `--fr`,
1,00 a 0,50 sur 200 images), et l infobulle donne l age exact avec le numero de l image-cle.

**La mesure demandee contredit l hypothese de la mission**, et c est elle qui justifie la
correction : sur les 4 985 images et les 8 joueurs, 21 899 fiches affichent un etat d inventaire ;
age median 8,4 s, quartiles 3,9 et 13,7 s, maximum 39,9 s. Sous une seconde : 7,1 % SEULEMENT, pas
90 %. L estompage sert donc en permanence.

**Les armes palissent avec le reste, et c est mesure** : quand les deux existent, l etat
d inventaire et la paire d armes viennent de la MEME image-cle 21 899 fois sur 21 899. Le lancer de
grenade, lui, ne palit jamais : evenement date, pas lecture d image-cle.

**Trouvaille de bord, traitee** : 7 380 fiches sur 29 279 (25,2 %) affichent des armes lues dans le
FUTUR — avant la premiere image-cle d une vie, `loadoutAt` se replie sur la plus proche a venir,
jusqu a 20,0 s en avance. Meme estompage sur la valeur absolue, et l infobulle le dit en toutes
lettres.

**Controles executes dans le navigateur, sur les 4 985 images** : 39 880 fiches, 8 901 mortes,
7 115 compteurs chiffres, 1 786 « retour ? » (7115 + 1786 = 8901), 29 279 fiches portant `--fr`,
moyenne 0,79, minimum 0,50, zero exception. Rendu verifie en theme sombre ET clair, a 1600 et
1366 px : aucun debordement, aucune barre de defilement horizontale. La fiche morte gagne 2 px
(le compteur prend la place de la rangee d armes, vide pendant la mort).

**Non traite, et note** : la periode ou le POC affiche des armes sans aucune ligne d inventaire
(9 080 fiches, 22,8 %) ne porte aucun marqueur de lacune — le cas « on ne sait pas » y est muet.
Hors perimetre de cette passe.

## [2026-07-28] Le trou de tirs n'est pas dans le film — et trois defauts graves d'affichage

**Statut** : Complete

### LE TROU DE TIRS : cause (c), rejet en aval

Enumeration SANS ANCRE sur les 27 morceaux, 30 418 paquets de type 0 : le record de degat qui
porte le tir (type 105) est present 832 fois, dont 519 en variante longue. **31 records longs et
22 courts tombent avant 66,0 s**, le premier a 30,2 s — soit 1,4 s AVANT la premiere mort. Ils
sont deja localises et journalises depuis le 2026-07-25.

La cause est la porte `uniqueSlotFor` de replay/shots.go, alimentee par la carte `owner` de
owners.go : elle ne couvre que 26 slots sur 99, et AUCUNE des 13 vies anterieures a 66 s. Un
record dont le slot n'y est pas est jete en silence.

C'est exactement la distinction que la methode designe comme l'erreur la plus couteuse : « le
format ne porte pas X » contre « notre lecteur ne trouve pas X ». L'enumeration sans ancre a
tranche.

**DECOUVERTE COLLATERALE, importante** : l'index d'attaquant du record n'est PAS l'index de joueur
du roster. Deux numerotations differentes. Permutation mesuree par DEUX chaines sans piece commune
(reappariement des 147 tirs publies ; recherche exhaustive sur 40 320 permutations contre le fil
des morts), qui donnent la meme bijection chiffre pour chiffre :
  0 whiteknight2519 · 1 JAVIERLOLITO540 · 2 JGtm · 3 LORD PEINX13
  4 IKE ILYA · 5 Akatsuki fire17 · 6 aldusbroncus · 7 VitaminA1688
Repere retabli, l'oracle des morts donne 49/95 contre un temoin a 7/95. Cas d'ecole : la premiere
mort (31,6 s, LORD PEINX13 au Disruptor par IKE ILYA) est precedee de SIX records Disruptor de
IKE ILYA entre 30,2 et 31,3 s. Ce sont litteralement les tirs manquants.
Cette permutation explique tres probablement le « TEMOIN 3 : Cliffhanger 0,24 » laisse NON RESOLU
au journal. Correction au passage : ce 0,24 etait calcule contre shots_fired, mauvais
denominateur ; contre shots_hit le rapport est 0,87 record par touche.

**Reparation mesuree, non estimee** : nommer chaque VIE par la MORT QUI LA TERMINE au lieu de la
faire voter. 91 vies sur 99, ecart median 0,0 image. Gain : tirs rattaches 147 -> 443 (3,0x), avant
66 s 0 -> 31, slots porteurs 17 -> 53, tranches vides 19/50 -> 8/50 (plancher du flux : 6/50).
Trois controles passes, dont la non-regression : sur les 125 tirs deja publies que le pont ideal
rattache aussi, le slot est identique 125 fois sur 125.

### LA REFUTATION DE LA STRUCTURE : AFFAIBLI

Elle n'avait jamais tourne. Elle a mordu.
- **Circularite PROUVEE par controle negatif** : rejouer la boucle avec des offsets FAUX (0x62,
  0x4C) rend le MEME score parfait 10 357/0 — en ne produisant qu'un cinquieme de la geometrie.
  Le « 10 357 sur 10 357 » ne discrimine donc rien.
- **Denominateur gonfle x8,3** : les 10 357 instances ne portent que 1 247 couples distincts.
- **Le controle publie de T1 surestime son pouvoir** : annonce « 100 % pour le bon appariement,
  5,1 % pour un LOD voisin » ; mesure : le LOD k+1 rend 98,55 %, le k+2 rend 99,46 %.
- **« 0 non resolue » est une CHAINE EN DUR** dans les legendes (c4_zones.py l.86, c5_img.py l.69).
- **Les deux ponts sont un argmin**, pas une propriete : trois boutons en dur, et la ligne retenue
  MINIMISE le nombre de segments. Pire, la tranche analysee n'est pas celle ou l'on marche —
  87,73 % des positions de joueur sont au-dessus ; a l'altitude mediane reelle, 1 passage de 2,2 m.
- **Le donut n'est pas dans le maillage** : plancher brut plein a 95,0 %, contraste couronne-noyau
  -10,9 pts. L'anneau n'apparait qu'apres une regle derivee du degagement.
- **La barre d'acceptation du chantier n'a jamais ete franchie** : sa propre docstring exige que le
  fer a cheval « sorte du lot » ; il arrive 14e sur 16 a tous les parametres testes.

**CE QUI SURVIT** : le chainage lui-meme n'est pas casse (balayage de tous les offsets u16 : seul
0x64 atteint 100 % sur T1, le suivant plafonne a 82,5 %). Et l'angle de coherence CONFIRME —
88,8 % des positions « debout » tombent a moins de 5 cm d'une surface du maillage annonce, contre
9,4 % pour un temoin a x/y permutes.

### TROIS DEFAUTS GRAVES D'AFFICHAGE, corriges

1. **La barre de bouclier etait multipliee par 100** : pleine des que le bouclier depassait 1 %,
   donc binaire, alors que le film porte 65 niveaux.
2. **La barre « vie » portait le BOUCLIER une seconde fois.** Mesure des colonnes : la 5 porte le
   bouclier en pourcentage, la 6 le meme en fraction, la 7 la VRAIE vie (163 valeurs). Le code
   lisait 5 et 6. La vie n'avait jamais ete lue.
3. **Un marqueur fantome dessine sur 99,7 % du film** : `t.pi === null` devenait la chaine "null"
   en servant de cle d'objet, donc une piste non attribuee recevait spawn = 0 et sa premiere
   position — transmise a 470,6 s — etait reportee jusqu'a l'image 0. Deux gardes posees : une
   piste sans joueur n'est pas un joueur, et un report en arriere a une portee (les quatre reports
   legitimes valent 22 a 35 s ; celui-ci en valait 470).

### AUTRES LIVRAISONS

Medailles : 166 visuels existaient deja dans static/medals/, rien a produire. 21 noms distincts
dans le fil, 21/21 resolus, 44/44 occurrences couvertes.
Compteur de reapparition : LU dans le film, pas deduit d'une constante. 90 episodes, 82 avec retour
lisible, mediane 8,0 s, 66 sur 82 entre 7,9 et 8,0 s.
Age d'une lecture d'inventaire : la mesure prealable a CONTREDIT l'hypothese de depart — age median
8,4 s et 7,1 % seulement sous la seconde, donc l'estompage sert en permanence.

## [2026-07-28] « Inventaire non lu » etait un bug — trouve par raisonnement sur le format

**Statut** : Complete

**Origine** : l'utilisateur, sans regarder le code : « c'est une structure universelle et
predictible, on a forcement une valeur ou 0, il n'y a pas d'entre-deux ».

**Il avait raison, et la regle violee etait deja appliquee ailleurs dans le meme fichier.** Le flux
est DIFFERENTIEL : un composant absent d'un record signifie INCHANGE, pas INCONNU. Le rejeu fait
le bon geste pour le bouclier (lastOf remonte colonne par colonne jusqu'a la derniere mesure) mais
invAt rendait le dernier etat et rien d'autre. Un joueur dont les grenades avaient ete lues a
l'image 563 et dont le record de 764 ne portait que la capacite les voyait disparaitre.

**Correction** : report champ par champ. Il est SUR parce qu'un slot EST une vie — il change a
chaque reapparition (LORD PEINX13 en occupe 13 sur ce film), donc le report ne peut jamais
franchir une mort.

**Mesure avant/apres** (balayage d'une image sur cinquante) : fiches vivantes sans ligne
d'inventaire 22,8 % -> 5,7 %. Le reste est une vraie lacune : un champ jamais lu depuis le debut
de la vie. « Pas encore transmis » et « transmis puis inchange » restent distincts.

**Lecon** : j'avais mesure « 0 etat sur 184 a des grenades presentes toutes a zero » et conclu que
la distinction etait saine. La mesure etait juste, la conclusion trop large — elle repondait a la
question « confond-on zero et absence ? » alors que la vraie question etait « traite-t-on
correctement l'absence ? ». Un raisonnement sur la NATURE du format a trouve ce qu'une mesure sur
la SORTIE ne pouvait pas voir.

## [2026-07-28] Requalification : la « permutation » etait notre propre tri

**Statut** : Complete (correction d'une annonce publiee le meme jour)

**Ce que j'avais annonce** : « l'index d'attaquant du record n'est PAS l'index de joueur du
roster, ce sont deux numerotations differentes », presente comme une decouverte sur le format.

**Objection de l'utilisateur** : « le player index me parait stable (sauf rage quit et nouvel
entrant), qu'il change en cours de partie ou selon les composants serait bizarre. Verifie aussi
que tu regardes bien le bon film. »

**Verification** : le pi du roster est un TRI ALPHABETIQUE ASCII (0 Akatsuki, 1 IKE, 2 JAVIER,
3 JGtm, 4 LORD, 5 VitaminA, 6 aldusbroncus, 7 whiteknight — majuscules avant minuscules). Ce n'est
pas un index du jeu, c'est un ordre que NOUS avons impose. Le film porte l'index interne du jeu.
Les deux n'ont aucune raison de coincider ; leur ecart n'apprend rien sur le format.

L'intuition de l'utilisateur etait juste : l'index joueur EST stable, et il ne change ni en cours
de partie ni selon le composant. Ce qui reste utile est la table de correspondance, pas la
« decouverte ».

**Film verifie, comme demande** : les chiffres cites (2 228 tires, 595 touches, 93 morts,
17 assistances) sont bien ceux de 000d5950, confirmes en base. Les 17 assistances concordent avec
ce que killsource a trouve et ce que le POC affiche — trois chemins.

**LA VRAIE CAUSE DES VIES NON NOMMEES, trouvee en repondant a l'autre question de l'utilisateur.**
Il demandait si on lit le fichier « comme une phrase » ou par adressage direct. Reponse lue dans
replay/owners.go : le rattachement se fait par VOTE HEURISTIQUE — les lancers de grenade votent
pour savoir quel index de joueur possede quel slot, le plus vote gagne — puis uniqueSlotFor exige
qu'exactement un slot corresponde, sinon l'evenement est jete en silence. Avec 70 lancers pour
99 vies, la carte ne couvre que 26 slots sur 99.

C'est la cause COMMUNE des vies non nommees, du trou de tirs et du plafond de rattachement. La
reparation deja identifiee (nommer chaque vie par la mort qui la termine) remplace le vote par la
lecture d'un fait date — le meme geste qui a debloque le chantier voisin.

**Lecon** : deux annonces trop larges corrigees dans la meme journee, toutes deux par des
objections de l'utilisateur portant sur la NATURE de ce qu'on manipule (un flux differentiel, un
index de joueur) et non sur les mesures. Les mesures etaient justes ; c'est leur interpretation
qui debordait.

## [2026-07-28] « Pourquoi on parle de vote ? » — le lien est ecrit dans le film

**Statut** : Complete (reorientation d'une proposition)

**La question de l'utilisateur** : « ok mais pourquoi on parle toujours de vote ? »

Elle est meilleure que ma proposition. Je proposais de REMPLACER le vote par une inference
meilleure (nommer les vies par la mort qui les termine). La bonne question est : pourquoi
vote-t-on, alors que le format est deterministe ?

**Reponse, trouvee en la posant** : l'archetype 5 EST le joueur (a ne pas confondre avec le 35,
le biped), et il porte deux composants jamais decodes :
    i10 player-primary-respawn-object-component
    i21 player-representation-component
Le second est, par son nom meme, l'entite qui represente le joueur dans le monde — donc son biped,
donc son slot. **Le rattachement joueur -> entite est SERIALISE. On vote parce que personne n'est
alle le lire.**

Le traverseur atteint deja l'archetype 5 (il consomme i16). Aller jusqu'a i21 est un probleme de
chaine de composants, qui a un chemin de solution connu : l'ancrage par signature, 98,4 % de
rappel et zero faux positif sur 650 641 positions.

**Ce que ca remet a sa place** : les trois defauts releves (vote, tri pris pour identite, rejet
silencieux) ne sont pas trois maladresses mais trois SYMPTOMES d'un decodage inacheve. On a comble
un trou de decodage par une inference, puis construit dessus. La reparation n'est pas d'ameliorer
l'inference, c'est de finir la lecture.

**Lecon de methode, la troisieme de la journee et la plus large** : mes trois annonces corrigees
aujourd'hui l'ont ete par des questions portant sur la NATURE de l'objet, jamais sur les chiffres.
« Un flux differentiel a-t-il des trous ? », « un index de joueur peut-il etre instable ? »,
« pourquoi voter dans un format deterministe ? ». Chaque fois, la mesure etait juste et
l'interpretation debordait. La regle a en tirer : avant de proposer une parade, se demander ce que
le format DEVRAIT contenir — et verifier qu'on l'a cherche.

## [2026-07-28] Plan de fiabilisation ecrit — le POC n'est pas viable en l'etat, et on dit pourquoi

**Statut** : Complete (plan ecrit, execution non commencee)

**Constat pose avec l'utilisateur** : le POC n'est pas viable. Quatre raisons chiffrees plutot
qu'un adjectif — rattachement par vote (26 slots sur 99), rejets silencieux (53 records perdus),
aucune couverture publiee (147 tirs affiches, 519 disponibles), et tout en injection manuelle dans
un fichier HTML.

**Les trois premieres sont le MEME defaut** : un decodage inacheve comble par une inference, puis
construit dessus. La cause est identifiee et elle est simple : le lien joueur -> entite est
SERIALISE dans le film (player-representation-component, archetype 5, rang i21), et le traverseur
s'arrete cinq rangs plus tot.

**Plan ecrit** : .ai/PLAN_REJEU_2D_FIABILISATION.md, six etapes sous contrat plan-execution.
  1 lire le lien joueur -> entite (avec repli mesure si la lecture echoue)
  2 brancher le rattachement dessus
  3 ne plus rien jeter en silence (compteurs categorises)
  4 publier une couverture, avec la porte du chantier voisin
  5 un ordre n'est pas une identite (renommage + garde-rail)
  6 sortir du POC vers l'architecture du depot

**Critere de succes** : le rejeu publie rattaches/disponibles par calque, et ce rapport depasse
85 % sur les tirs — contre 28 % aujourd'hui. Le critere est cale sur le PLAFOND DUR du format
(519 records pour 2 228 tirs : le film enregistre les degats, pas les tirs manques), pas sur 100 %.

**Sept decisions tranchees d'avance** pour empecher la derive en cours d'execution, dont la
premiere qui resume tout : on lit avant d'inferer, et une inference n'est acceptable qu'apres avoir
montre que la lecture est impossible — auquel cas elle se marque a l'ecran comme telle.
