package killsource

// rev_chronique_archive_2.go — LES RANGS `killsource-2026-09-21.2` A `killsource-2026-09-24` DE LA
// CHRONIQUE DES FAITS, TELS QU ILS ETAIENT ECRITS.
//
// ROTATION ORDINAIRE : `rev_chronique.go` atteignait 500 lignes et le ratchet de taille
// (`archlint/film_file_size_test.go`) refuse de le laisser grandir. Aucun mot n est reecrit, aucun
// octet de code n est touche : le fichier ne porte que des commentaires. La suite vivante de la
// chronique, a partir de `killsource-2026-09-26`, est dans `rev_chronique.go`.

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
