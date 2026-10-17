package port

import "errors"

// tactical_sequence_causes.go — LES CAUSES « DONNEES » D'UNE SEQUENCE ABSENTE (vue D de l'onglet
// Tactique).
//
// DEUX FAMILLES, ET LA FRONTIERE EST DE SECURITE. Un REFUS (`ErrReplayNotAvailable` : match que
// l'appelant n'a pas joue, forme invalide, garde du rejeu, capability) reste indiscernable de
// l'absence d'un rejeu (ADR 0029) : le client n'y lit rien. Une CAUSE ci-dessous n'est rendue
// qu'APRES la verification d'ouvrabilite — l'appelant a joue ce match, le dire ne revele rien — et
// dit pourquoi la sequence manque, pour que l'ecran distingue une fonction absente d'une donnee
// absente. Le handler les sert avec un code machine stable (cf. tactical_sequence.go), que le web
// lit pour ecrire lui-meme le texte.
var (
	// ErrSequenceAnalyseEnCours : le rejeu du match n'est pas encore construit ET l'analyse du film
	// est en cours (meme regle que `analysis_pending_parts`, cause `replay`).
	ErrSequenceAnalyseEnCours = errors.New("sequence: analyse du film en cours")
	// ErrSequenceFilmNonAnalyse : aucun rejeu construit pour ce match et aucune analyse en cours.
	ErrSequenceFilmNonAnalyse = errors.New("sequence: film non analyse, aucun rejeu construit")
	// ErrSequenceFilmSansHorloge : l'analyse du film existe mais ne contient pas l'horloge de la séquence
	// (origine ou echelle de temps absentes).
	ErrSequenceFilmSansHorloge = errors.New("sequence: analyse du film sans horloge")
	// ErrSequenceCarteNonMesuree : la carte du match n'a pas de rayon de radar connu pour sa variante.
	ErrSequenceCarteNonMesuree = errors.New("sequence: carte sans rayon de radar mesure")
	// ErrSequenceSansPosition : le film ne porte pas la position de la victime a l'instant de sa mort.
	ErrSequenceSansPosition = errors.New("sequence: position de la victime absente du film")
)
