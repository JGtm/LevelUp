//go:build research

package filmcorpus

// armes_vehicule_gate_research_test.go — LA GATE G4 DU LOT M4a (retours du rejeu 2026-09-23), cote
// hote.
//
// Instrument de recherche deplace du paquet `replay` de la brique (plan D2, item 2.11) : la gate
// G4 compte la part des tirs d arme de VEHICULE (moitie basse nulle) dont l arme a, au registre du
// titre (`vehicle_weapons.toml`, chargeur de LevelUp), un style ET un son (ou un silence decide) ;
// les tags `[[unknown]]` sont comptes a part. Les gates G3 restent dans la brique
// (`TestRRM4AGates`), qui n'a pas besoin du registre. Meme variable d'entree :
//
//	RR_M4A_DOCS=<dossier de documents> go test -tags research -count=1 \
//	  -run '^TestRRM4AGatesG4$' -v ./internal/filmcorpus/

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/mappings"
)

func TestRRM4AGatesG4(t *testing.T) {
	dir := os.Getenv("RR_M4A_DOCS")
	if dir == "" {
		t.Skip("instrument de gate : RR_M4A_DOCS requis")
	}
	reg := registreDesArmesDeVehicule(t)
	noms, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(noms)
	g4 := map[string]int{}
	for _, n := range noms {
		raw, err := os.ReadFile(n) //nolint:gosec // instrument de mesure
		if err != nil {
			t.Fatal(err)
		}
		var doc replay.ReplayDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		compterArmesDeVehicule(doc, reg, g4)
	}
	t.Logf("G4 : tirs d arme de vehicule %d ; registre complet %d ; silence decide %d ; "+
		"inconnu motive %d ; hors registre %d", g4["total"], g4["complet"], g4["silence"],
		g4["inconnu"], g4["hors"])
}

func compterArmesDeVehicule(doc replay.ReplayDocument, reg *mappings.VehicleWeaponSet, n map[string]int) {
	inconnus := reg.Unknown()
	for _, s := range doc.Shots {
		if len(s.Weapon) != 18 || !strings.HasSuffix(s.Weapon, "00000000") {
			continue
		}
		n["total"]++
		tag := s.Weapon[2:10]
		w, ok := reg.Weapon(tag)
		switch {
		case ok && w.Sound != "":
			n["complet"]++
		case ok:
			n["silence"]++
		case inconnus[tag] != "":
			n["inconnu"]++
		default:
			n["hors"]++
		}
	}
}
