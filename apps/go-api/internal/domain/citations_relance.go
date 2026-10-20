package domain

// CitationsRelanceIssue : ce que la relance des citations du film d'un match a fait pour UN
// joueur suivi (service citationsrelance).
type CitationsRelanceIssue string

const (
	// CitationsRelanceRecalculee : les citations du match ont été recalculées.
	CitationsRelanceRecalculee CitationsRelanceIssue = "recalculee"
	// CitationsRelanceAJour : rien à faire — pas encore de citations pour ce match (leur premier
	// calcul est au post-sync du joueur) ou citations déjà postérieures aux dérivations du film.
	CitationsRelanceAJour CitationsRelanceIssue = "a_jour"
	// CitationsRelanceJoueurOccupe : la base du joueur est tenue (son post-sync tourne) au-delà
	// de la borne d'attente ; ce post-sync recalculera lui-même.
	CitationsRelanceJoueurOccupe CitationsRelanceIssue = "joueur_occupe"
)
