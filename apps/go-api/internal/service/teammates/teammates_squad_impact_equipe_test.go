package teammates

// teammates_squad_impact_equipe_test.go — LE TABLEAU D'IMPACT PAR SESSION NE SE PUBLIE PAS SANS
// L'ÉQUIPE ALLIÉE LUE.
//
// Sans l'équipe alliée du joueur principal (Q32b), « a joué » retombe sur « a un rôle » : le net par
// match du tableau est faux, sans signal. Le bloc est alors absent, avec sa raison au journal.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestBuildSquadImpact_SansEquipeAllieeLue_PasDeBloc(t *testing.T) {
	cas := []struct {
		nom      string
		mainXUID string
		allies   func(repo *mockSquadRepo) []domain.AllyParticipant
	}{
		{"équipe alliée vide", "x_main", func(*mockSquadRepo) []domain.AllyParticipant { return nil }},
		{"sans xuid du principal", "", func(r *mockSquadRepo) []domain.AllyParticipant { return r.allyRows }},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			repo, all := soireesFixture()
			svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
			var sessions *domain.SquadImpactSessions
			logs := withCapturedLogs(t, func() {
				sessions = svc.buildSquadImpact(context.Background(), impactEscouade{
					rows: all, timeline: all, evenings: soireesDImpact(all, all, true),
					mainXUID: c.mainXUID, selected: []string{"Ally"},
					teammates: []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}},
					allies:    c.allies(repo),
				})
			})
			if sessions != nil {
				t.Fatalf("bloc nul attendu sans équipe alliée lue, obtenu %d sessions", len(sessions.Sessions))
			}
			if !strings.Contains(logs, "teammates_squad_impact_sessions_absent") || !strings.Contains(logs, `"reason":"no_ally_team"`) {
				t.Errorf("journal attendu avec la raison no_ally_team, obtenu :\n%s", logs)
			}
		})
	}
}
