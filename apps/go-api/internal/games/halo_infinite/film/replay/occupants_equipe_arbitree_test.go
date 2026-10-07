package replay

// occupants_equipe_arbitree_test.go — UN BOT DONT L'ENTITE ET LA DECLARATION DISENT DEUX EQUIPES : LA
// FEUILLE DE MATCH TRANCHE (occupants_equipe_arbitree.go, decision utilisateur du 2026-10-07).
//
//	A-FEUILLE    entite 1, declaration 0, feuille 0 : l'equipe publiee est celle de la feuille, la
//	             contradiction est comptee et gardee avec l'arbitrage ;
//	A-INCONNU    la feuille ne porte pas ce bot (autre bid, bid vide, equipe -1) : l'entite est publiee,
//	             la contradiction est comptee, sans arbitrage ;
//	A-ACCORD     entite et declaration d'accord, ou bot sans equipe declaree : la feuille ne change
//	             rien, meme quand elle dit une autre equipe ;
//	A-JOURNAL    un arbitrage s'ecrit en avertissement, un bot inconnu de la feuille en ERREUR, et
//	             au second compteur ; toute contradiction au premier ;
//	A-ASSEMBLAGE `BuildFromPositions` pose `Options.ScoreboardTeams` dans la liaison : Byrontron (entite
//	             0, declaration 1) est publie dans l'equipe de la feuille, et dans celle de son entite
//	             sans feuille ;
//	A-SYNC       le document interne de `PortagesAuSync` recoit la feuille du collecteur.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// botContredit : un bot d'index 8 et de `bid(7.0)` dont l'entite (equipe 1) couvre sa declaration,
// qui dit l'equipe `declaree`.
func botContredit(declaree *int) (RosterEntry, entreesDesOccupants) {
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 8, Team: 1, FirstKF: 1,
		LastKF: 1, Seen: 1})
	bot, id := botDeclare("343 PardonMy [bot]", 45, 60, declaree)
	bot.Bid, id.BotID = "bid(7.0)", 7
	in := entreesDeTest(scan)
	in.bots = []BotIdentity{id}
	return bot, in
}

func TestContradictionTrancheeParLaFeuille(t *testing.T) { // A-FEUILLE
	bot, in := botContredit(equipePtr(0))
	in.base = map[string]int{"bid(7.0)": 0, "123": 1}
	occ := lierLesOccupants([]RosterEntry{bot}, nil, in)
	if eq := occ.parEntree[0].equipe; eq == nil || *eq != 0 {
		t.Fatalf("equipe %v : attendu 0, celle de la feuille de match", deref(eq))
	}
	if occ.equipesContreDeclaration != 1 || len(occ.contradictions) != 1 {
		t.Fatalf("contradictions %d (%+v) : attendu 1", occ.equipesContreDeclaration, occ.contradictions)
	}
	c := occ.contradictions[0]
	if c.base == nil || *c.base != 0 || c.entite != 1 || c.declaree != 0 || c.bid != "bid(7.0)" {
		t.Fatalf("contradiction %+v : attendu entite 1, declaration 0, feuille 0, bid(7.0)", c)
	}
}

func TestBotInconnuDeLaFeuilleGardeSonEntite(t *testing.T) { // A-INCONNU
	for nom, cas := range map[string]struct {
		bid  string
		base map[string]int
	}{
		"autre bid":        {"bid(7.0)", map[string]int{"bid(9.0)": 0}},
		"sans feuille":     {"bid(7.0)", nil},
		"bid vide":         {"", map[string]int{"": 0}},
		"equipe inconnue":  {"bid(7.0)", map[string]int{"bid(7.0)": -1}},
		"feuille sans bot": {"bid(7.0)", map[string]int{"2533274800000000": 0}},
	} {
		bot, in := botContredit(equipePtr(0))
		bot.Bid, in.base = cas.bid, cas.base
		occ := lierLesOccupants([]RosterEntry{bot}, nil, in)
		if eq := occ.parEntree[0].equipe; eq == nil || *eq != 1 {
			t.Errorf("%s : equipe %v, attendu 1 (l'entite, la feuille ne connait pas ce bot)", nom, deref(eq))
		}
		if occ.equipesContreDeclaration != 1 || len(occ.contradictions) != 1 || occ.contradictions[0].base != nil {
			t.Errorf("%s : contradictions %d %+v, attendu 1 sans arbitrage", nom, occ.equipesContreDeclaration,
				occ.contradictions)
		}
	}
}

func TestSansContradictionLaFeuilleNeChangeRien(t *testing.T) { // A-ACCORD
	for nom, declaree := range map[string]*int{"accord": equipePtr(1), "sans declaration": nil} {
		bot, in := botContredit(declaree)
		in.base = map[string]int{"bid(7.0)": 0}
		occ := lierLesOccupants([]RosterEntry{bot}, nil, in)
		if eq := occ.parEntree[0].equipe; eq == nil || *eq != 1 {
			t.Errorf("%s : equipe %v, attendu 1 (l'entite : le film ne se contredit pas)", nom, deref(eq))
		}
		if occ.equipesContreDeclaration != 0 || len(occ.contradictions) != 0 {
			t.Errorf("%s : contradictions %d %+v, attendu 0", nom, occ.equipesContreDeclaration, occ.contradictions)
		}
	}
}

func TestJournalDesContradictionsDEquipe(t *testing.T) { // A-JOURNAL
	const extrait = "disent deux equipes"
	contre, sansArbitre := compteurExpvar(t, metriqueEquipesContreDeclaration), compteurExpvar(t, metriqueEquipesSansArbitre)
	tranchee := occupants{equipesContreDeclaration: 1,
		contradictions: []contradictionDEquipe{{nom: "b", bid: "bid(7.0)", entite: 1, base: equipePtr(0)}}}
	journal := journalDe(t, func() { journaliserLesEquipesDeclarees(context.Background(), "m", tranchee) })
	if !aUnEnregistrement(journal, "WARN", extrait) || aUnEnregistrement(journal, "ERROR", extrait) {
		t.Errorf("journal %v : un arbitrage par la feuille s'ecrit en avertissement, pas en ERREUR", journal)
	}
	if d1, d2 := compteurExpvar(t, metriqueEquipesContreDeclaration)-contre,
		compteurExpvar(t, metriqueEquipesSansArbitre)-sansArbitre; d1 != 1 || d2 != 0 {
		t.Errorf("compteurs +%d contradictions, +%d sans arbitre : attendu +1 et +0", d1, d2)
	}
	inconnu := occupants{equipesContreDeclaration: 1,
		contradictions: []contradictionDEquipe{{nom: "b", bid: "bid(7.0)", entite: 1}}}
	journal = journalDe(t, func() { journaliserLesEquipesDeclarees(context.Background(), "m", inconnu) })
	if !aUnEnregistrement(journal, "ERROR", extrait) {
		t.Errorf("journal %v : un bot que la feuille ne connait pas s'ecrit en ERREUR", journal)
	}
	if d1, d2 := compteurExpvar(t, metriqueEquipesContreDeclaration)-contre,
		compteurExpvar(t, metriqueEquipesSansArbitre)-sansArbitre; d1 != 2 || d2 != 1 {
		t.Errorf("compteurs +%d contradictions, +%d sans arbitre : attendu +2 et +1", d1, d2)
	}
}

func TestAssemblageArbitreLeBotParLaFeuille(t *testing.T) { // A-ASSEMBLAGE
	equipeDeByrontron := func(feuille map[string]int) *int {
		in := entreeDonosHumain(10, 15)
		in.Bots[2].Team = equipePtr(1) // son entite (2504) dit 0
		opt := Options{FrameIntervalMS: 100, BipedCreations: in.BipedCreations, PlayerIndices: in.PlayerIndices,
			Bots: in.Bots, PlayerEntities: in.Entities, ScoreboardTeams: feuille}
		doc := BuildFromPositions(context.Background(), "temoin", "halo_infinite", in.Positions, nil, opt)
		for _, e := range doc.Roster {
			if e.Bot && e.Bid == "bid(23.0)" {
				return e.Team
			}
		}
		t.Fatalf("Byrontron absent du roster publie : %+v", doc.Roster)
		return nil
	}
	if eq := equipeDeByrontron(map[string]int{"bid(23.0)": 1}); eq == nil || *eq != 1 {
		t.Errorf("equipe de Byrontron %v avec la feuille : attendu 1, celle de la feuille", deref(eq))
	}
	if eq := equipeDeByrontron(nil); eq == nil || *eq != 0 {
		t.Errorf("equipe de Byrontron %v sans feuille : attendu 0, celle de son entite", deref(eq))
	}
}

// TestPortagesAuSyncPorteLaFeuilleALaLiaison (A-SYNC) : le document interne des porteurs au sync
// recoit la feuille de match comme la cuisson, pour trancher les memes bots.
func TestPortagesAuSyncPorteLaFeuilleALaLiaison(t *testing.T) {
	e := EntreePorteursAuSync{Equipes: map[string]int{"bid(7.0)": 1}}
	if got := e.optionsDuRegistre(nil).ScoreboardTeams; got["bid(7.0)"] != 1 || len(got) != 1 {
		t.Fatalf("ScoreboardTeams %v : attendu la feuille du collecteur", got)
	}
}
