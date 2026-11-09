# NOTE 5.26 — LES NAISSANCES NON DECLAREES : DEUX PISTES, UNE ADRESSE CHACUNE, FERMEES

**Pourquoi ce lot.** L objectif de l utilisateur est un rendu DENSE et COMPLET du rejeu 2D. Sur
film dense, le calque des etats du joueur perd encore **35,4 %** des ticks de bipede de joueur
sur `bfecd02b` (85 843 / 242 711, 34,1 % nets) et **53,7 %** sur `4f77afc1` (lot 5.25), parce
que les trames delta s abandonnent au premier en-tete qui designe une entite NEE ET MORTE entre
deux images-cles (801 eid sur `bfecd02b`, 871 sur `4f77afc1`, aucun declare par une image-cle).
Chaque trame fermee de plus rendrait des ticks de joueur au rejeu. Ce lot a instruit les DEUX
dernieres pistes nommees ; **aucune ne tient** : AUCUN tick n est rendu, et le chiffre du 5.25
reste le chiffre.

Lot 5.26, branche `feat/decfilm-76`, base `43a01721e` (serie 5 complete, schema 68,
`grammar-2026-09-22.12`). Mesure et lecture seules : deux instruments `research`, AUCUNE ligne
de production, AUCUNE revision montee.

---

## 1. CE QUE LE DEPOT LIT ET NE LIT PAS

La pompe du depot, `source.Paquets` (`internal/source/film.go`), ENUMERE tous les types sans
les repartir ; ce sont les lecteurs qui filtrent. En production, ils ne lisent que le type
**0** (`PacketTypeDelta`, la trame delta) et le type **2** (`PacketTypeKeyframe`) ; le type
**1** (`LireBlocDeDatums`) et le type **8** (`PacketTypeRoster`) ne sont lus que par des
instruments `research` ; le type **7** (CHUNK_END) n arrete que la marche. Les types **6, 9, 10,
0xb, 0xc** ne sont lus par RIEN.

Chez l ecrivain, le repartiteur `FUN_1428e22c0` : 0 -> `FUN_1428e2778`, 1 -> `FUN_142989418`,
6 -> `FUN_142988084`, 7 -> `FUN_142985698`, 8 -> `FUN_142987bd4`, 9 -> curseur `+= taille`,
10 -> `FUN_142988244`, 0xb -> `FUN_1429882c8`, 0xc -> `FUN_1429875e4` ; 2, 3, 4, 5 ->
`FilmBlockReadError`.

## 2. PISTE (a) — LES TROIS HANDLERS : FERMEE PAR L ECRIVAIN, CONFIRMEE PAR LA MESURE

| type | handler | ce qu il lit | ce qu il ecrit | qui le consomme |
|---|---|---|---|---|
| 6 | `FUN_142988084` | le bloc (taille = en-tete + 4), lecteur de bits, `FUN_1406d5cc0(.,3)` (0 bit), **R(32)** | l appel `FUN_1410de6c4(v, 0)` (un INDICE DE JOUEUR : borne `+0x83c`, table de pas `0x458`) ; le repartiteur range `v` en `session+0x114` (`MOV [RBX+0x114],EAX` en `0x1428e2388`) et pose `session+0x118 = 0` | `session` = `DAT_144c23178` ; `+0x114` = **`DAT_144c2328c`**, lu par le SEUL `FUN_142e2e104` (vue Theater « Film View ») qui le passe a `FUN_1428c53e8` (est-ce le joueur local ?) puis, si oui, a `FUN_1428e24bc` (re-serialisation du monde VIVANT par sa vtable `+0x40`/`+0x48`, jusqu a `0xa00` entites, aucune lecture du film). `+0x118` est remis a 1 par `FUN_1428e2c8c` et `FUN_1428e2d18` (redemarrage, saut dans le film) |
| 0xb | `FUN_1429882c8` | alloue un tampon de la taille du bloc (`FUN_1429907c8`), le lit (`FUN_142988338`) | **rien** : le tampon est libere (`FUN_1405a3720`) | personne |
| 0xc | `FUN_1429875e4` | **R(32)** = un compte ; par entree `R(32)` x 3 + `FUN_1407eeba4` (entrees de `0x1440` o) | la table de BOTS : `FUN_142c26748` (« botHandle.IsValid() », « absoluteBotIndex », table de participants TLS `+0x4ab78`, pas `0x610`), `FUN_142c2e510`, `FUN_142c2d090` | le gestionnaire de bots — ni la table de datums (`monde+0x120`, pas `0x18` / 200), ni la garde |

Aucun des trois n atteint `FUN_1406cd128` (branche vive), `FUN_1406caad8` (garde d eid) ni la
table de datums ; et les deux seules entrees de cette table restent celles du 5.20.3 (c)
(`FUN_1408f1314` <- `NEW` de trame, `FUN_1408f1618` <- image-cle).

**LA MESURE** (`TestHandlers526`, `mouvement_5_26_handlers_research_test.go`) :

| | `dad793c7` | `bfecd02b` |
|---|---:|---:|
| paquets de type 6 · octets | 5 · 20 | 27 · 108 |
| taille · R(32) de tete | 4 o · **0** partout | 4 o · **0** partout |
| rang apres l image-cle du chunk | +1 (5 sur 5) | +1 (27 sur 27) |
| paquets de type 0xb | **0** | **0** |
| paquets de type 0xc · octets | 6 · 24 | 28 · 112 |
| taille · compte R(32) | 4 o · **0** partout | 4 o · **0** partout |
| rang apres l image-cle | +3 (5), +5 (1) | +3 (27), +5 (1) |
| eid rejetes (premier rejet) | 0 | 801 |
| dont un paquet 6/0xb/0xc ENTRE leur image-cle et leur premier rejet | — | **801 (100 %)** |
| paquets 6/0xb/0xc suivis IMMEDIATEMENT d un premier rejet | 0 / 11 | **2 / 55** |
| taux de base : paquets delta portant un premier rejet | 0 / 5 370 | 801 / 31 232 (**2,56 %**) |

Le « 100 % » est STRUCTUREL : les paquets 6 et 0xc sont l en-tete de chaque chunk (rang +1 et +3
apres l image-cle, avant toute trame). Ils ne precedent pas une naissance plus que le hasard
(2 sur 55 = 3,6 % contre 2,56 %), et leur contenu est CONSTANT : l indice de joueur 0, un compte
de bots nul. Population des autres types reproduite au paquet pres contre le 5.19.3.

## 3. PISTE (b) — LE RECORD 36 : FERMEE PAR L ECRIVAIN, CONFIRMEE PAR LA MESURE

Le record `action_weapon_fire` (tete de paquet `0xD2` ; 105 dans la numerotation de
`fire_events.go`, 36 dans celle de la trame) est lu en deux temps.

**Le repartiteur `FUN_14080a9d4`** : R(7) type, puis TROIS references gardees — R(1) garde, et
si posee `FUN_1406d3140(lecteur, domaine, &ref)`. Le domaine vient de la vtable du descripteur
(`0x143d0aca0`, entree `+0x58` = `0x14080a048` : `TEST EDX,EDX ; JNZ ; LEA EAX,[RDX+1]` puis
`SUB EDX,1 ; JZ ; MOV EAX,7 | MOV EAX,8`) : **ref0 = domaine 1, ref1 = domaine 8, ref2 =
domaine 7**. `FUN_1406d3140` prend base et compte en `DAT_1451f98d0 + domaine*8` : le domaine 7
(`DAT_1451f9908`) est EXACTEMENT celui de la boucle de records de `FUN_1406cd128` — ref2 est la
seule reference du record dans l espace d identifiants des trames delta. Le corps est ensuite
appele avec son cinquieme argument a **1** (`MOV byte ptr [RSP+0x20],0x1`, `0x14080aad1`).

**Le corps `FUN_14080c1f8`**, dans l ordre :

| champ | lecteur | ecrit en | ce que c est | `fire_events.go` / `fire_aim_modal.go` |
|---|---|---|---|---|
| a | R(1) | `+0x00` | variante courte | lu (bit 7 / `estCourt`) |
| b | R(1) | `+0x1c` | bloc supplementaire | lu (`estBloc`) |
| c | `FUN_141fcf670` | `+0x01` | attaquant (8 bits selon `fire_aim_modal.go`, lecteur non relu ici) | lu aux bits 35..40 (offsets fixes) ; saute (8) en modal |
| d | `FUN_1407f2034` | `+0x0c` | R(1) + [0 : R(5)] | saute |
| e | `FUN_1406d00ec` | `+0x08` | R(1) + [0 : R(2)] | saute |
| f | `FUN_14080d69c` | `+0x10` | arme, famille : R(1) + [1 : R(32)] | lu (bits 44..75) |
| g | `FUN_14080dec4` « variant_name » | `+0x14` | arme, variante R(32) | lu (bits 76..107) |
| i, j | R(1), `FUN_1406cf008` | `+0x1d`, `+0x02` | drapeaux | lus (bits 108..112) |
| bloc | R(1), R(1), `FUN_1431a0abc` | `+0x2dc..+0x2e4` | horodatage | non resolu hors ligne |
| comptes | `FUN_14080cc68` | `+0xf8`, `+0x34` | cibles, composantes | lus (chemin modal) |
| boucle sur `+0x34` (pas 0xc) | R(2), R(1), `FUN_1406d3140(.,1,.)` (`0x14080c580`) | `+0x38..` | identifiant de **domaine 1** | non lu (largeur runtime) |
| boucle sur `+0xf8` (pas 0x18) | R(4), R(1) garde, [R(W=`FUN_1406d310c(6)`), R(1) ou R(4), R(16), `FUN_140c1e924`] | `+0x0fc..` | aucun identifiant | non lu |
| reference typee A | `FUN_140c9e4d8` : R(1) ; si pose, `FUN_140c9e990` : R(2) genre, `FUN_1406d3140(., genre, .)` (`0x140c9e9c7` : domaine 1 ; `0x1423e5a69` : domaine 2), [R(1) R(6)] ; puis R(1), [`FUN_1406d84b4` x 2, R(1)], `FUN_140c9e738` | `+0x27c..+0x2a0` | reference TYPEE, domaine 1 ou 2 | non lu |
| reference typee B | `FUN_1408eff64` : R(1), genre R(2), id (`0x14234fe1c` : `LEA R8D,[RSI-0x3f]` avec `ESI = 0x40`, domaine 1 ; `0x14234ffac` : `MOV R8D,0x2`, domaine 2), [R(1) R(6)] | `+0x2c8..+0x2d8` | reference TYPEE, domaine 1 ou 2 | non lu |
| visee | R(30) | `+0x28` | direction du tir | lue (bit 113, ou post-comptes + 2) |
| queue | [si `+0x2dd` nul : R(1)[R(6)], R(6)] ; [bloc : `FUN_1431a0cbc`, `FUN_1406d84b4`] sinon [`FUN_14080cb98`, `FUN_14080cb50` (+ `FUN_14320c36c`, `FUN_142a40f18`)] ; R(6) ; R(1)[`FUN_1406d84b4`] ; R(1)[`FUN_14076e494`] (le R(1) de `+0x04` n est lu que si le 5e argument est nul, donc JAMAIS en rejeu) | `+0x2e0..+0x320` | aucun identifiant de datum | non lu |

**AUCUN champ du corps n est dans le domaine 7 et AUCUN ne porte un archetype** : l arme est un
identifiant de TAG (64 bits, famille + variante), pas un datum ni un archetype de projectile. Le
seul canal qui PEUT nommer une entite de trame est ref2.

**LA MESURE** (`TestTir526`, `mouvement_5_26_tir_research_test.go`, liste d eid de
`TestTicks525`, un decodage par film) :

| | `bfecd02b` | `4f77afc1` |
|---|---:|---:|
| eid rejetes · dont premier rejet sous un record 36 | 801 · **44** | 871 · **143** |
| ref0 (domaine 1) presente · = eid rejete | 44 · **0** | 143 · **0** |
| ref1 (domaine 8) presente | **0** | **0** |
| ref2 (domaine 7) presente | **0** | **0** |
| balayage bit a bit de l etendue localisee (index 13 + tete 2) | **0 / 44** (19 modaux, 2 921 bits) | **0 / 143** (86 modaux, 12 373 bits) |
| temoin (eid d une autre cible), meme balayage | 0 | 0 |
| mot de 32 bits exact | 0 | 0 |
| TOUS les records 36 du film : ref2 presente | **0 / 2 611** | **0 / 5 079** |
| TOUS les records 36 : ref0 presente, sonde = 1 | — | 5 079 / 5 079, 5 079 |

**L eid rejete ne figure dans le record 36 du meme paquet NI a l identique NI en (index, tete) :
0 / 44 et 0 / 143.** Et le record n a jamais de reference de domaine 7 : sur les 7 690 records 36
des deux films, ref2 et ref1 sont TOUJOURS absentes ; ref0 est toujours une reference de
domaine 1 a sonde 1 (index sur 9 bits relatif a la bande de bipedes, `event_list.go`) ; qu elle
soit le TIREUR n est pas verifie ici.

## 4. VERDICT

- **(a) ne tient pas** : `FUN_142988084` (type 6) ecrit un indice de joueur en
  `DAT_144c2328c`, consomme par la seule vue Theater `FUN_142e2e104` ; `FUN_1429882c8` (0xb) jette
  son bloc ; `FUN_1429875e4` (0xc) alimente la table de bots par `FUN_142c26748`. Ni la garde
  `FUN_1406caad8`, ni la table de datums, ni une declaration d entite. Mesure : contenu constant
  (0, 0), position structurelle d en-tete de chunk, 2 / 55 contre un taux de base de 2,56 %.
- **(b) ne tient pas** : le record 36 ne nomme aucune entite de trame (`FUN_14080a9d4` : ref2
  domaine 7 JAMAIS presente, 0 / 7 690 ; corps `FUN_14080c1f8` : identifiants de domaine 1 ou 2
  seulement) et ne porte aucun archetype. Mesure : **0 / 44** et **0 / 143**.
- **Donc pas de 5.26.3** : aucune liaison, aucune ligne de production, `grammar.Rev`,
  `facts.Rev` et `replay.SchemaVersion` (68) INCHANGES, aucun backfill.

## 5. LE RESTE, CHIFFRE (inchange)

`TestGate516` sur la base `43a01721e` : **3 940 / 30 387** trames delta fermees a reste NUL,
26 397 abandonnees, 50 debordements, 16 129 rejets hors datum, 254 liaisons par anticipation.
Ticks de bipede de joueur perdus : **85 843 / 242 711 = 35,4 %** sur `bfecd02b`,
**460 625 / 857 388 = 53,7 %** sur `4f77afc1` (5.25). Aucun de ces chiffres ne bouge : le lot ne
touche pas une ligne decodee. Les 801 et 871 eid rejetes restent des entites nees et mortes entre
deux images-cles, dont la naissance n est portee ni par une image-cle, ni par le bloc de type 1,
ni par un `NEW` lu, ni par les paquets 6 / 0xb / 0xc, ni par le record 36 du paquet du premier
rejet.
