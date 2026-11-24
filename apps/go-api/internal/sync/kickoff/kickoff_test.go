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
	p := Resolve(matchTemoin(start), FilmPublished(&t0))
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
	if p := Resolve(m, Film{}); !p.Unchanged() {
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
	p := Resolve(m, Film{})
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
	if p := Resolve(m, Film{}); !p.Unchanged() || p.Kickoff.Known() {
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

// matchFilm : un match dont le coup d'envoi rangé vient du rejeu (26,304 s), temps de jeu calculés dessus.
func matchFilm(start time.Time) Match {
	m := matchTemoin(start)
	m.RealStart, m.Source = at(start, 26.304), timeline.KickoffSourceFilm
	m.Participants[0].TimePlayed, m.Participants[1].TimePlayed = ip(573), ip(573)
	return m
}

// Rangé `film`, rejeu absent ou illisible : rien de neuf n'est su, rien ne change.
func TestResolve_RangeFilmRejeuAbsentOuIllisible(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	for nom, f := range map[string]Film{"absent": {State: FilmAbsent}, "illisible": {State: FilmUnreadable}} {
		p := Resolve(matchFilm(start), f)
		if !p.Unchanged() || !p.FilmKept || p.Kickoff.Source != timeline.KickoffSourceFilm {
			t.Fatalf("%s : plan = %+v, attendu le coup d'envoi du film garde, rien a ecrire", nom, p)
		}
	}
}

// Rangé `film`, rejeu présent qui ne publie plus de coup d'envoi (ou hors de la fenêtre) : le film dit
// qu'il n'y en a pas, l'estimation décide et les temps de jeu suivent.
func TestResolve_RangeFilmRejeuSansCoupDEnvoi(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	hors := int64(-1)
	for nom, f := range map[string]Film{"sans coup d envoi": FilmPublished(nil), "hors bornes": FilmPublished(&hors)} {
		p := Resolve(matchFilm(start), f)
		if p.FilmKept || p.Kickoff.Source != timeline.KickoffSourceAPI || !p.ValueChanged() || len(p.TimePlayed) != 2 {
			t.Fatalf("%s : plan = %+v, attendu l estimation (20 s) et deux temps de jeu recalcules", nom, p)
		}
		if nom == "hors bornes" && !p.Kickoff.FilmOutOfBounds {
			t.Fatal("le refus hors bornes n est pas dit")
		}
	}
}

// Rangé non `film`, rejeu absent : résolution normale (ici, rien ne change).
func TestResolve_RangeApiRejeuAbsent(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	if p := Resolve(matchTemoin(start), Film{State: FilmUnreadable}); p.FilmKept || !p.Unchanged() {
		t.Fatalf("plan = %+v", p)
	}
}

// Rangé `film`, rejeu absent ou illisible, un temps de jeu périmé : le registre reste (aucune
// écriture du coup d'envoi), le temps de jeu se recalcule sur le coup d'envoi rangé.
func TestResolve_RangeFilmRejeuAbsent_TempsDeJeuCorrige(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	for nom, f := range map[string]Film{"absent": {State: FilmAbsent}, "illisible": {State: FilmUnreadable}} {
		m := matchFilm(start)
		m.Participants[0].TimePlayed = ip(580) // périmé : calculé sur l'estimation à 20 s
		p := Resolve(m, f)
		if !p.FilmKept || p.KickoffChanged() || p.Write().Kickoff != nil {
			t.Fatalf("%s : plan = %+v, attendu le registre intact", nom, p)
		}
		if len(p.TimePlayed) != 1 || p.TimePlayed[0].XUID != m.Participants[0].XUID || p.TimePlayed[0].After != 573 {
			t.Fatalf("%s : temps de jeu = %+v, attendu un seul changement, a 573", nom, p.TimePlayed)
		}
	}
}

// La durée du registre est fausse (21 s pour un match dont un joueur a joué 462 s) : le coup d'envoi
// juste du rejeu est retenu, et le temps de jeu rangé des joueurs reste tel que l'API l'a donné,
// sans être recalculé sur une fenêtre de 21 s.
func TestResolve_DureeDuRegistreFausse(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	d := 21
	m := Match{
		MatchID: "m1", StartUTC: start, DurationSeconds: &d,
		RealStart: at(start, 10), Source: timeline.KickoffSourceAPI,
		Participants: []Participant{
			{XUID: "2603225103667623", FirstJoined: at(start, 10), PresentAtBeginning: true, TimePlayed: ip(462)},
			{XUID: "2547408409520575", FirstJoined: at(start, 12), PresentAtBeginning: true, TimePlayed: ip(450)},
		},
	}
	t0 := int64(38_000)
	p := Resolve(m, FilmPublished(&t0))
	if p.Kickoff.Source != timeline.KickoffSourceFilm || p.Kickoff.FilmOutOfBounds || p.Kickoff.OffsetMs != 38_000 {
		t.Fatalf("coup d'envoi = %+v, attendu le rejeu retenu", p.Kickoff)
	}
	if len(p.TimePlayed) != 0 {
		t.Fatalf("temps de jeu = %+v : la durée est fausse, les valeurs rangées restent", p.TimePlayed)
	}
}

// Un coup d'envoi `film` rangé, gardé parce que le rejeu n'est plus lisible, est contrôlé contre la
// fenêtre du match comme un coup d'envoi lu : hors fenêtre il est refusé (compté), et le match
// retombe sur l'estimation. Dans la fenêtre, ou dans la fenêtre bornée par le plus long temps de
// jeu quand la durée du registre est fausse, il est gardé tel quel.
func TestResolve_FilmGardeControleContreLaFenetre(t *testing.T) {
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	cas := []struct {
		nom      string
		duree    int
		joue     int
		offset   float64
		garde    bool
		source   timeline.KickoffSource
		horsBorn bool
	}{
		{"dans la fenêtre", 600, 580, 26, true, timeline.KickoffSourceFilm, false},
		{"après la durée du match", 600, 580, 700, false, timeline.KickoffSourceAPI, true},
		{"durée fausse, dans le temps de jeu", 21, 462, 38, true, timeline.KickoffSourceFilm, false},
		{"durée fausse, au-delà du temps de jeu", 21, 462, 700, false, timeline.KickoffSourceAPI, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d := c.duree
			m := Match{
				MatchID: "m1", StartUTC: start, DurationSeconds: &d,
				RealStart: at(start, c.offset), Source: timeline.KickoffSourceFilm,
				Participants: []Participant{
					{XUID: "2603225103667623", FirstJoined: at(start, 20), PresentAtBeginning: true, TimePlayed: ip(c.joue)},
				},
			}
			for _, f := range []Film{{State: FilmAbsent}, {State: FilmUnreadable}} {
				p := Resolve(m, f)
				if p.FilmKept != c.garde || p.Kickoff.Source != c.source || p.Kickoff.FilmOutOfBounds != c.horsBorn {
					t.Fatalf("film %v : gardé=%v source=%q horsBornes=%v, attendu %v/%q/%v",
						f.State, p.FilmKept, p.Kickoff.Source, p.Kickoff.FilmOutOfBounds, c.garde, c.source, c.horsBorn)
				}
			}
		})
	}
}
