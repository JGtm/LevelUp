package service

// openspartan_post_import_title_test.go — G.4 des finitions v7.5 (2026-09-13).
//
// L'INVARIANT : le titre stampé sur le ctx du post-import est celui des BASES
// OUVERTES (opts.TitleSlug), jamais celui du ctx entrant. Un import de joueur
// Infinite déclenché depuis un onglet Halo 5 (en-tête `X-LevelUp-Title`, session
// sur l'autre titre) ne doit écrire NI la chaîne LUSR NI la chaîne de performance
// du titre étranger dans la base Infinite.
//
// CE FICHIER REMPLACE LE RATCHET C.1 `TestPostImportLUSRCallIsTitleStamped`, qui
// exigeait le littéral `s.recomputeLUSR(ctxkeys.WithTitleSlug(ctx, opts.TitleSlug)`.
// Ce littéral n'existe plus : le stamp a remonté à l'entrée de Run, donc le ratchet
// aurait échoué sur un code DEVENU MEILLEUR. Sa garantie est reprise, élargie, par
// TestRunStampeLeTitreAvantLaPremiereEtape ci-dessous — qui couvre TOUTES les étapes
// au lieu d'une seule. Le test supprimé n'était pas dans
// `.ai/baselines/tests_pre_migration.jsonl` (né le 2026-09-13, baseline du 2026-06-26) :
// aucune entrée à y retirer, vérifié par grep.
//
// C.1 n'avait fermé que le volet LUSR — le stamp était posé sur le seul appel à
// recomputeLUSR. `recomputePerfScores` recevait encore le ctx brut, et sa
// classification (GetPerformanceChain, title-aware) est PERSISTÉE dans
// `player_match_enrichment.performance_chain`. Deux tests ici :
//   - la RÈGLE (postImportCtx) et sa conséquence mesurable sur GetPerformanceChain,
//     classifiers de test enregistrés PUIS RESTAURÉS ;
//   - un garde-rail de SOURCE : le stamp doit précéder la première étape, sans quoi
//     une étape ajoutée plus haut hériterait à nouveau du ctx de la requête.

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/sync"
)

// TestPostImportCtxImposeLeTitreDeLaBase — la règle seule.
func TestPostImportCtxImposeLeTitreDeLaBase(t *testing.T) {
	cas := []struct {
		nom       string
		ctxEntant string
		base      string
		want      string
	}{
		{"ctx d'un AUTRE titre", "halo_5", "halo_infinite", "halo_infinite"},
		{"ctx vide", "", "halo_infinite", "halo_infinite"},
		{"import Halo 5 depuis un onglet Infinite", "halo_infinite", "halo_5", "halo_5"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			ctx := context.Background()
			if c.ctxEntant != "" {
				ctx = ctxkeys.WithTitleSlug(ctx, c.ctxEntant)
			}
			if got := ctxkeys.TitleSlug(postImportCtx(ctx, c.base)); got != c.want {
				t.Fatalf("titre stampé = %q, want %q", got, c.want)
			}
		})
	}
}

// TestChainePerformanceSuitLeTitreDeLaBase — la CONSÉQUENCE mesurable : avec un
// classifier dédié enregistré pour un titre étranger, la chaîne de performance
// calculée sous le ctx du post-import est celle d'Infinite, pas celle du ctx entrant.
//
// Les classifiers sont des variables de paquet globales : ils sont enregistrés
// puis RESTAURÉS, sinon ce test empoisonnerait les suivants.
func TestChainePerformanceSuitLeTitreDeLaBase(t *testing.T) {
	// Titre SYNTHÉTIQUE : enregistrer un classifier sur `halo_5` écraserait celui
	// que le boot réel y pose, et la restauration remettrait le classifier par
	// défaut à sa place — un test ne doit pas laisser cette trace. Le slug ci-dessous
	// n'est câblé nulle part ailleurs (même procédé que
	// internal/sync/engine_lusr_title_ctx_integration_test.go).
	const titreEtranger = "titre_temoin_post_import"
	const chaineTemoin = "chaine_du_titre_etranger"

	sync.SetLUSRChainClassifierForTitle(titreEtranger, func(string) string { return chaineTemoin })
	t.Cleanup(func() { sync.SetLUSRChainClassifierForTitle(titreEtranger, sync.GetLUSRChain) })

	const pair = "Slayer:Arena Slayer"

	// Contrôle : sous le titre étranger, la chaîne témoin sort bien — sans quoi le
	// test ci-dessous serait vert pour une mauvaise raison.
	if got := sync.GetPerformanceChain(titreEtranger, pair, false, false); got != chaineTemoin {
		t.Fatalf("classifier de test non pris en compte : chaîne %s = %q, want %q", titreEtranger, got, chaineTemoin)
	}

	// Le cas réel : import halo_infinite déclenché depuis un onglet d'un autre titre.
	ctx := postImportCtx(ctxkeys.WithTitleSlug(context.Background(), titreEtranger), "halo_infinite")
	got := sync.GetPerformanceChain(ctxkeys.TitleSlug(ctx), pair, false, false)
	if got == chaineTemoin {
		t.Fatalf("la chaîne de performance suit le ctx ENTRANT (%q) au lieu du titre de la BASE — "+
			"une note Infinite serait classée sur la grille d'un autre titre", got)
	}
	if got == "" {
		t.Fatal("chaîne vide : GetPerformanceChain garantit qu'aucun match n'est orphelin")
	}
}

// TestRunStampeLeTitreAvantLaPremiereEtape — garde-rail de SOURCE. L'invariant
// « toutes les étapes » ne tient que si le stamp précède la première d'entre
// elles ; une étape ajoutée au-dessus hériterait sinon du ctx de la requête, en
// silence. Le contrôle lit le fichier plutôt que de simuler un Run complet (qui
// exigerait shared + metadata + provider réels pour ne prouver que cet ordre).
func TestRunStampeLeTitreAvantLaPremiereEtape(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "openspartan_post_import_service.go"))
	if err != nil {
		t.Fatalf("lecture de la source: %v", err)
	}
	texte := string(src)

	posStamp := strings.Index(texte, "ctx = postImportCtx(ctx, opts.TitleSlug)")
	if posStamp < 0 {
		t.Fatal("le stamp `ctx = postImportCtx(ctx, opts.TitleSlug)` a disparu de Run")
	}
	// Première étape appelée par Run : le premier `s.ensure`/`s.recompute` du fichier.
	posEtape := -1
	for _, prefixe := range []string{"s.ensureEnrichmentRows(", "s.recomputeCSR(", "s.recomputeLUSR("} {
		if p := strings.Index(texte, prefixe); p >= 0 && (posEtape < 0 || p < posEtape) {
			posEtape = p
		}
	}
	if posEtape < 0 {
		t.Fatal("aucune étape de post-import trouvée dans la source — le garde-rail ne contrôle plus rien")
	}
	if posStamp > posEtape {
		t.Errorf("le stamp du titre arrive APRÈS la première étape : les étapes situées avant "+
			"lui reçoivent le titre de la REQUÊTE (offsets stamp=%d étape=%d)", posStamp, posEtape)
	}
	// Le stamp par étape a été retiré : s'il réapparaît, c'est que quelqu'un a
	// re-stampé localement au lieu de s'appuyer sur celui de l'entrée.
	if strings.Contains(texte, "ctxkeys.WithTitleSlug(ctx, opts.TitleSlug)") &&
		!strings.Contains(texte, "func postImportCtx(") {
		t.Error("stamp de titre en ligne dans Run — passer par postImportCtx à l'entrée")
	}
}
