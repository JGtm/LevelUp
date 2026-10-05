package replay

import (
	"cmp"
	"context"
	"log/slog"
	"math"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// flag_film_bases.go — LA BASE ET LE CAMP DE CHAQUE DRAPEAU, LUS DANS LE FILM.
//
// # CE QUE LE FILM DIT DES BASES
//
// Un VOL (`flag_steals`) est une prise AU SOCLE : le porteur arrache le drapeau adverse de sa
// base. Le film porte l'instant du vol (statborg), la position du voleur (sa piste) et son equipe
// (`FlagCarryScan.TeamOf`). D'ou deux lectures directes :
//
//	VARIANTE ORDINAIRE   les vols d'un camp tombent tous au MEME point, et ce point est la base
//	                     du camp ADVERSE — celui qui n'y vole jamais, puisqu'on ne porte pas son
//	                     propre drapeau ;
//	DRAPEAU NEUTRE       les vols des DEUX camps tombent au meme point, le socle du centre.
//
// LA POSITION S'AFFINE PAR L'OBJET. Le moteur repose le drapeau AU POINT de son socle quand il
// rentre (renaissance de la vie libre, cf. [flagHomeExactDist]) : une renaissance repetee a
// portee du centre des vols EST le socle, au centimetre. Sans elle, la base reste au centre des
// vols — un repli nomme ([fallback.NomSocleDuFilmAuCentreDesVols]) : la rentree de l'objet ne s'y
// apparie plus au decimetre.
//
// # LA PLACE DU CATALOGUE
//
// Le catalogue de socles reste la source de la POSITION et du CAMP quand il les donne. Le film
// dit en plus QUELS socles sont en jeu : une carte qui declare des socles d'autres variantes (un
// socle central a camp inconnu, des socles en double) en publierait autant de drapeaux immobiles,
// et la regle « le drapeau adverse » ne saurait plus lequel nommer. Quand le film lit deux bases,
// les socles retenus sont donc ceux qu'il apparie — de preference celui dont le camp concorde
// avec le film, une carte pouvant declarer deux socles de camps opposes au meme point — ; un
// socle apparie sans camp prend celui du film ; une base sans socle a portee est publiee a la
// position lue. Quand le film lit une base NEUTRE et que l'objet n'a pas tranche « neutre », un
// socle du catalogue qui ne nomme aucun camp, a portee, devient le socle unique de la variante
// neutre. L'accord et la contradiction du catalogue avec le film se COMPTENT
// (`coverage.flagCarries.filmBaseAgree` / `filmBaseContradict`) : une contradiction ne change
// rien au catalogue, elle se voit.
//
// # LES SEUILS, ET LA MESURE QUI LES FONDE
//
// Tenus par la mesure sur les films de CTF du cache (instrument
// `drapeau_bases_film_research_test.go`) : dispersion des vols d'un camp sous 1,4 m pour un rayon
// de groupe de 3 m ; renaissance la plus proche du centre des vols sous 0,8 m, et sous 0,008 m du
// socle du catalogue ; centres des deux camps a moins de 0,8 m sur les parties a drapeau neutre,
// pour un seuil neutre de 3 m. La separation de deux bases d'equipe se mesure de 9,2 m (Isolation,
// socles du catalogue a 8,46 m) a 45 m : son seuil, 6 m, ne vient pas de ce releve mais de la
// geometrie des groupes (cf. [flagFilmBaseMinSeparationM]).

const (
	// flagFilmBaseRadiusM : rayon (m) du groupe de vols d'un camp, de la recherche d'une
	// renaissance autour de son centre, de l'appariement a un socle du catalogue et de la
	// coincidence des deux camps (variante neutre). Un vol se declenche quand le porteur touche le
	// socle : les vols d'un camp tiennent en 1,4 m au plus, le rayon leur laisse un facteur deux.
	flagFilmBaseRadiusM = 3.0
	// flagFilmBaseMinShare : part minimale des vols localises d'un camp que son groupe principal
	// doit tenir. Un camp dont les vols se partagent entre deux points (camps qui changent de cote,
	// piste mal nommee) ne donne aucune base.
	flagFilmBaseMinShare = 0.75
	// flagFilmBaseMinSeparationM : distance minimale (m) entre les deux bases d'equipe, LE DOUBLE
	// de [flagFilmBaseRadiusM] : a cette distance, les disques des deux groupes de vols sont
	// disjoints, aucun vol ne peut compter pour les deux camps, et chaque base s'apparie a ses
	// propres socles sans jamais atteindre ceux de l'autre. C'est la plus petite separation que la
	// lecture sait trancher ; les plus proches bases d'equipe connues (Isolation, 8,46 m) passent.
	// Jusqu'a [flagFilmBaseRadiusM] les deux camps volent au meme point (variante neutre) ; entre
	// les deux seuils, ni une base ni deux : rien n'est lu.
	flagFilmBaseMinSeparationM = 2 * flagFilmBaseRadiusM
	// flagFilmRebirthMin : nombre minimal de naissances de l'objet a moins de [flagHomeExactDist]
	// les unes des autres pour faire une RENAISSANCE. Un lacher ne se repete pas au centimetre.
	flagFilmRebirthMin = 2
)

// flagFilmBase est une base lue dans le film.
type flagFilmBase struct {
	// owner est le camp proprietaire, [TeamNeutral] pour le socle de la variante neutre.
	owner int
	x, y  float32
	// rebirth dit que la position est celle d'une renaissance de l'objet (le point du socle) ;
	// faux : c'est le centre des vols.
	rebirth bool
}

// flagFilmReading est ce que le film dit des bases : rien, UNE base neutre, ou DEUX bases
// d'equipe triees par camp. Aucune autre forme n'est produite.
type flagFilmReading struct {
	bases []flagFilmBase
}

func (r flagFilmReading) teams() bool   { return len(r.bases) == 2 }
func (r flagFilmReading) neutral() bool { return len(r.bases) == 1 }

// flagPoint est un point du plan, en metres monde.
type flagPoint struct{ x, y float32 }

// readFlagFilmBases lit les bases dans le film. Il faut exactement DEUX camps qui volent : un
// seul camp qui vole ne distingue pas la variante neutre de la variante ordinaire, et plus de
// deux camps ne se rangent pas en « l'adversaire ».
func readFlagFilmBases(ops []flagOpening, scan FlagCarryScan, ctx flagCarryCtx,
	idx map[string][]Track) flagFilmReading {
	parCamp := flagStealPointsByTeam(ops, scan.TeamOf, ctx, idx)
	if len(parCamp) != 2 {
		return flagFilmReading{}
	}
	camps := make([]int, 0, 2)
	for team := range parCamp {
		camps = append(camps, team)
	}
	slices.Sort(camps)
	var centres [2]flagPoint
	for i, team := range camps {
		c, ok := flagStealCentre(parCamp[team])
		if !ok {
			return flagFilmReading{}
		}
		centres[i] = c
	}
	renaissances := flagRebirthPoints(scan.Free)
	d := math.Sqrt(sqDist(centres[0].x, centres[0].y, centres[1].x, centres[1].y))
	switch {
	case d <= flagFilmBaseRadiusM:
		milieu := flagPoint{(centres[0].x + centres[1].x) / 2, (centres[0].y + centres[1].y) / 2}
		return flagFilmReading{bases: []flagFilmBase{flagRefineBase(TeamNeutral, milieu, renaissances)}}
	case d >= flagFilmBaseMinSeparationM:
		// LES VOLS DU CAMP camps[0] TOMBENT A LA BASE DE camps[1], et reciproquement.
		return flagFilmReading{bases: []flagFilmBase{
			flagRefineBase(camps[0], centres[1], renaissances),
			flagRefineBase(camps[1], centres[0], renaissances),
		}}
	}
	return flagFilmReading{}
}

// flagStealPointsByTeam rend, par camp du voleur, la position de chaque VOL : celle de sa piste
// publiee a la frame de la prise. Un vol dont le voleur n'a ni equipe lue ni position s'ecarte.
func flagStealPointsByTeam(ops []flagOpening, teamOf map[string]int, ctx flagCarryCtx,
	idx map[string][]Track) map[int][]flagPoint {
	out := map[int][]flagPoint{}
	for _, o := range ops {
		if !o.steal {
			continue
		}
		team, ok := teamOf[o.xuid]
		f0 := ctx.frameOfMatchMS(o.t0)
		if !ok || team == TeamNeutral || f0 < 0 || f0 >= ctx.frames {
			continue
		}
		if p, ok := pointOfXUIDAt(idx[o.xuid], f0); ok {
			out[team] = append(out[team], flagPoint{p.X, p.Y})
		}
	}
	return out
}

// flagStealCentre rend le centre du groupe PRINCIPAL d'un nuage de vols — le point qui a le plus
// de voisins a moins de [flagFilmBaseRadiusM], et ses voisins —, et dit si ce groupe tient la
// part [flagFilmBaseMinShare] du nuage. Les ex aequo vont au premier point : le resultat ne
// depend que de l'ordre des vols, qui est celui du temps.
func flagStealCentre(pts []flagPoint) (flagPoint, bool) {
	best, n := flagDensestPoint(pts, flagFilmBaseRadiusM)
	if best < 0 || float64(n) < flagFilmBaseMinShare*float64(len(pts)) {
		return flagPoint{}, false
	}
	return flagMeanNear(pts, pts[best], flagFilmBaseRadiusM), true
}

// flagDensestPoint rend l'index du point qui a le plus de voisins a moins de `r` (lui compris),
// et ce nombre ; (-1, 0) sur un nuage vide.
func flagDensestPoint(pts []flagPoint, r float64) (int, int) {
	best, n := -1, 0
	for i := range pts {
		c := 0
		for j := range pts {
			if sqDist(pts[i].x, pts[i].y, pts[j].x, pts[j].y) <= r*r {
				c++
			}
		}
		if c > n {
			best, n = i, c
		}
	}
	return best, n
}

// flagMeanNear rend le centre des points a moins de `r` de `c`.
func flagMeanNear(pts []flagPoint, c flagPoint, r float64) flagPoint {
	var sx, sy float64
	k := 0
	for _, p := range pts {
		if sqDist(p.x, p.y, c.x, c.y) <= r*r {
			sx, sy, k = sx+float64(p.x), sy+float64(p.y), k+1
		}
	}
	return flagPoint{float32(sx / float64(k)), float32(sy / float64(k))}
}

// flagRebirthPoints rend les points de RENAISSANCE de l'objet drapeau : chaque groupe d'au moins
// [flagFilmRebirthMin] naissances a moins de [flagHomeExactDist] du premier d'entre elles, reduit
// a son centre. L'ordre est celui des vies, donc deterministe.
func flagRebirthPoints(lives []flagFreeLife) []flagPoint {
	nais := make([]flagPoint, 0, len(lives))
	for _, l := range lives {
		if len(l.Pts) == 0 {
			continue
		}
		x, y := l.First()
		nais = append(nais, flagPoint{x, y})
	}
	pris := make([]bool, len(nais))
	var out []flagPoint
	for i := range nais {
		if pris[i] {
			continue
		}
		var groupe []flagPoint
		for j := i; j < len(nais); j++ {
			if !pris[j] && sqDist(nais[i].x, nais[i].y, nais[j].x, nais[j].y) <= flagHomeExactDist*flagHomeExactDist {
				pris[j], groupe = true, append(groupe, nais[j])
			}
		}
		if len(groupe) >= flagFilmRebirthMin {
			out = append(out, flagMeanNear(groupe, nais[i], flagHomeExactDist))
		}
	}
	return out
}

// flagRefineBase pose une base au point de la renaissance la plus proche du centre des vols,
// quand il en est une a moins de [flagFilmBaseRadiusM] ; au centre des vols sinon.
func flagRefineBase(owner int, c flagPoint, renaissances []flagPoint) flagFilmBase {
	best, bd := -1, flagFilmBaseRadiusM*flagFilmBaseRadiusM
	for i, p := range renaissances {
		if d := sqDist(p.x, p.y, c.x, c.y); d <= bd {
			best, bd = i, d
		}
	}
	if best < 0 {
		return flagFilmBase{owner: owner, x: c.x, y: c.y}
	}
	return flagFilmBase{owner: owner, x: renaissances[best].x, y: renaissances[best].y, rebirth: true}
}

// flagResolveSpawns rend le jeu de socles retenu pour ce film : celui du catalogue (variante
// tranchee par l'objet, cf. flag_neutral.go), complete et controle par les bases lues dans le
// film. Il pose les compteurs de la lecture dans la couverture.
//
// LE FILM NE L'EMPORTE QUE LA OU LE CATALOGUE SE TAIT : deux bases lues alors que l'objet a
// tranche « drapeau neutre » se comptent en contradiction et laissent le verdict du catalogue ;
// une base neutre lue sur une carte du catalogue n'en change les socles que si le socle a sa
// portee ne nomme aucun camp (cf. [flagChoiceOfNeutralBase]).
func flagResolveSpawns(scan FlagCarryScan, r flagFilmReading, cov *FlagCarriesCoverage,
	ctx flagCarryCtx) flagSpawnChoice {
	choix := flagChooseSpawns(scan)
	cov.FilmBases = len(r.bases)
	switch {
	case r.teams() && choix.Neutral:
		cov.FilmBaseContradict++
	case r.teams():
		choix = flagChoiceOfTeamBases(scan, choix, r, cov, ctx.fb)
	case r.neutral():
		choix = flagChoiceOfNeutralBase(scan, choix, r.bases[0], cov, ctx.fb)
	}
	logFlagFilmBases(ctx.journal, cov)
	return choix
}

// flagRetained est un socle retenu, avec son rang dans le catalogue (l'ordre de publication des
// drapeaux suit celui du catalogue ; une base sans socle vient apres).
type flagRetained struct {
	spawn FlagSpawn
	rang  int
	// fromFilm : le camp, ou le camp et la position, viennent du film. agree / contradict :
	// le controle d'un camp que le catalogue nomme.
	fromFilm, agree, contradict bool
	// approx : la position est le centre des vols, sans renaissance (repli nomme).
	approx bool
}

// flagChoiceOfTeamBases retient UN socle par base lue : celui du catalogue a portee, sinon la
// base elle-meme. Deux socles retenus au meme camp (catalogue contredit) : le choix du catalogue
// reste, et la contradiction se compte.
func flagChoiceOfTeamBases(scan FlagCarryScan, choix flagSpawnChoice, r flagFilmReading,
	cov *FlagCarriesCoverage, fb *fallback.Compteur) flagSpawnChoice {
	retenus := make([]flagRetained, 0, len(r.bases))
	pris := map[int]bool{}
	for _, b := range r.bases {
		retenus = append(retenus, flagRetainFor(scan.Spawns, b, pris))
	}
	for _, rt := range retenus {
		if rt.agree {
			cov.FilmBaseAgree++
		}
		if rt.contradict {
			cov.FilmBaseContradict++
		}
	}
	if retenus[0].spawn.Team == retenus[1].spawn.Team {
		return choix
	}
	slices.SortStableFunc(retenus, func(a, b flagRetained) int { return cmp.Compare(a.rang, b.rang) })
	spawns := make([]FlagSpawn, 0, len(retenus))
	for _, rt := range retenus {
		if rt.fromFilm {
			cov.SpawnsFromFilm++
		}
		if rt.approx {
			fb.Declenche(fallback.NomSocleDuFilmAuCentreDesVols)
		}
		spawns = append(spawns, rt.spawn)
	}
	choix.Spawns = spawns
	choix.TeamBirths = flagBirthsNear(scan.Free, spawns)
	return choix
}

// flagRetainFor rend le socle retenu pour une base, parmi les socles NON NEUTRES du catalogue a
// moins de [flagFilmBaseRadiusM] et pas deja pris : d'abord un socle dont le camp CONCORDE avec
// celui du film, puis un socle sans camp, puis seulement un socle qui le contredit ; a rang egal,
// le plus proche. Le rang passe avant la distance parce qu'une carte peut declarer deux socles de
// camps opposes au meme point, a quelques millimetres : la proximite n'y dit rien du camp. A
// defaut de socle, la base elle-meme.
func flagRetainFor(catalogue []FlagSpawn, b flagFilmBase, pris map[int]bool) flagRetained {
	best, bestRang, bd := -1, 0, 0.0
	for i, s := range catalogue {
		if s.Neutral || pris[i] {
			continue
		}
		d := sqDist(s.X, s.Y, b.x, b.y)
		if d >= flagFilmBaseRadiusM*flagFilmBaseRadiusM {
			continue
		}
		rang := flagCampRank(s.Team, b.owner)
		if best < 0 || rang < bestRang || (rang == bestRang && d < bd) {
			best, bestRang, bd = i, rang, d
		}
	}
	if best < 0 {
		return flagRetained{spawn: FlagSpawn{Team: b.owner, X: b.x, Y: b.y},
			rang: len(catalogue) + b.owner, fromFilm: true, approx: !b.rebirth}
	}
	pris[best] = true
	s := catalogue[best]
	rt := flagRetained{spawn: s, rang: best}
	switch s.Team {
	case TeamNeutral:
		rt.spawn.Team, rt.fromFilm = b.owner, true
	case b.owner:
		rt.agree = true
	default:
		rt.contradict = true
	}
	return rt
}

// Rangs d'un socle du catalogue face au camp d'une base lue (cf. [flagRetainFor]) : le plus
// petit l'emporte.
const (
	flagRankAgrees = iota
	flagRankNoCamp
	flagRankContradicts
)

// flagCampRank rend le rang d'un socle du camp `team` pour une base du camp `owner`.
func flagCampRank(team, owner int) int {
	switch team {
	case owner:
		return flagRankAgrees
	case TeamNeutral:
		return flagRankNoCamp
	}
	return flagRankContradicts
}

// flagChoiceOfNeutralBase retient les socles d'une partie ou le film lit une base NEUTRE :
//
//   - sans catalogue, la base lue devient le socle unique de la variante ;
//   - l'objet a tranche « neutre » : la base CONTROLE ce verdict (accord quand le socle unique
//     retenu est a sa portee) ;
//   - sinon, un socle du catalogue QUI NE NOMME AUCUN CAMP (sans camp, ou neutre par son label)
//     a portee de la base devient le socle unique de la variante neutre : les deux camps y
//     volent, le film tranche la ou le catalogue se tait ;
//   - sinon, le choix du catalogue reste et la contradiction se compte.
func flagChoiceOfNeutralBase(scan FlagCarryScan, choix flagSpawnChoice, b flagFilmBase,
	cov *FlagCarriesCoverage, fb *fallback.Compteur) flagSpawnChoice {
	if len(scan.Spawns) == 0 {
		if !b.rebirth {
			fb.Declenche(fallback.NomSocleDuFilmAuCentreDesVols)
		}
		spawns := []FlagSpawn{{Team: TeamNeutral, Neutral: true, X: b.x, Y: b.y}}
		cov.SpawnsFromFilm = 1
		return flagSpawnChoice{Spawns: spawns, Neutral: true, NeutralBirths: flagBirthsNear(scan.Free, spawns)}
	}
	if choix.Neutral {
		accord := len(choix.Spawns) == 1 &&
			sqDist(choix.Spawns[0].X, choix.Spawns[0].Y, b.x, b.y) <= flagFilmBaseRadiusM*flagFilmBaseRadiusM
		if accord {
			cov.FilmBaseAgree++
		} else {
			cov.FilmBaseContradict++
		}
		return choix
	}
	i, ok := flagCamplessSpawnNear(scan.Spawns, b)
	if !ok {
		cov.FilmBaseContradict++
		return choix
	}
	s := scan.Spawns[i]
	// Un socle neutre par son label ACCORDE le catalogue au film ; un socle sans camp recoit du
	// film son role.
	if s.Neutral {
		cov.FilmBaseAgree++
	} else {
		cov.SpawnsFromFilm++
	}
	spawns := []FlagSpawn{{Team: TeamNeutral, Neutral: true, X: s.X, Y: s.Y}}
	autres := make([]FlagSpawn, 0, len(scan.Spawns))
	for j, o := range scan.Spawns {
		if j != i && !o.Neutral {
			autres = append(autres, o)
		}
	}
	return flagSpawnChoice{Spawns: spawns, Neutral: true,
		NeutralBirths: flagBirthsNear(scan.Free, spawns), TeamBirths: flagBirthsNear(scan.Free, autres)}
}

// flagCamplessSpawnNear rend l'index du socle du catalogue le plus proche d'une base, a moins de
// [flagFilmBaseRadiusM], parmi ceux qui ne nomment aucun camp : neutres par leur label, ou sans
// camp (`TeamNeutral`).
func flagCamplessSpawnNear(spawns []FlagSpawn, b flagFilmBase) (int, bool) {
	best, bd := -1, flagFilmBaseRadiusM*flagFilmBaseRadiusM
	for i, s := range spawns {
		if !s.Neutral && s.Team != TeamNeutral {
			continue
		}
		if d := sqDist(s.X, s.Y, b.x, b.y); d <= bd {
			best, bd = i, d
		}
	}
	return best, best >= 0
}

// logFlagFilmBases journalise une CONTRADICTION entre le catalogue et le film : c'est le seul
// endroit ou elle se voit en production hors de la couverture.
func logFlagFilmBases(ctx context.Context, cov *FlagCarriesCoverage) {
	if ctx == nil || cov.FilmBaseContradict == 0 {
		return
	}
	slog.WarnContext(ctx, "rejeu : bases du drapeau lues dans le film en contradiction avec le catalogue de socles",
		"bases_film", cov.FilmBases, "accord", cov.FilmBaseAgree, "contradiction", cov.FilmBaseContradict)
}
