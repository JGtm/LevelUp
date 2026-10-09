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
// `rev_chronique_archive_5.go`, `grammar-2026-09-22.7` a `.12` dans `rev_chronique_archive_6.go`,
// `grammar-2026-09-24` a `grammar-2026-10-02.3` dans `rev_chronique_archive_7.go`,
// `grammar-2026-10-03` a `grammar-2026-10-08` dans `rev_chronique_archive_8.go`.
// La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`), le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place), puis le
// 2026-09-22 (lot 5.20.1, une CINQUIEME archive), puis le 2026-09-24 (lot M4b, une SIXIEME), puis
// le 2026-10-03 (integration de la vague 1 de la campagne de grammaire, une SEPTIEME), puis le
// 2026-10-06 (lot VA de la campagne, rangs `grammar-2026-09-27` a `.3` verses dans la septieme), puis le
// 2026-10-07 (fusion de `feat/v75` dans le lot des arrets de la vue B, rangs `grammar-2026-10-02` a
// `.3` verses dans la septieme), puis le 2026-10-08 (lot des arrets de la vue B, suite, une
// HUITIEME archive).
// C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du `grammar-2026-10-08.2`.

// ENTREE `grammar-2026-10-08.2` (2026-10-08, lot des arrets de la vue B, suite, `ti=12 i20` a
// `i27`) : les huit composants `managed-navpoint-visual-state-groups-component-0..7` se lisent
// (`FUN_140dbe1bc` : presence, puis identifiant, bloc de filtres, mot, un mot et une entree d ordre
// par filtre present, [consumeNavpointVisualStateGroup]). Contre `grammar-2026-10-08` : un record
// `ti=12` qui s arretait sur l un d eux se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.3` (2026-10-08, lot des arrets de la vue B, suite, `ti=10 i22`) : le
// composant `managed-object-interaction-filter-component` se lit (`FUN_140dbdf5c` : le bloc de
// filtres seul, `v = 1 < param_4`, [consumeManagedObjectInteractionFilter]). Contre
// `grammar-2026-10-08.2` : un record `ti=10` qui s arretait sur `i22` se lit jusqu au composant
// suivant.
//
// ENTREE `grammar-2026-10-08.4` (2026-10-08, lot des arrets de la vue B, suite, `ti=35 i59`) : LE
// CORPS DE L ANCRE DU GRAPPIN SE LIT COMME LE JEU LE LIT (`FUN_142f25e90`, ses huit etiquettes,
// [consumeAbilityAnchorBody]).
//
// Ce qui change, contre `grammar-2026-10-08.3` : la grammaire MESUREE du 2026-08-16 (deux etiquettes,
// trois « drapeaux » et un champ de sept bits) est remplacee par celle du lecteur du jeu : le prefixe
// `FUN_142f26e40` (une reference de categorie 1, puis `FUN_142f04664`, dont la position de
// `FUN_14076e494` au niveau 0x10 quand la reference est absente), six drapeaux, puis la branche de
// l etiquette. Aucune etiquette n arrete plus la marche. La position d une ancre n est publiee
// (`grappleReads`) que lue par `FUN_14076e524` a un index de plage, aux largeurs de la carte
// ([AbilityNonPredictedState.PosCarte]) ; [consume142f04664] rend sa position. Mini-bobine : les
// trames 3:636 et 3:722 passent le slot 513 et se ferment ; un record bipede de plus, et la passe
// des armes au sol ne rend plus les records `ti=37` de ces deux trames (`worldObjects_ti42` 54 -> 53,
// les points des slots 1525 a 1535 a ces deux instants).
//
// ENTREE `grammar-2026-10-08.5` (2026-10-08, lot des arrets de la vue B, suite, `ti=11 i4`) : le
// composant `managed-objective-interaction-filter-component` se lit (`FUN_140dbe170` : le bloc de
// filtres seul, `v = 1 < param_4`, [consumeObjectiveInteractionFilter]). Contre
// `grammar-2026-10-08.4` : un record `ti=11` qui s arretait sur `i4` se lit jusqu au composant
// suivant.
//
// ENTREE `grammar-2026-10-08.6` (2026-10-08, lot des arrets de la vue B, suite, `ti=10 i23`) : le
// composant `managed-object-flags-component` se lit (`FUN_1410d9b5c` -> `FUN_140f72efc`, `R(2)` plat,
// [consumeManagedObjectFlags]). Contre `grammar-2026-10-08.5` : un record `ti=10` qui s arretait sur
// `i23` se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.7` (2026-10-08, lot des arrets de la vue B, suite, `ti=12 i17`) : le
// composant `managed-navpoint-object-marker` se lit (`FUN_141169e68`, `R(32)` plat,
// [consumeNavpointObjectMarker]). Contre `grammar-2026-10-08.6` : un record `ti=12` qui s arretait
// sur `i17` se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.8` (2026-10-08, lot des arrets de la vue B, suite, `ti=10 i18` a `i21`) :
// les quatre composants `managed-object-networked-property-component` se lisent (`FUN_142ed5358`,
// `R(32)` plat, [consumeManagedObjectNetworkedProperty]). Contre `grammar-2026-10-08.7` : un record
// `ti=10` qui s arretait sur l un d eux se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.9` (2026-10-08, lot des arrets de la vue B, suite, `ti=45 i1`) : le
// composant `matchflow-focus-data-component` se lit (`FUN_141167744`, `R(6)` puis `R(4)`,
// [consumeMatchflowFocusData]). Contre `grammar-2026-10-08.8` : un record `ti=45` qui s arretait sur
// `i1` se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.10` (2026-10-08, lot des arrets de la vue B, suite, `ti=12 i13`) : le
// composant `managed-navpoint-top-progress` se lit (`FUN_142ed51d8`, `R(8)` quantifie,
// [consumeNavpointBarreDeProgression]). Contre `grammar-2026-10-08.9` : un record `ti=12` qui
// s arretait sur `i13` se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.11` (2026-10-08, lot des arrets de la vue B, suite, `ti=12 i15`) : le
// composant `managed-navpoint-bottom-progress` se lit (`FUN_142ed4fe4`, `R(8)` quantifie, le meme
// lecteur que `i13`, [consumeNavpointBarreDeProgression]). Contre `grammar-2026-10-08.10` : un record
// `ti=12` qui s arretait sur `i15` se lit jusqu au composant suivant.
//
// ENTREE `grammar-2026-10-08.12` (2026-10-08, lot des arrets de la vue B, suite, point (2)) :
// world-object i0 ([consumeObjectPositionMonde], exception datee GA2-5) lit, porte posee, la ligne
// 0x10 de la table DEFAUT (22/22/22) comme `FUN_14076e524` hors portee, au lieu des largeurs de la
// carte. Contre `grammar-2026-10-08.11` : sur `fccc61cd` (Launch Site), 599 des 635 trames a message
// de degats (genre 0) qui ne se fermaient pas se ferment (+10 949 paquets sains) ; les 20 films de
// la carte v2 ne changent pas. Images-cles (lues hors portee, comme avant) : ti=38 et ti=42 montent,
// deux records ti=37 cessent de fermer (golden de fermeture regenere, justification ecrite).
//
// ENTREE `grammar-2026-10-08.13` (2026-10-08, lot des arrets de la vue B, suite, point (3)) : UN NEW
// QUE LA FERMETURE DE SA TRAME PROUVE CREE SON ENTITE. Un NEW refuse contre une entite vivante
// ([contreditUneEntiteVivante]) est lie a la fin de la trame quand la fermeture prouve son en-tete
// ([World.lierLesNeufsProuves], `neufs_prouves.go`) ; hors de l etendue prouvee il reste refuse.
// Contre `grammar-2026-10-08.12` : `bf15f7ab` slot 553, le NEW du bipede de 14:1094 (trame fermee)
// est lie, et les 69 trames de 14:1094 a 14:1230 se ferment (30 avant). Carte v2 : +1 046 paquets
// sains, aucun perdu, aucun film en baisse.
//
// ENTREE `grammar-2026-10-09` (2026-10-09, lot `assist-film`) : LE FIL DES EVENEMENTS DE LA VUE A SE
// GARDE. La charge `PlayerGameEventSmall` (genre 82, [chargeEvenementJoueurCourt]) rend son `R(32)` de
// tete et son masque de destinataires (`32 x R(1)`) au lieu de les sauter ; un message dont le sac
// texte nomme un couple de participants se range dans `lecture.VueA.Fil` ([lecture.EvenementDeFil]),
// et la marche de killsource les rend (`LectureDeKillsource.Fil`). Contre `grammar-2026-10-08.13` :
// aucun bit n est lu autrement, aucune vue ne s arrete ailleurs ; seuls les messages a couple sont
// gardes en plus.
//
// ENTREE `grammar-2026-10-09.2` (2026-10-09, jalon LK du plan
// `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, branche `feat/ri-lk-images-cles`) : LES
// RECORDS D IMAGE-CLE SE LISENT SOUS LA PORTEE `DAT_144e61ea0` DE L ETAT COMPLET.
//
// RENUMEROTEE (2026-10-09, fusion de `feat/v75` `54b47a2b8` dans `feat/ri-lk-images-cles`) : cette
// entree etait `grammar-2026-10-09` sur la branche ; le lot `assist-film`, fusionne avant elle dans
// `feat/v75`, a pris ce rang. Elle passe au rang libre suivant ; ce qu elle decrit est mesure contre
// `grammar-2026-10-08.13` et s ajoute a `grammar-2026-10-09`, qui ne touche pas la marche d etat complet.
//
// Ce qui change, contre `grammar-2026-10-08.13` :
//   - la portee est un champ du lecteur ([Lecteur.portee]), posee par la seule marche d etat complet
//     la ou le jeu la pose (autour de l etat par defaut et de son mot de controle quand `n1 > 0`,
//     sur toute la boucle de composants quand `n2 > 0`) ; la garde de pleine precision
//     ([fullPrecisionGate]) en depend : les lecteurs de position y lisent le vecteur brut, R(96) ;
//   - la branche absolue d i0 sous la portee lit la forme de `FUN_1406cfe44` : h, R(96), la queue
//     fidele de `FUN_14076e3e4` ([consumeQueueDePoignee] : handle par `FUN_1408f0ac4(.., 0)`, 13 bits,
//     puis le mot de region), R(2) si les trois flottants sont finis ; deux arrets nommes,
//     `position_non_finie` et `largeur_handle_moteur_un` (handle annonce dans un film qui n exclut pas
//     le type de moteur 1, [GrammaireBalayage.MoteurUnPossible], derive du film) ;
//   - la structure de lecture porte l arret d un lecteur ([lecture.EtatArrete], cause
//     [lecture.CauseDArret]) au lieu de le confondre avec un composant non porte (ADR 0037 IR-4) ;
//   - sous la garde, sept exceptions datees du portage lisent comme le jeu
//     (player-desired-respawn-location, crew-order, tacmap-poiicon, -areaofinterest, -displayasset,
//     -cooptetherarea, -waypointstate, qui recoit le niveau du registre) et flock-destination ; le
//     geste [Lecteur.sousLaGardeSinonException] decide, l exception ne vaut que hors de la garde ;
//   - les bascules de profil `PorteeBaseline` et `GrammaireEcrivainI0` sont retirees. Leur critere
//     ecrit (atterrissage des records `ti=35` bornes au-dessus de 50 %) n etait pas tenu par la portee
//     seule (8/599) ; il l est par la portee et la forme d i0 du jeu ensemble, 369/599 = 61,6 %, avec
//     et sans bouchons (`TestKF7EFullStateLoop`) — amendement du critere decide par l utilisateur
//     (U-0, 2026-10-08) ; l instrument qu il nommait, `TestKF35CBaselineScope`, aveugle, est retire.
//
// MESURE (contexte de cuisson, plan LK) : records bipedes d image-cle fermes 410 -> 5 399 sur 10 710
// (28 films) ; ratchet de cuisson `ti=35` 53 -> 666 sur 1 368 (sept bobines) ; `ti=21` 0 -> 1 315 sur
// 1 750 (flock-destination) ; election des ancres, equipes et carte des trames delta identiques. Les
// baisses sont adjugees au plan (§2) : des fermetures de hasard de la base (`60ae07c4` slot 539 ;
// `50247b26`, film illisible des i22). Quatre lectures du jeu sous la garde sont REJETEES par la
// regle du jalon (une baisse, ou l election des ancres changee) et restent hors de la grammaire :
// unit-actor-state, world-object i0, generic-rigid-body-transforms, low-frequency (plan LK, §7 D-17,
// D-18, D-21, D-23).
//
// ENTREE `grammar-2026-10-09.3` (2026-10-09, lot 2.7.d1 du plan
// `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, etapes D1.1 a D1.3, branche
// `feat/ri-lk-images-cles`) : L ETAT COMPLET DU BIPEDE AUX IMAGES-CLES EST LU PAR LA GRAMMAIRE, LES
// FENETRES DE BITS PASSENT DERRIERE LA LECTURE.
//
// Ce qui change, contre `grammar-2026-10-09.2` :
//   - une seule marche de la phase des images-cles ([ScanEtatsDesImagesCles]) rend les armes
//     portees, l inventaire et la marque de portage de chaque record bipede ; les composants sont
//     resolus par NOM dans le registre du film et relus a l etendue de leur occurrence par
//     l assistant unique ([relireLOccurrence], portee et etat complet poses comme la marche) ;
//   - regle d admission par record (decision de l utilisateur du 2026-10-09, U-1 amendee) : (ferme
//     OU n(i22) = 4) ET T1 (au moins une arme, chaque famille non vide au catalogue, marche au-dela
//     du dernier emplacement) ET T2 (masque d i47 egal a la bitmap des compteurs d i22) ;
//   - valeurs publiees d un record admis (U-3, U-4) : familles des emplacements non vides dans leur
//     ordre (plus de « Dynamo Grenade » lue un bit trop tot) ; grenade selectionnee en base 0 ;
//     `DrawnSlot` = emplacement desire en main principale (param[1] d i42), -1 = absence ; rang de
//     capacite publie dans 16..23 seulement, compte hors domaine ; marque = la configuration de la
//     fenetre lue dans i11, i12 et i13 (U-2 (b)) ;
//   - les records non admis passent aux fenetres DERRIERE la lecture, marques recuperes et comptes
//     sous trois replis (`repli_fenetre_armes_image_cle`, `repli_fenetre_inventaire_image_cle`,
//     `repli_fenetre_marque_de_portage`) ; les records admis sont muets pour elles ;
//     `repli_plafond_grenade_par_defaut` ne se compte plus que si la fenetre a recu un record ;
//   - [lireJeuDArmes] rend les trois champs d i42 ; [ScanArmesDesImagesCles] lit les seules armes
//     (le sync des porteurs : ni regles d inventaire ni marque derriere la lecture).
//
// MESURE (28 films, contexte de cuisson, plan D1.0 a D1.2) : 6 507 records bipedes admis sur 10 710
// (f24 2 392 / 3 097, f25 492 / 653, f27 3 574 / 4 905, f20 34 / 1 652, f21 15 / 403) ; valeurs
// publiees egales a la relecture de l instrument sur 100 % des admis ; 0 record a armes fausses admis
// en formats 20-21 ; 1 admis sur 10 407 records du temoin decale d un bit ; 0 debordement ; replis =
// records non admis (4 203), film par film. Bobines du golden : 837 records admis, dont 11 ou la
// fenetre aurait lu autre chose.
