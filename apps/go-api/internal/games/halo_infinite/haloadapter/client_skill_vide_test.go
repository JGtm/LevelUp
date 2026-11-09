package haloadapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetMatchSkill_Corps200VideEstUneErreur : sur le chemin live, un 200 au corps vide n'est
// pas une réponse skill (le décodage JSON échoue) ; seul le décodage hors ligne
// (ParseMatchSkillResponseJSON) lit un corps vide comme une map vide.
func TestGetMatchSkill_Corps200VideEstUneErreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m, err := clientVers(srv, "s").GetMatchSkill(context.Background(), matchTest, []string{"1"})
	if err == nil {
		t.Fatalf("corps 200 vide : erreur attendue, obtenu map %v", m)
	}
	hors, err := ParseMatchSkillResponseJSON(nil)
	if err != nil || len(hors) != 0 {
		t.Fatalf("hors ligne, corps vide : map vide attendue, obtenu %v, %v", hors, err)
	}
}
