package grammar

// keyframe_world_attestation_test.go — L'ATTESTATION PAR ENTREES LIBRES (keyframe_world_attestation.go) :
// sur payloads synthetiques, puis sur le film reel dont les images-cles ont revele le defaut.
//
// Le test de film est SAUTE sans son repertoire de chunks (le depot ne versionne aucun film, la CI
// n'en a aucun). Usage :
//
//	ATTESTATION_FILM=<cache>/film_chunks/4898d586 \
//	  go test -count=1 -run TestAttestationSurFilm -v ./internal/games/halo_infinite/film/internal/grammar/

import (
	"os"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

// attestationFilmEnv nomme le repertoire de chunks du film a mesurer.
const attestationFilmEnv = "ATTESTATION_FILM"

// kfEcrireEntreesLibres ecrit n slots liberes : trois mots a 0xffffffff puis les 12 bits nuls de
// l'ecrivain (108 bits).
func kfEcrireEntreesLibres(w *bitWriter, n int) {
	for range n {
		w.bits(kfSent, 32)
		w.bits(kfSent, 32)
		w.bits(kfSent, 32)
		w.bits(0, keyframeEntreeLibreBits-96)
	}
}

// kfPayloadFausseAncre : un record de slot 10, dans le corps duquel une FAUSSE ancre de slot 12
// (generation 1, slot plus bas que le vrai) precede le vrai record de slot 16. `libres` entrees
// libres precedent le vrai record.
func kfPayloadFausseAncre(libres int) []byte {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, 5, 0)
	w.bits(1<<30|12, 32) // la fausse ancre : generation 1, slot 12, archetype 0
	w.bits(0, 32)
	w.bits(0, 300)
	kfEcrireEntreesLibres(w, libres)
	kfEcrireRecord(w, 1, 16, 5, 300)
	return w.buf
}

// TestElectionAttesteeBatLaFausseAncreDeSlotBas : le vrai record de slot 16 a devant lui les cinq
// entrees libres (11..15) que son slot exige ; la fausse ancre de slot 12 n'en a aucune. Le vrai
// record est elu, et l'election se compte attestee.
func TestElectionAttesteeBatLaFausseAncreDeSlotBas(t *testing.T) {
	recs, st := WalkKeyframeWorldStats(kfPayloadFausseAncre(5))
	if len(recs) != 2 || recs[0].Slot != 10 || recs[1].Slot != 16 {
		t.Fatalf("records %+v, attendu les slots [10 16]", recs)
	}
	if st.Elections != 1 || st.ElectionsAttestees != 1 || st.ElectionsNonAttestees != 0 {
		t.Fatalf("compteurs %+v : attendu 1 election, attestee", st)
	}
}

// TestElectionSansAttestationRestePourLaFausseAncre : un nombre d'entrees libres qui n'est pas
// celui de l'ecart de slot (quatre au lieu de cinq) n'atteste rien : le repli d'avant joue, et il se
// compte NON attesté.
func TestElectionSansAttestationRestePourLaFausseAncre(t *testing.T) {
	_, st := WalkKeyframeWorldStats(kfPayloadFausseAncre(4))
	if st.ElectionsAttestees != 0 || st.ElectionsNonAttestees < 1 {
		t.Fatalf("compteurs %+v : attendu des elections non attestees seulement", st)
	}
}

// TestEntreesLibresAvantEstExact : le compte s'arrete a la premiere entree qui n'est pas libre et
// respecte le plafond.
func TestEntreesLibresAvantEstExact(t *testing.T) {
	w := &bitWriter{}
	w.bits(0, 200)
	kfEcrireEntreesLibres(w, 3)
	q := 200 + 3*keyframeEntreeLibreBits // fin des trois entrees, avant le bourrage a l octet
	if n := entreesLibresAvant(w.buf, q, 10); n != 3 {
		t.Fatalf("compte %d, attendu 3", n)
	}
	if n := entreesLibresAvant(w.buf, q, 2); n != 2 {
		t.Fatalf("compte plafonne %d, attendu 2", n)
	}
}

// TestAttestationSurFilm : les images-cles 10 et 11 du film 4898d586 portent chacune les huit
// vehicules (ti=40, slots 770 771 776 777 778 780 781 782) que la marche perdait en elisant une
// fausse ancre ti=20 de slot 542 derriere le bipede 533.
func TestAttestationSurFilm(t *testing.T) {
	dir := os.Getenv(attestationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : test de film saute", attestationFilmEnv)
	}
	film, ok, err := filmcache.LoadFilmDir(dir)
	if err != nil || !ok {
		t.Fatalf("film %s : ok=%v err=%v", dir, ok, err)
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup("Behemoth")
	if err != nil {
		t.Fatalf("carte : %v", err)
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	fc.PoserLaCarteEtLeDecoupage()
	const vehicleTI = 40
	want := []int{770, 771, 776, 777, 778, 780, 781, 782}
	for _, ts := range []uint64{3722165256, 3742168443} {
		got := vehiculesDeLImageCle(t, fc, ts, vehicleTI)
		if len(got) != len(want) {
			t.Fatalf("image-cle %d : vehicules %v, attendu %v", ts, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("image-cle %d : vehicules %v, attendu %v", ts, got, want)
			}
		}
	}
}

// vehiculesDeLImageCle rend les slots de l'archetype `ti` de l'image-cle d'horodatage `ts`.
func vehiculesDeLImageCle(t *testing.T, fc *FilmContext, ts uint64, ti int) []int {
	t.Helper()
	for _, num := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe || pk.TimestampUS != ts {
				continue
			}
			var slots []int
			for _, r := range fc.MarcheDImageCle().Records(pk.Payload(data)) {
				if r.TI == ti {
					slots = append(slots, r.Slot)
				}
			}
			return slots
		}
	}
	t.Fatalf("image-cle %d introuvable", ts)
	return nil
}
