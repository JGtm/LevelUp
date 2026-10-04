package grammar

// canal_des_morts_test.go — LE CANAL DES MORTS DE LA MARCHE DES TRAMES, eprouve sur la bobine reelle a
// paquets delta (`../facts/killsource/testdata/minibobine_000d5950`, cf. `object_deaths_test.go`).

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// marchesAvecEtSansMorts marche la bobine deux fois, sans puis avec le canal des morts, chacune sur
// son contexte.
func marchesAvecEtSansMorts(t *testing.T) (sans, avec MarcheDesTrames, fcAvec *FilmContext) {
	t.Helper()
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	if sans, err = ScanMarcheDesTramesAvec(NewFilmContext(film), LecturesDeLaMarche{}); err != nil {
		t.Fatalf("marche sans les morts : %v", err)
	}
	fcAvec = NewFilmContext(film)
	if avec, err = ScanMarcheDesTramesAvec(fcAvec, LecturesDeLaMarche{Morts: true}); err != nil {
		t.Fatalf("marche avec les morts : %v", err)
	}
	return sans, avec, fcAvec
}

// TestLeCanalDesMortsNeChangeRienAuxAutresCanaux : la marche avec le canal des morts rend les memes
// etats de mouvement et le meme tir continu que sans lui. La recuperation des listes que la marche
// ne localise pas lit sans observation ; la bobine porte de ces listes, sans quoi le test ne
// garderait rien.
func TestLeCanalDesMortsNeChangeRienAuxAutresCanaux(t *testing.T) {
	sans, avec, fc := marchesAvecEtSansMorts(t)
	if n := fc.ComptesDesReplis().LocalisationsALargeurLibre; n == 0 {
		t.Fatal("aucune liste recuperee sur la bobine : le test ne garde rien")
	}
	if !reflect.DeepEqual(sans.MovementStates, avec.MovementStates) ||
		!reflect.DeepEqual(sans.MovementStateStats, avec.MovementStateStats) {
		t.Errorf("etats de mouvement differents avec le canal des morts : %d lectures contre %d",
			len(avec.MovementStates), len(sans.MovementStates))
	}
	if !reflect.DeepEqual(sans.ContinuousFire, avec.ContinuousFire) ||
		!reflect.DeepEqual(sans.ContinuousFireStats, avec.ContinuousFireStats) {
		t.Errorf("tir continu different avec le canal des morts : %d rafales contre %d",
			len(avec.ContinuousFire), len(sans.ContinuousFire))
	}
	if len(sans.ObjectDeaths) != 0 || len(sans.Occupancy) != 0 {
		t.Errorf("morts ou occupation lues sans le canal des morts : %d, %d", len(sans.ObjectDeaths),
			len(sans.Occupancy))
	}
}

// TestLeCanalDesMortsCompteSesDenominateurs : chaque trame delta marchee compte, les paquets a
// evenements localises comprennent les listes recuperees, et les images-cles liees sont comptees.
func TestLeCanalDesMortsCompteSesDenominateurs(t *testing.T) {
	_, avec, fc := marchesAvecEtSansMorts(t)
	st := avec.ObjectDeathStats
	recuperees := fc.ComptesDesReplis().LocalisationsALargeurLibre
	if st.Keyframes == 0 || st.Deltas == 0 || st.Packets == 0 {
		t.Fatalf("denominateurs muets : %d images-cles, %d trames, %d paquets marches", st.Keyframes, st.Deltas,
			st.Packets)
	}
	if st.LocatedPackets > st.EventPackets || st.Packets > st.Deltas {
		t.Errorf("denominateurs incoherents : %d localises sur %d a evenements, %d marches sur %d trames",
			st.LocatedPackets, st.EventPackets, st.Packets, st.Deltas)
	}
	if st.LocatedPackets < recuperees {
		t.Errorf("%d liste(s) recuperee(s) pour %d paquet(s) localise(s) : les recuperees n y sont pas",
			recuperees, st.LocatedPackets)
	}
	if st.Config.Obs != nil {
		t.Error("le cadre publie porte l observation de la marche")
	}
}

// TestLaRecuperationRendLeMondeIntact : sur chaque liste que la marche de la bobine ne localise pas,
// la recuperation lit des records et laisse le monde de la marche tel qu elle l a trouve — liaisons et
// positions comprises. LIMITE : les listes recuperees de la bobine ne lient ni ne delient aucun slot
// du monde (deux DEL de slots non lies, aucun NEW lie) ; la mutation qui retire la restauration y
// reste donc verte, et c est sa presence dans [MarcheDistribuee.recupererLaListe] qui la garde.
func TestLaRecuperationRendLeMondeIntact(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	m, err := NewFilmContext(film).nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	lues := 0
	m.parcourir(func(tr *trameLue) bool {
		if tr.debut >= 0 {
			return true
		}
		avant := m.monde.Snapshot()
		md := &MarcheDistribuee{Paquet: tr.paquet, marche: m}
		if recs, lus, _ := md.recupererLaListe(); lus && len(recs) > 0 {
			lues++
		}
		if !reflect.DeepEqual(m.monde.Snapshot(), avant) {
			t.Fatalf("paquet %d du chunk %d : la recuperation a modifie le monde de la marche", tr.paquet.Index,
				tr.paquet.Chunk)
		}
		return true
	})
	if lues == 0 {
		t.Fatal("aucune liste recuperee avec des records sur la bobine : le test ne garde rien")
	}
	t.Logf("%d liste(s) recuperee(s), le monde intact apres chacune", lues)
}
