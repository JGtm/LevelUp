// Package sync — engine_fetch_avertissements_test.go : les avertissements de fetch du pipeline
// parallèle (fetchAndPersistMatches) ne se perdent pas et suivent l'ordre des match_id.
//
// Test pur, sans DuckDB : un client dont chaque GetMatchStats échoue, aucune persistance (aucun
// match récupéré), aucun résolveur d'assets. `-race` ne tourne pas sur ce dépôt (driver DuckDB) :
// l'assertion porte sur ce qu'une course rendrait faux — le compte et l'ordre.
package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestFetchAndPersistMatches_AvertissementsCompletsEtOrdonnes(t *testing.T) {
	const n = 400
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("m%04d", i)
	}
	client := &mockHaloClient{getStatsErr: errors.New("panne simulee")}
	e := &SyncEngine{gamertag: "Temoin"}
	result := &domain.SyncResult{}

	persisted := e.fetchAndPersistMatches(context.Background(),
		historyPaginationInputs{client: client}, ids, result)

	if persisted != 0 {
		t.Fatalf("persisted = %d, attendu 0 (aucun fetch reussi)", persisted)
	}
	if got := client.callsGetStats.Load(); got != n {
		t.Fatalf("GetMatchStats appele %d fois, attendu %d", got, n)
	}
	if len(result.Warnings) != n {
		t.Fatalf("%d avertissements, attendu %d : des avertissements se sont perdus", len(result.Warnings), n)
	}
	for i, w := range result.Warnings {
		if !strings.HasPrefix(w, "fetchMatchData("+ids[i]+")") {
			t.Fatalf("avertissement %d = %q, attendu celui de %s (ordre des match_id)", i, w, ids[i])
		}
	}
}
