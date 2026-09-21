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
// `.29` a `.42` dans `rev_chronique_archive_2.go`. La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`) puis le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), puis le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place). C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du premier rang du 2026-09-18.
//
// ENTREE `grammar-2026-09-18` (2026-09-18, lot 5.1.1) : `.42` -> le premier rang du 18.
// LE POINT DE NAVIGATION (`ti=12`) EST LU DE `i1` AU MINUTEUR MANUEL.
//
// DOUZE LECTEURS NEUFS, RELEVES CHEZ LE DESERIALISEUR `+0x40` de chaque descripteur et recoupes
// par son serialiseur `+0x28` (`NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17`, § 5 a § 16) :
//
//	i1        `flags`, `FUN_141094130` : `R(8)`. C ETAIT LE BLOQUANT NOMME de l archetype.
//	i2..i6    les cinq `*-filter(s)`, qui partagent le bloc `FUN_140dbe400` — masque `R(4)`,
//	          drapeau `R(v ? 1 : 32)`, puis par filtre present un tag `R(4)` et sa charge (quinze
//	          tags, note § 3). `i2` y ajoute deux distances `R(16)` et deux par filtre ; `i3`/`i4`
//	          un `R(1)` et un par filtre ; les trois ferment sur `K` ordres de `v ? 3 : 2` bits.
//	i7/i8     `docking-order` `R(8)` (`FUN_142ed5050`), `docking-group-name` `R(32)`.
//	i9        `formatted-text`, `FUN_1410e7b90` : `R(8)` de compte, puis par entree un `R(32)`
//	          SUIVI DU SAC TEXTE DE `ti=11 i2` — reutilise, jamais recopie.
//	i10       `timers`, `FUN_1410d9040` : `2 x R(7)`, valeur - 1. MEME CHAMP que `ti=11 i0`, donc
//	          meme lecteur.
//	i11/i12   `manual-timer-initial-duration` / `-current-duration`, `FUN_142ed5194` et
//	          `FUN_142ed512c` : `R(17)` sur [-0,025 ; 6553,5752], soit un pas de 50 ms EXACT. CE
//	          SONT LES DEUX QUE LE LOT VISE — duree totale et temps restant du compte a rebours
//	          qu un objectif affiche. Le bassin du moteur, lui, ne le porte pas : `ti=11 i0` vaut
//	          « aucun minuteur » sur 446 records sur 446 (NOTE_3_7_REAPPARITION § 6 bis.3).
//
// UNE ENTREE DE TABLE, ET ELLE N EST PAS COSMETIQUE : `paramByComponent` pose `param_4 = 3` pour
// `i2` et `= 2` pour `i3`..`i6`. La valeur est LUE, pas ajustee — le slot `+0x10` du descripteur
// rend une constante (`MOV EAX,0x3 ; RET` a `0x14117e0e0`, `MOV EAX,0x2 ; RET` a `0x141179610`),
// et la meme chaine rend 3/3 sur trois composants `ti=35` dont la table porte deja la valeur de
// capture live. Avec le defaut 1, le drapeau ferait 32 bits au lieu d un et l ordre 2 au lieu de
// 3 : la marche se desynchroniserait des le premier filtre.
//
// AUCUNE ENTREE DE PROFIL. La seule largeur de RUNTIME du lot est celle de la reference d entite
// du domaine 0 (tags 6, 12, 13, 14) : `refDomWidth` et `readRecordID` sont appeles, pas recodes.
//
// LES CINQ FILTRES SONT `partiel`, PAS `porte` : un tag hors [0, 14] tombe chez le jeu sur
// `FUN_1411c8f80`, qui ne revient pas. Le lecteur rend `ported = false` — arret propre — plutot
// que de deviner une largeur. Un film sain n en porte pas.
//
// CE QUE LA MESURE DIT. Ratchet 0.A.3 : le bloquant de `ti=12` avance de `i1` a
// `i13 managed-navpoint-top-progress` sur les SEPT bobines. 0 ligne monte, 0 descend, aucun total
// ne bouge — et c etait prevu : la FERMETURE de `ti=12` demande encore `i13` a `i27`, quinze
// composants. Le bloquant qui avance de douze rangs EST la mesure du lot ; les largeurs sont
// tenues par `components_navpoint_test.go`, via `consumeByName` (donc le cablage avec).
//
// `facts.Rev` MONTE — mecaniquement, parce qu elle hache la VALEUR de cette constante, et non
// parce qu un fait change (`film/facts/` n est pas touche). Les lignes de kill en base deviennent
// candidates au backlog de redecodage, sur SIGNAL UTILISATEUR (D6). C EST L UNIQUE MONTEE DE
// `facts.Rev` DU LOT 5.1 : les volets suivants partagent ce rang.
//
// `SchemaVersion` reste 62 : aucun champ neuf n est publie (canal d observation seul).
//
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

// ENTREE `grammar-2026-09-20` (2026-09-20, lot 5.2b.1) : LE HORS-ROSTER DEGRADE, IL N ALERTE
// PLUS — ET C EST LE SEUL OCTET DE `grammar/` QUE CE LOT TOUCHE.
//
// `KillSourceHealth.OutOfRoster` sortait en ALERTE DURE ([KillSourceHealth.Alerts]), et
// `killsource.Result.LineByLinePublishable` refuse tout film en alerte : UN dead-state a l indice
// d un participant non compte eteignait donc la publication ligne par ligne du MATCH ENTIER.
// Mesure sur `b1ad85eb` (2026-09-20) : huit dead-states hors roster, `publishable = FALSE` sur
// les 77 lignes de `match_kill_events`, et la requete Q21b — qui filtre sur `publishable` — ne
// rendant plus rien, AUCUNE mort du kill-feed produit ne portait son arme.
//
// LE COMPTEUR MESURAIT DES LIGNES DEJA REFUSEES. `walkResult.selectCredible` ecarte tout indice
// `>= nPlayers` AVANT qu il ne devienne un candidat : une ligne comptee ici n atteint jamais la
// publication. L alerte punissait donc les AUTRES lignes — celles dont l indice est parfaitement
// dans le roster. Il passe en DEGRADATION ([KillSourceHealth.Degradations], nouvelle methode) :
// le verdict sort du domaine mesure, les lignes publient, et le message NOMME ce qui est refuse.
//
// AUCUN BIT LU NE CHANGE : `killhealth.go` ne lit pas un octet de film, il juge des compteurs.
// L empreinte monte parce qu elle hache les OCTETS de la couche (c est ecrit dans son en-tete).
//
// LE CONTROLE POSITIF DE DOMAINE RESTE ENTIER : le BTB `4f77afc1` sort toujours, et
// `TestKillSourceHealthRatioNInclutPasLeHorsRoster` verifie desormais ses trois criteres UN PAR
// UN (inexpliques 26.0 % > 18.0 %, couverture 76.5 % < 100 %, degradation nommee) — il ne peut
// plus tenir par le seul hors-roster.
//
// `facts.Rev` MONTE (elle hache cette valeur, et le lot change aussi `facts/killsource/`) :
// `killsource-2026-09-20`. `SchemaVersion` NE MONTE PAS — aucun champ n est ajoute au document.

// ENTREE `grammar-2026-09-20.2` (2026-09-20, lot 5.4.1) : L ANGLE DE ROULIS D `i2` N EST PLUS
// JETE — LA MOITIE MANQUANTE DE L AVANT DU CHASSIS.
//
// AUCUN BIT N EST LU AUTREMENT : `decodeObjectForwardAndUp` et le chemin « config » du mode 1
// consomment EXACTEMENT les memes largeurs qu avant (R(1)[+R(19)]+R(8) et R(1)[+R(30)]+R(30)).
// Ce qui change est ce qui est RENDU : le scalaire de queue, que le depot sautait, est l ANGLE
// que `FUN_1406d8678` combine a la direction pour construire le SECOND vecteur du composant.
// La direction ecrite est le vecteur HAUT (negatif mesure du lot 5.2b.2, |z| median 0,96 a
// 0,98) ; l AVANT est la perpendiculaire reconstruite. Cf. `orientation_frame.go`.
//
// POURQUOI LA REVISION MONTE ALORS QUE LA GRAMMAIRE D OCTETS EST INCHANGEE : la SORTIE de la
// couche change. `HasAim` / `AimRaw` etaient muets sur le chemin du mode 1 (la direction de
// 30 bits etait sautee) ; ils sont desormais poses, et le codec des faits les porte. Sur un film
// dont `i2` prend le mode 1 — le chemin DOMINANT des builds recents — les faits changent donc
// d octets. `AimVector` suit la largeur du mode au lieu de 19 bits en dur, sans quoi une
// direction de 30 bits se decoderait en vecteur arbitraire.
//
// `SchemaVersion` NE MONTE PAS : aucun champ neuf au document — `vehicles[].samples[].h` garde sa
// forme, seule sa SOURCE change.
//
// LA PREUVE A TENU, ET LE PORT EST POSE DANS CE MEME RANG (lot 5.4.3, 2026-09-21). Deux films,
// oracle du deplacement et temoin par permutation : `4f77afc1` mode 1, 35 350 echantillons,
// mediane **11,0 deg** contre un temoin a **88,8** ; `a349fea8` mode 1, mediane 23,4 (7,7 sur la
// population qui AVANCE), temoin 81,5. Le mode 0 est REFUTE (`a349fea8` : mediane 95,5 contre un
// temoin a 94,5, indiscernable — et sa direction lue n y est meme pas verticale, |z| median 0,585
// contre 0,979 pour le mode 1). Le cap publie sort donc du film sur le mode 1 et sur lui seul ;
// partout ailleurs la velocite reste la source, en repli nomme et compte.

// ENTREE `grammar-2026-09-21` (2026-09-21, lot 5.3.3-a) : `SimStateComplet` NE SE POSE PLUS A LA
// MAIN — IL SUIT LA CARTE DU MATCH.
//
// LA BASCULE EXISTAIT DEPUIS LE LOT R7-b (2026-08-17) AVEC SON CRITERE ECRIT : « que le chemin
// absolu d i0 tire ses trois largeurs de la CARTE du match ». Le critere etait TENU depuis les
// catalogues de cartes ; personne ne l avait relie au drapeau, et `i60 simulation-state` restait
// donc declare NON porte en production alors que sa queue est decodee
// (`consumeSimStateHandleTail`). Consequence mesuree : `i60` desynchronisait la traversee du
// bipede juste avant `i61-63`, et les composants d ETAT DE MOUVEMENT qui vivent derriere lui
// (`i29` accroupi, `i62` glissade) etaient perdus dans la moitie des records ou ils sont ecrits.
//
// CE QUI CHANGE, EN UNE LIGNE : la grammaire d un profil qui PORTE les largeurs de la carte
// declare `i60` complet ([grammaireSousCarte], `profil_balayage.go`). Deux portes, et deux
// seulement — le constructeur sous catalogue (`NewFilmContextForMap`) et le geste qui installe
// les largeurs sur un profil deja construit (`PoserLargeursObjetDuMondeDepuisDecoupage`, la
// porte de `killsource.ProfilDeDepartPourCarte` et de l installateur de `replay`,
// `replay/world_object_precision.go`). La
// seconde est obligatoire : `replay.poserProfilPuisCarte` remplace le profil ENTIER par celui
// que `killsource` a calibre, et une bascule posee a la seule construction y serait effacee sans
// un mot.
//
// LE DEFAUT GLOBAL RESTE FAUX, ET C EST LE CRITERE QUI L EXIGE : `NewFilmContext`
// (auto-detecte, SANS carte) sert les enveloppes `ScanFilm*(dir)` et les instruments, ou les
// largeurs d axe ne viennent pas de la carte. Garde-rail : `simstate_carte_test.go`.
//
// MESURE (film `bfecd02b`, carte `snowbound`, `DecodeFrameRecords`) : fautif `i60` 38 -> **0** ·
// records `ti=35` 11 150 -> **11 228** · `i29` lu 7 -> **14** · `i62` lu 6 -> **14** · desyncs
// reelles 3 130 -> **3 087** · trames saines 40,5 % -> **40,6 %** · `i21` 64,3 % -> 64,2 %.
//
// CE QUE LA CUISSON DE PRODUCTION EN VOIT : RIEN, ET C EST MESURE. A/B par `replay-build` sur
// deux films (`000d5950` et `bcb6d393`, carte `cliffhanger`, cache de faits vide a chaque passe),
// bascule levee puis abaissee : l artefact est BIT A BIT IDENTIQUE des deux cotes
// (`bcb2a510…` et `af57c5ea…`). Le `replay-equiv` du meme film le confirme etape par etape : la
// SEULE etape que la bascule deplace est le digest de `killsource`, et il porte la VALEUR du
// profil calibre (`Result.ProfilCalibre`, ou la bascule vit desormais) — compte inchange, octets
// du kill-feed inchanges. Ce que la bascule ouvre est la LECTURE DE LA TRAME, ou `i60` fermait la
// traversee du bipede avant `i61-63` ; les calques publies aujourd hui ne consomment pas ce qui
// est derriere.
//
// `facts.Rev` MONTE quand meme (elle hache la VALEUR de cette revision, mecaniquement) :
// `killsource-2026-09-21`. Le backlog de redecodage reste un geste de production sur signal
// utilisateur (D6), et cette entree-ci dit ce qu il rapporterait : rien sur le kill-feed.
// `SchemaVersion` NE MONTE PAS : aucun champ neuf, et aucun octet cuit ne change.

// ENTREE `grammar-2026-09-21.2` (2026-09-21, lot 5.3.6) : UN BALAYAGE NEUF — LES ETATS DE
// MOUVEMENT SORTENT DE LA COUCHE.
//
// AUCUN BIT N EST LU AUTREMENT, mais la couche RESPIRE une valeur de plus :
// `ScanMovementStates` (`movement_states.go`) rend les TRANSITIONS d etat du Spartan — accroupi
// `i29`, glissade `i62`, action de mobilite `i54` — par vie et par instant. Le document les
// publie en intervalles (`stances[]`, schema 65), donc la SORTIE de la couche change et la
// revision monte par valeur, pas seulement par empreinte.
//
// LA MARCHE EST CELLE DU FRAME-PROCESSEUR, ET C EST UNE DECISION MESUREE : la marche par
// chercheur d ancres des autres balayages de capacite annonce `i29` ZERO fois sur 162 444
// records de `bfecd02b`. `DecodeFrameViews` sur TROIS vues (ce que `FUN_142987460` deroule), les
// paquets a liste d evenements localises par `marchLocateStrict`, en rend 97 447 records `ti=35`
// dont 3 desynchronises et 7 941 lectures retenues.
//
// `i54` GAGNE UNE SECONDE PORTE DE PUBLICATION : son hook historique ne porte pas le slot, et un
// intervalle par VIE l exige. Les deux publient les MEMES deux drapeaux.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.2`.
// `replay.SchemaVersion` MONTE a 65 — le document porte un calque neuf —, et le codec des faits
// passe a `REPLAYINPUTS24` pour que les lectures voyagent avec les fixtures d entrees.

// ENTREE `grammar-2026-09-21.3` (2026-09-21, lot 5.7) : `ti=35 i55` N ETAIT PAS UNE LARGEUR DE
// DEUX BITS — SES QUATRE CHARGES SONT PORTEES.
//
// CE QUI A CHANGE, ET POURQUOI C EST UNE CORRECTION ET NON UN AJOUT. `consumeBipedPosturePhysics`
// lisait `R(2)` et s arretait, sur la foi d une glose qui disait de son repartiteur
// `FUN_141fd997c` « resolution d etat, 0 bit lu ». L ecrivain, relu a l octet, dit l inverse :
// `FUN_141fd997c` est LE REPARTITEUR D UNE UNION DISCRIMINEE — il pose un octet de genre
// (`dst+0x2c` = 1, 2, 3 pour les tags 1, 2, 3 ; le tag 0 prend une quatrieme voie) et appelle un
// lecteur de charge DIFFERENT par tag, de quinze a plus de cent bits. C est la decouverte D1 du
// lot 5.3, dont le cout n etait pas une desync mais une TRONCATURE : `i55` est le 56e des 64
// composants du bipede, donc les bits non lus decalent `i56` a `i63` et la boucle de la vue
// s arrete au record suivant.
//
// LES LARGEURS VIENNENT DE L ECRIVAIN, PAS D UN ESSAI. Les feuilles etaient toutes portees
// ailleurs dans la couche sauf `FUN_14080bd28` (un handle court `R(15)`, masque `& 0x7fff`) ; la
// largeur de `FUN_14076dc04` est lue au DESASSEMBLAGE de ses trois sites d appel
// (`142f263e5`, `142f2658b`, `1431c357d` : `R9D = 0x13`), jamais devinee.
//
// MESURE, `bfecd02b`, marche du jeu a trois vues, largeurs de carte installees. L ORACLE DE
// CONTENU TIENT et c est lui qui autorise le commit : etalon `i0` 85,5 % (inchange) · `i1`
// 77,5 % (inchange) · `i21` 65,2 -> 65,3 % · `i25` 97,0 -> 97,1 %. Records `ti=35` 97 447 ->
// 97 345 (-0,10 %), desyncs 3 -> 6 (`i59` 5, `i57` 1). LE COMPTE NE MONTE PAS, et c est dit :
// les essais d alignement de l inference de chaine (`deltaBodyTrial`) dependent de la largeur
// d `i55`, donc la corriger redistribue quelques alignements gagnants. L ECART EST DE UN POUR
// MILLE ET L ETALON NE BOUGE PAS : la correction est neutre en couverture, et juste en
// grammaire.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.3`.
// `replay.SchemaVersion` NE MONTE PAS : aucun champ neuf. Mais LE CONTENU CUIT CHANGE — les
// records du bipede ne ferment plus aux memes bits —, donc les fixtures de contrat et les
// goldens d assemblage se refigent, et `replay-equiv` deplace les etapes qui dependent du
// decodage.

// ENTREE `grammar-2026-09-21.4` (2026-09-21, lot 5.7.4) : LA PORTE DES ETATS DE MOUVEMENT
// S INSCRIT DANS LA NEUTRALISATION DES LECTURES SPECULATIVES — ELLE PUBLIAIT LES ESSAIS.
//
// AUCUN BIT N EST LU AUTREMENT : pas une largeur, pas un cadre, pas un ordre de composants. Ce
// qui change est CE QUI SORT de la couche, et il change beaucoup.
//
// LE DEFAUT. `traverseComponentLoop` est appelee par DEUX familles de chemins : la marche
// retenue, et les chemins SPECULATIFS qui essaient une lecture sur des bits qu ils
// abandonneront. Depuis le lot 2.2 ces derniers declarent leur speculation en eteignant les
// crochets de capture (`neutraliserCaptures`, `neutraliserCapturePosition` — « une lecture
// speculative n est pas une lecture »). La porte des etats de mouvement, posee au lot 5.3.4,
// NE S Y ETAIT JAMAIS INSCRITE. Et le LOCALISATEUR de paquet (`marchLocateStrict` et ses deux
// voisines) ne declarait rien du tout, alors qu il traverse l entite pour de vrai a chaque
// offset essaye.
//
// LA CORRECTION, A LA SOURCE ET EN UN SEUL POINT : `neutraliserEtatsDeMouvement` est la porte
// unique ; les deux neutralisations existantes l appellent, et les trois localisateurs
// l appellent directement. Aucun filtre aval, aucune logique par calque — c est la MARCHE qui
// dit « cette lecture est un essai », et elle seule le sait.
//
// CE QUE CELA VAUT, MESURE PAR LE BALAYAGE DE PRODUCTION LUI-MEME (`ScanMovementStates`, sous
// les largeurs d axe de la carte du match) :
//
//	                          | bfecd02b            | 4f77afc1
//	lectures retenues         | 7 463 ->     400    | 112 592 ->   1 430
//	ecartees (slot non lie)   | 7 940 ->      53    |  44 123 ->      68
//	doublons                  | 2 399 ->       0    |  47 670 ->       1
//	accroupi / glissade /     | 2 340 / 2 303 /     |  33 977 / 35 195 /
//	  action de mobilite      | 2 820               |  43 420
//	  ->                      |    52 /    47 / 301 |     191 /    222 / 1 017
//
// ET LE CONTROLE QUI TRANCHE : la porte rend desormais EXACTEMENT ce que les records retenus
// declarent. Confrontation lecture par lecture au compte des `Trace.Comps` des records rendus,
// sur les deux films : `i29` 76 = 76, `i55` 52 = 52, `i62` 56 = 56, `i54` 321 = 321, `i18`
// 117 = 117, `i1` 79 471 = 79 471 (`bfecd02b`) ; 236 / 287 / 230 / 1 033 / 515 / 220 844
// (`4f77afc1`). ZERO lecture fantome. Et la reconciliation ferme : 76 + 56 + 321 = 453 = 400
// retenues + 53 ecartees.
//
// CE QUI NE BOUGE PAS, ET C EST PROUVE : les positions et les vitesses publiees. Elles passent
// par `PosCaptureHook`, DEJA inscrit dans la neutralisation depuis le lot 2.2 — `replay-equiv`
// sur `bcb6d393` ne deplace que `movementStates`, `movementStates.stats` et `artifact`, les 54
// autres etapes sont identiques a l octet.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.4`.
// `replay.SchemaVersion` NE MONTE PAS : la FORME ne change pas, le CONTENU oui — `stances[]`
// perd les intervalles qu il tenait d essais jetes, et `coverage.stances` le dit par ses propres
// compteurs. C est `backfill-replay` qui re-cuit.

// ENTREE `grammar-2026-09-21.5` (2026-09-21, lot 5.9.4) : LE SAUT, DERIVE DE LA VITESSE
// VERTICALE ET PUBLIE SOUS UN GENRE QUI LE DIT.
//
// CE QUI CHANGE DANS LA COUCHE : `ScanMovementStates` capte desormais `EtatVitesse` (elle le
// jetait) et derive, apres la marche, les MONTEES de chaque vie par integration de la vitesse
// verticale TENUE d `i1`. Une montee fermee dont la hauteur integree tombe dans
// `types.SpartanJumpHeightM` +/- 10 % produit deux transitions `jumpDerived` — posee a l amorce,
// levee a la fin de la montee. Fichier neuf : `movement_states_jump.go`.
//
// LA HAUTEUR EST UN FAIT DE JEU, MESURE SUR DEUX FILMS (lot 5.7.5) : H = 0,85 m, montee 0,467 s
// sur `bfecd02b` (pic x 10,7 au-dessus de ses voisins) et 0,466 s sur `4f77afc1` (pic x 3,9).
// Tous les Spartans sautent la meme hauteur ; c est ce qui autorise la derivation, et c est
// pourquoi la constante vit dans `types/` avec sa mesure en commentaire, et non comme un reglage.
//
// DEUX REFUS ECRITS DANS LE CODE : un episode encore OUVERT a la fin de la marche n est pas
// publie (pas d instant de fin mesure, hauteur tronquee par le silence qui la termine) ; et un
// silence de replication de plus de 250 ms n est pas une vitesse tenue — `i1` ne voyage que sur
// changement, et integrer un tel silence FABRIQUERAIT des hauteurs.
//
// CE QUI N EST PAS FAIT, ET C EST UNE DECISION : le SPRINT. Sa chaine de donnees est complete
// (lot 5.9.1 — l etiquette d `i57` est l INDEX DE LA FENTE DE CAPACITE ACTIVE, et le bit 45 des
// drapeaux d unite est pose par `Sprint::Update` depuis une fraction rampee localement), mais la
// fente qui porte `'sasp'` n est pas nommee, et l utilisateur n a autorise aucune derive pour lui.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.5`.
// `replay.SchemaVersion` MONTE AUSSI (65 -> 66) : un genre neuf apparait dans `stances[].kind` et
// deux compteurs entrent dans `coverage.stances` — la FORME change.

// ENTREE `grammar-2026-09-21.6` (2026-09-21, lot 5.9.5) : LE SPRINT EST LU — `i57` PORTE
// L INDEX DE LA FENTE DE CAPACITE ACTIVE, ET L IMAGE NOMME LES TROIS FENTES.
//
// CE QUI CHANGE DANS LA COUCHE : `consumeBipedSpartanAbility` publie son etiquette par une porte
// qui PORTE LE SLOT (`EtatCapaciteActive`, septieme membre de l enumeration) — l ancienne porte
// `SpartanAbilityHook` ne le porte pas, et un intervalle PAR VIE l exige ; meme geste que la
// porte d `i54` a cote de `MobilityActionHook`. `ScanMovementStates` capte la nouvelle porte et
// publie le genre `sprint`. AUCUN BIT N EST LU AUTREMENT : le parcours de
// `consumeBipedSpartanAbility` est inchange, seule sa publication grandit.
//
// LE NOMMAGE VIENT DE L IMAGE. `FUN_1407e9ce4` aiguille sur le GROUPE DE TAG de la definition de
// capacite et appelle, pour chacun, un desenregistreur qui teste l index actif contre SA fente :
//
//	'saev' (0x73616576, esquive)  -> FUN_14319d0ac : fente `comp+0x1c`, index actif 0
//	'sasp' (0x73617370, SPRINT)   -> FUN_14319d1ec : fente `comp+0x20`, index actif 1
//	'sagh' (0x73616768, grappin)  -> FUN_14319d14c : fente `comp+0x24`, index actif 2
//
// Le flux ecrit `bloc+3 = R(2) - 1` (`FUN_142f268c4`), donc le brut `2` designe la fente 1. Le
// decalage vit en UN point, `sprintAbilitySlotRaw`.
//
// LE CONTROLE QUI VALIDE LA LECTURE DE L INDEX EST CELUI DU GRAPPIN : sur `4f77afc1`, la vitesse
// au sol pendant les intervalles de la fente 2 atteint 5,84 m/s au p90 contre 2,88 hors
// intervalle. Aucune autre capacite ne fait cela — c est la traction. Si la fente 2 est le
// grappin, l index est lu juste, donc la fente 1 est `'sasp'`.
//
// MESURE DE LA COUCHE (`bfecd02b`) : 625 lectures d `i57`, 416 sur un slot lie au bipede, 205 a
// la fente 1 et 209 a « aucune fente » — une pose pour une levee. `ScanMovementStates` publie
// 616 lectures `sprint` sur 65 vies (contre 52 `crouch`, 47 `slide`, 301 `mobility`). Sur
// `4f77afc1` : 1 088 lectures a la fente 1, 49 a la fente 2, 12 a la fente 0.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.6`.
// `replay.SchemaVersion` NE MONTE PAS UNE SECONDE FOIS : la v66 du meme lot porte DEJA le
// changement de forme de `stances[].kind`, et la chronique v66 nomme les deux genres.

// ENTREE `grammar-2026-09-21.7` (2026-09-21, lot 5.10.1) : L EMBARQUEMENT EST LU — `i10` PORTE
// LE PARENT ET SON SIEGE, ET LA MARCHE LES REND.
//
// CE QUI CHANGE DANS LA COUCHE : `object-parent-state` (`i10`) et `object-dissolver` (`i14`)
// entrent dans la COUCHE DE CAPTURE (`captureNames`) — leur valeur decodee voyage desormais dans
// `CompResult.Payload`, la ou elle etait jetee ; `WalkKeyframeRecords` garde les composants
// traverses ; et la marche des morts rend un second fait, les lectures d occupation
// (`ScanMarchFacts`, `vehicle_occupancy_march.go`). AUCUN BIT N EST LU AUTREMENT : les deux
// deserialiseurs sont scindes en `decode*` / `consume*`, et
// `TestCaptureConsumesSameBitsAsDispatch` echoue si les deux chemins divergeaient d un seul bit.
//
// L ECRIVAIN, RELU EN LECTURE SEULE (`FUN_140c1e4d0`, image base 140000000) : la branche
// ATTACHEE ecrit le handle du parent en +0x274 (`FUN_1406d3140`, categorie 1) ; la queue COMMUNE
// ecrit un entier de SIX bits en +0x3a0 derriere un bit de signe. La branche LIBRE efface les
// deux (0xffffffff, 0xffff) — ce sont les SENTINELLES qui designent les deux seuls champs
// capables de porter un embarquement.
//
// MESURE DE LA COUCHE (`4f77afc1`, carte `flood gulch` installee, oracle de contenu tenu : `i21`
// a 69,6 % sur 321 335 records `ti=35`) : 393 lectures d `i10` sur la bande bipede, 83 attachees
// sur un slot lie au bipede, dont 48 (57,8 %) nomment un slot `ti=40` avec la base 0x200 de la
// categorie — contre 4 (4,8 %) avec 0x300 et 1 (1,2 %) avec 0. Le champ de six bits vaut 0, 1
// ou 2 sur 42 de ces 43 lectures : conducteur, passager, tourelleur.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.7`.
// `replay.SchemaVersion` NE MONTE PAS : la FORME du document ne change pas — `vehicles[].rides[]`
// garde ses champs, et `seat` change de SOURCE, pas de type.
