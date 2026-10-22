// Package anonymisegamertags remplace, dans les fixtures de test, les gamertags de joueurs tiers
// par des pseudonymes de MEME longueur — dans les octets des films comme dans leurs derives.
//
// # POURQUOI C EST UN TEST ET NON UN BINAIRE
//
// L outil lit et reecrit des octets de film : il decompresse les chunks zlib, remplace des chaines
// a tout decalage de bit et recompresse. Hors de la couche `source` du decodeur, la regle de
// l ADR 0034 D-2 interdit ce geste au code de PRODUCTION (ratchet
// `archlint/no_raw_film_bytes_outside_source_test.go`, qui refuse `compress/zlib` sous `cmd/`) ;
// les tests en sont exemptes. C est donc la forme qu ont deja les autres ecrivains de fixtures de
// film du depot (`MiniFilmRegenerate`, la bobine v40 de `killsource`) : une porte de test
// verrouillee par une variable d environnement. Ce fichier n existe que pour porter cette
// documentation ; le code vit dans les fichiers `_test.go` du dossier.
//
// # CE QUE L OUTIL GARANTIT
//
//   - Un pseudonyme a la MEME longueur que l original, en octets UTF-8 comme en unites UTF-16 (la
//     table refuse sinon) : aucune longueur, aucun prefixe, aucun decalage du flux ne bouge, seuls
//     les bits des caracteres changent.
//   - Les gamertags se cherchent en UTF-16 petit- et gros-boutiste a CHACUN des huit decalages de
//     bit d un flux lu MSB d abord (la convention de `source.Bits`), et en UTF-8 aligne sur
//     l octet. Ce sont les trois formes sous lesquelles un film et ses derives les ecrivent (table
//     des joueurs de `chunk_00`, kill-feed et pied du chunk highlight, faits serialises, JSON).
//   - Un chunk zlib est decompresse, reecrit puis recompresse au niveau que son en-tete declare ;
//     un fichier gzip garde son en-tete ; un fichier brut (chunk deja clair, texte) est reecrit en
//     place. Un flux zlib TRONQUE est refuse : sa forme ne se reproduit pas.
//   - Un fichier sans occurrence n est pas reecrit, a l octet pres.
//
// # LA TABLE DE CORRESPONDANCE N EST PAS VERSIONNEE
//
// Elle associe les noms reels aux pseudonymes : la versionner annulerait l anonymisation. Format :
// une ligne `original<TAB>pseudonyme`, lignes vides et `#` ignorees. Les joueurs qui ont consenti
// a figurer dans les fixtures n y figurent simplement pas.
//
// # USAGE (depuis apps/go-api)
//
//	GAMERTAGS_CORRESPONDANCE=<table> GAMERTAGS_CHEMINS="<chemin>;<chemin>" \
//	  go test ./cmd/anonymise-gamertags/ -run TestAnonymiserFixtures -count=1
//
// rend l inventaire (occurrences par fichier) sans rien ecrire ; `GAMERTAGS_ECRIRE=1` ecrit. Les
// chemins (fichiers ou dossiers) sont relatifs a la racine du depot ou absolus. Comme les autres
// portes de regeneration, le test ECHOUE TOUJOURS en nommant ce qu il a vu ou ecrit : `go test`
// jette la sortie d un paquet qui passe.
//
// A REJOUER apres toute regeneration d une fixture depuis le cache de films (mini-bobines,
// `inputs_*.bin.gz`, `v41_chunk_he.bin`...) : la regeneration y remet les noms reels.
package anonymisegamertags
