package profile

// rev.go — LA REVISION DE LA COUCHE PROFIL, ET SA CHRONIQUE.
//
// # CE QUE CETTE REVISION DIT, ET CE QU ELLE COMMANDE
//
// La couche `profile` ne lit AUCUN octet de film — c est meme sa definition (ADR 0034 D-1) —
// mais elle porte les LARGEURS, les BORNES et les PLAGES que les lecteurs appliquent : le
// decoupage d i0, le decoupage du bloc `object-multiplayer-properties`, la transposition par
// build et par version de format, les bornes de dequantification par carte. Une ligne de cette
// table qui change fait lire au decodeur d autres bits aux memes offsets : c est un changement
// de DECODAGE, exactement comme une largeur corrigee dans `grammar`.
//
// Elle est donc la SECONDE des quatre (decision V15 (11) du PLAN_DECODEUR_FILM_2026-09-13),
// entre `source` — la porte aux octets, dont elle hache la VALEUR (V15 (12)) — et `grammar`, qui
// hache la sienne. Le sens unique est ainsi tenu par le mecanisme et non par la vigilance :
// une montee de `source.Rev` fait monter `profile.Rev`, qui fait monter `grammar.Rev`, qui fait
// monter `facts.Rev` — et le backlog killsource avec elle (D6, signal utilisateur).
//
// # POURQUOI ELLE NAIT MAINTENANT, ET PAS AU LOT 2.5.b QUI A CREE LA COUCHE
//
// Tant que la couche n avait pas de revision propre, ses octets etaient haches dans l empreinte
// de la GRAMMAIRE, et dans elle seule (`racinesGrammaire` portait cinq racines). Le gate mordait
// donc — rien ne passait — mais il ne disait pas CE QUI avait bouge : une borne de carte et un
// ordre de composants rendaient le meme diagnostic. Les quatre revisions du lot 2.6 rendent la
// reponse lisible sans rien relacher : chaque couche hache ses propres octets, et les valeurs
// de celles du dessous.
//
// # CE QUI EST HACHE, ET CE QUI NE L EST PAS
//
// Les sources `.go` NON-TEST de ce paquet, sous le mecanisme central (`film/revision`, lot
// 2.6.0) : chemin relatif A LA RACINE hache a cote du contenu, fins de ligne normalisees en LF,
// `testdata/` et `_test.go` ecartes. CE fichier est EXCLU : il DECRIT la couche, il n en fait pas
// partie — sans quoi faire monter la revision changerait aussi l empreinte, et la branche « la
// revision a change sans que la couche bouge » serait du code mort (correctif R1 P2-3).
//
// LE CATALOGUE DES CARTES N EST PAS HACHE, et c est une limite ASSUMEE : les bornes de
// dequantification par carte sont chargees depuis `data/titles/halo_infinite/reference/`
// (`map_bounds.go`), donc depuis des fichiers de DONNEES que cette empreinte ne voit pas. Hacher
// un repertoire de donnees de production ferait dependre un gate de source de l etat du poste.
// Ce que l empreinte garde est le CODE qui les lit et les compose ; ce que le catalogue change,
// le corpus gate le voit.
//
// VALEUR AMONT : `source.Rev`, et elle seule.
//
// # LA FORME, ET POURQUOI LE PREMIER RANG DU JOUR N A PAS DE SUFFIXE
//
// `profile-AAAA-MM-JJ[.N]`, `N >= 2` : deux LOTS du meme jour se separent par leur rang, et le
// premier lot du jour s ecrit SANS suffixe (`revision.ParserRevision` refuse `.1` — une meme
// position ne s ecrit pas de deux facons).
//
// # LA CHRONIQUE — UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// ENTREE `profile-2026-09-17` (2026-09-17, lot 2.6.1) : NAISSANCE DE LA REVISION DE PROFIL.
// AUCUNE LIGNE DE LA TABLE NE CHANGE.
//
// La couche existe depuis le lot 2.5.b, extraite de la grammaire par inversion de dependance.
// Ce rang pose la constante, son golden a historique et son gate ; il ne change AUCUNE largeur,
// AUCUNE borne, AUCUNE provenance, et aucun match deja decode n est candidat a quoi que ce soit.
// Ce qui change est la LISIBILITE du gate : jusqu ici la couche n etait hachee que dans
// l empreinte de la grammaire, qui rendait le meme diagnostic pour une borne de carte et pour un
// ordre de composants.
const Rev = "profile-2026-09-17"

// L EMPREINTE DES SOURCES DE LA COUCHE VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/profile_rev.golden` porte le couple (revision, empreinte) avec son historique, et
// `rev_test.go` le compare aux sources non-test de ce paquet et a la valeur de `source.Rev`.
//
// POURQUOI LE GOLDEN PORTE LES DEUX VALEURS (lecon du gate `killsource`, revue adversariale du
// 2026-09-12, constat P1-4) : tant qu un gate ne compare que l EMPREINTE, remettre la revision a
// sa valeur d avant — en gardant la nouvelle empreinte — reste VERT. Le couple rend les deux
// derives visibles, avec deux messages distincts.
