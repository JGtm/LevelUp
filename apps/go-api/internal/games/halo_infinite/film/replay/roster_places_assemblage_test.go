package replay

// roster_places_assemblage_test.go — LE ROSTER PUBLIE PAR L'ASSEMBLAGE (`BuildFromPositions`) : les
// deux passes branchees dans build_pistes.go, de bout en bout.
//
//	AS-DECLARATION  gabarit de `bf2a9f05` : le corps qu'une declaration BOT_METADATA nomme, l'humain de
//	                l'index absent, porte le nom de son bot, et ce bot entre au roster publie (compte
//	                dans `botsSuccesseurs`) ;
//	AS-SANS-PLACE   gabarit de `d1dfbc02` : un bot declare sans aucune vie, dans une equipe qui tient
//	                toutes ses places jusqu'a la fin, n'est pas au roster publie ; aucune place en trop,
//	                aucun depassement, `entrees` egal a la taille du roster publie.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestAssemblagePublieLeBotNommeParSaDeclaration(t *testing.T) { // AS-DECLARATION
	in := entreeDonosHumain(10, 15)
	opt := Options{FrameIntervalMS: 100, BipedCreations: in.BipedCreations, PlayerIndices: in.PlayerIndices,
		Bots: in.Bots, PlayerEntities: in.Entities}
	doc := BuildFromPositions(context.Background(), "temoin", "halo_infinite", in.Positions, nil, opt)
	nomme := false
	for _, tr := range doc.Tracks {
		if tr.Slot == 532 {
			nomme = tr.Bot == "343 Donos [bot]"
		}
	}
	if !nomme {
		t.Fatalf("la piste du corps 532 ne porte pas le nom de Donos : %+v", porteTraces(doc, 532))
	}
	present := false
	for _, e := range doc.Roster {
		present = present || (e.Bot && e.Name == "343 Donos [bot]")
	}
	s := doc.Coverage.Seats
	if !present || s.IdentitesHorsRoster != 0 || s.BotsSuccesseurs != 2 {
		t.Errorf("Donos au roster %v, identites hors roster %d, botsSuccesseurs %d : attendu vrai, 0, 2 "+
			"(Byrontron par son entite, Donos par sa declaration)", present, s.IdentitesHorsRoster, s.BotsSuccesseurs)
	}
}

func TestAssemblageNePubliePasLeBotSansVieNiPlace(t *testing.T) { // AS-SANS-PLACE
	var positions []grammar.BipedPosition
	var creations []grammar.BipedCreation
	idx := types.PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: 4}
	scan := scanDeTest([]int{0, 20, 40, 60, 80})
	for i := range 4 {
		slot := uint32(100 + i)
		creations = append(creations, grammar.BipedCreation{Slot: slot, Generation: 1, HasIndex: true,
			ParticipantIndex: uint32(i)}) //nolint:gosec // index de test
		for ms := 0; ms < 10_000; ms += 100 {
			positions = append(positions, pos(slot, ms, float32(i*10+ms%7), float32(ms%13), 1))
		}
		idx.ByXUID[uint64(1000+i)] = i
		scan.Entities = append(scan.Entities, grammar.PlayerEntity{Slot: 1300 + i, Index: i, Team: i / 2,
			LastKF: 4, Seen: 5})
	}
	equipe := 0
	opt := Options{FrameIntervalMS: 100, BipedCreations: creations, PlayerIndices: idx, PlayerEntities: scan,
		FilmTable: tableDeDebut(0, 1, 2, 3),
		Bots: []BotIdentity{{FilmIndex: 8, Name: "343 Ham Sammich [bot]", BotID: 24, Team: &equipe,
			Declarations: [][2]uint64{{9_000_000, 0}}}}}
	doc := BuildFromPositions(context.Background(), "temoin", "halo_infinite", positions, nil, opt)
	for _, e := range doc.Roster {
		if e.Bot {
			t.Fatalf("roster publie %+v : le bot sans vie ni place n'y entre pas", doc.Roster)
		}
	}
	s := doc.Coverage.Seats
	if len(doc.Roster) != 4 || s.Entrees != 4 || s.SansPlace != 0 || s.PlacesEnTrop != 0 || s.Depassements != 0 {
		t.Errorf("roster %d entrees, couverture %+v : 4 entrees, aucune place en trop ni depassement",
			len(doc.Roster), s)
	}
}
