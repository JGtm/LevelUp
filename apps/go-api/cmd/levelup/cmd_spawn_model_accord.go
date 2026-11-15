package main

// cmd_spawn_model_accord.go — L'ACCORD DES DEUX DEFINITIONS D'UNE REAPPARITION.
//
// Sur un document qui porte des lectures de fin de vie du film, la decision se prend sur elles
// (source `film`) et le repli sur l'ecart de 4 a 15 s ne s'applique pas. La mesure rejoue le
// meme document SANS ces lectures, ce qui fait decider le repli, et compare les deux verdicts
// piste par piste : ce que le repli valait sur le parc lu avant la montee du decodeur.

import "levelup/go-api/internal/analysis/tactical"

// accordDefinitions compte, sur les documents lus au film, les pistes que les deux definitions
// tiennent pour une reapparition, et celles qu'une seule tient.
type accordDefinitions struct {
	documentsFilm int
	accord        int
	filmSeul      int
	repliSeul     int
	// repliSeulManche : parmi `repliSeul`, les pistes que le film lit comme un debut de manche.
	repliSeulManche int
}

// comparer ajoute un document a la mesure. Un document sans aucune lecture du film n'y entre
// pas : les deux definitions y sont la meme.
func (a *accordDefinitions) comparer(e tactical.EntreeOccupation) {
	if !porteUneLectureDuFilm(e) {
		return
	}
	a.documentsFilm++
	film := tactical.NouvelIndexVies(e).Spawns()
	repli := tactical.NouvelIndexVies(sansLecturesDuFilm(e)).Spawns()
	for i, f := range film {
		r := repli[i]
		switch {
		case f.Reapparition && r.Reapparition:
			a.accord++
		case f.Reapparition:
			a.filmSeul++
		case r.Reapparition:
			a.repliSeul++
			if f.DebutDeManche {
				a.repliSeulManche++
			}
		}
	}
}

// porteUneLectureDuFilm dit si l'entree porte au moins une fin de vie lue ou un debut de compte.
func porteUneLectureDuFilm(e tactical.EntreeOccupation) bool {
	if len(e.DebutsDeCompte) > 0 {
		return true
	}
	for _, p := range e.Pistes {
		if p.Fin != nil {
			return true
		}
	}
	return false
}

// sansLecturesDuFilm rend une copie de l'entree privee de ses lectures du film ; l'entree d'origine
// n'est pas touchee.
func sansLecturesDuFilm(e tactical.EntreeOccupation) tactical.EntreeOccupation {
	nue := e
	nue.DebutsDeCompte = nil
	nue.Pistes = make([]tactical.Piste, len(e.Pistes))
	for i, p := range e.Pistes {
		p.Fin = nil
		nue.Pistes[i] = p
	}
	return nue
}
