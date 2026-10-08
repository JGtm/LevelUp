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
