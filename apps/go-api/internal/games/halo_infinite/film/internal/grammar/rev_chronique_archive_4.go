package grammar

// rev_chronique_archive_4.go — LA CHRONIQUE DE [Rev], RANGS `grammar-2026-09-20` A `.2` ET
// `grammar-2026-09-21` A `.4`.
//
// # POURQUOI UNE QUATRIEME ARCHIVE (2026-09-22, lot 5.14.1)
//
// La chronique ne peut que grandir : un lot, un rang, une entree. `rev_chronique.go` etait a
// 495 lignes en recevant l entree de la garde de table de vue et celle d `i57` ; l entree des
// deux autres classes de vue le fait repasser les 500, et le ratchet de taille
// (`archlint/film_file_size_test.go`) le refuse — a juste titre. Les trois archives existantes
// sont pleines ou proches du seuil. LA ROTATION SE FAIT EN CHAINE, comme pour
// `.ai/thought_log.md` : c est le geste ordinaire annonce par l en-tete des archives, pas un
// incident.
//
// L ORDRE DE LECTURE EST CELUI DE `fichiersDeChroniqueGrammar` (`rev_test.go`) : archive,
// archive_2, archive_3, archive_4, puis la chronique vivante. Le gate exige que les rangs s y
// suivent sans trou. DEPLACEMENT PUR : pas un mot des cinq entrees ne change.

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
