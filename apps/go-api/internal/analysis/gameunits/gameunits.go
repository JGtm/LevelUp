// Package gameunits — les unités de distance du jeu.
//
// Les positions du film, de la base et de l'API sont en UNITÉS DU JEU (« wu » de Slipspace). Une
// unité vaut 10 pieds, soit 3,048 m. Une valeur nommée `_m` qui sort d'une mesure du film est donc
// en unités, pas en mètres ; seules les portées annoncées par le jeu (le radar : 18 m en arène,
// 24 m en Grande bataille, `regulation.toml [radar_range_m]`) sont de vrais mètres, et se
// ramènent en unités par [MetersToUnits] avant toute comparaison avec une distance du film.
//
// SOURCE UNIQUE du facteur côté Go (garde-rail `archlint/gameunits_factor_test.go`) ; son jumeau
// web est `apps/web/src/lib/formatters/distance.ts`, tenu à la même valeur par ce même test.
package gameunits

// MetersPerUnit : les mètres d'une unité du jeu (10 pieds).
const MetersPerUnit = 3.048

// UnitsToMeters convertit une distance en unités du jeu en mètres.
func UnitsToMeters(units float64) float64 { return units * MetersPerUnit }

// MetersToUnits convertit une distance en vrais mètres en unités du jeu.
func MetersToUnits(meters float64) float64 { return meters / MetersPerUnit }
