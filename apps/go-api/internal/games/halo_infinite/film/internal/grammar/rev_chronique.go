package grammar

// rev_chronique.go — LA CHRONIQUE DE [Rev], UNE ENTREE PAR RANG.
//
// # POURQUOI CE FICHIER EXISTE (2026-09-18, lot 2.4.1)
//
// La chronique vivait dans le godoc de [Rev]. Au rang `.28` ce fichier passait 500
// lignes, et le ratchet de taille (`archlint/film_file_size_test.go`) le refusait — a juste
// titre : une chronique qui ne peut plus grandir cesse d etre tenue, et c est exactement le
// defaut F5 que `TestChroniqueCouvreLaRevisionCourante` a ete ecrit pour fermer. Elle vit donc
// ici, ou elle peut grandir, et `rev.go` ne garde que la regle et la constante.
//
// # CE FICHIER N EST PAS DE LA GRAMMAIRE
//
// Comme `rev.go`, il est EXCLU de l ensemble hache par l empreinte (cf.
// `fichiersHorsGrammaire`) : il DECRIT la grammaire, il n en fait pas partie. Sans cette
// exclusion, ecrire une entree changerait l empreinte, et la branche « la revision a change
// sans que la grammaire bouge » redeviendrait du code mort (revue R1, P2-3).
//
// # LA CHRONIQUE, A PARTIR DU `.11` : UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// LES TROIS DERNIERES ENTREES ONT ETE REECRITES LE 2026-09-16 (revue de jalon M1, RONDE 2,
// constat F5). Elles etaient EMPILEES et toutes trois annoncaient « `.11` -> `.12` », suivies de
// deux lignes « FUSION ... au rang suivant » qui racontaient une renumerotation que l integration
// n a jamais faite : la chronique s arretait donc a `.12` pendant que la constante valait `.14`,
// et les changements de COMPORTEMENT portes par `.13` et `.14` n avaient AUCUNE entree. Relevee
// commit par commit sur l integration (`git log --first-parent`), la suite reelle est celle-ci —
// un lot, un rang, dans l ordre ou les merges sont tombes.
//
// LES RANGS ANCIENS VIVENT DANS LES ARCHIVES : `.12` a `.28` dans `rev_chronique_archive.go`,
// `.29` a `.42` dans `rev_chronique_archive_2.go`, `grammar-2026-09-18.2` et `.3` dans
// `rev_chronique_archive_3.go`, `grammar-2026-09-20` a `.2` et `grammar-2026-09-21` a `.4` dans
// `rev_chronique_archive_4.go`, `grammar-2026-09-21.5` a `grammar-2026-09-22.6` dans
// `rev_chronique_archive_5.go`, `grammar-2026-09-22.7` a `.12` dans `rev_chronique_archive_6.go`.
// La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`), le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place), puis le
// 2026-09-22 (lot 5.20.1, une CINQUIEME archive), puis le 2026-09-24 (lot M4b, une SIXIEME).
// C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du `grammar-2026-09-24`.
//
// ENTREE `grammar-2026-09-24` (2026-09-24, integration de la vague D des retours du rejeu) : UNE
// SEULE MONTEE POUR LES LOTS DE LA VAGUE, partis de `fe7079f41` et qui avaient chacun pose
// `grammar-2026-09-23` sur leur branche ; la valeur de l integration est datee du jour, pour qu aucun
// fait ecrit par un lot seul (a sa revision de branche) ne se relise « a jour ». Les parties suivent.
//
// PARTIE M2.1 (2026-09-23, lot M2.1 de la campagne « retours rejeu ») : LES
// OCCUPANTS DU MATCH SONT LUS PAR ENTITE `ti=9`, ET LA TABLE PAR INDEX DEVIENT LE CONTROLE.
//
// CE QUE LE RANG CHANGE. `ScanPlayerTeams` rend, de la MEME passe sur les images-cles, les
// ENTITES (`player_entities.go`) : slot de replication, index de joueur, designateur, rangs de la
// premiere et de la derniere image-cle PORTEUSE, trous, instabilite. Aucun bit n est lu autrement
// — meme marche d etat complet, meme lecteur `lireEquipeDuRecord`, memes domaines —, et la table
// `index -> designateur` rendue est la meme a l octet (l etape observee `playerTeams` le prouve).
// Ce qui naît est une DONNEE que la passe jetait : la sonde P4 (`b1ad85eb`) a mesure trois entites
// d index 8 (designateurs 0, 0, 1 : trois bots de deux equipes), que l agregation par index
// rendait « divergentes » et donc muettes.
//
// LE RANG MONTE PARCE QUE LA COUCHE REND UNE LECTURE DE PLUS, pas parce qu une largeur bouge : la
// publication (`replay/occupants.go`) en tire la PRESENCE, l EQUIPE PAR ENTREE et la PLACE du
// roster, donc le contenu cuit change et `replay.SchemaVersion` monte au commit de publication.
//
// `facts.Rev` NE MONTE PAS : `killsource/` ne lit pas `ScanPlayerTeams`, et ce qu il gagne au meme
// lot (l instant des paquets BOT_METADATA) ne nourrit que le rejeu — aucune ligne de kill ne
// bouge. Son golden est refige parce qu il hache la VALEUR de `grammar.Rev` et ses propres
// sources (cf. la chronique de `facts`).
//
// PARTIE M3 (2026-09-23, campagne « retours rejeu », lot M3.1, puis M3.2 au meme rang) : LA
// MARCHE D IMAGE-CLE NE COUPE PLUS LA TABLE, ET SON ELECTION DEVIENT UN REPLI NOMME.
//
// CE QUE LE RANG CHANGE (`keyframe_world.go`, `keyframe_loadout.go`). Le balayeur d ancres
// d image-cle (`WalkKeyframeWorld`, lu par une quinzaine de balayages de cuisson et par le monde
// de `killsource`) choisit l ancre suivante par TROIS decisions dans cet ordre : le VOISIN
// immediat (slot+1, generation 1 — inchange), puis le RECALAGE sur l en-tete EXACT d un bipede
// (generation 1, mot d archetype de 32 bits egal a 35) quand il precede l ancre que l election
// retiendrait, puis l ELECTION (generation basse, slot bas), desormais le repli nomme
// `repli_ancre_d_image_cle_par_election`, compte par cuisson. Et une fenetre de 120 000 bits SANS
// candidat n arrete plus la marche : la recherche glisse jusqu au candidat suivant ou a la fin
// de table.
//
// LA MESURE QUI A DECIDE (quatre films, voie film, un a la fois) :
//
//	                    ancres           bipedes      perdues vs base   ajoutees confirmees
//	dad793c7  base 868  -> 902            4 -> 4       0                 33 / 34
//	81c02726  base 8619 -> 8896           116 -> 129   2 (fausses)       262 / 263 (+ 13 bipedes)
//	a0c36016  base 14659 -> 14890         159 -> 304   25 (la fausse     110 / 111 (+ 145 bipedes)
//	                                                   ancre 385/ti 19)
//	b1f01a33  base 6719 -> 6816           192 -> 198   5 (fausses)       67 / 96 (+ 6 bipedes)
//
// Les ancres « perdues » sont les fausses ancres de slot bas que l election retenait : 30 sur 32
// ne se repetent dans aucune autre image-cle, et les 25 d a0c36016 sont la MEME fausse ancre
// structurelle (slot 385, ti 19, a +825 bits de l en-tete du bipede 519 dans chaque image-cle —
// sonde P2). Les 29 ajouts non confirmes de b1f01a33 sont la chaine de slots CONSECUTIFS
// 1349..1376 de l image-cle 0, que la fenetre coupait (mecanisme B de la sonde P2).
//
// LA REGLE « GENERATION 1 PUIS LE PLUS PROCHE » (V2 de la sonde P2) A ETE MESUREE ET ECARTEE :
// elle rend les bipedes mais perd les autres ancres par milliers (a0c36016 14 659 -> 11 890,
// b1f01a33 6 719 -> 5 524 et trois bipedes perdus). La fenetre n est PAS retiree : la marche
// sans aucune fenetre rend EXACTEMENT les memes ancres que la fenetre glissante sur les quatre
// films, pour un temps multiplie par 5 a 6 ; elle ne borne plus que la PORTEE de l election.
//
// `facts.Rev` MONTE : le monde de `killsource` (`world.go` `preload()`) lie les slots que cette
// marche rend. Le codec des faits passe en v26 (`SchemaDesFaits` 4) et porte la SANTE de la marche
// (decisions et bipedes absents encadres) ; le document la publie en `coverage.keyframes` au
// commit de la montee de schema du meme lot (M3.2 : 69 sur la branche, 70 a l integration de la
// vague D), une seule montee pour le lot.
//
// LE MEME RANG PORTE LE LOT M3.2 (meme lot, une seule montee de revision) :
//
//   - L ETAT PAR DEFAUT DU BIPEDE LIT LE R(32) DE SA DERNIERE FEUILLE (`default_state.go`,
//     `uVar10 >= 12` : `FUN_14080d69c` = R(1) ; si 1, R(32) — la grammaire de l en-tete du
//     fichier, que le port avait amputee sur la foi de « 166 = 198 - 32 »). DEUX ORACLES : la
//     famille d i43 d un record NEW de naissance tombe sur celle que le catalogue du match
//     localise dans 293 records sur 293 (45 Arena, 106 Super Fiesta, 142 CTF ; 0 sans ce
//     R(32)) ; et `n2`, lu apres l etat par defaut d un record d image-cle, vaut 5088 sur
//     128 + 197 + 304 records des trois films, contre 2136725276 sans lui — la valeur de ce
//     R(32) lue a la place de `n2`. Tout record NEW de bipede des paquets delta se traverse
//     desormais aligne, et le chemin d etat complet de l image-cle aussi.
//   - LA DOTATION DE NAISSANCE EST LUE (`birth_loadouts.go`, `ScanBirthLoadouts`) : le record NEW
//     de chaque creation reconnue par `ScanBipedCreations`, traverse, et rendu SEULEMENT s il se
//     FERME — suivi d un delta propre sur un slot lie ou d un record NEW dont le monde ou une
//     image-cle ulterieure confirme l archetype. Mesure : 48/48, 104/106, 139/142 naissances
//     fermees sur les trois films ; 3 fermetures sur 1 776 au temoin decale. Aucune lecture de
//     repli : un record qui ne se ferme pas ne rend rien, et le refus se compte par cause.
//   - L EMPLACEMENT D ARME : `HeldWeaponChange.Emplacement` (rang du composant
//     `weapon-state-type-info` dans l archetype du film). La PREMIERE emission d un emplacement se
//     juge, POUR CHAQUE VIE, contre la dotation de naissance puis contre le dernier releve
//     d image-cle PASSE (`SpawnPredicate`, jamais un releve a venir), et la chaine des emissions
//     d un slot se coupe a chaque nouvelle creation du corps qui l occupe.
//
// REPRISE APRES LA REVUE ADVERSE DU LOT (2026-09-24), MEME RANG :
//
//   - UNE FIN DE TABLE A CHEVAL SUR DEUX FENETRES SE RECONNAIT (`kfScanGlissant`, constat F5). Le
//     compteur de sentinelles repartait de zero a chaque fenetre : une trainee coupee par la
//     frontiere (moins de 2 048 sentinelles de chaque cote) n etait jamais une fin de table, et le
//     glissement allait lire des ancres au-dela. La fenetre suivante reprend desormais au DEBUT de
//     la trainee sur laquelle la precedente a fini (`traine`, rendu par `kfScanNext`).
//   - `Glissements` ne compte plus que les fenetres vides FRANCHIES pour atteindre une ancre ; une
//     recherche qui finit sur la fin de table ou du payload n a rien franchi.
//
// PARTIE D-fix (2026-09-24, pre-integration de la vague D, meme rang) : L ELECTION NE CONTREDIT
// PLUS UN RECORD PROUVE, UNE ABSENCE LUE PAR REPLI NE PROUVE RIEN, ET LE MONDE D UNE MARCHE DE
// TRAMES NE GARDE PAS UNE LIAISON QUE LE FILM DEMENT.
//
// LA CAUSE (rouge M2 x M3 `TestEntitesTi9SurLesBobines`). Dans l image-cle d avant-match de
// `bcb6d393` (c1 p0), `fb1a1a72` (c1 p0, p1) et de `000d5950` (c1 p1, sur le film ENTIER : la
// mini-bobine n a pas de registre lisible, sa marche reste sans preuve), le record du
// slot 122 (ti 45) couvre ~125 000 bits ; la fenetre suivante porte les vrais 1280..1298 ET une
// fausse ancre 192 / ti 1 dans le corps du record 1298. L election (slot bas) la retenait : 1280..
// 1298 disparaissaient, dont 1297, le joueur gere de l index 0, que M2 lisait ARRIVE plus tard.
// Meme geste en c1 p3 de `bcb6d393` (fausse 1536 / ti 0, 29 records de 1537..1601 effaces).
//
// LA REGLE (`keyframe_world_preuve.go`). La table est croissante en slot. Un candidat PROUVE (marche
// d etat complet sous le cadre du film — profil par defaut, drapeau de controle de corruption, MPP
// de son format —, `n1 > 0`, composants sans desynchronisation, fin EXACTE sur un en-tete valide de
// slot superieur) interdit tout elu qui contredit l ordre bit/slot ; l election se rejoue sans les
// refutes. Aucun seuil. L exigence de CONTENU est mesuree : un record vide (172 bits) se ferme meme
// lu decale d un bit — 55 fausses preuves sur 39 471 candidats surement faux sans elle, 0 avec
// (re-mesure, DFIX-R9). Compteurs `Refutations` et `PreuvesContradictoires` (DFIX-R8). Toute marche
// de CUISSON est celle du film (garde-rail `archlint/keyframe_walk_proof_test.go`) ; hors cuisson,
// `ScanFilmWeaponDamages` marche sans preuve — backfill a decider par l utilisateur (DFIX-R4).
// Mesure (sept bobines) : 4 paquets changent, 1 refutation chacun ; perdues SEULEMENT les fausses
// ancres 192 (x3) et 1536 ; fermeture ti=9 1 738 -> 1 741 ; golden des familles : `carrierMarks`.
//
// LE PRINCIPE (`player_entities.go`, `player_entities_entetes.go`). Une absence n est PROUVEE que si
// l en-tete EXACT du record ti=9 de l entite n apparait a AUCUNE position de bit du payload : la
// marche perd aussi par le saut de largeur, le faux voisin et le record au-dela de sa fenetre, sans
// rien « ecarter » (DFIX-R2). Un en-tete trouve et non lu fait une image-cle DOUTEUSE pour ce slot
// (`DouteDAbsence`, persiste) : ni depart ni arrivee tardive, la presence ne borne que sur une
// absence PROUVEE. Compteurs `coverage.seats.imagesClesDouteuses` / `bornesDifferees` (0 sur les
// bobines avec la preuve ; sans elle, le seul slot 1297 de `bcb6d393` et `fb1a1a72`).
//
// L IMAGE-CLE DIT QUI N EST PLUS LA (`keyframe_liaison.go`). La marche des etats de mouvement ne
// retirait une liaison que sur un DEL LU : un DEL manque laissait celle du mort, et l occupant
// suivant du slot se decodait sous son archetype (`a0c36016`, slot 649 : cinq vies, 24 intervalles,
// perte nee a M3.1). A chaque image-cle, une liaison que ni la chaine, ni la table de datums, ni un
// candidat ecarte ne porte est oubliee (`MovementStateStats.LiaisonsOubliees`).
//
// UN NEW NE REMPLACE PAS UNE ENTITE VIVANTE (`frame_infer.go`, `contreditUneEntiteVivante`). Le jeu
// ne cree pas sur une entree occupee de sa table de datums : un NEW propre qui contredit une liaison
// EN DUR (chaine ou NEW) d un autre archetype est une lecture fausse, que la traversee du NEW de
// bipede (R(32), M3.2) atteignait (`0797ce72` c12 : « 123 ti 2 » ecrasait le `ti 4` de chaque paquet,
// onze vies perdues ; `396cfc92` : sept). Il n est plus lie, la marche continue (compteur
// `NeufsContreUnVivant`, dont l image-cle suivante rend le VERDICT — lecture fausse confirmee,
// vraie creation perdue par un DEL non lu, indecis — publie avec `LiaisonsOubliees` dans
// `coverage.stances`, DFIX-R6). Mesure (seize films non BTB, base -> tete) : aucune vie ne perd un
// intervalle, hors un accroupi d une frame de `c75f33b8` que l alignement de M3.2 efface.
//
// PARTIE M4b (2026-09-24, lot M4b de la campagne « retours rejeu » : le tir des vehicules), SOUS LE
// MEME RANG — la vague n est pas publiee, et son rang reste UNIQUE (empreinte recopiee).
//
// LA VUE DE CONTROLE EST LUE EN ENTIER (`frame_vue_controle.go`). L entree `kind 0`
// (`FUN_1406d0388`) porte les trois branches que le port refusait (`FUN_1406cd860` : champs de 5, 6
// et 5 bits relus au site d appel), rend ses entrees (index de controle, bloc d action, champs) et
// un verdict par paquet (`LectureVueC` : atteinte, fermee par l oracle de cadrage, arret nomme).
// LE BLOC D ACTION EST LU JUSTE (`bloc_action.go`, sorti de `unit_control.go`) : la queue
// `FUN_140c9e990` lit l entier a largeur variable en CATEGORIE 1 (genre 1) et 2 (genre 2), le
// vecteur `FUN_1431a0cbc` lit R(2) puis R(19) (mode 1) ou la position quantifiee de niveau 16
// (mode 0). Le meme bloc sert `i19 unit-actor-control` : la vue B le lit donc juste aussi.
// LE TIR CONTINU (`tir_continu.go`) : la marche des trames plie les entrees en RAFALES par bit
// tenu, avec les trous de lecture portes par la rafale (jamais un « pas de tir »).
// LE RECORD 36 EST LU PAR SA GRAMMAIRE (`fire_events.go`, `fire_aim_modal.go`) : type 36 seul (le
// 37 `weapon_overheat` ecarte), reference 0 (l unite tireuse), tireur sur CINQ bits, numero de
// tir, identifiant d arme ; plus aucun offset fixe.
// Mesure (trois films non BTB, avant -> apres) : vue B inchangee ; vue C fermee 81c02726
// 19,5 % -> 36,0 %, 8a485699 9,0 % -> 26,9 %, b1ad85eb 26,5 % -> 39,9 %.
//
// REPRISE DE LA PARTIE M4b (2026-09-25, revue du lot et gate G2 par famille), MEME RANG, empreinte
// recopiee : LA VUE B SE FERME LA OU LE TIR SE LIT. Le gate G1 laissait deux frags au Ghost dans un
// trou de lecture (81c02726, 500-658) et le gate G2 trouvait des pilotes qui tiraient sans rafale
// (8a485699 : Banshee de l index 7, dix touches et 37 tirs numerotes ; 1cd3848a : la LAAG, trois
// frags). Chaque trou a ete suivi jusqu a son record, et chaque cause est une LECTURE :
//
//	le debut de liste (`debut_de_liste.go`) : les records NEW de tete d un paquet a evenements
//	    (naissances, armes et equipement laches a la mort, projectiles) etaient sautes par le
//	    localisateur ; l objet jamais lie clotait ensuite chaque paquet sur son rejet. Candidat =
//	    en-tete NEW dans la bande de l archetype ; preuve = la chaine de records qui finit au bit
//	    pres sur le debut localise, ou la fermeture de la vue C quand il n y en a pas.
//	l etat par defaut du PROJECTILE (`default_state_ti41.go`, `FUN_1408efb58`, record NEW) : il
//	    sortait du repli « 0 bit ».
//	la queue du CORPS DE MORT (`components_object.go`) : l octet de tete `comp+0x1c` annonce le
//	    bloc de vitesse (lu dans le flux, pas un drapeau RAM) et un R(5) etait lu de trop — chaque
//	    paquet de mort perdait sa liste apres le bipede mort.
//	les deux positions du corps d `i54` (`components_biped_ability.go`) : niveau 0x10, largeurs
//	    ABSOLUES de la carte, pas celles du delta du bipede (Launch Site : 31 bits de trop peu).
//	six composants (`composants_vue_b_m4b.go`) : `ti=47 i2`, `ti=5 i22` et `i24`, `ti=10 i24`,
//	    `ti=40 i34` (porte runtime supposee : `repli_physique_de_type_de_vehicule_supposee`, compte)
//	    et `ti=40 i37`.
//
// Mesure (vue C fermee, base `c9ef97ec6` -> reprise ; instrument `m4b_tir_continu_research_test.go`) :
// 81c02726 19,5 % -> 76,2 %, 8a485699 9,0 % -> 63,2 %, b1ad85eb 26,5 % -> 89,4 % ; 1cd3848a 93,1 %
// et 7b0d89c4 80,5 % a la reprise. Fermeture d image-cle (golden du lot 0.A.3) : `ti=5` de 0 % a
// 100 % sur les sept bobines, `ti=47` de 0 % a 7-100 %. La position des transformations d un corps
// rigide (`ti=38 i18`, meme lecteur de niveau 0x10) N EST PAS reprise : lue dans l etat complet
// d une image-cle (pleine precision, R(96)) elle ferait baisser la fermeture `ti=38` du golden ;
// decouverte consignee au rapport du lot.
// Instruments (tag `research`) : `m4b_vueb_rejets_research_test.go` (causes des trous, rejets
// d en-tete, journal des paquets), `m4b_liste_research_test.go` (chaines de tete, vraie frontiere
// d un record, depart d un composant : le corps de mort de 8a485699 p692), `m4b_monture_research_test.go`
// (la 3e monture de 81c02726, OUVERTE : aucun embarquement lu apres la naissance du dispositif
// 2308, dont le NEW desynchronise sur `ti=43 i21`).
