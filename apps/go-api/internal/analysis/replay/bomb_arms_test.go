package replay

// bomb_arms_test.go — LA JOINTURE DE L'ARMEMENT, éprouvée SANS film et SANS base.
//
// Ce que ces tests vérifient, au-delà des comptes :
//
//	le RECALAGE      un décalage d'horloge non nul est APPLIQUÉ, par la règle PRIMAIRE comme
//	                 par le REPLI ; l'oublier casse les deux
//	                 (TestBombArmsRecalageHorloge — cf. le piège en tête de bomb_arms.go) ;
//	la PRIORITÉ      quand le lâcher et le porteur actif désignent des joueurs DIFFÉRENTS, le
//	                 lâcher gagne ; et il gagne même quand le repli d'un armement ANTÉRIEUR
//	                 convoitait la même période (c'est la structure en deux passes qui le
//	                 garantit, pas l'ordre chronologique) ;
//	le REFUS         deux porteurs couvrant l'instant armé laissent l'armement SANS acteur —
//	                 le repli ne tranche jamais entre deux ;
//	le silence       sans anneau OU sans portage, `bomb_arms` reste absent sur TOUS les
//	                 joueurs, jamais à zéro ;
//	l'acteur absent  un armement que la jointure ne résout pas est PUBLIÉ sans acteur, et sa
//	                 raison est comptée (`ArmingsNoCarrier` / `ArmingsNoBridge` /
//	                 `ArmingsAmbiguous`) ;
//	l'exclusivité    une même période de portage ne pose qu'une bombe ;
//	la cohérence     somme des `bomb_arms` == ArmingsAttributed == ByDrop + ByActiveCarry, et
//	                 la ventilation retombe sur le nombre d'armements datés.

import (
	"fmt"
	"testing"
)

// baArming fabrique un armement daté sur l'horloge du FILM. StartMS/T ne sont pas lus par la
// jointure (elle date sur la FIN de la montée) : ils portent des valeurs plausibles pour que
// l'entrée reste réaliste.
func baArming(timeMS int) BombArming {
	return BombArming{T: timeMS / 100, TimeMS: timeMS,
		StartT: (timeMS - 1200) / 100, StartMS: timeMS - 1200, FuseMS: BombFuseMS}
}

// baPeriodeMort fabrique une période fermée par la MORT du porteur — pas un geste de pose,
// donc hors de la règle primaire, mais candidate au repli : sa fin est datée par le fil des
// morts, donc « il la tenait à cet instant » y reste une mesure.
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

// TestBombArmsJointure : la règle PRIMAIRE (le lâcher), cas par cas.
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
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAttributed: 1, ArmingsByDrop: 1},
		},
		{
			nom:     "deux armements, deux poseurs distincts",
			armings: []BombArming{baArming(100000), baArming(200000)},
			periodes: []HeldObjectPeriod{
				bombPeriode(7, 80000, 100126),
				bombPeriode(9, 180000, 200090),
			},
			veut:    map[string]int{"7": 1, "9": 1},
			veutCov: BombStatsCoverage{Armings: 2, ArmingsAttributed: 2, ArmingsByDrop: 2},
		},
		{
			nom:     "le lacher le PLUS PROCHE gagne, l'autre reste libre",
			armings: []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{
				bombPeriode(7, 80000, 98000),
				bombPeriode(9, 90000, 100100),
			},
			veut:    map[string]int{"9": 1},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAttributed: 1, ArmingsByDrop: 1},
		},
		{
			nom:      "lacher HORS FENETRE et hors couverture : armement sans acteur",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{bombPeriode(7, 80000, 97400)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoCarrier: 1},
		},
		{
			nom:      "restee OUVERTE : ni lacher ni fin mesuree, armement sans acteur",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{baPeriodeOuverte(7, 80000)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoCarrier: 1},
		},
		{
			nom:      "prise APRES l'armement : on n'arme pas ce qu'on n'a pas encore",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{bombPeriode(7, 100500, 101000)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoCarrier: 1},
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
			veutCov: BombStatsCoverage{Armings: 2, ArmingsAttributed: 1, ArmingsByDrop: 1, ArmingsNoCarrier: 1},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			baJouer(t, c.armings, c.periodes, c.veut, c.veutCov)
		})
	}
}

// TestBombArmsRepliPorteurActif : le SECOND recours, arbitré par l'utilisateur le 2026-09-04.
// Il ne s'applique QUE là où la règle primaire est muette, et il refuse de trancher dès que
// deux porteurs couvrent l'instant armé.
func TestBombArmsRepliPorteurActif(t *testing.T) {
	cases := []struct {
		nom      string
		armings  []BombArming
		periodes []HeldObjectPeriod
		veut     map[string]int
		veutCov  BombStatsCoverage
	}{
		{
			// LE TEMOIN DU CORPUS : `35b75a31` @299 176 ms — la periode du porteur couvre
			// l'instant arme et ne se ferme que 4 245 ms APRES l'explosion. Aucun lacher
			// exploitable, mais un seul porteur : le repli le nomme.
			nom:      "le porteur TRAVERSE la pose : le repli le nomme",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{baPeriodeMort(7, 90000, 112000)},
			veut:     map[string]int{"7": 1},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAttributed: 1,
				ArmingsByActiveCarry: 1},
		},
		{
			nom:      "lacher hors fenetre mais la periode COUVRE l'instant : le repli nomme",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{bombPeriode(7, 80000, 102600)},
			veut:     map[string]int{"7": 1},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAttributed: 1,
				ArmingsByActiveCarry: 1},
		},
		{
			nom:     "DEUX porteurs couvrent l'instant : aucun acteur, jamais un arbitrage",
			armings: []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{
				baPeriodeMort(7, 80000, 110000),
				baPeriodeMort(9, 90000, 105000),
			},
			veut:    map[string]int{},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAmbiguous: 1},
		},
		{
			// Un porteur ANONYME reste un porteur : sa presence suffit a rendre l'instant
			// ambigu, meme si l'autre candidat est parfaitement nomme.
			nom:     "un candidat NOMME et un candidat ANONYME : toujours ambigu",
			armings: []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{
				baPeriodeMort(7, 80000, 110000),
				baPeriodeMort(0, 90000, 105000),
			},
			veut:    map[string]int{},
			veutCov: BombStatsCoverage{Armings: 1, ArmingsAmbiguous: 1},
		},
		{
			nom:      "UNIQUE candidat, mais slot NON PONTE : sans acteur, raison distincte",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{baPeriodeMort(0, 80000, 110000)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoBridge: 1},
		},
		{
			// La periode OUVERTE couvrirait TOUT armement posterieur a sa prise : l'admettre
			// ferait degenerer le repli en « le dernier qui a ramasse la bombe ».
			nom:      "une periode OUVERTE ne couvre rien : le repli reste muet",
			armings:  []BombArming{baArming(100000)},
			periodes: []HeldObjectPeriod{baPeriodeOuverte(7, 80000)},
			veut:     map[string]int{},
			veutCov:  BombStatsCoverage{Armings: 1, ArmingsNoCarrier: 1},
		},
		{
			nom:      "exclusivite : une periode ne sert qu'a UN armement, repli compris",
			armings:  []BombArming{baArming(100000), baArming(106000)},
			periodes: []HeldObjectPeriod{baPeriodeMort(7, 90000, 112000)},
			veut:     map[string]int{"7": 1},
			veutCov: BombStatsCoverage{Armings: 2, ArmingsAttributed: 1,
				ArmingsByActiveCarry: 1, ArmingsNoCarrier: 1},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			baJouer(t, c.armings, c.periodes, c.veut, c.veutCov)
		})
	}
}

// TestBombArmsLacherPrimeSurRepli : LE LÂCHER RESTE LA SOURCE PRIMAIRE. Deux épreuves,
// parce qu'il y a deux façons de perdre cette priorité.
func TestBombArmsLacherPrimeSurRepli(t *testing.T) {
	t.Run("deux joueurs DIFFERENTS designes : le lacher gagne", func(t *testing.T) {
		// Le 7 lache la bombe 150 ms apres l'instant arme (le geste de pose) ; le 9, lui, la
		// tenait aussi a cet instant et n'a jamais rien lache (periode fermee par sa mort).
		// Les deux regles ne designent PAS le meme joueur — le lacher doit l'emporter, et le
		// fait date doit dire que c'est LUI qui a nomme.
		baJouer(t, []BombArming{baArming(100000)},
			[]HeldObjectPeriod{
				bombPeriode(7, 95000, 100150),
				baPeriodeMort(9, 80000, 130000),
			},
			map[string]int{"7": 1, "9": 0},
			BombStatsCoverage{Armings: 1, ArmingsAttributed: 1, ArmingsByDrop: 1})
	})

	t.Run("le repli d'un armement ANTERIEUR ne vole pas la periode", func(t *testing.T) {
		// La periode du 9 court de 80 000 a 106 200 : elle COUVRE le premier armement
		// (100 000) et se ferme par un lacher 200 ms apres le second (106 000). Un parcours
		// chronologique unique melant les deux regles donnerait la periode au repli du
		// PREMIER armement, et le second — pourtant nomme par un GESTE — resterait anonyme.
		// La structure en deux passes l'interdit.
		got, evts := BuildBombStats(BombStatsInput{
			ArmingsRead: true, Armings: []BombArming{baArming(100000), baArming(106000)},
			CarryRead: true, Carry: bombCarryDe(bombPeriode(9, 80000, 106200)),
		})
		assertBombInt(t, got, "9", func(p BombPlayerStats) *int { return p.Arms }, 1, "arms")
		assertBombArmCoverage(t, got.Coverage, BombStatsCoverage{
			Armings: 2, ArmingsAttributed: 1, ArmingsByDrop: 1, ArmingsNoCarrier: 1})
		assertBombEvents(t, evts, []BombEvent{
			{Type: BombEventArmed, TimeMS: 100000},
			{Type: BombEventArmed, TimeMS: 106000, XUID: "9",
				ActorSource: BombActorSourceDrop},
		})
	})
}

// TestBombArmsRecalageHorloge est LE test du piège : les armements sont datés sur l'horloge du
// FILM, les périodes de portage sur celle du MATCH. Ici le décalage vaut 40 000 ms — très
// au-delà de la fenêtre de jointure — donc supprimer le terme `FilmToMatchOffsetMS` fait
// tomber l'attribution à zéro et rougit ce test.
//
// La valeur est volontairement énorme (les films témoins mesurent 16-81 ms) : un décalage de
// l'ordre du bruit ne prouverait rien, un test qui passe encore sans le recalage ne serait pas
// un test. LES DEUX RÈGLES sont éprouvées : le repli traverse les mêmes deux horloges que le
// lâcher, et un recalage oublié dans sa seule branche serait tout aussi invisible au gate réel.
func TestBombArmsRecalageHorloge(t *testing.T) {
	const offset = 40000

	t.Run("regle primaire : le lacher", func(t *testing.T) {
		// L'armement est daté 100 000 ms sur l'horloge du FILM, soit 140 000 ms sur celle du
		// MATCH — l'horloge des périodes de portage.
		in := BombStatsInput{
			ArmingsRead: true, Armings: []BombArming{baArming(100000)},
			CarryRead: true, Carry: bombCarryDe(bombPeriode(7, 120000, 140126)),
			FilmToMatchOffsetMS: offset,
		}
		got, evts := BuildBombStats(in)
		assertBombInt(t, got, "7", func(p BombPlayerStats) *int { return p.Arms }, 1, "arms")
		assertBombArmCoverage(t, got.Coverage, BombStatsCoverage{
			Armings: 1, ArmingsAttributed: 1, ArmingsByDrop: 1})
		// Le fait daté reste publié sur l'horloge du FILM : c'est celle des explosions, et la
		// persistance les pose sur le même axe.
		assertBombEvents(t, evts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000,
			XUID: "7", ActorSource: BombActorSourceDrop}})

		// Contre-épreuve : le MÊME jeu d'entrées sans le recalage ne nomme personne — et
		// l'armement reste publié, sans acteur. La période commence à 120 000, donc elle ne
		// couvre pas non plus 100 000 : le repli reste muet lui aussi.
		in.FilmToMatchOffsetMS = 0
		sans, sansEvts := BuildBombStats(in)
		assertBombArmCoverage(t, sans.Coverage, BombStatsCoverage{
			Armings: 1, ArmingsNoCarrier: 1})
		assertBombEvents(t, sansEvts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000}})
	})

	t.Run("repli : le porteur actif", func(t *testing.T) {
		// Aucun lâcher ici (période fermée par la mort) : seul le repli peut nommer, et il ne
		// le peut QUE si l'instant du film est recalé sur l'horloge du match.
		in := BombStatsInput{
			ArmingsRead: true, Armings: []BombArming{baArming(100000)},
			CarryRead: true, Carry: bombCarryDe(baPeriodeMort(8, 120000, 160000)),
			FilmToMatchOffsetMS: offset,
		}
		got, evts := BuildBombStats(in)
		assertBombInt(t, got, "8", func(p BombPlayerStats) *int { return p.Arms }, 1, "arms")
		assertBombArmCoverage(t, got.Coverage, BombStatsCoverage{
			Armings: 1, ArmingsAttributed: 1, ArmingsByActiveCarry: 1})
		assertBombEvents(t, evts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000,
			XUID: "8", ActorSource: BombActorSourceActiveCarry}})

		in.FilmToMatchOffsetMS = 0
		sans, sansEvts := BuildBombStats(in)
		assertBombArmCoverage(t, sans.Coverage, BombStatsCoverage{
			Armings: 1, ArmingsNoCarrier: 1})
		assertBombEvents(t, sansEvts, []BombEvent{{Type: BombEventArmed, TimeMS: 100000}})
	})
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
		assertBombArmCoverage(t, got.Coverage, BombStatsCoverage{
			Armings: 1, ArmingsNoCarrier: 1})
	})
}

// baJouer déroule un cas de jointure et applique les trois vérifications communes : les
// comptes par joueur, la ventilation de couverture, et le contrôle de cohérence des faits
// datés. Un seul endroit où l'on décide ce que « vérifier un cas » veut dire.
func baJouer(t *testing.T, armings []BombArming, periodes []HeldObjectPeriod,
	veut map[string]int, veutCov BombStatsCoverage,
) {
	t.Helper()
	got, evts := BuildBombStats(BombStatsInput{
		ArmingsRead: true, Armings: armings,
		CarryRead: true, Carry: bombCarryDe(periodes...),
	})
	for xuid, n := range veut {
		assertBombInt(t, got, xuid, func(p BombPlayerStats) *int { return p.Arms }, n, "arms")
	}
	assertBombArmCoverage(t, got.Coverage, veutCov)
	assertBombArmEvents(t, evts, armings, veut)
}

// assertBombArmCoverage compare la ventilation de la jointure ET vérifie les DEUX INVARIANTS
// publiés : attribués == par lâcher + par repli, et attribués + les trois raisons d'absence ==
// armements datés.
func assertBombArmCoverage(t *testing.T, got, veut BombStatsCoverage) {
	t.Helper()
	if got.Armings != veut.Armings || got.ArmingsAttributed != veut.ArmingsAttributed ||
		got.ArmingsByDrop != veut.ArmingsByDrop ||
		got.ArmingsByActiveCarry != veut.ArmingsByActiveCarry ||
		got.ArmingsNoCarrier != veut.ArmingsNoCarrier ||
		got.ArmingsNoBridge != veut.ArmingsNoBridge ||
		got.ArmingsAmbiguous != veut.ArmingsAmbiguous {
		t.Fatalf("couverture armements = %s, attendu %s",
			baFormatCouverture(got), baFormatCouverture(veut))
	}
	if got.ArmingsByDrop+got.ArmingsByActiveCarry != got.ArmingsAttributed {
		t.Fatalf("ventilation par regle %d + %d != %d attribues — une regle non comptee",
			got.ArmingsByDrop, got.ArmingsByActiveCarry, got.ArmingsAttributed)
	}
	somme := got.ArmingsAttributed + got.ArmingsNoCarrier + got.ArmingsNoBridge +
		got.ArmingsAmbiguous
	if somme != got.Armings {
		t.Fatalf("ventilation %d != %d armements dates — un armement perdu ou compte deux fois",
			somme, got.Armings)
	}
}

// baFormatCouverture rend la ventilation en une ligne lisible — l'échec d'un test doit dire ce
// qui a bougé, pas obliger à compter des champs dans un dump de structure.
func baFormatCouverture(c BombStatsCoverage) string {
	return fmt.Sprintf("{dates %d, attribues %d (lacher %d, repli %d), sansPorteur %d, "+
		"sansPont %d, ambigus %d}",
		c.Armings, c.ArmingsAttributed, c.ArmingsByDrop, c.ArmingsByActiveCarry,
		c.ArmingsNoCarrier, c.ArmingsNoBridge, c.ArmingsAmbiguous)
}

// assertBombArmEvents vérifie que CHAQUE armement daté est publié (avec ou sans acteur), que la
// somme des `bomb_arms` vaut le nombre d'armements attribués, et que TOUT fait avec acteur
// porte la règle qui l'a nommé — un acteur sans règle serait un acteur sans provenance.
func assertBombArmEvents(t *testing.T, evts []BombEvent, armings []BombArming, veut map[string]int) {
	t.Helper()
	publies, avecActeur := 0, 0
	for _, e := range evts {
		if e.Type != BombEventArmed {
			continue
		}
		publies++
		if e.XUID == "" {
			if e.ActorSource != "" {
				t.Fatalf("fait sans acteur mais source %q — incoherent", e.ActorSource)
			}
			continue
		}
		avecActeur++
		if e.ActorSource != BombActorSourceDrop && e.ActorSource != BombActorSourceActiveCarry {
			t.Fatalf("acteur %s nomme sans regle publiee (ActorSource %q)", e.XUID, e.ActorSource)
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
