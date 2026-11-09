package equipmentusage_test

// jumeau_film_test.go — LA COPIE DU DÉCODEUR DE FILM RESTE ÉGALE À CE PAQUET (plan D2, item 2.8 ;
// règle 6 de CLAUDE.md).
//
// Le décodeur de film (`film/replay/usage_summary_bilan.go`) porte sa propre liste des familles du
// bilan et son propre prédicat de pièce engendrée : il ne peut importer aucun paquet de LevelUp.
// Deux copies d'une même connaissance divergent en silence si rien ne les compare ; ce test les
// compare. Mutation qui doit le faire rougir : ajouter, retirer ou réordonner une famille d'un seul
// côté, ou ajouter une famille à pièce engendrée d'un seul côté.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/domain/equipmentusage"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

func TestJumeauFilm_FamillesDuBilanIdentiques(t *testing.T) {
	ici := equipmentusage.EquipmentOutcomeFamilies()
	film := replay.EquipmentOutcomeFamilies()
	if !slices.Equal(ici, film) {
		t.Fatalf("familles du bilan divergentes (ordre compris) :\n  domain/equipmentusage %v\n  film/replay           %v", ici, film)
	}
}

func TestJumeauFilm_PieceEngendreeIdentique(t *testing.T) {
	// Toutes les familles connues des deux côtés, plus des clés hors bilan du manifeste et une clé
	// vide : le prédicat doit répondre pareil partout, « non » compris.
	connues := append(equipmentusage.EquipmentOutcomeFamilies(), replay.EquipmentOutcomeFamilies()...)
	connues = append(connues, equipmentusage.EquipmentFamilyWall, equipmentusage.EquipmentFamilyPowerupCamo,
		equipmentusage.EquipmentFamilyPowerupOvershield, replay.UsageFamilyWallKey,
		"grapple", "thruster", "repulsor", "grenade_frag", "")
	vraies := 0
	for _, f := range connues {
		ici, film := equipmentusage.UsageFamilySpawnsPiece(f), replay.UsageFamilySpawnsPiece(f)
		if ici != film {
			t.Errorf("UsageFamilySpawnsPiece(%q) : domain/equipmentusage %v, film/replay %v", f, ici, film)
		}
		if ici {
			vraies++
		}
	}
	if vraies == 0 {
		t.Fatal("aucune famille à pièce engendrée parmi les familles connues : le test ne compare plus rien")
	}
}

func TestJumeauFilm_CleDuMurIdentique(t *testing.T) {
	if replay.UsageFamilyWallKey != equipmentusage.EquipmentFamilyWall {
		t.Fatalf("clé du mur : film/replay %q, domain/equipmentusage %q", replay.UsageFamilyWallKey, equipmentusage.EquipmentFamilyWall)
	}
}
