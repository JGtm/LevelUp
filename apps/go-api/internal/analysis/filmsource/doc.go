// Package filmsource — LA SOURCE UNIQUE DU FILM THEATER : DECOMPRESSER UNE FOIS, DECOUPER UNE FOIS.
//
// Un film Theater est une suite de chunks numerotes, chacun eventuellement compresse en zlib, et
// chaque chunk de donnees est une suite de paquets a en-tete fixe. Avant ce paquet, la chaine de
// cuisson d'un artefact de rejeu relisait et redecompressait le film entier ~36-40 fois, avec
// TROIS inflates et TROIS marcheurs de paquets DIVERGENTS (`filmdec`, `killsource`,
// `objectiveevents`). Ici : un chargement, un jeu de paquets, une grammaire.
// Reference : `.ai/V7.5/PLAN_CUISSON_PERF.md` §2 (conception) et §3 D1/D3.
//
// # POURQUOI UN PAQUET FEUILLE, ET POURQUOI PAS `filmdec`
//
// Ce paquet n'importe RIEN du depot (stdlib seule) — un garde-rail le verifie
// (`internal/archlint/filmsource_leaf_test.go`). Ce n'est pas une coquetterie d'architecture,
// c'est la seule position possible : `filmcache` importe `objectiveevents` (`filmcache.go`), et
// cinq tests INTERNES de `filmdec` importent `objectiveevents` ou `filmcache`
// (`sonde_registre_verdicts_test.go`, `navpoint_ti12_radial_test.go`,
// `objectif_ti11_minuteurs_test.go`, `ti47_annonces_test.go`, `zone_census_report_test.go`).
// Loger la source du film dans `filmdec` ferait donc importer `filmdec` par `objectiveevents`, ou
// `filmcache` par `filmdec` : un cycle, en production ou en test. Une feuille n'en cree aucun.
//
// # LA GRAMMAIRE (D3 REVISEE DU 2026-09-02) — ET LA MESURE QUI L'A ECRITE
//
// En-tete de 16 octets LITTLE-ENDIAN : [u16 type][2 octets][u32 taille][u64 horodatage]. Puis,
// dans l'ordre :
//
//	(1) arret si `off+16+taille > len(chunk)` (en-tete incoherent : fin de chunk ou padding) ;
//	(2) le paquet est EMIS, taille 0 comprise ;
//	(3) arret APRES avoir emis un paquet de type 7 (CHUNK_END) ;
//	(4) arret AVANT emission sur un paquet de taille 0 qui n'est PAS de type 7 (en-tete degenere).
//
// La regle (2) a remplace un « arret sur taille <= 0 » candidat, sur des FAITS et non par gout :
// mesure sur 1 378 films du cache (`.ai/V7.5/MESURES_CUISSON_PERF.md` §2b) plus un diagnostic
// paquet par paquet sur trois films, tous chunks. Sur les chunks de DONNEES, l'unique paquet de
// taille 0 est le terminateur CHUNK_END, en DERNIERE position, sans un octet apres (27/27, 32/32,
// 43/43 chunks) : « taille 0 » et « CHUNK_END » y sont LE MEME PAQUET. La candidate le supprimait
// et faisait diverger tous les chunks de donnees de la vue de `filmdec` ; la grammaire retenue les
// rend BIT-IDENTIQUES sur tout le cache. Les en-tetes degeneres de taille 0 au MILIEU d'un chunk
// (regle 4) n'existent que dans `chunk_00`, le REGISTRE — que personne ne marche comme un flux de
// paquets (`filmdec.ParseRegistryChunk` le lit comme un registre ECS).
//
// # L'INDEXATION DES CHUNKS : LA POSITION N'EST PAS LE NUMERO
//
// Un `Film` expose les chunks a la POSITION ou la source les donne, et cette position ne vaut pas
// le NUMERO du fichier. Sur un cache complet les deux coincident — `chunk_00.bin` est le registre
// a la position 0, les donnees suivent — mais une bobine partielle ou une fixture qui commence a
// `chunk_01.bin` (`replay/testdata/minifilm_000d5950`) les DESACCORDE : sa position 0 est un chunk
// de DONNEES. Confondre les deux revient a marcher un chunk de donnees comme un registre, ou
// l'inverse (`Packets(0)` sur le vrai registre rend tres peu de paquets — arret sur le premier
// en-tete degenere, regle 4).
//
// D'OU LA REGLE, POSEE AU LOT 1 (2026-09-02) : [LoadDir] SYNTHETISE TOUJOURS l'index depuis le nom
// du fichier — `Meta()[i].Index` est le NN de `chunk_NN.bin` a la position `i` — et fusionne le
// manifeste de l'appelant PAR NUMERO, jamais par position (une entree de manifeste dont le fichier
// manque est ignoree ; un fichier hors manifeste garde son index, sans type ni debut). Un
// consommateur ne demande donc plus « combien de chunks ? » mais « quels numeros ? », et lit le
// chunk 0 par son NUMERO :
//
//	le registre        la position dont `Meta[i].Index == 0` — absente sur une bobine partielle ;
//	les donnees        les positions dont `Meta[i].Index >= 1`, dans l'ordre du film.
//
// ARRET AU PREMIER TROU DE NUMEROTATION, cote consommateur (`filmdec.FilmChunkNumbers`) : c'est ce
// que faisait l'ancien `filmdec.CountFilmChunks`, qui s'arretait au premier `chunk_NN.bin`
// manquant, et le lot 1 est un refacto PUR — les sorties doivent etre identiques a l'octet. La
// regle est donc heritee sciemment, pas choisie : elle tombera avec les enveloppes de
// compatibilite, quand plus aucun appelant ne dependra de l'ancien comptage. Ce paquet, lui, ne
// s'arrete a aucun trou : il charge TOUT ce que la source donne (piege du chunk 41/63, cf.
// source.go).
//
// Ce paquet ne connait pas la semantique des chunks au-dela de ce numero : le role de chacun se
// lit dans le manifeste ([ChunkMeta], fourni par l'appelant), et l'interpretation viendra avec
// `FilmContext` (lot 2 du plan).
//
// # POLITIQUE MEMOIRE
//
// Les chunks DECOMPRESSES sont gardes pour la duree de vie du [Film] : c'est le prix a payer pour
// decoder une fois, et c'est deja ce que fait `killsource` aujourd'hui — pics mesures de 48 a
// 256 Mio sur les films sains, sous un plafond enfant de 3 Gio. Les payloads de paquets sont des
// SOUS-TRANCHES de ces buffers, jamais des copies : le cout d'un paquet est son en-tete decode,
// pas ses octets. Corollaire a connaitre : garder un seul `Packet` retient tout le chunk.
//
// Les films dits « bombes » (`51101d1d`, `a349fea8`, `1c4c63c2`, `60ae07c4`) ne le sont PAS par
// leur taille decompressee mais par l'amplification en aval (`objectiveevents.NamedEventsFrom`) :
// ce paquet n'est pas le lieu ou les plafonner (lot 4b du plan).
package filmsource
