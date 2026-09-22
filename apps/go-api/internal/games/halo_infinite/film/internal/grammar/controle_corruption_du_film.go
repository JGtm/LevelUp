package grammar

// controle_corruption_du_film.go — LE CONTROLE DE CORRUPTION PAR COMPOSANT D UN CONTEXTE DE
// FILM (lot 5.18.2).
//
// FICHIER SEPARE PAR LA TAILLE, PAS PAR LE SENS : `film_context.go` franchissait les 500 lignes
// du depot avec ces methodes ; elles restent celles de [FilmContext], et la REGLE qu elles
// appliquent vit une seule fois, dans [grammaireSousFilm] (`profil_balayage.go`).
//
// CE QUE CE FICHIER PORTE, ET POURQUOI IL EXISTE DU TOUT. Le drapeau ne vit PAS dans
// `FilmContext.bal` : [FilmContext.PoserProfilDeBalayage] remplace le profil ENTIER, et
// `replay.poserProfilPuisCarte` y installe le profil calibre par `killsource` — ou l INVARIANT
// quand le kill-feed n a pas pu se decoder. Un drapeau range dans `bal` serait donc efface par
// ce geste, sans un mot. Il est DERIVE a chaque rendu, depuis le film, et memorise une fois.

// controleDeCorruptionDuFilm rend le bit du film, DERIVE UNE FOIS puis memorise (meme paresse
// que les autres derivations de ce contexte : elle lit `chunk_00`).
func (c *FilmContext) controleDeCorruptionDuFilm() bool {
	if !c.corrLu {
		c.corrLu = true
		var g GrammaireBalayage
		g, c.corrLue = grammaireSousFilm(c.bal.Grammaire, c.Profile())
		c.corr = g.ControleDeCorruption
	}
	return c.corr
}

// ControleDeCorruptionRepli dit que ce film n a PAS declare son controle de corruption par
// composant — il ne porte pas de section d identification (5 films du cache, format 20). C est
// le repli `repli_controle_corruption_section_absente` du registre, et c est son COMPTE : un
// film, un verdict. Faux quand le film a parle, quelle que soit la valeur declaree.
func (c *FilmContext) ControleDeCorruptionRepli() bool {
	if c == nil {
		return true
	}
	c.controleDeCorruptionDuFilm()
	return !c.corrLue
}
