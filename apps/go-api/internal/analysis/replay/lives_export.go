package replay

// lives_export.go — LES VIES DU FILM, TELLES QUE LE NOMMAGE LES A LAISSÉES.
//
// # POURQUOI UNE EXPOSITION ET NON UN SECOND CALCUL
//
// Le collecteur de kills doit écrire les vies en base (`match_lives`, lot 7C). Elles sont
// DÉJÀ calculées : `ResolveSlotXUID` découpe les trajectoires, cale le fil des morts et nomme
// chaque vie — c'est tout le travail de `lives.go` + `owners.go` + `closures.go`. Les
// recalculer ailleurs produirait un second nommage qui divergerait au premier ajustement de
// `deathMatchWindowMS`, et les deux tables du même film ne diraient plus la même chose.
//
// Ce fichier ne fait donc qu'une chose : traduire l'état interne en type public, et convertir
// l'horloge. Aucune décision.

// VieNommee est une vie du film prête à sortir du paquet : une identité, deux instants sur
// l'horloge du MATCH, et la cause de sa fin.
//
// SEULES LES VIES NOMMÉES SORTENT. Une vie anonyme est un corps dont on ne sait pas à qui il
// appartient : l'écrire en base sous un xuid nul fabriquerait un joueur qui n'existe pas.
type VieNommee struct {
	// XUID identifie l'occupant. Jamais nul (les vies anonymes sont écartées).
	XUID uint64
	// DebutMS et FinMS bornent la vie sur l'horloge du MATCH, la même que
	// `match_kill_events.time_ms` — c'est ce qui rend les deux tables joignables.
	DebutMS, FinMS int64
	// Cause dit COMMENT la vie s'est terminée : CauseVieMort, CauseVieFinFilm ou
	// CauseVieCoupure. SEULE `death` dit que le joueur est mort.
	Cause string
	// NomPar dit COMMENT ON SAIT À QUI elle appartient : NomParMort (lecture du fil) ou
	// NomParFermeture (déduction par élimination).
	//
	// LES DEUX AXES SONT ORTHOGONAUX, ET LES SÉPARER EST LA LEÇON DU P0 : un survivant nommé
	// par fermeture porte `NomPar = closure` ET `Cause = film_end`. Les fondre en un seul
	// champ obligeait à choisir entre dire qui il est et dire qu'il a survécu — et le choix
	// fait le comptait mort.
	NomPar string
}

// ViesNommees rend les vies nommées du film, sur l'horloge du MATCH, triées par (début, xuid)
// pour que deux passes du même film écrivent le même ordre.
//
// # LA CONVERSION D'HORLOGE, ET POURQUOI ELLE EST ICI
//
// `lifeSpan` porte des microsecondes de l'horloge du FILM ; `Death.TimeMS` porte des
// millisecondes de l'horloge du MATCH. `bestDeathOffset` a mesuré le décalage entre les deux
// pour pouvoir nommer les vies (`horlogeFilm = horlogeMatch + DeathOffsetMS`) : on l'applique
// à l'envers. Le faire ICI, une fois, plutôt que chez l'appelant, évite qu'un second appelant
// le refasse dans l'autre sens — l'erreur serait un décalage constant, donc invisible à
// l'œil et fatale à toute jointure.
//
// RIEN NE SORT SI LE PONT N'A PAS ÉTÉ CONSTRUIT : sans fil des morts, il n'y a pas de calage,
// donc pas d'horloge de match. Publier des vies calées sur zéro les rendrait joignables avec
// n'importe quoi.
func (r IdentityRegistry) ViesNommees() []VieNommee {
	if r.ViesNommeesParLaLecture() == 0 {
		return nil
	}
	vies := r.Vies()
	out := make([]VieNommee, 0, len(vies))
	for _, l := range vies {
		if l.xuid == 0 {
			continue
		}
		out = append(out, VieNommee{
			XUID:    l.xuid,
			DebutMS: l.from/1000 - r.DeathOffsetMS(),
			FinMS:   l.to/1000 - r.DeathOffsetMS(),
			Cause:   l.cause,
			NomPar:  l.nomPar,
		})
	}
	trierVies(out)
	return out
}

// trierVies impose un ordre TOTAL et déterministe. `buildLifeSpans` parcourt déjà ses slots
// triés, mais l'ordre des vies d'un même instant dépendrait sinon de l'ordre des slots — un
// détail d'implémentation qui ferait bouger les lignes écrites d'une version à l'autre.
func trierVies(v []VieNommee) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && vieAvant(v[j], v[j-1]); j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

func vieAvant(a, b VieNommee) bool {
	if a.DebutMS != b.DebutMS {
		return a.DebutMS < b.DebutMS
	}
	if a.XUID != b.XUID {
		return a.XUID < b.XUID
	}
	return a.FinMS < b.FinMS
}
