# NOTE 5.19 — LE RECORD AVANT LE REJET : LA DIFFERENTIELLE, PUIS LES ECRIVAINS

Lot 5.19, branche `feat/decfilm-69`, base `d976be4e0` (lots 5.1 a 5.18, grammar-2026-09-22.7,
schema 67). Les lots 5.15 a 5.18 ont instruit le residu de `bfecd02b` sans le reduire, chacun en
refutant le suspect du precedent. Ce lot change de methode : une DIFFERENTIELLE d abord, qui
LOCALISE sans conclure, puis l ecrivain des composants qu elle nomme.

---

## 1. L INSTRUMENT

`mouvement_5_19_differentielle_research_test.go` (collecte) et
`mouvement_5_19_tableaux_research_test.go` (publication, deplacement pur sous le seuil de
500 lignes), tag `research`, test `TestDiff519`. Une passe, un decodage par paquet (les deux
instruments du 5.15 en faisaient deux et muaient le monde deux fois).

```
MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=snowbound \
  go test -tags=research -count=1 -v -timeout 60m -run '^TestDiff519$' \
  ./internal/games/halo_infinite/film/internal/grammar/
```

**Le gate du 5.16/5.18 est reproduit AU PAQUET avant toute lecture**, donc l instrument est bien
celui qui a mesure le residu :

| | `dad793c7` | `bfecd02b` |
|---|---:|---:|
| paquets delta | 5 365 | 30 387 |
| non localises | 5 | 845 |
| FERMES a reste NUL | **5 354** | **2 884** |
| REJET de slot inconnu (trois rangs portes) | **1** | **23 325** |
| debordements | 2 | 32 |
| autres restes (terminateur / desynchronisation / rejet partiel) | 8 | 4 146 |

(Les 23 325 sont les rejets dont les TROIS rangs sont portes ; le 5.15 comptait 23 452 rejets
sans cette condition, et 282 de plus sortent ici en « reste hors bourrage · rejet ».)

---

## 2. CE QUE LA DIFFERENTIELLE MONTRE — ET SES DEUX CONFONDANTS

### 2.1 Le masque du dernier record ne distingue PAS les deux populations

Les masques les plus frequents du dernier record avant le rejet sont EXACTEMENT ceux des paquets
qui ferment : `ti=35 masque 0x2200003` est le premier des deux cotes (8 007 fautifs, 155 fermes),
`0x2000003` le troisieme des deux cotes, `ti=40 masque 0x200000f` le quatrieme chez l un et le
cinquieme chez l autre. **Un paquet ne faute pas parce qu il porte une forme de record que les
autres n ont pas.**

### 2.2 Les largeurs lues par composant sont les MEMES des deux cotes

`i25 unit-command-tick` 10 bits (20 997 fautifs / 397 fermes), `i0 object-position-dynamic-
precision` 54 bits (20 458 / 395), `i1 object-translational-velocity-dynamic-precision` 31 bits
(19 138 / 329), `i21 unit-desired-aiming-vector` 25 bits (13 488 / 216), `i5 object-shield-
vitality` 29 bits (5 807 / 22). Les seules largeurs « que seuls les fautifs portent » sont des
classes a 1 a 6 occurrences — la queue de bruit d un curseur deja faux, pas un signal.

**`i25` est re-confirme comme confondant** (5.15.1 (i)) : premier par volume des deux cotes, MEME
largeur des deux cotes.

### 2.3 Premier confondant : la DENSITE

La comparaison « paquets fautifs contre paquets fermes » est biaisee parce que les paquets qui
ferment sont d abord des paquets PAUVRES : `ti=35` est present dans 96,6 % des fautifs contre
30,4 % des fermes, et 61,1 % des paquets fermes n ont pour dernier record qu un `ti=4
high-frequency` d un seul composant. Une presence « 1 339 fois chez les fautifs, 0 fois chez les
fermes » (`i32 weapon-state-overheated`) se lit d abord comme « ce composant vit dans les paquets
denses ».

### 2.4 Second confondant : la DOSE — et elle refute la loi geometrique

| deltas de bipede dans le paquet | fermes | fautifs | taux de fermeture |
|---:|---:|---:|---:|
| 0 | 2 007 | 786 | **71,9 %** |
| 1 | 3 | 52 | 5,5 % |
| 2 | 0 | 950 | 0,0 % |
| 3 | 4 | 3 227 | 0,1 % |
| 4 | 53 | 4 566 | 1,1 % |
| 5 | 303 | 7 417 | 3,9 % |
| 6 | 231 | 4 629 | 4,8 % |
| 7 | 208 | 1 225 | 14,5 % |
| 8 | 75 | 473 | 13,7 % |

Un paquet SANS delta de bipede ferme 7 fois sur 10 ; des qu il en porte un, il ne ferme plus. Et
la distribution des fautifs n est PAS geometrique (elle serait decroissante) : c est une cloche
centree sur 5, donc le nombre de bipedes lus avant la faute, tronque par elle.

---

## 3. LA DIFFERENTIELLE INTERNE — CELLE QUI N A AUCUN CONFONDANT

Les deux confondants disparaissent si l on compare, **DANS LE MEME PAQUET FAUTIF**, le DERNIER
record (celui apres lequel le curseur est faux) a TOUS CEUX QUI LE PRECEDENT (lus juste, puisque
le record suivant s est decode derriere eux). Meme film, meme trame, meme paquet, meme carte.

> **taux de faute d une classe = dernier / (dernier + precedents)**

23 325 derniers records contre 114 147 precedents. Voici la table des suspects, ordonnee.

### 3.1 Par TAUX de faute (les classes presque toujours terminales)

| classe | taux | dernier | precedents |
|---|---:|---:|---:|
| `ti=40` masque `0x200000a` (i1,i3,i25) | **100,0 %** | 45 | 0 |
| `ti=40` masque `0x200001f` (i0..i4,i25) | **98,8 %** | 82 | 1 |
| `ti=35` masque `0x40000002200023` (bit **54**) | **92,6 %** | 25 | 2 |
| `ti=40` masque `0x2000007` | **92,0 %** | 69 | 6 |
| `ti=42` masque `0xa` | **90,6 %** | 87 | 9 |
| `ti=40` masque `0x200000e` | **88,9 %** | 48 | 6 |
| `ti=40` masque `0x10` (i4 SEUL) | **87,2 %** | 238 | 35 |
| `ti=0` masque vide (record de type 2) | 86,2 % | 75 | 12 |
| `ti=35` masque `0x40000002200003` (bit **54**) | 83,3 % | 50 | 10 |
| `ti=32` masque `0xf` | 82,6 % | 76 | 16 |
| `ti=42` masque `0xe` | 81,4 % | 70 | 16 |
| `ti= 2` masque `0x1` | 80,3 % | 376 | 92 |
| `ti=40` masque `0x200000f` | **75,0 %** | **1 841** | 615 |
| `ti=37` masque `0xc` / `0xd` | 62,8 / 63,9 % | 98 / 69 | 58 / 39 |
| `ti=10` masque `0x4000000` | 53,6 % | 239 | 207 |
| `ti=42` masque `0xf` | 48,3 % | 143 | 153 |
| `ti=37` masque `0xf` | 43,3 % | 311 | 408 |
| `ti=35` masques usuels (`0x2200003`, `0x2200023`, `0x2000003`, …) | **10,7 a 25,1 %** | 18 856 | 85 889 |
| `ti= 4` masque `0x1` (`high-frequency`) | **1,1 %** | 244 | 22 604 |

### 3.2 Par composant ET par largeur lue

| composant | largeur | taux | dernier | precedents |
|---|---:|---:|---:|---:|
| `ti=40 i1 object-translational-velocity-dynamic-precision` | 2 | **92,7 %** | 191 | 15 |
| `ti=40 i3 object-angular-velocity-dynamic-precision` | 2 | **92,5 %** | 148 | 12 |
| `ti=35 i54 biped-mobility-action-component` | **344** | **90,3 %** | 102 | 11 |
| `ti=40 i2 object-forward-and-up-dynamic-precision` | 31 | **89,7 %** | 504 | 58 |
| `ti=40 i4 object-body-vitality-component` | 11 | **88,9 %** | 367 | 46 |
| `ti=32 i0 tacmap-areaofinterest` | 66 / 67 | 87,0 / 84,1 % | 20 / 90 | 3 / 17 |
| `ti= 2 i0 game-engine-team-mapping-component` | 58 / 62 | 85,7 / 77,8 % | 126 / 242 | 21 / 69 |
| `ti=42 i1 object-translational-velocity-component` | 1 | 78,6 % | 198 | 54 |
| `ti=40 i25 unit-command-tick-component` | 10 | **77,0 %** | **2 141** | 639 |
| `ti=40 i0 object-position-dynamic-precision` | 54 | **76,2 %** | **2 016** | 629 |
| `ti=40 i1 object-translational-velocity-dynamic-precision` | 31 | 75,7 % | 1 939 | 624 |
| `ti=40 i3 object-angular-velocity-dynamic-precision` | 29 | 75,6 % | 1 923 | 621 |
| `ti=40 i2 object-forward-and-up-dynamic-precision` | 64 | 73,0 % | 1 561 | 576 |
| `ti=42 / ti=37 object-*` (sans dyn.-prec.) | 28 / 30 / 52 | 41 a 57 % | ~2 400 | ~2 900 |
| `ti=10 i26 managed-object-rtpc-component` | 54 | 53,6 % | 239 | 207 |
| `ti=35 i1 object-translational-velocity-dynamic-precision` | 31 | 20,7 % | 17 199 | 66 018 |
| `ti=35 i0 object-position-dynamic-precision` | 54 | 20,0 % | 18 442 | 73 839 |
| `ti=35 i21 unit-desired-aiming-vector` | 25 | 19,1 % | 13 488 | 56 956 |
| `ti=35 i25 unit-command-tick` | 10 | 18,0 % | 18 856 | 85 889 |
| `ti=35 i5 object-shield-vitality` | 29 | 13,9 % | 5 807 | 35 992 |
| `ti= 4 i0 high-frequency` | 8 | **1,1 %** | 244 | 22 604 |

### 3.3 LES TROIS FAITS QUE CETTE TABLE ETABLIT

1. **Le taux de faute d un record de `ti=40` est de 73 a 89 % POUR CHACUN de ses composants,
   masque par masque.** Un record de vehicule est terminal trois fois sur quatre. Le taux est
   UNIFORME sur les six composants : ce n est donc pas « un composant de plus a porter », c est
   le record entier.
2. **Le meme composant, a la meme largeur, faute differemment selon l ARCHETYPE.**
   `object-body-vitality-component`, 11 bits : 88,9 % sur `ti=40`, 21,3 % sur `ti=35`. Le port
   lit les deux par le MEME deserialiseur et la MEME largeur (`FUN_140fb8978`). L archetype
   change donc la verite de la lecture sans changer ce que le port lit.
3. **`ti=4 high-frequency` a un taux de 1,1 % sur 22 848 lectures** : c est le seul temoin propre
   du film. Tout ce qui est au-dessus de 10 % est une lecture a instruire.

### 3.4 ET `dad793c7` NE PORTE AUCUNE DES CLASSES FAUTIVES

Sur le film a un joueur, les derniers records sont `ti=4` (97,8 %), `ti=35` (1,4 %), `ti=47`
(0,8 %), `ti=0` et `ti=5` (une fois chacun). **`ti=40`, `ti=2`, `ti=32`, `ti=42`, `ti=37` et
`ti=10` n y apparaissent jamais** — un seul rejet sur 5 365 paquets, sans record lu. Les
archetypes que la differentielle accuse sont exactement ceux que le film de calibration
n exerce pas.

---

## 4. LES ECRIVAINS LUS — QUATRE SUSPECTS, QUATRE CONFORMITES, ET UN REPORT REFERME

Ordre pris dans la table de la section 3 : la classe au plus fort taux dont TOUT le corps se lit
par des feuilles adressables.

### 4.1 `ti=40 i2 object-forward-and-up-dynamic-precision-component` — CONFORME, et D3 (5.14) EST RESOLU

C est le SEUL composant `partiel` du record de vehicule (`FUN_140c5f7ec`, niveau 2 au registre),
et la differentielle mesure ses deux formes a 31 bits (89,7 % de faute) et 64 bits (73,0 %).
64 = trois bits de porte + 61, donc le chemin « config » `FUN_142e29bac`, que le port lit
`R(1) ; si 0 -> R(30) ; puis R(30)`.

Le second champ de ce chemin est `FUN_1406d84b4`, **la fonction dont D3 (5.14) disait que la
largeur passe PAR LA PILE (`in_stack_00000028`) et que le desassemblage ne la resout pas a ses
sites d appel nus**. Elle est resolue ICI, au site d appel :

```
142e29cc4: MOV byte  ptr [RSP + 0x30],0x0     ; 7e argument
142e29cc9: MOV byte  ptr [RSP + 0x28],0x0     ; 6e argument
142e29cce: MOV dword ptr [RSP + 0x20],0x1e    ; 5e argument = 0x1e = 30
142e29cd6: CALL 0x1406d84b4
```

et `FUN_1406d84b4` est un lecteur de bits PLAT : `*(reader+0x2c) += in_stack_00000028`, aucune
porte, aucune branche. Le second champ vaut donc **R(30) inconditionnel**, exactement ce que le
port consomme. **Le port est conforme, et une largeur qui etait ASSUMEE est desormais PROUVEE.**

### 4.2 `ti=40 i4 object-body-vitality-component` — CONFORME AU BIT

`FUN_140fb8978` :

```
FUN_1406d84b4(reader, reader, DAT_143cd84ec, DAT_143cd8374, 8, 1, 1)   R(8)
FUN_1406cf008(reader) x 3                                              3 x R(1)
```

**11 bits, sans porte ni branche** — exactement ce que le port consomme. Or la classe
`ti=40 masque 0x10` (ce composant SEUL) faute a **87,2 %** : la faute n est donc PAS dans le
corps de ce record.

### 4.3 LA STRUCTURE DU RECORD DE LA BRANCHE VIVE — CONFORME

`FUN_1406cd128` (branche `DAT_14474cd78 != 0`) lit, par record :
`[R(1) -> DELTA, sinon R(2) type]` puis `[FUN_1406d310c(filigrane) bits + base]` puis `[R(2) tag]`,
puis `FUN_1406cbaa0(type, id, ...)`. Le selecteur base/largeur y est `DAT_144706104` (et non le bit
de configuration) — les deux formes coincidant deja (D4 du 5.15), rien ne change. Le corps DELTA de
`FUN_1406cbaa0` est `FUN_1406cdc04` (selecteur de baseline) puis `FUN_1406caad8` vers
`FUN_14076cb60`, ce que le port porte. Rien entre deux records.

### 4.4 `FUN_1408f1aa4` — LE LECTEUR DE CORPS DE `NEW` DE LA BRANCHE VIVE — CONFORME

Le 5.15.2 (a) avait note que la branche vive lit un `NEW` par `FUN_1408f1aa4` et non par
`FUN_141f86704` (branche 0), sans le porter. Lecture faite :

```
R(6)                                        l archetype -> descripteur *(param_1+0x18 + 8 + ti*8)
vtable[0x60](taille, tampon, reader, 1)     le DEFAULT-STATE
vtable[0x88](...) et vtable[0x30]()         aucun bit (pas d argument lecteur)
si (bitmap derive != 0 ou porte deja lue) :
    R(1)                                    la PORTE
    si posee : FUN_14076cb60                masque + boucle de composants
```

C est EXACTEMENT la structure de `TraverseEntity` (R(6), default-state par archetype, `t.Gate`
`R(1)`, `consumeMask`, boucle) — dont le commentaire cite deja `FUN_1408f1aa4`. **Le port est du
bon cote de la branche.** Seule nuance : chez l ecrivain la porte est lue SOUS un bitmap derive
(`uVar17`, calcule sans lire un bit) ; le port la lit sans condition, et son commentaire dit que
la retirer desynchronise — le bitmap est donc non vide en pratique.

### 4.5 LA REGLE D ARRET DU BRIEF EST ATTEINTE

Quatre suspects lus chez l ecrivain, quatre conformites, **aucune baisse des rejets**. Le brief
prescrit alors de rendre la differentielle — et la differentielle, elle, a trouve la cause
ailleurs que dans une largeur.

---

## 5. LA CAUSE, NOMMEE PAR LA MESURE : LE SLOT REJETE EST UNE ENTITE QUE PERSONNE N A DECLAREE

Trois mesures, dans cet ordre, et la conclusion du 5.16.2 en est RENVERSEE.

### 5.1 Le slot rejete, PONDERE PAR SON VOLUME, est un slot de bipede que DEUX sources declarent

`TestRejets519` : 567 slots distincts pour 23 325 rejets. Les **vingt-sept premiers slots — 72 %
du volume — sont dans la bande 521-601, et le balayeur d ancres ET la table de datums les
declarent tous `ti=35`.**

| slot | rejets | balayeur | datums |
|---:|---:|---|---|
| 543 | 1 014 | `ti=35` | `ti=35` |
| 539 | 974 | `ti=35` | `ti=35` |
| 556 | 972 | `ti=35` | `ti=35` |
| 552 | 945 | `ti=35` | `ti=35` |
| 597 | 894 | `ti=35` | `ti=35` |

**Le 5.16.2 avait mesure les slots DISTINCTS** (632, etendue quasi uniforme sur les treize bits,
mediane 3 307) et en avait conclu « ce ne sont pas des slots, ce sont des lectures a une position
FAUSSE ». Pondere par le volume, c est l inverse : **ce sont des slots, ce sont des bipedes, et
ils ont un archetype declare.** Les 500 slots epars de la queue sont le bruit ; les vingt-sept
premiers sont le residu.

### 5.2 A l instant du rejet, le slot n a JAMAIS ete lie — et aucun `DEL` n y est pour rien

`TestDelies519`, 290 records `DEL` lus sur tout le film :

| etat du slot au moment du rejet | rejets | part |
|---|---:|---:|
| **jamais lie par aucune source avant ce paquet** | **23 092** | **99,0 %** |
| delie par un record de type 2 (`DEL`) | 211 | 0,9 % |
| lie un jour par une image-cle, plus lie | 22 | 0,1 % |

Le faux `DEL` du 5.14.3 est donc DEFINITIVEMENT ecarte comme cause du residu.

### 5.3 Le slot n est meme pas un candidat d ancre du chunk, et la croissance n y est pour rien

`TestEcartes519` confronte le slot rejete aux candidats d ancre que `candidatsDeDatum` trouve a
position LIBRE dans le payload d image-cle du MEME chunk :

| | rejets | part |
|---|---:|---:|
| dans les candidats RETENUS par la croissance | 0 | 0,0 % |
| dans les candidats ECARTES par la croissance | 236 | 1,0 % |
| **dans AUCUN candidat du payload** | **23 089** | **99,0 %** |

`plusLongueSuiteCroissante` (5.16.4) n est donc pas la cause : le slot n est pas dans la table
d image-cle du chunk, pas meme parmi ses candidats ecartes.

### 5.4 LA COUVERTURE DE L IMAGE-CLE SUR UN FILM DENSE — D2 (5.16) CHIFFRE

`TestCouverture519`, les 27 chunks de `bfecd02b` : chaque payload d image-cle pese **1,32 a
1,37 million de bits**, et le balayeur d ancres s arrete entre **45,7 % et 55,8 %** du payload,
sur 424 a 483 ancres — pour **1 374 a 1 540 candidats ECARTES** par la contrainte de croissance.

| chunk | bits d image-cle | ancres | dernier bit | part du payload | slot max | datums (ambigus) |
|---:|---:|---:|---:|---:|---:|---|
| 1 | 1 318 136 | 424 | 602 695 | 45,7 % | 1 580 | 424 (1 374) |
| 2 | 1 350 712 | 459 | 644 773 | 47,7 % | 1 663 | 460 (1 523) |
| 8 | 1 360 624 | 471 | 676 069 | 49,7 % | 1 861 | 472 (1 532) |
| 20 | 1 362 864 | 472 | 729 073 | 53,5 % | 2 332 | 473 (1 525) |
| 27 | 1 343 112 | 454 | 742 968 | 55,3 % | 2 644 | 456 (1 495) |

**La moitie de chaque table d image-cle n est jamais lue.** C est D2 (5.16) — « la chaine de
l image-cle se coupe, la fenetre de 120 000 bits en est la cause » — porte du film a un joueur au
film dense, et chiffre.

### 5.5 ET LE PREMIER SLOT REJETE D UN CHUNK EST LE SLOT MAX DE SON IMAGE-CLE, PLUS UN

`TestPremierRejet519` : le chunk decode PROPREMENT jusqu a sa premiere faute, puis s effondre.

| chunk | premier paquet fautif | sur n deltas | slot rejete | slot max de l image-cle | `NEW` lus dans TOUT le chunk | paquets non fautifs |
|---:|---:|---:|---:|---:|---:|---:|
| 1 | 700 | 1 146 | 512 | 1 580 | 20 | 1 049 |
| 2 | 1 144 | 1 196 | **1 673** | **1 663** | **0** | 1 144 |
| 3 | 32 | 1 094 | **1 674** | **1 673** | 72 | 609 |
| 8 | 62 | 1 172 | **1 862** | **1 861** | 6 | 142 |
| 9 | 51 | 1 173 | **1 911** | **1 910** | 6 | 94 |
| 11 | 39 | 1 181 | **1 995** | **1 994** | 7 | 85 |
| 13 | 80 | 1 172 | **2 064** | **2 063** | 3 | 215 |
| 17 | 64 | 1 180 | **2 205** | 2 203 | 2 | 183 |
| 18 | 79 | 1 163 | **2 233** | **2 232** | 2 | 305 |
| 20 | 12 | 1 174 | **2 333** | **2 332** | 2 | 58 |
| 21 | 50 | 1 171 | **2 388** | **2 387** | 4 | 103 |
| 23 | 110 | 1 180 | **2 462** | **2 461** | 23 | 279 |
| 26 | 80 | 1 174 | **2 589** | **2 588** | 6 | 215 |
| 27 | — | 106 | — | 2 644 | 0 | **106 / 106** |

Dans TREIZE chunks sur 26, **le premier slot rejete est exactement `slot max de l image-cle + 1`**
(deux fois +2 ou +3). C est la signature de l ALLOCATEUR : la premiere entite creee apres
l instantane recoit le slot suivant.

**Et le chunk 2 lit ZERO record `NEW` sur 1 196 paquets delta, alors qu au moins dix entites y
naissent** (slot max 1 663 au chunk 2, 1 673 au chunk 3).

### 5.6 LA CAUSE, EN UNE PHRASE

> **Le residu de `bfecd02b` n est pas une largeur de composant : c est que le monde hors ligne
> n apprend JAMAIS la naissance d une entite entre deux images-cles. La premiere entite creee
> apres l image-cle d un chunk (slot = slot max + 1) est referencee par un delta que la garde
> rejette, le rejet emporte la queue du paquet — donc les `NEW` qui y vivaient — et le chunk
> s effondre a partir de la.**

Le 5.15.1 (f) (« ce n est pas une cascade ») est REFUTE par `TestPremierRejet519` : le chunk 1
ferme 700 paquets d affilee avant sa premiere faute, le chunk 2 en ferme 1 144, et le chunk 27
ferme ses 106 paquets. Le 5.15.1 (g) l avait devine sans le chiffrer (« 685 NEW pour
160 739 DELTA disent qu elle n en lit pas assez ») ; ce lot le date, le localise et le compte.

Ce qui reste a trouver est donc UNE chose, et elle a deux candidats mesurables, tous deux hors du
perimetre de ce lot :

1. **la marche d ancres d image-cle** (D2 du 5.16) : elle s arrete a la moitie du payload. La
   table complete donnerait l archetype de tous les slots du chunk — dont ceux nes apres
   l instantane du chunk PRECEDENT ;
2. **le record `NEW` lui-meme** : il faut savoir OU le jeu annonce la naissance d une entite
   entre deux images-cles. L en-tete rejete lit `prefixe 1` = DELTA (constant sur les six temoins
   dumpes : slot 1792, tag 1), donc ce n est pas un `NEW` mal cadre a cette position.
