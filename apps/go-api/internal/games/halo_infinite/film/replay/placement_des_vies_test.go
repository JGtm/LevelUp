package replay

// placement_des_vies_test.go — UNE RÈGLE DE LA MESURE D'UNE VIE PAR TEST (lot V1.2 du plan
// `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`).
//
// Chaque test a été vu ROUGE sous une mutation nommée au journal du lot V1 du plan, puis la
// mutation a été retirée. Les fixtures nomment leurs vies comme un film : par les morts du fil,
// avec le VRAI registre (`BuildIdentityRegistry`) — deux morts à deux instants différents épinglent
// le calage d'horloge à zéro, si bien que l'horloge du match est celle des positions.
//
// LE DÉCOR COMMUN : moi (111, slot 1, camp 0), un coéquipier (222, slot 2, camp 0), un adversaire
// (999, slot 9, camp 1), des pistes échantillonnées toutes les 100 ms (`pisteMonde`).

import (
	"fmt"
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// decorPlacement assemble une entrée à partir des pistes et des morts du fil, camps du décor.
func decorPlacement(pos []grammar.BipedPosition, mortsFilm []types.Death) EntreePlacement {
	reg := BuildIdentityRegistry(IdentityInput{
		Positions: pos, Deaths: mortsFilm, PlayerIndices: indexDe(111, 222, 999),
	})
	return EntreePlacement{
		Positions: pos,
		Registre:  reg,
		Equipes:   map[uint64]int{111: 0, 222: 0, 999: 1},
	}
}

// adversaireLointain : l'adversaire, loin de tout, vivant jusqu'à 15 s — il ne doit peser sur
// aucune mesure.
func adversaireLointain() ([]grammar.BipedPosition, types.Death) {
	return pisteMonde(9, 0, 15_000, 50, 50), mortFilm(999, 15_000)
}

// decorNominal : moi de 0 à 10 s en (0,0), le coéquipier de 0 à 12 s en (3,0).
func decorNominal() EntreePlacement {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 10_000, 0, 0), pisteMonde(2, 0, 12_000, 3, 0)...)
	pos = append(pos, adv...)
	return decorPlacement(pos, []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
}

// decorEquipeATerre : le coéquipier meurt à 4 s et ne revient qu'à 12 s.
func decorEquipeATerre() EntreePlacement {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 10_000, 0, 0), pisteMonde(2, 0, 4_000, 3, 0)...)
	pos = append(pos, pisteMonde(2, 12_000, 16_000, 3, 0)...)
	pos = append(pos, adv...)
	return decorPlacement(pos, []types.Death{
		mortFilm(222, 4_000), mortFilm(111, 10_000), mortFilm(222, 16_000), mAdv,
	})
}

// vieDe retrouve la vie d'un joueur qui commence à un instant, ou fait échouer le test.
func placementDe(t *testing.T, vies []PlacementVie, xuid uint64, debutMS int64) PlacementVie {
	t.Helper()
	for _, v := range vies {
		if v.XUID == xuid && v.DebutMS == debutMS {
			return v
		}
	}
	t.Fatalf("aucune vie de %d commencant a %d ms — rendues : %+v", xuid, debutMS, vies)
	return PlacementVie{}
}

// TestPlacementDesVies_OrdreDesCauses — UN INSTANT, UNE CAUSE, DANS L'ORDRE DE LA DÉCISION V3.
//
// Le coéquipier est à terre de 4,1 à 10 s. Je porte l'objectif de 5 à 5,9 s : ces dix instants
// sont À LA FOIS « porteur » et « équipe à terre », et c'est « porteur » qui les prend. Les
// cinquante autres restent « équipe à terre ».
func TestPlacementDesVies_OrdreDesCauses(t *testing.T) {
	e := decorEquipeATerre()
	e.Portages = map[uint64][]IntervalleDePort{111: {{DebutMS: 5_000, FinMS: 5_900}}}
	vies, _ := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 0)
	if v.PorteurMS != 1_000 || v.EquipeATerreMS != 5_000 {
		t.Fatalf("porteur %d ms / equipe a terre %d ms, attendu 1 000 / 5 000 : le port "+
			"d'objectif PRIME sur l'equipe a terre (%+v)", v.PorteurMS, v.EquipeATerreMS, v)
	}
}

// TestPlacementDesVies_EquipeATerre — aucun coéquipier vivant : il n'y a personne de qui être
// proche, et ces instants ne sont PAS mesurés (sans la règle, une distance infinie entrerait
// dans la médiane).
func TestPlacementDesVies_EquipeATerre(t *testing.T) {
	vies, _ := PlacementDesVies(decorEquipeATerre())
	v := placementDe(t, vies, 111, 0)
	if v.EquipeATerreMS != 6_000 || v.MesureMS != 4_100 {
		t.Fatalf("equipe a terre %d ms / mesure %d ms, attendu 6 000 / 4 100 (%+v)",
			v.EquipeATerreMS, v.MesureMS, v)
	}
	if v.MedianeM == nil || math.Abs(*v.MedianeM-3) > 0.001 {
		t.Fatalf("mediane = %v, attendu 3 m (les seuls instants mesures)", enClair(v.MedianeM))
	}
}

// TestPlacementDesVies_Porteur — le port d'objectif sort de la mesure, bornes INCLUSES.
func TestPlacementDesVies_Porteur(t *testing.T) {
	e := decorNominal()
	e.Portages = map[uint64][]IntervalleDePort{111: {{DebutMS: 2_000, FinMS: 2_900}}}
	vies, _ := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 0)
	if v.PorteurMS != 1_000 || v.MesureMS != 9_100 {
		t.Fatalf("porteur %d ms / mesure %d ms, attendu 1 000 / 9 100 : l'intervalle "+
			"[2 000, 2 900] couvre DIX instants de la grille (%+v)", v.PorteurMS, v.MesureMS, v)
	}
}

// TestPlacementDesVies_JoueurNonSitue — mes positions s'interrompent 2 s (sous le trou de 5 s
// qui couperait la vie) : pendant la première seconde la dernière position reste valable, puis
// je ne suis plus situé — c'est le véhicule, mesuré à 96 % en V0.1.
func TestPlacementDesVies_JoueurNonSitue(t *testing.T) {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 4_000, 0, 0), pisteMonde(1, 6_000, 10_000, 0, 0)...)
	pos = append(pos, pisteMonde(2, 0, 12_000, 3, 0)...)
	pos = append(pos, adv...)
	e := decorPlacement(pos, []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
	vies, _ := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 0)
	if v.NonSitueMS != 900 || v.CoequipierNonSitueMS != 0 || v.MesureMS != 9_200 {
		t.Fatalf("non situe %d / coequipier non situe %d / mesure %d ms, attendu 900 / 0 / "+
			"9 200 (%+v)", v.NonSitueMS, v.CoequipierNonSitueMS, v.MesureMS, v)
	}
}

// TestPlacementDesVies_CoequipierNonSitue — le coéquipier VIVANT cesse d'être répliqué : la
// distance au plus proche ne se calcule pas sans lui, l'instant sort de la mesure.
func TestPlacementDesVies_CoequipierNonSitue(t *testing.T) {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 10_000, 0, 0), pisteMonde(2, 0, 4_000, 3, 0)...)
	pos = append(pos, pisteMonde(2, 6_000, 12_000, 3, 0)...)
	pos = append(pos, adv...)
	e := decorPlacement(pos, []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
	vies, _ := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 0)
	if v.CoequipierNonSitueMS != 900 || v.MesureMS != 9_200 {
		t.Fatalf("coequipier non situe %d ms / mesure %d ms, attendu 900 / 9 200 (%+v)",
			v.CoequipierNonSitueMS, v.MesureMS, v)
	}
}

// TestPlacementDesVies_Mediane — la MÉDIANE, pas la moyenne, et la moyenne des deux valeurs
// centrales sur un nombre pair d'instants.
//
// Cent instants : quarante à 2 m, dix à 3 m, cinquante à 10 m. Médiane (3 + 10) / 2 = 6,5 m ;
// la moyenne vaudrait 6,1 m, la valeur centrale haute seule 10 m.
func TestPlacementDesVies_Mediane(t *testing.T) {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 9_900, 0, 0), pisteMonde(2, 0, 3_900, 2, 0)...)
	pos = append(pos, pisteMonde(2, 4_000, 4_900, 3, 0)...)
	pos = append(pos, pisteMonde(2, 5_000, 12_000, 10, 0)...)
	pos = append(pos, adv...)
	e := decorPlacement(pos, []types.Death{mortFilm(111, 9_900), mortFilm(222, 12_000), mAdv})
	vies, _ := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 0)
	if v.MesureMS != 10_000 {
		t.Fatalf("mesure %d ms, attendu 10 000 (cent instants) — %+v", v.MesureMS, v)
	}
	if v.MedianeM == nil || math.Abs(*v.MedianeM-6.5) > 0.001 {
		t.Fatalf("mediane = %v, attendu 6,5 m", enClair(v.MedianeM))
	}
}

// TestPlacementDesVies_HorsRadar — le temps mesuré où la distance DÉPASSE la portée : une
// distance ÉGALE à la portée est à portée.
func TestPlacementDesVies_HorsRadar(t *testing.T) {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 10_000, 0, 0), pisteMonde(2, 0, 4_900, 3, 0)...)
	pos = append(pos, pisteMonde(2, 5_000, 12_000, 5, 0)...)
	pos = append(pos, adv...)
	e := decorPlacement(pos, []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
	radar := 3.0
	e.RadarM = &radar
	vies, _ := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 0)
	if v.HorsRadarMS == nil || *v.HorsRadarMS != 5_100 {
		t.Fatalf("hors radar = %v, attendu 5 100 ms (les 51 instants a 5 m ; les 50 a "+
			"3 m pile sont a portee)", enClair(v.HorsRadarMS))
	}
	if v.RadarM == nil || *v.RadarM != radar {
		t.Fatalf("portee recopiee = %v, attendu %v", v.RadarM, radar)
	}
}

// TestPlacementDesVies_VieCourteNonMesuree — sous 2 000 ms MESURÉES, pas de médiane ; à 2 000
// pile, la médiane sort.
func TestPlacementDesVies_VieCourteNonMesuree(t *testing.T) {
	for _, cas := range []struct {
		nom      string
		finMS    int64
		mesureMS int64
		publiee  bool
	}{
		{"dix-neuf instants", 1_800, 1_900, false},
		{"vingt instants", 1_900, 2_000, true},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			adv, mAdv := adversaireLointain()
			pos := append(pisteMonde(1, 0, cas.finMS, 0, 0), pisteMonde(2, 0, 12_000, 3, 0)...)
			pos = append(pos, adv...)
			e := decorPlacement(pos, []types.Death{mortFilm(111, cas.finMS), mortFilm(222, 12_000), mAdv})
			vies, _ := PlacementDesVies(e)
			v := placementDe(t, vies, 111, 0)
			if v.MesureMS != cas.mesureMS || (v.MedianeM != nil) != cas.publiee {
				t.Fatalf("mesure %d ms, mediane %v : attendu %d ms et mediane publiee = %v",
					v.MesureMS, enClair(v.MedianeM), cas.mesureMS, cas.publiee)
			}
		})
	}
}

// decorDeuxVies : ma première vie de 0 à 10 s, la seconde de 16 à 25 s (le trou de 6 s coupe).
func decorDeuxVies() EntreePlacement {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 0, 10_000, 0, 0), pisteMonde(1, 16_000, 25_000, 0, 0)...)
	pos = append(pos, pisteMonde(2, 0, 12_000, 3, 0)...)
	pos = append(pos, adv...)
	return decorPlacement(pos, []types.Death{
		mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv, mortFilm(111, 25_000),
	})
}

// TestPlacementDesVies_FragPosthume — un frag APRÈS ma mort et avant ma réapparition (grenade,
// échange) appartient à la vie qui vient de finir.
func TestPlacementDesVies_FragPosthume(t *testing.T) {
	e := decorDeuxVies()
	e.Journal = []FragDuJournal{{TueurXUID: 111, VictimeXUID: 999, TempsMS: 10_200, Publiable: true}}
	vies, bilan := PlacementDesVies(e)
	premiere, seconde := placementDe(t, vies, 111, 0), placementDe(t, vies, 111, 16_000)
	if premiere.Frags != 1 || seconde.Frags != 0 || bilan.FragsRattaches != 1 {
		t.Fatalf("frags premiere %d / seconde %d / rattaches %d, attendu 1 / 0 / 1",
			premiere.Frags, seconde.Frags, bilan.FragsRattaches)
	}
}

// TestPlacementDesVies_FragAvantLaPremiereVie — compté dans le bilan, rattaché à aucune vie.
func TestPlacementDesVies_FragAvantLaPremiereVie(t *testing.T) {
	adv, mAdv := adversaireLointain()
	pos := append(pisteMonde(1, 2_000, 10_000, 0, 0), pisteMonde(2, 0, 12_000, 3, 0)...)
	pos = append(pos, adv...)
	e := decorPlacement(pos, []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
	e.Journal = []FragDuJournal{{TueurXUID: 111, VictimeXUID: 999, TempsMS: 1_000, Publiable: true}}
	vies, bilan := PlacementDesVies(e)
	v := placementDe(t, vies, 111, 2_000)
	if v.Frags != 0 || bilan.FragsAvantLaPremiereVie != 1 || bilan.FragsRattaches != 0 {
		t.Fatalf("frags de la vie %d / avant la premiere vie %d / rattaches %d, attendu 0 / 1 / 0",
			v.Frags, bilan.FragsAvantLaPremiereVie, bilan.FragsRattaches)
	}
}

// TestPlacementDesVies_TrahisonExclue — un frag sur un coéquipier n'est pas un frag de la vie.
func TestPlacementDesVies_TrahisonExclue(t *testing.T) {
	e := decorNominal()
	e.Journal = []FragDuJournal{{TueurXUID: 111, VictimeXUID: 222, TempsMS: 5_000, Publiable: true}}
	vies, bilan := PlacementDesVies(e)
	if v := placementDe(t, vies, 111, 0); v.Frags != 0 || bilan.Trahisons != 1 {
		t.Fatalf("frags %d / trahisons %d, attendu 0 / 1", v.Frags, bilan.Trahisons)
	}
}

// TestPlacementDesVies_FragNonPubliableExclu — une passe non publiable ne donne aucun frag.
func TestPlacementDesVies_FragNonPubliableExclu(t *testing.T) {
	e := decorNominal()
	e.Journal = []FragDuJournal{{TueurXUID: 111, VictimeXUID: 999, TempsMS: 5_000}}
	vies, bilan := PlacementDesVies(e)
	if v := placementDe(t, vies, 111, 0); v.Frags != 0 || bilan.FragsNonPubliables != 1 {
		t.Fatalf("frags %d / non publiables %d, attendu 0 / 1", v.Frags, bilan.FragsNonPubliables)
	}
}

// TestPlacementDesVies_VarianteSansPortee — sans portée de radar, ni portée ni temps hors radar :
// NULL, jamais un zéro qui se lirait « toujours à portée ».
func TestPlacementDesVies_VarianteSansPortee(t *testing.T) {
	vies, _ := PlacementDesVies(decorNominal())
	v := placementDe(t, vies, 111, 0)
	if v.RadarM != nil || v.HorsRadarMS != nil {
		t.Fatalf("portee %v / hors radar %v, attendu nil / nil", enClair(v.RadarM), enClair(v.HorsRadarMS))
	}
	if v.MedianeM == nil {
		t.Fatal("mediane nil : l'absence de portee ne doit pas retirer la mesure")
	}
}

// TestPlacementDesVies_PontNonPubliable — décision V1 amendée le 2026-09-29 (lot V2b) : une
// identité lue de deux façons rend l'attribution des positions fausse, donc RIEN n'est mesuré,
// mais chaque vie nommée est RENDUE, entière « non située » (même le portage, qui s'attribue lui
// aussi par le pont), la portée recopiée, les frags rattachés comme ailleurs, le refus dit.
func TestPlacementDesVies_PontNonPubliable(t *testing.T) {
	e := decorNominal()
	e.Registre.own.IndexDisagreements = 1
	r := 18.0
	e.RadarM = &r
	e.Portages = map[uint64][]IntervalleDePort{111: {{DebutMS: 5_000, FinMS: 5_900}}}
	e.Journal = []FragDuJournal{{TueurXUID: 111, VictimeXUID: 999, TempsMS: 5_000, Publiable: true}}
	vies, bilan := PlacementDesVies(e)
	if len(vies) != len(e.Registre.ViesNommees()) || len(vies) == 0 || !bilan.PontNonPubliable {
		t.Fatalf("%d vies pour %d vies nommees, pont non publiable = %v : attendu une ligne par vie "+
			"et le refus dit", len(vies), len(e.Registre.ViesNommees()), bilan.PontNonPubliable)
	}
	for _, v := range vies {
		grille := ((v.FinMS-v.DebutMS)/PasDeLaGrilleDesViesMs + 1) * PasDeLaGrilleDesViesMs
		if v.NonSitueMS != grille || v.MesureMS != 0 || v.PorteurMS != 0 || v.EquipeATerreMS != 0 ||
			v.CoequipierNonSitueMS != 0 || v.MedianeM != nil || v.DerniereMesure != nil {
			t.Fatalf("vie %+v : attendu %d ms non situees, rien d'autre, aucune mediane", v, grille)
		}
		if v.RadarM == nil || *v.RadarM != 18 || v.HorsRadarMS == nil || *v.HorsRadarMS != 0 {
			t.Fatalf("vie de %d : portee %s / hors radar %s, attendu 18 / 0", v.XUID,
				enClair(v.RadarM), enClair(v.HorsRadarMS))
		}
	}
	if v := placementDe(t, vies, 111, 0); v.Frags != 1 || bilan.FragsRattaches != 1 {
		t.Fatalf("frags de la vie %d / rattaches %d, attendu 1 / 1", v.Frags, bilan.FragsRattaches)
	}
}

// enClair rend la valeur pointée, ou « nil » — un pointeur imprimé tel quel ne dit rien.
func enClair[T any](p *T) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprint(*p)
}

// ─── COMPLÉMENTS DE LA REVUE ADVERSARIALE V5.1, R3 (2026-09-30) ──────────────────────────────
//
// Trois trous relevés par le relecteur, chacun fermé par un test rouge sous la mutation nommée.

// TestPlacementDesVies_FragALInstantDuDebutDeLaVieSuivante — la borne est `[début, début
// suivant)` : le frag posé EXACTEMENT au début de ma seconde vie est celui de la seconde, pas de
// la première. Mutation vue rouge : `>` en `>=` dans le `sort.Search` du rattachement.
func TestPlacementDesVies_FragALInstantDuDebutDeLaVieSuivante(t *testing.T) {
	for _, cas := range []struct {
		nom               string
		tempsMS           int64
		premiere, seconde int
	}{
		{"une milliseconde avant le debut", 15_999, 1, 0},
		{"exactement au debut", 16_000, 0, 1},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			e := decorDeuxVies()
			e.Journal = []FragDuJournal{{TueurXUID: 111, VictimeXUID: 999, TempsMS: cas.tempsMS, Publiable: true}}
			vies, bilan := PlacementDesVies(e)
			p, s := placementDe(t, vies, 111, 0), placementDe(t, vies, 111, 16_000)
			if p.Frags != cas.premiere || s.Frags != cas.seconde || bilan.FragsRattaches != 1 {
				t.Fatalf("frags premiere %d / seconde %d / rattaches %d, attendu %d / %d / 1",
					p.Frags, s.Frags, bilan.FragsRattaches, cas.premiere, cas.seconde)
			}
		})
	}
}

// TestPlacementDesVies_CausesSimultanees — trois PAIRES de causes vraies au même instant : c'est
// la plus prioritaire qui prend l'instant, l'autre n'est pas comptée. Mutation vue rouge : le
// bloc de la cause seconde placé avant celui de la première dans `classer`.
func TestPlacementDesVies_CausesSimultanees(t *testing.T) {
	adv, mAdv := adversaireLointain()
	// Mes positions s'interrompent de 4 à 6 s : non situé de 5,1 à 5,9 s (cf. JoueurNonSitue).
	mesTrous := append(pisteMonde(1, 0, 4_000, 0, 0), pisteMonde(1, 6_000, 10_000, 0, 0)...)

	t.Run("porteur et non situe : porteur", func(t *testing.T) {
		pos := append(append([]grammar.BipedPosition{}, mesTrous...), pisteMonde(2, 0, 12_000, 3, 0)...)
		e := decorPlacement(append(pos, adv...), []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
		e.Portages = map[uint64][]IntervalleDePort{111: {{DebutMS: 5_000, FinMS: 5_900}}}
		v := placementDe(t, mustVies(PlacementDesVies(e)), 111, 0)
		if v.PorteurMS != 1_000 || v.NonSitueMS != 0 || v.MesureMS != 9_100 {
			t.Fatalf("porteur %d / non situe %d / mesure %d ms, attendu 1 000 / 0 / 9 100 (%+v)",
				v.PorteurMS, v.NonSitueMS, v.MesureMS, v)
		}
	})

	t.Run("equipe a terre et non situe : equipe a terre", func(t *testing.T) {
		// Le coéquipier meurt à 4 s et ne revient qu'à 12 s (cf. decorEquipeATerre).
		pos := append(append([]grammar.BipedPosition{}, mesTrous...), pisteMonde(2, 0, 4_000, 3, 0)...)
		pos = append(pos, pisteMonde(2, 12_000, 16_000, 3, 0)...)
		e := decorPlacement(append(pos, adv...), []types.Death{
			mortFilm(222, 4_000), mortFilm(111, 10_000), mortFilm(222, 16_000), mAdv,
		})
		v := placementDe(t, mustVies(PlacementDesVies(e)), 111, 0)
		if v.EquipeATerreMS != 6_000 || v.NonSitueMS != 0 {
			t.Fatalf("equipe a terre %d / non situe %d ms, attendu 6 000 / 0 (%+v)",
				v.EquipeATerreMS, v.NonSitueMS, v)
		}
	})

	t.Run("non situe et coequipier non situe : non situe", func(t *testing.T) {
		pos := append(append([]grammar.BipedPosition{}, mesTrous...), pisteMonde(2, 0, 4_000, 3, 0)...)
		pos = append(pos, pisteMonde(2, 6_000, 12_000, 3, 0)...)
		e := decorPlacement(append(pos, adv...), []types.Death{mortFilm(111, 10_000), mortFilm(222, 12_000), mAdv})
		v := placementDe(t, mustVies(PlacementDesVies(e)), 111, 0)
		if v.NonSitueMS != 900 || v.CoequipierNonSitueMS != 0 || v.MesureMS != 9_200 {
			t.Fatalf("non situe %d / coequipier non situe %d / mesure %d ms, attendu 900 / 0 / 9 200 (%+v)",
				v.NonSitueMS, v.CoequipierNonSitueMS, v.MesureMS, v)
		}
	})
}

// mustVies écarte le bilan d'un appel à PlacementDesVies.
func mustVies(v []PlacementVie, _ BilanPlacement) []PlacementVie { return v }

// TestPlacementDesVies_FragsEcartesParLeurRefus — les branches de `fragRecevable` et le refus du
// tueur sans vie : chacune est comptée dans SON compteur du bilan, aucune n'est rattachée.
// Mutations vues rouges : retirer chaque refus ; `||` en `&&` sur les deux camps.
func TestPlacementDesVies_FragsEcartesParLeurRefus(t *testing.T) {
	for _, cas := range []struct {
		nom     string
		frag    FragDuJournal
		attendu BilanPlacement
	}{
		{"tueur inconnu (xuid nul)", FragDuJournal{VictimeXUID: 999, TempsMS: 5_000, Publiable: true},
			BilanPlacement{FragsTueurInconnu: 1}},
		{"camp du tueur inconnu", FragDuJournal{TueurXUID: 777, VictimeXUID: 999, TempsMS: 5_000, Publiable: true},
			BilanPlacement{FragsCampInconnu: 1}},
		{"camp de la victime inconnu", FragDuJournal{TueurXUID: 111, VictimeXUID: 777, TempsMS: 5_000, Publiable: true},
			BilanPlacement{FragsCampInconnu: 1}},
		{"tueur sans aucune vie", FragDuJournal{TueurXUID: 333, VictimeXUID: 999, TempsMS: 5_000, Publiable: true},
			BilanPlacement{FragsSansVie: 1}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			e := decorNominal()
			e.Equipes[333] = 0 // un camp, mais aucune vie nommée au registre
			e.Journal = []FragDuJournal{cas.frag}
			vies, bilan := PlacementDesVies(e)
			if bilan != cas.attendu {
				t.Fatalf("bilan %+v, attendu %+v", bilan, cas.attendu)
			}
			for _, v := range vies {
				if v.Frags != 0 {
					t.Fatalf("la vie de %d porte %d frag(s) : un frag ecarte ne se rattache pas", v.XUID, v.Frags)
				}
			}
		})
	}
}
