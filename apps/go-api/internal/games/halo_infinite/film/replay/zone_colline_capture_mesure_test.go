package replay

// zone_colline_capture_mesure_test.go — LA MESURE DE LA CAPTURE D UNE COLLINE, sur films reels.
//
// LA QUESTION : la jauge (tag 3) de l objet de mode KOTH est-elle la CAPTURE de la colline — une
// montee courte, menee par le camp seul dans la colline, dont l aboutissement fait basculer le
// proprietaire ? Le mesure par film, sur le bloc nomme de l objet de mode (designateur = cle) :
//
//	RAMPES       chaque montee de la jauge (definition de production `findZoneRamps`) : duree,
//	             depart, sommet, nombre d emissions ;
//	DESCENTES    les pas descendants qui ne sont PAS un retour au zero (une jauge qui se vide pas
//	             a pas quand la colline est contestee ou abandonnee) ;
//	BASCULES     chaque changement du proprietaire vers un camp : la rampe qui la precede, l ecart
//	             entre son sommet et la bascule, et le pousseur pendant la rampe ;
//	ORACLE       le delai entre le premier contact (`hillFirstContact`) et la premiere prise.
//
// SOUS GARDE D ENVIRONNEMENT (`HILL_FILMS` = repertoires de film separes par `;`), un processus,
// avant-plan, lecture seule :
//
//	$env:CGO_ENABLED=0
//	$env:HILL_FILMS="C:/.../data/cache/film_chunks/0d9a9af9;C:/.../data/cache/film_chunks/26602661"
//	go test -count=1 -run TestCollineCaptureMesure -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// hillMesureStepUS est le pas de la grille de mesure : 10 ms, plus fin que toute cadence d emission.
const hillMesureStepUS = 10_000

// TestCollineCaptureMesure publie, film par film, la forme de la jauge de l objet de mode KOTH.
func TestCollineCaptureMesure(t *testing.T) {
	env := os.Getenv("HILL_FILMS")
	if env == "" {
		t.Skip("mesure non demandee : HILL_FILMS vide")
	}
	for _, dir := range strings.Split(env, ";") {
		if dir = strings.TrimSpace(dir); dir != "" {
			hillMesureFilm(t, dir)
		}
	}
}

// hillMesureFilm mesure UN film.
func hillMesureFilm(t *testing.T, dir string) {
	t.Helper()
	id := filepath.Base(dir)
	sc, err := grammar.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Logf("%s : balayage ti=13 en echec : %v", id, err)
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
	b, named := zoneBlocDuSlot(d.slot, ser.noms, func(b zoneBlocNomme) uint32 { return b.cle })
	if !named {
		t.Logf("%s : designateur %d sans bloc nomme (voisinage=%v)", id, d.slot, d.parVoisinage)
		return
	}
	jauge, okJ := ser.noms.parNom[b.jauge]
	prop, okP := ser.noms.parNom[b.proprietaire]
	pous, okU := ser.noms.parNom[b.pousseur]
	t.Logf("%s : designateur %d (%d bascules), jauge %d(%v) %d emissions, proprietaire %d(%v), pousseur %d(%v)",
		id, d.slot, len(d.changes), jauge, okJ, len(ser.gauge[jauge]), prop, okP, pous, okU)
	t.Logf("%s : premier contact a %.2f s ; bascules du designateur : %s", id, ms(d.first)/1000,
		hillMesureInstants(d.changes))
	hillMesureRampes(t, id, ser.gauge[jauge], ser.owner[pous])
	slots := hillMesureSlots{jauge: jauge, prop: prop, pous: pous}
	hillMesureBascules(t, id, ser, slots, d)
	hillMesureDump(t, id, ser, slots)
}

// hillMesureCtx rend une grille de 10 ms couvrant toutes les lectures du film.
func hillMesureCtx(sc grammar.ManagedPropertyScan) zoneCtx {
	var lo, hi uint64
	for i, r := range sc.Reads {
		if i == 0 || r.TimestampUS < lo {
			lo = r.TimestampUS
		}
		hi = max(hi, r.TimestampUS)
	}
	return zoneCtx{origin: lo, step: hillMesureStepUS, frames: int((hi-lo)/hillMesureStepUS) + 1,
		intervalMS: hillMesureStepUS / 1000}
}

// ms convertit une frame de la grille de mesure en millisecondes.
func ms(f int) float64 { return float64(f * hillMesureStepUS / 1000) }

// hillMesureInstants rend une liste d instants en secondes.
func hillMesureInstants(fs []int) string {
	var sb strings.Builder
	for _, f := range fs {
		sb.WriteString(" ")
		sb.WriteString(fmt.Sprintf("%.2f", ms(f)/1000))
	}
	return sb.String()
}

// hillMesureSlots nomme les trois canaux du bloc de l objet de mode.
type hillMesureSlots struct{ jauge, prop, pous uint32 }

// hillMesureRampes publie chaque rampe et les descentes pas a pas de la jauge.
func hillMesureRampes(t *testing.T, id string, ss, pous []zoneSample) {
	t.Helper()
	ramps := findZoneRamps(0, ss)
	desc, resets, plats := 0, 0, 0
	for i := 1; i < len(ss); i++ {
		switch {
		case ss[i].v < ss[i-1].v && ss[i].v <= zoneGaugeQuantZero:
			resets++
		case ss[i].v < ss[i-1].v:
			desc++
		case ss[i].v == ss[i-1].v:
			plats++
		}
	}
	t.Logf("%s : %d rampes ; pas descendants hors zero %d, retours a zero %d, pas nuls %d",
		id, len(ramps), desc, resets, plats)
	for _, r := range ramps {
		n := 0
		for _, s := range ss {
			if s.t >= r.t0 && s.t <= r.tPeak {
				n++
			}
		}
		t.Logf("%s :   rampe %.2f s -> %.2f s (%.0f ms, %d emissions) %.3f -> %.3f, pousseur %s",
			id, ms(r.t0)/1000, ms(r.tPeak)/1000, ms(r.tPeak-r.t0), n,
			gaugeProgressOf(r.start), gaugeProgressOf(r.top), hillMesureValeurA(pous, r.tPeak))
	}
}

// hillMesureValeurA rend la derniere valeur d un canal a l instant f (ou avant).
func hillMesureValeurA(ss []zoneSample, f int) string {
	v, ok := uint64(0), false
	for _, s := range ss {
		if s.t > f {
			break
		}
		v, ok = s.v, true
	}
	if !ok {
		return "?"
	}
	if v == zoneNeutralOwner {
		return "neutre"
	}
	return fmt.Sprintf("%d", v)
}

// hillMesureBascules publie chaque prise de la colline par un camp et la rampe qui la precede.
func hillMesureBascules(t *testing.T, id string, ser zoneSeries, s hillMesureSlots, d hillDesignator) {
	t.Helper()
	ramps := findZoneRamps(0, ser.gauge[s.jauge])
	prev, premiere := uint64(zoneNeutralOwner), true
	for _, o := range ser.owner[s.prop] {
		if o.v == prev {
			continue
		}
		prev = o.v
		if o.v == zoneNeutralOwner {
			t.Logf("%s :   proprietaire -> neutre a %.2f s", id, ms(o.t)/1000)
			continue
		}
		ecart, duree := -1.0, -1.0
		for _, r := range ramps {
			if r.tPeak <= o.t+1 {
				ecart, duree = ms(o.t-r.tPeak), ms(r.tPeak-r.t0)
			}
		}
		t.Logf("%s :   prise par %d a %.2f s ; rampe precedente : duree %.0f ms, sommet -> prise %.0f ms",
			id, o.v, ms(o.t)/1000, duree, ecart)
		if premiere {
			t.Logf("%s :   ORACLE premier contact -> premiere prise : %.2f s", id, ms(o.t-d.first)/1000)
			premiere = false
		}
	}
}

// hillMesureDump publie, quand `HILL_DUMP` = "debut-fin" (secondes), les emissions brutes des trois
// canaux du bloc dans la fenetre : la forme de la jauge a l emission pres.
func hillMesureDump(t *testing.T, id string, ser zoneSeries, s hillMesureSlots) {
	t.Helper()
	var lo, hi float64
	if _, err := fmt.Sscanf(os.Getenv("HILL_DUMP"), "%f-%f", &lo, &hi); err != nil {
		return
	}
	for _, c := range []struct {
		nom string
		ss  []zoneSample
	}{{"jauge", ser.gauge[s.jauge]}, {"proprietaire", ser.owner[s.prop]}, {"pousseur", ser.owner[s.pous]}} {
		var sb strings.Builder
		for _, x := range c.ss {
			if sec := ms(x.t) / 1000; sec >= lo && sec <= hi {
				v := fmt.Sprintf("%d", x.v)
				if c.nom == "jauge" {
					v = fmt.Sprintf("%.3f", gaugeProgressOf(x.v))
				}
				fmt.Fprintf(&sb, " %.2f=%s", sec, v)
			}
		}
		t.Logf("%s : DUMP %s :%s", id, c.nom, sb.String())
	}
}
