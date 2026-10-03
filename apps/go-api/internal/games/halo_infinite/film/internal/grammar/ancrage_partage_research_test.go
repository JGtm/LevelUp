//go:build research

package grammar

// ancrage_partage_research_test.go — INSTRUMENT : CE QUE COUTE L ANCRAGE BIPEDE DE LA CUISSON, ET SI
// LES POSITIONS ANCRENT COMME LES HUIT PASSES (lot 2.4 de la representation intermediaire, mesure
// prealable aux decisions du lot).
//
// Trois questions, film par film, sous le contexte de la cuisson (entree de carte du catalogue) :
//  1. l ancrage seul ([walkDeltaBipedRecords] sous les parametres du contexte) : sa duree, le nombre
//     de records ancres et de paquets qui en portent ;
//  2. chacun des huit balayages ancres : sa duree complete, a comparer a celle de l ancrage, et la
//     marche de tous les corps ancres jusqu au premier composant non porte ;
//  3. l egalite des parametres de l ancrage des positions ([ScanBipedPositions]) avec ceux du
//     contexte (chunks, bande, decoupage, generations), puis l egalite des suites de records ancres.
//
//	FILM_CACHE_ROOT=<depot>/data/cache ANCRAGE_FILMS='084a804d=Fortitude Heavies;e5adf7b2=Fragmentation' \
//	  [ANCRAGE_COUTS=1] go test -tags research ./internal/games/halo_infinite/film/internal/grammar/ -run AncragePartage -v
//
// Sans ANCRAGE_COUTS, seule la troisieme question est jouee. Mesure du 2026-10-03 (lot 2.4) : sur
// quatre films, l ancrage coute 0,6 a 1,8 s, chaque balayage ancre 0 a 140 ms de plus (les
// changements d equipement 0,17 a 0,95 s, leur recuperation gatee), la marche de tous les corps
// ancres 31 a 159 ms ; sur les vingt films du corpus d equivalence, les parametres et les suites
// ancrees des positions sont ceux du contexte.

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestAncragePartage imprime, film par film, les trois mesures de l en-tete.
func TestAncragePartage(t *testing.T) {
	racine, liste := os.Getenv("FILM_CACHE_ROOT"), os.Getenv("ANCRAGE_FILMS")
	if racine == "" || liste == "" {
		t.Skip("FILM_CACHE_ROOT ou ANCRAGE_FILMS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		if os.Getenv("ANCRAGE_COUTS") != "" {
			mesurerLAncrage(t, id, NewFilmContextForMap(film, &entree, nil))
		}
		comparerLesParametres(t, id, NewFilmContextForMap(film, &entree, nil), &entree)
	}
}

// mesurerLAncrage mesure l ancrage seul, les huit balayages et la marche complete des corps.
func mesurerLAncrage(t *testing.T, id string, fc *FilmContext) {
	t.Helper()
	slots, chunks := fc.BipedSlots(), fc.ChunkNumbers()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("%s : decoupage : %v", id, err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		t.Fatalf("%s : archetype : %v", id, err)
	}
	fc.GenerationsVivantes()
	debut := time.Now()
	records, indices, paquets := 0, 0, 0
	dernier := [2]int{-1, -1}
	walkDeltaBipedRecords(fc, chunks, slots, lay, func(r deltaBipedRecord) {
		records++
		indices += len(r.Mask)
		if cle := [2]int{r.Chunk, r.Packet.Index}; cle != dernier {
			paquets, dernier = paquets+1, cle
		}
	})
	dAncrage := time.Since(debut)
	t.Logf("%s : ancrage %v — %d record(s) dans %d paquet(s), %d index de masque", id,
		dAncrage.Round(time.Millisecond), records, paquets, indices)

	pasDeNaissance := func(uint32, uint64) (SpawnState, bool) { return SpawnState{}, false }
	jamaisNe := func(uint32) (uint64, bool) { return 0, false }
	for _, b := range []struct {
		nom string
		f   func()
	}{
		{"heldWeaponChanges", func() { _, _, _ = ScanHeldWeaponChanges(fc, pasDeNaissance) }},
		{"inventoryDeltas", func() { _, _, _ = ScanInventoryDeltas(fc) }},
		{"abilityRanks", func() { _, _, _ = ScanAbilityRanks(fc) }},
		{"equipmentChanges", func() { _, _, _ = ScanEquipmentChanges(fc, jamaisNe) }},
		{"camoStates", func() { _, _, _ = ScanCamoStates(fc) }},
		{"grappleReads", func() { _, _, _ = ScanGrappleReads(fc) }},
		{"abilityImpulses", func() { _, _, _ = ScanAbilityImpulses(fc) }},
		{"abilityCharges", func() { _, _, _ = ScanAbilityCharges(fc) }},
		{"unitEquipment", func() { _, _ = ScanUnitEquipment(fc) }},
	} {
		debut = time.Now()
		b.f()
		d := time.Since(debut)
		t.Logf("%s :   %-18s %v (hors ancrage : %v)", id, b.nom, d.Round(time.Millisecond),
			(d - dAncrage).Round(time.Millisecond))
	}

	gram := grammaireRecord{lay: lay, arch: arch, prof: fc.ProfilDeBalayage(), obs: NouvelleObservation()}
	debut = time.Now()
	traverses, complets := 0, 0
	walkDeltaBipedRecords(fc, chunks, slots, lay, func(r deltaBipedRecord) {
		n := 0
		walkRecordComponents(r.Payload, r.I0, r.Total, r.Mask, gram, func(int) bool { n++; return true })
		traverses += n
		if n == len(r.Mask)-1 {
			complets++
		}
	})
	t.Logf("%s :   ancrage + marche complete des corps %v (hors ancrage : %v) — %d composant(s) traverse(s), "+
		"%d record(s) sur %d marches jusqu au bout du masque", id, time.Since(debut).Round(time.Millisecond),
		(time.Since(debut) - dAncrage).Round(time.Millisecond), traverses, complets, records)
}

// comparerLesParametres compare les parametres de l ancrage des positions a ceux du contexte, puis
// les suites de records ancres, paquet par paquet.
func comparerLesParametres(t *testing.T, id string, fc *FilmContext, entree *profile.MapQuantEntry) {
	t.Helper()
	film := fc.Film()
	chunksP := FilmChunkNumbers(film)
	if !slices.Equal(chunksP, fc.ChunkNumbers()) {
		t.Errorf("%s : chunks des positions %v, du contexte %v", id, chunksP, fc.ChunkNumbers())
	}
	bandeP := bipedSlotBand(NewFilmContextForMap(film, entree, nil), chunksP)
	bande := fc.BipedSlots()
	if !slices.Equal(bandeP.Slots(), bande.Slots()) {
		t.Errorf("%s : bande des positions (%d slots) differente de celle du contexte (%d slots)", id,
			bandeP.Count(), bande.Count())
	}
	layP, err := bipedI0Layout(film, ScanFilmOptions{Layout: fc.ImposedLayout()})
	if err != nil {
		t.Fatalf("%s : decoupage des positions : %v", id, err)
	}
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("%s : decoupage du contexte : %v", id, err)
	}
	if layP != lay {
		t.Errorf("%s : decoupage des positions %+v, du contexte %+v", id, layP, lay)
	}
	gensP := fc.GenerationsVivantes()
	egaux, differents := 0, 0
	for _, c := range chunksP {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			var a, b []deltaBipedRecord
			walkDeltaBipedPayload(pk.Payload(data), bandeP, layP, gensP.A(pk.TimestampUS), func(r deltaBipedRecord) {
				a = append(a, r)
			})
			walkDeltaBipedPayload(pk.Payload(data), bande, lay, fc.GenerationsVivantesA(pk.TimestampUS),
				func(r deltaBipedRecord) { b = append(b, r) })
			if memeAncrage(a, b) {
				egaux++
			} else {
				differents++
			}
		}
	}
	t.Logf("%s : parametres des positions egaux a ceux du contexte (chunks, bande de %d slots, decoupage "+
		"%+v) ; suites ancrees : %d paquet(s) egal(aux), %d different(s)", id, bande.Count(), lay, egaux,
		differents)
	if differents > 0 {
		t.Errorf("%s : %d paquet(s) ancres differemment", id, differents)
	}
}

// memeAncrage dit si deux suites de records ancres sont les memes : positions, slots, generations,
// masques.
func memeAncrage(a, b []deltaBipedRecord) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].I0 != b[i].I0 || a[i].Slot != b[i].Slot || a[i].Gen != b[i].Gen || !slices.Equal(a[i].Mask, b[i].Mask) {
			return false
		}
	}
	return true
}
