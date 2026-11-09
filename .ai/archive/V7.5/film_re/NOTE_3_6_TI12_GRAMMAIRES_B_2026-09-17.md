# ti=12 (navpoint) — grammaires des ecrivains, groupe B : i13 a i25 et les 8 `visual-state-groups` (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « porter les composants
> manquants archetype par archetype »), sans production : aucun film decode, aucun test Go,
> aucun fichier de production touche. Source : `HaloInfinite.exe` par Ghidra en LECTURE SEULE
> (HTTP direct `127.0.0.1:8089`, programme `HaloInfinite.exe`, base `0x140000000` ;
> `decompile_function`, `disassemble_function`, `disassemble_bytes`, `read_memory`,
> `get_xrefs_to`, `search_strings`, `search_byte_patterns`). Methode :
> `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` ; note d'archetype : `NOTE_3_6_TI12_2026-09-16.md`.
>
> Instrument / Ghidra lecture seule.
>
> Archetype : **ti=12 `managed-navpoint-*`** (28 composants, 2 portes : `i0`, `i14`).
> Groupe : **ti12b**, indices **13 a 25** dont l'ecrivain est nomme (`i13`..`i19`, plus le
> recontrole d'`i14` deja porte), et le `[!]` de la note d'archetype : les **8 instances
> `visual-state-groups-component-0..7`** (`i20`..`i27`) dont la chaine de resolution
> s'arretait a la table de pointeurs de noms `0x143d07f00`. **Resolu** (§9).
>
> Convention : « ecrivain » designe ici, comme dans les notes soeurs, la fonction a
> `descripteur + 0x40` = la colonne `deser_addr` d'`ecs_table.tsv`. Le §1.1 montre que c'est le
> LECTEUR du flux (le sérialiseur symetrique est a `+0x28`) ; c'est bien la grammaire a porter.

---

## 0. Ce que la chaine des descripteurs a donne (tout colle de `read_memory`)

| i | Composant | Descripteur | `+0x10` (§1.1 : niveau) | `+0x28` serialiseur | `+0x40` lecteur | Objet statique (§1.2) |
|---|---|---|---|---|---|---|
| 13 | `managed-navpoint-top-progress` | `0x143d08160` | `0x14117b4a0` = `return 1` | — | **`FUN_142ed51d8`** | `0x144746ec0` |
| 14 | `managed-navpoint-radial-progress` (porte) | `0x143d08110` | `0x14117b4a0` = 1 | — | `FUN_140fc8d14` | `0x144746eb8` |
| 15 | `managed-navpoint-bottom-progress` | `0x143d080c0` | `0x14117b4a0` = 1 | — | **`FUN_142ed4fe4`** | `0x144746eb0` |
| 16 | `managed-navpoint-override-flags` | `0x143d08070` | `0x14117b4a0` = 1 | — | **`FUN_140ebf834`** -> `FUN_140ebf854` | `0x144746f28` |
| 17 | `managed-navpoint-object-marker` | `0x143d082a0` | `0x14117b4a0` = 1 | — | **`FUN_141169e68`** -> `FUN_14080dec4` | `0x144746f20` |
| 18 | `managed-navpoint-position-offset` | `0x143d08250` | `0x14117b4a0` = 1 | — | **`FUN_140f04f68`** | `0x144746f18` |
| 19 | `managed-navpoint-visual-states-component` | `0x143d08200` | **`0x141179610` = `return 2`** | `0x140467a20` = `ret` (0 bit) | **`FUN_142ed521c`** | `0x144746f10` |
| 20..27 | `managed-navpoint-visual-state-groups-component-0..7` | **`0x143d081b0`** (un seul) | `0x14117b4a0` = 1 | `FUN_142edb178` | **`FUN_140dbe1bc`** | `0x144714810 + 0x10*k` |

Descripteurs colles en entier (qwords) :

```
0x143d08200 0x141191ab0 | 0x143d08208 0x14076ced0 | 0x143d08210 0x141179610 | 0x143d08218 0x141177cc0 (nom)
0x143d08220 0x1404ab600 | 0x143d08228 0x140467a20 | 0x143d08230 0x1411c8f80 | 0x143d08238 0x14076ce9c
0x143d08240 0x142ed521c (lecteur) | 0x143d08248 0x1404ab600                                        (i19)

0x143d081b0 0x141191ab0 | 0x143d081b8 0x14076ced0 | 0x143d081c0 0x14117b4a0 | 0x143d081c8 0x14064c620 (nom INDEXE)
0x143d081d0 0x1404ab600 | 0x143d081d8 0x142edb178 | 0x143d081e0 0x1411c8f80 | 0x143d081e8 0x14076ce9c
0x143d081f0 0x140dbe1bc (lecteur) | 0x143d081f8 0x1404ab600                                        (vsg)
```

Les quatre constantes de la signature lues ici sont `0x1404ab600` (`+0x20`, `+0x48`) et
`0x1411c8f80` (`+0x30`) — la note de methode ecrit `0x14049b600` et `0x141c8f880` (deux
transpositions de chiffres ; la note soeur ti=11 a fait le meme constat). `+0x10` vaut
`0x14117b4a0` OU `0x141179610` : ce n'est pas une constante de famille, c'est le NIVEAU (§1.1).

## 1. Deux faits transverses etablis par ce releve

### 1.1 La vtable, c'est `descripteur + 0x10` ; `vtable[0]()` rend le niveau, et ce niveau est le `param_4` des lecteurs

Le dispatcheur de lecture `FUN_14076cb60` (celui de `HANDOFF_FRAME_DECODER_L3.md` l.58) fait,
desassemblage colle :

```
14076cc69: MOV RAX,qword ptr [RBX]        ; RAX = vptr de l'objet composant
14076cc6f: CALL qword ptr [RAX]           ; vtable[0](obj)
14076cc71: MOV R13D,EAX                   ; -> R13D
...
14076cd11: MOV dword ptr [RSP + 0x20],R13D ; 5e argument
14076cd19: CALL qword ptr [RAX + 0x28]    ; vtable+0x28(obj, flux, record, &out, R13D)
```

et le slot `descripteur + 0x38` (`0x14076ce9c`) est un thunk, octets `48 8b 01 44 8b 4c 24 28
48 ff 60 30` = `MOV RAX,[RCX] ; MOV R9D,[RSP+0x28] ; JMP qword ptr [RAX + 0x30]`. Si le vptr de
l'objet vaut `descripteur + 0x10`, alors `vptr+0x28` = ce thunk et `vptr+0x30` =
`descripteur + 0x40` = le lecteur, appele avec **`R9D` = `param_4` = `vtable[0]()`**. Le
dispatcheur d'ecriture `FUN_142e2d6d4` appelle `vptr+0x18` = `descripteur + 0x28` : pour les 8
groupes c'est `FUN_142edb178`, qui ECRIT (`FUN_1406d49c4(flux, flux, iVar2 != -1)` — cette
fonction fait `acc = acc*2 | bit ; +0x2c += 1` : c'est l'ecrivain d'UN bit, la note de methode la
dit « R(1) rendu », c'est a corriger).

Preuve que `vptr = descripteur + 0x10` : `search_byte_patterns` du qword `0x143d081c0`
(`c0 81 d0 43 01 00 00 00`) rend **8 adresses** `0x144714810, ..820, ..830, ..840, ..850, ..860,
..870, ..880` (pas de 0x10) ; celui de `0x143d08210` (i19) rend `0x144746f10`. Et
`FUN_140e43dc4`, seule fonction qui reference `0x144714810`, est l'enregistrement de
l'archetype : 20 appels `FUN_14064dd28(lVar1, index, &objet, -1)` pour `index = 0..0x13`, puis
une boucle `*piVar4 = iVar2 ; FUN_14064dd28(lVar1, iVar2 + 0x14, ppuVar3, -1)` pour
`iVar2 = 0..7` sur les 8 objets (elle ECRIT l'index d'instance a `objet + 8`, ce qui explique
les `0` lus en statique), et finit par `*(param_1 + 0x4754) = 0xc` (= ti 12). Les 20 objets lus
(`0x144746e90..0x144746f2f`) portent chacun `vptr = descripteur_i + 0x10` : `i13 -> 0x143d08170`,
`i14 -> 0x143d08120`, `i15 -> 0x143d080d0`, `i16 -> 0x143d08080`, `i17 -> 0x143d082b0`,
`i18 -> 0x143d08260`, `i19 -> 0x143d08210` (et `i0 -> 0x143d07ea8` ... `i12 -> 0x143d08440`).
**L'ordre d'enregistrement EST l'index de composant du registre.**

Preuve que `vtable[0]()` est le niveau ET le `param_4` mesure. La chaine des quatre pas a ete
rejouee sur quatre composants `ti=35` dont le depot a MESURE `param_4` en direct
(`filmdec/component_param4.go`, table `paramByComponent`, 464 010 mesures) :

| Composant | Chaine | Accesseur | Slot | Descripteur | `+0x10` decompile | Mesure depot | `level` `ecs_table.tsv` |
|---|---|---|---|---|---|---|---|
| `object-maximum-vitalities-component` | `0x143c99380` | `0x14064c6b0` | `0x143d0b908` | `0x143d0b8f0` | `0x14117e0e0` : `return 3` | 3 | 3 |
| `object-frame-configuration-component` | `0x143c991a0` | `0x14064c670` | `0x143d0be88` | `0x143d0be70` | `0x1405f0ac0` : `return 0` | 0 | 0 |
| `unit-malleable-property-component` | `0x143c961e0` | `0x141175670` | `0x143d06ce8` | `0x143d06cd0` | `0x140c85020` : `return 4` | 4 | 4 |
| `object-low-frequency-component` | `0x143c99340` | `0x14064c690` | `0x143d0bf30` | `0x143d0bf18` | `0x141179610` : `return 2` | 2 | 2 |

Quatre sur quatre, plus `ti=11 i4` (`0x141179610` = 2, `level = 2`, note soeur ti=11) et, ici,
`ti=12 i19` (`0x141179610` = 2, `level = 2`) et `i20..i27` (`0x14117b4a0` = 1, `level = 1`).
Trois sources concordent : la constante de l'EXE, la colonne `level` du registre (lue dans le
film en `entree + 0x100`, `filmdec/registry.go:79`) et la capture live. Consequence pour le
port : **`param_4` d'un lecteur = `Archetype.Level(i)` du film** (le film est autoportant :
memoire du 16/09) ; la constante `vtable[0]()` n'est que la valeur que le build courant ECRIT.
La « valeur scalaire 2 non prouvee au binaire » de `HANDOFF_FRAME_DECODER_L3.md` l.230 est
prouvee ici.

Piege connexe : le TSV `.ai/V7.5/replay2d/registre_film/lotC/000d5950_delta_masques.tsv`
(17/08) porte une colonne `niveau` **decalee d'un index** (`i19 = 1`, `i20 = 2`, `i0 = 0`) —
c'est l'ancien cadrage `slot+4` corrige au lot 1.2 (`registry.go`, commentaire de `Levels`). Ne
pas y lire le niveau d'`i19`.

### 1.2 Le bloc « groupe d'etat visuel » (0x130 octets), partage avec `ti=11 i4`

`i19` et les 8 `visual-state-groups` lisent le MEME bloc de `0x130` octets, range dans le
composant navpoint a `+0x728 + k*0x130` (`k = 0..7`). Son en-tete (`FUN_140dbe400` : masque
R(4) a `+0x100`, scalaire a `+0x104`, 4 fentes de `0x40` octets a `+0x00/+0x40/+0x80/+0xc0`
chacune avec une etiquette R(4) et un corps a 15 etiquettes par `FUN_141e98e10` ->
`FUN_141e98c70` -> `FUN_141e98f90`) est **identique a `ti=11 i4`** et entierement releve dans
`NOTE_3_6_TI11_GRAMMAIRES_2026-09-17.md` §1.3 et §2 (table etiquette -> vtable -> corps,
`w0` = largeur d'une reference d'entite du domaine 0). Cette note ne le recopie pas ; elle
releve ce que `ti=12` AJOUTE autour : un prefixe (`FUN_140dbe218`) et une queue
(`FUN_140dbe25c` apres l'en-tete), tous deux gouvernes par un `mode` (`m`) passe en `R8D`.

`FUN_140dbe218(groupe, flux, m)` — desassemblage colle : `140dbe227 MOV EBX,R8D ;
140dbe22d LEA R8,[RCX + 0x128] ; 140dbe23a CALL 0x14080dec4 ; 140dbe23f MOV R9D,EBX ;
140dbe257 JMP 0x140dbe25c`. `FUN_140dbe25c(groupe, flux, ., m)` : `140dbe284 CALL 0x140dbe400`
(avec `R8D = m`), `140dbe289 LEA R8,[RDI + 0x108] ; 140dbe293 CALL 0x14080dec4`, boucle sur les
4 bits de `[RDI + 0x100]` : `140dbe34f LEA R8,[RDI + 0x10c] ; 140dbe359 ADD R8,R14 ;
140dbe35c CALL 0x14080dec4 ; 140dbe361 TEST R12D,R12D ; 140dbe364 JZ 0x142414d5a` (bloc
deporte = le `if (param_4 == 0) { R(4) -> +0x11c + b }` du decompile), puis `iVar2 =
(param_4 != 0) + 2` lu `iVar13` fois vers `+0x120 + k`, et `if (iVar13 < 4) *(+0x120 + iVar13)
= 0xff` (terminateur, 0 bit). Dans `FUN_140dbe400` : `iVar5 = (-(uint)(param_3 != 0) &
0xffffffe1) + 0x20` = **R(1) si `m != 0`, R(32) si `m == 0`**.

Grammaire d'un groupe, mode `m` (`nb` = nombre de bits poses dans le masque R(4)) :

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | identifiant du groupe (`+0x128`) | R(32) — `FUN_14080dec4` : `+0x2c] + 0x20` | toujours | aucune |
| 2 | masque des 4 fentes (`+0x100`) | R(4) — `FUN_140dbe598` : `+0x2c] + 4` | toujours | aucune |
| 3 | scalaire (`+0x104`) | **R(1) si `m != 0`, R(32) si `m == 0`** | toujours | `m` (voir §8/§9 : constante du site d'appel) |
| 4 | pour chaque fente `b` posee : etiquette | R(4) | bit `b` du masque | aucune |
| 5 | corps de la fente (etiquettes `0..0xe`, `0xf` = flux invalide) | voir note ti=11 §2 : `0` -> 0 ; `1` -> 1+1 ; `2` -> 1+32 ; `3`/`7` -> 1+4 ; `4` -> 1+9 ; `5` -> 1+3+n x (1 ou 14), n <= 7 ; `6` -> 1+4+n x (1 ou 3+w0), n <= 15 ; `8` -> 1+8 ; `9` -> 1+32 ; `0xa` -> 1+4+4+32 ; `0xb` -> 1+(1 ou 6) ; `0xc`/`0xd` -> 1+1+[w0+2]+32 ; `0xe` -> 1+1+[w0+2]+32+1 (le premier `1` est le bit commun `+0x08`, absent pour `0`) | bit `b` du masque | **`w0`** pour `6`, `0xc`, `0xd`, `0xe` = `FUN_1406d310c(cardinal du domaine 0)` (entree de profil, note ti=11 §3.2) |
| 6 | second identifiant (`+0x108`) | R(32) | toujours | aucune |
| 7 | pour chaque fente `b` posee : identifiant de fente (`+0x10c + 4b`) | R(32) | bit `b` du masque | aucune |
| 8 | pour chaque fente `b` posee : petit champ (`+0x11c + b`) | **R(4) si `m == 0`, rien sinon** | bit `b` du masque | `m` |
| 9 | `nb` fois : ordre (`+0x120 + k`) | **R(2) si `m == 0`, R(3) sinon** | `nb` fois | `m` |
| 10 | terminateur `0xff` | 0 bit | `nb < 4` | — |

Totaux : mode 1 = `69 + somme_b (39 + corps_b)` ; mode 0 = `100 + somme_b (42 + corps_b)`.
Masque vide : **69 bits** (mode 1), **100 bits** (mode 0). Les lecteurs des etiquettes sont ceux
de la note ti=11 ; recontroles ici par decompile : `FUN_141e9d6d0` R(1), `FUN_141e9d120` 32 x R(1),
`0x141e9d660 -> FUN_1407ef804` R(4) (`*param_3 = bVar4 - 1`), `FUN_141e9d670` R(9), `FUN_141e9a9c0`
R(3) + n x (`0x141e9ca50 -> FUN_142b67f08` : `FUN_1406cf008` R(1) ; si 0 `+0x2c] + 0xd`),
`FUN_141e9aa60` R(4) + n x (`FUN_141e9c9c0` : R(1) ; si 1 `FUN_1406d3140` + `FUN_1405d5dbc` 0 bit),
`0x141e9d0a0 -> FUN_140968284` R(4), `0x141e9d0b0 -> FUN_14109414c` R(8), `FUN_141e9d0c0` R(32),
`FUN_141e9d5e0` R(4)+R(4)+R(32), `FUN_141e9d1a0 -> FUN_1407f2058`, `FUN_141e9d440` /
`FUN_141e9cf50` / `FUN_141e9d260` R(1)[+ref]+R(32)[+R(1)]. Aucune divergence avec la note ti=11.

## 2. i13 `managed-navpoint-top-progress` — ecrivain `FUN_142ed51d8`

```c
lVar1 = *(longlong *)(param_3 + 0x10);
uVar2 = FUN_1406d84b4(param_2,param_2,DAT_143cd84ec,DAT_143cd8374,8,1,1);
*(undefined4 *)(lVar1 + 0x700) = uVar2;
return 1;
```

`DAT_143cd84ec` = `00 00 80 bf` = **-1,0f** ; `DAT_143cd8374` = `00 00 80 3f` = **+1,0f** ;
5e argument `[RSP+0x28] = 8` = largeur (`1406d84d0 MOVSXD RBX,dword ptr [RSP + 0x28]` puis
`1406d84f1 ADD dword ptr [R11 + 0x2c],EBX`) ; 6e `[RSP+0x30] = 1` ; 7e `[RSP+0x38] = 1`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| progression haute, quantum `q` (`+0x700`) | **R(8)** | toujours | aucune (largeur et bornes = litteraux) |

**Dequantification EXACTE de `FUN_1406d84b4(flux, flux, min, max, 8, 1, 1)`** (desassemblage
colle, `1406d8511..1406d85dc`) : `EDI = 1 << 8 = 256` ; `1406d8518 LEA ECX,[RDI-1] ;
1406d851b CMOVZ ECX,EDI` -> 6e arg = 1 donc `ECX = 255` ; 7e arg = 1 -> chemin `1406d8577` :
`q == 0` -> `min` ; `q == ECX-1 = 254` -> `max` ; sinon `pas = (max-min)/(ECX-2 = 253)`,
`valeur = min + (q-1)*pas + pas*0,5` (`0x143cd84b0` = `00 00 00 3f` = 0,5f) ; puis
`1406d8558 DEC ECX ; 1406d855a LEA EAX,[R9+R9] ; CMP EAX,ECX ; JZ` -> `2q == 254`, soit
`q == 127` -> `valeur = (min+max)*0,5` (double `0x143cd8910` = `3fe0000000000000` = 0,5).
Sur `[-1, +1]` : `0 -> -1,0` ; `254 -> +1,0` ; `127 -> 0,0` exactement ; `1..253 -> -1 + (q-1)*2/253 + 1/253`.
Le port actuel d'`i14` (`components_managed_object.go`, `NavpointRadialProgressValue ->
dequantMidpoint(q, 8, -1, 1)` = `min + (q+0,5)*(max-min)/256`) est une « convention RETENUE »
selon son propre commentaire : l'ecrivain dit autre chose. **A corriger par le lot qui porte
`i13`/`i15`, en une seule fonction pour les trois** (constat, pas corrige ici : hors perimetre).

Largeur totale : **8 bits, constante**. Statut : **releve**.

## 3. i14 `managed-navpoint-radial-progress` (porte) — recontrole de `FUN_140fc8d14`

Corps identique a `i13` au decalage pres : `FUN_1406d84b4(param_2,param_2,DAT_143cd84ec,
DAT_143cd8374,8,1,1)` -> `*(lVar1 + 0x704)`. **R(8)**, memes bornes, meme dequantification
(§2). Concorde avec `ecs_table.tsv` (`deser_addr = FUN_140fc8d14`, `R(8) dequantifie dans [-1,1]`).
Statut : **releve** (deja porte ; seule la dequantification est a aligner, §2).

## 4. i15 `managed-navpoint-bottom-progress` — ecrivain `FUN_142ed4fe4`

Corps identique : `FUN_1406d84b4(param_2,param_2,DAT_143cd84ec,DAT_143cd8374,8,1,1)` ->
`*(lVar1 + 0x708)`. Les trois progressions occupent `+0x700` (haute), `+0x704` (radiale),
`+0x708` (basse).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| progression basse, quantum `q` (`+0x708`) | **R(8)** | toujours | aucune |

Largeur totale : **8 bits, constante**. Statut : **releve**.

## 5. i16 `managed-navpoint-override-flags` — ecrivain `FUN_140ebf834`

```c
FUN_140ebf854(param_2,param_2,*(longlong *)(param_3 + 0x10) + 0x70c);
return 1;
```

`FUN_140ebf854` : chemin rapide `*(int *)(param_1 + 0x2c) + 5 ; << 5 ; uVar4 >> 0x1b`, chemin
lent `uVar7 = iVar1 - 0x3b ; +0x2c] + 5` -> un seul champ.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| drapeaux de surcharge (`+0x70c`, `uint32`) | **R(5)** | toujours | aucune |

Largeur totale : **5 bits, constante**. Statut : **releve**.

## 6. i17 `managed-navpoint-object-marker` — ecrivain `FUN_141169e68`

```c
FUN_14080dec4(param_2,"navpoint-object-marker",*(longlong *)(param_3 + 0x10) + 0x710);
return 1;
```

`FUN_14080dec4` : `+0x2c] + 0x20` sur les deux chemins ; la chaine `"navpoint-object-marker"`
(`0x14374d1d0`) est un libelle de debogage passe en `param_2`, jamais lu.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| marqueur d'objet (`+0x710`, `uint32`) | **R(32)** | toujours | aucune |

Largeur totale : **32 bits, constante**. Statut : **releve**.

## 7. i18 `managed-navpoint-position-offset` — ecrivain `FUN_140f04f68`

```c
lVar1 = *(longlong *)(param_3 + 0x10);
cVar2 = FUN_14076f91c();
if (cVar2 == '\0') { FUN_14076e524(&local_18,param_2,local_res18,0x10); }
else               { FUN_1411b259c(&local_18); }
*(undefined8 *)(lVar1 + 0x714) = local_18;  *(undefined4 *)(lVar1 + 0x71c) = local_10;
```

Trois flottants (`+0x714`, `+0x718`, `+0x71c`). La porte `FUN_14076f91c` ne lit AUCUN bit :
`uVar1 = 1; if ((DAT_144e61ea0 == '\0') && (DAT_145121140 != '\x01')) uVar1 = 0;`.
`DAT_144e61ea0` n'est mis a 1 que pendant les boucles de serialisation de verification
(`FUN_142e2d6d4` : `DAT_144e61ea0 = 1 ; ... ; DAT_144e61ea0 = 0`, idem `FUN_142e2d08c`,
`FUN_142e2bfd0`, `FUN_142e2c690`, `FUN_142e309b4`, `FUN_142e30b9c`, `FUN_142e31a0c`,
`FUN_142e31bf8`) ; `DAT_145121140` est l'etat 0..3 d'un sous-systeme (`FUN_140a93ec8(mode)` :
`mode 1 -> FUN_142b5c658 -> DAT_145121140 = 1`, `mode 2 -> 2`, sinon `3`, `0 -> 0`). Le depot a
deja tranche cette porte pour `i0` : elle rend 0 sur les films reels
(`NOTE_I0_TI41_POSITION_PROJECTILE.md` §2.1, `HANDOFF_FILMDEC_REP_FIX.md` l.47 « ecartee par
sweep »). Le chemin `FUN_1411b259c` = `FUN_1406d676c(flux, flux, dst, 0x60)` = **R(96)** brut
(3 x float32) n'est donc pas a porter ; il est consigne pour memoire.

Chemin normal `FUN_14076e524(&out, flux, &idx, L = 0x10)` — decompile colle :

```c
cVar6 = FUN_1406cf008(param_2);                       // R(1)  porte g
uVar7 = 0xffffffff;
if (cVar6 == '\0') {
  ... R(DAT_144632be0) -> uVar7                         // index de region, largeur RUNTIME
  if (uVar7 != 0xffffffff) {
    pfVar13 = (float *)(&DAT_14462cbe0 + (longlong)(int)uVar7 * 3);        // 3 paires (min,max) de la region
    lVar12 = (longlong)(int)uVar7 * 0x20 + lVar12;                          // lVar12 = L = 0x10
    local_48 = *(undefined8 *)(&DAT_1445ccbe0 + lVar12 * 0xc); ...           // 3 largeurs [region][L]
    goto LAB_14076e5f3;
  }
}
pfVar13 = (float *)&DAT_1445cc9c8;                                          // bornes PAR DEFAUT
local_48 = *(undefined8 *)(&DAT_1445cc9e0 + lVar12 * 0xc); ...               // 3 largeurs [defaut][L]
LAB_14076e5f3:
FUN_140cc5128(param_2, ., &local_38, &local_48);                            // 3 x R(largeur_i)
do { fVar14 = (pfVar13[1] - *pfVar13) / (float)(1 << (largeur_i & 0x1f));
     *(float *)(uVar10 + param_1) = (float)q_i * fVar14 + min_i + fVar14 * DAT_143cd84b0;   // 0,5f
} while (uVar10 < 0xc);
```

`FUN_140cc5128` : `lVar8 = 3 ; do { iVar1 = *(int *)param_4 ; ... +0x2c] + iVar1 ; param_4 += 4 }
while (--lVar8)` = trois lectures de largeurs distinctes. Valeurs lues en statique :
`DAT_1445cc9c8` = `c69c4000 469c4000` x 3 = **-20000,0f / +20000,0f** sur les trois axes ;
`DAT_1445cc9e0 + 0x10*0xc` (= `0x1445ccaa0`) = `00 00 00 00` x 3 ; `DAT_144632be0` = 0. Ces
largeurs sont **remplies au chargement de la carte** : `FUN_140be9a14` (`DAT_144632be0 = 1` ou
`= FUN_1406d310c(...)` = `ceil(log2(nb de regions))` ; `FUN_140be9b88(niveau, bornes)` pour les
regions ET pour la table par defaut `&DAT_1445cc9c8`, `iVar5 < 0x20`). C'est le mur documente du
depot (`HANDOFF_FRAME_DECODER_L3.md` l.432-434 : `W = min(26, bitLen(ceil(etendue / (2*pas(L)))))`,
`pas(L) = 2^(16-L)/120`), deja traite pour `i0` : table PAR REGION au niveau 16 batie sur
l'AABB de la carte, lue dans le film (`DetectI0Layout`, `map_bounds.go`, `registry.go`
commentaire de `Levels`). **`i18` est le MEME bloc, au MEME niveau `L = 0x10`, SANS la porte
externe d'`i0`** (`FUN_14076e420` n'intervient pas ici).

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 0 | (porte runtime `FUN_14076f91c`) | 0 bit | — | globales, 0 sur les films : non porte |
| 1 | porte `g` de region | **R(1)** — `FUN_1406cf008` | toujours | aucune |
| 2 | index de region `idx` | **R(IndexW)** | `g == 0` | **OUI** : `IndexW = DAT_144632be0` = `ceil(log2(nb de regions de la carte))` ; le depot modelise 1 |
| 3 | trois quanta `q_x, q_y, q_z` | **R(W_x) + R(W_y) + R(W_z)** — `FUN_140cc5128` | toujours | **OUI** : `g == 0` -> largeurs `[region idx][L=16]` = celles d'`i0` pour cette carte (`I0Layout`) ; `g == 1` -> largeurs `[defaut][L=16]` de la table `DAT_1445cc9e0`, derivees des bornes +-20000 par `FUN_140be9b88` : **non lues en statique** |
| — | dequantification | 0 bit | — | `v_i = min_i + q_i * pas_i + pas_i/2`, `pas_i = (max_i - min_i) / 2^W_i` (convention du milieu, `0x143cd84b0` = 0,5f) |

Largeur totale : **1 + [IndexW] + W_x + W_y + W_z**, non constante (carte). Statut :
**partiel** — la structure est entierement lue et les largeurs du chemin `g == 0` sont celles
que le depot sait deja lire pour `i0` ; mais les largeurs du chemin `g == 1` (table par defaut,
remplie au chargement) ne sont ni lues dans le binaire ni etablies par le depot (le lecteur
generique `consumeQuantVec3Values` lit les trois axes avec la MEME largeur dans les deux
branches, ce qui n'est pas ce que dit l'ecrivain). A confronter au port, pas ici.

## 8. i19 `managed-navpoint-visual-states-component` — ecrivain `FUN_142ed521c`

```
142ed5238: CMP R9D,0x1
142ed523c: JA 0x142ed5294            ; param_4 > 1 (non signe) -> rien a lire, return 1
142ed523e: MOV RDI,qword ptr [R8 + 0x10]
142ed5245: LEA R8,[RDI + 0x720]
142ed524c: CALL 0x14109414c          ; R(8) -> +0x720 (masque des 8 groupes)
142ed5261: MOV EAX,dword ptr [RDI + 0x720]
142ed5267: BT EAX,EBX ; JNC 0x142ed527c
142ed526c: XOR R8D,R8D               ; m = 0
142ed5275: CALL 0x140dbe218          ; groupe k (RBP = +0x728 + k*0x130), flux, m = 0
142ed527c: OR dword ptr [RSI],0xffffffff   ; sinon +0x128 du groupe k = -1 (RSI = +0x850 + k*0x130)
142ed5281: ADD RBP,0x130 ; ADD RSI,0x130 ; CMP EBX,0x8 ; JL
```

`param_4` = le niveau du composant (§1.1). **Sur le build courant `vtable[0]()` =
`0x141179610` = `return 2`, et `ecs_table.tsv` donne `level = 2` : `i19` lit 0 bit.** Le
serialiseur `+0x28 = 0x140467a20` est un `ret` nu (0 bit ecrit) — les deux faces concordent.
La grammaire ci-dessous est celle des films de niveau `<= 1` (anciens builds), et le port doit
la brancher sur `Archetype.Level(19)` du film, pas sur la constante.

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 0 | (rien) | 0 | `niveau >= 2` (`CMP R9D,1 ; JA`) | **OUI : niveau du registre du film** (2 sur le build courant) |
| 1 | masque des 8 groupes (`+0x720`) | **R(8)** — `FUN_14109414c` | `niveau <= 1` | — |
| 2 | pour `k = 0..7`, si bit `k` : groupe `k` en **mode 0** (§1.2 : `100 + somme_b (42 + corps_b)`) | variable | bit `k` du masque | `w0` selon les etiquettes |

Largeur totale : **0 bit, constante, pour `niveau >= 2`** ; `8 + somme_k groupe_k(m=0)` sinon.
Statut : **releve**. (Le recensement `000d5950_delta_masques.tsv` annonce `i19` 104 fois avec
0,88 % des records : compatible avec un bit de masque pose et une charge utile de 0 bit.)

## 9. i20..i27 `managed-navpoint-visual-state-groups-component-0..7` — le `[!]` resolu, ecrivain commun `FUN_140dbe1bc`

Le pas 3 de la methode s'arretait a la table `0x143d07f00`. Voie suivie : `get_xrefs_to
0x143d07f00` -> `14064c624 [DATA]`, `14064c62b [DATA]`, deux instructions d'une routine que
Ghidra ne connait pas comme fonction. Octets lus a `0x14064c620` : `48 63 41 08 48 8d 0d d5 b8
6b 03 48 8b 04 c1 c3` = `MOVSXD RAX,dword ptr [RCX+8] ; LEA RCX,[0x143d07f00] ; MOV RAX,[RCX +
RAX*8] ; RET` : **un accesseur de nom INDEXE**, `nom(this) = table[this->index]`, l'index etant
a `objet + 8` (celui que `FUN_140e43dc4` ecrit, §1.1). `get_xrefs_to 0x14064c620` ->
`143d081c8 [DATA]` = un slot de nom, d'ou **descripteur `0x143d081b0`** (§0) et lecteur
`+0x40` = **`FUN_140dbe1bc`**. La table, colle : `table[0] = 0x143d05cc8` = `...-0`,
`table[1] = 0x143d05be8` = `-1`, `table[2] = 0x143d05c20` = `-2`, `table[3] = 0x143d05c58` =
`-3`, `table[4] = 0x143d05c90` = `-4`, `table[5] = 0x143d05d00` = `-5`, `table[6] = 0x143d05d38`
= `-6`, `table[7] = 0x143d05d70` = `-7` (chaines lues) : l'index d'instance `k` EST le suffixe,
et l'enregistrement le place a l'index de registre `0x14 + k` = `i20 + k`.

```c
undefined8 FUN_140dbe1bc(longlong param_1,undefined8 param_2,longlong param_3)
{
  lVar1 = *(longlong *)(param_3 + 0x10);
  lVar3 = (longlong)*(int *)(param_1 + 8) * 0x130;          // k = index d'instance (objet + 8)
  cVar2 = FUN_1406cf008(param_2);                            // R(1)
  if (cVar2 == '\0') { *(undefined4 *)(lVar3 + 0x850 + lVar1) = 0xffffffff; }   // groupe k absent
  else               { FUN_140dbe218(lVar1 + 0x728 + lVar3,param_2,1); }        // groupe k, mode 1
  return 1;
}
```

Face ecrivain symetrique (`+0x28 = FUN_142edb178`) : `iVar2 = *(int *)(lVar1 + 0x850) ;
FUN_1406d49c4(param_2,param_2,iVar2 != -1) ; if (iVar2 != -1) { FUN_141d12268(param_2) ;
FUN_142c94dd4(lVar1 + 0x728,param_2) }` — un bit de presence puis le groupe : concordant.

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | presence du groupe `k` | **R(1)** — `FUN_1406cf008` | toujours | aucune |
| 2 | groupe `k` en **mode 1** (§1.2 : `69 + somme_b (39 + corps_b)`) | variable | presence = 1 | `w0` selon les etiquettes |

Largeur totale : **1 bit** (absent) ; **70 bits** (present, masque vide) ; sinon
`70 + somme_b (39 + corps_b)`. Non constante. Statut : **releve**, pour les 8 instances a la
fois — UN lecteur, parametre par `k` (adresse cible `+0x728 + k*0x130` ; la grammaire sur le fil
ne depend pas de `k`). `vtable[0]()` = `0x14117b4a0` = 1 = `level` d'`ecs_table.tsv` pour
`i20..i27` ; `FUN_140dbe1bc` ne lit pas `param_4` (le mode 1 est un litteral du site d'appel).

## 10. Ce que le port demandera

1. **Un seul bloc « groupe » pour `ti=11 i4`, `ti=12 i19` et `ti=12 i20..i27`** : l'en-tete
   (`FUN_140dbe400`, 15 etiquettes, note ti=11 §2) est commun ; `ti=12` y ajoute le prefixe
   R(32) de `FUN_140dbe218` et la queue de `FUN_140dbe25c` (R(32) + par fente R(32)[+R(4)] +
   `nb` x R(2|3)), gouvernes par le mode `m` (0 pour `i19`, 1 pour `i20..i27`) — deux
   parametres (`m`, et la presence ou non du prefixe/queue), pas trois copies.
2. **`param_4` = `Archetype.Level(i)` du film**, jamais une constante : `i19` lit 0 bit au
   niveau 2 (build courant), R(8) + groupes en mode 0 au niveau `<= 1`. La constante
   `vtable[0]()` du binaire sert de VERIFICATION du niveau ecrit par le build courant, et la
   table `paramByComponent` du depot peut etre remplacee par la lecture du registre pour tout
   composant dont le descripteur est connu (preuve 4/4 sur `ti=35`, §1.1).
3. **`w0`** (etiquettes `6`, `0xc`, `0xd`, `0xe`) = `ceil(log2(cardinal du domaine 0))` : entree
   de profil, deja identifiee par `NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md`.
4. **`i18`** = le lecteur quantifie d'`i0` SANS sa porte externe, au niveau `L = 16` : largeurs
   par region de la carte (`I0Layout` / `MapQuantCatalog`), `IndexW` de profil ; la porte runtime
   `FUN_14076f91c` ne se porte pas (0 sur les films) ; la branche `g == 1` (table par defaut)
   reste a etablir — la marquer et mesurer au port (DesyncAt), pas la supposer.
5. **`i13`/`i15`** = `i14` : un seul lecteur R(8) et UNE dequantification exacte (§2 : 253 pas,
   saturations aux codes 0 et 254, milieu exact au code 127) ; le `dequantMidpoint` d'`i14`
   est a aligner dans le meme lot.
6. **`i16`** R(5), **`i17`** R(32) : triviaux.
7. Ordre le moins cher : `i13`, `i15`, `i16`, `i17` (largeurs constantes, 53 bits a eux
   quatre), puis `i19` (0 bit sur le build courant : un `case` qui lit le niveau), puis
   `i20..i27` (le bloc groupe partage avec `ti=11 i4`), enfin `i18` (dependance carte).
   L'oracle reste le ratchet 0.A.3 : le bloquant nomme doit avancer d'un index a chaque port.
8. A corriger dans les documents, dans le commit qui porte : note de methode §2 (constantes
   `0x1404ab600` / `0x1411c8f80` ; `+0x10` = niveau, pas constante de famille), §4
   (`FUN_1406d49c4` = ecrivain d'un bit), §6 (« la chaine donne l'ECRIVAIN, pas le LECTEUR » :
   c'est l'inverse, `+0x40` est le lecteur — la symetrie tient) ; `HANDOFF_FRAME_DECODER_L3.md`
   l.230 (la provenance de `param_4` est prouvee au binaire).

## 11. Appels Ghidra en echec ou inutilisables (15)

- `get_function_by_address 0x14064c624` : « No function found » (attendu : accesseur de nom
  non defini comme fonction ; lu par `read_memory` et decode a la main).
- `decompile_function` « No function found » : `0x14076ced0` (slot `+0x08`, octets lus :
  `33 c0 48 89 02 48 89 42 08 48 8b c2 c3` = rend une structure de 16 octets a zero),
  `0x142edb0dc` et `0x142edae04` (serialiseurs `+0x28` d'`i0`/`i4`, non necessaires),
  `0x141e9d660`, `0x141e9ca50`, `0x141e9d0a0`, `0x141e9d0b0` (thunks des etiquettes 3, 5, 7, 8 :
  resolus par `disassemble_bytes`, cibles `FUN_1407ef804`, `FUN_142b67f08`, `FUN_140968284`,
  `FUN_14109414c`) — 7 appels.
- `disassemble_function 0x140dbe1bc` : listing de 178 Mo (le serveur a deroule bien au-dela de
  la fonction) ; ecarte, le decompile a suffi — 1 appel.
- `read_memory` avec adresse mal formee (decimal au lieu d'hexa, erreur de script) sur les 6
  vtables des etiquettes `6..0xb` : rejoue correctement ensuite — 6 appels.
