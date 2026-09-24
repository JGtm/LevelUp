//go:build cgo

package skill

// skill_v2_watermark_test.go — lot perf L6 (2026-09-23), items L6.1 et L6.3 : en
// régime stationnaire, le shadow LUSR v2 lit le filigrane (et l'éligibilité de ce
// qui le dépasse) sur le LECTEUR et ne prend AUCUN écrivain ; une ligne INFO par
// joueur et par cycle le dit, avec candidates et new.

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// TestLUSRV2Shadow_Stationary_NoWriterAndOneInfoLine : cycle 1, deux matchs neufs →
// une rafale ; cycle 2, rien de nouveau → zéro écrivain, zéro ligne écrite, et la
// sélection passe par le handle READ_ONLY (un attach RO refuse toute écriture). Le
// jeu porte le cas mesuré le 2026-09-23 : un match non notable (ici à trois équipes)
// resté au-dessus du filigrane pour toujours, plus un candidat sans chaîne LUSR.
func TestLUSRV2Shadow_Stationary_NoWriterAndOneInfoLine(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "") // shadow-only : seule la base shared est écrite

	path := openShadowTestFileDB(t)
	seed, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open seed: %v", err)
	}
	seedShadowFixtures(t, seed,
		shadowFixture{"st_slayer", pairSlayer, fixtureAt(60), seats2v2(2, 18, 6)},
		shadowFixture{"st_ctf", pairCTF, fixtureAt(120), seats2v2(3, 8, 11)},
		shadowFixture{"st_nochain", pairNoChain, fixtureAt(180), seats2v2(2, 20, 2)},
		shadowFixture{"st_three_teams", pairSlayer, fixtureAt(240), seatsThreeTeams()},
	)
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}
	ctx := context.Background()

	// Cycle 1 : deux matchs notables au-dessus du filigrane → UNE rafale.
	first := &roRwSplitAccess{path: path}
	if n, err := RunLUSRV2ShadowOwnerOnly(ctx, nil, first, "owner"); err != nil || n != 2 {
		t.Fatalf("cycle 1 : processed=%d err=%v, want 2 et nil", n, err)
	}
	if first.writeCalls != 1 {
		t.Fatalf("cycle 1 : writeCalls = %d, want 1", first.writeCalls)
	}
	rowsAfterFirst := countStateRowsAtPath(t, path)

	// Cycle 2 : rien de nouveau.
	buf, restore := captureSlog(t)
	defer restore()
	second := &roRwSplitAccess{path: path}
	n, err := RunLUSRV2ShadowOwnerOnly(ctx, nil, second, "owner")
	if err != nil || n != 0 {
		t.Fatalf("cycle 2 : processed=%d err=%v, want 0 et nil", n, err)
	}
	if second.writeCalls != 0 {
		t.Errorf("cycle 2 : writeCalls = %d, want 0 (aucun écrivain en régime stationnaire)", second.writeCalls)
	}
	if second.readCalls != 1 {
		t.Errorf("cycle 2 : readCalls = %d, want 1 (filigrane et éligibilité lus sur le lecteur)", second.readCalls)
	}
	if got := countStateRowsAtPath(t, path); got != rowsAfterFirst {
		t.Errorf("cycle 2 : %d lignes player_skill_state_v2, want %d (aucune écriture)", got, rowsAfterFirst)
	}
	requireLogLine(t, buf.String(), `msg="lusr_v2: rien de nouveau"`,
		"xuid=owner", "candidates=4", "new=0", "skipped_chain=1",
		"skipped_already_seen=2", "skipped_non_two_team=1", "skipped_imbalance=0")
}

// TestLUSRV2Shadow_NoCandidate_NoWriterAndOneInfoLine : un joueur sans aucun
// candidat ne prend pas d'écrivain et a quand même sa ligne INFO du cycle.
func TestLUSRV2Shadow_NoCandidate_NoWriterAndOneInfoLine(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "")

	buf, restore := captureSlog(t)
	defer restore()
	acc := &orderTrackingAccess{db: openShadowTestDB(t)}
	n, err := RunLUSRV2ShadowOwnerOnly(context.Background(), nil, acc, "nobody")
	if err != nil || n != 0 {
		t.Fatalf("processed=%d err=%v, want 0 et nil", n, err)
	}
	if acc.writeCalls != 0 {
		t.Errorf("writeCalls = %d, want 0", acc.writeCalls)
	}
	requireLogLine(t, buf.String(), `msg="lusr_v2: rien de nouveau"`,
		"xuid=nobody", "candidates=0", "new=0")
}

// countStateRowsAtPath compte les lignes player_skill_state_v2 d'une base fichier,
// par une connexion éphémère (les accès de test ont refermé les leurs).
func countStateRowsAtPath(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer db.Close() //nolint:errcheck
	return countStateRows(t, db)
}

// requireLogLine exige UNE ligne de journal portant msg, et chaque attribut
// attendu comme champ exact de cette ligne (format texte de captureSlog).
func requireLogLine(t *testing.T, out, msg string, attrs ...string) {
	t.Helper()
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, msg) {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 {
		t.Fatalf("%d ligne(s) %s, want 1 — journal :\n%s", len(lines), msg, out)
	}
	fields := make(map[string]bool)
	for _, f := range strings.Fields(lines[0]) {
		fields[f] = true
	}
	for _, a := range attrs {
		if !fields[a] {
			t.Errorf("ligne %s sans le champ %q : %s", msg, a, lines[0])
		}
	}
}
