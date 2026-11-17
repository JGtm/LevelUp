package tactical

import (
	"fmt"
	"strings"
	"testing"
)

// entreeCaracterisation : un match lu au film, trois joueurs et plusieurs vies par joueur :
// liberation du corps, compte a rebours, fin de manche, vie sans preuve.
func entreeCaracterisation() EntreeOccupation {
	return entreeTest([]DebutCompte{{XUID: "a", Frame: 598}},
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 103}),
		pisteTest("a", 0, 150, 300, 50, &FinLue{Raison: RaisonSupprimee, Frame: 300}),
		pisteTest("a", 0, 400, 600, 90, nil),
		pisteTest("a", 0, 700, 800, 120, nil),
		pisteTest("b", 1, 0, 200, 20, &FinLue{Raison: RaisonLiberee, Frame: 200}),
		pisteTest("b", 1, 260, 500, 60, nil),
		pisteTest("b", 1, 560, 700, 70, nil),
		pisteTest("d", 0, 0, 900, 30, nil),
		pisteTest("e", 1, 5, 900, 40, nil),
	)
}

// decrire rend, pour chaque piste, ce que l'index dit de la vie qu'elle ouvre.
func decrire(e EntreeOccupation) []string {
	spawns := NouvelIndexVies(e).Spawns()
	var out []string
	for i := range e.Pistes {
		s, ok := spawns[i]
		if !ok {
			out = append(out, fmt.Sprintf("%d: sans spawn", i))
			continue
		}
		ligne := fmt.Sprintf("%d: frame=%d reap=%v manche=%v", i, s.Frame, s.Reapparition, s.DebutDeManche)
		if s.Mort != nil {
			ligne += fmt.Sprintf(" mort=%d@%.0f", s.Mort.Frame, s.Mort.X)
		}
		for _, v := range s.Instantane {
			ligne += fmt.Sprintf(" [%s e%d x%.0f]", v.XUID, v.Equipe, v.X)
		}
		out = append(out, ligne)
	}
	return out
}

// TestCaracterisation_SpawnsALaLectureDuFilm fige la decision « reapparition » d'un document qui porte
// des lectures de fin de vie du film, avec la mort et l'instantane de chaque reapparition.
func TestCaracterisation_SpawnsALaLectureDuFilm(t *testing.T) {
	got := strings.Join(decrire(entreeCaracterisation()), "\n")
	if got != attenduSpawnsFilm {
		t.Errorf("spawns lus au film :\n%s\nattendu :\n%s", got, attenduSpawnsFilm)
	}
}

const attenduSpawnsFilm = `0: frame=0 reap=false manche=false
1: frame=150 reap=true manche=false mort=100@11 [b e1 x20] [d e0 x30] [e e1 x40]
2: frame=400 reap=false manche=true
3: frame=700 reap=true manche=false mort=600@91 [b e1 x71] [d e0 x30] [e e1 x40]
4: frame=0 reap=false manche=false
5: frame=260 reap=true manche=false mort=200@21 [a e0 x50] [d e0 x30] [e e1 x40]
6: frame=560 reap=false manche=false
7: frame=0 reap=false manche=false
8: frame=5 reap=false manche=false`
