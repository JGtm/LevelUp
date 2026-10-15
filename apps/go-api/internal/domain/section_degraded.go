package domain

import "errors"

// SectionDegradedError signale qu'une lecture a servi sa donnée PRINCIPALE mais qu'une section
// secondaire a échoué (ex. le tableau des scores est rendu, les stats d'objectif ont échoué).
// Le lecteur renvoie la donnée valide ET cette erreur : l'appelant garde la donnée, mais doit
// dire la panne (vue match : code `section_unavailable`) au lieu de la taire.
type SectionDegradedError struct {
	// Section nomme la section en panne, pour le journal (jamais exposée au contrat).
	Section string
	Err     error
}

func (e *SectionDegradedError) Error() string {
	return "section " + e.Section + " dégradée: " + e.Err.Error()
}

func (e *SectionDegradedError) Unwrap() error { return e.Err }

// IsSectionDegraded dit si err est (ou enveloppe) une SectionDegradedError.
func IsSectionDegraded(err error) bool {
	var d *SectionDegradedError
	return errors.As(err, &d)
}
