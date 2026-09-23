//go:build research

package replay

// rr_m4a_gates_research_test.go — LES GATES G3 ET G4 DU LOT M4a (retours du rejeu 2026-09-23),
// portes en test `research` depuis les instruments de l annexe (`tirs_enfants.mjs`, `sweep_w.mjs`).
//
// Il lit un dossier de DOCUMENTS (la sortie de `TestRRM4AParcDepuisLesFaits`, base ou branche —
// jamais `data/`), et mesure :
//
//	G3  l ECART entre un tir de TOURELLE et le vehicule qui la porte, en metres. Un tir de tourelle
//	    est un tir dont la vie `v` est une piece montee (chassis de la table, ou `part`), ou dont
//	    l episode du tireur a ete reporte d une tourelle (`rides[].turret`). Le porteur est la vie
//	    `v` elle-meme quand elle n est pas une piece ; `carrier` quand la piece le nomme ; a defaut
//	    (documents d avant le lot), la regle de l instrument de l annexe : la premiere vie du slot
//	    +1 puis +2 qui porte des echantillons et couvre l instant.
//	G4  la part des tirs d arme de VEHICULE (moitie basse nulle) dont l arme a, au registre du
//	    titre, un style ET un son (ou un silence decide) ; les tags `[[unknown]]` sont comptes a
//	    part, avec leur raison.
//
//	RR_M4A_DOCS=<dossier de documents> go test -tags research -count=1 \
//	  -run '^TestRRM4AGates$' -v ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

func TestRRM4AGates(t *testing.T) {
	dir := os.Getenv("RR_M4A_DOCS")
	if dir == "" {
		t.Skip("instrument de gate : RR_M4A_DOCS requis")
	}
	reg, err := mappings.LoadVehicleWeaponsFromFile(filepath.Join(repoRootForTest(t), "config",
		"titles", "halo_infinite", "mappings", "vehicle_weapons.toml"))
	if err != nil {
		t.Fatal(err)
	}
	noms, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(noms)
	var ecarts []float64
	g4 := map[string]int{}
	for _, n := range noms {
		raw, err := os.ReadFile(n) //nolint:gosec // instrument de mesure
		if err != nil {
			t.Fatal(err)
		}
		var doc ReplayDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		ecarts = append(ecarts, rrG3Ecarts(doc)...)
		rrG4Compter(doc, reg, g4)
	}
	sort.Float64s(ecarts)
	q := func(p float64) float64 {
		if len(ecarts) == 0 {
			return math.NaN()
		}
		return ecarts[int(p*float64(len(ecarts)-1))]
	}
	t.Logf("G3 : %d tirs de tourelle ; ecart au porteur mediane %.1f m, p90 %.1f m, max %.1f m",
		len(ecarts), q(0.5), q(0.9), q(1))
	t.Logf("G4 : tirs d arme de vehicule %d ; registre complet %d ; silence decide %d ; "+
		"inconnu motive %d ; hors registre %d", g4["total"], g4["complet"], g4["silence"],
		g4["inconnu"], g4["hors"])
}

// rrG3Ecarts rend l ecart au porteur de chaque tir de tourelle d un document.
func rrG3Ecarts(doc ReplayDocument) []float64 {
	var out []float64
	for _, s := range doc.Shots {
		if s.Vehicle == nil {
			continue
		}
		vie := rrVieQuiCouvre(doc.Vehicles, *s.Vehicle, s.T)
		if vie == nil || !(rrTirDeTourelle(*vie, s) || rrTirDUnePiecePortee(doc.Vehicles, *vie, s)) {
			continue
		}
		porteur := rrPorteur(doc.Vehicles, *vie, s.T)
		if porteur == nil {
			continue
		}
		x, y, ok := vehiclePosAt(*porteur, s.T)
		if !ok {
			continue
		}
		out = append(out, math.Hypot(float64(x-s.X), float64(y-s.Y)))
	}
	return out
}

func rrVieQuiCouvre(vies []VehicleTrack, slot uint32, t int) *VehicleTrack {
	for i := range vies {
		if vies[i].Slot == slot && t >= vies[i].T0-5 && t <= vies[i].T1Max+5 {
			return &vies[i]
		}
	}
	return nil
}

func rrTirDeTourelle(vie VehicleTrack, s Shot) bool {
	if _, piece := vehicleTurretOf(vie); piece || vie.Part != "" {
		return true
	}
	for _, r := range vie.Rides {
		if r.Slot == s.Slot && r.Turret != nil && s.T >= r.T0 && s.T <= r.T1 {
			return true
		}
	}
	return false
}

// rrTirDUnePiecePortee : le tir est pose sur un porteur (`v` = le chassis) alors que l episode du
// tireur est reste sur une de ses pieces (porteur non pilotable, occupant deja a bord).
func rrTirDUnePiecePortee(vies []VehicleTrack, porteur VehicleTrack, s Shot) bool {
	for _, p := range vies {
		if p.Carrier == nil || p.Carrier.Slot != porteur.Slot || p.Carrier.Gen != porteur.Gen {
			continue
		}
		for _, r := range p.Rides {
			if r.Slot == s.Slot && s.T >= r.T0 && s.T <= r.T1 {
				return true
			}
		}
	}
	return false
}

func rrPorteur(vies []VehicleTrack, vie VehicleTrack, t int) *VehicleTrack {
	if _, piece := vehicleTurretOf(vie); !piece && vie.Part == "" {
		return &vie
	}
	if vie.Carrier != nil {
		for i := range vies {
			if vies[i].Slot == vie.Carrier.Slot && vies[i].Gen == vie.Carrier.Gen {
				return &vies[i]
			}
		}
	}
	for d := uint32(1); d <= 2; d++ {
		for i := range vies {
			v := vies[i]
			if v.Slot == vie.Slot+d && len(v.Samples) > 0 && t >= v.T0 && t <= v.T1Max {
				return &vies[i]
			}
		}
	}
	return nil
}

func rrG4Compter(doc ReplayDocument, reg *mappings.VehicleWeaponSet, n map[string]int) {
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
