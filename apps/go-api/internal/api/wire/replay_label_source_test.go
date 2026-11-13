package wire

import (
	"errors"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/testutil"
)

// TestReplayLabelSource_ParCapaciteReplay : le catalogue de libellés du rejeu se rend pour un titre qui
// déclare la capacité `replay` (Halo Infinite), et seulement pour lui. Halo 5 et un titre inconnu rendent
// games.ErrCapabilityNotSupported : jamais le catalogue d'un autre titre, même quand les mappings
// d'Halo Infinite sont lisibles sous la même racine.
func TestReplayLabelSource_ParCapaciteReplay(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt : %v", err)
	}
	// Registre du serveur : le titre additionnel y est déclaré sans la capacité `replay`, comme le fait
	// son title.toml.
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{Slug: "halo_5", Name: "Halo 5", Provider: "halo_5", Status: title.StatusActive})
	if d := reg.Get("halo_5"); d == nil || d.HasCapability(title.CapReplay) {
		t.Fatal("pré-condition : halo_5 doit être enregistré sans la capacité replay")
	}
	src := replayLabelSource{repoRoot: root, titles: reg}

	cat, err := src.LabelCatalogue("halo_infinite")
	if err != nil {
		t.Fatalf("halo_infinite : %v", err)
	}
	if len(cat.Keys) == 0 {
		t.Error("le catalogue de halo_infinite doit nommer des armes")
	}

	for _, slug := range []string{"halo_5", "titre_inconnu", ""} {
		cat, err := src.LabelCatalogue(slug)
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			t.Errorf("%q : err = %v, attendu games.ErrCapabilityNotSupported", slug, err)
		}
		if len(cat.Keys) != 0 || len(cat.VehicleFamilies) != 0 {
			t.Errorf("%q : catalogue non vide rendu avec l'erreur", slug)
		}
	}
}
