package replay

// journal_de_publication.go — LES AVERTISSEMENTS QUI DISENT UN DEFAUT DU DOCUMENT PUBLIE.
//
// Le document que [PortagesAuSync] assemble n'est pas publie ([Options.documentInterne]) : il ne
// relit que ses calques de porteur et son calage, et il est assemble sans table du film, sans
// horloge du film ni balayage des capacites. Les avertissements qui disent ce qui manque a un
// document PUBLIE (table du film non employee, origine d'horloge non etablie et calques non recales,
// equipes non lues ou contredites par la base, impulsions et charges non balayees) y diraient a
// chaque synchronisation un defaut qui n'en est pas un. Ils y descendent en Debug ; le chemin publie
// garde son niveau.

import "log/slog"

// niveauDePublication rend le niveau d'un avertissement propre au document publie : WARN, ou Debug
// sur un document interne.
func niveauDePublication(documentInterne bool) slog.Level {
	if documentInterne {
		return slog.LevelDebug
	}
	return slog.LevelWarn
}

// niveauDePublication rend le niveau des avertissements propres au document publie, pour CET
// assemblage.
func (a *assemblage) niveauDePublication() slog.Level {
	return niveauDePublication(a.opt.documentInterne)
}
