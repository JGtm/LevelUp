package replay

// vies_un_echantillon_test.go — L'ORACLE DES VIES D'UN SEUL ECHANTILLON (lot 1.6.5).
//
// # POURQUOI CET INSTRUMENT EXISTE
//
// `DefaultMinPoints` est passe de 2 a 1 le 2026-09-14 (decision utilisateur : « si le film le dit,
// on publie »). L'utilisateur a IMPOSE l'oracle avec la decision : pour chaque vie d'un seul
// echantillon desormais publiee, une mort ECRITE a cet instant pour ce joueur, OU la derniere
// image avant une fin de manche, doit exister ; tout ORPHELIN est un defaut de LECTURE, pas un cas
// a filtrer.
//
// # CE QUE LA MESURE DU 2026-09-14 A RENDU, ET CE QU'ELLE VAUT
//
// 20 vies sur les huit builds (0 a 6 par film) : 1 mort ecrite, 5 fins de film, 14 ORPHELINES.
// Les quatorze se ferment toutes sur un TROU DE REPLICATION (`cause = cut`), et la mort la plus
// proche du meme joueur est a 0,95 s a 300 s — donc il n'y a pas de mort a cet instant. La lecture
// la plus economique est qu'un slot replique une seule image puis se tait plus de 5 s, ce qui
// DECOUPE une vie en deux la ou le film n'en ecrit qu'une. Consigne au plan §4.
//
// LA SOURCE DE MORT EST LE FIL DES MORTS DU FILM (`ScanDeaths`), c'est-a-dire le kill feed. Le
// DEAD-STATE des bipedes n'est pas lisible sur cette branche (le travail `ti=40` vit sur
// `wt/vehicule-deadstate`, non fusionnee) : l'oracle porte donc sur la seule source ecrite
// disponible, et c'est dit ici plutot que tu.
//
// # CE QUE CE TEST VERROUILLE
//
// La table par build, telle qu'elle est MESUREE. Une vie d'un echantillon qui apparait ou
// disparait, ou qui change de classe, fait rougir : c'est le denominateur de la question, et il ne
// doit pas bouger en silence.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// classeDeVie : les trois verdicts de l'oracle, et rien d'autre.
const (
	classeMortEcrite = "mort ecrite"
	classeFinDeFilm  = "fin de film"
	classeOrpheline  = "ORPHELINE"
)

// viesDUnEchantillonAttendues : la mesure d'APRES LA CONVERSION DE LA DECOUPE (lot 1.9.13,
// 2026-09-15), build par build (total, mort ecrite, fin de film, orphelines).
//
// LA MESURE D'AVANT EST GARDEE EN REGARD, parce qu'un « 0 orpheline » ne se lit pas sans les
// quatorze qu'il remplace :
//
//	film        avant {total, mort, finFilm, ORPH}   apres
//	000d5950    {1, 0, 0, 1}                         {0, 0, 0, 0}
//	a521164d    {6, 0, 2, 4}                         {2, 1, 1, 0}
//	60ae07c4    {4, 0, 0, 4}                         {0, 0, 0, 0}
//	11de8353    {3, 0, 1, 2}                         {1, 0, 1, 0}
//	111fa685    {2, 0, 0, 2}                         {0, 0, 0, 0}
//	e5adf7b2    {2, 1, 0, 1}                         {1, 1, 0, 0}
//	bcb6d393    {2, 0, 2, 0}                         {2, 0, 2, 0}
//	fb1a1a72    {0, 0, 0, 0}                         {0, 0, 0, 0}
//	TOTAL       20 vies, 1 mort, 5 fins, 14 ORPH     6 vies, 2 morts, 4 fins, 0 ORPH
//
// LES QUATORZE ORPHELINES ONT DISPARU PAR LEUR CAUSE, PAS PAR UN FILTRE : leur vie ne se ferme
// plus sur un trou de replication (le trou est devenu une LACUNE de la meme vie), si bien que
// l'echantillon isole n'est plus une vie a lui seul — il est le premier point de la vie qui
// continue. Les six qui restent sont celles que le film ferme vraiment : deux a une mort ecrite,
// quatre a la fin du film.
func viesDUnEchantillonAttendues() map[string][4]int {
	return map[string][4]int{
		"000d5950": {0, 0, 0, 0},
		"a521164d": {2, 1, 1, 0},
		"60ae07c4": {0, 0, 0, 0},
		"11de8353": {1, 0, 1, 0},
		"111fa685": {0, 0, 0, 0},
		"e5adf7b2": {1, 1, 0, 0},
		"bcb6d393": {2, 0, 2, 0},
		"fb1a1a72": {0, 0, 0, 0},
	}
}

func TestViesDUnEchantillonOntLeurOracle(t *testing.T) {
	var total [4]int
	for _, b := range goldenBuilds() {
		g, entry := chargerGoldenBuild(t, b)
		opt := g.options()
		opt.Labels = goldenCatalog(t)
		opt.MapQuant = &entry
		doc := BuildFromPositions(b.Short8, "halo_infinite", g.Positions, g.Fire, opt)
		if doc.Coverage != nil && doc.Coverage.Tracks != nil &&
			doc.Coverage.Tracks.RefusedMinPoints != 0 {
			t.Errorf("%s : %d vie(s) encore REFUSEE(S) au seuil par defaut — il vaut 1",
				b.Short8, doc.Coverage.Tracks.RefusedMinPoints)
		}
		got := classerViesDUnEchantillon(t, g, opt, doc)
		for i := range total {
			total[i] += got.comptes[i]
		}
		for _, l := range got.lignes {
			t.Log(l)
		}
		if want := viesDUnEchantillonAttendues()[b.Short8]; got.comptes != want {
			t.Errorf("%s : {total, mort, fin, orphelines} = %v, mesure du 2026-09-14 %v",
				b.Short8, got.comptes, want)
		}
	}
	t.Logf("TOTAL sur les huit builds : %d vie(s) d'un echantillon · %d mort ecrite · %d fin de "+
		"film · %d ORPHELINE(S)", total[0], total[1], total[2], total[3])
	if total != [4]int{6, 2, 4, 0} {
		t.Errorf("total {6, 2, 4, 0} attendu, mesure %v", total)
	}
	if total[3] != 0 {
		t.Errorf("%d vie(s) ORPHELINE(S) : ni mort ecrite, ni fin de manche, ni fin de film. "+
			"Chacune est un DEFAUT DE LECTURE a nommer au plan §4, jamais un cas a filtrer "+
			"(oracle utilisateur du lot 1.6.5, converti au lot 1.9.13)", total[3])
	}
}

// oracleVies porte les lignes lisibles et les quatre comptes d'un build.
type oracleVies struct {
	lignes  []string
	comptes [4]int
}

// classerViesDUnEchantillon classe chaque piste d'UN point contre le fil des morts et la cause de
// fin de sa vie, telle que le registre l'a posee.
func classerViesDUnEchantillon(t *testing.T, g *goldenInputs, opt Options,
	doc ReplayDocument) oracleVies {
	t.Helper()
	reg, clk := registreDuDocument(g, opt, doc)
	out := oracleVies{}
	for i := range doc.Tracks {
		tr := doc.Tracks[i]
		if len(tr.Points) != 1 {
			continue
		}
		cause, finVie := causeEtFinDeLaVie(reg, clk, tr)
		ecart := ecartALaMortLaPlusProche(opt.Deaths, reg.DeathOffsetMS(), tr.XUID, finVie)
		classe := classeOrpheline
		switch {
		case cause == CauseVieMort:
			classe = classeMortEcrite
		case cause == CauseVieFinFilm || tr.StartFrame >= doc.FrameCount-3:
			classe = classeFinDeFilm
		}
		out.comptes[0]++
		switch classe {
		case classeMortEcrite:
			out.comptes[1]++
		case classeFinDeFilm:
			out.comptes[2]++
		default:
			out.comptes[3]++
		}
		out.lignes = append(out.lignes, fmt.Sprintf(
			"VIE | %s | slot=%d frame=%d/%d xuid=%s cause=%s | mort la plus proche : %d ms -> %s",
			doc.MatchID, tr.Slot, tr.StartFrame, doc.FrameCount-1, tr.XUID, cause,
			ecart, classe))
	}
	return out
}

// registreDuDocument reconstruit le registre d'identite du document, avec le MEME axe de frames.
func registreDuDocument(g *goldenInputs, opt Options,
	doc ReplayDocument) (IdentityRegistry, IdentityClock) {
	sorted := append([]filmdec.BipedPosition(nil), g.Positions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TimestampUS < sorted[j].TimestampUS
	})
	var origin uint64
	if len(sorted) > 0 {
		origin = sorted[0].TimestampUS
	}
	clk := IdentityClock{OriginUS: origin, StepUS: uint64(opt.frameIntervalMS()) * 1000,
		FrameCount: doc.FrameCount}
	return BuildIdentityRegistry(IdentityInput{
		Positions: sorted, BipedCreations: opt.BipedCreations, Deaths: opt.Deaths,
		PlayerIndices: opt.PlayerIndices, FilmTable: opt.FilmTable, Bots: opt.Bots,
		Fire: fireRefs(g.Fire), RosterXUIDs: opt.RosterXUIDs, Participants: opt.Participants,
		Clock: clk, MatchID: doc.MatchID,
	}), clk
}

// causeEtFinDeLaVie rend la cause de fin de la vie qui couvre cette piste, et l'instant de sa fin.
func causeEtFinDeLaVie(reg IdentityRegistry, clk IdentityClock, tr Track) (string, int64) {
	for _, l := range reg.own.lives {
		if l.slot != tr.Slot {
			continue
		}
		if clk.frameOf(l.from) <= tr.StartFrame && tr.StartFrame <= clk.frameOf(l.to) {
			return l.cause, l.to
		}
	}
	return "", 0
}

// ecartALaMortLaPlusProche rend, en millisecondes, l'ecart entre la fin de la vie et la mort du
// MEME joueur la plus proche — le chiffre qui dit si l'oracle tient de peu ou pas du tout.
func ecartALaMortLaPlusProche(deaths []Death, offsetMS int64, xuid string, finVieUS int64) int64 {
	meilleur := int64(-1)
	for _, d := range deaths {
		if xuid != "" && fmt.Sprint(d.XUID) != xuid {
			continue
		}
		e := (d.TimeMS+offsetMS)*1000 - finVieUS
		if e < 0 {
			e = -e
		}
		if meilleur < 0 || e < meilleur {
			meilleur = e
		}
	}
	return meilleur / 1000
}
