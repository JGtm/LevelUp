package port

// FilmBacklog est l'arriéré de films d'un titre à la fin de la dernière passe des étapes film du
// post-sync : matchs que la passe a laissés derrière elle et qu'une passe suivante peut traiter.
type FilmBacklog struct {
	// Frags : matchs dont la source des frags (arme et assistance par mort) reste à lire dans le film.
	Frags int
	// Rejeux : matchs dont l'artefact de rejeu reste à construire.
	Rejeux int
	// Ecrits : films que les passes film du titre ont écrits depuis le démarrage (source des frags lue,
	// film archivé ou artefact de rejeu construit), cumul jamais remis à zéro. Ce n'est pas de l'arriéré :
	// Total et Pending l'ignorent ; l'ordonnanceur compare deux lectures pour savoir si les films avancent.
	Ecrits int
}

// Total est le nombre de films en retard, toutes étapes confondues.
func (b FilmBacklog) Total() int { return b.Frags + b.Rejeux }

// Pending dit s'il reste un arriéré de films.
func (b FilmBacklog) Pending() bool { return b.Total() > 0 }

// FilmBacklogReader lit l'arriéré de films d'un titre, sans écrire. Un titre qui n'a pas d'étape film
// (ou dont aucune passe n'a encore tourné) rend un arriéré vide.
type FilmBacklogReader interface {
	FilmBacklog(titleSlug string) FilmBacklog
}
