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
// `.ai/archive/V7.5/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, branche `feat/ri-lk-images-cles`) : LES
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
// `.ai/archive/V7.5/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, etapes D1.1 a D1.3, branche
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
//
// AMENDEE (2026-10-09, revue adversariale D1.4.6, branche non fusionnee : le rang `.3` n a quitte ni la
// branche ni ses mesures, il est amende au lieu de monter a `.4`, empreinte regeneree) :
//   - le jeu d armes que `FUN_1406d01fc` refuse (param[1] != -1 et param[1] = param[2]) arrete la
//     boucle d etat complet sur i42 du bipede et i38 du vehicule, comme le jeu (thunks `14109d298`,
//     `14116d3cc` ; `TEST AL,AL ; JZ` en `142e2c7ce`) : cause [lecture.ArretJeuDArmesRefuse] ;
//   - un record dont une occurrence deborde n est pas admis (`refusDebordement`) ;
//   - une selection d i47 hors du masque ou du domaine n est pas publiee et se compte
//     (`SelectionHorsMasque`), le contrat du canal delta ([selectionDansLeMasque], partage) ;
//   - un record n est marque recupere par la fenetre d inventaire que si sa lecture porte une valeur.
// MESURE de l amendement (28 films) : 460 records bipedes arretes sur i42, tous sur les trois films
// de format 20 ; admis inchanges (6 507) ; fermetures `ti=35` 5 401 -> 5 397 ; `replay-equiv` (20
// films) : seuls `a349fea8` et `a521164d` changent (4 selections tues) ; killsource identique.
//
// ENTREE `grammar-2026-10-14` (2026-10-14, lot F du plan `.ai/archive/V7.5/PLAN_AJUSTEMENTS_2026-10-14.md`, branche
// `feat/aj14-f`) : LA LECTURE DE LA VUE A RANGE SES MESSAGES DE POUSSEE. Aucun bit n est lu autrement :
// [lireUnMessage] garde les references qu il lisait et jetait, les charges des genres 104 et 105
// ([chargePousseeDuJoueur], [chargeObjetRepousse], `vue_a_poussees.go`) gardent leurs valeurs, et
// `lecture.VueA.Poussees` les porte ([lecture.MessageDePoussee]). Le balayage [ScanKnockbacks]
// (`poussees.go`) rend la paire 105 -> 104 de meme reference 0 dans une liste : joueur pousse, unite
// qui pousse, impulsion (direction `FUN_1406d8288`, norme `FUN_14076d6dc`).
//
// MESURE (2026-10-14) : 235 films du parc, contexte de la cuisson (carte et decoupage poses) : 135
// messages 104 lus, 135 paires, aucun isole ; normes 6,00 / 5,50 / 5,72 m/s seulement ; temoins
// 0d9a9af9, 215e7022 et a6ae19fb : 15 paires (3, 2, 10), dont les 14 rattachees a une piste
// accelerent le slot pousse dans les 300 ms (temoin decale de 5 s : 1 sur 14) ; auteur porteur du
// repulseur sur 130 paires sur 130.
//
// ENTREE `grammar-2026-10-14.2` (2026-10-14, lot « equipement porte » du plan
// `.ai/archive/V7.5/PLAN_AJUSTEMENTS_2026-10-14.md`, branche `feat/equip-porte`) : `ti=37 i28
// equipment-tracked-object-handles-stack-component` LIT n ENTREES, PLUS n+1. Le lecteur du jeu
// `FUN_140f72dec` lit R(4) n (`FUN_1424e1d48`), le range en `+0x544`, puis avance le pointeur de
// 8 octets AVANT de le comparer a `+0x544 + (n+1)*8` : n appels de `FUN_1408f0ac4(…,0)`. Le port
// (`consumeEquipmentTrackedStack2`) en faisait un de plus : le bit de presence d une entree fantome
// etait lu a CHAQUE lecture du composant (et 15 bits de plus quand il valait 1) : la marche se
// decalait et sortait par un rejet hors datum.
//
// MESURE (238 films du parc, contexte de la cuisson : entree de catalogue puis
// `PoserLaCarteEtLeDecoupage`) : trames portant un i28 d objet du monde fermees 0 sur 4 839 -> 4 726
// sur 4 841 ; trames fermees 7 538 932 -> 7 543 734 sur 7 668 648, aucun film en baisse ; deltas ti=37
// lus +2 832. Ce sont les trames ou l objet PORTE change d etat (pose de la faille du translocateur,
// 1b2d9e08 a 164,13 s) : elles etaient toutes perdues.
//
// ENTREE `grammar-2026-10-14.3` (2026-10-14, lot « equipement porte », branche `feat/equip-porte`) :
// L EQUIPEMENT PORTE EST LU. Aucun bit n est lu autrement : la couche de capture rend la valeur de
// quatre composants de ti=37 que la marche traversait deja (i24 energie, i21 activation, i22 signal
// de controle, i28 pile des poignees), par les MEMES lecteurs que le dispatch
// (`equipement_porte_lecteurs.go`, garde `TestCaptureConsumesSameBitsAsDispatch`). Un canal de la
// marche des trames ([canalDeLEquipementPorte], `equipement_porte.go`) suit les objets ti=37 crees
// attaches a un bipede (i10, parent en base 0x200) jusqu a leur suppression, et [ScanEquipementPorte]
// rend leurs lectures brutes. `EnergieDEquipement` dequantifie i24 comme `FUN_141087bec`
// (`FUN_1406d84b4`, 0..1, 14 bits, extremites exactes).
//
// MESURE (238 films du parc, contexte de la cuisson) : poses de faille = energie de l objet porte du
// translocateur a 0 puis i28 a une entree, toujours 1,619 a 1,641 s apres une activation i21
// (123/123) ; temoins 1b2d9e08 164,06 s, 6acce632 53,3 / 158,5 / 304,2 s a la frame pres.
// Repulseur 0d9a9af9 : 0,6 -> 0,4 -> 0,2 -> 0 aux trois activations, suppression = consommation.
//
// ENTREE `grammar-2026-10-15` (2026-10-15, lot « equipement porte », demande du user : « si on a
// l evenement pour le champ de reparation, on rebranche le son de fin ») : LES OBJETS QUE
// L EQUIPEMENT PORTE ENGENDRE, ET LEUR FIN. Aucun bit n est lu autrement : les entrees presentes de
// la pile i28 (`FUN_1408f0ac4(…,0)`, 13 + 2 bits) sont des HANDLES — `FUN_1406d3140` categorie 0,
// base 0x200 (`FUN_140d10bb0` pose 0x200 aux categories 0 et 1) — que le lecteur rend desormais
// (`EquipementLu.Suivis`). Le canal retient chaque vie nommee, meme apres la fin de l objet porte,
// et rend sa suppression (`tracked-end`).
//
// MESURE (238 films du parc, contexte de la cuisson, `champ_fin_research_test.go`) : 1 772 objets
// nommes par une pile. Champ de reparation : 18 sur 18 sont des ti=41 nes dans le paquet de la pile,
// 18 suppressions lues (12 apres la fin de l objet porte, que le retrait de la pile ne verrait pas) ;
// durees : 11 de 11,35 a 12,24 s, 6 de 1,5 a 4,6 s (champ detruit), une de 16,95 s. Capteur 834,
// ecran 241, traqueur 49 : tous ti=41 nes dans le paquet. Le retrait de la pile precede la
// suppression de 0,2 a 0,7 s (5676a9ba).
//
// ENTREE `grammar-2026-10-15.2` (2026-10-15, plan `.ai/archive/V7.5/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, branche
// `feat/ri-ecart-108`, etapes R1 a R3) : TROIS LECTURES DU JEU ENTRENT DANS LA MARCHE DES IMAGES-CLES.
//
//   - R1, LA TABLE D ENTREES (`keyframe_entrees.go`). La table d image-cle compte une entree par slot
//     (`FUN_1408be074`, 0x1fff) ; un slot libere s y ecrit en 108 bits d archetype `0xffffffff`
//     (`FUN_142e2bfd0`, `FUN_1408f1948`). La fermeture d un record lit les t - s - 1 entrees qui le
//     separent de l ancre suivante, le dernier record d un paquet ferme sur la fin de table ; la
//     preuve d image-cle saute les slots liberes et exige le slot `slot + 1 + k` ; les marches
//     deterministes lisent la table entree par entree ; la « fin de table » a 2 048 sentinelles du
//     balayeur, que le jeu n a pas, est retiree.
//   - R2, LE BIT DES FILMS 1.8 (`malleable_du_film.go`). Le bloc malleable d i53 (`FUN_1407efc5c`) n a
//     pas son drapeau final dans les films qui declarent la version majeure 0x25 (decision de
//     l utilisateur du 2026-10-15, option b).
//   - R3, world-object i0 (`FUN_14076e29c`) et generic-rigid-body i18 (`FUN_142f036f0`) sous la garde de
//     pleine precision : R(96), comme le jeu (decision de l utilisateur du 2026-10-15, option a pour i0).
//
// MESURE (28 films, contexte de la cuisson, contre `feat/v75` `39ead801f`) : bipedes fermes 5 397 ->
// 8 564 sur 10 710 ; tous archetypes 242 684 -> 418 264 sur 460 730 ; `ti=37` 2 413 -> 27 587, `ti=38` 30 733 ->
// 113 049, `ti=42` 3 125 -> 21 449, `ti=43` 10 919 -> 37 959, `ti=10` 14 265 -> 30 608 ; admissions
// T1 et T2 inchangees (6 507) ; equipes identiques ; election des ancres changee sur `64e8adfa` seul
// (deux regions, la fausse ancre slot 1536 `ti=0` refutee, dix records rendus) ; carte delta : seuls
// des paquets de `60ae07c4` passent fermes (+49, +1 597 sous le decoupage MPP). Baisses, toutes
// acceptees par la decision du 2026-10-15 : `50247b26` `ti=38` 451 -> 159 et `ti=42` 76 -> 16 (film de
// format 20 sans section d identification). Douze films HI_1_8_0 du parc : bipedes fermes 1 263 ->
// 7 042, aucune perte.
//
// ENTREE `grammar-2026-10-16` (2026-10-16, integration groupee des lots film du 15/10, branche
// `feat/integ-film`) : TROIS LOTS, UN RANG. Les lots « ks-a » (via « ks-d »), K-B et « faille a la
// pose » avaient chacun pris `grammar-2026-10-15.3` sur leur branche ; aucun de ces rangs n a ete
// fusionne seul, et l integration les reunit au premier rang libre apres `grammar-2026-10-15.2`
// (`grammar-2026-10-15.3` est abandonne : trois contenus differents l ont porte, un fait cuit sur une
// des branches ne doit pas passer pour frais). Une sous-entree par lot, dans l ordre des fusions.
//
// LOT « ks-a » (doublons de morts) : UN MARQUEUR DE FIN = UN EVENEMENT. Le lecteur des temps forts
// dedupliquait par POSITION du xuid et non par marqueur de fin : un faux motif de xuid valide
// (8 octets, 0x2d, 0xc0) dans la fiche d un joueur precedait le vrai de plusieurs milliers de bits,
// les deux retombaient sur le meme bloc de 60 octets et le meme marqueur, et chaque evenement sortait
// deux fois sous deux xuid. Il garde desormais le candidat le plus proche du marqueur
// (`retenirLePlusProche`). Aucun bit n est lu autrement ; ce qui change est QUELS xuid sortent.
// MESURE : 34db6b33 (fiche de Partycz, faux 2252025040223036 a 5 067 bits du vrai 2535472452512509)
// 28 groupes d evenements sous deux xuid avant, 0 apres ; parc local de 2 677 films : 1 film touche
// avant, 0 apres.
//
// LOT K-B (`fix/ks-b-equipe-bots`) : L EQUIPE D UN BOT SE LIT A LA POSITION DE SON BUILD, ET LES
// FRAGMENTS DU BALAYAGE SONT ECARTES. `profile.Rev` monte a `profile-2026-10-16`
// ([profile.PositionEquipeBotPour]). `bot_metadata.go` lit le bloc de 44 octets brut et en rend
// l equipe et le jumeau a la position du build ([LectureDesBots]) : sur HI_1_10_0 a HI_1_12_0,
// l octet 0x25 lu jusqu ici tombait dans la copie du nom (equipe 0 pour tout bot). Dans un paquet qui
// ferme, un nom balaye suffixe strict d un nom ecrit (`43 Hollis` pour `343 Hollis`) est ecarte et
// compte ([PaquetBotMetadata.FragmentsEcartes]) : sur ces builds chaque entree en produisait un, qui
// rendait chaque paquet incomplet. MESURE (617 films du parc dont la feuille porte un bot, decodage
// killsource de production, un film par processus) : bots dont l equipe se lit, accord avec la
// feuille 888/888 (HI_1_13_0), 111/111 (HI_1_12_0), 2/2 (HI_1_11_0), 10/10 (HI_1_10_0) ; avec
// l entite `ti=9` 85/85 et 1/1 ; paquets BOT_METADATA incomplets 707 -> 0.
//
// LOT « faille a la pose » (`feat/faille-pose`, question du user : « le mecanisme de pose repose sur
// l evenement de pose, pourquoi dependrait-il de la teleportation ? ») : LA CREATION DES OBJETS
// ENGENDRES EST LUE, AVEC SA POSITION. Aucun bit n est lu autrement. Le canal de l equipement porte
// retient chaque creation (record NEW) de la trame et les quanta de sa position i0, lus par le
// lecteur des pistes du monde (`lireQuantaObjetDuMonde`, extrait de `decodeWorldObjectPos` sans
// changement : chemin dominant, region de la carte, quanta satures rejetes) ; quand la pile i28 nomme
// une vie pour la premiere fois, sa creation est jointe a la lecture (`Spawned`), et
// [ScanEquipementPorte] la dequantifie aux bornes de la carte. MESURE (36 films a translocateur, 123
// poses, contexte de la cuisson, `TestFailleObjet`) : le paquet ou l energie du translocateur tombe a
// 0 cree une entite ti=38 (la faille), nommee par la pile au paquet suivant ; 123/123 positions lues.
// Contre l arrivee du premier echange (74) : ecart XY mediane 0,02 m, max 0,13 (l arrivee est 0,11 a
// 0,53 m plus haut) ; contre le porteur a l image de la pose (120) : XY mediane 0,04 m, max 0,17.
//
// ENTREE `grammar-2026-10-18` (2026-10-18, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E8.1 (b) (c),
// decision utilisateur D-16, option B du pilote) : L OCCUPATION DES VEHICULES QUE LA MARCHE DE
// KILLSOURCE RECOLTE EST RENDUE. Aucun bit n est lu autrement : le canal des morts recoltait deja les
// lectures d `object-parent-state` (i10) de la bande bipede pendant la marche de killsource, et les
// jetait. [LectureDeKillsource.Occupation] les rend, triees et dedoublonnees comme celles de la
// cuisson ([ScanMarcheDesTramesAvec], `TestLOccupationDeLaMarcheDeKillsourceEstCelleDeLaCuisson`) :
// la passe des positions du collecteur y lit qui est a bord (tenue hors embarquement, siege du
// vehicule occupe) sans seconde marche. `killsource.Rev` monte avec cette valeur.
//
// ENTREE `grammar-2026-10-21` (2026-10-21, controle n° 24 des controles de la v8, lot L3 des retours
// de controle) : UNE PISTE DE PROJECTILE QUE LA MARCHE A LUE SE
// GARDE HORS DE LA BANDE DES IMAGES-CLES. Aucun bit n est lu autrement : la marche nommait deja
// l archetype de chacun de ses records, et [FilmContext.pistesDerriereLaMarche] jetait ceux du
// projectile dont le slot sortait de la bande — une bande relevee dans des images-cles espacees de
// vingt secondes, qui ne voient pas un objet vivant moins d une seconde ([laMarcheDesigneSeule]). Une
// bande vide ne ferme plus le balayage des projectiles. MESURE (`grenade_vol_research_test.go`,
// contexte de la cuisson, un film par processus) : lancers suivis d une naissance de piste de
// production a 200 ms, avant -> apres : d6918972 135 -> 174 sur 174, d9781168 131 -> 172 sur 172,
// bfecd02b 67 -> 95 sur 95, 4f77afc1 190 -> 210 sur 215, 72b0a25e 2 -> 81 sur 81 (bande d un seul
// slot), 92c950ee 72 -> 76 sur 76 ; chaque lancer est aussi suivi d un message `projectile_detonate`
// (genre 5) de la vue A dans les 5 s. Mini-bobine : 51 -> 81 pistes de projectile.
//
// ENTREE `grammar-2026-10-25` (2026-10-25, diagnostic des images-cles sans vehicule du match 4898d586) :
// UNE ANCRE D IMAGE-CLE ELUE PAR LE REPLI SE PROUVE PAR LES ENTREES LIBRES QUI LA PRECEDENT. La table
// d image-cle porte une entree par slot ; entre un record de slot p et le suivant de slot s elle compte
// donc exactement s-p-1 slots liberes de 108 bits, contigus, juste devant le second. Un candidat qui a ce
// compte devant lui (ecart de slot >= 1) est ATTESTE et bat tout candidat non atteste, quel que soit son slot
// ou sa generation (`keyframe_world_attestation.go`, premiere cle de [kfCand.betterThan]). Le defaut
// corrige : apres le bipede 533, la fausse ancre ti=20 de slot 542, prise dans le corps des vehicules,
// battait le vrai record 770 par son slot bas et faisait ecarter les huit vehicules ti=40 des deux
// images-cles du match (aucune des 236 entrees libres attendues devant 542). Les elections se comptent
// attestees ou non (`KeyframeWalkStats.ElectionsAttestees`, `ElectionsNonAttestees`). Les ancres lues
// changent, donc le recensement des images-cles et tout ce qui en derive.
