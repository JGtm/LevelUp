package config

// prestige_expiry_test.go — garde-fou de date pour l'activation phasée Prestige.
//
// ADR 0005 (prestige-phased-activation) : l'activation est phasée staging → prod
// et expire fin Q3 2026. Sans décision (activer en prod ou retirer le module),
// le flag PrestigeEnabled resterait un "compatibility guard forever" — l'exact
// anti-pattern proscrit par CLAUDE.md.
//
// Ce test échoue volontairement au CI à partir du 2026-10-01 pour forcer
// l'arbitrage : soit Prestige est activé en prod et on retire ce garde-fou,
// soit le module est archivé/supprimé.

import (
	"testing"
	"time"
)

func TestPrestigeFlag_ExpiryReminder(t *testing.T) {
	deadline := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)
	if time.Now().After(deadline) {
		t.Errorf("Deadline Prestige atteinte (ADR 0005, fin Q3 2026) : " +
			"décider d'activer Prestige en prod (et retirer ce garde-fou) " +
			"ou d'archiver/supprimer le module. Ne pas se contenter de repousser la date.")
	}
}
