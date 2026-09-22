package grammar

// rev.go — LA REVISION DE LA GRAMMAIRE DU FILM.
//
// # LES QUATRE REVISIONS DE COUCHE, ET LE SENS UNIQUE QUI LES RELIE (decision V15 (11))
//
//	source.Rev    la porte aux octets : charger, decompresser, decouper, lire les bits.
//	profile.Rev   la table du decodeur : les largeurs, les bornes, les provenances.
//	Rev           CETTE constante — la grammaire de lecture : un cadre, un ordre de composants,
//	              un lecteur. Elle hache ses propres octets PLUS les VALEURS de `profile.Rev` et
//	              de `source.Rev` (V15 (12)).
//	facts.Rev     la sortie des faits : les lignes de kill deja en base sont candidates au
//	              backlog de redecodage. Elle hache la VALEUR de CETTE constante.
//
// La chaine est mecanique et c est tout son interet : une largeur corrigee dans `profile` fait
// monter `profile.Rev`, donc `Rev`, donc `facts.Rev` — sans que personne ait a y penser. Le
// backlog, lui, part sur SIGNAL UTILISATEUR (decouverte D6), jamais automatiquement.
//
// # LE CINQUIEME ETAGE, QUI N EST PAS UNE REVISION DE COUCHE
//
//	SchemaVersion   monte quand le CONTENU CUIT change : `backfill-replay` re-cuit tout artefact
//	                anterieur. Une largeur corrigee dans un composant que personne ne consomme
//	                encore fait monter `Rev` SEULE ; si l artefact publie s en trouve change,
//	                `SchemaVersion` monte a son tour. Confondre les etages, c est soit re-cuire
//	                le parc pour un commentaire, soit laisser en base des lignes decodees par une
//	                grammaire morte.
//
// # POURQUOI UNE CONSTANTE, ET PAS UN COMMENTAIRE
//
// La revision des faits a porte pendant des mois la consigne « la faire evoluer a chaque
// changement de decodage » : mesure du 2026-09-05, 14 commits sur le decodeur, ZERO bump. Une
// consigne ecrite dans un commentaire ne se tient pas toute seule. Le garde-rail qui rend
// celle-ci executoire est `rev_test.go` : il hache les sources de la couche et les valeurs de ses
// deux amonts, et rougit des qu une d elles bouge sans que cette constante monte.

// Rev est la revision de la grammaire de lecture du film.
//
// ELLE S APPELAIT `GrammarRev` JUSQU AU LOT 2.6.1 : le nom disait la couche deux fois une fois
// qualifie (`grammar.GrammarRev`), et les quatre couches portent desormais la meme forme
// (`source.Rev`, `profile.Rev`, `grammar.Rev`, `facts.Rev`). La SERIE, elle, n est pas
// renumerotee — c est la meme chronique, et une chronique reecrite ne dit plus ce qui s est
// passe.
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
// nommait au lot 1.9.2. `facts.Rev` ne bouge PAS (`film/facts/killsource/` n a pas bouge,
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
// (`grammar.UnknownFormatExpvarPairs` -> `filmdec_unknown_format_<n>`, cable dans
// `replay/mpp_format_inconnu.go`) et signale par un avertissement par film. Un consommateur qui
// decide de redecoder doit voir que la condition du repli a change de nom ; un exploitant doit
// voir qu un patch du jeu a change le format. `SchemaVersion` reste 59.
// FUSION (2026-09-16) : l integration portait `.8` (lot 1.9.4) et la branche du lot 1.9.1 ter
// `.9` (cle = version de format, repli compte) ; les deux sont reunies ici, au rang suivant.
// LOT 1.9.13 (2026-09-15) : `.7` -> `.8`. AUCUNE grammaire d octets ne change, et aucun bit lu
// n est lu autrement : `objectives.RoundBounds.Starts` est un ACCESSEUR de lecture sur des
// bornes de manche deja mesurees, que la decoupe des vies du rejeu consomme. L empreinte hache
// les OCTETS des trois paquets (cf. rev_test.go, « il ne distingue pas un
// changement de grammaire d une reformulation de commentaire ») : le faux positif coute cette
// ligne, et c est le marche assume du garde-rail. `facts.Rev` ne bouge PAS —
// `killsource/` n est pas touche.
// FUSION (2026-09-16) : l integration portait `.10` et la branche du lot 1.9.13 `.8` (accesseur
// neuf dans objectives, faux positif d empreinte) ; reunies ici, au rang suivant.
// LA CHRONIQUE DES RANGS, a partir du `.11`, vit dans `rev_chronique.go` — les rangs `.12` a
// `.27` dans `rev_chronique_archive.go` : une entree par rang, et rien qu une. Elle EST la
// documentation de cette constante — elle en a seulement ete sortie le 2026-09-18 (lot 2.4.1)
// parce que ce fichier avait atteint le seuil de 500 lignes et qu une chronique qui ne peut plus
// grandir cesse d etre tenue. LE GATE LIT LES DEUX FICHIERS, dans l ordre chronologique, et
// exige que les rangs s y suivent sans trou a partir du `.12`.
const Rev = "grammar-2026-09-22.13"
