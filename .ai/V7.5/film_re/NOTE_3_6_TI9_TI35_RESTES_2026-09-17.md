# ti=9 i9 et ti=35 i63 — les restes nommes par la synthese, releves (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « porter les composants
> manquants archetype par archetype »), groupe **ti9ti35** : les deux restes que
> `NOTE_3_6_SYNTHESE_2026-09-17.md` §7 nomme pour ces archetypes — (1) la boucle a etiquette
> de `FUN_141fcf160` (`ti=9 i9`, `NOTE_3_6_TI9_2026-09-16.md` : `partiel`) ; (2) les corps
> d'etiquette 17, 18, 19, 20, 21, 23 et `>= 24` d'`i63` (`ti=35`,
> `NOTE_3_6_TI35_GRAMMAIRES_2026-09-17.md` §6.1 : `partiel` / `non elucide`). Sans production :
> aucun film decode, aucun test, aucun fichier Go touche, rien ecrit dans Ghidra. Source :
> `HaloInfinite.exe` par Ghidra en LECTURE SEULE (HTTP direct `127.0.0.1:8089` : decompile /
> disassemble / read_memory / get_xrefs_to / search_strings). Methode :
> `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` (nom a `descripteur+0x18`, ecrivain a
> `descripteur+0x40` ; `+0x2c` du flux = compteur de bits, chaque `ADD [..+0x2c], N` = un
> champ de N bits ; un champ apparait deux fois dans le desassemblage — chemin rapide et
> chemin lent — compter les champs, pas les sites).
>
> Instrument / Ghidra lecture seule. Archetypes `ti=9` (joueur) et `ti=35` (bipede). Groupe
> ti9ti35.
>
> Toutes les adresses, largeurs et constantes de cette note sont COLLEES depuis Ghidra. Les
> lignes marquees « depot » citent le code Go existant pour dire ce que le port changera ;
> elles ne sont pas des mesures. Conventions : `R(n)` = lecture de n bits ; `gate8` =
> `FUN_1407f08bc` = `R(1)` puis, si 1, `R(8)` (sinon `0xffff`) ; `FUN_1406cf008` = `R(1)` ;
> `FUN_14080dec4(flux, nom, out)` = `R(32)` ; `FUN_1407f2058` = `R(1)` porte INVERSEE puis
> `R(5)` si 0 (`0xffffffff` si 1) ; `FUN_1406d84b4(flux, ?, min = XMM2, max = XMM3,
> n = [RSP+0x20], f6 = [RSP+0x28], f7 = [RSP+0x30])` = `R(n)` dequantifie (`f6` : diviseur
> `2^n - 1` au lieu de `2^n` ; `f7` : bornes exactes aux codes 0 et `2^n - 1`) ;
> `FUN_1406d676c(flux, ?, out, n)` = `R(n)` brut ; `FUN_142ef4db8` = 0 bit (destructeur
> virtuel de l'entree). Ces primitives sont celles de la table §2 de la note ti35 du 17.

---

## 0. Resume — les deux restes tombent, et les deux grammaires sont sans configuration

1. **`ti=9 i9`** : la « boucle a etiquette » est **entierement decidable hors ligne**. La
   grammaire complete est : `R(1)` present ; si 1 : `R(2)` ; `R(1)` texte ; si 1 : `R(32)`
   nom (« text ») ; `n = R(3)` ; puis `n` fois `k = R(3)` et un corps par `k` a largeurs
   LITTERALES : `k = 0` -> 0 bit ; `k = 1` -> `R(1)[0 -> R(5)]` ; `k = 2` -> `R(1)` puis
   `R(24)` (si 1) ou `R(32)` (si 0) ; `k = 3` -> `R(32)` ; `k = 4..7` -> `R(32)`. **Aucune
   dependance de configuration.** Le `ported = false` du depot sur `n > 0` n'a plus de raison.
2. **`ti=35 i63`** : les neuf corps restants (17, 18, 19, 20, 21, 22, 23, 24, 25..31) sont
   releves, tous a largeurs LITTERALES : 17 = `2 x R(15) + R(1) + gate8` ; 18 = `R(1) + R(2)
   + R(32) + R(1) + R(16) + R(8) + gate8` ; 19 = `R(32) + R(8) + gate8` ; 20 = `R(15)` ;
   21 = `gate8 + R(32) + R(4)` ; 22 = `gate8` ; 23 = `gate8 + R(32) + R(1)` ; 24 = `gate8 +
   R(32) + R(1) + R(1)` ; 25..31 = `R(32) + R(16) + R(32) + R(1) + R(8) + gate8`. La chaine
   de dispatch couvre donc les **32 valeurs** de `t = R(5)` ; `i63` n'a plus d'etiquette
   « non ouverte ».
3. Le descripteur d'`i9` (que la note du 16 n'avait pas releve) est **`0x143d08930`** :
   signature complete des 10 slots, `+0x40` = `0x141fcf160` — l'ecrivain de la table est
   confirme par la chaine des quatre pas.

Bilan : **0 non elucide de largeur** sur les deux archetypes. Restent trois residus de
SEMANTIQUE (valeur d'un champ, pas sa largeur), nommes au §4.

---

## 1. `ti=9 i9 managed-player-custom-input-prompt-widget`

### 1.1 Le descripteur (chaine des quatre pas)

> **Corrige apres verification independante (2026-09-17, seconde passe).** Trois adresses de
> slots etaient recopiees de travers dans la ligne « descripteur » : `14049b600` -> **`1404ab600`**
> (slots `+0x20` et `+0x48`) et `141c8f880` -> **`1411c8f80`** (slot `+0x30`). Ghidra rouvert en
> lecture seule sur les trois, le verificateur a raison :
> `read_memory address=0x143d08930 length=96` rend
> `b01a194101000000 d0ce764001000000 a0b4174101000000 400cfd4101000000 00b64a4001000000`
> `7000fd4101000000 808f1c4101000000 9cce764001000000 60f1fc4101000000 00b64a4001000000` ;
> `get_function_by_address 0x1404ab600` = entree `Reserve`, `0x1411c8f80` = entree `FUN_1411c8f80`,
> tandis que `0x14049b600` tombe au milieu de `FUN_14049b57c` et `0x141c8f880` au milieu de
> `FUN_141c8f3b0` (aucune entree de fonction). **Les deux slots qui portent la demonstration sont
> inchanges et exacts** : `+0x18` = `0x141fd0c40` (accesseur de nom), `+0x40` = `0x141fcf160`
> (ecrivain). Aucun champ du flux, aucune largeur, aucune condition de cette note n'est touche.

| Pas | Adresse | Preuve |
|---|---|---|
| chaine | `0x143c94db8` | `search_strings` : une seule occurrence |
| accesseur de nom | `0x141fd0c40` | seule reference a la chaine ; octets `48 8d 05 71 41 cc 01 c3` = `LEA RAX,[rip+0x1cc4171] ; RET` -> `0x141fd0c47 + 0x1cc4171 = 0x143c94db8` |
| slot du nom | `0x143d08948` | seule reference a l'accesseur |
| descripteur | **`0x143d08930`** | `0x143d08948 - 0x18` ; 10 slots lus (`read_memory 0x143d08930 len=96`, hex colle en tete de section) : `141191ab0`, `14076ced0`, `14117b4a0`, **`141fd0c40`**, `1404ab600` (`Reserve`), `141fd0070` (compagnon), `1411c8f80`, `14076ce9c`, **`141fcf160`**, `1404ab600` — les six pointeurs constants de la famille « managed-* » sont au bon offset |
| ecrivain | **`FUN_141fcf160`** | `descripteur + 0x40` |

### 1.2 La grammaire — `FUN_141fcf160(desc, flux, ctx)` puis `FUN_14080b034(etat+0x58, flux)` puis `n x FUN_1407f0ebc`

Etat de l'entite : `etat = [ctx+0x10]` (`141fcf16a MOV RDI,[R8+0x10]`). Sorties : `etat+0x84`
(present), `etat+0x80` (mode), `etat+0x58` (le « sac texte » : nom en `+0x58`, quatre
emplacements de 8 octets en `+0x5c`, `+0x64`, `+0x6c`, `+0x74` — dword valeur puis octet
type en `+4`).

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | `present` -> `etat+0x84` | `R(1)` | toujours | aucune | `141fcf17d INC dword ptr [RBX+0x2c]` (rapide) / `141fcf1b1 CALL FUN_1406d6c7c(flux, 1)` (lent) ; si 0 : `RET` avec `AL = 1` (`141fcf19c`, `141fcf22c`) — **0 bit de plus** |
| 2 | `mode` -> `etat+0x80` | `R(2)` | `present == 1` | aucune | `141fcf1dc ADD dword ptr [RBX+0x2c],0x2` / `141fcf1ff CALL FUN_1406d6c7c(flux, 2)` |
| 3 | appel `FUN_14080b034(etat+0x58, flux)` | — | `present == 1` | — | `141fcf20f LEA RCX,[RDI+0x58] ; 141fcf213 CALL 0x14080b034` |
| 3.1 | `texte` | `R(1)` (`FUN_1406cf008`) | toujours (dans 3) | aucune | `14080b056 CALL 0x1406cf008` ; si 0 : `[dst] = 0xffffffff` et les 4 octets de type a 0 (`14080b05f..14080b077`), `RET` — **0 bit de plus** |
| 3.2 | `nom` -> `dst+0` | `R(32)` (`FUN_14080dec4(flux, "text", dst)`) | `texte == 1` | aucune | `14080b097 LEA RDX,[0x1436f4b68]` (octets `74 65 78 74 00` = « text ») ; `14080b0a1 CALL 0x14080dec4` |
| 3.3 | `n` | `R(3)` | `texte == 1` | aucune | `14080b0bd ADD dword ptr [RBX+0x2c],0x3 ; 14080b0cd SHR R9,0x3d` (rapide) / `14080b162 ADD ..,0x3` (lent) ; valeur 0..7 |
| 3.4 | `n` fois : `FUN_1407f0ebc(flux, ?, &{flux, dst+4+8k})` | voir 1.3 | `n > 0` | aucune | boucle `14080b0ef..14080b109` (`R14` = compteur, `RSI` = emplacement, `ADD RSI,0x8`) |
| 3.5 | remise a 0 des types des emplacements `n..3` | 0 | `n < 4` | — | `14080b10b..14080b134` |

Observation (pas une regle de grammaire) : `n` est lu sur 3 bits (0..7) mais la structure n'a
que **4** emplacements (`dst+4..dst+0x24`, cf. `LEA RCX,[RAX+0x20]` en `14080b066`) ; un
ecrivain conforme n'emet pas `n > 4`. Le lecteur doit lire `n` corps quel que soit `n` — c'est
ce que fait le jeu.

### 1.3 Le corps d'etiquette — `FUN_1407f0ebc(flux, ?, ctx = {flux, emplacement})`

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| a | `k` (sous-type) | `R(3)` | toujours | aucune | `1407f0edd MOV EDI,0x3 ; 1407f0eeb ADD dword ptr [R10+0x2c],EDI ; 1407f0f02 SHR R9,0x3d` (rapide) / `1407f0f8e` (lent) |
| b | `k == 0` : type := 0 | **0** | `k == 0` | — | `1407f0f09 JZ 0x142326018` (bloc froid) ; decompile : `*(param_3[1] + 4) = 0` |
| c | `k == 1` : reference de participant | `R(1)` ; si **0** : `R(5)` (`FUN_1407f2058`) | `k == 1` | aucune | `1407f0f1c CALL 0x1407f2058` ; puis `1407f0f25 CALL 0x14049746c(idx)` et `1407f0f36 CALL 0x140e958c4(&local, idx)` = **0 bit** (aucun pointeur de flux passe ; decompiles : table TLS `[lVar2+0x78] + idx * [lVar2+0x40]`, borne `[lVar2+0x6c]`) ; type := 1 (`1407f0f4f`) ; valeur `0xffffffff` si la table rend nul (`1407f0f5e`) |
| d | `k == 2` : `FUN_142c70cd0(ctx)` | `R(1)` (`142c70cdc`) ; si **1** : `R(24)` = `FUN_1406d84b4(n = 0x18 [RSP+0x20] @142c70d02, f6 = 1 @142c70cfd, f7 = 0 @142c70cf8, min = XMM2 = DAT_143d13518 = 0xc5bb8000 = -6000,0f, max = XMM3 = DAT_143d1334c = 0x45bb8000 = +6000,0f)` (`142c70d0a`) ; si **0** : `R(32)` brut = `FUN_1406d676c(flux, ?, out, R9D = 0x20 @142c70d27)` (`142c70d33`) | `k == 2` | aucune | type := 2 (`142c70d4c`) |
| e | `k == 3` : `string_id` | `R(32)` (`FUN_14080dec4(flux, "string_id", &local)`) | `k == 3` | aucune | decompile de `FUN_1407f0ebc` (bloc froid `0x142325fb6..`) ; type := 3 |
| f | `k >= 4` (4..7) : `FUN_142c70be0(ctx)` | `R(32)` | `k >= 4` | aucune | `142c70c05 ADD dword ptr [R10+0x2c],0x20` (rapide) / `142c70c78` (lent) ; type := 4 |

Largeur d'un corps, `k` compris : `k = 0` -> 3 ; `k = 1` -> 4 ou 9 ; `k = 2` -> 28 ou 36 ;
`k = 3` -> 35 ; `k = 4..7` -> 35.

### 1.4 Largeur totale et statut

- `present = 0` : **1 bit**.
- `present = 1`, `texte = 0` : **4 bits**.
- `present = 1`, `texte = 1` : `4 + 32 + 3 + somme des corps` — variable, entierement
  determinee par les bits lus.

**Statut : releve.** Aucune porte sur un etat runtime, aucun appel virtuel, aucune table de
largeur : les deux seules fonctions qui touchent de la RAM externe (`FUN_14049746c`,
`FUN_140e958c4`, corps `k = 1`) ne recoivent pas le flux et ne changent que la VALEUR
rangee dans l'emplacement, jamais le nombre de bits consommes.

Depot (ce que le port changera) : `dispatch_player.go:298` lit deja `R(1)`, `R(2)`, `R(1)`,
`R(32)`, `R(3)` — dans le bon ordre — puis rend `ported = false` des que `n > 0` ; c'est la
seule chose a remplacer par la boucle du §1.3. Le meme sac texte `FUN_14080b034` est deja
decrit, cote tests seulement, par `playergameevent_0xe9_helpers_test.go:91` (evenement
`PlayerGameEventSmall`, type 82) et `r7_charges_lot2_research_test.go:121` ; le premier
marque `k = 2` « quantifie a largeur runtime » (`p.exact = false`) — **sur pieces c'est
faux** : `0x18` et `0x20` sont des litteraux (`142c70d02`, `142c70d27`) — et il ignore
`k >= 4` (`R(32)`). Le port ne doit pas recopier ces deux ecarts.

---

## 2. `ti=35 i63 biped-action-component` — les corps d'etiquette 17 a 31

- Index `63`, descripteur `0x143d0cdf0`, ecrivain `FUN_142f027f4` -> `FUN_142f26a20(etat =
  ctx+0xaa8, flux)` (note ti35 du 17, §1 et §6 — squelette, masque `3 x R(32)`, `n1 = R(4)`,
  boucle 2 = `popcount` du masque : rien n'est repris ici).
- Chaque element de la boucle 1 : `R(7)` puis `FUN_142ef4c98` -> `FUN_142ef1734` :
  **`t = R(5)`** (`*(param_1 + 0x2c) += 5`, relu ce jour) puis `FUN_141fd4814(t, {flux,
  &entree})`.
- Chaine de dispatch, relue : `FUN_141fd4814` (0..5) -> `FUN_142ef01c4` (6..11) ->
  `FUN_142ef0538` (`param_1 < 0x12` : 12..17 ; sinon `FUN_142ef0388`) -> `FUN_142ef0388`
  (`142ef0395 CMP RCX,0x17 ; JA 142ef0483` : 18..23 ; sinon `142ef0483 CALL 0x142ef0494`) ->
  `FUN_142ef0494` (`142ef04a6 CMP RCX,0x18` : **24** ; **tout autre** -> corps « 25 »). Les
  corps 0..16 sont dans la note ti35 du 17, §6.1 ; ceux-ci les completent.
- Chaque corps commence par `FUN_142ef4db8(entree)` (0 bit), une remise a zero de l'entree et
  l'ecriture de l'etiquette en `entree+0x28` (l'octet `0x11`..`0x19` que le decompile montre).

### 2.1 `t = 17` — `FUN_142eefd08` (via `FUN_142ef0538`, `iVar2 == 0x11`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | float -> `entree+0` | `R(15)` = `FUN_1406d84b4(n = 0xf, f6 = 0, f7 = 0, min = 0,0 (XORPS XMM2), max = DAT_143cd84e4 = 0x42700000 = 60,0f)` | toujours | aucune | `142eefd56 MOV dword ptr [RSP+0x20],0xf ; 142eefd51 [RSP+0x28] = 0 ; 142eefd49 [RSP+0x30] = 0 ; 142eefd5e CALL 0x1406d84b4` |
| 2 | float -> `entree+4` | `R(15)` = `FUN_1406d84b4(n = 0xf, f6 = 0, f7 = 0)` | toujours | aucune | `142eefd70 MOV dword ptr [RSP+0x20],0xf ; 142eefd6b/d63 = 0 ; 142eefd7c CALL 0x1406d84b4` — **XMM2/XMM3 ne sont pas recharges** entre les deux appels (residu semantique, §4) |
| 3 | octet -> `entree+0xa` | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `142eefd89 CALL 0x1406cf008 ; 142eefd92 MOV byte ptr [RSI+0xa],AL` |
| 4 | `gate8` -> `entree+8` | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `142eefd8e LEA RDX,[RSI+0x8] ; 142eefda7 JMP 0x1407f08bc` (appel de queue) |

Largeur : **32 ou 40 bits.** Statut : **releve** (les « 2 largeurs non relevees » de la
synthese valent `0xf` toutes deux).

### 2.2 `t = 18` — `FUN_14319c6f8(entree, flux)` (via `FUN_142ef0388`, `iVar4 == 0x12`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | float -> `entree+4` : `1,0f` (`DAT_143cd8374 = 0x3f800000`) si le bit vaut **0**, `0,0f` si 1 | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `14319c715 CALL 0x1406cf008 ; 14319c71e JZ ; 14319c725 MOVSS XMM0,[0x143cd8374] ; 14319c730 MOVSS [RSI+0x4],XMM0` |
| 2 | octet -> `entree+0` | `R(2)` (`FUN_142af27f8(flux, ?, R8 = entree)`) | toujours | aucune | `14319c72d MOV R8,RSI ; 14319c738 CALL 0x142af27f8` ; decompile de `FUN_142af27f8` : `+= 2`, `*param_3 = bVar4 >> 6` |
| 3 | dword -> `entree+0xc` | `R(32)` (`FUN_14080dec4(flux, ?, R8 = entree+0xc)`) | toujours | aucune | `14319c73d LEA R8,[RSI+0xc] ; 14319c744 CALL 0x14080dec4` |
| 4 | octet -> `entree+8` | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `14319c74c CALL 0x1406cf008 ; 14319c756 MOV byte ptr [RSI+0x8],AL` |
| 5 | ushort -> `entree+0x10` | `R(16)` (inline) | toujours | aucune | `14319c751 MOV EBP,0x40 ; 14319c766 LEA R11D,[RBP-0x30]` (= `0x10`) ; `14319c76f ADD dword ptr [RBX+0x2c],R11D` (rapide) / `14319c7e2` (lent) |
| 6 | octet -> `entree+9` | `R(8)` (inline) | toujours | aucune | `14319c828 ADD dword ptr [RBX+0x2c],0x8` (rapide) / `14319c88e` (lent) ; `14319c8c5 MOV byte ptr [RSI+0x9],R9B` |
| 7 | `gate8` -> `entree+0x12` | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `14319c8ba LEA RDX,[RSI+0x12] ; 14319c8c9 CALL 0x1407f08bc` |

Largeur : **61 ou 69 bits.** Statut : **releve**.

### 2.3 `t = 19` — `FUN_142eefe1c` (via `FUN_142ef0388`, `iVar4 == 0x13`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | dword -> `entree+0` | `R(32)` (`FUN_14080dec4(flux, "networked damagetype" @0x143e2b418, &local)`) | toujours | aucune | `142eefe46 LEA RDX,[0x143e2b418] ; 142eefe66 CALL 0x14080dec4 ; 142eefe87 MOV dword ptr [RSI],EAX` |
| 2 | float -> `entree+4` | `R(8)` = `FUN_1406d84b4(n = 8, f6 = 0, f7 = 0, min = 0,0 (XORPS XMM2), max = DAT_143cd8640 = 0x437f0000 = 255,0f)` | toujours | aucune | `142eefe89 MOV dword ptr [RSP+0x20],0x8 ; 142eefe82/e7d = 0 ; 142eefe91 CALL 0x1406d84b4` |
| 3 | `gate8` -> `entree+8` | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `142eefe96 LEA RDX,[RSI+0x8] ; 142eefeb1 JMP 0x1407f08bc` |

Largeur : **41 ou 49 bits.** Statut : **releve**.

### 2.4 `t = 20` — inline dans `FUN_142ef0388` (`iVar4 == 0x14`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | float -> `entree+0` | `R(15)` = `FUN_1406d84b4(n = 0xf, f6 = 0 (AL = 0), f7 = 1, min = 0,0 (XORPS XMM2), max = DAT_143cd842c = 0x40c90fdb = 6,2831855f = 2 pi)` | toujours | aucune | `142ef0435 MOV dword ptr [RSP+0x20],0xf ; 142ef0431 [RSP+0x28] = AL = 0 ; 142ef042c [RSP+0x30] = 1 ; 142ef043d CALL 0x1406d84b4 ; 142ef0442 MOVSS [RBX],XMM0` |

Largeur : **15 bits, constante.** Statut : **releve** (la « 1 largeur non relevee » vaut
`0xf` ; c'est un angle sur `[0, 2 pi]` avec bornes exactes aux codes 0 et `2^15 - 1`).

### 2.5 `t = 21` — `FUN_142eeff18` (via `FUN_142ef0388`, `iVar4 == 0x15`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | `gate8` -> `entree+0` (ushort) | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `142eeff38 MOV RDX,RSI ; 142eeff57 CALL 0x1407f08bc` |
| 2 | dword -> `entree+4` | `R(32)` (`FUN_14080d6f0(?, RDX = flux, R8 = entree+4)`) | toujours | aucune | `142eeff5c LEA R8,[RSI+0x4] ; 142eeff60 MOV RDX,RDI ; 142eeff63 CALL 0x14080d6f0` ; decompile de `FUN_14080d6f0` : `+= 0x20`, `*param_3 = uVar4` |
| 3 | dword -> `entree+8` | `R(4)` (`FUN_1424e1d48(flux)`) | toujours | aucune | `142eeff6b CALL 0x1424e1d48 ; 142eeff75 MOV dword ptr [RSI+0x8],EAX` ; decompile : `+= 4`, retour `uVar4 >> 0x3c` |

Largeur : **37 ou 45 bits.** Statut : **releve**.

### 2.6 `t = 22` — inline dans `FUN_142ef0388` (`iVar4 == 0x16`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | `gate8` -> `entree+0` (ushort) | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `142ef03e9 MOV RDX,RBX ; 142ef03f6 CALL 0x1407f08bc` |

Largeur : **1 ou 9 bits.** Statut : **releve** (deja dans la note ti35 du 17 ; confirme).

### 2.7 `t = 23` — `FUN_142ef004c` -> `FUN_1431b3760(entree, flux)` (via `FUN_142ef0388`, `iVar4 == 0x17`)

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | `gate8` -> `entree+0` (ushort) | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `142ef0089 JMP 0x1431b3760` ; `1431b3776 CALL 0x1407f08bc` |
| 2 | dword -> `entree+4` | `R(32)` (inline) | toujours | aucune | `1431b3790 ADD dword ptr [RBX+0x2c],0x20` (rapide) / `1431b3801` (lent) ; `1431b3838 MOV dword ptr [RDI+0x4],R9D` |
| 3 | bit 0 de `entree+8` | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `1431b383c CALL 0x1406cf008` ; decompile : `[+8] = ([+8] & 0xfe) OR bit` |

Largeur : **34 ou 42 bits.** Statut : **releve**.

### 2.8 `t = 24` — `FUN_142ef0494`, branche `param_1 == 0x18`

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | `gate8` -> `entree+0` (ushort) | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `142ef04e6 MOV RDX,RDI ; 142ef0501 CALL 0x1407f08bc` |
| 2 | dword -> `entree+4` | `R(32)` (`FUN_14080dec4(flux, ?, R8 = entree+4)`) | toujours | aucune | `142ef0506 LEA R8,[RDI+0x4] ; 142ef050d CALL 0x14080dec4` |
| 3 | octet -> `entree+8` | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `142ef0515 CALL 0x1406cf008 ; 142ef051d MOV byte ptr [RDI+0x8],AL` |
| 4 | octet -> `entree+9` | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `142ef0520 CALL 0x1406cf008 ; 142ef0525 MOV byte ptr [RDI+0x9],AL` |

Largeur : **35 ou 43 bits.** Statut : **releve**.

### 2.9 `t = 25..31` — `FUN_142ef0494`, branche `param_1 != 0x18` -> `FUN_1431c8c54(entree, flux)`

L'etiquette rangee en `entree+0x28` est **`0x19` quelle que soit la valeur de `t`**
(`142ef04ca MOV byte ptr [RBX+0x28],0x19`) : le jeu ne distingue pas 26..31 de 25 et leur
applique le meme corps. Un lecteur hors ligne fait pareil (residu semantique, §4).

| ordre | champ | largeur bits | condition | dependance de config | preuve |
|---|---|---|---|---|---|
| 1 | dword -> `entree+0` | `R(32)` (`FUN_14080dec4(flux, flux, entree)`) | toujours | aucune | `1431c8c74 CALL 0x14080dec4` |
| 2 | dword -> `entree+4` | `R(16)` (inline) | toujours | aucune | `1431c8c7c MOV EBP,0x40 ; 1431c8c8d LEA R11D,[RBP-0x30]` (= `0x10`) ; `1431c8c96 ADD dword ptr [RBX+0x2c],R11D` (rapide) / `1431c8d09` (lent) ; `1431c8d38 MOV dword ptr [RSI+0x4],R9D` |
| 3 | dword -> `entree+8` | `R(32)` (inline) | toujours | aucune | `1431c8d42 MOV R11D,0x20 ; 1431c8d54 ADD dword ptr [RBX+0x2c],R11D` (rapide) / `1431c8dc7` (lent) ; `1431c8df9 MOV dword ptr [RSI+0x8],R9D` |
| 4 | octet -> `entree+0xc` | `R(1)` (`FUN_1406cf008`) | toujours | aucune | `1431c8dfd CALL 0x1406cf008 ; 1431c8e02 MOV byte ptr [RSI+0xc],AL` |
| 5 | octet -> `entree+0xd` | `R(8)` (inline) | toujours | aucune | `1431c8e15 ADD dword ptr [RBX+0x2c],0x8` (rapide) / `1431c8e80` (lent) ; `1431c8eb0 MOV byte ptr [RSI+0xd],R9B` |
| 6 | `gate8` -> `entree+0xe` | `R(1)` ; si 1 : `R(8)` | toujours | aucune | `1431c8eb7 CALL 0x1407f08bc` (decompile : `FUN_1407f08bc(param_2, param_1 + 0xe)`) |

Largeur : **90 ou 98 bits.** Statut : **releve**.

### 2.10 Table recapitulative des 32 etiquettes d'`i63`

| `t` | corps | largeur bits (min / max) | source |
|---|---|---|---|
| 0..16 | note ti35 du 17, §6.1 | — | inchange |
| 17 | `FUN_142eefd08` | 32 / 40 | §2.1 |
| 18 | `FUN_14319c6f8` | 61 / 69 | §2.2 |
| 19 | `FUN_142eefe1c` | 41 / 49 | §2.3 |
| 20 | inline `FUN_142ef0388` | 15 / 15 | §2.4 |
| 21 | `FUN_142eeff18` | 37 / 45 | §2.5 |
| 22 | inline `FUN_142ef0388` | 1 / 9 | §2.6 |
| 23 | `FUN_142ef004c` -> `FUN_1431b3760` | 34 / 42 | §2.7 |
| 24 | `FUN_142ef0494` (branche `0x18`) | 35 / 43 | §2.8 |
| 25..31 | `FUN_142ef0494` -> `FUN_1431c8c54` | 90 / 98 | §2.9 |

Statut d'`i63` : **releve** — squelette, boucle 2 et les 32 corps d'etiquette. La seule
dependance de configuration de tout le composant reste celle du corps `t = 0`
(`FUN_1408f0ac4(dom 0)` : cardinal du domaine 0, deja porte par `readVarWidthInt(br, 0)`) ;
aucun des corps 17..31 n'en a.

Depot (ce que le port changera) : `components_biped_spartan.go:228` `consumeBipedActionTag`
porte 0..5 et un `default` « tag >= 6 -> 0 bit, verite EXE 2026-06-13 ». La note ti35 du 17
a deja montre que ce `default` est une continuation de dispatch, pas une voie d'erreur ; avec
cette note il n'y a plus AUCUNE valeur de `t` pour laquelle « 0 bit » soit vrai (`t = 22`
lit au moins 1 bit).

---

## 3. Ce que le port demandera

1. **`ti=9 i9`** — remplacer, dans `dispatch_player.go:298`, le `return variant, nil, false`
   sur `n > 0` par la boucle du §1.3 : `n` fois `k = R(3)` puis `switch k` { `0` : rien ;
   `1` : `R(1)`, si 0 `R(5)` ; `2` : `R(1)`, si 1 `R(24)` sinon `R(32)` ; `3` : `R(32)` ;
   `default` (4..7) : `R(32)` }. Le meme sac texte sert l'evenement `PlayerGameEventSmall`
   (type 82, `FUN_14080add8` -> `FUN_14080b034`) : un seul helper de production, et retirer la
   version des deux fichiers de test (`playergameevent_0xe9_helpers_test.go:91`,
   `r7_charges_lot2_research_test.go:121`) ou la faire pointer sur lui — regle des deux
   copies. La ligne `ecs_table.tsv` passe de `partiel` a `porte`, `grammar` = `R(1)[1 ->
   R(2) + R(1)[1 -> R(32) + n=R(3) + n x (R(3) + corps)]]`, `bits_typ` = `1` (cas commun
   `present = 0`), note « boucle a etiquette decidable, 5 corps a largeurs litterales ».
2. **`ti=35 i63`** — dans `consumeBipedActionTag`, apres les `case` 6..16 que la note ti35 du
   17 demande deja : ajouter les `case` 17..24 et un `default` pour 25..31 avec les
   grammaires des §2.1..§2.9 (toutes en `br.ReadBits(n)` + `gate8(br)` existant) ; supprimer
   toute branche « 0 bit » ; `ported = true` partout (il n'y a plus d'etiquette non ouverte).
   Le commentaire « verite EXE 2026-06-13 » et la mention « corps 6..11 inventes » sont a
   reecrire dans le meme commit (doc inversee, anti-pattern 9). La ligne `ecs_table.tsv` :
   `partiel` -> `porte` une fois le popcount de la boucle 2 branche (note ti35 du 17, §7.4),
   `bits_typ` reste `196` au cas commun (`n1 = 0`, `n2 = 0`).
3. **Entrees de profil / configuration a creer** : **0** pour les deux composants. Toutes les
   largeurs de cette note sont des litteraux d'instruction (`0x3`, `0x18`, `0x20`, `0xf`,
   `0x8`, `0x10`, `0x4`) ou des primitives a largeur fixe (`gate8`, `FUN_1407f2058`).
4. **Constantes de valeur** (ne changent aucune largeur) : `-6000,0 / +6000,0`
   (`DAT_143d13518 / DAT_143d1334c`, `i9 k = 2`) ; `60,0` (`DAT_143cd84e4`, `t = 17`) ;
   `255,0` (`DAT_143cd8640`, `t = 19`) ; `2 pi` (`DAT_143cd842c`, `t = 20`) ; `1,0`
   (`DAT_143cd8374`, `t = 18`) ; chaines de nom « text » (`0x1436f4b68`), « string_id »,
   « networked damagetype » (`0x143e2b418`) — ce sont des libelles de debogage passes a
   `FUN_14080dec4`, jamais lus dans le flux.
5. **Gates** : `ti=9` — fermeture 0 -> 1 717 / 1 717 sur les 7 bobines apres `i4` + `i9` (si
   elle ne monte pas, le golden nommera un composant qui n'est ni `i4` ni `i9` : il n'en
   reste aucun de non porte dans l'archetype) ; `ti=35` — 6 / 1 364 -> 1 364 / 1 364 apres
   `i60`, `i57`, `i59`, `i63` ; si `i63` reste bloquant avec `n1 > 0`, la sequence de `t`
   observee doit etre confrontee a la table §2.10 avant toute autre hypothese.

---

## 4. Ce qui reste non elucide — trois residus de SEMANTIQUE, aucun de largeur

| Ou | Quoi | Pourquoi ce n'est pas une largeur | Ce qui le leve |
|---|---|---|---|
| `ti=9 i9`, corps `k = 1` | la VALEUR rangee dans l'emplacement passe par deux tables runtime (`FUN_14049746c` : table TLS indexee par `R(5)`, `FUN_140e958c4`) | aucun pointeur de flux ne leur est passe ; les bits consommes sont `1 [+5]` quoi qu'elles rendent | rien a lever pour le port ; le decodeur publie l'index brut `R(5)` (ou `absent`), pas la valeur transformee |
| `ti=35 i63`, `t = 17`, champ 2 | bornes de dequantification du second `R(15)` : `XMM2`/`XMM3` ne sont pas recharges apres le premier `CALL 0x1406d84b4` (`142eefd63..142eefd7c`) | `n = 0xf`, `f6 = 0`, `f7 = 0` sont re-ecrits sur la pile ; la largeur est certaine ; seule la valeur flottante depend de ce que `FUN_1406d84b4` laisse dans `XMM2`/`XMM3` | lecture du prologue / epilogue de `FUN_1406d84b4` pour savoir s'il preserve `XMM2`/`XMM3` (0 bit en jeu) — utile seulement si le port veut publier la valeur |
| `ti=35 i63`, `t = 26..31` | semantique de ces valeurs : le jeu les lit avec le corps de `t = 25` et range l'etiquette `0x19` | la largeur est celle du §2.9 (90 / 98) pour les sept valeurs | aucune ; le lecteur reproduit le comportement du jeu (meme corps) et peut signaler `t > 25` comme etiquette hors enumeration |

Les deux residus deja notes par la note ti35 du 17 pour les corps 8 (« arguments de l'appel
interne non relus ligne a ligne ») et 14 (« min non lu ») sont hors du perimetre de cette
note et n'y sont pas repris.

---

## 5. Journal des appels Ghidra

Cinquante-quatre appels HTTP en lecture, **0 en echec**, aucune ecriture dans Ghidra :
2 lectures du catalogue `/mcp/schema` ; `list_open_programs` et `get_current_program_info`
(un seul programme, `HaloInfinite.exe`, base `0x140000000`) ; **21 `decompile_function`**
(`141fcf160`, `14080b034`, `1407f0ebc`, `142c70cd0`, `142c70be0`, `14049746c`, `140e958c4`,
`142af27f8`, `142ef1734`, `142ef0538`, `142ef0388`, `142eefd08`, `14319c6f8`, `142eefe1c`,
`142eeff18`, `14080d6f0`, `1424e1d48`, `142ef004c`, `1431b3760`, `142ef0494`, `1431c8c54`) ;
**14 `disassemble_function`** (`141fcf160`, `14080b034`, `1407f0ebc`, `142c70cd0`,
`142c70be0`, `142eefd08`, `14319c6f8`, `142eefe1c`, `142ef0388`, `142eeff18`, `142ef004c`,
`1431b3760`, `142ef0494`, `1431c8c54`) ; **12 `read_memory`** (six bornes flottantes, deux
bornes d'`i9 k = 2`, les chaines « text » et « networked damagetype », les octets de
l'accesseur `0x141fd0c40`, les 80 octets du descripteur `0x143d08930`) ; 1 `search_strings` ;
2 `get_xrefs_to` (chaine -> accesseur -> slot).

Anomalie d'outillage, pas une erreur : `disassemble_function 0x1407f0ebc` a rendu **209,5 Mo**
(le serveur a suivi les sauts vers les blocs froids `0x142325fb6` / `0x142326018` et deroule
bien au-dela de la fonction) ; la plage utile `1407f0ebc..1407f10c6` a ete extraite par `grep`
sur les adresses et le fichier purge — meme comportement que celui consigne par la synthese
(§9) pour `0x140c1dd44`.
