package replay

// tri_total.go — LES TRIS TOTAUX DES CALQUES PUBLIES (lot J10.1, 2026-09-27, DT-9).
//
// Chacun de ces tris ordonne une tranche PUBLIEE (armes au sol, lancers, tirs, portages du crane)
// ou une serie dont le « premier gagne » decide d une sortie (creations d une cle, rentrees de
// drapeau, periodes de lunette, positions d un joueur), et sa cle n etait pas prouvee unique :
// `sort.Slice` y tirait le rang des ex aequo. Chacun porte une chaine `cmp.Or` qui finit sur une
// cle unique, ou, quand aucun champ ne l est, un tri STABLE sur une entree deja dans l ordre du
// film — et le dit. Un test d ex aequo par tri (`tri_total_test.go`) ; le cliquet
// `archlint/film_tri_total_test.go` interdit tout nouvel appel `sort.Slice*` / `sort.Sort`.

import (
	"cmp"
	"slices"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// trierArmesAuSol range les armes au sol publiees dans un ordre TOTAL (lot J10.1, 2026-09-27, DT-9).
// La cle (apparition, arme) ne separait pas deux exemplaires de la meme arme nes a la meme frame —
// deux socles jumeaux, ou deux armes lachees ensemble — et leur rang publie tenait au tri. Le
// departage lit tous les champs publies ; ce qu il ne separe pas garde l ordre total de l entree
// (`gwPickupLess`, par le tri stable).
func trierArmesAuSol(out []GroundWeapon) {
	slices.SortStableFunc(out, func(a, b GroundWeapon) int {
		return cmp.Or(
			cmp.Compare(a.T0, b.T0), strings.Compare(a.W, b.W),
			cmp.Compare(a.X, b.X), cmp.Compare(a.Y, b.Y), cmp.Compare(a.Z, b.Z),
			cmp.Compare(a.T1, b.T1), cmp.Compare(a.T1Max, b.T1Max), strings.Compare(a.End, b.End),
			cmp.Compare(a.Picker, b.Picker), cmp.Compare(a.Dropper, b.Dropper),
			strings.Compare(a.Origin, b.Origin))
	})
}

// trierCreationsParInstant range les creations d une cle par instant, les ex aequo dans l ORDRE DU
// FILM (lot J10.1, 2026-09-27, DT-9). Une cle peut porter DEUX creations au MEME instant (cf.
// [flagFreeLess]) : la fin de vie de chacune est la creation SUIVANTE, donc leur rang decide de
// laquelle recoit une vie nulle. Leur seule cle unique est leur rang de balayage, que le tri
// STABLE conserve. Deux chaines l emploient : les socles ([padObjects]) et le drapeau libre.
func trierCreationsParInstant(list []types.EquipmentCreation) {
	slices.SortStableFunc(list, func(a, b types.EquipmentCreation) int {
		return cmp.Compare(a.TimestampUS, b.TimestampUS)
	})
}

// trierRentrees range les rentrees et retours de drapeau dans un ordre TOTAL (lot J10.1, 2026-09-27,
// DT-9) : instant, drapeau, point de naissance ; deux elements restes ex aequo sont identiques.
// Les deux drapeaux peuvent rentrer a la meme milliseconde, et le fermoir d un portage est le
// PREMIER de la liste qui convient.
func trierRentrees(out []flagHomecoming) {
	slices.SortFunc(out, func(a, b flagHomecoming) int {
		return cmp.Or(cmp.Compare(a.at, b.at), cmp.Compare(a.flag, b.flag),
			cmp.Compare(a.x, b.x), cmp.Compare(a.y, b.y))
	})
}

// trierGrenades range les lancers publies dans un ordre TOTAL : deux lancers tombent souvent sur la
// meme frame de la grille (10 Hz), et un departage arbitraire suffit a changer l artefact. La cle
// s arretait a la position (frame, index, slot, x, y) : deux lancers d un meme joueur a la meme
// position — l un lu par la naissance du projectile, l autre par le bipede — y restaient ex aequo
// en portant un rang de palette, une source ou un projectile differents (lot J10.1, 2026-09-27,
// DT-9). Le departage lit desormais tous les champs publies ; ce qu il ne separe pas est identique.
func trierGrenades(out []Grenade) {
	slices.SortFunc(out, func(a, b Grenade) int {
		return cmp.Or(
			cmp.Compare(a.T, b.T), cmp.Compare(a.Idx, b.Idx), cmp.Compare(a.Slot, b.Slot),
			cmp.Compare(a.X, b.X), cmp.Compare(a.Y, b.Y), cmp.Compare(a.Rank, b.Rank),
			strings.Compare(a.Src, b.Src), cmp.Compare(projDe(a), projDe(b)))
	})
}

// projDe rend l indice du projectile d un lancer, ou -1 quand il n en porte pas.
func projDe(g Grenade) int {
	if g.Proj == nil {
		return -1
	}
	return *g.Proj
}

// trierTirs range les tirs publies par frame, les tirs d une meme frame dans l ORDRE DU FILM (lot
// J10.1, 2026-09-27, DT-9). Une frame de la grille (100 ms) porte souvent plusieurs tirs, parfois
// d un meme tireur au meme point : aucun champ publie ne les separe, leur seule cle unique est leur
// rang dans le flux des evenements de tir, que le tri STABLE conserve.
func trierTirs(out []Shot) {
	slices.SortStableFunc(out, func(a, b Shot) int { return cmp.Compare(a.T, b.T) })
}

// trierPortagesDeCrane range les portages bruts du crane dans un ordre TOTAL (lot J10.1, 2026-09-27,
// DT-9) : debut, manche, porteur, PUIS fin. La liste est batie en iterant deux MAPS (slots, manches) ;
// deux slots rapportes au meme porteur — ou au porteur INCONNU, xuid vide — ouvrent des portages
// au meme instant, et leur rang, qui ordonne les periodes publiees, changeait d une execution a
// l autre. Restes egaux, deux portages sont identiques.
func trierPortagesDeCrane(out []skullRawCarry) {
	slices.SortFunc(out, func(a, b skullRawCarry) int {
		return cmp.Or(cmp.Compare(a.t0MS, b.t0MS), cmp.Compare(a.round, b.round),
			strings.Compare(a.xuid, b.xuid), cmp.Compare(a.t1MS, b.t1MS))
	})
}

// trierPeriodesDeLunette range les periodes d un slot par debut, puis fin, les ex aequo restants dans
// l ORDRE DE LEUR CLOTURE (lot J10.1, 2026-09-27, DT-9). La consultation rend la PREMIERE periode
// qui couvre l instant : deux periodes ouvertes au meme instant (deux bascules dans le meme
// paquet) ne doivent pas se departager au tri.
func trierPeriodesDeLunette(ps []zoomPeriode) {
	slices.SortStableFunc(ps, func(a, b zoomPeriode) int {
		return cmp.Or(cmp.Compare(a.debut, b.debut), cmp.Compare(a.fin, b.fin))
	})
}

// trierPointsParInstant range les positions d un joueur par instant, les ex aequo dans l ORDRE DU FILM
// (lot J10.1, 2026-09-27, DT-9) : deux sieges d un meme joueur echantillonnes a la meme
// milliseconde n ont pas d autre cle unique que leur rang, et le contexte d une mort lit la
// position la plus proche — l une d elles.
func trierPointsParInstant(v []point) {
	slices.SortStableFunc(v, func(a, b point) int { return cmp.Compare(a.tMS, b.tMS) })
}
