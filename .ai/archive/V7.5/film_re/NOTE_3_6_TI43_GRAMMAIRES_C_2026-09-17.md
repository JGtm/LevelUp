# ti=43 (device) — grammaires du groupe C : `i30` a `i35` (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, item 3.6.d, volet `ti=43`),
> sans production : aucun film decode, aucun test Go, aucun fichier de production touche.
> Source : `HaloInfinite.exe` par Ghidra en LECTURE SEULE (HTTP direct `127.0.0.1:8089`,
> programme unique `HaloInfinite.exe`, base `0x140000000` ; `decompile_function`,
> `disassemble_function`, `read_memory`, `list_open_programs`). Rien renomme, rien commente,
> rien tague. Methode : `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` ; note d'archetype :
> `NOTE_3_6_TI43_2026-09-16.md` ; conventions des notes soeurs
> `NOTE_3_6_TI43_GRAMMAIRES_A_2026-09-17.md` (groupe A, `i19`..`i29`) et `_B_` (`i11`..`i21`, `i2`).
>
> Instrument / Ghidra lecture seule. Archetype `ti=43` (`device (dispositif de carte)`, colonne
> `archetype` de `ecs_table.tsv`). Groupe **`ti43c`** : rangs 11 a 16 de la liste des 22
> ecrivains `device-*` nommes le 16/09, soit **`i30` a `i35`**. Les rangs 17 a 21 (`i36`..`i40`)
> ne sont pas traites ici.
>
> `R(n)` = lecture de `n` bits ; `Q(n; a..b)` = `FUN_1406d84b4` avec `bits = n`, `min = a`,
> `max = b`, `b6 = 0`, `b7 = 1` (contrat §1). Comme dans les notes soeurs, « ecrivain » designe la
> fonction a `descripteur + 0x40`, qui est le LECTEUR du flux (`param_2`) et ecrit dans
> l'instance du composant `obj = *(param_3 + 0x10)`.

---

## 0. L'etat, colle depuis le depot, et la calibration

`apps/go-api/internal/games/halo_infinite/film/filmdec/testdata/ecs_table.tsv`, l. 965-970
(colonnes `ti`, `archetype`, `i`, `component`, `level`, `status`, le reste vide) :

```
43	device (dispositif de carte)	30	device-in-primary-mode-component	1	non_porte
43	device (dispositif de carte)	31	device-dispenser-monitors-changed-component	1	non_porte
43	device (dispositif de carte)	32	device-dispenser-state-flags-component	1	non_porte
43	device (dispositif de carte)	33	device-dispenser-require-los-component	1	non_porte
43	device (dispositif de carte)	34	device-animation-layer-settings-component	1	non_porte
43	device (dispositif de carte)	35	device-animation-layer-state-component	1	non_porte
```

Le pas 4 de la methode a ete REJOUE sur les six descripteurs (`read_memory`, 80 octets chacun,
petit-boutiste), et l'identite a ete verifiee jusqu'a la chaine : `+0x18` -> accesseur de 8
octets `LEA RAX,[rip+disp] ; RET` -> chaine en `.rdata`.

| Rang | i | Descripteur | `+0x10` (niveau) | `+0x18` accesseur -> chaine | `+0x40` lu | Ecrivain de la note du 16/09 | Concorde |
|---|---|---|---|---|---|---|---|
| 11 | `i30` | `0x143d0c158` | `0x14117b4a0` | `0x141177430` -> `0x143c98b18` = `device-in-primary-mode-component` | `0x142f02c20` | `FUN_142f02c20` | oui |
| 12 | `i31` | `0x143d0c100` | `0x14117b4a0` | `0x141177420` -> `0x143c98b40` = `device-dispenser-monitors-changed-component` | `0x142f02a48` | `FUN_142f02a48` | oui |
| 13 | `i32` | `0x143d0c4d8` | `0x14117b4a0` | `0x141177410` -> `0x143c98c10` = `device-dispenser-state-flags-component` | `0x142f02bcc` | `FUN_142f02bcc` | oui |
| 14 | `i33` | `0x143d0c528` | `0x14117b4a0` | `0x141177400` -> `0x143c98c38` = `device-dispenser-require-los-component` | `0x142f02b7c` | `FUN_142f02b7c` | oui |
| 15 | `i34` | `0x143d0c488` | `0x14117b4a0` | `0x1411773f0` -> `0x143c98bb8` = `device-animation-layer-settings-component` | `0x140f44104` | `FUN_140f44104` | oui |
| 16 | `i35` | `0x143d0c578` | `0x14117b4a0` | `0x1411773e0` -> `0x143c98be8` = `device-animation-layer-state-component` | `0x141076f68` | `FUN_141076f68` | oui |

Six concordances sur six. Les six `+0x10` valent `0x14117b4a0` = `MOV EAX,1 ; RET` (synthese
§4.1) : `param_4` = niveau **1** pour les six, ce que dit deja la colonne `level` de la table.

Deux ecarts de SIGNATURE, notes sans etre creuses (la relation utile `+0x40` tient et le
`+0x10` concorde) : le descripteur d'`i30` porte `0x14076ced0` a `+0x00` et `0x142f020cc` a
`+0x08` (la forme §2 de la methode attend `0x141191ab0` puis `0x14076ced0`) ; celui d'`i31` porte
`0x1408d8220` a `+0x48` (attendu `0x1404ab600`). Les quatre autres ont la forme canonique
(`0x141191ab0`, `0x14076ced0`, `0x14117b4a0`, nom, `0x1404ab600`, compagnon, `0x1411c8f80`,
`0x14076ce9c`, ecrivain, `0x1404ab600`). Ne pas se servir des slots `+0x00` / `+0x08` / `+0x48`
comme signature de famille : ils varient.

## 1. Les primitives rencontrees (relevees une fois, citees ensuite)

| Primitive | Ce qu'elle lit | Preuve collee |
|---|---|---|
| `FUN_1406cf008(flux)` | `R(1)` | notes A §1 et B §1 (non rouverte) |
| `R(8)` en ligne (dans `FUN_142f02a48`) | `R(8)` : 8 bits de poids fort de l'accumulateur `+0x30` | decompile : `*(int *)(param_2 + 0x2c) += 8` sur les deux chemins ; desassemblage `142f02a7c ADD dword ptr [RDX + 0x2c],0x8` (chemin rapide, `142f02a8f SHR R9,0x38`) et `142f02ae7 ADD dword ptr [RBX + 0x2c],0x8` (chemin lent) |
| `FUN_1424ccc74(flux, _, byte *out)` | `R(5)` | decompile : `+0x2c += 5` sur les deux chemins (`iVar1 - 0x3b` = `iVar1 - 0x40 + 5` cote lent), `bVar4 >> 3` = 5 bits de poids fort ; `*param_3 = bVar4` |
| `FUN_142af27f8(flux, _, byte *out)` | `R(2)` | decompile : `+0x2c += 2` sur les deux chemins (`iVar1 - 0x3e`), `bVar4 >> 6` ; `*param_3 = bVar4` |
| `FUN_141015740(flux, flux, uint *out)` | `R(32)` mot brut | note A §1 (`+0x2c += 0x20`) |
| `FUN_143206d34(_, flux, uint *out)` | porte `G = R(1)` ; si `G = 1` : `FUN_141015740` = `R(32)` -> `*out` ; si `G = 0` : `*out = 0xffffffff`, aucun bit ; **rend `G`** | decompile (4 lignes) ; desassemblage `143206d4c CALL 0x1406cf008` / `143206d56 JZ 0x143206d65` / `143206d5e CALL 0x141015740` / `143206d65 OR dword ptr [RBX],0xffffffff` / `143206d6d MOV AL,DIL` |
| `FUN_1408f0ac4(dest, flux, cat, b)` | `R(1)` porte ; si 1 : `FUN_1406d3140(.., flux, cat, dest+4)` = reference d'entite (si `cat == 1` : `R(1)` sonde, puis `R(W)`, puis `R(2)` generation), `FUN_1406cb0cc()` (0 bit) et `FUN_1405d5dbc` (0 bit, resolution) ; si 0 : `dest[1] = 0xffffffff` ; `dest[0]` = handle resolu ou `0xffffffff` | re-decompilee ici : `cVar2 = FUN_1406cf008(); if (cVar2 == '\0') param_1[1] = 0xffffffff; else { FUN_1406d3140(); cVar3 = FUN_1406cb0cc(); if (cVar3 != '\0') { puVar1 = FUN_1405d5dbc(local_res20, param_1[1], 0); uVar4 = *puVar1; goto LAB_1408f0afd; } } uVar4 = 0xffffffff; LAB_1408f0afd: *param_1 = uVar4; return cVar2;` — identique a la note A §1 ; la largeur `W` et les plages sont dans la note A §1 (`FUN_1406d3140`) et `NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md` §2.3 |
| `FUN_1406d84b4(flux ; XMM2 = min ; XMM3 = max ; [RSP+0x20] = n ; [RSP+0x28] = b6 ; [RSP+0x30] = b7)` | `R(n)` dequantifie dans `[min, max]` ; `b7 = 1` : brut `0` -> `min`, brut `N - 1` -> `max`, sinon `min + (brut - 1 + 0,5) * (max - min) / (N - 2)` avec `N = 2^n` | contrat des notes A §1, B §1, synthese §4.3 / §8.1. **Complement releve ici, necessaire a `i34` et `i35`** : desassemblage `0x1406d84b4..0x1406d8673` (3 919 octets) — (a) cote appele les arguments pile sont a `+8` : `1406d84d0 MOVSXD RBX,dword ptr [RSP + 0x28]` (= `n`), `1406d84d7 MOV SIL,byte ptr [RSP + 0x30]` (= `b6`), `1406d851e CMP byte ptr [RSP + 0x38],0x0` (= `b7`) ; (b) **aucun `CALL` ni `JMP` sortant** : fonction feuille ; (c) `XMM2` et `XMM3` sont seulement LUS (`1406d8529 MOVAPS XMM1,XMM3`, `1406d852f SUBSS XMM1,XMM2`, `1406d854b ADDSS XMM0,XMM2`, `1406d858f MOVAPS XMM1,XMM3`, `1406d8592 SUBSS XMM1,XMM2`, `1406d85b8 ADDSS XMM1,XMM2`, `1406d85c4 CVTSS2SD XMM1,XMM2`, `1406d85c8 CVTSS2SD XMM0,XMM3`, `1406d863c MOVAPS XMM0,XMM3`, `1406d8644 MOVAPS XMM0,XMM2`), `XMM4` n'apparait pas ; aucune instruction n'a `XMM2`, `XMM3` ou `XMM4` pour destination. Donc un appelant qui ne repose pas `XMM2` / `XMM3` / `XMM4` entre deux appels (cas de `FUN_143206e48`, 3e et 4e appels, et de `FUN_143206f24`, 2e appel) garde les bornes de l'appel precedent |

Constantes flottantes lues en `.rdata` (`read_memory`, 4 octets, petit-boutiste) :

| Adresse | Octets | Valeur |
|---|---|---|
| `0x143cd8370` | `00000000` | 0,0f (seuil de `FUN_143206f24`) |
| `0x143cd8374` | `0000803f` | `0x3f800000` = 1,0f |
| `0x143cd84a8` | `0000c842` | `0x42c80000` = 100,0f |
| `0x143cd84ec` | `000080bf` | `0xbf800000` = -1,0f |
| `0x143cd8934` | `00000040` | `0x40000000` = 2,0f |

---

## 2. `i30` — `device-in-primary-mode-component`

- Descripteur `0x143d0c158` ; ecrivain **`FUN_142f02c20`** (3 parametres).
- Decompile colle : `cVar2 = FUN_1406cf008(param_2); bVar3 = *(byte *)(lVar1 + 0x582); if (cVar2 == '\0') bVar3 &= 0xfb; else bVar3 |= 4; *(byte *)(lVar1 + 0x582) = bVar3; return 1;`
- Desassemblage colle : `142f02c2d CALL 0x1406cf008` ; `142f02c32 MOV CL,byte ptr [RBX + 0x582]` ;
  `142f02c3c OR CL,0x4` / `142f02c41 AND CL,0xfb` ; `142f02c44 MOV byte ptr [RBX + 0x582],CL` ;
  `142f02c4a MOV AL,0x1`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| en mode primaire (bit 2, masque `0x04`, de l'octet `+0x582`) | 1 | aucune | aucune |

- **Largeur totale : 1 bit, constante.** Meme octet de drapeaux que `i27` (bit 1, `0x02`) et
  `i28` (bit 0, `0x01`) de la note A ; `i33` y pose le bit 3 (§5).
- Statut : **releve**.

## 3. `i31` — `device-dispenser-monitors-changed-component`

- Descripteur `0x143d0c100` ; ecrivain **`FUN_142f02a48`** (3 parametres ; **rend un booleen**).
- Decompile colle (tete et boucle) : `*(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 8;`
  (les deux chemins) ; `*(uint *)(lVar2 + 0x5a0) = uVar9; if ((int)uVar9 < 9) { if (0 < (int)uVar9) { puVar8 = (undefined4 *)(lVar2 + 0x5a4); do { *puVar8 = 0xffffffff; puVar8[1] = 0xffffffff; FUN_1408f0ac4((longlong)(int)uVar10 * 8 + 0x5a4 + lVar2,param_2,1); uVar9 = (int)uVar10 + 1; uVar10 = (ulonglong)uVar9; puVar8 = puVar8 + 2; } while ((int)uVar9 < *(int *)(lVar2 + 0x5a0)); } uVar5 = 1; } else { uVar5 = 0; } return uVar5;`
- Desassemblage colle : `142f02b17 MOV dword ptr [RBP + 0x5a0],R9D` ; `142f02b1e CMP R9D,0x8` /
  `142f02b22 JLE 0x142f02b28` ; sinon `142f02b24 XOR AL,AL` (rend 0) ; `142f02b28 TEST R9D,R9D` /
  `142f02b2b JLE 0x142f02b65` ; boucle : `142f02b34 OR dword ptr [RSI],0xffffffff` ;
  `142f02b37 MOV R8D,0x1` (categorie 1) ; `142f02b3d OR dword ptr [RSI + 0x4],0xffffffff` ;
  `142f02b47 LEA RCX,[0x5a4 + RCX*0x8]` / `142f02b4f ADD RCX,RBP` ; `142f02b52 CALL 0x1408f0ac4` ;
  `142f02b57 INC EDI` ; `142f02b5d CMP EDI,dword ptr [RBP + 0x5a0]` / `142f02b63 JL 0x142f02b34` ;
  `142f02b65 MOV AL,0x1`. Le 4e argument de `FUN_1408f0ac4` (`R9B`) n'est pas pose par ce site
  (comme pour `i24`) : il ne sert qu'a la resolution, 0 bit.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| nombre `N` de moniteurs -> `+0x5a0` (uint, valeur brute 0..255) | 8 | aucune | aucune |
| **validite** : `N <= 8` ; sinon le lecteur rend `0` (echec, aucun bit de plus n'est lu) | 0 | — | — |
| pour `k = 0..N-1` : porte `P_k` (`FUN_1406cf008` via `FUN_1408f0ac4`) -> entree `+0x5a4 + 8k` | 1 | `N > 0` | aucune |
| sonde `S_k` (`FUN_1406cf008` dans `FUN_1406d3140`, parce que `cat == 1`) | 1 | `P_k == 1` | aucune |
| index de l'entite moniteur (`R(W)`) | `W = ceil(log2(cardinal))` | `P_k == 1` | **OUI** : `S_k == 0` -> plage de la categorie 1 (`(&DAT_1451f98d4)[2]`, **13 au defaut** = `IDLowBits` de `FrameConfig`, calibre par film) ; `S_k == 1` -> entree 4 (`DAT_1451f98f0/f4`, **9**) — identique a `i24` / `i29` (note A §7, §12) |
| generation du handle | 2 | `P_k == 1` | aucune |

- Valeur rendue par entree : `gen << 30 | base + index` -> `+0x5a8 + 8k` ; handle resolu ou
  `0xffffffff` -> `+0x5a4 + 8k`. Le tableau a **8 entrees de 8 octets** (`+0x5a4..+0x5e3`), d'ou
  la borne `N <= 8` : un `N` de 9 a 255 est un flux invalide, pas une valeur.
- **Largeur totale : `8 + somme_k (1 ou 4 + W)`, non constante.** `N = 0` : 8 bits ; `N = 8`, tous
  `P_k = 0` : 16 bits ; `N = 8`, tous `P_k = 1`, `S_k = 0`, `W = 13` : `8 + 8 x 17 = 144` bits.
- **Premier ecrivain du groupe `device-*` qui peut ECHOUER** (`return 0`) : les cinq autres de
  ce groupe et les onze du groupe A rendent toujours 1. Le port doit traiter `N > 8` comme
  une desynchronisation declaree (meme regime que l'etiquette `0xf` de ti=11 `i4`), jamais
  comme une valeur a borner.
- Statut : **releve** (grammaire complete ; `W` suit le profil du film).

## 4. `i32` — `device-dispenser-state-flags-component`

- Descripteur `0x143d0c4d8` ; ecrivain **`FUN_142f02bcc`**.
- Decompile colle : `FUN_1424ccc74(param_2,param_2,*(longlong *)(param_3 + 0x10) + 0x5e4); return 1;`
- Desassemblage colle : `142f02bd0 MOV R8,qword ptr [R8 + 0x10]` ; `142f02bd7 ADD R8,0x5e4` ;
  `142f02bde CALL 0x1424ccc74` ; `142f02be3 MOV AL,0x1`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| drapeaux d'etat du distributeur (`FUN_1424ccc74`) -> `+0x5e4` (byte, valeurs 0..31) | 5 | aucune | aucune |

- **Largeur totale : 5 bits, constante.** Le SENS des 5 bits n'est pas dans la grammaire ; le
  port lira 5 bits et les rangera tels quels (meme primitive `1424ccc74 = R(5)` deja citee
  dans `components_object_state.go:323`).
- Statut : **releve** (la largeur ; le sens des bits n'est pas dans la grammaire).

## 5. `i33` — `device-dispenser-require-los-component`

- Descripteur `0x143d0c528` ; ecrivain **`FUN_142f02b7c`**.
- Decompile colle : `cVar2 = FUN_1406cf008(param_2); bVar3 = *(byte *)(lVar1 + 0x582); if (cVar2 == '\0') bVar3 &= 0xf7; else bVar3 |= 8; *(byte *)(lVar1 + 0x582) = bVar3; return 1;`
- Desassemblage colle : `142f02b89 CALL 0x1406cf008` ; `142f02b8e MOV CL,byte ptr [RBX + 0x582]` ;
  `142f02b98 OR CL,0x8` / `142f02b9d AND CL,0xf7` ; `142f02ba0 MOV byte ptr [RBX + 0x582],CL`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| ligne de vue requise (bit 3, masque `0x08`, de l'octet `+0x582`) | 1 | aucune | aucune |

- **Largeur totale : 1 bit, constante.** L'octet `+0x582` a maintenant quatre bits nommes :
  bit 0 `i28` (en cours d'utilisation), bit 1 `i27` (drapeau des charges), bit 2 `i30` (mode
  primaire), bit 3 `i33` (ligne de vue requise) — quatre composants, un octet.
- Statut : **releve**.

## 6. `i34` — `device-animation-layer-settings-component`

- Descripteur `0x143d0c488` ; ecrivain **`FUN_140f44104`** ; sous-lecteurs **`FUN_143206e48`**
  (une couche) et **`FUN_143206d34`** (porte + identifiant de couche).
- Decompile colle de l'ecrivain : `lVar3 = lVar1 + 0x5e8; cVar2 = FUN_1406cf008(param_2); if (cVar2 == '\0') { for (; lVar3 != lVar1 + 0x6a8; lVar3 = lVar3 + 0x18) *(undefined4 *)(lVar3 + 0x14) = 0xffffffff; } else { for (; lVar3 != lVar1 + 0x6a8; lVar3 = lVar3 + 0x18) FUN_143206e48(lVar3,param_2); } return 1;`
  — `(0x6a8 - 0x5e8) / 0x18 = 0xc0 / 0x18 = 8` couches de 24 octets.
- Desassemblage colle : `140f4411a ADD RBX,0x5e8` ; `140f44124 CALL 0x1406cf008` ;
  `140f4412b JNZ 0x142451898` (la branche `A = 1` est dans une section froide) ; chemin `A = 0` :
  `140f44131 LEA RAX,[RBX + 0xc0]` ; `140f4413a OR dword ptr [RBX + 0x14],0xffffffff` ;
  `140f4413e ADD RBX,0x18` ; `140f44142 CMP RBX,RAX` / `140f44145 JNZ 0x140f4413a` (8 tours, 0 bit).
  Section froide : `142451898 LEA RDI,[RBX + 0xc0]` ; `1424518a1 MOV RDX,RSI` / `1424518a4 MOV RCX,RBX` /
  `1424518a7 CALL 0x143206e48` ; `1424518ac ADD RBX,0x18` ; `1424518b0 CMP RBX,RDI` /
  `1424518b3 JNZ 0x1424518a1` ; `1424518b5 JMP 0x140f44147`.
- Decompile colle de `FUN_143206e48(couche, flux)` : `cVar1 = FUN_143206d34(param_1,param_2,param_1 + 5); if (cVar1 != '\0') { uVar2 = FUN_1406d84b4(param_2); *param_1 = uVar2; uVar2 = FUN_1406d84b4(param_2); param_1[1] = uVar2; uVar2 = FUN_1406d84b4(param_2); param_1[2] = uVar2; uVar2 = FUN_1406d84b4(param_2); param_1[3] = uVar2; FUN_142af27f8(param_2); }`
  — les arguments de `FUN_1406d84b4` passent par `XMM2` / `XMM3` / la pile et n'apparaissent pas
  dans le decompile ; ils sont dans le desassemblage :
  `143206e52 LEA R8,[RCX + 0x14]` / `143206e5c CALL 0x143206d34` / `143206e63 JZ 0x143206f19` ;
  1er : `143206e69 MOVSS XMM3,dword ptr [0x143cd8934]` (2,0f) / `143206e74 MOVSS XMM2,dword ptr [0x143cd84ec]` (-1,0f) /
  `143206e7c MOV byte ptr [RSP + 0x30],0x1` / `143206e81 MOV byte ptr [RSP + 0x28],0x0` /
  `143206e86 MOV dword ptr [RSP + 0x20],0xe` / `143206e8e CALL 0x1406d84b4` / `143206eae MOVSS dword ptr [RDI],XMM0` ;
  2e : `143206e93 MOVSS XMM4,dword ptr [0x143cd84a8]` (100,0f) / `143206e9b XORPS XMM2,XMM2` /
  `143206ea3 MOVAPS XMM3,XMM4` / `143206eb2 MOV dword ptr [RSP + 0x20],0xe` / `143206eba CALL 0x1406d84b4` /
  `143206edc MOVSS dword ptr [RDI + 0x4],XMM0` ;
  3e : `143206ebf MOVSS XMM3,dword ptr [0x143cd8374]` (1,0f) / `143206ed4 MOV dword ptr [RSP + 0x20],0xa` /
  `143206ee1 CALL 0x1406d84b4` (**`XMM2` non repose : vaut encore 0**, §1) / `143206efe MOVSS dword ptr [RDI + 0x8],XMM0` ;
  4e : `143206ef3 MOVAPS XMM3,XMM4` (100,0f, **`XMM4` conserve a travers deux appels**, §1) /
  `143206ef6 MOV dword ptr [RSP + 0x20],0xe` / `143206f03 CALL 0x1406d84b4` (`XMM2` = 0) /
  `143206f0c MOVSS dword ptr [RDI + 0xc],XMM0` ; puis `143206f08 LEA R8,[RDI + 0x10]` /
  `143206f14 CALL 0x142af27f8`. `[RSP+0x30] = 1` (`b7`) et `[RSP+0x28] = 0` (`b6`) sont reposes
  avant chacun des quatre appels.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| porte globale `A` (`FUN_1406cf008`) | 1 | aucune | aucune |
| pour chaque couche `c = 0..7` (`couche = +0x5e8 + 0x18 c`) : porte de couche `G_c` (`FUN_1406cf008` via `FUN_143206d34`) | 1 | `A == 1` | aucune |
| identifiant de couche (`FUN_141015740`) -> `couche+0x14` (uint ; `0xffffffff` si `G_c == 0` ou `A == 0`) | 32 | `A == 1` et `G_c == 1` | aucune |
| `Q(14; -1..2)` -> `couche+0x00` (float) | 14 | idem | aucune |
| `Q(14; 0..100)` -> `couche+0x04` (float) | 14 | idem | aucune |
| `Q(10; 0..1)` -> `couche+0x08` (float) | 10 | idem | aucune |
| `Q(14; 0..100)` -> `couche+0x0c` (float) | 14 | idem | aucune |
| drapeaux de couche (`FUN_142af27f8`) -> `couche+0x10` (byte, 0..3) | 2 | idem | aucune |

- **Largeur totale : 1 bit si `A = 0` ; sinon `1 + somme_c (1 ou 87)`, non constante** : une
  couche ouverte coute `1 + 32 + 14 + 14 + 10 + 14 + 2 = 87` bits ; bornes `9` (huit couches
  fermees) a `1 + 8 x 87 = 697` bits.
- Dequantification (`b7 = 1`) : `Q(14; -1..2)` : `N = 16384`, brut `0` -> -1,0, brut `16383` -> 2,0,
  sinon `-1 + (brut - 1 + 0,5) x 3 / 16382` ; `Q(14; 0..100)` : `(brut - 1 + 0,5) x 100 / 16382` ;
  `Q(10; 0..1)` : `N = 1024`, `(brut - 1 + 0,5) / 1022`.
- Le SENS des quatre flottants (poids, vitesse, phase, duree ?) n'est pas dans la grammaire ;
  seules les bornes le sont : `[-1, 2]`, `[0, 100]`, `[0, 1]`, `[0, 100]`. Le compte de 8 couches
  est une constante de l'objet (taille du tableau), pas une entree de profil.
- Statut : **releve** (grammaire complete, entierement decidee par des bits du flux).

## 7. `i35` — `device-animation-layer-state-component`

- Descripteur `0x143d0c578` ; ecrivain **`FUN_141076f68`** ; sous-lecteur **`FUN_143206f24`**.
- Decompile colle de l'ecrivain : `cVar2 = FUN_1406cf008(param_2); if (cVar2 != '\0') { for (lVar3 = lVar1 + 0x6a8; lVar3 != lVar1 + 0x6e8; lVar3 = lVar3 + 8) { cVar2 = FUN_1406cf008(param_2); if (cVar2 != '\0') FUN_143206f24(lVar3,param_2); } } return 1;`
  — `(0x6e8 - 0x6a8) / 8 = 8` entrees de 8 octets.
- Desassemblage colle : `141076f81 CALL 0x1406cf008` / `141076f88 JNZ 0x1424957b4` (section
  froide) ; chemin `A = 0` : `141076f93 MOV AL,0x1` (rien d'autre). Section froide :
  `1424957b4 ADD RBX,0x6a8` / `1424957bb LEA RDI,[RBX + 0x40]` ; boucle : `1424957c4 CALL 0x1406cf008` /
  `1424957cb JZ 0x1424957d8` / `1424957d3 CALL 0x143206f24` / `1424957d8 ADD RBX,0x8` /
  `1424957dc CMP RBX,RDI` / `1424957df JNZ 0x1424957c1` ; `1424957e1 JMP 0x141076f8e`.
- Decompile colle de `FUN_143206f24(entree, flux)` : `fVar2 = (float)FUN_1406d84b4(param_2,param_2,0,DAT_143cd8374,10,0,1); fVar1 = DAT_143cd8370; param_1[1] = fVar2; if (fVar1 < fVar2) { uVar3 = FUN_1406d84b4(param_2); *param_1 = uVar3; }`
  Desassemblage : `143206f30 MOVSS XMM3,dword ptr [0x143cd8374]` (1,0f) / `143206f3b MOV byte ptr [RAX + -0x18],0x1` /
  `143206f42 MOV byte ptr [RAX + -0x20],0x0` / `143206f46 XORPS XMM2,XMM2` / `143206f49 MOV dword ptr [RAX + -0x28],0xa` /
  `143206f53 CALL 0x1406d84b4` / `143206f58 COMISS XMM0,dword ptr [0x143cd8370]` (0,0f) /
  `143206f5f MOVSS dword ptr [RDI + 0x4],XMM0` / `143206f64 JBE 0x143206f84` ;
  `143206f66 MOV byte ptr [RSP + 0x30],0x1` / `143206f6e MOV byte ptr [RSP + 0x28],0x0` /
  `143206f73 MOV dword ptr [RSP + 0x20],0xe` / `143206f7b CALL 0x1406d84b4` (**`XMM2 = 0` et
  `XMM3 = 1,0` non reposes : conserves**, §1) / `143206f80 MOVSS dword ptr [RDI],XMM0`.
  (`RAX` = `RSP` d'entree ; apres `PUSH RDI` et `SUB RSP,0x40`, `RAX - 0x18` = `RSP + 0x30` = `b7`,
  `RAX - 0x20` = `RSP + 0x28` = `b6`, `RAX - 0x28` = `RSP + 0x20` = `n` : meme contrat.)

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| porte globale `A` (`FUN_1406cf008`) | 1 | aucune | aucune |
| pour chaque entree `e = 0..7` (`entree = +0x6a8 + 8 e`) : porte d'entree `G_e` (`FUN_1406cf008`) | 1 | `A == 1` | aucune |
| poids `w_e` = `Q(10; 0..1)` -> `entree+0x04` (float) | 10 | `A == 1` et `G_e == 1` | aucune |
| valeur `Q(14; 0..1)` -> `entree+0x00` (float) | 14 | idem **et `w_e > 0,0f`**, ce qui equivaut a **brut_10 != 0** (voir ci-dessous) | aucune |

- La condition du dernier champ est ecrite sur la valeur DEQUANTIFIEE (`COMISS` avec 0,0f,
  `JBE` = saute si `<= 0` ou non ordonne). Avec `b7 = 1` elle se decide sur les bits bruts :
  brut `0` -> `min` = 0,0 exact -> `JBE` pris, pas de lecture ; brut `1..1022` ->
  `(brut - 1 + 0,5) / 1022 > 0` ; brut `1023` -> `max` = 1,0. Donc `w_e > 0` <=> `brut_10 != 0`,
  sans flottant ni cas non ordonne possible. Le port comparera l'entier, pas le flottant.
- **Largeur totale : 1 bit si `A = 0` ; sinon `1 + somme_e (1, 11 ou 25)`, non constante** :
  entree fermee 1 bit ; ouverte avec poids nul `1 + 10 = 11` ; ouverte avec poids non nul
  `1 + 10 + 14 = 25` ; bornes `9` a `1 + 8 x 25 = 201` bits.
- Les entrees non lues gardent leur contenu memoire (aucune valeur par defaut posee par ce
  lecteur, contrairement a `i34` qui pose `0xffffffff` sur l'identifiant).
- Statut : **releve** (grammaire complete, entierement decidee par des bits du flux).

---

## 8. Recapitulatif du groupe C

| Composant | Ecrivain | Grammaire | Total (bits) | Config | Statut |
|---|---|---|---|---|---|
| `i30` | `FUN_142f02c20` | `R(1)` (bit `0x04` de `+0x582`) | 1 | aucune | releve |
| `i31` | `FUN_142f02a48` | `R(8)` N (N <= 8 sinon echec) + N x [`R(1)` P + [P : `R(1)` S + `R(W)` + `R(2)`]] | 8 + somme (1 / 4 + W) ; 8..144 au defaut | `W` = plage cat. 1 (`IDLowBits`) ou 9 | releve |
| `i32` | `FUN_142f02bcc` | `R(5)` | 5 | aucune | releve |
| `i33` | `FUN_142f02b7c` | `R(1)` (bit `0x08` de `+0x582`) | 1 | aucune | releve |
| `i34` | `FUN_140f44104` | `R(1)` A + [A : 8 x (`R(1)` G + [G : `R(32)` + `Q(14; -1..2)` + `Q(14; 0..100)` + `Q(10; 0..1)` + `Q(14; 0..100)` + `R(2)`])] | 1 / 9..697 | aucune | releve |
| `i35` | `FUN_141076f68` | `R(1)` A + [A : 8 x (`R(1)` G + [G : `Q(10; 0..1)` w + [w != 0 : `Q(14; 0..1)`]])] | 1 / 9..201 | aucune | releve |

Six sur six releves, aucun `partiel`, aucun `non elucide`. Aucun appel virtuel, aucun
branchement sur un etat RUNTIME : toutes les conditions sont des bits du flux (`A`, `G`, `P`,
`S`, `brut_10 != 0`) ou un compte lu dans le flux (`N`). La seule dependance de config est la
plage des references d'entite de `i31`, deja modelisee (`varWidthBits(1)`).

Les offsets memoire poursuivent la suite du groupe A (`+0x540..+0x59c`) sans trou :
`+0x582` (bits 2 et 3, octet deja partage par `i27` / `i28`) ; `0x59c + 4 = 0x5a0` (`i31`, compte)
puis `0x5a4 + 8 x 8 = 0x5e4` (`i32`) ; `0x5e4 + 4 = 0x5e8` (`i34`, 8 x 0x18) ; `0x5e8 + 0xc0 = 0x6a8`
(`i35`, 8 x 8) ; fin `0x6e8`. Les six fonctions peuplent le meme objet que le groupe A, et
`i36`..`i40` devraient commencer a `+0x6e8` ou reprendre un bit de `+0x582` — a verifier au
groupe D, pas a supposer.

## 9. Ce que le port demandera

1. **Aucune primitive nouvelle.** `br.ReadBit()` (`FUN_1406cf008`) ; `br.ReadBits(2)`
   (`FUN_142af27f8`, deja `components_biped_spartan.go:244`) ; `br.ReadBits(5)`
   (`FUN_1424ccc74`, deja cite `components_object_state.go:323`) ; `br.ReadBits(8)` en ligne ;
   `br.ReadBits(32)` (`FUN_141015740`) ; `consume1408f0ac4(br, 1)` (`bit_leaf_readers.go:92`,
   categorie explicite) pour chaque moniteur d'`i31` ; la lecture quantifiee `Q` =
   `br.ReadBits(n)` puis UNE fonction commune de dequantification fidele au contrat `b7 = 1`
   (synthese §4.3 : `dequantMidpoint`, `components_managed_object.go:198`, n'est PAS ce
   contrat). Bornes etablies ici : `[-1, 2]`, `[0, 100]`, `[0, 1]` (`i34`), `[0, 1]` (`i35`).
2. **Une seule entree de profil** : `W` de la categorie 1 pour `i31` (`varWidthBits(1)`,
   `varwidth.go:86`, calibre par `FrameConfig.IDLowBits`) ; ne jamais figer 13. Les comptes
   fixes (8 couches d'`i34`, 8 entrees d'`i35`, tableau de 8 moniteurs d'`i31`) sont des
   constantes de l'objet du build courant, pas des entrees de profil.
3. **`i31` peut echouer** : `N > 8` -> l'ecrivain rend 0. Le port declare la desynchronisation
   (`DesyncAt`, meme regime que l'etiquette `0xf` de ti=11) et ne borne pas `N` a 8 : un `N` de
   9 a 255 signifie que le flux est deja faux en amont.
4. **`i35` : porter la condition sur l'entier** (`brut_10 != 0`), pas sur le flottant
   dequantifie ; consigner l'equivalence dans le godoc (elle repose sur `b7 = 1`).
5. **`i32`, `i34` : ranger sans interpreter** (5 bits de drapeaux ; quatre flottants et 2 bits par
   couche) : le decodeur n'a besoin que des largeurs pour fermer les records, et aucun ecrivain
   ne dit le sens de ces champs.
6. **Ordre** : les six se lisent apres `i29` et avant `i36`..`i40`, dont la grammaire reste a
   relever (groupe D : `FUN_142f02bb0`, `FUN_142f02c94`, `FUN_142f02d28`, `FUN_14107bb68`,
   `FUN_141fd7bc0`). Porter A + C sans D ne ferme aucun record (D5 du 1.9.1 bis) : le gate reste
   `ti=43` a `4 106/4 106` apres les 22.
7. **`ecs_table.tsv`** : les six lignes (`status`, `deser_addr`, `grammar`, `bits_typ`,
   `code_source`) dans le commit qui porte le code, jamais avant (`ecs_table_guard_test.go`).
   Pour `i31`, `i34`, `i35`, `bits_typ` n'est pas un entier unique : distinguer largeur FIXE et
   NOMINALE comme pour `i21` / `i24` / `i29`.
8. **Deux ecarts de signature de descripteur** (`i30` slots `+0x00` / `+0x08`, `i31` slot
   `+0x48`, §0) : sans effet sur la grammaire ; a ne pas utiliser comme signature de famille et
   a ne pas « corriger » dans la note de methode sans les avoir ouverts (zero fix hors perimetre).

## 10. Journal des appels Ghidra

Quarante-sept appels HTTP en lecture, aucune ecriture : `list_open_programs` (1 programme,
`HaloInfinite.exe`, base `0x140000000`) ; `read_memory` x 24 (6 descripteurs de 80 octets,
6 accesseurs de 8 octets, 6 chaines de 48 octets, `0x143cd8370` sur 8 octets, puis
`0x143cd8934`, `0x143cd84a8`, `0x143cd84ec`, `0x143cd8374`, `0x143cd8370` sur 4 octets) ;
`decompile_function` x 12 (`142f02c20`, `142f02a48`, `142f02bcc`, `142f02b7c`, `140f44104`,
`141076f68`, `143206e48`, `143206f24`, `1424ccc74`, `1408f0ac4`, `143206d34`, `142af27f8`) ;
`disassemble_function` x 10 (`142f02c20`, `142f02a48`, `142f02bcc`, `142f02b7c`, `140f44104`,
`141076f68`, `143206e48`, `143206f24`, `143206d34`, `1406d84b4`). Deux desassemblages ont
deroule bien au-dela de la fonction (`140f44104` : 167 Mo ; `141076f68` : 159 Mo — le serveur
suit le `JNZ` vers la section froide `0x1424518xx` / `0x1424957xx`) : plages utiles extraites
par `grep` sur les adresses, fichiers purges. Aucune erreur HTTP, aucun appel refuse.
