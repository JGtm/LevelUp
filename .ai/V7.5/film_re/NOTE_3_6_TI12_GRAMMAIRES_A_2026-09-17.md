# ti=12 (navpoint), groupe ti12a (i0 a i12) — grammaires relevees (2026-09-17)

> Preparation du lot 3.6, item 3.6.b (volet `ti=12`), SANS production : aucun film decode,
> aucun test Go, aucune ligne de production. Source : `HaloInfinite.exe` par Ghidra en
> LECTURE SEULE (HTTP direct `127.0.0.1:8089`, rien renomme, rien commente). Methode :
> `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` ; note d'archetype :
> `NOTE_3_6_TI12_2026-09-16.md` (les 13 adresses de depart viennent d'elle, sauf `i0`
> qui vient de la note de methode et sert de contre-epreuve).
>
> Instrument / Ghidra lecture seule. Archetype : `ti=12 managed-navpoint`. Groupe : `ti12a`
> (composants d'index 0 a 12). Chiffres et adresses colles depuis Ghidra.

---

## 0. Ce que cette note ajoute

- **13 grammaires sur 13** (i0 a i12), toutes `releve`. Cinq sont des champs plats (`R(8)`,
  `R(32)`, `2 x R(7)`, `R(17)` x 2), une est une liste taguee deja connue du depot sous une
  autre etiquette (`i9`), cinq partagent UN SEUL bloc « jeu de filtres » a tags polymorphes
  (`i2` a `i6`) dont les 16 tags sont enumeres ici, decidables hors ligne.
- **Le `param_4` du moteur est RESOLU** : c'est la petite fonction du slot `+0x10` du
  descripteur (`MOV EAX, k ; RET`). Calibre 3/3 sur des composants dont le depot a MESURE la
  valeur en capture live (§2). Pour ce groupe : `i2 -> 3`, `i3..i6 -> 2`, tous les autres
  `-> 1`. Les portes de version des filtres (`2 < param_4`, `1 < param_4`) sont donc
  DECIDEES pour cet executable, et la mise en page qu'elles selectionnent est exactement
  celle que le serialiseur du meme build ecrit (§2, contre-epreuve).
- **Terminologie** : la fonction a `descripteur + 0x40` (celle que le plan et la note de
  methode appellent « ecrivain ») CONSOMME des bits (chargement `bswap` depuis le tampon,
  extraction des bits hauts, ecriture dans la structure) : c'est le deserialiseur, ce que
  `ecs_table.tsv` nomme `deser_addr`. Le slot `+0x28` est le SERIALISEUR (il pousse des
  bits dans l'accumulateur et vide des octets vers le tampon). Les deux sont symetriques et
  le `+0x28` a servi de contre-epreuve pour i0, i1, i2, i3, i5, i6, i7, i8, i9, i10, i11, i12.
- **Une seule dependance de configuration** dans tout le groupe : la largeur d'une reference
  d'entite du domaine 0 (`FUN_1406d3140`, tags 6, 12, 13, 14 des filtres), deja connue du
  depot (`NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md`, `frame_records.go` « IDLowBits »). Tout le
  reste est lu dans le flux ou constant dans l'executable.

## 1. Conventions et lecteurs communs

`R(n)` = lecture de `n` bits. Le flux (`param_2` des lecteurs) porte son compteur de bits a
`+0x2c`, l'accumulateur a `+0x30`, la reserve a `+0x38`, le curseur d'octets a `+0x40`, la
fin a `+0x10` ; chaque champ apparait deux fois dans le desassemblage (chemin rapide et
chemin lent `FUN_1406d6c7c(flux, n)`), on compte les champs.

| Lecteur | Ce qu'il lit | Releve |
|---|---|---|
| `FUN_1406cf008(flux)` | `R(1)` | note de methode |
| `FUN_1406d6c7c(flux, n)` | `R(n)`, chemin lent | decompile : `+0x2c += param_2`, rend `n` bits |
| `FUN_14080dec4(flux, etiquette, dest)` | `R(32)` | decompile : `+0x2c += 0x20`, `*dest = valeur` ; l'etiquette (« navpoint-sub-type », « text », « textStringId », « string_id », « watched-marker ») n'est pas lue |
| `FUN_14109414c(flux, _, dest)` | `R(8)` | decompile : `+0x2c += 8` |
| `FUN_140dbe598(flux, _, dest)` | `R(4)` | decompile : `+0x2c += 4` |
| `FUN_1407ef804(flux, _, dest)` et `FUN_140968284(flux, _, dest)` | `R(4)`, stocke `valeur - 1` | decompiles identiques : `*dest = bVar4 - 1` |
| `FUN_1410d9088(flux)` | `R(7)`, rend `valeur - 1` | decompile : `return uVar4 - 1` |
| `FUN_1407f2058(flux)` | porte INVERSEE `R(1)` ; si 0 : `R(5)` ; sinon rend `0xffffffff` | decompile (cf. sondage E2) |
| `FUN_142b67f08(flux)` | porte INVERSEE `R(1)` ; si 0 : `R(13)` (`0xd`) ; sinon `0xffffffff` | decompile : meme forme que `FUN_1407f2058`, largeur 13 |
| `FUN_1406d3140(_, flux, domaine, out)` | reference d'entite : `R(w)` + base puis `R(2)` de generation ; `w = FUN_1406d310c(cardinal)` avec `cardinal = DAT_1451f98d4[2*domaine]` (base `DAT_1451f98d0[2*domaine]`) si `DAT_144706104 != 0`, sinon `DAT_144706100` (base 0) ; si `w < 1` seul le `R(2)` est lu ; le `R(1)` supplementaire n'existe que pour `domaine == 1` | decompile ; **RUNTIME = config** |
| `FUN_1406d84b4(flux, _, min, max, n, f6, f7)` | `R(n)` dequantifie, voir ci-dessous | desassemblage `1406d84b4..1406d8673` |

**`FUN_1406d84b4`, la dequantification** (aucune lecture ne depend des drapeaux ; seule la
formule change) — `min` en `XMM2`, `max` en `XMM3`, `n` a `[RSP+0x28]`, `f6` a
`[RSP+0x30]` (`SIL`), `f7` a `[RSP+0x38]` :

- `pas_total = f6 ? 2^n - 1 : 2^n` (`SHL EDI,CL` puis `LEA ECX,[RDI-1]` / `CMOVZ`).
- `f7 == 0` : `v = min + (q + 0,5) * (max - min) / pas_total` (`DAT_143cd84b0 = 0x3f000000 = 0,5f`).
- `f7 != 0` : `q == 0 -> min` ; `q == pas_total - 1 -> max` ; sinon
  `v = min + (q - 1 + 0,5) * (max - min) / (pas_total - 2)`.
- `f6 != 0` et `2q == pas_total - 1` : `v = (min + max) * 0,5` exact
  (`DAT_143cd8910 = 0x3fe0000000000000 = 0,5`, calcul en double).

## 2. `param_4` : d'ou il vient, et ce qu'il vaut ici (RESOLU)

Les deserialiseurs `+0x40` ont la signature `(objet, flux, record, param_4)` et lisent la
donnee du composant a `*(record + 0x10)`. Cinq d'entre eux (`i2` a `i6`) branchent sur
`param_4` : `2 < param_4` pour `i2` (`FUN_140dbde1c`), `1 < param_4` pour `i3`, `i4`, `i5`,
`i6`. Le depot modelise ce parametre dans `component_param4.go` (`paramByComponent`,
valeurs MESUREES en capture live sur `ti=35`, defaut 1) mais ne sait pas d'ou il vient.

**Il vient du slot `+0x10` du descripteur**, qui est une fonction de six octets :

| Composant | Descripteur | `+0x10` | Desassemblage | `param_4` |
|---|---|---|---|---|
| `i0`, `i1`, `i7`, `i8`, `i9`, `i10`, `i11`, `i12` | (leurs descripteurs, §4..) | `0x14117b4a0` | `MOV EAX, 0x1 ; RET` | **1** |
| `i2` | `0x143d07fd0` | `0x14117e0e0` | `MOV EAX, 0x3 ; RET` | **3** |
| `i3`, `i4`, `i5`, `i6` | `0x143d07f80`, `0x143d07f30`, `0x143d083e0`, `0x143d08390` | `0x141179610` | `MOV EAX, 0x2 ; RET` | **2** |

**Calibration 3/3** sur trois composants `ti=35` dont `component_param4.go` porte la valeur
mesuree (chaine -> accesseur -> slot `+0x18` -> descripteur -> `+0x10`) :

| Composant (mesure du depot) | Chaine | Accesseur | Descripteur | `+0x10` | Lu |
|---|---|---|---|---|---|
| `object-maximum-vitalities-component` (3) | `0x143c99380` | `0x14064c6b0` | `0x143d0b8f0` | `0x14117e0e0` -> `MOV EAX, 0x3` | **3** |
| `unit-malleable-property-component` (4) | `0x143c961e0` | `0x141175670` | `0x143d06cd0` | `0x140c85020` -> `MOV EAX, 0x4` | **4** |
| `object-frame-configuration-component` (0) | `0x143c991a0` | `0x14064c670` | `0x143d0be70` | `0x1405f0ac0` -> `XOR EAX, EAX` | **0** |

**Contre-epreuve par le serialiseur du meme build** : le slot `+0x28` de `i2`
(`0x142edb148 : MOV RCX,[R8+0x30] ; XOR R8D,R8D ; ADD RCX,0x8 ; JMP FUN_141f19604`) ecrit,
sans aucune branche de version, un drapeau de **1 bit**, **aucun** champ de 4 bits par
filtre, et des entrees d'ordre de **3 bits** — c'est exactement la mise en page que le
lecteur prend quand `2 < param_4`, donc avec `param_4 = 3`. Meme constat pour `i3`
(`0x142edb168 -> FUN_141f1b3a8` : `FUN_1406d49c4` = ecriture de 1 bit, ordre sur 3 bits) et
`i5`/`i6` (`0x142edb158`, `0x142edae14 -> FUN_142c7023c` : drapeau ecrit sur 1 bit). Les
branches `else` des lecteurs (`R(32)` de drapeau, `R(4)` par filtre, `R(2)` d'ordre) sont la
mise en page d'anciennes versions du composant, que cet executable ne produit plus.

**Limite honnete** : le dispatcher qui appelle `+0x40` en lui passant `+0x10` n'a pas ete
localise (celui que `HANDOFF_FRAME_DECODER_L3.md` nomme, `FUN_14076cb60`, appelle
`[vtable+0x28]` a cinq arguments et `[vtable+0x48]` a trois sur des OBJETS dont la vtable
n'a pas la forme de ces descripteurs — pour cette famille, `+0x28` est le serialiseur et
`+0x48` vaut `XOR AL,AL ; RET`). Le cablage `+0x10 -> param_4` repose donc sur la
calibration 3/3 ci-dessus et sur la coherence lecteur/serialiseur, pas sur une lecture du
site d'appel.

## 3. Le bloc « jeu de filtres » commun a `i2`..`i6` : `FUN_140dbe400(dest, flux, v)`

`v` est le booleen de version passe par le deserialiseur du composant (`1 < param_4` ou
`2 < param_4`). La structure : quatre filtres de `0x40` octets a `dest + 0x40*i`, masque a
`dest + 0x100`, drapeau a `dest + 0x104`.

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | masque des filtres presents (`FUN_140dbe598`) -> `+0x100` | 4 | toujours | aucune |
| 2 | drapeau -> `+0x104` | **`v ? 1 : 32`** (`iVar5 = (-(v != 0) & 0xffffffe1) + 0x20`) | toujours | `param_4` (v) |
| 3 | pour chaque bit `i` (0..3) pose dans le masque : tag du filtre | 4 (`FUN_1406d6c7c(flux, 4)` sur le chemin lent) | bit `i` pose | aucune |
| 4 | charge du filtre `i` selon le tag : `FUN_141e98e10(tag, {flux, dest + 0x40*i})` | table ci-dessous | tag != 0 | tags 6, 12, 13, 14 : largeur de reference d'entite (domaine 0) |

Le rappel `FUN_141e98e10` traite les tags 0..5, `FUN_141e98c70` les tags 6..11,
`FUN_141e98f90` les tags 12..14 ; tout autre tag tombe sur `FUN_1411c8f80` (ne revient pas).
Pour tout tag non nul : construction de l'objet filtre (vtable par tag), **`R(1)`** ecrit a
`objet + 0x08`, puis appel virtuel **`vtable + 0x10`**`(objet, flux)` :

| Tag | vtable | `vtable + 0x10` | Charge lue apres le `R(1)` commun | Bits (hors le `R(4)` de tag) |
|---|---|---|---|---|
| 0 | — | — | rien (`objet + 0x38 = 0`) | 0 |
| 1 | `0x143c2ae08` | `FUN_141e9d6d0` | `R(1)` -> `+0x10` | 1 + 1 |
| 2 | `0x143c3df58` | `FUN_141e9d120` | 32 x `R(1)`, bit `i` du masque `+0x10` | 1 + 32 |
| 3 | `0x143c4ee88` | `0x141e9d660` -> `FUN_1407ef804` | `R(4)`, valeur - 1 | 1 + 4 |
| 4 | `0x143c4ee58` | `FUN_141e9d670` | `R(9)` | 1 + 9 |
| 5 | `0x143c4ebd0` | `FUN_141e9a9c0` | `R(3)` = `c` ; `c` x (`vtable + 0x28` = `0x141e9ca50` -> `FUN_142b67f08` : porte inversee `R(1)` ; si 0 : `R(13)`) | 1 + 3 + c x (1 ou 14) |
| 6 | `0x143c4eb70` | `FUN_141e9aa60` | `R(4)` = `c` ; `c` x (`vtable + 0x28` = `FUN_141e9c9c0` : `R(1)` ; si 1 : `FUN_1406d3140(_, flux, 0, out)` = `R(w0)` + `R(2)`) | 1 + 4 + c x (1 ou 1 + w0 + 2) ; **w0 = config** |
| 7 | `0x143c4ee70` | `0x141e9d0a0` -> `FUN_140968284` | `R(4)`, valeur - 1 | 1 + 4 |
| 8 | `0x143c4ef40` | `0x141e9d0b0` -> `FUN_14109414c` | `R(8)` | 1 + 8 |
| 9 | `0x143c4ef10` | `FUN_141e9d0c0` | `R(32)` | 1 + 32 |
| 10 | `0x143c4ef28` | `FUN_141e9d5e0` | `R(4)` (`FUN_1407ef804`) ; `R(4)` (`FUN_140968284`) ; `R(32)` | 1 + 40 |
| 11 | `0x143c4eef8` | `FUN_141e9d1a0` | `FUN_1407f2058` : porte inversee `R(1)` ; si 0 : `R(5)` | 1 + (1 ou 6) |
| 12 | `0x143c4eec8` | `FUN_141e9d440` | `R(1)` ; si 1 : `FUN_1406d3140(_, flux, 0, out)` ; puis `R(32)` | 1 + 1 + (0 ou w0 + 2) + 32 ; **w0 = config** |
| 13 | `0x143c4eee0` | `FUN_141e9cf50` | `R(1)` ; si 1 : `FUN_1406d3140(_, flux, 0, out)` ; puis `R(32)` (etiquette « watched-marker ») | idem 12 |
| 14 | `0x143c4ef58` | `FUN_141e9d260` | `R(1)` ; si 1 : `FUN_1406d3140(_, flux, 0, out)` ; puis `R(32)` ; puis `R(1)` | 1 + 1 + (0 ou w0 + 2) + 32 + 1 |
| 15 | — | `FUN_1411c8f80` | ne revient pas | valeur invalide dans un film sain |

Les quatre sites d'appel de `FUN_1406d3140` (tags 6, 12, 13, 14) posent `XOR R8D,R8D` :
**domaine 0** (`141e9ca15`, `141e9d49e`, `141e9cf9e`, `141e9d2be`). Sa largeur `w0` est
`ceil(log2(cardinal du domaine 0))`, valeur de RUNTIME (`DAT_1451f98d4[0]` si
`DAT_144706104`, sinon `DAT_144706100`) : entree de profil, jamais une constante ; le depot
porte deja cette primitive (« IDLowBits », `NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md` l. 181).

Le serialiseur symetrique de ce bloc est `FUN_142c7023c` (ecrit le masque par
`FUN_142c70090` = 4 bits, le drapeau sur 1 bit, puis par filtre le tag sur 4 bits et
`FUN_141e99630`) — c'est la fonction que le depot cite pour `ti=11 i4 interaction-filter`
(`components_managed_objective.go` l. 62) en la prenant pour le lecteur : le lecteur
symetrique est `FUN_140dbe400`, et la table de tags ci-dessus est la sienne.

## 4. `i0 managed-navpoint-sub-type-component` (contre-epreuve du port existant)

- Descripteur `0x143d07e98` ; deserialiseur `FUN_1410e0cac` -> `FUN_14080dec4(flux, "navpoint-sub-type", *(record + 0x10))`.
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| sous-type (etiquette « navpoint-sub-type ») -> `+0x00` | 32 | toujours | aucune |

- Largeur totale : **32**. Concorde avec `ecs_table.tsv` (`R(32)`, `dispatch_player.go`).
- Serialiseur `+0x28` `0x142edb0dc` : `FUN_1407edaf4(flux, etiquette 0x14371b980, [dest])`, 32 bits.
- Statut : **releve** (deja porte).

## 5. `i1 managed-navpoint-flags-component`

- Descripteur `0x143d08020` ; deserialiseur `FUN_141094130` -> `FUN_14109414c(flux, flux, *(record + 0x10) + 4)`.
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| drapeaux -> `+0x04` | 8 | toujours | aucune |

- Largeur totale : **8**. Confirme la note d'archetype (`ADD dword ptr [+0x2c], 0x8` aux deux chemins d'un meme champ).
- Serialiseur `+0x28` `0x142edaee8` : `FUN_142c74154(flux, _, dest + 4)`.
- Statut : **releve**.

## 6. `i2 managed-navpoint-visibility-distance-filters-component`

- Descripteur `0x143d07fd0` ; deserialiseur `FUN_140dbde1c` -> `FUN_140dbde44(dest = *(record + 0x10) + 8, flux, 0, v = (2 < param_4))`.
- `param_4 = 3` (`+0x10 = 0x14117e0e0`) donc **`v = 1`**.
- Le corps de `FUN_140dbde44` est scinde chaud/froid (`140dbde44..140dbdefc` et `142414b82..142414d46`) ; Ghidra lui attribue un corps aberrant (`140dbde44 - 142414d4a`) : le desassemblage a ete fait par plages.

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | bloc « jeu de filtres » `FUN_140dbe400(dest, flux, v)` | §3 (masque 4, drapeau `v ? 1 : 32`, par filtre 4 + charge) | toujours | `param_4` ; w0 pour les tags 6/12/13/14 |
| 2 | distance A (`FUN_1411b4e6c` = `FUN_1406d84b4(flux, flux, DAT_143cd84ec = -1,0f, DAT_143cd8348 = 1000,0f, 0x10, 0, 1)`) -> `+0x108` | 16 | toujours | aucune |
| 3 | distance B, idem -> `+0x10c` | 16 | toujours | aucune |
| 4 | pour chaque bit `i` du masque : paire de distances (`FUN_140dbdf00(_, flux, _, dest + 0x110 + 8i)` = 2 x `FUN_1411b4e6c`) — site `142414b82..142414b8f` | 2 x 16 | bit `i` pose | aucune |
| 5 | pour chaque bit `i` du masque : octet legacy -> `+0x130 + i` | 4 | bit `i` pose **et `v == 0`** (`TEST R12D,R12D ; JNZ` a `142414b94`) | `param_4` (jamais lu avec `param_4 = 3`) |
| 6 | pour `j` de 0 a `K - 1`, `K` = nombre de bits poses : ordre -> `+0x134 + j` | **`v ? 3 : 2`** (`EBP = 2 + (v != 0)`, `142414c6b..142414c81`) | `K > 0` | `param_4` |
| 7 | sentinelle `0xff` a `+0x134 + K` si `K < 4` | 0 (ecriture memoire, aucune lecture) | — | — |

- Dequantification des distances (`f7 = 1`) : `q = 0 -> -1` ; `q = 65535 -> 1000` ; sinon
  `v = -1 + (q - 1 + 0,5) * 1001 / 65534` (pas 0,0152745 ; `q = 1 -> -0,99236`, `q = 65534 -> 999,99236`).
- Largeur totale : variable ; **minimum 37 bits** (masque vide : 4 + 1 + 32) avec `v = 1`.
- Serialiseur `+0x28` `0x142edb148 -> FUN_141f19604` : `FUN_142c7023c` (bloc), `FUN_142c8ae0c` (= 2 x `FUN_142c8af90` -> `FUN_1406d22c0`, la paire), par filtre `FUN_142c8ae0c`, puis `K` x 3 bits ; **pas de champ de 4 bits par filtre, drapeau de 1 bit** : mise en page `v = 1`.
- Statut : **releve**.

## 7. `i3 managed-navpoint-visible-offscreen-filters-component`

- Descripteur `0x143d07f80` ; deserialiseur `FUN_140dbdf80` -> `FUN_140dbdfd8(dest = *(record + 0x10) + 0x140, flux, record, v = (1 < param_4))`.
- `param_4 = 2` (`+0x10 = 0x141179610`) donc **`v = 1`**.

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | bloc « jeu de filtres » `FUN_140dbe400(dest, flux, v)` | §3 | toujours | `param_4` ; w0 (tags 6/12/13/14) |
| 2 | drapeau (`FUN_1406cf008`) -> `+0x108` | 1 | toujours | aucune |
| 3 | pour chaque bit `i` du masque : drapeau du filtre (`FUN_1406cf008`) -> `+0x109 + i` | 1 | bit `i` pose | aucune |
| 4 | pour chaque bit `i` du masque : octet legacy -> `+0x10d + i` | 4 | bit `i` pose **et `v == 0`** | `param_4` (jamais lu avec `param_4 = 2`) |
| 5 | pour `j` de 0 a `K - 1` : ordre -> `+0x111 + j` | **`v ? 3 : 2`** (`iVar3 = (param_4 != 0) + 2`) | `K > 0` | `param_4` |
| 6 | sentinelle `0xff` a `+0x111 + K` si `K < 4` | 0 | — | — |

- Largeur totale : variable ; **minimum 6 bits** (4 + 1 + 1) avec `v = 1`.
- Serialiseur `+0x28` `0x142edb168 -> FUN_141f1b3a8` : `FUN_142c7023c`, `FUN_1406d49c4` (1 bit), par filtre `FUN_1406d49c4`, `K` x 3 bits : mise en page `v = 1`.
- Statut : **releve**.

## 8. `i4 managed-navpoint-can-be-occluded-filters-component`

- Descripteur `0x143d07f30` — **forme courte** : ses slots `+0x00`/`+0x08` (`0x143d05d38`,
  `0x143d05d70`) sont les deux dernieres entrees de la table de noms des huit
  `visual-state-groups` (`0x143d07f00..0x143d07f38`) ; la signature de famille commence a
  `+0x10 = 0x141179610`, `+0x18 = 0x141177da0` (accesseur de nom), `+0x20 = 0x1404ab600`,
  `+0x28 = 0x142edae04`, `+0x30 = 0x1411c8f80`, `+0x38 = 0x14076ce9c`, `+0x40 = 0x140dbdfac`,
  `+0x48 = 0x1404ab600`.
- Deserialiseur `FUN_140dbdfac` -> `FUN_140dbdfd8(dest = *(record + 0x10) + 0x258, flux, record, v = (1 < param_4))` ; `param_4 = 2` donc **`v = 1`**.
- Grammaire : **identique a `i3`** (§7), seule la destination change (`+0x258` au lieu de `+0x140`).
- Largeur totale : variable ; minimum 6 bits.
- Serialiseur `+0x28` `0x142edae04 : MOV RCX,[R8+0x30] ; ADD RCX,0x258 ; JMP FUN_141f1b3a8`.
- Statut : **releve**.

## 9. `i5 managed-navpoint-visibility-filter-component`

- Descripteur `0x143d083e0` ; deserialiseur `FUN_140dbe194` -> `FUN_140dbe400(dest = *(record + 0x10) + 0x370, flux, v = (1 < param_4))`.
- `param_4 = 2` donc **`v = 1`**.

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | masque (`FUN_140dbe598`) -> `+0x100` | 4 | toujours | aucune |
| 2 | drapeau -> `+0x104` | `v ? 1 : 32` = **1** | toujours | `param_4` |
| 3 | pour chaque bit `i` : tag `R(4)` puis charge (table §3) | 4 + charge | bit `i` pose | w0 pour les tags 6/12/13/14 |

- Largeur totale : variable ; **minimum 5 bits** (masque vide).
- Serialiseur `+0x28` `0x142edb158 -> FUN_142c7023c` (drapeau ecrit sur 1 bit).
- Statut : **releve**.

## 10. `i6 managed-navpoint-docking-filter-component`

- Descripteur `0x143d08390` ; deserialiseur `FUN_140dbdf34` -> `FUN_140dbe400(dest = *(record + 0x10) + 0x478, flux, v = (1 < param_4))` ; `param_4 = 2` donc **`v = 1`**.
- Grammaire : **identique a `i5`** (§9), destination `+0x478`.
- Serialiseur `+0x28` `0x142edae14 : MOV RCX,[R8+0x30] ; ADD RCX,0x478 ; JMP FUN_142c7023c`.
- Statut : **releve**.

## 11. `i7 managed-navpoint-docking-order-component`

- Descripteur `0x143d08340` ; deserialiseur `FUN_142ed5050` (lecture en ligne).
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| ordre d'accostage -> octet `+0x6f0` | 8 | toujours | aucune |

- Largeur totale : **8**.
- Serialiseur `+0x28` `0x142edae40` : ecriture en ligne de l'octet `[dest + 0x6f0]` (`MOVSX RCX, byte ptr [RAX + 0x6f0]`).
- Statut : **releve**.

## 12. `i8 managed-navpoint-docking-group-name-component`

- Descripteur `0x143d082f0` ; deserialiseur `FUN_142ed5028` -> `FUN_14080dec4(flux, "navpoint-docking-group-name", *(record + 0x10) + 0x6f4)`.
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| nom du groupe (identifiant de chaine) -> `+0x6f4` | 32 | toujours | aucune |

- Largeur totale : **32**.
- Serialiseur `+0x28` `0x142edae24` : `FUN_1407edaf4(flux, etiquette 0x143e0b570, [dest + 0x6f4])`.
- Statut : **releve**.

## 13. `i9 managed-navpoint-formatted-text-component`

- Descripteur `0x143d08520` ; deserialiseur `FUN_1410e7b90`.
- `param_4 = 1` (non utilise).

| # | Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|---|
| 1 | nombre d'entrees `N` -> `+0x580` (boucle si `0 < (int)N`) | 8 | toujours | aucune |
| 2 | pour chaque entree `k` (0x2c octets a partir de `+0x588`) : `textStringId` (`FUN_14080dec4`) -> `+0x588 + 0x2c k` | 32 | `k < N` | aucune |
| 3 | presence du texte formate (`FUN_14080b034`, `FUN_1406cf008`) | 1 | `k < N` | aucune |
| 4 | identifiant « text » (`FUN_14080dec4`) | 32 | presence = 1 | aucune |
| 5 | nombre d'arguments `M` | 3 | presence = 1 | aucune |
| 6 | pour chaque argument `m < M` (`FUN_1407f0ebc`) : tag | 3 | presence = 1 | aucune |
| 6.0 | tag 0 : rien | 0 | — | — |
| 6.1 | tag 1 : `FUN_1407f2058` = porte inversee `R(1)` ; si 0 : `R(5)` (index d'entite) | 1 ou 6 | tag = 1 | aucune |
| 6.2 | tag 2 : `FUN_142c70cd0` = `R(1)` ; si 0 : `R(32)` brut (`FUN_1406d676c(flux, _, out, 0x20)`) ; si 1 : `R(24)` dequantifie `FUN_1406d84b4(flux, _, DAT_143d13518 = -6000,0f, DAT_143d1334c = 6000,0f, 0x18, 1, 0)` | 1 + (32 ou 24) | tag = 2 | aucune |
| 6.3 | tag 3 : `R(32)` (etiquette « string_id ») | 32 | tag = 3 | aucune |
| 6.4 | tags 4..7 : `R(32)` (`FUN_142c70be0`) | 32 | tag >= 4 | aucune |

- Dequantification du tag 2 (`f6 = 1`, `pas_total = 16777215`, pas 0,00071525578) :
  `v = -6000 + (q + 0,5) * 12000 / 16777215` ; `q = 8388607 -> 0` exact.
- Largeur totale : variable ; **minimum 8 bits** (`N = 0`). La destination reserve 4 slots
  d'argument par entree (`FUN_14080b034` complete de 0 les slots `M..3`) : `M` est lu tel
  qu'ecrit, la borne n'est pas dans le lecteur.
- **Meme forme que `ti=11 i2/i9`** (`consumeObjectiveFormattedText`, `components_batch3.go`) :
  le serialiseur `+0x28` `0x142edaef8` ecrit 8 bits de `+0x580` puis, par entree,
  `FUN_1407edaf4` (32 bits, etiquette `0x14371c588`) et **`FUN_142c70d5c`** — la fonction que
  le depot cite comme serialiseur de `ti=11 i2`. Le lecteur d'argument `FUN_1407f0ebc`
  concorde tag par tag avec le `switch` de `consumeObjectiveFormattedText`.
- Statut : **releve**.

## 14. `i10 managed-navpoint-timers-component`

- Descripteur `0x143d084d0` ; deserialiseur `FUN_1410d9040` : boucle `for (p = dest + 0x6e8 ; p != dest + 0x6f0 ; p += 4)` -> **deux** appels de `FUN_1410d9088`.
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| minuteur 1 -> `+0x6e8`, valeur lue - 1 | 7 | toujours | aucune |
| minuteur 2 -> `+0x6ec`, valeur lue - 1 | 7 | toujours | aucune |

- Largeur totale : **14**. Identique a `ti=11 i0` (`consumeObjectiveTimers`, `2 x R(7)`, `valeur - 1`).
- Serialiseur `+0x28` `0x142edb0f4` : deux `FUN_142ed15a0(flux, _, valeur)` = 7 bits de `valeur + 1`.
- Statut : **releve**.

## 15. `i11 managed-navpoint-manual-timer-initial-duration-component`

- Descripteur `0x143d08480` ; deserialiseur `FUN_142ed5194` : `FUN_1406d84b4(flux, flux, DAT_143d13418, DAT_143e418a8, 0x11, 0, 0)` -> `+0x6f8`.
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| duree initiale, quantifiee dans [`DAT_143d13418 = 0xbccccccd = -0,025f`, `DAT_143e418a8 = 0x45cccc9a = 6553,5752f`] | 17 | toujours | aucune |

- Dequantification (`f6 = 0`, `f7 = 0`, `pas_total = 131072`, pas = 6553,6002 / 131072 = 0,0500000015) :
  `v = -0,025 + (q + 0,5) * 0,05 = q * 0,05` — **des pas de 50 ms**, `q = 0 -> 0`,
  `q = 131071 -> 6553,55`.
- Largeur totale : **17**.
- Serialiseur `+0x28` `0x142edb03c` : memes bornes chargees (`MOVSS XMM0,[0x143e418a8]`, `MOVSS XMM3,[0x143d13418]`).
- Statut : **releve**.

## 16. `i12 managed-navpoint-manual-timer-current-duration-component`

- Descripteur `0x143d08430` ; deserialiseur `FUN_142ed512c` : meme `FUN_1406d84b4(flux, flux, DAT_143d13418, DAT_143e418a8, 0x11, 0, 0)`, puis zone morte : si `|v| <= DAT_143cd837c` (`0x38d1b717 = 1,0e-4f` ; valeur absolue par `& DAT_143cd8380 = 0x7fffffff`) alors `0`, sinon `v` -> `+0x6fc`.
- `param_4 = 1` (non utilise).

| Champ | Largeur (bits) | Condition | Dependance de config |
|---|---|---|---|
| duree courante, quantifiee comme `i11` (pas de 50 ms) | 17 | toujours | aucune |

- La zone morte est un traitement de la valeur, pas une lecture. Largeur totale : **17**.
- Serialiseur `+0x28` `0x142edaff4` : memes bornes chargees.
- Statut : **releve**.

## 17. Ce que le port demandera

1. **`paramByComponent`** (`component_param4.go`) : poser
   `managed-navpoint-visibility-distance-filters-component -> 3` et les quatre autres
   `*-filter(s)-component` (`i3`, `i4`, `i5`, `i6`) `-> 2`. Avec le defaut 1 le lecteur prend
   la mise en page legacy (`R(32)` de drapeau, `R(4)` par filtre, `R(2)` d'ordre) et se
   desynchronise des le premier filtre. Les valeurs viennent du slot `+0x10` du descripteur
   (§2) ; si un film d'une ancienne version du composant apparait un jour, la cle sera la
   version de format du film (`build_profile.go`), jamais une re-mesure statistique.
2. **Un seul lecteur de bloc de filtres** (`FUN_140dbe400` + la table des 16 tags, §3),
   partage par `i2`..`i6` et par `ti=11 i4 interaction-filter` (dont le « reste a faire » —
   enumerer les vtables candidates de l'appel virtuel — est ferme par la table §3 : cinq tags
   1..5 chez `FUN_141e99630` cote serialiseur, quinze cote lecteur ; a verifier sur pieces
   au moment du port de `ti=11 i4`).
3. **Une seule entree de profil** : `w0`, largeur de reference d'entite du domaine 0
   (`FUN_1406d3140`, tags 6/12/13/14). Reutiliser la primitive existante du depot
   (« IDLowBits »), ne pas la recoder.
4. **Ordre le moins cher** : `i1`, `i7` (`R(8)`) ; `i0` (deja porte), `i8` (`R(32)`) ; `i10`
   (`2 x R(7)`, reutiliser `consumeObjectiveTimers`) ; `i11`, `i12` (`R(17)`) ; `i9`
   (reutiliser `consumeObjectiveFormattedText` sous une boucle `R(8)` x [`R(32)` + corps]) ;
   puis `i5`/`i6` (bloc seul), `i3`/`i4`, `i2`. Le ratchet 0.A.3 doit voir le bloquant du
   golden avancer d'un index a chaque port.
5. **Dequantifier a la lecture, ne jamais publier les quanta** : distances de `i2`
   (`[-1, 1000]`, `f7 = 1`), durees de `i11`/`i12` (pas de 50 ms), argument de type 2 de
   `i9` (`[-6000, 6000]`, `f6 = 1`) — formules §1 et §6/§13/§15.
6. **Piste pour les huit `visual-state-groups` (`i20`..`i27`, non resolus dans la note
   d'archetype)** : leur table de huit pointeurs de noms est a `0x143d07f00..0x143d07f38`
   et le descripteur de `i4` (forme courte) commence immediatement apres (§8). Le descripteur
   commun des huit instances est donc a chercher parmi les references a `0x143d07f00`, pas
   par un accesseur de nom.
7. **Terminologie a corriger dans la note de methode** au moment du port : `+0x40` =
   deserialiseur (lecteur), `+0x28` = serialiseur (ecrivain) ; le plan dit « ecrivain » pour
   `+0x40` et c'est bien la fonction a porter. Les constantes `+0x30 = 0x1411c8f80`
   (ne revient pas) et `+0x20/+0x48 = 0x1404ab600` (`XOR AL,AL ; RET`) sont celles relevees
   ici (la note de methode ecrit `0x141c8f880` et `0x14049b600`).

## 18. Appels Ghidra en echec (8)

`get_function_by_address(0x1454514f4)` (entree `.pdata`, attendu) ; `decompile_function` sur
`0x141e9d660`, `0x141e9d0a0`, `0x141e9d0b0`, `0x141e9ca50`, `0x142edb148`, `0x142c74154`
(« No function found » : petits thunks non definis comme fonctions, tous releves par
`disassemble_bytes`) ; `disassemble_function(0x140dbde44)` a rendu 178 Mo (corps chaud/froid
aberrant), remplace par `disassemble_bytes` sur les deux plages. Aucune adresse de cette note
ne vient d'un appel en echec.
