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
// `.29` a `.38` dans `rev_chronique_archive_2.go`. La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`) puis le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil). C est le geste ordinaire que
// l en-tete des archives annonce, pas un incident. Ce qui suit est la suite VIVANTE, a partir
// du `.39`.
//
// ENTREE `grammar-2026-09-15.39` (2026-09-17, lot 2.6.1) : HERITAGE, MEME GRAMMAIRE, EMPREINTE
// RE-POINTEE. AUCUN OCTET N EST LU AUTREMENT.
//
// La constante passe de `GrammarRev` a [Rev] et son gate passe sur le mecanisme central
// (`film/revision`, lot 2.6.0) : c est la derniere des quatre couches a heriter, et l allowlist
// de `archlint/no_ad_hoc_source_fingerprint_test.go` en devient VIDE.
//
// CE QUI CHANGE EST LE PERIMETRE DE L EMPREINTE, ET LUI SEUL. Elle hachait CINQ RACINES EN
// OCTETS — `source/`, `profile/`, `grammar/`, `facts/killsource/`, `facts/objectives/`, 183
// fichiers ; elle hache desormais `grammar/` SEUL (143 fichiers) plus les VALEURS de
// `profile.Rev` et de `source.Rev` (V15 (12)), dans cet ordre. Les couches du dessous ont chacune
// leur revision depuis le volet facts + source du meme lot : c est `source.Rev` qui hache
// `source/`, `profile.Rev` qui hache `profile/`, `facts.Rev` qui hache `facts/`. Hacher leurs
// octets ICI aurait fait monter la grammaire pour une couche du DESSOUS d elle, et surtout
// rendait le meme diagnostic pour une borne de carte, un ordre de composants et un appariement
// de kill-feed.
//
// RIEN N EST RELACHE : le sens unique passe des octets aux VALEURS AMONT, qui le tiennent plus
// strictement — une montee de `source.Rev` ou de `profile.Rev` fait monter `Rev`, donc
// `facts.Rev`, donc le backlog killsource (D6, signal utilisateur), sans que personne ait a y
// penser.
//
// POURQUOI UN RANG NEUF PLUTOT QU UNE EMPREINTE RECOPIEE. La preuve d equivalence du lot 2.6.0
// disait que le cadre HERITE rend, sur les cinq racines, exactement l empreinte figee au `.38`
// (`7994ce19…`) : l heritage de l OUTILLAGE ne coute donc rien. Le PERIMETRE, lui, change — 143
// fichiers et deux valeurs amont au lieu de 183 fichiers — et l empreinte mesuree vaut
// `966e3f3e…`. Deux quantites differentes ne se figent pas sous la meme ligne : le rang monte,
// et la raison est ecrite ici.
//
// `SchemaVersion` reste 60 ; aucun match deja decode n est candidat au backlog par un changement
// de sortie — `facts.Rev` monte MECANIQUEMENT parce qu elle hache la valeur ci-dessus, et sa
// propre entree le dit.
//
// ENTREE `grammar-2026-09-15.40` (2026-09-17, lot 3.3.1) : L AMORCE DES LANCERS DEVIENT UNE
// DONNEE DE PROFIL, ET LE DECODAGE CHANGE. Le balayage comparait 24 bits sur TOUS les films : il
// lisait donc le bit de poids fort de l identifiant comme un bit d amorce sur les builds
// anterieurs a `HI_1_12_0`, d ou ZERO lancer publie sur cinq temoins du corpus. Il lit desormais
// la largeur, la VALEUR de l amorce et la position de l index auteur au profil (neuf clefs,
// `profile/grenade.go`), derive le motif du `ti` projectile RESOLU PAR NOM dans le registre du
// film, et ecarte par le sixieme bit d index les naissances de `managed-player`, comptees.
// `profile.Rev` monte avec elle ; `facts.Rev` derriere ; `SchemaVersion` ne bouge PAS.

// ENTREE `grammar-2026-09-15.41` (2026-09-17, lot 3.4.1) : LA MARCHE DES MORTS CALIBREE PAR LA
// CARTE. LE CHEMIN ABSOLU D i0 CHANGE DE LARGEUR, DE PLAGE ET DE REGLE D EMISSION.
//
// C EST UN CHANGEMENT DE DECODAGE, pas une reformulation : des bits DIFFERENTS sont lus aux
// memes offsets, et des positions qui etaient jetees sont publiees.
//
//	`position_capture.go`        `Lecteur.absoluteAxisW` et `absAxisW` DISPARAISSENT avec le
//	                             champ `Movement.AbsoluteAxisW` : la largeur UNIFORME de 14 bits
//	                             ecrasait les trois largeurs de la carte sur le chemin absolu du
//	                             bipede. `absAxisWFor` suit desormais l index de plage — table
//	                             DEFAUT du build (`22/22/22`) pour `idx == -1`, table PAR INDEX
//	                             de la carte pour `idx >= 0` — et `dequantWorldAxis` suit le
//	                             MEME index pour ses bornes : chez `FUN_14076e524` largeurs et
//	                             bornes sortent de la meme AABB, les dissocier etait le defaut.
//	`components_position_i0.go`  CORRECTIF D1 (3.4). Le commentaire « index 1 / no-index =
//	                             +/-20000 » etait FAUX et le filtre `if idx != 0 { return }`
//	                             avec lui : seul `index == -1` prend la boite du build. La
//	                             position n est emise que si l index designe la plage
//	                             CATALOGUEE (`Region`), nulle sur 78 cartes sur 79 et EGALE A 1
//	                             sur Live Fire, dont les deux films du corpus voyaient donc
//	                             leurs 59 376 positions valides jetees et le reste garde.
//	                             La largeur de l index vient de la CARTE (`DAT_144632be0`,
//	                             2 bits sur Live Fire) et non plus du descripteur de l appelant.
//	                             `consumeAbsolutePayload` et `consumePredictedAbsolute` perdent
//	                             leur parametre `pd`, devenu inutile.
//
// CE QUI CHANGE A L ECHELLE DU BIT, sur Cliffhanger : le chemin absolu du bipede lisait
// `5 + 3x14 + 2 = 49` bits, il en lit `5 + 13+13+14 + 2 = 47` — exactement la mesure Cheat
// Engine du dispatch (une seule valeur distincte, 100 % de 154 158 releves). Le compte se ferme
// ou il ne se fermait pas.
//
// `facts.Rev` MONTE (elle hache la valeur de celle-ci) : les lignes de kill deja en base
// deviennent candidates au backlog de redecodage — sur SIGNAL UTILISATEUR (D6), jamais
// automatiquement. `SchemaVersion` : cf. le volet 3.4.2.
//
// ENTREE `grammar-2026-09-15.42` (2026-09-17, lot 3.6.a — RANG PRIS A LA FUSION : la branche
// portait `.41`, deja occupe par 3.4.1 sur l integration) : `.41` -> `.42`. LE JOUEUR GERE
// (`ti=9`) N A PLUS AUCUN COMPOSANT SANS LECTEUR.
//
// DEUX LECTEURS NEUFS, RELEVES CHEZ L ECRIVAIN (`components_managed_player.go`) :
//
//	i4  `managed-player-forge-weather-effect-overrides-component`, ecrivain `FUN_142ed5bc8` :
//	    `R(32)` + `R(32)`, 64 bits INCONDITIONNELS. C etait le BLOQUANT NOMME des sept bobines.
//	i9  `managed-player-custom-input-prompt-widget`, ecrivain `FUN_141fcf160` : `R(1)` present,
//	    `R(2)` mode, puis le SAC TEXTE `FUN_14080b034` (`R(1)` + `R(32)` nom + `n = R(3)` +
//	    `n` corps `FUN_1407f0ebc` a largeurs litterales). Le depot rendait `ported = false` des
//	    que `n` depassait 0 — une desynchronisation propre sur une grammaire entierement
//	    decidable hors ligne.
//
// AUCUNE ENTREE DE PROFIL : toutes ces largeurs sont des litteraux d instruction. Le sac texte
// devient le SEUL lecteur du depot pour `FUN_14080b034` : les deux instruments qui en portaient
// leur propre copie l appellent desormais.
//
// LES DIX `case` DE `ti=9` SONT RASSEMBLES dans `consumeManagedPlayerComponent` — les trois
// maillons qui se les partageaient etaient AU PLAFOND du ratchet de longueur, et porter `i4` n y
// avait pas de place. La scission est NEUTRE PAR CONSTRUCTION : aucune etiquette `case` n est
// dupliquee dans la chaine (verifie sur pieces), donc l ordre des maillons ne decide de rien.
//
// CE QUE LA MESURE DIT. Fermeture d image-cle de `ti=9` : sur les sept bobines du ratchet 0.A.3,
// `0 / 1 717` (bloquant `i4` sur les SEPT) -> `1 716 / 1 717`, plus aucun bloquant nomme ; le
// 1 717e est une ancre fortuite de `111fa685` (`n1 = 2 154 823 696` contre `12` sur les autres).
// Sur les SIX FILMS DE RECHERCHE, `0 / 1 679` (100 % de desync) -> **`1 679 / 1 679`**, temoin de
// hasard a `0 / 1 679` : l archetype ferme a 100 % des qu on quitte le bruit du balayeur d ancres.
//
// `facts.Rev` : ce lot ne la fait pas monter de son propre chef (aucune source de `facts/`
// touchee, aucun fait publie change) ; elle vaut celle de 3.4.1, qui la monte pour sa raison.
// `SchemaVersion` reste 61 : mesure sur pieces, le document cuit de `084a804d` est identique
// base -> tete sur TOUS ses chemins sauf `/coverage/decoder/grammarRev`.
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
