// Package facts PORTE LA REVISION DE LA COUCHE DES FAITS, ET RIEN D AUTRE.
//
// La couche `facts` est un ARBRE de paquets (`killsource`, `objectives`, `fallback`) : aucun
// d eux n est « la couche ». La revision, elle, en designe UNE seule — celle que chaque ligne de
// kill porte en base et qui commande le backlog de redecodage. Elle vit donc a la RACINE de
// l arbre, dans un paquet qui ne declare que cela : lui donner du code le rendrait importable
// pour autre chose, et un import de commodite finirait par ramener une dependance dans le
// paquet que tout le monde lit pour une chaine de caracteres.
package facts

// rev.go — LA REVISION DE LA COUCHE DES FAITS, ET SA CHRONIQUE.
//
// # D OU ELLE VIENT (lot 2.6.1, 2026-09-16)
//
// Elle est L HERITIERE DIRECTE de `killcollector.KillSourceDecoderRev` : meme serie, meme
// valeur, meme historique — le fichier qui suit EST celui de la constante d avant, deplace par
// `git mv`, et sa chronique n a pas ete renumerotee (decision V15 (16) du
// PLAN_DECODEUR_FILM_2026-09-13 : M2 est un jalon a ZERO difference de contenu, il n ouvre aucun
// backlog). La constante ne vit plus dans `sync/killcollector` : ce paquet la LIT et l ecrit sur
// chaque ligne produite, il ne la PORTE plus.
//
// POURQUOI CE DEPLACEMENT EST LE CORRECTIF D UN TROU, ET PAS UN RANGEMENT. Tant que la constante
// vivait hors de l arbre hache, un deplacement pur de la constante elle-meme laissait l empreinte
// verte sans regeneration (mesure du 2026-09-16) — et surtout l empreinte ne hachait que
// `killsource/`, alors que la sortie des faits depend aussi d `objectives/`, de `fallback/`, de
// la GRAMMAIRE et de la facon dont les octets sont atteints. Le defaut etait ecrit noir sur blanc
// dans l en-tete d avant (« le gate couvre le decodeur, pas son amont ») : une correction de
// grammaire qui change la sortie de `killsource` sans toucher un octet de `killsource/` ne
// faisait sonner personne, et les lignes deja en base portaient la revision courante — exclues A
// VIE du backlog.
//
// # CE QUE L EMPREINTE HACHE DESORMAIS
//
//	TOUT L ARBRE `film/facts/`   killsource, objectives, fallback ; CE fichier exclu (il DECRIT
//	                             la couche, il n en fait pas partie).
//	LA VALEUR DE `source.Rev`    la porte aux octets (V15 (12)).
//	LA VALEUR DE `grammar.Rev`   la grammaire de lecture — et elle-meme hache les valeurs de
//	                             `profile.Rev` et de `source.Rev` depuis le volet grammaire du
//	                             lot 2.6.1. Les quatre revisions se chainent : la plus basse qui
//	                             monte fait monter toutes celles du dessus.
//
// Hacher des VALEURS amont et pas leurs sources est ce qui rend la regle mecanique : une montee
// d une couche du dessous fait monter les faits sans que personne ait a y penser. C est plus
// strict qu avant, et c est le comportement voulu — un faux positif coute une ligne, un faux
// negatif coute un parc de lignes fausses en base.
//
// # CE QU UNE MONTEE COMMANDE : LE BACKLOG, ET IL PART SUR SIGNAL
//
// Chaque ligne de `match_kill_events` porte dans `decoder_rev` la revision qui l a produite. Une
// montee rend candidates au redecodage toutes les lignes portant une revision anterieure
// (`conditionBacklog`, `sync/killcollector/postsync.go`). Le redecodage du parc est un geste de
// PRODUCTION, reserve au pilote SUR SIGNAL UTILISATEUR (decision D6 du plan), JAMAIS automatique.
//

const Rev = "killsource-2026-09-21.6"

// L EMPREINTE DES SOURCES DE LA COUCHE VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/facts_rev.golden` porte le couple (revision, empreinte) avec son historique, et
// `rev_test.go` le compare aux sources NON-TEST de tout l arbre `film/facts/` et aux valeurs
// amont.
//
// POURQUOI UN GOLDEN ET PLUS UNE CONSTANTE (revue adversariale du 2026-09-12, constat P1-4).
// Tant que le test ne comparait que l EMPREINTE a une constante, remettre la revision ci-dessus a
// sa valeur d avant — en gardant la nouvelle empreinte — restait VERT : le gate ne tenait qu un
// des deux gestes qu il pretendait tenir. Le golden porte les DEUX, et le test distingue les deux
// echecs : « les sources de la couche ont change » et « la revision a change sans la couche ».
