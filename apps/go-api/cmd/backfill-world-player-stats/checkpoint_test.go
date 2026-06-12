//go:build cgo

package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// TestSeasonRank_RecentFirst : le tri par seasonRank met les saisons RÉCENTES
// d'abord (numérique), pas l'ordre alphabétique SQL qui mettait csrseason6-1 avant
// csrseason13-2 (bug observé : le backfill -season all démarrait sur les vieilles).
func TestSeasonRank_RecentFirst(t *testing.T) {
	got := []string{"csrseason6-1", "csrseason13-2", "csrseason10-1", "csrseason3-1", "csrseason12-1"}
	sort.SliceStable(got, func(i, j int) bool { return seasonRank(got[i]) > seasonRank(got[j]) })
	want := []string{"csrseason13-2", "csrseason12-1", "csrseason10-1", "csrseason6-1", "csrseason3-1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ordre = %v, want %v (récent d'abord)", got, want)
	}
}

// Les joueurs DÉJÀ tentés (done ∪ failed) sont sautés à la reprise ; -retry-failed
// ré-inclut les seuls échecs.
func TestCheckpoint_RemainingSkipsDoneAndFailed(t *testing.T) {
	cp := &checkpoint{Seasons: map[string]*seasonProgress{}}
	gts := []string{"A", "B", "C", "D"}
	cp.markDone("s", []string{"A"})
	cp.markFailed("s", []string{"B"})

	if got := cp.remaining("s", gts, 0, false); !reflect.DeepEqual(got, []string{"C", "D"}) {
		t.Errorf("remaining (défaut) = %v, want [C D]", got)
	}
	if got := cp.remaining("s", gts, 0, true); !reflect.DeepEqual(got, []string{"B", "C", "D"}) {
		t.Errorf("remaining (-retry-failed) = %v, want [B C D]", got)
	}
}

// attemptedCount = |done ∪ failed| restreint aux gamertags courants (un gamertag du
// checkpoint absent de la saison ne compte pas).
func TestCheckpoint_AttemptedCountCountsDoneAndFailed(t *testing.T) {
	cp := &checkpoint{Seasons: map[string]*seasonProgress{}}
	gts := []string{"A", "B", "C"}
	cp.markDone("s", []string{"A"})
	cp.markFailed("s", []string{"B"})
	if n := cp.attemptedCount("s", gts); n != 2 {
		t.Errorf("attemptedCount = %d, want 2 (A done + B failed)", n)
	}
	cp.markFailed("s", []string{"X"}) // hors gts → non compté
	if n := cp.attemptedCount("s", gts); n != 2 {
		t.Errorf("attemptedCount (X hors saison) = %d, want 2", n)
	}
}

func TestCheckpoint_MarkFailedDedups(t *testing.T) {
	cp := &checkpoint{Seasons: map[string]*seasonProgress{}}
	cp.markFailed("s", []string{"A", "A", "B"})
	cp.markFailed("s", []string{"B", "C"})
	if got := cp.get("s").Failed; !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("Failed = %v, want [A B C] (dé-dupliqué)", got)
	}
}

// -retry-failed : un joueur re-tenté avec succès sort de "failed".
func TestCheckpoint_MarkDoneClearsFailed(t *testing.T) {
	cp := &checkpoint{Seasons: map[string]*seasonProgress{}}
	cp.markFailed("s", []string{"A", "B"})
	cp.markDone("s", []string{"A"})
	if got := cp.get("s").Failed; !reflect.DeepEqual(got, []string{"B"}) {
		t.Errorf("Failed après retry réussi de A = %v, want [B]", got)
	}
}

// Régression : 2 joueurs en échec persistant rebloquaient la saison à chaque run
// (jamais marqués done → remaining jamais vide → markSeasonCompleteIfFull jamais
// déclenché). Désormais ils comptent comme tentés et la saison se complète.
func TestCheckpoint_SeasonCompletesDespiteFailures(t *testing.T) {
	cp := &checkpoint{Seasons: map[string]*seasonProgress{}}
	gts := make([]string, 502)
	for i := range gts {
		gts[i] = "p" + strconv.Itoa(i)
	}
	cp.markDone("s", gts[:500])
	cp.markFailed("s", gts[500:]) // 2 échecs

	if got := cp.remaining("s", gts, 0, false); len(got) != 0 {
		t.Fatalf("remaining = %d joueurs, want 0 (tous tentés)", len(got))
	}
	if att := cp.attemptedCount("s", gts); att != 502 {
		t.Fatalf("attemptedCount = %d, want 502", att)
	}
	f := cliFlags{checkpoint: filepath.Join(t.TempDir(), "cp.json")}
	markSeasonCompleteIfFull("s", gts, f, cp, cp.attemptedCount("s", gts))
	if !cp.completed("s") {
		t.Error("saison non marquée complète alors que 502/502 tentés (500 done + 2 failed)")
	}
}
