//go:build integration

// Package sync — engine_convergence_defaut_integration_test.go : défaut constaté par la convergence de
// bout en bout du moteur V1 (suivi en direct), gardé ici, sauté, jusqu'à sa correction.
package sync

import (
	"fmt"
	"testing"
)

// Un match dont les stats échouent de façon durable sur le chemin du suivi en direct : la règle D-5 du
// plan (« la fin se range seulement si chaque match inconnu listé a été rangé ou déclaré introuvable »)
// exige que la fin d'historique reste non rangée. Le moteur V1 ne verse un échec de récupération que
// dans les avertissements du résultat (engine_fetch.go, fetchAndPersistMatches) : la fin est rangée, le
// match est perdu pour toujours (le delta s'arrête au premier connu, loin devant lui).
func TestConvergence_V1_SuiviEnDirect_MatchEnEchecNeDoitPasRangerLaFin(t *testing.T) {
	t.Skip("défaut: le moteur V1 verse un échec de récupération (fetchMatchData, engine_fetch.go) dans les avertissements, pas dans les erreurs ; engine.go range alors la fin d historique (len(result.Errors)==0) et le match en échec est perdu pour toujours, contre la règle D-5 du plan 8.2.2. Le corriger demande aussi l abandon de D-16 côté V1, sinon un match durable garderait le suivi en direct en rattrapage à vie.")
	env, client, w := newConvergenceV1(t)
	// Monde réduit à 60 matchs, dont un (le 31e) toujours en échec.
	w.ids = w.ids[:60]
	client.history = makeHistory(w.ids...)
	failing := w.ids[30]
	client.failStats[failing] = fmt.Errorf("GetMatchStats(%s): HTTP 500", failing)

	var ranged bool
	for pass := 1; pass <= 6 && !ranged; pass++ {
		res, err := env.engine.RunDelta(t.Context(), convergenceV1Opts())
		if err != nil {
			t.Fatalf("passe %d : %v", pass, err)
		}
		ranged = historyKeyRanged(t, env)
		if ranged && registryHas(t, env, failing) {
			t.Fatal("le match en échec ne peut pas être au registre : le scénario est faux")
		}
		if res.HistoryPending == ranged {
			t.Fatalf("passe %d : clé rangée=%v et HistoryPending=%v", pass, ranged, res.HistoryPending)
		}
	}
	if ranged {
		t.Fatalf("la fin d'historique est rangée alors que le match %s n'a jamais été récupéré : il est perdu pour toujours", failing)
	}
}
