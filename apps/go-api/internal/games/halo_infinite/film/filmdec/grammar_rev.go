package filmdec

// grammar_rev.go — LA REVISION DE LA GRAMMAIRE DU FILM.
//
// # LA REGLE A TROIS ETAGES, ET CE QUE CHACUN PROTEGE
//
//	GrammarRev              monte a TOUT changement de grammaire — une largeur, un cadre, un
//	                        ordre de composants, un lecteur neuf. C'est la revision de CE qui lit
//	                        les octets du film.
//	KillSourceDecoderRev    monte quand la SORTIE de `killsource` peut changer : les lignes de
//	                        kill deja en base sont alors candidates au backlog de redecodage.
//	SchemaVersion           monte quand le CONTENU CUIT change : `backfill-replay` re-cuit tout
//	                        artefact anterieur.
//
// Les trois sont INDEPENDANTES et ne se remplacent pas. Une largeur corrigee dans un composant
// que personne ne consomme encore fait monter `GrammarRev` SEULE. La meme largeur, une fois
// branchee sur le kill feed, fait monter `KillSourceDecoderRev` aussi. Si l'artefact publie s'en
// trouve change, `SchemaVersion` monte a son tour. Confondre les trois, c'est soit re-cuire le
// parc pour un commentaire, soit laisser en base des lignes decodees par une grammaire morte.
//
// # POURQUOI UNE CONSTANTE, ET PAS UN COMMENTAIRE
//
// `KillSourceDecoderRev` a porte pendant des mois la consigne « la faire evoluer a chaque
// changement de decodage » : mesure du 2026-09-05, 14 commits sur le decodeur, ZERO bump. Une
// consigne ecrite dans un commentaire ne se tient pas toute seule. Le garde-rail qui rend
// celle-ci executoire est `grammar_rev_fingerprint_test.go` : il hache les sources de `filmdec`
// ET de `killsource`, et rougit des qu'une d'elles bouge sans que cette constante monte.

// GrammarRev est la revision de la grammaire de lecture du film.
//
// FORME : `grammar-AAAA-MM-JJ`, la date du jour ou la grammaire a change, suivie d'un `.N`
// quand un SECOND lot la change le MEME jour. Ce qui doit rester separable est le LOT, pas le
// commit : deux changements d'un meme lot partagent la revision (lot 1.1.5, 2026-09-14), deux
// LOTS ne la partagent pas.
//
// LE SUFFIXE EST NE AU LOT 1.2 (2026-09-14), et il corrige une regle qui se retournait contre
// son objet. La regle disait « deux changements le meme jour partagent la meme revision » ; le
// lot 1.2 (le registre lu a l'octet 8) tombait le meme jour que le lot 1.1 (l'equipe a l'octet
// 37 du pied). La partager aurait voulu dire regenerer le golden sur la branche « revision
// inchangee, empreinte differente » — c'est-a-dire faire taire le ratchet dans le cas precis
// pour lequel il existe : une grammaire qui change. La forme admet donc un rang, et la revision
// continue de nommer ce qu'elle nomme.
// LOT 1.9.2 (2026-09-15) : `grammar-2026-09-15.1` -> `grammar-2026-09-15.2`. Le résolveur de
// distance de touche (`weapon_hit_distance_resolver.go`) prend désormais l'ENTRÉE DE CATALOGUE de
// la carte au lieu de ses seules bornes, et en impose le DÉCOUPAGE d'i0 au balayage des positions
// — là où `ScanFilmOptions.Layout` restait nil, donc où `DetectI0LayoutOf` décidait. Aucune
// grammaire d'octets n'est réécrite ; ce qui change est QUELLE grammaire s'applique, et c'est
// exactement ce que cette révision doit nommer.
// FUSION (2026-09-16) : l integration portait `.3` (lots 1.9.2 et 1.9.3) et la branche du lot
// 1.9.1 bis `.6` (gardes n1/n2, quatre sites de ti=37, prefixe objet relu, profil par build) ;
// les deux grammaires sont reunies ici, au rang suivant.
// LOT 1.9.4 (2026-09-15) : `grammar-2026-09-15.7` -> `.8`. `DetectFilmMapEntry` est SUPPRIMEE de
// `weapon_hit_distance_resolver.go` — elle identifiait la CARTE d un film par la signature de ses
// largeurs d axe (`DetectI0Layout`), alors que son unique appelant tenait deja le nom de carte du
// match. AUCUN BIT LU NE CHANGE, et l empreinte monte quand meme parce qu elle hache des octets
// de source (c est ecrit dans son en-tete) : ce qui change est QUELLE carte, donc quelles bornes
// et quel decoupage, s appliquent a un film — le meme genre de changement que la revision `.2`
// nommait au lot 1.9.2. `KillSourceDecoderRev` ne bouge PAS (`film/killsource/` n a pas bouge,
// et son propre ratchet d empreinte fait foi) ; `SchemaVersion` non plus (le chemin de cuisson
// n appelait pas cette fonction — verifie le 2026-09-15 : equivalence 10/10 identiques,
// corpus gate 14 temoins a 0 gain / 0 perte / 0 changement).
// LOT 1.9.1 ter (2026-09-15) : `.7` -> `.8`. AUCUN BIT LU NE CHANGE, mais la CLE de la
// grammaire, si — et c'est exactement ce que cette revision doit nommer. Le decoupage du bloc
// `object-multiplayer-properties` etait keye par le NOM DE BUILD ; il l'est desormais par la
// VERSION DE FORMAT de `chunk_00` (`chunk_00+4`), qui est la valeur que le LECTEUR du jeu
// consulte (`FUN_14299ab50` : la largeur du registre et celle de la table par type en
// derivent ; `FUN_1428e1c0c` : c'est elle que les six branches de version de l'executable
// lisent). Sur les 1 351 films du cache les deux cles donnent le MEME decoupage — le
// changement est verifiable et neutre — mais la nouvelle couvre les cinq films sans section
// d'identification, que l'ancienne ne pouvait pas nommer.
// LOT 1.9.1 ter (2026-09-15, second commit) : `.8` -> `.9`. AUCUN BIT LU NE CHANGE ICI NON
// PLUS, et la revision monte pour la meme raison qu au `.8` : le CADRE. La condition du repli
// `repli_largeurs_mpp_calibrees_sur_le_film` est renommee `build_sans_profil_relu` ->
// `format_sans_profil_relu` (elle nommait une cle qui n existe plus), et le declenchement du
// repli sur une version de format INCONNUE est desormais COMPTE
// (`filmdec.UnknownFormatExpvarPairs` -> `filmdec_unknown_format_<n>`, cable dans
// `replay/mpp_format_inconnu.go`) et signale par un avertissement par film. Un consommateur qui
// decide de redecoder doit voir que la condition du repli a change de nom ; un exploitant doit
// voir qu un patch du jeu a change le format. `SchemaVersion` reste 59.
// FUSION (2026-09-16) : l integration portait `.8` (lot 1.9.4) et la branche du lot 1.9.1 ter
// `.9` (cle = version de format, repli compte) ; les deux sont reunies ici, au rang suivant.
const GrammarRev = "grammar-2026-09-15.10"
