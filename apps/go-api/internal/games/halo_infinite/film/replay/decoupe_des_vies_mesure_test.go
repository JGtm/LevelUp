package replay

// decoupe_des_vies_mesure_test.go — CE QUE LE SEUIL `lifeGapUS` DECIDE, MESURE COUPURE PAR
// COUPURE (lot 1.9.13).
//
// # LA QUESTION
//
// Une vie de joueur finit a une MORT ECRITE, a une FIN DE MANCHE ou a la FIN DU FILM ; un trou
// de replication n'est pas une mort (D13 : la grammaire prime, l'heuristique n'est qu'un repli).
// Aujourd'hui la decoupe est faite par un SEUIL : au-dela de `lifeGapUS` (5 s) sans position, la
// vie courante se ferme et une neuve s'ouvre. Cet instrument compte, pour chaque coupure que ce
// seuil decide, ce que le film ECRIT dans la fenetre du trou :
//
//	mort ecrite    le fil des morts porte une mort du MEME joueur dans la fenetre du trou
//	apparition     un record de CREATION de bipede pour ce slot y tombe : le film ecrit qu'un
//	               corps NEUF commence ici (cf. identity_registry_creation.go, mesure du lot E2 :
//	               aucun record de creation ne tombe DANS l'intervalle d'une vie, il la precede)
//	fin de manche  une frontiere de manche (`objectiveevents.RoundBounds.Starts`) y tombe
//	fin de film    le trou atteint le dernier instant replique du film
//	RIEN           le film ne dit RIEN — la vie est coupee A TORT, et c'est la population que
//	               le lot 1.9.13 convertit en LACUNE d'une seule et meme vie
//
// # LA SOURCE DE MORT
//
// Le FIL DES MORTS du film (`ScanDeaths`), c'est-a-dire le kill feed — la meme que l'oracle des
// vies d'un seul echantillon (`vies_un_echantillon_test.go`). Le DEAD-STATE des bipedes (`ti=40`)
// n'est pas lisible sur cette branche : le travail vit sur `wt/vehicule-deadstate`, non fusionnee.
// Dit ici plutot que tu.
//
// # CE QUE CE TEST VERROUILLE
//
// La table par build, telle qu'elle est MESUREE. C'est le denominateur de la conversion : une
// coupure qui change de classe doit se voir.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// Les quatre classes d'une coupure, et rien d'autre.
const (
	coupureMortEcrite  = "mort ecrite"
	coupureApparition  = "apparition ecrite"
	coupureFinDeManche = "fin de manche"
	coupureFinDeFilm   = "fin de film"
	coupureSansRien    = "RIEN"
)

// coupuresAttendues : la mesure d'APRES LA CONVERSION (lot 1.9.13, 2026-09-15), build par build
// (coupures, mort ecrite, apparition ecrite, fin de manche, fin de film, RIEN).
//
// LA MESURE D'AVANT, LE MEME JOUR, EST CELLE QUI A JUSTIFIE LE LOT — elle est gardee ici parce
// qu'un « 0 RIEN » ne se lit pas sans le nombre qu'il remplace :
//
//	film        avant {coupures, mort, apparition, manche, finFilm, RIEN}   apres
//	000d5950    { 6, 0, 0, 0, 0,  6}                                        { 0, 0, 0, 0, 0, 0}
//	a521164d    {57, 2, 0, 0, 0, 55}                                        { 2, 2, 0, 0, 0, 0}
//	60ae07c4    { 8, 0, 0, 0, 0,  8}                                        { 0, 0, 0, 0, 0, 0}
//	11de8353    {64, 0, 0, 0, 0, 64}                                        { 0, 0, 0, 0, 0, 0}
//	111fa685    {33, 0, 0, 0, 0, 33}                                        { 0, 0, 0, 0, 0, 0}
//	e5adf7b2    {40, 2, 0, 0, 0, 38}                                        { 2, 2, 0, 0, 0, 0}
//	bcb6d393    { 1, 0, 0, 0, 0,  1}                                        { 0, 0, 0, 0, 0, 0}
//	fb1a1a72    { 3, 0, 0, 0, 0,  3}                                        { 0, 0, 0, 0, 0, 0}
//	TOTAL       212 coupures, 4 mortes, 208 RIEN                            4 coupures, 4 mortes, 0 RIEN
//
// Les 208 coupures que RIEN ne justifiait sont devenues des LACUNES de la meme vie. Les quatre
// qui restent sont TOUTES appariees a une mort ecrite du joueur : c'est la definition meme d'une
// fin de vie.
func coupuresAttendues() map[string][6]int {
	return map[string][6]int{
		"000d5950": {0, 0, 0, 0, 0, 0},
		"a521164d": {2, 2, 0, 0, 0, 0},
		"60ae07c4": {0, 0, 0, 0, 0, 0},
		"11de8353": {0, 0, 0, 0, 0, 0},
		"111fa685": {0, 0, 0, 0, 0, 0},
		"e5adf7b2": {2, 2, 0, 0, 0, 0},
		"bcb6d393": {0, 0, 0, 0, 0, 0},
		"fb1a1a72": {0, 0, 0, 0, 0, 0},
	}
}

func TestCoupuresDeVieOntLeurCause(t *testing.T) {
	var total [6]int
	for _, b := range goldenBuilds() {
		g, entry := chargerGoldenBuild(t, b)
		opt := g.options()
		opt.Labels = goldenCatalog(t)
		opt.MapQuant = &entry
		doc := BuildFromPositions(b.Short8, "halo_infinite", g.Positions, g.Fire, opt)
		reg, _ := registreDuDocument(g, opt, doc)
		got := classerLesCoupures(reg, opt)
		for i := range total {
			total[i] += got.comptes[i]
		}
		for _, l := range got.lignes {
			t.Log(l)
		}
		t.Logf("COUPURES | %s | total=%d mort=%d apparition=%d manche=%d finFilm=%d RIEN=%d | "+
			"bornesDeManchePosees=%d RIENdontJoueurSansAucuneMort=%d",
			b.Short8, got.comptes[0], got.comptes[1], got.comptes[2], got.comptes[3],
			got.comptes[4], got.comptes[5], got.bornesDeManche, got.sansAucuneMort)
		if want := coupuresAttendues()[b.Short8]; got.comptes != want {
			t.Errorf("%s : {coupures, mort, apparition, manche, finFilm, RIEN} = %v, "+
				"mesure du 2026-09-15 %v", b.Short8, got.comptes, want)
		}
	}
	t.Logf("TOTAL sur les huit builds : %d coupure(s) · %d mort ecrite · %d apparition ecrite · "+
		"%d fin de manche · %d fin de film · %d RIEN",
		total[0], total[1], total[2], total[3], total[4], total[5])
	if total[5] != 0 {
		t.Errorf("%d coupure(s) de vie que le film ne justifie PAR RIEN — une vie finit a une "+
			"mort ECRITE, a une apparition, a une fin de manche ou a la fin du film (D13)", total[5])
	}
}

// bilanCoupures porte les lignes lisibles des coupures SANS CAUSE et les cinq comptes d'un build.
type bilanCoupures struct {
	lignes  []string
	comptes [6]int
	// bornesDeManche : combien de frontieres de manche le film porte. Sans ce chiffre, un
	// « 0 fin de manche » se lirait « aucune coupure ne tombe sur une manche » alors qu'il peut
	// dire « ce film n'a aucune borne de manche posable » — deux faits differents.
	bornesDeManche int
	// sansAucuneMort : parmi les coupures SANS CAUSE, celles dont l'occupant n'a AUCUNE mort
	// ecrite de tout le film. C'est exactement la population du repli
	// `repli_vie_coupee_au_trou_de_replication` apres conversion.
	sansAucuneMort int
}

// classerLesCoupures classe chaque coupure decidee par `lifeGapUS` contre ce que le film ecrit
// dans la fenetre du trou.
func classerLesCoupures(reg IdentityRegistry, opt Options) bilanCoupures {
	lives := reg.Vies()
	offUS := reg.DeathOffsetMS() * 1000
	manches := manchesEnFilmUS(scoreRecordsOf(opt.Score), reg.DeathOffsetMS())
	finDuFilm := int64(0)
	for _, l := range lives {
		finDuFilm = maxI64(finDuFilm, l.to)
	}
	out := bilanCoupures{bornesDeManche: len(manches)}
	for _, slot := range slotsDesVies(lives) {
		suite := viesDuSlot(lives, slot)
		for i := 0; i+1 < len(suite); i++ {
			cur, next := suite[i], suite[i+1]
			classe := classeDeLaCoupure(cur, next, coupureEntrees{
				deaths: opt.Deaths, creations: opt.BipedCreations, offUS: offUS,
				manches: manches, finDuFilm: finDuFilm,
			})
			out.comptes[0]++
			switch classe {
			case coupureMortEcrite:
				out.comptes[1]++
			case coupureApparition:
				out.comptes[2]++
			case coupureFinDeManche:
				out.comptes[3]++
			case coupureFinDeFilm:
				out.comptes[4]++
			default:
				out.comptes[5]++
				morts := mortsDuJoueur(opt.Deaths, cur.xuid)
				if morts <= 0 {
					out.sansAucuneMort++
				}
				out.lignes = append(out.lignes, fmt.Sprintf(
					"COUPURE SANS CAUSE | slot=%d | vie [%d, %d] -> [%d, %d] | trou %d ms | "+
						"xuid=%d mortsDuJoueur=%d",
					slot, cur.from, cur.to, next.from, next.to,
					(next.from-cur.to)/1000, cur.xuid, morts))
			}
		}
	}
	return out
}

// coupureEntrees porte ce que le film ECRIT autour d'une coupure. Une structure plutot que six
// parametres : le depot en borne cinq.
type coupureEntrees struct {
	deaths    []Death
	creations []filmdec.BipedCreation
	offUS     int64
	manches   []int64
	finDuFilm int64
}

// classeDeLaCoupure rend ce que le film ECRIT dans la fenetre du trou, dans l'ordre de la
// grammaire : une mort d'abord, une apparition ensuite, une fin de manche, la fin du film enfin.
func classeDeLaCoupure(cur, next lifeSpan, in coupureEntrees) string {
	// La fenetre du trou, elargie a gauche de la tolerance d'appariement du fil des morts
	// (`deathMatchWindowMS`) : une mort tombe a l'instant de la DERNIERE position repliquee, pas
	// apres elle.
	de, a := cur.to-deathMatchWindowMS*1000, next.from
	for _, d := range in.deaths {
		if cur.xuid != 0 && d.XUID != cur.xuid {
			continue
		}
		if t := d.TimeMS*1000 + in.offUS; t >= de && t <= a {
			return coupureMortEcrite
		}
	}
	for _, c := range in.creations {
		if c.Slot == cur.slot && int64(c.TimestampUS) > cur.to && int64(c.TimestampUS) <= a {
			return coupureApparition
		}
	}
	for _, m := range in.manches {
		if m >= de && m <= a {
			return coupureFinDeManche
		}
	}
	if cur.to >= in.finDuFilm {
		return coupureFinDeFilm
	}
	return coupureSansRien
}

// mortsDuJoueur compte les morts ECRITES d'un joueur sur tout le film — le chiffre qui dit si la
// coupure est convertible (le joueur meurt, donc ses vies se bornent) ou si elle releve du repli.
func mortsDuJoueur(deaths []Death, xuid uint64) int {
	if xuid == 0 {
		return -1
	}
	n := 0
	for _, d := range deaths {
		if d.XUID == xuid {
			n++
		}
	}
	return n
}

// slotsDesVies rend les slots portant au moins une vie, tries — l'ordre d'une map ne se publie
// pas.
func slotsDesVies(lives []lifeSpan) []uint32 {
	vus := map[uint32]bool{}
	var out []uint32
	for _, l := range lives {
		if !vus[l.slot] {
			vus[l.slot] = true
			out = append(out, l.slot)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// viesDuSlot rend les vies d'un slot dans l'ordre chronologique.
func viesDuSlot(lives []lifeSpan, slot uint32) []lifeSpan {
	var out []lifeSpan
	for _, l := range lives {
		if l.slot == slot {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].from < out[j].from })
	return out
}
