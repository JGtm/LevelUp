// Package profile porte LA TABLE DE PROFIL DU DECODEUR DE FILM : tout ce qui, dans la lecture
// d un film, ne se lit PAS dans le flux de bits (ADR 0034 D-1 et D-3).
//
// # LA COUCHE, ET SON SENS UNIQUE
//
//	source -> profile -> grammar -> facts -> replay
//
// `profile` n importe QUE `source`. Il ne lit AUCUN octet de film : c est la DONNEE, pas le
// lecteur. Le mouvement inverse — un balayage qui PRODUIT une valeur de profil en lisant le
// film (detection du decoupage i0, calibration du bloc MPP, controle de la table des joueurs) —
// vit en `grammar` et rend un type de CE paquet. C est precisement l inversion de dependance
// que le lot 2.5.b a faite : extraire la donnee sans emmener le lecteur.
//
// # CE QUE LA COUCHE PORTE
//
//	les TYPES DE VALEUR   I0Layout, MPPWidths, PrecisionDescriptor, Vec3Range / AxisRange,
//	                      FilmIdentity — des structures sans logique de lecture.
//	la TABLE              profile_table.go (les cles que le film ECRIT : format, build,
//	                      majeure, plus les invariants), build_profile.go (ce qui varie d un
//	                      build ou d un format a l autre), avec UNE PROVENANCE PAR LIGNE.
//	le CATALOGUE          map_bounds.go : les bornes de dequantification par carte, chargees
//	                      depuis `data/titles/halo_infinite/reference/` (jamais ecrites a
//	                      l execution).
//	le PROFIL RESOLU      Profile et [Resoudre] : la composition des trois cles DEJA LUES et de
//	                      l entree de catalogue de la carte, en une valeur immuable.
//
// # CE QU ELLE NE PORTE PAS, ET POURQUOI
//
// La RESOLUTION DEPUIS UN FILM (`grammar.ResolveProfile`) reste en `grammar` : elle ouvre le
// `chunk_00`, lit le registre, la version de format et la section d identification — trois
// lectures d octets. Elle appelle [Resoudre] avec ce qu elle a lu. Le profil ne sait pas ouvrir
// un film, et c est la propriete qui garde la couche honnete.
package profile
