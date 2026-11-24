package killcollector

// placement_des_vies_portee_test.go — LA PORTEE DU RADAR EST EN VRAIS METRES, LES DISTANCES DU FILM
// EN UNITES DU JEU.
//
// `replay.PlacementDesVies` (brique du decodeur, `replay/placement_des_vies.go`) compare chaque
// distance mesuree sur le film, en unites du jeu, a `EntreePlacement.RadarM` SANS conversion
// (`d > *m.radar`, ligne 234) et recopie cette valeur dans `PlacementVie.RadarM` (ligne 243), alors que
// sa documentation parle de metres. Le collecteur lui passe donc la portee DEJA CONVERTIE en unites
// (`gameunits.MetersToUnits`), et ecrit `radar_m` en vrais metres (18, 24), la valeur que les lecteurs
// comparent a la portee courante du manifeste.

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/gameunits"
	"levelup/go-api/internal/persist"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// TestEntreeDuPlacement_PorteeEnUnitesDuJeu : un radar de 18 m devient 18 / 3,048 = 5,9 unites. Une
// vie dont le coequipier le plus proche est a 7 unites (21,3 m) est HORS du radar : l'ancienne
// transmission (18, lu comme 18 unites, soit 55 m) la disait dedans.
func TestEntreeDuPlacement_PorteeEnUnitesDuJeu(t *testing.T) {
	radarM := 18.0
	e := entreeDuPlacement(materiauDIsolement{}, MatchIdentities{}, persist.KillSourceBatch{}, nil, &radarM)
	if e.RadarM == nil {
		t.Fatal("portee connue : le decodeur doit recevoir une portee")
	}
	if want := gameunits.MetersToUnits(18); math.Abs(*e.RadarM-want) > 1e-12 {
		t.Fatalf("portee transmise %v, attendu %v (18 m en unites du jeu)", *e.RadarM, want)
	}
	// La comparaison de la brique : `d > radar`.
	if d := 7.0; !(d > *e.RadarM) {
		t.Errorf("7 unites (21,3 m) doivent depasser un radar de 18 m (%v unites)", *e.RadarM)
	}
	if d := 5.0; d > *e.RadarM {
		t.Errorf("5 unites (15,2 m) ne doivent pas depasser un radar de 18 m (%v unites)", *e.RadarM)
	}
	if radarM != 18 {
		t.Errorf("la portee de l'appelant a ete modifiee : %v", radarM)
	}
}

func TestEntreeDuPlacement_SansPortee_PasDePortee(t *testing.T) {
	if e := entreeDuPlacement(materiauDIsolement{}, MatchIdentities{}, persist.KillSourceBatch{}, nil, nil); e.RadarM != nil {
		t.Fatalf("variante sans portee : %v transmis", *e.RadarM)
	}
}

// TestToPlacementRows_RadarEcritEnVraisMetres : la ligne porte `radar_m` = 18, jamais les 5,9
// unites que le decodeur a recopiees ; une vie sans portee reste sans portee.
func TestToPlacementRows_RadarEcritEnVraisMetres(t *testing.T) {
	enUnites, hors := gameunits.MetersToUnits(18), int64(300)
	radarM := 18.0
	rows, _ := toPlacementRows([]replay.PlacementVie{
		{XUID: 1, MesureMS: 8_000, HorsRadarMS: &hors, RadarM: &enUnites},
		{XUID: 2, MesureMS: 8_000},
	}, &radarM)
	if rows[0].RadarM == nil || *rows[0].RadarM != 18 {
		t.Fatalf("radar_m = %v, attendu 18 (vrais metres)", rows[0].RadarM)
	}
	if rows[1].RadarM != nil {
		t.Fatalf("une vie sans portee ecrit radar_m = %v", *rows[1].RadarM)
	}
}

// TestPlacementRev_ChangeAvecLaPorteeEnUnites : les vies rangees avec l'ancien rayon (trois fois trop
// large) portent l'ancienne revision ; la revision courante les rend candidates au rattrapage
// (`matchsAJour`).
func TestPlacementRev_ChangeAvecLaPorteeEnUnites(t *testing.T) {
	const ancienne = "placement-2026-10-17"
	if PlacementRev == ancienne {
		t.Fatalf("PlacementRev = %q : les placements ranges avec l'ancien rayon ne seraient pas recalcules", PlacementRev)
	}
}
