package kickoff

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
)

func at(base time.Time, s float64) *time.Time {
	t := base.Add(time.Duration(s * float64(time.Second)))
	return &t
}

func ip(v int) *int { return &v }

func matchTemoin(start time.Time) Match {
	d := 600
	return Match{
		MatchID: "m1", StartUTC: start, DurationSeconds: &d,
		RealStart: at(start, 20), Source: timeline.KickoffSourceAPI,
		Participants: []Participant{
			{XUID: "2603225103667623", FirstJoined: at(start, 20), PresentAtBeginning: true, TimePlayed: ip(580)},
			{XUID: "2547408409520575", FirstJoined: at(start, 20.5), PresentAtBeginning: true, TimePlayed: ip(579)},
			{XUID: "2535000000000001", TimePlayed: ip(600)}, // présence inconnue
		},
	}
}

// Le rejeu publie un coup d'envoi : registre et temps de jeu des joueurs à présence connue changent ;
// la présence inconnue garde sa valeur.
func TestResolve_LeRejeuDecide(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	t0 := int64(26_304)
	p := Resolve(matchTemoin(start), &t0)
	if p.Kickoff.Source != timeline.KickoffSourceFilm || !p.ValueChanged() || !p.SourceChanged() {
		t.Fatalf("plan = %+v", p)
	}
	if len(p.TimePlayed) != 2 || p.TimePlayed[0].After != 573 || *p.TimePlayed[0].Before != 580 || p.TimePlayed[1].After != 573 {
		t.Fatalf("temps de jeu = %+v, attendu deux joueurs a 573", p.TimePlayed)
	}
	w := p.Write()
	if w.Kickoff == nil || w.Kickoff.Source != "film" || len(w.TimePlayed) != 2 {
		t.Fatalf("ecriture = %+v", w)
	}
}

// Un match qui porte déjà ce que la résolution décide ne change rien (idempotence).
func TestResolve_DejaAJour(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	m := matchTemoin(start)
	if p := Resolve(m, nil); !p.Unchanged() {
		t.Fatalf("plan = %+v, attendu aucun changement (estimation = registre, temps de jeu 580 et 579)", p)
	}
}

// Une estimation refusée sans rejeu efface le coup d'envoi rangé : inconnu, et le temps de jeu se
// calcule depuis le début du match.
func TestResolve_EstimationRefuseeDevientInconnu(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	d := 600
	m := Match{
		MatchID: "m1", StartUTC: start, DurationSeconds: &d,
		RealStart: at(start, 4*3600), Source: timeline.KickoffSourceUnknown,
		Participants: []Participant{
			{XUID: "2603225103667623", FirstJoined: at(start, 4*3600), PresentAtBeginning: true, TimePlayed: ip(600)},
		},
	}
	p := Resolve(m, nil)
	if p.Kickoff.Known() || p.Kickoff.Estimate != timeline.T0QualitySuspiciousHigh || !p.ValueChanged() || p.RealStart != nil {
		t.Fatalf("plan = %+v, attendu inconnu", p)
	}
	if w := p.Write(); w.Kickoff == nil || w.Kickoff.RealStart != nil || w.Kickoff.Source != "" {
		t.Fatalf("ecriture = %+v, attendu remise a inconnu", w)
	}
	if len(p.TimePlayed) != 0 {
		t.Fatalf("temps de jeu = %+v : arrive apres la fin, la valeur rangee reste", p.TimePlayed)
	}
}

// Un match sans arrivée connue (Halo 5 par exemple) : inconnu, rien ne change.
func TestResolve_SansArriveeRienNeChange(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	d := 600
	m := Match{MatchID: "h5", StartUTC: start, DurationSeconds: &d,
		Participants: []Participant{{XUID: "2603225103667623", TimePlayed: ip(598)}}}
	if p := Resolve(m, nil); !p.Unchanged() || p.Kickoff.Known() {
		t.Fatalf("plan = %+v, attendu inconnu et inchange", p)
	}
}

func TestSummarize(t *testing.T) {
	d := Summarize([]int64{5, -1, 0, 30, 2}, 2)
	if d.N != 5 || d.Min != -1 || d.Max != 30 || d.P50 != 2 || d.Within != 3 {
		t.Fatalf("distribution = %+v", d)
	}
	if Summarize(nil, 2).N != 0 {
		t.Fatal("serie vide")
	}
}
