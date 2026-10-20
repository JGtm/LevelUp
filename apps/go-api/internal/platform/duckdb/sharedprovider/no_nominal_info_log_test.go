// Package sharedprovider — no_nominal_info_log_test.go : garde-rail du lot C
// « dé-bruitage de provider.log » (2026-08-26).
//
// **Pourquoi** : le cycle B-swap journalisait 4 lignes INFO à CHAQUE prise du
// writer, pour ~40 acquisitions/minute en prod — 77 331 INFO en 8 h, ~100 Mo/JOUR
// de provider.log, dans lesquels les 121 WARN « drain timeout » de la même
// fenêtre étaient noyés à 1 pour 640. Le nominal a été démoté en DEBUG derrière
// logSwapPhase, qui ne remonte en INFO que les phases dépassant
// slowSwapThreshold.
//
// **Ce que ce test empêche** : qu'un INFO nominal réapparaisse par la petite
// porte (nouvelle étape du cycle journalisée « pour voir », ou re-promotion d'un
// des quatre messages). Sans ce ratchet, la factorisation re-diverge et le
// volume revient — leçon consignée dans CLAUDE.md (règle des ≤ 2 copies).
//
// **Ce que ce test NE dit PAS** : il ne juge pas les WARN/ERROR, qui restent
// inconditionnels et souhaitables. Il ne remplace pas
// provider_log_levels_integration_test.go, qui vérifie le NIVEAU réellement émis
// par un cycle complet — ici on ne fait que scanner les sources.
//
// Tourne dans le gate par défaut (pas de build tag) : simple lecture de fichiers.

package sharedprovider

import (
	"os"
	"strings"
	"testing"
)

// allowedInfoSites : fragments des SEULS appels INFO légitimes du package.
// Toute nouvelle entrée doit venir avec sa justification datée — un INFO qui
// se déclenche à chaque cycle n'en est jamais une.
var allowedInfoSites = map[string]string{
	// L'unique routeur DEBUG/INFO des phases du cycle : c'est lui qui arbitre
	// sur la durée, donc lui seul a le droit d'émettre un INFO de cycle.
	"slog.InfoContext(ctx, msg, append(attrs,": "logSwapPhase — INFO conditionné au dépassement de slowSwapThreshold",
	// Recovery après StateError : événement exceptionnel (2 occurrences sur les
	// 8 h mesurées), pas du nominal — reste en INFO inconditionnel.
	`"sharedprovider: recovered from StateError"`: "recovery post-StateError, événement rare",
}

// TestNoUnconditionalInfoLogInSwapCycle interdit tout appel INFO hors allowlist
// dans le code non-test du package.
func TestNoUnconditionalInfoLogInSwapCycle(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("ReadFile %s: %v", name, rerr)
		}
		scanned++
		for i, line := range strings.Split(string(content), "\n") {
			if !strings.Contains(line, "slog.Info(") && !strings.Contains(line, "slog.InfoContext(") {
				continue
			}
			if allowedInfoLine(line) {
				continue
			}
			t.Errorf("%s:%d — appel INFO hors allowlist :\n\t%s\n"+
				"Le cycle B-swap nominal doit rester en DEBUG (~40 acquisitions/minute en prod : "+
				"un INFO par cycle = ~100 Mo/jour de provider.log, qui noie les WARN). "+
				"Passer par p.logSwapPhase pour un log conditionné à la durée, ou ajouter ce site "+
				"à allowedInfoSites avec une justification datée s'il est réellement exceptionnel.",
				name, i+1, strings.TrimSpace(line))
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier source scanné : le garde-rail ne garde rien")
	}
}

// allowedInfoLine indique si la ligne correspond à un site INFO allowlisté.
func allowedInfoLine(line string) bool {
	for fragment := range allowedInfoSites {
		if strings.Contains(line, fragment) {
			return true
		}
	}
	return false
}
