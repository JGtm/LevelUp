package service

// tactical_service_cellule_horloge_test.go — l'instant que l'horloge affichée du rejeu lira à
// l'événement : `max(0, instant − T0)` sur l'horloge « match », absent sans T0 et sur l'horloge « film ».

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func p64(v int64) *int64 { return &v }

func TestInstantAffiche(t *testing.T) {
	cas := []struct {
		nom     string
		clock   string
		instant int64
		t0      *int64
		want    *int64
	}{
		{"countdown de 22 s : 462 000 ms -> 7:20", domain.TacticalClockMatch, 462_000, p64(22_000), p64(440_000)},
		{"evenement avant le coup d'envoi : plancher a 0", domain.TacticalClockMatch, 10_000, p64(22_000), p64(0)},
		{"T0 inconnu : pas d'instant", domain.TacticalClockMatch, 462_000, nil, nil},
		{"horloge du film : pas d'instant", domain.TacticalClockFilm, 30_000, p64(22_000), nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := instantAffiche(c.clock, c.instant, c.t0)
			if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
				t.Errorf("instantAffiche = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCellule_InstantAfficheDuT0DuMatch : le service publie l'instant affiché d'après le T0 que le
// registre porte pour chaque match (celui du coup d'envoi mesuré dans le film quand il a été reporté).
func TestCellule_InstantAfficheDuT0DuMatch(t *testing.T) {
	repo := repoDeuxFaces()
	repo.pos.Points[0].TimeMs = 462_000
	repo.pos.Points[1].TimeMs = 470_000
	repo.pos.Univers.Matchs[0].T0Ms = p64(22_000)
	got := celluleLue(t, NewTacticalService(repo, capsCompletes(), tsMoi),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if len(got.Contributions) != 2 {
		t.Fatalf("contributions = %+v, want 2", got.Contributions)
	}
	for i, want := range []int64{440_000, 448_000} {
		if c := got.Contributions[i]; c.InstantAfficheMs == nil || *c.InstantAfficheMs != want {
			t.Errorf("contribution %d : instant affiché %v, want %d", i, c.InstantAfficheMs, want)
		}
		if got.Contributions[i].InstantMs == 0 {
			t.Errorf("contribution %d : l'instant brut doit rester publié", i)
		}
	}
}

// TestCellule_InstantAfficheAbsentSansT0 : un match dont le T0 est inconnu ne publie aucun instant affiché.
func TestCellule_InstantAfficheAbsentSansT0(t *testing.T) {
	repo := repoDeuxFaces()
	got := celluleLue(t, NewTacticalService(repo, capsCompletes(), tsMoi),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	for i, c := range got.Contributions {
		if c.InstantAfficheMs != nil {
			t.Errorf("contribution %d : instant affiché %d, want absent (T0 inconnu)", i, *c.InstantAfficheMs)
		}
	}
}
