package teammates

// composition_presence_test.go — VERROU SEMANTIQUE (ratchet), PAS un TDD.
//
// Ces tests passent des leur ecriture : le moteur respecte deja la regle. Ils existent
// pour qu'il continue de la respecter, parce que la regle n'etait ecrite NULLE PART et
// tenait par accident (ADR 0033).
//
// LA REGLE. Quitter un match n'est pas quitter la session (decision utilisateur du
// 2026-09-09). Un plantage du jeu, un redemarrage du PC, une deconnexion : le coequipier
// reste un membre de la composition pour ce match, et le match reste dans la population.
// L'appartenance ne depend donc JAMAIS de `present_at_completion`, `left_in_progress`,
// `present_at_beginning`, `joined_in_progress` ni `last_leave_time`.
//
// LE SCENARIO EST CELUI QUI A ETE MESURE (session du 27 aout 2026, match `2cf24f30`) :
// Chocoboflor perd son PC en cours de match, un bot `bid(3.0)` prend sa place sur l'equipe,
// il ne revient pas avant la fin. Ce match compte, sous les DEUX valeurs de l'option
// « composition exacte » — et le bot ne casse pas l'exclusivite.
//
// Le second test est le pendant STRUCTUREL : aucun type de la population escouade ne porte
// de champ de presence. Tant que c'est vrai, aucun filtre de presence ne PEUT etre ecrit
// dans cette couche par distraction ; le jour ou quelqu'un ajoute le champ, il tombe ici et
// doit lire l'ADR avant d'aller plus loin. Le pendant SQL est
// `no_presence_filter_test.go`.

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// newCrashedTeammateRepo : composition {AllyA, AllyB} sur UNE session de deux matchs.
// Sur m1, AllyB a quitte en cours de partie (crash) et un bot a pris sa place : il figure
// donc aux participants de l'equipe du main, comme le fait la base, et le bot aussi.
func newCrashedTeammateRepo() *mockSquadRepo {
	t0 := time.Date(2026, 8, 27, 19, 39, 0, 0, time.UTC)
	t1 := time.Date(2026, 8, 27, 19, 55, 0, 0, time.UTC)
	shared := []domain.SquadMatchRow{
		makeSquadRowSess("m1", "Streets", domain.OutcomeWin, "S_27_08", t0),
		makeSquadRowSess("m2", "Bazaar", domain.OutcomeWin, "S_27_08", t1),
	}
	return &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "xa", Gamertag: "AllyA", GamesTogether: 500},
			{XUID: "xb", Gamertag: "AllyB", GamesTogether: 450},
		},
		squadRowsByTeammate: map[string][]domain.SquadMatchRow{
			"xa": shared,
			"xb": shared,
		},
		// m1 : AllyB est aux participants MALGRE son depart, et le bot qui l'a remplace
		// est sur la meme equipe. m2 : partie normale.
		allyRows: []domain.AllyParticipant{
			ally("m1", "px"), ally("m1", "xa"), ally("m1", "xb"), ally("m1", "bid(3.0)"),
			ally("m2", "px"), ally("m2", "xa"), ally("m2", "xb"),
		},
	}
}

// TestPopulation_LeaverStaysInSession : le match du crash compte, sous les DEUX valeurs de
// l'option, sur TOUTES les surfaces.
func TestPopulation_LeaverStaysInSession(t *testing.T) {
	for _, exact := range []bool{false, true} {
		t.Run(map[bool]string{false: "option_off", true: "option_on"}[exact], func(t *testing.T) {
			repo := newCrashedTeammateRepo()
			svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
				newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
			)

			resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
				SelectedGamertags:      []string{"AllyA", "AllyB"},
				FilterExactComposition: exact,
			})
			if err != nil {
				t.Fatalf("GetPage: %v", err)
			}

			maps := map[string]bool{}
			for _, row := range resp.MapBreakdown {
				maps[row.MapUI] = true
			}
			if !maps["Streets"] {
				t.Errorf("le match quitte en cours (m1) a disparu du MapBreakdown : %v", maps)
			}
			if len(resp.MapBreakdown) != 2 {
				t.Errorf("MapBreakdown : want 2 cartes, got %d (%v)", len(resp.MapBreakdown), maps)
			}

			history := map[string]bool{}
			for _, row := range resp.MatchHistory {
				history[row.MatchID] = true
			}
			if !history["m1"] || !history["m2"] || len(resp.MatchHistory) != 2 {
				t.Errorf("MatchHistory : want {m1, m2}, got %v", history)
			}

			counts := map[string]int{}
			for _, s := range resp.CompositionSessions {
				counts[s.Label] = s.MatchCount
			}
			if counts["S_27_08"] != 2 {
				t.Errorf("CompositionSessions[S_27_08].MatchCount : want 2, got %d (%v)",
					counts["S_27_08"], counts)
			}
		})
	}
}

// TestPopulationTypes_CarryNoPresenceField : verrou structurel. Aucun type de la population
// escouade ne porte de champ de presence — donc aucun filtre de presence ne peut etre
// branche ici sans d'abord modifier un contrat et tomber sur ce test.
func TestPopulationTypes_CarryNoPresenceField(t *testing.T) {
	// Ces noms sont ceux des colonnes de `match_participants` qui disent la presence.
	// Les chercher en sous-chaine (insensible a la casse) attrape aussi les variantes
	// Go (`PresentAtCompletion`, `LeftInProgress`, ...).
	forbidden := []string{"presentat", "leftinprogress", "joinedinprogress", "lastleave"}

	for _, typ := range []reflect.Type{
		reflect.TypeOf(domain.SquadMatchRow{}),
		reflect.TypeOf(domain.AllyParticipant{}),
	} {
		for i := range typ.NumField() {
			name := strings.ToLower(typ.Field(i).Name)
			for _, bad := range forbidden {
				if strings.Contains(name, bad) {
					t.Errorf("%s.%s : champ de presence dans un type de population escouade. "+
						"Quitter un match n'est pas quitter la session (ADR 0033) — si ce champ "+
						"doit exister, il ne doit filtrer AUCUNE population.",
						typ.Name(), typ.Field(i).Name)
				}
			}
		}
	}
}
