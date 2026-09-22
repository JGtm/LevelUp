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
// `rev_chronique_archive_5.go`. La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes, comme `.ai/thought_log.md` ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`), le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place), puis le
// 2026-09-22 (lot 5.20.1, une CINQUIEME archive). C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du `grammar-2026-09-22.7`.
//
// ENTREE `grammar-2026-09-22.7` (2026-09-22, lot 5.18.2) : LE CONTROLE DE CORRUPTION PAR
// COMPOSANT EST LU DANS LE FILM, ET IL N A PLUS DE DEFAUT MUET.
//
// LE MAILLON, DE L ECRIVAIN JUSQU AU LECTEUR DE BITS (Ghidra, `HaloInfinite.exe`, base
// `0x140000000`, lecture seule) :
//
//	W(1)   `FUN_14299b198` @14299b25b  `FUN_1406d49c4(writer, byte[film+0xCB45C])`
//	R(1)   `FUN_14299ab50` @14299ac28  `FUN_1406cf008(lecteur)` -> `film+0xCB45C`
//	copie  `FUN_1428e219c` @1428e2239  `*(char *)(singleton + 0x1AE) = film[0xCB45C]`, sous la
//	                                   garde `*film == 0x29` (la version MAJEURE du film)
//	usage  `FUN_14076cea8()`           rend `DAT_144c23326` (= `DAT_144c23178 + 0x1AE`) en rejeu
//	                                   de film ; `FUN_14076cb60` s en sert comme `extra` : un
//	                                   `R(1)` de garde apres CHAQUE composant present, et si ce
//	                                   bit vaut 1, un `R(32)` sentinelle `0x0bcddcba`. Idem
//	                                   `FUN_142e2c690` sur le chemin d etat complet.
//
// LE BIT ETAIT DEJA ENJAMBE PAR LE DEPOT DEPUIS LE LOT 1.5.1 — c est le « booleen d un bit » de
// `base+0x0CB45C` dont `identDecalageBit` decale tout ce qui suit. Il est desormais LU
// ([lireControleDeCorruption], [profile.FilmIdentity.ControleDeCorruption]) et il DECIDE la
// grammaire : [grammaireSousFilm] est la regle, ecrite une fois, et elle a deux portes — le
// contexte de film ([FilmContext.ProfilDeBalayage], qui le DERIVE a chaque rendu pour qu un
// profil pose par-dessus ne puisse pas l effacer) et [GrammaireSousFilm] pour `killsource`, qui
// part de l invariant et ne construit pas de contexte.
//
// AUCUN BIT LU NE CHANGE SUR LE PARC, ET C EST MESURE : le drapeau vaut ZERO sur les 1 605 films
// du cache qui portent une section d identification (8 builds, 5 formats : `HI_1_13_0`/27 1349,
// `HI_1_12_0`/27 147, `HI_1_11_0`/25 57, `HI_1_10_0`/24 34, `HI_1_8_0`/24 13, `HI_1_9_0`/24 3,
// `HI_1_4_1`/21 1, `HI_1_5_1`/23 1) — c est-a-dire exactement l ancien defaut de structure. Le
// defaut se trouvait juste ; il l etait par HASARD, et un film qui leverait ce bit aurait
// desynchronise sans un mot. LE RANG MONTE POUR CELA : la couche lit une decision qu elle
// ignorait, pas parce qu un octet a bouge.
//
// GATE DE TRAME, joue avec la carte `snowbound` (celle qui reproduit le tableau du 5.16.4 a
// chaque chiffre ; `streets` sur `dad793c7` donne 5 285 et 72 debordements — la carte n est pas
// indifferente, et le gate n a pas ete joue au hasard) :
//
//	dad793c7  paquets a reste NUL 5 354 / 5 365 · debordements 2 · records 5 641
//	          `ti=35` 75, 0 desynchronise · rejets hors datum 2 · de vue 0 · datums 54
//	bfecd02b  paquets a reste NUL 2 884 / 30 387 · debordements 32 · records 176 786
//	          `ti=35` 129 572, 4 desynchronises · rejets hors datum 23 769 · de vue 0 · datums 10
//
// Chiffre pour chiffre le tableau du `.6` : le port ne deplace RIEN, et c est le resultat
// attendu d un drapeau qui vaut zero partout.
//
// `facts.Rev` NE MONTE PAS, et la decision est ecrite : sur chaque film du parc la valeur lue
// EGALE l ancien defaut, donc aucune ligne de `match_kill_events` ne se redecoderait autrement —
// AUCUN backlog killsource n est ouvert. `profile.Rev` ne monte pas non plus : la couche gagne un
// champ PORTEUR et son accesseur, pas une ligne de table, pas une largeur, pas une borne.
// `replay.SchemaVersion` reste a 67 : aucun champ publie ne change.
//
// LE REPLI EST NOMME ET DIT : `repli_controle_corruption_section_absente` (registre `filmdec`,
// `apres_lecture`) — les 5 films du cache sans section d identification ne declarent pas ce bit,
// la grammaire garde son invariant, [FilmContext.ControleDeCorruptionRepli] le compte et
// `killsource` l avertit par film.

// ENTREE `grammar-2026-09-22.8` (2026-09-22, lot 5.20.1) : LE LECTEUR D IMAGE-CLE EST LU EN
// ENTIER CHEZ L ECRIVAIN, ET LA MARCHE DETERMINISTE PORTE ENFIN SON CADRE.
//
// CE QUI A ETE LU, PAR ADRESSE (Ghidra lecture seule). Le bloc de type 2 d un film n est PAS
// consomme par le repartiteur de paquets : `FUN_1428e22c0` ne connait que neuf types (0, 1, 6,
// 7, 8, 9, 10, 0xb, 0xc) et le type 2 y tombe dans la queue de telemetrie `FilmBlockReadError`.
// Il passe par la SECONDE voie a en-tete de 16 octets — `FUN_1428e2a04` -> `FUN_1428e2a9c`, qui
// lit l en-tete, charge le payload dans `session+0x240`, en fait un lecteur par `FUN_1424c7b4c`
// et appelle `FUN_142e2bfd0`, LE LECTEUR D IMAGE-CLE. Sa boucle remplit un tableau d entrees de
// 200 octets, une par entite vivante, et lit :
//
//	[si `FUN_1428e1c0c(&DAT_144c23178)` > 7] R(1) -> `DAT_144706104`  une fois, en tete de payload
//	par entite :
//	  R(32) -> entree+0x00   l identifiant · R(32) -> entree+0x04   L ARCHETYPE, MOT PLEIN
//	  R(32) -> entree+0x0c   · R(4) (`FUN_142e29cf8`) -> entree+0x08 · R(8) -> entree+0x09
//	  = 108 bits, et si l archetype vaut `0xffffffff` l entree S ARRETE LA
//	  R(32) `n1` ; si > 0 : `FUN_142e31de8` puis `vtable[0x60]` (l etat par defaut), + R(32) de
//	                controle quand le drapeau film est mis
//	  R(32) `n2` ; si > 0 : `vtable[0x88]` (aucun bit) puis `FUN_1428e2b68` -> `FUN_142e2c690`,
//	                la boucle des 64 entrees NOMMEES de la table d archetype
//	                (`session+0x108 + 8 + ti*0x4100`, 0x104 octets par entree), SANS masque de
//	                presence, chacune deserialisee au niveau lu en `entree + 0x100`
//
// Ce corps EST celui que `WalkKeyframeFullState` porte depuis le lot 1.4 : rien de neuf n est
// recopie, et la table de 64 entrees est le REGISTRE du film (`registry.go`, meme cadrage).
//
// CE QUE LE RANG CORRIGE, ET C EST UNE GRAMMAIRE. `WalkKeyframeRecords` lisait un en-tete de
// 64 bits `[id:32][field:26][ti:6]` puis rejouait `TraverseEntity`, c est-a-dire le cadre du
// record NEW du chemin DELTA (R(6) d archetype, etat par defaut, PORTE, MASQUE). Il repartait
// donc 44 bits trop tot, au milieu du premier corps : la marche rendait UN record et s arretait
// sur « en-tete-invalide » sur les deux temoins. Elle enchaine desormais par
// `WalkKeyframeFullState`, et l entree SANS ARCHETYPE (`keyframeArchetypeNone`) se clot a ses
// 108 bits. Le « champ de 26 bits de semantique non etablie » N EXISTE PAS : les 32 bits a
// `q+32` SONT l archetype, `FUN_142e2bfd0` s en sert tel quel pour indexer
// `DAT_144e61d88 + 8 + ti*8`. L hypothese H1 du lot R5 — « le balayeur saute les records dont
// `Field26` n est pas nul » — est REFUTEE PAR L ECRIVAIN : de tels records ne peuvent pas
// exister. `readKeyframeHeader` exige donc le MOT PLEIN sous le cap objet (50) ou
// `0xffffffff`, et `KeyframeChainResult.SkippedFieldNonZero` devient
// `SkippedSansArchetype` — le seul intercale que le filtre fort du balayeur ne voit pas.
//
// CE QUE LA MESURE DIT (`TestMarche520`, `dad793c7`, carte `snowbound`) : la marche passe de
// 1 a 2 records par payload et s arrete desormais sur un composant NOMME —
// `i10 tacmap-mapdismissallock` de `ti=32` — et non plus sur un cadre faux. La marche est
// grammaticalement juste ; ce qui lui manque est le PORT DES COMPOSANTS du lot 3.6, puisqu un
// record d image-cle porte TOUS les composants de son archetype sans masque.
//
// LA FENETRE DE 120 000 BITS DU BALAYEUR RESTE, ET LA MESURE DIT POURQUOI. Elle n existe pas
// dans le jeu (`kfScanFenetreBits`, dument nommee et datee). La retirer n est PAS un gain net :
// sans fenetre le chunk 1 de `dad793c7` passe de 123 a 157 ancres (13,6 % -> 27,8 % du payload)
// mais les chunks 2 a 5 TOMBENT de 187 a 127, `betterThan` elisant un candidat lointain qui
// deraille la chaine. Echanger une heuristique contre une autre n est pas lire la grammaire :
// son retrait est gage sur la marche deterministe, au critere mesurable `KeyframeClosure` a
// 100 % (suivi : `keyframe_closure.golden`).
//
// `facts.Rev` NE MONTE PAS, et la decision est ecrite : `killsource/` marche par
// `DecodeFrameRecords` et `WalkKeyframeWorld`, et NI l un NI l autre ne change (le balayeur
// garde sa fenetre, `WalkKeyframeRecords` n a aucun appelant de production). Son golden est
// refige parce qu il hache la VALEUR de `grammar.Rev` — aucun backlog killsource n est ouvert.
// `replay.SchemaVersion` reste a 67 : aucun champ publie ne change.
//
// CE QUI PEUT BOUGER, ET C EST DIT : `readKeyframeHeader` sert de predicat « la marche a-t-elle
// atterri sur un en-tete ? » dans `navpoint_radial_scan.go` et `objective_scan.go` (compteur
// d observabilite `KeyChained`, drapeau `Reads[].Chained` — ce dernier voyage dans les faits de
// film persistes, `BombReads[].Chained`). Le predicat est desormais STRICT : il exigeait le
// `ti` des 6 bits de queue sous 50, il exige le MOT PLEIN de 32 bits. Aucun consommateur ne
// FILTRE sur ce drapeau (seuls les zones le font, et leur balayage n appelle pas ce lecteur) ;
// les faits de film deja cuits sont de toute facon a recuire, leur en-tete portant la revision
// de grammaire.

// ENTREE `grammar-2026-09-22.9` (2026-09-22, lot 5.22.2) : LE BLOC D ACTION DE LA VUE DE
// CONTROLE N ETAIT PAS UN TROU DE GRAMMAIRE, C ETAIT UN TROU DE CABLAGE.
//
// D2 (5.14) inscrivait `FUN_1406d025c` — le bloc ouvert 170 fois sur `bfecd02b` derriere la
// garde de l entree de controle — comme « dans le film et NON PORTE », et le lot 5.22 devait le
// porter pour y chercher le saut. La lecture de l ecrivain (Ghidra, lecture seule) dit qu il n y
// avait rien a porter : `FUN_1406d025c` est LE MEME deserialiseur que celui qu `i19
// unit-actor-control` appelle depuis `FUN_1408f0778`, et le depot le porte EN ENTIER depuis le
// lot 2.7 sous le nom `consume1406d025c` (2 x 3 bits par `FUN_1431ab1ec`, 2 x 2 bits par
// `FUN_1431ab1cc`, `FUN_1431a0bbc` R(1)[+R(8)], `FUN_1431a0abc` R(1)[+R(10)], le bloc
// `FUN_1431a0cbc`, la queue `FUN_1406d0f20` R(3), deux `FUN_1406d00ec` gardees par les drapeaux
// deja lus, et `FUN_142f26740`). `consumeActionsControle` ne lisait que la garde et rendait
// `false` : la vue C s arretait sur le bit d un bloc dont le decodeur vivait a cote.
//
// AUCUNE LARGEUR N EST NEUVE. La borne est posee A LA SORTIE (`br.BitPos() <= frameLen`) et non
// a l entree, parce qu un `placeDisponible` d entree devrait MAJORER une largeur qui depend des
// gardes — ce que la vue C refuse de faire.
//
// MESURE, GATE DU 5.14.2 INCHANGE (`TestClasses514Bourrage`, reste dans [0 ; 7] ET tous ses bits
// a ZERO) :
//
//	bfecd02b : paquets fermes 2 884 -> 2 900 (+16) sur 30 387, dont 2 900 / 2 900 a bits NULS
//	           et 0 portant un 1 — la grammaire neuve n en casse aucun
//	dad793c7 : 5 354 / 5 365 INCHANGE, et c est ce que la mesure du 5.14.4 annoncait (la garde
//	           d action est FERMEE sur les 5 202 entrees de ce film)
//
// `facts.Rev` NE MONTE PAS, et la decision est ecrite (celle du 5.14.3, mot pour mot) : la couche
// `facts` marche par `DecodeFrameRecords`, qui ne deroule pas les vues par rang — `killsource` ne
// voit pas ce cablage. Son golden est refige parce qu il hache la VALEUR de `grammar.Rev` ;
// aucun backlog killsource n est ouvert. `replay.SchemaVersion` : la montee 67 -> 68 de ce lot
// est celle du genre `clamber` (5.22.4), pas celle-ci.
