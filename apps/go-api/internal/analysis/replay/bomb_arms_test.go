package replay

// bomb_arms_test.go — LA JOINTURE DE L'ARMEMENT, éprouvée SANS film et SANS base.
//
// Ce que ces tests vérifient, au-delà des comptes :
//
//	le RECALAGE      un décalage d'horloge non nul est APPLIQUÉ ; l'oublier casse la jointure
//	                 (TestBombArmsRecalageHorloge — cf. le piège en tête de bomb_arms.go) ;
//	le silence       sans anneau OU sans portage, `bomb_arms` reste absent sur TOUS les
//	                 joueurs, jamais à zéro ;
//	l'acteur absent  un armement que la jointure ne résout pas est PUBLIÉ sans acteur, et sa
//	                 raison est comptée (`ArmingsNoDrop` / `ArmingsNoBridge`) ;
//	l'exclusivité    une même période de portage ne pose qu'une bombe ;
//	la cohérence     somme des `bomb_arms` == ArmingsAttributed, et la ventilation retombe sur
//	                 le nombre d'armements datés.

import (
	"testing"
)

// baArming fabrique un armement daté sur l'horloge du FILM. StartMS/T ne sont pas lus par la
// jointure (elle date sur la FIN de la montée) : ils portent des valeurs plausibles pour que
// l'entrée reste réaliste.
func baArming(timeMS int) BombArming {
	return BombArming{T: timeMS / 100, TimeMS: timeMS,
		StartT: (timeMS - 1200) / 100, StartMS: timeMS - 1200, FuseMS: BombFuseMS}
}

// baPeriodeMort fabrique une période fermée par la MORT du porteur — pas un geste de pose.
func baPeriodeMort(xuid uint64, debut, fin int) HeldObjectPeriod {
	p := bombPeriode(xuid, debut, fin)
	p.FinParMort = true
	return p
}

// baPeriodeOuverte fabrique une période que rien ne ferme avant la fin du film.
func baPeriodeOuverte(xuid uint64, debut int) HeldObjectPeriod {
	return HeldObjectPeriod{Slot: 5, XUID: xuid, DebutMS: debut,
		FinMS: HeldObjectOpenEndMS, Ouverte: true}
}

// TestBombArmsJointure : la règle des quatre conditions, cas par cas.
func TestBombArmsJointure(t *testing.T) {
	cases := []struct {
		nom      string
		armings  []BombArming
		periodes []HeldObjectPeriod
		veut     map[string]int // xuid -> bomb_arms attendu
		veutCov  BombStatsCoverage
	}{
		{
			nom:     "lacher 126 ms apres l'armement : le poseur est nomme",
			armings: []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{
				bombPeriode(7, 80000, 100126),
			},
			veut:    map[string]int{"7": 1},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAttributed: 1},
		},
		{
			nom:     "deux armements, deux poseurs distincts",
			armings: []BombArming{baArming(100000), baArming(200000)},
			periodes: []HeldObjectPeriod{
				bombPeriode(7, 80000, 100126),
				bombPeriode(9, 180000, 200090),
			},
			veut:    map[string]int{"7": 1, "9": 1},
			veutCov: BombStatsCoverage{Armings: 2, ArmingsAttributed: 2},
		},
		{
			nom:     "le lacher le PLUS PROCHE gagne, l'autre reste libre",
			armings: []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{
				bombPeriode(7, 80000, 98000),
				bombPeriode(9, 90000, 100100),
			},
			veut:    map[string]int{"9": 1},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAttributed: 1},
		},
		{
			nom:      "fermee par la MORT : aucun geste de pose, armement sans acteur",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{baPeriodeMort(7, 80000, 100126)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoDrop: 1},
		},
		{
			nom:      "restee OUVERTE : aucun lacher, armement sans acteur",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{baPeriodeOuverte(7, 80000)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoDrop: 1},
		},
		{
			nom:      "lacher HORS FENETRE (2 600 ms) : armement sans acteur",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{bombPeriode(7, 80000, 102600)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoDrop: 1},
		},
		{
			nom:      "prise APRES l'armement : on n'arme pas ce qu'on n'a pas encore",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{bombPeriode(7, 100500, 101000)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoDrop: 1},
		},
		{
			nom:      "slot NON PONTE : armement publie sans acteur, raison distincte",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{bombPeriode(0, 80000, 100126)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoBridge: 1},
		},
		{
			nom:     "exclusivite : une periode ne pose qu'une bombe",
			armings: []BombArming{baArming(100000), baArming(101000)},
			periodes: []HeldObjectPeriod{
				bombPeriode(7, 80000, 100126),
			},
			veut:    map[string]int{"7": 1},
			veutCov: BombStatsCoverage{Armings: 2, ArmingsAttributed: 1, ArmingsNoDrop: 1},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			got, evts := BuildBombStats(BombStatsInput{
				ArmingsRead: true, Armings: c.armings,
				CarryRead: true, Carry: bombCarryDe(c.periodes...),
			})
			for xuid, veut := range c.veut {
				assertBombInt(t, got, xuid, func(p BombPlayerStats) *int { return p.Arms },
					veut, "arms")
			}
			assertBombArmCoverage(t, got.Coverage, c.veutCov)
			assertBombArmEvents(t, evts, c.armings, c.veut)
		})
	}
}

// TestBombArmsRecalageHorloge est LE test du piège : les armements sont datés sur l'horloge du
// FILM, les périodes de portage sur celle du MATCH. Ici le décalage vaut 40 000 ms — très
// au-delà de la fenêtre de jointure — donc supprimer le terme `FilmToMatchOffsetMS` de
// `bombArmsByXUID` fait tomber l'attribution à zéro et rougit ce test.
//
// La valeur est volontairement énorme (les films témoins mesurent 16-81 ms) : un décalage de
// l'ordre du bruit ne prouverait rien, un test qui passe encore sans le recalage ne serait pas
// un test.
func TestBombArmsRecalageHorloge(t *testing.T) {
	const offset = 40000
	// L'armement est daté 100 000 ms sur l'horloge du FILM, soit 140 000 ms sur celle du
	// MATCH — l'horloge des périodes de portage.
	in := BombStatsInput{
		ArmingsRead: true, Armings: []BombArming{baArming(100000)},
		CarryRead: true, Carry: bombCarryDe(bombPeriode(7, 120000, 140126)),
		FilmToMatchOffsetMS: offset,
	}
	got, evts := BuildBombStats(in)
	assertBombInt(t, got, "7", func(p BombPlayerStats) *int { return p.Arms }, 1, "arms")
	if got.Coverage.ArmingsAttributed != 1 {
		t.Fatalf("recalage NON applique : %d armement(s) attribue(s), attendu 1 (couverture %+v)",
			got.Coverage.ArmingsAttributed, got.Coverage)
	}
	// Le fait daté reste publié sur l'horloge du FILM : c'est celle des explosions, et la
	// persistance les pose sur le même axe.
	assertBombEvents(t, evts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000, XUID: "7"}})

	// Contre-épreuve : le MÊME jeu d'entrées sans le recalage ne nomme personne — et
	// l'armement reste publié, sans acteur.
	in.FilmToMatchOffsetMS = 0
	sans, sansEvts := BuildBombStats(in)
	if sans.Coverage.ArmingsAttributed != 0 || sans.Coverage.ArmingsNoDrop != 1 {
		t.Fatalf("sans recalage : couverture %+v, attendu 0 attribue et 1 sans lacher",
			sans.Coverage)
	}
	assertBombEvents(t, sansEvts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000}})
}

// TestBombArmsSourcesNonLues : chacun des deux canaux manquant laisse `bomb_arms` ABSENT, et
// l'anneau seul publie quand même ses faits datés — sans acteur.
func TestBombArmsSourcesNonLues(t *testing.T) {
	armings := []BombArming{baArming(100000)}
	carry := bombCarryDe(bombPeriode(7, 80000, 100126))

	t.Run("anneau non lu : ni champ ni fait date", func(t *testing.T) {
		got, evts := BuildBombStats(BombStatsInput{CarryRead: true, Carry: carry})
		assertBombAbsent(t, got, func(p BombPlayerStats) bool { return p.Arms != nil }, "arms")
		assertBombEvents(t, evts, nil)
		if got.Coverage.Armings != 0 || got.Coverage.ArmingsRead {
			t.Fatalf("anneau non lu : couverture %+v, attendue muette", got.Coverage)
		}
	})

	t.Run("portage non lu : fait date publie SANS acteur, champ absent", func(t *testing.T) {
		got, evts := BuildBombStats(BombStatsInput{ArmingsRead: true, Armings: armings})
		assertBombAbsent(t, got, func(p BombPlayerStats) bool { return p.Arms != nil }, "arms")
		assertBombEvents(t, evts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000}})
		if got.Coverage.Armings != 1 || got.Coverage.ArmingsNoDrop != 1 {
			t.Fatalf("portage non lu : couverture %+v, attendu 1 arme et 1 sans lacher",
				got.Coverage)
		}
	})
}

// assertBombArmCoverage compare la ventilation de la jointure ET vérifie l'INVARIANT publié :
// attribués + sans lâcher + sans pont == armements datés.
func assertBombArmCoverage(t *testing.T, got, veut BombStatsCoverage) {
	t.Helper()
	if got.Armings != veut.Armings || got.ArmingsAttributed != veut.ArmingsAttributed ||
		got.ArmingsNoDrop != veut.ArmingsNoDrop || got.ArmingsNoBridge != veut.ArmingsNoBridge {
		t.Fatalf("couverture armements = {dates %d, attribues %d, sansLacher %d, sansPont %d}, "+
			"attendu {dates %d, attribues %d, sansLacher %d, sansPont %d}",
			got.Armings, got.ArmingsAttributed, got.ArmingsNoDrop, got.ArmingsNoBridge,
			veut.Armings, veut.ArmingsAttributed, veut.ArmingsNoDrop, veut.ArmingsNoBridge)
	}
	if somme := got.ArmingsAttributed + got.ArmingsNoDrop + got.ArmingsNoBridge; somme != got.Armings {
		t.Fatalf("ventilation %d != %d armements dates — un armement perdu ou compte deux fois",
			somme, got.Armings)
	}
}

// assertBombArmEvents vérifie que CHAQUE armement daté est publié (avec ou sans acteur) et que
// la somme des `bomb_arms` vaut le nombre d'armements attribués — le contrôle de cohérence.
func assertBombArmEvents(t *testing.T, evts []BombEvent, armings []BombArming, veut map[string]int) {
	t.Helper()
	publies, avecActeur := 0, 0
	for _, e := range evts {
		if e.Type != BombEventArmed {
			continue
		}
		publies++
		if e.XUID != "" {
			avecActeur++
		}
	}
	if publies != len(armings) {
		t.Fatalf("%d armements publies, %d dates — un armement non resolu doit rester publie",
			publies, len(armings))
	}
	somme := 0
	for _, n := range veut {
		somme += n
	}
	if avecActeur != somme {
		t.Fatalf("%d armements avec acteur, somme des bomb_arms = %d — les deux doivent coincider",
			avecActeur, somme)
	}
}
