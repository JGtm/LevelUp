//go:build research

package replay

// socle_hors_carte_research_test.go — L INSTRUMENT DU POINT 11 (2026-10-08) : OU UNE ARME DE
// SOCLE EST-ELLE PRISE quand son record de creation la pose hors de la carte ?
//
// Il relit les FAITS PERSISTES d un film (aucun octet de film decode) et met cote a cote, pour
// une famille d arme :
//
//	creations      les records de creation `ti=42` de la famille (cle, instant, position) ;
//	pistes         les echantillons delta de position de ces cles, s il y en a ;
//	ramassages     les ramassages natifs (`biped_pickup`) de la famille, et la position du
//	               ramasseur a cet instant, lue au nuage des bipedes.
//
// LECTURE SEULE, garde par trois variables — saute partout ailleurs, CI comprise :
//
//	SOCLE_FAITS=<repo>/data/cache/film_facts/halo_infinite/5c38f581.filmfacts.bin \
//	SOCLE_CARTE=prism SOCLE_FAMILLE=b533957e \
//	  go test -tags research ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestSocleHorsCarte$' -v -count=1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

func TestSocleHorsCarte(t *testing.T) {
	chemin := os.Getenv("SOCLE_FAITS")
	if chemin == "" {
		t.Skip("SOCLE_FAITS absent : mesure sautee")
	}
	entry := mapQuantEntryFromEnv(t, "SOCLE_CARTE", "SOCLE_BORNES")
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	fam64, err := strconv.ParseUint(os.Getenv("SOCLE_FAMILLE"), 16, 32)
	if err != nil {
		t.Fatalf("SOCLE_FAMILLE : %v", err)
	}
	in := f.Facts.FilmInputs
	scan := in.Pads.Weapons
	cles := map[uint32]bool{}
	for _, c := range scan.Creations {
		if w, ok := gwPadsIdentity(c); ok && w == uint32(fam64) {
			cles[c.Slot] = true
			t.Logf("CREATION slot %d gen %d  t %d us  (%.2f, %.2f, %.2f)", c.Slot, c.Gen,
				c.TimestampUS, c.X, c.Y, c.Z)
		}
	}
	for _, tr := range scan.Tracks {
		if !cles[tr.Slot] {
			continue
		}
		for _, p := range tr.Pts {
			t.Logf("PISTE slot %d gen %d  t %d us  (%.2f, %.2f, %.2f)", tr.Slot, tr.Gen,
				p.TimestampUS, p.X, p.Y, p.Z)
		}
	}
	pos := append([]grammar.BipedPosition(nil), in.Positions...)
	sort.Slice(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	for _, pk := range in.Pickups {
		if pk.CatalogID != uint32(fam64) {
			continue
		}
		p, ok := positionLaPlusProche(pos, pk.Slot, pk.TimestampUS)
		t.Logf("RAMASSAGE slot %d  t %d us  classe %d  ramasseur %v (%.2f, %.2f, %.2f) a %d us", pk.Slot,
			pk.TimestampUS, pk.Class, ok, p.X, p.Y, p.Z, p.TimestampUS)
	}
}

// positionLaPlusProche rend la position du slot la plus proche en temps de `tUS`.
func positionLaPlusProche(pos []grammar.BipedPosition, slot uint32, tUS uint64) (grammar.BipedPosition, bool) {
	var best grammar.BipedPosition
	var bestD uint64
	ok := false
	for _, p := range pos {
		if p.Slot != slot || !p.HasWorld {
			continue
		}
		d := equipTimeGap(p.TimestampUS, tUS)
		if !ok || d < bestD {
			best, bestD, ok = p, d, true
		}
	}
	return best, ok
}

// TestSoclesDepuisLesFaits — LE BANC DES TEMOINS DU POINT 11 : l assemblage de chaque film,
// rejoue DEPUIS SES FAITS (aucun octet de film decode), ecrit en JSON dans un dossier de sortie.
// Le meme binaire, compile sur la base puis sur le lot, rend les deux cotes de la comparaison :
// les memes faits, les memes catalogues, seul le code d assemblage change.
//
//	SOCLES_FAITS=<repo>/data/cache/film_facts/halo_infinite SOCLES_CARTES=<court8 TAB carte, par ligne>
//	SOCLES_SORTIE=<dossier> go test -tags research ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestSoclesDepuisLesFaits$' -v -count=1
func TestSoclesDepuisLesFaits(t *testing.T) {
	faits, cartes, sortie := os.Getenv("SOCLES_FAITS"), os.Getenv("SOCLES_CARTES"), os.Getenv("SOCLES_SORTIE")
	if faits == "" || cartes == "" || sortie == "" {
		t.Skip("SOCLES_FAITS, SOCLES_CARTES et SOCLES_SORTIE requis : banc saute")
	}
	table, err := os.ReadFile(cartes)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(strings.TrimSpace(string(table)), "\n") {
		c := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(c) < 2 || len(c[0]) != 8 {
			continue
		}
		t.Setenv("SOCLE_CARTE_BANC", c[1])
		entry := mapQuantEntryFromEnv(t, "SOCLE_CARTE_BANC", "SOCLE_BORNES")
		blob, err := os.ReadFile(filepath.Join(faits, c[0]+".filmfacts.bin"))
		if err != nil {
			t.Fatalf("%s : %v", c[0], err)
		}
		f, err := DecodeFilmFactsFile(blob, entry)
		if err != nil {
			t.Fatalf("%s : %v", c[0], err)
		}
		opt := f.Facts.options()
		opt.Labels = goldenCatalog(t)
		opt.MapQuant = &entry
		doc := BuildFromFacts(t.Context(), c[0], "halo_infinite", f, opt)
		out, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sortie, c[0]+".json"), out, 0o600); err != nil {
			t.Fatal(err)
		}
		gw := doc.Coverage.GroundWeapons
		t.Logf("%s (%s) : %d socles, couverture %+v", c[0], c[1], len(doc.WeaponPads), *gw)
	}
}
