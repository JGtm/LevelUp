package service

// replay_label_source_titre_test.go — la dégradation d'un titre sans catalogue de libellés du rejeu
// (Halo 5) : les services servent sans libellé du titre, et JAMAIS avec ceux d'un autre titre, même
// quand les mappings d'un autre titre sont présents sous la même racine.

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// sourceSansRejeu est la source d'un titre sans rejeu : elle refuse, et compte ses appels.
type sourceSansRejeu struct{ appels int }

func (s *sourceSansRejeu) LabelCatalogue(titleSlug string) (replay.LabelCatalog, error) {
	s.appels++
	return replay.LabelCatalog{}, fmt.Errorf("catalogue de %q : %w", titleSlug, games.ErrCapabilityNotSupported)
}

// TestReplayService_Halo5SansCatalogue : les armes du document gardent leur nom cuit mais ne reçoivent ni
// clé, ni teinte, ni rôle, ni explosion à l'impact — alors que les mappings de Halo Infinite sont bien
// sous la même racine. Une source absente (nil) donne le même service.
func TestReplayService_Halo5SansCatalogue(t *testing.T) {
	cas := map[string]*sourceSansRejeu{"source qui refuse": {}, "source absente": nil}
	for nom, src := range cas {
		t.Run(nom, func(t *testing.T) {
			root := t.TempDir()
			mappingsReels(t, root, title.DefaultSlug)
			artefactArmes(t, root, "halo_5", "a0000001")
			var labels port.ReplayLabelSource
			if src != nil {
				labels = src
			}

			doc, err := NewReplayService("halo_5", root, nil, labels).GetReplay(context.Background(), "a0000001")
			if err != nil {
				t.Fatalf("lecture du rejeu : %v", err)
			}
			if len(doc.WeaponLabels) == 0 {
				t.Fatal("les libellés cuits dans l'artefact doivent rester servis")
			}
			for id, lbl := range doc.WeaponLabels {
				if lbl.Key != "" || lbl.Tint != "" || lbl.Role != "" || lbl.Impact {
					t.Errorf("arme %s : %+v — aucun libellé d'un autre titre ne se pose sur Halo 5", id, lbl)
				}
			}
			if src != nil && src.appels == 0 {
				t.Error("la source du catalogue n'a pas été consultée")
			}
		})
	}
}

// TestReplayService_Halo5SansCatalogue_Vehicules : les familles de châssis gardent leur sprite, composé
// sous le dossier d'assets du titre, mais ne sont ni qualifiées ni nommées par le manifeste d'un autre titre.
func TestReplayService_Halo5SansCatalogue_Vehicules(t *testing.T) {
	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	artefactVehicules(t, root, "halo_5", "a0000002")

	doc, err := NewReplayService("halo_5", root, nil, &sourceSansRejeu{}).GetReplay(context.Background(), "a0000002")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(doc.VehicleLabels) == 0 {
		t.Fatal("les familles employées gardent leur sprite")
	}
	for famille, lbl := range doc.VehicleLabels {
		if lbl.Kind != "" || len(lbl.Text) != 0 {
			t.Errorf("famille %s : %+v — ni nature ni libellé d'un autre titre", famille, lbl)
		}
		if want := "/static/vehicles-assets/halo_5/replay/" + famille + ".png"; lbl.Img != want {
			t.Errorf("sprite de %s = %q, attendu %q (dossier d'assets du titre)", famille, lbl.Img, want)
		}
	}
}

// TestBuildSoloEmpriseBlock_Halo5SansCatalogue : le bloc d'Emprise d'un titre sans rejeu se construit
// sans catalogue d'armes ni de véhicules, sans panique, et consulte la source du titre.
func TestBuildSoloEmpriseBlock_Halo5SansCatalogue(t *testing.T) {
	src := &sourceSansRejeu{}
	block := buildSoloEmpriseBlock(context.Background(), soloEmpriseQuery{
		Page: "timeseries", Player: "P", PlayerXUID: "x1",
		Labels: src, TitleSlug: "halo_5", Locale: "fr",
		Current: []squademprise.Match{{MatchID: "m1"}},
	})
	if block == nil {
		t.Fatal("le bloc d'Emprise doit se servir, dégradé")
	}
	if src.appels == 0 {
		t.Error("la source du catalogue n'a pas été consultée")
	}
}
