package ops

// seed_demo_replay_cook_test.go — le câblage de la recuisson démo : la requête de l'enfant
// porte l'interdiction d'écrire les faits sous le dépôt et le plafond mémoire souple. Sans
// eux, la recuisson écrirait dans le vrai data/cache/film_facts, ou l'enfant prendrait le
// plafond par défaut (3 Gio) sur un VPS de 2 Gio. Aucun film n'est décodé.

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/port"
)

func TestReplayChildRequest_PorteLesGardesDeLaDemo(t *testing.T) {
	req := replayChildRequest(SeedDemoOptions{TitleSlug: "halo_infinite", RepoRoot: t.TempDir()},
		"m1", []string{"carte"}, port.MatchFacts{})
	if !req.SansEcritureDesFaits {
		t.Error("la requête n'interdit pas l'écriture des faits sous la racine du dépôt")
	}
	if req.SoftLimitBytes == 0 || req.SoftLimitBytes != demoReplaySoftLimit {
		t.Errorf("SoftLimitBytes = %d, attendu le plafond démo %d", req.SoftLimitBytes, demoReplaySoftLimit)
	}
	// La requête voyage en JSON jusqu'à l'enfant : les deux champs doivent y survivre.
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["sansEcritureDesFaits"] != true || m["softLimitBytes"] == nil {
		t.Errorf("champs absents de la requête sérialisée : %s", raw)
	}
}
