// Package importstore est le stockage des services d'import (OpenSpartan, import depuis le site) : il prend
// les bases qu'un import écrit (la base partagée sous le fournisseur B-swap, la base du joueur sous son
// bail), y écrit par les chemins existants (persist, helpers de sync) et rend aux services des ports de
// internal/port qui ne montrent aucun type SQL.
//
// Ce paquet n'est pas dans platform/duckdb parce qu'il compose sync et persist, que platform/duckdb
// n'importe pas. Il tourne sous une requête HTTP : le garde-rail « pas de writer partagé sur le chemin d'une
// requête » le lit comme internal/service.
package importstore
