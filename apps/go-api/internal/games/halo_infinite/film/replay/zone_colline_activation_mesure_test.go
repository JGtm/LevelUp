package replay

// zone_colline_activation_mesure_test.go — QUAND LA PREMIERE COLLINE S ACTIVE-T-ELLE, sur films reels.
//
// Par film, sur le bloc nomme de l objet de mode KOTH (designateur, proprietaire, pousseur, jauge),
// toutes les dates en millisecondes sur l horloge MOTEUR du film (celle des positions) :
//
//	debut      le premier paquet delta du film portant une lecture ti=13 (n importe quel slot) ;
//	imagesCles les instants des images-cles qui portent des records ti=13, et la PREMIERE ou le bloc
//	           de l objet de mode est present (avec sa designation) ;
//	delta      la premiere lecture delta d un slot du bloc, tous tags, chainee ou non ;
//	contact    le premier contact (`hillFirstContact`), borne actuelle de la 1re periode ;
//	suivantes  pour chaque bascule du designateur, le delai jusqu a la premiere emission du bloc
//	           qui la suit (proprietaire, pousseur ou jauge).
//
// SOUS GARDE D ENVIRONNEMENT (`HILL_FILMS`, repertoires separes par `;`), lecture seule :
//
//	$env:CGO_ENABLED=0
//	$env:HILL_FILMS="C:/.../film_chunks/0d9a9af9;..."
//	go test -count=1 -run TestCollineActivationMesure -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// TestCollineActivationMesure publie, film par film, les bornes de l activation de la 1re colline.
func TestCollineActivationMesure(t *testing.T) {
	env := os.Getenv("HILL_FILMS")
	if env == "" {
		t.Skip("mesure non demandee : HILL_FILMS vide")
	}
	for _, dir := range strings.Split(env, ";") {
		if dir = strings.TrimSpace(dir); dir != "" {
			hillActivationFilm(t, dir)
		}
	}
}

// hillActivationFilm mesure UN film.
func hillActivationFilm(t *testing.T, dir string) {
	t.Helper()
	id := filepath.Base(dir)
	sc, err := grammar.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Logf("%s : balayage en echec : %v", id, err)
		return
	}
	c := hillMesureCtx(sc)
	ser := zoneSeriesOf(sc.Reads, c)
	ser.noms = zoneNomsDesSlots(sc.KeyReads)
	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Logf("%s : aucun designateur", id)
		return
	}
	bloc := hillModeObjectSlots(ser.noms, d.slot)
	bloc = append(bloc, d.slot)
	debut := c.origin / 1000
	t.Logf("%s : debut (1re lecture ti=13) %d ms ; designateur %d, bloc %v", id, debut, d.slot, bloc)
	t.Logf("%s : images-cles ti=13 : %s", id, hillKeyInstants(sc.KeyReads, debut))
	t.Logf("%s : bloc aux images-cles : %s", id, hillBlocAuxImagesCles(sc.KeyReads, bloc, d.slot, debut))
	t.Logf("%s : 1re lecture delta du bloc +%d ms ; premier contact +%.0f ms", id,
		hillFirstDelta(sc.Reads, bloc)/1000-debut, ms(d.first))
	t.Logf("%s : premier contact a %d ms moteur ; premieres lectures du bloc :%s", id,
		debut+uint64(ms(d.first)), hillFirstBlocReads(sc.Reads, bloc, 6))
	t.Logf("%s : bascules du designateur -> 1re emission du bloc : %s", id, hillDelaisApresBascule(ser, bloc, d))
	if os.Getenv("HILL_APRES") != "" {
		hillLecturesApresBascule(t, id, sc, c, d)
	}
}

// hillKeyInstants rend les instants (relatifs au debut) des images-cles portant des records ti=13.
func hillKeyInstants(reads []grammar.ManagedPropertyRead, debut uint64) string {
	var ts []int64
	for _, r := range reads {
		v := int64(r.TimestampUS/1000) - int64(debut) //nolint:gosec // horodatage borne
		if !slices.Contains(ts, v) {
			ts = append(ts, v)
		}
	}
	slices.Sort(ts)
	return fmt.Sprint(ts)
}

// hillBlocAuxImagesCles rend, par image-cle, les slots du bloc presents et la designation lue.
func hillBlocAuxImagesCles(reads []grammar.ManagedPropertyRead, bloc []uint32, desig uint32, debut uint64) string {
	parT := map[int64][]string{}
	var ts []int64
	for _, r := range reads {
		if !slices.Contains(bloc, r.Slot) {
			continue
		}
		v := int64(r.TimestampUS/1000) - int64(debut) //nolint:gosec // horodatage borne
		if _, ok := parT[v]; !ok {
			ts = append(ts, v)
		}
		lab := fmt.Sprintf("%d:t%d", r.Slot, r.Tag)
		if r.Slot == desig {
			lab += fmt.Sprintf("=%#x", r.Value)
		}
		parT[v] = append(parT[v], lab)
	}
	slices.Sort(ts)
	var sb strings.Builder
	for _, v := range ts {
		fmt.Fprintf(&sb, " [+%d %v]", v, parT[v])
	}
	return sb.String()
}

// hillFirstDelta rend l horodatage (us) de la premiere lecture delta d un slot du bloc.
func hillFirstDelta(reads []grammar.ManagedPropertyRead, bloc []uint32) uint64 {
	var first uint64
	for _, r := range reads {
		if slices.Contains(bloc, r.Slot) && (first == 0 || r.TimestampUS < first) {
			first = r.TimestampUS
		}
	}
	return first
}

// hillDelaisApresBascule rend, pour chaque bascule du designateur, le delai (ms) jusqu a la
// premiere emission du bloc qui la suit.
func hillDelaisApresBascule(ser zoneSeries, bloc []uint32, d hillDesignator) string {
	var sb strings.Builder
	for _, b := range d.changes {
		best := -1
		for _, slot := range bloc {
			if slot == d.slot {
				continue
			}
			for _, ss := range [][]zoneSample{ser.owner[slot], ser.gauge[slot]} {
				for _, s := range ss {
					if s.t > b && (best < 0 || s.t < best) {
						best = s.t
					}
				}
			}
		}
		fmt.Fprintf(&sb, " %.2fs->+%.2fs", ms(b)/1000, ms(best-b)/1000)
	}
	return sb.String()
}

// hillFirstBlocReads publie les premieres lectures delta du bloc, en ms MOTEUR absolues.
func hillFirstBlocReads(reads []grammar.ManagedPropertyRead, bloc []uint32, n int) string {
	var sel []grammar.ManagedPropertyRead
	for _, r := range reads {
		if slices.Contains(bloc, r.Slot) {
			sel = append(sel, r)
		}
	}
	slices.SortStableFunc(sel, func(a, b grammar.ManagedPropertyRead) int {
		return int(int64(a.TimestampUS) - int64(b.TimestampUS)) //nolint:gosec // ecart borne
	})
	var sb strings.Builder
	for i, r := range sel {
		if i >= n {
			break
		}
		fmt.Fprintf(&sb, " [%d ms %d t%d v=%#x ch=%v f=%d]", r.TimestampUS/1000, r.Slot, r.Tag, r.Value, r.Chained, r.FilmIndex)
	}
	return sb.String()
}

// hillLecturesApresBascule publie, pour chaque bascule du designateur, TOUTES les lectures ti=13
// (tout slot, tout tag) dans [bascule + 3 s ; bascule + 5,6 s] : on y cherche un signal emis a
// chaque deplacement, a l instant ou la nouvelle colline devient prenable.
func hillLecturesApresBascule(t *testing.T, id string, sc grammar.ManagedPropertyScan, c zoneCtx, d hillDesignator) {
	t.Helper()
	for _, b := range d.changes {
		bUS := c.origin + uint64(b)*c.step //nolint:gosec // frame positive
		var sb strings.Builder
		for _, r := range sc.Reads {
			if r.TimestampUS < bUS+3_000_000 || r.TimestampUS > bUS+5_600_000 || !r.Chained {
				continue
			}
			fmt.Fprintf(&sb, " [+%.2f s%+d t%d f%d v%#x]", float64(r.TimestampUS-bUS)/1e6,
				int(r.Slot)-int(d.slot), r.Tag, r.FilmIndex, r.Value) //nolint:gosec // slots bornes
		}
		t.Logf("%s : APRES %.2f s :%s", id, ms(b)/1000, sb.String())
	}
}
