package grammar

// rev_chronique_archive_3.go — LA CHRONIQUE DE [Rev], RANGS `grammar-2026-09-18.2` ET `.3`.
//
// # POURQUOI UNE TROISIEME ARCHIVE (2026-09-21, lot 5.11.7)
//
// La chronique ne peut que grandir : un lot, un rang, une entree. `rev_chronique.go` a repasse
// les 500 lignes en recevant l entree de la garde de table de vue, et `rev_chronique_archive_2.go`
// est EXACTEMENT a 500 : il ne peut plus rien recevoir. Le fichier suivant s appelle `_3`, comme
// l en-tete de la seconde archive l avait annonce. LA ROTATION SE FAIT EN CHAINE, comme pour
// `.ai/thought_log.md`.
//
// L ORDRE DE LECTURE EST CELUI DE `fichiersDeChroniqueGrammar` (`rev_test.go`) : archive,
// archive_2, archive_3, puis la chronique vivante. Le gate exige que les rangs s y suivent sans
// trou. DEPLACEMENT PUR : pas un mot des deux entrees ne change.

// ENTREE `grammar-2026-09-18.2` (2026-09-18, lot 5.1.7-a) : `grammar-2026-09-18` -> `.2` (le
// premier lot du jour s ecrit sans suffixe, les suivants a partir de `.2`).
// `param_4` NE SE DEVINE PLUS : IL SE LIT DANS LE REGISTRE DU FILM.
//
// `param_4` est la propriete que le descripteur d un composant rend a `FUN_14076cb60` avant que
// son deserialiseur ne tourne, et huit desers du depot en font une LARGEUR (`i2`, `i10`, `i19`,
// `i20`, `i23`, `i53`, `i59`, `i62`, les cinq filtres de `ti=12` et `flock-destination`). Il
// venait de DEUX endroits : une table par nom de composant (`paramByComponent`, vingt entrees
// mesurees a la capture live) et, pour tout ce que la table ne listait pas, LE BALAYAGE DE
// `killsource.calibrateRSP` — 0 a 5, la valeur qui maximisait la croissance des slots sur les
// records de BIPEDE, que `replaybuild` passait ensuite a la cuisson du rejeu.
//
// IL EST LE `level` DU REGISTRE, celui que l entree de composant porte en `entree + 0x100` et que
// `FUN_142e2c690` passe au deserialiseur — c est-a-dire `Archetype.Level(i)`, que le traverseur
// descendait DEJA jusqu a `consumeByName` sous le nom `level` sans que personne s en serve. Trois
// sources independantes le disent et concordent : les vingt entrees de la table valent toutes le
// `level` de leur ligne d `ecs_table.tsv` (capture live, 464 010 mesures sur `ti=35`) ; les cinq
// filtres de `ti=12` que le lot 5.1.1 a LUS au slot `+0x10` du descripteur (3 pour `i2`, 2 pour
// `i3..i6`) sont exactement leurs `level` ; et aucun nom de composant ne porte deux `level`
// differents sur les 48 lignes concernees — ce qu une propriete de descripteur doit avoir.
//
// CE QUI CHANGE DE COMPORTEMENT, ET OU. Trois desers n avaient AUCUNE entree et prenaient donc la
// valeur balayee : `i10 object-parent-state` (vrai `level` 3, et c est le composant qui precede
// immediatement le dead-state de `ti=40`), `i19 unit-actor-control` (2), `i20 unit-actor-state`
// (4). Sur `4f77afc1` et `a349fea8` le balayage retenait 4, qui se comporte comme 3 / 2 / 4 pour
// les seuls tests que ces desers font (`< 2`, `> 1`, `> 2`, `>= 4`) : la faute etait LATENTE, et
// un film dont le balayage aurait retenu 0 ou 1 aurait lu les trois a la mauvaise largeur.
//
// CE QUI DISPARAIT. `paramForComponent`, `Lecteur.recordStateParam`, les champs `ParamEtat` /
// `ParamEtatImpose` du profil de balayage et leur poseur, `FilmContext.PoserParamEtat`,
// `killsource.calibrateRSP` avec `monotonicScore`, `RSP`, `RSPRatio`, `rspMax` et `rspStride`, et
// le `PoserParamEtat(0)` de `ProfilDeDepart`. Le repli nomme `repli_parametre_etat_record_infere`
// est RETIRE du registre : sa cible etait ecrite d avance — « lot qui trouvera la source LUE de
// `param_4` (registre ECS par composant, ou table du build) » — et son critere — « la valeur
// vient d une lecture ; le balayage devient oracle comme celui des largeurs, ou disparait » — est
// tenu par la disparition.
//
// CE QUI RESTE DE LA TABLE : un RATCHET. Elle garde un seul appelant, `offline_aim.go`, qui
// compose sa grammaire sans registre donc sans `Archetype.Level` ;
// `TestParamByComponentEgaleLeNiveauDuRegistre` confronte chaque entree au `level` d
// `ecs_table.tsv` et rougit aussi si un composant y porte deux niveaux — ce qui ferait tomber le
// raisonnement de ce lot.
// REMPLACE LE 2026-09-18 par `TestParam4TableEgaleLExecutable` et `TestParam4RegistreParBuild`
// (`param4_par_build_ratchet_test.go`) : le premier garde-rail confrontait UN SEUL registre et ne
// pouvait pas voir que `param_4` varie d un build a l autre. L entree ci-dessus reste ce qu elle
// etait le jour ou elle a ete ecrite.
//
// `facts.Rev` NE MONTE PAS. Elle vaut `killsource-2026-09-18` depuis le lot 5.1.1, qui est le
// rang de TOUT le lot 5.1 : ce volet le partage et RE-FIGE son golden. Aucune source de `facts/`
// n est touchee au sens des faits publies — `killsource/calibrate.go` et `decode.go` perdent une
// grandeur qu ils ne decidaient plus.
//
// `SchemaVersion` reste 62 : aucun champ neuf n est publie.
//
// ENTREE `grammar-2026-09-18.3` (2026-09-18, lot 5.1.7-b) : `.2` -> `.3`.
// L ETAT PAR DEFAUT DE `ti=40` EST LU, ET SA BOUCLE DE COMPOSANTS TOURNE ENFIN.
//
// `consumeKeyframeDefaultState` ne consomme que si l archetype est dans `defaultStateDeserByTI`.
// `ti=40` n y etait pas — par la regle de `default_state_arch.go` (« un archetype dont UNE largeur
// de feuille n est pas etablie statiquement n est PAS inscrit »), sa feuille 4 portant la mention
// « config-dependante ». Le jeu ecrivait donc 79 bits au minimum (`FUN_1410A5A74`), le lecteur en
// consommait ZERO, le `R(32) n2` se lisait 79 bits trop tot et rendait une valeur `<= 0` :
// `consumeFullStateDefaultBlock` rendait faux et LA BOUCLE DE COMPOSANTS N ETAIT JAMAIS LANCEE.
// C est ce que le golden 0.A.3 disait sans qu on le lise : `ti=40` a 0 ferme sur 777, colonne
// « bloquant » VIDE sur un archetype de 48 composants dont 16 non portes.
//
// LA MENTION ETAIT PERIMEE. Les deux globaux qu elle nommait — l index `DAT_144632be0`
// (`FUN_14076e524`) et les trois largeurs per-axe `DAT_1445cc9e0` (`FUN_140cc5128`) — entrent par
// le CATALOGUE DE LA CARTE depuis le lot 3.4.1, et les deux fonctions de la feuille sont portees
// depuis le lot R7-b : `FUN_14076e494` par `consumeSimStateHandleTail`, `FUN_140c1e79c` par
// `consume140c1e79c`. La feuille se LIT, a la largeur de la carte du match, comme le chemin
// world-object. `vehicleMediaFrameBits` disparait avec le modele qu il portait.
//
// LA MESURE, ET SON TEMOIN NEGATIF (`4f77afc1`, 1 140 records `ti=40` d image-cle) : la porte
// `bVar14` vaut 1 sur **470 records (41,2 %)** — elle n est pas negligeable, et la question ne
// pouvait pas se trancher en la supposant nominale. Feuille LUE, les deux populations butent au
// MEME rang sans exception : `bVar14 == 0` 661/661 a `i30`, `bVar14 == 1` 470/470 a `i30`. Feuille
// modelisee ABSENTE, les 470 rendent `DesyncAt == -1` — la boucle ne tourne pas. C est l oracle
// qui etablit la feuille, et il est binaire.
//
// CE QUI CHANGE, ET CE QUI NE CHANGE PAS. Le bloquant de `ti=40` passe de « (aucun) » a
// `i30 vehicle-auto-turret-triggers-component` : la fermeture ne monte pas — elle ne le peut pas
// tant que les seize `vehicle-*` ne sont pas portes — mais le golden cesse de mentir sur cet
// archetype. Ratchet 0.A.3 : 0 ligne en baisse. **Le document publie ne bouge d AUCUN octet** :
// mesure sur `4f77afc1`, `recensees=256 publiees=97`, `finDatee=3` avant comme apres. Le calque
// des vehicules passe par des balayages ANCRES (`ScanWorldObjectKeyframes`,
// `ScanVehicleCreationsForBand`), pas par la marche d etat complet — l hypothese qui attribuait
// `97/256` a ce defaut est REFUTEE par la mesure, et la cause de `97/256` reste a instruire.
//
// `facts.Rev` suit par VALEUR (elle hache cette constante) et garde son rang
// `killsource-2026-09-18`, qui est celui de tout le lot 5.1 : golden RE-FIGE, pas monte.
// `SchemaVersion` reste 62 : aucun octet publie ne change.
