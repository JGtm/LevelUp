package facts

// rev_chronique.go — LA CHRONIQUE DE LA REVISION DES FAITS, ET RIEN D AUTRE.
//
// SORTIE DE `rev.go` PAR DEPLACEMENT PUR le 2026-09-21 (lot 5.7) : le fichier passait 500
// lignes au moment ou l entree `.3` s y ajoutait, et le ratchet de taille
// (`archlint/film_file_size_test.go`) refuse de grandir. AUCUN OCTET DE CODE N EST TOUCHE — le
// fichier ne porte que des commentaires, exactement comme `grammar/rev_chronique.go`, dont ce
// decoupage reprend la forme. La regle de la chronique est INCHANGEE : une entree par rang,
// jamais reecrite, et la revision se decide AVANT le golden.

// # LA CHRONIQUE — UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// ENTREE `killsource-2026-09-16.7` (2026-09-17, lot 3.3.1) : LA REVISION MONTE DERRIERE LA
// GRAMMAIRE, ET LA SORTIE DU DECODEUR CHANGE — LES LANCERS DE GRENADE DES BUILDS ANCIENS.
// Aucun octet de `facts/` n est ecrit autrement ; ce qui monte
// est la VALEUR de `grammar.Rev` (elle-meme derriere `profile.Rev`), que cette revision hache —
// le sens unique des quatre couches joue exactement comme il est ecrit.
//
// CE QUI A CHANGE EN AMONT : l amorce du record de creation de projectile devient une donnee de
// PROFIL, keyee par les neuf clefs du depot (sept builds, deux versions majeures sans section
// d identification). Le balayage comparait 24 bits sur TOUS les films ; sur les builds anterieurs
// a `HI_1_12_0` son vingt-quatrieme bit est le bit de poids fort de l identifiant, d ou ZERO
// lancer publie sur cinq temoins du corpus. Mesure du volet recherche et de ce lot : 1 282
// lancers sur ces cinq temoins, plus `a521164d` (105), `11de8353` (145) et `50247b26` (95).
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE. Les lignes de
// `match_kill_events` deja en base portent la revision anterieure et deviennent CANDIDATES au
// redecodage (`conditionBacklog`, `sync/killcollector/postsync.go`) ; le redecodage du parc reste
// un geste de PRODUCTION, pris par le pilote. Les lancers de grenade ne sont pas des kill-events
// — ce qui change reellement pour `killsource` est la revision qu il estampille, pas ses lignes.
//
// `SchemaVersion` NE MONTE PAS : la FORME du document de rejeu est inchangee (aucun champ ajoute,
// la couverture du balayage est journalisee et ne voyage pas dans l artefact). Ce qui change est
// le CONTENU des artefacts des films anciens, et c est ce que le corpus gate mesure.
// ENTREE `killsource-2026-09-17` (2026-09-17, lot 3.4.1) : LA REVISION MONTE, `.7` -> le rang
// du jour.
// LA SORTIE DES FAITS CHANGE, ET C EST LE BUT DU LOT.
//
// TROIS CAUSES, chacune mesurable :
//
//	LA GRAMMAIRE   `grammar.Rev` passe au `.40` (le chemin absolu d i0 lit les largeurs et les
//	               bornes de la table PAR INDEX de la carte au lieu d une largeur UNIFORME de
//	               14 bits ; correctif D1 (3.4) sur la regle d emission). `facts.Rev` hache sa
//	               VALEUR : elle monterait meme si rien de `facts/` n avait bouge.
//	LA CALIBRATION `killsource/calibrate.go` : l inference des largeurs NE DECIDE PLUS. Les
//	               largeurs viennent du profil — c est-a-dire du catalogue de la carte, dont la
//	               loi est verifiee 79 cartes sur 79 — et le balayage devient un ORACLE qui
//	               compte les desaccords (`calibration.Desaccords`, publie dans
//	               `Result.Calibration`, qui ne sort pas de la CLI). Arbitrage utilisateur V17,
//	               M3-Q8 : « la valeur LUE prime sur la valeur mesuree ».
//	LA CARTE      `killsource.Decode` recoit desormais l ENTREE DE CATALOGUE de la carte du
//	              match (`Options.Carte`), depuis `replaybuild.BuildBytes` et depuis
//	              `sync/killcollector`. Elle DECIDE les largeurs d axe du chemin absolu de
//	              position, la ou ce paquet etait le seul chemin de decodage du depot a ne
//	              recevoir aucun catalogue et a devoir les inferer. Mesure sur `e5adf7b2`
//	              (Fragmentation, 17/17/15) : la voie MARCHE passe de 13 lignes appariees sur
//	              16 a 167 sur 169, la voie SCAN de 176 a 22, et les 191 morts publiees sur
//	              197 couples reels sont les MEMES des deux cotes — zero perte. Sans carte, le
//	              repli `repli_carte_absente_largeurs_par_defaut` est pose, compte et AVERTI
//	              par film : le decodeur lit alors les largeurs d UNE autre carte, et le dit.
//
// LES LIGNES DE KILL DEJA EN BASE DEVIENNENT CANDIDATES AU BACKLOG DE REDECODAGE, et ce
// backlog part sur SIGNAL UTILISATEUR (D6), JAMAIS automatiquement : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote.
// ENTREE `killsource-2026-09-17.2` (2026-09-17, lot 3.4.2) : LA REVISION MONTE PARCE QUE LE
// BALAYAGE CESSE DE DECIDER CE QU IL NE MESURE PAS.
//
// LA SORTIE DES FAITS CHANGE, et le changement est un RETRAIT DE BRUIT, pas un gain de lecture.
//
// CE QUE `replay-equiv` A MESURE (20 films, sans `-update`, §5 du plan) : le lot 3.4.1 faisait
// bouger `abilityImpulses` sur 7 films, `grappleReads.stats` sur 9 et `pads` sur 3 — 19 ecarts
// hors liste sur 14 films, dont QUATRE lectures publiees perdues. Aucune de ces etapes n aurait
// du bouger.
//
// LA CAUSE, INSTRUITE SUR DEUX FILMS EN LECTURE SEULE. `infererLargeurs` balayait ENSEMBLE la
// largeur d axe et la largeur du mot de poignee, et retenait le COUPLE de meilleur score sous
// une largeur d axe UNIFORME — celle que la production a CESSE de lire au lot 3.4.1, quand les
// largeurs sont passees au triplet de la carte. Le `iw` retenu etait donc l argmax dans un monde
// que le decodeur n habite plus. Et le garde-fou ne pouvait pas le voir : `flatRatio` teste la
// nettete de la largeur d AXE, puis le code prenait le `iw` du MEME gagnant sans verifier qu il
// fut discrimine — une seule mesure, deux grandeurs, un seul garde.
//
//	a521164d  au TRIPLET LU [17 17 15]   iw=1 272 · iw=2 272 · iw=3 272   AVEUGLE
//	          sous l UNIFORME            iw=1 226 · iw=2 226 · iw=3 230   4 records sur 226
//	64e8adfa  au TRIPLET LU [15 15 15]   61 · 61 · 61                     EGALITE PARFAITE
//
// Le critere est AVEUGLE a la grandeur qu il decidait : la valeur publiee roulait sur un ex aequo
// tranche par un `sort.Slice` INSTABLE, et elle voyageait jusqu au rejeu
// (`profilDeBalayageDeLaCuisson`, `replaybuild/kills.go`) ou tous les lecteurs derriere i0 en
// heritaient.
//
// CE QUE CE LOT FAIT : DEUX MESURES SEPAREES. L oracle d axe balaie ses 21 largeurs a mot de
// poignee FIGE et n ecrit rien ; la decision du mot de poignee score ses 3 candidats AU TRIPLET
// LU et ne les retient QUE s ils dominent la mediane d un facteur `flatRatio`. Sur les deux
// films instruits la mesure ne discrimine pas : l invariant 1 tient, sous le repli
// `repli_largeur_mot_de_poignee_inferee` re-motive au registre. Les deux tris sont rendus
// DETERMINISTES (score decroissant, puis largeur croissante).
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote.
//
// `SchemaVersion` NE MONTE PAS : aucun champ n est ajoute au document. `grammar.Rev` et
// `profile.Rev` NE MONTENT PAS : aucun octet de ces deux couches n est touche.
// ENTREE `killsource-2026-09-18` (2026-09-18, lot 5.1.1) : LA REVISION MONTE MECANIQUEMENT,
// `.2` -> le premier rang du 18. AUCUNE SOURCE DE `film/facts/` N EST TOUCHEE PAR CE LOT.
//
// CE QUI LA FAIT MONTER : l empreinte de cette couche hache les VALEURS de `source.Rev` et de
// `grammar.Rev`, et `grammar.Rev` monte au lot 5.1.1 (`grammar-2026-09-18` : l archetype
// `managed-navpoint` ti=12 est lu de `i1` au minuteur manuel, douze lecteurs neufs). La chaine
// est voulue : une grammaire qui change date les lignes deja decodees, meme quand le fait
// publie ne bouge pas encore.
//
// CE QUE LA SORTIE FAIT AUJOURD HUI : rien de plus. Aucun composant porte par 5.1.1 n alimente
// `killsource` — les douze lecteurs servent `ti=12`, que la chaine des morts ne marche pas.
// LE BACKLOG QU ELLE OUVRE EST DONC UN BACKLOG DE DATATION, pas de correction.
//
// C EST L UNIQUE MONTEE DE CETTE CONSTANTE POUR TOUT LE LOT 5.1, ET C EST DELIBERE : le volet
// 5.1.4 (l attribution de la fin de vie des vehicules) CHANGERA vraiment la sortie des faits, et
// il partagera ce rang — deux changements d un meme lot partagent la revision. Ouvrir deux
// backlogs pour un seul lot ferait redecoder le parc deux fois.
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote. UN BACKFILL
// `killsource-2026-09-17.2` TOURNAIT AU MOMENT DE CE LOT : la montee le rend candidat a son
// tour, ce que l utilisateur a accepte en ouvrant le lot (V26).
//
// `SchemaVersion` reste 62 ; `profile.Rev` ne monte pas (aucun octet de `profile/` touche).
// ENTREE `killsource-2026-09-20` (2026-09-20, lot 5.2b.1) : LE ROSTER DU DECODEUR VOIT LES
// REMPLACANTS, ET UN PARTICIPANT NON COMPTE N ETEINT PLUS LE MATCH.
//
// DEUX SOURCES POUR CETTE MONTEE, et elles vont dans le meme sens.
//
//	`facts/killsource/` CHANGE      une TROISIEME lecture d identite entre dans le roster
//	                                (`index_motif.go`) : les cinq bits qui precedent le motif du
//	                                xuid dans les chunks de replication, c est-a-dire ce que le
//	                                rejeu publie sous le nom `PlayerIndexTable`, par le MEME
//	                                resolveur (`weaponv3.ResolveXuidToPI`). Elle voit les joueurs
//	                                qui REMPLACENT un partant en cours de match, que la table de
//	                                `chunk_00` — ecrite a l ouverture du film — ignore.
//	`grammar.Rev` MONTE             `grammar-2026-09-20`, et cette couche hache sa valeur.
//
// CE QUE LA MESURE DIT, SUR `b1ad85eb` (Domicile, HI_1_13_0, 2026-09-20) : la table de
// `chunk_00` nomme HUIT sieges (0..7), BOT_METADATA tient le 8, et le kill-feed nomme un
// NEUVIEME humain — `Claudors` — que rien ne pouvait placer. Le motif du xuid le lit a l indice
// 10, UNANIME sur 22 chunks de replication sur 27, et il CONFIRME les huit sieges de la table
// (`MotifAgree = 8`, zero contradiction). Les huit dead-states hors roster disparaissent, la
// publication ligne par ligne s ouvre, 77 lignes sortent dont 63 a source NOMMEE.
//
// LA TABLE DE `chunk_00` GARDE LA MAIN quand les deux lectures se contredisent : elle est la
// plus eprouvee (314 accords sur 322 sieges, 30 films). Une contradiction se COMPTE
// (`FilmTablePinning.MotifContradict`), elle ne deplace rien — meme doctrine que le controle par
// les votes du kill-feed (D14 b).
//
// TROISIEME CHANGEMENT DE SORTIE, MESURE AU MEME ENDROIT : plusieurs bots declares sur un MEME
// slot ajoutaient chacun un nom au roster pour un seul indice, et les perdants restaient des
// NOMS LIBRES — de la matiere a inference. Deux noms de bot fantomes suffisaient a rendre
// `FilmTablePinning.AffectationUnique` faux des qu un indice se liberait, donc a refermer la
// publication que l epinglage du remplacant venait d ouvrir. Le vainqueur du slot ne change pas
// (le dernier declare) ; la succession REMPLACE le nom en place et se compte
// (`Roster.BotsSuccedes`).
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote. CELUI-CI EST UN
// BACKLOG DE CORRECTION, pas de datation : les matchs a remplacement changent de verdict de
// publication.
//
// `SchemaVersion` NE MONTE PAS : aucun champ n est ajoute au document.

// ENTREE `killsource-2026-09-21` (2026-09-21, lot 5.3.3-a) : LA REVISION MONTE MECANIQUEMENT
// DERRIERE LA GRAMMAIRE — `i60` EST DECLARE COMPLET QUAND LA CARTE EST LA.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21` :
// `SimStateComplet` ne se pose plus a la main, il SUIT les largeurs d axe de la carte du match
// (chronique de `grammar`, entree du meme jour). La traversee du bipede va donc plus loin sur
// tout film dont la carte est cataloguee — 38 desynchronisations d `i60` en moins sur le seul
// `bfecd02b`. Cette constante hache la VALEUR de la revision de grammaire : elle monte
// mecaniquement, et les lignes de `match_kill_events` anterieures deviennent candidates au
// backlog de redecodage (D6, SUR SIGNAL UTILISATEUR, jamais automatiquement).
//
// CE QUE CE BACKLOG RAPPORTERAIT, MESURE AVANT DE L OUVRIR : RIEN. A/B par `replay-build` sur
// `000d5950` et `bcb6d393`, bascule levee puis abaissee, cache de faits vide a chaque passe :
// artefact BIT A BIT IDENTIQUE. Le `replay-equiv` du meme film ne deplace que le digest de
// l etape `killsource`, et ce digest porte la VALEUR du profil calibre — compte et octets du
// kill-feed inchanges. Le pilote n a donc aucune raison de declencher ce backlog pour cette
// revision-ci.
//
// `SchemaVersion` NE MONTE PAS : aucun champ neuf au document, et aucun octet cuit ne change.

// ENTREE `killsource-2026-09-21.2` (2026-09-21, lot 5.3.6) : LA REVISION MONTE DERRIERE UN
// BALAYAGE NEUF DE LA COUCHE GRAMMAIRE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.2` : la couche
// rend une valeur de plus — les ETATS DE MOUVEMENT du Spartan (accroupi, glissade, action de
// mobilite), publies en `stances[]` au schema 65. Cette constante hache la VALEUR de la revision
// de grammaire : elle monte, et les lignes de `match_kill_events` anterieures deviennent
// candidates au backlog de redecodage (D6, SUR SIGNAL UTILISATEUR, jamais automatiquement).
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : rien de neuf. Le calque des etats est
// ADDITIF — un balayage de plus, sur un canal que le kill-feed ne lit pas. C est le CONTENU CUIT
// qui change (`SchemaVersion` 64 -> 65), et c est `backfill-replay` qui le re-cuit, pas ce
// backlog-ci.

// ENTREE `killsource-2026-09-21.3` (2026-09-21, lot 5.7) : LA REVISION MONTE DERRIERE UNE
// CORRECTION DE LARGEUR DE LA COUCHE GRAMMAIRE, ET CELLE-CI DEPLACE DES BITS.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.3` : les
// QUATRE CHARGES de `ti=35 i55 biped-posture-physics-component` sont portees (la glose « 0 bit
// lu » de son repartiteur `FUN_141fd997c` etait fausse : c est l union discriminee de l etat
// physique du bipede, de 15 a plus de cent bits selon le tag). Cette constante hache la VALEUR
// de la revision de grammaire : elle monte.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED, ET C EST A PRENDRE AU SERIEUX CETTE FOIS :
// contrairement aux deux entrees precedentes, LES RECORDS DU BIPEDE NE FERMENT PLUS AUX MEMES
// BITS. Sur `bfecd02b` la marche rend 97 345 records `ti=35` au lieu de 97 447 et 6
// desynchronisations au lieu de 3, a etalon de contenu INCHANGE (`i0` 85,5 %, `i1` 77,5 %,
// `i21` 65,3 %, `i25` 97,1 %). Un ecart de un pour mille sur la population de records peut
// deplacer une attribution de kill. LE REDECODAGE RESTE UN GESTE DE PRODUCTION SUR SIGNAL
// UTILISATEUR (D6), jamais automatique — mais pour cette revision-ci il n est PAS sans objet, et
// c est dit.
//
// `SchemaVersion` NE MONTE PAS : aucun champ neuf au document. Le CONTENU CUIT change, lui, et
// ce sont les fixtures de contrat et `backfill-replay` qui en repondent.

// ENTREE `killsource-2026-09-21.4` (2026-09-21, lot 5.7.4) : LA REVISION MONTE DERRIERE UNE
// CORRECTION DE PUBLICATION DE LA COUCHE GRAMMAIRE, ET LE KILL-FEED N EST PAS CONCERNE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.4` : la porte
// des etats de mouvement s inscrit dans la neutralisation des lectures speculatives, qu elle
// avait oubliee au lot 5.3.6 — elle publiait les essais d alignement de la marche, dans un
// rapport de 14 a 152 pour un (chronique de `grammar`, entree du meme jour). Cette constante
// hache la VALEUR de la revision de grammaire : elle monte.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : RIEN, et c est prouve cette fois-ci plutot
// qu argumente. Le correctif n eteint qu UN crochet, `EtatMouvementHook` ; `PosCaptureHook` et
// `UnitRefHook` etaient DEJA inscrits dans la neutralisation depuis le lot 2.2, donc aucune
// position, aucune vitesse, aucune reference d unite ne change. `replay-equiv` sur `bcb6d393`
// le mesure : 3 etapes deplacees sur 57 — `movementStates`, `movementStates.stats` et
// `artifact` —, et `killsource` en fait partie des 54 IDENTIQUES a l octet.
//
// `SchemaVersion` NE MONTE PAS : la FORME du document ne change pas. Son CONTENU change
// (`stances[]` perd les intervalles qu il tenait d essais jetes), et c est `backfill-replay` qui
// en repond.

// ENTREE `killsource-2026-09-21.5` (2026-09-21, lot 5.9.4) : LA REVISION MONTE DERRIERE UN GENRE
// NEUF DE LA COUCHE GRAMMAIRE, ET LE KILL-FEED N EST PAS CONCERNE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.5` : le
// balayage des etats de mouvement capte desormais la vitesse verticale d `i1` et en DERIVE les
// sauts, publies sous le genre `jumpDerived` (chronique de `grammar`, entree du meme jour).
// Cette constante hache la VALEUR de la revision de grammaire : elle monte.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : RIEN. Le correctif n ajoute qu une SORTIE a
// la couche — deux transitions de plus dans `MovementStateRead` — et ne touche aucun bit lu : ni
// `PosCaptureHook`, ni `UnitRefHook`, ni aucune largeur. Les positions, les vitesses et les
// references d unite publiees sont identiques a l octet.
//
// `SchemaVersion` MONTE (65 -> 66), parce que la FORME du document change : `stances[].kind` peut
// desormais valoir `jumpDerived`, et `coverage.stances` porte deux compteurs de plus.

// ENTREE `killsource-2026-09-21.6` (2026-09-21, lot 5.9.5) : LA REVISION MONTE DERRIERE UN
// SECOND GENRE DE LA COUCHE GRAMMAIRE, ET LE KILL-FEED N EST PAS CONCERNE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.6` : le
// deserialiseur d `i57` publie son etiquette par une porte qui porte le SLOT, et le balayage des
// etats de mouvement en tire le genre `sprint` — LU, pas derive (chronique de `grammar`, meme
// jour). Le PARCOURS DE BITS d `i57` est inchange. `SchemaVersion` ne monte pas une seconde
// fois : la v66 du meme lot porte deja la forme.

// ENTREE `killsource-2026-09-21.7` (2026-09-21, lot 5.10.1) : LA REVISION MONTE DERRIERE LA
// LECTURE DE L EMBARQUEMENT, ET LE KILL-FEED N EST PAS CONCERNE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.7` : `i10` et
// `i14` entrent dans la couche de capture, et la marche des morts rend un second fait — les
// lectures d occupation, d ou sort le SIEGE d un episode (chronique de `grammar`, meme jour). Le
// PARCOURS DE BITS des deux composants est inchange, et le garde-rail de la couche de capture
// l exige. `SchemaVersion` ne monte pas : la forme du document ne change pas, `seat` change de
// SOURCE.

// ENTREE `killsource-2026-09-21.8` (2026-09-21, lot 5.11.0-a) : LA REVISION MONTE DERRIERE UNE
// LARGEUR D `i63`, ET LE KILL-FEED N EST PAS CONCERNE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.8` : le compte
// du second tour d `i63 biped-action-component` se lit desormais dans le masque de tete du
// composant, la ou une constante le tenait a zero sur une doc inversee (chronique de `grammar`,
// meme jour). C est une VRAIE largeur qui change — `i63` consomme 196 bits a masque nul et
// 196 + 3 x popcount au-dela — donc la montee n est pas un faux positif d empreinte.
// `SchemaVersion` ne monte pas : la forme du document ne change pas.

// ENTREE `killsource-2026-09-21.9` (2026-09-21, lot 5.11.7) : LA REVISION MONTE DERRIERE LA
// GARDE DE TABLE DE VUE, ET CETTE FOIS LA SORTIE DES FAITS CHANGE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE, mais `grammar.Rev` passe a `grammar-2026-09-21.9` et le
// changement n est PAS un faux positif d empreinte : la marche cesse de lire au-dela de la fin
// des paquets, rend 17 115 records `ti=35` de plus sur `bfecd02b`, et `replay-equiv -films
// bcb6d393` deplace le digest de l etape `killsource` (chronique de `grammar`, meme jour).
//
// LES LIGNES DE KILL DEJA EN BASE DEVIENNENT DONC CANDIDATES AU BACKLOG DE REDECODAGE. Le
// redecodage du parc reste un geste de PRODUCTION, reserve au pilote SUR SIGNAL UTILISATEUR
// (decision D6 du plan), JAMAIS automatique. `SchemaVersion` ne monte pas : la forme du document
// ne change pas.

// ENTREE `killsource-2026-09-22` (2026-09-22, lot 5.13.1) : LA REVISION MONTE DERRIERE LA
// LECTURE DE LA VUE D UNE LIAISON D IMAGE-CLE, ET LA SORTIE NE BOUGE PAS SUR LES TEMOINS.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-22` : les deux
// bits de tete d un identifiant d image-cle sont le RANG DE LA VUE (`vue + 8`, ecrit par le
// registraire `FUN_1409c9860`), et [grammar.World.BindImageCle] les LIT au lieu de les jeter
// comme `BindWildcard` le faisait. Sur les deux films temoins l image-cle ne declare QU UN rang,
// donc la sortie est identique au bit : `dad793c7` 99,50 % de paquets fermes et 75 records
// `ti=35` ; `bfecd02b` 114 458 records `ti=35`, 5 desynchronises, etalon `i21` 65,2 %.
//
// LA MONTEE EST DONC PRUDENTIELLE, ET ELLE EST DITE : sur un film dont l image-cle declarerait
// DEUX rangs, les liaisons du second passent en vue inconnue et la marche y rend PLUS de records.
// Les lignes de kill deja en base deviennent candidates au backlog de redecodage — geste de
// PRODUCTION, pilote, SUR SIGNAL UTILISATEUR (D6), jamais automatique. `SchemaVersion` ne monte
// pas : la forme du document ne change pas.

// ENTREE `killsource-2026-09-22.2` (2026-09-22, lot 5.13.3) : `i57` PORTE EN ENTIER, ET LA
// SORTIE DES FAITS CHANGE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE, mais `grammar.Rev` passe a `grammar-2026-09-22.2` : la
// branche `tag == 3` d `i57` est desormais lue au lieu de desynchroniser (chronique de
// `grammar`, meme jour). Le golden de la mini-bobine le MONTRE : une ligne de kill passe de la
// voie `scan` a la voie `marche` (marche 6 -> 7, scan 3 -> 2), avec le meme verdict et
// `DESACCORD` toujours a 0 — la marche va plus loin, elle ne decide pas autrement.
//
// Les lignes de kill deja en base deviennent candidates au backlog de redecodage — geste de
// PRODUCTION, pilote, SUR SIGNAL UTILISATEUR (D6), jamais automatique. `SchemaVersion` ne monte
// pas : la forme du document ne change pas.

// SUITE DU MEME RANG `killsource-2026-09-22.2` (2026-09-23, lot M2.1 de la campagne « retours
// rejeu ») : LA REVISION NE MONTE PAS, ET LE CHOIX EST ECRIT.
//
// L empreinte bouge pour deux causes : la VALEUR de `grammar.Rev` (`grammar-2026-09-23`, les
// entites `ti=9` rendues par la meme passe que la table d equipes — `killsource/` ne lit pas
// `ScanPlayerTeams`) et `killsource/botmeta.go`, qui garde l INSTANT de chaque paquet BOT_METADATA
// (`BotEntry.Declarations`, `Roster.BotPaquetsIncomplets`). L agregat qui epingle le roster du
// kill-feed est le meme a l octet (memes bots, meme ordre de decouverte, meme `NBots`) : aucune
// ligne de `match_kill_events` ne change, donc AUCUN BACKLOG. Les declarations ne nourrissent que
// la publication du rejeu (lien bot -> entite, presence des bots) ; le fichier de faits les porte
// en section 5 et `SchemaDesFaits` monte au lot M2.2.
//
// SUITE DU MEME RANG `killsource-2026-09-22.2` (2026-09-23, lot M2.3 de la meme campagne) : LA
// REVISION NE MONTE PAS, ET LE CHOIX EST ECRIT.
//
// L empreinte bouge encore pour deux causes : `killsource/botmeta.go` date le paquet BOT_METADATA
// de TETE de chunk a l image-cle qui le precede (mesure `b1ad85eb` : ecrit 390 us apres elle, il
// porte l etat A l image-cle — seul `FromUS` d une declaration change) et `fallback/` gagne la
// tranche `replay/places` (les quatre replis de la publication du roster). L agregat du roster du
// kill-feed est le meme : aucune ligne de `match_kill_events` ne change, AUCUN BACKLOG.
