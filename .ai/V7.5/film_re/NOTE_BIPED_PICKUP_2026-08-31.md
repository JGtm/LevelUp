# biped_pickup (type 9) EST DECODE : le ramassage est nomme, date et attribue

Date : 2026-08-31. Chantier RAMASSAGE, worktree `wt/biped-pickup`. Instruments :
`apps/go-api/internal/analysis/filmdec/biped_pickup_{research,grammaire,confront}_test.go`,
sous garde `BIPED_PICKUP_FILM` (sautes sans elle, aucun effet en CI, aucune publication
production dans ce lot).

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
   ref0 : R(1) porte ; si 1 : R(8) index + R(2) gen      <- domaine 2  (LE RAMASSEUR ?)
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

- **`ref0` n'est PAS identifie.** C'est une reference du domaine 2, R(8), presente a 100 %,
  mais elle prenait deja **25 valeurs distinctes sur 50 evenements** dans la fenetre temoin de
  000d5950 (17 sur 34 pour 00502e52) : ce n'est donc PAS un index de joueur (8 joueurs dans
  ces parties). Ce n'est pas
  non plus un slot de bipede (ceux-ci vivent vers 512-615, domaine 1). L'hypothese « ref0 = le
  ramasseur » est **NON VERIFIEE** ; le lien ramasseur -> joueur reste a faire. C'est le
  travail restant pour lever la limite « prises sur socle sans xuid ».
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

## Ce que ce lot debloque pour le produit

La condition de reprise ecrite pour le son de ramassage est **levee sur la moitie du sujet** :

- l'evenement natif est **date a la milliseconde du paquet** (plus d'intervalle `[tLow, tHigh]`) ;
- il **nomme l'objet** au catalogue de production pour les armes (100 % des familles), et il
  **etiquette** les ramassages non-arme (classes R(3) 2 et 3) ;
- il **couvre le trou de rappel** d'i43..i46 : 5/7 puis 3/3 des arrivees que le canal actuel
  rate, arme nommee, contre un plancher de hasard de 9-14 % ;
- il **corrobore** le canal actuel a 100 % / 91,7 % (temoin 4,8 % / 0,0 %) : les deux canaux
  voient la meme chose quand ils voient tous les deux.

**Ce qui manque pour publier** : l'attribution au JOUEUR. `ref0` n'est pas resolue, et sans
elle un ramassage est date et nomme mais anonyme. Prochaine etape naturelle : identifier le
domaine 2 (correlation de `ref0` avec les fils de vie, avec les slots bipedes du meme paquet,
et avec le pont slot -> xuid existant de `killsource`), sur le modele de ce qui a resolu la
reference du domaine 1 de `damage_aftermath` par l'ajout de la base de plage.

## Reproduire

```
cd apps/go-api
CGO_ENABLED=0 BIPED_PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/filmdec/ -run BipedPickup -v -timeout 30m
```

Un film par process (bombe RAM avouee du corpus), lecture seule, verrou `LockProcessDecode`.
