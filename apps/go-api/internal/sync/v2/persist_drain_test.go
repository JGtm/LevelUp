// Package v2 — persist_drain_test.go : délai d'attente de la file de persistance (proportionnel au
// lot) et erreur typée de son expiration.
package v2

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/persist"
)

// Contrat du délai : au moins le triple de la mesure (une seconde par match), l'ancien délai fixe
// comme plancher, un plafond dur.
func TestPersistDrainTimeout_Contrat(t *testing.T) {
	if got := persistDrainTimeout(0); got != 60*time.Second {
		t.Errorf("lot vide : %v, want %v (l'ancien délai fixe)", got, persistDrainBase)
	}
	// 200 matchs mesurés à ~3,7 minutes : le délai en prend au moins le triple.
	if got, measured := persistDrainTimeout(200), 222*time.Second; got < 3*measured {
		t.Errorf("200 matchs : %v < 3 x %v mesurés", got, measured)
	}
	if persistDrainTimeout(10) >= persistDrainTimeout(11) {
		t.Error("le délai doit croître avec le lot")
	}
	if got := persistDrainTimeout(100_000); got != 20*time.Minute {
		t.Errorf("lot énorme : %v, want le plafond %v", got, persistDrainMax)
	}
	if got := persistDrainTimeout(-5); got != persistDrainBase {
		t.Errorf("lot négatif : %v, want %v", got, persistDrainBase)
	}
}

// slowQueue : une file dont le worker persiste chaque lot en `perBatch` ; le persister de cycle
// attend avec la formule donnée.
func slowQueue(t *testing.T, perBatch time.Duration) *persist.BatchQueue {
	t.Helper()
	q := setupTestBatchQueue(t)
	go func() {
		for b := range q.Channel() {
			time.Sleep(perBatch)
			_ = q.ACK(b.BatchID)
			q.RecordPersistResult(true)
		}
	}()
	return q
}

func slowBatch(n int) CycleBatch {
	const owner = "1000000000000001"
	xuids := []string{owner, "1000000000000002", "1000000000000003", "1000000000000004"}
	matches := map[string]SharedMatchData{}
	for i := range n {
		id := fmt.Sprintf("slow-%03d", i)
		matches[id] = SharedMatchData{MatchID: id, Fetcher: "owner", Stats: statsEquipe(id, xuids)}
	}
	return CycleBatch{
		CycleID: "slow", Matches: matches,
		PlayerBySlug: map[string]PlayerProfile{"owner": {Gamertag: "Owner", XUID: owner, PlayerSlug: "owner"}},
	}
}

// File plus lente que l'ancien délai (fixe) : avec le délai proportionnel au lot, le cycle réussit ;
// avec le délai fixe, l'attente expire et l'erreur typée dit combien de matchs étaient soumis.
func TestPersistCycle_FileLente_DelaiProportionnelAuLot(t *testing.T) {
	const n = 6
	perBatch := 80 * time.Millisecond // 480 ms au total
	fixed := 100 * time.Millisecond   // l'« ancien délai » à l'échelle du test

	t.Run("délai fixe : expiration typée", func(t *testing.T) {
		p := NewCycleBatchPersister("halo_infinite", slowQueue(t, perBatch), fixed, nil, nil, nil)
		err := p.PersistCycle(context.Background(), slowBatch(n))
		var dt *DrainTimeoutError
		if !errors.As(err, &dt) {
			t.Fatalf("err = %v, want *DrainTimeoutError", err)
		}
		if len(dt.Submitted) != n || dt.Waited != fixed {
			t.Errorf("DrainTimeoutError = %+v, want %d soumis après %v", dt, n, fixed)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Error("l'erreur doit garder la chaîne context.DeadlineExceeded")
		}
	})

	t.Run("délai proportionnel : cycle réussi", func(t *testing.T) {
		p := NewCycleBatchPersister("halo_infinite", slowQueue(t, perBatch), 0, nil, nil, nil)
		p.(*cycleBatchPersisterV1Bridge).drainFor = func(submitted int) time.Duration {
			return scaledDrainTimeout(fixed, 3*perBatch, 5*time.Second, submitted)
		}
		if err := p.PersistCycle(context.Background(), slowBatch(n)); err != nil {
			t.Fatalf("PersistCycle : %v (la file finit dans le délai proportionnel)", err)
		}
	})
}

// Une file qui ne se vide jamais (plafond dépassé) : l'expiration est typée, sans lot soumis par
// ce cycle ; un contexte annulé n'est PAS une expiration de l'attente.
func TestPersistCycle_Expiration_EtAnnulation(t *testing.T) {
	q := setupTestBatchQueue(t) // aucun worker : le lot soumis ne sort jamais du WAL
	p := NewCycleBatchPersister("halo_infinite", q, 60*time.Millisecond, nil, nil, nil)

	err := p.PersistCycle(context.Background(), slowBatch(3))
	var dt *DrainTimeoutError
	if !errors.As(err, &dt) || len(dt.Submitted) != 3 {
		t.Fatalf("err = %v, want une expiration typée avec 3 soumis", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = p.PersistCycle(ctx, CycleBatch{})
	if err != nil && errors.As(err, &dt) {
		t.Errorf("contexte annulé : %v, ne doit pas passer pour une expiration de l'attente", err)
	}
}
