# La carte de `chunk_00` : trois sections, dont deux que le depot n'avait jamais regardees

Date : 2026-08-30. Lot D du chantier « visee lunette ». Mesure **hors ligne, lecture seule**, sur
les films du cache. Aucun code de production n'a ete modifie ; aucun outil de retro-ingenierie
binaire n'a ete employe (le lot Ghidra tourne en parallele et n'est pas touche).

Instruments (tous sous garde d'environnement, sautes en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/chunk00_carte_research_test.go` | carte exhaustive, occupation, octets hors zones lues, chaines, champ `kind`, diff entre films, fenetres hexadecimales |
| `apps/go-api/internal/analysis/filmdec/registre_events_research_test.go` | D1 : noms d'evenement en clair, table par type, groupement par build |
| `apps/go-api/internal/analysis/filmdec/registre_hachage_research_test.go` | D1 : noms d'evenement haches, avec faux positifs mesures |
| `apps/go-api/internal/analysis/filmdec/chunk00_section3_research_test.go` | D2 : caracterisation de la troisieme section |
| `apps/go-api/internal/analysis/filmdec/event_types_catalogue_test.go` | donnee : le catalogue des types 50..127 lu dans l'exe (lot B) |
| `apps/go-api/internal/analysis/replay/visee_typage_trame_research_test.go` | D3 / D4 : cadrage du premier octet, confronte a la table du film |

---

## Resume executif

1. **`chunk_00` n'est pas « le registre ». Il porte TROIS sections**, et le depot n'en lisait
   qu'une. Sur 1 973 120 octets inflates : le registre occupe les 832 000 premiers (42 %), une
   section d'en-tete suit, puis une troisieme section d'environ un demi-megaoctet qui differe
   integralement d'un film a l'autre.
2. **Le « 118 blocs d'archetype » du dossier est un artefact de division.** `parseRegistry`
   divise le FICHIER ENTIER par la taille d'un bloc ; les blocs 50 et suivants ne sont pas du
   registre. Le registre fait **50 blocs** (0..49), dont 49 porteurs. Le compte de couples
   (archetype, composant) — 1 067 — reste juste.
3. **D1, volet « en clair » : NEGATIF, avec temoin positif.** Aucun des 13 noms d'evenement du
   catalogue de l'exe n'apparait dans `chunk_00` ni dans aucun autre chunk, en ASCII comme en
   UTF-16, alors que la meme recherche retrouve bien les noms de composant. Le film ne nomme pas
   ses evenements.
4. **D1, volet « hache » : sans precedent a imiter, et negatif.** Le champ `kind` du registre,
   seul candidat d'empreinte dans le format de slot, vaut **0 sur 1 066 des 1 067 slots** : le
   registre n'accompagne pas ses noms d'un hachage. Onze fonctions de hachage 32 bits ont ete
   passees sur les noms d'evenement, avec pour temoin negatif des leurres de meme forme dont le
   taux de touche EST le taux de faux positifs mesure sur ce flux.
5. **MAIS LE FILM DECLARE BIEN QUELQUE CHOSE PAR TYPE, ET CETTE TABLE SUIT LE BUILD.** Juste
   apres le registre, a `0x0CB208`, une table d'entiers u32 s'arrete exactement a la chaine
   d'identification du build. Sur les 1 367 `chunk_00` du cache, **son cardinal grandit avec la
   version du jeu** : 119 types en `HI_1_5_1`, 121 en `HI_1_8_0` a `HI_1_10_0`, 122 en
   `HI_1_11_0`, **123 en `HI_1_12_0` et `HI_1_13_0`**. Et **123 est exactement la borne du
   dispatcher relevee independamment dans l'exe au lot B** (`CMP R15,0x7b ; JNC`, types 0..122).
   Deux chaines sans etape commune, meme nombre. **L'hypothese de l'utilisateur est confirmee dans
   ce sens precis : le film DECRIT sa grammaire de trame par type et par build. Il ne la NOMME
   pas.**
6. **Corollaire lourd pour tout le chantier : l'espace des numeros de type n'est pas stable entre
   builds.** Un film de `HI_1_10_0` declare 121 types la ou un film de `HI_1_13_0` en declare 123.
   Une etiquette tiree de la table statique d'UN binaire ne se transpose donc ni aux autres builds
   ni, sans preuve d'alignement, au build analyse.
7. **Consequence directe sur le dossier lunette** : le type 126 `unit_zoom` est **hors de la
   plage de types que le film declare**, et l'ecart ne fait que grandir vers le passe. Son absence
   des 41 millions de paquets mesures a la phase 3 n'est donc pas une observation empirique
   fragile, c'est une **impossibilite structurelle**. Et l'incertitude n 5 de
   `NOTE_ENVELOPPE_EVENTS` (« borne 123 ou 128 ? ») est tranchee par le film lui-meme : **123**.
8. **La section d'identification donne le build en clair** : `6.10026.18411.0`, `HI_1_13_0`,
   `release`. Elle est identique sur les films du meme build ; le premier octet qui differe entre
   deux films tombe apres elle, a `0x0CB65C`.
9. **La troisieme section porte l'identite des joueurs.** Des gamertags y sont ecrits en UTF-16LE
   — `Chocoboflor` (un joueur suivi par le projet), `vamprym`, `RAZOR BLADE LEO` sur `00162144` ;
   `whiteknight2519`, `LORD PEINX13` sur `000d5950`. Le reste est dense et non decode.
10. **Le premier octet d'un paquet delta n'est pas un type d'evenement, et le bit de poids faible
    du lot D4 est explique.** 219 702 paquets du corpus font UN octet — trop court pour l'en-tete
    minimal d'un evenement (11 bits). Sous la grammaire de trame, le bit en question tombe dans
    l'identifiant du premier record : mesure, ce champ prend **7 valeurs** sur les paquets `0xD2`
    contre **50** sur les `0xD3`, ce qui suffit a expliquer « 9 identifiants d'arme propres d'un
    cote, 447 de l'autre » sans invoquer aucune variante de record.

---

## D2. La carte

### D2.1 Les trois sections

Mesure sur `00162144`, `000d5950`, `00502e52` — **taille inflatee identique au bit pres sur les
trois : 1 973 120 octets**, alors que les fichiers compresses different (433 937 / 435 425 /
434 401 octets).

| section | bornes | taille | contenu | statut |
|---|---|---|---|---|
| 1. Registre | `0x000000` .. `0x0CB200` | 832 000 o | 50 blocs de 64 slots de 260 o ; 49 blocs porteurs, 1 067 slots nommes | **connu** (`registry.go`) |
| 2. En-tete | `0x0CB200` .. `0x0CB45C` | 604 o | table de 123 u32 + trois champs de 32 o (version, build, saveur) + deux u32 | **nouveau (lot D)** |
| 3. Corps | `0x0CB65C` .. `0x14EB73` | ~538 200 o | donnee dense, propre au film | **nouveau, NON DECODE** |
| 4. Queue | `0x14EB74` .. `0x1E1C00` | ~602 100 o | zeros | — |

Le registre ne represente que **2,38 % des octets que `parseRegistry` lit reellement**
(46 902 octets de `kind`+`flags`+noms sur 1 973 120).

### D2.2 Occupation du registre, bloc par bloc

49 blocs porteurs, 1 067 slots (`bloc:slots`) :

```
0:27 1:15 2:18 3:2 4:1 5:27 6:58 7:64 9:10 10:30 11:34 12:28 13:34 14:3 15:4 16:6 17:2
18:32 19:32 20:3 21:18 22:1 23:33 24:27 25:1 26:33 27:64 28:1 29:32 30:3 31:11 32:1 33:1
34:17 35:64 36:20 37:31 38:20 39:20 40:48 41:22 42:21 43:41 44:1 45:2 46:64 47:3 48:2 49:35
```

Le bloc 8 est vide ; le dernier porteur est le bloc 49 ; la section suivante commence exactement
au bloc 50. **Piege signale** : sur `00162144`, `parseRegistry` rend **1 068** slots et un
50e « porteur » au bloc 71 — un faux positif, une suite de 260 octets de la section 3 qui passe
pour un nom ASCII termine par NUL. L'empreinte du registre de ce film differe donc de
`KnownRegistryFingerprint` sans qu'aucune grammaire n'ait bouge. Decouverte hors perimetre,
**non traitee** (regle 7) : `parseRegistry` devrait s'arreter au bloc 50 au lieu de diviser le
fichier entier.

### D2.3 La section 2 : ce que le film declare par type

```
0x0CB200  00 00 00 00 00 00 00 00   deux u32 nuls (ou bourrage du bloc precedent)
0x0CB208  123 x u32                 la TABLE PAR TYPE — valeurs 1..6
0x0CB3F4  "6.10026.18411.0"         version, champ de 32 octets
0x0CB414  "HI_1_13_0"               build, champ de 32 octets
0x0CB434  "release"                 saveur, champ de 32 octets
0x0CB454  0x0004187B 0x0086ED94     deux u32 (268 923 et 8 842 644) — role inconnu
```

Distribution des 123 valeurs : **1** x 104 · **2** x 9 · **3** x 5 · **4** x 2 · **5** x 2 ·
**6** x 1.

**Ce que la table est, et ce qu'elle n'est pas.** Elle porte UN entier court par type d'evenement,
elle est identique sur les films d'un meme build, et son cardinal egale la borne du dispatcher.
Elle ne porte AUCUN nom. La lecture la plus economique de valeurs entre 1 et 6, une par type,
figees par build, est une **version de serialisation par type** — la table qui permet a un film
d'etre rejoue par un moteur dont le codage a change. Cette lecture n'est pas prouvee : elle est
la seule compatible avec les trois faits mesures (une valeur par type, petites, constantes par
build). Ce qu'il faudrait pour la prouver : lire la fonction qui consomme ce champ dans l'exe —
travail Ghidra, hors du perimetre de ce lot.

### D2.3 bis La table suit le build — la preuve, sur 1 367 films

`TestD1Builds` lit le `chunk_00` de chaque film du cache et groupe par (build, version, cardinal
et empreinte de la table). **15 groupes**, dont 5 films sans section d'identification (chunk_00
incomplet). Le cardinal publie ci-dessous est celui du walk arriere, qui ramasse un u32 nul de
bourrage en tete : le nombre de TYPES est ce cardinal moins un.

| build | version | cardinal du walk | types declares | empreinte de la table | films |
|---|---|---|---|---|---|
| `HI_1_13_0` | 6.10026.18411.0 | 124 | **123** | `5f210b147befd52d` | 798 |
| `HI_1_13_0` | 6.10026.19225.0 | 124 | **123** | `5f210b147befd52d` | 308 |
| `HI_1_12_0` | 6.10026.17802.0 | 124 | **123** | `5f210b147befd52d` | 143 |
| `HI_1_12_0` | 6.10026.16630.0 | 124 | **123** | `5f210b147befd52d` | 4 |
| `HI_1_11_0` | 6.10026.15293.0 | 123 | **122** | `f7a27c1e0f960824` | 53 |
| `HI_1_11_0` | 6.10026.14397.0 | 123 | **122** | `f7a27c1e0f960824` | 4 |
| `HI_1_10_0` | 6.10026.13096.0 / .12282.0 / .13865.0 | 122 | **121** | `190fffd3ee6172c7` | 34 |
| `HI_1_9_0` | 6.10026.10506.0 / .11084.0 | 122 | **121** | `190fffd3ee6172c7` | 3 |
| `HI_1_8_0` | 6.10025.19737.0 | 122 | **121** | `e6ea0f9d098e0247` | 13 |
| `HI_1_5_1` | 6.10025.12948.0 | 120 | **119** | `032003cafcb59303` | 1 |
| — | sans section d'identification | 0 | — | — | 5 |

Trois faits se lisent dans ce tableau, et ils se renforcent :

1. **Le nombre de types croit avec la version du jeu** : 119, puis 121, puis 122, puis 123. C'est
   le comportement d'une table de types a laquelle on ajoute des entrees a chaque mise a jour.
2. **A version de jeu egale mais patch different, la table est identique** (les deux versions de
   `HI_1_13_0` partagent `5f210b14…`) : elle ne depend pas du patch, mais de la revision du
   format.
3. **A cardinal EGAL, les valeurs peuvent changer** : `HI_1_8_0` et `HI_1_9_0` declarent tous deux
   121 types mais avec des tables differentes (`e6ea0f9d…` contre `190fffd3…`). **Un contenu qui
   change sans que le cardinal bouge est exactement ce que fait une version par entree** — c'est
   le meilleur appui empirique de la lecture « version de serialisation par type », et il ne
   suppose rien du binaire.

La table du build de reference est versionnee en artefact :
`.ai/V7.5/film_re/chunk00_table_par_type.tsv` (123 lignes, valeur et nom catalogue en regard, avec
l'avertissement d'alignement en tete de fichier).

**Le decalage d'indexation reste ouvert.** Les 8 octets nuls a `0x0CB200` sont indiscernables du
bourrage du bloc 49, donc le DEBUT de la table n'est pas determinable sur pieces : 123 entrees a
`0x0CB208`, ou 125 a `0x0CB200`, ou 128 a `0x0CB1F4`, donnent la meme suite d'octets. C'est le
cardinal **123** que retient ce lot, parce qu'il fait coincider la fin de la plage de types avec
la borne du dispatcher mesuree par une tout autre voie — mais la coincidence n'est pas une preuve
d'alignement des index.

### D2.4 La section 3 : ce qu'on en sait, et ce qu'on n'en sait pas

Environ 538 200 octets par film, dense, **integralement differente d'un film a l'autre** : la
comparaison octet a octet de deux `chunk_00` du meme build donne 456 547 octets differents sur
1 973 120, et le premier ecart tombe a `0x0CB65C`, c'est-a-dire au debut exact de cette section.
Tout ce qui la precede — registre, table par type, identification du build — est **identique octet
pour octet** entre films du meme build. La croyance du depot (« le registre est identique d'un
film a l'autre ») est donc juste, et le lot la precise : elle vaut pour les deux premieres
sections, pas pour le fichier.

Cette section porte a elle seule la quasi-totalite du poids compresse de `chunk_00` (les 832 000
octets de registre sont a 95,40 % des zeros de bourrage). Ce que les mesures en disent :

| mesure | `00162144` | `000d5950` | `00502e52` |
|---|---|---|---|
| bornes / taille | `0x0CB65C`..`0x14EB73`, 537 880 o | ..`0x14E89F`, 537 156 o | ..`0x14E31D`, 535 746 o |
| entropie | 7,244 bits/o | 7,272 | 7,273 |
| octets nuls | 17,40 % | 16,90 % | 16,89 % |
| lecture en paquets de 16 o d'en-tete | **2,3 % du volume couvert** | idem | idem |
| debut de la donnee vraiment dense | `0x0CE6A8` (12 364 o apres le debut) | `0x0CE698` | — |

- **Ce n'est pas un flux de paquets.** Le decoupage a en-tete de 16 octets, celui de tous les
  autres chunks, ne couvre que 2,3 % du volume — et les « paquets » qu'il trouve sont les zeros de
  l'entree de section lus comme des en-tetes vides.
- **Ce n'est pas un flux compresse imbrique.** 27 en-tetes zlib plausibles la ou le hasard en
  donnerait environ 131 : moins que le hasard, donc rien.
- **Aucun pas de structure ne ressort.** La part d'octets egaux a distance d'un pas decroit
  regulierement de 17,0 % (pas 2) a 15,1 % (pas 1024) sans aucun pic : pas d'enregistrement de
  taille fixe a ces pas. Le niveau general, tres au-dessus des 0,391 % d'octets uniformes, vient
  des 17 % de zeros et de la repetition locale.
- **Elle porte l'identite des joueurs.** Recherche des chaines UTF-16LE : sur `00162144`,
  `vamprym` (`0x13CA9A`), `Chocoboflor` (`0x13F378`), `RAZOR BLADE LEO` (`0x1408A2`) ; sur
  `000d5950`, `whiteknight2519` (`0x13CCA6`), `LORD PEINX13` (`0x13F7AF`). **`Chocoboflor` est un
  joueur suivi par le projet** : ce sont bien des participants, pas des suites d'octets heureuses.
  Trois seulement par film, et espacees de ~10 ko : le reste du roster n'apparait pas en clair,
  ce qui est attendu d'un flux bit-packe ou seules les chaines tombant sur un octet plein restent
  lisibles.

**La section n'est pas decodee par ce lot et ne pretend pas l'etre.** Ce qui est etabli : elle
existe, elle fait un demi-megaoctet par film, elle est propre au match, elle porte au moins
l'identite des joueurs, et elle n'obeit a aucun des trois encadrements deja connus du format.

---

## D1. La table de noms d'evenements : elle n'existe pas dans le film

### D1a. En clair — negatif avec temoin positif

Recherche par sous-chaine dans `chunk_00` ET dans tous les autres chunks du film, en ASCII et en
UTF-16LE, avec trois orthographes voisines par nom (`_` remplace par `-`, suffixe `-component`,
`_` remplace par une espace) :

| nom cherche | `chunk_00` | autres chunks |
|---|---|---|
| `action_weapon_fire`, `biped_board_vehicle`, `unit_zoom`, `weapon_effect`, `weapon_overheat`, `projectile_detonate`, `unit_exit_vehicle`, `player_set_orbiting_camera_target`, `unit_switch_seat`, `weapon_empty_click`, `DebugSendCameraPosition`, `biped_pickup`, `equipment_teleport_request` | **0** | **0** |

**Temoin positif, dans la meme passe** : `object-position-dynamic-precision` = 2,
`unit-desired-aiming-vector` = 2, `weapon-state-type-info` = 4,
`biped-spartan-ability-energy` = 1. L'instrument trouve ce qui est present ; il ne trouve pas les
noms d'evenement parce qu'ils n'y sont pas.

### D1a bis. La question de l'INDEX, posee par le pilote, et ce que la mesure en dit

La question exacte etait : « un type de zoom figure-t-il dans le film sous un index DIFFERENT de
celui de la table statique de l'exe ? » Elle est legitime, parce que le zoom ne peut voyager que
par un evenement (aucun deserialiseur de replication n'ecrit l'octet de zoom de l'unite — verifie
par le pilote, hors de ce lot).

Ce que la mesure permet de dire, et ce qu'elle ne permet pas :

- **Le film ne nomme rien.** Il n'y a aucune table de noms d'evenements, donc aucun moyen, sur les
  seuls octets du film, de repondre « le type N s'appelle X ». La question ne se resout pas par
  lecture directe.
- **Mais le film COMPTE.** Il declare **123** types ; la table statique de l'exe en nomme **128**.
  Les deux espaces d'index ne peuvent donc PAS coincider sur toute leur longueur : **cinq entrees
  nommees de l'exe n'ont pas de place dans l'espace que le film declare.** Si elles sont retirees
  a la fin, les etiquettes 0..122 tiennent et `unit_zoom` (126) disparait simplement. Si elles sont
  retirees ailleurs, **toutes les etiquettes situees apres le point de retrait glissent**, et un
  type de zoom pourrait se trouver a un index plus bas. **Ce lot ne tranche pas entre ces deux
  cas** — et c'est precisement pourquoi les etiquettes de type manipulees depuis la phase 3
  doivent etre tenues pour non fiables.
- **Ce qui tient sans aucune hypothese d'alignement** : le dispatcher de l'exe refuse tout type
  `>= 123` (`CMP R15,0x7b`), et `unit_zoom` est a l'index statique 126 **dans la numerotation de
  l'exe elle-meme**. Sous cette numerotation, il n'est pas dispatchable. Le raisonnement ne
  traverse jamais la frontiere entre les deux espaces d'index : il reste du cote de l'exe.

**Le geste qui fermerait la question, et il est bon marche** : la table du film porte 123 valeurs
entre 1 et 6, avec un motif distinctif (une seule valeur a 6, deux a 5, deux a 4, cinq a 3, neuf a
2). Si un lot Ghidra lit le tableau equivalent cote exe — la version de serialisation par type,
consommee au voisinage de `0x144724A90` —, **l'alignement des deux suites se lit a l'oeil** et
l'index de chaque type nomme devient certain. C'est le handoff que ce lot laisse.

### D1b. Haches — pas de recette a imiter

Le format de slot du registre est `[u32 kind][u32 flags][nom ASCII]`. Si le film hachait des noms,
`kind` en serait le porteur naturel. Mesure : **`kind` vaut 0 sur 1 066 des 1 067 slots nommes**
(les deux valeurs restantes appartiennent au faux positif du bloc 71 et a un slot de bord). Il n'y
a donc **aucune empreinte de nom dans le registre** : « meme recette que les composants » veut dire
« en clair », et le volet hache n'a pas de precedent.

Le test est fait quand meme, avec des faux positifs **mesures** : onze fonctions 32 bits usuelles
(FNV-1a, FNV-1, CRC-32 IEEE, CRC-32C, djb2, djb2-xor, sdbm, ELF, Jenkins, MurmurHash3, somme
d'Adler), recherche aux deux boutismes et **a tout offset, pas seulement aligne** — le tamis le
plus permissif possible, 821 169 valeurs u32 distinctes indexees sur `00162144`. Trois populations
y passent :

| population | resultat, sur chacun des 3 films et chacune des 11 fonctions |
|---|---|
| 13 noms d'evenement | **0 touche** |
| 325 noms de composant (temoin) | **0 touche** |
| 13 leurres (deux lettres permutees) | **0 touche** |

Le taux de faux positifs mesure est donc **nul** : le negatif sur les evenements n'est pas noye
dans du bruit, il est net. Et le temoin des composants dit pourquoi : **ce format ne hache aucun
nom, meme ceux qu'il ecrit en clair juste a cote.**

---

## D3 / D4. Le premier octet d'un paquet delta

### Ce que le lot D apporte comme arbitre

La phase 7 laissait la « taxonomie NON TRANCHEE » : cadrage A (`type = octet >> 1`, ce que lit
`fire_events.go`) contre cadrage B (`type = octet & 0x7F`, le bit de poids fort etant le bit de
continuation etabli chez l'ecrivain au lot B2). Le lot D ajoute deux arbitres, et un troisieme
argument qui tranche sans statistique.

### Le recensement, a l'echelle du corpus

`TestD4CadrageOctet0` sur **1 369 films, 42 642 703 paquets delta**, 186 s. **50 valeurs
distinctes de premier octet**, dont quatre couvrent 91 % du flux :

| octet | paquets | part | taille min/moy/max | A : `o>>1` | B : `o&0x7F` | grammaire de trame |
|---|---|---|---|---|---|---|
| `0xA0` | 34 323 692 | 80,49 % | 6 / 210 / 3 399 | 80 | 32 | DELTA |
| `0xD2` | 2 535 816 | 5,95 % | 52 / 368 / 5 165 | 105 | 82 | DEL |
| `0xC7` | 1 023 286 | 2,40 % | 20 / 325 / 37 043 | 99 | 71 | FIN (vue vide) |
| `0xC0` | 983 883 | 2,31 % | 11 / 401 / 9 995 | 96 | 64 | FIN (vue vide) |
| `0xE9` | 922 724 | 2,16 % | 38 / 373 / 10 364 | 116 | 105 | DELTA |
| `0xD3` | 528 262 | 1,24 % | 29 / 330 / 6 220 | 105 | 83 | DEL |
| `0x80` | 219 702 | 0,52 % | **1 / 1 / 1** | 64 | 0 | FIN (vue vide) |
| `0xE5` | 195 824 | 0,46 % | 20 / 364 / 4 552 | 114 | 101 | DELTA |

**Invariant a l'echelle du corpus : le bit de poids fort du premier octet vaut 1 sur les
42 642 703 paquets, sans exception.** Le desassemblage du frame-processor l'avait etabli comme un
drapeau de configuration lu par un `R(1)` en tete de payload, et mesure a 100 % sur les 30 418
payloads d'un seul film (`frame_records.go`) ; il est desormais verifie sur quatre ordres de
grandeur de plus. Sous le cadrage B, ce meme bit devrait etre le bit de continuation de la liste
d'evenements : il faudrait alors que **chaque** trame du corpus, sans exception, porte au moins un
evenement.

### L'argument de longueur — decisif

L'en-tete minimal d'un evenement fait **11 bits** sous les deux cadrages : 7 bits de type, 3
portes de reference, plus le bit de continuation (B) ou de variante (A). **11 bits ne tiennent pas
dans un octet.** Or le corpus contient **219 702 paquets delta dont le payload fait exactement 1
octet, valant `0x80`** (0,515 % du flux ; 160 sur le seul `00162144`).

La lecture parcimonieuse de ces paquets est une **trame vide** : amorce de 2 bits (`10`) puis
boucle de records close aussitot (`R(1)=0`, `R(2)=00` = FIN) — exactement ce que lit le decodeur
de production (`frame_records.go`, `DefaultPacketPreambleBits = 2`). **Le premier octet d'un paquet
delta n'est donc pas un en-tete d'evenement** : c'est le debut d'une TRAME, et les « types »
manipules depuis la phase 3 sont une lecture d'un champ qui n'est pas celui qu'on croyait.

### Les criteres qui NE tranchent pas, publies comme tels

Trois seuils avaient ete ecrits avant la mesure ; deux ne concluent pas, et ils sont publies
quand meme.

| critere | mesure sur le corpus | verdict declare |
|---|---|---|
| **S1** hors de la plage de 123 types | cadrage A : 20 529 paquets (0,0481 %), octets `0xF6`, `0xF8`, `0xFA`, `0xFB` — cadrage B : 329 paquets (0,0008 %), octet `0xFB` | **NON CONCLUANT des deux cotes** (seuil 0,1 %). Le cadrage A est 62 fois plus fautif que B, mais le seuil ecrit avant la mesure ne permet pas d'en conclure, et il n'est pas deplace apres coup. |
| **S2** enumeration ou donnee libre | 50 valeurs distinctes sur 1 369 films | **NON CONCLUANT** (entre 32 et 64) |
| **S3** evenement greffe ou trame autonome | 0,0000 % des paquets partagent leur horodatage avec un autre paquet delta du meme chunk | **TRAMES AUTONOMES** : un paquet delta = un instant. Rien ne se greffe sur rien. |
| **S4** argument de longueur | 219 702 paquets d'un octet | **REFUTE** le cadrage « un paquet delta = un evenement » |

S3 et S4 vont dans le meme sens et n'ont aucune etape commune : l'un compte des horodatages,
l'autre compte des octets.

**Un cinquieme critere ne conclut pas, et il est publie tel quel.** `TestD3CadenceDesTrames`
devait confirmer positivement le modele de trame en montrant que les paquets se succedent a
cadence reguliere et que les `0xE5` y occupent des places ordinaires. Le critere exigeait que
l'ecart modal represente plus de 80 % des ecarts. Mesure sur trois films (91 091 ecarts) :
**l'ecart modal vaut 16 662 us — soit tres exactement la periode d'un tick a 60 Hz — mais il ne
represente que 0,17 % des ecarts.** La cadence est trop dispersee pour servir de temoin ; le
verdict declare est **NON CONCLUANT** et le seuil n'est pas redecoupe apres coup. La valeur de
l'ecart modal est consignee comme observation brute, sans en tirer de conclusion.

### Ce que cela change, et ce que cela ne change pas

- Cela **ne casse rien en production** : `fire_events.go` lit un champ de 64 bits a des offsets
  FIXES dans les paquets `0xD2` et cette lecture est validee par ailleurs (killsource 59/59 en
  Theater). Ce qui tombe, c'est l'ETIQUETTE « type 105 = `action_weapon_fire` », pas le decodage.
- Cela **explique le fait mesure du lot D4, et la mesure le confirme.** Sous la grammaire de
  trame, l'identifiant du premier record commence au bit 5 (amorce 2 bits, `R(1)` de type,
  `R(2)` de type) et court sur `IDLowBits` bits — 11 sur le film calibre. **Le bit 7, celui dont
  D4 devait etablir la semantique, en est le troisieme.** Critere ecrit avant la mesure : un
  champ d'identifiant prend un nombre restreint de valeurs (moins de 25 % des 2 048 possibles),
  un champ de bruit les prend presque toutes. Mesure sur trois films, 90 835 paquets :

  | population | valeurs distinctes / 2 048 | verdict declare |
  |---|---|---|
  | tous les paquets | 435 (21,2 %) | **IDENTIFIANT** |
  | paquets `0xD2` | **7** (0,3 %) | **IDENTIFIANT** |
  | paquets `0xD3` | **50** (2,4 %) | **IDENTIFIANT** |

  **Sept entites contre cinquante.** Les paquets `0xD2` forment une population homogene dont le
  premier record ne vise que 7 entites — d'ou un champ stable a offset fixe, et les 9 identifiants
  d'arme propres. Les `0xD3` en visent 50 : leurs octets a l'offset 44..107 ne decrivent pas la
  meme chose d'un paquet a l'autre, d'ou 447 valeurs distinctes. **Le bit de poids faible n'est ni
  une variante de record ni un bit de continuation : c'est un bit de l'identifiant du premier
  record de la trame.** La question ouverte du lot D4 est fermee.
- Cela **retire tout fondement** aux identifications de type faites par les phases 3 a 6 — dont
  « le zoom est le type 114 », deja retracte par la voie statistique au lot A. Le retrait est
  desormais adosse a une troisieme chaine, structurelle.

### Le type « zoom » dans le film

Le film declare 123 types (0..122). **Le type 126 `unit_zoom` est hors de cette plage**, sous les
deux cadrages : aucun film de ce build ne peut l'exprimer. Les octets qui le coderaient (`0xFC`,
`0xFD` en cadrage A ; `0xFE`, `0x7E` en cadrage B) sont absents du flux mesure.

### Les 125 paquets « 114 » de `00162144`

Ils valent tous `0xE5` en premier octet, sur une carte sans aucun vehicule. Le lot D ne les
renomme pas — il retire la question : `0xE5` n'est pas un numero de type. Sous la grammaire de
trame, `0xE5 = 1110 0101` se lit amorce `11`, puis `R(1)=1` = record DELTA. Ces 125 paquets sont
125 trames dont le premier record est un delta, rien de plus. L'anomalie « 125 evenements
d'embarquement en vehicule sans vehicule » n'a jamais eu besoin d'explication semantique : elle
etait une erreur de cadrage.

Precision utile pour qui relira les phases anterieures : l'octet du « type 114 » est `0xE5`
(195 824 paquets sur le corpus, coherent avec les « 197 255 sur les 1 367 films » de la phase 6),
et non `0xE4` comme l'ecrit la note de cloture de la phase 7 — `0xE4` n'apparait que 1 431 fois
sur tout le corpus. Les deux octets tombent sur le meme « type 114 » sous le cadrage A parce que
celui-ci jette le bit de poids faible ; c'est precisement ce bit que le lot D4 devait expliquer.

---

## Ce qui reste ouvert

1. **La section 3 de `chunk_00`** (~538 ko par film) n'est pas decodee. C'est la plus grosse
   donnee inexploree du format a ce jour, et elle porte au moins l'identite des joueurs.
2. **Le decalage d'indexation de la table par type** (123 a `0x0CB208` ou 125 a `0x0CB200`) n'est
   pas determinable sur les seuls octets du film. **Geste qui le ferme** : lire cote exe le
   tableau equivalent et aligner les deux suites — l'artefact `chunk00_table_par_type.tsv` est
   pret pour cela.
3. **La semantique des valeurs 1..6** : « version de serialisation par type » est la lecture que
   trois faits mesures soutiennent (une valeur par type, petites, cardinal croissant avec le
   build, contenu qui change a cardinal constant entre `HI_1_8_0` et `HI_1_9_0`). Ce n'est pas
   une preuve ; la fonction qui les consomme n'a pas ete lue.
4. **Les deux u32 de `0x0CB454`** (268 923 et 8 842 644) n'ont pas de role etabli.
5. **`parseRegistry` divise le fichier entier par la taille d'un bloc** et ramasse des faux
   positifs dans la section 3 (mesure : un sur `00162144`, qui rend 1 068 slots au lieu de 1 067
   et fausse l'empreinte du registre de ce film). Decouverte hors perimetre, **non traitee**
   (regle 7) : le correctif tiendrait en une borne a 50 blocs.
6. **Le modele de trame n'a pas de confirmation POSITIVE** : S3 et S4 disent ce que le premier
   octet n'est pas, la cadence des trames ne conclut pas. Le confirmer demanderait de rejouer le
   decodeur de production sur les paquets `0xD2`/`0xE5` et de verifier que la chaine de records
   s'y ferme proprement — instrument non ecrit par ce lot.
7. **Les etiquettes de type utilisees par les phases 3 a 7 sont sans fondement etabli**, et il
   faut le tenir pour acquis tant que le point 2 n'est pas ferme.
