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
// # L'INVARIANT DUR, AVANT TOUTE GEOMETRIE : JAMAIS SON PROPRE DRAPEAU
//
// En CTF on RENVOIE son drapeau, on ne le porte pas — c'est la regle du mode, tranchee par
// l'utilisateur, et elle prime sur toute inference geometrique. Un portage n'est donc JAMAIS
// pose sur le drapeau de l'equipe de son porteur : tout candidat qui y aboutit est REFUSE, et
// les MEMES TROIS REGLES (ci-dessous) se rejouent alors sur les seuls drapeaux ADVERSES. Aucun
// socle adverse : le portage sort NON ATTRIBUE (`flagIndex` = -1, compte en `unresolved`) et
// n'est publie sur aucun drapeau. **On n'invente jamais un drapeau.**
//
// LE REFUS EST UN FILTRE, PAS UN VETO (correctif E2-bis, 2026-09-08). Il se repliait sur « l'autre
// drapeau, s'il est unique » — regle qui ne sait trancher que sur DEUX socles. `084a804d` en
// porte SIX : quatre portages y sortaient non attribues (`unresolved` 0 -> 4, quatre segments de
// portage perdus) alors que la premiere regle — un drapeau adverse gisant au point de prise —
// les resolvait. Sur une carte a deux socles, filtrer rend exactement l'ancien repli.
//
// L'EQUIPE DU PORTEUR NE VIENT PAS DU FILM : elle arrive par `FlagInput.TeamOf`, une table
// xuid -> equipe DEJA RESOLUE par l'appelant, exactement comme le pont d'identite. Table vide
// (CLI hors ligne, ouvrier sans faits) : l'invariant se tait, et le comportement est celui
// d'avant, a l'octet pres.
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
	// retour : un `flag_returns` credite — il ne nomme pas son drapeau, on l'applique au SEUL
	// qui git au sol. home : une RENTREE de l'objet, qui nomme le sien par son socle.
	retour bool
	home   bool
	flag   int
}

// flagGroundTimeline rend les prises et les fins de tous les portages, dans l'ordre du TEMPS.
//
// A INSTANT EGAL, LA FIN PASSE AVANT LA PRISE — le meme ordre qu'`assembleFlagLives`
// ([flagLifeClose] avant [flagLifeOpen]), et pour la meme raison : un drapeau libere dans la
// meme milliseconde se reprend tout de suite, et c'est le cas NOMINAL (`bcb6d393`, la fin d'un
// portage et la prise du suivant tombent toutes deux a 180 531 ms).
func flagGroundTimeline(raws []flagCarryRaw, scan FlagCarryScan,
	ctx flagCarryCtx) []flagGroundEvent {
	out := make([]flagGroundEvent, 0, 2*len(raws))
	for i := range raws {
		out = append(out, flagGroundEvent{at: raws[i].t0, carry: i})
		out = append(out, flagGroundEvent{at: raws[i].t1, close: true, carry: i, flag: -1})
	}
	for _, t := range flagReturnTimes(scan) {
		out = append(out, flagGroundEvent{at: t, retour: true, carry: -1, flag: -1})
	}
	for _, h := range flagObjectHomecomings(scan, ctx) {
		out = append(out, flagGroundEvent{at: h.at, home: true, carry: -1, flag: h.flag})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].at != out[j].at {
			return out[i].at < out[j].at
		}
		return flagGroundRang(out[i]) < flagGroundRang(out[j])
	})
	return out
}

// flagGroundRang ordonne les evenements simultanes, dans le MEME ordre qu'`assembleFlagLives` :
// une fin libere le drapeau, puis un retour ou une rentree le ramene chez lui, puis seulement une
// prise le reclame.
func flagGroundRang(e flagGroundEvent) int {
	switch {
	case e.close:
		return 0
	case e.retour:
		return 1
	case e.home:
		return 2
	default:
		return 3
	}
}

// flagGround est l'etat des drapeaux pendant le parcours : ou chacun GIT, et lequel est EN JEU.
type flagGround struct {
	// sol[f] est la position du drapeau f POSE AU SOL, nil quand il n'y est pas — dans une
	// main, ou chez lui.
	sol []*[2]float32
	// enJeu[f] dit que le drapeau f a quitte son socle et n'y est pas rentre.
	//
	// TROIS FAITS L'Y RAMENENT, et il a fallu la revue DRAPEAUX-R1 pour que les trois soient
	// la : la CAPTURE, le RETOUR CREDITE (`flag_returns`) et la RENTREE DE L'OBJET a son socle.
	// Les deux derniers manquaient, et la note qui l'avouait ne disait que la moitie du
	// probleme (constat C4) : elle affirmait qu'ils « ne peuvent que laisser un drapeau en jeu
	// de trop, ce qui fait TAIRE la troisieme regle ». Vrai pour `enJeu` ; FAUX pour `sol`,
	// qui alimente la regle 2 — prioritaire — et pouvait donc la faire MENTIR, en rattachant
	// une prise a une position de lacher devenue caduque. Un retour remet desormais les DEUX.
	enJeu []bool
	// teamOf est la table xuid -> equipe fournie par l'appelant (`FlagInput.TeamOf`). Vide :
	// l'invariant « jamais son propre drapeau » se tait, faute d'equipe lue.
	teamOf map[string]int
}

// prendre note qu'un drapeau vient d'etre pris : il quitte le sol, et il est en jeu.
func (g *flagGround) prendre(f int) { g.sol[f], g.enJeu[f] = nil, true }

// rentrer ramene un drapeau CHEZ LUI : il n'est plus au sol, et il n'est plus en jeu. Les deux,
// jamais l'un sans l'autre — `sol` perime fait mentir la regle 2, `enJeu` perime fait taire la
// regle 3 (revue DRAPEAUX-R1, constat C4).
func (g *flagGround) rentrer(f int) {
	if f < 0 || f >= len(g.sol) {
		return
	}
	g.sol[f], g.enJeu[f] = nil, false
}

// seulAuSol rend l'UNIQUE drapeau pose au sol, ou -1 quand il y en a zero ou plusieurs.
//
// C'est la meme abstention qu'`applyFlagReturn` en aval, et pour la meme raison : un
// `flag_returns` est credite au joueur qui TOUCHE le drapeau de son equipe, l'evenement ne
// nomme ni l'objet ni le camp. A deux drapeaux au sol, rien ne les departage.
func (g *flagGround) seulAuSol() int {
	seul := -1
	for f, p := range g.sol {
		if p == nil {
			continue
		}
		if seul >= 0 {
			return -1
		}
		seul = f
	}
	return seul
}

// poser applique la FIN d'un portage : une fin CHEZ LUI (capture, ou rentree datee par le retour
// credite / l'objet) renvoie le drapeau a sa base, tout le reste le laisse au sol, a l'endroit du
// lacher.
func (g *flagGround) poser(r flagCarryRaw) {
	f := r.flagIndex
	if f < 0 || f >= len(g.sol) {
		return // fin d'un portage jamais attribue : il n'y a rien a poser.
	}
	if r.endsHome() {
		g.sol[f], g.enJeu[f] = nil, false
		return
	}
	g.sol[f] = &[2]float32{r.x1, r.y1}
}

// seulEnJeu rend l'UNIQUE drapeau en jeu, ou -1 quand il y en a zero ou plusieurs. Se taire a
// deux est la regle du fichier : rien ne departagerait les deux drapeaux.
func (g *flagGround) seulEnJeu(recevable func(int) bool) int {
	seul := -1
	for f, v := range g.enJeu {
		if !v || !drapeauRecevable(recevable, f) {
			continue
		}
		if seul >= 0 {
			return -1
		}
		seul = f
	}
	return seul
}

// drapeauRecevable applique le filtre de candidats, `nil` valant « tous ». Il est ECRIT UNE FOIS
// et partage par les trois regles : trois copies du meme `if recevable != nil` divergeraient.
func drapeauRecevable(recevable func(int) bool, f int) bool {
	return recevable == nil || recevable(f)
}

// choisir applique les trois regles de l'en-tete, dans l'ordre, sur les seuls drapeaux que
// `recevable` accepte (nil = tous). Le second retour dit que la TROISIEME a tranche — une
// attribution PAR ELIMINATION, que la couverture publie.
//
// LE FILTRE EXISTE POUR REJOUER LES MEMES REGLES APRES UN REFUS (correctif E2-bis) : l'invariant
// « jamais son propre drapeau » etait applique en VETO, apres coup, et le repli qui suivait
// (`autreDrapeau`) ne sait trancher que sur DEUX socles. Sur une carte a plus de deux
// (`084a804d` en porte six), un refus condamnait le portage a sortir non attribue alors que la
// premiere regle — un drapeau ADVERSE gisant au point de prise — le resolvait. Filtrer AVANT la
// geometrie ne relache aucune abstention : les trois regles sont les memes, sur moins de
// candidats.
func (g *flagGround) choisir(r flagCarryRaw, spawns []FlagSpawn, recevable func(int) bool) (int, bool) {
	if r.steal {
		return nearestSpawn(spawns, r.x0, r.y0, recevable), false
	}
	if f := nearestDroppedFlag(g.sol, r.x0, r.y0, recevable); f >= 0 {
		return f, false
	}
	if f := g.seulEnJeu(recevable); f >= 0 {
		return f, true
	}
	return nearestSpawn(spawns, r.x0, r.y0, recevable), false
}

// sonPropreDrapeau dit que le drapeau `f` appartient a l'equipe du porteur — ce qu'aucune regle
// du mode n'autorise. Rend faux des qu'une des deux equipes est inconnue : on ne refuse que sur
// une equipe LUE, jamais sur une absence.
func sonPropreDrapeau(spawns []FlagSpawn, f int, equipe int, connue bool) bool {
	if !connue || equipe == TeamNeutral || f < 0 || f >= len(spawns) {
		return false
	}
	return spawns[f].Team == equipe
}

// assignFlags attribue chaque portage a un drapeau (index dans la liste des socles), ou le
// laisse NON ATTRIBUE (`flagIndex` = -1) quand l'invariant refuse tous les candidats.
func assignFlags(raws []flagCarryRaw, scan FlagCarryScan, ctx flagCarryCtx,
	cov *FlagCarriesCoverage) {
	spawns := scan.Spawns
	if len(spawns) == 0 {
		for i := range raws {
			raws[i].flagIndex = 0
		}
		return
	}
	g := &flagGround{
		sol: make([]*[2]float32, len(spawns)), enJeu: make([]bool, len(spawns)),
		teamOf: scan.TeamOf,
	}
	for _, ev := range flagGroundTimeline(raws, scan, ctx) {
		switch {
		case ev.home:
			g.rentrer(ev.flag)
		case ev.retour:
			g.rentrer(g.seulAuSol())
		case ev.close:
			g.poser(raws[ev.carry])
		default:
			g.ouvrir(raws, ev.carry, spawns, cov)
		}
	}
}

// ouvrir attribue UNE prise, sous l'invariant dur, et note ce que la regle a decide.
func (g *flagGround) ouvrir(raws []flagCarryRaw, i int, spawns []FlagSpawn,
	cov *FlagCarriesCoverage) {
	equipe, connue := g.equipeDe(raws[i].xuid)
	f, parElimination := g.choisir(raws[i], spawns, nil)
	if sonPropreDrapeau(spawns, f, equipe, connue) {
		// LA GEOMETRIE A DESIGNE SON PROPRE DRAPEAU : refuse, et les MEMES TROIS REGLES se
		// rejouent sur les seuls drapeaux ADVERSES. `nearestSpawn` peut alors ne rien rendre
		// (aucun socle recevable) — le portage sort non attribue, comme avant.
		cov.OwnFlagRefused++
		f, parElimination = g.choisir(raws[i], spawns, func(k int) bool {
			return !sonPropreDrapeau(spawns, k, equipe, connue)
		})
	}
	raws[i].flagIndex = f
	if f < 0 {
		cov.Unresolved++
		return
	}
	if parElimination {
		cov.AssignedByPlay++
	}
	g.prendre(f)
}

// equipeDe rend l'equipe du porteur, et si elle est connue. Table vide : elle ne l'est jamais,
// et l'invariant se tait.
func (g *flagGround) equipeDe(xuid string) (int, bool) {
	t, ok := g.teamOf[xuid]
	return t, ok
}

// nearestDroppedFlag rend l'index du drapeau LACHE le plus proche du point, ou -1 si aucun n'est
// a portee.
func nearestDroppedFlag(dropped []*[2]float32, x, y float32, recevable func(int) bool) int {
	best, bd := -1, float64(flagPickupRadiusM*flagPickupRadiusM)
	for i, p := range dropped {
		if p == nil || !drapeauRecevable(recevable, i) {
			continue
		}
		if d := sqDist(p[0], p[1], x, y); d <= bd {
			best, bd = i, d
		}
	}
	return best
}

// nearestSpawn rend l'index du socle RECEVABLE le plus proche du point, ou -1 quand aucun socle
// n'est recevable.
func nearestSpawn(spawns []FlagSpawn, x, y float32, recevable func(int) bool) int {
	best, bd := -1, 0.0
	for i := range spawns {
		if !drapeauRecevable(recevable, i) {
			continue
		}
		if d := sqDist(spawns[i].X, spawns[i].Y, x, y); best < 0 || d < bd {
			best, bd = i, d
		}
	}
	return best
}
