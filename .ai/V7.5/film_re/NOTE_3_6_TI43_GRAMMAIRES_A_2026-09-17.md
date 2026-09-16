# ti=43 (device) — grammaires du groupe A : `i19` a `i29` (2026-09-17)

> Preparation du lot 3.6, item 3.6.d (volet `ti=43`), sans production : aucun film decode,
> aucun test Go, aucun fichier de production. Source : `HaloInfinite.exe` par Ghidra en
> LECTURE SEULE (HTTP direct `127.0.0.1:8089` : `decompile_function`, `disassemble_function`,
> `read_memory`). Rien n'a ete modifie dans Ghidra. Methode :
> `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` ; note d'archetype :
> `NOTE_3_6_TI43_2026-09-16.md`.
>
> Instrument / Ghidra lecture seule. Archetype `ti=43` (device, dispositif de carte).
> Groupe `ti43a` : rangs 0 a 10 de la liste des 22 ecrivains `device-*` nommes, soit
> **`i19` a `i29`**.
>
> **Corrige apres verification independante (synthese du 2026-09-17)** : la formule de
> dequantification des valeurs INTERIEURES en mode `b7 = 1` (§1, §2, §5, §8, §13, §14.1)
> etait celle de la branche `b7 = 0`. Relecture du desassemblage de `FUN_1406d84b4` :
> `1406d851e CMP byte ptr [RSP+0x38],0x0` / `1406d8523 JNZ 0x1406d8577` ; dans la branche
> `1406d8577` : `1406d858c LEA EAX,[RCX + -0x2]` (diviseur `N - 2`), `1406d858f MOVAPS XMM1,XMM3`
> / `1406d8592 SUBSS XMM1,XMM2` (`max - min`), `1406d859d LEA EAX,[R9 + -0x1]` (`brut - 1`),
> `1406d85a1 DIVSS XMM1,XMM0` (`pas = (max - min) / (N - 2)`), `1406d85ac MULSS XMM0,XMM1`,
> `1406d85b0 MULSS XMM1,dword ptr [0x143cd84b0]` (`pas * 0,5`), `1406d85b8 ADDSS XMM1,XMM2`,
> `1406d85bc JMP 0x1406d854f` (`ADDSS XMM0,XMM1`). Donc, avec `b7 = 1` :
> `valeur = min + (brut - 1) * pas + pas / 2` avec `pas = (max - min) / (N - 2)` ; seuls les
> codes `0` (`1406d8644`) et `N - 1` (`1406d863c`) rendent les bornes. La forme
> `min + brut * pas + pas / 2` avec `pas = (max - min) / N` (`1406d8525..1406d854f`) n'est
> empruntee que si `b7 = 0`, ce qu'aucun ecrivain de ce groupe ne fait. Largeurs, ordre des
> champs, conditions et bornes exactes : inchanges (verificateur independant : 3 composants
> relus, 2 concordants, 1 discordant = cette formule).

---

## 0. Perimetre — pourquoi « i0 a i10 » se lit « i19 a i29 »

Dans `ecs_table.tsv`, les composants `i0` a `i10` de `ti=43` sont `porte` (`i2` est `partiel`)
avec `deser_addr` et `code_source` remplis (`dispatch_object.go`, `dispatch_player.go`) : leur
grammaire est deja dans le code et rien n'est a relever. La note d'archetype nomme 22 ecrivains
`device-*`, de `i19` a `i40`, et c'est cette liste que le lot 3.6 doit porter. Le groupe A prend
donc les rangs 0 a 10 de cette liste : `i19`, `i20`, ..., `i29`. Le groupe B (`i30` a `i40`)
n'est pas traite ici.

Avant toute lecture, le pas 4 de la methode a ete REJOUE sur les onze descripteurs
(`read_memory` a `descripteur + 0x40`, 8 octets, petit-boutiste) : les onze pointeurs
concordent avec la note d'archetype.

| Rang | Composant | Descripteur | `+0x40` lu | Ecrivain |
|---|---|---|---|---|
| 0 | `i19` | `0x143d0ce98` | `0x143d0ced8` -> `0x1410156e4` | `FUN_1410156e4` |
| 1 | `i20` | `0x143d0cfd8` | `0x143d0d018` -> `0x142f02d04` | `FUN_142f02d04` |
| 2 | `i21` | `0x143d0cf38` | `0x143d0cf78` -> `0x1407f0678` | `FUN_1407f0678` |
| 3 | `i22` | `0x143d0cf88` | `0x143d0cfc8` -> `0x14100d310` | `FUN_14100d310` |
| 4 | `i23` | `0x143d0c200` | `0x143d0c240` -> `0x14100d2d0` | `FUN_14100d2d0` |
| 5 | `i24` | `0x143d0c1a8` | `0x143d0c1e8` -> `0x141167910` | `FUN_141167910` |
| 6 | `i25` | `0x143d0c2f0` | `0x143d0c330` -> `0x142f02c54` | `FUN_142f02c54` |
| 7 | `i26` | `0x143d0c250` | `0x143d0c290` -> `0x142f029e4` | `FUN_142f029e4` |
| 8 | `i27` | `0x143d0c0b0` | `0x143d0c0f0` -> `0x140bee524` | `FUN_140bee524` |
| 9 | `i28` | `0x143d0c008` | `0x143d0c048` -> `0x142f02bec` | `FUN_142f02bec` |
| 10 | `i29` | `0x143d0c058` | `0x143d0c098` -> `0x14116fcb0` | `FUN_14116fcb0` |

Convention : les onze fonctions sont des LECTEURS (elles lisent le flux `param_2` et ecrivent
dans l'instance du composant `*(param_3 + 0x10) + offset`). La note les appelle « ecrivains »
par continuite avec la methode (§6 : ecrivain et lecteur sont symetriques dans ce moteur).
Les offsets `+0x540` a `+0x59c` releves ci-dessous sont ceux de la MEMOIRE du composant, pas
l'ordre du flux ; ils forment une suite contigue, ce qui confirme que les onze fonctions
peuplent le meme objet.

## 1. Les primitives rencontrees (relevees une fois, citees ensuite)

| Primitive | Ce qu'elle lit | Preuve collee |
|---|---|---|
| `FUN_1406cf008(flux)` | `R(1)` | decompile : `*(int *)(param_1 + 0x2c) += 1`, bit de poids fort de `+0x30` ; chemin lent `FUN_1406d6c7c(param_1, 1)` |
| `FUN_141015740(flux, flux, uint*)` | `R(32)` mot brut | decompile : `+0x2c += 0x20` sur les deux chemins ; `*param_3 = uVar4` |
| `FUN_14080dec4(flux, nom, uint*)` | `R(32)` mot brut ; `param_2` (la chaine de nom) n'est JAMAIS lue : etiquette de debug | decompile : `+0x2c += 0x20` ; `param_2` absent du corps |
| `FUN_1407f08f8(flux, ushort*)` | `R(8)` | decompile : `+0x2c += 8` ; `*param_2 = uVar5 & 0xff` |
| `FUN_1407f08bc(flux, ushort*)` | `R(1)` porte ; si 1 : `FUN_1407f08f8` = `R(8)` ; si 0 : `*param_2 = 0xffff` sans bit | decompile complet (4 lignes) |
| `FUN_1406d676c(flux, flux, dest, n)` | `R(n)` BRUT copie en octets dans `dest` : boucle `for (; 0x3f < n; n -= 0x40)` avec `+0x2c += 0x40`, puis reste `+0x2c += n` | decompile ; `n` est en BITS |
| `FUN_1406d84b4(flux, flux, xmm2=min, xmm3=max, [rsp+0x20]=bits, [rsp+0x28]=b6, [rsp+0x30]=b7)` | `R(bits)` dequantifie dans `[min, max]` | desassemblage : `1406d84d0 MOVSXD RBX, [RSP+0x28]` (= `bits` cote appele), `1406d84f1 ADD [R11+0x2c], EBX` ; `1406d8513 SHL EDI, CL` (`N = 1 << bits`), `1406d851b CMOVZ ECX, EDI` (avec `b6 = 0` : `N = 2^bits`, sinon `2^bits - 1`) ; **branche `b7 = 0`** (`1406d851e CMP byte ptr [RSP+0x38], 0` non pris) : `1406d852f SUBSS XMM1, XMM2` / `1406d8533 DIVSS XMM1, XMM0` (`pas = (max - min) / N`) ; `1406d853f MULSS` + `1406d8543 MULSS XMM1, [0x143cd84b0]` (`= 0x3f000000 = 0,5f`) + `1406d854b/854f ADDSS` (`valeur = min + brut * pas + pas / 2`) ; **branche `b7 = 1`** (`1406d8523 JNZ 0x1406d8577`, la seule empruntee dans ce groupe) : `1406d8577 TEST R9D` -> brut `0` rend `min` (`1406d8644 MOVAPS XMM0, XMM2`), `1406d8580 LEA EAX,[RCX-1]` / `1406d8586 JZ` -> brut `N - 1` rend `max` (`1406d863c MOVAPS XMM0, XMM3`), sinon `1406d858c LEA EAX,[RCX-2]` / `1406d85a1 DIVSS` (`pas = (max - min) / (N - 2)`), `1406d859d LEA EAX,[R9-1]` / `1406d85ac MULSS` (`(brut - 1) * pas`), `1406d85b0 MULSS [0x143cd84b0]` + `1406d85b8 ADDSS XMM2` + `1406d85bc JMP 1406d854f` (`valeur = min + (brut - 1) * pas + pas / 2`) — **corrige apres verification independante** : la premiere redaction attribuait la formule de `b7 = 0` a `b7 = 1` |
| `FUN_1408f0ac4(dest, flux, cat, b)` | `R(1)` porte ; si 1 : `FUN_1406d3140(?, flux, cat, dest+4)` puis `FUN_1406cb0cc()` (0 bit : test de `DAT_1445a78a0`) et `FUN_1405d5dbc` (0 bit : resolution de handle) ; si 0 : `dest[1] = 0xffffffff`, `dest[0] = 0xffffffff` | decompile + desassemblage `1408f0ade MOV R14D, R8D` / `1408f0b18 MOV R8D, R14D` (la categorie passe telle quelle) |
| `FUN_1406d3140(?, flux, cat, uint*)` | reference d'entite : si `cat == 1` : `R(1)` sonde (`FUN_1406cf008`) qui bascule la plage sur `DAT_1451f98f0/f4` (entree 4) ; puis `R(W)` avec `W = FUN_1406d310c(cardinal)` = `ceil(log2(cardinal))`, `cardinal = (&DAT_1451f98d4)[cat*2]` si `DAT_144706104 != 0` sinon `DAT_144706100` ; puis `R(2)` generation (`+0x2c += 2`) ; rend `gen << 30 \| base + index` | decompile (`if ((param_3 == 1) && (cVar2 = FUN_1406cf008(param_2), cVar2 != '\0')) ... DAT_1451f98f0 / DAT_1451f98f4`) ; tableau des plages dans `NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md` §2.3 |

Constantes flottantes lues en `.rdata` (`read_memory`, 4 octets, petit-boutiste) :

| Adresse | Octets | Valeur |
|---|---|---|
| `0x143cd873c` | `00002041` | `0x41200000` = 10,0f |
| `0x143cd8374` | `0000803f` | `0x3f800000` = 1,0f |
| `0x143cd84e4` | `00007042` | `0x42700000` = 60,0f |
| `0x143cd84b0` | `0000003f` | `0x3f000000` = 0,5f (demi-pas de `FUN_1406d84b4`) |

---

## 2. `i19` — `device-position-animation-name-component`

- Descripteur `0x143d0ce98` ; ecrivain **`FUN_1410156e4`**.
- Desassemblage colle : `1410156f8 LEA R8, [RDI + 0x540]` / `1410156ff CALL 0x141015740` ;
  `141015704 MOVSS XMM3, [0x143cd873c]` ; `141015714 XORPS XMM2, XMM2` ;
  `14101571c MOV dword ptr [RSP+0x20], 0xa` ; `141015717 MOV byte ptr [RSP+0x28], 0` ;
  `14101570f MOV byte ptr [RSP+0x30], 1` ; `141015724 CALL 0x1406d84b4` ;
  `141015730 MOVSS [RDI + 0x544], XMM0`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| identifiant d'animation (`FUN_141015740`) -> `+0x540` | 32 | aucune | aucune |
| position d'animation (`FUN_1406d84b4`, `min = 0`, `max = 10,0f`, `b6 = 0`, `b7 = 1`) -> `+0x544` (float) | 10 | aucune | aucune |

- **Largeur totale : 42 bits, constante.** Concorde avec le releve du lot 1.9.1 bis (plan
  l. 4277 : « R(32) identifiant puis largeur 0xa dequantifie dans [0, 10] »). Complement
  apporte ici (**corrige apres verification independante**, `b7 = 1`) : `N = 2^10 = 1024`,
  brut `0` -> 0,0 exact, brut `1023` -> 10,0 exact ; sinon `pas = 10 / (N - 2) = 10 / 1022`
  et `valeur = (brut - 1) * pas + pas / 2` (brut `1` -> 0,00489, brut `1022` -> 9,99511).
  La premiere redaction (`pas = 10 / 1024`, `brut * pas + pas / 2`) etait la formule de la
  branche `b7 = 0`, jamais empruntee par cet ecrivain.
- Statut : **releve**.

## 3. `i20` — `device-position-animation-control-component`

- Descripteur `0x143d0cfd8` ; ecrivain **`FUN_142f02d04`**.
- Desassemblage colle : `142f02d0c MOV R9D, 0x100` ; `142f02d12 ADD R8, 0x548` ;
  `142f02d1c CALL 0x1406d676c`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| bloc brut copie a `+0x548` (32 octets) | 256 | aucune | aucune |

- **Largeur totale : 256 bits, constante.** `FUN_1406d676c` avec `n = 0x100` : quatre tours de
  64 bits, reste nul. La nature des 32 octets (chaine ? nom d'animation de controle ?) n'est
  pas dite par l'ecrivain ; le port lira 256 bits et les rangera tels quels.
- Statut : **releve** (la largeur ; le SENS des 32 octets n'est pas dans la grammaire).

## 4. `i21` — `device-position-group-component`

- Descripteur `0x143d0cf38` ; ecrivain **`FUN_1407f0678`**.
- Desassemblage colle : `1407f0686 MOV R9D, 0x20` ; `1407f0692 LEA R8, [RBX + 0x568]` ;
  `1407f0699 CALL 0x1406d676c` ; `1407f069e LEA RDX, [RBX + 0x56c]` ;
  `1407f06a8 CALL 0x1407f08bc`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| mot brut (`FUN_1406d676c`, `n = 0x20`) -> `+0x568` | 32 | aucune | aucune |
| porte P (`FUN_1406cf008` via `FUN_1407f08bc`) | 1 | aucune | aucune |
| indice de groupe (`FUN_1407f08f8`) -> `+0x56c` (ushort, masque `& 0xff`) | 8 | `P == 1` (si `P == 0` : `+0x56c = 0xffff`, aucun bit) | aucune |

- **Largeur totale : 33 bits si `P = 0`, 41 bits si `P = 1`.** Le port existant de
  `FUN_1407f08bc` est `consumeGateR(br, 8)` (`components_biped_ability.go:191`).
- Statut : **releve**.

## 5. `i22` — `device-power-component`

- Descripteur `0x143d0cf88` ; ecrivain **`FUN_14100d310`**.
- Desassemblage colle : `14100d316 MOVSS XMM3, [0x143cd8374]` ; `14100d31e XORPS XMM2, XMM2` ;
  `14100d332 MOV dword ptr [RSP+0x20], 0xe` ; `14100d32d MOV byte ptr [RSP+0x28], 0` ;
  `14100d328 MOV byte ptr [RSP+0x30], 1` ; `14100d33a CALL 0x1406d84b4` ;
  `14100d33f MOVSS [RBX + 0x570], XMM0`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| niveau de puissance (`FUN_1406d84b4`, `min = 0`, `max = 1,0f`, `b6 = 0`, `b7 = 1`) -> `+0x570` (float) | 14 | aucune | aucune |

- **Largeur totale : 14 bits, constante.** `N = 2^14 = 16384`, brut `0` -> 0,0, brut `16383`
  -> 1,0 ; sinon (`b7 = 1`, **corrige apres verification independante**) `pas = 1 / (N - 2) =
  1 / 16382` et `valeur = (brut - 1) * pas + pas / 2`.
- Statut : **releve**.

## 6. `i23` — `device-power-group-component`

- Descripteur `0x143d0c200` ; ecrivain **`FUN_14100d2d0`**.
- Decompile colle : `uVar2 = FUN_1406d84b4(param_2,param_2,0,DAT_143cd8374,0xe,0,1); *(undefined4 *)(lVar1 + 0x574) = uVar2;`
  — meme forme que `i22`, a l'offset de destination pres (`+0x574`).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| puissance de groupe (`FUN_1406d84b4`, `min = 0`, `max = 1,0f`, `b6 = 0`, `b7 = 1`) -> `+0x574` (float) | 14 | aucune | aucune |

- **Largeur totale : 14 bits, constante.**
- Statut : **releve**.

## 7. `i24` — `device-interaction-in-progress-component`

- Descripteur `0x143d0c1a8` ; ecrivain **`FUN_141167910`**.
- Desassemblage colle : `141167918 MOV R8D, 0x1` ; `14116791e ADD RCX, 0x588` ;
  `141167925 OR dword ptr [RCX], 0xffffffff` ; `141167928 OR dword ptr [RCX + 0x4], 0xffffffff` ;
  `14116792c CALL 0x1408f0ac4`. Le 4e argument (`R9B`, `param_4`) n'est pas pose par ce site :
  il ne sert qu'a `FUN_1405d5dbc` (resolution de handle, 0 bit).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| porte P (`FUN_1406cf008` via `FUN_1408f0ac4`) | 1 | aucune | aucune |
| sonde S (`FUN_1406cf008` dans `FUN_1406d3140`, parce que `cat == 1`) | 1 | `P == 1` | aucune |
| index de l'entite en interaction (`R(W)`) | `W = ceil(log2(cardinal))` | `P == 1` | **OUI** : `S == 0` -> plage de la categorie 1 (`(&DAT_1451f98d4)[2]`, base 512, cardinal `DAT_144706100 - 512`, **13 au defaut** = `IDLowBits` de `FrameConfig`, calibre par film) ; `S == 1` -> entree 4 (`DAT_1451f98f0/f4`, base 512, cardinal 512, **9**) |
| generation du handle | 2 | `P == 1` | aucune |

- Valeur rendue : `gen << 30 | base + index` -> `+0x58c` ; puis `+0x588` recoit le resultat de
  `FUN_1405d5dbc` (resolution runtime, 0 bit) ou `0xffffffff` si `FUN_1406cb0cc` rend 0.
- **Largeur totale : 1 bit si `P = 0` ; sinon `1 + 1 + W + 2` = 17 bits au defaut (`W = 13`)
  ou 13 bits si `S = 1` (`W = 9`).** `W` de la categorie 1 est une ENTREE DE PROFIL, jamais une
  constante (la meme table que `IDLowBits`, `frame_records.go:39-44`).
- Le port existant est exactement `consume1408f0ac4(br, 1)` (`bit_leaf_readers.go:92`,
  categorie explicite obligatoire depuis le 2026-09-15).
- Statut : **releve** (grammaire complete ; la largeur `W` suit le profil du film).

## 8. `i25` — `device-interaction-hold-time-component`

- Descripteur `0x143d0c2f0` ; ecrivain **`FUN_142f02c54`**.
- Desassemblage colle : `142f02c5a MOVSS XMM3, [0x143cd84e4]` ; `142f02c62 XORPS XMM2, XMM2` ;
  `142f02c76 MOV dword ptr [RSP+0x20], 0x8` ; `142f02c71 MOV byte ptr [RSP+0x28], 0` ;
  `142f02c6c MOV byte ptr [RSP+0x30], 1` ; `142f02c7e CALL 0x1406d84b4` ;
  `142f02c83 MOVSS [RBX + 0x59c], XMM0`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| temps de maintien (`FUN_1406d84b4`, `min = 0`, `max = 60,0f`, `b6 = 0`, `b7 = 1`) -> `+0x59c` (float ; secondes presumees : la borne 60 le suggere, l'ecrivain ne le dit pas) | 8 | aucune | aucune |

- **Largeur totale : 8 bits, constante.** `N = 256`, brut `0` -> 0,0, brut `255` -> 60,0 ;
  sinon (`b7 = 1`, **corrige apres verification independante**) `pas = 60 / (N - 2) = 60 / 254
  = 0,23622` et `valeur = (brut - 1) * pas + pas / 2` (la premiere redaction donnait
  `pas = 60 / 256 = 0,234375`, formule de la branche `b7 = 0`).
- Statut : **releve**.

## 9. `i26` — `device-control-action-string-override-component`

- Descripteur `0x143d0c250` ; ecrivain **`FUN_142f029e4`**.
- Desassemblage colle : `142f02a03 LEA RDX, [0x143e0d890]` (`"primary-action-string-override"`)
  / `142f02a0d CALL 0x14080dec4` ; `142f02a1a LEA RDX, [0x143e0d868]`
  (`"secondary-action-string-override"`) / `142f02a21 CALL 0x14080dec4` ;
  `142f02a2f MOV [RDI + 0x578], EAX` ; `142f02a39 MOV [RDI + 0x57c], EAX`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| identifiant de chaine « primary-action-string-override » (`FUN_14080dec4`) -> `+0x578` | 32 | aucune | aucune |
| identifiant de chaine « secondary-action-string-override » (`FUN_14080dec4`) -> `+0x57c` | 32 | aucune | aucune |

- **Largeur totale : 64 bits, constante.** Les deux chaines en clair sont des etiquettes de
  debug passees a `FUN_14080dec4`, qui ne les lit pas ; elles ne sont PAS dans le flux. Meme
  primitive que `player-representation-name` (`biped_creation.go:67`) et `voice-designator`
  (`components_biped_ability.go:170`).
- Statut : **releve**.

## 10. `i27` — `device-health-station-charges-component`

- Descripteur `0x143d0c0b0` ; ecrivain **`FUN_140bee524`**.
- Decompile colle : `cVar4 = FUN_1406cf008(param_2);` puis `bVar5 = *(byte *)(lVar2 + 0x582) | 2` /
  `& 0xfd` ; puis lecture en ligne : `*(int *)(param_2 + 0x2c) += 6` (les deux chemins,
  `iVar1 - 0x3a` = `iVar1 - 0x40 + 6` sur le chemin lent), `uVar8 = ... >> 10` (6 bits de poids
  fort d'un mot de 16), `*(ushort *)(lVar2 + 0x580) = uVar8`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| drapeau (bit 1, masque `0x02`, de l'octet `+0x582`) | 1 | aucune | aucune |
| nombre de charges -> `+0x580` (ushort, valeurs 0..63) | 6 | aucune | aucune |

- **Largeur totale : 7 bits, constante.** Ordre du flux : le drapeau d'abord, les 6 bits
  ensuite.
- Statut : **releve**.

## 11. `i28` — `device-health-station-in-use-component`

- Descripteur `0x143d0c008` ; ecrivain **`FUN_142f02bec`**.
- Decompile colle : `cVar2 = FUN_1406cf008(param_2); bVar3 = *(byte *)(lVar1 + 0x582); if (cVar2 == '\0') bVar3 &= 0xfe; else bVar3 |= 1; *(byte *)(lVar1 + 0x582) = bVar3;`

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| en cours d'utilisation (bit 0, masque `0x01`, de l'octet `+0x582`) | 1 | aucune | aucune |

- **Largeur totale : 1 bit, constante.** `i27` et `i28` ecrivent deux bits du MEME octet
  `+0x582` : deux composants, un seul octet de drapeaux.
- Statut : **releve**.

## 12. `i29` — `device-exclusive-user-component`

- Descripteur `0x143d0c058` ; ecrivain **`FUN_14116fcb0`**.
- Desassemblage colle : `14116fcb8 MOV R8D, 0x1` ; `14116fcbe ADD RCX, 0x590` ;
  `14116fcc5 CALL 0x1408f0ac4`. Difference avec `i24` : pas de pre-remplissage `0xffffffff`
  (c'est `FUN_1408f0ac4` qui pose `0xffffffff` sur le chemin `P = 0`).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| porte P (`FUN_1406cf008` via `FUN_1408f0ac4`) | 1 | aucune | aucune |
| sonde S (`FUN_1406cf008` dans `FUN_1406d3140`, `cat == 1`) | 1 | `P == 1` | aucune |
| index de l'utilisateur exclusif (`R(W)`) | `W = ceil(log2(cardinal))` | `P == 1` | **OUI** : identique a `i24` (`S == 0` -> categorie 1, 13 au defaut = `IDLowBits` ; `S == 1` -> entree 4, 9) |
| generation du handle | 2 | `P == 1` | aucune |

- Valeur rendue -> `+0x594` (`gen << 30 | base + index`), handle resolu -> `+0x590`.
- **Largeur totale : 1 bit si `P = 0` ; sinon `4 + W` = 17 au defaut, 13 si `S = 1`.** Le
  soupcon de la note d'archetype (« probablement une reference d'entite ») est CONFIRME par
  l'ecrivain.
- Port existant : `consume1408f0ac4(br, 1)`.
- Statut : **releve** (grammaire complete ; `W` suit le profil du film).

---

## 13. Recapitulatif du groupe A

| Composant | Ecrivain | Grammaire | Total (bits) | Config | Statut |
|---|---|---|---|---|---|
| `i19` | `FUN_1410156e4` | `R(32)` + `Q(10; 0..10)` | 42 | aucune | releve |
| `i20` | `FUN_142f02d04` | `R(256)` brut | 256 | aucune | releve |
| `i21` | `FUN_1407f0678` | `R(32)` + `R(1)` P + [P : `R(8)`] | 33 / 41 | aucune | releve |
| `i22` | `FUN_14100d310` | `Q(14; 0..1)` | 14 | aucune | releve |
| `i23` | `FUN_14100d2d0` | `Q(14; 0..1)` | 14 | aucune | releve |
| `i24` | `FUN_141167910` | `R(1)` P + [P : `R(1)` S + `R(W)` + `R(2)`] | 1 / 4 + W | `W` = plage cat. 1 (`IDLowBits`) ou 9 | releve |
| `i25` | `FUN_142f02c54` | `Q(8; 0..60)` | 8 | aucune | releve |
| `i26` | `FUN_142f029e4` | `R(32)` + `R(32)` | 64 | aucune | releve |
| `i27` | `FUN_140bee524` | `R(1)` + `R(6)` | 7 | aucune | releve |
| `i28` | `FUN_142f02bec` | `R(1)` | 1 | aucune | releve |
| `i29` | `FUN_14116fcb0` | `R(1)` P + [P : `R(1)` S + `R(W)` + `R(2)`] | 1 / 4 + W | identique a `i24` | releve |

`Q(n; a..b)` = `FUN_1406d84b4` avec `bits = n`, `min = a`, `max = b`, `b6 = 0`, `b7 = 1`
(`N = 2^n`, brut `0` -> `a`, brut `N - 1` -> `b`, sinon
`a + (brut - 1 + 0,5) * (b - a) / (N - 2)` — **corrige apres verification independante** : la
forme `a + (brut + 0,5) * (b - a) / N` de la premiere redaction est celle de `b7 = 0`).

Onze sur onze releves, aucun `partiel`, aucun `non elucide`. Aucun appel virtuel, aucun
branchement sur un etat RUNTIME qui deciderait de la largeur (la seule dependance est la table
des plages de reference, deja modelisee par `varWidthBits`). Somme des largeurs constantes
(`i19`, `i20`, `i22`, `i23`, `i25`, `i26`, `i27`, `i28`) : 406 bits ; `i21` ajoute 33 ou 41 ;
`i24` et `i29` ajoutent chacun 1 ou `4 + W`.

## 14. Ce que le port demandera

1. **Aucune primitive nouvelle** : les sept lecteurs rencontres existent deja dans `filmdec`
   — `br.ReadBit()` (`FUN_1406cf008`), `br.ReadBits(32)` (`FUN_141015740`, `FUN_14080dec4`),
   `br.ReadBits(n)` par tranches de 64 (`FUN_1406d676c`, cf. `components_biped_ability.go:295`
   pour le `R(96)`), `consumeGateR(br, 8)` (`FUN_1407f08bc`), `consume1408f0ac4(br, 1)`
   (`FUN_1408f0ac4` categorie 1) et la lecture quantifiee `FUN_1406d84b4` (`ReadBits(w)` puis
   dequantification). La dequantification `Q` doit appliquer les bornes exactes aux deux
   extremites (`b7 = 1`) et, a l'interieur, `(brut - 1 + 0,5) * (max - min) / (N - 2)` —
   diviseur `N - 2` et code decale de 1, **pas** la convention `(q + 0,5) / N` de
   `dequantMidpoint` (**corrige apres verification independante**) — a verifier contre la
   formule existante du depot avant de la reutiliser : les deux ne coincident pas (les bornes
   de `ability_state_hooks.go:29` etaient « non etablies » ; ici elles le sont : 10,0 / 1,0 /
   1,0 / 60,0).
2. **Une seule entree de profil** : la largeur `W` de la categorie 1 pour `i24` et `i29`, deja
   portee par `varWidthBits(1)` sur `varWidthDefaultRange` (`varwidth.go:70-92`) et calibree par
   `FrameConfig.IDLowBits`. Le port doit passer par `consume1408f0ac4(br, 1)` et non par une
   largeur en dur : 13 n'est qu'un defaut (9 ou 11 mesures ailleurs, `frame_records.go:118`).
3. **`i20` : 256 bits a ranger tels quels** (32 octets). Ne pas interpreter avant qu'un
   ecrivain ou un lecteur en dise le type ; le decodeur n'a besoin que de la largeur pour
   fermer les records.
4. **Ordre** : les onze se lisent dans l'ordre des index de `ti=43` (`i19` puis `i20` ... `i29`),
   apres les 19 composants `object-*` / `device-position` deja portes et avant le groupe B
   (`i30` a `i40`), dont la grammaire reste a relever. Comme le plan le dit (D5 du 1.9.1 bis),
   porter le groupe A seul ne ferme aucun record : le gate reste la fermeture de `ti=43` a
   `4 106/4 106` apres les 22.
5. **`ecs_table.tsv`** : les onze lignes (`status`, `deser_addr`, `grammar`, `bits_typ`,
   `code_source`) se mettent a jour dans le commit qui porte le code, jamais avant
   (`ecs_table_guard_test.go`). Pour `i21`, `i24`, `i29`, `bits_typ` n'est pas un entier unique :
   distinguer largeur FIXE et NOMINALE comme la note de `i0` le demande deja.
6. **Rappel du `partiel` en amont** : `i2 object-forward-and-up-dynamic-precision-component`
   (mode `C = 1`, `FUN_142e29bac`) precede les `device-*` ; il peut redevenir le bloquant une
   fois les 22 portes — a lire au golden regenere, pas a supposer.
