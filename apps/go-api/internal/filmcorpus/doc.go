// Package filmcorpus porte les tests du decodeur de film qui ont besoin de LevelUp.
//
// Le decodeur de film (`internal/games/halo_infinite/film`) est une brique : elle n'importe rien
// du depot (ADR 0041, `archlint/briques_imports_test.go`). Un test de la brique qui veut les
// libelles du titre (`internal/games/mappings`), le document servi (`internal/domain/replaydoc`) ou
// n'importe quel autre paquet de l'hote ne peut donc pas rester dans la brique. Il vit ICI,
// cote hote, et n'exerce que la SURFACE EXPORTEE de la brique (plan D2, decision D2-E, regle R1).
//
// # CE QUE CE PAQUET CONTIENT
//
// Des fichiers `_test.go` seuls, rien en production :
//   - les gardes qui confrontent un catalogue de LevelUp (TOML du titre, registre d'armes, document
//     servi) a ce que la brique attend ;
//   - les instruments de recherche (`*_research_test.go`, tag `research`) qui lisent les memes
//     catalogues ; ils gardent leur nom, leur tag et leur suffixe.
//
// Un test deplace ici GARDE SON NOM : les temoins de non-regression le retrouvent sous ce paquet.
//
// # CE QUI N'Y VIENT PAS
//
// Un test qui exerce des identifiants NON exportes de la brique reste dans son paquet (regle R2) et
// se reecrit sur des fixtures locales : la brique ne depend d'aucun chargeur de LevelUp.
package filmcorpus
