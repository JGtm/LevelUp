package replay

// bomb_stats_test.go — LES QUATRE STATISTIQUES D'ASSAUT, éprouvées SANS film et SANS base.
//
// Chaque cas construit ses entrées à la main (périodes de portage, kills appariés, actions
// d'objectif identifiées) : c'est exactement ce que `BuildBombStats` reçoit en production,
// mais fabriqué, donc reproductible et instantané. Aucun `ASSAUT_CACHE`, aucun décodage.
//
// Ce que ces tests VÉRIFIENT en propre, au-delà des comptes :
//
//	le silence      une source non lue laisse son champ `nil` sur TOUS les joueurs, jamais 0 ;
//	le zéro mesuré  une source lue qui ne trouve rien pour un joueur nommé ailleurs rend 0 ;
//	la mort         une période fermée par la mort compte dans le temps de portage ;
//	l'ouverture     une période restée ouverte compte en ramassage, PAS en temps.

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// bombDeton fabrique une action d'objectif identifiée « explosion de bombe ».
func bombDeton(timeMS int, xuid string) objectiveevents.IdentifiedEvent {
	return objectiveevents.IdentifiedEvent{
		NamedEvent: objectiveevents.NamedEvent{
			TimeMS: timeMS, Slot: 10, Stat: objectiveevents.StatBombDetonations,
		},
		XUID: xuid,
	}
}

// bombPeriode fabrique une période de portage FERMÉE par un lâcher.
func bombPeriode(xuid uint64, debut, fin int) HeldObjectPeriod {
	return HeldObjectPeriod{Slot: 1, XUID: xuid, DebutMS: debut, FinMS: fin}
}

// bombCarryDe assemble une chronologie à partir de périodes déjà décrites, en recalculant
// `CarryMSByXUID` selon la MÊME règle que `BuildHeldObjectCarry` (périodes fermées et
// pontées seulement) — les tests ne redéfinissent pas la règle, ils la rejouent.
func bombCarryDe(periods ...HeldObjectPeriod) HeldObjectCarry {
	out := HeldObjectCarry{Periods: periods, CarryMSByXUID: map[uint64]int{}}
	for _, p := range periods {
		if p.XUID != 0 && !p.Ouverte {
			out.CarryMSByXUID[p.XUID] += p.FinMS - p.DebutMS
		}
	}
	return out
}

// TestBombStatsDetonations : le compte par joueur et les faits datés viennent des actions
// identifiées, et rien d'autre du flux n'y entre.
func TestBombStatsDetonations(t *testing.T) {
	autre := objectiveevents.IdentifiedEvent{
		NamedEvent: objectiveevents.NamedEvent{
			TimeMS: 1500, Slot: 12, Stat: objectiveevents.StatKills,
		},
		XUID: "7",
	}
	cases := []struct {
		nom       string
		read      bool
		objs      []objectiveevents.IdentifiedEvent
		veutCount map[string]int // nil = champ absent attendu
		veutEvts  []BombEvent
	}{
		{
			nom: "source non lue : aucun compte, aucun evenement",
			objs: []objectiveevents.IdentifiedEvent{
				bombDeton(1000, "7"),
			},
		},
		{
			nom: "lue et vide : zero evenement, mais le champ existe", read: true,
			veutCount: map[string]int{},
		},
		{
			nom: "deux joueurs, une autre stat ignoree", read: true,
			objs: []objectiveevents.IdentifiedEvent{
				bombDeton(9000, "8"), autre, bombDeton(1000, "7"), bombDeton(4000, "7"),
			},
			veutCount: map[string]int{"7": 2, "8": 1},
			veutEvts: []BombEvent{
				{Type: BombEventDetonated, TimeMS: 1000, XUID: "7"},
				{Type: BombEventDetonated, TimeMS: 4000, XUID: "7"},
				{Type: BombEventDetonated, TimeMS: 9000, XUID: "8"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			got, evts := BuildBombStats(BombStatsInput{DetonationsRead: c.read, Objectives: c.objs})
			assertBombEvents(t, evts, c.veutEvts)
			if !c.read {
				assertBombAbsent(t, got, func(p BombPlayerStats) bool { return p.Detonations != nil },
					"detonations")
				return
			}
			for xuid, veut := range c.veutCount {
				assertBombInt(t, got, xuid, func(p BombPlayerStats) *int { return p.Detonations },
					veut, "detonations")
			}
			if got.Coverage.Detonations != len(c.veutEvts) {
				t.Fatalf("couverture detonations = %d, attendu %d",
					got.Coverage.Detonations, len(c.veutEvts))
			}
		})
	}
}

// TestBombStatsGrabs : un ramassage = une période ouverte par une prise, y compris celles que
// la mort ferme et celles qui restent ouvertes ; une période non pontée ne crédite personne.
func TestBombStatsGrabs(t *testing.T) {
	morte := bombPeriode(7, 5000, 9000)
	morte.FinParMort = true
	ouverte := HeldObjectPeriod{Slot: 3, XUID: 8, DebutMS: 20000,
		FinMS: HeldObjectOpenEndMS, Ouverte: true}
	sansPont := bombPeriode(0, 12000, 13000)

	cases := []struct {
		nom      string
		read     bool
		carry    HeldObjectCarry
		veut     map[string]int
		veutCov  BombStatsCoverage
		absentSi bool
	}{
		{
			nom:      "canal non balaye : champ absent",
			carry:    bombCarryDe(bombPeriode(7, 1000, 2000)),
			absentSi: true,
		},
		{
			nom: "prises fermees, par mort, ouverte, et une sans pont", read: true,
			carry: bombCarryDe(bombPeriode(7, 1000, 2000), morte, sansPont, ouverte),
			veut:  map[string]int{"7": 2, "8": 1},
			veutCov: BombStatsCoverage{
				Periods: 4, PeriodsNoBridge: 1, PeriodsOpen: 1, PeriodsByDeath: 1,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			got, _ := BuildBombStats(BombStatsInput{CarryRead: c.read, Carry: c.carry})
			if c.absentSi {
				assertBombAbsent(t, got, func(p BombPlayerStats) bool { return p.Grabs != nil },
					"grabs")
				return
			}
			for xuid, veut := range c.veut {
				assertBombInt(t, got, xuid, func(p BombPlayerStats) *int { return p.Grabs },
					veut, "grabs")
			}
			assertBombPeriodCoverage(t, got.Coverage, c.veutCov)
		})
	}
}

// TestBombStatsCarrierSeconds : le temps est la somme des périodes FERMÉES — celles que la
// mort ferme comprises — et une période restée ouverte n'y entre pas.
func TestBombStatsCarrierSeconds(t *testing.T) {
	parMort := bombPeriode(7, 10000, 13500)
	parMort.FinParMort = true
	ouverte := HeldObjectPeriod{Slot: 3, XUID: 9, DebutMS: 20000,
		FinMS: HeldObjectOpenEndMS, Ouverte: true}

	cases := []struct {
		nom      string
		read     bool
		carry    HeldObjectCarry
		veut     map[string]float64
		absentSi bool
	}{
		{
			nom:      "canal non balaye : champ absent",
			carry:    bombCarryDe(bombPeriode(7, 1000, 3000)),
			absentSi: true,
		},
		{
			nom: "lacher plus mort cumules, ouverte exclue", read: true,
			carry: bombCarryDe(bombPeriode(7, 1000, 3000), parMort, ouverte),
			// 2,0 s (lâcher) + 3,5 s (mort) pour 7 ; 9 est nommé mais n'a que de l'ouvert :
			// zéro MESURÉ, pas une absence.
			veut: map[string]float64{"7": 5.5, "9": 0},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			got, _ := BuildBombStats(BombStatsInput{CarryRead: c.read, Carry: c.carry})
			if c.absentSi {
				assertBombAbsent(t, got,
					func(p BombPlayerStats) bool { return p.TimeAsCarrierSeconds != nil },
					"time_as_bomb_carrier_seconds")
				return
			}
			for xuid, veut := range c.veut {
				p := bombRowOf(t, got, xuid)
				if p.TimeAsCarrierSeconds == nil {
					t.Fatalf("xuid %s : time_as_bomb_carrier_seconds absent, attendu %.3f", xuid, veut)
				}
				if *p.TimeAsCarrierSeconds != veut {
					t.Fatalf("xuid %s : time_as_bomb_carrier_seconds = %.3f, attendu %.3f",
						xuid, *p.TimeAsCarrierSeconds, veut)
				}
			}
		})
	}
}

// TestBombStatsCarriersKilled : le tueur d'une victime EN PORTAGE à l'instant du kill est
// crédité ; le kill hors période, le suicide et le tueur non nommé ne créditent personne.
func TestBombStatsCarriersKilled(t *testing.T) {
	parMort := bombPeriode(7, 10000, 13500)
	parMort.FinParMort = true
	carry := bombCarryDe(bombPeriode(7, 1000, 3000), parMort, bombPeriode(0, 40000, 41000))

	cases := []struct {
		nom       string
		carryRead bool
		killsRead bool
		kills     []KillRef
		veut      map[string]int
		absentSi  bool
	}{
		{
			nom: "kills non lus : champ absent", carryRead: true,
			kills:    []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: 13500}},
			absentSi: true,
		},
		{
			nom: "portage non lu : champ absent meme si les kills le sont", killsRead: true,
			kills:    []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: 13500}},
			absentSi: true,
		},
		{
			nom:       "kill pendant le portage, kill a la fermeture par mort",
			carryRead: true, killsRead: true,
			kills: []KillRef{
				{KillerXUID: 5, VictimXUID: 7, TimeMS: 2000},  // dans la 1re période
				{KillerXUID: 5, VictimXUID: 7, TimeMS: 13500}, // exactement à la mort
				{KillerXUID: 6, VictimXUID: 7, TimeMS: 13600}, // dans la tolérance de fin
			},
			veut: map[string]int{"5": 2, "6": 1, "7": 0},
		},
		{
			nom:       "hors periode, avant la prise, suicide, tueur non nomme",
			carryRead: true, killsRead: true,
			kills: []KillRef{
				{KillerXUID: 5, VictimXUID: 7, TimeMS: 5000},  // entre deux périodes
				{KillerXUID: 5, VictimXUID: 7, TimeMS: 900},   // avant la prise
				{KillerXUID: 7, VictimXUID: 7, TimeMS: 2000},  // suicide du porteur
				{KillerXUID: 0, VictimXUID: 7, TimeMS: 2000},  // tueur non nommé
				{KillerXUID: 5, VictimXUID: 4, TimeMS: 13500}, // victime non porteuse
			},
			veut: map[string]int{"7": 0},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			got, _ := BuildBombStats(BombStatsInput{
				CarryRead: c.carryRead, Carry: carry, KillsRead: c.killsRead, Kills: c.kills,
			})
			if c.absentSi {
				assertBombAbsent(t, got,
					func(p BombPlayerStats) bool { return p.CarriersKilled != nil }, "carriers_killed")
				return
			}
			for xuid, veut := range c.veut {
				assertBombInt(t, got, xuid, func(p BombPlayerStats) *int { return p.CarriersKilled },
					veut, "carriers_killed")
			}
			if got.Coverage.Kills != len(c.kills) {
				t.Fatalf("couverture kills = %d, attendu %d", got.Coverage.Kills, len(c.kills))
			}
		})
	}
}

// TestBombStatsAucuneSourceLue : sans aucun témoin, il ne sort AUCUNE ligne — pas une ligne de
// zéros. C'est la garde centrale de la règle « absent n'est pas zéro ».
func TestBombStatsAucuneSourceLue(t *testing.T) {
	got, evts := BuildBombStats(BombStatsInput{
		Objectives: []objectiveevents.IdentifiedEvent{bombDeton(1000, "7")},
		Carry:      bombCarryDe(bombPeriode(7, 1000, 3000)),
		Kills:      []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: 2000}},
	})
	if len(got.Players) != 0 {
		t.Fatalf("aucune source lue : %d joueurs publies, attendu 0", len(got.Players))
	}
	if len(evts) != 0 {
		t.Fatalf("aucune source lue : %d evenements publies, attendu 0", len(evts))
	}
	if got.Coverage.DetonationsRead || got.Coverage.CarryRead || got.Coverage.KillsRead {
		t.Fatalf("couverture : temoins de lecture inattendus %+v", got.Coverage)
	}
}

// bombRowOf rend la ligne d'un joueur, ou échoue si elle manque.
func bombRowOf(t *testing.T, got BombMatchStats, xuid string) BombPlayerStats {
	t.Helper()
	for _, p := range got.Players {
		if p.XUID == xuid {
			return p
		}
	}
	t.Fatalf("xuid %s absent des %d lignes publiees", xuid, len(got.Players))
	return BombPlayerStats{}
}

// assertBombInt vérifie un champ entier MESURÉ (présent, et à la bonne valeur).
func assertBombInt(t *testing.T, got BombMatchStats, xuid string,
	champ func(BombPlayerStats) *int, veut int, nom string) {
	t.Helper()
	p := bombRowOf(t, got, xuid)
	v := champ(p)
	if v == nil {
		t.Fatalf("xuid %s : %s absent, attendu %d", xuid, nom, veut)
	}
	if *v != veut {
		t.Fatalf("xuid %s : %s = %d, attendu %d", xuid, nom, *v, veut)
	}
}

// assertBombAbsent vérifie qu'AUCUNE ligne ne porte le champ : une source non lue se tait
// partout, elle ne rend pas des zéros.
func assertBombAbsent(t *testing.T, got BombMatchStats,
	present func(BombPlayerStats) bool, nom string) {
	t.Helper()
	for _, p := range got.Players {
		if present(p) {
			t.Fatalf("xuid %s : %s present alors que la source n'est pas lue", p.XUID, nom)
		}
	}
}

// assertBombEvents compare les faits datés, ordre compris.
func assertBombEvents(t *testing.T, got, veut []BombEvent) {
	t.Helper()
	if len(got) != len(veut) {
		t.Fatalf("evenements = %d (%+v), attendu %d", len(got), got, len(veut))
	}
	for i := range got {
		if got[i] != veut[i] {
			t.Fatalf("evenement %d = %+v, attendu %+v", i, got[i], veut[i])
		}
	}
}

// assertBombPeriodCoverage compare la ventilation des périodes.
func assertBombPeriodCoverage(t *testing.T, got, veut BombStatsCoverage) {
	t.Helper()
	if got.Periods != veut.Periods || got.PeriodsNoBridge != veut.PeriodsNoBridge ||
		got.PeriodsOpen != veut.PeriodsOpen || got.PeriodsByDeath != veut.PeriodsByDeath {
		t.Fatalf("couverture periodes = {%d, sansPont %d, ouvertes %d, parMort %d}, attendu "+
			"{%d, sansPont %d, ouvertes %d, parMort %d}",
			got.Periods, got.PeriodsNoBridge, got.PeriodsOpen, got.PeriodsByDeath,
			veut.Periods, veut.PeriodsNoBridge, veut.PeriodsOpen, veut.PeriodsByDeath)
	}
}
