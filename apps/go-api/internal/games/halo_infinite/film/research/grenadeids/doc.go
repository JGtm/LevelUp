//go:build research

// Package grenadeids est un INSTRUMENT DE RECHERCHE du lot 3.3 (volet recherche), hors
// production.
//
// LA QUESTION, ET L ORDRE DE PREUVE IMPOSE PAR V17 (plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`
// §1.4). L utilisateur a pose le 2026-09-18 une hypothese a tester AVANT toute liste blanche
// « ancienne » : le nombre et l ordre des types de grenade n ont pas change depuis la sortie du
// jeu, et des identifiants qui changeraient entre builds sont improbables ; ce qui a
// vraisemblablement bouge, c est la GRAMMAIRE — la position lue apres le marqueur, ou le
// marqueur lui-meme. L ordre de preuve est donc :
//
//	(1) les QUATRE identifiants ACTUELS apparaissent-ils AILLEURS dans les films anciens —
//	    a un autre decalage de bits autour de la position de production, ou hors de tout
//	    marqueur ? Si oui a un decalage STABLE, la grammaire a bouge, pas les identifiants ;
//	(2) sinon, ce qui suit le marqueur (la famille de huit valeurs relevee sur les versions
//	    33, 37, 39 et 40) s apparie-t-il aux DECREMENTS UNITAIRES du compteur i22 ?
//	(3) seulement alors, conclure.
//
// CE QUE L INSTRUMENT MESURE, PAR FILM ET EN UNE SEULE LECTURE DU FLUX DELTA :
//
//	passe A  les marqueurs du build, et pour chacun la valeur de 32 bits lue a chaque
//	         decalage d de la fenetre [-F, +F] autour de la position de production
//	         (`marqueur + 24 + d`), comptee quand elle tombe dans la liste blanche actuelle ;
//	passe B  un balayage ABSOLU du flux : toute occurrence d un des quatre identifiants
//	         actuels a n importe quelle position de bit, avec sa distance au marqueur le plus
//	         proche — c est la reponse a « et dans un autre champ ? » ;
//	passe C  l histogramme SANS liste blanche de ce qui suit le marqueur (la famille des huit
//	         valeurs anciennes), avec le champ d index joueur de 5 bits a +103 ;
//	passe D  (option, question 2) l appariement de chaque candidat de la passe C aux
//	         decrements unitaires du compteur i22 du meme instant, rang par rang.
//
// LE MARQUEUR N EST PAS UNE CONSTANTE DE FORMAT, et l instrument en tient compte. C est le
// milieu d un record de CREATION d entite : `[5 bits bas de typeIndex][19 bits d amorce de l
// etat par defaut]` (`.ai/ADDENDUM_ETAT_DE_L_ART_2026-07-26.md` §3, repris par
// `grammar/projectiles.go`). Le `0x4C0C00` de production encode donc `ti=41`, l archetype
// projectile DE CE BUILD. L instrument resout l archetype projectile du film PAR LE NOM de ses
// composants (`projectile-at-rest-state`, `projectile-tether-state`, ...) — jamais par rang,
// regle de l item 3.2.2 — et balaye AUSSI le marqueur derive de ce ti quand il differe.
//
// CE QUE CE N EST PAS : du code de production. Tous les fichiers portent le tag de compilation
// `research` : `go build ./...`, `go vet ./...` et la CI ne les voient pas. Aucun paquet de
// `internal/` ni de `cmd/` ne l importe, et aucun ne doit l importer. L instrument N ASSERTE
// RIEN et NE CORRIGE RIEN : il mesure, et il n ecrit aucun fichier.
//
// COMMENT S EN SERVIR (lecture seule, un film a la fois, aucune ecriture) :
//
//	cd apps/go-api
//	go vet -tags=research ./internal/games/halo_infinite/film/research/...
//	go run -tags=research ./internal/games/halo_infinite/film/research/cmd_grenadeids \
//	  -racine <parc>/data/cache/film_chunks \
//	  -films 111fa685,e5adf7b2,60ae07c4,a349fea8,fb1a1a72
//
// LE VERROU DE DECODAGE N EST PAS PRIS, ET C EST DELIBERE : `filmproc.AcquireSolo` ecrit un
// fichier de verrou sous la racine du cache, ce que le cadre de ce lot interdit (rien d ecrit
// sous `data/`). La serialisation est donc celle de l operateur — la « voie libre » du pilote.
// La sentinelle memoire, elle, est armee par le binaire : elle n ecrit rien.
package grenadeids
