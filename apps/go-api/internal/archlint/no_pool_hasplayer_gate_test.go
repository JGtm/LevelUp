package archlint

// no_pool_hasplayer_gate_test.go — UN PROFIL SANS TOKEN PROPRE SE SYNCHRONISE PAR LE POOL.
//
// POURQUOI (2026-09-16). Le pool de tokens sert n'importe quel joueur sur les endpoints
// PUBLICS (historique, stats, films, CSR — PolicyAnyPublic) ; seuls les endpoints soumis à la
// vie privée exigent le token du joueur (PolicyPinnedPlayer). Trois appelants faisaient
// pourtant `if !pool.HasPlayer(gt) { skip }` et sautaient EN BLOC un profil suivi sans token
// propre : la CLI `sync-delta --all`, la CLI `sync-full --all` et le cycle d'auto-sync
// (`checkSyncPreconditions`). Mesure du jour : Nuzzles, profil suivi depuis des mois, n'avait
// jamais été synchronisé — alors que seul son rang de carrière lui est inaccessible.
//
// LA RÈGLE. Hors du paquet `pool` (qui l'implémente) et des tests, `HasPlayer(` n'apparaît que
// là où l'appel SUIVANT est PolicyPinnedPlayer. Une seule ligne au monde le justifie
// aujourd'hui : le cron de personnalisation Spartan.
//
// Mutation qui doit le faire rougir : remettre un `if !pool.HasPlayer(...)` dans
// `cmd/levelup/cmd_sync.go` ou dans `internal/scheduler/auto_sync_run.go`.

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// exemptionsHasPlayer — allowlist DATÉE, une entrée. Chemins relatifs à `apps/go-api`, en
// slash. Y ajouter une ligne demande la même démonstration : l'appel qui suit immédiatement
// est `Acquire(ctx, pool.PolicyPinnedPlayer, …)` sur un endpoint privacy-gated.
//
//   - internal/scheduler/spartan_customization_cron.go (2026-09-16) : la personnalisation
//     Spartan est privacy-gated ; aucun autre token du parc ne peut la lire. Le `HasPlayer`
//     y évite un Acquire voué à l'échec, il ne saute pas une synchronisation.
var exemptionsHasPlayer = map[string]bool{
	"internal/scheduler/spartan_customization_cron.go": true,
}

// TestHasPlayerNeGardePasLaSync — LE RATCHET.
func TestHasPlayerNeGardePasLaSync(t *testing.T) {
	racine := racineGoAPI(t)
	var fautifs []string
	parcourirSourcesProduction(t, racine, func(rel, chemin string) {
		// Le paquet pool implémente HasPlayer : il est hors sujet.
		if strings.HasPrefix(rel, "internal/platform/auth/pool/") {
			return
		}
		if exemptionsHasPlayer[rel] {
			return
		}
		brut, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatalf("lecture de %s : %v", rel, err)
		}
		if strings.Contains(string(brut), "HasPlayer(") {
			fautifs = append(fautifs, rel)
		}
	})
	sort.Strings(fautifs)
	for _, rel := range fautifs {
		t.Errorf("%s garde un chemin derrière `HasPlayer(` — un profil sans token propre se "+
			"synchronise par le pool (D1, plan 2026-09-16) ; seul un endpoint PolicyPinnedPlayer "+
			"justifie une exemption, et elle s'écrit dans exemptionsHasPlayer avec sa date", rel)
	}
}
