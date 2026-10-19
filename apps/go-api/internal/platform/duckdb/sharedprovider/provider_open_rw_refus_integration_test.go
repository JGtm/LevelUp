//go:build integration

package sharedprovider_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_RefusOuvertureRW_DitEtCompte_integration : un handle lecture seule tenu hors du
// provider fait refuser l'OpenReadWrite du swap. Le refus est journalisé en ERROR avec le nombre de
// références restantes, compté sous la raison open_rw, et le provider revient en lecture ; une fois
// la référence rendue, l'écriture repasse (aucun état collant propre au provider).
func TestProvider_RefusOuvertureRW_DitEtCompte_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	sharedprovider.SetRetryBaseBackoffForTest(p, 20*time.Millisecond)
	logs := captureSlogJSON(t)

	etranger, err := duckdb.OpenReadOnly(path) // référence tenue hors du provider
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	avant := sharedprovider.OpenRWRefusalsForTest()
	if _, err := p.AcquireWriter(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "open RW after RO close") {
		t.Fatalf("AcquireWriter avec une référence étrangère : err = %v, attendu un refus d'ouverture RW", err)
	}
	if got := sharedprovider.OpenRWRefusalsForTest() - avant; got != 1 {
		t.Errorf("compteur open_rw : +%d, attendu +1", got)
	}
	var vu bool
	for _, rec := range providerRecordsFor(t, logs, path) {
		if rec.Level == "ERROR" && strings.Contains(rec.Msg, "écriture refusée") {
			vu = true
		}
	}
	if !vu || !strings.Contains(logs.String(), `"references_lecture_hors_provider":1`) {
		t.Errorf("log ERROR du refus avec references_lecture_hors_provider=1 absent :\n%s", logs.String())
	}

	attendreRO(t, p)
	if err := etranger.Close(); err != nil {
		t.Fatalf("Close étranger: %v", err)
	}
	w, err := p.AcquireWriter(context.Background())
	if err != nil {
		t.Fatalf("AcquireWriter après restitution de la référence : %v", err)
	}
	w.Release()
}

func attendreRO(t *testing.T, p sharedprovider.Provider) {
	t.Helper()
	for limite := time.Now().Add(5 * time.Second); p.State() != sharedprovider.StateRO; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(limite) {
			t.Fatalf("provider en %s, toujours pas revenu en lecture", p.State())
		}
	}
}
