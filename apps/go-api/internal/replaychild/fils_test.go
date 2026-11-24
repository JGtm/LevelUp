package replaychild

import (
	"encoding/json"
	"testing"
)

// TestRequest_LeNombreDeFilsVoyageVersLEnfant : la requete que le parent ecrit porte le nombre de fils du decodeur et
// l'enfant le relit tel quel ; absent, il vaut zero (un fil).
func TestRequest_LeNombreDeFilsVoyageVersLEnfant(t *testing.T) {
	raw, err := json.Marshal(Request{MatchID: "m", DecoderThreads: 4})
	if err != nil {
		t.Fatal(err)
	}
	var relu Request
	if err := json.Unmarshal(raw, &relu); err != nil {
		t.Fatal(err)
	}
	if relu.DecoderThreads != 4 {
		t.Errorf("fils relus = %d, attendu 4", relu.DecoderThreads)
	}
	var sans Request
	if err := json.Unmarshal([]byte(`{"matchId":"m"}`), &sans); err != nil || sans.DecoderThreads != 0 {
		t.Errorf("sans la cle : %d fils (err %v), attendu 0", sans.DecoderThreads, err)
	}
}
