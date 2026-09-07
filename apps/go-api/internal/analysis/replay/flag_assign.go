package replay

import "sort"

// flag_assign.go — A QUEL DRAPEAU UN PORTAGE APPARTIENT.
//
// # Le film ne le dit pas, et il faut partir de la
//
// Un portage est un compteur de statistique du statborg (`flag_grabs` / `flag_steals`) sur un
// SLOT, plus la trajectoire de son porteur. L'OBJET drapeau n'y figure jamais : il ne se lit que
// LIBRE (`flag_objects.go`), et meme alors il ne porte que son type et sa vie, jamais l'equipe
// qui le possede. L'EQUIPE DU PORTEUR n'est pas dans le film non plus (cf. `Track.Team`). Le
// rattachement est donc GEOMETRIQUE, et l'etiquette d'equipe vient du catalogue de carte
// (`flag_spawn.team_index`, `replaybuild/flagspawns.go`), par le socle retenu.
//
// # LES TROIS REGLES, DANS CET ORDRE
//
//	un VOL (`flag_steals`) se fait AU SOCLE      le drapeau du socle le plus proche ;
//	une PRISE (`flag_grabs`) ramasse un drapeau  celui qui git a moins de [flagPickupRadiusM] du
//	  DEJA AU SOL                                  point de prise ;
//	une PRISE ne peut porter que sur un          ce drapeau-la, quand il est le SEUL en jeu.
//	  drapeau DEJA EN JEU
//
// # LA TROISIEME REGLE MANQUAIT, ET SON ABSENCE FAISAIT PORTER SON PROPRE DRAPEAU
//
// Le repli sur le socle le plus proche est juste pour un VOL — il se fait a un socle par
// definition. Il est FAUX pour une PRISE, qui se fait la ou l'objet est tombe, souvent pres du
// socle ADVERSE : c'est-a-dire pres du socle du camp qui va marquer, donc pres du socle du
// PORTEUR. Mesure sur `bcb6d393` (CTF:Arena, 3-0, les quatre porteurs de l'equipe 0) : la 16e
// prise, ramassee a 10,4 m du socle de l'equipe 0 et a 41,1 m de celui de l'equipe 1, tombait
// sur le drapeau de l'equipe 0 — ce qu'aucune regle du mode n'autorise (on RENVOIE son drapeau,
// on ne le porte pas). Elle emportait avec elle la CAPTURE qui la ferme, publiee sur le mauvais
// drapeau.
//
// # ET UNE ERREUR D'ORDRE, QUI EST LA CAUSE PREMIERE
//
// L'etat du sol se tenait a jour EN PARCOURANT LES PRISES : a chaque portage attribue, sa
// position de LACHER etait aussitot inscrite comme « position courante du drapeau au sol ».
// Deux portages qui se recouvrent suffisent alors a inscrire un lacher qui n'a pas encore eu
// lieu. Sur `bcb6d393` le portage ouvert a 171 941 ms ne se ferme qu'a 325 913 ms — 145 s APRES
// la prise suivante — et sa position de lacher (-6,53 · -2,19) chassait du sol la position reelle
// (33,61 · 2,48) que la prise venait chercher a 0 m. Le parcours se fait donc desormais par
// EVENEMENTS DATES : une fin pose le drapeau, une prise l'enleve, dans l'ordre du temps.

// flagGroundEvent est un changement d'etat du SOL : une prise enleve un drapeau, une fin l'y
// repose — ou le renvoie a sa base quand cette fin est une capture.
type flagGroundEvent struct {
	at    int64
	close bool
	carry int
}

// flagGroundTimeline rend les prises et les fins de tous les portages, dans l'ordre du TEMPS.
//
// A INSTANT EGAL, LA FIN PASSE AVANT LA PRISE — le meme ordre qu'`assembleFlagLives`
// ([flagLifeClose] avant [flagLifeOpen]), et pour la meme raison : un drapeau libere dans la
// meme milliseconde se reprend tout de suite, et c'est le cas NOMINAL (`bcb6d393`, la fin d'un
// portage et la prise du suivant tombent toutes deux a 180 531 ms).
func flagGroundTimeline(raws []flagCarryRaw) []flagGroundEvent {
	out := make([]flagGroundEvent, 0, 2*len(raws))
	for i := range raws {
		out = append(out, flagGroundEvent{at: raws[i].t0, carry: i})
		out = append(out, flagGroundEvent{at: raws[i].t1, close: true, carry: i})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].at != out[j].at {
			return out[i].at < out[j].at
		}
		return out[i].close && !out[j].close
	})
	return out
}

// flagGround est l'etat des drapeaux pendant le parcours : ou chacun GIT, et lequel est EN JEU.
type flagGround struct {
	// sol[f] est la position du drapeau f POSE AU SOL, nil quand il n'y est pas — dans une
	// main, ou chez lui.
	sol []*[2]float32
	// enJeu[f] dit que le drapeau f a quitte son socle et n'y est pas rentre. Une CAPTURE l'y
	// ramene ; un lacher ne l'y ramene pas. Les retours credites et les rentrees d'objet ne
	// sont pas connus ici (ils vivent dans `assembleFlagLives`, en aval) : ils ne peuvent que
	// laisser un drapeau « en jeu » de trop, ce qui fait taire la troisieme regle au lieu de
	// la faire mentir.
	enJeu []bool
}

// prendre note qu'un drapeau vient d'etre pris : il quitte le sol, et il est en jeu.
func (g *flagGround) prendre(f int) { g.sol[f], g.enJeu[f] = nil, true }

// poser applique la FIN d'un portage : une capture renvoie le drapeau chez lui, tout le reste le
// laisse au sol, a l'endroit du lacher.
func (g *flagGround) poser(r flagCarryRaw) {
	f := r.flagIndex
	if f < 0 || f >= len(g.sol) {
		return // fin d'un portage jamais attribue : il n'y a rien a poser.
	}
	if r.captured {
		g.sol[f], g.enJeu[f] = nil, false
		return
	}
	g.sol[f] = &[2]float32{r.x1, r.y1}
}

// seulEnJeu rend l'UNIQUE drapeau en jeu, ou -1 quand il y en a zero ou plusieurs. Se taire a
// deux est la regle du fichier : rien ne departagerait les deux drapeaux.
func (g *flagGround) seulEnJeu() int {
	seul := -1
	for f, v := range g.enJeu {
		if !v {
			continue
		}
		if seul >= 0 {
			return -1
		}
		seul = f
	}
	return seul
}

// choisir applique les trois regles de l'en-tete, dans l'ordre. Le second retour dit que la
// TROISIEME a tranche — une attribution PAR ELIMINATION, que la couverture publie.
func (g *flagGround) choisir(r flagCarryRaw, spawns []FlagSpawn) (int, bool) {
	if r.steal {
		return nearestSpawn(spawns, r.x0, r.y0), false
	}
	if f := nearestDroppedFlag(g.sol, r.x0, r.y0); f >= 0 {
		return f, false
	}
	if f := g.seulEnJeu(); f >= 0 {
		return f, true
	}
	return nearestSpawn(spawns, r.x0, r.y0), false
}

// assignFlags attribue chaque portage a un drapeau (index dans la liste des socles).
func assignFlags(raws []flagCarryRaw, spawns []FlagSpawn, cov *FlagCarriesCoverage) {
	if len(spawns) == 0 {
		for i := range raws {
			raws[i].flagIndex = 0
		}
		return
	}
	g := &flagGround{sol: make([]*[2]float32, len(spawns)), enJeu: make([]bool, len(spawns))}
	for _, ev := range flagGroundTimeline(raws) {
		if ev.close {
			g.poser(raws[ev.carry])
			continue
		}
		f, parElimination := g.choisir(raws[ev.carry], spawns)
		if parElimination {
			cov.AssignedByPlay++
		}
		raws[ev.carry].flagIndex = f
		g.prendre(f)
	}
}

// nearestDroppedFlag rend l'index du drapeau LACHE le plus proche du point, ou -1 si aucun n'est
// a portee.
func nearestDroppedFlag(dropped []*[2]float32, x, y float32) int {
	best, bd := -1, float64(flagPickupRadiusM*flagPickupRadiusM)
	for i, p := range dropped {
		if p == nil {
			continue
		}
		if d := sqDist(p[0], p[1], x, y); d <= bd {
			best, bd = i, d
		}
	}
	return best
}

// nearestSpawn rend l'index du socle le plus proche du point.
func nearestSpawn(spawns []FlagSpawn, x, y float32) int {
	best, bd := 0, sqDist(spawns[0].X, spawns[0].Y, x, y)
	for i := 1; i < len(spawns); i++ {
		if d := sqDist(spawns[i].X, spawns[i].Y, x, y); d < bd {
			best, bd = i, d
		}
	}
	return best
}
