package replay

import (
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_carries.go — LA REGLE : de quoi est faite la vie d'un drapeau, et ou elle s'arrete.
//
// # Le principe, en une phrase
//
// On ne decode PAS le drapeau : on lit son PORTEUR. L'objet porte est a la position de celui qui
// le porte, et les evenements de statistique du statborg disent a la milliseconde QUAND le
// portage commence et QUAND il s'arrete.
//
// # Les quatre faits qui FERMENT un portage, et pourquoi le plus petit gagne
//
//	la CAPTURE du porteur       `flag_captures` : le drapeau rentre a sa base
//	sa MORT                     le fil des morts du film : il lache ce qu'il tenait
//	une NOUVELLE prise de lui   il ne peut pas prendre deux fois de suite sans avoir lache
//	la fin du match             borne par defaut
//
// Le LACHER VOLONTAIRE n'est observable par aucune de ces chaines et n'est donc PAS borne : un
// portage qui en contient un est trop long. Le biais joue CONTRE ce qui est affirme (un drapeau
// dessine dans une main qui ne le tient plus), jamais en sa faveur — c'est le sens dans lequel on
// veut se tromper, et le controle du marqueur le mesure.
//
// `flag_carriers_killed` est credite au TUEUR, pas a la victime : il ne nomme donc PAS le porteur
// qui tombe. Il ne sert ici qu'a fermer un portage que le fil des morts aurait manque, et
// SEULEMENT quand exactement UN portage est ouvert a cet instant — sinon rien n'indique lequel,
// et l'evenement se compte en incoherence plutot que de fermer au hasard.
//
// # A quel DRAPEAU un portage appartient
//
// En CTF on ne porte jamais son propre drapeau (le toucher le RENVOIE, c'est `flag_returns`). Un
// portage appartient donc toujours au drapeau adverse — mais « adverse » suppose de connaitre
// l'equipe du porteur, et **l'equipe n'est pas dans le film** (cf. `Track.Team`). L'attribution
// passe donc par la GEOMETRIE, en deux regles qui suivent le comportement de l'objet :
//
//	un VOL (`flag_steals`) se fait AU SOCLE : le drapeau est celui du socle le plus proche ;
//	une PRISE (`flag_grabs`) ramasse un drapeau DEJA au sol : c'est celui qui y est, si l'un
//	  d'eux git a moins de [flagPickupRadiusM] ; sinon on retombe sur le socle le plus proche.
//
// Carte hors du catalogue d'objectifs : aucun socle, tous les portages tombent dans un seul
// drapeau d'equipe [TeamNeutral]. Le calque reste vrai (les portages sont ceux qu'ils sont), il
// est seulement moins detaille — et la couverture publie `Spawns: 0`.

const (
	// flagGrabMergeMS : deux prises separees de moins que cela sont LA MEME action. Un vol
	// incremente son propre compteur (`flag_steals`) EN PLUS de `flag_grabs`, et rien ne garantit
	// que les deux emissions tombent sur la meme milliseconde. Sans cette fusion, la seconde
	// fermerait la fenetre de la premiere a duree quasi nulle, et la mesure perdrait un portage
	// par vol.
	flagGrabMergeMS = 250
	// flagPickupRadiusM : rayon, en metres monde, sous lequel une prise ramasse le drapeau LACHE
	// le plus proche plutot que d'etre attribuee au socle le plus proche. Un drapeau lache ne se
	// deplace pas : le ramasser, c'est etre a sa position — la tolerance ne couvre que le pas de
	// la grille de rejeu (100 ms de course) et le rayon de ramassage du jeu.
	flagPickupRadiusM = 8
)

// FlagSpawn est un socle de drapeau de la carte, en coordonnees monde.
type FlagSpawn struct {
	// Team est l'equipe proprietaire, telle que le fichier de carte la donne.
	Team int
	X, Y float32
}

// FlagCarryScan porte TOUT ce que le film et le catalogue de carte rendent du drapeau. Les
// lectures voyagent ensemble, et `Scanned` dit qu'elles ont abouti : une liste vide sans lui
// serait indistinguable d'un film qu'on n'a pas su lire.
type FlagCarryScan struct {
	Scanned bool
	// Signals porte le verdict de mode et les trois comptes qui le fondent.
	Signals objectiveevents.FlagFilmSignals
	// Events sont les evenements de la table DRAPEAU, PAR SLOT statborg (non identifies).
	//
	// POURQUOI PAS DEJA IDENTIFIES : `IdentifyNamedEvents` ECARTE silencieusement les slots que
	// le pont n'a pas nommes. En partant des evenements bruts, la couverture peut compter
	// exactement combien de prises sont perdues faute de pont (`NoBridge`) — un calque qui ne le
	// dirait pas annoncerait une exhaustivite qu'il n'a pas.
	Events []objectiveevents.NamedEvent
	// Identity est le pont slot statborg -> xuid, PAR MANCHE (le slot est reattribue d'une
	// manche a l'autre ; une prise est nommee par l'identite de sa manche, choisie sur son
	// instant). Sur un film mono-manche c'est le pont plat, a l'octet pres.
	Identity objectiveevents.RoundIdentity
	// Marks est le controle independant : les records de bipede d'image-cle portant le marqueur
	// de portage, plus les instants de TOUTES les images-cles.
	Marks filmdec.CarrierMarkScan
	// Spawns sont les socles `flag_spawn` de la carte (catalogue versionne d'objectifs).
	Spawns []FlagSpawn
	// Free sont les VIES LIBRES de l'objet drapeau (cf. flag_objects.go). Elles ne PUBLIENT rien
	// ici : elles CORRIGENT ce calque — elles DATENT le lacher volontaire, que rien d'autre ne
	// date, et remettent le lacher la ou l'objet repose. Seules celles nees AUX PIEDS D'UN
	// PORTEUR servent : c'est la sous-population que le controle 3 valide. Vides : le calque est
	// exactement celui d'avant le schema 15.
	Free []flagFreeLife
}

// flagCarryCtx regroupe ce que la regle consomme en plus du balayage : l'axe de temps, les
// pistes publiees, le fil des morts et le calage d'horloge film <-> match.
type flagCarryCtx struct {
	// matchClock est l'axe de temps et le calage d'horloge film <-> match, PARTAGES avec le
	// crane, la couronne et la bombe (match_clock.go) — la conversion n'est ecrite qu'une fois.
	matchClock
	// tracks sont les trajectoires PUBLIEES : c'est sur elles que le client dessinera, donc
	// c'est en elles qu'il faut trouver la position du drapeau.
	tracks []Track
	deaths []Death
	// slotXUID nomme le slot de BIPEDE des marques de portage (espace de slots different de
	// celui du statborg).
	slotXUID map[uint32]uint64
}

// flagOpening est une prise, avant tout bornage.
type flagOpening struct {
	slot  int
	xuid  string
	t0    int64 // horloge du MATCH, ms
	steal bool
}

// flagCarryRaw est un portage borne, avant mise en spans.
type flagCarryRaw struct {
	xuid     string
	t0, t1   int64 // horloge du MATCH, ms
	steal    bool
	x0, y0   float32
	x1, y1   float32
	captured bool
	// closed dit qu un FAIT a ferme le portage. Faux : rien ne l a ferme, il court jusqu a la
	// fin du rejeu et aucune transition de fin n est emise.
	closed     bool
	flagIndex  int
	confirmed  bool
	observable bool
}

// buildFlagCarries rend la vie de chaque drapeau et la couverture du calque.
//
// Rend (nil, nil) quand rien n'a ete balaye, et (nil, couverture) quand le film n'est pas reconnu
// comme du CTF : le calque reste vide, et la couverture dit POURQUOI.
func buildFlagCarries(scan FlagCarryScan, ctx flagCarryCtx) ([]FlagCarry, *FlagCarriesCoverage) {
	if !scan.Scanned {
		return nil, nil
	}
	cov := &FlagCarriesCoverage{
		FlagFilm: scan.Signals.IsFlagFilm(), Bursts: scan.Signals.Bursts,
		Captures: scan.Signals.Captures, Steals: scan.Signals.Steals, Spawns: len(scan.Spawns),
	}
	if !cov.FlagFilm {
		return nil, cov
	}
	openings := flagOpenings(scan.Events, scan.Identity)
	cov.Openings = len(openings)
	named := openings[:0:0]
	for _, o := range openings {
		if o.xuid == "" {
			cov.NoBridge++
			continue
		}
		named = append(named, o)
	}
	raws := boundFlagCarries(named, scan.Events, ctx)
	raws, cov.AmbiguousCarrierKills = closeByCarrierKills(raws, scan.Events, scan.Identity)
	// LE LACHER VOLONTAIRE SE FERME ICI, ET AVANT LES POSITIONS : c'est lui qui deplace `t1`,
	// donc le point de lacher que la ligne suivante ira lire sur la piste du porteur.
	raws, cov.ClosedByObject = closeByFreeLives(raws, ctx, scan)
	raws = attachFlagCarryPositions(raws, ctx, cov)
	// ... et le point de lacher se corrige APRES, sur la piste LIBRE : le porteur meurt rarement
	// la ou l'objet se pose. L'attribution du drapeau qui suit s'en sert.
	cov.DropsRepositioned = repositionFlagDrops(raws, ctx, scan)
	assignFlags(raws, scan.Spawns)
	markFlagCarries(raws, scan.Marks, ctx)
	tallyFlagCarries(raws, cov)
	cov.Overlaps, cov.ClosedOverlaps = countFlagOverlaps(raws)
	return assembleFlagLives(raws, scan, ctx, cov), cov
}

// flagOpenings rend les prises de l'oracle, par SLOT statborg, fusionnees et triees.
func flagOpenings(evs []objectiveevents.NamedEvent, identity objectiveevents.RoundIdentity) []flagOpening {
	bySlot := map[int][]flagOpening{}
	for _, e := range evs {
		steal := e.Stat == objectiveevents.StatFlagSteals
		if e.Stat != objectiveevents.StatFlagGrabs && !steal {
			continue
		}
		bySlot[e.Slot] = append(bySlot[e.Slot],
			flagOpening{slot: e.Slot, xuid: identity.At(e.Slot, e.TimeMS), t0: int64(e.TimeMS), steal: steal})
	}
	var out []flagOpening
	for _, ops := range bySlot {
		sort.SliceStable(ops, func(i, j int) bool { return ops[i].t0 < ops[j].t0 })
		for _, o := range ops {
			if n := len(out); n > 0 && out[n-1].slot == o.slot && o.t0-out[n-1].t0 <= flagGrabMergeMS {
				// Un vol l'emporte sur une prise jumelle : c'est lui qui dit d'ou vient le
				// drapeau (du socle), et l'attribution en depend.
				out[n-1].steal = out[n-1].steal || o.steal
				continue
			}
			out = append(out, o)
		}
	}
	sortFlagOpenings(out)
	return out
}

// sortFlagOpenings pose un ordre TOTAL (instant, puis slot) : sans lui, le parcours de map
// rendrait une sortie differente a chaque execution.
func sortFlagOpenings(ops []flagOpening) {
	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].t0 != ops[j].t0 {
			return ops[i].t0 < ops[j].t0
		}
		return ops[i].slot < ops[j].slot
	})
}

// boundFlagCarries ferme chaque prise au PREMIER des faits qui l'interrompent.
func boundFlagCarries(ops []flagOpening, evs []objectiveevents.NamedEvent, ctx flagCarryCtx) []flagCarryRaw {
	captures := timesBySlot(evs, objectiveevents.StatFlagCaptures)
	deaths := deathTimesByXUID(ctx.deaths)
	next := nextOpeningOfSlot(ops)
	end := flagMatchEnd(evs, ctx)
	out := make([]flagCarryRaw, 0, len(ops))
	for i, o := range ops {
		t1, captured, closed := end, false, false
		if c, ok := firstAfter(captures[o.slot], o.t0); ok && c < t1 {
			t1, captured, closed = c, true, true
		}
		if d, ok := firstAfter(deaths[o.xuid], o.t0); ok && d < t1 {
			t1, captured, closed = d, false, true
		}
		if n, ok := next[i]; ok && n < t1 {
			t1, captured, closed = n, false, true
		}
		out = append(out, flagCarryRaw{
			xuid: o.xuid, t0: o.t0, t1: t1, steal: o.steal,
			captured: captured, closed: closed, flagIndex: -1,
		})
	}
	return out
}

// nextOpeningOfSlot rend, par index de prise, l'instant de la prise SUIVANTE du meme slot.
func nextOpeningOfSlot(ops []flagOpening) map[int]int64 {
	last := map[int]int{}
	out := map[int]int64{}
	for i, o := range ops {
		if prev, ok := last[o.slot]; ok {
			out[prev] = o.t0
		}
		last[o.slot] = i
	}
	return out
}

// closeByCarrierKills raccourcit un portage quand `flag_carriers_killed` date une chute que le
// fil des morts n'a pas vue. Ne s'applique QUE si un seul portage est ouvert a cet instant :
// sinon rien ne dit lequel, et l'evenement se compte en incoherence.
func closeByCarrierKills(raws []flagCarryRaw, evs []objectiveevents.NamedEvent,
	identity objectiveevents.RoundIdentity) ([]flagCarryRaw, int) {
	ambiguous := 0
	for _, e := range evs {
		if e.Stat != objectiveevents.StatFlagCarriersKilled {
			continue
		}
		at, killer := int64(e.TimeMS), identity.At(e.Slot, e.TimeMS)
		open, several := -1, false
		for i := range raws {
			if raws[i].t0 >= at || at >= raws[i].t1 || raws[i].xuid == killer {
				continue
			}
			if open >= 0 {
				several = true
				break
			}
			open = i
		}
		switch {
		case several:
			ambiguous++
		case open >= 0:
			raws[open].t1, raws[open].captured = at, false
		}
	}
	return raws, ambiguous
}

// attachFlagCarryPositions pose la position de PRISE et celle de LACHER sur chaque portage, et
// ecarte ce qui n'en a pas. C'est le seul endroit qui rejette apres le pont : les compteurs de
// cause y sont.
func attachFlagCarryPositions(raws []flagCarryRaw, ctx flagCarryCtx, cov *FlagCarriesCoverage) []flagCarryRaw {
	idx := tracksByXUID(ctx.tracks)
	out := raws[:0:0]
	for _, r := range raws {
		f0 := ctx.frameOfMatchMS(r.t0)
		if f0 < 0 || f0 >= ctx.frames {
			cov.OutOfWindow++
			continue
		}
		p0, ok0 := pointOfXUIDAt(idx[r.xuid], f0)
		if !ok0 {
			cov.NoTrack++
			continue
		}
		r.x0, r.y0 = p0.X, p0.Y
		r.x1, r.y1 = p0.X, p0.Y
		if p1, ok1 := pointOfXUIDAt(idx[r.xuid], clampFrame(ctx.frameOfMatchMS(r.t1), ctx.frames)); ok1 {
			r.x1, r.y1 = p1.X, p1.Y
		}
		out = append(out, r)
	}
	return out
}

// tracksByXUID range les pistes publiees par joueur.
func tracksByXUID(tracks []Track) map[string][]Track {
	out := map[string][]Track{}
	for _, t := range tracks {
		if t.XUID != "" {
			out[t.XUID] = append(out[t.XUID], t)
		}
	}
	return out
}

// pointOfXUIDAt rend le point PUBLIE le plus proche de la frame demandee, parmi les pistes d'un
// joueur. Rend (_, false) si aucune piste n'a de point a moins d'une frame — le drapeau n'aurait
// alors pas de position a dessiner, et on prefere ne rien poser.
func pointOfXUIDAt(tracks []Track, frame int) (Point, bool) {
	best, bd, found := Point{}, 0, false
	for _, tr := range tracks {
		for _, p := range tr.Points {
			d := p.T - frame
			if d < 0 {
				d = -d
			}
			if !found || d < bd {
				best, bd, found = p, d, true
			}
		}
	}
	return best, found && bd <= 1
}

// assignFlags attribue chaque portage a un drapeau (index dans la liste des socles).
func assignFlags(raws []flagCarryRaw, spawns []FlagSpawn) {
	if len(spawns) == 0 {
		for i := range raws {
			raws[i].flagIndex = 0
		}
		return
	}
	dropped := make([]*[2]float32, len(spawns)) // position courante de chaque drapeau au sol
	for i := range raws {
		fi := -1
		if !raws[i].steal {
			fi = nearestDroppedFlag(dropped, raws[i].x0, raws[i].y0)
		}
		if fi < 0 {
			fi = nearestSpawn(spawns, raws[i].x0, raws[i].y0)
		}
		raws[i].flagIndex = fi
		if raws[i].captured {
			dropped[fi] = nil
			continue
		}
		dropped[fi] = &[2]float32{raws[i].x1, raws[i].y1}
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

// sqDist rend le carre de la distance plane entre deux points.
func sqDist(ax, ay, bx, by float32) float64 {
	dx, dy := float64(ax-bx), float64(ay-by)
	return dx*dx + dy*dy
}

// timesBySlot rend, par slot statborg, les instants tries d'une statistique.
func timesBySlot(evs []objectiveevents.NamedEvent, stat string) map[int][]int64 {
	out := map[int][]int64{}
	for _, e := range evs {
		if e.Stat == stat {
			out[e.Slot] = append(out[e.Slot], int64(e.TimeMS))
		}
	}
	for s := range out {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i] < out[s][j] })
	}
	return out
}

// deathTimesByXUID rend, par joueur, les instants tries de ses morts.
func deathTimesByXUID(deaths []Death) map[string][]int64 {
	out := map[string][]int64{}
	for _, d := range deaths {
		x := strconv.FormatUint(d.XUID, 10)
		out[x] = append(out[x], d.TimeMS)
	}
	for x := range out {
		sort.Slice(out[x], func(i, j int) bool { return out[x][i] < out[x][j] })
	}
	return out
}

// firstAfter rend le premier instant STRICTEMENT posterieur a t0.
func firstAfter(series []int64, t0 int64) (int64, bool) {
	for _, v := range series {
		if v > t0 {
			return v, true
		}
	}
	return 0, false
}

// flagMatchEnd rend une borne STRICTEMENT posterieure a tout fait date du match ET a la derniere
// frame publiee. Elle sert de fermeture par defaut : un portage qui l atteint n a ete ferme par
// rien, et son `closed` reste faux.
func flagMatchEnd(evs []objectiveevents.NamedEvent, ctx flagCarryCtx) int64 {
	end := ctx.matchMSOfFrame(ctx.frames - 1)
	for _, e := range evs {
		if int64(e.TimeMS) > end {
			end = int64(e.TimeMS)
		}
	}
	for _, d := range ctx.deaths {
		if d.TimeMS > end {
			end = d.TimeMS
		}
	}
	return end + 1
}
