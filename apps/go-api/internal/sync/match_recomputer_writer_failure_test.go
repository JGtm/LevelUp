package sync

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

var errSharedWriterDown = errors.New("ecrivain partage indisponible")

// flakyWriterProvider refuse l'écrivain partagé à partir de sa (failFrom)-ième prise.
type flakyWriterProvider struct {
	sharedprovider.Provider
	calls    atomic.Int32
	failFrom int32
}

func (p *flakyWriterProvider) AcquireWriter(ctx context.Context) (*sharedprovider.WriterHandle, error) {
	if p.calls.Add(1) >= p.failFrom {
		return nil, errSharedWriterDown
	}
	return p.Provider.AcquireWriter(ctx)
}

// Écrivain partagé perdu au milieu du recalcul d'une exclusion (la sentinelle de reset est
// posée, la première rafale du rejeu est refusée) : le recalcul n'est PAS un succès, la
// demande suivante le relance et le mène au bout (revue adversariale E10.1, constat C2).
func TestMatchRecomputer_EcrivainPerduPendantLeRejeu_EchecPuisRelance(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedLUSRShared(t, sharedPath, 12)
	seedLUSRPlayer(t, playerPath)

	base, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = base.Close() })
	flaky := &flakyWriterProvider{Provider: base, failFrom: 2}

	const matchID = "00000000-0000-0000-0000-000000000001"
	rec := NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", flaky)
	err = rec.RecomputeAfterExclusion(context.Background(), matchID)
	if !errors.Is(err, errSharedWriterDown) {
		t.Fatalf("recalcul sans écrivain pour le rejeu : err = %v, attendu l'erreur de l'écrivain", err)
	}
	if !rec.RecomputePending() {
		t.Fatal("RecomputePending = false : une demande identique doit relancer le recalcul échoué")
	}

	// L'écrivain revient : la demande suivante recalcule et mène le rejeu au bout.
	rec = NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", base)
	if err := rec.RecomputeAfterExclusion(context.Background(), matchID); err != nil {
		t.Fatalf("recalcul relancé : %v", err)
	}
	if rec.RecomputePending() {
		t.Error("RecomputePending = true après un recalcul réussi")
	}
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()
	var rated int
	if err := player.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type = 'LUSR'`).Scan(&rated); err != nil {
		t.Fatalf("lecture des notes LUSR: %v", err)
	}
	if rated != 12 {
		t.Errorf("%d match(s) notés LUSR après la relance, attendu 12", rated)
	}
}
