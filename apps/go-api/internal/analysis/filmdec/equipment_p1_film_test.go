package filmdec

// equipment_p1_film_test.go — VALIDATION SUR PIÈCES de la récupération gatée (P1.1) : sur le
// cas index du rapport R2 (Dynasty `1b2d9e08`, slot 535), le chemin de PRODUCTION retrouve
// l'émission manquée, l'étiquette Recovered, et le spent qui suit redevient fiable.
//
// Gardé par P1_FILM (patron TRANSLOC_FILM — les films ne sont pas versionnés), skip par
// défaut. UN décodage filmdec par process. Les validations de l'exemption du filtre de
// vitesse (P1.4) vivent dans transloc_exemption_film_test.go.
//
//	CGO_ENABLED=0 P1_FILM=<depot>/data/cache/film_chunks/1b2d9e08 \
//	  go test ./internal/analysis/filmdec/ -run '^TestP1Recuperation' -v -timeout 30m

import (
	"os"
	"strings"
	"testing"
)

const p1FilmEnv = "P1_FILM"

// p1BirthWitness construit le témoin de naissance depuis un balayage QuantaOnly (les
// horodatages ne dépendent d'aucune borne de carte).
func p1BirthWitness(t *testing.T, dir string) func(uint32) (uint64, bool) {
	t.Helper()
	scan := DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	first := map[uint32]uint64{}
	for _, p := range pos {
		if at, ok := first[p.Slot]; !ok || p.TimestampUS < at {
			first[p.Slot] = p.TimestampUS
		}
	}
	return func(slot uint32) (uint64, bool) {
		at, ok := first[slot]
		return at, ok
	}
}

// TestP1RecuperationDynasty rejoue le cas index du rapport R2 sur le chemin de PRODUCTION :
// l'émission manquée du slot 535 (compteur 6, rang 11 — le translocateur) doit sortir de
// ScanFilmEquipmentChanges étiquetée Recovered, entre le taken (c5 r4) et le spent (c7), et
// le spent doit porter Previous=11 au lieu du faux 4.
func TestP1RecuperationDynasty(t *testing.T) {
	dir := os.Getenv(p1FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : validation sur pièces sautée", p1FilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	changes, st, err := ScanFilmEquipmentChanges(dir, p1BirthWitness(t, dir))
	if err != nil {
		t.Fatalf("balayage : %v", err)
	}
	t.Logf("STATS : vies=%d emissions=%d recuperees=%d sautsResiduels=%d manquantesResiduelles=%d"+
		" tetesHorsNorme=%d repetitions=%d",
		st.Lives, st.Walk.Read, st.Recovered, st.CounterJumps, st.MissedEstimate,
		st.LivesFirstOffSpec, st.Repeats)
	for _, c := range changes {
		if c.Recovered {
			t.Logf("RECUPEREE slot %d c%d rang %d kind=%s gap=%d @%dus (chunk %d pkt %d)",
				c.Slot, c.Counter, c.Rank, c.Kind, c.Gap, c.TimestampUS, c.Chunk, c.PacketIndex)
		}
	}
	if !strings.Contains(dir, "1b2d9e08") {
		t.Log("film différent du cas index : constats chiffrés non contrôlés")
		return
	}
	// Le cas index (R2 §2) : c5 r4 @5 686 333 831 -> RECUPEREE c6 r11 @5 700 481 685 ->
	// c7 spent @5 733 614 360, chaîne close (gap 0 partout).
	var got []EquipmentChange
	for _, c := range changes {
		if c.Slot == 535 && c.TimestampUS >= 5_686_000_000 && c.TimestampUS <= 5_734_000_000 {
			got = append(got, c)
		}
	}
	if len(got) != 3 {
		t.Fatalf("%d émission(s) du slot 535 dans la fenêtre du cas index, attendu 3 : %+v", len(got), got)
	}
	rec := got[1]
	if !rec.Recovered || rec.Counter != 6 || rec.Rank != 11 || rec.TimestampUS != 5_700_481_685 {
		t.Errorf("émission récupérée inattendue : %+v (attendu Recovered c6 rang 11 @5700481685us)", rec)
	}
	if rec.Kind != EquipmentTaken || rec.Previous != 4 {
		t.Errorf("la récupérée doit être un ramassage venant du grappin (taken, from=4) : %+v", rec)
	}
	spent := got[2]
	if spent.Kind != EquipmentSpent || spent.Previous != 11 || spent.Gap != 0 {
		t.Errorf("le spent doit désormais porter Previous=11 et une chaîne close : %+v", spent)
	}
}
