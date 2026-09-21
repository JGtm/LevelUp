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

// ENTREE `grammar-2026-09-21.8` (2026-09-21, lot 5.11.0-a) : LE COMPTE DU SECOND TOUR D `i63`
// EST DANS LE FLUX, ET LA COUCHE LE CROYAIT EN RAM.
//
// CE QUI CHANGE DANS LA COUCHE, ET C EST UNE LARGEUR : `consumeBipedAction`
// (`i63 biped-action-component`, `FUN_142f027f4` -> `FUN_142f26a20`) lisait son bloc de tete de
// 3 x R(32) et le JETAIT, puis sautait son second tour sur une constante
// `bipedActionLoop2Count = 0`. Le commentaire qui la justifiait etait une DOC INVERSEE :
// « POPCOUNT of a 73-bit RAM bitmask on the component's own runtime state ... It cannot be
// recovered from the delta bits ».
//
// L ECRIVAIN, RELU EN LECTURE SEULE (image base 140000000) : `FUN_142f21b10(reader, _, param_3)`
// boucle `for (p = base; p != base+3; p++)` et ECRIT chaque `R(32)` dans `*param_3` ; le site
// d appel de tete de `FUN_142f26a20` passe `param_3 = param_1`, c est-a-dire la base d etat que
// `count2 = FUN_1409fe718(param_1, 0x49)` popcompte ensuite. Le masque N EST PAS un etat de RAM :
// c est le PREMIER CHAMP du composant. Le prologue le confirme — il sauve
// `etat[0xc..0x17] <- etat[0x0..0xb]` avant de laisser le flux ecraser les douze octets.
// FENETRE DU POPCOUNT, relue au bit : `((0x49 + 0x1f) >> 5) - 1 = 2` mots entiers, puis
// `p[2] & (0xffffffff >> (0x20 - (0x49 & 0x1f)))` = `p[2] & 0x1ff` — NEUF bits du troisieme mot.
// 32 + 32 + 9 = 73. Corps du tour : `FUN_1406cf008` = R(1), puis `FUN_14076e304` = R(2) si pose
// (les deux relus).
//
// MESURE, ET ELLE EST AMBIGUE — elle est ecrite telle quelle plutot que resumee. Masque de tete
// NUL sur la seule declaration d `i63` du film temoin `dad793c7` (un bipede, zero desync), sur 54
// des 73 de `bfecd02b`, et sur 37 des 299 de `4f77afc1` — ou les 262 autres forment une cloche
// centree sur 31 bits poses sur 73, c est-a-dire le profil de bits ALEATOIRES et non d un masque
// d actions. La nullite CORRELE avec l etalon du film (`bfecd02b` 77,4 % de masques nuls sur ses
// deltas pour `i0` a 85,5 % ; `4f77afc1` 12,7 % pour `i0` a 72,6 %) : les masques denses sont des
// `StartBit` deja decales EN AMONT, que `i63` — dernier et plus large composant — ABSORBAIT en
// silence. Effet net sur l oracle de contenu de `bfecd02b`, mesure A/B sur la meme base :
// records `ti=35` 97 345 -> **97 343** (perte de 2, 0,002 %), desyncs 6 -> 6, etalon `i0` 85,5 /
// `i1` 77,5 / `i21` 65,3 / `i25` 97,1 % inchange.
//
// DECISION ASSUMEE : LA GRAMMAIRE PRIME. L ecrivain est sans ambiguite, et garder une constante
// que la lecture refute serait un « compatibility guard forever ». La perte de 2 records est
// consignee au plan (case 5.11.0-a) ; elle ne vient pas de cette largeur mais de la derive amont
// que cette largeur cesse de masquer. Garde-rail :
// `components_biped_action_loop2_test.go` (fenetre de 73 bits, cout en bits du tour).
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.8`.
// `replay.SchemaVersion` NE MONTE PAS : la FORME du document ne change pas.

// ENTREE `grammar-2026-09-21.9` (2026-09-21, lot 5.11.7) : LES TROIS TABLES D ENTITES PAR VUE,
// ET LA MARCHE CESSE DE LIRE AU-DELA DE LA FIN DES PAQUETS.
//
// CE QUI CHANGE DANS LA COUCHE : `decodeInferLoop` — la boucle que `DecodeFrameViews` emprunte,
// donc toutes les marches du chantier — porte la GARDE DE TABLE DE VUE (`rejetDeVue`). Un delta
// dont le slot n appartient pas a la vue en cours n est plus un record a DEVINER : la vue s
// arrete sur son en-tete (`1 + idLow + 2` bits), sans lire un bit de corps.
//
// L ECRIVAIN, RELU EN LECTURE SEULE (image base 140000000). `FUN_142987460` appelle
// `FUN_1406cd128` UNE FOIS PAR VUE (`param_1 + 0x228` = trois objets de vue) et n ecrit RIEN
// apres elles. Dans la boucle, un DELTA n ouvre son corps que si la table DE CETTE VUE le
// reconnait :
//
//	lVar11 = (eid & 0x3fffffff) * 0xa0
//	if (vue[0x38][lVar11 + 8] == eid && vue[0x38][lVar11 + 2] == (short)type)
//	      FUN_141f86b58(...)   // le corps
//	else  uVar14 = 2 ; break   // LA VUE S ARRETE
//
// La table est un VECTEUR INDEXE PAR SLOT, agrandi a la demande (`FUN_1411b3c84`, capacite
// 0x1fff) et construit entree par entree (`FUN_1408f15c8`) : `(fin - debut) / 0xa0` en donne le
// cardinal. Une entree jamais posee porte donc `eid = 0`, et un slot inconnu de la vue est
// rejete au meme titre qu un slot d une autre vue.
//
// LA TRANSCRIPTION HORS LIGNE : [slotState.Vue] retient la vue ou la liaison a ete posee,
// [World.PoserVueCourante] annonce la vue marchee, [World.VuePossede] rend la garde. Les
// liaisons d IMAGE-CLE sont attribuees a la VUE 0 — c est la seule attribution que le film
// permette, et c est elle que la fermeture des paquets confirme (ci-dessous).
//
// MESURE, AVANT -> APRES, SUR DEUX FILMS :
//
//	dad793c7 paquets FERMES au curseur (reste 0..7)   0 %      -> 99,50 %  (5 068 a reste 0)
//	dad793c7 records FANTOMES `ti=6` slot 26          5 202    -> 0
//	dad793c7 records `ti=0` desynchronises            15       -> 0
//	dad793c7 records `ti=35`                          75       -> 75
//	bfecd02b records `ti=35`                          97 343   -> 114 458   (+17 115)
//	bfecd02b desyncs `ti=35`                          6        -> 5
//	bfecd02b etalon `i21`                             65,3 %   -> 65,2 %
//	bfecd02b paquets non localises                    1 924    -> 1 189
//	bcb6d393 `movementStates` (replay-equiv)          1 364    -> 1 489    (+125)
//
// Avant ce lot la marche consommait 57 bits DE PLUS que le paquet n en porte sur 95,7 % des
// paquets de `dad793c7`, et fabriquait un record `ti=6` a partir de zeros lus au-dela de la fin.
// Le gate est desormais mesurable : `DecodeFrameViewsCurseur` rend le curseur, et
// `TestMouvement5116Gate` exige un reste dans `[0 ; 7]` — le bourrage d octet, et rien d autre.
//
// CE QUE LA GARDE SUPERSEDE, ET C EST DIT : l inference d archetype sur slot non lie
// (lot 5.3.3-b) ne s applique plus par defaut, parce que l ecrivain ne devine pas. Le mecanisme
// reste joignable (`InferenceChaine`), et `frame_chain_infer_test.go` le met dans l etat ou il
// travaille (`withChain` abaisse `TablesParVue`).
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-21.9`, et
// cette fois la sortie des faits CHANGE REELLEMENT (`replay-equiv` deplace le digest de l etape
// `killsource`) : les lignes de kill deja en base deviennent candidates au backlog de
// redecodage, qui part sur SIGNAL UTILISATEUR (D6).
// `replay.SchemaVersion` NE MONTE PAS : la FORME du document ne change pas.

// ENTREE `grammar-2026-09-22` (2026-09-22, lot 5.13.1) : LA VUE D UNE LIAISON D IMAGE-CLE SE LIT
// AU LIEU DE S ATTRIBUER — LES DEUX BITS DE TETE SONT LE RANG DE LA VUE.
//
// CE QUE L ECRIVAIN DIT. La liste de reference qu un paquet d image-cle transporte est celle
// d UNE vue : son ecrivain `FUN_142f2e174` est le slot `+0x10` de la vtable de VUE
// `0x1436a87e0`, il ne parcourt que la table de SA vue, et il met les deux bits de tete de
// chaque identifiant a `vue + 8` (`142f2e2ec MOV ECX, dword ptr [RDI + 0x8]` puis
// `142f2e304 SHL ECX, 0x1e`, et de meme sur ses deux autres sites de genre). Et `vue + 8` est le
// RANG de la vue : c est le registraire `FUN_1409c9860(conteneur, rang, vue)` qui l ecrit,
// `*(int *)(param_3 + 1) = param_2`. Ce champ n est donc NI une generation NI un identifiant
// d entite — le journal RE du lot G (2026-08-27) le nommait `gen` sans avoir decompile
// l instruction.
//
// CE QUE LE DEPOT EN FAISAIT. `keyframe_world.go` lisait ce champ (`KeyframeRec.Gen`) et les
// deux binders d image-cle le JETAIENT : `BindWildcard(slot, ti)` posait `FullID = slot`,
// `GenAny`, et `Vue = 0` D OFFICE. [World.BindImageCle] le lit : le PREMIER rang rencontre est
// celui de la vue que la marche parcourt en premier ; un second rang irait en vue INCONNUE, qui
// ne rejette rien, au lieu d etre attribue d office au rang 0.
//
// MESURE. `TestImageCle513Vues` : UN SEUL rang par paquet d image-cle, et il vaut 1 sur les
// 5 paquets de `dad793c7` (123 a 186 records) comme sur les 60 de `bfecd02b` (424 a 482).
// `TestVues513EspaceDeNoms` : la vue parcourue en PREMIER rend 5 628 records sur `dad793c7`,
// tous de tag 1 (5 628 sur 5 628), et 157 250 sur 157 554 (99,81 %) sur `bfecd02b`.
// Les deux films temoins ne montrent donc qu un rang, et LA SORTIE NE BOUGE PAS :
// `dad793c7` 99,50 % de paquets fermes, 75 records `ti=35` ; `bfecd02b` 114 458 records `ti=35`,
// 5 desynchronises, etalon `i21` 65,2 % — a l identique. LA REVISION MONTE QUAND MEME parce que
// la lecture PEUT changer : sur un film dont l image-cle declarerait deux rangs, les liaisons du
// second passent en vue inconnue, et la marche y rend alors PLUS de records, pas moins.
//
// CE QUI EST REFUTE, ET C EST UNE MESURE. Les deux bits de tete d un identifiant de FLUX DELTA
// ne sont PAS le meme champ : `FUN_142f30610` ecrit l eid que la table de la vue porte
// (`*(uint *)(slot * 0xa0 + 8 + vue[0x38])`), pose par `FUN_1408f1730` a
// `*(byte *)(datum + 1) << 0x1e | slot` — un champ du DATUM, par entite. Confondre les deux dans
// la garde de vue (exiger l egalite des eids complets) coute 21 records `ti=35` sur `bfecd02b`
// (114 458 -> 114 437) : la garde compare donc le SLOT, et [World.VuePossede] le dit sur place.
//
// `facts.Rev` MONTE (elle hache la VALEUR de cette revision) : `killsource-2026-09-22`. La
// sortie des faits ne change pas sur les films temoins — seule la valeur hachee bouge.
// `replay.SchemaVersion` NE MONTE PAS : la FORME du document ne change pas.

// ENTREE `grammar-2026-09-22.2` (2026-09-22, lot 5.13.3) : `i57` EST PORTE EN ENTIER — L OCTET
// D ETAT RUNTIME QUI L EN EMPECHAIT N EN ETAIT PAS UN.
//
// La branche `tag == 3` d `i57 biped-spartan-ability` (`FUN_142f262d4`) etait la seule largeur
// indeterminee du composant, au motif que son corps est garde par `dst[2] & 1` et `dst[2] & 0x10`,
// « des octets d ETAT RUNTIME invisibles du flux ». Le desassemblage dit le contraire :
// `FUN_142f262d4` appelle `FUN_140f03dfc(dst)` en PREMIERE instruction — `142f262f2 MOV RDI, RCX`
// puis `142f262f5 CALL 140f03dfc`, RCX vaut encore `dst` — et cet initialiseur ecrit
// `*(undefined2 *)(param_1 + 2) = 0`. La porte `dst[2] & 1` est donc TOUJOURS fermee quand elle
// est testee, et la branche gardee par `dst[2] & 0x10` est inatteignable.
//
// Le corps se lit donc en entier : `R(1)` ; si 1 -> `R(6)` (`FUN_14297ea84`, largeur lue sur
// `if (0x40 - iVar1 < 6)`) ; puis `R(1)` ; si 1 -> la queue handle `FUN_14076e494`, le meme
// lecteur qu `i60`. `consumeBipedSpartanAbility` ne peut plus rendre `false`, et le dispatcheur
// rend `true` sans condition.
//
// C EST LA MEME LECON QU `i54` (`bloc[0x9d]` y est `flag1`, lu deux lignes plus haut par le meme
// deserialiseur) : quand une porte porte sur un champ de la structure de SORTIE, l initialiseur
// compte.
//
// MESURE, records RENDUS (instrument `TestMouvement511Partiels`) :
//
//	i57 non portees   bfecd02b : 1 -> 0 sur 651 declarations · 4f77afc1 : 44 -> 0 sur 2 531
//	desyncs `ti=35`   bfecd02b : 5 -> 4 · records `ti=35` 114 458 -> 114 458, etalon `i21` 65,2 %
//
// `facts.Rev` MONTE, et LA SORTIE DES FAITS CHANGE REELLEMENT : le golden de la mini-bobine
// deplace une ligne de kill de la voie `scan` a la voie `marche` (marche 6 -> 7, scan 3 -> 2),
// avec le MEME verdict et `DESACCORD` toujours a 0 — la marche va simplement plus loin.
// `replay.SchemaVersion` NE MONTE PAS : la FORME du document ne change pas.
