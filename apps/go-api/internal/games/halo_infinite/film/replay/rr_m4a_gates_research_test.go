//go:build research

package replay

// rr_m4a_gates_research_test.go — LES GATES G3 ET G4 DU LOT M4a (retours du rejeu 2026-09-23),
// portes en test `research` depuis les instruments de l annexe (`tirs_enfants.mjs`, `sweep_w.mjs`).
//
// Il lit un dossier de DOCUMENTS (la sortie de `TestRRM4AParcDepuisLesFaits`, base ou branche —
// jamais `data/`), et mesure :
//
//	G3  LE BON PORTEUR, PAR UNE PREUVE QUI NE DOIT RIEN A LA REGLE (revue adverse du lot, F5). L ecart
//	    tir -> porteur est TAUTOLOGIQUE apres le lot : le tir est pose A la position du porteur que
//	    la regle a choisi (0,0 m par construction) ; il reste mesure (`G3-annexe`), comme chiffre
//	    AVANT sur les documents de base (regle de l instrument de l annexe : premiere vie du slot +1
//	    puis +2 qui porte des echantillons et couvre l instant), jamais comme preuve. LA PREUVE est
//	    `G3-embarquement` : pour chaque episode d artilleur REPORTE sur un porteur
//	    (`rides[].turret`), le DERNIER POINT DU BIPEDE de l artilleur avant l episode contre la
//	    position du porteur a l entree — le joueur monte la ou le vehicule est. Ni le slot, ni la
//	    naissance, ni le tir n y entrent. Cible : mediane <= 2 m. `G3-naissance` (piece et porteur
//	    nes au meme point, au meme instant) est rapporte comme CONTROLE : depuis la reprise, la
//	    regle l exige, il n est donc plus independant.
//	G4  la part des tirs d arme de VEHICULE (moitie basse nulle) dont l arme a, au registre du
//	    titre, un style ET un son (ou un silence decide) ; les tags `[[unknown]]` sont comptes a
//	    part, avec leur raison.
//
//	RR_M4A_DOCS=<dossier de documents> go test -tags research -count=1 \
//	  -run '^TestRRM4AGates$' -v ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"fmt"
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
	var ecarts, embarquements, naissances []float64
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
		embarquements = append(embarquements, rrG3Embarquements(doc)...)
		naissances = append(naissances, rrG3Naissances(doc)...)
		rrG4Compter(doc, reg, g4)
	}
	t.Logf("G3-embarquement (PREUVE) : %s", rrQuantiles(embarquements))
	t.Logf("G3-naissance (controle, exige par la regle) : %s", rrQuantiles(naissances))
	t.Logf("G3-annexe (tir -> porteur, tautologique apres le lot) : %s", rrQuantiles(ecarts))
	t.Logf("G4 : tirs d arme de vehicule %d ; registre complet %d ; silence decide %d ; "+
		"inconnu motive %d ; hors registre %d", g4["total"], g4["complet"], g4["silence"],
		g4["inconnu"], g4["hors"])
}

// rrQuantiles rend « n, mediane, p90, max » d une serie en metres.
func rrQuantiles(v []float64) string {
	if len(v) == 0 {
		return "n = 0"
	}
	sort.Float64s(v)
	q := func(p float64) float64 { return v[int(p*float64(len(v)-1))] }
	return fmt.Sprintf("n = %d ; mediane %.1f m, p90 %.1f m, max %.1f m", len(v), q(0.5), q(0.9), q(1))
}

// rrG3Embarquements rend, pour chaque episode d artilleur reporte sur un porteur, l ecart entre le
// dernier point du bipede de l artilleur avant l episode et la position du porteur a l entree.
func rrG3Embarquements(doc ReplayDocument) []float64 {
	var out []float64
	for _, v := range doc.Vehicles {
		for _, r := range v.Rides {
			if r.Turret == nil {
				continue
			}
			b, ok := rrDernierPointAvant(doc.Tracks, r.Slot, r.T0)
			if !ok {
				continue
			}
			x, y, ok := vehiclePosAt(v, r.T0)
			if !ok {
				continue
			}
			out = append(out, math.Hypot(float64(x-b.X), float64(y-b.Y)))
		}
	}
	return out
}

// rrDernierPointAvant rend le dernier point publie d un slot strictement avant la frame `t`.
func rrDernierPointAvant(tracks []Track, slot uint32, t int) (Point, bool) {
	var best Point
	trouve := false
	for _, tr := range tracks {
		if tr.Slot != slot {
			continue
		}
		for _, p := range tr.Points {
			if p.T < t && (!trouve || p.T > best.T) {
				best, trouve = p, true
			}
		}
	}
	return best, trouve
}

// rrG3Naissances rend l ecart entre la naissance d une piece posee et celle de son porteur.
func rrG3Naissances(doc ReplayDocument) []float64 {
	var out []float64
	for _, v := range doc.Vehicles {
		if v.Carrier == nil || v.Spawn == nil {
			continue
		}
		for _, c := range doc.Vehicles {
			if c.Slot == v.Carrier.Slot && c.Gen == v.Carrier.Gen && c.Spawn != nil {
				out = append(out, math.Hypot(float64(v.Spawn.X-c.Spawn.X), float64(v.Spawn.Y-c.Spawn.Y)))
			}
		}
	}
	return out
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
