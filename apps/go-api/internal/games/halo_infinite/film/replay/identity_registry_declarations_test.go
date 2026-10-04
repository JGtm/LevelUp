package replay

// identity_registry_declarations_test.go — LE CORPS D'UN INDEX QUE PLUSIEURS BOTS SE RELAIENT,
// NOMME PAR LES DECLARATIONS BOT_METADATA QUAND L'ENTITE SE TAIT.
//
//	R-DECLARATION   gabarit de `c7f94693` (index 8, trois bots, aucun humain) : le bot dont toute
//	                la presence tombe entre deux images-cles n'a pas d'entite ; la declaration qui
//	                couvre la creation de son corps et sa vie le nomme ;
//	R-HUMAIN        un humain de la table tient l'index : la lecture se tait ;
//	R-DEBORDE       la vie deborde la declaration : la lecture se tait ;
//	R-DEUX-BOTS     deux bots distincts couvrent la creation et la vie : la lecture se tait.

import (
	"context"
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
