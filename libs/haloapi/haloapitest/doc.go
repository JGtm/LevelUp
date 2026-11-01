// Package haloapitest fournit l'enregistreur de requêtes des témoins « URL figées » : un
// http.RoundTripper qui note, pour chaque requête, la méthode, l'URL absolue et les en-têtes triés,
// et rend une réponse choisie par le test. Il sert aux clients de la brique haloapi et à leurs
// adaptateurs dans LevelUp.
//
// Ce paquet n'est jamais importé par du code de production : seuls des fichiers _test.go
// l'importent. Il n'utilise que la bibliothèque standard et ne construit ni ne détient aucun client
// HTTP.
package haloapitest
