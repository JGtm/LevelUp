package squadagg

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// sourceDeTest lit les catalogues sous `root`, comme le câblage, sans la porte de capacité.
type sourceDeTest struct{ root string }

func (s sourceDeTest) LabelCatalogue(titleSlug string) (replay.LabelCatalog, error) {
	return replaylabels.Catalogue(s.root, titleSlug)
}

// sourceSansRejeu : la source d'un titre sans rejeu, qui refuse.
type sourceSansRejeu struct{}

func (sourceSansRejeu) LabelCatalogue(string) (replay.LabelCatalog, error) {
	return replay.LabelCatalog{}, games.ErrCapabilityNotSupported
}

// sourceEnPanne : un catalogue illisible.
type sourceEnPanne struct{}

func (sourceEnPanne) LabelCatalogue(string) (replay.LabelCatalog, error) {
	return replay.LabelCatalog{}, errors.New("manifeste illisible")
}

// TestVehicleFamilyLabels — seules les familles que le manifeste du titre qualifie portent un
// libellé, dans la langue de la requête (français par défaut, anglais sur demande) ; les noms
// propres du jeu (Warthog) n'en ont pas, le client les affiche depuis leur clé.
func TestVehicleFamilyLabels(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	src := sourceDeTest{root: root}
	fr := VehicleFamilyLabels(ctx, src, "halo_infinite", "fr")
	if fr["tourelle_fixe"] != "Tourelle fixe" {
		t.Errorf("libellé FR = %q, attendu « Tourelle fixe »", fr["tourelle_fixe"])
	}
	if _, nomme := fr["warthog"]; nomme {
		t.Errorf("Warthog est un nom propre, sans libellé : %v", fr)
	}
	if en := VehicleFamilyLabels(ctx, src, "halo_infinite", "en"); en["tourelle_fixe"] != "Fixed turret" {
		t.Errorf("libellé EN = %q, attendu « Fixed turret »", en["tourelle_fixe"])
	}
}

// TestCatalogues_DegradentSansCatalogue : source absente, titre vide ou inconnu, titre sans rejeu
// (Halo 5) et catalogue illisible rendent une map vide non nil — jamais un panic, jamais les noms
// d'un autre titre.
func TestCatalogues_DegradentSansCatalogue(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cas := map[string]struct {
		src   port.ReplayLabelSource
		titre string
	}{
		"source absente":      {nil, "halo_infinite"},
		"titre vide":          {sourceDeTest{root: root}, ""},
		"titre inconnu":       {sourceDeTest{root: root}, "titre_inconnu"},
		"titre sans rejeu":    {sourceSansRejeu{}, "halo_5"},
		"catalogue illisible": {sourceEnPanne{}, "halo_infinite"},
	}
	for nom, c := range cas {
		if got := VehicleFamilyLabels(ctx, c.src, c.titre, "fr"); got == nil || len(got) != 0 {
			t.Errorf("VehicleFamilyLabels, %s : %v, attendu une map vide non nil", nom, got)
		}
		if got := WeaponCatalog(ctx, c.src, c.titre, "fr"); got == nil || len(got) != 0 {
			t.Errorf("WeaponCatalog, %s : %v, attendu une map vide non nil", nom, got)
		}
	}
}
