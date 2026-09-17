# ti=43 (device) — grammaires des ecrivains, groupe B : composants i11 a i21 (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « porter les composants
> manquants archetype par archetype »), sans production : aucun film decode, aucun test Go,
> aucun fichier de production touche. Source : `HaloInfinite.exe` par Ghidra en LECTURE SEULE
> (HTTP direct `127.0.0.1:8089`, programme `HaloInfinite.exe`, base `0x140000000`). Methode :
> `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` ; note d'archetype : `NOTE_3_6_TI43_2026-09-16.md`.
>
> Instrument / Ghidra lecture seule.
>
> Archetype : **ti=43 `device (dispositif de carte)`** (colonne `archetype` de `ecs_table.tsv`).
> L'intitule de mission disait « ti=43 armes » : c'est ti=42 qui est l'arme au sol ; le releve
> suit la table, pas l'intitule. Groupe : **ti43b**, indices **11 a 21**, plus la surveillance
> d'**i2** (`partiel`) et le recontrole d'**i19** (grammaire deja ecrite par la note ti=43).

---

## 0. Ce que la chaine des descripteurs a donne pour les douze composants

La chaine chaine -> accesseur -> slot du nom -> `slot + 0x28` (= `descripteur + 0x40`) a ete
rejouee sur les douze noms. Toutes les adresses ci-dessous sont collees depuis Ghidra.

| i | Composant | Chaine | Accesseur | Slot du nom | Descripteur (`slot - 0x18`) | Mot a `desc+0x00` | Ecrivain lu a `slot + 0x28` | `deser_addr` de la table / note | Concorde |
|---|---|---|---|---|---|---|---|---|---|
| 11 | `object-dead-state-component` | `0x143c99320` | `0x14064c6d0` | `0x143d0ba48` | `0x143d0ba30` | `0x141191ab0` | `FUN_140c1dce0` | `FUN_140c1dce0` | oui |
| 12 | `object-scale-component` | `0x143c99308` | `0x14064c6c0` | `0x143d0b958` | `0x143d0b940` | `0x141191ab0` | `FUN_1407dc6e4` | `FUN_1407dc6e4` | oui |
| 13 | `object-maximum-vitalities-component` | `0x143c99380` | `0x14064c6b0` | `0x143d0b908` | `0x143d0b8f0` | `0x141191ab0` | `FUN_1407ee054` | `FUN_1407ee054` | oui |
| 14 | `object-dissolver-component` | `0x143c99360` | `0x14064c6a0` | `0x143d0bf80` | `0x143d0bf68` | `0x141191ab0` | `FUN_140dd9f9c` | `FUN_140dd9f9c` | oui |
| 15 | `object-low-frequency-component` | `0x143c99340` | `0x14064c690` | `0x143d0bf30` | `0x143d0bf18` (voir remarque) | `0x14076ced0` | `FUN_1407ef088` | `FUN_1407ef088` | oui |
| 16 | `object-physics-flags-component` | `0x143c992e8` | `0x14064c680` | `0x143d0bfd0` | `0x143d0bfb8` | `0x141191ab0` | `FUN_1407ee070` | `FUN_1407ee070` | oui |
| 17 | `object-frame-configuration-component` | `0x143c991a0` | `0x14064c670` | `0x143d0be88` | `0x143d0be70` | `0x141191ab0` | `FUN_1407f0534` | `FUN_1407f0534` | oui |
| 18 | `device-position-component` | `0x143c990d8` | `0x1411774f0` | `0x143d0cf00` | `0x143d0cee8` | `0x141191ab0` | `FUN_140bef320` | `FUN_140bef320` | oui |
| 19 | `device-position-animation-name-component` | `0x143c99078` | `0x1411774e0` | `0x143d0ceb0` | `0x143d0ce98` | `0x141191ab0` | `FUN_1410156e4` | `FUN_1410156e4` | oui |
| 20 | `device-position-animation-control-component` | `0x143c990a8` | `0x1411774d0` | `0x143d0cff0` | `0x143d0cfd8` | `0x141191ab0` | `FUN_142f02d04` | `FUN_142f02d04` (note) | oui |
| 21 | `device-position-group-component` | `0x143c98f90` | `0x1411774c0` | `0x143d0cf50` | `0x143d0cf38` | `0x141191ab0` | `FUN_1407f0678` | `FUN_1407f0678` (note) | oui |
| 2 | `object-forward-and-up-dynamic-precision-component` | `0x143ca7380` | `0x140fc3ec0` | `0x143e2ba80` | — (autre famille, §12) | `"Transloc"` (ASCII) | `slot + 0x28` = `0x1411c8f80` : NE concorde PAS | `FUN_140c5f7ec` | non par `+0x28` ; oui par `slot + 0x20` (§12) |

Remarques :

- **i15** : le mot a `slot - 0x18` est `0x14076ced0`, qui est la constante `+0x08` de la forme
  §2 de la note de methode, pas `0x141191ab0`. La signature est donc decalee d'un slot pour ce
  descripteur (famille voisine, ou descripteur commencant a `0x143d0bf10`) — non creuse, parce
  que la relation utile `ecrivain = slot du nom + 0x28` tient et concorde avec la table.
- **i2** n'est pas de la famille a 10 slots : autour du slot du nom `0x143e2ba80`, la memoire lue
  (`0x143e2ba40..0x143e2babf`) porte des chaines ASCII (`Knockback`, `EquipmentRuntimeTypeData`,
  `Translocator`) puis `0x141179610` (`0x143e2ba78`), `0x140fc3ec0` (le nom, `0x143e2ba80`),
  `0x1404ab600`, `0x1431fd660`, `0x1431fd6a0`, **`0x140c5f7ec` a `0x143e2baa0`** (= `slot + 0x20`),
  `0x1411c8f80`, `0x1404ab600`, `0x1411a5ef0`. La table dit « chaine ASCII -> vtable[0x28] » : c'est
  coherent si le descripteur commence a `0x143e2ba78` (nom a `+0x08`, lecteur a `+0x28`). Ce
  lecteur a **5 parametres** (`param_4` = pointeur de reference, `param_5` = entier), la ou les
  ecrivains de la famille a 10 slots en ont 3 (`ctx`, `flux`, `record`).

## 1. Les briques communes rencontrees (pour lire les tableaux)

Objet de flux `flux` : `+0x2c` compteur de bits, `+0x30` accumulateur, `+0x38` bits en
reserve, `+0x40` curseur d'octets (note de methode §4). Briques relevees dans cette note :

| Brique | Ce qu'elle lit | Preuve |
|---|---|---|
| `FUN_1406cf008(flux)` | `R(1)` | note de methode |
| `R(1)` « en ligne » | `if (flux+0x38 < 0x40) { +0x2c += 1 ; acc <<= 1 } else FUN_1406d6c7c(flux, 1)` | `FUN_1406d6c7c(flux, n)` : `+0x2c += param_2`, rend `n` bits (decompile, l. 39-44) ; motif vu dans i14, `FUN_1406d1024`, `FUN_14076e744`, `FUN_140c1dd44` |
| `FUN_1406d310c(n)` | 0 bit ; rend `ceil(log2(n))` (position du bit haut, +1 si `n` n'est pas une puissance de 2) | decompile |
| `FUN_1406d676c(flux, flux, dest, n)` | **copie brute de `n` bits** dans `dest` : blocs de 64 (`+0x2c += 0x40` par bloc, octets renverses) puis reste `+0x2c += n mod 64`, octet par octet, MSB en tete | decompile (126 l.) |
| `FUN_1406d84b4(flux ; XMM2 = min ; XMM3 = max ; [RSP+0x20] = n ; [RSP+0x28] = a ; [RSP+0x30] = b)` | **`R(n)` dequantifie dans `[min, max]`** ; `a` et `b` ne changent que la valeur, jamais la largeur | desassemblage `0x1406d84b4..0x1406d8673` : `MOVSXD RBX,[RSP+0x28]` (= n cote appele), `SIL = [RSP+0x30]` (= a), `[RSP+0x38]` (= b) ; pas = `1 << n` (a=0) ou `(1 << n) - 1` (a=1) ; b=0 : `min + brut*(max-min)/pas + 0,5*(max-min)/pas` (`DAT_143cd84b0` = `0x3f000000` = 0,5f) et, si a=1 et `2*brut == pas-1`, milieu `(min+max)*0,5` (`DAT_143cd8910` = `0x3fe0000000000000`) ; b=1 : `brut=0 -> min`, `brut=pas-1 -> max`, sinon `min + (brut-1)*(max-min)/(pas-2) + demi-pas` |
| `FUN_141015740(flux, flux, out)` | `R(32)` (`+0x2c += 0x20`) | decompile |
| `FUN_14080d6f0(_, flux, out)` | `R(32)` | decompile |
| `FUN_14080d69c(_, flux, out, defaut)` | `R(1)` ; 0 -> `*out = defaut` ; 1 -> `FUN_14080d6f0` = `R(32)` | decompile |
| `FUN_1424ccc74(flux, _, out)` | `R(5)` | decompile |
| `FUN_1407f0354` / `FUN_1424d9a30` / `FUN_1407ef8e4` / `FUN_142af27f8` | `R(5)` / `R(3)` / `R(3)` / `R(2)` (octet de drapeaux) | decompiles |
| `FUN_1407ef804` / `FUN_1407ef724` | `R(4) - 1` / `R(6) - 1` | decompiles |
| `thunk_FUN_140e9fadc` -> `FUN_140e9fadc` | `R(7)` ; rend `valeur - 1` (0 -> -1) | decompile |
| `FUN_1424cd07c(flux)` | `R(6)` ; rend `valeur + 1` | decompile |
| `FUN_1424cd060(out, flux)` | `R(1)` | decompile |
| `FUN_1411b1ac0(out, flux)` -> `FUN_140e82b84` | `R(1)` ; si 1 -> `R(12)` (`+0x2c += 0xc`) | decompiles |
| `FUN_1406d1024(flux)` | porte INVERSEE : `R(1)` ; si 0 -> `R(6)` ; si 1 -> rend `0xffffffff` | decompile |
| `FUN_1409684dc(flux)` | `R(1)` ; si 0 -> `R(4)` ; si 1 -> rend `0xffffffff` | decompile |
| `FUN_1407f08f8(flux, out)` | `R(8)` (masque `0xff`) | decompile |
| `FUN_1407688b0(masque, i, bit)` | 0 bit (pose un bit dans un mot) | decompile |
| `FUN_1404d343c(obj)` | 0 bit (initialisation : `-1`, `0`, `1,0f`) | decompile |
| `FUN_1406d8288`, `FUN_1406d8228`, `FUN_1406d8678`, `FUN_140c5f9c8`, `FUN_140475500` | 0 bit (aucun `+0x2c`, aucun `FUN_1406cf008` : depaquetage / bornage) | grep sur les decompiles |

Constantes flottantes lues en `.rdata` (toutes sur 4 octets, petit-boutiste) :

| Adresse | Octets | Valeur |
|---|---|---|
| `0x143cd8374` | `0000803f` | 1,0f |
| `0x143cd873c` | `00002041` | 10,0f |
| `0x143cd84e4` | `00007042` | 60,0f |
| `0x143cd8d08` | `0000f042` | 120,0f |
| `0x143cd84ec` | `000080bf` | -1,0f |
| `0x143cd8918` | `db0f4940` | +3,1415927f |
| `0x143cd8920` | `db0f49c0` | -3,1415927f |
| `0x14472a654` (12 octets, via `PTR_DAT_14474c2e8`) | `00000000 00000000 0000803f` | vecteur (0, 0, 1) |

Convention des tableaux : `ctx` = `param_3` du lecteur, `obj` = `*(ctx + 0x10)` (les donnees du
composant), `ti` = `*(int *)(ctx + 0x30)`.

---

## 2. i11 `object-dead-state-component`

- Descripteur `0x143d0ba30` ; ecrivain **`FUN_140c1dce0`** (3 parametres).
- **Corrige apres verification independante (synthese du 2026-09-17)** : l'enumeration du
  bloc lourd omettait un `R(8)` inconditionnel (`FUN_140c1e3f0` -> `obj+0x74+0x1c`, l'octet de
  drapeaux) et presentait le `R(2)` sans sa condition, qui est le bit `0x10` de cet octet.
  Relecture Ghidra : decompile de `FUN_140c1dd44` (`FUN_14080d69c(...) ; FUN_140c1e3f0(param_2) ;
  ...`), desassemblage `140c1dd6c LEA R8,[R14 + 0x1c]` / `140c1dd73 CALL 0x140c1e3f0`,
  decompile de `FUN_140c1e3f0` (`+0x2c += 8`, `*param_3 = octet`), puis
  `140c1dfbf TEST byte ptr [R14 + 0x1c],0x10` / `140c1dfc4 JNZ 0x140c1e269` et, dans la branche,
  `140c1e2a7 MOV R9D,0xe` / `140c1e2ad MOV byte ptr [R14 + 0x1d],BL` / `140c1e2b1 CALL 0x140c1e9d4`
  (`FUN_140c1e9d4(flux, ., out, n)` = boucle de 3 x `R(n)`). Sans incidence sur ti=43 (la branche
  n'est prise que pour `ti` 0x23 / 0x28) ; l'enumeration ci-dessous est celle du binaire.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| mort -> `obj+0x70` | 1 | — | aucune |
| bloc lourd `FUN_140c1dd44(obj+0x74, flux)` — ordre du decompile, `p` = `obj+0x74` | variable : `FUN_14080d69c(p+0, -1)` = `R(1)`[+`R(32)`] ; **`FUN_140c1e3f0` = `R(8)` -> `p+0x1c`** (octet de drapeaux) ; 2 x `FUN_1407f2058` = `R(1)`[si 0 : +`R(5)`] -> `p+4`, `p+8` (resolution `FUN_14049746c` / `FUN_140e958c4`, 0 bit) ; `R(FUN_1406d310c(10) = 4)` -> `p+0xc` ; `R(3)` -> `p+0xe` ; `R(1)`[si 1 : `FUN_1406cf008` `R(1)`[+`FUN_14080d6f0` `R(32)`] -> `p+0x10`, `FUN_140c1e31c` -> `p+0x14`, `FUN_1406d1024` -> `p+0x18` ; si 0 : `-1`, `-1`, `-1`] ; `R(4)` -> `p+0x38` ; `FUN_1407f1f24` -> `p+0x3c` ; `FUN_1407f1e4c` -> `p+0x1e` ; `R(1)`[si 1 : `FUN_14076dc04` -> `p+0x20..+0x28` ; si 0 : `DAT_143cf0630` = (0, 0, 0)] ; **si `(p+0x1c) & 0x10`** : `R(2)` -> `p+0x1d` puis `FUN_140c1e9d4(flux, ., ., 0xe)` = 3 x `R(14)` -> `p+0x2c..+0x34`, dequantifies par la ligne `(p+0x1d)` de `DAT_143b8c6f0` (pas de `0x18` octets) ; sinon 0 bit, `p+0x1d = 0`, vecteur (0, 0, 0) ; `FUN_1424cd17c` -> `p+0x40` ; `FUN_1424cd150` -> `p+0x44` ; `FUN_14080d69c(p+0x4c, -1)` = `R(1)`[+`R(32)`] (appel de queue `140c1e01c JMP 0x14080d69c`) | `ti == 0x23` ou `ti == 0x28` | `ti` = identifiant d'archetype du record (contexte, pas un bit du flux) ; a l'interieur, toutes les branches sont sur des bits lus |
| drapeau -> `obj+0xc4` | 1 | `ti == 0x23` | idem |

- `0x23` = 35 = bipede, `0x28` = 40 = vehicule : ce sont les `ti` du depot. **Pour ti=43
  (`0x2b`) aucune des deux branches n'est prise : la grammaire est `R(1)`, 1 bit constant.**
- Concordance Go : `consumeObjectDeadState` (`components_object.go:111`) = `R(1)` seul hors
  `0x23`/`0x28` ; la forme lourde est `consumeObjectDeadStateBipedTI`. Le bloc lourd n'est pas
  releve ici : il est hors ti=43 et deja porte (ti=35 / ti=40).
- Statut : **releve** (pour ti=43).

## 3. i12 `object-scale-component`

- Descripteur `0x143d0b940` ; ecrivain **`FUN_1407dc6e4`**. Desassemblage `0x1407dc6e4..0x1407dc7d2`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| A | 1 | — | aucune |
| echelle -> `obj+0x48` | 15 | A = 0 | `FUN_1406d84b4`, n = `0xf` (`0x1407dc721`), min = 0 (`XORPS XMM2`), max = 10,0f (`[0x143cd873c]`), a = AL = 0, b = 1 |
| B | 1 | A = 0 | aucune |
| echelle cible -> `obj+0x4c` | 15 | A = 0 et B = 1 | `FUN_1406d84b4`, n = `0xf` (`0x1407dc74c`), a = 0, b = 1 ; XMM2/XMM3 **non recharges** a ce site (`0x1407dc754`) |
| duree -> `obj+0x50` | 12 | A = 0 et B = 1 | `FUN_1406d84b4`, n = `0xc` (`0x1407dc76e`), max = 120,0f (`[0x143cd8d08]`, `0x1407dc759`), a = 0, b = 1 ; puis `FUN_140475500(v, 0, 120)` (bornage, 0 bit) |
| octet -> `obj+0x54` | 5 | A = 0 et B = 1 | `FUN_1424ccc74` |

- A = 1 : `obj+0x48 = 0x3f800000` (1,0f), rien d'autre n'est lu.
- Largeurs : **1** (A=1) / **17** (A=0, B=0) / **49** (A=0, B=1).
- Concordance Go : `consumeObjectScale` (`components_object_state.go`) : `!R(1) -> R(15) ; R(1) -> R(15), R(12), R(5)` — identique.
- Statut : **releve**.

## 4. i13 `object-maximum-vitalities-component`

- Descripteur `0x143d0b8f0` ; ecrivain **`FUN_1407ee054`** -> `FUN_1407eef08(obj+0x388, flux)`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| drapeaux f -> `+0x17` | 5 | — | `FUN_1407f0354` |
| 3 x `FUN_1411b1ac0` (`+0x00`, `+0x0a`, `+0x0e`) | 3 x (1 [+12]) | `f & 4` | aucune |
| `FUN_1424cd060` (`+0x16`) | 1 | `f & 4` | aucune |
| 3 x `FUN_1411b1ac0` (`+0x02`, `+0x0c`, `+0x10`) | 3 x (1 [+12]) | `f & 8` | aucune |
| `FUN_1424cd060` (`+0x15`) | 1 | `f & 8` | aucune |
| 3 x `FUN_1411b1ac0` (`+0x04`, `+0x06`, `+0x08`) | 3 x (1 [+12]) | `f & 0x10` | aucune |
| 3 x `FUN_1424cd060` (`+0x12`, `+0x13`, `+0x14`) | 3 | — | aucune |

- Largeurs : minimum **8** (f = 0), maximum **127**. Le lecteur a 3 parametres et ne consulte
  aucun `param` (la table Go `paramByComponent` lui donne 3 : neutre ici).
- Concordance Go : `consumeObjectMaximumVitalities` — identique (`consume1411b1ac0` = `R(1)[+R(12)]`).
- Statut : **releve**.

## 5. i14 `object-dissolver-component`

- Descripteur `0x143d0bf68` ; ecrivain **`FUN_140dd9f9c`**. Desassemblage `0x140dd9f9c..`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| index -> `obj+0x3a8` | 4 | — | `R(FUN_1406d310c(0xe))` = `ceil(log2(14))` = 4 ; la constante `0xe` est FIGEE dans l'ecrivain (pas une config) |
| corps -> `obj+0x3ac..0x3b8` | 96 | index != `0xd` | `FUN_1406d676c(flux, flux, obj+0x3ac, R9D = 0x60)` (`0x140dda07b`) : copie brute |
| duree -> `obj+0x3b8` | 12 | index != `0xd` | `FUN_1406d84b4`, n = `0xc` (`0x140dda0a1`), min = 0, max = 10,0f (`[0x143cd873c]`), a = 1, b = 1 |
| drapeau -> `obj+0x3bc` | 1 | index != `0xd` | `R(1)` en ligne (chemin lent `FUN_1406d6c7c(flux, 1)`) |

- Largeurs : **4** (index = 13) / **113** (sinon).
- Concordance Go : `consumeObjectDissolver` (`R(4)`, `R(96)`, `R(12)`, `R(1)` hors etat neutre) — identique.
- Statut : **releve**.

## 6. i15 `object-low-frequency-component`

- Slot du nom `0x143d0bf30` (signature decalee, §0) ; ecrivain **`FUN_1407ef088`** (215 l.).
  Desassemblage `0x1407ef088..0x1407ef31c` ; bloc froid de la queue `0x142325e02..0x142325e4f`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| type -> `obj+0x3c4` | 2 | — | aucune |
| `thunk_FUN_140e9fadc` -> `obj+0x3c5` | 7 | type < 2 | aucune |
| -> `obj+0x3c6` | 8 | type < 2 | aucune |
| `FUN_1407ef804` -> `obj+0x3c7` | 4 | — | aucune (valeur `-1` puis `^ 0x25` a l'ecriture) |
| `FUN_1407ef724` -> `obj+0x3c8` | 6 | — | aucune (`^ 0x9e`) |
| compte N -> `obj+0x3cc` | 6 | — | aucune ; N est la valeur lue (0..63) ; le tableau vise `obj+0x3d4` par pas de 8 octets jusqu'a `obj+0x4d4` (32 entrees de capacite, borne non verifiee par le lecteur) |
| N x entree : c | 1 | — | aucune |
| . composante 0 | 12 | c = 0 | `FUN_1406d84b4`, n = `0xc` (`0x1407ef2db`), min = 0, max = XMM4 = 1,0f (`[0x143cd8374]`, `0x1407ef281`), a = 1, b = 1 |
| . composante 1 | 12 | c = 0 | `FUN_1406d84b4`, n = `0xc` (`0x1407ef2f8`), max = XMM5 = 60,0f (`[0x143cd84e4]`, `0x1407ef28c`), a = 1, b = 1 |
| . d | 1 | c = 1 | d = 0 -> (1,0 ; 0,0) ; d = 1 -> (0,0 ; 0,0) |
| 5 x `R(1)` -> `obj+0x4d4`, `+0x4d5`, `+0x4d6`, `+0x4d8`, `+0x4d9` | 5 | — | aucune |
| 2 x `R(1)` -> bits `0x80` et `0x1` de `obj+0x4dc` | 2 | — | aucune |
| `FUN_1409684dc` -> `obj+0x4e0` | 1 [+4] | `+4` si le bit vaut 0 | aucune |
| `FUN_1407ef6d4(obj+0x4e2)` : f | 1 | — | aucune |
| . `FUN_1424cd060`, `FUN_1411b1ac0`, `FUN_1424cd060` | 1, 1 [+12], 1 | f = 1 | aucune |
| `FUN_1407ef520(obj+0x4e8)` : `FUN_1407ef8e4` m | 3 | — | aucune |
| . `FUN_142af27f8`, `FUN_1424ccc74` | 2, 5 | `m & 1` et `m & 2` | aucune |
| . 3 x `FUN_1406d84b4` -> `+4`, `+8`, `+0xc` | 3 x 8 | `m & 1`, `m & 2` et `m & 4` | n = 8 (`0x1407ef585`, `0x1407ef59f`, `0x1407ef5c6`), a = 1, b = 1 ; max = 1,0f (`0x1407ef56d`), non recharge, 120,0f (`0x1407ef5b1`) |
| `FUN_1407ef4c8(obj+0x528)` : g | 1 | — | aucune ; g = 0 -> `-1`, `-1`, `-1,0f` (`[0x143cd84ec]`), 0 |
| . `FUN_1424d9a30` | 3 | g = 1 | aucune |
| . `FUN_141015740` | 32 | g = 1 | aucune |
| . `FUN_14080d69c(_, flux, +4, -1)` | 1 [+32] | g = 1 | aucune |
| . `FUN_1406d84b4` -> `+8` | 14 | g = 1 | n = `0xe` (`0x142325e41`), min = 0, max = 1,0f (`[0x143cd8374]`), a = 0, b = 1 |

- Largeur **variable** ; minimum 31 (type >= 2, N = 0, tous les bits de presence a « absent »).
  Aucune entree de config : toutes les branches sont sur des bits lus.
- Concordance Go : `consumeObjectLowFrequency` — grammaire identique, champ pour champ. Les
  commentaires Go nomment d'autres adresses pour les sous-lecteurs (`FUN_141fd7cf8`,
  `FUN_142af2a50`, `FUN_1407eddb4`, `FUN_142d55f00`, `FUN_1431a6e64`, `FUN_1431d2030`,
  `FUN_1431dad64`) que celles de cette image (`thunk_FUN_140e9fadc`, `FUN_1407ef804`,
  `FUN_1407ef724`, `FUN_1409684dc`, `FUN_1407ef6d4`, `FUN_1407ef520`, `FUN_1407ef4c8`) — les
  largeurs sont les memes. Consigne, non corrige (hors perimetre).
- Statut : **releve**.

## 7. i16 `object-physics-flags-component`

- Descripteur `0x143d0bfb8` ; ecrivain **`FUN_1407ee070`**.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| bits `0x4`, `0x8`, `0x10`, `0x20`, `0x40` de `obj+0x4dc` | 5 x 1 | — | aucune |

- Largeur **5 bits constante**. Concordance Go : `consumeObjectPhysicsFlags` (5 x `ReadBit`).
- Statut : **releve**.

## 8. i17 `object-frame-configuration-component`

- Descripteur `0x143d0be70` ; ecrivain **`FUN_1407f0534`** -> `FUN_1407f0550(obj+0x4f8, flux)`.
  Desassemblage `0x1407f0550..0x1407f061a` ; bloc froid `0x142325eb2..0x142325ed0`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| `FUN_1404d343c(obj)` | 0 | — | initialisation |
| presence p (`FUN_14080d69c(_, flux, obj+0, -1)`) | 1 | — | aucune |
| identifiant (`FUN_14080d6f0`) | 32 | p = 1 | aucune |
| compte (`FUN_1407f061c` -> `FUN_1424cd07c`) | 6 | p = 1 | aucune ; N = valeur + 1 (1..64) |
| N x bit (`FUN_1406cf008` puis `FUN_1407688b0`, pose dans le masque `obj+4`) | N x 1 | p = 1 | aucune |
| 3 x entree (`obj+0xc` a `obj+0x30`, pas de 12 octets) : | | | le nombre 3 est fige (`LEA RSI,[RAX+0x24]`, `0x1407f058a`) |
| . porte inversee (`FUN_1406d1024`) | 1 [+6] | `+6` si le bit vaut 0 | aucune |
| . g1 | 1 | — | aucune |
| . dequant | 12 | g1 = 1 | `FUN_1406d84b4`, n = `0xc` (`0x1407f060d`), min = 0, max = XMM4 = 1,0f (`[0x143cd8374]`, `0x1407f0593`), a = 0, b = 1 |
| . g2 | 1 | — | aucune |
| . dequant | 12 | g2 = 1 | `FUN_1406d84b4`, n = `0xc` (`0x142325ec2`), max = XMM5 = 60,0f (`[0x143cd84e4]`, `0x1407f059f`), a = 0, b = 1 |

- Largeurs : minimum **10** (p = 0, trois portes a 1, g1 = g2 = 0), maximum **202**.
  Aucun `param` consulte (la table Go lui donne 0 : neutre).
- Concordance Go : `consume1407f0550` (`unit_weaponstate.go`) — identique.
- Statut : **releve**.

## 9. i18 `device-position-component`

- Descripteur `0x143d0cee8` ; ecrivain **`FUN_140bef320`**.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| position -> `obj+0x538` | 14 | — | `FUN_1406d84b4(flux, flux, 0, DAT_143cd8374 = 1,0f, 0xe, 0, 1)` : `R(14)` dans `[0, 1]` |
| bit `0x10` de `obj+0x582` | 1 | — | aucune |

- Largeur **15 bits constante**. Concordance Go : `consumeDevicePosition` = `Skip(15)`
  (`components_walk_batch9.go`), dispatch `dispatch_biped.go:198` « R(14)+R(1) ».
- Statut : **releve**.

## 10. i19 `device-position-animation-name-component` (recontrole)

- Descripteur `0x143d0ce98` ; ecrivain **`FUN_1410156e4`**. Desassemblage `0x1410156e4..0x14101573d`.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| identifiant -> `obj+0x540` (`FUN_141015740`) | 32 | — | aucune |
| valeur -> `obj+0x544` | 10 | — | `FUN_1406d84b4`, n = `0xa` (`0x14101571c`), min = 0 (`XORPS XMM2`, `0x141015714`), max = 10,0f (`[0x143cd873c]`, `0x141015704`), a = 0 (`0x141015717`), b = 1 (`0x14101570f`) |

- Largeur **42 bits constante**. **Concorde** avec la note ti=43 (« `R(32)` identifiant puis
  `FUN_1406d84b4` largeur `0xa` dequantifie dans `[0, 10]` — 42 bits inconditionnels ») et
  avec `ecs_table.tsv` (`bits_typ` = 42).
- Statut : **releve** (confirme).

## 11. i20 `device-position-animation-control-component`

- Descripteur `0x143d0cfd8` ; ecrivain **`FUN_142f02d04`**.

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| bloc -> `obj+0x548..0x568` | 256 | — | `FUN_1406d676c(flux, flux, obj+0x548, 0x100)` : copie brute de 32 octets (la zone s'arrete exactement ou commence i21, `obj+0x568`) |

- Largeur **256 bits constante**. Aucune branche, aucune config. Le contenu n'est pas
  interprete ici (32 octets bruts : un tampon de nom est l'hypothese naturelle, non verifiee).
- Statut : **releve**.

## 12. i21 `device-position-group-component`

- Descripteur `0x143d0cf38` ; ecrivain **`FUN_1407f0678`**. Desassemblage `0x1407f0678..` (17 instr.).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| mot -> `obj+0x568` | 32 | — | `FUN_1406d676c(flux, flux, obj+0x568, R9D = 0x20)` (`0x1407f0686`, `0x1407f0692`) : copie brute |
| h (`FUN_1407f08bc(flux, obj+0x56c)`) | 1 | — | aucune ; h = 0 -> `obj+0x56c = 0xffff` |
| `FUN_1407f08f8` -> `obj+0x56c` | 8 | h = 1 | aucune (masque `0xff`) |

- Largeurs : **33** (h = 0) / **41** (h = 1).
- Statut : **releve**.

## 13. Surveillance d'i2 `object-forward-and-up-dynamic-precision-component` (`partiel`)

- Lecteur **`FUN_140c5f7ec(ctx, flux, record, param_4, param_5)`** (5 parametres, §0) ;
  desassemblage `0x140c5f7ec..` (56 instr.). Mode C=1 : **`FUN_142e29bac`** (89 instr.,
  desassemblage `0x142e29bac..0x142e29cf3`). Corps : `FUN_140c5f938` (B = 0, desassemblage
  `0x140c5f938..0x140c5f9c5`) et `FUN_140c5f8a8` (B = 1).

Tete (`FUN_140c5f7ec`) :

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| A | 1 | — | aucune ; A = 1 -> mode = 2, B force a 0 |
| B | 1 | A = 0 | aucune ; mode = 0 |
| C | 1 | **`param_5 > 1`** | **`param_5`** : 5e argument du lecteur, fourni par le dispatcheur, PAS ecrit dans le record. C = 1 -> mode = 1 |

Selection de la charge utile (identique dans les deux corps ; collee de `FUN_140c5f938`) :

| Test (ordre du code) | Charge utile | Bits |
|---|---|---|
| `CMP byte ptr [0x145121140],0x1 ; JNZ ; CMP R9D,0x1 ; JL` : **`DAT_145121140 == 1` et mode < 1** | `FUN_142e29bac` | 31 ou 61 |
| `SUB R9D,0x1 ; JZ` : mode == 1 | `FUN_142e29bac` | 31 ou 61 |
| `CMP R9D,0x1 ; JZ` : mode == 2 | `FUN_1406d676c(flux, ., fwd, 0x60)` + `FUN_1406d676c(flux, ., up, 0x60)` (`0x140c5f98a`, `0x140c5f998`) | 192 |
| sinon, B = 0 | `FUN_140c5fa84` puis `FUN_140c5f9c8` (0 bit) | 9 ou 28 |
| sinon, B = 1 | `FUN_14076e744(local, flux, *param_4)` puis `FUN_140c5f9c8` (0 bit) | 2 a 26 |

Les trois charges utiles a bits :

| Fonction | Grammaire | Largeur |
|---|---|---|
| `FUN_142e29bac` (mode 1) | F = `R(1)` ; F = 0 -> `R(30)` (`+0x2c += 0x1e`) puis `FUN_1406d8288(v, out, 0x1e)` (0 bit) ; F = 1 -> vecteur (0, 0, 1) de `0x14472a654` ; puis `FUN_1406d84b4`, n = `0x1e` (`0x142e29cce`), min = -pi (`[0x143cd8920]`), max = +pi (`[0x143cd8918]`), a = 0, b = 0 ; puis `FUN_1406d8678` (0 bit) | 31 (F = 1) / 61 (F = 0) |
| `FUN_140c5fa84` (mode 0, B = 0) | p = `R(1)` ; p = 0 -> `R(19)` (`+0x2c += 0x13`, decoupe par `DAT_144708568` / `DAT_14470856c`, 0 bit) ; puis `R(8)` inconditionnel | 9 / 28 |
| `FUN_14076e744` (mode 0, B = 1) | g1 = `R(1)` ; g1 = 0 -> g2 = `R(1)` ; g2 = 0 -> `R(19)` + `FUN_1406d8228` (0 bit) ; g2 = 1 -> `R(4)` + `R(4)` ; puis t = `R(1)` ; t = 1 -> `R(4)` | 2 a 26 |

Ce qui rend i2 `partiel`, releve dans l'executable :

1. **Une branche sur un global de processus, pas sur un bit du flux** : `DAT_145121140` est un
   octet de mode (0..3) pose par `FUN_140a93ec8(param)` — 0 ; 1 via `FUN_142b5c658`
   (installe `PTR_FUN_143dedb48` dans le singleton `DAT_145120f00`) ; 2 (`PTR_FUN_143dedf48`) ;
   3. Il compte 200 references (198 lectures, 2 ecritures : `0x140a93ee6`, `0x142b5c6ab`). La
   valeur 1 force le mode 1 meme quand C n'est pas lu. Le port le modelise deja par
   `FullPrecision` du profil (`components_movement.go:114-127`, faux en retail) — mais
   `decodeObjectForwardAndUpDynPrec` (`components_dynprec_orientation.go:103`) ne consulte PAS
   ce predicat : si `FullPrecision` etait vrai, la charge utile lue serait fausse.
2. **Une branche sur `param_5`**, valeur de contexte fournie par le dispatcheur. Le port la
   tabule : `paramByComponent[compForwardUpDynPrec] = 2` (`component_param4.go`, mesure du
   2026-09-03 sur `0d76e8f1` / `fccc61cd`). Avec 2, le bit C EST lu.
3. `param_4` (reference a la valeur precedente) n'alimente que la math de B = 1 : 0 bit.

Toutes les largeurs sont donc **decidables hors ligne** des que le profil porte (1) et (2). Le
statut `partiel` de la table tient a la phrase « mode C=1 (`FUN_142e29bac`) NON porte », que le
code contredit : `decodeObjectForwardAndUpDynPrec` appelle `consumeFwdUpDynPrecConfig`
(`R(1)[+R(30)] + R(30)`, `components_dynprec_orientation.go:159-171`) pour `Mode == 1`, et
`paramByComponent` vaut 2 (la godoc au-dessus de `consumeObjectForwardAndUpDynPrec`, l. 77-83,
dit encore « paramForComponent rend 1 » et « PAS porte »). **Doc et table en retard sur le
code** ; a re-statuer par le pilote — rien n'a ete modifie ici. Reste : la branche
`DAT_145121140 == 1` non modelisee dans ce lecteur (point 1).

- Statut : **partiel** au sens de la table, pour les deux raisons de contexte ci-dessus ;
  grammaire **entierement relevee**.

---

## 14. Ce que le port demandera

1. **i19, i20, i21 sont des grammaires a porter telles quelles**, sans entree de profil :
   42 bits ; 256 bits bruts ; 32 bits bruts + `R(1)[+R(8)]`. Elles rejoignent les 19 autres
   `device-*` (i22 a i40, groupes suivants) : le plan dit que porter i19 seul ne ferme aucun
   record, le gate reste la fermeture de ti=43 (0/4 106 -> 4 106/4 106 sur les 7 bobines).
2. **i11 a i18 concordent 8/8 avec le code Go** existant (`consumeObjectDeadState` en `R(1)`
   pour ti=43, `consumeObjectScale`, `consumeObjectMaximumVitalities`, `consumeObjectDissolver`,
   `consumeObjectLowFrequency`, `consumeObjectPhysicsFlags`, `consume1407f0550`,
   `consumeDevicePosition`). Rien a porter ; mais les colonnes `grammar` / `bits_typ` de
   `ecs_table.tsv` sont vides pour ces huit lignes et peuvent etre remplies depuis cette note
   dans le commit du lot (garde-rails `ecs_table_guard_test.go`).
3. **i11 branche sur l'identifiant d'archetype** (`ctx+0x30`) : ce n'est pas une entree de
   profil (le decodeur connait le `ti` du record), mais c'est une grammaire par archetype —
   `R(1)` pour ti=43, forme lourde pour 35 et 40.
4. **Deux entrees de profil pour i2** : `param_5` (aujourd'hui tabule a 2) et le mode de
   processus `DAT_145121140` (aujourd'hui `FullPrecision`, faux en retail). La premiere est
   deja consultee par le lecteur d'i2, la seconde ne l'est pas : le port devra soit la brancher
   dans `decodeObjectForwardAndUpDynPrec`, soit ecrire pourquoi elle est ignoree. Et
   re-statuer la ligne `partiel` de la table (§13).
5. **Le contrat du dequantifieur `FUN_1406d84b4`** (§1) suffit a lire toutes les largeurs de
   ce groupe : la largeur est `[RSP+0x20]` chez l'appelant, jamais dans XMM. Les bornes et les
   drapeaux a/b ne servent qu'a la VALEUR : le port qui ne fait que sauter des bits n'en a pas
   besoin ; celui qui publie une valeur (position de dispositif, echelle, duree) les a ici.
6. **Aucune largeur de ce groupe ne depend de la carte ni du mode de jeu** : tout est constante
   d'ecrivain, bit lu, ou (i2) contexte de processus / de dispatcheur. Rien a mesurer par
   statistique.
7. **Ecarts de documentation** a signaler au pilote, non corriges (zero fix hors perimetre) :
   les adresses de sous-lecteurs citees dans les commentaires de `consumeObjectLowFrequency`
   ne sont pas celles de cette image (§6) ; la godoc d'i2 et `ecs_table.tsv` disent « NON
   porte » pour un chemin que le code parcourt (§13).

## 15. Appels Ghidra en echec ou inutilisables

Trois : `disassemble_function` sur `0x1407ef4c8` et sur `0x1407f0550` a deborde de la fonction
(6 909 894 instructions, 219 Mo chacun — bornes de fonction non definies dans l'image ; le corps
utile a ete extrait par fenetre d'adresses, puis les fichiers purges) ; `get_function_callers`
sur `0x140c5f7ec` a rendu « No callers found for function: null » (le lecteur n'est atteint
que par donnees : `0x143e2baa0` et `0x14543ab98`). Tout le reste a repondu.
