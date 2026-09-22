# Note 5.18 — Le controle de corruption par composant est UN BIT DU FILM, et il vaut zero (2026-09-22)

> Lot 5.18, branche `feat/decfilm-68`, base `5a5fbfaf5` (integration des lots 5.1 a 5.17,
> `grammar-2026-09-22.6`, schema 67). Cette note est le DOUBLE de la section du plan
> (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « Post-chantier — lot 5.18 ») : le plan fait foi sur
> l avancement, cette note sur la GRAMMAIRE.
>
> Elle referme D1 (5.17) et D2 (5.17), les deux decouvertes que le lot precedent avait laissees
> ouvertes sur le singleton du film.

## 1. Le maillon, de l ecrivain jusqu au lecteur de bits

Toutes les adresses sont LUES (Ghidra, `HaloInfinite.exe`, base `0x140000000`, lecture seule).

| maillon | adresse | ce qu il fait |
|---|---|---|
| l ECRIVAIN de `chunk_00` | `FUN_14299b198` @14299b25b | `FUN_1406d49c4(writer, byte[film+0xCB45C])` = **W(1)** |
| le LECTEUR de `chunk_00` | `FUN_14299ab50` @14299ac28 | `FUN_1406cf008(lecteur)` = **R(1)** -> `film+0xCB45C` |
| le REPORT au singleton | `FUN_1428e219c` @1428e2239 | `*(char *)(singleton + 0x1AE) = (char)film[0x32d17]` (= l octet `0xCB45C`), sous la garde `*film == 0x29` |
| la LECTURE en rejeu | `FUN_14076cea8` @14076ceba | rend `DAT_144c23326` (= `DAT_144c23178 + 0x1AE`) en rejeu de film, `DAT_1450e24e8` en jeu vif |
| l USAGE | `FUN_14076cb60` | `extra` : apres CHAQUE composant present, `R(1)` de garde ; si ce bit vaut 1, `R(32)` sentinelle `0x0bcddcba` (« entity component corrupt: entity id:[0x%x] type:[%s] component:[%s] ») |
| l USAGE, etat complet | `FUN_142e2c690` | meme `extra`, meme `R(1)` + `R(32)` |

### 1.1 Le pseudo-code du lecteur d en-tete, largeur par largeur

`FUN_14299ab50(film, lecteur)`, toutes les largeurs prises en `R9D` au desassemblage :

```
R(0x20)                        -> film+0x00000     ; FilmMajorVersion
R(0x20)                        -> film+0x00004     ; LA VERSION DE FORMAT
R(FUN_141cfff30(film+4))       -> film+0x00008     ; LE REGISTRE   (0x659000 bits = 50 blocs)
R(FUN_141cffe20(film+4))       -> film+0xCB208     ; LA TABLE PAR TYPE (0xF60 bits = 123 u32)
R(0x100)                       -> film+0xCB3F4     ; version en clair
R(0x100)                       -> film+0xCB414     ; build en clair
R(0x100)                       -> film+0xCB434     ; saveur en clair
R(0x20)                        -> film+0xCB454     ; identifiant de build
R(0x20)                        -> film+0xCB458     ; changelist
R(1)                           -> film+0xCB45C     ; <<<< LE CONTROLE DE CORRUPTION
FUN_14299bcb0(film, film+4)
```

et son appelant `FUN_14299ac50` enchaine onze champs de plus puis `FUN_1407ee138(lecteur,
film+0xCE690, film+4)` — LA TABLE DES 32 JOUEURS, que le depot lit depuis le lot 1.5.2. Le
`chunk_00` que ce lecteur consomme est donc exactement celui que `player_table.go` et
`film_identity.go` lisent : la structure `film` fait `0x1E1D80` octets, la taille meme de
`chunk_00.bin` sur les deux temoins.

L ECRIVAIN MIROITE LE LECTEUR CHAMP POUR CHAMP (`FUN_14299b198`, litteraux `0x20`, `0x20`,
`0x659000`, `0xF60`, `0x100` x3, `0x20`, `0x20`, puis `FUN_1406d49c4` sur l octet `0xCB45C`).

### 1.2 Pourquoi l offset de STRUCTURE est l offset d OCTET du flux

Tout ce qui precede le drapeau est aligne sur l octet, et la chaine ferme sans un bit de reste :

```
0x00008 + 0x659000/8 = 0x00008 + 0xCB200 = 0xCB208      (fin du registre)
0xCB208 + 0xF60/8    = 0xCB208 + 0x001EC = 0xCB3F4      (fin de la table par type)
0xCB3F4 + 0x20 + 0x20 + 0x20                 = 0xCB454  (trois chaines de 32 octets)
0xCB454 + 4 + 4                              = 0xCB45C  (buildID, changelist)
```

Le drapeau est donc le **bit de poids fort de l octet `buildOff + identBoolOff`** dans le
`chunk_00` inflate — et c est LE MEME BIT que le depot enjambe depuis le lot 1.5.1 sous le nom
de « booleen d un bit », celui dont `identDecalageBit` decale tout ce qui suit (les deux champs
de nom, l horodatage, le corps). Le lot ne decouvre pas une position : il NOMME et LIT un bit
que le depot franchissait sans le regarder.

### 1.3 Le drapeau n a qu un seul ecrivain, et deux remises a zero

| site | ce qu il fait |
|---|---|
| `FUN_140eff23c` | CONSTRUCTEUR du singleton : `_DAT_144c23320 = 0` couvre `0x144c23326` |
| `FUN_140eff3a8` | RE-ARMEMENT a chaque chargement (`CommonSingleton.h`) : `*(u16 *)(this + 0x1ae) = 0` |
| `FUN_1428e219c` | LE SEUL ECRIVAIN : recopie l octet du film |
| `FUN_14076cea8` | LE SEUL LECTEUR |

Un seul site d instruction sur les 13 607 754 du binaire ecrit `[x + 0x1ae]` dans ce chemin
(`8887ae010000` @1428e2239). Le vtable `0x145620160` qui porte `FUN_1428e219c` est celui de
`FUN_1428e1c0c` (`0x1456200e8`, l accesseur de version de format sur `&DAT_144c23178`) : le
`this` de l ecrivain EST le singleton du film.

**LA GARDE `*param_2 == 0x29`** : `param_2+0` est la `FilmMajorVersion` du film, et `0x29` = 41
— la majeure dominante du cache (1 123 films sur 1 351 au releve du 2026-09-15). Un film d une
autre majeure ne traverse pas cette branche, et le champ du singleton garde son zero.

## 2. La valeur, mesuree par le lecteur de production

`TestDrapeau518Temoin` / `TestDrapeau518Corpus` (`//go:build research`, paquet `grammar`).

| film | build / format | buildOff | octet `0xCB45C` | drapeau |
|---|---|---:|---|---|
| `dad793c7` | `HI_1_13_0` / 27 | `0xCB414` | `00000000` | **faux** |
| `bfecd02b` | `HI_1_13_0` / 27 | `0xCB414` | `00000000` | **faux** |

Sur tout le cache : **0 drapeau leve sur 1 605 films lus** (5 sans section d identification),
8 builds, 5 formats — `HI_1_13_0`/27 1349, `HI_1_12_0`/27 147, `HI_1_11_0`/25 57,
`HI_1_10_0`/24 34, `HI_1_8_0`/24 13, `HI_1_9_0`/24 3, `HI_1_4_1`/21 1, `HI_1_5_1`/23 1.

**LE DEFAUT FAUX DU DEPOT ETAIT DONC JUSTE — PAR HASARD.** `GrammaireBalayage.ControleDeCorruption`
etait une bascule d instrument : rien ne lisait le film, et un film qui aurait leve ce bit
aurait desynchronise sans un mot. D1 (5.17) designait ce drapeau comme le suspect du residu de
`bfecd02b` ; la mesure l ECARTE, et elle l ecarte par une LECTURE (le bit vaut zero), pas par une
absence.

## 3. Le port : le drapeau vient du film, et rien ne peut le lui reprendre

| piece | role |
|---|---|
| `profile.FilmIdentity.ControleDeCorruption` | le champ, rempli par `lireControleDeCorruption` (`film_identity.go`) |
| `profile.Profile.IdentityRead()` | distingue « le film declare faux » de « le film ne declare rien » |
| `grammaireSousFilm(g, p)` (`profil_balayage.go`) | LA REGLE, ecrite une fois ; second rendu = le film a parle |
| `FilmContext.ProfilDeBalayage()` | DERIVE le drapeau a chaque rendu (`controle_corruption_du_film.go`) |
| `GrammaireSousFilm(bal, film)` | la porte de `killsource`, qui part de l invariant sans contexte |
| `FilmContext.ControleDeCorruptionRepli()` | le COMPTE du repli, un film un verdict |

**POURQUOI LE DRAPEAU NE VIT PAS DANS `FilmContext.bal`.** `PoserProfilDeBalayage` remplace le
profil ENTIER, et `replay.poserProfilPuisCarte` y installe le profil calibre par `killsource` —
ou l INVARIANT quand le kill-feed n a pas pu se decoder
(`replaybuild.profilDeBalayageDeLaCuisson`). Range dans `bal`, le drapeau du film serait efface
par ce geste, sans un mot. Il est DERIVE. Le ratchet `controle_corruption_du_film_test.go` le
fige : un profil etranger qui affirme le CONTRAIRE du film ne change ni `ProfilDeBalayage`, ni
`CadreDeBalayage`, ni `ContexteDeLecture`.

**LE REPLI EST NOMME** : `repli_controle_corruption_section_absente` (registre `filmdec`,
`apres_lecture`, `section_absente`) — les 5 films du cache sans section d identification
(`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`, format 20) ne declarent pas ce bit,
la grammaire garde son invariant, et `killsource` l AVERTIT par film
(`avertirReplisDeCalibration`). Ces cinq-la sont deja mis de cote par `profile.ErrUnknownBuild`.

## 4. D2 (5.17) est ferme aussi, et ce n est PAS un drapeau du film

`DAT_144c232e1` (= `DAT_144c23178 + 0x169`) a **deux ecrivains, et les deux sont dans la meme
fonction** :

```
FUN_1428e24bc(singleton, session) :        ; l aller-retour serialise (D6 du 5.16)
  ...
  *(u8 *)(singleton + 0x169) = 1           ; @1428e251c   <- LEVE
  ... alloue 0x1b000 o, vtable[0x10], FUN_142f2c658 par entree, writer -> reader,
      DAT_14474cd78 abaisse, vtable[0x40], vtable[0x48] par entree, restauration ...
  *(u8 *)(singleton + 0x169) = 0           ; @1428e2714   <- RABAISSE
```

C est une PORTEE de re-entrance autour de l aller-retour interne, pas une valeur chargee depuis
le film. Comme les deux remises a zero du singleton (`FUN_140eff23c` : `_DAT_144c232e0 = 0` ;
`FUN_140eff3a8` : `*(u32 *)(this + 0x168) = 0`) le laissent a zero, **il vaut ZERO pendant tout
decodage de paquet** : le filtre de composants et la substitution de niveau de `FUN_14076cb60`
sont donc ACTIFS en rejeu, et le 5.17.2 avait deja etabli que ce filtre n a pas d image hors
ligne (l index du depot est deja celui du registre du film, trois ratchets). **Rien a porter, et
la premisse « il vient lui aussi du film » est REFUTEE par ses ecrivains.**

## 5. Le gate

Carte `snowbound` pour les deux temoins — c est celle qui REPRODUIT le tableau du 5.16.4 a
chaque chiffre. Le controle est fait : `streets` sur `dad793c7` rend 5 285 paquets fermes et
72 debordements, donc la carte n est PAS indifferente au gate et il n a pas ete joue au hasard.

| mesure | `dad793c7` avant (5.16.4) | apres | `bfecd02b` avant (5.16.4) | apres |
|---|---:|---:|---:|---:|
| paquets a reste NUL | 5 354 / 5 365 | **5 354 / 5 365** | 2 884 / 30 387 | **2 884 / 30 387** |
| debordements | 2 | **2** | 32 | **32** |
| records rendus | 5 641 | **5 641** | 176 786 | **176 786** |
| records `ti=35` | 75 | **75** | 129 572 | **129 572** |
| desyncs `ti=35` | 0 | **0** | 4 | **4** |
| rejets hors datum | 2 | **2** | 23 769 | **23 769** |
| rejets de vue (repli) | 0 | **0** | 0 | **0** |
| liaisons de datum | 54 | **54** | 10 | **10** |
| records fantomes | (non publie) | 1 | (non publie) | 31 |

Chiffre pour chiffre. **LE PORT NE DEPLACE RIEN**, et c est le resultat attendu d un drapeau qui
vaut zero partout : la valeur lue EGALE l ancien defaut.

Les records fantomes (trace qui FINIT au-dela du payload) sont publies pour la premiere fois
depuis le 5.14 — le tableau du 5.16.4 ne portait pas la ligne. Ce ne sont pas les « fantomes »
du 5.14 (records rendus par un rang autre que la vue B, 13 -> 0 et 304 -> 0) : ce sont les
records des paquets qui DEBORDENT, et leur compte suit les debordements (2 et 32, inchanges).
Sur `dad793c7` l A/B `MOUV516_DATUMS=0` en rend **1** aussi : le fantome ne vient donc pas de la
table de datums du 5.16.4, il lui est anterieur.

### 5.1 Les deux mesures que le brief demande de rejouer en sortie

`replay-equiv -films bcb6d393` SANS `-update` : **les SIX memes ecarts, aux memes valeurs**
(`killsource`, `grappleReads.stats`, `vehicles`, `movementStates` **1 737** contre 4 469 en
reference, `movementStates.stats`, `artifact` **1 929 397** octets contre 1 912 592). La
reference est perimee depuis la fusion 5.10 (report D1 (5.11)) ; ce lot ne la deplace pas d un
octet. Decodage 15,4 s, pic 0,18 Gio, un film a la fois.

D5 (5.14), re-mesure sur `bfecd02b` (`TestClasses514Contenu` / `TestClasses514Restes`) :

* **les 21 index de controle epars de la vue C sont INCHANGES** — 29 classes, index 0 a 7
  denses (3 986 · 2 884 · 2 695 · 2 200 · 2 308 · 2 374 · 1 962 · 2 034), 8 a 31 a 2-27
  entrees. Ils restent du bruit du residu ;
* **rang 1 vue B desynchronise : 1 111 paquets** (1 752 716 bits de reste). Le plan portait
  1 110, chiffre du 5.14 — l unite d ecart date du **5.16.4** : `m533bLierMonde` appelle
  `LierTableDeDatums` depuis ce lot-la, et la table ajoute 463 records sur ce film. Ce lot-ci ne
  deplace aucun bit, et le gate le montre chiffre pour chiffre.

## 6. Ce que le lot lit en plus

**RIEN, ET LE TABLEAU PAR ARCHETYPE LE DIT.** Le drapeau vaut zero sur les deux temoins et sur
les 1 605 films du cache : aucun `R(1)` de garde, aucun `R(32)` sentinelle ne sont consommes,
donc aucun bit ne se lit autrement qu avant.

`TestGate516Contenu` sur `bfecd02b` : **207 etiquettes de composant**, et les archetypes rendent
exactement les comptes du 5.16.5 — `ti=35` 129 572, `ti=4` 28 531, `ti=40` 5 337, `ti=37` 4 551,
`ti=42` 2 804, `ti=2` 2 063, `ti=10` 1 025, `ti=41` 696, `ti=32` 329, `ti=0` 295, `ti=43` 104,
`ti=38` 62, `ti=47` 56, et 43 autres `ti` a moins de 40 records. **Aucun archetype nouveau, aucune
etiquette nouvelle, AUCUN canal d etat de bipede** — ce qui est la seule reponse possible quand
le drapeau lu egale l ancien defaut.

`grammar.Rev` monte a **`grammar-2026-09-22.7`** : la couche LIT une decision qu elle ignorait,
et c est cela que la revision date, pas un octet deplace. `facts.Rev` **NE MONTE PAS** — sur
chaque film du parc la valeur lue egale l ancien defaut, donc aucune ligne de
`match_kill_events` ne se redecoderait autrement : **AUCUN backlog killsource n est ouvert**.
`profile.Rev` ne monte pas non plus : la couche gagne un champ PORTEUR et son accesseur, pas une
ligne de table, pas une largeur, pas une borne. `replay.SchemaVersion` reste a **67** : aucun
champ publie ne change, donc pas de chronique v68, pas de fixture `replay_schema_68_*`, pas de
jumeaux `replaydoc`/`replayview`, pas de zod, pas d OpenAPI.

## 7. Les instruments

`mouvement_5_18_drapeau_research_test.go`, `//go:build research`, paquet `grammar`. Il n a besoin
QUE de `MOUV511_FILM` (ou de `CHUNK00_CORPUS` pour le balayage) : le drapeau vit dans `chunk_00`
et aucune largeur d axe n entre dans sa lecture.

| test | role |
|---|---|
| `TestDrapeau518Temoin` | le drapeau du film courant, avec les octets qui le portent |
| `TestDrapeau518Corpus` | la ventilation sur tout le cache, par build et par format |

Gate sans decodage : `TestControleDeCorruptionEstLeBitDe0xCB45C` (bit-exact, les huit bits de
l octet balayes) et `controle_corruption_du_film_test.go` (quatre ratchets : le drapeau vient du
film sur les DEUX constructeurs, un profil pose ne l efface pas, le repli se dit, un contexte nul
n affirme rien).

## 8. Le residu de `bfecd02b` : ce qui reste, et par quelle adresse le reprendre

D1 (5.17) est ECARTE — par une lecture, pas par une absence : le bit existe, il est lu, il vaut
zero, et il vaut zero sur les 1 605 films du cache. D2 (5.17) est REFUTE : `+0x169` est une portee
de re-entrance, pas un champ du film. Le residu de `bfecd02b` (2 884 paquets fermes sur 30 387,
23 769 rejets hors datum, 1 111 paquets « rang 1 desynchronise ») n a donc plus de suspect nomme
sur le singleton du film. Ce que le lot laisse ouvert, avec ses adresses :

| piste | adresse | ce qu elle ouvrirait |
|---|---|---|
| **le type 9** (D3 du 5.17) | branche `sVar2 == 9` de `FUN_1428e22c0` ; `*(int *)(*(param_1+0x130) + 0xf8) += *(int *)(paquet + 2)` | **631 561 octets sur `bfecd02b` contre 4 sur `dad793c7`** — le deuxieme plus gros bloc du film dense, que le repartiteur SAUTE en avancant le curseur d octets. Le rapport de taille entre les deux temoins suit exactement le rapport de leurs residus |
| la chaine d image-cle coupee | `WalkKeyframeWorld`, fenetre de 120 000 bits (D2 du 5.16) | les POSITIONS des records au-dela de la coupure ; la table de datums contourne, elle ne repare pas |
| les trois handlers non lus | 6 `FUN_142988084` (`session+0x114`), 0xb `FUN_1429882c8`, 0xc `FUN_1429875e4` | une source d etat que ni la trame ni l image-cle ne portent |
| l historique de baseline | `FUN_1406cdc04` -> `FUN_141fda280` -> `args[3]`/`args[4]` de `FUN_14076cb60` (D3 du 5.16) | des VALEURS justes sur 14 records de 174 606 — pas des largeurs, donc pas le residu |

Le 5.16.2 avait deja etabli que **526 des 632 slots rejetes de `bfecd02b` ne sont declares par
AUCUNE source lue** et couvrent uniformement les treize bits : ce ne sont pas des slots, ce sont
des lectures prises a une position FAUSSE. Le desalignement est donc EN AMONT, dans le corps d un
record — et la seule source de la taille de l ecart entre les deux temoins qui reste non lue est
le type 9.
