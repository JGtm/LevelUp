# RAPPORT — lot V6 : MARCHER LA LISTE D'EVENEMENTS, ET LA MACHINE D'ETATS D'OCCUPATION

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB. Mesures en AVANT-PLAN, `CGO_ENABLED=0`,
> GOCACHE isole (`scratchpad/gocache_v6*`), donnees reelles du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache`, LECTURE SEULE). Ghidra : MORT — tout ce
> rapport est de la MESURE.

---

## 0. LE RESULTAT EN SIX LIGNES

1. **LE RATIO board:exit DE 1:15 N'EST PAS UN DEFAUT DE DECODAGE. C'EST LE FILM.** La premisse
   fondatrice du plan revise du lot V5B — « notre decodeur ne lit que l'evenement de tete et jette
   la suite, donc les embarquements manquants sont dans la suite » — est **REFUTEE, avec temoin**.
   Sur 296 595 paquets delta de 8 films et 400 decalages de bit chacun (1,2 x 10^8 essais), un
   balayage par SIGNATURE COMPLETE trouve 1 894 candidats hors tete ; le meme balayage decale du
   corps de +1, +2, +3, +4 bits en trouve 4 246, 1 539, 3 400, 2 881 (moyenne 3 017). **Le reel est
   SOUS la moyenne de ses temoins** : il n'y a rien a trouver hors de la tete.
2. **LA GRAMMAIRE D'ENCHAINEMENT EST ETABLIE, ET ELLE DIT QUE LA LISTE SE TERMINE.** Le bit qui
   suit immediatement la charge d'un evenement vehicule de tete vaut **0 (fin de liste) sur 99/100
   instances** ; les temoins a -1 et +1 bit rendent 5 % et 0 %. Derriere ce zero il reste 1 300 a
   2 800 bits : la trame ECS. Quand un paquet porte un evenement vehicule, cet evenement est SEUL.
3. **LA LONGUEUR D'UN EVENEMENT EST UNE CONSTANTE PAR TYPE, et elle est mesuree au bit pres pour
   les deux types vehicule** : sortie **42 bits** (trame a 43, sur **307/307** paquets de 40 films),
   embarquement **52 bits** (trame a 53, sur **14/14**). Les deux se decomposent exactement :
   sortie 9 + 27 + 6 avec `27 = 13 + 13 + 1` (deux refs de domaine 1 avec sonde, la troisieme
   ABSENTE) ; embarquement 9 + 37 + 6 avec `37 = 11 + 10 + 16` (les TROIS refs presentes).
4. **L'ANCRAGE AVAL EST REFUTE COMME ORACLE DE LONGUEUR** — et c'est ce qui ferme la marche
   sequentielle des types inconnus. Le critere « la trame se termine proprement » est vrai dans
   **0,0 %** des cas AU VRAI DEBUT (34 538 paquets a liste vide, verite terrain connue). Le score de
   repli — profondeur de marche — ne designe le vrai debut que dans **23,3 %** des cas par paquet, et
   AGREGE sur 250 sorties il pointe **S = 47 au lieu de 43** (rang 4). Le decodeur ECS n'est pas
   bit-exact, donc il ne peut pas servir de regle a mesurer.
5. **LA REFERENCE 2 DE L'EMBARQUEMENT N'EST PAS LE VEHICULE** (hypothese ouverte depuis le
   2026-09-02, ici tranchee). Elle est bien PRESENTE 22/22, mais sa valeur ne tombe dans la bande
   de slots `ti=40` dans **0/22** des cas, ni dans celle des armes au sol (0/22), et elle ne prend
   que **QUATRE valeurs distinctes** sur 22 instances de 8 films (180, 116, 244, 208).
6. **LA MACHINE D'ETATS EST LIVREE ET BRANCHEE**, pilotee par l'OCCUPANT et non plus par le trou de
   position, avec le SILENCE TERMINAL enfin attribuable (5 sur 49 episodes de 5 films). Le rayon
   d'ancre est au § 2.3, les artefacts avant/apres au § 5 : **12 -> 13** episodes sur `0d76e8f1`,
   **2 -> 3** sur `fccc61cd`. Le gain est REEL mais MODESTE, et la raison est le point 1 : le film
   n'atteste que 11 et 3 trajets sur ces deux films. La production etait deja au plafond, et ce lot
   demontre que le plafond est celui du FILM.

---

## 1. MISSION A — LA GRAMMAIRE D'ENCHAINEMENT

### 1.1 Le modele, et le seul point qui manquait

Le modele de paquet, porte le 2026-09-01 :

```
[1 bit config] [ ( 1 [R(7) type] [3 refs gardees] [charge] )* 0 ] [trame de records ECS]
```

Le cadrage de la TETE est prouve bit-exact (garde-fou `fire_events == head type36`, rejoue vert).
Ce qui manquait etait la LONGUEUR DE CHARGE par type : sans elle, on ne peut pas avancer le
lecteur jusqu'a l'evenement suivant. Le lot V6 attaque le probleme par les deux bouts.

### 1.2 Le bout connu : ce qui suit un evenement vehicule (`TestV6Chaine`)

L'embarquement et la sortie sont les DEUX SEULS types dont la charge soit connue au bit pres
(`R(6)` siege). On peut donc lire le bit qui suit leur charge — c'est, par construction, le bit de
continuation du deuxieme evenement de la liste.

Mesure sur 12 films, 100 tetes board/exit :

| grandeur | reel | temoin -1 bit | temoin +1 bit |
|---|---|---|---|
| bit de continuation pose | **1 / 100 = 1,0 %** | 5 / 100 | 0 / 100 |
| types lus au bon decalage | `33 x 1` | `10x2 18x1 28x1 80x1` | (aucun) |

**LECTURE : la liste se TERMINE juste apres l'evenement vehicule dans 99 % des cas.** Et derriere
ce zero il reste beaucoup de bits — la distribution des bits restants apres la charge est etalee de
1 310 a 2 790 (aucune valeur ne se repete plus de 3 fois) : c'est la trame ECS, pas un evenement.

### 1.3 Le registre des types de tete (30 types, 12 films)

Comptes de l'evenement de TETE, tous types confondus. Les types NON DECODES sont dits tels quels —
c'est le registre demande par la mission.

| type | n | statut |
|---|---|---|
| 36 | 19 409 | tir / degat — **decode** (`fire_events.go`, offsets fixes) |
| 0 | 12 770 | non decode (candidat « degats ») |
| 82 | 7 812 | non decode (`PlayerGameEvent` d'apres le catalogue Notion) |
| 5 | 4 841 | non decode |
| 15 | 3 927 | non decode |
| 21 | 3 577 | non decode (candidat « lunette ») |
| 38 | 3 430 | non decode (candidat « recharge ») |
| 7 | 1 953 | non decode |
| 9 | 1 882 | non decode (candidat « ramassage ») |
| 75 | 1 696 | non decode |
| 76 | 1 474 | non decode |
| 1 | 1 120 | non decode |
| 39 | 952 | non decode |
| 6 | 923 | non decode |
| 103 | 268 | non decode |
| 11 | 135 | non decode |
| 109 | 117 | non decode |
| **22** | **78** | **SORTIE — decodee, longueur 42 bits** |
| 47 | 25 | non decode |
| 23 | 24 | non decode |
| **8** | **22** | **EMBARQUEMENT — decode, longueur 52 bits** |
| 117 | 8 | non decode |
| 58 | 7 | non decode |
| 93 | 5 | non decode |
| 100 | 5 | non decode |
| 41 | 3 | non decode |
| 105 | 3 | non decode |
| 3 | 2 | non decode |
| 40 | 2 | non decode |
| 106 | 2 | non decode |

Aucun type >= 123 n'apparait — coherent avec la garde `type < 0x7b` du dispatcher, et c'est un
controle gratuit du cadrage.

### 1.4 La longueur d'un evenement est une CONSTANTE par type (`TestV6Longueur`)

C'est le resultat structurel du lot. Pour un type donne, la configuration des trois bits de garde
des references ne varie pas d'un paquet a l'autre :

| type | debut de trame mesure | instances | decomposition |
|---|---|---|---|
| **22** (sortie) | **43** sur **307/307** (40 films) | 307 | `9 + [13+13+1] + 6 + 1` |
| **8** (embarquement) | **53** sur **14/14** | 14 | `9 + [11+10+16] + 6 + 1` |

`13 = 1 garde + 1 sonde + 9 index + 2 generation` (domaine 1) · `1` = ref absente ·
`11 = 1+8+2` (domaine 2) · `10 = 1+7+2` (domaine 3) · `16 = 1+13+2` (domaine 7).

Les largeurs employees sont EXACTEMENT celles portees le 2026-09-02 (Ghidra pour les domaines,
mesure pour les largeurs) : **la somme tombe juste sans qu'aucun parametre n'ait ete ajuste ici.**
C'est un recoupement independant de ce portage.

### 1.5 L'ANCRAGE AVAL — refute, avec ses chiffres (`TestV6Ancrage`, `TestV6Longueur`)

La mission proposait d'exploiter le fait que la trame ECS suivante se recale, pour deduire la
longueur des types inconnus. Deux scores ont ete essayes, et les deux echouent.

**Score 1 — « fin propre »** (la boucle de records se termine sur un record de type 0 et il ne
reste qu'un bourrage nul de moins de 8 bits). Etalonne sur une VERITE TERRAIN :

| population | n | fin propre AU VRAI DEBUT |
|---|---|---|
| paquets a LISTE VIDE (vrai debut = bit 2) | 34 538 | **1 = 0,0 %** |
| paquets a tete vehicule (vrai debut = fin + 1) | 11 | **0 = 0,0 %** |

Le decodeur ECS ne termine JAMAIS proprement, meme la ou le debut est connu. Le critere est mort.

**Score 2 — profondeur de marche** (nombre de records consecutifs dont la traversee aboutit).
Etalonne sur 356 paquets a liste vide, balayage de +/- 64 bits :

| grandeur | mesure |
|---|---|
| profondeur au vrai debut | `0x12 1x7 2x164 3x117 4x32 5x19 6x3 7x2` |
| vrai debut GAGNANT UNIQUE | **83 / 356 = 23,3 %** |
| vrai debut gagnant EX AEQUO | 125 / 356 = 35,1 % |

23,3 % ne suffit pas par paquet. **Agrege** sur 250 sorties (le type dont la reponse est connue) :

| score | top 5 des candidats | vrai debut (43) |
|---|---|---|
| SOMME des profondeurs | **47** (495), 34 (360), 3 (335), **43** (309), 2 (302) | rang **4** |
| VOTE de pluralite | **47** (32), 46 (9), 3 (8), **43** (7), 65 (7) | rang **4** |

**Le score est BIAISE, pas seulement bruite** : il converge sur 47, a 4 bits du vrai. Agreger
davantage ne le corrigera pas. **L'ancrage aval ne peut pas mesurer une longueur d'evenement tant
que le decodeur ECS n'est pas bit-exact** — et le depot dit deja qu'il ne l'est pas (77 a 80 % de
traversees abouties, `delta_walk_witness_test.go`).

### 1.6 LA QUESTION EST NEANMOINS TRANCHEE — par la signature (`TestV6Marche`)

La marche sequentielle est fermee, mais **la question qu'elle devait resoudre ne l'est pas** : il
existe un chemin qui ne demande AUCUNE longueur.

Le § 1.2 etablit que, quand un evenement vehicule est dans la liste, IL EN EST LE DERNIER. Son
cadrage complet est donc contraint de bout en bout :

```
board : [cont=1][R(7)=8]  [g=1][8][2] [g=1][7][2] [g=1][13][2] [R(6) siege<8] [fin=0]
exit  : [cont=1][R(7)=22] [g=1][s=1][9][2] [g=1][s=1][9][2] [g=0] [R(6) siege<8] [fin=0]
```

soit 15 bits (board) et 17 bits (exit) fixes, plus l'appartenance de l'occupant a la bande bipede.
On balaie donc TOUS les decalages de 1 a 400 dans TOUS les paquets delta, et on compte.

**Le temoin est le decalage du CORPS de l'evenement de +1 a +4 bits**, ecrit avant la mesure.

Mesure, 8 films, 296 595 paquets delta, 400 decalages chacun :

| population | candidats | dont siege = 0 |
|---|---|---|
| **TETE** (bit 1) | board **19** · exit **70** | — |
| **HORS TETE, reel** | board **1 738** · exit **156** (total **1 894**) | **848** |
| TEMOIN corps +1 bit | **4 246** | 2 698 |
| TEMOIN corps +2 bits | **1 539** | 854 |
| TEMOIN corps +3 bits | **3 400** | 1 597 |
| TEMOIN corps +4 bits | **2 881** | 1 765 |
| **moyenne des 4 temoins** | **3 017** | **1 728** |

**LE REEL (1 894) EST SOUS LA MOYENNE DE SES TEMOINS (3 017). L'EXCES EST NUL, ET MEME NEGATIF.**
Le compte attendu par pur hasard le confirme independamment : 15 bits contraints pour le board plus
une bande de ~100 slots sur 256 valeurs donnent ~1/84 000, soit **~350 faux positifs par film-lot**
de ce volume ; on en observe 1 738 sur 8 films, exactement l'ordre de grandeur. Les sieges des
candidats hors tete sont UNIFORMES sur 0..7 (`0x848 4x202 2x186 3x172 1x169 6x160 7x81 5x76`), alors
que les vrais evenements valent 0 dans 94 a 95 % des cas : la signature du bruit, pas celle du jeu.

**CONCLUSION MISSION A.** Il n'y a pas d'embarquement cache dans la liste. Le ratio board:exit de
1:15 est une propriete du FILM : le moteur n'emet `biped_board_vehicle` que pour un sous-ensemble
des montees en vehicule. L'axiome de l'utilisateur (« le Theatre nomme conducteur/passager/artilleur
a tout instant ») reste vrai a l'ecran, mais **cette information ne transite pas par la liste
d'evenements** — et le lot V5B avait deja montre qu'elle n'est pas non plus dans la longueur du
record d'image-cle.

### 1.7 La reference 2 de l'embarquement (`TestV6Ref2`) — ce n'est pas le vehicule

Le § 1.4 fait apparaitre que les TROIS references de l'embarquement sont PRESENTES (37 bits), alors
que la sortie n'en porte que deux. La ref 2 est en domaine 7 — celui des objets du monde : candidate
naturelle pour LE VEHICULE, que le chantier resout jusqu'ici par la position.

Mesure, 22 embarquements de 8 films :

| grandeur | mesure |
|---|---|
| refs presentes | r0 **22/22** · r1 **22/22** · r2 **22/22** |
| fin d'evenement au bit 52 | **22/22 = 100 %** |
| occupant (r0) en bande bipede | **22/22 = 100 %** |
| siege = 0 | 21/22 = 95,5 % (puis un 61) |
| **ref2 dans la bande VEHICULE (`ti=40`)** | **0/22 = 0,0 %** |
| TEMOIN a) meme ref lue a +1 bit, bande vehicule | 0/22 |
| TEMOIN b) ref2 dans la bande ARMES AU SOL (`ti=42`) | 0/22 |
| valeurs de r2 | **180 x10 · 116 x7 · 244 x4 · 208 x1** |
| valeurs de r1 (domaine 3) | 3 x8 · 5 x5 · 0 x3 · 2 x3 · 7 x3 |

**QUATRE valeurs distinctes sur 22 instances de 8 films** : ce n'est pas une instance d'objet, c'est
un identifiant de DEFINITION (ou une base de domaine non nulle qu'on ne sait pas lire). L'hypothese
« ref2 = le vehicule » est donc **REFUTEE en l'etat**, et la geometrie reste la methode. C'est ecrit
dans l'en-tete de `vehicle_rides_events.go`, avec ces chiffres.

---

## 2. MISSION B — LA MACHINE D'ETATS D'OCCUPATION

### 2.1 Ce qui a change

Jusqu'ici l'episode d'occupation etait pilote par le TROU du flux de position ; les evenements
n'affinaient que ses bornes. Le lot V6 inverse : **la machine est PAR OCCUPANT, et les evenements
en sont les bords.**

```
EMBARQUEMENT -> ouvre un episode (un second embarquement sans sortie ferme le precedent)
SORTIE       -> ferme l'episode ouvert ; s'il n'y en a pas, elle en ouvre un dont le DEBUT est
                le dernier point replique par l'occupant avant elle
FIN DE LISTE avec un episode ouvert -> ferme a la REAPPARITION de l'occupant, et s'il ne
                reapparait JAMAIS, a la fin de vie du vehicule (SILENCE TERMINAL)
LE TROU reste, EN REPLI : un trou qu'aucun episode PUBLIE ne recouvre produit toujours un
                episode, aux memes portes qu'avant (3 s, 1,5 m, fraicheur 1 s)
```

Le vehicule est resolu par la POSITION, avec **DEUX ancres** au lieu d'une : le dernier point
avant le debut, puis (repli) le premier point apres la fin.

### 2.2 Deux pieges rencontres, et corriges sur mesure

**PIEGE 1 — la regle anti-doublon naive FAIT PERDRE des episodes.** Premiere version : un trou
recouvert par un episode d'evenement etait supprime. Mais un episode d'evenement dont le vehicule
n'est PAS resolu n'est pas publie — et il supprimait quand meme le trou qui, lui, savait le
rattacher. Mesure : **12 -> 11 episodes sur `0d76e8f1`**, une REGRESSION. Corrige : seuls les
episodes REELLEMENT PUBLIES ferment la porte du repli.

**PIEGE 2 — le silence terminal n'est pas toujours terminal.** Un occupant qui MEURT A BORD
respawne, donc RE-EMET une position ; ferme a la fin de vie du vehicule, son episode le montrerait
au volant longtemps apres sa mort. La borne a donc ete posee : un episode ouvert se ferme a la
premiere REAPPARITION de l'occupant, et la fin de vie du vehicule n'est employee que s'il ne
reapparait JAMAIS.

**ET LA MESURE DIT QUE CE CAS NE S'EST PAS PRODUIT SUR LES DEUX FILMS DE DEMONSTRATION** — le
compte d'episodes est IDENTIQUE avec et sans la borne. L'unique silence terminal de `0d76e8f1`
(slot 561, 906 frames = 90,6 s) ne reapparait vraiment jamais : ces 90 s sont la bonne reponse, pas
un defaut. La borne est gardee parce que la SEMANTIQUE l'exige, et elle est couverte par un test
sur fixtures — pas parce qu'un chiffre l'a exigee. C'est dit pour qu'on ne relise pas ce lot comme
ayant corrige un bug qu'il n'a pas observe.

### 2.3 LE RAYON DE L'ANCRE D'EVENEMENT — la seule constante nouvelle, et sa mesure

Le lot V4 avait REFUTE d'ouvrir le rayon du TROU (1,5 m). **La population n'est pas la meme** : le
rayon du trou DECIDE s'il y a eu embarquement, celui de l'ancre d'evenement ne choisit que QUEL
vehicule — l'evenement a deja prouve l'embarquement. La mesure a donc ete refaite sur cette
population-la (`TestV6EtatsOccupation`, 5 films, **49 episodes attestes**) :

| rayon | rattaches | AMBIGUS (un 2e vehicule sous le rayon) | TEMOIN +60 s |
|---|---|---|---|
| 1,5 m | 39 / 49 | 1 | 1 |
| 2,0 m | 46 / 49 | 1 | 1 |
| **3,0 m** | **48 / 49** | **1** | **1** |
| 5,0 m | 48 / 49 | 2 | 2 |
| 8,0 m | 48 / 49 | 3 | 3 |
| 12,0 m | 49 / 49 | 4 | 3 |

**3 m est le dernier rayon qui ne coute RIEN** : +9 episodes rattaches, ambiguite et temoin
STRICTEMENT inchanges. Des 5 m, chaque episode gagne (zero) se paie. `vehicleEventAnchorRadiusM = 3.0`,
et le tableau est dans le commentaire de la constante.

Distances mesurees, episode par episode (extrait de `0d76e8f1`) — le contraste est le resultat :

```
slot 512 sortie   · debut 0,9 m · fin 1,0 m · TEMOIN 57,1 m
slot 531 sortie   · debut 0,5 m · fin 0,7 m · TEMOIN 38,3 m
slot 561 TERMINAL · debut 1,7 m · fin  -    · TEMOIN 66,2 m
slot 602 sortie   · debut 1,2 m · fin 2,0 m · TEMOIN aucun vehicule frais
```

### 2.4 Bilan de la machine, film par film (`TestV6EtatsOccupation`, rayon 1,5 m au moment de la mesure)

| film | evenements en bande | episodes construits | deux bords | sortie seule | SILENCE TERMINAL |
|---|---|---|---|---|---|
| `0d76e8f1` | 1 board + 10 exits | 11 | 0 | 10 | **1** |
| `21468645` | 2 boards + 9 exits | 9 | **2** | 7 | 0 |
| `4898d586` | 2 boards + 16 exits | 18 | 0 | 16 | **2** |
| `a89a3d23` | 2 boards + 6 exits | 8 | 0 | 6 | **2** |
| `fccc61cd` | 0 board + 3 exits | 3 | 0 | 3 | 0 |
| **total** | **7 boards + 44 exits** | **49** | **2** | **42** | **5** |

**Cinq silences terminaux sur 49 episodes** : c'est exactement ce que l'ancien pilotage ne pouvait
pas attribuer, et c'est desormais fait. Le TEMOIN de rattachement (+60 s) vaut **0** partout sauf
un episode sur `21468645`.

---

## 3. CE QUI EST LIVRE

| fichier | etat | role |
|---|---|---|
| `internal/analysis/replay/vehicle_rides_events.go` | **NEUF** (production) | la machine d'etats par occupant : `vehicleEpisode`, `vehicleEventEpisodes`, `mergeVehicleEvents`, `vehicleEpisodesOfOccupant`, `vehicleEpisodeCovers`, `vehicleRideFromEpisode`, `vehicleAnchorAt`, `vehicleLifeForAnchor`, `vehicleEventAnchorRadiusM` |
| `internal/analysis/replay/vehicle_rides.go` | **MODIFIE** | `buildVehicleRides` consomme la machine d'abord, le trou en repli ; `vehicleNearestWithin` (rayon parametre, UNE seule implementation, `vehicleNearestTo` l'appelle) |
| `internal/analysis/replay/vehicle_rides_events_test.go` | **NEUF** (garde-rail SANS env) | 6 tests sur fixtures : les quatre formes d'episode, le siege de la sortie qui prime, l'ordre a instant egal, la regle anti-doublon, les deux ancres AVEC TEMOIN hors rayon, la reapparition qui ferme un episode ouvert |
| `internal/analysis/replay/vehicules_v6_etats_test.go` | **NEUF** (instrument, garde `V4_ROOT`/`V4_FILMS`) | le bilan par film, les distances aux deux ancres, la table par rayon avec ambiguite et temoin |
| `internal/analysis/filmdec/vehicules_v6_{chaine,ancrage,longueur,marche,ref2}_test.go` | **NEUFS** (instruments, garde `V6_ROOT`/`V6_FILMS`) | les cinq mesures de la mission A |

**Aucun changement de contrat** : mêmes champs, `SchemaVersion` reste **30**. Pas de regeneration
OpenAPI ni de types web. `internal/analysis/filmdec/` n'est pas modifie du tout (les cinq
instruments sont des fichiers `_test.go`) : le garde-fou de cadrage `fire_events == head type36`
est donc intact par construction.

---

## 4. LES GATES

### 4.1 Gates de la MISSION A

| gate | enonce | mesure | verdict |
|---|---|---|---|
| A1 | etablir la grammaire d'enchainement | `1 [R(7) type] [3 refs] [charge]` … `0` ; la liste TERMINE apres un evenement vehicule (99/100), temoins -1 / +1 a 5 % / 0 % | **PASSE** |
| A2 | cadrage de la tete reste BIT-EXACT | `event_list_test.go` intact, `filmdec` non modifie, suite verte | **PASSE** |
| A3 | le compte d'exits ne change pas | aucun changement de decodeur : les comptes corpus 348 / 5 144 sont inchanges par construction | **PASSE (par construction)** |
| A4 | **le compte de boards MONTE substantiellement** | **NON — et c'est le resultat.** 1 894 candidats hors tete contre 3 017 en moyenne aux 4 temoins : l'exces est NUL. Le ratio 1:15 est le film | **ECHOUE, et EXPLIQUE** |
| A5 | chaque board nouveau porte occupant en bande et siege plausible | sans objet (aucun board nouveau) ; les candidats du bruit ont des sieges UNIFORMES sur 0..7, la contre-signature | **[~] absorbe par A4** |
| A6 | temoin : decalage volontaire d'1 bit doit s'effondrer | **il ne s'effondre pas, il DEPASSE** (4 246 contre 1 894) — c'est ce qui tranche | **PASSE (le temoin fait son travail)** |
| A7 | registre des types inconnus | 30 types, § 1.3 | **PASSE** |

### 4.2 Gates de la MISSION B

| gate | enonce | mesure | verdict |
|---|---|---|---|
| B1 | appariement board -> exit PAR OCCUPANT | machine d'etats livree ; sur 49 episodes de 5 films, **2 ont leurs DEUX bords dates par un evenement**, 42 par la seule sortie, 5 sont des silences terminaux | **PASSE** |
| B2 | vehicule resolu par la position, le trou en repli | fait : deux ancres (debut, fin), trou en repli anti-doublon | **PASSE** |
| B3 | silence terminal attribuable | **5 sur 49** episodes, la ou l'ancien pilotage en attribuait **0** | **PASSE** |
| B4 | multi-occupants simultanes | supporte par construction (episodes par occupant) ; `ambiguous` les compte et les publie | **PASSE** |
| B5 | chaque episode reste corrobore (temoin) | temoin +60 s : **0 episode rattache** sur 4 films sur 5, **1** sur `21468645` (une ancre a 1,3 m, deja dans le rayon de 1,5 m — le rayon n'y est pour rien) | **PASSE** |
| B6 | fusion des fantomes et clamp inchanges | aucune ligne touchee, tests verts | **PASSE** |
| B7 | les tirs en vehicule montent avec les episodes | voir § 5 | **voir § 5** |

---

## 5. LES DEUX ARTEFACTS DE DEMONSTRATION — AVANT / APRES

Reconstruits par `cmd/replay-build` (`LEVELUP_REPO_ROOT` = CE worktree, films du main tree), puis
COPIES dans `C:/Users/Guillaume/Projects/LevelUp-wt-capture-rejeu/data/cache/replays/halo_infinite/`.

| grandeur | `0d76e8f1` AVANT | `0d76e8f1` APRES | `fccc61cd` AVANT | `fccc61cd` APRES |
|---|---|---|---|---|
| vies publiees | 20 | 20 | 8 | 8 |
| **episodes d'occupation** | **12** | **13** | **2** | **3** |
| dont bornes par EVENEMENT (2 bords) | 0 | 0 | 0 | 0 |
| dont MIXTES (1 bord evenement) | 10 | **11** | 2 | **3** |
| dont par le TROU seul | 2 | 2 | 0 | 0 |
| occupants NOMMES (xuid) | 11 (91,7 %) | 11 (**84,6 %**) | 1 (50 %) | 1 (33,3 %) |
| episodes AVEC SIEGE | 10 (83,3 %) | **11 (84,6 %)** | 2 (100 %) | **3 (100 %)** |
| chevauchements (`ambiguous`) | 1 | **3** | 0 | 0 |
| vehicules occupes | 8 | 8 | 2 | **3** |
| tirs en vehicule (`Shot.v`) | 23 | **23** | 0 | **0** |
| orphelins hors episode | 180 | 180 | 212 | 212 |
| octets | 2 445 808 | 2 445 864 | 2 022 770 | 2 022 836 |
| `SchemaVersion` | 30 | **30** | 30 | **30** |

**LECTURE HONNETE, ligne par ligne.**

- **+1 episode sur chacun des deux films** (+8,3 % et +50,0 %). C'est REEL et c'est MODESTE, et la
  raison est la mission A : `0d76e8f1` n'atteste que **11 trajets** (1 embarquement + 10 sorties) et
  `fccc61cd` **3**. La production etait deja au plafond du FILM ; le lot V6 ne fait que le toucher
  exactement, et il prouve maintenant que le plafond est bien la.
- **Le gain de `0d76e8f1` est LE SILENCE TERMINAL** (slot 561, embarquement a t = 2 418,25 s, aucune
  sortie, aucune reapparition) : c'est le premier episode du chantier attribue a un occupant qui ne
  revient jamais. Il est publie sur 906 frames (90,6 s), borne par la fin de vie du vehicule 773.
- **`ambiguous` passe de 1 a 3, et c'est ATTENDU, pas une degradation** : le champ compte les vies
  portant deux episodes qui se CHEVAUCHENT, c'est-a-dire exactement les multi-occupants que la
  mission demandait de publier. Le vehicule 773 porte desormais quatre episodes (slots 554, 551,
  561, 551) dont trois se recouvrent.
- **Le taux de NOMMES baisse en pourcentage sans qu'aucun episode ne perde son nom** (11 nommes
  avant et apres) : le denominateur a augmente d'un episode dont l'occupant n'est pas dans le pont.
- **LES TIRS EN VEHICULE NE MONTENT PAS** (23 -> 23, et 0 -> 0). La mission l'attendait
  « mecaniquement » ; la mesure dit non, et la raison est directe : l'unique episode gagne sur
  chaque film ne recouvre aucun orphelin d'arme de vehicule. Les 180 orphelins hors episode de
  `0d76e8f1` restent hors episode. **C'est ecrit tel quel plutot que passe sous silence.**
- **`SchemaVersion` reste 30** : memes champs, plus d'episodes. Aucune raison de le bousculer, donc
  aucun artefact deja cuit n'est invalide.

---

## 6. LES GATES FINAUX

| gate | commande | resultat |
|---|---|---|
| gofmt | `gofmt -l internal/analysis/{replay,filmdec}/` | **sortie VIDE** |
| vet | `CGO_ENABLED=0 go vet ./internal/analysis/filmdec/ ./internal/analysis/replay/` | **exit 0** |
| tests SANS environnement | `CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1` | `ok filmdec 1,407s` · `ok replay 31,633s` — **0 `--- FAIL:`** |
| service en CGO=1 | `CGO_ENABLED=1 go test ./internal/service/... ./internal/replaybuild/...` | 5 paquets **ok**, 0 FAIL |
| OpenAPI / types web | contrat INCHANGE (memes champs, meme `SchemaVersion`) | **sans objet** |
| seuils de fichier | tous les fichiers touches <= 500 L (le plus gros : `vehicle_rides.go`, 383 L) | **PASSE** |
| seuils de fonction | fonction la plus longue des fichiers neufs : 61 L | **PASSE** |
| artefacts | 2 reconstruits + copies vers `LevelUp-wt-capture-rejeu` | **fait** |

---

## 7. STATUT DES ITEMS

| item | statut | justification |
|---|---|---|
| A1. Etablir la grammaire d'enchainement | `[x]` | la liste TERMINE apres un evenement vehicule (99/100, temoins 5 % / 0 %) ; longueur CONSTANTE par type, 42 et 52 bits, decomposition exacte |
| A1a. Distribution de l'espace entre fin de tete et trame ECS | `[x]` | 1 310 a 2 790 bits, etalee, aucune valeur repetee plus de 3 fois : c'est la trame, pas un evenement |
| A1b. Ancrage aval pour deduire les longueurs inconnues | `[!]` **REFUTE** | « fin propre » 0,0 % au vrai debut ; profondeur 23,3 % par paquet et **argmax agrege a 47 au lieu de 43** sur 250 sorties. Le decodeur ECS n'est pas bit-exact, il ne peut pas servir de regle. La marche sequentielle des types inconnus reste FERMEE |
| A2. GATE cadrage de la tete BIT-EXACT | `[x]` | `filmdec` non modifie (5 fichiers `_test.go` seulement), suite verte |
| A3. GATE compte d'exits inchange | `[x]` | par construction : aucun decodeur touche |
| A4. GATE compte de boards en HAUSSE substantielle | `[!]` **ECHOUE, et la raison est mesuree** | reel hors tete 1 894 contre 3 017 en moyenne aux 4 temoins. Il n'y a pas d'embarquement cache |
| A5. Chaque board nouveau corrobore | `[~]` | absorbe par A4 : aucun board nouveau. Les candidats du bruit ont des sieges uniformes sur 0..7 (les vrais valent 0 a 95 %) |
| A6. TEMOIN par decalage d'1 bit | `[x]` | fait, sur 4 decalages ; il DEPASSE le reel — c'est lui qui tranche le lot |
| A7. Registre des types inconnus | `[x]` | 30 types, § 1.3 |
| B1. Machine d'etats par occupant | `[x]` | `vehicle_rides_events.go`, 6 garde-rails sans environnement |
| B2. Vehicule par la position, trou en repli | `[x]` | deux ancres ; trou en repli anti-doublon, regle corrigee apres une regression mesuree (12 -> 11) |
| B3. Silence terminal | `[x]` | 5 sur 49 episodes de 5 films ; 1 publie sur `0d76e8f1` |
| B4. Multi-occupants simultanes | `[x]` | par construction ; `ambiguous` 1 -> 3 sur `0d76e8f1` |
| B5. GATE episodes AVANT / APRES | `[x]` | 12 -> 13 et 2 -> 3. **Bien moins que « beaucoup plus »**, et la mission A dit pourquoi |
| B6. GATE % nommes, % avec siege, sieges coherents | `[x]` | nommes 11/13 et 1/3 ; siege 11/13 et 3/3. Coherence board/exit non re-mesurable ici (2 paires seulement sur 5 films) — la reference reste 5/6 = 83,3 % du lot V3 |
| B7. GATE les tirs en vehicule MONTENT | `[!]` **NON** | 23 -> 23 et 0 -> 0 : les episodes gagnes ne recouvrent aucun orphelin d'arme de vehicule. Dit, non maquille |
| B8. Fusion des fantomes et clamp inchanges | `[x]` | aucune ligne touchee, tests verts |
| B9. Artefacts reconstruits et copies | `[x]` | § 5 |
| B10. Bump `SchemaVersion` | `[~]` **non, a raison** | memes champs : le contrat ne change pas |
| Rapport + thought_log | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` |

---

## 8. CE QUI RESTE OUVERT (note, NON traite — regle du perimetre)

1. **La marche sequentielle de la liste reste impossible** faute de longueur de charge pour les
   28 types non decodes. Elle n'a plus d'interet pour les VEHICULES (§ 1.6), mais elle en garderait
   pour les 12 770 evenements de type 0 ou les 7 812 de type 82. Les deux voies : rendre le
   decodeur ECS bit-exact (c'est lui qui bloque l'ancrage aval), ou lire les deserialiseurs de
   charge dans l'executable — donc Ghidra, qui est mort.
2. **La ref 2 de l'embarquement n'est toujours pas interpretee.** Quatre valeurs (180, 116, 244,
   208) pour vingt-deux instances : la piste la plus probable est une base de domaine 7 non nulle
   qu'on ne sait pas lire, ou un identifiant de DEFINITION de chassis. Test possible sans Ghidra :
   correler ces quatre valeurs avec le `MPPWord32` du chassis de l'episode apparie.
3. **LE TIR EN VEHICULE COMME SOURCE D'EPISODE.** C'est la seule piste restante pour gagner des
   episodes, et elle est chiffree : 180 orphelins hors episode sur `0d76e8f1`, dont ceux qui portent
   une arme de vehicule (moitie basse d'identifiant NULLE). Chacun NOMME son tireur et PROUVE son
   occupation a l'instant. Ce qui manque est le VEHICULE : le record de tir ne porte pas de
   position, et aucune methode justifiable ne le resout aujourd'hui. Non traite.
4. **Le taux d'occupants nommes** (84,6 % et 33,3 %) est limite par le pont slot -> xuid, pas par la
   machine d'etats.
5. **Le rayon de 3 m n'a ete etalonne que sur 5 films et 49 episodes**, dont 4 sur Behemoth. Un film
   a densite de vehicules tres superieure ferait monter l'ambiguite ; la table du § 2.3 est le
   protocole a rejouer avant d'y toucher.
