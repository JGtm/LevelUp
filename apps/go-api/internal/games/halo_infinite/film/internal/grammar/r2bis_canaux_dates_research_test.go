//go:build research

package grammar

// r2bis_canaux_dates_research_test.go — INSTRUMENT du lot R2-bis (plan de suite de l audit du
// decodeur, 2026-09-29, decouverte 2 du lot R2) : ce que les consommateurs du marcheur delta bipede
// acceptent et publient, canal par canal, sur UN film. Il est joue AVANT puis APRES la conversion des
// consommateurs au filtre de generation DATE, et les deux sorties se comparent ligne a ligne.
//
// Par canal : nombre d elements publies, compteurs du balayage (dont `Records`, les records du
// marcheur recus), empreinte SHA-256 de la sortie. Plus, pour le marcheur lui-meme : les records que
// le filtre atemporel accepte et que le filtre date refuse (en-tetes d un corps anterieurs a son
// record de creation), par generation. Saute sans les variables d environnement ; jamais en CI.
//
//	R2BIS_FILM=<parc>/data/cache/film_chunks/084a804d R2BIS_MAP="Fortitude Heavies" \
//	  R2BIS_OUT=<fichier hors data> go test ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run R2bisCanauxDates -count=1 -v

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

func TestR2bisCanauxDates(t *testing.T) {
	dir, carte, sortie := os.Getenv("R2BIS_FILM"), os.Getenv("R2BIS_MAP"), os.Getenv("R2BIS_OUT")
	if dir == "" || carte == "" || sortie == "" {
		t.Skip("R2BIS_FILM / R2BIS_MAP / R2BIS_OUT absents : instrument du lot R2-bis saute")
	}
	fc, wr := r2bisContexte(t, dir, carte)
	var lignes []string
	ligne := func(canal string, n int, stats any, v any) {
		// JSON et non %+v : les sorties portent des pointeurs (`InventoryDelta.Mag`...), dont %+v
		// imprime l ADRESSE — une empreinte qui changerait d un processus a l autre.
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s : %v", canal, err)
		}
		lignes = append(lignes, fmt.Sprintf("%s\tn=%d\tsha=%x\tstats=%+v", canal, n, sha256.Sum256(b), stats))
	}
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanBipedPositions(fc, opt)
	if err != nil {
		t.Fatal(err)
	}
	ligne("positions", len(pos), nil, pos)
	naissance := map[uint32]uint64{}
	for _, p := range pos {
		if at, ok := naissance[p.Slot]; !ok || p.TimestampUS < at {
			naissance[p.Slot] = p.TimestampUS
		}
	}
	bornAt := func(s uint32) (uint64, bool) { at, ok := naissance[s]; return at, ok }
	hw, hwSt, err := ScanHeldWeaponChanges(fc, nil)
	r2bisErr(t, "heldWeaponChanges", err)
	ligne("heldWeaponChanges", len(hw), hwSt, hw)
	inv, invSt, err := ScanInventoryDeltas(fc)
	r2bisErr(t, "inventoryDeltas", err)
	ligne("inventoryDeltas", len(inv), invSt, inv)
	ar, arSt, err := ScanAbilityRanks(fc)
	r2bisErr(t, "abilityRanks", err)
	ligne("abilityRanks", len(ar), arSt, ar)
	eq, eqSt, err := ScanEquipmentChanges(fc, bornAt)
	r2bisErr(t, "equipmentChanges", err)
	ligne("equipmentChanges", len(eq), eqSt, eq)
	camo, camoSt, err := ScanCamoStates(fc)
	r2bisErr(t, "camoStates", err)
	ligne("camoStates", len(camo), camoSt, camo)
	gr, grSt, err := ScanGrappleReads(fc)
	r2bisErr(t, "grappleReads", err)
	ligne("grappleReads", len(gr), grSt, gr)
	imp, impSt, err := ScanAbilityImpulses(fc)
	r2bisErr(t, "abilityImpulses", err)
	ligne("abilityImpulses", len(imp), impSt, imp)
	ch, chSt, err := ScanAbilityCharges(fc)
	r2bisErr(t, "abilityCharges", err)
	ligne("abilityCharges", len(ch), chSt, ch)
	ue, err := ScanUnitEquipment(fc)
	r2bisErr(t, "unitEquipment", err)
	ligne("unitEquipment", len(ue), nil, ue)
	aims, err := ScanBipedAimOnly(fc)
	r2bisErr(t, "aimOnly", err)
	ligne("aimOnly", len(aims), nil, aims)
	lignes = append(lignes, r2bisMarcheur(t, fc)...)
	if err := os.WriteFile(sortie, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, l := range lignes {
		t.Log(l)
	}
}

// r2bisContexte charge le film et son contexte sur la carte nommee, comme la cuisson.
func r2bisContexte(t *testing.T, dir, carte string) (*FilmContext, profile.Vec3Range) {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(racine, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Fatal(err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return NewFilmContextForMap(film, &e, nil), e.Range()
}

func r2bisErr(t *testing.T, canal string, err error) {
	t.Helper()
	if err != nil {
		t.Logf("%s : %v", canal, err)
	}
}

// r2bisMarcheur rend, pour le marcheur de production ET pour une marche au filtre atemporel, le
// nombre de records ancres, et ceux que la garde datee refuse (par generation).
func r2bisMarcheur(t *testing.T, fc *FilmContext) []string {
	t.Helper()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatal(err)
	}
	gens := fc.GenerationsVivantes()
	prod, prodAnterieurs := 0, map[uint32]int{}
	walkDeltaBipedRecords(fc, fc.ChunkNumbers(), fc.BipedSlots(), lay, func(r deltaBipedRecord) {
		prod++
		if !gens.A(r.Packet.TimestampUS).Accepte(r.Vie()) {
			prodAnterieurs[r.Gen]++
		}
	})
	atemp, anterieurs := 0, map[uint32]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			walkDeltaBipedPayload(pk.Payload(data), fc.BipedSlots(), lay, gens, func(r deltaBipedRecord) {
				atemp++
				if !gens.A(pk.TimestampUS).Accepte(r.Vie()) {
					anterieurs[r.Gen]++
				}
			})
		}
	}
	return []string{
		fmt.Sprintf("marcheur.production\trecords=%d\tanterieursACreation=%s", prod, r2bisParGen(prodAnterieurs)),
		fmt.Sprintf("marcheur.atemporel\trecords=%d\tanterieursACreation=%s", atemp, r2bisParGen(anterieurs)),
	}
}

func r2bisParGen(m map[uint32]int) string {
	var cles []int
	total := 0
	for g, n := range m {
		cles = append(cles, int(g))
		total += n
	}
	sort.Ints(cles)
	var b strings.Builder
	fmt.Fprintf(&b, "%d", total)
	for _, g := range cles {
		fmt.Fprintf(&b, " gen%d:%d", g, m[uint32(g)]) //nolint:gosec // generation sur 2 bits
	}
	return b.String()
}
