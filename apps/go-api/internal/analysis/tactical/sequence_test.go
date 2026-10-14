package tactical

import (
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

const pasImage = 100 // ms par image dans les fixtures

// piste fabrique une piste image par image : x(T) donne l'abscisse, y vaut y0.
func piste(deT, aT int, x func(t int) float32, y0 float32) []domain.TacticalPistePoint {
	var out []domain.TacticalPistePoint
	for t := deT; t <= aT; t++ {
		out = append(out, domain.TacticalPistePoint{T: t, X: x(t), Y: y0})
	}
	return out
}

// entreeDeBase : la victime court le long de x (1 m par image) et meurt a l'image 100 (10 000 ms) en
// x = 100 ; un coequipier immobile a 18 m du point de mort ; un tueur immobile a 40 m.
func entreeDeBase() EntreeSequence {
	t0 := int64(2000)
	return EntreeSequence{
		Victime: ActeurSequence{Role: domain.TacticalRoleVictime, XUID: "111",
			Points: piste(0, 100, func(t int) float32 { return float32(t) }, 0)},
		Tueur: &ActeurSequence{Role: domain.TacticalRoleTueur, XUID: "900",
			Points: piste(0, 200, func(int) float32 { return 140 }, 0)},
		Coequipiers: []ActeurSequence{{Role: domain.TacticalRoleCoequipier, XUID: "222",
			Points: piste(0, 200, func(int) float32 { return 118 }, 0)}},
		FrameIntervalMs: pasImage, MortFilmMs: 10_000, MortMatchMs: 12_000, T0Ms: &t0, RayonM: 18,
	}
}

func TestSequence_CercleSuitLaVictimePuisResteAuPointDeMort(t *testing.T) {
	seq, err := Sequence(entreeDeBase())
	if err != nil {
		t.Fatal(err)
	}
	if seq.Mort.X != 100 {
		t.Fatalf("point de mort = %+v, attendu x = 100", seq.Mort)
	}
	centre := map[int]float64{}
	for _, e := range seq.Etapes {
		centre[e.OffsetMs] = e.Centre.X
	}
	if centre[-4000] != 60 || centre[-1000] != 90 {
		t.Errorf("avant la mort le cercle suit la victime : %v", centre)
	}
	if centre[0] != 100 || centre[1000] != 100 || centre[3000] != 100 {
		t.Errorf("des l'instant 0 le cercle reste au point de mort : %v", centre)
	}
	if len(seq.Etapes) != 8 || seq.Etapes[0].OffsetMs != -4000 || seq.Etapes[7].OffsetMs != 3000 {
		t.Errorf("etapes = %d, de %d a %d, attendu 8 de -4000 a 3000", len(seq.Etapes), seq.Etapes[0].OffsetMs, seq.Etapes[len(seq.Etapes)-1].OffsetMs)
	}
}

func TestSequence_CompteDansLeRayon_BorneInclusive(t *testing.T) {
	seq, _ := Sequence(entreeDeBase())
	par := map[int]domain.TacticalSequenceEtape{}
	for _, e := range seq.Etapes {
		par[e.OffsetMs] = e
	}
	// a -4000 : centre x = 60, coequipier a 118 -> 58 m : hors rayon, mais positionne.
	if e := par[-4000]; e.DansLeRayon != 0 || e.Positionnes != 1 {
		t.Errorf("-4000 : %+v", e)
	}
	// a 0 : centre x = 100, coequipier a 118 -> 18 m pile : dans le rayon (borne inclusive).
	if e := par[0]; e.DansLeRayon != 1 {
		t.Errorf("0 : %+v, 18 m pour un rayon de 18 m est dans le rayon", e)
	}
}

func TestSequence_HorlogeAfficheeDuT0_PlancherAZero(t *testing.T) {
	seq, _ := Sequence(entreeDeBase())
	// mort a 12 000 ms de match, coup d'envoi a 2 000 : -4 s -> 6 000 ; la mort -> 10 000.
	for _, e := range seq.Etapes {
		if e.OffsetMs == -4000 && (e.InstantAfficheMs == nil || *e.InstantAfficheMs != 6000) {
			t.Errorf("-4000 : %v", e.InstantAfficheMs)
		}
		if e.OffsetMs == 0 && *e.InstantAfficheMs != 10_000 {
			t.Errorf("0 : %v", *e.InstantAfficheMs)
		}
	}
	in := entreeDeBase()
	t0 := int64(11_000) // coup d'envoi apres le debut de la sequence : plancher a 0
	in.T0Ms = &t0
	seq, _ = Sequence(in)
	if *seq.Etapes[0].InstantAfficheMs != 0 {
		t.Errorf("avant le coup d'envoi : %d, attendu 0", *seq.Etapes[0].InstantAfficheMs)
	}
	in.T0Ms = nil
	seq, _ = Sequence(in)
	if seq.Etapes[0].InstantAfficheMs != nil {
		t.Error("T0 inconnu : pas d'horloge affichee")
	}
}

func TestSequence_PasDInterpolationAUneLacune(t *testing.T) {
	in := entreeDeBase()
	// le coequipier disparait du film de l'image 70 a l'image 79 : le point 80 porte la lacune.
	var pts []domain.TacticalPistePoint
	for _, p := range piste(0, 200, func(t int) float32 { return float32(t) }, 5) {
		if p.T >= 71 && p.T <= 79 {
			continue
		}
		if p.T == 80 {
			p.G = 900
		}
		pts = append(pts, p)
	}
	in.Coequipiers[0].Points = pts
	seq, _ := Sequence(in)
	var trace domain.TacticalSequenceActeur
	for _, a := range seq.Acteurs {
		if a.XUID == "222" {
			trace = a
		}
	}
	for _, p := range trace.Points {
		// entre les images 70 et 80 (offsets -3000 et -2000) rien n'est invente.
		if p.OffsetMs > -3000 && p.OffsetMs < -2000 {
			t.Fatalf("echantillon %+v dans la lacune : une trace ne s'interpole pas a travers G > 0", p)
		}
		if p.OffsetMs == -2000 && !p.Coupure {
			t.Errorf("le premier echantillon apres la lacune doit etre marque Coupure : %+v", p)
		}
		if p.OffsetMs == -1800 && p.Coupure {
			t.Errorf("la trace reprend sans coupure : %+v", p)
		}
	}
}

func TestSequence_VictimeSansPositionAlaMort(t *testing.T) {
	in := entreeDeBase()
	in.Victime.Points = piste(0, 40, func(t int) float32 { return float32(t) }, 0) // fin du film bien avant
	if _, err := Sequence(in); !errors.Is(err, ErrSequenceSansMort) {
		t.Fatalf("err = %v, attendu ErrSequenceSansMort", err)
	}
}

func TestEquipeDeLaMort_StatutDistancesEtOrdre(t *testing.T) {
	x1, y1, d1 := 6.0, 0.0, 6.0
	x2, y2, d2 := 30.0, 0.0, 30.0
	ctx := domain.ContexteDeMort{Visibles: 2, HorsDeVue: 1, EnAttente: 0, Total: 3, PlusProcheM: &d1}
	lignes := []domain.MateDeMort{
		{MateXUID: "1000", Etat: domain.TacticalMateVisible, X: &x2, Y: &y2, DistanceM: &d2},
		{MateXUID: "999", Etat: domain.TacticalMateHorsDeVue},
		{MateXUID: "222", Etat: domain.TacticalMateVisible, X: &x1, Y: &y1, DistanceM: &d1},
	}
	tueur := &domain.TacticalPoint{X: 6, Y: 8}
	eq := EquipeDeLaMort(ctx, lignes, 18, true, tueur)
	if eq.Statut != domain.TacticalStatutAccompagnee || eq.RayonM == nil || *eq.RayonM != 18 {
		t.Fatalf("statut/rayon = %q / %v", eq.Statut, eq.RayonM)
	}
	ordre := []string{eq.Coequipiers[0].XUID, eq.Coequipiers[1].XUID, eq.Coequipiers[2].XUID}
	if ordre[0] != "222" || ordre[1] != "999" || ordre[2] != "1000" {
		t.Fatalf("ordre = %v, attendu numerique 222, 999, 1000", ordre)
	}
	proche := eq.Coequipiers[0]
	if !proche.DansLeRayon || proche.DistanceTueurM == nil || *proche.DistanceTueurM != 8 {
		t.Errorf("coequipier proche = %+v (distance au tueur attendue 8)", proche)
	}
	if lointain := eq.Coequipiers[2]; lointain.DansLeRayon {
		t.Errorf("a 30 m d'un rayon de 18 : hors rayon, %+v", lointain)
	}
	if hors := eq.Coequipiers[1]; hors.Point != nil || hors.DistanceMortM != nil || hors.DansLeRayon {
		t.Errorf("hors de vue : ni position ni distance, %+v", hors)
	}
	if sans := EquipeDeLaMort(ctx, lignes, 0, false, nil); sans.Statut != "" || sans.RayonM != nil || sans.Coequipiers[0].DansLeRayon {
		t.Errorf("sans rayon : ni statut ni appartenance au rayon, %+v", sans)
	}
}

func TestKillDeLaMort_ExactPuisTolerance(t *testing.T) {
	j := []domain.KillDuJournal{
		{VictimXUID: "111", TimeMs: 10_900}, {VictimXUID: "111", TimeMs: 10_000}, {VictimXUID: "222", TimeMs: 10_000},
	}
	if k := KillDeLaMort(j, "111", 10_000); k == nil || k.TimeMs != 10_000 {
		t.Errorf("exact : %+v", k)
	}
	if k := KillDeLaMort(j, "111", 10_800); k == nil || k.TimeMs != 10_900 {
		t.Errorf("le plus proche dans la tolerance : %+v", k)
	}
	if KillDeLaMort(j, "111", 12_500) != nil {
		t.Error("hors tolerance : nil")
	}
}
