package grammar

// player_teams_test.go — CE QUE LA LECTURE DE L'EQUIPE DOIT TENIR (lot 1.7.1).
//
// Aucune garde d'environnement : ces tests jouent sur les sept bobines par build versionnees
// (`replay/testdata/minifilm_*`), donc en CI.
//
//	E-LUE     Les sept bobines rendent une table d'equipes, sans record inatteint et sans
//	          divergence — ni par entite, ni par index.
//	E-INDEX   L'INDEX QUE LE FILM ECRIT EST LE RANG DU SIEGE. La suite des index des entites du
//	          PREMIER paquet d'image-cle vaut, terme a terme, la suite des `FilmIndex` de la
//	          table des joueurs de `chunk_00` (lot 1.5). C'est la mesure qui remplace
//	          l'appariement ORDINAL de `NOTE_EQUIPE_FILM_2026-09-12.md` par une lecture.
//	E-DOM     Tout designateur publie tient dans `-1..8` (`mp_team_designator` + « aucune »).
//	E-TEMOIN  Le meme champ relu a UN BIT de la position que la grammaire designe ne rend PAS
//	          la meme table : la lecture est ancree, pas chanceuse.
//	E-COUPE   Un payload TRONQUE ne fait paniquer aucun record, ne rend aucune lecture partielle,
//	          et le refus se compte.
//	E-REFUS   Un film sans registre, ou dont l'i0 de ti=9 n'est pas le designateur, est REFUSE
//	          avec sa cause — jamais lu au composant voisin.

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobineEquipes : ce que la lecture doit rendre sur chaque bobine par build.
//
// Les comptes sont ceux MESURES le 2026-09-14 par ce lecteur sur les bobines versionnees. Ils
// ne sont pas des seuils : une bobine qui en rendrait d'autres a change de grammaire ou de
// contenu, et les deux doivent rougir.
type bobineEquipes struct {
	film    string
	entites int // entites ti=9 distinctes (par slot de replication)
	indices int // index de joueur publies
	horsIdx int // records dont l'index sort de la table de 32
	// nonAtteint : records dont la marche n'atteint PAS i0 parce que leur mot de taille `n2`
	// vaut 0 — le jeu n'y ecrit aucun composant (FUN_142e2bfd0, la garde `if (0 < (int)uVar7)`
	// devant `vtable[0x88]`, relue le 2026-09-15). Ce ne sont pas des lectures manquees : ce sont
	// des records sans composants, et les lire etait lire du bruit.
	nonAtteint int
	equipes    []int
	sieges     int // sieges occupes de la table de `chunk_00` (lot 1.5)
	premiers   int // entites du PREMIER paquet d'image-cle porteur
}

// bobinesEquipes rend les sept bobines par build et la lecture attendue.
//
// `111fa685` PORTE LE SEUL RECORD HORS DOMAINE DU CORPUS, et il est ECRIT ICI plutot que taire :
// un record ti=9 y annonce l index 59, hors de la table de 32. Il est REFUSE et COMPTE ; sans
// cette ligne, un lecteur qui accepterait n importe quel index de 6 bits passerait le test.
func bobinesEquipes() []bobineEquipes {
	return []bobineEquipes{
		{film: "a521164d", entites: 26, indices: 26, sieges: 24, premiers: 24,
			equipes: []int{0, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0}},
		{film: "60ae07c4", entites: 8, indices: 8, sieges: 8, premiers: 8,
			equipes: []int{0, 1, 0, 1, 1, 1, 0, 0}},
		{film: "11de8353", entites: 27, indices: 26, sieges: 24, premiers: 24,
			equipes: []int{0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 1, 1, 0}},
		{film: "111fa685", entites: 25, indices: 25, horsIdx: 0, nonAtteint: 1, sieges: 24, premiers: 24,
			equipes: []int{1, 0, 1, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 0, 0, 0}},
		{film: "e5adf7b2", entites: 25, indices: 25, sieges: 23, premiers: 23,
			equipes: []int{1, 1, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 1, 0, 0, 0, 1}},
		{film: "bcb6d393", entites: 12, indices: 12, sieges: 8, premiers: 8,
			equipes: []int{1, 1, 1, 0, 0, 0, 1, 0, 1, 1, 1, 1}},
		{film: "fb1a1a72", entites: 8, indices: 8, sieges: 8, premiers: 8,
			equipes: []int{1, 1, 1, 0, 0, 1, 0, 0}},
	}
}

// bobineFilm charge une bobine par build.
func bobineFilm(t *testing.T, film string) *source.Film {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+film)
	f, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("bobine %s : %v — regenerer les bobines du lot 0.A.2", film, err)
	}
	return f
}

// TestScanPlayerTeamsSurLesBobines execute E-LUE, E-DOM et le gel des comptes.
func TestScanPlayerTeamsSurLesBobines(t *testing.T) {
	for _, b := range bobinesEquipes() {
		teams, rep := ScanPlayerTeams(NewFilmContext(bobineFilm(t, b.film)))
		if !rep.Lu() {
			t.Fatalf("%s : lecture refusee (archetypeAbsent=%v composant=%q)", b.film,
				rep.ArchetypeAbsent, rep.Component)
		}
		verifierRapportEquipes(t, b, rep)
		verifierTableEquipes(t, b, teams)
		t.Logf("%-10s paquets %2d records %3d lus %3d | entites %2d (div %d) | index %2d "+
			"(div %d, hors domaine %d/%d) | sans equipe %d", b.film, rep.Packets, rep.Records,
			rep.Read, rep.Entities, rep.EntityDivergences, rep.Indices, rep.IndexDivergences,
			rep.OutOfDomainIndex, rep.OutOfDomainValue, rep.NoTeam)
	}
}

// verifierRapportEquipes gele les compteurs du rapport (E-LUE).
func verifierRapportEquipes(t *testing.T, b bobineEquipes, rep TeamScanReport) {
	t.Helper()
	if rep.Unreached != b.nonAtteint {
		t.Errorf("%s : %d record(s) dont la marche n atteint pas i0, attendu %d",
			b.film, rep.Unreached, b.nonAtteint)
	}
	if rep.EntityDivergences != 0 || rep.IndexDivergences != 0 {
		t.Errorf("%s : %d entite(s) et %d index divergent — un joueur ne change pas d'equipe",
			b.film, rep.EntityDivergences, rep.IndexDivergences)
	}
	if rep.OutOfDomainValue != 0 {
		t.Errorf("%s : %d valeur(s) brute(s) hors de 0..%d", b.film, rep.OutOfDomainValue,
			teamDesignatorRawMax)
	}
	if rep.OutOfDomainIndex != b.horsIdx {
		t.Errorf("%s : %d index hors table, attendu %d", b.film, rep.OutOfDomainIndex, b.horsIdx)
	}
	if rep.Entities != b.entites || rep.Indices != b.indices {
		t.Errorf("%s : %d entites / %d index, attendu %d / %d", b.film, rep.Entities,
			rep.Indices, b.entites, b.indices)
	}
	if rep.Records != rep.Read+rep.Unreached+rep.OutOfDomainIndex+rep.OutOfDomainValue {
		t.Errorf("%s : les issues ne totalisent pas les records (%d contre %d)", b.film,
			rep.Read+rep.Unreached+rep.OutOfDomainIndex+rep.OutOfDomainValue, rep.Records)
	}
}

// verifierTableEquipes execute E-DOM et compare la table au gel.
func verifierTableEquipes(t *testing.T, b bobineEquipes, teams map[int]int) {
	t.Helper()
	idx := triIndex(teams)
	if len(idx) != len(b.equipes) {
		t.Fatalf("%s : %d index publies, le gel en porte %d", b.film, len(idx), len(b.equipes))
	}
	for rang, i := range idx {
		if i != rang {
			t.Errorf("%s : l'index publie au rang %d vaut %d — la table n'est plus contigue",
				b.film, rang, i)
		}
		if got := teams[i]; got != b.equipes[rang] {
			t.Errorf("%s index %d : designateur %d, attendu %d", b.film, i, got, b.equipes[rang])
		}
		if v := teams[i]; v < TeamNone || v >= teamDesignatorRawMax { // E-DOM
			t.Errorf("%s index %d : designateur %d hors de %d..%d", b.film, i, v, TeamNone,
				teamDesignatorRawMax-1)
		}
	}
}

// triIndex rend les index d'une table d'equipes, tries.
func triIndex(teams map[int]int) []int {
	out := make([]int, 0, len(teams))
	for i := range teams {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}

// TestScanPlayerTeamsIndexEstLeSiege execute E-INDEX : l'index que le film ecrit dans l'etat par
// defaut de ti=9 est le RANG DU SIEGE de la table des joueurs de `chunk_00`.
//
// C'EST LA MESURE QUI REMPLACE L'APPARIEMENT ORDINAL de la note du 2026-09-12. L'ordinal y
// reste comme CONTROLE : le k-ieme record du paquet doit porter l'index du k-ieme siege. Si un
// jour les deux divergent, ce test le dira au lieu de laisser un appariement muet se tromper.
func TestScanPlayerTeamsIndexEstLeSiege(t *testing.T) {
	for _, b := range bobinesEquipes() {
		film := bobineFilm(t, b.film)
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre illisible : %v", b.film, err)
		}
		chunk0, ok := FilmRegistryChunk(film)
		if !ok {
			t.Fatalf("%s : pas de chunk_00", b.film)
		}
		ident, err := ReadFilmIdentity(chunk0)
		if err != nil {
			t.Fatalf("%s : identite illisible : %v", b.film, err)
		}
		slots, _, err := ReadPlayerTable(chunk0, ident)
		if err != nil {
			t.Fatalf("%s : table des joueurs illisible : %v", b.film, err)
		}
		if len(slots) != b.sieges {
			t.Fatalf("%s : %d sieges, attendu %d", b.film, len(slots), b.sieges)
		}
		lus := indexDuPremierPaquet(t, fc, reg)
		if len(lus) != b.premiers {
			t.Fatalf("%s : %d entites au premier paquet, attendu %d", b.film, len(lus), b.premiers)
		}
		for k, s := range slots {
			if k >= len(lus) {
				break
			}
			if lus[k] != s.FilmIndex {
				t.Errorf("%s rang %d : le film ecrit l'index %d, le siege porte %d", b.film, k,
					lus[k], s.FilmIndex)
			}
		}
		t.Logf("%-10s %2d sieges, %2d entites au premier paquet, index %v", b.film, len(slots),
			len(lus), lus)
	}
}

// indexDuPremierPaquet rend les index de joueur des records ti=9 du PREMIER paquet d'image-cle
// qui en porte, dans l'ordre des positions de bit.
func indexDuPremierPaquet(t *testing.T, fc *FilmContext, reg *Registry) []int {
	t.Helper()
	for _, c := range fc.ChunkNumbers() {
		raw, paquets, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range paquets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(raw)
			var out []int
			for _, b := range keyframeBornesToutes(pay) {
				if b.TI != managedPlayerTypeIndex {
					continue
				}
				if idx, _, ok := lireEquipeDuRecord(pay, b.Bit, reg, ContexteParDefaut()); ok {
					out = append(out, idx)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

// TestScanPlayerTeamsTemoinDUnBit execute E-TEMOIN : relire le champ un bit plus loin ne rend
// pas la meme table. Un plancher de faux positifs se MESURE (regle 4 de
// `METHODE_RETRO_INGENIERIE_FILM`), il ne se suppose pas.
func TestScanPlayerTeamsTemoinDUnBit(t *testing.T) {
	for _, b := range bobinesEquipes() {
		film := bobineFilm(t, b.film)
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre illisible : %v", b.film, err)
		}
		bonnes, decalees, ecarts := 0, 0, 0
		for _, c := range fc.ChunkNumbers() {
			raw, paquets, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			for _, pk := range paquets {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				pay := pk.Payload(raw)
				for _, bo := range keyframeBornesToutes(pay) {
					if bo.TI != managedPlayerTypeIndex {
						continue
					}
					_, brut, ok := lireEquipeDuRecord(pay, bo.Bit, reg, ContexteParDefaut())
					if !ok {
						continue
					}
					bonnes++
					tr := WalkKeyframeFullState(pay, bo.Bit, reg, ContexteParDefaut())
					voisin := int(kfReadBits(pay, tr.Comps[0].StartBit+1, teamDesignatorBits))
					decalees++
					if voisin != brut {
						ecarts++
					}
				}
			}
		}
		if bonnes == 0 || ecarts == 0 {
			t.Errorf("%s : %d lectures, %d ecarts au voisin d'UN bit — une lecture qui ne bouge "+
				"pas d'un bit n'est pas ancree", b.film, bonnes, ecarts)
		}
		t.Logf("%-10s %3d lectures ; le voisin d'un bit differe sur %d/%d", b.film, bonnes,
			ecarts, decalees)
	}
}

// TestScanPlayerTeamsEntreeTronquee execute E-COUPE : aucune panique, aucune lecture partielle.
func TestScanPlayerTeamsEntreeTronquee(t *testing.T) {
	film := bobineFilm(t, "fb1a1a72")
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	pay, bornes := premierPaquetTI9(t, fc)
	coupes := []int{0, 1, 2, 8, 64, 512, 4096, len(pay) / 4, len(pay) / 2, len(pay) - 1}
	for _, n := range coupes {
		tronque := pay[:n]
		lus := 0
		for _, b := range bornes {
			if idx, brut, ok := lireEquipeDuRecord(tronque, b, reg, ContexteParDefaut()); ok {
				lus++
				if idx < 0 || idx >= playerTableSlots || brut < 0 || brut > teamDesignatorRawMax {
					t.Errorf("coupe a %d : lecture hors domaine (index %d, brut %d)", n, idx, brut)
				}
			}
		}
		t.Logf("coupe a %6d o : %d lecture(s) sur %d records, aucune panique", n, lus, len(bornes))
	}
}

// premierPaquetTI9 rend le payload du premier paquet d'image-cle porteur de ti=9 et les bits de
// depart de ses records ti=9.
func premierPaquetTI9(t *testing.T, fc *FilmContext) ([]byte, []int) {
	t.Helper()
	for _, c := range fc.ChunkNumbers() {
		raw, paquets, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range paquets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(raw)
			var bits []int
			for _, b := range keyframeBornesToutes(pay) {
				if b.TI == managedPlayerTypeIndex {
					bits = append(bits, b.Bit)
				}
			}
			if len(bits) > 0 {
				return pay, bits
			}
		}
	}
	t.Fatal("aucun paquet d'image-cle porteur de ti=9 dans la bobine")
	return nil, nil
}

// TestScanPlayerTeamsRefusSansRegistre execute E-REFUS : un film sans `chunk_00` ne rend pas une
// table vide, il rend un REFUS nomme.
func TestScanPlayerTeamsRefusSansRegistre(t *testing.T) {
	teams, rep := ScanPlayerTeams(NewFilmContext(nil))
	if teams != nil || !rep.ArchetypeAbsent || rep.Lu() {
		t.Fatalf("film nil : table %v, rapport %+v — attendu un refus nomme", teams, rep)
	}
	// La bobine historique n'a PAS de chunk_00 (cf. keyframe_closure_ratchet_test.go) : c'est le
	// cas reel du meme refus.
	sansRegistre := bobineFilm(t, "000d5950")
	teams, rep = ScanPlayerTeams(NewFilmContext(sansRegistre))
	if teams != nil || !rep.ArchetypeAbsent || rep.Lu() {
		t.Fatalf("bobine sans registre : table %v, rapport %+v — attendu un refus nomme", teams, rep)
	}
}
