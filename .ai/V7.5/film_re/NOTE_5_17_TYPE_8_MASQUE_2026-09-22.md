# Note 5.17 — Le paquet de type 8 est la population de la session, et le decalage du masque n a pas d image hors ligne (2026-09-22)

> Lot 5.17, branche `feat/decfilm-67`, base `58b4aba74` (integration des lots 5.1 a 5.16,
> `grammar-2026-09-22.6`, schema 67). Cette note est le DOUBLE de la section du plan
> (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « Post-chantier — lot 5.17 ») : le plan fait foi sur
> l avancement, cette note sur la GRAMMAIRE.
>
> Elle prend la suite de `NOTE_5_16_MODELE_RANG_1_2026-09-22.md` (§6 et §7, les maillons
> restants), dont elle referme le premier : D1 (5.16).

## 1. Le paquet de type 8 : sa grammaire, de la branche du repartiteur jusqu aux feuilles

`FUN_1428e22c0` aiguille sur `*(short *)paquet`. Sa branche `sVar2 == 8` est la SEULE qui
interroge `&DAT_144c23178` :

```
cVar3 = FUN_1428e1e94(&DAT_144c23178)                 ; ou vit la structure du film charge
lVar7 = cVar3 ? *(param_1+0x120) + 0x130 : *(param_1+0x108)
FUN_142987bd4(session, paquet, *(u32 *)(lVar7 + 4))   ; le 3e argument = LA VERSION DE FORMAT
```

`*(u32 *)(lVar7 + 4)` est exactement ce que rend `FUN_1428e1c0c` : la version de format du
registre inflate de `chunk_00`, que le depot lit depuis le lot 1.9.1 ter par
`FilmFormatVersionFromHeader` (`chunk_00 + 4`). **`&DAT_144c23178` n est donc pas un porteur de
table de composants : c est le singleton du film charge**, et la branche de type 8 ne lui demande
QUE l adresse de cette structure.

| maillon | adresse | ce qu il fait |
|---|---|---|
| le repartiteur | `FUN_1428e22c0` | types 0, 1, 2, 6, 7, 8, 9, 10, 0xb, 0xc ; defaut -> `FilmBlockReadError` |
| le lecteur du type 8 | `FUN_142987bd4` | le sujet de ce §, ci-dessous |
| la charge | `FUN_142988338(session, tampon, taille, 0)` | `memcpy` PUR du curseur d octets de la session (`session+0xf8`, base `+0xf0`, longueur `+0xe8`), curseur avance de `taille`. Aucune detente, aucun dechiffrement |
| le lecteur de bits | `FUN_1424c7b4c(&lecteur, tampon, taille)` puis `FUN_1406d5cc0(&lecteur, 3)` | mode 3 : `FUN_1406d6d34` amorce l accumulateur, le curseur de bits part a ZERO |

```
FUN_142987bd4(session, paquet, version) :
  N = R(32)                                     ; @142987c5a..142987d18 -> FUN_142975788(vec, N)
  pour chaque entree (stride 0x1440 octets) :
     si version >= 0x15 : FUN_1407f2058 = R(1) porte ; si 0 -> R(5)     ; entree+0x00
     FUN_1406d676c(..., entree+0x08, 0x40) = R(64)                      ; entree+0x08
     FUN_1407eeba4(entree+0x10, &lecteur)                               ; LE CORPS
  controle @142987d6e : bitsLus <= taille*8 ET pas de drapeau d erreur -> sinon REJET
```

et le corps, `FUN_1407eeba4`, dans son ordre exact (toutes les largeurs sont des LITTERAUX du
desassemblage, en `R9D` pour `FUN_1406d676c`) :

```
FUN_1407f01c4 :  n  = FUN_142bdeddc = R(11) + 1 ; n x R(1)      ; rec+0x000 (bitmap 0x100 o)
                 L1 = FUN_1411b1bd8 = R(12)     ; R(L1 * 8)     ; rec+0x100 / +0x108
                 L2 = FUN_1411b1b04 = R(8)      ; R(L2 * 32)    ; rec+0x908 / +0x910
R(0x340)                                                        ; rec+0xc48
FUN_1407f0094(..., 0x10) : jusqu a 16 x R(16), ARRET APRES le mot NUL  ; rec+0xc14
R(0x80)                                                         ; rec+0xc38
FUN_14080dec4 = R(32)  (« desired-representation »)             ; rec+0xcb0
FUN_1406d676c(..., 0x40) = R(64)                                ; rec+0xcb8
FUN_1407effb8 = R(10)                                           ; rec+0xc12
FUN_1407efedc = R(14)                                           ; rec+0xc36
FUN_1407ef724 = R(6), puis DEC (valeur - 1)                     ; rec+0xc35
R(8) inline                                                     ; rec+0xc10
FUN_1407eed64 = R(7)                                            ; rec+0xc34
FUN_1406cf008 = R(1)                                            ; rec+0xc11
R(0x39e0)   <- LARGEUR PAR BUILD, pas un litteral (cf. §1.2)    ; rec+0xcc0
R(0x160)    (appel terminal)                                    ; rec+0x1400
```

Les offsets CHAINENT sans trou : `0xc38 + 16 o = 0xc48`, `0xc48 + 104 o = 0xcb0`,
`0xcb0 + 4 = 0xcb8`, `0xcb8 + 8 = 0xcc0`, `0xcc0 + 1852 o = 0x13fc`, `0x1400 + 44 o = 0x142c`,
et l entree mesure `0x1440`. C est ce chainage qui prouve que `R9D` est bien la LARGEUR EN BITS.

### 1.1 Ce que le type 8 porte : la population de la session

Apres la boucle, `FUN_142987bd4` apparie chaque entree du paquet avec la liste VIVE des joueurs
par la cle de 8 octets `sub+0xcb8` (`CMP [RAX+0x8],RCX` @142987e73, contre `[joueur+0xcb8]`
@142987e01) :

* entree sans correspondant -> `FUN_1424d8a8c(..., iVar13, ...)`, `iVar13` balayant les index de
  manette 0..0x1f (`FUN_140b763e4`) : le joueur est AJOUTE ;
* entree avec correspondant -> `FUN_1424d512c` puis `FUN_141e2f400(objet+0x540, ...)` : MIS A JOUR ;
* joueur vivant que le paquet ne nomme plus -> `thunk_FUN_142b759a8` + `FUN_142b7f6b8` : RETIRE.

**Ce n est donc pas une table de composants, c est le roster de la session**, reemis a intervalle
regulier. Et c est pour cela que le type 8 pese 700 868 octets sur `bfecd02b` (8 joueurs) contre
20 622 sur `dad793c7` (1 joueur) : le chiffre que D1 (5.16) prenait pour l indice d une table de
compatibilite est simplement le produit du nombre de joueurs par la taille d un enregistrement.

### 1.2 Le corps EST celui de la table de 32 joueurs de `chunk_00` (lot 1.5.2)

Champ pour champ, largeur pour largeur, `FUN_1407eeba4` lit ce que `FUN_1407edea8` ecrit dans la
table de 32 joueurs de `chunk_00` — que le depot lit depuis le lot 1.5.2 (`player_table.go`,
`player_table_record.go`), avec un corpus de 1 351 films et 2 121 ecarts predits sur 2 121. Le
lot 5.17 a d abord ecrit une seconde copie de ce corps ; elle est supprimee (commit
`5.17.1 bis`). `decodeSlotQueue` est scinde en `decodeSlotCorps` (le corps COMMUN) plus le `u32`
de `slot+0x1448`, qui n existe que dans `chunk_00` — deplacement PUR, le diff hors commentaires
est l appel plus un `saute` deplace d une ligne.

Deux differences seulement, et les deux sont MESUREES :

1. **L en-tete.** `chunk_00` : 85 bits de champs (trois booleens, `u32`, 2 bits, jeton de 48 bits)
   puis le XUID. Type 8 : la porte `R(1)[R(5)]` (a partir de la version de format `0x15`) puis le
   XUID, et rien d autre.
2. **L ordre d octets du XUID.** `FUN_1406d676c` depose les octets du flux DANS L ORDRE DU FLUX
   (`*param_3 = BSWAP64(accumulateur)`, @1406d67cb) et le champ de structure est relu en
   little-endian : la valeur est l INVERSION D OCTETS de la lecture MSB-first. Sans cette
   inversion, `entree+0x08` de `dad793c7` vaut `0xe101d0c4f3010900` (du bruit) ; avec, il vaut
   `0x000901f3c4d001e1` — un XUID Xbox. La table de `chunk_00`, elle, est ecrite par un lecteur de
   bits et se lit MSB-first (`slotXuidLo` / `slotXuidHi`).

**Et la cle de jointure `sub+0xcb8` est LE MEME XUID**, dans le meme ordre d octets : mesure sur
les huit joueurs de `bfecd02b` (`0x6945f23ff0010900` <-> `0x000901f03ff24569`, octet pour octet).
Le champ que la table de `chunk_00` publie en `PlayerSlotShorts.Q64` sans le nommer est donc lui
aussi le XUID — ce qui explique que le jeu apparie sur lui.

Consequence de la factorisation : la largeur du bloc de `rec+0xcc0` n est PAS le litteral `0x39e0`
de ce build, c est `profile.PersonnalisationOctets(build) * 8` (1 852 o sur `HI_1_12_0` et
`HI_1_13_0`, 1 492 sur `HI_1_10_0`/`HI_1_11_0`, 1 312 sur `HI_1_8_0`/`HI_1_9_0`, 2 052 sur
`HI_1_4_1`). Un build inconnu est un ARRET, jamais une lecture au profil du voisin (ADR 0034 D-4).

### 1.3 Le maillon qui manquait, et il tenait a un `+ 1`

`FUN_142bdeddc` ne rend pas `R(11)` : il rend **`R(11) + 1`** (`LEA EAX,[R9 + 0x1]` @142bdeea4).
Sans ce `+ 1`, tout ce qui suit dans l entree glisse d un bit, et les deux longueurs prefixees
sortent de leurs bornes de structure — mesure sur `bfecd02b` : `L1 = 2 399` pour un champ de
2 048 octets, `L2 = 255` pour un champ de 192 mots, et les entrees 2 a 8 d un paquet decodees sur
du bruit. Le maximum de `R(11) + 1` est 2 048, ce qui remplit EXACTEMENT les `0x100` octets de
bitmap que `FUN_1407f01c4` met a zero avant de les remplir par `FUN_1407688b0(dest, k, bit)`.

C est le meme `+ 1` que `player_table_record.go` porte depuis le lot 1.5.2
(`slotMaskPrefixBits` + 1), lu a dix jours d ecart sur une AUTRE fonction : les deux lectures
tombent au meme endroit.

### 1.4 Le jeu ne verifie que le debordement

```
142987d6e: MOV [RBP-0x40],0x5
142987d75: MOV EAX,[RBP-0x48] ; SHL EAX,0x3        ; taille * 8
142987d7b: CMP [RBP-0x34],EAX ; SETG CL            ; bitsLus > bitsDisponibles ?
142987d81: CMP [RBP-0x3c],R14B ; JNZ erreur        ; drapeau d erreur du lecteur
142987d87: TEST CL,CL ; JNZ erreur
```

Le paquet est rejete si le curseur DEBORDE ou si le lecteur a leve son drapeau. La
SOUS-consommation est acceptee. Le gate du depot est plus strict : reste dans `[0 ; 7]` ET tous
ses bits a ZERO (le bourrage d octet est ecrit a zero).

### 1.5 Le gate du 5.17.1, et le recensement de types

| mesure | `dad793c7` | `bfecd02b` |
|---|---:|---:|
| version de format (`chunk_00 + 4`) | 27 | 27 |
| build / bloc de personnalisation | `HI_1_13_0` / 1 852 o | `HI_1_13_0` / 1 852 o |
| paquets de type 8 | 6 (20 622 o) | 28 (700 868 o) |
| entrees annoncees / lues / refusees | 6 / 6 / 0 | 224 / 224 / 0 |
| DEBORDEMENTS | **0** | **0** |
| reste par paquet | **4 bits** | **1 bit** |
| **paquets fermes a bourrage NUL** | **6 / 6** | **28 / 28** |
| XUID distincts (`entree+0x08`) | 1 | **8** |
| etiquettes distinctes (`rec+0xc14`) | 1 (« Neutral ») | **8 gamertags** |

Les huit entrees d un paquet de `bfecd02b` chainent au bit : 32 -> 27 970 -> 55 828 -> 83 790 ->
100 322 -> 127 782 -> 155 752 -> 183 635 -> 200 247, sur 200 248 bits de payload. Les huit
gamertags : Tataaannn, JGtm, Chocoboflor, Draconewt, MEK1906, Madina97294, SHN Lups99,
indahoopty8751 — et les huit XUID sont tous en `0x0009...`.

Le recensement de types, re-mesure HORS LIGNE, reproduit a l octet le tableau du §2.2 de la note
5.16 (qui l avait releve chez l ecrivain) :

| type | `dad793c7` | `bfecd02b` |
|---:|---:|---:|
| 0 (trame) | 5 370 paq. / 56 275 o | 31 232 paq. / 7 203 888 o |
| 1 (saute par le jeu) | 5 / 1 715 095 o | 27 / 9 261 513 o |
| 2 (image-cle) | 5 / 650 107 o | 27 / 4 593 116 o |
| 6 | 5 / 20 o | 27 / 108 o |
| 7 (`CHUNK_END`) | 6 / 0 o | 28 / 0 o |
| **8 (roster)** | **6 / 20 622 o** | **28 / 700 868 o** |
| 9 | 1 / 4 o | 1 / 631 561 o |
| 10 | 5 370 / 26 850 o | 31 232 / 251 227 o |
| 12 | 6 / 24 o | 28 / 112 o |

**Avant ce lot, le type 8 n etait ni lu ni compte** : le gate des paquets filtre sur
`PacketTypeDelta` (type 0), donc un paquet de type 8 lui etait invisible — il ne comptait pas
comme reste, il ne comptait pas du tout.

## 2. Le decalage du masque : D1 (5.16) est referme, et ce n etait pas un defaut du depot

### 2.1 Les quatre adresses

| maillon | adresse | ce qu il fait |
|---|---|---|
| l iterateur de DELTA | `FUN_14076cb60` | parcourt le descripteur d archetype DU PROCESSUS (`*(int *)(desc+0x4320)` composants, deserialiseurs en `desc + i*8`, `ti` en `desc+0x474c`) et teste `masque >> ((i - decales) & 0xff)` |
| le filtre | `FUN_1428e1dac(&filmSingleton, ti, nom)` | cherche `nom` dans le bloc `ti` du registre de `chunk_00` : `base + 8 + ti*0x4100`, pas de `0x104`, 64 entrees. FAUX -> `decales++`, ZERO bit consomme |
| le niveau | `FUN_1428e1b50(&filmSingleton, ti, nom)` | MEME cadrage, rend le `u32` en `entree + 0x100` (defaut 1). Si `deser->vtable[0x10](niveau)` est vrai -> `decales++`, ZERO bit : **c est la SECONDE sortie, que le 5.16 n avait pas vue** |
| la garde des deux sorties | `FUN_1404f2b4c()` | `TLS+0x238` porte un nom de contexte non vide ET `*((DAT_1445c5838 & 0xffff)*0x1134f0 + 0xea71c + DAT_145121d28) == 2`. Vrai UNIQUEMENT en rejeu de film |
| **l iterateur d ETAT COMPLET** | **`FUN_142e2c690`** | **et c est lui qui tranche** |

Le pseudo-code complet de `FUN_14076cb60` :

```
FUN_14076cb60(desc, args) -> bool
  lecteur = args[5]
  FUN_1406d7610(desc, lecteur, &masque)          ; LE MASQUE, avant tout
  extra = FUN_14076cea8()                        ; rejeu ? DAT_144c23326 : DAT_1450e24e8
  ok = true ; decales = 0
  pour i de 0 a *(int *)(desc+0x4320)-1 :
     d = *(ptr *)(desc + i*8)
     si FUN_1404f2b4c() et DAT_144c232e1 == 0 :
        si !FUN_1428e1dac(&filmSingleton, desc[0x474c], d->nom()) :
            decales++ ; continue                 ; ZERO BIT, ET LE MASQUE GLISSE
     si (masque >> ((i - decales) & 0xff)) & 1 :
        niveau = d->vtable[0x00]()               ; le niveau du DESCRIPTEUR
        si FUN_1404f2b4c() et DAT_144c232e1 == 0 :
            niveau = FUN_1428e1b50(&filmSingleton, ti, d->nom())     ; le niveau DU FILM
            si d->vtable[0x10](niveau) : decales++ ; continue        ; ZERO BIT, ET IL GLISSE
        prediction = (args[4] == 0) ? 0 : d->vtable[0x48](tampon, &args[3])
        ok = ok && d->vtable[0x28](lecteur, args, &prediction, niveau)
        si extra et FUN_1406cf008(lecteur) :
            R(32) != 0x0bcddcba -> « entity component corrupt: entity id:[0x%x] type:[%s]
                                     component:[%s] »
        args[0] bitset |= 1 << desc[0x4850 + i]  ; le bit de changement
        si !ok : args[0][0..3] = 0 ; return false
  return ok
```

Et `FUN_1406d7610` est confirme identique a `consumeMask` du depot : `R(1)` porte ; si 0,
`R(3)` cardinal puis `cardinal x R(6)` d index epars ; si 1, `R(64)` dense. Il rend le nombre de
bits consommes (4, `4 + 6n`, `0x41`) et son `param_1` — le descripteur — n est PAS utilise : la
largeur du masque ne depend donc pas du nombre de composants.

### 2.2 `FUN_142e2c690` tranche : l index du jeu EST l index du registre du film

```
FUN_142e2c690(descRuntime, lecteur, args, param_4 = &entree 0 du bloc ti du registre DU FILM) :
  extra = FUN_14076cea8()
  pour k de 0 a 63 :
     si *param_4 == 0 : sortie                        ; arret au premier nom vide
     niveau = *(u32 *)(param_4 + 0x100)               ; LE NIVEAU DU FILM
     d = *(ptr *)(descRuntime + k*8)                  ; d abord la POSITION
     si d->nom() != param_4 :                         ; sinon RECHERCHE PAR NOM sur 64 slots
        chercher d parmi descRuntime[0..63] tel que d->nom() == param_4
        si introuvable : DAT_144e61ea0 = 0 ; return 0
     si !d->vtable[0x28](lecteur, args, &pred, niveau) : return 0
     si extra et FUN_1406cf008(lecteur) : R(32)
     param_4 += 0x104
```

Aucun decalage, aucun masque : sur le chemin d etat complet, le jeu parcourt le bloc du registre
DU FILM entree par entree et resout le deserialiseur PAR NOM. **`i - decales` est donc la
conversion « descripteur du build qui REJOUE » -> « registre du film »**, et rien d autre.

### 2.3 Pourquoi le depot n a rien a porter

`arch.Components` EST le registre du film : `registry.go` lit les entrees NOMMEES du bloc `ti` de
`chunk_00` inflate avec les MEMES constantes de cadrage que le filtre — `registryEntryBase` = 8,
`archetypeBlockSize` = `0x4100`, `registrySlotSize` = `0x104`, `archetypeBlockSlots` = 64
(= la borne `0x3f` de l ecrivain), `registryEntryLevelOffset` = `0x100`. Le decodeur hors ligne
n a pas de descripteur de processus : il est DEJA du cote de l arrivee de la conversion. Porter
`i - decales` appliquerait deux fois la meme conversion.

Ce n est pas une absence, c est une EGALITE, et elle a deux moities. Les deux sont figees par
`masque_cadre_registre_test.go` :

| test | ce qu il fige |
|---|---|
| `TestCadreDuRegistreEstCeluiDuFiltre` | les cinq constantes de cadrage contre les litteraux de `FUN_1428e1dac` / `FUN_1428e1b50`, plus `64 x 0x104 == 0x4100` |
| `TestMasqueIndexeSurLEntreeDuRegistre` | la boucle DE PRODUCTION, masque a un seul bit, pour chaque index : le composant consomme est celui de l index de ce bit. Un port de `i - decales` ferait echouer tous les cas sauf `k = 0` |
| `TestNiveauVientDeLEntreeDuRegistre` | le niveau passe au deserialiseur vient de `entree + 0x100`, ce que `FUN_142e2c690` passe a `vtable[0x28]` |

## 3. Ce que le lot lit en plus, et ce qu il ne bouge pas

**Nouvellement lu** : le roster de type 8 — 6 + 224 enregistrements de joueur, 9 XUID Xbox et
9 etiquettes sur les deux temoins, avec `desired-representation`, le jeton de session et les six
champs courts. Rien d autre.

**Rien de nouveau sur le chemin de trame**, et c est verifiable sur pieces : le diff de production
depuis `58b4aba74` ne contient que des commentaires, l extraction PURE de `decodeSlotCorps`, et
`roster_type8.go` — qui n a AUCUN appelant de production. Aucune etiquette de composant
supplementaire, aucun archetype supplementaire, aucun canal d etat de bipede.

**`replay.SchemaVersion` reste a 67.** Pas de chronique v68, pas de fixture `replay_schema_68_*`,
pas de jumeaux `replaydoc`/`replayview`, pas de zod, pas d OpenAPI : il n y a rien a publier
puisque le document de rejeu ne change pas d un octet. `grammar.Rev` reste
`grammar-2026-09-22.6` — decision ECRITE : le lecteur de roster n a pas d appelant de production,
donc la sortie de la couche ne peut pas changer ; `facts.Rev` ne monte pas, aucun backlog
killsource n est ouvert.

**Le gate des paquets de trame tient par construction** : le tableau du 5.16.4 (`dad793c7`
5 354/5 365 a reste NUL, `bfecd02b` 2 884/30 387, 2 et 32 debordements, `ti=35` 75 et 129 572,
desyncs 0 et 4, rejets hors datum 2 et 23 769, rejets de vue 0) n est pas re-mesure, parce que
`TestGate516` exige une carte du catalogue et que les deux temoins n ont pas de nom de match
(report D3 (5.11) : **une carte substitut donne des positions fausses SANS ERREUR**). Le mesurer
au hasard serait publier un chiffre faux ; le diff, lui, PROUVE que le chemin est inchange.

## 4. Le residu de `bfecd02b` : D1 est ecarte, et le suspect suivant a une adresse

D1 (5.16) designait le decalage du masque comme premier suspect des 23 769 rejets. Le §2 l ecarte
par les adresses : le decalage n a pas d image hors ligne. Ce que la lecture de `FUN_14076cb60`
laisse en revanche OUVERT, et c est le suspect suivant :

**`FUN_14076cea8()` — LE CONTROLE DE CORRUPTION PAR COMPOSANT EST DECIDE PAR UN DRAPEAU DU FILM,
ET LE DEPOT NE LE LIT PAS.**

```
FUN_14076cea8() :
  si FUN_1404f2b4c() : return DAT_144c23326        ; EN REJEU DE FILM
  sinon              : return DAT_1450e24e8        ; en jeu vif
```

`DAT_144c23326` est a `DAT_144c23178 + 0x1ae` : c est un CHAMP DU SINGLETON DU FILM CHARGE. Il est
nul dans l image et n a qu une seule reference croisee, la lecture de `FUN_14076cea8` — il est
donc rempli au chargement du film, depuis le film. Or c est lui qui decide si chaque composant
PRESENT est suivi d un `R(1)` de garde (et, si ce bit vaut 1, d un `R(32)` sentinelle
`0x0bcddcba`). Cote depot, le drapeau equivalent est `GrammaireBalayage.ControleDeCorruption`, de
defaut **FAUX**, pose UNIQUEMENT par des instruments et **jamais lu dans le film**.

Un drapeau leve coute au moins un bit PAR COMPOSANT PRESENT. C est exactement la signature du
residu : un film a un joueur ferme 5 354/5 365, un film dense a 207 etiquettes de composant ferme
2 884/30 387. Le maillon a trouver est **l octet du film qui remplit `DAT_144c23178 + 0x1ae`** —
c est-a-dire le chemin de chargement du film qui ecrit ce champ, a chercher parmi les ecrivains du
singleton (meme famille que `FUN_1428e1c0c` / `FUN_1428e1e94`, qui lisent `+0x100`, `+0x108`,
`+0x120`).

Le second champ de la meme famille, `DAT_144c232e1`, est le KILL-SWITCH des deux sorties du
filtre ET de la substitution de niveau (`FUN_14076cb60` @14076cc19 et @14076cc7d, plus
`FUN_142e35a58` cote ecrivain d image-cle) : lui aussi vient du film et n est pas lu.

### 4.1 Decouvertes hors perimetre, consignees et NON traitees

| # | decouverte | ou la reprendre |
|---|---|---|
| **D1 (5.17)** | **LE CONTROLE DE CORRUPTION PAR COMPOSANT VIENT DU FILM** (`FUN_14076cea8` -> `DAT_144c23326` = `DAT_144c23178 + 0x1ae`, une seule xref, nul dans l image). Le depot porte `ControleDeCorruption` de defaut FAUX, jamais lu dans le film. Un drapeau leve coute >= 1 bit par composant present. | **le lot du residu de `bfecd02b`** : trouver l ecrivain de `DAT_144c23178 + 0x1ae` dans le chemin de chargement du film, puis lire le drapeau hors ligne. C est le suspect qui remplace D1 (5.16) |
| **D2 (5.17)** | **`DAT_144c232e1` EST LE KILL-SWITCH DU FILTRE DE COMPOSANTS ET DE LA SUBSTITUTION DE NIVEAU** (`FUN_14076cb60` @14076cc19 / @14076cc7d, `FUN_142e35a58`), et il vient lui aussi du singleton du film. | le meme lot que D1 (5.17) : les deux champs se remplissent au meme endroit |
| **D3 (5.17)** | **LE PAQUET DE TYPE 9 PESE 631 561 OCTETS SUR `bfecd02b` CONTRE 4 SUR `dad793c7`, ET LE JEU LE SAUTE LUI-MEME.** Sa branche du repartiteur avance le CURSEUR D OCTETS de la session de la taille du bloc et rend 1 : `*(int *)(*(param_1+0x130) + 0xf8) += *(int *)(paquet + 2)` — `session+0xf8` est exactement le curseur que `FUN_142988338` consomme. C est donc un troisieme bloc saute par construction, apres les types 1 et 2 (releve du 5.16). Trois autres types ont un handler que le depot ne lit pas : 6 (`FUN_142988084`, resultat range en `session+0x114`), 0xb (`FUN_1429882c8`), 0xc (`FUN_1429875e4`). | le lot qui cherchera une source d etat que ni la trame ni l image-cle ne portent. Le type 9 est le deuxieme plus gros bloc du film dense apres le type 1, et son handler dit qu il n est PAS destine a ce repartiteur |
| **D4 (5.17)** | **`TestTableJoueursCorpus` EST ROUGE SUR LE CACHE DE CE POSTE, ET C ETAIT DEJA LE CAS AVANT CE LOT** — verifie en restaurant `player_table_record.go` a `58b4aba74` : meme verdict, meme duree. Le cache porte 1 610 `chunk_00` (le test a ete ecrit pour 1 351) ; le bilan dit 1 604 films lus a 32 slots, 1 594/1 604 en accord avec l oracle, 0 contradiction de grammaire, mais un build `HI_1_5_1` **absent du profil** et 10 films ou l oracle de l instrument « perd la tete de la table » au-dela de 40 000 bits. Le test est hors du gate du depot (il SKIPPE sans `CHUNK00_CORPUS`). | le lot qui reprendra la table de `chunk_00` : ajouter la ligne `HI_1_5_1` au profil avec sa provenance (D-4), et decider si l oracle de l instrument doit suivre la croissance du cache ou si c est le test qui doit borner son corpus |

## 5. Les instruments

`mouvement_5_17_type8_research_test.go`, `//go:build research`, paquet `grammar`. Il n a besoin
QUE de `MOUV511_FILM` : le paquet de type 8 ne porte aucune position, donc aucune largeur d axe
n entre dans sa grammaire, et poser une carte substitut serait exactement le piege du report
D3 (5.11).

| test | role |
|---|---|
| `TestType817Population` | LE GATE du 5.17.1 : paquets, entrees annoncees/lues/refusees, debordements, reste, fermeture a bourrage NUL, XUID et etiquettes distincts |
| `TestType817Etapes` | position et longueur de CHAQUE entree, plus ses champs nommes. C est l instrument qui a LOCALISE le `+ 1` de `FUN_142bdeddc` |
| `TestType817Dump` | le payload en hexadecimal, les motifs UTF-16 alignes et les plages de zeros — l instrument qui a montre « Neutral » dans la queue non lue avant la correction |
| `TestType817Types` | le recensement de types de paquet du film, avec ses octets |

Gate sans decodage : `roster_type8_synth_test.go` (quatre tests, dont le decompte de bits au bit
contre un ecrivain synthetique) et `masque_cadre_registre_test.go` (trois ratchets).
