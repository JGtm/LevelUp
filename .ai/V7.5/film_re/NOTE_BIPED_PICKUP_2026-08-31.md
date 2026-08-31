# biped_pickup (type 9) EST DECODE : le ramassage est date, nomme ET attribue au ramasseur

> Historique de cette note. Le **lot 1** (grammaire, cadrage, confrontation produit) concluait
> « nomme et date — l'attribution reste a faire » : `ref0` n'etait pas identifiee. Le **lot 2**
> l'a resolue — `slot du ramasseur = 512 + ref0.index`, exact sur 32/32 paires de verite
> terrain. Le titre a ete corrige en consequence. Ce qui reste hors de portee de l'evenement
> est desormais l'INSTANCE de l'objet ramasse (donc le socle d'origine), pas le ramasseur.

Date : 2026-08-31. Chantier RAMASSAGE, worktree `wt/biped-pickup`. Instruments :
`apps/go-api/internal/analysis/filmdec/biped_pickup_{research,grammaire,confront,ref0,ref0_couverture}_test.go`,
sous garde `BIPED_PICKUP_FILM` (sautes sans elle, aucun effet en CI, aucune publication
production dans ces lots).

## Ce qui est PROUVE

1. **Le type 9 est bien `biped_pickup`, lu dans le binaire et non repris d'une note.** La
   table des descripteurs d'evenements vit a `ctx+0x210 + type*8` (marcheur de liste
   `FUN_14080a9d4`), remplie par `FUN_140e453b4`. Entree du type 9 : **0x144724e18**, vtable
   **0x143d0d758**. L'entree `vtable+0x08` de cette vtable vaut `0x141164e10`, l'UNIQUE
   fonction qui reference la chaine `"biped_pickup"` (0x143c97f98). Controle de methode : la
   meme lecture donne type 0 -> 0x144724f80 et type 21 -> 0x144724e80, exactement les deux
   descripteurs que le chantier trame avait etablis par un autre chemin.
2. **La grammaire complete, bit a bit.** Domaines des 3 references (`vtable+0x58` =
   `0x1410f92bc`) : `LEA EAX,[RDX+2]` pour l'index 0, puis bloc froid partage `0x14232a4ba`
   qui rend 8 pour l'index 1 et 7 pour l'index 2. Charge (`vtable+0x68` = `FUN_141037828`).
   Reference gardee (`FUN_1406d3140`) : `R(1) porte ; R(w) index ; R(2) generation` —
   TOUJOURS la generation, et seul le domaine 1 porte une sonde.

   ```
   [1 continuation][R(7) type = 9]
   ref0 : R(1) porte ; si 1 : R(8) index + R(2) gen      <- domaine 2 : LE RAMASSEUR (slot = 512 + index)
   ref1 : R(1) porte ; si 1 : R(13) + R(2)               <- domaine 8  (jamais presente)
   ref2 : R(1) porte ; si 1 : R(13) + R(2)               <- domaine 7  (jamais presente)
   charge : R(3) classe ; R(1) porte ; si 1 : R(32) identifiant d'objet
            (sinon la sentinelle 0xFFFFFFFF ; hors ligne on garde le brut — la resolution
             FUN_1407f21b4 est une table de jeu, elle ne consomme aucun bit)
   [1 fin de liste]
   ```

   Longueur modale : `1+8+2 + 1 + 1 + 3+1+32 + 1 = 50` bits apres le champ de type.
3. **Le cadrage tient, juge par l'oracle de trame, sur deux films.** Longueur mesuree :
   **50 bits sur 100/100** evenements seuls (000d5950, film entier) et **50 sur 60/60**
   (00502e52) — aucune variance, aucune exception. Et la longueur 50 est le PIC du scan
   empirique aveugle (etape 2, sans aucune hypothese de grammaire) : deux chaines
   independantes se ferment sans ajustement.

## L'oracle, calibre AVANT de servir (et le piege qu'il a fallu lever)

Le juge n'est pas la profondeur de trame mais **« la trame se ferme proprement ET consomme le
payload jusqu'a moins d'un octet de la fin »**. Il n'a de valeur qu'apres deux calibrations,
faites avant toute mesure de cadrage :

- **`FrameConfig.IDLowBits` est une valeur de RUNTIME, propre au film.** Avec le defaut 13, la
  separation de l'oracle etait de **1,9x** — inutilisable, et la premiere version du scan n'a
  rien tranche. Balaye contre la verite terrain des trames PURES (liste vide, cadrage connu au
  bit 2) : **9 sur 000d5950** (96,2 % de trames exactes contre 0,7 % pour le defaut 13) et
  **11 sur 00502e52** (85,2 %). C'est la meme table d'exe (`DAT_1451f98d0/d4`) qui donne les
  largeurs de reference — d'ou la verification separee de la largeur du domaine 2 ci-dessous.
- **Pouvoir separateur, une fois calibre** (000d5950, n=3000) : bon cadrage **93,0 %** ; +1 bit
  **0,0 %** ; +2 **0,0 %** ; +3 **0,0 %**. Sur 00502e52 : 85,2 % contre 0,0 / 0,0 / 0,0.

**Plafond, mesure et pas suppose.** Un paquet qui porte un evenement n'est pas un paquet
ordinaire (creations d'entites que le decodeur de trame traverse mal). Le plafond est donc
mesure sur une famille dont la grammaire est DEJA PROUVEE, `unit_zoom` (type 21) : **82,7 %**
(n=162) sur 000d5950 et **77,6 %** (n=152) sur 00502e52 — longueur 17 bits sur 162/162 et
152/152, conforme a la grammaire connue.

## Le verdict du cadrage, chiffres bruts (film ENTIER : 27 et 29 chunks)

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| paquets 0xC4 · dont type 9 | 135 · **135 (100 %)** | 73 · **73 (100 %)** |
| dont type 8 (`biped_board_vehicle`) | **0** | **0** |
| longueur de l'evenement | **50 bits sur 100/100** | **50 bits sur 60/60** |
| ref0 presente | 100 % | 100 % |
| ref1 / ref2 presentes | 0 % / 0 % | 0 % / 0 % |
| charge R(32) presente | 100 % | 100 % |
| trames EXACTES au cadrage retenu | **57,0 %** (57/100) | **55,0 %** (33/60) |
| plafond `unit_zoom` du meme film | 82,7 % | 77,6 % |
| part du plafond atteinte | **69 %** | **71 %** |
| temoins -1 / +1 / +2 / +3 bits (fenetre 12 chunks) | 0,0 / 0,0 / 0,0 / 0,0 % | 0,0 / 0,0 / 0,0 / 0,0 % |
| temoin -2 bits (fenetre 12 chunks) | 20,0 % | 0,0 % |

**Largeur du domaine 2, balayee au lieu d'etre supposee** (elle vient de la meme table de
runtime que `IDLowBits`) : sur le film entier, **R(8) = 60,0 %** contre une meilleure largeur
voisine a **3,5 %** (000d5950) et **60,0 %** contre **3,6 %** (00502e52). Toutes les autres
largeurs de la fenetre balayee [4,14] restent sous 9 %. Pic unique, largement separe : R(8) est
la largeur — et elle est la MEME sur les deux films alors que `IDLowBits` differe (9 vs 11),
ce qui montre que toutes les entrees de la table de runtime ne varient pas ensemble.

Le seuil pre-enregistre de ce balayage (« >= 74 %, soit 80 % du plafond `unit_zoom` ») avait ete
ecrit sur la fenetre temoin de 12 chunks, ou le plafond valait 92,7 %. Il n'est PAS atteint, et
il n'est pas reecrit apres coup : le verdict publie reste NON TENU. Ce que la mesure etablit,
c'est le pic unique et sa separation, pas le franchissement de ce seuil-la.

## La confrontation produit — c'est elle qui decide

Film entier (27 et 29 chunks). **Le piege evite** : 000d5950 porte 135 biped_pickup pour 493 s
de jeu ; « il existe un ramassage a moins d'une seconde » est vrai par hasard dans ~57 % des
cas, et « dans la fenetre d'image-cle de 20 s » l'est TOUJOURS. Le critere retenu exige donc
que l'evenement natif **nomme la meme arme** (`R(32)` == famille d'i43..i46), et chaque taux
est double d'un TEMOIN obtenu en decalant tous les ramassages de +37 s / -53 s / +91 s (on
garde le pire des trois).

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| biped_pickup / duree | 135 / 493 s | 73 / 522 s |
| emissions i43..i46 | 31 | 14 |
| **(a) prises i43..i46 retrouvees, ARME NOMMEE, a <= 500 ms** | **21 / 21 = 100 %** | **11 / 12 = 91,7 %** |
| temoin decale (pire des 3) | 1 / 21 = 4,8 % | 0 / 12 = 0,0 % |
| arrivees d'images-cles · expliquees par i43..i46 · TROU | 14 · 7 · **7** | 21 · 10 · **11** |
| **(b) trou NOMME par un biped_pickup** | **5 / 7 = 71,4 %** | 3 / 11 = 27,3 % |
| temoin decale (pire des 3) | 1 / 7 = 14,3 % | 1 / 11 = 9,1 % |
| (b') hors fenetres de reapparition (<= 2 arrivees simultanees) | 5 / 7 = 71,4 % | **3 / 3 = 100 %** |
| **(c) familles d'arme d'i43..i46 connues du type 9** | **10 / 10 = 100 %** | **11 / 11 = 100 %** |
| evenements portant une arme d'i43..i46 | 38 / 135 = 28,1 % | 24 / 73 = 32,9 % |

(b') est une lecture POSTERIEURE a la premiere mesure et publiee comme telle : le 27,3 % de
00502e52 est ecrase par UNE fenetre (slot 543) ou huit armes « arrivent » d'un coup — c'est une
reapparition dont l'arsenal complet est re-annonce, pas huit ramassages. Hors de cette fenetre,
3 trous sur 3 sont nommes.

**Les 7 arrivees lourdes du dossier sont bien celles-la** : Gravity Hammer x3, M41 SPNKr x3,
Stalker Rifle x1 sur 000d5950. Cinq sont nommees par un biped_pickup portant CETTE arme ; deux
ne le sont pas (un Gravity Hammer slot 529, le Stalker Rifle slot 557).

## LE R(3) EST UNE CLASSE D'OBJET — le resultat le plus net, reproduit

Table croisee `R(3)` x « l'identifiant est-il une famille d'arme connue d'i43..i46 ? » :

| R(3) | 000d5950 | 00502e52 |
|---|---|---|
| 0 | 43 evenements, **72,1 %** d'armes | 27 evenements, **63,0 %** d'armes |
| 1 | 10 evenements, **70,0 %** d'armes | 10 evenements, **70,0 %** d'armes |
| 2 | 51 evenements, **0,0 %** | 16 evenements, **0,0 %** |
| 3 | 31 evenements, **0,0 %** | 20 evenements, **0,0 %** |

**118 evenements de classe 2 ou 3 sur les deux films, ZERO arme.** Le champ separe donc les
ramassages d'ARME (classes 0 et 1) de ceux d'AUTRE CHOSE (classes 2 et 3 : equipement,
grenades, consommables — la rafle automatique en marchant du dossier `reference_pickup_
voluntary_vs_automatic`). Reponse a la question 4c : **oui, le type 9 couvre aussi
l'equipement et les grenades, et il les etiquette.**

## L'identifiant R(32) est un identifiant de CATALOGUE, pas un handle de partie

Sept valeurs au moins sont communes aux DEUX films (31913, 257365759, 2216347109, 2356674535,
2866415603, 3400195248, 4009088141). Un handle d'objet runtime ne se repeterait pas d'un match
a l'autre. Et les 10 (resp. 11) familles d'arme du canal i43..i46 sont TOUTES dans l'ensemble
des identifiants du type 9. Consequence : **le meme catalogue de production nomme deja ces
objets** — le nommage des ramassages d'arme est acquis sans travail supplementaire ; il reste a
nommer les identifiants de classe 2/3 (equipement, grenades), absents du catalogue armes.

## Ce qui N'EST PAS prouve, et les negatifs

- **(LOT 1, REFUTE PAR LE LOT 2 — conserve parce que l'erreur est instructive.)** Le lot 1
  concluait : « `ref0` n'est PAS identifiee ; 25 valeurs distinctes sur 50 evenements, donc ni
  un index de joueur ni un slot de bipede ». **C'etait faux**, et faux exactement comme sur
  `damage_aftermath` : je lisais l'INDEX BRUT sans la BASE, alors que le lecteur de l'exe
  reconstruit `(gen<<30) | (base + index)`. Voir « ref0 EST RESOLUE » ci-dessous. Lecon :
  devant une reference du modele M qui « ne ressemble a rien », ajouter la base AVANT de
  publier un negatif.
- **L'INSTANCE de l'objet ramasse n'est PAS dans l'evenement.** Le type 9 porte l'identifiant
  de CATALOGUE de l'objet (le R(32)), pas son handle monde : H-B mesuree et REFUTEE (voir plus
  bas). Consequence produit : le lien vers le SOCLE d'origine d'une prise reste l'affaire du
  canal spatial (schema 26) — l'evenement natif ne le donne pas.
- **Le residuel (~30 % du plafond) de trames non exactes n'est pas explique.** Il n'est pas du a la
  grammaire : aucun cadrage voisin ne fait mieux (0,0 % a -1, +1, +2, +3 sur les deux films),
  et le balayage de largeur donne un pic unique. Mais on reste a ~70 % du plafond `unit_zoom`,
  pas a 100 %. Le diagnostic des echecs (TestBipedPickupEchecs) montre que 0 % des paquets en
  echec sont sans aucun cadrage exact dans [10,120] — donc l'oracle trouve toujours QUELQUE
  chose ailleurs, et il n'est pas assez discriminant a l'echelle du paquet isole pour trancher
  ces cas. Piste : 5 des 12 echecs de 000d5950 tombent a -2 bits ; a n=5 ce n'est pas
  concluant (le plancher du temoin -2 vaut deja 7,3-10,0 % sur `unit_zoom`).
- **`biped_board_vehicle` (type 8) : ZERO occurrence** sur les deux films (0/135 et 0/73 des
  paquets 0xC4). Les arenes de ce corpus n'ont pas de vehicule ; le partage d'octet 0xC4 n'est
  donc jamais ambigu ici, mais il le deviendra sur un corpus BTB.
- **Nondeterminisme mesure** : le meme decodage rejoue donne 21 ou 23 trames exactes sur 35
  selon l'ordre des appels. Le decodeur de trame porte de l'etat global de process
  (`setAccumSlot`, accumulateurs i0) qui n'est pas remis a zero entre deux decodages. Sans
  effet sur les verdicts (les ecarts sont d'un ou deux paquets face a des ecarts de 60 points),
  mais a savoir avant d'ecrire un ratchet sur ces taux.
- La longueur 50 suppose que le `R(32)` optionnel de fin d'evenement du marcheur
  (`if FUN_14076cea8() && R(1)`) est ABSENT en mode film. Ce n'est pas lu dans l'exe, c'est
  DEDUIT : avec lui la longueur modale ne serait pas 50, et 50 est mesure sur 160/160
  evenements seuls des deux films, sans une seule exception.

## LOT 2 — `ref0` EST RESOLUE : c'est le RAMASSEUR, `slot = 512 + index`

**La verite terrain n'a pas ete refabriquee** : le lot 1 avait deja apparie, sans ambiguite,
des evenements type 9 a des emissions i43..i46 portant LA MEME arme a moins de 500 ms. Or une
emission i43..i46 est lue sur un record delta ANCRE : son slot de bipede est connu. Pour ces
paires, **le ramasseur est connu**, et c'est contre lui qu'on juge — pas contre un proxy.

Le juge n'est pas un taux de « liage » (leçon `damage_aftermath` : c'est un proxy faible) mais
la **correspondance EXACTE par evenement**. On ne balaye meme pas la base : on calcule l'ecart
`slot du ramasseur - index de ref0` paire par paire et on regarde son histogramme.

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| paires NON AMBIGUES de verite terrain | 21 (0 ecarte) | 11 (0 ecarte) |
| valeurs distinctes de `slot - index` | **1** | **1** |
| **base trouvee** | **512 sur 21/21 (100 %)** | **512 sur 11/11 (100 %)** |
| temoin (appariement permute d'un cran) | 16 valeurs, mode a 14,3 % | 9 valeurs, mode a 18,2 % |

**Une seule valeur distincte sur 32 paires, deux films, zero exception.** C'est la MEME base
que celle de la reference domaine 1 de `damage_aftermath` (~512, le debut de la plage des
bipedes). H-A est RETENUE.

**H-B (ref0 = l'objet ramasse) est REFUTEE** : `512 + ref0` egale le slot du RAMASSEUR sur
21/21 et 11/11 des paires, et un slot est unique — il ne peut pas designer a la fois le bipede
qui ramasse et l'objet ramasse.

### Les classes 2 et 3 (equipement, grenades) : meme base, verite terrain independante

Les paires ci-dessus sont toutes de classe R(3)=0 : elles ne prouvent la base que pour les
ramassages d'arme. Pour les classes 2 et 3, la verite terrain est ailleurs — le **canal
equipement i48** (`ScanFilmEquipmentChanges`), dont chaque emission est ancree sur un record
delta et porte donc le slot du bipede.

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| emissions d'equipement (vies) | 92 (77) | 82 (65) |
| evenements classe 2/3 ayant une emission a <= 500 ms | 26 | 13 |
| dont `512 + ref0` == **le slot emetteur** | **16 (61,5 %)** | **10 (76,9 %)** |
| temoins decales +37 / -53 / +91 s | **0,0 / 0,0 / 0,0 %** | **0,0 / 0,0 / 0,0 %** |
| CONTROLE classes 0/1 (armes) sur le meme canal | 30,8 % | 0,0 % |

**Combine : 26/39 = 66,7 %, contre 0,0 % sur les six temoins decales.** Et le controle positif
tranche l'objection « ce n'est que de la densite d'emissions » : les classes ARMES, mesurees
sur le MEME canal equipement avec la meme fenetre, tombent a 30,8 % et 0,0 %. L'appariement est
donc semantique. Les ~33 % de non-correspondance s'expliquent sans mystere : la fenetre de
500 ms admet aussi les emissions d'equipement des AUTRES joueurs, denses sur ces modes.

### Couverture — et pourquoi elle ne prouve rien toute seule

`512 + ref0` tombe dans la bande de bipedes pour **135/135 et 73/73 evenements, toutes classes
confondues (100 %)**. Mais le TEMOIN permute vaut lui aussi **100 %** : la bande fait ~100
slots contigus et un index quelconque y tombe toujours. **Cette mesure ne demontre rien**, elle
est publiee comme couverture et rien d'autre. Les juges sont les deux tables ci-dessus.

## Ce que ce chantier debloque pour le produit

La condition de reprise ecrite pour le son de ramassage est **levee sur QUI / QUOI / QUAND** :

- **QUAND** — l'evenement est date a la milliseconde du paquet (plus d'intervalle `[tLow, tHigh]`) ;
- **QUI** — `slot = 512 + ref0.index`, exact sur 32/32 paires de verite terrain. Ce slot est
  celui du canal i43..i46 (`HeldWeaponChange.Slot`), c'est-a-dire l'espace de slots que le
  pipeline de rejeu relie deja au joueur. **Reserve honnete** : la traversee slot -> xuid
  elle-meme n'a pas ete re-exercee dans ce lot, seule l'identite du slot est prouvee ;
- **QUOI** — l'identifiant de catalogue R(32) (100 % des familles d'arme d'i43..i46 connues) et
  la classe R(3) (armes 0/1 · equipement et grenades 2/3, separation a 0,0 % sur 118 evenements) ;
- **couverture** — 5/7 puis 3/3 des arrivees que le canal actuel rate, arme nommee, contre un
  plancher de hasard de 9-14 % ; et corroboration du canal actuel a 100 % / 91,7 % (temoin
  4,8 % / 0,0 %).

**Ce qui manque encore** : l'INSTANCE de l'objet, donc le SOCLE d'origine d'une prise. H-B est
refutee ; l'evenement ne porte pas de handle monde. Ce lien-la reste l'affaire du canal spatial
(schema 26). Autrement dit : « qui a ramasse quoi, quand » est acquis ; « depuis quel socle »
ne l'est pas et ne le sera pas par cet evenement.

## LOT 3 — LA PUBLICATION (schema 29)

Le canal est en production. Chemin : `filmdec.ScanFilmBipedPickups` (decodeur autonome, PAS un
hook — la liste d'evenements vit AVANT la trame, ces bits ne sont lus par aucun autre
consommateur) -> `replay.buildPickups` -> `doc.pickups`.

**Gate du portage** : accord PARFAIT production <-> instrument de recherche, evenement par
evenement (horodatage, slot, identifiant, classe), 135/135 et 73/73, zero rejet.

**Gate de la traversee slot -> xuid** (la reserve du lot 2, levee) : ramasseurs NOMMES
**127/135 = 94,1 %** et **73/73 = 100 %** · **0 collision de slot** sur les deux films — c'est
l'objection « les slots sont reattribues entre manches », mesuree et non supposee · 21/21 et
11/11 paires a slot identique des deux cotes. HONNETETE : les deux canaux rendant le MEME slot,
« meme xuid » en decoule PAR CONSTRUCTION ; ce n'est pas une mesure independante.

**Decision produit tranchee sur mesure — publier les classes non-arme.** Critere ecrit avant :
gain >= 30 %. Mesure : **80,5 % et 72,2 %** des non-armes n'ont AUCUNE emission i48 du MEME slot
a moins de 500 ms (temoin decale a 0,0 %). Elles comblent un trou, elles ne doublonnent pas
`equipmentChanges`.

**Le son.** Le son designe `168832f6` etait DEJA cable depuis le 2026-08-30 (schema 25) : la
condition de reprise etait deja levee, le brief du lot partait d'une premisse perimee, et aucune
re-livraison de `.wem` n'a ete necessaire. Ce qui restait vrai : les prises que `weaponChanges`
RATE etaient MUETTES. Regle posee — un ramassage natif d'arme sonne SI ET SEULEMENT SI aucun
`taken`/`swapped` ne le couvre (meme vie, meme famille, <= 5 frames). Mesure sur l'artefact
cuit : **32 sons ajoutes, 21 dedupliques, zero doublon**.

### La cuisson pilote, et ce qu'elle a montre

UNE invocation, en avant-plan, sur `000d5950`. Le binaire `cmd/replay-build` n'armait PAS le
plafond memoire — seul des trois binaires qui decodent un film a ne pas le faire. Il l'arme
desormais (`filmproc.Arm`, defaut 3 GiB, `--mem-gib 0` pour desarmer). **Pic mesure : 0,127 GiB.**

| verification | resultat |
|---|---|
| schema de l'artefact | **29** |
| ramassages publies | **135** (decodes 135, refuses 0) |
| ramasseurs nommes | **127 / 135 = 94,1 %** — identique a la mesure du lot 2 |
| armes / objets | 53 / 82 · classes 0:43 · 1:10 · 2:51 · 3:31 |
| identifiant du 1er ramassage | `00007ca9` = 31913, la valeur decodee a la main au lot 1 |
| sons ajoutes par le canal natif | **32** (21 dedupliques) |
| pic memoire | **0,127 GiB** (plafond 3 GiB) |

**NEGATIF DE LA CUISSON, ET IL COMPTE : la datation des `padPickups` n'a PAS ete exercee.** Ce
film (Super Fiesta sur Cliffhanger) rend **0 socle et 0 occupation** — ses 220 armes au sol sont
toutes des armes LACHEES, aucune ne s'agglomere en socle. `coverage.padDating` vaut donc
`{occupations:0, dated:0, ...}`. La levee de l'intervalle de vingt secondes est **prouvee par
trois tests unitaires** (dont un qui verifie qu'un film sans canal natif rend exactement les
`padPickups` du schema 28) mais **PAS par une cuisson**. Il faut un film a socles — un CTF Arena
— pour la voir a l'oeuvre. C'est la seule promesse du lot 3 qui reste non verifiee sur donnee
reelle.

### Ce que les gardes ont attrape avant moi

Quatre garde-fous du depot ont refuse le lot avant d'etre satisfaits, et c'est leur role :
le cliquet de `SchemaVersion` (raison ecrite exigee), le `contracttest` (quatre champs publies
par le Go et absents du contrat), les deux gardes de frontiere TypeScript (tableaux nullables,
racine et chemins profonds) et le garde-rail de parite du numero de schema cote web.

## REVUE ADVERSARIALE — ronde 1 (2026-09-01, deux relecteurs frais)

Sept constats recevables : **1 P0**, **4 P1**, **1 mecanique**, **4 P2 consignes sans
correction**. Tous les correctifs sont dans le meme worktree, gates repasses.

### P0 — LA DATATION DES `padPickups` ETAIT MORTE, ET LA CUISSON NE POUVAIT PAS LE MONTRER

Le defaut : `Pickup.W` s'ecrit `fmt.Sprintf("%08x", …)` — huit hexa MINUSCULES sans prefixe —
tandis que `WeaponPad.Weapon` sort de `formatWeaponFamily`, qui ecrit `"0x"` + huit
MAJUSCULES (et un NOM CANONIQUE pour les socles de power-up). **Les deux espaces ne coincident
jamais** : `hits` etait toujours vide, aucun `padPickups[].t` n'a jamais ete ecrit, aucun `xuid`
pose — et `coverage.padDating` publiait `{dated:0, uncovered:N}` **qui se lisait comme une
mesure de corpus alors que c'etait un defaut de format**. Verifie sur pieces avant correction :
l'artefact cuit porte `pickups[0].w = "00007ca9"` face a `loadouts[].w = "0x767DB96D"`.

**Pourquoi la cuisson pilote ne l'a pas revele** : son film ne porte AUCUN socle (0 occupation).
Le negatif publie au lot 3 (« la datation n'a pas ete exercee ») etait donc exact — et il
masquait un bogue, pas seulement une lacune de couverture. La lecon tient en une ligne : un
canal non exerce par la cuisson doit etre traite comme NON VERIFIE, jamais comme probablement bon.

**Correctif** : la normalisation se fait AU POINT DE JOINTURE (`padFamilyKey`), et nulle part
ailleurs. Les formes publiees ne bougent PAS — `weaponChanges[].w` s'ecrit ainsi depuis le
schema 25 et des clients peuvent deja le lire ; changer un contrat public pour un confort prive
aurait ete le mauvais echange. Les socles de POWER-UP (nom canonique) sont structurellement non
joignables : ils sont comptes a part (`PowerupPads`) au lieu d'etre noyes dans `Uncovered`, qui
laisserait croire que le canal a cherche et n'a pas trouve.

Le commentaire de `Pickup.W` qui affirmait « meme convention que `Loadout.W` » est corrige :
il etait faux de moitie, et c'est cette moitie qui a produit le bogue.

### P1 — quatre trous de preuve, tous combles avec leur inversion

| # | constat | correctif | inversion qui le prouve |
|---|---|---|---|
| a | le test de `datePadPickups` employait `"11223344"` des DEUX cotes — forme que la production ne produit jamais : vert AVEC et SANS le P0 | formes derivees de `formatWeaponFamily` et de `%08x`, plus un cas power-up | normalisation neutralisee -> **« LA JOINTURE EST MORTE »**, 3 tests tombent |
| b | AUCUN test CI ne couvrait le decodeur (garde `BIPED_PICKUP_FILM` -> Skip permanent) | fixture SYNTHETIQUE en memoire : paquet 0xC4 forge, type 8, type inconnu, troncatures, liste multiple | base 512 -> 0 : **1 test tombe** · classe et porte du catalogue inversees : **4 tests tombent** |
| c | dix endroits affirmaient encore « `xuid` vaut TOUJOURS `null` (79,7 % contre 90 %) » | balayage complet, formulation unique et datee « publie depuis le schema 29 » | — |
| d | la cle de FAMILLE de la dedup sonore n'etait testee nulle part (meme `w` des deux cotes partout) | cas « arme A puis arme B <= 500 ms » + bornes 5 et 6 frames | cle de famille retiree : **1 test tombe** · fenetre 5 -> 50 : **2 tests tombent** |

**Deux surprises pendant P1-b, et les tests ont eu raison contre moi.** (1) J'attendais
`RefusedNoRef` sur un payload tronque a deux octets : c'est `RefusedNoCatalog` — a cette
longueur la porte de ref0 est encore un vrai bit. (2) Une troncature de QUEUE est
**indetectable** : le bourrage se lisant a zero, un payload coupe apres le bit 25 rend un
evenement parfaitement decodable dont seul l'identifiant est faux. Le decodeur n'a pas a s'en
apercevoir — l'autorite sur la longueur d'un paquet est `FilmPacket.Size`, pas une heuristique
de contenu. Les deux proprietes sont ecrites dans le test au lieu d'etre supposees.

**Onzieme site trouve pendant le balayage P1-c**, hors liste : le golden lui-meme rendait la
phrase « avec un ramasseur nomme (l oracle ne le permet pas : 79,7 % contre 90 % exige) », et un
garde-rail de phrases l'exigeait. Corrige aussi — laisser une phrase sciemment fausse dans le
golden aurait ete l'anti-pattern qu'on corrigeait. Diff golden : **une ligne**.

### Mecanique

`NATIVE_PICKUP_MATCH_FRAMES` et `nativePickupsNotAlreadyHeard` etaient exportes sans aucun
importeur : de-exportes. `npx knip` : exit 0, aucun export mort signale.

## Decouvertes consignees, NON traitees (P2)

1. **Jointure `weaponLabels` impossible** pour `pickups[].w` ET `weaponChanges[].w` : la table
   de libelles est indexee sur la forme `formatWeaponFamily` (`"0x"` + majuscules), pas sur
   `%08x`. Anterieur au lot pour `weaponChanges` (schema 25). Le nommage cote client sera
   l'affaire du lot equipement.
2. **`ReadFilmChunk` en erreur -> `continue` sans log** (`biped_pickups.go`) : convention
   partagee par cinq balayeurs voisins. Dette de maison, a traiter en une passe ou pas du tout.
3. **Aucun garde ne consulte `own.SlotCollisions` avant publication** — anterieur au lot ;
   mesure a 0 collision sur les deux films, mais rien n'empeche une publication si ca changeait.
4. **Le golden texte n'affiche ni `pickups`, ni `weaponChanges`, ni `equipmentChanges`** :
   `goldenInputs.options()` ne les transmet pas. Les trois canaux vivent en production sans
   couverture de golden. Deja consigne au lot 3.

## Reproduire

```
cd apps/go-api
CGO_ENABLED=0 BIPED_PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/filmdec/ -run BipedPickup -v -timeout 30m
```

Un film par process (bombe RAM avouee du corpus), lecture seule, verrou `LockProcessDecode`.
