package killsource

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

// ENTREE `killsource-2026-09-24` (2026-09-24, integration de la vague D des retours du rejeu) :
// UNE SEULE MONTEE POUR LA VAGUE. Le lot M3 avait pose `killsource-2026-09-23` sur sa branche ;
// la valeur de l integration est datee du jour, pour qu aucun fait ni aucune ligne de kill ecrits
// par le lot seul (a sa revision de branche) ne se relisent « a jour ». Le lot M2 ne la faisait
// pas monter (cf. les deux suites du rang `killsource-2026-09-22.2` ci-dessus) ; elle monte pour
// M3, et elle hache la VALEUR reconciliee `grammar-2026-09-24`.
//
// PARTIE M3 (2026-09-23, campagne « retours rejeu », lot M3) : LE MONDE DE
// `killsource` LIE LES SLOTS QUE LA MARCHE D IMAGE-CLE REPAREE REND.
//
// AUCUN OCTET DE `facts/killsource/` N EST TOUCHE, mais `grammar.Rev` passe a
// `grammar-2026-09-24` (valeur de la vague) : le balayeur d image-cle recale sur l en-tete exact d un bipede et ne
// s arrete plus sur une fenetre vide (lot M3.1, chronique de `grammar`). `world.go` `preload()`
// lie la premiere declaration de chaque slot de TOUTES les images-cles du film par ce balayeur :
// les slots qu il atteint desormais (bipedes perdus par une fausse ancre de slot bas, suffixes
// que la fenetre coupait) entrent dans le monde de la marche des kills. La sortie PEUT donc
// changer — plus de records atteints, jamais une lecture autre d un record deja atteint.
//
// LE MEME RANG PORTE LE LOT M3.2 : l etat par defaut du bipede lit enfin le R(32) de sa derniere
// feuille (chronique de `grammar`). `killsource` traverse ce meme etat par defaut (calibration
// des largeurs, marche des morts) : sur la mini-bobine, seul l oracle de calibration bouge
// (profil plat, score 0 -> 1, decision inchangee), et aucune ligne de kill.
//
// Les lignes de kill deja en base deviennent candidates au backlog de redecodage — geste de
// PRODUCTION, pilote, SUR SIGNAL UTILISATEUR (D6), jamais automatique (Q3 du plan des retours
// rejeu : backfill killsource si la revision le chaine).
//
// PARTIE D-fix (2026-09-24, lot correctif de la pre-integration de la vague D, MEME RANG) : le monde
// de `killsource` (`world.go` `preload()`) lie les slots par la marche d image-cle DU FILM
// (`FilmContext.MarcheDImageCle`), qui refuse l elu qu un record prouve contredit : les fausses
// ancres 192 et 1536 ne se lient plus, les records qu elles effacaient si. Le registre des replis
// gagne `repli_borne_de_presence_differee_sur_doute` (tranche `replay/places`, revue adverse
// DFIX-R7). La sortie peut changer comme au rang M3 (plus de records atteints) ; la revision reste
// celle de la vague, empreinte recopiee. EXCEPTION NOMMEE : la precision par arme
// (`ScanFilmWeaponDamages`) marche encore sans preuve — decision de backfill a l utilisateur.
//
// PARTIE M4b (2026-09-25, lot M4b des retours du rejeu, MEME RANG, empreinte recopiee) : AUCUN
// OCTET DE `facts/killsource/` N EST TOUCHE. L empreinte bouge par la VALEUR reconciliee de
// `grammar.Rev` (reprise M4b de sa chronique : les records NEW de tete, l etat par defaut du
// projectile, la queue du corps de mort, six composants) et par `fallback/`, qui gagne
// `repli_physique_de_type_de_vehicule_supposee` (tranche `filmdec`). La marche des morts de
// `killsource` traverse ces lectures : la sortie PEUT changer comme aux rangs M3 et D-fix — plus de
// records atteints (le corps de mort ne perd plus la liste qui le suit). Mini-bobine : une mise a
// mort de plus lue par la MARCHE (8 contre 7, le scan en rend une de moins), memes armes et memes
// credits, l oracle de calibration seul bouge (decision inchangee). Le backlog reste celui de la
// vague (Q3 : backfill killsource a la cloture, sur signal utilisateur).
//
// PARTIE M7b (2026-09-24, lot M7b des retours du rejeu, parti du rang `killsource-2026-09-22.2`,
// reuni a CE RANG a l integration de la vague D le 2026-09-25, empreinte recopiee) : AUCUN OCTET
// DE `facts/killsource/` N EST TOUCHE. `fallback/` gagne deux entrees de DONNEES de la tranche
// `replay/vehicules` (`repli_tourelle_montee_loin_du_porteur`,
// `repli_episode_borne_par_la_vie_suivante`) dont les gardes vivent dans la PUBLICATION (episodes
// d occupation du calque des vehicules). Aucune ligne de kill ne change ; le backlog reste celui
// de la vague. L entree de la branche, datee a son rang, reste dans `rev_chronique_archive.go`.

// ENTREE `killsource-2026-09-26` (2026-09-26, jalon J7 du PLAN_SUITE_AUDIT_DECODEUR_FILM) : LA
// REVISION MONTE UNE FOIS POUR LES SEPT CONSTATS FK-1 A FK-7, ET LA SORTIE DU KILL-FEED CHANGE.
//
//	FK-2 (J7.1)  un nom de remplissage (`?`, `?N`) n est plus jamais publie : un assistant pose
//	             par l inference sur un remplissage sort REJETE `hors-roster` (colonne
//	             `assist_gamertag` NULL, `assist_rejected` = `hors-roster`) ; les temps 4 et 5 ne
//	             publient pas une ligne dont le nom pris au roster serait un remplissage
//	             (`Stats.NomsDeRemplissageRefuses`). Predicat unique `estNomDeRemplissage`.
//	FK-1 (J7.2)  l espace des humains est le NOMBRE DE SIEGES de la table du film, et non plus le
//	             nombre de noms du kill-feed : un remplacant ne desepingle plus le bot de relais. Un
//	             bot n est desepingle que si son slot tombe sous cette borne ET sur un siege que la
//	             table NOMME (revue adverse : un siege VACANT intercale ne desepingle rien). Sans
//	             table, aucun bot n est desepingle. Les bots non epingles sont publies
//	             (`Coverage.BotsNonEpingles`), journalises par film, et comptes par
//	             `replayidentity` (`killsource_bots_non_epingles`).
//	FK-3 (J7.3)  le lien par motif du xuid cherche aussi les joueurs qui tuent sans mourir. Ces
//	             candidats ne font que COMPLETER (revue adverse) : une lecture qui se contredit ou
//	             tombe sur un indice deja retenu les ecarte SEULS (`MotifTueursEcartes`), sans faire
//	             tomber l epinglage par motif du film.
//	FK-4 (J7.4)  le temps 4 ne reecrit plus un instant publie (`Stats.CollisionsDeMortDeBot`),
//	             SAUF une ligne du temps 3 posee sur un couple RECOLLE : elle ne confirme pas le
//	             tueur du couple, et cede a la mort de bot verifiee au meme instant (revue adverse,
//	             `Stats.AutoInfligeesSurCoupleFabriqueRemplacees`). Un couple recolle n est fantome
//	             que si sa mort de bot est PUBLIEE, ce qui tient `Covered <= RealPairs`. Revue ronde
//	             2 : « au meme instant » est a la milliseconde — le dead-state de la mort de bot doit
//	             tomber a l instant du kill-feed que la ligne porte ; une mort de bot prise par la
//	             fenetre a un kill VOISIN n est plus publiee a une fausse date et ne prive plus ce
//	             voisin de sa ligne. Un remplacement retire la provenance de la ligne remplacee :
//	             `Stats.Appariement` compte une provenance PAR LIGNE PUBLIEE (plus de `Fenetre`
//	             gonfle d une ligne disparue).
//	FK-5 (J7.5)  le numerateur de sante ne compte que des candidats, une fois : le temps 3 laisse
//	             les indices de bot aux temps de bot, et les inexpliques a indice de bot se
//	             comptent sur la population (plus sur le scan entier).
//	FK-6 (J7.6)  un kill-event lu deux fois dans le meme paquet (champs identiques) n est garde
//	             qu une fois, au bit le plus bas (`AssistStats.Doublons`) : plus de couple fabrique
//	             pour un kill orphelin voisin, plus de multi-attachement par doublon.
//	FK-7 (J7.7)  une carte fournie egale a l invariant (Cliffhanger) est une carte APPLIQUEE : plus
//	             de faux repli `repli_carte_absente_largeurs_par_defaut` ni de faux avertissement.
//
// LES LIGNES DE KILL DEJA EN BASE DEVIENNENT CANDIDATES AU BACKLOG DE REDECODAGE
// (`conditionBacklog`, `sync/killcollector/postsync.go`) : il est traite a la vague unique J11,
// geste de PRODUCTION pris par le pilote SUR SIGNAL UTILISATEUR (D6), jamais automatique.
// `SchemaVersion` ne monte pas (la forme du document de rejeu ne change pas) ; le codec des faits
// ne monte pas : la section 5 gagne des champs JSON, et un fichier ecrit sous la revision
// anterieure est refuse sur son EN-TETE, qui porte `killsource.Rev`.
//
// COMPLEMENT DU 2026-09-27, MEME RANG, MEME LOT NON PUBLIE (correctif J7 « carte obligatoire »,
// enquete ENQUETE_MARCHE_KILLSOURCE_2026-09-27). La revision NE MONTE PAS : `killsource-2026-09-26`
// n est pas publiee (la serie en base est `killsource-2026-09-24`), et deux changements d un meme
// lot partagent un rang — en ouvrir un second ferait redecoder le parc deux fois. L empreinte, elle,
// change (golden regenere par sa porte). Ce que la sortie gagne :
//
//	CARTE OBLIGATOIRE  `Decode` sans entree de catalogue portant des largeurs rend
//	                   `ErrCarteAbsente` ; plus aucun decodage aux largeurs de l invariant
//	                   (Cliffhanger) en production. Le collecteur met le film de cote
//	                   (`ecarte-carte-non-resolue`, `killsource_ecartes_carte_non_resolue`, aucun
//	                   marqueur de registre : il reste au backlog et sera decode quand sa carte se
//	                   resoudra). Le repli `repli_carte_absente_largeurs_par_defaut` est RETIRE du
//	                   registre ; seuls les instruments de recherche decodent sans carte, et le
//	                   DECLARENT (`Options.RechercheSansCarte`, interdit en production par ratchet).
//	CARTE LUE          la presence de la carte se lit sur l ENTREE (`carteApplicable`), plus sur une
//	                   difference de largeurs — acheve FK-7 sur l entree commise de Cliffhanger.
//
// LA MARCHE REVIENT SUR LE BANC, PAS EN PRODUCTION : le banc `TestGoldenFilms` decodait sans carte
// depuis `f3a2f00eb` ; sous leur carte, les quatre films rendent marche 356/369 et scan 7/8 au
// cumul (000d5950 94/91, 9b191a7f 84/80, 78919882 98/94, fccc61cd 93/91). En production, seuls les
// matchs SANS CARTE RESOLUE changent : ils etaient publies par le scan aux largeurs d une autre carte ;
// les NOUVELLES passes ne les publient plus (le match est mis de cote, il reste au backlog). LES
// LIGNES DEJA EN BASE RESTENT : celles d un match sans carte decode a `killsource-2026-09-24` (aux
// largeurs de Cliffhanger) demeurent dans `match_kill_events`, et aucune passe ne les remplace —
// le match n est plus jamais decode tant que sa carte ne se resout pas. Leur sort (purge ou non)
// est une decision renvoyee a J11 ; ce lot ne touche a aucune donnee.
//
// CORRECTIONS DE REVUE DU MEME JOUR, SANS EFFET SUR LA SORTIE (l empreinte ne bouge pas) : un match
// dont le registre ne porte ni `map_id` ni `map_name` (`port.ErrMatchMapUnknown`) est une carte NON
// RESOLUE, plus une panne retentee et telechargee a chaque cycle ; le post-sync ne relit plus la
// carte d un match deja constate sans elle sous le meme catalogue de bornes, pendant six heures au
// plus (registre en memoire, jauge `killsource_postsync_backlog_sans_carte`) et ne compte plus le
// backlog qu une fois par cycle. Le resolveur de carte du post-sync EMPRUNTE les metadonnees que le
// processus tient : un `map_name` reste UUID brut au registre est traduit comme au backfill hors
// ligne, au lieu d etre ecarte pour toujours (regression de la carte obligatoire, fermee le meme
// jour). Residu accepte : un film expire SANS carte n est jamais telecharge, donc jamais marque
// `MBitFilmAbsent`, et reste au backlog.

// ENTREE `killsource-2026-09-27` (2026-09-27, lot J5.5 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25) :
// LA REVISION MONTE DERRIERE LA GRAMMAIRE, PAR LA RECETTE DU SENS UNIQUE (ADR 0034 D-6 (2) et (3)).
//
// AUCUN OCTET DU PERIMETRE DE `killsource` N EST TOUCHE PAR LE JALON J5 hors `film/types`
// (renommage `EquipmentLifeKey` -> `LifeKey` du lot J5.1, deja recopie a revision constante). Ce
// qui monte est la VALEUR de `grammar.Rev` (`grammar-2026-09-24` -> `grammar-2026-09-27` : filtre
// de generation vivante GB-1, chaines d equipement par vie, slot de capture des records NEW), que
// la fermeture des imports de cette couche rencontre et hache. La chaine est mecanique : une montee
// de la couche du dessous remonte jusqu au backlog killsource, sans qu on ait a la plaider.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : RIEN ATTENDU, et c est argumente, non mesure
// par ce lot. `killsource` ne lit ni les positions bipedes, ni les huit canaux delta, ni les
// chaines d equipement que J5 change, et n installe aucun crochet de capture (le slot de capture
// pose sur un record NEW ne sert qu aux crochets et a l accumulateur, sans installateur en
// production). La preuve se lit au gate du superviseur (equivalence `killsource` sur `replay-equiv`
// et corpus gate).
//
// LES LIGNES DE `match_kill_events` DEJA EN BASE DEVIENNENT CANDIDATES au redecodage
// (`conditionBacklog`, `sync/killcollector/postsync.go`) : geste de PRODUCTION, pris par le pilote
// SUR SIGNAL UTILISATEUR (D6), jamais automatique. La meme vague (J11) rejoue de toute facon les
// faits d isolement, dont `IsolationDecoderRev` monte au meme lot.
//
// COMPLEMENT DU 2026-09-27 (lot J10.1 du meme plan, REVISION CONSTANTE) : la serie n a jamais ete
// publiee, son golden est regenere a revision constante par la recette. Ce qui change dans le
// perimetre : la VALEUR de `grammar.Rev` (`grammar-2026-09-27.3`) et cinq tris de cette couche
// rendus TOTAUX (DT-9) — les kill-events (instant, chunk, paquet, bit : `pickAssistHit` garde le
// premier porteur), les bots d un meme slot (ordre de decouverte, stable : l epinglage au
// kill-feed), les paquets de replication (instant, chunk, rang), les dead-states de la marche et les
// images-cles (ordre du film, stable), et l ordre des candidats de la bijection (le contenu du
// dead-state departage deux candidats sans position). Le kill-feed ne peut changer que sur des ex
// aequo que l ancien tri departageait au hasard.
//
// PORTE AUSSI J7 (fusion de `feat/suite-audit-decodeur-j7` du 2026-09-27) : `killsource-2026-09-26`
// n a JAMAIS ETE PUBLIEE SEULE — la serie en base reste `killsource-2026-09-24`, et ce rang
// `killsource-2026-09-27` est le premier publie qui contient les sept constats FK-1 a FK-7 et la carte
// obligatoire de l entree precedente, en plus de J5.5 et de J10. Un seul backlog de redecodage (J11)
// pour les deux entrees. A la fusion, les comptes de replis du lot J8.7 suivent les regles de J7 :
// le compte `repli_carte_absente_largeurs_par_defaut` disparait avec le repli, et celui de
// `repli_roster_indice_hors_bijection` juge le nom publie par le predicat unique
// `estNomDeRemplissage`.

// ENTREE `killsource-2026-10-02` (2026-10-02, lot L3a de la campagne de grammaire,
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6.0 points 1 et 3) : LA REVISION MONTE DERRIERE LA
// GRAMMAIRE, PARCE QUE LA SORTIE CHANGE.
//
// AUCUNE SOURCE DE LA COUCHE N EST TOUCHEE. Ce qui monte est la VALEUR de `grammar.Rev`
// (`grammar-2026-10-02.2` : la fin du moteur de partie `ti=0/1/2 i11..i17` portee, lecteur de
// minuteur unique), que la fermeture des imports de cette couche hache. La regle du plan est
// ecrite : un lot qui change une sortie monte `grammar.Rev` et `facts.Rev` (cette constante) suit ;
// `killsource.Rev` monte si la sortie change.
//
// CE QUI CHANGE DANS LA SORTIE `cmd/killsource json` (19 temoins de `config/replay_corpus.toml`,
// binaire de la base contre binaire du lot) : AUCUNE mort, aucune valeur, aucune voie. Changent le
// diagnostic d ORACLE de `calibration` (scores du profil plat et de `indexW_poignee`, 11 films) et,
// sur `111fa685`, deux compteurs de sante (`killsource_candidates_total` 226 -> 227,
// `killsource_unexplained_pair` 24 -> 25, d ou la population de la voie sequentielle 205 -> 206).
// Mesures : `campagne_grammaire_2026-10-01/LOT_L3a.md` §5.3.
//
// LE BACKLOG QU ELLE OUVRE EST UN BACKLOG DE DATATION : les lignes de `match_kill_events` deja en
// base deviennent candidates au redecodage (`conditionBacklog`, `sync/killcollector/postsync.go`),
// geste de PRODUCTION pris sur signal utilisateur (D6, D7 du plan), jamais automatique.
