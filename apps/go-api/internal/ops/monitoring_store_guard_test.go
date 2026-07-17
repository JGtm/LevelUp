// Package ops — monitoring_store_guard_test.go : garde-rail append-only.
//
// La base monitoring est append-only (ADR 0026) : le WRITER (monitoring_store.go)
// n'émet JAMAIS d'UPDATE ni de DELETE sur les tables d'événements
// (detection_events, detection_status_events, cron_runs, data_health_runs).
// Toute mutation d'état passe par un nouvel INSERT + la vue `_latest`.
//
// EXCEPTION SANCTIONNÉE (D3 revue 2026-07-17) : la rétention bornée
// (monitoring_retention.go, CapAndSweep façon notifications) émet un DELETE
// PURGE — jamais un UPDATE — pour borner la croissance (VPS 2 Go). Ce DELETE est
// mono-writer (même lease KindMonitoring que le flush), borné par un cap, et
// PRÉSERVE les vues `_latest` (protection du dernier événement par partition).
// Il vit dans un fichier dédié ; le writer principal reste, lui, strictement pur.
//
// Ce test scanne donc les DEUX sources : monitoring_store.go = zéro UPDATE/DELETE
// (writer pur) ; monitoring_retention.go = zéro UPDATE (purge, jamais mutation en
// place) — le DELETE y est autorisé.
package ops

import (
	"os"
	"regexp"
	"testing"
)

func TestMonitoringStore_NoUpdateOrDeleteOnAppendOnlyTables(t *testing.T) {
	tables := []string{"detection_events", "detection_status_events", "cron_runs", "data_health_runs"}

	// monitoring_store.go (writer append-only pur) : ni UPDATE ni DELETE.
	assertNoPatterns(t, "monitoring_store.go", tables, true /*forbidDelete*/)

	// monitoring_retention.go (rétention sanctionnée) : pas d'UPDATE (purge-only),
	// mais le DELETE de rétention y est autorisé.
	assertNoPatterns(t, "monitoring_retention.go", tables, false /*forbidDelete*/)
}

// assertNoPatterns échoue si le fichier contient un UPDATE (toujours interdit)
// ou, si forbidDelete, un DELETE FROM sur l'une des tables append-only.
func assertNoPatterns(t *testing.T, filename string, tables []string, forbidDelete bool) {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read source %s: %v", filename, err)
	}
	for _, table := range tables {
		updateRe := regexp.MustCompile(`(?i)UPDATE\s+` + table)
		if updateRe.Match(src) {
			t.Errorf("%s viole l'append-only : UPDATE interdit sur %q", filename, table)
		}
		if forbidDelete {
			delRe := regexp.MustCompile(`(?i)DELETE\s+FROM\s+` + table)
			if delRe.Match(src) {
				t.Errorf("%s viole l'append-only : DELETE FROM interdit sur %q (writer pur)", filename, table)
			}
		}
	}
}
