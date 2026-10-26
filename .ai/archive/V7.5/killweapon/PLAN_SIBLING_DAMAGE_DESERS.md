# CAMPAGNE : décoder les marqueurs DamageReport frères → arme par kill ~100% OFFLINE

> Objectif : porter la couverture arme-par-kill offline de **50% (0xd2 seul)** à **~97/98** (comme le live
> dual-hook), en décodant les **autres records DamageReport** (marqueurs frères `0xe9 0x89 0xc0 0xc2 0xc3 0xc7`)
> qui portent la famille d'arme (suffixe `0x42c9679f`) mais dont la structure diffère de `0xd2`.
> **Pur offline, sans CE runtime. INTERDIT : held-weapon (rejeté définitivement par le user).**

## Pourquoi cette piste (et pas le dead-state)
Le dead-state (`FUN_140c1dd44`) porte **tueur (EnumB) + victime (EnumA)** mais PAS la classe d'arme : verdict RE
06-07 (dans le code) = « *damage class resolved at REPLAY from the DamageReport pipeline (FUN_1407e00ac), not
stored per-kill in the film* ». Son `GlobalID(+0x10)` est une réf résolue via table runtime `DAT_144b404f0`
(vide au repos). Donc l'arme EN CLAIR vit dans les **records DamageReport** : `0xd2` (50%) + les frères (l'autre
moitié — exactement « l'autre 50% est ailleurs » du user).

## Lead concret
`FUN_14080c1f8` (déser `0xd2`) lit la famille via `FUN_14080dec4(param_4,"variant_name",…)`. Les AUTRES
fonctions qui appellent `FUN_14080dec4` ET sont dans le voisinage `0x14080Bxxx/0x14080Dxxx` sont les **déser
frères candidats** (chacune lit un variant_name d'arme) :
`FUN_14080cfe8 · FUN_14080d7cc · FUN_14080d878 · FUN_14080d900 · FUN_14080ddd0 · FUN_14080de24 ·
FUN_14080b034 · FUN_14080b1b8`.

## Étapes
1. **[en cours]** Décompiler chaque déser candidat → structure (attaquant ? famille ? suffixe ? layout bits).
2. Mapper chaque déser ↔ marqueur frère (par signature de structure / nb records).
3. Implémenter le décode Go générique par marqueur (attaquant slot + famille + temps).
4. Brancher dans `cmd/tmp_offgen` (roster type-8 déjà OK) → couverture combinée.
5. **Valider** contre la vérité-terrain live (`tmp_dualcap` 97/98) + `0xd2` (recouvrement).

## Acquis réutilisables
- Roster `slot→xuid` auto (bit-scan LE type-8) — FAIT.
- Décode `0xd2` (attaquant R5 bit36 + famille variant_name) — FAIT, validé.
- Kill-feed dynamique (xuid+gamertag) — FAIT.
- Vérité-terrain live `tools/ce/{dmgcapture_run2,killcapture}.bin` + `cmd/tmp_dualcap` (97/98) pour valider.

## Mapping déser ↔ marqueur : approche EMPIRIQUE (le dispatcher n'est pas traçable en statique)
Pour chaque marqueur frère, essayer le layout de CHAQUE déser candidat (au bon bit de départ) et garder celui
qui produit des **familles catalogue + suffixe** sur ~tous les records → ce déser = celui du marqueur. Puis le
champ R5 attaquant du déser donne l'attribution. (Sidestep du dispatcher vtable non-xref.)

## Findings desers (décompilation)
- **`FUN_14080cfe8`** (déser frère #1 confirmé) : lit `FUN_14080dec4(param_2,"variant-name",+4)` = FAMILLE
  d'arme (tag "variant-name" AVEC TIRET, ≠ "variant_name" underscore de 0xd2) ; résout un global-id via
  `DAT_144b404f0` ; puis R(18) (+0) · R(2) (+2) · **R(5)** (+0x1a, =val-1) · R(3) compteur `uVar7<5` ·
  **compteur × R(5)** (tableau +0xd, chaque `FUN_14080d69c` = R(1)+optR(32)). Les R(5) = champs joueur
  candidats (attaquant/victimes impactées). Sous-lecteurs (PAS top-level) : d7cc/d61c/d524/d4d0/d69c.
- **À décompiler** : `FUN_14080d878 · FUN_14080d900 · FUN_14080ddd0 · FUN_14080de24 · FUN_14080b034 ·
  FUN_14080b1b8` (top-level candidats restants).

## Journal
- 2026-06-12 : campagne ouverte. `FUN_14080cfe8` décompilé = déser frère (variant-name + R5 fields). Approche
  de mapping empirique définie. NEXT : décompiler les 6 autres candidats, puis matcher empiriquement aux
  marqueurs (0xe9/0x89/0xc0/0xc2/0xc3/0xc7) + localiser le R5 attaquant de chacun.

## DIAGNOSTIC DÉCISIF 2026-06-12 (cause du plafond 50%)
**Le plafond ~50% n'est PAS un manque de données ni un biais d'équipe** (testé : équipes équilibrées 22-28/43-50).
**C'est une DÉRIVE D'HORLOGE** entre la liste des kills (highlight TimeMS) et la liste des dégâts (packet ts) :
l'écart kill↔tick-tueur-le-plus-proche varie de **−109s à +14s** d'un kill à l'autre (diag tmp_offgen). La donnée
est là (ex: whiteknight a 14 ticks Stalker dans 0xd2) mais on ne peut pas lier kill↔dégât sans synchro.
- **Aussi confirmé** : les marqueurs frères (0xe9...) = APPARENCE d'arme (variantName/styleName/property-name),
  PAS du dégât. Seul 0xd2 (variant_name underscore, FUN_14080c1f8) est du DamageReport. Piste frères = MORTE.

## ✅ RÉSOLU 2026-06-12 — 96% PUR OFFLINE (warp linéaire + dernier-dégât-avant). Voir `cmd/tmp_offwarp`.
**Le décodeur FRAME n'est PAS nécessaire.** Causes réelles tranchées vs vérité-terrain live :
- **Donnée complète** : 519 `0xd2` = tous les dégâts (mêmes 17 armes que le live ; `FUN_14080a18c` traite 1 record/
  appel ; 605 events live = 605 ticks DISTINCTS, pas d'AoE ; 86 « manquants » = DoT/supercombine runtime, pas des
  records film). Frères `0xe9`… = apparence (distribution uniforme ~1/arme). **Diagnostic « frères=dégât » de la
  ligne ci-dessous : FAUX, infirmé.**
- **Plafond 50% = échelle d'horloge** : packet-ts flux ~946×/ms vs `TimeMS` jeu ; warp **linéaire** (R²=0.983).
  Critère gagnant = **dernier dégât du tueur AVANT l'instant du kill** (≠ « plus proche »).
- **Pipeline offline** (`cmd/tmp_offwarp`) : décode 0xd2 + roster type-8 + kill-feed highlight + warp linéaire dérivé
  offline (fit packet-ts→TimeMS, raffiné nearest puis last-before) + attribution last-before. **96% par kill** vs
  ground-truth live sur `000d5950`. Slots offline == idx live (8/8).
- **NEXT** : productionniser en service `internal/analysis/`. Pas de FRAME decoder, pas de held-weapon.

## ~~NOUVEAU PLAN (le vrai) : SYNCHRO HORLOGES via les MORTS → finir le décodeur FRAME~~ (ABANDONNÉ — voir ci-dessus)
Les morts apparaissent dans les DEUX horloges : highlight death (TimeMS) + dead-state dans le flux FRAME
(frame ts). Matcher dead-state-onset (frame ts, par biped slot 512+pi → roster) ↔ highlight death (TimeMS) →
table de warp frame_ts↔TimeMS → convertir chaque kill en frame_ts → matcher le dégât 0xd2 du tueur → arme.
- **Blocage** : le décodeur FRAME (filmdec) désync et ne décode fiablement qu'1 biped (slot 519) — les branches
  i54/i59/i63 (FUN_1408f0264 etc.) non portées coupent le walk avant les bipeds précédents.
- **Donc** : porter i54/i59/i63 au Ghidra → walk des 8 bipeds → temps de mort des 8 → warp → matching correct.
- Bonus possible : le dead-state porte aussi GlobalID (réf source-de-dégât) = arme par-mort si résoluble.
