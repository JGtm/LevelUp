// convert_vehicles_destruction_test.go — LA DESTRUCTION DE VEHICULE, DU DOCUMENT STOCKE AU DOCUMENT
// SERVI, avec la semantique d'absence du decodeur : un bloc absent reste absent, un champ non etabli
// reste absent, un destructeur bot garde son index et son drapeau, jamais un xuid.
package replayview_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"levelup/go-api/internal/service/replayview"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func vehiculeDetruit(slot uint32, d *replay.VehicleDestruction) replay.VehicleTrack {
	tEnd := 300
	return replay.VehicleTrack{Slot: slot, Family: "Ghost", End: "destroyed", TEnd: &tEnd, Destruction: d}
}

func TestProjectionDestructionDeVehicule(t *testing.T) {
	cause := uint32(2899431412)
	stocke := replay.ReplayDocument{Vehicles: []replay.VehicleTrack{
		vehiculeDetruit(1, &replay.VehicleDestruction{
			By:       &replay.VehicleDestroyer{I: 21, XUID: "2534182457688434"},
			Cause:    &cause,
			Occupant: "2534937356463633",
			Killed:   []string{"2534937356463633", "2536412345715584"},
		}),
		vehiculeDetruit(2, &replay.VehicleDestruction{By: &replay.VehicleDestroyer{I: 5, Bot: true}}),
		vehiculeDetruit(3, nil),
	}}
	raw, err := json.Marshal(replayview.FromArtifact(stocke))
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}
	var corps struct {
		Vehicles []map[string]any `json:"vehicles"`
	}
	if err := json.Unmarshal(raw, &corps); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if len(corps.Vehicles) != 3 {
		t.Fatalf("%d vehicules servis, attendu 3", len(corps.Vehicles))
	}

	memeArbre(t, "destruction complete", corps.Vehicles[0]["destruction"],
		`{"by":{"i":21,"xuid":"2534182457688434"},"cause":2899431412,`+
			`"occupant":"2534937356463633","killed":["2534937356463633","2536412345715584"]}`)
	memeArbre(t, "destructeur bot", corps.Vehicles[1]["destruction"], `{"by":{"i":5,"bot":true}}`)
	if _, present := corps.Vehicles[2]["destruction"]; present {
		t.Errorf("un vehicule sans bloc ne doit pas en recevoir un : %v", corps.Vehicles[2])
	}
}

// memeArbre compare le sous-arbre servi a un JSON attendu, sans dependre de l'ordre des cles.
func memeArbre(t *testing.T, nom string, servi any, attendu string) {
	t.Helper()
	var veut any
	if err := json.Unmarshal([]byte(attendu), &veut); err != nil {
		t.Fatalf("%s : attendu illisible: %v", nom, err)
	}
	if !reflect.DeepEqual(servi, veut) {
		t.Errorf("%s = %v, attendu %v", nom, servi, veut)
	}
}
