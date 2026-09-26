package source

// rev.go — LA REVISION DE LA COUCHE SOURCE, ET SA CHRONIQUE.
//
// # CE QUE CETTE REVISION DIT, ET CE QU ELLE COMMANDE
//
// La couche `source` est LA PORTE AUX OCTETS du film (ADR 0034 D-1, D-2) : charger, decompresser,
// decouper en chunks et en paquets, lire les bits. Tout ce que les couches du dessus decodent
// passe par elle. Une revision de source qui monte veut donc dire la chose la plus large du
// chantier : LA LECTURE DES OCTETS A CHANGE, TOUT RE-DECODE — il n y a pas de fait, pas de
// document cuit, pas de ligne de kill dont on puisse dire qu il n est pas concerne.
//
// C est aussi la raison pour laquelle elle est la PREMIERE des quatre (decision V15 (11) du
// PLAN_DECODEUR_FILM_2026-09-13) : les couches qui l importent hachent sa VALEUR (V15 (12)), donc
// un changement de lecture d octets fait monter les faits MECANIQUEMENT, sans que personne ait a
// y penser.
//
// # CE QUI EST HACHE, ET CE QUI NE L EST PAS
//
// Les sources `.go` NON-TEST de ce paquet, sous le mecanisme central (`film/revision`,
// lot 2.6.0) : chemin relatif A LA RACINE hache a cote du contenu, fins de ligne normalisees en
// LF, `testdata/` et `_test.go` ecartes. CE fichier est EXCLU : il DECRIT la couche, il n en fait
// pas partie — sans quoi faire monter la revision changerait aussi l empreinte, et la branche
// « la revision a change sans que la couche bouge » serait du code mort (meme correctif que
// `fichiersHorsGrammaire` cote grammaire, revue R1 P2-3).
//
// AUCUNE VALEUR AMONT : `source` est la racine du sens unique, elle ne depend d aucune autre
// couche. C est aussi ce qui rend son empreinte comparable a celle d un mecanisme sans amont
// (`revision.Calculer` n ecrit AUCUN octet quand la liste d amonts est vide).
//
// # LA FORME, ET POURQUOI LE PREMIER RANG DU JOUR N A PAS DE SUFFIXE
//
// `source-AAAA-MM-JJ[.N]`, `N >= 2` : deux LOTS du meme jour se separent par leur rang, et le
// premier lot du jour s ecrit SANS suffixe (`revision.ParserRevision` refuse `.1` — une meme
// position ne s ecrit pas de deux facons).
//
// # LA CHRONIQUE — UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// ENTREE `source-2026-09-16` (2026-09-16, lot 2.6.1) : NAISSANCE DE LA REVISION DE SOURCE.
// AUCUN OCTET N EST LU AUTREMENT.
//
// La couche `source` existe depuis le lot 2.5.a et porte depuis le lot 2.4.1 le lecteur de bits
// canonique du depot. Elle etait jusqu ici hachee dans l empreinte de la GRAMMAIRE et dans elle
// seule : une largeur de lecture de bits corrigee faisait monter `GrammarRev`, jamais une
// revision qui lui soit propre — et `facts` n avait donc aucun moyen de dire « ce que je publie
// depend aussi de la facon dont les octets sont atteints ». Ce rang pose la constante, son
// golden a historique et son gate ; il ne change AUCUNE lecture, et aucun match deja decode
// n est candidat a quoi que ce soit.
//
// ENTREE `source-2026-09-16.2` (2026-09-16, lot 2.5.e) : DEUX ENTREES NEUVES A LA PORTE,
// AUCUNE LECTURE CHANGEE.
//
// La descente de la grammaire restee dans `internal/analysis` (decision V15 (4)) a amene ici
// deux conventions qui vivaient dans le lecteur des temps forts :
//
//	[U32BE]         le seul entier BIG-endian du film — l horodatage du bloc d evenement du
//	                chunk des temps forts, que la forme d origine lisait en `uint:32`. Il
//	                s ecrivait `binary.BigEndian.Uint32` dans `internal/analysis`.
//	[ErrEnTeteZlib] la sentinelle qui separe les DEUX echecs de [Decompresser]. Le lecteur des
//	                temps forts recoit son chunk clair du CDN et compresse du cache : il doit
//	                traverser sur l en-tete et remonter l erreur sur la casse EN COURS de flux.
//	                Sans sentinelle, ce choix serait un test de chaine de caracteres.
//
// [Decompresser] enveloppe desormais son erreur d en-tete dans cette sentinelle : le TEXTE de
// l erreur change, la valeur rendue non. AUCUN OCTET N EST LU AUTREMENT — aucun appelant de
// [Load], de [Paquets] ni des lecteurs de bits ne voit une valeur differente, et rien n est
// candidat a un redecodage.
const Rev = "source-2026-09-16.2"

// L EMPREINTE DES SOURCES DE LA COUCHE VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/source_rev.golden` porte le couple (revision, empreinte) avec son historique, et
// `rev_test.go` le compare aux sources non-test de ce paquet.
//
// POURQUOI LE GOLDEN PORTE LES DEUX VALEURS (lecon du gate `killsource`, revue adversariale du
// 2026-09-12, constat P1-4) : tant qu un gate ne compare que l EMPREINTE, remettre la revision a
// sa valeur d avant — en gardant la nouvelle empreinte — reste VERT. Le couple rend les deux
// derives visibles, avec deux messages distincts.
