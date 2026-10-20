package duckdb

// career_repo_rivals_test.go — LES DEUX CLASSEMENTS DES RIVAUX, TRIÉS EN GO (lot B du plan perf,
// 2026-09-27) : l'ordre est celui de l'ancien `ORDER BY <clé> DESC, match_count DESC,
// opp_xuid ASC LIMIT 10` SQL, ex aequo compris, et l'agrégat partagé n'est pas modifié.

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
)

func rivalDeTest(xuid string, frags, deaths, matchs int) rivalLu {
	return rivalLu{CareerRivalRawRow: domain.CareerRivalRawRow{XUID: xuid, Frags: frags, Deaths: deaths, MatchCount: matchs}}
}

func TestClasserRivaux_OrdreTotalEtLimite(t *testing.T) {
	tous := []rivalLu{
		rivalDeTest("b", 5, 1, 2),
		rivalDeTest("a", 5, 1, 2), // ex aequo complet avec b : xuid croissant
		rivalDeTest("c", 5, 9, 3), // même clé, plus de matchs : devant
		rivalDeTest("d", 7, 0, 1),
	}
	for i := 0; i < 12; i++ {
		tous = append(tous, rivalDeTest(fmt.Sprintf("z%02d", i), 1, 2, 1))
	}
	avant := fmt.Sprint(tous)

	vic := classerRivaux(tous, func(l rivalLu) int { return l.Frags })
	var ordre []string
	for _, l := range vic {
		ordre = append(ordre, l.XUID)
	}
	want := "[d c a b z00 z01 z02 z03 z04 z05]"
	if got := fmt.Sprint(ordre); got != want {
		t.Errorf("souffre-douleur = %s, want %s", got, want)
	}

	nem := classerRivaux(tous, func(l rivalLu) int { return l.Deaths })
	if nem[0].XUID != "c" || nem[1].XUID != "z00" || len(nem) != rivauxParClassement {
		t.Errorf("némésis = %v, want c puis z00 (deux morts, un match, xuid croissant), %d lignes", nem, rivauxParClassement)
	}
	if fmt.Sprint(tous) != avant {
		t.Error("classerRivaux a modifié l'agrégat partagé par les deux classements")
	}
	if got := classerRivaux(nil, func(l rivalLu) int { return l.Frags }); len(got) != 0 {
		t.Errorf("aucun rival : got %v", got)
	}
}
