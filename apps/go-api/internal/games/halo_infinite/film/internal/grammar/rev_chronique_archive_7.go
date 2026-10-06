package grammar

// rev_chronique_archive_7.go - LA CHRONIQUE DE [Rev], RANGS `grammar-2026-09-24` A `grammar-2026-09-27.3`.
//
// # POURQUOI CETTE SEPTIEME ARCHIVE (2026-10-03, integration de la vague 1 de la campagne de grammaire)
//
// `rev_chronique.go` aurait passe 500 lignes avec l entree `grammar-2026-10-03` (lot L4a) et le
// ratchet de taille (`archlint/film_file_size_test.go`) refuse de grandir. Le rang
// `grammar-2026-09-24` est donc verse ici, mot pour mot, et la suite VIVANTE repart a
// `grammar-2026-09-27`. Geste ordinaire, annonce par l en-tete des six archives precedentes. Cette
// archive est dans `fichiersDeChroniqueGrammar` (`rev_test.go`), donc hors de l empreinte : y
// ecrire ne fait pas monter la couche. Le 2026-10-06 (lot VA de la campagne, fusion de `feat/v75`),
// les rangs `grammar-2026-09-27` a `.3` y ont ete verses a leur tour, mot pour mot, pour la meme
// raison ; la suite VIVANTE repart a `grammar-2026-10-02`.

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
//
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
