# Note 5.13 — Le rang de la vue, et ce que la « vue 3 » est vraiment (2026-09-22)

> Lot 5.13.1, branche `feat/decfilm-63`. Le plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`,
> section « Post-chantier — lot 5.13 ») fait foi sur l avancement, cette note sur la GRAMMAIRE.
> Elle CORRIGE deux lectures du journal RE et de la passation 5.11.

## 1. Les deux bits de tete d un identifiant d image-cle sont le RANG DE LA VUE

L encodeur de la liste de REFERENCE d une vue de replication est `FUN_142f2e174`
(`replication_entity_manager_view.cpp`), slot `+0x10` de la vtable de vue `0x1436a87e0`. Il
parcourt la table d entites DE SA VUE (`vue+0x38` a `vue+0x40`, pas de `0xa0`, bitmap de presence
`vue+0x58`, masque d interet par vue `1 << vue[0x14]`) et ecrit, par entite retenue, UN mot de
32 bits :

```
*mot = *(int *)(vue + 8) << 0x1e | *mot & <masque> | slot & 0x1fff | <genre>
FUN_140bbd808(mot, priorite)      // *mot = *mot & 0xff801fff | (priorite & 0x3ff) << 0xd
```

soit `[rang:2 @30][genre:2 @23][priorite:10 @13][slot:13 @0]` — l `id:32` de l en-tete
`[id:32][field:26][ti:6]` que `keyframe_world.go` lit deja.

**Les deux bits de tete viennent de `vue + 8`, pas de l entite**, et c est verifie sur
l instruction : `142f2e2ec MOV ECX, dword ptr [RDI + 0x8]` puis `142f2e304 SHL ECX, 0x1e`, ou
`RDI` est `param_1`, c est-a-dire la vue (meme registre que `vue+0x38`, `vue+0x58`, `vue+0x14`).
Les trois sites de genre (`142f2e304`, `142f2e38a`, `142f2e440`) lisent le meme champ.

**Et `vue + 8` est le RANG de la vue** : le registraire `FUN_1409c9860(conteneur, rang, vue)`
l y ecrit — `*(int *)(param_3 + 1) = param_2` — en rangeant la vue dans le tableau que
`FUN_142987460` parcourt. `FUN_141f855b4` l appelle pour les rangs 0, 1 et 2.

Le journal RE du lot G (2026-08-27, `WALK_PORT_NOTES.md` §1) nommait ce champ `gen` sans avoir
decompile l instruction. Ce n est ni une generation, ni un identifiant d entite.

## 2. Ce que le film porte, mesure

| film | paquets d image-cle | records par paquet | rangs distincts |
|---|---:|---|---|
| `dad793c7` | 5 | 123 a 186 | **{1}** |
| `bfecd02b` | 60 | 424 a 482 | **{1}** |

`TestImageCle513Vues`. L image-cle d un film est donc MONO-RANG : elle enumere la table d UNE
vue, celle du gestionnaire d entites. Et la vue que la marche hors ligne parcourt en PREMIER est
celle qui rend les records (`TestVues513EspaceDeNoms`) : `dad793c7` 5 628 records dont tag 1 sur
**5 628 / 5 628** ; `bfecd02b` **157 250 / 157 554** (99,81 %).

« Toutes les liaisons d image-cle vont en vue 0 » n etait pas une limite du portage — c est ce
que le film porte. Le port (`World.BindImageCle`) le LIT desormais au lieu de l attribuer
d office : premier rang rencontre -> la vue de la marche ; tout autre rang -> vue INCONNUE, qui
ne rejette rien.

## 3. Le flux DELTA n ecrit PAS le meme champ — et le confondre coute des records

`FUN_142f30610` ecrit l identifiant que la table de la vue porte
(`uVar3 = *(uint *)(slot * 0xa0 + 8 + vue[0x38])`, passe a `FUN_142f2c754`), et cet identifiant
est pose par `FUN_1408f1730` a `*(byte *)(datum + 1) << 0x1e | slot` : un champ du DATUM, par
entite. Exiger l egalite des deux dans la garde de table de vue coute **21 records `ti=35`** sur
`bfecd02b` (114 458 -> 114 437). `World.VuePossede` compare donc le SLOT, et le dit sur place.

## 4. « L entite 7140 generation 2 de la vue 3 » n existe pas

`FUN_142987460` appelle `vtable[0x40]` sur ses trois vues. Les trois vtables (`/read_memory`)
n ont PAS la meme fonction a ce slot :

| vtable | `+0x40` | grammaire de la boucle de records |
|---|---|---|
| `0x1436a8700` | `FUN_14076a1c4` | si `vue[0x11]` : **zero bit** ; sinon boucle `R(1)` (0 = fin) puis UN corps (`FUN_14080a9d4`). **Rend toujours zero record** (`*param_6 = 0`) |
| `0x1436a87e0` | `FUN_1406cd128` | le GESTIONNAIRE D ENTITES : `[R(32) film]`, `prefixe R(1)`, `si 0 -> type R(2)`, `idLow + tag 2`, corps selon le type. **La seule grammaire que la marche porte** |
| `0x1436a8770` | `FUN_1406cf548` | `[prologue FUN_142f2539c si drapeau]` puis boucle `R(1)` (0 = fin), `kind R(2)`, trois handlers (`FUN_1406d0388` / `FUN_142f29b38` / `FUN_142f29e54`), `kind == 3` = zero bit |

Les deux « en-tetes de record rejetes » du pied de trame (lot 5.11.7) ne sont donc pas des
en-tetes du gestionnaire d entites : ce sont les flux des DEUX AUTRES classes de vue, lus avec la
grammaire du gestionnaire. `slot 7136 tag 0` et `slot 7140 tag 2` ne designent AUCUNE entite — ce
sont `R(1)` + `kind R(2)` + du corps d une autre grammaire, decoupes en `[prefixe][idLow][tag]`,
et les bits 27 / 30 qui basculaient au decollage sont des bits DE CE CORPS.

Controle sur le film : sur `dad793c7` les vues de rang 1 et 2 ne rendent que **13 records**, TOUS
de type DEL (slots 0, 260, 261, 262) — jamais un NEW, donc jamais un archetype ; et le slot 7140
n apparait dans aucun record d image-cle (elles s arretent au slot 1 345).

**La piste « nommer l entite de la vue 3 » est fermee par la grammaire, pas par une absence** :
l objet a nommer n existe pas.

## 5. Ce que cela ouvre — la route vers 100 % de paquets fermes

Porter les DEUX autres boucles de records (`FUN_14076a1c4`, `FUN_1406cf548`) est la route vers la
fermeture complete des paquets : les 13 records DEL fantomes de `dad793c7` et ses 18
debordements viennent tous de la grammaire du gestionnaire appliquee a ces deux flux. C est un lot
a soi seul (trois handlers a mesurer pour `FUN_1406cf548`, un pour `FUN_14076a1c4`), consigne au
§4 du plan.

## 6. Le mantling : la queue d `i54` est la charge utile d un message reseau nomme (lot 5.13.2)

La queue d `i54` (`FUN_1407ea38c` en ecriture, `FUN_1408f02c8` en lecture, bloc = `etat + 0x11f8`)
est la CHARGE UTILE du message reseau `initiate_mobility_action` :

| maillon | adresse | ce qu il dit |
|---|---|---|
| nom du type | `143c97470` | `initiate_mobility_action` (thunk `1411685a0`, vtable `143d0a000`..`143d0a118`) |
| trace | `143e0c500` | `"biped-initiate-mobility-action: relevance = %5.3f"`, pertinence `FUN_142e30074` |
| ECRITURE | `FUN_1407ebc58` | `R(1) = bloc[0x9d]` puis `FUN_1407ea38c(bloc, writer)` |
| LECTURE | `FUN_142ef8f04` | `bloc[0x9d] = R(1)` ; `bloc[0x9e] = 0` ; `FUN_1408f02c8(bloc, reader)` |

**La garde `bloc[0x9d]` est donc NOMMEE : c est le bit de presence du message.** Dans le composant
`i54` elle vaut `etat + 0x1295` = `flag1`, lu deux lignes plus haut par le meme deserialiseur —
la lecture du depot est confirmee par un second site, ecrit independamment. Les trois vec3 de la
queue sont les parametres geometriques d un message d AMORCE d action de mobilite.

**Les quatre valeurs de `bloc + 0x9c` (= `etat + 0x1294`) ne sont PAS nommees, et le binaire ne
porte aucune etiquette pour elles** : `search_strings` sur `mobility` / `Mobility` /
`MobilityAction` rend 10 chaines, aucune d etiquette de valeur ; `CharacterPhysicsMode*` n en a
que 4 et designe un AUTRE champ (le predicat `SpartanAbilityIsClambering` passe par
`FUN_142c66808` -> `FUN_1406b8244(datum) == 2`, qui lit le mode dans l objet de physique du
personnage a `obj + 0x2dc + *(u16)(obj + 0x2de)`) ; la recherche d instructions sur `0x1294` rend
34 sites sur tout le binaire et AUCUN dans la chaine de mobilite (17 sont des
`MOV word ptr [RDI + 0x1294], BP` de la famille `141c*`, collision d offset annoncee au piege 6
de la passation) — le code vivant adresse le champ par le POINTEUR DE BLOC (`+0x9c`).

Les quatre slots de la vtable du message hors serialiseurs (`142ef58f8`, `142f0203c`,
`142f05308`, `142c46770`) ne sont pas des fonctions DEFINIES dans le projet Ghidra courant.

**Consequence tenue** : `stances[].kind` `mobility` ne devient pas `clamber`, et le lot ne monte
pas de schema. Le maillon restant est l APPLIQUEUR du message sur le personnage.

## 7. Les corps partiels du bipede (lot 5.13.3)

### 7.1 `i57` est porte en entier — l « octet d etat runtime » n en est pas un

`FUN_142f262d4` (branche `tag == 3` d `i57`) appelle `FUN_140f03dfc(dst)` en PREMIERE
instruction — `142f262f2 MOV RDI, RCX` puis `142f262f5 CALL 140f03dfc`, RCX vaut encore `dst` —
et cet initialiseur ecrit `*(undefined2 *)(param_1 + 2) = 0`. La porte `(dst[2] & 1) == 0` est
donc TOUJOURS ouverte quand elle est testee, et la branche gardee par `dst[2] & 0x10` est
INATTEIGNABLE. Le corps :

```
FUN_140f03dfc(dst)                      0 bit — et il met dst[2] a zero
a = R(1)                 -> dst[0]
si a != 0 : R(6)         (FUN_14297ea84, largeur lue sur `if (0x40 - iVar1 < 6)`)
t = R(1)                 -> dst[1]
si t != 0 : FUN_14076e494(br, dst+0x18, 0x10, 0, param_3, 0)   = la queue handle d i60
```

Meme lecon qu `i54` (`bloc[0x9d]` y est `flag1`, lu deux lignes plus haut). Mesure, records
RENDUS : `i57` non portees 1 -> 0 sur 651 declarations (`bfecd02b`) et 44 -> 0 sur 2 531
(`4f77afc1`) ; desyncs `ti=35` 5 -> 4 ; records `ti=35` et etalon `i21` inchanges. Le golden de
la mini-bobine `killsource` deplace UNE ligne de kill de `scan` a `marche` (marche 6 -> 7,
scan 3 -> 2), meme verdict, `DESACCORD` toujours 0.

### 7.2 `i59` : le dispatcheur a six etiquettes, et la seule largeur qui reste

`FUN_142f25e90` : `FUN_142f21c0c` lit `R(3)` et range `brut + 1` ; prefixe commun
`FUN_142f26e40` = `FUN_1408f0ac4(cat 1)` puis `FUN_142f04664(..., *(int*)(a+4) != -1, ...)`,
puis `FUN_14297ea84` = `R(6)` ; puis, par etiquette :

| etiquette | corps |
|---|---|
| 1 | `FUN_1407f08bc` |
| 2 | `FUN_1408f0ac4(cat 5)` + `FUN_1407f08bc` |
| 3 | `FUN_1408f0ac4(cat 0)` + `(cat 5)` + 3 x `FUN_142f26e9c` + `R(24)` + `R(9)` |
| 4 et 5 | `FUN_1408f0ac4(cat 5)` + 1 x `FUN_142f26e9c` + `FUN_14076e494(0x10)` + `R(24)` + `R(9)` |
| 6 | `FUN_1407f08bc` ; si SA porte valait 0 -> `(cat 5)` ; `(cat 0)` + 2 x `FUN_142f26e9c` + `R(1)` + `R(24)` |
| > 6 | zero bit |

`FUN_142f04664(dst, br, flag, p4)` : `flag == 0` -> `FUN_14076e494(br, dst, 0x10, 0, p4, 0)` (la
position absolue aux largeurs de la carte) ; sinon `R(2)` + `FUN_140c1e924` + `R(1) [+ R(16)]`.

**Cela CONFIRME le port mesure du grappin** : les « trois bits de drapeaux a 000 » sont la porte
du handle `cat 1`, celle de pleine precision et celle du bloc quantifie ; le `R(7)` plus `gate8`
sont `R(6)` plus les portes des `FUN_1408f0ac4` de l etiquette, et les deux formes observees se
recollent au bit (16 bits pour l etiquette 2, 8 pour l etiquette 3).

Largeurs fermees par ce lot : `FUN_1407f08bc` = `R(1)` ; si 1 -> `R(8)` (`FUN_1407f08f8`) ·
`FUN_14297ea84` = `R(6)` · `FUN_1408f0ac4` categories 0 et 5 (table `varwidth`, lot 1.9.1 bis).
**Il reste UNE largeur : `FUN_140c1e924`**, dont les trois champs tirent leur `w` d une table
indexee par un octet (`&DAT_143b8c6f0 + param_3 * 0x18`) que le desassemblage ne resout pas au
site d appel. Porter les etiquettes 1, 4, 5, 6 exige de reecrire le prefixe commun, et le
reecrire casserait la fermeture PROUVEE des etiquettes 2 et 3 (`TestI59AnchorWalkProof`, ecarts
0) pour 4 records sur 654 : NON TRAITE, et la raison est celle-la.

### 7.3 `i60` : complet en mesure, statut `partiel` par decision

0 record non porte sur 29 declarations (`bfecd02b`) et sur 16 (`4f77afc1`). La grammaire est
resolue (R7-b) et la source des largeurs d axe l est (5.3.3-a, `grammaireSousCarte`). La decision
que la passation demandait est prise : `SimStateComplet` reste FAUX sans carte, parce que le
critere ecrit est que le chemin absolu d `i0` tire ses trois largeurs de la CARTE du match. Le
dispatcheur rend donc `br.p.Grammaire.SimStateComplet`, et le ratchet G1 — qui DERIVE le statut
de la table du code — impose `partiel`. Ce qui est corrige, c est la note de la table, qui
decrivait un blocage disparu.
