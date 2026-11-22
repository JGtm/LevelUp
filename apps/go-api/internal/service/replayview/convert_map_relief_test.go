// convert_map_relief_test.go — LE RELIEF DU FOND, DU SIDECAR AU DOCUMENT SERVI : les deux noms de
// fichier traversent la projection, et leur absence reste une absence (aucun champ servi).
package replayview_test

import (
	"encoding/json"
	"strings"
	"testing"

	"levelup/go-api/internal/mapfondsidecar"
	"levelup/go-api/internal/service/replayview"
)

func TestMapBackgroundOf_ProjetteLeRelief(t *testing.T) {
	fond := &mapfondsidecar.MapBackground{ReliefImage: "carte.relief.webp", ReliefData: "carte.relief.json"}
	servi := replayview.MapBackgroundOf(fond)
	if servi.ReliefImage != "carte.relief.webp" || servi.ReliefData != "carte.relief.json" {
		t.Fatalf("relief projeté = %q, %q", servi.ReliefImage, servi.ReliefData)
	}
	corps, err := json.Marshal(servi)
	if err != nil {
		t.Fatal(err)
	}
	for _, attendu := range []string{`"reliefImage":"carte.relief.webp"`, `"reliefData":"carte.relief.json"`} {
		if !strings.Contains(string(corps), attendu) {
			t.Errorf("corps servi sans %s : %s", attendu, corps)
		}
	}

	sans, err := json.Marshal(replayview.MapBackgroundOf(&mapfondsidecar.MapBackground{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sans), "relief") {
		t.Errorf("champ du relief servi sans relief : %s", sans)
	}
}
