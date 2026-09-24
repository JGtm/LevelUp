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

// ENTREE `grammar-2026-09-22.9` (2026-09-22, lot 5.21.1) : LE BLOC DE TYPE 1
// QUI PRECEDE CHAQUE IMAGE-CLE EST LU (`type1_datums.go`, `FUN_1429883ec`).
//
// CE QUE LE RANG AJOUTE. Un lecteur, pas une largeur deplacee : `LireBlocDeDatums` consomme les
// 343 019 octets du bloc de type 1 — 8 191 entrees de 79 bits (`R(6)` drapeaux, `R(8)`
// generation, `R(32)` compteur de generation, 33 x `R(1)` de masque par vue, LSB d abord), puis
// 8 191 masques de composants de 256 bits, puis cinq mots de 32 bits. Aucun decodeur de trame,
// d image-cle ou de record ne change : la couche GAGNE un lecteur, elle n en modifie aucun.
//
// LA GRAMMAIRE EST FERMEE PAR L ARITHMETIQUE, PUIS PAR LA MESURE. 8 191 x (79 + 256) + 160 =
// 2 744 145 bits = 343 019 octets a sept bits de bourrage pres, et 343 019 est la taille
// CONSTANTE mesuree du bloc sur les deux films (5.20.3 (d)). Gate (i) : 32 blocs sur 32 (5 sur
// `dad793c7`, 27 sur `bfecd02b`) fermes a sept bits, cardinal 8 191 partout.
//
// D1 (5.20) EST CORRIGE PAR LA MESURE : `+0x04` N EST PAS L ARCHETYPE. `FUN_142e2aab4` pre-remplit
// le conteneur avec `+0x04 = 1` avant la lecture, et le bloc n y porte que TROIS valeurs sur
// 221 157 entrees — 1, 2 et 3 —, toujours egales a `+0x01 + 1`. C est le COMPTEUR DE GENERATION
// du slot. Ce que le bloc porte d utile est ailleurs, et c est mesure : le bitmap de 256 bits par
// slot est le MASQUE DE PRESENCE DES COMPOSANTS, indexe comme `Archetype.Components` — ZERO bit
// hors des bornes de l archetype sur les 32 blocs, et 184 masques distincts pour 2 ambigus sur
// `bfecd02b`.
//
// `facts.Rev` NE MONTE PAS : `killsource/` marche par `DecodeFrameRecords` et
// `WalkKeyframeWorld`, et ce rang n appelle ni ne modifie l un ni l autre ; `LireBlocDeDatums`
// n a aucun appelant de production. Son golden est refige parce qu il hache la VALEUR de
// `grammar.Rev` — aucun backlog killsource n est ouvert. `replay.SchemaVersion` reste a 67.

// ENTREE `grammar-2026-09-22.10` (2026-09-22, lot 5.21) : L EXCLUSION DE LA CHRONIQUE EST ALIGNEE
// SUR SON INTENTION — AUCUNE GRAMMAIRE NE BOUGE.
//
// D3 du lot 5.20 : `fichiersHorsGrammaire` (`rev_test.go`) n excluait que `rev.go`,
// `rev_chronique.go`, `rev_chronique_archive.go` et `_2`, alors que la chronique est rotee
// jusqu a `_5`. Ecrire une ligne dans une archive recente faisait donc monter l empreinte de la
// couche — exactement ce que l exclusion existe pour eviter —, et son commentaire disait « les
// TROIS fichiers » en en listant quatre. L exclusion DERIVE desormais de
// `fichiersDeChroniqueGrammar` : une seule liste, et la prochaine rotation ne peut plus les
// desaccorder. `revision/equivalence_test.go`, qui redeclare le perimetre pour le confronter,
// est aligne dans le meme commit.
//
// CE RANG NE DEPLACE AUCUN OCTET DE DECODAGE : il retire trois fichiers de PROSE de l empreinte.
// `facts.Rev` ne monte pas ; son golden, celui des formes et les fixtures de contrat sont refiges
// parce qu ils hachent ou publient la VALEUR de `grammar.Rev`. `replay.SchemaVersion` reste a 67.

// ENTREE `grammar-2026-09-22.11` (2026-09-22, lots 5.22.2 et 5.22.4) : LE BLOC D ACTION DE LA VUE DE
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

// SUITE DU MEME RANG `grammar-2026-09-22.11` (2026-09-22, lot 5.22.4, fusionne dans l integration au rang .11 avec le 5.22.2) : L ACTION DE MOBILITE EST NOMMEE —
// LE GENRE PUBLIE PASSE DE `mobility` A `clamber`.
//
// La couche `grammar` ecrit l etiquette de genre des etats de mouvement (`movement_states.go`,
// `recevoir`) : sa SORTIE change, meme si aucun bit n est lu autrement. Le verdict est celui de
// l utilisateur dans Theater — neuf intervalles d `i54` confrontes image par image sur
// `bfecd02b`, neuf escalades de rebord, aucun contre-exemple — et le vocabulaire du jeu le
// corrobore (`_action_hoist` / `_action_vault` / `_action_climb_attach`, `143ca0100` ;
// `CharacterPhysicsModeClambering`, `143df73d0`). Details : `types/grammar_mouvement.go` et
// l entree v68 de `replay/document_chronicle.go`.
//
// `facts.Rev` NE MONTE PAS : `killsource` ne lit aucun etat de mouvement. Son golden est refige
// parce qu il hache la VALEUR de `grammar.Rev`. `replay.SchemaVersion` MONTE (67 -> 68), parce
// qu une valeur d enum publie est de la FORME.

// ENTREE `grammar-2026-09-22.12` (2026-09-22, lots 5.23.1, 5.23.2 et 5.23.3 — fusionnes au rang .12 dans l integration) : LA TABLE ANTICIPEE DES ARCHETYPES —
// UN LECTEUR D IMAGE-CLE DE PLUS, AUCUN DECODEUR DEPLACE.
//
// CE QUE LE RANG AJOUTE. `keyframe_anticipe.go` : `TableAnticipee` construit, par une passe sur
// les images-cles de TOUS les chunks, la table `(slot, tete) -> archetype` du film entier, datee
// par chunk. Elle n a AUCUN appelant a ce rang : aucune marche de trame, d image-cle ou de record
// ne change d un bit. La couche GAGNE un lecteur, elle n en modifie aucun.
//
// LA CLE EST CELLE QUE LE JEU COMPARE, ET ELLE EST LUE CHEZ L ECRIVAIN. `FUN_1406caad8` indexe la
// table de datums par `eid & 0x3fffffff` puis exige `*(uint *)(slot * 200 + base) == eid` —
// l eid ENTIER, ses deux bits de tete compris — avant de lire le moindre bit de corps, et rend
// l archetype en `+0x04`. C est la MEME entree de 200 octets que `FUN_142e2bfd0` remplit depuis
// l image-cle (lot 5.20.1) : les deux bits de tete d une image-cle sont donc exactement ceux
// qu un delta doit presenter. Ce qu ils SIGNIFIENT reste ce que le 5.13.1 a etabli (rang de vue
// chez `FUN_142f2e174`, generation du datum chez `FUN_1408f1730`) et les deux films temoins ne
// les departagent pas ; la CLE, elle, n est pas ambigue.
//
// LA MESURE (`TestTable523`, `bfecd02b`) : 12 688 declarations, 1 015 cles distinctes, **ZERO**
// cle portee par plus d un archetype, une seule tete rencontree (`1`). Sur les 23 325 rejets,
// la table en resout **17 432 (74,7 %)** — 17 430 par le chunk SUIVANT, 1 a +8, 1 a +13 —, dont
// `ti=35` 16 932. Les 5 893 restants (25,3 %) ne sont declares par AUCUNE image-cle du film.
// Une cle reduite au seul slot ne resoudrait que 19 rejets de plus, et laisserait passer 638
// en-tetes dont la tete n existe nulle part dans le film : on cle sur ce que le jeu compare.
//
// `facts.Rev` NE MONTE PAS : la table n a aucun appelant, `DecodeFrameRecords` et
// `WalkKeyframeWorld` sont intouches — aucun backlog killsource. Son golden est refige parce
// qu il hache la VALEUR de `grammar.Rev`. `replay.SchemaVersion` reste a 67.

// SUITE DU MEME RANG `grammar-2026-09-22.12` (2026-09-22, lot 5.23.2) : LA LIAISON PAR ANTICIPATION — UN
// REPLI NOMME, DATE ET COMPTE, AU SEUL POINT DE REJET.
//
// CE QUE LE RANG CHANGE, ET OU. `rejetDeVue` recoit l eid COMPLET (et non le slot : la cle que
// `FUN_1406caad8` compare porte les deux bits de tete) et consulte la table anticipee du film
// AVANT de compter un rejet hors datum. `World.LierParAnticipation` pose alors la liaison de la
// table de datums (`BindDatum` : `Soft`, `GenAny`, vue INCONNUE, sans position), la COMPTE par
// archetype (`Observation.LiaisonsParAnticipation`) et journalise le premier usage du film.
// Sans table installee — le cas de tout appelant qui ne la pose pas — pas un bit ne change.
//
// CE N EST PAS UNE GRAMMAIRE. Le record de NAISSANCE n est toujours pas lu : le repli lie
// l entite sur la foi d une image-cle ULTERIEURE, et rend ainsi lisible la SUITE du flux. Il est
// NOMME, DATE (2026-09-22) et COMPTE, et le code le dit la ou on lirait la naissance.
//
// LA MESURE, APRES CE SEUL CHANGEMENT (`TestGate516`, A/B `MOUV523_ANTICIPE=0`) :
//
//	dad793c7 : paquets a reste NUL 5 354 -> 5 355 ; debordements 2 ; fantomes 1 ; ti=35 75 a
//	           0 desynchronise ; rejets hors datum 2 -> 1 ; 8 liaisons (ti=13).
//	bfecd02b : paquets a reste NUL 2 884 -> 3 919 (+1 035) ; rejets hors datum 23 769 -> 16 129
//	           (-7 640) ; records 176 786 -> 240 488 ; ti=35 129 572 -> 164 232, desyncs 4 -> 4 ;
//	           ti=40 5 337 -> 16 141, ti=37 4 551 -> 10 855, ti=42 2 804 -> 10 064, ti=10
//	           1 025 -> 3 041, ti=32 329 -> 791 ; 254 liaisons.
//
// DEUX COMPTEURS DE FAUTE MONTENT SUR LE FILM DENSE, ET LA CAUSE EST DANS CE RANG : debordements
// 32 -> 50 et fantomes 31 -> 49. Le balayage par archetype (`MOUV523_TI`) l attribue a
// l anticipation du BIPEDE (ti=35 seul : 48 et 47), et la raison est celle du 5.16.2 — les slots
// rejetes se concentrent dans la bande 521-601, que toutes les images-cles ulterieures
// declarent, donc un en-tete pris a une position FAUSSE y tombe et lit un corps qui deborde.
// AUCUN paquet ne passe de FERME a fautif : les 18 quittent « reste hors bourrage » (27 471 ->
// 26 418) pour « debordement », et 1 035 le quittent pour « ferme ». `ti=42` anticipe seul RETIRE
// dix debordements.
//
// `facts.Rev` NE MONTE PAS : `killsource/` marche par `DecodeFrameRecords`, qui ne passe pas par
// `rejetDeVue`, et son monde ANTICIPE DEJA — `killsource/world.go` `preload()` lie la premiere
// declaration de chaque slot de TOUTES les images-cles du film. Aucun backlog killsource. Son
// golden est refige parce qu il hache la VALEUR de `grammar.Rev`. `replay.SchemaVersion` reste
// a 67 : 222 etiquettes de composant lues contre 207, mais aucune n est un canal PUBLIE, et
// `Observation` n est jamais publie.

// SUITE DU MEME RANG `grammar-2026-09-22.12` (2026-09-22, lot 5.23.3) : LE REPLI ENTRE EN PRODUCTION PAR
// `ScanMovementStates` — ET LA MESURE DIT POURQUOI C EST LE SEUL.
//
// CE QUE LE RANG CHANGE. `ScanMovementStates` construit la table anticipee du film
// (`ConstruireTableAnticipee`, une passe sur les images-cles de tous les chunks, 1,8 s sur
// `bfecd02b`, aucun decodage de trame), la pose sur son monde et annonce le chunk courant avant
// chaque liaison. Aucun autre fichier de decodage ne bouge.
//
// ET LES AUTRES MARCHES DE PRODUCTION N EN ONT PAS BESOIN, PARCE QU ELLES ANTICIPENT DEJA — plus
// largement, sans datation et sans cle :
//
//	killsource/world.go `preload()`      lie la PREMIERE declaration de chaque slot de TOUTES
//	                                     les images-cles du film, avant de marcher ;
//	object_deaths_march.go `newMarchTimeline()`  fait exactement le meme geste (vehicules,
//	                                     morts d objets, occupations).
//
// C est la raison MESUREE pour laquelle `facts.Rev` ne monte pas et pour laquelle les calques
// `vehicles` / `rides` / `equipmentEpisodes` du document ne bougent pas d une unite : leurs
// mondes connaissaient deja ces slots. `ScanMovementStates` etait la seule marche de production
// qui ne liait que les images-cles DEJA VUES, et c est elle qui gagne.
//
// LE RENDU, MESURE SUR `bfecd02b` (`replay-build`, carte snowbound, faits de film purges pour
// forcer le decodage) : `stances` **616 -> 841** — sprint 355 -> 501, saut derive 252 -> 327,
// mobilite 9 -> 12, accroupi 0 -> 1. Tout le reste a l identique : 90 pistes, 27 703 points,
// 11 vehicules, 3 embarquements, 10 episodes d equipement, 2 568 tirs, 142 ramassages.
// Artefact 2 238 332 -> 2 249 698 octets.
//
// `replay.SchemaVersion` reste **67** : aucune forme ne change, aucun champ n est ajoute.
// `facts.Rev` NE MONTE PAS (cf. ci-dessus) — aucun backlog killsource. Les goldens sont refiges
// parce qu ils hachent ou publient la VALEUR de `grammar.Rev`.

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
// PLUS UN RECORD PROUVE PAR LA GRAMMAIRE DU FILM, ET UNE ABSENCE LUE PAR REPLI NE PROUVE RIEN.
//
// LA CAUSE (rouge M2 x M3 `TestEntitesTi9SurLesBobines`). Dans l image-cle d avant-match de
// `bcb6d393` (c1 p0), `fb1a1a72` (c1 p0, p1) et de la mini-bobine `000d5950` (c1 p1), le record du
// slot 122 (ti 45) couvre ~125 000 bits ; la fenetre suivante porte les vrais 1280..1298 ET une
// fausse ancre 192 / ti 1 (`0x400000C0 00000001`) a l interieur du record 1298. L election (slot
// bas) la retenait, et 1280..1298 disparaissaient — dont 1297, le joueur gere de l index 0, que M2
// lisait alors ARRIVE plus tard. Meme geste en c1 p3 de `bcb6d393` : la fausse 1536 (ti 0) effacait
// 1537..1601 (29 records).
//
// LA REGLE (`keyframe_world_preuve.go`). La table est croissante en slot. Un candidat PROUVE —
// marche d etat complet sous le cadre du film (profil par defaut, drapeau de controle de
// corruption, MPP de son format : invariants du film), `n1 > 0`, composants traverses sans
// desynchronisation, fin EXACTE sur un en-tete valide de slot superieur — interdit tout elu qui
// contredit l ordre bit/slot ; l election se rejoue sans les refutes (si tous le sont, l ancienne
// election tient). Aucun seuil. L exigence de CONTENU est mesuree : un record vide (172 bits) se
// ferme meme lu decale d un bit (chaines ti 41/21/25) — 61 fausses preuves sur 163 559 candidats
// surement faux sans elle, 0 changement hors des quatre paquets cibles avec elle. Compteur :
// `KeyframeWalkStats.Refutations` (`coverage.keyframes.refutations`).
//
// L UNIFORMITE. Tous les balayages de cuisson marchent par `FilmContext.MarcheDImageCle` ; la forme
// SANS preuve (`WalkKeyframeWorld`) reste celle des instruments, allowlist fermee
// (`archlint/keyframe_walk_proof_test.go`). Mesure (7 bobines + mini-bobine) : 5 paquets changent,
// 1 refutation chacun ; perdues SEULEMENT les fausses ancres 192 (x4) et 1536 ; regagnes 1280..1298
// (x4) et les 29 records de 1537..1601. Fermeture : ti=9 1 738 -> 1 741 records, tous fermes.
// Golden des familles : `carrierMarks` seul (records marches 1456 -> 1474, marques 0).
//
// LE PRINCIPE (`player_entities.go`). Les candidats ECARTES par recalage ou election, et les
// records ti=9 illisibles d une image-cle, en font une image-cle DOUTEUSE pour ces slots
// (`DouteDAbsence`, persiste avec les entites) : une absence n y prouve ni un depart ni une arrivee
// tardive. La presence (`replay/occupants_presence.go`) ne borne que sur une absence PROUVEE, sinon
// elle differe la borne a l image-cle prouvee suivante. Compteurs `coverage.seats.imagesClesDouteuses`
// et `bornesDifferees` (0 et 0 sur les sept bobines apres la regle).
//
// ET L IMAGE-CLE DIT QUI N EST PLUS LA (`keyframe_liaison.go`, meme partie). La marche des etats de
// mouvement ne retirait une liaison que sur un DEL LU ; un DEL manque laissait la liaison du mort,
// et l occupant suivant du slot se decodait sous son archetype (`a0c36016` : NEW d un `ti 30` au
// chunk 27, bipede ne au 38 sur le slot 649, cinq vies et 24 intervalles perdus — la marche de M3,
// qui lie les tables entieres, menait les deltas jusqu au NEW du mort). A chaque image-cle, une
// liaison que ni la chaine, ni la table de datums, ni un candidat ECARTE ne porte est oubliee.
// Mesure (cinq temoins) : 0 vie ne perd un intervalle, 15 en gagnent, les 5 de `a0c36016` rendues ;
// les dotations de naissance gardent leur liaison (l oubli y retirait 1 fermeture sur 104).
