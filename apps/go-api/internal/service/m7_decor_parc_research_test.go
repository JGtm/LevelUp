//go:build research

package service

// m7_decor_parc_research_test.go — INSTRUMENT DU LOT M7 (retours du rejeu, 2026-09-24) : le verdict
// de decor de carte, par le CHEMIN DE PRODUCTION (`resolveVehicleScenery`, fonds du depot), sur un
// parc de documents. Lecture seule : documents d un dossier hors data/, noms de carte d un journal
// d export (`<court8> : ... cartes [Nom Nom]`, sortie de `levelup replay-facts-export`).
//
//	M7_DOCS=<dossier *.json> M7_EXPORT=<export.log> \
//	  go test -tags research -run TestM7DecorParc -v ./internal/service/
//
// Sortie : une ligne `M7VIE` par vie de vehicule publiee (L1.3 = les cinq conditions de pose,
// verdict M7), une ligne `M7DOC` par document portant un verdict, et le bilan `M7PARC` : vies que
// L1.3 masquait, vies que M7 masque, vies EN JEU (non candidates) masquees — attendu 0.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

func m7CartesDuJournal(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	re := regexp.MustCompile(`^\s*([0-9a-f]{8}) : .*cartes \[(.*)\]`)
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := re.FindStringSubmatch(sc.Text()); m != nil {
			noms := strings.Fields(m[2])
			out[m[1]] = strings.Join(noms[:len(noms)/2], " ")
		}
	}
	return out
}

func TestM7DecorParc(t *testing.T) {
	docs, export := os.Getenv("M7_DOCS"), os.Getenv("M7_EXPORT")
	if docs == "" || export == "" {
		t.Skip("M7_DOCS, M7_EXPORT requis")
	}
	s := sceneryService(t)
	cartes := m7CartesDuJournal(t, export)
	noms, err := filepath.Glob(filepath.Join(docs, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var nDocs, l13, masques, enJeuMasques, candidates, zoneInconnue int
	for _, chemin := range noms {
		court := strings.TrimSuffix(filepath.Base(chemin), ".json")
		blob, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatal(err)
		}
		var doc replay.ReplayDocument
		if err := json.Unmarshal(blob, &doc); err != nil {
			t.Fatal(err)
		}
		nDocs++
		debut := time.Now()
		keys := port.MatchMapKeys{Names: []string{cartes[court]}}
		s.resolveVehicleScenery(context.Background(), &doc, court, keys)
		duree := time.Since(debut)
		caches := map[[2]uint32]string{}
		if v := doc.VehicleScenery; v != nil {
			for _, h := range v.Hidden {
				caches[[2]uint32{h.Slot, h.Gen}] = h.Reason
			}
			candidates += v.Candidates
			zoneInconnue += v.ZoneUnknown
			fmt.Printf("M7DOC %s carte=%q zone=%s sol=%s candidates=%d masquees=%d dans_la_zone=%d zone_inconnue=%d duree=%v\n",
				court, cartes[court], v.Zone, v.Floor, v.Candidates, len(v.Hidden), v.InPlayArea, v.ZoneUnknown, duree)
		}
		for _, veh := range doc.Vehicles {
			if len(veh.Samples) == 0 {
				continue
			}
			pose := vehicleIsPosedOnly(veh)
			raison, cache := caches[[2]uint32{veh.Slot, veh.Gen}]
			if pose {
				l13++
			}
			if cache {
				masques++
				if !pose {
					enJeuMasques++
				}
			}
			fmt.Printf("M7VIE %s slot=%d gen=%d fam=%s L13=%v M7=%v raison=%s\n",
				court, veh.Slot, veh.Gen, veh.Family, pose, cache, raison)
		}
	}
	fmt.Printf("M7PARC documents=%d L13=%d M7=%d en_jeu_masques=%d candidates=%d zone_inconnue=%d\n",
		nDocs, l13, masques, enJeuMasques, candidates, zoneInconnue)
	m7SiPosees(t, s, docs, cartes)
	if enJeuMasques != 0 {
		t.Errorf("%d vie(s) en jeu masquee(s)", enJeuMasques)
	}
}

// m7SiPosees : TEMOIN DE ROBUSTESSE. Chaque vie de vehicule EN JEU (non candidate, avec au moins
// un echantillon) est ramenee a une POSE SEULE a son premier echantillon, puis passee a la regle :
// combien seraient masquees si personne n y avait touche ? Attendu 0 hors des vies aberrantes.
func m7SiPosees(t *testing.T, s *replayService, docs string, cartes map[string]string) {
	t.Helper()
	noms, _ := filepath.Glob(filepath.Join(docs, "*.json"))
	total, horsZone := 0, 0
	for _, chemin := range noms {
		court := strings.TrimSuffix(filepath.Base(chemin), ".json")
		blob, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatal(err)
		}
		var doc replay.ReplayDocument
		if err := json.Unmarshal(blob, &doc); err != nil {
			t.Fatal(err)
		}
		var vies []replay.VehicleTrack
		for _, v := range doc.Vehicles {
			if len(v.Samples) == 0 || vehicleIsPosedOnly(v) || v.Family == "" {
				continue
			}
			p := v.Samples[0]
			p.T = 0
			v.T0, v.End, v.Rides, v.Samples = 0, replay.VehicleEndFilmEnd, nil, []replay.VehicleSample{p}
			vies = append(vies, v)
		}
		if len(vies) == 0 {
			continue
		}
		doc.Vehicles = vies
		keys := port.MatchMapKeys{Names: []string{cartes[court]}}
		s.resolveVehicleScenery(context.Background(), &doc, court, keys)
		total += len(vies)
		for _, h := range doc.VehicleScenery.Hidden {
			horsZone++
			fmt.Printf("M7SIPOSEE %s slot=%d gen=%d raison=%s\n", court, h.Slot, h.Gen, h.Reason)
		}
		if doc.VehicleScenery.Zone != sceneryZoneMap {
			fmt.Printf("M7SIPOSEE %s zone=%s (%d vies non jugees)\n", court, doc.VehicleScenery.Zone, len(vies))
		}
	}
	fmt.Printf("M7SIPOSEES vies_en_jeu=%d masquees_si_posees=%d\n", total, horsZone)
}
