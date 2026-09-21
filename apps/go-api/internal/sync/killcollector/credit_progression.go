package killcollector

// credit_progression.go — LA PASSE CREDIT DIT OU ELLE EN EST (lot 5.12, 2026-09-21).
//
// # LE DEFAUT, ET CE QU IL A COUTE
//
// La passe des FILMS journalise chaque match (debut, fin, duree, resultat) : on voit vivre les
// 1 598 decodages. La passe CREDIT, elle, n annoncait que son total (« 9 144 matchs a examiner »)
// puis ne disait plus rien jusqu au bilan. Le 2026-09-21 elle a tourne PLUS DE 22 HEURES sans une
// ligne : impossible de distinguer « lente » de « bloquee », impossible d estimer une fin,
// impossible de decider de l interrompre. Le cout du silence est la, entier : on a attendu.
//
// # CE QUE CE FICHIER POSE
//
// Un pas de progression PAR COMPTEUR, pas par horloge : une ligne tous les
// `pasDeProgressionCredit` matchs examines. Le choix n est pas cosmetique — un cadencement par
// duree (« une ligne toutes les 30 s ») ne se teste qu avec une horloge injectee ou une attente,
// alors qu un cadencement par compteur se teste par une table de valeurs. Et il porte la meme
// information : le rythme des lignes EST le debit.
//
// L ETA est LINEAIRE et annoncee comme telle. Un match de la passe credit coute ~10 ms
// (`credit_cost_integration_test.go`) et ce cout ne depend pas du match : l extrapolation
// lineaire est ici honnete, la ou elle mentirait sur la passe des films (un film de BTB coute
// dix fois un film d arene).

import "time"

// pasDeProgressionCredit : une ligne de progression tous les 500 matchs examines.
//
// A ~10 ms par match, c est une ligne toutes les ~5 s sur une passe de 9 144 matchs — assez pour
// voir le debit, assez peu pour ne pas noyer le journal (18 lignes, le dernier match n en etant
// jamais un — c est le bilan qui l annonce).
const pasDeProgressionCredit = 500

// doitJournaliserProgression : ce match est-il un jalon ?
//
// Le dernier match N EN EST JAMAIS UN, meme s il tombe sur le pas : le bilan final l annonce
// deja, et deux lignes identiques a une seconde d intervalle se lisent comme un bug.
func doitJournaliserProgression(examines, total int) bool {
	if examines <= 0 || examines >= total {
		return false
	}
	return examines%pasDeProgressionCredit == 0
}

// etaLineaire : le temps restant estime, extrapole du debit observe.
//
// Rend zero quand il n y a rien a extrapoler (aucun match examine, ou passe finie) — un appelant
// qui journaliserait un ETA de zero dit « c est fini », ce qui est vrai dans les deux cas.
func etaLineaire(examines, total int, ecoule time.Duration) time.Duration {
	if examines <= 0 || examines >= total || ecoule <= 0 {
		return 0
	}
	return time.Duration(float64(ecoule) / float64(examines) * float64(total-examines))
}
