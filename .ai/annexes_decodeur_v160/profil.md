# Profil du temps de decodage (decodeur de film v1.5.0, tete de `piste/v0`)

Mesure faite le 2026-11-23, machine a 16 coeurs, un film a la fois, memoire libre superieure a 15 Gio.
Aucun changement de comportement du decodeur : seul `cmd/film-bench` recoit deux options
(`-profil-cpu`, `-profil-tas`, avec `-seulement filmNN`) ; sans elles, les mesures du releve sont
inchangees (un seul test `nil` de plus par etape observee, dans la fermeture d'observation).

## Methode et limites

- Une passe du decodeur par film (comme `executer` du banc : memes options minimales, carte seule),
  sous `runtime/pprof` : un profil CPU, puis un profil du tas (echantillon tous les 4 Kio, plus une
  prise du tas au plus haut vu par un echantillonneur toutes les 25 ms : `filmNN.tas.pprof.pic`).
- Les comptes d'appels viennent d'un binaire compile avec `go build -cover -covermode=count` (compteurs
  de blocs, sources intactes), lancee sur les memes films : `couv/filmNN.cov`.
- Le profil CPU rallonge la passe (film06 : 23,0 s profilee contre 17,9 s au releve de reference) : lire
  les PARTS, pas les durees. Les pourcentages ci-dessous sont des parts des echantillons CPU (le total
  depasse la duree murale de 2 a 6 % : le ramasse-miettes travaille en parallele).
- Non mesure : la serialisation du document et des faits (le banc ne la chronometre pas ; elle n'est pas
  dans les 90 % du decodage).

Reference de la passe profilee (ms) : film01 decodage 2 155 ; film04 23 009 ; film06 19 904 ; film09 11 263 ; film12 4 220.

## 1. Part de chaque etape (cumule, en parts des echantillons CPU)

Les lignes `dont` sont contenues dans la ligne au-dessus ; les lignes `Bit a bit` et `GC` traversent
les etapes (elles sont des sous-ensembles d'etapes, pas des etapes en plus).

| etape | film01 | film06 | film09 | film04 | film12 |
|---|---|---|---|---|---|
| echantillons CPU totaux (ms) | 2 600 | 20 610 | 11 710 | 25 890 | 4 750 |
| Marche des trames (`(*marcheurDesTrames).parcourir`) | 23,5 % | 38,6 % | 37,1 % | 47,5 % | 43,6 % |
| &nbsp;&nbsp;dont localisation de la liste d'evenements (`LocaliserBoucleDeRecords`) | 0 | 32,6 % | 31,2 % | 28,2 % | 31,6 % |
| &nbsp;&nbsp;dont lecture d'essai de la vue A (`lectureDEssai`, avec `World.Snapshot`) | 7,3 % | 3,2 % | 2,3 % | 9,3 % | 2,7 % |
| Marche des images-cles (`(*marcheDesImagesCles).parcourir`) | 36,9 % | 7,3 % | 9,8 % | 10,1 % | 22,9 % |
| &nbsp;&nbsp;dont recherche d'ancres (`kfRecherche.suivante`) | 14,6 % | 3,0 % | 5,6 % | 3,7 % | 4,8 % |
| Cloture du canal des lectures de bipedes (`canalDesLecturesBipedes.Clore`) | 0,8 % | 12,2 % | 9,5 % | 3,2 % | 2,5 % |
| Balayage hors marche : positions (`balayerPositions`) | 20,4 % | 4,0 % | 3,7 % | 6,6 % | 14,5 % |
| Balayage hors marche : monde, vehicules et placements (`balayerMonde`) | 4,6 % | 31,5 % | 30,1 % | 9,2 % | 8,8 % |
| Balayage hors marche : pont, grenades et sieges (`balayerPont`) | 5,8 % | 2,9 % | 3,5 % | 1,9 % | 5,1 % |
| Balayage hors marche : capacites, equipement (`balayerCapacites`) | 1,5 % | 0 | 0 | 10,0 % | 0 |
| Publication (`assembler`) | 9,2 % | 1,1 % | 1,2 % | 5,2 % | 5,1 % |
| Bit a bit : ancrage des bipedes (`walkDeltaBipedPayloadJusqua`) | 0 | 19,6 % | 15,1 % | 3,8 % | 3,4 % |
| Bit a bit : tirs et visee (`scanBipedAimRecordsJusqua`) | 0 | 6,8 % | 7,0 % | 1,7 % | 1,7 % |
| Bit a bit : recuperation d'equipement (`scanEquipRecoveryPacket`) | 1,5 % | 0 | 0 | 9,9 % | 0 |
| Lecture de bits, a plat (`internal/source.*`) | 32,3 % | 20,5 % | 24,8 % | 20,5 % | 20,2 % |
| GC : `runtime.gcBgMarkWorker` | 6,9 % | 0,9 % | 0,7 % | 2,4 % | 4,6 % |
| Allocation : `runtime.mallocgc` (assists de GC compris) | 5,0 % | 3,4 % | 4,6 % | 6,6 % | 5,3 % |
| Tri (`slices.Sort*`) | 3,1 % | ~0 | ~0 | 0,5 % | 0,6 % |
| Serialisation | non mesuree (hors passe) | | | | |

Lecture :

- Le ramasse-miettes pese de 1 a 7 % (marquage en fond) et l'allocation de 3 a 7 % : il n'est le
  probleme d'aucun film. Volume alloue sur la passe : film01 4,3 Gio, film06 8,6 Gio, film09 4,9 Gio,
  film04 27 Gio, film12 4,7 Gio.
- Le tri est negligeable. Le chargement et la decompression ne sont pas dans cette passe (leur part du
  releve : 0,1 a 0,8 % du total).
- Sur film01 le temps est reparti sur trois blocs a parts voisines (marche des trames, marche des
  images-cles, positions) : aucun point chaud. Sur les quatre autres films, la localisation de la liste et
  les balayages bit a bit (ancrage, tirs, equipement) mangent de 37 % (film12, hors positions) a 59 % (film06).

Premiere etape publiee par l'observateur (ms depuis le debut de la passe) : film01 1 577 sur 2 181
(la marche precede tout), film06 11 975 sur 19 931, film09 7 016 sur 11 333, film04 16 174 sur 23 076,
film12 3 086 sur 4 247 : les 60 a 72 % du decodage sont la marche, le reste les balayages qui la suivent.

## 2. Les 15 fonctions les plus couteuses, par film

Les listes completes, cumulees (`*.cpu.cum.txt`) et a plat (`*.cpu.flat.txt`), sont a cote de ce fichier.
Pour chaque film : cumule (hors `main`, `runtime.main`, `BuildFromFilmAvecFaits`, `scanFilmInputs`), puis a plat.

### film01 (reference saine)

Cumule : `ScanMarcheDesTramesAvec` 1,24 s ; `marcheDesImagesCles.parcourir` 0,96 ; `Distribuer` 0,85 ;
`balayerPositions` 0,53 ; `marcheurDesTrames.parcourir` 0,61 ; `kfRecherche.suivante` 0,38 ; `PremierMotif64` 0,36 ;
`lireTrameParRangs` 0,35 ; `decodeInferLoop` 0,29 ; `lirePontDIdentite` 0,26 ; `canalDeLEtatCompletBipede.ImageCle` 0,24 ;
`assembler` 0,24 ; `TableDeDatums` 0,22.
A plat : `source.jugement.huit` 0,30 (11,5 %) ; `source.BitsAt` 0,16 ; `runtime.mapaccess1_fast32` 0,13 ;
`runtime.memmove` 0,10 ; `balayageDeTable.enteteA` 0,09 ; `maps.ctrlGroup.matchH2` 0,09 ; `ChercherMotifs64` 0,08 ;
`Bits.ReadBits` 0,07 ; `BitsBourres` 0,06 ; `aeshashbody` 0,05.
Par etape : lecture de bits (`jugement.huit`, `BitsAt`, `ChercherMotifs64`, `ReadBits`) ; marche des trames
(`lireTrameParRangs`, `decodeInferLoop`) ; marche des images-cles (`kfRecherche.suivante`, `TableDeDatums`,
`enteteA`) ; recherche d'un etat (`motsParRecord` : 0,23 s, dont 0,20 en `mapaccess1_fast32`) ; distribution
(`Distribuer`) ; GC (`gcBgMarkWorker` 0,18 s) ; publication (`assembler` 0,24 s).

### film06 (version 31, la plus ancienne)

Cumule : `ScanMarcheDesTramesAvec` 11,37 s ; `Distribuer` 10,73 ; `marcheurDesTrames.parcourir` 7,96 ;
`LocaliserBoucleDeRecords` 6,71 ; `balayerMonde` 6,50 ; `TryDeltaAt` 5,92 ; `marchLocateStrict` 4,85 ;
`recupererLaListe` 4,84 ; `marchSignature123` 4,60 ; `walkDeltaBipedPayloadJusqua` 4,03 ; `decodeFilmVehicleScan` 3,89 ;
`decodeDelta` 3,70 ; `matchBipedHeader` 3,55 ; `debutDeLaVueBDeCuisson` 2,82 ; `traverseComponentLoopFrom` 2,81.
A plat : `source.BitsAt` 2,08 (10,1 %) ; `matchBipedHeader` 1,74 ; `source.BitsTolerants` 0,68 ;
`passeDesCreations.payload` 0,50 ; `TryDeltaAt` 0,49 ; `walkDeltaBipedPayloadJusqua` 0,47 ; `LecteurSur` 0,45 ;
`archetypeDeLEnTeteNEW` 0,43 ; `bigEndian.Uint64` 0,42 ; `I0Layout.TotalBits` 0,41 ; `Bits.ReadBits` 0,41 ;
`matchAimOnlyRecord` 0,40 ; `marchSignature123` 0,39 ; `MouvementParDefaut` 0,34 ; `BitsStricts` 0,34.
Par etape : lecture de bits (`BitsAt`, `BitsTolerants`, `BitsStricts`, `ReadBits`) ; ancrage bit a bit
(`matchBipedHeader`, `walkDeltaBipedPayloadJusqua`, `matchAimOnlyRecord`) ; recherche d'un etat / localisation
(`LocaliserBoucleDeRecords`, `TryDeltaAt`, `marchLocateStrict`, `marchSignature123`, `decodeDelta`,
`traverseComponentLoopFrom`) ; relecture d'une liste (`recupererLaListe`) ; distribution (`Distribuer`,
`canalDesLecturesBipedes.Clore` 2,51 s, `canalDesVitalites.Trame` 2,63 s, `canalDesMorts.Trame` 2,23 s) ; allocation
(`mallocgc` 0,71 s) ; GC 0,19 s.

### film09 (version 33)

Cumule : `ScanMarcheDesTramesAvec` 6,28 ; `Distribuer` 5,62 ; `marcheurDesTrames.parcourir` 4,34 ;
`LocaliserBoucleDeRecords` 3,65 ; `balayerMonde` 3,53 ; `TryDeltaAt` 3,24 ; `recupererLaListe` 2,73 ;
`marchLocateStrict` 2,67 ; `marchSignature123` 2,50 ; `decodeDelta` 2,31 ; `traverseComponentLoopFrom` 1,91 ;
`decodeFilmVehicleScan` 1,89 ; `walkDeltaBipedPayloadJusqua` 1,77 ; `decodeFilmPlacements` 1,63 ; `matchBipedHeader` 1,60.
A plat : `BitsAt` 1,22 (10,4 %) ; `matchBipedHeader` 0,71 ; `jugement.huit` 0,46 ; `BitsTolerants` 0,44 ;
`passeDesCreations.payload` 0,31 ; `semawakeup` 0,27 ; `ReadBits` 0,26 ; `bigEndian.Uint64` 0,22 ; `TryDeltaAt` 0,21.

### film04 (version 39, 69 morceaux)

Cumule : `ScanMarcheDesTramesAvec` 14,60 ; `Distribuer` 13,62 ; `marcheurDesTrames.parcourir` 12,29 ;
`LocaliserBoucleDeRecords` 7,31 ; `marcherLePaquet` 6,79 ; `TryDeltaAt` 6,70 ; `debutDeLaVueBDeCuisson` 5,24 ;
`marchLocateStrict` 5,08 ; `marchSignature123` 4,93 ; `recupererLaListe` 4,87 ; `decodeDelta` 4,82 ;
`localiserLaListe` 4,67 ; `traverseComponentLoopFrom` 4,07 ; `BitsAt` 3,01 ; `scanEquipRecoveryPacket` 2,57 ;
`balayerCapacites` 2,58 ; `balayerMonde` 2,37 ; `lectureDEssai` 2,40 ; `World.Snapshot` 1,73.
A plat : `BitsAt` 2,42 (9,4 %) ; `TryDeltaAt` 0,57 ; `scanEquipRecoveryPacket` 0,56 ; `memclrNoHeapPointers` 0,55 ;
`traverseComponentLoopFrom` 0,50 ; `decodeDelta` 0,40 ; `mapaccess1_fast32` 0,31 ; `mapassign_fast32` 0,26.
GC : `gcBgMarkWorker` 0,61 s (2,4 %) ; `mallocgc` 1,71 s (6,6 %).

### film12 (version 37)

Cumule : `ScanMarcheDesTramesAvec` 2,69 ; `Distribuer` 2,46 ; `marcheurDesTrames.parcourir` 2,07 ;
`LocaliserBoucleDeRecords` 1,50 ; `TryDeltaAt` 1,30 ; `marchLocateStrict` 1,14 ; `marcheDesImagesCles.parcourir` 1,09 ;
`marchSignature123` 1,04 ; `recupererLaListe` 0,92 ; `decodeDelta` 0,82 ; `debutDeLaVueBDeCuisson` 0,76 ;
`balayerPositions` 0,69 ; `traverseComponentLoopFrom` 0,62 ; `canalDesVitalites.Trame` 0,57.
A plat : `BitsAt` 0,35 ; `mapaccess1_fast32` 0,21 ; `TryDeltaAt` 0,14 ; `ReadBits` 0,10 ; `gcBgMarkWorker` 0,22 s.

## 3. Les trois anomalies

### 3.1 film06 (8 fois film01) et film09 (4 fois film01 avec moins de morceaux) : la liste d'evenements se LOCALISE A TATONS sur ces films

Ce n'est ni une boucle quadratique, ni une relecture depuis le debut a chaque image-cle : c'est un CHEMIN
PROPRE AUX ANCIENNES VERSIONS, parcouru a chaque trame.

La cause, ligne par ligne :

1. `internal/grammar/vue_a_lecture.go:116-150` (`lireLaVueA`) : la vue A n'est « portee » (`out.Porte`, ligne 138) que si
   la lecture atteint son terminateur ; elle s'arrete des le premier message quand `lisible` est faux
   (`vue_a_lecture.go:129` : `lisible = br.ReadBit() && g.classe != vueAIllisible`) ou quand un message n'est pas lu
   (`lireUnMessage`). `internal/grammar/vue_a_versions.go:143-181` (`tableDesGenresDuFilm`, `classeDesGenres`) : un film est
   de classe « illisible » quand la table de genres qu'il declare n'est ni egale ni prefixe de la table native. La classe
   des films 06, 09 et 12 n'a pas ete lue directement (cause la plus probable, non verifiee) ; ce qui est mesure, c'est
   que leur vue A n'est jamais portee (point 2).
2. `internal/grammar/localisateur.go:214-217` (`debutParLaVueA`) : `if !a.Porte { return -1 }`. Mesure par
   compteurs de blocs : `a.Porte` est faux pour 100 % des paquets a evenements de film06 (7 779 sur 7 779),
   film09 (4 956 sur 4 956) et film12 (6 056 sur 6 056). Sur film01 (classe prefixe) la vue A decide les
   7 751 paquets sur 7 751 par `lectureDEssai`, et `localiserLaListe` n'est appele aucune fois.
3. `internal/grammar/marche_trames.go:229-236` (`debutDeLaVueBDeCuisson`) renvoie alors a `localiserLaListe`
   (`internal/grammar/debut_de_liste.go:50`), qui appelle `LocaliserBoucleDeRecords` (`localisateur.go:144`) puis
   `marchLocateStrict` (`localisateur.go:170-182`) : une boucle sur CHAQUE bit du payload (`for s := 2; s+largeur < nb; s++`)
   qui, des que le bit precedent est nul, decode un delta complet par `TryDeltaAt` (`frame_records.go:233`,
   `decodeDelta`, `traverseComponentLoop`) pour y chercher la signature du slot 123.
4. Quand la liste n'a pas ete localisee, les canaux des morts et des vitalites la RECUPERENT, chacun pour son compte :
   `canal_des_morts.go:85` et `canal_des_vitalites.go:84` appellent `recupererLaListe` (`canal_des_morts.go:160`), qui
   relance `LocaliserBoucleDeRecords` en ordre `SignaturePuisLargeurLibre` : une 2e boucle stricte, puis une 3e boucle
   (`marchLocateFallback`, `localisateur.go:187-199`) sur le meme payload, deux fois (un canal chacun).

Comptes (compteurs de blocs, une passe) :

| | film01 | film06 | film09 | film12 | film04 |
|---|---|---|---|---|---|
| paquets a evenements decides par la vue A | 7 751 | 0 | 0 | 0 | 18 823 |
| paquets localises a tatons (`localiserLaListe`) | 0 | 7 779 | 4 956 | 6 056 | 6 174 |
| paquets non localises, relus 2 fois par les canaux (`recupererLaListe` / 2) | 0 | 2 255 | 1 286 | 2 575 | 3 800 |
| appels de `marchLocateStrict` | 0 | 12 289 | 7 528 | 11 206 | 13 774 |
| appels de `marchSignature123` (un `TryDeltaAt` chacun) | 0 | 52,6 M | 27,5 M | 13,0 M | 44,3 M |
| appels de `TryDeltaAt` au total | 175 | 73,5 M | 38,3 M | 18,9 M | 67,9 M |
| appels de `matchBipedHeader` (ancrage bit a bit) | 0,35 M | 332 M | 170 M | 14,3 M | 77,9 M |
| appels de `traverseComponentLoopFrom` | 0,75 M | 5,5 M | 2,7 M | 1,3 M | 6,6 M |

Pour film06 : 6,71 s de 20,6 (32,6 %) dans `LocaliserBoucleDeRecords`, dont 4,79 s (71 %) dans la relecture par les
canaux (`debutRecupere`, `localisateur.go` via `canal_des_morts.go:160`), et 1,92 s dans la localisation de la marche. Le
film06 a 17 919 trames contre 16 824 pour film01 : la taille n'explique rien, le chemin explique tout. Pour film09 :
31,2 % en localisation, 22 % en balayages bit a bit (ancrage 15,1 % + tirs 7,0 %).

Le second facteur est un balayage bit a bit des payloads que la marche n'a pas lus : `positionsDeVehicule`
(`positions_de_vehicule.go:38-76`) rejoue `walkDeltaBipedPayloadJusqua` (`delta_biped_walk.go:91`) jusqu'a
`finSelonLesTrames` ; plus un payload est mal lu par la marche (liste non localisee), plus ce reste est long :
`matchBipedHeader` est appele 332 M fois sur film06 (9 265 bits par appel du marcheur) et 0,35 M fois sur film01
(10 bits par appel). Les memes balayages servent `ScanBipedAimOnly` (`offline_aim_only.go:110`, `payloadEntier`) et
`canalDesLecturesBipedes.recuperer`. `balayerMonde` (vehicules et placements) passe ainsi de 4,6 % a 31,5 % sur film06.

Sans ces deux chemins (localisation a tatons et balayages bit a bit : 59 % de film06, 53 % de film09), film06
retomberait vers 8,4 s d'echantillons, contre 2,6 pour film01 ; le reste est encore le cout de decodages d'essai
(`traverseComponentLoopFrom` 5,5 M appels contre 0,75 M).

Pourquoi film09 est plus lent que film01 avec moins de morceaux : ses 21 morceaux portent 11 130 trames mais ses
payloads sont plus longs et sa vue A est illisible : 4 956 localisations a tatons, 170 M ancrages de bit (7 640 bits par
appel du marcheur contre 10 sur film01).

Pour film12 (version 37, aussi sans vue A) : 31,6 % en localisation, mais ses payloads sont courts et la marche lit
presque tout (144 bits par appel du marcheur) : le chemin existe, l'ecart a film01 est de 2 a 3 fois seulement.

### 3.2 film04 : 2,2 Gio de tas

Ce que le profil du tas dit (prise du tas au plus haut observe, `film04.tas.pprof.pic`, 819 Mio vivants a ce
moment ; echantillonneur : tas en usage au plus haut 1 677 Mio ; mesure de reference du banc 2,2 Gio, qui compte la
memoire demandee a l'OS, ramasse-miettes compris) :

| structure vivante au plus haut | Mio | allouee ou |
|---|---|---|
| copie de la suite des positions apres `DropIsolated` | 282 | `offline_filters.go:34-60` (`DropIsolated`, appel `offline_biped_band.go:147`) |
| suite des positions de bipedes lue | 244 | `positions_lues.go:168-190` (`lireLesPositions`, `append` sans pre-dimensionnement) |
| octets du film (morceaux) | 87 | `os.ReadFile` du chargement |
| ajouts de positions (`positionsBipedes.noter`, `ajouterLePaquet`) | 40 + 37 | `positions_lues.go`, `offline_biped.go` |
| vitesses d'etats de mouvement | 32 | `(*movementStateScanner).vitesse` |

Film04 compte 1 317 266 positions de bipedes (compteur de bloc `positions_lues.go:185-191`), a environ 130-190 octets
chacune (`BipedPosition`, `offline_biped.go:65`, avec directions et vitalites). A un meme instant vivent : la suite lue
(`lireLesPositions`), sa copie filtree par `DropIsolated` (qui recopie tout, 1 317 251 gardees sur 1 317 266), puis la copie
de `DropTeleportsExcept` (`offline_filters.go:131`, appel `offline_biped_band.go:151`). Les trois copies d'une suite quasi
identique expliquent la majeure partie des quelque 820 Mio vivants ; le ramasse-miettes (reglage par defaut : le tas peut
atteindre le double du vivant avant collecte) porte la mesure vers 1,7 a 2,2 Gio.

Volume alloue (27 Gio sur la passe, `alloc_space`) : `World.cloneSlots` 6,2 Gio (23 %, `world.go:403-405`,
161 445 appels, une copie complete de la table des slots par `Snapshot`) ; `traverseComponentLoopFrom` 4,1 Gio
(`traverse.go:348`, les `Comps` des decodages d'essai) ; `replay.indexBySlot` 3,3 Gio (`shots.go:174`, construit 4 fois :
`buildOwners`, `buildShots`, `buildGrenades`, `applyTo`) ; `vehiclePositionsBySlot` 1,7 ; `DropIsolated` 1,4 ;
`lireLesPositions` 1,3 ; `DropTeleportsExcept` 1,1. Ce sont des allocations temporaires, pas la rétention du pic.

Pas un probleme de ramasse-miettes en temps (2,4 % de CPU) : un probleme de PIC (trois copies simultanees d'une suite de
1,3 M de positions), qui compte parce que le plafond de memoire est de 2 a 3 Gio.

## 4. Axes du plan : ce que le profil permet d'en attendre

Gain exprime en part du total (temps), au meilleur cas ; la machine compte 16 coeurs.

### Axe 1 : decompression sur plusieurs fils (V1) : gain au plus 0,1 a 0,8 %

Decompression : 3,5 a 11,6 ms ; chargement (lecture, decompression, decoupage) : 9 a 31 ms ; ensemble 0,08 % (film06)
a 0,8 % (film01) du total. Meme a cout nul le gain plafonne a ce chiffre. **Abandonner.**

### Axe 2 : plusieurs morceaux a la fois (V2) : au plus 10 a 17 % apres les correctifs de localisation ; risque eleve

- La marche des trames pese 23,5 % (film01) a 47,5 % (film04) des echantillons ; elle contient les couts de la
  localisation (voir 3.1), qui disparaitraient d'abord par des correctifs plus simples.
- Est-elle sequentielle par nature ? Elle porte UN monde pour tout le film (`marche_trames.go:109-126`,
  `monde: NewWorld(reg)` ; boucle `parcourir`, lignes 132-150) : la table des slots, ses liaisons (NEW/DEL) et les
  `Snapshot`/`Restore` passent d'un morceau au suivant. L'image-cle du morceau (`lierLesImagesClesDuChunk`, ligne 144)
  re-lie des slots a chaque debut de morceau, ce qui rend la dependance PARTIELLE : un slot absent de l'image-cle
  depend encore des morceaux precedents. Que ce soit suffisant n'est pas etabli (voir section 7).
- Gain theorique a 4 fils si la marche seule se parallelise parfaitement : `part x 3/4` = 17,6 % (film01),
  29 % (film06), 35,6 % (film04) sur l'etat actuel ; une fois la localisation corrigee (axe 3), la marche tombe vers
  20-30 % et le gain vers 10-17 %. Chaque relecture en sequence (morceau qui depend du precedent) le reduit encore ;
  la memoire monte avec le nombre de copies d'etat (film04 est deja a 1,7-2,2 Gio).
- **A reduire** : a decider sur un nouveau profil, apres l'axe 3 et le parallelisme des balayages independants
  (section 5, G3), qui rend davantage pour moins de risque.

### Axe 3 : lectures refaites (V3) : gain reel 10 a 28 % sur les anciennes versions, pas sur film01

Ce que le profil montre de refait, avec le cout :

| lecture refaite | cout (part des echantillons) | verdict |
|---|---|---|
| liste localisee 3 fois (marche + canal des morts + canal des vitalites), puis tentee au repli 2 fois (`recupererLaListe`) | film06 4,8 s = 23 %, film09 2,7 s = 23 %, film04 4,9 s = 19 %, film12 0,9 s = 19 % ; film01 0 | A FAIRE : voir G2 |
| recherche des ancres d'images-cles pour les trois lecteurs (V3.2) | `kfRecherche.suivante` 14,6 % (film01) mais UNE seule recherche par payload : `keyframe_world_marche.go:85-104` memoise par payload (`mem.parPayload` : 556 appels, 29 recherches reelles sur film01 ; compteur de `marcherLaTable` = 29/30/42 payloads) | ABANDONNER : deja fait ; l'ancien chiffre « 30 % » (registre des reports) est perime |
| copie complete de la table des slots a chaque essai (`World.Snapshot`, `world.go:403-420`) | film04 6,7 %, film06 2,8 %, film01 2,7 %, film12 2,5 %, film09 1,7 % ; 23 % du volume alloue de film04 | A REDUIRE : voir G5 |
| `indexBySlot` x4 en publication (`shots.go:174`) | CPU 1,0 % (film04) ; 12 % du volume alloue | ABANDONNER pour la vitesse (memoire seulement) |
| sonde de calibration (V3.3) | `CalibrateMPPWidthsOf` 1,5 % sur film01, moins de 1 % ailleurs | `[!]` : sous le seuil de 1 % sauf film01 (40 ms) |
| recuperation d'equipement (V3.3) | film04 9,9 % (`scanEquipRecoveryPacket`, `equipment_recovery.go:210`) ; 0 a 1,5 % ailleurs | A FAIRE sur film04 : voir G7 |

Gain de l'axe tel que le plan l'ecrit (V3.2 + V3.3) : 0 a 1,5 % sur la plupart des films, 10 % sur film04. Le gain
reel de « lectures refaites » est ailleurs : G2 (et G1).

### Axe 4 : lecture au choix des canaux (V4) : de 13 % (film01) a 60-70 % (film06, film09) pour un appelant a un seul canal

La marche (trames et images-cles) est necessaire a tout canal : elle fabrique le monde. Ce qu'un appelant qui ne veut
qu'un canal peut sauter, par canal :

- balayages hors marche qu'il ne lit pas : `balayerMonde` (4,6 % a 31,5 %), `balayerPont` (1,9 a 5,8 %),
  `balayerCapacites` (10 % film04), `balayerPositions` (4 a 20 %), publication partielle ;
- la cloture du canal des lectures de bipedes (`Clore`, 12,2 % film06) ;
- la recuperation de liste par les canaux des morts et des vitalites (`recupererLaListe`, 19-23 % sur les anciennes versions) :
  un appelant qui ne lit ni morts ni vitalites la saute entierement.

Estimation pour un appelant « positions seules » (ne garde que `balayerPositions` et la marche) :
saute monde + pont + capacites + `Clore` + recuperation : film01 ~13 %, film12 ~36 %, film04 ~43 %, film09 ~66 %,
film06 ~70 % (parts cumulees non recouvrantes de la section 1). Pour « morts seuls » (garde la recuperation, saute positions,
monde, pont, capacites ; avec ou sans `Clore`) : film06 38 a 50 %, film09 37 a 47 %, film04 28 a 31 %, film01 32 a 33 %. Verdict : **a faire**, mais son
gain est celui d'un APPELANT PARTICULIER : le document complet (cas de LevelUp) n'en tire rien.

## 5. Gains que le plan ne prevoyait pas

Classes par rapport gain sur risque. Les gains ne s'additionnent pas toujours (G1 et G2 portent sur les memes boucles).

| rang | gain | fonction ou lieu | estimation | risque |
|---|---|---|---|---|
| G2 | Ne localiser qu'une fois par trame non localisee, pour les deux canaux (morts, vitalites) | `canal_des_morts.go:160-177` (`recupererLaListe`), appels `canal_des_morts.go:85` et `canal_des_vitalites.go:84` | 100 % redondant entre les deux canaux (meme payload, meme monde apres `Restore`) : film06 2,4 s (11,6 %), film09 1,35 s (11,5 %), film04 2,4 s (9,4 %), film12 0,46 s (9,7 %) ; en plus, reutiliser le verdict de la marche (`localiserLaListe` strict deja fait) : +4 a 7 % si le monde est identique (a verifier) | tres faible : meme resultat, memoire par paquet |
| G1 | Ne decoder un delta d'essai que si l'identifiant porte le slot 123 (ou la generation du monde) | `localisateur.go:162-167` (`marchSignature123`) et `localisateur.go:187-199` (`marchLocateFallback`), via `TryDeltaAt` `frame_records.go:233` : la condition finale est une conjonction dont les premiers termes (`rec.Slot == 123`, generation) se lisent dans l'en-tete, avant `decodeDelta` | `TryDeltaAt` pese 5,9 s sur film06 dont 3,5 s en `decodeDelta` et 1,5 s en `LecteurSur`/`poserCadre` ; -17 a -24 % sur film06, -18 a -23 % film09, -14 a -18 % film04, -17 a -20 % film12 ; supprime aussi 2,7 Gio d'allocations (`traverse.go:348`) ; film01 : 0 | faible a moyen : meme booleen, a prouver par comparaison des positions rendues sur le corpus |
| G7 | Tester slot et generation avant de lire la poignee complete a chaque bit | `equipment_recovery.go:210-260` (`scanEquipRecoveryPacket`, `LireHandleDelta` 1,15 s de 2,57 s) | film04 : jusqu'a -5 a -7 % | faible |
| G4 | Avance par mot (saut des bits nuls) dans les balayages bit a bit | `delta_biped_walk.go:91-107`, `offline_biped.go:318` (`matchBipedHeader`), `offline_aim_only.go:110` ; lecture de bits a plat 20-25 % des echantillons | -5 a -9 % sur film06, film09 (332 M et 170 M d'appels) | moyen (la grammaire d'origine avance « d'un bit » : l'equivalence se prouve par les memes sorties) |
| G3 | Faire tourner les balayages d'apres la marche en parallele entre eux (et `decodeFilmVehicleScan` avec `decodeFilmPlacements` dans `balayerMonde`) | `replay/film_scan.go` (`balayerPositions`, `balayerMonde`, `balayerPont`, `balayerCapacites`) | ces balayages pesent 28 % (film04, film12) a 38 % (film06) des echantillons, mais le plus long d'entre eux borne le gain : -8 a -20 % de duree murale a plusieurs fils (film06 : 38 % -> ~19 %, borne par `decodeFilmVehicleScan` 3,9 s ; film01 : ~20 %) ; sans effet a 1 fil (DD-3) | moyen : etat partage du `FilmContext` (caches, verrous) a verifier ; les crochets publient |
| G5 | Eviter la copie complete de la table des slots a chaque essai | `world.go:403-420` (`cloneSlots`, 161 445 appels sur film04) ; `debut_de_liste.go:225` (`lectureDEssai`) | -2 a -7 % de CPU, -6 Gio alloues sur film04 (-23 %) ; copie a la demande ou journal d'annulation | moyen (invariant de `Restore`) |
| G6 | Ne pas tripler la suite des positions au plus haut | `offline_filters.go:34-60` (`DropIsolated`), `offline_biped_band.go:147-151`, `positions_lues.go:168` (`lireLesPositions`, pre-dimensionner) ; une variante interne de filtrage en place (les fonctions exportees gardent leur contrat) | memoire : environ -280 a -560 Mio vivants sur film04, soit de 0,4 a 0,6 Gio de moins au plus haut mesure ; temps : ~1 % | faible |
| G8 | Index de slots par tableau plutot que par `map[uint32]` | `runtime.mapaccess1_fast32` 8,5 % film01, 7,8 % film12 ; `motsParRecord` (`keyframe_loadout.go:164`) 0,23 s de 2,6 s sur film01 | -4 a -6 % sur film01 et film12 | faible, effort moyen |

Ordre de bataille propose (gain / risque) : G2, puis G1 (les deux ensemble : -25 a -32 % sur film06, film09, film12 ; la
localisation passe de 31-33 % a moins de 8 %), G6 (memoire de film04), G7, G4, G8 ; G3 et G5 ensuite, apres un nouveau
profil.

Le ramasse-miettes (1 a 7 %) ne vaut aucun reglage : le monter en GOGC achete du temps contre du pic memoire,
alors que film04 est deja a 2,2 Gio.

## 6. Verdict des axes du plan

| axe | gain attendu | risque | verdict |
|---|---|---|---|
| V1 decompression sur plusieurs fils | 0,1 a 0,8 % du total | faible | abandonner |
| V2 plusieurs morceaux a la fois | 10-17 % apres G1/G2 (35 % theorique sur film04 avant) | eleve (monde unique, ordre des crochets, memoire) | a reduire : reporter, decider sur un nouveau profil ; G3 d'abord |
| V3 lectures refaites : V3.2 (une recherche d'images-cles) | 0 % : deja memoisee | - | abandonner |
| V3 lectures refaites : V3.3 (sonde, equipement) | sonde 0-1,5 % ; equipement 10 % sur film04 | faible | a faire pour l'equipement (G7), sonde `[!]` |
| V3 elargi : G2 et G5 | G2 : 9-12 % ; G5 : 2-7 % | tres faible ; moyen | a faire (G2), a reduire (G5) |
| V4 lecture au choix des canaux | 13 % (film01) a 70 % (film06) pour un appelant a un canal ; 0 % pour le document complet | moyen | a faire, ecrit pour l'appelant du lecteur 3D |

## 7. Ce que je n'ai pas pu etablir

- Que le monde est le meme au moment de la localisation de la marche et de la recuperation par les canaux : G2 est
  sur (meme trame, `Restore` entre les deux), le prolongement vers la marche demande une comparaison.
- Si un morceau peut se decoder sans l'etat du precedent (V2) : la lecture de `marche_trames.go` montre un monde
  unique partiellement re-lie par l'image-cle ; l'independance ne se prouve que par un essai (chunk decode a part
  contre chunk decode en suite), pas lu ici.
- La part d'anciennes versions dans le parc de films : le profil ne la donne pas ; les gains de G1 et G2 valent pour
  les films dont la vue A est illisible (versions 31 a 37 mesurees) et, en partie, pour film04 (version 39 : 1 313
  paquets sans vue A portee et 4 861 qui ne se ferment pas).
- Le cout reel du ramasse-miettes en duree murale (le marquage se fait sur d'autres coeurs) : le profil dit sa part de
  CPU, pas son effet sur le temps ecoule.
- Le pic exact de film04 en memoire demandee a l'OS : le tas profile est celui du dernier ramasse-miettes ; l'ordre de
  grandeur (820 Mio vivants, 1,7 Gio de tas en usage, 2,2 Gio mesures) est coherent, la repartition fine ne l'est pas.
- L'effet du profileur : la passe profilee est plus longue (film06 23 s contre 17,9 s au releve) ; seules les parts
  sont fiables.
- La serialisation du document et des faits : hors du banc, non profilee.
- Les fichiers de profil portent un identifiant de construction contenant un chemin du poste : ils restent hors depot.

## Fichiers

`filmNN.cpu.pprof`, `filmNN.tas.pprof`, `filmNN.tas.pprof.pic`, `filmNN.cpu.cum.txt`, `filmNN.cpu.flat.txt`,
`filmNN.cpu.txt` et `filmNN.tas.txt` (observations d'etapes et durees de chaque passe), `etapes.md` (table des etapes),
`couv/filmNN.cov` (compteurs de blocs), `etapes.py`, `plat.py`, `appels.py`, `appels2.py` (extraction), `run.sh`, `runc.sh`.
Lecture : `go tool pprof -top -cum filmNN.cpu.pprof` ; `go tool pprof -sample_index=alloc_space -top filmNN.tas.pprof`.
