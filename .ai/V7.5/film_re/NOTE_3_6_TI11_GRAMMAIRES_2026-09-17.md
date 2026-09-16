# ti=11 (objectif) — les corps du filtre d'interaction, releves un par un (2026-09-17)

> Preparation du lot 3.6, item 3.6.b, seconde moitie (la premiere : `NOTE_3_6_TI11_2026-09-16.md`).
> Sans production : aucun film decode, aucun test Go, aucun fichier de production touche.
> Source : `HaloInfinite.exe` par Ghidra en LECTURE SEULE (HTTP direct `127.0.0.1:8089`,
> `decompile_function`, `disassemble_function`, `read_memory`, `get_xrefs_to`, `search_strings`).
> Methode : `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md`.
>
> Instrument / Ghidra lecture seule. Archetype `ti=11` (managed-objective). Groupe `ti11`.

---

## 0. Ce que cette note change par rapport a celle du 16/09

La note du 16/09 relevait une table d'etiquettes a **6 cas** (`case 0..5` du `switch` de
`FUN_141e99630`) et laissait **une inconnue** : les corps derriere l'appel virtuel
`vtable[+0x08]`. Les deux points sont revises ici, sur pieces :

1. **Il y a 15 etiquettes, pas 6.** Le `default` de `FUN_141e99630` n'est pas un `assert` : il
   saute vers `FUN_141e99150` (etiquettes `6..0xb`), dont le `default` saute vers
   `FUN_141e99a70` (etiquettes `0xc`, `0xd`, `0xe`), et c'est seulement `param_1 != 0xe` dans
   cette derniere qui appelle `FUN_1411c8f80` (assertion, « Subroutine does not return »).
   L'etiquette `0xf` est donc la seule valeur invalide d'un champ de 4 bits.
2. **Les 14 corps sont releves.** La cle etait le LECTEUR symetrique : la fabrique
   `FUN_141e98e10` -> `FUN_141e98c70` -> `FUN_141e98f90` pose, pour chaque etiquette, le
   pointeur de vtable dans la fente (`*plVar4 = &PTR_FUN_...`), ce qui donne la table
   etiquette -> vtable, puis `vtable+0x08` = ecrivain et `vtable+0x10` = lecteur.

Consequence pour le depot : `TI11_SPEC_10_FEUILLES.md` (§ table, ligne i4) qui affirme
« sel6..15 assert » et « sel5 R(3)count + recursion » est **faux sur ces deux points** ; la
grammaire correcte est au §2.

## 1. Le composant `i4 managed-objective-interaction-filter-component`

### 1.1 Chaine des quatre pas (collee)

| Pas | Adresse | Contenu |
|---|---|---|
| chaine | `0x143c95338` | `managed-objective-interaction-filter-component` (`search_strings`, 1 seule occurrence) |
| accesseur de nom | `0x141177f90` | seule reference a la chaine |
| slot du nom (`descripteur + 0x18`) | `0x143d090d0` | seule reference a l'accesseur |
| descripteur | `0x143d090b8` | 10 slots, `read_memory` ci-dessous |

Descripteur `0x143d090b8` (qwords colles) :

```
0x143d090b8 0x141191ab0
0x143d090c0 0x14076ced0
0x143d090c8 0x141179610
0x143d090d0 0x141177f90   <- accesseur de nom (+0x18)
0x143d090d8 0x1404ab600
0x143d090e0 0x142edb5cc   <- +0x28 : ECRIVAIN (thunk -> FUN_142c7023c)
0x143d090e8 0x1411c8f80
0x143d090f0 0x14076ce9c
0x143d090f8 0x140dbe170   <- +0x40 : LECTEUR (wrapper -> FUN_140dbe400)
0x143d09100 0x1404ab600
```

**Calibration de famille (a retenir pour tout `ti=11`)** : la signature de ce descripteur
n'est pas celle du §2 de la note de methode (`+0x10 = 0x141179610` au lieu de `0x14117b4a0`,
`+0x20`/`+0x48 = 0x1404ab600` au lieu de `0x14049b600`, `+0x30 = 0x1411c8f80` au lieu de
`0x141c8f880`). Dans cette famille, **`+0x28` est l'ecrivain et `+0x40` est le lecteur**, les
deux verifies par decompile :

- `0x142edb5cc` : octets `49 8b 48 30 48 83 c1 48 e9 63 4c d9 ff` = `MOV RCX,[R8+0x30] ; ADD RCX,0x48 ;
  JMP 0x142c7023c` (Ghidra ne le connait pas comme fonction). `FUN_142c7023c` **ecrit** (decalages
  `<<` dans l'accumulateur `+0x30`, rincage vers `[+0x40]`).
- `FUN_140dbe170` : `FUN_140dbe400(*(param_3 + 0x10) + 0x48, param_2, 1 < param_4); return 1;`.
  `FUN_140dbe400` **lit** (charge depuis `[+0x40]`, decale `>>`).

La ligne `i4` de `ecs_table.tsv` (`deser_addr = FUN_142edb5cc -> FUN_142c7023c`) nomme donc
l'ecrivain, et `TI11_SPEC_10_FEUILLES.md` (`FUN_140dbe170 -> FUN_140dbe400`) nomme le lecteur :
les deux sont justes, ce sont les deux faces du meme composant.

### 1.2 L'objet serialise (colle des deux faces)

`param_1` = bloc du composant `+0x48`. Quatre FENTES de `0x40` octets a `param_1 + k*0x40`
(`k = 0..3`), puis le masque `uint32` a `param_1 + 0x100` et un scalaire `uint32` a
`param_1 + 0x104`. Chaque fente est un objet polymorphe : `+0x00` pointeur de vtable, `+0x08`
octet « bit commun » (`*(byte *)(param_2 + 1) & 1`), `+0x38` octet d'ETIQUETTE (`bVar1 = *pbVar6`,
`pbVar6 = param_1 + 0x38 + k*0x40` ; la fabrique ecrit `*(undefined1 *)(plVar4 + 7) = <etiquette>`).

### 1.3 Grammaire de l'en-tete — ecrivain `FUN_142c7023c`, lecteur `FUN_140dbe400`

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | masque des fentes (`param_1 + 0x100`) | **R(4)** — `FUN_142c70090` : `+0x2c] + 4` ; lecteur `FUN_140dbe598` : `+0x2c] + 4` | toujours | aucune (litteral) |
| 2 | scalaire (`param_1 + 0x104`) | **R(1)** cote ecrivain (`+0x2c] + 1`, inconditionnel) ; cote lecteur `iVar5 = (-(uint)(param_3 != 0) & 0xffffffe1) + 0x20` = **R(1) si `level > 1`, R(32) sinon** | toujours | **OUI : `level` du composant** (le `param_4` que l'EXE passe a `FUN_140dbe170`, `1 < param_4`) — entree de profil, lue dans le film. `ecs_table.tsv` donne `level = 2` pour `i4` -> R(1) sur les films du build courant ; la branche R(32) est la grammaire des anciens films |
| 3 | pour `k = 0..3` (`do { ... } while (iVar4 < 4)`), si le bit `k` du masque est pose : etiquette de la fente | **R(4)** (`+0x2c] + 4` ; `FUN_1406d6e28(..., 4)` en chemin lent) | bit `k` du masque | aucune (4 fentes et 4 bits sont des litteraux) |
| 4 | corps de la fente | voir §2 | bit `k` du masque | selon l'etiquette |

Total : **5 bits** (masque vide, film courant) ; **36 bits** (masque vide, `level <= 1`) ;
sinon `5 + somme_k (4 + corps(etiquette_k))`.

### 1.4 Le dispatch et l'appel virtuel (colle)

Ecrivains : `FUN_141e99630` (`switch(param_1)`, `case 0..5`, table de saut `0x141e99a54`)
-> `default: FUN_141e99150()` (`case 6..0xb`) -> `default: FUN_141e99a70()` (`if (param_1 == 0xc)`,
`else if (param_1 == 0xd)`, `else { if (param_1 != 0xe) FUN_1411c8f80(); ... }`).

Lecteurs / fabrique : `FUN_141e98e10` (`case 0..5`) -> `default: FUN_141e98c70()` (`case 6..0xb`)
-> `default: FUN_141e98f90()` (`0xc`, `0xd`, `0xe`, sinon `FUN_1411c8f80`).

Dans les trois ecrivains, chaque cas `1..0xe` fait la MEME chose : `R(1)` du bit commun
(`uVar6 = *(byte *)(param_2 + 1) & 1 ; +0x2c] + 1`) puis un saut indirect. Le desassemblage
fixe les registres au saut (extrait du cas 1) :

```
141e9964e: MOV R11,qword ptr [RDX]      ; R11 = la fente (objet)
141e99651: MOV RDX,qword ptr [R8]       ; RDX = le flux
141e99654: MOVZX R10D,byte ptr [R11 + 0x8]
...
141e9966e: MOV RCX,R11                  ; this = la fente
141e9967f: MOV RAX,qword ptr [R11]      ; RAX = vtable
141e99682: JMP qword ptr [RAX + 0x8]    ; ecrivain virtuel (this, flux)
```

Le lecteur, apres avoir pose la vtable et lu `R(1)` dans `+0x08`, appelle
`(**(code **)(*plVar4 + 0x10))(plVar4, lVar1)` : **`vtable+0x08` = ecrivain, `vtable+0x10` = lecteur**.

## 2. Table etiquette -> vtable -> corps (tout colle de `read_memory` sur les vtables)

Le « corps » ci-dessous EXCLUT le `R(1)` du bit commun et le `R(4)` d'etiquette ; la colonne
« fente » les INCLUT (`4 + 1 + corps`) pour un calcul direct.

| Etiquette | vtable | Ecrivain (`+0x08`) | Lecteur (`+0x10`) | Corps | Fente (bits) |
|---|---|---|---|---|---|
| `0` | aucune (`*(+0x38) = 0`) | — | — | rien | **4** |
| `1` | `0x143c2ae08` | `0x141e9e0b0` (non definie dans Ghidra, decodee a la main) | `FUN_141e9d6d0` | R(1) | **6** |
| `2` | `0x143c3df58` | `0x141e9d8e0` (idem) | `FUN_141e9d120` | 32 x R(1) | **37** |
| `3` | `0x143c4ee88` | `0x141e9d870` -> `FUN_142af2a50` | `0x141e9d660` -> `FUN_1407ef804` | R(4) (valeur+1) | **9** |
| `4` | `0x143c4ee58` | `0x141e9e060` (decodee a la main) | `FUN_141e9d670` | R(9) | **14** |
| `5` | `0x143c4ebd0` | `FUN_141e9ab10` | `FUN_141e9a9c0` | R(3) n + n x (R(1) porte inversee [+ R(13)]) | **8 + n x {1 ou 14}**, n <= 7 |
| `6` | `0x143c4eb70` | `0x141e9abb0` (decodee a la main) | `FUN_141e9aa60` | R(4) n + n x (R(1) [+ ref. entite domaine 0]) | **9 + n x {1 ou 3+w0}**, n <= 15 |
| `7` | `0x143c4ee70` | `0x141e9d870` -> `FUN_142af2a50` (le meme que `3`) | `0x141e9d0a0` -> `FUN_140968284` | R(4) (valeur+1) | **9** |
| `8` | `0x143c4ef40` | `0x141e9d880` -> `0x142c74154` (decodee a la main) | `0x141e9d0b0` -> `FUN_14109414c` | R(8) | **13** |
| `9` | `0x143c4ef10` | `0x141e9d890` (decodee a la main) | `FUN_141e9d0c0` | R(32) | **37** |
| `0xa` | `0x143c4ef28` | `FUN_141e9dff0` | `FUN_141e9d5e0` | R(4) + R(4) + R(32) | **45** |
| `0xb` | `0x143c4eef8` | `0x141e9dbc0` (decodee a la main) | `FUN_141e9d1a0` -> `FUN_1407f2058` | R(1) porte INVERSEE [+ R(5)] | **6 ou 11** |
| `0xc` | `0x143c4eec8` | `FUN_141e9deb0` | `FUN_141e9d440` | R(1) [+ ref. entite domaine 0] + R(32) | **38 ou 40+w0** |
| `0xd` | `0x143c4eee0` | `FUN_141e9d730` | `FUN_141e9cf50` | R(1) [+ ref. entite domaine 0] + R(32) | **38 ou 40+w0** |
| `0xe` | `0x143c4ef58` | `FUN_141e9dcd0` | `FUN_141e9d260` | R(1) [+ ref. entite domaine 0] + R(32) + R(1) | **39 ou 41+w0** |
| `0xf` | — | `FUN_1411c8f80` (assertion) | `FUN_1411c8f80` | flux invalide | — |

`w0` = largeur d'une reference d'entite du domaine 0 = `FUN_1406d310c(cardinal)`, **13 au
defaut** (§3.2) — dependance de config, jamais une constante du port.

Les qwords des vtables (colles, `read_memory 0x143c4eb80..0x143c4efa0`, `0x143c2ae08`,
`0x143c3df58`, `0x143c4eb70`) :

```
0x143c2ae08 0x14117a960 | 0x143c2ae10 0x141e9e0b0 | 0x143c2ae18 0x141e9d6d0          (etiq. 1)
0x143c3df58 0x14071fe44 | 0x143c3df60 0x141e9d8e0 | 0x143c3df68 0x141e9d120          (etiq. 2)
0x143c4ee88 0x14071fd34 | 0x143c4ee90 0x141e9d870 | 0x143c4ee98 0x141e9d660          (etiq. 3)
0x143c4ee58 0x140c1ef04 | 0x143c4ee60 0x141e9e060 | 0x143c4ee68 0x141e9d670          (etiq. 4)
0x143c4ebd0 0x141e9a930 | 0x143c4ebd8 0x141e9ab10 | 0x143c4ebe0 0x141e9a9c0 | +0x20 0x141e9cb40 | +0x28 0x141e9ca50  (etiq. 5)
0x143c4eb70 0x141e9a930 | 0x143c4eb78 0x141e9abb0 | 0x143c4eb80 0x141e9aa60 | +0x20 0x141e9ca60 | +0x28 0x141e9c9c0  (etiq. 6)
0x143c4ee70 0x142c741ec | 0x143c4ee78 0x141e9d870 | 0x143c4ee80 0x141e9d0a0          (etiq. 7)
0x143c4ef40 0x142c74228 | 0x143c4ef48 0x141e9d880 | 0x143c4ef50 0x141e9d0b0          (etiq. 8)
0x143c4ef10 0x142c74268 | 0x143c4ef18 0x141e9d890 | 0x143c4ef20 0x141e9d0c0          (etiq. 9)
0x143c4ef28 0x142c742f0 | 0x143c4ef30 0x141e9dff0 | 0x143c4ef38 0x141e9d5e0          (etiq. 0xa)
0x143c4eef8 0x142c742a4 | 0x143c4ef00 0x141e9dbc0 | 0x143c4ef08 0x141e9d1a0          (etiq. 0xb)
0x143c4eec8 0x141e9cec0 | 0x143c4eed0 0x141e9deb0 | 0x143c4eed8 0x141e9d440          (etiq. 0xc)
0x143c4eee0 0x141e9cb50 | 0x143c4eee8 0x141e9d730 | 0x143c4eef0 0x141e9cf50          (etiq. 0xd)
0x143c4ef58 0x141e9cd40 | 0x143c4ef60 0x141e9dcd0 | 0x143c4ef68 0x141e9d260          (etiq. 0xe)
```

### 2.0 Etiquette `0` — fente vide

Fabrique : `case 0: *(undefined1 *)(param_2[1] + 0x38) = 0; return;`. Ecrivain :
`case 0: return;`. **0 bit** apres l'etiquette. Statut : **releve**.

### 2.1 Etiquette `1` — un drapeau

Fabrique : vtable `PTR_LAB_143c2ae08`, `+0x08 = 0`, `+0x10 = 1` (octet), `+0x38 = 1`.
Ecrivain `0x141e9e0b0`, octets `44 0f b6 49 10 8b 42 38 41 83 e1 01 4c 8b 42 30 ff 42 2c 83 f8 40 ...`
= `MOVZX R9D,byte [RCX+0x10] ; ... ; AND R9D,1 ; ... ; INC dword [RDX+0x2c]` puis les deux chemins
(reserve / rincage) d'un champ de 1 bit. Lecteur `FUN_141e9d6d0` : `+0x2c] + 1`,
`*(byte *)(param_1 + 0x10) = bit`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun (`+0x08`) | R(1) | toujours | — |
| drapeau (`+0x10`) | R(1) | toujours | — |

Corps **1 bit**, fente **6 bits**. Statut : **releve**.

### 2.2 Etiquette `2` — 32 drapeaux

Fabrique : vtable `PTR_LAB_143c3df58`, `+0x08 = 0`, `+0x10 = 0` (dword), `+0x38 = 2`.
Ecrivain `0x141e9d8e0` (0x2e0 octets, non definie dans Ghidra) : `MOV EBX,8 ; MOV R10D,1`,
corps deroule 4 fois (`TEST [R11+0x10],R10D` puis `ROL EAX,1` / `ROL EAX,2` / `ROL EAX,3`,
chaque test suivi de `INC dword [RDX+0x2c]` et d'une ecriture de 1 bit), puis `41 c1 c2 04`
`ROL R10D,4` (offset 0x2c4), `48 83 eb 01` `SUB RBX,1` (offset 0x2c8), `0f 85 42 fd ff ff` `JNE`
(offset 0x2cb) = **8 x 4 = 32 bits**. Lecteur `FUN_141e9d120` :
`do { ... R(1) ... uVar3 = uVar3 + 1; } while ((int)uVar3 < 0x20);` dans `+0x10` bit a bit.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| masque 32 bits (`+0x10`), bit 0 en premier | 32 x R(1) | toujours | — |

Corps **32 bits**, fente **37 bits**. Statut : **releve**.

### 2.3 Etiquette `3` — un petit index (biais +1)

Fabrique : vtable `PTR_FUN_143c4ee88`, `+0x08 = 0`, `+0x10 = 8` (octet), `+0x38 = 3`.
Ecrivain `0x141e9d870`, octets `44 0f b6 41 10 48 8b ca e9 d3 51 c5 00` = `MOVZX R8D,byte [RCX+0x10] ;
MOV RCX,RDX ; JMP 0x142af2a50`. `FUN_142af2a50(flux, ., char v)` : `uVar5 = (int)param_3 + 1 ;
+0x2c] + 4` = **R(4) de `v + 1`**. Lecteur `0x141e9d660` = `LEA R8,[RCX+0x10] ; MOV RCX,RDX ;
JMP 0x1407ef804` ; `FUN_1407ef804` : `+0x2c] + 4 ; *param_3 = bVar4 - 1`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| index (`+0x10`), ecrit `valeur + 1`, lu `- 1` (`-1` = 0 sur le fil) | R(4) | toujours | — |

Corps **4 bits**, fente **9 bits**. Statut : **releve**.

### 2.4 Etiquette `4` — une valeur sur 9 bits

Fabrique : vtable `PTR_FUN_143c4ee58`, `+0x08 = 0`, `+0x10 = 0` (dword), `+0x38 = 4`.
Ecrivain `0x141e9e060`, octets colles : `44 8b 42 38 b8 40 00 00 00 41 2b c0 4c 8b ca 8b 51 10 83 f8 09
7c 1d 41 83 41 2c 09 41 8d 40 09 41 89 41 38 49 8b 41 30 48 c1 e0 09 48 0b c2 49 89 41 30 c3 0f ae e8
41 b8 09 00 00 00 49 8b c9 e9 83 8d 83 fe` = `MOV EDX,[RCX+0x10] ; CMP EAX,9 ; JL lent ;
ADD dword [R9+0x2c],9 ; ... SHL RAX,9 ; ... RET ; lent : MOV R8D,9 ; JMP FUN_1406d6e28`
(cible `0x141e9e0a5 - 0x017c727d = 0x1406d6e28`). Lecteur `FUN_141e9d670` : `+0x2c] + 9`,
`FUN_1406d6c7c(param_2, 9)`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| valeur (`+0x10`) | R(9) | toujours | — |

Corps **9 bits**, fente **14 bits**. Statut : **releve**.

### 2.5 Etiquette `5` — liste de 0..7 indices sur 13 bits

Fabrique : vtable `PTR_FUN_143c4ebd0`, `plVar4[0..4] = 0`, `+0x14 = 0`, `+0x38 = 5`.
Ecrivain `FUN_141e9ab10` :

```
uVar1 = *(uint *)(param_1 + 2);                 // n = dword +0x10
+0x2c] + 3  (ou FUN_1406d6e28(param_2, uVar1, 3))
for (plVar3 = param_1 + 3;                       // elements a +0x18, pas de 4 octets
     plVar3 != (longlong *)((longlong)param_1 + ((longlong)(int)lVar2 + 6) * 4); ...)
  (**(code **)(*param_1 + 0x20))(param_1, param_2, (int)*plVar3);
```

`vtable+0x20 = 0x141e9cb40` = `48 8b ca e9 dc b5 cc 00` = `MOV RCX,RDX ; JMP 0x142b68124`.
`FUN_142b68124(flux, ., uint v)` : `FUN_1406d49c4(param_1, param_2, param_3 == 0xffffffff)` =
**R(1) porte INVERSEE** (1 = absent) ; `if (param_3 != 0xffffffff) { +0x2c] + 0xd ... << 0xd }` =
**R(13) de `v`** (le `0xd` est un LITTERAL du code, pas un `FUN_1406d310c`).
Lecteur `FUN_141e9a9c0` : `+0x2c] + 3` (ou `FUN_1406d6c7c(param_2, 3)`), puis `n` fois
`vtable+0x28 = 0x141e9ca50 -> 0x142b67f08` : `FUN_1406cf008()` R(1), si 0 : `+0x2c] + 0xd` R(13),
sinon `0xffffffff`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| n (`+0x10`) | R(3) | toujours | — |
| pour chaque element : porte inversee (1 = `-1`) | R(1) | n fois | — |
| element (`+0x18 + i*4`) | R(13) | si porte = 0 | — (litteral `0xd`) |

Corps **3 + n x (1 ou 14)** bits, `n` dans `0..7` ; fente **8 a 106 bits**. Statut : **releve**.
Il n'y a **aucune recursion** (correction de `TI11_SPEC_10_FEUILLES.md`).

### 2.6 Etiquette `6` — liste de 0..15 references d'entite

Fabrique (`FUN_141e98c70`, `case 6`) : vtable `PTR_FUN_143c4eb70`, `plVar4[0..6] = 0`,
`+0x14 = 0`, `+0x38 = 6`. Ecrivain `0x141e9abb0` (0xb0 octets, non definie dans Ghidra), decodee :
`MOV EDX,[RSI+0x10] ; CMP EAX,4 ; JL lent ; ADD dword [RDI+0x2c],4 ; ... SHL RAX,4` (lent :
`MOV R8D,4 ; CALL FUN_1406d6e28`) = **R(4) de n** ; puis `MOVSXD RBP,[RSI+0x10] ; LEA RBX,[RSI+0x18] ;
ADD RBP,6 ; LEA RBP,[RSI+RBP*4]` et boucle `MOV R8D,[RBX] ; CALL [RAX+0x20] ; ADD RBX,4 ; CMP RBX,RBP ;
JNE` = n appels de `vtable+0x20 = FUN_141e9ca60(this, flux, valeur)`.

`FUN_141e9ca60` (decompile + desassemblage `141e9ca6c CALL 0x1409a615c`, `141e9cb1b MOV dword ptr
[RSP + 0x20],0xffffffff`, `141e9cb23 XOR R8D,R8D`, `141e9cb29 CALL 0x1406d5110`) :
`iVar2 = FUN_1409a615c(param_3) ; bVar6 = iVar2 != -1 ; +0x2c] + 1` = **R(1) presence** ;
`if (bVar6) FUN_1406d5110(., param_2, 0, iVar2, 0xffffffff)` = **reference d'entite, domaine 0**
(§3.2). Lecteur `FUN_141e9aa60` : `+0x2c] + 4` (ou `FUN_1406d6c7c(param_2, 4)`), puis `n` fois
`vtable+0x28 = FUN_141e9c9c0` : R(1) ; si 1 : `FUN_1406d3140()` puis `FUN_1405d5dbc(...)` ;
sinon `*param_2 = 0xffffffff` ; range dans `+0x18 + i*4`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| n (`+0x10`) | R(4) | toujours | — |
| pour chaque element : presence (1 = present) | R(1) | n fois | — |
| reference d'entite domaine 0 : `(handle & 0x3fffffff) - base` | R(w0) | si presence = 1 et `w0 > 0` | **OUI** : `w0 = FUN_1406d310c(cardinal du domaine 0)` (§3.2) |
| generation `(handle >> 30) & 3` | R(2) | si presence = 1 | — |

Corps **4 + n x (1 ou 3 + w0)** bits, `n` dans `0..15` ; fente **9 a 9 + 15 x 16 = 249 bits** avec
`w0 = 13`. Statut : **releve** (largeur de la reference = entree de profil).

### 2.7 Etiquette `7` — un petit index (biais +1), variante `-1` par defaut

Fabrique : vtable `PTR_FUN_143c4ee70`, `+0x08 = 0`, `+0x10 = 0xff` (octet, soit `-1`), `+0x38 = 7`.
Ecrivain `vtable+0x08 = 0x141e9d870` : **le meme stub que l'etiquette 3** -> `FUN_142af2a50`
= R(4) de `v + 1`. Lecteur `0x141e9d0a0` = `LEA R8,[RCX+0x10] ; MOV RCX,RDX ; JMP 0x140968284` ;
`FUN_140968284` : `+0x2c] + 4 ; *param_3 = bVar4 - 1` (corps identique a `FUN_1407ef804`).

Corps **4 bits**, fente **9 bits**. Statut : **releve**. Seule la vtable (donc la classe, et
la valeur par defaut `-1`) distingue `7` de `3` ; sur le fil, meme grammaire.

### 2.8 Etiquette `8` — un octet

Fabrique : vtable `PTR_FUN_143c4ef40`, `+0x08 = 0`, `+0x10 = 0` (dword), `+0x38 = 8`.
Ecrivain `0x141e9d880` = `4c 8d 41 10 48 8b ca e9 c8 68 dd 00` = `LEA R8,[RCX+0x10] ; MOV RCX,RDX ;
JMP 0x142c74154` (`0x141e9d88c + 0x00dd68c8`). Cible non definie dans Ghidra, octets colles
`45 8b 08 48 8b d1 8b 49 38 b8 40 00 00 00 2b c1 4c 8b 42 30 83 42 2c 08 83 f8 08 7c 12 49 c1 e0 08
8d 41 08 4d 0b c1 89 42 38 4c 89 42 30 c3` = `MOV R9D,[R8] ; ... ; ADD dword [RDX+0x2c],8 ; CMP EAX,8 ;
JL lent ; SHL R8,8 ; OR R8,R9 ; ... RET` puis le chemin lent a `b9 08 00 00 00` = **R(8)**.
Lecteur `0x141e9d0b0` = `LEA R8,[RCX+0x10] ; MOV RCX,RDX ; JMP 0x14109414c`
(`0x141e9d0bc - 0x00e08f70`) ; `FUN_14109414c` : `+0x2c] + 8`, `*param_3 = uVar5`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| valeur (`+0x10`) | R(8) | toujours | — |

Corps **8 bits**, fente **13 bits**. Statut : **releve**.

### 2.9 Etiquette `9` — un mot de 32 bits

Fabrique : vtable `PTR_FUN_143c4ef10`, `+0x08 = 0`, `+0x10 = 0xffffffff`, `+0x38 = 9`.
Ecrivain `0x141e9d890`, octets `44 8b 42 38 b8 40 00 00 00 41 2b c0 4c 8b ca 8b 51 10 83 f8 20 7c 1d
41 83 41 2c 20 41 8d 40 20 41 89 41 38 49 8b 41 30 48 c1 e0 20 48 0b c2 49 89 41 30 c3 0f ae e8 41 b8
20 00 00 00 49 8b c9 e9 53 95 83 fe` = meme forme que l'etiquette 4 avec `0x20` (cible lente
`0x141e9d8d5 - 0x017c6aad = 0x1406d6e28`). Lecteur `FUN_141e9d0c0` : `+0x2c] + 0x20`.

Corps **32 bits**, fente **37 bits**. Statut : **releve**.

### 2.10 Etiquette `0xa` — deux petits index et un mot

Fabrique : vtable `PTR_FUN_143c4ef28`, `+0x08 = 0`, `+0x10 = 0xff08` (word : `+0x10 = 8`,
`+0x11 = 0xff`), `+0x14 = 0xffffffff`, `+0x38 = 0xa`. Ecrivain `FUN_141e9dff0` (desassemblage) :
`141e9dff6 MOVZX R8D,byte ptr [RCX + 0x10] ; 141e9e004 CALL 0x142af2a50` ;
`141e9e009 MOVZX R8D,byte ptr [R11 + 0x11] ; 141e9e011 CALL 0x142af2a50` ;
`141e9e01e MOV EDX,dword ptr [R11 + 0x14]` puis `+0x2c] + 0x20` (lent : `141e9e04b MOV R8D,0x20 ;
141e9e059 JMP 0x1406d6e28`). Lecteur `FUN_141e9d5e0` : `FUN_1407ef804(param_2, param_2, param_1 + 0x10)`,
`FUN_140968284(param_2)`, puis `+0x2c] + 0x20` dans `+0x14`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| index 1 (`+0x10`), ecrit `v + 1` | R(4) | toujours | — |
| index 2 (`+0x11`), ecrit `v + 1` | R(4) | toujours | — |
| mot (`+0x14`) | R(32) | toujours | — |

Corps **40 bits**, fente **45 bits**. Statut : **releve**.

### 2.11 Etiquette `0xb` — un index de joueur (porte inversee + 5 bits)

Fabrique : vtable `PTR_FUN_143c4eef8`, `+0x08 = 0`, `+0x10 = 0xffffffff`, `+0x38 = 0xb`.
Ecrivain `0x141e9dbc0` (0x110 octets, non definie dans Ghidra), decodee : `MOV R11D,-1 ;
MOV EDX,[RCX+0x10] ; MOV ECX,EDX ; SHR ECX,1 ; AND ECX,0x7fff ; CMP EDX,-1 ; CMOVNE R11D,ECX`
(index = `(handle >> 1) & 0x7fff`, ou `-1`) ; `XOR R9D,R9D ; CMP R11D,-1 ; SETE R9B ; INC dword [RAX+0x2c]`
= **R(1) = (index == -1)**, porte INVERSEE ; puis `CMP R11D,-1 ; JE ret ; ... CMP R10D,5 ; JL lent ;
ADD dword [RAX+0x2c],5 ; ... SHL RCX,5` (lent : `MOV R8D,5 ; JMP FUN_1406d6e28`) = **R(5) de l'index**.
Lecteur `FUN_141e9d1a0` : `uVar1 = FUN_1407f2058(param_2)` (la porte inversee `R(1)` puis `R(5)`
deja nommee par la note de methode, §4), puis resolution runtime (`FUN_14049746c`, `FUN_140496bac`)
vers `+0x10`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| porte inversee (1 = pas d'index) | R(1) | toujours | — |
| index | R(5) | si porte = 0 | — (litteral `5`) |

Corps **1 ou 6 bits**, fente **6 ou 11 bits**. Statut : **releve**.

### 2.12 Etiquette `0xc` — une reference d'entite optionnelle et un mot

Fabrique (`FUN_141e98f90`) : vtable `PTR_FUN_143c4eec8`, `+0x08 = 0`, `+0x10 = <pointeur TLS,
non serialise>`, `+0x18 = -1` (qword), `+0x38 = 0xc`. Ecrivain `FUN_141e9deb0` :
`iVar3 = FUN_1409a615c(*(undefined4 *)(param_1 + 0x18)) ; bVar6 = iVar3 != -1 ; +0x2c] + 1` =
**R(1) presence** ; `if (bVar6) FUN_1406d5110()` avec, au desassemblage, `141e9df7a MOV dword ptr
[RSP + 0x20],0xffffffff ; 141e9df82 XOR R8D,R8D ; 141e9df85 MOV RDX,RBX ; 141e9df88 CALL 0x1406d5110`
(`param_3 = 0` -> domaine 0, `param_5 = -1`) ; puis `uVar1 = *(uint *)(param_1 + 0x1c)` et
`+0x2c] + 0x20` (lent `141e9dfc6 MOV R8D,0x20 ; 141e9dfde JMP 0x1406d6e28`) = **R(32)**.
Lecteur `FUN_141e9d440` : R(1) ; si 1 : `FUN_1406d3140()` + resolution runtime dans `+0x18` ;
puis `+0x2c] + 0x20` dans `+0x1c`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| presence | R(1) | toujours | — |
| reference d'entite domaine 0 | R(w0) | si presence = 1 et `w0 > 0` | **OUI** (§3.2) |
| generation | R(2) | si presence = 1 | — |
| mot (`+0x1c`) | R(32) | toujours | — |

Corps **33 ou 35 + w0** bits, fente **38 ou 40 + w0** (`53` avec `w0 = 13`). Statut : **releve**.

### 2.13 Etiquette `0xd` — une reference d'entite optionnelle et un identifiant de chaine

Fabrique : vtable `PTR_FUN_143c4eee0`, `+0x08 = 0`, `+0x10 = -1` (qword), `+0x38 = 0xd`.
Ecrivain `FUN_141e9d730` : meme squelette que `0xc` sur `*(param_1 + 0x10)` (`141e9d748 CALL
0x1409a615c`, `141e9d7fa MOV dword ptr [RSP + 0x20],0xffffffff ; 141e9d802 XOR R8D,R8D ;
141e9d808 CALL 0x1406d5110`), puis `uVar1 = *(uint *)(param_1 + 0x14)` R(32) (`141e9d846 MOV R8D,0x20 ;
141e9d85e JMP 0x1406d6e28`). Lecteur `FUN_141e9cf50` : R(1) [+ `FUN_1406d3140()`] dans `+0x10`,
puis `FUN_14080dec4(param_2, "watched-marker", param_1 + 0x14)` ; `FUN_14080dec4` : `+0x2c] + 0x20`,
`*param_3 = uVar4` = R(32) (le libelle `"watched-marker"` est un nom de debogage du champ, il
n'est pas sur le fil).

Corps **33 ou 35 + w0** bits, fente **38 ou 40 + w0**. Statut : **releve**.

### 2.14 Etiquette `0xe` — reference optionnelle, mot, drapeau

Fabrique : vtable `PTR_FUN_143c4ef58`, `+0x08 = 0`, `+0x10 = <pointeur TLS, non serialise>`,
`+0x18 = 0xffffffff`, `+0x1c = 0`, `+0x20 = 0` (octet), `+0x38 = 0xe`. Ecrivain `FUN_141e9dcd0` :
squelette de `0xc` sur `+0x18` (`141e9dced CALL 0x1409a615c`, `141e9ddaa MOV dword ptr
[RSP + 0x20],0xffffffff ; 141e9ddb2 XOR R8D,R8D ; 141e9ddb8 CALL 0x1406d5110`), puis
`uVar6 = *(uint *)(param_1 + 0x1c)` R(32) (`141e9ddea MOV R8D,0x20 ; 141e9ddf3 CALL 0x1406d6e28`),
puis `uVar6 = *(byte *)(param_1 + 0x20) & 1 ; +0x2c] + 1` = **R(1)**. Lecteur `FUN_141e9d260` :
R(1) [+ `FUN_1406d3140()`] dans `+0x18`, `+0x2c] + 0x20` dans `+0x1c`, `+0x2c] + 1` dans `+0x20`.

| Champ | Largeur | Condition | Config |
|---|---|---|---|
| bit commun | R(1) | toujours | — |
| presence | R(1) | toujours | — |
| reference d'entite domaine 0 | R(w0) | si presence = 1 et `w0 > 0` | **OUI** (§3.2) |
| generation | R(2) | si presence = 1 | — |
| mot (`+0x1c`) | R(32) | toujours | — |
| drapeau (`+0x20`) | R(1) | toujours | — |

Corps **34 ou 36 + w0** bits, fente **39 ou 41 + w0**. Statut : **releve**.

## 3. Les sous-lecteurs communs, nommes

### 3.1 Largeurs fixes

| Fonction | Role | Preuve collee |
|---|---|---|
| `FUN_142af2a50(flux, ., char v)` | ecrit **R(4) de `v + 1`** | `uVar5 = (int)param_3 + 1 ; *(int *)(param_1 + 0x2c) + 4` |
| `FUN_1407ef804` / `FUN_140968284(flux, ., char *dst)` | lisent R(4), rendent `- 1` | `+0x2c] + 4 ; *param_3 = bVar4 - 1` (deux copies du meme corps) |
| `FUN_14109414c(flux, ., uint *dst)` | lit R(8) | `+0x2c] + 8` |
| `FUN_14080dec4(flux, nom, uint *dst)` | lit R(32) | `+0x2c] + 0x20` |
| `FUN_1406d49c4(flux, ., byte b)` | ecrit R(1) de `b` | `+0x2c] + 1` ou `FUN_1406d6e28(param_1, b, 1)` |
| `FUN_1406d6e28(flux, valeur, n)` / `FUN_1406d6c7c(flux, n)` | chemin lent d'ecriture / de lecture de `n` bits | `+0x2c] + param_3` / `+0x2c] + param_2` |
| `FUN_1409a615c(handle)` | resolution RUNTIME d'un handle vers un index (`FUN_140477000` puis `+0x114`), `-1` si absent | ne touche pas le flux ; ne pilote que le bit de presence |
| `FUN_1411c8f80` | assertion sans retour | `/* WARNING: Subroutine does not return */` |

### 3.2 La reference d'entite (`FUN_1406d5110`, domaine 0) — ENTREE DE PROFIL

Appelee ici toujours avec `param_3 = 0` (`XOR R8D,R8D`), `param_4 = index`, `param_5 = -1`.
Chemin colle du decompile :

```
uVar8 = param_4 >> 0x1e & 3;                                   // generation
if (DAT_144706104 == '\0') { iVar3 = 0; uVar6 = DAT_144706100; }
else { iVar3 = (&DAT_1451f98d0)[param_3 * 2]; uVar6 = (&DAT_1451f98d4)[param_3 * 2]; }  // base, cardinal du domaine 0
// (param_3 == 1 : porte 0x23/0x28 supplementaire — NON prise ici)
iVar2 = FUN_1406d310c(uVar6);                                  // largeur = ceil(log2(cardinal))
if (0 < iVar2) { uVar9 = ((uint)param_4 & 0x3fffffff) - iVar3; +0x2c] + iVar2; ... }   // R(largeur)
+0x2c] + 2;  ...  uVar8                                        // R(2) generation
```

`NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md` (§2.3) donne la table des domaines : domaine 0 = base
512, cardinal `DAT_144706100 - 512`, **largeur 13 au defaut** (`DAT_144706100 = 0x1FFF`) ; si la
bascule `DAT_144706104` est a 0, base 0 et cardinal `DAT_144706100`, largeur 13 aussi. La bascule
est ecrite depuis le film (`1429874ab`, `R(1)` du paquet) : **`w0` est une entree de profil,
resolue par le meme mecanisme que les references de `ti=35` / `ti=40`, jamais un `13` en dur.**

## 4. Qui d'autre lit ou ecrit cette liste de filtres (`get_xrefs_to`)

Lecteur partage `FUN_140dbe400` : `FUN_140dbde44`, `FUN_140dbdf34`, `FUN_140dbdf5c`,
`FUN_140dbdfd8`, `FUN_140dbe170`, `FUN_140dbe194`, `FUN_140dbe25c`, `FUN_142ed7068`.
`LOT_RE_LETTRE_HUD.md` §4.3 en nomme cinq pour `ti=12` (i2..i6) ; `FUN_140dbdf5c` est
`managed-object-interaction-filter-component` (descripteur `0x143d092f0`, `+0x28 = 0x142edb250`
= `MOV RCX,[R8+0x30] ; ADD RCX,0x68 ; JMP ...`, `+0x40 = FUN_140dbdf5c` -> `FUN_140dbe400(+0x68, ., 1 < param_4)`) ;
`FUN_142ed7068` est un wrapper identique a `FUN_140dbe170` (`+0x48`, `1 < param_4`) pour un
autre descripteur ; `FUN_140dbe25c` prefixe la liste puis lit un R(32) et par bit du masque un
R(32) [+ R(4) si `param_4 == 0`].

Ecrivain partage `FUN_142c7023c` : `FUN_141f19604`, `FUN_141f1b3a8`, `FUN_142c94dd4` l'appellent
en PREFIXE puis ecrivent d'autres champs (un R(1) ou `FUN_142c8ae0c` / `FUN_141d12268`, puis par
bit pose du masque autant de R(3)) — ce sont les ecrivains des listes « avec distance » de
`ti=12`, hors perimetre de cette note mais a porter avec le meme noyau.

## 5. Ce que le port demandera

1. **Un seul noyau `interactionFilterList(level)`** : R(4) masque ; R(1) si `level > 1` sinon
   R(32) ; pour chacun des 4 bits poses : R(4) etiquette puis la fente (§2). Le `level` vient du
   registre du film (colonne `level` de `ecs_table.tsv`, `2` pour `i4`), pas d'une constante.
2. **Quinze fentes, toutes decidables hors ligne** : aucune ne branche sur un etat runtime autre
   que ce qui est ECRIT dans le flux (les bits de presence sont sur le fil ; `FUN_1409a615c` ne
   sert qu'a les produire cote ecrivain). Etiquette `0xf` = flux invalide : desync declaree, pas
   un `panic`.
3. **Une entree de profil** : la largeur `w0` des references d'entite du domaine 0 (etiquettes
   `6`, `0xc`, `0xd`, `0xe`) — reutiliser le resolveur de domaines deja construit pour `ti=35` /
   `ti=40` (`NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md`), et non un `13`.
4. **Deux erreurs a corriger dans les documents anterieurs** au moment du port :
   `TI11_SPEC_10_FEUILLES.md` (« sel5 R(3)count + recursion » -> R(3) n + n x (R(1) [+ R(13)]) ;
   « sel6..15 assert » -> 6..0xe existent, 0xf seul asserte) et `NOTE_3_6_TI11_2026-09-16.md`
   §3 (« 6 a 15 non atteints » -> 6..0xe atteints par chainage des `default`). La ligne `i4` de
   `ecs_table.tsv` passe de `non_porte` a `porte` dans le commit qui porte, jamais avant
   (`ecs_table_guard_test.go`).
5. **Mesure d'acceptation inchangee** : fermeture de `ti=11` de 0/325 a 325/325 sur les 7
   bobines ; un intermediaire designe une etiquette mal lue. Le meme noyau ferme du meme coup
   `ti=12` i5 et i6 (`LF(lvl>1)` seul) et prefixe i2/i3/i4.
6. **Ecart de largeur attendu** : avec le masque le plus souvent vide, `i4` pese 5 bits sur les
   films du build courant ; un film ancien (`level <= 1`) en pese 36. Le golden doit tolerer les
   deux sans heuristique : c'est le `level` qui tranche.

## 6. Statut de l'item du plan

3.6.b (volet `ti=11`) **PREPARE EN ENTIER** : en-tete au bit pres, 15 etiquettes relevees
(14 corps + la fente vide), 1 entree de profil nommee (`w0`), 1 dependance de version nommee
(`level`). Aucun composant `non_elucide` ; aucun `partiel`. Le port peut se dimensionner : un
noyau, quinze branches, deux entrees de profil.

## 7. Journal des appels Ghidra

~70 appels HTTP en lecture (`decompile_function`, `disassemble_function`, `read_memory`,
`get_xrefs_to`, `get_function_by_address`, `search_strings`, `inspect_memory_content`), aucun
point d'entree d'ecriture. 16 appels ont rendu une erreur, tous attendus : 2 mauvais parametre
(`name=` au lieu de `address=`), 14 « No function found » sur des stubs que Ghidra n'a pas
definis comme fonctions (`0x142edb5cc`, `0x141e99a60`, `0x141e9d870`, `0x141e9e060`, `0x141e9d880`,
`0x141e9d890`, `0x141e9dbc0`, `0x141e9d660`, `0x141e9d0a0`, `0x141e9d0b0`, `0x141e9cb40`,
`0x142c74155` x2, `0x142edb5cc` via `get_function_by_address`) — tous decodes a la main depuis
`read_memory`, octets colles dans la section correspondante.
