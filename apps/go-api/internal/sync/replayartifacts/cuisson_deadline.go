package replayartifacts

// cuisson_deadline.go — LES BORNES DE TEMPS D'UNE CUISSON : la borne dure d'un film et le plancher de budget.
// Extrait de cuisson.go pour tenir le seuil de taille de fichier ; mêmes déclarations, même paquet.

import "time"

// DeadlineParFilm : la borne DURE de la cuisson d'UN film, quand le budget du cycle en laisse
// autant.
//
// POURQUOI ELLE EXISTE (PLAN_CUISSON_PERF item 5.5). Le budget de cycle s'applique ENTRE deux
// matchs : il ne peut rien contre un enfant qui ne rend jamais la main. Un enfant bloqué —
// spirale GC sous le plafond souple, disque qui ne répond plus, processus suspendu par l'OS —
// tenait donc le cycle de synchronisation INDÉFINIMENT, et avec lui tout ce qui vient après
// (PSA, agrégats, médias). La deadline coupe l'enfant, le film compte en échec, et le cycle
// continue : c'est la même doctrine que le protocole de codes de sortie — la santé de la passe
// ne dépend jamais de la santé d'un film.
//
// QUINZE MINUTES, ET C'EST LARGE À DESSEIN : le film le plus cher du corpus se cuit en moins de
// deux minutes (mesures §6 du plan). Cette borne n'est pas un réglage de performance, c'est un
// dernier rempart — la couper court transformerait un film lent en échec, ce que personne ne
// demande.
const DeadlineParFilm = 15 * time.Minute

// PlancherCuisson : le solde de budget SOUS LEQUEL AUCUNE CUISSON NE SE LANCE.
//
// POURQUOI IL EXISTE (revue du lot 6, constat 6.3). La borne d'un film vaut
// `min(solde, DeadlineParFilm)` et n'avait pas de plancher : un solde de deux secondes — ou nul —
// donnait à la cuisson un contexte DÉJÀ EXPIRÉ. L'enfant était tué aussitôt né, le film comptait
// en ÉCHEC, et un WARN « artefact rejeu non construit » accusait le décodage d'une panne qui n'en
// était pas une. C'est exactement ce que la doctrine de ce fichier interdit : LE BUDGET S'APPLIQUE
// ENTRE DEUX MATCHS, JAMAIS AU MILIEU D'UN — et une cuisson lancée pour être tuée trois lignes
// plus loin, c'est le milieu d'un match. Sous le plancher, le cycle REPORTE (Info, pas Warn : un
// report est nominal) et le match revient au cycle suivant, l'étape étant idempotente.
//
// TRENTE SECONDES, ET C'EST MESURÉ : après le lot 4, un film témoin se cuit en 15 à 19 s
// (`MESURES_CUISSON_PERF.md` §1 : 15,7 / 18,6 / 18,2 s). Trente secondes laissent donc passer la
// cuisson médiane et refusent celles qui n'ont plus la place de finir. Le pire film sain du corpus
// (BTB à 26 joueurs, ~1 min 40) ne rentre dans aucun plancher raisonnable : le reporter au cycle
// suivant EST le bon comportement, et sa borne dure reste [DeadlineParFilm]. La valeur est un
// dixième de [BudgetParCycle] — le cycle ne se prive que de sa dernière tranche.
const PlancherCuisson = 30 * time.Second

// deadlineDuFilm rend la borne effective d'UNE cuisson : le minimum du solde de budget et de
// [DeadlineParFilm]. Rien ne sert de laisser un enfant courir vingt minutes quand le cycle
// s'arrête dans trois.
//
// LE SOLDE REÇU N'EST JAMAIS SOUS [PlancherCuisson] : l'appelant a déjà refusé de lancer la
// cuisson en dessous (cf. `buildAll`). Cette fonction ne rend donc plus jamais un délai
// dérisoire — un contexte expiré à la naissance était un échec compté pour rien.
func deadlineDuFilm(d Deps, restant time.Duration) time.Duration {
	max := DeadlineParFilm
	if d.DeadlineParFilm > 0 {
		max = d.DeadlineParFilm
	}
	if restant < max {
		return restant
	}
	return max
}
