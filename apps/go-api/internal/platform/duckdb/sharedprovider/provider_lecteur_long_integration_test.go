//go:build integration

package sharedprovider_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// lecteurQuiTient prend une lecture et la rend par le release retourné : son nom doit apparaître
// comme appelant dans les journaux du provider.
func lecteurQuiTient(t *testing.T, p sharedprovider.Provider) func() {
	t.Helper()
	_, release, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return release
}

// TestProvider_LecteurLongNommeEtCompte_integration : un lecteur qui tient la base fait expirer le
// drain d'un writer ; le WARN du drain nomme ce lecteur et sa durée, et son relâchement au-delà du
// seuil est journalisé (appelant, durée) et compté.
func TestProvider_LecteurLongNommeEtCompte_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	sharedprovider.SetDrainTimeoutForTest(p, 100*time.Millisecond)
	sharedprovider.SetSlowReaderThresholdForTest(p, 50*time.Millisecond)
	logs := captureSlogJSON(t)
	avant := sharedprovider.SlowReadersForTest()

	release := lecteurQuiTient(t, p)
	if _, err := p.AcquireWriter(context.Background()); !errors.Is(err, sharedprovider.ErrDrainTimeout) {
		t.Fatalf("AcquireWriter pendant la lecture : err = %v, attendu ErrDrainTimeout", err)
	}
	release()

	var drain, lent bool
	for _, line := range strings.Split(logs.String(), "\n") {
		if strings.Contains(line, "drain timeout") && strings.Contains(line, "lecteurQuiTient") &&
			strings.Contains(line, " ms)") {
			drain = true
		}
		if strings.Contains(line, "lecture tenue longtemps") && strings.Contains(line, "lecteurQuiTient") &&
			strings.Contains(line, `"held_ms"`) {
			lent = true
		}
	}
	if !drain {
		t.Errorf("le WARN du drain expiré ne nomme pas le lecteur en vol :\n%s", logs.String())
	}
	if !lent {
		t.Errorf("le relâchement de la lecture longue n'est pas journalisé avec son appelant :\n%s", logs.String())
	}
	if got := sharedprovider.SlowReadersForTest() - avant; got != 1 {
		t.Errorf("compteur des lectures longues : +%d, attendu +1", got)
	}

	// Une lecture brève n'est ni journalisée ni comptée.
	avant = sharedprovider.SlowReadersForTest()
	lecteurQuiTient(t, p)()
	if got := sharedprovider.SlowReadersForTest() - avant; got != 0 {
		t.Errorf("lecture brève comptée comme longue (+%d)", got)
	}
}
