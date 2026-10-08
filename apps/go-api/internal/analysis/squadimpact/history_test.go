package squadimpact

import (
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func at(day, hour, minute int) time.Time {
	return time.Date(2026, 9, day, hour, minute, 0, 0, time.UTC)
}

// soiree — n matchs de la session label, à partir du jour et de l'heure donnés.
func soiree(label string, day, hour, n int) []EveningMatch {
	out := make([]EveningMatch, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, EveningMatch{
			MatchID: fmt.Sprintf("%s-%d", label, i), SessionLabel: label,
			StartTime: at(day, hour, 10*i), Won: i%2 == 0,
		})
	}
	return out
}

func labels(evenings []Evening) []string {
	out := make([]string, 0, len(evenings))
	for _, e := range evenings {
		out = append(out, e.Label)
	}
	return out
}

// TestSelectEvenings_DeuxSessionsDuMemeJourRestentDeux : l'app découpe le 7 septembre en deux
// sessions (19 h 26 et 20 h 24) ; elles restent deux soirées, jamais fusionnées par date.
func TestSelectEvenings_DeuxSessionsDuMemeJourRestentDeux(t *testing.T) {
	current := soiree("S3", 8, 19, 3)
	timeline := append(append(soiree("S1", 7, 19, 4), soiree("S2", 7, 20, 1)...), current...)
	got := SelectEvenings(current, timeline)
	if fmt.Sprint(labels(got)) != "[S1 S2 S3]" {
		t.Fatalf("soirées %v, attendu [S1 S2 S3]", labels(got))
	}
	if len(got[0].Matches) != 4 || len(got[1].Matches) != 1 {
		t.Errorf("matchs par soirée %d / %d, attendu 4 / 1", len(got[0].Matches), len(got[1].Matches))
	}
}

// TestSelectEvenings_SoireeAfficheeEtPrecedentes : la soirée affichée garde ses seuls matchs
// affichés ; une soirée jouée APRÈS elle n'est pas « précédente » ; au plus onze soirées.
func TestSelectEvenings_SoireeAfficheeEtPrecedentes(t *testing.T) {
	var timeline []EveningMatch
	for d := 1; d <= 14; d++ {
		timeline = append(timeline, soiree(fmt.Sprintf("D%02d", d), d, 19, 2)...)
	}
	// Population affichée : un seul match de la soirée du 13 (un filtre l'a réduite).
	current := []EveningMatch{timeline[24]}
	got := SelectEvenings(current, timeline)
	if len(got) != domain.SquadImpactHistoryMaxEvenings {
		t.Fatalf("%d soirées, attendu %d", len(got), domain.SquadImpactHistoryMaxEvenings)
	}
	last := got[len(got)-1]
	if last.Label != "D13" || len(last.Matches) != 1 {
		t.Errorf("dernière soirée %s (%d matchs), attendu D13 réduite à 1 match", last.Label, len(last.Matches))
	}
	if got[0].Label != "D03" {
		t.Errorf("première soirée %s, attendu D03 (les dix précédentes de D13)", got[0].Label)
	}
	for _, e := range got {
		if e.Label == "D14" {
			t.Error("D14, jouée après la soirée affichée, n'est pas une soirée précédente")
		}
	}
}

// TestSelectEvenings_SansLibelle : un match sans session n'appartient à aucune soirée.
func TestSelectEvenings_SansLibelle(t *testing.T) {
	current := []EveningMatch{{MatchID: "m1", StartTime: at(1, 19, 0)}}
	if got := SelectEvenings(current, current); got != nil {
		t.Errorf("attendu aucune soirée, obtenu %v", labels(got))
	}
}

// TestBuildHistory_CumulParJoueur : comptes, points et net par joueur ; rôles dans l'ordre du
// barème ; un joueur sans rôle a un net nul ; victoires du joueur principal.
func TestBuildHistory_CumulParJoueur(t *testing.T) {
	evenings := SelectEvenings(soiree("S1", 1, 19, 3), nil)
	roles := map[string][]Attribution{
		"S1-0": {{Player: "Main", Role: RoleKamikaze}, {Player: "Main", Role: RoleFirstBlood}},
		"S1-1": {{Player: "Main", Role: RoleKamikaze}, {Player: "A", Role: RoleLastCasualty}},
		"S1-2": {{Player: "Inconnu", Role: RoleFirstBlood}},
	}
	h := BuildHistory(HistoryInput{Evenings: evenings, Players: []string{"Main", "A", "B"}, Roles: roles})
	if h == nil || len(h.Evenings) != 1 {
		t.Fatalf("une soirée attendue, obtenu %+v", h)
	}
	ev := h.Evenings[0]
	if ev.Matches != 3 || ev.Wins != 2 || ev.StartTime != "2026-09-01T19:00:00Z" {
		t.Errorf("soirée %+v, attendu 3 matchs, 2 victoires, début 2026-09-01T19:00:00Z", ev)
	}
	main := ev.Players[0]
	if main.Player != "Main" || main.Points != 0 || len(main.Roles) != 2 ||
		main.Roles[0] != (domain.SquadImpactRoleCount{Role: RoleFirstBlood, Count: 1, Points: 2}) ||
		main.Roles[1] != (domain.SquadImpactRoleCount{Role: RoleKamikaze, Count: 2, Points: -2}) {
		t.Errorf("Main = %+v, attendu Premier sang ×1 (+2) puis Kamikaze ×2 (−2), net 0", main)
	}
	if a := ev.Players[1]; a.Points != -2 || len(a.Roles) != 1 {
		t.Errorf("A = %+v, attendu Boulet −2", a)
	}
	if b := ev.Players[2]; b.Player != "B" || b.Points != 0 || len(b.Roles) != 0 {
		t.Errorf("B = %+v, attendu aucun rôle", b)
	}
	if len(h.Scale) != len(stackOrder) || h.Scale[0] != (domain.SquadImpactRoleWeight{Role: RoleClutchFinisher, Points: 2}) {
		t.Errorf("barème publié %+v", h.Scale)
	}
}

// TestBuildHistory_AucunRole : aucune soirée n'attribue de rôle à l'escouade (titre sans
// événements horodatés ni équipe alliée) : le bloc se retire.
func TestBuildHistory_AucunRole(t *testing.T) {
	evenings := SelectEvenings(soiree("S1", 1, 19, 2), nil)
	if h := BuildHistory(HistoryInput{Evenings: evenings, Players: []string{"Main"}}); h != nil {
		t.Errorf("attendu nil, obtenu %+v", h)
	}
}
