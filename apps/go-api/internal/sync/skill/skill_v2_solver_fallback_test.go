package skill

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
)

func ptr(v float64) *float64 { return &v }

// matchDivergent rend le match 4v6 déterministe dont le solveur ne converge pas avec les
// observations de frags et de morts (tous à N(25, 25/3), priors par défaut, l'équipe A gagne),
// poids de temps joué de 0,5 pour les deux derniers de l'équipe B et un abandon dans l'équipe A.
func matchDivergent() countedMatch {
	g := skillv2.Gaussian{Mu: 25, Sigma: 25.0 / 3.0}
	killsA, deathsA := []float64{17, 17, 15, 1}, []float64{9, 13, 10, 13}
	killsB, deathsB := []float64{18, 7, 20, 3, 0, 12}, []float64{9, 3, 11, 20, 7, 18}
	weightsB := []float64{1, 1, 1, 1, 0.5, 0.5}

	counts := &skillv2.CountInputs{}
	for i := range killsA {
		counts.TeamA = append(counts.TeamA, skillv2.PlayerCounts{Kills: ptr(killsA[i]), Deaths: ptr(deathsA[i]), Weight: 1})
	}
	for i := range killsB {
		counts.TeamB = append(counts.TeamB, skillv2.PlayerCounts{Kills: ptr(killsB[i]), Deaths: ptr(deathsB[i]), Weight: weightsB[i]})
	}
	counts.TeamA[3].Quit = true
	counts.TeamA[3].QuitPenaltyDelta = skillv2.DefaultQuitDeltaUnrelated

	return countedMatch{
		matchID: "m_divergent",
		match: skillv2.TwoTeamMatch{
			TeamA:   []skillv2.Gaussian{g, g, g, g},
			TeamB:   []skillv2.Gaussian{g, g, g, g, g, g},
			ResultA: skillv2.TeamWin,
		},
		counts: counts,
		priors: skillv2.DefaultPriors(),
	}
}

// Le match de la fixture ne converge pas avec ses comptes, via la chaîne réelle du calcul : le
// repli le rattrape, rend une note, compte le repli, journalise, et conserve le poids de temps
// joué et l'abandon (le résultat égale un calcul direct sans frags ni morts, et diffère d'un calcul
// qui aurait aussi perdu l'abandon).
func TestSolveWithCountFallback_NoteRendueApresNonConvergence(t *testing.T) {
	in := matchDivergent()
	if _, _, err := skillv2.UpdateTwoTeamWithCountsEP(in.match, in.counts, in.priors); !errors.Is(err, skillv2.ErrNotConverged) {
		t.Fatalf("la fixture doit diverger avec les comptes : err = %v", err)
	}
	logs, restore := captureSlog(t)
	defer restore()
	before := solverCountFallbackTotal.Value()

	gotA, gotB, err := solveWithCountFallback(context.Background(), skillv2.UpdateTwoTeamWithCountsEP, in)

	if err != nil {
		t.Fatalf("le repli doit rendre une note : err = %v", err)
	}
	if len(gotA) != 4 || len(gotB) != 6 {
		t.Fatalf("posteriors = %d et %d, attendu 4 et 6", len(gotA), len(gotB))
	}
	if delta := solverCountFallbackTotal.Value() - before; delta != 1 {
		t.Errorf("compteur du repli +%d, attendu +1", delta)
	}
	for _, want := range []string{"level=WARN", "match_id=m_divergent", "recalcul sans frags ni morts"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("journal sans %q :\n%s", want, logs.String())
		}
	}

	wantA, wantB, err := skillv2.UpdateTwoTeamWithCountsEP(in.match, withoutCountObservations(in.counts), in.priors)
	if err != nil {
		t.Fatalf("calcul direct sans comptes : %v", err)
	}
	if !sameGaussians(gotA, wantA) || !sameGaussians(gotB, wantB) {
		t.Errorf("le repli diffère d'un calcul direct sans comptes :\n got  %v %v\n want %v %v", gotA, gotB, wantA, wantB)
	}

	noQuit := withoutCountObservations(in.counts)
	noQuit.TeamA[3].Quit = false
	lostA, _, err := skillv2.UpdateTwoTeamWithCountsEP(in.match, noQuit, in.priors)
	if err != nil {
		t.Fatalf("calcul sans comptes ni abandon : %v", err)
	}
	if sameGaussians(gotA, lostA) {
		t.Error("l'abandon n'est pas conservé par le repli")
	}
}

func sameGaussians(a, b []skillv2.Gaussian) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i].Mu-b[i].Mu) > 1e-12 || math.Abs(a[i].Sigma-b[i].Sigma) > 1e-12 {
			return false
		}
	}
	return true
}

// Un second échec du recalcul sans comptes n'est pas avalé : l'erreur est rendue, reste reconnue
// comme une non-convergence (donc classée comme avant par noteComputeFailure), le recalcul reçoit
// bien des comptes sans frags ni morts avec le poids et l'abandon, et l'entrée n'est pas modifiée.
func TestSolveWithCountFallback_SecondEchecResteClasse(t *testing.T) {
	in := matchDivergent()
	var seen []*skillv2.CountInputs
	solve := func(_ skillv2.TwoTeamMatch, c *skillv2.CountInputs, _ skillv2.Priors) ([]skillv2.Gaussian, []skillv2.Gaussian, error) {
		seen = append(seen, c)
		return nil, nil, fmt.Errorf("UpdateTwoTeamWithCountsEP: %w en 500 itérations", skillv2.ErrNotConverged)
	}
	before := solverCountFallbackTotal.Value()

	gotA, gotB, err := solveWithCountFallback(context.Background(), solve, in)

	if err == nil || !errors.Is(err, skillv2.ErrNotConverged) {
		t.Fatalf("err = %v, attendu une erreur reconnue comme ErrNotConverged", err)
	}
	if gotA != nil || gotB != nil {
		t.Errorf("aucun posterior attendu sur échec, reçu %v %v", gotA, gotB)
	}
	if len(seen) != 2 || seen[0] != in.counts {
		t.Fatalf("appels du solveur = %d, attendu 2 (comptes complets puis sans comptes)", len(seen))
	}
	if delta := solverCountFallbackTotal.Value() - before; delta != 1 {
		t.Errorf("compteur du repli +%d, attendu +1", delta)
	}

	retry := seen[1]
	for side, team := range [][]skillv2.PlayerCounts{retry.TeamA, retry.TeamB} {
		for i, pc := range team {
			if pc.Kills != nil || pc.Deaths != nil {
				t.Errorf("équipe %d joueur %d : frags ou morts conservés au recalcul", side, i)
			}
		}
	}
	if retry.TeamB[4].Weight != 0.5 || retry.TeamA[0].Weight != 1 {
		t.Errorf("poids de temps joué non conservés : %v %v", retry.TeamB[4].Weight, retry.TeamA[0].Weight)
	}
	if !retry.TeamA[3].Quit || retry.TeamA[3].QuitPenaltyDelta != skillv2.DefaultQuitDeltaUnrelated {
		t.Errorf("abandon non conservé : %+v", retry.TeamA[3])
	}
	if in.counts.TeamA[0].Kills == nil || in.counts.TeamB[5].Deaths == nil {
		t.Error("les comptes d'entrée ont été modifiés")
	}

	var s shadowRunStats
	shadowRunContext{}.noteComputeFailure(err, &s)
	if s.skippedNotConverged != 1 {
		t.Errorf("classement du second échec : non convergés = %d, attendu 1", s.skippedNotConverged)
	}
}

// Seule la non-convergence avec des comptes déclenche le repli : une autre erreur, ou une
// non-convergence sans observation à retirer, est rendue telle quelle après un seul appel.
func TestSolveWithCountFallback_SeulementLaNonConvergenceAvecComptes(t *testing.T) {
	autre := errors.New("skill_v2: Beta doit être > 0")
	nonConverge := fmt.Errorf("%w en 500 itérations", skillv2.ErrNotConverged)
	sansObservation := &skillv2.CountInputs{
		TeamA: []skillv2.PlayerCounts{{Weight: 1, Quit: true}},
		TeamB: []skillv2.PlayerCounts{{Weight: 1}},
	}
	cases := []struct {
		name   string
		err    error
		counts *skillv2.CountInputs
	}{
		{"autre erreur", autre, matchDivergent().counts},
		{"non-convergence sans comptes", nonConverge, nil},
		{"non-convergence sans observation", nonConverge, sansObservation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			solve := func(skillv2.TwoTeamMatch, *skillv2.CountInputs, skillv2.Priors) ([]skillv2.Gaussian, []skillv2.Gaussian, error) {
				calls++
				return nil, nil, tc.err
			}
			in := matchDivergent()
			in.counts = tc.counts
			before := solverCountFallbackTotal.Value()

			_, _, err := solveWithCountFallback(context.Background(), solve, in)

			if err != tc.err {
				t.Errorf("err = %v, attendu l'erreur d'origine %v", err, tc.err)
			}
			if calls != 1 {
				t.Errorf("appels du solveur = %d, attendu 1", calls)
			}
			if delta := solverCountFallbackTotal.Value() - before; delta != 0 {
				t.Errorf("compteur du repli +%d, attendu +0", delta)
			}
		})
	}
}

// Un calcul qui converge du premier coup ne passe pas par le repli.
func TestSolveWithCountFallback_ConvergenceDirecte(t *testing.T) {
	in := matchDivergent()
	in.counts = nil
	before := solverCountFallbackTotal.Value()

	a, b, err := solveWithCountFallback(context.Background(), skillv2.UpdateTwoTeamWithCountsEP, in)

	if err != nil || len(a) != 4 || len(b) != 6 {
		t.Fatalf("convergence directe : %d %d err=%v", len(a), len(b), err)
	}
	if delta := solverCountFallbackTotal.Value() - before; delta != 0 {
		t.Errorf("compteur du repli +%d, attendu +0", delta)
	}
}

// Le calcul d'un match entier (computeMatchToSkillV2, le chemin du rejeu et du sync) rattrape la
// non-convergence : rosters et comptes de la fixture, états seedés depuis les priors. Un appel
// direct au solveur, sans le repli, rendrait ErrNotConverged.
func TestComputeMatchToSkillV2_RattrapeLaNonConvergence(t *testing.T) {
	const gameplayDurMs = 600_000
	div := matchDivergent()
	member := func(xuid string, pc skillv2.PlayerCounts) rosterMember {
		return rosterMember{
			xuid:           xuid,
			kills:          pc.Kills,
			deaths:         pc.Deaths,
			timePlayedSecs: sql.NullFloat64{Float64: pc.Weight * gameplayDurMs / 1000, Valid: true},
		}
	}
	var teamA, teamB []rosterMember
	for i, pc := range div.counts.TeamA {
		pc.Quit = false
		teamA = append(teamA, member(fmt.Sprintf("a%d", i), pc))
	}
	for i, pc := range div.counts.TeamB {
		teamB = append(teamB, member(fmt.Sprintf("b%d", i), pc))
	}
	repo := &fakeSkillV2Repo{states: map[string]*domain.SkillV2State{}}
	before := solverCountFallbackTotal.Value()

	got, err := computeMatchToSkillV2(context.Background(), repo, nil, skillv2.DefaultPriors(), nil, quitTimeline{},
		"m_divergent", "arena_slayer", time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC),
		teamA, teamB, skillv2.TeamWin, "a0", gameplayDurMs)

	if err != nil {
		t.Fatalf("le calcul du match doit être rattrapé par le repli : %v", err)
	}
	if got.ownerNew == nil || len(got.newA) != 4 || len(got.newB) != 6 {
		t.Fatalf("note attendue : ownerNew=%v newA=%d newB=%d", got.ownerNew, len(got.newA), len(got.newB))
	}
	if delta := solverCountFallbackTotal.Value() - before; delta != 1 {
		t.Errorf("compteur du repli +%d, attendu +1", delta)
	}
}

// Hors du moteur skill_v2, de ce fichier et des tests, personne n'appelle (ni ne référence)
// skillv2.UpdateTwoTeamWithCountsEP : un appel direct contournerait le repli et rouvrirait les
// matchs sans note. Le chemin est solveMatchWithCounts.
func TestUpdateTwoTeamWithCountsEPAppeleSeulementParLeRepli(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`\.UpdateTwoTeamWithCountsEP\b`)
	engineDir := filepath.Join(root, "internal", "analysis", "skill_v2") + string(filepath.Separator)
	allowed := filepath.Join(root, "internal", "sync", "skill", "skill_v2_solver_fallback.go")
	scanned := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			path == allowed || strings.HasPrefix(path, engineDir) {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		if pattern.Match(src) {
			t.Errorf("%s référence skillv2.UpdateTwoTeamWithCountsEP : passer par solveMatchWithCounts", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("garde-rail aveugle : %d fichiers parcourus sous %s", scanned, root)
	}
}
