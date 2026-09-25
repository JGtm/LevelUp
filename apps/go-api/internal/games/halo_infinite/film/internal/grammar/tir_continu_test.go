package grammar

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// tir_continu_test.go — LES RAFALES DU TIR CONTINU, pliees des verdicts de vue C paquet par paquet
// (lot M4b). Chaque cas est une suite de paquets : lu (avec des entrees) ou TROU.

// paquetTC est un paquet du scenario : son instant, et son verdict.
type paquetTC struct {
	ts      uint64
	trou    bool
	entrees []EntreeDeControle
}

// tire rend l entree d un joueur qui tient la gachette principale de sa main 0 (ou non).
func tire(index int, tenue bool) EntreeDeControle {
	e := EntreeDeControle{Index: index, Bloc: true, Action: BlocDAction{Present: true,
		Arme: [2]int{ArmeAbsente, ArmeAbsente}}}
	if tenue {
		e.Action.Gachettes[0], e.Action.Arme[0] = 1, 0
	}
	return e
}

// plier joue un scenario dans le collecteur et rend les rafales et les compteurs.
func plier(ps []paquetTC) ([]types.ContinuousFireBurst, types.ContinuousFireStats) {
	var st types.ContinuousFireStats
	c := nouveauCollecteurTirContinu(&st)
	for _, p := range ps {
		c.ouvrir(p.ts)
		if !p.trou {
			c.recevoir(LectureVueC{Atteinte: true, Fermee: true, Entrees: p.entrees})
		}
		c.fermer(false)
	}
	return c.terminer(), st
}

// TestRafaleDePresseALacher : l entree lue qui pose le bit ouvre la rafale, celle qui ne le pose
// plus la ferme — comme Theater. Un paquet lu sans l entree du joueur ne change rien.
func TestRafaleDePresseALacher(t *testing.T) {
	rs, st := plier([]paquetTC{
		{ts: 100, entrees: []EntreeDeControle{tire(2, false)}},
		{ts: 200, entrees: []EntreeDeControle{tire(2, true)}},
		{ts: 300, entrees: []EntreeDeControle{tire(5, false)}}, // l entree d un autre joueur
		{ts: 400, entrees: []EntreeDeControle{tire(2, true)}},
		{ts: 500, entrees: []EntreeDeControle{tire(2, false)}},
	})
	if len(rs) != 1 {
		t.Fatalf("%d rafales, attendu 1 : %+v", len(rs), rs)
	}
	r := rs[0]
	if r.FilmIndex != 2 || r.StartUS != 200 || r.EndUS != 500 ||
		r.StartBound != types.ContinuousFireBoundPressed || r.EndBound != types.ContinuousFireBoundReleased ||
		r.Entries != 2 || len(r.Holes) != 0 || r.Weapon != 0 {
		t.Errorf("rafale %+v, attendu joueur 2, [200, 500), presse -> lache, 2 entrees, arme 0", r)
	}
	if st.Bursts != 1 || st.BurstsWithHole != 0 || st.Firing != 2 || st.Closed != 5 || st.Holes != 0 {
		t.Errorf("compteurs %+v", st)
	}
}

// TestLeTrouEstPorteParLaRafale : un trou pendant une gachette tenue, suivi d une entree qui la
// tient toujours, est un passage INTERIEUR, muet et compte ; la rafale continue.
func TestLeTrouEstPorteParLaRafale(t *testing.T) {
	rs, st := plier([]paquetTC{
		{ts: 1_000, entrees: []EntreeDeControle{tire(2, true)}},
		{ts: 2_000, trou: true},
		{ts: 3_000, trou: true},
		{ts: 4_000, entrees: []EntreeDeControle{tire(2, true)}},
		{ts: 5_000, entrees: []EntreeDeControle{tire(2, false)}},
	})
	if len(rs) != 1 || len(rs[0].Holes) != 1 {
		t.Fatalf("rafales %+v, attendu une rafale a un trou interieur", rs)
	}
	if h := rs[0].Holes[0]; h.StartUS != 2_000 || h.EndUS != 4_000 {
		t.Errorf("trou %+v, attendu [2000, 4000) : du premier paquet non lu a l entree suivante", h)
	}
	if st.Holes != 2 || st.HoleRuns != 1 || st.InnerHoles != 1 || st.BurstsWithHole != 1 ||
		st.HeldHoleMS != 2 {
		t.Errorf("compteurs %+v, attendu 2 trous en 1 suite, 1 interieur, 2 ms tus", st)
	}
}

// TestUnLacherDansLeTrouFinitLaRafaleAuTrou : l entree lue apres le trou ne tient plus la
// gachette — le lacher est DANS le trou, la rafale finit au premier paquet non lu.
func TestUnLacherDansLeTrouFinitLaRafaleAuTrou(t *testing.T) {
	rs, _ := plier([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{tire(3, true)}},
		{ts: 20, trou: true},
		{ts: 30, entrees: []EntreeDeControle{tire(3, false)}},
	})
	if len(rs) != 1 || rs[0].EndUS != 20 || rs[0].EndBound != types.ContinuousFireBoundHole {
		t.Fatalf("rafales %+v, attendu une rafale finie a 20 (trou)", rs)
	}
}

// TestUnePresseApresUnTrouCommenceSurLEntree : un bit non tenu avant le trou, pose apres, ouvre
// une rafale sur la premiere entree lue — borne « trou » : le debut reel est dans le trou.
func TestUnePresseApresUnTrouCommenceSurLEntree(t *testing.T) {
	rs, _ := plier([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{tire(3, false)}},
		{ts: 20, trou: true},
		{ts: 30, entrees: []EntreeDeControle{tire(3, true)}},
		{ts: 40, entrees: []EntreeDeControle{tire(3, false)}},
	})
	if len(rs) != 1 || rs[0].StartUS != 30 || rs[0].StartBound != types.ContinuousFireBoundHole {
		t.Fatalf("rafales %+v, attendu une rafale ouverte a 30 (trou)", rs)
	}
}

// TestUneEntreeSansBlocNeDitRien : une entree qui ne porte pas le bloc de 0x68 ne lache pas la
// gachette et ne leve pas l inconnu d un trou.
func TestUneEntreeSansBlocNeDitRien(t *testing.T) {
	sansBloc := EntreeDeControle{Index: 1}
	rs, _ := plier([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{tire(1, true)}},
		{ts: 20, entrees: []EntreeDeControle{sansBloc}},
		{ts: 30, entrees: []EntreeDeControle{tire(1, false)}},
	})
	if len(rs) != 1 || rs[0].EndUS != 30 {
		t.Fatalf("rafales %+v, attendu une rafale [10, 30)", rs)
	}
}

// TestFinDeFilm : une rafale tenue au dernier paquet finit au dernier paquet ; tenue avec un trou
// OUVERT, elle finit au debut du trou (rien ne dit qu elle a dure au-dela).
func TestFinDeFilm(t *testing.T) {
	rs, _ := plier([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{tire(1, true), tire(4, true)}},
		{ts: 20, entrees: []EntreeDeControle{tire(4, true)}},
		{ts: 30, trou: true},
		{ts: 40, entrees: []EntreeDeControle{tire(4, true)}},
	})
	if len(rs) != 2 {
		t.Fatalf("%d rafales, attendu 2 : %+v", len(rs), rs)
	}
	if rs[0].FilmIndex != 1 || rs[0].EndUS != 30 || rs[0].EndBound != types.ContinuousFireBoundHole {
		t.Errorf("joueur 1 : %+v, attendu fin au trou ouvert (30)", rs[0])
	}
	if rs[1].FilmIndex != 4 || rs[1].EndUS != 40 || rs[1].EndBound != types.ContinuousFireBoundFilmEnd ||
		len(rs[1].Holes) != 1 {
		t.Errorf("joueur 4 : %+v, attendu fin du film (40) avec un trou interieur", rs[1])
	}
}

// TestLesCausesDesTrousSontVentilees : chaque paquet non lu tombe sous UNE cause.
func TestLesCausesDesTrousSontVentilees(t *testing.T) {
	var st types.ContinuousFireStats
	c := nouveauCollecteurTirContinu(&st)
	for i, l := range []LectureVueC{
		{},
		{Atteinte: true},
		{Atteinte: true, Arret: ArretVueCKindNonPorte},
		{Atteinte: true, Arret: ArretVueCBlocBC},
		{Atteinte: true, Arret: ArretVueCDebordement},
	} {
		c.ouvrir(uint64(i + 1))
		c.recevoir(l)
		c.fermer(false)
	}
	c.ouvrir(9)
	c.fermer(true)
	if st.Holes != 6 || st.OpenViewB != 1 || st.NotClosing != 1 || st.StopKind != 1 ||
		st.StopBlockBC != 1 || st.StopOverflow != 1 || st.Unlocated != 1 || st.Reached != 4 {
		t.Errorf("ventilation %+v", st)
	}
}
