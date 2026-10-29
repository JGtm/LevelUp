package grammar

// contexte_de_carte.go — LE CONTEXTE DE CARTE D UNE CUISSON, POSE EN UN SEUL GESTE.
//
// Apres le profil calibre par killsource, la cuisson et la lecture des porteurs au sync posent sur
// le contexte du film les largeurs d axe de la carte du match, puis le decoupage du bloc MPP que la
// grammaire resout pour le film (`replay.poserProfilPuisCarte`). Le ratchet de fermeture d image-cle
// en contexte de cuisson (`keyframe_closure_cuisson_ratchet_test.go`) mesure sous ce meme contexte :
// il appelle ce geste-ci, jamais une copie, et ne peut donc pas mesurer un autre contexte que celui
// de la cuisson. Garde-rail : `archlint/pose_de_carte_unique_test.go` (aucun autre code de
// production ne pose les largeurs de la carte sur un profil, hors une liste fermee et datee).

// PoseDeLaCarte dit ce que [FilmContext.PoserLaCarteEtLeDecoupage] a pose, pour que l appelant
// compte et journalise ce qui ne l a pas ete.
type PoseDeLaCarte struct {
	// LargeursAbsentes : l entree de catalogue de la carte ne porte pas ses trois largeurs d axe ;
	// le profil garde les siennes (le repli est celui de l appelant).
	LargeursAbsentes bool
	// MPP : la resolution du decoupage du bloc MPP ; le contexte ne la porte que si elle decide.
	MPP ResolutionMPP
}

// PoserLaCarteEtLeDecoupage pose sur ce contexte les largeurs d axe de la carte que son profil de
// film porte ([profile.Profile.Map]) — largeurs, index et region de la plage, et `SimStateComplet`
// avec elles ([ProfilDeBalayage.PoserLargeursObjetDuMondeDepuisDecoupage]) —, en verse les replis au
// rapport du contexte, puis pose le decoupage du bloc MPP que [FilmContext.ResolutionMPP] decide. Une
// entree sans largeurs laisse le profil ; un decoupage qui ne decide pas laisse le contexte.
func (c *FilmContext) PoserLaCarteEtLeDecoupage() PoseDeLaCarte {
	var pose PoseDeLaCarte
	e := c.Profile().Map()
	if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {
		pose.LargeursAbsentes = true
	} else {
		bal := c.ProfilDeBalayage()
		c.NoterReplis(bal.PoserLargeursObjetDuMondeDepuisDecoupage(e.Layout()))
		c.PoserProfilDeBalayage(bal)
	}
	pose.MPP = c.ResolutionMPP()
	if pose.MPP.Decide() {
		c.PoserMPP(pose.MPP.Widths)
	}
	return pose
}
