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
// # LES XUIDS
//
// Un xuid est remplace par son fictif, tire par `internal/identitefictive` (fonction pure, dans la
// plage `[0x0009000000000000, 0x000A000000000000[` que le decodeur exige pour reperer la table des
// joueurs). Il se cherche sous cinq formes : 64 bits petit-boutiste et 64 bits gros-boutiste (MSB
// d abord, table du registre) a CHACUN des huit decalages de bit, puis en texte aligne a bornes
// nettes — decimal a seize chiffres, hexadecimal a seize chiffres, hexadecimal octets inverses.
// Toutes ont la meme longueur avant et apres. La table refuse un xuid hors plage ou en double, deux
// originaux de meme fictif, et un fictif qui serait aussi un original. Le bilan donne le nombre de
// xuids de la liste que des fichiers portaient : un ecart avec une mesure faite par ailleurs se lit
// la.
//
// Le registre VERSIONNE des fictifs est `libs/film/testdata/IDENTITES.tsv`, et son garde-rail
// (`TestLesXuidsDesFixturesSontFictifs`) refuse tout xuid de la plage, hors registre, dans un
// fichier texte de `libs/film`.
//
// # LES TABLES D ORIGINE NE SONT PAS VERSIONNEES
//
// La table des noms associe les noms reels aux pseudonymes, la liste des xuids donne les originaux :
// les versionner annulerait l anonymisation. Format de la table des noms : une ligne
// `original<TAB>pseudonyme[<TAB>donnees]`, lignes vides et `#` ignorees ; `donnees` reserve le nom
// aux fichiers sous un dossier `testdata` (un nom qui est aussi un chemin de module, comme celui du
// mainteneur, ne doit pas etre touche dans `go.mod` ni dans le code). Format de la liste des xuids :
// un xuid decimal par ligne. Les joueurs qui ont consenti a figurer dans les fixtures n y figurent
// simplement pas. `cmd/replay-equiv` relit la meme liste pour retraduire les faits figes en xuids du
// parc (`-originaux`).
//
// # USAGE (depuis apps/go-api)
//
//	GAMERTAGS_CORRESPONDANCE=<table> XUIDS_ORIGINAUX=<liste> GAMERTAGS_CHEMINS="<chemin>;<chemin>" //	  go test ./cmd/anonymise-gamertags/ -run TestAnonymiserFixtures -count=1
//
// rend l inventaire (occurrences par fichier, par dossier et par forme) sans rien ecrire ;
// `GAMERTAGS_ECRIRE=1` ecrit. L une des deux variables suffit (noms seuls, ou xuids seuls). Les
// chemins (fichiers ou dossiers) sont relatifs a la racine du depot ou absolus. Comme les autres
// portes de regeneration, le test ECHOUE TOUJOURS en nommant ce qu il a vu ou ecrit : `go test`
// jette la sortie d un paquet qui passe.
//
// A REJOUER apres toute regeneration d une fixture depuis le cache de films (mini-bobines,
// `inputs_*.bin.gz`, `v41_chunk_he.bin`...) : la regeneration y remet les noms et les xuids reels.
package anonymisegamertags
