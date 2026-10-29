// Package filmfetch est l'exploitation des films Halo Infinite (plan D3, décision D3-D) : il prend
// les octets que rend la brique `internal/haloapi/infinite` (manifeste, blobs COMPRESSÉS du CDN) et
// en fait les fragments de film que LevelUp lit.
//
// Ce qui vit ici et pas dans la brique :
//
//   - le cache disque local ([LocalFilmCache], hérité du projet Python) consulté AVANT l'API ;
//   - la décompression zlib des blobs (`decfilm.Decompresser` : le CDN sert le format de stockage
//     des films, pas un encodage HTTP) ;
//   - la règle « seul un film finalisé se télécharge » (`finalise.Finalise`, temps forts) ;
//   - la version de film lue dans le registre d'un film en cache ;
//   - la sélection des fragments par type et le verdict « film absent » (404/410 = résultat vide) ;
//   - les compteurs expvar du téléchargement (posés sur la brique par ses crochets).
//
// La brique ne lit aucun octet de film ; ce paquet est sous `internal/sync`, racine surveillée par
// `archlint/no_raw_film_bytes_outside_source_test.go`.
package filmfetch
