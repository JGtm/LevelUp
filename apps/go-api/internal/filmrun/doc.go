// Package filmrun — L'EXECUTION D'UNE BOUCLE SUR DES FILMS : UN FILM, UN PROCESSUS BORNE.
//
// Ce paquet decide QUAND et COMMENT un decodage de film tourne sur la machine ; le decodeur
// (`libs/film`) ne le decide pas (ADR 0034). Il vient du paquet `filmproc` de la brique, coupe en
// deux : la sentinelle memoire est restee dans la brique (`libs/film/memguard`), l'exploitation est
// ici.
//
//   - LE LANCEUR ([NewRunner], [NewRunnerRoots], [Runner.Run]) : le parent re-execute son propre
//     binaire sur un seul film, sequentiellement, et traduit le code de sortie de l'enfant en
//     [Issue]. La RAM d'un film est rendue a l'OS par construction, et un film qui fait exploser
//     sa memoire n'emporte que lui. L'enfant nait sans console (`platform/childproc`) et en
//     priorite CPU basse, avec les racines du parent imposees dans son environnement ([Roots]).
//   - LE VERROU « UN DECODAGE A LA FOIS SUR CETTE MACHINE » ([AcquireSolo], [AcquireSoloWait],
//     [ReadDecodeHolder]) : verrou du noyau sur `film_decode.lock` sous la racine du cache film,
//     detenteur decrit a cote. Les tests et instruments de la brique prennent le meme verrou par
//     leur propre copie du protocole (`libs/film/internal/decodelock`, parite tenue par
//     `archlint/film_verrou_decodage_parite_test.go`).
//   - LA PRIORITE BASSE du processus courant ([LowerOwnPriority]), pour un outil d'operateur qui
//     decode lui-meme.
//
// Il faut les trois ensemble — un processus par film, un plafond dur par processus
// (`memguard.Arm`), une priorite basse — pour que la machine de travail reste utilisable pendant
// une boucle de mesure.
package filmrun
