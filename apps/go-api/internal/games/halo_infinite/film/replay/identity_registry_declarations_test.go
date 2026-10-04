package replay

// identity_registry_declarations_test.go — LE CORPS D'UN INDEX QUE PLUSIEURS BOTS SE RELAIENT,
// NOMME PAR LES DECLARATIONS BOT_METADATA QUAND L'ENTITE SE TAIT.
//
//	R-DECLARATION   gabarit de `c7f94693` (index 8, trois bots, aucun humain) : le bot dont toute
//	                la presence tombe entre deux images-cles n'a pas d'entite ; la declaration qui
//	                couvre la creation de son corps et sa vie le nomme ;
//	R-HUMAIN        un humain de la table tient l'index : la lecture se tait ;
//	R-DEBORDE       la vie deborde la declaration : la lecture se tait ;
//	R-DEUX-BOTS     deux bots distincts couvrent la creation et la vie : la lecture se tait ;
//	R-DEUX-A-LA-CREATION  deux bots declares a la creation, un seul couvre la vie : elle se tait ;
//	R-CREATION      la declaration couvre la vie mais nait apres la creation : elle se tait ;
//	R-UNE-DECLARATION  aucune declaration du bot ne couvre a la fois creation et vie : elle se tait ;
//	R-UN-SEUL-BOT   un index d'un seul bot n'entre pas dans la lecture ;
//	R-BORNES        une declaration est [debut, fin), `fin == 0` court jusqu'au bout ;
//	R-NEGATIF       un instant negatif ne se lit pas.

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// entreeDonos : un humain (slot 100, index 0) et le corps 532 de `343 Donos`, index 8, cree a
// 50,59 s ; trois bots declarent l'index 8. Seule l'entite de Byrontron (a partir de 421,2 s) est
// lue sur cet index : aucune entite ne vit a la creation du corps 532.
func entreeDonos(finDeVie uint64) IdentityInput {
	var pos []grammar.BipedPosition
	for t := uint64(500_000); t <= 460_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 0))
	}
	for t := uint64(50_616_045); t <= finDeVie; t += 500_000 {
		pos = append(pos, posAt(532, t, 2, 2, 0))
	}
	scan := grammar.PlayerEntityScan{Scanned: true}
	for t := uint64(1_200_000); t <= 461_200_000; t += 20_000_000 {
		scan.KeyframesUS = append(scan.KeyframesUS, t)
	}
	fin := len(scan.KeyframesUS) - 1
	scan.Entities = []grammar.PlayerEntity{
		{Slot: 1297, Index: 0, Team: 0, FirstKF: 0, LastKF: fin, Seen: fin + 1},
		{Slot: 2504, Index: 8, Team: 0, FirstKF: 21, LastKF: fin, Seen: fin - 20}, // 421,2 s ..
	}
	return IdentityInput{
		Positions:      pos,
		BipedCreations: []grammar.BipedCreation{creationDe(100, 500_000, 0), creationDe(532, 50_585_841, 8)},
		PlayerIndices:  types.PlayerIndexTable{ByXUID: map[uint64]int{111: 0}, Readings: 20},
		Bots: []BotIdentity{
			{FilmIndex: 8, Name: "343 Donos [bot]", BotID: 6, Declarations: [][2]uint64{{42_431_556, 54_386_913}}},
			{FilmIndex: 8, Name: "343 The Thumb [bot]", BotID: 57, Declarations: [][2]uint64{{149_616_358, 149_716_485}}},
			{FilmIndex: 8, Name: "343 Byrontron [bot]", BotID: 23, Declarations: [][2]uint64{{417_543_778, 0}}},
		},
		Entities: scan,
		Clock:    IdentityClock{OriginUS: 500_000, StepUS: 100_000, FrameCount: 4600},
		MatchID:  "temoin",
	}
}

func TestCorpsSansEntiteNommeParLaDeclarationQuiCouvreSaVie(t *testing.T) {
	reg := BuildIdentityRegistry(context.Background(), entreeDonos(54_088_267))
	if got := bidsParSlot(reg)[532]; got != "bid(6.0)" {
		t.Fatalf("corps 532 : bid %q, attendu %q (la seule declaration qui couvre sa creation et sa vie)",
			got, "bid(6.0)")
	}
	if reg.creation.ParDeclaration != 1 || reg.creation.ParEntite != 0 {
		t.Errorf("par declaration %d, par entite %d : attendu 1 et 0", reg.creation.ParDeclaration,
			reg.creation.ParEntite)
	}
	for _, l := range reg.Vies() {
		if l.slot == 532 && l.nomPar != NomParCreation {
			t.Errorf("corps 532 : voie %q, attendu %q — une LECTURE du film", l.nomPar, NomParCreation)
		}
	}
}

func TestDeclarationSeTaitSurUnIndexTenuParUnHumain(t *testing.T) {
	in := entreeDonos(54_088_267)
	in.PlayerIndices.ByXUID[222] = 8
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := bidsParSlot(reg)[532]; got != "" || reg.creation.ParDeclaration != 0 {
		t.Fatalf("corps 532 : bid %q, par declaration %d — un humain tient l'index, la declaration se tait",
			got, reg.creation.ParDeclaration)
	}
}

func TestDeclarationSeTaitQuandLaVieLaDeborde(t *testing.T) {
	reg := BuildIdentityRegistry(context.Background(), entreeDonos(56_000_000))
	if got := bidsParSlot(reg)[532]; got != "" || reg.creation.ParDeclaration != 0 {
		t.Fatalf("corps 532 : bid %q, par declaration %d — la vie deborde la declaration, rien n'est prouve",
			got, reg.creation.ParDeclaration)
	}
}

func TestDeclarationSeTaitQuandDeuxBotsCouvrentLaVie(t *testing.T) {
	in := entreeDonos(54_088_267)
	in.Bots[1].Declarations = [][2]uint64{{40_000_000, 60_000_000}}
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := bidsParSlot(reg)[532]; got != "" || reg.creation.ParDeclaration != 0 {
		t.Fatalf("corps 532 : bid %q, par declaration %d — deux bots couvrent la vie, rien ne les departage",
			got, reg.creation.ParDeclaration)
	}
}

// R-TABLEAU : le refus du tableau sur un index que plusieurs bots declarent est compte comme tel,
// pas comme un index que personne ne declare.
func TestTableauCompteLIndexPartageAPart(t *testing.T) {
	in := entreeDonos(56_000_000)
	in.Participants = []Participant{{ID: "111"}, {ID: "bid(6.0)"}, {ID: "bid(57.0)"}, {ID: "bid(23.0)"}}
	reg := BuildIdentityRegistry(context.Background(), in)
	if reg.tableau.IndexPartage == 0 || reg.tableau.SansCandidat != 0 {
		t.Fatalf("index partage %d, sans candidat %d : le refus vient de l'index partage",
			reg.tableau.IndexPartage, reg.tableau.SansCandidat)
	}
}

// R-DEUX-A-LA-CREATION : deux bots distincts sont declares a l'instant ou le corps nait ; un seul
// (Donos) couvre toute la vie. Rien ne dit lequel des deux est ne dans ce corps : la lecture se tait.
func TestDeclarationSeTaitQuandDeuxBotsSontDeclaresALaCreation(t *testing.T) {
	in := entreeDonos(54_088_267)
	in.Bots[1].Declarations = [][2]uint64{{40_000_000, 52_000_000}} // The Thumb, finit avant la mort
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := bidsParSlot(reg)[532]; got != "" || reg.creation.ParDeclaration != 0 {
		t.Fatalf("corps 532 : bid %q, par declaration %d — deux bots declares a la creation",
			got, reg.creation.ParDeclaration)
	}
}

// R-CREATION : la declaration de Donos commence entre la creation du corps (50,586 s) et son
// premier echantillon (50,616 s) : elle couvre la vie, pas la creation. La lecture se tait.
func TestDeclarationSeTaitQuandElleNaitApresLaCreation(t *testing.T) {
	in := entreeDonos(54_088_267)
	in.Bots[0].Declarations = [][2]uint64{{50_600_000, 54_386_913}}
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := bidsParSlot(reg)[532]; got != "" || reg.creation.ParDeclaration != 0 {
		t.Fatalf("corps 532 : bid %q, par declaration %d — aucun bot declare a la creation",
			got, reg.creation.ParDeclaration)
	}
}

// R-UNE-DECLARATION : Donos est declare a la creation par une declaration, puis pendant la vie par
// une autre ; aucune des deux ne couvre la creation ET la vie. La lecture se tait.
func TestDeclarationQuiCouvreLaVieDoitAussiCouvrirLaCreation(t *testing.T) {
	in := entreeDonos(54_088_267)
	in.Bots[0].Declarations = [][2]uint64{{42_431_556, 50_600_000}, {50_610_000, 54_386_913}}
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := bidsParSlot(reg)[532]; got != "" || reg.creation.ParDeclaration != 0 {
		t.Fatalf("corps 532 : bid %q, par declaration %d — aucune declaration ne couvre creation et vie",
			got, reg.creation.ParDeclaration)
	}
}

// R-UN-SEUL-BOT : un index qu'un seul bot declare n'entre pas dans la lecture (l'index le nomme
// deja) ; le corps reste compte `IndexBot`, jamais `ParDeclaration`.
func TestDeclarationNeLitPasLIndexDUnSeulBot(t *testing.T) {
	in := entreeDonos(54_088_267)
	in.Bots = in.Bots[:1]
	reg := BuildIdentityRegistry(context.Background(), in)
	if reg.creation.ParDeclaration != 0 || reg.creation.IndexBot != 1 {
		t.Fatalf("par declaration %d, indexBot %d : attendu 0 et 1 — un seul bot sur l'index",
			reg.creation.ParDeclaration, reg.creation.IndexBot)
	}
}

// R-BORNES : une declaration est l'intervalle [debut, fin) ; `fin == 0` court jusqu'au bout.
func TestDeclarationCouvreBornes(t *testing.T) {
	decl := [][2]uint64{{10, 20}}
	cas := []struct {
		de, a uint64
		want  bool
	}{
		{10, 19, true},  // debut inclus
		{10, 20, false}, // fin exclue
		{9, 19, false},  // commence avant
		{12, 12, true},  // un instant
	}
	for _, c := range cas {
		if got := declarationCouvre(decl, c.de, c.a); got != c.want {
			t.Errorf("[%d,%d] dans [10,20) : %v, attendu %v", c.de, c.a, got, c.want)
		}
	}
	if !declarationCouvre([][2]uint64{{10, 0}}, 10, math.MaxUint64-1) {
		t.Error("declaration ouverte [10, fin) : doit couvrir jusqu'au bout")
	}
}

// R-NEGATIF : un instant anterieur a l'origine du film (negatif) ne se lit pas, meme face a une
// declaration ouverte qui couvrirait tout.
func TestDeclarationSeTaitSurUnInstantNegatif(t *testing.T) {
	m := botsDesIndexPartages{8: {
		{FilmIndex: 8, Name: "343 Donos [bot]", BotID: 6, Declarations: [][2]uint64{{0, 0}}},
		{FilmIndex: 8, Name: "343 The Thumb [bot]", BotID: 57, Declarations: [][2]uint64{{900, 950}}},
	}}
	if bid, lu := m.botDeclareSurLaVie(8, 5, lifeSpan{from: -1, to: 100}); lu {
		t.Fatalf("vie qui commence a -1 : nommee %q, attendu le silence", bid)
	}
	if bid, lu := m.botDeclareSurLaVie(8, 5, lifeSpan{from: 5, to: 100}); !lu || bid != "bid(6.0)" {
		t.Fatalf("meme vie a partir de 5 : %q, %v, attendu %q", bid, lu, "bid(6.0)")
	}
}
