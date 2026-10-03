//go:build research

package service

// replay_vehicle_aloft_parc_research_test.go — INSTRUMENT DU CHANTIER « Falcon de Behemoth »
// (2026-10-02) : le verdict de decor, par le CHEMIN DE PRODUCTION (`resolveVehicleScenery`, fonds
// du depot), sur un parc de documents, AVANT (regle de pose seule) et APRES (plus la vie tenue en
// l air a vide), ventile par build. Lecture seule : documents d un dossier, noms de carte d un
// fichier `court|carte|mode` (export de match_registry).
//
//	DECOR_DOCS=<dossier *.json> DECOR_CARTES=<cartes.csv> \
//	  go test -tags research -run TestDecorTenuEnLAirParc -v ./internal/service/
//
// Sortie : une ligne `ALOFT` par vie masquee par la regle, une ligne `PRES` par vie inoccupee qui
// se pose au moins 0,3 m au-dessus de sa naissance sans etre masquee (les quasi-cas), une ligne
// `SIPERDUE` par vie OCCUPEE qui remplirait les conditions 3 a 5 si son occupation n etait pas lue
// (ce que la condition de derive protege), une ligne `BUILD` par build et le bilan `PARC`.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

type carteMatch struct{ carte, mode string }

func decorCartes(t *testing.T, path string) map[string]carteMatch {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]carteMatch{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if p := strings.SplitN(sc.Text(), "|", 3); len(p) == 3 {
			out[p[0]] = carteMatch{carte: p[1], mode: p[2]}
		}
	}
	return out
}

// mesureVie : l elevation de la plus basse station au-dessus de la naissance et la derive maximale
// en plan, sans regarder l occupation (ok faux = aucune station).
func mesureVie(v replay.VehicleTrack, minFrames int) (rise, drift float32, ok bool) {
	bx, by, bz := vehicleBirth(v)
	for _, s := range v.Samples {
		d := float32(math.Hypot(float64(s.X-bx), float64(s.Y-by)))
		if d > drift {
			drift = d
		}
	}
	rest, ok := lowestStand(vehicleHeldPoints(v), minFrames)
	return rest - bz, drift, ok
}

type bilanBuild struct {
	docs, vies, avant, apres int
	parFamille               map[string]int
}

func TestDecorTenuEnLAirParc(t *testing.T) {
	docs, cartesCSV := os.Getenv("DECOR_DOCS"), os.Getenv("DECOR_CARTES")
	if docs == "" || cartesCSV == "" {
		t.Skip("DECOR_DOCS, DECOR_CARTES requis")
	}
	s := sceneryService(t)
	cartes := decorCartes(t, cartesCSV)
	noms, err := filepath.Glob(filepath.Join(docs, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	court8 := regexp.MustCompile(`^[0-9a-f]{8}\.json$`)
	builds := map[string]*bilanBuild{}
	var maxRiseHorsRegle, minRiseRegle, maxDriftRegle float32 = -1e9, 1e9, 0
	minDriftOccupe := float32(1e9)
	var occupesMasques, siPerdue int
	for _, chemin := range noms {
		if !court8.MatchString(filepath.Base(chemin)) {
			continue
		}
		court := strings.TrimSuffix(filepath.Base(chemin), ".json")
		blob, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatal(err)
		}
		var doc replay.ReplayDocument
		if err := json.Unmarshal(blob, &doc); err != nil {
			t.Fatal(err)
		}
		if doc.SchemaVersion == 0 || doc.FrameIntervalMS <= 0 {
			continue
		}
		build := "(sans build)"
		if doc.Coverage.Decoder != nil && doc.Coverage.Decoder.Build != "" {
			build = doc.Coverage.Decoder.Build
		}
		b := builds[build]
		if b == nil {
			b = &bilanBuild{parFamille: map[string]int{}}
			builds[build] = b
		}
		b.docs++
		cm := cartes[court]
		s.resolveVehicleScenery(context.Background(), &doc, court, port.MatchMapKeys{Names: []string{cm.carte}})
		raisons := map[[2]uint32]string{}
		if v := doc.VehicleScenery; v != nil {
			for _, h := range v.Hidden {
				raisons[[2]uint32{h.Slot, h.Gen}] = h.Reason
				b.apres++
				if h.Reason != sceneryReasonAloftUnoccupied {
					b.avant++
				}
			}
		}
		minFrames := standMinFrames(doc.FrameIntervalMS)
		for _, v := range doc.Vehicles {
			if v.Part != "" || len(v.Samples) == 0 {
				continue
			}
			b.vies++
			rise, drift, ok := mesureVie(v, minFrames)
			raison := raisons[[2]uint32{v.Slot, v.Gen}]
			occupe := len(v.Rides) > 0
			switch {
			case raison == sceneryReasonAloftUnoccupied:
				b.parFamille[v.Family]++
				if occupe {
					occupesMasques++
				}
				minRiseRegle = min(minRiseRegle, rise)
				maxDriftRegle = max(maxDriftRegle, drift)
				fmt.Printf("ALOFT %s build=%s carte=%q mode=%q slot=%d gen=%d fam=%s eleve=%.2f derive=%.2f\n",
					court, build, cm.carte, cm.mode, v.Slot, v.Gen, v.Family, rise, drift)
			case occupe:
				minDriftOccupe = min(minDriftOccupe, drift)
				if ok && rise >= aloftMinRiseM && drift <= aloftMaxDriftM {
					siPerdue++
					fmt.Printf("SIPERDUE %s carte=%q slot=%d gen=%d fam=%s eleve=%.2f derive=%.2f\n",
						court, cm.carte, v.Slot, v.Gen, v.Family, rise, drift)
				}
			case ok:
				maxRiseHorsRegle = max(maxRiseHorsRegle, rise)
				if rise >= 0.3 {
					fmt.Printf("PRES %s carte=%q slot=%d gen=%d fam=%s eleve=%.2f derive=%.2f raison=%q\n",
						court, cm.carte, v.Slot, v.Gen, v.Family, rise, drift, raison)
				}
			}
		}
	}
	cles := make([]string, 0, len(builds))
	for k := range builds {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	var tot bilanBuild
	for _, k := range cles {
		b := builds[k]
		tot.docs += b.docs
		tot.vies += b.vies
		tot.avant += b.avant
		tot.apres += b.apres
		fmt.Printf("BUILD %s documents=%d vies=%d masquees_avant=%d masquees_apres=%d tenues_en_l_air=%v\n",
			k, b.docs, b.vies, b.avant, b.apres, b.parFamille)
	}
	fmt.Printf("PARC documents=%d vies=%d masquees_avant=%d masquees_apres=%d occupees_masquees=%d "+
		"si_occupation_perdue=%d eleve_min_regle=%.2f eleve_max_hors_regle=%.2f derive_max_regle=%.2f "+
		"derive_min_occupee=%.2f\n", tot.docs, tot.vies, tot.avant, tot.apres, occupesMasques, siPerdue,
		minRiseRegle, maxRiseHorsRegle, maxDriftRegle, minDriftOccupe)
	if occupesMasques != 0 {
		t.Errorf("%d vie(s) occupee(s) masquee(s)", occupesMasques)
	}
}
