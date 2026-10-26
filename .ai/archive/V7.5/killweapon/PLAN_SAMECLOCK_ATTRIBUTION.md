# PLAN — Attribution arme-par-kill EXACTE via corrélation MÊME-HORLOGE (build dédié)

> Objectif : éliminer l'imprécision du warp temps-jeu↔flux en corrélant **dégât ↔ mort dans la MÊME horloge
> (flux/FRAME)**. Résultat visé : attribution exacte y compris Team Slayer (vs ~58% par-kill actuel sur loadout
> BR/MA40). Zéro held-weapon, zéro warp. Preuve que ça marche : le live dual-hook même-horloge = **97/98**.
> Statut : **direction validée, blocage localisé au bit près (2026-06-12). Build dédié à mener.**

## ⚠️ CORRECTION DE CAP (2026-06-12, reprise post-compact) — LIRE EN PREMIER
La priorité historique « porter i23 → i0/i1 → i5 » est **FAUSSE** pour atteindre le dead-state. Vérifié empiriquement
(`cmd/tmp_archdump`, `tmp_dspre`, `tmp_dsvalid`, `tmp_killeridx`) :
- **Le dead-state est i11.** Seuls **i0-i10** le précèdent. **i23 vient APRÈS i11 → hors-sujet.** Le score maskcorr
  d'i23 mesurait le désync d'i63 (records entièrement walkés), pas l'accès au dead-state.
- Dans les **death-frames** (3158 records i11), le combo dominant (44.5%) = **[i3 i6 i7 i9]** : **i0/i1/i5 absents**.
  Les composants qui dominent avant i11 = **i9 obje (94%), i7 damage-sections (90%), i6 region-state (84%), i3 (55%)**.
- **Taux de dead-state valide = 2.2% = bruit, UNIFORME sur tous les combos** (même `masque=i11 seul`). Donc le bug
  n'est **pas un composant pré-i11 isolé** : c'est le **framing du record + les largeurs des composants haute-fréquence**
  (obje i9, region i6, damage i7), toutes **runtime/quant peuplées au map-load** et lues 0 statiquement.
- **VRAI DÉBLOCAGE = décoder la config de précision/réplication du film** (`DAT_1445cc9e0`/`DAT_144632be0`/`DAT_145121140`
  depuis le header chunk_00/keyframe). C'est l'étape §3 [align] ci-dessous, **promue en priorité #1**. Sans ces
  largeurs, tout portage composant-par-composant reste futile.
- **Décision produit en attente** : (a) RE config précision (offline-pur exact, gros), (b) calibration CE par-map
  (exact, pas offline-pur), (c) livrer le warp (96% Fiesta / 58% Team Slayer). Cf thought_log 2026-06-12.

## MÉCANISME DE LARGEUR DE QUANTIFICATION — RÉSOLU (2026-06-12, RE Ghidra)
Les largeurs de bits des composants à précision dynamique ne sont PAS un mystère runtime : elles se **calculent**
depuis des constantes statiques du .exe. Chaîne RE :
- **Setup map-load** : `FUN_140be9a14` remplit les tables de précision (`DAT_1445cc9e0` défaut, `DAT_1445ccbe0` table
  indexée) depuis un contexte runtime + des constantes statiques.
- **Calcul de largeur** : `FUN_140be9b88(level, ranges)` → pour chaque axe :
  `width = min(0x1a, bitLen(ceil(range_axe / (2·step))))` (et `width=0x1a` si step < seuil `DAT_143cd837c`≈1e-4).
- **Step** : `FUN_140be9c78(level)` = `2^(16-level) / 120` (C = `0x3c088889` = 1/120 @143cd9758 ; pour level>16, step=1/120).
- **Range défaut** : `DAT_143b8c6b8` = **(-20000, +20000)** par axe (range = 40000), constante statique .rdata.
- **Consommateur position** : `FUN_14076e524` lit `DAT_144632be0` bits = index ; si `0xffffffff` → défaut
  `DAT_1445cc9e0 + level·0xc` (3 largeurs) ; sinon ligne table `DAT_1445ccbe0`. Puis `FUN_140cc5128` lit les 3 axes.

**Formule finale** : `step(L)=2^(16-L)/120` ; `width_axe(L)=min(26, bitLen(ceil(40000/(2·step(L)))))`.
**VALIDÉ** : L=0 → ceil(40000·120/2^17)=37 → bitLen(37)=**6/6/6** = exactement la mesure CE existante. ⇒ **offline-pur
exact atteignable** : largeurs calculées depuis le .exe, zéro CE/zéro table-de-map pour le chemin défaut.

**Cible réelle** (cf thought_log) : i0 position déjà bon (6/6/6) mais présent 12% des death-frames. Porter le MÊME
mécanisme à **i6 region (84%), i7 damage (90%), i9 obje (94%)** — chacun a son deser + son range/level statiques.
Puis re-mesurer `tmp_dsvalid` (doit monter au-dessus de 2.2%).

## Pourquoi cette voie (et pourquoi l'actuelle plafonne)
- Aujourd'hui : `0xd2` (arme, horloge **stream**/packet-ts) corrélé au kill-feed highlight (`TimeMS`, horloge
  **temps-jeu**) via un **warp linéaire**. R²=0.9967 mais résidu ~1-2s → en Team Slayer (BR+MA40 alternés sur ~1-2s)
  le « dernier dégât avant le kill » attrape la mauvaise rafale. **C'est un problème d'horloge, pas de méthode ni de
  held-weapon** (vérifié : mapping joueur = identité par Hungarian ; trace kill slot3 = rafale MA40 réelle ~1,5s
  après le TimeMS du highlight).
- Le jeu ne stocke AUCUN lien direct kill↔arme dans le film (vérifié par décompilation) : le kill-event component
  (`FUN_14104bd08`) porte victime+tueur+%dmg+**final-blow**+assistant mais **pas l'arme** ; le `0xd2` porte
  l'arme mais **pas la victime**. Le replay « triche » en re-calculant au runtime (état partagé même tick) = ce que
  fait le live dual-hook. Offline = il faut une **ancre dans la même horloge que les dégâts**.

## L'ancre = le DEAD-STATE dans le flux FRAME
- `consumeObjectDeadStateBiped` (`filmdec/components_object.go`) : DeadState porte **EnumA = victime** (R5) et
  **EnumB = tueur** (R5), + GlobalID. C'est exactement (victime, tueur) en horloge FRAME.
- Le dead-state est un composant **objet** (indice bas) → vient AVANT le désync i63 (biped, indice haut) dans le
  walk → **atteignable si l'alignement amont est bit-exact**.
- Les `0xd2` (dégâts) sont dans le MÊME flux (mêmes chunks REPLICATION_DATA, même packet-ts). Donc :
  ```
  mort (EnumB=tueur, packet-ts)  ⋈  0xd2 (attaquant==tueur, arme, packet-ts)  → arme du dernier dégât au tick de mort
  ```
  Aucun warp. Exact.

## Le blocage précis (localisé 2026-06-12)
Le walk FRAME (`filmdec/traverse.go::traverseComponentLoop`) désync AVANT le dead-state sur certains bipeds :
1. **Résidue du default-state biped** (`TraverseEntity`, typeIndex 35) : ~260 bits lus par des chemins dont les
   largeurs viennent du **header du film** (config précision/réplication : `DAT_1445cc9e0` axis widths,
   `DAT_144632be0` index width, `DAT_145121140`), lues à **0 statiquement**. Actuellement : préfixe bit-exact 120
   bits + `Skip(residue)` calibré. Si `defaultStateBits` est juste, le Skip **préserve l'alignement** — à vérifier
   par biped.
2. **Composants objet non portés** sur le chemin biped (cas `default` de `consumeByName`, ligne 223 :
   object position/velocity/angular/region/damage/constraint/parent/scale, unit-actor-control/state/malleable).
   Si l'un est présent AVANT le dead-state et non porté → désync.
3. **i63 (biped-action)** : value-gated loop, désync propre — mais APRÈS le dead-state, donc **pas bloquant** pour
   l'extraction des morts.
- Symptôme actuel : sonde `cmd/tmp_killeridx` → `Mort=true` sur 201 frames vs ~9 morts réelles = bits dead-state
  désalignés (mauvais amont).
- **CONCRÈTEMENT DÉMONTRÉ (2026-06-12, run `CGO_ENABLED=0 go run ./cmd/tmp_killeridx`)** : 1191 dead-states
  `Mort=true` vs **93 morts réelles** (sur-détection), et `EnumA`/`EnumB` sortent en **garbage** (valeurs 16/26/25/31/23,
  > 7 = indices joueur invalides). Les largeurs i0-i9 calibrées NE suffisent PAS : l'amont reste désaligné. ⇒ le
  décodeur ne produit AUCUN (victime, tueur) propre aujourd'hui. C'est l'étape 1-3 du plan ci-dessous qui débloque.

## DIAGNOSTIC i63 conduit (2026-06-12, autonomie)
- `tmp_framedesync` : **blocage #1 = `i63 biped-action-component` (typeIdx=35) sur 12909 frames** ; #2 = game-engine
  i0 (5501). Le dead-state (composant objet) est AVANT i63 → seul le 1er biped/frame est fiable (les suivants
  garbage après le désync i63). ⇒ porter le chemin biped jusqu'à i63 débloque tous les bipeds.
- `tmp_i63debug` : les tags 0-11 sont **tous valides** (présents dans les séquences OK). Désyncs **intermittents** :
  le même tag (0/2/7) est tantôt OK tantôt suivi de garbage (tag≥12). Cas « premier tag garbage » (`[26]`,`[31]`)
  = misalignement **EN AMONT** de la boucle loop1 (le subblock 96 bits + count1 sont confirmés ; donc un composant
  biped ANTÉRIEUR i47-i62 a une partie variable mal portée). i63 n'est que le point de DÉTECTION, pas la cause.
- **Réfuté empiriquement** : `defaultReplRange` (DAT_144706100, largeur du var-width-int) n'est PAS la cause — le
  balayage `tmp_replsweep` (largeurs 11-15) est plat (~3670 désyncs biped). (Setter `SetDefaultReplRange` ajouté.)
- **Réfuté empiriquement #2** : `recordStateParam` (count par-composant qui gate i10/i19/i20/i23) — sweep `tmp_replsweep`
  {0,1,2,3,4} plat (~3650-3717 désyncs). Donc PAS un réglage global non plus.
- **COUPABLES IDENTIFIÉS** (`tmp_maskcorr` : corrélation présence-composant vs désync, parmi records ayant i63) :
  | composant | Δ(désync−clean) |
  |---|---|
  | **i23 unit-malleable-property** | +0.19 (gated `recordStateParam`, partie variable) |
  | i1 object-translational-velocity-**dynamic-precision** | +0.18 (largeurs quantif position) |
  | i5 object-shield-vitality | +0.16 |
  | i0 object-position-**dynamic-precision** | +0.15 (largeurs quantif position) |
  | i37 weapon-state-rounds-inventory | +0.12 |
  Pas UN coupable mais **plusieurs composants à partie variable** (le désalignement s'accumule).
- **VRAI PROCHAIN PAS (priorisé)** : porter/valider bit-exact, dans l'ordre : **(1) i23 unit-malleable-property**
  (RE FUN + sa dépendance recordStateParam par-record), **(2) i0/i1 dynamic-precision** (sourcer les largeurs de
  quantif position du header film, vs le hardcode `TraversalPrecision{1,6/6/6}`), **(3) i5 shield-vitality**. Chaque
  fix doit faire MONTER le compte de records propres (`tmp_replsweep`/`tmp_framedesync`) et tendre vers l'alignement
  du dead-state (`tmp_killeridx` : EnumA/EnumB doivent tomber dans 0-7 et matcher la vérité kill-feed).
  Outils en place : `tmp_framedesync`, `tmp_i63debug`+biDebug, `tmp_maskcorr`, `tmp_replsweep`, `tmp_killeridx`.

## Plan d'exécution (étapes ordonnées, risque croissant)
1. **[diag]** Outil qui, pour CHAQUE biped de chaque frame, log `DesyncAt` + le nom du 1er composant non porté
   AVANT le dead-state. → liste exacte des composants à porter (priorisée par fréquence).
2. **[port]** Porter ces composants objet manquants (chacun = un `consume*` bit-exact depuis Ghidra : FUN cibles
   à décompiler, largeur mesurable). Réutiliser `BitReader` + le pattern des `consume*` existants.
3. **[align]** Valider le total `defaultStateBits` par biped (le Skip de résidue doit tomber pile sur le 1er
   composant présent). Source possible des largeurs : **parser le header du film (ChunkType=1)** pour
   `DAT_1445cc9e0`/`DAT_144632be0` au lieu du hardcode `TraversalPrecision{1,6/6/6}`.
4. **[extract]** Une fois l'amont bit-exact : extraire (frame/packet-ts, victime=EnumA, tueur=EnumB) pour toutes
   les morts. Valider le compte (~93-98 morts) + les paires (tueur,victime) vs la vérité live `killcapture.bin`.
5. **[correlate]** Corréler dégât↔mort par packet-ts (même horloge) : arme du kill = dernier `0xd2` du tueur
   (attaquant==EnumB) à packet-ts <= packet-ts(mort). Valider l'accuracy par-kill vs `tmp_dualcap` (cible : ~95%+
   y compris Team Slayer 9b191a7f).
6. **[prod]** Brancher dans `internal/sync/backfill_weapons.go` (remplace le warp + le fire-scanner).

## Acquis réutilisables (ne pas recommencer)
- `filmdec/bitreader.go` (BitReader MSB-first) + des dizaines de `consume*` portés.
- `DeadState{EnumA victime, EnumB tueur, GlobalID}` déjà décodé (`components_object.go`).
- Vérité-terrain : `tools/ce/{dmgcapture_run2,killcapture}.bin` (000d5950) + `9b191a7f_{dmg,kill}.bin`.
- Pipeline warp actuel (`cmd/tmp_offwarp`, 94% Fiesta / ~80% agrégé Team Slayer) = fallback en attendant.
- Setup Ghidra/CE MCP opérationnels (cf mémoires reference_ghidra_mcp_setup / reference_cheatengine_mcp_setup).

## Réfuté ce tour-ci (ne pas re-tenter)
- Devinage « marqueur type-0 + offset » pour le kill-event (0xe6=98/0xc7=97/0xd3) : **échoue** (en-tête de
  réplication de longueur variable ; histogramme tueur garbage). Le dispatcher accède aux descripteurs par index
  calculé → non traçable en RE statique. ⇒ passer par le dead-state dans le walk FRAME, PAS par un marqueur isolé.
