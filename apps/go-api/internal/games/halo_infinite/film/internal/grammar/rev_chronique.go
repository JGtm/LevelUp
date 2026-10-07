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
// `grammar-2026-09-24` a `grammar-2026-09-27.3` dans `rev_chronique_archive_7.go`.
// La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`), le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place), puis le
// 2026-09-22 (lot 5.20.1, une CINQUIEME archive), puis le 2026-09-24 (lot M4b, une SIXIEME), puis
// le 2026-10-03 (integration de la vague 1 de la campagne de grammaire, une SEPTIEME), puis le
// 2026-10-06 (lot VA de la campagne, rangs `grammar-2026-09-27` a `.3` verses dans la septieme).
// C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du `grammar-2026-10-02`.

// ENTREE `grammar-2026-10-02` (2026-10-02, lot L0 de la campagne de grammaire,
// `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA FERMETURE D UN PAQUET SUIT LES REGLES DE
// L ECRIVAIN (decision D2).
//
// Un paquet delta est FERME quand sa vue C se lit jusqu a son terminateur avec un reste de 0 a 7
// bits nuls ET qu aucune regle de l ecrivain n est contredite (`ecrivain_invariants.go`) : sortie de
// la vue B sur un en-tete rejete (`FUN_142f2e174`, `FUN_142f2cee0`, `FUN_142f2cc78`), ordre NEW*,
// DELTA*, DEL* a slots croissants (`FUN_14076b9c8`), masque que `FUN_142e2da44` peut ecrire (aucun bit
// au-dela du dernier composant de l archetype, juge a la traversee ; epars de sept au plus a index
// croissants ; dense au-dela), vue C de l enregistreur (`FUN_142f2c3b0`, `FUN_14076b0e8`,
// `FUN_1406d5bf4` : kind 0, index croissants, 32 entrees au plus, bit d en-tete a 0, jamais le code
// analogique 63). [LectureVueC] porte les deux verdicts (`FermeeAuBit`, `Fermee`) et la premiere
// regle contredite.
//
// Ce qui change en sortie : les deux lecteurs de `Fermee` en production. `debutParFermeture` prend
// le premier candidat d ou le paquet FERME, a defaut le premier d ou il ferme au bit pres (la tete
// gardee, le paquet non ferme) ; ce second rang est le repli `repli_debut_de_liste_ferme_au_bit`,
// compte dans `coverage.fallbacks` (un nom neuf du rapport, aucun champ neuf). Le collecteur du tir
// continu voit un trou la ou il lisait une vue C factice. Le masque se lit par `lireMasque` (memes bits que `consumeMask`). La carte de fermeture
// classe la sortie par rejet avant les causes de la vue C, la regle contredite apres elles, et
// requalifie le bloc 0xbc en desalignement. Mesure sur 20 films (`campagne_grammaire_2026-10-01/
// LOT_L0.md`) : 284 704 paquets fermes au bit pres avant, 276 327 fermes apres ; sous le juge de ce
// lot, aucun paquet sain perdu, aucun film en baisse en paquets ni en records utiles sains (sous
// l ancien juge a trois regles, la sortie par rejet requalifie environ 1 008 paquets sains : c est
// l objet de D2).
//
// `killsource.Rev` NE MONTE PAS : sa fermeture hache la valeur de cette constante, mais aucun de ses
// lecteurs ne lit `Fermee` ; sortie JSON identique a l octet sur les 19 temoins, golden regenere a
// revision constante. `source.Rev` non plus (un accesseur neuf, aucune valeur lue changee).
// `replay.SchemaVersion` reste 76 : le document ne change que par les revisions de calque et par le
// tir continu (compteurs, rafales lues sur des paquets factices retirees) et les etats de mouvement
// de quelques listes, que la revision de grammaire des calques signale deja ; `replay-equiv` : 5
// etapes sur 61 divergent sur les 20 films, les 56 autres sont identiques a l octet.
//
// ENTREE `grammar-2026-10-02.2` (2026-10-02, lot L8 de la campagne de grammaire,
// `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : `ti=3 low-frequency` EST PORTE, ET `high-frequency`
// SE LIT PAR LA TABLE DE L ARCHETYPE.
//
// `low-frequency` (`ti=3 i0`) se lit par FUN_142ed4aec (table 0x143d07b40, ecrivain FUN_142eda938) :
// position, orientation, R(16) + R(8) + R(2), puis R(6) entrees {R(3) drapeaux, position et
// orientation sous drapeau, R(16), R(5)} ; il n etait pas porte (traversee arretee). `high-frequency`
// est enregistre sous DEUX tables : `ti=3 i1` (FUN_140e460fc, table 0x143d07af0, FUN_142ed4880 :
// R(16) + R(8) + R(2), 26 bits), lu jusqu ici par le R(8) de `ti=4 i0` (FUN_140e462d8, table
// 0x143d06a60, FUN_14076d034), qui ne change pas ; un autre archetype ne le lit plus
// (`components_frequences.go`). `ecs_table.tsv` porte les trois lecteurs ; le controle G6
// (`ecs_dispatch_table_guard_test.go`) tient le routage par table.
//
// Ce qui change en sortie : les records `ti=3` se traversent, les paquets qui les portent se lisent
// plus loin (carte de fermeture et mesures : `campagne_grammaire_2026-10-01/LOT_L8.md`).
//
// ENTREE `grammar-2026-10-02.3` (2026-10-02, lot L3a de la campagne de grammaire,
// `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA FIN DU MOTEUR DE PARTIE, LUE DANS LE JEU.
//
// Six composants des archetypes du moteur (`ti=0`, `ti=1`, `ti=2`, index `i11` a `i17`) passent de
// « non porte » (arret du record) a porte, chacun sur son lecteur et son ecrivain relus dans
// HaloInfinite.exe HI_1_13_0 (`components_moteur_de_partie.go`) : `i11` R(128) ; `i13` compte R(13)
// puis un bit par volume ; `i14` tronc commun puis la forme que le NIVEAU du registre du film
// designe (`CMP R9D, 2` de `FUN_142f0328c`) ; `i15` masque R(64) puis les fentes presentes
// (`FUN_1407ee87c`) ; `i16` R(7) + R(1) ; `i17` R(8). Le lecteur de minuteur `FUN_140d580d0` (et sa
// forme longue `FUN_142ba78dc`) n existe plus qu une fois (`lecteur_minuteur.go`, garde-rail
// `lecteur_minuteur_guard_test.go`) ; ses cinq copies (`ti=5 i2`, `ti=0 i5`, `i6`, `i7`, `i12`)
// lisent les memes bits qu avant.
//
// Ce qui change en sortie : les records du moteur se lisent jusqu au bout au lieu de s arreter sur
// le premier de ces composants ; les paquets qui les portent peuvent fermer. Mesures et pertes
// instruites : `campagne_grammaire_2026-10-01/LOT_L3a.md`.
//
// ENTREE `grammar-2026-10-03` (2026-10-03, lot L4a de la campagne de grammaire,
// `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LES COMPOSANTS PROPRES AU VEHICULE (`ti=40`, i30 a
// i47) SE LISENT DANS LES RECORDS A MASQUE, ET LA PORTE `+0x818` EST UNE LOI DU MASQUE.
//
// Ce qui change : `composants_vehicule_ti40.go`, dernier maillon de la chaine de dispatch, lit
// `i30` a `i47` chez leurs deserialiseurs (`FUN_142f04994`, `FUN_14115f33c`, `FUN_142f04b70`,
// `FUN_142f0496c`, `FUN_142f04b34`, `14116d3cc` -> `FUN_1406d01fc`, `FUN_142f04884`, `FUN_142f04a00`,
// `FUN_142f04a4c`, `FUN_142f04ac0`, `FUN_142f02508`, `FUN_142f04bcc`, `FUN_142f04a20`), et `i33`
// (`FUN_142f02474` -> `FUN_14320c4c8`) et `i34` sous la porte `+0x818`. Les deux ecrivains du
// masque (`FUN_142f09c74`, difference ; `FUN_142f0cca0` via `FUN_143208c18`, capture qui remplace
// le masque complet de `FUN_142e32138`) ne posent les bits 33 et 34 que porte posee : dans un record
// lu avec un masque (DELTA, NEW), l annonce PROUVE la porte. Le repli
// `repli_physique_de_type_de_vehicule_supposee` est retire ; son compteur s appelle
// `VehicleTypePhysicsByWriterLaw`. Un etat complet d image-cle (sans masque, `FUN_142e2c690`) ne
// lit aucun de ces composants hors `i37` ([Lecteur.etatComplet]) : la porte y depend du chassis
// (lot L4b), et la marche d image-cle est inchangee.
//
// Mesure sur 20 films (`campagne_grammaire_2026-10-01/LOT_L4a.md`) : 276 327 -> 277 743 paquets
// sains (+1 416), records utiles sains +27 482, AUCUN sain perdu sur aucun film ; 1 435 paquets
// gagnes fermes au bit, dont 35 contredits par une regle de l ecrivain (2,4 %). Les 1 666 paquets
// arretes par un composant `ti=40` non porte passent a 0.
//
// `killsource.Rev`, `source.Rev` et `objectives.Rev` NE MONTENT PAS (goldens regeneres a revision
// constante) : sortie `cmd/killsource json` identique a l octet sur 18 des 19 temoins, et sur
// `e5adf7b2` seul le diagnostic de l oracle de calibration (non persiste) change. Le document
// change par les morts de vehicule lues (records qui ne se desynchronisent plus apres l etat de
// mort), les compteurs de repli de killsource et du tir continu, et quelques listes de plus :
// `replay.SchemaVersion` ne monte pas, la revision des calques le signale.
//
// ENTREE `grammar-2026-10-03.2` (2026-10-03, integration de la vague 1 de la campagne de grammaire
// et corrections de sa revue adverse, `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : `low-frequency`
// N EST PAS PORTE DANS UN ETAT COMPLET D IMAGE-CLE, ET LA TETE PORTE UNE VALEUR A ELLE.
//
// La boucle d etat complet du jeu (`FUN_142e2c690`) pose la portee `DAT_144e61ea0` sur toute la
// boucle de composants (142e2c6b8 / 142e2c76a, lecteur appele en 142e2c7c9) ; sous elle,
// `FUN_14076e494` lit la position brute, R(96) (`FUN_14076f91c`, `FUN_1411b259c`). La marche d image-cle
// du depot ne pose pas cette portee : `ti=3 i0` (FUN_142ed4aec), porte au rang `.2` du 2026-10-02,
// y etait lu a la largeur du delta. Il rend desormais « non porte » sous [Lecteur.etatComplet] (la
// traversee s arrete, comme avant ce rang) ; sa lecture en record a masque ne change pas.
//
// LA VALEUR : `grammar-2026-10-03` etait aussi celle de la branche du lot L4a seul, sans L8 ni L3a.
// Une revision designe un contenu : la tete de la vague prend un rang qu aucune branche de lot n a
// porte, pour qu aucun fait ecrit par un lot seul ne se relise a jour.
//
// Ce qui change en sortie, contre la tete integree du rang precedent (`campagne_grammaire_2026-10-01/
// vague1_tsv/revue_*`) : carte de fermeture v2 des 20 films identique (paquets, records utiles,
// bloquants) ; `cmd/killsource json` identique a l octet sur les 19 temoins ; `keyframe_closure.golden`
// sans compte qui bouge, `ti=3` retrouve son bloquant `i0 low-frequency` ; fixtures de contrat
// identiques hors chaines de revision. `killsource.Rev` reste `killsource-2026-09-27` (empreinte
// recopiee, cf. sa chronique) ; `replay.SchemaVersion` reste 77.
//
// ENTREE `grammar-2026-10-03.3` (2026-10-04, lot LS de la campagne de grammaire,
// `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA SIGNATURE DU LOCALISATEUR SE LIT SUR TOUT OBJET DE
// L ARCHETYPE `high-frequency`, PAS SUR LE SEUL SLOT 123.
//
// Ce qui change (`localisateur.go`) : dans un paquet a evenements dont le slot 123 ne porte aucune
// signature, le premier delta de 35 bits a composant unique d un AUTRE slot que le monde lie a
// l archetype `high-frequency` ([archetypeHauteFrequence], la cle de table du dispatch, `FUN_140e462d8`)
// ouvre la boucle de records. Tous les objets de l archetype ont le meme ecrivain (`FUN_142eda680`),
// et la vue B ecrit ses DELTA par slot croissant (`FUN_142f2e174`, `FUN_14076b9c8`) : dans les modes a
// objectif porte, le premier delta haute frequence est celui d un autre slot (124 a 135, 304). Ordre
// propre a chaque site : la cuisson ([localiserLaListe]) essaie la signature du slot 123, puis la
// fermeture par NEW de tete, puis cette signature, sans repli a largeur libre ; les deux marches qui
// lisent les morts essaient la signature du slot 123 a la generation du monde, puis celle-ci, puis
// le repli a largeur libre.
//
// Mesure sur 20 films (`campagne_grammaire_2026-10-01/LOT_LS.md`) : 313 495 -> 332 624 paquets
// sains (+19 129), records utiles sains +176 939, AUCUN sain perdu sur aucun film ; listes non
// localisees 47 854 -> 26 043. `killsource.Rev` monte (`killsource-2026-10-04`) : la voie publiee
// de 229 morts passe du balayage a la marche sur les 19 temoins, sans autre valeur changee.
// RETIRE au rang `.5` (revue adverse de la vague 2) : ce rang n a vecu que sur la branche de la
// campagne.
//
// ENTREE `grammar-2026-10-03.4` (2026-10-04, lot LT de la campagne de grammaire,
// `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA PREUVE PAR CHAINE REFUSE UN RECORD DONT LE
// MASQUE CONTREDIT L ECRIVAIN.
//
// Ce qui change (`debut_de_liste.go`, [pasDEssai]) : la chaine de tete d une liste d evenements
// ([debutParChaine]) ne traverse plus un record NEW ou DELTA dont le masque contredit
// `FUN_142e2da44` (bit au-dela du dernier composant de l archetype, `i < *(desc+0x4320)` ; dense
// d au plus sept composants ; epars a index non croissants). Le debut de la signature est garde.
// Le second rang de [debutParFermetureRangee] (repli `repli_debut_de_liste_ferme_au_bit`) est
// inchange.
//
// Mesure sur 20 films (`campagne_grammaire_2026-10-01/LOT_LT.md`) : 332 624 -> 332 668 paquets
// sains (+44), records utiles sains +1 023, AUCUN sain perdu sur aucun film ; les cinq sains perdus
// de la vague 1 (`fb1a1a72` 7:92, `1c4c63c2` 11:1620, `4f77afc1` 25:874, 37:1188, 59:682) sont
// retrouves. `killsource.Rev`, `source.Rev` et `objectives.Rev` NE MONTENT PAS : sortie
// `cmd/killsource json` identique a l octet sur les 19 temoins (killsource ne passe pas par la
// chaine de tete).
//
// ENTREE `grammar-2026-10-03.5` (2026-10-04, vague 2 de la campagne de grammaire apres sa revue
// adverse, `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LU ET LT SUR `feat/v75`, LS RETIRE ; LA
// CHAINE DE TETE SUIT L ORDRE DE LA VUE B ; LA LARGEUR DE LA SIGNATURE SE DERIVE DU CADRE.
//
// Ce qui change, contre `grammar-2026-10-03.2` tel que `feat/v75` le porte (`6fa631df0`) :
//   - le rang `.3` (LS) est RETIRE : ses ordres de localisation par site sont mesures, pas lus dans
//     le jeu, et le bit nul qu il exigeait devant la signature haute frequence n est pas ecrit par
//     le jeu (revue adverse de la vague 2) ; le localisateur est celui de LU ;
//   - le rang `.4` (LT) reste : la chaine de tete refuse un masque que l ecrivain n ecrit pas ;
//   - [chaineJusqua] suit aussi la loi d ecriture de la vue B ([ordreDeLaVueB] : NEW*, DELTA*,
//     DEL*, slots strictement croissants dans chaque groupe ; `FUN_142f2e174`, `FUN_14076b9c8`),
//     jusqu au record du debut localise inclus ;
//   - [largeurDeSignature] : la largeur de la signature stricte vient des ecrivains (`FUN_1406d3140`,
//     `FUN_1406cdc04`, `FUN_142e2da44`, `FUN_142eda680`) sous le cadre du film ; 35 bits au cadre par
//     defaut.
//
// Mesure sur 20 films contre `6fa631df0` (`campagne_grammaire_2026-10-01/vague2_tsv/revue/`) : carte
// v2 313 495 -> 313 542 paquets sains (+47 : LT +44, ordre de la chaine +3), records utiles sains
// +1 087, AUCUN sain perdu sur aucun film ; la largeur derivee ne change pas un octet de la carte.
// `cmd/killsource json` identique a l octet sur 20 films : `killsource.Rev` reste
// `killsource-2026-09-27`. `replay-equiv` : divergent `artifact`, `movementStates.stats`,
// `continuousFire.stats` et `continuousFire` (2 films) ; `objectives`, `killRefs`, `vehicles`,
// `movementStates` identiques : `objectives.Rev` ne monte pas. `replay.SchemaVersion` reste 78. Ce
// rang remplace la valeur du meme nom de la tete d integration d avant la revue (`da6ecda38`, jamais
// fusionnee, empreinte egale a celle du `.4`).
//
// ENTREE `grammar-2026-10-04` (2026-10-04, lot 2.7.a de la representation intermediaire,
// `.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE2_2026-10-03.md`) : LES MORTS D OBJET ET L OCCUPATION
// SONT UN CANAL DE LA MARCHE DES TRAMES ; LA MARCHE A HUIT VUES EST RETIREE.
//
// Ce qui change, contre `grammar-2026-10-03.5` :
//   - [ScanMarcheDesTramesAvec] rend les morts d objet et l occupation ([canalDesMorts]) quand on les
//     lui demande — la cuisson, pour un calque de vehicules balaye : les records de la vue B de
//     chaque trame, sous la regle d acceptation de [objectDeathHarvest] ; la marche a huit
//     vues (chronologie des images-cles, calibration d `IDLowBits`, deroulage) est retiree, et la
//     largeur d identifiant bas est celle de l en-tete de la marche ;
//   - une liste d evenements que le debut de liste de la cuisson ne localise pas est recuperee pour
//     ce canal seul ([debutRecupere] : signature puis largeur libre, `repli_localisation_largeur_libre`),
//     sous le monde de la marche, rendu intact ;
//   - [ObjectDeathStats] perd les quatre champs de la calibration ; le repli
//     `repli_cadre_de_marche_par_defaut_conserve` est retire ;
//   - les morts de vehicule se lisent sous les largeurs MPP de la marche des trames, celles du
//     contexte, et plus sous celles que les vehicules calibrent sur les poses des formats sans
//     largeur relue : une largeur mesuree n entre pas dans la marche de toutes les entites.
//
// Preuve sur les vingt films du corpus d equivalence contre `87cdfa761` (passe `ri27c`) : etats de
// mouvement et tir continu IDENTIQUES sur les vingt ; `cmd/killsource json` identique a l octet sur
// les 19 temoins (`killsource.Rev` et `objectives.Rev` ne montent pas) ; divergent `vehicles` et
// `artifact`, plus les deux etapes neuves `vehicleDeaths` ; records de mort de vehicule de la
// cuisson 142 -> 148 sur les dix films qui en portent (`e5adf7b2` 17 -> 15 et `60ae07c4` 1 -> 0,
// lus jusqu ici sous les largeurs calibrees ; `1c4c63c2` 4 -> 7, `084a804d` 35 -> 37).
// `replay.SchemaVersion` reste 78.
//
// ENTREE `grammar-2026-10-05` (2026-10-05, item 2.7.a0 de la representation intermediaire ;
// decision de l utilisateur du meme jour) : LE DECOUPAGE MPP D UN FILM DES FORMATS 20, 21, 24 ET 25
// EST CELUI QU IL DECLARE PAR LA TAILLE D ETAT DE CREATION DE SES OBJETS.
//
// Ce qui change, contre `grammar-2026-10-04` :
//   - [FilmContext.DeclarationMPP] lit, dans la premiere image-cle qui en porte, le mot `n1` de
//     chaque record d un archetype de la cle ([profile.CleDuDecoupageMPP]) et rend le decoupage
//     qu ils designent tous ([profile.MPPPourTailleDeclaree], `profile-2026-10-05`) ;
//   - [FilmContext.ResolutionMPP] rend celui de la version de format quand elle le porte (27), la
//     declaration sinon ; c est la porte unique des poses d equipement et des socles et vehicules
//     de la cuisson, et la cuisson la pose sur son contexte pour toutes ses lectures
//     (`replay.poserLeDecoupageMPPDuFilm`) ; la calibration sur les poses ne decide plus que pour un
//     film qui ne declare rien sans discordance ;
//   - [ResolutionMPP.Relue] devient [ResolutionMPP.Decide], avec la provenance du decoupage.
//
// killsource ne change pas : la declaration n entre ni dans l en-tete de la marche ni dans la
// preuve des ancres d image-cle, et son contexte garde l invariant.
//
// ENTREE `grammar-2026-10-06` (2026-10-05, lot VA de la campagne de grammaire, etape V1) : LA VUE A
// SE LIT MESSAGE PAR MESSAGE, PAR UNE SEULE LECTURE ; AUCUNE SORTIE NE CHANGE.
//
// Ce qui change, contre `grammar-2026-10-03.5` (`87cdfa761`) :
//   - la table des 123 genres de message et leurs versions natives sont portees, avec les charges de
//     47 genres (45 du lot LN, plus Script (15) et biped_throw_initiate (39) de la recherche R2, qui
//     lit aussi les positions a index des genres 5 et 6 sur la region jouee) ; 13 genres sont vides
//     (`vue_a_genres.go`, `vue_a_versions.go`, `vue_a_charges*.go`) ; la simulation de
//     l enregistreur entre dans l identite du film (`profile-2026-10-06`) ;
//   - la regle des versions en deux classes : table du film EGALE a la table native (film recent),
//     PREFIXE STRICT (film ancien), sinon la vue A ne se lit pas au-dela de sa tete ;
//   - [lireLaVueA] est la seule lecture de la vue A (`consumeVueA` supprime) : la marche des trames la
//     joue une fois par trame en rangeant la tete ([rangerLaTete]) et la passe a [lireTrameParRangs] ;
//     la tete donnee aux canaux et la route vers la localisation sont lues a l identique ;
//   - la fin de la vue A lue est rangee dans `lecture.Paquet.VueA` ; elle ne decide d AUCUNE
//     localisation (etape V2).
//
// Mesure sur 20 films contre `87cdfa761` (`campagne_grammaire_2026-10-01/LOT_VA_V1.md`) : carte v2
// (`fermeture_paquets.tsv`) IDENTIQUE A L OCTET, 313 542 paquets sains ; `cmd/killsource json`
// identique a l octet sur 20 films : `killsource.Rev` reste `killsource-2026-09-27` ; `replay-equiv` :
// seule l etape `artifact` diverge (chaine de revision), `objectives` identique : `objectives.Rev` ne
// monte pas. `replay.SchemaVersion` reste 78. La tete et la vue A rangees sont celles d avant, paquet
// par paquet, sur les 20 films (sonde `va_v1_research_test.go`). Le rang : `grammar-2026-10-04` et
// `grammar-2026-10-05` sont pris par l etape 2.7.a de la representation intermediaire, `.6` du
// 2026-10-03 par le lot LR ; le premier rang libre et sans trou est celui du 2026-10-06.
//
// ENTREE `grammar-2026-10-06.2` (2026-10-05, fusion de `feat/v75` dans l etape 2 de la
// representation intermediaire) : LES DEUX RANGS PRECEDENTS SONT REUNIS, ET LE CANAL DES MORTS LIT
// LA TETE PAR [listeAnnoncee].
//
// Ce qui change, contre `grammar-2026-10-05` et `grammar-2026-10-06` :
//   - les deux branches sont reunies sans autre changement de grammaire ;
//   - [canalDesMorts] reconnait un paquet a evenements par [listeAnnoncee], le predicat de la
//     marche : une vue A lue jusqu a son terminateur est TERMINEE meme quand elle porte des
//     messages, et l ancien test (vue A arretee) les aurait comptes sans evenements et prives de la
//     recuperation de leur liste.
// ENTREE `grammar-2026-10-06.3` (2026-10-06, lot LR de la campagne de grammaire, vague 3,
// `campagne_grammaire_2026-10-01/LOT_LR.md`) : UN RECORD NEW DONT LE LECTEUR D ETAT DU JEU ECHOUE
// N EST PAS LU ; LE SECOND RANG DE LA FERMETURE NE MODIFIE PAS LE MONDE.
//
// Ce qui change, contre `grammar-2026-10-06.2` :
//   - `FUN_14080cfe8` echoue sur un compte MPP superieur a quatre (`CMP ECX,0x4 ; JA` @14080d238) ;
//     les lecteurs d etat qui le lisent rendent alors 0, sauf `FUN_1408efb58` (`ti=41`) quand le
//     drapeau 2 est pose et l index absent ; `FUN_1408f1aa4` ne lit pas le corps d un record NEW
//     dont l etat echoue. [TraverseEntity] arrete un tel record a la fin de son etat
//     ([EntityTrace.EtatIllisible]) et le juge le contredit ([InvariantEtatDeCreationIllisible]) ;
//     aucun bit lu ne change (`etat_de_creation.go`) ;
//   - la marche qui part du second rang de [debutParFermetureRangee] ne lie aucun NEW et ne delie
//     aucun DEL (`debut_non_prouve.go`).
//
// Mesure sur 20 films contre `8dfadd07e`, sous le decoupage MPP que chaque film declare
// (`-mpp-declare`) : carte v2 397 824 -> 399 135 paquets sains (+1 311) ; les 12 films en baisse
// ne perdent que des listes ouvertes sur un NEW que le jeu ne lit pas. Le rang `.6` du 2026-10-03,
// que portait la tete du lot avant sa reprise, n a jamais ete fusionne.
//
// ENTREE `grammar-2026-10-06.4` (2026-10-06, lot VA de la campagne de grammaire, etapes V2 et V3,
// fusion de `feat/v75` a `fed1efed2` puis `b033d30f0`, corrections de la revue et decisions du
// pilote du 2026-10-06) : LA FIN DE LA VUE A FIXE LE DEBUT DE LA VUE B ; LA VARIANTE DE PARTIE DU
// FILM DECIDE LES GENRES 85 ET 116.
//
// Le lot portait, sur sa branche seule, trois rangs jamais fusionnes (`.2` et `.3` de l etape V2, `.4`
// de l etape V3) ; `.2` et `.3` etaient deja pris par `feat/v75` avec d autres empreintes. Ils sont
// reunis ici en un seul rang, le premier libre apres ceux de `feat/v75`.
//
// Ce qui change, contre `grammar-2026-10-06.3` :
//   - la fin E de la vue A lue jusqu a son terminateur est le debut de la vue B ([debutParLaVueA],
//     [lecture.DebutParVueA]) : `FUN_142f2c3b0` ecrit vue A, un bit 0, vue B, bout a bout, et
//     `FUN_142987460` commence la vue B au bit qui suit ; decisions de l utilisateur du 2026-10-04 :
//     film a table des genres EGALE a celle du jeu, E toujours, sans reprise a la signature quand la
//     marche depuis E bute ; film a table PREFIXE, E seulement si la marche depuis E ferme le paquet
//     sans regle de l ecrivain contredite ; vue A lue en partie : jamais. La table EGALE ne vaut que
//     sous la version majeure que le jeu joue (`FUN_1428e219c` : `*film == 0x29`) : un film de table
//     egale sous une autre majeure (HI_1_12_0, 0x28) suit la regle PREFIXE ([classeSousLaMajeure],
//     decision du pilote du 2026-10-06) ;
//   - la suite des genres rangee dit ou sa numerotation devient presumee
//     (`lecture.VueA.PremierPresume`, [premierGenrePresume]) ; aucun bit lu ne change ;
//   - la regle sert la cuisson ([debutDeLaVueBDeCuisson]), dont le canal des morts recoit les
//     records, et la marche de `killsource` ([DebutDeLaVueB], [VueADuFilmSousCarte]) ; le canal des
//     morts ne localise lui-meme ([debutRecupere]) que la liste que la cuisson n a pas localisee ;
//   - le localisateur exige que sa position suive un bit nul, le terminateur de la vue A
//     ([precedeDuTerminateur], seule implantation du test) ; les candidats NEW de tete n y sont pas
//     soumis ;
//   - le corps de `chunk_00` est lu jusqu a la table des joueurs (`FUN_1407ee138`, ses deux messages
//     Bond CompactBinary v2), et sa variante de partie rend m_gameEngineType, killcamEnabled et
//     playOfTheGameEnabled ([profile.VarianteDePartie], `film_variante_de_partie.go`) ; un champ du
//     chemin d un autre type que celui lu, ou un champ que la lecture ne sait pas enjamber devant
//     un champ attendu, arrete la lecture (rien n est lu) ;
//   - teleport_effects (116, `FUN_142ef93e0`) se lit quand le type de moteur du film n est pas 1 ;
//   - PlayerKilledEvent (85, `FUN_14104bd08`) se lit, partie fixe seule, quand les drapeaux du film
//     rendent fausse la garde de sa queue quels que soient les reglages d execution qu il ne porte pas.
//
// MESURES : `campagne_grammaire_2026-10-01/LOT_VA_V2.md` (etape V2, contre `87cdfa761`),
// `LOT_VA_V3.md` (etape V3, contre `8c83e2d3a` ; §15 : fusion et corrections de la revue, contre
// `fed1efed2` ; §16 : decisions du pilote, contre `b033d30f0`).
//
// ENTREE `grammar-2026-10-06.5` (2026-10-06, lot 2.7.b de la representation intermediaire,
// `.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE2_2026-10-03.md`) : LES HUIT LECTEURS DE COMPOSANTS
// BIPEDES LISENT LA MARCHE DES TRAMES, L ANCRAGE PASSE DERRIERE ELLE.
//
// Ce qui change, contre `grammar-2026-10-06.4` :
//   - [canalDesLecturesBipedes] recueille, pendant la marche des trames, les publications des onze
//     crochets des huit lecteurs (charges, impulsions, rangs, camouflage, grappin, arme portee,
//     deltas d inventaire, equipement), datees de la position du lecteur de la marche, et les
//     attribue au composant du record bipede delta qui les porte ; les lecteurs les rejouent
//     ([lecturesBipedes]) au lieu de marcher les records ancres ;
//   - l ancrage d en-tete bipede ne rend plus que les records d un slot que la marche n a pas lu
//     dans le paquet, hors de ce que la fermeture de la trame prouve (repli
//     `repli_ancrage_bipede_apres_la_marche`) : un debut de vue B LU (tete, fin de la vue A) prouve
//     tout le paquet, un debut LOCALISE seulement la liste lue depuis lui ;
//   - un corps mort n agit plus (le record du dead-state et ceux du meme corps jusqu au NEW) ; une
//     emission d arme portee qui repete la famille precedente de l emplacement, ou annonce un
//     emplacement vide sans occupant connu, est `Restated` ; la garde des generations vivantes
//     datees vaut pour les records de la marche ;
//   - la porte des essais eteint les douze crochets de canal ([Observation.neutraliserLesCrochetsDeCanal]) ;
//   - le canal des etats de mouvement ne compte plus un paquet a debut lu dans la vue A comme
//     localise ni comme ouvert par un NEW de tete.
//
// Preuve : gate de corpus (19 temoins) : aucun oracle ne bouge ; les lectures montent partout,
// aucune n est perdue contre la base hors des lachers d une arme inconnue ; un portage de bombe se
// ferme a l armement (`c75f33b8`), d ou la montee de `killcollector.PlacementRev`.
// `replay.SchemaVersion` reste 79 : la publication depuis les faits ne change pas, les faits si.
// Le rang : `.4` est pris par la vue A V2 et V3 de la campagne, fusionnee avant ce lot.
//
// ENTREE `grammar-2026-10-06.6` (2026-10-07, branche `feat/zones-etat-initial`) : L ETAT DES
// PROPRIETES RESEAU DE ti=13 SE LIT AUSSI DANS LES IMAGES-CLES.
//
// Ce qui change, contre `grammar-2026-10-06.5` :
//   - [ScanManagedProperties] joue, apres les trames delta, la phase des images-cles pour un canal
//     qui interprete la valeur scalaire de ti=13 (`i1`, [canalDesProprietesGerees],
//     `zone_state_scan_images_cles.go`) : il relit chaque occurrence a son etendue avec le
//     deserialiseur de production et rend [ManagedPropertyScan.KeyReads], records FERMES seuls,
//     avec les comptes `KeyRecords`, `KeyClosed`, `KeyBroken`, `KeyUnproven`, `KeyRefused` ;
//   - les lectures delta (`Reads`) et leurs comptes ne changent pas : aucun bit lu ne change sur la
//     voie delta.
//
// Une propriete n est emise en trame delta qu a son changement ; une base que la variante de
// Bastion donne a un camp au coup d envoi n apparaissait qu a sa premiere reprise. Le rejeu ouvre
// desormais le premier intervalle de proprietaire a la premiere image-cle qui le dit
// (`replay/zone_states_etat_initial.go`, `replay.SchemaVersion` 82). `killsource` et `objectives`
// n appellent pas ce balayage : leurs revisions restent constantes, leurs empreintes sont
// regenerees. Le rang : `.5` est pris par le lot 2.7.b de la representation intermediaire,
// fusionne avant ce lot.
//
// ENTREE `grammar-2026-10-07` (2026-10-07, lot des arrets de la vue B, `ti=43`) : LES COMPOSANTS
// `device-*` DU DISPOSITIF DE CARTE SE LISENT.
//
// Ce qui change, contre `grammar-2026-10-06.6` : le maillon [consumeComposantsDispositif]
// (`components_device_ti43.go`) lit `ti=43` `i18` a `i40` par les lecteurs du jeu (port du lot L2
// de la campagne, repris ; `i37` par [lireMinuteur142ba78dc], n = 10). Un record `ti=43` qui
// arretait la traversee se lit jusqu au bout ; `i31` au-dela de huit moniteurs arrete le record
// comme le jeu. Carte v2 des 20 films : aucun paquet sain perdu, aucun film en baisse.
