# RAPPORT — lot V5B : LIRE LE BLOC DU RECORD D'IMAGE-CLE VEHICULE

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB. Mesures en AVANT-PLAN, `CGO_ENABLED=0`,
> GOCACHE isole (`scratchpad/gocache_v5b`), donnees reelles du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache`, LECTURE SEULE).
> `internal/analysis/replay/` (production), `apps/web/`, `cmd/weapon-sounds/` : NON touches.
> Ghidra : MORT (verifie au lot V5, `curl` exit 7) — tout ce rapport est de la MESURE.

---

## 0. LE RESULTAT EN SIX LIGNES

1. **LE BLOC EXISTE, ET C'EST UNE INSERTION AU SENS STRICT.** Le modele
   « O = F[0:p] + BLOC(d) + F[p:] » (meme vehicule, images-cles voisines) explique **95,4 a
   97,7 %** des bits compares, contre 66-91 % pour les deux modeles degeneres. Le temoin
   libre/libre reste a **96-98 % d'accord PARTOUT** : la chute d'accord de la paire reelle n'est
   pas de la derive de contenu.
2. **LA TAILLE EST 89 BITS, EXACTEMENT, ET ELLE SE REPETE** : 10 blocs sur 18 attestes valent
   exactement +89, sur **6 films independants**. La position `p` varie (364 a 422 bits depuis le
   debut du record) : ce qui precede est de longueur variable.
3. **LE BLOC NE NOMME PAS SON OCCUPANT — refutation exhaustive.** Balayage de TOUT le dictionnaire
   d'encodages du chantier (4 formes d'entite du lot V5, references gardees de largeur 7/8/13 des
   evenements board/exit, siege `R(6)`) a tous les decalages du bloc +/- 48 bits, sur 18 instances :
   le meilleur canal donne **1/18 = 5,6 %**, et **le temoin par permutation des occupants donne
   1/18 lui aussi**. Le siege « passe » a 16/18 — le temoin permute aussi : les 18 sieges attestes
   valent 0, le canal lit un zero.
4. **LE VRAI FACTEUR EST LA CINEMATIQUE, PAS L'OCCUPATION — et cela CORRIGE le lot V5.** Sur 104
   records `ti=40` mesurables (8 films, sans aucun oracle), un vehicule **A L'ARRET** porte un
   exces >= 89 bits dans **25,0 %** des cas, un vehicule **EN MOUVEMENT** dans **66,2 %**. A
   mouvement controle, l'occupation attestee ajoute encore (**80,0 % contre 58,1 %**, n=68), mais
   la classe temoin est contaminee. Le « signal d'occupation » du lot V5 est, pour l'essentiel, un
   signal de MOUVEMENT.
5. **« UN BLOC PAR OCCUPANT » N'A PAS PU ETRE TESTE : le corpus ne contient AUCUN vehicule a deux
   occupants attestes SIMULTANEMENT.** Le Warthog multi-occupants annonce de `0d76e8f1` se defait
   a l'appariement : ep6 (slot 554) et ep8 (slot 551) vont au vehicule 773 mais ne se recouvrent
   pas dans le temps (2405,6-2420,7 puis 2422,1-2443,2), et ep7 (slot 559), qui les recouvre, est
   apparie au vehicule 792. N=2 n'existe nulle part. Le dire est le resultat.
6. **LES LONGUEURS NE SONT PAS QUANTIFIEES PAR 89.** Sur les vehicules a >= 2 longueurs distinctes,
   **2 sur 9** ont toutes leurs longueurs congruentes modulo 89. Les ecarts observes : 19, 20, 29,
   33, 45, 46, 52, 56, 62, 72, **89**, 108, 137, 171, **190**, 217, 269, 278, 279, 334, 348, 367,
   517. **+89 et +190 se repetent, le reste non** : il y a PLUSIEURS blocs a geometrie variable,
   pas un quantum unique.

**Ce qui est livre** : `filmdec/vehicle_occupancy.go` (production, additif, exporte) —
`ScanFilmVehicleOccupancy` / `VehicleKeyframeStates` (l'exces de longueur par image-cle et par
vehicule, avec sa ligne de base par chassis) et `FindKeyframeBlockInsertion` (le localisateur
d'insertion par maximum de vraisemblance, pur, en temps lineaire), plus trois garde-rails SANS
environnement. **Le decodeur `{occupant, siege}` demande par la mission N'EST PAS livre : il n'y a
rien a decoder — voir § 3.**

---

## 1. LE PROTOCOLE, ET LES CINQ INSTRUMENTS

Vérité terrain : identique au lot V5 (episodes attestes = [debut du trou du flux de position d'un
slot bipede, instant de la SORTIE decodee qui le referme], seuils 3 s / +/- 2 s / temoin 37 s ;
appariement episode -> vehicule par la position a la reapparition de l'occupant). Les emprises de
record viennent de `KeyframeRecordSpans` (production, lot V5) et sont TOUJOURS filtrees a
`SlotGap == 1` — sans ce garde-fou une « longueur » couvre plusieurs records (lot V5 § 6.3 :
l'ecart apparent tombe de +1 348 a +151 bits une fois le filtre pose).

| instrument | question | garde |
|---|---|---|
| `TestV5BDiff` | profils d'accord bit a bit AVANT / ARRIERE, occupe contre libre, + temoin libre/libre | `V5_ROOT`/`V5_FILMS` |
| `TestV5BBloc` | position d'insertion au bit pres (max. de vraisemblance) + dump des bits | idem |
| `TestV5BLongueurs` | histogramme des longueurs par vehicule — SANS oracle | idem |
| `TestV5BMouvement` / `TestV5BVitesse` | le confondant cinematique — table 2x2, puis exces contre deplacement SANS oracle | idem |
| `TestV5BChamps` | le bloc porte-t-il un champ connu ? temoin par permutation | idem |
| `TestV5BAncrage` | `p` tombe-t-il sur une frontiere de composant ? | idem |
| `TestV5BMulti` | « un bloc par occupant » | idem |

---

## 2. OU LE BLOC S'INSERE — ET LA PREUVE QUE C'EN EST UN

### 2.1 Les profils d'accord (`TestV5BDiff`, 4 films, 9 paires reelles, 59 paires temoin)

Accord bit a bit, par fenetre de 32 bits, aligne sur le DEBUT du record :

| decalage | 0..256 | 288 | 320 | 384 | 512 | 1024 | 1536 | 2048 |
|---|---|---|---|---|---|---|---|---|
| **reel occupe/libre** | **100 %** | 82 % | **64 %** | 66 % | 66 % | 56 % | 81 % | 77 % |
| **temoin libre/libre** | 100 % | 96 % | **90 %** | 90 % | 97 % | 97 % | 98 % | 95 % |

Aligne sur la FIN du record :

| decalage depuis la fin | 0..1216 | 1248 | 1344 | 1856 | 2112 |
|---|---|---|---|---|---|
| **reel occupe/libre** | 76-98 % | 76 % | **61 %** | 56 % | 52 % |
| **temoin libre/libre** | 91-99 % | 94 % | **91 %** | 91 % | 95 % |

**LECTURE.** Le temoin libre/libre est a 96-98 % PARTOUT : deux records libres du meme vehicule
sont presque identiques (vehicules a l'arret). La paire reelle est a **100 % jusqu'au bit 288**
puis s'effondre a 64 % : c'est une frontiere nette. Alignee sur la FIN, elle tient jusqu'a
~1 250 bits de la queue. Pour un record de 1 766 bits, 1 766 - 300 - 89 = 1 377 : les deux
frontieres se rejoignent. **C'est la signature d'une insertion, pas d'un changement de contenu.**

### 2.2 La position au bit pres (`TestV5BBloc`)

`v5bInsertion` / `FindKeyframeBlockInsertion` cherche le `p` qui maximise l'accord sous le modele
d'insertion (sommes prefixes, temps lineaire). Les deux modeles DEGENERES — tout le decalage
rejete en tete, tout rejete en queue — sont le temoin interne.

| film | vehicule | Lo | Lf | d | **p** | accord | temoin tete | temoin queue |
|---|---|---|---|---|---|---|---|---|
| `0d76e8f1` | 773 | 2 398 | 2 309 | **89** | **422** | **2 257/2 309 = 97,7 %** | 2 107 | 1 544 |
| `53ce4390` | 781 | 2 398 | 2 309 | **89** | **364** | **2 236/2 309 = 96,8 %** | 2 093 | 1 561 |
| `a89a3d23` | 769 | 1 808 | 1 719 | **89** | **420** | **1 645/1 719 = 95,7 %** | 1 523 | 1 174 |
| `21468645` | 770 | 1 766 | 1 677 | **89** | **393** | **1 600/1 677 = 95,4 %** | 1 485 | 1 151 |
| `21468645` | 791 | 1 808 | 1 719 | **89** | **393** | **1 671/1 719 = 97,2 %** | 1 527 | 1 176 |
| `0d76e8f1` | 771 | 2 095 | 1 747 | 348 | 1 661 | 1 252/1 747 = 71,7 % | 1 066 | 1 210 |

Les cinq blocs a d = 89 sont nets (95-98 %, tres au-dessus des deux degeneres). Le bloc a d = 348
ne l'est pas (71,7 %) : ce n'est pas UNE insertion propre, et c'est dit.

**LA POSITION N'EST PAS CONSTANTE** (364, 393, 393, 420, 422). Deux vehicules DIFFERENTS du meme
film (`21468645` : 770 et 791) donnent le meme `p = 393`, mais deux films donnent 364 et 422. Le
contenu qui precede le bloc est de longueur variable — coherent avec des composants a precision
dynamique en amont.

### 2.3 Ce que le bloc contient : de l'entropie, pas un champ

Les 89 bits, releves tels quels (`0d76e8f1` veh 773, occupant slot 551) :

```
00000001010100000111101010000010100100110101011101001100001011101000101000110110110000100
```

Aucune structure evidente : pas de champ nul, pas de constante partagee entre instances, pas de
prefixe commun. C'est la signature d'un bloc de VALEURS (quantifiees), pas d'un bloc
d'identifiants.

### 2.4 L'ancrage dans la grammaire (`TestV5BAncrage`) — indicatif seulement

`WalkKeyframeBody` (variante etat complet) sur la paire `21468645` veh 770 rend, en decalage
relatif au debut du record :

| composant | occupe | libre | ecart |
|---|---|---|---|
| i0 object-position-dynamic-precision | 64 | 64 | 0 |
| i1 object-translational-velocity-dynamic-precision | 177 | 177 | 0 |
| i2 object-forward-and-up-dynamic-precision | 208 | 208 | 0 |
| i3 object-angular-velocity-dynamic-precision | 236 | 236 | 0 |
| i6 object-region-state | 291 | 291 | 0 |
| **i7 object-damage-sections** | **328** | **301** | **+27** |
| i8 object-constraint | 549 | 457 | **+92** |

`p = 393` tombe DANS i7 (`object-damage-sections`), et le cumul des deux ecarts (27 + 65 = 92)
approche 89. **AVERTISSEMENT : cette lecture n'est PAS fiable.** La grammaire de `ti=40` est
connue fausse (i2/i3 refutes en V1a/V2b) et la deuxieme paire du meme test (veh 791) place la
divergence sur i6 et non i7. L'ancrage est une PISTE (etat de degat / d'occupation de region),
pas une conclusion.

---

## 3. LE BLOC NE NOMME PERSONNE — la refutation exhaustive (`TestV5BChamps`)

**LE DICTIONNAIRE EST CELUI DU CHANTIER, pas une devinette.** Les evenements board/exit lisent des
references gardees `[garde:1][index:w][generation:2]` avec w = 8 (domaine 2, l'occupant d'un
embarquement), w = 7 (domaine 3), w = 13 (domaine 7), puis un siege en `R(6)`
(`event_list.go`). S'ajoutent les 4 formes d'entite deja balayees au lot V5 (`s13`, `g15h`,
`g15l`, `h32`), chacune testee contre le SLOT absolu et contre l'INDEX relatif a la base bipede.

**LA REGLE DE DECISION EST ECRITE AVANT LA MESURE** : un canal (forme x decalage) ne compte que
s'il designe le BON occupant sur TOUTES les instances. Avec 18 instances et un champ de 8 bits,
une coincidence sur toutes vaut (1/256)^18 — un seul canal passant serait decisif.

Balayage : 17 formes x 185 decalages (le bloc +/- 48 bits) x 18 instances attestees, 8 films.

| cible | **reel** | **temoin (occupants permutes d'un cran)** |
|---|---|---|
| slot de l'occupant | **1/18 = 5,6 %** (`brut-s13`, decalage -26) | **1/18 = 5,6 %** |
| index de l'occupant | **1/18 = 5,6 %** (`brut-idx-s13`, decalage -28) | **1/18 = 5,6 %** |
| siege `R(6)` | 16/18 = 88,9 % (decalage +187) | **16/18 = 88,9 %** |

**LE SIEGE EST UN FAUX POSITIF, et le temoin le demasque** : les 18 sieges attestes valent TOUS 0
(le corpus n'atteste que des conducteurs), donc n'importe quel champ de 6 bits nul « predit » le
siege. Le temoin permute, qui attend les memes zeros, fait le meme score. **Aucun siege n'est
decode.**

**VERDICT : le bloc ne porte NI l'occupant NI le siege, sous aucun des encodages que le moteur
emploie ailleurs.** Ce n'est pas « on n'a pas trouve » : le temoin par permutation egale le reel a
la decimale pres sur les trois cibles.

---

## 4. LE CONFONDANT QUI CHANGE LA CONCLUSION DU LOT V5

### 4.1 Enonce AVANT la mesure

Un vehicule occupe est un vehicule CONDUIT, donc un vehicule qui BOUGE. Les quatre premiers
composants de `ti=40` sont `object-position-`, `object-translational-velocity-`,
`object-forward-and-up-` et `object-angular-velocity-dynamic-precision` : a precision dynamique,
ils emettent plus de bits quand l'objet est dynamique. Le « bloc d'occupation » peut n'etre qu'un
bloc de mouvement.

### 4.2 La table 2x2 (`TestV5BMouvement`, seuil 30 quanta sur +/- 1,5 s)

Deux cases tuent l'interpretation « occupation » :

| film | vehicule | occupe+bouge | occupe+arret | **libre+bouge** | libre+arret |
|---|---|---|---|---|---|
| `53ce4390` | 781 | 2 398 (k=1) | — | **2 398 (k=2)** | 2 309 (k=3) |
| `21468645` | 775 | — | — | **2 095 (k=2)** | 2 006 (k=2) |
| `0d76e8f1` | 773 | 2 398 (k=1) | **2 119 (k=1)** | — | 2 309 (k=1) |

Le vehicule 781 de `53ce4390` a **exactement la meme longueur (2 398) libre-en-mouvement
qu'occupe-en-mouvement**, et 2 309 a l'arret : l'ecart de 89 bits suit le MOUVEMENT, pas
l'attestation. Le vehicule 775 de `21468645` le refait sans aucune occupation attestee
(2 095 - 2 006 = **+89**). Et le vehicule 773 de `0d76e8f1`, occupe MAIS A L'ARRET, est le PLUS
COURT de ses trois cases (2 119) — l'inverse de ce que l'hypothese occupation predit.

### 4.3 La mesure SANS oracle, 8 films (`TestV5BVitesse`)

Pour chaque record `ti=40` a voisin immediat : exces de longueur par rapport a la plus courte
longueur jamais observee POUR CE VEHICULE, croise avec le deplacement du vehicule autour de
l'image-cle. **Aucun episode attesté n'intervient.**

Records examines : **1 148**. Sans estimation de deplacement (< 2 echantillons de position du
vehicule dans la fenetre) : **1 044**. Retenus : **104**. *(Le flux de position vehicule est
epars : c'est la limite de cette mesure, et elle est dite.)*

| deplacement (quanta / 3 s) | n | exces median | exces >= 89 bits |
|---|---|---|---|
| 0 | 15 | 0 | 5 (33,3 %) |
| < 1 | 8 | 0 | 1 (12,5 %) |
| < 10 | 8 | 52 | 2 (25,0 %) |
| < 30 | 5 | 0 | 1 (20,0 %) |
| < 100 | 3 | 72 | 1 (33,3 %) |
| < 300 | 4 | 279 | 2 (50,0 %) |
| < 1 000 | 28 | **89** | **23 (82,1 %)** |
| >= 1 000 | 33 | **89** | 19 (57,6 %) |

**VERDICT MOUVEMENT : a l'arret (<= 30 quanta) 9/36 = 25,0 % ; en mouvement 45/68 = 66,2 %.**

**A MOUVEMENT CONTROLE** (records en mouvement seulement) : occupation ATTESTEE
**20/25 = 80,0 %** ; sans episode atteste **25/43 = 58,1 %**.

**LECTURE HONNETE.** Le mouvement explique l'essentiel (25 % -> 66 %). L'occupation attestee
ajoute un residu (58 % -> 80 %) mais (a) n = 68, (b) la classe « sans episode atteste » est
contaminee : le ratio board:exit = 1:15 garantit que la majorite des trajets n'y sont pas attestes.
**Ce residu n'est pas exploitable comme oracle, et on ne peut pas trancher entre « le bloc porte
aussi de l'occupation » et « les records "libres" en mouvement sont des trajets non attestes ».**

---

## 5. « UN BLOC PAR OCCUPANT » : LE CORPUS NE PERMET PAS DE LE TESTER (`TestV5BMulti`)

Appariement episode -> vehicule sur `0d76e8f1`, les 10 episodes attestes :

| vehicule | occupants attestes (fenetres) |
|---|---|
| 768 | 522 [2200,9 -> 2203,5] |
| 771 | 512 [2157,0 -> 2159,2] · 514 [2166,6 -> 2184,6] · 515 [2212,4 -> 2231,2] |
| **773** | **554 [2405,6 -> 2420,7]** · **551 [2422,1 -> 2443,2]** |
| 776 | 531 [2293,9 -> 2299,7] |
| 777 | 514 [2155,6 -> 2163,8] |
| 791 | 602 [2700,4 -> 2744,6] |
| 792 | 559 [2413,4 -> 2521,9] |

Le vehicule 773 porte bien DEUX occupants, mais **SEQUENTIELLEMENT** : 554 descend a 2420,7,
551 monte a 2422,1. Le seul episode qui recouvre les deux (559, [2413,4 -> 2521,9]) est apparie au
vehicule **792**, pas 773 — appariement par la position a la reapparition, la seule methode dont
on dispose (l'evenement de sortie ne nomme pas le vehicule, V3_EMBARQUEMENT § 4).

**Aucune image-cle du corpus ne tombe sur un vehicule a N >= 2 occupants attestes.** Longueur par
nombre d'occupants, la ou la mesure existe :

| vehicule | N=0 | N=1 | N=2 |
|---|---|---|---|
| 771 | 1 747 (k=8) | 2 095 (k=2) | **absent** |
| 773 | 2 119 (k=15) | 2 398 (k=2) | **absent** |

**L'hypothese « un bloc par occupant » n'est ni confirmee ni refutee : elle n'a pas ete testable.**
Elargir l'oracle (marche de la liste ENTIERE d'evenements, cf. V3_EMBARQUEMENT § 4.3) est le
prealable.

---

## 6. LES LONGUEURS NE SONT PAS UN QUANTUM DE 89 (`TestV5BLongueurs`)

Sans aucun oracle, tous les vehicules du film a >= 2 longueurs distinctes :

| film | vehicule | longueurs observees | congruentes mod 89 ? |
|---|---|---|---|
| `0d76e8f1` | 768 | 2 127, 2 317 (+190) | non |
| `0d76e8f1` | 771 | 1 728, 1 747, 1 761, 1 774, 2 095 | non |
| `0d76e8f1` | 773 | 2 119, 2 309 (+190), 2 398 (+279) | non |
| `21468645` | 768 | 2 127, 2 298 (+171) | non |
| `21468645` | **770** | **1 677, 1 766 (+89)** | **oui** |
| `21468645` | 775 | 1 728, 2 006, 2 062, 2 095 | non |
| `21468645` | 776 | 1 234, 1 254 (+20) | non |
| `21468645` | **791** | **1 719, 1 808 (+89)** | **oui** |
| `21468645` | 794 | 1 703, 1 840 (+137) | non |

**2 sur 9.** `+89` et `+190` se repetent chacun deux fois ; tout le reste est unique. Il y a
PLUSIEURS blocs optionnels de tailles differentes, et des variations fines (+19, +20, +33, +46)
qui sont la signature d'une precision dynamique. **Le record ne croit pas par quanta de 89 bits**
— d'ou le choix, dans le decodeur livre, de traiter 89 comme un PLANCHER et non comme une unite de
comptage.

---

## 7. LES GATES DE LA MISSION

| gate | enonce | mesure | verdict |
|---|---|---|---|
| **1** | localiser le bloc (offset, prefixe/porte) | insertion prouvee, `d = 89` exact sur 10/18, `p` = 364..422 (variable), modele a 95,4-97,7 % contre 66-91 % aux degeneres. **Aucun bit de presence ni compteur n'a ete isole** : `p` variant d'un film a l'autre, il n'y a pas de porte a un decalage fixe | **PARTIEL** |
| **2** | « un bloc par occupant » sur le Warthog multi-occupants | **NON TESTABLE** : aucun vehicule du corpus n'a 2 occupants attestes simultanement (§ 5) | **[!] impossible** |
| **3** | le champ decode designe l'OCCUPANT de l'episode, 100 % des images-cles couvertes ; siege accorde | **1/18 = 5,6 %**, EGAL au temoin par permutation. Siege : 16/18 au reel ET au temoin (tous les sieges valent 0) | **ECHOUE** |
| **4** | decodeur `filmdec/vehicle_occupancy.go` + tests non gardes | livre, mais il rend une MESURE (exces de longueur) et non `{occupant, siege}` — voir § 8 | **PARTIEL, honnete** |
| **temoin** | permutation des occupants | fait, et il EGALE le reel : c'est lui qui tranche le § 3 | **PASSE** |
| **nouveau** | specificite : le bloc est-il de l'occupation ? | **NON, ou pas seulement** : 25,0 % a l'arret contre 66,2 % en mouvement, sans aucun oracle, sur 104 records de 8 films | **CORRIGE LE LOT V5** |

**CE QUE CE LOT CHANGE AU LOT V5.** V5 concluait « un signal d'occupation qui passe ses temoins
(b) et (d) ». V5B montre que ce signal est **domine par la cinematique** : le temoin de V5 (decalage
temporel de 37 s) ne pouvait pas le voir, parce qu'un vehicule occupe a t l'est souvent encore a
t+37 s, et surtout parce qu'il ne controlait pas le mouvement. Le controle bipede de V5 (§ 6.4) ne
protegeait pas non plus : un bipede a bord ne bouge pas dans SON flux, il n'a plus de flux.
**L'axiome de l'utilisateur reste intact — l'etat d'occupation existe et le mode Theatre l'affiche —
mais il n'est ni dans ce bloc, ni dans la longueur du record.**

---

## 8. CE QUI EST LIVRE

| fichier | etat | role |
|---|---|---|
| `internal/analysis/filmdec/vehicle_occupancy.go` | **NEUF** (production, additif, exporte) | `VehicleKeyframeBlockBits` (89), `VehicleKeyframeState`, `VehicleKeyframeStates` (PUR), `ScanFilmVehicleOccupancy`, `KeyframeRecordBits`, `KeyframeBlockInsertion`, `FindKeyframeBlockInsertion` (PUR, lineaire) |
| `internal/analysis/filmdec/vehicle_occupancy_test.go` | **NEUF** (garde-rail SANS env) | 3 tests sur payloads fabriques : le localisateur retrouve une insertion connue au bit pres ET bat ses deux modeles degeneres ; le cas « pas d'insertion » ; la ligne de base PAR VEHICULE, l'exces, et le rejet des emprises a voisin saute |
| `vehicules_v5b_{diff,bloc,controle,champs}_test.go` | **NEUFS** (instruments, garde `V5_ROOT`/`V5_FILMS`) | les sept mesures de ce rapport |
| `vehicules_v5b_diff_test.go` (`kfBit` -> `keyframeBitAt`) | — | l'accesseur de bit est passe en production avec le decodeur ; les instruments l'appellent au lieu de le redefinir |

**Le contrat du decodeur est ecrit dans son en-tete, avec les chiffres de la refutation.**
`ScanFilmVehicleOccupancy` rend, par image-cle et par vehicule, `LengthBits`, `BaselineBits`
(la plus courte emprise mesurable du MEME vehicule), `ExcessBits` et `ExtraBlock`. Il dit
explicitement que **ce n'est pas un oracle d'occupation** : un appelant qui en ferait « ce vehicule
est occupe » se tromperait une fois sur quatre a l'arret et une fois sur trois en mouvement.
`Measurable` (= `SlotGap == 1`) porte le garde-fou du lot V5 : les emprises a voisin saute sont
rendues mais n'ont ni ligne de base ni exces.

**Rien n'est branche dans `internal/analysis/replay/`.**

Suite sans environnement (tout ce qui est garde saute ; les garde-rails neufs tournent) :

```
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
gofmt -l internal/analysis/filmdec/   -> vide
go vet ./internal/analysis/filmdec/   -> propre
```

Commandes de rejeu (avant-plan, GOCACHE isole, `CGO_ENABLED=0`, `V5_ROOT=<cache>`) :

```
V5_FILMS=0d76e8f1,53ce4390,a89a3d23,21468645 \
  go test ./internal/analysis/filmdec/ -run TestV5BDiff  -v -timeout 120m   # 185 s
V5_FILMS=<idem>            go test ... -run TestV5BBloc      -v -timeout 120m   # 214 s
V5_FILMS=0d76e8f1,21468645 go test ... -run TestV5BLongueurs -v -timeout  60m   #  12 s
V5_FILMS=<8 films>         go test ... -run TestV5BVitesse   -v -timeout 120m   # 542 s
V5_FILMS=<8 films>         go test ... -run TestV5BChamps    -v -timeout 180m   # 470 s
V5_FILMS=<4 films>         go test ... -run TestV5BMouvement -v -timeout 120m   # 259 s
V5_FILMS=21468645          go test ... -run TestV5BAncrage   -v -timeout  60m   #  33 s
V5_FILMS=0d76e8f1          go test ... -run TestV5BMulti     -v -timeout  60m   #  41 s
```

Les 8 films : `0d76e8f1,fccc61cd,e232ffce,829abef9,53ce4390,4898d586,a89a3d23,21468645`.

---

## 9. LE PLAN D'INTEGRATION DEMANDE (etat aux ancres + transitions)

La mission demandait le plan d'une machine d'etats d'occupation « ancres image-cle + bords
board/exit ». **Ce plan reste VALIDE, mais son volet "ancre" perd son contenu** : les images-cles
n'apportent pas d'etat d'occupation lisible. Voici ce qui tient, et ce qui le remplace.

**CE QUI TIENT (les bords, dates a la milliseconde).** Les evenements sont decodes et valides :
la SORTIE (`EventUnitExitVehicle`, occupant 100 % en bande, fermeture de trou 90,7 % contre 0 % au
temoin) et l'EMBARQUEMENT (`EventBipedBoardVehicle`, domaines 2/3/7 lus dans l'executable,
occupant 22/22 en bande). Ils donnent des BORDS surs.

**CE QUI MANQUE, ET C'EST LE VRAI GOULOT.** Le ratio board:exit mesure est de **1:15** : il y a
quinze fois plus de sorties decodees que d'embarquements. Une machine d'etats a bords a besoin des
DEUX. Le prealable n'est donc plus l'image-cle mais **la marche de la liste ENTIERE d'evenements**
(V3_EMBARQUEMENT § 4.3) : aujourd'hui seul l'evenement de TETE de chaque paquet est decode
(`PacketHeadEventType`), les suivants de la liste sont perdus. C'est la piste a plus fort levier du
chantier, et elle est mecanique (la grammaire de la liste est connue : `1 [R(7) type] [3 refs
gardees] [charge]` repete jusqu'a un `0`), pas exploratoire.

**LE SUBSTITUT D'ANCRE, en attendant.** Le TROU du flux de position d'un bipede est deja un
detecteur d'embarquement valide (V1a.4) : il donne un bord de MONTEE date, sans nommer le
vehicule. Couple a l'appariement par la position (10/10 et 2/2 sur les deux films-demo), il
reconstruit un episode complet. C'est la voie qui a le meilleur rapport preuve/cout aujourd'hui,
et elle n'a besoin d'AUCUNE image-cle — donc elle ne subit pas le plafond de 58,3 % de couverture
du lot V5.

**L'ORDRE PROPOSE POUR LE LOT SUIVANT** :
1. marcher la liste ENTIERE d'evenements -> multiplier les embarquements decodes ;
2. machine d'etats [board -> exit] par occupant, vehicule resolu par la position ;
3. reserver `ScanFilmVehicleOccupancy` a ce qu'il est : un INDICE d'etat supplementaire, utilisable
   au plus comme depart d'un futur travail sur la cinematique — pas comme entree d'occupation.

---

## 10. STATUT DES ITEMS

| item | statut | justification |
|---|---|---|
| 1. Aligner et diffe bit a bit occupe/libre ; localiser l'insertion | `[x]` | profils AVANT/ARRIERE + temoin libre/libre ; `p` = 364..422, `d` = 89, modele a 95,4-97,7 % |
| 1bis. Trouver le prefixe/porte qui annonce le bloc (bit de presence, compteur) | `[!]` **non trouve** | `p` n'est pas constant d'un film a l'autre : il n'y a pas de porte a decalage fixe a exhiber. La grammaire `ti=40` etant fausse, on ne peut pas remonter au composant porteur |
| 2. « Un bloc par occupant » sur le Warthog multi-occupants | `[!]` **non testable** | aucun vehicule du corpus n'a 2 occupants attestes SIMULTANEMENT (§ 5) — l'appariement separe ep6/ep8 (sequentiels) de ep7 (autre vehicule) |
| 3. Decoder les champs avec les grammaires connues | `[x]` **refute** | 17 formes x 185 decalages x 18 instances : meilleur canal 1/18, temoin permute 1/18. Siege 16/18 au reel ET au temoin (tous les sieges valent 0) |
| 3bis. GATE occupant a 100 % des images-cles couvertes | `[!]` **ECHOUE** | 5,6 %, au niveau du hasard |
| 3ter. GATE siege accorde aux evenements | `[!]` **ECHOUE** | aucun siege decode ; le corpus n'atteste que des sieges 0 |
| 4. Decodeur `vehicle_occupancy.go` additif exporte + tests non gardes | `[x]` | livre ; il rend l'EXCES mesure, pas `{occupant, siege}` — contrat ecrit dans l'en-tete avec les chiffres |
| 4bis. Plan d'integration | `[x]` | § 9 — et il est REVISE : la voie image-cle est abandonnee, le goulot est le ratio board:exit 1:15 |
| 5. Rapport + thought_log | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` |
| Nouveau : specificite mouvement / occupation | `[x]` | 25,0 % contre 66,2 %, sans oracle, 104 records de 8 films — **corrige la conclusion du lot V5** |

**Ce qui reste ouvert** (note, non traite — regle du perimetre) :

1. **Marcher la liste ENTIERE d'evenements.** Ratio board:exit = 1:15. C'est le goulot de tout le
   chantier vehicule, et la grammaire est connue.
2. **Reparer la grammaire de `ti=40`** (i2/i3 refutes en V1a/V2b). Sans elle, aucun ancrage de bloc
   dans un composant n'est concluant — le § 2.4 en est l'illustration.
3. **Identifier ce que porte reellement le bloc de 89 bits.** Les deux candidats mesures sont la
   cinematique (i1/i3, precision dynamique) et l'etat de degat (i6/i7). Un test possible sans
   Ghidra : correler la VALEUR du bloc avec le deplacement observe ensuite (si c'est une vitesse,
   elle predit la position suivante).
4. **Le bloc de +190 bits**, qui se repete lui aussi (deux vehicules, deux films) et n'a pas ete
   ouvert.
