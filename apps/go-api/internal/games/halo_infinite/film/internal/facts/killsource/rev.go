package killsource

// rev.go — LA REVISION DE LA SORTIE KILLSOURCE, ET SA CHRONIQUE.
//
// # D OU ELLE VIENT
//
// Elle est L HERITIERE DIRECTE de `facts.Rev` (lot 2.6.1, 2026-09-16), elle-meme heritiere de
// `killcollector.KillSourceDecoderRev` : meme serie, meme valeur, meme historique — la chronique
// qui suit (`rev_chronique.go`, `rev_chronique_archive.go`) est celle de la constante d avant,
// deplacee sans renumerotation.
//
// ELLE N EST PLUS LA REVISION DE TOUT L ARBRE `facts/` DEPUIS LE LOT J3.3 (2026-09-26,
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (c)). `facts.Rev` hachait `killsource/`,
// `objectives/` et `fallback/` : une correction d objectifs faisait monter la revision que porte
// chaque ligne de kill en base, donc rouvrait le backlog killsource pour une sortie que
// `killsource` ne produit pas (l audit du 2026-09-24 le mesure : `killsource` n importe ni
// `objectives` ni `fallback`). Il y a desormais UNE REVISION PAR CONSOMMATEUR DE FAITS : celle-ci
// pour la sortie du kill-feed, `objectives.Rev` pour les objectifs et le statborg.
//
// LA VALEUR EST GARDEE A L IDENTIQUE, et c est la condition du lot : les lignes deja en base
// portent `killsource-2026-09-24` dans `decoder_rev`, et en changer la valeur les rendrait toutes
// candidates au backlog pour un changement d OUTILLAGE.
//
// # CE QUE L EMPREINTE HACHE
//
// Le PERIMETRE de la couche : la fermeture des imports de production de ce paquet (lot J3.2),
// figee par `testdata/killsource_perimetre.golden` — ses sources, les paquets qu il importe hors
// couche (`film/types`, `film/damagetag` et ses tables embarquees, `domain/highlightevent`) — et
// les VALEURS des couches qu il importe : `source.Rev`, `profile.Rev`, `grammar.Rev`. Ce fichier et
// la chronique sont EXCLUS : ils DECRIVENT la couche, ils n en font pas partie.
//
// # CE QU UNE MONTEE COMMANDE : LE BACKLOG, ET IL PART SUR SIGNAL
//
// Chaque ligne de `match_kill_events` porte dans `decoder_rev` la revision qui l a produite. Une
// montee rend candidates au redecodage toutes les lignes portant une revision anterieure
// (`conditionBacklog`, `sync/killcollector/postsync.go`). Le redecodage du parc est un geste de
// PRODUCTION, reserve au pilote SUR SIGNAL UTILISATEUR (decision D6 du plan), JAMAIS automatique.
//
// # LA FORME
//
// `killsource-AAAA-MM-JJ[.N]`, `N >= 2`. Le prefixe est `killsource` depuis toujours : la serie
// est reprise sans renumerotation (V15 (16)).

// Rev est la revision de la sortie killsource.
const Rev = "killsource-2026-09-24"

// L EMPREINTE DE LA COUCHE VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/killsource_rev.golden` porte le couple (revision, empreinte) avec son historique — c est
// le golden de `facts.Rev`, deplace sans renumerotation — et `rev_test.go` le compare au perimetre.
//
// POURQUOI UN GOLDEN ET PAS UNE CONSTANTE (revue adversariale du 2026-09-12, constat P1-4) : tant
// que le test ne comparait que l EMPREINTE, remettre la revision a sa valeur d avant — en gardant
// la nouvelle empreinte — restait VERT. Le golden porte les DEUX.
