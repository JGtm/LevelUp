package objectiveevents

// rosterfit.go — L'EFFECTIF DU MATCH TIENT-IL DANS LE STATBORG ? La garde d'effectif du calque
// des ACTIONS d'objectif.
//
// # Pourquoi cette question se pose
//
// Le statborg ne connait que HUIT slots d'entite de joueur — `statSlotMin` a `statSlotMax`,
// pairs, soit 10, 12, ... 24 (cf. l'en-tete des constantes de `statborg.go`). C'est une
// contrainte de FORMAT, pas un reglage : au-dela, le film n'a plus de place pour dire de qui il
// parle. Sur un match a plus de huit joueurs, les compteurs lus sur ces huit slots ne sont pas
// « les huit premiers joueurs » — ils ne correspondent a personne de facon reproductible.
//
// # Ce que ca coutait, mesure
//
// Le calque du PORTAGE du drapeau avait deja sa garde ([FlagFilmSignals.IsFlagFilm]) et se
// taisait sur les films BTB. Le calque des ACTIONS, lui, n'en avait AUCUNE et publiait ses
// comptes tels quels. Audit du 2026-09-10 §6, oracle API `match_objective_stats_latest` :
//
//	`4f77afc1` (BTB:CTF, 18 sieges)   65 prises de drapeau publiees pour 4 reelles
//	`879a4dba` (BTB:CTF, 23 sieges)   9 prises pour 23, 4 vols pour 5, 8 porteurs stoppes
//	                                  pour 10 — un desaccord dans les deux sens
//	`5676a9ba` (BTB:TC, 21 sieges)    23 captures de zone pour 51, 4 securisations pour 9
//
// 129 actions publiees sur ces trois films, dont 65 qui n'ont PAS EU LIEU. Elles sont rendues a
// l'ecran comme n'importe quelle autre.
//
// # La regle, et pourquoi elle est de la meme famille qu'`IsFlagFilm`
//
// Un silence DOCUMENTE vaut mieux qu'un calque faux. La garde refuse le film entier plutot que
// de trier les actions une a une : rien dans le film ne dit LESQUELLES des huit series
// correspondent a un joueur reel — c'est la question meme que le plafond de slots rend
// insoluble. Le refus se COMPTE (`refusedByRoster` a la couverture) et se JOURNALISE : un
// calque muet dont personne ne sait pourquoi il est muet est pire que le calque faux.
//
// CE QU'ELLE NE DIT PAS. Elle ne prouve pas que le plafond de slots CAUSE les comptes faux —
// l'audit range ce lien parmi les hypotheses non fermees (§10). Elle constate que le format ne
// peut pas porter l'effectif, et refuse de publier sur cette base.

// StatPlayerSlots est le nombre de slots d'entite de JOUEUR que le statborg peut porter : les
// slots pairs de `statTeamSlotMax` exclu a `statSlotMax` inclus.
//
// DERIVE DES CONSTANTES DE FORMAT, jamais ecrit en dur : si la bande de slots change, ce compte
// change avec elle.
const StatPlayerSlots = (statSlotMax - statTeamSlotMax) / 2

// RosterFitsStatborg dit si `n` SIEGES tiennent dans les slots du statborg.
//
// # CE QUI SE COMPTE EST UN SIEGE, PAS UNE PERSONNE — et la mesure l'a impose
//
// La premiere ecriture de cette garde comptait les LIGNES de la feuille de match. Elle
// refusait alors CINQ films d'arene parfaitement mesures — `8bc6074f`, `396cfc92`,
// `572e236b`, `4ecdf3e7`, `bf5ced1b` — qui portent 9 ou 10 lignes pour huit sieges : un
// joueur part, un bot tient la place, un remplacant arrive. Trois lignes pour un seul siege.
// Le cout aurait ete de 633 actions EXACTES retirees, dont les 152 de `396cfc92` et les 143
// de `572e236b`, deux films a 1,000 joueur par joueur.
//
// Un SIEGE est occupe par au plus une personne a la fois. Il se compte donc en retirant des
// lignes celles qui n'en ouvrent pas un : les BOTS (ils remplissent une place liberee) et les
// joueurs qui ONT REJOINT EN COURS (ils en prennent une). Mesure sur le parc, 2026-09-11 :
//
//	films d'ARENE   7 ou 8 sieges pour 8 a 10 lignes  (les cinq ci-dessus, plus les 4v4 nets)
//	films BTB       18, 21 et 23 sieges               (`4f77afc1` 36 lignes, `5676a9ba` 31,
//	                                                  `879a4dba` 26)
//
// Les deux populations sont separees d'un facteur superieur a DEUX, et le plafond de format
// (huit slots) tombe exactement a la frontiere haute des arenes.
//
// `n <= 0` rend VRAI : un appelant qui ne connait pas l'effectif (cuisson sans faits de match)
// ne doit pas voir son calque disparaitre — la garde refuse ce qu'elle MESURE, jamais ce
// qu'elle ignore.
func RosterFitsStatborg(n int) bool { return n <= StatPlayerSlots }
