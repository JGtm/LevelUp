# ti=43 (device) — grammaires du groupe D : `i36` a `i40` (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « porter les composants
> manquants archetype par archetype »), sans production : aucun film decode, aucun test Go,
> aucun fichier de production touche, rien de modifie dans Ghidra. Source : `HaloInfinite.exe`
> par Ghidra en LECTURE SEULE (HTTP direct `127.0.0.1:8089` : `decompile_function`,
> `disassemble_function`, `read_memory`). Methode : `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` ;
> note d'archetype : `NOTE_3_6_TI43_2026-09-16.md` ; conventions et primitives communes :
> `NOTE_3_6_TI43_GRAMMAIRES_A_2026-09-17.md` (note A) et `NOTE_3_6_TI43_GRAMMAIRES_B_2026-09-17.md`
> (note B).
>
> Instrument / Ghidra lecture seule. Archetype `ti=43` (device, dispositif de carte).
> Groupe `ti43d` : composants **`i36` a `i40`** (rangs 17 a 21 de la liste des 22 ecrivains
> `device-*`), soit les cinq derniers de l'archetype. Etat dans `ecs_table.tsv` avant cette
> note : les cinq lignes `43 <i> <nom> non_porte` avec `deser_addr`, `grammar`, `bits_typ`,
> `code_source` VIDES (colle depuis le fichier).

---

## 0. La chaine des descripteurs, rejouee sur les cinq composants

Pas 4 de la methode (`read_memory` a `descripteur + 0x40`, 8 octets, petit-boutiste) et, en
plus, pas 2 et 3 a rebours (`descripteur + 0x18` -> accesseur `LEA RAX,[rip+disp]` / `RET` ->
chaine) pour prouver l'IDENTITE de chaque descripteur, pas seulement son ecrivain.

| i | Composant (chaine lue a l'adresse calculee) | Descripteur | `+0x18` lu | Accesseur (8 octets lus) | Chaine | `+0x40` lu | Ecrivain | Concorde note ti=43 |
|---|---|---|---|---|---|---|---|---|
| 36 | `device-dispenser-state-component` | `0x143d0c390` | `0x1411773d0` | `488d05 9916b202 c3` -> `0x1411773d7 + 0x02b21699` | `0x143c98a70` | `0x142f02bb0` | `FUN_142f02bb0` | oui |
| 37 | `device-object-dispenser-timer-component` | `0x143d0c340` | `0x1411773c0` | `488d05 8116b202 c3` -> `0x1411773c7 + 0x02b21681` | `0x143c98a48` | `0x142f02c94` | `FUN_142f02c94` | oui |
| 38 | `device-position-transition-velocity-component` | `0x143d0c3e8` | `0x1411773b0` | `488d05 e116b202 c3` -> `0x1411773b7 + 0x02b216e1` | `0x143c98a98` | `0x142f02d28` | `FUN_142f02d28` | oui |
| 39 | `device-machine-flags-component` | `0x143d0c438` | `0x1411773a0` | `488d05 8116b202 c3` -> `0x1411773a7 + 0x02b21681` | `0x143c98a28` | `0x14107bb68` | `FUN_14107bb68` | oui |
| 40 | `device-interaction-start-time-override-component` | `0x143d0c2a0` | `0x141fd7da0` | `488d05 8111cc01 c3` -> `0x141fd7da7 + 0x01cc1181` | `0x143c98f28` | `0x141fd7bc0` | `FUN_141fd7bc0` | oui |

Cinq concordances sur cinq, nom ET ecrivain. Signature des descripteurs (`read_memory`, 80
octets a chaque descripteur, dix mots de 8) :

| Offset | `i36` | `i37` | `i38` | `i39` | `i40` |
|---|---|---|---|---|---|
| `+0x00` | `0x141191ab0` | `0x141191ab0` | **`0x14076ced0`** | `0x141191ab0` | `0x141191ab0` |
| `+0x08` | `0x14076ced0` | `0x14076ced0` | **`0x142f02128`** | `0x14076ced0` | `0x14076ced0` |
| `+0x10` | `0x14117b4a0` | `0x14117b4a0` | `0x14117b4a0` | `0x14117b4a0` | `0x14117b4a0` |
| `+0x18` | nom | nom | nom | nom | nom |
| `+0x20` | `0x1404ab600` | `0x1404ab600` | `0x1404ab600` | `0x1404ab600` | `0x1404ab600` |
| `+0x28` | `0x142f0591c` | `0x142f05b88` | `0x142f05d10` | `0x142f05ae0` | `0x141fd7c00` |
| `+0x30` | `0x1411c8f80` | `0x1411c8f80` | `0x1411c8f80` | `0x1411c8f80` | `0x1411c8f80` |
| `+0x38` | `0x14076ce9c` | `0x14076ce9c` | `0x14076ce9c` | `0x14076ce9c` | `0x14076ce9c` |
| `+0x40` | ecrivain | ecrivain | ecrivain | ecrivain | ecrivain |
| `+0x48` | **`0x1408d8220`** | `0x1404ab600` | `0x1404ab600` | `0x1404ab600` | `0x1404ab600` |

Remarques, consignees sans correction (zero fix hors perimetre) :

- **`i38`** a les deux premiers slots differents (`0x14076ced0`, `0x142f02128`) — meme
  phenomene que `i15` en note B §0 (signature decalee ou famille voisine). Les slots `+0x10` a
  `+0x48` sont standard et la relation `nom + 0x28 = ecrivain` tient : rien a creuser.
- **`i36`** a `+0x48 = 0x1408d8220` au lieu de `0x1404ab600` : dernier slot variant, sans
  effet sur `+0x18` / `+0x40`.
- Les constantes `+0x20` / `+0x48` et `+0x30` lues ici (`0x1404ab600`, `0x1411c8f80`) sont
  celles que la note B a aussi lues (§0, region d'`i2`) ; la note de methode §2 ecrit
  `0x14049b600` et `0x141c8f880` — chiffres permutes dans la note de methode, a corriger par
  son auteur, pas ici.

Convention (note A §0) : les cinq fonctions sont des LECTEURS de flux (`param_2` = flux,
`param_3` = contexte, `obj = *(param_3 + 0x10)`), appeles « ecrivains » par symetrie. Les
offsets `+0x53c`, `+0x584`, `+0x598`, `+0x6e8..0x6ff`, `+0x700..0x71f` sont ceux de la MEMOIRE
du composant. Ils s'inserent dans la suite deja relevee par les notes A et B (`+0x538`
`i18`, `+0x540` `i19`, ..., `+0x580`/`+0x582` `i27`/`i28`, `+0x59c` `i25`) : `i38` occupe
le trou `+0x53c` entre `i18` et `i19`, `i39` le mot `+0x584` apres l'octet de drapeaux
`+0x582`, `i40` le mot `+0x598` juste avant `i25` — un seul objet, comme attendu.

## 1. Les primitives rencontrees (relevees une fois, citees ensuite)

Rappel de l'objet de flux (note de methode §4) : `+0x2c` compteur de bits (chaque
`ADD [..+0x2c], N` = un champ de `N` bits, present DEUX fois : chemin rapide / chemin lent),
`+0x30` accumulateur, `+0x38` bits en reserve, `+0x40` curseur d'octets.

| Primitive | Ce qu'elle lit | Preuve collee | Deja relevee |
|---|---|---|---|
| `FUN_1406cf008(flux)` | `R(1)` | note A §1 | note A |
| `FUN_1406d84b4(flux, flux, xmm2 = min, xmm3 = max, [rsp+0x20] = n, [rsp+0x28] = b6, [rsp+0x30] = b7)` | `R(n)` dequantifie dans `[min, max]` ; `b6`, `b7` ne changent que la VALEUR | note A §1 (formule `b7 = 1` corrigee) ; complement de cette note : desassemblage `0x1406d84b4..0x1406d8673` (129 instructions) — **aucune ecriture dans `XMM2` ni `XMM3`** (10 lectures, 0 ecriture), ce qui rend legitime la reutilisation des bornes sans rechargement par `FUN_140d580d0` et `FUN_142ba78dc` (§3) | note A / B |
| `FUN_1408f0ac4(dest, flux, cat, b)` | `R(1)` porte P ; P = 1 : `FUN_1406d3140(?, flux, cat, dest+4)` puis `FUN_1406cb0cc()` (0 bit) et `FUN_1405d5dbc` (0 bit, resolution de handle) -> `dest+0` ; P = 0 : `dest[1] = 0xffffffff`, `dest[0] = 0xffffffff` | decompile relu : `cVar2 = FUN_1406cf008(); if (cVar2 == '\0') { param_1[1] = 0xffffffff; } else { FUN_1406d3140(); cVar3 = FUN_1406cb0cc(); if (cVar3 != '\0') { puVar1 = FUN_1405d5dbc(local_res20, param_1[1], 0); ... } }` | note A §1 |
| `FUN_1406d3140(?, flux, cat, uint*)` | reference d'entite : **si et seulement si `cat == 1`** : `R(1)` sonde ; puis `R(W)` avec `W = FUN_1406d310c(cardinal)` ; puis `R(2)` generation ; rend `gen << 30 \| base + index` | decompile relu : `if (DAT_144706104 != '\0') { uVar8 = (&DAT_1451f98d0)[param_3 * 2]; uVar7 = (&DAT_1451f98d4)[param_3 * 2]; }` (sinon `uVar7 = DAT_144706100`, base 0) ; `if (((param_3 == 1) && (cVar2 = FUN_1406cf008(param_2), cVar2 != '\0')) && ...) { uVar8 = DAT_1451f98f0; uVar7 = DAT_1451f98f4; }` ; `iVar3 = FUN_1406d310c(uVar7);` ; `+0x2c += iVar3` ; `+0x2c += 2` ; `*param_4 = uVar11 << 0x1e \| uVar8` | note A §1, `NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md` §2.3 |
| `FUN_1407f0354(flux, ., byte*)` | `R(5)` -> octet | decompile relu : `+0x2c += 5` sur les deux chemins (`1407f0379 ADD dword ptr [RCX + 0x2c],0x5`, `1407f03d0 ADD dword ptr [R10 + 0x2c],0x5`), `1407f038c SHR R9,0x3b` (5 bits de poids fort), `*param_3 = bVar4` ; **aucune instruction XMM** dans la fonction (63 instructions, `0x1407f0354..0x1407f042d`) | note B §1 |
| **`FUN_140d580d0(dest, flux, n, xmm3 = max)`** | `Q(n; 0..max)` -> `dest+0` ; `Q(n; 0..max)` -> `dest+4` ; appel de queue `FUN_1407f0354` = `R(5)` -> `dest+0xb` | decompile : `uVar1 = FUN_1406d84b4(param_2,param_2,0,param_4,param_3,0,1); *param_1 = uVar1; uVar1 = FUN_1406d84b4(param_2); param_1[1] = uVar1; FUN_1407f0354(param_2);` ; desassemblage : `140d580e4 MOV EBX,R8D` (n) ; `140d580f4 XORPS XMM2,XMM2` (min = 0) ; `140d580f1 MOV dword ptr [RAX + -0x28],EBX` (= `[RSP+0x20]` = n) ; `140d580ea MOV byte ptr [RAX + -0x20],0x0` (b6) ; `140d580e0 MOV byte ptr [RAX + -0x18],0x1` (b7) ; `140d580fa CALL 0x1406d84b4` ; `140d58110 MOVSS dword ptr [RDI],XMM0` ; `140d5810c MOV dword ptr [RSP + 0x20],EBX` ; `140d58107 MOV byte ptr [RSP + 0x28],0x0` ; `140d580ff MOV byte ptr [RSP + 0x30],0x1` ; `140d58114 CALL 0x1406d84b4` (XMM2/XMM3 non recharges, voir ci-dessus) ; `140d5811d MOVSS dword ptr [RDI + 0x4],XMM0` ; `140d58119 LEA R8,[RDI + 0xb]` ; `140d58134 JMP 0x1407f0354`. Seule ecriture XMM de la fonction : `XORPS XMM2,XMM2` (jamais `XMM3`) | le depot la connait deja comme « `R(n) + R(n) + FUN_1407f0354 = R(5)` » avec `n = 0x10` (`components_walk_batch9.go:43-45`, `consumeGameEngineCampaignTimer`) |
| **`FUN_142ba78dc(dest, flux, n, xmm3 = max)`** | `FUN_140d580d0(dest, flux, n, max)` puis `Q(n; 0..max)` -> `dest+0xc` | decompile : `FUN_140d580d0(); uVar1 = FUN_1406d84b4(param_2); *(param_1 + 0xc) = uVar1;` ; desassemblage : `142ba78eb MOV EBX,R8D` ; `142ba78ee MOV RDI,RDX` ; `142ba78f1 MOV RSI,RCX` ; `142ba78f4 CALL 0x140d580d0` (RCX = dest, RDX = flux, R8D = n, XMM3 = max transmis tels quels) ; `142ba78f9 MOV byte ptr [RSP + 0x30],0x1` ; `142ba78fe XORPS XMM2,XMM2` ; `142ba7901 MOV byte ptr [RSP + 0x28],0x0` ; `142ba7906 MOV RCX,RDI` ; `142ba7909 MOV dword ptr [RSP + 0x20],EBX` ; `142ba790d CALL 0x1406d84b4` ; `142ba7917 MOVSS dword ptr [RSI + 0xc],XMM0` | non |
| **`FUN_143206ae0(p, flux)`** | 2 x { `R(1)` G ; G = 1 : `FUN_1408f0ac4(entree, flux, cat = 0)` ; `R(3)` -> `entree+8` } | §2 | non |
| `R(3)` et `R(9)` « en ligne » | `if (0x40 - reserve < n) { chemin lent } else { +0x2c += n ; acc <<= n }` ; valeur = `n` bits de poids fort de l'accumulateur | `FUN_143206ae0` : `143206b3a ADD dword ptr [RBX + 0x2c],0x3` / `143206bab ADD dword ptr [RBX + 0x2c],0x3`, `143206b49 SHR R9,0x3d` ; `FUN_14107bb68` : `14107bb8e ADD dword ptr [RDX + 0x2c],0x9` / `14107bc06 ADD dword ptr [R10 + 0x2c],0x9`, `14107bba1 SHR R9,0x37` | motif de la note B §1 |

Constantes flottantes lues en `.rdata` (`read_memory`, 4 octets, petit-boutiste) :

| Adresse | Octets | Valeur | Utilisee par |
|---|---|---|---|
| `0x143d13304` | `00001644` | `0x44160000` = **600,0f** | `i37` |
| `0x143d13298` | `ffffef41` | `0x41efffff` = **29,999998f** (le flottant immediatement inferieur a 30,0f = `0x41f00000`) | `i38` |
| `0x143cd84e4` | `00007042` | `0x42700000` = 60,0f (deja en note A) | `i40` |

Globaux de plage (image statique, `read_memory` 32 octets) : `0x144706100` = `ff1f0000 01000000
fe070000 fe070000 60fb7344 01000000 ...` soit `DAT_144706100 = 0x1fff = 8191`, octet
`DAT_144706104 = 0x01` dans l'image ; `0x1451f98d4..0x1451f98f3` = 32 octets a zero (table
remplie a l'execution par `FUN_140d10bb0`, note des bandes §2.3). Le depot precise que
`DAT_144706104` est en outre RELU DEPUIS LE FILM (`varwidth.go:96+` : un `R(1)` en tete de
paquet si la version du format est `> 7`). Dans les deux etats de la garde, la categorie 0 donne
`W0 = ceil(log2(8191)) = 13` ou `ceil(log2(8191 - 512 = 7679)) = 13` : **13 au defaut**, mais
c'est une ENTREE DE PROFIL (`IDLowBits`), jamais une constante.

---

## 2. `i36` — `device-dispenser-state-component`

- Descripteur `0x143d0c390` ; ecrivain **`FUN_142f02bb0`** -> **`FUN_143206ae0(obj + 0x6e8, flux)`**.
- Desassemblage de l'ecrivain colle (7 instructions) : `142f02bb4 MOV RCX,qword ptr [R8 + 0x10]` ;
  `142f02bb8 ADD RCX,0x6e8` ; `142f02bbf CALL 0x143206ae0` ; `142f02bc4 MOV AL,0x1`. `RDX`
  (= flux) n'est pas touche : `FUN_143206ae0` le recoit en `param_2` (son decompile lit bien
  `param_2 + 0x2c`, `+0x30`, `+0x38`, `+0x40`).
- `FUN_143206ae0` colle : `143206aef LEA RSI,[RCX + 0x18]` (fin de boucle) ; `143206be3 ADD RDI,0xc`
  (pas) -> **deux entrees de 12 octets** (`obj+0x6e8..0x6f3`, `obj+0x6f4..0x6ff`) ; par entree :
  `143206b05 CALL 0x1406cf008` ; `143206b0c JZ 0x143206b1e` ; `143206b0e XOR R8D,R8D` /
  `143206b11 MOV RDX,RBX` / `143206b14 MOV RCX,RDI` / `143206b17 CALL 0x1408f0ac4` ;
  `143206b1e OR dword ptr [RDI],0xffffffff` / `143206b21 OR dword ptr [RDI + 0x4],0xffffffff` ;
  puis `143206b3a ADD dword ptr [RBX + 0x2c],0x3` (chemin rapide) / `143206bab ADD dword ptr
  [RBX + 0x2c],0x3` (chemin lent) ; `143206b49 SHR R9,0x3d` ; `143206bdf MOV byte ptr [RDI + 0x8],R9B`.
  Le 4e argument de `FUN_1408f0ac4` (`R9B`, `param_4`) n'est pas pose par ce site : il ne sert
  qu'a `FUN_1405d5dbc` (0 bit), comme pour `i24` (note A §7).

Par entree (x 2, dans l'ordre) :

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| porte externe G (`FUN_1406cf008`) | 1 | aucune | aucune |
| porte interne P (`FUN_1406cf008` via `FUN_1408f0ac4`) | 1 | `G == 1` (si `G == 0` : `entree+0 = entree+4 = 0xffffffff`, aucun bit) | aucune |
| index de l'entite distribuee (`R(W0)` dans `FUN_1406d3140`, **`cat == 0`** : PAS de bit de sonde) | `W0 = ceil(log2(cardinal_0))` | `G == 1` et `P == 1` | **OUI** : plage de la categorie 0 (`(&DAT_1451f98d4)[0]` si `DAT_144706104 != 0`, sinon `DAT_144706100`) — **13 au defaut** (`IDLowBits` de `FrameConfig`, calibre par film) ; la garde `DAT_144706104` est elle-meme un bit du film en tete de paquet |
| generation du handle (`R(2)`) | 2 | `G == 1` et `P == 1` | aucune |
| etat de l'emplacement -> `entree+8` (octet, valeurs 0..7) | 3 | aucune (lu que G vaille 0 ou 1) | aucune |

- Valeurs : `entree+4` recoit `gen << 30 | base + index` ; `entree+0` recoit le handle resolu
  (`FUN_1405d5dbc`, 0 bit) ou `0xffffffff`.
- **Largeur par entree : 4 (`G = 0`) ; 5 (`G = 1`, `P = 0`) ; `7 + W0` = 20 au defaut (`G = 1`,
  `P = 1`). Largeur totale : de 8 a `2 x (7 + W0)` = 40 au defaut** (combinaisons : 8, 9, 10,
  24, 25, 40 avec `W0 = 13`).
- Differences avec `i24` / `i29` (note A §7, §12), a ne pas confondre au port : (a) une porte
  EXTERNE G en plus de la porte P de `FUN_1408f0ac4` ; (b) la categorie est **0** et non 1 —
  aucun bit de sonde S n'est depense (le decompile de `FUN_1406d3140` n'appelle
  `FUN_1406cf008` que si `param_3 == 1`) ; (c) un `R(3)` de queue inconditionnel par entree ;
  (d) deux entrees.
- Port existant : `consume1408f0ac4(br, 0)` (`bit_leaf_readers.go:92`, categorie explicite
  obligatoire depuis le 2026-09-15) ; `varWidthRange(0)` rend deja `varWidthDefaultRange - 0x200`
  (`varwidth.go:71-73`), soit la meme largeur que la categorie 1 sans sonde.
- Statut : **releve** (grammaire complete ; `W0` suit le profil du film).

## 3. `i37` — `device-object-dispenser-timer-component`

- Descripteur `0x143d0c340` ; ecrivain **`FUN_142f02c94`**.
- Decompile colle : `for (lVar3 = lVar1 + 0x700; lVar3 != lVar1 + 0x720; lVar3 = lVar3 + 0x10) {
  cVar2 = FUN_1406cf008(param_2); if (cVar2 == '\0') { *(lVar3 + 4) = 0; *(lVar3 + 0xb) = 1; }
  else { FUN_142ba78dc(lVar3,param_2,10,DAT_143d13304); } }`.
- Desassemblage colle : `142f02caa ADD RBX,0x700` ; `142f02cb1 LEA RDI,[RBX + 0x20]` ;
  `142f02ce6 ADD RBX,0x10` -> **deux entrees de 16 octets** (`obj+0x700..0x70f`,
  `obj+0x710..0x71f`) ; par entree : `142f02cba CALL 0x1406cf008` ; `142f02cc1 JZ 0x142f02cde` ;
  `142f02cc3 MOVSS XMM3,dword ptr [0x143d13304]` (= 600,0f, recharge a CHAQUE tour) ;
  `142f02ccb MOV R8D,0xa` ; `142f02cd1 MOV RDX,RSI` ; `142f02cd4 MOV RCX,RBX` ;
  `142f02cd7 CALL 0x142ba78dc` ; branche G = 0 : `142f02cde AND dword ptr [RBX + 0x4],0x0` ;
  `142f02ce2 MOV byte ptr [RBX + 0xb],0x1`.

Par entree (x 2, dans l'ordre) :

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| porte G (`FUN_1406cf008`) | 1 | aucune | aucune (si `G == 0` : `entree+4 = 0`, `entree+0xb = 1`, aucun bit) |
| valeur 1 -> `entree+0` (float) : `FUN_1406d84b4` via `FUN_140d580d0`, `min = 0`, `max = 600,0f`, `b6 = 0`, `b7 = 1` | 10 | `G == 1` | aucune (`n = 10` litteral `MOV R8D,0xa` ; borne litterale de `.rdata`) |
| valeur 2 -> `entree+4` (float) : idem (bornes non rechargees mais preservees, §1) | 10 | `G == 1` | aucune |
| octet -> `entree+0xb` : `FUN_1407f0354` (appel de queue de `FUN_140d580d0`) | 5 | `G == 1` | aucune |
| valeur 3 -> `entree+0xc` (float) : `FUN_1406d84b4` dans `FUN_142ba78dc`, `min = 0`, `max = XMM3` (preserve : `FUN_140d580d0` n'ecrit que `XMM2`, `FUN_1407f0354` aucun XMM, `FUN_1406d84b4` ni `XMM2` ni `XMM3`), `b6 = 0`, `b7 = 1` | 10 | `G == 1` | aucune |

- **Largeur par entree : 1 (`G = 0`) ou 36 (`G = 1`). Largeur totale : 2, 37 ou 72.**
- Dequantification (note A, `b7 = 1`) : `N = 2^10 = 1024`, brut `0` -> 0,0, brut `1023` ->
  600,0, sinon `pas = 600 / 1022` et `valeur = (brut - 1) * pas + pas / 2`. Les trois floats
  sont presumes des temps (borne 600 = 10 minutes), l'ecrivain ne le dit pas.
- Le depot connait deja `FUN_140d580d0` sous la forme « `R(n) + R(n) + R(5)` » avec `n = 0x10`
  (`consumeGameEngineCampaignTimer`, `components_walk_batch9.go:43-45`) : meme forme, `n = 10`
  ici, plus un quatrieme `Q(10)` propre a `FUN_142ba78dc`.
- Statut : **releve**.

## 4. `i38` — `device-position-transition-velocity-component`

- Descripteur `0x143d0c3e8` (signature decalee sur deux slots, §0) ; ecrivain **`FUN_142f02d28`**.
- Decompile colle : `uVar2 = FUN_1406d84b4(param_2,param_2,0,DAT_143d13298,0x12,0,1);
  *(undefined4 *)(lVar1 + 0x53c) = uVar2;`
- Desassemblage colle : `142f02d2e MOVSS XMM3,dword ptr [0x143d13298]` ; `142f02d36 XORPS XMM2,XMM2` ;
  `142f02d4a MOV dword ptr [RSP + 0x20],0x12` ; `142f02d45 MOV byte ptr [RSP + 0x28],0x0` ;
  `142f02d40 MOV byte ptr [RSP + 0x30],0x1` ; `142f02d52 CALL 0x1406d84b4` ;
  `142f02d57 MOVSS dword ptr [RBX + 0x53c],XMM0`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| vitesse de transition (`FUN_1406d84b4`, `min = 0`, `max = 29,999998f` = `0x41efffff`, `b6 = 0`, `b7 = 1`) -> `+0x53c` (float) | 18 | aucune | aucune |

- **Largeur totale : 18 bits, constante.** `N = 2^18 = 262 144`, brut `0` -> 0,0, brut `262 143`
  -> 29,999998, sinon `pas = 29,999998 / 262 142` et `valeur = (brut - 1) * pas + pas / 2`. La
  borne est le predecesseur de 30,0f, pas 30,0f : a reproduire telle quelle si la valeur est
  publiee (elle ne change pas la largeur).
- `+0x53c` est le trou entre `device-position` (`+0x538`, `i18`) et `device-position-animation-name`
  (`+0x540`, `i19`).
- Statut : **releve**.

## 5. `i39` — `device-machine-flags-component`

- Descripteur `0x143d0c438` ; ecrivain **`FUN_14107bb68`**.
- Lecture en ligne, sans appel. Desassemblage colle : `14107bb72 MOV R11D,dword ptr [RDX + 0x38]` ;
  `14107bb76 MOV EAX,0x40` ; `14107bb7f SUB EAX,R11D` ; `14107bb89 CMP EAX,0x9` ; `14107bb8c JL 0x14107bbbd` ;
  chemin rapide : `14107bb8e ADD dword ptr [RDX + 0x2c],0x9` ; `14107bb95 SHL RAX,0x9` ;
  `14107bba1 SHR R9,0x37` ; `14107bbb0 MOV dword ptr [RBX + 0x584],R9D` ; chemin lent :
  `14107bc06 ADD dword ptr [R10 + 0x2c],0x9` ; `14107bc02 ADD R11D,-0x37` (= `- 0x40 + 9`) ;
  `14107bc2c SHR RDI,0x37`. Decompile : `*(int *)(param_2 + 0x2c) += 9` sur les deux chemins ;
  `*(uint *)(lVar2 + 0x584) = uVar6` avec `uVar6 = ... >> 0x17` (9 bits de poids fort d'un
  mot de 32).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| drapeaux de machine -> `+0x584` (dword, bits 0..8) | 9 | aucune | aucune |

- **Largeur totale : 9 bits, constante.** Le SENS de chacun des neuf bits n'est pas dans
  l'ecrivain (il range le mot brut) ; le port lira 9 bits et les rangera tels quels.
- Statut : **releve** (la largeur ; la semantique des bits n'est pas dans la grammaire).

## 6. `i40` — `device-interaction-start-time-override-component`

- Descripteur `0x143d0c2a0` ; ecrivain **`FUN_141fd7bc0`**.
- Decompile colle : `uVar2 = FUN_1406d84b4(param_2,param_2,0,DAT_143cd84e4,8,0,1);
  *(undefined4 *)(lVar1 + 0x598) = uVar2;`
- Desassemblage colle : `141fd7bc6 MOVSS XMM3,dword ptr [0x143cd84e4]` ; `141fd7bce XORPS XMM2,XMM2` ;
  `141fd7be2 MOV dword ptr [RSP + 0x20],0x8` ; `141fd7bdd MOV byte ptr [RSP + 0x28],0x0` ;
  `141fd7bd8 MOV byte ptr [RSP + 0x30],0x1` ; `141fd7bea CALL 0x1406d84b4` ;
  `141fd7bef MOVSS dword ptr [RBX + 0x598],XMM0`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| temps de depart force (`FUN_1406d84b4`, `min = 0`, `max = 60,0f`, `b6 = 0`, `b7 = 1`) -> `+0x598` (float ; secondes presumees, comme `i25`) | 8 | aucune | aucune |

- **Largeur totale : 8 bits, constante.** Meme forme et memes bornes que `i25`
  `device-interaction-hold-time-component` (note A §8, `+0x59c`) : `N = 256`, brut `0` -> 0,0,
  brut `255` -> 60,0, sinon `pas = 60 / 254` et `valeur = (brut - 1) * pas + pas / 2`.
- Statut : **releve**.

---

## 7. Recapitulatif du groupe D

| Composant | Ecrivain | Grammaire | Total (bits) | Config | Statut |
|---|---|---|---|---|---|
| `i36` | `FUN_142f02bb0` -> `FUN_143206ae0` | 2 x { `R(1)` G + [G : `R(1)` P + [P : `R(W0)` + `R(2)`]] + `R(3)` } | 8 .. `2 x (7 + W0)` = 40 au defaut | `W0` = plage cat. 0 (`IDLowBits`, 13 au defaut), garde `DAT_144706104` | releve |
| `i37` | `FUN_142f02c94` -> `FUN_142ba78dc` -> `FUN_140d580d0` | 2 x { `R(1)` G + [G : `Q(10; 0..600)` + `Q(10; 0..600)` + `R(5)` + `Q(10; 0..600)`] } | 2 / 37 / 72 | aucune | releve |
| `i38` | `FUN_142f02d28` | `Q(18; 0..29,999998)` | 18 | aucune | releve |
| `i39` | `FUN_14107bb68` | `R(9)` | 9 | aucune | releve |
| `i40` | `FUN_141fd7bc0` | `Q(8; 0..60)` | 8 | aucune | releve |

`Q(n; a..b)` = `FUN_1406d84b4` avec `bits = n`, `min = a`, `max = b`, `b6 = 0`, `b7 = 1`
(`N = 2^n`, brut `0` -> `a`, brut `N - 1` -> `b`, sinon `a + (brut - 1 + 0,5) * (b - a) / (N - 2)`,
note A §13 corrigee).

Cinq sur cinq releves, aucun `partiel`, aucun `non elucide`. Aucun appel virtuel, aucun
branchement sur un etat RUNTIME qui deciderait de la largeur : la seule dependance est la
table des plages de reference (`i36`, categorie 0), deja modelisee par `varWidthBits`. Somme des
largeurs constantes (`i38`, `i39`, `i40`) : 35 bits ; `i36` ajoute 8 a 40 ; `i37` ajoute 2 a 72.

Ce groupe clot la liste des 22 ecrivains `device-*` (notes A : `i19..i29`, C : `i30..i35`,
D : `i36..i40`). L'entree « ti=43 `i30`..`i40` : 11 grammaires » de la synthese §7 se reduit
donc, du cote D, a zero reste.

## 8. Ce que le port demandera

1. **Aucune primitive nouvelle.** `br.ReadBit()` (`FUN_1406cf008`), `br.ReadBits(3 | 5 | 9 | 18)`,
   `consume1408f0ac4(br, 0)` (`FUN_1408f0ac4`, **categorie 0**, `bit_leaf_readers.go:92`), la
   lecture quantifiee `FUN_1406d84b4` (`ReadBits(n)` puis dequantification `b7 = 1`, note A
   §14.1). `FUN_140d580d0` existe deja sous forme `R(n) + R(n) + R(5)` avec `n = 16`
   (`components_walk_batch9.go:43-45`) : le port de `i37` peut generaliser ce lecteur a `n`
   variable plutot que d'en ecrire une troisieme copie (regle des <= 2 copies).
2. **Une seule entree de profil** : `W0` de la categorie 0 pour `i36`. `varWidthRange(0)` rend
   deja `varWidthDefaultRange - 0x200` (`varwidth.go:71-73`) et la garde `DAT_144706104` est
   deja lue depuis le film (`varwidth.go:96+`). Passer par `consume1408f0ac4(br, 0)` et jamais
   par une largeur en dur ; 13 n'est qu'un defaut.
3. **`i36` n'est pas `i24` / `i29`** : porte EXTERNE puis porte interne, categorie 0 (pas de
   sonde), `R(3)` de queue inconditionnel, deux entrees. Un copier-coller du port de `i24`
   desynchroniserait le flux de 1 bit par entree (sonde en trop) et de 3 bits par entree (queue
   manquante).
4. **`i37`** : la porte G est lue par entree, et la borne 600 est rechargee a chaque tour par
   l'appelant ; les quatre champs d'une entree presente sont `10 + 10 + 5 + 10 = 35` bits dans
   cet ordre (valeur, valeur, octet, valeur), l'octet AVANT la troisieme valeur.
5. **Bornes a reproduire si les valeurs sont publiees** : `600,0f` (`i37`), `29,999998f`
   (`0x41efffff`, `i38` — pas 30,0f), `60,0f` (`i40`). Pour un port qui ne fait que sauter des
   bits, seules les largeurs comptent.
6. **Ordre** : `i36` a `i40` se lisent apres `i30` a `i35` (groupe C) et closent `ti=43`. Le
   gate reste celui du plan (D5 du 1.9.1 bis) : fermeture de `ti=43` de `0/4 106` a
   `4 106/4 106` sur les 7 bobines, mesuree au golden regenere — et le `partiel` d'`i2`
   (note B §13) peut alors redevenir le bloquant.
7. **`ecs_table.tsv`** : les cinq lignes (`status`, `deser_addr`, `grammar`, `bits_typ`,
   `code_source`) se mettent a jour dans le commit qui porte le code, jamais avant
   (`ecs_table_guard_test.go`). Pour `i36` et `i37`, `bits_typ` n'est pas un entier unique :
   distinguer largeur FIXE et NOMINALE comme la note d'`i0` le demande deja.
8. **Ecart de documentation a signaler au pilote, non corrige** : les constantes de signature
   `+0x20` / `+0x48` et `+0x30` de la note de methode §2 (`0x14049b600`, `0x141c8f880`) ne
   sont pas celles lues sur les cinq descripteurs de ce groupe ni sur ceux de la note B
   (`0x1404ab600`, `0x1411c8f80`).

## 9. Appels Ghidra en echec ou inutilisables

Aucun. Tous les appels (`read_memory` x 27, `decompile_function` x 10, `disassemble_function`
x 10) ont repondu ; les desassemblages sont restes dans les bornes de leurs fonctions.
