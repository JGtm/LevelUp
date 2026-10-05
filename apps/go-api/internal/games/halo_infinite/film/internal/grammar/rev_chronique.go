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
// `grammar-2026-09-24` dans `rev_chronique_archive_7.go`.
// La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`), le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place), puis le
// 2026-09-22 (lot 5.20.1, une CINQUIEME archive), puis le 2026-09-24 (lot M4b, une SIXIEME), puis
// le 2026-10-03 (integration de la vague 1 de la campagne de grammaire, une SEPTIEME).
// C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du `grammar-2026-09-27`.

// ENTREE `grammar-2026-09-27` (2026-09-27, lot J5.5 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25) :
// UNE SEULE MONTEE POUR TOUT LE JALON J5 (lots J5.1 a J5.4), dont les commits avaient refige le
// golden a revision constante en attendant celle-ci. Ce qui change est ce que la grammaire REND :
// des lectures de plus (les corps de generation >= 2) et des chaines rattachees a la bonne vie.
//
// J5.1 (neutre, dit pour memoire) : la cle de vie `types.EquipmentLifeKey` devient `types.LifeKey`
// (sans alias) et le handle se lit par un lecteur UNIQUE (`handle.go`, `LireHandle` /
// `LireHandleDelta`) ; aucun bit n est lu autrement.
//
// J5.2 (GB-1, DT-8) : LE FILTRE DE GENERATION VIVANTE (`generations_vivantes.go`). Les positions
// bipedes et les huit canaux delta n acceptaient qu une generation de handle egale a 1
// (`ScanFilmOptions.RequireTag1`, supprime, et trois copies en dur) : quand le pool de slots
// reboucle, les corps de generation 2 et 3 n avaient aucune position ni lecture delta. Une
// generation est desormais lue quand une creation de bipede ou un record `ti=35` d image-cle l a vue
// designer un corps ; un slot dont aucune generation n est connue retombe sur l ancien filtre, repli
// nomme `repli_generation_vivante_inconnue_tag1`. L etage du pont d identite (`pont_identite.go`)
// lit les creations AVANT les positions ; les vehicules sont lus sous toutes les generations.
// Mesure J5.0 (`.ai/V7.5/film_re/MESURE_GB1_2026-09-27.md`) : quatre films touches (`084a804d`,
// `1c4c63c2`, `a349fea8`, `4f77afc1`), les quinze autres temoins sans aucune vie de generation >= 2.
//
// J5.3 : l equipement, sa recuperation et l arme tenue sont CHAINES PAR VIE (`LifeKey`) et non plus
// par slot : un slot recycle ne prolonge plus la chaine du corps precedent.
//
// J5.4 GA1-3 : `decodeInferLoop` pose le slot de capture de CHAQUE record, NEW compris (D13 de la
// note 5.3 levee) : un NEW ne publie plus ses etats de mouvement sous le slot du record precedent.
//
// `killsource.Rev` MONTE derriere elle (`killsource-2026-09-27`), par la recette du sens unique : sa
// fermeture hache la VALEUR de cette constante. Aucune lecture qu il consomme n a change (il ne lit
// ni les positions bipedes, ni les canaux delta, ni les chaines d equipement, et n installe aucun
// crochet de capture) : aucun changement du kill-feed n est attendu (cf. sa chronique).
// `objectives.Rev` n est pas concerne : sa fermeture ne rencontre que `source`. `replay.SchemaVersion` monte (73 -> 74) et `killcollector.IsolationDecoderRev` aussi
// (les positions du collecteur passent par le meme etage du pont).
//
// ENTREE `grammar-2026-09-27.2` (2026-09-27, lot J6.3 du plan de suite de l audit du decodeur) :
// UN SEUL PORTAGE DE `FUN_14076e524`. Le rang `.2` distingue ce lot de J5, qui a monte
// `grammar-2026-09-27` en parallele ; rang fixe a la fusion des deux lots : `.2` suit J5.
//
// CE QUE LE RANG CHANGE (releve Ghidra du 2026-09-27, `lecteur_position.go`). Le lecteur de position
// quantifiee et ses deux enveloppes (`FUN_14076e494` : garde de pleine precision ; `FUN_14076e420` :
// bit precHigh) ont UN portage, parametre par l IMMEDIAT DE NIVEAU du site d appel. Tous les sites
// l appellent avec l immediat releve, que tient `lecteur_position_ratchet_test.go`. Lectures qui
// changent :
//
//	asset-transform `ti=44 i0`   niveau 0x1E : 26/26/26 et index sur `DAT_144632be0` (7/7/7 et 1 bit)
//	unit-actor-state (visee)     le 0x10 est un NIVEAU : garde, porte, index, trois axes (R(16) plat)
//	crew-order, tacmap-poiicon et -poiiconoffset, flock-destination, player-desired-respawn-location
//	                              plus de bit precHigh, axes a la ligne 0x10 (6 + niveau du registre)
//	tacmap-waypointstate, -areaofinterest, -displayasset, -cooptetherarea, spawn-filter-type,
//	selectable-zone-data         largeurs de la ligne 0x10 et garde (descripteur de traversee)
//	tacmap-waypointstate         + R(1) quand le niveau du registre depasse 1 (`FUN_140f04d88`)
//	i0 bipede, precHigh = 1       `FUN_141f85880` : 3 x 14 bits sur +/-100, puis R(2) (0 bit)
//	i0 absolu predit, cVar1 = 1   idem (0 bit)
//	i0 repli du delta predit      `FUN_14076e524` nu : un bit precHigh de moins
//	trame media de l etat par defaut du bipede   garde, porte, index, axes (branche inerte)
//	`DAT_144632be0`               lue du profil partout (GA2-3) — plus de 1 cable
//	etat par defaut `ti=13`       la charge de chaque variant apres l etiquette (GA2-4)
//	translocateur                 le meme lecteur, sous la garde ; oracle 18/18 inchange
//
// TROIS EXCEPTIONS DATEES gardent leur ancien lecteur (`lecteur_position_exceptions.go`, decision du
// superviseur) : flock-position `ti=21 i16`, world-object i0 (porte posee) et `ti=38 i18` — la
// lecture du jeu y fait baisser la fermeture des bobines.
//
// Mesure (bobines du depot) : fermeture d image-cle `ti=13` de 0 % a ~100 % sur les sept bobines,
// aucune autre ligne ne bouge ; carte de fermeture sans aucun compte `fermes` modifie, une ligne
// neuve (ks_000d5950 `ti=23` 0/1). Golden des familles de mini-bobine inchange.
//
// ENTREE `grammar-2026-09-27.3` (2026-09-27, lot J10.7 du plan de suite de l audit du decodeur) :
// UNE SEULE MONTEE POUR LE JALON J10 (determinisme et correctifs residuels), dont les lots J10.2 a
// J10.6 avaient refige le golden a revision constante en attendant celle-ci.
//
// J10.2 (GB-4, neutre, dit pour memoire) : `ScanFilmWeaponDamages` ne rejoue plus de monde (plus de
// `_, _ = DecodeFrameRecords`, plus de base d atterrissage calculee puis jetee) ; aucun bit n est lu
// autrement, aucune sortie ne change.
//
// J10.5 (GA1-4) : la coincidence de la suite croissante des datums d image-cle suit la regle de la
// grammaire (a slot egal, la generation la plus basse, puis le candidat le plus tot : l ordre de
// `kfCand.betterThan`) — la suite gardait la coincidence la plus TARDIVE ; `ambigus` compte des
// SLOTS (vus avec deux archetypes, ou ecartes en entier), plus des candidats.
//
// J10.6 (GA1-5) : la table des joueurs laisse le recul sur les vacants de tete decider a completude
// egale (`>=`) : le bourrage de queue ne le neutralise plus, et le rang des joueurs d un roster dont
// le slot 0 est vacant n est plus decale.
//
// J10.1 (GB-3, faiblesse 9, DT-9) : LES TRIS QUI DECIDENT D UNE SORTIE SONT TOTAUX. Dans la
// grammaire : le fil des morts (instant, xuid, gamertag) ; les emissions d equipement — chaine
// d une vie et fusion stricte/recuperee departagees par la position du record (`abilityEmission.Bit`,
// le bit du composant i0), liste publiee rangee par (instant, chunk, paquet, slot, generation,
// offset, bit) et non plus dans l ordre d iteration de la map des vies (GB-3) ; les series navpoint,
// les declarations de la table anticipee, les echantillons d i0, les positions et les degats de
// l appariement tir -> touche, les deux tris des tirs `weaponscan` (la deduplication garde le PREMIER
// d un amas) : ex aequo dans l ordre du film ; teleportations et bascules de lunette : (instant,
// slot). Une sortie ne change que sur des ex aequo que l ancien tri departageait au hasard.
//
// Hors de la grammaire, au meme lot : les calques du rejeu (J10.3 RB2-6 la tolerance de siege en
// `time.Duration`, J10.4 RB2-7 le lien prise -> arme au sol borne par `LowUS`, J10.1 RA2-5 le roster
// et dix-huit autres tris, les relais de bots de `replaybuild`) — `replay.SchemaVersion` monte
// (75 -> 76 : J8, fusionne avant, avait pris le 75) ; `killsource` (cinq tris et l ordre des
// candidats) et `objectives` (le pied de film ; `objectives-2026-09-27`, montee par J8) gardent
// leur revision, series jamais publiees : golden regenere a revision constante.
//
// A LA FUSION DE J10 DANS LA BRANCHE DE SUITE D AUDIT (2026-09-27, apres J8) : l empreinte de ce
// rang couvre aussi les comptes de repli de J8.7 (treize replis de la grammaire comptes par le
// rapport porte par le `FilmContext`, sans changer une lecture), que J8 avait figes a revision
// constante sous `.2`. La ligne `.2` du golden garde sa valeur de fusion J6.
//
// A REVISION CONSTANTE, LOT J6-bis (2026-09-28, serie jamais publiee) : les baisses que la
// regeneration des entrees figees attribuait a J6, remontees au paquet (marche des trames des huit
// builds, J6 contre son parent). Quatre sites gardent leur lecture d AVANT J6, en EXCEPTIONS
// DATEES au format de J6 (la lecture du jeu y fait baisser une fermeture) : flock-destination
// (`11de8353`, deux listes, 38 entrees de controle), tacmap-poiicon (`11de8353`, une liste),
// player-desired-respawn-location (`e5adf7b2`, une liste, 14 entrees) et l etat par defaut de
// ti=13 DANS UN RECORD NEW (`fb1a1a72` et `60ae07c4`, quatre listes, 9 entrees ;
// `default_state_ti13_neuf.go` — l image-cle garde la lecture du jeu). Plus aucune liste fermee avant
// J6 ne se perd sur les huit builds ; les fermetures que la lecture du jeu apportait a ces seuls
// sites (six listes de `60ae07c4`, une de `000d5950`, `111fa685` et `e5adf7b2`) attendent le
// critere de retrait.
//
// A REVISION CONSTANTE, LOT R3 (2026-09-29, serie jamais publiee) : les baisses que le G-corpus de
// J11.1 attribuait a J6 (C4) et a J6-bis (C5), remontees au paquet (marche des trames de douze films,
// parent de J6 contre la tete du plan). Cinq sites de plus gardent leur lecture d AVANT J6, en
// EXCEPTIONS DATEES au meme format : tacmap-displayasset (`51ebbc0f`, dix paquets, 61 entrees de
// controle), tacmap-areaofinterest (`51ebbc0f`, un paquet), tacmap-cooptetherarea (`c75f33b8`, une
// liste), crew-order (`084a804d`, une liste, 14 entrees) et la branche precHigh = 1 de l i0 absolu
// du bipede (`0797ce72` et `084a804d`, trois listes, 37 entrees). C5 n est pas un site de plus :
// l exception ti=13 de J6-bis rend a `51ebbc0f` des fermetures gagnees par J6, sans rien prendre a
// la reference. Plus aucun paquet ferme avant J6 ne se perd sur les douze films.
//
// A REVISION CONSTANTE, LOT R2 (2026-09-28, serie jamais publiee ; constat C2 du G-corpus J11.1) :
// le filtre de generation vivante de J5.2 est DATE pour le balayage des positions
// (`generations_vivantes.go`, [GenerationsVivantes.A]). Un en-tete delta dont le handle designe un
// corps AVANT le record de creation de ce corps n est la replication d aucun corps : il est refuse,
// sauf pour le premier corps de generation 1 d un slot (regle R-B1 du rejeu, inchangee). Mesure :
// 14, 6, 5 et 28 positions de production aberrantes retirees sur `084a804d`, `a349fea8`,
// `4f77afc1`, `1c4c63c2` (z de -937 m, borne de carte, saut de 3,3 s) ; les records reels que ces
// faux ancrages masquaient reviennent (+15 sur `a349fea8`). Films sans slot reboucle : aucun record
// concerne. Les huit canaux delta, l arme tenue et la recuperation d equipement gardent le filtre
// atemporel (hors du lot : decouverte au rapport R2).
//
// A REVISION CONSTANTE, LOT R2-bis (2026-09-29, serie jamais publiee ; decouverte 2 du lot R2) : la
// MEME garde datee ([GenerationsVivantes.A], par [FilmContext.GenerationsVivantesA]) s applique aux
// autres lecteurs de records delta bipedes — le marcheur des huit canaux (`walkDeltaBipedRecords` :
// rang et charges de capacite, impulsions, camouflage, grappin, arme tenue, inventaire, equipement de
// l unite), la recuperation d equipement et la visee seule. En-tetes anterieurs a la creation de
// leur corps refuses par le marcheur : 32, 33, 20 et 102 sur `084a804d`, `a349fea8`, `4f77afc1`,
// `1c4c63c2` (records rendus 27, 29, 18 et 91 de moins : le curseur libere retrouve des records
// reels) ; emissions retirees, entre autres : arme tenue 1/0/2/1, rang de capacite 1/0/0/5, visee
// seule 92/122/335/406. Positions inchangees ; films sans slot recycle (`bcb6d393`, `51ebbc0f`,
// `d9781168`, `e5adf7b2`) : toutes les sorties identiques a l octet. Garde-rail :
// `generations_vivantes_datees_ratchet_test.go`.
//
// A REVISION CONSTANTE, LOT R3-bis (2026-09-30, serie jamais publiee) : la baisse de lecture de
// `4f77afc1` sous la reference J4.0.5, remontee au paquet (carte de fermeture paquet par paquet de
// vingt films, la reference contre la tete du plan). Neuf paquets fermes a la reference et perdus a
// la tete, tous introduits par J6 et non par R3 (les cinq exceptions de R3 n y retirent que des
// fermetures gagnees par J6). Deux sites de plus gardent leur lecture d AVANT J6, en EXCEPTIONS
// DATEES au meme format : les deux vecteurs de unit-actor-state (`FUN_14058c058`, seize bits plats ;
// `4f77afc1` 38:410, 54:316, 54:1140 et la chaine 12:1118..12:1128, 133 entrees de controle) et
// tacmap-waypointstate (`ti=34 i7`, position aux largeurs de la traversee, sans le R(1) de
// `param_4 > 1` ; `d9781168` 34:336, 4 entrees). Prix : onze paquets que J6 avait gagnes, jamais
// fermes a la reference (`084a804d` six, `e5adf7b2` deux, `111fa685`, `4f77afc1`, `d9781168` un
// chacun). Plus aucun paquet ferme a la reference ne se perd sur les vingt films.
//
// ENTREE `grammar-2026-10-02` (2026-10-02, lot L0 de la campagne de grammaire,
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA FERMETURE D UN PAQUET SUIT LES REGLES DE
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
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : `ti=3 low-frequency` EST PORTE, ET `high-frequency`
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
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA FIN DU MOTEUR DE PARTIE, LUE DANS LE JEU.
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
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LES COMPOSANTS PROPRES AU VEHICULE (`ti=40`, i30 a
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
// et corrections de sa revue adverse, `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : `low-frequency`
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
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA SIGNATURE DU LOCALISATEUR SE LIT SUR TOUT OBJET DE
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
// `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LA PREUVE PAR CHAINE REFUSE UN RECORD DONT LE
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
// adverse, `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`) : LU ET LT SUR `feat/v75`, LS RETIRE ; LA
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
