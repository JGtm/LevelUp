// Package replayverite est le BANC DE VERITE du decodeur de film : il juge un artefact de rejeu
// publie contre des ORACLES (les verites officielles du match) et par des COMPTES DE VIOLATIONS
// (ce qu'aucun match reel ne peut contenir), puis compare deux revisions du meme temoin.
//
// Conception : .ai/V7.5/film_re/BANC_DE_VERITE_CONCEPTION_2026-09-30.md (decisions D-1 a D-8).
//
// # Ce qu'il lit, et ce qu'il ne lit pas
//
// Le JSON PUBLIE (la forme que le client lit), les faits du match (`domain.MatchFacts`, ceux-la
// memes qu'a lus la cuisson) et, pour les phases suivantes, l'oracle officiel
// (`domain.MatchOracle`). Aucun octet de film, aucune base, aucun paquet `film/internal`, et PAS le
// type `replay.ReplayDocument` : un oracle ne partage pas les types du code qu'il juge, et le
// ratchet `archlint/film_facade_surface_test.go` compte les identifiants de `film/replay` cites
// hors de `film/`. Les structures de document.go ne portent que les cles lues ; `forme_test.go`
// echoue si l'une d'elles disparait de `film/replay/testdata/document_shape.golden`, et
// `dependances_test.go` si le paquet importe quoi que ce soit du decodeur.
//
// # La circularite
//
// Les faits du match sont une ENTREE de la cuisson : un slot statborg nomme par `triplet_feuille`
// l'a ete PARCE QUE son K/D/A egale l'API, un camp rattache par `teamIdentity` `a` ou `a0` l'a ete
// PARCE QUE son score final egale `teamScores`. Le banc lit la METHODE publiee et compte ces unites
// en `Exclus`, jamais en vrais positifs.
//
// # Le verdict
//
// Il ne juge que l'AVANT/APRES sur un meme temoin, avec les memes faits : les valeurs absolues
// (le statborg lu pour 8 joueurs seulement en BTB, les portes de carte, les builds anciens)
// s'affichent a titre d'information et ne decident rien. `FAUX` : un faux positif d'oracle ou une
// classe de violation en hausse. `MANQUE` : un faux negatif d'oracle en hausse ou une preuve interne
// en baisse. Sinon `ok`.
package replayverite
