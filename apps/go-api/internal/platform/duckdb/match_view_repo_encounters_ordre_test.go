package duckdb

// match_view_repo_encounters_ordre_test.go — L'ORDRE DES RENCONTRES D'UN MATCH EST DETERMINISTE.
//
// Q23 trie par nombre de matchs en commun ; deux joueurs a egalite sortaient dans l'ordre que
// l'agregat voulait bien rendre. Le departage est le xuid.

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"
)

func TestMatchViewRepo_Encounters_OrdreDeterministeALegalite(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tacMatch(t, pdb, "m_courant", tacCarteA, base.Add(time.Hour))
	tacMatch(t, pdb, "m_ancien", tacCarteA, base)
	tacParticipant(t, pdb, "m_courant", tacXUIDMoi, 0, 2)
	tacParticipant(t, pdb, "m_ancien", tacXUIDMoi, 0, 2)

	// Quinze rencontres posees dans l'ordre inverse du xuid ; deux d'entre elles sont aussi
	// dans le match ancien (deux matchs en commun), les treize autres a egalite a un match.
	const n = 15
	for i := n; i >= 1; i-- {
		xuid := fmt.Sprintf("2533274000001%03d", i)
		tacParticipant(t, pdb, "m_courant", xuid, i%2, 2)
		if i == 4 || i == 11 {
			tacParticipant(t, pdb, "m_ancien", xuid, 1, 2)
		}
	}
	attendu := []string{"2533274000001004", "2533274000001011"}
	for i := 1; i <= n; i++ {
		if i != 4 && i != 11 {
			attendu = append(attendu, fmt.Sprintf("2533274000001%03d", i))
		}
	}
	sort.Strings(attendu[2:])

	repo := NewMatchViewRepo(pdb, tacXUIDMoi)
	for essai := 0; essai < 20; essai++ {
		got, err := repo.GetMatchEncounters(context.Background(), "m_courant", tacXUIDMoi)
		if err != nil {
			t.Fatalf("GetMatchEncounters: %v", err)
		}
		ordre := make([]string, len(got))
		for i, e := range got {
			ordre[i] = e.XUID
		}
		if fmt.Sprint(ordre) != fmt.Sprint(attendu) {
			t.Fatalf("essai %d : ordre %v, attendu %v", essai, ordre, attendu)
		}
	}
}
