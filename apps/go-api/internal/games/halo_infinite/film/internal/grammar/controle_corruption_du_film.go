package grammar

// controle_corruption_du_film.go — LA GRAMMAIRE QU UN FILM DECLARE, DANS SON CONTEXTE : le controle
// de corruption par composant (lot 5.18.2) et la grammaire de la vue A (lot VA).
//
// FICHIER SEPARE PAR LA TAILLE, PAS PAR LE SENS : `film_context.go` franchissait les 500 lignes
// du depot avec ces methodes ; elles restent celles de [FilmContext], et la REGLE qu elles
// appliquent vit une seule fois, dans [grammaireSousFilm] (`profil_balayage.go`).
//
// CE QUE CE FICHIER PORTE, ET POURQUOI IL EXISTE DU TOUT. Ces champs ne vivent PAS dans
// `FilmContext.bal` : [FilmContext.PoserProfilDeBalayage] remplace le profil ENTIER, et
// `replay.poserProfilPuisCarte` y installe le profil calibre par `killsource` — ou l INVARIANT
// quand le kill-feed n a pas pu se decoder. Un champ range dans `bal` serait donc efface par
// ce geste, sans un mot. Ils sont DERIVES a chaque rendu, depuis le film, et memorises une fois.

// grammaireDuFilm est ce que le film declare, derive une fois : son controle de corruption
// ([grammaireSousFilm]) et la grammaire de sa vue A ([grammaireDeLaVueASousFilm]) ; s il porte sa
// section d identification, et si son absence a ete comptee.
type grammaireDuFilm struct {
	controle bool
	vueA     grammaireDeLaVueA
	lue      bool
	derive   bool
	compte   bool
}

// grammaireDuFilmDerivee derive la grammaire du film UNE FOIS puis la memorise (meme paresse que
// les autres derivations de ce contexte : elle lit `chunk_00`).
func (c *FilmContext) grammaireDuFilmDerivee() *grammaireDuFilm {
	if !c.duFilm.derive {
		c.duFilm.derive = true
		var g GrammaireBalayage
		g, c.duFilm.lue = grammaireSousFilm(c.bal.Grammaire, c.Profile())
		c.duFilm.controle = g.ControleDeCorruption
		c.duFilm.vueA = grammaireDeLaVueASousFilm(c.Profile())
	}
	return &c.duFilm
}

// controleDeCorruptionDuFilm rend le bit du film, pour un lecteur qui l EMPLOIE : un film sans
// section d identification le lit a sa valeur de repli, que ce geste compte, UNE fois par film.
func (c *FilmContext) controleDeCorruptionDuFilm() bool {
	d := c.grammaireDuFilmDerivee()
	if !d.lue && !d.compte {
		d.compte = true
		// Repli `repli_controle_corruption_section_absente`, UNE fois par film (lot J8.7).
		c.NoterReplis(ComptesDesReplis{ControleDeCorruptionNonDeclare: 1})
	}
	return d.controle
}

// grammaireDeLaVueA rend la grammaire de la vue A que le film declare (lot VA).
func (c *FilmContext) grammaireDeLaVueA() grammaireDeLaVueA {
	if c == nil {
		return grammaireDeLaVueA{}
	}
	return c.grammaireDuFilmDerivee().vueA
}

// profilDeLaVueA rend le profil de la passe des tetes ([distribuerLesTetes]), qui ne lit que la vue A
// : [FilmContext.ProfilDeBalayage], sans compter le repli `repli_controle_corruption_section_absente`.
// Un film sans section d identification ne declare pas sa table des genres : sa vue A ne se lit pas
// au-dela de sa tete, ou aucun controle de corruption n est lu. La lecture n emploie pas la valeur de
// repli ; elle ne la compte pas.
func (c *FilmContext) profilDeLaVueA() ProfilDeBalayage {
	if c == nil {
		return ProfilDeBalayageParDefaut()
	}
	bal := c.bal
	bal.Grammaire.ControleDeCorruption = c.grammaireDuFilmDerivee().controle
	return bal
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
	return !c.duFilm.lue
}
