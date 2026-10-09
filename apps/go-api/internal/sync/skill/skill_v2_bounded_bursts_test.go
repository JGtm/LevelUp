//go:build cgo

package skill

// skill_v2_bounded_bursts_test.go — rafales d'écrivain BORNÉES du shadow LUSR v2 (lot perf
// L9-go, 2026-09-23, revue adversariale B, P1). Une rafale unique par joueur et par cycle
// (lot L6) tenait l'écrivain partagé 2,5 s d'un seul tenant pour 300 candidats sur la base
// de test (TestRevB_SingleBurstHoldGrowsWithBacklog). Désormais : au plus 50 matchs ou 2 s
// par rafale, reprise du reste de la file dans le même cycle, zéro écrivain en régime
// stationnaire (inchangé, skill_v2_watermark_test.go), mêmes écritures.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

// burstRecordingAccess : roRwSplitAccess qui mesure, pour chaque rafale Write, la durée de
// détention et le nombre de matchs qu'elle a persistés (lignes d'état du joueur ajoutées
// sous elle, comptées sur son handle avant de le rendre).
type burstRecordingAccess struct {
	roRwSplitAccess
	holds   []time.Duration
	matches []int
	states  int
}

func (a *burstRecordingAccess) Write(ctx context.Context, step string) (*sql.DB, func(), error) {
	db, release, err := a.roRwSplitAccess.Write(ctx, step)
	if err != nil {
		return nil, nil, err
	}
	start := time.Now()
	return db, func() {
		a.holds = append(a.holds, time.Since(start))
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM player_skill_state_v2 WHERE xuid = 'owner'`).Scan(&n); err == nil {
			a.matches = append(a.matches, n-a.states)
			a.states = n
		}
		release()
	}, nil
}

// withBurstLimits remplace les bornes des rafales le temps du test.
func withBurstLimits(t *testing.T, l lusrBurstLimits) {
	t.Helper()
	prev := defaultLUSRBurstLimits
	defaultLUSRBurstLimits = l
	t.Cleanup(func() { defaultLUSRBurstLimits = prev })
}

// logLinesWith rend les lignes du journal qui portent msg.
func logLinesWith(out, msg string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, msg) {
			lines = append(lines, l)
		}
	}
	return lines
}

// TestLUSRV2Shadow_RafalesBornees_300Candidats : 300 candidats neufs (premier cycle d'un
// joueur jamais scoré) passent sous plusieurs rafales, chacune d'au plus 50 matchs et de
// moins de 2 s, toutes dans le même cycle ; une ligne INFO par rafale.
func TestLUSRV2Shadow_RafalesBornees_300Candidats(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "")
	path := openShadowTestFileDB(t)
	seed, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 300; i++ {
		seedShadow2v2(t, seed, fmt.Sprintf("m%04d", i), base.Add(time.Duration(i)*time.Hour))
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}

	acc := &burstRecordingAccess{roRwSplitAccess: roRwSplitAccess{path: path}}
	buf, restore := captureSlog(t)
	defer restore()
	processed, err := RunLUSRV2ShadowOwnerOnly(context.Background(), nil, acc, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 300 {
		t.Fatalf("processed = %d, want 300 (toute la file dans le même cycle)", processed)
	}
	if acc.writeCalls < 6 || len(acc.matches) != acc.writeCalls {
		t.Fatalf("%d rafales (%d mesurées), want au moins 6 (300 matchs, 50 au plus par rafale)", acc.writeCalls, len(acc.matches))
	}
	total := 0
	for i, n := range acc.matches {
		total += n
		if n > defaultLUSRBurstLimits.maxMatches || acc.holds[i] >= defaultLUSRBurstLimits.maxHold {
			t.Errorf("rafale %d : %d matchs tenus %v, want <= %d et < %v", i+1, n, acc.holds[i],
				defaultLUSRBurstLimits.maxMatches, defaultLUSRBurstLimits.maxHold)
		}
	}
	if total != 300 {
		t.Errorf("matchs persistés sous les rafales = %d, want 300", total)
	}
	t.Logf("%d rafales, matchs %v, détentions %v", acc.writeCalls, acc.matches, acc.holds)
	out := buf.String()
	if got := len(logLinesWith(out, `msg="lusr_v2: rafale bornée"`)); got != acc.writeCalls {
		t.Errorf("%d ligne(s) INFO `lusr_v2: rafale bornée`, want une par rafale (%d)", got, acc.writeCalls)
	}
	requireLogLine(t, out, `msg="lusr_v2: rafale terminée"`,
		"candidates=300", "new=300", "processed=300", fmt.Sprintf("bursts=%d", acc.writeCalls))
}

// TestLUSRV2Shadow_RafaleRendueApresMaxHold : une rafale qui a tenu l'écrivain maxHold le
// rend après le match en cours, et la file reprend sous une nouvelle rafale. Horloge
// pilotée : 500 ms par lecture (une au début de la rafale, une après chaque match) → 2 s
// atteintes au 4e match ; 10 candidats = rafales de 4, 4 et 2.
func TestLUSRV2Shadow_RafaleRendueApresMaxHold(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "")
	clock := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	withBurstLimits(t, lusrBurstLimits{maxMatches: 50, maxHold: 2 * time.Second, now: func() time.Time {
		clock = clock.Add(500 * time.Millisecond)
		return clock
	}})
	db := openShadowTestDB(t)
	base := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		seedShadow2v2(t, db, fmt.Sprintf("mh%02d", i), base.Add(time.Duration(i)*time.Hour))
	}
	acc := &orderTrackingAccess{db: db}
	buf, restore := captureSlog(t)
	defer restore()
	processed, err := RunLUSRV2ShadowOwnerOnly(context.Background(), nil, acc, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 10 || acc.writeCalls != 3 {
		t.Fatalf("processed=%d, rafales=%d, want 10 et 3 (4 + 4 + 2)", processed, acc.writeCalls)
	}
	lines := logLinesWith(buf.String(), `msg="lusr_v2: rafale bornée"`)
	for i, want := range []string{"matches=4", "matches=4", "matches=2"} {
		if i >= len(lines) || !strings.Contains(lines[i], want) || !strings.Contains(lines[i], fmt.Sprintf("bursts=%d", i+1)) {
			t.Errorf("rafale %d : ligne %q, want %s", i+1, strings.Join(lines, " | "), want)
		}
	}
	if acc.violation != "" {
		t.Errorf("garde anti-deadlock violée : %s", acc.violation)
	}
}

// TestLUSRV2Shadow_ParityWithLegacyOrchestration_RafalesMultiples : la parité des écritures
// du lot L6 tient quand la file se découpe en rafales. Avec deux matchs par rafale, le match
// dont l'écriture canonique échoue (groupe btb tenu) et le match suivant du même groupe
// tombent dans deux rafales différentes : le groupe doit rester tenu d'une rafale à l'autre
// (anti-trou, fix 2026-06-07), comme dans l'orchestration d'avant.
func TestLUSRV2Shadow_ParityWithLegacyOrchestration_RafalesMultiples(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")
	t.Setenv(lusrModeCouplingEnvFlag, "1")
	t.Setenv(lusrSquadOffsetEnvFlag, "1")
	withBurstLimits(t, lusrBurstLimits{maxMatches: 2, maxHold: time.Hour, now: time.Now})

	legacy, current := newParityWorld(t), newParityWorld(t)
	for _, w := range []parityWorld{legacy, current} {
		seedShadowFixtures(t, w.shared, parityHistory()...)
		if n := runLegacyOrchestration(t, w, "owner"); n != 4 {
			t.Fatalf("phase 1 : processed = %d, want 4", n)
		}
		seedShadowFixtures(t, w.shared, parityArrivals()...)
	}
	nLegacy := runLegacyOrchestration(t, legacy, "owner")
	acc := &orderTrackingAccess{db: current.shared}
	nCurrent, err := RunLUSRV2ShadowOwnerOnly(context.Background(), current.player, acc, "owner")
	if err != nil {
		t.Fatalf("phase 2 : RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if nLegacy != 3 || nCurrent != 3 {
		t.Errorf("phase 2 : processed avant=%d après=%d, want 3 et 3", nLegacy, nCurrent)
	}
	if acc.writeCalls < 3 {
		t.Errorf("phase 2 : %d rafale(s), want plusieurs (2 matchs par rafale)", acc.writeCalls)
	}
	requireSameWrites(t, legacy, current, "phase 2 (rafales de 2)")
}
