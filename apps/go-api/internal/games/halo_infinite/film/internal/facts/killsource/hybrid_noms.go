package killsource

// hybrid_noms.go — LE NOM QU UNE LIGNE PUBLIEE PREND AU ROSTER (lot J7.1, constat FK-2, DT-6).
//
// Deux temps de l hybride publient un nom que le kill-feed ne porte pas : la VICTIME d une mort de
// bot (temps 4) et le TUEUR d une mort infligee par un bot (temps 5). Les deux le prennent au roster
// de replication, par l indice que le dead-state designe. Ce nom passe par [pass.nomPubliable], le
// seul endroit ou une ligne l obtient : un nom de remplissage ([estNomDeRemplissage]) ne sort JAMAIS
// en ligne publiee, et chaque refus se compte ([Stats.NomsDeRemplissageRefuses]).

// autoSurCoupleFabrique : LA SEULE LIGNE QUE LE TEMPS 4 A LE DROIT DE REMPLACER (revue adverse du
// lot J7, constat FK-4). « Un instant publie ne se reecrit jamais » reste la regle ; l exception est
// la plus etroite qui rende la mort de bot verifiee sans publier un kill qui n a pas eu lieu :
//
//	le couple de l instant est FABRIQUE par le recollage (`m.fab`), ET
//	la ligne publiee vient du temps 3 (`OriginSelfSource`) — appariee sur la VICTIME SEULE, son
//	dead-state designe la victime elle-meme : rien dans le film ne confirme le TUEUR du couple.
//
// Une ligne du temps 1 ou 2 sur ce meme couple fabrique, elle, a ete appariee sur le couple ENTIER
// par un dead-state (le film confirme « K tue V ») : elle reste, et la mort de bot est une collision
// comptee. Quand le temps 4 remplace, le couple devient fantome (`pass.fantomes`) : il sort du
// denominateur avec la ligne qui en sort du numerateur, `Covered <= RealPairs` tient.
func autoSurCoupleFabrique(m botMatch, prev Kill) bool {
	return m.fab && prev.Read.Origin == OriginSelfSource
}

// nomPubliable rend le nom que le roster donne a l indice `i`, et faux quand ce nom ne designe
// personne. Le refus est compte ici, une fois par ligne refusee.
func (p *pass) nomPubliable(i int) (string, bool) {
	nom := p.ctx.roster.nameOf(i)
	if estNomDeRemplissage(nom) {
		p.nomsDeRemplissage++
		return "", false
	}
	return nom, true
}
