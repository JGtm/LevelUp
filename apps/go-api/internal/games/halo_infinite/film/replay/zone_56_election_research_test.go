//go:build research

package replay

// zone_56_election_research_test.go — LOT 5.6, POINT 3 : L ELECTION DE PRODUCTION, SUR LE FILM.
//
// DEPLACEMENT PUR depuis `zone_56_camp_research_test.go` (le ratchet de taille a refuse le
// fichier a 547 lignes). La correlation du point 2 vit la-bas ; ici vit la verification sur
// piece du code LIVRE — `electZoneCapturer` joue sur un vrai film, sans cuisson ni base.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

//
// La correlation ci-dessus essaie tous les couples ; la PRODUCTION, elle, recoit le canal de
// propriete deja elu par `pairOwnerSlots` et doit en deduire le POUSSEUR seule
// (`electZoneCapturer`). Cette passe joue l election de production pour CHAQUE reference
// possible et colle ce qu elle rend : c est la verification sur piece du code livre, sans
// cuisson ni base.
//
//	ZONE56P_FILM=<abs>/data/cache/film_chunks/396cfc92 \
//	  go test -tags=research -count=1 -v -run '^TestZone56ElectionDeProduction$' ./...replay/

// TestZone56ElectionDeProduction — CE QUE `electZoneCapturer` REND SUR LE FILM.
func TestZone56ElectionDeProduction(t *testing.T) {
	dir := os.Getenv("ZONE56P_FILM")
	if dir == "" {
		t.Skip("instrument de mesure : ZONE56P_FILM requis")
	}
	fc := zone56Contexte(t, dir, os.Getenv("ZONE56P_CARTE"))
	sc, err := grammar.ScanManagedProperties(fc)
	if err != nil {
		t.Fatalf("balayage ti=13 : %v", err)
	}
	ser := zoneSeriesOf(sc.Reads, zoneCtx{origin: 0, step: 1000, frames: 1 << 30})
	camps := make([]uint32, 0, 8)
	for _, sl := range sortedZoneSlots(ser.ownerChained) {
		if zoneCampLike(ser.ownerChained[sl]) {
			camps = append(camps, sl)
		}
	}
	t.Logf("%s : %d canaux a valeurs de camp (serie CHAINEE) : %v",
		filepath.Base(dir), len(camps), camps)
	t.Log("| jauge | reference (proprietaire) | POUSSEUR elu | rampes | camp LU | camp par REPLI |")
	t.Log("|---:|---:|---|---:|---:|---:|")
	for _, sl := range sortedZoneSlots(ser.gauge) {
		if !zone56JaugeReelle(ser.gauge[sl]) {
			continue
		}
		ramps := findZoneRamps(sl, ser.gauge[sl])
		for _, ownerSlot := range camps {
			owner := ser.owner[ownerSlot]
			capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{
				owner: owner, ownerSlot: ownerSlot, win: 20,
			})
			if len(capt) == 0 {
				continue
			}
			lu, repli := zone56CompterSources(ramps, owner, capt)
			t.Logf("| %d | %d | %s | %d | **%d** | %d |",
				sl, ownerSlot, zone56SlotDe(ser, capt), len(ramps), lu, repli)
		}
	}
}

// zone56CompterSources compte, sur les rampes d une zone, les camps LUS et ceux qui retombent
// sur le repli. Le denominateur est le nombre de rampes.
func zone56CompterSources(ramps []zoneRamp, owner, capt []zoneSample) (int, int) {
	lu, repli := 0, 0
	for _, r := range ramps {
		if team, ok := zoneRampCapturerRead(capt, r, nil); ok {
			if team != nil {
				lu++
			}
			continue
		}
		if zoneRampCapturerDeduit(r, owner, nil, 20, nil) != nil {
			repli++
		}
	}
	return lu, repli
}

// zone56SlotDe retrouve le slot d une serie elue — le rapport doit NOMMER le canal.
func zone56SlotDe(ser zoneSeries, capt []zoneSample) string {
	for _, sl := range sortedZoneSlots(ser.ownerChained) {
		if len(ser.ownerChained[sl]) == len(capt) && len(capt) > 0 &&
			ser.ownerChained[sl][0] == capt[0] {
			return fmt.Sprintf("slot %d", sl)
		}
	}
	return "slot inconnu"
}
