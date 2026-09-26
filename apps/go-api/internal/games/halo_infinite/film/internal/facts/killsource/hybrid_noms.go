package killsource

// hybrid_noms.go — LE NOM QU UNE LIGNE PUBLIEE PREND AU ROSTER (lot J7.1, constat FK-2, DT-6).
//
// Deux temps de l hybride publient un nom que le kill-feed ne porte pas : la VICTIME d une mort de
// bot (temps 4) et le TUEUR d une mort infligee par un bot (temps 5). Les deux le prennent au roster
// de replication, par l indice que le dead-state designe. Ce nom passe par [pass.nomPubliable], le
// seul endroit ou une ligne l obtient : un nom de remplissage ([estNomDeRemplissage]) ne sort JAMAIS
// en ligne publiee, et chaque refus se compte ([Stats.NomsDeRemplissageRefuses]).

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
