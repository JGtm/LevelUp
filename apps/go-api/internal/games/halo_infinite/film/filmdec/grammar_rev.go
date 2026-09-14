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
const GrammarRev = "grammar-2026-09-14.5"
