package replay

// identity_registry_film_table_test.go — LA COMPOSITION « FILM D ABORD, CHUNKS EN COMPLEMENT ».
//
// Ce que ces tests prouvent :
//
//	T-ORDRE     un xuid que la table du film assoit prend SON index, meme si le controle en donne
//	            un autre — « l index c est l index » ;
//	T-COMPLET   la table effective n est JAMAIS plus pauvre que la lecture des chunks seule ;
//	T-REPLI     un xuid sans siege prend la voie `PlayerIndexTable` et incremente `repli` ;
//	T-CONTROLE  accord / contradiction / silence comptent ce que le controle dit, sans rien decider ;
//	T-ABSTENTION un vacant INTERCALE refuse la table entiere, sous la cause `vacant_intercale` ;
//	T-REFUS     une table refusee laisse la lecture des chunks seule, et le refus est publie.

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// tableDuFilm fabrique une table LUE portant les sieges donnes.
func tableDuFilm(sieges ...FilmPlayerSeat) FilmPlayerTable {
	return FilmPlayerTable{Seats: sieges, Build: "HI_1_13_0",
		Occupied: len(sieges), Vacant: 32 - len(sieges)}
}

// entreeIdentite fabrique l entree minimale d une composition.
func entreeIdentite(t FilmPlayerTable, controle map[uint64]int) IdentityInput {
	return IdentityInput{
		FilmTable:     t,
		PlayerIndices: PlayerIndexTable{ByXUID: controle, Readings: 26},
	}
}

func TestCompositionLaTableDuFilmPrimeSurLeControle(t *testing.T) {
	in := entreeIdentite(
		tableDuFilm(
			FilmPlayerSeat{FilmIndex: 0, XUID: 11, Gamertag: "Alpha"},
			FilmPlayerSeat{FilmIndex: 3, XUID: 22, Gamertag: "Bravo"},
		),
		map[uint64]int{11: 0, 22: 9, 33: 7},
	)
	got := composerTableDIndex(in)

	// T-ORDRE : le xuid 22 garde l index 3 du FILM, pas le 9 du controle.
	if got.table.ByXUID[22] != 3 {
		t.Errorf("xuid 22 -> index %d, attendu 3 (la table du film fait foi)", got.table.ByXUID[22])
	}
	if got.voieDuLienDIndex(22) != canonical.MethodFilmPlayerTable {
		t.Errorf("voie %q, attendue %q", got.voieDuLienDIndex(22), canonical.MethodFilmPlayerTable)
	}
	// T-REPLI : le xuid 33 n a pas de siege — il entre par la lecture des chunks.
	if got.table.ByXUID[33] != 7 {
		t.Errorf("xuid 33 -> index %d, attendu 7 (repli)", got.table.ByXUID[33])
	}
	if got.voieDuLienDIndex(33) != canonical.MethodPlayerIndexTable {
		t.Errorf("voie du repli %q, attendue %q",
			got.voieDuLienDIndex(33), canonical.MethodPlayerIndexTable)
	}
	// T-CONTROLE : un accord, une contradiction, zero silence, un repli.
	c := got.couverture
	if c.Direct != 2 || c.Fallback != 1 {
		t.Errorf("direct=%d repli=%d, attendus 2 et 1", c.Direct, c.Fallback)
	}
	if c.Accord != 1 || c.Contradiction != 1 || c.Silence != 0 {
		t.Errorf("accord=%d contradiction=%d silence=%d, attendus 1, 1, 0",
			c.Accord, c.Contradiction, c.Silence)
	}
	if c.Seats != 2 || !c.Read {
		t.Errorf("sieges=%d lue=%v, attendus 2 et true", c.Seats, c.Read)
	}
	if got.noms[11] != "Alpha" {
		t.Errorf("gamertag du film perdu : %q", got.noms[11])
	}
}

func TestCompositionCompteLeSilenceDuControle(t *testing.T) {
	in := entreeIdentite(
		tableDuFilm(FilmPlayerSeat{FilmIndex: 5, XUID: 44, Gamertag: "Charlie"}),
		map[uint64]int{},
	)
	got := composerTableDIndex(in)
	if got.couverture.Silence != 1 {
		t.Fatalf("silence=%d, attendu 1 — le controle ne porte pas ce xuid",
			got.couverture.Silence)
	}
	if got.couverture.Accord != 0 || got.couverture.Contradiction != 0 {
		t.Errorf("un silence n est ni un accord ni une contradiction : %d / %d",
			got.couverture.Accord, got.couverture.Contradiction)
	}
}

// TestCompositionNEstJamaisPlusPauvreQueLeControle (T-COMPLET) est l invariant qui interdit la
// PERTE : le gate corpus echoue sur toute baisse, et remplacer une lecture par l autre en
// produirait une (la table du film est celle du DEBUT du film).
func TestCompositionNEstJamaisPlusPauvreQueLeControle(t *testing.T) {
	controle := map[uint64]int{1: 0, 2: 1, 3: 2, 4: 3}
	cas := map[string]FilmPlayerTable{
		"table lue partielle": tableDuFilm(FilmPlayerSeat{FilmIndex: 0, XUID: 1}),
		"table refusee":       {Refusal: FilmTableNoSection},
		"table vide":          {},
		"vacant intercale": func() FilmPlayerTable {
			t := tableDuFilm(FilmPlayerSeat{FilmIndex: 0, XUID: 1})
			t.InterleavedVacant = true
			return t
		}(),
	}
	for nom, film := range cas {
		t.Run(nom, func(t *testing.T) {
			got := composerTableDIndex(entreeIdentite(film, controle))
			for x, pi := range controle {
				if got.table.ByXUID[x] != pi {
					t.Errorf("xuid %d perdu ou deplace : %d au lieu de %d",
						x, got.table.ByXUID[x], pi)
				}
			}
			if len(got.table.ByXUID) < len(controle) {
				t.Errorf("%d lien(s) pour %d au controle — la composition a PERDU",
					len(got.table.ByXUID), len(controle))
			}
		})
	}
}

// TestCompositionRefuseUnVacantIntercale (T-ABSTENTION) : le rang absolu et l index parmi les
// occupes divergent, aucun oracle ne les departage (decouverte D3 (1.5)) — on se tait.
func TestCompositionRefuseUnVacantIntercale(t *testing.T) {
	film := tableDuFilm(FilmPlayerSeat{FilmIndex: 4, XUID: 77, Gamertag: "Delta"})
	film.InterleavedVacant = true
	got := composerTableDIndex(entreeIdentite(film, map[uint64]int{77: 2}))
	if got.couverture.Read {
		t.Fatal("une table a vacant intercale ne doit PAS etre employee")
	}
	if got.couverture.Refusal != canonical.FilmTableInterleavedVacant {
		t.Errorf("refus %q, attendu %q", got.couverture.Refusal,
			canonical.FilmTableInterleavedVacant)
	}
	if got.table.ByXUID[77] != 2 {
		t.Errorf("le lien doit venir du controle : %d au lieu de 2", got.table.ByXUID[77])
	}
	if got.couverture.Direct != 0 || got.couverture.Fallback != 1 {
		t.Errorf("direct=%d repli=%d, attendus 0 et 1", got.couverture.Direct,
			got.couverture.Fallback)
	}
}

// TestCompositionPublieLaCauseDuRefus (T-REFUS) : les cinq causes de `filmdec` traversent jusqu a
// la couverture, parce qu un artefact doit dire POURQUOI il a ete cuit sans la table du film.
func TestCompositionPublieLaCauseDuRefus(t *testing.T) {
	for _, cause := range []FilmTableRefusal{
		FilmTableNoRegistry, FilmTableNoSection, FilmTableUnknownBuild,
		FilmTableTruncated, FilmTableNotFound,
	} {
		t.Run(string(cause), func(t *testing.T) {
			got := composerTableDIndex(entreeIdentite(
				FilmPlayerTable{Refusal: cause}, map[uint64]int{9: 1}))
			if got.couverture.Read {
				t.Fatal("une table refusee ne doit pas etre lue")
			}
			if got.couverture.Refusal != string(cause) {
				t.Errorf("refus publie %q, attendu %q", got.couverture.Refusal, cause)
			}
			if got.couverture.Direct != 0 {
				t.Errorf("%d lien(s) direct(s) sur une table refusee", got.couverture.Direct)
			}
		})
	}
}

// TestRosterPrendLesSiegesEtLesNomsDuFilm (lot 1.6.2) : le roster publie gagne les joueurs que la
// table du film assoit et que la lecture des chunks ne trouve pas, ET les gamertags des joueurs a
// ZERO MORT — que le fil des morts ne peut pas nommer, par construction.
func TestRosterPrendLesSiegesEtLesNomsDuFilm(t *testing.T) {
	const (
		quiMeurt   = uint64(2533274800000001)
		quiNeMeurt = uint64(2533274800000002)
		quiRejoint = uint64(2533274800000003)
	)
	film := tableDuFilm(
		FilmPlayerSeat{FilmIndex: 0, XUID: quiMeurt, Gamertag: "Alpha"},
		FilmPlayerSeat{FilmIndex: 1, XUID: quiNeMeurt, Gamertag: "Bravo"},
	)
	in := entreeIdentite(film, map[uint64]int{quiMeurt: 0, quiRejoint: 5})
	in.Deaths = []Death{{XUID: quiMeurt, Gamertag: "Alpha", TimeMS: 1000}}
	reg := BuildIdentityRegistry(in)

	roster := buildRoster(reg.TableDIndex(), nomsDesJoueurs(reg, in.Deaths), nil, teamPublication{})
	parXUID := map[string]RosterEntry{}
	for _, r := range roster {
		parXUID[r.XUID] = r
	}
	if len(roster) != 3 {
		t.Fatalf("%d joueur(s) au roster, attendus 3 (deux sieges + un arrivant)", len(roster))
	}
	if got := parXUID["2533274800000002"]; got.Name != "Bravo" {
		t.Errorf("le joueur a ZERO MORT est publie sans nom (%q) — le film ecrit le sien", got.Name)
	}
	if got := parXUID["2533274800000003"]; got.FilmIndex != 5 {
		t.Errorf("l arrivant en cours de partie est perdu ou deplace : index %d", got.FilmIndex)
	}
	// LA TAILLE REELLE DE L ESCOUADE EST UNE DONNEE, et elle n est PAS le compte du roster : le
	// roster porte aussi les arrivants, la table du film porte les sieges du debut.
	if got := reg.CouvertureTableDuFilm().Seats; got != 2 {
		t.Errorf("sieges publies %d, attendus 2", got)
	}
}

// TestCompositionRetireUnIndexQueDeuxXUIDSeDisputent — LE CORRECTIF DE LA REVUE DE JALON M1
// (lentille L4) : la table COMPOSEE est injective par INDEX, comme l'est la lecture des chunks
// seule (`injectiveOrEmpty`).
//
// # LE DEFAUT QUE CE TEST GARDE
//
// `composerTableDIndex` ne testait que la duplication de XUID (`completerParLesChunks` :
// « ce xuid est-il deja pose ? »), jamais celle d'INDEX — et sa sortie REMPLACE la table gardee
// (`BuildIdentityRegistry` : `in.PlayerIndices = reg.filmTable.table`). Il suffisait qu'un xuid
// soit assis a l'index i par le film sans figurer dans `PlayerIndices`, pendant qu'un REMPLACANT
// est lu a ce meme index i dans les chunks — et la reprise d'index d'un partant est mesuree
// (`11de8353` index 23, `51101d1d` index 6). `indexToXUIDOf` ecrasait alors sans garde, a l'ordre
// d'iteration d'une map : l'identite d'une vie changeait d'une cuisson a l'autre.
//
// # MUTATION
//
// Retirer l'appel a `retirerLesIndexEnCollision` dans `composerTableDIndex` : les deux xuids
// reviennent dans la table, le compteur retombe a zero, et le sous-test « determinisme » rougit
// une execution sur deux — c'est pour cela qu'il boucle.
func TestCompositionRetireUnIndexQueDeuxXUIDSeDisputent(t *testing.T) {
	const (
		partant     = uint64(2533274800000011)
		remplacant  = uint64(2533274800000022)
		indifferent = uint64(2533274800000033)
	)
	// Le partant tient l'index 7 dans la table du film et ne figure PAS dans les chunks ; le
	// remplacant y est lu au MEME index 7. Le troisieme joueur est le temoin : il doit survivre.
	in := entreeIdentite(
		tableDuFilm(
			FilmPlayerSeat{FilmIndex: 7, XUID: partant, Gamertag: "Partant"},
			FilmPlayerSeat{FilmIndex: 2, XUID: indifferent, Gamertag: "Temoin"},
		),
		map[uint64]int{remplacant: 7, indifferent: 2},
	)
	got := composerTableDIndex(in)

	if _, y := got.table.ByXUID[partant]; y {
		t.Errorf("le partant garde l index 7 alors que deux xuids le revendiquent")
	}
	if _, y := got.table.ByXUID[remplacant]; y {
		t.Errorf("le remplacant prend l index 7 alors que deux xuids le revendiquent")
	}
	if got.table.ByXUID[indifferent] != 2 {
		t.Errorf("le temoin a perdu son index (%d) : la collision ne doit toucher que l index 7",
			got.table.ByXUID[indifferent])
	}
	if got.couverture.IndexCollisions != 1 {
		t.Errorf("collisionsIndex = %d, attendu 1 : une collision se COMPTE, elle ne se tait pas",
			got.couverture.IndexCollisions)
	}
	// Les compteurs ne decrivent que ce qui RESTE publie : un seul lien direct (le temoin), zero
	// repli (le remplacant est parti avec la collision).
	if got.couverture.Direct != 1 || got.couverture.Fallback != 0 {
		t.Errorf("direct=%d repli=%d, attendus 1 et 0 — ils doivent compter la table PUBLIEE",
			got.couverture.Direct, got.couverture.Fallback)
	}
	if got.noms[partant] != "Partant" {
		t.Errorf("le gamertag du film est perdu (%q) : il est porte par le XUID, pas par l index",
			got.noms[partant])
	}
	// LE DETERMINISME EST LA PROPRIETE MENACEE, et c'est l'ordre d'iteration d'une map qui la
	// menacait : on recompose, et l'identite publiee ne doit jamais bouger.
	t.Run("determinisme", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			c := composerTableDIndex(in)
			if len(c.table.ByXUID) != 1 || c.table.ByXUID[indifferent] != 2 {
				t.Fatalf("passe %d : table = %v, attendue {temoin: 2} a chaque composition",
					i, c.table.ByXUID)
			}
		}
	})
}

// TestCompositionSansCollisionNeRetireRien — LA BORNE BASSE du correctif : deux index distincts
// n en sont pas une, et rien ne doit partir.
func TestCompositionSansCollisionNeRetireRien(t *testing.T) {
	in := entreeIdentite(
		tableDuFilm(FilmPlayerSeat{FilmIndex: 0, XUID: 11, Gamertag: "Alpha"}),
		map[uint64]int{22: 1},
	)
	got := composerTableDIndex(in)

	if got.couverture.IndexCollisions != 0 {
		t.Errorf("collisionsIndex = %d sur deux index distincts", got.couverture.IndexCollisions)
	}
	if got.table.ByXUID[11] != 0 || got.table.ByXUID[22] != 1 {
		t.Errorf("table = %v, attendue {11:0, 22:1}", got.table.ByXUID)
	}
	if got.couverture.Direct != 1 || got.couverture.Fallback != 1 {
		t.Errorf("direct=%d repli=%d, attendus 1 et 1",
			got.couverture.Direct, got.couverture.Fallback)
	}
}
