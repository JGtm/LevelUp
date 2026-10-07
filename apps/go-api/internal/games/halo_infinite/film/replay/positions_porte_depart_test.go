package replay

// positions_porte_depart_test.go — R-B3 : aucune vie d'un corps apres le depart PROUVE de l'occupant
// qui vivait a sa creation (positions_porte_depart.go).
//
//	R-B3-ECARTE      gabarit de `859da825` : corps cree pendant la presence de l'occupant, aucune
//	                 position avant son depart prouve, des positions apres : toutes ecartees, comptees ;
//	R-B3-VIE-AVANT   une position precede le depart : la vie a commence avec l'occupant, rien n'est ecarte ;
//	R-B3-NON-PROUVE  l'absence de l'occupant n'est prouvee par aucune image-cle : rien n'est ecarte ;
//	R-B3-DEUX        deux entites de l'index vivent a la creation : la regle se tait ;
//	R-B3-NON-BALAYE  film sans balayage des entites : la regle se tait ;
//	R-B3-CORPS-SUIVANT  le slot porte un corps suivant, d'un occupant present : seul le premier corps
//	                 est ecarte ;
//	R-B3-SANS-ENTITE aucune entite ne vit a la creation du corps : la regle se tait ;
//	R-B3-INSTABLE    l entite qui vit a la creation est instable : la regle se tait ;
//	R-B3-JOURNAL     ce qui est ecarte se dit en AVERTISSEMENT et au compteur expvar ;
//	R-B3-ASSEMBLAGE  par `BuildFromPositions` : aucune piste publiee pour le corps ecarte.

import (
	"bytes"
	"context"
	"expvar"
	"log/slog"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// porteDepartScan : des images-cles porteuses toutes les 20 s de 0 a 200 s ; l'entite d'index 2
// (slot 1329) est lue de 0 a 20 s, son absence est prouvee a 40 s ; l'entite d'index 5 (slot 1335)
// est lue tout le film.
func porteDepartScan() grammar.PlayerEntityScan {
	s := grammar.PlayerEntityScan{Scanned: true}
	for t := uint64(0); t <= 200_000_000; t += 20_000_000 {
		s.KeyframesUS = append(s.KeyframesUS, t)
	}
	fin := len(s.KeyframesUS) - 1
	s.Entities = []grammar.PlayerEntity{
		{Slot: 1329, Index: 2, Team: 0, FirstKF: 0, LastKF: 1, Seen: 2},
		{Slot: 1335, Index: 5, Team: 0, FirstKF: 0, LastKF: fin, Seen: fin + 1},
	}
	return s
}

// porteDepartCreation : un record de creation lu, a `ms` millisecondes.
func porteDepartCreation(slot uint32, ms int, gen, index uint32) grammar.BipedCreation {
	return grammar.BipedCreation{Slot: slot, Generation: gen, TimestampUS: uint64(ms) * 1000,
		HasIndex: true, ParticipantIndex: index}
}

// porteDepartPositions : `n` positions du slot, une toutes les 100 ms a partir de `debutMS`.
func porteDepartPositions(slot uint32, debutMS, n int) []grammar.BipedPosition {
	out := make([]grammar.BipedPosition, 0, n)
	for i := range n {
		out = append(out, pos(slot, debutMS+100*i, float32(i), 10, 1))
	}
	return out
}

func TestPorteDepartEcarteLesPositionsApresLeDepartProuve(t *testing.T) {
	var cov couverturePorte
	in := porteDepartPositions(548, 60_000, 4)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)},
		porteDepartScan(), &cov)
	if len(out) != 0 || cov.ApresDepart != 4 || cov.CorpsApresDepart != 1 {
		t.Fatalf("retenues %d, apresDepart %d, corpsApresDepart %d ; attendu 0, 4, 1", len(out),
			cov.ApresDepart, cov.CorpsApresDepart)
	}
}

func TestPorteDepartGardeUneVieCommenceeAvantLeDepart(t *testing.T) {
	var cov couverturePorte
	in := append(porteDepartPositions(548, 16_000, 2), porteDepartPositions(548, 60_000, 4)...)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)},
		porteDepartScan(), &cov)
	if len(out) != len(in) || cov.ApresDepart != 0 || cov.CorpsApresDepart != 0 {
		t.Fatalf("retenues %d sur %d, apresDepart %d : la vie a commence avec l'occupant, rien ne s'ecarte",
			len(out), len(in), cov.ApresDepart)
	}
}

func TestPorteDepartSeTaitSansAbsenceProuvee(t *testing.T) {
	scan := porteDepartScan()
	for r := 2; r < len(scan.KeyframesUS); r++ {
		scan.Doutes = append(scan.Doutes, grammar.DouteDAbsence{Rang: r, Slot: 1329})
	}
	var cov couverturePorte
	in := porteDepartPositions(548, 60_000, 4)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)}, scan, &cov)
	if len(out) != len(in) || cov.CorpsApresDepart != 0 {
		t.Fatalf("retenues %d sur %d : aucune image-cle ne prouve le depart, la regle se tait", len(out), len(in))
	}
}

func TestPorteDepartSeTaitQuandDeuxEntitesViventALaCreation(t *testing.T) {
	scan := porteDepartScan()
	scan.Entities = append(scan.Entities, grammar.PlayerEntity{Slot: 1400, Index: 2, Team: 1, FirstKF: 0,
		LastKF: 1, Seen: 2})
	var cov couverturePorte
	in := porteDepartPositions(548, 60_000, 4)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)}, scan, &cov)
	if len(out) != len(in) || cov.CorpsApresDepart != 0 {
		t.Fatalf("retenues %d sur %d : deux occupants possibles a la creation, la regle se tait",
			len(out), len(in))
	}
}

func TestPorteDepartSeTaitSansBalayageDesEntites(t *testing.T) {
	var cov couverturePorte
	in := porteDepartPositions(548, 60_000, 4)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)},
		grammar.PlayerEntityScan{}, &cov)
	if len(out) != len(in) || cov.CorpsApresDepart != 0 {
		t.Fatalf("retenues %d sur %d : sans balayage, rien n'est prouve", len(out), len(in))
	}
}

func TestPorteDepartNEcarteQueLeCorpsDuPartant(t *testing.T) {
	var cov couverturePorte
	in := append(porteDepartPositions(548, 60_000, 4), porteDepartPositions(548, 75_000, 3)...)
	creations := []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2),
		porteDepartCreation(548, 70_000, 2, 5)}
	out := ecarterApresLeDepart(in, creations, porteDepartScan(), &cov)
	if len(out) != 3 || cov.ApresDepart != 4 || cov.CorpsApresDepart != 1 {
		t.Fatalf("retenues %d, apresDepart %d, corps %d ; attendu 3, 4, 1 (le corps suivant est celui "+
			"d'un occupant present)", len(out), cov.ApresDepart, cov.CorpsApresDepart)
	}
	for _, p := range out {
		if p.TimestampUS < 70_000_000 {
			t.Errorf("position retenue a %d us : elle appartient au corps du partant", p.TimestampUS)
		}
	}
}

func TestPorteDepartParLAssemblage(t *testing.T) {
	in := porteFoule(300, 0)
	in = append(in, porteDepartPositions(548, 60_000, 4)...)
	opt := Options{FrameIntervalMS: 100, PlayerEntities: porteDepartScan(),
		BipedCreations: []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)}}
	doc := BuildFromPositions(context.Background(), "m", "halo_infinite", in, nil, opt)
	if trs := porteTraces(doc, 548); len(trs) != 0 {
		t.Fatalf("pistes du slot 548 = %d, attendu 0 : le corps du partant n'a aucune vie publiee", len(trs))
	}
	if trs := porteTraces(doc, 900); len(trs) != 1 {
		t.Errorf("pistes du slot 900 = %d, attendu 1 : la regle ne touche que le corps du partant", len(trs))
	}
}

// R-B3-SANS-ENTITE (garde de la fenetre) : le corps 550 est cree a 50 s, apres le depart prouve de
// l'occupant d'index 2 (40 s), par un occupant qu'aucune entite ne montre (un bot entre deux
// images-cles) : aucune entite ne vit a sa creation, la regle se tait — elle ne prete pas au corps le
// depart de l'occupant precedent de l'index.
func TestPorteDepartSeTaitPourLeCorpsDUnOccupantSansEntite(t *testing.T) {
	var cov couverturePorte
	in := porteDepartPositions(550, 51_000, 4)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(550, 50_000, 1, 2)},
		porteDepartScan(), &cov)
	if len(out) != len(in) || cov.CorpsApresDepart != 0 {
		t.Fatalf("retenues %d sur %d : aucune entite ne vit a la creation du corps, rien ne s'ecarte",
			len(out), len(in))
	}
}

// R-B3-INSTABLE (garde) : l'entite qui vit a la creation est instable (index ou designateur change) :
// elle n'elit aucun occupant, la regle se tait.
func TestPorteDepartSeTaitSurUneEntiteInstable(t *testing.T) {
	scan := porteDepartScan()
	scan.Entities[0].Unstable = true
	var cov couverturePorte
	in := porteDepartPositions(548, 60_000, 4)
	out := ecarterApresLeDepart(in, []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)}, scan, &cov)
	if len(out) != len(in) || cov.CorpsApresDepart != 0 {
		t.Fatalf("retenues %d sur %d : une entite instable ne prouve aucun depart", len(out), len(in))
	}
}

// R-B3-JOURNAL : ce que la regle ecarte se dit, par l assemblage, en AVERTISSEMENT et au compteur
// expvar — la couverture servie ne le porte pas, c est sa seule trace.
func TestPorteDepartSeDitAuJournalEtAuCompteur(t *testing.T) {
	avant := compteurExpvar(t, metriqueViesApresDepart)
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(prev)
	in := append(porteFoule(300, 0), porteDepartPositions(548, 60_000, 4)...)
	opt := Options{FrameIntervalMS: 100, PlayerEntities: porteDepartScan(),
		BipedCreations: []grammar.BipedCreation{porteDepartCreation(548, 15_000, 1, 2)}}
	BuildFromPositions(context.Background(), "m", "halo_infinite", in, nil, opt)
	avertie := false
	for _, ligne := range strings.Split(buf.String(), "\n") {
		avertie = avertie || (strings.Contains(ligne, `"level":"WARN"`) && strings.Contains(ligne, "depart prouve"))
	}
	if !avertie {
		t.Errorf("journal %q : attendu un AVERTISSEMENT du corps ecarte", buf.String())
	}
	if apres := compteurExpvar(t, metriqueViesApresDepart); apres != avant+1 {
		t.Errorf("compteur %s : %d -> %d, attendu +1 (un corps)", metriqueViesApresDepart, avant, apres)
	}
}

// compteurExpvar lit un compteur de la carte expvar « levelup » (0 s'il n'existe pas encore).
func compteurExpvar(t *testing.T, nom string) int64 {
	t.Helper()
	m, ok := expvar.Get("levelup").(*expvar.Map)
	if !ok || m == nil {
		t.Fatal("carte expvar « levelup » absente")
	}
	v := m.Get(nom)
	if v == nil {
		return 0
	}
	n, err := strconv.ParseInt(v.String(), 10, 64)
	if err != nil {
		t.Fatalf("compteur %s illisible (%q) : %v", nom, v.String(), err)
	}
	return n
}
